# „Anpassungen" (Self-Service-Customization) — Bau-Fortschritt

> SSOT-Konzept: `KONZEPT.md`. Recherche: `IST-A/B/C.md` + `MARKT-A/B/C.md`.
> Stand: 2026-07-21 (Session #25).

## v1 — Fundament-Trio (Overlay-basiert)

| Stufe | Inhalt | Status | Commit |
|---|---|---|---|
| **v1.0** | Overlay-Config-Fundament: Typen + `resolveConfig` (default→vendor→tenant + Provenance) + MSW (labels/value-sets) + RBAC-Key `admin:customization:manage` + Audit + i18n ×4 | ✅ **fertig + verifiziert** | `a43bffcb` |
| **v1.1** | Custom-Fields-Editor (vereinheitlicht, 5 Entitäten, 9 Feldtypen, Progressive Disclosure, Soft-Delete-Schutz) + „Anpassungen"-Hub als 9. Admin-Tab | ✅ **fertig + verifiziert** | `2bdd3407` |
| **v1.2** | Label-Override-Editor („Begriffe"-Tab, Provenance Standard/Von Zentria/Angepasst, Reset, Sprach-Auswahl) | ⚠️ **gebaut — 1 offener Punkt** | `a57e11d7` |
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
- **⚠️ OFFEN (USP-Kern, vor v1.2-Abschluss zu lösen):** Die **globale Live-Wirkung** der Label-Overrides greift NUR bei Komponenten mit direktem `t()` im Render, NICHT bei `useMemo`-gecachten wie der **Sidebar** (`useFilteredNavItems` memoized `t(item.label)`). Ursache: `applyLabelOverlay` ruft `i18n.changeLanguage(sameLocale)` — das erzeugt kein neues `t`, also re-computed das useMemo nicht. Folge: gespeicherte Umbenennungen erscheinen im Editor (React Query), aber die Sidebar/gecachte Flächen aktualisieren nicht (auch nicht nach Reload via Bootstrap). Playwright-Headless kann die Live-Wirkung ohnehin nicht reproduzieren → **headed-Browser-Verifikation nötig** (Dariens visuelles Review ist der richtige Ort). **Fix-Optionen:** globaler „overlay-version"-State in die useMemo-Deps aller Label-Konsumenten, ODER Overlays synchron VOR dem ersten React-Render mergen (i18n-Init await statt useEffect), ODER `i18n.emit`-Trick der ein neues `t` erzwingt. Durchdacht angehen, nicht spekulativ.

## Nächste Stufen = UI-Editoren (visuell + Screenshot-QA-pflichtig)
v1.1–v1.4 bauen sichtbare Oberflächen → Standard-Gate inkl. Playwright-Screenshot-QA + Bilder ansehen. Jede Stufe: Agent-Bau + Review + i18n ×4 + scoped tsc + eslint + Screenshot-QA + 1 Commit.
