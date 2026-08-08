package server

// Coverage for internal/server/helpdesk_grpc.go (BACKLOG b-cov-server-helpdesk).
// helpdesk_csat_config_test.go and helpdesk_intake_error_test.go already cover
// the CSAT config codec and the intake-validation error mapping -- this file
// does not repeat either. It adds: (a) mapHelpdeskError as a full sentinel
// table, (b) the proto <-> domain converters (nil-optional vs populated), and
// (c) the validation paths of the RPC handlers, plus the participant filter
// in ListTickets (own-scope security) against a real *helpdesk.Service backed
// by an in-memory stub repository.

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/helpdesk"
	"github.com/kmuhub/kmuhub/internal/middleware"
	helpdeskv1 "github.com/kmuhub/kmuhub/proto/helpdesk/v1"
)

// ============================================================================
// Stub repository (server-package copy -- helpdesk's own tests are internal)
// ============================================================================

type stubHelpdeskRepo struct {
	mu       sync.Mutex
	tickets  map[uuid.UUID]*helpdesk.Ticket
	messages map[uuid.UUID][]*helpdesk.TicketMessage
	queues   map[uuid.UUID]*helpdesk.TicketQueue

	contactExists bool
	companyExists bool
}

func newStubHelpdeskRepo() *stubHelpdeskRepo {
	return &stubHelpdeskRepo{
		tickets:       make(map[uuid.UUID]*helpdesk.Ticket),
		messages:      make(map[uuid.UUID][]*helpdesk.TicketMessage),
		queues:        make(map[uuid.UUID]*helpdesk.TicketQueue),
		contactExists: true,
		companyExists: true,
	}
}

// seedTicket inserts a ticket directly, bypassing Service.CreateTicket's
// business validation -- tests only care about ListTickets' filtering here.
func (r *stubHelpdeskRepo) seedTicket(t *helpdesk.Ticket) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *t
	r.tickets[t.ID] = &cp
}

func (r *stubHelpdeskRepo) CreateTicket(_ context.Context, t *helpdesk.Ticket) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *t
	r.tickets[t.ID] = &cp
	return nil
}

func (r *stubHelpdeskRepo) GetTicketByID(_ context.Context, id, tenantID uuid.UUID) (*helpdesk.Ticket, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tickets[id]
	if !ok || t.TenantID != tenantID {
		return nil, helpdesk.ErrTicketNotFound
	}
	cp := *t
	return &cp, nil
}

func (r *stubHelpdeskRepo) ListTickets(_ context.Context, tenantID uuid.UUID, statusFilter *string, participantID, contactID, orgID *uuid.UUID, page, pageSize int) ([]*helpdesk.Ticket, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var matched []*helpdesk.Ticket
	for _, t := range r.tickets {
		if t.TenantID != tenantID {
			continue
		}
		if statusFilter != nil && t.Status != *statusFilter {
			continue
		}
		if participantID != nil {
			isAssignee := t.AssigneeID != nil && *t.AssigneeID == *participantID
			isRequester := t.RequesterID != nil && *t.RequesterID == *participantID
			if !isAssignee && !isRequester {
				continue
			}
		}
		if contactID != nil && (t.ContactID == nil || *t.ContactID != *contactID) {
			continue
		}
		if orgID != nil && (t.OrgID == nil || *t.OrgID != *orgID) {
			continue
		}
		cp := *t
		matched = append(matched, &cp)
	}

	total := len(matched)
	start := min((page-1)*pageSize, total)
	end := min(start+pageSize, total)
	return matched[start:end], total, nil
}

func (r *stubHelpdeskRepo) UpdateTicket(_ context.Context, t *helpdesk.Ticket) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tickets[t.ID]; !ok {
		return helpdesk.ErrTicketNotFound
	}
	cp := *t
	r.tickets[t.ID] = &cp
	return nil
}

func (r *stubHelpdeskRepo) DeleteTicket(_ context.Context, id, _ uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tickets, id)
	return nil
}

func (r *stubHelpdeskRepo) ContactExists(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return r.contactExists, nil
}

func (r *stubHelpdeskRepo) CompanyExists(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return r.companyExists, nil
}

func (r *stubHelpdeskRepo) FindOpenTicketsByRequester(_ context.Context, _, _ uuid.UUID, _ string) ([]*helpdesk.Ticket, error) {
	return nil, nil
}

func (r *stubHelpdeskRepo) GetTicketBySourceMessage(_ context.Context, tenantID, messageID uuid.UUID) (*helpdesk.Ticket, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.tickets {
		if t.TenantID == tenantID && t.SourceMessageID != nil && *t.SourceMessageID == messageID {
			cp := *t
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *stubHelpdeskRepo) MergeTicketTx(_ context.Context, source *helpdesk.Ticket, _ uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tickets[source.ID]; !ok {
		return helpdesk.ErrTicketNotFound
	}
	cp := *source
	r.tickets[source.ID] = &cp
	return nil
}

func (r *stubHelpdeskRepo) SubmitCsatTx(_ context.Context, _, ticketID uuid.UUID, rating int16, comment *string, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tickets[ticketID]
	if !ok {
		return helpdesk.ErrTicketNotFound
	}
	t.CsatRating = &rating
	t.CsatComment = comment
	return nil
}

func (r *stubHelpdeskRepo) IssueCsatSurveyTokenTx(_ context.Context, _, _ uuid.UUID, _ string, _, _, _ time.Time) (bool, error) {
	return true, nil
}

func (r *stubHelpdeskRepo) GetCsatSurveyByToken(_ context.Context, _ string) (*helpdesk.CsatSurveyToken, error) {
	return nil, helpdesk.ErrCsatSurveyNotFound
}

func (r *stubHelpdeskRepo) RedeemCsatSurveyTx(_ context.Context, _, _, _ uuid.UUID, _ int16, _ *string, _ time.Time) (int, error) {
	return 0, helpdesk.ErrCsatSurveyNotFound
}

func (r *stubHelpdeskRepo) CreateMessage(_ context.Context, m *helpdesk.TicketMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messages[m.TicketID] = append(r.messages[m.TicketID], m)
	return nil
}

func (r *stubHelpdeskRepo) ListMessagesByTicket(_ context.Context, ticketID, _ uuid.UUID) ([]*helpdesk.TicketMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.messages[ticketID], nil
}

func (r *stubHelpdeskRepo) ReassignMessages(_ context.Context, _, _, _ uuid.UUID) error {
	return nil
}

func (r *stubHelpdeskRepo) CreateQueue(_ context.Context, q *helpdesk.TicketQueue) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *q
	r.queues[q.ID] = &cp
	return nil
}

func (r *stubHelpdeskRepo) GetQueueByID(_ context.Context, id, tenantID uuid.UUID) (*helpdesk.TicketQueue, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	q, ok := r.queues[id]
	if !ok || q.TenantID != tenantID {
		return nil, helpdesk.ErrQueueNotFound
	}
	cp := *q
	return &cp, nil
}

func (r *stubHelpdeskRepo) ListQueues(_ context.Context, tenantID uuid.UUID) ([]*helpdesk.TicketQueue, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*helpdesk.TicketQueue
	for _, q := range r.queues {
		if q.TenantID == tenantID {
			cp := *q
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *stubHelpdeskRepo) UpdateQueue(_ context.Context, q *helpdesk.TicketQueue) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.queues[q.ID]; !ok {
		return helpdesk.ErrQueueNotFound
	}
	cp := *q
	r.queues[q.ID] = &cp
	return nil
}

func (r *stubHelpdeskRepo) DeleteQueue(_ context.Context, id, _ uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.queues, id)
	return nil
}

func (r *stubHelpdeskRepo) CreateCannedResponse(_ context.Context, _ *helpdesk.CannedResponse) error {
	return nil
}

func (r *stubHelpdeskRepo) GetCannedResponseByID(_ context.Context, _, _ uuid.UUID) (*helpdesk.CannedResponse, error) {
	return nil, helpdesk.ErrCannedResponseNotFound
}

func (r *stubHelpdeskRepo) ListCannedResponses(_ context.Context, _ uuid.UUID) ([]*helpdesk.CannedResponse, error) {
	return nil, nil
}

func (r *stubHelpdeskRepo) UpdateCannedResponse(_ context.Context, _ *helpdesk.CannedResponse) error {
	return helpdesk.ErrCannedResponseNotFound
}

func (r *stubHelpdeskRepo) DeleteCannedResponse(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

func (r *stubHelpdeskRepo) CreateSLAPolicy(_ context.Context, _ *helpdesk.SLAPolicy) error {
	return nil
}

func (r *stubHelpdeskRepo) GetSLAPolicyByID(_ context.Context, _, _ uuid.UUID) (*helpdesk.SLAPolicy, error) {
	return nil, helpdesk.ErrSLAPolicyNotFound
}

func (r *stubHelpdeskRepo) ListSLAPolicies(_ context.Context, _ uuid.UUID) ([]*helpdesk.SLAPolicy, error) {
	return nil, nil
}

func (r *stubHelpdeskRepo) UpdateSLAPolicy(_ context.Context, _ *helpdesk.SLAPolicy) error {
	return helpdesk.ErrSLAPolicyNotFound
}

func (r *stubHelpdeskRepo) DeleteSLAPolicy(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

func (r *stubHelpdeskRepo) CreateKBArticle(_ context.Context, _ *helpdesk.KBArticle) error {
	return nil
}

func (r *stubHelpdeskRepo) GetKBArticleByID(_ context.Context, _, _ uuid.UUID) (*helpdesk.KBArticle, error) {
	return nil, helpdesk.ErrKBArticleNotFound
}

func (r *stubHelpdeskRepo) ListKBArticles(_ context.Context, _ uuid.UUID) ([]*helpdesk.KBArticle, error) {
	return nil, nil
}

func (r *stubHelpdeskRepo) UpdateKBArticle(_ context.Context, _ *helpdesk.KBArticle) error {
	return helpdesk.ErrKBArticleNotFound
}

func (r *stubHelpdeskRepo) DeleteKBArticle(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

func (r *stubHelpdeskRepo) CreateRoutingRule(_ context.Context, _ *helpdesk.RoutingRule) error {
	return nil
}

func (r *stubHelpdeskRepo) GetRoutingRuleByID(_ context.Context, _, _ uuid.UUID) (*helpdesk.RoutingRule, error) {
	return nil, helpdesk.ErrRoutingRuleNotFound
}

func (r *stubHelpdeskRepo) ListRoutingRules(_ context.Context, _ uuid.UUID) ([]*helpdesk.RoutingRule, error) {
	return nil, nil
}

func (r *stubHelpdeskRepo) UpdateRoutingRule(_ context.Context, _ *helpdesk.RoutingRule) error {
	return helpdesk.ErrRoutingRuleNotFound
}

func (r *stubHelpdeskRepo) DeleteRoutingRule(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

func (r *stubHelpdeskRepo) GetBusinessHours(_ context.Context, tenantID uuid.UUID) (*helpdesk.BusinessHours, error) {
	return &helpdesk.BusinessHours{TenantID: tenantID, Timezone: "Europe/Berlin"}, nil
}

func (r *stubHelpdeskRepo) UpsertBusinessHours(_ context.Context, bh *helpdesk.BusinessHours) error {
	return nil
}

func (r *stubHelpdeskRepo) GetHelpdeskStats(_ context.Context, _ uuid.UUID) (*helpdesk.HelpdeskStats, error) {
	return &helpdesk.HelpdeskStats{}, nil
}

// ============================================================================
// Server construction helpers
// ============================================================================

// newTestHelpdeskServer returns a server with a nil *helpdesk.Service. Usable
// ONLY for RPC paths that return before touching s.svc -- every request-parse
// error in helpdesk_grpc.go runs before the service call, so this is safe for
// them and panics immediately for anything reaching the happy path.
func newTestHelpdeskServer() *HelpdeskGRPCServer {
	return NewHelpdeskGRPCServer(nil, nil, nil)
}

func newHelpdeskServerWithRepo(repo helpdesk.Repository) (*HelpdeskGRPCServer, *helpdesk.Service) {
	svc := helpdesk.NewService(repo, nil)
	return NewHelpdeskGRPCServer(svc, nil, nil), svc
}

func helpdeskTenantContext(tenantID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), middleware.TenantIDKey, tenantID.String())
}

// ============================================================================
// mapHelpdeskError -- full sentinel table
// ============================================================================

func TestMapHelpdeskError_AllSentinels(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"ticket not found", helpdesk.ErrTicketNotFound, codes.NotFound},
		{"queue not found", helpdesk.ErrQueueNotFound, codes.NotFound},
		{"sla policy not found", helpdesk.ErrSLAPolicyNotFound, codes.NotFound},
		{"canned response not found", helpdesk.ErrCannedResponseNotFound, codes.NotFound},
		{"invalid status", helpdesk.ErrInvalidStatus, codes.InvalidArgument},
		{"invalid priority", helpdesk.ErrInvalidPriority, codes.InvalidArgument},
		{"cannot merge self", helpdesk.ErrCannotMergeSelf, codes.InvalidArgument},
		{"already merged", helpdesk.ErrAlreadyMerged, codes.FailedPrecondition},
		{"kb article not found", helpdesk.ErrKBArticleNotFound, codes.NotFound},
		{"routing rule not found", helpdesk.ErrRoutingRuleNotFound, codes.NotFound},
		{"contact not found", helpdesk.ErrContactNotFound, codes.InvalidArgument},
		{"org not found", helpdesk.ErrOrgNotFound, codes.InvalidArgument},
		{"invalid source channel", helpdesk.ErrInvalidSourceChannel, codes.InvalidArgument},
		{"invalid channel", helpdesk.ErrInvalidChannel, codes.InvalidArgument},
		{"invalid requester email", helpdesk.ErrInvalidRequesterEmail, codes.InvalidArgument},
		{"requester name too long", helpdesk.ErrRequesterNameTooLong, codes.InvalidArgument},
		{"invalid custom fields", helpdesk.ErrInvalidCustomFields, codes.InvalidArgument},
		{"missing requester", helpdesk.ErrMissingRequester, codes.InvalidArgument},
		{"csat survey not found", helpdesk.ErrCsatSurveyNotFound, codes.NotFound},
		{"csat comment too long", helpdesk.ErrCsatCommentTooLong, codes.InvalidArgument},
		{"invalid csat rating", helpdesk.ErrInvalidCsatRating, codes.InvalidArgument},
		{"message already linked", helpdesk.ErrMessageAlreadyLinked, codes.AlreadyExists},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := mapHelpdeskError(tc.err)
			requireGRPCCode(t, err, tc.want)
		})
	}

	t.Run("nil returns nil", func(t *testing.T) {
		assert.NoError(t, mapHelpdeskError(nil))
	})

	t.Run("unknown error maps to Internal", func(t *testing.T) {
		requireGRPCCode(t, mapHelpdeskError(assertErr("some unmapped failure")), codes.Internal)
	})
}

type assertErr string

func (e assertErr) Error() string { return string(e) }

// ============================================================================
// Proto <-> domain converters
// ============================================================================

func TestRequesterIDToProto(t *testing.T) {
	assert.Equal(t, "", requesterIDToProto(nil))
	id := uuid.New()
	assert.Equal(t, id.String(), requesterIDToProto(&id))
}

func TestTicketToProto_OptionalFieldsNil(t *testing.T) {
	ticket := &helpdesk.Ticket{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		Subject:      "Broken login",
		Status:       helpdesk.TicketStatusOpen,
		Priority:     helpdesk.TicketPriorityNormal,
		CustomFields: map[string]any{},
	}
	proto := ticketToProto(ticket)

	assert.Equal(t, ticket.ID.String(), proto.GetId())
	assert.Equal(t, "", proto.GetRequesterId())
	assert.Nil(t, proto.AssigneeId)
	assert.Nil(t, proto.QueueId)
	assert.Nil(t, proto.DueAt)
	assert.Nil(t, proto.MergedIntoId)
	assert.Nil(t, proto.FirstResponseAt)
	assert.Nil(t, proto.ResolvedAt)
	assert.Nil(t, proto.ContactId)
	assert.Nil(t, proto.OrgId)
	assert.Nil(t, proto.SourceChannel)
	assert.Nil(t, proto.SourceMessageId)
	assert.Nil(t, proto.CsatRating)
	assert.Nil(t, proto.CsatComment)
}

func TestTicketToProto_OptionalFieldsPopulated(t *testing.T) {
	assigneeID, queueID, contactID, orgID, sourceMsgID, mergedID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	requesterID := uuid.New()
	now := time.Now().UTC()
	sourceChannel := "email"
	rating := int16(4)
	comment := "gut geloest"

	ticket := &helpdesk.Ticket{
		ID:              uuid.New(),
		TenantID:        uuid.New(),
		Subject:         "VPN down",
		Status:          helpdesk.TicketStatusOpen,
		Priority:        helpdesk.TicketPriorityHigh,
		RequesterID:     &requesterID,
		AssigneeID:      &assigneeID,
		QueueID:         &queueID,
		ContactID:       &contactID,
		OrgID:           &orgID,
		SourceChannel:   &sourceChannel,
		SourceMessageID: &sourceMsgID,
		MergedIntoID:    &mergedID,
		DueAt:           &now,
		FirstResponseAt: &now,
		ResolvedAt:      &now,
		CsatRating:      &rating,
		CsatComment:     &comment,
		CustomFields:    map[string]any{"foo": "bar"},
	}
	proto := ticketToProto(ticket)

	assert.Equal(t, requesterID.String(), proto.GetRequesterId())
	require.NotNil(t, proto.AssigneeId)
	assert.Equal(t, assigneeID.String(), proto.GetAssigneeId())
	require.NotNil(t, proto.QueueId)
	assert.Equal(t, queueID.String(), proto.GetQueueId())
	require.NotNil(t, proto.ContactId)
	assert.Equal(t, contactID.String(), proto.GetContactId())
	require.NotNil(t, proto.OrgId)
	assert.Equal(t, orgID.String(), proto.GetOrgId())
	require.NotNil(t, proto.SourceChannel)
	assert.Equal(t, sourceChannel, proto.GetSourceChannel())
	require.NotNil(t, proto.SourceMessageId)
	assert.Equal(t, sourceMsgID.String(), proto.GetSourceMessageId())
	require.NotNil(t, proto.MergedIntoId)
	assert.Equal(t, mergedID.String(), proto.GetMergedIntoId())
	require.NotNil(t, proto.DueAt)
	require.NotNil(t, proto.CsatRating)
	assert.Equal(t, int32(4), proto.GetCsatRating())
	require.NotNil(t, proto.CsatComment)
	assert.Equal(t, comment, proto.GetCsatComment())
}

func TestTicketQueueToProto_OptionalFields(t *testing.T) {
	bare := &helpdesk.TicketQueue{ID: uuid.New(), TenantID: uuid.New(), Name: "General"}
	proto := ticketQueueToProto(bare)
	assert.Nil(t, proto.DefaultAssigneeId)
	assert.Nil(t, proto.SlaPolicyId)

	assigneeID, slaID := uuid.New(), uuid.New()
	full := &helpdesk.TicketQueue{ID: uuid.New(), TenantID: uuid.New(), Name: "VIP", DefaultAssigneeID: &assigneeID, SLAPolicyID: &slaID}
	proto = ticketQueueToProto(full)
	require.NotNil(t, proto.DefaultAssigneeId)
	assert.Equal(t, assigneeID.String(), proto.GetDefaultAssigneeId())
	require.NotNil(t, proto.SlaPolicyId)
	assert.Equal(t, slaID.String(), proto.GetSlaPolicyId())
}

func TestSLAPolicyToProto_BusinessHours(t *testing.T) {
	bare := &helpdesk.SLAPolicy{ID: uuid.New(), TenantID: uuid.New(), Name: "Standard"}
	proto := slaPolicyToProto(bare)
	assert.Nil(t, proto.BusinessHours)

	full := &helpdesk.SLAPolicy{ID: uuid.New(), TenantID: uuid.New(), Name: "Premium", BusinessHours: map[string]any{"mon": "9-17"}}
	proto = slaPolicyToProto(full)
	require.NotNil(t, proto.BusinessHours)
	assert.Contains(t, proto.GetBusinessHours(), "mon")
}

func TestRoutingRuleToProto_OptionalTargetQueue(t *testing.T) {
	bare := &helpdesk.RoutingRule{ID: uuid.New(), TenantID: uuid.New(), Name: "Default", Conditions: map[string]any{}}
	proto := routingRuleToProto(bare)
	assert.Nil(t, proto.TargetQueueId)
	assert.Equal(t, "{}", proto.GetConditions())

	queueID := uuid.New()
	full := &helpdesk.RoutingRule{ID: uuid.New(), TenantID: uuid.New(), Name: "VIP", TargetQueueID: &queueID, Conditions: map[string]any{"priority": "high"}}
	proto = routingRuleToProto(full)
	require.NotNil(t, proto.TargetQueueId)
	assert.Equal(t, queueID.String(), proto.GetTargetQueueId())
	assert.Contains(t, proto.GetConditions(), "priority")
}

func TestBusinessHoursToProto_ZeroUpdatedAt(t *testing.T) {
	bh := &helpdesk.BusinessHours{TenantID: uuid.New(), Timezone: "Europe/Berlin"}
	proto := businessHoursToProto(bh)
	assert.Nil(t, proto.UpdatedAt)

	bh.UpdatedAt = time.Now().UTC()
	proto = businessHoursToProto(bh)
	assert.NotNil(t, proto.UpdatedAt)
}

func TestKBArticleToProto(t *testing.T) {
	authorID := uuid.New()
	a := &helpdesk.KBArticle{ID: uuid.New(), TenantID: uuid.New(), Title: "FAQ", Content: "...", Category: "general", Status: string(helpdesk.KBArticleStatusPublished), AuthorID: authorID}
	proto := kbArticleToProto(a)
	assert.Equal(t, authorID.String(), proto.GetAuthorId())
	assert.Equal(t, "FAQ", proto.GetTitle())
}

func TestCannedResponseToProto(t *testing.T) {
	cr := &helpdesk.CannedResponse{ID: uuid.New(), TenantID: uuid.New(), Name: "Greeting", Body: "Hallo!"}
	proto := cannedResponseToProto(cr)
	assert.Equal(t, "Greeting", proto.GetName())
	assert.Equal(t, "Hallo!", proto.GetBody())
}

func TestTicketMessageToProto(t *testing.T) {
	m := &helpdesk.TicketMessage{ID: uuid.New(), TicketID: uuid.New(), AuthorID: uuid.New(), Body: "hi", Attachments: []string{"a.png"}}
	proto := ticketMessageToProto(m)
	assert.Equal(t, "hi", proto.GetBody())
	assert.Equal(t, []string{"a.png"}, proto.GetAttachments())
}

// ============================================================================
// Validation paths (nil-service -- every case here returns before s.svc is touched)
// ============================================================================

func TestHelpdeskGRPC_ValidationErrors(t *testing.T) {
	srv := newTestHelpdeskServer()
	ctxNoTenant := context.Background()
	ctxTenant := helpdeskTenantContext(uuid.New())

	cases := []struct {
		name string
		call func() error
		want codes.Code
	}{
		{"CreateTicket invalid tenant_id", func() error {
			_, err := srv.CreateTicket(ctxNoTenant, &helpdeskv1.CreateTicketRequest{TenantId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"CreateTicket invalid requester_id", func() error {
			_, err := srv.CreateTicket(ctxNoTenant, &helpdeskv1.CreateTicketRequest{TenantId: uuid.NewString(), RequesterId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"CreateTicket invalid assignee_id", func() error {
			bad := "not-a-uuid"
			_, err := srv.CreateTicket(ctxNoTenant, &helpdeskv1.CreateTicketRequest{TenantId: uuid.NewString(), AssigneeId: &bad})
			return err
		}, codes.InvalidArgument},
		{"CreateTicket invalid queue_id", func() error {
			bad := "not-a-uuid"
			_, err := srv.CreateTicket(ctxNoTenant, &helpdeskv1.CreateTicketRequest{TenantId: uuid.NewString(), QueueId: &bad})
			return err
		}, codes.InvalidArgument},
		{"CreateTicket invalid contact_id", func() error {
			bad := "not-a-uuid"
			_, err := srv.CreateTicket(ctxNoTenant, &helpdeskv1.CreateTicketRequest{TenantId: uuid.NewString(), ContactId: &bad})
			return err
		}, codes.InvalidArgument},
		{"CreateTicket invalid org_id", func() error {
			bad := "not-a-uuid"
			_, err := srv.CreateTicket(ctxNoTenant, &helpdeskv1.CreateTicketRequest{TenantId: uuid.NewString(), OrgId: &bad})
			return err
		}, codes.InvalidArgument},

		{"GetTicket missing tenant context", func() error {
			_, err := srv.GetTicket(ctxNoTenant, &helpdeskv1.GetTicketRequest{TicketId: uuid.NewString()})
			return err
		}, codes.InvalidArgument},
		{"GetTicket invalid ticket_id", func() error {
			_, err := srv.GetTicket(ctxTenant, &helpdeskv1.GetTicketRequest{TicketId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},

		{"ListTickets invalid tenant_id", func() error {
			_, err := srv.ListTickets(ctxNoTenant, &helpdeskv1.ListTicketsRequest{TenantId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"ListTickets invalid participant_id", func() error {
			bad := "not-a-uuid"
			_, err := srv.ListTickets(ctxNoTenant, &helpdeskv1.ListTicketsRequest{TenantId: uuid.NewString(), ParticipantId: &bad})
			return err
		}, codes.InvalidArgument},
		{"ListTickets invalid contact_id", func() error {
			bad := "not-a-uuid"
			_, err := srv.ListTickets(ctxNoTenant, &helpdeskv1.ListTicketsRequest{TenantId: uuid.NewString(), ContactId: &bad})
			return err
		}, codes.InvalidArgument},
		{"ListTickets invalid org_id", func() error {
			bad := "not-a-uuid"
			_, err := srv.ListTickets(ctxNoTenant, &helpdeskv1.ListTicketsRequest{TenantId: uuid.NewString(), OrgId: &bad})
			return err
		}, codes.InvalidArgument},

		{"UpdateTicket missing tenant context", func() error {
			_, err := srv.UpdateTicket(ctxNoTenant, &helpdeskv1.UpdateTicketRequest{TicketId: uuid.NewString()})
			return err
		}, codes.InvalidArgument},
		{"UpdateTicket invalid ticket_id", func() error {
			_, err := srv.UpdateTicket(ctxTenant, &helpdeskv1.UpdateTicketRequest{TicketId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},

		{"CloseTicket missing tenant context", func() error {
			_, err := srv.CloseTicket(ctxNoTenant, &helpdeskv1.CloseTicketRequest{TicketId: uuid.NewString()})
			return err
		}, codes.InvalidArgument},
		{"CloseTicket invalid ticket_id", func() error {
			_, err := srv.CloseTicket(ctxTenant, &helpdeskv1.CloseTicketRequest{TicketId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},

		{"SubmitCsat missing tenant context", func() error {
			_, err := srv.SubmitCsat(ctxNoTenant, &helpdeskv1.SubmitCsatRequest{TicketId: uuid.NewString(), Rating: 3})
			return err
		}, codes.InvalidArgument},
		{"SubmitCsat invalid ticket_id", func() error {
			_, err := srv.SubmitCsat(ctxTenant, &helpdeskv1.SubmitCsatRequest{TicketId: "not-a-uuid", Rating: 3})
			return err
		}, codes.InvalidArgument},
		{"SubmitCsat rating too low", func() error {
			_, err := srv.SubmitCsat(ctxTenant, &helpdeskv1.SubmitCsatRequest{TicketId: uuid.NewString(), Rating: 0})
			return err
		}, codes.InvalidArgument},
		{"SubmitCsat rating too high", func() error {
			_, err := srv.SubmitCsat(ctxTenant, &helpdeskv1.SubmitCsatRequest{TicketId: uuid.NewString(), Rating: 6})
			return err
		}, codes.InvalidArgument},

		{"SubmitCsatByToken rating too low", func() error {
			_, err := srv.SubmitCsatByToken(ctxNoTenant, &helpdeskv1.SubmitCsatByTokenRequest{Token: "tok", Rating: 0})
			return err
		}, codes.InvalidArgument},
		{"SubmitCsatByToken rating too high", func() error {
			_, err := srv.SubmitCsatByToken(ctxNoTenant, &helpdeskv1.SubmitCsatByTokenRequest{Token: "tok", Rating: 6})
			return err
		}, codes.InvalidArgument},

		{"ReopenTicket missing tenant context", func() error {
			_, err := srv.ReopenTicket(ctxNoTenant, &helpdeskv1.ReopenTicketRequest{TicketId: uuid.NewString()})
			return err
		}, codes.InvalidArgument},
		{"ReopenTicket invalid ticket_id", func() error {
			_, err := srv.ReopenTicket(ctxTenant, &helpdeskv1.ReopenTicketRequest{TicketId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},

		{"AssignTicket missing tenant context", func() error {
			_, err := srv.AssignTicket(ctxNoTenant, &helpdeskv1.AssignTicketRequest{TicketId: uuid.NewString(), AssigneeId: uuid.NewString()})
			return err
		}, codes.InvalidArgument},
		{"AssignTicket invalid ticket_id", func() error {
			_, err := srv.AssignTicket(ctxTenant, &helpdeskv1.AssignTicketRequest{TicketId: "not-a-uuid", AssigneeId: uuid.NewString()})
			return err
		}, codes.InvalidArgument},
		{"AssignTicket invalid assignee_id", func() error {
			_, err := srv.AssignTicket(ctxTenant, &helpdeskv1.AssignTicketRequest{TicketId: uuid.NewString(), AssigneeId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},

		{"MergeTickets missing tenant context", func() error {
			_, err := srv.MergeTickets(ctxNoTenant, &helpdeskv1.MergeTicketsRequest{SourceTicketId: uuid.NewString(), TargetTicketId: uuid.NewString()})
			return err
		}, codes.InvalidArgument},
		{"MergeTickets invalid source_ticket_id", func() error {
			_, err := srv.MergeTickets(ctxTenant, &helpdeskv1.MergeTicketsRequest{SourceTicketId: "not-a-uuid", TargetTicketId: uuid.NewString()})
			return err
		}, codes.InvalidArgument},
		{"MergeTickets invalid target_ticket_id", func() error {
			_, err := srv.MergeTickets(ctxTenant, &helpdeskv1.MergeTicketsRequest{SourceTicketId: uuid.NewString(), TargetTicketId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},

		{"CreateTicketFromMessage invalid tenant_id", func() error {
			_, err := srv.CreateTicketFromMessage(ctxNoTenant, &helpdeskv1.CreateTicketFromMessageRequest{TenantId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"CreateTicketFromMessage invalid requester_id", func() error {
			_, err := srv.CreateTicketFromMessage(ctxNoTenant, &helpdeskv1.CreateTicketFromMessageRequest{TenantId: uuid.NewString(), RequesterId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"CreateTicketFromMessage invalid message_id", func() error {
			_, err := srv.CreateTicketFromMessage(ctxNoTenant, &helpdeskv1.CreateTicketFromMessageRequest{TenantId: uuid.NewString(), RequesterId: uuid.NewString(), MessageId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"CreateTicketFromMessage no inbox client", func() error {
			_, err := srv.CreateTicketFromMessage(ctxNoTenant, &helpdeskv1.CreateTicketFromMessageRequest{TenantId: uuid.NewString(), RequesterId: uuid.NewString(), MessageId: uuid.NewString()})
			return err
		}, codes.Unavailable},

		{"AddMessage missing tenant context", func() error {
			_, err := srv.AddMessage(ctxNoTenant, &helpdeskv1.AddMessageRequest{TicketId: uuid.NewString(), AuthorId: uuid.NewString()})
			return err
		}, codes.InvalidArgument},
		{"AddMessage invalid ticket_id", func() error {
			_, err := srv.AddMessage(ctxTenant, &helpdeskv1.AddMessageRequest{TicketId: "not-a-uuid", AuthorId: uuid.NewString()})
			return err
		}, codes.InvalidArgument},
		{"AddMessage invalid author_id", func() error {
			_, err := srv.AddMessage(ctxTenant, &helpdeskv1.AddMessageRequest{TicketId: uuid.NewString(), AuthorId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},

		{"ListMessages missing tenant context", func() error {
			_, err := srv.ListMessages(ctxNoTenant, &helpdeskv1.ListMessagesRequest{TicketId: uuid.NewString()})
			return err
		}, codes.InvalidArgument},
		{"ListMessages invalid ticket_id", func() error {
			_, err := srv.ListMessages(ctxTenant, &helpdeskv1.ListMessagesRequest{TicketId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},

		{"CreateQueue invalid tenant_id", func() error {
			_, err := srv.CreateQueue(ctxNoTenant, &helpdeskv1.CreateQueueRequest{TenantId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"CreateQueue invalid default_assignee_id", func() error {
			bad := "not-a-uuid"
			_, err := srv.CreateQueue(ctxNoTenant, &helpdeskv1.CreateQueueRequest{TenantId: uuid.NewString(), DefaultAssigneeId: &bad})
			return err
		}, codes.InvalidArgument},
		{"CreateQueue invalid sla_policy_id", func() error {
			bad := "not-a-uuid"
			_, err := srv.CreateQueue(ctxNoTenant, &helpdeskv1.CreateQueueRequest{TenantId: uuid.NewString(), SlaPolicyId: &bad})
			return err
		}, codes.InvalidArgument},
		{"UpdateQueue missing tenant context", func() error {
			_, err := srv.UpdateQueue(ctxNoTenant, &helpdeskv1.UpdateQueueRequest{QueueId: uuid.NewString()})
			return err
		}, codes.InvalidArgument},
		{"UpdateQueue invalid queue_id", func() error {
			_, err := srv.UpdateQueue(ctxTenant, &helpdeskv1.UpdateQueueRequest{QueueId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"ListQueues invalid tenant_id", func() error {
			_, err := srv.ListQueues(ctxNoTenant, &helpdeskv1.ListQueuesRequest{TenantId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"DeleteQueue missing tenant context", func() error {
			_, err := srv.DeleteQueue(ctxNoTenant, &helpdeskv1.DeleteQueueRequest{QueueId: uuid.NewString()})
			return err
		}, codes.InvalidArgument},
		{"DeleteQueue invalid queue_id", func() error {
			_, err := srv.DeleteQueue(ctxTenant, &helpdeskv1.DeleteQueueRequest{QueueId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},

		{"CreateCannedResponse invalid tenant_id", func() error {
			_, err := srv.CreateCannedResponse(ctxNoTenant, &helpdeskv1.CreateCannedResponseRequest{TenantId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"UpdateCannedResponse missing tenant context", func() error {
			_, err := srv.UpdateCannedResponse(ctxNoTenant, &helpdeskv1.UpdateCannedResponseRequest{CannedResponseId: uuid.NewString()})
			return err
		}, codes.InvalidArgument},
		{"UpdateCannedResponse invalid id", func() error {
			_, err := srv.UpdateCannedResponse(ctxTenant, &helpdeskv1.UpdateCannedResponseRequest{CannedResponseId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"DeleteCannedResponse missing tenant context", func() error {
			_, err := srv.DeleteCannedResponse(ctxNoTenant, &helpdeskv1.DeleteCannedResponseRequest{CannedResponseId: uuid.NewString()})
			return err
		}, codes.InvalidArgument},
		{"DeleteCannedResponse invalid id", func() error {
			_, err := srv.DeleteCannedResponse(ctxTenant, &helpdeskv1.DeleteCannedResponseRequest{CannedResponseId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"ListCannedResponses invalid tenant_id", func() error {
			_, err := srv.ListCannedResponses(ctxNoTenant, &helpdeskv1.ListCannedResponsesRequest{TenantId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},

		{"CreateSLAPolicy invalid tenant_id", func() error {
			_, err := srv.CreateSLAPolicy(ctxNoTenant, &helpdeskv1.CreateSLAPolicyRequest{TenantId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"CreateSLAPolicy invalid business_hours JSON", func() error {
			_, err := srv.CreateSLAPolicy(ctxNoTenant, &helpdeskv1.CreateSLAPolicyRequest{TenantId: uuid.NewString(), BusinessHours: strPtr("{not json")})
			return err
		}, codes.InvalidArgument},
		{"UpdateSLAPolicy missing tenant context", func() error {
			_, err := srv.UpdateSLAPolicy(ctxNoTenant, &helpdeskv1.UpdateSLAPolicyRequest{SlaPolicyId: uuid.NewString()})
			return err
		}, codes.InvalidArgument},
		{"UpdateSLAPolicy invalid id", func() error {
			_, err := srv.UpdateSLAPolicy(ctxTenant, &helpdeskv1.UpdateSLAPolicyRequest{SlaPolicyId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"UpdateSLAPolicy invalid business_hours JSON", func() error {
			_, err := srv.UpdateSLAPolicy(ctxTenant, &helpdeskv1.UpdateSLAPolicyRequest{SlaPolicyId: uuid.NewString(), BusinessHours: strPtr("{not json")})
			return err
		}, codes.InvalidArgument},
		{"ListSLAPolicies invalid tenant_id", func() error {
			_, err := srv.ListSLAPolicies(ctxNoTenant, &helpdeskv1.ListSLAPoliciesRequest{TenantId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"ApplySLAPolicy missing tenant context", func() error {
			_, err := srv.ApplySLAPolicy(ctxNoTenant, &helpdeskv1.ApplySLAPolicyRequest{TicketId: uuid.NewString(), SlaPolicyId: uuid.NewString()})
			return err
		}, codes.InvalidArgument},
		{"ApplySLAPolicy invalid ticket_id", func() error {
			_, err := srv.ApplySLAPolicy(ctxTenant, &helpdeskv1.ApplySLAPolicyRequest{TicketId: "not-a-uuid", SlaPolicyId: uuid.NewString()})
			return err
		}, codes.InvalidArgument},
		{"ApplySLAPolicy invalid sla_policy_id", func() error {
			_, err := srv.ApplySLAPolicy(ctxTenant, &helpdeskv1.ApplySLAPolicyRequest{TicketId: uuid.NewString(), SlaPolicyId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"DeleteSLAPolicy missing tenant context", func() error {
			_, err := srv.DeleteSLAPolicy(ctxNoTenant, &helpdeskv1.DeleteSLAPolicyRequest{SlaPolicyId: uuid.NewString()})
			return err
		}, codes.InvalidArgument},
		{"DeleteSLAPolicy invalid id", func() error {
			_, err := srv.DeleteSLAPolicy(ctxTenant, &helpdeskv1.DeleteSLAPolicyRequest{SlaPolicyId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"GetSLAStatus missing tenant context", func() error {
			_, err := srv.GetSLAStatus(ctxNoTenant, &helpdeskv1.GetSLAStatusRequest{TicketId: uuid.NewString()})
			return err
		}, codes.InvalidArgument},
		{"GetSLAStatus invalid ticket_id", func() error {
			_, err := srv.GetSLAStatus(ctxTenant, &helpdeskv1.GetSLAStatusRequest{TicketId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"GetSLAStatus invalid sla_policy_id", func() error {
			bad := "not-a-uuid"
			_, err := srv.GetSLAStatus(ctxTenant, &helpdeskv1.GetSLAStatusRequest{TicketId: uuid.NewString(), SlaPolicyId: &bad})
			return err
		}, codes.InvalidArgument},

		{"ListKBArticle invalid tenant_id", func() error {
			_, err := srv.ListKBArticle(ctxNoTenant, &helpdeskv1.ListKBArticleRequest{TenantId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"CreateKBArticle invalid tenant_id", func() error {
			_, err := srv.CreateKBArticle(ctxNoTenant, &helpdeskv1.CreateKBArticleRequest{TenantId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"UpdateKBArticle missing tenant context", func() error {
			_, err := srv.UpdateKBArticle(ctxNoTenant, &helpdeskv1.UpdateKBArticleRequest{ArticleId: uuid.NewString()})
			return err
		}, codes.InvalidArgument},
		{"UpdateKBArticle invalid article_id", func() error {
			_, err := srv.UpdateKBArticle(ctxTenant, &helpdeskv1.UpdateKBArticleRequest{ArticleId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"DeleteKBArticle missing tenant context", func() error {
			_, err := srv.DeleteKBArticle(ctxNoTenant, &helpdeskv1.DeleteKBArticleRequest{ArticleId: uuid.NewString()})
			return err
		}, codes.InvalidArgument},
		{"DeleteKBArticle invalid article_id", func() error {
			_, err := srv.DeleteKBArticle(ctxTenant, &helpdeskv1.DeleteKBArticleRequest{ArticleId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},

		{"ListRoutingRule invalid tenant_id", func() error {
			_, err := srv.ListRoutingRule(ctxNoTenant, &helpdeskv1.ListRoutingRuleRequest{TenantId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"CreateRoutingRule invalid tenant_id", func() error {
			_, err := srv.CreateRoutingRule(ctxNoTenant, &helpdeskv1.CreateRoutingRuleRequest{TenantId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"CreateRoutingRule invalid conditions JSON", func() error {
			_, err := srv.CreateRoutingRule(ctxNoTenant, &helpdeskv1.CreateRoutingRuleRequest{TenantId: uuid.NewString(), Conditions: "{not json"})
			return err
		}, codes.InvalidArgument},
		{"CreateRoutingRule invalid target_queue_id", func() error {
			bad := "not-a-uuid"
			_, err := srv.CreateRoutingRule(ctxNoTenant, &helpdeskv1.CreateRoutingRuleRequest{TenantId: uuid.NewString(), TargetQueueId: &bad})
			return err
		}, codes.InvalidArgument},
		{"UpdateRoutingRule missing tenant context", func() error {
			_, err := srv.UpdateRoutingRule(ctxNoTenant, &helpdeskv1.UpdateRoutingRuleRequest{RuleId: uuid.NewString()})
			return err
		}, codes.InvalidArgument},
		{"UpdateRoutingRule invalid rule_id", func() error {
			_, err := srv.UpdateRoutingRule(ctxTenant, &helpdeskv1.UpdateRoutingRuleRequest{RuleId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"UpdateRoutingRule invalid conditions JSON", func() error {
			cond := "{not json"
			_, err := srv.UpdateRoutingRule(ctxTenant, &helpdeskv1.UpdateRoutingRuleRequest{RuleId: uuid.NewString(), Conditions: &cond})
			return err
		}, codes.InvalidArgument},
		{"UpdateRoutingRule invalid target_queue_id", func() error {
			bad := "not-a-uuid"
			_, err := srv.UpdateRoutingRule(ctxTenant, &helpdeskv1.UpdateRoutingRuleRequest{RuleId: uuid.NewString(), TargetQueueId: &bad})
			return err
		}, codes.InvalidArgument},
		{"DeleteRoutingRule missing tenant context", func() error {
			_, err := srv.DeleteRoutingRule(ctxNoTenant, &helpdeskv1.DeleteRoutingRuleRequest{RuleId: uuid.NewString()})
			return err
		}, codes.InvalidArgument},
		{"DeleteRoutingRule invalid rule_id", func() error {
			_, err := srv.DeleteRoutingRule(ctxTenant, &helpdeskv1.DeleteRoutingRuleRequest{RuleId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},

		{"GetBusinessHours invalid tenant_id", func() error {
			_, err := srv.GetBusinessHours(ctxNoTenant, &helpdeskv1.GetBusinessHoursRequest{TenantId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"UpdateBusinessHours invalid tenant_id", func() error {
			_, err := srv.UpdateBusinessHours(ctxNoTenant, &helpdeskv1.UpdateBusinessHoursRequest{TenantId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
		{"UpdateBusinessHours invalid schedule_json", func() error {
			_, err := srv.UpdateBusinessHours(ctxNoTenant, &helpdeskv1.UpdateBusinessHoursRequest{TenantId: uuid.NewString(), ScheduleJson: "{not json"})
			return err
		}, codes.InvalidArgument},
		{"UpdateBusinessHours invalid holidays_json", func() error {
			_, err := srv.UpdateBusinessHours(ctxNoTenant, &helpdeskv1.UpdateBusinessHoursRequest{TenantId: uuid.NewString(), HolidaysJson: "{not json"})
			return err
		}, codes.InvalidArgument},

		{"GetHelpdeskStats invalid tenant_id", func() error {
			_, err := srv.GetHelpdeskStats(ctxNoTenant, &helpdeskv1.GetHelpdeskStatsRequest{TenantId: "not-a-uuid"})
			return err
		}, codes.InvalidArgument},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireGRPCCode(t, tc.call(), tc.want)
		})
	}
}

func strPtr(s string) *string { return &s }

// ============================================================================
// ListTickets participant filter -- own-scope security
// ============================================================================

func TestListTickets_ParticipantFilter(t *testing.T) {
	repo := newStubHelpdeskRepo()
	srv, _ := newHelpdeskServerWithRepo(repo)

	tenantID := uuid.New()
	agentA, agentB := uuid.New(), uuid.New()

	ticketAssignedToA := &helpdesk.Ticket{
		ID: uuid.New(), TenantID: tenantID, Subject: "A's ticket", Status: helpdesk.TicketStatusOpen,
		Priority: helpdesk.TicketPriorityNormal, AssigneeID: &agentA, CustomFields: map[string]any{},
	}
	ticketRequestedByA := &helpdesk.Ticket{
		ID: uuid.New(), TenantID: tenantID, Subject: "raised by A", Status: helpdesk.TicketStatusOpen,
		Priority: helpdesk.TicketPriorityNormal, RequesterID: &agentA, CustomFields: map[string]any{},
	}
	ticketAssignedToB := &helpdesk.Ticket{
		ID: uuid.New(), TenantID: tenantID, Subject: "B's ticket", Status: helpdesk.TicketStatusOpen,
		Priority: helpdesk.TicketPriorityNormal, AssigneeID: &agentB, CustomFields: map[string]any{},
	}
	repo.seedTicket(ticketAssignedToA)
	repo.seedTicket(ticketRequestedByA)
	repo.seedTicket(ticketAssignedToB)

	t.Run("without participant filter returns every tenant ticket", func(t *testing.T) {
		resp, err := srv.ListTickets(context.Background(), &helpdeskv1.ListTicketsRequest{TenantId: tenantID.String()})
		require.NoError(t, err)
		assert.Equal(t, int32(3), resp.GetTotal())
		assert.Len(t, resp.GetTickets(), 3)
	})

	t.Run("with participant filter returns only A's tickets, not B's", func(t *testing.T) {
		participant := agentA.String()
		resp, err := srv.ListTickets(context.Background(), &helpdeskv1.ListTicketsRequest{
			TenantId:      tenantID.String(),
			ParticipantId: &participant,
		})
		require.NoError(t, err)
		assert.Equal(t, int32(2), resp.GetTotal())
		ids := make(map[string]bool)
		for _, ticket := range resp.GetTickets() {
			ids[ticket.GetId()] = true
		}
		assert.True(t, ids[ticketAssignedToA.ID.String()])
		assert.True(t, ids[ticketRequestedByA.ID.String()])
		assert.False(t, ids[ticketAssignedToB.ID.String()])
	})
}
