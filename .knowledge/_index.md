---
tags: [index]
updated: 2026-03-26
---
# KMU Hub — Knowledge Base

## Projektstand (2026-03-26)
- **Alle 20 Feature-Phasen abgeschlossen** (103/103 Plans)
- **Modus:** Beta Hardening (seit 2026-02-27)
- **Phase A (Technisch):** ✅ Abgeschlossen — Core Wiring, D9 Merge, Lint 0
- **Phase B (Hardening):** B1-B7 abgeschlossen + B8 UI Hardening (Audit Score 12→17.5/20)
- **Hetzner Prod:** ✅ Live, alle 10 Services healthy, Deploy-Pipeline mit Auto-Rollback
- **GSD Workflow:** Entfernt (2026-03-26) — Commands, Hooks, Settings bereinigt
- GitHub: github.com/Lukes-Git-Beginning/KMU-Hub (private), branch: main

### Beta Roadmap (3 Phasen, 3 Tracks)
| Phase | Zeitraum | Technisch | Legal | Business |
|-------|----------|-----------|-------|----------|
| A — Core Wiring | Maerz 2026 | ✅ Abgeschlossen (API-Hooks, D9 Merge, Lint-Cleanup) | Anwalt, Unternehmensform | Kundengespraech, Hetzner |
| B — Beta Hardening | Maerz-April 2026 | ✅ B1-B7 + B8 UI Hardening (A11y, Perf, Theming, Audit 17.5/20) | AGB, DSGVO, AVV/DPA | Website, Preisliste |
| C — Beta Launch | Mai 2026 | Performance, Self-Hosted-Paket | Rechtstexte live, Impressum | Pilot-Onboarding |

### Kritische Blocker
1. Legal (AVV/DPA) = Blocker fuer Pilot-Onboarding mit echten Kundendaten

## Notes

### Kern
- [[architektur]] — Go Microservices, Frontend, Auth, CI/CD
- [[stack]] — Tech-Stack Entscheidungen und Strategy
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
