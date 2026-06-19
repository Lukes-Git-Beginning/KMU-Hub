# Main-Terminal — berichte MSW-Ausbau + Demo-Tiefe (B-1 … B-5)

> **Main-Terminal, Hauptklon `…/KMU Hub`, Dev-Port 5173, Branch `main`.** Ich baue **nur berichte**. notifications gehört dem Sub (`parallel/notifications`) — fass es nicht an. Lies zuerst `.planning/parallel-batch/README.md` (Lane-Regeln, Gates).

## Ausgangslage (Ist-Abgleich 2026-06-19)
berichte ist **~1 von 4 Tabs lebendig**. `BerichtePage.tsx` hat 4 Tabs: `dashboard` (DashboardGrid — 9 KPI-Cards mit recharts-Sparklines, lebt), `erstellen` (ReportBuilder — Dropdown **leer**), `geplant` (ScheduleList — **EmptyState dauerhaft**), `datev` (DatevView — **leer/„nicht konfiguriert"**). recharts ist real im Einsatz (LineChart/BarChart/ResponsiveContainer in DashboardGrid + KPICard).

**Kernproblem:** `mocks/handlers/berichte.ts` registriert **nur** `GET /api/v1/berichte/kpis` (9 DEMO_KPIS). Es fehlen Handler für `/definitions`, `/definitions/:id/run`, `/definitions/:id/export`, `/schedules` (CRUD). → 3 von 4 Tabs sind im Demo tot, Hero-Charts bleiben nach „Laden"-Klick leer. `berichteHandlers` ist bereits in `handlers/index.ts` eingetragen → **kein index.ts-Touch**, nur `berichte.ts` ausbauen.

i18n: alle 4 Sprachen haben ~99 `berichte.*`-Keys (Parität ok), aber mehrere `defaultValue`-Fallbacks im Code (`berichte.chart.noData`, `berichte.chart.laedt`, `berichte.datev.laedt`, `berichte.datev.noRows`) fehlen in den JSONs. Modul-Settings: **kein** berichte-Eintrag in `module-settings-registry.tsx`. Keine `shared/SortMenu`-Nutzung.

## Workflow pro Punkt
bauen → i18n ×4 (`{var}`, ICU-Plural) → MSW-Daten → Compile-Gate (`npm run build > /tmp/build.log 2>&1; echo "EXIT $?"` + `grep -i error`, **nie `| tail`**) → Playwright-QA gegen **:5173** + Bilder ansehen → commit + push auf `main` → Eintrag in `qa-berichte.md`.

---

### B-1 — MSW-Vollausbau (Kern, Blocker)  `[FOUNDATION]`
**Ist:** Nur `GET /berichte/kpis` gemockt. `ReportBuilder.tsx` ~L83–91 zeigt leeres Dropdown (`useDefinitions` ohne Handler), „Bericht generieren" dauerhaft disabled. `ScheduleList.tsx` ~L261–268 dauer-EmptyState. `DatevView` ~L80–88 „noch nicht konfiguriert".
**Soll:** `mocks/handlers/berichte.ts` ausbauen (stateful, in-memory):
- `GET /definitions` → 5 System-Berichte (z. B. Umsatzverlauf, Offene Tickets, Lagerwert, Gewinnrate, BWA) mit Feldern/Kategorien.
- `POST /definitions/:id/run` → Dummy-Zeitreihe + Aggregat-Werte (für Hero-Charts + DATEV-Tabelle).
- `POST /definitions/:id/export` (Format PDF/XLSX/CSV) → echte Blob-Response.
- `GET/POST/PUT/DELETE /schedules` → 2–3 vorbefüllte Demo-Schedules, CRUD stateful.
Prüfe die Client-Typen (`api/`-Client + Hooks) auf das erwartete wire shape, Handler exakt daran angleichen (Lehre aus automatisierung A-1: Vertrag muss matchen, sonst still tot).
**Verify:** Erstellen-Tab Dropdown gefüllt + „generieren" aktiv; Geplant-Tab zeigt Schedules; DATEV-Tab zeigt Tabelle; 0 Raw-Keys.

### B-2 — Hero-Charts auto-laden + Drilldown-Modal + DATEV-Toggle  `[PATTERN]`
**Ist:** DashboardGrid ~L191–204 Hero-Charts „Noch keine Daten geladen." (manueller „Laden"-Klick, kein run-Handler). Drilldown ~L163–177 öffnet nur einen Inline-Text-Platzhalter (`berichte.dashboard.drilldownPending`), **kein Modal**. `DatevView.tsx` ~L96–107 BWA/SuSa-Tab-Buttons haben **kein onClick** (Variante immer `availableVariants[0]`).
**Soll:**
- Hero-Charts bei Tab-/Mount **auto-laden** (nutzen jetzt den B-1-run-Handler), kein manueller Klick mehr nötig.
- Drilldown beim KPI-Klick → `shared/DetailModal` (zentriert, sticky Close) mit Mini-Zeitreihe (recharts) + Kennzahlen-Tabelle. Ganze KPI-Card klickbar (`role=button` + Keyboard), innere Controls `stopPropagation`.
- DATEV BWA/SuSa-Toggle reparieren: `onClick` setzen → echter Variantenwechsel.
**Verify:** Hero-Charts rendern ohne Klick; KPI-Klick → zentriertes Modal mit Chart; BWA↔SuSa wechselt sichtbar; Close sticky beim Scrollen.

### B-3 — Schedules stateful + Alerts-UI
**Ist:** Nach B-1 lädt ScheduleList Daten; Toggle/Delete müssen stateful wirken. Alert-Schwellwerte fehlen komplett.
**Soll:**
- Schedule-Toggle (aktiv/pausiert) + Delete **stateful** (wirken sofort + überleben in der Session). „Nächster Lauf"-Spalte befüllen (aus Cron-Ausdruck grob berechnet, Demo).
- Create-Dialog um ein **Alert-Schwellwert-Feld** ergänzen (z. B. „benachrichtige wenn Wert > X") — rein FE/Demo, kein Backend nötig.
**Verify:** Schedule anlegen → erscheint; Toggle → Status wechselt; Delete → weg; Alert-Feld speichert in den Demo-State.

### B-4 — Sortierung + Modul-Einstellungen
**Ist:** Keine `shared/SortMenu`-Nutzung; `module-settings-registry.tsx` hat keinen berichte-Eintrag.
**Soll:**
- `shared/SortMenu` in ScheduleList (Felder Name / Zeitplan / Status, Richtung asc/desc).
- Modul-Settings-Eintrag in `module-settings-registry.tsx` (`id: 'berichte'`, navMatch, Icon) mit `ModuleSettingsShell`:
  - **personal:** Standard-Export-Format (PDF/XLSX/CSV) + Standard-Zeitraum.
  - **tenant:** erlaubte Export-Formate + E-Mail-Domains für geplante Berichte (admin-gated).
  - Mindestens eine personal-Pref **real anwenden** (z. B. Default-Format im ReportBuilder vorbelegen).
  - ⚠ Koordination: Sub trägt zeitgleich `id: 'notifications'` in **dieselbe** Registry ein — auf `main` bauen, finaler Merge behält beide (siehe README Regel 2).
**Verify:** SortMenu reordert Schedules beidseitig; Einstellungs-Fenster zeigt „Berichte"-Eintrag mit personal/tenant; Default-Format wirkt.

### B-5 — i18n-Bereinigung + Demo-Tiefe-Schlusscheck
**Soll:**
- Alle `defaultValue`-Fallback-Keys in die JSONs übernehmen (`berichte.chart.noData`, `berichte.chart.laedt`, `berichte.datev.laedt`, `berichte.datev.noRows` + alle weiteren, die der Code per `defaultValue` setzt) — ×4 Sprachen.
- Hardcoded Placeholder i18n-ieren (z. B. `ScheduleList.tsx` ~L314 `placeholder="Monatlicher Umsatzbericht"`).
- Sweep: keine Toast-only-Stubs / toten Buttons mehr; alle 4 Tabs ×4 Sprachen (1440 + 1024) Screenshot-QA, leere + gefüllte Zustände, EN-Umschalten sauber.
**Verify:** 0 Raw-Keys / 0 `{{var}}` / 0 Console-Errors über alle 4 Tabs, DE+EN.

---

## Out of scope (NICHT in diesem Batch — 🔒 Luke)
- **P2** echter No-Code-Query-Builder (modul-übergreifend) — bleibt der System-Definition-Picker.
- **P4** echtes DATEV-Backend + externe BI — DATEV-Tab zeigt Demo-Daten aus MSW.
- Verschiebbare Dashboard-Kacheln (DnD) — optional späterer Tiefe-Punkt, nicht hier.

## Definition of Done (berichte review-reif)
Alle 5 Punkte verifiziert (Screenshots angesehen), 4 von 4 Tabs lebendig, 0 Raw-Keys / 0 `{{var}}` / 0 Console-Errors, jede Phase ein Commit+Push auf `main`, `qa-berichte.md` gepflegt. Danach `.planning/reviews/berichte.md` für Nico anlegen (Muster wie die 6 bestehenden reviews/-Dateien).
