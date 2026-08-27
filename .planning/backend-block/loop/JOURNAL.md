# Backend-Nachtloop — Journal Lauf 14

Append-only. Ein Eintrag je Iteration, **ans Dateiende**, nie einsortieren. Form und Pflichtzeilen
stehen in `ITERATION.md` Schritt 6.

Frühere Läufe liegen vollständig im Archiv:
`archive/lauf-1-2/` (58 Units) · `archive/lauf-3/` (61) · `archive/lauf-4/` (54) ·
`archive/lauf-5/` (41) · `archive/lauf-6/` (46) · `archive/lauf-7/` (71) ·
`archive/lauf-8/` (94) · `archive/lauf-9/` (37) · `archive/lauf-10/` (93) ·
`archive/lauf-11/` (131 Einträge, davon 121 done) · `archive/lauf-12/` (99, davon 41 done) ·
`archive/lauf-13/` (120, davon 21 done — abgestürzt, siehe unten). Alle inkl. Laufbilanz.

---

## Laufkontext

- **Ausgangspunkt:** Lauf 13 gemergt als `572cb6b0` und deployt. `main` = `backend-loop`,
  Fast-Forward, **nicht** rebased. CI grün auf `572cb6b0` (Run 33073076446, alle fünf Jobs
  inkl. E2E, 57 Steps).
- **Migrationen:** Repo-Kopf = lokaler DB-Kopf = Produktionskopf = **325**, `schema_migrations`
  clean. Lauf 13 hat keine Migration angelegt. Nächste freie Nummer wäre 326 — aber immer zur
  Laufzeit ermitteln:
  `ls backend/migrations | grep -E "^[0-9]{6}" | sort | tail -1`
- **Coverage-Start:** **71,6 %** gesamt bei Gate 15 % (CI-Artefakt auf `572cb6b0`).
  Lauf 13 hat 69,6 → 71,6 gehoben, in 21 Iterationen.
- **Queue:** 58 Units, die dreizehn verifizierten Lecks am Kopf. Begründung im Kommentarblock
  über den Units in `BACKLOG.yml`.

## Was aus Lauf 13 gelernt wurde — drei Dinge, die dich direkt betreffen

1. **Ein Listeneintrag darf nicht mit einem Backtick beginnen.** YAML verbietet das als erstes
   Zeichen eines Plain-Skalars. Genau daran ist Lauf 13 um 04:37 gestorben, nach 21 von 130
   Iterationen: `BACKLOG.yml` wurde unparsebar, der nächste `--state` des Treibers scheiterte,
   der Lauf war vorbei. Schreib `- >-` und den Text eingerückt darunter. Seit `7d59d43e` blockt
   `hooks/loop-guard.sh` einen Commit, der den Backlog kaputt hinterlässt — du merkst es also
   sofort und reparierst es, statt die Nacht zu beenden.

2. **Ein verifizierter Tenant- oder Auth-Befund gehört an den KOPF von `BACKLOG.yml`, nicht ans
   Ende.** Lauf 13 hat zwei über REST erreichbare Cross-Tenant-Lecks gefunden und regelkonform
   angehängt — Position 44 und 45 von 58. Bei Dateireihenfolge als einziger Priorität heißt das
   „nie". Die Regel steht jetzt in `ITERATION.md`.

3. **`go test -race` läuft auf dieser Maschine nicht** (kein C-Compiler, `-race` braucht cgo).
   Dein Gate kann Data Races grundsätzlich nicht sehen. Lauf 13 hat zwei eingebaut, die lokal in
   jedem Gate grün waren und CI rot machten: ein ungeschützter `append` in einem Mock, der erst
   nebenläufig erreicht wurde, nachdem ein No-op-Stub verdrahtet war, und ein `t.Cleanup`, das
   paketweite Variablen zurückschrieb, während eine Worker-Goroutine sie noch las. Schreibst du
   einen Test, der Goroutinen startet oder gemeinsamen Zustand nebenläufig anfasst, **schreib das
   in `offen:`** — abgenommen ist er erst nach CI.

---
