# Review — dashboard

> **Status:** review-reif (D-1…D-5, Main-Terminal, gemergt, `f4a6844d`).
> **Lane:** FE/UX-Review (mock-first). Echte Backend-Persistenz = Lukes Lane.
> **Screens:** `/#/dashboard` (persönlich + Team-Umschalter oben rechts) · `/#/settings/dashboard` (Rollen-Standards).

## Was gebaut wurde (Definition of Done)
- [x] **D-1 — Persistenz + Crash-Fix**: Layout-Änderungen (Widget hinzufügen/entfernen) überleben Reload (MSW PUT/DELETE `/dashboard/layout` stateful). Admin-Rollen-Seite „Aktuelles Layout als Standard" → grüner Toast, **kein Weiß-Screen** mehr (war harter `undefined.map`-Crash).
- [x] **D-2 — Demo-Tiefe / tote Buttons**: MyTasks-Zeile → `/work/my-tasks`, Absences-Zeile → `/team` (beide auch per Tastatur). „Empfohlene Widgets" + → fügt Widget hinzu + Karte verschwindet. „Heute im Überblick" zeigt echte ungelesene Zahl. Geburtstage MSW-backed.
- [x] **D-3 — KPI lizenz-/modulabhängig**: verifiziert (war bereits korrekt). Demo zeigt alle 19 Widgets (fail-open, gewollt); Gating wirkt per Flag (CRM=AUS entfernt CRM-Widgets aus Grid + Picker).
- [x] **D-4 — Team-Dashboard**: Umschalter → Team-Status (Presence), Geburtstage, Stempeluhr, Offene Tickets, Team-Arbeitszeit. 6 Mitarbeiter mit unterschiedlichen, konsistenten Wochenstunden via MSW (kein Client-Seeding), Balken über/unter Ziel eingefärbt.
- [x] **D-5 — Cross-Module/Alerts + DnD**: „offene Aufgaben" toten Link `/work/tasks` → `/work/my-tasks` korrigiert. Edit-Modus zeigt Grip- + Resize-Handles + X + „Widget hinzufügen" pro Widget.

## Worauf besonders achten
- Layout im Edit-Modus umbauen → Reload → bleibt der Stand?
- Persönlich- ↔ Team-Umschalter durchklicken; alle Widgets gefüllt, keine leeren Karten?
- Alle Widget-Zeilen/Links: führen sie zum richtigen Ziel (keine toten Links)?

## Bekannte offene Punkte (NICHT neu melden)
- **Abwesenheiten-Widget war leer** (HR-Pipeline-Mismatch: `select: data.entries` vs. Handler `{absences}` + Feld-Mismatch + Duplikat-Handler). Das war ein Cross-Lane-Befund **vor** dem team-Tiefe-Pass — **team TM-1** hat den Datenvertrag repariert und meldet das Widget jetzt als gefüllt (Markus + 2× abwesend). **Bitte auf aktuellem `main` gegenprüfen:** zeigt das Widget jetzt Personen? Falls weiter leer → als Befund notieren (P1), falls gefüllt → erledigt.

## Out of scope (kein Mangel)
- Echte Backend-Persistenz der Layouts, echte Lizenz-Flags (Gating ist demo-fail-open).

## Befunde
> Format pro Zeile: **Schweregrad** (P0 / P1 / P2) · **was** · **wo** (Datei/Screen) · **Repro**.

_(noch keine — von Nico zu füllen)_
