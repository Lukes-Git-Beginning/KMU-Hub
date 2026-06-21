# wiki Phase B — auf die gemeinsame Block-Engine (HAUPT-Terminal)

> Folgt auf **Phase A** (commit `f72aa2db`): gemeinsame Engine in `components/shared/document/`
> (types, block-registry, CoreBlocks, DocumentBlockEditor, DocumentReader mode print|web).
> berichte läuft transparent darauf. Jetzt: **wiki vom TipTap-Fließeditor auf das Block-System.**
> Kontext/Vision: [[project_document_engine]]. Engine-Doku: `components/shared/document/index.ts`.

## Entscheidungen (Darien, 2026-06-21)
- **Bestehende wiki-Artikel:** NICHT konvertieren — **neu als Block-Dokumente seeden** (frische, reiche Demo-Artikel, die die Block-Typen schön zeigen). Reine Demo-Mocks, kein Verlust echter Daten.
- **@Mention + [[Wikilink]]:** ja, in den gemeinsamen Long-Form-Text-Block integrieren. **NEU/wichtig:** beim **Klick auf eine Verlinkung** öffnet ein kleines **Vorschau-Popover** (wer/was ist das Ziel) mit **„weitere Infos" → Sprung zum Kontakt im Kontakte-Modul** (bzw. zum Ziel-Artikel). Aktuell (berichte) ist die Verlinkung tot — das ist ein modulübergreifendes Feature, das wir hier als Muster bauen (`shared/`).
- **Umfang:** Kern zuerst (dieser Plan, PB-1…PB-5), wiki-Spezialblöcke danach (Batch 2, s.u.).

## Architektur-Leitplanke — der Long-Form-Text-Block
Der Knackpunkt „flüssiges Schreiben" lösen wir so: Der **Text-Block ist ein vollwertiger TipTap** (mit internen Überschriften H1–H3, Listen, Bold/Italic/Link) — Enter = nahtlos neue Zeile im selben Block, kein „+Block"-Klick pro Absatz. Die **Block-Struktur** ist für die *speziellen* Elemente (Bild, Callout, Toggle, Code, Tabelle, Diagramm), die man zwischen den Fließtext setzt. So bekommt man flüssiges Long-Form UND Block-Modularität. Der separate Heading-Block bleibt optional für klare Sektions-Trenner.

## PB-1…PB-5 (Kern-Batch, je ein Commit)
- **PB-1 wiki-Block-Schema + Registry + Seeds.** `wikiBlockRegistry` (= `createCoreBlockDefs()` ohne berichte-only Typen; Long-Form-Text als Haupt-Block). wiki-content-Schema `{rows: DocRow[]}` statt `{html}`. MSW: 5–6 Demo-Artikel NEU als Block-Dokumente seeden (Onboarding, DSGVO, CRM-Anleitung, Backup, Infrastruktur), die Text/Heading/Liste/Callout/Bild zeigen. `wiki-adapter` auf Block-Schema.
- **PB-2 wiki-Editor → DocumentBlockEditor.** `WikiEditor` nutzt `DocumentBlockEditor` (registry=wikiRegistry, `enableColumns` je nach Geschmack — eher an, für „Bild neben Text"). Long-Form-Text-Block (TipTap voll). Save → `content:{rows}`. `WikiRichEditor` (TipTap-Canvas) raus. Cover/Icon-Kopf (WikiIdentityBar) bleibt.
- **PB-3 wiki-Reader → DocumentReader mode='web'.** `WikiArticle` nutzt shared `DocumentReader` web. Drumherum bleibt: `WikiArticleHeader` (Breadcrumbs/Cover/Icon/Meta/Tags), Anhänge, Versionen. **TOC** aus Block-Headings generieren (statt HTML-Parse).
- **PB-4 Verlinkung mit Vorschau-Popover.** @Mention + [[Wikilink]] als Extensions im shared Long-Form-Text-Block. **Klick auf eine Verlinkung → `shared/`-Vorschau-Popover** (Name/Typ/Kurzinfo + „weitere Infos"-Button → navigiert zum Kontakt im Kontakte-Modul bzw. zum Ziel-Artikel). Muster wiederverwendbar (auch für berichte später).
- **PB-5 Cleanup + Tiefe + QA.** Alte TipTap-Extensions (Callout/Details/Figure/WikiLink/WikiMention) entfernen soweit ersetzt; Versions-Diff auf Block-Basis vereinfachen; Demo-Tiefe-Audit; i18n ×4; Playwright-QA (Editor schreiben, Block einfügen, Lesen, Verlinkung klicken → Popover → Sprung). Screenshots ansehen, mit berichte vergleichen.

## Was vom Sub-Stand (wiki Batch 1 + 2) bleibt (editor-unabhängig)
WikiArticleHeader, WikiIdentityBar (Cover+Icon), WikiToc, WikiAttachments, WikiSearch, WikiSidebar/TreeNode, WikiCategoryDialog, WikiMoveDialog, WikiVersionHistory, WikiShareDialog, WikiTemplateManager/Dialog, WikiTagEditor, WikiEmptyCanvas, `wikiReading.ts`.
**Wird ersetzt:** WikiRichEditor (TipTap-Canvas), WikiEditor-Shell (wird Shell um DocumentBlockEditor), WikiArticle Lese-Body (→ DocumentReader web), die TipTap-Custom-Nodes.

## Batch 2 (danach) — wiki-Spezialblöcke
Toggle/Aufklapp · Code (lowlight) · einfache Tabelle · Anhang/Datei-Block. Je als Registry-Def in `wikiBlockRegistry` + ggf. nach `shared/document/blocks` hochziehen, wenn auch andere Module sie wollen.

## Build-+-Verify pro Phase
bauen → i18n ×4 (`{var}`, ICU-Plural) → gescopter Typecheck (`tsconfig.doccheck.json` um wiki-Dateien erweitern; foreground, echter Exit) → Playwright-QA gegen 5173 (#/wiki) → PNGs WIRKLICH ansehen (mit berichte vergleichen, Verlinkung-Popover testen) → ein Commit (explizite Pfade) → push (pull --rebase über Sub-Pushes).
