# RESUME — Batch 4 (berichte R-5 Integration + R-6 Datenquellen)

> **Startklar, sobald (a) Darien das berichte-R-3/R-4-Review durch hat und (b) das Sub-Terminal wiki fertig ist.**
> Haupt-Terminal (`KMU Hub`, 5173) = **berichte R-5 (Integration)** — Darien gewählt 2026-06-20.
> Sub-Terminal (`KMU-Hub-review`, 5174) = **berichte R-6 (Datenquellen)** — hoch parallelisierbar.
> Spec: `.planning/berichte-report-authoring-VISION.md` (R-5, R-6). Workflow: [[feedback_sub_terminal_5phase]].
> Review-Findings aus R-3/R-4 (FIX-Phasen ZUERST einschieben, falls vorhanden): `.planning/berichte-r3r4-review-findings.md`.

## Stand
- berichte R-3/R-4 fertig + gepusht: B5-1 `185cdf84` · B5-2 `5afce9b2` · B5-3 `b295a40d` · B5-4 `16a811ea` · B5-5 `db46a6e9`. Backend-gaps `2332875e`.
- **Reihenfolge:** erst Dariens R-3/R-4-Review-Fixes (als FIX-* aus findings-Datei), DANN B6-*.
- Nach R-5 + R-6 ist der Report-Authoring-Strang R-0…R-6 komplett → **berichte review-reif für Nico**. (Separat offen bleibt „Erstellen-Builder E-1…E-5".)

## SCHRITT 0 (beide Terminals): `git pull --rebase origin main`

---

## Hauptterminal · berichte R-5 = B6-1…B6-5 (5 Phasen)

**Ist-Stand gescoutet:** Lese-Modus-Header ist bereits voll (Back/Titel/Badge/Toggle/Zeitplan/Drucken/3-Punkte-StatusActions). R-5-Aktionen daher in ein **konsolidiertes Aktions-Menü** (das bestehende `StatusActions`-Popover in `ReportDocumentEditor.tsx` erweitern oder ein eigenes „Teilen"-Dropdown), NICHT weitere Header-Buttons. work-tasks-Liste + task-files-MSW existieren; CRM-attach + documents-upload-Handler müssen ergänzt werden; `share_token` fehlt im Typ.

| # | Phase | Inhalt | Machbarkeit |
|---|-------|--------|-------------|
| B6-1 | R-5a An Aufgabe anhängen | Aktions-Menü im Lese-Modus + „An Aufgabe anhängen" → Typeahead über `GET /api/v1/tasks` → `POST /tasks/:id/files` `{name:doc.title, source_type:'report', source_id:doc.id, url:lese-link}`. Toast. (Datei-Liste in work zeigt „Bericht-Link"-Icon.) | ✅ FE-mockbar (beide MSW da) |
| B6-2 | R-5a An Kontakt/Deal anhängen | Gleicher Flow, CRM-Ziel. **MSW-Handler ergänzen** (`POST /api/v1/contacts/:id/files` o.ä. — `kontakte.ts` hat noch keinen). Typeahead über Kontakte/Deals. | ✅ FE-mockbar (Handler neu) |
| B6-3 | R-5b PDF in Dokumente ablegen | „Als PDF ablegen" → `window.print`-Blob (R-3a-Surrogat) bzw. Server-PDF-Stub → `POST /documents/files/upload` (**Handler ergänzen**, `documents.ts` hat GET-list + stateful `files`-Array) + Ordner-Picker (Default „Berichte") + Auto-Tag „Bericht". Toast mit Link. | ✅ FE-mockbar (Upload-Handler neu) |
| B6-4 | R-5c Externer Share-Link | `share_token` ins `ReportDocument`-Typ + MSW. Aktions-Menü „Externen Link erzeugen" (Toggle, Ablauf 30/90/unbegrenzt, optional Passwort). Token in MSW + Link-Kopier (`cosmi://share/report/:token` Demo). | 🔒 FE-mockbar; echter externer Zugriff = Luke |
| B6-5 | Geteilte-Links-Übersicht + Polish | Liste aktiver Tokens (Ablauf/Aufrufe-Counter/Widerrufen) im Aktions-Menü. i18n-Audit aller neuen `berichte.docs.share.*`/`berichte.docs.attach.*`-Keys ×4. Demo-Tiefe-Check. | ✅ FE-mockbar |

**Pro Phase:** bauen → i18n ×4 (`{var}`, ICU-Plural) → gescopter Typecheck (`tsconfig.b6check.json` nach Muster `tsconfig.b5check.json`) → Playwright-Screenshot-QA gegen 5173 (Muster `scripts/qa-berichte-b5-*.mjs`) → **PNGs ansehen** → ein Commit (explizite Pfade) → push (`pull --rebase` über Sub-Pushes).
**Backend-Bedarf** (backend-gaps): R-5c externer unauth. Zugriff + echter PDF-Blob (R-3b) — Luke.

---

## Sub-Terminal · berichte R-6 = Datenquellen (copy-paste ins KMU-Hub-review-Terminal, sobald wiki fertig)

```
Du bist das Sub-Terminal im Sub-Terminal-5-Phasen-Modus. Arbeitsverzeichnis = dieser KMU-Hub-review-Klon (Dev 5174). Das Hauptterminal baut parallel berichte R-5 — du fasst NUR report-sources + registry an. Sprache: Deutsch (Umlaute, Eszett).

SCHRITT 0: git pull --rebase origin main

KONTEXT (Ist-Stand): modules/berichte/report-sources/ hat 5 Quellen (finanzen, helpdesk, kommunikation, kontakte, work) + registry.ts + types.ts + sample-utils.ts. SourcePicker liest aus REPORT_SOURCES (registry) → neue Quellen erscheinen automatisch im Chart-Block-Picker. Merge-Punkt = NUR registry.ts.

DEIN BATCH — berichte R-6 (Datenquellen-Ausbau), 5 Phasen je ein Commit:
- R6-1: hr.source.ts (Mitarbeiter/Abwesenheit/Urlaubskonto) + zeiterfassung.source.ts (Zeit-Eintrag/Analytic-Rollup/Team-Woche) — je sampleRows() mit realistischen Demo-Werten + Zeile in registry.ts.
- R6-2: vertraege.source.ts (Vertrag + contract_value als Measure — Feld in vertraege-MSW-Daten ergänzen falls fehlend) + einkauf.source.ts (Bestellung/Lieferantenbewertung/Rahmenvertrag).
- R6-3: fuhrpark.source.ts (Fahrzeug/Tankbuchung/Fahrtenbuch/Schaden) + rapporte.source.ts (Arbeitsrapport/Rapport-Zeile).
- R6-4: Bestehende 5 Quellen auf min. 12 Felder vertiefen (aktuell 5–7) — Feldlandkarte in der VISION Teil 6 als Vorlage. Dimensions + Measures sinnvoll typisieren.
- R6-5: Schlusscheck — alle 11 Quellen im SourcePicker sichtbar, sampleRows liefern Daten, Live-Vorschau rendert je Quelle eine Beispiel-Grafik. QA gegen 5174 (Chart-Block-Picker → Neue Grafik → Quelle wählen). Demo-Tiefe.

BUILD-+-VERIFY PRO PHASE: bauen → (kaum i18n, falls Labels: i18n/messages/{de,en,fr,it}.json, {var} nicht {{var}}) → gescopter Typecheck (desktop/tsconfig.r6check.json nach Muster tsconfig.b5check.json; desktop/node_modules/.bin/tsc -p … --noEmit, foreground) → Playwright-QA gegen http://localhost:5174 (#/berichte → Tab Berichte → Bericht → Diagramm-Block → Neue Grafik → Quelle) → PNGs ansehen → ein Commit.

STANDARDS: keine Emojis, Theme-Tokens, CURRENT_USER aus shared-ids. Conventional Commits, English imperative, KEINE AI-Attribution, EXPLIZITE Pfade (NIE git add -A). Nach Commit push; bei Ablehnung pull --rebase (registry.ts-Konflikte: alle Quellen-Zeilen behalten). Dev-Server 5174 nicht killen.

ABSCHLUSS: Bilanz — committed Phasen (Hashes), QA je Phase, was offen.
```
