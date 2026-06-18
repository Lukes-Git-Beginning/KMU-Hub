# Phase 01 (Pilot) — Kalender: Tag/Woche/Monat-Views echt machen

> **Modul:** calendar · **Risiko:** mittel (klar gemustert, reines FE) · **Backend:** nicht nötig — Events aus `useEvents` (Hook da) bzw. Demo-Mock.
> Pilot von Strom D. Die Views sind aktuell **Platzhalter** — das ist der echte Gap. Es gibt starke Muster im Repo.

## Ziel
Im Kalender-Modul (`/kalender` bzw. `/calendar`) die drei Ansichten **Tag · Woche · Monat** mit echten Terminen rendern. Der View-Switcher + Navigation existiert schon in der Layout-Shell; nur die View-Inhalte fehlen.

## Ist-Stand (was schon da ist)
- Shell: `modules/calendar/CalendarLayout.tsx` — Toolbar mit View-Switcher (`useCalendarStore`: `currentView` `'day'|'week'|'month'`, `currentDate`, `navigateForward/Backward`, `goToToday`). Die Views-Bereiche sind Platzhalter.
- Store: `stores/calendar.ts` (`CalendarView`, currentDate, sidebarOpen, …).
- Events-Hook: `api/hooks/useEvents.ts` · Kalender-Hook: `api/hooks/useCalendars.ts`.
- i18n-Präfix: **`kalender.*`** (z.B. `kalender.view.day/week/month` existieren bereits).
- `date-fns` + `date-fns/locale` `de` sind im Modul in Gebrauch.

## Muster-Vorlagen (1:1 als Vorlage)
- **Monatsgrid:** `modules/work/calendar/WorkCalendarView.tsx` + `modules/work/calendar/calendar-utils.ts` (heute gebaut!) — Mo-ausgerichtetes Wochen-Grid, lokale Date-Keys, Items pro Tag als Chips, heute/Wochenend-/Fremdmonats-Styling. **Kopiere die Grid-Logik** (statt Tasks → Events).
- **Wochen-/Monatsraster (alternativ):** `modules/profil/tabs/zeiterfassung/WeekView.tsx` + `MonthView.tsx` + `time-utils.ts`.
- **Tagesansicht:** vertikale Stunden-Spalte (0–24h) mit Event-Blöcken nach Start/Ende positioniert (Stunde × Höhe).

## Schritte
1. `git checkout -B marathon/dein-pc` (einmal), `git pull` einmal. App `/kalender` öffnen.
2. `modules/calendar/views/` anlegen mit `MonthView.tsx`, `WeekView.tsx`, `DayView.tsx`.
3. Events laden: `useEvents({ from, to })` für den sichtbaren Bereich (aus `currentDate` + view ableiten). Falls der Hook im Demo leer ist → kleiner lokaler Demo-Seed (stabil) + Hinweis nach `backend-handover-luke.md`.
4. **MonthView:** Mo-ausgerichtetes 7-Spalten-Grid (Muster `calendar-utils.ts`), Events am Starttag als getönte Chips (Klick → Detail/Popover), Überlauf „+N", heute hervorgehoben.
5. **WeekView:** 7 Spalten (Mo–So) × Stundenraster; Events als Blöcke nach Uhrzeit. **DayView:** eine Spalte, gleiches Stundenraster.
6. In `CalendarLayout` die Platzhalter durch `currentView === 'month' ? <MonthView/> : 'week' ? <WeekView/> : <DayView/>` ersetzen; Navigation/Today wirken auf alle drei.
7. **i18n ×4:** neue Labels `kalender.*` in `{de,en,fr,it}.json`, einfache Klammern. (Wochentag-/Monatsnamen über `date-fns`-Locale, nicht i18n.)
8. Verifizieren, Review-Faden, commit, push.

## i18n-Keys
Präfix `kalender.*` (z.B. `kalender.noEvents`, `kalender.allDay`). 4 Sprachen, `{var}`.

## Demo-Handler
Falls `useEvents` im Demo leer: lokaler Demo-Seed im View (stabil, aus Datum abgeleitet) — Hauptsache Events sind sichtbar. Echter Events-Endpoint/CRUD → `backend-handover-luke.md` (Calendar P2).

## Definition-of-Done
- [ ] Monat/Woche/Tag rendern echte (oder Demo-)Termine; Switcher + Navigation + Heute wirken auf alle drei.
- [ ] MonthView nutzt das `calendar-utils`-Grid-Muster (Mo-ausgerichtet, heute hervorgehoben).
- [ ] Keine Roh-i18n-Keys, keine Emojis, keine ASCII-Umlaute; leere Tage sehen sauber aus.
- [ ] Gescopter Typecheck grün (`tsconfig.calendar-views.json`), QA grün, Screenshots aller 3 Views @1440 angesehen.
- [ ] Review-Faden in `reviews/calendar.md`.

## QA-Hinweis
`scripts/qa-calendar-views.mjs`: `/#/kalender` öffnen → je View umschalten → prüfen, dass ein Grid/Stundenraster + mind. ein Event-Element rendert, 0 Roh-Keys, 0 pageErrors. Screenshot je View.
