package trigger

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// maxDueEntitiesPerAutomation caps how many entities one automation may fire
// for in a single tick. The poller starts a goroutine per fired entity, so an
// uncapped resolver on a tenant with a five-figure overdue backlog would spawn
// that many concurrent executions on the first tick after the feature goes
// live.
//
// lean: a flat cap that drops the tail until the next tick rather than a work
// queue. Raise it, or introduce a queue, when a tenant legitimately has more
// than 200 entities due for one automation within one interval.
const maxDueEntitiesPerAutomation = 200

// triggerCalendarEventUpcoming is the one time-based trigger type without a
// constant in internal/notification/event -- it is synthetic (nothing outside
// this package ever publishes it) and so has no place in the shared event
// vocabulary. Named here so the registry entry and the resolver's switch
// cannot drift apart.
const triggerCalendarEventUpcoming = "calendar.event.upcoming"

// ErrUnknownTimeTrigger is returned by a DueResolver for a trigger type it has
// no due-query for. It is deliberately an error rather than an empty result:
// a TriggerDefinition newly flagged TimeBased=true but never taught to a
// resolver would otherwise sit in the poller silently doing nothing.
var ErrUnknownTimeTrigger = errors.New("no due resolver for trigger type")

// DueEntity is one concrete thing that makes a time-based trigger due right
// now -- an invoice that went overdue, a calendar event about to start.
type DueEntity struct {
	// Key deduplicates fires via automation_time_trigger_fires. Its
	// granularity is the resolver's decision (once per entity, or once per
	// entity per day), not the caller's -- see migration 000303.
	Key string

	// ResourceID is the entity's own ID, carried onto EventPayload.ResourceID.
	ResourceID string

	// Payload is NESTED, e.g. {"invoice": {"days_overdue": 12}}, because
	// condition.getFieldValue walks a dotted field like "invoice.days_overdue"
	// as nested maps. A flat {"invoice.days_overdue": 12} would never match.
	Payload map[string]any
}

// DueResolver answers, for one time-based trigger type and one tenant, which
// entities are due at now. Implementations MUST scope every query by tenantID
// explicitly: the poller runs under system context, so RLS will not do it.
type DueResolver interface {
	Resolve(ctx context.Context, triggerType string, tenantID uuid.UUID, now time.Time) ([]DueEntity, error)
}
