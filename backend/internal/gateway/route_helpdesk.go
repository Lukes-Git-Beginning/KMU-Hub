package gateway

import (
	"encoding/json"
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

	r.Route("/api/v1/helpdesk", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(RequireAuthenticated)

		// Tickets
		r.With(middleware.RequirePermission("helpdesk", "write")).Post("/tickets", h.HandleCreateTicket)
		r.With(middleware.RequirePermission("helpdesk", "read")).Get("/tickets", h.HandleListTickets)
		r.With(middleware.RequirePermission("helpdesk", "read")).Get("/tickets/{id}", h.HandleGetTicket)
		r.With(middleware.RequirePermission("helpdesk", "write")).Put("/tickets/{id}", h.HandleUpdateTicket)
		r.With(middleware.RequirePermission("helpdesk", "write")).Post("/tickets/{id}/close", h.HandleCloseTicket)
		r.With(middleware.RequirePermission("helpdesk", "write")).Post("/tickets/{id}/reopen", h.HandleReopenTicket)
		r.With(middleware.RequirePermission("helpdesk", "write")).Post("/tickets/{id}/assign", h.HandleAssignTicket)
		r.With(middleware.RequirePermission("helpdesk", "write")).Post("/tickets/{id}/merge", h.HandleMergeTickets)

		// Messages
		r.With(middleware.RequirePermission("helpdesk", "write")).Post("/tickets/{id}/messages", h.HandleAddMessage)
		r.With(middleware.RequirePermission("helpdesk", "read")).Get("/tickets/{id}/messages", h.HandleListMessages)

		// Queues
		r.With(middleware.RequirePermission("helpdesk", "write")).Post("/queues", h.HandleCreateQueue)
		r.With(middleware.RequirePermission("helpdesk", "read")).Get("/queues", h.HandleListQueues)
		r.With(middleware.RequirePermission("helpdesk", "write")).Put("/queues/{id}", h.HandleUpdateQueue)
		r.With(middleware.RequirePermission("helpdesk", "write")).Delete("/queues/{id}", h.HandleDeleteQueue)

		// Canned responses
		r.With(middleware.RequirePermission("helpdesk", "write")).Post("/canned-responses", h.HandleCreateCannedResponse)
		r.With(middleware.RequirePermission("helpdesk", "read")).Get("/canned-responses", h.HandleListCannedResponses)
		r.With(middleware.RequirePermission("helpdesk", "write")).Put("/canned-responses/{id}", h.HandleUpdateCannedResponse)
		r.With(middleware.RequirePermission("helpdesk", "write")).Delete("/canned-responses/{id}", h.HandleDeleteCannedResponse)

		// SLA policies
		r.With(middleware.RequirePermission("helpdesk", "write")).Post("/sla-policies", h.HandleCreateSLAPolicy)
		r.With(middleware.RequirePermission("helpdesk", "read")).Get("/sla-policies", h.HandleListSLAPolicies)
		r.With(middleware.RequirePermission("helpdesk", "write")).Put("/sla-policies/{id}", h.HandleUpdateSLAPolicy)
		r.With(middleware.RequirePermission("helpdesk", "write")).Post("/tickets/{id}/sla", h.HandleApplySLAPolicy)
		r.With(middleware.RequirePermission("helpdesk", "read")).Get("/tickets/{id}/sla-status", h.HandleGetSLAStatus)
	})
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

	var body struct {
		Subject    string  `json:"subject"`
		Priority   string  `json:"priority"`
		AssigneeID *string `json:"assignee_id,omitempty"`
		QueueID    *string `json:"queue_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Subject == "" {
		response.Error(w, http.StatusBadRequest, "subject is required")
		return
	}

	grpcReq := &helpdeskv1.CreateTicketRequest{
		RequesterId: userID,
		Subject:     body.Subject,
		Priority:    body.Priority,
		AssigneeId:  body.AssigneeID,
		QueueId:     body.QueueID,
	}

	resp, err := client.CreateTicket(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (h *HelpdeskRoutes) HandleListTickets(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 20)

	grpcReq := &helpdeskv1.ListTicketsRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
	}
	if sf := r.URL.Query().Get("status"); sf != "" {
		grpcReq.StatusFilter = &sf
	}

	resp, err := client.ListTickets(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
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

	response.JSON(w, http.StatusOK, resp)
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

	var body struct {
		Subject    *string `json:"subject,omitempty"`
		Priority   *string `json:"priority,omitempty"`
		AssigneeID *string `json:"assignee_id,omitempty"`
		QueueID    *string `json:"queue_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &helpdeskv1.UpdateTicketRequest{
		TicketId:   ticketID,
		Subject:    body.Subject,
		Priority:   body.Priority,
		AssigneeId: body.AssigneeID,
		QueueId:    body.QueueID,
	}

	resp, err := client.UpdateTicket(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
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

	response.JSON(w, http.StatusOK, resp)
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

	response.JSON(w, http.StatusOK, resp)
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

	var body struct {
		AssigneeID string `json:"assignee_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.AssigneeID == "" {
		response.Error(w, http.StatusBadRequest, "assignee_id is required")
		return
	}

	resp, err := client.AssignTicket(r.Context(), &helpdeskv1.AssignTicketRequest{
		TicketId:   ticketID,
		AssigneeId: body.AssigneeID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
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

	var body struct {
		TargetTicketID string `json:"target_ticket_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.TargetTicketID == "" {
		response.Error(w, http.StatusBadRequest, "target_ticket_id is required")
		return
	}

	_, err = client.MergeTickets(r.Context(), &helpdeskv1.MergeTicketsRequest{
		SourceTicketId: sourceID,
		TargetTicketId: body.TargetTicketID,
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

	var body struct {
		Body        string   `json:"body"`
		Internal    bool     `json:"internal"`
		Attachments []string `json:"attachments"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Body == "" {
		response.Error(w, http.StatusBadRequest, "body is required")
		return
	}

	resp, err := client.AddMessage(r.Context(), &helpdeskv1.AddMessageRequest{
		TicketId:    ticketID,
		AuthorId:    userID,
		Body:        body.Body,
		Internal:    body.Internal,
		Attachments: body.Attachments,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
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

	response.JSON(w, http.StatusOK, resp)
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

	var body struct {
		Name              string  `json:"name"`
		DefaultAssigneeID *string `json:"default_assignee_id,omitempty"`
		SLAPolicyID       *string `json:"sla_policy_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Name == "" {
		response.Error(w, http.StatusBadRequest, "name is required")
		return
	}

	resp, err := client.CreateQueue(r.Context(), &helpdeskv1.CreateQueueRequest{
		Name:              body.Name,
		DefaultAssigneeId: body.DefaultAssigneeID,
		SlaPolicyId:       body.SLAPolicyID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (h *HelpdeskRoutes) HandleListQueues(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	resp, err := client.ListQueues(r.Context(), &helpdeskv1.ListQueuesRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
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

	var body struct {
		Name              *string `json:"name,omitempty"`
		DefaultAssigneeID *string `json:"default_assignee_id,omitempty"`
		SLAPolicyID       *string `json:"sla_policy_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.UpdateQueue(r.Context(), &helpdeskv1.UpdateQueueRequest{
		QueueId:           queueID,
		Name:              body.Name,
		DefaultAssigneeId: body.DefaultAssigneeID,
		SlaPolicyId:       body.SLAPolicyID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
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

	var body struct {
		Name string `json:"name"`
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Name == "" {
		response.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	if body.Body == "" {
		response.Error(w, http.StatusBadRequest, "body is required")
		return
	}

	resp, err := client.CreateCannedResponse(r.Context(), &helpdeskv1.CreateCannedResponseRequest{
		Name: body.Name,
		Body: body.Body,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (h *HelpdeskRoutes) HandleListCannedResponses(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	resp, err := client.ListCannedResponses(r.Context(), &helpdeskv1.ListCannedResponsesRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
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

	var body struct {
		Name *string `json:"name,omitempty"`
		Body *string `json:"body,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.UpdateCannedResponse(r.Context(), &helpdeskv1.UpdateCannedResponseRequest{
		CannedResponseId: crID,
		Name:             body.Name,
		Body:             body.Body,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
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

	var body struct {
		Name              string  `json:"name"`
		FirstResponseMins int32   `json:"first_response_mins"`
		ResolutionMins    int32   `json:"resolution_mins"`
		BusinessHours     *string `json:"business_hours,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Name == "" {
		response.Error(w, http.StatusBadRequest, "name is required")
		return
	}

	resp, err := client.CreateSLAPolicy(r.Context(), &helpdeskv1.CreateSLAPolicyRequest{
		Name:              body.Name,
		FirstResponseMins: body.FirstResponseMins,
		ResolutionMins:    body.ResolutionMins,
		BusinessHours:     body.BusinessHours,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (h *HelpdeskRoutes) HandleListSLAPolicies(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient()
	if err != nil {
		respondServiceUnavailable(w, h.ServiceName())
		return
	}

	resp, err := client.ListSLAPolicies(r.Context(), &helpdeskv1.ListSLAPoliciesRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
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

	var body struct {
		Name              *string `json:"name,omitempty"`
		FirstResponseMins *int32  `json:"first_response_mins,omitempty"`
		ResolutionMins    *int32  `json:"resolution_mins,omitempty"`
		BusinessHours     *string `json:"business_hours,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.UpdateSLAPolicy(r.Context(), &helpdeskv1.UpdateSLAPolicyRequest{
		SlaPolicyId:       policyID,
		Name:              body.Name,
		FirstResponseMins: body.FirstResponseMins,
		ResolutionMins:    body.ResolutionMins,
		BusinessHours:     body.BusinessHours,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
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

	var body struct {
		SLAPolicyID string `json:"sla_policy_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.SLAPolicyID == "" {
		response.Error(w, http.StatusBadRequest, "sla_policy_id is required")
		return
	}

	resp, err := client.ApplySLAPolicy(r.Context(), &helpdeskv1.ApplySLAPolicyRequest{
		TicketId:    ticketID,
		SlaPolicyId: body.SLAPolicyID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
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

	response.JSON(w, http.StatusOK, resp)
}
