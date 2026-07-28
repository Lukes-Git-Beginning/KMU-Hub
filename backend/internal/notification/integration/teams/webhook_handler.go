package teams

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/infracloudio/msbotbuilder-go/schema"

	"github.com/kmuhub/kmuhub/internal/notification/integration"
)

// WebhookHandler processes inbound Teams Bot Framework activities.
//
// It runs inside the notification service, not the gateway: Bot Framework JWT
// verification, the integration repository and the tenant resolution all live
// here, and the gateway stays a proxy that never parses the payload.
type WebhookHandler struct {
	client        *Client
	repo          integration.Repository
	linkService   *integration.AccountLinkService
	tenants       integration.TenantResolver
	notifications integration.NotificationAcknowledger
}

// NewWebhookHandler creates a new Teams webhook handler.
func NewWebhookHandler(
	client *Client,
	repo integration.Repository,
	linkService *integration.AccountLinkService,
	tenants integration.TenantResolver,
	notifications integration.NotificationAcknowledger,
) *WebhookHandler {
	return &WebhookHandler{
		client:        client,
		repo:          repo,
		linkService:   linkService,
		tenants:       tenants,
		notifications: notifications,
	}
}

// Process verifies and dispatches one inbound Teams activity.
//
// The Bot Framework adapter validates the JWT against the raw payload, so the
// tunneled bytes are handed to it unchanged inside a synthetic request. Nothing
// is read from the database before that check passes.
func (h *WebhookHandler) Process(ctx context.Context, req integration.WebhookRequest) integration.WebhookResponse {
	if h.client == nil {
		return integration.TextResponse(http.StatusServiceUnavailable, "teams integration not configured")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "/", bytes.NewReader(req.Body))
	if err != nil {
		return integration.TextResponse(http.StatusInternalServerError, "internal error")
	}
	for name, value := range req.Headers {
		httpReq.Header.Set(name, value)
	}
	if httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	incoming, err := h.client.HandleIncomingActivity(ctx, httpReq)
	if err != nil {
		slog.Warn("teams webhook: failed to verify or parse activity", "error", err)
		return integration.TextResponse(http.StatusUnauthorized, "unauthorized")
	}

	switch incoming.Activity.Type {
	case schema.Invoke:
		return h.handleInvoke(ctx, incoming)
	case schema.Message:
		return h.handleMessage(ctx, incoming)
	default:
		return integration.EmptyResponse(http.StatusOK)
	}
}

// handleInvoke processes Action.Execute (adaptiveCard/action) invocations.
func (h *WebhookHandler) handleInvoke(ctx context.Context, incoming *IncomingAction) integration.WebhookResponse {
	ctx, tenantID, err := h.tenantContext(ctx, incoming)
	if err != nil {
		return invokeMessage("Diese Teams-Organisation ist keinem Cosmi-Mandanten zugeordnet. Bitte im Cosmi-Admin unter Integrationen die Tenant-ID hinterlegen.")
	}

	link, err := h.repo.GetAccountLink(ctx, integration.PlatformTeams, incoming.ExternalUserID)
	if err != nil {
		slog.Info("teams webhook: unlinked user attempted action",
			"external_user_id", incoming.ExternalUserID,
		)
		return invokeMessage("Bitte verknuepfen Sie Ihr Konto mit dem Befehl: /kmuhub link")
	}

	// Only acknowledge has an effect in Cosmi today; approve, reject and reply
	// have no counterpart yet and must not be answered as if they had.
	if incoming.ActionType != string(integration.ActionAcknowledge) {
		slog.Info("teams webhook: unsupported action requested",
			"action_type", incoming.ActionType,
			"notification_id", incoming.NotificationID,
			"user_id", link.KMUHubUserID,
		)
		return invokeMessage("Diese Aktion wird von Cosmi noch nicht ausgefuehrt. Bitte erledigen Sie sie in Cosmi.")
	}

	if err := h.acknowledge(ctx, tenantID, link, incoming.NotificationID); err != nil {
		slog.Error("teams webhook: acknowledge failed",
			"notification_id", incoming.NotificationID,
			"user_id", link.KMUHubUserID,
			"error", err,
		)
		return invokeMessage("Die Benachrichtigung konnte nicht als gelesen markiert werden.")
	}

	return invokeMessage("Als gelesen markiert.")
}

// handleMessage processes text messages (e.g., /kmuhub link command).
func (h *WebhookHandler) handleMessage(ctx context.Context, incoming *IncomingAction) integration.WebhookResponse {
	text := incoming.Activity.Text

	if text != "/kmuhub link" && text != "link" {
		return textMessage("Verfuegbare Befehle:\n- `/kmuhub link` - Konto verknuepfen\n")
	}

	ctx, _, err := h.tenantContext(ctx, incoming)
	if err != nil {
		return textMessage("Diese Teams-Organisation ist keinem Cosmi-Mandanten zugeordnet. Bitte im Cosmi-Admin unter Integrationen die Tenant-ID hinterlegen.")
	}

	token, err := h.linkService.GenerateLinkToken(ctx, integration.PlatformTeams, incoming.ExternalUserID)
	if err != nil {
		slog.Error("teams webhook: failed to generate link token", "error", err)
		return textMessage("Fehler beim Erstellen des Verknuepfungstokens. Bitte versuchen Sie es erneut.")
	}

	return textMessage("Ihr Verknuepfungstoken (gueltig fuer 5 Minuten):\n\n`" + token +
		"`\n\nBitte fuegen Sie diesen Token in KMU Hub unter Einstellungen > Integrationen > Konto verknuepfen ein.")
}

// tenantContext resolves the Teams organisation to a Cosmi tenant and attaches
// it. Every data access below this line runs under the resolved tenant.
func (h *WebhookHandler) tenantContext(ctx context.Context, incoming *IncomingAction) (context.Context, uuid.UUID, error) {
	workspaceID := activityTenantID(incoming.Activity)

	tenantID, err := h.tenants.ResolveTenant(ctx, integration.PlatformTeams, workspaceID)
	if err != nil {
		slog.Warn("teams webhook: organisation not mapped to a tenant",
			"teams_tenant_id", workspaceID,
			"error", err,
		)
		return ctx, uuid.Nil, err
	}
	return integration.WithTenant(ctx, tenantID), tenantID, nil
}

// activityTenantID reads channelData.tenant.id, the Azure AD tenant of the
// Teams organisation the activity came from.
func activityTenantID(act schema.Activity) string {
	tenant, ok := act.ChannelData["tenant"].(map[string]any)
	if !ok {
		return ""
	}
	id, _ := tenant["id"].(string)
	return id
}

// acknowledge marks the notification read for the linked user.
func (h *WebhookHandler) acknowledge(ctx context.Context, tenantID uuid.UUID, link *integration.AccountLink, notificationID string) error {
	notifID, err := uuid.Parse(notificationID)
	if err != nil {
		return err
	}
	if h.notifications == nil {
		return errors.New("notification service not wired")
	}
	_, err = h.notifications.MarkRead(ctx, tenantID, notifID, link.KMUHubUserID)
	return err
}

// invokeMessage builds the Bot Framework invoke response shape.
func invokeMessage(text string) integration.WebhookResponse {
	return integration.JSONResponse(http.StatusOK, map[string]any{
		"statusCode": 200,
		"type":       "application/vnd.microsoft.activity.message",
		"value":      text,
	})
}

// textMessage builds a plain Teams message response.
func textMessage(text string) integration.WebhookResponse {
	return integration.JSONResponse(http.StatusOK, map[string]any{
		"type": "message",
		"text": text,
	})
}
