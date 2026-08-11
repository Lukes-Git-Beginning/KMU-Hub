package server

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/inbox/adapter"
	"github.com/kmuhub/kmuhub/internal/inbox/message"
	"github.com/kmuhub/kmuhub/internal/inbox/routing"
	"github.com/kmuhub/kmuhub/internal/inbox/team"
	"github.com/kmuhub/kmuhub/internal/inbox/thread"
	"github.com/kmuhub/kmuhub/internal/models"
	inboxv1 "github.com/kmuhub/kmuhub/proto/inbox/v1"
)

// ============================================================================
// Stub repositories (server-package copies -- the inbox packages' own
// service_test.go stubs are internal to those packages)
// ============================================================================

type stubMessageRepo struct {
	mu       sync.Mutex
	messages map[uuid.UUID]*models.InboxMessage
}

func newStubMessageRepo() *stubMessageRepo {
	return &stubMessageRepo{messages: make(map[uuid.UUID]*models.InboxMessage)}
}

func (r *stubMessageRepo) mutate(tenantID, id uuid.UUID, fn func(*models.InboxMessage)) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.messages[id]
	if !ok || m.TenantID != tenantID {
		return message.ErrMessageNotFound
	}
	fn(m)
	return nil
}

func (r *stubMessageRepo) Create(_ context.Context, msg *models.InboxMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}
	cp := *msg
	r.messages[msg.ID] = &cp
	return nil
}

func (r *stubMessageRepo) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.InboxMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.messages[id]
	if !ok || m.TenantID != tenantID {
		return nil, message.ErrMessageNotFound
	}
	cp := *m
	return &cp, nil
}

func (r *stubMessageRepo) List(_ context.Context, filter message.ListFilter) ([]*models.InboxMessage, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*models.InboxMessage
	for _, m := range r.messages {
		if m.TenantID != filter.TenantID {
			continue
		}
		cp := *m
		out = append(out, &cp)
	}
	return out, len(out), nil
}

func (r *stubMessageRepo) Update(_ context.Context, msg *models.InboxMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.messages[msg.ID]; !ok {
		return message.ErrMessageNotFound
	}
	cp := *msg
	r.messages[msg.ID] = &cp
	return nil
}

func (r *stubMessageRepo) MarkRead(_ context.Context, tenantID, id uuid.UUID) error {
	return r.mutate(tenantID, id, func(m *models.InboxMessage) { m.IsRead = true })
}

func (r *stubMessageRepo) MarkUnread(_ context.Context, tenantID, id uuid.UUID) error {
	return r.mutate(tenantID, id, func(m *models.InboxMessage) { m.IsRead = false })
}

func (r *stubMessageRepo) ToggleStar(_ context.Context, tenantID, id uuid.UUID) error {
	return r.mutate(tenantID, id, func(m *models.InboxMessage) { m.IsStarred = !m.IsStarred })
}

func (r *stubMessageRepo) SetStatus(_ context.Context, tenantID, id uuid.UUID, status string) error {
	return r.mutate(tenantID, id, func(m *models.InboxMessage) { m.Status = status })
}

func (r *stubMessageRepo) AddTag(_ context.Context, tenantID, id uuid.UUID, tag string) error {
	return r.mutate(tenantID, id, func(m *models.InboxMessage) {
		if slices.Contains(m.Tags, tag) {
			return
		}
		m.Tags = append(m.Tags, tag)
	})
}

func (r *stubMessageRepo) RemoveTag(_ context.Context, tenantID, id uuid.UUID, tag string) error {
	return r.mutate(tenantID, id, func(m *models.InboxMessage) {
		out := m.Tags[:0]
		for _, t := range m.Tags {
			if t != tag {
				out = append(out, t)
			}
		}
		m.Tags = out
	})
}

func (r *stubMessageRepo) Archive(_ context.Context, tenantID, id uuid.UUID) error {
	return r.mutate(tenantID, id, func(m *models.InboxMessage) { m.IsArchived = true })
}

func (r *stubMessageRepo) Unarchive(_ context.Context, tenantID, id uuid.UUID) error {
	return r.mutate(tenantID, id, func(m *models.InboxMessage) { m.IsArchived = false })
}

func (r *stubMessageRepo) Snooze(_ context.Context, tenantID, id uuid.UUID, until time.Time) error {
	return r.mutate(tenantID, id, func(m *models.InboxMessage) { m.SnoozedUntil = &until })
}

func (r *stubMessageRepo) UnsnoozeExpired(_ context.Context) (int, error) { return 0, nil }

func (r *stubMessageRepo) AssignMessage(_ context.Context, tenantID, id, assigneeID uuid.UUID) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.messages[id]
	if !ok || m.TenantID != tenantID {
		return false, message.ErrMessageNotFound
	}
	if m.AssignedTo != nil {
		return false, nil
	}
	m.AssignedTo = &assigneeID
	return true, nil
}

// GetUnreadCounts mirrors the production query in postgres_repository.go:
// unread, not archived, and not currently snoozed (snoozed_until NULL or in
// the past), grouped by channel.
func (r *stubMessageRepo) GetUnreadCounts(_ context.Context, userID uuid.UUID) ([]models.UnreadCount, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	counted := map[string]int{}
	now := time.Now()
	for _, m := range r.messages {
		if m.UserID != userID || m.IsRead || m.IsArchived {
			continue
		}
		if m.SnoozedUntil != nil && m.SnoozedUntil.After(now) {
			continue
		}
		counted[m.Channel]++
	}
	var out []models.UnreadCount
	for ch, c := range counted {
		out = append(out, models.UnreadCount{Channel: ch, Count: c})
	}
	return out, nil
}

func (r *stubMessageRepo) BulkMarkRead(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) error {
	for _, id := range ids {
		if err := r.MarkRead(ctx, tenantID, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *stubMessageRepo) BulkArchive(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) error {
	for _, id := range ids {
		if err := r.Archive(ctx, tenantID, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *stubMessageRepo) GetBySourceID(_ context.Context, userID uuid.UUID, channel, sourceID string) (*models.InboxMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, m := range r.messages {
		if m.UserID == userID && m.Channel == channel && m.SourceID == sourceID {
			cp := *m
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *stubMessageRepo) NotifyDelivery(_ context.Context, _ string) error { return nil }

// --- team ---

type stubTeamRepo struct {
	mu      sync.Mutex
	inboxes map[uuid.UUID]*models.TeamInbox
	members map[uuid.UUID][]*models.TeamInboxMember
}

func newStubTeamRepo() *stubTeamRepo {
	return &stubTeamRepo{
		inboxes: make(map[uuid.UUID]*models.TeamInbox),
		members: make(map[uuid.UUID][]*models.TeamInboxMember),
	}
}

func (r *stubTeamRepo) CreateTeamInbox(_ context.Context, inbox *models.TeamInbox) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *inbox
	r.inboxes[inbox.ID] = &cp
	return nil
}

func (r *stubTeamRepo) UpdateTeamInbox(_ context.Context, inbox *models.TeamInbox) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.inboxes[inbox.ID]; !ok {
		return team.ErrTeamInboxNotFound
	}
	cp := *inbox
	r.inboxes[inbox.ID] = &cp
	return nil
}

func (r *stubTeamRepo) DeleteTeamInbox(_ context.Context, tenantID, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	ib, ok := r.inboxes[id]
	if !ok || ib.TenantID != tenantID {
		return team.ErrTeamInboxNotFound
	}
	delete(r.inboxes, id)
	delete(r.members, id)
	return nil
}

func (r *stubTeamRepo) GetTeamInbox(_ context.Context, tenantID, id uuid.UUID) (*models.TeamInbox, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ib, ok := r.inboxes[id]
	if !ok || ib.TenantID != tenantID {
		return nil, team.ErrTeamInboxNotFound
	}
	cp := *ib
	return &cp, nil
}

func (r *stubTeamRepo) ListByUser(_ context.Context, tenantID, userID uuid.UUID) ([]*models.TeamInbox, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*models.TeamInbox
	for id, ib := range r.inboxes {
		if ib.TenantID != tenantID {
			continue
		}
		for _, m := range r.members[id] {
			if m.UserID == userID {
				cp := *ib
				out = append(out, &cp)
				break
			}
		}
	}
	return out, nil
}

func (r *stubTeamRepo) AddMember(_ context.Context, member *models.TeamInboxMember) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, m := range r.members[member.TeamInboxID] {
		if m.UserID == member.UserID {
			return team.ErrAlreadyMember
		}
	}
	cp := *member
	r.members[member.TeamInboxID] = append(r.members[member.TeamInboxID], &cp)
	return nil
}

func (r *stubTeamRepo) RemoveMember(_ context.Context, teamInboxID, userID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	members := r.members[teamInboxID]
	for i, m := range members {
		if m.UserID == userID {
			r.members[teamInboxID] = append(members[:i], members[i+1:]...)
			return nil
		}
	}
	return nil
}

func (r *stubTeamRepo) ListMembers(_ context.Context, teamInboxID uuid.UUID) ([]*models.TeamInboxMember, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*models.TeamInboxMember
	for _, m := range r.members[teamInboxID] {
		cp := *m
		out = append(out, &cp)
	}
	return out, nil
}

func (r *stubTeamRepo) IsMember(_ context.Context, teamInboxID, userID uuid.UUID) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, m := range r.members[teamInboxID] {
		if m.UserID == userID {
			return true, nil
		}
	}
	return false, nil
}

func (r *stubTeamRepo) GetMemberRole(_ context.Context, teamInboxID, userID uuid.UUID) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, m := range r.members[teamInboxID] {
		if m.UserID == userID {
			return m.Role, nil
		}
	}
	return "", nil
}

func (r *stubTeamRepo) IncrementAssigneeIndex(_ context.Context, tenantID, teamInboxID uuid.UUID) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ib, ok := r.inboxes[teamInboxID]
	if !ok || ib.TenantID != tenantID {
		return 0, team.ErrTeamInboxNotFound
	}
	ib.NextAssigneeIndex++
	return ib.NextAssigneeIndex, nil
}

func (r *stubTeamRepo) GetMemberCount(_ context.Context, teamInboxID uuid.UUID) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.members[teamInboxID]), nil
}

func (r *stubTeamRepo) CountAdmins(_ context.Context, teamInboxID uuid.UUID) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, m := range r.members[teamInboxID] {
		if m.Role == "admin" {
			count++
		}
	}
	return count, nil
}

// --- routing ---

type stubRoutingRepo struct {
	mu    sync.Mutex
	rules map[uuid.UUID]*models.RoutingRule
}

func newStubRoutingRepo() *stubRoutingRepo {
	return &stubRoutingRepo{rules: make(map[uuid.UUID]*models.RoutingRule)}
}

func (r *stubRoutingRepo) Create(_ context.Context, rule *models.RoutingRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *rule
	r.rules[rule.ID] = &cp
	return nil
}

func (r *stubRoutingRepo) Update(_ context.Context, rule *models.RoutingRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.rules[rule.ID]; !ok {
		return routing.ErrRuleNotFound
	}
	cp := *rule
	r.rules[rule.ID] = &cp
	return nil
}

func (r *stubRoutingRepo) Delete(_ context.Context, tenantID, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	rule, ok := r.rules[id]
	if !ok || rule.TenantID != tenantID {
		return routing.ErrRuleNotFound
	}
	delete(r.rules, id)
	return nil
}

func (r *stubRoutingRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*models.RoutingRule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rule, ok := r.rules[id]
	if !ok || rule.TenantID != tenantID {
		return nil, routing.ErrRuleNotFound
	}
	cp := *rule
	return &cp, nil
}

func (r *stubRoutingRepo) ListActive(_ context.Context, tenantID uuid.UUID, channel *string) ([]*models.RoutingRule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*models.RoutingRule
	for _, rule := range r.rules {
		if rule.TenantID != tenantID || !rule.IsActive {
			continue
		}
		if channel != nil && (rule.Channel == nil || *rule.Channel != *channel) {
			continue
		}
		cp := *rule
		out = append(out, &cp)
	}
	return out, nil
}

func (r *stubRoutingRepo) ListAll(_ context.Context, tenantID uuid.UUID) ([]*models.RoutingRule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*models.RoutingRule
	for _, rule := range r.rules {
		if rule.TenantID == tenantID {
			cp := *rule
			out = append(out, &cp)
		}
	}
	return out, nil
}

// --- thread ---

type stubThreadRepo struct {
	mu       sync.Mutex
	messages map[uuid.UUID][]*thread.ThreadMessage // keyed by messageID
	canned   map[uuid.UUID]*thread.CannedResponse
}

func newStubThreadRepo() *stubThreadRepo {
	return &stubThreadRepo{
		messages: make(map[uuid.UUID][]*thread.ThreadMessage),
		canned:   make(map[uuid.UUID]*thread.CannedResponse),
	}
}

func (r *stubThreadRepo) AppendThreadMessage(_ context.Context, m *thread.ThreadMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	cp := *m
	r.messages[m.MessageID] = append(r.messages[m.MessageID], &cp)
	return nil
}

func (r *stubThreadRepo) SeedInboundIfEmpty(_ context.Context, tenantID, messageID uuid.UUID, senderName, body string, createdAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.messages[messageID]) > 0 {
		return nil
	}
	r.messages[messageID] = append(r.messages[messageID], &thread.ThreadMessage{
		ID:         uuid.New(),
		TenantID:   tenantID,
		MessageID:  messageID,
		AuthorName: senderName,
		Direction:  thread.DirectionInbound,
		Body:       body,
		CreatedAt:  createdAt,
	})
	return nil
}

func (r *stubThreadRepo) ListThreadMessages(_ context.Context, tenantID, messageID uuid.UUID) ([]*thread.ThreadMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*thread.ThreadMessage
	for _, m := range r.messages[messageID] {
		if m.TenantID == tenantID {
			cp := *m
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *stubThreadRepo) CreateCannedResponse(_ context.Context, c *thread.CannedResponse) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	cp := *c
	r.canned[c.ID] = &cp
	return nil
}

func (r *stubThreadRepo) ListCannedResponses(_ context.Context, tenantID uuid.UUID) ([]*thread.CannedResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*thread.CannedResponse
	for _, c := range r.canned {
		if c.TenantID == tenantID {
			cp := *c
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *stubThreadRepo) GetCannedResponse(_ context.Context, tenantID, id uuid.UUID) (*thread.CannedResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.canned[id]
	if !ok || c.TenantID != tenantID {
		return nil, thread.ErrCannedResponseNotFound
	}
	cp := *c
	return &cp, nil
}

func (r *stubThreadRepo) UpdateCannedResponse(_ context.Context, c *thread.CannedResponse) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.canned[c.ID]; !ok {
		return thread.ErrCannedResponseNotFound
	}
	cp := *c
	r.canned[c.ID] = &cp
	return nil
}

func (r *stubThreadRepo) DeleteCannedResponse(_ context.Context, tenantID, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.canned[id]
	if !ok || c.TenantID != tenantID {
		return thread.ErrCannedResponseNotFound
	}
	delete(r.canned, id)
	return nil
}

// ============================================================================
// Test helpers
// ============================================================================

type inboxTestDeps struct {
	messageRepo *stubMessageRepo
	teamRepo    *stubTeamRepo
	routingRepo *stubRoutingRepo
	threadRepo  *stubThreadRepo
}

// newTestInboxServer wires an InboxGRPCServer to real Service instances backed
// by in-memory stub repos (rather than the nil-service trick used by
// newTestDocumentServer): message.Reply/Forward need a live adapters registry
// to exercise their error path, and team.Service.ClaimMessage needs the same
// message repo instance the message service uses.
func newTestInboxServer() (*InboxGRPCServer, *inboxTestDeps) {
	msgRepo := newStubMessageRepo()
	teamRepo := newStubTeamRepo()
	routingRepo := newStubRoutingRepo()
	threadRepo := newStubThreadRepo()

	msgSvc := message.NewService(msgRepo, adapter.NewAdapterRegistry())
	teamSvc := team.NewService(teamRepo, msgRepo)
	routingSvc := routing.NewService(routingRepo, msgRepo, nil)
	threadSvc := thread.NewService(threadRepo)

	srv := NewInboxGRPCServer(msgSvc, teamSvc, routingSvc, threadSvc)
	return srv, &inboxTestDeps{
		messageRepo: msgRepo,
		teamRepo:    teamRepo,
		routingRepo: routingRepo,
		threadRepo:  threadRepo,
	}
}

func seedInboxMessage(t *testing.T, deps *inboxTestDeps, tenantID, userID uuid.UUID, mutate func(*models.InboxMessage)) *models.InboxMessage {
	t.Helper()
	msg := &models.InboxMessage{
		ID:         uuid.New(),
		TenantID:   tenantID,
		UserID:     userID,
		Channel:    "email",
		SourceID:   "src-1",
		SenderName: "Sender",
		Status:     "open",
		ReceivedAt: time.Now().UTC(),
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	if mutate != nil {
		mutate(msg)
	}
	require.NoError(t, deps.messageRepo.Create(context.Background(), msg))
	return msg
}

// ============================================================================
// Status family (SetMessageStatus)
// ============================================================================

func TestSetMessageStatus_HappyPath(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), nil)

	resp, err := srv.SetMessageStatus(tenantCtx(tenantID), &inboxv1.SetMessageStatusRequest{
		MessageId: msg.ID.String(),
		Status:    "resolved",
	})
	require.NoError(t, err)
	require.Equal(t, "resolved", resp.Message.Status)
}

func TestSetMessageStatus_InvalidStatus(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), nil)

	_, err := srv.SetMessageStatus(tenantCtx(tenantID), &inboxv1.SetMessageStatusRequest{
		MessageId: msg.ID.String(),
		Status:    "not-a-real-status",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSetMessageStatus_MissingTenant(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.SetMessageStatus(context.Background(), &inboxv1.SetMessageStatusRequest{
		MessageId: uuid.New().String(),
		Status:    "open",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSetMessageStatus_InvalidMessageID(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.SetMessageStatus(tenantCtx(uuid.New()), &inboxv1.SetMessageStatusRequest{
		MessageId: "not-a-uuid",
		Status:    "open",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSetMessageStatus_NotFound(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.SetMessageStatus(tenantCtx(uuid.New()), &inboxv1.SetMessageStatusRequest{
		MessageId: uuid.New().String(),
		Status:    "open",
	})
	requireGRPCCode(t, err, codes.NotFound)
}

// ============================================================================
// Tags family (AddMessageTag / RemoveMessageTag)
// ============================================================================

func TestAddMessageTag_HappyPath(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), nil)

	resp, err := srv.AddMessageTag(tenantCtx(tenantID), &inboxv1.AddMessageTagRequest{
		MessageId: msg.ID.String(),
		Tag:       "vip",
	})
	require.NoError(t, err)
	require.Contains(t, resp.Message.Tags, "vip")
}

func TestAddMessageTag_EmptyTagRejected(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), nil)

	_, err := srv.AddMessageTag(tenantCtx(tenantID), &inboxv1.AddMessageTagRequest{
		MessageId: msg.ID.String(),
		Tag:       "   ",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestAddMessageTag_InvalidMessageID(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.AddMessageTag(tenantCtx(uuid.New()), &inboxv1.AddMessageTagRequest{
		MessageId: "bad-id",
		Tag:       "vip",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestRemoveMessageTag_HappyPath(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), func(m *models.InboxMessage) {
		m.Tags = []string{"vip", "urgent"}
	})

	resp, err := srv.RemoveMessageTag(tenantCtx(tenantID), &inboxv1.RemoveMessageTagRequest{
		MessageId: msg.ID.String(),
		Tag:       "vip",
	})
	require.NoError(t, err)
	require.NotContains(t, resp.Message.Tags, "vip")
	require.Contains(t, resp.Message.Tags, "urgent")
}

func TestRemoveMessageTag_NotFound(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.RemoveMessageTag(tenantCtx(uuid.New()), &inboxv1.RemoveMessageTagRequest{
		MessageId: uuid.New().String(),
		Tag:       "vip",
	})
	requireGRPCCode(t, err, codes.NotFound)
}

// ============================================================================
// Thread family (ListThreadMessages)
// ============================================================================

func TestListThreadMessages_SeedsInboundOnFirstRead(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), func(m *models.InboxMessage) {
		m.SenderName = "Kunde Müller"
		m.Preview = "Hallo, ich habe eine Frage."
	})

	resp, err := srv.ListThreadMessages(tenantCtx(tenantID), &inboxv1.ListThreadMessagesRequest{
		MessageId: msg.ID.String(),
	})
	require.NoError(t, err)
	require.Len(t, resp.Messages, 1)
	require.Equal(t, "Kunde Müller", resp.Messages[0].AuthorName)
	require.Nil(t, resp.Messages[0].AuthorId)
}

func TestListThreadMessages_InvalidMessageID(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.ListThreadMessages(tenantCtx(uuid.New()), &inboxv1.ListThreadMessagesRequest{
		MessageId: "bad-id",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListThreadMessages_MessageNotFound(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.ListThreadMessages(tenantCtx(uuid.New()), &inboxv1.ListThreadMessagesRequest{
		MessageId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.NotFound)
}

// ============================================================================
// Forward / Reply family
// ============================================================================

func TestForwardMessage_AdapterNotFound(t *testing.T) {
	// The stub adapter registry has nothing registered, so any channel hits
	// message.ErrAdapterNotFound -> codes.Unimplemented. This is the only
	// reachable error path without wiring a real channel adapter.
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), nil)

	_, err := srv.ForwardMessage(tenantCtx(tenantID), &inboxv1.ForwardMessageRequest{
		MessageId: msg.ID.String(),
		UserId:    uuid.New().String(),
		To:        "someone@example.com",
	})
	requireGRPCCode(t, err, codes.Unimplemented)
}

func TestForwardMessage_InvalidUserID(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), nil)

	_, err := srv.ForwardMessage(tenantCtx(tenantID), &inboxv1.ForwardMessageRequest{
		MessageId: msg.ID.String(),
		UserId:    "bad-id",
		To:        "someone@example.com",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestForwardMessage_MessageNotFound(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.ForwardMessage(tenantCtx(uuid.New()), &inboxv1.ForwardMessageRequest{
		MessageId: uuid.New().String(),
		UserId:    uuid.New().String(),
		To:        "someone@example.com",
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestReplyToMessage_AdapterNotFound(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), nil)

	_, err := srv.ReplyToMessage(tenantCtx(tenantID), &inboxv1.ReplyToMessageRequest{
		MessageId: msg.ID.String(),
		UserId:    uuid.New().String(),
		Body:      "Danke fuer Ihre Nachricht.",
	})
	requireGRPCCode(t, err, codes.Unimplemented)
}

func TestReplyToMessage_InvalidMessageID(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.ReplyToMessage(tenantCtx(uuid.New()), &inboxv1.ReplyToMessageRequest{
		MessageId: "bad-id",
		UserId:    uuid.New().String(),
		Body:      "x",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// Empty-list wire shape ([] not null)
// ============================================================================

func TestListMessages_EmptyResultIsNotNil(t *testing.T) {
	srv, _ := newTestInboxServer()
	tenantID := uuid.New()
	userID := uuid.New()

	resp, err := srv.ListMessages(tenantCtx(tenantID), &inboxv1.ListMessagesRequest{
		UserId: userID.String(),
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Messages)
	require.Empty(t, resp.Messages)
}

func TestListTeamInboxes_EmptyResultIsNotNil(t *testing.T) {
	srv, _ := newTestInboxServer()
	resp, err := srv.ListTeamInboxes(tenantCtx(uuid.New()), &inboxv1.ListTeamInboxesRequest{
		UserId: uuid.New().String(),
	})
	require.NoError(t, err)
	require.NotNil(t, resp.TeamInboxes)
	require.Empty(t, resp.TeamInboxes)
}

func TestListTeamMembers_EmptyResultIsNotNil(t *testing.T) {
	srv, _ := newTestInboxServer()
	resp, err := srv.ListTeamMembers(context.Background(), &inboxv1.ListTeamMembersRequest{
		TeamInboxId: uuid.New().String(),
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Members)
	require.Empty(t, resp.Members)
}

func TestListRoutingRules_EmptyResultIsNotNil(t *testing.T) {
	srv, _ := newTestInboxServer()
	resp, err := srv.ListRoutingRules(tenantCtx(uuid.New()), &inboxv1.ListRoutingRulesRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp.Rules)
	require.Empty(t, resp.Rules)
}

func TestGetUnreadCount_EmptyResultIsNotNil(t *testing.T) {
	srv, _ := newTestInboxServer()
	resp, err := srv.GetUnreadCount(context.Background(), &inboxv1.GetUnreadCountRequest{
		UserId: uuid.New().String(),
	})
	require.NoError(t, err)
	require.NotNil(t, resp.ByChannel)
	require.Empty(t, resp.ByChannel)
}

// ============================================================================
// Error mapping (mapInboxError)
// ============================================================================

func TestMapInboxError_AllSentinels(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"MessageNotFound", message.ErrMessageNotFound, codes.NotFound},
		{"AlreadyAssigned", message.ErrAlreadyAssigned, codes.AlreadyExists},
		{"InvalidSnoozeTime", message.ErrInvalidSnoozeTime, codes.InvalidArgument},
		{"AdapterNotFound", message.ErrAdapterNotFound, codes.Unimplemented},
		{"DuplicateMessage", message.ErrDuplicateMessage, codes.AlreadyExists},
		{"InvalidStatus", message.ErrInvalidStatus, codes.InvalidArgument},
		{"InvalidTag", message.ErrInvalidTag, codes.InvalidArgument},
		{"ForwardNotSupported", message.ErrForwardNotSupported, codes.Unimplemented},
		{"CannedResponseNotFound", thread.ErrCannedResponseNotFound, codes.NotFound},
		{"TeamInboxNotFound", team.ErrTeamInboxNotFound, codes.NotFound},
		{"NotTeamAdmin", team.ErrNotTeamAdmin, codes.PermissionDenied},
		{"NotTeamMember", team.ErrNotTeamMember, codes.PermissionDenied},
		{"AlreadyMember", team.ErrAlreadyMember, codes.AlreadyExists},
		{"CannotRemoveLastAdmin", team.ErrCannotRemoveLastAdmin, codes.FailedPrecondition},
		{"ManualClaimInRoundRobin", team.ErrManualClaimInRoundRobin, codes.FailedPrecondition},
		{"RuleNotFound", routing.ErrRuleNotFound, codes.NotFound},
		{"InvalidConditions", routing.ErrInvalidConditions, codes.InvalidArgument},
		{"InvalidAction", routing.ErrInvalidAction, codes.InvalidArgument},
		{"Unknown", errUnmappedInbox, codes.Internal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireGRPCCode(t, mapInboxError(tc.err), tc.code)
		})
	}
}

var errUnmappedInbox = &unmappedInboxErr{}

type unmappedInboxErr struct{}

func (e *unmappedInboxErr) Error() string { return "unmapped inbox error" }

// ============================================================================
// Converters (7)
// ============================================================================

func TestToInboxMessageInfo_Populated(t *testing.T) {
	senderID := uuid.New()
	senderEmail := "kunde@example.com"
	snoozedUntil := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	assignedTo := uuid.New()
	teamInboxID := uuid.New()
	crmContactID := uuid.New()

	m := &models.InboxMessage{
		ID:           uuid.New(),
		UserID:       uuid.New(),
		Channel:      "chat",
		SourceID:     "src-1",
		SenderName:   "Kunde",
		SenderID:     &senderID,
		SenderEmail:  &senderEmail,
		Subject:      "Betreff",
		Preview:      "Vorschau",
		IsRead:       true,
		IsStarred:    true,
		IsArchived:   false,
		Status:       "open",
		SnoozedUntil: &snoozedUntil,
		AssignedTo:   &assignedTo,
		TeamInboxID:  &teamInboxID,
		Tags:         []string{"vip"},
		DeepLink:     "kmuhub://inbox/1",
		CRMContactID: &crmContactID,
		Metadata:     []byte(`{"foo":"bar"}`),
		ReceivedAt:   time.Now().UTC(),
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	info := toInboxMessageInfo(m)
	require.Equal(t, m.ID.String(), info.Id)
	require.Equal(t, inboxv1.Channel_CHANNEL_CHAT, info.Channel)
	require.Equal(t, senderID.String(), *info.SenderId)
	require.Equal(t, senderEmail, *info.SenderEmail)
	require.NotNil(t, info.SnoozedUntil)
	require.Equal(t, assignedTo.String(), *info.AssignedTo)
	require.Equal(t, teamInboxID.String(), *info.TeamInboxId)
	require.Equal(t, crmContactID.String(), *info.CrmContactId)
	require.NotNil(t, info.Metadata)
	require.Equal(t, "bar", info.Metadata.Fields["foo"].GetStringValue())
}

func TestToInboxMessageInfo_NilOptionalFields(t *testing.T) {
	m := &models.InboxMessage{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		Channel:    "email",
		ReceivedAt: time.Now().UTC(),
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	info := toInboxMessageInfo(m)
	require.Nil(t, info.SenderId)
	require.Nil(t, info.SenderEmail)
	require.Nil(t, info.SnoozedUntil)
	require.Nil(t, info.AssignedTo)
	require.Nil(t, info.TeamInboxId)
	require.Nil(t, info.CrmContactId)
	require.Nil(t, info.Metadata)
}

func TestToTeamInboxInfo_WithAndWithoutDescription(t *testing.T) {
	createdBy := uuid.New()
	base := &models.TeamInbox{
		ID:             uuid.New(),
		Name:           "Support",
		AssignmentMode: "round_robin",
		Visibility:     "private",
		CreatedBy:      createdBy,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	info := toTeamInboxInfo(base)
	require.Equal(t, "", info.Description)
	require.Equal(t, inboxv1.AssignmentMode_ASSIGNMENT_MODE_ROUND_ROBIN, info.AssignmentMode)
	require.Equal(t, inboxv1.TeamInboxVisibility_VISIBILITY_PRIVATE, info.Visibility)

	desc := "Team fuer Support-Anfragen"
	base.Description = &desc
	info = toTeamInboxInfo(base)
	require.Equal(t, desc, info.Description)
}

func TestToTeamMemberInfo(t *testing.T) {
	m := &models.TeamInboxMember{
		TeamInboxID: uuid.New(),
		UserID:      uuid.New(),
		Role:        "admin",
		CreatedAt:   time.Now().UTC(),
	}
	info := toTeamMemberInfo(m)
	require.Equal(t, m.TeamInboxID.String(), info.TeamInboxId)
	require.Equal(t, m.UserID.String(), info.UserId)
	require.Equal(t, inboxv1.TeamMemberRole_TEAM_ROLE_ADMIN, info.Role)
}

func TestToRoutingRuleInfo_WithAndWithoutChannel(t *testing.T) {
	rule := &models.RoutingRule{
		ID:         uuid.New(),
		Name:       "Auto-Tag VIP",
		Conditions: []byte(`{"field":"sender_email","operator":"contains","value":"@vip.de"}`),
		Actions:    []byte(`{"actions":[{"type":"add_tag","config":{"tag":"vip"}}]}`),
		Priority:   1,
		IsActive:   true,
		CreatedBy:  uuid.New(),
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	info := toRoutingRuleInfo(rule)
	require.Nil(t, info.Channel)
	require.NotNil(t, info.Conditions)
	require.NotNil(t, info.Actions)

	ch := "email"
	rule.Channel = &ch
	info = toRoutingRuleInfo(rule)
	require.NotNil(t, info.Channel)
	require.Equal(t, inboxv1.Channel_CHANNEL_EMAIL, *info.Channel)
}

func TestToThreadMessageInfo_WithAndWithoutAuthor(t *testing.T) {
	tm := &thread.ThreadMessage{
		ID:         uuid.New(),
		MessageID:  uuid.New(),
		AuthorName: "System",
		Direction:  thread.DirectionInbound,
		Body:       "Hallo",
		CreatedAt:  time.Now().UTC(),
	}
	info := toThreadMessageInfo(tm)
	require.Nil(t, info.AuthorId)

	authorID := uuid.New()
	tm.AuthorID = &authorID
	info = toThreadMessageInfo(tm)
	require.Equal(t, authorID.String(), *info.AuthorId)
}

func TestToCannedResponseInfo(t *testing.T) {
	c := &thread.CannedResponse{
		ID:        uuid.New(),
		Name:      "Begruessung",
		Body:      "Hallo, danke fuer Ihre Nachricht.",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	info := toCannedResponseInfo(c)
	require.Equal(t, c.ID.String(), info.Id)
	require.Equal(t, c.Name, info.Name)
	require.Equal(t, c.Body, info.Body)
}

func TestProtoMessageToInboxMessage_Nil(t *testing.T) {
	m := protoMessageToInboxMessage(nil)
	require.NotNil(t, m)
	require.Equal(t, uuid.Nil, m.ID)
}

func TestProtoMessageToInboxMessage_RoundTrip(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	senderID := uuid.New()
	senderEmail := "kunde@example.com"
	receivedAt := timestamppb.New(time.Now().UTC().Truncate(time.Second))

	p := &inboxv1.InboxMessageInfo{
		Id:          id.String(),
		UserId:      userID.String(),
		Channel:     inboxv1.Channel_CHANNEL_CHAT,
		SourceId:    "src-1",
		SenderName:  "Kunde",
		SenderId:    ptrString(senderID.String()),
		SenderEmail: &senderEmail,
		Subject:     "Betreff",
		Preview:     "Vorschau",
		IsRead:      true,
		Status:      "open",
		Tags:        []string{"vip"},
		DeepLink:    "kmuhub://inbox/1",
		ReceivedAt:  receivedAt,
	}

	m := protoMessageToInboxMessage(p)
	require.Equal(t, id, m.ID)
	require.Equal(t, userID, m.UserID)
	require.Equal(t, "chat", m.Channel)
	require.Equal(t, senderID, *m.SenderID)
	require.Equal(t, senderEmail, *m.SenderEmail)
	require.Equal(t, receivedAt.AsTime(), m.ReceivedAt)
}

func TestProtoMessageToInboxMessage_InvalidUUIDsAreIgnored(t *testing.T) {
	p := &inboxv1.InboxMessageInfo{
		Id:       "not-a-uuid",
		UserId:   "also-not-a-uuid",
		SenderId: ptrString("nope"),
	}
	m := protoMessageToInboxMessage(p)
	require.Equal(t, uuid.Nil, m.ID)
	require.Equal(t, uuid.Nil, m.UserID)
	require.Nil(t, m.SenderID)
}

func ptrString(s string) *string { return &s }

// ============================================================================
// Enum round trips
// ============================================================================

func TestChannelRoundTrip(t *testing.T) {
	cases := []struct {
		domain string
		proto  inboxv1.Channel
	}{
		{"email", inboxv1.Channel_CHANNEL_EMAIL},
		{"chat", inboxv1.Channel_CHANNEL_CHAT},
		{"notification", inboxv1.Channel_CHANNEL_NOTIFICATION},
	}
	for _, tc := range cases {
		require.Equal(t, tc.proto, stringToChannel(tc.domain))
		require.Equal(t, tc.domain, channelToString(tc.proto))
	}
	require.Equal(t, inboxv1.Channel_CHANNEL_UNSPECIFIED, stringToChannel("unknown"))
	require.Equal(t, "", channelToString(inboxv1.Channel_CHANNEL_UNSPECIFIED))
}

func TestAssignmentModeRoundTrip(t *testing.T) {
	cases := []struct {
		domain string
		proto  inboxv1.AssignmentMode
	}{
		{"manual", inboxv1.AssignmentMode_ASSIGNMENT_MODE_MANUAL},
		{"round_robin", inboxv1.AssignmentMode_ASSIGNMENT_MODE_ROUND_ROBIN},
	}
	for _, tc := range cases {
		require.Equal(t, tc.proto, stringToAssignmentMode(tc.domain))
		require.Equal(t, tc.domain, assignmentModeToString(tc.proto))
	}
	require.Equal(t, inboxv1.AssignmentMode_ASSIGNMENT_MODE_UNSPECIFIED, stringToAssignmentMode("unknown"))
	// Unspecified degrades to "manual", not an empty string -- the proto default
	// must not silently create a team inbox with no assignment strategy at all.
	require.Equal(t, "manual", assignmentModeToString(inboxv1.AssignmentMode_ASSIGNMENT_MODE_UNSPECIFIED))
}

func TestVisibilityRoundTrip(t *testing.T) {
	cases := []struct {
		domain string
		proto  inboxv1.TeamInboxVisibility
	}{
		{"open", inboxv1.TeamInboxVisibility_VISIBILITY_OPEN},
		{"private", inboxv1.TeamInboxVisibility_VISIBILITY_PRIVATE},
	}
	for _, tc := range cases {
		require.Equal(t, tc.proto, stringToVisibility(tc.domain))
		require.Equal(t, tc.domain, visibilityToString(tc.proto))
	}
	require.Equal(t, inboxv1.TeamInboxVisibility_VISIBILITY_UNSPECIFIED, stringToVisibility("unknown"))
	require.Equal(t, "open", visibilityToString(inboxv1.TeamInboxVisibility_VISIBILITY_UNSPECIFIED))
}

func TestTeamMemberRoleRoundTrip(t *testing.T) {
	cases := []struct {
		domain string
		proto  inboxv1.TeamMemberRole
	}{
		{"admin", inboxv1.TeamMemberRole_TEAM_ROLE_ADMIN},
		{"member", inboxv1.TeamMemberRole_TEAM_ROLE_MEMBER},
	}
	for _, tc := range cases {
		require.Equal(t, tc.proto, stringToTeamMemberRole(tc.domain))
		require.Equal(t, tc.domain, teamMemberRoleToString(tc.proto))
	}
	require.Equal(t, inboxv1.TeamMemberRole_TEAM_ROLE_UNSPECIFIED, stringToTeamMemberRole("unknown"))
	require.Equal(t, "member", teamMemberRoleToString(inboxv1.TeamMemberRole_TEAM_ROLE_UNSPECIFIED))
}

// ============================================================================
// structToJSON / jsonToStruct helpers
// ============================================================================

func TestStructToJSON_Nil(t *testing.T) {
	data, err := structToJSON(nil)
	require.NoError(t, err)
	require.JSONEq(t, "{}", string(data))
}

func TestStructToJSON_Populated(t *testing.T) {
	s, err := structpb.NewStruct(map[string]any{"field": "sender_email"})
	require.NoError(t, err)
	data, err := structToJSON(s)
	require.NoError(t, err)
	require.JSONEq(t, `{"field":"sender_email"}`, string(data))
}

func TestJSONToStruct_Nil(t *testing.T) {
	s, err := jsonToStruct(nil)
	require.NoError(t, err)
	require.Nil(t, s)
}

func TestJSONToStruct_Object(t *testing.T) {
	s, err := jsonToStruct([]byte(`{"field":"x"}`))
	require.NoError(t, err)
	require.Equal(t, "x", s.Fields["field"].GetStringValue())
}

func TestJSONToStruct_TopLevelArrayReturnsError(t *testing.T) {
	// The "wrap it" fallback in jsonToStruct re-parses into a map[string]any,
	// which itself fails for a bare JSON array -- so this returns the original
	// protojson error rather than actually wrapping it. Not a live bug: routing
	// rule actions are always persisted as a wrapped object ({"actions":[...]}),
	// so CreateRoutingRule/UpdateRoutingRule never feed a top-level array here.
	_, err := jsonToStruct([]byte(`[{"type":"add_tag"}]`))
	require.Error(t, err)
}

// ============================================================================
// Page token helpers
// ============================================================================

func TestPageTokenRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Nanosecond)
	id := uuid.New()
	token := buildPageToken(now, id)
	gotTime, gotID, err := parsePageToken(token)
	require.NoError(t, err)
	require.True(t, now.Equal(gotTime))
	require.Equal(t, id, gotID)
}

func TestParsePageToken_InvalidFormat(t *testing.T) {
	_, _, err := parsePageToken("no-pipe-separator")
	require.Error(t, err)
}
