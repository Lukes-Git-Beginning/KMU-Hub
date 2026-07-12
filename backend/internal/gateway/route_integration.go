package gateway

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	slackadapter "github.com/kmuhub/kmuhub/internal/notification/integration/slack"
	teamsadapter "github.com/kmuhub/kmuhub/internal/notification/integration/teams"
	"github.com/kmuhub/kmuhub/internal/server/response"
	notificationv1 "github.com/kmuhub/kmuhub/proto/notification/v1"
)

// IntegrationRoutes handles HTTP routes for Teams/Slack integration admin config,
// channel mappings, account linking, and inbound webhooks.
type IntegrationRoutes struct {
	registry *ServiceRegistry

	// Inbound webhook handlers (injected directly, not via gRPC)
	teamsWebhook *teamsadapter.WebhookHandler
	slackWebhook *slackadapter.WebhookHandler
	slackOAuth   *slackadapter.OAuthHandler
}

// NewIntegrationRoutes creates a new IntegrationRoutes.
func NewIntegrationRoutes(registry *ServiceRegistry) *IntegrationRoutes {
	return &IntegrationRoutes{registry: registry}
}

// SetTeamsWebhookHandler sets the Teams webhook handler for inbound activities.
func (ir *IntegrationRoutes) SetTeamsWebhookHandler(h *teamsadapter.WebhookHandler) {
	ir.teamsWebhook = h
}

// SetSlackWebhookHandler sets the Slack webhook handler for interactions.
func (ir *IntegrationRoutes) SetSlackWebhookHandler(h *slackadapter.WebhookHandler) {
	ir.slackWebhook = h
}

// SetSlackOAuthHandler sets the Slack OAuth handler for install flow.
func (ir *IntegrationRoutes) SetSlackOAuthHandler(h *slackadapter.OAuthHandler) {
	ir.slackOAuth = h
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
	if ir.teamsWebhook == nil {
		response.Error(w, http.StatusNotFound, "teams integration not configured")
		return
	}
	ir.teamsWebhook.HandleWebhook(w, r)
}

func (ir *IntegrationRoutes) HandleSlackInteraction(w http.ResponseWriter, r *http.Request) {
	if ir.slackWebhook == nil {
		response.Error(w, http.StatusNotFound, "slack integration not configured")
		return
	}
	ir.slackWebhook.HandleInteraction(w, r)
}

func (ir *IntegrationRoutes) HandleSlackSlashCommand(w http.ResponseWriter, r *http.Request) {
	if ir.slackWebhook == nil {
		response.Error(w, http.StatusNotFound, "slack integration not configured")
		return
	}
	ir.slackWebhook.HandleSlashCommand(w, r)
}

func (ir *IntegrationRoutes) HandleSlackOAuthInstall(w http.ResponseWriter, r *http.Request) {
	if ir.slackOAuth == nil {
		response.Error(w, http.StatusNotFound, "slack oauth not configured")
		return
	}
	ir.slackOAuth.HandleInstall(w, r)
}

func (ir *IntegrationRoutes) HandleSlackOAuthCallback(w http.ResponseWriter, r *http.Request) {
	if ir.slackOAuth == nil {
		response.Error(w, http.StatusNotFound, "slack oauth not configured")
		return
	}
	ir.slackOAuth.HandleCallback(w, r)
}
