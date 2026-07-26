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
go build -p 2 ./internal/<svc>/... ./internal/gateway/... ./cmd/<svc>/... ./cmd/gateway/...
go vet ./internal/<svc>/... ./internal/gateway/...
golangci-lint run --config .golangci.yml ./internal/<svc>/... ./internal/gateway/...
go test ./internal/<svc>/...
```

`go build ./...` **nicht** verwenden — laeuft auf dieser Maschine in einen OOM. Immer `-p 2` und gezielt
auf die betroffenen Pakete.

Gate-Kommandos **nie durch eine Pipe** laufen lassen (`| head`, `| tail`): der Exit-Code ist dann der der
Pipe und immer 0 — rote Gates werden unsichtbar. Ausgabe bei Bedarf in eine Datei umleiten und die lesen.

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

Stand 2026-07-26: lokaler Kopf **243** = Repo-Kopf. Naechste freie Nummer **000244** — aber immer zur
Laufzeit ermitteln, nie annehmen (Luke migriert parallel):

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
