package slack

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	slackapi "github.com/slack-go/slack"

	"github.com/kmuhub/kmuhub/internal/notification/integration"
)

// WebhookHandler processes inbound Slack interactive message callbacks and slash commands.
//
// It runs inside the notification service, not the gateway: the signing secret,
// the integration repository and the tenant resolution all live here, and the
// gateway stays a proxy that never parses the payload.
type WebhookHandler struct {
	client        *Client
	signingSecret string
	repo          integration.Repository
	linkService   *integration.AccountLinkService
	tenants       integration.TenantResolver
	notifications integration.NotificationAcknowledger
}

// NewWebhookHandler creates a new Slack webhook handler.
func NewWebhookHandler(
	client *Client,
	signingSecret string,
	repo integration.Repository,
	linkService *integration.AccountLinkService,
	tenants integration.TenantResolver,
	notifications integration.NotificationAcknowledger,
) *WebhookHandler {
	return &WebhookHandler{
		client:        client,
		signingSecret: signingSecret,
		repo:          repo,
		linkService:   linkService,
		tenants:       tenants,
		notifications: notifications,
	}
}

// Process verifies and dispatches one inbound Slack request.
//
// Order is load-bearing: the signature is checked against the untouched request
// body before anything is parsed and long before any data is read. The previous
// implementation verified against url.Values.Encode() of the parsed form, which
// re-sorts and re-escapes the fields — a slash command carries a dozen fields
// and never survived that round trip, so every /kmuhub call answered 401.
func (h *WebhookHandler) Process(ctx context.Context, req integration.WebhookRequest) integration.WebhookResponse {
	if err := h.verifySignature(req); err != nil {
		slog.Warn("slack webhook: signature verification failed", "kind", req.Kind, "error", err)
		return integration.TextResponse(http.StatusUnauthorized, "unauthorized")
	}

	form, err := url.ParseQuery(string(req.Body))
	if err != nil {
		return integration.TextResponse(http.StatusBadRequest, "bad request")
	}

	switch req.Kind {
	case integration.WebhookKindSlackInteraction:
		return h.handleInteraction(ctx, form)
	case integration.WebhookKindSlackCommand:
		return h.handleSlashCommand(ctx, form)
	default:
		return integration.TextResponse(http.StatusBadRequest, "unknown webhook kind")
	}
}

// verifySignature checks the Slack request signature over the raw body.
func (h *WebhookHandler) verifySignature(req integration.WebhookRequest) error {
	if h.signingSecret == "" {
		return errors.New("slack signing secret not configured")
	}

	header := http.Header{}
	for name, value := range req.Headers {
		header.Set(name, value)
	}

	sv, err := slackapi.NewSecretsVerifier(header, h.signingSecret)
	if err != nil {
		return err
	}
	if _, err := sv.Write(req.Body); err != nil {
		return err
	}
	return sv.Ensure()
}

// ============================================================================
// Interactions (Block Kit button clicks)
// ============================================================================

func (h *WebhookHandler) handleInteraction(ctx context.Context, form url.Values) integration.WebhookResponse {
	payload := form.Get("payload")
	if payload == "" {
		return integration.TextResponse(http.StatusBadRequest, "missing payload")
	}

	var callback slackapi.InteractionCallback
	if err := json.Unmarshal([]byte(payload), &callback); err != nil {
		slog.Error("slack webhook: failed to parse interaction callback", "error", err)
		return integration.TextResponse(http.StatusBadRequest, "bad request")
	}

	if callback.Type != slackapi.InteractionTypeBlockActions || len(callback.ActionCallback.BlockActions) == 0 {
		return integration.EmptyResponse(http.StatusOK)
	}

	externalUserID := callback.User.ID

	tenantID, err := h.tenants.ResolveTenant(ctx, integration.PlatformSlack, callback.Team.ID)
	if err != nil {
		slog.Warn("slack webhook: workspace not mapped to a tenant",
			"team_id", callback.Team.ID,
			"error", err,
		)
		h.postEphemeral(ctx, callback.Channel.ID, externalUserID,
			"Dieser Slack-Workspace ist keinem Cosmi-Mandanten zugeordnet. Bitte im Cosmi-Admin unter Integrationen die Workspace-ID hinterlegen.")
		return integration.EmptyResponse(http.StatusOK)
	}
	ctx = integration.WithTenant(ctx, tenantID)

	link, err := h.repo.GetAccountLink(ctx, integration.PlatformSlack, externalUserID)
	if err != nil {
		h.postEphemeral(ctx, callback.Channel.ID, externalUserID,
			"Bitte verknuepfen Sie Ihr Konto mit dem Befehl: `/kmuhub link`")
		return integration.EmptyResponse(http.StatusOK)
	}

	action := callback.ActionCallback.BlockActions[0]
	actionType := extractActionType(action.ActionID)
	notificationID := action.Value

	// Only acknowledge has an effect in Cosmi today. Approve, reject and reply
	// have no counterpart yet -- the card must not claim otherwise, so they get
	// an honest ephemeral reply and the original message stays untouched.
	if actionType != string(integration.ActionAcknowledge) {
		slog.Info("slack webhook: unsupported action requested",
			"action_type", actionType,
			"notification_id", notificationID,
			"user_id", link.KMUHubUserID,
		)
		h.postEphemeral(ctx, callback.Channel.ID, externalUserID,
			"Diese Aktion wird von Cosmi noch nicht ausgefuehrt. Bitte erledigen Sie sie in Cosmi.")
		return integration.EmptyResponse(http.StatusOK)
	}

	if err := h.acknowledge(ctx, tenantID, link, notificationID); err != nil {
		slog.Error("slack webhook: acknowledge failed",
			"notification_id", notificationID,
			"user_id", link.KMUHubUserID,
			"error", err,
		)
		h.postEphemeral(ctx, callback.Channel.ID, externalUserID,
			"Die Benachrichtigung konnte nicht als gelesen markiert werden.")
		return integration.EmptyResponse(http.StatusOK)
	}

	// The card is only rewritten after the action actually landed.
	h.markCardResolved(ctx, &callback, link, actionType)

	return integration.EmptyResponse(http.StatusOK)
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

// markCardResolved rewrites the original message in place with a resolved banner.
func (h *WebhookHandler) markCardResolved(ctx context.Context, callback *slackapi.InteractionCallback, link *integration.AccountLink, actionType string) {
	if h.client == nil || callback.Channel.ID == "" || callback.MessageTs == "" {
		return
	}

	displayName := link.ExternalDisplayName
	if displayName == "" {
		displayName = "Benutzer"
	}

	updatedBlocks := buildResolvedBlocks(callback.Message.Text, actionType, displayName)
	if err := h.client.UpdateMessage(ctx, callback.Channel.ID, callback.MessageTs, updatedBlocks); err != nil {
		slog.Warn("slack webhook: failed to update message",
			"channel_id", callback.Channel.ID,
			"error", err,
		)
	}
}

func (h *WebhookHandler) postEphemeral(ctx context.Context, channelID, externalUserID, text string) {
	if h.client == nil || channelID == "" {
		return
	}
	if err := h.client.PostEphemeral(ctx, channelID, externalUserID, text); err != nil {
		slog.Warn("slack webhook: failed to post ephemeral", "channel_id", channelID, "error", err)
	}
}

// ============================================================================
// Slash commands (/kmuhub)
// ============================================================================

func (h *WebhookHandler) handleSlashCommand(ctx context.Context, form url.Values) integration.WebhookResponse {
	externalUserID := form.Get("user_id")
	teamID := form.Get("team_id")

	switch strings.TrimSpace(form.Get("text")) {
	case "link":
		return h.handleSlashLink(ctx, teamID, externalUserID)
	case "unlink":
		return h.handleSlashUnlink(ctx, teamID, externalUserID)
	default:
		return ephemeral("Verfuegbare Befehle:\n- `/kmuhub link` - Konto verknuepfen\n- `/kmuhub unlink` - Verknuepfung aufheben")
	}
}

func (h *WebhookHandler) handleSlashLink(ctx context.Context, teamID, externalUserID string) integration.WebhookResponse {
	ctx, resp, ok := h.tenantContext(ctx, teamID)
	if !ok {
		return resp
	}

	token, err := h.linkService.GenerateLinkToken(ctx, integration.PlatformSlack, externalUserID)
	if err != nil {
		slog.Error("slack slash: failed to generate link token", "error", err)
		return ephemeral("Fehler beim Erstellen des Verknuepfungstokens. Bitte versuchen Sie es erneut.")
	}

	return ephemeral("Ihr Verknuepfungstoken (gueltig fuer 5 Minuten):\n\n`" + token +
		"`\n\nBitte fuegen Sie diesen Token in KMU Hub unter Einstellungen > Integrationen > Konto verknuepfen ein.")
}

func (h *WebhookHandler) handleSlashUnlink(ctx context.Context, teamID, externalUserID string) integration.WebhookResponse {
	ctx, resp, ok := h.tenantContext(ctx, teamID)
	if !ok {
		return resp
	}

	link, err := h.repo.GetAccountLink(ctx, integration.PlatformSlack, externalUserID)
	if err != nil {
		return ephemeral("Kein verknuepftes Konto gefunden.")
	}

	if err := h.repo.DeleteAccountLink(ctx, link.ID); err != nil {
		slog.Error("slack slash: failed to unlink", "error", err)
		return ephemeral("Fehler beim Aufheben der Verknuepfung.")
	}

	return ephemeral("Konto-Verknuepfung erfolgreich aufgehoben.")
}

// tenantContext resolves the workspace to a tenant and attaches it. The bool is
// false when the caller must return the accompanying response untouched — no
// data access may happen without a resolved tenant.
func (h *WebhookHandler) tenantContext(ctx context.Context, teamID string) (context.Context, integration.WebhookResponse, bool) {
	tenantID, err := h.tenants.ResolveTenant(ctx, integration.PlatformSlack, teamID)
	if err != nil {
		slog.Warn("slack slash: workspace not mapped to a tenant", "team_id", teamID, "error", err)
		return ctx, ephemeral("Dieser Slack-Workspace ist keinem Cosmi-Mandanten zugeordnet. Bitte im Cosmi-Admin unter Integrationen die Workspace-ID hinterlegen."), false
	}
	return integration.WithTenant(ctx, tenantID), integration.WebhookResponse{}, true
}

// ephemeral builds a Slack ephemeral message response.
func ephemeral(text string) integration.WebhookResponse {
	return integration.JSONResponse(http.StatusOK, map[string]any{
		"response_type": "ephemeral",
		"text":          text,
	})
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
