package event

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
)

// EventHandler is a function that processes an event payload.
type EventHandler func(ctx context.Context, event models.EventPayload) error

// EventRepository provides access to the events durability table.
type EventRepository interface {
	CreateEvent(ctx context.Context, event *models.Event) error
	ListUnprocessed(ctx context.Context, limit int) ([]models.Event, error)
	MarkProcessed(ctx context.Context, eventID string) error
}

// BusOption configures the event bus.
type BusOption func(*EventBus)

// WithReconnectWait sets the duration to wait before reconnecting.
func WithReconnectWait(d time.Duration) BusOption {
	return func(b *EventBus) {
		b.reconnectWait = d
	}
}

// EventBus listens on PostgreSQL LISTEN/NOTIFY and dispatches events to registered handlers.
type EventBus struct {
	connString    string
	handlers      map[string][]EventHandler
	mu            sync.RWMutex
	reconnectWait time.Duration
}

// NewEventBus creates a new event bus that listens on the "events" channel.
func NewEventBus(connString string, opts ...BusOption) *EventBus {
	bus := &EventBus{
		connString:    connString,
		handlers:      make(map[string][]EventHandler),
		reconnectWait: 5 * time.Second,
	}
	for _, opt := range opts {
		opt(bus)
	}
	return bus
}

// RegisterHandler registers a handler for a specific event type.
// Use "*" as the event type to receive all events (wildcard handler).
func (bus *EventBus) RegisterHandler(eventType string, handler EventHandler) {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	bus.handlers[eventType] = append(bus.handlers[eventType], handler)
}

// Listen starts the event bus listener with automatic reconnection.
// This method blocks until the context is cancelled.
func (bus *EventBus) Listen(ctx context.Context) error {
	for {
		err := bus.listenLoop(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		slog.Warn("event bus connection lost, reconnecting",
			"error", err,
			"retry_in", bus.reconnectWait,
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(bus.reconnectWait):
		}
	}
}

// listenLoop connects to PostgreSQL, issues LISTEN, and processes notifications.
func (bus *EventBus) listenLoop(ctx context.Context) error {
	conn, err := pgx.Connect(ctx, bus.connString)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, "LISTEN "+PGNotifyChannel)
	if err != nil {
		return err
	}

	slog.Info("event bus listening", "channel", PGNotifyChannel)

	for {
		notification, err := conn.WaitForNotification(ctx)
		if err != nil {
			return err
		}

		var event models.EventPayload
		if err := json.Unmarshal([]byte(notification.Payload), &event); err != nil {
			slog.Error("failed to unmarshal event payload",
				"error", err,
				"payload", notification.Payload,
			)
			continue
		}

		bus.dispatch(ctx, event)
	}
}

// dispatch sends the event to all matching handlers and wildcard handlers.
//
// The context is stamped with the event's tenant first. Handlers run on the
// listener's background context, which carries no request and therefore no
// tenant — and every table they touch (events since 000271, notifications since
// 000122, inbox_messages) is under RLS, whose policy admits nothing when
// app.tenant_id is empty. Without this the notification insert fails with
// "new row violates row-level security policy", ProcessEvent logs it and moves
// on, and the event disappears. Stamping here rather than in each handler keeps
// the every-handler guarantee: a new consumer registered on this bus is
// tenant-scoped by construction.
func (bus *EventBus) dispatch(ctx context.Context, event models.EventPayload) {
	if event.TenantID != uuid.Nil {
		ctx = context.WithValue(ctx, middleware.TenantIDKey, event.TenantID.String())
	}

	bus.mu.RLock()
	handlers := make([]EventHandler, 0)

	// Specific handlers for this event type
	if h, ok := bus.handlers[event.Type]; ok {
		handlers = append(handlers, h...)
	}

	// Wildcard handlers
	if h, ok := bus.handlers["*"]; ok {
		handlers = append(handlers, h...)
	}
	bus.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			slog.Error("event handler failed",
				"event_type", event.Type,
				"error", err,
			)
		}
	}
}

// ProcessBacklog processes unprocessed events from the durability table.
// This is the catch-up mechanism for events missed during downtime.
func (bus *EventBus) ProcessBacklog(ctx context.Context, repo EventRepository) error {
	events, err := repo.ListUnprocessed(ctx, 1000)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	slog.Info("processing event backlog", "count", len(events))

	for _, evt := range events {
		// TenantID comes from the stored row (column added in 000271). Before it
		// existed this replay could not restore the tenant, so every event
		// caught up after a restart reached the handlers as uuid.Nil.
		payload := models.EventPayload{
			Type:      evt.EventTypeKey,
			TenantID:  evt.TenantID,
			Priority:  evt.Priority,
			ModuleID:  evt.ModuleID,
			Timestamp: evt.CreatedAt,
		}

		if evt.ActorID != nil {
			payload.ActorID = evt.ActorID.String()
		}
		if evt.ResourceID != nil {
			payload.ResourceID = *evt.ResourceID
		}

		// Parse the stored payload for additional fields
		if evt.Payload != nil {
			var extra map[string]json.RawMessage
			if err := json.Unmarshal(evt.Payload, &extra); err == nil {
				if v, ok := extra["target_user_ids"]; ok {
					_ = json.Unmarshal(v, &payload.TargetUserIDs)
				}
				if v, ok := extra["title"]; ok {
					_ = json.Unmarshal(v, &payload.Title)
				}
				if v, ok := extra["body"]; ok {
					_ = json.Unmarshal(v, &payload.Body)
				}
				if v, ok := extra["deep_link"]; ok {
					_ = json.Unmarshal(v, &payload.DeepLink)
				}
				if v, ok := extra["group_key"]; ok {
					_ = json.Unmarshal(v, &payload.GroupKey)
				}
			}
			payload.Payload = evt.Payload
		}

		bus.dispatch(ctx, payload)

		if err := repo.MarkProcessed(ctx, evt.ID.String()); err != nil {
			slog.Error("failed to mark event as processed",
				"event_id", evt.ID,
				"error", err,
			)
		}
	}

	return nil
}

// HandlerCount returns the total number of registered handlers (for testing).
func (bus *EventBus) HandlerCount() int {
	bus.mu.RLock()
	defer bus.mu.RUnlock()
	count := 0
	for _, handlers := range bus.handlers {
		count += len(handlers)
	}
	return count
}
