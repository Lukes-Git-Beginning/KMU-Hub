# Phase (Pilot) — Formulare: Modul-Einstellungen (FormulareSettingsPanel)

> **Modul:** formulare · **Risiko:** niedrig (gemustert) · **Backend:** nicht nötig (persönlich lokal, tenant mock-first).
> Sicherer FE-Pilot für die formulare-Lane. „settings-komplett"-Standard.

## Ziel
„Formulare"-Eintrag im Modul-Einstellungs-Fenster mit **Persönlich** + **Für alle**.

## Muster-Vorlagen
- `components/shared/ModuleSettingsShell` · `modules/settings/module-settings-registry.tsx` (Hot File, additiv) · bestehende `*SettingsPanel` (grep).
- Bestehende Formulare-Dateien: `modules/formulare/FormularePage.tsx`, Store `stores/formulare.ts`, Hook `api/hooks/useFormulare.ts`.

## Inhalte
- **Persönlich** (`stores/formularePrefs.ts`): Standard-Ansicht (Builder/Liste), Vorschau-Breite.
- **Für alle** (`stores/formulareSettings.ts`, mock-first): DSGVO-Pflichtfeld-Default (Einwilligungs-Checkbox an/aus), Standard-Benachrichtigungs-Empfänger bei Eingang, erlaubte Embed-Domains (CORS-Hinweis), Branding (Logo/Farbe für öffentliche Formulare). Mock-first → `backend-handover-luke.md`.

## Schritte
1. `marathon/nico`. `stores/formularePrefs.ts` + `FormulareSettingsPanel.tsx` (`modules/formulare/settings/`) via `ModuleSettingsShell`.
2. In `module-settings-registry.tsx` registrieren (`navMatch: '/formulare'`).
3. Mind. eine persönliche Pref real anwenden.
4. **i18n ×4** `formulare.settings.*`.
5. Verifizieren, Review-Faden, commit, push.

## Definition-of-Done
- [ ] „Formulare" im Modul-Einstellungs-Fenster, beide Bereiche, Lock korrekt.
- [ ] Eine persönliche Pref greift real.
- [ ] i18n ×4, scoped tsc grün (`tsconfig.formulare-settings.json`), QA grün, Screenshots angesehen.
- [ ] Review-Faden in `reviews/formulare.md`.

## QA-Hinweis
`scripts/qa-formulare-settings.mjs`: Modul-Einstellungen → „Formulare" → beide Bereiche + Sektionen, 0 Roh-Keys/pageErrors. Screenshots.
