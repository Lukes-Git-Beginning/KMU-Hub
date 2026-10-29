package action

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/event"
)

// EventTypeAutomationNotification is the event type the notification.send
// action publishes. ModuleID is event.ModuleAutomation, which the automation
// consumer and engine skip -- the action's own event never starts a workflow.
const EventTypeAutomationNotification = "automation.notification"

// EventPublisher puts an event on the bus (event.EmitEvent in production).
type EventPublisher func(ctx context.Context, payload models.EventPayload) error

// SendNotificationAction delivers a notification by publishing an event on the
// bus. The notification service's InboxConsumer turns it into an inbox message
// for the target user and pushes it over WebSocket. The consumer's insert only
// lands for a user of the event's tenant (inbox message repository Create).
type SendNotificationAction struct {
	publish EventPublisher
}

// NewSendNotificationAction creates a new SendNotificationAction.
func NewSendNotificationAction(publish EventPublisher) *SendNotificationAction {
	return &SendNotificationAction{publish: publish}
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
	if id, err := uuid.Parse(targetUserID); err != nil || id == uuid.Nil {
		err := fmt.Errorf("target_user_id %q is not a user id", targetUserID)
		return ActionResult{Success: false, Error: err.Error()}, err
	}
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return ActionResult{Success: false, Error: "no tenant in context"}, err
	}
	if a.publish == nil {
		err := fmt.Errorf("notification.send: no event publisher configured")
		return ActionResult{Success: false, Error: err.Error()}, err
	}

	title := resolveTemplate(cfg.Title, env)
	body := resolveTemplate(cfg.Body, env)

	// The id doubles as ResourceID, which the consumer stores as the inbox
	// message's source_id (its dedup key).
	// lean: one fresh id per execution; key it on execution+step once the engine
	// retries actions, so a retry does not deliver twice.
	notificationID := uuid.New().String()

	if err := a.publish(ctx, models.EventPayload{
		Type:          EventTypeAutomationNotification,
		TenantID:      tenantID,
		Priority:      models.PriorityNormal,
		ModuleID:      event.ModuleAutomation,
		ResourceID:    notificationID,
		TargetUserIDs: []string{targetUserID},
		Title:         title,
		Body:          body,
		DeepLink:      resolveTemplate(cfg.DeepLink, env),
		Timestamp:     time.Now().UTC(),
	}); err != nil {
		return ActionResult{Success: false, Error: "publish notification: " + err.Error()}, err
	}

	slog.Info("automation notification published",
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
