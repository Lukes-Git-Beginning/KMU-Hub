package fuhrpark

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/event"
)

// EventEmitter is an optional interface for emitting TUEV reminder notification events.
// If set, the TuevWorker will emit events on each triggered reminder.
// Keeping it optional (nil-safe) means tests can omit the dependency.
type EventEmitter interface {
	EmitTuevEvent(ctx context.Context, payload models.EventPayload) error
}

// PGEventEmitter emits notification events via PostgreSQL NOTIFY.
type PGEventEmitter struct {
	pool *pgxpool.Pool
}

// NewPGEventEmitter creates a PGEventEmitter backed by the given pool.
func NewPGEventEmitter(pool *pgxpool.Pool) *PGEventEmitter {
	return &PGEventEmitter{pool: pool}
}

// EmitTuevEvent implements EventEmitter by forwarding the payload to pg_notify.
func (e *PGEventEmitter) EmitTuevEvent(ctx context.Context, payload models.EventPayload) error {
	return event.EmitEvent(ctx, e.pool, payload)
}
