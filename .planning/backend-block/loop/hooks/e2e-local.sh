#!/usr/bin/env bash
# Lokaler E2E-Stack fuer den Nachtloop - nachgebaut nach dem Job `e2e` in .github/workflows/ci.yml.
#
#   bash .planning/backend-block/loop/hooks/e2e-local.sh                    # volle Suite
#   bash .planning/backend-block/loop/hooks/e2e-local.sh 'TestSchichten'    # go test -run <Regex>
#
# Startet Wegwerf-Postgres/Redis/MinIO in Docker (eigene Ports, die Dev-DB docker-postgres-1 bleibt
# unberuehrt), migriert frisch, baut die Services nativ, startet sie mit der CI-Env und faehrt
# `go test -tags=e2e`. Raeumt in jedem Fall auf (trap). Exit-Code = der von go test.
#
# Alle Services binden auf 127.0.0.1 (die Port-Variablen sind Listen-Adressen, `net.Listen` nimmt sie
# woertlich): keine Windows-Firewall-Abfrage, nichts im LAN offen.
#
# lean: Services laufen wie in CI als DB-Superuser `kmuhub_test`, nicht als `kmuhub_app` (RLS greift
# fuer Superuser nicht). Umstellen, wenn der CI-Job das auch tut.
set -uo pipefail

ROOT=$(cd "$(dirname "$0")/../../../.." && pwd)
BACKEND="$ROOT/backend"
BIN="$BACKEND/.e2e-bin"
LOGDIR="$ROOT/.planning/backend-block/loop/logs/e2e"
RUN_REGEX="${1:-}"
export PATH="$PATH:/c/Program Files/Go/bin:$HOME/go/bin"

# Alle Ports unter 49152: Windows reserviert im dynamischen Bereich wechselnde Bloecke fuer
# Hyper-V/WinNAT (`netsh int ipv4 show excludedportrange protocol=tcp`; am 2026-10-25 lag 50060-50159
# darin, der dialer bekam "bind: forbidden"). Die Bloecke verschieben sich nach einem Neustart.
PG_PORT=45432; REDIS_PORT=46379; MINIO_PORT=45900
PG_URL="postgres://kmuhub_test:kmuhub_test@localhost:$PG_PORT/kmuhub_test?sslmode=disable"
MINIO_IMAGE="ghcr.io/lukes-git-beginning/minio:RELEASE.2025-04-22T22-12-26Z"
CONTAINERS="kmuhub-e2e-postgres kmuhub-e2e-redis kmuhub-e2e-minio"

# name:grpc-port:health-port:health-env  -  Reihenfolge = Startreihenfolge wie in CI, Gateway zuletzt.
# Health-Ports wie in internal/config/config.go, gRPC-Ports = Config-Default minus 9000 (500xx -> 410xx,
# siehe oben). auth heisst HEALTH_PORT, alle anderen <NAME>_HEALTH_PORT.
SERVICES="
auth:41051:9091:HEALTH_PORT
crm:41052:9092:
chat:41053:9093:
work:41055:9095:
document:41057:9097:
dialer:41061:9101:
biz:41058:9098:
notification:41054:9094:
inventar:41070:9110:
einkauf:41071:9111:
schichten:41075:9115:
produktion:41072:9112:
vertraege:41073:9113:
rapporte:41074:9114:
fuhrpark:41076:9116:
vermietung:41077:9117:
wiki:41062:9102:
berichte:41063:9103:
formulare:41064:9104:
helpdesk:41065:9105:
"

log() { echo "[e2e-local $(date +%H:%M:%S)] $*"; }

# Git Bash wandelt Argumente wie `/data` oder `/var/lib/...` in Windows-Pfade um, das zerlegt die
# Container-Pfade. Nur fuer docker abschalten: taskkill braucht die Umwandlung (`//F` -> `/F`).
dk() { MSYS_NO_PATHCONV=1 docker "$@"; }

cleanup() {
    taskkill //F //IM "kmuhub-e2e-*" >/dev/null 2>&1 || true
    dk rm -f $CONTAINERS >/dev/null 2>&1 || true
}

wait_http() {  # url, name, sekunden
    local i
    for ((i = 0; i < $3; i++)); do
        curl -sf "$1" >/dev/null 2>&1 && return 0
        sleep 1
    done
    log "FEHLER: $2 nicht gesund nach $3 s ($1)"
    return 1
}

# Reste eines abgebrochenen Laufs entfernen, bevor Ports geprueft werden.
cleanup
trap cleanup EXIT
mkdir -p "$LOGDIR" "$BIN"
rm -f "$LOGDIR"/*.log

# Belegter Port = fremder Prozess (z. B. Dev-Stack). Der Health-Check wuerde sonst DEN treffen und die
# Tests liefen gegen den falschen Stack.
# Einmal netstat statt einer Verbindungsprobe je Port: ein abgewiesener Connect kostet unter
# Windows rund zwei Sekunden, bei 44 Ports anderthalb Minuten.
# Lauschende Sockets an der Gegenadresse `:0` erkennen, nicht am Status: das deutsche Windows
# schreibt "ABHOEREN" (in CP850), nicht "LISTENING".
listening=$(netstat -an -p tcp | sed 's/\r$//' | awk '$1 == "TCP" && $3 ~ /:0$/ {print $2}')
ports="8080 $PG_PORT $REDIS_PORT $MINIO_PORT $(echo "$SERVICES" | awk -F: 'NF {print $2, $3}')"
for port in $ports; do
    if echo "$listening" | grep -qE "[:.]$port\$"; then
        log "ABBRUCH: Port $port ist belegt (laeuft der Dev-Stack?)."; exit 2
    fi
done

# --- Infrastruktur ----------------------------------------------------------
log "Starte Postgres/Redis/MinIO"
dk run -d --name kmuhub-e2e-postgres -p 127.0.0.1:$PG_PORT:5432 --tmpfs /var/lib/postgresql/data \
    -e POSTGRES_DB=kmuhub_test -e POSTGRES_USER=kmuhub_test -e POSTGRES_PASSWORD=kmuhub_test \
    pgvector/pgvector:pg16 >/dev/null || exit 2
dk run -d --name kmuhub-e2e-redis -p 127.0.0.1:$REDIS_PORT:6379 redis:7-alpine >/dev/null || exit 2
dk run -d --name kmuhub-e2e-minio -p 127.0.0.1:$MINIO_PORT:9000 \
    -e MINIO_ROOT_USER=kmuhub -e MINIO_ROOT_PASSWORD=kmuhub_dev "$MINIO_IMAGE" server /data >/dev/null || exit 2

for ((i = 0; i < 60; i++)); do
    dk exec kmuhub-e2e-postgres pg_isready -U kmuhub_test -d kmuhub_test >/dev/null 2>&1 && break
    sleep 1
done
wait_http "http://127.0.0.1:$MINIO_PORT/minio/health/live" minio 60 || exit 2

log "Migriere frische DB"
migrate -path "$BACKEND/migrations" -database "$PG_URL" up >"$LOGDIR/migrate.log" 2>&1 || {
    log "FEHLER: migrate up (siehe $LOGDIR/migrate.log)"; tail -20 "$LOGDIR/migrate.log"; exit 2; }

# --- Build (nur wenn sich backend/ seit dem letzten Build geaendert hat) ------
cd "$BACKEND" || exit 2
names=$(echo "$SERVICES" | awk -F: 'NF {print $1}')
stamp="$(git rev-parse HEAD)-$(git status --porcelain -- . | sha1sum | cut -c1-12)"
if [ "$(cat "$BIN/.stamp" 2>/dev/null)" != "$stamp" ] || [ "$(ls "$BIN"/kmuhub-e2e-*.exe 2>/dev/null | wc -l)" -lt 21 ]; then
    log "Baue 21 Binaries (GOMAXPROCS=2 -p 2)"
    for n in $names gateway; do
        GOMAXPROCS=2 go build -p 2 -o "$BIN/kmuhub-e2e-$n.exe" "./cmd/$n" >"$LOGDIR/build-$n.log" 2>&1 || {
            log "FEHLER: go build ./cmd/$n"; cat "$LOGDIR/build-$n.log"; exit 2; }
    done
    echo "$stamp" >"$BIN/.stamp"
else
    log "Binaries aktuell, kein Build"
fi

# --- Services ---------------------------------------------------------------
export DATABASE_URL="$PG_URL"
export REDIS_URL="redis://localhost:$REDIS_PORT"
export MINIO_ENDPOINT="localhost:$MINIO_PORT"
export JWT_SECRET="ci-e2e-test-secret-minimum-32-characters"
export GATEWAY_URL="http://localhost:8080"
export GATEWAY_HTTP_PORT="127.0.0.1:8080"
export RATE_LIMIT_RPS=1000
export AUTH_SELF_REGISTRATION_ENABLED=true
for m in INVENTAR EINKAUF SCHICHTEN PRODUKTION VERTRAEGE RAPPORTE FUHRPARK VERMIETUNG WIKI BERICHTE FORMULARE HELPDESK; do
    export "COSMI_MODULE_${m}_ENABLED=true"
done
# Listen-Adressen aller Services vorab setzen: ein Service darf beim Start einen anderen anwaehlen.
while IFS=: read -r name grpc health henv; do
    [ -z "$name" ] && continue
    up=$(echo "$name" | tr '[:lower:]' '[:upper:]')
    export "${up}_GRPC_PORT=127.0.0.1:$grpc"
    export "${up}_GRPC_ADDRESS=localhost:$grpc"
    export "${henv:-${up}_HEALTH_PORT}=127.0.0.1:$health"
done <<<"$SERVICES"

while IFS=: read -r name grpc health henv; do
    [ -z "$name" ] && continue
    "$BIN/kmuhub-e2e-$name.exe" >"$LOGDIR/$name.log" 2>&1 &
    wait_http "http://127.0.0.1:$health/health" "$name" 60 || { tail -30 "$LOGDIR/$name.log"; exit 2; }
done <<<"$SERVICES"
"$BIN/kmuhub-e2e-gateway.exe" >"$LOGDIR/gateway.log" 2>&1 &
wait_http "http://127.0.0.1:8080/health" gateway 60 || { tail -30 "$LOGDIR/gateway.log"; exit 2; }
log "Stack steht (21 Prozesse), starte Tests ${RUN_REGEX:+mit -run '$RUN_REGEX'}"

# --- Tests ------------------------------------------------------------------
args=(-tags=e2e ./test/e2e/... -count=1 -v)
[ -n "$RUN_REGEX" ] && args+=(-run "$RUN_REGEX")
go test "${args[@]}" >"$LOGDIR/go-test.log" 2>&1
rc=$?
grep -E '^(--- FAIL|FAIL|ok|PASS)' "$LOGDIR/go-test.log" | tail -40
log "go test Exit $rc. Volles Log: $LOGDIR/go-test.log, Service-Logs daneben."
exit $rc
