---
phase: 06-project-management
plan: 10
subsystem: api, ui
tags: [time-tracking, timer, protobuf, grpc, tanstack-query, zustand, postgresql]

# Dependency graph
requires:
  - phase: 06-08
    provides: Task CRUD, entity links, search, custom fields, file attachments
provides:
  - Time tracking backend (migration, models, repository, service, gRPC, gateway)
  - Timer start/stop with auto-stop previous
  - Manual time entry CRUD
  - Time entry list per task with user attribution
  - Task time summary aggregation
  - Frontend timer component with real-time elapsed display
  - Timer state persistence across navigation (Zustand store)
affects: [07-real-time, 08-video-voice, 12-invoicing, 13-hr]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Separate timeentry package under internal/work/ (consistent with project/status/task/comment)"
    - "Auto-stop previous timer pattern: StartTimer stops existing timer before creating new one"
    - "requestAnimationFrame for smooth elapsed counter in TaskTimer component"
    - "Zustand-persisted timer state for cross-navigation persistence"

key-files:
  created:
    - backend/migrations/000030_create_time_entries.up.sql
    - backend/migrations/000030_create_time_entries.down.sql
    - backend/internal/models/time_entry.go
    - backend/internal/work/timeentry/errors.go
    - backend/internal/work/timeentry/repository.go
    - backend/internal/work/timeentry/postgres_repository.go
    - backend/internal/work/timeentry/service.go
    - desktop/src/renderer/src/api/hooks/useTimeEntries.ts
    - desktop/src/renderer/src/modules/work/components/TaskTimer.tsx
    - desktop/src/renderer/src/modules/work/components/TimeEntryList.tsx
    - desktop/src/renderer/src/modules/work/components/ManualTimeEntryDialog.tsx
  modified:
    - backend/proto/work/v1/work.proto
    - backend/proto/work/v1/work.pb.go
    - backend/proto/work/v1/work_grpc.pb.go
    - backend/internal/server/work_grpc.go
    - backend/internal/gateway/route_work.go
    - backend/internal/models/task.go
    - backend/cmd/work/main.go
    - backend/api/openapi.yaml
    - desktop/src/renderer/src/api/types.ts
    - desktop/src/renderer/src/stores/work.ts
    - desktop/src/renderer/src/modules/work/tasks/TaskDetailPage.tsx

key-decisions:
  - "Migration 000030 (not 000031): 06-09 Gantt plan did not require a migration, so next available was 000030"
  - "Separate timeentry package instead of extending task package: cleaner separation of concerns"
  - "Auto-stop previous timer in service layer, not frontend: ensures single active timer invariant at DB level"
  - "Partial index idx_time_entries_active for O(1) active timer lookup per user"
  - "requestAnimationFrame for timer display: smoother than setInterval, auto-pauses in background tabs"

patterns-established:
  - "Time entry package pattern: errors.go + repository.go + postgres_repository.go + service.go"
  - "Timer auto-stop: StartTimer returns both new entry and stopped entry for UI feedback"
  - "Active timer polling at 60s intervals via TanStack Query refetchInterval"

# Metrics
duration: 10min
completed: 2026-02-08
---

# Phase 6 Plan 10: Task Timer Summary

**Full time tracking vertical slice: start/stop timer with auto-stop, manual entry, time entry list, task time summary, HH:MM:SS elapsed counter persisting across navigation**

## Performance

- **Duration:** 10 min
- **Started:** 2026-02-08T22:16:48Z
- **Completed:** 2026-02-08T22:26:54Z
- **Tasks:** 2
- **Files modified:** 22

## Accomplishments
- Complete backend time tracking: PostgreSQL migration, Go models, repository, service, 8 gRPC RPCs, 8 gateway HTTP routes, OpenAPI spec
- Frontend timer with real-time HH:MM:SS elapsed counter using requestAnimationFrame, start/stop toggle, auto-stop previous timer hint
- Manual time entry dialog with date, start time, hours/minutes duration, and description
- Time entry list with user attribution, edit/delete for owned entries, running timer indicator
- Timer state persistence in Zustand store across page navigation
- Task time summary showing total duration and entry count

## Task Commits

Each task was committed atomically:

1. **Task 1: Backend -- migration, models, proto, repository, service, gRPC, and gateway routes** - `61b44e0` (feat)
2. **Task 2: Frontend -- timer component, time entry list, manual entry dialog, and integration** - `3b437e7` (feat)

## Files Created/Modified
- `backend/migrations/000030_create_time_entries.up.sql` - time_entries table with indexes including partial index for active timers
- `backend/migrations/000030_create_time_entries.down.sql` - rollback migration
- `backend/internal/models/time_entry.go` - TimeEntry, TimeEntryWithUser, TimeEntrySummary, ActiveTimer structs
- `backend/internal/models/task.go` - Added TaskActionTimerStarted/Stopped/TimeEntryAdded constants
- `backend/internal/work/timeentry/errors.go` - Domain error definitions
- `backend/internal/work/timeentry/repository.go` - Repository interface with CRUD, list, timer, summary methods
- `backend/internal/work/timeentry/postgres_repository.go` - PostgreSQL implementation with 10 query methods
- `backend/internal/work/timeentry/service.go` - Business logic: StartTimer (auto-stop), StopTimer, AddManualEntry, UpdateEntry, DeleteEntry, ListByTask, GetTaskTimeSummary
- `backend/proto/work/v1/work.proto` - 8 new RPCs + TimeEntryProto, ActiveTimerProto, TimeSummaryProto messages
- `backend/internal/server/work_grpc.go` - 8 gRPC handler methods + proto converters + error mapping
- `backend/internal/gateway/route_work.go` - 8 HTTP routes: timer start/stop/active, time entry CRUD, time summary
- `backend/cmd/work/main.go` - Timeentry repo + service initialization
- `backend/api/openapi.yaml` - 8 new endpoints with schemas
- `desktop/src/renderer/src/api/hooks/useTimeEntries.ts` - 8 TanStack Query hooks
- `desktop/src/renderer/src/stores/work.ts` - Timer state: activeTimerTaskId, activeTimerStartedAt, activeTimerEntryId
- `desktop/src/renderer/src/modules/work/components/TaskTimer.tsx` - Start/stop button, elapsed counter, summary
- `desktop/src/renderer/src/modules/work/components/TimeEntryList.tsx` - Entry list with edit/delete
- `desktop/src/renderer/src/modules/work/components/ManualTimeEntryDialog.tsx` - Manual entry form dialog
- `desktop/src/renderer/src/modules/work/tasks/TaskDetailPage.tsx` - Integrated timer + entry list in sidebar

## Decisions Made
- Used migration number 000030 (06-09 Gantt did not need a migration)
- Created separate `timeentry` package rather than adding to task package for clean separation
- Auto-stop previous timer in service layer ensures single-timer invariant at database level
- Used partial index `WHERE ended_at IS NULL` for efficient active timer lookup
- requestAnimationFrame for elapsed counter -- smoother than setInterval and auto-pauses in background

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 6 (Project Management) is now complete with all 10 plans executed
- Time tracking data available for future reporting/invoicing modules
- Timer API ready for integration with Phase 8 (Video/Voice meetings)

## Self-Check: PASSED

---
*Phase: 06-project-management*
*Completed: 2026-02-08*
