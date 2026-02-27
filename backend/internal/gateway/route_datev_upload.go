package gateway

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// DatevUploadRoutes handles HTTP routes for the DATEV upload integration.
type DatevUploadRoutes struct {
	registry *ServiceRegistry
}

// NewDatevUploadRoutes creates a new DatevUploadRoutes.
func NewDatevUploadRoutes(registry *ServiceRegistry) *DatevUploadRoutes {
	return &DatevUploadRoutes{registry: registry}
}

// ServiceName returns "biz" to reuse the existing gRPC connection
// (DATEV upload service co-hosted on biz binary).
func (dr *DatevUploadRoutes) ServiceName() string { return "biz" }

// getDatevUploadClient lazily obtains a gRPC client for the DATEV upload service.
func (dr *DatevUploadRoutes) getDatevUploadClient() (bizv1.DatevUploadServiceClient, error) {
	conn, err := dr.registry.GetConnection("biz")
	if err != nil {
		return nil, err
	}
	return bizv1.NewDatevUploadServiceClient(conn), nil
}

// RegisterRoutes registers all DATEV upload HTTP routes.
func (dr *DatevUploadRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/finance/datev", func(r chi.Router) {
		// OAuth callback — NO auth (public, called by DATEV redirect)
		r.Get("/oauth/callback", dr.HandleOAuthCallback)

		// Admin routes — require auth + admin role
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Use(middleware.RequireRole("admin"))

			// OAuth management
			r.Get("/oauth/authorize", dr.HandleGetAuthURL)
			r.Post("/disconnect", dr.HandleDisconnect)
			r.Get("/status", dr.HandleGetConnectionStatus)

			// Upload operations
			r.Post("/upload", dr.HandleUploadBuchungsstapel)
			r.Post("/upload/beleg/{invoice_id}", dr.HandleUploadBeleg)

			// Config
			r.Get("/config", dr.HandleGetUploadConfig)
			r.Put("/config", dr.HandleUpdateUploadConfig)

			// Logs
			r.Get("/upload/logs", dr.HandleListUploadLogs)
		})
	})
}

// ============================================================================
// OAuth Handlers
// ============================================================================

// HandleGetAuthURL returns the DATEV OAuth authorization URL.
func (dr *DatevUploadRoutes) HandleGetAuthURL(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDatevUploadClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	tenantID := getTenantID(r)

	resp, err := client.GetDatevAuthURL(r.Context(), &bizv1.GetDatevAuthURLRequest{
		TenantId:    tenantID,
		RedirectUrl: r.URL.Query().Get("redirect_url"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"authorization_url": resp.GetAuthorizationUrl(),
	})
}

// HandleOAuthCallback handles the DATEV OAuth redirect callback.
// This endpoint is public (no auth middleware) because DATEV redirects here.
func (dr *DatevUploadRoutes) HandleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDatevUploadClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" {
		errorMsg := r.URL.Query().Get("error")
		if errorMsg == "" {
			errorMsg = "missing authorization code"
		}
		http.Redirect(w, r, "/settings/integrations?datev_error="+errorMsg, http.StatusFound)
		return
	}

	// State contains the tenant_id
	resp, err := client.HandleDatevOAuthCallback(r.Context(), &bizv1.HandleDatevOAuthCallbackRequest{
		TenantId:    state,
		Code:        code,
		RedirectUrl: r.URL.Query().Get("redirect_url"),
	})
	if err != nil {
		http.Redirect(w, r, "/settings/integrations?datev_error=connection_failed", http.StatusFound)
		return
	}

	if !resp.GetSuccess() {
		http.Redirect(w, r, "/settings/integrations?datev_error="+resp.GetErrorMessage(), http.StatusFound)
		return
	}

	http.Redirect(w, r, "/settings/integrations?datev_connected=true", http.StatusFound)
}

// HandleDisconnect disconnects the DATEV integration.
func (dr *DatevUploadRoutes) HandleDisconnect(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDatevUploadClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	tenantID := getTenantID(r)

	_, err = client.DisconnectDatev(r.Context(), &bizv1.DisconnectDatevRequest{
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"success": true})
}

// HandleGetConnectionStatus returns the current DATEV connection status.
func (dr *DatevUploadRoutes) HandleGetConnectionStatus(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDatevUploadClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	tenantID := getTenantID(r)

	resp, err := client.GetDatevConnectionStatus(r.Context(), &bizv1.GetDatevConnectionStatusRequest{
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"connected": resp.GetConnected(),
	})
}

// ============================================================================
// Upload Handlers
// ============================================================================

// HandleUploadBuchungsstapel triggers a DATEV Buchungsstapel export and upload.
func (dr *DatevUploadRoutes) HandleUploadBuchungsstapel(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDatevUploadClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	tenantID := getTenantID(r)

	var body struct {
		StartDate       string `json:"start_date"`
		EndDate         string `json:"end_date"`
		FiscalYearStart string `json:"fiscal_year_start"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.UploadDatevBuchungsstapel(r.Context(), &bizv1.UploadDatevBuchungsstapelRequest{
		TenantId:        tenantID,
		StartDate:       body.StartDate,
		EndDate:         body.EndDate,
		FiscalYearStart: body.FiscalYearStart,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	result := map[string]interface{}{
		"success":        resp.GetSuccess(),
		"document_count": resp.GetDocumentCount(),
		"file_size":      resp.GetFileSize(),
	}
	if resp.GetErrorMessage() != "" {
		result["error_message"] = resp.GetErrorMessage()
	}

	response.JSON(w, http.StatusOK, result)
}

// HandleUploadBeleg uploads a single invoice PDF as a DATEV Belegbild.
func (dr *DatevUploadRoutes) HandleUploadBeleg(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDatevUploadClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	invoiceID := chi.URLParam(r, "invoice_id")

	resp, err := client.UploadDatevBeleg(r.Context(), &bizv1.UploadDatevBelegRequest{
		TenantId:  tenantID,
		InvoiceId: invoiceID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	result := map[string]interface{}{
		"success": resp.GetSuccess(),
	}
	if resp.GetErrorMessage() != "" {
		result["error_message"] = resp.GetErrorMessage()
	}

	response.JSON(w, http.StatusOK, result)
}

// ============================================================================
// Config Handlers
// ============================================================================

// HandleGetUploadConfig returns the current DATEV upload configuration.
func (dr *DatevUploadRoutes) HandleGetUploadConfig(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDatevUploadClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	tenantID := getTenantID(r)

	resp, err := client.GetDatevUploadConfig(r.Context(), &bizv1.GetDatevUploadConfigRequest{
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"client_number":       resp.GetClientNumber(),
		"auto_upload_enabled": resp.GetAutoUploadEnabled(),
		"upload_after_export": resp.GetUploadAfterExport(),
	})
}

// HandleUpdateUploadConfig updates the DATEV upload configuration.
func (dr *DatevUploadRoutes) HandleUpdateUploadConfig(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDatevUploadClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	tenantID := getTenantID(r)

	var body struct {
		ClientNumber      string `json:"client_number"`
		AutoUploadEnabled bool   `json:"auto_upload_enabled"`
		UploadAfterExport bool   `json:"upload_after_export"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	_, err = client.UpdateDatevUploadConfig(r.Context(), &bizv1.UpdateDatevUploadConfigRequest{
		TenantId:          tenantID,
		ClientNumber:      body.ClientNumber,
		AutoUploadEnabled: body.AutoUploadEnabled,
		UploadAfterExport: body.UploadAfterExport,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"success": true})
}

// ============================================================================
// Log Handlers
// ============================================================================

// HandleListUploadLogs returns recent DATEV upload log entries.
func (dr *DatevUploadRoutes) HandleListUploadLogs(w http.ResponseWriter, r *http.Request) {
	client, err := dr.getDatevUploadClient()
	if err != nil {
		respondServiceUnavailable(w, dr.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	limit := parseLimit(r, 20, 100)

	resp, err := client.ListDatevUploadLogs(r.Context(), &bizv1.ListDatevUploadLogsRequest{
		TenantId: tenantID,
		Limit:    int32(limit),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	entries := make([]map[string]interface{}, 0, len(resp.GetEntries()))
	for _, e := range resp.GetEntries() {
		entry := map[string]interface{}{
			"id":             e.GetId(),
			"upload_type":    e.GetUploadType(),
			"status":         e.GetStatus(),
			"file_size":      e.GetFileSize(),
			"document_count": e.GetDocumentCount(),
			"started_at":     e.GetStartedAt().AsTime(),
		}
		if e.GetErrorMessage() != "" {
			entry["error_message"] = e.GetErrorMessage()
		}
		if e.GetCompletedAt() != nil {
			entry["completed_at"] = e.GetCompletedAt().AsTime()
		}
		entries = append(entries, entry)
	}

	response.JSON(w, http.StatusOK, entries)
}
