# Phase 6: Project Management - Context

**Gathered:** 2026-02-08
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can manage their daily work through tasks and projects without leaving the Hub. Delivers: Work gRPC service, task/project CRUD, list + Kanban views, comments with @mentions, file attachments, CRM/chat entity linking, search, filters, and custom fields. Does NOT include: Gantt charts, time tracking, sprint planning, or resource allocation.

</domain>

<decisions>
## Implementation Decisions

### Task structure & workflow
- **Statuses:** Per-project custom statuses. Each project defines its own status set (columns). No global defaults imposed — project creator configures statuses at project creation.
- **Subtasks:** Multi-level nesting supported. Tasks can nest arbitrarily deep (task > subtask > sub-subtask). Nesting reflected in both list and Kanban views.
- **Priorities:** 4 levels — Urgent, High, Medium, Low. Color-coded (red, orange, yellow, blue).
- **Dependencies:** Multiple dependency types supported — blocks, is blocked by, relates to, duplicates. Blocked tasks show a visual indicator. No Gantt visualization in this phase, but the dependency data model supports future Gantt rendering.

### Project organization
- **Access control:** Claude's Discretion (see below)
- **Default view:** No project-level default. Each user selects list or Kanban independently per project. Preference persisted per user per project.
- **Templates:** Admins/managers can save a project as a template. Creating from template pre-populates tasks, statuses, and custom fields.
- **Standalone tasks:** Allowed. Users can create personal tasks not tied to any project, shown in a "My Tasks" view. Tasks can optionally be moved into a project later.

### Views & interaction
- **Kanban columns:** Columns = project statuses. Dragging a card changes its status. One column per status.
- **List grouping:** User-selectable grouping dimension — by status, by assignee, by priority, by due date, or flat (no grouping). Sortable within groups.
- **Task detail:** Opens as a slide-over side panel first (keeps list/board visible). User can click "expand" to navigate to a full task page for heavy editing (comments, attachments, activity log).
- **Inline editing:** Status, assignee, priority, and due date editable directly from list/board views. Title editable by clicking it. All other fields require the detail panel.

### CRM & cross-module linking
- **Linking UX:** Search-and-link field with context-aware auto-suggest. Task title/description content triggers entity suggestions (e.g., "Acme" suggests Acme Corp). User confirms suggestions. Can link multiple entities.
- **Linkable entity types:** CRM contacts, companies, deals, AND chat channels/messages. Clicking a link navigates to that entity.
- **CRM-side display:** Both — a dedicated "Tasks" tab on contact/deal detail pages showing linked tasks with status/assignee/due date, PLUS activity timeline entries when task status changes occur.
- **Auto-populate from CRM:** When creating a task from a CRM deal, auto-fills linked entity (the deal), title prefix (deal name), and assignee (deal owner). User can change everything before saving.

### Claude's Discretion
- Project access control model — Claude picks the right approach for KMU teams (5-200 employees), balancing visibility with privacy
- Task numbering/ID scheme (sequential per project, global, etc.)
- Comment threading model (flat vs threaded)
- Activity log detail level
- Empty state designs
- Loading skeletons and transitions

</decisions>

<specifics>
## Specific Ideas

- Multi-level nesting like ClickUp — tasks can nest arbitrarily, but the UI should stay clean (collapse/expand, not overwhelming depth indicators)
- Kanban drag-and-drop should feel snappy and immediate (optimistic updates)
- Side panel + expand pattern like Linear — quick glance in panel, full context on expand
- Context-aware auto-suggest for CRM linking should be helpful but not intrusive — suggestions shown, never auto-applied
- Chat message linking enables traceability: "This task was discussed in #sales" with a clickable reference

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 06-project-management*
*Context gathered: 2026-02-08*
