package gateway

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/kmuhub/kmuhub/internal/featureflag"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/modules"
	"github.com/kmuhub/kmuhub/internal/server/response"
	settingsv1 "github.com/kmuhub/kmuhub/proto/settings/v1"
)

// SettingsRoutes handles HTTP routes for the Settings service.
// Settings runs in the same binary as auth (port :50051), so we reuse the "auth"
// gRPC connection — identical pattern to SecurityRoutes.
type SettingsRoutes struct {
	registry *ServiceRegistry
	// flags resolves the deployment-wide modules.* flags for the licence
	// endpoints. The settings service deliberately does not see them: they
	// describe what this deployment runs, not what the tenant booked.
	flags *featureflag.Registry
}

// NewSettingsRoutes creates a new SettingsRoutes with the given service registry
// and feature flags.
func NewSettingsRoutes(registry *ServiceRegistry, flags *featureflag.Registry) *SettingsRoutes {
	return &SettingsRoutes{registry: registry, flags: flags}
}

// ServiceName returns "auth" because the Settings gRPC server is co-located
// with the auth service on port :50051.
func (sr *SettingsRoutes) ServiceName() string { return "auth" }

// getSettingsClient lazily obtains a gRPC client for the Settings service.
func (sr *SettingsRoutes) getSettingsClient() (settingsv1.SettingsServiceClient, error) {
	conn, err := sr.registry.GetConnection("auth")
	if err != nil {
		return nil, err
	}
	return settingsv1.NewSettingsServiceClient(conn), nil
}

// RegisterRoutes registers all settings and module-lead HTTP routes.
//
// Routes summary:
//
//	GET    /api/v1/tenant/module-leads                          — list all leads (?user_id=, ?module_id=)
//	GET    /api/v1/tenant/module-leads/me                       — my lead modules
//	PUT    /api/v1/tenant/module-leads/{user_id}/{module_id}    — grant (admin, module-leads:write)
//	DELETE /api/v1/tenant/module-leads/{user_id}/{module_id}    — revoke (admin, module-leads:write)
//
//	GET    /api/v1/settings/{module_id}                         — resolved (user > tenant)
//	GET    /api/v1/settings/{module_id}/tenant                  — tenant-scope raw
//	PUT    /api/v1/settings/{module_id}/tenant                  — update tenant-scope (lead/admin)
//	GET    /api/v1/settings/{module_id}/user                    — my user-scope raw
//	PUT    /api/v1/settings/{module_id}/user                    — update my user-scope
//
//	GET    /api/v1/admin/branding                                — tenant branding (name/logo/icon/accent)
//	PUT    /api/v1/admin/branding                                — replace tenant branding (admin)
func (sr *SettingsRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/tenant/module-leads", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("module-leads", "read")).Get("/", sr.HandleListModuleLeads)
		r.With(middleware.RequirePermission("module-leads", "read")).Get("/me", sr.HandleGetMyModuleLeads)
		r.With(middleware.RequirePermission("module-leads", "write")).Put("/{user_id}/{module_id}", sr.HandleGrantModuleLead)
		r.With(middleware.RequirePermission("module-leads", "write")).Delete("/{user_id}/{module_id}", sr.HandleRevokeModuleLead)
	})

	r.Route("/api/v1/tenant/module-grants", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("module-grants", "read")).Get("/", sr.HandleListModuleGrants)
		r.With(middleware.RequirePermission("module-grants", "write")).Post("/bulk-revoke", sr.HandleBulkRevokeModuleAccess)
		r.With(middleware.RequirePermission("module-grants", "write")).Put("/{user_id}/{module_id}", sr.HandleGrantModuleAccess)
		r.With(middleware.RequirePermission("module-grants", "write")).Delete("/{user_id}/{module_id}", sr.HandleRevokeModuleAccess)
	})

	r.Route("/api/v1/admin/license", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("license", "read")).Get("/", sr.HandleGetTenantLicense)
		r.With(middleware.RequirePermission("license", "write")).Patch("/", sr.HandleSetTenantModuleActive)
	})

	r.Route("/api/v1/admin/subscription", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("license", "read")).Get("/", sr.HandleGetTenantSubscription)
	})

	r.Route("/api/v1/admin/branding", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("settings", "read")).Get("/", sr.HandleGetBranding)
		r.With(middleware.RequirePermission("settings", "write")).Put("/", sr.HandlePutBranding)
	})

	r.Route("/api/v1/settings/{module_id}", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("settings", "read")).Get("/", sr.HandleGetResolvedSettings)
		r.With(middleware.RequirePermission("settings", "read")).Get("/tenant", sr.HandleGetTenantSettings)
		r.With(middleware.RequirePermission("settings", "write")).Put("/tenant", sr.HandlePutTenantSettings)
		r.With(middleware.RequirePermission("settings", "read")).Get("/user", sr.HandleGetUserSettings)
		r.With(middleware.RequirePermission("settings", "write")).Put("/user", sr.HandlePutUserSettings)
	})

	r.Route("/api/v1/users/preferences", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("settings", "read")).Get("/", sr.HandleGetUserPreferences)
		r.With(middleware.RequirePermission("settings", "write")).Put("/", sr.HandlePutUserPreferences)
	})
}

// ============================================================================
// Module-lead handlers
// ============================================================================

// HandleListModuleLeads lists module-lead assignments.
// Optional query params: user_id (UUID), module_id (string).
func (sr *SettingsRoutes) HandleListModuleLeads(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	req := &settingsv1.ListModuleLeadsRequest{TenantId: tenantID.String()}
	if uid := r.URL.Query().Get("user_id"); uid != "" {
		req.UserId = &uid
	}
	if mid := r.URL.Query().Get("module_id"); mid != "" {
		req.ModuleId = &mid
	}

	resp, err := client.ListModuleLeads(r.Context(), req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

// HandleGetMyModuleLeads returns the module IDs the caller leads.
// FE: used by useIsModuleLead hook to hydrate the moduleLeads store.
func (sr *SettingsRoutes) HandleGetMyModuleLeads(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	resp, err := client.GetMyModuleLeads(r.Context(), &settingsv1.GetMyModuleLeadsRequest{
		TenantId: tenantID.String(),
		UserId:   userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

// HandleGrantModuleLead grants module-lead rights to a user.
// Guard: RequirePermission("module-leads","write") — admin only via migration seed.
func (sr *SettingsRoutes) HandleGrantModuleLead(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	callerID := middleware.GetUserID(r.Context())
	if callerID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	targetUserID, ok := validateUUIDParam(w, r, "user_id")
	if !ok {
		return
	}
	moduleID := chi.URLParam(r, "module_id")
	if moduleID == "" {
		response.Error(w, http.StatusBadRequest, "module_id is required")
		return
	}

	resp, err := client.GrantModuleLead(r.Context(), &settingsv1.GrantModuleLeadRequest{
		TenantId:  tenantID.String(),
		UserId:    targetUserID,
		ModuleId:  moduleID,
		GrantedBy: callerID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

// HandleRevokeModuleLead revokes module-lead rights.
// Guard: RequirePermission("module-leads","write") — admin only via migration seed.
func (sr *SettingsRoutes) HandleRevokeModuleLead(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	targetUserID, ok := validateUUIDParam(w, r, "user_id")
	if !ok {
		return
	}
	moduleID := chi.URLParam(r, "module_id")
	if moduleID == "" {
		response.Error(w, http.StatusBadRequest, "module_id is required")
		return
	}

	_, err = client.RevokeModuleLead(r.Context(), &settingsv1.RevokeModuleLeadRequest{
		TenantId: tenantID.String(),
		UserId:   targetUserID,
		ModuleId: moduleID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
}

// ============================================================================
// Module-access grant handlers
// ============================================================================

// userModuleGrantJSON is the flat shape the ModuleAssignmentTab consumes
// (lib/pricing.ts UserModuleGrant). Timestamps are RFC3339 strings and field
// names are camelCase, so we transform the proto rather than passing it through.
type userModuleGrantJSON struct {
	UserID       string  `json:"userId"`
	UserName     string  `json:"userName"`
	ModuleID     string  `json:"moduleId"`
	GrantedAt    string  `json:"grantedAt"`
	LastActiveAt *string `json:"lastActiveAt"`
}

func toUserModuleGrantJSON(g *settingsv1.UserModuleGrant) userModuleGrantJSON {
	out := userModuleGrantJSON{
		UserID:    g.GetUserId(),
		UserName:  g.GetUserName(),
		ModuleID:  g.GetModuleId(),
		GrantedAt: g.GetGrantedAt().AsTime().Format(time.RFC3339),
	}
	if g.LastActiveAt != nil {
		s := g.GetLastActiveAt().AsTime().Format(time.RFC3339)
		out.LastActiveAt = &s
	}
	return out
}

// HandleListModuleGrants returns all module-access grants for the tenant.
func (sr *SettingsRoutes) HandleListModuleGrants(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	req := &settingsv1.ListModuleGrantsRequest{TenantId: tenantID.String()}
	if uid := r.URL.Query().Get("user_id"); uid != "" {
		req.UserId = &uid
	}
	if mid := r.URL.Query().Get("module_id"); mid != "" {
		req.ModuleId = &mid
	}

	resp, err := client.ListModuleGrants(r.Context(), req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	grants := make([]userModuleGrantJSON, 0, len(resp.GetGrants()))
	for _, g := range resp.GetGrants() {
		grants = append(grants, toUserModuleGrantJSON(g))
	}
	response.JSON(w, http.StatusOK, map[string]any{"grants": grants})
}

// HandleGrantModuleAccess grants a user access to a module (admin only).
func (sr *SettingsRoutes) HandleGrantModuleAccess(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	callerID := middleware.GetUserID(r.Context())
	if callerID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	targetUserID, ok := validateUUIDParam(w, r, "user_id")
	if !ok {
		return
	}
	moduleID := chi.URLParam(r, "module_id")
	if moduleID == "" {
		response.Error(w, http.StatusBadRequest, "module_id is required")
		return
	}

	resp, err := client.GrantModuleAccess(r.Context(), &settingsv1.GrantModuleAccessRequest{
		TenantId:  tenantID.String(),
		UserId:    targetUserID,
		ModuleId:  moduleID,
		GrantedBy: callerID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, toUserModuleGrantJSON(resp))
}

// HandleRevokeModuleAccess revokes a user's access to a module (admin only).
func (sr *SettingsRoutes) HandleRevokeModuleAccess(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	targetUserID, ok := validateUUIDParam(w, r, "user_id")
	if !ok {
		return
	}
	moduleID := chi.URLParam(r, "module_id")
	if moduleID == "" {
		response.Error(w, http.StatusBadRequest, "module_id is required")
		return
	}

	_, err = client.RevokeModuleAccess(r.Context(), &settingsv1.RevokeModuleAccessRequest{
		TenantId: tenantID.String(),
		UserId:   targetUserID,
		ModuleId: moduleID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
}

// bulkRevokeGrantsRequest is the HTTP body for POST /module-grants/bulk-revoke.
type bulkRevokeGrantsRequest struct {
	Pairs []struct {
		UserID   string `json:"userId" validate:"required,uuid"`
		ModuleID string `json:"moduleId" validate:"required"`
	} `json:"pairs" validate:"required"`
}

// HandleBulkRevokeModuleAccess revokes multiple grants in one call (admin only).
func (sr *SettingsRoutes) HandleBulkRevokeModuleAccess(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	req, ok := decodeAndValidate[bulkRevokeGrantsRequest](w, r)
	if !ok {
		return
	}

	refs := make([]*settingsv1.ModuleGrantRef, 0, len(req.Pairs))
	for _, p := range req.Pairs {
		refs = append(refs, &settingsv1.ModuleGrantRef{UserId: p.UserID, ModuleId: p.ModuleID})
	}

	resp, err := client.BulkRevokeModuleAccess(r.Context(), &settingsv1.BulkRevokeModuleAccessRequest{
		TenantId: tenantID.String(),
		Refs:     refs,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"revokedCount": resp.GetRevokedCount()})
}

// ============================================================================
// Settings handlers
// ============================================================================

// HandleGetResolvedSettings returns merged settings (user wins over tenant).
func (sr *SettingsRoutes) HandleGetResolvedSettings(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}
	moduleID := chi.URLParam(r, "module_id")

	resp, err := client.GetResolvedSettings(r.Context(), &settingsv1.GetResolvedSettingsRequest{
		TenantId: tenantID.String(),
		UserId:   userID,
		ModuleId: moduleID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

// HandleGetTenantSettings returns raw tenant-scope settings for a module.
func (sr *SettingsRoutes) HandleGetTenantSettings(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	moduleID := chi.URLParam(r, "module_id")

	resp, err := client.GetTenantSettings(r.Context(), &settingsv1.GetTenantSettingsRequest{
		TenantId: tenantID.String(),
		ModuleId: moduleID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

// putSettingsRequest is the HTTP request body for PUT /settings/{module_id}/tenant|user.
// Using a flat map gives a friendlier API surface: {"defaultView":"week","workStartHour":8}
// instead of the array-of-entries proto format.
type putSettingsRequest struct {
	Settings map[string]json.RawMessage `json:"settings" validate:"required"`
}

// HandlePutTenantSettings updates tenant-scope settings.
// Service-level RBAC: caller must be admin OR module-lead for this module.
func (sr *SettingsRoutes) HandlePutTenantSettings(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	callerID := middleware.GetUserID(r.Context())
	if callerID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}
	moduleID := chi.URLParam(r, "module_id")

	req, ok := decodeAndValidate[putSettingsRequest](w, r)
	if !ok {
		return
	}

	entries, err := rawMapToSettingEntries(req.Settings)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid settings value: "+err.Error())
		return
	}

	resp, err := client.PutTenantSettings(r.Context(), &settingsv1.PutTenantSettingsRequest{
		TenantId:  tenantID.String(),
		ModuleId:  moduleID,
		UpdatedBy: callerID,
		Entries:   entries,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

// HandleGetUserSettings returns raw user-scope settings.
func (sr *SettingsRoutes) HandleGetUserSettings(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}
	moduleID := chi.URLParam(r, "module_id")

	resp, err := client.GetUserSettings(r.Context(), &settingsv1.GetUserSettingsRequest{
		TenantId: tenantID.String(),
		UserId:   userID,
		ModuleId: moduleID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

// HandlePutUserSettings updates the caller's user-scope settings (own user only).
func (sr *SettingsRoutes) HandlePutUserSettings(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}
	moduleID := chi.URLParam(r, "module_id")

	req, ok := decodeAndValidate[putSettingsRequest](w, r)
	if !ok {
		return
	}

	entries, err := rawMapToSettingEntries(req.Settings)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid settings value: "+err.Error())
		return
	}

	resp, err := client.PutUserSettings(r.Context(), &settingsv1.PutUserSettingsRequest{
		TenantId: tenantID.String(),
		UserId:   userID,
		ModuleId: moduleID,
		Entries:  entries,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// User preferences handlers
// ============================================================================
//
// GET/PUT /api/v1/users/preferences is a thin, fixed-key wrapper around the
// existing user_settings table (migration 000138) — no dedicated table, no
// new endpoint family. It differs from GET/PUT /settings/{module_id}/user in
// two ways: it is scoped to the fixed module_id "profile", and it rejects
// unknown keys instead of accepting arbitrary JSONB.

// userPreferencesModuleID is the fixed module_id user_settings rows are
// stored under for this endpoint.
const userPreferencesModuleID = "profile"

// userPreferenceKeys is the allowlist for PUT /users/preferences.
var userPreferenceKeys = map[string]bool{
	"language": true,
	"theme":    true,
	"region":   true,
}

// HandleGetUserPreferences returns the caller's user-wide preferences
// (language/theme/region), stored under module_id "profile".
func (sr *SettingsRoutes) HandleGetUserPreferences(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	resp, err := client.GetUserSettings(r.Context(), &settingsv1.GetUserSettingsRequest{
		TenantId: tenantID.String(),
		UserId:   userID,
		ModuleId: userPreferencesModuleID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

// putUserPreferencesRequest is the HTTP body for PUT /users/preferences.
// Full-replace semantics: whatever keys are present become the caller's
// complete preference set — a previously stored key absent here is deleted,
// not left untouched (unlike PUT /settings/{module_id}/user).
type putUserPreferencesRequest struct {
	Preferences map[string]json.RawMessage `json:"preferences" validate:"required"`
}

// HandlePutUserPreferences replaces the caller's user-wide preferences.
// Any user_id present in the body is ignored — the target is always the
// caller from the auth context, never a body-supplied value.
func (sr *SettingsRoutes) HandlePutUserPreferences(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	req, ok := decodeAndValidate[putUserPreferencesRequest](w, r)
	if !ok {
		return
	}

	for key := range req.Preferences {
		if !userPreferenceKeys[key] {
			response.Error(w, http.StatusBadRequest, "unknown preference key: "+key)
			return
		}
	}

	entries, err := rawMapToSettingEntries(req.Preferences)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid preference value: "+err.Error())
		return
	}

	resp, err := client.ReplaceUserSettings(r.Context(), &settingsv1.ReplaceUserSettingsRequest{
		TenantId: tenantID.String(),
		UserId:   userID,
		ModuleId: userPreferencesModuleID,
		Entries:  entries,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Conversion helpers (HTTP ↔ proto)
// ============================================================================

// rawMapToSettingEntries converts a map[string]json.RawMessage from the HTTP
// request body into proto SettingEntry slice. Each raw JSON value is unmarshaled
// into a structpb.Value so the gRPC layer can forward it as JSONB.
func rawMapToSettingEntries(m map[string]json.RawMessage) ([]*settingsv1.SettingEntry, error) {
	entries := make([]*settingsv1.SettingEntry, 0, len(m))
	for k, raw := range m {
		var anyVal interface{}
		if err := json.Unmarshal(raw, &anyVal); err != nil {
			return nil, err
		}
		protoVal, err := structpb.NewValue(anyVal)
		if err != nil {
			return nil, err
		}
		entries = append(entries, &settingsv1.SettingEntry{
			Key:   k,
			Value: protoVal,
		})
	}
	return entries, nil
}

// ============================================================================
// Branding handlers
// ============================================================================

// putBrandingRequest is the HTTP body for PUT /admin/branding. It is a full
// replace, not a patch: logo_object_key/icon_object_key omitted or "" clears
// the field. accent_color is required and validated against the Cosmi swatch
// palette server-side (settings.PutBranding), never a free hex value.
type putBrandingRequest struct {
	Name          string `json:"name" validate:"required,max=200"`
	LogoObjectKey string `json:"logo_object_key"`
	IconObjectKey string `json:"icon_object_key"`
	AccentColor   string `json:"accent_color" validate:"required"`
}

// HandleGetBranding returns the caller's tenant branding. Logo/icon are
// object keys, not URLs — the frontend resolves them via the existing
// POST /api/v1/files/presign-download (same pattern as User.avatar_url).
func (sr *SettingsRoutes) HandleGetBranding(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	resp, err := client.GetBranding(r.Context(), &settingsv1.GetBrandingRequest{
		TenantId: tenantID.String(),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

// HandlePutBranding replaces the caller's tenant branding. Guard only checks
// settings:write; the service enforces admin-or-module-lead, which resolves
// to admin-only here since no UI ever grants module-lead for "branding".
func (sr *SettingsRoutes) HandlePutBranding(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	callerID := middleware.GetUserID(r.Context())
	if callerID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	req, ok := decodeAndValidate[putBrandingRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.PutBranding(r.Context(), &settingsv1.PutBrandingRequest{
		TenantId:  tenantID.String(),
		UpdatedBy: callerID,
		Branding: &settingsv1.Branding{
			Name:          req.Name,
			LogoObjectKey: req.LogoObjectKey,
			IconObjectKey: req.IconObjectKey,
			AccentColor:   req.AccentColor,
		},
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Licensing handlers
// ============================================================================

// tenantModuleJSON is the shape the licence tab consumes (TenantModule in
// desktop/src/renderer/src/api/admin-types.ts). flag_key stays server-side.
type tenantModuleJSON struct {
	ModuleID      string `json:"moduleId"`
	Group         string `json:"group"`
	Active        bool   `json:"active"`
	AssignedSeats int32  `json:"assignedSeats"`
}

// tenantSubscriptionJSON mirrors MockTenantData in api/hooks/useBilling.ts.
// billingPeriodEnd and totalSeats are null when unset resp. unlimited — the FE
// reads both with a `??` fallback.
type tenantSubscriptionJSON struct {
	PlanType         string  `json:"planType"`
	SupportTier      string  `json:"supportTier"`
	Status           string  `json:"status"`
	BillingPeriodEnd *string `json:"billingPeriodEnd"`
	TotalSeats       *int32  `json:"totalSeats"`
}

// moduleAvailable reports whether this deployment runs the module at all. A
// module the deployment does not run cannot be active for any tenant, however
// the activation table reads — the flag is the upper bound.
func (sr *SettingsRoutes) moduleAvailable(moduleID string) bool {
	flagKey := modules.FlagKey(moduleID)
	return flagKey == "" || sr.flags.IsEnabled(flagKey)
}

func (sr *SettingsRoutes) toTenantModuleJSON(m *settingsv1.TenantModule) tenantModuleJSON {
	active := m.GetActive() && sr.moduleAvailable(m.GetModuleId())
	seats := m.GetAssignedSeats()
	if !active {
		seats = 0
	}
	return tenantModuleJSON{
		ModuleID:      m.GetModuleId(),
		Group:         m.GetGroup(),
		Active:        active,
		AssignedSeats: seats,
	}
}

// HandleGetTenantLicense lists every catalogue module with its tenant-wide
// activation state.
func (sr *SettingsRoutes) HandleGetTenantLicense(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	resp, err := client.GetTenantLicense(r.Context(), &settingsv1.GetTenantLicenseRequest{
		TenantId: tenantID.String(),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	out := make([]tenantModuleJSON, 0, len(resp.GetModules()))
	for _, m := range resp.GetModules() {
		out = append(out, sr.toTenantModuleJSON(m))
	}
	response.JSON(w, http.StatusOK, map[string]any{"modules": out})
}

// setTenantModuleRequest is the HTTP body for PATCH /admin/license.
type setTenantModuleRequest struct {
	ModuleID string `json:"moduleId"`
	Active   *bool  `json:"active"`
}

// HandleSetTenantModuleActive activates or deactivates one module tenant-wide.
func (sr *SettingsRoutes) HandleSetTenantModuleActive(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	callerID := middleware.GetUserID(r.Context())
	if callerID == "" {
		response.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req setTenantModuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Active == nil {
		response.Error(w, http.StatusBadRequest, "active is required")
		return
	}
	if _, ok := modules.Get(req.ModuleID); !ok {
		response.Error(w, http.StatusNotFound, "unknown module")
		return
	}
	// Activating a module this deployment does not run would store a row the
	// GET immediately masks back to inactive — say so instead.
	if *req.Active && !sr.moduleAvailable(req.ModuleID) {
		response.Error(w, http.StatusConflict, "module is not available in this deployment")
		return
	}

	resp, err := client.SetTenantModuleActive(r.Context(), &settingsv1.SetTenantModuleActiveRequest{
		TenantId:  tenantID.String(),
		ModuleId:  req.ModuleID,
		Active:    *req.Active,
		UpdatedBy: callerID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"module": sr.toTenantModuleJSON(resp)})
}

// HandleGetTenantSubscription returns the booked plan of the caller's tenant.
func (sr *SettingsRoutes) HandleGetTenantSubscription(w http.ResponseWriter, r *http.Request) {
	client, err := sr.getSettingsClient()
	if err != nil {
		respondServiceUnavailable(w, sr.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	resp, err := client.GetTenantSubscription(r.Context(), &settingsv1.GetTenantSubscriptionRequest{
		TenantId: tenantID.String(),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	out := tenantSubscriptionJSON{
		PlanType:    resp.GetPlanType(),
		SupportTier: resp.GetSupportTier(),
		Status:      resp.GetStatus(),
	}
	if resp.BillingPeriodEnd != nil {
		end := resp.GetBillingPeriodEnd()
		out.BillingPeriodEnd = &end
	}
	if resp.TotalSeats != nil {
		seats := resp.GetTotalSeats()
		out.TotalSeats = &seats
	}
	response.JSON(w, http.StatusOK, map[string]any{"subscription": out})
}
