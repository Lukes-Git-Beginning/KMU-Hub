# Backend-Nachtloop — Journal

Append-only. Jede Iteration haengt genau einen Block an. Von unten lesen.

Format:

```markdown
## Iteration <n> — <unit-id> — <done|blocked> — <YYYY-MM-DD HH:MM>
- commit: <sha oder ->
- gebaut: <ein bis drei Zeilen, was real existiert>
- gate: build ok | vet ok | lint ok | test ok | migration ok | rls-smoke ok|n.a.
- verify vorgaenger: <sauber | Befund + angelegte Fix-Unit>
- offen: <was Luke morgens pruefen muss>
```

---

## Iteration 0 — Harness aufgesetzt — 2026-07-26

- commit: -
- gebaut: Loop-Harness (Guard-Hook + Regressionstest, ITERATION.md, BACKLOG.yml mit
  22 Phase-3-Units, run-loop.ps1, GATE-COMMANDS.md).
- gate: Guard-Regressionstest 35/35 gruen. Lokale DB auf Migrationskopf 243
  (= Repo-Kopf). RLS-Smoke gegen `contacts` verifiziert: eigener Tenant 12 Zeilen,
  fremder Tenant 0.
- verify vorgaenger: n.a. (erster Eintrag)
- offen: Trockenlauf ueber zwei Iterationen unter Aufsicht, bevor ein Nachtlauf startet.

## Iteration 1 — p3-einkauf-total-amount — done — 2026-07-26

- commit: e91cdf2a
- gebaut: Repository-Methode `RecomputePOTotal` (SQL-Aggregat SUM(quantity*unit_price)
  in purchase_orders.total_amount, tenant-gescoped) + Aufruf aus AddPOLine/UpdatePOLine/
  DeletePOLine im Service. CreatePO bleibt bei "0" (korrekt, da 0 Zeilen bei Anlage).
  4 neue Unit-Tests (Add/Add-mehrzeilig/Update/Delete) pruefen den total_amount-Wert
  nach der jeweiligen Mutation.
- gate: build ok | vet ok | lint ok (0 issues) | test ok | migration n.a. (keine neue
  Tabelle/Spalte) | rls-smoke n.a. (kein neuer SELECT-Pfad, bestehende Policy greift
  unveraendert)
- verify vorgaenger: n.a. (Iteration 0 war reines Harness-Setup, kein Unit-Commit)
- offen: keins. Naechste Unit ist p3-einkauf-cancel (deps erfuellt).

## Iteration 2 — p3-einkauf-cancel — done — 2026-07-26

- commit: 5901a151
- gebaut: `Service.CancelPO` (analog zu SubmitPO/ReceiveGoods) storniert eine PO nur aus
  den Status draft/submitted/sent (POStatusCancelled existierte bereits im Enum),
  alles andere liefert `ErrPONotCancellable`. Neue RPC `CancelPO` im .proto (Request
  tenant_id+po_id, Response wiederverwendet POResponse{po}) + Regen von .pb.go/_grpc.pb.go.
  gRPC-Server-Handler + Fehler-Mapping (FailedPrecondition -> 409) in einkauf_grpc.go.
  Gateway-Route `POST /pos/{id}/cancel` via bestehende Permission `einkauf:po write`
  (kein neuer Guard, kein Seed noetig) — Handler geht ueber `client.CancelPO`, keine
  direkte Service-Instanz im Gateway. 3 neue Unit-Tests (Erfolg ueber alle drei offenen
  Stati, Ablehnung ueber alle vier nicht-offenen Stati, PO-not-found).
- gate: build ok | vet ok | lint ok (0 issues) | test ok | migration n.a. (keine neue
  Tabelle/Policy, bestehende purchase_orders-Policy unveraendert) | rls-smoke n.a.
  (kein neuer SELECT-Pfad)
- verify vorgaenger: sauber (e91cdf2a gegen alle 6 Fehlerklassen geprueft — Business-Logik
  korrekt in Service/Repository, tenant-gescopte SQL, kein gRPC-Bypass, kein Stub, kein
  Proto-Touch, kein neuer Guard, keine neue Tabelle).
- NACHTRAG (Hauptsession, Trockenlauf-Review): Diese Iteration meldete "test ok", war aber in CI
  ROT. `TestOpenAPIRouteDrift` (`internal/gateway`) schlug fehl, weil die neue Route
  `POST /einkauf/pos/{id}/cancel` keinen Eintrag in `api/openapi.yaml` hatte. Das Gate lief nur
  ueber `./internal/einkauf/...`, wo dieser Test nicht liegt. Behoben in `986633e9`
  (Spec-Eintrag inkl. 409). Systemisch behoben: `go test ./internal/gateway/` ist jetzt Pflicht,
  sobald eine Route dazukommt, und "Route ohne Spec-Eintrag" ist Fehlerklasse 7 im Verify-Vorspann.
- offen: FE-Client (`desktop/src/renderer/src/api/einkauf-client.ts`) erwartet bereits
  genau diese Route/Response-Form (mock-first) — kein FE-Aenderungsbedarf. Der
  `🔒 Backend-Gap`-Kommentar im FE-Client und im MSW-Handler (`mocks/handlers/einkauf.ts`)
  kann jetzt entfernt werden, ist aber nicht Teil dieser Iteration (FE-Datei, nicht
  Backend-Scope). DB-Gate (RLS-Smoke) nicht gelaufen, da keine Tabelle/Policy angefasst —
  bewusst n.a., nicht uebersprungen.

## Trockenlauf-Abnahme (Hauptsession) — 2026-07-26

Zwei Iterationen unter Aufsicht, 7 min und 6 min, $2,41 und $3,49. Ergebnis: **abgenommen.**

Abnahme-Kriterien:
- Commits nur auf `backend-loop`, `main` durch den Loop unberuehrt.
- Guard feuert nachweislich **innerhalb** von `claude -p`: eine Sonde `git push origin main`
  wurde vom Hook blockiert, nicht nur vom Prompt abgeraten.
- Iteration 2 hat den Commit von Iteration 1 im Verify-Vorspann geprueft.
- Unabhaengige Gegenpruefung der Diffs durch die Hauptsession: Handler geht ueber
  `client.CancelPO` (kein gRPC-Bypass), `.proto` und beide `.pb.go` im selben Commit, SQL in
  beiden Ebenen tenant-gescoped, `COALESCE(...,0)` faengt den Null-Zeilen-Fall.
- CI am Draft-PR #14 gruen: Lint, Test, Validate OpenAPI. CD ist erwartungsgemaess nicht gelaufen.

Zwei echte Befunde, beide systemisch geschlossen:
1. **Route ohne Spec-Eintrag** — siehe Nachtrag zu Iteration 2. Gate erweitert, Fehlerklasse 7.
2. **Rebase erzwingt Force-Push** — `git rebase origin/main` schrieb die Branch-Historie um, der
   Push wurde als non-fast-forward abgelehnt, und Force-Push ist verboten. Der Loop merged jetzt
   `origin/main` statt zu rebasen; der Guard blockt `rebase` und laesst `merge` nur mit Ziel
   `origin/main` zu.

Ein roter Check bleibt und ist **kein** Code-Befund: `Claude Code Review` scheitert an einem Bug
in `claude-code-action@v1.0.137` ("Internal error: directory mismatch ... this indicates a bug").

## Iteration 3 — p3-einkauf-exportpo-remove — done — 2026-07-26 19:45
- commit: 33516ae1
- gebaut: ExportPO-Stub komplett entfernt (RPC, Gateway-Route
  `POST /pos/{id}/export`, gRPC-Handler, Service-Methode, Proto-Messages,
  openapi.yaml-Eintrag). Vorher lieferte der Endpoint immer nur einen leeren
  Octet-Stream-Payload zurueck.
- verifiziert vor dem Loeschen: `grep -rniE "exportpo"` ueber `backend/` und
  `desktop/` zeigt keinen verbleibenden Aufrufer. Der FE-Client
  (`einkauf-client.ts:143`) hat schon einen Kommentar, der bestaetigt, dass
  Exporte seit 2026-07-16 clientseitig laufen (`einkauf-export.ts`); der
  Kommentar selbst wurde nicht angefasst (FE-Datei, ausserhalb Scope dieser
  Unit).
- proto regeneriert: `protoc --go_out=. --go-grpc_out=.` manuell (kein
  `proto-einkauf`-Target im Makefile fuer diesen Service), Diff in
  `einkauf.pb.go`/`einkauf_grpc.pb.go` besteht ausschliesslich aus dem
  entfernten Message-Paar und den daraus folgenden Index-Verschiebungen
  (msgTypes[31]->..., rawDesc) — keine unerwartete Reformatierung.
- gate: build ok (voller `go build ./...` lief erst nach `GOFLAGS="-p=2"` durch —
  Default-Parallelitaet riss die Maschine in ein `out of memory allocating heap
  arena map` beim Bauen von `cmd/gateway`; mit `-p=2` sauber) | vet ok
  | lint ok (`golangci-lint run ./internal/einkauf/... ./internal/gateway/...
  ./internal/server/...`, 0 issues) | test ok (`go test
  ./internal/einkauf/... ./internal/gateway/... ./internal/server/...`, inkl.
  `TestOpenAPIRouteDrift`) | openapi ok (`npx @apidevtools/swagger-cli
  validate` lokal gruen) | migration n.a. (keine Tabelle/Policy angefasst)
  | rls-smoke n.a. (kein SELECT-Pfad veraendert, nur ein toter Pfad entfernt)
- verify vorgaenger: sauber (`ef71800e` war bereits durch die Hauptsession im
  vorigen Trockenlauf-Review gegengeprueft, siehe NACHTRAG oben — keine
  weiteren unverifizierten Commits dazwischen ausser harness-internen
  `docs(planning)`/`fix(planning)`-Commits).
- offen: Wenn Luke will, kann der stale FE-Kommentar in `einkauf-client.ts`
  ("the backend ExportPO endpoint is a stub") jetzt praezisiert werden
  (Endpoint existiert nicht mehr, nicht nur "ist ein Stub") — kosmetisch,
  keine Funktionsaenderung noetig. `GOFLAGS="-p=2"` als Build-Workaround: falls
  der OOM bei vollem `go build ./...` systematisch auftritt (nicht nur diese
  Maschine/dieser Moment), gehoert das als generelle Anmerkung in die
  Loop-Betriebsnotizen, nicht nur hierher.

## Iteration 4 — p3-document-wire-shape — done — 2026-07-26 19:59

- commit: f9aa3752
- verify vorgaenger: sauber (33516ae1 unabhaengig gegengeprueft — Grep ueber
  `desktop/` und `backend/` bestaetigt keinen Aufrufer der geloeschten Route;
  `EinkaufDetailModals.tsx:handleExportPdf` baut das PDF nachweislich
  clientseitig via `buildPOPdf`, ruft nie den Client-Kommentar-referenzierten
  Backend-Endpoint auf).
- Bestandsaufnahme vor dem Bauen: Der BACKLOG-Scope-Text war teilweise stale.
  `eaf0c79f` (vor dieser Iteration) hatte "Alle Create-Responses gewrappt"
  bereits erledigt (Share/Tag/Link-POST liefern `{share}`/`{tag}`/`{link}`,
  Revert liefert bereits `{version}`). Reales Restproblem war die
  bare-Array-vs-wrapped-Inkonsistenz bei sechs List-Endpoints und ein
  Null-vs-Empty-Array-Defekt an vier weiteren Stellen — beides durch
  gegenlesen von `document-client.ts` (dort explizit als Workaround
  dokumentiert: `unwrapList`/`listTotal`-Normalizer) und den FE-Typen in
  `document-types.ts` verifiziert, nicht angenommen.
- gebaut: `response.ProtoListWrapped[T]` (neuer Helper neben `ProtoList`,
  gleiche protojson-Kodierung pro Item, aber unter einem Key gewrappt plus
  optionale Zusatzfelder). Sechs Handler in `route_document.go` umgestellt:
  `HandleListFolders` -> `{folders, total}` (total = len, Proto hat kein
  eigenes total-Feld), `HandleGetFolderPath` -> `{segments}`,
  `HandleListFileVersions` -> `{versions}`, `HandleListFileEntityLinks` ->
  `{links}`, `HandleListShares` -> `{shares}`, `HandleListTags` -> `{tags}`.
  Alle sechs matchen jetzt exakt die FE-Typen in `document-types.ts`
  (`ListFoldersResponse`, `ListVersionsResponse`, etc.).
  Zusaetzlich `emptyIfNil[T]`-Helper gegen den Null-vs-Empty-Array-Defekt:
  `HandleListFiles` (resp.Files), `HandleListSharedWithMe` (resp.Files,
  resp.Folders), `HandleSearchFiles` (resp.Results), `HandleListVirtualFiles`
  (resp.Files) gingen bisher unveraendert an `response.JSON` — ein nil-Slice
  aus dem gRPC-Response serialisiert dort via encoding/json als `null`, nicht
  `[]`. Das ist exakt der Bug, den der FE-Client-Kommentar in
  `document-client.ts:134` fuer "files" schon dokumentiert
  (`{ files: null, total: 0 }` bei leerer Liste).
  `openapi.yaml` fuer alle sechs umgestellten Endpoints synchron nachgezogen
  (Schema von bare array auf `{key: [...]}` gewrappt).
  2 neue Unit-Tests in `response_proto_test.go`
  (`TestProtoListWrapped_EmptySliceIsEmptyArray`,
  `TestProtoListWrapped_ExtraFieldsAndItems`) decken den neuen Helper direkt
  ab, inkl. Nil-Slice-Regressionstest.
- bewusst NICHT angefasst:
  1. `GetWOPIDiscovery` (bare-Array-Response) — kein FE-Aufrufer im ganzen
     Repo (nur ein auto-generierter OpenAPI-TS-Typ in `api/types.ts`, kein
     `documentWopiApi`-Call dafuer). Aendern haette nur Risiko ohne Nutzen.
  2. `HandleListSharedWithMe`-Wireshape (`{files,folders,total}`) — der
     FE-Typ (`ListSharesResponse{shares}`) passt semantisch nicht:
     `ListSharedWithMeResponse` im Proto liefert echte File-/Folder-Entities
     (fuer eine "durchstoebern"-Ansicht), nicht Share-Records mit
     `permission`/`shared_by`. Das waere ein RPC-Redesign (Service muesste
     Share-Records mit denormalisierten Entity-Feldern liefern), keine
     Wire-Shape-Mechanik, und passt nicht in eine Ein-Commit-Iteration.
     Zusaetzlich: `DokumentePage.tsx` zieht `useSharedWithMe(...)` aktuell nur
     in eine `_sharedData`-Variable (unused, Underscore-Konvention) — die
     "Shared with me"-Sidebar-Ansicht rendert die Daten heute gar nicht.
     Der Backend-Wert ist korrekt (matcht Proto + openapi.yaml bereits vor
     dieser Iteration); der FE-Typ ist falsch. Fix gehoert in den FE-Client
     (`ListSharesResponse` vs. einen neuen `ListSharedWithMeResponse`-Typ),
     ausserhalb des Backend-Loop-Scopes. Fuer Luke: eigene kleine FE-Unit.
  3. `/documents/files/{id}/activity` — FE-Client (`documentFileApi.
     listActivity`) ruft eine Route auf, die im Gateway gar nicht registriert
     ist. Kein Wire-Shape-Defekt, sondern ein fehlender Endpoint; laut
     FE-Typ-Kommentar ("Mock-first — backend activity log ist in
     backend-gaps.md getrackt") bereits bekannte, separate Luecke.
- gate: build ok (`GOFLAGS=-p=2 go build ./...`) | vet ok | lint ok
  (`golangci-lint run ./internal/gateway/... ./internal/server/...`,
  0 issues) | test ok (`go test ./internal/gateway/... ./internal/server/...`,
  inkl. `TestOpenAPIRouteDrift` 656 Routen vs. 711 dokumentierte Pfade) |
  openapi ok (`npx @apidevtools/swagger-cli validate` lokal gruen) |
  migration n.a. (keine Tabelle/Policy angefasst) | rls-smoke n.a. (kein
  neuer SELECT-Pfad, nur Serialisierung der bestehenden Antworten geaendert)
- offen: Fuer Luke — (1) `HandleListSharedWithMe` FE-Typ-Fix (siehe oben,
  Punkt 2) ist eine schnelle FE-Unit, kein Backend-Scope; (2) fehlender
  `/documents/files/{id}/activity`-Endpoint ist eine echte Feature-Luecke,
  aktuell nicht im BACKLOG.yml als eigene Unit erfasst — ggf. nachtragen,
  falls die Activity-Timeline im Dokumente-Modul Prioritaet bekommt.

## Iteration 5 — p3-helpdesk-wire-ticket — done — 2026-07-26 20:12

- commit: - (keine Code-Aenderung, nur BACKLOG.yml/JOURNAL.md)
- verify vorgaenger: sauber. `f9aa3752` (document wire-shape) gegen alle
  sieben Klassen geprueft: keine gRPC-Umgehung, kein Proto-Touch, keine neue
  Tabelle/Guard/Route — reine Response-Serialisierung, Diff bestaetigt es.
- gebaut: nichts — Unit war bereits vollstaendig umgesetzt, lange bevor
  backend-gaps.md/BACKLOG.yml geschrieben wurden. Commit `d2473afb` (2026-06-28,
  "feat(helpdesk): denormalize assignee/requester names, add
  description/category/ticket_number") deckt beide Units
  `p3-helpdesk-wire-ticket` UND `p3-helpdesk-ticket-number` vollstaendig ab:
  - `Ticket`-Proto traegt `assignee_name` (optional), `requester_name`,
    `description`, `category`, `ticket_number` (helpdesk.proto:86-92),
    .pb.go im selben Commit regeneriert.
  - `ticketSelectColumns` in postgres_repository.go joint per `LEFT JOIN
    users` auf assignee_id/requester_id, COALESCEd auf Vor-/Nachname dann
    E-Mail; nicht aufloesbare Referenz (NULL-Assignee oder RLS-gefilterter
    Fremd-User) liefert leeren/optionalen Namen statt Fehler — kein Crash-Pfad.
  - `assignee_id` ist DB-seitig UUID-typisiert (000077) und am Gateway
    `validate:"omitempty,uuid"` — die im Backlog vermerkte "Vorbestand"-Sorge
    (Freitext-Namen als assignee_id) ist mit dem heutigen Code-Stand nicht
    mehr zutreffend.
  - Migration `000236_helpdesk_ticket_fields`: neue Spalten +
    `helpdesk_ticket_counters` (RLS aktiv), Bestandstickets per
    `ROW_NUMBER() OVER (PARTITION BY tenant_id ...)` rueckwirkend
    nummeriert, Counter korrekt geseeded, up/down beide gefuellt.
    `CreateTicket` alloziert die naechste Nummer atomar per
    `INSERT ... ON CONFLICT DO UPDATE ... RETURNING next_number - 1`.
  - Lokaler Migrationskopf ist 243 = Repo-Kopf, 000236 ist also seit langem
    angewendet.
  Habe beide Units in BACKLOG.yml auf `done` gesetzt und die Begruendung dort
  hinterlegt, statt sie stumm abgearbeitet zu lassen — sonst haette eine
  spaetere Iteration hier nochmal recherchiert und nichts zu bauen gefunden.
- gate: build ok (`go build -p 2 ./internal/helpdesk/... ./internal/gateway/...
  ./cmd/helpdesk/... ./cmd/gateway/...`) | vet ok | lint ok (0 issues,
  `./internal/helpdesk/... ./internal/gateway/...`) | test ok
  (`go test ./internal/helpdesk/...`, `go test ./internal/gateway/...`) |
  migration ok (Kopf 243 = Repo-Kopf, 000236 bereits angewendet) | rls-smoke
  ok — manuell nachgezogen (siehe unten), da die vorhandenen DB-Tests
  (`TestTenantIsolation_Helpdesk`, `TestTenantIsolation_TicketMessages_DB`)
  lokal grundsaetzlich nicht aussagekraeftig sind (siehe offen).
- rls-smoke Detail (psql, `kmuhub` → `SET ROLE kmuhub_app`, Testzeile
  eingefuegt und wieder geloescht): eigener Tenant
  (`00000000-0000-0000-0000-000000000001`) → 1 Zeile, fremder Tenant
  (`...ff`) → 0 Zeilen. Bestanden.
- offen: **Lokale DB-Tests mit RLS-Anspruch sind aktuell blind.**
  `deploy/docker/.env` setzt `DATABASE_URL` UND `MIGRATION_DATABASE_URL` auf
  die Superuser-Rolle `kmuhub` — Postgres-Superuser umgehen RLS immer,
  unabhaengig von `FORCE ROW LEVEL SECURITY`. Dadurch liefern
  `TestTenantIsolation_Helpdesk` und `TestTenantIsolation_TicketMessages_DB`
  (und vermutlich alle `*_DB`-Tests in anderen Paketen, die `testutil.
  PoolFromEnv` nutzen) lokal fuer JEDE Tabelle "expected 0, got 1" — nicht
  weil RLS kaputt ist, sondern weil der Testlauf nie mit `kmuhub_app`
  verbindet. In CI ist das korrekt verdrahtet (`ci.yml` setzt dort
  `kmuhub_app:app_dev@.../kmuhub_test`), daher faellt es dort nicht auf.
  Habe versucht, lokal explizit `DATABASE_URL=postgres://kmuhub_app:app_dev@
  localhost:5432/kmuhub` zu setzen — Passwort `app_dev` (aus Migration
  000121-Kommentar "Lokal-Dev nutzt 'app_dev'") schlaegt fehl
  (`password authentication failed`), das lokale Compose-Postgres hat
  offenbar ein abweichendes Passwort fuer `kmuhub_app`. Fuer Luke: entweder
  `ALTER ROLE kmuhub_app PASSWORD 'app_dev'` lokal nachziehen, oder das
  tatsaechliche lokale Passwort in GATE-COMMANDS.md dokumentieren — sonst
  bleibt jede zukuenftige `*_DB`-Testverifikation in diesem Loop auf den
  manuellen psql-`SET ROLE`-Umweg angewiesen.
