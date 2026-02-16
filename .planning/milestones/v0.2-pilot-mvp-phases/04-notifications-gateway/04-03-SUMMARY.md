---
phase: 04-notifications-gateway
plan: 03
subsystem: gateway-notification-integration
tags: [gateway, notifications, websocket, docker, openapi, event-emission, pg-notify]

dependency-graph:
  requires: ["04-01", "04-02"]
  provides: ["notification-http-routes", "notification-websocket", "event-emission-crm-chat", "notification-docker-compose"]
  affects: ["05-desktop-electron", "06-work-service"]

tech-stack:
  added: []
  patterns:
    - "pg_notify dual-channel: 'events' for notification service, 'notification_delivery' for gateway WebSocket push"
    - "EventEmitter interface pattern for decoupled event emission from CRM and chat services"
    - "PGEventEmitter concrete implementation wrapping pgxpool.Pool"
    - "PostgreSQL LISTEN/NOTIFY reconnect loop with exponential backoff in gateway"
    - "RouteRegistrar pattern extended for notification routes"

key-files:
  created:
    - backend/internal/gateway/route_notification.go
    - backend/internal/notification/event/emit.go
    - backend/internal/crm/deal/event_emitter.go
    - backend/internal/chat/message/event_emitter.go
  modified:
    - backend/internal/server/websocket.go
    - backend/cmd/gateway/main.go
    - backend/internal/crm/deal/service.go
    - backend/internal/chat/message/service.go
    - backend/internal/chat/message/repository.go
    - backend/internal/chat/message/postgres_repository.go
    - backend/internal/chat/message/service_test.go
    - backend/internal/notification/notification/repository.go
    - backend/internal/notification/notification/postgres_repository.go
    - backend/internal/notification/notification/service.go
    - backend/internal/notification/notification/service_test.go
    - backend/cmd/crm/main.go
    - backend/cmd/chat/main.go
    - deploy/docker/docker-compose.yml
    - backend/api/openapi.yaml

decisions:
  - id: "04-03-01"
    title: "Dual pg_notify channels for event flow"
    detail: "'events' channel carries full EventPayload for notification service processing; 'notification_delivery' channel carries lightweight delivery payload for gateway WebSocket push"
  - id: "04-03-02"
    title: "EventEmitter as optional SetEventEmitter pattern"
    detail: "Services can optionally have an event emitter wired via SetEventEmitter to avoid mandatory constructor changes and maintain backward compatibility"
  - id: "04-03-03"
    title: "Best-effort event emission"
    detail: "Event emission errors are logged but do not fail the primary operation (message creation, deal update)"
  - id: "04-03-04"
    title: "NotifyDelivery in notification repo"
    detail: "Notification service signals gateway after storing each notification via pg_notify on delivery channel, keeping delivery decoupled from processing"

metrics:
  duration: "~20min"
  completed: "2026-02-07"
---

# Phase 04 Plan 03: Notification Delivery + Integration Summary

**One-liner:** Gateway notification HTTP routes (13 endpoints), WebSocket real-time push via pg_notify LISTEN, CRM/Chat event emission, Docker Compose notification service, and OpenAPI spec for all notification endpoints.

## What Was Built

### Task 1: Gateway Notification Routes + WebSocket Extension (ca8474a)

**HTTP Routes (route_notification.go):**
- 13 HTTP handlers implementing the `RouteRegistrar` interface
- All routes under `/api/v1/notifications` with auth middleware and permission checks
- Handlers: ListNotifications, GetUnreadCount, MarkRead, MarkAllRead, GetPreferences, UpdatePreference, ListEventTypes, MuteResource, UnmuteResource, ListMutedResources, GetQuietHours, UpdateQuietHours, ToggleDND
- Each handler obtains a lazy gRPC connection via ServiceRegistry, extracts userID from auth middleware, and proxies to the notification gRPC service

**WebSocket Extension (websocket.go):**
- Added 4 notification message types: `notification.new`, `notification.read`, `notification.read_all`, `notification.unread_count`
- Added public methods: `SendNotificationToUser`, `SendNotificationRead`, `SendNotificationReadAll`, `SendNotificationUnreadCount`
- Uses existing `sendToUser` infrastructure for targeted delivery

**Gateway Main (main.go):**
- Registered notification service in ServiceRegistry with `cfg.NotificationGRPCAddress`
- Added `NotificationRoutes` to route registrars list
- Added `startNotificationDeliveryListener` goroutine that uses raw `pgx.Connect` to LISTEN on `notification_delivery` channel
- Reconnection loop with 5-second backoff for pg_notify listener resilience
- Parses delivery payload JSON and calls `wsHub.SendNotificationToUser` for real-time push

### Task 2: Event Emission + Docker Compose + OpenAPI (f694f95)

**Event Emission Infrastructure (emit.go):**
- `EmitEvent(ctx, pool, payload)` helper that marshals `EventPayload` and calls `pg_notify('events', ...)`
- Size warning at 7500 bytes (pg_notify limit is 8000)
- Shared by both CRM and chat event emitters

**CRM Deal Service Events (deal/service.go + deal/event_emitter.go):**
- `EventEmitter` interface with `EmitDealEvent` method
- `PGEventEmitter` implementation wrapping pgxpool.Pool
- Deal stage changes emit `crm.deal.stage_changed` events targeting the deal owner
- Deal owner assignment changes emit `crm.deal.assigned` events targeting the new owner
- Events emitted best-effort after successful repository operations

**Chat Message Service Events (message/service.go + message/event_emitter.go):**
- `EventEmitter` interface with `EmitChatEvent` method
- `PGEventEmitter` implementation wrapping pgxpool.Pool
- Mentions emit `chat.mention` events (excluding self-mentions)
- DM messages emit `chat.dm.new` events targeting the DM recipient
- Added `GetDMRecipient` and `GetChannelName` to message repository for recipient resolution

**Notification Service Delivery Signal (service.go + repository.go):**
- Added `NotifyDelivery(ctx, payload)` to Repository interface + PostgreSQL implementation
- After processing each notification, the service calls `notifyDelivery` which emits a lightweight JSON payload on the `notification_delivery` pg_notify channel
- Payload contains: notification_id, user_id, title, event_type_key, module_id, priority

**Docker Compose (docker-compose.yml):**
- Added `notification` service container using `Dockerfile.notification`
- Depends on postgres (healthy) + migrate (completed)
- Exposed ports: 50054 (gRPC), 9094 (health/metrics)
- Gateway depends on notification service and has `NOTIFICATION_GRPC_ADDRESS` configured

**OpenAPI Spec (openapi.yaml):**
- Added `notifications` tag
- Added 13 endpoint paths matching all gateway routes
- Added 7 schemas: Notification, NotificationPreference, UpdateNotificationPreferenceRequest, EventType, NotificationMute, QuietHours, UpdateQuietHoursRequest

## Event Flow Architecture

```
CRM Deal Change / Chat Message
         |
         v
  PGEventEmitter.Emit*Event()
         |
         v
  pg_notify('events', EventPayload)
         |
         v
  Notification Service EventBus LISTEN
         |
         v
  ProcessEvent -> Preferences -> Grouper -> Dispatcher
         |
         v
  NotifyDelivery -> pg_notify('notification_delivery', deliveryPayload)
         |
         v
  Gateway LISTEN -> wsHub.SendNotificationToUser()
         |
         v
  WebSocket -> Electron Desktop Client
```

## Deviations from Plan

None - plan executed exactly as written.

## Task Commits

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | Gateway Notification Routes + WebSocket Extension | ca8474a | route_notification.go, websocket.go, main.go |
| 2 | Event Emission + Docker Compose + OpenAPI | f694f95 | emit.go, deal/event_emitter.go, message/event_emitter.go, docker-compose.yml, openapi.yaml |

## Verification Results

- `go build ./...` -- all packages compile
- `go test ./...` -- all tests pass (74 notification tests, all chat/CRM tests)
- Gateway builds with notification routes and delivery listener
- CRM service builds with deal event emitter wired
- Chat service builds with message event emitter wired
- Docker Compose validated with notification service container

## Next Phase Readiness

Phase 4 is now complete. All 3 plans executed:
- 04-01: Gateway modernization (ServiceRegistry, per-service route files)
- 04-02: Notification service backend (proto, migrations, event bus, preferences)
- 04-03: Notification delivery + integration (routes, WebSocket, event emission, Docker)

Ready for Phase 5 (Desktop App Electron Shell) with:
- Full notification system operational end-to-end
- WebSocket real-time push ready for Electron
- All HTTP endpoints documented in OpenAPI spec

## Self-Check: PASSED
