# wiki — Review-Paket (Darien, phasenweise)

> **Zur Seite gelegt für Dariens eigenen Review** (NICHT direkt an Nico — wir passen erst noch an).
> Status: **Phase B (PB-1…PB-5) = komplett & verifiziert.** wiki läuft jetzt auf der **gemeinsamen Block-Engine**
> (`components/shared/document`) — genau wie berichte. Du gehst es Phase für Phase durch, trägst Anpassungen
> in die „Anpassungen"-Spalte ein; danach arbeite ich die Findings als FIX-Phasen ab → erst dann an Nico.
>
> **Reviewen:** Dev-Server `cd desktop && npm run dev` (Port 5173) → Modul **Wissen/Wiki** (`/#/wiki`)
> → Artikel öffnen (z. B. „Willkommen im Cosmi-Wiki") → **Bearbeiten**.
>
> **Zentrale Frage über allem:** Fühlt sich Erstellen + Lesen jetzt wie das berichte-Authoring an
> (gleiche Block-Engine, gleicher Seiten-Look)?

## Strang-Kontext
- **Batch 1 (W-1…W-5)** + **Batch 1 Tiefe (WT-1…WT-5)** — Editor-Grundlage, Kategorien, Tags, Versionen, Anhänge/Share, Suche — **bereits von dir reviewt**.
- **Batch 2 (WP-1…WP-5, TipTap-Editor-Premium)** — **von Phase B ERSETZT** (das alte Review-Paket dazu ist obsolet). Drumherum (Cover/Icon/Reader/Vorlagen) blieb.
- **Phase B (PB-1…PB-5, 2026-06-21)** — Umstellung auf die Block-Engine. ← **dieses Paket.** Bauplan: `.planning/wiki-phaseB-VISION.md`.

---

## Phasen-Review-Tabelle (Phase B)

| Phase | Was gebaut wurde | Klick-Pfad zum Prüfen | Wichtigste Prüfpunkte | Anpassungen (Darien) |
|---|---|---|---|---|
| **PB-1** Block-Engine-Umstellung (`4b92105d`) | wiki läuft auf `components/shared/document` (wie berichte): rows-Schema + Adapter (`extractRows`/`rowsToHtml`/`htmlToRows`), 8 Demo-Artikel als Block-Dokumente, Editor→`DocumentBlockEditor`, Reader→`DocumentReader` | Artikel öffnen → **Bearbeiten** | Editor ist der **Block-Editor** (nicht mehr TipTap-Fließtext); 8 Demo-Artikel laden sauber; Lesen/Bearbeiten klar getrennt; fühlt sich an wie berichte-Authoring | |
| **PB-2** Frameless Long-Form + Cover/Icon (`666a6798`) | Seiten-Look statt boxed Karten; Bubble-Menü (H1/H2/H3 + Listen) beim Markieren; Cover-Bild + Icon/Initial-Kopf pro Artikel | **Bearbeiten** → Text markieren · Artikel-Kopf | Kein umrahmter Karten-Kasten; ruhiger Seiten-Look; Bubble-Menü erscheint beim Markieren; Cover/Icon setzbar + konsistent in Editor/Reader/Liste; keine Emojis | |
| **PB-3** TOC + Anhänge + Bild-Picker (`00e24706`) | Block-Heading-TOC (`tocFromRows`/`headingAnchorId`); **Anhänge auch im Edit-Modus**; **Bild-Block per Datei-Auswahl** (statt „Bild-URL") | Artikel mit Überschriften · Anhang-Bereich · Bild-Block einfügen | TOC springt zu Überschriften; Anhänge im Edit sichtbar + verwaltbar; Bild per Datei-Picker (kein URL-Feld mehr) | |
| **PB-4a** Link-Vorschau (`a30ba713`) | `shared/LinkPreviewPopover`: Klick auf `[[Wikilink]]`/`@Mention` → Vorschau-Karte → Sprung. `RichTextEditor` `extraExtensions`-Prop (Links beim Edit erhalten) | Reader/Editor → `[[Link]]` bzw. `@Mention` anklicken | Vorschau-Popover erscheint; Sprung zum Ziel funktioniert; sauber in Lesen **und** Bearbeiten | |
| **H1-Fix** Größen-Umschalter (`26640b59`) | kryptisches „H1"-Pill → klarer **„ÜBERSCHRIFT [1][2]"**-Größen-Umschalter | Editor-Toolbar | Beschriftung klar verständlich; H1/H2 sauber umschaltbar | |
| **PB-4b** Inline-Autocomplete (`714fa3d2`) | `[[` → Artikel-Picker, `@` → Personen-Picker (`WikiSuggest`, zero-dep, Notion-Stil) | Im Editor `[[` bzw. `@` tippen | Picker erscheint inline + tastatur-navigierbar; Auswahl fügt Wikilink/Mention korrekt ein | |
| **PB-5** Versions-Diff + Cleanup (`3923f200`) | tote Extensions (Callout/Details/Figure/lowlight) entfernt; `adaptVersion` projiziert Block-Versionen→HTML → Versions-Diff funktioniert wieder | Artikel → **Verlauf/Versionen** → zwei Versionen vergleichen | Versions-Diff zeigt Änderungen (Block-Snapshots); kein toter Code/Bruch | |

---

## QA-Screenshots / Skripte (zum Querlesen, ohne App)
- QA-Skripte: `desktop/scripts/qa-wiki-pb1.mjs` … `qa-wiki-pb5.mjs` (Re-Run: Dev-Server 5173, dann `node scripts/qa-wiki-pb<n>.mjs`).
- CI-Fix `3aefb538`: toter `DocumentReader`-Import (Phase A) + `WikiRichEditor` ref-in-render entfernt, `WikiRichEditor` gelöscht.

## Direkter Vergleich zum Vorbild
Stell beim Review **wiki-Editor neben berichte-Editor** (Modul Berichte → „Neuer Bericht" → „Block einfügen"):
gleiche Block-Engine, gleiche Ruhe, gleiches „leerer-Canvas"-Gefühl? Differenzen → Findings.

## Offene Punkte / bewusst gemockt (🔒 = Luke-Backend)
- **P1** Backend-Swap (TanStack Query) 🔒 — FE läuft auf Zustand/MSW, swap-ready (im FE-Review nicht nötig).
- **Batch 2 Spezialblöcke** (Toggle/Code/Tabelle/Anhang) — **NICHT** Teil von Phase B; kommt als Block-Engine-Erweiterung (`.planning/batch-NEXT-B-blockengine.md`). wiki bekommt sie automatisch, da gemeinsame Engine.
- **P3** Zugang/Freigabe (Public-Modus, Kategorie-RBAC) 🔒 — Share-Verwaltung (WT-4) da, Rest Luke.
- **P4** Suche + KI (Artikel-aus-Ticket) 🔒.

## Findings-Sammelstelle (während des Reviews füllen)
> Pro Finding: `[Phase] Kurzbeschreibung — gewünschtes Verhalten`. Ich arbeite die Liste danach als FIX-Phasen ab.

- [ ] …
- [ ] …
