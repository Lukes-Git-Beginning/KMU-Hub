# Handoff — work-Quick-Actions DONE, berichte-Erstellen-Builder offen

> **Stand 2026-06-19 nachts.** main=`a046b0d9` (sauber, gepusht). Build grün. wip-Branch gelöscht.

## ✅ Fertig + gepusht (main)
- **work Quick-Actions W-1…W-3** (`a046b0d9`, Squash-Merge): MyTasks-Zeilen als Geschwister-Buttons (kein `role=button`-Div mehr — das schluckte den Accessible-Name des Menü-Buttons → Menü navigierte statt zu öffnen). Quick-Complete-Checkbox + Aktions-Menü (Erledigen/Mir zuweisen/Fälligkeit/In Projekt verschieben/Löschen) + Löschen-Confirm. **Kanban-Karten** dieselben Quick-Actions (Checkbox + Hover-Menü, dnd-sicher via `data-card-control`-Guard + `onPointerDown`-Stop). **Task-Detail-Panel + -Page** Löschen-Button mit Confirm. 14 i18n-Keys ×4. QA: `qa-work-mytasks.mjs` + `qa-work-kanban.mjs` (alle grün, Screenshots verifiziert).
  - **tsc-Crash GEFIXT** (`87868ade`): Der `Debug Failure. No error for last overload signature`-Crash kam von `i18next.d.ts` `resources: typeof de.json` (~8.5k Keys) × überladenes `t()` (TS-Bug #63195). Fix: `resources`-Typ weggelassen → `t()` = `(key)=>string`. **tsc läuft wieder durch.** Trade-off: keine compile-zeitliche t()-Key-Validierung (Gate bleibt Screenshot-QA). Deckt jetzt ~31 **vorbestehende** latente Typfehler im work-Modul auf → `.planning/tsc-latent-type-errors.md` (Aufräum-Aufgabe, keiner aus dem Quick-Actions-Code).
- **berichte B-1…B-5** + **F1/F2 Live-Fixes** (Schedule→DetailModal, KPI-Sparklines+Tooltip).
- **notifications N-1…N-5** gemergt + reviews-Dateien.

## 🔧 OFFEN — berichte „Erstellen"-Builder (Darien: viel mehr Tiefe)

Aktueller „Erstellen"-Tab ist flach (nur System-Bericht-Dropdown + Format + Zeitraum). **Marktanalyse-Ergebnis (Metabase/HubSpot/Looker Studio) → 5-Phasen-Roadmap, FE-mock-first** (echter Query-Executor = Luke 🔒):
- **E-1 — Feld-Picker + Viz-Switcher (Kern):** Modul wählen → Felder als Checkbox-Liste → Visualisierungstyp-Picker (Tabelle/Balken/Linie/Donut/KPI) mit sofortiger recharts-Live-Vorschau + Zeitraum-Selektor. Bericht benennen + als eigene Definition speichern (MSW-stub).
- **E-2 — Filter-Builder (Kern):** typ-aware Filter (Feld → Operator is/contains/>/</between → Wert), bis 5 Filter mit AND/OR, Filter-Chips, Vorschau reagiert.
- **E-3 — Aggregation + Grouping (Kern):** Dimension vs. Measure, Aggregation (Count/Sum/Avg/Min/Max), Group-by bis 2 Dimensionen, Pivot-light. `query_config` ausdehnen, MSW liefert passende series/totals.
- **E-4 — Speichern + Bibliothek + Dashboard-Pin:** „Meine Berichte"-Liste (benennen, Modul-Badge, zuletzt bearbeitet), Bericht öffnen → Builder-State wiederhergestellt, „Zu Dashboard hinzufügen". (🔒 echter Persist-Endpoint = Luke; FE-State reicht für Demo.)
- **E-5 — Advanced (Post-MVP):** berechnete Felder (FE-Formel), Scheduled Export 🔒, Sharing 🔒, KI-Viz-Empfehlung (heuristisch FE).
- Visualisierungs-Kern-Set (recharts-baubar): Tabelle · Balken · Linie · Fläche · Donut · KPI-Zahl · Combo · Gauge.

## Vorgehen (Stand jetzt)
1. ✅ **work W-1…W-3 erledigt + in main** (`a046b0d9`).
2. ⏭ **berichte-Erstellen-Builder** (E-1 ff.) als neue Phasen — Scope vor E-1 mit Darien kurz abstimmen (welche Module/Felder im Picker, Viz-Kernset, MSW-Shape).
Dev-Server-Regel: nur 1 auf :5173 (läuft frisch via `npm run dev`), vor Neustart killen (PowerShell). Gate = Vite+Playwright-QA (tsc crasht flaky). add-*-i18n.mjs sind gitignored.
