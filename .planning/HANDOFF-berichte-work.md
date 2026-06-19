# Handoff — berichte-Fixes done, work-Aufgaben + berichte-Erstellen offen

> **Stand 2026-06-19 spät.** Context-Schnitt für neues Terminal. main=`8e06cd5c` (sauber, gepusht). Build grün.

## ✅ Fertig + gepusht (main)
- **berichte B-1…B-5** (Batch 4 Main) + **F1/F2 Live-Fixes**: Schedule-Zeilen klickbar → DetailModal mit Lauf-Historie; Dashboard-KPI-Sparklines mit echten Werten + Hover-Tooltip. Alle verifiziert.
- **notifications N-1…N-5** (Batch 4 Sub) gemergt + reviews/notifications.md.
- Nico-Review-Dateien: `.planning/reviews/{berichte,notifications,…}.md`.

## 🔧 OFFEN 1 — work „Aufgaben"-Schnellaktionen (Darien-Befund)
**Befund (Ist-Abgleich):** Aktionen existieren in der Projekt-**Liste** (TaskRow, Inline-Popovers) + im **Detail-Fenster**, aber NICHT in **„Meine Aufgaben"** (MyTasksPage, Hauptansicht) und nicht auf **Kanban-Karten**. **Quick-Complete fehlt überall** (man muss Status-Dropdown auf „Erledigt"). **Löschen** hat gar keine UI (Hook+MSW fertig). Alle Mutationen sind MSW-stateful → reiner FE-Fix.

### W-1 — MyTasks Quick-Actions: auf Branch `wip/work-quick-actions` (NICHT in main!)
Gebaut: Quick-Complete-Checkbox links, Aktions-Menü (Erledigen/Mir zuweisen/Fälligkeit-Quickpick/In Projekt verschieben/Löschen), Löschen-Confirm-Dialog; `useUpdateTask` um `completed_at`/`is_closed`/nullable `due_date` erweitert. Build grün.
**⚠ OFFENER BUG:** Klick auf das Drei-Punkte-Menü **navigiert zur Detail-Page statt das Popover zu öffnen** (`detailLeaked: true` im QA, Popover öffnet nie). QA: `desktop/scripts/qa-work-mytasks.mjs` (auf dem Branch).
**Debugging-Hypothesen fürs neue Terminal:**
1. Die ganze Zeile ist `<div role="button" onClick={openTask}>`; der `closest('button')`-Guard im Zeilen-onClick greift nicht zuverlässig mit Radix `PopoverTrigger asChild`. **Sicherste Lösung:** Zeile NICHT als role=button auf dem ganzen Div — stattdessen nur den Titel-/Meta-Bereich als eigenen klickbaren Button, Checkbox+Menü daneben als Geschwister (kein Bubbling-Konflikt).
2. QA: `getByRole('button', {name:/Aktionen/})` zählte **10** statt 5 → evtl. matcht der Selektor zu viel / klickt den falschen Button. i18n-Key `work.myTasks.actions` prüfen (existiert er mit anderem Wert?).
3. Der `role="presentation"`-Wrapper mit `stopPropagation` um den Popover ist evtl. kontraproduktiv → entfernen und Hypothese 1 umsetzen.

### W-2 — Kanban-Karte Quick-Actions (offen, nicht gebaut)
KanbanCard: Quick-Complete-Checkbox + Hover-Aktionsleiste (Fälligkeit, Assignee, Drei-Punkte mit Löschen). Bisher nur DnD + Detail-Klick.

### W-3 — Löschen im Task-Detail (offen, nicht gebaut)
`TaskDetailPanel` + `TaskDetailPage`: Löschen-Button (Confirm) im Footer/Header. `useDeleteTask` ready.

## 🔧 OFFEN 2 — berichte „Erstellen"-Builder (Darien: viel mehr Tiefe, Marktanalyse gemacht)
Aktueller „Erstellen"-Tab ist flach (nur System-Bericht-Dropdown + Format + Zeitraum). **Marktanalyse-Ergebnis (Metabase/HubSpot/Looker Studio) → 5-Phasen-Roadmap, FE-mock-first** (echter Query-Executor = Luke 🔒):
- **E-1 — Feld-Picker + Viz-Switcher (Kern):** Modul wählen → Felder als Checkbox-Liste → Visualisierungstyp-Picker (Tabelle/Balken/Linie/Donut/KPI) mit sofortiger recharts-Live-Vorschau + Zeitraum-Selektor. Bericht benennen + als eigene Definition speichern (MSW-stub).
- **E-2 — Filter-Builder (Kern):** typ-aware Filter (Feld → Operator is/contains/>/</between → Wert), bis 5 Filter mit AND/OR, Filter-Chips, Vorschau reagiert.
- **E-3 — Aggregation + Grouping (Kern):** Dimension vs. Measure, Aggregation (Count/Sum/Avg/Min/Max), Group-by bis 2 Dimensionen, Pivot-light. `query_config` ausdehnen, MSW liefert passende series/totals.
- **E-4 — Speichern + Bibliothek + Dashboard-Pin:** „Meine Berichte"-Liste (benennen, Modul-Badge, zuletzt bearbeitet), Bericht öffnen → Builder-State wiederhergestellt, „Zu Dashboard hinzufügen". (🔒 echter Persist-Endpoint = Luke; FE-State reicht für Demo.)
- **E-5 — Advanced (Post-MVP):** berechnete Felder (FE-Formel), Scheduled Export 🔒, Sharing 🔒, KI-Viz-Empfehlung (heuristisch FE).
- Visualisierungs-Kern-Set (recharts-baubar): Tabelle · Balken · Linie · Fläche · Donut · KPI-Zahl · Combo · Gauge.

## Vorgehen neues Terminal (Darien-Wunsch: „beide nacheinander")
1. **Erst W-1 fixen** (Branch `wip/work-quick-actions` auschecken, Bug nach Hypothese 1 lösen, QA grün, dann W-2 + W-3), nach main mergen.
2. **Dann berichte-Erstellen-Builder** (E-1 ff.) als neue Phasen.
Dev-Server-Regel: nur 1 auf :5173, vor Neustart killen (PowerShell). Build-Gate echter Exit. add-*-i18n.mjs sind gitignored.
