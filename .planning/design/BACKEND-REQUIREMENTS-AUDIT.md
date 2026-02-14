# Backend Requirements Audit — KMU Hub Desktop App

> Exhaustive audit of every frontend feature, the backend APIs it needs, what already exists, and what is missing.
> Updated: 2026-02-14 (rev 2 — includes D7-D9 new features)

---

## Quick Summary

| Status | Count |
|--------|-------|
| Backend BUILT + Frontend wired | Auth, CRM, Chat, Notifications, Dashboard, Work |
| Backend BUILT but Frontend NOT wired yet | CRM hooks, Work hooks, Chat hooks, Notification hooks |
| Frontend BUILT, NO backend (mock data) | ~180 endpoints across 10 modules |

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
| **TOTAL** | **~148 new** | | |

Plus: CRM (~30), Chat (~17), Work (~35), Notifications (~7) endpoints are BUILT but UI not wired yet.

**Grand total:** ~148 new + ~89 existing-but-unwired = **~237 endpoint integrations**

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

### Tier 4 — LOW PRIORITY:
11. Settings persistence (all 11 tabs)
12. Infrastructure admin (7 tabs)
13. Work Profiles
14. Wire existing CRM/Chat/Work/Notification hooks to UI

### Key Insight:
Frontend is **significantly ahead** of backend. When Luke builds each service, Zustand mock stores need to be replaced with TanStack Query hooks (same pattern as existing CRM/Chat/Work hooks in `api/hooks/`).

### Migration Pattern:
1. Luke builds Go service + endpoints
2. Luke updates OpenAPI spec
3. Darien creates TanStack Query hooks (following existing patterns)
4. Darien replaces Zustand store calls with hook calls in UI
5. Keep Zustand for UI-only state (sidebar, compose panel, etc.)
