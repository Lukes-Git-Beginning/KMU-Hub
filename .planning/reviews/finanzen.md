# finanzen — Modul-Review / Fortschritt

> Modul: `desktop/src/renderer/src/modules/finanzen/` (Sidebar „Buchhaltung", Route `/finanzen`).
> Strategie: **Symbiose** (`finanzen-buchhaltung-strategy.md`) — eigenständige Faktura-Kette + DATEV/Bexio-Export, kein Buchhaltungs-Ersatz. `modules/buchhaltung/` bleibt dead/untouched.
> Re-Plan P1–P5 (Darien greenlit 2026-06-14, ersetzt den alten „buchhaltung löschen"-P1):

| Phase | Inhalt | Status |
|---|---|---|
| **P1** | Faktura-Kette schließen: wiederkehrende Rechnungen · Fremdwährung (EUR/CHF/USD) · OP-Liste · Storno↔Originalrechnung | ✅ **done** (`c597c32`) |
| P2 | Ausgaben/Belege + Kontierung: Expenses/Transactions Zustand→MSW · Beleg-Upload · SKR03/04 + manuelle Kontierung · Berichte-Tab migrieren | ⏳ next |
| P3 | DATEV/Bexio-Export (Settings „Für alle"): DATEV-EXTF-Flow (Berater-/Mandanten-Nr + Konto-Mapping) · Bexio-OAuth-UI · BMD-CSV · fehlende MSW-Handler | ✗ |
| P4 | E-Rechnung (ZUGFeRD/XRechnung) + GoBD-Belegarchiv (Launch-Blocker, FE + Luke-BE) | ✗ |
| P5 | Banking (CAMT.053/MT940 + Auto-Matching) · EÜR-Auswertung (recharts) · finanzen-Moduleinstellungen-Feinschliff | ✗ |

## P1 — was gebaut wurde (2026-06-14)
Neue Dateien: `OpenItemsTab.tsx`, `RecurringInvoicesTab.tsx`, `RecurringInvoiceDialog.tsx`, `mocks/data/finance-recurring.ts`.
Geändert: `types/finance-types.ts` (Currency/Recurring-Typen + optionale `total_net/total_gross`), `stores/finance.ts` (`formatMoney`), `mocks/handlers/finance.ts` (stateful: create/cancel/credit-note-create/recurring-CRUD/generate), `mocks/data/invoices.ts` (3 CHF-Dokumente), `api/finance-client.ts` + `api/hooks/useFinance.ts` (recurring), `FinanzenPage.tsx` (2 Tabs + Storno-Actions + Währungs-Anzeige), `InvoiceFormDialog`/`QuoteFormDialog` (Währung+Kurs), `CreditNoteDialog` (Storno-Modus), `InvoiceDetailPanel` (verknüpfte Gutschriften + Storno-Button + währungsbewusst). i18n ×4 (67 Keys, ICU-Plurals).

**2 echte Bugs per Screenshot-QA gefangen + gefixt:** (1) Gutschriften-Tab crashte (`cn.original_invoice_id.slice` auf undefined — Seed-Notes hatten nur `invoice_number`); (2) Invoice/Quote/CreditNote-Dialoge crashten beim Befüllen aus Listen-Daten (`line_items` undefined, Liste liefert `items`). Beide Populate-/Normalize-Pfade nehmen jetzt beide Shapes.

## Backend-Gaps (für Luke) — in `backend-gaps.md` §finanzen ergänzen
Echte Endpoints für: recurring (CRUD/generate/pause), credit-note create+send, invoice cancel, quote-Mutationen (send/accept/reject/convert), payments record. P3/P4: DATEV-EXTF-Generierung, ZUGFeRD/XRechnung-XML, GoBD-Archiv, Banking (FinAPI/CAMT/MT940), Bexio-OAuth.

## Lehren aus dieser Phase (wichtig für nächste)
- **`.gitignore` Zeile 60 `data/` ignoriert das ganze `mocks/data/`-Verzeichnis.** Neue Mock-Daten-Dateien MÜSSEN per `git add -f` geadded werden, sonst bricht der Build für alle (bestehende data-Dateien sind nur tracked, weil früher force-added).
- **Kalter tsc ist hier kaputt:** crasht mit internem Compiler-Fehler („Debug Failure. No error for last overload signature", Overload-Auflösung, u.a. `calcRemainingAmount`). Kein nutzbares Gate. Echtes Gate = **Vite-Build (kompiliert alle Module, App läuft) + Playwright-Screenshot-QA**. IDE-LSP-MCP war in dieser Session nicht verbunden.
- **Playwright-QA ohne MCP:** Standalone-Skript (`node qa.cjs` mit `playwright`-Paket), Screenshots in PNG → mit Read-Tool ansehen. App auto-authentifiziert in Demo-Mode (kein Login nötig), Renderer auf `localhost:5173`, Route `/#/finanzen`. Dev-Server: `npm run dev` (electron-vite, startet auch Electron-Fenster). Nach QA Dev-Server + Electron per PowerShell killen.
- **Mock-Shape ≠ Type:** Listen-Endpoints liefern `items`/`total_gross`/`customer_name` (nicht im `Invoice`-Type), Detail-Endpoint normalisiert zu `line_items`/`tax_breakdown`. Beim Lesen von Listen-Daten beide Shapes bedenken.
