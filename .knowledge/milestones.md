---
tags: [fortschritt, milestones]
updated: 2026-05-08
---
# Milestones

## Sprint 3 Closure 2026-05-08 — Welle 1 Production-Deploy + Welle 2/3 Ansible + Dialer-Coverage

**14 Direct-to-Main-Commits** an einem Marathon-Tag (8h+). Sprint-3-Gate erreicht, **8/8 Pflicht-Tasks done**. Plan-Files: `~/.claude/plans/wir-machen-weiter-an-adaptive-lark.md` (Welle 0/1) + `~/.claude/plans/wir-machen-weiter-an-abstract-cray.md` (Welle 2/3/4).

### Welle 1 — Production-Deploy 81 → 115 (Marathon)

Server-Deploy von Migration **81 → 115** (34 Migrations-Ramp), 32 Container healthy auf `3abec5f`. **9 Hotfix-Commits** zur Deploy-Infrastruktur (7 versteckte Production-Bugs aufgedeckt):

| # | SHA | Bug |
|---|---|---|
| 1 | `089c2d4` | rollback.sh-Service-Liste hatte nur 10 von 25 Sprint-2-Services |
| 2 | `f4add92` | PRODUCTION_TEMPLATE fehlte SLACK_WEBHOOK_URL fuer Alertmanager |
| 3 | `53dd5b6` | `docker buildx bake` parallel-build OOM bei 16 GB RAM (24 Go-Builds gleichzeitig) — serial-Loop-Fix |
| 4 | `c7a9a76` | Migrations 000114+115 referenzieren `tenants(id)` ohne dass `tenants`-Tabelle je angelegt wurde — `CREATE TABLE IF NOT EXISTS` + Sentinel an Anfang von 000114 |
| 5 | `3c1ffcd` | `redis:7.2.7-alpine` kann RDB-v12 (von 7.4+) nicht lesen — Pin auf `redis:7.4-alpine` |
| 6 | `32588ed` | `minio/mc:RELEASE.2025-04-16T19-25-36Z` aus Docker Hub entfernt — Tag-Rotation auf 2025-05-21 |
| 7 | `7da7ed8` | healthcheck.sh `((HEALTHY++))` mit `set -e` brach nach erstem [OK] ab — `HEALTHY=$((HEALTHY + 1))` |
| 8 | `61b0996` | healthcheck.sh nutzt veralteten compose-Pfad (vor `deploy/docker`-Refactor) — `COMPOSE_FILES_DIR + ENV_FILE` align mit deploy.sh |
| 9 | `3abec5f` | healthcheck.sh checkt `https://localhost` statt configured Domain — `--resolve $CADDY_HEALTHCHECK_HOST:443:127.0.0.1` |

Smoke-Test 12/21 PASS (9 known-broken Tests deferred — siehe MEMORY `project_sprint3_welle1_deploy.md`). Alle 25 Application-Services + Caddy + Postgres + Redis (7.4) + MinIO + Prometheus + Grafana + Alertmanager healthy. OnlyOffice unhealthy seit 2 Monaten (separater Bug).

### Welle 2/3 — S3.7 Dialer + S3.1 Ansible

| Commit | Welle | Inhalt |
|---|---|---|
| `1f6c4c0 test(dialer)` | 2A | Dialer-Coverage 12% → **31.8%** (./internal/dialer/...). 4 neue Test-Files: `phone_test.go` (NormalizePhoneE164 + FormatDuration table-driven), `redis_agent_store_test.go` (ValidateTransition-State-Machine + parseAgentStatusData), `dialer_grpc_test.go` (mapDialerError 10 Sentinels + Handler-Tests). `service_test.go` 607 → 1169 LOC (9 Service-Methoden Cover). **Bonus-Fix:** `mapDialerError` mappt `consent.ErrNoConsent` jetzt auf `codes.PermissionDenied` (war `codes.Internal`). |
| `a8d77fc feat(ansible)` | 2B | `deploy/ansible/` greenfield: `ansible.cfg` + `site.yml` + provision/update playbooks + inventory hosts.yml (pilot-0-zfa + turn-shared, Platzhalter-IPs) + group_vars + roles/foundation (19 Tasks: apt + docker + UFW incl. **7882/udp gap-fix** + fail2ban + cron) + roles/secrets (2 Tasks: 12 Secrets via openssl + env.production.j2). 18 Files, ~570 LOC. |
| `71f7c90 feat(ansible)` | 3A | roles/app-deploy (15 Tasks: git pull + render-configs + serial-build + migrate + rolling-restart + healthcheck + smoke). `templates/Caddyfile.j2` mit `{{ pilot_domain }}` + `caddy_security_headers`-Loop. `playbooks/provision.yml` pilots-Play jetzt `foundation + secrets + app-deploy`. `.ansible-lint` `role-name`-Skip mit Begruendung (app-deploy-Hyphen). |
| `562e9c5 feat(ansible)` | 3B | roles/turn (14 Tasks: coturn + UFW 3478/5349/relay-range + Let's Encrypt via certbot standalone + Renew-Hook + turnserver.conf.j2). DNS-Helper: optional `community.general.cloudflare_dns`-Task gated auf `cloudflare_api_token`. |
| `d8f917e docs(roadmap)` | 4 | Sprint-3-Closure ROADMAP-Update — S3.1 + S3.7 als done, Migration-Head 115, Sprint-3-Gate erreicht. |

**Ansible-Stand:** 4 Roles, **50 Tasks**, ansible-lint **production-profile 0 failures** ueber alle Roles. Tooling: Native-Windows-Ansible nicht funktioniert (`No module named 'grp'`), Verifikation via Docker-Wrapper (`willhallonline/ansible:latest` + `MSYS_NO_PATHCONV=1` + Volume-Mount). Pattern reproduzierbar fuer kuenftige Ansible-Arbeit.

**Sprint-3-Tasks Final:**

| Task | Status |
|---|---|
| S3.MT.2 Option-B Phase 2 (~38 Tabellen) | ✅ Migrations 000114+115 (gestern + heute Hotfix `c7a9a76`) |
| S3.1 Ansible-Playbook | ✅ `a8d77fc` + `71f7c90` + `562e9c5` |
| S3.2 CI Security-Scans | ✅ gosec + trivy + npm audit |
| S3.3 Dialer LogCallOutcome Tx | ✅ UpdateSessionWithEvent atomic |
| S3.4 Image-Tags pinnen | ✅ + 2 Welle-1-Korrekturen (redis 7.4, minio/mc Tag-Rotation) |
| S3.5 Alertmanager + Slack | ✅ (`SLACK_WEBHOOK_URL` Server-Set noch offen, kein Blocker) |
| S3.6 cd.yml Auto-Deploy | ✅ workflow_run-Trigger |
| S3.7 Dialer-Coverage 12% → 30% | ✅ `1f6c4c0` (31.8% real) |

S3.MT.4 Audit auf Sprint 5 verschoben (User-Entscheidung). **Naechste Schritte:** Sprint 4 ab 2026-06-08 (`finance_invoices.line_items`-Normalisierung nach ADR-0007, Skeleton in `a1a8d54` bereit) → Sprint 5 (Peer-Review + Rigorosum Runde 3) → Launch 01.07.

**Lessons:**
- **Image-Pinning ohne expiration-tracking ist fragil.** minio/mc-Tags rotieren weg, redis-Pins koennen Down-grade sein. Sprint-3-Image-Pin-Commit `7a22d83` brauchte 2 Korrekturen.
- **Migrations 000114/115 wurden gestern committed ohne lokalen Migrate-Run-Test.** Der `tenants`-FK-Bug waere bei einem `migrate up` von leerer DB sofort aufgefallen. Strategischer Followup: lokaler `make migrate-up` Pre-Commit-Hook ODER CI-Job der eine fresh-DB hochzieht.
- **healthcheck.sh hatte 3 Bugs gleichzeitig** (`set -e` vs `((counter++))`, Pfad-Drift, Domain-Hardcode). Nie integrativ getestet. Cleanup uebernommen.
- **Subagent-STOP-Direktive funktioniert.** Welle-2-Stream-B-V1 stoppte sauber am Pre-Check (Ansible-Tooling fehlt) ohne Files anzulegen — Lesson aus Welle 4B (partielle Verifikation als "alles gruen") funktional bewiesen.
- **Server-DB war leer (91 KB pg_dump)** — Backfills in Hot-Risk-Migrations 000114/115 waren risikolos. Bei Pilot-1 mit Million-Row-Tabellen anders bewerten.
- **deploy.sh inline-rollback rebuilds redundant** wenn `PREV_SHA == NEW_SHA` (followup).
- **Server-RAM 16 GB ohne Swap** — knapp fuer parallel Go-Builds. Serial-Build-Fix gut, langfristig CCX21 (32 GB) fuer Pilot-1 evaluieren.

## Sprint 3 Session 2026-05-08 — Option-B Phase 2 Abschluss + CI Security + Dialer-Tx

**9 Direct-to-Main-Commits** in 4 Wellen (8.5h). 6 von 8 Sprint-3-Pflicht-Tasks erledigt. S3.MT.2 (Option-B Phase 2) komplett. Build/Vet/Test/tsc/npm test alle gruen. Plan-File: `~/.claude/plans/wir-haben-heute-von-composed-emerson.md`.

| Commit | Welle | Summary |
|--------|-------|---------|
| `5a3023a chore(deploy)` | 0 | Sprint-2-Server-Drift-Fixes: 8 Sprint-2-Services in deploy.sh-Restart-Liste; 5 von 5 Sprint-2-TODOs als done markiert |
| `76e4986 test(welle4b)` | 1A | F6-F9 Followups + Pre-Consent-Test-Refactor: DB-Backed Cross-Tenant-Tests Calendar/Email/Recordings, assert.Eventually statt time.Sleep (F7), F8+F9 verifiziert |
| `291e1f7 fix(welle3-p2)` | 1B | Frontend P2-Followups: useSetRecordingConsent global invalidate, offline-queue retryDeadLetter, OfflineBanner Retry-Buttons, 3 neue Vitest-Cases |
| `7a22d83 chore(deploy)` | 1C | Image-Tags gepinnt (S3.4), Alertmanager v0.27 + alertmanager.yml + DBConnectionsHigh-Rule (S3.5), cd.yml workflow_run-Trigger + concurrency-Group + Slack-Notify (S3.6) |
| `241686e ci` | 1D | 3 parallele CI-Jobs: gosec + trivy fs-scan + npm audit (S3.2); .gosec.yaml-Baseline (G104/G304/G108) |
| `0cba503 feat(tenant)` | 2A | Migration 000114: 16 Tabellen (user-dashboard/preferences/document-layer/security-auth/caldav-push). Repos: dashboard tenantID-Cache-Key, document/folder |
| `eab0181 fix(dialer)` | 2C | UpdateSessionWithEvent atomare Tx, Mutex auf mockCallRepo, TestLogCallOutcome_Concurrent_SameSession (S3.3) |
| `298f8ea feat(tenant)` | 2B/3 | Migration 000115: 22 Tabellen (caldav-sync/integration-mappings/chat-call-guest). Repos: bexio + lexware; 8 neue Cross-Tenant-Tests |
| `a1a8d54 docs(adr)` | 3 | ADR-0007 Finance Line Items Normalization, Sprint-4-Plan, 3 Skeleton-Tests `t.Skip("ADR-0007: pending")` |

**Sprint-3-Tasks:**

| Task | Status |
|------|--------|
| S3.MT.2 Option-B Phase 2 (~38 Tabellen) | ✅ Migrations 000114 + 000115 |
| S3.2 CI Security-Scans | ✅ gosec + trivy + npm audit |
| S3.3 Dialer LogCallOutcome Tx | ✅ UpdateSessionWithEvent atomic |
| S3.4 Image-Tags pinnen | ✅ docker-compose.yml + .prod.yml |
| S3.5 Alertmanager + Slack | ✅ prom/alertmanager:v0.27.0 |
| S3.6 cd.yml Auto-Deploy | ✅ workflow_run-Trigger |
| S3.1 Ansible-Playbook | ⏭ Naechste Session (5 PT) |
| S3.7 Dialer-Coverage 12 → 30% | ⏭ Naechste Session (8 PT) |

**Naechste Schritte:** S3.1 Ansible + S3.7 Dialer-Coverage → Server-Deploy (Migrations 82→115) → Sprint 4 (Start 2026-06-08, finance_line_items-Normalisierung nach ADR-0007).

## Sprint 2 Welle 4B Session 2026-05-07 — Option-B Phase 2 + Idempotency HardMode-Bereitschaft

Drei Sub-Wellen (4B.1 + 4B.2 + 4B.3) mit insgesamt 10 Sonnet-Subagents (4B.1: 4 parallel + 1 Sweep, 4B.2: 4 parallel, 4B.3: 1 Explore + 1 Fix). Zwei konsolidierte Direct-to-Main-Commits.

| Commit | Inhalt |
|--------|--------|
| `b868fb6 feat(welle4b): option-b phase 2 + idempotency hardmode readiness` | 105 Files, +3687/-1358. 5 neue Migrations 000109-000113 (Calendar/Work-Internal, Email/Inbox/Notification, Security/CRM-Aux, Automation-Exec/Channel-Memberships, Idempotency partial Index). 49 neue tenant_id-Spalten + JOIN-Backfill fuer automation_executions+channel_memberships. 16+ Repository-Wirings (work/calendar+meeting+resource, email/*, notification/*, inbox/*, crm/tag+consent+search). Idempotency `Complete()` Composite-PK-Fix + HardMode-Env-Flag in main.go (Default WarnMode, Dev-Default Hard via docker-compose). 13 gRPC-Handler auf `middleware.GetTenantID(ctx)`. 12 P2-7 Cross-Tenant-Tests + 3 finance JSONB-Tests. 8 P2-Followups integral (P2-1, -2, -3, -5, -6, -7, -8, P3-3). |
| `1b1eb37 fix(welle4b): close 5 P0+P1 findings from welle 4B.3 sweep` | 9 Files, +195/-97. **F1 P0:** `video_grpc.StartRecording` schrieb uuid.Nil als tenant_id auf jeder neuen recording-Row — tenantID jetzt vor if/else extrahiert + an Variadic. **F2 P0:** 12 Deal/Activity-Handler in `crm_grpc.go` von `req.TenantId`-Spoof auf `middleware.GetTenantID(ctx)` (Welle-3.5-P0-Carryover). **F3+F10 P1:** meeting_action_items INSERT/UPDATE/DELETE/Get mit tenant_id-Filter + neue `GetActionItemByID`-Methode. **F4 P1:** ConvertActionItemsToTasks uuid.Nil-Guard. **F5 P1:** .env.example IDEMPOTENCY_MODE=hard auskommentiert. |

**Verifikation:** `go build ./...` 0 Errors, `go vet ./...` 0 Issues, `go test ./... -count=1` alle Pakete OK, `npx tsc --noEmit` 0 TypeErrors, `npx vitest run` 14/14 Files / 202/202 Tests pass.

**4 P2/P3 deferred** in `docs/sprint2-welle4b-followups.md`: F6 echte DB-Backed Cross-Tenant-Boundary-Tests (statt nur HTTP-Smoke), F7 TestHardMode time.Sleep flaky, F8 email/account ListAllActive Caller-Audit-Comment, F9 ListBrowsable shared-Filter-Frage. Plus Idempotency HardMode Prod-Cutover (separate Sprint-3-Deploy-Aktion nach Pilot-1).

**Cross-Stream-Drift-Lesson:** Drei Mal trafen stale IDE-Diagnostics — Subagent meldete "alles gruen", IDE-Diagnostics zeigten Sig-Drift in `cmd/*/main.go` und gRPC-Handlern, `go build ./...` direkt war clean. Authoritative Verifikation ist `go build ./...`, nicht IDE-Diagnostics.

## Sprint 2 Welle 4A Session 2026-04-29 — Repository-Layer-Wirings + Idempotency Client Coverage

Vier parallele Sonnet-Subagents (Stream A/B/C/D), ein konsolidierter Direct-to-Main-Commit `4ff5fa2 feat(welle4a): wire repos with tenantID + idempotency client coverage`. 109 Files, +3326/-3325 Zeilen (Net +1, Test-Konsolidierung kompensiert Insertions). Stream B Bonus: `work/event` mitgewired (calendar_events). Stream C Repair: Cross-Package-Test-Mock-Fallout in `work/task/service_test.go` und `automation/workflow/service_test.go` nachgezogen.

| Cluster | Inhalt |
|---------|--------|
| **Backend Repository-Wirings (~75 Files)** | `automation/workflow` + `automation/engine`, `chat/channel`, `crm/customfield`/`crm/report`/`crm/savedfilter`, `dialer` (+ `consent_test.go`), `document/file` + `document/wopi` + `gateway/wopi_adapter` + `cmd/document/main`, `email/message`, `inbox/message`/`inbox/routing`/`inbox/team`, `security/audit` + `models/security`, `work/event`/`work/project`/`work/timeentry` + `work/task/service_test`. Repository/Service-Signaturen tragen `tenantID uuid.UUID` durch, Postgres-Repos enforcen `WHERE tenant_id=$1` first. |
| **gRPC Layer (10 Handler)** | `automation_grpc.go`, `calendar_grpc.go`, `chat_grpc.go`, `crm_grpc.go`, `dialer_grpc.go`, `document_grpc.go`, `email_grpc.go`, `inbox_grpc.go`, `security_grpc.go`, `work_grpc.go` lesen `tenant_id` jetzt aus `middleware.GetTenantID(ctx)` statt aus Proto-Feld. Erweitert die Welle-3.5-Lesart auf den vollen gRPC-Layer. |
| **Frontend authenticatedFetch-Helper** | Neuer `desktop/src/renderer/src/api/utils/authenticatedFetch.ts` zentralisiert Auth-Header + Idempotency-Key-Generierung + Offline-Queue-Hooks + Error-Mapping. **32 API-Clients refactored**: automation, berichte, bexio, caldav, calendar, crm-import, datev-upload, dialer, einkauf, email, finance, formulare, fuhrpark, helpdesk, hr, inbox, integration, inventar, lexware, notification, plugin, produktion, rapporte, schichten, security, vermietung, vertraege, video, wiki + 3 weitere. Eliminiert Duplikat-Code in jedem Client. |
| **Frontend Idempotency-Coverage-Test-Suite** | Neue `desktop/src/renderer/src/api/__tests__/idempotency-coverage.test.ts` mit **29 Cases** — pro API-Client-Familie ein Round-Trip ueber `authenticatedFetch` plus Idempotency-Key-Header-Verifikation. Stellt sicher, dass der Frontend-Idempotency-Rollout flaechendeckend ist (Voraussetzung fuer Idempotency HardMode in Welle 4B). |
| **Tests** | Repo-weit `go vet ./...` 0 Issues, `go test -count=1 ./...` alle Pakete OK (kein FAIL). 24 `cmd/<svc>` seriell mit `-ldflags="-w -s"` gebaut (Linker-OOM-Workaround, siehe [[troubleshooting]]). Frontend: 14 Test-Files, 202/202 Tests gruen (inkl. neuer 29 idempotency-coverage). |

**Gesamt:** 1 Commit auf main (`4ff5fa2`), keine Migrations. Build+Vet+Test+tsc+npm test gruen. Pause-Gate aktiv: User-Review → Welle 4B (Top-30+ Tabellen Option-B Phase 2 + restliche Repo-Wirings + Idempotency HardMode + P2/P3-Followups aus `docs/sprint2-welle3-followups.md`).

## Sprint 2 Welle 3.5 Session 2026-04-29 — Bugfix-Sweep nach Welle 3

Konsolidierter Hardening-Commit `d443ab4 fix(welle3): close 34 findings from welle 3 review` — 17 P0 + 17 P1 closed, P2/P3 in `docs/sprint2-welle3-followups.md` deferred. 47 Files, +1029/-434 Zeilen. Build+Vet+Tests+tsc+npm test alle gruen (`173/173`).

| Cluster | Inhalt |
|---------|--------|
| **gRPC Tenant-Spoof-Sweep** | `chat_grpc.go`/`crm_grpc.go`/`work_grpc.go`/`video_grpc.go`/`dialer_grpc.go` lesen `tenant_id` jetzt aus `middleware.GetTenantID(ctx)` statt `req.GetTenantId()`. Verhindert Tenant-Spoof bei Service-zu-Service-Calls oder kompromittiertem Gateway. 14+ Methoden affected. |
| **Repository-Tenant-Filter-Sweep** | `deal/activity/task/pipelinestage/chat-message/recording postgres_repository.go` enforcen `WHERE id=$1 AND tenant_id=$2` auf jedem UPDATE/DELETE/GetByID/Search-Pfad plus `RowsAffected==0`-Sentinel. Repository-Interfaces an PostgresRepository-Signaturen synchronisiert (`CachedRepository` reicht tenant_id durch). 14 Model-Structs mit einheitlicher `TenantID`-Tag. `pipelinestage.scanStage` liest Spalte aus 000106 (vorher Scan-Mismatch). |
| **Idempotency-Hardening** | Migration 000108 setzt PK auf `(tenant_id, key)` (Cross-Tenant-Cache-Replay-Schutz fuer HardMode). Atomare Reserve via `INSERT ... ON CONFLICT DO UPDATE RETURNING`. `errors.Is`-Matching auf `ErrInFlight`/`ErrConflict`/`ErrKeyMissing` (vorher String-Equality, fail-open auf wrapped errors). `context.WithoutCancel` fuer async Complete (kein Deadlock auf Handler-Return). Cleanup-Worker bekommt echte `pg_try_advisory_xact_lock`-Leader-Election. Middleware-Stack-Position fixiert nach Auth+RBAC. |
| **Recording-Robustness** | `StartRecording` reordert Pre-Consent-Check VOR `CreateRecording` (verhindert Orphan-Rows bei `ErrConsentPending`). `MarkInitiatorConsent` + `GetPreConsentStatus` filtern `tenant_id` mit `RowsAffected==0`-Sentinel. `route_video.go` setzt RBAC-Permission-Middleware vor den Endpoint. |
| **Frontend** | `CallControls`: Doppelklick-Guard via `startRecording.isPending`/`stopRecording.isPending`/`confirmInitiatorConsent.isPending`. `handleConfirmStart` wrapped in try/catch + `sonner.toast.error` bei Failure (kein Orphan-Recording-State, User kann erneut bestaetigen). `offline-queue.ts`: 409 (in-flight) als Retry-Class statt silent-drop, `Content-Type` nur bei vorhandenem Body. |
| **Tests** | `gateway/tenant_isolation_test.go` um 4 Cases fuer `/recordings/{id}/initiator-consent` erweitert (no-tenant, empty-tid, valid-tid, two-tenant). Migration `000106_tenant_id_retrofit_phase1.down.sql` ergaenzt mit Doc-Kommentar zur bewussten FK-Abwesenheit. Frontend-Tests: 173/173 gruen. |

**Gesamt:** 1 Commit auf main (`d443ab4`), Migration 000108 hinzu (Head: `108`). Build+Vet+test+tsc+npm test gruen. Pause-Gate aktiv: User-Review → Welle 4 (Idempotency HardMode + restliche 15 Top-20-Repos + Top-30+ Tabellen). P2/P3-Followups in `docs/sprint2-welle3-followups.md`.

## Sprint 2 Welle 3 Session 2026-04-28 (Spaetabend) — R2-P0.4 + R2-P0.7 + Option-B Phase 1

Drei parallele Sonnet-Subagents, drei direct-to-main Commits, Migrations 000105/000106/000107 (statt 104/105/106 weil W2D-A 000104 belegt). Schliesst die letzten beiden offenen R2-P0-Blocker (Recording-Consent UX + Offline-Queue) und startet die Option-B-Retrofit-Welle ueber ~50 Tabellen (Phase 1: Top-20 Hot-Path).

| Commit | Stream | Inhalt |
|--------|--------|--------|
| `174a7e4` | Stream B (R2-P0.7) | Migration 000105 `idempotency_keys` (tenant-scoped, expires_at-Cleanup). Backend: `internal/idempotency/` Postgres-Repo (`ON CONFLICT DO NOTHING` Reserve-Race), `middleware.Idempotency` in **WarnMode** Default mit Replay/Conflict-422/InFlight-409+Retry-After:2/Fresh-Pfaden + Auth-Whitelist, Cleanup-Goroutine 1h-Tick auf gateway main. Frontend: `api/offline-queue.ts` IndexedDB-Queue (idb-keyval, max 5 parallel, exp-Backoff, Dead-Letter nach 5 Versuchen), `api/idempotency.ts` UUIDv4-Auto-Header, `client.ts` enqueued bei `!navigator.onLine` statt `OfflineError`-Throw, `useOnlineStatus.drain()` bei Online-Event, `OfflineBanner`-UI. Tests: 20 Backend (50.2% Middleware-Cov) + 11 Frontend (fake-indexeddb). |
| `5d7fb0d` | Stream C (Option-B Phase 1) | Migration 000106 — `tenant_id UUID NOT NULL DEFAULT '00...000001'` + Per-Table-Index + 9 Composite-Hot-Path-Indizes auf 20 Tabellen: deals/activities/tasks/projects/channels/messages/notifications/time_entries/calendar_events/email_messages/inbox_messages/deal_stage_history/pipeline_stages/saved_filters/custom_field_definitions/automations/document_files/recordings/dialer_call_sessions/audit_log. Top-5 (deals/activities/tasks/messages/notifications) komplett gewired: Repo CREATE/GetByID/List/Delete mit `tenant_id`-First-Filter, Service-Signaturen mit tenantID-Param via `middleware.GetTenantID`, Proto-Erweiterung um `tenant_id` auf 14 RPCs (crm.proto, work.proto, chat.proto), gRPC-Handler parsen `req.TenantId`, Gateway-Routes mit `GetTenantID`-First-Action (401 bei Fehler). 15 neue Cross-Tenant-Tests in `tenant_isolation_test.go`. Restliche 15 Tabellen haben Spalte+Default, Full-Wiring deferred auf Welle 4. |
| `f6af609` | Stream A (R2-P0.4) | Migration 000107 — `recordings.pre_recording_consent_at` + `initiator_consent_id`, `recording_consents.responded_at`, Partial-Index. Backend: `recording.Service.ConfirmInitiatorConsent` stempelt, `StartRecording` returniert `ErrPreConsentMissing` (HTTP 412) wenn Stamp fehlt, neuer gRPC-RPC via `proto/video/v1/video_pre_consent_ext.go` (Handfile-Extension-Pattern, kein .proto-Regen). Endpoint `POST /api/v1/video/recordings/{id}/initiator-consent`. Frontend: `RecordingInitiatorDialog` (Radix AlertDialog, non-dismissible) ZWINGEND vor `startRecording` — Initiator bestaetigt aktiv. `RecordingActiveBanner` (roter Top-Stripe waehrend Aufnahme). i18n-Keys `recordingBanner.*` + `recordingInitiator.*` in de/en/fr/it. Tests: 4 Service + 3 Gateway + 4 Frontend. |

**Gesamt:** 90 Files, ~4200 LOC, 3 Commits + 1 Knowledge-Commit. `go build ./...` + `go vet ./...` + `go test ./...` + `tsc --noEmit` alle gruen. Idempotency-Middleware bleibt in **WarnMode** bis Welle 4 (Frontend-Rollout der Idempotency-Keys muss vollstaendig ausgerollt sein, dann HardMode).

**Pause-Gate aktiv:** Nach Welle 3 → User-Review + Bugfix-Sweep (Welle 3.5, analog Welle 2C-Pattern), bevor Welle 4 startet (Top-30+ Tabellen + restliche 15 Top-20-Repos + Idempotency HardMode).

## Sprint 2 Welle 2D Session 2026-04-28 (Abend) — JWT-Tenant-Hardening

Welle-1-Altlast geschlossen: vor Welle 2D hatten 11 Gateway-Routes hardcoded `<modul>PlaceholderTenantID = "00000000-...-000000000001"` ohne JWT-Claim-Extraction → Cross-Tenant-Isolation auf HTTP-Ebene defekt. Drei aufeinanderfolgende Commits, alle direct-to-main:

| Commit | Inhalt |
|--------|--------|
| `33450e7` (W2D-A) | `auth.Claims.TenantID string \`json:"tid"\`` + `CreateAccessToken(userID, tenantID, ...)` + Migration 000104 (`users.tenant_id` mit `idx_users_tenant`) + Middleware `GetTenantID(ctx) (uuid.UUID, error)` (fail-closed via `ErrMissingTenantID`). 11 Routes refactored: rapporte/schichten/fuhrpark/vermietung/inventar/einkauf/produktion/berichte/formulare/wiki/vertraege. 10 neue `gateway/tenant_isolation_test.go` Tests. |
| `c421fac` (W2D-B Hotfix) | `auth/postgres_repository.go` SELECTed `tenant_id` jetzt — vorher leeres Feld → `tid`-Claim immer empty trotz Issuance. Service-Layer + Test-Update. |
| `8f055e3` (W2D-C Sweep) | 5 Cross-Layer-Holes geschlossen: `dialer_grpc.go`/`helpdesk_grpc.go` (13 Proto-Requests um `tenant_id` erweitert + pb.go regen), `route_wiki.go` (4 Handler verwarfen tenantID), `route_biz.go::getTenantID(r)` rief `GetUserID` statt `GetTenantID` (UserID-als-TenantID-Surrogate in 90 Call-Sites quer durch biz/billing/invoices/quotes/ext/hr/lexware/bexio/datev). |

**Gesamt:** 42 Files, ~2200 LOC, 3 Commits. Proto-Aenderungen in `dialer.proto` + `helpdesk.proto`. W2D-B war ursprünglich als separater Schritt geplant, wurde aber redundant (Server-Drift bereits in `bed88ab` → `fd7e3b2` 2026-04-25/26 erledigt). `go build ./...` + `go vet ./...` + Tests gruen. Pflicht-Pattern fuer alle neuen Routes: `tenantID, err := middleware.GetTenantID(r.Context())` als erste Aktion → 401 bei Fehler. Details: [[security]] "JWT Tenant-Claim & Cross-Layer-Hardening".

## Sprint 2 Welle 2A Session 2026-04-28 — 4 Handwerk-Module Backend

4 parallele Sonnet-Subagents schreiben rapporte/schichten/fuhrpark/vermietung Backends direkt auf main, je mit Welle-1-Inventar als Template-Anker. Plan: `~/.claude/plans/alright-was-steht-als-noble-rabin.md`.

| Schritt | Details |
|---------|---------|
| Welle 2A — fuhrpark | `e4b1a62` (Self-Commit Subagent) — 22 Files, ~6k LOC, Migrations 000096+97 (vehicles/services/damages), 18 RPCs, TÜV-Reminder-Cron-Worker mit `pg_try_advisory_xact_lock` Leader-Election + 7d/1d-Vorlauf-Fenster + Idempotenz via `tuev_reminder_sent_at`, Ports 50076/9116, Coverage 39.7% (21 Tests) |
| Welle 2A — rapporte/schichten/vermietung | `c52839f` (konsolidierter Commit) — 53 Files, 21.8k Insertions. rapporte (50074/9114, Migrations 92+93, 35 Tests, 35.6% Cov, Approval-State-Machine + GPS-Tag), schichten (50075/9115, Migrations 94+95, 38 Tests, 35.6% Cov, ArbZG §5 Pre-Check inkl. DST-Spring-Forward), vermietung (50077/9117, Migrations 98+99, 34 Tests, 41.3% Cov, GIST-tstzrange-Overlap-Index gegen Doppelbuchung) |
| Pflicht-Guards aus `ad04191` | Alle 4 Module von Anfang an: Pre-Check vor State-Transitions/Mengenwrites, `tenant_id`-Filter in jedem Get-by-ID, `RowsAffected() == 0` Sentinel auf jedem UPDATE/DELETE |
| Knowledge-Base Update | `_index.md`/`architektur.md`/`datenbank.md`/`api.md`/`milestones.md` |

**Gesamt:** 2 Commits auf main, 75 Files, ~30k LOC, 128 Tests, alle Module-Coverages 35–41% (>30%-Ziel), `go build ./...` + `go vet ./...` clean. Alle 4 Module Feature-Flag default OFF. Open: TÜV-Notification-Delivery (Sprint-3-Wiring), `assigned_driver_id` FK (Sprint-3-Team-Wiring), rapporte-PDF-Export (Sprint-3-Library). Subagent-Lessons: Self-Commit-Anomalie (fuhrpark), kuenftig `kein git add/commit` im Briefing erzwingen. Race-Conditions auf shared Files (config.go, gateway/main.go, docker-compose.yml) durch Edit-Tool-Konfliktdetection sauber aufgeloest.

**Welle 2B (Frontend-Hooks):** ✅ Commit `1a94503` — 12 Files, 2.904 LOC, ~70 Hooks. 4 parallele Sonnet-Subagents schrieben pro Modul je 3 Files (`<modul>-types.ts`, `<modul>-client.ts`, `hooks/use<Modul>.ts`). Pattern: TanStack Query, Stale-Time 30s (Calendar 60s, TÜV-Due 5min). Mock-Stores in `desktop/.../stores/` unangetastet — FeatureGate-Switch auf Komponenten-Layer. tsc --noEmit clean.

**Welle 2C (Bugfix-Sweep):** ✅ 4 parallele Explore-Reviews (read-only) → 27 Findings → 23 Welle-2A/2B-Bugs gefixt in Commit `a4d189e` (36 Files, +866/-124, 4 neue Migrations). 4 Findings sind Welle-1-Altlast (hardcoded Placeholder-TenantID in 7 Routes) und werden separat in einer Cross-Module-Task adressiert.

**Welle 2C Highlights:**
- **P0 Sicherheit:** fuhrpark TÜV-Cron Cross-Tenant-Lecks geschlossen (FindVehiclesDueTuev/MarkTuevReminderSent mit tenant_id), rapporte UploadAttachment ObjectKey-Path-Validierung gegen Tenant-Prefix, rapporte separate `:approve`-Permission (Migration 000100, nur admin)
- **P1 Datenintegritaet:** vermietung Migration 000101 mit `EXCLUDE USING GIST` (DB-Layer-Race-Schutz CreateRental) + UNIQUE auf rental_inspections (tenant, rental_id, kind), schichten ArbZG bidirektional (vor + nach), ApplyTemplate idempotent (Infinite-Loop-Bug behoben), Migration 000102 UNIQUE shift_assignments tenant-scoped, Migration 000103 capacity-Field + Pre-Check, rapporte Approve/Reject atomic UPDATE, rapporte GetReportStatsCounts SQL GROUP BY (statt limit=100000), fuhrpark Mileage-Decrement-Reject
- **P1 Korrektheit:** vermietung Operator-Precedence-Fix in UpdateRental, schichten ArbZG `<` (war `<=`), fuhrpark BETWEEN `::date`-Cast, vermietung Calendar filtert reserved/active, ReplacePhotos via neuem proto-bool
- **P2/P3:** fuhrpark Cron-Scan-Fenster 2h (Drift-Margin), 4 Frontend-Cache-Invalidation-Fixes, GPS-Range-Validation

**Tests:** 134 gruen. Coverage rapporte 33.9%, schichten 35.2%, fuhrpark 39.8%, vermietung 40.9% — alle ueber 30%-Ziel.

**Welle-1-Altlast (Cross-Module):** `route_{rapporte,schichten,fuhrpark,vermietung,inventar,einkauf,produktion}.go` haben hardcoded `<modul>PlaceholderTenantID = "00000000-0000-0000-0000-000000000001"` ohne JWT-Claim-Extraction. Cross-Tenant-Isolation auf HTTP-Ebene **funktioniert nicht**. Eigene Sprint-2-Task vor Pilot-1: JWT-Claim-Extraction-Refactor fuer alle 7 Module gemeinsam.

## Sprint 1 Session 2026-04-18 — R2-P0 Batch A + Wiki + Helpdesk end-to-end
| Schritt | Details |
|---------|---------|
| R2-P0.2 LiveKit-Secrets Startup-Assertion | `310c803` — Prod crasht bei devkey/devsecret |
| R2-P0.5 Egress-Webhook `egress_ended` | `d8f89d4` — setzt `recordings.status=completed` + `file_url` |
| R2-P0.6 Lexware Webhook HMAC | `787c327` — HMAC-SHA256 Verifikation auf eingehenden Webhooks |
| R2-P0.3 Recording-Consent-Bug | `efd752a` — `StartRecording` pruft Consent fuer alle Call-Teilnehmer |
| R2-P0.1 coturn-Prep (flag-off) | `a9749fa` — `livekit.yaml` Overlay + Go-TURN-Credential-Propagation (Overlay-Ansatz in Session 2026-04-19 als konzeptionell falsch revertet, siehe `deploy/turn/livekit-integration.md`) |
| R2-P0.1 coturn live | 2026-04-19 Session: Hetzner CAX11 FSN1, `turn.zentria.tech:3478`, LiveKit `use_external_ip:true` aktiv — Backend-Wiring (TURN-Credentials im AccessToken) offen, Sprint-2 S2.R2.1b |
| S1.1 Wiki Backend-Modul | `601a815` — 15 RPCs, Postgres-FTS (tsvector+GIN, deutsch), 5 Tabellen, Coverage 38.2% |
| S1.4 Helpdesk Backend-Modul | `c2d179e` — 22 RPCs, SLA-Engine + Ticket-Merge (ILIKE-Prefix), Coverage 39.3% |
| S1.1 Wiki Wiring | `75c783e` — Proto, gRPC-Server, `cmd/wiki` Binary, `route_wiki.go` hinter `modules.wiki`-Flag |
| S1.4 Helpdesk Wiring | `2d8f6d3` — Proto, gRPC-Server, `cmd/helpdesk` Binary, `route_helpdesk.go` hinter `modules.helpdesk`-Flag |
| Frontend Clients + Hooks | `eed1329` — `wiki-client.ts`/`useWiki.ts` (21 Hooks), `helpdesk-client.ts`/`useHelpdesk.ts` (28 Hooks), `useRecordingStatus` Polling |
| Gateway + Docker-Compose Activation | `0ac916c` — Registry-Register aktiv, `Dockerfile.wiki`/`Dockerfile.helpdesk`, Services in compose |
| Knowledge-Base Update | `7349ba3` — `_index.md`/`api.md`/`datenbank.md` |

**Gesamt:** 13 Commits auf main, `go build ./...` + `go test ./...` gruen. Wiki + Helpdesk default-OFF via Feature-Flags. Offen: `.env.example` uncommitted (Hook-Whitelist blockiert). TURN: coturn live seit 2026-04-19 (CAX11 FSN1), LiveKit-Wiring im video-Service bleibt Sprint-2-Task (S2.R2.1b).

## Sprint 1 Session 2026-04-19 — S1.2 Berichte Completion
3-Wellen-Subagent-Pipeline fuer die verbleibenden 5 Work-Packages (WP-3/5/6/7/11). Plan: `~/.claude/plans/sodele-was-steht-als-structured-raccoon.md`. Ports 50063/9103 (Luecke zwischen wiki und helpdesk gefuellt).
| Schritt | Details |
|---------|---------|
| WP-3 Export-Layer | `5039f79` — `internal/berichte/export/` mit PDF (maroto v2) + CSV (strings.Builder, UTF-8-BOM + Semikolon fuer DATEV) + XLSX (excelize v2.8.1). Coverage 80.2% ueber Golden-File-Tests |
| WP-5 gRPC-Server + cmd | `a742b9e` — `server/berichte_grpc.go` (14 RPCs, UUID-Validation, `mapBerichteError`), `cmd/berichte/main.go` mit Scheduler-Goroutine + Graceful-Shutdown, `Dockerfile.berichte`, Config-Felder (`BerichteGRPCPort/Address/HealthPort`). Coverage 77.6% |
| go.mod tidy | `22fe40f` — cron/v3 + excelize/v2 von indirect → direct (nach Welle 1) |
| WP-6 Gateway-Routes | `e76441a` — `gateway/route_berichte.go` mit 14 HTTP-Endpoints, `modules.berichte`-Gate, RBAC-Middleware `RequirePermission("berichte:reports", read|write)`, Export-Response via `Content-Disposition`. Coverage 57%. Migration 000080 seed_berichte_permissions |
| WP-7 Docker-Compose | `98d60c3` — `berichte`-Service-Block (dev + prod), Gateway `depends_on: berichte {service_healthy}` + `BERICHTE_GRPC_ADDRESS=berichte:50063` |
| WP-11 Final-Wire + Smoke | `a4b2cc9` — Exporter-Stub in `cmd/berichte/main.go` durch `export.NewExporter`-Adapter ersetzt, `smoke.sh` um 3 Berichte-Checks (Definitions/Run/Export-MIME) erweitert — Flag-OFF gracefully als Pass. ROADMAP S1.2 ✅ Done |

**Gesamt:** 6 Commits auf main. Gate S1.2 erfuellt. Berichte default-OFF via `modules.berichte`-Flag. Tenant-ID bleibt Placeholder `00000000-…-000001` bis JWT-Claim-Extraktion in Sprint 2 (Option-B Phase 1).

## Sprint 1 S1.PREP — Production-Redeploy (2026-04-19/20) ✅

Erster Full-Redeploy des Hetzner-CPX42 seit 2026-03-08. Server hing auf `fa17fc3` mit Dev-Secrets (`kmuhub_dev`/`devkey`) und 10 von 11 Services seit 6 Wochen als "unhealthy" markiert. Gewandert nach `980eba3` — 171 Commits, 20 Migrations (62→81), 4 neue Module live.

| Phase | Details |
|---|---|
| Deploy-Hygiene-Commit | `980eba3` — `deploy.sh` fixt: `COMPOSE_FILES_DIR`+`ENV_FILE` getrennt vom Git-Root, `--env-file` an alle Compose-Calls, Rolling-Restart-Liste um `dialer/wiki/helpdesk/berichte/formulare` erweitert. Migration `000079` via `ON CONFLICT DO NOTHING` idempotent. `ONLYOFFICE_JWT_SECRET` im PRODUCTION_TEMPLATE dokumentiert. |
| Server-Side Patches (skip-worktree) | `livekit.yaml` mit echtem LIVEKIT_API_KEY/SECRET befuellt (kein ENV-Sub moeglich). `docker-compose.yml`: 18× hardcoded `kmuhub_dev` durch `${DATABASE_URL}`/`${POSTGRES_PASSWORD}` ersetzt, alle `wget --spider` (HEAD) durch `-q -O /dev/null` (GET), `formulare`-Healthcheck auf `/healthz` umgebogen. |
| Postgres-Alignment | `ALTER USER kmuhub WITH PASSWORD <32-char>` um laufendes `docker_pgdata`-Volume mit `.env.production` zu synchronisieren. `DATABASE_URL` URL-encoded (Passwort enthielt Base64-Sonderzeichen). |
| Build-Strategie | Parallel-Bake killt sich selbst auf CPX42 (16 GB, kein Swap) — sequenzieller Build ueber 17 Services in ~10 min. |
| Migrate | 62→81 in 4.7s sauber durchgelaufen, inkl. pgvector-Extension + tenant_id-Retrofits + 4 neuer Modul-Schemas. |
| Rolling Restart | 14 Services sofort healthy nach Healthcheck-GET-Patch, `formulare` nach `/healthz`-Patch. Gateway + Caddy zuletzt. |
| Post-Deploy | `https://app.zentria.tech/health` → `{"status":"healthy","commit":"980eba3"}`, alle 15 Business-Services + LiveKit (ohne devkey-Warnung) + Infra healthy. |

**Backups (`/opt/kmuhub/backups/`):** `env_preredeploy_20260419_2122.production`, `livekit_preredeploy_20260419_2122.yaml`, `pg_dumpall_preredeploy_20260419_2122.sql` (367 KB, 10595 Zeilen, Stand Migration 61).

**7 Infrastruktur-Bugs** dokumentiert, 1 in Commit gefixt, 6 server-seitig patched, alle als Sprint-2-TODO-Liste in MEMORY `project_server_redeploy_20260419.md` und [[deployment]] verzeichnet.

**Commits:** `980eba3` (deploy.sh+migrations+template) und `3dbe057` (ROADMAP-Update).

## Abgeschlossene Meilensteine
| Meilenstein | Phasen | Abgeschlossen |
|------------|--------|--------------|
| Foundation | 1-3 | vor GSD-Adoption |
| Pilot MVP | 4-8 | 2026-02-11 |
| Compliance & Comms | 9-11 | 2026-02-17 |
| Business Suite | 12-13 | 2026-02-19 |
| Aggregation & Automation | 14-16 | 2026-02-20 |
| Integrations | 17-19 (+17.5 Guest Chat) | 2026-02-26 |
| Extensibility | 20 (Plugin System + WASM) | 2026-02-26 |

## Beta Phase A — Core Wiring (abgeschlossen)
| Schritt | Abgeschlossen |
|---------|--------------|
| 9 Core-Module auf echte API-Hooks migriert | 2026-03-05 |
| D9 Design-Merge (Waves 15-20) | 2026-03-07 |
| Lint-Cleanup (347 ESLint-Probleme auf 0) | 2026-03-07 |
| Phase A Dead-Code Audit | 2026-03-05 |

## Beta Phase B9 — Crash-Fixes (2026-04-01)
| Schritt | Details |
|---------|---------|
| MSW durch Fetch-Interceptor ersetzt | demo-mode.ts, sauberer als MSW Service Worker |
| RichTextEditor entfernt | Ungenutzte shared component |
| Business Roadmap erstellt | docs/BUSINESS-ROADMAP.md |
| ErrorBoundary: Route-Reset | ModuleShell key={location.pathname}, kein Reload nötig |
| 9 Modul-Crashes gefixt | Inventar (duplicate import), Einkauf (null guard), Formulare (null guard), Vermietung (objectName/currency), Dashboard Widgets (camelCase, activities, pipeline) |
| 5 weitere Null-Guards | CalendarUpcoming (today/dd scope), MyCalendar (now scope), EinkaufPage (showWareneingangDialog?.id), FormularePage (showShareDialog?.name, editingForm), ZustandsprotokollDialog (reservation?.objectName) |
| Projekte Mock-Daten | project_key, is_template, Handler pagination |
| Crash-Verifikation | Alle Module crash-frei verifiziert (0 JS-Errors) |

## Beta Phase B10 — Design Audit & Rebrand (2026-04-01)
| Schritt | Details |
|---------|---------|
| Design Audit (36 Screenshots) | Manuelle Screenshot-Session, alle Module visuell geprüft, Score ~6.6/10 |
| Auth Redesign | AuthLayout mit Split-Layout + Brand-Panel für Login/Register |
| Empty States normalisiert | Projekte (EmptyGeneric), Zeiterfassung (EmptyCalendar), Buchhaltung (Props gefixt) |
| Team Tabs Overflow | Fade-Mask + scrollbar-hide für 11-Tab-Leiste |
| Wiki Selection Highlight | Full-row bg → Left-border Accent |
| Kalender Headers | text-[10px] → text-xs font-medium (Wochen- und Monatsansicht) |
| Rebrand: KMU Hub → Cosmi | 36 Dateien, alle user-sichtbaren Texte |
| Locale: de-CH → de-DE | 104 Dateien, Default-Währung CHF → EUR |
| Umlaut-Normalisierung | ae/oe/ue → ä/ö/ü in ~255 Dateien (nur Display-Text, nicht Code) |

## i18n Migration Sprint (2026-04-06)
| Schritt | Details |
|---------|---------|
| Library-Migration | react-intl → i18next v26 + react-i18next v17 + i18next-icu v2 |
| Wave 1: Module | 32 Module instrumentiert (useTranslation + t() Calls) |
| Wave 1: Komponenten | 9 Komponentengruppen instrumentiert (46 Dateien) |
| Additions-JSONs | 41 JSON-Dateien in `i18n/additions/` — 4.500+ Schluessel |
| Merge-System | `mergedDE` in `i18n.ts` — alle Additions statisch in de.json integriert |
| Verbleibend (Wave 2+3) | ~47 Dateien (Settings, Integrations, Sub-Pages, Dialoge) |
| Verbleibend | Keys in de.json konsolidieren, EN/FR/IT-Uebersetzungen, Strict Types |

## Performance-Optimierung (2026-04-08)
| Phase | Inhalt | Status |
|-------|--------|--------|
| Phase 1 — Quick Wins | Bundle-Analyzer, Fonts self-host, Demo Dead Code, motion entfernt | ✅ |
| Phase 2 — Frontend | Chunk-Splitting, Async Persister, HR Polling, React Compiler, List Virtualization | ✅ |
| Phase 3 — Backend | N+1 Queries (Contact 61→4, Deal 121→7), Batch-Inserts, owner_id Index, Pool Fix, PG Tuning | ✅ |
| Phase 4 — Electron | V8 Compile Cache, modulePreload, Skeleton Screen | ✅ |
| Phase 5 — Gateway | Audit Logger Worker Pool, gRPC Keep-Alive, pprof | ✅ (Redis Caching offen) |

5 parallele Agenten (Worktree-Isolation), 6 Commits.
Detaillierter Plan: `docs/PERFORMANCE-PLAN.md`

## Dialer-Modul Phase 1 (2026-04-09)
| Sub-Phase | Inhalt | Status |
|-----------|--------|--------|
| 1A — Foundation | Proto (27 RPCs), 5 Migrations (063-067), Service-Skeleton, Docker | ✅ |
| 1B — Backend Core | service.go (24 Methoden), 4 Repos, Redis Agent-Status, CRM-Bridge, gRPC-Server | ✅ |
| 1C — Gateway + Permissions | 25 REST-Endpoints, Permission-Migration (068), route_dialer.go (1014 LoC) | ✅ |
| 1D — Frontend | 26 Dateien, DialerWorkspace (4-Phasen Call-Flow), Campaigns, Dashboard, Settings, Mock-Handler, i18n (4 Sprachen) | ✅ |
| 1E — Integration | CRM-Timeline live, Callback-Notifications, Filter-Import, Bug Fixes (ContactID, wrap_up, skip), EventEmitter, Unit/Gateway/E2E Tests | ✅ |

Strategische Roadmap: `docs/DIALER-ROADMAP.md`

## Rigorosum Runde 1 + 2 (2026-04-18)
- **Runde 1 (wild-wren):** Gesamtnote 3.3 — 7 P0-Launch-Blocker + 8 P1 + 7 P2 + 9 P3 identifiziert (Backend/Frontend/Ops)
- **Runde 2 Vertiefung (functional-seahorse):** Gesamtnote 4.1 — 9 neue P0-Blocker in Integrationen, Realtime-Kern, DB-Schema + 12 P1 + 15 P2 + 6 P3
- Kombinierte Launch-Reife **3.7** → Launch auf 2026-07-01 verschoben (+4 Wochen)
- Strategische Entscheidungen: Option-B-Full Multi-Tenancy, coturn self-hosted, Join-with-Consent, WASM Feature-Flag OFF, `finance_invoices.line_items` vor Launch normalisieren
- Details: siehe `docs/ROADMAP.md` (Single Source of Truth) und MEMORY `project_rigorosum_april.md`, `project_rigorosum_runde2.md`

## Sprint 0 — Launch-Blocker abgeraeumt (2026-04-18) ✅
Alle 7 R1-P0-Blocker + R2-P1.2 (WASM-OFF) + Cleanup + Modul-Scope-Matrix in drei parallelen Wellen.
| # | Task | PR |
|---|---|---|
| S0.1 | Migration 000075: `consent_records.contact_id` ON DELETE SET NULL | #5 |
| S0.2 | `AssertConsent`-Wrapper vor SendEmail + DialerCall (`crm/consent/`) | #10 |
| S0.3 | Prod-Secrets Startup-Assertion (`JWT`, `VAULT`, `WOPI_JWT`, `MINIO`) | #6 |
| S0.4 | DOMPurify `lib/sanitize.ts` fuer 5 Call-Sites | #9 |
| S0.5 | OnlyOffice `JWT_ENABLED: true` in Prod-Override | #7 |
| S0.6 | Feature-Flag-Registry (16 Flags, API, useFeatureFlags, FeatureGate) + WASM Build-Tag `!no_wasm` | #11 |
| S0.7 | ICU-Plural-Klammer-Fix (18 Strings × 4 Sprachen) | #3 |
| S0.8 | `mobile/`-Ordner entfernt, Pitch auf PWA | #4 |
| S0.9 | `docs/MODULES_SCOPE_MATRIX.md` (14 Module) | #8 |

Gate S0 bestanden. Sprint 1 startet 2026-04-28 mit 7 Modulen + TURN-Server + LiveKit-Secrets + Recording-Consent-Fix + Egress-Webhook + Lexware-HMAC (R2-P0 Batch A).

## Code Review Hardening (2026-04-09)
| Finding | Status |
|---------|--------|
| tenant_id auf contacts/companies (Migration 070) | ✅ |
| Desktop Tests reparieren (62/62 grün) | ✅ |
| IP Filter Fail-Close mit 5min TTL | ✅ |
| gRPC mTLS (optional, env-var-gesteuert) | ✅ |
| Gateway Bloat reduzieren (route_crm_ext.go) | ✅ |

Commits (cdaeefd–8f13465): 14 neue gRPC RPCs (11 CRM + 3 Biz), pgxpool aus Gateway entfernt, ~570 Zeilen Boilerplate reduziert, 3 große Dateien gesplittet, main.go cleanup.
Vorherige Commits (d136ea6, 96740e8, 927dbcf): HR tenant_id, SQL aus Gateway in Repos, Security/CI/Coverage Fixes.

## Verwandte Notes
- [[i18n]] — i18n-Architektur & Schluessel-Konventionen
- [[design]] — Frontend Wiring Progress
- [[architektur]] — Technischer Kontext, Performance-Patterns
