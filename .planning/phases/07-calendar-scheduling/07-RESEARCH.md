# Phase 7: Calendar & Scheduling - Research

**Researched:** 2026-02-08
**Domain:** Calendar/scheduling backend (Go gRPC) + frontend (React calendar views, event management, resource booking)
**Confidence:** HIGH (codebase patterns verified, external libs confirmed via docs/web research)

## Summary

Phase 7 adds a Calendar module to the Work gRPC microservice (not a new service) following the established backend architecture patterns. The backend handles calendars, events, recurring rules (RRULE/RFC 5545), RSVP invitations, resource booking, holiday data, and LiveKit meeting link generation. The frontend provides day/week/month calendar views, event creation/editing, shared calendar overlays, and a room booking timeline.

The most architecturally complex aspects are: (1) recurring event storage and expansion with RRULE, including three-way edit scope ("this event" / "this and future" / "all events"), (2) the calendar UI grid with click-to-create, drag-to-resize, and overlapping event layout, and (3) double-booking prevention for resources using PostgreSQL exclusion constraints. Phase 14 (CalDAV) will build on this data model, so RFC 5545 compliance in the schema is critical for forward compatibility.

The calendar backend extends the existing Work service rather than creating a new microservice. This is the established pattern from the service consolidation decision -- Work handles projects, tasks, AND now calendars/events. The frontend uses a custom-built calendar grid (not react-big-calendar) to achieve the exact Google Calendar interaction patterns specified in the context decisions, while leveraging existing infrastructure (Radix UI, Tailwind, TanStack Query, date-fns).

**Primary recommendation:** Extend the Work service with calendar/event/resource domain packages. Use `teambition/rrule-go` for server-side RRULE expansion, `rrule` (npm) for client-side preview. Build a custom calendar grid with CSS Grid for the frontend rather than wrapping react-big-calendar, which has SCSS/Tailwind conflicts and limited control over Google Calendar-style interactions. Use Nager.Date API to seed DACH holiday data, cache it locally in PostgreSQL. Use PostgreSQL exclusion constraints (`btree_gist`) to enforce room double-booking prevention at the database level.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Calendar views & navigation:**
- Default view is week view when opening the Calendar module
- Week view shows 5 days (Mon-Fri) by default, user-configurable to 5 or 7 days in settings
- Day agenda sidebar instead of mini month calendar -- sidebar shows a chronological event list for the selected day, updates when clicking a date in the main view
- Full-screen day view uses an hourly timeline (vertical time grid with events placed at their time position)
- Month view shows 2-3 events per cell, then "+N more" link that opens a popover/expands
- Prominent "Heute" button in toolbar for one-click return to current date
- Task deadlines from PM module appear as a separate toggleable layer (distinct color/style) that can be switched on/off

**Event creation & editing:**
- Both creation methods: click/drag on time slot in calendar AND a "+ New Event" button
- Click on time slot opens a compact popover (title + time + optional details) with a "Mehr Optionen" link to the full form
- User-defined color categories for events (e.g., "Meeting", "Focus time", "Travel") -- users create their own categories with colors
- Location field: free-text for external locations plus a separate "Room" dropdown that links to resource calendars
- Reminders: per-event override with global default -- global default (e.g., 15 min before), overridable per individual event
- Event description is plain text (no rich text editor)
- Recurring events: presets + custom -- quick presets (daily, weekly, monthly, yearly) plus a "Custom..." option for full RRULE control
- Editing recurring events: three-way scope -- "This event" / "This and future events" / "All events" (Google Calendar pattern)

**Shared calendars & overlay:**
- Role-based creation: managers+ can create shared calendars, members can only view/join ones they have access to
- Overlapping events from different calendars displayed as side-by-side columns in the same time slot
- Colleague availability shown as simple busy/available indicator per time slot when scheduling
- Three-level permissions: View / Edit / Admin per shared calendar
- Calendar colors: default color per calendar + user override
- Calendar list in sidebar, grouped by type: "My calendars", "Shared calendars", "Other" with toggle checkboxes
- Calendar discovery: browse + subscribe -- a "Browse calendars" page showing all shared calendars the user has access to
- Full notification integration: event invitations create notifications via Phase 4 system, RSVP responses update event and notify organizer

**Room booking & resources:**
- Rooms + equipment as bookable resources (meeting rooms, beamers, company cars, phone booths, etc.)
- Room availability shown as horizontal timelines per room for the selected day -- click empty slot to book
- Room attributes: capacity, equipment tags (beamer, whiteboard), floor/location -- users can filter rooms by these
- Double-booking: blocked with alternative suggestions (show "Room occupied" with list of available alternative rooms)
- Resource management: admin + managers can create, edit, and delete resources
- Booking flow: instant booking -- first come, first served, no approval required
- Booking a resource automatically creates a calendar event on the booker's personal calendar
- Dual entry points: dedicated "Raeume" page with timeline view AND room selector within event creation form

### Claude's Discretion
- Exact calendar grid component library / implementation approach
- Sidebar layout proportions and responsive behavior
- RRULE storage format details (RFC 5545 compliance is required, implementation details flexible)
- Holiday data source and storage approach
- Notification timing defaults
- Event drag-to-resize behavior in week/day views
- Keyboard shortcuts for calendar navigation

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

## Discretion Recommendations

### Calendar Grid Implementation Approach

**Recommendation: Custom-built calendar grid with CSS Grid, NOT react-big-calendar**

react-big-calendar (707k weekly npm downloads, 8.6k GitHub stars) is the most popular option, but has significant drawbacks for this project:

1. **Tailwind CSS incompatibility**: react-big-calendar uses SCSS for styling. Overriding its styles with Tailwind requires complex CSS overrides that can break rendering. The project uses Tailwind CSS 4 exclusively.
2. **Limited click-to-create interaction**: The Google Calendar popover-on-click pattern requires deep customization of react-big-calendar's event creation flow.
3. **Side-by-side column layout**: Overlapping events from different calendars as side-by-side columns is not a built-in react-big-calendar feature.
4. **CalDAV future compatibility**: Phase 14 adds CalDAV -- a custom grid gives full control over RFC 5545 rendering without library abstractions.

A custom calendar grid built with CSS Grid provides:
- Full control over Google Calendar interaction patterns (click-to-create popover, drag-to-resize, "+N more")
- Native Tailwind CSS styling without override conflicts
- Exact layout control for side-by-side overlapping events
- date-fns (already in project) for all date calculations
- Clean integration with existing Radix UI components (Popover, Dialog)

The grid structure:
- **Week view**: CSS Grid with `grid-template-columns: auto repeat(N, 1fr)` where N is 5 or 7 days. Rows are 15-minute time slots. Events positioned absolutely within day columns.
- **Day view**: Single-column hourly timeline. Same time grid as week view but full width.
- **Month view**: CSS Grid with `grid-template-columns: repeat(7, 1fr)`. Fixed-height cells with event bars and "+N more" overflow.

**Confidence:** HIGH -- multiple open-source Google Calendar clones (github.com/vladyslav-soltanovskyi/react-calendar, github.com/lramos33/big-calendar) demonstrate this is achievable with CSS Grid + React, and the project already has the primitives (date-fns, Radix UI, Tailwind).

### RRULE Storage Format

**Recommendation: Store RRULE as RFC 5545 string + server-side expansion**

Store the RRULE string directly in the database (e.g., `FREQ=WEEKLY;BYDAY=MO,WE,FR;UNTIL=20260630T235959Z`). The server expands occurrences on-the-fly when querying a date range. No pre-computed `event_instances` table -- expansion is done in the service layer using `teambition/rrule-go`.

Rationale:
- Simpler schema (no event_instances sync job)
- RFC 5545 string is directly usable for CalDAV export in Phase 14
- RRULE expansion for a 31-day window (month view) is fast (microseconds for typical rules)
- Exceptions (modified/cancelled instances) stored in a separate `event_exceptions` table with `original_date` as the key

For "This and future events" modifications:
- The original event's RRULE gets an `UNTIL` date set to the split point
- A new event is created with the modified RRULE starting from the split date
- This follows the RFC 5545 THISANDFUTURE pattern and is how Google Calendar implements it

**Confidence:** HIGH -- verified via RFC 5545 specification. The THISANDFUTURE split pattern is the industry standard approach.

### Holiday Data Source and Storage

**Recommendation: Nager.Date API for initial seeding + local PostgreSQL cache**

Approach:
1. **Seed on deployment/first-run**: Call Nager.Date API (`https://date.nager.at/api/v3/PublicHolidays/{year}/{country}`) for DE, AT, CH to get holiday data for the current and next year
2. **Store in PostgreSQL**: `public_holidays` table with `date`, `name`, `local_name`, `country_code`, `subdivision_codes` (array of ISO-3166-2 codes like `DE-BY`, `AT-9`, `CH-ZH`)
3. **Annual refresh**: Background job or admin trigger to fetch next year's holidays
4. **User configuration**: Each user selects their Bundesland/Kanton in settings; calendar filters holidays by their subdivision

Why Nager.Date over Go libraries:
- `wlbr/feiertage` (Go library) covers only Germany and Austria, NOT Switzerland. Last release was May 2020.
- Nager.Date covers all three DACH countries with subdivision-level data via REST API
- The API is free, no rate limits, returns subdivision codes in ISO-3166-2 format
- Self-hosted option: Nager.Date is open-source .NET, can be deployed as Docker container if API dependency is a concern

**Confidence:** MEDIUM -- Nager.Date API is well-documented but is an external dependency. Verified it returns `counties` field with ISO-3166-2 subdivision codes. Fallback: hardcode DACH holiday data (it rarely changes year to year, most holidays are fixed or Easter-relative).

### Notification Timing Defaults

**Recommendation: 15 minutes before for timed events, 18:00 day before for all-day events**

- **Timed events default**: 15 minutes before start time (most common in Google Calendar, Outlook)
- **All-day events default**: 18:00 the evening before (gives users time to prepare for next-day events)
- **Available preset options**: 0 min, 5 min, 10 min, 15 min, 30 min, 1 hour, 2 hours, 1 day, 2 days
- **Custom**: Users can set any number of minutes/hours/days before
- **Multiple reminders**: Support up to 3 reminders per event (e.g., 1 day + 15 min before)

**Confidence:** HIGH -- these are Google Calendar defaults and widely expected by users.

### Event Drag-to-Resize Behavior

**Recommendation: Vertical drag on bottom edge to extend/shorten duration, snap to 15-minute intervals**

- **Resize handle**: Bottom edge of event block in week/day view shows a thin drag handle on hover
- **Snap grid**: Resize snaps to 15-minute intervals (matching the time grid granularity)
- **Visual feedback**: During drag, show the new end time as a tooltip/badge near the cursor
- **Optimistic update**: Apply resize immediately, sync with server, snap back on error
- **Constraints**: Minimum event duration of 15 minutes, maximum of 24 hours
- **Drag-to-move**: Click and drag the event body (not the resize handle) to move to a different time slot or day
- **Cross-day drag**: In week view, dragging an event to a different day column moves it to that day

**Confidence:** HIGH -- standard Google Calendar behavior.

### Keyboard Shortcuts

**Recommendation: Google Calendar-inspired shortcuts**

| Shortcut | Action |
|----------|--------|
| `T` | Go to today ("Heute") |
| `D` | Switch to day view |
| `W` | Switch to week view |
| `M` | Switch to month view |
| `J` or `Right` | Next period (next day/week/month) |
| `K` or `Left` | Previous period |
| `C` | Create new event |
| `Escape` | Close popover/dialog |
| `Delete` | Delete selected event |

Shortcuts only active when no input field is focused.

**Confidence:** HIGH -- follows Google Calendar conventions that DACH users likely know.

### Sidebar Layout Proportions

**Recommendation: Fixed-width sidebar (280px), collapsible**

- **Width**: 280px fixed (not responsive width, but can be collapsed)
- **Collapse**: Toggle button to hide sidebar entirely (full-width calendar view)
- **Sections from top to bottom**:
  1. Date navigation (prev/next buttons, month/year display, "Heute" button) -- 60px
  2. Day agenda (scrollable list of today's events, updates on date click) -- flexible height
  3. Calendar list (My Calendars, Shared Calendars, Other) with toggle checkboxes -- flexible height, collapsible groups
- **Responsive**: On smaller screens (< 1200px), sidebar auto-collapses to icon-only or hides behind a hamburger toggle

**Confidence:** MEDIUM -- layout proportions are subjective. 280px matches Google Calendar sidebar width.

## Standard Stack

### Core (Backend - Go, extends Work service)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go standard library + slog | 1.25.6 | Service implementation, structured logging | Already in use |
| google.golang.org/grpc | 1.78.0 | Inter-service communication (Work service) | Already in use |
| github.com/jackc/pgx/v5 | 5.8.0 | PostgreSQL driver + connection pool | Already in use |
| github.com/google/uuid | 1.6.0 | UUID generation for all entities | Already in use |
| google.golang.org/protobuf | 1.36.11 | Proto serialization | Already in use |
| **github.com/teambition/rrule-go** | latest | RFC 5545 RRULE parsing and expansion | Most complete Go RRULE library, partial port of python-dateutil |
| **github.com/livekit/server-sdk-go/v2** | latest | LiveKit room creation + token generation | Official LiveKit Go SDK for meeting link generation |
| golang-migrate | (CLI) | Database migrations | Already in use |

### Core (Frontend - React/TypeScript)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| react | ^19.0.0 | UI framework | Already in use |
| date-fns | ^4.0.0 | Date calculations (startOfWeek, eachDayOfInterval, format, etc.) | Already in use |
| **rrule** | ^2.8.1 | Client-side RRULE preview in recurrence editor | Standard RFC 5545 implementation for JS, used by FullCalendar |
| @tanstack/react-query | ^5.x | Server state management for events | Already in use |
| zustand | ^5.x | Client state (selected calendars, view preferences, sidebar state) | Already in use |
| @radix-ui/react-popover | installed | Event quick-create popover | Already in use |
| @radix-ui/react-dialog | installed | Full event form modal, room booking dialog | Already in use |
| @radix-ui/react-select | installed | Calendar selector, room selector, recurrence presets | Already in use |
| lucide-react | ^0.470 | Calendar icons (CalendarDays, Clock, MapPin, Users, Video) | Already in use |

### New Dependencies to Install

```bash
# Backend (new Go dependencies)
cd backend
go get github.com/teambition/rrule-go
go get github.com/livekit/server-sdk-go/v2
go get github.com/livekit/protocol

# Frontend (new JS dependency)
cd desktop
npm install rrule
```

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Custom calendar grid | react-big-calendar | SCSS conflicts with Tailwind 4, limited control over Google Calendar interactions; custom gives full design control |
| Custom calendar grid | @schedule-x/react | Newer alternative, but less battle-tested; ~5k GitHub stars vs custom with full control |
| teambition/rrule-go | stephens2424/rrule | Fewer features, less actively maintained; teambition is a port of python-dateutil |
| Nager.Date API | wlbr/feiertage (Go) | Go library but no Switzerland support, last release 2020; Nager covers all DACH |
| Nager.Date API | OpenHolidays API | Similar coverage but less documentation; Nager is more widely used |
| RRULE string storage | Pre-computed instances table | Simpler queries but complex sync, storage overhead; string storage is CalDAV-compatible |

## Architecture Patterns

### Recommended Backend Structure (extends Work service)

```
backend/
  internal/
    work/
      calendar/
        errors.go
        repository.go            # Calendar CRUD
        postgres_repository.go
        service.go               # Calendar management, permission checks
        service_test.go
      event/
        errors.go
        repository.go            # Event CRUD, recurrence expansion, RSVP
        postgres_repository.go
        rrule.go                 # RRULE parsing/expansion wrapper
        service.go               # Event creation, recurrence, invitations
        service_test.go
        event_emitter.go         # pg_notify for calendar notifications
      resource/
        errors.go
        repository.go            # Resource CRUD, availability queries
        postgres_repository.go
        service.go               # Booking, conflict detection, alternatives
        service_test.go
      holiday/
        errors.go
        repository.go            # Holiday CRUD
        postgres_repository.go
        service.go               # Holiday fetching/caching, region filtering
        service_test.go
        nager_client.go          # HTTP client for Nager.Date API
      livekit/
        service.go               # LiveKit token generation, room naming
        service_test.go
  proto/
    work/
      v1/
        work.proto               # Extended with Calendar RPCs (add to existing)
  internal/
    gateway/
      route_calendar.go          # HTTP routes for Calendar endpoints
    models/
      calendar.go                # Calendar, CalendarMember, CalendarSubscription
      event.go                   # Event, EventAttendee, EventException, EventReminder
      resource.go                # Resource, ResourceAttribute, ResourceBooking
      holiday.go                 # PublicHoliday
      event_category.go          # EventCategory (user-defined color categories)
```

### Recommended Frontend Structure

```
desktop/src/renderer/src/
  modules/
    calendar/
      CalendarLayout.tsx          # Module layout with sidebar + main view
      CalendarSidebar.tsx         # Day agenda + calendar list + toggles
      views/
        WeekView.tsx              # 5 or 7 day week grid
        DayView.tsx               # Full-screen hourly timeline
        MonthView.tsx             # Month grid with event bars
        ViewToolbar.tsx           # View switcher, date nav, "Heute" button
      events/
        EventPopover.tsx          # Quick-create popover on time slot click
        EventForm.tsx             # Full event creation/edit form
        EventDetailPanel.tsx      # Slide-over event detail
        EventCard.tsx             # Event block in calendar grid
        RecurrenceEditor.tsx      # RRULE presets + custom editor
        RecurrenceEditScope.tsx   # "This event / Future / All" dialog
        ReminderEditor.tsx        # Per-event reminder settings
        AttendeePicker.tsx        # Invite colleagues + RSVP display
      calendars/
        CalendarListSidebar.tsx   # "My Calendars", "Shared", "Other" groups
        CalendarBrowsePage.tsx    # Browse + subscribe to shared calendars
        CalendarSettingsDialog.tsx # Calendar create/edit/permissions
        CalendarColorPicker.tsx   # Color selection for calendar/categories
      resources/
        ResourcesPage.tsx         # "Raeume" page with timeline view
        ResourceTimeline.tsx      # Horizontal timeline per room
        ResourceFilterBar.tsx     # Filter by capacity, equipment, floor
        ResourceBookingDialog.tsx # Booking form
        ResourceSelector.tsx      # Room dropdown in event form
      components/
        TimeGrid.tsx              # Shared hourly grid component (week + day)
        MonthGrid.tsx             # Shared month grid component
        EventBlock.tsx            # Positioned event block within time grid
        AllDayRow.tsx             # All-day event row above time grid
        TimeSlotOverlay.tsx       # Click/drag selection overlay
        CategoryBadge.tsx         # Color-coded event category indicator
        AvailabilityIndicator.tsx # Busy/available indicator for scheduling
        HolidayBadge.tsx          # Holiday marker in calendar cells
        TaskDeadlineLayer.tsx     # Toggleable PM task deadline overlay
  api/
    hooks/
      useCalendars.ts            # TanStack Query hooks for calendars
      useEvents.ts               # Events hooks (CRUD, recurrence)
      useResources.ts            # Resource/room hooks
      useHolidays.ts             # Holiday data hooks
  stores/
    calendar.ts                  # Zustand: selected calendars, view state, sidebar
```

### Pattern 1: RRULE Expansion in Service Layer

**What:** Server expands RRULE to concrete dates for a requested time window, merging exceptions.
**When to use:** Any event list/calendar view query.

```go
// internal/work/event/rrule.go
import "github.com/teambition/rrule-go"

func ExpandRecurrence(rruleStr string, dtstart time.Time, windowStart, windowEnd time.Time) ([]time.Time, error) {
    r, err := rrule.StrToRRule(rruleStr)
    if err != nil {
        return nil, fmt.Errorf("parse rrule: %w", err)
    }
    r.DTStart(dtstart)
    return r.Between(windowStart, windowEnd, true), nil
}

// In service.go, when listing events for a date range:
func (s *Service) ListEventsInRange(ctx context.Context, calendarIDs []uuid.UUID, start, end time.Time) ([]models.ExpandedEvent, error) {
    // 1. Fetch non-recurring events in range
    events, err := s.repo.ListEventsInRange(ctx, calendarIDs, start, end)

    // 2. Fetch recurring events that could have instances in range
    recurring, err := s.repo.ListRecurringEventsOverlapping(ctx, calendarIDs, start, end)

    // 3. Expand each recurring event
    for _, re := range recurring {
        occurrences, err := ExpandRecurrence(re.RRule, re.StartTime, start, end)
        // 4. Fetch exceptions for this event in range
        exceptions, err := s.repo.ListEventExceptions(ctx, re.ID, start, end)
        // 5. Apply exceptions (skip cancelled, modify changed)
        expanded := applyExceptions(re, occurrences, exceptions)
        events = append(events, expanded...)
    }

    return events, nil
}
```

### Pattern 2: Three-Way Recurring Event Edit

**What:** "This event" / "This and future events" / "All events" scope for recurring event edits.
**When to use:** Any modification to a recurring event.

```go
func (s *Service) UpdateRecurringEvent(ctx context.Context, eventID uuid.UUID, scope string, changes EventChanges) error {
    switch scope {
    case "this":
        // Create an exception for this specific occurrence
        return s.repo.CreateEventException(ctx, eventID, changes.OriginalDate, &changes)

    case "this_and_future":
        // 1. Set UNTIL on original event's RRULE to day before split
        err := s.repo.SetRRuleUntil(ctx, eventID, changes.OriginalDate.Add(-24*time.Hour))
        // 2. Create new event with modified properties, starting from split date
        newEvent := buildSplitEvent(originalEvent, changes)
        return s.repo.CreateEvent(ctx, newEvent)

    case "all":
        // Update the master event directly (all instances change)
        return s.repo.UpdateEvent(ctx, eventID, changes)
    }
}
```

### Pattern 3: PostgreSQL Exclusion Constraint for Double-Booking Prevention

**What:** Database-level enforcement that no two bookings for the same resource overlap.
**When to use:** Resource booking table.

```sql
-- Requires btree_gist extension (already standard in PostgreSQL)
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE resource_bookings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES calendar_events(id) ON DELETE CASCADE,
    booked_by UUID NOT NULL REFERENCES users(id),
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    cancelled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Prevent overlapping bookings for the same resource
    EXCLUDE USING GIST (
        resource_id WITH =,
        tstzrange(start_time, end_time) WITH &&
    ) WHERE (cancelled_at IS NULL)
);
```

### Pattern 4: LiveKit Meeting Link Generation

**What:** Auto-generate a LiveKit room link when creating an event with video call enabled.
**When to use:** Event creation with `has_video_call = true`.

```go
// internal/work/livekit/service.go
import (
    "github.com/livekit/protocol/auth"
)

type Service struct {
    apiKey    string
    apiSecret string
    wsURL     string  // e.g., "wss://livekit.example.com"
}

func (s *Service) GenerateMeetingLink(roomName string) string {
    return fmt.Sprintf("%s/room/%s", s.wsURL, roomName)
}

func (s *Service) GenerateJoinToken(roomName, identity string) (string, error) {
    at := auth.NewAccessToken(s.apiKey, s.apiSecret)
    grant := &auth.VideoGrant{
        RoomJoin: true,
        Room:     roomName,
    }
    at.AddGrant(grant).
        SetIdentity(identity).
        SetValidFor(24 * time.Hour)  // Token valid for event day
    return at.ToJWT()
}

// Room naming convention: "cal-{eventID}" for uniqueness
func RoomName(eventID uuid.UUID) string {
    return fmt.Sprintf("cal-%s", eventID.String()[:8])
}
```

### Pattern 5: Calendar Grid with CSS Grid (Frontend)

**What:** Custom week view time grid using CSS Grid for precise event positioning.
**When to use:** Week and day calendar views.

```typescript
// components/TimeGrid.tsx
const SLOT_HEIGHT = 48; // px per hour
const SLOTS_PER_HOUR = 4; // 15-minute intervals
const SLOT_HEIGHT_15MIN = SLOT_HEIGHT / SLOTS_PER_HOUR;

function TimeGrid({ days, events, onSlotClick, onEventResize }: TimeGridProps) {
  const hours = Array.from({ length: 24 }, (_, i) => i);

  return (
    <div
      className="grid"
      style={{
        gridTemplateColumns: `60px repeat(${days.length}, 1fr)`,
        gridTemplateRows: `auto repeat(${24 * SLOTS_PER_HOUR}, ${SLOT_HEIGHT_15MIN}px)`,
      }}
    >
      {/* Time labels column */}
      {hours.map(hour => (
        <div
          key={hour}
          className="text-xs text-muted-foreground pr-2 text-right"
          style={{ gridRow: `${hour * SLOTS_PER_HOUR + 2} / span ${SLOTS_PER_HOUR}` }}
        >
          {format(set(new Date(), { hours: hour }), 'HH:mm')}
        </div>
      ))}

      {/* Day columns with events */}
      {days.map((day, dayIndex) => (
        <DayColumn
          key={day.toISOString()}
          day={day}
          events={eventsForDay(events, day)}
          gridColumn={dayIndex + 2}
          onSlotClick={onSlotClick}
        />
      ))}
    </div>
  );
}

// Event positioning within a day column
function positionEvent(event: CalendarEvent): React.CSSProperties {
  const startMinutes = event.startTime.getHours() * 60 + event.startTime.getMinutes();
  const endMinutes = event.endTime.getHours() * 60 + event.endTime.getMinutes();
  const durationMinutes = endMinutes - startMinutes;

  return {
    position: 'absolute',
    top: `${(startMinutes / 60) * SLOT_HEIGHT}px`,
    height: `${(durationMinutes / 60) * SLOT_HEIGHT}px`,
    // For overlapping events: divide width by column count
    width: `${100 / event.columnCount}%`,
    left: `${(event.columnIndex / event.columnCount) * 100}%`,
  };
}
```

### Anti-Patterns to Avoid

- **Pre-computing all recurring event instances in the database**: Do NOT create an `event_instances` table with materialized rows for every recurrence. This creates a sync problem, wastes storage, and makes RRULE modifications (especially "this and future") extremely complex. Expand on-the-fly in the service layer.
- **Using react-big-calendar with CSS overrides**: Do NOT import react-big-calendar and override its SCSS with Tailwind. This creates a maintenance nightmare and limits control over interaction patterns.
- **Application-level double-booking checks without DB constraints**: Do NOT rely only on service-layer checks for room double-booking. Race conditions will allow double bookings. Use PostgreSQL exclusion constraints.
- **Storing timezone as string offset (e.g., "+02:00")**: Store IANA timezone names (e.g., "Europe/Berlin"). Offsets change with DST; timezone names are stable.
- **Fetching all calendars' events in separate queries**: Use a single query with `calendar_id IN (...)` for visible calendars, not N separate queries.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| RRULE parsing/expansion | Custom recurrence calculator | teambition/rrule-go (backend), rrule (frontend) | RFC 5545 is deceptively complex; BYSETPOS, WKST, timezone-aware expansion have hundreds of edge cases |
| Double-booking prevention | Application-level mutex/lock | PostgreSQL EXCLUDE USING GIST constraint | Database-level enforcement is race-condition-free, handles concurrent bookings atomically |
| Holiday calculations | Easter date algorithm + fixed dates | Nager.Date API + local cache | Easter varies by year, regional holidays vary by Bundesland/Kanton, edge cases everywhere |
| LiveKit token generation | Custom JWT signing | livekit/protocol auth package | Token format, grant structure, and signing must match LiveKit server expectations exactly |
| Overlapping event layout | Custom collision algorithm | Well-known column-packing algorithm | Google Calendar uses a specific algorithm to assign columns to overlapping events; implement the standard greedy interval partitioning |
| Notification delivery | Custom scheduling | Existing pg_notify event bus + notification service | Event types, delivery preferences, quiet hours, grouping all already built in Phase 4 |

**Key insight:** The calendar domain has more "looks simple, actually hard" problems than any previous phase. RRULE expansion, timezone handling, overlapping event layout, and double-booking prevention all have standard solutions that should be used rather than reinvented.

## Common Pitfalls

### Pitfall 1: Timezone Handling in Recurring Events

**What goes wrong:** A weekly meeting set for "every Tuesday at 10:00 CET" shows at 09:00 or 11:00 when DST changes.
**Why it happens:** Storing event times in UTC and converting naively ignores DST transitions.
**How to avoid:** Store event start_time with timezone (TIMESTAMPTZ in PostgreSQL). Store the user's IANA timezone (e.g., "Europe/Berlin") on the event. When expanding RRULE, expand in the event's timezone, then convert to UTC for storage/comparison. The `teambition/rrule-go` library supports timezone-aware expansion.
**Warning signs:** Events shifting by 1 hour after March/October DST transitions; recurring events at wrong times for users in different timezones.

### Pitfall 2: "This and Future Events" Edit Creates Orphaned Data

**What goes wrong:** Splitting a recurring series leaves attendee RSVPs, exceptions, and resource bookings pointing to the wrong event.
**Why it happens:** The original event ID is referenced by attendees, exceptions, and bookings. After splitting, some data belongs to the original series and some to the new one.
**How to avoid:** When splitting ("this and future events"):
1. Identify all exceptions and attendees on or after the split date
2. Move them to the new event (update foreign key)
3. Move resource bookings for instances on or after the split date
4. This must be a single transaction
**Warning signs:** RSVPs showing on wrong instances; resource bookings appearing on cancelled instances.

### Pitfall 3: Overlapping Event Layout Algorithm Complexity

**What goes wrong:** Events render on top of each other instead of side-by-side, or leave unused white space.
**Why it happens:** Naive layout assigns each event its own column; proper layout requires interval graph coloring.
**How to avoid:** Use the standard greedy column-packing algorithm:
1. Sort events by start time, then by duration (longest first)
2. For each event, find the first column where it doesn't overlap with any existing event
3. If no column is free, create a new column
4. Divide the day's width equally among the maximum column count
This is O(n*k) where n is events and k is max columns -- fast for typical calendars (< 20 events/day).
**Warning signs:** Events stacking vertically instead of side-by-side; columns with inconsistent widths.

### Pitfall 4: Month View "+N More" State Management

**What goes wrong:** Clicking "+2 more" in month view opens a popover, but the popover doesn't update when the user navigates to a different month, or the popover position is wrong after window resize.
**Why it happens:** Popover anchor position is cached, not recalculated on layout changes.
**How to avoid:** Use Radix UI Popover with its built-in positioning (uses Floating UI under the hood). The anchor is the "+N more" text element itself. Close all popovers on month navigation. Don't use absolute pixel positioning -- let Radix handle it.
**Warning signs:** Popover appearing in the wrong cell; popover floating detached from its anchor after scroll/resize.

### Pitfall 5: Race Condition in Resource Booking Without DB Constraints

**What goes wrong:** Two users book the same room for the same time within milliseconds of each other. Both get confirmation.
**Why it happens:** Application checks for conflicts, finds none, inserts -- but between the check and insert, the other user's insert completed.
**How to avoid:** Use PostgreSQL exclusion constraints (EXCLUDE USING GIST). The database itself rejects the second insert with a unique violation. Catch the error and return "Room occupied" with alternative suggestions.
**Warning signs:** Double bookings appearing in room timeline; users reporting conflicting reservations.

### Pitfall 6: N+1 Queries When Rendering Calendar Views

**What goes wrong:** Week view fetches events, then for each event fetches attendees, then calendar info, then resource info -- 100+ queries for a week with 30 events.
**Why it happens:** Naive repository that fetches related data per event.
**How to avoid:** Use SQL JOINs and subqueries:
- List events query includes calendar name/color via JOIN
- Attendee count as a subquery: `(SELECT COUNT(*) FROM event_attendees WHERE event_id = e.id) AS attendee_count`
- Resource name via LEFT JOIN on resource_bookings
- Batch-fetch attendee details only for the event detail view, not the calendar grid
**Warning signs:** Calendar view loading > 500ms; visible loading spinner when switching weeks.

## Code Examples

### Database Migration: Calendar Core Tables

```sql
-- Migration: 000032_create_calendars.up.sql

-- Calendars (personal and shared)
CREATE TABLE calendars (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    calendar_type VARCHAR(20) NOT NULL DEFAULT 'personal',  -- 'personal', 'shared', 'resource'
    color VARCHAR(7) NOT NULL DEFAULT '#4285F4',  -- default hex color
    owner_id UUID NOT NULL REFERENCES users(id),
    is_default BOOLEAN NOT NULL DEFAULT false,  -- user's primary calendar
    timezone VARCHAR(50) NOT NULL DEFAULT 'Europe/Berlin',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_calendars_owner ON calendars (owner_id);
CREATE INDEX idx_calendars_type ON calendars (calendar_type);

-- Calendar membership/subscriptions
CREATE TABLE calendar_members (
    calendar_id UUID NOT NULL REFERENCES calendars(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission VARCHAR(10) NOT NULL DEFAULT 'view',  -- 'view', 'edit', 'admin'
    color_override VARCHAR(7),  -- user's personal color override
    is_visible BOOLEAN NOT NULL DEFAULT true,  -- toggle visibility
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (calendar_id, user_id)
);

CREATE INDEX idx_calendar_members_user ON calendar_members (user_id);

-- User-defined event categories
CREATE TABLE event_categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    color VARCHAR(7) NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_event_categories_user ON event_categories (user_id);
CREATE UNIQUE INDEX idx_event_categories_name ON event_categories (user_id, LOWER(name));
```

### Database Migration: Events and Recurrence

```sql
-- Migration: 000033_create_events.up.sql

CREATE TABLE calendar_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    calendar_id UUID NOT NULL REFERENCES calendars(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    location TEXT,  -- free-text location
    resource_id UUID REFERENCES resources(id) ON DELETE SET NULL,  -- linked room/equipment
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    is_all_day BOOLEAN NOT NULL DEFAULT false,
    timezone VARCHAR(50) NOT NULL DEFAULT 'Europe/Berlin',
    -- Recurrence
    rrule TEXT,  -- RFC 5545 RRULE string, NULL for non-recurring
    recurrence_end TIMESTAMPTZ,  -- UNTIL from RRULE for query optimization
    -- Video call
    has_video_call BOOLEAN NOT NULL DEFAULT false,
    livekit_room_name VARCHAR(100),
    -- Category
    category_id UUID REFERENCES event_categories(id) ON DELETE SET NULL,
    -- Metadata
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_events_calendar ON calendar_events (calendar_id);
CREATE INDEX idx_events_time_range ON calendar_events (start_time, end_time);
CREATE INDEX idx_events_recurring ON calendar_events (calendar_id) WHERE rrule IS NOT NULL;
CREATE INDEX idx_events_resource ON calendar_events (resource_id) WHERE resource_id IS NOT NULL;
CREATE INDEX idx_events_created_by ON calendar_events (created_by);

-- Event attendees (RSVP)
CREATE TABLE event_attendees (
    event_id UUID NOT NULL REFERENCES calendar_events(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rsvp_status VARCHAR(10) NOT NULL DEFAULT 'pending',  -- 'pending', 'accepted', 'declined', 'tentative'
    responded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id, user_id)
);

CREATE INDEX idx_event_attendees_user ON event_attendees (user_id);

-- Event exceptions (for recurring event modifications/cancellations)
CREATE TABLE event_exceptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES calendar_events(id) ON DELETE CASCADE,
    original_date DATE NOT NULL,  -- the original occurrence date being modified/cancelled
    is_cancelled BOOLEAN NOT NULL DEFAULT false,
    -- Override fields (NULL = use parent event values)
    title VARCHAR(500),
    description TEXT,
    location TEXT,
    start_time TIMESTAMPTZ,
    end_time TIMESTAMPTZ,
    resource_id UUID REFERENCES resources(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (event_id, original_date)
);

CREATE INDEX idx_event_exceptions_event ON event_exceptions (event_id, original_date);

-- Event reminders
CREATE TABLE event_reminders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES calendar_events(id) ON DELETE CASCADE,
    minutes_before INTEGER NOT NULL,  -- e.g., 15, 60, 1440 (1 day)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_event_reminders_event ON event_reminders (event_id);

-- User calendar preferences (global defaults)
CREATE TABLE user_calendar_preferences (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    default_view VARCHAR(10) NOT NULL DEFAULT 'week',  -- 'day', 'week', 'month'
    week_days INTEGER NOT NULL DEFAULT 5,  -- 5 or 7
    default_reminder_minutes INTEGER NOT NULL DEFAULT 15,
    default_allday_reminder_minutes INTEGER NOT NULL DEFAULT 1080,  -- 18:00 day before (18*60)
    subdivision_code VARCHAR(10),  -- ISO-3166-2 for holiday filtering (e.g., 'DE-BY', 'CH-ZH')
    show_task_deadlines BOOLEAN NOT NULL DEFAULT true,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Database Migration: Resources and Room Booking

```sql
-- Migration: 000034_create_resources.up.sql

CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE resources (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    resource_type VARCHAR(20) NOT NULL,  -- 'room', 'equipment', 'vehicle'
    capacity INTEGER,
    floor VARCHAR(50),
    location VARCHAR(255),
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_resources_type ON resources (resource_type) WHERE is_active = true;

-- Resource equipment tags (beamer, whiteboard, etc.)
CREATE TABLE resource_tags (
    resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    tag VARCHAR(50) NOT NULL,
    PRIMARY KEY (resource_id, tag)
);

CREATE INDEX idx_resource_tags_tag ON resource_tags (tag);

-- Resource bookings with double-booking prevention
CREATE TABLE resource_bookings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    event_id UUID NOT NULL REFERENCES calendar_events(id) ON DELETE CASCADE,
    booked_by UUID NOT NULL REFERENCES users(id),
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    cancelled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Database-level double-booking prevention
    EXCLUDE USING GIST (
        resource_id WITH =,
        tstzrange(start_time, end_time) WITH &&
    ) WHERE (cancelled_at IS NULL)
);

CREATE INDEX idx_resource_bookings_resource ON resource_bookings (resource_id, start_time, end_time);
CREATE INDEX idx_resource_bookings_event ON resource_bookings (event_id);
```

### Database Migration: Holidays

```sql
-- Migration: 000035_create_holidays.up.sql

CREATE TABLE public_holidays (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    date DATE NOT NULL,
    name VARCHAR(255) NOT NULL,
    local_name VARCHAR(255) NOT NULL,
    country_code CHAR(2) NOT NULL,  -- 'DE', 'AT', 'CH'
    is_global BOOLEAN NOT NULL DEFAULT false,  -- applies to entire country
    subdivision_codes TEXT[],  -- ISO-3166-2 codes: {'DE-BY', 'DE-NW'} or NULL for global
    holiday_type VARCHAR(20) NOT NULL DEFAULT 'public',  -- 'public', 'bank', 'observance'
    year INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (date, country_code, name)
);

CREATE INDEX idx_holidays_date_range ON public_holidays (date, country_code);
CREATE INDEX idx_holidays_year_country ON public_holidays (year, country_code);
CREATE INDEX idx_holidays_subdivision ON public_holidays USING GIN (subdivision_codes);
```

### RRULE Frontend Preview

```typescript
// Using rrule npm package for client-side recurrence preview
import { RRule, Frequency } from 'rrule';

function getRecurrencePreview(rruleStr: string, count = 5): Date[] {
  const rule = RRule.fromString(rruleStr);
  return rule.all((_, i) => i < count);
}

// Preset RRULE strings for the recurrence editor
const RECURRENCE_PRESETS = {
  daily: 'FREQ=DAILY',
  weekdays: 'FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR',
  weekly: (day: string) => `FREQ=WEEKLY;BYDAY=${day}`,
  biweekly: (day: string) => `FREQ=WEEKLY;INTERVAL=2;BYDAY=${day}`,
  monthly: (dayOfMonth: number) => `FREQ=MONTHLY;BYMONTHDAY=${dayOfMonth}`,
  yearly: (month: number, day: number) => `FREQ=YEARLY;BYMONTH=${month};BYMONTHDAY=${day}`,
};
```

### Event Query with Calendar Join (Backend)

```sql
-- Fetch events for visible calendars in a date range
SELECT
    e.id, e.title, e.description, e.location,
    e.start_time, e.end_time, e.is_all_day, e.timezone,
    e.rrule, e.has_video_call, e.livekit_room_name,
    c.id AS calendar_id, c.name AS calendar_name, c.color AS calendar_color,
    cm.color_override,
    COALESCE(cm.color_override, c.color) AS display_color,
    ec.name AS category_name, ec.color AS category_color,
    r.name AS resource_name,
    (SELECT COUNT(*) FROM event_attendees WHERE event_id = e.id) AS attendee_count,
    (SELECT rsvp_status FROM event_attendees WHERE event_id = e.id AND user_id = $4) AS my_rsvp
FROM calendar_events e
JOIN calendars c ON e.calendar_id = c.id
LEFT JOIN calendar_members cm ON cm.calendar_id = c.id AND cm.user_id = $4
LEFT JOIN event_categories ec ON e.category_id = ec.id
LEFT JOIN resources r ON e.resource_id = r.id
WHERE e.calendar_id = ANY($1)
  AND (
    -- Non-recurring: event overlaps with range
    (e.rrule IS NULL AND e.start_time < $3 AND e.end_time > $2)
    OR
    -- Recurring: event started before range end and recurrence hasn't ended before range start
    (e.rrule IS NOT NULL AND e.start_time < $3 AND (e.recurrence_end IS NULL OR e.recurrence_end > $2))
  )
ORDER BY e.start_time;
-- $1: calendar_ids[], $2: range_start, $3: range_end, $4: current_user_id
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Pre-computed event_instances table | On-the-fly RRULE expansion | Standard practice | Simpler schema, no sync jobs, CalDAV-compatible |
| Application-level booking conflict check | PostgreSQL exclusion constraints (GIST) | PostgreSQL 9.2+ | Race-condition-free, atomic enforcement |
| SCSS-based calendar libraries (react-big-calendar) | Custom CSS Grid calendars or CSS-variable-based theming | 2024-2025 trend | Better Tailwind integration, full design control |
| Integer UTC offsets for timezone | IANA timezone names (Europe/Berlin) | Always | Correct DST handling |
| react-beautiful-dnd for event dragging | Native pointer events + custom drag handlers | 2023+ (rbd deprecated) | Calendar drag (resize + move) is simpler than Kanban sortable -- pointer events suffice |
| Separate calendar microservice | Extended Work/Biz service | Service consolidation decision | Fewer services to maintain, shared DB transaction scope |

**Deprecated/outdated:**
- `react-beautiful-dnd` for calendar event dragging: Deprecated. Use native pointer events for calendar drag/resize.
- UTC offset storage: Always use IANA timezone names.
- Pre-computed instances table: Overhead not justified when service-layer expansion is fast.

## Open Questions

1. **Calendar service vs Work service scope**
   - What we know: The service consolidation decision groups functionality into Work/Biz/Automation services. Calendar is logically part of "Work" (scheduling is work-related).
   - What's unclear: Adding 40+ RPCs to the existing Work proto file (already 88 RPCs) may become unwieldy. Should calendar have its own proto file but still run in the Work service binary?
   - Recommendation: Create a separate `proto/calendar/v1/calendar.proto` file for clean separation, but have the Work service binary (`cmd/work/main.go`) register both WorkService and CalendarService gRPC servers on the same port. This keeps the service count manageable while allowing clean proto separation. Alternative: add all RPCs to work.proto (simpler, one gRPC client in gateway).

2. **Recurring event reminders scheduling**
   - What we know: Event reminders need to fire N minutes before each occurrence. The notification system exists (Phase 4).
   - What's unclear: Who schedules the reminder trigger? A background job? The gateway on WebSocket connection?
   - Recommendation: A lightweight cron-like goroutine in the Work service that queries "events with reminders in the next 5 minutes" every minute. For each match, emit a pg_notify event. This avoids scheduling individual jobs per occurrence and works naturally with RRULE expansion. The notification service picks up the event and delivers via WebSocket/push.

3. **Task deadline layer data source**
   - What we know: Task deadlines from PM module should appear as a toggleable layer. The Work service has both tasks and calendar data.
   - What's unclear: Should the calendar view query tasks directly, or should the frontend fetch both events and tasks in parallel?
   - Recommendation: Add a dedicated RPC `ListTaskDeadlinesInRange(start, end)` that returns lightweight task stubs (id, title, due_date, project_key, priority). The frontend fetches this alongside events and renders as a distinct layer. This keeps the calendar event query fast and the layer toggle a client-side filter.

4. **LiveKit configuration dependency**
   - What we know: LiveKit is in the tech stack (CLAUDE.md) but no LiveKit Go SDK is in go.mod yet. CAL-06 requires auto-generating a LiveKit room link.
   - What's unclear: Is LiveKit actually deployed yet? Do we have API keys configured?
   - Recommendation: Implement the LiveKit integration behind a feature flag. If LiveKit config (API key, secret, WS URL) is not set in environment variables, the "Video Call" toggle is hidden in the UI. This way the calendar works without LiveKit, and video integration activates automatically when LiveKit is deployed (likely Phase 8).

## Sources

### Primary (HIGH confidence)
- Codebase analysis: All patterns verified by reading actual source files
  - `backend/internal/work/*` -- service/repository/error pattern for extending Work service
  - `backend/internal/notification/event/*` -- event bus, types, emit pattern
  - `backend/internal/config/config.go` -- environment variable configuration pattern
  - `backend/cmd/work/main.go` -- service entry point pattern
  - `backend/internal/gateway/route_work.go` -- route registrar pattern
  - `backend/proto/work/v1/work.proto` -- gRPC proto definition pattern
  - `backend/migrations/000024-000031` -- migration patterns
  - `desktop/package.json` -- current dependency versions
  - `desktop/src/renderer/src/modules/work/` -- frontend module structure

### Secondary (MEDIUM confidence)
- RFC 5545 (iCalendar specification) -- RRULE format, THISANDFUTURE pattern, RECURRENCE-ID
- teambition/rrule-go GitHub + pkg.go.dev -- API, usage examples, RFC compliance
- Nager.Date API (date.nager.at) -- REST API, subdivision support verified
- LiveKit official docs (docs.livekit.io) -- Go SDK token generation
- PostgreSQL documentation -- EXCLUDE USING GIST, btree_gist, tstzrange
- react-big-calendar vs FullCalendar comparison articles -- confirmed decision to build custom

### Tertiary (LOW confidence)
- codegenes.net blog -- hybrid schema pattern for recurring events (used for design comparison only)
- Various Google Calendar clone repositories -- CSS Grid layout patterns
- WebSearch results for DACH holiday data sources -- Nager.Date confirmed as best option

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all backend libraries already in use; only new dependencies are rrule-go, LiveKit SDK, and rrule (npm)
- Architecture: HIGH -- extends existing Work service following verified patterns
- Data model: HIGH -- RRULE string storage + exceptions table is the CalDAV-compatible standard; exclusion constraints are PostgreSQL best practice
- Frontend calendar grid: MEDIUM -- custom build is more work than a library but gives full control; CSS Grid approach verified in multiple open-source implementations
- Holiday data: MEDIUM -- Nager.Date API is external dependency; fallback approach documented
- LiveKit integration: MEDIUM -- SDK API verified via official docs but LiveKit deployment status unknown
- Pitfalls: HIGH -- timezone, double-booking, and RRULE modification pitfalls are well-documented in calendar engineering literature

**Research date:** 2026-02-08
**Valid until:** 2026-03-08 (stable domain; calendar standards don't change)
