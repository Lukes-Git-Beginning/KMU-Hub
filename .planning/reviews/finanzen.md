# finanzen — Modul-Review / Fortschritt

> Modul: `desktop/src/renderer/src/modules/finanzen/` (Sidebar „Buchhaltung", Route `/finanzen`).
> Strategie: **Symbiose** (`finanzen-buchhaltung-strategy.md`) — eigenständige Faktura-Kette + DATEV/Bexio-Export, kein Buchhaltungs-Ersatz. `modules/buchhaltung/` bleibt dead/untouched.
> Re-Plan P1–P5 (Darien greenlit 2026-06-14, ersetzt den alten „buchhaltung löschen"-P1):

| Phase | Inhalt | Status |
|---|---|---|
| **P1** | Faktura-Kette schließen: wiederkehrende Rechnungen · Fremdwährung (EUR/CHF/USD) · OP-Liste · Storno↔Originalrechnung | ✅ **done** (`c597c32`) |
| **P2** | Ausgaben/Belege + Kontierung: Expenses/Transactions Zustand→MSW · Beleg-Upload · SKR03/04 + manuelle Kontierung · Berichte-Tab migrieren | ✅ **done** (`d5061af`) |
| **P2.5** | **Demo-Tiefe & Detail-Views (NEU, aus Audit 2026-06-16, Darien-Vorgabe).** Jede Unterkategorie öffnet eine echte Detail-/Inhaltsansicht (wie Dokumente/Kontakte); jeder Download/Export erzeugt eine sichtbare Datei/Viewer im Demo; tote Mutationen + Placeholder-Daten verkabeln. Detail siehe unten. **Empfehlung: läuft als Nächstes, vor/teils parallel zu P3.** | ✗ |
| P3 | DATEV/Bexio-Export (Settings „Für alle"): DATEV-EXTF-Flow (Berater-/Mandanten-Nr + Konto-Mapping) · Bexio-OAuth-UI · BMD-CSV · fehlende MSW-Handler | ✗ |
| P4 | E-Rechnung (ZUGFeRD/XRechnung) + GoBD-Belegarchiv (Launch-Blocker, FE + Luke-BE) | ✗ |
| P5 | Banking (CAMT.053/MT940 + Auto-Matching) · EÜR-Auswertung (recharts) · finanzen-Moduleinstellungen-Feinschliff | ✗ |

## P1 — was gebaut wurde (2026-06-14)
Neue Dateien: `OpenItemsTab.tsx`, `RecurringInvoicesTab.tsx`, `RecurringInvoiceDialog.tsx`, `mocks/data/finance-recurring.ts`.
Geändert: `types/finance-types.ts` (Currency/Recurring-Typen + optionale `total_net/total_gross`), `stores/finance.ts` (`formatMoney`), `mocks/handlers/finance.ts` (stateful: create/cancel/credit-note-create/recurring-CRUD/generate), `mocks/data/invoices.ts` (3 CHF-Dokumente), `api/finance-client.ts` + `api/hooks/useFinance.ts` (recurring), `FinanzenPage.tsx` (2 Tabs + Storno-Actions + Währungs-Anzeige), `InvoiceFormDialog`/`QuoteFormDialog` (Währung+Kurs), `CreditNoteDialog` (Storno-Modus), `InvoiceDetailPanel` (verknüpfte Gutschriften + Storno-Button + währungsbewusst). i18n ×4 (67 Keys, ICU-Plurals).

**2 echte Bugs per Screenshot-QA gefangen + gefixt:** (1) Gutschriften-Tab crashte (`cn.original_invoice_id.slice` auf undefined — Seed-Notes hatten nur `invoice_number`); (2) Invoice/Quote/CreditNote-Dialoge crashten beim Befüllen aus Listen-Daten (`line_items` undefined, Liste liefert `items`). Beide Populate-/Normalize-Pfade nehmen jetzt beide Shapes.

## P2 — was gebaut wurde (2026-06-16)
Neue Dateien: `lib/skr-accounts.ts` (SKR03/04-Kontenrahmen + Kategorie→Konto-Vorschlag), `lib/receipt-cache.ts` (Session-Blob für Beleg-Vorschau, CSP-safe), `stores/financeTenant.ts` (Kontenrahmen tenant-weit), `mocks/data/finance-ledger.ts` (`git add -f`!), `ReceiptPreviewDialog.tsx`, `KontierungSettings.tsx`, `tabs/BerichteTab.tsx`.
Geändert: `ExpenseFormDialog.tsx` (war null-Stub → echtes Create/Edit-Formular mit Konto-Auto-Vorschlag + Beleg-Upload), `tabs/ExpensesTab.tsx` (Konto-Spalte + „Kontieren"-Warnung bei unkontiert + Beleg-Indikator/Vorschau), `mocks/handlers/finance.ts` (stateful expenses CRUD/approve/reject/receipt + transactions list/delete), `api/finance-client.ts` (financeExpenseApi/financeTransactionApi), `api/hooks/useFinanceLedger.ts` (Zustand→MSW, +useUpdateExpense/useAttachReceipt), `FinanzenPage.tsx` (Berichte-Tab registriert), `stores/finance.ts` (Expense.account/receiptName + FinanceTabKey 'berichte'), `FinanceSettingsPanel.tsx` (Sektion „Kontierung & Kontenrahmen"). i18n ×4 (27 Keys, ICU single-brace).

**Strategie-Klärung (Darien greenlit):** Kontierung = leichtes Sachkonto-Mapping (Auto-Vorschlag aus Kategorie, manuell überschreibbar) NUR als Daten-Vorbereitung für den DATEV-Export — KEINE Buchungssätze/Soll-Haben (strategie-konform). Beleg-Upload swap-ready: kein Backend → nur Metadaten (Dateiname) + Blob-Cache für Vorschau; Seed-Belege zeigen Platzhalter.

**QA (Playwright, alle Screenshots angesehen):** Ausgaben-Liste, Neu-Formular (Konto auto-vorgeschlagen), „Kontieren"-Edit (zeigt Vorschlag statt „Kein Konto"), Beleg-Vorschau (Platzhalter), Berichte (datengetriebene Monats-Bars + Kategorien), Transaktionen (MSW intakt), Settings-Kontierung (SKR03-Select). Keine Bugs. **Lehre bestätigt:** Onboarding-Modal („Willkommen bei Cosmi") blockt erste Klicks → in QA-Scripts zuerst „Überspringen" klicken.

## P2.5 — Demo-Tiefe & Detail-Views (Audit 2026-06-16)
Darien-Befund: „In den Unterkategorien gibt es überall Download/Export-Positionen, aber Klick → nichts passiert. Sollte sich wie bei Dokumente/Kontakte öffnen und Inhalt/Infos zeigen — aktuell sehr oberflächlich." Voll-Audit über alle Tabs bestätigt das. Ursache: viele Aktionen sind Toast-Stubs oder rufen Endpoints, die im Demo-Mode (MSW) gar nicht existieren. **4 Buckets:**

**A. Downloads/Exports erzeugen nichts Sichtbares**
- PDF-Vorschau (`PDFPreviewPanel`) = graue Platzhalter-Kästchen statt echtem, datengetriebenem Dokument; Download/Drucken/Vollbild/Zoom/Seiten = Toast oder tot.
- PDF-Download Rechnungen/Angebote/Gutschriften/Mahnungen → `getPDF` ruft `/…/pdf` (kein MSW) → Klick schlägt fehl.
- DATEV-Export (`ExportDialog`) → `/finance/export/datev` (kein MSW). Bexio-CSV + BMD-NTCS (Export-Tab) = nur `toast.success`, keine Datei.
- QR-Rechnung („PDF herunterladen"/„Drucken"/„An Rechnung anhängen") + E-Rechnung („XML herunterladen"/„Erneut validieren") = nur Toast; QR-Code selbst = grauer Platzhalter mit Hardcoded-Daten.
- → **Ziel:** jeder Download erzeugt eine echte (Demo-)Datei/Blob + Viewer; PDF-Vorschau aus echten Rechnungsdaten rendern. (DATEV-CSV-Inhalt = P3, echte ZUGFeRD/XRechnung-Erzeugung = P4 — hier nur die Verkabelung/sichtbares Verhalten.)

**B. Detail-/Inhaltsansichten fehlen pro Unterkategorie** (Kern von Dariens Wunsch — wie Dokumente/Kontakte)
- Nur **Rechnungen** haben ein Detail-Panel (`InvoiceDetailPanel`). KEINE Detail-/Inhaltsansicht bei: **Angebote, Gutschriften, Wiederkehrend, Transaktionen, Ausgaben** (nur Edit-Form), **Mahnungen**, **OP-Liste** (Zeile → sollte Rechnung öffnen), **Dashboard-Mini-Listen** (Zeilen nicht klickbar), **Berichte** (Balken nicht klickbar → kein Drill-down).
- → **Ziel:** Zeilenklick öffnet überall eine Detail-Ansicht mit allen Infos (Positionen, Historie, Status, Verknüpfungen, Beleg) — konsistentes Muster, idealerweise wiederverwendbares `shared/DetailPanel`-Pattern.

**C. Tote Mutationen verkabeln (MSW fehlt)**
- Angebot senden/annehmen/ablehnen/in Rechnung umwandeln; Zahlung erfassen (`POST /payments`); Mahnung erkennen/senden/eskalieren + Mahn-Konfig speichern; Settings speichern (`PUT /settings`). Aktuell Klick → Fehler statt State-Änderung.

**D. Placeholder-Daten auf MSW heben**
- **Banking** (`BankingWidget`): Konten/Transaktionen Hardcoded; „Verbinden"/„Synchronisieren"/Match-Buttons = Toast/ohne Handler. **Belegkette**: `mockChains` hardcoded, „Zum Kunden"/„Nächster Schritt" tot. **Stunden→Rechnung** (`HoursToInvoiceDialog`): `mockTimeEntries` hardcoded, „Rechnung erstellen" = Toast (legt keine Rechnung an). **GoBD-Audit-Log** (im Invoice-Detail): Hardcoded Fake-Einträge.

**Größe:** umfangreich, eigener Phasen-Block (ggf. in 5er-Batch-Unterschritte splitten: B-Detailviews zuerst, dann A-Downloads, dann C/D). Vollständige Aktions-für-Aktions-Tabelle liegt im Audit-Output der Session 2026-06-16.

## Geparkt — technischer Rename `finanzen` → `buchhaltung` (mit Luke klären)
Darien will den technischen Modulnamen an das User-Label „Buchhaltung" angleichen. **Noch NICHT umsetzen** — erst mit Luke besprechen (Merge-Risiko, er arbeitet ggf. parallel). Umfang wenn beschlossen: toten `modules/buchhaltung/`-Ordner löschen/mergen · `modules/finanzen/` → `modules/buchhaltung/` (alle Imports) · Route `/finanzen` → `/buchhaltung` (Registry-navMatch hat beide schon) · i18n-Namespace konsolidieren (aktuell gemischt `finanzen.*` + `buchhaltung.*`) · Stores optional. Eigener, isolierter Commit.

## Backend-Gaps (für Luke) — in `backend-gaps.md` §finanzen ergänzen
Echte Endpoints für: recurring (CRUD/generate/pause), credit-note create+send, invoice cancel, quote-Mutationen (send/accept/reject/convert), payments record. **NEU P2:** `/finance/expenses` (GET/POST/PUT/approve/reject/receipt/DELETE) + `/finance/transactions` (GET/DELETE) — aktuell nur stateful MSW. Beleg-Upload braucht echten File-Upload-Endpoint (+ GoBD-Archiv-Verknüpfung). P3/P4: DATEV-EXTF-Generierung (SKR-Konten kommen jetzt aus `Expense.account`), ZUGFeRD/XRechnung-XML, GoBD-Archiv, Banking (FinAPI/CAMT/MT940), Bexio-OAuth.

## Lehren aus dieser Phase (wichtig für nächste)
- **`.gitignore` Zeile 60 `data/` ignoriert das ganze `mocks/data/`-Verzeichnis.** Neue Mock-Daten-Dateien MÜSSEN per `git add -f` geadded werden, sonst bricht der Build für alle (bestehende data-Dateien sind nur tracked, weil früher force-added).
- **Kalter tsc ist hier kaputt:** crasht mit internem Compiler-Fehler („Debug Failure. No error for last overload signature", Overload-Auflösung, u.a. `calcRemainingAmount`). Kein nutzbares Gate. Echtes Gate = **Vite-Build (kompiliert alle Module, App läuft) + Playwright-Screenshot-QA**. IDE-LSP-MCP war in dieser Session nicht verbunden.
- **Playwright-QA ohne MCP:** Standalone-Skript (`node qa.cjs` mit `playwright`-Paket), Screenshots in PNG → mit Read-Tool ansehen. App auto-authentifiziert in Demo-Mode (kein Login nötig), Renderer auf `localhost:5173`, Route `/#/finanzen`. Dev-Server: `npm run dev` (electron-vite, startet auch Electron-Fenster). Nach QA Dev-Server + Electron per PowerShell killen.
- **Mock-Shape ≠ Type:** Listen-Endpoints liefern `items`/`total_gross`/`customer_name` (nicht im `Invoice`-Type), Detail-Endpoint normalisiert zu `line_items`/`tax_breakdown`. Beim Lesen von Listen-Daten beide Shapes bedenken.
