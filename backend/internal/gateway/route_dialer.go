package gateway

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	dialerv1 "github.com/kmuhub/kmuhub/proto/dialer/v1"
)

// DialerRoutes handles HTTP routes for the Dialer backend service.
type DialerRoutes struct {
	registry *ServiceRegistry
}

// NewDialerRoutes creates a new DialerRoutes with the given service registry.
func NewDialerRoutes(registry *ServiceRegistry) *DialerRoutes {
	return &DialerRoutes{registry: registry}
}

// ServiceName returns the backend service name.
func (dr *DialerRoutes) ServiceName() string { return "dialer" }

// getDialerClient lazily obtains a gRPC client for the Dialer service.
func (dr *DialerRoutes) getDialerClient() (dialerv1.DialerServiceClient, error) {
	conn, err := dr.registry.GetConnection("dialer")
	if err != nil {
		return nil, err
	}
	return dialerv1.NewDialerServiceClient(conn), nil
}

// RegisterRoutes registers all Dialer HTTP routes.
func (dr *DialerRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/dialer", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(RequireAuthenticated)

		// Additive RBAC guards (RequirePermissionAny keeps the legacy key valid
		// while granting the capability-catalog.ts fine keys). Dialer's legacy
		// resources already live in module:subject shape, so campaigns:read,
		// calls:write, agent:manage and outcomes:manage concatenate to the exact
		// catalog key already and stay plain RequirePermission -- the one legacy
		// verb that diverges is campaigns "write" vs. the catalog's "manage"
		// action, so campaign-management routes get the additive wrapper.
		// Skip/Requeue are reachable from two FE surfaces sharing one handler:
		// the agent's live workspace queue (DialerWorkspacePage, calls:write,
		// unchanged) and the supervisor's campaign-detail queue table
		// (ContactQueueTable.tsx, "Derived from dialer:campaigns:manage") -- a
		// manager without calls:write would otherwise 403 on the second path, so
		// campaigns:manage is added as a second alternative on those two routes.
		campaignManage := middleware.RequirePermissionAny([2]string{"dialer:campaigns", "write"}, [2]string{"dialer:campaigns", "manage"})
		queueAction := middleware.RequirePermissionAny([2]string{"dialer:calls", "write"}, [2]string{"dialer:campaigns", "manage"})

		// Campaigns
		r.Route("/campaigns", func(r chi.Router) {
			r.With(middleware.RequirePermission("dialer:campaigns", "read")).Get("/", dr.HandleListCampaigns)
			r.With(campaignManage).Post("/", dr.HandleCreateCampaign)
			r.With(middleware.RequirePermission("dialer:campaigns", "read")).Get("/{id}", dr.HandleGetCampaign)
			r.With(campaignManage).Put("/{id}", dr.HandleUpdateCampaign)
			r.With(campaignManage).Post("/{id}/start", dr.HandleStartCampaign)
			r.With(campaignManage).Post("/{id}/pause", dr.HandlePauseCampaign)
			r.With(campaignManage).Delete("/{id}", dr.HandleArchiveCampaign)

			// Campaign contacts
			r.With(campaignManage).Post("/{id}/contacts", dr.HandleAddContactsToCampaign)
			r.With(middleware.RequirePermission("dialer:campaigns", "read")).Get("/{id}/contacts", dr.HandleListCampaignContacts)
			r.With(middleware.RequirePermission("dialer:calls", "write")).Get("/{id}/next", dr.HandleGetNextContact)
			r.With(queueAction).Post("/{id}/contacts/{cid}/skip", dr.HandleSkipContact)
			r.With(queueAction).Post("/{id}/contacts/{cid}/requeue", dr.HandleRequeueContact)

			// Campaign dashboard
			r.With(middleware.RequirePermission("dialer:campaigns", "read")).Get("/{id}/dashboard", dr.HandleGetCampaignDashboard)
		})

		// Calls
		r.Route("/calls", func(r chi.Router) {
			r.With(middleware.RequirePermission("dialer:calls", "write")).Post("/", dr.HandleInitiateDialerCall)
			r.With(middleware.RequirePermission("dialer:calls", "write")).Post("/{id}/outcome", dr.HandleLogCallOutcome)
			r.With(middleware.RequirePermission("dialer:calls", "write")).Put("/{id}/notes", dr.HandleSaveCallNotes)
			r.With(middleware.RequirePermission("dialer:calls", "write")).Post("/{id}/complete", dr.HandleCompleteWrapUp)
		})

		// Agent status
		r.Route("/agent-status", func(r chi.Router) {
			r.With(middleware.RequirePermission("dialer:agent", "manage")).Get("/", dr.HandleGetAgentStatus)
			r.With(middleware.RequirePermission("dialer:agent", "manage")).Put("/", dr.HandleSetAgentStatus)
			r.With(middleware.RequirePermission("dialer:agent", "manage")).Get("/campaign/{campaignId}", dr.HandleGetCampaignAgents)
		})

		// Outcomes
		r.Route("/outcomes", func(r chi.Router) {
			r.With(middleware.RequirePermission("dialer:outcomes", "manage")).Get("/", dr.HandleListCallOutcomes)
			r.With(middleware.RequirePermission("dialer:outcomes", "manage")).Post("/", dr.HandleCreateCallOutcome)
			r.With(middleware.RequirePermission("dialer:outcomes", "manage")).Put("/{id}", dr.HandleUpdateCallOutcome)
			r.With(middleware.RequirePermission("dialer:outcomes", "manage")).Delete("/{id}", dr.HandleDeleteCallOutcome)
		})

		// Agent dashboard
		r.With(middleware.RequirePermission("dialer:agent", "manage")).Get("/dashboard/agent", dr.HandleGetAgentDashboard)

		// Supervisor overview (tenant-wide aggregate)
		r.With(middleware.RequirePermission("dialer:campaigns", "read")).Get("/supervisor", dr.HandleGetSupervisorOverview)

		// Contact call history
		r.With(middleware.RequirePermission("dialer:calls", "write")).Get("/contacts/{id}/calls", dr.HandleGetContactCalls)
	})
}

// ============================================================================
// Request types
// ============================================================================

type createCampaignRequest struct {
	Name             string   `json:"name" validate:"required"`
	Description      *string  `json:"description,omitempty"`
	Mode             int32    `json:"mode" validate:"oneof=0 1 2 3"`
	AssignedAgentIDs []string `json:"assigned_agent_ids,omitempty" validate:"omitempty,dive,uuid"`
	Settings         *string  `json:"settings,omitempty"`
}

type updateCampaignRequest struct {
	Name             *string  `json:"name,omitempty" validate:"omitempty,min=1"`
	Description      *string  `json:"description,omitempty"`
	AssignedAgentIDs []string `json:"assigned_agent_ids,omitempty" validate:"omitempty,dive,uuid"`
	Settings         *string  `json:"settings,omitempty"`
}

type addContactsToCampaignRequest struct {
	ContactIDs []string `json:"contact_ids,omitempty" validate:"omitempty,dive,uuid"`
	FilterID   *string  `json:"filter_id,omitempty"`
}

type initiateDialerCallRequest struct {
	CampaignContactID string  `json:"campaign_contact_id" validate:"required,uuid"`
	AgentID           string  `json:"agent_id" validate:"omitempty,uuid"`
	CallSessionID     *string `json:"call_session_id,omitempty"`
}

type logCallOutcomeRequest struct {
	OutcomeID     string  `json:"outcome_id" validate:"required,uuid"`
	Notes         *string `json:"notes,omitempty"`
	NextAction    *string `json:"next_action,omitempty"`
	CallbackAt    *string `json:"callback_at,omitempty"`
	AppointmentID *string `json:"appointment_id,omitempty" validate:"omitempty,uuid"`
}

type saveCallNotesRequest struct {
	Notes string `json:"notes" validate:"required"`
}

type completeWrapUpRequest struct {
	AgentID string `json:"agent_id" validate:"omitempty,uuid"`
}

type setAgentStatusRequest struct {
	UserID     string  `json:"user_id" validate:"omitempty,uuid"`
	Status     int32   `json:"status" validate:"oneof=0 1 2 3 4 5"`
	CampaignID *string `json:"campaign_id,omitempty" validate:"omitempty,uuid"`
}

type createCallOutcomeRequest struct {
	Label         string `json:"label" validate:"required"`
	Color         string `json:"color"`
	IsPositive    bool   `json:"is_positive"`
	IsCallback    bool   `json:"is_callback"`
	IsAppointment bool   `json:"is_appointment"`
	SortOrder     int32  `json:"sort_order"`
}

type updateCallOutcomeRequest struct {
	Label         *string `json:"label,omitempty" validate:"omitempty,min=1"`
	Color         *string `json:"color,omitempty"`
	IsPositive    *bool   `json:"is_positive,omitempty"`
	IsCallback    *bool   `json:"is_callback,omitempty"`
	IsAppointment *bool   `json:"is_appointment,omitempty"`
	SortOrder     *int32  `json:"sort_order,omitempty"`
	IsActive      *bool   `json:"is_active,omitempty"`
}

// ============================================================================
// Campaign Handlers
// ============================================================================

func (dr *DialerRoutes) HandleListCampaigns(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 20)

	grpcReq := &dialerv1.ListCampaignsRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
	}

	if sf := r.URL.Query().Get("status_filter"); sf != "" {
		if v, err := strconv.Atoi(sf); err == nil {
			status := dialerv1.CampaignStatus(v)
			grpcReq.StatusFilter = &status
		}
	}

	resp, err := client.ListCampaigns(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (dr *DialerRoutes) HandleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	req, ok := decodeAndValidate[createCampaignRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &dialerv1.CreateCampaignRequest{
		Name:             req.Name,
		Mode:             dialerv1.CampaignMode(req.Mode),
		AssignedAgentIds: req.AssignedAgentIDs,
	}
	if req.Description != nil {
		grpcReq.Description = req.Description
	}
	if req.Settings != nil {
		grpcReq.Settings = req.Settings
	}

	resp, err := client.CreateCampaign(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp)
}

func (dr *DialerRoutes) HandleGetCampaign(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	campaignID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetCampaign(r.Context(), &dialerv1.GetCampaignRequest{CampaignId: campaignID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (dr *DialerRoutes) HandleUpdateCampaign(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	campaignID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateCampaignRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &dialerv1.UpdateCampaignRequest{
		CampaignId:       campaignID,
		AssignedAgentIds: req.AssignedAgentIDs,
	}
	if req.Name != nil {
		grpcReq.Name = req.Name
	}
	if req.Description != nil {
		grpcReq.Description = req.Description
	}
	if req.Settings != nil {
		grpcReq.Settings = req.Settings
	}

	resp, err := client.UpdateCampaign(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (dr *DialerRoutes) HandleStartCampaign(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	campaignID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.StartCampaign(r.Context(), &dialerv1.StartCampaignRequest{
		CampaignId: campaignID,
		UserId:     userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (dr *DialerRoutes) HandlePauseCampaign(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	campaignID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.PauseCampaign(r.Context(), &dialerv1.PauseCampaignRequest{
		CampaignId: campaignID,
		UserId:     userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (dr *DialerRoutes) HandleArchiveCampaign(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	campaignID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.ArchiveCampaign(r.Context(), &dialerv1.ArchiveCampaignRequest{
		CampaignId: campaignID,
		UserId:     userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "campaign archived"})
}

// ============================================================================
// Campaign Contact Handlers
// ============================================================================

func (dr *DialerRoutes) HandleAddContactsToCampaign(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	campaignID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[addContactsToCampaignRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &dialerv1.AddContactsToCampaignRequest{
		CampaignId: campaignID,
		ContactIds: req.ContactIDs,
	}
	if req.FilterID != nil {
		grpcReq.FilterId = req.FilterID
	}

	resp, err := client.AddContactsToCampaign(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (dr *DialerRoutes) HandleListCampaignContacts(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	campaignID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	page, pageSize := parsePagination(r, 1, 20)

	grpcReq := &dialerv1.ListCampaignContactsRequest{
		CampaignId: campaignID,
		Page:       int32(page),
		PageSize:   int32(pageSize),
	}

	if sf := r.URL.Query().Get("status_filter"); sf != "" {
		if v, err := strconv.Atoi(sf); err == nil {
			status := dialerv1.CampaignContactStatus(v)
			grpcReq.StatusFilter = &status
		}
	}

	resp, err := client.ListCampaignContacts(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (dr *DialerRoutes) HandleGetNextContact(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	campaignID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		agentID = middleware.GetUserID(r.Context())
	}
	if agentID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	resp, err := client.GetNextContact(r.Context(), &dialerv1.GetNextContactRequest{
		CampaignId: campaignID,
		AgentId:    agentID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (dr *DialerRoutes) HandleSkipContact(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	contactID, ok := validateUUIDParam(w, r, "cid")
	if !ok {
		return
	}

	_, err = client.SkipContact(r.Context(), &dialerv1.SkipContactRequest{
		CampaignContactId: contactID,
		AgentId:           userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "contact skipped"})
}

func (dr *DialerRoutes) HandleRequeueContact(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	contactID, ok := validateUUIDParam(w, r, "cid")
	if !ok {
		return
	}

	_, err = client.RequeueContact(r.Context(), &dialerv1.RequeueContactRequest{
		CampaignContactId: contactID,
		AgentId:           userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "contact requeued"})
}

// ============================================================================
// Call Session Handlers
// ============================================================================

func (dr *DialerRoutes) HandleInitiateDialerCall(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[initiateDialerCallRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &dialerv1.InitiateDialerCallRequest{
		CampaignContactId: req.CampaignContactID,
		AgentId:           req.AgentID,
	}
	if grpcReq.AgentId == "" {
		grpcReq.AgentId = userID
	}
	if req.CallSessionID != nil {
		grpcReq.CallSessionId = req.CallSessionID
	}

	resp, err := client.InitiateDialerCall(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp)
}

func (dr *DialerRoutes) HandleLogCallOutcome(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	sessionID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[logCallOutcomeRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &dialerv1.LogCallOutcomeRequest{
		DialerCallSessionId: sessionID,
		OutcomeId:           req.OutcomeID,
	}
	if req.Notes != nil {
		grpcReq.Notes = req.Notes
	}
	if req.NextAction != nil {
		grpcReq.NextAction = req.NextAction
	}
	if req.AppointmentID != nil {
		grpcReq.AppointmentId = req.AppointmentID
	}
	if req.CallbackAt != nil {
		t, err := time.Parse(time.RFC3339, *req.CallbackAt)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid callback_at format, expected RFC3339")
			return
		}
		grpcReq.CallbackAt = timestamppb.New(t)
	}

	resp, err := client.LogCallOutcome(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (dr *DialerRoutes) HandleSaveCallNotes(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	sessionID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[saveCallNotesRequest](w, r)
	if !ok {
		return
	}

	_, err = client.SaveCallNotes(r.Context(), &dialerv1.SaveCallNotesRequest{
		DialerCallSessionId: sessionID,
		Notes:               req.Notes,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "notes saved"})
}

func (dr *DialerRoutes) HandleCompleteWrapUp(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	sessionID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[completeWrapUpRequest](w, r)
	if !ok {
		return
	}

	agentID := req.AgentID
	if agentID == "" {
		agentID = userID
	}

	_, err = client.CompleteWrapUp(r.Context(), &dialerv1.CompleteWrapUpRequest{
		DialerCallSessionId: sessionID,
		AgentId:             agentID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "wrap-up completed"})
}

// ============================================================================
// Agent Status Handlers
// ============================================================================

func (dr *DialerRoutes) HandleGetAgentStatus(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = middleware.GetUserID(r.Context())
	}
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	resp, err := client.GetAgentStatus(r.Context(), &dialerv1.GetAgentStatusRequest{
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (dr *DialerRoutes) HandleSetAgentStatus(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[setAgentStatusRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &dialerv1.SetAgentStatusRequest{
		UserId: req.UserID,
		Status: dialerv1.AgentDialerStatus(req.Status),
	}
	if grpcReq.UserId == "" {
		grpcReq.UserId = userID
	}
	if req.CampaignID != nil {
		grpcReq.CampaignId = req.CampaignID
	}

	resp, err := client.SetAgentStatus(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (dr *DialerRoutes) HandleGetCampaignAgents(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	campaignID, ok := validateUUIDParam(w, r, "campaignId")
	if !ok {
		return
	}

	resp, err := client.GetCampaignAgents(r.Context(), &dialerv1.GetCampaignAgentsRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Call Outcome Handlers
// ============================================================================

func (dr *DialerRoutes) HandleListCallOutcomes(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	grpcReq := &dialerv1.ListCallOutcomesRequest{}

	if ia := r.URL.Query().Get("include_inactive"); ia != "" {
		if v, err := strconv.ParseBool(ia); err == nil {
			grpcReq.IncludeInactive = &v
		}
	}

	resp, err := client.ListCallOutcomes(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (dr *DialerRoutes) HandleCreateCallOutcome(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	req, ok := decodeAndValidate[createCallOutcomeRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateCallOutcome(r.Context(), &dialerv1.CreateCallOutcomeRequest{
		Label:         req.Label,
		Color:         req.Color,
		IsPositive:    req.IsPositive,
		IsCallback:    req.IsCallback,
		IsAppointment: req.IsAppointment,
		SortOrder:     req.SortOrder,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp)
}

func (dr *DialerRoutes) HandleUpdateCallOutcome(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	outcomeID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateCallOutcomeRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &dialerv1.UpdateCallOutcomeRequest{
		OutcomeId: outcomeID,
	}
	if req.Label != nil {
		grpcReq.Label = req.Label
	}
	if req.Color != nil {
		grpcReq.Color = req.Color
	}
	if req.IsPositive != nil {
		grpcReq.IsPositive = req.IsPositive
	}
	if req.IsCallback != nil {
		grpcReq.IsCallback = req.IsCallback
	}
	if req.IsAppointment != nil {
		grpcReq.IsAppointment = req.IsAppointment
	}
	if req.SortOrder != nil {
		grpcReq.SortOrder = req.SortOrder
	}
	if req.IsActive != nil {
		grpcReq.IsActive = req.IsActive
	}

	resp, err := client.UpdateCallOutcome(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (dr *DialerRoutes) HandleDeleteCallOutcome(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	outcomeID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteCallOutcome(r.Context(), &dialerv1.DeleteCallOutcomeRequest{
		OutcomeId: outcomeID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "outcome deleted"})
}

// ============================================================================
// Dashboard Handlers
// ============================================================================

func (dr *DialerRoutes) HandleGetCampaignDashboard(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	campaignID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetCampaignDashboard(r.Context(), &dialerv1.GetCampaignDashboardRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (dr *DialerRoutes) HandleGetAgentDashboard(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		agentID = middleware.GetUserID(r.Context())
	}
	if agentID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	resp, err := client.GetAgentDashboard(r.Context(), &dialerv1.GetAgentDashboardRequest{
		AgentId: agentID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Supervisor Handler
// ============================================================================

func (dr *DialerRoutes) HandleGetSupervisorOverview(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	resp, err := client.GetSupervisorOverview(r.Context(), &dialerv1.GetSupervisorOverviewRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Contact Call History Handler
// ============================================================================

func (dr *DialerRoutes) HandleGetContactCalls(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDialerClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	contactID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetContactCalls(r.Context(), &dialerv1.GetContactCallsRequest{
		CampaignContactId: contactID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}
