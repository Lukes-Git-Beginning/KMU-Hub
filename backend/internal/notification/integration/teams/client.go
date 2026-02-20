package teams

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/infracloudio/msbotbuilder-go/core"
	"github.com/infracloudio/msbotbuilder-go/core/activity"
	"github.com/infracloudio/msbotbuilder-go/schema"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/integration"
)

// Client wraps the Teams Bot Framework adapter for sending and updating messages.
type Client struct {
	adapter core.Adapter
	appID   string
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
		adapter: adapter,
		appID:   appID,
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
