# „Anpassungen" (Self-Service-Customization) — Bau-Fortschritt

> SSOT-Konzept: `KONZEPT.md`. Recherche: `IST-A/B/C.md` + `MARKT-A/B/C.md`.
> Stand: 2026-07-21 (Session #25).

## v1 — Fundament-Trio (Overlay-basiert)

| Stufe | Inhalt | Status | Commit |
|---|---|---|---|
| **v1.0** | Overlay-Config-Fundament: Typen + `resolveConfig` (default→vendor→tenant + Provenance) + MSW (labels/value-sets) + RBAC-Key `admin:customization:manage` + Audit + i18n ×4 | ✅ **fertig + verifiziert** | `a43bffcb` |
| **v1.1** | Custom-Fields-Editor (vereinheitlicht, 5 Entitäten, 9 Feldtypen, Progressive Disclosure, Soft-Delete-Schutz) + „Anpassungen"-Hub als 9. Admin-Tab | ✅ **fertig + verifiziert** | `2bdd3407` |
| **v1.2** | Label-Override-Editor (Whitelist ~22 Keys, `addResourceBundle`-Live-Preview, Vendor/Tenant-Herkunft) | ⏳ offen | — |
| **v1.3** | Value-Sets-Editor (zentrale Wertelisten, Referenz-Auflösung in ≥1 Modul als Proof, Soft-Delete) | ⏳ offen | — |
| **v1.4** | „Anpassungen"-Hub (Admin-Fläche + 3 Editoren + Modul-Schnellzugriffe in `ModuleSettingsShell` + Vendor/Tenant-Herkunfts-Banner + Template-Stub) | ⏳ offen | — |

### v1.0 — Verifikations-Nachweis
- **Dateien:** `api/customization-types.ts` (neu), `mocks/data/customization.ts` (neu), `mocks/handlers/customization.ts` (neu), `handlers/index.ts` (+Registrierung), `config/capability-catalog.ts` (+Key), `mocks/data/rbac.ts` (+it_admin-Grant; admin via `catalogCapabilityKeys('admin')`), `mocks/data/audit-events.ts` (+3 Actions), `scripts/i18n-customization-v10.mjs` (32 Keys ×4), `scripts/qa-customization-resolver.mjs` (12-Test-Smoke), `tsconfig.customcheck.json`.
- **Resolver:** `resolveLabelOverrides(locale, base?)` + `resolveValueSet(id, base?)` → Wert + `provenance: 'default'|'vendor'|'tenant'` je Eintrag. tenant > vendor > default. `base=1` = reine Baseline (R-6-Muster).
- **Gates:** scoped tsc **0 Fehler in allen v1.0-Dateien** (8 Rest = Alt-Baseline in automation/crm/finance/hr, transitiv via index.ts) · `eslint` clean (selbst nachgeprüft, korrekter Pfad) · i18n 32 Keys ×4 mit echten fr/it (selbst stichprobe: Anpassungen/Customization/Personnalisation/Personalizzazione) · Smoke 12/12 (tenant>vendor>default, Provenance, base) · admin-Key-Kette selbst verifiziert.
- **Bewusst ausgeklammert:** Custom Fields (eigene BE-Persistenz → v1.1 andocken), Vendor-Session-Detection (`activeConfigLayer()` = immer 'tenant' bis R-5-GDAP-Wiring).

## Nächste Stufen = UI-Editoren (visuell + Screenshot-QA-pflichtig)
v1.1–v1.4 bauen sichtbare Oberflächen → Standard-Gate inkl. Playwright-Screenshot-QA + Bilder ansehen. Jede Stufe: Agent-Bau + Review + i18n ×4 + scoped tsc + eslint + Screenshot-QA + 1 Commit.
