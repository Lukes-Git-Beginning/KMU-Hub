---
tags: [index]
updated: 2026-04-10
---
# Cosmi — Knowledge Base

## Projektstand (2026-04-10)
- **Alle 20 Feature-Phasen abgeschlossen** (103/103 Plans)
- **Modus:** Dialer-Modul Entwicklung + Pre-Launch Sprint
- **Code Review Hardening:** ✅ Alle 5 Findings erledigt (inkl. Gateway Bloat Refactoring)
- **Branding:** "KMU Hub" → "Cosmi" (Software), "Zentria" (Firma), zentria.tech
- **Phase A (Technisch):** ✅ Abgeschlossen — Core Wiring, D9 Merge, Lint 0
- **Phase B (Hardening):** ✅ B1-B10 abgeschlossen
- **i18n Sprint:** ✅ Abgeschlossen (7.221 Keys × 4 Sprachen)
- **Performance Sprint:** ✅ Abgeschlossen (5 Phasen)
- **Dialer Phase 1A+1B:** ✅ Backend-Foundation + Core-Logik (11. Microservice, 67 Migrations, 27 RPCs)
- **Hetzner Prod:** ✅ Live, alle 11 Services healthy (inkl. Dialer), Deploy-Pipeline mit Auto-Rollback
- **Testing:** Playwright MCP via Chrome CDP (Port 9222) fuer E2E-Verifikation
- GitHub: github.com/Lukes-Git-Beginning/KMU-Hub (private), branch: main

### Beta Roadmap (3 Phasen, 3 Tracks)
| Phase | Zeitraum | Technisch | Legal | Business |
|-------|----------|-----------|-------|----------|
| A — Core Wiring | März 2026 | ✅ Abgeschlossen (API-Hooks, D9 Merge, Lint-Cleanup) | Anwalt, Unternehmensform | Kundengespräch, Hetzner |
| B — Beta Hardening | März-April 2026 | ✅ B1-B8 UI Hardening + B9 Crash-Fixes + B10 Design Audit & Rebrand | AGB, DSGVO, AVV/DPA | Website, Preisliste |
| C — Beta Launch | Mai 2026 | Performance, Self-Hosted-Paket | Rechtstexte live, Impressum | Pilot-Onboarding |

### Kritische Blocker
1. Legal (AVV/DPA) = Blocker für Pilot-Onboarding mit echten Kundendaten

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
