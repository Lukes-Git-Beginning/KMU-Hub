---
phase: 06-project-management
plan: 07
subsystem: desktop-work-module
tags: [task-detail, comments, file-attachments, activity-log, dependencies, slide-over-panel]

dependency-graph:
  requires: [06-05, 06-06]
  provides: [task-detail-panel, task-detail-page, comment-thread, activity-log, file-attachments, dependency-management]
  affects: [06-08]

tech-stack:
  added: []
  patterns: [slide-over-panel, quote-reply-comments, @mention-dropdown, multipart-file-upload, two-column-detail-layout, nested-routing]

key-files:
  created:
    - desktop/src/renderer/src/api/hooks/useTaskComments.ts
    - desktop/src/renderer/src/api/hooks/useTaskActivities.ts
    - desktop/src/renderer/src/api/hooks/useTaskFiles.ts
    - desktop/src/renderer/src/modules/work/tasks/TaskDetailPanel.tsx
    - desktop/src/renderer/src/modules/work/tasks/TaskDetailPage.tsx
    - desktop/src/renderer/src/modules/work/components/CommentThread.tsx
    - desktop/src/renderer/src/modules/work/components/ActivityLog.tsx
    - desktop/src/renderer/src/modules/work/components/TaskFileAttachments.tsx
    - desktop/src/renderer/src/modules/work/components/DependencyList.tsx
  modified:
    - desktop/src/renderer/src/api/hooks/useTasks.ts
    - desktop/src/renderer/src/modules/work/projects/ProjectDetailPage.tsx

decisions:
  - id: "06-07-01"
    decision: "Fixed overlay panel (not Radix Sheet) for task detail slide-over"
    reason: "Simple CSS transform approach, no additional dependency; matches the locked decision for slide-over pattern"
  - id: "06-07-02"
    decision: "Two-step file upload: multipart to /api/v1/files/upload then metadata POST to task files"
    reason: "Reuses existing MinIO upload infrastructure from chat module; task file endpoint accepts JSON metadata only"
  - id: "06-07-03"
    decision: "Nested Routes inside ProjectDetailPage for task detail page routing"
    reason: "WorkLayout already uses wildcard path 'projects/:id/*', so nested Routes in ProjectDetailPage handles sub-routing cleanly"
  - id: "06-07-04"
    decision: "Combined activity+comments view with tab switching (Alle/Kommentare/Aktivitaet)"
    reason: "Users can see interleaved timeline or filter to comments/activity only; matches plan's 'interleaved chronologically' requirement"

metrics:
  duration: "~10min"
  completed: "2026-02-08"
---

# Phase 6 Plan 7: Task Detail and Collaboration Summary

Task detail experience with slide-over panel, full detail page, comment thread with @mentions and quote-reply, file attachments via MinIO, chronological activity log, and dependency management.

## Task Commits

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | API hooks for comments, activities, files | fc88987 | useTaskComments.ts, useTaskActivities.ts, useTaskFiles.ts |
| 2 | Task detail panel, full page, comments, activity, files, dependencies | cec94be | TaskDetailPanel.tsx, TaskDetailPage.tsx, CommentThread.tsx, ActivityLog.tsx, TaskFileAttachments.tsx, DependencyList.tsx |

## What Was Built

### API Hooks (Task 1)
- **useTaskComments**: list, create (with quoted_comment_id), update, delete mutations with invalidation chains to both task-comments and task-activities
- **useTaskActivities**: paginated activity log query for chronological task history
- **useTaskFiles**: list files, attach (multipart upload to MinIO + metadata POST), remove file mutations
- **getFileDownloadUrl**: helper for presigned download URLs via /api/v1/files/{id}/download
- **formatFileSize**: human-readable file size utility
- **useTaskDependencies, useCreateDependency, useDeleteDependency**: added to useTasks.ts

### Task Detail Panel (Slide-over)
- 400px fixed panel sliding in from right with semi-transparent backdrop
- Controlled by workStore (activeTaskId, taskPanelOpen)
- Shows: task key badge, editable title, status/priority/assignee/due date (all inline-editable via popovers), editable description, subtask list, quick comment input
- "Erweitern" button navigates to full task detail page
- Closes on X, Escape key, or backdrop click

### Task Detail Page (Full Page)
- Route: /work/projects/:id/tasks/:taskId (nested inside ProjectDetailPage)
- Two-column layout: main content (left) + sidebar (right)
- Left: breadcrumb, large editable title, editable description, subtask section with add button, tabbed activity+comments feed (Alle/Kommentare/Aktivitaet)
- Right: status, priority, assignee, due date, dependencies, entity links placeholder (06-08), custom fields placeholder (06-08), file attachments, created-by metadata

### Comment Thread
- Flat comment list with avatar, author name, relative timestamps (German)
- Quote-reply: clicking Reply sets quoted preview above input, sent as quoted_comment_id
- @mentions: typing "@" opens dropdown of project members (filtered as user types), keyboard navigation (arrow keys + Enter/Tab)
- Inline edit and delete for own comments
- @mention text rendered with highlighted background

### Activity Log
- Chronological feed with action-type icons and colors
- German descriptions for: status_changed, assigned, priority_changed, created, commented, attachment_added, linked, dependency_added
- Status changes visually prominent with background highlight

### File Attachments
- File list with mime-type icons, filename, size, uploader, relative timestamp
- Upload via "Datei anhaengen" button (file picker, supports multiple)
- Two-step upload: multipart to MinIO, then metadata attachment to task
- Download via presigned URL (opens in new tab)
- Delete for own uploads

### Dependency List
- Grouped by type: Blockiert, Blockiert durch, Verwandt, Duplikat von
- Each entry clickable (navigates to linked task)
- "Abhaengigkeit hinzufuegen" popover with type selector and task search
- Filters out current task and already-linked tasks from search results

### ProjectDetailPage Updates
- Added nested Routes: index -> ProjectBoardView, tasks/:taskId -> TaskDetailPage
- TaskDetailPanel rendered as overlay within the board view
- Imported Routes and Route from react-router-dom

## Deviations from Plan

None - plan executed exactly as written.

## Decisions Made

1. **Fixed overlay panel vs Radix Sheet**: Used CSS transform approach with fixed positioning rather than adding @radix-ui/react-dialog Sheet component. Simpler, no new dependency.
2. **Two-step file upload flow**: MinIO upload via multipart fetch, then JSON metadata POST to task files endpoint. Matches existing backend architecture.
3. **Nested Routes in ProjectDetailPage**: Instead of adding routes in WorkLayout.tsx, used nested Routes inside ProjectDetailPage since the wildcard route already exists.
4. **Tab-based activity/comments view**: Combined view with tab switching (Alle/Kommentare/Aktivitaet) rather than a single interleaved timeline, giving users control over what they see.

## Next Phase Readiness

Plan 06-08 (Entity Links and Custom Fields) can proceed. The task detail page already has placeholder sections for entity links and custom fields in the sidebar, ready to be replaced with actual implementations.

## Self-Check: PASSED
