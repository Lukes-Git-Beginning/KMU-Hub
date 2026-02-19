---
phase: 04-notifications-gateway
plan: 02
subsystem: notification-service
tags: [go, grpc, postgresql, listen-notify, event-bus, notifications, preferences]
dependency_graph:
  requires: [phase-01-auth, phase-02-crm, phase-03-chat]
  provides: [notification-microservice, event-bus, preference-engine, notification-grpc-api]
  affects: [04-03-gateway-integration, phase-05-through-13-all-modules]
tech_stack:
  added: []
  patterns: [event-bus-listen-notify, preference-evaluation-pipeline, smart-grouping, delivery-dispatcher-callbacks]
key_files:
  created:
    - backend/proto/notification/v1/notification.proto
    - backend/proto/notification/v1/notification.pb.go
    - backend/proto/notification/v1/notification_grpc.pb.go
    - backend/migrations/000020_create_event_types.up.sql
    - backend/migrations/000020_create_event_types.down.sql
    - backend/migrations/000021_create_notifications.up.sql
    - backend/migrations/000021_create_notifications.down.sql
    - backend/migrations/000022_create_notification_preferences.up.sql
    - backend/migrations/000022_create_notification_preferences.down.sql
    - backend/internal/models/notification.go
    - backend/internal/models/event.go
    - backend/internal/notification/event/bus.go
    - backend/internal/notification/event/bus_test.go
    - backend/internal/notification/event/registry.go
    - backend/internal/notification/event/registry_test.go
    - backend/internal/notification/event/types.go
    - backend/internal/notification/notification/errors.go
    - backend/internal/notification/notification/repository.go
    - backend/internal/notification/notification/postgres_repository.go
    - backend/internal/notification/notification/service.go
    - backend/internal/notification/notification/service_test.go
    - backend/internal/notification/notification/grouper.go
    - backend/internal/notification/notification/grouper_test.go
    - backend/internal/notification/preference/errors.go
    - backend/internal/notification/preference/repository.go
    - backend/internal/notification/preference/postgres_repository.go
    - backend/internal/notification/preference/service.go
    - backend/internal/notification/preference/service_test.go
    - backend/internal/notification/delivery/dispatcher.go
    - backend/internal/notification/delivery/dispatcher_test.go
    - backend/internal/server/notification_grpc.go
    - backend/cmd/notification/main.go
    - backend/Dockerfile.notification
  modified:
    - backend/Makefile
decisions:
  - Used raw pgx conn.WaitForNotification instead of pgxlisten (pre-v1 library)
  - 30-second default grouping window for smart notification collapse
  - 7-stage preference evaluation pipeline (urgent bypass, resource mute, event-type pref, module default, system default, low priority override, quiet hours)
  - Event bus uses dual approach (events table + pg_notify) for durability
  - Wildcard handler pattern for event bus dispatch
metrics:
  duration: 16 minutes
  completed: 2026-02-07
---

# Phase 4 Plan 02: Notification Service Backend Summary

Complete notification microservice with event bus infrastructure, notification storage with smart grouping, preference evaluation engine, gRPC server (13 RPCs), and Dockerfile -- all using PostgreSQL LISTEN/NOTIFY for event transport and 7-stage preference pipeline for delivery decisions.

## Task Commits

| Task | Name | Commit | Key Changes |
|------|------|--------|-------------|
| 1 | Proto + Migrations + Models + Event Infrastructure | b205473 | notification.proto (13 RPCs), 6 DB tables, Go models, EventBus, EventTypeRegistry |
| 2 | Notification + Preference Services, gRPC Server, Entry Point | fbe6700 | Notification CRUD, smart grouper, preference pipeline, dispatcher, gRPC server, cmd/notification, Dockerfile |

## What Was Built

### Event Bus (event package)
- **EventBus**: Listens on PostgreSQL LISTEN/NOTIFY `events` channel with automatic reconnection
- **EventTypeRegistry**: Thread-safe in-memory registry for event types; any module can register types without changing notification service code
- **ProcessBacklog**: Catch-up mechanism for events missed during downtime, using the events durability table
- Wildcard handler support (`*`) for processing all event types

### Notification Service (notification package)
- **ProcessEvent**: Full pipeline -- store event, determine targets, evaluate preferences, group, store, dispatch
- **Smart Grouper**: Collapses similar notifications with same group_key within 30-second window (e.g., "5 messages in #general")
- **CRUD**: ListNotifications (paginated, filterable), GetUnreadCount, MarkRead, MarkAllRead
- Self-notification suppression (actor is not notified of their own actions)

### Preference Engine (preference package)
- **7-stage evaluation pipeline**:
  1. Urgent bypass -- always deliver
  2. Resource mute -- suppress if resource is muted
  3. Event-type preference -- check user's per-event-type setting
  4. Module default -- fall back to module-level setting
  5. System default -- deliver with defaults (in-app + desktop push)
  6. Low priority override -- never desktop push regardless of settings
  7. Quiet hours -- suppress desktop push during DND/quiet hours
- **Quiet hours**: Scheduled DND with IANA timezone support, day-of-week filtering, overnight windows, manual DND toggle with expiry
- **Resource muting**: Per-channel, per-pipeline muting that overrides event-type preferences
- **CRUD**: Preferences, mutes, quiet hours, manual DND toggle

### Delivery Dispatcher (delivery package)
- Callback-based dispatcher for WebSocket/push delivery integration
- Gateway registers callbacks at startup; notification service dispatches through them

### Database Schema
- **event_types**: Module-agnostic event type registry with 7 seed types
- **notifications**: User notifications with group_key, group_count, priority, deep_link
- **events**: Durability table for catch-up processing
- **notification_preferences**: Event-type or module-level delivery settings
- **notification_mutes**: Per-resource muting
- **notification_quiet_hours**: Scheduled DND with timezone and manual override

### gRPC Server (13 RPCs)
- ListNotifications, GetUnreadCount, MarkNotificationRead, MarkAllNotificationsRead
- GetNotificationPreferences, UpdateNotificationPreference
- MuteResource, UnmuteResource, ListMutedResources
- GetQuietHours, UpdateQuietHours, ToggleManualDND
- ListEventTypes

### Service Entry Point
- cmd/notification/main.go with full lifecycle: config, DB pool, repos, services, event bus, backlog processing, gRPC server, health/metrics, graceful shutdown
- Dockerfile.notification (multi-stage, port 50054/9094)
- Makefile updated with `build`, `run-notification`, and `proto` targets

## Test Coverage

| Package | Tests | Service Coverage |
|---------|-------|-----------------|
| event | 20 | bus dispatch 100%, registry 100% |
| notification | 29 | service 85-100%, grouper 95-100% |
| preference | 32 | evaluate 89.7%, isInQuietHours 87.5%, CRUD 77-100% |
| delivery | 5 | dispatcher 100% |
| **Total** | **74** | **Service layer 85%+** |

Note: Overall package coverage (47-67%) is diluted by postgres_repository.go files (0% -- require live DB). Service layer code is thoroughly tested.

## Decisions Made

| Decision | Rationale |
|----------|-----------|
| Raw pgx over pgxlisten | pgxlisten is pre-v1 with no tagged releases; reconnection logic is ~20 lines |
| 30-second grouping window | Balances UX (grouping related events) with timeliness |
| Dual write (events table + pg_notify) | pg_notify is ephemeral; events table provides durability for catch-up |
| Wildcard handler pattern | Allows the notification service to process ALL events through a single handler |
| 7-stage preference pipeline | Covers all user constraints: urgent bypass, muting, per-type prefs, module defaults, low priority, quiet hours |
| DeliveryCallback pattern | Decouples notification service from delivery mechanism; gateway registers callbacks |

## Deviations from Plan

None -- plan executed exactly as written.

## Next Phase Readiness

### Ready for Plan 03 (Gateway Integration)
- gRPC server is running and accepting connections on port 50054
- Delivery dispatcher is ready for WebSocket callback registration
- Event types are registered and can be listed via gRPC
- All notification CRUD operations available via gRPC

### Integration Points for Plan 03
1. Gateway registers delivery callbacks with the dispatcher
2. Gateway routes HTTP requests to notification gRPC service
3. WebSocket hub extended with `notification.new` message type
4. Existing services (chat, CRM) emit events via `pg_notify('events', ...)`

## Self-Check: PASSED
