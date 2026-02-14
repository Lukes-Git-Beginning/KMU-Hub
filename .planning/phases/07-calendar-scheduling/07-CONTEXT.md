# Phase 7: Calendar & Scheduling - Context

**Gathered:** 2026-02-08
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can manage their schedules, book meetings, and coordinate team availability entirely within the Hub. This includes personal and shared calendars with day/week/month views, recurring events (RRULE), RSVP invitations, room and equipment booking, and DACH holiday support. Video call integration (LiveKit link generation) is in scope. Full video/voice calling is Phase 8.

</domain>

<decisions>
## Implementation Decisions

### Calendar views & navigation
- Default view is **week view** when opening the Calendar module
- Week view shows 5 days (Mon-Fri) by default, **user-configurable** to 5 or 7 days in settings
- **Day agenda sidebar** instead of mini month calendar -- sidebar shows a chronological event list for the selected day, updates when clicking a date in the main view
- Full-screen day view uses an **hourly timeline** (vertical time grid with events placed at their time position)
- Month view shows 2-3 events per cell, then **"+N more" link** that opens a popover/expands
- **Prominent "Heute" button** in toolbar for one-click return to current date
- Task deadlines from PM module appear as a **separate toggleable layer** (distinct color/style) that can be switched on/off

### Event creation & editing
- **Both** creation methods: click/drag on time slot in calendar AND a "+ New Event" button
- Click on time slot opens a **compact popover** (title + time + optional details) with a "Mehr Optionen" link to the full form
- **User-defined color categories** for events (e.g., "Meeting", "Focus time", "Travel") -- users create their own categories with colors
- Location field: **free-text for external locations** plus a **separate "Room" dropdown** that links to resource calendars
- Reminders: **per-event override with global default** -- global default (e.g., 15 min before), overridable per individual event
- Event description is **plain text** (no rich text editor)
- Recurring events: **presets + custom** -- quick presets (daily, weekly, monthly, yearly) plus a "Custom..." option for full RRULE control (every 2 weeks, specific days, etc.)
- Editing recurring events: **three-way scope** -- "This event" / "This and future events" / "All events" (Google Calendar pattern)

### Shared calendars & overlay
- **Role-based creation**: managers+ can create shared calendars, members can only view/join ones they have access to
- Overlapping events from different calendars displayed as **side-by-side columns** in the same time slot
- Colleague availability shown as **simple busy/available indicator** per time slot when scheduling (no detailed calendar view of others)
- **Three-level permissions**: View / Edit / Admin per shared calendar
- Calendar colors: **default color per calendar + user override** (each user can customize how a calendar appears on THEIR view)
- **Calendar list in sidebar**, grouped by type: "My calendars", "Shared calendars", "Other" with toggle checkboxes
- Calendar discovery: **browse + subscribe** -- a "Browse calendars" page showing all shared calendars the user has access to, with subscribe button
- **Full notification integration**: event invitations create notifications via Phase 4 system, RSVP responses update event and notify organizer

### Room booking & resources
- **Rooms + equipment** as bookable resources (meeting rooms, beamers, company cars, phone booths, etc.)
- Room availability shown as **horizontal timelines per room** for the selected day -- click empty slot to book
- Room attributes: **capacity, equipment tags (beamer, whiteboard), floor/location** -- users can filter rooms by these
- Double-booking: **blocked with alternative suggestions** (show "Room occupied" with list of available alternative rooms for the same time)
- Resource management: **admin + managers** can create, edit, and delete resources
- Booking flow: **instant booking** -- first come, first served, no approval required
- Booking a resource **automatically creates a calendar event** on the booker's personal calendar
- **Dual entry points**: dedicated "Raeume" page with timeline view AND room selector within event creation form

### Claude's Discretion
- Exact calendar grid component library / implementation approach
- Sidebar layout proportions and responsive behavior
- RRULE storage format details (RFC 5545 compliance is required, implementation details flexible)
- Holiday data source and storage approach
- Notification timing defaults
- Event drag-to-resize behavior in week/day views
- Keyboard shortcuts for calendar navigation

</decisions>

<specifics>
## Specific Ideas

- Sidebar is a **day agenda** (not a mini month grid) -- shows today's events chronologically, updates when clicking a date in the main view
- Calendar should feel like Google Calendar in interaction patterns (click-to-create, popover quick create, "+N more" overflow) but with the KMU Hub design language
- Week start is always Monday (DACH standard, per CAL-07)
- Room timeline view should feel like a physical meeting room booking display -- clear visual blocks showing occupied/free

</specifics>

<deferred>
## Deferred Ideas

None -- discussion stayed within phase scope

</deferred>

---

*Phase: 07-calendar-scheduling*
*Context gathered: 2026-02-08*
