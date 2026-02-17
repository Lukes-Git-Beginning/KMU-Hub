---
phase: 11-documents-files-wopi-onlyoffice
plan: 02
subsystem: api, database
tags: [go, postgresql, minio, document-management, file-versioning, sharing, entity-linking]

requires:
  - phase: 11-documents-files-wopi-onlyoffice
    plan: 01
    provides: "Document tables, Go domain models, service scaffold, chatfile.FileStore interface"
provides:
  - "Folder service with CRUD, tree navigation, and space initialization for personal/team spaces"
  - "File service with upload, versioning (create/revert), move, copy, soft-delete, presigned URLs"
  - "Share service with read/write permission grants and access checking"
  - "Entity link operations for CRM/PM integration (contact, company, deal, project, task)"
affects: [11-03, 11-04, 11-05, 11-06]

tech-stack:
  added: []
  patterns: [document-folder-service-pattern, file-version-revert-as-new, share-access-check-pattern, chatfile-filestore-reuse]

key-files:
  created:
    - backend/internal/document/folder/service.go
    - backend/internal/document/folder/repository.go
    - backend/internal/document/folder/postgres_repository.go
    - backend/internal/document/folder/errors.go
    - backend/internal/document/file/service.go
    - backend/internal/document/file/repository.go
    - backend/internal/document/file/postgres_repository.go
    - backend/internal/document/file/errors.go
    - backend/internal/document/share/service.go
    - backend/internal/document/share/repository.go
    - backend/internal/document/share/postgres_repository.go
    - backend/internal/document/share/errors.go
  modified: []

key-decisions:
  - "File service reuses chatfile.FileStore interface from chat package (no new MinIO client)"
  - "Version revert creates a new version with old content (append-only version history)"
  - "File move updates folder_id only, MinIO storage key unchanged (reference by file ID)"
  - "DACH default folders: personal=Dokumente/Bilder/Vorlagen, team=Allgemein/Projekte/Vorlagen"

patterns-established:
  - "Document sub-package pattern: folder/, file/, share/ under internal/document/ with repository interface + PostgreSQL implementation + service + errors"
  - "Recursive CTE for folder path breadcrumbs and circular-move prevention"
  - "Share access checking via HasAccess(entityType, entityID, userID) returning (bool, permission, error)"
  - "Entity link with AllowedEntityTypes map validation for extensible CRM/PM linking"

requirements-completed: [DOC-01, DOC-02, DOC-04, DOC-05, DOC-08]

duration: 5min
completed: 2026-02-17
---

# Phase 11 Plan 02: Document Service Core Business Logic Summary

**Folder CRUD with recursive tree navigation and DACH space defaults, file versioning with MinIO integration via chatfile.FileStore reuse, and share service with permission-level access checking**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-17T16:42:56Z
- **Completed:** 2026-02-17T16:47:36Z
- **Tasks:** 2
- **Files modified:** 12

## Accomplishments
- Folder service with 8 methods: CRUD, tree path (recursive CTE), space initialization for personal and team with German default folders
- File service with 14 methods: upload, versioning (create + revert-as-new), move, copy, soft-delete, presigned download URLs, entity linking to CRM/PM entities
- Share service with 5 methods: share/unshare entities, list shares, check access with permission level
- All 3 packages follow repository pattern with PostgreSQL implementations using pgx connection pool

## Task Commits

Each task was committed atomically:

1. **Task 1: Folder service + space initialization** - `81de8d5` (feat)
2. **Task 2: File service + share service + entity link** - `e73d1ee` (feat)

## Files Created/Modified
- `backend/internal/document/folder/errors.go` - 9 sentinel errors for folder operations
- `backend/internal/document/folder/repository.go` - Repository interface with 9 methods + ListFilter/UpdateInput types
- `backend/internal/document/folder/postgres_repository.go` - PostgreSQL implementation with recursive CTEs for path and descendant checking
- `backend/internal/document/folder/service.go` - Service with CRUD, validation, circular-move prevention, InitializeUserSpace/InitializeTeamSpace
- `backend/internal/document/file/errors.go` - 10 sentinel errors + AllowedEntityTypes map for entity linking
- `backend/internal/document/file/repository.go` - Repository interface with 15 methods covering files, versions, search, entity links
- `backend/internal/document/file/postgres_repository.go` - PostgreSQL implementation with dynamic query building, tag filtering, version management
- `backend/internal/document/file/service.go` - Service with 14 methods using chatfile.FileStore for MinIO operations
- `backend/internal/document/share/errors.go` - 5 sentinel errors for share operations
- `backend/internal/document/share/repository.go` - Repository interface with 6 methods including HasAccess
- `backend/internal/document/share/postgres_repository.go` - PostgreSQL implementation with user name JOINs for denormalized display
- `backend/internal/document/share/service.go` - Service with 5 methods including CheckAccess for permission verification

## Decisions Made
- File service reuses `chatfile.FileStore` interface from the chat package instead of creating a new MinIO client, maintaining a single abstraction for object storage operations across services
- Version revert creates a new latest version containing the old content (append-only history, never deletes versions)
- File move only updates `folder_id` in the database; the MinIO storage key remains unchanged since files are referenced by UUID, not path
- DACH-appropriate default folders: personal space gets "Dokumente", "Bilder", "Vorlagen"; team space gets "Allgemein", "Projekte", "Vorlagen"

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Service packages ready for gRPC server implementation (Plan 11-03 or 11-04)
- Repository interfaces ready for unit testing with mock implementations
- Share service CheckAccess method available for file/folder operations to validate permissions
- Entity link supports 5 entity types (contact, company, deal, project, task) for CRM/PM integration

## Self-Check: PASSED

All 12 created files verified on disk. Both task commits (81de8d5, e73d1ee) verified in git log.

---
*Phase: 11-documents-files-wopi-onlyoffice*
*Completed: 2026-02-17*
