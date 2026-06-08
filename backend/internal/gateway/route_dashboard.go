package gateway

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
)

// DashboardRoutes handles HTTP routes for dashboard layout management.
// Unlike other route registrars that proxy to gRPC services, this one
// operates directly against the gateway's PostgreSQL pool via a local service.
type DashboardRoutes struct {
	service *DashboardService
}

// NewDashboardRoutes creates a new DashboardRoutes.
func NewDashboardRoutes(service *DashboardService) *DashboardRoutes {
	return &DashboardRoutes{service: service}
}

// ServiceName returns the backend service name.
func (d *DashboardRoutes) ServiceName() string { return "dashboard" }

// RegisterRoutes registers all dashboard HTTP routes.
func (d *DashboardRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/dashboard", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(RequireAuthenticated)

		// User layout endpoints
		r.Get("/layout", d.HandleGetDashboard)
		r.Put("/layout", d.HandleSaveDashboard)
		r.Delete("/layout", d.HandleResetToDefaults)

		// Admin-only role default endpoints
		r.With(middleware.RequireRole("admin")).Get("/defaults/{role}", d.HandleGetDefaults)
		r.With(middleware.RequireRole("admin")).Put("/defaults/{role}", d.HandleSaveDefaults)
	})
}

// ============================================================================
// User Layout Handlers
// ============================================================================

// HandleGetDashboard returns the effective dashboard layout for the authenticated user.
func (d *DashboardRoutes) HandleGetDashboard(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID := middleware.GetUserID(r.Context())

	// Determine user's primary role for default fallback
	role := primaryRole(middleware.GetUserRoles(r.Context()))

	layout, err := d.service.GetDashboard(r.Context(), tenantID, userID, role)
	if err != nil {
		slog.Error("failed to get dashboard",
			"user_id", userID,
			"error", err,
		)
		response.Error(w, http.StatusInternalServerError, "failed to get dashboard layout")
		return
	}

	response.JSON(w, http.StatusOK, layout)
}

// saveDashboardRequest is the request body for saving a user's dashboard layout.
type saveDashboardRequest struct {
	Layout        json.RawMessage `json:"layout"`
	ActiveWidgets json.RawMessage `json:"active_widgets"`
}

// HandleSaveDashboard saves the authenticated user's personal dashboard layout.
func (d *DashboardRoutes) HandleSaveDashboard(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID := middleware.GetUserID(r.Context())

	// decodeAndValidate handles JSON decode; json.RawMessage fields carry no validate tags.
	req, ok := decodeAndValidate[saveDashboardRequest](w, r)
	if !ok {
		return
	}

	// Content validation for json.RawMessage fields cannot use struct tags — keep as code.
	if len(req.Layout) == 0 || len(req.ActiveWidgets) == 0 {
		response.Error(w, http.StatusBadRequest, "layout and active_widgets are required")
		return
	}
	if !json.Valid(req.Layout) || !json.Valid(req.ActiveWidgets) {
		response.Error(w, http.StatusBadRequest, "layout and active_widgets must be valid JSON")
		return
	}

	layout, err := d.service.SaveDashboard(r.Context(), tenantID, userID, req.Layout, req.ActiveWidgets)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to save dashboard layout")
		return
	}

	response.JSON(w, http.StatusOK, layout)
}

// HandleResetToDefaults removes the user's personal layout override.
func (d *DashboardRoutes) HandleResetToDefaults(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	userID := middleware.GetUserID(r.Context())

	if err := d.service.ResetToDefaults(r.Context(), tenantID, userID); err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to reset dashboard layout")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "reset to defaults"})
}

// ============================================================================
// Admin Default Handlers
// ============================================================================

// HandleGetDefaults returns the default layout for a given role.
func (d *DashboardRoutes) HandleGetDefaults(w http.ResponseWriter, r *http.Request) {
	role := chi.URLParam(r, "role")
	if !isValidRole(role) {
		response.Error(w, http.StatusBadRequest, "invalid role: must be admin, manager, or member")
		return
	}

	def, err := d.service.GetDefaults(r.Context(), role)
	if err != nil {
		if errors.Is(err, ErrDashboardNotFound) {
			response.Error(w, http.StatusNotFound, "no default layout for role: "+role)
			return
		}
		response.Error(w, http.StatusInternalServerError, "failed to get default layout")
		return
	}

	response.JSON(w, http.StatusOK, def)
}

// saveDefaultsRequest is the request body for saving role default layouts.
type saveDefaultsRequest struct {
	Layout        json.RawMessage `json:"layout"`
	ActiveWidgets json.RawMessage `json:"active_widgets"`
}

// HandleSaveDefaults saves the default layout for a given role.
func (d *DashboardRoutes) HandleSaveDefaults(w http.ResponseWriter, r *http.Request) {
	role := chi.URLParam(r, "role")
	if !isValidRole(role) {
		response.Error(w, http.StatusBadRequest, "invalid role: must be admin, manager, or member")
		return
	}

	// decodeAndValidate handles JSON decode; json.RawMessage fields carry no validate tags.
	req, ok := decodeAndValidate[saveDefaultsRequest](w, r)
	if !ok {
		return
	}

	// Content validation for json.RawMessage fields cannot use struct tags — keep as code.
	if len(req.Layout) == 0 || len(req.ActiveWidgets) == 0 {
		response.Error(w, http.StatusBadRequest, "layout and active_widgets are required")
		return
	}
	if !json.Valid(req.Layout) || !json.Valid(req.ActiveWidgets) {
		response.Error(w, http.StatusBadRequest, "layout and active_widgets must be valid JSON")
		return
	}

	def, err := d.service.SaveDefaults(r.Context(), role, req.Layout, req.ActiveWidgets)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to save default layout")
		return
	}

	response.JSON(w, http.StatusOK, def)
}

// ============================================================================
// Helpers
// ============================================================================

// validRoles is the set of valid dashboard role keys.
var validRoles = map[string]bool{
	"admin":   true,
	"manager": true,
	"member":  true,
}

// isValidRole checks whether a role string is a recognized dashboard role.
func isValidRole(role string) bool {
	return validRoles[role]
}

// primaryRole returns the highest-priority role from a list.
// Priority: admin > manager > member.
func primaryRole(roles []string) string {
	for _, r := range roles {
		if r == "admin" {
			return "admin"
		}
	}
	for _, r := range roles {
		if r == "manager" {
			return "manager"
		}
	}
	return "member"
}
