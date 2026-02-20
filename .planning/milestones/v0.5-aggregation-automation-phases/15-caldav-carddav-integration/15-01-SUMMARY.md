---
phase: 15-caldav-carddav-integration
plan: 01
subsystem: api
tags: [caldav, carddav, bcrypt, sync-token, go-webdav, go-ical, rfc6578, app-passwords]

# Dependency graph
requires:
  - phase: 07-calendar-scheduling
    provides: Calendar events and models for CalDAV sync
  - phase: 02-crm-core
    provides: Contact models for CardDAV sync
provides:
  - AppSpecificPassword service with bcrypt-based create/validate/revoke
  - SyncTokenService with RFC 6578 incremental sync version tracking and change log
  - PostgreSQL migrations for app_specific_passwords, caldav_sync_versions, caldav_change_log, caldav_settings
  - Go models for all CalDAV entities
  - go-webdav v0.7.0 and go-ical Go dependencies
affects: [15-caldav-carddav-integration, 16-automation-engine]

# Tech tracking
tech-stack:
  added: [go-webdav v0.7.0, go-ical]
  patterns: [app-specific password auth, sync token versioning, change log for incremental sync]

key-files:
  created:
    - backend/migrations/000049_create_app_passwords.up.sql
    - backend/migrations/000049_create_app_passwords.down.sql
    - backend/migrations/000050_create_caldav_sync.up.sql
    - backend/migrations/000050_create_caldav_sync.down.sql
    - backend/internal/models/caldav.go
    - backend/internal/caldav/errors.go
    - backend/internal/caldav/app_password_repo.go
    - backend/internal/caldav/postgres_app_password.go
    - backend/internal/caldav/app_password.go
    - backend/internal/caldav/sync_token.go
    - backend/tools/caldav_deps.go
  modified:
    - backend/go.mod
    - backend/go.sum

key-decisions:
  - "User UUID as CalDAV username for v1 simplicity (avoids email resolution via auth gRPC)"
  - "Bcrypt cost 12 for app-specific passwords (balance of security and validation speed)"
  - "Sync token format sync-token-{N} for human-readable debugging"

patterns-established:
  - "App-specific password pattern: generate random hex, bcrypt hash, store prefix for UI identification"
  - "Sync token pattern: atomic version increment + change log insert in single transaction"
  - "CalDAV settings table for org-level feature toggles (key-value with upsert)"

requirements-completed: [INT-03]

# Metrics
duration: 3min
completed: 2026-02-20
---

# Phase 15 Plan 01: CalDAV Data Foundation Summary

**App-specific passwords with bcrypt auth, sync token versioning with RFC 6578 change log, and go-webdav/go-ical dependencies for CalDAV/CardDAV integration**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-20T10:58:27Z
- **Completed:** 2026-02-20T11:01:52Z
- **Tasks:** 2
- **Files modified:** 13

## Accomplishments
- Two migration pairs (000049, 000050) for app-specific passwords and CalDAV sync tracking
- App-specific password service with bcrypt cost 12 create/validate/list/revoke and org-level toggle
- Sync token service for RFC 6578 incremental sync with atomic version increment and change log
- Go models for AppSpecificPassword, CalDAVSyncVersion, CalDAVChangeLogEntry, CalDAVSetting
- go-webdav v0.7.0 and go-ical added to go.mod

## Task Commits

Each task was committed atomically:

1. **Task 1: Migrations + models + Go deps** - `de397f6` (feat)
2. **Task 2: App-specific password service + sync token service** - `3f82d72` (feat)

## Files Created/Modified
- `backend/migrations/000049_create_app_passwords.up.sql` - App-specific passwords table with bcrypt hash storage
- `backend/migrations/000049_create_app_passwords.down.sql` - Down migration for app passwords
- `backend/migrations/000050_create_caldav_sync.up.sql` - CalDAV sync versions, change log, and settings tables
- `backend/migrations/000050_create_caldav_sync.down.sql` - Down migration for sync tables
- `backend/internal/models/caldav.go` - Go models for all CalDAV entities
- `backend/internal/caldav/errors.go` - Domain errors for caldav package
- `backend/internal/caldav/app_password_repo.go` - AppPasswordRepository interface
- `backend/internal/caldav/postgres_app_password.go` - PostgreSQL implementation of AppPasswordRepository
- `backend/internal/caldav/app_password.go` - AppPasswordService (create, validate, list, revoke, org toggle)
- `backend/internal/caldav/sync_token.go` - SyncTokenService (get, increment, changes, parse sync tokens)
- `backend/tools/caldav_deps.go` - Build-time dependency imports for go-webdav and go-ical
- `backend/go.mod` - Added go-webdav v0.7.0 and go-ical dependencies
- `backend/go.sum` - Updated dependency checksums

## Decisions Made
- Used user UUID as CalDAV username for v1 simplicity (avoids email resolution via auth gRPC)
- Bcrypt cost 12 for app-specific passwords (balance of security and validation speed across multiple passwords)
- Sync token format "sync-token-{N}" for human-readable debugging in CalDAV clients
- Empty slice (not nil) returned from List/FindActiveByUser for consistent JSON serialization
- caldav_settings key-value table for org-level toggles with upsert semantics

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- CalDAV data foundation complete: models, migrations, services ready
- Plan 15-02 can build CalDAV/CardDAV backend adapters on top of these services
- AppPasswordService provides HTTP Basic Auth validation for CalDAV protocol handlers
- SyncTokenService provides incremental sync primitives for RFC 6578 sync-collection

## Self-Check: PASSED

- All 11 created files verified on disk
- Commit de397f6 (Task 1) verified in git log
- Commit 3f82d72 (Task 2) verified in git log
- Go build ./internal/caldav/ compiles
- Go build ./internal/models/ compiles
- go-webdav and go-ical present in go.mod
- No fmt.Println in new files

---
*Phase: 15-caldav-carddav-integration*
*Completed: 2026-02-20*
