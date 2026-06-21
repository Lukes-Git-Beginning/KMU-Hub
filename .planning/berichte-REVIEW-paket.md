# berichte — Review-Paket (Darien, phasenweise)

> **Zur Seite gelegt für Dariens eigenen Review** (NICHT direkt an Nico — wir passen erst noch an).
> Status: **Report-Authoring R-0…R-6 = komplett & verifiziert.** Du gehst es Phase für Phase durch,
> trägst Anpassungen in die „Anpassungen"-Spalte ein; danach arbeite ich die Findings ab → erst dann an Nico.
>
> **Reviewen:** Dev-Server `cd desktop && npm run dev` (Port 5173) → App → Modul **Berichte** (`/#/berichte`).
> Vier Tabs: **Dashboard · Berichte · Geplant · DATEV**. Das Authoring lebt im Tab **Berichte**
> (Bericht öffnen / „Neuer Bericht").

## Zwei Stränge (Kontext)
1. **Report-Authoring R-0…R-6** — Dokument-Editor (Blöcke), Lese-Modus, PDF, Scheduling, Integration, Datenquellen. ← **dieses Paket**
2. **Erstellen-Builder E-1…E-5** (eigenständiger No-Code-Tab) — im MASTER-TRACKER separat geführt. Der Builder-Flow (SourcePicker → FieldPicker → Viz → Filter → Vorschau) ist faktisch über den **Chart-Block-Picker** („Neue Grafik") realisiert; ob zusätzlich ein eigenständiger Builder-Tab gewünscht ist, ist eine offene Produktfrage → siehe „Offene Punkte".

---

## Phasen-Review-Tabelle

| Phase | Was gebaut wurde | Klick-Pfad zum Prüfen | Wichtigste Prüfpunkte | Anpassungen (Darien) |
|---|---|---|---|---|
| **R-0** Fundament + Tab | Bibliothek (`BerichtLibrary`) + Editor-Shell, 3 Seed-Berichte, Status-Badge, Seiten-Schätzung | Tab **Berichte** → Liste der Berichte | Grid/Listen-Umschaltung, Status-Filter, Karte ganze Zeile klickbar, Skeleton beim Laden | |
| **R-1a** Block-Editor-Kern | 11 Block-Typen, Inline-Edit, dnd-Reorder, Auto-Save, Skeleton | Bericht öffnen → **Bearbeiten** → „Block einfügen" | Alle Block-Typen einfügbar; Cover/Heading/Text/Bullet/KPI/Callout/Image inline editierbar; Drag-Handle reordert; Auto-Save-Indikator; Löschen-Confirm | |
| **R-1b** Spalten + Chart-Picker | 2/3-Spalten-Zeilen, Chart-Block-Picker (Modal) mit „Neue Grafik" + „Aus Bibliothek" | „Diagramm"-Block einfügen → „Grafik konfigurieren" | Spalten-Presets (50/50, 60/40, 33/33/33); Picker: Quelle→Feld→Viz→Vorschau; „Aus Bibliothek" listet Definitionen; Tabelle analog | |
| **R-2** Lese-Modus + Lebenszyklus | Edit/Lese-Toggle, Status-Kette draft→final→released→archived, Snapshot, sticky Header | Header-Toggle **Lesen/Bearbeiten** + „Als fertig markieren" / „Freigeben" | Lese-Modus blendet Editor-UI aus; alle Blöcke vollwertig gerendert; sticky Header überlappt nicht; Status-Guards (mind. 1 Block); released = schreibgeschützt; Öffnen startet im Lese-Modus | |
| **R-2p** Reader-Premium-Politur | A4-Gefühl, Editorial-Typo (Playfair/report-serif), Deckblatt-Design, 65ch | Lese-Modus visuell prüfen | Papier-/Seiten-Gefühl, Ränder, Kopf-/Fußzeile; Titel-Hierarchie H1≠H2; Lesebreite; Deckblatt komponiert; Callouts ruhig; KPI/Charts konsistent | |
| **R-3** PDF-Export | Print-CSS + `window.print()` (R-3a). Server-PDF (R-3b) = 🔒 Luke | Lese-Modus → **Drucken/PDF** → Chromium-Druckvorschau | Deckblatt Seite 1, Charts/KPIs nicht abgeschnitten, Seitenzahl in Fußzeile, Schrift eingebettet | |
| **R-4** Scheduling am Bericht | „Zeitplan einrichten" (nach Freigabe), Rhythmus/Empfänger/Format, Lauf-Historie, „Jetzt senden" | Freigegebener Bericht → Header **Zeitplan** | Guard: nur ≥ released planbar; Modal zentriert; Rhythmus/Empfänger/Format; nächste Ausführung; Lauf-Historie (5); manueller Trigger | |
| **R-5** Integration | An Aufgabe/Kontakt anhängen, PDF→Dokumente, externer Share-Link + Verwaltung | Lese-Modus → **Teilen-Menü** | „An Aufgabe anhängen" (Typeahead, taucht in work-Datei-Liste auf); „An Kontakt"; „PDF in Dokumente" (Ordner-Picker, Tag „Bericht"); externer Link (Ablauf/Passwort) + „Geteilte Links"-Übersicht (Aufrufe/Widerrufen) | |
| **R-6** Datenquellen | **11 Quellen** im Chart-Picker (6 neu + 5 vertieft auf ≥12 Felder) | „Diagramm"-Block → „Grafik konfigurieren" → **Neue Grafik** → Datenquelle | Alle 11 sichtbar; je Quelle realistische Felder + gerenderte Live-Vorschau; deutsche Labels; Currency/Percent-Format | |

**Die 11 Datenquellen (R-6):** Rechnungen · Kontakte & Deals · Projekte & Aufgaben · Tickets · Posteingang · Mitarbeiter · Zeiterfassung · Verträge · Bestellungen · Fuhrpark · Arbeitsrapporte.

---

## QA-Screenshots (zum Querlesen, ohne App)
- `desktop/.qa-screenshots/berichte-r6-1` … `-r6-5` — Datenquellen (R-6), je Quelle eine Live-Vorschau.
- `desktop/.qa-screenshots/berichte-b1…b6`, `…/berichte-builder*` — Editor/Lese-Modus/Scheduling/Integration (R-1…R-5).
- QA-Skripte: `desktop/scripts/qa-berichte-*.mjs` (Re-Run: Dev-Server 5173, dann `node scripts/qa-berichte-<x>.mjs`).

## Offene Punkte / bewusst gemockt (🔒 = Luke-Backend)
- **R-3b** echter Server-PDF-Blob (Playwright-Service) 🔒 — aktuell `window.print()` als FE-Surrogat.
- **R-5c** echter unauthentifizierter externer Share-Zugriff 🔒 — Token wird FE-seitig erzeugt/verwaltet, Link ist Demo.
- **R-4** Cron-Executor + Mailer 🔒 — FE-Demo (stateful) vollständig.
- **Produktfrage:** eigenständiger „Erstellen-Builder"-Tab (E-1…E-5) zusätzlich zum Chart-Block-Picker — oder reicht der Picker? (Darien-Entscheidung)
- **R-6 i18n:** Field-Labels laufen über `defaultValue` (wie alle bestehenden Quellen); voller Field-Key-Sweep wäre modulübergreifend, separat.

## Findings-Sammelstelle (während des Reviews füllen)
> Pro Finding: `[Phase] Kurzbeschreibung — gewünschtes Verhalten`. Ich arbeite die Liste danach als FIX-Phasen ab.

- [ ] …
- [ ] …
