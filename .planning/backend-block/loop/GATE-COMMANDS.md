# Gate-Kommandos — verifiziert 2026-07-26

Exakte, auf dieser Maschine getestete Kommandos. Nicht abwandeln, nicht raten. Alle aus dem **Repo-Root**,
sofern nicht anders angegeben.

## Voraussetzung: PATH

```bash
export PATH="$PATH:/c/Program Files/Go/bin:$HOME/go/bin"
```

Ohne das findet die Shell weder `go` noch `golangci-lint` noch `migrate`.

## Statisches Gate

```bash
cd backend
export DATABASE_URL="postgres://kmuhub_app:app_dev@localhost:5432/kmuhub?sslmode=disable"
go build -p 2 ./internal/<svc>/... ./internal/gateway/... ./cmd/<svc>/... ./cmd/gateway/...
go vet ./internal/<svc>/... ./internal/gateway/...
golangci-lint run --config .golangci.yml ./internal/<svc>/... ./internal/gateway/...
go test -count=1 ./internal/<svc>/...
go test -count=1 ./internal/gateway/   # PFLICHT sobald eine Route dazukam
```

### Warum `DATABASE_URL` zwingend dazugehoert

Ohne die Variable ruft `testutil.SkipIfNoDB` in **jedem** Integrationstest `t.Skip` auf. Der Lauf meldet
dann `ok` fuer Pakete, deren DB-Tests gar nicht liefen — gruen als Aussage darueber, dass nichts geprueft
wurde.

Real passiert (Nachtlauf 1, CI-Lauf 30258205043): `TestIntegrationWrites_LandInCallerTenant` und der
Bestandstest `TestTenantIsolation_Integration_DB` seedeten beide `TenantA` + `slack` in
`integration_configs` und riefen beide `t.Parallel()`. Die Tabelle ist seit Migration 000125
`UNIQUE (platform, tenant_id)`, also starb der Verlierer des Rennens am Duplicate Key. Lokal gruen (beide
uebersprungen), in CI rot. Behoben, indem der Write-Test eigene Tenants erzeugt statt der geteilten
Konstanten — das Muster fuer jeden neuen Test, der eine per-Tenant-eindeutige Zeile seedet.

Die Rolle ist `kmuhub_app`, **nicht** `kmuhub`: der Superuser hat `BYPASSRLS`, unter ihm bestehen
RLS-Isolationstests, ohne irgendetwas zu beweisen. Passwort einmalig setzen, falls die Rolle es lokal noch
nicht hat (dieselbe Konvention wie in `ci.yml`):

```bash
docker exec docker-postgres-1 psql -U kmuhub -d kmuhub -c "ALTER ROLE kmuhub_app WITH PASSWORD 'app_dev'"
```

### Warum `./internal/gateway/` zwingend dazugehoert

Dort liegt `TestOpenAPIRouteDrift` (`internal/gateway/openapi_drift_test.go`): er gleicht **jede**
registrierte `/api/v1/*`-Route gegen `api/openapi.yaml` ab und schlaegt fehl, sobald eine Route ohne
Pfad-Eintrag existiert. Stand 2026-07-26: 657 registrierte Routen gegen 711 dokumentierte Pfade.

Das ist real passiert (Trockenlauf, Iteration 2): der neue Endpoint `POST /einkauf/pos/{id}/cancel`
war gebaut und getestet, aber nicht in der Spec — `go test ./internal/einkauf/...` war gruen, CI rot.
Gezielt pruefen:

```bash
go test ./internal/gateway/ -run TestOpenAPIRouteDrift
```

Gibt es einen begruendeten Fall, in dem eine Route bewusst undokumentiert bleibt, traegt man sie in
`apiV1UndocumentedAllowlist` ein — mit Begruendung, nicht um den Test still zu machen.

`go build ./...` **nicht** verwenden — laeuft auf dieser Maschine in einen OOM. Immer `-p 2` und gezielt
auf die betroffenen Pakete.

Gate-Kommandos **nie durch eine Pipe** laufen lassen (`| head`, `| tail`): der Exit-Code ist dann der der
Pipe und immer 0 — rote Gates werden unsichtbar. Ausgabe bei Bedarf in eine Datei umleiten und die lesen.

## Coverage messen — die verbindliche Definition

Zwei Zahlen, zwei Zwecke. Sie zu verwechseln ist real passiert: Lauf 6 nannte **26,0 %** für
`internal/server`, Lauf 7 maß **27,6 %** für dasselbe Paket am selben Tag. Beide Zahlen waren
richtig gemessen — nur eben zwei verschiedene Dinge.

### 1. Innerhalb einer Iteration: paket-eigen

Kein Extraschritt, das bestehende Gate-Kommando bekommt nur ein Flag:

```bash
go test -count=1 -coverprofile=/tmp/cov.out ./internal/<pkg>/
go tool cover -func=/tmp/cov.out | tail -1
```

**Genau ein Paket, ohne `...`.** `./internal/server/...` würde `internal/server/response`
mitzählen und die Zahl gegenüber dem Bezugswert verschieben. Der Bezugswert steht als
`coverage_start:` an jeder Unit im `BACKLOG.yml` und bleibt über den ganzen Lauf derselbe — so
entsteht eine monotone Kurve statt einer Kette unverbundener Deltas.

`go tool cover` ist ein Report, kein Gate: die Pipe auf `tail` ist hier ausdrücklich erlaubt. Für
`go test` und `golangci-lint` bleibt sie verboten (der Exit-Code wäre der der Pipe, siehe oben).

### 2. Über Läufe hinweg: nur das CI-Artefakt

```bash
gh run download <run-id> -n coverage-report -D "$SCRATCH/cov"
```

Gefiltert wird wie das CI-Gate, aggregiert je Paketverzeichnis:

```bash
grep -v "github.com/kmuhub/kmuhub/proto/" coverage.out | awk 'NR>1{
  split($1,a,":"); f=a[1]; sub(/^github.com\/kmuhub\/kmuhub\//,"",f); split(f,p,"/");
  k=p[1]"/"p[2]; tot[k]+=$2; if($3>0) cov[k]+=$2; T+=$2; if($3>0) C+=$2
} END{ printf "GESAMT %.2f%%\n", C*100/T;
  for(k in tot) printf "%6.1f %7d %s\n", cov[k]*100/tot[k], tot[k]-cov[k], k }' | sort -k2 -rn
```

Spalten: Prozent, ungedeckte Statements, Paket. Für eine Datei-Aufschlüsselung innerhalb eines
Pakets dasselbe awk mit `k=f` statt `k=p[1]"/"p[2]`.

Verifiziertes Referenzergebnis (2026-08-10, Artefakt aus Run `31373430274`, finaler Tree von
Lauf 7): **GESAMT 47.75 %**, `internal/gateway` 34,9 %, `internal/server` 47,7 %,
`internal/produktion` 22,3 %.

### Warum beide Zahlen dieselbe Größe sind

`.github/workflows/ci.yml:111` fährt `go test ./... -coverprofile=coverage.out -covermode=atomic`
**ohne `-coverpkg`**. Ohne dieses Flag misst jedes Paket ausschließlich sich selbst — die
Einträge für `internal/server/*.go` im Artefakt stammen also allein aus dem
`internal/server`-Testbinary. Ein lokales `go test -coverprofile ./internal/server/` misst
genau dasselbe. Die Zahlen sind damit direkt vergleichbar.

**Nicht** vergleichbar ist `go tool cover -func | tail -1` auf einem **repo-weiten** Profil: das
ist der Gesamtwert über alle Pakete, keine Paketzahl. Genau diese Verwechslung erzeugte die
26,0 % aus Lauf 6.

## Lokale Datenbank

Container: `docker-postgres-1` (Custom-Image mit `pg_cron`), Port 5432, DB `kmuhub`.

Hochfahren (falls aus):

```bash
docker compose --env-file deploy/docker/.env -f deploy/docker/docker-compose.yml up -d --build postgres
```

Status:

```bash
docker compose --env-file deploy/docker/.env -f deploy/docker/docker-compose.yml ps postgres
```

### Migration anwenden

Die URLs in `deploy/docker/.env` zeigen auf den Docker-Hostnamen `postgres`. Vom Windows-Host aus muss der
Host auf `localhost` umgeschrieben werden:

```bash
set -a; . deploy/docker/.env; set +a
MIG_LOCAL=$(echo "$MIGRATION_DATABASE_URL" | sed 's|@postgres:|@localhost:|')
migrate -path backend/migrations -database "$MIG_LOCAL" up
migrate -path backend/migrations -database "$MIG_LOCAL" version    # Kopf pruefen
```

Stand 2026-08-10: lokaler Kopf **309** (`dirty = f`) = Repo-Kopf = Produktionskopf. Naechste freie
Nummer **000310** — aber immer zur Laufzeit ermitteln, nie annehmen (Luke migriert parallel):

```bash
ls backend/migrations | grep -E '^[0-9]{6}' | sort | tail -1
```

`MIGRATION_DATABASE_URL` nutzt die Rolle `kmuhub` (Superuser, fuer DDL). `DATABASE_URL` nutzt `kmuhub_app`
(NOSUPERUSER NOBYPASSRLS) — das ist die Rolle, unter der die App laeuft und unter der RLS greift.

### RLS-Smoke

Pflicht, sobald du eine Tabelle, eine Policy oder einen tenant-gescopten SELECT angefasst hast.
`docker exec` braucht **`-i`**, sonst wird stdin nicht durchgereicht und das Skript laeuft stumm ins Leere.

**Fallstrick:** Die Tenant-ID **vor** `SET ROLE kmuhub_app` aufloesen. Danach ist auch die `tenants`-Tabelle
RLS-gefiltert, ein `SELECT id FROM tenants` liefert NULL und der Positiv-Fall wird faelschlich 0.

Vorlage (mit `contacts` als Referenz verifiziert — Tabelle und Tenant-ID an deine Unit anpassen):

```bash
docker exec -i docker-postgres-1 psql -U kmuhub -d kmuhub -tA <<'SQL'
SELECT 'verteilung: ' || tenant_id::text || ' (' || count(*)::text || ')' FROM <deine_tabelle> GROUP BY tenant_id;
SET ROLE kmuhub_app;
SELECT set_config('app.tenant_id', '00000000-0000-0000-0000-000000000001', false) IS NOT NULL;
SELECT 'eigener Tenant -> ' || count(*)::text FROM <deine_tabelle>;
SELECT set_config('app.tenant_id', '00000000-0000-0000-0000-0000000000ff', false) IS NOT NULL;
SELECT 'fremder Tenant -> ' || count(*)::text FROM <deine_tabelle>;
RESET ROLE;
SQL
```

**Bestanden heisst: eigener Tenant > 0 UND fremder Tenant = 0.** Zwei Nullen sind kein Beweis, sondern ein
kaputter Test — dann liegen entweder keine Daten vor oder die Tenant-ID stimmt nicht.

Verifiziertes Referenzergebnis (2026-07-26, `contacts`): eigener Tenant 12, fremder Tenant 0.

Relevante Bausteine im Schema: Setting ist **`app.tenant_id`** (nicht `app.current_tenant`), Helper
`current_tenant_id()`, Policy-Form `tenant_id = current_tenant_id() OR is_system_context()`, Helper zum
Aktivieren `enable_tenant_rls()` / `enable_tenant_rls_via_join()`. 230 Tabellen haben RLS aktiv.

## Proto-Regen

```bash
export PATH="$PATH:/c/Program Files/Go/bin:$HOME/go/bin"
cd backend && make proto-<target>
```

protoc 33.4 und die Makefile-Targets sind verifiziert. `.proto`-Aenderung und regenerierte `.pb.go` gehoeren
in **denselben** Commit.

## Nicht im Loop-Gate

- **e2e / Full-Stack-Bringup** — braucht alle 24 Services; zu schwer fuer eine Iteration.
- **CI** — laeuft am Draft-PR nach dem Push, nicht lokal.
- **Prod-Smoke** — gehoert Luke.
- **Der Race-Detector** — `go test -race` verlangt cgo und damit einen C-Compiler. Auf dieser
  Maschine gibt es keinen (`gcc`, `cc`, `clang` fehlen alle), der Aufruf bricht mit
  `-race requires cgo` ab. **Ein Loop-Gate kann Data Races also grundsaetzlich nicht sehen.**
  CI faehrt `go test ./... -race` (`ci.yml:111`) und sieht sie sehr wohl: Lauf 13 hat zwei
  eingebaut, die lokal in jedem Gate gruen waren und den CI-Lauf 33071992258 rot machten —
  ein ungeschuetzter `append` in einem Mock, der erst nebenlaeufig erreicht wurde, nachdem
  ein No-op-Stub verdrahtet war, und ein `t.Cleanup`, das paketweite Variablen zurueckschrieb,
  waehrend eine Worker-Goroutine sie noch las.

  Solange kein C-Compiler installiert ist, gilt: **Tests, die Goroutinen starten oder
  gemeinsamen Zustand nebenlaeufig anfassen, sind lokal nicht abgenommen.** Wer so einen Test
  schreibt, vermerkt das in `offen:`, damit es beim Merge-Review geprueft wird. Ein
  `mingw-w64`/TDM-GCC auf der Maschine wuerde diese Luecke schliessen.
