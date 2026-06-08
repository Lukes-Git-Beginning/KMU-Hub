# Phase 02 — Zeiterfassung: eigenständiges Modul (aus dem Profil-Tab herauslösen)

> **Modul:** zeiterfassung · **Risiko:** mittel (viel ist Wiederverwendung) · **Backend:** nicht nötig — `timetracking`-Store + Demo.
> Aktuell ist Zeiterfassung nur ein **Tab im Profil**. Diese Phase macht daraus ein eigenständiges Modul mit eigener Route — die Views existieren schon und werden wiederverwendet.

## Ziel
Ein eigenständiges Modul `/zeiterfassung` mit eigener Layout-Shell, das die bestehenden Zeiterfassungs-Views nutzt, plus **Projekt/Kunde-Zuordnung** pro Eintrag und einen **Stundenkonto-Saldo** (Soll/Ist, mock-first).

## Ist-Stand (was schon da ist)
- Views (fertig, im Profil-Tab): `modules/profil/tabs/zeiterfassung/` — `TodayView.tsx`, `WeekView.tsx`, `MonthView.tsx`, `OverviewView.tsx`, `ReportsView.tsx`, `TeamView.tsx`, `ManualEntryForm.tsx`, `ExportDialog.tsx`, `time-utils.ts`.
- Tab-Wrapper: `modules/profil/tabs/ZeiterfassungTab.tsx`.
- Store: `stores/timetracking.ts` (entries, categories, targets, teamActivity).

## Muster-Vorlagen
- Modul-Shell mit View-Switcher: `modules/calendar/CalendarLayout.tsx` (Toolbar + View-Umschalter) oder `modules/work/WorkLayout.tsx`.
- Route-Registrierung: bestehende Module in `App.tsx` (Lazy-Import + Route — **Hot File, additiv**).

## Schritte
1. `modules/zeiterfassung/ZeiterfassungLayout.tsx` anlegen: Shell mit Sub-Navigation (Heute · Woche · Monat · Übersicht · Berichte · Team), die die **bestehenden** Views aus `profil/tabs/zeiterfassung/` rendert (importieren, nicht kopieren).
2. Route `/zeiterfassung` in `App.tsx` registrieren (Lazy) + Sidebar-Eintrag (Hot Files, additiv).
3. Den Profil-Tab `ZeiterfassungTab` auf einen schlanken Verweis „im Modul öffnen" reduzieren ODER vorerst bestehen lassen (kein Bruch) — Entscheidung im Review-Faden notieren.
4. **Projekt/Kunde-Zuordnung:** `ManualEntryForm` + Eintrags-Modell um optionales `projectId`/`customerId` erweitern (Picker aus `useProjects`/Kontakte-Hook; mock-first wenn nötig). Anzeige in den Views als kleines Label.
5. **Stundenkonto-Saldo:** in `OverviewView` (oder neuer `BalanceCard`) Soll (aus `targets`) vs. Ist (Σ entries) je Periode + kumulierter Saldo. Mock-first, klar als „Vorschau" markiert → echtes Stundenkonto = `backend-handover-luke.md`.
6. **i18n ×4** für neue Labels `zeiterfassung.*` (bestehende `profil.zeiterfassung.*` weiter nutzen wo passend).
7. Verifizieren, Review-Faden, commit, push.

## i18n-Keys
Neue modulweite Labels unter `zeiterfassung.*`; bestehende `profil.zeiterfassung.*` bleiben gültig. 4 Sprachen.

## Demo-Handler
Keiner (timetracking ist ein Zustand-Store mit Demo-Daten). Projekt/Kunde-Picker mock-first falls Hooks leer.

## Definition-of-Done
- [ ] `/zeiterfassung` als eigenes Modul erreichbar (Route + Sidebar), nutzt die bestehenden Views (kein Duplikat).
- [ ] Manuelle Einträge können Projekt/Kunde bekommen; das Label erscheint in den Views.
- [ ] Stundenkonto-Saldo (Soll/Ist + kumuliert) sichtbar, mock-first als „Vorschau" markiert.
- [ ] Profil-Tab funktioniert weiterhin (kein Bruch).
- [ ] i18n ×4, scoped tsc grün (`tsconfig.zeiterfassung-standalone.json`), QA grün, Screenshots angesehen.
- [ ] Review-Faden in `reviews/zeiterfassung.md`.

## QA-Hinweis
`scripts/qa-zeiterfassung-standalone.mjs`: `/#/zeiterfassung` öffnen → Sub-Views durchschalten → manuellen Eintrag mit Projekt anlegen → Saldo-Karte sichtbar → 0 Roh-Keys/pageErrors. Screenshots Heute/Woche/Übersicht.
