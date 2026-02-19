---
phase: "06"
plan: "01"
subsystem: "work-service-foundation"
tags: ["protobuf", "grpc", "migrations", "models", "database-schema"]
dependency-graph:
  requires: ["phase-04-notifications", "phase-02-crm"]
  provides: ["work-service-proto", "project-task-schema", "project-task-models", "work-config"]
  affects: ["06-02", "06-03", "06-04", "06-05", "06-06", "06-07", "06-08"]
tech-stack:
  added: []
  patterns: ["work-service-proto-definition", "task-priority-namespacing", "project-status-workflow"]
key-files:
  created:
    - "backend/proto/work/v1/work.proto"
    - "backend/proto/work/v1/work.pb.go"
    - "backend/proto/work/v1/work_grpc.pb.go"
    - "backend/migrations/000024_create_projects.up.sql"
    - "backend/migrations/000024_create_projects.down.sql"
    - "backend/migrations/000025_create_tasks.up.sql"
    - "backend/migrations/000025_create_tasks.down.sql"
    - "backend/migrations/000026_create_task_collaboration.up.sql"
    - "backend/migrations/000026_create_task_collaboration.down.sql"
    - "backend/internal/models/project.go"
    - "backend/internal/models/task.go"
  modified:
    - "backend/internal/models/custom_field.go"
    - "backend/internal/config/config.go"
decisions:
  - id: "06-01-01"
    decision: "Prefix task constants with Task* to avoid collision with notification model"
    rationale: "notification.go already defines PriorityUrgent/PriorityLow/ValidPriorities; task priorities have different set (urgent/high/medium/low vs urgent/normal/low)"
    alternatives: ["Rename notification constants instead", "Use separate packages"]
metrics:
  duration: "~6min"
  completed: "2026-02-08"
---

# Phase 6 Plan 1: Proto, Migrations, and Models Summary

**One-liner:** Work service proto (43 RPCs), 11 database tables with FTS and RBAC, Go models with namespaced constants

## What Was Done

### Task 1: Work service proto definition
- Created `work.proto` with `WorkService` defining 43 gRPC RPCs across 10 domains:
  - Projects (5 RPCs), Project Members (4), Templates (2), Project Statuses (5)
  - Tasks (7), Task Dependencies (3), Comments (4), Entity Links (4)
  - Activity Log (1), Files (3), Custom Fields (2), Preferences (2), Search (1)
- All message types use `google.protobuf.Timestamp` for timestamps and `string` for UUIDs
- ListTasks supports comprehensive filtering: project, assignee, status, priority, date range, parent task, search, pagination, sorting
- SearchTasks supports multi-project search with combined filters
- Generated Go code compiles without errors

### Task 2: Database migrations (11 tables)
- **Migration 000024** (projects foundation): `projects`, `project_members`, `project_statuses`
  - Unique project_key among non-archived projects
  - Role-based membership (owner/member/viewer)
  - Customizable status workflow per project (default, closed flags)
  - RBAC permission seeds for projects and tasks resources
- **Migration 000025** (tasks): `tasks`, `task_dependencies`
  - Hierarchical tasks with parent_task_id + depth
  - Full-text search trigger (German config, same pattern as CRM)
  - Priority constraint (urgent/high/medium/low)
  - Dependency types: blocks, blocked_by, relates_to, duplicates
  - 10 indexes including GIN for search vector
- **Migration 000026** (collaboration): `task_comments`, `task_entity_links`, `task_activities`, `task_files`, `user_project_preferences`, `task_custom_field_values`
  - Comments with quote/reply support
  - Entity links to contact/company/deal/channel/message
  - Activity log with JSONB old_value/new_value
  - File attachments via MinIO storage keys
  - Per-user view preferences (list/kanban, grouping, sorting)
  - Reuses existing custom_field_definitions table

### Task 3: Go models and config
- Created `models/project.go`: Project, ProjectWithDetails, ProjectMember, ProjectStatus, UserProjectPreference
- Created `models/task.go`: Task, TaskWithRelations, TaskDependency, TaskComment, TaskEntityLink, TaskActivity, TaskFile
- Validation constants with `Task` prefix to avoid namespace collision
- Extended EntityType with `EntityTypeTask = "task"` in custom_field.go
- Added WorkGRPCPort (:50055), WorkGRPCAddress, WorkHealthPort (:9095) to config

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed priority constant naming collision with notification model**
- **Found during:** Task 3
- **Issue:** `PriorityUrgent`, `PriorityLow`, and `ValidPriorities` already declared in `notification.go` with different value sets (notification: urgent/normal/low vs task: urgent/high/medium/low)
- **Fix:** Prefixed task constants with `Task` prefix: `TaskPriorityUrgent`, `TaskPriorityHigh`, `TaskPriorityMedium`, `TaskPriorityLow`, `ValidTaskPriorities`. Also prefixed action constants: `TaskActionCreated`, `TaskActionStatusChanged`, etc.
- **Files modified:** `backend/internal/models/task.go`
- **Commit:** bcbd782

## Task Commits

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | Work service proto | 066d78e | work.proto, work.pb.go, work_grpc.pb.go |
| 2 | Database migrations | 04e34f6 | 000024-000026 (6 migration files) |
| 3 | Go models + config | bcbd782 | project.go, task.go, custom_field.go, config.go |

## Verification Results

- Proto compilation: PASSED
- Full backend build (`go build ./...`): PASSED
- Go vet on models/config/proto: PASSED
- 43 RPCs defined in WorkService
- 11 tables created across 3 migrations
- All model structs have correct json tags (snake_case)

## Decisions Made

| ID | Decision | Rationale |
|----|----------|-----------|
| 06-01-01 | Prefix task constants with Task* | Avoid collision with notification model's priority constants which use different value sets |

## Next Phase Readiness

Plan 06-02 (repository layer) can proceed. All artifacts are in place:
- Proto defines the gRPC contract for all Work service operations
- Migrations define the complete database schema
- Models provide Go types matching the schema
- Config enables the Work service to bind its ports

## Self-Check: PASSED
