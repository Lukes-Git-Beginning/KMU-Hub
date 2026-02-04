# Roadmap KMU Hub

## Phase 1: Foundation (Monat 1-4)

**Ziel:** Solides technisches Fundament

- [ ] Auth Service (JWT, Refresh Tokens, RBAC)
- [ ] User Management (CRUD, Rollen, Einladungen)
- [ ] API Gateway (Routing, Rate Limiting, CORS)
- [ ] PostgreSQL Schema Design + Migrations
- [ ] Redis Setup (Sessions, Cache)
- [ ] CI/CD Pipeline (GitHub Actions)
- [ ] Docker-Compose fuer lokale Entwicklung
- [ ] OpenAPI Spec v1
- [ ] Logging + Monitoring Setup (slog + Prometheus)
- [ ] E2E Test-Framework

## Phase 2: CRM Core (Monat 5-8)

**Ziel:** Funktionierendes CRM-Backend

- [ ] Kontakte (CRUD, Custom Fields, Tags, Import/Export)
- [ ] Unternehmen (CRUD, Verknuepfung mit Kontakten)
- [ ] Deals / Pipeline (Kanban, Stages, Drag & Drop)
- [ ] Aktivitaeten (Calls, Meetings, Notizen, E-Mails)
- [ ] Such-Engine (PostgreSQL Full-Text Search)
- [ ] Filter + Views (gespeicherte Filter)
- [ ] Basis-Reporting (Pipeline Value, Conversion Rates)
- [ ] Config-basierte Custom Fields Engine

## Phase 3: Chat & Messaging (Monat 9-12)

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

## Phase 4: Desktop App (Monat 13-16)

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

## Phase 5: Mobile App (Monat 17-18)

**Ziel:** React Native Mobile Client

- [ ] React Native Projekt-Setup
- [ ] CRM Core Features (read/create Kontakte, Deals)
- [ ] Chat (Channels, DMs)
- [ ] Push Notifications
- [ ] Kamera fuer Visitenkarten-Scan

## Phase 6: Video + Beta (Monat 18-20)

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
| 4 | Foundation complete, API laeuft |
| 8 | CRM-Backend feature-complete |
| 12 | Chat funktioniert E2E |
| 16 | Desktop App Alpha |
| 18 | Mobile App Alpha |
| 20 | Beta Launch mit ersten Kunden |
