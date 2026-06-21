# Batch-NEXT · Paket B (SUB-Terminal) — Dokument-Block-Engine: Spezialblöcke + Diagramm-Datenquellen

> **10-Phasen-Autonom-Batch** ([[feedback_ten_phase_autonomous_batch]]). Dieses Paket fährt das **Sub-Terminal** (`KMU-Hub-review`, Port 5174).
> **Disjunkt zu Paket A** (mails, Haupt-Terminal). Berührt NUR `components/shared/document/**` + die Block-Integration in `modules/wiki` / `modules/berichte`, i18n-Keys `document.*` / `blocks.*`.

## Ziel
Die gemeinsame **Block-Engine** (`components/shared/document`, [[project_document_engine]]) um die fehlenden **Spezialblöcke** + einen **Diagramm-Block mit kaskadierender Datenquellen-Auswahl** erweitern. Mehrwert sofort in **wiki UND berichte** (beide laufen auf der Engine).

## Ist-Stand (verifiziert 2026-06-22)
- API sauber: `defineBlock(def)`, `buildRegistry(defs)`, `resolveBlockDef` in `block-registry.ts`; `BlockTypeDef`/`BlockEditProps`/`BlockViewProps`.
- **Vorhandene Block-Typen** (`blocks/CoreBlocks.tsx`): `text`, `heading`, `bullet`, `callout`, `divider`, `image`.
- **Fehlend** (= dieses Paket): `toggle`, `code`, `table`, `attachment`, `columns`, `quote`/`bookmark`, `chart`.
- Erweiterungspunkt steht (wiki Phase B `wikiBlockRegistry`, berichte `report-sources`).

## Recherche-Auftrag (VOR dem Bauen — Gate)
1. `defineBlock`-Vertrag genau lesen: wie werden `edit`/`view`/`serialize`/`toHtml` erwartet? Muster an `CoreBlocks.tsx` (callout/image) abgucken.
2. Wie hängen wiki (`wiki-blocks.tsx`) und berichte ihre Registries zusammen — wo neue Defs registrieren, damit beide sie bekommen?
3. lowlight ist im Projekt vorhanden (vendor-editor) — für Code-Block-Highlighting wiederverwenden statt neu ziehen.
4. Datenquellen-Block: gibt es aus berichte `report-sources` schon Feld-Metadaten/Series, die der Chart-Block wiederverwenden kann (keine Doppelarbeit zum berichte-Erstellen-Builder E-1)?

## Gate-Fragen an Darien (gebündelt stellen, dann bauen)
- **Block-Set-Priorität:** Reihenfolge/Auswahl der 7 Blöcke — alle, oder Fokus auf die 4 „echten" Spezialblöcke (Toggle/Code/Tabelle/Anhang)?
- **Diagramm-Block (DB-7/8):** soll der in die Block-Engine, ODER ist das doch Teil des berichte-Erstellen-Builders (E-1) und hier nur der „Block-Wrapper"? (Überschneidung klären — sonst Doppelarbeit.)
- **berichte-Review-Abhängigkeit:** Dein berichte-Review steht noch aus. Block-Engine NUR additiv erweitern (neue Defs), KEINE berichte-Review-Findings vorwegnehmen — ok so?

## Phasen (vorläufig — beim Gate verfeinern)
- [ ] **DB-1** Toggle/Aufklapp-Block (`defineBlock`: Heading + collapsible Body, Edit+View)
- [ ] **DB-2** Code-Block (lowlight Syntax-Highlight, Sprachwahl-Dropdown, Copy-Button)
- [ ] **DB-3** Tabelle-Block (Zeilen/Spalten add/remove, Inline-Edit, Header-Zeile)
- [ ] **DB-4** Anhang/Datei-Block (Datei-Picker, Icon+Name+Größe, Download; MSW)
- [ ] **DB-5** Spalten/Columns-Layout-Block (2–3 Spalten, je verschachtelter Inhalt)
- [ ] **DB-6** Quote/Zitat-Block + Bookmark/Embed-Block (URL→Vorschaukarte, `shared/LinkPreviewPopover`-Muster)
- [ ] **DB-7** Diagramm-Block I: kaskadierende Datenquellen-Auswahl (Modul→Feld-Picker→Viz-Typ), MSW Feld-Metadaten je Modul
- [ ] **DB-8** Diagramm-Block II: recharts Live-Vorschau (Balken/Linie/Fläche/Donut/KPI) + Zeitraum-Selektor
- [ ] **DB-9** Block-Picker/Slash-Menü-Integration aller neuen Blöcke + Reorder (DnD) + „in anderen Block konvertieren"
- [ ] **DB-10** i18n ×4 + Demo-Tiefe in BEIDEN Verwendern (wiki + berichte) + Playwright-QA + Screenshots WIRKLICH ansehen

## Disjunktheit / Konflikt-Vermeidung
- **NUR** `components/shared/document/**` + Block-Registrierungspunkte in `modules/wiki` / `modules/berichte` anfassen.
- **Finger weg von** `modules/mails/**`, `mocks/handlers/email.ts` (= Paket A).
- i18n-Dateien (`de/en/fr.json`): nur `document.*` / `blocks.*`-Keys → rebased additiv mit Paket A.
- **wiki Phase B / berichte R-0…R-6 sind review-reif (Darien-Review offen)** — nur additiv erweitern, keine bestehenden Block-Renderings umbauen.

## Build-+-Verify-Standard (pro Phase, [[feedback_two_terminal_nico]])
bauen → i18n ×4 → gescopter tsc (foreground, echter Exit, NIE `| tail`) → `eslint src/ --quiet` ([[feedback_lint_before_push]]) → Playwright-Screenshot-QA + PNGs ansehen → ein Commit (explizite Pfade) → push (`pull --rebase` über parallele Pushes).
