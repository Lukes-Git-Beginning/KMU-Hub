---
phase: 17-integration-teams-slack
plan: 01
subsystem: database, api
tags: [teams, slack, integration, postgresql, protobuf, pgx, msbotbuilder-go, slack-go]

# Dependency graph
requires:
  - phase: 04-notifications-gateway
    provides: notification event system, pg_notify event bus, delivery callback pattern
  - phase: 09-security-compliance
    provides: vault service for encrypted credential storage
provides:
  - 5 PostgreSQL tables for integration persistence (configs, channel_mappings, account_links, delivery_log, link_tokens)
  - Go domain models and repository layer for all integration entities
  - 13 new proto RPCs for integration configuration, channel mappings, and account linking
  - Go dependencies for Teams Bot Framework and Slack API
affects: [17-02, 17-03, 18-bexio, 19-abacus-rma]

# Tech tracking
tech-stack:
  added: [msbotbuilder-go v0.2.5, slack-go/slack v0.17.3, go-teams-notify/v2 v2.14.0]
  patterns: [platform-agnostic integration repository, upsert account linking, JSONB module filtering]

key-files:
  created:
    - backend/migrations/000053_create_integration_tables.up.sql
    - backend/migrations/000053_create_integration_tables.down.sql
    - backend/internal/notification/integration/types.go
    - backend/internal/notification/integration/errors.go
    - backend/internal/notification/integration/repository.go
    - backend/internal/notification/integration/postgres_repository.go
    - backend/tools/integration_deps.go
  modified:
    - backend/internal/notification/event/types.go
    - backend/proto/notification/v1/notification.proto
    - backend/go.mod
    - backend/go.sum

key-decisions:
  - "msbotbuilder-go core sub-package import required (root package has no Go files)"
  - "Upsert semantics on CreateAccountLink (ON CONFLICT DO UPDATE) per research pitfall #4 for single active link per external user"
  - "JSONB @> operator for module-level channel mapping filtering (modules stored as JSON array)"
  - "Credentials vault key reference in config, never exposed in proto responses"

patterns-established:
  - "Integration repository pattern: platform-agnostic CRUD with JSONB module filtering"
  - "Module color map: centralized hex colors for notification card theming across platforms"

requirements-completed: [INT-04, INT-05]

# Metrics
duration: 4min
completed: 2026-02-20
---

# Phase 17 Plan 01: Data Foundation Summary

**Integration data layer with 5 PostgreSQL tables, Go domain models, repository with JSONB module filtering, 13 proto RPCs, and Teams/Slack Go dependencies**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-20T15:28:29Z
- **Completed:** 2026-02-20T15:32:14Z
- **Tasks:** 2
- **Files modified:** 11

## Accomplishments
- Created 5 integration tables (configs, channel_mappings, account_links, delivery_log, link_tokens) with proper indexes, constraints, and FK relationships
- Built full Repository interface and PostgreSQL implementation covering CRUD for all entities, including JSONB-based module filtering and upsert account linking
- Extended notification.proto with 13 new RPCs for integration configuration, channel mapping management, and account linking
- Added msbotbuilder-go, slack-go/slack, and go-teams-notify/v2 to go.mod for subsequent adapter implementation

## Task Commits

Each task was committed atomically:

1. **Task 1: Database migration + Go domain models + errors** - `c009ecb` (feat)
2. **Task 2: Repository layer + proto extension + Go deps** - `db0fd75` (feat)

## Files Created/Modified
- `backend/migrations/000053_create_integration_tables.up.sql` - 5 integration tables with indexes and constraints
- `backend/migrations/000053_create_integration_tables.down.sql` - Reverse drop in dependency order
- `backend/internal/notification/integration/types.go` - Domain models, platform/status constants, module color map
- `backend/internal/notification/integration/errors.go` - 10 sentinel errors for integration operations
- `backend/internal/notification/integration/repository.go` - Repository interface with 18 methods
- `backend/internal/notification/integration/postgres_repository.go` - Full PostgreSQL implementation with pgx
- `backend/proto/notification/v1/notification.proto` - Extended with 13 integration RPCs and request/response messages
- `backend/tools/integration_deps.go` - Build tag file retaining Teams/Slack deps in go.mod
- `backend/internal/notification/event/types.go` - Added ModuleIntegration constant
- `backend/go.mod` - 3 new direct dependencies
- `backend/go.sum` - Updated checksums

## Decisions Made
- Used `github.com/infracloudio/msbotbuilder-go/core` sub-package import instead of root package (root has no Go files, only sub-packages)
- CreateAccountLink uses ON CONFLICT DO UPDATE (upsert) on (platform, external_user_id) to prevent duplicate account links per research pitfall #4
- ListActiveMappingsForModule uses JSONB containment operator (@>) with JOIN on active configs for efficient module-filtered queries
- Proto IntegrationConfigInfo intentionally omits credentials_vault_key to keep secrets server-side only

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed msbotbuilder-go import path**
- **Found during:** Task 2 (Go dependency installation)
- **Issue:** Plan specified `github.com/infracloudio/msbotbuilder-go` as import, but the root module has no Go package (only sub-packages: core, connector, protocol, schema)
- **Fix:** Changed import to `github.com/infracloudio/msbotbuilder-go/core` in integration_deps.go
- **Files modified:** backend/tools/integration_deps.go
- **Verification:** `go mod tidy` succeeds, `go build ./internal/notification/integration/...` compiles
- **Committed in:** db0fd75 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Necessary fix for Go module resolution. No scope creep.

## Issues Encountered
None beyond the auto-fixed import path issue.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Data layer complete: tables, models, repository, and proto RPCs ready for Plan 17-02 (forwarder engine + platform adapters)
- Go dependencies available for Teams Bot Framework and Slack API client usage
- Account linking flow data model supports the token-based linking described in research

## Self-Check: PASSED

All 8 created files verified on disk. Both task commits (c009ecb, db0fd75) verified in git log.

---
*Phase: 17-integration-teams-slack*
*Completed: 2026-02-20*
