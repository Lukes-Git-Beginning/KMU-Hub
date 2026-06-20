# Berichte → echtes Report-Authoring-System — VISION & Plan (Handoff)

> **Stand 2026-06-20 (aktualisiert nach R-0-Analyse).** Großer Pivot nach Darien-Feedback: Das
> berichte-Modul ist aktuell „nur ein Modul zum Grafiken erstellen". Es soll ein **vollwertiges
> Bericht-Authoring-System** werden — mehrseitige Dokumente (Text + Titel + Grafiken),
> Lebenszyklus, Lese-Modus, professionelle PDF, Scheduling, Versand, Cross-Modul-Integration.
> **Der bisher gebaute Builder (E-0…E-5, `.planning/berichte-builder-plan.md`) ist NICHT weg** —
> er wird zur untersten „Grafik-Widget"-Schicht (ein Baustein im Dokument).
>
> **Stand 2026-06-20 (Batch 1+2+3 fertig):** R-0 ✅ · **R-1a ✅** · **B1-6 ✅** · **R-1b ✅**
> (B2-1…B2-5: Spalten-Zeilen + Breiten-Presets, Chart-Block-Picker Bibliothek+Neu-Tab inkl. inline
> Builder mit Filter/Zeitraum + echte Werte-Datalist, KPI-Block + KPI-Reihe, Callout/Image/Table-Editoren) ·
> **R-2 ✅** (B3-1: Lebenszyklus-Buttons + Guards + „Freigegeben am" + Reader-Empty-State) ·
> **R-2p ✅** (B3-2…B3-5: A4-Bögen-Reader + Akzentlinie + Kopf-/Fußzeile + „Seite X von Y" · Playfair-
> Editorial-Typografie + premium Deckblatt + 65ch · Block-Politur + Cover-eigene-Seite · Print-CSS-Grundlage
> `report-print.css`). Plus Darien-Review-Fixes (Filter/Datalist, Sticky-Header, Mehrspalten-Block-z-index).
> **Nächster Batch (RESUME-batch-3.md): R-3 PDF-Export + R-4 Scheduling.** Danach R-5 (Integration) + R-6 (Datenquellen).

---

## 1. Darien-Vision (O-Ton-Anforderungen, vollständig)

1. **Tab-Benennung:** Der Tab „Erstellen" (nach Dashboard) sollte „Berichte" o.ä. heißen — dort
   liegen ja auch die gespeicherten Berichte.
2. **Automatische Berichte:** Möglichkeit, Berichte automatisch erstellen zu lassen (z.B. jede
   Woche). **Am Bericht selbst** definiert (beim Erstellen/Bearbeiten), NICHT in
   Modul-Einstellungen verstreut.
3. **Feld-/Parameter-Auswahl massiv ausbauen:** Auswählen können, was genau im Bericht erscheint
   (Kunden oder alles Mögliche, was in einem Modul relevant ist) — **für JEDES Modul nachschauen**,
   welche Daten relevant sind.
4. **Berichte öffnen/lesen:** Aktuell kann man keinen Bericht öffnen/lesen. Man muss Berichte
   **ansehen** können (Lese-Modus), nicht nur im Bearbeiten-/Erstellen-Modus.
5. **Echte mehrseitige Berichte (Dokument-Charakter):** 10–20+ Seiten mit **Text, Titel, allem**
   im Tool. Auswählen können, **wo die Grafiken im Bericht erscheinen** und welche Parameter dafür
   wichtig sind.
6. **Lebenszyklus + Verteilung:** speichern · dokumentieren · versenden · als **fertig markieren**
   · an zuständige Personen **freigeben/verschicken**.
7. **Cross-Modul-Integration (Leitbeispiel):** Ein 20-seitiger Verkaufsbericht mit Grafiken wird
   erstellt → als fertig markiert → freigegeben/verschickt → jemand liest ihn → will ihn als
   **Aufgabe** an einen Mitarbeiter geben („analysieren + Verkaufsmodell anpassen") → **Bericht an
   die Aufgabe dranhängen** (dafür muss er ein Dokument sein ODER beim Aufgabe-Erstellen anhängbar)
   → vielleicht geht der Bericht auch an **jemanden ohne Cosmi** → dann muss es eine **1a-PDF**
   sein, die wie ein vollständiger echter Bericht aussieht.

---

## 2. Architektur — 5 Schichten

```
5. LEBENSZYKLUS + VERTEILUNG  Entwurf→Fertig→Freigegeben→Archiviert · 1a-PDF-Export
                              · an Aufgabe anhängen · → dokumente · extern teilen
4. BERICHT-DOKUMENT (Editor)  Mehrseitig, Block-basiert: Deckblatt·Titel·Text·
                              Grafik·Tabelle·KPI·Seitenumbruch · Lese↔Bearbeiten
3. GRAFIK-WIDGET  ← E-0…E-5   Eine Visualisierung (Quelle→Felder→Viz→Filter→Style)
   (gebaut)                   wird als wiederverwendbarer Baustein in Blöcke eingebettet
2. DATENQUELLEN-REGISTRY      Felder pro Modul — `report-sources/` — MASSIV ausbauen
   (5 da, ~7 fehlen)          (alle Module, reiche Feldlandkarte → Teil 6)
1. AUTOMATISIERUNG            „jede Woche erstellen + an X schicken" — am Bericht
   (CRUD da, Kopplung trivial)
```

---

## 3. Markt-Recherche-Synthese (Metabase/PowerBI/Looker/Notion/JasperReports)

- **Editor-Paradigma → BLOCK-basiert** (wie Notion), NICHT Canvas (Power BI Report Builder /
  Crystal Reports). Begründung: KMU-Anwender schreiben einen Monatsbericht wie einen Word-Brief
  (oben anfangen, runterarbeiten). Canvas = Support-Hölle (Pixel-Positionierung). Dokument fließt
  top-down, Blöcke per „+" einfügen, automatische Paginierung beim PDF-Export, Seitenumbruch als
  expliziter Block.
- **MVP-Block-Set (11):** Deckblatt (Titel/Logo/Datum/Autor) · H1 · H2 · Fließtext (Rich) ·
  **Chart (= eingebettetes E-1-Widget)** · Datentabelle · Kennzahl-Highlight (große Zahl + Trend)
  · Callout/Empfehlung · Aufzählung · Trennlinie · Seitenumbruch · Bild/Logo. Kopf-/Fußzeile mit
  Seitenzahl = globales Report-Setting (kein Block). **Phase 2:** Auto-Inhaltsverzeichnis (aus
  H1/H2), Kommentare. *(Callout und Bullet sind bereits im Typ-System vorhanden — Codepfad
  günstiger als ursprünglich geschätzt.)*
- **Lebenszyklus → 4 Status, KEIN Approval-Gate** (zu bürokratisch für KMU):
  `Entwurf → Fertig → Freigegeben → Archiviert`. Entwurf = nur Ersteller bearbeitet, kein Versand.
  Fertig = intern lesbar. Freigegeben = read-only für Berechtigte, Scheduling aktiv, externer Link
  generierbar. **Snapshot** bei Freigabe + bei jedem Schedule-Run (Datum+Lauf-ID). Keine
  Git-artige Versionierung.
- **Lese- vs. Bearbeiten-Modus:** zwei klar getrennte States. Lese-Modus = sauberes Dokument ohne
  UI-Chrome (wie PDF-Vorschau), auch das was externe Empfänger über Share-Link sehen.
  PDF-Download-Button prominent im Lese-Modus.
- **PDF-Strategie → Playwright Headless Chromium SERVER-SEITIG (= Luke).** CSS
  `@page { size:A4 }`, `page-break-inside:avoid` (keine abgeschnittenen Charts/Tabellen),
  `@page:first` (Deckblatt ohne Kopfzeile), Plus-Jakarta-Sans eingebettet, SVG-Charts
  (vektorbasiert). `mini-pdf.ts` reicht NICHT (text-only, single-page). Paged.js optional für
  laufende Kopf-/Fußzeilen. **Was eine 1a-PDF ausmacht (Prio):** Deckblatt > durchgehende
  Kopf-/Fußzeile mit Logo+„Seite 3 von 8" > keine abgeschnittenen Blöcke > Inhaltsverzeichnis mit
  Seitenzahlen (Phase 2) > eingebettete Schriften.
- **Scheduling:** am Bericht konfiguriert (Rhythmus täglich/wöchentlich/monatlich, Empfänger
  intern+extern, Format PDF). Guard: nur Status ≥ Freigegeben wird versendet. Snapshot pro Run.
- **Verteilung:** intern (rollenbasiert, in Modul-Liste) · extern (Token-Read-Link, kein Login,
  optional Ablauf/Passwort, **geteilt mit formulare `public_tokens`-Mechanik** — gleiches
  Token-Muster, eigene Route) · anhängen an Aufgabe/Deal/Kontakt (Referenz-Link, nicht
  Datei-Kopie → kein Versionschaos).
- **Bewusst weglassen (KMU-MVP):** Canvas/Pixel-Layout · Conditional-Formatting/Ausdrucks-Engine
  (SSRS) · daten-getriebene Subscriptions · Kommentar-Threads (Phase 2) · iFrame-Embeds (brechen
  im PDF) · Branching/Merge-Versionierung.

---

## 4. Infrastruktur-Ist (was da ist / was fehlt)

**Schon da (wiederverwenden!):**

- **TipTap-RichTextEditor:** `components/shared/RichTextEditor/RichTextEditor.tsx` (v3.20), voll
  ausgebaut (Toolbar/BubbleMenu, `readOnly`/`compact`-Props, `onChange(html)`, Extensions:
  link/image/table/task-list/text-align/underline/placeholder). **Direkt für Fließtext-Blöcke +
  Lese-Modus nutzbar.** Lazy: `LazyRichTextEditor`.
- **dnd-kit** bereits im Projekt (Formulare-DnD-Builder nutzt es) — direkt für Block-Reorder
  nutzbar, kein neues npm-Paket.
- **dokumente-Upload:** `POST /api/v1/documents/files/upload` (multipart, `folder_id`) → für
  PDF-Ablage. Kein `source_type:'report'`-Feld → via Standard-Tag „Bericht" oder Feld ergänzen.
- **work Task-Files:** `GET/POST /api/v1/tasks/:id/files` (Metadaten-POST, kein Multipart),
  `DELETE /task-files/:id`. Task-Create nimmt KEINEN Anhang direkt → Anhang als 2. Schritt nach
  Erstellen.
- **Scheduling:** `ReportSchedule` (cron, recipients, format, params/alert_threshold, toggle) +
  volle CRUD in `mocks/handlers/berichte.ts`. **`definition_id` koppelt Schedule schon 1:1 an
  einen Bericht** → „Zeitplan einrichten"-Button im Editor = nur `definition_id` vorbelegen, kein
  Schema-Change.
- **Builder E-0…E-5:** Source→Felder→Aggregation→Filter→DateRange→Viz→Style→Preview→Save +
  MyReportsLibrary + Dashboard-Pin. `BuilderQueryConfig` als `query_config` (JSONB).
- **BerichteSettingsPanel:** schon vorhanden mit `personal`- + `tenant`-Bereich
  (Standard-Format/Zeitraum/Palette; erlaubte Formate; Schedule-Domains).
- **MSW-CRUD für ReportDocuments:** vollständig in `mocks/handlers/berichte.ts` — 3 Seed-Dokumente
  (Verkaufsbericht Q2/released, Monatsbericht Juni/final, Helpdesk KW24/draft), CRUD-Endpoints,
  Seeding gut strukturiert mit Row-/Spalten-Helfern.
- **ReportDocument-Typen:** vollständig in `api/berichte-types.ts` — alle 11 Block-Typen als
  Union, Row→Column→Block-Layout-Modell, `ReportDocSettings`, Status-Enum, Input-Typen.

**Fehlt / Lücken:**

1. **Echter WYSIWYG-Block-Editor** (R-1) — `ReportDocumentEditor` ist nur Outline-Shell, kein
   editierbarer Canvas.
2. **Echte PDF-Engine** — kein jspdf/pdfmake/pdf-lib/puppeteer in package.json; nur text-only
   single-page `mini-pdf`. → **Playwright server-seitig = Luke** (kritischster Backend-Bedarf).
3. **Lese-Modus, Lebenszyklus-Guards, Status-Übergangs-Buttons** — fehlen komplett.
4. **Chart-Block-Picker im Editor** — aktuell kein Weg, inline eine neue Grafik anzulegen oder
   aus bestehenden Definitionen auszuwählen.
5. **Vorlagen-System** — 5 Standard-Vorlagen aus Markt-Synthese nicht vorhanden (kein
   `template_id`-Resolve, kein Template-Picker, kein Seed).
6. **Report-Sources fehlen** für hr/zeiterfassung, vertraege, einkauf, fuhrpark, rapporte (je
   `*.source.ts` + Registry-Zeile). Bestehende 5 Quellen sind zu dünn (nur 5–7 Felder) → reiche
   Feldlandkarte (Teil 6) einarbeiten.
7. **`contract_value`** fehlt im vertraege-Modell — Measure lässt sich nicht abbilden.
8. **Externer Share-Link, Snapshots, dokumente-Ablage-Trigger nach Export** — komplett neu.
9. **Skeleton-Loading** in `BerichtLibrary` und `ReportDocumentEditor` — aktuell nur Text-Spinner
   statt Shimmer-Skeletons (projektweiter Standard).

---

## 5. Phasen-Plan R-0…R-6 (geschärft)

### ✅ R-0 — Fundament + Tab (ABGESCHLOSSEN, commit 424fa859)

- `ReportDocument`-Typen + MSW-CRUD + 3 Seed-Dokumente
- Tab „Berichte" als Bibliothek (`BerichtLibrary`) + `ReportDocumentEditor`-Shell (Outline-Ansicht,
  editierbarer Titel, Status-Badge, Seiten-Schätzung)
- `doc-utils.ts` (BLOCK_META, blockSummary, estimatePageCount), `ReportStatusBadge`

**Bekannte Lücken aus R-0 (für R-1 adressieren):**

- Skeleton-Loading fehlt in Bibliothek + Editor
- Editor hat nur Outline, kein Bearbeiten
- Lese-Modus noch nicht vorhanden

---

### R-1a — Block-Editor Kern (FE, ~5 Phasen-Batch-Start)

**Ziel:** Ein Bericht kann aktiv bearbeitet werden. Blöcke einfügen, inline bearbeiten,
  neu anordnen.

**Akzeptanzkriterien:**

- `+`-Button am Ende jeder Zeile öffnet ein Inline-Menü mit allen 11 Block-Typen (Icon + Bezeichnung, keine Emoji); Tastatur-navigierbar.
- Neuer Block wird als neue Zeile (ein-spaltig) direkt unter dem Trigger eingefügt.
- Jeder Block hat eine **minimale Inline-Bearbeitung**:
  - `cover`: Felder Titel / Untertitel / Autor inline editierbar (Input-Felder direkt im Block).
  - `heading`: Inline-Text-Input (kein RTE, plain text).
  - `text`: `RichTextEditor` (readOnly=false, compact=true, Toolbar sichtbar beim Fokus).
  - `bullet`: Zeile pro Item, Enter = neues Item, Backspace auf leerem Item = löschen.
  - `callout`: Varianten-Picker (info/success/warning/recommendation als Icon-Toggle) + TipTap
    für den Fließtext-Bereich.
  - `kpi`: Felder Label / Wert / Einheit als Input; changePercent als Zahlfeld.
  - `divider` / `pagebreak`: kein Inhalt, nur Drag-Handle + Löschen-Button.
  - `image`: URL-Input + alt + caption.
  - `chart` / `table`: Platzhalter-Ansicht mit Caption-Input + „Grafik konfigurieren"-Button →
    öffnet Chart-Block-Picker (R-1b).
- Block-Reorder via **dnd-kit** Drag-Handle (links am Block, nur sichtbar beim Hover). Zeilen werden
  als Ganzes verschoben, nicht einzelne Blöcke innerhalb einer Zeile.
- Löschen-Button (Papierkorb, `stopPropagation`) am Block rechts oben, erscheint beim Hover;
  Bestätigung via `ConfirmDialog` nur bei Text-Blöcken mit Inhalt.
- Auto-Save: PATCH `/berichte/documents/:id` nach 800 ms Inaktivität (debounce), Save-Indikator
  im Header (gespeichert / speichert…).
- **Skeleton-Loading** in Bibliothek und Editor (Shimmer-Cards, projektweiter Standard).
- i18n: alle neuen Schlüssel `berichte.editor.*`, `{var}`-Syntax (nicht `{{var}}`).

**FE-mockbar:** vollständig — CRUD-Endpoints in MSW vorhanden.
**Backend-Bedarf:** keiner.

---

### R-1b — Spalten-Layout + Chart-Block-Picker

**Ziel:** KPI-Reihen und „Text neben Chart"-Layouts sind anlegbar. Grafiken inline konfigurieren.

**Akzeptanzkriterien:**

- Im Block-Einfüge-Menü: Option „Zeile mit 2 Spalten" und „Zeile mit 3 Spalten" (je mit
  Vorschau-Icon, z.B. `|  |` bzw. `| | |`).
- Eine bestehende Zeile lässt sich per Kontext-Menü (3-Punkte am Zeilenrand) in eine
  2- oder 3-Spalten-Zeile umwandeln; Inhalte verbleiben in Spalte 1.
- Spalten-Breite via vorgefertigten Presets: 50/50 · 60/40 · 40/60 · 33/33/33 (Buttons in
  Zeilen-Kontext-Menü, kein Pixel-Drag).
- **Chart-Block-Picker** (Modal, zentriert, `shared/DetailModal` oder eigenes):
  - Tab 1 „Neue Grafik": öffnet den Builder-Konfigurations-Flow (SourcePicker → FieldPicker →
    VizSwitcher → FilterBuilder → Preview) inline im Modal; Speichern erzeugt neue
    `ReportDefinition` und trägt `definitionId` in den `ChartBlock` ein.
  - Tab 2 „Aus Bibliothek": Liste aller `ReportDefinition`-Einträge (kind=custom + system) mit
    Suche und Vorschau; Auswahl trägt `definitionId` ein.
  - Nach Auswahl/Speichern: Block zeigt `ChartRenderer`-Vorschau (wie im Builder, Daten aus MSW)
    + Caption-Input.
- `TableBlock` analog: selber Picker, aber `viz: 'table'` vorbelegt.
- `KpiBlock` als Einzel-Wert; `KPI-Reihe` = 3-Spalten-Zeile mit je einem KpiBlock → kein
  Spezial-Block nötig.

**FE-mockbar:** vollständig.
**Backend-Bedarf:** keiner für FE-Demo.

---

### R-2 — Lese-Modus + Lebenszyklus

**Ziel:** Berichte können gelesen und durch die Statuskette geführt werden.

**Akzeptanzkriterien:**

- **Edit/Lese-Toggle im Header** (zwei Tabs oder Toggle-Button: „Bearbeiten" / „Lesen").
  Lese-Modus blendet alle Editor-UI aus (kein Drag-Handle, kein +, kein Hover-Papierkorb).
- **Lese-Modus rendert Blöcke vollwertig:**
  - `cover`: vollständiges Deckblatt (Titel groß, Untertitel, Autor, Datum, optional Logo).
  - `heading`: typografisch differenziert (H1 > H2, Cosmi-Designsystem).
  - `text`: `RichTextEditor` mit `readOnly=true` (kein Toolbar, kein Bubble-Menu), voller HTML.
  - `chart` / `table`: `ChartRenderer` mit echten Daten aus MSW.
  - `kpi`: gestalteter KPI-Block (große Zahl, Einheit, Trend-Pfeil + Farbe).
  - `callout`: farbige Box je nach `variant` (info=blau, success=grün, warning=gelb,
    recommendation=violett/primary).
  - `bullet`: `<ol>` oder `<ul>` je nach `ordered`.
  - `divider` / `pagebreak`: visuell als Trennlinie (pagebreak = gestrichelt + Label „Seitenumbruch"
    nur im Editor, im Lese-Modus unsichtbar).
  - `image`: `<img>` + Caption.
- **Sticky Header** bleibt beim Scrollen sichtbar: Zurück-Button + Titel + Status-Badge +
  Edit/Lese-Toggle + PDF-Button. Kein Überlappen mit Inhalt.
- **Status-Übergangs-Buttons** im Header (kontextsensitiv):
  - draft → Fertig: Button „Als fertig markieren". Guard: mind. 1 Block vorhanden.
  - final → Freigeben: Button „Freigeben". Guard: Status muss `final` sein.
    Freigabe setzt `released_at` + PATCH status='released'. Intern schreibgeschützt danach.
  - released → Archivieren: Button „Archivieren" (im 3-Punkte-Menü des Headers, nicht prominent).
  - archived → kein Weiter; nur Duplizieren als neues draft.
  - draft/final: Bearbeiten bleibt möglich. released: Editor ist gesperrt (lesbar, nicht editierbar).
- **Snapshot-Logik:** bei Übergang → `released` ein Snapshot-Record (`snapshot_at`, `snapshot_rows`
  JSONB) in MSW speichern. Im Lese-Modus anzeigen: „Freigegeben am {date}".
- Aus der Bibliothek öffnet ein Klick direkt im Lese-Modus (nicht Edit-Modus). Edit nur über
  expliziten Toggle oder „Bearbeiten"-Aktion.
- Empty-State im Lese-Modus falls Dokument keine Blöcke hat: illustrierter Hinweis
  „Noch kein Inhalt — zum Bearbeiten wechseln."

**FE-mockbar:** Snapshot-Record in MSW simulierbar.
**Backend-Bedarf:** Snapshot-Persistenz (JSONB) → Luke; Status-Guards FE-seitig.

---

### R-2p — Reader-Premium-Politur (Darien-Feedback 2026-06-20)

**Kontext:** B1-6 (commit 2916acd9) hat den Lese-Reader **funktional** gebaut — Deckblatt,
Überschriften, Fließtext, KPI-Kacheln, echte Diagramme (ChartRenderer), Callouts, Spalten-Layout.
Darien: *„ist ok fürs Erste, aber da müssen wir nochmal deutlich rüber."* Ziel dieser Phase: vom
funktionalen Reader zum **echten Premium-Bericht-Dokument**, das wie ein gedruckter Geschäftsbericht
wirkt (und optisch dem späteren PDF aus R-3 entspricht).

**Akzeptanzkriterien (reine Layout-/Design-Arbeit, kein Backend):**
- **Echtes Papier-/Seiten-Gefühl:** A4-Proportionen, klare Seitenränder, dezenter Schatten;
  Seitenumbrüche optional sichtbar als Seitentrennung. Kopf-/Fußzeile gerendert (Logo + „Seite X
  von Y" aus `settings.showHeader/showFooter/showPageNumbers`).
- **Editorial-Typografie:** echte Hierarchie (Deckblatt-Titel groß, Serif-Option Playfair Display
  für Titel/H1; H1 ≠ H2 klar), Lesebreite (~65ch) für Fließtext, vertikaler Rhythmus + konsistente
  Section-Abstände.
- **Deckblatt-Design:** vollwertiges Cover (vertikal komponiert, Akzentlinie/-fläche aus
  `settings.accentColor`, Logo-Platz, Periode/Autor/Datum sauber gesetzt) — nicht nur zentrierter Text.
- **KPI-Kacheln & Diagramme:** konsistente Höhen + Rahmen, Diagramm-Titel/Legende, Akzentfarbe
  durchziehen; Tabellen-Block poliert.
- **Callouts:** ruhigere, professionellere Boxen (Icon je `variant`, dezente Farbe).
- **Drucknah:** Reader-Layout als Print-CSS-Vorbereitung für R-3 (gleiche Maße/Typo).

**FE-mockbar:** vollständig. **Backend-Bedarf:** keiner.
**Skills gezielt einsetzen:** `frontend-design`, `typeset`, `polish`, `arrange`, `impeccable`.
**Einordnung:** eigener Batch oder mit R-1b/R-2 kombinierbar; Design-lastig → Hauptterminal.

---

### R-3 — PDF-Export (1a)

**Ziel:** Ein professionell aussehender PDF-Export ist möglich.

**Unterphase R-3a — Print-CSS + window.print() (FE, sofort baubar):**

- Print-Stylesheet `modules/berichte/styles/report-print.css`:
  - `@page { size:A4; margin:20mm }` global
  - `@page:first { margin-top:0 }` (Deckblatt randlos)
  - `.block-atomic { page-break-inside:avoid }` für chart/table/kpi/kpi-row
  - `.block-pagebreak { page-break-after:always }` für explizite Seitenumbrüche
  - Kopf-/Fußzeile via `@page` (CSS-Paged-Media) oder als Fixed-Element mit
    `position:fixed; top:0` (Browser-Variante, funktioniert in Chrome-Print)
  - „Seite X von Y" via CSS-Counter (funktioniert in Chromium-Print).
  - Plus-Jakarta-Sans via `@font-face` einbetten (lokale Datei oder CDN-Font-Load vor Print).
- `window.print()`-Trigger-Button im Lese-Modus (nur im Print ausgelöst aus dem Lese-Modus-HTML).
- Akzeptanzkriterium: in Chromium-Druckvorschau entsteht ein visuell vollständiger Bericht
  (Deckblatt Seite 1, Charts/KPIs nicht abgeschnitten, Seitenzahl in Fußzeile sichtbar).

**Unterphase R-3b — Server-PDF via Playwright (Backend-Bedarf = Luke):**

- FE erstellt temporäre Render-URL `/berichte/documents/:id/print` (Token-geschützt, kein UI-Chrome).
- Luke schreibt `berichte-pdf` Service: GET URL → Playwright `page.pdf({ format:'A4', ... })` →
  Response als `application/pdf`-Blob.
- FE: `GET /api/v1/berichte/documents/:id/export/pdf` → Blob → `<a download>` → echter Download.
- Akzeptanzkriterium: heruntergeladene PDF öffnet sich in Acrobat/Preview mit korrekter
  Seitenanordnung, Schriften eingebettet, Charts als Vektoren (SVG → PDF native).

**FE-mockbar (R-3a vollständig):** window.print() braucht kein Backend.
**Backend-Bedarf:** nur R-3b — Playwright-Service (Luke-Bedarf, in `backend-gaps.md` tracken).

---

### R-4 — Scheduling am Bericht

**Ziel:** Berichte können automatisch erstellt und versendet werden.

**Akzeptanzkriterien:**

- „Zeitplan einrichten"-Button in der Editor-Header-Toolbar (nur sichtbar wenn Status = released
  oder als Hinweis wenn status < released: „Erst freigeben, dann planen").
- Öffnet ein **Scheduling-Modal** (zentriert, nicht Slide-over):
  - Rhythmus: täglich / wöchentlich (Wochentag wählbar) / monatlich (Tag wählbar) / quartalsweise.
  - Empfänger: interne Nutzer (aus Nutzer-Liste, Typeahead) + externe E-Mail-Adressen.
  - Format: PDF (Standard), XLSX, CSV — nur erlaubte Formate aus Tenant-Settings.
  - Nächste Ausführung (read-only Vorschau).
  - Toggle aktiv/inaktiv.
- CRUD koppelt an `ReportSchedule.definition_id`; MSW-CRUD vorhanden → `definition_id` = ID des
  `ReportDocument` (MSW-seitig gleiche Tabelle, Feld bereits vorhanden).
- Guard: Scheduling-Modal zeigt Warnung und deaktiviert „Speichern" wenn Status ≠ `released`.
- **Lauf-Historie**: Tab oder Abschnitt im Scheduling-Modal: letzte 5 Läufe mit Datum, Status
  (success/failed/skipped), Empfänger-Anzahl.
- Einzel-„Jetzt senden"-Button (manueller Trigger, `trigger: 'manual'`) im Modal.
- Modul-Einstellungen (Tenant-Bereich) ergänzen: „Geplante Berichte nur bei Status ≥ Freigegeben"
  als Info-Text (Regel ist bereits im System-Konzept verankert, keine neue Konfiguration nötig).

**FE-mockbar:** vollständig — ReportSchedule-CRUD bereits in MSW.
**Backend-Bedarf:** Cron-Executor + Mailer (Luke); FE-Demo ist vollständig ohne Backend.

---

### R-5 — Integration (Verteilung, Anhang, Share-Link)

**Ziel:** Berichte gelangen an Personen innerhalb und außerhalb von Cosmi.

**Unterphase R-5a — Bericht an Aufgabe anhängen:**

- Im Lese-Modus-Header: Aktions-Button „An Aufgabe anhängen" (oder im 3-Punkte-Menü).
- Öffnet ein Suchfeld (Typeahead über Aufgaben aus MSW `GET /work/tasks`).
- Nach Auswahl: `POST /api/v1/tasks/:id/files` mit Metadaten (name=Bericht-Titel,
  source_type='report', source_id=documentId, url=Lese-Link). Keine Datei-Kopie.
- In der Aufgaben-Detailansicht (work-Modul) erscheint der Bericht in der Dateiliste
  als „Bericht-Link" (Icon unterschiedlich von Datei-Icon).
- Analog: „An Kontakt/Deal anhängen" (CRM) — gleicher Flow, anderer Endpoint.

**Unterphase R-5b — PDF → dokumente ablegen:**

- Im Lese-Modus: Button „Als PDF in Dokumente ablegen".
- Trigger: FE generiert Print-PDF (R-3a window.print → Blob) oder ruft Server-PDF auf (R-3b).
- `POST /api/v1/documents/files/upload` mit PDF-Blob + `folder_id` (wählbar aus Ordner-Picker oder
  Standard-Ordner „Berichte"). Tag „Bericht" wird automatisch gesetzt.
- Nach Upload: Toast „Gespeichert in Dokumente" mit Link auf die Datei.

**Unterphase R-5c — Externer Token-Read-Link:**

- Geteiltes Konzept mit formulare (`public_tokens`-Mechanik, `formulare-distribution-VISION.md`).
- Im Freigeben-Dialog (Übergang → released): Option „Externer Link generieren" (optional, per
  Toggle).
- Erzeugt Token (`share_token` UUID) in MSW, speichert in `ReportDocument.share_token` (neues
  Feld). Ablaufdatum wählbar (30 / 90 Tage / unbegrenzt). Optionales Passwort-Feld.
- Share-Link: `https://app.zentria.tech/share/report/:token` — im Electron-Client reicht
  `cosmi://share/report/:token` für Demo.
- Externe Seite (Backend-Bedarf Luke) rendert Lese-Modus-HTML ohne Auth, kein Sidebar-Chrome.
- Im Modul: „Geteilte Links"-Übersicht im 3-Punkte-Menü des Berichts → Liste aktiver Tokens
  mit Ablauf / Aufrufe (counter) / Widerrufen-Button.
- FE-Demo: `share_token` in MSW speichern + simulierter Link-Kopier-Button. Echter
  unauthentifizierter Zugriff = Luke.

**FE-mockbar:** R-5a vollständig (work-Task-Files-MSW vorhanden). R-5b vollständig mit
window.print-Blob. R-5c teilweise (Token speichern/anzeigen → FE; echter externer Zugriff → Luke).
**Backend-Bedarf:** R-5c externer Zugriff (unauthentifizierter Endpunkt); R-3b für echten PDF-Blob
bei R-5b.

---

### R-6 — Datenquellen-Ausbau

**Ziel:** Alle Module als reiche Berichtsquellen verfügbar.

**Akzeptanzkriterien:**

- Bestehende 5 Quellen (finanzen, kontakte, work, helpdesk, kommunikation) vertiefen: min.
  12 Felder pro Quelle statt aktuell 5–7. Feldlandkarte in Teil 6 als Vorlage.
- 6 neue `*.source.ts`-Dateien anlegen + je eine Zeile in `registry.ts`:
  - `hr.source.ts` (Mitarbeiter / Abwesenheit / Urlaubskonto)
  - `zeiterfassung.source.ts` (Zeit-Eintrag / Analytic-Rollup / Team-Woche)
  - `vertraege.source.ts` (Vertrag + `contract_value` als Measure — Feld muss ergänzt werden)
  - `einkauf.source.ts` (Bestellung / Lieferantenbewertung / Rahmenvertrag)
  - `fuhrpark.source.ts` (Fahrzeug / Tankbuchung / Fahrtenbuch / Schaden)
  - `rapporte.source.ts` (Arbeitsrapport / Rapport-Zeile)
- Jede neue Source hat `sampleRows()` mit realistischen Demo-Werten.
- `contract_value` in `vertraege`-MSW-Daten ergänzen (Feld bisher fehlend).
- Alle neuen Sources erscheinen im Chart-Block-Picker (SourcePicker liest aus `REPORT_SOURCES`
  → automatisch ohne Extra-Arbeit).
- R-6 ist parallelisierbar: je Source in eigenem Sub-Terminal (nur `registry.ts` als Merge-Punkt).

**FE-mockbar:** vollständig — nur FE-Dateien.
**Backend-Bedarf:** keiner für FE-Demo.

---

## 6. Vorlagen-System (neu — war in R-0 nicht geplant)

**5 Standard-Vorlagen** aus Markt-Synthese (Metabase / Databox / Klipfolio):

| ID | Name | Haupt-Modul | Block-Struktur (Kurzform) |
|----|------|-------------|--------------------------|
| `tpl-monatsmanagement` | Monats-Management-Bericht | cross | Deckblatt · KPI-Reihe (3) · H1 „Umsatz" · Chart · H1 „Helpdesk" · Chart · Callout |
| `tpl-vertriebsquartal` | Vertriebsbericht Quartal | crm | Deckblatt · H1 · KPI-Reihe · Chart (Pipeline) · Tabelle (Top-Deals) · Callout |
| `tpl-helpdesk-woche` | Helpdesk-Wochenbericht | helpdesk | Deckblatt · KPI-Reihe · Chart (Offen/Gelöst) · Bullet (Eskalationen) |
| `tpl-finanzen-jahresabschluss` | Finanzübersicht Jahresabschluss | finanzen | Deckblatt · H1 · Chart (Umsatz 12M) · Tabelle (BWA) · H2 · Callout |
| `tpl-hr-team` | Team- & HR-Monatsreport | hr | Deckblatt · KPI-Reihe (Köpfe/Abwesenheit/Urlaubsquote) · Tabelle · Bullet |

**Implementierung:**

- Template-Seed in MSW: `DEMO_TEMPLATES: ReportTemplate[]` (Type: Subset von `ReportDocument`
  ohne `id`/`tenant_id`/`status`/`created_by`). Eigener Endpoint `GET /berichte/templates`.
- **Template-Picker** in `BerichtLibrary`: separater Abschnitt „Von Vorlage starten"
  (horizontal scrollbarer Strip mit Template-Cards: Name + Beschreibung + Block-Anzahl-Hint).
  Klick → `POST /berichte/documents` mit `template_id` vorbelegt, `rows` aus Template kopiert.
- Template-Karte zeigt kleine Strukturvorschau (Block-Icons in Zeilen, nicht interaktiv).
- Keine eigene Verwaltungs-UI für Vorlagen in MVP — nur System-Vorlagen, keine custom-Vorlagen.
- `template_id` ist bereits in `ReportDocument` vorhanden.
- Einbinden in **R-1a** (Template-Picker kann parallel zum Editor gebaut werden, da unabhängig von
  Block-Editor-Mechanik).

---

## 7. Modulspezifische Berichtsfelder (Feldlandkarte — für R-6 / Schicht 2)

> Reiche Liste aus Cosmi-MSW/Types. Pro Modul: Entität → Schlüssel-Dimensionen / Measures.

- **finanzen:** Rechnung (issue_date/due_date·status[draft/sent/paid/overdue/cancelled]·customer·
  currency·tax_rate·payment_terms | total_net/total_gross/tax) · Angebot
  (status[accepted/rejected]·…) · Gutschrift (is_storno·reason) · **Mahnung**
  (level 1/2/3·status·fee·interest) · **Ausgabe**
  (category·supplier·project·account·status | amount) · Transaktion (matchStatus) ·
  Wiederkehrend (interval·status·next_run·generated_count) · Unbilled-Time
  (duration_hours·hourly_rate·amount·billed).
- **kontakte/crm:** Kontakt (created_at·title·company·tags·country) · Firma
  (industry·country·contactCount) · **Deal**
  (stage[lead/qualified/proposal/negotiation/won/lost]·owner·close_date | value·probability) ·
  Aktivität (type[call/email/meeting/note]·count) · Segment/Tags.
- **work:** Projekt (status·priority·owner·dates·is_template | progress·task_count·completed·
  member_count) · **Aufgabe** (status_name·priority·assignee·due_date·project·tags |
  estimated_hours·subtasks·comments) · Zeit-Eintrag (user·is_manual·billed | duration_seconds) ·
  Task-Datei · Abhängigkeit.
- **helpdesk:** Ticket (status[open/pending/solved/closed]·priority·assignee·queue·
  created/resolved | response_mins·resolution_mins·count) · SLA
  (status[on_track/at_risk/breached]) · Queue · Stats
  (open/avg_response/resolved_this_week/CSAT).
- **kommunikation/inbox:** Nachricht (received_at·channel[email/chat/notification]·
  is_read/starred/archived·assigned_to·tags·crm_contact | response_mins·count).
- **team/hr:** Mitarbeiter (department·position·contract_type[full/part/mini]·start_date |
  work_days/leave_days) · Abwesenheit (type·status·days) · Urlaubskonto
  (total/used/remaining/pending) · Personaldoc (category·fileSize).
- **zeiterfassung:** Zeit-Eintrag
  (date·project·customer·activity·status·billable | totalMinutes/netWork/break) ·
  Analytic-Rollup (billable/nonbillable/overtime/workedDays) · Team-Woche
  (department·weekStatus | weekMinutes/target/overtime).
- **vertraege:** Vertrag (contract_type[rental/service/employment/nda]·
  status[draft/active/expired/terminated]·starts/ends | Laufzeit·count) ⚠ `contract_value`
  fehlt im Modell · Erinnerung (reminder_type·status).
- **dokumente:** Datei (created_at·mime_type·space[personal/team/project]·creator·tags·favorite |
  file_size) · Version.
- **einkauf:** Bestellung (status·order_date·supplier·currency | total_amount) ·
  Lieferantenbewertung (category·rating) · Rahmenvertrag (status·total/used_value).
- **fuhrpark:** Fahrzeug (make/model/year·fuel_type·status·tuev_due | mileage_km) · Tankbuchung
  (| liters·cost) · Fahrtenbuch (purpose·is_private | km) · Schaden (severity·status·cost).
- **rapporte:** Arbeitsrapport (status[draft/submitted/approved/rejected]·report_date·
  author·reviewer·count) · Rapport-Zeile (unit | quantity).
- **automatisierung/notifications:** Trigger-Typ·Aktionstyp·status·execution_count (als
  „Automatisierungs-Audit").

---

## 8. Offene Entscheidungen (aktualisiert)

Entschiedene Punkte aus der Ursprungsliste:

- **Editor-Paradigma:** Block-basiert (bestätigt durch R-0-Implementierung).
- **Bericht ↔ dokumente:** eigenständig im berichte-Modul mit Brücke (Anhang + Upload).
- **Status-Modell:** 4 Status ohne Approval-Gate (bestätigt).

**Neu / noch offen (für Darien):**

1. **Template-Timing:** Vorlagen-Picker in R-1a parallel aufbauen, oder erst nach Editor-Kern in
   eigener Mini-Phase? (Empfehlung: parallel, da unabhängige Komponente.)
2. **Chart-Block-Picker Modal-Tiefe:** Voller Builder-Flow (SourcePicker → FieldPicker → Viz →
   Filter → Preview) im Modal — oder nur Bibliothek-Auswahl, neuer Builder weiterhin im eigenen
   Tab? (Empfehlung: vollständiger Flow im Modal, weil Tab-Wechsel mitten im Bericht-Editieren den
   Kontext bricht.)
3. **Lese-Modus als Default beim Öffnen:** ja (Empfehlung) — oder direkt in Bearbeiten?
   Konsequenz: freigegebene Berichte sind immer zuerst read-only, was den Guard-Workflow erzwingt.
4. **`share_token`-Feld in `ReportDocument`:** FE-seitig als optionales Feld ergänzen (MSW +
   Typ), oder warten auf Luke-Backend? (Empfehlung: FE-seitig jetzt, Token = simulated UUID in
   MSW; echter Link = R-5c Backend-Phase.)
5. **R-5a Aufgaben-Anhang-Darstellung im work-Modul:** soll der angehängte Bericht als
   normaler Datei-Eintrag oder als Sondertyp (Icon „Bericht", klickbar direkt in Lese-Modus)
   erscheinen? Abhängig von work-Modul-Scope (nicht im berichte-Scope zu entscheiden — Absprache
   mit Nico nötig).
6. **Scope R-6 vor R-5 oder nach R-5?** Quellen-Ausbau hat Nutzen ab dem Moment, wo Grafiken in
   Berichte eingebettet werden (R-1b). Empfehlung: R-6 parallel zu R-2/R-3 in Sub-Terminal starten,
   weil Sources vollständig disjunkt von Editor-Code sind.

---

## 9. 5-Phasen-Batch-Empfehlung

Jeder Batch = ~5 Phasen, die zusammen review-reif sind (vollständig testbar via
Playwright-Screenshot-QA ohne Backend-Dependency). Übergabe-Kriterium = Nico sieht im Screenshot
keine Raw-Keys, keine leeren Zustände ohne EmptyState, keine hängenden UI-Elemente.

**Batch 1 (= erster Batch, jetzt starten):**

| Nr. | Inhalt | Scope |
|-----|--------|-------|
| 1 | Skeleton-Loading in `BerichtLibrary` + `ReportDocumentEditor` (Shimmer) | R-0-Nacharbeit |
| 2 | Template-Picker in Bibliothek (5 Vorlagen, Template-Seed in MSW) | Vorlagen-System |
| 3 | Block-Editor Kern: `+`-Menü + cover/heading/text/bullet/divider/pagebreak inline bearbeiten | R-1a Teil 1 |
| 4 | Block-Reorder (dnd-kit Drag-Handle) + Löschen mit ConfirmDialog | R-1a Teil 2 |
| 5 | Auto-Save (debounce 800ms) + Save-Indikator im Header | R-1a Teil 3 |

**Batch 2:**

| Nr. | Inhalt | Scope |
|-----|--------|-------|
| 6 | Spalten-Layout (2/3-Spalten-Zeilen anlegen + Preset-Breiten) | R-1b Teil 1 |
| 7 | Chart-Block-Picker Modal (Bibliothek-Tab: bestehende Definitionen auswählen) | R-1b Teil 2 |
| 8 | Chart-Block-Picker Modal (Neu-Tab: Builder-Flow inline) | R-1b Teil 3 |
| 9 | KPI-Block vollständig + KPI-Reihe als 3-Spalten-Zeile | R-1b Teil 4 |
| 10 | Callout + Image-Blöcke + Block-Typ-Picker vollständiges i18n-Review | R-1a Abschluss |

**Batch 3:**

| Nr. | Inhalt | Scope |
|-----|--------|-------|
| 11 | Lese-Modus Rendering (alle Block-Typen vollwertig) | R-2 Teil 1 |
| 12 | Edit/Lese-Toggle + Sticky Header im Lese-Modus | R-2 Teil 2 |
| 13 | Status-Übergangs-Buttons + Guards (draft→final→released→archived) | R-2 Teil 3 |
| 14 | Snapshot-Anzeige (released_at + Snapshot-Info in Lese-Modus-Header) | R-2 Teil 4 |
| 15 | R-6 Start: hr.source.ts + zeiterfassung.source.ts (Sub-Terminal parallel) | R-6 Teil 1 |

**Batch 4:**

| Nr. | Inhalt | Scope |
|-----|--------|-------|
| 16 | Print-CSS + window.print()-Pfad (R-3a vollständig) | R-3a |
| 17 | Scheduling-Modal (Rhythmus / Empfänger / Format / Toggle) | R-4 Teil 1 |
| 18 | Lauf-Historie im Scheduling-Modal + Jetzt-senden-Button | R-4 Teil 2 |
| 19 | R-6: vertraege.source.ts + einkauf.source.ts + fuhrpark.source.ts | R-6 Teil 2 |
| 20 | R-6: rapporte.source.ts + bestehende 5 Sources vertiefen auf ≥12 Felder | R-6 Teil 3 |

**Batch 5:**

| Nr. | Inhalt | Scope |
|-----|--------|-------|
| 21 | Bericht an Aufgabe anhängen (R-5a) | R-5a |
| 22 | PDF → dokumente ablegen (R-5b, window.print-Blob) | R-5b |
| 23 | Share-Token FE-Simulation (R-5c MSW + UI, echter Link = Luke) | R-5c |
| 24 | Modul-Einstellungen: Abschnitt „Dokument-Vorlagen" (read-only Liste der 5 Vorlagen) | Settings |
| 25 | Vollständiges i18n-Audit (alle Schlüssel `berichte.docs.*`, `berichte.editor.*`, `{var}`) | QA |

---

## 10. Backend-Bedarf für Luke (kritisch, priorisiert)

| Prio | Bedarf | Phase |
|------|--------|-------|
| P0 | **PDF-Engine (Playwright Headless Chromium, server-seitig)** | R-3b |
| P0 | ReportDocument-Persistenz (blocks JSONB, status, `released_at`, `snapshot_rows`) | R-2 |
| P1 | Schedule-Run-Executor + Mailer | R-4 |
| P1 | `share_token`-Feld + unauthentifizierter Token-Endpunkt | R-5c |
| P2 | `dokumente`-`source_type:'report'`-Feld | R-5b |
| P2 | `contract_value`-Feld in vertraege-Modell | R-6 |

---

## Verweise

- Bisheriger Grafik-Builder (Schicht 3): `.planning/berichte-builder-plan.md` (E-0…E-5, fertig).
- Datenquellen-Registry-Muster: `desktop/src/renderer/src/modules/berichte/report-sources/`.
- Wiederverwenden: `components/shared/RichTextEditor`, `shared/DetailModal`, `shared/SortMenu`,
  `ChartRenderer` (E-1), `shared/ConfirmDialog`, `shared/EmptyState`, `shared/ItemActions`.
- dnd-kit: bereits im Projekt (Formulare-Builder), kein npm-add nötig.
- Formulare Token-Konzept (Parallel-Mechanik): `.planning/formulare-distribution-VISION.md`.
