---
phase: 14-event-infrastructure-unified-inbox
verified: 2026-02-20T01:00:00Z
status: passed
score: 5/5 must-haves verified
re_verification: null
gaps: []
human_verification:
  - test: "Navigate to /kommunikation in the desktop app and verify the three-column layout renders correctly"
    expected: "Sidebar on left with Smart Views/Channel filters/Team Inboxes, message list in center, detail pane opens on message click"
    why_human: "Visual layout and React rendering cannot be verified programmatically without running the app"
  - test: "Click a message in the list and use keyboard shortcuts j/k to navigate, e to archive, s to star, r to reply"
    expected: "Keyboard navigation works without requiring mouse; actions reflect immediately via optimistic updates"
    why_human: "Keyboard event handler behavior requires interactive testing"
  - test: "Hover over a message in the list and verify quick action icons appear (reply, snooze, archive, assign, star)"
    expected: "Icons appear on hover, clicking snooze opens SnoozePopover with three preset buttons and custom date/time"
    why_human: "Hover state and popover interaction require interactive testing"
  - test: "Click a team inbox message and verify the 'Ubernehmen' (claim) button appears and works"
    expected: "Claim button visible when message is unassigned; clicking atomically assigns to current user"
    why_human: "Team inbox assignment workflow requires live backend and auth context"
  - test: "Open Routing Rules Editor (admin/manager role), create a rule with AND condition, and use the Test button"
    expected: "Rule test panel shows 'Trifft zu' (green) or 'Trifft nicht zu' (red) correctly"
    why_human: "Admin role check and interactive condition builder require live session"
---

# Phase 14: Event Infrastructure + Unified Inbox Verification Report

**Phase Goal:** All modules emit structured events via PostgreSQL LISTEN/NOTIFY, and users get a single aggregated inbox across Email, Chat, and Notifications -- the foundation for Automation Engine
**Verified:** 2026-02-20T01:00:00Z
**Status:** PASSED
**Re-verification:** No -- initial verification

---

## Goal Achievement

### Observable Truths (from ROADMAP.md Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | All existing services emit structured events via PostgreSQL events table + pg_notify | VERIFIED | PGEventEmitter in 6 packages (email, document, invoice, quote, leave, timetracking); event.EmitEvent() delegates to pg_notify; SetEventEmitter + emit calls wired in all 6 service.go files |
| 2 | User sees a unified inbox aggregating Email, Chat DMs/@mentions, and Notifications in a single view | VERIFIED | KommunikationPage (304 lines) renders three-column layout; /kommunikation route registered in App.tsx; sidebar nav entry with MessageSquareText icon; InboxSidebar shows all three channels |
| 3 | User can reply, mark-read, and triage items from the unified inbox without switching modules | VERIFIED | MessageDetail has channel-adaptive inline reply (email=textarea, chat=input, notification=no-reply text); 25+ TanStack Query mutation hooks with optimistic updates (useMarkRead, useArchiveMessage, useToggleStar, useSnoozeMessage, useReplyToMessage); keyboard shortcuts j/k/e/s/r registered |
| 4 | Team inboxes allow shared mailbox concepts with assignment and routing rules | VERIFIED | TeamInboxSettings component (325 lines) with manual/round-robin radio buttons and member management; ClaimMessage with atomic WHERE assigned_to IS NULL guard; routing engine with 4 action types; 25 HTTP endpoints include team inbox and routing rule CRUD |
| 5 | Channel adapters normalize messages from different sources into a unified format | VERIFIED | ChannelAdapter interface with FetchNewMessages/HandleReply/MarkReadOnSource; EmailAdapter, ChatAdapter, NotificationAdapter all implement interface (compile-time assertion); InboxConsumer on EventBus maps module_id to channel and normalizes event fields into InboxMessage |

**Score: 5/5 truths verified**

---

## Required Artifacts

### Plan 14-01 Artifacts

| Artifact | Status | Lines | Key Evidence |
|----------|--------|-------|--------------|
| `backend/proto/inbox/v1/inbox.proto` | VERIFIED | - | `service InboxService` at line 10; exactly 27 RPCs counted |
| `backend/migrations/000047_create_inbox_tables.up.sql` | VERIFIED | - | 4 CREATE TABLE statements: inbox_messages, team_inboxes, team_inbox_members, routing_rules |
| `backend/internal/models/inbox.go` | VERIFIED | - | `type InboxMessage struct` present |
| `backend/internal/notification/event/types.go` | VERIFIED | - | EventEmailReceived, EventDocumentUploaded, EventInvoiceCreated, EventLeaveRequested, EventInboxItemCreated, ModuleEmail, ModuleDocument constants all present |
| `backend/internal/email/message/event_emitter.go` | VERIFIED | 31 | PGEventEmitter struct; EmitEmailEvent delegates to event.EmitEvent; SetEventEmitter wired in service.go |

### Plan 14-02 Artifacts

| Artifact | Status | Lines | Key Evidence |
|----------|--------|-------|--------------|
| `backend/internal/inbox/adapter/adapter.go` | VERIFIED | 68 | `type ChannelAdapter interface` with Channel/FetchNewMessages/HandleReply/MarkReadOnSource; AdapterRegistry |
| `backend/internal/inbox/message/service.go` | VERIFIED | 234 | `func (s *Service) Create`, Reply, StartSnoozeWorker goroutine (60s polling) |
| `backend/internal/inbox/team/service.go` | VERIFIED | 246 | `func (s *Service) ClaimMessage` with atomic claim; AutoAssignMessage with round-robin retry |
| `backend/internal/inbox/routing/evaluator.go` | VERIFIED | 160 | `func (c *Condition) Evaluate` with AND/OR/leaf recursion and 11 operators |
| `backend/internal/inbox/routing/evaluator_test.go` | VERIFIED | 461 | `func TestEvaluateLeaf_Equals` at line 35; 15+ test functions covering all operators, AND, OR, nested, edge cases |

### Plan 14-03 Artifacts

| Artifact | Status | Lines | Key Evidence |
|----------|--------|-------|--------------|
| `backend/internal/server/inbox_grpc.go` | VERIFIED | 1079 | `type InboxGRPCServer struct` at line 23; ListMessages, ReplyToMessage, ClaimMessage all implemented; delegates to messageService |
| `backend/internal/gateway/route_inbox.go` | VERIFIED | 1212 | `type InboxRoutes struct`; 25 HTTP routes registered; inboxv1.NewInboxServiceClient wired; ServiceName returns "notification" |
| `backend/cmd/notification/main.go` | VERIFIED | 456 | RegisterInboxServiceServer at line 152; InboxConsumer registered on EventBus at line 113; StartSnoozeWorker at line 116; circular loop prevention (skip module_id="inbox") |

### Plan 14-04 Artifacts

| Artifact | Status | Lines | Key Evidence |
|----------|--------|-------|--------------|
| `desktop/src/renderer/src/api/inbox-types.ts` | VERIFIED | 134 | `export interface InboxMessage` at line 18; TeamInbox, RoutingRule, Condition types exported |
| `desktop/src/renderer/src/api/hooks/useInbox.ts` | VERIFIED | 488 | `useInboxMessages` at line 40; uses `import * as inboxClient from '../inbox-client'`; 25+ hooks with optimistic updates |
| `desktop/src/renderer/src/stores/kommunikation.ts` | VERIFIED | 59 | `export const useKommunikationStore` with activeView, selectedMessageId, sidebarCollapsed state |
| `desktop/src/renderer/src/modules/kommunikation/KommunikationPage.tsx` | VERIFIED | 304 | Three-column layout with InboxSidebar, MessageList, MessageDetail; imports and uses useInboxMessages, useKommunikationStore |
| `desktop/src/renderer/src/modules/kommunikation/RoutingRulesEditor.tsx` | VERIFIED | 773 | ConditionRow recursive component; AND/OR toggle at line 151; OPERATOR_OPTIONS array |

---

## Key Link Verification

### Plan 14-01 Key Links

| From | To | Via | Status | Evidence |
|------|-----|-----|--------|----------|
| `email/message/event_emitter.go` | `notification/event/emit.go` | event.EmitEvent() | WIRED | Line 30: `return event.EmitEvent(ctx, e.pool, payload)` |
| `migrations/000048...up.sql` | `notification/event/types.go` | event_key values match Go constants | WIRED | EventEmailReceived = "email.message.received" matches seed SQL |

### Plan 14-02 Key Links

| From | To | Via | Status | Evidence |
|------|-----|-----|--------|----------|
| `adapter/email_adapter.go` | `inbox/message/service.go` | FetchNewMessages returns normalized InboxMessages | WIRED | FetchNewMessages defined at line 57; returns []models.InboxMessage |
| `routing/service.go` | `routing/evaluator.go` | Evaluate() called during rule matching | WIRED | Line 127: `if condition.Evaluate(msg)` |
| `inbox/message/service.go` | `adapter/adapter.go` | HandleReply routes through adapter | WIRED | Line 144: `return a.HandleReply(ctx, msg.ID, userID, body)` |
| `inbox/team/service.go` | `inbox/message/repository.go` | ClaimMessage uses message repo AssignMessage | WIRED | Line 126: `assigned, err := s.messageRepo.AssignMessage(ctx, messageID, userID)` |

### Plan 14-03 Key Links

| From | To | Via | Status | Evidence |
|------|-----|-----|--------|----------|
| `server/inbox_grpc.go` | `inbox/message/service.go` | gRPC handler delegates to messageService | WIRED | Line 25: `messageService *message.Service`; used in ListMessages (line 91), MarkRead (line 137) |
| `gateway/route_inbox.go` | `proto/inbox/v1/inbox_grpc.pb.go` | gateway calls inbox gRPC client | WIRED | Line 16: `inboxv1` import; line 35: `getInboxClient()`; line 40: `inboxv1.NewInboxServiceClient(conn)` |
| `cmd/notification/main.go` | `server/inbox_grpc.go` | registers InboxService gRPC server | WIRED | Line 152: `inboxv1.RegisterInboxServiceServer(grpcServer, inboxGRPC)` |
| `cmd/notification/main.go` | `notification/event/bus.go` | InboxConsumer registered as event handler | WIRED | Line 113: `eventBus.RegisterHandler("*", inboxConsumer.HandleEvent)` |
| `cmd/gateway/main.go` | `gateway/route_inbox.go` | NewInboxRoutes registered in gateway route registrar | WIRED | Line 144: `gateway.NewInboxRoutes(registry)` |

### Plan 14-04 Key Links

| From | To | Via | Status | Evidence |
|------|-----|-----|--------|----------|
| `KommunikationPage.tsx` | `api/hooks/useInbox.ts` | TanStack Query hooks fetch inbox data | WIRED | Line 13: `import { useInboxMessages, useArchiveMessage, useToggleStar }`; line 87: `useInboxMessages(filter)` |
| `api/hooks/useInbox.ts` | `api/inbox-client.ts` | API client makes HTTP requests | WIRED | Line 10: `import * as inboxClient from '../inbox-client'`; line 43: `inboxClient.listMessages(filter)` |
| `App.tsx` | `modules/kommunikation/KommunikationPage.tsx` | React Router lazy import | WIRED | Line 51: `const KommunikationPage = lazy(() => import('@/modules/kommunikation/KommunikationPage'))`; line 183: route registered |
| `nav-items.ts` | `KommunikationPage.tsx` | /kommunikation nav entry in sidebar navigation | WIRED | Line 62: `{ id: 'kommunikation', to: '/kommunikation', icon: MessageSquareText, label: 'Kommunikation' }` |

---

## Requirements Coverage

The PLAN files declare requirement IDs INBOX-01, INBOX-02, INBOX-03, EVENT-01. These IDs are **defined only in ROADMAP.md** -- they do not appear in `.planning/REQUIREMENTS.md`.

**IMPORTANT DISCREPANCY FOUND:** The `.planning/REQUIREMENTS.md` tracking table maps INT-01, INT-02, INT-03 to Phase 14. However, the ROADMAP.md correctly maps INT-01/02/03 to **Phase 15 (CalDAV/CardDAV)**. This is a documentation inconsistency in the requirements tracking table only. The actual phase content matches ROADMAP.md Phase 14 precisely. INT-01/02/03 (CalDAV/CardDAV endpoints) were NOT built in Phase 14 and should not have been -- the ROADMAP.md is correct; the REQUIREMENTS.md tracking table has an error.

| Requirement | Source | Description | Status | Evidence |
|-------------|--------|-------------|--------|----------|
| EVENT-01 | ROADMAP.md Phase 14 | All services emit structured events via pg_notify | SATISFIED | 6 PGEventEmitter files; event.EmitEvent() wired in all 6 service.go files; 20 event type constants seeded |
| INBOX-01 | ROADMAP.md Phase 14 | Unified inbox data foundation (proto, migrations, models) | SATISFIED | 27-RPC proto; 4 DB tables in migration 000047; Go models for InboxMessage/TeamInbox/RoutingRule |
| INBOX-02 | ROADMAP.md Phase 14 | Inbox service business logic layer | SATISFIED | message.Service, team.Service, routing.Service; 3 channel adapters; snooze worker; routing evaluator with 26 tests |
| INBOX-03 | ROADMAP.md Phase 14 | Frontend unified inbox UI | SATISFIED | KommunikationPage three-column layout; 25+ TanStack Query hooks; /kommunikation route; sidebar nav entry |
| INT-01 (in REQUIREMENTS.md tracking) | REQUIREMENTS.md tracking table only | CalDAV endpoint for calendar sync | NOT APPLICABLE | INT-01 belongs to Phase 15 per ROADMAP.md -- tracking table has an error. Phase 15 is pending. |
| INT-02 (in REQUIREMENTS.md tracking) | REQUIREMENTS.md tracking table only | CardDAV endpoint for contact sync | NOT APPLICABLE | INT-02 belongs to Phase 15 per ROADMAP.md -- tracking table has an error. Phase 15 is pending. |
| INT-03 (in REQUIREMENTS.md tracking) | REQUIREMENTS.md tracking table only | CalDAV/CardDAV authenticated access | NOT APPLICABLE | INT-03 belongs to Phase 15 per ROADMAP.md -- tracking table has an error. Phase 15 is pending. |

**Action required:** Update `.planning/REQUIREMENTS.md` tracking table to change INT-01, INT-02, INT-03 from "Phase 14" to "Phase 15".

---

## Anti-Patterns Found

| File | Pattern | Severity | Assessment |
|------|---------|----------|------------|
| `adapter/email_adapter.go:58-60` | Returns `nil, nil` when client is nil | INFO | Intentional graceful degradation per design; documented with slog.Debug; adapters initialized with nil clients until concrete gRPC connections established |
| `adapter/chat_adapter.go` | Returns nil when client is nil | INFO | Same pattern -- intentional graceful degradation |
| `adapter/notification_adapter.go` | Returns nil when client is nil | INFO | Same pattern -- intentional graceful degradation |
| `14-04-SUMMARY.md` | ESLint config missing (eslint v9 migration pending) | INFO | Pre-existing issue, not introduced by Phase 14; TypeScript compilation used as primary type check |

No blockers or warnings found. The nil-client pattern is documented, architected intentionally, and matches the plan specification for graceful degradation when dependent services are not yet fully wired.

---

## Human Verification Required

### 1. Three-Column Layout Rendering

**Test:** Launch the Electron desktop app, navigate to /kommunikation
**Expected:** Left sidebar (collapsible, ~256px) with three sections (Smart Views, Kanaele, Team-Postfaecher); center message list with channel-colored badges; right detail pane opens on message click
**Why human:** Visual CSS layout requires running app to verify Tailwind classes render correctly across themes

### 2. Keyboard Triage Shortcuts

**Test:** With focus outside an input, press j (next message), k (previous message), e (archive), s (star), r (reply)
**Expected:** Messages navigate and triage actions fire without mouse interaction; actions show sonner toasts
**Why human:** document.addEventListener keyboard events require interactive session to test

### 3. Snooze Popover Interaction

**Test:** Hover over a message in the list, click the Clock icon, verify three preset buttons appear and clicking one sets snooze
**Expected:** "1 Stunde", "Morgen frueh", "Naechste Woche" presets; custom date/time inputs; message disappears from inbox after snooze
**Why human:** Hover states and Radix Popover interaction require browser rendering

### 4. Team Inbox Claim Workflow

**Test:** As a team inbox member, open a message assigned to that team inbox and click "Ubernehmen"
**Expected:** Message claims atomically to current user; if another user claimed first, "already assigned" error surfaces
**Why human:** Atomic database claim behavior and race conditions require live backend with auth context

### 5. Routing Rules AND/OR Condition Builder

**Test:** Open Routing Rules Editor (admin role required), click "Regel erstellen", build a rule with an AND condition containing two sub-conditions, click "Regel testen"
**Expected:** Recursive ConditionRow renders nested indentation; test panel correctly evaluates the condition against the test message; visual nesting with colored borders
**Why human:** Recursive React component rendering and interactive rule builder require visual/interactive testing

---

## Gaps Summary

No gaps were found. All 5 Success Criteria from ROADMAP.md are fully verified in the codebase:

1. **Event emission** -- 6 PGEventEmitter files across all previously non-emitting services (email, document, invoice, quote, leave, timetracking), each wired via SetEventEmitter into the corresponding service.go files with emit calls at CRUD operation points.

2. **Unified inbox view** -- KommunikationPage presents a complete three-column layout with InboxSidebar (smart views, channel filters, team inboxes), MessageList (channel badges, two-line format, quick actions on hover), and MessageDetail (channel-adaptive reply).

3. **Triage without switching modules** -- Reply, mark-read, archive, star, snooze, and assign all operate directly from the unified inbox. 25+ TanStack Query mutations with optimistic updates. Keyboard shortcuts registered. Channel-adaptive reply (email/chat/notification) in MessageDetail.

4. **Team inboxes with assignment and routing** -- TeamInboxSettings dialog with manual/round-robin modes and member management. ClaimMessage with atomic database guard (WHERE assigned_to IS NULL). RoutingRulesEditor with AND/OR condition tree builder, action execution (route_to_team, assign_to, add_tags, auto_reply), and inline rule testing.

5. **Channel adapters** -- ChannelAdapter interface with compile-time assertions for EmailAdapter, ChatAdapter, NotificationAdapter. InboxConsumer on EventBus normalizes events from all modules into InboxMessage. Circular loop prevention (skip module_id="inbox").

One documentation discrepancy to remediate: REQUIREMENTS.md tracking table incorrectly maps INT-01/02/03 to Phase 14. These belong to Phase 15 (CalDAV/CardDAV). This does not block phase completion -- the phase was built correctly per ROADMAP.md.

---

_Verified: 2026-02-20T01:00:00Z_
_Verifier: Claude (gsd-verifier)_
