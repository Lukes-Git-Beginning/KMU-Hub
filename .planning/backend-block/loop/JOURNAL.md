# Backend-Nachtloop — Journal Lauf 8

Append-only. Ein Eintrag je Iteration, **ans Dateiende**, nie einsortieren. Form und Pflichtzeilen
stehen in `ITERATION.md` Schritt 6.

Frühere Läufe liegen vollständig im Archiv:
`archive/lauf-1-2/` (58 Units) · `archive/lauf-3/` (61) · `archive/lauf-4/` (54) ·
`archive/lauf-5/` (41) · `archive/lauf-6/` (46) · `archive/lauf-7/` (71, inkl. `logs/`).

---

## Laufkontext

- **Ausgangspunkt:** `backend-loop` auf `origin/main` gemergt (nicht rebased). `main` = `c6d1c972`
  (PR #20, Lauf 7). Produktion: Migrationskopf **309 clean**, 30 Container healthy, `/health` mit
  23 Services.
- **Migrationen:** Repo-Kopf = lokaler Kopf = Produktionskopf = **309**. Nächste freie **310** —
  aber immer zur Laufzeit ermitteln.
- **Lokale DB:** `docker-postgres-1` healthy, Rolle `kmuhub_app` mit Passwort `app_dev` verifiziert.
  `DATABASE_URL` ist damit kein Alibi.
- **Backlog:** `BACKLOG.yml`, Blöcke A–F. Null `blocked`-Units zum Laufbeginn — die neun aus Lauf 7
  sind am 2026-08-10 entschieden worden (zwei wurden Lauf-8-Units, vier geparkt, zwei gestrichen,
  siehe `BACKLOG-PARKED.yml`).
- **Fenster:** 15:00–19:30 und 23:00–09:00, über das einmalige Pausenfenster des Treibers als **ein**
  Lauf gefahren (`-UntilTime "09:00" -PauseFrom "19:30" -PauseTo "23:00"`). Ein Prozess, ein Push,
  ein CI-Lauf.
- **Workflow-Zustand beim Start:** `Claude PR Review` `disabled_manually`, `Security Review` vor dem
  Anlegen des Draft-PRs disabled (beide haben kein Draft-Gate und würden beim `opened`-Event zünden).

### Neu in diesem Lauf

- **`coverage:` ist Pflichtzeile** im Journal-Eintrag, mit `coverage_start:` je Unit als festem
  Bezugswert. In Lauf 7 haben nur 8 von 71 Iterationen eine Zahl notiert, obwohl Coverage das
  erklärte Laufziel war — der Endstand musste aus dem CI-Artefakt rekonstruiert werden. Die
  Messmethode ist in `GATE-COMMANDS.md` unter „Coverage messen" festgeschrieben.
- **`mutations-probe:` steht jetzt ebenfalls im Template**, nicht mehr nur im Backlog-Kopf. Sie kam
  in Lauf 7 71/71 und hat zwölf reale Produktionsbugs zutage gefördert; sie bleibt unverändert
  Pflicht.
- **Der Treiber warnt bei Drift.** Stimmt die Nummer der neuen Überschrift nicht mit seiner
  Iterationszählung überein oder fehlt der gelieferte Zeitstempel, schreibt er eine gelbe
  `DRIFT:`-Zeile ins `run.log`. Der Lauf bricht deswegen nicht ab.
- **Zeitstempel sind Pflichtwerte**, keine Formatvorschläge: exakt die Zeichenkette aus dem
  Laufkontext-Block. „(Lauf 8)" oder „(siehe Commit-Zeit)" ist ein Fehler.
- **Der Treiber liest `ITERATION.md` jetzt als UTF-8.** Bis Lauf 7 kam jeder Gedankenstrich als
  `â€”` beim Modell an (PS 5.1 liest UTF-8 ohne BOM sonst als ANSI).

---
