# Backend Requirements Audit — KMU Hub Desktop App

> Exhaustive audit of every frontend feature, the backend APIs it needs, what already exists, and what is missing.
> Generated: 2026-02-10

---

## Table of Contents
1. [Module-by-Module Frontend Needs](#1-module-by-module-frontend-needs)
2. [Existing Backend Endpoints](#2-existing-backend-endpoints)
3. [Gap Analysis](#3-gap-analysis)
4. [Priority List](#4-priority-list)

---

## 1. Module-by-Module Frontend Needs

### 1.1 Authentication (auth)

**Files:**
- `stores/auth.ts`
- `modules/auth/LoginPage.tsx`
- `api/client.ts`
- `api/websocket.ts`

**Data Model (User):**
- id, email, firstName, lastName, roles[]

**API Endpoints USED:**
| Method | Endpoint | Purpose | Status |
|--------|----------|---------|--------|
| POST | `/api/v1/auth/login` | Login with email/password | BUILT |
| POST | `/api/v1/auth/logout` | Revoke refresh token | BUILT |
| POST | `/api/v1/auth/refresh` | Refresh access token | BUILT |
| GET | `/api/v1/auth/me` | Get current user profile | BUILT |
| POST | `/api/v1/auth/register` | Register new user | BUILT |

**Real-time:** WebSocket connection at `/api/v1/ws?token={accessToken}` (BUILT)

**Auth/Roles:** 5 roles defined in frontend: admin, manager, member, hr, it_support. Backend needs to return these in `user.roles[]`.

**Electron IPC:** `window.electronAPI.auth.getStoredTokens()`, `storeTokens()`, `clearTokens()` -- this is Electron-side, not backend.

---

### 1.2 Dashboard

**Files:**
- `stores/dashboard.ts`
- `modules/dashboard/DashboardPage.tsx`
- `modules/dashboard/widgets/ActivityFeed.tsx`
- `modules/dashboard/widgets/DealPipeline.tsx`
- `modules/dashboard/widgets/NotificationSummary.tsx`
- `modules/dashboard/widgets/QuickActions.tsx`
- `modules/dashboard/widgets/RecentContacts.tsx`
- `modules/dashboard/widgets/UnreadMessages.tsx`
- `modules/settings/DashboardSettings.tsx`
- `api/hooks/useDashboard.ts`

**API Endpoints USED:**
| Method | Endpoint | Purpose | Status |
|--------|----------|---------|--------|
| GET | `/api/v1/dashboard/layout` | Fetch user's dashboard layout | BUILT |
| PUT | `/api/v1/dashboard/layout` | Save user's dashboard layout | BUILT |
| DELETE | `/api/v1/dashboard/layout` | Reset to role defaults | BUILT |
| GET | `/api/v1/dashboard/defaults/{role}` | Get role default layout (admin) | BUILT |
| PUT | `/api/v1/dashboard/defaults/{role}` | Set role default layout (admin) | BUILT |

**Widget Data Sources (each widget calls its own APIs):**
- RecentContacts: Needs `GET /api/v1/contacts` (BUILT)
- DealPipeline: Needs `GET /api/v1/deals` + `GET /api/v1/pipeline-stages` (BUILT)
- UnreadMessages: Needs `GET /api/v1/channels/unread` (BUILT)
- ActivityFeed: Needs `GET /api/v1/activities` (BUILT)
- QuickActions: Navigation only, no API
- NotificationSummary: Needs `GET /api/v1/notifications/unread-count` (BUILT)

**Roles:** All authenticated users can view dashboard. Admin can edit role defaults.

---

### 1.3 CRM Module (Contacts, Companies, Deals, Activities, Pipeline, Search)

**Files:**
- `api/hooks/useContacts.ts`, `api/hooks/useCompanies.ts`, `api/hooks/useDeals.ts`
- `api/hooks/useActivities.ts`, `api/hooks/usePipelineStages.ts`, `api/hooks/useSearch.ts`
- `modules/crm/contacts/ContactsListPage.tsx`, `ContactDetailPage.tsx`
- `modules/crm/companies/CompaniesListPage.tsx`, `CompanyDetailPage.tsx`
- `modules/crm/deals/DealsListPage.tsx`, `DealDetailPage.tsx`, `DealPipelineView.tsx`
- `modules/crm/activities/ActivitiesListPage.tsx`
- `modules/crm/search/CRMSearchPage.tsx`
- `modules/crm/CRMLayout.tsx`

**API Endpoints USED:**

**Contacts:**
| Method | Endpoint | Purpose | Status |
|--------|----------|---------|--------|
| GET | `/api/v1/contacts` | List contacts (paginated, search, filters) | BUILT |
| GET | `/api/v1/contacts/{id}` | Get contact detail | BUILT |
| POST | `/api/v1/contacts` | Create contact | BUILT |
| PATCH | `/api/v1/contacts/{id}` | Update contact | BUILT |
| DELETE | `/api/v1/contacts/{id}` | Delete contact | BUILT |

**Companies:**
| Method | Endpoint | Purpose | Status |
|--------|----------|---------|--------|
| GET | `/api/v1/companies` | List companies | BUILT |
| GET | `/api/v1/companies/{id}` | Get company detail | BUILT |
| GET | `/api/v1/companies/{id}/contacts` | Get company's contacts | BUILT |
| POST | `/api/v1/companies` | Create company | BUILT |
| PATCH | `/api/v1/companies/{id}` | Update company | BUILT |
| DELETE | `/api/v1/companies/{id}` | Delete company | BUILT |

**Deals:**
| Method | Endpoint | Purpose | Status |
|--------|----------|---------|--------|
| GET | `/api/v1/deals` | List deals (with filters) | BUILT |
| GET | `/api/v1/deals/{id}` | Get deal detail | BUILT |
| POST | `/api/v1/deals` | Create deal | BUILT |
| PATCH | `/api/v1/deals/{id}` | Update deal | BUILT |
| DELETE | `/api/v1/deals/{id}` | Delete deal | BUILT |
| POST | `/api/v1/deals/{id}/stage` | Move deal to pipeline stage | BUILT |

**Pipeline Stages:**
| Method | Endpoint | Purpose | Status |
|--------|----------|---------|--------|
| GET | `/api/v1/pipeline-stages` | List all pipeline stages | BUILT |
| POST | `/api/v1/pipeline-stages` | Create pipeline stage | BUILT |
| PATCH | `/api/v1/pipeline-stages/{id}` | Update pipeline stage | BUILT |
| DELETE | `/api/v1/pipeline-stages/{id}` | Delete pipeline stage | BUILT |
| POST | `/api/v1/pipeline-stages/reorder` | Reorder pipeline stages | BUILT |

**Activities:**
| Method | Endpoint | Purpose | Status |
|--------|----------|---------|--------|
| GET | `/api/v1/activities` | List activities (filterable) | BUILT |
| GET | `/api/v1/activities/{id}` | Get activity detail | BUILT |
| POST | `/api/v1/activities` | Create activity | BUILT |
| PUT | `/api/v1/activities/{id}` | Update activity | BUILT |
| DELETE | `/api/v1/activities/{id}` | Delete activity | BUILT |
| POST | `/api/v1/activities/{id}/complete` | Mark activity complete | BUILT |

**Search:**
| Method | Endpoint | Purpose | Status |
|--------|----------|---------|--------|
| GET | `/api/v1/search` | Global CRM search (contacts, companies, deals, activities) | BUILT |

---

### 1.4 Chat Module

**Files:**
- `api/hooks/useChannels.ts`, `api/hooks/useMessages.ts`
- `modules/chat/ChatLayout.tsx`
- `modules/chat/channels/ChannelList.tsx`, `ChannelHeader.tsx`, `CreateChannelDialog.tsx`
- `modules/chat/messages/MessageBubble.tsx`, `MessageInput.tsx`, `MessageList.tsx`
- `modules/chat/threads/ThreadPanel.tsx`

**API Endpoints USED:**
| Method | Endpoint | Purpose | Status |
|--------|----------|---------|--------|
| GET | `/api/v1/channels` | List user's channels | BUILT |
| GET | `/api/v1/channels/{id}` | Get channel detail | BUILT |
| GET | `/api/v1/channels/{id}/members` | Get channel members | BUILT |
| POST | `/api/v1/channels` | Create channel | BUILT |
| POST | `/api/v1/channels/{id}/join` | Join public channel | BUILT |
| POST | `/api/v1/channels/{id}/leave` | Leave channel | BUILT |
| GET | `/api/v1/channels/dm` | List DM conversations | BUILT |
| POST | `/api/v1/channels/dm` | Create/get DM | BUILT |
| GET | `/api/v1/channels/unread` | Get unread counts | BUILT |
| POST | `/api/v1/channels/{id}/read` | Mark channel as read | BUILT |
| GET | `/api/v1/channels/{id}/messages` | List messages (cursor pagination) | BUILT |
| POST | `/api/v1/channels/{id}/messages` | Send message | BUILT |
| PUT | `/api/v1/messages/{id}` | Edit message | BUILT |
| DELETE | `/api/v1/messages/{id}` | Delete message | BUILT |
| GET | `/api/v1/messages/{id}/thread` | Get thread replies | BUILT |

**WebSocket Events (Real-time):**
- `message.new` -- new message in channel
- `message.updated` -- message edited
- `message.deleted` -- message deleted
- `thread.reply.new` -- new thread reply
- `typing.start` / `typing.stop` -- typing indicators
- `channel.subscribe` / `channel.unsubscribe` -- channel presence

All WebSocket events: BUILT

---

### 1.5 Notifications

**Files:**
- `api/hooks/useNotifications.ts`
- `modules/notifications/NotificationBell.tsx`, `NotificationCenter.tsx`

**API Endpoints USED:**
| Method | Endpoint | Purpose | Status |
|--------|----------|---------|--------|
| GET | `/api/v1/notifications` | List notifications (paginated, filterable) | BUILT |
| GET | `/api/v1/notifications/unread-count` | Get unread count | BUILT |
| POST | `/api/v1/notifications/{id}/read` | Mark notification as read | BUILT |
| POST | `/api/v1/notifications/read-all` | Mark all as read | BUILT |
| GET | `/api/v1/notifications/preferences` | Get notification preferences | BUILT |
| PUT | `/api/v1/notifications/preferences` | Update notification preference | BUILT |
| GET | `/api/v1/notifications/event-types` | List notification event types | BUILT |

**WebSocket Events:**
- `notification.new` -- new notification arrived
- `notification.unread_count` -- updated unread count

---

### 1.6 Work Module (Projects, Tasks, Kanban, Comments, Files, Dependencies)

**Files:**
- `api/hooks/useProjects.ts`, `api/hooks/useTasks.ts`
- `api/hooks/useTaskComments.ts`, `api/hooks/useTaskActivities.ts`, `api/hooks/useTaskFiles.ts`
- `stores/work.ts`
- `modules/work/` (20+ component files)

**API Endpoints USED:**

**Projects:**
| Method | Endpoint | Purpose | Status |
|--------|----------|---------|--------|
| GET | `/api/v1/projects` | List projects | BUILT |
| GET | `/api/v1/projects/{id}` | Get project detail | BUILT |
| POST | `/api/v1/projects` | Create project | BUILT |
| PUT | `/api/v1/projects/{id}` | Update project | BUILT |
| POST | `/api/v1/projects/{id}/archive` | Archive project | BUILT |
| GET | `/api/v1/projects/{id}/members` | List project members | BUILT |
| POST | `/api/v1/projects/{id}/members` | Add member | BUILT |
| DELETE | `/api/v1/projects/{id}/members/{userId}` | Remove member | BUILT |
| GET/POST/PUT/DELETE | `/api/v1/projects/{id}/statuses` | Status CRUD | BUILT |
| POST | `/api/v1/projects/{id}/statuses/reorder` | Reorder statuses | BUILT |
| GET/PUT | `/api/v1/projects/{id}/preferences` | User preferences | BUILT |
| POST | `/api/v1/projects/{id}/template` | Save as template | BUILT |
| POST | `/api/v1/projects/from-template` | Create from template | BUILT |

**Tasks:**
| Method | Endpoint | Purpose | Status |
|--------|----------|---------|--------|
| GET | `/api/v1/tasks` | List tasks (filterable) | BUILT |
| GET/POST/PUT/DELETE | `/api/v1/tasks/{id}` | Task CRUD | BUILT |
| POST | `/api/v1/tasks/{id}/move` | Move task (status + order) | BUILT |
| GET | `/api/v1/tasks/{id}/subtasks` | List subtasks | BUILT |
| GET/POST/DELETE | `/api/v1/tasks/{id}/dependencies` | Dependencies | BUILT |
| GET/POST/DELETE | `/api/v1/tasks/{id}/links` | Entity links | BUILT |
| GET/PUT | `/api/v1/tasks/{id}/custom-fields` | Custom fields | BUILT |
| GET/POST/PUT/DELETE | `/api/v1/tasks/{id}/comments` | Comments | BUILT |
| GET | `/api/v1/tasks/{id}/activities` | Activity log | BUILT |
| GET/POST/DELETE | `/api/v1/tasks/{id}/files` | File attachments | BUILT |
| GET | `/api/v1/work/search` | Search tasks | BUILT |

---

### 1.7 Kontakte (D7 Extended Contacts — Zustand mock store)

**Files:** `stores/contacts.ts`, `modules/kontakte/`

**IMPORTANT:** Separate from CRM contacts. Uses Zustand localStorage, NOT CRM API hooks.

**Data Model (Contact):**
- id, salutation, firstName, lastName, initials, email, phone, mobile, company, jobTitle, department
- address: { street, zip, city, country }
- website, category (employee/customer/partner), status (active/prospect/inactive)
- tags[], notes, socialMedia: { linkedin, xing }
- lastContact, projects[], createdAt, isFavorite, activities[]

**GAP:** CRM contacts API exists but uses simpler model. D7 needs: salutation, mobile, department, address object, website, category, status, tags[], socialMedia, lastContact, projects[], isFavorite, activities[]. These need to be added to the CRM contact model or handled via custom_fields.

---

### 1.8 Meetings

**Files:** `stores/meetings.ts`, `modules/meetings/`

**Data Model:** id, title, status, project, date, startTime, duration, room, isVideoCall, recurrence, reminder, description, participants[], files[], whiteboardLink

**API Endpoints NEEDED:** ~9 endpoints (list, detail, CRUD, cancel, duplicate, participants, join room)

**GAP:** NO meeting service. Phase 8.

---

### 1.9 Dokumente

**Files:** `stores/documents.ts`, `modules/dokumente/`

**Data Model:** DocFile (id, name, type, size, tags, versions, sharing) + DocFolder (id, name, parentId)

**API Endpoints NEEDED:** ~16 endpoints (CRUD, upload, download, preview, versioning, sharing, folders)

**GAP:** NO document service. Phase 11.

---

### 1.10 E-Mail

**Files:** `stores/mails.ts`, `modules/mails/`

**Data Model:** Email (from, to, cc, bcc, subject, body, attachments) + MailFolder

**API Endpoints NEEDED:** ~16 endpoints (folders, messages, send, drafts, read/unread, star, move, archive, delete)

**GAP:** NO email service. Phase 10. Requires IMAP sync + SMTP.

---

### 1.11 Kalender

**Files:** `modules/kalender/`

**API Endpoints NEEDED:** ~15 endpoints (events CRUD, calendars, categories, rooms, availability, holidays)

**GAP:** NO calendar service. Phase 7.

---

### 1.12 Team / HR

**Files:** `stores/team.ts`, `modules/team/`

**Data Model:** TeamMember (with department, status, contractType, workload) + HRRequest (vacation, sick, etc.)

**API Endpoints NEEDED:** ~13 endpoints (members, departments, HR requests, absences)

**GAP:** NO team/HR service. Phase 13.

---

### 1.13 Buchhaltung (Finance)

**Files:** `stores/finance.ts`, `modules/buchhaltung/`

**Data Model:** Invoice (with line items, payments) + Transaction + Expense

**API Endpoints NEEDED:** ~22 endpoints (invoices, transactions, expenses, PDF gen, DATEV export, reports)

**GAP:** NO finance service. Phase 12. GoBD compliance needed.

---

### 1.14 Time Tracking

**Files:** `stores/timetracking.ts`, `modules/profil/tabs/zeiterfassung/`

**Data Model:** TimeEntry, TimeCategory, TimeTemplate, WorkTarget, AbsenceRequest, TeamActivity, ActiveTimer

**API Endpoints NEEDED:** ~28 endpoints (timer, entries, categories, templates, targets, team activity, reports, absences)

**GAP:** Phase 6 has partial timer plans. Full time tracking is Phase 13.

---

### 1.15 Settings

**Files:** `stores/settings.ts`, `modules/settings/`

**11 Tabs:** Profile, Appearance, Language, Security (2FA), Notifications (7x3 matrix), Mail, Calendar, Finance, Team/HR, Privacy (DSGVO), About

**API Endpoints NEEDED:** ~24 endpoints

**GAP:** Notification prefs partially exist. Everything else saves to localStorage only.

---

### 1.16 Infrastruktur (Admin)

**Files:** `modules/admin/InfrastrukturPage.tsx`

**7 Tabs:** Overview, Services, Backups, Storage, Security, Updates, Logs

**API Endpoints NEEDED:** ~11 endpoints

**GAP:** NO admin infrastructure API. Late-stage operational tooling.

---

## 2. Existing Backend Summary

| Service | Status | Phase |
|---------|--------|-------|
| Auth | DONE | 1 |
| CRM | DONE | 2 |
| Chat | DONE | 3 |
| Notification | DONE | 4 |
| Gateway | DONE | 4-5 |
| Dashboard | DONE | 5 |
| Work | 80% DONE (missing Gantt + Timer) | 6 |

---

## 3. Gap Analysis Summary

### FULLY BUILT (Frontend wired to backend):
- Auth, CRM, Chat, Notifications, Dashboard, Work/Projects

### FRONTEND BUILT, NO BACKEND (Mock data):
| Module | Endpoints Needed | Luke's Phase | Priority |
|--------|-----------------|--------------|----------|
| Kontakte (extended) | ~2 | Phase 10 | HIGH |
| Calendar | ~15 | Phase 7 | HIGH |
| Meetings | ~9 | Phase 8 | HIGH |
| Time Tracking | ~28 | Phase 6/13 | HIGH |
| Documents | ~16 | Phase 11 | MEDIUM |
| Email | ~16 | Phase 10 | MEDIUM |
| Team/HR | ~13 | Phase 13 | MEDIUM |
| Finance | ~22 | Phase 12 | MEDIUM |
| Settings | ~24 | Phase 9+ | LOW |
| Infrastruktur | ~11 | - | LOW |
| **TOTAL** | **~159** | | |

---

## 4. Priority List for Luke

### Tier 1 — BLOCKING:
1. **Phase 6: Task Timer** — Frontend TimeTrackerWidget already in header
2. **Phase 6: Gantt Chart** — Frontend may need read-only view first
3. **Extended Contact Fields** — D7 KontaktePage needs more fields than CRM API has

### Tier 2 — HIGH PRIORITY:
4. **Phase 7: Calendar** — KalenderPage fully built
5. **Phase 8: Meetings + Video** — MeetingsPage fully built
6. **Phase 8: Presence System** — Team page shows online/away status

### Tier 3 — MEDIUM PRIORITY:
7. **Phase 10: Email** — MailsPage fully built
8. **Phase 11: Documents** — DokumentePage fully built
9. **Phase 12: Finance** — BuchhaltungPage fully built
10. **Phase 13: HR + Time Tracking** — TeamPage + Zeiterfassung (6 views!) fully built

### Tier 4 — LOW PRIORITY:
11. Settings persistence
12. User Profile persistence
13. Infrastructure admin

### Key Insight:
Frontend is SIGNIFICANTLY ahead of backend. When Luke builds each service, Zustand stores need to be replaced with TanStack Query hooks (same pattern as CRM/Chat/Work).

### Recommended Approach:
1. Luke finishes Phase 6 (Gantt + Timer)
2. Phase 7 (Calendar) — frontend ready
3. Phase 8 (Meetings/Video) — frontend ready
4. For D7 modules, Luke builds backend per roadmap (Phases 10-13), then Darien wires frontend to real APIs
