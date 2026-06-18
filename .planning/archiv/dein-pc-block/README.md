# Dein-PC-Block — Delegations-Paket Strom D (calendar · dokumente · zeiterfassung)

Specs für den Dein-PC-Strom (remote gefahren). KICKOFF liegt eine Ebene höher: `../dein-pc-KICKOFF.md`.

## Reihenfolge
1. `../dein-pc-KICKOFF.md` — Rolle, Lane, Branch (`marathon/dein-pc`), Regeln.
2. `../collision-map.md` + `../multi-stream-workflow.md` — Pflicht.
3. `../nico-block/RUNBOOK.md` + `../nico-block/WORKFLOW.md` — Build-+-Verify-Prozess (für alle Ströme gleich).
4. **Piloten (eine Phase nach der anderen):**
   - `calendar-p1-views.md` — Tag/Woche/Monat-Views echt machen (Views sind aktuell Platzhalter). **Start hier.**
   - `zeiterfassung-p1-standalone.md` — eigenständiges Modul aus dem Profil-Tab.
   - `dokumente-settings.md` — Modul-Einstellungen (settings-komplett).
5. Danach die Folgephasen aus `../module-phase-plans.md` (Module „→ Strom D").

## Lane-Regeln
Nur calendar · dokumente · zeiterfassung. Branch `marathon/dein-pc`. Hot Files additiv. Pro Phase: bauen → i18n ×4 → scoped tsc → QA → Screenshots ansehen → Review-Faden in `../reviews/<modul>.md` → commit → push.
