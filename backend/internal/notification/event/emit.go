package event

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
)

// ErrMissingTenant is returned when an event carries no tenant and none can be
// taken from the context. The emit is skipped rather than sent: EventBus.dispatch
// only stamps app.tenant_id when the payload carries one, so a tenant-less event
// reaches every handler unscoped and its RLS-guarded inserts (notifications,
// events, inbox_messages) are rejected one layer down, where the failure is only
// logged and the event is lost.
var ErrMissingTenant = errors.New("event payload has no tenant and none in context")

// PGNotifyDeliveryChannel is the PostgreSQL LISTEN channel used by the gateway
// to receive notification delivery signals for WebSocket push.
const PGNotifyDeliveryChannel = "notification_delivery"

// EmitEvent sends an event payload via PostgreSQL NOTIFY on the events channel.
// This should be called after the triggering operation succeeds.
// The payload must be under 8000 bytes (PostgreSQL NOTIFY limit).
func EmitEvent(ctx context.Context, pool *pgxpool.Pool, payload models.EventPayload) error {
	// Request-driven emitters build the payload from their own arguments and
	// leave TenantID zero; scheduled ones (workers, pollers, backlog replay)
	// already carry it from the row they processed. Filling it here rather than
	// in each of the callers keeps every future emitter tenant-scoped by
	// construction.
	if payload.TenantID == uuid.Nil {
		tenantID, err := middleware.GetTenantID(ctx)
		if err != nil {
			slog.WarnContext(ctx, "event emit skipped: no tenant",
				"type", payload.Type,
				"module", payload.ModuleID,
				"resource_id", payload.ResourceID,
				"error", err,
			)
			return ErrMissingTenant
		}
		payload.TenantID = tenantID
	}

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
