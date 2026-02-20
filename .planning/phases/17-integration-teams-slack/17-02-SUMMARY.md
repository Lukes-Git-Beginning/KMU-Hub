---
phase: 17-integration-teams-slack
plan: 02
subsystem: api, integration
tags: [teams, slack, bot-framework, block-kit, adaptive-cards, grpc, gateway, webhooks, oauth]

# Dependency graph
requires:
  - phase: 17-integration-teams-slack
    provides: 5 PostgreSQL tables, Go domain models, repository layer, 13 proto RPCs, Teams/Slack Go deps
  - phase: 04-notifications-gateway
    provides: notification event system, DeliveryCallback pattern, dispatcher
provides:
  - Platform-agnostic notification forwarder with module-based routing and auto-disable
  - Teams Bot Framework client with Adaptive Card v1.4 and Action.Execute
  - Slack API client with Block Kit messages and in-place card updates
  - Account linking service with SHA-256 token flow
  - Per-channel rate limiter (1 msg/sec Slack, 4 msg/sec Teams)
  - Teams and Slack inbound webhook handlers with signature verification
  - 13 integration gRPC RPCs on notification service
  - 18 HTTP endpoints on gateway (admin config, mappings, account links, webhooks)
  - Notification binary with integration forwarder as DeliveryCallback
  - Docker Compose integration env vars for notification and gateway
affects: [17-03, 18-bexio, 19-abacus-rma]

# Tech tracking
tech-stack:
  added: []
  patterns: [PlatformPoster interface for adapter pattern, functional option pattern for gRPC server deps, nil-safe platform client initialization]

key-files:
  created:
    - backend/internal/notification/integration/forwarder.go
    - backend/internal/notification/integration/rate_limiter.go
    - backend/internal/notification/integration/account_link_service.go
    - backend/internal/notification/integration/teams/client.go
    - backend/internal/notification/integration/teams/card_builder.go
    - backend/internal/notification/integration/teams/webhook_handler.go
    - backend/internal/notification/integration/slack/client.go
    - backend/internal/notification/integration/slack/block_builder.go
    - backend/internal/notification/integration/slack/webhook_handler.go
    - backend/internal/notification/integration/slack/oauth.go
    - backend/internal/gateway/route_integration.go
  modified:
    - backend/internal/server/notification_grpc.go
    - backend/proto/notification/v1/notification.pb.go
    - backend/proto/notification/v1/notification_grpc.pb.go
    - backend/cmd/notification/main.go
    - deploy/docker/docker-compose.yml

key-decisions:
  - "PlatformPoster interface decouples forwarder from concrete Teams/Slack clients"
  - "Proto codegen regenerated (was missing from 17-01, blocking server/gateway compilation)"
  - "WithIntegration functional option pattern for backward-compatible gRPC server extension"
  - "Nil-safe platform initialization: missing env vars = platform disabled, not crash"
  - "Inbound webhook routes bypass JWT auth but verify platform-specific signatures"

patterns-established:
  - "PlatformPoster adapter interface: PostNotification(ctx, mapping, notif, actions) for platform clients"
  - "Functional option pattern for gRPC server dependency injection (WithIntegration)"
  - "Gateway webhook proxy pattern: IntegrationRoutes delegates to injected webhook handlers"

requirements-completed: [INT-04, INT-05, INT-06]

# Metrics
duration: 4min
completed: 2026-02-20
---

# Phase 17 Plan 02: Forwarder Engine + Platform Adapters Summary

**Notification forwarder with Teams Bot Framework and Slack API adapters, 18 HTTP gateway endpoints, inbound webhook handlers with platform signature verification, and notification binary integration**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-20T15:47:53Z
- **Completed:** 2026-02-20T15:52:00Z
- **Tasks:** 3
- **Files modified:** 16

## Accomplishments
- Built platform-agnostic notification forwarder with module-based routing (most-specific-wins), 30s mapping cache, rate limiting, and auto-disable after 10+ consecutive failures
- Implemented Teams Bot Framework client with Adaptive Card v1.4 (Action.Execute buttons, in-place card updates) and Slack API client with Block Kit messages (chat.postMessage, chat.update, PostEphemeral)
- Created inbound webhook handlers for both platforms: Teams processes invoke/message activities with JWT verification, Slack verifies signing secrets for interactive messages and slash commands
- Extended notification gRPC server with 13 integration RPCs and gateway with 18 HTTP endpoints covering admin config, channel mappings, account linking, and inbound webhooks
- Wired integration forwarder as DeliveryCallback in notification binary with nil-safe platform initialization

## Task Commits

Each task was committed atomically:

1. **Task 1: Forwarder engine + platform adapters + account linking + rate limiter** - `8abd405` (feat)
2. **Task 2: Inbound webhook handlers + gRPC server + gateway routes** - `c8dab3f` (feat)
3. **Task 3: Notification binary integration + Docker Compose** - `c0f0194` (feat)

## Files Created/Modified
- `backend/internal/notification/integration/forwarder.go` - Platform-agnostic notification forwarder with MappingCache, failure tracking, auto-disable
- `backend/internal/notification/integration/rate_limiter.go` - Per-channel token bucket rate limiter (1/sec Slack, 4/sec Teams)
- `backend/internal/notification/integration/account_link_service.go` - Token-based account linking with SHA-256 hashing and 5-min expiry
- `backend/internal/notification/integration/teams/client.go` - Teams Bot Framework client with proactive messaging
- `backend/internal/notification/integration/teams/card_builder.go` - Adaptive Card v1.4 builder with module icons and action buttons
- `backend/internal/notification/integration/teams/webhook_handler.go` - Teams inbound handler for invoke (Action.Execute) and message (/kmuhub link)
- `backend/internal/notification/integration/slack/client.go` - Slack API client for chat.postMessage, chat.update, PostEphemeral
- `backend/internal/notification/integration/slack/block_builder.go` - Block Kit message builder with module colors and action buttons
- `backend/internal/notification/integration/slack/webhook_handler.go` - Slack inbound handler for interactive messages and slash commands
- `backend/internal/notification/integration/slack/oauth.go` - Slack OAuth install flow (authorize redirect + token exchange)
- `backend/internal/gateway/route_integration.go` - 18 HTTP routes: admin config CRUD, channel mappings, account linking, inbound webhooks
- `backend/internal/server/notification_grpc.go` - 13 integration gRPC RPCs with functional option pattern
- `backend/proto/notification/v1/notification.pb.go` - Regenerated proto Go code with integration types
- `backend/proto/notification/v1/notification_grpc.pb.go` - Regenerated proto gRPC Go code with integration service
- `backend/cmd/notification/main.go` - Integration forwarder as DeliveryCallback, Teams/Slack client init, WithIntegration option
- `deploy/docker/docker-compose.yml` - Integration env vars for notification and gateway services

## Decisions Made
- PlatformPoster interface (`PostNotification(ctx, mapping, notif, actions)`) decouples the forwarder from concrete Teams/Slack clients, enabling nil-safe disabled-platform handling
- Regenerated proto Go code that was missing from 17-01 commit (proto extended but codegen not run), which was blocking server and gateway package compilation
- Used functional option pattern `WithIntegration(repo, linkService)` on NotificationGRPCServer for backward-compatible dependency injection without changing the base constructor signature
- Nil-safe platform initialization: empty TEAMS_APP_ID or SLACK_BOT_TOKEN env vars result in nil client (platform disabled), not a startup crash
- Inbound webhook routes bypass standard JWT auth middleware but verify platform-specific signatures (Slack signing secret HMAC-SHA256, Teams Bot Framework JWT)
- Context-dependent action buttons: HR leave events get acknowledge+approve+reject, task/CRM get acknowledge+reply, finance/default get acknowledge only
- Bot-only identity by default per DSGVO recommendation (research discretion #1): cards show "KMU Hub" as sender, not actor name

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Regenerated proto Go code (missing from 17-01)**
- **Found during:** Task 2 (gRPC server compilation)
- **Issue:** Plan 17-01 extended notification.proto with 13 integration RPCs and messages, but protoc was never run to regenerate the Go code. Server and gateway packages failed to compile because ListIntegrationConfigsRequest etc. were undefined in the generated package.
- **Fix:** Ran protoc to regenerate notification.pb.go and notification_grpc.pb.go from the extended .proto file
- **Files modified:** backend/proto/notification/v1/notification.pb.go, backend/proto/notification/v1/notification_grpc.pb.go
- **Verification:** `go build ./internal/server/...` and `go build ./internal/gateway/...` compile successfully
- **Committed in:** c8dab3f (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Necessary fix for proto codegen that was missed in 17-01. No scope creep.

## Issues Encountered
- Proto regeneration initially output to wrong directory (`backend/notification/v1/` instead of `backend/proto/notification/v1/`) due to `-I proto` flag in protoc invocation. Fixed by removing the `-I` flag and using the full relative path to the proto file.

## User Setup Required
None - integration is disabled by default when environment variables are not set. Teams/Slack setup requires admin configuration in KMU Hub settings (covered in Plan 17-03 frontend).

## Next Phase Readiness
- Full backend for Teams & Slack integration complete: forwarder, adapters, webhook handlers, gRPC RPCs, gateway routes
- Plan 17-03 (frontend) can build the admin settings UI for integration configuration, setup wizards, and account linking
- All 18 HTTP endpoints ready for frontend consumption

## Self-Check: PASSED

All 16 created/modified files verified on disk. All 3 task commits (8abd405, c8dab3f, c0f0194) verified in git log.

---
*Phase: 17-integration-teams-slack*
*Completed: 2026-02-20*
