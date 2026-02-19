package message

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/event"
)

// EventEmitter is an optional interface for emitting notification events
// when email messages are received, sent, or deleted.
type EventEmitter interface {
	EmitEmailEvent(ctx context.Context, payload models.EventPayload) error
}

// PGEventEmitter emits email events via PostgreSQL NOTIFY.
type PGEventEmitter struct {
	pool *pgxpool.Pool
}

// NewPGEventEmitter creates a new PGEventEmitter that emits events via pg_notify.
func NewPGEventEmitter(pool *pgxpool.Pool) *PGEventEmitter {
	return &PGEventEmitter{pool: pool}
}

// EmitEmailEvent implements EventEmitter by sending the payload via pg_notify.
func (e *PGEventEmitter) EmitEmailEvent(ctx context.Context, payload models.EventPayload) error {
	return event.EmitEvent(ctx, e.pool, payload)
}
