---
tags: [deployment, docker, ci-cd]
updated: 2026-04-19
---
# Deployment & Infrastruktur

## Docker Compose (Lokal + Self-Hosted)
Datei: `deploy/docker/docker-compose.yml`

### Infrastruktur-Container
| Service | Image | Port | Zweck |
|---------|-------|------|-------|
| postgres | postgres:16-alpine | 5432 | Hauptdatenbank |
| redis | redis:7-alpine | 6379 | Cache + Rate Limiting |
| minio | minio:latest | 9000/9001 | S3-kompatibler File-Storage |
| onlyoffice | onlyoffice/documentserver | 8088/8443 | Document-Editing |
| livekit | livekit-server | 7880/7881/7882 | Video/Audio (WebRTC) |
| livekit-egress | livekit-egress | — | Recording-Koordination |

### Backend-Services
| Service | Port | Abhaengigkeiten |
|---------|------|-----------------|
| auth | 50051 | postgres, redis, migrate |
| crm | 50052 | postgres, redis |
| chat | 50053 | postgres, redis, minio |
| notification | 50054 | postgres, redis |
| work | 50055 | postgres, redis, livekit |
| email | 50056 | postgres, redis |
| document | 50057 | postgres, redis, minio, onlyoffice |
| biz | 50058 | postgres, redis |
| automation | 50059 | postgres, redis |
| plugin | 50060 | postgres, redis |
| dialer | 50061 | postgres, redis |
| wiki | 50062 | postgres, migrate |
| berichte | 50063 | postgres, migrate (Scheduler als Goroutine im Binary) |
| helpdesk | 50065 | postgres, migrate |
| **gateway** | **8080** | **alle Services** (`depends_on: service_healthy`) |

### Health Checks
- Alle Services: `wget --spider http://localhost:{port}/health`
- Interval: 5-10s, Timeout: 5s, Retries: 10-15
- Restart: `unless-stopped`
- Gateway `/health` zeigt: status, checks, registered_services, **version**, **commit**, **build_time**

## CI/CD
### CI Pipeline (`.github/workflows/ci.yml`)
- **Trigger:** Push auf main/develop, PRs (nur bei backend/ Änderungen)
- **Go Version:** 1.25.6
- **Jobs (parallel):**
  1. **Lint** — golangci-lint v2.8
  2. **Test** — `go test ./... -race`, Coverage-Report, 30-Tage-Artifact
  3. **Build** — `make build` (abhaengig von Lint+Test)
  4. **E2E** — Integration Tests (abhaengig von Lint+Test)
  5. **Smoke** — Go Smoke Tests (abhaengig von E2E)
  6. **OpenAPI Validate** — Spec-Validierung
- Service-Container: postgres:16-alpine + redis:7-alpine

### Desktop CI Pipeline (`.github/workflows/ci-desktop.yml`)
- **Trigger:** Push auf main/develop, PRs (nur bei desktop/ Aenderungen)
- **Node Version:** 20
- **Jobs:** Lint → Typecheck → Test → Build

### Weitere Workflows
- **`claude-pr.yml`** — Automatisches Claude Code PR-Review (Architektur-Compliance, Security)
- **`security-review.yml`** — Security-fokussiertes Code-Review bei PRs

### CD Pipeline (`.github/workflows/cd.yml`)
- **Trigger:** `workflow_dispatch` (manuell, mit optional `skip_backup`)
- **Environment:** `production` (GitHub Environment Protection)
- SSH auf Hetzner-Server, fuehrt `deploy.sh` aus
- Post-Deploy: Remote Health Check via curl gegen `https://app.zentria.tech/health`
- **Secret:** `HETZNER_SSH_KEY`

## Deploy Scripts (`deploy/scripts/`)

### deploy.sh — Automatisiertes Deployment
Flow: `lock → snapshot → backup → pull → build → migrate → rolling restart → health check → smoke test → log → unlock`
- **Deployment Lock:** PID-File (`/opt/kmuhub/.deploy.lock`), verhindert parallele Deploys
- **Pre-Deploy Snapshot:** `PREV_SHA` + Migrations-Stand
- **Build-Args:** `--build-arg BUILD_VERSION/BUILD_COMMIT/BUILD_TIME` für ldflags
- **Auto-Rollback:** Bei Health-Check- oder Smoke-Failure: checkout PREV_SHA, migrate down, rebuild, restart
- **Deploy-History:** TSV-Log (`/opt/kmuhub/deploy-history.log`): timestamp, prev_sha, new_sha, status, duration
- **No-Change Detection:** Skipped wenn SHA identisch
- Flags: `--skip-backup`, `--service=<name>`

### rollback.sh — Manueller Rollback
- `./rollback.sh` — Rollback zum vorherigen Deploy (aus deploy-history.log)
- `./rollback.sh --to <sha>` — Rollback zu spezifischem Commit
- `./rollback.sh --list` — Letzte 10 Deployments tabellarisch
- Steps: Lock → Backup → Checkout → Rebuild → Restart → Health Check → Log

### smoke.sh — Curl/jq Smoke Tests (22 Tests, 7 Kategorien)
Laeuft ohne Go-Toolchain auf jedem Server, <30 Sekunden.

| Kategorie | Tests |
|-----------|-------|
| Infrastruktur (5) | Gateway health, 10 Services, HTTPS-Cert, Response <2s, Version-Info |
| Auth Flow (3) | Register, Login + JWT, /auth/me |
| CRM CRUD (3) | POST, GET, DELETE /contacts |
| Security (3) | Unauth 401, CORS-Headers, HSTS |
| Performance (3) | /health <500ms, /auth/login <2s, /contacts <1s |
| Cross-Service (2) | Chat-Channel, Dashboard |
| Berichte (3) | GET /berichte/definitions, POST /run, POST /export?format=pdf (MIME-Check) — gated durch `modules.berichte`, 404 akzeptiert wenn Flag OFF |

Flags: `--base-url URL`, `--verbose`, `--expect-version SHA`
Cleanup: Smoke-User wird am Ende per DELETE entfernt.

### healthcheck.sh — Docker Service Health
- Prüft alle 10 Services + Gateway + Postgres + Redis + Caddy
- Exit 0 = healthy, Exit 2 = failures

### backup.sh / restore.sh — Datenbank-Backup
- Backup-Cron taeglich 02:00

## Deployment-Reihenfolge (KRITISCH)
1. Lock erwerben (flock)
2. Snapshot (PREV_SHA)
3. Backup erstellen
4. Code pullen
5. Images bauen (mit Build-Version)
6. Migrations ausführen
7. Rolling Restart (infra → services → gateway)
8. Health Check — bei Fehler: Auto-Rollback
9. Smoke Tests — bei Fehler: Auto-Rollback
10. Erfolg loggen + Lock freigeben

## PostgreSQL Tuning (docker-compose.prod.yml, 2026-04-08)
Fuer Hetzner CPX42 (16GB RAM):
```
shared_buffers=4GB, effective_cache_size=12GB, work_mem=64MB,
maintenance_work_mem=512MB, max_connections=150
```
Konfiguriert als `command:` Args im postgres Service.

## pprof Profiling
- Aktivierung: `ENABLE_PPROF=true` Environment-Variable
- Endpoint: `/debug/pprof` (gemountet auf `http.DefaultServeMux`)
- Nur fuer Development/Staging — NICHT in Production

## Produktion (Hetzner)
- **Server:** CPX42 (8 vCPU, 16GB RAM), Ubuntu 24.04, Nuernberg
- **IP:** 178.104.38.195, **Domain:** app.zentria.tech
- **SSH:** `ssh -i ~/.ssh/hetzner_kmuhub deploy@178.104.38.195`
- **App-Pfad:** `/opt/kmuhub/`, Compose aus `deploy/docker/`
- **Git Pull:** `sudo GIT_SSH_COMMAND='ssh -i /home/deploy/.ssh/github_deploy' git pull origin main`
- **HTTPS:** Caddy + Let's Encrypt, HSTS, HTTP/2
- **Firewall:** Hetzner Cloud Firewall `kmuhub-fw` (10 Regeln, IPv4+IPv6)
- **Monitoring:** Prometheus + Grafana (localhost-only, SSH-Tunnel)

## Gateway Build-Version (ldflags)
```dockerfile
# Dockerfile.gateway
ARG BUILD_VERSION=dev
ARG BUILD_COMMIT=unknown
ARG BUILD_TIME=unknown
RUN go build -ldflags "-X .../gateway.BuildVersion=$BUILD_VERSION -X .../gateway.BuildCommit=$BUILD_COMMIT -X .../gateway.BuildTime=$BUILD_TIME" ...
```
Package-Level Vars in `backend/internal/gateway/route_health.go`:
`BuildVersion`, `BuildCommit`, `BuildTime`

## Build-Tag `no_wasm` (Prod-Builds, 2026-04-18)
- `make build-prod` ruft `go build -tags no_wasm ./...` — das WASM-Plugin-Runtime-File-Set wird durch den Stub `runtime_disabled.go` ersetzt
- Production Container muessen mit diesem Target gebaut werden, Default-`make build` enthaelt weiterhin die Runtime (fuer lokale Tests)
- Laufzeit-Schutz zusaetzlich via Feature-Flag `plugins.wasm=false` (Env `COSMI_WASM_PLUGINS_ENABLED`)
- Details: [[integrationen]] und [[architektur]]

## OnlyOffice Prod-JWT (2026-04-18)
- `deploy/docker/docker-compose.prod.yml` setzt `JWT_ENABLED: "true"` explizit (vorher: Dev-Default geerbt)
- Secret muss in `/opt/kmuhub/.env.production` als `ONLYOFFICE_JWT_SECRET` + gleicher Wert im Document-Service gepflegt sein
- `smoke.sh` enthaelt seit Sprint 0 einen JWT-Check gegen den OnlyOffice-Container

## Kubernetes (Sekundaer)
- Manifeste: `deploy/k8s/` (base/, overlays/, namespace.yaml)
- Status: Konfiguration vorhanden; primaeres Deployment weiterhin Docker Compose
- Kustomize-Struktur: base + environment overlays

## OnlyOffice vs. Collabora
- **Aktuell:** OnlyOffice DocumentServer (Port 8088, Docker) — aktiv
- **Geplant:** Collabora als Ersatz (MPL 2.0 sicherer als AGPL) — noch nicht umgesetzt

## Self-Hosted (Kunden)
- Docker Compose Setup
- Automatische Backups via Cron
- Update via Docker Image Tags

## Verwandte Notes
- [[architektur]] — Service-Architektur
- [[security]] — Infrastruktur-Security
- [[testing]] — Smoke Tests (Go + Bash)
- [[integrationen]] — Docker-Container für LiveKit, OnlyOffice
