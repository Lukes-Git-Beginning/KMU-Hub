---
tags: [index]
updated: 2026-04-18
---
# Cosmi — Knowledge Base

## Projektstand (2026-04-18)
- **Sprint 0 abgeschlossen:** Alle 7 R1-P0-Blocker + Cleanup + Modul-Scope-Matrix gemerged (9 PRs), Gate S0 bestanden
- **Launch-Datum:** **2026-07-01** (+4 Wochen nach Rigorosum Runde 2)
- **Alle 20 Feature-Phasen abgeschlossen** (103/103 Plans)
- **Branding:** "Cosmi" (Software), "Zentria" (Firma), zentria.tech
- **Sprint 1 (startet 2026-04-28):** 7 Module (wiki, berichte, formulare, helpdesk, vertraege, buchhaltung-Completion, video-Completion) + R2-P0 Batch A (TURN self-hosted, LiveKit-Secrets, Recording-Consent, Egress-Webhook, Lexware-HMAC)
- **Sprint 1 Welle 1–4 (2026-04-18):** ✅ R2-P0 Batch A komplett (LiveKit-Secrets-Assertion, Egress-Webhook, Lexware-HMAC, Recording-Consent, coturn-Prep flag-off), ✅ Wiki-Modul (15 RPCs, FTS) + Helpdesk-Modul (22 RPCs, SLA + Merge) end-to-end (Migration + Domain + Proto + gRPC + Gateway-Route + FE-Client/Hooks + Dockerfile + docker-compose)
- **Sprint 1 Welle 5 (2026-04-18) — berichte Backend-Kern:** ✅ Migration 000079 (4 Tabellen + 8 System-Seeds), Proto 14 RPCs, `backend/internal/berichte/` Service-Layer (52.4% Cov), `executor/` mit 8 kind-handlers + DashboardKPIs (92.1% Cov), `scheduler/` mit cron-dispatch + atomic claim + nil-tolerant Exporter/Mailer (91.5% Cov). Offen: Exporter (PDF/CSV/XLSX), gRPC-Server, Gateway-Routes, Docker, Recharts-Page-Migration.
- **Feature-Flag-Registry:** ✅ Live (16 Flags), siehe [[architektur]]
- **Consent-Enforcement:** ✅ Vor SendEmail + DialerCall aktiv, siehe [[security]]
- **WASM-Plugin-System:** Feature-Flag OFF bis Phase D + Build-Tag `no_wasm`, siehe [[integrationen]]
- **Mobile:** PWA auf Desktop-Basis (kein React Native mehr)
- **Hetzner Prod:** ✅ Live, alle 11 Services healthy, Deploy-Pipeline mit Auto-Rollback
- **Testing:** Playwright MCP via Chrome CDP (Port 9222) fuer E2E-Verifikation
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
