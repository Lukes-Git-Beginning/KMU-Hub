# KMU Hub

All-in-One CRM fuer DACH-KMUs mit EU-Datensouveraenitaet.

---

## Vision

Ein CRM-System, das durch eine einwoechige Onsite-Prozessanalyse massgeschneidert wird. Statt starrer Standardsoftware passt sich KMU Hub an bestehende Workflows an — durch konfigurationsbasierte Anpassung und ein WASM-Plugin-System fuer komplexe Erweiterungen.

**Zielgruppe:** Branchenunabhaengige KMUs im DACH-Raum (5–200 Mitarbeiter)

**Kerneigenschaften:**

- EU-Datensouveraenitaet (Hosting ausschliesslich in der EU)
- SaaS- und Self-Hosted-Optionen
- Deutschland-First: EUR, MWSt-konform, de-DE Locale
- DSGVO-konform mit Audit-Logging und Datenschutz-Vault
- 3-Tier Preismodell: Starter, Business, Enterprise

---

## Tech-Stack

| Komponente | Technologie |
|------------|-------------|
| Backend | Go — Microservices + API Gateway |
| Desktop | Electron + React 19 + TypeScript |
| Mobile | React Native (geplant) |
| Datenbank | PostgreSQL 16 |
| Cache | Redis 7 |
| Dateispeicher | MinIO (S3-kompatibel) |
| Video/Voice | LiveKit (self-hostable) |
| Kollaboration | Collabora (WOPI-Protokoll) |
| Inter-Service | gRPC mit Protocol Buffers |
| CI/CD | GitHub Actions |
| Hosting | EU-only (Hetzner/OVH) |

### Frontend-Stack

- **UI:** Radix UI + Tailwind CSS 4.0 + Lucide Icons
- **State:** Zustand 5.0 + TanStack Query 5.0 (mit Persistence)
- **Routing:** React Router 7.0 (Hash-basiert fuer Electron)
- **Build:** Electron Vite 5.0 + TypeScript 5.7

### Backend-Stack

- **Routing:** chi/v5
- **Datenbank:** pgx/v5 (PostgreSQL Driver)
- **E-Mail:** go-imap/v2 + go-smtp + go-message
- **Kalender:** go-webdav + go-ical (CalDAV)
- **Dokumente:** docconv/v2 (Konvertierung)
- **Automation:** expr-lang (Regelauswertung)
- **Integrationen:** Teams Bot Framework + Slack API

---

## Architektur

```
                        ┌─────────────┐
                        │   Desktop   │
                        │  (Electron) │
                        └──────┬──────┘
                               │ HTTPS
                        ┌──────▼──────┐
                        │   Gateway   │
                        │   (HTTP)    │
                        └──────┬──────┘
                               │ gRPC
          ┌────────────────────┼────────────────────┐
          │         │          │          │          │
     ┌────▼───┐ ┌──▼───┐ ┌───▼──┐ ┌────▼───┐ ┌───▼────┐
     │  Auth  │ │ CRM  │ │ Chat │ │  Work  │ │ Email  │
     └────────┘ └──────┘ └──────┘ └────────┘ └────────┘
          │         │          │          │          │
     ┌────▼───┐ ┌──▼────┐ ┌──▼───┐ ┌───▼────┐ ┌───▼──────┐
     │Notif.  │ │  Biz  │ │ Doc  │ │Automat.│ │  CalDAV  │
     └────────┘ └───────┘ └──────┘ └────────┘ └──────────┘
                               │
                        ┌──────▼──────┐
                        │ PostgreSQL  │
                        │ Redis/MinIO │
                        └─────────────┘
```

**Prinzipien:**

- **Thick Services, Thin Handlers** — Business-Logik in Service-Layer, Handler nur fuer I/O
- **API-First** — OpenAPI-Spec vor Implementation
- **Graceful Degradation** — Services fallen unabhaengig aus
- **Structured Logging** — slog fuer alle Services
- **Idempotente Operationen** — Alle API-Calls sicher wiederholbar
- **Migrations-Only** — Datenbankschema ausschliesslich via golang-migrate

---

## Module & Features

### Kernmodule

| Modul | Beschreibung |
|-------|-------------|
| **CRM** | Kontakte, Deals, Opportunities, Pipeline-Management, Custom Fields, Tags |
| **Chat** | Interne Echtzeit-Kommunikation mit Dateiaustausch |
| **E-Mail** | IMAP/SMTP-Integration mit CRM-Verknuepfung |
| **Kalender** | Terminplanung mit CalDAV-Synchronisation |
| **Aufgaben & Projekte** | Kanban-Boards, Gantt-Charts, Zeiterfassung |
| **Dokumente** | Dateiverwaltung mit WOPI-Kollaboration (Collabora) und Volltextsuche |
| **Finanzen** | Rechnungen, Angebote, Mahnwesen, DATEV-Export |
| **Personal (HR)** | Mitarbeiterverwaltung, Abwesenheiten, Organigramm |
| **Dashboard** | Konfigurierbares Dashboard mit Widgets |

### Kommunikation & Zusammenarbeit

| Modul | Beschreibung |
|-------|-------------|
| **Video & Voice** | Videokonferenzen und Sprachanrufe via LiveKit |
| **Unified Inbox** | Kanaluebergreifende Nachrichtenaggregation (E-Mail, Chat, Benachrichtigungen) |
| **Benachrichtigungen** | Push-Notifications mit Praeferenzsteuerung |
| **Teams-Integration** | Bidirektionale Benachrichtigungen via Adaptive Cards |
| **Slack-Integration** | Bidirektionale Benachrichtigungen via Block Kit |

### System & Administration

| Modul | Beschreibung |
|-------|-------------|
| **Automation** | Regelbasierte Workflows mit Trigger/Condition/Action-Engine |
| **Sicherheit** | Zwei-Faktor-Authentifizierung, Audit-Logging, DSGVO-Compliance |
| **Verwaltung** | Benutzerverwaltung, Rollen & Berechtigungen, Einladungssystem |
| **Einstellungen** | Organisations-, Benutzer- und Modulkonfiguration |

### Branchenmodule (Plugin-Kandidaten)

Fuhrpark, Produktion, Rapporte, Schichten, Inventar, Einkauf, Helpdesk, Vermietung, Vertraege, Formulare

---

## Projektstruktur

```
KMU Hub/
├── backend/
│   ├── api/                    # OpenAPI-Spezifikation
│   ├── cmd/                    # Service-Binaries (10 Services + Gateway)
│   ├── internal/               # Interne Packages (auth, crm, chat, ...)
│   ├── migrations/             # PostgreSQL-Migrationen (53 Paare)
│   ├── proto/                  # Protocol Buffer Definitionen (14 Services)
│   └── Makefile                # Build, Test, Lint, Migrate, Proto
├── desktop/
│   ├── src/
│   │   ├── main/               # Electron Main Process
│   │   ├── preload/            # Preload Scripts
│   │   └── renderer/
│   │       └── src/
│   │           ├── api/        # API Client, Types, TanStack Query Hooks
│   │           ├── components/ # Shared UI-Komponenten
│   │           ├── modules/    # Feature-Module (34 Module)
│   │           ├── stores/     # Zustand State Stores
│   │           └── themes/     # 5-Layer Desk-Theme-System
│   └── package.json
├── deploy/
│   └── docker/
│       └── docker-compose.yml  # Lokale Entwicklung + Self-Hosted
├── docs/
│   ├── ARCHITECTURE.md         # Architektur-Entscheidungen
│   ├── LEARNINGS.md            # Lessons Learned
│   └── PRICING.md              # Preismodell
├── .github/
│   └── workflows/
│       └── ci.yml              # CI-Pipeline
├── .planning/                  # GSD Planungsdateien
├── CLAUDE.md                   # Entwicklungsrichtlinien
└── README.md
```

---

## Voraussetzungen

- **Go** >= 1.25
- **Node.js** >= 20
- **PostgreSQL** >= 16
- **Redis** >= 7
- **protoc** (Protocol Buffer Compiler)
- **Docker & Docker Compose** (optional, fuer lokale Infrastruktur)
- **Make** (fuer Backend-Build-Kommandos)

---

## Quickstart

### Mit Docker (empfohlen)

```bash
# Infrastruktur + alle Services starten
cd deploy/docker
docker-compose up -d

# Status pruefen
docker-compose ps

# Logs verfolgen
docker-compose logs -f gateway
```

### Manuell

```bash
# Backend bauen
cd backend
make build

# Migrationen ausfuehren
make migrate-up

# Gateway starten (HTTP-Einstiegspunkt)
make run-gateway

# In einem neuen Terminal: Desktop-App starten
cd desktop
npm install
npm run dev
```

### Umgebungsvariablen

Erstelle eine `.env`-Datei basierend auf `.env.example` (falls vorhanden) oder konfiguriere folgende Bereiche:

- **Datenbank:** PostgreSQL-Verbindung
- **Cache:** Redis-Verbindung
- **Dateispeicher:** MinIO/S3-Konfiguration
- **Video:** LiveKit-Verbindung
- **E-Mail:** IMAP/SMTP-Server
- **Integrationen:** Teams und Slack Credentials (optional)

> **Hinweis:** `.env`-Dateien werden NICHT committed. Secrets ausschliesslich ueber Umgebungsvariablen konfigurieren.

---

## Entwicklung

### Backend

```bash
cd backend

# Alle Services bauen
make build

# Einzelne Services starten
make run-gateway
make run-auth
make run-crm
make run-chat
make run-notification
make run-work
make run-automation

# Tests
make test                      # Unit Tests
make test-coverage             # Mit Coverage-Report

# Linting
make lint                      # golangci-lint

# Datenbank
make migrate-up                # Migrationen ausfuehren
make migrate-down              # Letzte Migration zurueckrollen
make migrate-create name=xxx   # Neue Migration erstellen

# Protocol Buffers
make proto                     # Proto-Dateien kompilieren
```

### Desktop

```bash
cd desktop

npm install                    # Dependencies installieren
npm run dev                    # Electron im Dev-Modus
npm run build                  # Production Build
npm run test                   # Tests
npm run lint                   # ESLint
```

### Docker

```bash
cd deploy/docker

docker-compose up -d           # Alle Services starten
docker-compose down            # Stoppen
docker-compose logs -f <name>  # Service-Logs
```

---

## CI/CD

Die CI-Pipeline (GitHub Actions) prueft bei jedem Push:

1. **Lint** — golangci-lint mit projektspezifischer Konfiguration
2. **Test** — Unit Tests mit Race Detector und Coverage (PostgreSQL + Redis Containers)
3. **Build** — Kompilierung aller 10 Services
4. **E2E** — End-to-End Integrationstests
5. **OpenAPI** — Validierung der API-Spezifikation

Coverage-Artefakte werden 30 Tage aufbewahrt.

---

## API

Die REST-API wird ueber den Gateway-Service bereitgestellt. Die vollstaendige Spezifikation liegt in `backend/api/openapi.yaml` (~14.000 Zeilen).

API-Client-Types fuer das Frontend werden automatisch generiert:

```bash
cd desktop
npx openapi-typescript ../backend/api/openapi.yaml -o src/renderer/src/api/types.ts
```

---

## Themes

KMU Hub verfuegt ueber ein 5-Layer Desk-Theme-System mit 5 Themes:

- **Cozy** — Warme Holztoene
- **Dreamy** — Sanfte Pastellfarben
- **Raumstation** — Dunkles Sci-Fi
- **Clean** — Minimalistisch hell
- **Minimal** — Reduziert und fokussiert

Themes verwenden OKLCH-Farbtokens und 77 PNG-Assets fuer den Schreibtisch-Hintergrund.

---

## Deployment

### Self-Hosted (Kunden)

- Docker Compose Setup fuer einfaches Deployment
- Automatische Backups via Cron
- Update-Mechanismus ueber Docker Image Tags
- Alle Daten verbleiben beim Kunden

### SaaS

- Kubernetes auf Hetzner Cloud (EU)
- Blue-Green Deployment
- Automatische Skalierung
- EU-Datensouveraenitaet garantiert

---

## Dokumentation

| Dokument | Beschreibung |
|----------|-------------|
| [`CLAUDE.md`](CLAUDE.md) | Entwicklungsrichtlinien, Architekturregeln, Kommandos |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Architektur-Entscheidungen (ADRs) |
| [`docs/LEARNINGS.md`](docs/LEARNINGS.md) | Lessons Learned aus Vorgaengerprojekt |
| [`docs/PRICING.md`](docs/PRICING.md) | Preismodell fuer DACH-Markt |
| [`backend/api/openapi.yaml`](backend/api/openapi.yaml) | REST-API Spezifikation |

---

## Lizenz

Proprietaer. Alle Rechte vorbehalten.
