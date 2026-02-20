package slack

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	slackapi "github.com/slack-go/slack"
)

// OAuthHandler handles the Slack OAuth install flow.
type OAuthHandler struct {
	clientID     string
	clientSecret string
	redirectURL  string
}

// NewOAuthHandler creates a new Slack OAuth handler.
func NewOAuthHandler(clientID, clientSecret, redirectURL string) *OAuthHandler {
	return &OAuthHandler{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
	}
}

// Bot scopes required for KMU Hub Slack integration.
const slackBotScopes = "chat:write,chat:write.public,commands,incoming-webhook,users:read"

// HandleInstall redirects to Slack OAuth authorization URL.
func (h *OAuthHandler) HandleInstall(w http.ResponseWriter, r *http.Request) {
	authURL := fmt.Sprintf(
		"https://slack.com/oauth/v2/authorize?client_id=%s&scope=%s&redirect_uri=%s",
		url.QueryEscape(h.clientID),
		url.QueryEscape(slackBotScopes),
		url.QueryEscape(h.redirectURL),
	)

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// HandleCallback exchanges the OAuth code for a bot token.
// In production, the bot_access_token should be stored encrypted via vault key reference.
func (h *OAuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code parameter", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	resp, err := slackapi.GetOAuthV2Response(
		http.DefaultClient,
		h.clientID,
		h.clientSecret,
		code,
		h.redirectURL,
	)
	if err != nil {
		slog.Error("slack oauth exchange failed", "error", err)
		http.Error(w, "oauth exchange failed", http.StatusInternalServerError)
		return
	}

	slog.Info("slack oauth completed",
		"team_id", resp.Team.ID,
		"team_name", resp.Team.Name,
		"bot_user_id", resp.BotUserID,
	)

	// NOTE: In production, store resp.AccessToken via vault and update integration config.
	// For now, log success and redirect back to settings.
	// The frontend will call the CreateIntegrationConfig endpoint with the vault key.

	// Redirect back to KMU Hub integrations settings
	http.Redirect(w, r, "/settings?tab=integrations&slack=connected", http.StatusTemporaryRedirect)
}
