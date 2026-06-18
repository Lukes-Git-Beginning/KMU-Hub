# Main-Terminal — dashboard Tiefe-Pass (D-1 … D-5)

> Mein Batch (Hauptklon, Port 5173). Modul **dashboard** ist ~75–85 % fertig — DnD (react-grid-layout) komplett, Persistenz via Zustand+server-sync da, Cross-Module/Alerts echt verdrahtet. Restarbeit = Verkabeln + Bugs + Demo-Tiefe. Lane-Regeln: `README.md`.

## Ist-Kern (2026-06-17)
- Store: `stores/dashboard.ts` (Zustand persist `cosmi-dashboard`, v2; `personalLayouts`/`personalActiveWidgets` + `teamLayouts`/`teamActiveWidgets`; `debouncedServerSync` → `PUT /api/v1/dashboard/layout`; `initFromServer`). `stores/dashboardSettings.ts`, `stores/dashboardPrefs.ts`.
- DnD: `components/widgets/WidgetContainer.tsx` (react-grid-layout, 12 Spalten, drag/resize im Edit-Modus) — **fertig, nur verifizieren**.
- MSW: `mocks/handlers/dashboard.ts` antwortet `GET /dashboard/layout` immer `{ layout: null }`, **kein PUT**, kein active_widgets, kein feature-flags-Handler.

## D-1 — Persistenz scharf + Store-Bug
- `mocks/handlers/dashboard.ts`: `PUT/GET /api/v1/dashboard/layout` (inkl. `active_widgets`, stateful echo), `GET/PUT /api/v1/dashboard/defaults/{role}`, `GET /api/v1/feature-flags`. → server-sync feuert nicht mehr ins 404-Leere.
- Store-Bug: `modules/settings/DashboardSettings.tsx` L79–80 nutzt `s.layouts`/`s.activeWidgets` (v2 umbenannt) → auf `personalLayouts`/`personalActiveWidgets`. `handleCopyCurrentLayout` kopiert sonst `undefined`.
- `initFromServer()` beim App-/DashboardPage-Start aufrufen (wird aktuell nie getriggert).

## D-2 — Demo-Tiefe (tote Buttons)
- `widgets/MyTasks.tsx` L104–113: Zeilen `role="button"`+cursor aber **kein onClick** → Klick navigiert zur Task (work-Detail).
- `widgets/Absences.tsx` L97: Zeilen cursor aber kein onClick → navigieren (zeiterfassung/Abwesenheit).
- `components/dashboard/ProfileWidgetSuggestions.tsx` L102: Plus-Button **ohne Handler** → Widget wirklich hinzufügen.
- `widgets/CrossModuleOverview.tsx` L140–143: Chat-`unreadCount` hardcoded 0 → an `useChannels`/unread anbinden.
- `widgets/Birthdays.tsx`: hardcoded `getUpcomingBirthdays()` → MSW-Hook/echte Quelle.

## D-3 — KPI lizenz-/modulabhängig (Demo zeigt alles)
- `GET /api/v1/feature-flags` MSW-Handler, **alle Flags an** (Darien: Demo zeigt alles).
- `allowedWidgets` (dashboardSettings) ↔ feature-flags konsistent machen (zwei Gating-Ebenen synchron). Gating funktioniert, schränkt Demo aber nicht ein. `WidgetRegistry` `module`-Feld + `isWidgetAllowed` sind da.

## D-4 — Team-Dashboard
- `widgets/TeamWorktime.tsx` L65–76: Mitarbeiter 2–6 `SEEDED_MINUTES_BY_INDEX` → echte MSW-`useWorkTimeEntries` für alle.
- Presence (`usePresence`/`useBulkPresence`) im TeamStatus rund; ScopeToggle Personal/Team verifizieren.
- Rollen-Templates-Seite (`DashboardSettings.tsx`) mit D-1-Handlern verbinden.

## D-5 — Cross-Module/Alerts-Feinschliff + DnD-Verify
- `CrossModuleOverview` + `AlertsSection` Feinschliff (alle Rows Links, leere Zustände sauber).
- DnD (D-2-unabhängig) nur verifizieren — fertig.

## DoD
Alle 5 verifiziert (Screenshots), 0 Raw-Keys/Errors, je 1 Commit+Push (rebase davor), `qa-dashboard.md` gepflegt → dashboard review-reif → Nico.
