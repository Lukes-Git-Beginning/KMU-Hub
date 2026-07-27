# Übergabe nach Nachtlauf 1 — Stand 2026-07-27, 03:35

> Wiedereinstiegspunkt für die nächste Session. Alles Nötige steht hier; der Rest ist Detail in
> `loop/JOURNAL.md` (Lauf-Protokoll) und `loop/README.md` (Bedienung).

---

## 1. Wo wir stehen

Der Backend-Nachtloop hat **26.07. 19:37 → 27.07. 03:23** durchgearbeitet, 30 Iterationen, und sich
selbst beendet („Keine offenen Units mehr im Backlog"). Kein Abbruch, kein Leerlauf.

| | |
|---|---|
| Branch | **`backend-loop`**, 62 Commits vor `origin/backend-loop` — **nichts gepusht** |
| `main` | `a0e48636`, identisch mit `origin/main`, vom Loop unberührt |
| Units | 36 erledigt, 2 blockiert (Start waren 22 — ~14 hat der Loop selbst gefunden) |
| Diff | 205 Dateien, +36.937 / −4.889 · 35 Code-Commits + 31 Journal-Commits |
| Migrationen | 000244–000252, lokal alle angewendet · Repo-Kopf = DB-Kopf **252** · Prod steht auf 243 |

**Laufende Umgebung** (kann bleiben, spart Zeit):
- Postgres läuft (`docker-postgres-1`, healthy, Port 5432, Migrationskopf 252).
- Das PowerShell-Fenster des Loops (PID 22152) ist noch offen — nur `-NoExit`, der Lauf ist beendet.
  Kann zu.

## 2. Abnahme — was geprüft ist und was nicht

**Grün, verifiziert auf dem finalen Tree:**
- `go build ./...`, `go vet ./...`, `go test ./...` (Exit 0, null FAIL)
- Beide OpenAPI-Drift-Richtungen (`TestOpenAPIRouteDrift` + der neue `TestOpenAPISpecDrift`)
- Kein gRPC-Layer-Bypass · alle 8 geänderten `.proto` haben passende `.pb.go`
- Alle 20 neuen `RequirePermission`-Paare existieren in `permissions` (gegen die lokale DB geprüft)
- Alle neuen Tabellen: `tenant_id NOT NULL` + RLS

**Rot / offen:**
- **`golangci-lint` Exit 1, 5× staticcheck SA5011.** 2 davon sind **pre-existing auf `main`**
  (`internal/work/task/service_test.go`, per Lint-Lauf auf main verifiziert), ~3 neu in
  `internal/biz/tenant_isolation_open_items_test.go`. ⚠ **CI-Lint war am 26.07. auf PR #14 grün** —
  lokal v2.8.0/Windows gegen CI v2.8/Linux. Diskrepanz ungeklärt, vor einem Merge klären.
- **RLS-Smoke auf `report_share_tokens` nicht aussagekräftig** — Tabelle leer, zwei Nullen sind kein
  Beweis (RLS ist aber nachweislich aktiv).

## 3. Was deine Entscheidung braucht

1. **`p3-berichte-server-pdf`** (blocked) — `chart`/`table`-Blöcke lassen sich mit maroto/v2 nicht
   rendern, ohne eine neue Dependency (go-chart/gonum-plot) aufzunehmen. Vorschlag des Loops:
   im PDF als Datentabelle rendern, `lean:`-Marker mit Upgrade-Trigger „wenn Kunden PDF-Charts als
   Pflicht fordern".
2. **`p3-fe-only-features-scope-decision`** (blocked) — Features, die es nur im Frontend gibt.
   Scope-Frage, keine Mechanik.
3. **Lint-Diskrepanz** (s.o.).
4. **Merge-Strategie** für 66 Commits: am Stück oder in Etappen reviewen.

## 4. Der wichtigste inhaltliche Befund

**Dreimal fehlendes Tenant-Scoping beim Schreiben, in drei Modulen:**

| Commit | Modul | Defekt |
|---|---|---|
| `41bf1080` | auth | `invitations` hatte **gar keine** `tenant_id`; Unique-Index auf `(email)` galt global über alle Mandanten |
| `d78f9176` | hr | Pausen-Einträge: Tenant beim Schreiben nicht gesetzt, Lesen ungescoped |
| `f4be722e` | notification | Integrations-Writes ohne `tenant_id` |

Das ist genau die Klasse aus der Memory-Regel „NULLABLE tenant_id Pre-RLS Audit" — dreimal nicht
gefahren. **Empfehlung für den nächsten Block: ein gezielter Audit über alle Schreibpfade**, statt
darauf zu hoffen, dass es bei diesen dreien bleibt. Bei produktiv erzwungenem RLS erzeugt das
Phantom-404 beim Lesen und Zeilen ohne Mandantenzuordnung beim Schreiben.

Weitere Funde: DATEV-Upload meldete `success=true` ohne jede Übertragung (erst auf 501 gesetzt, dann
echt gebaut, `c7802ef3`) · Berichte-Authoring war FE-Fiktion, Persistenz fehlte serverseitig komplett
(Migr. 245) · Reverse-Drift-Guard gebaut, weil der Bestandstest nur eine Richtung prüfte (`21bf691e`).

## 5. Vorschlag für die nächste Session

Reihenfolge nach Hebel:

1. **Review + Merge** des Nachtlaufs. Erst Lint klären, dann pushen und CI **einmal** laufen lassen
   (`git push origin backend-loop` + Draft-PR), nach grünem CI mergen. Der Loop pusht bewusst nicht —
   Actions-Minuten und die API-Key-abgerechneten Review-Workflows.
2. **Die zwei blockierten Units entscheiden** (§3) — beide sind kleine Entscheidungen, keine Analyse.
3. **Tenant-Write-Audit** als neuen Loop-Block aufsetzen (§4). Das ist der grösste offene Hebel und
   eignet sich gut für den Loop: uniform, klar prüfbar, pro Modul partitionierbar.
4. **Phase 1 (RBAC-Fundament)** in einer Opus-Session — Plan liegt fertig in `PHASE-1-RBAC-PLAN.md`,
   bewusst nicht Loop-Arbeit (seriell, Proto-/RLS-heikel).

## 6. Nächsten Lauf starten

```powershell
# Immer: Trockenlauf zuerst
powershell -ExecutionPolicy Bypass -File .planning\backend-block\loop\run-loop.ps1 -MaxIterations 2

# Nachtlauf
powershell -ExecutionPolicy Bypass -File .planning\backend-block\loop\run-loop.ps1 -UntilTime "07:30"

# Stoppen
New-Item -ItemType File .planning\backend-block\loop\STOP
```

Vor einem neuen Lauf: `BACKLOG.yml` mit neuen Units füllen (die alten stehen auf `done`/`blocked`),
Branch auf `backend-loop`, Postgres oben, Guard-Test grün (macht der Treiber selbst als Vorflug).

## 7. Fallstricke aus der Nacht — nicht nochmal reinlaufen

- **`.ps1` rein ASCII halten.** PS 5.1 liest Skripte ohne BOM als ANSI; ein Em-Dash wird zum Smart
  Quote und beendet einen String → Parse-Fehler Dutzende Zeilen später.
- **`bash` aus PowerShell löst auf WSL auf**, nicht auf Git Bash. Absoluten Pfad nutzen
  (`C:\Program Files\Git\bin\bash.exe`).
- **Rate-Limit-Erkennung nie per nacktem `429`-Grep** — die Ziffern stecken in Kosten-Floats
  (`total_cost_usd: 5.36732429…`). Nur bei `is_error`/Exit≠0 auswerten. (Ist gefixt.)
- **Journal-Uhrzeiten sind geraten** und unbrauchbar — der Agent hat keine Uhr. `logs/run.log` ist die
  Wahrheit.
- **Gate-Kommandos nie durch Pipes** — der Exit-Code ist dann der der Pipe. Explizit prüfen.
- **`git rebase` auf dem Loop-Branch** erzwingt einen Force-Push. Der Loop merged `origin/main`.
- **Grep auf „ENABLE ROW LEVEL SECURITY" übersieht `CALL enable_tenant_rls(...)`** — beides prüfen.
