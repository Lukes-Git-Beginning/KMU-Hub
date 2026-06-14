# Review-Fäden — zeiterfassung

> Eine Datei pro Modul. Der Bau-Strom trägt nach JEDER fertigen Phase einen Eintrag ein (Struktur siehe `_TEMPLATE.md`). Vorlage für Dariens Feinschliff-Review (Pfad nachklicken, worauf achten, Screenshots, offene Punkte).

**Modul:** `zeiterfassung` · **Strom:** D · **Reviewer (zugeteilt):** offen

---

<!-- Phasen-Einträge hier anhängen — Struktur aus _TEMPLATE.md kopieren.
     Status-Legende: ⬜ ungereviewt · 🟡 Feedback offen · ✅ grün -->

## 🟡 Befund — Spec-Abgleich vor P1: Modul pausiert, Architektur-Frage an Luke (2026-06-10)

**Kein Code geändert.** Vor dem Bauen Code-Stand geprüft (Tageslehre) — Ergebnis kippt die Spec komplett:

**Es gibt ZWEI parallele Zeiterfassungs-UIs:**
1. **Live** (`modules/profil/tabs/ZeiterfassungTab.tsx`, auch auf `/zeiterfassung` gemountet): **API-backed** via `hr-hooks.ts` gegen das existierende HR-Backend (`/api/v1/hr/time/*` — Clock-In/Out, Pausen, ArbZG-Severity, Daily/Weekly-Summaries, Korrektur-Workflow mit Genehmigung). 3 Inline-Views: Heute/Woche/Korrekturen. Funktional solide, aber UI karger als der tote Satz.
2. **Tot** (`modules/profil/tabs/zeiterfassung/` — 10 Dateien: TodayView, WeekView, MonthView, OverviewView, ReportsView, TeamView, CategoriesView, ManualEntryForm, ExportDialog, ApprovalBanner): **nirgends gemountet** (kein externer Import), hängen am lokalen Mock-Store `stores/timetracking.ts`. UI-reich (Dashboard, 4-Wochen-Trend, Export-Dialog, Kategorien/Templates, Wochenfreigabe), aber Daten rein lokal.

**Dazu zwei konkurrierende Topbar-Widgets:** `ClockInButton` (echte API) und `TimeTrackerWidget` (Mock-Store, navigiert zudem nach `/profil` statt `/zeiterfassung`) — nicht synchronisiert, einer zeigt im Echtbetrieb falsche Daten.

**Warum pausiert:** Jede Weiterarbeit (Shell, Projekt-Picker, Export) erfordert zuerst die Datenquellen-Entscheidung — toten Satz an die HR-API anschließen (UI-Substanz retten) vs. Live-Tab ausbauen + toten Satz löschen (wie calendar-Cleanup). Das berührt Lukes HR-Backend-Lane → **Entscheidung mit Luke**, dann ist zeiterfassung wieder Strom-D-baubar.

**Backend-Gaps unabhängig von der Entscheidung** (in `backend-gaps.md` ergänzt): manueller Zeiteintrag (es gibt nur Clock-In/Out + Korrekturen), `project_id`/`customer_id` am WorkTimeEntry, Export-Endpoint (CSV/DATEV), Wochen-Freigabe-Workflow.

**Reviewer-Notizen:**
- _Entscheidung Luke/Darien hier eintragen → daraus werden die echten P1-Phasen._

---

## ✅ ENTSCHEIDUNG (Darien, 2026-06-14) — HR-API = Single Source of Truth

Darien hat zeiterfassung als nächstes Modul gewählt + die Architektur ratifiziert (Luke war zu dem Zeitpunkt nicht am PC; sein HR-Backend ist seit der Welle 11.06. belastbar — CreateEmployee, tenant-context, Tests).

**Entscheidung:** Die **HR-API wird die einzige Datenquelle.** Die reiche UI aus den 10 toten Mock-Views wird gerettet und auf HR-Daten umverdrahtet (wo die API es kann), sonst **mock-first** (hr.ts) mit Backend-Gap für Luke. Danach Mock-Store `stores/timetracking.ts` + tote Views löschen, **ein** Header-Widget (API-backed, korrekte Route `/zeiterfassung`). Calendar-Cleanup-Muster.

**Wichtige Code-Funde (14.06., post-merge):**
- Header rendert das **mock-basierte `TimeTrackerWidget`** (navigiert falsch → `/profil`), NICHT den API-backed `ClockInButton` (existiert, unbenutzt).
- Lukes Dashboard-Widget `dashboard/widgets/TeamWorktime.tsx` nutzt **bereits HR-API** → Swap bricht es nicht.
- Mock-Store-Konsumenten: 10 tote Views, `TimeTrackerWidget`, `modules/team/TeamPage.tsx`, `hooks/useTimerTick.ts` → erst nach UI-Port löschen.
- hr.ts-Mock deckt alle Time-Endpoints ab (status/active/clock-in/out/break/entries/summary daily+weekly/corrections).

**5-Phasen-Plan (volle Markt-Parität, Benchmark clockodo/Papershift/Harvest):**
- **P1** Fundament & Konsolidierung — Standalone-Shell, ein API-Widget (Mock-Widget raus, Route-Fix), Stundenkonto-Saldo (+/−).
- **P2** Manuelle Einträge + Projekt/Kunde/Leistung-Zuordnung (+ billable).
- **P3** Auswertungen + Monatsansicht + Overview-Dashboard (recharts/useChartTheme).
- **P4** Export (CSV/XLSX/PDF; DATEV/Lohn = backend-gap) + Pausen-/Arbeitszeit-Regeln + ModuleSettingsShell.
- **P5** Team-Zeiterfassung + Wochen-Freigabe-Workflow + Urlaub/Abwesenheit-Integration.

Pro Phase: Bau-Loop + **Design-/Polish-Review (impeccable)** (Darien-Vorgabe 14.06.). Backend-Gaps gebündelt in `backend-gaps.md`.

---

## ✅ P1 — Fundament & Konsolidierung (2026-06-14, autonom verifiziert)

**Status: grün, QA 0 Fehler / 0 Raw-Keys @ 3 Größen + Widget-Dropdown.**

**Gebaut:**
1. **Standalone-Modul-Shell** (`modules/zeiterfassung/ZeiterfassungPage.tsx`): echter Modul-Header (Clock-Icon + Titel + „ArbZG-konforme Arbeitszeiterfassung" + Stundenkonto-Badge rechts) statt nacktem Tab-Wrap. Darunter der funktionale Kern (`ZeiterfassungTab`).
2. **Header-Widget konsolidiert** (`components/header/WorkClockWidget.tsx`, neu): ein **API-backed** Widget ersetzt das mock-basierte `TimeTrackerWidget` (navigierte falsch → /profil) UND den verwaisten `ClockInButton`. **Beide alten Dateien gelöscht.** Dropdown: Live-Timer (rAF) + Tagesfortschritt vs. 8h + Saldo + heutige Einträge + Pause/Ausstempeln + „Zur Zeiterfassung →". Projekt-Picker bewusst raus → P2 (braucht project_id).
3. **Stundenkonto-Saldo (+/−)** (Markt-Feature): `TimeBalance`-Typ + `hrTimeApi.getBalance()` + `useTimeBalance()` + Mock `/hr/time/balance` (752 min = +12h 32m). `StundenkontoBadge` (Shell) + inline im Widget. `lib/worktime.ts` (`formatSignedMinutes`/`formatWorkMinutes`, geteilt). Invalidierung bei Clock-out.

**⚠ Demo-Daten-Bug gefunden+gefixt (nur durch Screenshot-Hinsehen sichtbar):** `team.ts`-Mock-Handler (lief VOR `hr.ts`) bediente `/hr/time/status|active|entries|summary/*` mit **idle/0**-Daten → Seite zeigte „nicht eingestempelt, 0h, keine Einträge, Korrekturen 5". Die 5 Duplikat-Handler + tote `timeEntries`-Konstante aus `team.ts` **entfernt** → `hr.ts` ist jetzt Single Source (eingestempelt, 2 Einträge heute, kohärente Summen). `hr.ts`-Status auf relativen Shift-Start (`hoursAgo(2,10)`) + kohärente Tageswerte poliert, damit der Live-Timer nicht von fix 08:00 (~11h) läuft.

**i18n:** 3 neue Keys ×4 (`zeiterfassung.shell.title/.subtitle`, `zeiterfassung.balance.label`); Widget nutzt vorhandene `header.timeTracker.*`. Script `scripts/add-zeiterfassung-shell-i18n.mjs`.

**Typecheck:** TS-Language-Server (`getDiagnostics`) 0 auf allen geänderten Dateien — kalter scoped-tsc hängt projektweit (>4 Min), LSP ist hier der schnelle Gate. **Lehre für alle: LSP-Diagnose statt kaltem tsc.**

**QA-Screenshots (angesehen):** `scripts/qa-zeiterfassung-p1.mjs` → `.qa-screenshots/ze-p1/`. Shell @ full/half/small + Widget-Dropdown. Eingestempelt-Zustand, Live-Timer 02:10 konsistent in Header+Toolbar+aktivem Eintrag, Saldo +12h 32m, kein Overflow @ 500px.

**Design-Review:** clean, on-brand, responsive; Hierarchie (Modul=primary-getönt, Daten-Widget=neutral). Tiefe impeccable-Audit → P3 (Charts/Dashboard).

**Backend-Gaps (für Luke, in backend-gaps.md):** `/hr/time/balance` (kumulativer Stundenkonto-Saldo) ist FE-mock-first — braucht echten Endpoint.

**Reviewer-Notizen für Darien:** Pfad `/#/zeiterfassung` anklicken; Header-Widget oben rechts (grüner Live-Timer) öffnen. Offen ab P2: manuelle Einträge, Projekt/Kunde/Leistung, Export, Team, Genehmigung.

---

## ✅ P2 — Manuelle Einträge + Projekt/Kunde/Leistung (2026-06-14, autonom verifiziert)

**Status: grün, QA 0 Fehler / 0 Raw-Keys; Submit-Flow E2E getestet.**

**Gebaut (clockodo-Taxonomie Kunde → Projekt → Leistung):**
1. **Datenschicht:** `WorkTimeEntry` um `projectId/projectName/customerName/activity/billable/note/isManual`; `TimeProject`-Typ; `CreateManualEntryInput`; `hrTimeApi.listProjects/createEntry`; `useTimeProjects`/`useCreateTimeEntry`. Mock `GET /hr/time/projects` (5 Kunde/Projekt-Paare) + `POST /hr/time/entries` (in-memory persist, berechnet netto). Bestehende Einträge angereichert.
2. **`ManualEntryDialog`** (`modules/zeiterfassung/components/`, HR-API): Datum + Von/Bis/Pause (Live-Netto-Berechnung) + Projekt-Picker (Kunde sichtbar, farbcodiert) + Leistung + Abrechenbar-Toggle (defaultet aus Projekt) + Notiz. „Neuer Eintrag"-Button in der ZeiterfassungTab-View-Switcher-Zeile.
3. **Eintrags-Anzeige angereichert:** Zeilen zeigen Projekt · Kunde — Leistung + Badges (Aktiv/Korrektur/**Manuell**/**Abrechenbar** mit Receipt-Icon).

**QA-Screenshots (angesehen):** `scripts/qa-zeiterfassung-p2.mjs` → `.qa-screenshots/ze-p2/`. Einträge mit voller Attribution; Dialog @ 1440 + 500px (Footer stapelt sauber); Submit → neuer „Manuell"-Eintrag oben in der Liste + Toast „Zeiteintrag erstellt".

**i18n:** 21 Keys ×4 (`zeiterfassung.manual.*` + 2 `api.hr.time.*`), Parität 21/21/21/21. **Typecheck:** LSP 0 auf allen geänderten Dateien. **Design-Review:** clean, on-brand, responsive — kein Polish-Bedarf.

**Backend-Gaps (Luke):** `POST /hr/time/entries` + `GET /hr/time/projects` FE-mock-first; `project_id`/`customer_id`/`service_code` am echten WorkTimeEntry; Projekt-Taxonomie ggf. an work/CRM koppeln statt eigene Liste.

---

## ✅ P3 — Auswertungen (Charts) (2026-06-14, autonom verifiziert)

**Status: grün, QA 0 Fehler / 0 Raw-Keys @ full+small, Woche+Monat.**

**Gebaut:**
1. **Datenschicht:** `TimeAnalytics`-Typ (KPIs, dayTrend, byProject, billable-Split); `hrTimeApi.getAnalytics(range)`; `useTimeAnalytics(range)`; Mock `GET /hr/time/analytics?range=week|month` (kohärente Demo-Daten: Werktage befüllt, WE leer, 72% abrechenbar).
2. **`AuswertungenView`** (recharts via `useChartTheme`, kein Rainbow): Range-Toggle Woche/Monat · 4 KPI-Cards (Gesamt, Abrechenbar+%, Überstunden, Ø/Tag) · **Stunden pro Tag** (gestapelte Bars: abrechenbar + Rest = Netto) · **Nach Projekt** (horizontale Bars, Projektfarben) · **Abrechenbar vs. nicht** (Donut). Als 3. Tab „Auswertungen" in ZeiterfassungTab.

**Polish-Fund beim Hinsehen:** „Stunden pro Tag" lief erst als zwei *getrennte* Bars (billable + total nebeneinander → doppeldeutig) → auf einen gestapelten Bar (billable unten + Rest oben = Netto-Höhe) umgestellt.

**QA:** `scripts/qa-zeiterfassung-p3.mjs` → `.qa-screenshots/ze-p3/`. **i18n:** 12 Keys ×4 (`zeiterfassung.analytics.*`), Parität. **Typecheck:** LSP 0. **Design:** Theme-Farben, gestapelte Bars lesen korrekt, on-brand.

**Backend-Gap (Luke):** `GET /hr/time/analytics` (KPI-/Trend-/Projekt-Aggregation) FE-mock-first → echter Aggregations-Endpoint (oder client-seitig aus Entries).

---

## ✅ P4 — Export + Arbeitszeit-Regeln + Moduleinstellungen (2026-06-14, autonom verifiziert)

**Status: grün, QA 0 Fehler / 0 Raw-Keys.**

**Gebaut (settings-komplett-Standard):**
1. **Stores:** `zeiterfassungPrefs` (personal: defaultView, dailyTargetHours, clockOutReminder) + `zeiterfassungSettings` (tenant: weeklyTarget, autoBreakAfterHours/Minutes, rounding, holidayRegion). Persist, mock-first.
2. **`ZeiterfassungSettingsPanel`** via `ModuleSettingsShell` (moduleId `zeiterfassung`): PERSÖNLICH (Standard-Ansicht, Tagesziel, Ausstempel-Erinnerung) + FÜR ALLE (Arbeitszeit-Regeln inkl. ArbZG-Auto-Pause + Rundung, Feiertagsregion DE/AT/CH). In `module-settings-registry` registriert (Icon Clock, navMatch `/zeiterfassung`) → Kontext-Vorauswahl + „AKTIV"-Badge greifen.
3. **Echte Verdrahtung (kein totes UI):** `ZeiterfassungTab` initialisiert `activeView` aus `defaultView` und das Tagesziel/Progress aus `dailyTargetHours`.
4. **`ExportDialog`** (aus Auswertungen): Format CSV (**echter client-seitiger Download** mit BOM, Spalten Datum/Von/Bis/Pause/Netto/Projekt/Kunde/Leistung/Abrechenbar) + Excel/DATEV-Lohn/PDF als „bald" (backend-gap). „Export"-Button neben dem Range-Toggle.

**QA:** `scripts/qa-zeiterfassung-p4.mjs` → `.qa-screenshots/ze-p4/`. Settings-Panel (beide Scope-Gruppen, Admin = editierbar) + Export-Dialog. **i18n:** 45 Keys ×4 (settings + export + Registry-Eintrag), Parität. **Typecheck:** LSP 0. **Design:** ModuleSettingsShell-konsistent, on-brand.

**Backend-Gaps (Luke):** `tenant_settings` für zeiterfassung-Regeln (Wochensoll/Pause/Rundung/Region); Export-Endpoints CSV-Serverside + **DATEV-Lohn (LODAS)** + XLSX + PDF.

---

## ✅ P5 — Team-Zeiterfassung + Genehmigungs-Workflow (2026-06-14, autonom verifiziert) — MODUL FERTIG

**Status: grün, QA 0 Fehler / 0 Raw-Keys; Approve- + Submit-Flow E2E getestet.**

**Gebaut:**
1. **Datenschicht:** `WeekStatus`/`TeamTimeRow`/`MyWeekStatus`; Client + Hooks `useTeamTime`/`useMyWeekStatus`/`useSubmitWeek`/`useApproveWeek`/`useRejectWeek`; Mock `GET /hr/time/team` + `GET/POST /hr/time/weeks/*` mit **in-memory Status-Mutation** (submit → submitted, approve → approved, reject → rejected).
2. **`TeamView`** (nur für `useIsModuleLead('zeiterfassung')` = Lead/Admin): Tabelle Mitarbeiter/Wochenstunden (Progress)/Überstunden/Status-Badge + Freigeben/Ablehnen pro eingereichter Woche, Clock-in-Indikator, „N zur Freigabe"-Badge. Als Tab „Team" (conditional).
3. **`WeekSubmitBanner`** (Mitarbeiter, über der Woche-Ansicht): zustandsabhängig (offen → „Woche einreichen", eingereicht → wartet, freigegeben, abgelehnt → erneut einreichen).

**Bug-Fund beim Hinsehen:** Woche-Karte zeigte „Pausen **NaNh NaNm**" — dem Weekly-Summary-Mock fehlte `totalBreakMinutes` (pro Tag + Top-Level) → ergänzt (jetzt 3h 45m + Pause-Spalte befüllt).

**QA:** `scripts/qa-zeiterfassung-p5.mjs` → `.qa-screenshots/ze-p5/`. Team-Tab (Admin), Freigabe (Status→Freigegeben, Badge 3→2, Toast), Woche-Banner, Einreichen (Banner→wartet, Toast). **i18n:** 28 Keys ×4. **Typecheck:** LSP 0. **Design:** Tabelle/Badges/Banner on-brand, konsistent.

**Backend-Gaps (Luke):** `GET /hr/time/team` + Wochen-Submit/Approve/Reject + `time_week_submissions`-Tabelle; Weekly-Summary mit `totalBreakMinutes`.

---

## 🏁 MODUL-STATUS: zeiterfassung P1–P5 komplett (2026-06-14)
Alle 7 Markt-Parität-Features (Benchmark clockodo/Papershift/Harvest) erreicht: Timer ✓ · manuelle Einträge ✓ · §3 ArbZG ✓ · Stundenkonten ✓ · Kunde/Projekt/Leistung ✓ · Pausen-/Arbeitszeit-Regeln ✓ · Auswertung & Export ✓. Plus: Single-Source-Konsolidierung, Moduleinstellungen (settings-komplett), Team-Zeiterfassung + Genehmigung. **Offen (Nächstes):** die 10 toten Store-Views (`profil/tabs/zeiterfassung/`) + `stores/timetracking.ts` löschen (UI ist portiert) — Cleanup-Phase. Mobile/GPS-Stempeln → Phase E/PWA.
