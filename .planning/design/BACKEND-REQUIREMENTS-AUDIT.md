# Backend Requirements Audit — KMU Hub Desktop App

> Exhaustive audit of every frontend feature, the backend APIs it needs, what already exists, and what is missing.
> Updated: 2026-02-15 (rev 3 — includes D7-D9 + hub expansion modules)

---

## Quick Summary

| Status | Count |
|--------|-------|
| Backend BUILT + Frontend wired | Auth, CRM, Chat, Notifications, Dashboard, Work |
| Backend BUILT but Frontend NOT wired yet | CRM hooks, Work hooks, Chat hooks, Notification hooks |
| Frontend BUILT, NO backend (mock data) | ~250 endpoints across 20+ modules |

---

## 1. FULLY BUILT (Backend + Frontend Connected)

### 1.1 Auth
- Login, Logout, Refresh, Register, Me — all wired
- WebSocket at `/api/v1/ws` — wired
- 5 roles: admin, manager, member, hr, it_support

### 1.2 Dashboard Layout
- GET/PUT/DELETE `/api/v1/dashboard/layout` — wired
- Role default layouts — wired
- **But:** Widget data inside dashboard is still MOCK (see Section 3)

### 1.3 CRM (Contacts, Companies, Deals, Pipeline, Activities, Search)
- ~30 endpoints BUILT in backend
- TanStack Query hooks exist in `api/hooks/`
- **But:** UI still uses Zustand stores, NOT the hooks yet. Luke's backend works, Darien hasn't wired it.

### 1.4 Chat (Channels, Messages, Threads)
- ~17 endpoints BUILT + WebSocket events
- TanStack Query hooks exist
- **But:** UI not wired yet

### 1.5 Notifications
- ~7 endpoints BUILT + WebSocket events
- TanStack Query hooks exist
- **But:** UI not wired yet

### 1.6 Work (Projects, Tasks, Kanban, Comments, Files, Dependencies)
- ~35 endpoints BUILT (missing: Gantt + Timer)
- TanStack Query hooks exist
- **But:** UI not wired yet

---

## 2. FRONTEND BUILT, NO BACKEND — Module Details

### 2.1 E-Mail Module (NEW features since D7+)

**Store:** `stores/mails.ts` — 13 mock emails, 6 folders, composeDraft state
**Files:** `MailsPage.tsx`, `ComposeModal.tsx`, `ComposeInline.tsx`, `ComposeWindowPage.tsx`, `compose-shared.tsx`

**Frontend Features:**
- 3-column layout: folders / email list / detail
- Compose: inline panel in detail area + pop-out to separate OS window (Electron BrowserWindow)
- Reply, Reply-All, Forward
- Send, Save Draft
- Mark read/unread, Star, Archive, Move to folder
- Delete → Trash → Permanent delete, Empty trash
- Custom folders (CRUD)
- Drucken (print email)
- Exportieren (PDF export)
- In Dateien speichern (save email to Documents store)
- Contact suggestions in compose (autocomplete)

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/mails` | List emails (folder filter, search, pagination) |
| GET | `/api/v1/mails/{id}` | Get email detail |
| POST | `/api/v1/mails/send` | Send email (SMTP) |
| POST | `/api/v1/mails/drafts` | Save draft |
| PATCH | `/api/v1/mails/{id}` | Update flags (read, starred) |
| POST | `/api/v1/mails/{id}/archive` | Archive email |
| POST | `/api/v1/mails/{id}/move` | Move to folder |
| DELETE | `/api/v1/mails/{id}` | Delete (trash or permanent) |
| POST | `/api/v1/mails/trash/empty` | Empty trash |
| GET | `/api/v1/mails/folders` | List folders |
| POST | `/api/v1/mails/folders` | Create custom folder |
| PATCH | `/api/v1/mails/folders/{id}` | Rename folder |
| DELETE | `/api/v1/mails/folders/{id}` | Delete folder |
| GET | `/api/v1/mails/{id}/pdf` | Export email as PDF |
| POST | `/api/v1/mails/{id}/save-to-files` | Save email to Documents |
| POST | `/api/v1/mails/sync` | Trigger IMAP sync |

**Complexity:** HIGH — requires IMAP sync service + SMTP relay
**Note:** Compose pop-out window is Electron-only (IPC), no backend needed for that.

---

### 2.2 Buchhaltung / Finance

**Store:** `stores/finance.ts` — 5 invoices, 10 transactions, 5 expenses
**Files:** `BuchhaltungPage.tsx`, `InvoiceFormDialog.tsx`, `InvoiceDetailPanel.tsx`, `ExpenseFormDialog.tsx`, `PaymentRecordDialog.tsx`, `ExportDialog.tsx`

**Frontend Features:**
- Invoices + Quotes CRUD (with line items: description, qty, unit price, discount, VAT)
- Payment recording (partial payments, multiple methods)
- Invoice send, cancel, duplicate
- Expenses CRUD with approve/reject workflow
- Transactions list
- Financial reports (bar chart, category breakdown)
- DATEV export dialog
- Stats dashboard (Einnahmen, Ausgaben, Saldo, Offene Rechnungen)

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/finance/invoices` | List invoices/quotes (filter, search) |
| POST | `/api/v1/finance/invoices` | Create invoice/quote |
| GET | `/api/v1/finance/invoices/{id}` | Get invoice detail |
| PATCH | `/api/v1/finance/invoices/{id}` | Update invoice |
| DELETE | `/api/v1/finance/invoices/{id}` | Delete invoice (draft only) |
| POST | `/api/v1/finance/invoices/{id}/send` | Send invoice via email |
| POST | `/api/v1/finance/invoices/{id}/cancel` | Cancel invoice |
| POST | `/api/v1/finance/invoices/{id}/duplicate` | Duplicate invoice |
| POST | `/api/v1/finance/invoices/{id}/payments` | Record payment |
| GET | `/api/v1/finance/invoices/{id}/pdf` | Generate PDF |
| GET | `/api/v1/finance/transactions` | List transactions |
| POST | `/api/v1/finance/transactions` | Create transaction |
| DELETE | `/api/v1/finance/transactions/{id}` | Delete transaction |
| GET | `/api/v1/finance/expenses` | List expenses |
| POST | `/api/v1/finance/expenses` | Create expense |
| POST | `/api/v1/finance/expenses/{id}/approve` | Approve expense |
| POST | `/api/v1/finance/expenses/{id}/reject` | Reject expense |
| DELETE | `/api/v1/finance/expenses/{id}` | Delete expense |
| GET | `/api/v1/finance/export/datev` | DATEV CSV export |
| GET | `/api/v1/finance/reports` | Financial reports (date range) |

**Complexity:** HIGH — GoBD compliance, PDF generation, DATEV format

---

### 2.3 Kontakte (Extended Contacts)

**Store:** `stores/contacts.ts` — 14 contacts, 3 groups
**Note:** Separate from CRM contacts API. Uses richer data model.

**Frontend Features:**
- CRUD contacts (with address, social media, tags, notes, activities)
- Favorite toggle
- Duplicate contact
- Bulk import (CSV)
- Contact groups (CRUD + member management)
- Quick actions: Email, Call (LiveKit), Message (Chat)
- Category filter (employee/customer/partner)
- Activity history per contact

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/contacts` | List (paginated, filters, search) |
| POST | `/api/v1/contacts` | Create contact |
| GET | `/api/v1/contacts/{id}` | Get detail |
| PATCH | `/api/v1/contacts/{id}` | Update contact |
| DELETE | `/api/v1/contacts/{id}` | Delete contact |
| POST | `/api/v1/contacts/{id}/favorite` | Toggle favorite |
| POST | `/api/v1/contacts/{id}/duplicate` | Duplicate |
| POST | `/api/v1/contacts/import` | Bulk import (CSV) |
| GET | `/api/v1/contacts/groups` | List groups |
| POST | `/api/v1/contacts/groups` | Create group |
| PATCH | `/api/v1/contacts/groups/{id}` | Update group |
| DELETE | `/api/v1/contacts/groups/{id}` | Delete group |
| POST | `/api/v1/contacts/groups/{id}/members` | Add to group |
| DELETE | `/api/v1/contacts/groups/{id}/members/{cid}` | Remove from group |

**Note:** CRM contacts API exists but needs extended fields: salutation, mobile, department, address, website, category, status, tags, socialMedia, lastContact, projects, isFavorite, activities. Either extend CRM model or create separate service.

---

### 2.4 Dokumente

**Store:** `stores/documents.ts` — 12 files, 9 folders (including system: root, shared, favorites, vault)

**Frontend Features:**
- Upload files (drag & drop)
- Download, rename, move, delete files
- Favorite toggle
- Share with users (view/edit permissions)
- Version history
- CRUD folders
- File preview (PDF, images)
- Search & filter (tags, type, folder)
- Vault (secure storage)
- Storage usage display
- **NEW:** Emails can be saved as documents from Mails module

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/documents` | List files (filter, search) |
| POST | `/api/v1/documents/upload` | Upload file (multipart) |
| GET | `/api/v1/documents/{id}` | Get file metadata |
| GET | `/api/v1/documents/{id}/download` | Download file |
| PATCH | `/api/v1/documents/{id}` | Rename/move/tag file |
| DELETE | `/api/v1/documents/{id}` | Delete file |
| POST | `/api/v1/documents/{id}/favorite` | Toggle favorite |
| POST | `/api/v1/documents/{id}/share` | Share with users |
| GET | `/api/v1/documents/{id}/versions` | Get version history |
| GET | `/api/v1/folders` | List folders |
| POST | `/api/v1/folders` | Create folder |
| PATCH | `/api/v1/folders/{id}` | Rename/move folder |
| DELETE | `/api/v1/folders/{id}` | Delete folder |
| GET | `/api/v1/storage/usage` | Storage usage stats |

**Complexity:** HIGH — file upload/download, versioning, vault encryption

---

### 2.5 Team / HR

**Store:** `stores/team.ts` — 12 members, 7 HR requests, 6 departments

**Frontend Features:**
- Team member list with status (active/away/offline)
- CRUD members (with contract type, workload, skills, projects)
- Invite member (email)
- Deactivate member
- HR requests: vacation, sick leave, overtime, homeoffice
- Approve/reject requests
- Absence calendar view
- Department management

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/team/members` | List members |
| POST | `/api/v1/team/members` | Add/invite member |
| PATCH | `/api/v1/team/members/{id}` | Update member |
| POST | `/api/v1/team/members/{id}/deactivate` | Deactivate |
| DELETE | `/api/v1/team/members/{id}` | Delete member |
| GET | `/api/v1/team/requests` | List HR requests |
| POST | `/api/v1/team/requests` | Create request |
| POST | `/api/v1/team/requests/{id}/approve` | Approve |
| POST | `/api/v1/team/requests/{id}/reject` | Reject |
| DELETE | `/api/v1/team/requests/{id}` | Delete request |
| GET | `/api/v1/team/departments` | List departments |

---

### 2.6 Zeiterfassung (Time Tracking)

**Store:** `stores/timetracking.ts` — 15 entries, 6 categories, 4 templates, 6 team activities, 4 absences

**Frontend Features:**
- Active timer (start/pause/resume/stop) — persistent in header widget
- Manual time entries (CRUD)
- Categories (CRUD)
- Templates for quick entry
- Team activity overview (who's tracking what)
- Absence requests (CRUD + approve/reject)
- 6 sub-views: Overview, Timer, Entries, Team, Reports, Absences
- Work target tracking (Soll vs. Ist hours)
- CSV/PDF report export

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | `/api/v1/timetracking/timer/start` | Start timer |
| POST | `/api/v1/timetracking/timer/pause` | Pause |
| POST | `/api/v1/timetracking/timer/resume` | Resume |
| POST | `/api/v1/timetracking/timer/stop` | Stop (creates entry) |
| GET | `/api/v1/timetracking/entries` | List entries (date filter) |
| POST | `/api/v1/timetracking/entries` | Create manual entry |
| PATCH | `/api/v1/timetracking/entries/{id}` | Update entry |
| DELETE | `/api/v1/timetracking/entries/{id}` | Delete entry |
| GET | `/api/v1/timetracking/categories` | List categories |
| POST | `/api/v1/timetracking/categories` | Create category |
| PATCH | `/api/v1/timetracking/categories/{id}` | Update |
| DELETE | `/api/v1/timetracking/categories/{id}` | Delete |
| GET | `/api/v1/timetracking/templates` | List templates |
| POST | `/api/v1/timetracking/templates` | Create template |
| DELETE | `/api/v1/timetracking/templates/{id}` | Delete |
| GET | `/api/v1/timetracking/team-activity` | Team activity status |
| GET | `/api/v1/timetracking/absences` | List absences |
| POST | `/api/v1/timetracking/absences` | Request absence |
| POST | `/api/v1/timetracking/absences/{id}/approve` | Approve |
| POST | `/api/v1/timetracking/absences/{id}/reject` | Reject |
| DELETE | `/api/v1/timetracking/absences/{id}` | Delete |
| GET | `/api/v1/timetracking/reports` | Generate report (CSV/PDF) |

**Complexity:** HIGH — timer state sync, team activity real-time, report generation

---

### 2.7 Meetings

**Store:** `stores/meetings.ts` — 8 meetings with agenda, files, participants

**Frontend Features:**
- CRUD meetings (with agenda items, participants, room, recurrence)
- Cancel/duplicate meeting
- Agenda items (add, toggle, remove)
- Notes editing
- Join meeting (LiveKit video room)
- Start 1:1 call (LiveKit)
- Call overlay (in-call UI)
- File attachments per meeting

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/meetings` | List meetings (date filter) |
| POST | `/api/v1/meetings` | Create meeting |
| PATCH | `/api/v1/meetings/{id}` | Update meeting |
| DELETE | `/api/v1/meetings/{id}` | Delete |
| POST | `/api/v1/meetings/{id}/cancel` | Cancel |
| POST | `/api/v1/meetings/{id}/duplicate` | Duplicate |
| POST | `/api/v1/meetings/{id}/agenda` | Add agenda item |
| PATCH | `/api/v1/meetings/{id}/agenda/{item}` | Toggle agenda |
| DELETE | `/api/v1/meetings/{id}/agenda/{item}` | Remove agenda |
| PATCH | `/api/v1/meetings/{id}/notes` | Update notes |
| POST | `/api/v1/meetings/{id}/join` | Get LiveKit token |
| POST | `/api/v1/calls/start` | Start 1:1 call |
| POST | `/api/v1/calls/end` | End call |

**Complexity:** HIGH — LiveKit integration, room management

---

### 2.8 Kalender

**Files:** `modules/kalender/` — KalenderPage, RoomBookingView, CategoryManager, CalendarBrowse

**Frontend Features:**
- Month/Week/Day/Agenda views
- CRUD calendar events (with recurrence, reminders, categories)
- Room booking
- Category management (colors, labels)
- Import/export (.ics)
- Public holidays (region-based)
- Work hours configuration

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/calendar/events` | List events (date range) |
| POST | `/api/v1/calendar/events` | Create event |
| PATCH | `/api/v1/calendar/events/{id}` | Update event |
| DELETE | `/api/v1/calendar/events/{id}` | Delete event |
| GET | `/api/v1/calendar/categories` | List categories |
| POST | `/api/v1/calendar/categories` | Create category |
| PATCH | `/api/v1/calendar/categories/{id}` | Update |
| DELETE | `/api/v1/calendar/categories/{id}` | Delete |
| GET | `/api/v1/calendar/rooms` | List rooms |
| POST | `/api/v1/calendar/rooms/{id}/book` | Book room |
| GET | `/api/v1/calendar/rooms/{id}/availability` | Check availability |
| GET | `/api/v1/calendar/holidays` | Public holidays (by region) |
| POST | `/api/v1/calendar/import` | Import .ics |
| GET | `/api/v1/calendar/export` | Export .ics |

---

### 2.9 Settings

**Store:** `stores/settings.ts` — all settings as localStorage defaults

**11 Tabs:** Profile, Appearance, Language, Security (2FA), Notifications (7x3 matrix), Mail (IMAP/SMTP), Calendar, Finance, Team/HR, Privacy (DSGVO), About

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/settings` | Get all user settings |
| PATCH | `/api/v1/settings/profile` | Update profile |
| PATCH | `/api/v1/settings/appearance` | Theme, desk theme |
| PATCH | `/api/v1/settings/language` | Locale, timezone |
| PATCH | `/api/v1/settings/notifications` | Notification matrix |
| PATCH | `/api/v1/settings/mail` | IMAP/SMTP config |
| PATCH | `/api/v1/settings/calendar` | Calendar config |
| PATCH | `/api/v1/settings/finance` | Finance config |
| PATCH | `/api/v1/settings/team-admin` | Team admin settings |
| POST | `/api/v1/settings/security/enable-2fa` | Enable 2FA |
| POST | `/api/v1/settings/security/disable-2fa` | Disable 2FA |
| POST | `/api/v1/settings/avatar` | Upload avatar |

---

### 2.10 Infrastruktur (Admin)

**Files:** `modules/admin/InfrastrukturPage.tsx` — 7 tabs

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/admin/status` | System overview (CPU, RAM, disk) |
| GET | `/api/v1/admin/services` | Service health (API, DB, Redis, LiveKit) |
| POST | `/api/v1/admin/services/{id}/restart` | Restart service |
| GET | `/api/v1/admin/backups` | List backups |
| POST | `/api/v1/admin/backups` | Create backup |
| GET | `/api/v1/admin/logs` | System logs |
| GET | `/api/v1/admin/users/active` | Active users count |

---

### 2.11 Inventar (Warehouse/Inventory)

**Store:** `stores/inventar.ts` — articles, categories, movements, suppliers
**Files:** `modules/inventar/InventarPage.tsx` (1038 LOC)

**Frontend Features:**
- Article CRUD (SKU, barcode, dimensions, weight, min stock, location)
- Category tree management
- Stock movements (Eingang/Ausgang/Korrektur/Umlagerung)
- Supplier management with rating
- Low stock warnings, reorder suggestions
- Barcode scanning (planned)
- Export (CSV)

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/inventar/articles` | List articles (filter, search, pagination) |
| POST | `/api/v1/inventar/articles` | Create article |
| GET | `/api/v1/inventar/articles/{id}` | Get article detail |
| PATCH | `/api/v1/inventar/articles/{id}` | Update article |
| DELETE | `/api/v1/inventar/articles/{id}` | Delete article |
| GET | `/api/v1/inventar/categories` | List categories |
| POST | `/api/v1/inventar/categories` | Create category |
| PATCH | `/api/v1/inventar/categories/{id}` | Update category |
| DELETE | `/api/v1/inventar/categories/{id}` | Delete category |
| GET | `/api/v1/inventar/movements` | List stock movements |
| POST | `/api/v1/inventar/movements` | Record movement |
| GET | `/api/v1/inventar/suppliers` | List suppliers |
| POST | `/api/v1/inventar/suppliers` | Create supplier |
| PATCH | `/api/v1/inventar/suppliers/{id}` | Update supplier |
| DELETE | `/api/v1/inventar/suppliers/{id}` | Delete supplier |
| GET | `/api/v1/inventar/low-stock` | Low stock warnings |
| GET | `/api/v1/inventar/export` | CSV export |

**Complexity:** MEDIUM — standard CRUD + stock calculations

---

### 2.12 Schichtplanung (Shift Planning)

**Store:** `stores/schichten.ts` — employees, shifts, swap requests, templates
**Files:** `modules/schichten/SchichtenPage.tsx` (1096 LOC)

**Frontend Features:**
- Weekly visual grid (Mo-So x employees), colored shift blocks
- Shift CRUD (type: Frueh/Spaet/Nacht/Frei/Pikett)
- Swap requests (request, approve, reject)
- Templates (save/load week plans)
- Overtime tracking
- Conflict detection (double shifts)
- Availability preferences per employee

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/schichten/plans` | Get shift plan (week filter) |
| POST | `/api/v1/schichten/shifts` | Create shift |
| PATCH | `/api/v1/schichten/shifts/{id}` | Update shift |
| DELETE | `/api/v1/schichten/shifts/{id}` | Delete shift |
| GET | `/api/v1/schichten/employees` | List employees with availability |
| PATCH | `/api/v1/schichten/employees/{id}/availability` | Update availability |
| GET | `/api/v1/schichten/swap-requests` | List swap requests |
| POST | `/api/v1/schichten/swap-requests` | Create swap request |
| POST | `/api/v1/schichten/swap-requests/{id}/approve` | Approve swap |
| POST | `/api/v1/schichten/swap-requests/{id}/reject` | Reject swap |
| GET | `/api/v1/schichten/templates` | List templates |
| POST | `/api/v1/schichten/templates` | Save template |
| POST | `/api/v1/schichten/templates/{id}/apply` | Apply template to week |
| DELETE | `/api/v1/schichten/templates/{id}` | Delete template |

**Complexity:** MEDIUM — conflict detection, overtime calculation

---

### 2.13 Einkauf (Purchasing)

**Store:** `stores/einkauf.ts` — orders, suppliers, deliveries
**Files:** `modules/einkauf/EinkaufPage.tsx` (1209 LOC)

**Frontend Features:**
- Purchase orders CRUD (line items, supplier, delivery date)
- Supplier management with rating, payment terms
- Delivery tracking (status: bestellt/unterwegs/teilgeliefert/geliefert)
- Goods receipt (partial deliveries)
- Reorder suggestions from inventory low stock
- Approval workflow (above threshold)
- Price comparison

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/einkauf/orders` | List orders (filter, search) |
| POST | `/api/v1/einkauf/orders` | Create order |
| GET | `/api/v1/einkauf/orders/{id}` | Get order detail |
| PATCH | `/api/v1/einkauf/orders/{id}` | Update order |
| DELETE | `/api/v1/einkauf/orders/{id}` | Delete order |
| POST | `/api/v1/einkauf/orders/{id}/approve` | Approve order |
| POST | `/api/v1/einkauf/orders/{id}/send` | Send to supplier |
| POST | `/api/v1/einkauf/orders/{id}/receive` | Record goods receipt |
| GET | `/api/v1/einkauf/suppliers` | List suppliers |
| POST | `/api/v1/einkauf/suppliers` | Create supplier |
| PATCH | `/api/v1/einkauf/suppliers/{id}` | Update supplier |
| DELETE | `/api/v1/einkauf/suppliers/{id}` | Delete supplier |
| GET | `/api/v1/einkauf/deliveries` | List deliveries |
| GET | `/api/v1/einkauf/reorder-suggestions` | Reorder suggestions |

**Complexity:** MEDIUM — links to Inventar for stock levels

---

### 2.14 Helpdesk (Ticketing)

**Store:** `stores/helpdesk.ts` — tickets, SLAs, knowledge base articles
**Files:** `modules/helpdesk/HelpdeskPage.tsx` (1008 LOC)

**Frontend Features:**
- Tickets CRUD (priority, category, assignee, SLA)
- Ticket status workflow (offen → in Bearbeitung → wartend → geloest → geschlossen)
- SLA tracking with countdown timers
- Knowledge base articles (CRUD)
- Customer portal link
- Canned responses
- Ticket merge, escalation

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/helpdesk/tickets` | List tickets (filter, search) |
| POST | `/api/v1/helpdesk/tickets` | Create ticket |
| GET | `/api/v1/helpdesk/tickets/{id}` | Get ticket detail |
| PATCH | `/api/v1/helpdesk/tickets/{id}` | Update ticket |
| DELETE | `/api/v1/helpdesk/tickets/{id}` | Delete ticket |
| POST | `/api/v1/helpdesk/tickets/{id}/assign` | Assign ticket |
| POST | `/api/v1/helpdesk/tickets/{id}/escalate` | Escalate ticket |
| POST | `/api/v1/helpdesk/tickets/{id}/merge` | Merge tickets |
| POST | `/api/v1/helpdesk/tickets/{id}/comments` | Add comment |
| GET | `/api/v1/helpdesk/slas` | List SLA policies |
| GET | `/api/v1/helpdesk/kb/articles` | List KB articles |
| POST | `/api/v1/helpdesk/kb/articles` | Create KB article |
| PATCH | `/api/v1/helpdesk/kb/articles/{id}` | Update article |
| DELETE | `/api/v1/helpdesk/kb/articles/{id}` | Delete article |

**Complexity:** MEDIUM — SLA timer calculation, escalation rules

---

### 2.15 Fuhrpark (Fleet Management)

**Store:** `stores/fuhrpark.ts` — vehicles, maintenance, fuel logs, positions, routes
**Files:** `modules/fuhrpark/FuhrparkPage.tsx` (1334 LOC)

**Frontend Features:**
- Vehicle CRUD (type, license plate, mileage, insurance, TUV dates)
- Maintenance scheduling + history
- Fuel log entries (amount, cost, mileage)
- Cost analysis per vehicle
- GPS tracking tab (vehicle positions, route history)
- Driver assignment
- Document attachments per vehicle

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/fuhrpark/vehicles` | List vehicles |
| POST | `/api/v1/fuhrpark/vehicles` | Create vehicle |
| GET | `/api/v1/fuhrpark/vehicles/{id}` | Get vehicle detail |
| PATCH | `/api/v1/fuhrpark/vehicles/{id}` | Update vehicle |
| DELETE | `/api/v1/fuhrpark/vehicles/{id}` | Delete vehicle |
| GET | `/api/v1/fuhrpark/vehicles/{id}/maintenance` | Maintenance history |
| POST | `/api/v1/fuhrpark/vehicles/{id}/maintenance` | Schedule maintenance |
| PATCH | `/api/v1/fuhrpark/maintenance/{id}` | Update maintenance |
| GET | `/api/v1/fuhrpark/vehicles/{id}/fuel` | Fuel log |
| POST | `/api/v1/fuhrpark/vehicles/{id}/fuel` | Add fuel entry |
| GET | `/api/v1/fuhrpark/vehicles/{id}/costs` | Cost analysis |
| GET | `/api/v1/fuhrpark/tracking/positions` | Current positions (all vehicles) |
| GET | `/api/v1/fuhrpark/tracking/routes` | Route history (vehicle + date) |
| POST | `/api/v1/fuhrpark/tracking/positions` | Report position (from device) |

**Complexity:** HIGH — GPS tracking requires device integration + WebSocket updates

---

### 2.16 Produktion (Manufacturing)

**Store:** `stores/produktion.ts` — BOMs, production orders, quality checks
**Files:** `modules/produktion/ProduktionPage.tsx` (1199 LOC)

**Frontend Features:**
- Bill of Materials (BOM) CRUD with component tree
- Production orders (status: geplant → in Produktion → Qualitaetskontrolle → fertig)
- Quality checks with pass/fail/conditional
- Machine utilization tracking
- Material requirements planning (from BOM + orders)
- Scrap/waste tracking

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/produktion/boms` | List BOMs |
| POST | `/api/v1/produktion/boms` | Create BOM |
| GET | `/api/v1/produktion/boms/{id}` | Get BOM detail |
| PATCH | `/api/v1/produktion/boms/{id}` | Update BOM |
| DELETE | `/api/v1/produktion/boms/{id}` | Delete BOM |
| GET | `/api/v1/produktion/orders` | List production orders |
| POST | `/api/v1/produktion/orders` | Create order |
| PATCH | `/api/v1/produktion/orders/{id}` | Update order (status) |
| DELETE | `/api/v1/produktion/orders/{id}` | Delete order |
| POST | `/api/v1/produktion/orders/{id}/quality-check` | Record quality check |
| GET | `/api/v1/produktion/machines` | List machines |
| GET | `/api/v1/produktion/machines/{id}/utilization` | Machine utilization |
| GET | `/api/v1/produktion/mrp` | Material requirements |

**Complexity:** MEDIUM — BOM explosion, MRP calculation

---

### 2.17 Berichte (Reports/Analytics)

**Store:** `stores/berichte.ts` — report templates, saved reports
**Files:** `modules/berichte/BerichtePage.tsx` (921 LOC)

**Frontend Features:**
- KPI dashboard (revenue, costs, profit, headcount)
- Chart types (bar, line, pie, area)
- Custom report builder (select metrics, date range, grouping)
- Saved reports (CRUD)
- Scheduled report generation
- Export (PDF, CSV, Excel)

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/berichte/kpis` | KPI summary (date range) |
| GET | `/api/v1/berichte/charts/{type}` | Chart data (revenue/costs/etc) |
| GET | `/api/v1/berichte/saved` | List saved reports |
| POST | `/api/v1/berichte/saved` | Save report config |
| DELETE | `/api/v1/berichte/saved/{id}` | Delete saved report |
| POST | `/api/v1/berichte/generate` | Generate report (returns data) |
| GET | `/api/v1/berichte/export/{id}` | Export as PDF/CSV/Excel |
| GET | `/api/v1/berichte/scheduled` | List scheduled reports |
| POST | `/api/v1/berichte/scheduled` | Create scheduled report |
| DELETE | `/api/v1/berichte/scheduled/{id}` | Delete scheduled |

**Complexity:** HIGH — aggregation across all modules, chart data, PDF generation

---

### 2.18 Vertraege (Contract Management)

**Store:** `stores/vertraege.ts` — contracts with types, history, termination
**Files:** `modules/vertraege/VertraegePage.tsx` (1234 LOC)

**Frontend Features:**
- Contract CRUD (6 types: Mietvertrag, Liefervertrag, Servicevertrag, Arbeitsvertrag, Lizenz, Versicherung)
- Contract lifecycle (Entwurf → Aktiv → Gekuendigt → Ausgelaufen → Archiviert)
- Laufzeit visualization with progress bar
- Termination workflow (Kuendigung with date + reason)
- Reminder system (30/60/90 days before expiry)
- History/audit trail per contract
- Document attachments

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/vertraege` | List contracts (filter by type/status) |
| POST | `/api/v1/vertraege` | Create contract |
| GET | `/api/v1/vertraege/{id}` | Get contract detail |
| PATCH | `/api/v1/vertraege/{id}` | Update contract |
| DELETE | `/api/v1/vertraege/{id}` | Delete contract |
| POST | `/api/v1/vertraege/{id}/terminate` | Terminate contract |
| POST | `/api/v1/vertraege/{id}/renew` | Renew contract |
| GET | `/api/v1/vertraege/{id}/history` | Audit trail |
| GET | `/api/v1/vertraege/expiring` | Contracts expiring soon |
| GET | `/api/v1/vertraege/reminders` | Active reminders |
| POST | `/api/v1/vertraege/{id}/attachments` | Upload document |
| GET | `/api/v1/vertraege/types` | List contract types |

**Complexity:** MEDIUM — reminder cron job, audit trail

---

### 2.19 Formulare (Form Builder)

**Store:** `stores/formulare.ts` — forms, fields, submissions, templates
**Files:** `modules/formulare/FormularePage.tsx` (1493 LOC)

**Frontend Features:**
- Form builder with 9 field types (text, textarea, number, email, phone, date, select, radio, rating)
- Inline editor with live preview (split view)
- Field configuration (required, placeholder, options)
- Drag-to-reorder fields
- Form templates (duplicate to create new)
- Submissions list grouped by form
- Typed answer rendering (stars for rating, badges for select)
- Public form sharing (link generation)

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/formulare` | List forms |
| POST | `/api/v1/formulare` | Create form |
| GET | `/api/v1/formulare/{id}` | Get form with fields |
| PATCH | `/api/v1/formulare/{id}` | Update form metadata |
| DELETE | `/api/v1/formulare/{id}` | Delete form |
| POST | `/api/v1/formulare/{id}/duplicate` | Duplicate form |
| POST | `/api/v1/formulare/{id}/fields` | Add field |
| PATCH | `/api/v1/formulare/{id}/fields/{fid}` | Update field |
| DELETE | `/api/v1/formulare/{id}/fields/{fid}` | Remove field |
| POST | `/api/v1/formulare/{id}/fields/reorder` | Reorder fields |
| GET | `/api/v1/formulare/{id}/submissions` | List submissions |
| POST | `/api/v1/formulare/{id}/submit` | Submit form (public) |
| GET | `/api/v1/formulare/{id}/submissions/{sid}` | Get submission detail |
| DELETE | `/api/v1/formulare/{id}/submissions/{sid}` | Delete submission |
| GET | `/api/v1/formulare/templates` | List templates |

**Complexity:** MEDIUM — dynamic field schema, public form endpoint (no auth)

---

### 2.20 Vermietung (Rental Management)

**Store:** `stores/vermietung.ts` — objects, reservations
**Files:** `modules/vermietung/VermietungPage.tsx` (1412 LOC)

**Frontend Features:**
- Rental objects CRUD (4 types: Geraet, Raum, Fahrzeug, Werkzeug)
- Availability calendar (Objects x Days weekly grid)
- Reservation CRUD with status (reserviert/aktiv/zurueckgegeben/storniert)
- Object detail panel with reservation history
- Conflict detection (double bookings)
- Pricing (hourly/daily/weekly rates)

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/vermietung/objects` | List rental objects |
| POST | `/api/v1/vermietung/objects` | Create object |
| GET | `/api/v1/vermietung/objects/{id}` | Get object detail |
| PATCH | `/api/v1/vermietung/objects/{id}` | Update object |
| DELETE | `/api/v1/vermietung/objects/{id}` | Delete object |
| GET | `/api/v1/vermietung/objects/{id}/availability` | Check availability (date range) |
| GET | `/api/v1/vermietung/reservations` | List reservations |
| POST | `/api/v1/vermietung/reservations` | Create reservation |
| GET | `/api/v1/vermietung/reservations/{id}` | Get reservation detail |
| PATCH | `/api/v1/vermietung/reservations/{id}` | Update reservation |
| POST | `/api/v1/vermietung/reservations/{id}/return` | Mark as returned |
| POST | `/api/v1/vermietung/reservations/{id}/cancel` | Cancel reservation |
| DELETE | `/api/v1/vermietung/reservations/{id}` | Delete reservation |
| GET | `/api/v1/vermietung/calendar` | Weekly availability grid |

**Complexity:** MEDIUM — availability conflict detection, pricing calculation

---

### 2.21 Rapporte (Field Reports)

**Store:** `stores/rapporte.ts` — reports, measurements, templates
**Files:** `modules/rapporte/RapportePage.tsx` (1471 LOC)

**Frontend Features:**
- Daily field reports (project, weather, temperature, workers, activities, materials)
- Photo attachments
- Aufmass (measurements) with area/volume auto-calculation
- Report templates (save/load)
- Digital signature placeholder (planned)
- Worker time tracking per report
- PDF export

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/rapporte` | List reports (date/project filter) |
| POST | `/api/v1/rapporte` | Create report |
| GET | `/api/v1/rapporte/{id}` | Get report detail |
| PATCH | `/api/v1/rapporte/{id}` | Update report |
| DELETE | `/api/v1/rapporte/{id}` | Delete report |
| POST | `/api/v1/rapporte/{id}/photos` | Upload photos |
| GET | `/api/v1/rapporte/{id}/pdf` | Export as PDF |
| GET | `/api/v1/rapporte/measurements` | List measurements |
| POST | `/api/v1/rapporte/measurements` | Create measurement |
| PATCH | `/api/v1/rapporte/measurements/{id}` | Update measurement |
| DELETE | `/api/v1/rapporte/measurements/{id}` | Delete measurement |
| POST | `/api/v1/rapporte/measurements/{id}/positions` | Add position |
| GET | `/api/v1/rapporte/templates` | List templates |
| POST | `/api/v1/rapporte/templates` | Create template |
| DELETE | `/api/v1/rapporte/templates/{id}` | Delete template |

**Complexity:** MEDIUM — photo upload, PDF generation, measurement calculations

---

### 2.22 Mahnwesen (Dunning — Extension of Buchhaltung)

**Store:** `stores/finance.ts` — dunnings with levels 1-3
**Files:** `modules/buchhaltung/BuchhaltungPage.tsx` (Mahnungen tab)

**Frontend Features:**
- Dunning table (level 1/2/3 indicators)
- Filter by Mahnstufe + Status
- Send dunning (email)
- Escalate to next level
- Link to original invoice

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/finance/dunnings` | List dunnings |
| POST | `/api/v1/finance/dunnings` | Create dunning (from overdue invoice) |
| POST | `/api/v1/finance/dunnings/{id}/send` | Send dunning email |
| POST | `/api/v1/finance/dunnings/{id}/escalate` | Escalate to next level |
| PATCH | `/api/v1/finance/dunnings/{id}` | Update dunning status |
| GET | `/api/v1/finance/dunnings/overdue` | Auto-detect overdue invoices |

**Complexity:** LOW — extends existing finance service

---

### 2.23 Lohn (Payroll — Extension of Team)

**Store:** `stores/team.ts` — payroll entries with salary breakdown
**Files:** `modules/team/TeamPage.tsx` (Lohn tab)

**Frontend Features:**
- Monthly payroll table (gross, AHV, pension, tax, net)
- Month selector
- Lohnlauf starten (batch payroll run)
- Individual payslip view

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/team/payroll` | List payroll entries (month filter) |
| GET | `/api/v1/team/payroll/{id}` | Get payslip detail |
| POST | `/api/v1/team/payroll/run` | Start payroll run |
| GET | `/api/v1/team/payroll/run/{id}/status` | Check run status |
| PATCH | `/api/v1/team/payroll/{id}` | Adjust entry |
| GET | `/api/v1/team/payroll/{id}/pdf` | Export payslip PDF |
| GET | `/api/v1/team/payroll/summary` | Monthly summary (totals) |
| POST | `/api/v1/team/payroll/export` | Export for accounting |

**Complexity:** HIGH — Swiss social security calculations (AHV/IV/EO, BVG, tax)

---

### 2.24 Schulungen (Training — Extension of Team)

**Store:** `stores/team.ts` — trainings, participations, certificates
**Files:** `modules/team/TeamPage.tsx` (Schulungen tab)

**Frontend Features:**
- Training catalog (CRUD)
- Training types (intern, extern, online, zertifizierung)
- Participation tracking (angemeldet → teilgenommen → bestanden/nicht bestanden)
- Certificate management (upload, expiry tracking)

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/team/trainings` | List trainings |
| POST | `/api/v1/team/trainings` | Create training |
| PATCH | `/api/v1/team/trainings/{id}` | Update training |
| DELETE | `/api/v1/team/trainings/{id}` | Delete training |
| GET | `/api/v1/team/trainings/{id}/participants` | List participants |
| POST | `/api/v1/team/trainings/{id}/participate` | Record participation |
| PATCH | `/api/v1/team/participations/{id}` | Update participation status |
| POST | `/api/v1/team/certificates` | Upload certificate |
| GET | `/api/v1/team/certificates/expiring` | Expiring certificates |

**Complexity:** LOW — standard CRUD + file upload

---

### 2.25 Wiki (Knowledge Base — Extension of Dokumente)

**Store:** `stores/documents.ts` — wiki articles, categories
**Files:** `modules/dokumente/DokumentePage.tsx` (Wiki tab)

**Frontend Features:**
- Wiki articles CRUD (markdown content, tags, category)
- Category management
- Full-text search
- Recent changes feed
- Tag filtering

**Endpoints NEEDED:**

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v1/wiki/articles` | List articles (category, tag, search) |
| POST | `/api/v1/wiki/articles` | Create article |
| GET | `/api/v1/wiki/articles/{id}` | Get article |
| PATCH | `/api/v1/wiki/articles/{id}` | Update article |
| DELETE | `/api/v1/wiki/articles/{id}` | Delete article |
| GET | `/api/v1/wiki/categories` | List categories |
| POST | `/api/v1/wiki/categories` | Create category |
| GET | `/api/v1/wiki/search` | Full-text search |
| GET | `/api/v1/wiki/recent` | Recent changes |

**Complexity:** MEDIUM — full-text search (PostgreSQL tsvector or Meilisearch)

---

## 3. Dashboard Widget Data (Still Mock)

The dashboard layout API works, but widget data is not wired:

| Widget | Data Source | Status |
|--------|-----------|--------|
| RecentContacts | `stores/contacts.ts` | MOCK — needs contacts API |
| DealPipeline | Hardcoded | MOCK — needs deals API |
| UnreadMessages | `stores/mails.ts` | MOCK — needs mails API |
| ActivityFeed | Hardcoded | MOCK — needs activities API |
| QuickActions | Navigation only | OK |
| NotificationSummary | Hardcoded | MOCK — needs notifications API |

---

## 4. Endpoint Count Summary

### Core Modules (from rev 2)

| Module | Endpoints | Luke's Phase | Priority |
|--------|----------|-------------|----------|
| Kontakte (extended) | 14 | Phase 10 | HIGH |
| Calendar | 14 | Phase 7 | HIGH |
| Meetings | 13 | Phase 8 | HIGH |
| Time Tracking | 22 | Phase 6/13 | HIGH |
| Documents | 14 | Phase 11 | MEDIUM |
| Email | 16 | Phase 10 | MEDIUM |
| Team/HR | 11 | Phase 13 | MEDIUM |
| Finance | 20 | Phase 12 | MEDIUM |
| Settings | 12 | Phase 9+ | LOW |
| Infrastruktur | 7 | — | LOW |
| Work Profiles | 5 | Phase 6 | LOW |
| **Subtotal** | **~148** | | |

### Industry Modules (NEW — rev 3)

| Module | Endpoints | Priority |
|--------|----------|----------|
| Inventar | 17 | HIGH |
| Schichtplanung | 14 | HIGH |
| Einkauf | 14 | MEDIUM |
| Helpdesk | 14 | MEDIUM |
| Fuhrpark | 14 | HIGH |
| Produktion | 13 | MEDIUM |
| Berichte | 10 | MEDIUM |
| Vertraege | 12 | MEDIUM |
| Formulare | 15 | MEDIUM |
| Vermietung | 14 | MEDIUM |
| Rapporte | 15 | MEDIUM |
| **Subtotal** | **~152** | |

### Module Extensions (NEW — rev 3)

| Extension | Endpoints | Parent Module | Priority |
|-----------|----------|---------------|----------|
| Mahnwesen | 6 | Finance | MEDIUM |
| Lohn (Payroll) | 8 | Team | HIGH |
| Schulungen | 9 | Team | LOW |
| Wiki | 9 | Documents | MEDIUM |
| **Subtotal** | **~32** | | |

### Grand Total

| Category | Count |
|----------|-------|
| Core modules (mock data) | ~148 |
| Industry modules (mock data) | ~152 |
| Module extensions (mock data) | ~32 |
| **Total NEW endpoints needed** | **~332** |
| Existing but UI not wired (CRM/Chat/Work/Notifications) | ~89 |
| **Grand total integrations** | **~421** |

---

## 5. Priority List for Luke

### Tier 1 — BLOCKING (already planned):
1. **Phase 6: Task Timer** — TimeTrackerWidget in header needs real-time sync
2. **Phase 6: Gantt Chart** — Frontend may need read-only view
3. **Extended Contact Fields** — D7 KontaktePage needs more fields than CRM has

### Tier 2 — HIGH VALUE:
4. **Phase 7: Calendar** — KalenderPage fully built (4 views + room booking)
5. **Phase 8: Meetings + Video** — MeetingsPage + LiveKit integration
6. **Phase 8: Presence System** — Team page shows online/away status

### Tier 3 — MEDIUM VALUE:
7. **Phase 10: Email** — MailsPage fully built (inline compose, pop-out window, print/export/save)
8. **Phase 11: Documents** — DokumentePage fully built (upload, share, vault, versions)
9. **Phase 12: Finance** — BuchhaltungPage fully built (invoices, payments, expenses, DATEV)
10. **Phase 13: HR + Time Tracking** — TeamPage + Zeiterfassung (6 sub-views) fully built

### Tier 4 — INDUSTRY MODULES (NEW):
11. **Inventar** — 17 endpoints, standard CRUD + stock movements
12. **Schichtplanung** — 14 endpoints, weekly grid + swap requests
13. **Einkauf** — 14 endpoints, links to Inventar
14. **Helpdesk** — 14 endpoints, SLA tracking
15. **Fuhrpark** — 14 endpoints, GPS tracking needs WebSocket
16. **Produktion** — 13 endpoints, BOM + MRP
17. **Berichte** — 10 endpoints, cross-module aggregation
18. **Vertraege** — 12 endpoints, reminder cron
19. **Formulare** — 15 endpoints, public submission endpoint (no auth)
20. **Vermietung** — 14 endpoints, availability conflict detection
21. **Rapporte** — 15 endpoints, photo upload + PDF export

### Tier 5 — MODULE EXTENSIONS (NEW):
22. **Mahnwesen** — 6 endpoints, extends Finance
23. **Lohn/Payroll** — 8 endpoints, Swiss social security math (HIGH)
24. **Schulungen** — 9 endpoints, certificate tracking
25. **Wiki** — 9 endpoints, full-text search

### Tier 6 — LOW PRIORITY:
26. Settings persistence (all 11 tabs)
27. Infrastructure admin (7 tabs)
28. Work Profiles
29. Wire existing CRM/Chat/Work/Notification hooks to UI

### Key Insight:
Frontend is **significantly ahead** of backend. ~332 new endpoints needed across 20+ modules. When Luke builds each service, Zustand mock stores need to be replaced with TanStack Query hooks (same pattern as existing CRM/Chat/Work hooks in `api/hooks/`).

### DB Tables Needed (NEW modules):

| Module | Tables |
|--------|--------|
| Inventar | `articles`, `article_categories`, `stock_movements`, `suppliers` |
| Schichten | `shifts`, `shift_templates`, `swap_requests`, `employee_availability` |
| Einkauf | `purchase_orders`, `purchase_order_items`, `deliveries` |
| Helpdesk | `tickets`, `ticket_comments`, `sla_policies`, `kb_articles` |
| Fuhrpark | `vehicles`, `maintenance_records`, `fuel_logs`, `vehicle_positions`, `vehicle_routes` |
| Produktion | `boms`, `bom_components`, `production_orders`, `quality_checks`, `machines` |
| Berichte | `saved_reports`, `scheduled_reports` |
| Vertraege | `contracts`, `contract_history`, `contract_reminders` |
| Formulare | `forms`, `form_fields`, `form_submissions`, `form_answers` |
| Vermietung | `rental_objects`, `reservations` |
| Rapporte | `field_reports`, `report_workers`, `report_activities`, `report_materials`, `measurements`, `measurement_positions`, `report_templates` |
| Mahnwesen | `dunnings` (extends finance) |
| Lohn | `payroll_entries`, `payroll_runs` (extends team) |
| Schulungen | `trainings`, `training_participations`, `certificates` (extends team) |
| Wiki | `wiki_articles`, `wiki_categories` (extends documents) |

**Total new tables:** ~45

### Migration Pattern:
1. Luke builds Go service + endpoints
2. Luke updates OpenAPI spec
3. Darien creates TanStack Query hooks (following existing patterns)
4. Darien replaces Zustand store calls with hook calls in UI
5. Keep Zustand for UI-only state (sidebar, compose panel, etc.)
