---
phase: 10-email-integration
plan: 06
subsystem: grpc, gateway, frontend, docker
tags: [email, grpc, gateway, tanstack-query, compose, imap, smtp, crm-link, docker]

# Dependency graph
requires:
  - phase: 10-email-integration
    provides: email.proto (39 RPCs), all email domain services (account, message, send, signature, attachment, sync, contact)
  - phase: 09-security-compliance
    provides: vault encryption for IMAP/SMTP credentials
provides:
  - EmailGRPCServer implementing all 39 RPCs with thin-handler pattern
  - EmailRoutes gateway registrar with ~40 HTTP endpoints under /api/v1/email/
  - Dockerfile.email + docker-compose entry (gRPC :50056, health :9096)
  - Full cmd/email/main.go wiring with adapter pattern for interface mismatches
  - Frontend email-types.ts mirroring proto messages
  - Frontend email-client.ts fetch wrapper with auth middleware
  - TanStack Query hooks (30+) for all email operations
  - Rewired stores/mails.ts to ephemeral compose-only state
  - MailsPage with three-column layout using TanStack Query data
  - ComposeInline, ComposeModal, ComposeWindowPage using API mutations
  - CRM link badges on email detail view
affects: [phase-10-verification, frontend-email-module, docker-compose]

# Tech tracking
tech-stack:
  added: []
  patterns: [adapter pattern for cross-package interface bridging, TanStack Query hooks with query key factory, typed fetch wrapper with auth middleware, ephemeral Zustand store for cross-window compose state]

key-files:
  created:
    - backend/internal/server/email_grpc.go
    - backend/internal/email/contactlink/repository.go
    - backend/internal/gateway/route_email.go
    - backend/Dockerfile.email
    - desktop/src/renderer/src/api/email-types.ts
    - desktop/src/renderer/src/api/email-client.ts
    - desktop/src/renderer/src/api/hooks/useEmail.ts
  modified:
    - backend/cmd/email/main.go
    - backend/cmd/gateway/main.go
    - deploy/docker/docker-compose.yml
    - desktop/src/renderer/src/stores/mails.ts
    - desktop/src/renderer/src/modules/mails/MailsPage.tsx
    - desktop/src/renderer/src/modules/mails/ComposeInline.tsx
    - desktop/src/renderer/src/modules/mails/ComposeModal.tsx
    - desktop/src/renderer/src/modules/mails/ComposeWindowPage.tsx
---

# Plan 10-06 Summary: gRPC Server, Gateway, Docker & Frontend Email Client

## What was built

### Backend (Task 1)

**EmailGRPCServer** (`backend/internal/server/email_grpc.go`)
- Implements all 39 RPCs from email.proto using thin-handler pattern
- Delegates to domain services: account, message, send, signature, attachment, sync, contactlink, import/export
- `mapEmailError()` maps domain errors to gRPC status codes
- Conversion helpers for all proto message types

**ContactLink Repository** (`backend/internal/email/contactlink/repository.go`)
- PostgreSQL repository for email-to-CRM contact linking
- Idempotent `Create` with `ON CONFLICT DO NOTHING`
- `GetByContactID` with JOIN to email_messages for pagination

**Gateway Routes** (`backend/internal/gateway/route_email.go`)
- ~40 HTTP endpoints organized by domain (accounts, folders, messages, send, signatures, links, sync, attachments, contacts)
- Multipart file upload handler for attachments (50MB limit)
- Permission-based access control via middleware

**Docker** (`backend/Dockerfile.email`, `deploy/docker/docker-compose.yml`)
- Multi-stage build (golang:1.25-alpine -> alpine:3.20)
- Email service at gRPC :50056, health :9096
- Gateway depends_on email service with env vars

**cmd/email/main.go** — full service wiring with three adapter types:
- `attachSyncAdapter`: bridges `io.Reader` vs `interface{}` for attachment sync
- `folderSyncAdapter`: bridges `Create` vs `CreateFolder` method names
- `sendAccountAdapter`: bridges `account.Credentials` vs `send.Credentials` types
- `emailVaultAdapter`: wraps HKDF-derived AES-256 key for credential encryption
- `noopEncryptor`: plaintext fallback for dev mode

### Frontend (Tasks 2 & 3)

**Types** (`desktop/src/renderer/src/api/email-types.ts`)
- Full TypeScript types mirroring proto messages (snake_case)
- Covers accounts, folders, messages, send/compose, signatures, CRM links, sync, attachments

**API Client** (`desktop/src/renderer/src/api/email-client.ts`)
- Typed fetch wrapper with auth token injection and 401 refresh
- Offline mutation guard
- Organized by domain: emailAccountApi, emailFolderApi, emailMessageApi, emailSendApi, emailSignatureApi, emailLinkApi, emailSyncApi, emailAttachmentApi, emailContactApi

**TanStack Query Hooks** (`desktop/src/renderer/src/api/hooks/useEmail.ts`)
- 30+ hooks covering all email operations
- Query key factory (`emailKeys`) for consistent cache invalidation
- Mutations auto-invalidate related queries (messages, folders)
- Sync status polling every 30s

**Store** (`desktop/src/renderer/src/stores/mails.ts`)
- Rewired from 300+ line Zustand mock store to 33 lines
- Only retains `composeDraft` / `setComposeDraft` for Electron pop-out window
- All data now comes from TanStack Query hooks

**UI Components** — All rewired from Zustand mock to TanStack Query:
- `MailsPage.tsx`: Three-column layout with folder sidebar, message list, reading pane. CRM link badges, sync button, loading states
- `ComposeInline.tsx`: Inline compose with reply/forward/send/draft API mutations
- `ComposeModal.tsx`: Dialog-based compose with same API integration
- `ComposeWindowPage.tsx`: Electron pop-out compose window with draft state

## Verification

- `go build ./...` — passes
- `npx tsc --noEmit` — passes
- `docker compose config` — valid
- All email module files compile without errors
