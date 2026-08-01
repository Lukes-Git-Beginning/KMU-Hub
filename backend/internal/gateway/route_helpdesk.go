package gateway

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/featureflag"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	helpdeskv1 "github.com/kmuhub/kmuhub/proto/helpdesk/v1"
)

// HelpdeskRoutes handles HTTP routes for the Helpdesk module.
type HelpdeskRoutes struct {
	registry *ServiceRegistry
	flags    *featureflag.Registry
}

// NewHelpdeskRoutes creates a new HelpdeskRoutes with the given service registry and feature flags.
func NewHelpdeskRoutes(registry *ServiceRegistry, flags *featureflag.Registry) *HelpdeskRoutes {
	return &HelpdeskRoutes{registry: registry, flags: flags}
}

// ServiceName returns the service name for the route registrar.
func (h *HelpdeskRoutes) ServiceName() string { return "helpdesk" }

// getClient lazily obtains a gRPC client for the helpdesk service.
func (h *HelpdeskRoutes) getClient() (helpdeskv1.HelpdeskServiceClient, error) {
	conn, err := h.registry.GetConnection("helpdesk")
	if err != nil {
		return nil, err
	}
	return helpdeskv1.NewHelpdeskServiceClient(conn), nil
}

// RegisterRoutes mounts all helpdesk HTTP routes behind the feature flag modules.helpdesk.
// Routes are only registered if the flag is enabled.
func (h *HelpdeskRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	if !h.flags.IsEnabled("modules.helpdesk") {
		return
	}

	// Additive guards: every route keeps its legacy coarse key AND accepts the
	// matching capability-catalogue key. Permissions are baked into the access
	// token at login and never re-read per request, so swapping a key outright
	// would 403 every user holding a still-valid token. Extend, never swap.
	//
	// Queues, SLA policies, routing rules and business hours have no
	// catalogue counterpart (backend-gaps §RBAC: helpdesk BE seeds only the
	// coarse read/write pair beyond tickets/kb/canned/stats) — those routes
	// stay on the coarse key as-is.
	var (
		hdTicketRead   = middleware.RequirePermissionAny([2]string{"helpdesk", "read"}, [2]string{"helpdesk:ticket", "read"})
		hdTicketCreate = middleware.RequirePermissionAny([2]string{"helpdesk", "write"}, [2]string{"helpdesk:ticket", "create"})
		hdTicketEdit   = middleware.RequirePermissionAny([2]string{"helpdesk", "write"}, [2]string{"helpdesk:ticket", "edit"})
		hdTicketReply  = middleware.RequirePermissionAny([2]string{"helpdesk", "write"}, [2]string{"helpdesk:ticket", "reply"})
		hdKbManage     = middleware.RequirePermissionAny([2]string{"helpdesk", "write"}, [2]string{"helpdesk:kb", "manage"})
		hdCannedManage = middleware.RequirePermissionAny([2]string{"helpdesk", "write"}, [2]string{"helpdesk:canned", "manage"})
		hdStatsView    = middleware.RequirePermissionAny([2]string{"helpdesk", "read"}, [2]string{"helpdesk:stats", "view"})
	)

	r.Route("/api/v1/helpdesk", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(RequireAuthenticated)

		// Tickets. close/reopen/assign/merge are state changes on the ticket
		// and map to ticket:edit, matching the FE gate in HelpdeskPage.tsx.
		r.With(hdTicketCreate).Post("/tickets", h.HandleCreateTicket)
		r.With(hdTicketRead).Get("/tickets", h.HandleListTickets)
		r.With(hdTicketRead).Get("/tickets/{id}", h.HandleGetTicket)
		r.With(hdTicketEdit).Put("/tickets/{id}", h.HandleUpdateTicket)
		r.With(hdTicketEdit).Post("/tickets/{id}/close", h.HandleCloseTicket)
		r.With(hdTicketEdit).Post("/tickets/{id}/reopen", h.HandleReopenTicket)
		r.With(hdTicketEdit).Post("/tickets/{id}/assign", h.HandleAssignTicket)
		r.With(hdTicketEdit).Post("/tickets/{id}/merge", h.HandleMergeTickets)

		// Messages
		r.With(hdTicketReply).Post("/tickets/{id}/messages", h.HandleAddMessage)
		r.With(hdTicketRead).Get("/tickets/{id}/messages", h.HandleListMessages)

		// Queues
		r.With(middleware.RequirePermission("helpdesk", "write")).Post("/queues", h.HandleCreateQueue)
		r.With(middleware.RequirePermission("helpdesk", "read")).Get("/queues", h.HandleListQueues)
		r.With(middleware.RequirePermission("helpdesk", "write")).Put("/queues/{id}", h.HandleUpdateQueue)
		r.With(middleware.RequirePermission("helpdesk", "write")).Delete("/queues/{id}", h.HandleDeleteQueue)

		// Canned responses
		r.With(hdCannedManage).Post("/canned-responses", h.HandleCreateCannedResponse)
		r.With(middleware.RequirePermission("helpdesk", "read")).Get("/canned-responses", h.HandleListCannedResponses)
		r.With(hdCannedManage).Put("/canned-responses/{id}", h.HandleUpdateCannedResponse)
		r.With(hdCannedManage).Delete("/canned-responses/{id}", h.HandleDeleteCannedResponse)

		// SLA policies
		r.With(middleware.RequirePermission("helpdesk", "write")).Post("/sla-policies", h.HandleCreateSLAPolicy)
		r.With(middleware.RequirePermission("helpdesk", "read")).Get("/sla-policies", h.HandleListSLAPolicies)
		r.With(middleware.RequirePermission("helpdesk", "write")).Put("/sla-policies/{id}", h.HandleUpdateSLAPolicy)
		r.With(middleware.RequirePermission("helpdesk", "write")).Delete("/sla-policies/{id}", h.HandleDeleteSLAPolicy)
		r.With(middleware.RequirePermission("helpdesk", "write")).Post("/tickets/{id}/sla", h.HandleApplySLAPolicy)
		r.With(middleware.RequirePermission("helpdesk", "read")).Get("/tickets/{id}/sla-status", h.HandleGetSLAStatus)

		// Knowledge-base articles
		r.With(middleware.RequirePermission("helpdesk", "read")).Get("/kb-articles", h.HandleListKBArticles)
		r.With(hdKbManage).Post("/kb-articles", h.HandleCreateKBArticle)
		r.With(hdKbManage).Put("/kb-articles/{id}", h.HandleUpdateKBArticle)
		r.With(hdKbManage).Delete("/kb-articles/{id}", h.HandleDeleteKBArticle)

		// Routing rules
		r.With(middleware.RequirePermission("helpdesk", "read")).Get("/routing-rules", h.HandleListRoutingRules)
		r.With(middleware.RequirePermission("helpdesk", "write")).Post("/routing-rules", h.HandleCreateRoutingRule)
		r.With(middleware.RequirePermission("helpdesk", "write")).Put("/routing-rules/{id}", h.HandleUpdateRoutingRule)
		r.With(middleware.RequirePermission("helpdesk", "write")).Delete("/routing-rules/{id}", h.HandleDeleteRoutingRule)

		// Business hours
		r.With(middleware.RequirePermission("helpdesk", "read")).Get("/business-hours", h.HandleGetBusinessHours)
		r.With(middleware.RequirePermission("helpdesk", "write")).Put("/business-hours", h.HandleUpdateBusinessHours)

		// Stats
		r.With(hdStatsView).Get("/stats", h.HandleGetHelpdeskStats)
	})
}

// ============================================================================
// Request types
// ============================================================================

type createTicketRequest struct {
	Subject     string  `json:"subject" validate:"required,max=200"`
	Priority    string  `json:"priority" validate:"omitempty,oneof=low normal high urgent"`
	AssigneeID  *string `json:"assignee_id,omitempty" validate:"omitempty,uuid"`
	QueueID     *string `json:"queue_id,omitempty" validate:"omitempty,uuid"`
	Description *string `json:"description,omitempty"`
	Category    *string `json:"category,omitempty" validate:"omitempty,max=100"`
}

type updateTicketRequest struct {
	Subject    *string `json:"subject,omitempty" validate:"omitempty,min=1,max=200"`
	Priority   *string `json:"priority,omitempty" validate:"omitempty,oneof=low normal high urgent"`
	AssigneeID *string `json:"assignee_id,omitempty" validate:"omitempty,uuid"`
	QueueID    *string `json:"queue_id,omitempty" validate:"omitempty,uuid"`
}

type assignTicketRequest struct {
	AssigneeID string `json:"assignee_id" validate:"required,uuid"`
}

type mergeTicketsRequest struct {
	TargetTicketID string `json:"target_ticket_id" validate:"required,uuid"`
}

type addMessageRequest struct {
	Body        string   `json:"body" validate:"required"`
	Internal    bool     `json:"internal"`
	Attachments []string `json:"attachments"`
}

type createQueueRequest struct {
	Name              string  `json:"name" validate:"required"`
	DefaultAssigneeID *string `json:"default_assignee_id,omitempty" validate:"omitempty,uuid"`
	SLAPolicyID       *string `json:"sla_policy_id,omitempty" validate:"omitempty,uuid"`
}

type updateQueueRequest struct {
	Name              *string `json:"name,omitempty" validate:"omitempty,min=1"`
	DefaultAssigneeID *string `json:"default_assignee_id,omitempty" validate:"omitempty,uuid"`
	SLAPolicyID       *string `json:"sla_policy_id,omitempty" validate:"omitempty,uuid"`
}

type createCannedResponseRequest struct {
	Name string `json:"name" validate:"required"`
	Body string `json:"body" validate:"required"`
}

type updateCannedResponseRequest struct {
	Name *string `json:"name,omitempty" validate:"omitempty,min=1"`
	Body *string `json:"body,omitempty" validate:"omitempty,min=1"`
}

type createSLAPolicyRequest struct {
	Name              string  `json:"name" validate:"required"`
	FirstResponseMins int32   `json:"first_response_mins"`
	ResolutionMins    int32   `json:"resolution_mins"`
	BusinessHours     *string `json:"business_hours,omitempty"`
}

type updateSLAPolicyRequest struct {
	Name              *string `json:"name,omitempty" validate:"omitempty,min=1"`
	FirstResponseMins *int32  `json:"first_response_mins,omitempty"`
	ResolutionMins    *int32  `json:"resolution_mins,omitempty"`
	BusinessHours     *string `json:"business_hours,omitempty"`
}

type applySLAPolicyRequest struct {
	SLAPolicyID string `json:"sla_policy_id" validate:"required,uuid"`
}

type createKBArticleRequest struct {
	Title    string `json:"title" validate:"required,max=300"`
	Content  string `json:"content"`
	Category string `json:"category" validate:"omitempty,max=100"`
	Status   string `json:"status" validate:"omitempty,oneof=draft published"`
}

type updateKBArticleRequest struct {
	Title    *string `json:"title,omitempty" validate:"omitempty,min=1,max=300"`
	Content  *string `json:"content,omitempty"`
	Category *string `json:"category,omitempty" validate:"omitempty,max=100"`
	Status   *string `json:"status,omitempty" validate:"omitempty,oneof=draft published"`
}

type updateBusinessHoursRequest struct {
	ScheduleJSON string `json:"schedule_json" validate:"required"`
	HolidaysJSON string `json:"holidays_json" validate:"required"`
	Timezone     string `json:"timezone" validate:"required"`
}

type createHelpdeskRoutingRuleRequest struct {
	Name          string  `json:"name" validate:"required"`
	Conditions    string  `json:"conditions"`
	TargetQueueID *string `json:"target_queue_id,omitempty" validate:"omitempty,uuid"`
	Priority      int32   `json:"priority"`
	Enabled       bool    `json:"enabled"`
}

type updateHelpdeskRoutingRuleRequest struct {
	Name          *string `json:"name,omitempty" validate:"omitempty,min=1"`
	Conditions    *string `json:"conditions,omitempty"`
	TargetQueueID *string `json:"target_queue_id,omitempty" validate:"omitempty,uuid"`
	Priority      *int32  `json:"priority,omitempty"`
	Enabled       *bool   `json:"enabled,omitempty"`
}

// ============================================================================
// Ticket Handlers
// ============================================================================

func (h *HelpdeskRoutes) HandleCreateTicket(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	req, ok := decodeAndValidate[createTicketRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &helpdeskv1.CreateTicketRequest{
		TenantId:    tenantID.String(),
		RequesterId: userID,
		Subject:     req.Subject,
		Priority:    req.Priority,
		AssigneeId:  req.AssigneeID,
		QueueId:     req.QueueID,
		Description: req.Description,
		Category:    req.Category,
	}

	resp, err := client.CreateTicket(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp)
}

func (h *HelpdeskRoutes) HandleListTickets(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	page, pageSize := parsePagination(r, 1, 20)

	grpcReq := &helpdeskv1.ListTicketsRequest{
		TenantId: tenantID.String(),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}
	if sf := r.URL.Query().Get("status"); sf != "" {
		grpcReq.StatusFilter = &sf
	}
	// At scope "own" the list shrinks to tickets the caller raised or is
	// assigned — the same rows the ticket detail view would let them open.
	ownerID, ok := ownerFilterForScope(w, r, "helpdesk:ticket", "read")
	if !ok {
		return
	}
	grpcReq.ParticipantId = ownerID

	resp, err := client.ListTickets(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (h *HelpdeskRoutes) HandleGetTicket(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	ticketID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetTicket(r.Context(), &helpdeskv1.GetTicketRequest{TicketId: ticketID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (h *HelpdeskRoutes) HandleUpdateTicket(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	ticketID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateTicketRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &helpdeskv1.UpdateTicketRequest{
		TicketId:   ticketID,
		Subject:    req.Subject,
		Priority:   req.Priority,
		AssigneeId: req.AssigneeID,
		QueueId:    req.QueueID,
	}

	resp, err := client.UpdateTicket(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (h *HelpdeskRoutes) HandleCloseTicket(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	ticketID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.CloseTicket(r.Context(), &helpdeskv1.CloseTicketRequest{TicketId: ticketID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (h *HelpdeskRoutes) HandleReopenTicket(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	ticketID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ReopenTicket(r.Context(), &helpdeskv1.ReopenTicketRequest{TicketId: ticketID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (h *HelpdeskRoutes) HandleAssignTicket(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	ticketID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[assignTicketRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.AssignTicket(r.Context(), &helpdeskv1.AssignTicketRequest{
		TicketId:   ticketID,
		AssigneeId: req.AssigneeID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (h *HelpdeskRoutes) HandleMergeTickets(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	sourceID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[mergeTicketsRequest](w, r)
	if !ok {
		return
	}

	_, err = client.MergeTickets(r.Context(), &helpdeskv1.MergeTicketsRequest{
		SourceTicketId: sourceID,
		TargetTicketId: req.TargetTicketID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "tickets merged"})
}

// ============================================================================
// Message Handlers
// ============================================================================

func (h *HelpdeskRoutes) HandleAddMessage(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	ticketID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[addMessageRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.AddMessage(r.Context(), &helpdeskv1.AddMessageRequest{
		TicketId:    ticketID,
		AuthorId:    userID,
		Body:        req.Body,
		Internal:    req.Internal,
		Attachments: req.Attachments,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp)
}

func (h *HelpdeskRoutes) HandleListMessages(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	ticketID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ListMessages(r.Context(), &helpdeskv1.ListMessagesRequest{TicketId: ticketID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Queue Handlers
// ============================================================================

func (h *HelpdeskRoutes) HandleCreateQueue(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	req, ok := decodeAndValidate[createQueueRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateQueue(r.Context(), &helpdeskv1.CreateQueueRequest{
		Name:              req.Name,
		DefaultAssigneeId: req.DefaultAssigneeID,
		SlaPolicyId:       req.SLAPolicyID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp)
}

func (h *HelpdeskRoutes) HandleListQueues(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	resp, err := client.ListQueues(r.Context(), &helpdeskv1.ListQueuesRequest{TenantId: tenantID.String()})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (h *HelpdeskRoutes) HandleUpdateQueue(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	queueID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateQueueRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateQueue(r.Context(), &helpdeskv1.UpdateQueueRequest{
		QueueId:           queueID,
		Name:              req.Name,
		DefaultAssigneeId: req.DefaultAssigneeID,
		SlaPolicyId:       req.SLAPolicyID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (h *HelpdeskRoutes) HandleDeleteQueue(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	queueID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteQueue(r.Context(), &helpdeskv1.DeleteQueueRequest{QueueId: queueID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "queue deleted"})
}

// ============================================================================
// Canned Response Handlers
// ============================================================================

func (h *HelpdeskRoutes) HandleCreateCannedResponse(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	req, ok := decodeAndValidate[createCannedResponseRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateCannedResponse(r.Context(), &helpdeskv1.CreateCannedResponseRequest{
		Name: req.Name,
		Body: req.Body,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp)
}

func (h *HelpdeskRoutes) HandleListCannedResponses(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	resp, err := client.ListCannedResponses(r.Context(), &helpdeskv1.ListCannedResponsesRequest{TenantId: tenantID.String()})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (h *HelpdeskRoutes) HandleUpdateCannedResponse(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	crID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateCannedResponseRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateCannedResponse(r.Context(), &helpdeskv1.UpdateCannedResponseRequest{
		CannedResponseId: crID,
		Name:             req.Name,
		Body:             req.Body,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (h *HelpdeskRoutes) HandleDeleteCannedResponse(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	crID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteCannedResponse(r.Context(), &helpdeskv1.DeleteCannedResponseRequest{CannedResponseId: crID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "canned response deleted"})
}

// ============================================================================
// SLA Policy Handlers
// ============================================================================

func (h *HelpdeskRoutes) HandleCreateSLAPolicy(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	req, ok := decodeAndValidate[createSLAPolicyRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateSLAPolicy(r.Context(), &helpdeskv1.CreateSLAPolicyRequest{
		Name:              req.Name,
		FirstResponseMins: req.FirstResponseMins,
		ResolutionMins:    req.ResolutionMins,
		BusinessHours:     req.BusinessHours,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp)
}

func (h *HelpdeskRoutes) HandleListSLAPolicies(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	resp, err := client.ListSLAPolicies(r.Context(), &helpdeskv1.ListSLAPoliciesRequest{TenantId: tenantID.String()})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (h *HelpdeskRoutes) HandleUpdateSLAPolicy(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	policyID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateSLAPolicyRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateSLAPolicy(r.Context(), &helpdeskv1.UpdateSLAPolicyRequest{
		SlaPolicyId:       policyID,
		Name:              req.Name,
		FirstResponseMins: req.FirstResponseMins,
		ResolutionMins:    req.ResolutionMins,
		BusinessHours:     req.BusinessHours,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (h *HelpdeskRoutes) HandleApplySLAPolicy(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	ticketID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[applySLAPolicyRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.ApplySLAPolicy(r.Context(), &helpdeskv1.ApplySLAPolicyRequest{
		TicketId:    ticketID,
		SlaPolicyId: req.SLAPolicyID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (h *HelpdeskRoutes) HandleGetSLAStatus(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	ticketID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	grpcReq := &helpdeskv1.GetSLAStatusRequest{TicketId: ticketID}
	if pid := r.URL.Query().Get("sla_policy_id"); pid != "" {
		grpcReq.SlaPolicyId = &pid
	}

	resp, err := client.GetSLAStatus(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (h *HelpdeskRoutes) HandleDeleteSLAPolicy(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	policyID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteSLAPolicy(r.Context(), &helpdeskv1.DeleteSLAPolicyRequest{SlaPolicyId: policyID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "sla policy deleted"})
}

// ============================================================================
// KB Article Handlers
// ============================================================================

func (h *HelpdeskRoutes) HandleListKBArticles(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	resp, err := client.ListKBArticle(r.Context(), &helpdeskv1.ListKBArticleRequest{TenantId: tenantID.String()})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (h *HelpdeskRoutes) HandleCreateKBArticle(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	req, ok := decodeAndValidate[createKBArticleRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateKBArticle(r.Context(), &helpdeskv1.CreateKBArticleRequest{
		TenantId: tenantID.String(),
		Title:    req.Title,
		Content:  req.Content,
		Category: req.Category,
		Status:   req.Status,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp)
}

func (h *HelpdeskRoutes) HandleUpdateKBArticle(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	articleID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateKBArticleRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateKBArticle(r.Context(), &helpdeskv1.UpdateKBArticleRequest{
		ArticleId: articleID,
		Title:     req.Title,
		Content:   req.Content,
		Category:  req.Category,
		Status:    req.Status,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (h *HelpdeskRoutes) HandleDeleteKBArticle(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	articleID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteKBArticle(r.Context(), &helpdeskv1.DeleteKBArticleRequest{ArticleId: articleID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "kb article deleted"})
}

// ============================================================================
// Routing Rule Handlers
// ============================================================================

func (h *HelpdeskRoutes) HandleListRoutingRules(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	resp, err := client.ListRoutingRule(r.Context(), &helpdeskv1.ListRoutingRuleRequest{TenantId: tenantID.String()})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (h *HelpdeskRoutes) HandleCreateRoutingRule(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	req, ok := decodeAndValidate[createHelpdeskRoutingRuleRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &helpdeskv1.CreateRoutingRuleRequest{
		TenantId:      tenantID.String(),
		Name:          req.Name,
		Conditions:    req.Conditions,
		TargetQueueId: req.TargetQueueID,
		Priority:      req.Priority,
		Enabled:       req.Enabled,
	}

	resp, err := client.CreateRoutingRule(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp)
}

func (h *HelpdeskRoutes) HandleUpdateRoutingRule(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	ruleID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateHelpdeskRoutingRuleRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateRoutingRule(r.Context(), &helpdeskv1.UpdateRoutingRuleRequest{
		RuleId:        ruleID,
		Name:          req.Name,
		Conditions:    req.Conditions,
		TargetQueueId: req.TargetQueueID,
		Priority:      req.Priority,
		Enabled:       req.Enabled,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (h *HelpdeskRoutes) HandleDeleteRoutingRule(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	ruleID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteRoutingRule(r.Context(), &helpdeskv1.DeleteRoutingRuleRequest{RuleId: ruleID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "routing rule deleted"})
}

// ============================================================================
// Business Hours Handlers
// ============================================================================

func (h *HelpdeskRoutes) HandleGetBusinessHours(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	resp, err := client.GetBusinessHours(r.Context(), &helpdeskv1.GetBusinessHoursRequest{
		TenantId: tenantID.String(),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (h *HelpdeskRoutes) HandleUpdateBusinessHours(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	req, ok := decodeAndValidate[updateBusinessHoursRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateBusinessHours(r.Context(), &helpdeskv1.UpdateBusinessHoursRequest{
		TenantId:     tenantID.String(),
		ScheduleJson: req.ScheduleJSON,
		HolidaysJson: req.HolidaysJSON,
		Timezone:     req.Timezone,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Stats Handler
// ============================================================================

func (h *HelpdeskRoutes) HandleGetHelpdeskStats(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	resp, err := client.GetHelpdeskStats(r.Context(), &helpdeskv1.GetHelpdeskStatsRequest{TenantId: tenantID.String()})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}
