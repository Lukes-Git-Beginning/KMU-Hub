# Roadmap — Cosmi by Zentria

> **Letztes Update:** 2026-04-08
> **Status:** Pre-Launch Sprint (08.04 – 01.05)
> **Launch-Datum:** 01.05.2026 (UG-Gruendung + Pilot-Start)
> **Detaillierte Docs:** `docs/PERFORMANCE-PLAN.md`, `docs/PRICING.md`, `docs/ARCHITECTURE.md`

---

## Projektuebersicht

**Cosmi** ist eine All-in-One Business-Plattform fuer DACH-KMUs (5-200 Mitarbeiter).
Entwickelt von **Zentria** (Luke, Darien, Nico, Moritz). EU-gehostet (Hetzner Nuernberg), DSGVO by Design.

| Komponente | Technologie |
|-----------|-------------|
| Backend | 10 Go Microservices (Gateway + 9 gRPC Services) |
| Desktop | Electron + React + TypeScript + TanStack Query |
| Datenbank | PostgreSQL 16 + Redis 7 (Cache only) |
| Video | LiveKit (self-hostable) |
| Dokumente | OnlyOffice (WOPI) |
| Plugins | Config-basiert + WASM (wazero) |
| i18n | 4 Sprachen (DE/EN/FR/IT), 7.221 Keys |
| Hosting | Hetzner CPX42, Docker Compose, Caddy TLS |

---

## Was ist fertig

### Feature Development — 20 Phasen, 103 Plaene (abgeschlossen 2026-02-26)

| Meilenstein | Phasen | Datum |
|------------|--------|-------|
| Foundation (Auth, CRM, Chat) | 1-3 | Pre-GSD |
| Pilot MVP (Gateway, Desktop, Work, Kalender, Video) | 4-8 | 2026-02-11 |
| Compliance & Comms (Security, Design, Dokumente) | 9-11 | 2026-02-17 |
| Business Suite (Finanzen, HR) | 12-13 | 2026-02-19 |
| Aggregation & Automation (Inbox, CalDAV, Workflows) | 14-16 | 2026-02-20 |
| Integrations (Teams/Slack, Guest Chat, Bexio, DATEV) | 17-19 | 2026-02-26 |
| Extensibility (WASM Plugins, Industry Templates) | 20 | 2026-02-26 |

### Beta Hardening — Phase A + B (abgeschlossen 2026-04-01)

**Phase A — Core Wiring (Maerz 2026)**
- 9 Core-Module auf echte API-Hooks migriert
- D9 Design-Merge (Waves 15-20)
- Lint-Cleanup: 347 ESLint-Probleme → 0

**Phase B — Beta Hardening (Maerz-April 2026)**
- B1-B8: UI-Hardening (~210 Dateien, 26 Custom-Modals → Radix Dialog, ~500 Hardcoded Colors → Design-Tokens)
- B9: 9 Modul-Crashes gefixt, MSW → Fetch-Interceptor
- B10: Design Audit (36 Screenshots), Cosmi-Rebrand (36 Dateien), Locale de-CH → de-DE

**i18n Migration (2026-04-06 bis 2026-04-08)**
- react-intl → i18next v26 + react-i18next v17
- 7.221 Keys × 4 Sprachen, strict TypeScript types

**Performance Optimierung (2026-04-08)**
- Phase 1: Bundle-Analyzer, Fonts self-hosted, Demo Dead Code, motion entfernt
- Phase 2: Chunk-Splitting, Async Persister, HR Polling, React Compiler, List Virtualization
- Phase 3: N+1 Queries (Contact 61→4, Deal 121→7), Batch-Inserts, owner_id Index, Pool Fix, PG Tuning
- Phase 4: V8 Compile Cache, modulePreload, Skeleton Screen
- Phase 5: Audit Logger Worker Pool, gRPC Keep-Alive, pprof (Redis Caching offen)

### Infrastruktur (live seit 2026-03-08)

- Hetzner CPX42: alle 10 Services + Gateway healthy
- `app.zentria.tech` mit HTTPS/HSTS via Caddy
- Deploy-Pipeline: `deploy.sh` mit Lock, Auto-Rollback, Smoke Gate
- Backup-Cron (taeglich 02:00), Prometheus + Grafana
- 5 CI Workflows (Backend, Desktop, CD, PR Review, Security Scan)

### Website (launch-ready)

- `zentria.tech` auf Vercel (Astro 5)
- Analytics (Plausible, GSC, Bing)
- Legal-Platzhalter fuer Post-UG-Daten

---

## Pre-Launch Sprint (08.04 – 01.05)

**23 Tage bis Launch. Fokus: Performance + Polish + Demo-Qualitaet.**

### Woche 1-2 (08.04 – 21.04) — Performance ✅ ERLEDIGT

Alle Performance-Optimierungen in einer Session abgeschlossen (5 parallele Agenten):

| # | Task | Status |
|---|------|--------|
| 1 | Vite Chunk-Splitting | ✅ |
| 2 | HR Polling 30s → 5min | ✅ |
| 3 | `modulePreload` aktivieren | ✅ |
| 4 | React Query → Async Persister (IndexedDB) | ✅ |
| 5 | localStorage Keys (`cosmi-*`) | ✅ |
| 6 | N+1 Queries Contact List (61 → 4) | ✅ |
| 7 | N+1 Queries Deal List (121 → 7) | ✅ |
| 8 | Connection Pool Fix (MaxConns 25 → 10) | ✅ |
| 9 | Index `contacts.owner_id` | ✅ |
| 10 | Batch-Inserts Tags & Custom Fields | ✅ |
| + | React Compiler (annotation mode, 3 Components) | ✅ |
| + | V8 Compile Cache | ✅ |
| + | Audit Logger Worker Pool | ✅ |
| + | Skeleton Screen | ✅ |
| + | gRPC Keep-Alive + pprof | ✅ |
| + | PG Tuning (shared_buffers, work_mem) | ✅ |
| + | List Virtualization (ContactsListPage) | ✅ |

### Woche 3 (22.04 – 28.04) — Demo-Qualitaet

| # | Task | Aufwand | Impact |
|---|------|---------|--------|
| 11 | **Kommunikation-Modul Fixes** (9 TODOs: broken filters, no-op notes/tags) | 1 Tag | Hoch — sichtbar kaputt |
| 15 | **Demo-Mode Visual Verification** (alle Screens durchklicken) | 2 Std | Hoch — Qualitaetssicherung |

### Woche 4 (29.04 – 01.05) — Final Polish + Deploy

| # | Task | Aufwand | Impact |
|---|------|---------|--------|
| 16 | **Prod-Build + Bundle-Report Analyse** | 1 Std | Verification |
| 17 | **Prod-Deployment auf `app.zentria.tech`** | 1 Std | Launch |
| 18 | **Smoke Tests auf Production** | 30 Min | Verification |
| 19 | **Website: Echte Firmendaten eintragen** (nach UG-Gruendung) | 1 Std | Legal-Pflicht |

### Bewusst NICHT im Sprint

| Item | Grund |
|------|-------|
| Redis Caching Layer | Erst relevant bei >10 gleichzeitigen Usern |
| PgBouncer | Connection Pool Fix reicht vorerst |
| FinAPI Banking-Integration | Kein Backend-Pfad, zu gross fuer Sprint |
| OnlyOffice → Collabora Migration | Lizenz-Risiko erst bei kommerziellem Einsatz relevant |
| Desktop-Installer Pipeline | Piloten bekommen Dev-Build oder gepacktes Archiv |
| Teams/Slack live testen | Kein Pilot-Kunde braucht das initial |
| Kubernetes | Docker Compose reicht fuer CPX42 |
| Mehr Desktop-Tests | Wichtig, aber nicht launch-blocking |

---

## Dialer-Modul (April – Juni 2026)

**Ziel:** Von Click-to-Call MVP zur marktfaehigen CRM+Telefonie-Loesung.
**Strategische Roadmap:** `docs/DIALER-ROADMAP.md`

### Phase 1 — MVP: Click-to-Call + Kampagnen + Preview-Mode

| Sub-Phase | Scope | Status |
|-----------|-------|--------|
| **1A — Foundation** | Proto (27 RPCs), 5 Migrations (063-067), Service-Skeleton, Gateway-Stub, Docker | ✅ Erledigt (09.04) |
| **1B — Backend Core** | Service-Logik (24 Methoden), 4 Repositories, Redis Agent-Status, CRM-Bridge, E.164, gRPC-Server, 3 Automation-Trigger | ✅ Erledigt (09.04) |
| **1C — Gateway + Demo-Mode** | REST-API (25 Endpoints), Permission-Migration, interaktiver Demo-Mode | Offen |
| **1D — Frontend** | 15+ Komponenten, DialerWorkspace, CampaignList, Dashboard, AgentStatusBar, Click-to-Call, i18n (4 Sprachen) | Offen |
| **1E — Integration** | CRM-Timeline, Callback-Notifications, CRM-Filter-Import, E2E-Verifikation | Offen |

**Architektur-Entscheidungen:**
- Eigenstaendiger `dialer`-Service (Port 50061/9101) — 11. Microservice
- CRM-Bridge via gRPC (Interface fuer Phase-2 HubSpot-Erweiterung)
- Call-State als Append-Only Event-Log (`dialer_call_events`)
- `FOR UPDATE SKIP LOCKED` fuer Phase-2 Power-Dialer-Parallelitaet
- `tenant_id` auf allen Tabellen von Tag 1

### Phase 2 — PSTN Power Dialer (geplant Q3 2026)

SIP-Telefonie via LiveKit SIP Gateway, Power-Dialer Auto-Advance, HubSpot-Sync, DSGVO-Recording, Supervisor-View, DNC-Listen.

### Phase 3 — Contact-Center-Plattform (geplant Q4 2026+)

Predictive Dialing (Erlang-C), Inbound ACD, AI Call Scoring, Multi-Tenant API.

---

## Post-Launch Roadmap (Mai – September 2026)

### Phase C — Pilot-Betrieb (Mai-Juni 2026)

**Ziel:** 3 kostenlose Pilot-KMUs onboarden, Feedback sammeln.

| Track | Tasks |
|-------|-------|
| **Legal** | AVV/DPA mit Anwalt finalisieren, AGB + Datenschutzerklaerung, Impressum mit HRB-Nummer |
| **Onboarding** | Pilot-Kunden einrichten, Onsite-Prozessanalyse (1. Woche), Config/Branchenpakete |
| **Demo-Video** | Hero-Video (3-5 Min) + Outreach-Video (60-90 Sek) aufnehmen |
| **Monitoring** | Fehler-Tracking aufsetzen, Performance-Metriken sammeln |
| **Feedback-Loop** | Woechentliche Pilot-Calls, Issue-Tracking |

### Phase D — Production Hardening (Juni-Juli 2026)

**Ziel:** Von Demo-Qualitaet zu Production-Qualitaet.

| Task | Prioritaet |
|------|-----------|
| Redis Caching Layer (Cache-Aside, 3 Tiers) | Hoch |
| PgBouncer als DB Connection Pool | Hoch |
| Desktop-Test-Coverage aufbauen (28/36 Module bei 0%) | Hoch |
| ~~React Compiler 1.0~~ | ✅ Vorgezogen (annotation mode) |
| ~~List Virtualization~~ | ✅ Vorgezogen (ContactsListPage) |
| ~~gRPC Keep-Alive Tuning~~ | ✅ Vorgezogen |
| ~~pprof Profiling~~ | ✅ Vorgezogen |
| ~~PostgreSQL Tuning~~ | ✅ Vorgezogen |
| OnlyOffice → Collabora Migration | Niedrig |
| Auto-Deploy CD Pipeline (on merge to main) | Niedrig |

### Phase E — Kommerzieller Launch (August-September 2026)

**Ziel:** Erster zahlender Kunde, Marketing-Maschine starten.

| Task | Prioritaet |
|------|-----------|
| Stripe/Payment-Integration | Kritisch |
| Desktop-Installer Pipeline (electron-builder) | Kritisch |
| Self-Hosted ORBIT Paket (Docker auf Synology NAS) | Hoch |
| Pricing-Seite auf zentria.tech | Hoch |
| LinkedIn-Praesenz + Cold Outreach (UWG §7 konform) | Hoch |
| FinAPI Banking-Integration | Mittel |
| Teams/Slack Webhooks live schalten | Mittel |
| Industry-Module verdrahten (bei Pilot-Bedarf) | Nach Bedarf |
| Pitch Deck finalisieren | Mittel |

---

## Scope-Entscheidungen

### 11 Industry-Module auf Demo-Daten

Einkauf, Inventar, Produktion, Vermietung, Fuhrpark, Rapporte, Schichten, Vertraege, Wiki, Formulare, Berichte — bleiben auf Mock-Daten bis ein Pilot-Kunde sie braucht. Verdrahtung ueber Plugin-System (Phase 20).

### Kommunikation-Modul: Teilfix

9 TODOs (broken filters, no-op notes/tags, missing thread model). Im Pre-Launch Sprint werden die sichtbarsten Bugs gefixt; volles Threading ist Post-Launch.

### Backend-Wiring noch offen

Einige Frontend-Formulare senden Felder nicht ans Backend (`EmployeeProfile` fehlt `birthday`, `phone`, `skills`, `location`). Wird bei Pilot-Feedback priorisiert.

---

## Kritische Blocker

| Blocker | Status | Deadline |
|---------|--------|----------|
| UG-Gruendung Zentria | Notar-Termin geplant | 01.05.2026 |
| AVV/DPA (Auftragsverarbeitungsvertrag) | Offen, braucht UG | Juni-Juli 2026 |
| AGB + Impressum | Platzhalter live | Nach UG |
| ~~Backend N+1 Queries~~ | ✅ Behoben (Contact 61→4, Deal 121→7) | 08.04.2026 |
| ~~Connection Pool Exhaustion~~ | ✅ Behoben (MaxConns 25→10, PG Tuning) | 08.04.2026 |

---

## Metriken & Ziele

| Metrik | Aktuell | Ziel Launch (01.05) | Ziel Q3 2026 |
|--------|---------|---------------------|--------------|
| Backend-Services healthy | 11/11 | 11/11 | 11/11 |
| DB-Queries pro Contact-Liste | ~~61~~ 4 | 4 | 4 |
| DB-Queries pro Deal-Liste | ~~121~~ 7 | 7 | 7 |
| Desktop-Test-Coverage | ~5% | ~5% | 50%+ |
| Backend-Test-Coverage | Solide (240+ CRM, 162+ Gateway) | Solide | 80%+ |
| Sprachen | 4 (DE/EN/FR/IT) | 4 | 4 |
| Pilot-Kunden | 0 | 3 (kostenlos) | 3-5 |
| Zahlende Kunden | 0 | 0 | 5-10 |
| MRR | 0 EUR | 0 EUR | 5.000+ EUR |

---

## Revenue-Projektion

| Zeitraum | Konservativ | Optimistisch |
|----------|------------|-------------|
| Jahr 1 | 196.000 EUR | 335.000 EUR |
| Break-Even | 1 zahlender Kunde | 1 zahlender Kunde |
| Fixkosten | ~50 EUR/Monat | ~50 EUR/Monat |

Pricing: Modul-x-User (2-7 EUR/User/Modul), Branchenpakete ab ~25 EUR/User/Mo, ORBIT Self-Hosted ab 4.000 EUR/Jahr.

---

*Naechstes Update: nach Pre-Launch Sprint (01.05.2026)*
