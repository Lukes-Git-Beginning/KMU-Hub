package server

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/dialer"
	"github.com/kmuhub/kmuhub/internal/middleware"
	dialerv1 "github.com/kmuhub/kmuhub/proto/dialer/v1"
)

// DialerGRPCServer implements the DialerService gRPC server.
type DialerGRPCServer struct {
	dialerv1.UnimplementedDialerServiceServer
	svc *dialer.Service
}

// NewDialerGRPCServer creates a new Dialer gRPC server.
func NewDialerGRPCServer(svc *dialer.Service) *DialerGRPCServer {
	return &DialerGRPCServer{svc: svc}
}

// ============================================================================
// Campaign Management (7 RPCs)
// ============================================================================

func (s *DialerGRPCServer) CreateCampaign(ctx context.Context, req *dialerv1.CreateCampaignRequest) (*dialerv1.Campaign, error) {
	// TenantId comes from proto field (gateway passes it) — fall back to context
	var tenantID uuid.UUID
	if tid := req.GetTenantId(); tid != "" {
		parsed, err := uuid.Parse(tid)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
		}
		tenantID = parsed
	} else {
		ctxTenant, err := middleware.GetTenantID(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
		}
		tenantID = ctxTenant
	}

	var assignedAgentIDs []uuid.UUID
	for _, idStr := range req.GetAssignedAgentIds() {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid agent_id %q: %v", idStr, err)
		}
		assignedAgentIDs = append(assignedAgentIDs, id)
	}

	var settings *dialer.CampaignSettings
	if raw := req.GetSettings(); raw != "" {
		var cs dialer.CampaignSettings
		if err := json.Unmarshal([]byte(raw), &cs); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid settings JSON: %v", err)
		}
		settings = &cs
	}

	// createdBy is not in proto v1 — use zero UUID; will be replaced by server-side
	// user extraction once proto v2 adds the field.
	var createdBy uuid.UUID

	c, err := s.svc.CreateCampaign(ctx, tenantID, createdBy, req.GetName(), req.Description, campaignModeFromProto(req.GetMode()), assignedAgentIDs, settings)
	if err != nil {
		return nil, mapDialerError(err)
	}
	return campaignToProto(c), nil
}

func (s *DialerGRPCServer) GetCampaign(ctx context.Context, req *dialerv1.GetCampaignRequest) (*dialerv1.Campaign, error) {
	id, err := uuid.Parse(req.GetCampaignId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid campaign_id: %v", err)
	}
	// GetCampaignRequest has no tenant_id field in proto v1 — read from JWT context
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}
	c, err := s.svc.GetCampaignForTenant(ctx, id, tenantID)
	if err != nil {
		return nil, mapDialerError(err)
	}
	return campaignToProto(c), nil
}

func (s *DialerGRPCServer) ListCampaigns(ctx context.Context, req *dialerv1.ListCampaignsRequest) (*dialerv1.ListCampaignsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing or invalid tenant_id")
	}

	var statusFilter *string
	if req.StatusFilter != nil && req.GetStatusFilter() != dialerv1.CampaignStatus_CAMPAIGN_STATUS_UNSPECIFIED {
		sf := campaignStatusFromProto(req.GetStatusFilter())
		statusFilter = &sf
	}

	page := int(req.GetPage())
	if page < 1 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
	if pageSize < 1 {
		pageSize = 20
	}

	campaigns, total, err := s.svc.ListCampaigns(ctx, tenantID, statusFilter, page, pageSize)
	if err != nil {
		return nil, mapDialerError(err)
	}

	protoCampaigns := make([]*dialerv1.Campaign, len(campaigns))
	for i, c := range campaigns {
		protoCampaigns[i] = campaignToProto(c)
	}
	return &dialerv1.ListCampaignsResponse{
		Campaigns: protoCampaigns,
		Total:     int32(total),
	}, nil
}

func (s *DialerGRPCServer) UpdateCampaign(ctx context.Context, req *dialerv1.UpdateCampaignRequest) (*dialerv1.Campaign, error) {
	id, err := uuid.Parse(req.GetCampaignId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid campaign_id: %v", err)
	}

	var assignedAgentIDs []uuid.UUID
	for _, idStr := range req.GetAssignedAgentIds() {
		aid, err := uuid.Parse(idStr)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid agent_id %q: %v", idStr, err)
		}
		assignedAgentIDs = append(assignedAgentIDs, aid)
	}

	var settings *dialer.CampaignSettings
	if raw := req.GetSettings(); raw != "" {
		var cs dialer.CampaignSettings
		if err := json.Unmarshal([]byte(raw), &cs); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid settings JSON: %v", err)
		}
		settings = &cs
	}

	c, err := s.svc.UpdateCampaign(ctx, id, req.Name, req.Description, settings, assignedAgentIDs)
	if err != nil {
		return nil, mapDialerError(err)
	}
	return campaignToProto(c), nil
}

func (s *DialerGRPCServer) StartCampaign(ctx context.Context, req *dialerv1.StartCampaignRequest) (*dialerv1.Campaign, error) {
	id, err := uuid.Parse(req.GetCampaignId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid campaign_id: %v", err)
	}
	c, err := s.svc.StartCampaign(ctx, id)
	if err != nil {
		return nil, mapDialerError(err)
	}
	return campaignToProto(c), nil
}

func (s *DialerGRPCServer) PauseCampaign(ctx context.Context, req *dialerv1.PauseCampaignRequest) (*dialerv1.Campaign, error) {
	id, err := uuid.Parse(req.GetCampaignId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid campaign_id: %v", err)
	}
	c, err := s.svc.PauseCampaign(ctx, id)
	if err != nil {
		return nil, mapDialerError(err)
	}
	return campaignToProto(c), nil
}

func (s *DialerGRPCServer) ArchiveCampaign(ctx context.Context, req *dialerv1.ArchiveCampaignRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(req.GetCampaignId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid campaign_id: %v", err)
	}
	if err := s.svc.ArchiveCampaign(ctx, id); err != nil {
		return nil, mapDialerError(err)
	}
	return &emptypb.Empty{}, nil
}

// ============================================================================
// Campaign Contacts / Queue (5 RPCs)
// ============================================================================

func (s *DialerGRPCServer) AddContactsToCampaign(ctx context.Context, req *dialerv1.AddContactsToCampaignRequest) (*dialerv1.AddContactsToCampaignResponse, error) {
	campaignID, err := uuid.Parse(req.GetCampaignId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid campaign_id: %v", err)
	}

	var contactIDs []uuid.UUID
	for _, idStr := range req.GetContactIds() {
		cid, err := uuid.Parse(idStr)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid contact_id %q: %v", idStr, err)
		}
		contactIDs = append(contactIDs, cid)
	}

	var filterID *uuid.UUID
	if req.FilterId != nil {
		fid, err := uuid.Parse(req.GetFilterId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid filter_id: %v", err)
		}
		filterID = &fid
	}

	added, skipped, err := s.svc.AddContactsToCampaign(ctx, campaignID, contactIDs, filterID)
	if err != nil {
		return nil, mapDialerError(err)
	}
	return &dialerv1.AddContactsToCampaignResponse{
		AddedCount:   int32(added),
		SkippedCount: int32(skipped),
	}, nil
}

func (s *DialerGRPCServer) GetNextContact(ctx context.Context, req *dialerv1.GetNextContactRequest) (*dialerv1.CampaignContact, error) {
	campaignID, err := uuid.Parse(req.GetCampaignId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid campaign_id: %v", err)
	}
	cc, err := s.svc.GetNextContact(ctx, campaignID)
	if err != nil {
		return nil, mapDialerError(err)
	}
	return campaignContactToProto(cc), nil
}

func (s *DialerGRPCServer) SkipContact(ctx context.Context, req *dialerv1.SkipContactRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(req.GetCampaignContactId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid campaign_contact_id: %v", err)
	}
	if err := s.svc.SkipContact(ctx, id); err != nil {
		return nil, mapDialerError(err)
	}
	return &emptypb.Empty{}, nil
}

func (s *DialerGRPCServer) RequeueContact(ctx context.Context, req *dialerv1.RequeueContactRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(req.GetCampaignContactId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid campaign_contact_id: %v", err)
	}
	if err := s.svc.RequeueContact(ctx, id); err != nil {
		return nil, mapDialerError(err)
	}
	return &emptypb.Empty{}, nil
}

func (s *DialerGRPCServer) ListCampaignContacts(ctx context.Context, req *dialerv1.ListCampaignContactsRequest) (*dialerv1.ListCampaignContactsResponse, error) {
	campaignID, err := uuid.Parse(req.GetCampaignId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid campaign_id: %v", err)
	}

	var statusFilter *string
	if req.StatusFilter != nil && req.GetStatusFilter() != dialerv1.CampaignContactStatus_CAMPAIGN_CONTACT_STATUS_UNSPECIFIED {
		sf := contactStatusFromProto(req.GetStatusFilter())
		statusFilter = &sf
	}

	page := int(req.GetPage())
	if page < 1 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
	if pageSize < 1 {
		pageSize = 50
	}

	contacts, total, err := s.svc.ListCampaignContacts(ctx, campaignID, statusFilter, page, pageSize)
	if err != nil {
		return nil, mapDialerError(err)
	}

	protoContacts := make([]*dialerv1.CampaignContact, len(contacts))
	for i, cc := range contacts {
		protoContacts[i] = campaignContactToProto(cc)
	}
	return &dialerv1.ListCampaignContactsResponse{
		Contacts: protoContacts,
		Total:    int32(total),
	}, nil
}

// ============================================================================
// Call Sessions (4 RPCs)
// ============================================================================

func (s *DialerGRPCServer) InitiateDialerCall(ctx context.Context, req *dialerv1.InitiateDialerCallRequest) (*dialerv1.DialerCallSession, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing tenant_id: %v", err)
	}

	ccID, err := uuid.Parse(req.GetCampaignContactId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid campaign_contact_id: %v", err)
	}
	agentID, err := uuid.Parse(req.GetAgentId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid agent_id: %v", err)
	}

	var videoCallSessionID *uuid.UUID
	if req.CallSessionId != nil {
		vid, err := uuid.Parse(req.GetCallSessionId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid call_session_id: %v", err)
		}
		videoCallSessionID = &vid
	}

	cs, err := s.svc.InitiateDialerCall(ctx, tenantID, ccID, agentID, videoCallSessionID)
	if err != nil {
		return nil, mapDialerError(err)
	}
	return dialerCallSessionToProto(cs), nil
}

func (s *DialerGRPCServer) LogCallOutcome(ctx context.Context, req *dialerv1.LogCallOutcomeRequest) (*dialerv1.DialerCallSession, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing tenant_id: %v", err)
	}

	sessionID, err := uuid.Parse(req.GetDialerCallSessionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid dialer_call_session_id: %v", err)
	}
	outcomeID, err := uuid.Parse(req.GetOutcomeId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid outcome_id: %v", err)
	}

	var callbackAt *time.Time
	if req.CallbackAt != nil {
		t := req.GetCallbackAt().AsTime()
		callbackAt = &t
	}

	var appointmentID *uuid.UUID
	if req.AppointmentId != nil {
		aid, err := uuid.Parse(req.GetAppointmentId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid appointment_id: %v", err)
		}
		appointmentID = &aid
	}

	cs, err := s.svc.LogCallOutcome(ctx, tenantID, sessionID, outcomeID, req.Notes, req.NextAction, callbackAt, appointmentID)
	if err != nil {
		return nil, mapDialerError(err)
	}
	return dialerCallSessionToProto(cs), nil
}

func (s *DialerGRPCServer) SaveCallNotes(ctx context.Context, req *dialerv1.SaveCallNotesRequest) (*emptypb.Empty, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing tenant_id: %v", err)
	}

	sessionID, err := uuid.Parse(req.GetDialerCallSessionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid dialer_call_session_id: %v", err)
	}
	if err := s.svc.SaveCallNotes(ctx, tenantID, sessionID, req.GetNotes()); err != nil {
		return nil, mapDialerError(err)
	}
	return &emptypb.Empty{}, nil
}

func (s *DialerGRPCServer) CompleteWrapUp(ctx context.Context, req *dialerv1.CompleteWrapUpRequest) (*emptypb.Empty, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing tenant_id: %v", err)
	}

	sessionID, err := uuid.Parse(req.GetDialerCallSessionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid dialer_call_session_id: %v", err)
	}
	if err := s.svc.CompleteWrapUp(ctx, tenantID, sessionID); err != nil {
		return nil, mapDialerError(err)
	}
	return &emptypb.Empty{}, nil
}

// ============================================================================
// Agent Status (3 RPCs)
// ============================================================================

func (s *DialerGRPCServer) SetAgentStatus(ctx context.Context, req *dialerv1.SetAgentStatusRequest) (*dialerv1.AgentStatus, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing or invalid tenant_id")
	}

	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}

	agentStatus := agentStatusFromProto(req.GetStatus())

	var campaignID *uuid.UUID
	if req.CampaignId != nil {
		cid, err := uuid.Parse(req.GetCampaignId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid campaign_id: %v", err)
		}
		campaignID = &cid
	}

	entry, err := s.svc.SetAgentStatus(ctx, userID, agentStatus, campaignID, tenantID)
	if err != nil {
		return nil, mapDialerError(err)
	}
	return agentStatusToProto(entry), nil
}

func (s *DialerGRPCServer) GetAgentStatus(ctx context.Context, req *dialerv1.GetAgentStatusRequest) (*dialerv1.AgentStatus, error) {
	userID, err := uuid.Parse(req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}
	entry, err := s.svc.GetAgentStatus(ctx, userID)
	if err != nil {
		return nil, mapDialerError(err)
	}
	return agentStatusToProto(entry), nil
}

func (s *DialerGRPCServer) GetCampaignAgents(ctx context.Context, req *dialerv1.GetCampaignAgentsRequest) (*dialerv1.GetCampaignAgentsResponse, error) {
	campaignID, err := uuid.Parse(req.GetCampaignId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid campaign_id: %v", err)
	}
	entries, err := s.svc.GetCampaignAgents(ctx, campaignID)
	if err != nil {
		return nil, mapDialerError(err)
	}
	agents := make([]*dialerv1.AgentStatus, len(entries))
	for i, e := range entries {
		agents[i] = agentStatusToProto(&e)
	}
	return &dialerv1.GetCampaignAgentsResponse{Agents: agents}, nil
}

// ============================================================================
// Call Outcomes (4 RPCs)
// ============================================================================

func (s *DialerGRPCServer) ListCallOutcomes(ctx context.Context, req *dialerv1.ListCallOutcomesRequest) (*dialerv1.ListCallOutcomesResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing or invalid tenant_id")
	}

	includeInactive := false
	if req.IncludeInactive != nil {
		includeInactive = req.GetIncludeInactive()
	}

	outcomes, err := s.svc.ListCallOutcomes(ctx, tenantID, includeInactive)
	if err != nil {
		return nil, mapDialerError(err)
	}
	protoOutcomes := make([]*dialerv1.CallOutcome, len(outcomes))
	for i, o := range outcomes {
		protoOutcomes[i] = callOutcomeToProto(o)
	}
	return &dialerv1.ListCallOutcomesResponse{Outcomes: protoOutcomes}, nil
}

func (s *DialerGRPCServer) CreateCallOutcome(ctx context.Context, req *dialerv1.CreateCallOutcomeRequest) (*dialerv1.CallOutcome, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing or invalid tenant_id")
	}

	o, err := s.svc.CreateCallOutcome(ctx, tenantID,
		req.GetLabel(),
		req.GetColor(),
		req.GetIsPositive(),
		req.GetIsCallback(),
		req.GetIsAppointment(),
		int(req.GetSortOrder()),
	)
	if err != nil {
		return nil, mapDialerError(err)
	}
	return callOutcomeToProto(o), nil
}

func (s *DialerGRPCServer) UpdateCallOutcome(ctx context.Context, req *dialerv1.UpdateCallOutcomeRequest) (*dialerv1.CallOutcome, error) {
	outcomeID, err := uuid.Parse(req.GetOutcomeId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid outcome_id: %v", err)
	}

	var sortOrder *int
	if req.SortOrder != nil {
		so := int(req.GetSortOrder())
		sortOrder = &so
	}

	o, err := s.svc.UpdateCallOutcome(ctx, outcomeID, req.Label, req.Color, req.IsPositive, req.IsCallback, req.IsAppointment, sortOrder, req.IsActive)
	if err != nil {
		return nil, mapDialerError(err)
	}
	return callOutcomeToProto(o), nil
}

func (s *DialerGRPCServer) DeleteCallOutcome(ctx context.Context, req *dialerv1.DeleteCallOutcomeRequest) (*emptypb.Empty, error) {
	outcomeID, err := uuid.Parse(req.GetOutcomeId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid outcome_id: %v", err)
	}
	if err := s.svc.DeleteCallOutcome(ctx, outcomeID); err != nil {
		return nil, mapDialerError(err)
	}
	return &emptypb.Empty{}, nil
}

// ============================================================================
// Dashboard (2 RPCs)
// ============================================================================

func (s *DialerGRPCServer) GetCampaignDashboard(ctx context.Context, req *dialerv1.GetCampaignDashboardRequest) (*dialerv1.CampaignDashboard, error) {
	campaignID, err := uuid.Parse(req.GetCampaignId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid campaign_id: %v", err)
	}
	stats, err := s.svc.GetCampaignDashboard(ctx, campaignID)
	if err != nil {
		return nil, mapDialerError(err)
	}
	return campaignStatsToProto(campaignID, stats), nil
}

func (s *DialerGRPCServer) GetAgentDashboard(ctx context.Context, req *dialerv1.GetAgentDashboardRequest) (*dialerv1.AgentDashboard, error) {
	agentID, err := uuid.Parse(req.GetAgentId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid agent_id: %v", err)
	}
	stats, err := s.svc.GetAgentDashboard(ctx, agentID)
	if err != nil {
		return nil, mapDialerError(err)
	}
	return agentStatsToProto(agentID, stats), nil
}

// ============================================================================
// Proto ↔ Domain conversion helpers
// ============================================================================

func campaignToProto(c *dialer.Campaign) *dialerv1.Campaign {
	agentIDs := make([]string, len(c.AssignedAgentIDs))
	for i, id := range c.AssignedAgentIDs {
		agentIDs[i] = id.String()
	}

	settingsJSON, _ := json.Marshal(c.Settings)
	settingsStr := string(settingsJSON)

	msg := &dialerv1.Campaign{
		Id:               c.ID.String(),
		TenantId:         c.TenantID.String(),
		Name:             c.Name,
		Description:      c.Description,
		Status:           campaignStatusToProto(c.Status),
		Mode:             campaignModeToProto(c.Mode),
		Settings:         &settingsStr,
		CreatedBy:        c.CreatedBy.String(),
		AssignedAgentIds: agentIDs,
		ContactCount:     int32(c.ContactCount),
		CompletedCount:   int32(c.CompletedCount),
		CreatedAt:        timestamppb.New(c.CreatedAt),
		UpdatedAt:        timestamppb.New(c.UpdatedAt),
	}
	if c.StartedAt != nil {
		msg.StartedAt = timestamppb.New(*c.StartedAt)
	}
	if c.CompletedAt != nil {
		msg.CompletedAt = timestamppb.New(*c.CompletedAt)
	}
	return msg
}

func campaignContactToProto(cc *dialer.CampaignContact) *dialerv1.CampaignContact {
	msg := &dialerv1.CampaignContact{
		Id:             cc.ID.String(),
		CampaignId:     cc.CampaignID.String(),
		ContactId:      cc.ContactID.String(),
		Position:       int32(cc.Position),
		Status:         contactStatusToProto(cc.Status),
		Notes:          cc.Notes,
		CallCount:      int32(cc.CallCount),
		ContactName:    cc.ContactName,
		ContactPhone:   cc.ContactPhone,
	}
	if cc.OutcomeID != nil {
		s := cc.OutcomeID.String()
		msg.OutcomeId = &s
	}
	if cc.ContactCompany != "" {
		msg.ContactCompany = &cc.ContactCompany
	}
	if cc.CallbackAt != nil {
		msg.CallbackAt = timestamppb.New(*cc.CallbackAt)
	}
	if cc.LastCalledAt != nil {
		msg.LastCalledAt = timestamppb.New(*cc.LastCalledAt)
	}
	return msg
}

func dialerCallSessionToProto(cs *dialer.CallSession) *dialerv1.DialerCallSession {
	msg := &dialerv1.DialerCallSession{
		Id:                cs.ID.String(),
		CampaignContactId: cs.CampaignContactID.String(),
		AgentId:           cs.AgentID.String(),
		Notes:             cs.Notes,
		NextAction:        cs.NextAction,
		CreatedAt:         timestamppb.New(cs.CreatedAt),
	}
	if cs.CallSessionID != nil {
		s := cs.CallSessionID.String()
		msg.CallSessionId = &s
	}
	if cs.OutcomeID != nil {
		s := cs.OutcomeID.String()
		msg.OutcomeId = &s
	}
	if cs.DurationSeconds != nil {
		d := int32(*cs.DurationSeconds)
		msg.DurationSeconds = &d
	}
	if cs.AppointmentID != nil {
		s := cs.AppointmentID.String()
		msg.AppointmentId = &s
	}
	if cs.WrapUpCompletedAt != nil {
		msg.WrapUpCompletedAt = timestamppb.New(*cs.WrapUpCompletedAt)
	}
	return msg
}

func agentStatusToProto(a *dialer.AgentStatusEntry) *dialerv1.AgentStatus {
	msg := &dialerv1.AgentStatus{
		UserId:    a.UserID.String(),
		Status:    agentStatusEnumToProto(a.Status),
		Since:     timestamppb.New(a.Since),
		FirstName: a.FirstName,
		LastName:  a.LastName,
	}
	if a.CampaignID != nil {
		s := a.CampaignID.String()
		msg.CampaignId = &s
	}
	return msg
}

func callOutcomeToProto(o *dialer.CallOutcome) *dialerv1.CallOutcome {
	return &dialerv1.CallOutcome{
		Id:            o.ID.String(),
		TenantId:      o.TenantID.String(),
		Label:         o.Label,
		Color:         o.Color,
		IsPositive:    o.IsPositive,
		IsCallback:    o.IsCallback,
		IsAppointment: o.IsAppointment,
		SortOrder:     int32(o.SortOrder),
		IsActive:      o.IsActive,
	}
}

func campaignStatsToProto(campaignID uuid.UUID, s *dialer.CampaignStats) *dialerv1.CampaignDashboard {
	breakdown := make([]*dialerv1.OutcomeCount, len(s.OutcomeBreakdown))
	for i, oc := range s.OutcomeBreakdown {
		breakdown[i] = &dialerv1.OutcomeCount{
			OutcomeId:    oc.OutcomeID.String(),
			OutcomeLabel: oc.OutcomeLabel,
			OutcomeColor: oc.OutcomeColor,
			Count:        int32(oc.Count),
		}
	}
	return &dialerv1.CampaignDashboard{
		CampaignId:              campaignID.String(),
		TotalContacts:           int32(s.TotalContacts),
		CompletedContacts:       int32(s.CompletedContacts),
		PendingContacts:         int32(s.PendingContacts),
		SkippedContacts:         int32(s.SkippedContacts),
		CallbackContacts:        int32(s.CallbackContacts),
		TotalCalls:              int32(s.TotalCalls),
		PositiveCalls:           int32(s.PositiveCalls),
		AverageDurationSeconds:  float32(s.AvgDurationSecs),
		OutcomeBreakdown:        breakdown,
	}
}

func agentStatsToProto(agentID uuid.UUID, s *dialer.AgentStats) *dialerv1.AgentDashboard {
	breakdown := make([]*dialerv1.OutcomeCount, len(s.OutcomeBreakdown))
	for i, oc := range s.OutcomeBreakdown {
		breakdown[i] = &dialerv1.OutcomeCount{
			OutcomeId:    oc.OutcomeID.String(),
			OutcomeLabel: oc.OutcomeLabel,
			OutcomeColor: oc.OutcomeColor,
			Count:        int32(oc.Count),
		}
	}
	msg := &dialerv1.AgentDashboard{
		AgentId:                agentID.String(),
		TotalCallsToday:        int32(s.TotalCallsToday),
		AverageDurationSeconds: float32(s.AvgDurationSecs),
		CallsPerHour:           float32(s.CallsPerHour),
		OutcomeBreakdown:       breakdown,
	}
	if s.ActiveCampaignID != nil {
		cid := s.ActiveCampaignID.String()
		msg.ActiveCampaignId = &cid
	}
	if s.ActiveCampaignName != nil {
		msg.ActiveCampaignName = s.ActiveCampaignName
	}
	return msg
}

// ============================================================================
// Enum converters
// ============================================================================

func campaignStatusToProto(s string) dialerv1.CampaignStatus {
	switch s {
	case dialer.CampaignStatusDraft:
		return dialerv1.CampaignStatus_CAMPAIGN_STATUS_DRAFT
	case dialer.CampaignStatusActive:
		return dialerv1.CampaignStatus_CAMPAIGN_STATUS_ACTIVE
	case dialer.CampaignStatusPaused:
		return dialerv1.CampaignStatus_CAMPAIGN_STATUS_PAUSED
	case dialer.CampaignStatusCompleted:
		return dialerv1.CampaignStatus_CAMPAIGN_STATUS_COMPLETED
	case dialer.CampaignStatusArchived:
		return dialerv1.CampaignStatus_CAMPAIGN_STATUS_ARCHIVED
	default:
		return dialerv1.CampaignStatus_CAMPAIGN_STATUS_UNSPECIFIED
	}
}

func campaignStatusFromProto(s dialerv1.CampaignStatus) string {
	switch s {
	case dialerv1.CampaignStatus_CAMPAIGN_STATUS_DRAFT:
		return dialer.CampaignStatusDraft
	case dialerv1.CampaignStatus_CAMPAIGN_STATUS_ACTIVE:
		return dialer.CampaignStatusActive
	case dialerv1.CampaignStatus_CAMPAIGN_STATUS_PAUSED:
		return dialer.CampaignStatusPaused
	case dialerv1.CampaignStatus_CAMPAIGN_STATUS_COMPLETED:
		return dialer.CampaignStatusCompleted
	case dialerv1.CampaignStatus_CAMPAIGN_STATUS_ARCHIVED:
		return dialer.CampaignStatusArchived
	default:
		return ""
	}
}

func campaignModeToProto(m string) dialerv1.CampaignMode {
	switch m {
	case dialer.CampaignModePreview:
		return dialerv1.CampaignMode_CAMPAIGN_MODE_PREVIEW
	case dialer.CampaignModePower:
		return dialerv1.CampaignMode_CAMPAIGN_MODE_POWER
	case dialer.CampaignModePredictive:
		return dialerv1.CampaignMode_CAMPAIGN_MODE_PREDICTIVE
	default:
		return dialerv1.CampaignMode_CAMPAIGN_MODE_UNSPECIFIED
	}
}

func campaignModeFromProto(m dialerv1.CampaignMode) string {
	switch m {
	case dialerv1.CampaignMode_CAMPAIGN_MODE_PREVIEW:
		return dialer.CampaignModePreview
	case dialerv1.CampaignMode_CAMPAIGN_MODE_POWER:
		return dialer.CampaignModePower
	case dialerv1.CampaignMode_CAMPAIGN_MODE_PREDICTIVE:
		return dialer.CampaignModePredictive
	default:
		return dialer.CampaignModePreview
	}
}

func contactStatusToProto(s string) dialerv1.CampaignContactStatus {
	switch s {
	case dialer.ContactStatusPending:
		return dialerv1.CampaignContactStatus_CAMPAIGN_CONTACT_STATUS_PENDING
	case dialer.ContactStatusInProgress:
		return dialerv1.CampaignContactStatus_CAMPAIGN_CONTACT_STATUS_IN_PROGRESS
	case dialer.ContactStatusCompleted:
		return dialerv1.CampaignContactStatus_CAMPAIGN_CONTACT_STATUS_COMPLETED
	case dialer.ContactStatusSkipped:
		return dialerv1.CampaignContactStatus_CAMPAIGN_CONTACT_STATUS_SKIPPED
	case dialer.ContactStatusCallback:
		return dialerv1.CampaignContactStatus_CAMPAIGN_CONTACT_STATUS_CALLBACK
	default:
		return dialerv1.CampaignContactStatus_CAMPAIGN_CONTACT_STATUS_UNSPECIFIED
	}
}

func contactStatusFromProto(s dialerv1.CampaignContactStatus) string {
	switch s {
	case dialerv1.CampaignContactStatus_CAMPAIGN_CONTACT_STATUS_PENDING:
		return dialer.ContactStatusPending
	case dialerv1.CampaignContactStatus_CAMPAIGN_CONTACT_STATUS_IN_PROGRESS:
		return dialer.ContactStatusInProgress
	case dialerv1.CampaignContactStatus_CAMPAIGN_CONTACT_STATUS_COMPLETED:
		return dialer.ContactStatusCompleted
	case dialerv1.CampaignContactStatus_CAMPAIGN_CONTACT_STATUS_SKIPPED:
		return dialer.ContactStatusSkipped
	case dialerv1.CampaignContactStatus_CAMPAIGN_CONTACT_STATUS_CALLBACK:
		return dialer.ContactStatusCallback
	default:
		return ""
	}
}

func agentStatusEnumToProto(s string) dialerv1.AgentDialerStatus {
	switch s {
	case dialer.AgentStatusAvailable:
		return dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_AVAILABLE
	case dialer.AgentStatusOnCall:
		return dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_ON_CALL
	case dialer.AgentStatusWrapUp:
		return dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_WRAP_UP
	case dialer.AgentStatusBreak:
		return dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_BREAK
	case dialer.AgentStatusOffline:
		return dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_OFFLINE
	default:
		return dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_UNSPECIFIED
	}
}

func agentStatusFromProto(s dialerv1.AgentDialerStatus) string {
	switch s {
	case dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_AVAILABLE:
		return dialer.AgentStatusAvailable
	case dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_ON_CALL:
		return dialer.AgentStatusOnCall
	case dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_WRAP_UP:
		return dialer.AgentStatusWrapUp
	case dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_BREAK:
		return dialer.AgentStatusBreak
	case dialerv1.AgentDialerStatus_AGENT_DIALER_STATUS_OFFLINE:
		return dialer.AgentStatusOffline
	default:
		return dialer.AgentStatusOffline
	}
}

// ============================================================================
// Error mapping
// ============================================================================

func mapDialerError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, dialer.ErrCampaignNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, dialer.ErrCallSessionNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, dialer.ErrOutcomeNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, dialer.ErrCampaignNotDraft):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, dialer.ErrCampaignNotActive):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, dialer.ErrInvalidStatusTransition):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, dialer.ErrNoContactsAvailable):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, dialer.ErrContactAlreadyInCampaign):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, dialer.ErrAgentNotAvailable):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, dialer.ErrCampaignHasNoContacts):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
