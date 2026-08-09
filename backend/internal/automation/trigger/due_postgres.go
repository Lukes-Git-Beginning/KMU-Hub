package trigger

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/notification/event"
)

// upcomingEventWindow is how far ahead calendar.event.upcoming looks. It
// matches the window the trigger registry advertises to the user ("fires 15
// minutes before a calendar event") and must stay >= the poller interval, or
// events slip between two ticks unnoticed.
const upcomingEventWindow = 15 * time.Minute

// PostgresDueResolver resolves due entities straight from the shared database.
//
// It reads tables owned by other modules (finance_invoices, calendar_events)
// rather than calling their services: the poller has no request context, no
// user, and no gRPC identity to call with, and both queries are read-only
// projections of two columns. The tenant filter is therefore written out
// explicitly in every query -- the poller runs under system context, so RLS is
// open and the WHERE clause is the only thing keeping tenant A's automation
// off tenant B's invoices.
type PostgresDueResolver struct {
	pool *pgxpool.Pool
}

// NewPostgresDueResolver creates a DueResolver backed by pool.
func NewPostgresDueResolver(pool *pgxpool.Pool) *PostgresDueResolver {
	return &PostgresDueResolver{pool: pool}
}

// Resolve dispatches to the query for triggerType.
func (r *PostgresDueResolver) Resolve(ctx context.Context, triggerType string, tenantID uuid.UUID, now time.Time) ([]DueEntity, error) {
	switch triggerType {
	case event.EventInvoiceOverdue:
		return r.overdueInvoices(ctx, tenantID, now)
	case triggerCalendarEventUpcoming:
		return r.upcomingCalendarEvents(ctx, tenantID, now)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownTimeTrigger, triggerType)
	}
}

// overdueInvoices returns one entity per invoice whose due date has passed and
// which is neither paid nor cancelled nor still a draft.
//
// The key carries the date, so an invoice yields one fire per day for as long
// as it stays overdue. That is what the shipped "invoice-overdue-dunning"
// template needs: it filters invoice.days_overdue >= 14, so a single
// once-per-invoice fire on day one would evaluate false and burn the only
// chance that invoice ever had.
func (r *PostgresDueResolver) overdueInvoices(ctx context.Context, tenantID uuid.UUID, now time.Time) ([]DueEntity, error) {
	const query = `
		SELECT id,
		       invoice_number,
		       gross_total::float8,
		       status,
		       due_date,
		       ($2::date - due_date) AS days_overdue
		FROM finance_invoices
		WHERE tenant_id = $1
		  AND status IN ('sent', 'overdue')
		  AND due_date < $2::date
		ORDER BY due_date ASC
		LIMIT $3`

	day := now.Format("2006-01-02")

	rows, err := r.pool.Query(ctx, query, tenantID, day, maxDueEntitiesPerAutomation)
	if err != nil {
		return nil, fmt.Errorf("query overdue invoices: %w", err)
	}
	defer rows.Close()

	var entities []DueEntity
	for rows.Next() {
		var (
			id          uuid.UUID
			number      string
			grossTotal  float64
			status      string
			dueDate     time.Time
			daysOverdue int
		)
		if scanErr := rows.Scan(&id, &number, &grossTotal, &status, &dueDate, &daysOverdue); scanErr != nil {
			return nil, fmt.Errorf("scan overdue invoice: %w", scanErr)
		}
		entities = append(entities, DueEntity{
			Key:        fmt.Sprintf("invoice:%s:%s", id, day),
			ResourceID: id.String(),
			Payload: map[string]any{
				"invoice": map[string]any{
					"id":           id.String(),
					"number":       number,
					"total":        grossTotal,
					"status":       status,
					"due_date":     dueDate.Format("2006-01-02"),
					"days_overdue": daysOverdue,
				},
			},
		})
	}
	return entities, rows.Err()
}

// upcomingCalendarEvents returns one entity per event starting inside the next
// upcomingEventWindow.
//
// lean: recurring events are matched on their stored start_time only. rrule
// instances are not materialised anywhere in this repo, so a series fires for
// its first occurrence and no other. Expand here once rrule expansion exists
// server-side (today it lives in the desktop client).
func (r *PostgresDueResolver) upcomingCalendarEvents(ctx context.Context, tenantID uuid.UUID, now time.Time) ([]DueEntity, error) {
	const query = `
		SELECT id,
		       title,
		       start_time,
		       FLOOR(EXTRACT(EPOCH FROM (start_time - $2::timestamptz)) / 60)::int AS minutes_until
		FROM calendar_events
		WHERE tenant_id = $1
		  AND start_time >= $2::timestamptz
		  AND start_time < $3::timestamptz
		ORDER BY start_time ASC
		LIMIT $4`

	rows, err := r.pool.Query(ctx, query, tenantID, now, now.Add(upcomingEventWindow), maxDueEntitiesPerAutomation)
	if err != nil {
		return nil, fmt.Errorf("query upcoming calendar events: %w", err)
	}
	defer rows.Close()

	var entities []DueEntity
	for rows.Next() {
		var (
			id           uuid.UUID
			title        string
			startTime    time.Time
			minutesUntil int
		)
		if scanErr := rows.Scan(&id, &title, &startTime, &minutesUntil); scanErr != nil {
			return nil, fmt.Errorf("scan upcoming calendar event: %w", scanErr)
		}
		entities = append(entities, DueEntity{
			// No date in the key: an event starts once, so a reminder that
			// fires on two consecutive days is a bug, not a feature.
			Key:        fmt.Sprintf("calendar_event:%s", id),
			ResourceID: id.String(),
			Payload: map[string]any{
				"event": map[string]any{
					"id":            id.String(),
					"title":         title,
					"start_time":    startTime.Format(time.RFC3339),
					"minutes_until": minutesUntil,
				},
			},
		})
	}
	return entities, rows.Err()
}
