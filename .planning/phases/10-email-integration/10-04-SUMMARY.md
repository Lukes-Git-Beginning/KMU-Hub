---
phase: 10-email-integration
plan: 04
subsystem: api, database
tags: [grpc, protobuf, imap, smtp, email, go-imap, go-smtp, go-message, go-vcard, jwz, gocsv, postgresql, migrations]

# Dependency graph
requires:
  - phase: 09-security-compliance
    provides: vault encryption (VaultEncryptor), auth infrastructure, config patterns
  - phase: 02-crm-core
    provides: contacts table, CRM models, crm.proto pattern
provides:
  - email.v1.EmailService gRPC definition with 39 RPCs
  - Database schema for email accounts, folders, messages, contact links, attachments, signatures
  - Contact visibility model (shared/personal) via migration 000042
  - Go models for all email entities
  - Email service binary scaffold on gRPC port :50056
  - Domain error packages for 7 email sub-packages
  - Go dependencies for email protocol handling (go-imap v2, go-smtp, go-message, go-vcard, jwz, gocsv, go-sasl)
affects: [10-05, 10-06, 10-07, gateway, crm]

# Tech tracking
tech-stack:
  added: [go-imap/v2 v2.0.0-beta.8, go-smtp v0.24.0, go-message v0.18.2, go-vcard, jwz v1.4.0, gocsv, go-sasl]
  patterns: [email service microservice pattern, email proto definition, email domain error packages]

key-files:
  created:
    - backend/proto/email/v1/email.proto
    - backend/proto/email/v1/email.pb.go
    - backend/proto/email/v1/email_grpc.pb.go
    - backend/migrations/000041_create_email_tables.up.sql
    - backend/migrations/000041_create_email_tables.down.sql
    - backend/migrations/000042_add_contact_visibility.up.sql
    - backend/migrations/000042_add_contact_visibility.down.sql
    - backend/internal/models/email.go
    - backend/cmd/email/main.go
    - backend/tools/email_deps.go
    - backend/internal/email/account/errors.go
    - backend/internal/email/sync/errors.go
    - backend/internal/email/message/errors.go
    - backend/internal/email/send/errors.go
    - backend/internal/email/signature/errors.go
    - backend/internal/email/attachment/errors.go
    - backend/internal/email/contact/errors.go
  modified:
    - backend/internal/models/contact.go
    - backend/internal/config/config.go
    - backend/go.mod
    - backend/go.sum

key-decisions:
  - "39 RPCs in EmailService covering accounts, folders, messages, send/compose, signatures, CRM linking, sync, attachments, and contact import/export"
  - "Email service on gRPC port :50056, health/metrics on :9096"
  - "tools/email_deps.go (separate from internal/tools/) to retain email Go dependencies before service code imports them"
  - "search_vector tsvector with German config and GIN index on email_messages for full-text search"
  - "Contact visibility CHECK constraint (shared, personal) with owner_id FK to users"

patterns-established:
  - "Email sub-package error files: one errors.go per domain package (account, sync, message, send, signature, attachment, contact)"
  - "Email service scaffold: same pattern as cmd/crm/main.go with stub UnimplementedEmailServiceServer"

# Metrics
duration: 8min
completed: 2026-02-16
---

# Phase 10 Plan 04: Email Service Data Foundation Summary

**email.v1 proto with 39 RPCs, 6-table DB schema with tsvector search, Go models, and email service binary scaffold on :50056**

## Performance

- **Duration:** 8 min
- **Started:** 2026-02-16T15:25:58Z
- **Completed:** 2026-02-16T15:33:54Z
- **Tasks:** 2
- **Files modified:** 21

## Accomplishments
- email.proto defines EmailService with 39 RPCs covering all email domains: accounts (5), folders (3), messages (8), send/compose (4), signatures (6), CRM linking (4), sync (3), attachments (2), import/export (4)
- Migration 000041 creates 6 email tables with full-text search (tsvector + GIN index + auto-update trigger) and 10+ indexes
- Migration 000042 adds shared/personal visibility model to contacts table with CHECK constraint
- Go models mirror all DB tables with proper types, JSON tags, and helper methods
- Email service binary compiles and starts as a gRPC server with health/metrics endpoints
- 7 Go dependencies installed for email protocol handling (IMAP, SMTP, MIME, vCard, threading, CSV)
- 7 domain error packages established for email sub-packages

## Task Commits

Each task was committed atomically:

1. **Task 1: Email proto definition + Go dependency installation** - `1923813` (feat)
2. **Task 2: Database migrations + Go models + email service scaffold** - `89e5389` (feat)

## Files Created/Modified
- `backend/proto/email/v1/email.proto` - EmailService gRPC definition with 39 RPCs
- `backend/proto/email/v1/email.pb.go` - Generated protobuf Go code
- `backend/proto/email/v1/email_grpc.pb.go` - Generated gRPC Go code
- `backend/tools/email_deps.go` - Build-tagged dependency retention for email libraries
- `backend/migrations/000041_create_email_tables.up.sql` - 6 email tables + tsvector search
- `backend/migrations/000041_create_email_tables.down.sql` - Reverse migration
- `backend/migrations/000042_add_contact_visibility.up.sql` - Contact visibility columns
- `backend/migrations/000042_add_contact_visibility.down.sql` - Reverse migration
- `backend/internal/models/email.go` - Go structs for all email entities
- `backend/internal/models/contact.go` - Added Visibility and OwnerID fields
- `backend/internal/config/config.go` - Added EmailGRPCPort, EmailGRPCAddress, EmailHealthPort
- `backend/cmd/email/main.go` - Email service entry point with stub gRPC server
- `backend/internal/email/account/errors.go` - ErrAccountNotFound, ErrAccountExists, ErrInvalidCredentials, ErrConnectionFailed
- `backend/internal/email/sync/errors.go` - ErrSyncInProgress, ErrIMAPConnectionLost, ErrUIDValidityChanged
- `backend/internal/email/message/errors.go` - ErrMessageNotFound, ErrFolderNotFound, ErrThreadNotFound
- `backend/internal/email/send/errors.go` - ErrSendFailed, ErrSMTPAuthFailed, ErrInvalidRecipient
- `backend/internal/email/signature/errors.go` - ErrSignatureNotFound
- `backend/internal/email/attachment/errors.go` - ErrAttachmentNotFound, ErrAttachmentTooLarge
- `backend/internal/email/contact/errors.go` - ErrImportFailed, ErrInvalidCSV, ErrInvalidVCard
- `backend/go.mod` - 7 new email dependencies added
- `backend/go.sum` - Updated checksums

## Decisions Made
- 39 RPCs (vs ~35 planned): added SetDefaultSignature as separate RPC and split import/export by format for cleaner API
- tools/email_deps.go placed in backend/tools/ (new directory) rather than backend/internal/tools/ to separate email deps from security deps
- search_vector uses German text search configuration (consistent with CRM contact search pattern)
- Contact visibility uses CHECK constraint (shared, personal) rather than enum type for migration simplicity
- Email service binary uses stub UnimplementedEmailServiceServer -- all RPCs return Unimplemented until 10-05 wires real services

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Proto, migrations, models, and service scaffold ready for 10-05 (account management + IMAP sync engine)
- All 7 Go email dependencies installed and compilable
- Email sub-package directories created with error definitions for each domain
- Config extended with email service ports for gateway integration in 10-07

## Self-Check: PASSED

- All 17 created files verified on disk
- Both task commits (1923813, 89e5389) verified in git log
- Full backend build (`go build ./...`) passes
- Go vet (`go vet ./...`) passes

---
*Phase: 10-email-integration*
*Completed: 2026-02-16*
