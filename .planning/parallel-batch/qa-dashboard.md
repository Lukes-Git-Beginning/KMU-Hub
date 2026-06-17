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
