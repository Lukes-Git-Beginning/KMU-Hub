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

## Iteration 1 — p3-einkauf-total-amount — done — 2026-07-26

- commit: e91cdf2a
- gebaut: Repository-Methode `RecomputePOTotal` (SQL-Aggregat SUM(quantity*unit_price)
  in purchase_orders.total_amount, tenant-gescoped) + Aufruf aus AddPOLine/UpdatePOLine/
  DeletePOLine im Service. CreatePO bleibt bei "0" (korrekt, da 0 Zeilen bei Anlage).
  4 neue Unit-Tests (Add/Add-mehrzeilig/Update/Delete) pruefen den total_amount-Wert
  nach der jeweiligen Mutation.
- gate: build ok | vet ok | lint ok (0 issues) | test ok | migration n.a. (keine neue
  Tabelle/Spalte) | rls-smoke n.a. (kein neuer SELECT-Pfad, bestehende Policy greift
  unveraendert)
- verify vorgaenger: n.a. (Iteration 0 war reines Harness-Setup, kein Unit-Commit)
- offen: keins. Naechste Unit ist p3-einkauf-cancel (deps erfuellt).
