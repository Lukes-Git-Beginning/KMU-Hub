# Berichte → echtes Report-Authoring-System — VISION & Plan (Handoff)

> **Stand 2026-06-20.** Großer Pivot nach Darien-Feedback: Das berichte-Modul ist aktuell „nur ein Modul zum Grafiken erstellen". Es soll ein **vollwertiges Bericht-Authoring-System** werden — mehrseitige Dokumente (Text + Titel + Grafiken), Lebenszyklus, Lese-Modus, professionelle PDF, Scheduling, Versand, Cross-Modul-Integration.
> **Der bisher gebaute Builder (E-0…E-5, `.planning/berichte-builder-plan.md`) ist NICHT weg** — er wird zur untersten „Grafik-Widget"-Schicht (ein Baustein im Dokument).
> **Nächster Schritt: neues Terminal startet hier**, stimmt die offenen Entscheidungen (unten) mit Darien ab, baut dann in Phasen R-0…R-6.

---

## 1. Darien-Vision (O-Ton-Anforderungen, vollständig)

1. **Tab-Benennung:** Der Tab „Erstellen" (nach Dashboard) sollte „Berichte" o.ä. heißen — dort liegen ja auch die gespeicherten Berichte.
2. **Automatische Berichte:** Möglichkeit, Berichte automatisch erstellen zu lassen (z.B. jede Woche). **Am Bericht selbst** definiert (beim Erstellen/Bearbeiten), NICHT in Modul-Einstellungen verstreut.
3. **Feld-/Parameter-Auswahl massiv ausbauen:** Auswählen können, was genau im Bericht erscheint (Kunden oder alles Mögliche, was in einem Modul relevant ist) — **für JEDES Modul nachschauen**, welche Daten relevant sind.
4. **Berichte öffnen/lesen:** Aktuell kann man keinen Bericht öffnen/lesen. Man muss Berichte **ansehen** können (Lese-Modus), nicht nur im Bearbeiten-/Erstellen-Modus.
5. **Echte mehrseitige Berichte (Dokument-Charakter):** 10–20+ Seiten mit **Text, Titel, allem** im Tool. Auswählen können, **wo die Grafiken im Bericht erscheinen** und welche Parameter dafür wichtig sind.
6. **Lebenszyklus + Verteilung:** speichern · dokumentieren · versenden · als **fertig markieren** · an zuständige Personen **freigeben/verschicken**.
7. **Cross-Modul-Integration (Leitbeispiel):** Ein 20-seitiger Verkaufsbericht mit Grafiken wird erstellt → als fertig markiert → freigegeben/verschickt → jemand liest ihn → will ihn als **Aufgabe** an einen Mitarbeiter geben („analysieren + Verkaufsmodell anpassen") → **Bericht an die Aufgabe dranhängen** (dafür muss er ein Dokument sein ODER beim Aufgabe-Erstellen anhängbar) → vielleicht geht der Bericht auch an **jemanden ohne Cosmi** → dann muss es eine **1a-PDF** sein, die wie ein vollständiger echter Bericht aussieht.

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

- **Editor-Paradigma → BLOCK-basiert** (wie Notion), NICHT Canvas (Power BI Report Builder / Crystal Reports). Begründung: KMU-Anwender schreiben einen Monatsbericht wie einen Word-Brief (oben anfangen, runterarbeiten). Canvas = Support-Hölle (Pixel-Positionierung). Dokument fließt top-down, Blöcke per „+" einfügen, automatische Paginierung beim PDF-Export, Seitenumbruch als expliziter Block.
- **MVP-Block-Set (10):** Deckblatt (Titel/Logo/Datum/Autor) · H1 · H2 · Fließtext (Rich) · **Chart (= eingebettetes E-1-Widget)** · Datentabelle · Kennzahl-Highlight (große Zahl + Trend) · Trennlinie · Seitenumbruch · Bild/Logo. Kopf-/Fußzeile mit Seitenzahl = globales Report-Setting (kein Block). **Phase 2:** Auto-Inhaltsverzeichnis (aus H1/H2), Callout/Zitat, Kommentare.
- **Lebenszyklus → 4 Status, KEIN Approval-Gate** (zu bürokratisch für KMU): `Entwurf → Fertig → Freigegeben → Archiviert`. Entwurf = nur Ersteller bearbeitet, kein Versand. Fertig = intern lesbar. Freigegeben = read-only für Berechtigte, Scheduling aktiv, externer Link generierbar. **Snapshot** bei Freigabe + bei jedem Schedule-Run (Datum+Lauf-ID). Keine Git-artige Versionierung.
- **Lese- vs. Bearbeiten-Modus:** zwei klar getrennte States. Lese-Modus = sauberes Dokument ohne UI-Chrome (wie PDF-Vorschau), auch das was externe Empfänger über Share-Link sehen. PDF-Download-Button prominent im Lese-Modus.
- **PDF-Strategie → Playwright Headless Chromium SERVER-SEITIG (= Luke 🔒).** CSS `@page { size:A4 }`, `page-break-inside:avoid` (keine abgeschnittenen Charts/Tabellen), `@page:first` (Deckblatt ohne Kopfzeile), Plus-Jakarta-Sans eingebettet, SVG-Charts (vektorbasiert). `mini-pdf.ts` reicht NICHT (text-only, single-page). Paged.js optional für laufende Kopf-/Fußzeilen. **Was eine 1a-PDF ausmacht (Prio):** Deckblatt > durchgehende Kopf-/Fußzeile mit Logo+„Seite 3 von 8" > keine abgeschnittenen Blöcke > Inhaltsverzeichnis mit Seitenzahlen (Phase 2) > eingebettete Schriften.
- **Scheduling:** am Bericht konfiguriert (Rhythmus täglich/wöchentlich/monatlich, Empfänger intern+extern, Format PDF). Guard: nur Status ≥ Freigegeben wird versendet. Snapshot pro Run.
- **Verteilung:** intern (rollenbasiert, in Modul-Liste) · extern (Token-Read-Link, kein Login, optional Ablauf/Passwort) · anhängen an Aufgabe/Deal/Kontakt (Referenz-Link, nicht Datei-Kopie → kein Versionschaos).
- **Bewusst weglassen (KMU-MVP):** Canvas/Pixel-Layout · Conditional-Formatting/Ausdrucks-Engine (SSRS) · daten-getriebene Subscriptions · Kommentar-Threads (Phase 2) · iFrame-Embeds (brechen im PDF) · Branching/Merge-Versionierung.

---

## 4. Infrastruktur-Ist (was da ist / was fehlt)

**✅ Schon da (wiederverwenden!):**
- **TipTap-RichTextEditor:** `components/shared/RichTextEditor/RichTextEditor.tsx` (v3.20), voll ausgebaut (Toolbar/BubbleMenu, `readOnly`/`compact`-Props, `onChange(html)`, Extensions: link/image/table/task-list/text-align/underline/placeholder). **Direkt für Fließtext-Blöcke + Lese-Modus nutzbar.** Lazy: `LazyRichTextEditor`. (Hinweis: wiki nutzt es noch NICHT, sondern ein Textarea — Inkonsistenz, aber egal für uns.)
- **dokumente-Upload:** `POST /api/v1/documents/files/upload` (multipart, `folder_id`) → für PDF-Ablage. Kein `source_type:'report'`-Feld → via Standard-Tag „Bericht" oder Feld ergänzen.
- **work Task-Files:** `GET/POST /api/v1/tasks/:id/files` (Metadaten-POST, kein Multipart), `DELETE /task-files/:id`. Task-Create nimmt KEINEN Anhang direkt → Anhang als 2. Schritt nach Erstellen.
- **Scheduling:** `ReportSchedule` (cron, recipients, format, params/alert_threshold, toggle) + volle CRUD in `mocks/handlers/berichte.ts`. **`definition_id` koppelt Schedule schon 1:1 an einen Bericht** → „Zeitplan einrichten"-Button im Editor = nur `definition_id` vorbelegen, kein Schema-Change.
- **Builder E-0…E-5:** Source→Felder→Aggregation→Filter→DateRange→Viz→Style→Preview→Save + MyReportsLibrary + Dashboard-Pin. `BuilderQueryConfig` als `query_config` (JSONB).

**❌ Fehlt / Lücken:**
1. **Echte PDF-Engine** — kein jspdf/pdfmake/pdf-lib/puppeteer in package.json; nur text-only single-page `mini-pdf`. → **Playwright server-seitig = Luke** (kritischster Backend-Bedarf).
2. **ReportDocument-Schema** — `ReportDefinition` hat kein `blocks[]`/`body_html`/`status`/Deckblatt-Meta. Braucht Erweiterung (oder neue Entität `ReportDocument` die Widgets referenziert).
3. **Report-Sources fehlen** für hr/zeiterfassung, vertraege, einkauf, fuhrpark, rapporte (je `*.source.ts` + Registry-Zeile). Bestehende 5 Quellen sind zudem zu dünn (nur 5–7 Felder) → reiche Feldlandkarte (Teil 6) einarbeiten.
4. **vertraege ohne `contract_value`** — Measure fehlt im Modell.
5. **Lese-Modus, Lebenszyklus-Status, externer Share-Link, Snapshots** — komplett neu.
6. **dokumente-Ablage-Trigger nach Export + Aufgaben-Anhang-Flow** — Endpoints da, Verkabelung neu.

---

## 5. Phasen-Plan (Vorschlag R-0…R-6 — vor R-1 mit Darien Editor-Paradigma bestätigen)

| # | Phase | Inhalt |
|---|-------|--------|
| **R-0** | Fundament + Tab | `ReportDocument`-Schema (blocks[], status, cover-meta, settings) · Tab „Erstellen"→„Berichte" umstrukturieren (Bibliothek + „Neuer Bericht") · MSW-CRUD für Dokumente |
| **R-1** | Block-Dokument-Editor | Block-Liste (Deckblatt/H1/H2/Text/Chart/Tabelle/KPI/Trennlinie/Seitenumbruch/Bild), „+"-Einfügen, Drag-Reorder (dnd-kit), **Fließtext = TipTap-`RichTextEditor`**, **Chart-Block bettet E-1-Widget ein** (Quelle/Felder/Viz inline wählen). |
| **R-2** | Lese-Modus + Lebenszyklus | Lese↔Bearbeiten-Toggle · Status Entwurf→Fertig→Freigegeben→Archiviert · Snapshot bei Freigabe · Bericht **öffnen/lesen** aus Bibliothek. |
| **R-3** | PDF-Export (1a) | Print-CSS (`@page`, page-break-inside:avoid, Deckblatt, Kopf-/Fußzeile+Seitenzahl) · **Server-PDF via Playwright = Luke** (Übergabe-Doc). Übergangsweise `window.print()`-Pfad für Demo. |
| **R-4** | Scheduling am Bericht | „Automatisch erstellen+senden"-Einstellung im Editor (Rhythmus/Empfänger/Format) → koppelt an `definition_id` · Guard Status≥Freigegeben · Lauf-Historie. |
| **R-5** | Integration | PDF → dokumente ablegen · Bericht an Aufgabe anhängen (work `tasks/:id/files`, + Option beim Task-Erstellen) · externer Token-Read-Link. |
| **R-6** | Datenquellen-Ausbau | Alle Module als reiche Sources (Feldlandkarte Teil 6): hr/zeiterfassung, vertraege(+value), einkauf, fuhrpark, rapporte + bestehende 5 vertiefen. |

Reihenfolge justierbar. R-6 kann früh teil-parallel laufen (Sub-Terminal-tauglich: je `*.source.ts` disjunkt).

---

## 6. Modulspezifische Berichtsfelder (Feldlandkarte — für R-6 / Schicht 2)

> Reiche Liste aus Cosmi-MSW/Types. Pro Modul: Entität → Schlüssel-Dimensionen / Measures.

- **finanzen:** Rechnung (issue_date/due_date·status[draft/sent/paid/overdue/cancelled]·customer·currency·tax_rate·payment_terms | total_net/total_gross/tax) · Angebot (status[accepted/rejected]·…) · Gutschrift (is_storno·reason) · **Mahnung** (level 1/2/3·status·fee·interest) · **Ausgabe** (category·supplier·project·account·status | amount) · Transaktion (matchStatus) · Wiederkehrend (interval·status·next_run·generated_count) · Unbilled-Time (duration_hours·hourly_rate·amount·billed).
- **kontakte/crm:** Kontakt (created_at·title·company·tags·country) · Firma (industry·country·contactCount) · **Deal** (stage[lead/qualified/proposal/negotiation/won/lost]·owner·close_date | value·probability) · Aktivität (type[call/email/meeting/note]·count) · Segment/Tags.
- **work:** Projekt (status·priority·owner·dates·is_template | progress·task_count·completed·member_count) · **Aufgabe** (status_name·priority·assignee·due_date·project·tags | estimated_hours·subtasks·comments) · Zeit-Eintrag (user·is_manual·billed | duration_seconds) · Task-Datei · Abhängigkeit.
- **helpdesk:** Ticket (status[open/pending/solved/closed]·priority·assignee·queue·created/resolved | response_mins·resolution_mins·count) · SLA (status[on_track/at_risk/breached]) · Queue · Stats (open/avg_response/resolved_this_week/CSAT).
- **kommunikation/inbox:** Nachricht (received_at·channel[email/chat/notification]·is_read/starred/archived·assigned_to·tags·crm_contact | response_mins·count).
- **team/hr:** Mitarbeiter (department·position·contract_type[full/part/mini]·start_date | work_days/leave_days) · Abwesenheit (type·status·days) · Urlaubskonto (total/used/remaining/pending) · Personaldoc (category·fileSize).
- **zeiterfassung:** Zeit-Eintrag (date·project·customer·activity·status·billable | totalMinutes/netWork/break) · Analytic-Rollup (billable/nonbillable/overtime/workedDays) · Team-Woche (department·weekStatus | weekMinutes/target/overtime).
- **vertraege:** Vertrag (contract_type[rental/service/employment/nda]·status[draft/active/expired/terminated]·starts/ends | Laufzeit·count) ⚠ `contract_value` fehlt im Modell · Erinnerung (reminder_type·status).
- **dokumente:** Datei (created_at·mime_type·space[personal/team/project]·creator·tags·favorite | file_size) · Version.
- **einkauf:** Bestellung (status·order_date·supplier·currency | total_amount) · Lieferantenbewertung (category·rating) · Rahmenvertrag (status·total/used_value).
- **fuhrpark:** Fahrzeug (make/model/year·fuel_type·status·tuev_due | mileage_km) · Tankbuchung (| liters·cost) · Fahrtenbuch (purpose·is_private | km) · Schaden (severity·status·cost).
- **rapporte:** Arbeitsrapport (status[draft/submitted/approved/rejected]·report_date·author·reviewer·count) · Rapport-Zeile (unit | quantity).
- **automatisierung/notifications:** Trigger-Typ·Aktionstyp·status·execution_count (als „Automatisierungs-Audit").

---

## 7. Offene Entscheidungen für Darien (vor R-1)

1. **Editor-Paradigma bestätigen:** Block-basiert (Empfehlung, Notion-Stil) vs. Canvas/Seiten-Designer? → Mockup-Vergleich zeigen.
2. **Bericht ↔ dokumente:** eigenständig im berichte-Modul (mit PDF-Export→dokumente-Ablage + Aufgaben-Anhang als Brücke)? (Analog zur formulare-Entscheidung: eigenständig + Brücke.)
3. **Scope/Reihenfolge:** R-0…R-6 wie oben, oder Prioritäten anders (z.B. PDF früher, weil „1a-PDF an Externe" Kern-Wunsch)?
4. **Status-Modell:** 4 Status ohne Approval-Gate ok, oder doch Freigabe-Gate?
5. **Externer Share-Link:** im Scope, oder erstmal nur PDF-Versand an Externe?

## 8. Backend-Bedarf für Luke (kritisch)
- **PDF-Engine (Playwright Headless Chromium, server-seitig)** — der einzige echte Blocker für „1a-PDF". HTML→PDF mit `@page`/page-break/Schrift-Einbettung. Bis dahin Demo via `window.print()`.
- ReportDocument-Persistenz (blocks JSONB, status, snapshots), Schedule-Run-Executor + Mailer, externer Token-Link, dokumente-`source_type:'report'`.

---

## Verweise
- Bisheriger Grafik-Builder (Schicht 3): `.planning/berichte-builder-plan.md` (E-0…E-5, fertig).
- Datenquellen-Registry-Muster: `desktop/src/renderer/src/modules/berichte/report-sources/`.
- Wiederverwenden: `components/shared/RichTextEditor`, `shared/DetailModal`, `shared/SortMenu`, `ChartRenderer` (E-1).
