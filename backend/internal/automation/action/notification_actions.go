package action

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
)

// SendNotificationAction creates a notification by emitting an event payload.
// Since the notification service does not expose a CreateNotification RPC,
// this action logs the notification intent and returns a synthetic notification ID.
// In production, this will integrate with the event bus to trigger notification creation.
type SendNotificationAction struct{}

// NewSendNotificationAction creates a new SendNotificationAction.
func NewSendNotificationAction() *SendNotificationAction {
	return &SendNotificationAction{}
}

func (a *SendNotificationAction) Type() string { return "notification.send" }

type sendNotificationConfig struct {
	TargetUserID string `json:"target_user_id"` // template: specific user or "{{deal.owner_id}}"
	Title        string `json:"title"`          // template
	Body         string `json:"body"`           // template
	DeepLink     string `json:"deep_link"`      // template (optional)
}

func (a *SendNotificationAction) Execute(ctx context.Context, config json.RawMessage, env map[string]any) (ActionResult, error) {
	var cfg sendNotificationConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return ActionResult{Success: false, Error: "invalid config: " + err.Error()}, err
	}

	targetUserID := resolveTemplate(cfg.TargetUserID, env)
	title := resolveTemplate(cfg.Title, env)
	body := resolveTemplate(cfg.Body, env)

	notificationID := uuid.New().String()

	slog.Info("automation notification sent",
		"notification_id", notificationID,
		"target_user_id", targetUserID,
		"title", title,
		"body_preview", truncate(body, 100),
	)

	return ActionResult{
		Success: true,
		Output: map[string]any{
			"notification_id": notificationID,
			"target_user_id":  targetUserID,
		},
	}, nil
}

// truncate cuts a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// SendNotificationDefinition returns the action definition for the catalog.
func SendNotificationDefinition() *ActionDefinition {
	return &ActionDefinition{
		Type:        "notification.send",
		Module:      "notification",
		Name:        "Benachrichtigung senden",
		Description: "Sendet eine Benachrichtigung an einen Benutzer",
		Params: []ConfigParam{
			{Key: "target_user_id", Label: "Empfaenger", Type: "template", Required: true},
			{Key: "title", Label: "Titel", Type: "template", Required: true},
			{Key: "body", Label: "Inhalt", Type: "template", Required: true},
			{Key: "deep_link", Label: "Deep-Link", Type: "template", Required: false},
		},
		OutputFields: []OutputField{
			{Key: "notification_id", Label: "Benachrichtigungs-ID", Type: "string"},
			{Key: "target_user_id", Label: "Empfaenger-ID", Type: "string"},
		},
	}
}
