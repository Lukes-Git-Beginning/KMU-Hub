# Batch-NEXT2 · Paket B (SUB-Terminal) — formulare → review-reif

> **10-Phasen-Autonom-Batch** ([[feedback_ten_phase_autonomous_batch]]). Fährt das **Sub-Terminal** (`KMU-Hub-review`, Port 5174).
> **Disjunkt zu Paket A** (kommunikation, Haupt-Terminal). Berührt NUR `modules/formulare/**`, `mocks/handlers/formulare.ts`, `mocks/data/form*.ts`, i18n-Keys `formulare.*`.

## Ziel
Das **formulare**-Modul (Form-Builder) von „gebaut" auf **review-reif → Nico** bringen: Demo-Tiefe-Standard + P1 (DnD-Reordering, DSGVO-Feld, E-Mail-Benachrichtigung) + Feldtypen + bedingte Logik + Einreichungen-Ansicht + Analytics/Embedding + Settings.

## Ist-Stand (verifiziert 2026-06-22, grob)
- `modules/formulare/` ~5508 Zeilen, aber nur ~2 Dateien → **monolithische Komponenten** (beim Bauen ggf. in Unterkomponenten zerlegen, sauber halten).
- MSW-Handler: `mocks/handlers/formulare.ts`. **Settings-Dir existiert bereits** (prüfen, ob `ModuleSettingsShell` oder altes Muster → ggf. heben).
- Tracker P1 (DnD + DSGVO-Feld + Mail) ist der Kern; P2 Webhook/CRM 🔒 + P4 Zahlungen/E-Signatur 🔒 = Luke → NICHT in diesem Paket.

## Recherche-Auftrag (VOR dem Bauen — Gate)
1. Wie tief ist formulare? (Builder mit welchen Feldtypen? Einreichungen-Ansicht da? Vorschau? send/save — wirken sie, oder Toast-Stub? MSW stateful?)
2. Welche der 10 Phasen unten sind schon (teil-)erledigt → streichen/zusammenfassen.
3. Demo-Tiefe-Audit: Formular/Einreichung als `shared/DetailModal`? ganze Zeile klickbar? Export echt oder Stub?
4. Settings: ist `modules/formulare/settings` schon `ModuleSettingsShell` (personal+tenant) oder altes Flat-Muster?

## Gate-Fragen an Darien (gebündelt stellen, dann bauen)
- **Feldtypen-Umfang:** welche Typen sind Pflicht (Text, Textarea, Auswahl, Checkbox, Datei-Upload, Datum, Bewertung, Unterschrift-Mock)?
- **Bedingte Logik:** Felder ein-/ausblenden je nach Antwort — ja, und wie tief (1 Regel/Feld oder Regel-Builder)?
- **DSGVO-Feld:** nur Einwilligungs-Checkbox + Hinweistext, oder auch Aufbewahrungs-/Löschfrist pro Formular?
- **Analytics-Tiefe:** Einreichungen über Zeit + Conversion reicht, oder Feld-Drop-off-Analyse?

## Phasen (vorläufig — beim Gate verfeinern)
- [ ] **FO-1** Demo-Tiefe-Standard: stateful MSW (speichern/einreichen persistieren), Formular/Einreichung als `shared/DetailModal`, ganze Zeile klickbar, tote Buttons/Stubs fixen ([[feedback_module_depth_standard]] [[feedback_detail_modal_standard]])
- [ ] **FO-2** Builder: DnD-Reordering der Felder (sortierbar, dnd-sicher) + Feld duplizieren/löschen
- [ ] **FO-3** DSGVO-Einwilligungs-Feld + Pflicht-Consent + Datenschutz-Hinweis-Block (Aufbewahrungsfrist je Gate)
- [ ] **FO-4** E-Mail-Benachrichtigung bei Einreichung: Empfänger-Config + Mock-Versand + Bestätigungs-Mail an Absender
- [ ] **FO-5** Feldtypen vervollständigen (Datei-Upload, Datum, Auswahl/Radio, Bewertung, Unterschrift-Mock) je Gate
- [ ] **FO-6** Bedingte Logik (Felder ein-/ausblenden je nach Antwort) — Live in Vorschau + Einreichung
- [ ] **FO-7** Einreichungen-Ansicht: Liste + `shared/DetailModal` + Filter/`shared/SortMenu` + echter CSV-Export ([[feedback_recurring_ux_patterns]])
- [ ] **FO-8** Analytics (Einreichungen über Zeit, Conversion, optional Feld-Drop-off) + Embedding-Snippet (iFrame/JS-Code zum Kopieren)
- [ ] **FO-9** Settings-Panel `modules/formulare/settings` auf `ModuleSettingsShell` heben (personal: Standard-Empfänger; tenant: DSGVO-Defaults, Aufbewahrung, erlaubte Feldtypen)
- [ ] **FO-10** i18n ×4 Vollständigkeit (`formulare.*`, `{var}`, ICU-Plural) + Demo-Tiefe-Schlusscheck + Playwright-QA + Screenshots WIRKLICH ansehen

## Disjunktheit / Konflikt-Vermeidung
- **NUR** `modules/formulare/**`, `mocks/handlers/formulare.ts`, `mocks/data/form*.ts`.
- **Finger weg von** `modules/kommunikation/**`, `mocks/handlers/chat.ts` (= Paket A).
- i18n (`de/en/fr/it.json`): nur `formulare.*`-Keys → rebased additiv mit Paket A.

## Build-+-Verify-Standard (pro Phase)
bauen → i18n ×4 → gescopter tsc (foreground, echter Exit, NIE `| tail`) → `eslint src/ --quiet` ([[feedback_lint_before_push]]) → Playwright-Screenshot-QA + PNGs ansehen → ein Commit (explizite Pfade, `mocks/data/` braucht ggf. `git add -f`) → push (`pull --rebase`). Scoped tsconfig anlegen (`tsconfig.formularecheck.json` existiert ggf. schon — prüfen). Latenz: vorbestehende latente tsc-Fehler ignorieren.
