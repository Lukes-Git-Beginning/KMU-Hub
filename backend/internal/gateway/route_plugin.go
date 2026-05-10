package gateway

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	pluginv1 "github.com/kmuhub/kmuhub/proto/plugin/v1"
)

// PluginRoutes handles HTTP routes for the Plugin service
type PluginRoutes struct {
	registry *ServiceRegistry
}

// NewPluginRoutes creates a new PluginRoutes
func NewPluginRoutes(registry *ServiceRegistry) *PluginRoutes {
	return &PluginRoutes{registry: registry}
}

// ServiceName returns "plugin"
func (pr *PluginRoutes) ServiceName() string { return "plugin" }

func (pr *PluginRoutes) getClient() (pluginv1.PluginServiceClient, error) {
	conn, err := pr.registry.GetConnection("plugin")
	if err != nil {
		return nil, err
	}
	return pluginv1.NewPluginServiceClient(conn), nil
}

// RegisterRoutes registers all plugin HTTP routes
func (pr *PluginRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/plugins", func(r chi.Router) {
		r.Use(authMiddleware)

		// Manifests (admin-only for mutations, read for all)
		r.Get("/manifests", pr.HandleListManifests)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("admin"))
			r.Post("/manifests", pr.HandleCreateManifest)
			r.Get("/manifests/{manifest_id}", pr.HandleGetManifest)
			r.Delete("/manifests/{manifest_id}", pr.HandleDeleteManifest)
		})

		// Installations
		r.Get("/installations", pr.HandleListInstallations)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("admin"))
			r.Post("/installations", pr.HandleInstallPlugin)
			r.Get("/installations/{installation_id}", pr.HandleGetInstallation)
			r.Post("/installations/{installation_id}/enable", pr.HandleEnablePlugin)
			r.Post("/installations/{installation_id}/disable", pr.HandleDisablePlugin)
			r.Delete("/installations/{installation_id}", pr.HandleUninstallPlugin)

			// Permissions
			r.Post("/installations/{installation_id}/permissions", pr.HandleApprovePermissions)
			r.Get("/installations/{installation_id}/permissions", pr.HandleListPermissions)

			// Settings
			r.Get("/installations/{installation_id}/settings", pr.HandleGetSettings)
			r.Put("/installations/{installation_id}/settings", pr.HandleUpdateSettings)
			r.Get("/installations/{installation_id}/settings/schema", pr.HandleGetSettingsSchema)
		})

		// Validation Rules (admin-only)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("admin"))
			r.Get("/validation-rules", pr.HandleListValidationRules)
			r.Post("/validation-rules", pr.HandleCreateValidationRule)
			r.Put("/validation-rules/{rule_id}", pr.HandleUpdateValidationRule)
			r.Delete("/validation-rules/{rule_id}", pr.HandleDeleteValidationRule)
		})

		// Workflow Rules (admin-only)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("admin"))
			r.Get("/workflow-rules", pr.HandleListWorkflowRules)
			r.Post("/workflow-rules", pr.HandleCreateWorkflowRule)
			r.Put("/workflow-rules/{rule_id}", pr.HandleUpdateWorkflowRule)
			r.Delete("/workflow-rules/{rule_id}", pr.HandleDeleteWorkflowRule)
		})

		// Industry Templates (read for all)
		r.Get("/templates", pr.HandleListTemplates)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("admin"))
			r.Post("/templates/{template_id}/apply", pr.HandleApplyTemplate)
		})

		// Execution Logs (admin-only)
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("admin"))
			r.Get("/execution-logs", pr.HandleListExecutionLogs)
		})
	})
}

// ============================================================================
// Manifest Handlers
// ============================================================================

func (pr *PluginRoutes) HandleListManifests(w http.ResponseWriter, r *http.Request) {
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	resp, err := client.ListManifests(r.Context(), &pluginv1.ListManifestsRequest{
		PluginType: r.URL.Query().Get("type"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp.GetManifests())
}

func (pr *PluginRoutes) HandleCreateManifest(w http.ResponseWriter, r *http.Request) {
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	var req pluginv1.CreateManifestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.CreateManifest(r.Context(), &req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp.GetManifest())
}

func (pr *PluginRoutes) HandleGetManifest(w http.ResponseWriter, r *http.Request) {
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	resp, err := client.GetManifest(r.Context(), &pluginv1.GetManifestRequest{
		ManifestId: chi.URLParam(r, "manifest_id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp.GetManifest())
}

func (pr *PluginRoutes) HandleDeleteManifest(w http.ResponseWriter, r *http.Request) {
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	_, err = client.DeleteManifest(r.Context(), &pluginv1.DeleteManifestRequest{
		ManifestId: chi.URLParam(r, "manifest_id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]bool{"success": true})
}

// ============================================================================
// Installation Handlers
// ============================================================================

func (pr *PluginRoutes) HandleListInstallations(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant context")
		return
	}

	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	resp, err := client.ListInstallations(r.Context(), &pluginv1.ListInstallationsRequest{
		TenantId: tenantID.String(),
		Status:   r.URL.Query().Get("status"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp.GetInstallations())
}

func (pr *PluginRoutes) HandleInstallPlugin(w http.ResponseWriter, r *http.Request) {
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	var req pluginv1.InstallPluginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.InstallPlugin(r.Context(), &req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp.GetInstallation())
}

func (pr *PluginRoutes) HandleGetInstallation(w http.ResponseWriter, r *http.Request) {
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	resp, err := client.GetInstallation(r.Context(), &pluginv1.GetInstallationRequest{
		InstallationId: chi.URLParam(r, "installation_id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp.GetInstallation())
}

func (pr *PluginRoutes) HandleEnablePlugin(w http.ResponseWriter, r *http.Request) {
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	resp, err := client.EnablePlugin(r.Context(), &pluginv1.EnablePluginRequest{
		InstallationId: chi.URLParam(r, "installation_id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp.GetInstallation())
}

func (pr *PluginRoutes) HandleDisablePlugin(w http.ResponseWriter, r *http.Request) {
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	resp, err := client.DisablePlugin(r.Context(), &pluginv1.DisablePluginRequest{
		InstallationId: chi.URLParam(r, "installation_id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp.GetInstallation())
}

func (pr *PluginRoutes) HandleUninstallPlugin(w http.ResponseWriter, r *http.Request) {
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	_, err = client.UninstallPlugin(r.Context(), &pluginv1.UninstallPluginRequest{
		InstallationId: chi.URLParam(r, "installation_id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]bool{"success": true})
}

// ============================================================================
// Permission Handlers
// ============================================================================

func (pr *PluginRoutes) HandleApprovePermissions(w http.ResponseWriter, r *http.Request) {
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	var body struct {
		Permissions []string `json:"permissions"`
		GrantedBy   string   `json:"granted_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	_, err = client.ApprovePermissions(r.Context(), &pluginv1.ApprovePermissionsRequest{
		InstallationId: chi.URLParam(r, "installation_id"),
		Permissions:    body.Permissions,
		GrantedBy:      body.GrantedBy,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (pr *PluginRoutes) HandleListPermissions(w http.ResponseWriter, r *http.Request) {
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	resp, err := client.ListGrantedPermissions(r.Context(), &pluginv1.ListGrantedPermissionsRequest{
		InstallationId: chi.URLParam(r, "installation_id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp.GetPermissions())
}

// ============================================================================
// Settings Handlers
// ============================================================================

func (pr *PluginRoutes) HandleGetSettings(w http.ResponseWriter, r *http.Request) {
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	resp, err := client.GetPluginSettings(r.Context(), &pluginv1.GetPluginSettingsRequest{
		InstallationId: chi.URLParam(r, "installation_id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(resp.GetSettings()))
}

func (pr *PluginRoutes) HandleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	var body json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	_, err = client.UpdatePluginSettings(r.Context(), &pluginv1.UpdatePluginSettingsRequest{
		InstallationId: chi.URLParam(r, "installation_id"),
		Settings:       string(body),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (pr *PluginRoutes) HandleGetSettingsSchema(w http.ResponseWriter, r *http.Request) {
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	resp, err := client.GetPluginSettingsSchema(r.Context(), &pluginv1.GetPluginSettingsSchemaRequest{
		InstallationId: chi.URLParam(r, "installation_id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(resp.GetSchema()))
}

// ============================================================================
// Validation Rule Handlers
// ============================================================================

func (pr *PluginRoutes) HandleListValidationRules(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant context")
		return
	}

	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	resp, err := client.ListValidationRules(r.Context(), &pluginv1.ListValidationRulesRequest{
		TenantId:   tenantID.String(),
		EntityType: r.URL.Query().Get("entity_type"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp.GetRules())
}

func (pr *PluginRoutes) HandleCreateValidationRule(w http.ResponseWriter, r *http.Request) {
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	var req pluginv1.CreateValidationRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.CreateValidationRule(r.Context(), &req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp.GetRule())
}

func (pr *PluginRoutes) HandleUpdateValidationRule(w http.ResponseWriter, r *http.Request) {
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	var req pluginv1.UpdateValidationRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.RuleId = chi.URLParam(r, "rule_id")

	resp, err := client.UpdateValidationRule(r.Context(), &req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp.GetRule())
}

func (pr *PluginRoutes) HandleDeleteValidationRule(w http.ResponseWriter, r *http.Request) {
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	_, err = client.DeleteValidationRule(r.Context(), &pluginv1.DeleteValidationRuleRequest{
		RuleId: chi.URLParam(r, "rule_id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]bool{"success": true})
}

// ============================================================================
// Workflow Rule Handlers
// ============================================================================

func (pr *PluginRoutes) HandleListWorkflowRules(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant context")
		return
	}

	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	resp, err := client.ListWorkflowRules(r.Context(), &pluginv1.ListWorkflowRulesRequest{
		TenantId:     tenantID.String(),
		TriggerEvent: r.URL.Query().Get("trigger_event"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp.GetRules())
}

func (pr *PluginRoutes) HandleCreateWorkflowRule(w http.ResponseWriter, r *http.Request) {
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	var req pluginv1.CreateWorkflowRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.CreateWorkflowRule(r.Context(), &req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp.GetRule())
}

func (pr *PluginRoutes) HandleUpdateWorkflowRule(w http.ResponseWriter, r *http.Request) {
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	var req pluginv1.UpdateWorkflowRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.RuleId = chi.URLParam(r, "rule_id")

	resp, err := client.UpdateWorkflowRule(r.Context(), &req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp.GetRule())
}

func (pr *PluginRoutes) HandleDeleteWorkflowRule(w http.ResponseWriter, r *http.Request) {
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	_, err = client.DeleteWorkflowRule(r.Context(), &pluginv1.DeleteWorkflowRuleRequest{
		RuleId: chi.URLParam(r, "rule_id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]bool{"success": true})
}

// ============================================================================
// Template Handlers
// ============================================================================

func (pr *PluginRoutes) HandleListTemplates(w http.ResponseWriter, r *http.Request) {
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	resp, err := client.ListIndustryTemplates(r.Context(), &pluginv1.ListIndustryTemplatesRequest{
		Industry: r.URL.Query().Get("industry"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp.GetTemplates())
}

func (pr *PluginRoutes) HandleApplyTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant context")
		return
	}

	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	var body struct {
		AppliedBy string `json:"applied_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.ApplyIndustryTemplate(r.Context(), &pluginv1.ApplyIndustryTemplateRequest{
		TenantId:   tenantID.String(),
		TemplateId: chi.URLParam(r, "template_id"),
		AppliedBy:  body.AppliedBy,
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

func (pr *PluginRoutes) HandleListExecutionLogs(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant context")
		return
	}

	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	limit := int32(100)
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, parseErr := parseIntParam(l); parseErr == nil {
			limit = int32(parsed)
		}
	}

	resp, err := client.ListExecutionLogs(r.Context(), &pluginv1.ListExecutionLogsRequest{
		TenantId:       tenantID.String(),
		InstallationId: r.URL.Query().Get("installation_id"),
		Limit:          limit,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp.GetLogs())
}

func parseIntParam(s string) (int, error) {
	return strconv.Atoi(s)
}
