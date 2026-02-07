# Requirements: KMU Hub

**Defined:** 2026-02-07
**Core Value:** Every employee completes their entire workday without opening another program

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Notifications

- [ ] **NOTF-01**: User receives in-app notifications (bell icon with unread count, notification center)
- [ ] **NOTF-02**: User receives desktop push notifications when Hub is in background (Electron)
- [ ] **NOTF-03**: User can configure notification preferences per event type and channel

### Desktop App

- [ ] **DESK-01**: Desktop app has sidebar navigation with module loading and cross-module routing
- [ ] **DESK-02**: User can personalize their workspace with widgets and customizable dashboard layouts
- [ ] **DESK-03**: Admin can configure role-based default dashboards (different presets for CEO, manager, office worker)
- [ ] **DESK-04**: Desktop app provides basic offline functionality with local caching for recently accessed data

### Project Management

- [ ] **PM-01**: User can create and manage tasks with assignees, due dates, priority levels, and customizable statuses
- [ ] **PM-02**: User can organize tasks into projects with shared settings and member access
- [ ] **PM-03**: User can view tasks in a sortable, filterable list view
- [ ] **PM-04**: User can view and manage tasks on a Kanban board with drag-and-drop columns per status
- [ ] **PM-05**: User can add threaded comments on tasks with @mentions (reusing chat infrastructure)
- [ ] **PM-06**: User can attach files to tasks (reusing MinIO file storage)
- [ ] **PM-07**: User can search and filter tasks across all projects
- [ ] **PM-08**: User can link tasks to CRM entities (create task from deal, link task to contact)
- [ ] **PM-09**: User can add custom fields to tasks per project (reusing CRM custom fields engine)

### Calendar & Scheduling

- [ ] **CAL-01**: User has a personal calendar with day, week, and month views
- [ ] **CAL-02**: Teams have shared calendars visible to all team members
- [ ] **CAL-03**: User can create timed and all-day events with recurring support (RRULE/RFC 5545)
- [ ] **CAL-04**: User can invite colleagues to events and track RSVP responses (accept/decline/maybe)
- [ ] **CAL-05**: User can book meeting rooms and resources via dedicated resource calendars
- [ ] **CAL-06**: Creating a calendar meeting auto-generates a LiveKit video call link
- [ ] **CAL-07**: Calendar displays DACH public holidays per Bundesland/Kanton with week starting Monday

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

### Video & Voice Calls

- [ ] **VID-01**: User can make 1:1 video calls via LiveKit
- [ ] **VID-02**: User can join group video calls with up to 25 participants
- [ ] **VID-03**: User can make audio-only calls (camera off)
- [ ] **VID-04**: User can share their screen or a specific window during a call
- [ ] **VID-05**: User can mute/unmute microphone and toggle camera during a call
- [ ] **VID-06**: User can start a call directly from a chat channel or DM
- [ ] **VID-07**: User can record calls with DSGVO-compliant participant consent (stored in MinIO)

### HR Module

- [ ] **HR-01**: Employee can submit leave/vacation requests for manager approval with workflow
- [ ] **HR-02**: System tracks leave balance with Urlaubsanspruch calculation (BUrlG-konform, pro-rata, carry-over)
- [ ] **HR-03**: Team absence calendar shows who is out when (integrated with main calendar)
- [ ] **HR-04**: Employee can clock in/out for time tracking with daily and weekly summaries
- [ ] **HR-05**: Time tracking enforces Arbeitszeitgesetz rules (max 8h/10h, 11h rest, break requirements)
- [ ] **HR-06**: Employee profiles include department, position, contract type, and access-controlled document storage
- [ ] **HR-07**: Sick leave recording with AU (doctor's note) upload after 3 days

### Finance Module

- [ ] **FIN-01**: User can create quotes (Angebote) with line items, tax calculation, and PDF generation
- [ ] **FIN-02**: User can create invoices (Rechnungen) compliant with GoBD (immutable once sent, sequential numbering, all Pflichtangaben)
- [ ] **FIN-03**: System calculates MwSt/USt correctly (19% standard, 7% reduced, 0% Reverse Charge for EU B2B, Kleinunternehmerregelung)
- [ ] **FIN-04**: User can track payment status per invoice (draft, sent, overdue, paid, cancelled)
- [ ] **FIN-05**: User can convert a CRM deal to a quote and then to an invoice in a seamless flow
- [ ] **FIN-06**: User can export Buchungsstapel in DATEV-compatible CSV format for Steuerberater
- [ ] **FIN-07**: User can create credit notes (Gutschriften) referencing original invoices

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

- **PM-10**: Task dependencies (finish-to-start blocking)
- **PM-11**: Subtasks and checklists for breaking down large tasks
- **PM-12**: Timeline/Gantt view for visual project planning
- **PM-13**: Recurring tasks with RRULE-based recurrence
- **PM-14**: Workload view showing task distribution per person
- **PM-15**: Task templates for standardized processes

### Calendar & Scheduling

- **CAL-08**: Free/busy availability view across all team members
- **CAL-09**: iCal import/export (.ics) for migration and external exchange
- **CAL-10**: CRM activity sync (calendar event <-> CRM meeting activity, bidirectional)
- **CAL-11**: PM task deadlines shown as overlay on calendar
- **CAL-12**: CalDAV server for external client sync (Outlook, Thunderbird, macOS Calendar)
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
| NOTF-01 | Phase 4 | Pending |
| NOTF-02 | Phase 4 | Pending |
| NOTF-03 | Phase 4 | Pending |
| DESK-01 | Phase 5 | Pending |
| DESK-02 | Phase 5 | Pending |
| DESK-03 | Phase 5 | Pending |
| DESK-04 | Phase 5 | Pending |
| PM-01 | Phase 6 | Pending |
| PM-02 | Phase 6 | Pending |
| PM-03 | Phase 6 | Pending |
| PM-04 | Phase 6 | Pending |
| PM-05 | Phase 6 | Pending |
| PM-06 | Phase 6 | Pending |
| PM-07 | Phase 6 | Pending |
| PM-08 | Phase 6 | Pending |
| PM-09 | Phase 6 | Pending |
| CAL-01 | Phase 7 | Pending |
| CAL-02 | Phase 7 | Pending |
| CAL-03 | Phase 7 | Pending |
| CAL-04 | Phase 7 | Pending |
| CAL-05 | Phase 7 | Pending |
| CAL-06 | Phase 7 | Pending |
| CAL-07 | Phase 7 | Pending |
| MAIL-01 | Phase 9 | Pending |
| MAIL-02 | Phase 9 | Pending |
| MAIL-03 | Phase 9 | Pending |
| MAIL-04 | Phase 9 | Pending |
| MAIL-05 | Phase 9 | Pending |
| MAIL-06 | Phase 9 | Pending |
| MAIL-07 | Phase 9 | Pending |
| MAIL-08 | Phase 9 | Pending |
| MAIL-09 | Phase 9 | Pending |
| VID-01 | Phase 8 | Pending |
| VID-02 | Phase 8 | Pending |
| VID-03 | Phase 8 | Pending |
| VID-04 | Phase 8 | Pending |
| VID-05 | Phase 8 | Pending |
| VID-06 | Phase 8 | Pending |
| VID-07 | Phase 8 | Pending |
| HR-01 | Phase 11 | Pending |
| HR-02 | Phase 11 | Pending |
| HR-03 | Phase 11 | Pending |
| HR-04 | Phase 11 | Pending |
| HR-05 | Phase 11 | Pending |
| HR-06 | Phase 11 | Pending |
| HR-07 | Phase 11 | Pending |
| FIN-01 | Phase 10 | Pending |
| FIN-02 | Phase 10 | Pending |
| FIN-03 | Phase 10 | Pending |
| FIN-04 | Phase 10 | Pending |
| FIN-05 | Phase 10 | Pending |
| FIN-06 | Phase 10 | Pending |
| FIN-07 | Phase 10 | Pending |
| AUTO-01 | Phase 12 | Pending |
| AUTO-02 | Phase 12 | Pending |
| AUTO-03 | Phase 12 | Pending |
| AUTO-04 | Phase 12 | Pending |
| AUTO-05 | Phase 12 | Pending |
| AUTO-06 | Phase 12 | Pending |
| PLUG-01 | Phase 13 | Pending |
| PLUG-02 | Phase 13 | Pending |
| PLUG-03 | Phase 13 | Pending |
| PLUG-04 | Phase 13 | Pending |
| PLUG-05 | Phase 13 | Pending |
| PLUG-06 | Phase 13 | Pending |
| PLUG-07 | Phase 13 | Pending |

**Coverage:**
- v1 requirements: 66 total
- Mapped to phases: 66
- Unmapped: 0

---
*Requirements defined: 2026-02-07*
*Last updated: 2026-02-07 after roadmap creation*
