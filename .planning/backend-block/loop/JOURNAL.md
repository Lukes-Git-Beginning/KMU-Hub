# Backend-Nachtloop — Journal

Append-only. Jede Iteration haengt genau einen Block an. Von unten lesen.

Format:

```markdown
## Iteration <n> — <unit-id> — <done|blocked> — <YYYY-MM-DD HH:MM>
- commit: <sha oder ->
- gebaut: <ein bis drei Zeilen, was real existiert>
- gate: build ok | vet ok | lint ok | test ok | migration ok | rls-smoke ok|n.a.
- verify vorgaenger: <sauber | Befund + angelegte Fix-Unit>
- offen: <was Luke morgens pruefen muss>
```

---

## Iteration 0 — Harness aufgesetzt — 2026-07-26

- commit: -
- gebaut: Loop-Harness (Guard-Hook + Regressionstest, ITERATION.md, BACKLOG.yml mit
  22 Phase-3-Units, run-loop.ps1, GATE-COMMANDS.md).
- gate: Guard-Regressionstest 35/35 gruen. Lokale DB auf Migrationskopf 243
  (= Repo-Kopf). RLS-Smoke gegen `contacts` verifiziert: eigener Tenant 12 Zeilen,
  fremder Tenant 0.
- verify vorgaenger: n.a. (erster Eintrag)
- offen: Trockenlauf ueber zwei Iterationen unter Aufsicht, bevor ein Nachtlauf startet.
