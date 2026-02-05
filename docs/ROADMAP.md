# Roadmap KMU Hub

## Phase 1: Foundation (Monat 1-2)

**Ziel:** Solides technisches Fundament

- [x] Auth Service (JWT, Refresh Tokens, RBAC)
- [x] User Management (CRUD, Rollen, Einladungen)
- [x] API Gateway (Routing, Rate Limiting, CORS)
- [x] PostgreSQL Schema Design + Migrations (7 Migrations)
- [x] Redis Setup (Sessions, Cache, Rate-Limit Fallback)
- [x] CI/CD Pipeline (GitHub Actions: lint, test, build, openapi-validate, e2e)
- [x] Docker-Compose fuer lokale Entwicklung (Full-Stack: postgres->redis->migrate->auth->gateway)
- [x] OpenAPI Spec v1
- [x] Logging + Monitoring Setup (slog + Prometheus Metrics + Health Checks)
- [x] E2E Test-Framework (Go E2E Tests + CI Job)

## Phase 2: CRM Core (Monat 2-4)

**Ziel:** Funktionierendes CRM-Backend

### Sprint 1: Foundation (complete)
- [x] Custom Fields Engine (Definitions CRUD + Validator)
- [x] Tags System (CRUD + Junction Tables)

### Sprint 2: Core Entities (complete)
- [x] Kontakte (CRUD, Custom Fields, Tags, Search)
- [x] Unternehmen (CRUD, Kontakt-Verknuepfung)

### Sprint 3: Deals Pipeline
- [ ] Pipeline Stages (CRUD, Reorder, Defaults)
- [ ] Deals (CRUD, Stage Transitions)

### Sprint 4: Activities & Search
- [ ] Aktivitaeten (Calls, Meetings, Notizen, E-Mails, Tasks)
- [ ] Such-Engine (PostgreSQL Full-Text Search)

### Sprint 5: Filters & Reporting
- [ ] Filter + Views (gespeicherte Filter)
- [ ] Basis-Reporting (Pipeline Value, Conversion Rates)

## Phase 3: Chat & Messaging (Monat 4-5)

**Ziel:** Internes Kommunikations-Tool

- [ ] WebSocket-basierter Chat Service
- [ ] Channels (public, private)
- [ ] Direct Messages
- [ ] Threads
- [ ] File Sharing (Uploads, Previews)
- [ ] Mentions (@user, @channel)
- [ ] Notifications (In-App, Push)
- [ ] Read Receipts + Typing Indicators
- [ ] Chat-Suche

## Phase 4: Desktop App (Monat 5-7)

**Ziel:** Electron Desktop Client

- [ ] Electron Shell + Auto-Update
- [ ] React Component Library (DaisyUI/Tailwind)
- [ ] CRM UI (Kontakte, Deals, Pipeline, Aktivitaeten)
- [ ] Chat UI (Channels, DMs, Threads)
- [ ] System Tray Integration
- [ ] Native Notifications
- [ ] Keyboard Shortcuts
- [ ] Offline-Support (lokaler Cache)
- [ ] Multi-Window Support

## Phase 5: Mobile App (Monat 7-8)

**Ziel:** React Native Mobile Client

- [ ] React Native Projekt-Setup
- [ ] CRM Core Features (read/create Kontakte, Deals)
- [ ] Chat (Channels, DMs)
- [ ] Push Notifications
- [ ] Kamera fuer Visitenkarten-Scan

## Phase 6: Video + Beta (Monat 8-10)

**Ziel:** Video-Integration und Beta-Launch

- [ ] LiveKit Integration
- [ ] 1:1 Video Calls
- [ ] Gruppen-Calls
- [ ] Screen Sharing
- [ ] Call Recording
- [ ] WASM Plugin System (SDK + Dokumentation)
- [ ] Self-Hosted Deployment Package (Docker-Compose)
- [ ] Beta-Kunden Onboarding (3-5 Kunden)
- [ ] Feedback-Loop + Iteration

## Meilensteine

| Monat | Meilenstein |
|-------|------------|
| 2 | Foundation complete, API laeuft |
| 4 | CRM-Backend feature-complete |
| 5 | Chat funktioniert E2E |
| 7 | Desktop App Alpha |
| 8 | Mobile App Alpha |
| 10 | Beta Launch mit ersten Kunden |

> **Anmerkung:** Urspruengliche Schaetzung war 18-20 Monate (1 Dev, ohne AI).
> Durch AI-gestuetzte Entwicklung (1 Dev + AI-Coder) wurde die Timeline auf 8-10 Monate komprimiert.
> Phase 1 wurde groesstenteils in wenigen Abend-Sessions fertiggestellt.

---

## Fortschritts-Log

### 05.02.2026 — Phase 2 Sprint 2 Complete (1 Dev + AI-Coder)

**Fokus:** CRM Core — Contacts & Companies (Sprint 2)

- Contacts vollstaendig implementiert:
  - Migration 000007 (contacts, companies, custom_field_values, FK constraints)
  - `internal/crm/contact/` Package (service, repository, postgres_repository, errors)
  - `internal/models/contact.go` — Contact + ContactWithRelations Models
  - gRPC Server Methoden (CRUD + AddTags, RemoveTags)
  - HTTP Gateway Endpoints: `/api/v1/contacts`, `/api/v1/contacts/{id}/tags`
  - 33 Unit Tests (100% Coverage fuer Service Layer)
- Companies vollstaendig implementiert:
  - `internal/crm/company/` Package (service, repository, postgres_repository, errors)
  - `internal/models/company.go` — Company + CompanyWithRelations Models
  - gRPC Server Methoden (CRUD + GetCompanyContacts)
  - HTTP Gateway Endpoints: `/api/v1/companies`, `/api/v1/companies/{id}/contacts`
  - 26 Unit Tests (100% Coverage fuer Service Layer)
- Features:
  - Email Uniqueness (case-insensitive, optional)
  - Company-Contact Linking (SET NULL on delete)
  - Tags per Entity (via junction tables)
  - Custom Fields Storage (JSONB)
  - Search by name/email
- **Sprint 2 Core Entities ist damit abgeschlossen**

---

### 05.02.2026 — Phase 2 Sprint 1 Complete (1 Dev + AI-Coder)

**Fokus:** CRM Core — Tags System (Sprint 1 Task 2)

- Tags System vollstaendig implementiert:
  - Migration 000006 (tags Tabelle + 4 Junction Tables + Permissions)
  - `internal/crm/tag/` Package (service, repository, postgres_repository, errors)
  - `internal/models/tag.go` — Tag Model mit Hex-Color-Validation
  - gRPC Server Methoden (Create, Get, List, Update, Delete)
  - HTTP Gateway Endpoints: `/api/v1/tags`
  - 31 Unit Tests (100% Coverage fuer Service Layer)
- Tags Features:
  - Scoped per Entity Type (contact, company, deal, activity)
  - Hex Color Validation (#rrggbb Format)
  - Duplicate Name Detection per Entity Type
  - In-Use Check vor Loeschung
- **Sprint 1 Foundation ist damit abgeschlossen**

---

### 05.02.2026 — Phase 2 Start (1 Dev + AI-Coder)

**Fokus:** CRM Core — Custom Fields Engine (Sprint 1 Task 1)

- CRM Microservice Grundstruktur aufgesetzt:
  - `cmd/crm/main.go` — CRM Service Entry Point
  - `internal/crm/customfield/` — Custom Fields Domain Package
  - `proto/crm/v1/crm.proto` — gRPC Service Definition (alle CRM RPCs)
- Custom Field Definitions implementiert:
  - Migration 000005 (custom_field_definitions Tabelle + Permissions)
  - Service Layer mit Validation (Entity Type, Field Type, Options fuer Select/Multiselect)
  - Repository Interface + PostgreSQL Implementation
  - gRPC Server + HTTP Gateway Endpoints
  - Unit Tests (100% Coverage fuer Service Layer)
- Gateway erweitert: CRM gRPC Client + Custom Fields HTTP Handlers
- Docker-Compose aktualisiert: CRM Service Container
- Dockerfile.crm erstellt

---

### 05.02.2026 — Session 1 (1 Dev + AI-Coder)

**Fokus:** Phase 1 abgeschlossen — User Management komplett

- User Management Feature vollstaendig implementiert:
  - GET /auth/me — Profil abrufen
  - POST /auth/change-password — Passwort aendern
  - POST/GET/DELETE /invitations — Einladungssystem (erstellen, auflisten, stornieren)
  - POST /invitations/{token}/accept — Einladung annehmen
- Database Migration 000004 (invitations Tabelle)
- gRPC Service + HTTP Gateway erweitert
- Unit Tests + E2E Tests hinzugefuegt
- CI Pipeline gruen (alle 5 Jobs: lint, test, build, openapi-validate, e2e)
- **Phase 1 ist damit vollstaendig abgeschlossen**

---

### 04.02.2026 — Abend-Session (1 Dev + AI-Coder)

**Fokus:** CI/CD Stabilisierung und E2E-Pipeline

- E2E CI Job analysiert und gefixt (Auth-Container Healthcheck schlug in CI fehl)
- `MinConns` von 5 auf 2 reduziert fuer stabilere Pool-Initialisierung in CI
- Explizites `ConnectTimeout` (10s) fuer Postgres-Verbindungen hinzugefuegt
- Auth-Container `start_period` auf 30s erhoeht
- Redis als explizite Dependency fuer Auth-Service hinzugefuegt
- Phase 1 steht nun fast vollstaendig — nur User Management (CRUD, Einladungen) fehlt noch
