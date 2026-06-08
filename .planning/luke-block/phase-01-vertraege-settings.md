# Phase 01 (Pilot) — Verträge: Modul-Einstellungen (VertraegeSettingsPanel)

> **Modul:** vertraege · **Risiko:** niedrig (klar gemustert) · **Backend:** nicht nötig — persönliche Prefs lokal, tenant-Settings mock-first.
> **Pilot-Phase** für Strom L: setzt den verbindlichen „settings-komplett"-Standard um, mit exaktem Muster im Repo. Gleicher Aufbau wie kontakte/finanzen/team/work.

## Ziel
Das Modul **Verträge** bekommt — wie jedes andere Modul — einen Eintrag im **Modul-Einstellungs-Fenster** mit zwei Bereichen: **Persönlich** (User-Prefs) + **Für alle** (tenant-weit, nur Modul-Leiter). Damit ist vertraege „settings-komplett".

## Ist-Stand (was schon da ist)
- Modul-UI: `desktop/src/renderer/src/modules/vertraege/` (vollständige FE-Only auf Zustand-Store).
- **Muster-Vorlagen (1:1 als Vorlage nehmen — such nach bestehenden `*SettingsPanel`):**
  - `components/shared/ModuleSettingsShell` — rendert automatisch die Bereich-Header „Persönlich"/„Für alle" + Lock für Nicht-Leiter. Sektionen mit `scope: 'personal' | 'tenant'`.
  - `modules/settings/module-settings-registry.tsx` — hier den Modul-Eintrag registrieren (Gruppe `module`, `navMatch: '/vertraege'`, Icon, `roles`).
  - Bestehende Panels als Blaupause: `modules/work/settings/WorkSettingsPanel` bzw. `FinanceSettingsPanel`/`TeamSettingsPanel` (grep `SettingsPanel`).
  - `stores/<modul>Prefs.ts`-Muster für persönliche Prefs · `hooks/useModuleSettings` (`useIsModuleLead`).

## Schritte
1. `git checkout -B marathon/luke-fe` (falls noch nicht). `git pull` einmal. App: `/vertraege` öffnen.
2. **`stores/vertraegePrefs.ts`** anlegen (analog `workPrefs`): persönliche Prefs, z.B. `defaultView`, `density`, `defaultReminderLeadDays`. Lokal persistiert.
3. **`VertraegeSettingsPanel.tsx`** (in `modules/vertraege/settings/`) via `ModuleSettingsShell`:
   - **Persönlich** (`scope: 'personal'`): Standard-Ansicht/Dichte/Erinnerungs-Vorlaufzeit — REAL an `vertraegePrefs` verdrahtet (greift im Modul).
   - **Für alle** (`scope: 'tenant'`, mock-first über `stores/vertraegeSettings.ts`): Vertragsarten/Kategorien · Standard-Laufzeiten/Kündigungsfristen · Nummernkreis-Format · Erinnerungs-Defaults (tenant). Klar als mock-first markiert → `backend-handover-luke.md` (du baust das BE eh selbst).
4. **Registrieren** in `module-settings-registry.tsx` (ein additiver Eintrag).
5. Persönliche Prefs im Modul anwenden (mind. eine greift sichtbar, z.B. `defaultView` beim Öffnen).
6. **i18n ×4**: alle neuen Labels als `vertraege.settings.*` in `{de,en,fr,it}.json`, einfache Klammern.
7. Verifizieren (unten), Review-Faden schreiben, commit, Branch-Push.

## i18n-Keys
Präfix `vertraege.settings.*` (Sektionen, Feld-Labels, Hinweise) — 4 Sprachen, `{var}`-Interpolation, keine `{{}}`.

## Demo-Handler
Keiner nötig (persönlich = lokal; tenant = mock-first-Store). Falls ein Endpoint später echt wird → `backend-handover-luke.md`.

## Definition-of-Done
- [ ] „Verträge"-Eintrag im Modul-Einstellungs-Fenster, mit Bereichen **Persönlich** + **Für alle** (via `ModuleSettingsShell`).
- [ ] Mind. eine persönliche Pref greift real im Modul (verdrahtet, nicht nur sichtbar).
- [ ] „Für alle" zeigt die tenant-Sektionen (mock-first), Lock für Nicht-Leiter korrekt.
- [ ] i18n ×4 vollständig, keine Roh-Keys.
- [ ] Gescopter Typecheck grün (`tsconfig.vertraege-settings.json` über nur geänderte Dateien), QA-Script grün, Screenshots @1440 + @760 angesehen.
- [ ] Review-Faden in `reviews/vertraege.md` geschrieben.

## QA-Hinweis
`desktop/scripts/qa-vertraege-settings.mjs` (aus bestehendem `qa-*.mjs` ableiten): Modul-Einstellungen öffnen (unten links „Modul-Einstellungen") → „Verträge" → prüfen: beide Bereich-Header sichtbar, alle Sektionen rendern, 0 Roh-Keys, 0 pageErrors. Persönliche Pref setzen → reload → greift. Screenshots beider Bereiche.
