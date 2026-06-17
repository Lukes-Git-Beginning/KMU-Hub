package vertraege

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/event"
)

// EventEmitter is an optional interface for emitting reminder notification events.
// If set, the ReminderWorker will emit events on each delivered reminder.
type EventEmitter interface {
	EmitReminderEvent(ctx context.Context, payload models.EventPayload) error
}

// PGEventEmitter emits notification events via PostgreSQL NOTIFY.
type PGEventEmitter struct {
	pool *pgxpool.Pool
}

// NewPGEventEmitter creates a new PGEventEmitter that emits events via pg_notify.
func NewPGEventEmitter(pool *pgxpool.Pool) *PGEventEmitter {
	return &PGEventEmitter{pool: pool}
}

// EmitReminderEvent implements EventEmitter by sending the payload via pg_notify.
func (e *PGEventEmitter) EmitReminderEvent(ctx context.Context, payload models.EventPayload) error {
	return event.EmitEvent(ctx, e.pool, payload)
}
