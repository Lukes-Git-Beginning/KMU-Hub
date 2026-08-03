---
tags: [troubleshooting, debug]
updated: 2026-08-03
---
# Troubleshooting & Bekannte Probleme

## LiveKit-Webhook-Delivery: Prod sandte keine Webhooks (✅ BEHOBEN Wave 0, 2026-06-23)

**Symptom:** Recordings werden in Prod nie `completed`; Auto-Close (`room_finished`) unmöglich.
**Ursache:** `deploy/docker/livekit-secrets.yaml.tmpl` ist **last-wins** (effektive Prod-Config = letztes
`--config`, KEIN key-merge mit Base-`livekit.yaml`) und enthält `webhook.api_key`, aber **kein
`webhook.urls`** und **keinen `room`-Block** → LiveKit hat kein Sende-Ziel + kein `empty_timeout`.
**Diagnose (read-only):** `docker exec docker-livekit-1 cat /etc/livekit-secrets.yaml | grep -A4 webhook`
(zeigt nur api_key) + `docker logs docker-gateway-1 | grep -i webhook` (leer = nie empfangen).
**Lehre:** Last-wins gilt für den GANZEN Config-Baum — fehlende Top-Level-Keys (`webhook.urls`, `room`)
werden NICHT aus der Base ergänzt; das Secrets-Template muss self-contained sein (keys + webhook.urls +
room + voller rtc-Block). Gleiche Falle wie der rtc-only-Overlay-Crash (Video-Resume 2026-06-21) und der
Compose-Overlay-Gap (unten). **Fix-Spec:** `.planning/meeting-parity/wave-0-autoclose.md` (Wave 0.1/0.2).
**✅ BEHOBEN (Wave 0, 2026-06-23, `d1be2bd4`):** `webhook.urls` + `room.empty_timeout:900` ins Secrets-Template;
`room_finished`→gRPC `CompleteMeetingByRoom` + Backstop-Sweeper (prod-verifiziert aktiv). **2-Schicht-Bug:** der
Egress-gRPC (`CompleteRecordingByEgress`/`FailRecordingByEgress`) war NIE in die `VideoService_ServiceDesc`
registriert (Hand-Stub `video_egress_ext.go`) → `codes.Unimplemented`; via Proto-Regen behoben. **⚠ DEPLOY-FALLE:**
`compose up -d livekit` lädt geänderte Mount-Config NICHT → Config-Revision-`labels` am `livekit`-Service in
`docker-compose.prod.yml` bumpen erzwingt Container-Recreate (sonst greift kein tmpl-Change).

## Compose-Overlay-Gap: Dev-Secrets liefen in Production (2026-06-05)

**Symptom-Klasse:** Feature funktioniert "irgendwie nicht" in Prod (Video-Calls tot, Webhook-Validierung
im Skip-Modus), obwohl `.env.production` korrekte Werte hat. **Ursache:** Die Basis-Compose hardkodierte
Secrets in `environment:`-Bloecken; das Prod-Overlay ueberschrieb nur 3 von ~20 Stellen — alle 24
Services liefen mit Dev-`JWT_SECRET`, `minioadmin`, `devkey`. Diagnose-Muster (wertfrei!):
`docker exec <c> printenv <VAR> | grep -c <dev-marker>` → Count statt Wert.
**Fix-Architektur:** `${VAR:-dev-default}`-Interpolation ueberall + Requirements-Assertion, siehe
[[deployment]] + [[security]]. Incident-Historie: `docs/livekit-env-production-followups.md` (F-A–F-K).

Drei verwandte Fallen aus derselben Session:
- **"Populated by gateway"-Kommentare luegen:** `JoinCallResponse.ws_url`/`StartMeetingResponse.token`
  waren IMMER leer — der Kommentar beschrieb nie existenten Code, der Desktop-Client speiste die leeren
  Felder direkt in die LiveKit-Connection. Bei "X wird woanders befuellt"-Kommentaren: Befuellung grep-verifizieren.
- **RLS-Read-Gap, dritter Fund:** `call_sessions`-SELECTs lasen `tenant_id` nicht → Join-INSERT in
  `call_participants` mit `uuid.Nil` → SQLSTATE 42501. Wer TenantID vererbt, MUSS sie zuruecklesen.
- **CI-Paths-Filter:** `deploy/**`-only-Commits triggern kein CI; CD haengt an CI →
  `gh workflow run CD --ref main`.

## Production-DB-Zugriff (Sprint 4 Welle 1 Lesson)

**psql-User ist `kmuhub`, nicht `postgres`.** `docker-compose.yml` setzt `POSTGRES_USER: kmuhub` und `POSTGRES_DB: kmuhub`. Default-`postgres`-Role existiert nicht in der Production-DB. Manuelle ad-hoc-SQL-Commands:

```bash
sudo docker compose --env-file /opt/kmuhub/.env.production \
  -f /opt/kmuhub/deploy/docker/docker-compose.yml \
  -f /opt/kmuhub/deploy/docker/docker-compose.prod.yml \
  exec -T postgres psql -U kmuhub -d kmuhub -c "SELECT ..."
```

`-U postgres` crasht mit `FATAL: role "postgres" does not exist`.

## Auto-Rollback-Drift (Sprint 4 Welle 1 Lesson)

`deploy.sh` Auto-Rollback bei Smoke-Failure rollback **nur Code, nicht DB-Migrations**. Wenn die fehlgeschlagene Welle eine Schema-Änderung enthielt (z.B. NOT-NULL-Spalte), entsteht Drift: DB hat die Spalte, Code kennt sie nicht → naechste INSERT crasht.

**Triage bei Smoke-Fail nach Schema-Aenderung:**
1. Pruefen ob Smoke-Failure echtes Problem oder False-Positive (z.B. `SMOKE_ADMIN_TOKEN` expired, OnlyOffice known issue).
2. Wenn False-Positive: `deploy.sh --skip-smoke` als Forward-Fix. Code+Schema kommen wieder ueberein.
3. Wenn echtes Problem: vor `--skip-smoke` erst echten Bug fixen, sonst kettenwirkung.

`--skip-smoke` ist die Notbremse, sparingly nutzen — der Smoke ist die letzte Schutzschicht vor Prod-Regression.

## Migration-Backfill-Spaltennamen IMMER verifizieren (Sprint 4 Welle 1 Lesson)

Migration 000119 wurde anhand von Welle-0.6-Pattern geschrieben, aber `dialer_call_events.session_id` war eine Annahme — echte Spalte heisst `dialer_call_session_id`. Crash beim Production-Deploy, dirty `schema_migrations`-Tabelle.

**Pre-Flight vor Migrations mit Backfill-JOINs:**
```bash
grep -A20 "CREATE TABLE.*<table>" backend/migrations/*.sql
```
Spaltennamen visuell verifizieren, nicht annehmen. golang-migrate killt sonst die Tx in der Mitte und hinterlaesst die Migration als `version=N, dirty=true`.

**Recovery aus dirty=true:**
```bash
psql -U kmuhub -d kmuhub -c "UPDATE schema_migrations SET version=<previous>, dirty=false;"
# dann re-deploy mit gefixter Migration
```

## Frontend-QA-Lessons FE-Lane (2026-06-10/11, Marathon Strom L)

### React-ErrorBoundary-Crashes sind Playwright-unsichtbar
ErrorBoundary-Renderfehler erzeugen KEIN `page.on('pageerror')`-Event — QA, die nur URL-Change + pageErrors prüft, meldet grün, während die Zielseite „Etwas ist schiefgelaufen" zeigt (Phase-10-P0: FinanzenPage-Crash durch Mock-Feldnamen-Mismatch `items`≠`line_items`). Pflicht: nach JEDER Navigation `checkErrorBoundary()` (body.innerText-Scan) als harte fail-Bedingung.

### Literal-Listen-Falle bei dynamischen t()-Codes
Muster `isActionCode(x) ? t(`prefix.${x}`) : x` mit `as const`-Literal-Array: jeder NEUE Code muss in die Liste, sonst rendert die UI den rohen Code. rawKeys-Scans finden das NICHT (Codes ohne Punkte). Gate: Screenshot der betroffenen Sektion wirklich ansehen (Phase-10-P1: `HISTORY_ACTION_CODES` in VertraegePage).

### Flag-/State-Overrides in Playwright: addInitScript, nie evaluate
`page.evaluate()` nach dem Mount erreicht `useMemo`-Logik nie — Override via `page.addInitScript()` VOR dem Load + `page.reload({waitUntil:'networkidle'})`. addInitScript akkumuliert über den Browser-Kontext → pro Flag-Szenario frischer `browser.newContext()`. Assert-Muster: Positiv-Kontrolle (Quelle sichtbar mit Flag AN) + gezieltes Verschwinden (Flag AUS) bei weiterhin sichtbarer anderer Quelle.

### Weiche QA-Asserts (wiederkehrendes falsches-Grün-Muster)
Verboten: bloßer URL-Match, OR-Ketten schwacher Bedingungen, stille Fallback-Zweige („Edit-Button nicht gefunden → pass"). pass muss eine Zustandsänderung beweisen, die NUR durch die Kern-Interaktion entstehen kann. Builder-Selbstreport nie ohne unabhängige Wiederholung (Verifier-Subagent) + eigenes Screenshot-Review trauen.

### Verifier-Subagents: Encoding-Fehlalarme
Inhalts-Urteile über UTF-8-Dateien NIE via PowerShell `Get-Content` (PS 5.1 ohne `-Encoding utf8` → falscher „Mojibake-P0"-Alarm in Welle 2) — Read/Grep-Tools im Verifier-Prompt vorschreiben; Hauptsession prüft Verifier-P0s einmal selbst gegen, bevor Fix-Zyklen starten.

### tsc-Crash „Debug Failure. No error for last overload signature"
Bekannter TS-Tooling-Bug bei gescoptem tsc über den Chat-Import-Graphen — erreicht seit Phase 9 (UserProfileCard an 5 Call-Sites) auch den vertraege-Graphen. Kein Code-Fehler (App + QA grün); Scope granularer schneiden, TS-Version-Bump = offene Darien-Frage.

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

## protojson `int64` → JSON-String beim Umstieg auf `response.Proto` (R3-P0-1, 2026-06-21)
- `response.Proto` (protojson) fixt `Timestamp` (`{seconds,nanos}`→RFC3339), rendert aber `int64`/`uint64` als JSON-**String** (proto3-Spec), waehrend `encoding/json` Zahlen liefert.
- **Falle:** Beim Umstellen `response.JSON`→`response.Proto` pro Handler die FE auf int64-Felder pruefen. Beispiel: `Recording.file_size_bytes` (int64) wurde String → FE-Typ auf `number | string | null` geweitet + `Number()`-Coerce beim Lesen.
- **Regel:** Kein globaler Blind-Sweep aller ~40 Route-Dateien — pro Modul int64-FE-Audit. Enums unkritisch (`encoding/json` rendert proto-Enums = int32-Typen ohnehin als Zahl, `UseEnumNumbers:true` aendert nichts).
- **Welle 1/1b/2 migriert (2026-07-05, `ff2d5db1`+`a3a67f01`):** rapporte/inventar = **echte Bugfixes** (kein FE-Normalizer → sichtbar kaputt). CRM = nur Konsistenz (Zeitfelder in `crm.proto` sind bereits `string`, kein Timestamp-Objekt). Welle 2 (work/wiki/hr/calendar/helpdesk/inbox/vertraege/notification) = Konsistenz-Härtung, da alle Module client-seitig `normalizeWireTimestamps` (`api/*-client.ts`) fahren. int64-FE-Coerce gemacht: inventar `quantity`/`min_quantity`, work/wiki/document-version `file_size`.
- **[GELÖST 2026-07-12 Runde 3 — alle drei via `protoc` regeneriert, `c9d40d66`] Fake-pb.go war NICHT migrierbar (3 Dateien, nicht nur plugin):** `plugin.pb.go`, **`biz/v1/lexware.pb.go`, `biz/v1/datev_upload.pb.go`** tragen `DO NOT EDIT`-Header, haben aber `grep -c ProtoReflect` = 0 → Typen erfüllen `proto.Message` nicht → `response.Proto` bricht den Build. Makefile `proto-biz` dokumentiert lexware/datev selbst als „regenerate deliberately". Fix = echte `protoc`-Regenerierung (kein `proto-plugin`-Make-Target existiert). Nur plugin hat sichtbaren Bug (ExecutionLog/PluginDetail „Invalid Date"); lexware/datev konvertieren manuell. Systemcheck: `grep -Lc ProtoReflect backend/proto/*/v1/*.pb.go`.
- **map-envelope-Falle:** Handler wie `map[string]any{"folder": resp.Folder}` reichen den verschachtelten Proto durch — `encoding/json` serialisiert dessen Timestamps weiterhin als `{seconds,nanos}`. Fix: verschachtelten Wert lokal protojson→`json.RawMessage` marshalen (so bei inbox canned-response) ODER `response.Proto(resp)` wenn die Response-Message dieselbe Envelope-Form hat. **hr map-wrapper 2026-07-12 migriert** (12 Handler, `hrMarshalSlice[T]`+`cannedResponseMarshaler`; nullable Werte → `json.RawMessage("null")` statt protojson-`"{}"`, sonst brechen JS-Truthy-Checks). **document** bleibt bewusst offen (`lean:`-Marker in `route_document.go`; FE normalisiert client-seitig, `file_size` muss numerisch bleiben).
- **2026-07-12 Runde 2 migriert (14 Module, 5 durchlaufende Subagenten-Wellen, worktree-isoliert):** auth/berichte/automation/fuhrpark/einkauf/produktion/security/formulare/settings/vermietung/schichten/booking/integration + hr-Envelope. Fast alle echte Bugfixes (FE-Typen erwarteten längst `string`); int64-FE-Coerce nur fuhrpark (mileage_km/cost_cents/km) + security (AuditEntry.sequence_num). **Runde 3 (12-07, FINAL) — R3-P0-1 getilgt:** biz/Finanzen (`3374b3d5`, 47 Handler in 6 `route_biz_*` via 2 Worktree-Agenten; Geld-Wire-Risiko war NULL — Beträge = proto-`string`-Decimals, FE string-safe, keine FE-Änderung; einziges int64 `GobdDocumentProto.file_size_bytes` = tote UI) + Fake-pb.go plugin/lexware/datev via `protoc` (protoc-gen-go v1.36.11) echt regeneriert (`c9d40d66`, satisfy `proto.Message`, keine API-Drift); plugin 17 Handler→`response.Proto`/`ProtoList` (fixt ExecutionLog „Invalid Date"), lexware/datev NUR Regen (Handler nutzen bewusste `.AsTime()`-Reshape-Maps → kein Bug, Wechsel würde Wire-Shape ändern); Makefile: lexware/datev in `proto-biz` + neues `proto-plugin`-Target verhindert Fake-pb.go-Rezidiv. **crm/chat/email = KEIN Bug** (Timestamps sind `string` by design — die alte „Welle 3–6"-Liste nannte sie fälschlich). Referenz: `route_dialer.go`/`route_rapporte.go`, Envelope: `route_hr.go`.

## FE/BE-Contract-Mismatch (nested/wrapped vs flach, camelCase vs snake_case, falsche URL-Pfade) — Schwester-Klasse (2026-07-12)
Timestamp-/int64-**unabhängige** Schwester der ProtoTimestamp-Schuld: der FE-Client liest die Wire-Shape falsch (flach statt verschachtelt/gewrappt, camelCase statt BE-snake_case, oder ruft einen falschen URL-Pfad), und die MSW-Mocks bilden **exakt die falsche FE-Erwartung** nach → maskiert in Demo, bricht erst gegen echtes BE (`undefined`→leere Views, 400/404).
- **Wire-Wahrheit:** `response.Proto` = flaches Objekt · `response.ProtoList` = **nacktes Array** · map-envelope = `{key:...}` (via `hrMarshalSlice`/`cannedResponseMarshaler`). protojson `UseProtoNames:true` (snake_case), `UseEnumNumbers:true` (Enums als **Int**), `EmitUnpopulated:false` (Zero-Werte weggelassen), int64→String. Saubere FE-Referenz: `helpdesk-client.ts` (`unwrapList`), `booking-client.ts` (1:1), `hr-client.ts` `hrTimeApi`/`hrLeaveApi` (async unwrap + snake→camel/Enum-Int/Decimal-Adapter).
- **6-Baustellen-Kampagne (3 Wellen, Worktree-Agenten + Main-Session-Gate, `c13586a3`→`39f6393c`→`e8bb19df`, alle CD-deployt):** hr/Leave (Envelope-Unwrap + Adapter; **Bonus-Bug:** ausgehende POST-Bodies camelCase→snake_case) · Integrations (**BE-Fix:** `HandleGetLinkStatus` ehrt jetzt `{platform}`-Pfad-Param + liefert flach — Proto-Request hat kein platform-Feld) · auth/2FA+Sessions+Audit (`security-client.ts` Pfade/Bodies; Audit `first_invalid_sequence`; `RecoveryCodes.recovery_codes`) · automation (echte Stats-Felder, `success_rate` FE-berechnet ÷0-Guard, Dead-Field `total`→`total_count`) · produktion (`.order`-Envelope-Unwrap, Typ schon snake_case 1:1 → kein Adapter) · formulare (Drilldown auf 4 echte BE-Zähler gekappt, fiktive Analytics fieldStats/conversion/dropoff/timeseries raus — kein Backend, bräuchte View-Tracking; 18 tote i18n-Keys symmetrisch über 4 Locales).
- **Zwei bewusste BE-Wahrheit-Ausnahmen:** Integrations (Handler war buggy → **BE** gefixt) · formulare (FE-Erwartung auf BE-Realität **reduziert**, echte Analytics = Produkt-Ticket).
- **Gate (Worktree-Agenten haben kein `node_modules`):** Main-Session cherry-pickt + `tsc --noEmit` (Default-tsconfig) + `eslint src/` (0 errors) + `vitest run` + Go bei BE-Änderung, committet gesammelt, pausiert pro Welle. ⚠ Electron-Screenshot-QA nicht durchführbar (GUI nicht erfassbar). Detail + Backlog: memory `project_fe_be_contract_mismatch_20260712`.

## „Die Route fehlt" ist meistens „das FE ruft ein Segment zu viel" (Nachtlauf 4, 2026-08-03)

Unterklasse des Contract-Mismatch oben, aber mit eigener Diagnose-Reihenfolge — und die falsche
Reihenfolge kostet einen halben Tag Backend-Arbeit fuer nichts. Der Backend-Nachtloop hatte vier
Units im Backlog, die als „fehlende Route" gescopt waren. In **keinem** der vier Faelle fehlte etwas
im Backend:

| Symptom | Realitaet |
|---|---|
| Kontakt-Timeline liefert 404 | Route existiert seit Langem, E2E-getestet. `useTimeline.ts:40` rief `/api/v1/crm/contacts/{id}/timeline` — ein `/crm/`-Segment, das **kein** anderer CRM-Hook nutzt |
| Ressourcen-Buchung liefert 404 | `BookResource`/`CancelBooking` existieren inkl. EXCLUDE-GIST-Konfliktpruefung. `useResources.ts:123,138` rief `/calendar/resources/bookings` statt `/calendar/bookings`. `GET /resources/{id}/availability` war korrekt — der Fehler betraf nur zwei von drei Aufrufen derselben Datei |
| Admin-Billing „ohne Backend" | Das FE ruft **gar keine** Route: `useBilling.ts` liefert `Promise.resolve(MOCK_*)` bzw. liest `localStorage`. Die Daten existieren real unter `/api/v1/admin/subscription` und `/admin/license` |
| Gehaltsabrechnungen fehlen | Es gibt keine Datenquelle: `EmployeeProfile` traegt nur `HourlyRate`, kein Monatsgehalt, keinen Payroll-Lauf. Der MSW-Mock erfindet `net = gross * 0.675`. Die DATEV-Integration ist Buchungsstapel-Export, keine Lohnquelle |

**Diagnose-Reihenfolge bei jedem gemeldeten 404 auf einer „fehlenden" Route:**

1. `grep -rn "<pfad-fragment>" backend/internal/gateway/` — ist die Route registriert, nur anders?
2. Den FE-Pfad gegen die **Nachbar-Hooks derselben Domain** halten. Ein Segment, das nur *eine*
   Datei fuehrt, ist fast immer der Fehler — nicht die Route.
3. Ruft das FE ueberhaupt? `Promise.resolve(MOCK_*)` im Hook heisst: es gibt kein Contract, das
   verletzt sein koennte.
4. Existiert eine echte Datenquelle? Wenn der MSW-Mock Werte *ausrechnet* (Naeherungen, feste Listen,
   abgeleitete Prozentsaetze), ist die Route nicht „fehlend", sondern **noch nicht entworfen**.

Warum es so lange unbemerkt bleibt: der MSW-Mock antwortet auf den *falschen* Pfad genauso brav wie
auf den richtigen. In Demo und in jedem Test ist alles gruen; erst gegen ein echtes Gateway kommt
der 404. Deshalb gehoert der Mock-Pfad an die Wire-Wahrheit angeglichen, nicht an die FE-Erwartung.

## CI Desktop „Run ESLint" rot, aber `eslint .` zeigt Hunderte Probleme (2026-06-21)
- CI fuehrt `npm run lint` = **`eslint src/`** (nur `desktop/src/`), NICHT `eslint .`. Der `desktop/design-reference/`-Ordner ist Grundrauschen (~100 no-unused-vars/no-explicit-any) und **zaehlt nicht** fuer CI.
- **Diagnose:** `npx eslint src/ -f json` aggregieren statt `eslint .`. Beim letzten Vorfall blockten genau **2** Fehler main (`unused-imports/no-unused-imports` + `react-hooks/refs` = ref-Mutation im Render).
- **Regel:** Vor Push aus `desktop/`: `npm run lint` (exakt der CI-Befehl). „tsc gruen" ≠ „lint gruen".

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
- **Fix:** `tenantID, err := middleware.GetTenantID(ctx)` in jedem gRPC-Handler. Proto-Feld bleibt im Wire-Format, wird aber serverseitig ignoriert oder hoechstens fuer Logging genutzt. Welle 3.5 hat das Pattern auf 14+ Methoden in chat/crm/work/video/dialer-gRPC umgestellt. **Nachzuegler-Sweep 2026-06-05 (`a02f3632`):** 4 Dialer-Leftovers (ListCampaigns/SetAgentStatus/ListCallOutcomes/CreateCallOutcome) gaben "missing or invalid tenant_id" 400 weil das Gateway die Felder nie setzte.
- **Test:** Tenant-Isolation-Tests muessen einen Two-Tenant-Scenario abdecken (User Tenant A schickt Request mit `tenant_id=B` im Body — Backend muss `tenant_id=A` aus dem Context durchsetzen).

### Phantom-404: GetByID filtert tenant_id, scannt sie aber nicht (E2E-Modernisierung, 2026-06-05)
- **Symptom:** GET auf eine Ressource funktioniert, UPDATE/DELETE derselben Ressource liefert "not found" — obwohl Row existiert und Tenant stimmt.
- **Ursache:** Repo-`GetByID` hat `WHERE id=$1 AND tenant_id=$2`, nimmt `tenant_id` aber nicht in SELECT+Scan auf → `model.TenantID = uuid.Nil`. Das folgende `UPDATE ... WHERE id=$X AND tenant_id=model.TenantID` matcht 0 Rows → `ErrNotFound`.
- **War in:** `work/task.GetByID` (Task-Update 404) + `dialer.GetSessionByID` (LogCallOutcome 404). Systematischer Sweep aller Repos = Followup F2 in `docs/e2e-modernization-followups.md`.
- **Fix:** tenant_id in SELECT+Scan aufnehmen ODER im Service nach dem Load `model.TenantID = tenantID` setzen (durch den gefilterten Load bereits verifiziert).

### gRPC-Service ohne TenantInboundUnaryInterceptor / Bridge ohne Outbound (2026-06-05)
- **Symptom 1:** Jeder tenant-scoped RPC eines Services gibt "missing tenant context" — `cmd/<service>/main.go` hat den `middleware.TenantInboundUnaryInterceptor()` nicht registriert (war: document).
- **Symptom 2:** Service-zu-Service-Call (z.B. Dialer→CRM-Bridge) gibt Unauthenticated — die direkte gRPC-Connection hat keinen `TenantOutboundUnaryInterceptor` als DialOption (Gateway-Connections sind global abgedeckt, direkte Bruecken NICHT).
- **Symptom 3:** FK-Verletzung auf `created_by`/Membership-Check schlaegt immer fehl — Handler uebergibt `uuid.Nil` als Actor statt `middleware.GetUserID(ctx)` (x-user-id-Metadata) zu lesen (war: work DeleteTask, dialer CreateCampaign).
- **Checkliste neuer Service:** Inbound-Interceptor in ChainUnaryInterceptor · jede ausgehende Service-Bridge mit Outbound-Interceptor · Tenant aus ctx, Actor aus ctx.

### RequirePermission-Guard ohne Permission-Seed → 403 fuer ALLE inkl. Admin (Migration 129, 2026-06-05)
- **Symptom:** Komplette Modul-Routen geben 403 "insufficient permissions" fuer jeden User — auch Admin. Monatelang unbemerkt, wenn das Modul nie end-to-end getestet wird.
- **Ursache:** `RequirePermission("documents", "read")` & Co. referenzierten Permissions, die keine Migration je in die `permissions`-Tabelle geschrieben hat. Admin bekommt neue Permissions NICHT automatisch (Migration-000002-CROSS-JOIN galt nur fuer den damaligen Bestand). 35 Strings betroffen (documents/email/finance/formulare/helpdesk/hr/inbox/wiki/automations/search/settings/recording).
- **Fix + Praevention:** Migration 129 (admin-only). Diff-Check Code-vs-DB vor Modul-Launches — Kommando in [[security]] RBAC + Memory `feedback_permission_seed_check.md`. Permissions sind JWT-Snapshot → nach Seed Re-Login noetig.

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

## nano + Shell-Heredoc-Verwechslung beim Env-Edit (2026-05-09)

- **Symptom:** Nach `sudo nano /opt/kmuhub/.env.production` ist die Variable scheinbar gesetzt, aber Docker liest sie nicht. Eine `.env.production.save`-Datei mit aktuellem Timestamp existiert. Manchmal endet die `.save`-Datei mit einer literalen Zeile `EOF`.
- **Ursache:** Eine Shell-Anleitung mit `sudo tee -a file <<EOF` ... `EOF` wurde komplett in den nano-Buffer reingepasted, statt in die Shell. Nano interpretiert den Block als Datei-Inhalt — die `tee -a`-Zeile, der Variablen-Wert, und die `EOF`-Markierung landen alle als Text. Beim Schliessen mit `Ctrl+C` oder Session-Abbruch (statt `Ctrl+O`) speichert nano nicht in die Original-Datei sondern legt `.env.production.save` als Crash-Recovery-Backup an. Die echte `.env.production` bleibt unveraendert.
- **Diagnose:**
  ```bash
  sudo grep -c '^TARGET_VAR=' /opt/kmuhub/.env.production       # erwartet: 1, ist: 0
  sudo ls -la /opt/kmuhub/.env.production.save                  # falls vorhanden → nano-Crash
  sudo tail -3 /opt/kmuhub/.env.production.save                 # literal EOF am Ende → Heredoc-Paste-Bug
  ```
- **Recovery (idempotent):**
  ```bash
  sudo cp /opt/kmuhub/.env.production /opt/kmuhub/.env.production.bak.$(date +%s)
  sudo grep '^TARGET_VAR=' /opt/kmuhub/.env.production.save | sudo tee -a /opt/kmuhub/.env.production > /dev/null
  sudo rm /opt/kmuhub/.env.production.save   # enthaelt potentiell sensitive Werte
  ```
- **Praevention:** Heredoc-Bloecke (`<<EOF ... EOF`) gehoeren in die Shell, NIEMALS in einen Editor. Wenn nano gewollt ist: nur den `KEY=value`-String selber tippen, keinen `tee`-Befehl drumrum.

## render-configs.sh schreibt literal `${OLD_VAR}` weil Server-Code-Pull fehlt (2026-05-09)

- **Symptom:** Nach Variablen-Refactor (z.B. `SLACK_WEBHOOK_URL` → `ALERT_WEBHOOK_URL`) zeigt die gerenderte Config-Datei (`alertmanager.yml`) noch literal `slack_api_url: "${SLACK_WEBHOOK_URL}"`. envsubst hat nichts ersetzt obwohl die neue Variable in `.env.production` steht.
- **Ursache:** Der Code-Refactor wurde in main commited, aber der Production-Server hat den Pull noch nicht gemacht. `render-configs.sh` und `alertmanager.yml.tmpl` auf dem Server sind noch die alte Version mit `${OLD_VAR}`. envsubst's Whitelist-Mode kennt nur die alte Variable, die aber nicht mehr in `.env.production` definiert ist → leerer String, oder das literale `${OLD_VAR}` bleibt durchgereicht.
- **Diagnose:**
  ```bash
  sudo grep NEW_VAR /opt/kmuhub/deploy/scripts/render-configs.sh        # 0 Treffer = Pull fehlt
  sudo ls /opt/kmuhub/deploy/docker/<config>.yml.tmpl                   # File-not-found = Pull fehlt
  ```
- **Fix:** `git pull` machen, dann render-configs.sh erneut laufen lassen:
  ```bash
  cd /opt/kmuhub
  sudo GIT_SSH_COMMAND='ssh -i /home/deploy/.ssh/github_deploy' git pull origin main
  sudo bash -c 'set -a && source .env.production && deploy/scripts/render-configs.sh'
  ```
- **Praevention:** Bei Variablen-Refactor in `render-configs.sh` oder Template-Files immer mit `git pull origin main` auf dem Server starten, NICHT direkt mit `render-configs.sh`. Alternative: voller `deploy.sh`-Run (zieht git+migrate+build+restart in einem Schritt).

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

## Merge-/CI-Lessons FE-Lane-Merge (2026-06-11 Nacht)

- **CI besteht aus ZWEI Workflows: `CI` und `CI Desktop`** — `gh run watch <ein-run>` reicht nach einem Push NICHT. Nach dem FE-Merge war CI grün, CI Desktop aber rot (1 eslint-Error), und **CD wird bei rotem CI Desktop geskippt** → Deploy blieb unbemerkt aus. Regel: nach jedem Push `gh run list --commit <sha>` und ALLE Runs prüfen.
- **`npm test` im desktop/ = `vitest` im WATCH-Mode** — hängt endlos (kein Exit). Für Gates immer `npx vitest run`.
- **Explore-Agents + `git grep <branch>` = Branch-SNAPSHOT, nicht Branch-DIFF** — der Treffer kann unverändertes Merge-Base-Erbe sein. Vor Sweep-Planung mit `git diff base...branch -- <datei>` verifizieren, sonst plant man Fixes für Dateien, die der Merge ohnehin von main übernimmt (so geschehen: ContractType-Sweep + Presign-Umstellung waren obsolet).
- **Merge-Halbschatten bei Store-Mocks:** main-Tests, die zustandsbasierte Stores mocken, brechen still, wenn die Lane neue Selektoren ergänzt (`scope` undefined → Bedingung false → UI-Teil fehlt). Nach Merges Vitest IMMER laufen lassen, Mock-Stores um neue Felder ergänzen.
- **tsc ist repo-weit wieder 0 Fehler** (Stand 2026-06-11 Nacht) — die ~98 typed-i18n-Altfehler sind beseitigt; Full-`npx tsc --noEmit` taugt wieder als Gate (Exit-Code direkt prüfen, nie durch Pipes).

## Multi-Agent-Orchestrierung — Failure-Modes (FE-Wiring Welle 2, 2026-06-18)

Bei parallelen Sonnet-Subagenten (ein Modul je Agent) traten vier Failure-Modes auf, die alle erst durch unabhängige Verifikation auffielen:

- **"Nicht pushen"-Anweisung wird ignoriert.** 2 von 3 Subagenten committeten + pushten trotz expliziter Prompt-Anweisung direkt auf `main` → CI Desktop rot + CD deployte Stub nach Prod. Eine Prompt-Zeile ist KEINE Enforcement-Grenze. **Regel:** Subagenten mit `isolation: "worktree"` starten (physische Trennung von main); die **Main-Session macht ALLE Commits/Pushes**. Vor jedem Commit: `git worktree list`, `git log origin/main`, Dateiliste vs. Report, `TaskStop` laufender Agenten.
- **"Build/Test grün" ≠ architektonisch korrekt.** Ein Agent rief im Gateway den Service **direkt in-process** auf (eigener DB-Pool) statt über den gRPC-Client → umgeht den Tenant-RLS-Kontext-Hook (`PrepareConn` setzt `app.tenant_id` aus gRPC-Metadata) → in Prod **leere Tabellen / Phantom-404**. Tests waren grün, weil sie Mocks statt echter DB/RLS nutzen. **Regel:** Gateway-Routes MÜSSEN über `sr.getClient()`/gRPC laufen, nie über einen im Gateway instanziierten Service (Architektur-Regel "Thick Services, Thin Handlers" + RLS). Nach Proto-Edit IMMER per-Modul regenerieren und `.pb.go`-Diff prüfen — sonst fließen neue Felder nie durch gRPC.
- **"Voll end-to-end" gemeldet, aber Stubs geliefert.** gRPC-Handler waren `codes.Unimplemented`; MSW kaschierte es im Demo. **Regel:** Claim per `grep Unimplemented` + Lesen der Handler gegenprüfen.
- **QA-Assert grün bei gecrashter Seite.** Die store→Hook-Migration verletzte die Rules-of-Hooks (Hook nach early-return) → React-ErrorBoundary fing den Crash ab → kein `pageerror`, keine Raw-Keys → QA-Script meldete „grün", obwohl die Seite tot war. Außerdem: QA-Script mischte puppeteer-Import + Playwright-API → wäre nie gelaufen. **Regel:** Screenshots IMMER wirklich ansehen; im QA-Script auf ErrorBoundary-Text prüfen (z.B. „Etwas ist schiefgelaufen" / „Rendered more hooks"). Playwright-Boilerplate aus `desktop/scripts/qa-welle0-absences.mjs` (electronAPI-Stub + `cosmi-ui`-Onboarding) wiederverwenden.

## Verwandte Notes
- [[architektur]] — Architektur-Regeln
- [[i18n]] — i18n-Architektur & Konventionen
- [[deployment]] — Docker & CI/CD
- [[stack]] — Dev-Tooling & Pfade
