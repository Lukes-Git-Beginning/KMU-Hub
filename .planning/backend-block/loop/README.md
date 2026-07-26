# Backend-Nachtloop

Autonome Bau-Schleife fuer die Backend-Aufhol-Strecke. Laeuft in einer eigenen Shell, arbeitet
`BACKLOG.yml` Unit fuer Unit ab, ein Commit pro Unit, und kann eine Nacht oder laenger durchlaufen.

## Warum das ueber Nacht haelt

Jede Iteration ist ein **eigener `claude -p`-Prozess mit frischem Kontext**. Der Zustand liegt in
`BACKLOG.yml`, `JOURNAL.md` und der Git-Historie — nicht im Kontextfenster. Deshalb gibt es keine
Auto-Compact-Degradation ueber acht Stunden, und der Lauf ist beliebig verlaengerbar.

## Starten

```powershell
# Immer zuerst: Trockenlauf unter Aufsicht
powershell -ExecutionPolicy Bypass -File .planning\backend-block\loop\run-loop.ps1 -MaxIterations 2

# Nachtlauf bis 07:30
powershell -ExecutionPolicy Bypass -File .planning\backend-block\loop\run-loop.ps1 -UntilTime "07:30"
```

Der Treiber bricht vor dem ersten Start ab, wenn der Guard-Test rot ist oder der Branch nicht
`backend-loop` heisst.

## Stoppen

```powershell
New-Item -ItemType File .planning\backend-block\loop\STOP
```

Der Loop beendet nach der laufenden Iteration. Strg+C bricht sofort ab (die laufende Iteration
verliert dann ihren unfertigen Arbeitsstand — Commits bleiben).

## Sicherheitsmodell

Der Loop arbeitet auf `backend-loop` und pusht hoechstens diesen Branch. `hooks/loop-guard.sh` blockt
als PreToolUse-Hook **hart** (exit 2): Push nach main, Merge, `reset --hard`, Force-Push, `gh pr merge`,
Workflow-Dispatch, Production-SSH, `.env.production`, prod-Compose, `deploy.sh`.

Das ist bewusst ein Hook und kein Prompt-Text: dokumentierte Faelle in diesem Repo zeigen, dass
Agenten die Textanweisung "nicht pushen" ignorieren.

Der offene PR bleibt **Draft**. CI laeuft darauf (CI triggert auf PRs gegen `main`), CD nicht
(CD haengt an `workflow_run` auf `main`). Gemergt wird von Hand.

## Morgens

```bash
git log --oneline main..backend-loop      # ein Commit pro Unit
gh pr checks                              # CI-Status am Draft-PR
```

Dann `JOURNAL.md` von unten lesen: `blocked`-Units mit Grund und die `offen:`-Zeilen — dort steht,
was der Loop nicht selbst verifizieren konnte.

Stichprobe gegen die typischen Fehlklassen lohnt sich trotz Verify-Vorspann:

```bash
git diff main..backend-loop -- backend/internal/gateway/ | grep -nE '^\+.*(Service\{|localSvc|\.svc\.)'   # gRPC-Layer umgangen?
git diff main..backend-loop | grep -nE '^\+.*(Unimplemented|TODO)'                                        # Stub durchgerutscht?
git diff --stat main..backend-loop -- '*.proto' '*.pb.go'                                                 # Proto ohne Regen?
```

## Dateien

| Datei | Rolle |
|---|---|
| `run-loop.ps1` | Treiber. Vorflug-Checks, Deadline, STOP-Sentinel, Rate-Limit-Backoff |
| `ITERATION.md` | Der konstante Prompt jeder Iteration |
| `BACKLOG.yml` | Die Queue. Eine Unit = eine Iteration = ein Commit |
| `JOURNAL.md` | Append-only Protokoll |
| `GATE-COMMANDS.md` | Verifizierte Gate-Kommandos (Build, DB, RLS-Smoke, Proto) |
| `loop-settings.json` | `--settings`-Datei, aktiviert den Guard nur waehrend Loop-Laeufen |
| `hooks/loop-guard.sh` | Die harte Grenze |
| `hooks/test-loop-guard.sh` | Regressionstest, muss vor jedem Lauf gruen sein |
| `logs/` | `--output-format json` je Iteration (nicht versioniert) |

## Grenzen

Der Loop macht **nicht**: Phase 1 (RBAC-Fundament, seriell und judgment-lastig — Luke in einer
Opus-Session), Phase 4 (Branchen-BE, Post-Launch), Merge nach main, Deploy, neue
`config.RequireX`-Assertionen, Scharfschalten neuer `modules.*`-Flags.

Kann er eine Unit nicht ehrlich gruen bekommen, markiert er sie `blocked` mit Grund und nimmt die
naechste. Ein ehrliches `blocked` ist mehr wert als ein gruener Commit ueber einem Stub.
