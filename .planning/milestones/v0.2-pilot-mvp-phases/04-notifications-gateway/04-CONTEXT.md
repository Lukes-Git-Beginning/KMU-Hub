# Phase 4: Notifications + Gateway Modernization - Context

**Gathered:** 2026-02-07
**Status:** Ready for planning

<domain>
## Phase Boundary

Centralized notification system that any module can use to notify users in real time, plus gateway refactoring so adding new backend services requires only a route handler registration (lazy connections, graceful degradation). This phase does NOT build the Desktop Electron app (Phase 5) but prepares the notification infrastructure that Phase 5 will render.

</domain>

<decisions>
## Implementation Decisions

### Notification UX
- Dropdown panel accessed via bell icon (like GitHub/Slack) — stays on current page
- Smart grouping: collapse similar events (e.g., "5 new messages in #general") with expand/collapse
- Capped badge count: exact number up to threshold, then "99+" style cap
- Click marks individual notification as read and navigates to source; "Mark all as read" button available; viewing the panel does NOT auto-mark as read

### Notification Preferences
- Two-level granularity: module-level defaults + per-event-type overrides
- Quiet hours: scheduled recurring DND (e.g., 18:00-08:00) as default + manual toggle override for ad-hoc situations
- Per-resource muting: users can mute specific chat channels, CRM pipelines, or any future module resource — overrides event-type settings for that resource

### Event Taxonomy
- Chat events: default to mentions + DMs only; users can opt into "all messages" per channel
- 3-tier priority system:
  - **Urgent**: always delivered, even during DND (e.g., system alerts, direct escalations)
  - **Normal**: delivered per user preferences (e.g., @mentions, deal assignments)
  - **Low**: in-app only, batched/collapsed (e.g., informational updates)
- CRM events: notify on assignment/ownership changes and deal stage transitions by default
- Generic event bus: module-agnostic event type registry — any module registers event types + default preferences; future modules plug in without notification service changes

### Desktop Push Behavior
- Click-to-navigate: clicking a desktop push notification opens the Hub and deep-links directly to the source item (message, deal, task)
- Quick action buttons: 1-2 action buttons on push notifications (e.g., "Reply" on messages, "Mark as read")
- Notification sounds: enabled by default with preset options; different sounds per priority tier; user can disable or choose presets
- System tray: dot indicator (colored dot when unread notifications exist) — subtle, not a badge count

### Claude's Discretion
- Delivery channels selection (in-app bell + desktop push baseline; email digest if it makes sense for desktop-first app)
- Gateway refactoring technical approach (lazy gRPC connections, per-service route handlers, graceful degradation patterns)
- Event bus implementation (PostgreSQL LISTEN/NOTIFY vs other patterns)
- Notification storage schema and retention policy
- Smart grouping algorithm details
- Sound preset selection

</decisions>

<specifics>
## Specific Ideas

- Notification bell should feel snappy — dropdown panel, not a page navigation
- Smart grouping is important to avoid notification fatigue from Chat (many messages collapse into "X new messages in #channel")
- The event bus must be designed for extensibility from day 1 — Phases 5-13 all register their own event types without touching the notification service
- Per-resource muting is critical for Chat channels but should work generically (future: mute a specific project, a specific deal pipeline, etc.)
- DND must have both scheduled and manual modes — scheduled for work-life balance, manual for "in a meeting right now"
- Priority tiers: Urgent should break through DND (sparingly used), Low should never generate desktop push

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 04-notifications-gateway*
*Context gathered: 2026-02-07*
