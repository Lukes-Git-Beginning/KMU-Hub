# Phase 6: Project Management - Research

**Researched:** 2026-02-08
**Domain:** Task/project management backend (Go gRPC) + frontend (React Kanban/List views)
**Confidence:** HIGH (codebase patterns verified, external libs confirmed via docs)

## Summary

Phase 6 introduces the "Work" gRPC microservice -- a new backend service following the exact same architecture as existing CRM, Chat, Auth, and Notification services. The backend work is well-understood because it follows established patterns: repository interface, service layer, gRPC server, gateway route registrar, PostgreSQL migrations, and event emission for notifications.

The frontend introduces the first drag-and-drop interaction in the desktop app (the existing DealPipelineView is read-only columns without DnD). The Kanban board requires a DnD library. Two strong options exist: `@dnd-kit/core` + `@dnd-kit/sortable` (modular, hook-based, React-native) and `@hello-pangea/dnd` (simpler API, fork of react-beautiful-dnd). Given the project already uses Radix UI + Tailwind + shadcn/ui patterns, and needs fine-grained control for multi-level subtask nesting in Kanban, `@dnd-kit` is the stronger choice.

The data model is the most architecturally significant decision: multi-level task nesting with dependencies. The adjacency list approach (parent_id column) is the recommended strategy -- it matches the project's PostgreSQL-first philosophy, keeps writes simple, and PostgreSQL recursive CTEs handle read queries efficiently for the expected hierarchy depth (3-5 levels max in practice). A materialized `depth` column avoids recursive queries for common operations.

**Primary recommendation:** Follow the existing service pattern exactly (new `cmd/work`, `internal/work/*`, `proto/work/v1/work.proto`), use adjacency list with `parent_task_id` for nesting, `@dnd-kit` for Kanban DnD, and project-level membership for access control.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Task structure & workflow:**
- Per-project custom statuses. Each project defines its own status set (columns). No global defaults imposed -- project creator configures statuses at project creation.
- Multi-level nesting supported. Tasks can nest arbitrarily deep (task > subtask > sub-subtask). Nesting reflected in both list and Kanban views.
- 4 priority levels: Urgent, High, Medium, Low. Color-coded (red, orange, yellow, blue).
- Multiple dependency types: blocks, is blocked by, relates to, duplicates. Blocked tasks show visual indicator. Data model supports future Gantt rendering.

**Project organization:**
- No project-level default view. Each user selects list or Kanban independently per project. Preference persisted per user per project.
- Templates: Admins/managers can save a project as a template. Creating from template pre-populates tasks, statuses, and custom fields.
- Standalone tasks allowed. Users can create personal tasks not tied to any project, shown in a "My Tasks" view. Tasks can optionally be moved into a project later.

**Views & interaction:**
- Kanban columns = project statuses. Dragging a card changes its status. One column per status.
- List grouping: User-selectable grouping dimension -- by status, by assignee, by priority, by due date, or flat (no grouping). Sortable within groups.
- Task detail: Opens as slide-over side panel first (keeps list/board visible). User can click "expand" to full task page for heavy editing (comments, attachments, activity log).
- Inline editing: Status, assignee, priority, and due date editable directly from list/board. Title editable by clicking it. All other fields require detail panel.

**CRM & cross-module linking:**
- Search-and-link field with context-aware auto-suggest. Task title/description triggers entity suggestions. User confirms. Can link multiple entities.
- Linkable entity types: CRM contacts, companies, deals, AND chat channels/messages.
- CRM-side display: Dedicated "Tasks" tab on contact/deal detail pages showing linked tasks with status/assignee/due date, PLUS activity timeline entries on task status changes.
- Auto-populate from CRM: When creating a task from a CRM deal, auto-fills linked entity, title prefix (deal name), and assignee (deal owner). User can change before saving.

### Claude's Discretion
- Project access control model -- Claude picks the right approach for KMU teams (5-200 employees)
- Task numbering/ID scheme (sequential per project, global, etc.)
- Comment threading model (flat vs threaded)
- Activity log detail level
- Empty state designs
- Loading skeletons and transitions

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

## Discretion Recommendations

### Project Access Control Model

**Recommendation: Project-level membership with roles (owner/member/viewer)**

For KMU teams (5-200 employees), a project-level membership model balances visibility with privacy:

| Role | Can View | Can Edit Tasks | Can Manage Project |
|------|----------|----------------|-------------------|
| Owner | Everything | Everything | Settings, members, statuses, templates |
| Member | Everything | Own + assigned tasks, create tasks | No |
| Viewer | Everything | Nothing | No |

**Implementation:** A `project_members` junction table with `project_id`, `user_id`, `role` (enum: owner/member/viewer). The project creator is automatically `owner`. Admins (system-level) can access all projects. Non-members cannot see or access the project at all.

**Rationale:** KMUs need simple, visible-by-default team projects (most are shared) but also private projects for HR, management, or personal use. This model avoids complex permission matrices while still allowing privacy. Standalone tasks (no project) are always private to the creator.

**Confidence:** HIGH -- matches existing RBAC pattern (system roles: admin/manager/member) while adding project-scoped access.

### Task Numbering Scheme

**Recommendation: Per-project sequential numbers with project prefix**

Format: `{PROJECT_KEY}-{SEQUENCE}` (e.g., `ACME-42`, `HR-7`, `SALES-156`)

- Each project gets a unique short key (2-10 uppercase alphanumeric characters, set at project creation, unique across org).
- Tasks within a project get auto-incrementing integer numbers starting at 1.
- The project_key + task_number combination is the human-readable identifier.
- The database primary key remains UUID for consistency with the rest of the system.
- Standalone tasks (no project) use a system project key like `TASK` (e.g., `TASK-1`).

**Implementation:** A `next_task_number` counter column on the `projects` table, incremented atomically via `UPDATE ... RETURNING`. The `task_number` column on tasks is an integer, not the formatted string -- formatting happens at the API layer.

**Confidence:** HIGH -- this is the standard pattern used by Jira, Linear, Asana, and others. Users can reference tasks in comments and chat using the short key.

### Comment Threading Model

**Recommendation: Flat comments with quote-reply**

Flat (non-threaded) comments displayed in chronological order. Users can quote a previous comment when replying (shows a preview snippet of the quoted comment above the reply). No nested threads.

**Rationale:**
- Task comments are typically short and focused (unlike chat). Threading adds UI complexity without proportional value.
- The chat module already has full threading support -- users who want threaded discussions can use chat and link the channel/message to the task.
- Flat is easier to implement, easier to search, and easier to follow for small teams.
- Linear, Asana, and Notion all use flat comments on tasks (threads are for chat/messages).

**Confidence:** MEDIUM -- this is a design preference. Flat is simpler to implement and maintains feature velocity. Can always add threading later.

### Activity Log Detail Level

**Recommendation: Field-level change tracking for key fields**

Track changes to: status, assignee, priority, due_date, title, description, linked entities, custom fields. Each change recorded as a structured `task_activity` entry with:
- `actor_id` (who made the change)
- `action` (created, updated, status_changed, assigned, commented, attachment_added, linked, unlinked)
- `field_name` (which field changed, nullable for non-field actions)
- `old_value` / `new_value` (JSON-encoded, nullable)
- `timestamp`

Display as a chronological activity feed on the task detail page, interleaved with comments. Status changes and assignments are the most prominent entries (larger, with color-coded indicators). Field edits are shown in a compact format ("Luke changed priority from Medium to High").

**Confidence:** HIGH -- this is standard practice in project management tools.

### Empty State Designs

**Recommendation: Contextual empty states with clear call-to-action**

- **No projects:** Illustration + "Erstellen Sie Ihr erstes Projekt" + Create Project button
- **Empty project (no tasks):** "Dieses Projekt hat noch keine Aufgaben" + Create Task button + suggestion to import from template
- **Empty Kanban column:** Dashed border placeholder card with "Aufgabe hierher ziehen oder neue erstellen"
- **My Tasks empty:** "Keine offenen Aufgaben" + suggestion to create standalone task or browse projects
- **No search results:** "Keine Ergebnisse fuer '{query}'" + suggestion to broaden filters

**Confidence:** HIGH -- follows existing patterns in the codebase (DealPipelineView already has empty column states).

### Loading Skeletons and Transitions

**Recommendation: Skeleton screens matching content layout**

- **Project list:** Skeleton rows with placeholder blocks for name, member avatars, task count
- **Kanban board:** Skeleton columns (same as DealPipelineView pattern) with 3 placeholder cards each
- **Task list:** Skeleton table rows
- **Task detail panel:** Skeleton blocks for title, metadata row, description area
- **Optimistic updates for DnD:** Card moves immediately on drag-end, no loading state. If server rejects, card snaps back with a toast error.

**Confidence:** HIGH -- existing `Skeleton` component from shadcn/ui already in use; DealPipelineView has skeleton loading pattern to reuse.

## Standard Stack

### Core (Backend - Go)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go standard library + slog | 1.25.6 | Service implementation, structured logging | Already in use across all services |
| google.golang.org/grpc | 1.78.0 | Inter-service communication | Already in use; Work service follows same pattern |
| github.com/jackc/pgx/v5 | 5.8.0 | PostgreSQL driver + connection pool | Already in use |
| github.com/google/uuid | 1.6.0 | UUID generation for all entities | Already in use |
| google.golang.org/protobuf | 1.36.11 | Proto serialization | Already in use |
| github.com/minio/minio-go/v7 | 7.0.98 | File attachments (reuse chat file infrastructure) | Already in use for chat files |
| golang-migrate | (CLI) | Database migrations | Already in use |

No new Go dependencies required. The Work service uses the exact same dependency set as CRM/Chat.

### Core (Frontend - React/TypeScript)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| @dnd-kit/core | ^6.x | Drag-and-drop foundation | Modular, accessible, hook-based, TypeScript-first, zero deps |
| @dnd-kit/sortable | ^8.x | Sortable containers for Kanban | Built for multi-container sorting (columns + cards) |
| @dnd-kit/utilities | ^3.x | CSS transform utilities | Helper for drag overlay styling |
| @tanstack/react-query | ^5.x | Server state management | Already in use |
| react-router-dom | ^7.x | Routing for work module | Already in use |
| zustand | ^5.x | Client state (view preferences, panel state) | Already in use |
| date-fns | ^4.x | Date formatting for due dates | Already in use |
| lucide-react | ^0.470 | Icons | Already in use |
| Radix UI (dialog, popover, etc.) | Various | Task detail panel, dropdowns | Already in use |

### New Dependencies to Install

```bash
# Desktop (new DnD library only)
cd desktop
npm install @dnd-kit/core @dnd-kit/sortable @dnd-kit/utilities
```

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| @dnd-kit | @hello-pangea/dnd | Simpler API but less control for nested/complex interactions; DnD-kit better for multi-container sorting needed in Kanban |
| @dnd-kit | pragmatic-drag-and-drop | Smaller bundle but headless (more work), less React-idiomatic; DnD-kit has better React integration |
| Adjacency list | Closure table | Better read performance but much more complex writes; overkill for 3-5 level nesting |
| Adjacency list | PostgreSQL ltree | Powerful path queries but requires extension, less portable for self-hosted; adjacency + recursive CTE sufficient |

## Architecture Patterns

### Recommended Backend Structure (Work Service)

```
backend/
  cmd/
    work/
      main.go                    # Service entry point (same pattern as cmd/crm)
  internal/
    work/
      project/
        errors.go
        repository.go            # Repository interface
        postgres_repository.go   # PostgreSQL implementation
        service.go               # Business logic
        service_test.go          # Unit tests (mock repo)
      task/
        errors.go
        event_emitter.go         # pg_notify for notifications
        repository.go
        postgres_repository.go
        service.go
        service_test.go
      comment/
        errors.go
        repository.go
        postgres_repository.go
        service.go
        service_test.go
      status/
        errors.go
        repository.go
        postgres_repository.go
        service.go
        service_test.go
      template/
        errors.go
        repository.go
        postgres_repository.go
        service.go
        service_test.go
  proto/
    work/
      v1/
        work.proto               # All Work service RPCs
  internal/
    gateway/
      route_work.go              # HTTP routes for Work service
    models/
      project.go                 # Project, ProjectMember, ProjectStatus models
      task.go                    # Task, TaskDependency, TaskComment, TaskLink models
    server/
      work_grpc.go               # gRPC server implementation
```

### Recommended Frontend Structure

```
desktop/src/renderer/src/
  modules/
    work/
      WorkLayout.tsx              # Module layout with sub-nav (Projects, My Tasks)
      projects/
        ProjectsListPage.tsx      # All projects the user can access
        ProjectDetailPage.tsx     # Project with Kanban/List toggle
      tasks/
        MyTasksPage.tsx           # Standalone + assigned tasks across all projects
        TaskCard.tsx              # Kanban card component (draggable)
        TaskRow.tsx               # List row component (inline editing)
        TaskDetailPanel.tsx       # Slide-over side panel
        TaskDetailPage.tsx        # Full page view (comments, attachments, activity)
        TaskForm.tsx              # Create/edit form (reused in panel and page)
      kanban/
        KanbanBoard.tsx           # DnD board with columns
        KanbanColumn.tsx          # Droppable column (status)
        KanbanCard.tsx            # Draggable card
      list/
        TaskListView.tsx          # Grouped/sorted list
        TaskListHeader.tsx        # Grouping/sorting controls
      components/
        StatusBadge.tsx           # Color-coded status badge
        PriorityBadge.tsx         # Color-coded priority indicator
        TaskLinkField.tsx         # CRM entity search-and-link
        CommentThread.tsx         # Comment list + input
        ActivityLog.tsx           # Task activity timeline
  api/
    hooks/
      useProjects.ts             # TanStack Query hooks for projects
      useTasks.ts                # TanStack Query hooks for tasks
      useTaskComments.ts         # Comments hooks
  stores/
    work.ts                      # Zustand store (view preferences per project)
```

### Pattern 1: Service Layer with Event Emission (Backend)

**What:** Task status changes and assignments emit events via pg_notify for real-time notifications.
**When to use:** Any task mutation that should trigger a notification.
**Example (follows existing deal event emitter pattern):**

```go
// internal/work/task/event_emitter.go
type EventEmitter interface {
    EmitTaskEvent(ctx context.Context, payload models.EventPayload) error
}

type PGEventEmitter struct {
    pool *pgxpool.Pool
}

func (e *PGEventEmitter) EmitTaskEvent(ctx context.Context, payload models.EventPayload) error {
    return event.EmitEvent(ctx, e.pool, payload)
}

// In service.go, after status change:
if s.eventEmitter != nil {
    _ = s.eventEmitter.EmitTaskEvent(ctx, models.EventPayload{
        Type:          event.EventWorkTaskStatusChanged,
        ModuleID:      event.ModuleWork,
        ActorID:       actorID.String(),
        ResourceID:    task.ID.String(),
        TargetUserIDs: targetUserIDs,
        Title:         "Aufgabenstatus geaendert",
        Body:          fmt.Sprintf("%s: %s -> %s", task.Title, oldStatus, newStatus),
        DeepLink:      fmt.Sprintf("/work/projects/%s/tasks/%s", task.ProjectID, task.ID),
    })
}
```

### Pattern 2: Optimistic Kanban DnD (Frontend)

**What:** Drag-and-drop updates task status optimistically on the client, then syncs with server.
**When to use:** Kanban card drag between status columns.
**Example:**

```typescript
// Using @dnd-kit with TanStack Query optimistic updates
function handleDragEnd(event: DragEndEvent) {
  const { active, over } = event
  if (!over || active.id === over.id) return

  const taskId = active.id as string
  const newStatusId = over.id as string

  // Optimistic update via TanStack Query
  queryClient.setQueryData(['tasks', projectId], (old) => {
    return old?.map(t => t.id === taskId ? { ...t, status_id: newStatusId } : t)
  })

  // Server sync
  moveTask.mutate(
    { taskId, statusId: newStatusId },
    {
      onError: () => {
        // Snap back on failure
        queryClient.invalidateQueries({ queryKey: ['tasks', projectId] })
        toast.error('Status konnte nicht geaendert werden')
      },
    }
  )
}
```

### Pattern 3: Gateway Route Registrar (Backend)

**What:** Work service HTTP routes follow the same RouteRegistrar pattern as CRM, Chat, etc.
**When to use:** All HTTP endpoints for the Work service.
**Example (follows existing route_crm.go pattern):**

```go
// internal/gateway/route_work.go
type WorkRoutes struct {
    registry *ServiceRegistry
}

func NewWorkRoutes(registry *ServiceRegistry) *WorkRoutes {
    return &WorkRoutes{registry: registry}
}

func (w *WorkRoutes) ServiceName() string { return "work" }

func (w *WorkRoutes) getWorkClient() (workv1.WorkServiceClient, error) {
    conn, err := w.registry.GetConnection("work")
    if err != nil {
        return nil, err
    }
    return workv1.NewWorkServiceClient(conn), nil
}

func (w *WorkRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
    r.Route("/api/v1/projects", func(r chi.Router) {
        r.Use(authMiddleware)
        r.With(middleware.RequirePermission("projects", "read")).Get("/", w.HandleListProjects)
        r.With(middleware.RequirePermission("projects", "write")).Post("/", w.HandleCreateProject)
        // ... etc
    })
    r.Route("/api/v1/tasks", func(r chi.Router) {
        r.Use(authMiddleware)
        r.With(middleware.RequirePermission("tasks", "read")).Get("/", w.HandleListTasks)
        // ... etc
    })
}
```

### Anti-Patterns to Avoid

- **Business logic in gateway handlers:** All task validation, authorization (project membership check), status transition logic, dependency validation must be in the Work service layer, NOT in gateway HTTP handlers. Handlers only parse/validate HTTP input and call gRPC.
- **Dual-write for task files:** Task file attachments reuse the existing MinIO file store and chat file infrastructure. Do NOT create a separate file storage system. The file is uploaded via the existing `/api/v1/files/upload` endpoint with a new context parameter (task_id instead of channel_id).
- **Global mutable state for DnD:** All drag state managed via @dnd-kit hooks (useDraggable, useDroppable, useSortable). No global Zustand store for drag operations.
- **N+1 queries for task lists:** Task list queries must JOIN with statuses, assignees, and tags in a single query. Do not fetch tasks then loop to fetch related data.
- **String-concatenation for status filtering:** Use parameterized queries, never build SQL with string interpolation even for "safe" values like status names.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Drag-and-drop | Custom mouse/touch event handlers | @dnd-kit | Accessibility (keyboard, screen readers), touch support, collision detection, drop animations -- hundreds of edge cases |
| Task ordering within status | Manual sort_order management | @dnd-kit/sortable's `arrayMove` | Handles reorder index calculation, prevents race conditions with optimistic updates |
| Recursive subtask queries | Application-level tree traversal | PostgreSQL recursive CTE | Database does it faster, atomically, and handles concurrency |
| Full-text search on tasks | Custom LIKE queries | PostgreSQL tsvector + GIN index | Already proven pattern in CRM (migration 000011); handles German stemming, ranking, prefix matching |
| File upload for task attachments | New upload endpoint | Existing `/api/v1/files/upload` with task context | MinIO integration, virus scanning hook, quota tracking, thumbnail generation all already built |
| Notification delivery | Custom WebSocket push | Existing pg_notify event bus + notification service | Event types, delivery preferences, quiet hours, grouping all already built |
| Custom field engine for tasks | New custom fields implementation | Existing `custom_field_definitions` + `*_custom_field_values` pattern | Schema, validation, CRUD already built in CRM Phase 2 |

**Key insight:** Phase 6 is primarily a new domain model (projects/tasks) on top of existing infrastructure. The chat file system, notification bus, custom fields engine, and search patterns are all reusable. The only truly new frontend capability is Kanban DnD.

## Common Pitfalls

### Pitfall 1: Sort Order Gaps and Conflicts in Kanban

**What goes wrong:** When moving cards between columns or reordering within a column, sort_order values can collide or leave gaps, causing incorrect display order or constraint violations.
**Why it happens:** Concurrent users drag cards simultaneously; naive integer sort_order increments lead to duplicates.
**How to avoid:** Use fractional ordering (e.g., midpoint between neighbors) or lexicographic ordering (e.g., "a", "ab", "b"). Alternatively, store ordered arrays of task IDs per status column. Periodically normalize sort orders to prevent precision issues.
**Warning signs:** Tasks appearing in wrong order after page refresh; duplicate sort_order values in database.

### Pitfall 2: Circular Dependencies

**What goes wrong:** Task A blocks Task B which blocks Task C which blocks Task A -- infinite loop.
**Why it happens:** No validation when creating dependency links.
**How to avoid:** Before creating a "blocks" dependency, run a cycle detection check using recursive CTE: traverse the dependency graph from the target task to see if it reaches back to the source task. Reject with a clear error message.
**Warning signs:** Stack overflow or timeout in dependency queries; tasks permanently marked as "blocked."

### Pitfall 3: Subtask Depth Explosion

**What goes wrong:** Users create 10+ levels of nesting, making UI unreadable and queries slow.
**Why it happens:** "Arbitrarily deep" nesting with no practical limit.
**How to avoid:** Enforce a configurable max depth (recommend 5 levels). Check depth on task creation/move. The UI should collapse deeply nested tasks by default and show depth indicators. The `depth` column (materialized from parent chain) makes depth checks O(1).
**Warning signs:** Recursive CTE queries taking >100ms; UI rendering thousands of nested rows.

### Pitfall 4: N+1 Queries on Task Lists with Relations

**What goes wrong:** Fetching 50 tasks, then for each task: fetch assignee, fetch status, fetch tags, fetch subtask count -- 200+ queries.
**Why it happens:** Naive repository implementation that fetches related data separately.
**How to avoid:** Use SQL JOINs and subqueries in the list query. Fetch tasks with their status name, assignee name, tag list, and subtask count in a single query (or at most 2: one for tasks, one for tags via junction table).
**Warning signs:** List API response time >500ms; database connection pool exhaustion.

### Pitfall 5: Optimistic Update Rollback UX

**What goes wrong:** User drags card to new column, server rejects (e.g., missing required fields for that status), card stays in wrong column until page refresh.
**Why it happens:** Optimistic update applied but error handling doesn't revert UI state.
**How to avoid:** TanStack Query's `onError` callback must `invalidateQueries` for the task list, which forces a refetch and reverts the optimistic state. Show a toast with the server error message. The user sees the card snap back.
**Warning signs:** Card position doesn't match server state after errors; stale data after mutations.

### Pitfall 6: Project Access Checks in Wrong Layer

**What goes wrong:** Gateway handler checks project membership, but the Work service also checks, leading to double-checking or worse, inconsistent enforcement.
**Why it happens:** Unclear ownership of authorization between gateway and service.
**How to avoid:** The Work service is the single source of truth for project membership checks. The gateway only checks system-level permissions (can this user access the "tasks" resource at all). Project-level membership is checked in the Work service because it has access to the project_members table.
**Warning signs:** Unauthorized access to project tasks; 403 errors that should be 404 (leaking project existence).

## Code Examples

### Database Migration: Core Tables (verified pattern from existing migrations)

```sql
-- Projects
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    project_key VARCHAR(10) NOT NULL,  -- e.g. 'ACME', 'HR'
    next_task_number INTEGER NOT NULL DEFAULT 1,
    is_template BOOLEAN NOT NULL DEFAULT false,
    template_source_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_projects_key ON projects (project_key) WHERE archived_at IS NULL;
CREATE INDEX idx_projects_created_by ON projects (created_by);

-- Project Members
CREATE TABLE project_members (
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'member',  -- 'owner', 'member', 'viewer'
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (project_id, user_id)
);

CREATE INDEX idx_project_members_user ON project_members (user_id);

-- Project Statuses (per-project custom statuses)
CREATE TABLE project_statuses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    color VARCHAR(7),  -- hex color
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_default BOOLEAN NOT NULL DEFAULT false,
    is_closed BOOLEAN NOT NULL DEFAULT false,  -- marks "done" statuses
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_project_statuses_project ON project_statuses (project_id, sort_order);
CREATE UNIQUE INDEX idx_project_statuses_name ON project_statuses (project_id, LOWER(name));

-- Tasks
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,  -- nullable for standalone tasks
    task_number INTEGER NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    status_id UUID REFERENCES project_statuses(id) ON DELETE SET NULL,
    priority VARCHAR(10) NOT NULL DEFAULT 'medium',  -- urgent, high, medium, low
    assignee_id UUID REFERENCES users(id) ON DELETE SET NULL,
    parent_task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    depth INTEGER NOT NULL DEFAULT 0,  -- materialized depth for fast queries
    sort_order DOUBLE PRECISION NOT NULL DEFAULT 0,  -- fractional for reordering
    due_date DATE,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_tasks_project ON tasks (project_id) WHERE project_id IS NOT NULL;
CREATE INDEX idx_tasks_assignee ON tasks (assignee_id) WHERE assignee_id IS NOT NULL;
CREATE INDEX idx_tasks_parent ON tasks (parent_task_id) WHERE parent_task_id IS NOT NULL;
CREATE INDEX idx_tasks_status ON tasks (status_id) WHERE status_id IS NOT NULL;
CREATE INDEX idx_tasks_due_date ON tasks (due_date) WHERE due_date IS NOT NULL;
CREATE INDEX idx_tasks_created_by ON tasks (created_by);
CREATE INDEX idx_tasks_project_number ON tasks (project_id, task_number);
CREATE INDEX idx_tasks_priority ON tasks (priority);
CREATE INDEX idx_tasks_created_at ON tasks (created_at DESC);

-- Task Dependencies
CREATE TABLE task_dependencies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    target_task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    dependency_type VARCHAR(20) NOT NULL,  -- 'blocks', 'blocked_by', 'relates_to', 'duplicates'
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_task_id, target_task_id, dependency_type)
);

CREATE INDEX idx_task_deps_source ON task_dependencies (source_task_id);
CREATE INDEX idx_task_deps_target ON task_dependencies (target_task_id);

-- Task Comments
CREATE TABLE task_comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id),
    content TEXT NOT NULL,
    quoted_comment_id UUID REFERENCES task_comments(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_task_comments_task ON task_comments (task_id, created_at);

-- Task Entity Links (CRM + Chat)
CREATE TABLE task_entity_links (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    entity_type VARCHAR(20) NOT NULL,  -- 'contact', 'company', 'deal', 'channel', 'message'
    entity_id UUID NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (task_id, entity_type, entity_id)
);

CREATE INDEX idx_task_entity_links_task ON task_entity_links (task_id);
CREATE INDEX idx_task_entity_links_entity ON task_entity_links (entity_type, entity_id);

-- Task Activity Log
CREATE TABLE task_activities (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    actor_id UUID NOT NULL REFERENCES users(id),
    action VARCHAR(50) NOT NULL,  -- 'created', 'status_changed', 'assigned', 'priority_changed', etc.
    field_name VARCHAR(50),
    old_value JSONB,
    new_value JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_task_activities_task ON task_activities (task_id, created_at);

-- Task File Attachments (reuses chat file infrastructure)
CREATE TABLE task_files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    file_size BIGINT NOT NULL,
    storage_key VARCHAR(500) NOT NULL,
    thumbnail_key VARCHAR(500),
    uploaded_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_task_files_task ON task_files (task_id);

-- User View Preferences (per user per project)
CREATE TABLE user_project_preferences (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    view_type VARCHAR(10) NOT NULL DEFAULT 'list',  -- 'list' or 'kanban'
    list_group_by VARCHAR(20) DEFAULT 'status',  -- 'status', 'assignee', 'priority', 'due_date', 'none'
    list_sort_by VARCHAR(20) DEFAULT 'created_at',
    list_sort_desc BOOLEAN DEFAULT false,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, project_id)
);

-- Task Custom Field Values (reuses custom field engine)
CREATE TABLE task_custom_field_values (
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    field_id UUID NOT NULL REFERENCES custom_field_definitions(id) ON DELETE CASCADE,
    value JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (task_id, field_id)
);

CREATE INDEX idx_task_custom_field_values_field ON task_custom_field_values (field_id);
```

### Recursive Subtask Query

```sql
-- Get all subtasks of a task (with depth limit)
WITH RECURSIVE subtree AS (
    SELECT id, title, parent_task_id, depth, sort_order
    FROM tasks
    WHERE parent_task_id = $1

    UNION ALL

    SELECT t.id, t.title, t.parent_task_id, t.depth, t.sort_order
    FROM tasks t
    INNER JOIN subtree s ON t.parent_task_id = s.id
    WHERE t.depth <= $2  -- max depth limit
)
SELECT * FROM subtree ORDER BY depth, sort_order;
```

### Cycle Detection for Dependencies

```sql
-- Check if adding dependency source->target would create a cycle
WITH RECURSIVE chain AS (
    SELECT target_task_id AS task_id
    FROM task_dependencies
    WHERE source_task_id = $2  -- target of proposed link
    AND dependency_type IN ('blocks', 'blocked_by')

    UNION ALL

    SELECT td.target_task_id
    FROM task_dependencies td
    INNER JOIN chain c ON td.source_task_id = c.task_id
    WHERE td.dependency_type IN ('blocks', 'blocked_by')
)
SELECT EXISTS (SELECT 1 FROM chain WHERE task_id = $1);  -- does it reach source?
```

### DnD-Kit Kanban Board Setup (Frontend)

```typescript
import {
  DndContext,
  DragOverlay,
  closestCorners,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragStartEvent,
  type DragEndEvent,
  type DragOverEvent,
} from '@dnd-kit/core'
import {
  SortableContext,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'

function KanbanBoard({ statuses, tasks }) {
  const [activeTask, setActiveTask] = useState(null)
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor)
  )

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCorners}
      onDragStart={handleDragStart}
      onDragOver={handleDragOver}
      onDragEnd={handleDragEnd}
    >
      <div className="flex gap-4 overflow-x-auto">
        {statuses.map((status) => (
          <KanbanColumn key={status.id} status={status}>
            <SortableContext
              items={tasksByStatus[status.id]}
              strategy={verticalListSortingStrategy}
            >
              {tasksByStatus[status.id].map((task) => (
                <KanbanCard key={task.id} task={task} />
              ))}
            </SortableContext>
          </KanbanColumn>
        ))}
      </div>
      <DragOverlay>
        {activeTask ? <KanbanCard task={activeTask} isDragOverlay /> : null}
      </DragOverlay>
    </DndContext>
  )
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| react-beautiful-dnd | @dnd-kit or @hello-pangea/dnd | 2023 (rbd deprecated) | react-beautiful-dnd is unmaintained; dnd-kit is the modern standard |
| Integer sort_order | Fractional/lexicographic ordering | Ongoing | Avoids reindex of all items on every reorder; critical for DnD performance |
| Server-first rendering | Optimistic updates + SWR | Standard practice | Users expect instant feedback on drag operations |
| Separate task management app | Integrated PM within CRM | Industry trend | Linear, Monday, ClickUp all integrate PM with other tools |

**Deprecated/outdated:**
- `react-beautiful-dnd`: Deprecated by Atlassian, replaced by pragmatic-drag-and-drop. Do not use.
- Global integer sort_order: Leads to write amplification when reordering; use fractional ordering instead.

## Open Questions

1. **File attachment reuse scope**
   - What we know: Chat files use MinIO with `chat_files` table, uploaded via `/api/v1/files/upload`. Task files need similar storage.
   - What's unclear: Should task files go in the same MinIO bucket or a separate one? Should the upload endpoint be shared or separate?
   - Recommendation: Same bucket, separate key prefix (`tasks/{task_id}/{file_id}`). Create a new `/api/v1/tasks/{id}/files` upload endpoint in the Work gateway routes that reuses the MinIO file store but writes to `task_files` table instead of `chat_files`. This keeps file logic clean while reusing MinIO infrastructure.

2. **Custom fields entity type extension**
   - What we know: Custom fields use `EntityType` enum (contact, company, deal, activity). Tasks need to be added.
   - What's unclear: Does adding 'task' to EntityType break existing queries or require migration changes?
   - Recommendation: Add `EntityTypeTask EntityType = "task"` to the Go models and add it to `ValidEntityTypes`. No migration needed for the definitions table (it stores entity_type as VARCHAR). The new `task_custom_field_values` junction table handles task-specific values.

3. **WebSocket events for task updates**
   - What we know: Chat and notifications both push via WebSocket. Task status changes should be real-time for Kanban boards.
   - What's unclear: Should task updates go through the existing notification event bus, or should there be direct WebSocket push (like chat messages)?
   - Recommendation: Use the existing notification event bus for task assignments and status changes (these are user-targeted notifications). For real-time board updates (another user moved a card), add a new pg_notify channel `task_updates` that the gateway listens to and broadcasts to project members via WebSocket. This keeps notification logic separate from real-time UI sync.

4. **Template task IDs**
   - What we know: Templates are saved projects. Creating from template pre-populates tasks and statuses.
   - What's unclear: How to deep-copy tasks with parent_task_id references (subtask hierarchy) and dependency links when IDs change.
   - Recommendation: Copy in topological order (parents before children), maintain a mapping of old_id -> new_id, and remap parent_task_id and dependency references using the mapping. This is a single-transaction operation in the service layer.

## Sources

### Primary (HIGH confidence)
- Codebase analysis: All patterns verified by reading actual source files
  - `backend/internal/crm/*` -- service/repository/error pattern
  - `backend/internal/chat/*` -- file upload, search, event emission patterns
  - `backend/internal/notification/*` -- event bus, delivery, preference patterns
  - `backend/internal/gateway/*` -- route registrar, service registry patterns
  - `backend/proto/crm/v1/crm.proto` -- gRPC service definition pattern
  - `backend/migrations/000001-000023` -- migration patterns, index naming, permission seeding
  - `backend/cmd/crm/main.go` -- service entry point pattern
  - `backend/internal/config/config.go` -- environment variable configuration
  - `desktop/src/renderer/src/` -- React app structure, routing, API hooks, Zustand stores
  - `desktop/package.json` -- current dependency versions

### Secondary (MEDIUM confidence)
- @dnd-kit documentation (dndkit.com) -- API, sortable preset, collision detection
- PostgreSQL documentation (postgresql.org/docs) -- recursive CTEs, tsvector, ltree
- Marmelab blog (2026-01-15) -- Kanban board with shadcn/ui patterns
- Multiple React DnD comparison articles (2025-2026) -- library selection rationale

### Tertiary (LOW confidence)
- WebSearch results for project management database schemas -- general patterns, not PostgreSQL-specific
- Hacker News discussions on dnd-kit vs pragmatic-drag-and-drop -- community opinions, not benchmarks

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all backend libraries already in use; only new frontend dependency is @dnd-kit
- Architecture: HIGH -- follows exact same patterns as existing CRM/Chat services (verified by reading source)
- Data model: HIGH -- adjacency list is well-understood; PostgreSQL recursive CTEs verified via official docs
- Frontend DnD: MEDIUM -- @dnd-kit is the best-supported option but requires careful implementation for multi-container sorting
- Pitfalls: HIGH -- drawn from experience with similar systems and verified anti-patterns

**Research date:** 2026-02-08
**Valid until:** 2026-03-08 (stable domain; @dnd-kit API unlikely to change significantly)
