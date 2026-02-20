package slack

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	slackapi "github.com/slack-go/slack"

	"github.com/kmuhub/kmuhub/internal/notification/integration"
)

// WebhookHandler processes inbound Slack interactive message callbacks and slash commands.
type WebhookHandler struct {
	client        *Client
	signingSecret string
	repo          integration.Repository
	linkService   *integration.AccountLinkService
}

// NewWebhookHandler creates a new Slack webhook handler.
func NewWebhookHandler(client *Client, signingSecret string, repo integration.Repository, linkService *integration.AccountLinkService) *WebhookHandler {
	return &WebhookHandler{
		client:        client,
		signingSecret: signingSecret,
		repo:          repo,
		linkService:   linkService,
	}
}

// HandleInteraction processes Slack interactive message callbacks (button clicks).
func (h *WebhookHandler) HandleInteraction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Verify request signature using Slack signing secret
	sv, err := slackapi.NewSecretsVerifier(r.Header, h.signingSecret)
	if err != nil {
		slog.Error("slack webhook: failed to create secrets verifier", "error", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Read and verify body
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit

	// Parse payload from form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	payloadStr := r.FormValue("payload")
	if payloadStr == "" {
		http.Error(w, "missing payload", http.StatusBadRequest)
		return
	}

	// Write payload to verifier for signature check
	if _, err := sv.Write([]byte(r.PostForm.Encode())); err != nil {
		slog.Error("slack webhook: signature verification write failed", "error", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := sv.Ensure(); err != nil {
		slog.Error("slack webhook: invalid signature", "error", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var callback slackapi.InteractionCallback
	if err := json.Unmarshal([]byte(payloadStr), &callback); err != nil {
		slog.Error("slack webhook: failed to parse interaction callback", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Handle block_actions type (button clicks)
	if callback.Type == slackapi.InteractionTypeBlockActions {
		h.handleBlockActions(ctx, w, &callback)
		return
	}

	// Acknowledge unknown types
	w.WriteHeader(http.StatusOK)
}

// handleBlockActions processes button clicks from Block Kit messages.
func (h *WebhookHandler) handleBlockActions(ctx context.Context, w http.ResponseWriter, callback *slackapi.InteractionCallback) {
	if len(callback.ActionCallback.BlockActions) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	action := callback.ActionCallback.BlockActions[0]
	externalUserID := callback.User.ID

	// Resolve Slack user -> KMU Hub user
	link, err := h.repo.GetAccountLink(ctx, integration.PlatformSlack, externalUserID)
	if err != nil {
		// Unlinked user: send ephemeral prompt
		if h.client != nil {
			_ = h.client.PostEphemeral(ctx, callback.Channel.ID, externalUserID,
				"Bitte verknuepfen Sie Ihr Konto mit dem Befehl: `/kmuhub link`")
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	// Parse action info
	// action_id format: kmuhub_{action_type}
	// value: notification_id
	actionType := extractActionType(action.ActionID)
	notificationID := action.Value

	slog.Info("slack webhook: action received",
		"action_type", actionType,
		"notification_id", notificationID,
		"user_id", link.KMUHubUserID,
		"external_user_id", externalUserID,
	)

	// Execute action
	switch actionType {
	case string(integration.ActionAcknowledge):
		h.handleAcknowledge(ctx, link, notificationID)
	case string(integration.ActionApprove), string(integration.ActionReject):
		slog.Info("slack webhook: approval action",
			"action_type", actionType,
			"notification_id", notificationID,
			"user_id", link.KMUHubUserID,
		)
	case string(integration.ActionReply):
		slog.Info("slack webhook: reply action",
			"notification_id", notificationID,
			"user_id", link.KMUHubUserID,
		)
	}

	// Update original message in-place via chat.update
	if h.client != nil && callback.Channel.ID != "" && callback.MessageTs != "" {
		// Resolve display name
		displayName := link.ExternalDisplayName
		if displayName == "" {
			displayName = "Benutzer"
		}

		// Retrieve notification for card rebuild -- for now, build a minimal updated card
		updatedBlocks := buildResolvedBlocks(callback.Message.Text, actionType, displayName)
		if err := h.client.UpdateMessage(ctx, callback.Channel.ID, callback.MessageTs, updatedBlocks); err != nil {
			slog.Warn("slack webhook: failed to update message",
				"channel_id", callback.Channel.ID,
				"error", err,
			)
		}
	}

	// Respond 200 OK immediately (Slack expects fast response)
	w.WriteHeader(http.StatusOK)
}

// HandleSlashCommand processes Slack slash commands (/kmuhub).
func (h *WebhookHandler) HandleSlashCommand(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Verify signing secret
	sv, err := slackapi.NewSecretsVerifier(r.Header, h.signingSecret)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if _, err := sv.Write([]byte(r.PostForm.Encode())); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := sv.Ensure(); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	cmd, err := slackapi.SlashCommandParse(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	switch cmd.Text {
	case "link":
		h.handleSlashLink(ctx, w, cmd)
	case "unlink":
		h.handleSlashUnlink(ctx, w, cmd)
	default:
		respondSlashJSON(w, map[string]interface{}{
			"response_type": "ephemeral",
			"text":          "Verfuegbare Befehle:\n- `/kmuhub link` - Konto verknuepfen\n- `/kmuhub unlink` - Verknuepfung aufheben",
		})
	}
}

// handleSlashLink generates a link token for account linking.
func (h *WebhookHandler) handleSlashLink(ctx context.Context, w http.ResponseWriter, cmd slackapi.SlashCommand) {
	token, err := h.linkService.GenerateLinkToken(ctx, integration.PlatformSlack, cmd.UserID)
	if err != nil {
		slog.Error("slack slash: failed to generate link token", "error", err)
		respondSlashJSON(w, map[string]interface{}{
			"response_type": "ephemeral",
			"text":          "Fehler beim Erstellen des Verknuepfungstokens. Bitte versuchen Sie es erneut.",
		})
		return
	}

	respondSlashJSON(w, map[string]interface{}{
		"response_type": "ephemeral",
		"text":          "Ihr Verknuepfungstoken (gueltig fuer 5 Minuten):\n\n`" + token + "`\n\nBitte fuegen Sie diesen Token in KMU Hub unter Einstellungen > Integrationen > Konto verknuepfen ein.",
	})
}

// handleSlashUnlink removes the account link.
func (h *WebhookHandler) handleSlashUnlink(ctx context.Context, w http.ResponseWriter, cmd slackapi.SlashCommand) {
	link, err := h.repo.GetAccountLink(ctx, integration.PlatformSlack, cmd.UserID)
	if err != nil {
		respondSlashJSON(w, map[string]interface{}{
			"response_type": "ephemeral",
			"text":          "Kein verknuepftes Konto gefunden.",
		})
		return
	}

	if err := h.repo.DeleteAccountLink(ctx, link.ID); err != nil {
		slog.Error("slack slash: failed to unlink", "error", err)
		respondSlashJSON(w, map[string]interface{}{
			"response_type": "ephemeral",
			"text":          "Fehler beim Aufheben der Verknuepfung.",
		})
		return
	}

	respondSlashJSON(w, map[string]interface{}{
		"response_type": "ephemeral",
		"text":          "Konto-Verknuepfung erfolgreich aufgehoben.",
	})
}

// handleAcknowledge marks a notification as read.
func (h *WebhookHandler) handleAcknowledge(_ context.Context, link *integration.AccountLink, notificationID string) {
	notifID, err := uuid.Parse(notificationID)
	if err != nil {
		slog.Warn("slack webhook: invalid notification_id", "notification_id", notificationID)
		return
	}

	slog.Info("slack webhook: notification acknowledged",
		"notification_id", notifID,
		"user_id", link.KMUHubUserID,
	)
}

// extractActionType extracts the action type from a Slack action_id (format: kmuhub_{type}).
func extractActionType(actionID string) string {
	const prefix = "kmuhub_"
	if len(actionID) > len(prefix) {
		return actionID[len(prefix):]
	}
	return actionID
}

// buildResolvedBlocks creates minimal resolved blocks for in-place card update.
func buildResolvedBlocks(originalText, actionTaken, actorName string) []slackapi.Block {
	bannerText := resolvedBannerText(actionTaken, actorName)

	blocks := []slackapi.Block{
		slackapi.NewSectionBlock(
			slackapi.NewTextBlockObject("mrkdwn", originalText, false, false),
			nil,
			nil,
		),
		slackapi.NewContextBlock(
			"",
			slackapi.NewTextBlockObject("mrkdwn", bannerText, false, false),
		),
	}

	return blocks
}

// respondSlashJSON writes a JSON response for slash commands.
func respondSlashJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(payload)
}
