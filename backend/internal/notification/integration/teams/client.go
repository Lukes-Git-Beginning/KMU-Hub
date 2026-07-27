package teams

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/infracloudio/msbotbuilder-go/connector/auth"
	"github.com/infracloudio/msbotbuilder-go/core"
	"github.com/infracloudio/msbotbuilder-go/core/activity"
	"github.com/infracloudio/msbotbuilder-go/schema"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/integration"
)

// Client wraps the Teams Bot Framework adapter for sending and updating messages.
type Client struct {
	adapter     core.Adapter
	appID       string
	appPassword string

	// tokenURL is the Bot Framework login endpoint the connection probe posts to.
	// It is a field so a test can point it at a local server; production always
	// gets the value NewClient computes.
	tokenURL   string
	httpClient *http.Client
}

// NewClient creates a new Teams Bot Framework client.
func NewClient(appID, appPassword string) (*Client, error) {
	setting := core.AdapterSetting{
		AppID:       appID,
		AppPassword: appPassword,
	}

	adapter, err := core.NewBotAdapter(setting)
	if err != nil {
		return nil, fmt.Errorf("create bot adapter: %w", err)
	}

	return &Client{
		adapter:     adapter,
		appID:       appID,
		appPassword: appPassword,
		tokenURL: auth.ToChannelFromBotLoginURLPrefix + auth.DefaultChannelAuthTenant +
			auth.ToChannelFromBotTokenEndpointPathTOCHANNELFROMBOTTOKENENDPOINTPATH,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// ProbeConnection verifies the bot credentials by requesting a channel token
// from the Bot Framework login endpoint — the same client_credentials exchange
// the connector performs before every send, minus the send. It touches no
// conversation, so it is safe to run from the setup wizard.
func (c *Client) ProbeConnection(ctx context.Context) (*integration.ProbeResult, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", c.appID)
	form.Set("client_secret", c.appPassword)
	form.Set("scope", auth.ToChannelFromBotOauthScope)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("reach bot framework login endpoint: %w", err)
	}
	defer resp.Body.Close()

	// The error body carries AAD's reason (invalid_client, unauthorized_client);
	// it is bounded because an error page is not worth unbounded memory.
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bot framework rejected the credentials (%s): %s",
			resp.Status, strings.TrimSpace(string(body)))
	}

	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &token); err != nil || token.AccessToken == "" {
		return nil, fmt.Errorf("bot framework returned no access token")
	}

	return &integration.ProbeResult{
		Detail: fmt.Sprintf("bot framework issued a channel token for app %s", c.appID),
	}, nil
}

// PostNotification sends a notification as an Adaptive Card to a Teams channel.
func (c *Client) PostNotification(ctx context.Context, mapping *integration.ChannelMapping, notif *models.Notification, actions []integration.ActionType) (*integration.DeliveryResult, error) {
	// Build Adaptive Card
	card := BuildCard(notif, actions)
	cardJSON, err := json.Marshal(card)
	if err != nil {
		return nil, fmt.Errorf("marshal adaptive card: %w", err)
	}

	// Parse conversation reference from mapping's platform_data
	var convRef schema.ConversationReference
	if err := json.Unmarshal(mapping.PlatformData, &convRef); err != nil {
		return nil, fmt.Errorf("parse conversation reference: %w", err)
	}

	// Build attachments for the message
	attachments := []schema.Attachment{
		{
			ContentType: "application/vnd.microsoft.card.adaptive",
			Content:     json.RawMessage(cardJSON),
		},
	}

	// Send proactive message via Bot Framework adapter using MsgOption
	err = c.adapter.ProactiveMessage(ctx, convRef, activity.HandlerFuncs{
		OnMessageFunc: func(turn *activity.TurnContext) (schema.Activity, error) {
			return turn.SendActivity(activity.MsgOptionAttachments(attachments))
		},
	})
	if err != nil {
		return nil, fmt.Errorf("send proactive message: %w", err)
	}

	slog.Debug("teams notification sent",
		"notification_id", notif.ID,
		"channel_id", mapping.ChannelID,
	)

	return &integration.DeliveryResult{
		PlatformMessageID: "", // Bot Framework proactive doesn't return activity ID directly
		Success:           true,
	}, nil
}

// UpdateCard updates an existing card in-place after user interaction.
func (c *Client) UpdateCard(ctx context.Context, mapping *integration.ChannelMapping, activityID string, notif *models.Notification, actionTaken, actorName string) error {
	card := BuildUpdatedCard(notif, actionTaken, actorName)
	cardJSON, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("marshal updated card: %w", err)
	}

	// Parse conversation reference
	var convRef schema.ConversationReference
	if err := json.Unmarshal(mapping.PlatformData, &convRef); err != nil {
		return fmt.Errorf("parse conversation reference: %w", err)
	}

	attachments := []schema.Attachment{
		{
			ContentType: "application/vnd.microsoft.card.adaptive",
			Content:     json.RawMessage(cardJSON),
		},
	}

	err = c.adapter.ProactiveMessage(ctx, convRef, activity.HandlerFuncs{
		OnMessageFunc: func(turn *activity.TurnContext) (schema.Activity, error) {
			return turn.SendActivity(activity.MsgOptionAttachments(attachments))
		},
	})
	if err != nil {
		return fmt.Errorf("update card: %w", err)
	}

	return nil
}

// HandleIncomingActivity processes an incoming request from Teams Bot Framework.
// It verifies the JWT token and parses the action payload.
func (c *Client) HandleIncomingActivity(ctx context.Context, r *http.Request) (*IncomingAction, error) {
	act, err := c.adapter.ParseRequest(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("parse teams request: %w", err)
	}

	result := &IncomingAction{
		Activity:       act,
		ExternalUserID: act.From.ID,
	}

	// Parse action data from Activity.Value for invoke/action activities
	if act.Value != nil {
		valueBytes, err := json.Marshal(act.Value)
		if err == nil {
			var actionData ActionData
			if err := json.Unmarshal(valueBytes, &actionData); err == nil {
				result.ActionType = actionData.ActionType
				result.NotificationID = actionData.NotificationID
				result.ReplyText = actionData.ReplyText
			}
		}
	}

	return result, nil
}

// IncomingAction represents a parsed incoming action from Teams.
type IncomingAction struct {
	Activity       schema.Activity
	ExternalUserID string
	ActionType     string
	NotificationID string
	ReplyText      string
}

// ActionData is the data payload embedded in Action.Execute buttons.
type ActionData struct {
	ActionType     string `json:"action_type"`
	NotificationID string `json:"notification_id"`
	ReplyText      string `json:"reply_text,omitempty"`
}
