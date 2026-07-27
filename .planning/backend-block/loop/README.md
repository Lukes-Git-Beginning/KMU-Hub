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

## Kosten

**Der Loop selbst kostet kein Geld.** Er laeuft ueber das Claude-Max-Abo (`authMethod: claude.ai`,
kein `ANTHROPIC_API_KEY` gesetzt). Das `total_cost_usd`-Feld in `logs/iter-*.json` ist ein
API-**Aequivalenzwert** — was es ueber die API gekostet haette — keine Rechnung. Verbraucht werden
Abo-Limits (5-Stunden-Fenster und Wochen-Cap), nicht Euro.

**Korrektur 2026-07-27:** eine fruehere Fassung behauptete hier, `Claude PR Review` und
`Security Review` liefen mit einem separat abgerechneten `ANTHROPIC_API_KEY`. Das ist falsch — ein
solches Secret existiert im Repo nicht (`gh secret list` zeigt nur `CLAUDE_CODE_OAUTH_TOKEN`). Beide
nutzen dasselbe Abo wie der Loop: sie kosten **Cap, kein Geld**.

Der einzige echte Geldposten sind damit **Actions-Minuten** (privates Repo). Ein CI-Lauf ist rund
10 Runner-Minuten. Deshalb: **genau ein Push pro Nacht**, ausgefuehrt vom Treiber nach der letzten
Iteration (`run-loop.ps1`, Abschnitt CI-Phase) — nicht vom Agenten, fuer den der Guard `git push`
weiterhin hart blockt. Morgens liegt damit ein echtes CI-Signal vor, nicht nur ein lokales.

Das ersetzt das lokale Gate **nicht**: ein CI-Lauf am Ende der Nacht findet einen Fehler erst, wenn
zwanzig Commits darauf stehen. `DATABASE_URL` (sonst laufen alle DB-Tests ins Skip) und
`go test ./internal/gateway/` bei Routen-Aenderungen bleiben Pflicht.

**Voraussetzung fuer das CI-Signal: ein offener PR auf `backend-loop`.** `ci.yml` triggert nur auf
`push: [main]` und `pull_request: [main]` — ein Push auf einen Branch ohne PR startet nichts. Der
Treiber legt bewusst keinen an (`gh pr create` wuerde die beiden Review-Workflows zuenden, beide ohne
Draft-Gate) und meldet stattdessen im Log, dass kein Lauf startete. Ist der PR gemergt, vor der
naechsten Nacht einen neuen anlegen:

```bash
gh workflow disable "Claude PR Review"; gh workflow disable "Security Review"
gh pr create --base main --head backend-loop --draft --title "Backend-Nachtlauf"
gh workflow enable "Claude PR Review"; gh workflow enable "Security Review"
```

Die reale Grenze eines Dauerlaufs ist der Wochen-Cap des Abos: ein Loop, der ihn leerlaeuft,
blockiert deine eigenen Sessions. Ueber `-MaxIterations` und `-UntilTime` steuerbar.

## Sicherheitsmodell

Der Loop arbeitet auf `backend-loop` und committet ausschliesslich **lokal**.
`hooks/loop-guard.sh` blockt als PreToolUse-Hook **hart** (exit 2): jeden `git push`,
`gh pr create/merge`, Wechsel auf main, `git merge` mit anderem Ziel als `origin/main`,
`git rebase`, `reset --hard`, Workflow-Dispatch, Production-SSH, `.env.production`,
prod-Compose, `deploy.sh`.

Das ist bewusst ein Hook und kein Prompt-Text: dokumentierte Faelle in diesem Repo zeigen, dass
Agenten die Textanweisung "nicht pushen" ignorieren. Verifiziert ist auch, dass der Hook
**innerhalb von `claude -p`** feuert, nicht nur in interaktiven Sessions.

Rebase ist blockiert, weil er die Branch-Historie umschreibt und den naechsten Push
force-pflichtig macht. Der Loop merged `origin/main` herein.

## Morgens

```bash
git log --oneline main..backend-loop      # ein Commit pro Unit, alles noch lokal
git diff --stat main..backend-loop
```

Willst du CI, pushst du selbst — einmal, bewusst:

```bash
git push origin backend-loop
gh pr create --draft --base main --head backend-loop --title "..." --body "..."
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
