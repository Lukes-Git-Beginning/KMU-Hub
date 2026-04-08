---
tags: [index]
updated: 2026-04-08
---
# Cosmi — Knowledge Base

## Projektstand (2026-04-06)
- **Alle 20 Feature-Phasen abgeschlossen** (103/103 Plans)
- **Modus:** Beta Hardening (seit 2026-02-27)
- **Branding:** "KMU Hub" → "Cosmi" (Software), "Zentria" (Firma), zentria.tech
- **Phase A (Technisch):** ✅ Abgeschlossen — Core Wiring, D9 Merge, Lint 0
- **Phase B (Hardening):** B1-B8 UI Hardening + B9 Crash-Fixes + B10 Design Audit & Rebrand
- **B10 (2026-04-01):** Design Audit (36 Screenshots), Cosmi Rebrand, de-DE Locale, Umlaut-Normalisierung
- **i18n Sprint (2026-04-06):** react-intl → i18next Migration, 4.500+ Schluessel, 41 Additions-JSONs, alle 35 Module + 9 Komponentengruppen instrumentiert — Wave 2+3 + Uebersetzungen noch offen
- **Hetzner Prod:** ✅ Live, alle 10 Services healthy, Deploy-Pipeline mit Auto-Rollback
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
