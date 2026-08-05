package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/helpdesk"
	"github.com/kmuhub/kmuhub/internal/middleware"
	helpdeskv1 "github.com/kmuhub/kmuhub/proto/helpdesk/v1"
	inboxv1 "github.com/kmuhub/kmuhub/proto/inbox/v1"
	settingsv1 "github.com/kmuhub/kmuhub/proto/settings/v1"
)

// HelpdeskGRPCServer implements the HelpdeskService gRPC server.
type HelpdeskGRPCServer struct {
	helpdeskv1.UnimplementedHelpdeskServiceServer
	svc *helpdesk.Service
	// inboxClient is optional; nil if the inbox (notification) service is
	// unreachable. Only CreateTicketFromMessage needs it -- every other RPC
	// degrades gracefully without it.
	inboxClient inboxv1.InboxServiceClient
	// settingsClient is optional; nil if the settings service (co-located
	// with auth) is unreachable. GetCsatConfig/SetCsatConfig degrade
	// gracefully without it (Get returns defaults, Set is Unavailable).
	settingsClient settingsv1.SettingsServiceClient
}

// NewHelpdeskGRPCServer creates a new Helpdesk gRPC server. inboxClient and
// settingsClient may be nil, which disables CreateTicketFromMessage
// (Unavailable) resp. degrades CSAT config access, but leaves every other RPC
// unaffected.
func NewHelpdeskGRPCServer(svc *helpdesk.Service, inboxClient inboxv1.InboxServiceClient, settingsClient settingsv1.SettingsServiceClient) *HelpdeskGRPCServer {
	return &HelpdeskGRPCServer{svc: svc, inboxClient: inboxClient, settingsClient: settingsClient}
}

// ============================================================================
// CSAT tenant configuration (no dedicated RPC yet -- Go-level only, for
// csat-survey-token (BACKLOG.yml) to call once ticket-close needs to decide
// whether to mint a survey token. No FE contract binds a route to this yet,
// so none is added here (see JOURNAL.md, csat-tenant-config).
// ============================================================================

// GetCsatConfig returns the tenant's CSAT survey configuration, defaulting to
// helpdesk.DefaultCsatConfig() when the tenant has never written one or the
// settings service is unreachable -- CSAT config is a secondary concern that
// must not break ticket flows.
func (s *HelpdeskGRPCServer) GetCsatConfig(ctx context.Context, tenantID uuid.UUID) (helpdesk.CsatConfig, error) {
	if s.settingsClient == nil {
		return helpdesk.DefaultCsatConfig(), nil
	}
	resp, err := s.settingsClient.GetTenantSettings(ctx, &settingsv1.GetTenantSettingsRequest{
		TenantId: tenantID.String(),
		ModuleId: helpdesk.CsatConfigModuleID,
	})
	if err != nil {
		return helpdesk.CsatConfig{}, err
	}
	return csatConfigFromEntries(resp.GetEntries()), nil
}

// SetCsatConfig validates and writes the tenant's CSAT survey configuration.
// RBAC (admin or module-lead for "helpdesk_csat") is enforced by the settings
// service's PutTenantSettings, not here.
func (s *HelpdeskGRPCServer) SetCsatConfig(ctx context.Context, tenantID, callerID uuid.UUID, cfg helpdesk.CsatConfig) (helpdesk.CsatConfig, error) {
	if err := helpdesk.ValidateCsatConfig(cfg); err != nil {
		return helpdesk.CsatConfig{}, err
	}
	if s.settingsClient == nil {
		return helpdesk.CsatConfig{}, status.Error(codes.Unavailable, "settings service not available")
	}
	value, err := csatConfigToValue(cfg)
	if err != nil {
		return helpdesk.CsatConfig{}, err
	}
	if _, err := s.settingsClient.PutTenantSettings(ctx, &settingsv1.PutTenantSettingsRequest{
		TenantId:  tenantID.String(),
		ModuleId:  helpdesk.CsatConfigModuleID,
		UpdatedBy: callerID.String(),
		Entries: []*settingsv1.SettingEntry{
			{Key: helpdesk.CsatConfigEntryKey, Value: value},
		},
	}); err != nil {
		return helpdesk.CsatConfig{}, err
	}
	return cfg, nil
}

// csatConfigToValue encodes a CsatConfig as the single JSON object stored
// under helpdesk.CsatConfigEntryKey (a full replace, not a sparse patch --
// there is exactly one CSAT config per tenant).
func csatConfigToValue(cfg helpdesk.CsatConfig) (*structpb.Value, error) {
	return structpb.NewValue(map[string]any{
		"enabled":       cfg.Enabled,
		"delay_minutes": float64(cfg.SurveyDelayMinutes),
		"question":      cfg.SurveyQuestion,
	})
}

// csatConfigFromEntries decodes the CsatConfigEntryKey entry, defaulting any
// field that is absent or the wrong JSON type -- including every field, when
// the tenant has no row yet (entries is empty).
func csatConfigFromEntries(entries []*settingsv1.SettingEntry) helpdesk.CsatConfig {
	cfg := helpdesk.DefaultCsatConfig()
	for _, e := range entries {
		if e.GetKey() != helpdesk.CsatConfigEntryKey {
			continue
		}
		raw, ok := e.GetValue().AsInterface().(map[string]any)
		if !ok {
			continue
		}
		if v, ok := raw["enabled"].(bool); ok {
			cfg.Enabled = v
		}
		if v, ok := raw["delay_minutes"].(float64); ok {
			cfg.SurveyDelayMinutes = int32(v)
		}
		if v, ok := raw["question"].(string); ok {
			cfg.SurveyQuestion = v
		}
	}
	return cfg
}

// ============================================================================
// Ticket Lifecycle (8 RPCs)
// ============================================================================

func (s *HelpdeskGRPCServer) CreateTicket(ctx context.Context, req *helpdeskv1.CreateTicketRequest) (*helpdeskv1.Ticket, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing or invalid tenant_id")
	}

	requesterID, err := uuid.Parse(req.GetRequesterId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid requester_id: %v", err)
	}

	var assigneeID *uuid.UUID
	if req.AssigneeId != nil {
		id, err := uuid.Parse(req.GetAssigneeId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid assignee_id: %v", err)
		}
		assigneeID = &id
	}

	var queueID *uuid.UUID
	if req.QueueId != nil {
		id, err := uuid.Parse(req.GetQueueId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid queue_id: %v", err)
		}
		queueID = &id
	}

	var contactID *uuid.UUID
	if req.ContactId != nil {
		id, err := uuid.Parse(req.GetContactId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid contact_id: %v", err)
		}
		contactID = &id
	}

	var orgID *uuid.UUID
	if req.OrgId != nil {
		id, err := uuid.Parse(req.GetOrgId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid org_id: %v", err)
		}
		orgID = &id
	}

	t, err := s.svc.CreateTicket(ctx, tenantID, requesterID, req.GetSubject(), req.GetPriority(), assigneeID, queueID, req.GetDescription(), req.GetCategory(), contactID, orgID)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return ticketToProto(t), nil
}

func (s *HelpdeskGRPCServer) GetTicket(ctx context.Context, req *helpdeskv1.GetTicketRequest) (*helpdeskv1.Ticket, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	id, err := uuid.Parse(req.GetTicketId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid ticket_id: %v", err)
	}
	t, err := s.svc.GetTicket(ctx, id, tenantID)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return ticketToProto(t), nil
}

func (s *HelpdeskGRPCServer) ListTickets(ctx context.Context, req *helpdeskv1.ListTicketsRequest) (*helpdeskv1.ListTicketsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing or invalid tenant_id")
	}

	var statusFilter *string
	if req.StatusFilter != nil {
		sf := req.GetStatusFilter()
		statusFilter = &sf
	}

	var participantID *uuid.UUID
	if req.ParticipantId != nil {
		pid, parseErr := uuid.Parse(req.GetParticipantId())
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid participant_id")
		}
		participantID = &pid
	}

	var contactID *uuid.UUID
	if req.ContactId != nil {
		cid, parseErr := uuid.Parse(req.GetContactId())
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
		}
		contactID = &cid
	}

	var orgID *uuid.UUID
	if req.OrgId != nil {
		oid, parseErr := uuid.Parse(req.GetOrgId())
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid org_id")
		}
		orgID = &oid
	}

	page := max(int(req.GetPage()), 1)
	pageSize := int(req.GetPageSize())
	if pageSize < 1 {
		pageSize = 20
	}

	tickets, total, err := s.svc.ListTickets(ctx, tenantID, statusFilter, participantID, contactID, orgID, page, pageSize)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}

	protoTickets := make([]*helpdeskv1.Ticket, len(tickets))
	for i, t := range tickets {
		protoTickets[i] = ticketToProto(t)
	}
	return &helpdeskv1.ListTicketsResponse{
		Tickets: protoTickets,
		Total:   int32(total),
	}, nil
}

func (s *HelpdeskGRPCServer) UpdateTicket(ctx context.Context, req *helpdeskv1.UpdateTicketRequest) (*helpdeskv1.Ticket, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	id, err := uuid.Parse(req.GetTicketId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid ticket_id: %v", err)
	}

	var assigneeID *uuid.UUID
	if req.AssigneeId != nil {
		aid, err := uuid.Parse(req.GetAssigneeId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid assignee_id: %v", err)
		}
		assigneeID = &aid
	}

	var queueID *uuid.UUID
	if req.QueueId != nil {
		qid, err := uuid.Parse(req.GetQueueId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid queue_id: %v", err)
		}
		queueID = &qid
	}

	var contactID *uuid.UUID
	if req.ContactId != nil {
		cid, err := uuid.Parse(req.GetContactId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid contact_id: %v", err)
		}
		contactID = &cid
	}

	var orgID *uuid.UUID
	if req.OrgId != nil {
		oid, err := uuid.Parse(req.GetOrgId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid org_id: %v", err)
		}
		orgID = &oid
	}

	t, err := s.svc.UpdateTicket(ctx, id, tenantID, req.Subject, req.Priority, assigneeID, queueID, contactID, orgID)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return ticketToProto(t), nil
}

func (s *HelpdeskGRPCServer) CloseTicket(ctx context.Context, req *helpdeskv1.CloseTicketRequest) (*helpdeskv1.Ticket, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	id, err := uuid.Parse(req.GetTicketId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid ticket_id: %v", err)
	}
	t, err := s.svc.CloseTicket(ctx, id, tenantID)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	s.issueCsatSurvey(ctx, tenantID, t)
	return ticketToProto(t), nil
}

// issueCsatSurvey mints a CSAT survey link for a ticket that was just closed,
// if the tenant has surveys enabled.
//
// Every failure in here is logged and swallowed: closing the ticket is what the
// caller asked for and has already succeeded at this point, the survey is the
// side errand. Returning an error would roll a successful close back into a
// failed RPC for the agent who clicked "close".
func (s *HelpdeskGRPCServer) issueCsatSurvey(ctx context.Context, tenantID uuid.UUID, t *helpdesk.Ticket) {
	cfg, err := s.GetCsatConfig(ctx, tenantID)
	if err != nil {
		slog.WarnContext(ctx, "helpdesk: csat config unavailable, no survey token issued",
			"ticket_id", t.ID, "error", err)
		return
	}
	if _, err := s.svc.IssueCsatSurveyToken(ctx, t, cfg); err != nil {
		slog.WarnContext(ctx, "helpdesk: csat survey token not issued",
			"ticket_id", t.ID, "error", err)
	}
}

func (s *HelpdeskGRPCServer) SubmitCsat(ctx context.Context, req *helpdeskv1.SubmitCsatRequest) (*helpdeskv1.Ticket, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	id, err := uuid.Parse(req.GetTicketId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid ticket_id: %v", err)
	}
	// Range check stays in the service; here we only guard the int32 -> int16
	// narrowing so an out-of-range wire value cannot wrap into a valid rating.
	rating := req.GetRating()
	if rating < 1 || rating > 5 {
		return nil, status.Error(codes.InvalidArgument, helpdesk.ErrInvalidCsatRating.Error())
	}

	var comment *string
	if req.Comment != nil {
		c := strings.TrimSpace(req.GetComment())
		if c != "" {
			comment = &c
		}
	}

	t, err := s.svc.SubmitCsat(ctx, id, tenantID, int16(rating), comment)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return ticketToProto(t), nil
}

// SubmitCsatByToken redeems a survey link. Unlike every other RPC here it does
// NOT read a tenant from the context: the public caller has none, and the
// token resolves the tenant itself inside the service. Nothing about the
// ticket goes back over the wire beyond the number the customer already has
// from the invitation mail.
func (s *HelpdeskGRPCServer) SubmitCsatByToken(
	ctx context.Context,
	req *helpdeskv1.SubmitCsatByTokenRequest,
) (*helpdeskv1.SubmitCsatByTokenResponse, error) {
	// Guards the int32 -> int16 narrowing so an out-of-range wire value cannot
	// wrap into a valid rating; the range check itself lives in the service.
	rating := req.GetRating()
	if rating < 1 || rating > 5 {
		return nil, status.Error(codes.InvalidArgument, helpdesk.ErrInvalidCsatRating.Error())
	}

	var comment *string
	if req.Comment != nil {
		c := strings.TrimSpace(req.GetComment())
		if c != "" {
			comment = &c
		}
	}

	res, err := s.svc.SubmitCsatByToken(ctx, req.GetToken(), int16(rating), comment)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return &helpdeskv1.SubmitCsatByTokenResponse{
		TicketNumber: int32(res.TicketNumber), //nolint:gosec // per-tenant counter, far below int32
		Rating:       int32(res.Rating),
	}, nil
}

func (s *HelpdeskGRPCServer) ReopenTicket(ctx context.Context, req *helpdeskv1.ReopenTicketRequest) (*helpdeskv1.Ticket, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	id, err := uuid.Parse(req.GetTicketId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid ticket_id: %v", err)
	}
	t, err := s.svc.ReopenTicket(ctx, id, tenantID)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return ticketToProto(t), nil
}

func (s *HelpdeskGRPCServer) AssignTicket(ctx context.Context, req *helpdeskv1.AssignTicketRequest) (*helpdeskv1.Ticket, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	ticketID, err := uuid.Parse(req.GetTicketId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid ticket_id: %v", err)
	}
	assigneeID, err := uuid.Parse(req.GetAssigneeId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid assignee_id: %v", err)
	}
	t, err := s.svc.AssignTicket(ctx, ticketID, tenantID, assigneeID)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return ticketToProto(t), nil
}

func (s *HelpdeskGRPCServer) MergeTickets(ctx context.Context, req *helpdeskv1.MergeTicketsRequest) (*emptypb.Empty, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	sourceID, err := uuid.Parse(req.GetSourceTicketId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid source_ticket_id: %v", err)
	}
	targetID, err := uuid.Parse(req.GetTargetTicketId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid target_ticket_id: %v", err)
	}
	if err := s.svc.MergeTickets(ctx, tenantID, sourceID, targetID); err != nil {
		return nil, mapHelpdeskError(err)
	}
	return &emptypb.Empty{}, nil
}

// CreateTicketFromMessage fetches the source inbox message from the inbox
// service and delegates persistence + idempotency to the helpdesk service.
// The orchestration lives here rather than in the gateway, mirroring
// BizGRPCServer.CreateQuoteFromDeal: one RPC call does both steps.
func (s *HelpdeskGRPCServer) CreateTicketFromMessage(ctx context.Context, req *helpdeskv1.CreateTicketFromMessageRequest) (*helpdeskv1.CreateTicketFromMessageResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing or invalid tenant_id")
	}

	requesterID, err := uuid.Parse(req.GetRequesterId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid requester_id: %v", err)
	}

	messageID, err := uuid.Parse(req.GetMessageId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}

	if s.inboxClient == nil {
		return nil, status.Error(codes.Unavailable, "inbox service not available")
	}

	msgResp, err := s.inboxClient.GetMessage(ctx, &inboxv1.GetMessageRequest{MessageId: req.GetMessageId()})
	if err != nil {
		slog.Error("CreateTicketFromMessage: fetch inbox message failed", "message_id", req.GetMessageId(), "error", err)
		return nil, err
	}
	msg := msgResp.GetMessage()

	t, created, err := s.svc.CreateTicketFromMessage(ctx, tenantID, requesterID, messageID, channelToString(msg.GetChannel()), msg.GetSubject(), msg.GetPreview())
	if err != nil {
		return nil, mapHelpdeskError(err)
	}

	return &helpdeskv1.CreateTicketFromMessageResponse{
		Ticket:  ticketToProto(t),
		Created: created,
	}, nil
}

// ============================================================================
// Messages (2 RPCs)
// ============================================================================

func (s *HelpdeskGRPCServer) AddMessage(ctx context.Context, req *helpdeskv1.AddMessageRequest) (*helpdeskv1.TicketMessage, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	ticketID, err := uuid.Parse(req.GetTicketId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid ticket_id: %v", err)
	}
	authorID, err := uuid.Parse(req.GetAuthorId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid author_id: %v", err)
	}

	m, err := s.svc.AddMessage(ctx, ticketID, tenantID, authorID, req.GetBody(), req.GetInternal(), req.GetAttachments())
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return ticketMessageToProto(m), nil
}

func (s *HelpdeskGRPCServer) ListMessages(ctx context.Context, req *helpdeskv1.ListMessagesRequest) (*helpdeskv1.ListMessagesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	ticketID, err := uuid.Parse(req.GetTicketId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid ticket_id: %v", err)
	}
	msgs, err := s.svc.ListMessages(ctx, ticketID, tenantID)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	protoMsgs := make([]*helpdeskv1.TicketMessage, len(msgs))
	for i, m := range msgs {
		protoMsgs[i] = ticketMessageToProto(m)
	}
	return &helpdeskv1.ListMessagesResponse{Messages: protoMsgs}, nil
}

// ============================================================================
// Queue Management (4 RPCs)
// ============================================================================

func (s *HelpdeskGRPCServer) CreateQueue(ctx context.Context, req *helpdeskv1.CreateQueueRequest) (*helpdeskv1.TicketQueue, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing or invalid tenant_id")
	}

	var defaultAssigneeID *uuid.UUID
	if req.DefaultAssigneeId != nil {
		id, err := uuid.Parse(req.GetDefaultAssigneeId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid default_assignee_id: %v", err)
		}
		defaultAssigneeID = &id
	}

	var slaPolicyID *uuid.UUID
	if req.SlaPolicyId != nil {
		id, err := uuid.Parse(req.GetSlaPolicyId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid sla_policy_id: %v", err)
		}
		slaPolicyID = &id
	}

	q, err := s.svc.CreateQueue(ctx, tenantID, req.GetName(), defaultAssigneeID, slaPolicyID)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return ticketQueueToProto(q), nil
}

func (s *HelpdeskGRPCServer) UpdateQueue(ctx context.Context, req *helpdeskv1.UpdateQueueRequest) (*helpdeskv1.TicketQueue, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	id, err := uuid.Parse(req.GetQueueId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid queue_id: %v", err)
	}

	var defaultAssigneeID *uuid.UUID
	if req.DefaultAssigneeId != nil {
		aid, err := uuid.Parse(req.GetDefaultAssigneeId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid default_assignee_id: %v", err)
		}
		defaultAssigneeID = &aid
	}

	var slaPolicyID *uuid.UUID
	if req.SlaPolicyId != nil {
		sid, err := uuid.Parse(req.GetSlaPolicyId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid sla_policy_id: %v", err)
		}
		slaPolicyID = &sid
	}

	q, err := s.svc.UpdateQueue(ctx, id, tenantID, req.Name, defaultAssigneeID, slaPolicyID)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return ticketQueueToProto(q), nil
}

func (s *HelpdeskGRPCServer) ListQueues(ctx context.Context, req *helpdeskv1.ListQueuesRequest) (*helpdeskv1.ListQueuesResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing or invalid tenant_id")
	}
	queues, err := s.svc.ListQueues(ctx, tenantID)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	protoQueues := make([]*helpdeskv1.TicketQueue, len(queues))
	for i, q := range queues {
		protoQueues[i] = ticketQueueToProto(q)
	}
	return &helpdeskv1.ListQueuesResponse{Queues: protoQueues}, nil
}

func (s *HelpdeskGRPCServer) DeleteQueue(ctx context.Context, req *helpdeskv1.DeleteQueueRequest) (*emptypb.Empty, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	id, err := uuid.Parse(req.GetQueueId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid queue_id: %v", err)
	}
	if err := s.svc.DeleteQueue(ctx, id, tenantID); err != nil {
		return nil, mapHelpdeskError(err)
	}
	return &emptypb.Empty{}, nil
}

// ============================================================================
// Canned Responses (4 RPCs)
// ============================================================================

func (s *HelpdeskGRPCServer) CreateCannedResponse(ctx context.Context, req *helpdeskv1.CreateCannedResponseRequest) (*helpdeskv1.CannedResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing or invalid tenant_id")
	}
	cr, err := s.svc.CreateCannedResponse(ctx, tenantID, req.GetName(), req.GetBody())
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return cannedResponseToProto(cr), nil
}

func (s *HelpdeskGRPCServer) UpdateCannedResponse(ctx context.Context, req *helpdeskv1.UpdateCannedResponseRequest) (*helpdeskv1.CannedResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	id, err := uuid.Parse(req.GetCannedResponseId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canned_response_id: %v", err)
	}
	cr, err := s.svc.UpdateCannedResponse(ctx, id, tenantID, req.Name, req.Body)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return cannedResponseToProto(cr), nil
}

func (s *HelpdeskGRPCServer) DeleteCannedResponse(ctx context.Context, req *helpdeskv1.DeleteCannedResponseRequest) (*emptypb.Empty, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	id, err := uuid.Parse(req.GetCannedResponseId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canned_response_id: %v", err)
	}
	if err := s.svc.DeleteCannedResponse(ctx, id, tenantID); err != nil {
		return nil, mapHelpdeskError(err)
	}
	return &emptypb.Empty{}, nil
}

func (s *HelpdeskGRPCServer) ListCannedResponses(ctx context.Context, req *helpdeskv1.ListCannedResponsesRequest) (*helpdeskv1.ListCannedResponsesResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing or invalid tenant_id")
	}
	list, err := s.svc.ListCannedResponses(ctx, tenantID)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	proto := make([]*helpdeskv1.CannedResponse, len(list))
	for i, cr := range list {
		proto[i] = cannedResponseToProto(cr)
	}
	return &helpdeskv1.ListCannedResponsesResponse{CannedResponses: proto}, nil
}

// ============================================================================
// SLA Policies (5 RPCs)
// ============================================================================

func (s *HelpdeskGRPCServer) CreateSLAPolicy(ctx context.Context, req *helpdeskv1.CreateSLAPolicyRequest) (*helpdeskv1.SLAPolicy, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing or invalid tenant_id")
	}

	var businessHours map[string]any
	if req.BusinessHours != nil && req.GetBusinessHours() != "" {
		if err := json.Unmarshal([]byte(req.GetBusinessHours()), &businessHours); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid business_hours JSON: %v", err)
		}
	}

	p, err := s.svc.CreateSLAPolicy(ctx, tenantID, req.GetName(),
		int(req.GetFirstResponseMins()), int(req.GetResolutionMins()), businessHours)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return slaPolicyToProto(p), nil
}

func (s *HelpdeskGRPCServer) UpdateSLAPolicy(ctx context.Context, req *helpdeskv1.UpdateSLAPolicyRequest) (*helpdeskv1.SLAPolicy, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	id, err := uuid.Parse(req.GetSlaPolicyId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid sla_policy_id: %v", err)
	}

	var firstResponseMins *int
	if req.FirstResponseMins != nil {
		v := int(req.GetFirstResponseMins())
		firstResponseMins = &v
	}

	var resolutionMins *int
	if req.ResolutionMins != nil {
		v := int(req.GetResolutionMins())
		resolutionMins = &v
	}

	var businessHours map[string]any
	if req.BusinessHours != nil && req.GetBusinessHours() != "" {
		if err := json.Unmarshal([]byte(req.GetBusinessHours()), &businessHours); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid business_hours JSON: %v", err)
		}
	}

	p, err := s.svc.UpdateSLAPolicy(ctx, id, tenantID, req.Name, firstResponseMins, resolutionMins, businessHours)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return slaPolicyToProto(p), nil
}

func (s *HelpdeskGRPCServer) ListSLAPolicies(ctx context.Context, req *helpdeskv1.ListSLAPoliciesRequest) (*helpdeskv1.ListSLAPoliciesResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing or invalid tenant_id")
	}
	policies, err := s.svc.ListSLAPolicies(ctx, tenantID)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	protoPolicies := make([]*helpdeskv1.SLAPolicy, len(policies))
	for i, p := range policies {
		protoPolicies[i] = slaPolicyToProto(p)
	}
	return &helpdeskv1.ListSLAPoliciesResponse{Policies: protoPolicies}, nil
}

func (s *HelpdeskGRPCServer) ApplySLAPolicy(ctx context.Context, req *helpdeskv1.ApplySLAPolicyRequest) (*helpdeskv1.Ticket, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	ticketID, err := uuid.Parse(req.GetTicketId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid ticket_id: %v", err)
	}
	policyID, err := uuid.Parse(req.GetSlaPolicyId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid sla_policy_id: %v", err)
	}
	t, err := s.svc.ApplySLAPolicy(ctx, ticketID, tenantID, policyID)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return ticketToProto(t), nil
}

func (s *HelpdeskGRPCServer) DeleteSLAPolicy(ctx context.Context, req *helpdeskv1.DeleteSLAPolicyRequest) (*emptypb.Empty, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	id, err := uuid.Parse(req.GetSlaPolicyId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid sla_policy_id: %v", err)
	}
	if err := s.svc.DeleteSLAPolicy(ctx, id, tenantID); err != nil {
		return nil, mapHelpdeskError(err)
	}
	slog.InfoContext(ctx, "helpdesk: sla policy deleted via grpc", "policy_id", id)
	return &emptypb.Empty{}, nil
}

func (s *HelpdeskGRPCServer) GetSLAStatus(ctx context.Context, req *helpdeskv1.GetSLAStatusRequest) (*helpdeskv1.GetSLAStatusResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	ticketID, err := uuid.Parse(req.GetTicketId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid ticket_id: %v", err)
	}

	var policyID *uuid.UUID
	if req.SlaPolicyId != nil {
		pid, err := uuid.Parse(req.GetSlaPolicyId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid sla_policy_id: %v", err)
		}
		policyID = &pid
	}

	slaStatus, err := s.svc.GetSLAStatus(ctx, ticketID, tenantID, policyID)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return &helpdeskv1.GetSLAStatusResponse{Status: string(slaStatus)}, nil
}

// ============================================================================
// Knowledge-Base Articles (4 RPCs)
// ============================================================================

func (s *HelpdeskGRPCServer) ListKBArticle(ctx context.Context, req *helpdeskv1.ListKBArticleRequest) (*helpdeskv1.ListKBArticleResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing or invalid tenant_id")
	}
	articles, err := s.svc.ListKBArticles(ctx, tenantID)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	protoArticles := make([]*helpdeskv1.KBArticle, len(articles))
	for i, a := range articles {
		protoArticles[i] = kbArticleToProto(a)
	}
	return &helpdeskv1.ListKBArticleResponse{Articles: protoArticles}, nil
}

func (s *HelpdeskGRPCServer) CreateKBArticle(ctx context.Context, req *helpdeskv1.CreateKBArticleRequest) (*helpdeskv1.KBArticle, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing or invalid tenant_id")
	}
	// Use a zero UUID as author placeholder when not provided via context
	authorID := uuid.Nil

	a, err := s.svc.CreateKBArticle(ctx, tenantID, authorID, req.GetTitle(), req.GetContent(), req.GetCategory(), req.GetStatus())
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	slog.InfoContext(ctx, "helpdesk: kb article created via grpc", "article_id", a.ID)
	return kbArticleToProto(a), nil
}

func (s *HelpdeskGRPCServer) UpdateKBArticle(ctx context.Context, req *helpdeskv1.UpdateKBArticleRequest) (*helpdeskv1.KBArticle, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	id, err := uuid.Parse(req.GetArticleId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid article_id: %v", err)
	}
	a, err := s.svc.UpdateKBArticle(ctx, id, tenantID, req.Title, req.Content, req.Category, req.Status)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return kbArticleToProto(a), nil
}

func (s *HelpdeskGRPCServer) DeleteKBArticle(ctx context.Context, req *helpdeskv1.DeleteKBArticleRequest) (*emptypb.Empty, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	id, err := uuid.Parse(req.GetArticleId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid article_id: %v", err)
	}
	if err := s.svc.DeleteKBArticle(ctx, id, tenantID); err != nil {
		return nil, mapHelpdeskError(err)
	}
	slog.InfoContext(ctx, "helpdesk: kb article deleted via grpc", "article_id", id)
	return &emptypb.Empty{}, nil
}

// ============================================================================
// Routing Rules (4 RPCs)
// ============================================================================

func (s *HelpdeskGRPCServer) ListRoutingRule(ctx context.Context, req *helpdeskv1.ListRoutingRuleRequest) (*helpdeskv1.ListRoutingRuleResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing or invalid tenant_id")
	}
	rules, err := s.svc.ListRoutingRules(ctx, tenantID)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	protoRules := make([]*helpdeskv1.RoutingRule, len(rules))
	for i, rr := range rules {
		protoRules[i] = routingRuleToProto(rr)
	}
	return &helpdeskv1.ListRoutingRuleResponse{Rules: protoRules}, nil
}

func (s *HelpdeskGRPCServer) CreateRoutingRule(ctx context.Context, req *helpdeskv1.CreateRoutingRuleRequest) (*helpdeskv1.RoutingRule, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing or invalid tenant_id")
	}

	var conditions map[string]any
	if c := req.GetConditions(); c != "" {
		if err := json.Unmarshal([]byte(c), &conditions); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid conditions JSON: %v", err)
		}
	}

	var targetQueueID *uuid.UUID
	if req.TargetQueueId != nil {
		id, err := uuid.Parse(req.GetTargetQueueId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid target_queue_id: %v", err)
		}
		targetQueueID = &id
	}

	rr, err := s.svc.CreateRoutingRule(ctx, tenantID, req.GetName(), conditions, targetQueueID, int(req.GetPriority()), req.GetEnabled())
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	slog.InfoContext(ctx, "helpdesk: routing rule created via grpc", "rule_id", rr.ID)
	return routingRuleToProto(rr), nil
}

func (s *HelpdeskGRPCServer) UpdateRoutingRule(ctx context.Context, req *helpdeskv1.UpdateRoutingRuleRequest) (*helpdeskv1.RoutingRule, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	id, err := uuid.Parse(req.GetRuleId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid rule_id: %v", err)
	}

	var conditions map[string]any
	if req.Conditions != nil && req.GetConditions() != "" {
		if err := json.Unmarshal([]byte(req.GetConditions()), &conditions); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid conditions JSON: %v", err)
		}
	}

	var targetQueueID *uuid.UUID
	if req.TargetQueueId != nil {
		qid, err := uuid.Parse(req.GetTargetQueueId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid target_queue_id: %v", err)
		}
		targetQueueID = &qid
	}

	var priority *int
	if req.Priority != nil {
		p := int(req.GetPriority())
		priority = &p
	}

	var enabled *bool
	if req.Enabled != nil {
		e := req.GetEnabled()
		enabled = &e
	}

	rr, err := s.svc.UpdateRoutingRule(ctx, id, tenantID, req.Name, conditions, targetQueueID, priority, enabled)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return routingRuleToProto(rr), nil
}

func (s *HelpdeskGRPCServer) DeleteRoutingRule(ctx context.Context, req *helpdeskv1.DeleteRoutingRuleRequest) (*emptypb.Empty, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	id, err := uuid.Parse(req.GetRuleId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid rule_id: %v", err)
	}
	if err := s.svc.DeleteRoutingRule(ctx, id, tenantID); err != nil {
		return nil, mapHelpdeskError(err)
	}
	slog.InfoContext(ctx, "helpdesk: routing rule deleted via grpc", "rule_id", id)
	return &emptypb.Empty{}, nil
}

// ============================================================================
// Business Hours (2 RPCs)
// ============================================================================

func (s *HelpdeskGRPCServer) GetBusinessHours(ctx context.Context, req *helpdeskv1.GetBusinessHoursRequest) (*helpdeskv1.BusinessHoursResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing or invalid tenant_id")
	}
	bh, err := s.svc.GetBusinessHours(ctx, tenantID)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return businessHoursToProto(bh), nil
}

func (s *HelpdeskGRPCServer) UpdateBusinessHours(ctx context.Context, req *helpdeskv1.UpdateBusinessHoursRequest) (*helpdeskv1.BusinessHoursResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing or invalid tenant_id")
	}

	var schedule map[string]helpdesk.DaySchedule
	if s := req.GetScheduleJson(); s != "" {
		if err := json.Unmarshal([]byte(s), &schedule); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid schedule_json: %v", err)
		}
	}
	if schedule == nil {
		schedule = map[string]helpdesk.DaySchedule{}
	}

	var holidays []helpdesk.HolidayEntry
	if h := req.GetHolidaysJson(); h != "" {
		if err := json.Unmarshal([]byte(h), &holidays); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid holidays_json: %v", err)
		}
	}
	if holidays == nil {
		holidays = []helpdesk.HolidayEntry{}
	}

	bh := &helpdesk.BusinessHours{
		TenantID: tenantID,
		Schedule: schedule,
		Holidays: holidays,
		Timezone: req.GetTimezone(),
	}
	updated, err := s.svc.UpsertBusinessHours(ctx, bh)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	slog.InfoContext(ctx, "helpdesk: business hours updated via grpc", "tenant_id", tenantID)
	return businessHoursToProto(updated), nil
}

// ============================================================================
// Stats (1 RPC)
// ============================================================================

func (s *HelpdeskGRPCServer) GetHelpdeskStats(ctx context.Context, req *helpdeskv1.GetHelpdeskStatsRequest) (*helpdeskv1.HelpdeskStats, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing or invalid tenant_id")
	}
	stats, err := s.svc.GetHelpdeskStats(ctx, tenantID)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	protoBreakdown := make([]*helpdeskv1.WeeklyDayCount, len(stats.WeeklyBreakdown))
	for i, d := range stats.WeeklyBreakdown {
		protoBreakdown[i] = &helpdeskv1.WeeklyDayCount{Label: d.Label, Count: int32(d.Count)}
	}
	return &helpdeskv1.HelpdeskStats{
		OpenTickets:          int32(stats.OpenTickets),
		AvgResponseTime:      stats.AvgResponseTime,
		ResolvedThisWeek:     int32(stats.ResolvedThisWeek),
		CustomerSatisfaction: stats.CustomerSatisfaction,
		WeeklyBreakdown:      protoBreakdown,
	}, nil
}

// ============================================================================
// Proto ↔ Domain conversion helpers
// ============================================================================

func ticketToProto(t *helpdesk.Ticket) *helpdeskv1.Ticket {
	msg := &helpdeskv1.Ticket{
		Id:          t.ID.String(),
		TenantId:    t.TenantID.String(),
		Subject:     t.Subject,
		Status:      t.Status,
		Priority:    t.Priority,
		RequesterId: t.RequesterID.String(),
		CreatedAt:   timestamppb.New(t.CreatedAt),
		UpdatedAt:   timestamppb.New(t.UpdatedAt),
	}
	if t.AssigneeID != nil {
		s := t.AssigneeID.String()
		msg.AssigneeId = &s
	}
	if t.QueueID != nil {
		s := t.QueueID.String()
		msg.QueueId = &s
	}
	if t.DueAt != nil {
		msg.DueAt = timestamppb.New(*t.DueAt)
	}
	if t.MergedIntoID != nil {
		s := t.MergedIntoID.String()
		msg.MergedIntoId = &s
	}
	if t.FirstResponseAt != nil {
		msg.FirstResponseAt = timestamppb.New(*t.FirstResponseAt)
	}
	if t.ResolvedAt != nil {
		msg.ResolvedAt = timestamppb.New(*t.ResolvedAt)
	}
	msg.Description = t.Description
	msg.Category = t.Category
	msg.TicketNumber = int32(t.TicketNumber)
	msg.RequesterName = t.RequesterName
	if t.AssigneeName != nil {
		msg.AssigneeName = t.AssigneeName
	}
	if t.ContactID != nil {
		s := t.ContactID.String()
		msg.ContactId = &s
	}
	if t.OrgID != nil {
		s := t.OrgID.String()
		msg.OrgId = &s
	}
	if t.SourceChannel != nil {
		msg.SourceChannel = t.SourceChannel
	}
	if t.SourceMessageID != nil {
		s := t.SourceMessageID.String()
		msg.SourceMessageId = &s
	}
	if t.CsatRating != nil {
		v := int32(*t.CsatRating)
		msg.CsatRating = &v
	}
	if t.CsatComment != nil {
		msg.CsatComment = t.CsatComment
	}
	return msg
}

func ticketMessageToProto(m *helpdesk.TicketMessage) *helpdeskv1.TicketMessage {
	return &helpdeskv1.TicketMessage{
		Id:          m.ID.String(),
		TicketId:    m.TicketID.String(),
		AuthorId:    m.AuthorID.String(),
		Body:        m.Body,
		Internal:    m.Internal,
		Attachments: m.Attachments,
		CreatedAt:   timestamppb.New(m.CreatedAt),
	}
}

func ticketQueueToProto(q *helpdesk.TicketQueue) *helpdeskv1.TicketQueue {
	msg := &helpdeskv1.TicketQueue{
		Id:        q.ID.String(),
		TenantId:  q.TenantID.String(),
		Name:      q.Name,
		CreatedAt: timestamppb.New(q.CreatedAt),
		UpdatedAt: timestamppb.New(q.UpdatedAt),
	}
	if q.DefaultAssigneeID != nil {
		s := q.DefaultAssigneeID.String()
		msg.DefaultAssigneeId = &s
	}
	if q.SLAPolicyID != nil {
		s := q.SLAPolicyID.String()
		msg.SlaPolicyId = &s
	}
	return msg
}

func cannedResponseToProto(cr *helpdesk.CannedResponse) *helpdeskv1.CannedResponse {
	return &helpdeskv1.CannedResponse{
		Id:        cr.ID.String(),
		TenantId:  cr.TenantID.String(),
		Name:      cr.Name,
		Body:      cr.Body,
		CreatedAt: timestamppb.New(cr.CreatedAt),
		UpdatedAt: timestamppb.New(cr.UpdatedAt),
	}
}

func slaPolicyToProto(p *helpdesk.SLAPolicy) *helpdeskv1.SLAPolicy {
	msg := &helpdeskv1.SLAPolicy{
		Id:                p.ID.String(),
		TenantId:          p.TenantID.String(),
		Name:              p.Name,
		FirstResponseMins: int32(p.FirstResponseMins),
		ResolutionMins:    int32(p.ResolutionMins),
		CreatedAt:         timestamppb.New(p.CreatedAt),
		UpdatedAt:         timestamppb.New(p.UpdatedAt),
	}
	if len(p.BusinessHours) > 0 {
		if raw, err := json.Marshal(p.BusinessHours); err == nil {
			s := string(raw)
			msg.BusinessHours = &s
		}
	}
	return msg
}

func kbArticleToProto(a *helpdesk.KBArticle) *helpdeskv1.KBArticle {
	return &helpdeskv1.KBArticle{
		Id:        a.ID.String(),
		TenantId:  a.TenantID.String(),
		Title:     a.Title,
		Content:   a.Content,
		Category:  a.Category,
		Status:    a.Status,
		AuthorId:  a.AuthorID.String(),
		CreatedAt: timestamppb.New(a.CreatedAt),
		UpdatedAt: timestamppb.New(a.UpdatedAt),
	}
}

func routingRuleToProto(rr *helpdesk.RoutingRule) *helpdeskv1.RoutingRule {
	msg := &helpdeskv1.RoutingRule{
		Id:        rr.ID.String(),
		TenantId:  rr.TenantID.String(),
		Name:      rr.Name,
		Priority:  int32(rr.Priority),
		Enabled:   rr.Enabled,
		CreatedAt: timestamppb.New(rr.CreatedAt),
		UpdatedAt: timestamppb.New(rr.UpdatedAt),
	}
	if condJSON, err := json.Marshal(rr.Conditions); err == nil {
		msg.Conditions = string(condJSON)
	}
	if rr.TargetQueueID != nil {
		s := rr.TargetQueueID.String()
		msg.TargetQueueId = &s
	}
	return msg
}

func businessHoursToProto(bh *helpdesk.BusinessHours) *helpdeskv1.BusinessHoursResponse {
	msg := &helpdeskv1.BusinessHoursResponse{
		TenantId: bh.TenantID.String(),
		Timezone: bh.Timezone,
	}
	if !bh.UpdatedAt.IsZero() {
		msg.UpdatedAt = timestamppb.New(bh.UpdatedAt)
	}
	if raw, err := json.Marshal(bh.Schedule); err == nil {
		msg.ScheduleJson = string(raw)
	}
	if raw, err := json.Marshal(bh.Holidays); err == nil {
		msg.HolidaysJson = string(raw)
	}
	return msg
}

// ============================================================================
// Error mapping
// ============================================================================

func mapHelpdeskError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, helpdesk.ErrTicketNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, helpdesk.ErrQueueNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, helpdesk.ErrSLAPolicyNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, helpdesk.ErrCannedResponseNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, helpdesk.ErrInvalidStatus):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, helpdesk.ErrInvalidPriority):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, helpdesk.ErrCannotMergeSelf):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, helpdesk.ErrAlreadyMerged):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, helpdesk.ErrKBArticleNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, helpdesk.ErrRoutingRuleNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, helpdesk.ErrContactNotFound):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, helpdesk.ErrOrgNotFound):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, helpdesk.ErrInvalidSourceChannel):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, helpdesk.ErrCsatSurveyNotFound):
		// Unknown, malformed, expired, revoked and already-redeemed links all
		// land here on purpose: one verdict, no existence oracle.
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, helpdesk.ErrCsatCommentTooLong):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, helpdesk.ErrInvalidCsatRating):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, helpdesk.ErrMessageAlreadyLinked):
		// Only reachable if the post-conflict re-fetch in
		// Service.CreateTicketFromMessage itself fails after losing the race
		// (the happy path resolves this into an existing-ticket return, not
		// an error) -- surfaced as Conflict rather than Internal.
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		slog.Error("unhandled helpdesk service error", "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}
