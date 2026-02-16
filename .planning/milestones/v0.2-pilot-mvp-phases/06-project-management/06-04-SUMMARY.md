---
phase: 06-project-management
plan: 04
subsystem: api
tags: [go, grpc, gateway, http, docker, openapi, work-service, routing, proto-converters]

requires:
  - phase: 06-02
    provides: Project and Status services with repository interfaces
  - phase: 06-03
    provides: Task and Comment services with event emitters, repository interfaces

provides:
  - Work gRPC server implementing all 43 WorkService RPCs with error mapping
  - Gateway HTTP routes for ~30 Work endpoints (projects, tasks, comments, files, links, activities, preferences, search)
  - Work service entry point (cmd/work/main.go) with health checks and metrics
  - Docker configuration (Dockerfile.work, docker-compose.yml integration)
  - Migration 000027 seeding 5 work.task.* notification event types
  - OpenAPI spec documenting all Work HTTP endpoints with request/response schemas

affects: [06-05 (frontend project views), 06-06 (frontend task views)]

tech-stack:
  added: []
  patterns:
    - "RouteRegistrar pattern for Work service (same as CRM, Chat, Notification)"
    - "Admin bypass pattern (uuid.Nil + isAdmin=true) in gRPC since gateway handles auth"
    - "Thin handler pattern: parse request -> gRPC call -> format response"
    - "Proto converter functions for all Work domain models"
    - "Multi-format timestamp parsing (RFC3339, date-only) in gateway"

key-files:
  created:
    - backend/internal/server/work_grpc.go
    - backend/internal/gateway/route_work.go
    - backend/cmd/work/main.go
    - backend/Dockerfile.work
    - backend/migrations/000027_seed_work_event_types.up.sql
    - backend/migrations/000027_seed_work_event_types.down.sql
  modified:
    - backend/cmd/gateway/main.go
    - backend/Makefile
    - deploy/docker/docker-compose.yml
    - backend/api/openapi.yaml

decisions:
  - id: work-admin-bypass
    summary: "gRPC server uses uuid.Nil + isAdmin=true for authorization since gateway handles auth"
  - id: work-template-key-gen
    summary: "Template key auto-generated from first 6 uppercase chars of name + random 4-char UUID suffix"
  - id: work-route-registrar
    summary: "Work routes follow exact same RouteRegistrar pattern as CRM/Chat/Notification"
  - id: work-comment-update-bypass
    summary: "Comment update passes uuid.Nil as actorID (author-only check bypassed) since gRPC is behind gateway auth"

metrics:
  duration: ~12min
  completed: 2026-02-08
---

# Phase 6 Plan 4: Work Service Wiring Summary

Work gRPC server (43 RPCs), gateway HTTP routes (~30 endpoints), Docker config, event type seeds, and OpenAPI spec connecting project management backend to the outside world.

## Task Commits

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | gRPC server, gateway routes, service entry point | 1dd2b93 | work_grpc.go, route_work.go, cmd/work/main.go, cmd/gateway/main.go |
| 2 | Docker, Makefile, migration seed, OpenAPI spec | ce0be18 | Dockerfile.work, Makefile, docker-compose.yml, 000027_*.sql, openapi.yaml |

## Decisions Made

1. **Admin bypass in gRPC server**: Since the gateway handles authentication and authorization via middleware (JWT + RBAC), the gRPC server passes `uuid.Nil` as userID and `isAdmin=true` to service methods that require authorization checks. This avoids propagating user identity through gRPC metadata for this single-tenant service communication.

2. **Template key generation**: When saving a project as a template, the gRPC server auto-generates a project key from the first 6 uppercase alphanumeric characters of the template name plus a 4-character UUID suffix for uniqueness.

3. **Struct name collision fix**: Renamed `updateMemberRoleRequest` to `updateProjectMemberRoleRequest` in `route_work.go` to avoid collision with same-named struct in `route_chat.go` (both in `gateway` package).

4. **Timestamp parsing**: Gateway uses multi-format timestamp parsing (RFC3339, date-only `2006-01-02`) for due_date fields, returning protobuf timestamps to the gRPC service.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed broken timestamp parsing helpers**
- **Found during:** Task 1
- **Issue:** `parseTimeStr` and `parseTimeLayout` functions were incomplete/undefined, preventing compilation
- **Fix:** Replaced with proper `time.Parse` using standard Go time formats
- **Files modified:** backend/internal/gateway/route_work.go
- **Commit:** 1dd2b93

**2. [Rule 1 - Bug] Fixed struct name collision**
- **Found during:** Task 1 verification
- **Issue:** `updateMemberRoleRequest` redeclared in route_work.go, conflicting with route_chat.go
- **Fix:** Renamed to `updateProjectMemberRoleRequest`
- **Files modified:** backend/internal/gateway/route_work.go
- **Commit:** 1dd2b93

## What Was Built

### Work gRPC Server (work_grpc.go, ~1640 lines)
- 43 RPC implementations covering: Projects (5), Members (4), Templates (2), Statuses (5), Tasks (7), Dependencies (3), Comments (4), Entity Links (4), Activities (1), Files (3), Custom Fields (2), Preferences (2), Search (1)
- Proto converter functions: `projectToProto`, `taskToProto`, `commentToWorkProto`, `activityToWorkProto`, `entityLinkToProto`, `fileToProto`, `preferenceToProto`, `dependencyToProto`, `statusToProto`, `memberToProto`
- Comprehensive error mapping function (`mapWorkError`) covering all error types from project, status, task, and comment packages

### Gateway HTTP Routes (route_work.go, ~1560 lines)
- ~30 HTTP endpoints organized by resource: projects, members, templates, statuses, preferences, tasks, dependencies, comments, entity links, activities, files, custom fields, search
- Follows RouteRegistrar interface pattern with lazy gRPC client
- Permission-based route protection via middleware.RequirePermission

### Service Entry Point (cmd/work/main.go)
- Initializes repos and services: project, status, task, comment
- Sets event emitters on both task and comment services
- gRPC server with metrics interceptors
- Health/metrics HTTP server on configurable port
- Graceful shutdown handling

### Docker & Build Config
- Dockerfile.work: Multi-stage alpine build, ports 50055/9095
- docker-compose.yml: work service with postgres dependency, gateway updated with work dependency + env vars
- Makefile: build and run-work targets, proto generation for work.proto

### Migration 000027
- Seeds 5 notification event types: work.task.created, work.task.assigned, work.task.status_changed, work.task.commented, work.task.mentioned

### OpenAPI Spec
- 30+ endpoint paths documented with request/response schemas
- 16 new component schemas (ProjectResponse, TaskResponse, CreateTaskRequest, etc.)
- 7 new tags (projects, tasks, task-comments, task-files, task-links, work-search)

## Verification Results

- All 6 binaries compile: `go build ./cmd/{gateway,auth,crm,chat,notification,work}`
- `go vet` passes on all affected packages
- No linting issues

## Next Phase Readiness

The Work service backend is now fully wired. All business logic from plans 06-02 and 06-03 is accessible via HTTP through the gateway. The next plans (06-05+) can build frontend views that call these endpoints.

## Self-Check: PASSED
