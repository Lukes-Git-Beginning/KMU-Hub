---
tags: [index]
updated: 2026-05-08
---
# Cosmi — Knowledge Base

## Projektstand (2026-05-08)
- **Sprint 0 abgeschlossen:** Alle 7 R1-P0-Blocker + Cleanup + Modul-Scope-Matrix gemerged (9 PRs), Gate S0 bestanden
- **Sprint 1 abgeschlossen (2026-04-26):** 7/7 Module live + 5/5 R2-P0 Batch A done, Gate S1 bestanden
- **Sprint 2 Welle 1 done (2026-04-26):** inventar/einkauf/produktion + 20-Blocker-Bugfix-Sweep (`ad04191`) auf main
- **Sprint 2 Welle 2A+2B+2C done (2026-04-28):** Backends (`e4b1a62`+`c52839f`), Frontend-Hooks (`1a94503`), 23-Bugfix-Sweep (`a4d189e`). **7/14 Module live** mit Frontend-Hooks.
- **Sprint 2 Welle 2D done (2026-04-28 Abend):** ✅ JWT-Tenant-Hardening — `33450e7` + `c421fac` + `8f055e3`. Details: [[security]] + [[milestones]].
- **Sprint 2 Welle 3 done (2026-04-28 Spaetabend):** ✅ Drei parallele Sonnet-Subagents — `174a7e4` (Stream B: Migration 000105 idempotency_keys + Middleware WarnMode + Frontend Offline-Queue), `5d7fb0d` (Stream C: Migration 000106 Top-20 tenant_id-Retrofit, Top-5 voll gewired), `f6af609` (Stream A: Migration 000107 recording-pre-consent + Initiator-Dialog). R2-P0.4 + R2-P0.7 erledigt, Option-B Phase 1 lebendig. Details: [[milestones]].
- **Sprint 2 Welle 3.5 done (2026-04-29):** ✅ Hardening-Sweep `d443ab4` — 34 Findings closed (17 P0 + 17 P1). gRPC-Tenant-Spoof-Sweep (Context statt Proto-Feld), Repository-Tenant-Filter-Sweep (deal/activity/task/pipelinestage/chat/recording), Migration 000108 `idempotency_keys` PK → `(tenant_id, key)`, atomare Reserve, `errors.Is`-Matching, Pre-Consent-Reorder, Frontend Doppelklick-Guard + offline-queue 409-Retry. Details: [[milestones]].
- **Sprint 2 Welle 4A done (2026-04-29):** ✅ `4ff5fa2 feat(welle4a): wire repos with tenantID + idempotency client coverage` — 4 parallele Sonnet-Subagents, 109 Files. Backend: Repository-Layer-Wirings ueber automation/chat/crm/dialer/document/email/inbox/security/work-Domaenen + 10 gRPC-Handler auf `middleware.GetTenantID(ctx)` umgestellt. Frontend: neuer `api/utils/authenticatedFetch.ts`-Helper, 32 API-Clients refactored, neue 29-Case Idempotency-Coverage-Test-Suite. Build+Vet+Test+tsc+npm test gruen (202/202). Details: [[milestones]].
- **Sprint 2 Welle 4B done (2026-05-07):** ✅ Drei Sub-Wellen, 10 Sonnet-Subagents, 2 Commits `b868fb6` + `1b1eb37` — 114 Files kumuliert. 5 neue Migrations 000109-000113 (Calendar/Email/Inbox/Notification/Security/CRM-Aux/Automation-Exec/Channels-Memberships, ~49 tenant_id-Spalten + JOIN-Backfill + Idempotency partial Index). 16+ Repository-Wirings + 13 gRPC-Handler. Idempotency `Complete()`-Composite-PK-Fix + HardMode-Env-Flag (Default WarnMode, Dev-Default Hard). 12 Cross-Tenant-Tests + 3 finance JSONB-Tests + 8 P2-Followups integral. 4B.3-Sweep schloss 2 P0 (StartRecording uuid.Nil, crm_grpc 12x Spoof) + 3 P1 + integral F10. 4 P2/P3 deferred in `docs/sprint2-welle4b-followups.md`. Build+Vet+Test+tsc+npm test gruen (202/202). Details: [[milestones]].
- **Sprint 3 abgeschlossen (2026-05-08):** ✅ **8/8 Tasks done.** Welle-Tagesgesamt: 14 Direct-to-Main-Commits. **Welle 0+1 (Marathon):** 9 Hotfix-Commits zur Deploy-Infrastruktur (rollback.sh-Service-Liste, Alertmanager-Webhook-Template, serial-build OOM-Fix, **tenants-Tabellen-Bootstrap in 000114**, redis 7.4-bump, minio/mc image-rotation, healthcheck.sh dreifach gefixt) + Production-Server **Migration 81 → 115 deployed**, 32 Container healthy auf `3abec5f`. **Welle 2:** `1f6c4c0` (Dialer-Coverage 12% → 31.8% + Bonus-Fix `consent.ErrNoConsent` → `codes.PermissionDenied`) + `a8d77fc` (ansible foundation+secrets, 19+2 Tasks, 12 generierte Secrets). **Welle 3:** `71f7c90` (ansible app-deploy + Caddyfile.j2, 15 Tasks) + `562e9c5` (ansible turn + DNS-Helper, 14 Tasks). **Welle 4:** `d8f917e` ROADMAP-Closure. 4 Ansible-Roles, 50 Tasks, ansible-lint **production-profile 0 failures**. Tooling: Docker-Wrapper (`willhallonline/ansible:latest` + `MSYS_NO_PATHCONV=1`) weil Native-Windows-Ansible nicht funktioniert. Details: [[milestones]] + [[deployment]] + [[testing]] + [[troubleshooting]].
- **Launch-Datum:** **2026-07-01** (+4 Wochen nach Rigorosum Runde 2)
- **Alle 20 Feature-Phasen abgeschlossen** (103/103 Plans)
- **Branding:** "Cosmi" (Software), "Zentria" (Firma), zentria.tech
- **Naechste Schritte:** Sprint 4 ab 2026-06-08 (`finance_invoices.line_items`-Normalisierung nach ADR-0007, Skeleton in `a1a8d54` bereit, R2-P1) → Sprint 5 (Peer-Review + Rigorosum Runde 3, Launch-Freigabe). **Vor Sprint 4:** Pilot-0-IP nach Hetzner-VM-Bestellung in `deploy/ansible/inventory/hosts.yml` setzen, Smoke-Test 12/21 reparieren (RBAC-Bootstrap via `SMOKE_ADMIN_TOKEN`), `ALERT_WEBHOOK_URL` (Discord-Webhook fuer `#cosmi-prod-alerts` mit `/slack`-Suffix) auf Server-`/opt/kmuhub/.env.production` + GitHub-Secret setzen (Alertmanager stumm), CCX21-RAM-Upgrade fuer Pilot-1 evaluieren.
- **Sprint 1 Welle 1–4 (2026-04-18):** ✅ R2-P0 Batch A komplett (LiveKit-Secrets-Assertion, Egress-Webhook, Lexware-HMAC, Recording-Consent, coturn-Prep flag-off), ✅ Wiki-Modul (15 RPCs, FTS) + Helpdesk-Modul (22 RPCs, SLA + Merge) end-to-end
- **Sprint 1 Welle 5 (2026-04-18) — berichte Backend-Kern:** ✅ Migration 000079, Proto 14 RPCs, Service-Layer (52.4% Cov), Executor (92.1% Cov), Scheduler (91.5% Cov), Frontend-Client + Recharts-Page
- **Sprint 1 Welle 6 (2026-04-19) — S1.2 berichte Completion:** ✅ WP-3 Export-Layer (PDF/CSV/XLSX, 80.2% Cov), WP-5 gRPC-Server + cmd (77.6% Cov, Ports 50063/9103), WP-6 Gateway-Routes + ACL-Seed 000080 (57% Cov), WP-7 Docker-Compose, WP-11 Smoke. Gate S1.2 bestanden
- **Feature-Flag-Registry:** ✅ Live (16 Flags), siehe [[architektur]]
- **Consent-Enforcement:** ✅ Vor SendEmail + DialerCall aktiv, siehe [[security]]
- **WASM-Plugin-System:** Feature-Flag OFF bis Phase D + Build-Tag `no_wasm`, siehe [[integrationen]]
- **Mobile:** PWA auf Desktop-Basis (kein React Native mehr)
- **Hetzner Prod:** ✅ **Live auf `3abec5f` seit 2026-05-08** — alle 25 Application-Services + Caddy + Postgres + Redis (7.4) + MinIO + Prometheus + Grafana + Alertmanager healthy (32 Container). Migration-Head `115`. Welle-1-Marathon 81→115 mit 9 Hotfix-Commits (image-pinning fragility, healthcheck-Bugs, RAM-Limit). OnlyOffice unhealthy seit 2 Monaten (separater Bug, kein Pilot-Blocker). Details in [[deployment]] und [[troubleshooting]].
- GitHub: github.com/Lukes-Git-Beginning/KMU-Hub (private), branch: main

### Roadmap (Single Source of Truth: `docs/ROADMAP.md`)
| Sprint | Zeitraum | Fokus |
|---|---|---|
| S0 | 2026-04-21–04-27 | ✅ 7 R1-P0-Blocker + Modul-Scope-Matrix + Cleanup |
| S1 | 2026-04-28–05-10 | 7 Module + R2-P0 Batch A (TURN, LiveKit, Recording, Egress, Lexware) |
| S2 | 2026-05-11–05-24 | 7 weitere Module + R2-P0 Batch B + Option-B Phase 1 (Top-20 Tabellen + RLS) |
| S3 | 2026-05-25–06-07 | Option-B Phase 2 (Rest ~30 Tabellen) + Ansible + CI-Security-Scans |
| S4 | 2026-06-08–06-21 | `finance_invoices.line_items`-Normalisierung + R1-P1 + R2-P1 |
| S5 | 2026-06-22–06-30 | End-to-End, Peer-Review, Rigorosum Runde 3, Launch-Freigabe |

### Aktive Blocker
1. **Legal (AVV/DPA):** wartet auf UG-Gruendung 2026-06-01
2. **9 R2-P0-Blocker:** auf Sprint 1+2 verteilt, siehe `docs/ROADMAP.md §4`
3. **~50 Tabellen Option-B-Retrofit:** Sprint 2+3

## Notes

### Business
- [[pricing]] — Modul-x-User Preismodell (COSMI + ORBIT), Branchenpakete, Support-Stufen

### Kern
- [[architektur]] — Go Microservices, Frontend, Auth, CI/CD
- [[stack]] — Tech-Stack Entscheidungen und Strategy
- [[i18n]] — Internationalisierung (i18next), Schluessel-Konventionen, Modul-Status
- [[design]] — Design-System, Themes, D9 Status
- [[milestones]] — Abgeschlossene und aktive Meilensteine

### Technisch
- [[datenbank]] — Schema, Tabellen, Migrations, Index-Strategie
- [[api]] — OpenAPI Spec, Endpoint-Gruppen, Auth-Flow, Frontend-Integration
- [[security]] — JWT, RBAC, CORS, Rate Limiting, Vault, GDPR
- [[integrationen]] — Bexio, Lexware, DATEV, LiveKit, CalDAV, WOPI
- [[deployment]] — Docker Compose, CI/CD, Hetzner, Self-Hosted
- [[testing]] — Test-Strategie, Coverage-Ziele, Mock-Patterns
- [[troubleshooting]] — Bekannte Probleme, Lessons Learned, Dev-Umgebung
