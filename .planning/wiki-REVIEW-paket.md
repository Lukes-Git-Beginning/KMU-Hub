# wiki — Review-Paket (Darien, phasenweise)

> **Zur Seite gelegt für Dariens eigenen Review.** Anlass: dein Befund nach Batch 1 —
> *„das Erstellen sieht kein bisschen so aus wie beim berichte-Modul"*. **Batch 2 (WP-1…WP-5)**
> hebt den Editor/Reader auf berichte-Niveau. Du gehst es Phase für Phase durch, trägst Anpassungen
> in die „Anpassungen"-Spalte ein; danach arbeite ich die Findings ab.
>
> **Reviewen:** Dev-Server `cd desktop && npm run dev` (5173) → Modul **Wissen/Wiki** (`/#/wiki`)
> → Artikel öffnen (z. B. „Willkommen im Cosmi-Wiki") → **Bearbeiten**.
>
> **Zentrale Frage über allem:** Fühlt sich Erstellen + Lesen jetzt wie das berichte-Authoring an?

## Strang-Kontext
- **Batch 1 (W-1…W-5)** + **Batch 1 Tiefe (WT-1…WT-5)** — Editor-Grundlage, Kategorien, Tags, Versionen, Anhänge/Share, Suche — bereits von dir reviewt.
- **Batch 2 (WP-1…WP-5)** — Editor-Premium, **dieses Paket**. Commits: `3fcc943d` `16806324` `004b87c2` `f2a12ed0` `24415497`.

---

## Phasen-Review-Tabelle (Batch 2)

| Phase | Was gebaut wurde | Klick-Pfad zum Prüfen | Wichtigste Prüfpunkte | Anpassungen (Darien) |
|---|---|---|---|---|
| **WP-1** Editor-Shell + Look | Frameless editorial Canvas, Playfair-Überschriften (report-serif), 65ch, großes Titel-Heading, dezente **sticky** Toolbar | Artikel → **Bearbeiten** | Kein umrahmter Kasten mehr; Titel als großes Editorial-Heading (kein Input-Look); ruhige Hierarchie; Toolbar bleibt beim Scrollen; Read/Edit klar getrennt | |
| **WP-2** Slash-Menü + reiche Blöcke | `/`-Einfügemenü (Icon+Label, tastatur-navigierbar) + Callout / Code (Syntax) / Toggle / Trenner / Bild+Caption | Im Canvas `/` tippen | Menü erscheint + navigierbar; alle Block-Typen einfügbar; Callout-Varianten (Info/Warnung/Tipp/Empfehlung); Code-Highlight; Toggle auf/zu; Read-Modus rendert sauber + DOMPurify-safe | |
| **WP-3** Cover + Icon | Cover-Bild + Icon/Initial pro Artikel (Custom-SVG, keine Emojis) | Editor-Kopf · Lese-Kopf · Artikelliste | Cover setzbar/änderbar; Icon/Initial konsistent in Editor, Reader, Liste; keine Emojis | |
| **WP-4** Editorial-Reader + TOC | Lese-Modus mit Premium-Typo, Breadcrumbs (Bereich › Kategorie › Artikel), Lesezeit, Meta-Leiste, **Inhaltsverzeichnis** (H1–H3, Anker, Scroll-Spy, sticky) | Artikel öffnen (Lese-Modus) | Ruhige Typo; Breadcrumbs korrekt; Lesezeit plausibel; TOC springt + markiert aktiven Abschnitt; sticky bei langen Artikeln | |
| **WP-5** Vorlagen-CRUD + Politur | Eigene Vorlagen (erstellen/bearbeiten/löschen + Vorschau) statt 3 hardcoded; Editor-Empty-State als Joy-Moment | Neuer Artikel → Vorlagen-Auswahl; leerer Editor | Vorlagen-CRUD echt (MSW); Vorschau; Empty-State mit Personality (Custom-SVG/Wording); Demo-Tiefe | |

---

## Mein technischer Verify (unabhängig, gegen 5173)
- ✅ Slash-Menü erscheint, sticky Toolbar vorhanden, Cover gerendert, Reader mit TOC-Spalte.
- ✅ Keine Raw-Keys, keine `{{}}`, keine `�`-Zeichen, keine Page-Errors.
- ✅ Playfair-CSS korrekt definiert (`.wiki-canvas h1/h2/h3` → `'Playfair Display'`, `styles/wiki-content.css`).
- ⚠ **Rein ästhetische Abnahme „wie berichte" = dein Call** — das war dein Ausgangsbefund; ich kann's technisch bestätigen, aber nicht für dich beurteilen.
- Screenshots: `desktop/.qa-screenshots/wiki-batch2-check/` (Reader, Editor, Slash-Menü) + Sub-QA `desktop/.qa-screenshots/wiki/` (wp1…wp5, im Sub-Klon erzeugt).

## Direkter Vergleich zum Vorbild
Stell beim Review **wiki-Editor neben berichte-Editor** (Modul Berichte → „Neuer Bericht" → „Block einfügen"):
gleiche Ruhe? gleiche Typo-Hierarchie? gleiches „leerer-Canvas"-Gefühl? Differenzen → Findings.

## Findings-Sammelstelle (während des Reviews füllen)
> Pro Finding: `[WP-x] Kurzbeschreibung — gewünschtes Verhalten`. Ich mache daraus FIX-Phasen.

- [ ] …
- [ ] …
