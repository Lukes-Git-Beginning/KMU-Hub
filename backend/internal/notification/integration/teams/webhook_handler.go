package teams

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/infracloudio/msbotbuilder-go/schema"

	"github.com/kmuhub/kmuhub/internal/notification/integration"
)

// NotificationService is the interface for marking notifications read, etc.
type NotificationService interface {
	MarkReadByID(ctx context.Context, notifID, userID uuid.UUID) error
}

// WebhookHandler processes inbound Teams Bot Framework activities.
type WebhookHandler struct {
	client      *Client
	repo        integration.Repository
	linkService *integration.AccountLinkService
}

// NewWebhookHandler creates a new Teams webhook handler.
func NewWebhookHandler(client *Client, repo integration.Repository, linkService *integration.AccountLinkService) *WebhookHandler {
	return &WebhookHandler{
		client:      client,
		repo:        repo,
		linkService: linkService,
	}
}

// HandleWebhook processes incoming Teams Bot Framework activities.
// Teams calls this endpoint for all bot interactions: messages, invoke (Action.Execute), etc.
func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse and verify the incoming activity (JWT verification by adapter)
	incoming, err := h.client.HandleIncomingActivity(ctx, r)
	if err != nil {
		slog.Error("teams webhook: failed to parse activity", "error", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	act := incoming.Activity

	switch act.Type {
	case schema.Invoke:
		h.handleInvoke(ctx, w, incoming)
	case schema.Message:
		h.handleMessage(ctx, w, incoming)
	default:
		// Acknowledge other activity types with 200 OK
		w.WriteHeader(http.StatusOK)
	}
}

// handleInvoke processes Action.Execute (adaptiveCard/action) invocations.
func (h *WebhookHandler) handleInvoke(ctx context.Context, w http.ResponseWriter, incoming *IncomingAction) {
	// Resolve external user -> KMU Hub user
	link, err := h.repo.GetAccountLink(ctx, integration.PlatformTeams, incoming.ExternalUserID)
	if err != nil {
		// Unlinked user: respond with link instructions card
		slog.Info("teams webhook: unlinked user attempted action",
			"external_user_id", incoming.ExternalUserID,
		)
		respondWithLinkPrompt(w)
		return
	}

	// Execute action based on type
	switch incoming.ActionType {
	case string(integration.ActionAcknowledge):
		h.handleAcknowledge(ctx, link, incoming)
	case string(integration.ActionApprove), string(integration.ActionReject):
		// For v1, log the action -- module-specific routing will be added via automation engine
		slog.Info("teams webhook: approval action received",
			"action_type", incoming.ActionType,
			"notification_id", incoming.NotificationID,
			"user_id", link.KMUHubUserID,
		)
	case string(integration.ActionReply):
		slog.Info("teams webhook: reply action received",
			"notification_id", incoming.NotificationID,
			"reply_text", incoming.ReplyText,
			"user_id", link.KMUHubUserID,
		)
	}

	// Respond within 5 seconds per Teams requirement with 200 OK
	// The updated card will be sent asynchronously if needed
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"statusCode": 200,
		"type":       "application/vnd.microsoft.activity.message",
		"value":      "Aktion verarbeitet",
	})
}

// handleMessage processes text messages (e.g., /kmuhub link command).
func (h *WebhookHandler) handleMessage(ctx context.Context, w http.ResponseWriter, incoming *IncomingAction) {
	text := incoming.Activity.Text

	if text == "/kmuhub link" || text == "link" {
		h.handleLinkCommand(ctx, w, incoming)
		return
	}

	// Default: respond with help text
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"type": "message",
		"text": "Verfuegbare Befehle:\n- `/kmuhub link` - Konto verknuepfen\n",
	})
}

// handleLinkCommand generates a link token and responds with instructions.
func (h *WebhookHandler) handleLinkCommand(ctx context.Context, w http.ResponseWriter, incoming *IncomingAction) {
	token, err := h.linkService.GenerateLinkToken(ctx, integration.PlatformTeams, incoming.ExternalUserID)
	if err != nil {
		slog.Error("teams webhook: failed to generate link token", "error", err)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "message",
			"text": "Fehler beim Erstellen des Verknuepfungstokens. Bitte versuchen Sie es erneut.",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"type": "message",
		"text": "Ihr Verknuepfungstoken (gueltig fuer 5 Minuten):\n\n`" + token + "`\n\nBitte fuegen Sie diesen Token in KMU Hub unter Einstellungen > Integrationen > Konto verknuepfen ein.",
	})
}

// handleAcknowledge marks a notification as read.
func (h *WebhookHandler) handleAcknowledge(ctx context.Context, link *integration.AccountLink, incoming *IncomingAction) {
	notifID, err := uuid.Parse(incoming.NotificationID)
	if err != nil {
		slog.Warn("teams webhook: invalid notification_id", "notification_id", incoming.NotificationID)
		return
	}

	// Log the acknowledgement -- actual MarkRead is done via gRPC from gateway
	slog.Info("teams webhook: notification acknowledged",
		"notification_id", notifID,
		"user_id", link.KMUHubUserID,
	)
}

// respondWithLinkPrompt sends a card telling unlinked users to link their account.
func respondWithLinkPrompt(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"statusCode": 200,
		"type":       "application/vnd.microsoft.activity.message",
		"value":      "Bitte verknuepfen Sie Ihr Konto mit dem Befehl: /kmuhub link",
	})
}
