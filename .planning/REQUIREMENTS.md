# Requirements: KMU Hub

**Defined:** 2026-02-07
**Core Value:** Every employee completes their entire workday without opening another program

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Notifications

- [x] **NOTF-01**: User receives in-app notifications (bell icon with unread count, notification center)
- [x] **NOTF-02**: User receives desktop push notifications when Hub is in background (Electron)
- [x] **NOTF-03**: User can configure notification preferences per event type and channel

### Desktop App

- [x] **DESK-01**: Desktop app has sidebar navigation with module loading and cross-module routing
- [x] **DESK-02**: User can personalize their workspace with widgets and customizable dashboard layouts
- [x] **DESK-03**: Admin can configure role-based default dashboards (different presets for CEO, manager, office worker)
- [x] **DESK-04**: Desktop app provides basic offline functionality with local caching for recently accessed data

### Project Management

- [x] **PM-01**: User can create and manage tasks with assignees, due dates, priority levels, and customizable statuses
- [x] **PM-02**: User can organize tasks into projects with shared settings and member access
- [x] **PM-03**: User can view tasks in a sortable, filterable list view
- [x] **PM-04**: User can view and manage tasks on a Kanban board with drag-and-drop columns per status
- [x] **PM-05**: User can add threaded comments on tasks with @mentions (reusing chat infrastructure)
- [x] **PM-06**: User can attach files to tasks (reusing MinIO file storage)
- [x] **PM-07**: User can search and filter tasks across all projects
- [x] **PM-08**: User can link tasks to CRM entities (create task from deal, link task to contact)
- [x] **PM-09**: User can add custom fields to tasks per project (reusing CRM custom fields engine)
- [x] **PM-10**: Tasks can have dependencies (finish-to-start blocking) with visual indicators on blocked tasks
- [x] **PM-11**: Tasks support multi-level subtasks with nesting for breaking down large tasks
- [x] **PM-15**: Admin/manager can save a project as a template and create new projects from templates
- [x] **PM-16**: User can view a project timeline as a Gantt chart with task bars, dependency arrows, and date range navigation
- [x] **PM-17**: User can start/stop a timer on a task to track time spent, with manual time entry and per-task time summaries

### Calendar & Scheduling

- [ ] **CAL-01**: User has a personal calendar with day, week, and month views
- [ ] **CAL-02**: Teams have shared calendars visible to all team members
- [ ] **CAL-03**: User can create timed and all-day events with recurring support (RRULE/RFC 5545)
- [ ] **CAL-04**: User can invite colleagues to events and track RSVP responses (accept/decline/maybe)
- [ ] **CAL-05**: User can book meeting rooms and resources via dedicated resource calendars
- [ ] **CAL-06**: Creating a calendar meeting auto-generates a LiveKit video call link
- [ ] **CAL-07**: Calendar displays DACH public holidays per Bundesland/Kanton with week starting Monday

### Video & Voice Calls

- [x] **VID-01**: User can make 1:1 video calls via LiveKit
- [x] **VID-02**: User can join group video calls with up to 25 participants
- [x] **VID-03**: User can make audio-only calls (camera off)
- [x] **VID-04**: User can share their screen or a specific window during a call
- [x] **VID-05**: User can mute/unmute microphone and toggle camera during a call
- [x] **VID-06**: User can start a call directly from a chat channel or DM
- [x] **VID-07**: User can record calls with DSGVO-compliant participant consent (stored in MinIO)

### Meetings

- [x] **MEET-01**: User can schedule a meeting with agenda, attendees, time slot, and optional recurring schedule
- [x] **MEET-02**: Meeting attendees see a pre-meeting lobby with agenda, attendee list, and shared documents
- [x] **MEET-03**: During a meeting, participants can take shared or private notes linked to the meeting record
- [x] **MEET-04**: After a meeting, a summary record with attendees, duration, notes, and action items is created
- [x] **MEET-05**: Meeting records are linked to calendar events and optionally to CRM entities

### Chat Enhancements

- [x] **CHAT-01**: User can react to messages with emoji reactions (add, remove, reaction counts)
- [x] **CHAT-02**: Users see presence/online status of colleagues (online, away, offline, in a call) across the app

### Security & Compliance

- [ ] **SEC-01**: User can enable TOTP-based two-factor authentication via authenticator app
- [ ] **SEC-02**: Admin can enforce 2FA for all users or specific roles
- [ ] **SEC-03**: System maintains a tamper-evident audit log of all security-relevant actions (login, permission changes, data access, data export)
- [ ] **SEC-04**: Admin can view and search the audit log with date range, user, and action type filters
- [ ] **SEC-05**: User can request a DSGVO-compliant data export (all personal data in machine-readable format)
- [ ] **SEC-06**: Admin can execute DSGVO data deletion (right to erasure) with cascading anonymization across all modules
- [ ] **SEC-07**: Admin can view and terminate active user sessions with device/IP/last activity details
- [ ] **SEC-08**: System provides encrypted-at-rest secret vault for sensitive configuration (API keys, SMTP passwords)
- [ ] **SEC-09**: Admin can configure password policies (minimum length, complexity, expiration, re-use prevention)
- [ ] **SEC-10**: System supports IP allowlist/blocklist for admin access restriction
- [ ] **SEC-11**: Frontend implements i18n framework with DE/FR/IT/EN locale support and runtime language switching

### Email Integration

- [ ] **MAIL-01**: User can send emails from within the Hub via configured SMTP server
- [ ] **MAIL-02**: User can receive and read emails in the Hub via IMAP connection
- [ ] **MAIL-03**: Emails are automatically linked to CRM contacts by email address matching
- [ ] **MAIL-04**: Related emails are grouped into threaded conversations (References/In-Reply-To)
- [ ] **MAIL-05**: User can send and receive email attachments (stored in MinIO)
- [ ] **MAIL-06**: User can manage their HTML email signature with Impressum
- [ ] **MAIL-07**: User can reply, reply all, and forward emails preserving threading
- [ ] **MAIL-08**: User sees read/unread status synced with IMAP server
- [ ] **MAIL-09**: User can view and navigate email folders mapped from IMAP

### CRM Enhancements

- [ ] **CRM-01**: User can import contacts from CSV/vCard files with field mapping and duplicate detection preview
- [ ] **CRM-02**: User can export contacts to CSV/vCard format with field selection
- [ ] **CRM-03**: Contacts support two-level visibility: company-shared (visible to all) and personal (owner-only), with admin override

### Documents & Files

- [ ] **DOC-01**: User can browse files in a hierarchical folder structure with breadcrumb navigation
- [ ] **DOC-02**: User can upload files via drag-and-drop or file picker with progress indicator and multi-file support
- [ ] **DOC-03**: User can preview common file types inline (PDF, images, text, Markdown)
- [ ] **DOC-04**: System maintains file version history; user can upload new versions and revert to previous ones
- [ ] **DOC-05**: User can share files/folders with team members or specific users with read/write permissions
- [ ] **DOC-06**: User can search files by name, content (full-text extraction for PDF/text), and tags
- [ ] **DOC-07**: User can tag files with custom labels for organization and filtering
- [ ] **DOC-08**: Files can be linked to CRM entities, projects, and other modules via entity linking
- [ ] **DOC-09**: User can search across all modules (CRM, PM, Chat, Email, Files) from a single global search bar with unified ranked results
- [ ] **DOC-10**: Chat file attachments are accessible through the central file manager and subject to per-user/per-role access controls

### Finance Module

- [x] **FIN-01**: User can create quotes (Angebote) with line items, tax calculation, and PDF generation
- [x] **FIN-02**: User can create invoices (Rechnungen) compliant with GoBD (immutable once sent, sequential numbering, all Pflichtangaben)
- [x] **FIN-03**: System calculates MwSt/USt correctly (19% standard, 7% reduced, 0% Reverse Charge for EU B2B, Kleinunternehmerregelung)
- [x] **FIN-04**: User can track payment status per invoice (draft, sent, overdue, paid, cancelled)
- [x] **FIN-05**: User can convert a CRM deal to a quote and then to an invoice in a seamless flow
- [x] **FIN-06**: User can export Buchungsstapel in DATEV-compatible CSV format for Steuerberater
- [x] **FIN-07**: User can create credit notes (Gutschriften) referencing original invoices

### HR Module

- [x] **HR-01**: Employee can submit leave/vacation requests for manager approval with workflow
- [x] **HR-02**: System tracks leave balance with Urlaubsanspruch calculation (BUrlG-konform, pro-rata, carry-over)
- [ ] **HR-03**: Team absence calendar shows who is out when (integrated with main calendar)
- [x] **HR-04**: Employee can clock in/out for time tracking with daily and weekly summaries
- [x] **HR-05**: Time tracking enforces Arbeitszeitgesetz rules (max 8h/10h, 11h rest, break requirements)
- [ ] **HR-06**: Employee profiles include department, position, contract type, and access-controlled document storage
- [ ] **HR-07**: Sick leave recording with AU (doctor's note) upload after 3 days

### External Integrations

- [ ] **INT-01**: System exposes a CalDAV endpoint for bidirectional calendar sync with Outlook, Thunderbird, and macOS Calendar
- [ ] **INT-02**: System exposes a CardDAV endpoint for bidirectional contact sync with external clients
- [ ] **INT-03**: CalDAV/CardDAV supports per-user authenticated access with proper ACL
- [ ] **INT-04**: Admin can configure Microsoft Teams webhook for notification forwarding to a Teams channel
- [ ] **INT-05**: Admin can configure Slack webhook for notification forwarding to a Slack channel
- [ ] **INT-06**: Users can perform basic interactions (acknowledge, respond) from Teams/Slack back to KMU Hub
- [ ] **INT-07**: Admin can connect to Bexio API via OAuth2 authentication
- [ ] **INT-08**: Contacts sync bidirectionally between KMU Hub CRM and Bexio
- [ ] **INT-09**: Invoices sync from KMU Hub Finance to Bexio accounting
- [ ] **INT-10**: Admin can connect to Abacus ERP via API key or OAuth2 authentication
- [ ] **INT-11**: Contacts sync bidirectionally between KMU Hub CRM and Abacus
- [ ] **INT-12**: Invoices/financial documents sync from KMU Hub Finance to Abacus
- [ ] **INT-13**: Admin can connect to Run my Accounts API via authentication
- [ ] **INT-14**: Contacts sync bidirectionally between KMU Hub CRM and Run my Accounts
- [ ] **INT-15**: Financial documents sync from KMU Hub Finance to Run my Accounts

### Automation Engine

- [ ] **AUTO-01**: User can create automations using trigger-action model ("When X happens, do Y")
- [ ] **AUTO-02**: System provides 10-15 pre-built triggers across all modules (deal stage change, task complete, invoice overdue, etc.)
- [ ] **AUTO-03**: System provides 8-10 pre-built actions across all modules (send notification, create task, send email, update field, etc.)
- [ ] **AUTO-04**: User can add conditional logic (if/else with field-value conditions, AND/OR operators) to automations
- [ ] **AUTO-05**: User can view automation execution logs with timestamps, inputs/outputs, and success/failure status
- [ ] **AUTO-06**: User can enable/disable automations without deleting them

### Plugin System

- [ ] **PLUG-01**: Admin can customize the Hub via config-based settings (custom fields, workflows, validation rules) without code
- [ ] **PLUG-02**: System runs WASM plugins in sandboxed wazero runtime with memory limits, time limits, and no direct host access
- [ ] **PLUG-03**: Admin can install and remove plugins with declared permission models requiring admin approval
- [ ] **PLUG-04**: Installed plugins have a configuration UI auto-rendered from their declared settings schema
- [ ] **PLUG-05**: Each module provides defined extension points (before/after hooks on CRUD operations, custom validation, data transformation)
- [ ] **PLUG-06**: Plugins access Hub data through a versioned, rate-limited API (no direct database access)
- [ ] **PLUG-07**: Industry-specific configuration templates (Branchenvorlagen) are available for common DACH industries

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Project Management

- **PM-12**: Interactive Gantt chart editing (drag to resize/move task bars, create dependencies by drawing arrows)
- **PM-13**: Recurring tasks with RRULE-based recurrence
- **PM-14**: Workload view showing task distribution per person

### Calendar & Scheduling

- **CAL-08**: Free/busy availability view across all team members
- **CAL-09**: iCal import/export (.ics) for migration and external exchange
- **CAL-10**: CRM activity sync (calendar event <-> CRM meeting activity, bidirectional)
- **CAL-11**: PM task deadlines shown as overlay on calendar
- **CAL-13**: External booking page (Calendly-style) for clients

### Email Integration

- **MAIL-10**: Shared team inboxes (info@company.de managed by multiple people)
- **MAIL-11**: Email templates with CRM variable substitution
- **MAIL-12**: Email tracking (open/read receipts, DSGVO opt-in only)
- **MAIL-13**: Scheduled sending ("Send Monday at 9:00")
- **MAIL-14**: Auto-link emails to CRM deals by keyword/subject matching
- **MAIL-15**: BCC-to-CRM for logging emails from external clients

### Video & Voice

- **VID-08**: Breakout rooms for splitting large meetings into groups
- **VID-09**: Virtual backgrounds (client-side ML processing)
- **VID-10**: Noise suppression (client-side or LiveKit server-side)
- **VID-11**: CRM activity auto-log (call metadata linked to CRM contact)

### Meetings

- **MEET-06**: AI-powered meeting transcription and summary generation
- **MEET-07**: Automatic action item extraction from meeting notes

### Chat

- **CHAT-03**: Custom emoji upload and management
- **CHAT-04**: Threaded emoji reactions (react to specific thread replies)

### Security & Compliance

- **SEC-12**: Hardware security key support (FIDO2/WebAuthn) as 2FA method
- **SEC-13**: Single Sign-On via SAML/OIDC for enterprise customers
- **SEC-14**: Data Loss Prevention rules (block sharing of sensitive data patterns)

### HR Module

- **HR-08**: Company org chart (visual who-reports-to-whom)
- **HR-09**: Overtime tracking and compensation (Gleitzeitkonto)
- **HR-10**: Digital onboarding checklists (reusing PM task templates)
- **HR-11**: Mutterschutz/Elternzeit tracking with legal deadlines
- **HR-12**: DATEV-compatible time/absence export for payroll provider
- **HR-13**: Absence analytics (anonymized sick day trends, leave utilization)

### Finance Module

- **FIN-08**: Recurring invoices (monthly retainer billing)
- **FIN-09**: Payment reminders/Mahnwesen (automated 1st/2nd/3rd Mahnung)
- **FIN-10**: Multi-currency support (EUR/CHF/USD/GBP with exchange rates)
- **FIN-11**: Time tracking to invoice conversion (hours at rate -> line items)
- **FIN-12**: XRechnung/ZUGFeRD electronic invoice format
- **FIN-13**: Bank account connection via FinTS/HBCI for auto-reconciliation

### Documents & Files

- **DOC-11**: Fine-grained file sharing permissions (view/edit/download per user/role)
- **DOC-12**: File locking for collaborative editing prevention

### External Integrations

- **INT-16**: Generic webhook/REST connector for arbitrary external services
- **INT-17**: DATEV Unternehmen Online direct API integration (beyond CSV export)

### General

- **GEN-01**: Duplikat-Erkennung (fuzzy matching across contacts, companies)
- **GEN-02**: Kontakt-Gruppen (static and dynamic grouping of contacts for bulk operations)
- **GEN-03**: Arbeitsprofile (multiple user contexts with different sidebar/dashboard configs)
- **GEN-04**: Wetter-Widget (weather dashboard widget via external API)
- **GEN-05**: Auto-Save (draft auto-save for forms and editors with connection loss recovery)
- **GEN-06**: Organigramm (visual organization chart from HR hierarchy data)
- **GEN-07**: Arbeitsinteressen (employee skill/interest profiles for project matching)
- **GEN-08**: Ausgabenverwaltung (expense receipt upload, categorization, cost tracking)

### Automation Engine

- **AUTO-07**: Visual drag-and-drop workflow builder
- **AUTO-08**: Cross-module multi-step workflows (deal won -> create project -> schedule kickoff -> send invoice)
- **AUTO-09**: Scheduled automations (cron-based triggers)
- **AUTO-10**: Delay/wait steps in workflows
- **AUTO-11**: Inbound webhooks (external systems trigger Hub automations)
- **AUTO-12**: Outbound webhooks (Hub automations trigger external systems)

### Plugin System

- **PLUG-08**: Plugin marketplace with curated, reviewed plugins
- **PLUG-09**: Plugin SDK and documentation for third-party developers
- **PLUG-10**: Custom entity types defined by plugins (e.g., "Vehicles" for fleet management)
- **PLUG-11**: UI extension points (custom sidebar widgets, dashboard cards, detail tabs)
- **PLUG-12**: Plugin versioning with SemVer and breaking change controls

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Full document editor / Wiki | Separate product (Notion/Docs). Link/embed external docs instead. Simple knowledge base later. |
| Mobile app (React Native) | Desktop-first. Mobile after desktop daily-driver is proven. |
| Office suite replacement (Word/Excel) | Integrate, don't rebuild. Connectors for customer-preferred tools. |
| PSTN/phone integration | SIP gateway is extremely complex. VoIP only in v1. |
| Webinar mode (1000+ viewers) | Different architecture. Group calls to 25 sufficient for SMBs. |
| Live transcription / AI summaries | Requires speech-to-text + summarization pipeline. Out of v1 scope. |
| Full bookkeeping (Buchhaltung) | Legal complexity (HGB). Export to DATEV instead. |
| Payroll processing (Lohnabrechnung) | Legal minefield (taxes, social security). Export hours to provider. |
| Tax filing (Steuererklarung) | Legal liability. Steuerberater territory. |
| Full ERP / inventory management | Different product category (Warenwirtschaft). Basic product catalog only. |
| POS / cash register (Kassensystem) | TSE hardware requirement. Specialized retail tool. |
| Email marketing / bulk sends | Deliverability, CAN-SPAM. Integrate with Mailchimp/Brevo. |
| Spam filtering | Gmail/Outlook already handle this. Rely on upstream provider. |
| Custom email hosting (MTA) | Running mail servers is ops nightmare. IMAP/SMTP proxy only. |
| AI scheduling assistant | Complex optimization. Manual availability first. |
| Code-based automation scripting | Security nightmare. WASM plugins for complex logic. |
| General integration platform (Zapier clone) | 5000+ connectors is a product. Webhooks + case-by-case connectors. |
| Open plugin marketplace | Quality control, security review. Curated first-party only in v1. |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| NOTF-01 | Phase 4 | Complete |
| NOTF-02 | Phase 4 | Complete |
| NOTF-03 | Phase 4 | Complete |
| DESK-01 | Phase 5 | Complete |
| DESK-02 | Phase 5 | Complete |
| DESK-03 | Phase 5 | Complete |
| DESK-04 | Phase 5 | Complete |
| PM-01 | Phase 6 | Pending |
| PM-02 | Phase 6 | Pending |
| PM-03 | Phase 6 | Pending |
| PM-04 | Phase 6 | Pending |
| PM-05 | Phase 6 | Pending |
| PM-06 | Phase 6 | Pending |
| PM-07 | Phase 6 | Pending |
| PM-08 | Phase 6 | Pending |
| PM-09 | Phase 6 | Pending |
| PM-10 | Phase 6 | Pending |
| PM-11 | Phase 6 | Pending |
| PM-15 | Phase 6 | Pending |
| PM-16 | Phase 6 | Pending |
| PM-17 | Phase 6 | Pending |
| CAL-01 | Phase 7 | Pending |
| CAL-02 | Phase 7 | Pending |
| CAL-03 | Phase 7 | Pending |
| CAL-04 | Phase 7 | Pending |
| CAL-05 | Phase 7 | Pending |
| CAL-06 | Phase 7 | Pending |
| CAL-07 | Phase 7 | Pending |
| VID-01 | Phase 8 | Complete |
| VID-02 | Phase 8 | Complete |
| VID-03 | Phase 8 | Complete |
| VID-04 | Phase 8 | Complete |
| VID-05 | Phase 8 | Complete |
| VID-06 | Phase 8 | Complete |
| VID-07 | Phase 8 | Complete |
| MEET-01 | Phase 8 | Complete |
| MEET-02 | Phase 8 | Complete |
| MEET-03 | Phase 8 | Complete |
| MEET-04 | Phase 8 | Complete |
| MEET-05 | Phase 8 | Complete |
| CHAT-01 | Phase 8 | Complete |
| CHAT-02 | Phase 8 | Complete |
| SEC-01 | Phase 9 | Pending |
| SEC-02 | Phase 9 | Pending |
| SEC-03 | Phase 9 | Pending |
| SEC-04 | Phase 9 | Pending |
| SEC-05 | Phase 9 | Pending |
| SEC-06 | Phase 9 | Pending |
| SEC-07 | Phase 9 | Pending |
| SEC-08 | Phase 9 | Pending |
| SEC-09 | Phase 9 | Pending |
| SEC-10 | Phase 9 | Pending |
| SEC-11 | Phase 9 | Pending |
| MAIL-01 | Phase 10 | Pending |
| MAIL-02 | Phase 10 | Pending |
| MAIL-03 | Phase 10 | Pending |
| MAIL-04 | Phase 10 | Pending |
| MAIL-05 | Phase 10 | Pending |
| MAIL-06 | Phase 10 | Pending |
| MAIL-07 | Phase 10 | Pending |
| MAIL-08 | Phase 10 | Pending |
| MAIL-09 | Phase 10 | Pending |
| CRM-01 | Phase 10 | Pending |
| CRM-02 | Phase 10 | Pending |
| CRM-03 | Phase 10 | Pending |
| DOC-01 | Phase 11 | Pending |
| DOC-02 | Phase 11 | Pending |
| DOC-03 | Phase 11 | Pending |
| DOC-04 | Phase 11 | Pending |
| DOC-05 | Phase 11 | Pending |
| DOC-06 | Phase 11 | Pending |
| DOC-07 | Phase 11 | Pending |
| DOC-08 | Phase 11 | Pending |
| DOC-09 | Phase 11 | Pending |
| DOC-10 | Phase 11 | Pending |
| FIN-01 | Phase 12 | Complete |
| FIN-02 | Phase 12 | Complete |
| FIN-03 | Phase 12 | Complete |
| FIN-04 | Phase 12 | Complete |
| FIN-05 | Phase 12 | Complete |
| FIN-06 | Phase 12 | Complete |
| FIN-07 | Phase 12 | Complete |
| HR-01 | Phase 13 | Complete |
| HR-02 | Phase 13 | Complete |
| HR-03 | Phase 13 | Pending |
| HR-04 | Phase 13 | Complete |
| HR-05 | Phase 13 | Complete |
| HR-06 | Phase 13 | Pending |
| HR-07 | Phase 13 | Pending |
| INT-01 | Phase 14 | Pending |
| INT-02 | Phase 14 | Pending |
| INT-03 | Phase 14 | Pending |
| INT-04 | Phase 15 | Pending |
| INT-05 | Phase 15 | Pending |
| INT-06 | Phase 15 | Pending |
| INT-07 | Phase 16 | Pending |
| INT-08 | Phase 16 | Pending |
| INT-09 | Phase 16 | Pending |
| INT-10 | Phase 17 | Pending |
| INT-11 | Phase 17 | Pending |
| INT-12 | Phase 17 | Pending |
| INT-13 | Phase 18 | Pending |
| INT-14 | Phase 18 | Pending |
| INT-15 | Phase 18 | Pending |
| AUTO-01 | Phase 19 | Pending |
| AUTO-02 | Phase 19 | Pending |
| AUTO-03 | Phase 19 | Pending |
| AUTO-04 | Phase 19 | Pending |
| AUTO-05 | Phase 19 | Pending |
| AUTO-06 | Phase 19 | Pending |
| PLUG-01 | Phase 20 | Pending |
| PLUG-02 | Phase 20 | Pending |
| PLUG-03 | Phase 20 | Pending |
| PLUG-04 | Phase 20 | Pending |
| PLUG-05 | Phase 20 | Pending |
| PLUG-06 | Phase 20 | Pending |
| PLUG-07 | Phase 20 | Pending |

**Coverage:**
- v1 requirements: 117 total
- Mapped to phases: 117
- Unmapped: 0

---
*Requirements defined: 2026-02-07*
*Last updated: 2026-02-11 after Phase 8 completion (VID-01..07, MEET-01..05, CHAT-01..02 marked Complete)*
