# Start-Prompt — Nico-Review-Übergabepaket schnüren

> **Für ein frisches Terminal** (Review-Klon `C:\Users\darie\Documents\KMU-Hub-review`, branch main).
> Erstellt 2026-06-19 nach Batch 3. Ziel: Darien kann Nico EINEN Text schicken, den sie ihrem
> VS-Code-Claude gibt, damit die ab morgen die fertigen Module reviewt (~2–3 Tage/Modul).
> **Reine Doku-Arbeit unter `.planning/` — kein Produktcode, keine i18n ändern.**

## Erst lesen (Ist-Stand, nichts annehmen)
1. `git checkout main && git pull --ff-only origin main`
2. `.planning/parallel-batch/qa-combined.md` (automatisierung + profil, frisch)
3. `.planning/parallel-batch/qa-profil.md`
4. `.planning/archiv/parallel-batch/qa-team.md`, `qa-helpdesk.md`, `qa-dashboard.md`, `qa-vertraege.md`
5. `.planning/MASTER-TRACKER.md` (Pipeline + „Neue projektweite Standards") und `CLAUDE.md` („Modul-Arbeit: Build-+-Verify-Standard"). `.planning/nico-block/` RUNBOOK/WORKFLOW als Stil-Vorlage.

## 6 review-reife Module (alle FE-mock-first; echtes Backend = Lukes Lane, NICHT Teil des Reviews)
team · helpdesk · dashboard · vertraege · automatisierung · profil

## Liefern
1. **`.planning/nico-block/REVIEW-ONBOARDING.md`** — kopierfertiger Text für Nicos VS-Code-Claude. Self-contained:
   - Rolle: FE/UX-Reviewer (kein Backend).
   - Setup: eigener Klon auf aktuellem main, `npm install`, `npm run dev` auf eigenem Port, durchklicken + Screenshots ansehen.
   - **Wogegen prüfen (Demo-Tiefe-Standards):** echte Detail-Ansicht beim Zeilenklick · GANZE Zeile klickbar · Detail = zentriertes `shared/DetailModal` mit sticky/immer-sichtbarem Close · keine toten Buttons/Toast-Stubs · keine leeren Screens · i18n ×4 ohne Raw-Keys/`{{}}` · EN-Umschalten sauber · projektweite UX (Sortierung Feld+Richtung, Zurück-Buttons, Modul-Einstellungen personal+tenant).
   - **Befund-Format:** Modul · Schweregrad (P0 blockt / P1 sollte / P2 nice) · was · wo (Datei/Screen) · Repro.
   - **Klarstellen:** fehlende echte Backend-Anbindung zählt NICHT als Mangel (mock-first).
   - Reihenfolge der 6 Module + Tempo ~2–3 Tage/Modul. Befunde in `.planning/reviews/<modul>.md` eintragen + Darien melden.
2. **`.planning/reviews/<modul>.md`** je Modul (6 Stück) — knappe Definition-of-Done-Checkliste aus der jeweiligen `qa-*.md` (Häkchen-Liste der gebauten Punkte + leerer „Befunde"-Block).
3. Ein sauberer Commit (Conventional, **keine AI-Attribution**) + `git pull --rebase` + push. Danach Darien: „Nico-Paket fertig — Onboarding-Text in `.planning/nico-block/REVIEW-ONBOARDING.md`".

## Kontext-Stand bei Erstellung
main=`59e6fc22`. Batch 3 (automatisierung A-1…A-5 + profil P-1…P-5) gemergt + review-reif. Insgesamt 6 Module warten auf Nicos Review; keins formal übergeben.
