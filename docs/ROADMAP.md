# Roadmap KMU Hub

> **Aktuelle Roadmap:** [`.planning/ROADMAP.md`](../.planning/ROADMAP.md)
>
> Diese Datei ist eine kompakte Zusammenfassung. Fuer Details, Checklisten
> und den aktuellen Fortschritt siehe die Planning-Roadmap.

---

## Ueberblick

KMU Hub durchlief zwei Entwicklungsphasen:

1. **Feature Development** (Phasen 1-20) — abgeschlossen am 2026-02-26
2. **Beta Preparation** (Phasen A-C) — aktiv seit 2026-02-27

### Feature Development (abgeschlossen)

20 Entwicklungsphasen mit 103 Plaenen, ausgefuehrt in ~11.5 Stunden.

| Meilenstein | Phasen | Abgeschlossen |
|------------|--------|--------------|
| Foundation (Auth, CRM, Chat) | 1-3 | vor GSD-Adoption |
| Pilot MVP (Gateway, Desktop, Work, Kalender, Video) | 4-8 | 2026-02-11 |
| Compliance & Comms (Security, Design, Documents) | 9-11 | 2026-02-17 |
| Business Suite (Finanzen, HR) | 12-13 | 2026-02-19 |
| Aggregation & Automation (Inbox, CalDAV, Workflows) | 14-16 | 2026-02-20 |
| Integrations (Teams/Slack, Gast-Chat, Bexio, DATEV) | 17-19 | 2026-02-26 |
| Extensibility (WASM Plugins, Industry Templates) | 20 | 2026-02-26 |

### Beta Preparation (aktiv)

Drei parallele Tracks ueber drei Phasen:

| Phase | Zeitraum | Technisch | Legal | Business |
|-------|----------|-----------|-------|----------|
| **A — Core Wiring** | Maerz 2026 | ✅ 9 Module verdrahtet | Anwalt, Unternehmensform | Kundengespraech, Hetzner |
| **B — Beta Hardening** | April 2026 | DokumentePage, D9 Design, E2E | AGB, DSGVO, AVV/DPA | Website, Preisliste |
| **C — Beta Launch** | Mai 2026 | Performance, Self-Hosted | Rechtstexte live | Pilot-Onboarding |

### Aktueller Stand (2026-03-05)

- **Phase A Track 1 (Technisch):** Abgeschlossen — alle Core-Module auf echte API-Hooks migriert
- **Phase A Track 2+3 (Legal/Business):** Noch nicht gestartet
- **Naechster Schritt:** Phase B — DokumentePage verdrahten oder D9 Design-Merge

### Scope-Entscheidung

11 Industry-Module (Einkauf, Inventar, Produktion, Vermietung, Fuhrpark, Rapporte,
Schichten, Vertraege, Wiki, Formulare, Berichte) bleiben fuer Beta auf Demo-Daten.
Diese werden erst bei konkretem Pilot-Kundenbedarf verdrahtet (Plugin-System aus Phase 20).

---

## Tech-Stack

| Komponente | Technologie |
|-----------|-------------|
| Backend | Go Microservices (5 Binaerdateien: Gateway, Auth, CRM, Chat, Notification) + Work/Biz/Automation |
| Desktop | Electron + React + TypeScript + TanStack Query |
| Datenbank | PostgreSQL (34+ Migrations) + Redis |
| Video | LiveKit (self-hostable) |
| Plugins | Config-basiert + WASM (komplexe Erweiterungen) |
| Hosting | EU-only (Hetzner), SaaS + Self-Hosted |

---

## Kritische Blocker fuer Beta

1. **Legal (AVV/DPA)** — ohne Auftragsverarbeitungsvertrag keine echten Kundendaten
2. **Produktionsserver (Hetzner)** — muss vor Deployment stehen
3. **D9 Design-Merge** — Visual Polish vor erster Pilot-Demo

---

*Letztes Update: 2026-03-05*
*Detaillierte Roadmap: `.planning/ROADMAP.md`*
