package gdpr

// NotificationRetentionHandler is the eighth handler on the retention engine
// from A10 (retention.go). Fund aus scan-personal-data-tables-without-
// retention-mapping (Iteration 43): the `notifications` table (title,
// free-text body, actor_id) has no entry in the retention registry and is
// touched by none of the other seven handlers. Unlike contacts or tickets,
// there is no business reason to keep an old, already-read notification
// around — this is the clearest of the three findings from that scan.
//
// Delete only, never anonymize: an anonymized notification ("[geloescht] hat
// ... erwaehnt") is worthless to its recipient — the notification's only
// purpose is to be read, and a blanked-out one serves neither the recipient
// nor a retention audit. SupportsAction says so explicitly instead of
// quietly accepting both actions.
//
// An UNREAD notification is excluded from Due regardless of age: deleting a
// notification nobody has seen yet is a feature loss (a message the
// recipient never got to see silently disappears), not clean-up. That
// exclusion lives directly in the Plan query, not as a post-hoc filter, the
// same way CalendarEventRetentionHandler excludes an open-ended recurring
// series at the query level rather than in Skipped.
import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// NotificationRetentionHandler is the notifications entry in the retention
// registry.
type NotificationRetentionHandler struct {
	pool *pgxpool.Pool
}

// NewNotificationRetentionHandler wires the handler.
func NewNotificationRetentionHandler(pool *pgxpool.Pool) *NotificationRetentionHandler {
	return &NotificationRetentionHandler{pool: pool}
}

// ResourceType is the value a retention_policies row carries for
// notifications.
func (h *NotificationRetentionHandler) ResourceType() string { return "notifications" }

// Table names the governed table for the admin view.
func (h *NotificationRetentionHandler) Table() string { return "notifications" }

// DateColumn discloses the retention clock and the unread exclusion.
func (h *NotificationRetentionHandler) DateColumn() string {
	return "created_at (nur gelesene Benachrichtigungen; ungelesene sind nie faellig)"
}

// SupportsAction reports that notifications can only be hard-deleted — see
// the file header for why anonymizing one is pointless.
func (h *NotificationRetentionHandler) SupportsAction(action string) bool {
	return action == models.RetentionActionDelete
}

// Plan lists read notifications past their retention cutoff. Never modifies
// data. An unread notification never appears here, no matter how old.
func (h *NotificationRetentionHandler) Plan(ctx context.Context, tenantID uuid.UUID, cutoff time.Time, _ string) (*RetentionPlan, error) {
	rows, err := h.pool.Query(ctx,
		`SELECT id FROM notifications
		  WHERE tenant_id = $1 AND is_read = true AND created_at < $2
		  ORDER BY created_at ASC`,
		tenantID, cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("notification retention: list candidates: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, fmt.Errorf("notification retention: scan candidate: %w", scanErr)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("notification retention: iterate candidates: %w", err)
	}
	return &RetentionPlan{Due: ids}, nil
}

// Apply deletes the given notifications. Idempotent by construction — a
// second run finds nothing left to delete.
func (h *NotificationRetentionHandler) Apply(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, action, _ string) (int, error) {
	if action != models.RetentionActionDelete {
		return 0, fmt.Errorf("notification retention: unsupported action %q", action)
	}
	tag, err := h.pool.Exec(ctx,
		`DELETE FROM notifications WHERE tenant_id = $1 AND id = ANY($2)`,
		tenantID, ids,
	)
	if err != nil {
		return 0, fmt.Errorf("notification retention: delete: %w", err)
	}
	return int(tag.RowsAffected()), nil
}
