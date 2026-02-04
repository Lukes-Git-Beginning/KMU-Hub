# Roadmap KMU Hub

## Phase 1: Foundation (Monat 1-2)

**Ziel:** Solides technisches Fundament

- [x] Auth Service (JWT, Refresh Tokens, RBAC)
- [ ] User Management (CRUD, Rollen, Einladungen) — teilweise via Auth Service abgedeckt
- [x] API Gateway (Routing, Rate Limiting, CORS)
- [x] PostgreSQL Schema Design + Migrations (3 Migrations)
- [x] Redis Setup (Sessions, Cache, Rate-Limit Fallback)
- [x] CI/CD Pipeline (GitHub Actions: lint, test, build, openapi-validate, e2e)
- [x] Docker-Compose fuer lokale Entwicklung (Full-Stack: postgres->redis->migrate->auth->gateway)
- [x] OpenAPI Spec v1
- [x] Logging + Monitoring Setup (slog + Prometheus Metrics + Health Checks)
- [x] E2E Test-Framework (Go E2E Tests + CI Job)

## Phase 2: CRM Core (Monat 2-4)

**Ziel:** Funktionierendes CRM-Backend

- [ ] Kontakte (CRUD, Custom Fields, Tags, Import/Export)
- [ ] Unternehmen (CRUD, Verknuepfung mit Kontakten)
- [ ] Deals / Pipeline (Kanban, Stages, Drag & Drop)
- [ ] Aktivitaeten (Calls, Meetings, Notizen, E-Mails)
- [ ] Such-Engine (PostgreSQL Full-Text Search)
- [ ] Filter + Views (gespeicherte Filter)
- [ ] Basis-Reporting (Pipeline Value, Conversion Rates)
- [ ] Config-basierte Custom Fields Engine

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

### 04.02.2026 — Abend-Session (1 Dev + AI-Coder)

**Fokus:** CI/CD Stabilisierung und E2E-Pipeline

- E2E CI Job analysiert und gefixt (Auth-Container Healthcheck schlug in CI fehl)
- `MinConns` von 5 auf 2 reduziert fuer stabilere Pool-Initialisierung in CI
- Explizites `ConnectTimeout` (10s) fuer Postgres-Verbindungen hinzugefuegt
- Auth-Container `start_period` auf 30s erhoeht
- Redis als explizite Dependency fuer Auth-Service hinzugefuegt
- Phase 1 steht nun fast vollstaendig — nur User Management (CRUD, Einladungen) fehlt noch
