# GSD: Execute Phase

Du fuehrst den **naechsten offenen Plan** der aktuellen Phase aus.

## Deine Aufgabe

1. **Finde den naechsten Plan:**
   - Lies `.planning/STATE.md` fuer aktuelle Position
   - Finde das naechste PLAN-File ohne zugehoeriges SUMMARY-File in `.planning/phases/{PHASE_NAME}/`
   - Pruefe `depends_on` — alle Abhaengigkeiten muessen SUMMARY-Files haben

2. **Lies den Plan** und verstehe:
   - `<objective>` — Was ist das Ziel?
   - `<tasks>` — Welche Tasks muessen ausgefuehrt werden?
   - `must_haves` — Was muss am Ende wahr sein?
   - `<verification>` — Wie wird Erfolg geprueft?

3. **Fuehre die Tasks aus:**
   - Arbeite Task fuer Task durch
   - Committe nach jedem logischen Block (Conventional Commits)
   - Halte dich an die Architektur-Regeln aus CLAUDE.md
   - Nutze bestehende Patterns und wiederverwendbare Komponenten

4. **Verifiziere:**
   - Pruefe alle `must_haves.truths`
   - Pruefe alle `must_haves.artifacts` (Datei existiert, `contains` String vorhanden)
   - Pruefe alle `must_haves.key_links`
   - Fuehre `<success_criteria>` Kommandos aus

5. **Erstelle das SUMMARY-File:**

```markdown
---
phase: {phase-slug}
plan: {NN}
subsystem: {subsysteme}
tags: [{tags}]

requires:
  - phase: {phase-slug}
    plan: {vorheriger plan}
    provides: {was der vorherige plan bereitgestellt hat}
provides:
  - {was dieser plan bereitstellt}
affects: [{zukuenftige phasen}]

tech-stack:
  added: [{neue dependencies}]
  patterns: [{neue patterns}]

key-files:
  created:
    - {dateipfad}
  modified:
    - {dateipfad}

key-decisions:
  - "{entscheidung mit begruendung}"

patterns-established:
  - "{pattern das in zukunft wiederverwendet werden kann}"

requirements-completed: [{REQ-IDs}]

duration: {N}min
completed: {YYYY-MM-DD}
---

# Phase {N} Plan {NN}: {Titel} Summary

**{Einzeiler was gemacht wurde}**

## Performance
- **Duration:** {N} min
- **Tasks:** {N}
- **Files created:** {N}
- **Files modified:** {N}

## Accomplishments
- {Was wurde erreicht, mit technischen Details}

## Task Commits
1. **Task 1: {Name}** - `{commit-hash}` ({type})

## Decisions Made
- {Entscheidungen waehrend der Ausfuehrung}

## Deviations from Plan
{Abweichungen mit Begruendung, oder "None"}

## Issues Encountered
{Probleme, oder "None"}

## Self-Check: {PASSED/FAILED}
{Verifikations-Ergebnis}

---
*Phase: {phase-slug}*
*Completed: {YYYY-MM-DD}*
```

6. **Update STATE.md:**
   - Plan-Counter erhoehen
   - Status aktualisieren
   - Velocity-Tabelle updaten
   - Neue Decisions in Accumulated Context eintragen
   - Session Continuity aktualisieren

## Wichtig

- **Keine AI-Attribution** in Commits
- **Conventional Commits:** feat:, fix:, docs:, refactor:, test:, chore:
- **Structured Logging:** slog, kein fmt.Println / console.log
- **Thick Services, Thin Handlers** — Business-Logik in Services
- Config `.planning/config.json`: mode=yolo bedeutet autonome Ausfuehrung ohne Rueckfragen (ausser bei Unklarheiten)
