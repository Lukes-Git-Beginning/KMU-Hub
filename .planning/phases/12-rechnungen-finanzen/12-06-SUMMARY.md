---
phase: 12-rechnungen-finanzen
plan: 06
subsystem: api
tags: [grpc, pdf, maroto, protobuf, gateway, streaming]

# Dependency graph
requires:
  - phase: 12-rechnungen-finanzen
    provides: "biz gRPC service, pdf.Generator, gateway route_biz.go with 501 PDF stubs"
provides:
  - "4 PDF generation gRPC RPCs (GenerateQuotePDF, GenerateInvoicePDF, GenerateCreditNotePDF, GenerateDunningPDF)"
  - "Gateway HTTP-to-gRPC PDF binary streaming for all 4 document types"
  - "dunning.Service.GetByID method for individual dunning record lookup"
affects: [12-rechnungen-finanzen, frontend-finance]

# Tech tracking
tech-stack:
  added: []
  patterns: [respondPDF helper for binary gRPC-to-HTTP streaming, fresh pdf.Generator per request for latest company settings]

key-files:
  created: []
  modified:
    - backend/proto/biz/v1/biz.proto
    - backend/proto/biz/v1/biz.pb.go
    - backend/proto/biz/v1/biz_grpc.pb.go
    - backend/internal/server/biz_grpc.go
    - backend/internal/gateway/route_biz.go
    - backend/internal/biz/dunning/service.go

key-decisions:
  - "Fresh pdf.Generator per request with latest company settings from DB (not reusing startup instance)"
  - "respondPDF helper consolidates Content-Type/Disposition/Length headers for all 4 handlers"
  - "Dunning PDF filename varies by level: Zahlungserinnerung (level 1), 1_Mahnung (level 2), 2_Mahnung (level 3)"

patterns-established:
  - "respondPDF: reusable gateway helper for streaming gRPC PDF bytes as HTTP attachment"
  - "Per-request Generator: create fresh pdf.Generator from DB settings to reflect real-time company changes"

requirements-completed: [FIN-01, FIN-02]

# Metrics
duration: 7min
completed: 2026-02-18
---

# Phase 12 Plan 06: PDF Binary Streaming Summary

**Wire gRPC PDF generation RPCs through gateway to HTTP, replacing 4 x 501 stubs with real PDF binary downloads**

## Performance

- **Duration:** 7 min
- **Started:** 2026-02-18T14:07:00Z
- **Completed:** 2026-02-18T14:14:00Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- All 4 PDF endpoints now return real application/pdf binary data instead of HTTP 501
- Frontend requestBlob hooks can now download Angebot, Rechnung, Gutschrift, and Mahnung PDFs
- German filenames (Angebot_, Rechnung_, Gutschrift_, Zahlungserinnerung_/Mahnung_) for proper download naming

## Task Commits

Each task was committed atomically:

1. **Task 1: Add PDF RPCs to proto and implement in gRPC server** - `3688ba7` (feat)
2. **Task 2: Wire gateway PDF handlers to stream gRPC PDF bytes** - `020131d` (feat)

## Files Created/Modified
- `backend/proto/biz/v1/biz.proto` - Added 4 PDF RPCs and 8 request/response messages to FinanceService
- `backend/proto/biz/v1/biz.pb.go` - Regenerated protobuf Go code
- `backend/proto/biz/v1/biz_grpc.pb.go` - Regenerated gRPC Go code with PDF client/server interfaces
- `backend/internal/server/biz_grpc.go` - Implemented 4 GeneratePDF methods: fetch doc + settings + generate + return bytes
- `backend/internal/gateway/route_biz.go` - Rewrote 4 PDF handlers to call gRPC RPCs, added respondPDF helper, deleted respondPDFNotImplemented
- `backend/internal/biz/dunning/service.go` - Added GetByID method (was missing, needed for dunning PDF lookup)

## Decisions Made
- Fresh pdf.Generator per request ensures company settings changes (logo, address) take effect immediately without service restart
- respondPDF helper avoids 4x duplication of Content-Type/Disposition/Length header logic
- Dunning PDF filename uses German terminology matching the dunning level (Zahlungserinnerung vs Mahnung)
- Added dunning.Service.GetByID (was only on repository) since the gRPC server needs to fetch individual dunning records

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added dunning.Service.GetByID method**
- **Found during:** Task 1 (gRPC server implementation)
- **Issue:** dunning service had no GetByID -- only repository exposed it. The gRPC server calls service methods, not repositories directly.
- **Fix:** Added GetByID to dunning.Service as a passthrough to repo.GetByID
- **Files modified:** backend/internal/biz/dunning/service.go
- **Verification:** Build compiles, GenerateDunningPDF method works
- **Committed in:** 3688ba7 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Essential for dunning PDF generation. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All PDF download functionality complete for Phase 12 finance module
- Plan 12-07 (gap closure) can proceed
- Frontend requestBlob hooks are ready to consume the new PDF binary endpoints

## Self-Check: PASSED

- All 6 modified files verified on disk
- Commits 3688ba7 (Task 1) and 020131d (Task 2) verified in git log
- Backend builds cleanly (go build ./...)
- respondPDFNotImplemented: 0 occurrences (deleted)
- application/pdf: present in gateway respondPDF helper
- 4 GeneratePDF RPC definitions in biz.proto

---
*Phase: 12-rechnungen-finanzen*
*Completed: 2026-02-18*
