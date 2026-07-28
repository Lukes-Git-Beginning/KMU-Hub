package gateway

import (
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/notification/integration"
	"github.com/kmuhub/kmuhub/internal/server/response"
	notificationv1 "github.com/kmuhub/kmuhub/proto/notification/v1"
)

// maxWebhookBody caps an inbound platform payload. Slack and Teams stay far
// below this; the limit exists because these five routes are the only
// unauthenticated write paths in the gateway.
const maxWebhookBody = 1 << 20 // 1 MiB

// webhookForwardHeaders are the only request headers tunneled to the
// notification service: the two Slack signature headers, the Bot Framework
// bearer token and the content type the form parser needs. Forwarding the whole
// header set would ship cookies and proxy headers into another service for no
// reason.
var webhookForwardHeaders = []string{
	"Content-Type",
	"X-Slack-Signature",
	"X-Slack-Request-Timestamp",
	"Authorization",
}

// IntegrationRoutes handles HTTP routes for Teams/Slack integration admin config,
// channel mappings, account linking, and inbound webhooks.
type IntegrationRoutes struct {
	registry *ServiceRegistry
}

// NewIntegrationRoutes creates a new IntegrationRoutes.
func NewIntegrationRoutes(registry *ServiceRegistry) *IntegrationRoutes {
	return &IntegrationRoutes{registry: registry}
}

// ServiceName returns "notification" to reuse the existing gRPC connection
// (integration RPCs co-hosted on notification service).
func (ir *IntegrationRoutes) ServiceName() string { return "notification" }

// getNotificationClient lazily obtains a gRPC client.
func (ir *IntegrationRoutes) getNotificationClient() (notificationv1.NotificationServiceClient, error) {
	conn, err := ir.registry.GetConnection("notification")
	if err != nil {
		return nil, err
	}
	return notificationv1.NewNotificationServiceClient(conn), nil
}

// RegisterRoutes registers all integration HTTP routes.
func (ir *IntegrationRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/integrations", func(r chi.Router) {
		// ================================================================
		// Admin config routes (auth + admin role required)
		// ================================================================
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Use(middleware.RequireRole("admin"))

			// Integration configs
			r.Get("/configs", ir.HandleListConfigs)
			r.Get("/configs/{platform}", ir.HandleGetConfig)
			r.Post("/configs", ir.HandleCreateConfig)
			r.Put("/configs/{platform}", ir.HandleUpdateConfig)
			r.Delete("/configs/{platform}", ir.HandleDeleteConfig)
			r.Post("/configs/{platform}/test", ir.HandleTestConfig)

			// Channel mappings
			r.Get("/configs/{platform}/mappings", ir.HandleListMappings)
			r.Post("/configs/{platform}/mappings", ir.HandleCreateMapping)
			r.Put("/mappings/{id}", ir.HandleUpdateMapping)
			r.Delete("/mappings/{id}", ir.HandleDeleteMapping)
		})

		// ================================================================
		// Account linking routes (auth, any user)
		// ================================================================
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)

			r.Post("/link", ir.HandleLinkAccount)
			r.Delete("/link/{platform}", ir.HandleUnlinkAccount)
			r.Get("/link/{platform}/status", ir.HandleGetLinkStatus)
		})

		// ================================================================
		// Inbound webhooks (NO standard auth -- external platforms call these)
		// Platform-specific signature verification in handlers
		// ================================================================
		r.Post("/teams/webhook", ir.HandleTeamsWebhook)
		r.Post("/slack/interact", ir.HandleSlackInteraction)
		r.Post("/slack/commands", ir.HandleSlackSlashCommand)
		r.Get("/slack/oauth/install", ir.HandleSlackOAuthInstall)
		r.Get("/slack/oauth/callback", ir.HandleSlackOAuthCallback)
	})
}

// ============================================================================
// Admin Config Handlers
// ============================================================================

func (ir *IntegrationRoutes) HandleListConfigs(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	resp, err := client.ListIntegrationConfigs(r.Context(), &notificationv1.ListIntegrationConfigsRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (ir *IntegrationRoutes) HandleGetConfig(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	platform := chi.URLParam(r, "platform")
	resp, err := client.GetIntegrationConfig(r.Context(), &notificationv1.GetIntegrationConfigRequest{
		Platform: platform,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

type createConfigRequest struct {
	Platform            string `json:"platform" validate:"required"`
	CredentialsVaultKey string `json:"credentials_vault_key" validate:"required"`
	Metadata            string `json:"metadata"`
}

func (ir *IntegrationRoutes) HandleCreateConfig(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[createConfigRequest](w, r)
	if !ok {
		return
	}

	if req.Metadata == "" {
		req.Metadata = "{}"
	}

	resp, err := client.CreateIntegrationConfig(r.Context(), &notificationv1.CreateIntegrationConfigRequest{
		Platform:            req.Platform,
		CredentialsVaultKey: req.CredentialsVaultKey,
		Metadata:            req.Metadata,
		CreatedBy:           userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp)
}

type updateConfigRequest struct {
	IsActive            bool    `json:"is_active"`
	CredentialsVaultKey *string `json:"credentials_vault_key,omitempty"`
	Metadata            string  `json:"metadata"`
}

func (ir *IntegrationRoutes) HandleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	platform := chi.URLParam(r, "platform")

	// First get the config to obtain its ID
	getResp, err := client.GetIntegrationConfig(r.Context(), &notificationv1.GetIntegrationConfigRequest{
		Platform: platform,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	req, ok := decodeAndValidate[updateConfigRequest](w, r)
	if !ok {
		return
	}

	if req.Metadata == "" {
		req.Metadata = "{}"
	}

	grpcReq := &notificationv1.UpdateIntegrationConfigRequest{
		Id:       getResp.Config.Id,
		IsActive: req.IsActive,
		Metadata: req.Metadata,
	}
	if req.CredentialsVaultKey != nil {
		grpcReq.CredentialsVaultKey = req.CredentialsVaultKey
	}

	resp, err := client.UpdateIntegrationConfig(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (ir *IntegrationRoutes) HandleDeleteConfig(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	platform := chi.URLParam(r, "platform")

	// Get config to obtain ID
	getResp, err := client.GetIntegrationConfig(r.Context(), &notificationv1.GetIntegrationConfigRequest{
		Platform: platform,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	_, err = client.DeleteIntegrationConfig(r.Context(), &notificationv1.DeleteIntegrationConfigRequest{
		Id: getResp.Config.Id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "config deleted"})
}

func (ir *IntegrationRoutes) HandleTestConfig(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	platform := chi.URLParam(r, "platform")

	resp, err := client.TestIntegrationConfig(r.Context(), &notificationv1.TestIntegrationConfigRequest{
		Platform: platform,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Channel Mapping Handlers
// ============================================================================

func (ir *IntegrationRoutes) HandleListMappings(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	platform := chi.URLParam(r, "platform")

	// Get config ID from platform
	getResp, err := client.GetIntegrationConfig(r.Context(), &notificationv1.GetIntegrationConfigRequest{
		Platform: platform,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	resp, err := client.ListChannelMappings(r.Context(), &notificationv1.ListChannelMappingsRequest{
		ConfigId: getResp.Config.Id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

type createMappingRequest struct {
	ChannelID   string   `json:"channel_id" validate:"required"`
	ChannelName string   `json:"channel_name" validate:"required"`
	Modules     []string `json:"modules" validate:"required,min=1"`
}

func (ir *IntegrationRoutes) HandleCreateMapping(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	platform := chi.URLParam(r, "platform")

	// Get config ID
	getResp, err := client.GetIntegrationConfig(r.Context(), &notificationv1.GetIntegrationConfigRequest{
		Platform: platform,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	req, ok := decodeAndValidate[createMappingRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateChannelMapping(r.Context(), &notificationv1.CreateChannelMappingRequest{
		ConfigId:    getResp.Config.Id,
		ChannelId:   req.ChannelID,
		ChannelName: req.ChannelName,
		Modules:     req.Modules,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp)
}

type updateMappingRequest struct {
	ChannelID   string   `json:"channel_id" validate:"required"`
	ChannelName string   `json:"channel_name" validate:"required"`
	Modules     []string `json:"modules" validate:"required,min=1"`
	IsActive    bool     `json:"is_active"`
}

func (ir *IntegrationRoutes) HandleUpdateMapping(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	mappingID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateMappingRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateChannelMapping(r.Context(), &notificationv1.UpdateChannelMappingRequest{
		Id:          mappingID,
		ChannelId:   req.ChannelID,
		ChannelName: req.ChannelName,
		Modules:     req.Modules,
		IsActive:    req.IsActive,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (ir *IntegrationRoutes) HandleDeleteMapping(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	mappingID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteChannelMapping(r.Context(), &notificationv1.DeleteChannelMappingRequest{
		Id: mappingID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "mapping deleted"})
}

// ============================================================================
// Account Linking Handlers
// ============================================================================

type linkAccountRequest struct {
	Token string `json:"token" validate:"required"`
}

func (ir *IntegrationRoutes) HandleLinkAccount(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[linkAccountRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.LinkAccount(r.Context(), &notificationv1.LinkAccountRequest{
		Token:  req.Token,
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (ir *IntegrationRoutes) HandleUnlinkAccount(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	platform := chi.URLParam(r, "platform")

	_, err = client.UnlinkAccount(r.Context(), &notificationv1.UnlinkAccountRequest{
		Platform: platform,
		UserId:   userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "account unlinked"})
}

func (ir *IntegrationRoutes) HandleGetLinkStatus(w http.ResponseWriter, r *http.Request) {
	client, err := ir.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	platform := chi.URLParam(r, "platform")

	resp, err := client.GetAccountLinkStatus(r.Context(), &notificationv1.GetAccountLinkStatusRequest{
		UserId: userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	for _, link := range resp.GetLinks() {
		if link.GetPlatform() != platform {
			continue
		}

		status := map[string]interface{}{
			"linked":                true,
			"platform":              platform,
			"external_display_name": link.GetExternalDisplayName(),
		}
		if linkedAt := link.GetLinkedAt(); linkedAt != nil {
			status["linked_at"] = linkedAt.AsTime().Format(time.RFC3339)
		}

		response.JSON(w, http.StatusOK, status)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"linked":   false,
		"platform": platform,
	})
}

// ============================================================================
// Inbound Webhook Handlers (no standard auth)
// ============================================================================

func (ir *IntegrationRoutes) HandleTeamsWebhook(w http.ResponseWriter, r *http.Request) {
	ir.proxyPlatformWebhook(w, r, integration.PlatformTeams, integration.WebhookKindTeamsActivity)
}

func (ir *IntegrationRoutes) HandleSlackInteraction(w http.ResponseWriter, r *http.Request) {
	ir.proxyPlatformWebhook(w, r, integration.PlatformSlack, integration.WebhookKindSlackInteraction)
}

func (ir *IntegrationRoutes) HandleSlackSlashCommand(w http.ResponseWriter, r *http.Request) {
	ir.proxyPlatformWebhook(w, r, integration.PlatformSlack, integration.WebhookKindSlackCommand)
}

// proxyPlatformWebhook tunnels one inbound platform request to the notification
// service and writes its answer back verbatim.
//
// The gateway deliberately does no work of its own here: it does not parse the
// payload, does not verify the signature and touches no database. Slack signs
// the exact bytes it sent and the Bot Framework JWT covers the payload, so
// re-serializing anywhere along the way would break verification -- tunneling
// the raw body is the only form that keeps the check meaningful. It also keeps
// the signing secret in the notification service, where the platform tokens
// already live, and lets the tenant be resolved from the platform identity
// behind the gRPC boundary instead of by a direct repository call from a proxy.
func (ir *IntegrationRoutes) proxyPlatformWebhook(w http.ResponseWriter, r *http.Request, platform, kind string) {
	client, err := ir.getNotificationClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxWebhookBody))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "request body unreadable or too large")
		return
	}

	headers := make(map[string]string, len(webhookForwardHeaders))
	for _, name := range webhookForwardHeaders {
		if value := r.Header.Get(name); value != "" {
			headers[name] = value
		}
	}

	resp, err := client.HandlePlatformWebhook(r.Context(), &notificationv1.HandlePlatformWebhookRequest{
		Platform: platform,
		Kind:     kind,
		Body:     body,
		Headers:  headers,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	statusCode := int(resp.GetStatusCode())
	if statusCode < 100 || statusCode > 599 {
		slog.Error("integration webhook: service returned an invalid status code",
			"platform", platform,
			"kind", kind,
			"status_code", statusCode,
		)
		response.Error(w, http.StatusBadGateway, "invalid response from notification service")
		return
	}

	contentType := resp.GetContentType()
	if contentType == "" {
		contentType = "text/plain; charset=utf-8"
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(statusCode)
	if _, err := w.Write(resp.GetBody()); err != nil {
		slog.Warn("integration webhook: failed to write response", "platform", platform, "error", err)
	}
}

// respondOAuthNotAvailable answers the two Slack OAuth routes.
//
// They stay off on purpose rather than for lack of wiring: the install flow
// hands back a per-workspace bot token, and there is nowhere to keep it.
// integration_configs.credentials_vault_key is a bare string with no resolver
// behind it, and the callback arrives unauthenticated, so it would additionally
// need a signed state carrying the tenant from an authenticated install. Both
// are their own unit; answering 501 is the honest state until then.
func (ir *IntegrationRoutes) respondOAuthNotAvailable(w http.ResponseWriter) {
	response.Error(w, http.StatusNotImplemented, "slack oauth install is not available on this deployment")
}

func (ir *IntegrationRoutes) HandleSlackOAuthInstall(w http.ResponseWriter, _ *http.Request) {
	ir.respondOAuthNotAvailable(w)
}

func (ir *IntegrationRoutes) HandleSlackOAuthCallback(w http.ResponseWriter, _ *http.Request) {
	ir.respondOAuthNotAvailable(w)
}
