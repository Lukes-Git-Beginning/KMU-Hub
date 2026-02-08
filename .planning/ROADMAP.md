# Roadmap: KMU Hub

## Overview

KMU Hub expands from its completed foundation (Auth, CRM, Chat) into a comprehensive workplace platform across 17 additional phases (20 total). The journey continues with Project Management (including Gantt and time tracking), then Calendar and an expanded Video/Voice/Meetings phase. Security & Compliance gates the Beta release with 2FA, audit logging, DSGVO compliance, and i18n. Email and Documents round out the communication layer with global search. Finance and HR deliver the Business Suite. Five focused integration mini-phases (CalDAV, Teams/Slack, Bexio, Abacus, Run my Accounts) connect the Hub to the Swiss/DACH business ecosystem. Finally, Automation and Plugins provide the extensibility layer. The architecture consolidates new modules into 3 backend services (Work, Biz, Automation) to keep operational complexity manageable for a solo developer.

## Milestones

- ✅ **Foundation** - Phases 1-3 (Auth/Infra, CRM Core, Chat & Messaging)
- 📋 **Pilot MVP** - Phases 4-8 (Notifications, Desktop, PM, Calendar, Video/Meetings -- daily-driver for pilot customer)
- 📋 **Compliance & Comms** - Phases 9-11 (Security & Compliance, Email, Documents & Files -- enterprise-ready communication)
- 📋 **Business Suite** - Phases 12-13 (Finance, HR -- operational and revenue tools)
- 📋 **Integrations** - Phases 14-18 (CalDAV/CardDAV, Teams/Slack, Bexio, Abacus, Run my Accounts -- external connectivity)
- 📋 **Extensibility** - Phases 19-20 (Automation, Plugins -- customization and integration layer)

## Phases

<details>
<summary>✅ Foundation (Phases 1-3) - COMPLETE</summary>

- [x] **Phase 1: Auth & Infrastructure** - JWT auth, API gateway, PostgreSQL/Redis, CI/CD, user management
- [x] **Phase 2: CRM Core** - Contacts, companies, deals pipeline, activities, filters, reports
- [x] **Phase 3: Chat & Messaging** - Channels, DMs, threads, mentions, read receipts, file sharing, search

</details>

### Pilot MVP (Phases 4-8)

- [x] **Phase 4: Notifications + Gateway Modernization** - Centralized notification system and gateway refactoring for multi-service scale
- [x] **Phase 5: Desktop App Shell** - Electron workspace with module loading, personalization, and role-based dashboards
- [x] **Phase 6: Project Management** - Tasks, projects, Kanban boards, Gantt chart, task timer, subtasks, dependencies, templates, CRM integration, custom fields
- [ ] **Phase 7: Calendar & Scheduling** - Personal/shared calendars, recurring events, room booking, DACH holidays
- [ ] **Phase 8: Video, Voice & Meetings** - LiveKit-powered calls, screen sharing, recording, meeting management, emoji reactions, presence/online status

### Compliance & Comms (Phases 9-11)

- [ ] **Phase 9: Security & Compliance** - 2FA, audit log, DSGVO export/deletion, session management, secret vault, i18n (DE/FR/IT/EN)
- [ ] **Phase 10: Email Integration** - Full IMAP/SMTP email, CRM auto-linking, threading, contact import/export, two-level contacts
- [ ] **Phase 11: Documents & Files** - File browser, upload, preview, versioning, sharing, full-text search, tags, global search

### Business Suite (Phases 12-13)

- [ ] **Phase 12: Finance Module** - GoBD-compliant quotes and invoices, tax calculation, DATEV export
- [ ] **Phase 13: HR Module** - Leave management, time tracking, employee profiles, ArbZG/BUrlG compliance

### Integrations (Phases 14-18)

- [ ] **Phase 14: Integration - CalDAV/CardDAV** - Bidirectional calendar and contact sync with Outlook, Thunderbird, macOS
- [ ] **Phase 15: Integration - Teams & Slack** - Notification forwarding and basic bidirectional interaction
- [ ] **Phase 16: Integration - Bexio** - Contact and invoice sync with Bexio accounting
- [ ] **Phase 17: Integration - Abacus** - Contact and invoice sync with Abacus ERP/accounting
- [ ] **Phase 18: Integration - Run my Accounts** - Contact and financial document sync with Run my Accounts

### Extensibility (Phases 19-20)

- [ ] **Phase 19: Automation Engine** - Trigger-condition-action workflows across all modules
- [ ] **Phase 20: Plugin System** - Config-based customization, WASM runtime, extension points, industry templates

## Phase Details

### Phase 4: Notifications + Gateway Modernization
**Goal**: Every module can notify users in real time, and the gateway architecture scales to support 7+ backend services
**Depends on**: Phase 3 (Chat -- WebSocket hub, notification patterns from mentions)
**Requirements**: NOTF-01, NOTF-02, NOTF-03
**Success Criteria** (what must be TRUE):
  1. User sees a notification bell with unread count that updates in real time when events occur across any module
  2. User receives desktop push notifications (Electron system tray) when the Hub is minimized or in background
  3. User can configure per-event-type and per-channel notification preferences (mute, desktop only, all, etc.)
  4. Gateway connects to backend services lazily and returns 503 for unavailable routes instead of crashing on startup
  5. Adding a new backend service to the gateway requires only registering a route handler (no monolithic handler changes)
**Plans**: 3 plans

Plans:
- [x] 04-01-PLAN.md -- Gateway modernization (ServiceRegistry with lazy gRPC, per-service route handlers, graceful degradation)
- [x] 04-02-PLAN.md -- Notification service backend (proto, migrations, event bus, notification + preference services, gRPC server)
- [x] 04-03-PLAN.md -- Notification delivery + integration (gateway HTTP routes, WebSocket push, event emission from CRM/Chat, Docker Compose)

### Phase 5: Desktop App Shell
**Goal**: Users have a functional Electron desktop application that serves as the single window for their workday, with CRM and Chat modules already usable
**Depends on**: Phase 4 (Notifications -- desktop push integration, gateway modernization)
**Requirements**: DESK-01, DESK-02, DESK-03, DESK-04
**Success Criteria** (what must be TRUE):
  1. User launches the Electron app and navigates between CRM, Chat, and future modules via a persistent sidebar
  2. User can add, remove, and rearrange dashboard widgets to personalize their workspace
  3. Admin can configure role-based default dashboards so a CEO sees different defaults than an office worker
  4. User can view recently accessed contacts, deals, and messages when briefly offline (local cache)
  5. Each module loads independently (lazy loading) and the app stays under 300MB RAM with 2-3 active modules
**Plans**: 7 plans

Plans:
- [x] 05-01-PLAN.md -- Electron shell foundation (electron-vite config, secure IPC bridge, safeStorage auth, Tailwind v4, all deps installed)
- [x] 05-02-PLAN.md -- App shell + auth flow (API client, WebSocket manager, Zustand stores, sidebar navigation, login page, routing)
- [x] 05-03-PLAN.md -- CRM module UI (contacts, companies, deals list + pipeline, activities, search with TanStack Query hooks)
- [x] 05-04-PLAN.md -- Chat module + notifications (channels, real-time messaging, typing indicators, threads, notification bell, desktop push)
- [x] 05-05-PLAN.md -- Dashboard + widget system (react-grid-layout grid, 6 widgets, drag-and-drop, widget picker, layout persistence)
- [x] 05-06-PLAN.md -- Role-based dashboards + backend (dashboard_layouts migration, API endpoints, admin settings, server sync)
- [x] 05-07-PLAN.md -- Offline caching + final verification (TanStack Query persistence, offline banner, CORS update, memory check, human verify)

### Phase 6: Project Management
**Goal**: Users can manage their daily work through tasks and projects without leaving the Hub, with visual timeline planning and time tracking
**Depends on**: Phase 5 (Desktop shell -- UI for task views), Phase 4 (Notifications -- task assignment alerts)
**Requirements**: PM-01, PM-02, PM-03, PM-04, PM-05, PM-06, PM-07, PM-08, PM-09, PM-10, PM-11, PM-15, PM-16, PM-17
**Success Criteria** (what must be TRUE):
  1. User can create a task with an assignee, due date, priority, and custom status, and the assignee is notified
  2. User can organize tasks into projects with shared settings and control which team members have access
  3. User can switch between a sortable/filterable list view and a drag-and-drop Kanban board for the same project
  4. User can comment on a task with @mentions (reusing chat infrastructure) and attach files (reusing MinIO)
  5. User can search across all projects and filter by assignee, status, priority, due date, and custom fields
  6. User can link a task to a CRM deal or contact (e.g., "Follow up on Acme deal") and navigate between them
  7. User can view a project timeline as a Gantt chart showing task bars with dependency arrows and date range navigation
  8. User can start/stop a timer on a task to track time spent, with manual time entry and per-task time summaries
**Plans**: 10 plans

Plans:
- [x] 06-01-PLAN.md -- Proto + migrations + models + config (Work service data foundation)
- [x] 06-02-PLAN.md -- Project + status service packages (CRUD, membership, templates, preferences)
- [x] 06-03-PLAN.md -- Task + comment service packages (CRUD, nesting, dependencies, events, comments)
- [x] 06-04-PLAN.md -- gRPC server + gateway routes + Docker + OpenAPI (connection layer)
- [x] 06-05-PLAN.md -- Frontend module shell + API hooks + project pages (frontend foundation)
- [x] 06-06-PLAN.md -- Task list view + Kanban board with @dnd-kit DnD (core views)
- [x] 06-07-PLAN.md -- Task detail panel + comments + files + activity log (collaboration)
- [x] 06-08-PLAN.md -- CRM linking + search + filters + My Tasks + templates (integration)
- [x] 06-09-PLAN.md -- Gantt chart view (read-only timeline with task bars, dependency arrows, date range scrolling, critical path)
- [x] 06-10-PLAN.md -- Task timer + time tracking (start/stop timer, manual time entry, time logs per task, summary views)

### Phase 7: Calendar & Scheduling
**Goal**: Users can manage their schedules, book meetings, and coordinate team availability entirely within the Hub
**Depends on**: Phase 6 (PM -- Work service exists, task deadlines on calendar), Phase 4 (Notifications -- event reminders)
**Requirements**: CAL-01, CAL-02, CAL-03, CAL-04, CAL-05, CAL-06, CAL-07
**Success Criteria** (what must be TRUE):
  1. User has a personal calendar with day, week, and month views displaying their events
  2. Teams have shared calendars visible to all members, and users can overlay multiple calendars
  3. User can create recurring events (daily standup, weekly meeting) using standard recurrence rules
  4. User can invite colleagues to events, and invitees can accept, decline, or mark as tentative
  5. User can book a meeting room or resource via dedicated resource calendars that show availability
  6. Creating a calendar event with a video call option auto-generates a LiveKit room link in the event details
  7. Calendar displays DACH public holidays for the user's configured Bundesland/Kanton, with weeks starting Monday
**Plans**: TBD

Plans:
- [ ] 07-01: Calendar backend (event models, RRULE recurring events, personal/shared calendars, RSVP)
- [ ] 07-02: Calendar UI (day/week/month views, event creation, multi-calendar overlay)
- [ ] 07-03: Resource booking + DACH features (room/resource calendars, holiday data, LiveKit link generation)

### Phase 8: Video, Voice & Meetings
**Goal**: Users can make video/voice calls, manage meetings end-to-end, see colleague presence, and react to messages -- replacing Zoom/Teams
**Depends on**: Phase 7 (Calendar -- video call links in events), Phase 3 (Chat -- call-from-chat, emoji infrastructure)
**Requirements**: VID-01, VID-02, VID-03, VID-04, VID-05, VID-06, VID-07, MEET-01, MEET-02, MEET-03, MEET-04, MEET-05, CHAT-01, CHAT-02
**Success Criteria** (what must be TRUE):
  1. User can make a 1:1 video call to a colleague and both see each other's video and hear audio
  2. User can join a group video call with up to 25 participants with a gallery view
  3. User can toggle camera off for audio-only calling and mute/unmute microphone during any call
  4. User can share their entire screen or a specific application window during a call
  5. User can start a call directly from a chat channel or DM conversation with one click
  6. User can record a call after all participants give DSGVO-compliant consent, and the recording is stored in MinIO
  7. User can schedule a meeting with agenda, attendees, and see a pre-meeting lobby with shared documents
  8. After a meeting, a summary record is created with notes and action items linkable as tasks
  9. Users can react to chat messages with emoji reactions (add, remove, reaction counts)
  10. Users see presence indicators (online/away/offline/in a call) for colleagues across the app
**Plans**: 5 plans (estimated)

Plans:
- [ ] 08-01: LiveKit infrastructure + call service (LiveKit Docker setup, room management, token generation, call metadata)
- [ ] 08-02: Call UI + media controls (video/audio rendering, mute/camera toggle, screen sharing, gallery view)
- [ ] 08-03: Call integration + recording (call-from-chat, call-from-calendar, DSGVO consent flow, recording to MinIO)
- [ ] 08-04: Meeting management (scheduling, agenda, lobby, notes, post-meeting summary, action items, CRM linking)
- [ ] 08-05: Emoji reactions + presence system (message reactions, WebSocket presence tracking, status indicators)

### Phase 9: Security & Compliance
**Goal**: The Hub meets enterprise security requirements and DSGVO compliance obligations, with multi-language support for the Swiss market
**Depends on**: Phase 8 (all user-facing modules exist for audit coverage), Phase 1 (Auth -- extend with 2FA)
**Requirements**: SEC-01, SEC-02, SEC-03, SEC-04, SEC-05, SEC-06, SEC-07, SEC-08, SEC-09, SEC-10, SEC-11
**Success Criteria** (what must be TRUE):
  1. User can enable TOTP-based 2FA and admin can enforce it for all users or specific roles
  2. All security-relevant actions are logged in a tamper-evident audit log searchable by admin
  3. User can request DSGVO data export containing all their personal data in machine-readable format
  4. Admin can execute right-to-erasure with cascading anonymization across all modules
  5. Admin can view and terminate active sessions with device/IP information
  6. Sensitive configuration (API keys, SMTP passwords) is stored encrypted at rest in a vault
  7. User can switch the UI between DE, FR, IT, and EN languages at runtime
**Plans**: 4 plans (estimated)

Plans:
- [ ] 09-01: Two-factor authentication (TOTP setup, verification, enforcement, recovery codes)
- [ ] 09-02: Audit log + session management (tamper-evident log, search/filter UI, session listing/termination)
- [ ] 09-03: DSGVO compliance (data export pipeline, data deletion with cascading anonymization, consent management)
- [ ] 09-04: Secret vault + password policies + i18n framework (encrypted config storage, password rules, locale framework with DE/FR/IT/EN)

### Phase 10: Email Integration
**Goal**: Users can send and receive email within the Hub without switching to an external email client, with automatic CRM context, contact import/export, and two-level contact visibility
**Depends on**: Phase 6 (PM -- Work service exists for email sub-domain), Phase 4 (Notifications -- new email alerts)
**Requirements**: MAIL-01, MAIL-02, MAIL-03, MAIL-04, MAIL-05, MAIL-06, MAIL-07, MAIL-08, MAIL-09, CRM-01, CRM-02, CRM-03
**Success Criteria** (what must be TRUE):
  1. User can compose and send an email via their configured SMTP server from within the Hub
  2. User can receive and read emails synced from their IMAP server, with read/unread status synced bidirectionally
  3. Emails are automatically linked to matching CRM contacts by email address, showing email history on contact profiles
  4. Related emails are grouped into threaded conversations based on References/In-Reply-To headers
  5. User can send and receive email attachments, with files stored in MinIO
  6. User can manage their HTML email signature including Impressum (legally required in DACH business email)
  7. User can reply, reply-all, and forward emails with proper thread preservation
  8. User can navigate IMAP folder structure (Inbox, Sent, Drafts, custom folders) within the Hub
  9. User can import contacts from CSV/vCard and export contacts to CSV/vCard with field selection
  10. Contacts support company-shared and personal visibility levels with admin override
**Plans**: 4 plans (estimated)

Plans:
- [ ] 10-01: IMAP sync engine (connection management, folder mapping, incremental sync, read status bidirectional sync)
- [ ] 10-02: SMTP send + threading (compose, send, reply/reply-all/forward, thread reconstruction, attachments via MinIO)
- [ ] 10-03: Email UI + CRM integration (inbox view, folder navigation, threaded conversation view, signature editor, auto-link to CRM contacts)
- [ ] 10-04: Contact import/export + two-level contacts (CSV/vCard import with field mapping, export, shared vs. personal contact visibility)

### Phase 11: Documents & Files
**Goal**: Users can manage, share, and find documents and files across the entire Hub from a central file manager, with a global search spanning all modules
**Depends on**: Phase 3 (Chat -- MinIO file infrastructure), Phase 6 (PM -- task file attachments)
**Requirements**: DOC-01, DOC-02, DOC-03, DOC-04, DOC-05, DOC-06, DOC-07, DOC-08, DOC-09, DOC-10
**Success Criteria** (what must be TRUE):
  1. User can browse files in a folder hierarchy with breadcrumb navigation
  2. User can upload files via drag-and-drop with progress indication and multi-file support
  3. User can preview PDF, images, and text files inline without downloading
  4. User can upload new file versions and revert to previous versions
  5. User can share files/folders with specific team members with read/write permissions
  6. User can search files by name, content, and tags across the entire file store
  7. User can search across ALL modules (CRM, PM, Chat, Email, Files) from a single global search bar with unified ranked results
  8. Chat file attachments are accessible through the central file manager and subject to per-user/per-role access controls
**Plans**: 4 plans (estimated)

Plans:
- [ ] 11-01: File service + folder structure (folder CRUD, file metadata, MinIO integration, permission model)
- [ ] 11-02: File UI + preview + versioning (folder browser, drag-drop upload, inline preview, version history)
- [ ] 11-03: File search + tagging (full-text extraction, tag CRUD, search indexing)
- [ ] 11-04: Global search (unified search across CRM, PM, Chat, Email, Files with ranked results)

### Phase 12: Finance Module
**Goal**: Users can create legally compliant quotes and invoices, track payments, and export to their Steuerberater -- replacing standalone invoicing tools
**Depends on**: Phase 2 (CRM -- deal-to-quote flow), Phase 4 (Notifications -- overdue invoice alerts)
**Requirements**: FIN-01, FIN-02, FIN-03, FIN-04, FIN-05, FIN-06, FIN-07
**Success Criteria** (what must be TRUE):
  1. User can create a quote (Angebot) with line items and tax calculation, and generate a PDF
  2. User can create an invoice (Rechnung) that is GoBD-compliant: immutable once sent, sequentially numbered, containing all legally required fields (Pflichtangaben)
  3. System correctly calculates MwSt/USt at 19% standard, 7% reduced, 0% for EU B2B Reverse Charge, and supports Kleinunternehmerregelung
  4. User can track payment status per invoice (draft, sent, overdue, paid, cancelled) and see a dashboard overview
  5. User can convert a CRM deal to a quote and then to an invoice in a seamless multi-step flow
  6. User can export a Buchungsstapel in DATEV-compatible CSV format for their Steuerberater
  7. User can create credit notes (Gutschriften) that properly reference the original invoice
**Plans**: TBD

Plans:
- [ ] 12-01: Biz service foundation + document models (immutable invoice data model, sequential numbering, GoBD constraints, quote/invoice/credit note CRUD)
- [ ] 12-02: Tax calculation + PDF generation (MwSt/USt rules, Reverse Charge, Kleinunternehmer, PDF templates with Pflichtangaben)
- [ ] 12-03: Finance workflows + export (deal-to-quote-to-invoice flow, payment tracking, overdue alerts, DATEV Buchungsstapel CSV export)

### Phase 13: HR Module
**Goal**: Employees can manage leave, track time, and access HR documents within the Hub, fully compliant with German labor law
**Depends on**: Phase 7 (Calendar -- absence calendar integration), Phase 12 (Finance -- Biz service exists for HR sub-domain)
**Requirements**: HR-01, HR-02, HR-03, HR-04, HR-05, HR-06, HR-07
**Success Criteria** (what must be TRUE):
  1. Employee can submit a leave request and their manager receives it for approval, with the full request/approve/reject workflow
  2. System correctly calculates leave balance per BUrlG (20-30 days based on contract, part-time pro-rata, carryover to March 31)
  3. Team absence calendar shows who is out when, integrated with the main calendar module
  4. Employee can clock in and out for time tracking, with daily and weekly hour summaries
  5. Time tracking enforces ArbZG rules: warns at 8h, blocks at 10h daily, enforces 11h rest between shifts, and requires breaks
  6. Employee profiles include department, position, and contract type, with access-controlled document storage (contracts, Zeugnisse)
  7. Sick leave can be recorded with AU (doctor's note) upload required after 3 consecutive days
**Plans**: TBD

Plans:
- [ ] 13-01: Leave management (request/approval workflow, BUrlG balance calculation, absence calendar integration)
- [ ] 13-02: Time tracking (clock in/out, ArbZG rule enforcement, daily/weekly summaries, break validation)
- [ ] 13-03: Employee profiles + documents (HR data extension of auth users, document storage in MinIO, sick leave with AU upload)

### Phase 14: Integration - CalDAV/CardDAV
**Goal**: External calendar and contact clients (Outlook, Thunderbird, macOS) can sync bidirectionally with KMU Hub
**Depends on**: Phase 7 (Calendar -- calendar data model), Phase 2 (CRM -- contact data model)
**Requirements**: INT-01, INT-02, INT-03
**Success Criteria** (what must be TRUE):
  1. User can add their KMU Hub calendar to Outlook/Thunderbird via CalDAV URL and see events sync bidirectionally
  2. User can add their KMU Hub contacts to an external client via CardDAV URL
  3. Sync is authenticated per user with proper access control
**Plans**: 2 plans (estimated)

Plans:
- [ ] 14-01: CalDAV server (go-webdav integration, event CRUD via CalDAV protocol, auth, ACL)
- [ ] 14-02: CardDAV server + sync testing (contact CRUD via CardDAV, multi-client testing)

### Phase 15: Integration - Teams & Slack
**Goal**: KMU Hub notifications and basic interactions flow to/from Microsoft Teams and Slack
**Depends on**: Phase 4 (Notifications -- notification infrastructure), Phase 9 (Security -- webhook secret management)
**Requirements**: INT-04, INT-05, INT-06
**Success Criteria** (what must be TRUE):
  1. Admin can configure Teams/Slack webhook and KMU Hub notifications appear in the configured channel
  2. Users can perform basic interactions (acknowledge, respond) from Teams/Slack back to KMU Hub
**Plans**: 2 plans (estimated)

Plans:
- [ ] 15-01: Outbound webhooks (Teams + Slack webhook configuration, notification forwarding, message formatting)
- [ ] 15-02: Inbound interactions (Teams/Slack interactive messages, response routing back to KMU Hub)

### Phase 16: Integration - Bexio
**Goal**: Swiss SMBs can sync their CRM contacts and invoices with Bexio accounting software
**Depends on**: Phase 2 (CRM -- contacts), Phase 12 (Finance -- invoices)
**Requirements**: INT-07, INT-08, INT-09
**Success Criteria** (what must be TRUE):
  1. Admin can authenticate with Bexio via OAuth2
  2. Contacts sync bidirectionally between KMU Hub CRM and Bexio
  3. Invoices created in KMU Hub are pushed to Bexio
**Plans**: 2 plans (estimated)

Plans:
- [ ] 16-01: Bexio connector (OAuth2 flow, API client, contact sync engine)
- [ ] 16-02: Invoice sync + conflict resolution (invoice push to Bexio, field mapping, error handling)

### Phase 17: Integration - Abacus
**Goal**: Swiss SMBs can sync their CRM contacts and invoices with Abacus ERP/accounting software
**Depends on**: Phase 2 (CRM -- contacts), Phase 12 (Finance -- invoices)
**Requirements**: INT-10, INT-11, INT-12
**Success Criteria** (what must be TRUE):
  1. Admin can authenticate with Abacus ERP via API key or OAuth2
  2. Contacts sync bidirectionally between KMU Hub CRM and Abacus
  3. Invoices created in KMU Hub are pushed to Abacus
**Plans**: 2 plans (estimated)

Plans:
- [ ] 17-01: Abacus connector (OAuth2/API key flow, API client, contact sync engine)
- [ ] 17-02: Invoice sync + conflict resolution (invoice push to Abacus, field mapping, error handling)

### Phase 18: Integration - Run my Accounts
**Goal**: Swiss SMBs can sync their CRM contacts and financial documents with Run my Accounts cloud accounting
**Depends on**: Phase 2 (CRM -- contacts), Phase 12 (Finance -- invoices)
**Requirements**: INT-13, INT-14, INT-15
**Success Criteria** (what must be TRUE):
  1. Admin can authenticate with Run my Accounts API
  2. Contacts sync bidirectionally between KMU Hub CRM and Run my Accounts
  3. Financial documents sync from KMU Hub Finance to Run my Accounts
**Plans**: 2 plans (estimated)

Plans:
- [ ] 18-01: Run my Accounts connector (auth flow, API client, contact sync)
- [ ] 18-02: Financial document sync + error handling (document push, field mapping, reconciliation)

### Phase 19: Automation Engine
**Goal**: Users can automate repetitive workflows across all Hub modules using simple trigger-action rules
**Depends on**: Phases 4-18 (all modules emitting events via PostgreSQL LISTEN/NOTIFY)
**Requirements**: AUTO-01, AUTO-02, AUTO-03, AUTO-04, AUTO-05, AUTO-06
**Success Criteria** (what must be TRUE):
  1. User can create an automation rule like "When a deal moves to Won stage, create an invoice draft" using a trigger-action model
  2. System offers 10-15 pre-built triggers across modules (deal stage change, task completed, invoice overdue, leave approved, new email from CRM contact, etc.)
  3. System offers 8-10 pre-built actions across modules (send notification, create task, send email, update CRM field, create calendar event, etc.)
  4. User can add conditional logic with if/else branching and AND/OR operators (e.g., "only if deal value > 10000")
  5. User can view execution logs showing when each automation ran, what triggered it, what it did, and whether it succeeded or failed
  6. User can enable/disable automations without deleting them (pause and resume)
**Plans**: TBD

Plans:
- [ ] 19-01: Automation service foundation (event consumer via LISTEN/NOTIFY, workflow storage, trigger registry, action registry)
- [ ] 19-02: Workflow execution engine (condition evaluation with expr-lang, action execution via gRPC, execution logging, enable/disable)
- [ ] 19-03: Pre-built automations (10-15 triggers + 8-10 actions across all modules, testing, documentation)

### Phase 20: Plugin System
**Goal**: Admins can customize the Hub for their company's specific processes without modifying source code, and developers can extend it via sandboxed WASM plugins
**Depends on**: Phase 19 (Automation -- plugin triggers via workflow engine), all modules (stable extension points)
**Requirements**: PLUG-01, PLUG-02, PLUG-03, PLUG-04, PLUG-05, PLUG-06, PLUG-07
**Success Criteria** (what must be TRUE):
  1. Admin can customize the Hub via config-based settings (custom fields, validation rules, workflow rules) without writing any code
  2. System runs WASM plugins in a sandboxed wazero runtime with enforced memory limits, execution time limits, and no direct host access
  3. Admin can install and remove plugins, with each plugin declaring required permissions that need explicit admin approval
  4. Installed plugins show a configuration UI auto-rendered from their declared settings schema (JSON Schema-based)
  5. Each module provides defined extension points (before/after hooks on CRUD operations) that plugins can register for
  6. Plugins access Hub data through a versioned, rate-limited API -- never through direct database access
  7. Industry-specific configuration templates (Branchenvorlagen) are available for at least 3 common DACH industries (e.g., Handwerk, Beratung, Handel)
**Plans**: TBD

Plans:
- [ ] 20-01: Config-based customization framework (extend existing custom fields, add workflow rules, validation rules, UI for admin configuration)
- [ ] 20-02: WASM plugin runtime (wazero sandbox, host function API, plugin lifecycle management, permission model)
- [ ] 20-03: Extension points + plugin API (before/after hooks in all modules, versioned plugin API, rate limiting, SDK)
- [ ] 20-04: Industry templates (3+ Branchenvorlagen as config packages, example WASM plugins, documentation)

## Progress

**Execution Order:**
Phases execute in numeric order: 4 -> 5 -> 6 -> 7 -> 8 -> 9 -> 10 -> 11 -> 12 -> 13 -> 14 -> 15 -> 16 -> 17 -> 18 -> 19 -> 20
Decimal phases (if inserted) execute between their surrounding integers.

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 4. Notifications + Gateway | 3/3 | Complete | 2026-02-07 |
| 5. Desktop App Shell | 7/7 | Complete | 2026-02-08 |
| 6. Project Management | 10/10 | Complete | 2026-02-08 |
| 7. Calendar & Scheduling | 0/3 | Not started | - |
| 8. Video, Voice & Meetings | 0/5 | Not started | - |
| 9. Security & Compliance | 0/4 | Not started | - |
| 10. Email Integration | 0/4 | Not started | - |
| 11. Documents & Files | 0/4 | Not started | - |
| 12. Finance Module | 0/3 | Not started | - |
| 13. HR Module | 0/3 | Not started | - |
| 14. Integration: CalDAV/CardDAV | 0/2 | Not started | - |
| 15. Integration: Teams & Slack | 0/2 | Not started | - |
| 16. Integration: Bexio | 0/2 | Not started | - |
| 17. Integration: Abacus | 0/2 | Not started | - |
| 18. Integration: Run my Accounts | 0/2 | Not started | - |
| 19. Automation Engine | 0/3 | Not started | - |
| 20. Plugin System | 0/4 | Not started | - |

---
*Roadmap created: 2026-02-07*
*Phases 1-3 completed prior to GSD adoption*
*Last updated: 2026-02-08 after second feature gap analysis (expanded from 18 to 20 phases -- added Abacus, Run my Accounts, chat file permissions)*
