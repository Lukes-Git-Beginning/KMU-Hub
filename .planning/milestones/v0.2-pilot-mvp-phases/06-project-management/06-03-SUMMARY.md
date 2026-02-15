---
phase: 06-project-management
plan: 03
subsystem: api
tags: [go, task, comment, service, repository, event-emitter, pg-notify, nesting, dependencies, cycle-detection]

requires:
  - phase: 06-01
    provides: Go models (Task, TaskComment, TaskDependency, TaskActivity, TaskFile, TaskEntityLink) and migration tables
  - phase: 06-02
    provides: Project and Status services with repository interfaces for membership checks and status lookups

provides:
  - Task service with CRUD, nesting (max depth 5), dependencies (cycle detection), Kanban moves (fractional ordering), activity logging, event emission, entity linking, file attachments, custom fields, search, and template copying
  - Comment service with flat comments, quote-reply, @mention parsing, notification events, and activity logging
  - Work event types (work.task.*) and ModuleWork constant in notification event system
  - Task repository interface and PostgreSQL implementation with recursive CTEs and dynamic filtering

affects: [06-04 (gRPC server), 06-05 (gateway routes), 06-06 (frontend task views)]

tech-stack:
  added: []
  patterns:
    - "Task event emitter pattern (optional EmitTaskEvent interface, PGEventEmitter via pg_notify)"
    - "Cross-service dependency: comment service depends on task repository for activity logging"
    - "Recursive CTE for subtask queries and cycle detection in dependencies"
    - "Fractional sort_order (float64) for Kanban card positioning"

key-files:
  created:
    - backend/internal/work/task/errors.go
    - backend/internal/work/task/repository.go
    - backend/internal/work/task/postgres_repository.go
    - backend/internal/work/task/event_emitter.go
    - backend/internal/work/task/service.go
    - backend/internal/work/task/service_test.go
    - backend/internal/work/comment/errors.go
    - backend/internal/work/comment/repository.go
    - backend/internal/work/comment/postgres_repository.go
    - backend/internal/work/comment/service.go
    - backend/internal/work/comment/service_test.go
  modified:
    - backend/internal/notification/event/types.go

key-decisions:
  - "Standalone tasks get task_number=0 (no project counter increment)"
  - "Comment service depends on taskRepo.CreateActivity for activity logging (not just event emission)"
  - "@mention pattern uses @{uuid} format for deterministic user resolution"
  - "Cycle detection only for blocking dependencies (blocks/blocked_by), not relates_to/duplicates"
  - "MoveTask handles completed_at setting/clearing based on status is_closed flag"

patterns-established:
  - "Cross-package dependency: comment.Service takes task.Repository for activity logging"
  - "Task event emitter shared between task and comment services via task.EventEmitter interface"
  - "Activity logging via logFieldChange helper with JSON-encoded old/new values"

duration: 10min
completed: 2026-02-08
---

# Phase 6 Plan 3: Task & Comment Services Summary

**Task service with CRUD, nesting (max depth 5), cycle-detecting dependencies, Kanban moves, activity logging, and pg_notify events; Comment service with quote-reply and @mention parsing**

## Performance

- **Duration:** 10 min
- **Started:** 2026-02-08T13:21:37Z
- **Completed:** 2026-02-08T13:31:34Z
- **Tasks:** 2
- **Files modified:** 12

## Accomplishments
- Full task lifecycle: create, update, move (Kanban), delete with project membership enforcement
- Multi-level nesting with depth enforcement at 5 levels and parent-project validation
- Dependency management with cycle detection via recursive CTE, self-dependency and duplicate checks
- Activity logging for all key field changes (title, status, priority, assignee, due_date, description)
- Notification events for task creation, assignment, status change, comment, and @mention
- Template task copying with topological ordering, parent/dependency ID remapping
- Flat comments with quote-reply validation and @mention parsing
- 36 unit tests across task (14) and comment (22) packages, all passing

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement task package** - `cfbd29e` (feat)
2. **Task 2: Implement comment package** - `903d728` (feat)

## Files Created/Modified
- `backend/internal/work/task/errors.go` - Task-specific error types (14 errors)
- `backend/internal/work/task/repository.go` - Repository interface with 25+ methods, filter structs
- `backend/internal/work/task/postgres_repository.go` - PostgreSQL implementation with recursive CTEs, dynamic query building
- `backend/internal/work/task/event_emitter.go` - pg_notify event emission following deal pattern
- `backend/internal/work/task/service.go` - Task business logic (CRUD, nesting, deps, move, template copy)
- `backend/internal/work/task/service_test.go` - 14 tests covering all business rules
- `backend/internal/work/comment/errors.go` - Comment-specific error types (7 errors)
- `backend/internal/work/comment/repository.go` - Repository interface with TaskCommentWithAuthor type
- `backend/internal/work/comment/postgres_repository.go` - PostgreSQL implementation with author JOIN and quoted preview
- `backend/internal/work/comment/service.go` - Comment business logic (CRUD, quote-reply, mentions)
- `backend/internal/work/comment/service_test.go` - 22 tests covering CRUD, quoting, mentions, permissions
- `backend/internal/notification/event/types.go` - Added ModuleWork and 5 work.task.* event types

## Decisions Made
- Standalone tasks (no project_id) receive task_number=0 -- they have no project counter to increment
- Comment service takes task.Repository as a dependency for activity logging via CreateActivity, not just for task existence checks
- @mention format is `@{uuid}` (e.g., `@{550e8400-e29b-41d4-a716-446655440000}`) for deterministic parsing; front-end resolves display names
- Cycle detection is only enforced for blocking dependencies (blocks/blocked_by); relates_to and duplicates are allowed freely since they don't create execution dependencies
- MoveTask handles completed_at lifecycle: sets when moving to closed status, clears when moving back to open

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed StatusName reference on wrong type**
- **Found during:** Task 1 (service.go Update method)
- **Issue:** Code referenced `task.StatusName` but `task` was `*models.Task` which does not have StatusName (it is on `TaskWithRelations`)
- **Fix:** Changed to `existing.StatusName` which is the `TaskWithRelations` variable
- **Files modified:** backend/internal/work/task/service.go
- **Verification:** `go vet` passes
- **Committed in:** cfbd29e (Task 1 commit)

**2. [Rule 1 - Bug] Fixed List return value assignment in comment service**
- **Found during:** Task 2 (service.go Create method)
- **Issue:** `s.repo.List()` returns 3 values but only 2 were captured
- **Fix:** Changed to `result, _, listErr := s.repo.List(...)`
- **Files modified:** backend/internal/work/comment/service.go
- **Verification:** `go vet` passes
- **Committed in:** 903d728 (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (2 bugs)
**Impact on plan:** Both were compile-time bugs caught by go vet. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Task and Comment services are ready for gRPC server integration (06-04)
- Repository interfaces defined; PostgreSQL implementations ready for real database testing
- Event types registered for notification service integration
- Template copying logic ready for project.CreateFromTemplate orchestration

## Self-Check: PASSED

---
*Phase: 06-project-management*
*Completed: 2026-02-08*
