package gateway

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// LexwareRoutes handles HTTP routes for the Lexware integration.
type LexwareRoutes struct {
	registry *ServiceRegistry
}

// NewLexwareRoutes creates a new LexwareRoutes.
func NewLexwareRoutes(registry *ServiceRegistry) *LexwareRoutes {
	return &LexwareRoutes{registry: registry}
}

// ServiceName returns "biz" to reuse the existing gRPC connection
// (Lexware service co-hosted on biz binary).
func (lr *LexwareRoutes) ServiceName() string { return "biz" }

// getLexwareClient lazily obtains a gRPC client for the Lexware integration service.
func (lr *LexwareRoutes) getLexwareClient() (bizv1.LexwareIntegrationServiceClient, error) {
	conn, err := lr.registry.GetConnection("biz")
	if err != nil {
		return nil, err
	}
	return bizv1.NewLexwareIntegrationServiceClient(conn), nil
}

// RegisterRoutes registers all Lexware integration HTTP routes.
func (lr *LexwareRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/integrations/lexware", func(r chi.Router) {
		// Webhook endpoint — NO auth (public, called by Lexware)
		r.Post("/webhooks", lr.HandleWebhookEvent)

		// Admin routes — require auth + admin role
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Use(middleware.RequireRole("admin"))

			// Connection management
			r.Post("/connect", lr.HandleConnect)
			r.Post("/disconnect", lr.HandleDisconnect)
			r.Get("/status", lr.HandleGetConnectionStatus)
			r.Post("/test", lr.HandleTestConnection)

			// Sync
			r.Post("/sync/trigger", lr.HandleTriggerSync)
			r.Get("/sync/status", lr.HandleGetSyncStatus)
			r.Get("/sync/logs", lr.HandleListSyncLogs)

			// Field mappings
			r.Get("/mappings/{entity_type}", lr.HandleGetFieldMappings)
			r.Put("/mappings/{entity_type}", lr.HandleUpdateFieldMappings)

			// Manual push
			r.Post("/push/invoice/{id}", lr.HandlePushInvoice)
			r.Post("/push/quote/{id}", lr.HandlePushQuote)
		})
	})
}

// ============================================================================
// Connection Handlers
// ============================================================================

// HandleConnect stores the API key and activates the Lexware integration.
func (lr *LexwareRoutes) HandleConnect(w http.ResponseWriter, r *http.Request) {
	client, err := lr.getLexwareClient()
	if err != nil {
		respondServiceUnavailable(w, lr.ServiceName())
		return
	}

	tenantID := getTenantID(r)

	var body struct {
		APIKey string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.APIKey == "" {
		response.Error(w, http.StatusBadRequest, "api_key is required")
		return
	}

	resp, err := client.ConnectLexware(r.Context(), &bizv1.ConnectLexwareRequest{
		TenantId: tenantID,
		ApiKey:   body.APIKey,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	if !resp.GetSuccess() {
		response.Error(w, http.StatusBadRequest, resp.GetErrorMessage())
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"success": true})
}

// HandleDisconnect disconnects the Lexware integration.
func (lr *LexwareRoutes) HandleDisconnect(w http.ResponseWriter, r *http.Request) {
	client, err := lr.getLexwareClient()
	if err != nil {
		respondServiceUnavailable(w, lr.ServiceName())
		return
	}

	tenantID := getTenantID(r)

	_, err = client.DisconnectLexware(r.Context(), &bizv1.DisconnectLexwareRequest{
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"success": true})
}

// HandleGetConnectionStatus returns the current Lexware connection status.
func (lr *LexwareRoutes) HandleGetConnectionStatus(w http.ResponseWriter, r *http.Request) {
	client, err := lr.getLexwareClient()
	if err != nil {
		respondServiceUnavailable(w, lr.ServiceName())
		return
	}

	tenantID := getTenantID(r)

	resp, err := client.GetLexwareConnectionStatus(r.Context(), &bizv1.GetLexwareConnectionStatusRequest{
		TenantId: tenantID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	result := map[string]interface{}{
		"connected": resp.GetConnected(),
	}
	if resp.GetConnectedAt() != nil {
		result["connected_at"] = resp.GetConnectedAt().AsTime()
	}

	response.JSON(w, http.StatusOK, result)
}

// HandleTestConnection tests the Lexware API connection.
func (lr *LexwareRoutes) HandleTestConnection(w http.ResponseWriter, r *http.Request) {
	client, err := lr.getLexwareClient()
	if err != nil {
		respondServiceUnavailable(w, lr.ServiceName())
		return
	}

	tenantID := getTenantID(r)

	resp, err := client.TestLexwareConnection(r.Context(), &bizv1.TestLexwareConnectionRequest{
		TenantId: tenantID,
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
// Sync Handlers
// ============================================================================

// HandleTriggerSync triggers a manual Lexware sync.
func (lr *LexwareRoutes) HandleTriggerSync(w http.ResponseWriter, r *http.Request) {
	client, err := lr.getLexwareClient()
	if err != nil {
		respondServiceUnavailable(w, lr.ServiceName())
		return
	}

	tenantID := getTenantID(r)

	var body struct {
		SyncType string `json:"sync_type"`
	}
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	resp, err := client.TriggerLexwareSync(r.Context(), &bizv1.TriggerLexwareSyncRequest{
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

// HandleGetSyncStatus returns the Lexware sync status overview.
func (lr *LexwareRoutes) HandleGetSyncStatus(w http.ResponseWriter, r *http.Request) {
	client, err := lr.getLexwareClient()
	if err != nil {
		respondServiceUnavailable(w, lr.ServiceName())
		return
	}

	tenantID := getTenantID(r)

	resp, err := client.GetLexwareSyncStatus(r.Context(), &bizv1.GetLexwareSyncStatusRequest{
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
		"webhook_enabled":       resp.GetWebhookEnabled(),
		"total_contacts_mapped": resp.GetTotalContactsMapped(),
		"total_invoices_mapped": resp.GetTotalInvoicesMapped(),
		"total_quotes_mapped":   resp.GetTotalQuotesMapped(),
	}
	if resp.GetLastContactSyncAt() != nil {
		result["last_contact_sync_at"] = resp.GetLastContactSyncAt().AsTime()
	}
	if resp.GetLastSyncError() != "" {
		result["last_sync_error"] = resp.GetLastSyncError()
	}
	if resp.GetLastSyncErrorAt() != nil {
		result["last_sync_error_at"] = resp.GetLastSyncErrorAt().AsTime()
	}

	response.JSON(w, http.StatusOK, result)
}

// HandleListSyncLogs returns recent Lexware sync log entries.
func (lr *LexwareRoutes) HandleListSyncLogs(w http.ResponseWriter, r *http.Request) {
	client, err := lr.getLexwareClient()
	if err != nil {
		respondServiceUnavailable(w, lr.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	limit := parseLimit(r, 20, 100)

	resp, err := client.ListLexwareSyncLogs(r.Context(), &bizv1.ListLexwareSyncLogsRequest{
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

// HandleGetFieldMappings returns Lexware field mappings for an entity type.
func (lr *LexwareRoutes) HandleGetFieldMappings(w http.ResponseWriter, r *http.Request) {
	client, err := lr.getLexwareClient()
	if err != nil {
		respondServiceUnavailable(w, lr.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	entityType := chi.URLParam(r, "entity_type")

	resp, err := client.GetLexwareFieldMappings(r.Context(), &bizv1.GetLexwareFieldMappingsRequest{
		TenantId:   tenantID,
		EntityType: entityType,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.GetMappings())
}

// HandleUpdateFieldMappings saves Lexware field mappings for an entity type.
func (lr *LexwareRoutes) HandleUpdateFieldMappings(w http.ResponseWriter, r *http.Request) {
	client, err := lr.getLexwareClient()
	if err != nil {
		respondServiceUnavailable(w, lr.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	entityType := chi.URLParam(r, "entity_type")

	var body struct {
		Mappings []*bizv1.LexwareFieldMappingEntry `json:"mappings"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	_, err = client.UpdateLexwareFieldMappings(r.Context(), &bizv1.UpdateLexwareFieldMappingsRequest{
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

// HandlePushInvoice pushes an invoice to Lexware.
func (lr *LexwareRoutes) HandlePushInvoice(w http.ResponseWriter, r *http.Request) {
	client, err := lr.getLexwareClient()
	if err != nil {
		respondServiceUnavailable(w, lr.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	invoiceID := chi.URLParam(r, "id")

	resp, err := client.PushInvoiceToLexware(r.Context(), &bizv1.PushInvoiceToLexwareRequest{
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
	if resp.GetLexwareId() != "" {
		result["lexware_id"] = resp.GetLexwareId()
	}
	if resp.GetErrorMessage() != "" {
		result["error_message"] = resp.GetErrorMessage()
	}

	response.JSON(w, http.StatusOK, result)
}

// HandlePushQuote pushes a quote to Lexware.
func (lr *LexwareRoutes) HandlePushQuote(w http.ResponseWriter, r *http.Request) {
	client, err := lr.getLexwareClient()
	if err != nil {
		respondServiceUnavailable(w, lr.ServiceName())
		return
	}

	tenantID := getTenantID(r)
	quoteID := chi.URLParam(r, "id")

	resp, err := client.PushQuoteToLexware(r.Context(), &bizv1.PushQuoteToLexwareRequest{
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
	if resp.GetLexwareId() != "" {
		result["lexware_id"] = resp.GetLexwareId()
	}
	if resp.GetErrorMessage() != "" {
		result["error_message"] = resp.GetErrorMessage()
	}

	response.JSON(w, http.StatusOK, result)
}

// ============================================================================
// Webhook Handler
// ============================================================================

// HandleWebhookEvent processes incoming Lexware webhook events.
// This endpoint is public (no auth middleware) because Lexware calls it directly.
func (lr *LexwareRoutes) HandleWebhookEvent(w http.ResponseWriter, r *http.Request) {
	client, err := lr.getLexwareClient()
	if err != nil {
		respondServiceUnavailable(w, lr.ServiceName())
		return
	}

	var body struct {
		EventType      string `json:"event_type"`
		ResourceID     string `json:"resource_id"`
		OrganizationID string `json:"organization_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.HandleLexwareWebhookEvent(r.Context(), &bizv1.HandleLexwareWebhookEventRequest{
		EventType:      body.EventType,
		ResourceId:     body.ResourceID,
		OrganizationId: body.OrganizationID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]bool{"success": resp.GetSuccess()})
}
