# Phase (Pilot, dokumente) — Dokumente: Modul-Einstellungen (DokumenteSettingsPanel)

> **Modul:** dokumente · **Risiko:** niedrig (gemustert) · **Backend:** nicht nötig (persönlich lokal, tenant mock-first).
> Sicherer Einstieg für die dokumente-Lane (das feature-reiche Modul). „settings-komplett"-Standard. Die Plan-P1 (Move/Copy, granulare Rechte, Datei-Kommentare) folgt im Backlog.

## Ziel
„Dokumente"-Eintrag im Modul-Einstellungs-Fenster mit **Persönlich** + **Für alle**.

## Muster-Vorlagen
- `components/shared/ModuleSettingsShell` · `modules/settings/module-settings-registry.tsx` (Hot File, additiv) · bestehende `*SettingsPanel` (grep).
- Bestehende Dokumente-Dateien: `modules/dokumente/DokumentePage.tsx` (+ FileDetailPanel, ShareDialog, VersionHistory, OnlyOfficeEditor …), Hooks `api/hooks/useDocuments.ts` + `useDocumentUpload.ts`.

## Inhalte
- **Persönlich** (`stores/dokumentePrefs.ts`): Standard-Ansicht (Liste/Kacheln), Standard-Sortierung (Feld+Richtung — `shared/SortMenu`-Muster), Dichte.
- **Für alle** (`stores/dokumenteSettings.ts`, mock-first): Speicher-Quota pro Tier (Anzeige), erlaubte Dateitypen, Standard-Freigabe-Rechte (privat/Team), OnlyOffice-Editor an/aus, Aufbewahrung/Papierkorb-Tage. Mock-first → `backend-handover-luke.md`.

## Schritte
1. `marathon/dein-pc`. `stores/dokumentePrefs.ts` + `DokumenteSettingsPanel.tsx` (`modules/dokumente/settings/`) via `ModuleSettingsShell`.
2. In `module-settings-registry.tsx` registrieren (`navMatch: '/dokumente'`, Icon).
3. Mind. eine persönliche Pref real anwenden (z.B. Standard-Ansicht/Sortierung greift in der Dateiliste).
4. **i18n ×4** `dokumente.settings.*`.
5. Verifizieren, Review-Faden, commit, push.

## Definition-of-Done
- [ ] „Dokumente" im Modul-Einstellungs-Fenster, beide Bereiche, Lock korrekt.
- [ ] Eine persönliche Pref greift real in der Dateiliste.
- [ ] i18n ×4, scoped tsc grün (`tsconfig.dokumente-settings.json`), QA grün, Screenshots angesehen.
- [ ] Review-Faden in `reviews/dokumente.md`.

## QA-Hinweis
`scripts/qa-dokumente-settings.mjs`: Modul-Einstellungen → „Dokumente" → beide Bereiche + Sektionen, 0 Roh-Keys/pageErrors. Pref setzen → reload → greift. Screenshots.
