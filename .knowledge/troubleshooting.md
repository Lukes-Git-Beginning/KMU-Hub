---
tags: [troubleshooting, debug]
updated: 2026-05-09
---
# Troubleshooting & Bekannte Probleme

## Architektur-Fehler (NICHT wiederholen)
Aus Vorgaenger-Projekt (slot_booking_webapp) gelernt:

- **Dual-Write vermeiden** — NUR PostgreSQL, Redis = Cache. Nie JSON+DB parallel
- **Business-Logik in Services** — Nicht in Handlern, nicht in DB-Queries
- **Service erweitern** — Bestehende Services erweitern, KEINE neuen Stores/JSON-Files
- **Migrations via Tool** — `make migrate-create`, nie manuelles SQL
- **Komponenten wiederverwenden** — Nicht kopieren, Component-Library nutzen

## Tailwind + CSP
- Tailwind JIT (Runtime) braucht `unsafe-eval` → inkompatibel mit CSP Nonces
- **Loesung:** Tailwind v4 IMMER pre-compilen (Vite Plugin, nicht Runtime)
- Aktuell korrekt konfiguriert in `electron.vite.config.ts`

## Test-Patterns
- Patch-Pfade muessen dort sein wo importiert wird, nicht wo definiert
- Keine verschachtelten Contexts
- Test-Isolation: Jeder Test raeumt seine Daten auf

## Docker Compose
- **Reihenfolge:** Services haben `depends_on` mit Health-Check-Conditions
- **Health-Check-Timeout:** OnlyOffice braucht bis zu 60s Start-Period
- **Volumes:** `pgdata` und `minio_data` persistent, `docker-compose down -v` loescht alles
- **Rebuild nach Code-Änderung:** `docker-compose build <service> && docker-compose up -d <service>`

## Windows/Dev-Umgebung
- **protoc Pfad:** `C:/Users/Luke/AppData/Local/Microsoft/WinGet/Packages/.../protoc.exe`
- **Go Pfad:** `export PATH="$PATH:/c/Program Files/Go/bin:$HOME/go/bin"`
- **GitHub CLI:** `"C:/Program Files/GitHub CLI/gh.exe"`
- **Shell:** Git Bash (Unix-Syntax, nicht Windows CMD)

## Go-Linker-OOM bei `go build ./...` (Windows, viele cmd-Mains)
- **Symptom:** `runtime: cannot allocate memory` aus `cmd/link` waehrend repo-weitem `go build ./...`. Tritt typischerweise mid-build auf (z.B. beim Bauen von `cmd/chat`), kein Code-Fehler.
- **Ursache:** 24 Microservice-Mains (`backend/cmd/<svc>`) — der Go-Build linkt sie parallel (`GOMAXPROCS`-viele gleichzeitig). Auf Windows mit knappem freien RAM platzt der Linker-Heap.
- **Workaround:** Seriell pro Service bauen, optional mit `-ldflags="-w -s"` (strip debug+symbol-Table fuer kleinere Binaries und weniger Linker-Memory):
  ```bash
  cd backend
  for d in cmd/*/; do
    svc=$(basename "$d")
    go build -ldflags="-w -s" -o "/tmp/kmuhub_build/$svc" "./cmd/$svc" || break
  done
  ```
- **Weitere Optionen:** `go build -p 1 ./...` (komplett serieller Compile/Link) oder `GOMEMLIMIT=4GiB go build ./...`. CI ist nicht betroffen — Linux-Linker hat genug Heap.
- **Verifikation ohne Build:** `go vet ./...` + `go test ./...` belasten den Linker nicht und liefern volle Korrektheits-Aussage.
- Notiert nach Sprint 2 Welle 4A (2026-04-29) — vier Subagents bauten nur `./internal/...`, der repo-weite Build war erst danach an der Reihe. Seriell + ldflags hat alle 24 Services sauber gebaut.

## golangci-lint
- Version 2 erfordert `version: "2"` in `.golangci.yml`
- `goimports` aus Formatters entfernt (CI-Issues)
- Action: golangci-lint v2.8 (action v7)

## Radix Dialog Null-Access Pattern
- Radix Dialog rendert `<DialogContent>` im DOM auch wenn `open={false}`
- Alle Zugriffe auf Dialog-State im Content muessen null-safe sein
- **Pattern:** `showDialog?.property` oder `{showDialog && ...}` im DialogContent
- Betraf: EinkaufPage, FormularePage, ZustandsprotokollDialog (alle gefixt 2026-04-01)

## useMemo Scope-Fehler
- Variablen die INNERHALB von `useMemo()` deklariert werden sind AUSSERHALB nicht verfügbar
- Wenn JSX auf diese Variablen zugreift → `ReferenceError: x is not defined`
- **Fix:** Variable im Return-Objekt des useMemo zurückgeben
- Betraf: CalendarUpcoming (today/dd), MyCalendar (now) (gefixt 2026-04-01)

## Häufige Fehler
- `fmt.Println` / `console.log` statt Structured Logging → slog verwenden
- Hardcoded Secrets → Environment Variables
- CORS Wildcard → Explizite Allowlist
- Deployment ohne Backup → IMMER zuerst Backup

## i18n Migration — Lessons Learned
- **Agent Token-Limits:** Massen-Instrumentierung (200+ Dateien) ueberschreitet Kontext — Waves von 30-50 Dateien, separate Commits
- **JSON-Extraktion trennen:** Erst Schluessel in additions/*.json extrahieren, dann useTranslation/t()-Calls einfuegen — reduziert Merge-Konflikte
- **`keySeparator: false` ist kritisch:** Ohne diese Option wuerde `"crm.contacts.title"` als nested Object geparst — immer explizit setzen
- **Marken-Namen nicht uebersetzen:** "Cosmi", "Zentria" nie in `t()` wrappen
- **ICU-Syntax:** i18next-icu verwendet `{count, plural, one {…} other {…}}` — nicht react-intl's `=1 {…}` Notation
- **ICU-Plural-Klammer-Bug (2026-04-18):** 18 Strings hatten eine fehlende schliessende `}` der aeusseren Plural-Klammer — Renderfehler traten nur bei nicht-trivialen Counts auf. Fix + Regressions-Test `plural.test.ts` gatet jeden neuen Plural-String. Siehe [[i18n]].

## Git-Hook / Staging-Regeln
- **`.env*`-Dateien:** Der Pre-Commit-Hook blockt jeden `git add`-Befehl, in dem `.env` im Pfad auftaucht (auch `.env.example`). Konsequenz: Env-Var-Dokumentation im Code-Kommentar oder README ablegen, nicht in `.env.example` eintragen. Wer das trotzdem braucht, muss die Datei manuell nachpflegen.
- **Conventional Commits + keine AI-Attribution:** `Co-Authored-By`, "Generated by" etc. werden projektweit nicht committed.

## Production-Redeploy Lessons (2026-04-19/20)

Aus dem ersten Full-Redeploy des Hetzner-Prod-Servers von `fa17fc3` auf `980eba3` — Details in MEMORY `project_server_redeploy_20260419.md`. Alle folgenden Symptome koennen bei einem Fresh-Deploy auf einen laenger nicht angefassten Server erneut auftreten.

### Docker-Compose-File hat hardcoded Credentials
- **Symptom:** `docker compose run --rm migrate` scheitert mit `error: pq: password authentication failed for user "kmuhub"`, obwohl `.env.production` das korrekte Passwort enthaelt und das Postgres-Volume per `ALTER USER` synchronisiert wurde.
- **Ursache:** `deploy/docker/docker-compose.yml` hatte bis 2026-04-19 in 17 Service-Definitionen und 1× `POSTGRES_PASSWORD` hardcoded `kmuhub_dev`. Service-env ueberschreibt env-file.
- **Fix:** Alle Vorkommen durch `${DATABASE_URL}` / `${POSTGRES_PASSWORD}` ersetzen. Auf Server via skip-worktree erledigt, muss in Sprint 2 auf `main`.

### Docker-Healthcheck faellt, obwohl `/health` 200 OK liefert
- **Symptom:** `docker compose ps` zeigt Services als "unhealthy", aber `wget -qO- http://localhost:9091/health` aus dem Container liefert `{"status":"healthy", ...}`.
- **Ursache:** Healthcheck nutzt `wget --no-verbose --spider` (= HTTP HEAD). Go-Services registrieren `/health` nur fuer GET → 404 auf HEAD. Docker interpretiert das als unhealthy.
- **Fix (temporaer, server-side):** `--spider` durch `-q -O /dev/null` (GET) ersetzen. Nachhaltig: Backend-Router auch fuer HEAD registrieren.

### `formulare`-Service ist "unhealthy", andere laufen
- **Symptom:** Alle 14 anderen Services healthy, nur `formulare` nicht. Gateway startet nicht, weil `depends_on: formulare {service_healthy}`.
- **Ursache:** `formulare` registriert `/healthz` statt `/health` (inkonsistent mit Rest). Healthcheck pingt `/health` → 404.
- **Fix:** Healthcheck im compose auf `http://localhost:9104/healthz` aendern, ODER Backend-Endpoint zusaetzlich als `/health` registrieren.

### `git pull` schlaegt fehl bei `assume-unchanged`-File
- **Symptom:** `error: Your local changes to the following files would be overwritten by merge: deploy/docker/livekit.yaml. Aborting`.
- **Ursache:** `--assume-unchanged` ist eine Git-Optimierung fuer "ich lies die Datei nicht", **nicht** fuer "darf lokal abweichen". Bei `pull` mit incoming changes bricht Git ab.
- **Fix:** `git update-index --skip-worktree <file>` ist die richtige Option. Workflow bei PR, der die Datei aendert:
  ```bash
  git update-index --no-skip-worktree deploy/docker/livekit.yaml
  git checkout -- deploy/docker/livekit.yaml  # zum Repo-State zurueck
  git pull origin main                        # faehrt als Fast-Forward durch
  # neuen Patch anwenden, danach:
  git update-index --skip-worktree deploy/docker/livekit.yaml
  ```

### `DATABASE_URL` scheitert mit `invalid integer value "..."` for port
- **Symptom:** psql-Connect mit DATABASE_URL aus `.env.production` fails mit `psql: error: invalid integer value "4Y9ri4f5VuwyD5QCbDmPLK+Oj2MT" for connection option "port"`.
- **Ursache:** POSTGRES_PASSWORD enthaelt Base64-Sonderzeichen (`+`, `/`, `=`). Der URL-Parser interpretiert das erste Sonderzeichen als Ende des Passworts → der Rest landet im Port-Feld.
- **Fix:** Passwort in der DATABASE_URL URL-encoden. Python:
  ```python
  import urllib.parse
  urllib.parse.quote(pw, safe='')
  ```
  In der `.env.production` nur die `DATABASE_URL` aktualisieren; `POSTGRES_PASSWORD` selbst bleibt im Klartext.

### Postgres-Image-Wechsel fuehrt nicht zu Data-Loss
- **Sorge:** Wechsel von `postgres:16-alpine` auf `pgvector/pgvector:pg16` (Commit `31c0402` im Sprint-1-Delta) koennte initdb neu triggern und Production-Daten loeschen.
- **Realitaet:** Docker re-initialisiert PGDATA NUR wenn das Volume leer ist. Wenn Volume persistent (hier `docker_pgdata`), startet der neue Container ueber den bestehenden Daten. Kein Data-Loss.
- **Precaution:** Vor Image-Wechsel trotzdem `pg_dumpall` in `/opt/kmuhub/backups/` ablegen. Eine fehlende oder auf falschen Pfad geroutete `docker-compose.yml`-Volume-Referenz wuerde sonst ein frisches Volume anlegen.

### Parallel-Build OOM auf CPX42
- **Symptom:** `docker compose build` ohne `--parallel`-Flag killt sich selbst mit `failed to execute bake: signal: killed` nach 2-3 Minuten.
- **Ursache:** Docker Buildx Bake versucht alle ~17 Services parallel zu bauen. Go-Compilation pro Service ~1 GB, parallel → >15 GB RAM. CPX42 hat 16 GB ohne Swap → OOM-Kill.
- **Fix:** Sequenziell bauen: `for svc in ...; do $COMPOSE build "$svc"; done`, oder `BUILDKIT_MAX_PARALLELISM=3` setzen.

### Backup-Cron laeuft silent nicht
- **Symptom:** `/opt/kmuhub/backups/` enthaelt nur einen einzigen alten Dump, obwohl `crontab -u deploy -l` einen 02:00-Eintrag zeigt.
- **Ursache:** `/var/log/kmuhub-backup.log` existiert nicht. Entweder fehlt die Datei mit passenden Permissions, oder das Script failed vor dem ersten `echo`.
- **Diagnose (Sprint-2-Task):** Check `sudo -u deploy /opt/kmuhub/deploy/scripts/backup.sh` manuell, `journalctl -u cron`, und Permissions auf `/var/log/`.

## Tenant-Isolation Lessons (Sprint 2 Welle 2D, 2026-04-28)

Drei Anti-Pattern, die Welle 1 hinterlassen hat. Vor jedem neuen Modul-Wiring pruefen.

### Hardcoded `<modul>PlaceholderTenantID`-Konstanten
- **Symptom:** Routes sehen so aus: `tenantID, _ := uuid.Parse(rapportePlaceholderTenantID)`. Cross-Tenant-Isolation auf HTTP-Ebene effektiv aus.
- **Fix:** `tenantID, err := middleware.GetTenantID(r.Context())` als erste Aktion in jedem Handler. Bei Fehler 401 zurueck (kein Default-Tenant). Konstante komplett loeschen — Compiler findet alle Reste.
- **Test:** `gateway/tenant_isolation_test.go`-Pattern kopieren: no-tenant + empty-tid + valid-tid.

### `tenant_id`-Spalte ohne SELECT im Repository
- **Symptom:** JWT signiert `tid`, Middleware liest aber leeren String → `ErrMissingTenantID` → 401 trotz korrekter Auth.
- **Diagnose:** `grep -n "SELECT" backend/internal/<modul>/postgres_repository.go` und schauen ob `tenant_id` im Column-Set ist. War Hotfix-Anlass `c421fac` fuer auth.
- **Lesson:** Wenn ein Migration-Patch eine Spalte hinzufuegt, im selben Commit alle Repository-SELECTs aktualisieren — sonst kommt der Wert nie in der App-Schicht an.

### `getTenantID` ruft heimlich `GetUserID`
- **Symptom:** Code compiliert, Tests laufen — aber jeder Call schreibt UserID als TenantID in den gRPC-Request. In Single-Tenant-Dev faellt das nicht auf, in Multi-Tenant-Tests schlagen alle Cross-Tenant-Checks fehl.
- **War in:** `route_biz.go::getTenantID(r)` → 90 Call-Sites quer durch biz/billing/invoices/quotes/ext/hr/lexware/bexio/datev.
- **Fix:** `getTenantID(r)` returniert `(string, error)`, ruft `middleware.GetTenantID` auf, Callsites pruefen `err != nil`. Beim Refactoring darauf achten, dass kein Helper noch UserID-by-mistake durchschleift.

### Proto-Requests ohne `tenant_id`-Field
- **Symptom:** gRPC-Service-Code hat einen Helper wie `extractTenantID()` der eine Konstante zurueckgibt, weil die Proto-Definition kein `tenant_id` kennt.
- **Fix:** Proto-File patchen (`tenant_id = N;` mit naechstem freien Field-Index), `make proto` (oder protoc-Aufruf), gRPC-Server liest `tenant_id` aus dem gRPC-Context via `middleware.GetTenantID(ctx)`. War in dialer + helpdesk auf 13 RPCs.

### gRPC liest `tenant_id` aus Proto-Request statt aus Context (Welle 3.5)
- **Symptom:** gRPC-Server-Methode ruft `req.GetTenantId()` und filtert damit Repos. Funktioniert im Happy-Path. Bei Service-zu-Service-Calls oder einem kompromittierten Gateway kann ein Caller eine fremde TenantID ins Request-Feld schreiben — der Repo-Filter folgt willig.
- **Fix:** `tenantID, err := middleware.GetTenantID(ctx)` in jedem gRPC-Handler. Proto-Feld bleibt im Wire-Format, wird aber serverseitig ignoriert oder hoechstens fuer Logging genutzt. Welle 3.5 hat das Pattern auf 14+ Methoden in chat/crm/work/video/dialer-gRPC umgestellt.
- **Test:** Tenant-Isolation-Tests muessen einen Two-Tenant-Scenario abdecken (User Tenant A schickt Request mit `tenant_id=B` im Body — Backend muss `tenant_id=A` aus dem Context durchsetzen).

## Stale IDE-Diagnostics bei Cross-Stream-Subagent-Refactor (Welle 4B, 2026-05-07 — bestaetigt Sprint 3, 2026-05-08)

- **Symptom:** Subagent-Output sagt "alles gruen — go build/vet/test alle pass". IDE-Diagnostics-Stream meldet aber kurz danach Sig-Drift in `cmd/*/main.go` oder `server/*_grpc.go` mit Compiler-Errors wie `*X.PostgresRepository does not implement Y.Repository (wrong type for method Z)`.
- **Ursache:** IDE-Diagnostics arbeiten auf einem Snapshot des File-System-Zustands der manchmal hinter dem letzten Subagent-Save haengt. Bei Subagents die ueber 100+ Files refactoren und am Ende einen Sweep ueber Wirings machen, kommt das Diagnostic-Update nicht synchron mit dem letzten Save.
- **Fix:** **Authoritative Verifikation ist immer `cd backend && go build -tags no_wasm ./...` + `go vet ./...` + `go test ./...`** (frischer Build vom Disk-State). Wenn alle drei gruen sind, ist der Code korrekt — unabhaengig davon was der LSP-Cache zeigt. Nicht auf IDE-Diagnostics als Compiler-Authority verlassen.
- **Beispiel Welle 4B:** Drei Mal trafen Diagnostics mit "PostgresRepository implementiert Interface nicht" — `go build ./...` direkt war jedes Mal clean.
- **Sprint 3 bestaetigt:** Das Muster wiederholte sich in Welle 2A (cmd/dialer/main.go, cmd/document/main.go schienen broken laut LSP). `go build -tags no_wasm ./...` war clean. LSP-Cache-Refresh loest das Symptom, Code-Aenderungen wegen LSP-Errors sind falsche Behandlung.
- Im Session-Memory dokumentiert: `project_sprint2_welle4b.md` + `project_sprint3_session_20260508.md`.

## Frontend-Mutation-Patterns (Welle 3.5)

### Doppelklick-Guard auf zweistufigen Mutations
- **Symptom:** User klickt schnell zweimal auf "Aufnahme starten". Erste Mutation erstellt eine Recording-Row, zweite versucht es nochmal — Race-Condition zwischen `startRecording` und dem nachfolgenden `confirmInitiatorConsent`. Bei Fehlschlag steht eine Recording-Row ohne Consent-Stamp in der DB.
- **Fix:** Guard am Anfang des Click-Handlers gegen ALLE involvierten Mutations: `if (startRecording.isPending || stopRecording.isPending || confirmInitiatorConsent.isPending) return`. TanStack-Query `isPending` ist die richtige Quelle, nicht ein eigenes `useState`-Flag.
- **Pattern in:** `desktop/src/renderer/src/features/video/CallControls.tsx`.

### Try/catch um zweite Mutation einer two-step-Sequenz
- **Symptom:** `await mutateA(); await mutateB()` — wenn `mutateB` failt, hinterlaesst `mutateA` einen Orphan-State. User sieht keinen Fehler, weil React-Query den Throw schluckt aber das Rendering nicht aktualisiert.
- **Fix:** `try { await mutateB.mutateAsync(...) } catch (err) { toast.error(err instanceof Error ? err.message : 'Fallback-Text') }`. Dialog-Close NUR im Success-Pfad. User kann erneut bestaetigen ohne neue Row anzulegen.
- **Pattern in:** `CallControls.handleConfirmStart` (Welle 3.5-Fix).

### Offline-Queue: 409 ist Retry, nicht Success
- **Symptom:** Offline-Queue drained beim `online`-Event. Backend antwortet 409 Conflict (Idempotency-Key in-flight). Queue interpretiert non-2xx als generic-fail oder schlimmer als Success und droppt das Item.
- **Fix:** 409 explizit als Retry-Class behandeln (`Retry-After`-Header respektieren), nicht als terminales Failure. Queue setzt das Item zurueck in den Pending-Pool und versucht es nach Backoff neu. `Content-Type: application/json` nur setzen wenn das Item tatsaechlich einen Body hat (sonst lehnt das Backend mit 400 ab).
- **Pattern in:** `desktop/src/renderer/src/api/offline-queue.ts` (Welle-3.5-Fix).

## smoke.sh `curl -sf` + `-w "%{http_code}"`-Konkat-Bug (2026-05-09, `308e9b2`)

- **Symptom:** Smoke-Test 1 (`/contacts` unauthenticated) failt mit `expected '401', got '401000'`. Test-Logik korrekt, aber das verglichene Code-String-Argument enthaelt zwei Werte hintereinander.
- **Ursache:** Pattern `curl -sf -o /dev/null -w "%{http_code}" "$URL" || echo "000"`. Mit `-f` setzt curl Exit 22 bei HTTP >= 400, ABER hat den Code via `-w` schon auf stdout geschrieben. Der `||`-Fallback `echo "000"` wird zusaetzlich getriggert → Output ist die Konkatenation `401` + `000` = `"401000"`.
- **Fix:** `-f` aus allen `-w "%{http_code}"`- und `-w "%{time_total}"`-Patterns entfernen, nur `-s` lassen. Die `|| echo "000"`-Fallback fuer Connection-Failures bleibt erhalten und greift dann nur noch wenn curl wirklich keine Connection bekam (curl exit != 0 ohne Output).
- **Seiteneffekt-Fund:** Beim Audit der Fix-Stellen kamen zwei outdated Endpoints heraus: Chat-Channel-POST braucht `is_private: false` (nicht `type: public`), Dashboard ist `/api/v1/dashboard/layout` (nicht `/api/v1/dashboard`).
- **Followup:** Tests 9/10/11 (Contacts CRUD) bleiben rot — frisch registrierte Smoke-User landen auf Default-Rolle `member` mit Read-Only-Permissions. Service-User-Bootstrap fuer Smoke-Tests ist eigene Sprint-4-Task.
- **Lesson:** Die `curl -sf -w "%{http_code}"`-Kombination ist ein subtiler Anti-Pattern in Shell-basierten Smoke-Tests. Wer einen HTTP-Code zurueck will, darf das `-f` nicht setzen — sonst ueberlagern Exit-Code- und Output-Logik.

## Welle-1-Hotfix-Lessons (Sprint 3 Welle 1, 2026-05-08)

Aus dem Marathon-Deploy `980eba3` → `3abec5f` (Migration 81 → 115). 9 Hotfix-Commits, 7 versteckte Production-Bugs. Details in MEMORY `project_sprint3_welle1_deploy.md`.

### Image-Pin ohne Expiration-Tracking ist fragil

- **Symptom 1 (`minio/mc`):** `docker compose up -d createbucket` failt mit `Error response from daemon: pull access denied for minio/mc:RELEASE.2025-04-16T19-25-36Z`. Image war vom Docker-Hub-Maintainer geloescht.
- **Symptom 2 (`redis`):** `redis_1 | Bad file format reading the append only file: make a backup of your AOF file, then use ./redis-check-aof --fix`. Persistent-Volume hat RDB-v12 (geschrieben von redis 7.4+) — `redis:7.2.7-alpine` kann es nicht lesen.
- **Ursache:** Image-Pinning ohne Tracking. `latest` ist instabil, aber explizite Pins koennen entweder geloescht werden (minio/mc) oder Down-Grade beim Volume-Swap sein (redis).
- **Fix:** Tags rotieren auf neuere Releases (minio/mc: `2025-05-21...`, redis: `7.4-alpine`).
- **Followup:** Renovate/Dependabot fuer Image-Tags konfigurieren — automatisches PR bei Image-Updates plus CI-Smoke-Test.

### Migration referenziert FK ohne dass Tabelle existiert

- **Symptom:** Migrate-Run failt mit `pq: relation "tenants" does not exist` bei Migration 000114, obwohl die Migration auf Dev gegruent gerunnt war.
- **Ursache:** Auf Dev existierte `tenants` aus alten Test-Setups. Production-DB war essenziell leer (91 KB pg_dump) und kannte die Tabelle nicht. Migrations 000114+115 referenzierten `tenants(id)` ohne Bootstrap-Statement.
- **Fix:** `CREATE TABLE IF NOT EXISTS tenants(...)` + Sentinel-Insert (`'00000000-0000-0000-0000-000000000001'`) am Anfang von 000114 nachgereicht (`c7a9a76`). Idempotent — laeuft auf Dev no-op, auf Prod legt es Tabelle + Sentinel an.
- **Lesson:** Lokaler `make migrate-up` von leerer DB als Pre-Commit-Hook OR CI-Job. Welle 1 hat 9 Migrations gleichzeitig drauf gehabt — alle gegen Schemas getestet, keine gegen leere DB.

### healthcheck.sh hatte drei unabhaengige Bugs

1. **`set -e` + `((HEALTHY++))`:** `set -e` bricht bei Exit-Code != 0 ab. `((HEALTHY++))` evaluiert zuerst, dann inkrementiert — wenn `HEALTHY=0`, ist der Pre-Increment-Wert `0` → exit 1 → set-e killt das Script nach dem ersten gesunden Service. Fix: `HEALTHY=$((HEALTHY+1))`.
2. **Compose-Pfad-Drift:** Skript suchte Compose-Files unter `/opt/kmuhub/`, die liegen aber in `/opt/kmuhub/deploy/docker/`. Selbe Bug-Klasse wie `980eba3`-Fix in `deploy.sh`. Fix: `COMPOSE_FILES_DIR + ENV_FILE` aus `deploy.sh` uebernommen.
3. **Caddy-Domain hardcoded:** Skript curlte `https://localhost`, Caddy hat aber Vhost auf `app.zentria.tech`. Cert-Mismatch. Fix: `--resolve $CADDY_HEALTHCHECK_HOST:443:127.0.0.1`.
- **Lesson:** Standalone-Skripte werden nie integrativ getestet. Bei der naechsten Runde `healthcheck.sh` in `deploy.sh` als Step nutzen, damit Drift sofort auffaellt.

### Parallel `docker buildx bake` killt 16-GB-Hosts

- **Symptom:** `docker buildx bake` ohne `--parallel`-Flag killt sich selbst mit `failed to execute bake: signal: killed` nach 2-3 Minuten. Server hat 24 Go-Microservices, jeder Build ~1 GB → >24 GB Memory.
- **Ursache:** CPX42 hat 16 GB RAM ohne Swap. OOM-Kill.
- **Fix:** Step 3 in `deploy.sh` macht jetzt `for svc in app_services; do compose build $svc; done`. Sequenziell, jeder Service schliesst seinen Prozess vor dem naechsten. Build-Dauer ~10 Min, akzeptabel.
- **Followup:** CCX21 (32 GB) fuer Pilot-1 evaluieren — dann waere parallel-bake wieder moeglich, Build-Dauer ~3-4 Min.

### Native-Windows-Ansible funktioniert nicht

- **Symptom:** `pip install --user ansible-core` durchlaufen, aber `ansible-playbook --version` failt mit `ModuleNotFoundError: No module named 'grp'` (in `ansible/cli/__init__.py`).
- **Ursache:** Ansible nutzt das Unix-only `grp`-Modul (Posix Group-Database) — wird auf Windows nicht ausgeliefert. CPython auf Windows hat das Modul nicht im Standard-Library-Pool.
- **Fix (Windows-Dev-Box):** Ansible via Docker-Container nutzen — `willhallonline/ansible:latest` enthaelt ansible-core 2.19 + Collections (`community.general`, `community.docker`, `community.crypto`, `ansible.posix`) + `ansible-lint`. Wrapper-Pattern:
  ```bash
  MSYS_NO_PATHCONV=1 docker run --rm \
    -e ANSIBLE_ROLES_PATH=/work/deploy/ansible/roles \
    -v "/c/Users/Luke/Documents/KMU Hub:/work" \
    -w /work/deploy/ansible \
    willhallonline/ansible:latest \
    ansible-playbook -i inventory/hosts.yml --syntax-check site.yml
  ```
  `MSYS_NO_PATHCONV=1` ist ZWINGEND in Git-Bash, sonst translatiert MSYS `/work` zu `C:/Program Files/Git/work`.
- **Real-Apply** gegen Linux-Server weiterhin nur von einer Linux-Control-Node. Docker-Wrapper deckt nur Syntax-Check / Lint / List-Tasks / `--check`-Dry-Run ab.

## Git-Workflow & Recovery (Sprint 1+)

- **Branch-Strategie:** Ab Sprint 1 (2026-04-18) ist **direct-to-main** Default. Keine Feature-Branches, keine PRs — ausser der User fordert explizit einen PR. Sprint 0 lief noch mit PRs.
- **CI-Rot-Recovery:** Immer `git revert <sha>` (erzeugt neuen Commit, bewahrt History).
- **NIE** `git reset --hard` auf gepushte Commits.
- **NIE** Force-Push (`git push --force`) auf `main`. Auch nicht "kurz mal zum aufraeumen".
- **Commit-Messages:** Englisch, imperativ ("Add contact endpoint", nicht "Added contact endpoint"). Conventional Commits (`feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`).
- **Push-Rhythmus:** Am Ende jeder Session pushen, um lokale/remote Divergenz zu vermeiden.

## Verwandte Notes
- [[architektur]] — Architektur-Regeln
- [[i18n]] — i18n-Architektur & Konventionen
- [[deployment]] — Docker & CI/CD
- [[stack]] — Dev-Tooling & Pfade
