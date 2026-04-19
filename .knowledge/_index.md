---
tags: [index]
updated: 2026-04-19
---
# Cosmi — Knowledge Base

## Projektstand (2026-04-19)
- **Sprint 0 abgeschlossen:** Alle 7 R1-P0-Blocker + Cleanup + Modul-Scope-Matrix gemerged (9 PRs), Gate S0 bestanden
- **Launch-Datum:** **2026-07-01** (+4 Wochen nach Rigorosum Runde 2)
- **Alle 20 Feature-Phasen abgeschlossen** (103/103 Plans)
- **Branding:** "Cosmi" (Software), "Zentria" (Firma), zentria.tech
- **Sprint 1 Progress:** **3/7 Module live** — wiki (S1.1), helpdesk (S1.4), berichte (S1.2). **4/5 R2-P0 Batch A done** — S1.R2.2/3/5/6 gemerged, S1.R2.1 TURN-Deploy wartet auf CPX11-Provisionierung
- **Naechste Sprint-1-Tasks:** S1.3 formulare, S1.5 vertraege, S1.6 buchhaltung-Completion, S1.7 video-Rest
- **Sprint 1 Welle 1–4 (2026-04-18):** ✅ R2-P0 Batch A komplett (LiveKit-Secrets-Assertion, Egress-Webhook, Lexware-HMAC, Recording-Consent, coturn-Prep flag-off), ✅ Wiki-Modul (15 RPCs, FTS) + Helpdesk-Modul (22 RPCs, SLA + Merge) end-to-end
- **Sprint 1 Welle 5 (2026-04-18) — berichte Backend-Kern:** ✅ Migration 000079, Proto 14 RPCs, Service-Layer (52.4% Cov), Executor (92.1% Cov), Scheduler (91.5% Cov), Frontend-Client + Recharts-Page
- **Sprint 1 Welle 6 (2026-04-19) — S1.2 berichte Completion:** ✅ WP-3 Export-Layer (PDF/CSV/XLSX, 80.2% Cov), WP-5 gRPC-Server + cmd (77.6% Cov, Ports 50063/9103), WP-6 Gateway-Routes + ACL-Seed 000080 (57% Cov), WP-7 Docker-Compose, WP-11 Smoke. Gate S1.2 bestanden
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
