# Project Research Summary

**Project:** KMU Hub - All-in-One Workplace Platform for DACH SMBs
**Domain:** Integrated workplace platform (CRM + Chat + PM + Calendar + Email + Video + HR + Finance + Automation + Plugins)
**Researched:** 2026-02-07
**Confidence:** MEDIUM-HIGH (Strong codebase analysis, training data patterns, DACH regulatory knowledge. Library versions need verification.)

## Executive Summary

KMU Hub is expanding from 4 working microservices (Gateway, Auth, CRM, Chat) to a comprehensive workplace platform targeting 5-200 employee companies in the DACH region. The research reveals a critical architectural tension: building 8 new modules as separate microservices would create 12+ containers that no solo developer can maintain, while the self-hosted deployment promise requires operational simplicity. The recommended solution is **service consolidation by domain affinity**: group related modules into 3 new gRPC services (Work, Biz, Automation) instead of 8 separate ones, reducing total services to 7 while preserving logical boundaries.

The research uncovered that **DACH regulatory compliance is non-negotiable** for HR and Finance modules. German labor law (ArbZG, BUrlG) and tax law (GoBD, UStG) impose strict data model requirements that cannot be retrofitted. An immutable invoice model, DATEV export capability, and proper time-tracking constraints must be architectural foundations, not features. The competitive analysis shows that 80% of module functionality can be built with existing patterns (reusing CRM's custom fields engine, chat's file infrastructure, auth's RBAC), while 20% requires carefully chosen external components (LiveKit for video, wazero for plugins, emersion/go-* for email/calendar protocols).

The most critical risk is **scope paralysis**: 8 new modules represent 7-10 months of solo development. The pilot customer (financial education center) needs value NOW, not eventually. The research strongly recommends shipping a Pilot MVP with 4-5 core modules (current 3 + Notifications + Project Management OR Calendar) before building the full suite. Email integration, while valuable, is deceptively complex (4-6 weeks vs. estimated 2-3) due to MIME/IMAP quirks. The automation engine and plugin system should be LAST, not first, because they require stable APIs across all other modules to be useful.

## Key Findings

### Recommended Stack

**Core principle:** Maximize reuse of existing stack (Go 1.25, chi, gRPC, pgx, Redis, MinIO), add only essential domain-specific libraries.

**8 new backend dependencies (Go modules):**
- **robfig/cron/v3** — Cron expression parsing for recurring tasks (PM), calendar events, and automation schedules. Chosen because Go stdlib has no cron parser and this is the de-facto standard.
- **expr-lang/expr** — Expression evaluation engine for automation conditions ("deal.value > 10000 AND stage == 'Won'"). Pure Go, safe sandbox, faster than full JS runtime (goja).
- **tetratelabs/wazero** — WASM runtime for plugin system. Only pure-Go WASM runtime (no CGO), critical for cross-compilation and Alpine Docker images.
- **livekit/server-sdk-go/v2** — LiveKit integration for video/voice. LiveKit already chosen (ADR-005), SDK is thin orchestration layer.
- **emersion/go-imap/v2** — IMAP client for email sync. Standard Go library for IMAP, actively maintained.
- **emersion/go-smtp** — SMTP client for sending email. More robust than stdlib net/smtp.
- **emersion/go-webdav** — CalDAV server implementation. Enables Outlook/Thunderbird bidirectional sync, critical for DACH market where Outlook dominates.
- **microcosm-cc/bluemonday** — HTML sanitization for email content rendering. Security requirement for displaying HTML emails in Electron.

**~8 new frontend dependencies (npm):**
- **date-fns** — Date manipulation library across PM, Calendar, HR, Finance. Replaces moment.js (deprecated), tree-shakeable.
- **@dnd-kit/core + @dnd-kit/sortable** — Drag-and-drop for Kanban boards (PM). Successor to react-beautiful-dnd (deprecated).
- **dompurify** — Client-side HTML sanitization for email rendering (defense-in-depth with bluemonday).
- **recharts** — Charts for finance dashboards and CRM reports. Lighter than Chart.js.
- **livekit-client + @livekit/components-react** — LiveKit WebRTC client + pre-built UI components.
- **@fullcalendar/react** OR **@schedule-x/react** — Calendar UI. FullCalendar has better ecosystem, schedule-x is newer/lighter.
- **quill** OR **tiptap** — Rich text email composer. Quill is simpler, sufficient for email.

**No new infrastructure services** except LiveKit (Docker image `livekit/livekit-server`). All other modules use existing PostgreSQL, Redis, MinIO.

**Critical architectural decision from STACK.md:** Do NOT use separate message broker (NATS, RabbitMQ, Kafka) for inter-service events. Use **PostgreSQL LISTEN/NOTIFY** with an `events` table. Zero new infrastructure, handles SMB scale (<10K events/sec), upgrade path to NATS exists if needed.

### Expected Features

**Module priority (from FEATURES.md complexity analysis):**

**Priority 1 (Build First — Daily Workflow):**
- **Notifications** (1-2 weeks) — Unblocks all modules. Chat notifications (Sprint 5) is foundation.
- **Project Management** (3-4 weeks) — Tasks, Kanban boards, Gantt timeline, dependencies, recurring tasks, time tracking, CRM-linked tasks. Reuses CRM custom fields engine and pipeline stages pattern.
- **Calendar** (3-4 weeks) — Personal/shared calendars, recurring events (RRULE), availability, CRM activity sync, room booking, video call integration. CalDAV server for Outlook sync is differentiator.

**Priority 2 (Business Operations):**
- **Email Integration** (4-6 weeks) — CRITICAL: Estimated 4-6 weeks, not 2-3. IMAP/SMTP proxy (not a mail server), email-to-CRM linking, threading, shared inboxes. Start with send-only + BCC logging, full sync is Phase 2.
- **Video & Voice** (2-3 weeks) — LiveKit integration: 1:1 calls, group calls, screen sharing, call-from-chat. Most logic in LiveKit SDK, thin orchestration in Chat service.
- **Finance** (4-6 weeks) — Quotes, invoices (immutable, GoBD-compliant), PDF generation, tax calculation (19%/7% DE, 20%/10% AT, 8.1%/2.6% CH), payment tracking, **DATEV CSV export** (mandatory for DACH market).

**Priority 3 (Advanced):**
- **HR** (3-5 weeks) — Leave requests (German BUrlG compliance), absence calendar, time tracking (ArbZG max hours/rest period validation), employee profiles, onboarding checklists. Export hours to DATEV for payroll.
- **Automation Engine** (3-5 weeks) — Trigger->condition->action workflows. Depends on all other services emitting events. Start with 10 hardcoded automations, generalize later.

**Priority 4 (Extensibility):**
- **Plugin System** (4-6 weeks) — Config-based customization (80%) + WASM plugins (20%). Extension points in existing services. Plugin API via host functions (no direct DB access). Industry templates (Handwerker, Beratung) as config packages.

**Anti-features (explicitly DO NOT build in v1):**
- Agile/Scrum tooling (too developer-specific)
- Full CalDAV/CardDAV server (use iCal import/export first)
- Email client replacement (focus on CRM-relevant emails)
- PSTN/phone integration (VoIP only)
- Payroll processing (export to provider)
- Full double-entry bookkeeping (DATEV export only)
- Visual workflow builder (API/config-based first)
- Plugin marketplace (curated first-party only)

**DACH-specific must-haves:**
- Feiertage per Bundesland/Kanton (16 DE states + 9 AT + 26 CH cantons)
- Week starts Monday, 24-hour time format
- DATEV Buchungsstapel CSV export (finance)
- XRechnung/ZUGFeRD for B2G invoicing (Phase 2)
- ArbZG compliance (8h max/day, 11h rest, breaks)
- BUrlG leave calculation (20-24 days, part-time pro-rata, carryover to March 31)
- GoBD immutable invoices (no UPDATE, only credit notes)
- German-language templates and UI

### Architecture Approach

**Service topology: 7 total (4 existing + 3 new), NOT 12+**

Current 4 services are preserved. 8 new modules grouped into 3 services:

| Service | Modules | Port | Rationale |
|---------|---------|------|-----------|
| **Work** | Project Management, Calendar, Email | :50054 | Domain affinity: scheduling, assignments, timelines. A task has a due date (calendar), an email creates a task, events link to projects. |
| **Biz** | HR, Finance | :50055 | Administrative back-office functions. Low coupling to real-time features. Both relate to employee/company data. Separate from CRM (customer-facing). |
| **Automation** | Workflow Engine, WASM Plugin Runtime | :50056 | Cross-cutting concern observing events from ALL services. Unique runtime requirements (WASM sandbox). Must be separate. |

**Gateway** connects to all 7 services (adds 3 new gRPC client connections).

**Why consolidation:**
- 1 developer cannot maintain 12 Dockerfiles, 12 compose entries, 12 proto files, 12 gRPC servers, 12 health checks.
- Current CRM service already groups 10 sub-domains (contacts, companies, deals, activities, tags, custom fields, pipeline, reports, filters).
- Self-hosted customers need simple deployment (target: 4 containers max for self-hosted bundle).
- SaaS can still scale services independently in Kubernetes.

**Inter-service communication patterns:**

1. **Synchronous gRPC (Gateway -> Services)** — Existing pattern, no change.
2. **Cross-service reads via shared database** — Services read (but NEVER write) other services' tables. E.g., Work service reads `contacts.first_name` via JOIN instead of gRPC call. Safe because PostgreSQL is single source of truth and there's no database-per-service split. Rule: each service OWNS tables (only it writes), may READ others.
3. **Event bus via PostgreSQL LISTEN/NOTIFY** — Services emit domain events (`NOTIFY kmuhub_events`) after successful writes. Automation service and Gateway consume events. Events persisted to `events` table for audit/replay. Upgrade path to NATS exists if volume exceeds PostgreSQL capacity (~10K/sec).
4. **WebSocket broadcasting (Gateway Hub)** — Existing hub extended to broadcast non-chat events (task.assigned, calendar.event, leave.approved) to desktop clients. Gateway listens to `NOTIFY kmuhub_realtime` channel.

**Major architectural components (new modules):**

1. **Work Service** — 3 sub-domains (project, calendar, email) sharing a database schema. Single gRPC endpoint `WorkService` with RPCs grouped by sub-domain. Internal communication via direct service layer calls (like CRM contacts -> tags). External writes via gRPC to other services.
2. **Biz Service** — 2 sub-domains (HR, finance). Same pattern as Work. Finance reads CRM tables for invoice-to-deal linking. HR extends Auth user model with employee-specific fields.
3. **Automation Service** — Event consumer (LISTEN/NOTIFY), workflow execution engine (expr-lang condition evaluation), action executor (gRPC client of ALL services), WASM plugin runtime (wazero sandbox). Must be gRPC client of all 6 other backend services.
4. **Plugin System (embedded in Automation)** — Two-tier: config-based (80%, no code) + WASM (20%, sandboxed code). Host functions expose controlled API (no direct DB access). Plugins triggered via workflow engine or extension points (pre-save hooks in services).
5. **LiveKit Integration (in Chat Service)** — LiveKit server runs as external Docker container. Chat service uses LiveKit SDK to create rooms, generate tokens, handle webhooks. Desktop uses LiveKit client SDK for WebRTC.

**Desktop app module architecture:**
- Module-based lazy loading (React.lazy + code splitting)
- Each module registers routes, sidebar items, permissions
- Shared component library + API client layer
- Zustand for state management (per-module stores)
- Target: <300MB RAM, module unloading after 5min inactivity

**Build order (dependency flow):**
1. Phase 3 (Chat) ✓ Complete
2. Phase 4 (Desktop Shell + CRM/Chat UI) — Pure frontend, no backend
3. Phase 5 (Work Service) — PM first, then Calendar, then Email
4. Phase 6 (Video/Voice) — LiveKit integration in Chat service
5. Phase 7 (Biz Service) — HR first, then Finance
6. Phase 8 (Automation + Plugins) — LAST, needs stable APIs from all modules

**Event emission: Incremental, not deferred.** Add `NOTIFY` calls to services as they're built (Chat Sprint 5 = notifications, CRM/Work/Biz = domain events). By Phase 8, all events already exist for automation engine to consume.

### Critical Pitfalls

**From PITFALLS.md analysis, top 5 project killers:**

1. **Gateway God Object (CRITICAL)** — Adding 8 modules to gateway creates coupling point where one failing service blocks entire gateway. Prevention: Refactor gateway NOW (before module 5) to lazy-connect gRPC clients, split `GatewayHandler` into per-module route registrars, remove `os.Exit(1)` on connection failure (return 503 for unavailable routes instead).

2. **Shared Database, Hidden Coupling (CRITICAL)** — All services share one PostgreSQL database. Without ownership rules, services write to each other's tables causing cascading failures. Prevention: Document table ownership (SCHEMA_OWNERS.md), enforce "services own tables (write) but may read others (read-only)", forbid cross-service writes (use gRPC), add migration CI check for sequence conflicts.

3. **DACH Regulatory Compliance as Afterthought (CRITICAL)** — HR and Finance data models MUST encode legal requirements from day 1. German ArbZG (max 8h/day, 11h rest), BUrlG (vacation calculation), GoBD (immutable invoices), UStG (VAT handling), DATEV compatibility cannot be retrofitted. Prevention: Research and document regulatory requirements in dedicated ADRs BEFORE writing code. Hire Steuerberater for 2-hour GoBD consultation. Make compliance constraints the foundation, not a feature.

4. **Scope Paralysis — 12 Modules for 1 Developer (CRITICAL)** — Current pace ~1 module/month = 12 months without shipping. Pilot customer (Zentrum fur finanzielle Aufklarung) needs value NOW. Prevention: Define Pilot MVP explicitly (likely Auth + CRM + Chat + Calendar = 4 modules). Ship Pilot MVP, iterate with feedback. Defer modules without customer demand. Set 2-3 week time-boxes per module.

5. **Config Struct Explosion (HIGH)** — Current `Config` struct has 22 fields for 4 services. Adding 8 modules pushes to 60+ fields, every service loads fields it never uses. Prevention: Refactor to per-service config structs NOW. Each service has `config.LoadAuth()`, `config.LoadCRM()`, etc. with shared `CommonConfig` embedded.

**Additional high-severity pitfalls:**
- **Proto file sprawl** — CRM proto already large. Split into entity-specific protos before adding more.
- **Cross-module feature interactions** — Users expect integration ("create deal from chat", "calendar events on PM timeline"). Build entity linking system and unified activity feed early.
- **Self-hosted deployment complexity** — 12+ Docker containers is NOT simple self-hosting. Build consolidated binary option (one binary, all services in-process) for self-hosted customers.
- **Email integration underestimated** — 4-6 weeks, not 2-3. MIME parsing, threading, IMAP sync, HTML sanitization are deceptively complex. Scope v1 to send-only + BCC logging.
- **Multi-tenancy bolted on** — Add `organization_id` to all tables NOW. Retrofitting multi-tenancy to 50+ tables guarantees data leaks. Use PostgreSQL RLS as safety net.

**Solo developer amplification factors:**
- No code review safety net (every mistake ships)
- Context switching cost is devastating (finish one module before starting next)
- Bus factor of 1 (obsessive documentation required)
- AI-generated code accelerates tech debt (periodic refactoring sprints needed)

## Implications for Roadmap

### Recommended Phase Structure

Based on dependencies, domain complexity, and pilot customer needs:

#### Phase 4: Desktop App Foundation (Current Priority)
**Rationale:** Pure frontend work, no new backend services. Builds the shell for all future modules.
**Delivers:** Electron app structure, module registration system, lazy loading, shared component library, workspace layout.
**Avoids:** Building backend modules without UX to test them.
**Research needed:** None (standard React patterns).

#### Phase 5: Notifications + Project Management (Pilot MVP Completion)
**Rationale:** Notifications unblock all future modules. PM is second most-used feature after chat (task management is core to daily workflow). Both can deliver value to pilot customer immediately.
**Delivers:**
- Centralized notification service (in-app, desktop push, email digest, per-user preferences)
- Tasks with assignees, due dates, statuses, priorities
- Projects as containers
- List + Kanban board views
- Task comments (reuse chat comment infrastructure)
- File attachments (reuse MinIO infrastructure)
- CRM-linked tasks (create task from deal)
- Task search (extend PostgreSQL FTS)
**Addresses:** Notification fatigue (Pitfall 12), cross-module integration foundation.
**Avoids:** Building more modules without notification infrastructure in place.
**Stack:** Reuses existing patterns. Frontend adds `@dnd-kit/*` for Kanban, backend is pure domain logic.
**Duration estimate:** 4-5 weeks (1 week notifications, 3-4 weeks PM).
**Research needed:** None for notifications. PM needs `/gsd:research-phase` for Gantt/timeline libraries if including that (defer to v2?).

#### Phase 6: Calendar OR Video/Voice (Pilot Customer Decision)
**Rationale:** Depends on which pilot customer needs more urgently. Calendar enables scheduling, video enables remote collaboration. Both integrate with chat.
**Option A (Calendar):**
- Personal + shared calendars, events, recurring (RRULE), availability
- CRM activity sync (meeting with contact), PM task deadlines on calendar
- Video call auto-link (calendar event creates LiveKit room)
- iCal import/export (defer CalDAV server to Phase 2)
**Option B (Video/Voice):**
- LiveKit integration: 1:1 calls, group calls, screen sharing
- Call-from-chat button, call history
- CRM activity auto-log (call with contact)
- Recording (requires LiveKit Egress, optional for v1)
**Duration estimate:** 3-4 weeks (Calendar), 2-3 weeks (Video).
**Research needed:** Calendar needs RRULE library research, CalDAV feasibility check. Video needs LiveKit SDK version verification, Electron screen sharing API check.

#### Phase 7: Work Service Completion (The Other One)
**Rationale:** Complete the Work service by building whichever module wasn't done in Phase 6.
**Duration estimate:** 2-4 weeks depending on module.
**Avoids:** Leaving Work service incomplete (better to finish one service fully before starting Biz).

#### Phase 8: Biz Service — Finance (Revenue Generation)
**Rationale:** Finance before HR because it directly generates revenue (invoices) and is likely needed by pilot customer (financial education center). HR is internal operations.
**Delivers:**
- Quotes, invoices (immutable data model per GoBD)
- PDF generation with company letterhead
- Tax calculation (19%/7% DE default, configurable for AT/CH)
- Payment status tracking, overdue reminders
- Sequential invoice numbering (no gaps)
- **DATEV Buchungsstapel CSV export** (mandatory day 1)
- Deal-to-quote-to-invoice pipeline (CRM integration)
**Addresses:** GoBD compliance (Pitfall 3), DATEV integration (Pitfall 13).
**Avoids:** Building generic invoicing without DACH compliance.
**Duration estimate:** 5-6 weeks (includes DATEV format research).
**Research needed:** `/gsd:research-phase` for GoBD requirements (immutable model design), DATEV CSV format specification, tax calculation rules for AT/CH, XRechnung/ZUGFeRD (Phase 2 feature but research implications now).
**CRITICAL PRE-WORK:** 2-hour consultation with Steuerberater on GoBD compliance before writing data model.

#### Phase 9: Biz Service — HR (Operational Efficiency)
**Rationale:** HR is last "feature module" before automation/extensibility. Needed for Arbeitszeitgesetz compliance (German time-tracking mandate since 2022).
**Delivers:**
- Employee profiles (extend Auth users)
- Leave requests + approval workflow (German BUrlG compliance)
- Leave balance tracking (20-24 days, part-time pro-rata, carryover to March 31)
- Absence calendar (integrates with Calendar module)
- Time tracking (clock in/out, max 8h/day validation per ArbZG, 11h rest period check)
- Sick leave recording (AU after 3 days per German law)
- Onboarding checklists (reuse PM task system)
- Document storage (contracts, Zeugnisse) in MinIO
- DATEV hours export for payroll
**Addresses:** ArbZG compliance (Pitfall 3).
**Avoids:** Building generic HR without German labor law constraints.
**Duration estimate:** 4-5 weeks.
**Research needed:** `/gsd:research-phase` for ArbZG/BUrlG requirements (max hours, rest periods, leave calculation), Mutterschutz/Elternzeit legal requirements, Austrian/Swiss labor law variations, DATEV payroll export format.

#### Phase 10: Automation Engine (Cross-Module Orchestration)
**Rationale:** All feature modules exist, stable APIs, events already being emitted. Now we connect them.
**Delivers:**
- Trigger-action workflow engine (event-driven + scheduled)
- 10 pre-built automations (deal won -> create invoice, task completed -> notify manager, etc.)
- Condition evaluation (expr-lang: "deal.value > 10000 AND stage == 'Won'")
- Action execution (gRPC calls to all services)
- Workflow logs (audit trail)
- API/config-based workflow creation (defer visual builder)
**Addresses:** Cross-module integration (Pitfall 7).
**Avoids:** Premature abstraction (Pitfall 14) — building automation before knowing what to automate.
**Duration estimate:** 3-5 weeks.
**Research needed:** None (use expr-lang as decided, workflow pattern is standard).

#### Phase 11: Plugin System (Customization Layer)
**Rationale:** Last because it needs stable extension points across all modules.
**Delivers:**
- Config-based customization (already exists: custom fields, workflows)
- WASM plugin runtime (wazero)
- Plugin host functions (GetContact, CreateTask, HTTPGet, Log)
- Plugin API + SDK (Go/Rust)
- Extension points (pre-save hooks in services)
- 2-3 example plugins (custom validation, external API connector)
- Industry templates as config packages (Handwerker, Beratung)
**Duration estimate:** 4-6 weeks (includes SDK design, security sandbox, host function implementation).
**Research needed:** `/gsd:research-phase` for wazero API (host function implementation), WASM-Go interop patterns, plugin SDK design, security boundaries.

#### Email Integration: Special Case
**Recommendation: DEFER to Phase 2 OR scope to minimal v1**
**Rationale:** Email is 4-6 weeks of complex work (Pitfall 9). Pilot customer almost certainly has email already. What they need is CRM linking, not a full client.
**Minimal v1 scope (if included):**
- Send email via SMTP (from within KMU Hub)
- BCC-to-CRM logging (forward BCC to special address, auto-link to contact)
- Email-to-task conversion (manual: user clicks "create task from email")
- NO full IMAP sync, NO threading, NO shared inbox (Phase 2)
**Duration if minimal:** 2-3 weeks.
**Duration if full sync:** 5-6 weeks.
**Research needed:** IMAP/SMTP library evaluation (emersion/go-imap v2 API), MIME parsing edge cases, HTML sanitization strategy, OAuth2 for Gmail/Outlook.

### Phase Ordering Rationale

1. **Dependencies flow left to right:** Notifications before PM (PM needs notifications). PM before Automation (Automation needs PM events). Automation before Plugins (Plugins triggered via Automation).
2. **Desktop foundation before backend complexity:** Phase 4 (Desktop) unblocks parallel UI + backend work and gives us a testbed.
3. **Value to pilot customer prioritized:** Auth + CRM + Chat (done) + Notifications + PM + Calendar = Pilot MVP that delivers daily-driver value. This is 5-6 modules total, roughly 6-7 months from project start. Ship this, get feedback, iterate.
4. **Finance before HR:** Revenue-generating features before internal operations. Pilot customer (financial education) likely needs invoicing.
5. **Automation last:** Needs events from all modules. Building it first means no events to consume.
6. **Plugins last:** Needs stable extension points. Building it first means no services to extend.
7. **Email deferred or minimized:** Complex, pilot customer has email already, focus on CRM integration not full client.

### Research Flags

**Phases needing `/gsd:research-phase` during planning:**

- **Phase 6 (Calendar)** — RRULE library research, CalDAV protocol feasibility, timezone handling patterns.
- **Phase 8 (Finance)** — GoBD compliance requirements (immutable invoice model), DATEV Buchungsstapel format specification, XRechnung/ZUGFeRD structure.
- **Phase 9 (HR)** — ArbZG/BUrlG detailed requirements (max hours, rest periods, leave calculation), Mutterschutz/Elternzeit legal deadlines, DATEV payroll export format.
- **Phase 11 (Plugin System)** — wazero host function API, WASM-Go interop patterns, plugin security boundaries, SDK design.
- **Email (if full sync)** — IMAP/SMTP library API deep-dive, MIME edge cases, threading algorithm, OAuth2 flows.

**Phases with standard patterns (skip additional research):**

- **Phase 4 (Desktop Shell)** — Standard React module architecture.
- **Phase 5 (Notifications + PM)** — Reuses existing patterns (notifications extend chat, PM reuses CRM patterns).
- **Phase 6 (Video/Voice)** — LiveKit has excellent documentation, standard integration pattern.
- **Phase 10 (Automation)** — expr-lang is well-documented, workflow engine pattern is standard.

### Gateway Refactoring Trigger

**CRITICAL: Gateway must be refactored BEFORE starting Phase 5 (module 5).**

Current gateway pattern (all gRPC clients initialized in main.go, single GatewayHandler struct) does not scale beyond 4 services. Refactoring must happen before adding Work service (module 5).

**Refactoring checklist:**
- Split `GatewayHandler` into per-service handlers
- Lazy-connect gRPC clients (don't dial all at startup)
- Remove `os.Exit(1)` on connection failure (return 503 for unavailable routes)
- Per-service config structs (embedded CommonConfig)
- Route registration via interface (each service registers its routes)

**Estimated effort:** 1 week (should be Sprint 0 of Phase 5).

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| **Stack recommendations** | MEDIUM-HIGH | Go libraries well-researched from training data. Versions need verification (WebSearch unavailable). Architectural patterns (PostgreSQL LISTEN/NOTIFY, wazero, LiveKit) are HIGH confidence. Frontend library choices (dnd-kit, recharts) are MEDIUM (npm ecosystem changes fast). |
| **Feature landscape** | HIGH | PM, Calendar, Email, Video, HR, Finance, Automation, Plugins are well-established product categories with stable feature sets. DACH-specific requirements (Feiertage, DATEV, ArbZG) are based on German federal law (stable, long-standing regulations). |
| **Architecture patterns** | HIGH | Based on direct codebase analysis of 4 existing services. Service consolidation recommendation is opinionated but justified by solo developer constraint. PostgreSQL LISTEN/NOTIFY pattern is standard. Cross-service read via shared DB is pragmatic for single-DB architecture. |
| **Pitfalls identification** | HIGH | Based on codebase inspection (gateway coupling, config explosion), project context (solo dev, scope), and domain knowledge (DACH compliance, email complexity). Solo developer risks (bus factor, context switching) are based on team structure. |
| **DACH regulatory requirements** | MEDIUM | General requirements (GoBD, ArbZG, BUrlG, UStG) are HIGH confidence (fundamental German law). Specific implementation details (exact DATEV CSV fields, ArbZG exception rules for industries) are MEDIUM confidence and require domain expert verification. |
| **Duration estimates** | MEDIUM | Based on current project pace (~4 weeks/module with AI assistance). Email estimate (4-6 weeks) is higher confidence than others due to known complexity. Finance (5-6 weeks) includes DATEV research buffer. |

**Overall confidence: MEDIUM-HIGH**

Architectural recommendations and pitfall identification are HIGH confidence (based on codebase analysis and solo dev constraints). Stack choices and feature priorities are MEDIUM-HIGH (training data patterns, need version verification). DACH compliance details are MEDIUM (legal frameworks known, implementation specifics need expert review).

### Gaps to Address During Planning

**Critical gaps requiring expert consultation:**

1. **GoBD compliance specifics** — Consult with Steuerberater (tax advisor) on immutable invoice data model BEFORE building Finance module. Document retention vs. GDPR deletion requirements need legal advice.

2. **ArbZG/BUrlG implementation details** — Consult with German HR compliance expert on edge cases: Gleitzeitkonto (flexible time accounts), Ueberstundenabbau (overtime compensation), Teilzeit/Minijob rules. Part-time leave pro-rata calculation has nuances.

3. **DATEV format specification** — Obtain official DATEV Buchungsstapel format documentation (not publicly available in detail). May require DATEV developer account or partner consultation.

4. **Austrian/Swiss labor law variations** — Research focuses on German law. Austrian (25 vacation days, UID-Nummer) and Swiss (4 weeks vacation, cantonal holidays, MwSt rates) variations need verification with local experts if targeting those markets in v1.

5. **Library version verification** — ALL library versions from STACK.md are training data estimates. Verify against official sources before adding to go.mod/package.json:
   - wazero (latest v1.x)
   - livekit/server-sdk-go (v2.x)
   - emersion/go-imap (v2.x)
   - expr-lang/expr (latest)
   - @dnd-kit/core (latest)
   - @livekit/components-react (latest)

**Technical gaps requiring phase-specific research:**

- **RRULE expansion algorithm** — Calendar phase needs to research PostgreSQL + Go patterns for recurring event expansion (hybrid approach: store rule + pre-expand occurrences).
- **WASM host function ABI** — Plugin phase needs to design stable ABI for WASM-Go communication (protobuf serialization likely, needs wazero API research).
- **HTML email sanitization** — Email phase needs both server-side (bluemonday) and client-side (dompurify) strategy, plus Electron sandboxing approach.

**Product gaps requiring pilot customer feedback:**

- **Module priority** — Research assumes pilot customer needs PM + Calendar. Validate with customer: which 4-5 modules constitute their Pilot MVP? They may prioritize differently.
- **DATEV export necessity** — Assumption: DATEV export is mandatory. But pilot customer (Zentrum fur finanzielle Aufklarung) is education sector — do they use DATEV or different accounting software? Validate before spending 2 weeks on DATEV export.
- **Video vs. Calendar priority** — Which does pilot customer need first for daily operations? This affects Phase 6 choice.

## Sources

### Primary Sources (HIGH confidence)
- **Codebase analysis:** Direct inspection of existing 4-service architecture (gateway, auth, CRM, chat), migrations, proto files, docker-compose, config structures, service patterns. 134 Go source files, 19 migrations analyzed.
- **Project documentation:** `docs/ARCHITECTURE.md` (ADRs 1-6), `docs/LEARNINGS.md` (10 pitfalls from previous project), `CLAUDE.md` (architecture rules), `PROJECT.md`, `ROADMAP.md`.
- **Domain frameworks:** PostgreSQL LISTEN/NOTIFY (official docs), gRPC patterns (protobuf official), Go microservice patterns (stdlib + established libraries).

### Secondary Sources (MEDIUM confidence)
- **Go library ecosystem:** robfig/cron, expr-lang/expr, tetratelabs/wazero, emersion/go-* libraries, livekit/server-sdk-go — all based on training data knowledge (pre-2025). Library capabilities and APIs are well-established, but exact current versions need verification.
- **Frontend libraries:** @dnd-kit/*, recharts, dompurify, livekit-client — training data knowledge. npm ecosystem changes faster than Go ecosystem, higher verification priority.
- **DACH regulations:** GoBD (Grundsatze ordnungsmasiger Buchfuhrung), ArbZG (Arbeitszeitgesetz), BUrlG (Bundesurlaubsgesetz), UStG (Umsatzsteuergesetz), BDSG (Bundesdatenschutzgesetz) — German federal law, stable frameworks. Training data includes general requirements.

### Tertiary Sources (LOW confidence, needs validation)
- **DATEV format details:** Training data includes general knowledge of DATEV Buchungsstapel format (CSV structure, SKR03/SKR04 account charts). Exact field specifications need official DATEV documentation.
- **XRechnung/ZUGFeRD specifications:** Training data includes awareness of electronic invoice standards. Exact XML schema and implementation patterns need EN 16931 specification and ZUGFeRD documentation.
- **Austrian/Swiss regulatory variations:** Training data includes awareness of differences (e.g., 25 vacation days in Austria, 4 weeks in Switzerland). Exact legal requirements need local expert verification.
- **Specific library version compatibility:** Training data cannot guarantee that wazero v1.8.x works with Go 1.25.6, or that @livekit/components-react current version works with React 19. These need runtime verification.

### Research Limitations
- **WebSearch unavailable:** All research based on training data (cutoff ~January 2025). No live verification of current library versions, 2026 regulatory changes, or recent best practices evolution.
- **No pilot customer input:** Feature priorities and module selection based on general DACH SMB needs. Actual pilot customer (Zentrum fur finanzielle Aufklarung) requirements not validated during research.
- **Solo developer context:** Recommendations heavily weighted toward operational simplicity (service consolidation, self-hosted deployment, maintenance burden). Different recommendations might apply with larger team.

---

## Ready for Roadmap: YES

**Key inputs for roadmap creation:**
- Service consolidation strategy (7 services, not 12)
- Phase structure (11 phases recommended, with Pilot MVP at Phase 6)
- Duration estimates (27-41 weeks total, Pilot MVP at ~7 months)
- Research flags (4 phases need deeper research during planning)
- Gateway refactoring trigger (must happen before Phase 5)
- DACH compliance checkpoints (Steuerberater consultation before Finance, HR legal research before HR module)

**Next steps:**
1. Validate Pilot MVP module selection with actual pilot customer
2. Decide Phase 6 priority: Calendar vs. Video (depends on customer need)
3. Schedule Steuerberater consultation (2 hours) for GoBD requirements
4. Verify library versions before Phase 5 begins
5. Plan gateway refactoring sprint (Sprint 0 of Phase 5)

---

*Research completed: 2026-02-07*
*Synthesized from: STACK.md, FEATURES.md, ARCHITECTURE.md, PITFALLS.md*
