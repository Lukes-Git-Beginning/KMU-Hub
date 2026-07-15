# Bexio-Invoice-Pull — Review-Paket (für neues Terminal)

> **Erstellt Session #7 (2026-07-15). Schritt 1 der Darien-Sequenz.** Ziel: Darien reviewt Lukes Subagent-gebautes Bexio-Invoice-Pull-FE hands-on. Dieses Doc hat alles, um das ohne Rückfragen zu ermöglichen.
> **App-Start:** `cd desktop && npx electron-vite dev --mode demo` (Demo-Modus). CosmiLaunch-Splash läuft kurz, dann drin. **Kein Docker/Backend nötig — außer für die 2 Mocks unten.**

## ⚠ KERN-BEFUND: Feature ist im Demo-Modus NICHT demonstrierbar

Das Bexio-FE ist **vollständig gebaut** (Wizard + Invoice-Pull-Toggle, Read-only-Badge/Banner, alle 6 mutierenden Aktionen `!isExternal`-geguarded, i18n ×4 komplett). **Aber im Demo-Modus sieht Darien die zwei Kernteile NICHT** — das ist selbst ein Review-Befund (Demo-Tiefe-Lücke):

1. **Keine `source='bexio'`-Rechnung im Seed** (`mocks/data/invoices.ts` — alle 19 Rechnungen ohne `source`-Feld) → Read-only-Ansicht (Badge/Banner/fehlende Buttons) wird von keiner Rechnung ausgelöst.
2. **Keine Mock-Handler für die Bexio-Sync-Endpoints** → `mocks/handlers/settings.ts:33-35` liefert nur `/integrations/bexio/status` → `{connected:false}`. Fehlend: `/sync/status`, `/sync/logs`, `/oauth/authorize`, `/sync/trigger`, `/sync/config`, `/mappings/:entity`. → Sync-Dashboard bleibt leer, Wizard hängt auf Schritt 1.

## → ERST BAUEN (damit reviewbar, ~10–15 Min, reine Demo-Mocks)

**Mock A — Bexio-Rechnung in `mocks/data/invoices.ts`:** eine Rechnung mit `source: 'bexio'` ergänzen (Struktur der bestehenden Rechnungen übernehmen; Status z.B. `'paid'` — Bexio besitzt den Zahlstatus). Damit greift der `isExternal`-Pfad in `InvoiceDetailPanel`.

**Mock B — Bexio-Sync-Mocks in `mocks/handlers/settings.ts` (oder `mocks/handlers/integration.ts`):**
- `/integrations/bexio/status` auf `{connected:true}` (damit die Karte das **Dashboard** statt des Wizards öffnet).
- `GET /integrations/bexio/sync/status` → Shape aus `BexioSyncDashboard.tsx:127-153` ableiten: `invoice_pull_enabled:true`, `last_invoice_pull_at`, `total_invoices_mapped`, dito contact/quote/payment-Felder.
- `GET /integrations/bexio/sync/logs` → ein paar Log-Einträge (Shape aus `bexio-client.ts` `listSyncLogs`).

Response-Shapes exakt aus `api/bexio-client.ts` + `api/bexio-types.ts` (flache Wire-Shape!) ableiten. Danach: scoped tsc + eslint + Screenshot-QA (Launch-Skip: `sessionStorage['cosmi:launch-played']='1'` im addInitScript — s. `scripts/qa-video-actions.mjs`).

## Review-Checkliste (was Darien prüft, sobald demonstrierbar)

**① Wizard** — *Einstellungen (Zahnrad) → Tab „Integrationen" (nur Admin) → Bexio-Karte „Konfigurieren"*
- `BexioSetupWizard.tsx` — 4 Schritte (OAuth `:254` / Sync-Richtungen `:313` / Feld-Zuordnung `:478` / Erster Sync `:490`)
- **Invoice-Pull-Toggle** `BexioSetupWizard.tsx:436-448` (Schritt 2), + Intervall-Dropdown 1/5/10/15 Min `:451-472`. Config-Persist beim Finish `:121-132`.

**② Sync-Dashboard** — Bexio-Karte (wenn connected) → „Dashboard"
- `BexioSyncDashboard.tsx:127-153` — 4 Karten: Kontakte / **Rechnungen** (`:134-139`) / Angebote / Zahlungen (je Aktiv/Inaktiv + Anzahl + letzte Sync)

**③ Read-only-Rechnung** — *Finanzen → Tab „Rechnungen" → Bexio-Rechnung anklicken → `InvoiceDetailPanel`*
- `isExternal = invoice.source === 'bexio'` (`InvoiceDetailPanel.tsx:141`)
- **Badge „Bexio"** neben Nummer (`:217-221`) · **blauer Info-Banner** (`:261-268`)
- **Ausgeblendet** (alle `!isExternal`): Edit `:571` · Send `:576` · RecordPayment `:586` · MarkPaid `:592` · Cancel `:601` · Storno `:612`

**④ Design/Verständlichkeit** — ist klar *dass* die Rechnung aus Bexio kommt + *warum* schreibgeschützt? Passt's zur Cosmi-Ästhetik?

## Gefundene Risiken (Darien mitgeben / prüfen)
- **PDF-Download ohne `!isExternal`-Guard** (`InvoiceDetailPanel.tsx:562-569`) — bleibt bei Bexio-Rechnungen sichtbar. Gewollt? Liefert `/finance/invoices/:id/pdf` für gespiegelte Rechnungen ein echtes PDF?
- **Doppelte Datei:** `IntegrationSettingsTab.tsx` (alt, `BexioConfigPanel`) vs `IntegrationsSettingsTab.tsx` (neu, Wizard) — prüfen ob beide geroutet werden = toter Code aufräumen.

## i18n
Vollständig ×4 (de/en/fr/it): `finanzen.invoiceDetail.externalReadonlyBadge` („Bexio"), `...externalReadonlyBanner`, `settings.integrations.bexio.setup.invoicePullLabel/invoicePullHelp`.
