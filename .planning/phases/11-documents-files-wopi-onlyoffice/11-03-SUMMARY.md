---
phase: 11-documents-files-wopi-onlyoffice
plan: 03
subsystem: api, database
tags: [go, postgresql, tsvector, docconv, text-extraction, tagging, virtual-folders, full-text-search]

requires:
  - phase: 11-documents-files-wopi-onlyoffice
    plan: 01
    provides: "Document tables (document_files, document_tags, document_file_tags), Go domain models, docconv dependency"
provides:
  - "Text extractor wrapping docconv for 12 MIME types"
  - "Document search service with PostgreSQL tsvector ranking and ts_headline snippets"
  - "Tag CRUD service with idempotent file-tag associations"
  - "Virtual folder service querying chat_files, email_attachments, task_files with user ACL"
affects: [11-04, 11-05, 11-06]

tech-stack:
  added: []
  patterns: [async-extract-and-index, virtual-folder-union-query, idempotent-tagging]

key-files:
  created:
    - backend/internal/document/search/extractor.go
    - backend/internal/document/search/service.go
    - backend/internal/document/search/repository.go
    - backend/internal/document/search/postgres_repository.go
    - backend/internal/document/search/errors.go
    - backend/internal/document/tag/service.go
    - backend/internal/document/tag/repository.go
    - backend/internal/document/tag/postgres_repository.go
    - backend/internal/document/tag/errors.go
    - backend/internal/document/virtual/service.go
    - backend/internal/document/virtual/repository.go
    - backend/internal/document/virtual/postgres_repository.go
  modified: []

key-decisions:
  - "FileRepo interface in search package avoids circular import with file package (gRPC server bridges them)"
  - "Virtual folder ListAll uses UNION ALL with per-source delegation for filtered requests"
  - "Extractor returns empty string on error rather than failing (search just won't find content for that file)"

patterns-established:
  - "Async ExtractAndIndex pattern: called via goroutine post-upload, decoupled from upload response"
  - "Virtual folder UNION ALL across chat_files/email_attachments/task_files with user ACL JOINs"
  - "Idempotent tagging via ON CONFLICT DO NOTHING on composite PK"

requirements-completed: [DOC-06, DOC-07, DOC-10]

duration: 5min
completed: 2026-02-17
---

# Phase 11 Plan 03: Search, Tag, and Virtual Folder Services Summary

**Docconv text extractor for 12 MIME types, tsvector search with German config and ts_headline snippets, tag CRUD with idempotent file associations, and virtual folder service surfacing chat/email/task attachments**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-17T16:42:58Z
- **Completed:** 2026-02-17T16:47:35Z
- **Tasks:** 2
- **Files modified:** 12

## Accomplishments
- Text extractor wrapping docconv for 12 MIME types (PDF, DOCX, XLSX, PPTX, TXT, CSV, HTML, RTF, Markdown, MSWord, MS Excel, MS PowerPoint) with 1MB index truncation
- Document search service using PostgreSQL tsvector with ts_rank_cd(normalization 32) scoring and ts_headline snippets, with filters for folder, owner, and tags
- Tag service with CRUD, hex color validation, and idempotent file-tag associations via ON CONFLICT DO NOTHING
- Virtual folder service querying 3 existing attachment tables (chat_files, email_attachments, task_files) with proper user ACL via channel_memberships, email_accounts ownership, and project_members/assignee checks

## Task Commits

Each task was committed atomically:

1. **Task 1: Text extractor + search service** - `4b03476` (feat)
2. **Task 2: Tag service + virtual folder service** - `ac3333c` (feat)

## Files Created/Modified
- `backend/internal/document/search/errors.go` - ErrSearchQueryRequired, ErrExtractionFailed, ErrUnsupportedFormat
- `backend/internal/document/search/extractor.go` - Docconv wrapper with CanExtract/Extract for 12 MIME types
- `backend/internal/document/search/repository.go` - Repository interface, SearchFilter, SearchResult, FileRepo interface
- `backend/internal/document/search/postgres_repository.go` - tsvector search with ts_rank_cd and ts_headline
- `backend/internal/document/search/service.go` - Search + async ExtractAndIndex methods
- `backend/internal/document/tag/errors.go` - ErrTagNotFound, ErrTagNameRequired, ErrTagNameTooLong, ErrTagNameConflict, ErrInvalidColor, ErrTagAlreadyAssigned
- `backend/internal/document/tag/repository.go` - Tag Repository interface with 8 methods
- `backend/internal/document/tag/postgres_repository.go` - Full implementation with file_count subqueries and idempotent tagging
- `backend/internal/document/tag/service.go` - CreateTag, ListTags, DeleteTag, TagFile, UntagFile, ListFileTags
- `backend/internal/document/virtual/repository.go` - Virtual Repository interface for chat/email/task files
- `backend/internal/document/virtual/postgres_repository.go` - UNION ALL queries with user ACL JOINs
- `backend/internal/document/virtual/service.go` - ListVirtualFiles with optional sourceType filtering

## Decisions Made
- FileRepo interface defined in search package (not importing file package) to avoid circular dependency; gRPC server will bridge the two packages at runtime
- Virtual folder ListAll delegates to individual source methods when sourceType is specified, only uses UNION ALL when all sources are requested
- Extractor returns empty string on extraction error rather than failing the indexing process (search gracefully degrades)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed color validation error type in tag service**
- **Found during:** Task 2 (tag service)
- **Issue:** Color validation was returning ErrTagNameConflict instead of a proper error
- **Fix:** Added ErrInvalidColor to errors.go and used it in CreateTag
- **Files modified:** backend/internal/document/tag/errors.go, backend/internal/document/tag/service.go
- **Verification:** Package compiles, correct error returned for invalid color
- **Committed in:** ac3333c (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Minor error type correction for correctness. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Search, tag, and virtual packages ready for gRPC server integration (Plan 11-04)
- ExtractAndIndex ready to be called asynchronously from file upload handler
- Virtual folder service ready for frontend integration (Plan 11-05)
- All 3 packages compile cleanly alongside existing document packages

## Self-Check: PASSED

All 12 created files verified on disk. Both task commits (4b03476, ac3333c) verified in git log.

---
*Phase: 11-documents-files-wopi-onlyoffice*
*Completed: 2026-02-17*
