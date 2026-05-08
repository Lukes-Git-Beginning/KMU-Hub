---
tags: [deployment, docker, ci-cd]
updated: 2026-05-08
---
# Deployment & Infrastruktur

> **Aktueller Prod-Stand (2026-04-20):** `app.zentria.tech` auf `980eba3`, Migration-Head 81, alle 15 Business-Services healthy. Erster Full-Redeploy seit 2026-03-08 — Details in MEMORY `project_server_redeploy_20260419.md`. **Wichtig:** Auf dem Server sind `deploy/docker/livekit.yaml` und `deploy/docker/docker-compose.yml` via `git update-index --skip-worktree` aus Git-Sight genommen, weil sie lokale Patches + Secrets enthalten, die noch nicht in `main` committed sind. Nächste PRs, die diese Files ändern, müssen vor dem Pull `--no-skip-worktree` + manuellen Re-Patch durchlaufen.
>
> **Welle 4B (2026-05-07):** `deploy/docker/docker-compose.yml` und `backend/.env.example` setzen `IDEMPOTENCY_MODE=hard` im Gateway-Environment fuer Dev. Production bleibt unset → WarnMode default. Prod-Cutover auf HardMode ist Sprint-3-Aktion nach Pilot-1.

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
- **Aufruf auf Prod:** `sudo env GIT_SSH_COMMAND='ssh -i /home/deploy/.ssh/github_deploy' /opt/kmuhub/deploy/scripts/deploy.sh` (GIT_SSH_COMMAND noetig, weil root keinen eigenen GitHub-Key hat)

**Gefixt in `980eba3` (2026-04-19):**
- `COMPOSE_FILES_DIR` und `ENV_FILE` getrennt vom Git-`COMPOSE_DIR` (vorher: Script suchte Compose-Files unter `/opt/kmuhub/`, die aber in `/opt/kmuhub/deploy/docker/` liegen)
- `--env-file /opt/kmuhub/.env.production` wird jetzt an jeden `docker compose`-Call uebergeben (vorher: Prod-Env wurde ignoriert)
- Rolling-Restart-Liste um `dialer wiki helpdesk berichte formulare` erweitert

**Noch offen (Sprint 2):**
- `livekit`, `livekit-egress` fehlen in der Restart-Liste — nach einem Deploy laeuft LiveKit weiter mit altem Config-Mount. Workaround: manuell `$COMPOSE up -d --force-recreate livekit livekit-egress` nach dem Script.
- Auto-Rollback-Pfad referenziert `$DATABASE_URL` als Shell-Var (nicht aus env-file) — funktioniert nur wenn vor dem Script `source .env.production` gemacht wurde.

### rollback.sh — Manueller Rollback
- `./rollback.sh` — Rollback zum vorherigen Deploy (aus deploy-history.log)
- `./rollback.sh --to <sha>` — Rollback zu spezifischem Commit
- `./rollback.sh --list` — Letzte 10 Deployments tabellarisch
- Steps: Lock → Backup → Checkout → Rebuild → Restart → Health Check → Log
- **Wichtig:** macht KEIN `migrate down`. Nur der interne Auto-Rollback-Pfad in `deploy.sh` berechnet und fuehrt `migrate down N` aus. Bei manuellem Rollback ueber ein Schema-Aenderung hinweg: vorher explizit `$COMPOSE run --rm migrate -path /migrations -database "$DATABASE_URL" down N`.

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
- Backup-Cron taeglich 02:00 (in `deploy`-User-crontab)
- **Known Issue (seit 2026-03-08):** `/var/log/kmuhub-backup.log` existiert nicht → Cron laeuft silent und schreibt keine Backups. Sprint-2-Task: Root-Cause untersuchen (Permission? Working-Dir? Log-Rotation?). Als Sicherheit: manueller `pg_dumpall` vor jedem Deploy.
- **`jq` ist NICHT installiert** auf dem Prod-Server — `smoke.sh` failt mit `ERROR: jq is required but not installed`. Sprint-2-Task: `sudo apt install jq`.

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

## Server-Side Patches via `skip-worktree` (2026-04-19/20, Stand 2026-05-08)

Auf dem Prod-Server waren zwei Dateien lokal gepatched. Stand 2026-05-08 (Sprint 3 Welle 0) sind die meisten Fixes in `main` committed — siehe Aktualisierung unten. Siehe auch MEMORY `project_server_redeploy_20260419.md`.

**`livekit.yaml`** bleibt skip-worktree, weil LiveKit keine ENV-Substitution im yaml unterstuetzt. **Loesung in main**: `deploy/scripts/render-configs.sh` rendert `livekit-secrets.yaml` via `envsubst` aus Template + `.env.production`-Vars; `deploy.sh` ruft das in Step 2.5 vor jedem Build auf. Beim naechsten Server-Deploy wird `livekit.yaml` aus skip-worktree entfernt und das gerenderte File genutzt.

**`docker-compose.yml`** kann von skip-worktree entfernt werden — alle Patches sind in main:
- `DATABASE_URL`/`POSTGRES_PASSWORD` via `${...}` parametrisiert (Sprint 2)
- `RegisterHealth` registriert sowohl GET als auch HEAD (`backend/internal/server/http.go:46-47`), `wget --spider` funktioniert sauber
- `formulare`-Service registriert `/health` (Konsistenz mit allen anderen Services)

**Server-Side-Cleanup-Schritte (naechster Deploy):**
```bash
# 1. skip-worktree entfernen
sudo -u deploy bash -c "cd /opt/kmuhub && git update-index --no-skip-worktree deploy/docker/docker-compose.yml deploy/docker/livekit.yaml"
# 2. lokale livekit.yaml umbenennen (wird ersetzt durch render-configs.sh)
sudo mv /opt/kmuhub/deploy/docker/livekit.yaml /opt/kmuhub/deploy/docker/livekit.yaml.legacy
# 3. Pull + Deploy (deploy.sh ruft render-configs.sh in Step 2.5 auf)
```

## Sprint-2-TODOs fuer Deploy-Hygiene (Stand 2026-05-08)

| # | Item | Status |
|---|---|---|
| 1 | `docker-compose.yml` 18× hardcoded `kmuhub_dev` eliminieren | ✅ **erledigt** in main, `${DATABASE_URL}`/`${POSTGRES_PASSWORD}` ueberall |
| 2 | Backend `/health` HEAD-Support | ✅ **erledigt** — `RegisterHealth` registriert GET+HEAD (`backend/internal/server/http.go:46-47`) |
| 3 | `formulare`-Service `/health` registrieren | ✅ **erledigt** — Service nutzt `/health` wie alle anderen (`backend/cmd/formulare/main.go:86`) |
| 4 | `deploy.sh` `livekit`+`livekit-egress` in Rolling-Restart-Liste | ✅ **erledigt** in main |
| 5 | `livekit.yaml` Template-Renderer | ✅ **erledigt** — `deploy/scripts/render-configs.sh` + `deploy.sh` Step 2.5 |
| 6 | `deploy.sh` Sprint-2-Services in Rolling-Restart (Welle 0 Sprint 3) | ✅ **erledigt 2026-05-08** — inventar/einkauf/produktion/vertraege/rapporte/schichten/fuhrpark/vermietung in beiden Listen (deploy + rollback) |
| 7 | Backup-Cron Root-Cause `/var/log/kmuhub-backup.log` | ⏭ Server-Operations (eigener Block) |
| 8 | `jq` auf Prod-Server installieren | ⏭ Server-Operations (eigener Block) |

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

### App-Server CPX42
- **Server:** CPX42 (8 vCPU, 16GB RAM, x86_64), Ubuntu 24.04, Nuernberg (nbg1)
- **IP:** 178.104.38.195, **Domain:** app.zentria.tech
- **SSH:** `ssh -i ~/.ssh/hetzner_kmuhub deploy@178.104.38.195`
- **App-Pfad:** `/opt/kmuhub/`, `.env.production` direkt dort (NICHT in `deploy/docker/`)
- **Compose:** aus `deploy/docker/` mit `-f docker-compose.yml -f docker-compose.prod.yml`
- **Git Pull:** `sudo GIT_SSH_COMMAND='ssh -i /home/deploy/.ssh/github_deploy' git pull origin main`
- **HTTPS:** Caddy + Let's Encrypt, HSTS, HTTP/2
- **Firewall:** Hetzner Cloud Firewall `kmuhub-fw` (7 Regeln: SSH/80/443/7880/7881/7882-UDP/ICMP, Source Any IPv4+IPv6)
- **Monitoring:** Prometheus + Grafana + Alertmanager (alle localhost-only, SSH-Tunnel)
- **Alertmanager:** `prom/alertmanager:v0.27.0`, Port 9093, Config `deploy/docker/alertmanager.yml` — Slack-Alerts via `${SLACK_WEBHOOK_URL}` (`.env.production`), 3 Rules: ServiceDown (2m), HighErrorRate (5%), DBConnectionsHigh (80% max_connections)

### TURN-Server CAX11 (seit 2026-04-19)
- **Server:** CAX11 (ARM Ampere, 2 vCPU, 4GB RAM, 40GB SSD, 20TB Traffic, ~€3.80/M), Ubuntu 24.04, Falkenstein (fsn1)
- **IP:** 5.75.246.217, **Domain:** turn.zentria.tech (A-Record bei Cloudflare, DNS only; PTR in Hetzner)
- **SSH:** `ssh -i ~/.ssh/hetzner_kmuhub root@5.75.246.217`
- **Firewall:** `kmuhub-turn-fw` (4 Regeln: 22/TCP, 3478/TCP+UDP, 49152-65535/UDP)
- **Service:** coturn 4.6.1 (systemd `coturn.service`), plain TURN/UDP, noch kein TLS
- **Config:** `/etc/turnserver.conf` (`lt-cred-mech` + `use-auth-secret`, `static-auth-secret` shared mit App-Server `.env.production:TURN_SECRET`)
- **Deploy-Artefakte:** `deploy/turn/setup.sh` + `deploy.sh` + `turnserver.conf.template`
- **Status:** Server live, aber LiveKit-Wiring in video-service offen (Sprint 2 S2.R2.1b, siehe `deploy/turn/livekit-integration.md` Option B)

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
