# QA-Protokoll — dashboard (Main-Terminal)

> Pro fertiger Phase ein Eintrag: was gebaut, was Darien anschauen soll, Screenshots. `[PATTERN]` = zuerst ansehen (betrifft mehrere Phasen).

## D-1 — Persistenz scharf + Store-Crash-Fix ✅
**Gebaut:** MSW-Handler für `PUT/DELETE /dashboard/layout` + `GET/PUT /dashboard/defaults/{role}` (stateful, in-memory) ergänzt; Store-Crash in der Admin-Rollen-Seite gefixt (`s.layouts`→`personalLayouts`); `initFromServer()` beim Dashboard-Mount aufgerufen.
**Dateien:** `mocks/handlers/dashboard.ts`, `modules/settings/DashboardSettings.tsx`, `modules/dashboard/DashboardPage.tsx`.
**Was du anschauen sollst:**
1. **Admin-Rollen-Seite** `#/settings/dashboard` → Tab Administrator/Manager/Mitarbeiter, dann **„Aktuelles Layout als Standard"** klicken → muss **grünen Erfolgs-Toast** zeigen, **kein Weiß-Screen** (das war vorher ein harter Crash: `undefined.map`). Screenshot `4-after-copy.png`.
2. **Dashboard anpassen** (Button oben rechts) → Widget hinzufügen/entfernen → nach Reload bleibt der Stand (server-sync läuft jetzt durch statt 404).
**Verifiziert:** pageErrors 0, raw-keys 0, alle 5 Persistenz-Endpoints 200 + stateful echo. Screenshots: `.qa-screenshots/dashboard-d1/`.

## D-2 — Demo-Tiefe / tote Buttons ✅
**Gebaut:** (1) MyTasks-Zeile klickbar → navigiert zu `/work/my-tasks` (+ Keyboard). (2) Absences-Zeile klickbar → `/team`. (3) ProfileWidgetSuggestions „+"-Button → fügt echtes Widget hinzu + Toast + Karte verschwindet (war: Button ohne Handler). (4) CrossModule „ungelesene Nachrichten" zeigt echte Zahl via `useUnreadCounts()` (war: hardcoded 0). (5) Birthdays MSW-backed via `useBirthdays()` (war: direkter mock-db-Import).
**Dateien:** `widgets/MyTasks.tsx`, `widgets/Absences.tsx`, `components/dashboard/ProfileWidgetSuggestions.tsx`, `widgets/CrossModuleOverview.tsx`, `widgets/Birthdays.tsx`, `api/hooks/useBirthdays.ts`, `mocks/handlers/dashboard.ts` (birthdays-Endpoint).
**Was du anschauen sollst:**
1. **MyTasks-Widget** → Zeile klicken → landet in „Meine Aufgaben". Screenshot `mytasks-widget.png`.
2. **„Empfohlene Widgets"-Karte** → „+" klicken → grüner Toast „… zum Dashboard hinzugefügt" + Karte weg. Screenshot `4-after-plus.png`.
3. **„Heute im Überblick"** → zeigt jetzt **„8 ungelesene Nachrichten"** (vorher 0). Screenshot `6-unread-recheck.png`.
4. **Geburtstage-Widget** → lädt 5 Einträge via MSW. Screenshot `5-widgets.png`.
**[BEOBACHTUNG — separates Thema, nicht D-2]** Das **Abwesenheiten-Widget ist leer** („0 Personen heute abwesend") wegen eines **vorbestehenden Pipeline-Bugs außerhalb der dashboard-Lane**: `useAbsenceCalendar` macht `select: data => data.entries`, aber beide MSW-Handler liefern `{absences: …}` (Shape-Mismatch); zusätzlich Feld-Mismatch (`user_id` vs. `employeeId`) und ein **Duplikat-Handler** (`hr.ts` liefert `[]` und überschattet `team.ts`). Mein onClick-Fix ist korrekt, aber erst testbar, wenn das Widget Daten zeigt. → Kandidat für einen HR-/team-Lane-Fix oder einen späteren Tiefe-Punkt.
**Verifiziert:** Build exit 0, pageErrors 0, raw-keys 0. Screenshots: `.qa-screenshots/dashboard-d2/`.

## D-3 — KPI lizenz-/modulabhängig ✅ (war bereits erfüllt, verifiziert)
**Befund:** Bei genauer Prüfung war D-3 schon vollständig implementiert — **kein Code-Bau nötig** (ehrlicher als ein toter MSW-Handler, der im Demo-Mode wegen `enabled: !IS_DEMO` nie aufgerufen würde).
- KPI-Widgets haben `module`-Felder (kpi-revenue→finance, kpi-tasks→tasks, kpi-deals→crm) in `WidgetRegistry`.
- **Zwei Gating-Ebenen korrekt kombiniert:** Picker filtert `allowedWidgets.includes(id) && isWidgetAllowed(module)` (Tenant-Admin UND Lizenz), Grid filtert nach Lizenz-Flag.
- **Demo zeigt alles** (fail-open, weil `flags` im Demo undefined) — dein Wunsch.
**Verifiziert** via `scripts/qa-dashboard-gating.mjs` (`pass: true` beide Szenarien): (b) Demo zeigt alle 19 Widgets; (c) „CRM=AUS" injiziert → CRM-Widgets aus Grid + Picker raus, non-CRM bleibt. pageErrors 0, raw-keys 0.
**Was du anschauen sollst:** nichts Neues — Demo zeigt wie gewünscht alle Widgets; das Lizenz-Gating wirkt technisch (per Flag-Injection getestet, nicht im Demo sichtbar).

## D-4 — Team-Dashboard ✅
**Gebaut:** TeamWorktime liest jetzt **echte Pro-Mitarbeiter-Wochenstunden** via neuem MSW-Endpoint `GET /dashboard/team-worktime` (`useTeamWorktime`) — **kein clientseitiges Seeding** (`SEEDED_MINUTES_BY_INDEX`) mehr; Werte sind deterministisch + konsistent pro Mitarbeiter, swap-ready. ScopeToggle + Presence verifiziert.
**Dateien:** `api/hooks/useTeamWorktime.ts`, `mocks/handlers/dashboard.ts` (team-worktime-Endpoint), `widgets/TeamWorktime.tsx`.
**Was du anschauen sollst:**
1. **Dashboard → „Team"** (Umschalter oben rechts) → **Team-Dashboard** mit Team-Status (Presence), Geburtstage, Stempeluhr, Offene Tickets, Team-Arbeitszeit. Screenshot `2-team-scope.png`.
2. **Team-Arbeitszeit** → 6 Mitarbeiter mit unterschiedlichen, konsistenten Wochenstunden (39h/40h/42h/43h/44h/32h), Balken über/unter Ziel eingefärbt. Screenshot `1-team-widgets.png`.
**Verifiziert:** Build exit 0, 6 distinct hour values, ScopeToggle ok, pageErrors 0, raw-keys 0. Screenshots: `.qa-screenshots/dashboard-d4/`. (Abwesenheiten-Widget weiter leer = der unter D-2 dokumentierte HR-Pipeline-Befund.)
