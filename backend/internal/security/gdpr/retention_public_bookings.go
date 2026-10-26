package gdpr

// PublicBookingRetentionHandler implements RetentionHandler for resource_type
// "public_bookings" — Lauf 17 decision "Paket Logs/Buchungen" (2026-10-25):
// 24 months after the appointment. The twenty-four months are the
// retention_days an admin sets on the policy row, not a constant here.
//
// Clock is the appointment date (public_bookings.date, a DATE): the table has
// no end column, only the day and the "HH:MM" start slot, and a booking is
// over by the end of that day. A row is due once its date lies before the
// cutoff's calendar day, so the appointment day itself is never cut short.
// Status does not matter: a cancelled booking carries the same customer name,
// email and phone as a completed one.
//
// Delete-only. A booking is the external customer's name, email, phone and
// free-text notes plus a slot; blanking those leaves nothing worth keeping.
// Nothing references public_bookings (calendar_event_id is a plain column,
// no foreign key), so the delete cascades nowhere; the calendar event the
// booking created is governed by the calendar_events policy.

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PublicBookingRetentionHandler is the public_bookings entry in the retention
// registry.
type PublicBookingRetentionHandler struct {
	pool *pgxpool.Pool
}

// NewPublicBookingRetentionHandler wires the handler.
func NewPublicBookingRetentionHandler(pool *pgxpool.Pool) *PublicBookingRetentionHandler {
	return &PublicBookingRetentionHandler{pool: pool}
}

// ResourceType is the value a retention_policies row carries for public bookings.
func (h *PublicBookingRetentionHandler) ResourceType() string { return "public_bookings" }

// Table names the governed table for the admin view.
func (h *PublicBookingRetentionHandler) Table() string { return "public_bookings" }

// DateColumn discloses the clock — see the file header.
func (h *PublicBookingRetentionHandler) DateColumn() string { return "date" }

// SupportsAction reports that public bookings can only be hard-deleted.
func (h *PublicBookingRetentionHandler) SupportsAction(action string) bool {
	return action == models.RetentionActionDelete
}

// Plan lists bookings whose appointment date lies before the cutoff day. Never
// modifies data.
func (h *PublicBookingRetentionHandler) Plan(ctx context.Context, tenantID uuid.UUID, cutoff time.Time, _ string) (*RetentionPlan, error) {
	rows, err := h.pool.Query(ctx,
		`SELECT id FROM public_bookings
		  WHERE tenant_id = $1 AND date < $2::date
		  ORDER BY date ASC`,
		tenantID, cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("public booking retention: list candidates: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, fmt.Errorf("public booking retention: scan candidate: %w", scanErr)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("public booking retention: iterate candidates: %w", err)
	}
	return &RetentionPlan{Due: ids}, nil
}

// Apply deletes the given bookings. Idempotent: a second run over the same ids
// finds nothing left.
func (h *PublicBookingRetentionHandler) Apply(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, action, _ string) (int, error) {
	if action != models.RetentionActionDelete {
		return 0, fmt.Errorf("public booking retention: unsupported action %q", action)
	}
	tag, err := h.pool.Exec(ctx,
		`DELETE FROM public_bookings WHERE tenant_id = $1 AND id = ANY($2)`,
		tenantID, ids,
	)
	if err != nil {
		return 0, fmt.Errorf("public booking retention: delete: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

var _ RetentionHandler = (*PublicBookingRetentionHandler)(nil)
