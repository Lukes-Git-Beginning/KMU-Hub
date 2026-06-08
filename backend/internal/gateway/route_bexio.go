package gateway

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// BexioRoutes handles HTTP routes for the Bexio integration.
type BexioRoutes struct {
	registry *ServiceRegistry
}

// NewBexioRoutes creates a new BexioRoutes.
func NewBexioRoutes(registry *ServiceRegistry) *BexioRoutes {
	return &BexioRoutes{registry: registry}
}

// ServiceName returns "biz" to reuse the existing gRPC connection
// (Bexio service co-hosted on biz binary).
func (br *BexioRoutes) ServiceName() string { return "biz" }

// getBexioClient lazily obtains a gRPC client for the Bexio integration service.
func (br *BexioRoutes) getBexioClient() (bizv1.BexioIntegrationServiceClient, error) {
	conn, err := br.registry.GetConnection("biz")
	if err != nil {
		return nil, err
	}
	return bizv1.NewBexioIntegrationServiceClient(conn), nil
}

// RegisterRoutes registers all Bexio integration HTTP routes.
func (br *BexioRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/integrations/bexio", func(r chi.Router) {
		// OAuth callback — NO auth (public, called by Bexio redirect)
		r.Get("/oauth/callback", br.HandleOAuthCallback)

		// Admin routes — require auth + admin role
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Use(middleware.RequireRole("admin"))

			// OAuth management
			r.Get("/oauth/authorize", br.HandleGetAuthURL)
			r.Post("/disconnect", br.HandleDisconnect)
			r.Get("/status", br.HandleGetConnectionStatus)

			// Sync
			r.Post("/sync/trigger", br.HandleTriggerSync)
			r.Get("/sync/status", br.HandleGetSyncStatus)
			r.Get("/sync/logs", br.HandleListSyncLogs)

			// Field mappings
			r.Get("/mappings/{entity_type}", br.HandleGetFieldMappings)
			r.Put("/mappings/{entity_type}", br.HandleUpdateFieldMappings)

			// Manual push
			r.Post("/push/invoice/{invoice_id}", br.HandlePushInvoice)
			r.Post("/push/quote/{quote_id}", br.HandlePushQuote)
		})
	})
}

// ============================================================================
// OAuth Handlers
// ============================================================================

// HandleOAuthCallback handles the Bexio OAuth redirect callback.
// This endpoint is public (no auth middleware) because Bexio redirects here.
func (br *BexioRoutes) HandleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	client, err := br.getBexioClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" {
		errorMsg := r.URL.Query().Get("error")
		if errorMsg == "" {
			errorMsg = "missing authorization code"
		}
		http.Redirect(w, r, "/settings/integrations?bexio_error="+errorMsg, http.StatusFound)
		return
	}

	// State contains the tenant_id
	resp, err := client.HandleBexioOAuthCallback(r.Context(), &bizv1.HandleBexioOAuthCallbackRequest{
		TenantId: state,
		Code:     code,
	})
	if err != nil {
		http.Redirect(w, r, "/settings/integrations?bexio_error=connection_failed", http.StatusFound)
		return
	}

	if !resp.GetSuccess() {
		http.Redirect(w, r, "/settings/integrations?bexio_error="+resp.GetErrorMessage(), http.StatusFound)
		return
	}

	http.Redirect(w, r, "/settings/integrations?bexio_connected=true", http.StatusFound)
}

// HandleGetAuthURL returns the Bexio OAuth authorization URL.
func (br *BexioRoutes) HandleGetAuthURL(w http.ResponseWriter, r *http.Request) {
	client, err := br.getBexioClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}

	resp, err := client.GetBexioAuthURL(r.Context(), &bizv1.GetBexioAuthURLRequest{
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

// HandleDisconnect disconnects the Bexio integration.
func (br *BexioRoutes) HandleDisconnect(w http.ResponseWriter, r *http.Request) {
	client, err := br.getBexioClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}

	_, err = client.DisconnectBexio(r.Context(), &bizv1.DisconnectBexioRequest{
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"success": true})
}

// HandleGetConnectionStatus returns the current Bexio connection status.
func (br *BexioRoutes) HandleGetConnectionStatus(w http.ResponseWriter, r *http.Request) {
	client, err := br.getBexioClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}

	resp, err := client.GetBexioConnectionStatus(r.Context(), &bizv1.GetBexioConnectionStatusRequest{
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	result := map[string]interface{}{
		"connected": resp.GetConnected(),
		"org_name":  resp.GetOrgName(),
	}
	if resp.GetConnectedAt() != nil {
		result["connected_at"] = resp.GetConnectedAt().AsTime()
	}

	response.JSON(w, http.StatusOK, result)
}

// ============================================================================
// Sync Handlers
// ============================================================================

// HandleTriggerSync triggers a manual sync.
func (br *BexioRoutes) HandleTriggerSync(w http.ResponseWriter, r *http.Request) {
	client, err := br.getBexioClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}

	type triggerBexioSyncRequest struct {
		SyncType string `json:"sync_type"`
	}
	var body triggerBexioSyncRequest
	if r.Body != nil {
		var ok bool
		body, ok = decodeAndValidate[triggerBexioSyncRequest](w, r)
		if !ok {
			return
		}
	}

	resp, err := client.TriggerBexioSync(r.Context(), &bizv1.TriggerBexioSyncRequest{
		TenantId: tenantID,
		SyncType: body.SyncType,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusAccepted, map[string]string{
		"sync_id": resp.GetSyncId(),
		"status":  resp.GetStatus(),
	})
}

// HandleGetSyncStatus returns the sync status overview.
func (br *BexioRoutes) HandleGetSyncStatus(w http.ResponseWriter, r *http.Request) {
	client, err := br.getBexioClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}

	resp, err := client.GetBexioSyncStatus(r.Context(), &bizv1.GetBexioSyncStatusRequest{
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	result := map[string]interface{}{
		"contact_sync_enabled":  resp.GetContactSyncEnabled(),
		"invoice_push_enabled":  resp.GetInvoicePushEnabled(),
		"quote_push_enabled":    resp.GetQuotePushEnabled(),
		"payment_poll_enabled":  resp.GetPaymentPollEnabled(),
		"total_contacts_mapped": resp.GetTotalContactsMapped(),
		"total_invoices_mapped": resp.GetTotalInvoicesMapped(),
		"total_quotes_mapped":   resp.GetTotalQuotesMapped(),
	}
	if resp.GetLastContactSyncAt() != nil {
		result["last_contact_sync_at"] = resp.GetLastContactSyncAt().AsTime()
	}
	if resp.GetLastPaymentPollAt() != nil {
		result["last_payment_poll_at"] = resp.GetLastPaymentPollAt().AsTime()
	}
	if resp.GetLastSyncError() != "" {
		result["last_sync_error"] = resp.GetLastSyncError()
	}
	if resp.GetLastSyncErrorAt() != nil {
		result["last_sync_error_at"] = resp.GetLastSyncErrorAt().AsTime()
	}

	response.JSON(w, http.StatusOK, result)
}

// HandleListSyncLogs returns recent sync log entries.
func (br *BexioRoutes) HandleListSyncLogs(w http.ResponseWriter, r *http.Request) {
	client, err := br.getBexioClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	limit := parseLimit(r, 20, 100)

	resp, err := client.ListBexioSyncLogs(r.Context(), &bizv1.ListBexioSyncLogsRequest{
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
			"id":              e.GetId(),
			"sync_type":       e.GetSyncType(),
			"status":          e.GetStatus(),
			"items_processed": e.GetItemsProcessed(),
			"items_created":   e.GetItemsCreated(),
			"items_updated":   e.GetItemsUpdated(),
			"items_failed":    e.GetItemsFailed(),
			"started_at":      e.GetStartedAt().AsTime(),
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

// ============================================================================
// Field Mapping Handlers
// ============================================================================

// HandleGetFieldMappings returns field mappings for an entity type.
func (br *BexioRoutes) HandleGetFieldMappings(w http.ResponseWriter, r *http.Request) {
	client, err := br.getBexioClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	entityType := chi.URLParam(r, "entity_type")

	resp, err := client.GetBexioFieldMappings(r.Context(), &bizv1.GetBexioFieldMappingsRequest{
		TenantId:   tenantID,
		EntityType: entityType,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.GetMappings())
}

// HandleUpdateFieldMappings saves field mappings for an entity type.
func (br *BexioRoutes) HandleUpdateFieldMappings(w http.ResponseWriter, r *http.Request) {
	client, err := br.getBexioClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	entityType := chi.URLParam(r, "entity_type")

	type updateBexioFieldMappingsRequest struct {
		Mappings []*bizv1.BexioFieldMappingEntry `json:"mappings" validate:"required,min=1"`
	}
	body, ok := decodeAndValidate[updateBexioFieldMappingsRequest](w, r)
	if !ok {
		return
	}

	_, err = client.UpdateBexioFieldMappings(r.Context(), &bizv1.UpdateBexioFieldMappingsRequest{
		TenantId:   tenantID,
		EntityType: entityType,
		Mappings:   body.Mappings,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"success": true})
}

// ============================================================================
// Push Handlers
// ============================================================================

// HandlePushInvoice pushes an invoice to Bexio.
func (br *BexioRoutes) HandlePushInvoice(w http.ResponseWriter, r *http.Request) {
	client, err := br.getBexioClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	invoiceID := chi.URLParam(r, "invoice_id")

	resp, err := client.PushInvoiceToBexio(r.Context(), &bizv1.PushInvoiceToBexioRequest{
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
	if resp.GetBexioId() != "" {
		result["bexio_id"] = resp.GetBexioId()
	}
	if resp.GetErrorMessage() != "" {
		result["error_message"] = resp.GetErrorMessage()
	}

	response.JSON(w, http.StatusOK, result)
}

// HandlePushQuote pushes a quote to Bexio.
func (br *BexioRoutes) HandlePushQuote(w http.ResponseWriter, r *http.Request) {
	client, err := br.getBexioClient()
	if err != nil {
		respondServiceUnavailable(w, br.ServiceName())
		return
	}

	tenantID, err := getTenantID(r)
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	quoteID := chi.URLParam(r, "quote_id")

	resp, err := client.PushQuoteToBexio(r.Context(), &bizv1.PushQuoteToBexioRequest{
		TenantId: tenantID,
		QuoteId:  quoteID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	result := map[string]interface{}{
		"success": resp.GetSuccess(),
	}
	if resp.GetBexioId() != "" {
		result["bexio_id"] = resp.GetBexioId()
	}
	if resp.GetErrorMessage() != "" {
		result["error_message"] = resp.GetErrorMessage()
	}

	response.JSON(w, http.StatusOK, result)
}
