# Branchen-Block — Demo-Tiefe auf Standard (×7)

> **Auftrag (Darien, 2026-07-15):** Die 7 Branchen-Module auf **Demo-Tiefe review-reif**, **voll auf Standard** — damit Luke das Backend gebündelt bauen kann und der Onboarding/Info-Center-Block alle Modul-Funktionen abbilden kann.
> **Basis:** verifiziertes Audit (2 Explore-Agents, Session #9). Grundgerüst + P1-Backend-Wiring stehen; es fehlt der Demo-Tiefe-Feinschliff.
> **Voraussetzung:** frisches Terminal, erst `git pull`. Dev-Server + Screenshot-QA wie gehabt.

---

## ✅ STAND Session #13 (2026-07-16): 7 von 7 fertig — BRANCHEN-BLOCK KOMPLETT

| Modul | Commit | Referenz-tauglich für |
|---|---|---|
| ✅ **inventar** (Pilot) | `1d1fac6c` | Modal-Umbau, Settings-Stores, SortMenu, CSV, Sub-Listen-Back-Kette, Workflow-Vervollständigung (Inventur-Zählung) |
| ✅ **vermietung** | `aafe636a` | Status-Lifecycle-Aktionen aus ungenutzten Hooks, Kalender-Slot→Modal, Konfliktcheck + tenant-Policy (bufferDays) |
| ✅ **rapporte** | `23c39644` | **echter PDF-Export ohne Library** (`modules/rapporte/rapporte-export.ts`, WinAnsi-Umlaute), Prefs seeden Dialog-Defaults, tenant-Policy blockt Aktion (Unterschrift-Pflicht) |
| ✅ **schichten** | `9cb06ab7` | Grid-Zellen-Klick→Modal mit Inline-Aktions-Formular (Tausch), tenant-Policy gatet Modal-Aktion (swapEnabled) + speist Berechnung (maxWeeklyHours→ArbZG) + Dialog-Default (defaultBreakMinutes), Drag&Drop-Selbst-Drop-Guard |
| ✅ **fuhrpark** | `10a32584` | **Mock-Bugs via Screenshot-QA gefunden** (fehlender GET /services-Handler = Tab immer leer, Seed-Feld-Drift scheduled_date→scheduled_at, km nicht berechnet = NaN), Zeilen-Detail-Modals für Tabellen-Tabs, Client-Typ-Re-Exports, plateFor-Lookup (Adapter tragen nur vehicle_id), Platzhalter-Datum 2099→"—" |
| ✅ **einkauf** | `be6ad91a` | **Listen-Query-ohne-Relationen-Bug** (Detail/Wareneingang zogen `po.lines` aus der Liste → immer leer; Fix: `usePOLines` im Modal), **mock-first-Endpoint bei BE-Lücke** (cancelPO + MSW + backend-gaps 🔒), Warenkorb→Sammelbestellung pro Lieferant, Query-Invalidierung über Entitätsgrenzen (Line-Mutationen → PO-Liste, total_amount!), Seeds relativ via date-helpers |
| ✅ **produktion** | `b4472b6d` | **Statuswechsel-Endpoints lagen 7/7 ungenutzt bereit** (start/complete/cancel in Client UND BE mit Transition-Guards — kein Luke nötig!), **stateless-MSW-Befund** (create/patch/status mutierten nichts → neue Aufträge/QS-Prüfungen verschwanden beim Refetch), **Gantt mit Fix-Datumsfenster = im Demo immer leer** (Feb-2026-Range → rollierendes Heute-Fenster), echter Fortschritt aus WorkSteps statt hardcoded 50 %, Ausschuss aus QC-defects statt hardcoded 0, Schritt-Abhaken via ungenutztem useUpdateWorkStep, 4-Modal-Back-Kette (Order↔BOM/QC/Maschine), mock-first `bom_id` (BE-Order ohne BOM-Link 🔒) |

**Erprobtes Rezept pro Modul (aus 3 Durchläufen, je ~1 Commit):**
1. **Marktrecherche-Agent zuerst** (Web-Research-Sub-Agent, ~3 Min parallel zum Code-Lesen): Detail-Ansicht / Status-Lifecycle / Settings personal-vs-tenant / Exporte / Listen-UX der 3–4 Marktführer. Ergebnis bestimmt Settings-Felder + Workflow-Lücken. Paritäts-Funde (zu groß für Demo-Tiefe) im RESUME-NEXT notieren, nicht bauen.
2. **API-Schicht VOR dem Bauen greppen** — in allen 3 Modulen existierten ungenutzte Hooks/Endpoints (Inventur-Create+Counts, startRental/endRental/Inspections). Erst schauen, was da ist; oft ist der „tote Button" nur unverdrahtet.
3. **Dateien pro Modul:** `modules/<m>/<X>DetailModal.tsx` (+ ggf. Zweit-Modal mit onBack-Kette) · `modules/<m>/<m>-shared.ts` (Status-Maps/Helper aus der Page extrahieren) · `modules/<m>/<m>-export.ts` (CSV; downloadCsv/downloadBlob wiederverwenden) · `stores/<m>Prefs.ts` + `stores/<m>Tenant.ts` (Muster videoPrefs/dialerTenant; ⚠ Namenskollision prüfen — vermietungPrefs war ein Daten-Store → vermietungViewPrefs) · `modules/<m>/settings/<M>SettingsPanel.tsx` · Registry-Eintrag + Hydrator ×2 · i18n ×4 · `tsconfig.<m>check.json` · `scripts/qa-<m>-tiefe.mjs`.
4. **Wiederkehrende Alt-Bugs mitfixen:** Dialog-stale-state (Remount-key `key={open ? (editItem?.id ?? 'new') : 'closed'}`) · totes `getExportUrl`/`getExportPDFUrl` mit fehlendem `API_BASE_URL` in `api/<m>-client.ts` (**fuhrpark-client.ts:227 hat es noch!**) · Zeilen/Karten ohne role=button+Tastatur.
5. **Gate:** scoped tsc (Baseline-Fehler nur in unberührten Dateien ok) → eslint --quiet → QA-Skript (Muster qa-rapporte-tiefe.mjs: Modal, Downloads via waitForEvent('download'), SortMenu, Settings-Panel-Text, Raw-Keys, pageerrors) → **Bilder ansehen** → Commit + Push (Auto-Deploy).

---

## Das gemeinsame Muster — jedes Branchen-Modul braucht dieselben 5 Dinge

Alle Referenzen wurden in Session #9 (video) frisch gebaut — direkt abkupfern:

1. **`DetailPanel` (Slide-over) → `shared/DetailModal`** (zentriertes Cosmi-Fenster).
   - `DetailModal` ist **API-kompatibel zu `DetailPanel`** (`open/onClose/title/subtitle/badge/footer/children`) → meist nur Import + Component-Name tauschen. Referenz: `components/shared/DetailModal.tsx` + `modules/video/CallHistoryDetailModal.tsx` (Meta-Grid + Sektionen + Footer-Aktionen).
   - **Ganze Zeile klickbar** (`role="button"` + `onClick`, innere Buttons `stopPropagation`). Auch Sub-Listen (Wartung/Tanken, Reservierungen, Lagerorte, Qualität) klickbar machen → Detail-Modal.

2. **Settings-Panel registrieren** (personal + tenant scope). Keins der 7 hat einen Eintrag.
   - `moduleId` = der Modul-Name (rapporte/schichten/fuhrpark/vermietung/inventar/einkauf/produktion) — **alle sind schon `ModuleId`/`LEADABLE_MODULES`** (`lib/module-settings.ts`), also **kein `SettingsModuleId`-Typ-Update nötig**.
   - Bauen wie video (#9): `stores/<modul>Prefs.ts` (personal, nach `stores/crmPrefs.ts`) + `stores/<modul>Tenant.ts` (tenant, nach `stores/dialerTenant.ts`), beide in `hooks/useHydrateModuleSettings.ts` registrieren. Panel nach `modules/dialer/settings/DialerSettingsPanel.tsx` / `modules/video/settings/VideoSettingsPanel.tsx`. Eintrag in `modules/settings/module-settings-registry.tsx` (`id`, `navMatch: ['/<route>']`, `component`). `moduleSettings.entries.<id>` i18n ×4.

3. **Tote Buttons / Toast-Stubs raus** — echte Handler oder wirksame Aktionen (kein `toast.success()` ohne Effekt). Downloads = **echter Blob** (Muster: `triggerTextDownload` in `CallHistoryDetailModal.tsx`, oder `modules/finanzen/lib/finance-export.ts` `downloadCsv`).

4. **`SortMenu`** (`components/shared/SortMenu`) wo Listen sortierbar sein sollten (Feld + Richtung). Muster: `modules/notifications/NotificationCenter.tsx` / heutiges team `TeamPage.tsx`.

5. **`shared/EmptyState`** für alle leeren Zustände (kein leerer Screen, kein Custom-Div).

**Gate pro Modul (Build-+-Verify-Standard):** bauen → i18n ×4 (`{var}`, nie `{{var}}`; ICU-Plural) → scoped tsc (nur geänderte Dateien) → eslint geänderte Dateien → Playwright-Screenshot-QA **+ Bilder wirklich ansehen** → iterieren bis grün → ein Commit + Push (PUSH-MODE → Auto-Deploy). Scoped-tsc: eigene `tsconfig.<modul>check.json` anlegen (Muster `tsconfig.videocheck.json`).

---

## Reihenfolge (Rest)

**KEINE — alle 7 Module fertig.** ~~Pilot: inventar~~ ✅ · ~~vermietung~~ ✅ · ~~rapporte~~ ✅ · ~~schichten~~ ✅ · ~~fuhrpark~~ ✅ · ~~einkauf~~ ✅ · ~~produktion~~ ✅ (siehe Stand-Block oben). Weiter geht's mit Onboarding/Info-Center O-0 + Bexio-Review-Punkte — `.planning/RESUME-NEXT.md` Top-Block.

---

## Audit-Findings pro Modul (Session #9, verifiziert)

| Modul | Aufwand | Konkrete Fehlstellen |
|---|---|---|
| **inventar** | klein | „Neue Inventur"-Button = toter `toast.success()` (`InventarPage.tsx:1237`) → echter API-Dialog · Lagerort-Karten nicht klickbar (kein Detail) · Artikel-Detail = `DetailPanel`→Modal · kein Export · kein Settings-Panel · kein SortMenu |
| **vermietung** | klein | Reservierungs-Zeilen kein Click-to-Detail · Kalender-Slot-Klick auf belegt = nur Toast (`VermietungPage.tsx:1113`) · Objekt-Detail = `DetailPanel`→Modal · kein Export · kein Settings-Panel |
| **rapporte** | klein | PDF-Export = Toast-Stub ohne Blob (`RapportePage.tsx:800`) → echter Download · Detail = `DetailPanel`→Modal · kein Settings-Panel · kein SortMenu |
| **schichten** | mittel | Grid-Zelle (belegt) = nur Toast (`SchichtenPage.tsx:554`) → Detail-Modal · „Vorlage bearbeiten" toter Button (`:818`) → Dialog · PDF-Export Toast-Stub (`:797`) · kein Settings-Panel |
| **fuhrpark** | mittel | `AddTripDialog` „not yet implemented" (`FuhrparkPage.tsx:1206`) → bauen · Wartung/Tanken-Zeilen kein Detail · Fahrzeug-Detail = `DetailPanel`→Modal · kein Export · kein Settings-Panel |
| **einkauf** | mittel | mehrere tote `toast.info()`: „Bestellung/Lieferant bearbeiten" (`:354/:836/:1166/:1361`), „Lieferant deaktivieren" (`:841`), „Warenkorb" (`:962`), „Neuer Abruf" (`:1118`) → echte Dialoge/Aktionen · Bestell/Lieferanten-Detail = `DetailPanel`→Modal · kein Settings-Panel |
| **produktion** | **groß** | **Statuswechsel = toter `toast.success()` ohne Mutation** (`ProduktionPage.tsx:472`) — Endpoint mit Luke prüfen · Qualitäts-Tab-Zeilen nicht klickbar · Auftrags-Detail = `DetailPanel`→Modal · kein Export · kein Settings-Panel |

**Projektweit:** kein Modul nutzt `role="button"` auf Zeilen (nur `cursor-pointer`+`onClick`) → beim Modal-Umbau ARIA nachziehen. `SortMenu` fehlt überall.

---

## Wichtige Env / Konventionen (Session #9 verifiziert)
- **moduleId=`meetings`-Lehre gilt sinngemäß:** Branchen-moduleIds sind schon LEADABLE → Settings-Panel direkt baubar.
- **CosmiLaunch-Splash im Screenshot-QA überspringen:** `sessionStorage['cosmi:launch-played']='1'` im Playwright `addInitScript` (Muster: `scripts/qa-video-tiefe.mjs` / `qa-notif-tiefe.mjs`).
- **Blob-Download-Helper** inline (Muster `CallHistoryDetailModal.tsx` `triggerTextDownload`) oder `finance-export.ts`/`mail-export.ts` `downloadBlob`.
- **i18n** flach in `i18n/messages/{de,en,fr,it}.json` (Punkt-Notation-Keys, `{var}` nicht `{{var}}`). Alle 4 Sprachen pflegen.
- **Nicht committen:** `deploy/docker/docker-compose.flags.yml`, `desktop/scripts/qa-dialer-callflow.mjs` (bleiben untracked).

**Voller Wiedereinstieg:** `.planning/RESUME-NEXT.md` #9-Block. Master-Plan: `.planning/MASTER-PLAN.md` §3 (Branchen).
