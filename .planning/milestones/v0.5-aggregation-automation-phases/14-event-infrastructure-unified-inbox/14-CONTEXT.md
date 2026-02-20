# Phase 14: Event Infrastructure + Unified Inbox - Context

**Gathered:** 2026-02-19
**Status:** Ready for planning

<domain>
## Phase Boundary

Two deliverables in one phase:

1. **Event Infrastructure** — All existing services (CRM, Chat, Notifications, Email, Documents, Finance, HR) emit structured events via a PostgreSQL `events` table + `pg_notify`. This is the foundation for Phase 16 Automation Engine and powers real-time updates to the Unified Inbox.

2. **Unified Inbox ("Kommunikation")** — A single aggregated inbox across Email, Chat DMs/@mentions, and Notifications. Users triage, reply, snooze, and route items without switching modules. Team inboxes enable shared mailbox workflows with assignment and routing rules.

**Frontend naming:** "Kommunikation" module = external/cross-module comms. "Chat" stays as the internal team messaging module.

**NOT in scope:** Automation triggers/actions (Phase 16), calendar sync (Phase 15), Teams/Slack integration (Phase 17). Routing rules here are inbox-specific; cross-module workflow automation is Phase 16.

</domain>

<decisions>
## Implementation Decisions

### Inbox layout & item display
- Three-column layout: channel/filter sidebar | message list | detail/preview pane (Outlook/Gmail desktop pattern)
- Items from different channels distinguished by colored icon badges (blue envelope for email, green chat bubble, orange bell for notifications)
- Message list shows two-line items: Line 1 = sender name + timestamp, Line 2 = subject or first-line preview snippet
- Left sidebar organized in three sections:
  - **Smart views** (top): All, Unread, Starred/Flagged, Assigned to me
  - **Channel filters** (middle): Email, Chat, Notifications
  - **Team inboxes** (bottom): dynamically listed per user membership

### Triage & actions
- GTD-style inbox-zero workflow: process each item by replying, delegating, snoozing, or archiving
- Snooze with presets (1h, tomorrow morning, next week) AND custom date/time picker — item disappears and reappears in inbox at chosen time
- Full quick-action set on hover in message list: reply, snooze, archive, assign, star — handle most items without opening detail pane
- Inline reply works for ALL channel types — the reply box in the detail pane adapts per channel (email composer for email, chat input for chat, action buttons for notifications)

### Team inboxes & assignment
- Each team inbox configurable as either manual-claim OR round-robin auto-assign (team admin chooses the mode)
- Visibility configurable per team: open (everyone sees all items) or private (only see unassigned queue + own assigned items)
- Users can belong to unlimited team inboxes; each appears in their sidebar
- Team inbox creation restricted to admin + manager roles

### Routing rules
- AND/OR condition builder (not just single conditions) — e.g., `(channel = email AND sender contains @firma.de) OR (subject contains 'Dringend')`
- Actions: route to team inbox, auto-assign to person, add tags/labels, AND auto-reply (e.g., "Wir haben Ihre Nachricht erhalten")
- Both global rules (apply to all channels) and per-channel rules (channel-specific overrides/additions)
- Rich condition fields: sender/author, subject line, body content keywords, CRM contact link status, existing tags

### Claude's Discretion
- Event infrastructure architecture (events table schema, pg_notify channel naming, consumer framework design)
- Channel adapter implementation patterns (normalization schema, adapter interfaces)
- Materialized inbox table design and update strategy
- Loading states, empty states, error handling
- Exact color values for channel badges (should fit the existing desk theme system)
- Keyboard shortcuts for power-user triage

</decisions>

<specifics>
## Specific Ideas

- The inbox should feel like a productivity tool (Gmail + Linear hybrid), not a passive notification feed — the user actively processes items toward inbox zero
- "Kommunikation" is the external-facing comms hub; Chat remains the internal team messaging tool
- Routing rules in this phase are specifically for inbox item routing — general-purpose automation (trigger-condition-action across all modules) is Phase 16's domain, but the condition evaluator should be designed with reuse in mind
- Auto-reply action is important for customer-facing teams (Support, Sales) who need instant acknowledgment

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 14-event-infrastructure-unified-inbox*
*Context gathered: 2026-02-19*
