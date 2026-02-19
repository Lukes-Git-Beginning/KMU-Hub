---
phase: 11-documents-files-wopi-onlyoffice
plan: 01
subsystem: api, database
tags: [grpc, protobuf, postgresql, minio, wopi, docconv, document-management]

requires:
  - phase: 10-email-integration
    provides: "Email service pattern (proto, config ports, binary scaffold, tools deps)"
provides:
  - "DocumentService proto with 34 RPCs across 7 functional areas"
  - "Database schema: 7 document tables + 1 WOPI locks table with tsvector search"
  - "Go domain models for all document entities"
  - "Document service binary scaffold on :50057/:9097"
  - "SearchEntityFile added to global search types"
affects: [11-02, 11-03, 11-04, 11-05, 11-06]

tech-stack:
  added: [code.sajari.com/docconv/v2]
  patterns: [document-service-binary-isolation, folder-space-type-enum, tsvector-german-search]

key-files:
  created:
    - backend/proto/document/v1/document.proto
    - backend/migrations/000043_create_document_tables.up.sql
    - backend/migrations/000043_create_document_tables.down.sql
    - backend/migrations/000044_create_wopi_locks.up.sql
    - backend/migrations/000044_create_wopi_locks.down.sql
    - backend/internal/models/document.go
    - backend/tools/document_deps.go
    - backend/cmd/document/main.go
  modified:
    - backend/internal/models/search.go
    - backend/internal/config/config.go
    - backend/go.mod
    - backend/go.sum

key-decisions:
  - "Document service as separate binary (not shared with work service) due to OS-level docconv/poppler dependencies"
  - "German tsvector config for document search (consistent with email/CRM search)"
  - "Service ports :50057 (gRPC) and :9097 (health) following sequential port pattern"

patterns-established:
  - "Folder space type enum: personal/team/project for multi-tenant folder hierarchy"
  - "WOPI lock table with 30-minute default TTL for collaborative editing"
  - "File soft-delete with deleted_at timestamp and partial index on non-deleted files"

requirements-completed: [DOC-01, DOC-02, DOC-04, DOC-05, DOC-06, DOC-07, DOC-08, DOC-10]

duration: 5min
completed: 2026-02-17
---

# Phase 11 Plan 01: Document Data Foundation Summary

**DocumentService proto with 34 RPCs, 8 PostgreSQL tables with German tsvector search, Go domain models, and service scaffold on :50057/:9097**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-17T16:34:29Z
- **Completed:** 2026-02-17T16:39:17Z
- **Tasks:** 2
- **Files modified:** 12

## Accomplishments
- DocumentService proto defining 34 RPCs across 7 areas: folders (8), files (10), shares (4), tags (5), entity links (3), search (2), WOPI (2)
- Database migrations creating 8 tables with proper indexes, constraints, FK cascades, and a tsvector auto-update trigger
- 9 Go domain model structs matching proto messages and SQL schema
- Document service binary scaffold that compiles and listens on gRPC :50057 and health :9097

## Task Commits

Each task was committed atomically:

1. **Task 1: Proto definition + Go dependency retention** - `328ca14` (feat)
2. **Task 2: Database migrations + Go models + service scaffold** - `30fc566` (feat)

## Files Created/Modified
- `backend/proto/document/v1/document.proto` - DocumentService gRPC definition with 34 RPCs, 5 enums, 10 domain messages
- `backend/tools/document_deps.go` - Build-constrained import retaining docconv/v2 in go.mod
- `backend/migrations/000043_create_document_tables.up.sql` - 7 document tables with indexes, constraints, tsvector trigger
- `backend/migrations/000043_create_document_tables.down.sql` - Reverse migration dropping all document tables
- `backend/migrations/000044_create_wopi_locks.up.sql` - WOPI lock table with 30-min default TTL
- `backend/migrations/000044_create_wopi_locks.down.sql` - Drop wopi_locks
- `backend/internal/models/document.go` - 9 Go structs: DocumentFolder, DocumentFile, DocumentFileVersion, DocumentShare, DocumentTag, DocumentEntityLink, VirtualFile, WOPILock, FolderPathSegment
- `backend/internal/models/search.go` - Added SearchEntityFile to global search entity types
- `backend/internal/config/config.go` - Added DocumentGRPCPort and DocumentHealthPort
- `backend/cmd/document/main.go` - Document service binary with gRPC + health/metrics servers
- `backend/go.mod` - Added code.sajari.com/docconv/v2 dependency
- `backend/go.sum` - Updated checksums

## Decisions Made
- Document service runs as separate binary (isolated from work service) because docconv requires OS-level dependencies (poppler, tesseract) that should be in their own Docker image
- German tsvector configuration for document full-text search, consistent with email and CRM search patterns
- Service ports follow sequential pattern: :50057 (gRPC) after email :50056, :9097 (health) after email :9096

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added DocumentGRPCPort and DocumentHealthPort to config**
- **Found during:** Task 2 (service scaffold)
- **Issue:** Config struct lacked document service port fields needed by cmd/document/main.go
- **Fix:** Added DocumentGRPCPort (default :50057) and DocumentHealthPort (default :9097) to config.go
- **Files modified:** backend/internal/config/config.go
- **Verification:** Service binary compiles and references correct config fields
- **Committed in:** 30fc566 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** Config extension was necessary for the service scaffold to compile. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Proto contract ready for gRPC server implementation (Plan 11-02)
- Database schema ready for repository layer (Plan 11-02/03)
- Go models ready for service layer business logic (Plan 11-03)
- Service scaffold ready for gRPC service registration (Plan 11-02)

## Self-Check: PASSED

All 9 created files verified on disk. Both task commits (328ca14, 30fc566) verified in git log.

---
*Phase: 11-documents-files-wopi-onlyoffice*
*Completed: 2026-02-17*
