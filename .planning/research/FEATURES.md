# Feature Landscape

**Domain:** All-in-one workplace platform for DACH SMBs (5-200 employees)
**Researched:** 2026-02-07
**Confidence:** MEDIUM (based on training data, no live web verification available)

**Note on sources:** WebSearch and WebFetch were unavailable during this research session. All findings are based on training data knowledge of these product categories as of early 2025. Confidence levels reflect this limitation. The domains researched (PM tools, calendars, email, video, HR, finance, automation, plugins) are well-established and evolve incrementally, so findings should be largely current. DACH-specific legal requirements (GoBD, DSGVO, DATEV) are stable regulatory frameworks unlikely to have changed fundamentally.

---

## Module 1: Project Management

### Competitive Landscape

Key players: Jira (enterprise), Asana (mid-market), Monday.com (SMB-friendly), Linear (developer-focused), ClickUp (feature-rich), Notion (lightweight). For DACH SMBs, Monday.com and Asana dominate because Jira is too complex and Linear is too developer-centric.

### Table Stakes

Features users expect. Missing = "this project management is incomplete."

| Feature | Why Expected | Complexity | Dependencies | Notes |
|---------|-------------|------------|--------------|-------|
| Tasks with assignees, due dates, priority | Core unit of work | Medium | Auth (users), CRM (contacts optional) | Support single and multiple assignees |
| Task statuses (customizable) | Every team has different workflows | Medium | None | Default: To Do, In Progress, Done. Custom statuses per project. |
| Projects as containers for tasks | Organizational hierarchy | Low | None | Project = collection of tasks with shared settings |
| List view | Most basic way to see tasks | Low (frontend) | Desktop app | Sortable, filterable table of tasks |
| Board/Kanban view | Visual workflow management | Medium (frontend) | Desktop app | Drag-and-drop columns representing statuses |
| Comments on tasks | Discussion in context | Low | Chat (shared comment engine) | Threaded comments, @mentions reuse chat infrastructure |
| File attachments on tasks | Reference documents | Low | File sharing (MinIO, reuse chat file infra) | Reuse existing file upload/storage |
| Due date reminders/notifications | People forget deadlines | Medium | Notification system | In-app + optional email |
| Task search and filtering | Find tasks across projects | Low | Existing search infrastructure | Extend PostgreSQL FTS to task content |
| Activity log per task | Who did what when | Low | Audit infrastructure | Status changes, comments, assignments logged |

### Differentiators

| Feature | Value Proposition | Complexity | Dependencies | Notes |
|---------|-------------------|------------|--------------|-------|
| CRM-linked tasks | "Create task from deal" or "link task to contact" | Medium | CRM module | Unique to all-in-one: tasks reference CRM entities directly |
| Timeline/Gantt view | Visual project planning with dependencies | High (frontend) | Desktop app | Important for construction, event planning SMBs |
| Task dependencies (finish-to-start) | Block tasks until predecessors complete | Medium | None | Start simple: finish-to-start only. Other types later. |
| Recurring tasks | Repeating work (monthly reports, weekly meetings) | Medium | Scheduler/cron infrastructure | RRULE-based recurrence |
| Time tracking on tasks | "How long did this take?" | Medium | HR module (optional link) | Built-in timer + manual entry. Links to HR/billing later. |
| Subtasks/checklists | Break down large tasks | Low | None | Two levels: subtasks (full tasks) and checklists (simple checkboxes) |
| Task templates | Standardize recurring processes | Medium | None | "New Employee Onboarding" template creates 15 predefined tasks |
| Workload view | See who is overloaded | Medium (frontend) | User management | Calendar-style view of task distribution per person |
| Automation triggers | "When status changes to Done, notify manager" | Medium | Automation engine | Depends on automation module |
| Custom fields on tasks | Extend task data per project/company | Low | Existing custom fields engine | Reuse CRM custom fields infrastructure |

### Anti-Features (Do NOT Build in v1)

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Agile/Scrum tooling (sprints, story points, velocity) | Too developer-specific, confuses non-technical SMB users | Simple status-based workflows. Add Agile as optional plugin later. |
| Resource management/capacity planning | Enterprise complexity, rarely used by SMBs under 50 people | Workload view is sufficient. Full resource planning is Phase 2+. |
| Multiple project methodologies | Overwhelming choice. Monday.com succeeds by being opinionated. | One flexible system: statuses + optional dependencies. Not "choose Scrum vs Kanban vs Waterfall." |
| Built-in Gantt chart editor | Extremely complex to build well. Bad Gantt is worse than no Gantt. | Start with timeline view (read-only visualization). Interactive Gantt is Phase 2+. |
| Portfolio management | Multi-project dashboards are enterprise features | Single project view first. Cross-project reporting later. |

### DACH-Specific Requirements

| Requirement | Rationale | Complexity |
|-------------|-----------|------------|
| German-language task templates | DACH SMBs operate in German. Default templates in German. | Low |
| Betriebsrat visibility controls | Works councils may require limits on who sees task assignments/time | Medium |
| Feiertage (public holidays) per Bundesland | Due date calculations must respect regional holidays (16 Bundeslaender + AT + CH cantons) | Medium |

---

## Module 2: Calendar & Scheduling

### Competitive Landscape

Key players: Google Calendar (dominant), Outlook/Exchange (enterprise), Cal.com (open-source scheduling), Calendly (appointment booking). For DACH SMBs, Outlook dominates in corporate settings, Google Calendar in smaller companies. Cal.com is the open-source reference for scheduling features.

### Table Stakes

| Feature | Why Expected | Complexity | Dependencies | Notes |
|---------|-------------|------------|--------------|-------|
| Personal calendar per user | Every user needs to see their schedule | Medium | Auth (users) | Day, week, month views |
| Shared/team calendars | See team availability | Medium | Auth (teams/roles) | Read access by default, write access configurable |
| Event creation (title, time, location, description) | Basic calendar function | Low | None | Support all-day and timed events |
| Recurring events | Weekly meetings, monthly reviews | Medium | None | RRULE (RFC 5545) for recurrence rules |
| Event invitations + RSVP | Invite colleagues to meetings | Medium | Notification system | Accept/Decline/Maybe, attendance tracking |
| Reminders/notifications | Don't miss meetings | Low | Notification system | 15min, 1hr, 1 day before. Customizable per event. |
| Availability/free-busy view | See when people are available | Medium | None | Aggregate across calendars per user |
| Drag-and-drop event editing | Reschedule by dragging | Medium (frontend) | Desktop app | Standard UX expectation from Google Calendar |
| Color-coded categories | Visual distinction between event types | Low | None | Meeting, deadline, out-of-office, personal |
| Timezone support | Multi-location teams, international clients | Medium | None | IANA timezone database. Display in user's local timezone. |

### Differentiators

| Feature | Value Proposition | Complexity | Dependencies | Notes |
|---------|-------------------|------------|--------------|-------|
| CRM activity sync | "Meeting with contact X" auto-creates CRM activity | Medium | CRM activities module | Bidirectional: calendar event <-> CRM meeting activity |
| Room/resource booking | Book meeting rooms, equipment, vehicles | Medium | None | Resource calendars alongside personal calendars |
| External booking page | Clients book time with you (like Calendly) | High | Public-facing endpoint | Huge differentiator: replaces Calendly/Cal.com subscription |
| Task due dates on calendar | See project deadlines alongside meetings | Low | Project management | Read-only overlay from PM tasks |
| Video call auto-link | "Create meeting" auto-generates LiveKit room | Low | Video module | One-click: creates event + video room link |
| CalDAV server | Sync with external clients (macOS Calendar, Thunderbird) | High | None | Makes KMU Hub a proper calendar server, not just internal |
| iCal export/import (.ics) | Exchange events with external parties | Medium | None | Standard interchange format. Import from Google/Outlook migration. |
| Working hours configuration | Define when each user is available | Low | None | Per-user: Mon-Fri 9-17, exceptions for part-time workers |

### Anti-Features (Do NOT Build in v1)

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Full CalDAV/CardDAV server | Extremely complex protocol to implement correctly. Radicale took years. | Start with iCal import/export. CalDAV is Phase 2+ or use existing server (Radicale) as embedded component. |
| Google Calendar / Outlook bidirectional sync | OAuth complexity, rate limits, conflict resolution nightmare | One-way import first. Bidirectional sync as integration connector later. |
| AI scheduling assistant | "Find the best time for 5 people" is complex optimization | Manual availability check first. AI scheduling is a future differentiator. |
| Public calendar widget | Embeddable calendar on external websites | External booking page covers the use case. Full widget is overkill. |

### DACH-Specific Requirements

| Requirement | Rationale | Complexity |
|-------------|-----------|------------|
| Feiertage per Bundesland/Kanton | Germany has 16 states with different holidays, Austria 9, Switzerland 26 cantons | Medium |
| Week starts on Monday | DACH convention (not Sunday like US) | Low |
| 24-hour time format default | DACH standard. Offer 12h as option. | Low |
| Bruckentage (bridge days) awareness | Days between holidays and weekends. Common PTO days in DACH. | Low |
| Betriebsurlaub (company shutdown) | Companies close entirely for periods (e.g., Christmas). Block calendar for all users. | Medium |

---

## Module 3: Email Integration

### Competitive Landscape

Key players: HubSpot (email tracking + CRM sync), Salesforce (email-to-lead), Front (shared inboxes), Hiver (Gmail-based helpdesk), Missive (team email). For DACH SMBs, Outlook/Exchange dominates corporate email. Many use web.de, GMX, or custom domain email via hosting providers.

### Table Stakes

| Feature | Why Expected | Complexity | Dependencies | Notes |
|---------|-------------|------------|--------------|-------|
| Send email from within Hub | Core "don't leave the app" promise | High | SMTP configuration per user/org | Must support custom SMTP servers (not just Gmail/Outlook) |
| Receive/read email in Hub | Complete email experience | High | IMAP/POP3 polling or push | IMAP IDLE for near-real-time, or polling interval |
| Email-to-CRM contact linking | See all emails with a contact in their CRM profile | Medium | CRM contacts | Match by email address. Manual linking as fallback. |
| Email threading/conversations | Group related emails together | Medium | None | References/In-Reply-To headers for threading |
| Attachments (send and receive) | Fundamental email feature | Medium | File storage (MinIO) | Store attachments in MinIO, link to email record |
| Signature management | Professional email appearance | Low | None | HTML signature editor per user |
| Reply, Reply All, Forward | Basic email actions | Medium | None | Preserve threading and headers correctly |
| Unread/read status | Know what needs attention | Low | None | Sync read status back to IMAP if possible |
| Folder/label view | Organize emails | Medium | None | Map IMAP folders. Support custom labels in Hub. |

### Differentiators

| Feature | Value Proposition | Complexity | Dependencies | Notes |
|---------|-------------------|------------|--------------|-------|
| Shared team inboxes | info@company.de managed by multiple people | High | Auth (teams) | Assignment, internal notes, collision avoidance |
| Email templates | Standardized responses, proposals | Medium | None | Variables: {contact_name}, {deal_value}, {company_name} |
| Email tracking (open/read) | Know if client read your proposal | Medium | Tracking pixel, privacy implications | DSGVO WARNING: must be opt-in and disclosed. Not default. |
| Auto-link emails to deals | "Email about Deal X" auto-associates | Medium | CRM deals | Keyword/subject matching, manual override |
| Scheduled sending | "Send Monday at 9:00" | Medium | Scheduler/cron | Common feature in Gmail/Outlook, expected by power users |
| Email sequences/drip campaigns | Automated follow-up emails | High | Automation engine | "If no reply in 3 days, send follow-up." Sales use case. |
| BCC-to-CRM | Forward BCC to log emails automatically | Medium | Inbound email processing | Classic CRM pattern: BCC a special address to log |

### Anti-Features (Do NOT Build in v1)

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Full email client replacement | Building Thunderbird/Outlook is a multi-year project | Focus on CRM-relevant emails. Users keep their email client for personal/non-CRM email. |
| Spam filtering | Extremely complex ML problem. Gmail/Outlook already do this. | Rely on upstream email provider's spam filter. Only show inbox. |
| Custom email hosting (MTA) | Running a mail server (Postfix/Dovecot) is ops nightmare | SMTP/IMAP integration with existing providers. Self-hosted customers use their own mail server. |
| Calendar invites via email | RFC 5545 calendar attachment handling is complex | Use Hub calendar directly. Email calendar invites are a Phase 2+ feature. |
| Email marketing (bulk sends) | Deliverability, bounce handling, CAN-SPAM compliance | Integrate with Mailchimp/Brevo as connector. Don't build mass email. |

### DACH-Specific Requirements

| Requirement | Rationale | Complexity |
|-------------|-----------|------------|
| DSGVO-compliant email storage | Emails contain personal data. Retention policies, right-to-deletion. | Medium |
| Impressum in signatures | German law requires business contact info in commercial emails | Low |
| Email archiving (GoBD) | German tax law requires business email archiving for 6-10 years | High |
| De-Mail awareness | German secure email standard. Rarely used but some Behoerden require it. | Low (document, don't implement) |
| Custom SMTP support | DACH SMBs often use local hosting providers (Strato, 1&1, Hetzner mail) not Gmail/Outlook | Medium |

---

## Module 4: Video & Voice Calls

### Competitive Landscape

Key players: Zoom (dominant), Microsoft Teams (enterprise), Google Meet (workspace), Jitsi (open-source), LiveKit (open-source SFU). KMU Hub already chose LiveKit (ADR-005) for self-hostability and Go compatibility.

### Table Stakes

| Feature | Why Expected | Complexity | Dependencies | Notes |
|---------|-------------|------------|--------------|-------|
| 1:1 video calls | Basic video communication | Medium | LiveKit server | Click on user -> call. WebRTC via LiveKit SDK. |
| Group video calls (up to 25) | Team meetings | Medium | LiveKit server | SFU handles routing. 25 is enough for SMBs. |
| Audio-only calls | When video not needed/available | Low | LiveKit server | Toggle camera off. Audio-only rooms. |
| Screen sharing | Presentations, demos, support | Medium | LiveKit + Electron APIs | Desktop app can share screen or specific window |
| Mute/unmute, camera on/off | Basic call controls | Low | LiveKit client SDK | Standard call controls UI |
| Call from chat | "Start call" button in channel/DM | Low | Chat module, LiveKit | Creates LiveKit room, posts link in chat |
| Join link | Share link for others to join | Low | None | URL that opens Hub and connects to call room |
| Call quality indicators | Know if connection is poor | Low | LiveKit stats API | Network quality, resolution, frame rate display |
| In-call chat | Text chat during video call | Low | Chat module | Dedicated chat for call participants |
| Hand raising | Signal desire to speak in group calls | Low | LiveKit data channels | Simple ephemeral state via data channel |

### Differentiators

| Feature | Value Proposition | Complexity | Dependencies | Notes |
|---------|-------------------|------------|--------------|-------|
| Call recording + storage | Compliance, training, reference | High | LiveKit Egress, MinIO | Records to MinIO. DSGVO: requires consent of all participants. |
| CRM activity auto-log | "Call with Contact X" logged automatically | Medium | CRM activities | Links call metadata (duration, participants) to CRM contact |
| Scheduled calls with calendar | "Meeting at 14:00" auto-creates call room | Low | Calendar module | Calendar event with LiveKit room link |
| Breakout rooms | Split large meeting into small groups | High | LiveKit room management | Used in workshops, training sessions |
| Virtual backgrounds | Professional appearance from home | Medium | Client-side ML processing | Electron can run background replacement locally |
| Noise suppression | Improve audio quality | Medium | LiveKit or client-side | LiveKit offers server-side or use RNNoise client-side |
| Dial-in via phone (PSTN) | Join call from phone number | Very High | SIP gateway, telephony provider | Nice-to-have for SMBs with field workers. Very complex. |
| Whiteboard in calls | Collaborative drawing/brainstorming | High | Canvas sync, data channels | Reuse as standalone feature later |

### Anti-Features (Do NOT Build in v1)

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| PSTN/phone integration | SIP gateway is extremely complex. Telephony is a whole product. | Start with VoIP only. PSTN as paid add-on connector later. |
| Webinar mode (1000+ viewers) | Different architecture (broadcast vs conference). Not SMB need. | Group calls up to 25-50 participants is sufficient. |
| Live transcription | Requires speech-to-text service. Complex for German. | Defer to Phase 2+. Consider Whisper API integration later. |
| AI meeting summaries | Requires transcription first, then summarization. Two complex systems. | Manual meeting notes linked to CRM activity. AI features out of v1 scope. |
| Custom layouts (speaker view, gallery) | Complex frontend work for marginal UX benefit | Default gallery view. Speaker highlight based on active speaker detection (LiveKit provides this). |

### DACH-Specific Requirements

| Requirement | Rationale | Complexity |
|-------------|-----------|------------|
| DSGVO recording consent | All participants must consent before recording starts | Medium |
| EU-hosted TURN/STUN servers | Media relay servers must be in EU for data sovereignty | Medium (deployment) |
| Call data retention limits | Call recordings subject to data minimization (Art. 5 DSGVO) | Low (policy) |
| Fernmeldegeheimnis awareness | German telecommunications secrecy law applies to voice calls | Low (compliance doc) |

---

## Module 5: HR Module

### Competitive Landscape

Key players: Personio (DACH market leader for SMBs), BambooHR (US-centric), HiBob (modern mid-market), Kenjo (DACH startup), absence.io (leave management DACH). Personio is the gold standard for what DACH SMBs expect from HR software.

### Table Stakes

| Feature | Why Expected | Complexity | Dependencies | Notes |
|---------|-------------|------------|--------------|-------|
| Employee profiles (master data) | Core HR record | Medium | Auth (users are employees) | Extend user profiles: department, position, start date, contract type |
| Leave/vacation requests + approval | #1 reason SMBs buy HR software | High | Calendar, notification system | Request -> Manager approval -> Calendar block. Track remaining days. |
| Leave balance tracking | "How many vacation days do I have left?" | Medium | None | German law: minimum 20 days (5-day week), 24 days (6-day week). Carry-over rules. |
| Absence calendar | See who is out when | Medium | Calendar module | Team view of leaves, sick days, holidays |
| Sick leave recording | Legal documentation requirement | Medium | None | Doctor's note upload after 3 days (German law: AU after 3. Tag) |
| Company org chart | Who reports to whom | Medium (frontend) | Auth (user hierarchy) | Visual org chart. Driven by manager-report relationship. |
| Document storage per employee | Contracts, certificates, Zeugnisse | Medium | File storage (MinIO) | Access-controlled: only HR + employee + manager |
| Time tracking (basic) | "When did I start/stop work?" | Medium | None | Clock in/out. Daily/weekly summaries. Important for Arbeitszeitgesetz. |
| Public holiday calendar | See company-wide holidays | Low | Calendar module | Pre-loaded per Bundesland/Kanton |
| Employee self-service | Employees update own data, request leave | Low | Auth (RBAC) | Reduces HR admin burden significantly |

### Differentiators

| Feature | Value Proposition | Complexity | Dependencies | Notes |
|---------|-------------------|------------|--------------|-------|
| Digital onboarding checklists | New employee gets task list: "Sign contract, set up accounts, read handbook" | Medium | Project management | Reuse task system. Template per role. |
| Digital personnel file (digitale Personalakte) | Complete employee record in one place | Medium | File storage | German HR standard. Replaces physical folder. |
| Overtime tracking + compensation | Track hours beyond contract. Time-off or pay. | High | Time tracking | Complex: Gleitzeitkonto, Ueberstundenabbau, Auszahlung |
| Absence analytics | Sick day trends, leave utilization | Medium | Reporting | Aggregate stats. DSGVO: only anonymized/aggregated for managers. |
| Employee surveys/feedback | Periodic satisfaction surveys | Medium | None | Anonymous surveys. Simple forms, not full survey tool. |
| Certificate/training tracker | Who has which certifications, when do they expire? | Low | None | Relevant for regulated industries (finance, healthcare) |
| Integration with payroll (DATEV export) | Export hours/absences to payroll provider | High | Finance module | DACH SMBs use Steuerberater (tax advisor) who uses DATEV. |

### Anti-Features (Do NOT Build in v1)

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Payroll processing | Legal minefield: tax calculation, social security, Krankenkasse. Personio doesn't even do this natively. | Export data to DATEV/payroll provider. Build export, not processing. |
| Recruiting/ATS (applicant tracking) | Separate product category. Not core for 5-50 employee companies. | Simple job posting link + application email. Full ATS later or via connector. |
| Performance reviews (OKR, 360-feedback) | Complex HR process. SMBs under 50 rarely formalize this. | Simple notes on employee profile. Full performance management later. |
| Benefits administration | DACH benefits (bAV, Jobticket, etc.) have complex tax implications | Document benefits in employee profile. Don't calculate tax advantages. |
| Shift planning/rostering | Complex scheduling algorithm. Relevant only for shift-work industries. | Basic time tracking only. Shift planning as plugin for specific industries. |
| Learning management (LMS) | Full LMS is a separate product (SAP SuccessFactors, Cornerstone) | Training tracker (who attended what). Not course hosting. |

### DACH-Specific Requirements (CRITICAL for HR)

| Requirement | Rationale | Complexity |
|-------------|-----------|------------|
| Urlaubsanspruch calculation | German law: min 20 days/year (5-day week). Pro-rata for mid-year starts. Carry-over until 31.03 next year. | High |
| Arbeitszeitgesetz compliance | Max 8h/day (extendable to 10h with compensation), 11h rest between shifts, break rules | High |
| Mutterschutz / Elternzeit | Maternity/parental leave tracking with legal deadlines | Medium |
| Krankheitstage tracking | Sick days: first 6 weeks Entgeltfortzahlung (employer pays), then Krankengeld (health insurance) | Medium |
| Betriebsrat data access rules | Works council has specific data access rights. Must be configurable. | Medium |
| Aufbewahrungspflichten | Personnel files must be retained for specific periods (up to 10 years for some documents) | Medium |
| Austrian / Swiss variations | AT: 25 vacation days minimum. CH: 4 weeks minimum, cantonal holidays. Different labor laws. | Medium |
| Minijob / Teilzeit tracking | Different rules for mini-job (520 EUR/month), part-time, full-time employees | Medium |
| Probezeit (probation period) tracking | 6-month probation standard in Germany. Different notice periods. | Low |
| DATEV-compatible time export | Export format that Steuerberater/payroll providers can import | High |

---

## Module 6: Finance Module

### Competitive Landscape

Key players: lexoffice (DACH market leader for small business), sevDesk (DACH cloud accounting), Billomat (invoicing), FastBill (freelancers), DATEV (tax advisor standard), Bexio (Swiss market). For SMBs, lexoffice and sevDesk dominate. But they are standalone tools, not integrated into a workplace platform.

### Table Stakes

| Feature | Why Expected | Complexity | Dependencies | Notes |
|---------|-------------|------------|--------------|-------|
| Quote/Angebot creation | Start of sales process | Medium | CRM (deals, contacts) | Create quote from deal. PDF generation. |
| Invoice/Rechnung creation | Core business document | High | CRM (contacts, companies) | Must comply with GoBD requirements. Sequential numbering. |
| Invoice PDF generation | Professional-looking documents | Medium | PDF library | Company logo, legal info, line items, tax calculation |
| Tax calculation (MwSt/USt) | Legal requirement | Medium | None | 19% standard, 7% reduced (Germany). Reverse charge for EU B2B. |
| Payment status tracking | Is this invoice paid? | Low | None | Draft, Sent, Overdue, Paid, Cancelled |
| Customer/contact billing info | Invoice address, VAT ID, payment terms | Low | CRM contacts/companies | Extend CRM entities with billing fields |
| Expense tracking | Record business expenses | Medium | File storage | Receipt photo upload, categorization, amount |
| Revenue overview/dashboard | "How much did we earn this month?" | Medium | Reporting | Monthly/quarterly/yearly revenue charts |
| Sequential invoice numbering | Legal requirement in DACH | Low | None | Format: RE-2026-0001. No gaps allowed (GoBD). |
| Credit notes (Gutschriften) | Correct or reverse an invoice | Medium | Invoice system | References original invoice. Negative line items. |

### Differentiators

| Feature | Value Proposition | Complexity | Dependencies | Notes |
|---------|-------------------|------------|--------------|-------|
| Deal-to-quote-to-invoice pipeline | CRM deal -> quote -> approved -> invoice, all in one flow | High | CRM deals | Massive differentiator: no tool-switching for the sales-to-billing flow |
| DATEV export | Export to Steuerberater's system | High | None | Buchungsstapel export format. Critical for DACH market adoption. |
| Recurring invoices | Monthly retainer billing | Medium | Scheduler/cron | "Invoice Client X 2000 EUR on 1st of every month" |
| Payment reminders (Mahnwesen) | Automated dunning: 1st, 2nd, 3rd reminder | Medium | Email, automation | Zahlungserinnerung -> 1. Mahnung -> 2. Mahnung -> Inkasso-Hinweis |
| Bank account connection (FinTS/HBCI) | Auto-match payments to invoices | Very High | Banking API (FinTS) | German banking standard. Auto-reconciliation. Complex but high value. |
| Multi-currency support | International clients | Medium | None | EUR default. CHF for Swiss, USD/GBP for international. Exchange rates. |
| Time tracking to invoice | "Bill 40 hours at 150 EUR/hr" | Medium | Time tracking (HR module) | Convert tracked time to invoice line items |
| Prepayment/deposit invoices (Abschlagsrechnungen) | Partial billing for large projects | Medium | Invoice system | Common in construction, consulting, project work |

### Anti-Features (Do NOT Build in v1)

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Full double-entry bookkeeping (Buchhaltung) | Extremely complex. Legal requirements (HGB). This is what DATEV is for. | Export to DATEV. Focus on Belegerfassung (document capture), not Buchfuehrung (bookkeeping). |
| Payroll (Lohnabrechnung) | Legal minefield: Lohnsteuer, Sozialversicherung, Kirchensteuer. Even Personio partners with DATEV for this. | Export hours/data to payroll provider. Never process payroll. |
| Tax filing (Steuererklarung) | Legal liability. Tax advisors do this. | Provide data to Steuerberater. Not our business. |
| Full ERP inventory management | Different product category (Warenwirtschaft). Not needed by service companies. | Basic product/service catalog for invoice line items. Full inventory is plugin territory. |
| POS/cash register (Kassensystem) | TSE (Technische Sicherheitseinrichtung) requirement since 2020. Hardware integration. | Out of scope. Retail SMBs need specialized POS. |
| Banking (payment processing) | Heavily regulated (BaFin/PSD2). Requires banking license for some features. | Payment tracking only. Actual banking stays in bank app. |

### DACH-Specific Requirements (CRITICAL for Finance)

| Requirement | Rationale | Complexity |
|-------------|-----------|------------|
| GoBD compliance | Grundsaetze ordnungsmaessiger Buchfuehrung und DV-gestuetzter Buchfuehrungssysteme. Invoices immutable once sent. | High |
| Pflichtangaben auf Rechnungen | Required invoice fields: company name, address, Steuernummer/USt-IdNr, invoice number, date, line items, tax amounts | Medium |
| Reverse Charge (EU B2B) | Innergemeinschaftliche Lieferung: no VAT charged, buyer pays in their country | Medium |
| Kleinunternehmerregelung | Small business exemption (under 22k EUR/year): no VAT charged, special note required on invoice | Low |
| Aufbewahrungspflicht (10 years) | All invoices, quotes, business correspondence must be retained 10 years | Medium |
| DATEV export format | Buchungsstapel CSV format that tax advisors import into DATEV | High |
| Austrian variations | Different VAT rates (20% standard, 10%/13% reduced). UID-Nummer instead of USt-IdNr. | Medium |
| Swiss variations | CHF as currency. MwSt rates (8.1% standard, 2.6% reduced, 3.8% hospitality). No EU VAT rules. | Medium |
| XRechnung / ZUGFeRD | Electronic invoice formats. Mandatory for B2G (government) in Germany since 2020. Increasingly B2B. | High |
| Skonto (early payment discount) | "2% discount if paid within 10 days." Common DACH business practice. | Low |

---

## Module 7: Automation Engine

### Competitive Landscape

Key players: Zapier (external, 5000+ integrations), Make/Integromat (external, visual), n8n (open-source, self-hostable), Power Automate (Microsoft), Pipedrive Automations (CRM-built-in). For an all-in-one platform, the model should be more like Monday.com's built-in automations or Pipedrive's: pre-built triggers and actions within the platform, not a generic integration platform.

### Table Stakes

| Feature | Why Expected | Complexity | Dependencies | Notes |
|---------|-------------|------------|--------------|-------|
| Trigger-action model | "When X happens, do Y" | High | Event system across all modules | Core automation primitive. All modules must emit events. |
| Pre-built triggers | Deal stage changed, task completed, invoice overdue, etc. | Medium | All modules | Start with 10-15 common triggers across CRM, PM, Finance |
| Pre-built actions | Send notification, create task, send email, update field, etc. | Medium | All modules | Start with 8-10 common actions |
| Conditional logic (if/else) | "Only if deal value > 10000 EUR" | Medium | None | Simple field-value conditions. AND/OR operators. |
| Automation logs | "What did this automation do?" | Medium | None | Execution history with timestamps, input/output, success/failure |
| Enable/disable automations | Turn off without deleting | Low | None | Active/inactive toggle |
| Per-module automation examples | "Here's what you can automate for CRM" | Low | None | Pre-built templates users can activate |

### Differentiators

| Feature | Value Proposition | Complexity | Dependencies | Notes |
|---------|-------------------|------------|--------------|-------|
| Cross-module workflows | "When deal is won (CRM), create project (PM), schedule kickoff (Calendar), send invoice (Finance)" | High | All modules | THE killer feature of an all-in-one platform. This is why you build integrated. |
| Visual workflow builder | Drag-and-drop automation creation | High (frontend) | Desktop app | Like Monday.com or n8n. Non-technical users can build workflows. |
| Scheduled automations | "Every Monday at 9:00, generate weekly report" | Medium | Scheduler/cron | Cron-based triggers, not just event-based |
| Multi-step workflows | Chain multiple actions sequentially | High | None | Step 1 output feeds Step 2 input. Error handling per step. |
| Delay/wait steps | "Wait 3 days, then send reminder" | Medium | Scheduler | Requires persistent workflow state |
| Webhook triggers (inbound) | External systems trigger Hub automations | Medium | None | Enables external integration without full connector |
| Webhook actions (outbound) | Hub automations trigger external systems | Medium | None | POST to external URL with payload |
| Template marketplace | Pre-built automation recipes | Low | None | "Sales follow-up", "New employee onboarding", "Invoice overdue reminder" |

### Anti-Features (Do NOT Build in v1)

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| General-purpose integration platform (Zapier clone) | 5000+ connectors is a full product. Not our core. | Build internal automations. External integration via webhooks + case-by-case connectors. |
| Code-based automations (JavaScript/Python scripting) | Security nightmare. Execution sandbox required. | Config-based automation. WASM plugins for complex logic. |
| AI-powered automation suggestions | "We noticed you always do X after Y" - requires ML pipeline | Manual automation creation. AI suggestions in future version. |
| Complex branching (parallel paths, loops) | Enterprise workflow engine complexity. BPMN territory. | Sequential steps with if/else. No parallel branches or loops in v1. |
| Error recovery/retry logic | Complex distributed systems problem | Log errors, notify admin. Manual re-trigger. Automatic retry in v2. |

### DACH-Specific Requirements

| Requirement | Rationale | Complexity |
|-------------|-----------|------------|
| Audit trail for automated actions | GoBD: automated financial document creation must be traceable | Medium |
| DSGVO data processing in automations | Automations handling personal data must respect consent and purpose limitation | Medium |
| German-language automation templates | Templates and trigger/action descriptions in German | Low |

---

## Module 8: Plugin/Extension System

### Competitive Landscape

Key players: Slack (App Directory, Bolt SDK), Notion (API + integrations), Jira (Atlassian Marketplace, Connect/Forge), Shopify (App framework), WordPress (plugin ecosystem). KMU Hub already decided on Config + WASM two-tier system (ADR-004).

### Table Stakes

| Feature | Why Expected | Complexity | Dependencies | Notes |
|---------|-------------|------------|--------------|-------|
| Config-based customization | Custom fields, workflows, validation rules without code | Medium | Existing custom fields engine | 80% of customization needs. JSON/YAML configuration. |
| Plugin installation/removal | Add/remove functionality safely | Medium | Plugin runtime | Install from file/URL. Clean uninstall (remove data, hooks). |
| Plugin permissions model | What can a plugin access? | High | Auth (RBAC) | Plugins declare required permissions. Admin approves. Principle of least privilege. |
| Plugin enable/disable | Turn off without uninstalling | Low | None | Active/inactive state |
| Plugin configuration UI | Settings for installed plugins | Medium (frontend) | Desktop app | Plugin declares config schema, Hub renders form |
| Extension points (hooks) | Defined places where plugins can inject behavior | High | All modules | Before/after CRUD operations, custom validation, data transformation |
| Plugin API (read/write Hub data) | Plugins need to access CRM contacts, tasks, etc. | High | All modules, Auth | Versioned API exposed to WASM sandbox. Rate-limited. |
| Plugin error isolation | Bad plugin must not crash Hub | High | WASM runtime (wazero) | WASM sandbox. Timeout execution. Memory limits. |

### Differentiators

| Feature | Value Proposition | Complexity | Dependencies | Notes |
|---------|-------------------|------------|--------------|-------|
| Industry templates (Branchenvorlagen) | "Install the Handwerker-Package" adds construction-specific fields, workflows, document templates | Medium | Config system | THE KMU Hub USP. Onsite analysis -> custom template -> install. |
| Plugin marketplace (later) | Community-contributed extensions | Very High | Distribution, review, billing | Phase 2+ at earliest. Start with first-party and customer-specific plugins. |
| UI extension points | Plugin adds custom sidebar widget, dashboard card, or detail tab | High (frontend) | Desktop app, WASM | Render plugin UI within Hub. iframe-based or WASM-rendered. |
| Custom entity types | Plugin defines entirely new data entities (e.g., "Vehicles" for a fleet management plugin) | Very High | Database, CRUD engine | Powerful but complex. Consider for v2+. |
| Webhook connectors as plugins | External service integration packaged as plugin | Medium | Automation engine | "DATEV Connector" plugin that handles export format |
| Plugin SDK + documentation | Developers can build plugins | High | None | TypeScript/Rust SDK for WASM. Plugin template project. Getting started guide. |
| Plugin versioning | Update plugins without breaking existing installations | Medium | Plugin runtime | SemVer. Breaking changes require admin approval. |

### Anti-Features (Do NOT Build in v1)

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Open marketplace (anyone can publish) | Quality control, security review, liability. App stores are full-time businesses. | Curated plugins: first-party + approved partner plugins only. |
| Server-side plugin execution (native) | Security risk. Plugin can access filesystem, network, secrets. | WASM-only for server-side. Sandboxed, no host access. |
| Plugin billing/monetization | In-app purchases, revenue sharing, subscription management for plugins | First-party plugins included in plan. Custom plugins billed as consulting. |
| Plugin-to-plugin communication | Exponential complexity. Plugin dependency hell. | Plugins communicate through Hub APIs only. No direct plugin-to-plugin calls. |
| Real-time UI plugins | Plugin that adds live-updating dashboard widgets | Static UI extension points first. Real-time plugin rendering is Phase 2+. |

### DACH-Specific Requirements

| Requirement | Rationale | Complexity |
|-------------|-----------|------------|
| German plugin descriptions and UI | Plugins must present in German for non-English-speaking SMB users | Low |
| DSGVO compliance per plugin | Each plugin that processes personal data needs documented purpose and legal basis | Medium |
| Branchenspezifische templates | Pre-built configurations for common DACH industries: Handwerk, Beratung, Finanzdienstleistung, Gesundheitswesen | Medium |

---

## Cross-Module Feature Dependencies

Understanding dependencies is critical for phase ordering.

```
Legend: A --> B means "B depends on A" (A must be built first)

Auth (DONE) --> Everything

CRM (DONE) --> Email Integration (email-to-contact linking)
CRM (DONE) --> Finance (quote-from-deal, invoice-to-contact)
CRM (DONE) --> Project Management (task-to-contact, task-from-deal)

Chat (DONE) --> Video & Voice (call-from-chat button)
Chat (DONE) --> Notifications (notification delivery channel)

Notification System --> Calendar (event reminders)
Notification System --> Project Management (due date reminders)
Notification System --> HR (leave request approvals)
Notification System --> Finance (payment reminders)
Notification System --> Automation (notification as action)

Calendar --> Project Management (task deadlines on calendar)
Calendar --> HR (absence calendar overlay)
Calendar --> Video & Voice (scheduled calls)

Project Management --> Automation (task triggers/actions)
HR --> Finance (time-tracking-to-invoice, DATEV export)
Finance --> Automation (invoice triggers/actions)

All Modules --> Plugin System (extension points in each module)
All Modules --> Automation Engine (events from each module)
```

### Critical Path Analysis

1. **Notification System** is the most critical dependency -- Calendar, PM, HR, Finance, and Automation all need it. Build first or concurrently with the first feature module.

2. **Calendar** is a hub module -- PM, HR, and Video all integrate with it. Build early.

3. **Automation Engine** requires events from all modules -- build the event infrastructure early, but the automation UI/builder can come later.

4. **Plugin System** must be designed early (extension points planned into each module) but can be implemented later.

---

## MVP Recommendation per Module

### Priority 1: Build First (Core Daily Workflow)

| Module | MVP Scope | Rationale |
|--------|-----------|-----------|
| **Notifications** | In-app notifications, email notifications, per-user preferences | Unblocks all other modules. Chat notifications (Sprint 5) is the starting point. |
| **Project Management** | Tasks, projects, list/board views, assignees, due dates, comments | Second most-used daily feature after chat. Links to CRM deals. |
| **Calendar** | Personal calendars, shared calendars, events, recurring events, availability | Third daily-use feature. Required for scheduling, PM deadlines, HR absences. |

### Priority 2: Build Second (Business Operations)

| Module | MVP Scope | Rationale |
|--------|-----------|-----------|
| **Email Integration** | Send/receive email in Hub, CRM contact linking, threading | "Don't leave the app" promise. But complex (IMAP/SMTP). |
| **Video & Voice** | 1:1 calls, group calls, screen sharing, call-from-chat | LiveKit is already chosen. Replaces Zoom dependency. |
| **Finance** | Quotes, invoices, PDF generation, payment tracking, MwSt calculation | Revenue-generating. Pilot customer (financial education center) likely needs this. |

### Priority 3: Build Third (Advanced Features)

| Module | MVP Scope | Rationale |
|--------|-----------|-----------|
| **HR** | Leave requests, absence calendar, basic time tracking, employee profiles | Important for daily-driver promise but not week-1 critical. |
| **Automation Engine** | Trigger-action model, 10 triggers, 8 actions, automation logs | Requires all other modules to emit events. Build after modules exist. |

### Priority 4: Build Last (Extensibility)

| Module | MVP Scope | Rationale |
|--------|-----------|-----------|
| **Plugin System** | Config-based customization, WASM runtime, 3-5 extension points, plugin API | USP for customer-specific tailoring. But needs stable core modules first. |

### Defer to Post-MVP

- CalDAV server (use iCal import/export instead)
- External booking pages (Calendly replacement)
- Full email client (focus on CRM-relevant emails)
- PSTN/phone calls (VoIP only)
- Full bookkeeping (DATEV export instead)
- Payroll processing (export to payroll provider)
- Visual workflow builder (simple rule-based first)
- Plugin marketplace (curated plugins only)
- Bank account integration (FinTS/HBCI)
- XRechnung/ZUGFeRD electronic invoicing

---

## Complexity Budget

Estimated backend complexity per module (for a solo developer with AI assistance):

| Module | Estimated Effort | New DB Tables | New gRPC Service | Complexity Driver |
|--------|-----------------|---------------|-------------------|-------------------|
| Notifications | 1-2 weeks | 2-3 | No (extend gateway) | Multi-channel delivery, preference system |
| Project Management | 3-4 weeks | 4-6 | Yes (PM service) | Task dependencies, multiple views, custom fields |
| Calendar | 3-4 weeks | 3-4 | Yes (Calendar service) | Recurring events (RRULE), timezone handling, availability |
| Email Integration | 4-6 weeks | 3-5 | Yes (Email service) | IMAP polling, SMTP sending, threading, attachment handling |
| Video & Voice | 2-3 weeks | 2-3 | No (extend gateway) | LiveKit integration, WebRTC. Most logic in client SDK. |
| Finance | 4-6 weeks | 5-8 | Yes (Finance service) | GoBD compliance, PDF generation, tax calculation, DATEV export |
| HR | 3-5 weeks | 5-7 | Yes (HR service) | Leave calculation, Arbeitszeitgesetz, DACH labor law variations |
| Automation Engine | 3-5 weeks | 3-5 | Yes (Automation service) | Event bus, workflow execution, state management |
| Plugin System | 4-6 weeks | 2-3 | No (embedded in gateway) | WASM runtime, plugin API, security sandbox |

**Total estimated: 27-41 weeks** (7-10 months) for all modules at MVP scope.

---

## Sources and Confidence Assessment

| Area | Confidence | Rationale |
|------|------------|-----------|
| Project Management features | HIGH | PM tools are extremely well-documented. Jira, Asana, Monday, Linear feature sets are stable and well-known. |
| Calendar features | HIGH | Calendar standards (iCal, CalDAV) are RFC-defined. Feature sets are stable. |
| Email Integration | HIGH | Email protocols (IMAP, SMTP) are decades-old standards. CRM email integration patterns well-established. |
| Video & Voice | HIGH | LiveKit is already chosen (ADR-005). WebRTC feature sets well-known. |
| HR features (general) | MEDIUM | General HR feature knowledge is solid. DACH-specific labor law details should be verified with a legal advisor. |
| HR features (DACH legal) | MEDIUM | Based on German labor law knowledge (BUrlG, ArbZG, MuSchG). Austrian and Swiss specifics need verification. |
| Finance features (general) | HIGH | Invoice/quote features well-established across all CRM/finance tools. |
| Finance features (DACH legal) | MEDIUM | GoBD, UStG requirements are known but complex. XRechnung/ZUGFeRD specifics should be verified with current standards. DATEV export format specifics need official documentation. |
| Automation Engine | HIGH | Trigger-action pattern is universal (Zapier, Make, n8n all use it). Architecture patterns well-documented. |
| Plugin System | MEDIUM | WASM plugin systems are newer. wazero capabilities based on training data. Plugin API design patterns from Slack/Shopify well-known. |

**Overall confidence: MEDIUM** -- Feature listings are solid, but DACH-specific legal requirements and current library versions should be verified before implementation. WebSearch/WebFetch unavailability prevented live verification.

---

*Feature landscape research: 2026-02-07*
*Note: Research conducted without web access. All findings based on training data (cutoff ~early 2025). DACH legal requirements should be verified with current official sources before implementation.*
