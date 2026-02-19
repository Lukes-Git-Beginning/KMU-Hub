---
phase: 06-project-management
plan: 09
subsystem: desktop-work-module
tags: [gantt-chart, timeline, critical-path, dependency-arrows, svg, date-fns, zoom]

dependency-graph:
  requires: [06-05, 06-06, 06-07, 06-08]
  provides: [gantt-chart-view, critical-path-visualization, dependency-arrow-rendering, timeline-zoom]
  affects: [06-10, phase-07]

tech-stack:
  added: []
  patterns: [split-panel-scroll-sync, svg-overlay-arrows, forward-backward-pass-cpm, date-to-pixel-mapping]

key-files:
  created:
    - desktop/src/renderer/src/modules/work/gantt/gantt-utils.ts
    - desktop/src/renderer/src/modules/work/gantt/GanttChart.tsx
    - desktop/src/renderer/src/modules/work/gantt/GanttTimeline.tsx
    - desktop/src/renderer/src/modules/work/gantt/GanttTaskRow.tsx
    - desktop/src/renderer/src/modules/work/gantt/GanttDependencyArrows.tsx
    - backend/migrations/000031_add_gantt_view_type.up.sql
    - backend/migrations/000031_add_gantt_view_type.down.sql
  modified:
    - desktop/src/renderer/src/modules/work/projects/ProjectDetailPage.tsx
    - desktop/src/renderer/src/api/hooks/useProjects.ts

key-decisions:
  - "Migration renumbered to 000031 to avoid collision with uncommitted 000030 time_entries migration"
  - "Batch dependency fetching via useQueries for tasks with has_blocked_deps flag"
  - "Unscheduled tasks (no due_date) shown in separate section below timeline, not on bars"
  - "Critical path uses forward/backward pass with topological sort via Kahn's algorithm"
  - "Gantt view is read-only: bars are not draggable, only clickable to open detail panel"

patterns-established:
  - "Split-panel layout: fixed-width left (task info) + scrollable right (timeline)"
  - "SVG overlay pattern for dependency arrows with pointer-events:none"
  - "Pure utility module (gantt-utils.ts) separating date math from React components"

duration: ~8min
completed: 2026-02-08
---

# Phase 6 Plan 9: Gantt Chart View Summary

**Read-only Gantt chart with critical path highlighting, SVG dependency arrows, day/week/month zoom, and timeline scroll as third project view option (PM-16)**

## Performance

- **Duration:** ~8 min
- **Started:** 2026-02-08T22:15:13Z
- **Completed:** 2026-02-08
- **Tasks:** 2
- **Files modified:** 9

## Accomplishments
- Pure utility module (gantt-utils.ts) with timeline config, date-to-pixel mapping, column generation, and CPM critical path algorithm
- Four Gantt components: GanttChart (container), GanttTimeline (header), GanttTaskRow (bar rendering), GanttDependencyArrows (SVG overlay)
- Three-way view toggle (List/Kanban/Gantt) in project detail page with GanttChartSquare icon
- Dependency arrows drawn as SVG bezier curves with arrowhead markers between blocking tasks
- Critical path tasks highlighted with red ring, critical arrows in solid red
- Day/week/month zoom with "Heute" button to scroll to current date
- Unscheduled tasks shown in "Ungeplant" section below the timeline
- Backend migration 000031 adds 'gantt' to user_project_preferences view_type CHECK constraint

## Task Commits

Each task was committed atomically:

1. **Task 1: Gantt utility functions and critical path algorithm** - `807b535` (feat)
2. **Task 2: Gantt chart components and project detail integration** - `0aab42a` (feat)

## Files Created/Modified
- `desktop/src/renderer/src/modules/work/gantt/gantt-utils.ts` - Pure date math, positioning, critical path, API mapping
- `desktop/src/renderer/src/modules/work/gantt/GanttChart.tsx` - Main container with data fetching, zoom, scroll, layout
- `desktop/src/renderer/src/modules/work/gantt/GanttTimeline.tsx` - Sticky dual-row timeline header
- `desktop/src/renderer/src/modules/work/gantt/GanttTaskRow.tsx` - Task info + colored bar with tooltip
- `desktop/src/renderer/src/modules/work/gantt/GanttDependencyArrows.tsx` - SVG bezier arrows between tasks
- `desktop/src/renderer/src/modules/work/projects/ProjectDetailPage.tsx` - Added Gantt as third view option
- `desktop/src/renderer/src/api/hooks/useProjects.ts` - view_type expanded to include 'gantt'
- `backend/migrations/000031_add_gantt_view_type.up.sql` - Extend CHECK constraint
- `backend/migrations/000031_add_gantt_view_type.down.sql` - Revert CHECK constraint

## Decisions Made
- Migration numbered 000031 (not 000030) to avoid collision with uncommitted time_entries migration from plan 06-10 prep
- Dependencies fetched via useQueries batch for each task with has_blocked_deps flag (no project-level dependency endpoint exists)
- Gantt view is read-only in v1: clicking bars opens detail panel but bars are not draggable
- Critical path algorithm uses CPM forward/backward pass with Kahn's topological sort; handles cycles gracefully (returns empty set)
- Bar colors prefer status_color, fall back to priority-based colors (urgent=red, high=orange, medium=blue, low=gray)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Migration number collision with uncommitted time_entries migration**
- **Found during:** Task 2 (migration creation)
- **Issue:** Uncommitted 000030_create_time_entries migration files from plan 06-10 prep already existed on disk
- **Fix:** Renumbered gantt migration to 000031 to avoid golang-migrate collision
- **Files modified:** backend/migrations/000031_add_gantt_view_type.{up,down}.sql
- **Committed in:** 0aab42a

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Migration renumbering necessary to avoid tooling conflict. No scope creep.

## Issues Encountered
- Pre-existing Go build error in backend/cmd/work/main.go (missing timeentry.Service parameter) -- this is from uncommitted plan 06-10 preparation code, not related to Gantt changes. Frontend build and type-check both succeed cleanly.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Plan 06-10 (Task Timer) is the final plan in Phase 6
- All PM-01 through PM-09 + PM-16 requirements now satisfied
- Remaining: PM-17 (Task timer) to complete Phase 6

## Self-Check: PASSED

---
*Phase: 06-project-management*
*Completed: 2026-02-08*
