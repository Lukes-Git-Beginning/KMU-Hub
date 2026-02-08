---
phase: 06-project-management
plan: 06
subsystem: desktop-frontend
tags: [react, dnd-kit, kanban, task-list, inline-editing, grouping, sorting]
requires:
  - 06-05 (Work module frontend foundation, API hooks, Zustand store)
provides:
  - Task list view with 5-dimension grouping and inline editing
  - Kanban board with DnD drag-and-drop and optimistic status updates
  - Subtask nesting in both list and Kanban views
  - Task creation dialog
  - Shared StatusBadge and PriorityBadge components
affects:
  - 06-07 (task detail slide-over panel uses same TaskData types and work store)
  - 06-08 (advanced features may extend list/kanban views)
tech-stack:
  added: ["@dnd-kit/core", "@dnd-kit/sortable", "@dnd-kit/utilities", "@radix-ui/react-select"]
  patterns: [optimistic-updates, fractional-ordering, client-side-grouping, inline-editing-popovers]
key-files:
  created:
    - desktop/src/renderer/src/modules/work/components/StatusBadge.tsx
    - desktop/src/renderer/src/modules/work/components/PriorityBadge.tsx
    - desktop/src/renderer/src/modules/work/components/TaskCreateDialog.tsx
    - desktop/src/renderer/src/modules/work/list/TaskListView.tsx
    - desktop/src/renderer/src/modules/work/list/TaskListHeader.tsx
    - desktop/src/renderer/src/modules/work/list/TaskRow.tsx
    - desktop/src/renderer/src/modules/work/kanban/KanbanBoard.tsx
    - desktop/src/renderer/src/modules/work/kanban/KanbanColumn.tsx
    - desktop/src/renderer/src/modules/work/kanban/KanbanCard.tsx
    - desktop/src/renderer/src/modules/work/kanban/KanbanSubtaskGroup.tsx
  modified:
    - desktop/package.json
    - desktop/package-lock.json
    - desktop/src/renderer/src/modules/work/projects/ProjectDetailPage.tsx
key-decisions:
  - Client-side grouping instead of server-side for instant group switching
  - closestCorners collision detection for multi-container Kanban DnD
  - Max 3 visual nesting levels on Kanban to keep board clean
  - Subtask cards not independently draggable on Kanban (parent moves all)
duration: ~7min
completed: 2026-02-08
---

# Phase 6 Plan 6: Task List and Kanban Views Summary

Sortable/filterable task list with 5-dimension grouping plus DnD Kanban board using @dnd-kit with optimistic status updates and subtask nesting in both views.

## Performance

| Metric | Value |
|--------|-------|
| Duration | ~7 minutes |
| Started | 2026-02-08T17:13:18Z |
| Completed | 2026-02-08T17:20:02Z |
| Tasks | 2/2 |
| Files created | 10 |
| Files modified | 3 |

## Accomplishments

### Task 1: Shared Components and Dependencies
- Installed @dnd-kit/core, @dnd-kit/sortable, @dnd-kit/utilities
- Created StatusBadge component rendering colored pill with status color/name and optional checkmark for closed status
- Created PriorityBadge component with icon+label (full mode) and icon-only (compact mode) for urgent/high/normal/low
- Created TaskCreateDialog with Radix Dialog: title (required), status (from project statuses), priority (select), assignee (from project members), due date (date picker), collapsible description field, optional parent task for subtask creation

### Task 2: Task List View, Kanban Board, and ProjectDetailPage Integration
- **TaskListHeader**: Controls bar with group-by selector (Status/Assignee/Priority/Due Date/None), sort-by selector with asc/desc toggle, filter popover with priority pills and status dropdown
- **TaskRow**: Inline-editable row with indentation for subtask depth, expand/collapse toggle, task key display, double-click title editing, status/priority/assignee/due-date inline editing via Popovers, blocked indicator, subtask count
- **TaskListView**: Main list component using useTasks hook, client-side grouping into collapsible groups, recursive subtask tree rendering, preference persistence via useSetPreference
- **KanbanBoard**: DndContext with PointerSensor (5px activation distance) and KeyboardSensor, closestCorners collision detection, optimistic cache updates via queryClient.setQueryData, rollback on error via query invalidation, DragOverlay with elevated card clone
- **KanbanColumn**: useDroppable zone per status, SortableContext with verticalListSortingStrategy, hierarchical task display (top-level cards + subtask groups), empty column placeholder
- **KanbanCard**: useSortable for drag-and-drop, shows task key, title, priority (compact), assignee, due date (with overdue highlighting), blocked indicator, subtask progress bar, overlay variant styling
- **KanbanSubtaskGroup**: Collapsible indented subtask display beneath parent cards, max 3 depth levels with "N more nested" fallback, thin left-border connector, reduced padding/size for subtask cards, recursive nesting support
- **ProjectDetailPage**: Updated from placeholder to render TaskListView or KanbanBoard based on view_type preference, view toggle persists via useSetPreference, "Neue Aufgabe" button opens TaskCreateDialog

## Task Commits

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | Shared components and @dnd-kit | bc3ee51 | StatusBadge.tsx, PriorityBadge.tsx, TaskCreateDialog.tsx, package.json |
| 2 | Task list view, Kanban board, view toggle | 77b4b30 | TaskListView.tsx, KanbanBoard.tsx, KanbanCard.tsx, ProjectDetailPage.tsx |

## Files Created

- `desktop/src/renderer/src/modules/work/components/StatusBadge.tsx` -- Colored status badge with closed indicator
- `desktop/src/renderer/src/modules/work/components/PriorityBadge.tsx` -- Priority badge with icon, compact mode
- `desktop/src/renderer/src/modules/work/components/TaskCreateDialog.tsx` -- Task creation dialog
- `desktop/src/renderer/src/modules/work/list/TaskListView.tsx` -- Main list view with grouping
- `desktop/src/renderer/src/modules/work/list/TaskListHeader.tsx` -- List controls bar
- `desktop/src/renderer/src/modules/work/list/TaskRow.tsx` -- Single task row with inline editing
- `desktop/src/renderer/src/modules/work/kanban/KanbanBoard.tsx` -- DnD Kanban board
- `desktop/src/renderer/src/modules/work/kanban/KanbanColumn.tsx` -- Droppable status column
- `desktop/src/renderer/src/modules/work/kanban/KanbanCard.tsx` -- Draggable task card
- `desktop/src/renderer/src/modules/work/kanban/KanbanSubtaskGroup.tsx` -- Collapsible subtask nesting

## Files Modified

- `desktop/package.json` -- Added @dnd-kit/core, @dnd-kit/sortable, @dnd-kit/utilities, @radix-ui/react-select
- `desktop/package-lock.json` -- Lock file updated
- `desktop/src/renderer/src/modules/work/projects/ProjectDetailPage.tsx` -- Replaced placeholder with real views

## Decisions Made

1. **Client-side grouping**: Group tasks by dimension on the client rather than making separate server queries per group. Enables instant group switching without network roundtrip. Works well for typical project sizes (< 500 tasks).
2. **closestCorners collision detection**: Chosen over closestCenter for better multi-container DnD behavior when dragging cards between Kanban columns.
3. **Max 3 Kanban nesting depth**: Deeply nested subtasks beyond 3 levels show "N more nested" summary link to prevent visual clutter on the board.
4. **Subtask cards not independently draggable on Kanban**: Dragging a parent card doesn't cascade-move subtasks (they already share the same column). Subtasks can be individually moved via list view or detail panel.
5. **Fractional sort_order**: New cards placed at end of target column (maxOrder + 1.0) for simple ordering. Future refinement can use midpoint insertion.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Missing @radix-ui/react-select runtime dependency**

- **Found during:** Task 2 build verification
- **Issue:** The select.tsx UI component imports @radix-ui/react-select but it was not listed in package.json dependencies. The package existed in node_modules (likely hoisted from another dependency) but Rollup's strict resolution rejected it at build time.
- **Fix:** Ran `npm install @radix-ui/react-select` to add it as an explicit dependency.
- **Files modified:** desktop/package.json, desktop/package-lock.json
- **Commit:** 77b4b30

## Issues Encountered

None beyond the deviation documented above.

## Next Phase Readiness

Plan 06-07 (task detail slide-over panel) can proceed immediately:
- TaskData type is exported from TaskRow.tsx for reuse
- useWorkStore.openTaskPanel/closeTaskPanel already integrated in TaskRow and KanbanCard click handlers
- The task detail panel is expected to be a slide-over that opens when clicking a task title
- All inline editing patterns (status, priority, assignee, due date) are established and can be reused in the detail panel

## Self-Check: PASSED
