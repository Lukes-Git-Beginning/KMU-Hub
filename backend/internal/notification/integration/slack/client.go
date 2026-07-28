package slack

import (
	"context"
	"fmt"
	"log/slog"

	slackapi "github.com/slack-go/slack"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/integration"
)

// Client wraps the Slack API for posting and updating messages.
type Client struct {
	api      *slackapi.Client
	botToken string
}

// NewClient creates a new Slack API client.
func NewClient(botToken string) *Client {
	return &Client{
		api:      slackapi.New(botToken),
		botToken: botToken,
	}
}

// ProbeConnection verifies the bot token against Slack's auth.test endpoint.
// auth.test is the cheapest call that actually reaches Slack with the token and
// it writes nothing into any channel, which is what an admin pressing "test
// connection" expects.
func (c *Client) ProbeConnection(ctx context.Context) (*integration.ProbeResult, error) {
	resp, err := c.api.AuthTestContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("slack auth.test: %w", err)
	}

	return &integration.ProbeResult{
		Detail: fmt.Sprintf("authenticated as %s in workspace %s", resp.User, resp.Team),
	}, nil
}

// PostNotification sends a notification as a Block Kit message to a Slack channel.
func (c *Client) PostNotification(ctx context.Context, mapping *integration.ChannelMapping, notif *models.Notification, actions []integration.ActionType) (*integration.DeliveryResult, error) {
	blocks := BuildBlocks(notif, actions)

	// Text fallback for notifications/non-block-kit clients
	fallbackText := notif.Title
	if notif.Body != nil {
		fallbackText = fmt.Sprintf("%s: %s", notif.Title, *notif.Body)
	}

	// Get module color for attachment sidebar
	moduleColor := integration.ModuleColors[notif.ModuleID]
	if moduleColor == "" {
		moduleColor = "#6b7280"
	}

	_, ts, err := c.api.PostMessageContext(
		ctx,
		mapping.ChannelID,
		slackapi.MsgOptionBlocks(blocks...),
		slackapi.MsgOptionText(fallbackText, false),
		slackapi.MsgOptionAttachments(slackapi.Attachment{
			Color: moduleColor,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("post slack message: %w", err)
	}

	slog.Debug("slack notification sent",
		"notification_id", notif.ID,
		"channel_id", mapping.ChannelID,
		"message_ts", ts,
	)

	return &integration.DeliveryResult{
		PlatformMessageID: ts,
		Success:           true,
	}, nil
}

// UpdateMessage updates an existing message in-place for card updates after interaction.
func (c *Client) UpdateMessage(ctx context.Context, channelID, messageTs string, blocks []slackapi.Block) error {
	_, _, _, err := c.api.UpdateMessageContext(
		ctx,
		channelID,
		messageTs,
		slackapi.MsgOptionBlocks(blocks...),
	)
	if err != nil {
		return fmt.Errorf("update slack message: %w", err)
	}
	return nil
}

// PostEphemeral sends an ephemeral (private) message to a specific user in a channel.
// Used for "please link your account" prompts to unlinked users.
func (c *Client) PostEphemeral(ctx context.Context, channelID, userID, text string) error {
	_, err := c.api.PostEphemeralContext(
		ctx,
		channelID,
		userID,
		slackapi.MsgOptionText(text, false),
	)
	if err != nil {
		return fmt.Errorf("post ephemeral message: %w", err)
	}
	return nil
}
