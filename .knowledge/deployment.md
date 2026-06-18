---
tags: [deployment, docker, ci-cd]
updated: 2026-06-18
---
# Deployment & Infrastruktur

> **Prod-Stand (live gemessen 2026-06-18):** `app.zentria.tech`, Production-Migrationskopf **209** (Repo-Kopf **000213** — CD-Lag von 4 Migr. zum Mess-Zeitpunkt), **`COSMI_ENV=production` SCHARF**, Container healthy. Seit 2026-06-06 brachten die FE↔Backend-Wiring-Wellen Migr. 148–213. **Vorgeschichte (2026-06-06 frueh, nach LiveKit/COSMI_ENV-Cluster):** `564f238b`, Migration-Head **131**, alle Container healthy, **`COSMI_ENV=production` SCHARF** (Prod-Secrets-Assertion aktiv), Smoke **24/24**, **Video-Calls erstmals end-to-end funktional** (`/rtc/validate`=200). Alle Compose-Secrets laufen jetzt ueber `${VAR:-dev-default}`-Interpolation aus `.env.production` (vorher: Dev-Secrets in Prod, Incident-Doku `docs/livekit-env-production-followups.md`). Davor (2026-06-05 nach E2E-Modernisierung): `91a3014c`, Head 129, erster automatischer CD-Deploy. **Lokal/main = Prod synchron, jeder gruene Push auf main deployt automatisch.**
>
> **skip-worktree-Status (Stand 2026-05-08 nach Welle-1-Marathon):** keine aktiven Markierungen mehr. `livekit.yaml`-Patch aus alter Era ist obsolet (livekit-secrets.yaml-render-overlay ersetzt das per `render-configs.sh` in `deploy.sh` Step 2.5).
>
> **Welle 4B (2026-05-07):** `deploy/docker/docker-compose.yml` und `backend/.env.example` setzen `IDEMPOTENCY_MODE=hard` im Gateway-Environment fuer Dev. Production bleibt unset → WarnMode default. Prod-Cutover auf HardMode ist post-Pilot-1-Aktion.
>
> **Ansible-Playbook (Sprint 3, 2026-05-08):** `deploy/ansible/` ist live mit 4 Roles (foundation/secrets/app-deploy/turn) und 50 Tasks insgesamt. Inventory `pilots` + `turn` mit Platzhalter-IPs (Pilot-0-IP wird vor Real-Provisioning gesetzt). ansible-lint **production-profile 0 failures**. Verifikation auf Windows via Docker-Wrapper (`willhallonline/ansible:latest` + `MSYS_NO_PATHCONV=1`). Details siehe Abschnitt "Ansible Pilot-Provisioning" unten.

## Compose-Secret-Interpolation (2026-06-05, `68158907`)

**Hardrule:** JEDES Secret in der Basis-Compose steht als `${VAR:-<dev-default>}` — Dev bleibt
zero-config, Production gewinnt automatisch via `--env-file /opt/kmuhub/.env.production`.
Niemals Secret-Literale in `environment:`-Bloecke (Incident: Dev-`JWT_SECRET`/`minioadmin`/
`devkey` liefen monatelang in Prod, weil das Prod-Overlay nur 3 von ~20 Secret-Stellen abdeckte).

- Interpoliert: `JWT_SECRET` (24 Services), `VAULT_MASTER_SECRET`, `WOPI_JWT_SECRET`,
  `MINIO_ROOT_USER/PASSWORD` (inkl. createbucket-Args), `MINIO_ACCESS_KEY/SECRET_KEY`,
  `LIVEKIT_API_KEY/SECRET/WEBHOOK_SECRET`, `LIVEKIT_WS_URL`, `COSMI_ENV` (alle 24 Services)
- **Gerenderte Secret-Configs** via `render-configs.sh` (deploy.sh Step 2.5, alle `.gitignore`d):
  `livekit-secrets.yaml`, **`egress-secrets.yaml`** (neu — Prod-Overlay mountet es ueber
  `/etc/egress.yaml`; das eingecheckte `egress.yaml` bleibt Dev-only), `alertmanager.yml`
- Prod-Overlay deckt seit 2026-06-05 ALLE Services mit logging+resource-limits (vorher fehlten
  dialer + die 9 Welle-2/3-Module komplett)
- Verifikation lokal: `docker compose --env-file <dummy> -f docker-compose.yml [-f docker-compose.prod.yml] config` und auf 0 Dev-Marker greppen
- Gegenstueck im Backend: Requirements-Assertion, siehe [[security]] "Prod-Secrets Startup-Assertion"

## Docker Compose (Lokal + Self-Hosted)
Datei: `deploy/docker/docker-compose.yml`

### Infrastruktur-Container
| Service | Image | Port | Zweck |
|---------|-------|------|-------|
| postgres | pgvector/pgvector:pg16 | 5432 | Hauptdatenbank (Volume `docker_pgdata`) |
| redis | redis:7.4-alpine | 6379 | Cache + Rate Limiting (Pin auf 7.4 wegen RDB-v12-Kompat) |
| minio | minio/minio:RELEASE.2025-05-21T... | 9000/9001 | S3-kompatibler File-Storage. **Seit 2026-06-11 public erreichbar als `s3.zentria.tech`** (Caddy-Route, `1aef2f45`) — Presign-URLs (`/api/v1/files/presign-*`) müssen browser-reachable sein. CORS via Server-Env `MINIO_API_CORS_ALLOW_ORIGIN` statt `mc cors set` (existiert nicht, `1e65662a`) |
| minio/mc | minio/mc:RELEASE.2025-05-21T... | — | Bucket-Init (Tag-Rotation siehe Welle 1) |
| onlyoffice | onlyoffice/documentserver | 8088/8443 | Document-Editing (unhealthy seit 2 Monaten) |
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
- **Trigger:** Push auf main, PRs (nur bei `backend/**`-Änderungen)
- **Go Version:** 1.25.6
- **Jobs (Gate-Kern, seit 2026-06-09 verschlankt):**
  1. **Lint** — golangci-lint v2.8
  2. **Test** — `go test ./... -race`, Coverage-Gate 15%, 30-Tage-Artifact
  3. **E2E** — Integration Tests (abhaengig von Lint+Test). Startet auth/crm/chat/work/document/**dialer**/gateway als Binaries + MinIO via docker-run; Job-Env `RATE_LIMIT_RPS=1000` (CI-only, Prod bleibt 100). **Lesson 2026-06-05:** Service-Liste muss mit `test/e2e/` synchron bleiben — der Dialer fehlte und `dialer_test.go` konnte nie gruen werden.
  4. **OpenAPI Validate** — Spec-Validierung
- **Pipeline-Split 2026-06-09 (`8f6aaa32`):** redundanter `build`-Job entfernt (`go test ./...` kompiliert bereits alle `cmd/*`), `smoke`→`nightly.yml`, Security-Scans (gosec/trivy/npm-audit)→`scans.yml`. Spart per-Push Actions-Minuten (~9 Jobs → 4). Schwere Jobs NICHT zurueck in ci.yml mergen. Siehe Memory `project_ci_pipeline_split_20260609.md`.
- Service-Container: pgvector/pgvector:pg16 + redis:7-alpine
- **Komplett gruen seit 2026-06-05** (rot seit 2026-03-08; Repair + E2E-Modernisierung, Lessons in Memory `feedback_ci_lessons_20260605.md`)
- **Paths-Filter:** CI triggert nur auf `backend/**` + `ci.yml` (NICHT mehr `desktop/**` seit 2026-06-09 — das laeuft separat in ci-desktop.yml). docs-/`.knowledge/`-Commits triggern weder CI noch CD. **Achtung:** auch `deploy/**`-only-Commits triggern kein CI, und CD haengt per `workflow_run` an CI → fuer reine Deploy-Aenderungen `gh workflow run CD --ref main` dispatchen (Lesson 2026-06-05, `ce2a5e5d`)

### Desktop CI Pipeline (`.github/workflows/ci-desktop.yml`)
- **Trigger:** Push auf main, PRs (nur bei desktop/ Aenderungen)
- **Node Version:** 20
- **Jobs:** checks (Lint + Typecheck + Test, ein Runner/`npm ci`) → Build

### Weitere Workflows
- **`scans.yml`** (seit 2026-06-09) — gosec + trivy fs-scan + npm audit. Trigger: woechentlich (Mo 04:00 UTC) + bei `go.mod`/`go.sum`/`package-lock.json`-Aenderung + `workflow_dispatch`. Trivy-Cache-Key wochenbasiert (vorher `github.run_id` = nie ein Hit). Details: [[security]] CI Security-Scans.
- **`nightly.yml`** (seit 2026-06-09) — self-contained Go-Smoke-Suite (baut eigene 6 Services, kein e2e-Artefakt). Trigger: taeglich 03:00 UTC + `workflow_dispatch`.
- **`claude-pr.yml`** — Automatisches Claude Code PR-Review (Architektur-Compliance, Security)
- **`security-review.yml`** — Security-fokussiertes Code-Review bei PRs

### CD Pipeline (`.github/workflows/cd.yml`)
- ✅ **LIVE seit 2026-06-05** — erster automatischer Production-Deploy erfolgreich (Run `27017597354`, 18m04s, Code `91a3014c`, Migrations 118→129, 14/14 Container healthy, kein Rollback). **Jeder gruene Push auf main deployt seither automatisch nach Production.**
- **Trigger:** `workflow_run` (nach erfolgreichem `ci.yml` auf main) + `workflow_dispatch` (manuell, mit optional `skip_backup`)
- **Concurrency-Group:** `production` — parallele Deploys werden abgebrochen (`cancel-in-progress: false`, neuere warten)
- **Environment:** `production` (GitHub Environment Protection)
- SSH auf Hetzner-Server, fuehrt `deploy.sh` aus
- Post-Deploy: Remote Health Check via curl gegen `https://app.zentria.tech/health` + Slack-Notify bei Erfolg/Fehler
- **Secret:** `HETZNER_SSH_KEY`, `ALERT_WEBHOOK_URL` (Discord-Webhook im Slack-Compat-Mode, siehe Alertmanager-Block unten)

## Deploy Scripts (`deploy/scripts/`)

### deploy.sh — Automatisiertes Deployment
Flow: `lock → snapshot → backup → pull → build → migrate → rolling restart → health check → smoke test → log → unlock`
- **Deployment Lock:** PID-File (`/opt/kmuhub/.deploy.lock`), verhindert parallele Deploys
- **Pre-Deploy Snapshot:** `PREV_SHA` + Migrations-Stand
- **Build-Args:** `--build-arg BUILD_VERSION/BUILD_COMMIT/BUILD_TIME` für ldflags
- **Auto-Rollback:** Bei Health-Check- oder Smoke-Failure: checkout PREV_SHA, **rebuild Code, restart Container — aber KEIN `migrate down`**. Code geht zurück, DB-Schema bleibt vorne → Drift moeglich. Bei Schema-aendernden Wellen: Smoke-Failure-Triage VOR re-deploy, ggf. `--skip-smoke` als Notbremse (Welle 1 Lesson 2026-05-10).
- **Deploy-History:** TSV-Log (`/opt/kmuhub/deploy-history.log`): timestamp, prev_sha, new_sha, status, duration
- **No-Change Detection:** Skipped wenn SHA identisch
- Flags: `--skip-backup`, `--skip-smoke` (seit `25af970`, 2026-05-10), `--force`, `--service=<name>`
- **Aufruf auf Prod:** `sudo GIT_SSH_COMMAND='ssh -i /home/deploy/.ssh/github_deploy' bash /opt/kmuhub/deploy/scripts/deploy.sh --force` (GIT_SSH_COMMAND noetig, weil root keinen eigenen GitHub-Key hat — Inline-Env-Form, sudo `env_reset` strippt sonst die Variable). Absoluter Pfad zwingend, deploy@-User landet im `/home/deploy/`.

**Gefixt in `980eba3` (2026-04-19):**
- `COMPOSE_FILES_DIR` und `ENV_FILE` getrennt vom Git-`COMPOSE_DIR` (vorher: Script suchte Compose-Files unter `/opt/kmuhub/`, die aber in `/opt/kmuhub/deploy/docker/` liegen)
- `--env-file /opt/kmuhub/.env.production` wird jetzt an jeden `docker compose`-Call uebergeben (vorher: Prod-Env wurde ignoriert)
- Rolling-Restart-Liste um `dialer wiki helpdesk berichte formulare` erweitert

**Gefixt in Welle-1-Marathon (2026-05-08, 9 Hotfix-Commits):**
- `089c2d4` rollback.sh-Service-Liste auf alle 25 Sprint-2-Services erweitert (vorher 10)
- `f4add92` `SLACK_WEBHOOK_URL=` Slot in PRODUCTION_TEMPLATE (Alertmanager-Webhook-Pfad — 2026-05-09 in Discord-Refactor zu `ALERT_WEBHOOK_URL` umbenannt)
- `53dd5b6` Step 3 Build laeuft jetzt **seriell** (`for svc in app_services; do compose build $svc; done`) — parallel-bake killt 16-GB-Hosts mit OOM
- `c7a9a76` Migration 000114: `CREATE TABLE IF NOT EXISTS tenants` als Bootstrap am Anfang (FK-Resolution-Fix; vorher referenzierten 000114+115 `tenants(id)` ohne dass die Tabelle je angelegt war)
- `3c1ffcd` redis Pin auf 7.4-alpine (vorher 7.2.7 — RDB-v12 von 7.4+ unlesbar)
- `32588ed` minio/mc image-Tag-Rotation (vorheriger Tag aus Docker Hub entfernt)
- `7da7ed8` `healthcheck.sh` `((HEALTHY++))` → `HEALTHY=$((HEALTHY+1))` (set -e brach nach erstem [OK] ab)
- `61b0996` `healthcheck.sh` Compose-Pfad align mit `deploy.sh` (`COMPOSE_FILES_DIR + ENV_FILE`)
- `3abec5f` `healthcheck.sh` `--resolve $CADDY_HEALTHCHECK_HOST:443:127.0.0.1` (vorher `https://localhost`, Caddy-Vhost-Mismatch)

**Bekannte Issues (Sprint 4-Followups):**
- `livekit`/`livekit-egress` Restart bei Deploy → manuell `$COMPOSE up -d --force-recreate livekit livekit-egress` nach Script.
- Auto-Rollback `$DATABASE_URL` als Shell-Var (nicht aus env-file) — vor Script `source .env.production` noetig.
- Auto-Rollback rebuilds redundant wenn `PREV_SHA == NEW_SHA` (followup).
- `ALERT_WEBHOOK_URL` muss noch in `/opt/kmuhub/.env.production` auf dem Server gesetzt werden — Alertmanager laeuft sonst stumm. Architektur seit Discord-Refactor (2026-05-09): `alertmanager.yml.tmpl` wird von `render-configs.sh` mit `envsubst '$ALERT_WEBHOOK_URL'` gerendert, leerer Wert → `slack_api_url: ""` (Alertmanager startet sauber, sendet nur nicht). Discord-Webhook fuer `#cosmi-prod-alerts` im zentria-intel-Server, **`/slack`-Suffix angehaengt** fuer Slack-Format-Kompatibilitaet. Aktivierung: Webhook in `.env.production` setzen + `render-configs.sh` + Alertmanager-Restart (oder voller `deploy.sh`-Run).

### rollback.sh — Manueller Rollback
- `./rollback.sh` — Rollback zum vorherigen Deploy (aus deploy-history.log)
- `./rollback.sh --to <sha>` — Rollback zu spezifischem Commit
- `./rollback.sh --list` — Letzte 10 Deployments tabellarisch
- Steps: Lock → Backup → Checkout → Rebuild → Restart → Health Check → Log
- **Wichtig:** macht KEIN `migrate down`. Nur der interne Auto-Rollback-Pfad in `deploy.sh` berechnet und fuehrt `migrate down N` aus. Bei manuellem Rollback ueber ein Schema-Aenderung hinweg: vorher explizit `$COMPOSE run --rm migrate -path /migrations -database "$DATABASE_URL" down N`.

### smoke.sh — Curl/jq Smoke Tests (24 Tests, Stand 2026-06-05: 24/24 PASS)
Laeuft ohne Go-Toolchain auf jedem Server, <30 Sekunden. `--skip-smoke` aus `cd.yml` entfernt
(`914a12dd`) — Smoke gated wieder jeden CD-Deploy mit Auto-Rollback. Token-Bootstrap: bei
gesetzten `SMOKE_ADMIN_EMAIL`+`PASSWORD` loggt das Skript IMMER frisch ein und ueberschreibt
ein stales `SMOKE_ADMIN_TOKEN` — JWT-Rotationen brechen den Smoke daher nicht.

| Kategorie | Tests |
|-----------|-------|
| Infrastruktur (5) | Gateway health, 10 Services, HTTPS-Cert, Response <2s, Version-Info |
| Auth Flow (3) | Register, Login + JWT, /auth/me |
| CRM CRUD (3) | POST, GET, DELETE /contacts |
| Security (3) | Unauth 401, CORS-Headers, HSTS |
| Performance (3) | /health <500ms, /auth/login <2s, /contacts <1s |
| Cross-Service (2) | Chat-Channel (POST `is_private: false`), Dashboard (`/api/v1/dashboard/layout`) |
| Berichte (3) | GET /berichte/definitions, POST /run, POST /export?format=pdf (MIME-Check) — gated durch `modules.berichte`, 404 akzeptiert wenn Flag OFF |

Flags: `--base-url URL`, `--verbose`, `--expect-version SHA`
Cleanup: Smoke-User wird am Ende per DELETE entfernt.

### healthcheck.sh — Docker Service Health
- Prueft alle App-Service-Container (23 gRPC-Microservices + Gateway = 24 `cmd`-Binaries) + Postgres + Redis + MinIO + LiveKit (×2) + OnlyOffice + Caddy + Monitoring (Prometheus/Grafana/Alertmanager)
- Exit 0 = healthy, Exit 2 = failures
- Welle-1-Hotfix: `set -e` + `((HEALTHY++))` Bash-Bug behoben (`HEALTHY=$((HEALTHY+1))`); Compose-Pfad align mit `deploy.sh`; Caddy-Healthcheck nutzt `--resolve $CADDY_HEALTHCHECK_HOST:443:127.0.0.1` statt hardcoded `https://localhost`. Welle-1-Standalone: 14/14 ✅.

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

## Server-Side Patches via `skip-worktree` (Historisch, 2026-04-19/20)

> **Status (2026-05-08):** Keine aktiven Markierungen mehr. Welle-1-Marathon hat alle Patches aus `main` durch das `render-configs.sh`-Workflow ersetzt. Sektion bleibt fuer Recovery-Lessons. Siehe MEMORY `project_server_redeploy_20260419.md` + `project_sprint3_welle1_deploy.md`.

Auf dem Prod-Server waren zwei Dateien lokal gepatched (`livekit.yaml` + `docker-compose.yml`). Sprint-3-Welle-1-Deploy hat den Cleanup ausgefuehrt:
- `docker-compose.yml`: `DATABASE_URL`/`POSTGRES_PASSWORD` via `${...}` parametrisiert (Sprint 2), `wget --spider` durch `wget -qO-` ersetzt, `formulare` `/health` aligned
- `livekit.yaml`: ersetzt durch `livekit-secrets.yaml`-Render-Overlay (`render-configs.sh` + `deploy.sh` Step 2.5 + `envsubst`-Template aus `.env.production`)

## Sprint-2-TODOs fuer Deploy-Hygiene (Stand 2026-05-08)

| # | Item | Status |
|---|---|---|
| 1 | `docker-compose.yml` 18× hardcoded `kmuhub_dev` eliminieren | ✅ **erledigt** in main, `${DATABASE_URL}`/`${POSTGRES_PASSWORD}` ueberall |
| 2 | Backend `/health` HEAD-Support | ✅ **erledigt** — `RegisterHealth` registriert GET+HEAD (`backend/internal/server/http.go:46-47`) |
| 3 | `formulare`-Service `/health` registrieren | ✅ **erledigt** — Service nutzt `/health` wie alle anderen (`backend/cmd/formulare/main.go:86`) |
| 4 | `deploy.sh` `livekit`+`livekit-egress` in Rolling-Restart-Liste | ✅ **erledigt** in main |
| 5 | `livekit.yaml` Template-Renderer | ✅ **erledigt** — `deploy/scripts/render-configs.sh` + `deploy.sh` Step 2.5. Erweitert 2026-05-09 (`68c0f99`) um `alertmanager.yml.tmpl` (Volume-Mount-Pfad interpoliert `${ALERT_WEBHOOK_URL}` nie, jetzt envsubst-Render). Discord-Refactor 2026-05-09b: Variable von `SLACK_WEBHOOK_URL` zu `ALERT_WEBHOOK_URL` (provider-agnostisch, Discord-Slack-Compat-Mode via `/slack`-Suffix). |
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
- **HTTPS:** Caddy + Let's Encrypt, HSTS, HTTP/2. Seit `7d492bb6`: `handle /rtc*` proxyt auf `livekit:7880` (LiveKit-Signaling mit TLS; twirp-Admin-API bleibt intern; Media via WebRTC-Ports 7881/7882 + TURN). `LIVEKIT_WS_URL=wss://app.zentria.tech` in `.env.production`. **Caddyfile-Aenderungen brauchen `docker compose restart caddy`** (Mount-Inhalt triggert kein Recreate)
- **Firewall:** Hetzner Cloud Firewall `kmuhub-fw` (7 Regeln: SSH/80/443/7880/7881/7882-UDP/ICMP, Source Any IPv4+IPv6)
- **Monitoring:** Prometheus + Grafana + Alertmanager (alle localhost-only, SSH-Tunnel)
- **Alertmanager:** `prom/alertmanager:v0.27.0`, Port 9093, Config wird zur Deploy-Zeit aus `deploy/docker/alertmanager.yml.tmpl` via `render-configs.sh` (Step 2.5 in `deploy.sh`) gerendert (`alertmanager.yml` ist `.gitignore`d). Webhook via `${ALERT_WEBHOOK_URL}` aus `.env.production` — Discord-Webhook im **Slack-Compat-Mode** (URL endet auf `/slack`), Receiver `slack_configs` bleibt unveraendert. Channel `#cosmi-prod-alerts` im zentria-intel-Discord-Server. Empty → `slack_api_url: ""`, Service startet sauber ohne Notifications. 3 Rules: ServiceDown (2m), HighErrorRate (5%), DBConnectionsHigh (80% max_connections).

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

## Ansible Pilot-Provisioning (S3.1, 2026-05-08)

`deploy/ansible/` ist die Pilot-Onboarding-Schicht — eine Pilot-Instanz wird in <30 Min vollautomatisch deployed (Voraussetzung fuer das Instanz-pro-Pilot-Modell ab M3, siehe MEMORY `project_multi_tenancy.md`).

### Verzeichnis-Struktur
```
deploy/ansible/
├── .ansible-lint                              # role-name skip mit Begruendung (app-deploy-Hyphen)
├── README.md                                  # Run-Anleitung + Vault-Setup
├── ansible.cfg                                # roles_path, inventory, host_key_checking=False
├── inventory/
│   ├── hosts.yml                              # pilot-0-zfa + turn-shared (PLATZHALTER_IP)
│   └── group_vars/{all,pilots,turn}.yml
├── site.yml                                   # import_playbook: provision.yml
├── playbooks/
│   ├── provision.yml                          # foundation + secrets + app-deploy fuer pilots; foundation + turn fuer turn-Hosts
│   └── update.yml                             # nur app-deploy (re-run)
└── roles/
    ├── foundation/                            # 19 Tasks: apt + docker + UFW + fail2ban + cron
    ├── secrets/                               #  2 Tasks: 12 Secrets via openssl + env.production.j2
    ├── app-deploy/                            # 15 Tasks: git pull + render-configs + serial-build + migrate + rolling-restart + healthcheck + smoke
    └── turn/                                  # 14 Tasks: coturn + UFW (3478/5349/relay) + Let's Encrypt + Renew-Hook
```

### Roles im Detail

| Role | Source-of-Truth | Pflicht-Variablen | Notiz |
|---|---|---|---|
| `foundation` | `deploy/scripts/setup-cron.sh`, `deploy/turn/setup.sh`, `deploy/hetzner/firewall.sh` | `deploy_user_pubkeys` (zwingend in `vault.yml`) | UFW oeffnet 22/80/443/7881-tcp/**7882-udp**/50000-60000-udp (7882/udp ist Lueckenfix vs. `firewall.sh`) |
| `secrets` | `deploy/docker/PRODUCTION_TEMPLATE` | `env_target_path` (default `/opt/kmuhub/.env.production`) | 12 Secrets generiert (`JWT_SECRET`, `VAULT_MASTER_SECRET`, `WOPI_JWT_SECRET`, `ONLYOFFICE_JWT_SECRET`, `LIVEKIT_API_KEY/SECRET/WEBHOOK_SECRET`, `TURN_SECRET`, `POSTGRES_PASSWORD`, `MINIO_ROOT_PASSWORD`, `MINIO_SECRET_KEY`, `GF_SECURITY_ADMIN_PASSWORD`); `no_log: true` aktiv |
| `app-deploy` | `deploy/scripts/deploy.sh` Steps 2.5-7 | `repo_url`, `deploy_ssh_key_path`, `pilot_domain` | Build laeuft seriell ueber `app_services` (16 GB RAM-Constraint), Caddyfile.j2 mit `{{ pilot_domain }}` und `{{ caddy_security_headers }}`-Loop, smoke `failed_when: false` (12/21 Known-Broken bis Repair) |
| `turn` | `deploy/turn/turnserver.conf.template`, `deploy/hetzner/coturn-setup.sh` | `turn_realm`, `certbot_email`, `cloudflare_api_token` (optional) | TLS via Let's Encrypt standalone + Renew-Hook `/etc/letsencrypt/renewal-hooks/deploy/coturn.sh`; optional Cloudflare-DNS-Helper gated auf `cloudflare_api_token` |

### Verifikation auf Windows-Dev-Box

Native-Windows-Ansible funktioniert NICHT (`No module named 'grp'` — Unix-only). Verifikation laeuft in einem Docker-Container:

```bash
# Setup einmalig: Image pullen
docker pull willhallonline/ansible:latest

# Wrapper-Pattern (MSYS_NO_PATHCONV ZWINGEND fuer Git-Bash, sonst Path-Translation)
MSYS_NO_PATHCONV=1 docker run --rm \
  -e ANSIBLE_ROLES_PATH=/work/deploy/ansible/roles \
  -v "/c/Users/Luke/Documents/KMU Hub:/work" \
  -w /work/deploy/ansible \
  willhallonline/ansible:latest \
  ansible-playbook -i inventory/hosts.yml --syntax-check site.yml

# ansible-lint production-profile (Image hat es vorinstalliert)
MSYS_NO_PATHCONV=1 docker run --rm \
  -v "/c/Users/Luke/Documents/KMU Hub:/work" \
  -w /work/deploy/ansible \
  willhallonline/ansible:latest \
  ansible-lint roles/
```

Real-Apply gegen Linux-Server weiterhin nur von einer Linux-Control-Node moeglich. Pilot-0-IP wird vor Real-Provisioning in `inventory/hosts.yml` gesetzt (nach Hetzner-VM-Bestellung + ZFA-Akquise-Bestaetigung).

### Vault-Setup (Operator-Pflicht vor erstem Run)

```bash
ansible-vault create deploy/ansible/inventory/group_vars/pilots/vault.yml
# darin definieren: deploy_user_pubkeys (List[String]), cloudflare_api_token (optional)
ansible-playbook -i inventory/hosts.yml site.yml --ask-vault-pass
```

KEIN `vault.yml` mit Pseudo-Secrets im Repo — Operator initialisiert selbst. README dokumentiert das Pattern.

### Status & Followups
- ansible-lint **production-profile 0 failures** ueber alle 4 Roles
- Welle-3-Subagent-Drift war benign (beide Streams haben `provision.yml` an disjunkten Plays editiert, `git pull --rebase` hat sauber gemerged)
- **Out-of-Scope heute:** Real-Apply gegen Test-VM (Multipass/Vagrant-Setup) — separater Tag

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
