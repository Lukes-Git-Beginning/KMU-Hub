# RESUME — Batch 2 (berichte R-1b + formulare Batch B)

> **Handoff für neues Terminal.** Stand `main = 3a961845` (alles gepusht, working tree clean).
> Sub-Terminal-Modus: Hauptterminal (KMU Hub, Port 5173) baut berichte, Sub-Terminal
> (KMU-Hub-review, Port 5174) baut formulare. Je 5-Phasen-Batch, dann Review durch Darien.
> Workflow-Memory: `feedback_sub_terminal_5phase`.

## Was fertig ist
- **berichte:** R-0 (Fundament) + R-1a (Block-Editor-Kern: Skeleton, Vorlagen-Picker, +-Menü,
  Inline-Edit cover/heading/text/bullet/divider/pagebreak, dnd-Reorder, Löschen, Auto-Save) +
  **B1-6** (Lese-Reader: Deckblatt/Text/KPI-Kacheln/echte Diagramme/Callout, Spalten-Layout).
  KPI/Chart/Table sind im EDITOR noch Platzhalter (im Reader gerendert).
- **formulare:** Batch A = FD-0 (Lifecycle + „geschlossen"), FD-1 (Share-Link Token/Clipboard),
  FD-2 (Verteil-Übersicht), FT-1 (Submissions-Tiefe), FT-4 (View+Filter). Alle gemergt.
- **Offenes Darien-Feedback:** Reader „ok fürs Erste, muss deutlich rüber" → Phase **R-2p**
  (Reader-Premium-Politur) im berichte-VISION-Doc eingetragen, als eigener Design-Batch SPÄTER.

## Hauptterminal · berichte = R-1b (5 Phasen)
Spezifikation: `.planning/berichte-report-authoring-VISION.md` Abschnitt R-1b + Batch 2.
Editor-Datei: `desktop/src/renderer/src/modules/berichte/components/documents/BlockEditor.tsx`.
Chart-Picker wiederverwendet den bestehenden Builder (`components/builder/SourcePicker`,
`FieldPicker`, `VizSwitcher`, `useReportPreview`). Reader: `DocumentReader.tsx`.

| # | Phase |
|---|---|
| B2-1 | Spalten-Zeilen anlegen (Zeile in 2/3 Spalten teilen, Preset-Breiten 50/50 · 60/40) |
| B2-2 | Chart-Block-Picker — Tab „Aus Bibliothek" (gespeicherte Definitionen wählen) |
| B2-3 | Chart-Block-Picker — Tab „Neu" (Builder-Flow inline: Quelle/Felder/Viz) |
| B2-4 | KPI-Block voll editierbar (Label/Wert/Einheit/Trend/Quelle) + „KPI-Reihe einfügen" (3 Spalten) |
| B2-5 | Callout-Block (Variant+Titel+RichText) + Image-Block + Tabellen-Block-Picker |

Pro Phase: bauen → i18n ×4 (`{var}`, ICU-Plural) → gescopter Typecheck (neuer
`tsconfig.b2check.json` nach Muster `tsconfig.r0check.json`) → Playwright-Screenshot-QA gegen
5173 (Muster `scripts/qa-berichte-b1-3.mjs`) → Bilder ansehen → ein Commit (explizite Pfade) →
gebündelt pushen (pull --rebase über Sub-Terminal-Pushes).

## Sub-Terminal · formulare = Batch B (5 Phasen) — copy-paste ins KMU-Hub-review-Terminal

```
Du bist das Sub-Terminal im Sub-Terminal-5-Phasen-Modus, zweiter Batch. Arbeitsverzeichnis = dieser KMU-Hub-review-Klon (Dev-Server Port 5174). Das Hauptterminal baut parallel berichte R-1b — du fasst NUR formulare-Dateien + i18n an. Sprache: Deutsch (Umlaute, Eszett).

SCHRITT 0 — aktuellen Stand holen:  git pull --rebase origin main

PFLICHTLEKTÜRE:
1. .planning/formulare-distribution-VISION.md — Abschnitte 9c (Builder-Tiefe), 9d (Auswertung), 10 (FT-2/FT-3/FT-5 mit Akzeptanzkriterien). DAS ist deine Spezifikation. Beachte: FD-3 ist mit FT-3 ZUSAMMENGELEGT (konsolidiert).
2. Dein Batch-A-Code (FD-0…FD-2, FT-1, FT-4) ist bereits gemergt — bau darauf auf. Ist-Code: desktop/src/renderer/src/modules/formulare/FormularePage.tsx + .../settings/FormulareSettingsPanel.tsx + mocks/handlers/formulare.ts.

DEIN BATCH B — 5 Phasen, je ein Commit:
- FT-2a Feld-Validierungsregeln: im Feld-Konfig-Dialog optionale Regeln — min/max Länge (text/textarea), min/max Wert (number), Pattern-Typ-Dropdown für text („Frei"/„PLZ 5 Ziffern"/„Telefon DACH"/„IBAN", speichert intern Regex). KEIN freies Regex-Feld. Validierung in der öffentlichen/Vorschau-Ausfüllansicht anwenden + Fehlermeldung.
- FT-2b Rating/NPS-Feldtyp + Seitentitel pro Seite (mehrseitig) + DACH-Vorlagen-Pack (zusätzliche Vorlagen) + Danke-Seite-Konfiguration (Text/optional Redirect-URL).
- FT-3a Auswertungs-Tab im Formular-Detail-Modal (3. Tab neben Details + Eingänge): Kennzahlen-Zeile (aus GET /schemas/:id/stats) + pro Feld eine Auswertungskarte — Balkendiagramm für select/radio/checkbox via ChartRenderer (aus modules/berichte/components/charts), Freitext-Vorschau-Liste für text/textarea, Datumsverteilung für date.
- FT-3b Drop-off pro Seite (% Abbruch je Seite bei mehrseitigen Formularen) + Conversion-Rate (Aufrufe→Submissions, MSW-stateful aus den Share-Link-Aufrufen von FD-2).
- FT-5 Settings-Vervollständigung (Standard-Danke-Seiten-Text, Benachrichtigungs-E-Mail, View-Präferenz Grid/Liste) + Vorlagen-Management (eigene Vorlagen erstellen/pinnen).

BUILD-+-VERIFY-STANDARD PRO PHASE (verbindlich):
bauen → i18n ×4 (i18n/messages/{de,en,fr,it}.json; {var} NICHT {{var}}; Plural als ICU {count, plural, one {…} other {…}}) → gescopter Typecheck (desktop/tsconfig.formulareBatchB.json nach Muster von tsconfig.formularecheck.json; desktop/node_modules/.bin/tsc -p tsconfig.formulareBatchB.json --noEmit, foreground) → Playwright-Screenshot-QA gegen http://localhost:5174 (Muster: desktop/scripts/qa-formulare-ft1.mjs aus Batch A; Hash-Route /#/formulare) → die PNGs WIRKLICH ansehen (Raw-Keys/Emojis/Theme/Layout/leere Zustände, mehrere Datensätze + Breiten) → iterieren bis grün → ein Commit.

PROJEKTWEITE STANDARDS (sonst Review-Rückläufer):
Detail = shared/DetailModal, ganze Zeile klickbar (innere Buttons stopPropagation). Sortierung Feld+Richtung via shared/SortMenu. Sticky Back/Close. Leere Zustände = EmptyState mit Aktion. Keine Toast-Stubs/toten Endpoints — echte MSW-Funktion. KEINE Emojis. Theme-Tokens (bg-info-light, text-success …). Skeleton statt Spinner. CURRENT_USER aus shared-ids.ts. Neue Dateien unter mocks/data/ brauchen git add -f.

GIT:
Conventional Commits, English imperative, KEINE AI-Attribution. Commit pro Phase mit EXPLIZITEN Datei-Pfaden (git add desktop/.../formulare/... i18n/messages/*.json tsconfig.formulareBatchB.json) — NIE git add -A/.  Nach jedem Commit: git push origin main; bei Ablehnung git pull --rebase origin main (i18n-Konflikte: beide Key-Bereiche behalten), dann erneut push. Dev-Server (5174) nicht killen.

ABSCHLUSS: kurze Bilanz — welche Phasen committed (Hashes), QA-Ergebnis je Phase, was offen blieb.
```
