# Nächstes Terminal — Plan (Editor-Pivot: G2 + G3, dann Rollout)

> Erstellt 2026-07-25 (Session #28, Context voll). Fresh Terminal: `git pull`, dieses Doc + `BUILD-PROGRESS.md §Editor-Pivot` + `EDITOR-PIVOT-SPEC.md` lesen.
> **Stand:** Helpdesk-Pilot KOMPLETT (P1–P4 + Feedback F1/F2 + G1). **8 Commits lokal auf main, NICHT gepusht** (Darien reviewt lokal; Push = Hetzner-Auto-Deploy, cd.yml scharf → vor Push mit Darien klären).
> **Reihenfolge:** erst **G2**, dann **G3**, dann **direkt weiter mit dem Editor-Rollout** (Rich-Module). Jede Phase: bauen → i18n ×4 → scoped tsc (`tsconfig.customcheck.json`, gefiltert auf geänderte Dateien) → `eslint src/…` → Playwright-QA gegen laufende Dev-App (`QA_BASE=http://localhost:5173`) + **Bilder ansehen** → 1 Commit. Muster + Gates-Details in `BUILD-PROGRESS.md §Editor-Pivot`.

## Muster (etabliert am Helpdesk-Pilot — für alles wiederverwenden)
- Statisches Label editierbar → `<EditableText dkey="i18n.key" />` (Einfach-Klick-Rename).
- Label in einem Control (Reiter/Karte) → `<EditableText … interactive />` (Einfach-Klick navigiert, Doppelklick benennt um).
- Status/Priorität/Typ-Chips → Value-Set: Modul konsumiert `useModuleValueSet(id)` (Label **und** Farbe via `VsChip`), Picker iterieren **aktive** Optionen; Set in `editorModules.valueSetIds` registrieren + in `DEFAULT_VALUE_SETS` (customization.ts) anlegen.
- Reiter/Bereiche an/aus → `editorModules.areas` + Modul filtert per `useModuleAreas(moduleKey)[key] !== false`.
- Option löschen → Trash im WertelistenPanel (Basis-Option = `active:false` + `valueSetMigrations`; Modul remappt Datensätze via `useValueSetMigration`).
- Mutationen/rausführende Aktionen → `guard(handler)` (`useEditorGuard`); State-Navigation (Reiter/Detail öffnen) NICHT guarden (begehbar).
- **`VsChip` (aktuell lokal in `HelpdeskPage.tsx`) beim ersten Rollout-Modul nach `components/shared/` heben** und von Helpdesk + neuen Modulen importieren.

---

## G2 — „Felder" verständlich machen (Variante A, empfohlen)

**Problem:** `FelderPanel` ist ein voller Custom-Field-Editor, aber definierte Felder erscheinen NICHT in der Vorschau (Ticket-Detail rendert nur `ticket.customFields`, wenn ein Ticket Werte hat) → abgekoppelt, Darien: „keinen Plan was man anpasst".

**Plan (Variante A — Felder edit-in-place sichtbar):**
1. Im **Ticket-Detail** (`TicketDetailPanel` in `HelpdeskPage.tsx`) eine **„Zusatzfelder"-Sektion** rendern, die die DEFINIERTEN Felder für `helpdesk_ticket` zeigt (nicht nur die mit Werten) — leere mit Platzhalter/„—". Quelle: die effektive Feldliste (Draft-Snapshot ⊕ Live-Store; im Editor über den Draft, live über `useCustomFields('helpdesk_ticket')`). So schlägt Anlegen/Umbenennen im Felder-Panel **sofort im geöffneten Ticket** durch (edit-in-place-Konsistenz).
2. Feld-**Labels** im Detail via `EditableText`? — Nein: Feld-Namen werden über das Felder-Panel/`FieldEditorModal` bearbeitet (eigene Persistenz). Statt Inline-Rename: Klick auf ein Feld im Detail könnte das Felder-Panel fokussieren (optionaler Polish). MVP = Felder nur SICHTBAR machen (mit Werten/Platzhaltern).
3. Auch im **„Neues Ticket"-Dialog** die Custom-Felder als Eingaben zeigen (dort befüllt man sie) — im Editor via Guard eh no-op, aber für die Live-App korrekt.
4. Panel-**Klarheit:** „Felder" → Label/Beschreibung „Zusatzfelder — eigene Felder, die im Ticket-Formular erscheinen" (i18n). Kein Umbenennen des Codes nötig, nur der sichtbare Titel/Hint.

**Confirm für Darien (am Terminal-Start):** G2-Q4 — sollen Custom-Felder prominente Editor-Dimension bleiben (Empfehlung: ja, Teil der Massanfertigung) oder nachrangig? · Q2 — Felder auch im Neu-Dialog zeigen (Empfehlung: ja).

**Gates:** scoped tsc + eslint + i18n ×4 + `qa-editor-helpdesk-g2.mjs` (Felder-Panel: Feld anlegen → erscheint im Ticket-Detail; Bilder ansehen).

---

## G3 — Wissensdatenbank auf Block-Dokument-Engine

**Ist:** KB-Artikel (`KBArticleDetail` in `HelpdeskPage.tsx`) nutzen `LazyRichTextEditor` (TipTap, HTML in `article.content`). `shared/document` existiert: `DocumentBlockEditor` (edit) + `DocumentReader` (view) + Blocks (`blocks/CoreBlocks.tsx`, `SpecialBlocks.tsx`). Konsumenten: `wiki` (`WikiEditor`/`WikiArticle`/`wiki-blocks.tsx`) + `berichte` (`ReportDocumentEditor`/`berichte-blocks.tsx`).

**Plan:**
1. KB-Artikel-**Editor** auf `DocumentBlockEditor` umstellen, **Viewer** auf `DocumentReader`. Block-Set = **Wiki-Set wiederverwenden** (`wiki-blocks.tsx`) — evtl. leicht reduziert, aber erst 1:1 versuchen.
2. **Datenmodell:** `article.content` wird ein Block-Dokument (JSON) statt HTML. **Bestehende HTML-Artikel** (Seeds + `KB_BODIES`) beim Laden in **einen HTML/Text-Block** wrappen (nahtlos, kein Datenverlust); neue Artikel voll blockbasiert. Adapter-Funktion HTML→Block beim Öffnen.
3. KB-**Erstellen** (neuer Artikel) über denselben Block-Editor.
4. Prüfen: `useUpdateKBArticle`/KB-Types (`api/helpdesk-types`) — `content`-Feld ggf. Block-JSON-tauglich (String bleibt, JSON.stringify).

**Confirm für Darien:** G3-Q3 — nur KB jetzt (Empfehlung) ODER Startschuss, die Block-Engine überall bei „Einträge erstellen" auszurollen (Dokument-Engine Phase B, größer)?

**Gates:** scoped tsc + eslint + i18n ×4 (falls neue Strings) + `qa-editor-helpdesk-g3.mjs` (KB-Artikel öffnen → Block-Editor; neuer Artikel; Bilder ansehen).

---

## Danach — Editor-Rollout (direkt weiter, Darien-Vorgabe)

Sobald G2+G3 durch: **Rollout des Editor-Musters über die Rich-Module** (Reihenfolge nach `MODUL-AUDIT.md`): **finanzen, inventar, einkauf, vertraege, produktion, vermietung, formulare, work** — dann Medium (kalender, zeiterfassung, rapporte, fuhrpark). Pro Modul: `editorModules`-Eintrag (labelKeys/valueSetIds/areas/fieldEntities), Instrumentierung nach Muster, Value-Sets in `DEFAULT_VALUE_SETS` anlegen + Modul konsumiert sie (Label+Farbe), Reiter→areas, Mutationen→guard. **Ab Pilot per Sub-Agents parallelisierbar** (mechanisches Markieren, je Modul eigener QA-Screenshot-Lauf). `VsChip` vorher nach `shared/`. Kontakte = Router-Sonderweg (separat, siehe MODUL-AUDIT).

**Nicht vergessen:** `shared/`-Extraktion von `VsChip`; jede Status-artige Liste = Value-Set (nicht feste i18n-Enums); backend-gaps pflegen (Record-Migration bei Deploy, moduleAreas-Persistenz, Label/Value-Set-Overlay, terminiertes Deploy). RBAC-Zusammenspiel pro Modul mitdenken (Review-Bündel-Vorgabe: Modul + Settings + Editor + Rollen-Konfig).
