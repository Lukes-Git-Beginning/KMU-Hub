---
phase: 14-event-infrastructure-unified-inbox
plan: 03
subsystem: api, infra
tags: [grpc, gateway, http, event-consumer, inbox, co-hosting, openapi]

# Dependency graph
requires:
  - phase: 14-event-infrastructure-unified-inbox
    provides: InboxService proto (27 RPCs), inbox service packages (message, team, routing, adapter), event emitters, EventBus
  - phase: 04-notification-gateway
    provides: EventBus, pg_notify, notification binary, gRPC server infrastructure
provides:
  - InboxGRPCServer implementing all 27 RPCs with error mapping
  - InboxConsumer on EventBus creating inbox_messages from all module events
  - 25 HTTP endpoints under /api/v1/inbox/ in gateway
  - InboxService co-hosted in notification binary on same gRPC port :50054
  - Snooze worker running as background goroutine
  - OpenAPI spec with inbox endpoint documentation
affects: [14-04, 16-automation-engine]

# Tech tracking
tech-stack:
  added: []
  patterns: [gRPC service co-hosting in existing binary, EventBus wildcard consumer with circular-loop prevention, page-token cursor pagination via RFC3339Nano|UUID]

key-files:
  created:
    - backend/internal/server/inbox_grpc.go
    - backend/internal/gateway/route_inbox.go
  modified:
    - backend/cmd/notification/main.go
    - backend/cmd/gateway/main.go
    - backend/api/openapi.yaml

key-decisions:
  - "InboxRoutes ServiceName returns 'notification' to reuse existing gRPC connection (co-hosted service)"
  - "InboxConsumer uses messageRepo directly for NotifyDelivery instead of exposing repo through service"
  - "Page token format is RFC3339Nano|UUID for cursor-based pagination"
  - "Docker Compose unchanged -- inbox co-hosted in notification container requires no new service"

patterns-established:
  - "gRPC service co-hosting: register multiple service servers on same grpc.Server in one binary"
  - "EventBus wildcard consumer: register handler for '*' events with module_id skip for circular prevention"
  - "Cursor page token: RFC3339Nano|UUID string encoding for cursor-based pagination"

requirements-completed: [INBOX-01, INBOX-02, INBOX-03]

# Metrics
duration: 8min
completed: 2026-02-20
---

# Phase 14 Plan 03: Inbox gRPC Server, Gateway Routes, and Event Consumer Summary

**27-RPC InboxGRPCServer with full error mapping, 25 HTTP gateway endpoints, InboxConsumer on EventBus for cross-module message ingestion, co-hosted in notification binary on :50054**

## Performance

- **Duration:** 8 min
- **Started:** 2026-02-19T23:41:44Z
- **Completed:** 2026-02-19T23:50:16Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- InboxGRPCServer implements all 27 RPCs (14 messages, 8 team inboxes, 5 routing rules) with proper error mapping from domain errors to gRPC status codes
- InboxConsumer registered on EventBus processes events from all modules into per-user inbox_messages, applies routing rules, and emits pg_notify for WebSocket push
- Gateway exposes 25 HTTP endpoints under /api/v1/inbox/ with auth middleware, admin/manager role checks on team and routing endpoints
- Notification binary co-hosts InboxService alongside NotificationService on the same gRPC port :50054 (no new binary or port needed)
- Snooze worker starts as background goroutine with 60s polling interval
- OpenAPI spec documents all inbox endpoints with complete request/response schemas

## Task Commits

Each task was committed atomically:

1. **Task 1: InboxService gRPC server + inbox event consumer** - `ef03192` (feat)
2. **Task 2: Gateway HTTP routes + OpenAPI + Docker config** - `4c71d10` (feat)

## Files Created/Modified
- `backend/internal/server/inbox_grpc.go` - InboxGRPCServer with all 27 RPCs, converters, error mapping, page token helpers
- `backend/internal/gateway/route_inbox.go` - InboxRoutes with 25 HTTP endpoints for messages, team inboxes, routing rules
- `backend/cmd/notification/main.go` - Updated to co-host InboxService, InboxConsumer, snooze worker
- `backend/cmd/gateway/main.go` - Registered InboxRoutes in route registrar list
- `backend/api/openapi.yaml` - Added inbox-messages, inbox-teams, inbox-routing tags, paths, and schemas

## Decisions Made
- InboxRoutes returns ServiceName "notification" to reuse the existing gateway-to-notification gRPC connection, since InboxService is co-hosted in the same binary
- InboxConsumer takes messageRepo directly (not through service) to access NotifyDelivery for pg_notify WebSocket push
- Page token uses RFC3339Nano|UUID format to match the cursor-based pagination in the message repository
- Docker Compose requires no changes since InboxService shares the notification container, port, and health check

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All 27 inbox RPCs available via gRPC on notification service port :50054
- All 25 HTTP endpoints available via gateway under /api/v1/inbox/
- InboxConsumer actively ingests events from all modules into unified inbox
- Routing rules evaluated and applied automatically on message creation
- Ready for Plan 04 (frontend inbox UI with React components and TanStack Query hooks)
- Channel adapters initialized with nil clients (concrete implementations needed when services fully integrated)

## Self-Check: PASSED
