# „Anpassungen" (Self-Service-Customization) — Bau-Fortschritt

> SSOT-Konzept: `KONZEPT.md`. Recherche: `IST-A/B/C.md` + `MARKT-A/B/C.md`.
> Stand: 2026-07-21 (Session #25).

## v1 — Fundament-Trio (Overlay-basiert)

| Stufe | Inhalt | Status | Commit |
|---|---|---|---|
| **v1.0** | Overlay-Config-Fundament: Typen + `resolveConfig` (default→vendor→tenant + Provenance) + MSW (labels/value-sets) + RBAC-Key `admin:customization:manage` + Audit + i18n ×4 | ✅ **fertig + verifiziert** | `a43bffcb` |
| **v1.1** | Custom-Fields-Editor (vereinheitlicht, 5 Entitäten, 9 Feldtypen, Progressive Disclosure, Soft-Delete-Schutz) + „Anpassungen"-Hub als 9. Admin-Tab | ✅ **fertig + verifiziert** | `2bdd3407` |
| **v1.2** | Label-Override-Editor („Begriffe"-Tab, Provenance Standard/Von Zentria/Angepasst, Reset, Sprach-Auswahl) | ✅ **fertig + verifiziert** (Live-Wirkung gefixt) | `a57e11d7` + Fix |
| **v1.3** | Value-Sets-Editor (zentrale Wertelisten, Referenz-Auflösung in ≥1 Modul als Proof, Soft-Delete) | ⏳ offen | — |
| **v1.4** | „Anpassungen"-Hub (Admin-Fläche + 3 Editoren + Modul-Schnellzugriffe in `ModuleSettingsShell` + Vendor/Tenant-Herkunfts-Banner + Template-Stub) | ⏳ offen | — |

### v1.0 — Verifikations-Nachweis
- **Dateien:** `api/customization-types.ts` (neu), `mocks/data/customization.ts` (neu), `mocks/handlers/customization.ts` (neu), `handlers/index.ts` (+Registrierung), `config/capability-catalog.ts` (+Key), `mocks/data/rbac.ts` (+it_admin-Grant; admin via `catalogCapabilityKeys('admin')`), `mocks/data/audit-events.ts` (+3 Actions), `scripts/i18n-customization-v10.mjs` (32 Keys ×4), `scripts/qa-customization-resolver.mjs` (12-Test-Smoke), `tsconfig.customcheck.json`.
- **Resolver:** `resolveLabelOverrides(locale, base?)` + `resolveValueSet(id, base?)` → Wert + `provenance: 'default'|'vendor'|'tenant'` je Eintrag. tenant > vendor > default. `base=1` = reine Baseline (R-6-Muster).
- **Gates:** scoped tsc **0 Fehler in allen v1.0-Dateien** (8 Rest = Alt-Baseline in automation/crm/finance/hr, transitiv via index.ts) · `eslint` clean (selbst nachgeprüft, korrekter Pfad) · i18n 32 Keys ×4 mit echten fr/it (selbst stichprobe: Anpassungen/Customization/Personnalisation/Personalizzazione) · Smoke 12/12 (tenant>vendor>default, Provenance, base) · admin-Key-Kette selbst verifiziert.
- **Bewusst ausgeklammert:** Custom Fields (eigene BE-Persistenz → v1.1 andocken), Vendor-Session-Detection (`activeConfigLayer()` = immer 'tenant' bis R-5-GDAP-Wiring).

### v1.2 — Stand + offener Punkt (KRITISCH für nächste Session)
- **Funktioniert + verifiziert:** „Begriffe"-Editor (Sub-Tab im Anpassungen-Hub), Provenance-Badges (Standard grau / Von Zentria blau / Angepasst grün, R-6-Muster), Umbenennen/Reset/„Alle zurücksetzen", Sprach-Auswahl (de/en/fr/it), Persistenz via MSW, Audit-Events. i18n ×4 echt, scoped tsc 0 Fehler, eslint clean.
- **2 Fixes von mir (verifiziert):** (a) **Default-Anzeige** — „Cosmi-Standard" zeigte den gemergten Wert statt des echten Code-Defaults (getResourceBundle nach addResourceBundle kontaminiert). Fix: `getLabelDefault` liest einen Snapshot, der in `useLabelOverlay.captureDefaults` VOR dem Merge eingefroren wird. QA-bestätigt („Support-Desk / Cosmi-Standard: Helpdesk"). (b) **Tote Whitelist-Keys** — `nav.crm/nav.work/nav.admin.label` haben KEINEN Konsumenten im FE; die Sidebar rendert `layout.navItems.*`. Fix: Whitelist + Vendor-Seed + BegriffeTab-Gruppe auf die echten `layout.navItems.contacts/projects/tasks/team/finance/helpdesk/admin` umgestellt.
- **✅ GELÖST (Session #25, root-cause statt Symptom):** Die bisherige Diagnose („`changeLanguage(sameLocale)` erzeugt kein neues `t`, useMemo re-computed nicht") war **falsch**. Quellen-Verifikation: i18next 26 emittiert `languageChanged` auch bei gleicher Locale (`done()` läuft immer, `i18next.js:1981`); react-i18next 17 recomputt `t` bei jedem Revision-Bump (`useTranslation.js:77-82`) → `t` wechselt sehr wohl die Identität. **Echte Ursache: `i18next-icu` memoized kompilierte MessageFormat-Instanzen nach `lng.ns.key` (`i18nextICU.js:4235`) und liefert `bindI18n: ''` + `bindI18nStore: ''` als Default (`:4178-4181`) → `clearCache` wird NIE an ein Event gebunden.** Sobald die Sidebar `t('layout.navItems.contacts')` einmal rendert (cached „Kontakte"), liefert der ICU-Cache diesen Wert für immer — jeder `addResourceBundle`+Re-Render liest den veralteten Formatter. Der Editor (React Query, kein ICU) zeigt korrekt, die Sidebar (ICU-gecacht) bleibt stale. Betraf **beide** Fälle: Vendor-Bootstrap UND Tenant-Live.
  - **Fix (`i18n/i18n.ts`):** `.use(new ICU({ bindI18nStore: 'added removed', bindI18n: 'languageChanged' }))` — das Store-`added`-Event von `addResourceBundle` leert jetzt den ICU-Cache, `languageChanged` deckt Sprach-Wechsel. Ein Einzeiler an der Wurzel; `applyLabelOverlay`/Bootstrap unverändert.
  - **Isoliertes Experiment (kontrolliert):** DEFAULT-Config → `t()` bleibt „Kontakte" nach addResourceBundle+changeLanguage (STALE ✗); mit Fix → „Klienten" (WORKS ✓, greift schon nach addResourceBundle allein).
  - **QA `qa-customization-v12-final.mjs` 3/3 PASS + Screenshot angesehen:** P1 Sidebar zeigt „Patienten" (Vendor-Bootstrap) ✓ · P2 Umbenennen→„Klienten" → Sidebar **live „Klienten" ohne Reload** ✓. scoped tsc 0 Fehler in geänderten Dateien (nur 11 Alt-Baseline in hr/finance/automation/crm/sanitize) · eslint `i18n.ts` clean.

## Nächste Stufen = UI-Editoren (visuell + Screenshot-QA-pflichtig)
v1.1–v1.4 bauen sichtbare Oberflächen → Standard-Gate inkl. Playwright-Screenshot-QA + Bilder ansehen. Jede Stufe: Agent-Bau + Review + i18n ×4 + scoped tsc + eslint + Screenshot-QA + 1 Commit.

---

# NEU-AUSRICHTUNG: Modul-Editor (ab Session #26, 2026-07-22)

> **v1.3/v1.4 der alten Roadmap SUPERSEDED.** Darien-Richtung nach v1.0–v1.2: die Admin-Listen sind zu mager + modul-entkoppelt. Neu = **modul-zentrischer edit-in-place-Editor im eigenen Fenster** (Sandbox, isoliert vom Live). Recherche-Gate durchlaufen (`IST-EDITOR.md` + `MARKT-EDITOR.md` + `DRAFT-DEPLOY.md`), 4 Entscheide gefallen. **SSOT = `EDITOR-KONZEPT.md`.** Das Fundament v1.0–v1.2 BLEIBT die Maschinerie (Resolver, ICU-Live-Fix, Custom-Fields, Value-Sets, Audit, RBAC-Key); nur die UX wird neu.

**4 Entscheide (Darien 2026-07-22):** ① In-App-Overlay (self-contained für späteren Fenster-Umzug) ② nur Trio-Panel v1, Layout = Stufe 2 ③ Terminierung gleich in v1 (Job gemockt, Luke-Cron) ④ 2 Pilot-Module (Kontakte + Helpdesk).

## Editor-Bau-Phasen

| Phase | Inhalt | Status | Commit |
|---|---|---|---|
| **E-1 Fundament** | Draft-Schicht (4. Overlay-Ebene) im Resolver · `DraftConfigProvider` · Drafts-Store (save/deploy/schedule/rollback + Scheduler-Mock) · Audit-Actions | ✅ **fertig + verifiziert** | (dieser Commit) |
| **E-2 EditorFrame** | Overlay-Rahmen (self-contained: Sandbox-QueryClient + MemoryRouter + Provider) · Toolbar · Amber-Draft-Banner · Drei-Panel · Commit-Footer | ⏳ offen | — |
| **E-3 Trio-Panel** | `CustomFieldsTab`+`BegriffeTab` integrieren (modul-gefiltert) · `ValueSetsTab` neu · Live-Preview-Verdrahtung | ⏳ offen | — |
| **E-4 Modul-Galerie** | `AnpassungenHubPage` → Kacheln (Kontakte + Helpdesk) · editierbares Manifest pro Modul (Vendor-Ebene) | ⏳ offen | — |
| **E-5 Deploy** | Übernehmen-Dropdown (Jetzt/Terminiert/Entwurf) · Deploy-Dialog + DatePicker + Ankündigung · Entwurf-Liste · Rollback | ⏳ offen | — |

### E-1 — Verifikations-Nachweis
- **Typen (`api/customization-types.ts`):** `ConfigProvenance` +`'draft'` (4. Stufe, gewinnt) · `LocaleLabelMap` exportiert · `CustomizationDraft`/`CustomizationDraftPayload`/`DraftStatus`(draft→scheduled→live→superseded)/`DeployMode`/`SaveDraftInput`/`DeployDraftInput`.
- **Resolver (`mocks/data/customization.ts`):** `resolveLabelOverrides(locale, base?, draftOverlay?)` + `resolveValueSet(id, base?, draftOverlay?)` — draft-Schicht gewinnt per Key/Option über tenant. Neu: `applyDraftToTenant()` (Promotion in tenant-Layer, nur Whitelist-Keys, zählt Applied) · `snapshotTenant()`/`restoreTenant()` (Rollback-Basis) · `TenantSnapshot`-Typ.
- **Drafts-Store (`mocks/data/customization-drafts.ts`, neu):** `saveDraft` (Blueprint, kein Impact) · `commitDraftNow`/`scheduleDraft`/`deployDraft` (3-Wege) · `promote()` (snapshot→apply→live, altes live→superseded) · `runDueScheduledDeploys()` (Scheduler-Mock, steht für Lukes Cron) · `rollbackDeploy()`/`canRollback()` · Audit `draft_saved/draft_deleted/deploy_live/deploy_scheduled/rolled_back`.
- **Provider (`modules/admin/anpassungen/editor/DraftConfigProvider.tsx`, neu):** React-Context+Reducer, hält Draft-Session-State, `setDraftLabel/resetDraftLabel/setDraftValueSet/resetDraftValueSet/resetAll/buildPayload/loadDraft`, `isDirty`+`changeCount`. Live-Preview: `applyLabelOverlay(resolveLabelOverrides(locale, false, draftLabels))` bei jeder Änderung. **R-1-Scrub:** beim Unmount `restoreLabelOverlay(locale, touchedKeys, resolveLabelOverrides(locale))` → Draft-only-Keys zurück auf Default (verhindert Live-Kontamination).
- **i18n-Helfer (`i18n/useLabelOverlay.ts`):** `restoreLabelOverlay(locale, scrubKeys, resolved)` neu (setzt Scrub-Keys auf `getLabelDefault`, re-applied dann das persistente Overlay).
- **BegriffeTab:** `draft`-Provenance-Badge (amber) ergänzt (Folge der Typ-Erweiterung).
- **Gates:** scoped tsc (`tsconfig.customcheck.json` +2 Dateien) **0 Fehler in allen E-1-Dateien** (11 Rest = Alt-Baseline sanitize/automation/crm/finance/hr) · `eslint` clean · **vitest `customization-draft.test.ts` 9/9 PASS** (draft>tenant-Merge, draft>default, base ignoriert draft, Value-Set-Draft-Option+Name, saveDraft-kein-Impact, commitDraftNow-Promotion, rollback-Restore, applyDraftToTenant-Whitelist-Filter, Scheduler due-vs-future).
- **E-1 hat KEINE UI** → statt Playwright ein Logik-Vitest. Screenshot-QA ab E-2 (EditorFrame sichtbar).
- **Luke-Paket (backend-gaps, nach FE-Bau):** `tenant_customization_drafts`-Tabelle (sparse, `scheduled_at`, Status-Maschine) · Promotion-Cron · Draft-Overlay serverseitig (nur Editor-Session) · Rollback modul-granular · Audit an Deploy-Routen · editierbares Manifest pro Modul.
