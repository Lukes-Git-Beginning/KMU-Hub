package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/helpdesk"
	helpdeskv1 "github.com/kmuhub/kmuhub/proto/helpdesk/v1"
)

// HelpdeskGRPCServer implements the HelpdeskService gRPC server.
type HelpdeskGRPCServer struct {
	helpdeskv1.UnimplementedHelpdeskServiceServer
	svc *helpdesk.Service
}

// NewHelpdeskGRPCServer creates a new Helpdesk gRPC server.
func NewHelpdeskGRPCServer(svc *helpdesk.Service) *HelpdeskGRPCServer {
	return &HelpdeskGRPCServer{svc: svc}
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

	t, err := s.svc.CreateTicket(ctx, tenantID, requesterID, req.GetSubject(), req.GetPriority(), assigneeID, queueID)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return ticketToProto(t), nil
}

func (s *HelpdeskGRPCServer) GetTicket(ctx context.Context, req *helpdeskv1.GetTicketRequest) (*helpdeskv1.Ticket, error) {
	id, err := uuid.Parse(req.GetTicketId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid ticket_id: %v", err)
	}
	t, err := s.svc.GetTicket(ctx, id)
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

	page := max(int(req.GetPage()), 1)
	pageSize := int(req.GetPageSize())
	if pageSize < 1 {
		pageSize = 20
	}

	tickets, total, err := s.svc.ListTickets(ctx, tenantID, statusFilter, page, pageSize)
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

	t, err := s.svc.UpdateTicket(ctx, id, req.Subject, req.Priority, assigneeID, queueID)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return ticketToProto(t), nil
}

func (s *HelpdeskGRPCServer) CloseTicket(ctx context.Context, req *helpdeskv1.CloseTicketRequest) (*helpdeskv1.Ticket, error) {
	id, err := uuid.Parse(req.GetTicketId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid ticket_id: %v", err)
	}
	t, err := s.svc.CloseTicket(ctx, id)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return ticketToProto(t), nil
}

func (s *HelpdeskGRPCServer) ReopenTicket(ctx context.Context, req *helpdeskv1.ReopenTicketRequest) (*helpdeskv1.Ticket, error) {
	id, err := uuid.Parse(req.GetTicketId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid ticket_id: %v", err)
	}
	t, err := s.svc.ReopenTicket(ctx, id)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return ticketToProto(t), nil
}

func (s *HelpdeskGRPCServer) AssignTicket(ctx context.Context, req *helpdeskv1.AssignTicketRequest) (*helpdeskv1.Ticket, error) {
	ticketID, err := uuid.Parse(req.GetTicketId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid ticket_id: %v", err)
	}
	assigneeID, err := uuid.Parse(req.GetAssigneeId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid assignee_id: %v", err)
	}
	t, err := s.svc.AssignTicket(ctx, ticketID, assigneeID)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return ticketToProto(t), nil
}

func (s *HelpdeskGRPCServer) MergeTickets(ctx context.Context, req *helpdeskv1.MergeTicketsRequest) (*emptypb.Empty, error) {
	sourceID, err := uuid.Parse(req.GetSourceTicketId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid source_ticket_id: %v", err)
	}
	targetID, err := uuid.Parse(req.GetTargetTicketId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid target_ticket_id: %v", err)
	}
	if err := s.svc.MergeTickets(ctx, sourceID, targetID); err != nil {
		return nil, mapHelpdeskError(err)
	}
	return &emptypb.Empty{}, nil
}

// ============================================================================
// Messages (2 RPCs)
// ============================================================================

func (s *HelpdeskGRPCServer) AddMessage(ctx context.Context, req *helpdeskv1.AddMessageRequest) (*helpdeskv1.TicketMessage, error) {
	ticketID, err := uuid.Parse(req.GetTicketId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid ticket_id: %v", err)
	}
	authorID, err := uuid.Parse(req.GetAuthorId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid author_id: %v", err)
	}

	m, err := s.svc.AddMessage(ctx, ticketID, authorID, req.GetBody(), req.GetInternal(), req.GetAttachments())
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return ticketMessageToProto(m), nil
}

func (s *HelpdeskGRPCServer) ListMessages(ctx context.Context, req *helpdeskv1.ListMessagesRequest) (*helpdeskv1.ListMessagesResponse, error) {
	ticketID, err := uuid.Parse(req.GetTicketId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid ticket_id: %v", err)
	}
	msgs, err := s.svc.ListMessages(ctx, ticketID)
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

	q, err := s.svc.UpdateQueue(ctx, id, req.Name, defaultAssigneeID, slaPolicyID)
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
	id, err := uuid.Parse(req.GetQueueId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid queue_id: %v", err)
	}
	if err := s.svc.DeleteQueue(ctx, id); err != nil {
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
	id, err := uuid.Parse(req.GetCannedResponseId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canned_response_id: %v", err)
	}
	cr, err := s.svc.UpdateCannedResponse(ctx, id, req.Name, req.Body)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return cannedResponseToProto(cr), nil
}

func (s *HelpdeskGRPCServer) DeleteCannedResponse(ctx context.Context, req *helpdeskv1.DeleteCannedResponseRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(req.GetCannedResponseId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canned_response_id: %v", err)
	}
	if err := s.svc.DeleteCannedResponse(ctx, id); err != nil {
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

	p, err := s.svc.UpdateSLAPolicy(ctx, id, req.Name, firstResponseMins, resolutionMins, businessHours)
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
	ticketID, err := uuid.Parse(req.GetTicketId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid ticket_id: %v", err)
	}
	policyID, err := uuid.Parse(req.GetSlaPolicyId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid sla_policy_id: %v", err)
	}
	t, err := s.svc.ApplySLAPolicy(ctx, ticketID, policyID)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return ticketToProto(t), nil
}

func (s *HelpdeskGRPCServer) GetSLAStatus(ctx context.Context, req *helpdeskv1.GetSLAStatusRequest) (*helpdeskv1.GetSLAStatusResponse, error) {
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

	slaStatus, err := s.svc.GetSLAStatus(ctx, ticketID, policyID)
	if err != nil {
		return nil, mapHelpdeskError(err)
	}
	return &helpdeskv1.GetSLAStatusResponse{Status: string(slaStatus)}, nil
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
	default:
		slog.Error("unhandled helpdesk service error", "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}
