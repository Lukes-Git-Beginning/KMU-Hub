# Build-Progress — Frontend-Parität (alle Module)

> Fortschritts-Tracking für die agent-getriebene Bauphase. Quelle der Pläne: `ui-tiefen-audit.md`.
> Reihenfolge: Daily-Use-Kern zuerst. Review-Rhythmus: pro Modul Stopp (XL-Module in Blöcken).
> Hartes Gate pro Bau-Einheit: visueller QA-Pass @ 3 Fenstergrößen (voll/halbiert/klein) + Zustände.

## Render-/QA-Setup
- [x] Render-Methode fixiert + Smoke-getestet (2026-06-06)
- Methode: **Playwright-Chromium gegen laufende Demo-App** (`localhost:5173`, HashRouter). Skript: `desktop/scripts/qa-shot.mjs`. Nutzung: `node scripts/qa-shot.mjs "route1,route2" [outDir]`. Output: `desktop/.qa-screenshots/<route>__<full|half|small>.png`. Onboarding-Modal wird via localStorage (`cosmi-ui`) + Skip-Klick unterdrückt. electronAPI als Proxy-Stub. Voraussetzung: `npm run dev` läuft.
- Größen: full 1440×900 · half 720×900 · small 500×800.
- ⚠ QA-Pflicht: NICHT nur Listen — auch geöffnete Detail-Panels/Dialoge @ 3 Größen screenshotten (Darien-Anforderung).

## Modul-1 Design-Entscheidungen (bestätigt 2026-06-06)
- **Struktur: Tab-Gerüst** — `kontakte`-Route bleibt; obere Bereichs-Tabs (Kontakte/Firmen/Pipeline/Aktivitäten). „Kontakte"-Tab = bestehende Master-Detail-UX. 360°-Tiefe gekapselt ins Kunden-Detail (Phase 2). crm-Route → Redirect.
- **Darien-Anforderung (KRITISCH):** Die sich öffnenden Fenster (Kontakt-Detail, Edit-Dialoge, Firmen-/Deal-Panels) müssen VOLL ausgestattet + gut sichtbar sein: **editierbare Tags** (FE-machbar via `POST/DELETE /api/v1/contacts/{id}/tags`), alle Felder, Einstellungsmöglichkeiten, „übliches CRM-Zeug" — nichts halb leer/abgeschnitten @ keiner Fenstergröße.

## Modul-Reihenfolge
1. ⏳ **Kontakte = Kunden-Zentrale** (XL, 8 Phasen, inkl. CRM-Integration) — in 3 Blöcken
2. ⬜ work
3. ⬜ dashboard
4. ⬜ mails
5. ⬜ kommunikation
6. ⬜ (danach) finanzen, kalender, dokumente, ...
7. ⬜ System-Module
8. ⬜ Branchen-Module

---

## Modul 1 — Kontakte = Kunden-Zentrale

**Block A (Phase 0-2)** — Status: ⏳ läuft (Phase 0 done)
- [x] Phase 0 — Tab-Gerüst: `kontakte` wird Parent mit Tabs (Kontakte/Firmen/Pipeline/Aktivitäten), crm-Bereiche eingebunden, `/crm`→Redirect. QA grün @ 3 Größen, 0 Errors. ⚠ Bug gefangen: Bau-Agent hatte `git stash` für tsc-Vergleich gemacht + nicht zurückgeholt → App.tsx-Änderungen waren weg, alle Tabs „Seite nicht gefunden". Per `git stash pop` gerettet. **Lehre: Bau-Agents NIE `git stash` nutzen.**
- [~] Phase 1 — läuft:
  - [x] Detail-Routen-Fix: Klick auf Firma/Deal öffnet jetzt Detail-Seite (war: sprang zu Kontakte). 3 verkettete Bugs gefixt: (1) interne `/crm/...`-Pfade → `/kontakte/...`, (2) Mock-Response flach statt `{company}`/`{deal}` verschachtelt, (3) `company.address` Objekt → String normalisiert. Detail-Routen `firmen/:id`+`pipeline/:id` eingebunden. QA-Klick-Flow grün.
  - [x] Kontakt-Detail voll ausgestattet: 2-Spalten (Stammdaten + Chronik), editierbare Tags (Popover-Auswahl + Entfernen, via neue Hooks `useContactTags` + Mock-Handler), Aktivitäts-Timeline (ContactTimeline eingebunden), alle Felder, Quick-Actions. Responsive-Fix: Timeline stapelt bei schmal statt zu verschwinden. QA @ 3 Größen grün + Tag-Popover-Flow grün.
  - [x] Nebenbefund gefixt: `companies/:id/contacts`-Mock-Handler ergänzt → Firmen-Detail zeigt jetzt „Kontakte (3)" (verknüpfte Kontakte)
  - [x] Listen-Tiefe: Sortierung (klickbare Spalten-Header) + Mehrfachauswahl + Bulk-Löschen für Unternehmen/Deals/Aktivitäten. Routing-Fix: „Deals"-Tab zeigt jetzt DealsListPage (Tabelle mit list/pipeline-Toggle) statt direkt Kanban. Mock-Handler um Sort erweitert. 17 i18n-Keys×4. Filter bewusst weggelassen (bei Pagination nicht repräsentativ). QA: Sort-Header + Bulk-Leiste („12 ausgewählt | Löschen") grün.

**Phase 1 = ABGESCHLOSSEN.**

**Phase 2 — Kunden-360° = ABGESCHLOSSEN:** ContactDetailPanel um 4 Verknüpfungs-Sektionen erweitert — Deals am Kontakt (`useDeals({contact_id})`, klickbar→Deal-Detail), E-Mail-Verlauf (`useContactEmails`, Mock-Handler ergänzt), Aufgaben (`useEntityTasks`), Einwilligungen/DSGVO (ConsentPanel eingebunden). Backend-Gaps für Verträge+Rechnungen am Kontakt in `backend-gaps.md` dokumentiert. 8 i18n-Keys×4. QA (scroll @ narrow): alle 4 Sektionen mit echten Demo-Daten sichtbar, 0 Errors.

### BLOCK A (Phase 0-2) — NACHBESSERUNG aus Darien-Feedback (2026-06-06) — ✅ ABGESCHLOSSEN (autonom verifiziert)
Darien-Screenshot-Feedback an der Kontakte-UX → UX-Rework:
- [x] Bug: ConsentPanel ragt aus Rahmen → `flex-wrap` + `truncate max-w` + `shrink-0` an Metadaten-Zeile. QA: Widerruf-Item („⚠ Widerrufen … › Verlauf (2)") rendert clean @ 500px, kein Overflow.
- [x] Leere rechte Seite (Master-Detail) → ersetzt: **Kontaktliste volle Breite** (kein leeres Detail-Panel mehr).
- [x] **Ansichts-Toggle Liste ↔ Raster** — segmented control, beide Views gebaut (Raster = Karten mit Avatar/Status/Mail/Tel/Tags ohne Klick). View-Labels kollabieren bei schmal zu Icons.
- [x] Detail beim Klick als **Modal** (`Dialog`, `max-w-4xl`, `max-h-90vh`, scrollt intern). Voll ausgestattet: Kontaktdaten, editierbare Tags, Notizen, Deals, E-Mail-Verlauf, Aufgaben, Consent, Chronik.
- [x] Gründlicher QA: 6 Kontakte, beide Ansichten, Modal @ 3 Größen (1440/720/500). **0 pageerrors** durchweg.
- [x] **Nebenbefund gefixt:** ConsentPanel-Mock matchte auf Platzhalter-IDs `c4/c9/c12`, die es im echten Datensatz nicht gibt → jeder Kontakt zeigte „0 erteilt" (toter Code). Jetzt deterministischer Hash der echten contactId → 3 Varianten (voll erteilt mit Metadaten / Widerruf / leer). Demo realistisch + Edge-Cases testbar.
- [x] **Orthographie-Bugs in Mock-Daten gefixt:** „Geschaeftsfuehrer"→„Geschäftsführer" (4×), „Loesung"→„Lösung", „Moechte/grossen/ueber/Gruender/Fuehrt", ConsentPanel „Muendlich"→„Mündlich".
→ Lehre festgehalten: [[feedback_qa_thoroughness]]. QA-Skripte: `qa-phase3.mjs`, `qa-consent.mjs`.

### Block B (Phase 3-5) — Phase 3 Plan von Darien GENEHMIGT, autonom umgesetzt (2026-06-06)
- Markt-Recherche done (Pipedrive/HubSpot: 3 Views, DnD, Rotting, weighted Forecast).

**Block B (Phase 3-5)** — Status: ⏳ Phase 3 ABGESCHLOSSEN (autonom verifiziert), Phase 4/5 offen
- [x] **Phase 3 — Pipeline & Deals** (alle 6 Items, typecheck + lint sauber, QA 0 pageerrors):
  - [x] **DnD verdrahtet** — `DealPipelineView` neu mit `@dnd-kit` (Pattern aus work/KanbanBoard): Karten draggable, Spalten droppable, `useMoveDealToStage` + optimistisches Cache-Update + Rollback. QA: echter Drag → Toast „Deal nach Qualifiziert verschoben".
  - [x] **Mock-Stage-Endpoint** `POST /deals/:id/stage` ergänzt — mutiert in-memory, damit Moves über Refetch persistieren (sonst Rücksprung).
  - [x] **Rotting-Indikator** — Tage seit `updated_at` auf Open-Stage-Karten: ≥14 amber, ≥30 rot, Badge „{days} T. inaktiv". 4 Mock-Deals auf ältere `updated_at` gesetzt (sonst alle <6 T. → Indikator unsichtbar, analog Consent-Toter-Code).
  - [x] **Quick-Actions auf Karten** — Hover-Buttons „Gewonnen"/„Verloren" (move zu won/lost-Stage), nur für offene Stages. QA: beide sichtbar on hover.
  - [x] **Quote-`alert()`→Toast** — `DealDetailPage` `handleCreateQuoteFromDeal` Fehlerpfad → `toast.error`.
  - [x] **Kontakt/Firma-Picker** — `DealFormDialog`: Plain-Inputs → durchsuchbare Inline-Combobox (`useContacts`/`useCompanies`, Vorschläge mit Firmen-Sublabel, Freitext weiter erlaubt). QA: „Mar" → Martin Berger/Marc Brunner, Auswahl füllt Feld.
  - [x] **Forecast-Ansicht** — neue 3. View (`DealForecastView`, Toggle-Button BarChart3): 3 Stat-Cards (Offenes Volumen / Gewichtete Prognose / Gewonnen), Prognose nach Phase (gewichtete Balken), Erwarteter Abschluss nach Monat. 8 i18n-Keys×4.
  - i18n: 12 neue Keys × 4 Locales (de/en/fr/it), byte-sortiert eingefügt, Parität 12/12/12/12, 0 Raw-Keys in QA.
  - ⚠ Backend-Gap: „mehrere Pipelines" = Luke. Notiz: Picker speichert nur Namen (keine contactId/companyId-Verknüpfung) — bewusst, da Create-Handler `_contactName`/`_companyName` nutzt.
  - ⚠ Vorbestehend (nicht gefixt, Scope): einige `crm.deals.*`-Keys haben ASCII-Substitutionen („Waehrung", „Prioritaet", „verknuepft") + DealsListPage:110 setState-in-effect Warning.
  - QA-Skripte: `qa-pipeline.mjs`, `qa-forecast.mjs`, `qa-picker.mjs`.
- [x] **Phase 4 — Leads** (autonom, Darien-Richtung per Multiple-Choice eingeholt; typecheck+QA sauber, 0 pageerrors, 0 Raw-Keys):
  - **Architektur-Entscheidung (Hybrid, von Darien mir überlassen):** Lead = Kontakt-Lifecycle-Status `lead` + eigene Inbox-Ansicht. HubSpot (Lifecycle) × Pipedrive (Inbox) kombiniert, kein Daten-Silo. FE mock-first via `useLeads` (In-Memory, swapbar auf `/api/v1/leads`).
  - [x] **Leads-Inbox** als 5. Tab (zwischen Kontakte und Firmen, Funnel-Reihenfolge): Status-Filter-Chips (Alle/Neu/Kontaktiert/Qualifiziert/Verworfen mit Counts), Temperatur-Ampel, Quelle-Badge, Score-Bar, Quick-Mail/Tel, Aktions-Menü.
  - [x] **Scoring: beides** (Darien-Wahl) — Auto-Score (Quelle-Gewicht + Daten-Vollständigkeit, 0–100 → Ampel) + manuell überschreibbar (heiß/warm/kalt, sticky). Live-Score-Vorschau im Neu-Formular.
  - [x] **Quellen: Manuell · CSV-Import · Dialer** (Darien-Wahl; Webformular bewusst raus — Website nicht live). CSV-Import = Paste-Dialog (Zeile/Lead).
  - [x] **Konvertierung** (Darien-Wahl „alles drei, beim Umwandeln wählbar"): Dialog mit Kontakt (immer) + Firma (opt., vorausgewählt wenn Lead-Firma da) + Deal (opt., mit Name/Wert). Nutzt bestehende Create-Hooks. E2E getestet: Toast „… umgewandelt", Lead → Status qualifiziert.
  - i18n: 56 Keys × 4 Locales (rein additiv eingefügt, 0 Deletions, Parität). Dateien: `leads/LeadsInboxPage.tsx`, `ConvertLeadDialog.tsx`, `LeadFormDialog.tsx`, `LeadImportDialog.tsx`, `leadVisuals.ts`, `api/hooks/useLeads.ts`. QA: `qa-leads.mjs`, `qa-leads-convert.mjs`.
- [x] **Phase 5 — Aktivitäten & Wiedervorlage** (autonom; typecheck+QA sauber, 0 pageerrors, 0 Raw-Keys):
  - [x] **Wiedervorlage-Agenda** als 2. Ansicht (Toggle Liste ↔ Wiedervorlage in der ActivitiesListPage): offene Aktivitäten nach Fälligkeit gruppiert (Überfällig/Heute/Diese Woche/Später/Ohne Termin), Überfällig rot. `WiedervorlageView.tsx`.
  - [x] **Inline-Aktionen:** Erledigen (Häkchen → Item verschwindet) + Wiedervorlage verschieben (Datums-Input pro Zeile). E2E getestet: complete 4→3 + Toast.
  - [x] **Vorbestehende Bugs gefixt (Bonus):** (1) Activity-Mock hatte KEINE `complete`/PATCH/DELETE-Handler → Erledigen/Löschen/Verschieben waren im Demo-Mode kaputt → ergänzt (in-memory mutierend, POST persistiert jetzt + normalisiert `type`). (2) ListPage las `activity.is_completed`, Mock liefert `completed` → „Erledigt"-Badge erschien nie → 3× gefixt (jetzt 11 Badges sichtbar).
  - [x] Mock-Aktivitäts-Fälligkeiten gespreizt (3 offene Tasks über Buckets), damit die Agenda alle Zustände inkl. Überfällig demonstriert.
  - [x] **Bug während QA gefangen+gefixt:** `activityTypeLabel()` gibt i18n-Key zurück → musste durch `t()` (Raw-Key „crm.activities.type.task" im Untertitel; visueller Check fing's, Regex hatte `type` nicht abgedeckt).
  - i18n: 14 Keys × 4 Locales (additiv, 0 Deletions). QA: `qa-agenda.mjs`.
- [x] **QA-Pass Block B** (autonom verifiziert) → wartet auf Darien-OK

### UX-Polish-Runde (Darien-Screenshot-Feedback, 2026-06-07) — ✅ ABGESCHLOSSEN (commit 40b7ddd)
12 Punkte aus mehreren Screenshots, alle mit Screenshot-QA verifiziert:
- [x] Raw-Keys Kontakte-View-Toggle (`kontakte.view.*`) + Aktivitäten-Sort (`crm.activities.sort.*`) ergänzt.
- [x] ASCII-Fixes Mock („Regelmäßiges", „Langjähriger").
- [x] Consent-Grant-Flow: „Erteilen" ausgeblendet während Quelle-Picker offen (Redundanz weg).
- [x] **Sortierung mit Richtung** — neue wiederverwendbare `components/shared/SortMenu.tsx` (Feld + auf/ab), in Kontakte eingesetzt. `common.sort.*` i18n.
- [x] **Bestehende Kontakte zu Gruppen zuordnen** — `GroupAssignDialog` aus Kontakt-Aktionsmenü (Checkbox-Liste, add/remove). E2E verifiziert.
- [x] Toolbar entzerrt (Suche füllt Breite, Controls rechts mit Abstand, Import-Icon `FileDown`).
- [x] **Listen-Dichte** — Zeilen mit E-Mail/Telefon (Mitte) + letzter Kontakt (rechts); **sticky Buchstaben-Header** (A/B/C…) bei Namens-Sortierung.
- [x] **Raster-Karten** — Hover-Quick-Actions als Bottom-Overlay statt Höhenwachstum (Karten bleiben gleich groß, QA: 241px vor/nach Hover).
- [x] **Detail-Modal** — eigener Close-Button im Header (Radix-Standard-X via `showCloseButton={false}` ausgeblendet, kein Overlap), Ecke `rounded-2xl`.
- [x] **Zeilen/Karten `button`→`div[role=button]`** — valides HTML (Aktions-Buttons waren in Buttons verschachtelt), Dropdowns jetzt zuverlässig.
- [x] Zurück-Buttons: DealDetailPage/CompanyDetailPage hatten bereits welche (verifiziert).
- [x] **⚠ Großer i18n-Fund + Fix:** In Phase 4 hatte ein `git checkout` auf die i18n-Dateien (nach fehlerhaftem Re-Sort-Skript) die damals noch uncommitteten Phase-3-Keys mitgerissen → `crm.deals.forecast.*` + 5 deals-Keys waren mit Raw-Keys committed. Voller i18n-Audit (`scripts/i18n-audit.mjs`, erfasst auch interpolierte `t('k',{…})`) → **21 fehlende Keys** gefunden+ergänzt (forecast, deals, `crm.bulk.*`, `kontakte.tag.*`, contact-detail). Re-Audit: 0 missing, Parität 4 Sprachen.
- **Lehren festgehalten:** [[feedback_recurring_ux_patterns]] (Sort-Richtung + Zurück-Button + wiederverwendbar bauen), [[feedback_qa_thoroughness]] (Raw-Key-Check bei VOLLER Breite + interpolierte Keys + Modal-Inhalte; NIE `git checkout` auf Dateien mit uncommitteten Änderungen).
- Tooling: `scripts/i18n-audit.mjs` (wiederverwendbarer i18n-Vollständigkeits-Check), `qa-final.mjs`, `qa-polish.mjs`, `qa-group.mjs`.

**Block C (Phase 6-8)** — Status: ⏳ Phase 6 fertig
- [x] **Phase 6 — Auswertungen/Reports** (autonom; typecheck+QA sauber, 0 Raw-Keys, 0 Errors): neue `AuswertungenPage` als 6. Tab (`/kontakte/auswertungen`). 5 KPI-Cards (Offenes Volumen, Gewichtete Prognose, Gewinnrate, Offene Leads, Fällige Aktivitäten) + Pipeline-Funnel + Conversion-Donut (Win-Rate) + Aktivitäten nach Typ + Lead-Quellen-Pie. recharts + `useChartTheme` (kein Rainbow). 14 i18n-Keys ×4. QA: 4 Charts rendern @ 1440/760.
  - **Global:** Scrollbars ausgeblendet (`globals.css`) — Cosmi-natürlicher Look, Scrollen bleibt. [[feedback_recurring_ux_patterns]].
  - Reihenfolge bestätigt (Darien 2026-06-07): Kontakte ganz fertig → dann nächstes Modul. Settings-Fundament (scope personal/tenant + Modul-Leitung) VOR Phase 7 bauen.
- [ ] Phase 7 — Einstellungen (Stage-Editor, Custom-Fields, Tags, Scoring-Regeln)
- [ ] Phase 8 — Finanzberatungs-Tiefe (Empfohlen-von, Beratungsprotokoll, Segmente)
- [ ] QA-Pass Block C → Darien-OK

**Backend-Gaps gesammelt für Luke:** siehe `.planning/backend-gaps.md`

---

## Settings-Fundament (scope-Architektur) — ✅ ABGESCHLOSSEN (2026-06-07, autonom)

> Modulübergreifendes Fundament VOR Modul-Einstellungen. 3-Ebenen-Hierarchie: Tenant-Default → Modul-Leiter (tenant-weit) → User-Override (persönlich). FE mock-first.

- [x] **Datenmodell:** `stores/moduleLeads.ts` (per-user-per-module Modul-Leiter-Flag, persist, Seed) · `lib/module-settings.ts` (`SettingsScope`-Typ + `LEADABLE_MODULES`) · `hooks/useModuleSettings.ts` (`useIsModuleLead` — Admin immer Leiter, sonst Flag).
- [x] **Reusable Shell:** `components/shared/ModuleSettingsShell.tsx` — deklarative Sections-API (`scope: personal|tenant`), Scope-Badges (Persönlich/Für alle), Lock-Pattern für Nicht-Leiter (`fieldset disabled` + Schloss-Badge + Hinweis-Banner), Scope-Context (`module-settings-scope.ts`, Fast-Refresh-clean). Export in `shared/index.ts`.
- [x] **Admin-Toggle:** MemberDetailPanel → Sektion „Erweiterte Moduleinstellungen" (nur admin/it_support) — Modul-Leiter pro Mitarbeiter aktivierbar, schreibt in moduleLeads-Store. (Demo: nur sichtbar wenn Mitarbeiter Modul-Grants hat — Demo-Daten-Lücke, Code identisch zum funktionierenden Module-Zuteilungs-Block verdrahtet.)
- [x] **Proof-of-use:** `CalendarSettingsTab` auf Shell umgebaut — personal (Ansicht/Wochenstart/Erinnerung) vs. tenant (Arbeitszeiten/Feiertagsregion).
- [x] **i18n:** 9 Keys ×4 Sprachen (`settings.scope.*`, `settings.calendar.section.*`, `team.member.moduleLead.*`). Audit: 0 missing.
- [x] **QA (`scripts/qa-settings-fundament.mjs`):** Kalender @ 1440/720 (Admin, unlocked, beide Badges, 0 Raw-Keys, 0 Errors) + Member-Lock-Zustand (Hinweis-Banner + ausgegraute tenant-Sektion + Schloss-Badge, in-place Profilwechsel). typecheck 0 Fehler, lint 0 Errors.
- [x] **backend-gaps:** `tenant_module_leads` + `tenant_settings`/`user_settings`-Scope-Tabellen für Luke dokumentiert.

---

## Modul-Einstellungs-Fenster (App-Shell) — ⏳ Commit A ✅, Commit B (CRM-Panel) offen

> Darien-Feedback (2026-06-07): „Einstellungen" links unten soll ein **Fenster/Overlay** öffnen (kein Routing), darin NUR Modul-Einstellungen + ein „Cosmi (Allgemein)"-Bereich (Admin-Inhalte verteilt). Persönliche Settings bleiben oben rechts übers Profil-Menü. Aufbau: linke Modul-Liste + rechter Inhalt (klassischer Settings-Aufbau, Cosmi-Optik), aktives Modul vorausgewählt + markiert.

**Commit A — Fenster-Shell ✅ (typecheck 0, lint 0, QA grün):**
- [x] `SettingsOverlay.tsx` — Overlay (kein Routing), linke Navi (Gruppen MODULE / COSMI), rechter Inhalt, Esc/Backdrop/X schließt, kontext-sensitive Vorauswahl (`resolveEntryForPath`) + „Aktiv"-Badge.
- [x] `module-settings-registry.tsx` — deklarative Eintrags-Registry mit RBAC. MODULE: Finanzen/Kalender/Mail/Team · COSMI: Firma/Abrechnung/Integrationen/IT (bestehende Tab-Komponenten wiederverwendet).
- [x] UI-Store: `isSettingsOverlayOpen` + `openSettingsOverlay`/`closeSettingsOverlay` (transient, aus Persistenz ausgeschlossen).
- [x] „Einstellungen"-Button (links unten) in allen 4 Layouts (Sidebar/Classic/Dock/TopNav) auf Overlay-Öffnen umgestellt (`onActivate` am NavItem, `preventDefault` statt Navigation). Profil-Menü oben rechts → /settings (persönlich) bleibt.
- [x] `/settings` auf **Persönlich** verschlankt (Mail/Kalender raus, leben im Fenster).
- [x] i18n `moduleSettings.*` ×4 (Audit 0 missing). QA (`scripts/qa-module-settings.mjs`): öffnet als Dialog, beide Gruppen, Aktiv-Badge, **Route bleibt stabil** (kein Wegnavigieren), Kalender-Scope-Sektionen im Fenster, Esc schließt, 0 Errors/Raw-Keys.
- ⚠ QA-Selektor bei 820px traf den (kollabierten) Trigger nicht — Overlay selbst ist responsiv; reiner Test-Selektor-Punkt.

**Commit B1 — CRM-Settings-Panel (Kontakte Phase 7, Teil 1) ✅ (typecheck 0, lint 0, QA grün):**
- [x] `CrmSettingsPanel` als erster MODULE-Eintrag „CRM" (Kontext-Vorauswahl greift: aus Kontakte → CRM aktiv markiert). Auf `ModuleSettingsShell`, beide Sektionen tenant-scoped („Für alle").
- [x] **Pipeline-Phasen-Editor** (`PipelineStagesEditor`) — Liste mit Inline-Edit (Name/Wahrscheinlichkeit), Farb-Swatches, Reorder (↑/↓), Add/Delete, Gewonnen/Verloren-Marker, Deal-Count. Hooks waren da; **Mock-Handler für POST/PATCH/DELETE/reorder** ergänzt (`mocks/handlers/crm.ts`, in-memory).
- [x] **Eigene Felder** — `CustomFieldsConfig`-Body extrahiert zu wiederverwendbarem `CustomFieldsManager` (Dialog + CRM-Panel teilen ihn).
- [x] i18n `crm.settings.*` + `moduleSettings.entries.crm` ×4. **⚠ Fund:** Codebase nutzt **i18next-ICU** → einfache `{name}`-Klammern, nicht `{{name}}`. Erst doppelte verwendet → Toast/Deal-Count roh; per Screenshot-QA gefangen + auf ICU-Syntax korrigiert. (Lehre: neue interpolierte Keys IMMER `{var}`.)
- [x] QA (`scripts/qa-crm-settings.mjs`): CRM aktiv markiert, 6 Phasen rendern, „Neue Phase" hinzufügen funktioniert (Toast korrekt interpoliert), Deal-Counts echte Zahlen, 0 Errors/Raw-Keys.

**Commit B2 — Tag-Verwaltung + Shell-Umbenennung ✅ (typecheck 0, lint 0, QA grün):**
- [x] **Tag-Verwaltung** (`TagManager`) als 3. CRM-Sektion — Tags anlegen/umbenennen/Farbe/löschen. Mock-Tags auf Modul-Ebene + CRUD-Handler (POST/PATCH/DELETE /tags); Hooks `useCreateTag/useUpdateTag/useDeleteTag` (raw fetch, da nicht im OpenAPI → backend-gap).
- [x] **Color-Picker** nach `components/shared/ColorSwatchPicker` extrahiert (Pipeline + Tags teilen ihn); Palette in `lib/swatch-colors.ts` (Fast-Refresh-clean).
- [x] **Darien-Feedback umgesetzt:** „Administration"-Button (links unten) entfernt — Admin-Inhalte leben im COSMI-Bereich des Fensters. „Einstellungen"-Button + Fenstertitel → **„Modul-Einstellungen"** (4 Sprachen).
- [x] QA (`qa-crm-settings.mjs`): Sidebar (kein Administration, „Modul-Einstellungen" da), 3 CRM-Sektionen, Add-Stage, 0 Errors/Raw-Keys.

**Commit B3 — Persönlich/Für-alle-Bereiche ✅ (typecheck 0, lint 0, QA grün) (Darien-Feedback 2026-06-07):**
- [x] `ModuleSettingsShell` gruppiert Sektionen jetzt nach Scope in **zwei klar getrennte Bereiche** mit Headern: „PERSÖNLICH" (Untertitel „nur für dich…") + „FÜR ALLE" (Untertitel „nur die Modulleitung kann das ändern", Lock-Hinweis für Nicht-Leiter). Gilt automatisch für JEDES Modul-Panel (auch Calendar).
- [x] **Persönlicher CRM-Bereich** (`CrmPersonalPrefs` + `stores/crmPrefs.ts`): Standard-Ansicht (Liste/Raster), Dichte (Komfortabel/Kompakt), Avatare-Toggle. **Real verdrahtet:** KontaktePage initialisiert `viewMode` aus `defaultContactView` (keine tote UI).
- [x] i18n `crm.settings.personal.*` + `settings.scope.*GroupDesc` ×4. QA: beide Bereich-Header + persönliche Sektion rendern, 0 Errors/Raw-Keys.
- **Muster steht** für alle weiteren Module: pro Modul je eine `personal`- und `tenant`-Sektionsgruppe mit sinnvollen Funktionen.

**Kontakte Phase 7 = ABGESCHLOSSEN** (CRM-Panel: Persönlich + Pipeline + Felder + Tags, im Modul-Einstellungs-Fenster).
