---
phase: 04-notifications-gateway
plan: 01
subsystem: gateway
tags: [go, grpc, gateway, refactoring, service-registry, modular-architecture]
dependency-graph:
  requires: [phase-01, phase-02, phase-03]
  provides: [modular-gateway, service-registry, per-service-routes, graceful-degradation]
  affects: [04-02, 04-03, all-future-phases]
tech-stack:
  added: []
  patterns: [service-registry, route-registrar-interface, lazy-grpc-connections, per-service-503-degradation]
key-files:
  created:
    - backend/internal/gateway/registry.go
    - backend/internal/gateway/registry_test.go
    - backend/internal/gateway/route_registrar.go
    - backend/internal/gateway/helpers.go
    - backend/internal/gateway/route_auth.go
    - backend/internal/gateway/route_crm.go
    - backend/internal/gateway/route_chat.go
    - backend/internal/gateway/route_health.go
  modified:
    - backend/cmd/gateway/main.go
    - backend/internal/server/http.go
    - backend/internal/config/config.go
    - deploy/docker/docker-compose.yml
decisions:
  - "Keep WebSocket hub setup in main.go (cross-cutting concern, not per-service)"
  - "File upload handler stays in server package, registered by main.go alongside wsHub"
  - "respondGRPCError and helpers moved to gateway package, not reused from server"
  - "HealthHandler kept in server/http.go since auth/crm/chat services import it"
  - "Notification service config fields added to config.go as preparation for Plan 02"
metrics:
  duration: "~10 minutes"
  completed: "2026-02-07"
---

# Phase 4 Plan 1: Gateway Modularization Summary

Refactored the monolithic 3300-line GatewayHandler into a modular, lazy-connecting architecture with ServiceRegistry, per-service route handlers, and 503 graceful degradation per-service.

## Task Commits

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | ServiceRegistry + RouteRegistrar Interface + Helpers | `6700bfc` | registry.go, registry_test.go, route_registrar.go, helpers.go |
| 2 | Extract Monolithic Handler into Per-Service Route Files | `4a89675` | route_auth.go, route_crm.go, route_chat.go, route_health.go, main.go, http.go, config.go, docker-compose.yml |

## What Changed

### ServiceRegistry (registry.go)
- Manages lazy gRPC connections to backend services
- `Register(name, address)` stores address without connecting
- `GetConnection(name)` creates/caches connections with double-checked locking
- `Close()` gracefully shuts down all connections
- `grpc.NewClient()` is non-blocking; actual TCP connection happens on first RPC call

### RouteRegistrar Interface (route_registrar.go)
- `RegisterRoutes(r chi.Router, authMiddleware)` + `ServiceName() string`
- Each per-service route module implements this interface
- Adding a new service = create route_xxx.go + 2 lines in main.go

### Per-Service Route Files
- **route_auth.go**: Auth, Users, Invitations (17 handlers)
- **route_crm.go**: Custom Fields, Tags, Contacts, Companies, Pipeline Stages, Deals, Activities, Search, Saved Filters, Reports (40+ handlers)
- **route_chat.go**: Channels, Messages, DMs, Threads, Mentions, Read Receipts, Files, Chat Search (25+ handlers)
- **route_health.go**: Health check with registered services list

### Graceful Degradation
- Each handler calls `getXxxClient()` at start, returns 503 "service unavailable" on failure
- Gateway starts even if backend services are unreachable
- Docker Compose gateway depends_on relaxed to `service_started` for auth/crm/chat

### main.go Rewrite
- Replaced 3 eager `grpc.NewClient()` calls with single `ServiceRegistry`
- Route registration via loop over `[]RouteRegistrar`
- WebSocket hub setup extracted to `setupWebSocketHub()` helper
- `defer registry.Close()` replaces 3 individual `defer conn.Close()`

### http.go Cleanup
- Reduced from 3353 lines to 31 lines
- Only `HealthHandler` function remains (used by auth/crm/chat services)

## Deviations from Plan

None - plan executed exactly as written.

## Decisions Made

1. **WebSocket hub in main.go**: The WebSocket hub needs both chat and auth gRPC clients and is cross-cutting. Keeping it in main.go is cleaner than forcing it into a single route module.

2. **File upload in main.go**: The file upload handler depends on the WebSocket hub for broadcast. It stays registered in main.go alongside the hub.

3. **HealthHandler in server/http.go**: The standalone `HealthHandler` function is imported by auth, crm, and chat service main.go files. Kept in server package to avoid breaking those imports.

4. **Notification config preparation**: Added `NotificationGRPCPort`, `NotificationGRPCAddress`, and `NotificationHealthPort` to config.go for Plan 02.

## Next Phase Readiness

### Plan 02 (Notification Service)
- Config fields ready in config.go
- Adding notification routes requires: create `route_notification.go`, add `registry.Register("notification", cfg.NotificationGRPCAddress)` and `NewNotificationRoutes(registry)` to registrars list in main.go

### Plan 03 (WebSocket Extension)
- WebSocket hub stays in `internal/server/websocket.go` as planned
- The hub already uses registry connections for chat and auth clients

## Self-Check: PASSED
