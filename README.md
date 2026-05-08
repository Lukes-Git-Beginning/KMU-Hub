# Cosmi

> All-in-One CRM für DACH-KMUs mit EU-Datensouveränität — maßgeschneidert durch einwöchige Onsite-Prozessanalyse.

**Hersteller:** Zentria UG (i.G.) · **Software:** Cosmi · **Production:** [app.zentria.tech](https://app.zentria.tech)

---

## Vision

Standardsoftware zwingt KMUs in Workflows, die nicht zu ihren Prozessen passen. Cosmi dreht das Verhältnis um: Eine einwöchige Onsite-Prozessanalyse beim Pilot-Kunden ist Teil des Onboardings, das Ergebnis fließt direkt in die Konfiguration der Instanz. Branchenmodule decken Handwerk, Fuhrpark, Vermietung, Schichten, Rapporte, Vertragsverwaltung und mehr ab — als first-class Module, nicht als Add-Ons.

**Zielgruppe:** Branchenunabhängige KMUs im DACH-Raum, 5–200 Mitarbeiter.

**USP:**

- **Onsite-Maßanfertigung** statt SaaS-One-Size-Fits-All
- **EU-Datensouveränität** — Hosting ausschließlich in der EU (Hetzner Nürnberg/Falkenstein)
- **Self-Hosted oder SaaS** — eine Codebase, beide Modelle
- **Instanz-pro-Pilot** ab Stufe Orbit (eigene Datenbank, kein Multi-Tenancy-Compromise)
- **DACH-First** — EUR, MWSt-konform, GoBD-tauglich, DSGVO mit Audit-Logging und Vault-Service

---

## Status

| | |
|---|---|
| **Phase** | Pre-Launch (Sprint 3 abgeschlossen 2026-05-08) |
| **Launch-Datum** | 2026-07-01 (ZFA-Pilot-0) |
| **Version** | 0.1.0 (Pre-Release) |
| **Production** | [app.zentria.tech](https://app.zentria.tech) — 25 Application-Services, Migration-Head 115, alle Container healthy |
| **Repo** | `github.com/Lukes-Git-Beginning/KMU-Hub` (privat) |
| **Roadmap** | [`docs/ROADMAP.md`](docs/ROADMAP.md) (Single Source of Truth, 6-Sprint-Plan bis Launch) |
| **Code-Reife** | Kombinierte Launch-Reife 3.7 (Rigorosum Runde 2 — Note 4.1; Runde 3 in Sprint 5) |

---

## Module

14 first-class Module + 5 Kommunikations-/Admin-Module. Jedes Modul ist via Feature-Flag (`modules.<name>`) toggleable und gehört zu einem Pricing-Tier.

### Kernmodule (Cosmi-Tier)

| Modul | Beschreibung |
|---|---|
| **CRM** | Kontakte, Deals, Pipeline, Custom Fields, Tags, Consent-Management |
| **Chat** | Echtzeit-Kommunikation, Threads, Reactions, Datei-Austausch |
| **E-Mail** | IMAP/SMTP mit CRM-Verknüpfung |
| **Kalender** | Termine + CalDAV-Sync, Ressourcen-Buchung |
| **Aufgaben & Projekte** | Kanban, Gantt, Zeiterfassung, Task-Dependencies |
| **Dokumente** | OnlyOffice/Collabora WOPI-Integration, Volltextsuche |
| **Finanzen** | Rechnungen (ZUGFeRD), Angebote, Mahnwesen, DATEV-Export, Bexio/Lexware-Sync |
| **Personal (HR)** | Mitarbeiter, ArbZG-Compliance, Abwesenheiten |
| **Wiki** | FTS-basierte interne Wissensbasis |
| **Helpdesk** | SLA-Engine, Ticket-Merge, Queues, Canned Responses |
| **Berichte** | Custom Reports mit PDF/CSV/XLSX-Export, Scheduled Runs |
| **Formulare** | Schemas + Submissions + HMAC-signierte Webhooks |
| **Video & Voice** | LiveKit (self-hostable), Recording mit Pre-Consent-Flow |
| **Dialer** | Outbound-Campaigns, Agent-Queue, Outcome-Tracking |

### Branchenmodule (Cosmi-Branchenpaket)

| Modul | Beschreibung |
|---|---|
| **Fuhrpark** | Fahrzeuge, TÜV-Reminder-Cron, Service-Historie, Schadenserfassung |
| **Inventar** | Lagerverwaltung, Stock-Warnings |
| **Einkauf** | Bestellungen, Wareneingang, Lieferanten |
| **Produktion** | Produktionsaufträge, Stücklisten |
| **Rapporte** | GPS-getaggte Tagesrapporte mit Approval-Workflow |
| **Schichten** | Schichtplanung mit ArbZG-§5-Pre-Check (DST-aware) |
| **Vermietung** | Mietobjekte mit GIST-tstzrange-Doppelbuchungs-Schutz |
| **Verträge** | Vertragsverwaltung, Laufzeiten, Kündigungsfristen |

### Kommunikation & Admin

Unified Inbox · Notifications · Teams-Bot · Slack-Bot · Automation-Engine (Trigger/Condition/Action) · Sicherheit (2FA, Audit-Log, GDPR-Export/Erasure, Vault) · RBAC (admin/manager/member, resource:action-Permissions) · Feature-Flag-Registry (16 Flags, Live)

Vollständige Modul-Scope-Matrix mit Tabellen/RPCs/Hooks: [`docs/MODULES_SCOPE_MATRIX.md`](docs/MODULES_SCOPE_MATRIX.md)

---

## Pricing

**Modul-x-User-Modell** (siehe [`docs/PRICING.md`](docs/PRICING.md)):

- **Cosmi** (SaaS) — Kernmodule + optionale Module pro User/Monat
- **Cosmi Branchenpaket** — Cosmi + alle Branchenmodule
- **Orbit** (Self-Hosted) — Pod (single-tenant) / Station (multi-tenant) / Command (Enterprise)

Onsite-Prozessanalyse + Maßanfertigung als einmalige Onboarding-Pauschale.

---

## Architektur

```
                          ┌─────────────────┐
                          │  Desktop / PWA  │
                          │ Electron + React│
                          └────────┬────────┘
                                   │ HTTPS (Caddy + Let's Encrypt)
                          ┌────────▼────────┐
                          │     Gateway     │
                          │  (HTTP, chi/v5) │
                          └────────┬────────┘
                                   │ gRPC (mTLS optional)
   ┌──────────────────────────────┼──────────────────────────────┐
   │                               │                                │
   │  Kern-Services (10):           │   Branchen-Services (8):       │
   │  auth · crm · chat · work       │   inventar · einkauf · produktion │
   │  email · document · biz         │   vertraege · rapporte · schichten│
   │  notification · automation      │   fuhrpark · vermietung           │
   │  plugin · dialer                │                                │
   │  wiki · helpdesk · berichte     │   Realtime / Files:             │
   │  formulare                      │   livekit · livekit-egress      │
   │                                 │   minio · onlyoffice            │
   └────────────────┬────────────────┴──────────────┬─────────────┘
                    │                                │
              ┌─────▼─────┐                    ┌────▼─────┐
              │ PostgreSQL │                    │ Redis 7.4 │
              │  (pgvector│                    │  (Cache,  │
              │   + 115   │                    │   Rate-   │
              │ Migrations)│                    │   Limit)  │
              └────────────┘                    └───────────┘
```

**25 Application-Services + Caddy + Postgres + Redis + MinIO + LiveKit (×2) + Monitoring (Prometheus/Grafana/Alertmanager).**

### Architektur-Prinzipien

- **Thick Services, Thin Handlers** — Business-Logik im Service-Layer, Handler nur Parse/Call/Respond
- **API-First** — OpenAPI-Spec (`backend/api/openapi.yaml`, ~14k Zeilen) vor Implementation
- **Migrations-Only** — Schema ausschließlich via golang-migrate (`make migrate-create name=xxx`), nie manuelles SQL
- **Structured Logging** — `slog` durchgängig, kein `fmt.Println`
- **Idempotente Operationen** — `Idempotency-Key`-Header auf POST/PUT/PATCH/DELETE (HardMode in Dev, WarnMode in Prod bis Pilot-1)
- **Tenant-Isolation Option-B** — `tenant_id UUID NOT NULL` auf ~123 Tabellen, JWT-Claim `tid` durchgängig per `middleware.GetTenantID(ctx)`
- **Graceful Degradation** — Services können unabhängig ausfallen (CRM ohne Chat etc.)
- **Security-First** — JWT (15min) + Refresh-Rotation, RBAC, IP-Filter mit Fail-Close, CORS-Allowlist (keine Wildcards), CSP, HSTS, Vault-Service für OAuth/API-Keys, DOMPurify-HTML-Sanitization, Pre-Recording-Consent

ADRs: [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md). Lessons aus Vorgängerprojekt: [`docs/LEARNINGS.md`](docs/LEARNINGS.md).

---

## Tech-Stack

| Komponente | Technologie |
|---|---|
| Backend | Go 1.25 — 25 Microservices, gRPC + Protocol Buffers |
| Routing | chi/v5 (HTTP), gRPC mit Interceptor-Stack |
| Datenbank | PostgreSQL 16 (`pgvector/pgvector:pg16`), pgx/v5 Driver |
| Cache | Redis 7.4 (Rate-Limit, Idempotency, Agent-Status) |
| Files | MinIO (S3-kompatibel) |
| Video | LiveKit + Egress (self-hostable, coturn TURN-Server in Falkenstein) |
| Documents | OnlyOffice DocumentServer (WOPI-Protokoll) |
| Desktop | Electron + React 19 + TypeScript 5.7 + Vite 5 |
| Mobile | PWA auf Desktop-Basis (kein React Native) |
| UI | Radix UI + Tailwind CSS 4 + Lucide Icons + Magic UI |
| State | Zustand 5 + TanStack Query 5 (Persistence + Offline-Queue) |
| i18n | i18next v26 + i18next-icu (de/en/fr/it) |
| CI/CD | GitHub Actions (lint/test/build/E2E/smoke/security-scans), automated CD via `cd.yml` |
| Provisioning | Ansible (Pilot-Onboarding, 4 Roles + 50 Tasks, ansible-lint production-profile) |
| Hosting | Hetzner Cloud (EU) — App-Server CPX42 Nürnberg, TURN-Server CAX11 Falkenstein |

---

## Repository-Struktur

```
KMU Hub/
├── backend/                           # Go Microservices
│   ├── api/openapi.yaml               # OpenAPI-Spec (~14k Zeilen)
│   ├── cmd/                           # 25 Service-Binaries
│   ├── internal/                      # Service-Implementierungen
│   ├── migrations/                    # PostgreSQL-Migrationen (115 Paare)
│   ├── proto/                         # Protocol-Buffer-Definitionen
│   └── Makefile                       # Build, Test, Lint, Migrate, Proto
├── desktop/                           # Electron-App
│   └── src/renderer/src/
│       ├── api/                       # API-Clients + TanStack-Query-Hooks
│       ├── modules/                   # Feature-Module
│       ├── stores/                    # Zustand State Stores
│       └── themes/                    # 5-Layer Desk-Theme-System
├── deploy/
│   ├── docker/                        # docker-compose.yml + .prod.yml + Caddyfile
│   ├── ansible/                       # Pilot-Provisioning (S3.1)
│   ├── turn/                          # coturn-Setup
│   ├── hetzner/                       # Hetzner-Firewall + coturn-TLS
│   ├── k8s/                           # Kubernetes-Manifeste (sekundär)
│   └── scripts/                       # deploy.sh, rollback.sh, healthcheck.sh, smoke.sh, backup.sh
├── docs/
│   ├── ROADMAP.md                     # 6-Sprint-Plan bis Launch (Single Source of Truth)
│   ├── ARCHITECTURE.md                # ADRs
│   ├── LEARNINGS.md                   # Lessons aus Vorgängerprojekt
│   ├── PRICING.md                     # Pricing-Modell
│   └── MODULES_SCOPE_MATRIX.md        # 14 Module × Tabellen/RPCs/Flags
├── .knowledge/                        # Obsidian Knowledge Base (Architektur, Datenbank, ...)
├── .github/workflows/                 # ci.yml, ci-desktop.yml, cd.yml, security-review.yml
├── CLAUDE.md                          # Entwicklungsrichtlinien
└── README.md
```

---

## Quickstart

### Voraussetzungen

- **Go** ≥ 1.25
- **Node.js** ≥ 20
- **Docker & Docker Compose** (für lokale Infrastruktur)
- **PostgreSQL** ≥ 16 + **Redis** ≥ 7 (oder per Docker Compose, siehe unten)
- **protoc** (Protocol-Buffer-Compiler)
- **Make**

### Mit Docker (empfohlen)

```bash
# Env-File anlegen
cp deploy/docker/PRODUCTION_TEMPLATE deploy/docker/.env
# .env editieren — alle Secrets setzen

# Infrastruktur + alle Services starten
docker compose -f deploy/docker/docker-compose.yml up -d

# Status prüfen
docker compose -f deploy/docker/docker-compose.yml ps

# Gateway-Logs verfolgen
docker compose -f deploy/docker/docker-compose.yml logs -f gateway
```

Dev-Environment hat `IDEMPOTENCY_MODE=hard` als Default (siehe `.env.example`).

### Manuell

```bash
# Backend bauen + Migrationen + Gateway starten
cd backend
make build
make migrate-up
make run-gateway

# In neuem Terminal: Desktop-App starten
cd desktop
npm install
npm run dev
```

---

## Entwicklung

### Backend

```bash
cd backend

# Build (alle 25 Services)
make build
make build-prod          # mit -tags no_wasm (Production-Build)

# Tests
make test                # Unit-Tests + Race-Detector
make test-coverage       # mit Coverage-Report
make e2e-test            # Integration-Tests (Build-Tag e2e)
make smoke-test          # Smoke-Tests gegen lokales Gateway

# Linting
make lint                # golangci-lint v2.8

# Datenbank
make migrate-up
make migrate-down
make migrate-create name=xxx

# Protocol Buffers
make proto

# Einzel-Service-Run
make run-gateway
make run-auth
# (analog für alle weiteren Services)
```

### Desktop

```bash
cd desktop
npm install
npm run dev              # Electron im Dev-Modus
npm run build            # Production-Build
npm run typecheck        # TypeScript-Check
npm test                 # Vitest
npm run lint             # ESLint
```

### Demo-Mode

Für Frontend-Tests ohne Backend: `RENDERER_VITE_DEMO_MODE=true npm run dev` — aktiviert einen Fetch-Interceptor mit realistischen Mock-Daten (`desktop/src/renderer/src/mocks/`).

---

## Deployment

### Self-Hosted (Orbit-Tier)

Docker-Compose-Setup mit täglichem Backup-Cron, Update via Image-Tags. Alle Daten verbleiben beim Kunden. Komplette Anleitung in `deploy/docker/PRODUCTION_TEMPLATE` und [`.knowledge/deployment.md`](.knowledge/deployment.md).

### SaaS (Cosmi-Tier)

**Instanz-pro-Pilot-Modell** ab M3:

- **App-Server pro Pilot:** Hetzner CPX42 (16 GB RAM) — eigene Datenbank, eigene Domain (`<pilot>.zentria.tech`)
- **Geteilter TURN-Server:** Hetzner CAX11 in Falkenstein (`turn.zentria.tech`)
- **Provisioning:** Ansible-Playbook (`deploy/ansible/site.yml`) — foundation + secrets + app-deploy + turn-Roles, Pilot-Onboarding in <30 Min

```bash
# Pilot-Inventory editieren
vim deploy/ansible/inventory/hosts.yml      # IP einsetzen
ansible-vault create deploy/ansible/inventory/group_vars/pilots/vault.yml

# Real-Apply (nur von Linux-Control-Node — Windows kann nur Syntax-Check via Docker)
cd deploy/ansible
ansible-playbook -i inventory/hosts.yml site.yml --ask-vault-pass
```

### CI/CD

GitHub Actions auf jeden Push:

| Job | Tool | Trigger |
|---|---|---|
| Lint | golangci-lint v2.8 | `backend/` |
| Test | go test + Race-Detector | `backend/` |
| Build | go build | nach Lint+Test |
| E2E | Integration-Tests mit Service-Containers | nach Lint+Test |
| Smoke | Go-Smoke-Tests | nach E2E |
| OpenAPI | Spec-Validierung | parallel |
| Security | gosec + trivy + npm audit | parallel |
| Desktop | typecheck + lint + test + build | `desktop/` |
| CD | SSH-Deploy auf Production | nach erfolgreichem CI auf main |

CD-Pipeline ist concurrency-gated (`production`-Group), nutzt `deploy.sh` mit Auto-Rollback bei Health-Check- oder Smoke-Failure, postet Slack-Notification.

---

## API

REST-API über den Gateway-Service. Vollständige Spezifikation: `backend/api/openapi.yaml` (~14.000 Zeilen, 28 Endpoint-Domains).

API-Client-Types für das Frontend werden generiert:

```bash
cd desktop
npx openapi-typescript ../backend/api/openapi.yaml -o src/renderer/src/api/types.ts
```

Authentifizierung: JWT Access (15min) + Opaque Refresh-Token (7d) mit Rotation und Theft-Detection. Tenant-Claim `tid` ist Pflicht — fehlend/leer → 401.

---

## Themes

5-Layer Desk-Theme-System mit OKLCH-Farbtokens und 77 PNG-Hintergrund-Assets:

- **Cozy** — warme Holztöne
- **Dreamy** — sanfte Pastellfarben
- **Raumstation** — dunkles Sci-Fi
- **Clean** — minimalistisch hell
- **Minimal** — reduziert und fokussiert

Designphilosophie + Motion-Tokens + Anti-Patterns: [`.knowledge/design.md`](.knowledge/design.md)

---

## Dokumentation

| Dokument | Inhalt |
|---|---|
| [`CLAUDE.md`](CLAUDE.md) | Entwicklungsrichtlinien, Architektur-Regeln, Kommandos |
| [`docs/ROADMAP.md`](docs/ROADMAP.md) | 6-Sprint-Plan bis Launch 2026-07-01 (Single Source of Truth) |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Architektur-Entscheidungen (ADRs) |
| [`docs/LEARNINGS.md`](docs/LEARNINGS.md) | Lessons Learned aus Vorgängerprojekt |
| [`docs/PRICING.md`](docs/PRICING.md) | Modul-x-User Preismodell |
| [`docs/MODULES_SCOPE_MATRIX.md`](docs/MODULES_SCOPE_MATRIX.md) | 14 Module × Tabellen/RPCs/Hooks/Flags |
| [`backend/api/openapi.yaml`](backend/api/openapi.yaml) | REST-API-Spezifikation |
| [`.knowledge/`](.knowledge/) | Obsidian Knowledge Base (Architektur, Datenbank, Security, Deployment, …) |

---

## Lizenz

Proprietär. Alle Rechte vorbehalten. Zentria UG (i.G.).
