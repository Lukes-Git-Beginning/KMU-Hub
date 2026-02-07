package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PGNotifyDeliveryChannel is the PostgreSQL LISTEN channel used by the gateway
// to receive notification delivery signals for WebSocket push.
const PGNotifyDeliveryChannel = "notification_delivery"

// EmitEvent sends an event payload via PostgreSQL NOTIFY on the events channel.
// This should be called after the triggering operation succeeds.
// The payload must be under 8000 bytes (PostgreSQL NOTIFY limit).
func EmitEvent(ctx context.Context, pool *pgxpool.Pool, payload models.EventPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	if len(data) > 7500 {
		slog.Warn("event payload approaching NOTIFY limit",
			"size", len(data),
			"type", payload.Type,
		)
	}

	_, err = pool.Exec(ctx, "SELECT pg_notify('events', $1)", string(data))
	if err != nil {
		slog.Error("failed to emit event via pg_notify",
			"type", payload.Type,
			"error", err,
		)
		return fmt.Errorf("pg_notify: %w", err)
	}

	slog.Debug("event emitted",
		"type", payload.Type,
		"module", payload.ModuleID,
		"targets", len(payload.TargetUserIDs),
	)

	return nil
}
