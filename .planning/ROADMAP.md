# Roadmap: KMU Hub — Beta Preparation

## Status

Alle 20 Entwicklungsphasen (103/103 Pläne) abgeschlossen am 2026-02-26.
Das Projekt wechselt von **Feature Development** → **Beta Preparation**.
Beta-Ziel: **Ende April / Mai 2026** (ca. 2-3 Monate).

## Tracks

Drei parallele Tracks laufen durch alle Beta-Phasen:

| Track | Fokus |
|-------|-------|
| **Track 1 — Technisch** | Frontend API-Wiring + QA |
| **Track 2 — Legal** | DSGVO, AVV, AGB, GoBD |
| **Track 3 — Business** | Go-to-Market, Pilot, Infrastruktur |

---

## Phase A — März 2026: "Core Wiring"

### Track 1 — Technisch

- [x] **KontaktePage** → echtes Backend via React Query (erledigt 2026-02-27)
- [ ] **FirmenPage** → useCompanies hooks (stores/crm.ts Mock entfernen)
- [ ] **DealsPage** → useDeals hooks (Pipeline-View anbinden)
- [ ] **WorkPage** → useTasks, useProjects hooks (stores/work.ts entrümpeln)
- [ ] **KalenderPage** → useEvents, useCalendar hooks (Mock bleibt für Feiertage)
- [ ] **FinanzenPage** → bereits TanStack Query, Rechnungen/Angebote finalisieren
- [ ] **Dashboard-Widgets** → echte Daten (KPI-Widgets: Finance-API, Tasks-API)

### Track 2 — Legal

- [ ] Steuerberater / Rechtsanwalt beauftragen
- [ ] Unternehmensform klären (UG/GmbH DE oder GmbH CH)
- [ ] DSGVO-Prüfung: Welche Daten verarbeiten wir, wo liegen sie?

### Track 3 — Business

- [ ] Zentrum für finanzielle Aufklärung: Konkretes Gespräch ansetzen
- [ ] Hetzner-Produktionsserver aufsetzen (Docker Compose + SSL + Domain)

---

## Phase B — April 2026: "Beta Hardening"

### Track 1 — Technisch

- [ ] **TeamPage** → useTeam hooks (stores/team.ts Mock entfernen)
- [ ] **Chat-Module** → echt (stores/kommunikation.ts + meetings.ts)
- [ ] **DokumentePage** → useDocuments hooks
- [ ] **D9 Design-Merge** → Visual Polish + Accessibility von design/brainstorm cherry-picken
- [ ] E2E-Tests für kritische Pfade (Auth, Kontakt CRUD, Rechnung erstellen)
- [ ] Fehler-Monitoring einrichten (Sentry oder äquivalent)
- [ ] Backup-Strategie Produktion (automatisch via Cron)

### Track 2 — Legal

- [ ] **AGB** — Anwalt ausarbeiten lassen
- [ ] **Datenschutzerklärung** (DSGVO Art. 13/14) — für App + Website
- [ ] **AVV / DPA** — Template für Pilot-Kunden (Auftragsverarbeitungsvertrag)
- [ ] GoBD-Dokumentation (Finanzmodul — interne Verfahrensdokumentation)

### Track 3 — Business

- [ ] Marketing-Website / Landing Page aufbauen
- [ ] Preisliste finalisieren (SaaS-Tiers + Self-Hosted-Lizenz)
- [ ] Onboarding-Prozess dokumentieren (1-Woche-Onsite-Analyse Ablauf)
- [ ] Support-Konzept (wie werden Pilot-Kunden betreut?)

---

## Phase C — Mai 2026: "Beta Launch"

### Track 1 — Technisch

- [ ] Industry-spezifische Module bleiben für v1 auf Demo-Daten (Plugin-Roadmap)
- [ ] Performance-Audit + Optimierung (Query-N+1, Bundle Size)
- [ ] Self-Hosted Docker-Compose-Paket polieren (Update-Mechanismus)
- [ ] Beta-Deployment auf Produktionsserver

### Track 2 — Legal

- [ ] Alle Rechtstexte reviewed + live auf Website/App
- [ ] Hetzner-AVV / Cloud-Provider-DPAs abschließen
- [ ] Impressum (Pflicht für Website)

### Track 3 — Business

- [ ] **Pilot-Onboarding:** Erster Kunde live schalten
- [ ] Feedback-Loop einrichten (Beta-Feedback sammeln)
- [ ] Investoren-/Partner-Deck falls relevant

---

## Scope: Mock bis v2

Diese Stores werden für den Beta-Launch **nicht verdrahtet** (bleiben Demo-Daten):

| Store | Modul | Begründung |
|-------|-------|------------|
| `einkauf.ts` | Einkauf/Beschaffung | Industry-Template (Phase 20 Plugin) |
| `inventar.ts` | Inventar | Industry-Template |
| `produktion.ts` | Produktion | Industry-Template |
| `vermietung.ts` | Vermietung | Industry-Template |
| `fuhrpark.ts` | Fuhrpark | Industry-Template (Referenz-Plugin vorhanden) |
| `rapporte.ts` | Rapporte | Industry-Template |
| `schichten.ts` | Schichten | Erst bei konkretem Kundenbedarf |
| `vertraege.ts` | Verträge | Erst bei konkretem Kundenbedarf |
| `wiki.ts` | Wiki | Erst bei konkretem Kundenbedarf |
| `formulare.ts` | Formulare | Erst bei konkretem Kundenbedarf |
| `berichte.ts` | Berichte | Erst bei konkretem Kundenbedarf |

**Strategie:** Erst wenn ein realer Pilot-Kunde ein Modul konkret braucht,
wird es vollständig verdrahtet. Qualität vor Quantität.

---

## Kritische Blocker

1. **Legal ist Blocker für Pilot-Onboarding** — ohne AVV/DPA dürfen keine echten
   Kundendaten verarbeitet werden → **sofort starten**
2. **Produktionsserver** muss vor Beta-Deployment stehen (Hetzner aufsetzen)
3. **D9 Design-Merge** bevor Pilot-Kunde die App sieht

---

## Frontend Wiring Progress

| Modul | Store/Quelle | Status |
|-------|-------------|--------|
| KontaktePage | contacts.ts | ✅ Verdrahtet (2026-02-27) |
| FirmenPage | crm.ts | ⏳ Phase A |
| DealsPage | crm.ts | ⏳ Phase A |
| WorkPage | work.ts | ⏳ Phase A |
| KalenderPage | calendar.ts | ⏳ Phase A |
| FinanzenPage | finance.ts | ⏳ Phase A (fast fertig) |
| Dashboard-Widgets | finance + work API | ⏳ Phase A |
| TeamPage | team.ts | ⏳ Phase B |
| Chat-Module | kommunikation.ts | ⏳ Phase B |
| DokumentePage | dokumente.ts | ⏳ Phase B |
| Einkauf | einkauf.ts | 🔒 v2 (Mock/Plugin) |
| Inventar | inventar.ts | 🔒 v2 (Mock/Plugin) |
| Produktion | produktion.ts | 🔒 v2 (Mock/Plugin) |
| Vermietung | vermietung.ts | 🔒 v2 (Mock/Plugin) |
| Fuhrpark | fuhrpark.ts | 🔒 v2 (Mock/Plugin) |
| Rapporte | rapporte.ts | 🔒 v2 (Mock/Plugin) |
| Schichten | schichten.ts | 🔒 v2 (on demand) |
| Verträge | vertraege.ts | 🔒 v2 (on demand) |
| Wiki | wiki.ts | 🔒 v2 (on demand) |
| Formulare | formulare.ts | 🔒 v2 (on demand) |
| Berichte | berichte.ts | 🔒 v2 (on demand) |

---

## Archiv: Feature Development (Phasen 1-20)

Alle 20 Entwicklungsphasen abgeschlossen. Details in der Git-Historie und
unter `.planning/milestones/`.

| Meilenstein | Phasen | Abgeschlossen |
|------------|--------|--------------|
| Foundation | 1-3 | vor GSD-Adoption |
| Pilot MVP | 4-8 | 2026-02-11 |
| Compliance & Comms | 9-11 | 2026-02-17 |
| Business Suite | 12-13 | 2026-02-19 |
| Aggregation & Automation | 14-16 | 2026-02-20 |
| Integrations | 17-19 | 2026-02-26 |
| Extensibility | 20 | 2026-02-26 |

---

*Phase 1-20 (Feature Development) abgeschlossen: 2026-02-26*
*Beta Preparation gestartet: 2026-02-27*
*Letztes Update: 2026-02-27*
