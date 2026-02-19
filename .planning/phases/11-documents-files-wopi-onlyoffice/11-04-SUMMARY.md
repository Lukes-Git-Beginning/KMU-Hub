---
phase: 11-documents-files-wopi-onlyoffice
plan: 04
subsystem: api
tags: [grpc, wopi, onlyoffice, gateway, http, docker, global-search]

# Dependency graph
requires:
  - phase: 11-02
    provides: "Document service packages (folder, file, share, tag)"
  - phase: 11-03
    provides: "Search, virtual files, and WOPI database migration"
provides:
  - "DocumentGRPCServer implementing all 34 RPCs"
  - "WOPI protocol endpoints (CheckFileInfo, GetFile, PutFile, Lock)"
  - "~30 HTTP gateway endpoints for document CRUD"
  - "Global search endpoint with parallel fan-out"
  - "Docker Compose for document service and OnlyOffice"
affects: [frontend-documents, phase-12-finanzen, phase-14-unified-inbox]

# Tech tracking
tech-stack:
  added: [onlyoffice-documentserver]
  patterns: [wopi-protocol, global-search-fanout, grpc-to-wopi-adapter]

key-files:
  created:
    - backend/internal/server/document_grpc.go
    - backend/internal/document/wopi/token.go
    - backend/internal/document/wopi/lock.go
    - backend/internal/document/wopi/handler.go
    - backend/internal/gateway/route_document.go
    - backend/internal/gateway/route_wopi.go
    - backend/internal/gateway/route_search_global.go
    - backend/internal/gateway/wopi_adapter.go
    - backend/Dockerfile.document
    - backend/proto/document/v1/document.pb.go
    - backend/proto/document/v1/document_grpc.pb.go
  modified:
    - backend/cmd/document/main.go
    - backend/cmd/gateway/main.go
    - backend/internal/config/config.go
    - deploy/docker/docker-compose.yml

key-decisions:
  - "WOPI handler runs in gateway with gRPC-backed file adapter, not directly in document service"
  - "Global search fans out to CRM and documents only; email returns empty gracefully until search RPC exists"
  - "OnlyOffice runs with JWT_ENABLED=false for dev; production will enable JWT"
  - "WOPI routes registered at root level (/wopi/files/) per WOPI spec, not under /api/v1/"

patterns-established:
  - "WOPI FileServiceInterface adapter: bridges gRPC client to local interface for WOPI handler"
  - "Global search fan-out: parallel goroutines with 500ms timeout, graceful degradation per module"
  - "Root-level routes for protocol endpoints (WOPI) that don't use standard auth middleware"

requirements-completed: [DOC-01, DOC-02, DOC-03, DOC-04, DOC-05, DOC-06, DOC-07, DOC-08, DOC-09, DOC-10]

# Metrics
duration: 25min
completed: 2026-02-17
---

# Phase 11 Plan 04: Connection Layer Summary

**Document gRPC server (34 RPCs), WOPI protocol endpoints with separate JWT, ~30 gateway HTTP routes, global search fan-out, and Docker Compose with OnlyOffice**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-02-17T18:00:00Z
- **Completed:** 2026-02-17T18:25:00Z
- **Tasks:** 2
- **Files modified:** 15

## Accomplishments
- Full DocumentGRPCServer with all 34 RPCs and comprehensive error mapping to gRPC status codes
- WOPI package with separate JWT tokens (10-hour TTL), PostgreSQL-backed locks (30-min expiry), and HTTP handlers
- Gateway routes exposing ~30 HTTP endpoints for folder, file, share, tag, search, virtual, WOPI token
- WOPI protocol routes at root level (/wopi/files/) for OnlyOffice integration
- Global search endpoint with parallel fan-out to CRM and documents (500ms timeout)
- Docker Compose additions: document service (with docconv deps) and OnlyOffice Document Server

## Task Commits

Each task was committed atomically:

1. **Task 1: gRPC server + WOPI package + document service wiring** - `ff9ca89` (feat)
2. **Task 2: Gateway routes + global search + Docker Compose + gateway wiring** - `a8798fc` (feat)

## Files Created/Modified
- `backend/internal/server/document_grpc.go` - Full gRPC server with 34 RPCs, proto conversion helpers, error mapping
- `backend/internal/document/wopi/token.go` - WOPI JWT token service (HS256, 10-hour TTL, separate secret)
- `backend/internal/document/wopi/lock.go` - PostgreSQL lock service (30-min expiry, INSERT ON CONFLICT)
- `backend/internal/document/wopi/handler.go` - WOPI HTTP handler (CheckFileInfo, GetFile, PutFile, Lock operations)
- `backend/internal/gateway/route_document.go` - ~30 HTTP routes for document CRUD, shares, tags, search, virtual
- `backend/internal/gateway/route_wopi.go` - 4 WOPI protocol routes at root level
- `backend/internal/gateway/route_search_global.go` - Global search with parallel fan-out
- `backend/internal/gateway/wopi_adapter.go` - Adapter bridging gRPC client to wopi.FileServiceInterface
- `backend/cmd/document/main.go` - Wired all services, repos, WOPI lock cleanup goroutine
- `backend/cmd/gateway/main.go` - Registered document service, WOPI handler, global search
- `backend/internal/config/config.go` - Added WOPIJWTSecret env var
- `backend/Dockerfile.document` - Multi-stage with docconv runtime deps (poppler-utils, wv, antiword, unrtf)
- `deploy/docker/docker-compose.yml` - Added document service and OnlyOffice with volumes
- `backend/proto/document/v1/document.pb.go` - Generated protobuf Go code
- `backend/proto/document/v1/document_grpc.pb.go` - Generated gRPC Go code

## Decisions Made
- WOPI handler is instantiated in the gateway (not the document microservice) because OnlyOffice calls back to the gateway URL. A WOPIFileAdapter bridges the document gRPC client to the wopi.FileServiceInterface.
- Global search only fans out to CRM and documents for now. Email returns empty results gracefully (no SearchMessages RPC exists yet). This is documented as future work for Phase 14 Unified Inbox.
- OnlyOffice runs with JWT disabled in dev mode for simplicity. Production will enable JWT with a separate ONLYOFFICE_JWT_SECRET.
- WOPI protocol routes are registered at router root level (/wopi/files/) per WOPI specification, outside the standard RouteRegistrar loop, and without standard auth middleware.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Missing models import in document_grpc.go**
- **Found during:** Task 1 (gRPC server implementation)
- **Issue:** Proto conversion functions referenced *models.DocumentFolder etc. but models package was not imported
- **Fix:** Added `"github.com/kmuhub/kmuhub/internal/models"` to import block
- **Files modified:** backend/internal/server/document_grpc.go
- **Verification:** `go build ./internal/server/` compiles
- **Committed in:** ff9ca89 (Task 1 commit)

**2. [Rule 1 - Bug] createTagRequest name collision in gateway**
- **Found during:** Task 2 (gateway routes)
- **Issue:** CRM routes already define createTagRequest; document routes reused the same name
- **Fix:** Renamed to createDocumentTagRequest in route_document.go
- **Files modified:** backend/internal/gateway/route_document.go
- **Verification:** `go build ./internal/gateway/` compiles
- **Committed in:** a8798fc (Task 2 commit)

**3. [Rule 1 - Bug] Wrong proto enum name for FileSortField unspecified**
- **Found during:** Task 2 (gateway routes)
- **Issue:** Used FILE_SORT_FIELD_UNSPECIFIED but proto defines FILE_SORT_UNSPECIFIED
- **Fix:** Corrected to FileSortField_FILE_SORT_UNSPECIFIED
- **Files modified:** backend/internal/gateway/route_document.go
- **Verification:** `go build ./internal/gateway/` compiles
- **Committed in:** a8798fc (Task 2 commit)

**4. [Rule 3 - Blocking] Email search RPC does not exist**
- **Found during:** Task 2 (global search implementation)
- **Issue:** Plan referenced emailv1.SearchMessages but email proto has no such RPC
- **Fix:** Replaced with stub that returns empty results immediately (graceful fallback)
- **Files modified:** backend/internal/gateway/route_search_global.go
- **Verification:** `go build ./cmd/gateway/` compiles
- **Committed in:** a8798fc (Task 2 commit)

---

**Total deviations:** 4 auto-fixed (3 bugs, 1 blocking)
**Impact on plan:** All auto-fixes necessary for compilation. No scope creep.

## Issues Encountered
None beyond the auto-fixed deviations listed above.

## User Setup Required
None - no external service configuration required. Docker Compose handles all service orchestration.

## Next Phase Readiness
- Phase 11 backend is complete: proto, migrations, repos, services, gRPC server, gateway routes, WOPI, Docker
- Frontend document module can now call all ~30 HTTP endpoints
- OnlyOffice collaborative editing is wired and ready for frontend iframe integration
- Global search endpoint ready for the search UI component

## Self-Check: PASSED

All 11 created files verified present. Both task commits (ff9ca89, a8798fc) found in git log.

---
*Phase: 11-documents-files-wopi-onlyoffice*
*Completed: 2026-02-17*
