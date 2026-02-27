package gateway

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	automationv1 "github.com/kmuhub/kmuhub/proto/automation/v1"
)

// AutomationRoutes handles HTTP routes for the automation service.
type AutomationRoutes struct {
	registry *ServiceRegistry
}

// NewAutomationRoutes creates a new AutomationRoutes with the given service registry.
func NewAutomationRoutes(registry *ServiceRegistry) *AutomationRoutes {
	return &AutomationRoutes{registry: registry}
}

// ServiceName returns the service name for the route registrar.
func (ar *AutomationRoutes) ServiceName() string { return "automation" }

// getClient lazily obtains a gRPC client for the automation service.
func (ar *AutomationRoutes) getClient() (automationv1.AutomationServiceClient, error) {
	conn, err := ar.registry.GetConnection("automation")
	if err != nil {
		return nil, err
	}
	return automationv1.NewAutomationServiceClient(conn), nil
}

// RegisterRoutes registers all automation HTTP routes.
func (ar *AutomationRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/automations", func(r chi.Router) {
		r.Use(authMiddleware)

		// CRUD
		r.With(middleware.RequirePermission("automations", "write")).Post("/", ar.HandleCreateAutomation)
		r.With(middleware.RequirePermission("automations", "read")).Get("/", ar.HandleListAutomations)
		r.With(middleware.RequirePermission("automations", "read")).Get("/{id}", ar.HandleGetAutomation)
		r.With(middleware.RequirePermission("automations", "write")).Put("/{id}", ar.HandleUpdateAutomation)
		r.With(middleware.RequirePermission("automations", "write")).Delete("/{id}", ar.HandleDeleteAutomation)

		// Enable/Disable
		r.With(middleware.RequirePermission("automations", "write")).Post("/{id}/enable", ar.HandleEnableAutomation)
		r.With(middleware.RequirePermission("automations", "write")).Post("/{id}/disable", ar.HandleDisableAutomation)

		// Execution logs
		r.With(middleware.RequirePermission("automations", "read")).Get("/{id}/executions", ar.HandleListExecutions)
		r.With(middleware.RequirePermission("automations", "read")).Get("/executions/{executionId}", ar.HandleGetExecution)

		// Catalog
		r.With(middleware.RequirePermission("automations", "read")).Get("/triggers", ar.HandleListTriggers)
		r.With(middleware.RequirePermission("automations", "read")).Get("/actions", ar.HandleListActions)

		// Templates
		r.With(middleware.RequirePermission("automations", "read")).Get("/templates", ar.HandleListTemplates)
		r.With(middleware.RequirePermission("automations", "write")).Post("/templates/{templateId}/create", ar.HandleCreateFromTemplate)

		// Testing
		r.With(middleware.RequirePermission("automations", "write")).Post("/test-condition", ar.HandleTestCondition)
		r.With(middleware.RequirePermission("automations", "write")).Post("/dry-run", ar.HandleDryRun)

		// Stats (admin only)
		r.With(middleware.RequireRole("admin")).Get("/stats", ar.HandleGetStats)
	})
}

// ============================================================================
// CRUD Handlers
// ============================================================================

func (ar *AutomationRoutes) HandleCreateAutomation(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var body struct {
		Name          string          `json:"name"`
		Description   string          `json:"description"`
		Scope         string          `json:"scope"`
		TriggerType   string          `json:"trigger_type"`
		TriggerConfig json.RawMessage `json:"trigger_config"`
		Conditions    json.RawMessage `json:"conditions"`
		Actions       json.RawMessage `json:"actions"`
		MaxSteps      int32           `json:"max_steps"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if body.Name == "" {
		response.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	if body.TriggerType == "" {
		response.Error(w, http.StatusBadRequest, "trigger_type is required")
		return
	}

	grpcReq := &automationv1.CreateAutomationRequest{
		OwnerId:     userID,
		Name:        body.Name,
		Description: body.Description,
		Scope:       parseAutomationScope(body.Scope),
		TriggerType: body.TriggerType,
		MaxSteps:    body.MaxSteps,
	}

	if body.TriggerConfig != nil {
		if s, sErr := rawJSONToAutomationStruct(body.TriggerConfig); sErr == nil {
			grpcReq.TriggerConfig = s
		}
	}
	if body.Conditions != nil {
		if s, sErr := rawJSONToAutomationStruct(body.Conditions); sErr == nil {
			grpcReq.Conditions = s
		}
	}
	if body.Actions != nil {
		if s, sErr := rawJSONToAutomationStruct(body.Actions); sErr == nil {
			grpcReq.Actions = s
		}
	}

	resp, err := client.CreateAutomation(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (ar *AutomationRoutes) HandleListAutomations(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	grpcReq := &automationv1.ListAutomationsRequest{
		OwnerId: r.URL.Query().Get("owner_id"),
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if n, pErr := strconv.Atoi(limit); pErr == nil {
			grpcReq.Limit = int32(n)
		}
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if n, pErr := strconv.Atoi(offset); pErr == nil {
			grpcReq.Offset = int32(n)
		}
	}
	if scope := r.URL.Query().Get("scope"); scope != "" {
		s := parseAutomationScope(scope)
		grpcReq.Scope = &s
	}
	if tt := r.URL.Query().Get("trigger_type"); tt != "" {
		grpcReq.TriggerType = &tt
	}
	if isActive := r.URL.Query().Get("is_active"); isActive != "" {
		val := isActive == "true"
		grpcReq.IsActive = &val
	}

	resp, err := client.ListAutomations(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (ar *AutomationRoutes) HandleGetAutomation(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	automationID := chi.URLParam(r, "id")
	if _, parseErr := uuid.Parse(automationID); parseErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid automation id")
		return
	}

	resp, err := client.GetAutomation(r.Context(), &automationv1.GetAutomationRequest{
		AutomationId: automationID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (ar *AutomationRoutes) HandleUpdateAutomation(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	automationID := chi.URLParam(r, "id")
	if _, parseErr := uuid.Parse(automationID); parseErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid automation id")
		return
	}

	var body struct {
		Name          *string          `json:"name,omitempty"`
		Description   *string          `json:"description,omitempty"`
		Scope         *string          `json:"scope,omitempty"`
		TriggerType   *string          `json:"trigger_type,omitempty"`
		TriggerConfig *json.RawMessage `json:"trigger_config,omitempty"`
		Conditions    *json.RawMessage `json:"conditions,omitempty"`
		Actions       *json.RawMessage `json:"actions,omitempty"`
		MaxSteps      *int32           `json:"max_steps,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &automationv1.UpdateAutomationRequest{
		AutomationId: automationID,
		OwnerId:      userID,
	}
	if body.Name != nil {
		grpcReq.Name = body.Name
	}
	if body.Description != nil {
		grpcReq.Description = body.Description
	}
	if body.Scope != nil {
		s := parseAutomationScope(*body.Scope)
		grpcReq.Scope = &s
	}
	if body.TriggerType != nil {
		grpcReq.TriggerType = body.TriggerType
	}
	if body.TriggerConfig != nil {
		if s, sErr := rawJSONToAutomationStruct(*body.TriggerConfig); sErr == nil {
			grpcReq.TriggerConfig = s
		}
	}
	if body.Conditions != nil {
		if s, sErr := rawJSONToAutomationStruct(*body.Conditions); sErr == nil {
			grpcReq.Conditions = s
		}
	}
	if body.Actions != nil {
		if s, sErr := rawJSONToAutomationStruct(*body.Actions); sErr == nil {
			grpcReq.Actions = s
		}
	}
	if body.MaxSteps != nil {
		grpcReq.MaxSteps = body.MaxSteps
	}

	resp, err := client.UpdateAutomation(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (ar *AutomationRoutes) HandleDeleteAutomation(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	automationID := chi.URLParam(r, "id")
	if _, parseErr := uuid.Parse(automationID); parseErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid automation id")
		return
	}

	_, err = client.DeleteAutomation(r.Context(), &automationv1.DeleteAutomationRequest{
		AutomationId: automationID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "automation deleted"})
}

// ============================================================================
// Enable/Disable Handlers
// ============================================================================

func (ar *AutomationRoutes) HandleEnableAutomation(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	automationID := chi.URLParam(r, "id")
	if _, parseErr := uuid.Parse(automationID); parseErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid automation id")
		return
	}

	resp, err := client.EnableAutomation(r.Context(), &automationv1.EnableAutomationRequest{
		AutomationId: automationID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (ar *AutomationRoutes) HandleDisableAutomation(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	automationID := chi.URLParam(r, "id")
	if _, parseErr := uuid.Parse(automationID); parseErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid automation id")
		return
	}

	resp, err := client.DisableAutomation(r.Context(), &automationv1.DisableAutomationRequest{
		AutomationId: automationID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Execution Log Handlers
// ============================================================================

func (ar *AutomationRoutes) HandleListExecutions(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	automationID := chi.URLParam(r, "id")
	if _, parseErr := uuid.Parse(automationID); parseErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid automation id")
		return
	}

	grpcReq := &automationv1.ListExecutionsRequest{
		AutomationId: &automationID,
	}
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		s := parseExecutionStatus(statusStr)
		grpcReq.Status = &s
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if n, pErr := strconv.Atoi(limit); pErr == nil {
			grpcReq.Limit = int32(n)
		}
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if n, pErr := strconv.Atoi(offset); pErr == nil {
			grpcReq.Offset = int32(n)
		}
	}

	resp, err := client.ListExecutions(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (ar *AutomationRoutes) HandleGetExecution(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	executionID := chi.URLParam(r, "executionId")
	if _, parseErr := uuid.Parse(executionID); parseErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid execution id")
		return
	}

	resp, err := client.GetExecution(r.Context(), &automationv1.GetExecutionRequest{
		ExecutionId: executionID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Catalog Handlers
// ============================================================================

func (ar *AutomationRoutes) HandleListTriggers(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	resp, err := client.ListTriggerDefinitions(r.Context(), &automationv1.ListTriggerDefinitionsRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (ar *AutomationRoutes) HandleListActions(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	resp, err := client.ListActionDefinitions(r.Context(), &automationv1.ListActionDefinitionsRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Template Handlers
// ============================================================================

func (ar *AutomationRoutes) HandleListTemplates(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	grpcReq := &automationv1.ListTemplatesRequest{}
	if cat := r.URL.Query().Get("category"); cat != "" {
		grpcReq.Category = &cat
	}

	resp, err := client.ListTemplates(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (ar *AutomationRoutes) HandleCreateFromTemplate(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	templateID := chi.URLParam(r, "templateId")

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Name == "" {
		response.Error(w, http.StatusBadRequest, "name is required")
		return
	}

	resp, err := client.CreateFromTemplate(r.Context(), &automationv1.CreateFromTemplateRequest{
		TemplateId: templateID,
		OwnerId:    userID,
		Name:       body.Name,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

// ============================================================================
// Testing Handlers
// ============================================================================

func (ar *AutomationRoutes) HandleTestCondition(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	var body struct {
		Condition json.RawMessage        `json:"condition"`
		SampleEnv map[string]interface{} `json:"sample_env"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &automationv1.TestConditionRequest{}

	if body.Condition != nil {
		if s, sErr := rawJSONToAutomationStruct(body.Condition); sErr == nil {
			grpcReq.Conditions = s
		}
	}
	if body.SampleEnv != nil {
		if s, sErr := structpb.NewStruct(body.SampleEnv); sErr == nil {
			grpcReq.SampleEvent = s
		}
	}

	resp, err := client.TestCondition(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (ar *AutomationRoutes) HandleDryRun(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	var body struct {
		AutomationID string                 `json:"automation_id"`
		SampleEnv    map[string]interface{} `json:"sample_env"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if _, parseErr := uuid.Parse(body.AutomationID); parseErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid automation_id")
		return
	}

	grpcReq := &automationv1.DryRunAutomationRequest{
		AutomationId: body.AutomationID,
	}
	if body.SampleEnv != nil {
		if s, sErr := structpb.NewStruct(body.SampleEnv); sErr == nil {
			grpcReq.SampleEvent = s
		}
	}

	resp, err := client.DryRunAutomation(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Stats Handler
// ============================================================================

func (ar *AutomationRoutes) HandleGetStats(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	resp, err := client.GetAutomationStats(r.Context(), &automationv1.GetAutomationStatsRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Helpers
// ============================================================================

func parseAutomationScope(s string) automationv1.AutomationScope {
	switch s {
	case "personal":
		return automationv1.AutomationScope_SCOPE_PERSONAL
	case "team":
		return automationv1.AutomationScope_SCOPE_TEAM
	case "organization":
		return automationv1.AutomationScope_SCOPE_ORGANIZATION
	default:
		return automationv1.AutomationScope_SCOPE_PERSONAL
	}
}

func parseExecutionStatus(s string) automationv1.ExecutionStatus {
	switch s {
	case "running":
		return automationv1.ExecutionStatus_EXECUTION_STATUS_RUNNING
	case "completed":
		return automationv1.ExecutionStatus_EXECUTION_STATUS_COMPLETED
	case "failed":
		return automationv1.ExecutionStatus_EXECUTION_STATUS_FAILED
	case "skipped":
		return automationv1.ExecutionStatus_EXECUTION_STATUS_SKIPPED
	case "aborted":
		return automationv1.ExecutionStatus_EXECUTION_STATUS_ABORTED
	default:
		return automationv1.ExecutionStatus_EXECUTION_STATUS_UNSPECIFIED
	}
}

func rawJSONToAutomationStruct(data json.RawMessage) (*structpb.Struct, error) {
	if len(data) == 0 {
		return nil, nil
	}
	s := &structpb.Struct{}
	if err := s.UnmarshalJSON(data); err != nil {
		return nil, err
	}
	return s, nil
}
