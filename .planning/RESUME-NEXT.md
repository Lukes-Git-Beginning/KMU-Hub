# RESUME — nächster Einstieg (Stand 2026-06-21, Session-Ende)

> **Direkt-Wiedereinstieg.** main = `f72aa2db` (Phase A gepusht), working tree clean, alles gepusht.
> SCHRITT 0 (beide Terminals): `git pull --rebase origin main`

## Was fertig ist (diese Session)
- **berichte R-6 (Datenquellen) KOMPLETT** → berichte Report-Authoring R-0…R-6 fertig. **review-reif → Darien-Review** (Paket `.planning/berichte-REVIEW-paket.md`). Darien reviewt selbst, passt dann 1-2 Settings + Diagramm-Verlinkungen mit mir an.
- **wiki Batch 2 (Sub, WP-1…WP-5)** gebaut, aber **Darien-Befund: „nicht gut" — der Sub hat den wiki-TipTap-Fließeditor verschönert statt das berichte-Block-System zu übernehmen.** Wird in Phase B ersetzt. Review-Paket `.planning/wiki-REVIEW-paket.md` (Drumherum bleibt).
- **★ Dokument-Engine Phase A KOMPLETT** (commit `f72aa2db`): gemeinsame Block-Engine `components/shared/document/` (types, block-registry, CoreBlocks, DocumentBlockEditor, DocumentReader mode print|web). **berichte läuft transparent darauf** — Editor + Reader screenshot-identisch verifiziert. Kontext/Vision: [[project_document_engine]].

## ▶▶ NÄCHSTER BATCH — 2 Terminals parallel
**Haupt-Terminal (`KMU Hub`, 5173) = wiki Phase B (Kern, PB-1…PB-5).**
- Bauplan: **`.planning/wiki-phaseB-VISION.md`** (alle Entscheidungen drin). Engine-Doku: `components/shared/document/index.ts`.
- Kurz: wiki vom TipTap-Fließeditor auf die Block-Engine. Demo-Artikel NEU als Block-Dokumente seeden. Long-Form-Text-Block = vollwertiger TipTap (flüssiges Schreiben) + Block-Struktur für Spezial-Elemente. @Mention/[[Wikilink]] mit **Vorschau-Popover beim Klick** (Ziel-Info + „weitere Infos" → Kontakte-Modul) — NEUES Feature, modulübergreifend. Spezialblöcke (Toggle/Code/Tabelle/Anhang) = Batch 2 danach.

**Sub-Terminal (`KMU-Hub-review`, 5174) = notifications zu review-reif (N-1…N-5).**
- Copy-paste-Paket: **`.planning/notifications-SUB.md`** (fertig, Block unten in der Datei). Disjunkt zu wiki.
- Kurz: Quiet-Hours/DND durchsetzen, Store-Kohärenz, Sidebar-Nav + Demo-Seeds, Pin/Dismiss persistieren, Settings + Schlusscheck.

**Terminal-Zuordnung:** Darien startet beide. Wenn der Sub läuft → Haupt beginnt PB-1. (Wer am cwd: `KMU Hub`=Haupt/wiki, `KMU-Hub-review`=Sub/notifications.)

## Danach
- wiki **Batch 2**: Spezialblöcke (Toggle/Code/Tabelle/Anhang) in `wikiBlockRegistry`.
- berichte/wiki **Darien-Reviews** → Findings als FIX-Phasen → dann Nico.
- **Großer Block (gemeinsam):** kaskadierende Datenquellen-Auswahl für Diagramm-Blöcke (Modul→Kategorie→Filter z.B. Kunde/Branche→Wert) → in alle Dokument-Erstellungs-Stellen. Siehe [[project_document_engine]].

## Build-+-Verify-Standard (beide Terminals, pro Phase)
bauen → i18n ×4 (`{var}`, ICU-Plural) → gescopter Typecheck (foreground, echter Exit, NIE `| tail`) → Playwright-Screenshot-QA + PNGs WIRKLICH ansehen → ein Commit (explizite Pfade) → push (pull --rebase über parallele Pushes).
Latenz-Hinweis: vorbestehender tsc-Fehler in `useTasks.ts` ignorieren (nicht unsere Datei).

## Pläne/Pakete
`.planning/wiki-phaseB-VISION.md` · `.planning/notifications-SUB.md` · `.planning/berichte-REVIEW-paket.md` · `.planning/wiki-REVIEW-paket.md` · `.planning/MASTER-TRACKER.md`. Detail-Verlauf: [[project_resume_log]].
