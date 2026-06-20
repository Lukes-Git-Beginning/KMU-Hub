# Sub-Terminal Kickoff — Modul „formulare" review-reif machen

> **Self-contained Auftrag für das zweite Terminal (Klon `KMU-Hub-review`, Port 5174).** Du arbeitest parallel zum Haupt-Terminal, das gerade den berichte-Report-Builder baut. Eure Lanes sind disjunkt — einzige Berührungspunkte sind die i18n-Dateien und `module-settings-registry.tsx` (siehe „Konflikt-Vermeidung").

## Rolle & Ziel
Du bist FE/UX-Entwickler an **Cosmi** (CRM für DACH-KMU, Software="Cosmi", Firma="Zentria"). Stack: Electron + React 19 + TS, Demo-Mode via MSW. Sprache: Kommunikation Deutsch, Code Englisch. **Keine ASCII-Umlaute** in User-facing-Text (immer „für", nie „fuer"), **keine Emojis in der UI**.

**Ziel:** Das Modul **formulare** komplett zu Ende bringen, sodass es „review-reif" ist (alle FE-Phasen + Demo-Tiefe; Backend darf gemockt sein). Modul soll im Demo lebendig sein, echte Detail-Ansichten haben, wirksame Exports, Moduleinstellungen, sauberes i18n.

## Setup
```bash
cd "C:/Users/darie/Documents/KMU-Hub-review/desktop"   # der Review-Klon
git fetch origin && git checkout main && git pull --ff-only
git checkout -b parallel/formulare                       # eigener Branch
npm run dev                                               # electron-vite, Demo-Mode (nimm Port 5174 falls 5173 belegt)
```
Dev-Server-Regel: nur 1 pro QA-Runde. Vor Neustart killen (PowerShell `Get-CimInstance Win32_Process` + `Stop-Process` — `pkill -f vite` greift unter Windows NICHT).

## Schritt 0 — Ist-Analyse (PFLICHT, zuerst)
Bekannte Eckpunkte (vom Haupt-Terminal vorab gescannt, aber selbst verifizieren):
- `modules/formulare/FormularePage.tsx` — **2457 Zeilen, schon umfangreich** (Tabs, Formular-Liste, Builder-Ansatz, Submissions, Templates). Nutzt `stores/formulare` (DraftSchema, FormField, FormFieldType), `api/hooks/useFormulare` (useFormSchemas/Create/Update/Delete/Duplicate/ExportSubmissions/UpdateSubmissionStatus), `api/formulare-types` (FormSchema, FormSubmission, FormSubmissionStatus).
- **KEINE MSW-Handler-Datei** unter `mocks/handlers/` für formulare gefunden → **wahrscheinlich ist der Demo-Mode (teilweise) tot** (Hooks rufen Endpoints, die kein Handler bedient → leere Listen / failedApiRequests). **Das ist der vermutete Haupt-Befund — verifizieren:** Modul im Demo öffnen, Browser-Network + Console auf 404/leere Antworten prüfen.
- **154 formulare-i18n-Keys (DE)** existieren bereits → i18n ist NICHT bei null (anders als wiki). Prüfe nur EN/FR/IT-Parität + ob Raw-Keys/`{{var}}`-Doppelklammern auftauchen.
- Route registriert (`App.tsx`: `/formulare`).

Erstelle nach der Analyse eine **Demo-Tiefe-Audit-Notiz** in `.planning/reviews/formulare.md` (Befunde, was tot/flach ist) und leite daraus deine Phasen ab. Halte dich grob an den Vorschlag unten, justiere nach Befund.

## Phasen-Vorschlag (F-1…F-5, justierbar)
- **F-1 — MSW-Vertrag + Demo lebendig (Kern):** `mocks/handlers/formulare.ts` neu bauen, in der Handler-Registry registrieren. Alle von `useFormulare` genutzten Endpoints bedienen (list/get/create/update/delete/duplicate Schemas, list/get Submissions, update-status, export). Stateful + geseedet: 5–8 Demo-Formulare (verschiedene Feldtypen), je Formular ein paar Submissions mit Status. Ziel: Liste/Detail/Submissions zeigen echte Inhalte, 0 failedApiRequests/pageErrors.
- **F-2 — Zeilen → DetailModal (Tiefe):** Klick auf Formular-Zeile UND Submission-Zeile öffnet `shared/DetailModal` (zentriertes Fenster, GANZE Zeile als `div role=button`, innere Buttons `stopPropagation`). Im Fenster alle Infos + Aktionen (Vorschau, Bearbeiten, Duplizieren, Teilen-Link, Status ändern, Löschen mit Confirm). Kein Slide-over.
- **F-3 — Formular-Builder schärfen (DnD):** Der Builder soll echte Drag-and-drop-Feld-Anordnung haben (dnd-kit, wie work-Kanban: `data-*`-Guard + `onPointerDown`-Stop gegen Klick-Konflikte). Feldtypen-Palette (Text/Textarea/Checkbox/Radio/Select/Datum/Zahl/Datei/E-Mail/Link), Feld-Eigenschaften (Label, Pflichtfeld, Platzhalter, Optionen), Live-Vorschau. Reihenfolge persistent im Store.
- **F-4 — DSGVO-Einwilligungsfeld + Submission-Tiefe:** Spezial-Feldtyp „Einwilligung/Consent" (Pflicht-Checkbox mit verlinktem Datenschutz-Text, Zweckbindung) — DACH-Pflicht für öffentliche Formulare. Submission-Detail zeigt erfasste Consent-Zustimmung + Zeitstempel. Export (CSV/XLSX) liefert echten Blob-Download (kein Toast-Stub).
- **F-5 — Moduleinstellungen + i18n + QA:** Eintrag im Modul-Einstellungs-Fenster via `ModuleSettingsShell` (in `modules/settings/module-settings-registry.tsx` registrieren, Gruppe `module`, navMatch `['/formulare']`). **Persönlich** (z.B. Standard-Ansicht Liste/Kachel, Standard-Status-Filter) + **Für alle** (z.B. Standard-Datenschutz-Text, Submission-Aufbewahrung, Benachrichtigung bei Eingang, Standard-Export-Format). i18n-Keys ×4 (DE/EN/FR/IT) für alle neuen Strings. Playwright-Screenshot-QA über mehrere Zustände/Breiten, Screenshots wirklich ansehen.

## Build-+-Verify-Standard (pro Phase, verbindlich)
bauen → i18n ×4 (**Interpolation `{var}`, NIE `{{var}}`** — ICU-Plugin; Plural als ICU `{count, plural, …}`) → Demo-Handler falls nötig → **gescopter Typecheck** (`tsconfig.fNcheck.json` über NUR geänderte Dateien, `node_modules/.bin/tsc --noEmit -p ...`, NIE Full-tsc als Gate — das crasht/dauert ~30 Min) → **Playwright-Screenshot-QA gegen :5174 + Screenshots ANSEHEN** (Raw-Keys/`{{}}`/Emojis/Layout/leere Zustände) → iterieren bis grün → ein Commit + Push pro Phase. „Kompiliert ja" reicht NICHT — die Bilder müssen angeschaut werden.

QA-Skripte als `desktop/scripts/qa-formulare-*.mjs` (Playwright gegen die laufende Dev-URL). `reviews/formulare.md` nach jeder Phase fortschreiben (DoD-Häkchen + Befunde).

## Konflikt-Vermeidung (Hot Files mit Haupt-Terminal)
Das Haupt-Terminal ändert parallel `i18n/messages/*.json` (berichte.builder.*-Keys) und evtl. `module-settings-registry.tsx`. Damit der spätere Merge sauber bleibt:
- **i18n:** Füge deine `formulare.*`-Keys hinzu, fass berichte.*-Keys nicht an. Beim Merge-Konflikt in den 4 JSONs: **beide Blöcke behalten**.
- **module-settings-registry.tsx:** Nur deinen formulare-Eintrag ergänzen. Beim Konflikt: beide Einträge behalten.
- **Gezielt committen:** `git add <konkrete Dateien>`, NIE `git add -A` (zieht sonst gitignorierte Helfer-Skripte `add-*-i18n.mjs` rein). Neue Dateien unter `mocks/data/` brauchen `git add -f` (`.gitignore` blockt `data/`).

## Merge (am Ende, wenn alle Phasen grün)
```bash
git push -u origin parallel/formulare          # Branch pushen
```
Das **Haupt-Terminal merged** `parallel/formulare` → `main` (koordiniert mit Darien), damit es nicht mit dem berichte-Merge kollidiert. Wenn du selbst mergst: erst `git fetch`, dann `git checkout main && git pull --ff-only && git merge parallel/formulare` — Konflikte nur in den Hot Files (beide Blöcke behalten). Wenn main lokal linear voraus ist: direkt `git push` (fast-forward), KEIN `git pull --rebase` nach lokalem Merge-Commit.

## Definition of Done (formulare review-reif)
☐ Demo-Mode lebendig (0 failedApiRequests/pageErrors, Listen + Details + Submissions gefüllt)
☐ Formular-Zeile + Submission-Zeile → DetailModal mit allen Infos/Aktionen
☐ DnD-Formular-Builder funktioniert, alle Feldtypen, Live-Vorschau, Reihenfolge persistent
☐ DSGVO-Einwilligungsfeld + Consent im Submission-Detail + echter Export-Download
☐ Moduleinstellungen (personal + tenant) via ModuleSettingsShell
☐ i18n ×4 vollständig, 0 Raw-Keys, 0 `{{}}`-Doppelklammern (per Screenshot verifiziert)
☐ `reviews/formulare.md` mit DoD-Häkchen + Befunden, je Phase ein Commit/Push
