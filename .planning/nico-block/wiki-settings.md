# Phase (Pilot) — Wiki: Modul-Einstellungen (WikiSettingsPanel)

> **Modul:** wiki · **Risiko:** niedrig (gemustert) · **Backend:** nicht nötig (persönlich lokal, tenant mock-first).
> Sicherer FE-Pilot für die wiki-Lane (die Plan-P1 „Backend-Swap" ist Luke-gated → erst nach dessen Endpoints; bis dahin baut Nico FE-Phasen). Setzt den „settings-komplett"-Standard um.

## Ziel
„Wiki"-Eintrag im Modul-Einstellungs-Fenster mit **Persönlich** + **Für alle** (via `ModuleSettingsShell`).

## Muster-Vorlagen
- `components/shared/ModuleSettingsShell` (Sektionen mit `scope: 'personal'|'tenant'`, Lock für Nicht-Leiter).
- `modules/settings/module-settings-registry.tsx` (Eintrag registrieren — Hot File, additiv).
- Bestehende Panels als Blaupause: grep `SettingsPanel` (z.B. WorkSettingsPanel/FinanceSettingsPanel).
- Bestehende Wiki-Dateien: `modules/wiki/` (WikiPage, WikiEditor, WikiSidebar, …), Store `stores/wiki.ts`.

## Inhalte
- **Persönlich** (`stores/wikiPrefs.ts`, real verdrahtet): Standard-Ansicht (Baum/Liste), Editor-Breite, Sidebar-Default offen/zu.
- **Für alle** (`stores/wikiSettings.ts`, mock-first): Kategorien-Taxonomie (Name+Farbe), Freigabe-Defaults (intern/öffentlich), Public-Modus-Toggle, Kategorie-RBAC-Hinweis. Klar mock-first → `backend-handover-luke.md`.

## Schritte
1. `marathon/nico` auschecken. `stores/wikiPrefs.ts` + `WikiSettingsPanel.tsx` (in `modules/wiki/settings/`) via `ModuleSettingsShell`.
2. In `module-settings-registry.tsx` registrieren (`navMatch: '/wiki'`, Icon, `roles`).
3. Mind. eine persönliche Pref real anwenden (z.B. Standard-Ansicht beim Öffnen).
4. **i18n ×4** `wiki.settings.*`, einfache Klammern.
5. Verifizieren, Review-Faden, commit, push.

## Definition-of-Done
- [ ] „Wiki" im Modul-Einstellungs-Fenster mit beiden Bereichen; Lock für Nicht-Leiter korrekt.
- [ ] Eine persönliche Pref greift real im Modul.
- [ ] i18n ×4, keine Roh-Keys. Scoped tsc grün (`tsconfig.wiki-settings.json`), QA grün, Screenshots @1440+760 angesehen.
- [ ] Review-Faden in `reviews/wiki.md`.

## QA-Hinweis
`scripts/qa-wiki-settings.mjs`: Modul-Einstellungen → „Wiki" → beide Bereich-Header + alle Sektionen, 0 Roh-Keys/pageErrors. Pref setzen → reload → greift. Screenshots beider Bereiche.
