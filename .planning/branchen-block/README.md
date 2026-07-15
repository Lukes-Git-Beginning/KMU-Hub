# Branchen-Block — Demo-Tiefe auf Standard (×7)

> **Auftrag (Darien, 2026-07-15):** Die 7 Branchen-Module auf **Demo-Tiefe review-reif**, **voll auf Standard** — damit Luke das Backend gebündelt bauen kann und der Onboarding/Info-Center-Block alle Modul-Funktionen abbilden kann.
> **Basis:** verifiziertes Audit (2 Explore-Agents, Session #9). Alle 7 sind heute **TEILWEISE** (produktion **NUR GRUNDGERÜST**). Grundgerüst + P1-Backend-Wiring stehen; es fehlt der Demo-Tiefe-Feinschliff.
> **Voraussetzung:** frisches Terminal, erst `git pull` (main ≥ `1627b7c2`). Dev-Server + Screenshot-QA wie gehabt.

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

## Reihenfolge

1. **Pilot: `inventar`** (klein, deckt ALLE 5 Muster-Elemente ab) — **Spec: `pilot-inventar.md`**. Zuerst komplett bauen → Darien-Review → Muster steht.
2. Danach die restlichen 6 nach dem Pilot-Muster, aufsteigend nach Aufwand, gut für 2-Terminal (disjunkte Module, keine Hot-File-Kollision außer i18n-JSONs + `module-settings-registry.tsx` + `useHydrateModuleSettings.ts` → die serialisieren, nicht doppelt gleichzeitig editieren):
   - **klein:** vermietung, rapporte
   - **mittel:** schichten, fuhrpark, einkauf
   - **groß:** produktion (Statuswechsel-Mutation fehlt komplett — ggf. mit Luke abklären ob Endpoint da; sonst mock-first + 🔒-Zeile)

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
