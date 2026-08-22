package gdpr

// CalendarEventRetentionHandler is the seventh handler on the retention
// engine from A10 (retention.go). Fund aus
// scan-personal-data-tables-without-retention-mapping (Iteration 43): the
// calendar module (calendar_events, with event_attendees/event_exceptions/
// event_reminders cascading off it) collects free-text personal data
// (title, description, location) with no retention mapping, even though A9
// (dsar_search.go, calendarEventsModule) already surfaces the same data as
// personal in DSAR export.
//
// A plain "start_time/end_time older than cutoff" filter would be wrong for
// a recurring event: rrule + recurrence_end describe a whole series, and the
// stored start_time/end_time are only its FIRST occurrence. A daily standup
// created two years ago with no end date is still happening today — deleting
// it because its first occurrence is old would remove an active series.
// recurrence_end is therefore the clock for a recurring event (NULL means
// "still recurring, next occurrence unknown, never due"); end_time is the
// clock for a one-off event.
//
// event_attendees, event_exceptions and event_reminders all cascade off
// calendar_events(id) ON DELETE CASCADE (migration 000033), so a hard delete
// takes all three with it for free. Anonymize does not get that for free: it
// clears calendar_events' own free-text fields, then does the same for any
// event_exceptions override that carries its own title/description/location
// — NULL there means "use the parent event's value", so resetting an
// override back to NULL keeps that inheritance instead of leaving a
// still-personal exception ("Verschoben wegen Arzttermin Frau Muster")
// standing next to its now-anonymized parent.

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// CalendarEventRetentionHandler is the calendar_events entry in the
// retention registry.
type CalendarEventRetentionHandler struct {
	pool *pgxpool.Pool
}

// NewCalendarEventRetentionHandler wires the handler.
func NewCalendarEventRetentionHandler(pool *pgxpool.Pool) *CalendarEventRetentionHandler {
	return &CalendarEventRetentionHandler{pool: pool}
}

// ResourceType is the value a retention_policies row carries for calendar
// events.
func (h *CalendarEventRetentionHandler) ResourceType() string { return "calendar_events" }

// Table names the governed table for the admin view.
func (h *CalendarEventRetentionHandler) Table() string { return "calendar_events" }

// DateColumn discloses the retention clock, which differs by event shape —
// see the file header for why a recurring series cannot use
// start_time/end_time the way a one-off event does.
func (h *CalendarEventRetentionHandler) DateColumn() string {
	return "end_time fuer Einzeltermine, recurrence_end fuer Serien (rrule ohne Enddatum ist nie faellig); bei anonymize zusaetzlich updated_at"
}

// SupportsAction reports that calendar events can be both hard-deleted and
// anonymized.
func (h *CalendarEventRetentionHandler) SupportsAction(action string) bool {
	return action == models.RetentionActionDelete || action == models.RetentionActionAnonymize
}

// Plan lists events past their retention cutoff. Never modifies data.
//
// A one-off event (rrule NULL) is due once its own end_time is past cutoff.
// A recurring event (rrule set) is due only if recurrence_end is set AND
// past cutoff — an open-ended series (recurrence_end NULL) never appears
// here, at any cutoff, because its next occurrence is unknown and may be in
// the future.
func (h *CalendarEventRetentionHandler) Plan(ctx context.Context, tenantID uuid.UUID, cutoff time.Time, action string) (*RetentionPlan, error) {
	query := `
		SELECT id FROM calendar_events
		 WHERE tenant_id = $1
		   AND (
		     (rrule IS NULL AND end_time < $2)
		     OR (rrule IS NOT NULL AND recurrence_end IS NOT NULL AND recurrence_end < $2)
		   )`
	if action == models.RetentionActionAnonymize {
		query += ` AND updated_at < $2`
	}
	query += ` ORDER BY end_time ASC`

	rows, err := h.pool.Query(ctx, query, tenantID, cutoff)
	if err != nil {
		return nil, fmt.Errorf("calendar event retention: list candidates: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, fmt.Errorf("calendar event retention: scan candidate: %w", scanErr)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("calendar event retention: iterate candidates: %w", err)
	}
	return &RetentionPlan{Due: ids}, nil
}

// Apply deletes or anonymizes the given events.
//
// Delete relies on the ON DELETE CASCADE from event_attendees,
// event_exceptions and event_reminders to calendar_events (migration 000033)
// to take all three with it in the same statement.
//
// Anonymize clears the event's three free-text fields (title, description,
// location) and stamps updated_at for idempotency, then resets any
// event_exceptions override for those same fields back to NULL — see the
// file header for why NULL, not a bracketed label, is correct there.
func (h *CalendarEventRetentionHandler) Apply(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, action, anonymizedLabel string) (int, error) {
	switch action {
	case models.RetentionActionDelete:
		tag, err := h.pool.Exec(ctx,
			`DELETE FROM calendar_events WHERE tenant_id = $1 AND id = ANY($2)`,
			tenantID, ids,
		)
		if err != nil {
			return 0, fmt.Errorf("calendar event retention: delete: %w", err)
		}
		return int(tag.RowsAffected()), nil
	case models.RetentionActionAnonymize:
		tag, err := h.pool.Exec(ctx,
			`UPDATE calendar_events
			    SET title = '[' || $3 || ']',
			        description = '[' || $3 || ']',
			        location = '[' || $3 || ']',
			        updated_at = NOW()
			  WHERE tenant_id = $1 AND id = ANY($2)`,
			tenantID, ids, anonymizedLabel,
		)
		if err != nil {
			return 0, fmt.Errorf("calendar event retention: anonymize: %w", err)
		}
		affected := int(tag.RowsAffected())

		if _, err := h.pool.Exec(ctx,
			`UPDATE event_exceptions
			    SET title = NULL, description = NULL, location = NULL
			  WHERE tenant_id = $1 AND event_id = ANY($2)
			    AND (title IS NOT NULL OR description IS NOT NULL OR location IS NOT NULL)`,
			tenantID, ids,
		); err != nil {
			return affected, fmt.Errorf("calendar event retention: clear exception overrides: %w", err)
		}
		return affected, nil
	default:
		return 0, fmt.Errorf("calendar event retention: unsupported action %q", action)
	}
}
