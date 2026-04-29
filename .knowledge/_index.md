---
tags: [index]
updated: 2026-04-29
---
# Cosmi — Knowledge Base

## Projektstand (2026-04-29)
- **Sprint 0 abgeschlossen:** Alle 7 R1-P0-Blocker + Cleanup + Modul-Scope-Matrix gemerged (9 PRs), Gate S0 bestanden
- **Sprint 1 abgeschlossen (2026-04-26):** 7/7 Module live + 5/5 R2-P0 Batch A done, Gate S1 bestanden
- **Sprint 2 Welle 1 done (2026-04-26):** inventar/einkauf/produktion + 20-Blocker-Bugfix-Sweep (`ad04191`) auf main
- **Sprint 2 Welle 2A+2B+2C done (2026-04-28):** Backends (`e4b1a62`+`c52839f`), Frontend-Hooks (`1a94503`), 23-Bugfix-Sweep (`a4d189e`). **7/14 Module live** mit Frontend-Hooks.
- **Sprint 2 Welle 2D done (2026-04-28 Abend):** ✅ JWT-Tenant-Hardening — `33450e7` + `c421fac` + `8f055e3`. Details: [[security]] + [[milestones]].
- **Sprint 2 Welle 3 done (2026-04-28 Spaetabend):** ✅ Drei parallele Sonnet-Subagents — `174a7e4` (Stream B: Migration 000105 idempotency_keys + Middleware WarnMode + Frontend Offline-Queue), `5d7fb0d` (Stream C: Migration 000106 Top-20 tenant_id-Retrofit, Top-5 voll gewired), `f6af609` (Stream A: Migration 000107 recording-pre-consent + Initiator-Dialog). R2-P0.4 + R2-P0.7 erledigt, Option-B Phase 1 lebendig. Details: [[milestones]].
- **Sprint 2 Welle 3.5 done (2026-04-29):** ✅ Hardening-Sweep `d443ab4` — 34 Findings closed (17 P0 + 17 P1). gRPC-Tenant-Spoof-Sweep (Context statt Proto-Feld), Repository-Tenant-Filter-Sweep (deal/activity/task/pipelinestage/chat/recording), Migration 000108 `idempotency_keys` PK → `(tenant_id, key)`, atomare Reserve, `errors.Is`-Matching, Pre-Consent-Reorder, Frontend Doppelklick-Guard + offline-queue 409-Retry. Details: [[milestones]].
- **Launch-Datum:** **2026-07-01** (+4 Wochen nach Rigorosum Runde 2)
- **Alle 20 Feature-Phasen abgeschlossen** (103/103 Plans)
- **Branding:** "Cosmi" (Software), "Zentria" (Firma), zentria.tech
- **Naechste Schritte:** Welle 4 (restliche 15 Top-20-Repos + Top-30+ Tabellen + Idempotency HardMode + Frontend-Idempotency-Rollout fuer alle Mutations + P2/P3-Followups aus `docs/sprint2-welle3-followups.md`)
- **Sprint 1 Welle 1–4 (2026-04-18):** ✅ R2-P0 Batch A komplett (LiveKit-Secrets-Assertion, Egress-Webhook, Lexware-HMAC, Recording-Consent, coturn-Prep flag-off), ✅ Wiki-Modul (15 RPCs, FTS) + Helpdesk-Modul (22 RPCs, SLA + Merge) end-to-end
- **Sprint 1 Welle 5 (2026-04-18) — berichte Backend-Kern:** ✅ Migration 000079, Proto 14 RPCs, Service-Layer (52.4% Cov), Executor (92.1% Cov), Scheduler (91.5% Cov), Frontend-Client + Recharts-Page
- **Sprint 1 Welle 6 (2026-04-19) — S1.2 berichte Completion:** ✅ WP-3 Export-Layer (PDF/CSV/XLSX, 80.2% Cov), WP-5 gRPC-Server + cmd (77.6% Cov, Ports 50063/9103), WP-6 Gateway-Routes + ACL-Seed 000080 (57% Cov), WP-7 Docker-Compose, WP-11 Smoke. Gate S1.2 bestanden
- **Feature-Flag-Registry:** ✅ Live (16 Flags), siehe [[architektur]]
- **Consent-Enforcement:** ✅ Vor SendEmail + DialerCall aktiv, siehe [[security]]
- **WASM-Plugin-System:** Feature-Flag OFF bis Phase D + Build-Tag `no_wasm`, siehe [[integrationen]]
- **Mobile:** PWA auf Desktop-Basis (kein React Native mehr)
- **Hetzner Prod:** ✅ **Live auf `980eba3` seit 2026-04-19** — alle 15 Business-Services healthy, LiveKit ohne devkey-Warnung, Migration-Head `81`. Erster Full-Redeploy seit 2026-03-08 (171 Commits). 6 Infra-Bugs per skip-worktree server-seitig gepatched, Sprint-2-Cleanup in [[deployment]] und [[troubleshooting]] notiert.
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
