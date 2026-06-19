# QA-Protokoll — berichte (Main-Terminal, Branch `main`, Dev-Port 5173)

> Build-+-Verify pro Punkt: bauen → i18n ×4 (`{var}`/ICU) → MSW-Daten → `npm run build` (echter Exit) → Playwright-Screenshot-QA + Bilder angesehen → Commit+Push. QA-Scripts: `qa-berichte.mjs`, `qa-berichte-schedules.mjs`, `qa-berichte-settings.mjs`, `qa-berichte-i18n.mjs`. Screenshots `.qa-screenshots/berichte/`.

| Punkt | Inhalt | Status | Verify (Screenshots angesehen) |
|---|---|---|---|
| **B-1** | MSW-Vollausbau: 6 Definitionen, run/export(Blob)/schedules-CRUD, stateful. `berichte.ts` exakt an `berichte-types`/Hooks angeglichen | ✅ | 4 Tabs leben: 9 KPIs+18 Sparklines, Erstellen-Dropdown gefüllt (7 Optionen) + Generieren aktiv, 3 Schedules, DATEV 8 Zeilen. 0 Raw-Keys/Errors. Commit `88e7fd5c` |
| **B-2** | Hero-Charts auto-laden + Ticket-Chart eigene Def · KPI-Klick→DetailModal (Mini-Chart+Kennzahlen) · DATEV BWA/SuSa-Toggle (`onClick`-Fix) | ✅ | heroLoaded true, 11 recharts-Surfaces, Drilldown-Modal mit Chart, SuSa-Toggle wechselt (Konto/Saldo, 7 Zeilen). Commit `43a2ee76` |
| **B-3** | Schedules stateful (Toggle/Delete/Create) · „Nächster Lauf"-Spalte (Cron) · Alert-Schwellwert-Feld + Bell · Epoch-Fallback-Fix | ✅ | Nächster Lauf 22.06./01.07./Pausiert korrekt; Create 3→4 + Bell; Toggle 3→2 aktiv; Delete 4→3; „Noch nicht gelaufen". Commit `16437a36` |
| **B-4** | `shared/SortMenu` in ScheduleList · Modul-Settings-Eintrag (personal Format/Zeitraum real angewendet + tenant Formate/Domains) | ✅ | Settings-Panel rendert personal+tenant, „Berichte" preselektiert, Sort Monatliche↔Wöchentlicher. 0 Raw-Keys. Commit `3129283c` |
| **B-5** | 38 defaultValue-Keys ×4 migriert (DE/EN/FR/IT) · Hardcode-Placeholder i18n · EN-Schlusscheck | ✅ | EN-Sweep alle Tabs + Settings sauber (Next Run/Trend (demo)/Default format/Allowed export formats), 0 deutsche UI-Reste, 0 Raw-Keys/Doppelklammern/Errors |

## Out of scope (NICHT gebaut — 🔒 Luke)
No-Code-Query-Builder (P2), echtes DATEV-/BI-Backend (P4), Dashboard-Kachel-DnD. Tenant-Settings demo-stateful.

## Definition of Done — berichte review-reif ✅ (5/5)
4 von 4 Tabs lebendig, 0 Raw-Keys / 0 `{{var}}` / 0 Console-Errors über alle Tabs (DE+EN), jede Phase Commit+Push auf `main`. Untracked Helfer: `add-berichte-settings-i18n.mjs`, `add-berichte-rest-i18n.mjs`. Nico-Review-Checkliste: `.planning/reviews/berichte.md`.
