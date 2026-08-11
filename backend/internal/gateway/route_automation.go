package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	automationv1 "github.com/kmuhub/kmuhub/proto/automation/v1"
)

// webhookSignatureHeader is the header inbound webhook deliveries carry their
// HMAC-SHA256 signature in, matching the outbound convention already used by
// internal/formulare/worker.go (X-Cosmi-Signature: sha256=<hex>).
const webhookSignatureHeader = "X-Cosmi-Signature"

// maxWebhookBodyBytes caps the request body accepted by HandleTriggerWebhook.
// Must match (or be tighter than) workflow.maxWebhookBodyBytes -- kept as an
// independent constant here since the gateway package cannot import the
// unexported workflow constant, and enforcing it before the bytes ever leave
// this handler is the point (defense-in-depth, not merely a mirrored value).
const maxWebhookBodyBytes = 256 * 1024

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
		r.Use(RequireAuthenticated)

		// Additive RBAC guards (RequirePermissionAny keeps the legacy key valid
		// while granting the capability-catalog.ts fine keys AutomatisierungPage.tsx/
		// AutomationDetailModal.tsx actually gate on). Unlike every other module in
		// this loop, the legacy resource here is the bare "automations" WITHOUT the
		// module prefix (backend-gaps.md, seeded pre-000129) -- so NONE of the
		// legacy keys concatenate to a catalog key by coincidence, and every route
		// needs the additive wrapper. Triggers/actions/templates have no dedicated
		// catalog verb of their own: they're read/authoring helpers consumed from
		// AutomationDetailModal (opened from the automations list, automations:read)
		// and the "templates" tab, which AutomatisierungPage.tsx gates on
		// automations:create (browsing templates is only useful if you can
		// instantiate one). Test-condition/dry-run run from AutomationWizard, which
		// is reused for both the create flow and "Bearbeiten" edit hand-off, so they
		// accept either automations:create or automations:edit. Stats stays on
		// RequireRole("admin") -- a different mechanism entirely, no catalog key.
		automationsRead := middleware.RequirePermissionAny([2]string{"automations", "read"}, [2]string{"automatisierung:automations", "read"})
		automationsCreate := middleware.RequirePermissionAny([2]string{"automations", "write"}, [2]string{"automatisierung:automations", "create"})
		automationsEdit := middleware.RequirePermissionAny([2]string{"automations", "write"}, [2]string{"automatisierung:automations", "edit"})
		automationsDelete := middleware.RequirePermissionAny([2]string{"automations", "write"}, [2]string{"automatisierung:automations", "delete"})
		automationsToggle := middleware.RequirePermissionAny([2]string{"automations", "write"}, [2]string{"automatisierung:automations", "toggle"})
		executionsRead := middleware.RequirePermissionAny([2]string{"automations", "read"}, [2]string{"automatisierung:executions", "read"})
		automationsAuthor := middleware.RequirePermissionAny([2]string{"automations", "write"}, [2]string{"automatisierung:automations", "create"}, [2]string{"automatisierung:automations", "edit"})

		// CRUD
		r.With(automationsCreate).Post("/", ar.HandleCreateAutomation)
		r.With(automationsRead).Get("/", ar.HandleListAutomations)
		r.With(automationsRead).Get("/{id}", ar.HandleGetAutomation)
		r.With(automationsEdit).Put("/{id}", ar.HandleUpdateAutomation)
		r.With(automationsDelete).Delete("/{id}", ar.HandleDeleteAutomation)

		// Enable/Disable
		r.With(automationsToggle).Post("/{id}/enable", ar.HandleEnableAutomation)
		r.With(automationsToggle).Post("/{id}/disable", ar.HandleDisableAutomation)

		// Execution logs
		r.With(executionsRead).Get("/{id}/executions", ar.HandleListExecutions)
		r.With(executionsRead).Get("/executions/{executionId}", ar.HandleGetExecution)

		// Catalog
		r.With(automationsRead).Get("/triggers", ar.HandleListTriggers)
		r.With(automationsRead).Get("/actions", ar.HandleListActions)

		// Templates
		r.With(automationsCreate).Get("/templates", ar.HandleListTemplates)
		r.With(automationsCreate).Post("/templates/{templateId}/create", ar.HandleCreateFromTemplate)

		// Testing
		r.With(automationsAuthor).Post("/test-condition", ar.HandleTestCondition)
		r.With(automationsAuthor).Post("/dry-run", ar.HandleDryRun)

		// Stats (admin only)
		r.With(middleware.RequireRole("admin")).Get("/stats", ar.HandleGetStats)
	})
}

// RegisterPublicRoutes mounts the unauthenticated inbound webhook trigger.
// Called from cmd/gateway/main.go OUTSIDE the registrars loop, directly on r,
// following the booking/berichte/document public-route pattern.
//
// publicRateLimit must be the stricter public limiter, not the global one:
// this is the only path in the module that answers without a JWT, and the
// per-automation HMAC secret is the only other thing standing between an
// external caller and repeated signature-guessing/DoS attempts.
func (ar *AutomationRoutes) RegisterPublicRoutes(r chi.Router, publicRateLimit func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(publicRateLimit)
		r.Post("/api/v1/public/automations/webhooks/{automationId}", ar.HandleTriggerWebhook)
	})
}

// ============================================================================
// CRUD Handlers
// ============================================================================

type createAutomationRequest struct {
	Name          string          `json:"name"         validate:"required"`
	Description   string          `json:"description"`
	Scope         string          `json:"scope"        validate:"omitempty,oneof=personal team organization"`
	TriggerType   string          `json:"trigger_type" validate:"required"`
	TriggerConfig json.RawMessage `json:"trigger_config"`
	Conditions    json.RawMessage `json:"conditions"`
	Actions       json.RawMessage `json:"actions"`
	MaxSteps      int32           `json:"max_steps"`
}

func (ar *AutomationRoutes) HandleCreateAutomation(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	body, ok := decodeAndValidate[createAutomationRequest](w, r)
	if !ok {
		return
	}

	scope, ok := parseAutomationScope(body.Scope)
	if !ok {
		response.Error(w, http.StatusBadRequest, "scope must be personal, team, or organization, got: "+body.Scope)
		return
	}

	grpcReq := &automationv1.CreateAutomationRequest{
		OwnerId:     userID,
		Name:        body.Name,
		Description: body.Description,
		Scope:       scope,
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
		if l, lErr := rawJSONToAutomationList(body.Actions); lErr == nil {
			grpcReq.Actions = l
		}
	}

	resp, err := client.CreateAutomation(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp)
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
		s, ok := parseAutomationScope(scope)
		if !ok {
			response.Error(w, http.StatusBadRequest, "scope must be personal, team, or organization, got: "+scope)
			return
		}
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

	response.Proto(w, http.StatusOK, resp)
}

func (ar *AutomationRoutes) HandleGetAutomation(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	automationID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetAutomation(r.Context(), &automationv1.GetAutomationRequest{
		AutomationId: automationID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (ar *AutomationRoutes) HandleUpdateAutomation(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	automationID, ok := validateUUIDParam(w, r, "id")
	if !ok {
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
		s, ok := parseAutomationScope(*body.Scope)
		if !ok {
			response.Error(w, http.StatusBadRequest, "scope must be personal, team, or organization, got: "+*body.Scope)
			return
		}
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
		if l, lErr := rawJSONToAutomationList(*body.Actions); lErr == nil {
			grpcReq.Actions = l
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

	response.Proto(w, http.StatusOK, resp)
}

func (ar *AutomationRoutes) HandleDeleteAutomation(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	automationID, ok := validateUUIDParam(w, r, "id")
	if !ok {
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

	automationID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.EnableAutomation(r.Context(), &automationv1.EnableAutomationRequest{
		AutomationId: automationID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (ar *AutomationRoutes) HandleDisableAutomation(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	automationID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.DisableAutomation(r.Context(), &automationv1.DisableAutomationRequest{
		AutomationId: automationID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
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

	automationID, ok := validateUUIDParam(w, r, "id")
	if !ok {
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

	response.Proto(w, http.StatusOK, resp)
}

func (ar *AutomationRoutes) HandleGetExecution(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	executionID, ok := validateUUIDParam(w, r, "executionId")
	if !ok {
		return
	}

	resp, err := client.GetExecution(r.Context(), &automationv1.GetExecutionRequest{
		ExecutionId: executionID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
}

type createFromTemplateRequest struct {
	Name string `json:"name" validate:"required"`
}

func (ar *AutomationRoutes) HandleCreateFromTemplate(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	templateID := chi.URLParam(r, "templateId")

	body, ok := decodeAndValidate[createFromTemplateRequest](w, r)
	if !ok {
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

	response.Proto(w, http.StatusCreated, resp)
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

	response.Proto(w, http.StatusOK, resp)
}

type dryRunRequest struct {
	AutomationID string                 `json:"automation_id" validate:"required,uuid"`
	SampleEnv    map[string]interface{} `json:"sample_env"`
}

func (ar *AutomationRoutes) HandleDryRun(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	body, ok := decodeAndValidate[dryRunRequest](w, r)
	if !ok {
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Webhook Trigger Handler
// ============================================================================

// HandleTriggerWebhook processes an inbound webhook delivery for the
// automation identified by the {automationId} path segment. Unauthenticated
// by design (external senders carry no JWT); signature verification and
// tenant resolution happen in workflow.Service.TriggerWebhook, reached here
// strictly through the gRPC client per the gateway's thin-handler rule.
func (ar *AutomationRoutes) HandleTriggerWebhook(w http.ResponseWriter, r *http.Request) {
	client, err := ar.getClient()
	if err != nil {
		respondServiceUnavailable(w, ar.ServiceName())
		return
	}

	automationID, ok := validateUUIDParam(w, r, "automationId")
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxWebhookBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		response.Error(w, http.StatusRequestEntityTooLarge, "webhook payload too large")
		return
	}

	resp, err := client.TriggerWebhook(r.Context(), &automationv1.TriggerWebhookRequest{
		AutomationId:   automationID,
		Body:           body,
		Signature:      r.Header.Get(webhookSignatureHeader),
		IdempotencyKey: r.Header.Get("Idempotency-Key"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusAccepted, map[string]bool{"duplicate": resp.GetDuplicate()})
}

// ============================================================================
// Helpers
// ============================================================================

// parseAutomationScope maps a scope filter/body value to its proto enum. An
// empty value keeps the historical default of SCOPE_PERSONAL, so callers
// that omit the field see no behavior change. ok is false only for a
// non-empty value that isn't personal/team/organization — the caller must
// reject those with 400 instead of silently narrowing to personal, which
// used to turn a typo like "organisation" into a plausible-looking wrong
// answer instead of a visible error.
func parseAutomationScope(s string) (automationv1.AutomationScope, bool) {
	switch s {
	case "", "personal":
		return automationv1.AutomationScope_SCOPE_PERSONAL, true
	case "team":
		return automationv1.AutomationScope_SCOPE_TEAM, true
	case "organization":
		return automationv1.AutomationScope_SCOPE_ORGANIZATION, true
	default:
		return automationv1.AutomationScope_SCOPE_PERSONAL, false
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

// rawJSONToAutomationList parses the actions field, which carries a JSON
// array (see models.ActionConfig on the server side), unlike trigger_config/
// conditions which carry a JSON object and go through rawJSONToAutomationStruct.
func rawJSONToAutomationList(data json.RawMessage) (*structpb.ListValue, error) {
	if len(data) == 0 {
		return nil, nil
	}
	l := &structpb.ListValue{}
	if err := l.UnmarshalJSON(data); err != nil {
		return nil, err
	}
	return l, nil
}
