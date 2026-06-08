# Phase (Pilot, dashboard) — Dashboard: Modul-Einstellungen (DashboardSettingsPanel)

> **Modul:** dashboard · **Risiko:** niedrig (gemustert) · **Backend:** nicht nötig (persönlich lokal, tenant mock-first).
> Erster FE-Pilot der dashboard-Lane (nach vertraege). „settings-komplett"-Standard. Die DnD-/Persistenz-Phasen folgen im Backlog.

## Ziel
„Dashboard"-Eintrag im Modul-Einstellungs-Fenster mit **Persönlich** + **Für alle**.

## Muster-Vorlagen
- `components/shared/ModuleSettingsShell` · `modules/settings/module-settings-registry.tsx` (Hot File, additiv) · bestehende `*SettingsPanel` (grep).
- Bestehende Dashboard-Dateien: `modules/dashboard/DashboardPage.tsx`, `modules/dashboard/widgets/*` (viele Widgets da), Store `stores/dashboard.ts`, Hook `api/hooks/useDashboard.ts`.

## Inhalte
- **Persönlich** (`stores/dashboardPrefs.ts`): Standard-Dichte, Begrüßung an/aus, Standard-Zeitraum der KPI-Widgets.
- **Für alle** (`stores/dashboardSettings.ts`, mock-first): Team-Default-Layout (welche Widgets neue User sehen), erlaubte Widgets je Lizenz/Modul, KPI-Quellen-Defaults. Mock-first → `backend-handover-luke.md` (Layout-Persistenz + Alert-Queries = dein Backend).

## Schritte
1. `marathon/luke-fe`. `stores/dashboardPrefs.ts` + `DashboardSettingsPanel.tsx` (`modules/dashboard/settings/`) via `ModuleSettingsShell`.
2. In `module-settings-registry.tsx` registrieren (`navMatch: '/'` bzw. Dashboard-Route, Icon).
3. Mind. eine persönliche Pref real anwenden (z.B. Dichte/Zeitraum greift in den Widgets).
4. **i18n ×4** `dashboard.settings.*`.
5. Verifizieren, Review-Faden, commit, push.

## Definition-of-Done
- [ ] „Dashboard" im Modul-Einstellungs-Fenster, beide Bereiche, Lock korrekt.
- [ ] Eine persönliche Pref greift real in den Widgets.
- [ ] i18n ×4, scoped tsc grün (`tsconfig.dashboard-settings.json`), QA grün, Screenshots angesehen.
- [ ] Review-Faden in `reviews/dashboard.md`.

## QA-Hinweis
`scripts/qa-dashboard-settings.mjs`: Modul-Einstellungen → „Dashboard" → beide Bereiche + Sektionen, 0 Roh-Keys/pageErrors. Pref setzen → reload → greift. Screenshots.
