---
tags: [index]
updated: 2026-03-05
---
# KMU Hub — Knowledge Base

## Projektstand (2026-03-04)
- **Alle 20 Feature-Phasen abgeschlossen** (103/103 Plans)
- **Modus:** Beta-Vorbereitung (seit 2026-02-27)
- GitHub: github.com/Lukes-Git-Beginning/KMU-Hub (private), branch: main

### Beta Roadmap (3 Phasen, 3 Tracks)
| Phase | Zeitraum | Technisch | Legal | Business |
|-------|----------|-----------|-------|----------|
| A — Core Wiring | Maerz 2026 | CRM+Work+Kalender+Finanzen verdrahten | Anwalt, Unternehmensform | Kundengespraech, Hetzner |
| B — Beta Hardening | April 2026 | Team+Chat+Dokumente, D9 Design, E2E | AGB, DSGVO, AVV/DPA | Website, Preisliste |
| C — Beta Launch | Mai 2026 | Performance, Self-Hosted-Paket | Rechtstexte live, Impressum | Pilot-Onboarding |

### Kritische Blocker
1. Legal (AVV/DPA) = Blocker fuer Pilot-Onboarding mit echten Kundendaten
2. Produktionsserver (Hetzner) = Blocker fuer Beta-Deployment
3. D9 Design-Merge = vor erstem Pilot-Kunden-Demo

## Notes

### Kern
- [[architektur]] — Go Microservices, Frontend, Auth, CI/CD
- [[stack]] — Tech-Stack Entscheidungen und Strategy
- [[design]] — Design-System, Themes, D9 Status
- [[milestones]] — Abgeschlossene und aktive Meilensteine
- [[user-preferences]] — Entwickler-Praeferenzen und Regeln

### Technisch
- [[datenbank]] — Schema, Tabellen, Migrations, Index-Strategie
- [[api]] — OpenAPI Spec, Endpoint-Gruppen, Auth-Flow, Frontend-Integration
- [[security]] — JWT, RBAC, CORS, Rate Limiting, Vault, GDPR
- [[integrationen]] — Bexio, Lexware, DATEV, LiveKit, CalDAV, WOPI
- [[deployment]] — Docker Compose, CI/CD, Hetzner, Self-Hosted
- [[testing]] — Test-Strategie, Coverage-Ziele, Mock-Patterns
- [[troubleshooting]] — Bekannte Probleme, Lessons Learned, Dev-Umgebung
