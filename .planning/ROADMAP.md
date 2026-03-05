# Roadmap: KMU Hub — Beta Preparation

## Status

Alle 20 Entwicklungsphasen (103/103 Plaene) abgeschlossen am 2026-02-26.
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

## Phase A — Maerz 2026: "Core Wiring"

### Track 1 — Technisch

- [x] **KontaktePage** → echtes Backend via React Query (erledigt 2026-02-27)
- [x] **FirmenPage** → useCompanies hooks (erledigt 2026-02-27)
- [x] **DealsPage** → useDeals hooks, Pipeline-View (erledigt 2026-02-27)
- [x] **WorkPage** → useTasks, useProjects hooks (erledigt 2026-02-27)
- [x] **KalenderPage** → useEvents, useCalendar hooks (erledigt 2026-02-27)
- [x] **FinanzenPage** → TanStack Query finalisiert (erledigt 2026-02-27)
- [x] **Dashboard-Widgets** → echte Daten (erledigt 2026-02-27)
- [x] **TeamPage** → useEmployees/useUpdateEmployee/useCreateEmployee hooks (vorgezogen aus Phase B, erledigt 2026-03-04)
- [x] **Chat-Module (Kommunikation)** → useInboxMessages/useInboxMessage hooks (vorgezogen aus Phase B, erledigt 2026-03-04)
- [x] **Work TODOs** — mock components annotated with backend endpoint TODOs (erledigt 2026-03-05)
- [x] **Finance TODOs** — mock components annotated with backend endpoint TODOs (erledigt 2026-03-05)
- [x] **Store Cleanup** — kommunikation.ts dead code removed (537 lines mock data + 9 unused mutations) (erledigt 2026-03-05)

### Track 2 — Legal

- [ ] Steuerberater / Rechtsanwalt beauftragen
- [ ] Unternehmensform klaeren (UG/GmbH DE oder GmbH CH)
- [ ] DSGVO-Pruefung: Welche Daten verarbeiten wir, wo liegen sie?

### Track 3 — Business

- [ ] Zentrum fuer finanzielle Aufklaerung: Konkretes Gespraech ansetzen
- [ ] Hetzner-Produktionsserver aufsetzen (Docker Compose + SSL + Domain)

---

## Phase B — April 2026: "Beta Hardening"

### Track 1 — Technisch

- [ ] **DokumentePage** → useDocuments hooks
- [ ] **D9 Design-Merge** → Visual Polish + Accessibility von design/brainstorm cherry-picken
- [ ] E2E-Tests fuer kritische Pfade (Auth, Kontakt CRUD, Rechnung erstellen)
- [ ] Fehler-Monitoring einrichten (Sentry oder aequivalent)
- [ ] Backup-Strategie Produktion (automatisch via Cron)

**Remaining mock data (blocked on backend endpoints — TODOs already in code):**

| Component | Backend Needed |
|-----------|---------------|
| `work/BudgetSection.tsx` | `GET /projects/{id}/time-entries?grouped_by=person` |
| `work/AuslastungReport.tsx` | `GET /projects/{id}/team-utilization` |
| `finanzen/BankingWidget.tsx` | FinAPI integration |
| `finanzen/BelegketteTab.tsx` | `GET /finance/document-chains` |
| `finanzen/HoursToInvoiceDialog.tsx` | `GET /time-entries?billed=false` |
| `kommunikation/CannedResponsePicker.tsx` | Canned response CRUD API |

### Track 2 — Legal

- [ ] **AGB** — Anwalt ausarbeiten lassen
- [ ] **Datenschutzerklaerung** (DSGVO Art. 13/14) — fuer App + Website
- [ ] **AVV / DPA** — Template fuer Pilot-Kunden (Auftragsverarbeitungsvertrag)
- [ ] GoBD-Dokumentation (Finanzmodul — interne Verfahrensdokumentation)

### Track 3 — Business

- [ ] Marketing-Website / Landing Page aufbauen
- [ ] Preisliste finalisieren (SaaS-Tiers + Self-Hosted-Lizenz)
- [ ] Onboarding-Prozess dokumentieren (1-Woche-Onsite-Analyse Ablauf)
- [ ] Support-Konzept (wie werden Pilot-Kunden betreut?)

---

## Phase C — Mai 2026: "Beta Launch"

### Track 1 — Technisch

- [ ] Industry-spezifische Module bleiben fuer v1 auf Demo-Daten (Plugin-Roadmap)
- [ ] Performance-Audit + Optimierung (Query-N+1, Bundle Size)
- [ ] Self-Hosted Docker-Compose-Paket polieren (Update-Mechanismus)
- [ ] Beta-Deployment auf Produktionsserver

### Track 2 — Legal

- [ ] Alle Rechtstexte reviewed + live auf Website/App
- [ ] Hetzner-AVV / Cloud-Provider-DPAs abschliessen
- [ ] Impressum (Pflicht fuer Website)

### Track 3 — Business

- [ ] **Pilot-Onboarding:** Erster Kunde live schalten
- [ ] Feedback-Loop einrichten (Beta-Feedback sammeln)
- [ ] Investoren-/Partner-Deck falls relevant

---

## Scope: Mock bis v2

Diese Stores werden fuer den Beta-Launch **nicht verdrahtet** (bleiben Demo-Daten):

| Store | Modul | Begruendung |
|-------|-------|------------|
| `einkauf.ts` | Einkauf/Beschaffung | Industry-Template (Phase 20 Plugin) |
| `inventar.ts` | Inventar | Industry-Template |
| `produktion.ts` | Produktion | Industry-Template |
| `vermietung.ts` | Vermietung | Industry-Template |
| `fuhrpark.ts` | Fuhrpark | Industry-Template (Referenz-Plugin vorhanden) |
| `rapporte.ts` | Rapporte | Industry-Template |
| `schichten.ts` | Schichten | Erst bei konkretem Kundenbedarf |
| `vertraege.ts` | Vertraege | Erst bei konkretem Kundenbedarf |
| `wiki.ts` | Wiki | Erst bei konkretem Kundenbedarf |
| `formulare.ts` | Formulare | Erst bei konkretem Kundenbedarf |
| `berichte.ts` | Berichte | Erst bei konkretem Kundenbedarf |

**Strategie:** Erst wenn ein realer Pilot-Kunde ein Modul konkret braucht,
wird es vollstaendig verdrahtet. Qualitaet vor Quantitaet.

---

## Kritische Blocker

1. **Legal ist Blocker fuer Pilot-Onboarding** — ohne AVV/DPA duerfen keine echten
   Kundendaten verarbeitet werden → **sofort starten**
2. **Produktionsserver** muss vor Beta-Deployment stehen (Hetzner aufsetzen)
3. **D9 Design-Merge** bevor Pilot-Kunde die App sieht

---

## Frontend Wiring Progress

| Modul | Store/Quelle | Status |
|-------|-------------|--------|
| KontaktePage | contacts.ts | ✅ Verdrahtet (2026-02-27) |
| FirmenPage | crm.ts | ✅ Verdrahtet (2026-02-27) |
| DealsPage | crm.ts | ✅ Verdrahtet (2026-02-27) |
| WorkPage | work.ts | ✅ Verdrahtet (2026-02-27) |
| KalenderPage | calendar.ts | ✅ Verdrahtet (2026-02-27) |
| FinanzenPage | finance.ts | ✅ Verdrahtet (2026-02-27) |
| Dashboard-Widgets | finance + work API | ✅ Verdrahtet (2026-02-27) |
| TeamPage | team.ts → hr-hooks.ts | ✅ Verdrahtet (2026-03-04) |
| Chat-Module | kommunikation.ts → inbox API | ✅ Verdrahtet (2026-03-04) |
| DokumentePage | dokumente.ts | ⏳ Phase B |
| Einkauf | einkauf.ts | 🔒 v2 (Mock/Plugin) |
| Inventar | inventar.ts | 🔒 v2 (Mock/Plugin) |
| Produktion | produktion.ts | 🔒 v2 (Mock/Plugin) |
| Vermietung | vermietung.ts | 🔒 v2 (Mock/Plugin) |
| Fuhrpark | fuhrpark.ts | 🔒 v2 (Mock/Plugin) |
| Rapporte | rapporte.ts | 🔒 v2 (Mock/Plugin) |
| Schichten | schichten.ts | 🔒 v2 (on demand) |
| Vertraege | vertraege.ts | 🔒 v2 (on demand) |
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
*Letztes Update: 2026-03-05*
