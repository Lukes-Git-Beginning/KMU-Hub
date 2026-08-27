# Backend-Nachtloop — Journal Lauf 13

Lauf 13 lief in der Nacht auf den 2026-08-27 auf `backend-loop` und **brach nach 21 von
130 Iterationen ab**, rund 7,4 Stunden vor Fensterende. Ursache war kein Codefehler:
Iteration 21 schrieb in `BACKLOG.yml` ein `done_when`-Element, das mit einem Backtick
beginnt — YAML verbietet den Backtick als erstes Zeichen eines Plain-Skalars. Der
darauffolgende `backlog-check.py --state`-Aufruf des Treibers scheiterte, der Lauf endete.
Behoben in `c13589b3`, gegen Wiederholung abgesichert in `hooks/loop-guard.sh`.

Zweiter Befund aus diesem Lauf, ebenfalls ohne Codeanteil: die 70 DRIFT-Zeilen im `run.log`
sind **allesamt Fehlalarme**. Der Detektor las die hoechste `## Iteration N` ueber die ganze
append-only Datei, fand 43 aus Lauf 12 und verglich sie gegen 1–21. Die Selbstueberwachung
war den ganzen Lauf blind. Behoben, indem der Detektor nur noch Zeilen unterhalb des
Dateistands vom Laufbeginn auswertet.

---

## Iteration 1 — cov-email-sync-worker-and-imap-client — done — 2026-08-27 01:01
- commit: fcd73c8b
- gebaut: `internal/email/sync/worker.go` (`syncFolders`/`syncFolder` von konkretem `*IMAPClient`
  auf schmale Interfaces `folderLister`/`folderFetcher` verengt, `initialBackoff`/`maxBackoff`
  von `const` auf `var` — Muster von `imapDialTimeout`/`imapHandshakeDeadline` in imap_client.go
  uebernommen, "vars nur damit Tests sie schrumpfen koennen, Produktion weist nie neu zu").
  Drei neue Testdateien: `worker_sync_test.go` (syncFolders/syncFolder gegen Fakes:
  UIDVALIDITY-Wechsel wischt lokale Nachrichten, Erstsync filtert per 30-Tage-Cutoff, Delta-Sync
  filtert NICHT, ein fehlschlagender CreateSynced-Aufruf mitten im Batch bricht den Rest nicht ab,
  bereits getrackte Ordner werden nicht doppelt angelegt, ein fehlschlagendes CreateFolder wird
  geloggt statt den Lauf abzubrechen), `worker_run_test.go` (`newWorker` setzt alle Felder;
  `Worker.Run` versucht nach einem fehlgeschlagenen `syncCycle` erneut, mit Backoff, und reagiert
  auf Context-Cancel auch WAEHREND des Backoff-Waits, nicht erst beim naechsten Schleifendurchlauf;
  `GetDecryptedCredentials` schlaegt ueber einen Fake-`account.Repository` sofort fehl, kein
  Netzzugriff noetig), `engine_lifecycle_test.go` (`Engine.Start`/`StartWorker`/
  `startWorkerInternal`: ein Worker pro aktivem Account, `ListAllActive`-Fehler startet keine
  Worker, unbekannte Account-ID liefert `ErrSyncInProgress`, ein erneuter `StartWorker`-Aufruf fuer
  dieselbe ID stoppt den alten Worker zuerst). `account.Service` laesst sich vollstaendig ueber
  seine bestehenden `Repository`/`VaultEncryptor`-Interfaces faken — kein Produktionscode dafuer
  noetig ausser der oben genannten Interface-Verengung in worker.go.
  Tenant-Zuordnung (done_when-Punkt): kommt aus `w.account.TenantID` (Worker haelt das geladene
  `*models.EmailAccount`, das bereits tenant-gescoped aus `accountService.ListAllActive`/
  `GetByID` kommt) und wird in `envelopeToMessage` auf jede synchronisierte Nachricht geschrieben
  (`TestEnvelopeToMessage` bestand das schon vor dieser Iteration).
  IMAP-Client (done_when-Punkt "Abdeckung vorhanden oder eigene Unit"): der Client wrapt
  `imapclient.Client` konkret, kein Interface — Erfolgspfade brauchen einen antwortenden
  Fake-IMAP-Server (Wire-Protokoll), nicht nur einen TCP-Listener. Als eigene Unit angelegt
  (siehe neue-units), Fehlerpfade (nil-Client, Handshake-Timeout, sofortiger Verbindungsabbruch)
  waren schon vor dieser Iteration abgedeckt.
- gate: build ok (`go build -p 2 ./internal/email/... ./internal/gateway/... ./cmd/gateway/...`)
  | vet ok (`go vet ./internal/email/...`) | lint ok (`golangci-lint run ./internal/email/...`,
  0 issues) | test ok (DATABASE_URL gesetzt, `go test -count=1 ./internal/email/...` — 12
  Unterpakete, alle `ok`, 0 uebersprungen) | migration n.a. (keine) | rls-smoke n.a. (keine neue
  Tabelle/Policy) | gateway-test n.a. (keine Route angefasst, Build trotzdem gruen)
- coverage: internal/email/sync 34,6 % -> 64,6 % (selbst gemessen vor/nach mit
  `go tool cover -func`; deckt sich mit `coverage_start` der Unit, CI-Stand 32949396303 war
  noch nicht ueberholt). Funktions-Feinbild: `syncFolders` 16,7 % -> 100 %, `syncFolder`
  8,3 % -> 97,2 %, `newWorker` 0 % -> 100 %, `Run` 0 % -> 83,3 %, `StartWorker` 0 % -> 95,5 %,
  `startWorkerInternal` 0 % -> 100 %; `syncCycle` 0 % -> 15,0 %, `idleLoop`/`pollLoop` bleiben
  0 % (haengen an einem echten IMAP-Client-Erfolgspfad, siehe neue-units).
- mutations-probe: Cutoff-Filter in `syncFolder` testweise mit `if false && highestUID == 0 &&
  envDate.Before(cutoff)` deaktiviert -> `TestSyncFolder_InitialSync_FiltersMessagesOlderThanCutoff`
  wurde rot ("[0xc000250000 0xc0002501c0] should have 1 item(s), but has 2" — die 60 Tage alte
  Nachricht wurde nicht mehr herausgefiltert) -> zurueckgedreht, `git diff --stat` zeigt danach nur
  noch die beabsichtigte Interface-Verengung (+21/-3 in worker.go), Test wieder gruen.
- verify vorgaenger: sauber (`4107e6b8`, letzter Bau-Commit vor dieser Iteration, geprueft gegen
  alle acht Fehlerklassen — reine Testdatei gegen dreizehn bereits bestehende, ungetestete
  Gateway-Handler, kein neuer Endpunkt, kein `.proto`, kein `RequirePermission`, kein
  gRPC-Bypass, kein Stub-Marker). Der Backlog-Kopf-Hinweis aus Iteration 43 zur vorbestehenden
  `blocked`-Unit `fix-409-double-meaning-on-grpc-conflict-routes` betraf BACKLOG-NEXT.yml, nicht
  diese Datei — beim Preflight-Lauf dieser Iteration (`hooks/backlog-check.py --preflight`) kam
  "alle drei Dateien valide, keine Verstoesse", also bereits vor dieser Iteration erledigt.
- neue-units: cov-imap-client-fake-protocol-server (Erfolgs- und NO/BAD-Fehlerpfade fuer
  Login/SelectFolder/FetchHeaders/FetchBody/SetFlags/Idle/HasIDLE/Noop/ListFolders brauchen
  einen antwortenden Fake-IMAP-Server, kein reiner TCP-Listener)
- offen: keine

## Iteration 2 — fix-websocket-presence-subscribe-missing-tenant-check — done — 2026-08-27 01:13
- commit: ad060b80ce7441dc05779e4d7d22c7a39eafbc5d
- gebaut: `WebSocketHub.handlePresenceSubscribe` (`backend/internal/server/websocket.go`)
  prueft jetzt fuer jede `target_user_id`, ob sie zum Tenant des abonnierenden Users gehoert,
  bevor sie in `presenceSubscribers` landet. Neues Feld `userTenants map[string]string`
  (userID -> tenantID) wird in `registerUserTenant` beim Connect aus `claims.TenantID` gesetzt
  (`HandleWebSocket`, direkt nach `registerConnection`) und in `unregisterConnection` geloescht,
  sobald der User keine Verbindung mehr hat (gleiches Muster wie das bestehende
  `rateLimiters`-Cleanup). Fuer jede nicht-eigene `targetUserID` baut die neue Methode
  `tenantAllowsPresenceTarget` einen mit `middleware.TenantIDKey` auf den Anrufer-Tenant
  gescopten Kontext und ruft darin `h.userInfoFunc` auf — denselben gRPC-Client-Callback
  (`authClient.GetUser`), der schon fuer die Namensauflösung existiert. Die RLS-Policy
  `user_isolation` auf `users` (`tenant_id = current_tenant_id() OR id = current_user_id()`,
  Migration 000120) liefert fuer eine fremde `tenant_id` "not found", was hier als Ablehnung
  gewertet wird — kein neuer DB-Zugriff, keine neue RPC, reine Wiederverwendung des
  bestehenden gRPC-Pfads mit korrekt gesetztem Tenant-Kontext. Selbst-Abo (`targetUserID ==
  userID`) ist immer erlaubt, unabhaengig vom Tenant-Cache. Fehlt der Anrufer-Tenant im Cache
  oder ist `userInfoFunc` nil, wird fail-closed abgelehnt (kein automatisches Zulassen).
  Neue Testdatei `websocket_presence_subscribe_test.go` (7 Tests): Same-Tenant-Ziel erlaubt,
  Cross-Tenant-Ziel abgelehnt, gemischter Batch behaelt nur das Same-Tenant-Ziel,
  Selbst-Abo immer erlaubt (auch ohne Tenant-Cache-Eintrag), unbekannter Anrufer-Tenant lehnt
  Fremd-Ziele ab, `userInfoFunc == nil` lehnt Fremd-Ziele ab, und `unregisterConnection` loescht
  den gecachten Tenant nach der letzten Verbindung (danach wird ein vorher erlaubtes Ziel
  wieder abgelehnt).
- gate: build ok (`go build -p 2 ./internal/server/... ./internal/gateway/... ./cmd/gateway/...`)
  | vet ok (`go vet ./internal/server/...`) | lint ok (`golangci-lint run ./internal/server/...`,
  0 issues) | test ok (DATABASE_URL gesetzt, `go test -count=1 ./internal/server/` — 1905 PASS,
  0 SKIP, 71,4 % Statement-Coverage) | gateway-test ok (`go test -count=1 ./internal/gateway/`,
  gruen — keine Route angefasst, `TestOpenAPIRouteDrift` lief trotzdem mit) | migration n.a.
  (keine neue Tabelle/Spalte) | rls-smoke n.a. (keine neue Tabelle/Policy, bestehende
  `user_isolation`-Policy nur gelesen, nicht geaendert)
- coverage: n.a. (Verhaltens-Fix laut Unit-Definition, kein Coverage-Ziel; Gesamtpaket
  `internal/server` zur Referenz 71,3 % (CI 32949396303) -> 71,4 % nach dieser Iteration,
  selbst gemessen mit `go tool cover`)
- mutations-probe: `tenantAllowsPresenceTarget` testweise auf `_ = err; return true`
  gesetzt (Tenant-Check faktisch deaktiviert) -> `TestHandlePresenceSubscribe_CrossTenantTargetIsRejected`
  und `TestHandlePresenceSubscribe_MixedBatchKeepsOnlySameTenantTargets` wurden beide rot
  ("Should be false" auf dem Cross-Tenant-Abo) -> per `cp`-Sicherungskopie zurueckgedreht,
  `git diff --stat` zeigt danach wieder nur die beabsichtigte Aenderung, alle 7 Tests wieder
  gruen.
- verify vorgaenger: sauber (`fcd73c8b`, geprueft gegen alle acht Fehlerklassen — reine
  Testdatei + Interface-Verengung `*IMAPClient` -> `folderLister`/`folderFetcher` in
  `worker.go`, kein gRPC-Bypass, kein Stub, kein `.proto` geaendert, kein `RequirePermission`,
  keine neue Tabelle, keine Route, kein Guard ersetzt).
- neue-units: fix-websocket-chat-send-message-missing-tenant-id (kein Vermutungsfund,
  sondern verifiziert: `chatv1.SendMessageRequest.tenant_id` ist ein Pflichtfeld,
  `ChatGRPCServer.SendMessage` parst es unbedingt und gibt bei leerem String
  `codes.InvalidArgument` zurueck — `WebSocketHub.handleSendMessage`,
  `WebSocketHub.handleGuestSendMessage` UND die REST-Gast-Route `route_guest.go` setzen es
  nie. Nur die authentifizierte REST-Route `route_chat.go` setzt es korrekt ueber
  `middleware.GetTenantID(r.Context())`. Damit schlaegt aktuell jede Chat-Nachricht ueber
  WebSocket UND jeder Gast-Chat fehl, nicht nur ein Tenant-Scoping-Detail)
- offen: Ob dieser Sendepfad-Ausfall produktiv unbemerkt bleibt, weil das Frontend WS nur zum
  Lesen/Broadcasten nutzt und Nachrichten ausschliesslich ueber `route_chat.go` sendet, ist
  nicht geprueft — steht als erste Pruef-Frage in der neuen Unit. Der Fund selbst (fehlendes
  `TenantId`) ist am Code verifiziert, nicht spekulativ.

## Iteration 3 — fix-fuhrpark-gps-ingest-no-vehicle-tenant-check — done — 2026-08-27 01:29
- commit: 7a8492c3e7d9bb44662a1f95a545412723c761b9
- gebaut: `Service.IngestGpsPositions` (`backend/internal/fuhrpark/service.go:858`) prueft jetzt
  vor dem Insert per `s.repo.GetVehicle(ctx, tenantID, vehicleID)`, ob `vehicleID` einem
  Fahrzeug des aufrufenden Tenants gehoert — exakt das Vorbild aus `GetVehicleHistory`
  (`service.go:884`). `GetVehicle` liefert bei fremdem Tenant `ErrVehicleNotFound`, der
  gRPC-Handler `FuhrparkGRPCServer.IngestGpsPositions` mappt das ueber das bereits bestehende
  `mapFuhrparkError` auf `codes.NotFound` — keine neue Fehlerbehandlung noetig, nur die
  fehlende Pruefung selbst. Vorher wurde ein `vehicle_id` aus fremdem Tenant klaglos
  akzeptiert und eine `gps_positions`-Zeile mit dem TenantId des Angreifers, aber
  `vehicle_id` eines fremden Fahrzeugs, eingefuegt.
  Tests: `TestService_IngestGpsPositions_ForeignTenantVehicle_Rejected` (Mock-Repository,
  `service_extended_test.go`) und `TestService_IngestGpsPositions_RejectsForeignTenantVehicle`
  (echte DB, `postgres_repository_gap_test.go`) — Letzterer haengt die Ablehnung zusaetzlich
  an eine `SELECT count(*) FROM gps_positions WHERE vehicle_id=...`-Pruefung, dass wirklich
  keine Zeile geschrieben wurde, und belegt danach im selben Test den Erfolgsfall mit dem
  eigenen Tenant. `TestService_IngestGpsPositions_Success` (bestehender Test) musste auf ein
  echtes, per `CreateVehicle` angelegtes Fahrzeug umgestellt werden, weil er vorher zwei
  freie `uuid.New()`-Werte ohne Fahrzeug-Datensatz verwendete und jetzt sonst selbst am neuen
  Check gescheitert waere.
- gate: build ok (`go build -p 2 ./internal/fuhrpark/... ./internal/gateway/... ./internal/server/... ./cmd/gateway/...`)
  | vet ok (`go vet ./internal/fuhrpark/...`) | lint ok (`golangci-lint run ./internal/fuhrpark/...`,
  0 issues) | test ok (DATABASE_URL gesetzt, `go test -count=1 ./internal/fuhrpark/...` — gruen,
  0 SKIP) | gateway-test ok (`go test -count=1 ./internal/gateway/`, gruen — keine Route
  angefasst, `TestOpenAPIRouteDrift` lief trotzdem mit) | migration n.a. (keine neue
  Tabelle/Spalte) | rls-smoke n.a. (keine neue Tabelle/Policy, `vehicles`-Tabelle nur gelesen)
- coverage: n.a. (Bugfix laut Unit-Definition, kein Coverage-Ziel; Gesamtpaket
  `internal/fuhrpark` zur Referenz 81,3 % (selbst gemessen vor der Aenderung) -> 81,3 % danach —
  neue Zeilen und neue Tests gleichen sich rechnerisch aus)
- mutations-probe: den `GetVehicle`-Fehler in `IngestGpsPositions` testweise mit `_ = err`
  verschluckt statt zurueckzugeben -> `TestService_IngestGpsPositions_ForeignTenantVehicle_Rejected`
  UND `TestService_IngestGpsPositions_RejectsForeignTenantVehicle` (DB-Test) wurden beide rot
  (Insert ging durch, `n=1` statt `0`, kein `ErrVehicleNotFound`) -> per Sicherungskopie
  zurueckgedreht, `git diff` zeigt danach wieder nur die beabsichtigte 4-Zeilen-Aenderung,
  alle Fuhrpark-Tests wieder gruen.
- verify vorgaenger: sauber (`ad060b80`, geprueft gegen alle acht Fehlerklassen — Handler geht
  weiterhin ueber `authClient.GetUser` als gRPC-Client, kein Stub, kein `.proto` geaendert,
  kein `RequirePermission`, keine neue Tabelle, keine Route, kein Guard ersetzt; `userInfoFunc`
  in `cmd/gateway/setup.go` bestaetigt als echter gRPC-Client-Aufruf).
- neue-units: keine
- offen: keine

## Iteration 4 - fix-carddav-missing-tenant-context-blocks-all-operations - done - 2026-08-27 01:36
- commit: cff259b77f84607d9c03ed11ce85e3f46c1177e0
- gebaut: Root-Cause-Fix an der gemeinsamen Wurzel statt in jeder Funktion. Neu:
  `caldav.NewCtxInjector(pool)` (`caldav_backend.go`) - loest ueber das schon vorhandene
  `resolveTenantID` den Tenant des per App-Passwort authentifizierten Users auf und stempelt
  `middleware.TenantIDKey` UND `middleware.UserIDKey` in den Context, genau die Keys, die die
  JWT-Auth-Middleware setzt. `CalDAVCtxInjector` (`internal/gateway/route_caldav.go`) gibt
  jetzt `(context.Context, error)` zurueck, `basicAuthMiddleware` reicht den Context durch und
  antwortet bei Aufloesungsfehler mit **500, nicht 401** (das Passwort war gueltig; DAV-Clients
  deaktivieren ein Konto nach wiederholten 401 - ein DB-Ausfall haette sonst alle CalDAV-User
  dauerhaft ausgesperrt). `cmd/gateway/setup.go` verdrahtet `caldavpkg.NewCtxInjector(pool)`
  statt `caldavpkg.CtxWithUser`.
  Wirkung ist breiter als die Unit annahm: der Fix repariert nicht nur die gRPC-Seite
  (`TenantOutboundUnaryInterceptor` haengt `x-tenant-id` nur an, wenn `GetTenantID(ctx)` auf dem
  CLIENT klappt - vorher nie), sondern automatisch auch jede DIREKTE Pool-Abfrage der beiden
  DAV-Backends: `database.NewPostgresPool`s PrepareConn-Hook stempelt `app.tenant_id`/
  `app.user_id` aus genau diesem Context (`postgres.go:60-84`), ohne die filtert RLS jede Zeile
  weg. Damit ist der Root Cause der Schwester-Unit
  `fix-caldav-write-and-exceptions-blocked-by-missing-tenant-ctx` mit erledigt - siehe
  `neue-units:`/`offen:`.
  Tests: `TestListAddressObjects_RealCardDAVContext_ResolvesTenant` und
  `TestGetAddressObject_RealCardDAVContext_ResolvesTenant` (bisher `_NoTenant_Returns401`) bauen
  ihren Context jetzt durch den **echten Produktions-Injector** statt von Hand und erwarten
  Erfolg; der Helfer `asIfTenantMiddlewareFixed` ist ersatzlos weg, alle vier bestehenden
  Listing-/Anonymisierungs-Tests laufen ueber denselben echten Pfad.
  Neu fuer die Tenant-Grenze (done_when #2): `TestCardDAVCtxInjector_ResolvesOnlyTheOwnTenant` -
  zweiter echter Tenant mit eigenem User, dessen App-Passwort-Context weder den geteilten
  Firmenkontakt aus Tenant A im Company-Adressbuch listet (leer) noch ihn per bekannter
  Contact-ID direkt abrufen kann (404, nicht 401 - die Zeile ist unsichtbar, nicht gesperrt).
  Gateway-Seite: `TestBasicAuthMiddleware_PassesInjectedTenantContextDownstream` (Handler sieht
  Tenant und User) und `TestBasicAuthMiddleware_ContextInjectionFailure_Returns500` (prueft
  zusaetzlich, dass **kein** `WWW-Authenticate` mitgeht).
- gate: build ok (`go build -p 2 ./internal/caldav/... ./internal/gateway/... ./cmd/gateway/...`)
  | vet ok (`go vet ./internal/caldav/... ./internal/gateway/...`) | lint ok
  (`golangci-lint run ./internal/caldav/... ./internal/gateway/...`, 0 issues) | test ok
  (DATABASE_URL gesetzt, Rolle `kmuhub_app`; `go test -count=1 ./internal/caldav/` gruen,
  **160 Tests, 0 SKIP** - verbose gezaehlt) | gateway-test ok (`go test -count=1
  ./internal/gateway/` gruen, `TestOpenAPIRouteDrift` + `TestOpenAPIRouteDriftParserSanity`
  explizit gruen; keine Route angefasst) | migration n.a. (keine Schema-Aenderung)
  | rls-smoke n.a. (keine neue Tabelle/Policy - die Aenderung fuettert die bestehenden Policies
  ueber die GUCs, das belegen die DB-Tests inkl. Fremd-Tenant-Fall)
- coverage: `internal/caldav` 68,6 % -> 68,6 % (selbst gemessen, `go tool cover -func`; Bugfix
  ohne Coverage-Ziel, neue Zeilen und neue Tests gleichen sich aus). Deckt sich exakt mit dem
  `coverage_start` der Coverage-Tabelle im Backlog-Kopf (68,6 %).
- mutations-probe: in `NewCtxInjector` die Zeile
  `ctx = context.WithValue(ctx, middleware.TenantIDKey, tenantID.String())` durch `_ = tenantID`
  ersetzt (also genau der alte Bug, aber mit gesetztem UserIDKey) -> alle DREI neuen DB-Tests
  rot: `TestListAddressObjects_RealCardDAVContext_ResolvesTenant`,
  `TestGetAddressObject_RealCardDAVContext_ResolvesTenant` und
  `TestCardDAVCtxInjector_ResolvesOnlyTheOwnTenant` (dort 401 statt 404). Per Sicherungskopie
  zurueckgedreht, `git diff --stat` zeigt danach wieder nur die beabsichtigte Aenderung,
  `go test ./internal/caldav/` wieder gruen.
- verify vorgaenger: sauber (`7a8492c3`, geprueft gegen alle acht Fehlerklassen - 4 Zeilen in
  `Service.IngestGpsPositions`, die `s.repo.GetVehicle(ctx, tenantID, vehicleID)` vorschalten;
  kein gRPC-Bypass (Service-Ebene, nicht Handler), kein Stub, kein `.proto`, kein
  `RequirePermission`, keine neue Tabelle, keine Route, kein Guard ersetzt. Der
  Nachfolge-Commit `acf11811` aendert nur eine Journal-Zeile).
- neue-units: keine. Der Fund gehoert zu einer bereits existierenden Unit: der Root Cause von
  `fix-caldav-write-and-exceptions-blocked-by-missing-tenant-ctx` ist durch diesen Commit mit
  behoben (PrepareConn bekommt den Tenant jetzt auch fuer die beiden Direkt-Pool-Abfragen
  `checkCalendarWritePermission` und `listEventExceptions`). Das steht als NACHTRAG in deren
  `notes:` in `BACKLOG.yml`; die Unit bleibt `todo`, weil ihre beiden
  `_RealCalDAVContext_`-Tests ihren Context noch von Hand mit `CtxWithUser` bauen und deshalb
  weiter den alten Zustand dokumentieren - dort steht Beweisfuehrung offen, vermutlich keine
  Code-Aenderung mehr.
- offen: (1) Fuer Luke zum Nachvollziehen: `basicAuthMiddleware` macht jetzt EINE zusaetzliche
  DB-Abfrage pro CalDAV/CardDAV-Request (`SELECT tenant_id FROM users WHERE id=$1`, per
  `sysctx.With` an RLS vorbei - der einzige legitime tenant-uebergreifende Lookup, weil Basic
  Auth den Tenant nirgends mitbringt). `app_specific_passwords` traegt seit Migration 114 selbst
  ein `tenant_id NOT NULL` - `pwService.Validate` koennte den Tenant also ohne zweite Abfrage
  liefern. Das waere die schnellere und quellenreinere Loesung, kostet aber eine
  Signaturaenderung an `CalDAVPasswordService.Validate` quer durch Gateway-Interface und
  Adapter; bewusst nicht in diesem Commit. (2) Die vier `resolveTenantID`-Aufrufe fuer die
  Sync-Token-Writes in `caldav_backend.go`/`carddav_backend.go` sind seit diesem Commit
  redundant - der Tenant steht im Context. Ebenfalls bewusst nicht angefasst (fremde Funktionen,
  reine Optimierung). (3) Kein Produktions-Zugriff: dass echte Clients (DAVx5/Apple Kontakte)
  danach syncen, ist gegen echte Postgres und einen echten CRM-gRPC-Server belegt, aber nicht
  gegen einen echten DAV-Client.

## Iteration 5 — fix-rapporte-measurement-position-cross-tenant-insert — done — 2026-08-27 01:46
- commit: 9db416e3
- gebaut: `PostgresRepository.AddMeasurementPosition` (postgres_repository.go:672) fuegt nicht
  mehr blind per `VALUES` ein, sondern per `INSERT ... SELECT ... WHERE EXISTS (SELECT 1 FROM
  measurements WHERE id=$3 AND tenant_id=$2)` und meldet bei `RowsAffected()==0`
  `ErrMeasurementNotFound`. Guard und Insert sind eine einzige Anweisung — kein Fenster
  zwischen Pruefung und Schreiben. Der Fix sitzt in der Repository-Funktion, weil die Kette
  Gateway -> gRPC -> Repo nur diesen einen Aufrufer hat und sowohl FK (`REFERENCES
  measurements(id)`, prueft nur Existenz) als auch RLS-Policy (prueft die `tenant_id` der NEUEN
  Zeile) die Eigentuemerfrage strukturell offenlassen. `mapRapporteError` bildet
  `ErrMeasurementNotFound` bereits auf `codes.NotFound` ab, der Handler damit auf 404 — kein
  neuer Fehlerpfad noetig. Der Doku-Test heisst jetzt
  `TestAddMeasurementPosition_RejectsMeasurementIDFromAnotherTenant`, erwartet die Ablehnung,
  prueft zusaetzlich dass Tenant B's Aufmass danach 0 Positionen hat (also wirklich nichts
  geschrieben wurde) und dass eine voellig unbekannte `measurement_id` denselben Fehler bekommt.
  Interface-Kommentar in `repository.go` nennt die neue Fehlerzusage.
- gate: build ok (`go build -p 2 ./internal/rapporte/... ./internal/server/... ./internal/gateway/...
  ./cmd/gateway/...`) | vet ok | lint ok (`golangci-lint run ./internal/rapporte/...
  ./internal/server/... ./internal/gateway/...`, 0 issues) | test ok (DATABASE_URL gesetzt, Rolle
  `kmuhub_app`; `go test -count=1 ./internal/rapporte/` gruen, **86 Tests, 0 SKIP** — verbose
  gezaehlt; `./internal/gateway/` und `./internal/server/` ebenfalls gruen) | migration n.a.
  (keine Schema-Aenderung — der Fix sitzt in der Query, nicht im Schema) | rls-smoke n.a. (keine
  neue Tabelle/Policy; der Zwei-Tenant-DB-Test deckt die Grenze ab)
- coverage: `internal/rapporte` 76,8 % -> 76,9 % (selbst gemessen mit `go tool cover -func`,
  Vorher-Wert ueber `git stash push -- backend/internal/rapporte/`). `coverage_start` der Unit
  ist n.a. (Bugfix), der Wert steht auch in der Coverage-Tabelle im Backlog-Kopf nicht.
- mutations-probe: zwei Durchgaenge, weil der erste ein FALSCH-GRUEN war und genau das der
  interessante Befund ist. (1) Nur `AND tenant_id=$2` aus dem EXISTS entfernt -> Test bleibt
  GRUEN: der Subselect laeuft als `kmuhub_app`, und die RLS-Policy der Eltern-Tabelle
  `measurements` filtert die fremde Zeile schon vorher weg. (2) Der echte Ur-Bug: das ganze
  `SELECT ... WHERE EXISTS` zurueck auf `VALUES ($1,...,$9)` -> `--- FAIL:
  TestAddMeasurementPosition_RejectsMeasurementIDFromAnotherTenant ... expected
  ErrMeasurementNotFound ..., got <nil>`. Aus Sicherungskopie zurueckgedreht, `git diff --stat`
  zeigt wieder nur die beabsichtigten 12 Zeilen, Paket wieder gruen. Der explizite
  `tenant_id`-Filter bleibt drin (Defense-in-Depth fuer einen etwaigen Aufruf unter
  `sysctx`/BYPASSRLS), aber er ist NICHT die tragende Zeile — das steht so auch in den `notes`
  der neuen Unit.
- verify vorgaenger: sauber (`cff259b7`, gegen alle acht Fehlerklassen geprueft — kein
  gRPC-Bypass (die Aenderung liegt in Middleware und Injector, die DAV-Backends rufen weiter
  ueber ihre Clients), kein Stub, kein `.proto`, kein `RequirePermission`, keine neue Tabelle,
  keine neue Route (`route_caldav.go` aendert nur die Injector-Signatur und den Fehlerpfad der
  bestehenden Basic-Auth-Middleware), kein Guard ersetzt. Der Nachfolge-Commit `7b9beecf`
  aendert nur eine Journal-Zeile).
- neue-units: `fix-rapporte-worker-and-measurement-parent-report-tenant-check` — dieselbe
  Fehlerklasse eine Ebene hoeher: `RapporteGRPCServer.AddWorker` (rapporte_grpc.go:539) und
  `RapporteGRPCServer.CreateMeasurement` (rapporte_grpc.go:600) gehen am Service vorbei direkt
  aufs Repository und pruefen die `report_id` nie gegen den Tenant; beide Repo-Funktionen
  fuegen per reinem `VALUES` ein. Gegenbeweis, dass es kein Generalproblem ist:
  `Service.AddLine` und `Service.UploadAttachment` rufen beide vorher `GetReport(ctx, tenantID,
  reportID)`.
- offen: (1) `DeleteMeasurementPosition` war der zweite Prueffall aus den `notes` der Unit — er
  ist SAUBER (`DELETE ... WHERE id=$1 AND tenant_id=$2` plus `RowsAffected()==0` ->
  `ErrPositionNotFound`), kein Handlungsbedarf. (2) Fuer Luke wichtiger als der Fix selbst: die
  Mutations-Probe (1) oben zeigt, dass ein DB-Test unter `kmuhub_app` einen fehlenden
  Tenant-Filter im Subselect NICHT sichtbar macht, solange die Eltern-Tabelle eine RLS-Policy
  traegt. Wer diese Bug-Klasse anderswo "per Test" fuer erledigt erklaert, muss den ganzen
  Guard entfernen, nicht nur den Filter. (3) Bestehende Datenlage nicht geprueft: ob in der
  lokalen oder der Produktions-DB bereits `measurement_positions`-Zeilen liegen, deren
  `tenant_id` von der `tenant_id` ihrer `measurements`-Zeile abweicht, ist offen — kein
  Prod-Zugriff. Falls ja, faende sie `SELECT p.id FROM measurement_positions p JOIN measurements
  m ON m.id=p.measurement_id WHERE p.tenant_id <> m.tenant_id;` als `kmuhub` (Superuser, sonst
  filtert RLS die Antwort selbst weg).

## Iteration 6 — fix-generatejointoken-missing-event-tenant-check — done — 2026-08-27 01:52
- commit: c448273f
- gebaut: `CalendarGRPCServer.GenerateJoinToken` (calendar_grpc.go:1293) holt jetzt zuerst
  `middleware.GetTenantID(ctx)` und laedt das Event ueber `s.eventService.Get(ctx, eventID,
  tenantID)` — genau der Pfad, den `GetEvent` (calendar_grpc.go:461-482) schon benutzt —, bevor
  ueberhaupt ein Raumname gebildet wird. Vorher gab es an KEINER Stelle einen Event-Lookup: die
  UUID wurde geparst und direkt an `GenerateRoomName`/`GenerateJoinToken` gereicht, also bekam
  jeder Aufrufer mit tenant-weiter `calendars:write`-Berechtigung ein 24 h gueltiges,
  signiertes LiveKit-Token fuer eine BELIEBIGE UUID, auch fuer Events fremder Tenants.
  `mapCalendarError` bildet `event.ErrEventNotFound` bereits auf `codes.NotFound` ab
  (calendar_grpc.go:1614), der Gateway-Handler damit ueber `respondGRPCError` auf 404 — kein
  neuer Fehlerpfad, keine Routen- oder Proto-Aenderung, kein Spec-Eintrag noetig.
  Zur "abgesagt"-Frage aus dem scope: das Datenmodell kennt keinen Cancel-Status
  (`models.CalendarEvent` hat kein Status-Feld), Absage IST das Loeschen der Zeile
  (`event.Service.DeleteEvent` emittiert `calendar.event.cancelled` und ruft `repo.Delete`).
  Der Existenz-Check deckt den Fall damit vollstaendig mit ab — das steht so im Code-Kommentar.
  `TestGenerateJoinToken` legt jetzt ein echtes Event im Stub-Repo an, statt mit einer
  erfundenen `uuid.New()` Erfolg zu erwarten; dazu zwei neue Unterfaelle
  ("event belongs to another tenant", "event does not exist", beide NotFound + `resp == nil`)
  und "missing tenant context" (InvalidArgument). Der veraltete Kommentarblock in
  `route_calendar_resources_reminders_test.go`, der den Bug als "NOT fixed" fuehrte, ist auf
  den neuen Stand gezogen.
- gate: build ok (`go build -p 2 ./internal/server/... ./internal/gateway/... ./internal/work/...
  ./cmd/gateway/...`) | vet ok | lint ok (`golangci-lint run ./internal/server/...
  ./internal/gateway/...`, 0 issues) | test ok (DATABASE_URL gesetzt, Rolle `kmuhub_app`;
  `go test -count=1 ./internal/server/` gruen mit **4523 PASS, 0 SKIP, 0 FAIL** — verbose
  gezaehlt; `./internal/gateway/` gruen inkl. `TestOpenAPIRouteDrift` separat) | migration n.a.
  (kein Schema-Thema — der Fix ist ein fehlender Lookup, keine fehlende Spalte) | rls-smoke n.a.
  (keine neue Tabelle/Policy)
- coverage: `internal/server` 71,4 % -> 71,4 % (beide Werte selbst gemessen mit
  `go tool cover -func`, Vorher-Stand ueber `git show HEAD:<datei>` in eine Sicherungskopie).
  Kein Zugewinn und das ist erwartbar: `GenerateJoinToken` stand VOR dem Fix schon bei 100,0 %
  Statement-Coverage und steht danach wieder bei 100,0 %. Genau das ist der Beleg fuer die
  Grenze der Kennzahl — ein Handler kann voll abgedeckt sein und trotzdem eine fehlende
  Autorisierung enthalten, weil Coverage misst, ob eine Zeile lief, nicht ob die richtige Zeile
  da war. `coverage_start` der Unit war "n.a. (Sicherheits-Fix)", passt.
- mutations-probe: zwei Durchgaenge. (1) Den kompletten Lookup entfernt -> `go test` bricht mit
  `build failed` ab (`tenantID` unbenutzt), also kein verwertbares Signal, verworfen. (2) Die
  tragende Zeile auf `_, _ = s.eventService.Get(ctx, eventID, tenantID)` gebrochen (Lookup
  laeuft, Fehler wird verworfen — exakt die Wirkung des Ur-Bugs, aber compilierbar) ->
  `--- FAIL: TestGenerateJoinToken/event_belongs_to_another_tenant` UND
  `--- FAIL: TestGenerateJoinToken/event_does_not_exist`, happy path bleibt gruen. Aus
  `cp`-Sicherungskopie zurueckgedreht, `git diff --stat` zeigt wieder dieselben 101/23 Zeilen
  wie vor der Probe, Pakete `./internal/server/` und `./internal/gateway/` wieder gruen.
- verify vorgaenger: sauber (`9db416e3`, gegen alle acht Fehlerklassen geprueft — kein
  gRPC-Bypass (die Aenderung liegt allein in `rapporte/postgres_repository.go`), kein Stub,
  kein `.proto`, kein `RequirePermission`, keine neue Tabelle (nur die INSERT-Query der
  bestehenden `measurement_positions` umgestellt), Wire-Shape unveraendert (der Rueckgabetyp
  ist derselbe `*MeasurementPosition`), keine neue Route, kein ersetzter Guard. Der
  Nachfolge-Commit `37518d34` aendert nur eine Journal-Zeile).
- neue-units: `fix-createevent-never-sets-tenant-id` (opus) und
  `harden-livekit-room-name-truncation-collides-across-tenants` (sonnet).
  (1) Beim Pruefen der Nachbar-RPCs aufgefallen und am Code verifiziert: `CreateEvent`
  (calendar_grpc.go:391-457) ruft `middleware.GetTenantID` NIE auf und setzt
  `event.CreateInput.TenantID` nirgends. `event.Service.Create` (work/event/service.go:77) laedt
  den Kalender daraufhin mit der Null-UUID als Tenant, und
  `calendar/postgres_repository.go:41` filtert explizit `WHERE id=$1 AND tenant_id=$2` — fuer
  jeden echten Tenant kann das nie treffen. Der Befund stand schon als Kommentar im Testfile,
  hatte aber KEINE Unit; jetzt hat er eine, mit zwei Klaerungsfragen vorweg, weil "Events
  anlegen geht ueberhaupt nicht" nicht zum gemeldeten Betrieb passt.
  (2) `GenerateRoomName` (work/livekit/service.go:73-76) kuerzt auf `"cal-" + uuid[:8]`, also
  32 Bit ohne Tenant-Anteil — mein Fix schliesst das Token-Loch, nicht die
  Raumnamen-Kollision zwischen Tenants.
- offen: (1) Der Fix erzwingt nur den TENANT-Scope, keine Pro-Event-ACL. Wer im selben Tenant
  `calendars:write` hat, bekommt weiter fuer JEDES Event des Tenants ein Token, auch ohne
  Teilnehmer- oder Ersteller-Beziehung. Das ist dieselbe offene Architekturfrage wie bei
  `HandleListEventAttendees`/`GetEvent` (Journal Iteration 35 aus Lauf 12) und gehoert Luke,
  nicht dem Loop — das Minimum steht jetzt. (2) BEWUSST NICHT gebaut: eine Pruefung auf
  `evt.HasVideoCall`. Sie waere plausibel, aendert aber Verhalten fuer Events, bei denen das
  Flag nachtraeglich gesetzt wird, und stand nicht in den `done_when` — falls gewuenscht, ist
  es eine Zeile mehr im selben Block. (3) Nicht geprueft: ob bereits ausgegebene Tokens
  betroffen sind. LiveKit-Tokens sind 24 h gueltig und werden nicht serverseitig gefuehrt; ein
  vor dem Fix ausgestelltes Token fuer einen fremden Raum bleibt bis zum Ablauf gueltig. Falls
  Luke das ausschliessen will, hilft nur ein Wechsel des LiveKit-`apiSecret`.

## Iteration 7 — scan-tenant-filter-on-read-paths — done — 2026-08-27 02:00
- commit: c738ee21
- gebaut: Kein Code. Systematischer Scan der drei Wege an RLS vorbei (Redis, Objektspeicher,
  Postgres ohne Tenant-Kontext). Vollstaendige Schluessel-/Pfad-/Kontext-Inventur unten;
  drei Befunde als Units am Backlog-Ende.

  **(1) REDIS — sieben Schluesselfamilien, das ist die vollstaendige Flaeche.** Ermittelt
  ueber alle Nicht-Test-Dateien, die `redis/go-redis` importieren (10 Stueck: `cmd/dialer`,
  `cmd/gateway/setup.go`, `cmd/work`, `internal/cache`, `internal/database/redis.go`,
  `internal/dialer/redis_agent_store.go`, `internal/health`, `internal/middleware/ratelimit.go`,
  `internal/server/websocket.go`, `internal/work/presence/redis_store.go`), danach jede
  Redis-Kommandostelle einzeln aufgelistet — es gibt 21, alle in fuenf Dateien.
    - `cache:biz:dashboard:{tenant}:{von}:{bis}` (biz/dashboard/cached_repository.go:29) —
      Tenant IM Schluessel. SICHER.
    - `cache:pipelinestages:list:{tenant}` und `cache:pipelinestage:{tenant}:{id}`
      (crm/pipelinestage/cached_repository.go:14-15) — Tenant im Schluessel, und die
      Invalidierung in Create/Update/Delete/Reorder nutzt denselben Praefix. SICHER.
    - `cache:dashboard:defaults:{tenant}:{rolle}` und `cache:dashboard:user:{tenant}:{user}`
      (gateway/cached_dashboard_repository.go:13-14) — SICHER, und der Grund steht als
      Kommentar im Code: vor Migration 000274 waren Rollen-Defaults installationsweit, ein
      rollen-only-Schluessel wuerde den ersten waermenden Tenant 30 Minuten lang an alle
      anderen ausliefern.
    - `ratelimit:{userID|IP}` (middleware/ratelimit.go:76) — kein Tenant, ABSICHT. Die
      User-Variante ist eine global eindeutige UUID, kollidiert also nicht. Die IP-Variante
      teilen sich zwei Tenants hinter demselben NAT ein Kontingent — das ist ein geteiltes
      Limit, kein Datenabfluss, und im Auslieferungsmodell "ein Server pro Kunde" ohnehin
      gegenstandslos. KEIN Befund.
    - `ws:subscriptions:{channelID}` (server/websocket.go:37, SADD 1434 / SREM 1455) — kein
      Tenant, aber SICHER mit Grund: der Schluessel wird ausschliesslich GESCHRIEBEN. Es gibt
      im ganzen Repo keine lesende Operation darauf (kein SMEMBERS, kein SISMEMBER) — der
      Kommentar nennt ihn "Mirror fuer Phase D", die lokale Map ist massgeblich. Die
      Mitgliedschaft wird vorher ueber `chatClient.GetChannel(id, userID)` geprueft
      (websocket.go:685-692), also ueber RLS in Postgres. Er traegt heute keine
      Autorisierungsentscheidung.
    - `presence:{userID}` (work/presence/redis_store.go:14/51) — **BEFUND**, siehe unten.
    - `dialer:agent:status:{userID}` und `dialer:campaign:agents:{campaignID}`
      (dialer/redis_agent_store.go:14-15) — **BEFUND**, siehe unten.

  **(2) OBJEKTSPEICHER — sechs Pfadfamilien, ermittelt ueber alle vier Nicht-Test-Dateien,
  die `minio-go` importieren, plus alle Schluesselbildner im Repo.** Es gibt keinen
  Datei-System-Fallback: die einzige Nicht-MinIO-Implementierung ist `UnavailableStore`
  (chat/file/filestore.go:22), die auf jeden Aufruf einen Fehler zurueckgibt.
    - `{tenant}/{scope}/{uuid}{ext}` (document/file/presign.go:98) — Tenant im Pfad, aus
      `middleware.GetTenantID`, nie aus dem Request. Der Download prueft ihn zusaetzlich
      (`strings.HasPrefix`, presign.go:124-127) und antwortet sonst PermissionDenied.
      `scope` gegen Allowlist, `filepath.Ext` schneidet den Dateinamen auf die Endung, also
      auch kein Traversal ueber den Namen. SICHER.
    - `{tenant}/branding/…` — derselbe Pfad, zusaetzlich validiert beim Zurueckschreiben
      (`brandingObjectKeyValid`, settings/branding.go:60-66). SICHER.
    - `gobd/{tenant}/{docID}/{filename}` (biz/gobdarchive/service.go:88) — Tenant im Pfad.
      SICHER.
    - `documents/{spaceType}/{spaceID}/{folderID}/{fileID}/{filename}` sowie
      `documents/copy/…` und `documents/versions/…` (document/file/service.go:438/685/833/914)
      — KEIN Tenant im Pfad, trotzdem sicher mit Grund: der Schluessel wird nie vom Aufrufer
      geliefert, sondern immer aus einer tenant-gescopten DB-Zeile gelesen, und alle
      Bestandteile sind UUIDs, kollidieren also nicht. Anmerkung in `offen:`.
    - `channels/{channelID}/files/{fileID}/{filename}` (chat/file/service.go:136) — KEIN
      Tenant, sicher mit Grund: `GetPresignedURL` laedt die Zeile ueber
      `repo.GetFileByID` (RLS) und prueft danach `IsChannelMember` (service.go:222-240).
      Zwei Huerden, beide in Postgres.
    - `{prefix}/{accountID}/{messageUID}/{filename}` (email/attachment/store.go:39) — KEIN
      Tenant, sicher mit Grund: `GetDownloadURL` holt den Schluessel ueber
      `repo.GetMinIOKeyByID(ctx, id, tenantID)` (attachment/service.go:90-96), der Tenant ist
      also Teil der Abfrage. Der Aufrufer kann keinen Schluessel setzen.
    - Aufzeichnungen: `fileURLToObjectKey(*rec.FileURL, …)` (work/recording/service.go:651)
      leitet den Schluessel aus der DB-Zeile ab, davor steht eine Teilnehmer-ACL
      (service.go:629-644). SICHER.

  **(3) POSTGRES OHNE GESETZTEN TENANT-KONTEXT — drei Klassen, alle einzeln eingeordnet.**
    - **39 `WithSystemContext`-Stellen** (Vollzaehlung ueber `internal/` + `cmd/`, ohne
      Tests). Sieben davon sind Aufloeser fuer oeffentliche Links, die genau eine Zeile ueber
      eine eindeutige Token-Spalte holen und danach sofort in den Tenant-Scope zurueckkehren
      (`berichte.GetShareTokenBySecret`, `document/file.GetShareLinkByToken`,
      `formulare.GetShareLinkByToken`, `helpdesk.GetCsatSurveyByToken`,
      `wiki.GetShareTokenByToken`, `caldav.FindActiveByUser`,
      `notification/integration.ResolveTenant`) — jede traegt ihre Begruendung als
      Kommentar, keine ist eine Auflistung, keine nimmt einen Filter. Drei liegen im
      LiveKit-Webhook-Pfad (`CompleteRecordingByEgress`, `FailRecordingByEgress`,
      `CompleteMeetingByRoom`, video_grpc.go:1438/1462/1481) und sind tenantlos, weil LiveKit
      keinen Tenant kennt; erreichbar nur ueber `POST /api/v1/webhooks/livekit`, das die
      Signatur mit `lkwebhook.Receive` prueft und ohne gueltigen Authorization-Header 401
      antwortet (route_video.go:1467-1476). Der Rest sind Scheduler, Worker und
      Aufraeumlaeufe (Retention, Snooze, E-Mail-Sync, Bexio/Lexware-Scheduler,
      WOPI-Lock-Cleanup, Guest-Session-Cleanup) — kein Anfragepfad. ALLE ABSICHT.
    - **Zwei direkte `pgx.Connect` ausserhalb von `NewPostgresPool`** (also ohne den
      GUC-Hook): `cmd/gateway/setup.go:88` und `notification/event/bus.go:89`. Beide machen
      ausschliesslich `LISTEN` + `WaitForNotification` und fassen keine Tabelle an. Der
      Event-Bus stempelt den Tenant vor dem Dispatch aus der Nutzlast
      (`bus.dispatch`, bus.go:135-137). ABSICHT.
    - **Fehlender Tenant im Anfragepfad** — die Pool-Voreinstellung ist fail-closed
      (postgres.go:60-62 laesst die GUCs leer, die Policy filtert dann alles weg). Das ist
      kein Leck, kann aber eine Schreiboperation lautlos wirkungslos machen. Genau ein
      solcher Fall gefunden: **BEFUND** `route_caldav.go:453`.

  **DREI BEFUNDE**
    - `presence:{userID}`: der WS-Pfad ist seit Iteration 2 dicht
      (`tenantAllowsPresenceTarget`), der HTTP/gRPC-Pfad NICHT. `GetPresence` und
      `GetBulkPresence` (video_grpc.go:1361/1370) reichen die Fremd-UUID ungeprueft an den
      Redis-Store durch; erreichbar ueber `GET /api/v1/presence/{userId}` und
      `POST /api/v1/presence/bulk`. Redis hat kein Netz darunter.
    - `dialer:agent:status:{userID}` / `dialer:campaign:agents:{campaignID}`: der Eintrag
      TRAEGT `tenant_id` (redis_agent_store.go:37/104), gelesen wird das Feld nirgends.
      `GET /api/v1/dialer/agents?user_id=…` und
      `GET /api/v1/dialer/agents/campaign/{campaignId}` liefern damit fremde Agentenstatus;
      die Kampagnen-Variante gleich als Liste.
    - `route_caldav.go:453`: der Widerruf des Test-App-Passworts laeuft mit
      `context.Background()` gegen eine RLS-Tabelle und trifft null Zeilen — das Passwort
      bleibt gueltig, und die Tabelle hat keine Ablaufspalte.
- gate: build n.a. | vet n.a. | lint n.a. | test n.a. — **kein Go-Code geaendert**, die Unit
  fordert ausdruecklich "Kein Verhalten geaendert". `git status` zeigt genau zwei Dateien:
  BACKLOG.yml und JOURNAL.md. | migration n.a. | rls-smoke **ok und tragend**: als
  `kmuhub_app` mit leerem `app.tenant_id` gegen die lokale DB (Migrationskopf 325, clean)
  liefert `app_specific_passwords` 0 sichtbare Zeilen und ein UPDATE 0 betroffene Zeilen,
  waehrend die Tabelle als `kmuhub` 6 Zeilen enthaelt. Das ist der Beleg fuer den
  CalDAV-Befund, nicht nur dessen Herleitung. `backlog-check.py --preflight` gruen.
- coverage: n.a. (Scan, kein Coverage-Ziel — `coverage_start` der Unit sagt dasselbe)
- mutations-probe: n.a. (kein Code geaendert, es gibt nichts zu brechen)
- verify vorgaenger: sauber (`c448273f`, gegen alle acht Fehlerklassen geprueft. Kein
  gRPC-Bypass — die Aenderung liegt im gRPC-Server selbst und ruft `s.eventService.Get`, den
  Weg, den `GetEvent` im selben File schon geht. Kein Stub. Kein `.proto` im Diff. Kein
  `RequirePermission` angefasst, also weder Seed-Pflicht noch verlorener Alt-Key. Keine neue
  Tabelle. Wire-Shape unveraendert, der Erfolgsfall gibt dieselbe
  `GenerateJoinTokenResponse` zurueck. Keine neue Route, `openapi.yaml` nicht im Diff und
  auch nicht noetig. Der Nachfolge-Commit `6cad8da8` aendert nur eine Journal-Zeile.)
- neue-units: `fix-getpresence-rpc-missing-tenant-check` (opus),
  `fix-dialer-agent-status-missing-tenant-check` (opus),
  `fix-caldav-test-revoke-runs-without-tenant-context` (sonnet). Die ersten beiden sind
  Redis-Funde und deshalb opus, wie die Unit es vorschreibt; der dritte liegt in Postgres mit
  RLS darunter, faellt also fail-closed aus und ist sonnet.
- offen: (1) **Was tief geprueft ist:** die Redis-Flaeche vollstaendig (alle 21
  Kommandostellen einzeln gelesen, alle sieben Schluesselfamilien bis zum Aufrufer verfolgt),
  die Objektspeicher-Flaeche vollstaendig (alle sechs Pfadfamilien bis zu der Stelle
  verfolgt, die den Schluessel liefert), und die drei Postgres-Klassen. **Was nur gegrept
  ist:** die 39 `WithSystemContext`-Stellen — ich habe je 8 bis 18 Zeilen Kontext gelesen und
  die Doku-Kommentare bewertet, aber nicht jede der sieben Token-Abfragen Zeile fuer Zeile
  gegen ihr SQL geprueft. Sie tragen alle denselben Aufbau und dieselbe Begruendung; wer
  einen davon anfasst, sollte das SQL trotzdem selbst lesen. **Was gar nicht angesehen
  wurde:** der WASM-Plugin-Speicher (Feature-Flag OFF, 0,0 % Coverage) und alles unter
  `internal/testsupport`/`internal/testutil`.
  (2) `documents/…`-Schluessel tragen keinen Tenant, die Presign-Schluessel schon. Beide Wege
  sind fuer sich sicher, sie sind aber nicht kombinierbar: `GetPresignedDownloadURL` verlangt
  den Tenant-Praefix und wuerde einen `documents/…`-Schluessel mit PermissionDenied ablehnen.
  Heute kollidiert das nicht (der einzige Aufrufer ist `route_files.go:113` mit einem
  Presign-Schluessel), aber es ist eine Falle fuer den naechsten, der die beiden Wege
  zusammenlegen will. Keine Unit, weil nichts kaputt ist.
  (3) `ws:subscriptions:{channelID}` ist heute schreib-only. Sobald Phase D daraus liest — der
  Kommentar kuendigt genau das an —, wird aus der fehlenden Tenant-Komponente ein echtes
  Thema. Ebenfalls keine Unit, weil es ein zukuenftiger und kein bestehender Zustand ist.
  (4) Der Widerruf-Befund erklaert moeglicherweise Alt-Zeilen in Produktion: dort koennten
  aktive `app_specific_passwords` mit Label `connection-test` stehen, eines pro je
  ausgefuehrtem Verbindungstest. Lokal sind es 6 Zeilen insgesamt (Label nicht geprueft, weil
  die Spalte unter RLS als `kmuhub_app` nicht lesbar ist und ich als `kmuhub` nur gezaehlt
  habe). Produktion muss Luke selbst nachsehen — der Loop hat dort keinen Zugriff.

## Iteration 8 — cov-email-send-service-consent-path — done — 2026-08-27 02:11
- commit: 0d082014
- gebaut: `internal/email/send/service_test.go`-Umfeld bereits mit vier Consent-Tests
  (blockiert, erlaubt, `contact_id` fehlt, Repo-Fehler) bestueckt — die deckten den
  Asserter-Aufruf in `Send` schon vollstaendig ab. Neu ist
  `internal/email/send/smtp_rejection_test.go`: ein minimaler In-Prozess-SMTP-Server
  (`net.Listen` + `net/textproto`), der die Handshake normal durchlaeuft und dann an einer von
  drei Stellen mit einem permanenten 5xx antwortet (MAIL FROM, RCPT TO, DATA). Ein Table-Test
  (`TestSend_SMTPRejection_WrapsErrSendFailed`) prueft fuer alle drei Stufen, dass `Send`
  `ErrSendFailed` zurueckgibt und `messageCreator.Create` NICHT aufgerufen wird — eine
  abgelehnte Zustellung wird also nicht als lokal gespeicherte "gesendete" Nachricht gefuehrt.
  Festgeschrieben (Feststellung, kein Bug): fehlt `contact_id`, laesst der Asserter den Versand
  unveraendert durch (`TestSend_NoContactID_SkipsConsentCheck`, bereits vorhanden) — das ist die
  bewusste Zwischenloesung, bis das Frontend `contact_id` setzt (G1-Punkt 2). Es gibt keine
  Bounce-Verarbeitung im Code (`grep -ri bounce` in `internal/email` liefert nichts): eine
  SMTP-Ablehnung wird nicht separat als Bounce erfasst, sondern lediglich als `ErrSendFailed`
  an den Aufrufer durchgereicht, der ihn ueber `email_grpc.go:1450` auf `codes.Internal` mappt —
  der Anrufer sieht also einen Fehler, es geht nichts still verloren, aber es existiert keine
  Bounce-Historie. Das ist eine Feststellung, keine neue Unit: nichts ist kaputt, es fehlt nur
  eine Funktion, die niemand verlangt hat.
- gate: build ok | vet ok | lint ok (`golangci-lint` 0 issues) | test ok
  (`go test ./internal/email/send/...` gruen, 0 uebersprungen — das Paket hat keine
  `SkipIfNoDB`-Stellen, reine Unit-Tests) | migration n.a. | rls-smoke n.a. (keine Tabelle
  angefasst)
- coverage: internal/email/send 57,6 % -> 64,8 % | service.go:Send 30,4 % -> 65,2 %
- mutations-probe: `Send`s Fehlerbehandlung nach `sendSMTP` auf ein verschlucktes `_ = err`
  gesetzt (SMTP-Ablehnung wird ignoriert) — alle drei Subtests von
  `TestSend_SMTPRejection_WrapsErrSendFailed` wurden rot ("error is expected but got nil"),
  danach `service.go` per Backup zurueckgespielt (`git diff` zeigt keine Abweichung vom
  Commit-Stand) und `go test ./internal/email/send/...` erneut gruen.
- verify vorgaenger: sauber (`c738ee21` + `abc4e2c0` gegen alle acht Fehlerklassen geprueft.
  Beide Commits aendern ausschliesslich `BACKLOG.yml`/`JOURNAL.md`, kein Go-Code — keine der
  Fehlerklassen kann hier zuschlagen.)
- neue-units: keine
- offen: keine. `internal/gateway/route_email.go` und `email_grpc.go` waren nur Lesequelle,
  nicht angefasst — `go test ./internal/gateway/` daher nicht Teil dieses Gates.

## Iteration 9 — cov-wiki-repository-and-share-tokens — done — 2026-08-27 02:35
- commit: fad481c5
- gebaut: Drei neue Testdateien fuer die zehn im Scope genannten, zuvor ungetesteten
  Gateway-Handler und die groesste Luecke im Repository:
  1. `backend/internal/gateway/route_wiki_test.go` (erweitert) — 22 neue Tests fuer
     HandleListArticles, HandleSearchArticles (inkl. MissingQuery-400), HandleListCategories,
     HandleUpdateCategory, HandleDeleteCategory, HandleListAttachments, HandleDeleteAttachment,
     HandleCreateShareToken, HandleListShareTokens, HandleRevokeShareToken — je
     ServiceUnavailable- und, wo zutreffend, InvalidJSON-/Invalid-UUID-Fehlerpfad, im selben
     Muster wie die bereits vorhandenen HandleCreateArticle/HandleCreateCategory-Tests.
     Zwei Testnamen (`TestHandleListAttachments_ServiceUnavailable`,
     `TestHandleDeleteAttachment_ServiceUnavailable`) kollidierten mit bereits vorhandenen
     Namen in `route_rapporte_test.go` (eigene Attachment-Handler) — mit `TestWiki`-Praefix
     eindeutig gemacht.
  2. `backend/internal/wiki/postgres_repository_db_test.go` (neu) — zwei DB-Tests gegen echtes
     Postgres/RLS: `TestWikiArticleReads_ScopeToCallerTenant` deckt GetArticleByID/BySlug,
     SlugExists, ListArticles (Search-/Published-Filter, Fremd-Tenant-Leerlauf),
     SearchArticles (Volltreffer + published-only-Filter nach Unpublish), UpdateArticle,
     ListAttachments und DeleteArticle (inkl. stiller No-Op bei Fremd-Tenant) ab.
     `TestWikiCategories_CRUDScopesToCallerTenant` deckt CreateCategory/GetCategory/
     ListCategories/UpdateCategory/DeleteCategory komplett ab. `postgres_repository.go` hatte
     bislang KEINE eigene DB-Testdatei — nur `tenant_write_test.go` (Schreibpfade) und
     `tenant_isolation_phase2_test.go` (rohes RLS via SeedRow) existierten; die Lesepfade und
     die komplette Kategorie-Flaeche liefen ungetestet.
  Schwerpunkt-Fragen der Unit, mit Beleg beantwortet:
  (1) Drei-Token-Vergleich (WOPI aus Iteration 34, Recording-Download aus Iteration 24,
  Wiki-Share hier): alle drei sind strukturell verschieden. WOPI bettet `file_id`+`tenant_id`
  als JWT-Claims ein (10 h fest, KEIN Einzel-Widerruf — nur die grobe tenant-weite
  "documents:write"-Berechtigung sperrt). Recording-Download ist eine Presigned-URL mit 1 h
  fester Laufzeit, Tenant-Bindung ausschliesslich ueber RLS am authentifizierten Aufrufer, ein
  einmal ausgestellter Link bleibt bis Ablauf gueltig (kein Sofort-Widerruf). Wiki-Share ist
  der einzige der drei mit ECHTEM Einzel-Widerruf: der Token ist eine eigene DB-Zeile
  (32-Byte crypto/rand, `revoked_at` weich gesetzt), unauthentifiziert nutzbar, der Tenant wird
  aus der Token-Zeile selbst aufgeloest (System-Context-Lookup, danach Re-Entry in
  Tenant-Scope), und jede Einloesung prueft den Artikel-Status live nach (Unpublish killt
  bereits verteilte Links sofort). Die Abweichung ist der eigentliche Befund: nur Wiki-Share
  bietet granulare, sofortige Widerrufbarkeit — kein Fund, weil das die bewusst staerkere
  Eigenschaft ist, nicht die schwaechere.
  (2) `HandleSearchArticles`-Tenant-Filter: `postgres_repository.go:158-166` setzt
  `tenant_id = $1` explizit in SQL (nicht nur RLS) UND `published = TRUE` — durch den neuen
  DB-Test bewiesen (Fremd-Tenant-Suche liefert 0 Treffer trotz identischem tsquery-Treffer im
  eigenen Tenant). Kein Fund.
  (3) Repository gegen echtes SQL: siehe oben, ungetagt, `DATABASE_URL` gesetzt.
- gate: build ok (`./internal/wiki/... ./internal/gateway/...`) | vet ok | lint ok
  (`golangci-lint run --config .golangci.yml ./internal/wiki/... ./internal/gateway/...`,
  0 issues) | test ok (`internal/wiki` 47/47 gruen, `internal/gateway` komplett gruen,
  `DATABASE_URL` gesetzt, 0 uebersprungene Tests in beiden Paketen) | migration n.a. (keine
  neue Tabelle/Policy) | rls-smoke ok (Teil der neuen DB-Tests: Fremd-Tenant-Reads/-Writes
  liefern durchgehend 0 Zeilen bzw. `Err*NotFound`, nie stillen Erfolg)
- coverage: internal/wiki 53,5 % -> 75,9 % (`postgres_repository.go` 29,9 % -> deutlich
  hoeher, alle zuvor 0-%-Methoden inkl. GetArticleByID/BySlug, SlugExists, ListArticles,
  SearchArticles, UpdateArticle, DeleteArticle, ListAttachments und die komplette
  Kategorie-CRUD jetzt 76-100 %) | internal/gateway 69,3 % -> 69,6 % (`route_wiki.go`: alle
  zehn im Scope genannten Handler vorher 0 %, jetzt 24-60 % Funktionsabdeckung ueber
  Fehlerpfade)
- mutations-probe: zwei getrennte Proben, beide per `cp`-Backup zurueckgespielt.
  (a) Gateway: in `HandleSearchArticles` die `q == ""`-Pruefung entfernt ->
  `TestHandleSearchArticles_MissingQuery` sofort rot (503 statt 400, Fehlermeldung nennt einen
  Verbindungsfehler statt "q query parameter is required"). (b) Repository: in `ListArticles`
  das `Published`-Filter-Argument auf `!*filter.Published` invertiert ->
  `TestWikiArticleReads_ScopeToCallerTenant` sofort rot ("published filter): total=0 len=0,
  want 1"). Beide Dateien per `cp`-Sicherungskopie zurueckgespielt, `git diff --stat` danach
  leer, `go test ./internal/wiki/... ./internal/gateway/` erneut komplett gruen.
- verify vorgaenger: sauber (`0d082014` gegen alle acht Fehlerklassen geprueft — reiner neuer
  Testdateidiff (`smtp_rejection_test.go`), kein gRPC-Bypass moeglich, kein Stub, kein
  `.proto` im Diff, kein `RequirePermission` angefasst, keine neue Tabelle, kein Wire-Shape,
  keine neue Route.)
- neue-units: keine
- offen: keine. `internal/server/wiki_grpc.go` war nur Lesequelle zum Verstehen der
  Response-Formen, nicht angefasst.

## Iteration 10 — cov-gateway-helpdesk-sla-queues-and-kb — done — 2026-08-27 02:55
- commit: 4267c87e
- gebaut:
  1. `backend/internal/helpdesk/sla_test.go` (neu) — sechs Tests fuer `ComputeStatus`/
     `ApplyPolicy`. Kern des Scopes: `TestComputeStatus_AcrossWeekend` (Freitag-18:00-CEST
     Fensterbeginn, Sonntag-Faelligkeit, on_track/at_risk/breached ueber das Wochenende
     geprueft) und `TestApplyPolicy_AcrossDSTSpringForward` (Europe/Berlin,
     2026-03-29-Sommerzeitwechsel, `now.Add(duration)` addiert exakt Elapsed-Minuten, nicht
     Wall-Clock-Minuten — Due-Zeit landet korrekt auf 04:00 CEST statt der wall-clock-naiven
     Erwartung 03:30). Beide sind der geforderte Beleg aus dem Backlog-Punkt (1). Dabei
     Doppel-Deklaration mit vier bereits existierenden Tests aus `service_test.go` entdeckt
     und bereinigt (TestComputeStatus_NoDueAt, TestApplyPolicy_NilPolicy) — die Datei hatte
     schon Basis-SLA-Tests, nur keine Wochenend-/DST-Faelle.
  2. Frage aus den Notes beantwortet: Die Zeitrechnung IST injizierbar (`ComputeStatus`/
     `ApplyPolicy` nehmen `now time.Time` als Parameter, der Service reicht nur
     `time.Now().UTC()` durch) — kein Designfehler, keine Extra-Unit noetig.
     `ApplyPolicy`s eigener Doc-Kommentar bestaetigt zusaetzlich: Business-Hours-Subtraktion
     ist bewusst NICHT implementiert ("assume 24/7 availability") — der Weekend-Test belegt
     genau dieses dokumentierte Verhalten (kein Business-Hours-Skip), kein neuer Fund.
  3. `backend/internal/gateway/route_helpdesk_test.go` — 42 neue Tests fuer alle 13 im Scope
     genannten Handler (HandleGetSLAStatus, HandleListSLAPolicies, HandleUpdateSLAPolicy,
     HandleDeleteSLAPolicy, HandleListQueues, HandleUpdateQueue, HandleDeleteQueue,
     HandleGetBusinessHours, HandleUpdateBusinessHours, HandleListKBArticles,
     HandleDeleteKBArticle, HandleGetHelpdeskStats, HandleCreateTicketFromMessage), jeweils
     inkl. Fehlerpfad (ServiceUnavailable/MissingTenant/InvalidJSON/Validation/InvalidIDUUID)
     und ReachesRPC-Nachweis, nach dem etablierten Muster ohne bufconn-Stub fuer
     HelpdeskServiceClient. `HandleCreateTicketFromMessage` zusaetzlich dokumentiert: Tenant-
     und Requester-ID kommen aus dem Auth-Context, nicht aus dem Body (der nur `message_id`
     traegt) — ein konvertiertes Ticket kann nicht auf einen fremden Tenant/Kontakt zeigen.
  4. Backlog-Punkt (2) beantwortet, ohne Code zu aendern: `HandleDeleteQueue`/
     `HandleDeleteSLAPolicy` bei vorhandenen Tickets pruefen KEINE Referenz-Integritaet im
     Service — brauchen es auch nicht, weil `tickets.queue_id` und `tickets.sla_policy_id`
     beide `ON DELETE SET NULL` sind (`000077_create_helpdesk.up.sql` Zeilen 20+36). Ein
     Loeschen mit angehaengten Tickets schlaegt nie fehl, es entkoppelt sie nur. Als
     Kommentar im Testfile dokumentiert, kein Fund.
- gate: build ok (`./internal/helpdesk/... ./internal/gateway/...`) | vet ok | lint ok
  (`golangci-lint run --config .golangci.yml ./internal/helpdesk/... ./internal/gateway/...`,
  0 issues) | test ok (`internal/helpdesk` komplett gruen, `internal/gateway` komplett gruen,
  `DATABASE_URL` gesetzt, 0 uebersprungene Tests in beiden Paketen) | migration n.a. (keine
  neue Tabelle/Policy) | rls-smoke n.a. (keine Tabellen-/Policy-Aenderung)
- coverage: internal/gateway 69,3 % -> 70,3 % (`route_helpdesk.go`: alle dreizehn im Scope
  genannten Handler vorher 0 %, jetzt 91,7–94,4 % Funktionsabdeckung ueber Fehlerpfade) |
  internal/helpdesk 81,5 % -> 81,6 % (marginal, `sla.go` war durch bestehende Tests schon
  gut abgedeckt — die neue Weekend-/DST-Deckung ist inhaltlich, nicht flaechenmaessig
  wertvoll)
- mutations-probe: zwei getrennte Proben, beide per `cp`-Backup zurueckgespielt.
  (a) Gateway: in `updateBusinessHoursRequest` das `validate:"required"`-Tag von
  `ScheduleJSON` auf `"omitempty"` geaendert -> `TestHandleUpdateBusinessHours_MissingScheduleJSON`
  sofort rot (503 statt 400, echter RPC-Connect-Fehler statt Validierungsfehler). (b) SLA:
  in `ComputeStatus` die Breached-Bedingung von `!now.Before(due)` auf `now.Before(due)`
  invertiert -> sieben Tests sofort rot, darunter beide neuen
  (`TestComputeStatus_AcrossWeekend`, `TestApplyPolicy_AcrossDSTSpringForward`) und fuenf
  bereits bestehende. Beide Dateien per `cp`-Sicherungskopie zurueckgespielt, `git diff --stat`
  danach leer, `go test ./internal/helpdesk/... ./internal/gateway/` erneut komplett gruen.
- verify vorgaenger: sauber (`fad481c5` gegen alle acht Fehlerklassen geprueft — reiner
  neuer Testdateidiff (`route_wiki_test.go`, `postgres_repository_db_test.go`), kein
  gRPC-Bypass moeglich, kein Stub, kein `.proto` im Diff, kein `RequirePermission`
  angefasst, keine neue Tabelle, kein Wire-Shape, keine neue Route.)
- neue-units: keine
- offen: keine.

## Iteration 11 — cov-gateway-chat-reactions-bookmarks-search — done — 2026-08-27 03:15
- commit: fc68f158
- gebaut:
  1. `backend/internal/gateway/route_chat_reactions_bookmarks_search_test.go` (neu) — 41 Tests
     fuer alle 14 im Scope genannten Handler (HandleGetMessages, HandleGetThreadReplies,
     HandleGetUnreadCounts, HandleGetUserMentions, HandleListDMs, HandleListReactions,
     HandleToggleReaction, HandleGetReactionSummary, HandleListBookmarks,
     HandleToggleBookmark, HandleListChannelFiles, HandleGetFileThumbnailURL,
     HandleSearchChat, HandleUpdateChannel), jeweils ServiceUnavailable/NoUserID/
     InvalidUUID/InvalidJSON/Validation/ReachesRPC nach etabliertem Muster.
  2. Schwerpunkt (1) aus dem Scope untersucht: `HandleSearchChat` -> `search.Service.Search`
     (`internal/chat/search/service.go:60-70`). Befund: wenn der Aufrufer `channel_id` als
     Query-Parameter mitschickt, wird dieser Channel DIREKT durchsucht, ohne vorher
     `GetUserChannelIDs` aufzurufen und Mitgliedschaft zu pruefen — der Membership-Filter
     greift nur auf dem Pfad OHNE `channel_id`. Kontrastprobe: `bookmark.Service.Toggle`
     (`internal/chat/bookmark/service.go:39-42`) ruft bewusst `message.Service.GetByID` auf,
     WEIL das Tenant+Membership prueft — genau dieses Muster fehlt in `search.Service.Search`.
     Als Coverage-Unit KEIN Verhalten geaendert (harte Grenze). Stattdessen: Bug-Test in
     `internal/chat/search/service_test.go` ergaenzt, der den aktuellen (fehlerhaften)
     Zustand mit klarer Kennzeichnung dokumentiert (Testname
     `BUG:_explicit_channel_id_bypasses_membership_check`), und Fix-Unit
     `fix-chat-search-channel-filter-bypasses-membership` ans Backlog-Ende gehaengt.
  3. Schwerpunkt (2): `HandleListDMs`/`HandleGetUserMentions` — beide tenant-scoped ueber den
     Service (ListDMs nimmt tenantID explizit vom Handler-Kontext), kein Fund.
  4. Schwerpunkt (3), `HandleGetFileThumbnailURL`: gleiches Muster wie
     GetFileDownloadURL/WOPI/Wiki-Share — Handler reicht `userID` durch, Autorisierung liegt
     im Service. Kein neuer Fund, konsistent mit den drei vorherigen Token-Units.
  5. Schwerpunkt (4), Umschalter bei Nebenlaeufigkeit: `reaction.PostgresRepository.AddReaction`
     nutzt `ON CONFLICT DO NOTHING` auf der zusammengesetzten PK
     (message_id, user_id, emoji) (Migration 000038); `bookmark.PostgresRepository.Add`
     dasselbe auf (user_id, message_id) (Migration 000262). Beide race-sicher per Design.
     Neuer echter DB-Test `internal/work/reaction/postgres_repository_db_test.go`
     (`TestPostgresAddReaction_ConcurrentToggle_OnlyOneRow`, 5 parallele Goroutinen, echter
     Pool) belegt das gegen die reale INSERT-Bahn, nicht nur gegen den SQL-Text.
  6. Geloeschte-Nachrichten-Frage aus den Notes: bereits durch bestehenden DB-Test
     `internal/chat/search/postgres_repository_test.go::TestPostgresRepository_SearchMessages`
     bewiesen (`is_deleted=true`-Zeile wird ausgeschlossen) — kein neuer Test noetig, nur
     verifiziert.
- gate: build ok (`./internal/gateway/... ./internal/chat/... ./internal/work/reaction/...`)
  | vet ok | lint ok (0 issues) | test ok (`internal/gateway` komplett gruen,
  `internal/chat/...` komplett gruen, `internal/work/reaction` komplett gruen,
  `DATABASE_URL` gesetzt, 0 uebersprungene Tests) | migration n.a. (keine neue
  Tabelle/Policy) | rls-smoke n.a. (keine Tabellen-/Policy-Aenderung)
- coverage: internal/gateway 69,3 % -> 71,2 % (route_chat.go: alle 14 Ziel-Handler vorher
  0 %, jetzt 84,2–96,7 % Funktionsabdeckung) | internal/work/reaction n.a. (nicht der
  Referenzwert dieser Unit; neuer DB-Test bei 38,9 % Paket-Deckung nach der Aenderung,
  vorher nicht gemessen)
- mutations-probe: zwei getrennte Proben, beide per `cp`-Backup zurueckgespielt.
  (a) Gateway: in `HandleSearchChat` die `uuid.Parse`-Fehlerpruefung invertiert
  (`err == nil` statt `err != nil`) -> `TestHandleSearchChat_InvalidChannelID` sofort rot
  (503 statt 400) und `TestHandleSearchChat_ReachesRPC` ebenfalls rot (400 statt 503).
  (b) Reaction: `ON CONFLICT DO NOTHING` aus `AddReaction`s INSERT entfernt ->
  `TestPostgresAddReaction_ConcurrentToggle_OnlyOneRow` sofort rot
  (`duplicate key value violates unique constraint "message_reactions_pkey"`). Beide
  Dateien per `cp`-Sicherungskopie zurueckgespielt, `git diff --stat` danach leer,
  betroffene Pakete erneut komplett gruen.
- verify vorgaenger: sauber (`4267c87e` gegen alle acht Fehlerklassen geprueft — reiner
  neuer Testdateidiff (`route_helpdesk_test.go`, `sla_test.go`), kein gRPC-Bypass moeglich,
  kein Stub, kein `.proto` im Diff, kein `RequirePermission` angefasst, keine neue Tabelle,
  kein Wire-Shape, keine neue Route.)
- neue-units: fix-chat-search-channel-filter-bypasses-membership (Datenleck innerhalb des
  Tenants: `HandleSearchChat` mit explizitem `channel_id` umgeht die
  Channel-Mitgliedschaftspruefung komplett, verifiziert per Test in `service_test.go`)
- offen: `fix-chat-search-channel-filter-bypasses-membership` ist ein echtes Datenleck
  (kein Tenant-Uebergriff dank RLS, aber jeder User im Tenant kann private Channels lesen,
  wenn er die UUID kennt/eraet) — sollte zeitnah gebaut werden, steht aber am Backlog-Ende
  wie von den Laufregeln fuer Coverage-Unit-Funde vorgeschrieben. Luke kann sie bei Bedarf
  manuell nach vorne ziehen.

## Iteration 12 — cov-gateway-vertraege-lifecycle-and-signature — done — 2026-08-27 02:56
- commit: 76175de2
- gebaut:
  1. `backend/internal/gateway/route_vertraege_lifecycle_signature_test.go` (neu) — 34 Tests
     fuer alle zehn im Scope genannten Handler (HandleGetContract, HandleListContracts,
     HandleExportContract, HandleListContractEvents, HandleListParties, HandleRemoveParty,
     HandleListReminders, HandleUpdateReminder, HandleDeleteReminder,
     HandleSaveContractSignature), jeweils ServiceUnavailable/NoTenantID/InvalidUUID/
     ReachesRPC nach etabliertem Muster; HandleListContracts zusaetzlich mit
     InvalidContactID (bislang ungetesteter Zweig), HandleSaveContractSignature mit
     MissingSignatureData/MissingSignedBy (validate:"required").
  2. Schwerpunkt (1) aus dem Scope untersucht: kein Immutability-Schutz nach der
     Unterschrift. `PostgresRepository.SaveSignature` (postgres_repository.go:27) hat kein
     "AND signature_data IS NULL", `Service.SaveSignature` (service.go:543) prueft nur
     Format/Groesse/Pflichtfelder der neuen Signatur. Verifiziert per neuem echten DB-Test
     `TestSaveSignature_OverwritesExistingSignatureWithoutGuard`
     (`postgres_repository_db_test.go`): zweiter `SaveSignature`-Aufruf gelingt und
     ueberschreibt `signature_data`/`signed_at`/`signed_by` spurlos. Exakt derselbe
     Fehler wie in Lauf 12 fuer rapporte und vermietung gefunden (beide dort ebenfalls
     noch offene `todo`-Fix-Units) — jetzt das dritte von drei Signatur-Modulen.
     Als Coverage-Unit KEIN Verhalten geaendert; Fix-Unit
     `fix-vertraege-signature-overwritable-after-signing` ans Backlog-Ende gehaengt.
  3. Schwerpunkt (2) aus dem Scope untersucht: `HandleRemoveParty` bei einer
     unterzeichnenden Partei. Weder `Service.RemoveParty` (service.go:430) noch
     `PostgresRepository.RemoveParty` (postgres_repository.go:245) pruefen
     `ContractParty.SignedOn` vor dem Loeschen — eine Partei, die bereits unterschrieben
     hat, wird genauso entfernt wie eine, die nie unterschrieben hat, und der
     `party_removed`-Event haelt nur die `party_id` fest. Verifiziert per neuem
     BUG-Test `TestService_RemoveParty_SignedParty_BUG_EvidenceRemovedWithoutGuard`
     (`contract_events_test.go`, mock-Repo). Fix-Unit
     `fix-vertraege-remove-signed-party-destroys-evidence` ans Backlog-Ende gehaengt.
  4. Schwerpunkt (3), Erinnerungs-Worker: `ReminderWorker.processReminders` haengt an
     keinem Lock, sondern an einem atomaren `UPDATE ... WHERE status='pending'`
     (`ClaimDueReminders`) — kein Advisory-Lock-Leak-Muster. Bereits per Mock-Test
     belegt, dass ein zweiter Lauf ein bereits versandtes Reminder nicht erneut
     verarbeitet (`TestReminderWorker_EmitsEventForDueReminder`). Zusaetzlich echten
     SQL-Beweis nachgezogen: `TestRepository_ClaimDueRemindersAndMarkSent` ruft
     `ClaimDueReminders` jetzt ein zweites Mal auf und prueft, dass das bereits
     geclaimte Reminder nicht erneut zurueckkommt.
- gate: build ok (`./internal/vertraege/... ./internal/gateway/... ./cmd/vertraege/...
  ./cmd/gateway/...`) | vet ok | lint ok (`golangci-lint run --config .golangci.yml
  ./internal/vertraege/... ./internal/gateway/...`, 0 issues) | test ok
  (`internal/vertraege` komplett gruen, `internal/gateway` komplett gruen,
  `DATABASE_URL` gesetzt, 0 uebersprungene Tests in beiden Paketen) | migration n.a.
  (keine neue Tabelle/Policy) | rls-smoke n.a. (keine Tabellen-/Policy-Aenderung)
- coverage: internal/gateway 71,2 % -> 71,9 % (route_vertraege.go: alle zehn Ziel-Handler
  vorher 0 % Funktionsabdeckung) | internal/vertraege 82,6 % -> 82,6 % (unveraendert —
  die neuen BUG-Tests durchlaufen Code-Pfade, die durch bestehende Erfolgstests bereits
  abgedeckt waren; Wert ist selbst gemessen vor/nach der Aenderung, nicht der
  CI-Bezugswert 69,3 %/n.a. aus der Unit)
- mutations-probe: zwei getrennte Proben, beide per `cp`-Backup zurueckgespielt.
  (a) Gateway: in `saveContractSignatureRequest` das `validate:"required"`-Tag von
  `SignatureData` auf `"omitempty"` geaendert -> `TestHandleSaveContractSignature_MissingSignatureData`
  sofort rot (503 statt 400, echter RPC-Connect-Fehler statt Validierungsfehler).
  (b) Repository: `AND signature_data IS NULL` in die UPDATE-WHERE-Klausel von
  `PostgresRepository.SaveSignature` eingefuegt (das waere der eigentliche Fix) ->
  `TestSaveSignature_OverwritesExistingSignatureWithoutGuard` sofort rot ("contract not
  found" beim zweiten Aufruf, weil der Test genau das aktuelle, ungefixte Verhalten
  dokumentiert). Beide Dateien per `cp`-Sicherungskopie zurueckgespielt, `git diff --stat`
  danach leer, betroffene Pakete erneut komplett gruen.
- verify vorgaenger: sauber (`fc68f158` gegen alle acht Fehlerklassen geprueft — reiner
  neuer Testdateidiff (drei neue/erweiterte Testdateien), kein gRPC-Bypass moeglich, kein
  Stub, kein `.proto` im Diff, kein `RequirePermission` angefasst, keine neue Tabelle, kein
  Wire-Shape, keine neue Route; die neue `fix-chat-search-channel-filter-bypasses-membership`-Unit
  wurde korrekt ans Dateiende gehaengt, nicht eingefuegt.)
- neue-units: fix-vertraege-signature-overwritable-after-signing (Signatur nach dem
  Signieren beliebig oft ueberschreibbar — drittes Modul mit demselben Fehler nach
  rapporte/vermietung aus Lauf 12), fix-vertraege-remove-signed-party-destroys-evidence
  (RemoveParty loescht Beweismittel einer bereits unterzeichnenden Partei ohne Guard)
- offen: beide neuen Fix-Units sind unabhaengige echte Bugs, stehen aber wie von den
  Laufregeln vorgeschrieben am Backlog-Ende. `fix-vertraege-signature-overwritable-after-signing`
  betrifft dieselbe Fehlerklasse wie die beiden bereits offenen rapporte/vermietung-Units —
  falls Luke das gemeinsam anfasst (root-cause-Refactor auf einen Helper statt drei
  Einzelfixes), waere das ein eigener Architekturschnitt, keine Iterations-Erweiterung.

## Iteration 13 — cov-gateway-schichten-shifts-and-templates — done — 2026-08-27 03:07
- commit: e00fc946
- gebaut:
  1. `backend/internal/gateway/route_schichten_test.go` (erweitert) — 33 neue Tests fuer
     alle neun im Scope genannten Handler (HandleGetShift, HandleUpdateShift,
     HandleDeleteShift, HandleListAssignments, HandleUnassignEmployee, HandleApplyTemplate,
     HandleUpdateTemplate, HandleDeleteTemplate, HandleGetShiftStats), jeweils
     ServiceUnavailable/InvalidUUID/ReachesRPC nach etabliertem Muster; vier
     `TestHandleUpdateTemplate_*`/`TestHandleDeleteTemplate_*`-Namen kollidierten mit
     bereits existierenden Tests in `route_rapporte_test.go` (Rapporte hat eigene
     Templates) und wurden auf `TestHandleSchichtenUpdateTemplate_*`/
     `TestHandleSchichtenDeleteTemplate_*` umbenannt.
  2. Schwerpunkt (1) aus dem Scope untersucht: `HandleApplyTemplate`/`Service.ApplyTemplate`
     (service.go:487) IST idempotent — `ShiftExistsForTemplate` prueft vor jedem
     `CreateShift`, ob am selben Tag/Zeitraum/Titel schon eine Schicht existiert, und
     ueberspringt sie dann. Kein Fund, kein Fix-Bedarf. Laeuft NICHT in einer Transaktion
     (jeder Tag ein eigener `CreateShift`-Aufruf), aber weil die Idempotenz ueber die
     Existenzpruefung laeuft und nicht ueber einen DB-Constraint, ist ein Teilausfall
     zwischen zwei Tagen kein Datenintegritaetsproblem, nur ein moeglicher Teil-Erfolg —
     kein neuer Fund.
  3. Schwerpunkt (2) aus dem Scope untersucht: Kollisionspruefung fuer doppelt eingeteilte
     Mitarbeiter FEHLT. `Service.AssignEmployee` (service.go:273) prueft Kapazitaet, ArbZG-
     Ruhezeit (`validateRestPeriod`, bidirektional) und JArbSchG, aber keinen direkten
     Zeit-Overlap: eine neue Schicht, die vollstaendig innerhalb einer bereits zugewiesenen
     Schicht desselben Mitarbeiters liegt, findet in keiner Ruhezeit-Richtung einen Treffer
     (beide Repo-Abfragen suchen benachbarte, nicht ueberlappende Schichten). Verifiziert per
     neuem Test `TestService_AssignEmployee_BUG_OverlappingShiftsNotRejected`
     (`internal/schichten/service_test.go`): eine 09-17-Uhr-Schicht plus eine vollstaendig
     darin verschachtelte 10-14-Uhr-Schicht lassen sich beide demselben Mitarbeiter
     zuweisen, beide Aufrufe liefern `nil` Fehler. Wie von done_when verlangt: Feststellung
     mit Beleg hier im Journal; ob ein Guard gebaut werden soll, ist Lukes Entscheidung —
     als `decide-schichten-assign-employee-overlap-guard` nach `BACKLOG-NEXT.yml` gehaengt
     (kein Fix-Unit-Bedarf in `BACKLOG.yml`, weil done_when das explizit so vorsieht).
  4. Schwerpunkt (3) aus dem Scope untersucht: `HandleGetShiftStats` auf Tenant-Filter und
     leeren Zeitraum. Tenant-Filter ist korrekt (`TenantId` wird immer gesetzt). Der leere
     Zeitraum ist aber ein echter Bug, kein Feature: `GetShiftStatsRequest` traegt
     `optional from`/`to`, `schichten_grpc.go:443-451` liest beide und der Service reicht sie
     bis zum Repository durch — aber `HandleGetShiftStats`
     (`route_schichten.go:735`) liest `r.URL.Query()` ueberhaupt nicht und setzt `from`/`to`
     nie auf der gRPC-Anfrage. Jeder Aufruf liefert Stats ueber die gesamte
     Mandanten-Historie, unabhaengig vom angefragten Zeitraum. Verifiziert per neuem Test
     `TestHandleGetShiftStats_IgnoresFromToQueryParams` (Grenzen eines Gateway-Unit-Tests
     ohne Mock-gRPC-Server: beide Aufrufe, mit und ohne from/to, erreichen denselben
     ServiceUnavailable-Ausgang — das beobachtbare Symptom, nicht die fehlende Wire-Werte).
     Als Coverage-Unit KEIN Verhalten geaendert; Fix-Unit
     `fix-schichten-stats-ignores-date-range-filter` ans Backlog-Ende gehaengt.
  5. HR-Zeiterfassungs-Abgleich aus den Notes: Schichtplanung (`schichten`) und
     HR-Zeiterfassung (`internal/biz/hr/timetracking`) sind vollstaendig getrennte Module —
     kein Code-Pfad verbindet eine geplante `ShiftAssignment` mit einem erfassten
     `TimeEntry`. Feststellung, kein Fund (kein bestehendes Feature, das kaputt waere).
- gate: build ok (`./internal/schichten/... ./internal/gateway/... ./cmd/schichten/...
  ./cmd/gateway/...`) | vet ok | lint ok (`golangci-lint run --config .golangci.yml
  ./internal/schichten/... ./internal/gateway/...`, 0 issues) | test ok
  (`internal/schichten` komplett gruen, `internal/gateway` komplett gruen, `DATABASE_URL`
  gesetzt, 0 uebersprungene Tests in beiden Paketen) | migration n.a. (keine neue
  Tabelle/Policy) | rls-smoke n.a. (keine Tabellen-/Policy-Aenderung)
- coverage: internal/gateway 71,9 % -> 72,4 % (route_schichten.go: alle neun Ziel-Handler
  vorher 0 % Funktionsabdeckung) | internal/schichten 79,4 % -> 79,4 % (unveraendert — der
  neue BUG-Test durchlaeuft ausschliesslich bereits von bestehenden Erfolgstests abgedeckte
  Codepfade in AssignEmployee; Wert ist selbst gemessen vor/nach der Aenderung per Stash der
  neuen Testdateien, nicht der CI-Bezugswert 69,3 %/n.a. aus der Unit)
- mutations-probe: zwei getrennte Proben, beide per `cp`-Backup zurueckgespielt.
  (a) Gateway: in `HandleUnassignEmployee` den zweiten `validateUUIDParam`-Aufruf von
  `"employee_id"` auf `"id"` geaendert (doppelter Parameter-Name) ->
  `TestHandleUnassignEmployee_InvalidEmployeeIDUUID` sofort rot (503 statt 400, echter
  RPC-Connect-Fehler statt Validierungsfehler, weil beide UUIDs jetzt als gueltig gelesen
  wurden). (b) Service: in `AssignEmployee` den Kapazitaets-Vergleich von `count >=
  *shift.Capacity` auf `count > *shift.Capacity` geaendert (Off-by-one) ->
  `TestService_AssignEmployee_CapacityExceeded_Rejected` sofort rot (kein Fehler statt
  `ErrShiftFull` bei exakt erreichter Kapazitaet); der neue BUG-Test blieb dabei gruen (er
  pinnt ein anderes, unabhaengiges Verhalten). Beide Dateien per `cp`-Sicherungskopie
  zurueckgespielt, `git diff --stat` danach leer, betroffene Pakete erneut komplett gruen.
- verify vorgaenger: sauber (`76175de2` gegen alle acht Fehlerklassen geprueft — reiner
  neuer Testdateidiff (drei neue/erweiterte Testdateien in vertraege/gateway), kein
  gRPC-Bypass moeglich, kein Stub, kein `.proto` im Diff, kein `RequirePermission`
  angefasst, keine neue Tabelle, kein Wire-Shape, keine neue Route; beide neuen Fix-Units
  wurden korrekt ans Dateiende gehaengt, nicht eingefuegt.)
- neue-units: fix-schichten-stats-ignores-date-range-filter (HandleGetShiftStats liest
  from/to-Query-Parameter nie, obwohl die ganze Kette bis zum Repository sie bereits
  unterstuetzt — BACKLOG.yml, todo); decide-schichten-assign-employee-overlap-guard
  (Produktentscheidung, ob AssignEmployee eine echte Overlap-Pruefung braucht —
  BACKLOG-NEXT.yml, blocked)
- offen: `fix-schichten-stats-ignores-date-range-filter` ist ein reiner Handler-Nachtrag
  (Muster liegt in derselben Datei bei `HandlePublishShifts` vor) und sollte zeitnah baubar
  sein. `decide-schichten-assign-employee-overlap-guard` braucht Lukes Entscheidung, ob und
  wie hart der Guard greifen soll, inkl. SwapRequest-Genehmigung als moeglicher zweiter
  Angriffspunkt fuer dieselbe Ueberschneidung.

## Iteration 14 — fix-fuhrpark-vehicle-routes-daily-km-always-zero — done — 2026-08-27 03:08
- commit: 220fd564
- gebaut:
  1. `internal/fuhrpark/postgres_repository.go` — `GetVehicleRoutes` setzte `DailyKm` hart auf
     `0` (Kommentar `// calculated from positions if needed`), obwohl die Positionsfolge des
     Tages (`ORDER BY recorded_at ASC`) bereits vorlag. Neue Funktion `sumHaversineKm`
     (Standard-Haversine-Formel, Erdradius 6371 km) summiert die Distanz zwischen
     aufeinanderfolgenden Positionen; `DailyKm: sumHaversineKm(positions)` ersetzt die
     Hartkodierung. Keine neue Query, keine `.proto`-Aenderung (Feld existiert im Proto
     bereits), reine Repository-Berechnung.
  2. `internal/fuhrpark/postgres_repository_gap_test.go` — bestehenden Test
     `TestGpsPositions_IngestGetAndRouteAggregation` um einen Assert ergaenzt: die drei
     geseedeten Positionen (52.5/13.4, 52.6/13.5, 52.7/13.6, je 0.1 Grad Lat/Lng
     auseinander) ergeben zwei Haversine-Hops von je ~13 km, `DailyKm` muss also im Fenster
     20-30 km liegen. Kein neuer Testfall noetig, das bestehende Fixture deckt es ab.
- gate: build ok (`./internal/fuhrpark/... ./internal/gateway/... ./internal/server/...
  ./cmd/fuhrpark/... ./cmd/gateway/...`) | vet ok | lint ok (`golangci-lint run --config
  .golangci.yml ./internal/fuhrpark/... ./internal/gateway/...`, 0 issues) | test ok
  (`internal/fuhrpark` komplett gruen inkl. DB-Test, `internal/gateway` komplett gruen,
  `DATABASE_URL` gesetzt, 0 uebersprungene Tests) | migration n.a. (keine neue
  Tabelle/Policy) | rls-smoke n.a. (keine Tabellen-/Policy-Aenderung; bestehender Test
  deckt den Tenant-Join in `GetVehicleRoutes` bereits als RLS-Smoke ab)
- coverage: n.a. (Bugfix laut Unit-Definition, kein Coverage-Ziel; internal/fuhrpark
  Gesamtpaket 81,5 % nach der Aenderung, `go tool cover -func` selbst gemessen)
- mutations-probe: `DailyKm: sumHaversineKm(positions)` per `cp`-Backup temporaer zurueck auf
  `DailyKm: 0` gesetzt -> `TestGpsPositions_IngestGetAndRouteAggregation` sofort rot
  (`expected DailyKm around 26 km for two ~13km hops, got 0`). Backup zurueckgespielt,
  `git diff --stat` danach leer, `internal/fuhrpark/...` erneut komplett gruen.
- verify vorgaenger: sauber (`e00fc946` geprueft — reiner Testdateidiff in
  `route_schichten_test.go`/`service_test.go`, kein gRPC-Bypass moeglich, kein Stub, kein
  `.proto` im Diff, kein `RequirePermission` angefasst, keine neue Tabelle, kein Wire-Shape,
  keine neue Route)
- neue-units: keine
- offen: keine

## Iteration 15 — cov-gateway-produktion-planning-and-capacity — done — 2026-08-27 03:12
- commit: f5b86621
- gebaut:
  1. `internal/gateway/route_produktion_remaining_test.go` (neu) — die acht in der Unit
     genannten, bis dato ungetesteten Handler aus `route_produktion.go`: HandleDeleteOrder,
     HandleGetMaterialAvailability, HandleListMachineBookings, HandleUpdateMachineBooking,
     HandleDeleteMachineBooking, HandleGetPlan, HandleUpdatePlan, HandleGetCapacityOverview.
     Je Handler ServiceUnavailable/InvalidUUID/InvalidJSON nach dem etablierten Muster aus
     `route_produktion_orders_test.go`; `HandleGetCapacityOverview` zusaetzlich mit einem
     eigenen Test fuer die fehlende `machine_id`-Query-Param-Pruefung, weil das die einzige
     Validierung in dieser Datei ist, die VOR der RPC-Grenze im Gateway selbst passiert.
  2. `internal/gateway/route_produktion_ext_handlers_test.go` (neu) — alle 17 Handler aus
     `route_produktion_ext.go` (BOMs, WorkSteps, Machines, QualityChecks), die bislang nur
     ueber `TestProduktionExtRoutes_Registered`/`_MatchAgainstOrderSubtree` (Routing, kein
     Handler-Aufruf) beruehrt waren. Gleiches Muster: ServiceUnavailable, ungueltige UUID am
     jeweils richtigen Chi-Param-Namen (`bomId`/`stepId`/`machineId`/`checkId` — nicht `id`),
     Pflichtfeld-Validierung wo vorhanden (`product_name`/`sku`/`name`/`inspector`,
     `order_id`-UUID bei HandleCreateQualityCheck).
  3. Bug-Suche laut Scope, kein Fund noetig als Fix: `Service.CreateMachineBooking` und
     `UpdateMachineBooking` (`internal/produktion/service.go:412-500`) fuehren die
     Konflikt-Pruefung bereits ATOMAR unter `pg_advisory_xact_lock`
     (`CreateBookingWithLock`) bzw. per `FindConflictingBooking` mit Selbst-Ausschluss aus —
     derselbe Ressourcen-Race wie bei Kalenderressourcen/Vermietung ist hier bereits sauber
     gegen TOCTOU abgesichert (Kommentar im Code verweist explizit darauf). Kein
     Nachbau-Bedarf, im Gegensatz zur Vermutung in der Unit-Notiz.
  4. `HandleGetOrder`/`HandleDeleteOrder` bei laufender Produktion: keine gateway-lokale
     Logik, die Referenz-Integritaets-Pruefung liegt service-seitig (nicht Teil dieser Unit,
     Handler leitet nur durch) — als Feststellung im Journal, kein Fund.
- gate: build ok (`./internal/produktion/... ./internal/gateway/... ./cmd/produktion/...
  ./cmd/gateway/...`) | vet ok | lint ok (`golangci-lint run --config .golangci.yml
  ./internal/produktion/... ./internal/gateway/...`, 0 issues) | test ok (`internal/gateway`
  komplett gruen inkl. `TestOpenAPIRouteDrift` 836 Routen gegen 838 Spec-Pfade, `DATABASE_URL`
  gesetzt, 0 uebersprungene Tests; `internal/produktion` komplett gruen) | migration n.a.
  (keine neue Tabelle/Policy) | rls-smoke n.a. (keine Tabellen-/Policy-Aenderung, reiner
  Testdateidiff)
- coverage: internal/gateway 72,4 % -> 73,4 % (selbst gemessen vor/nach per temporaerem
  Entfernen der beiden neuen Testdateien, nicht der CI-Bezugswert 69,3 % aus der Unit — der
  ist seit Iteration 13/14 durch andere gateway-Units bereits ueberholt)
- mutations-probe: `HandleGetBOM` per `cp`-Backup temporaer von `validateUUIDParam(w, r,
  "bomId")` auf `validateUUIDParam(w, r, "id")` geaendert (falscher Chi-Param-Name) ->
  `TestHandleGetBOM_InvalidUUID` sofort rot (`error = "invalid id", want to contain "invalid
  bomId"`). Backup zurueckgespielt, `git diff --stat` danach leer, `internal/gateway/...`
  erneut komplett gruen.
- verify vorgaenger: sauber (`220fd564` geprueft — reine Repository-Berechnung
  (`sumHaversineKm`), kein gRPC-Bypass, kein Stub, kein `.proto` im Diff, kein
  `RequirePermission` angefasst, keine neue Tabelle, kein Wire-Shape, keine neue Route)
- neue-units: keine
- offen: keine

## Iteration 16 — cov-settings-repository-and-service — done — 2026-08-27 03:18
- commit: 4c6163b4
- gebaut:
  1. `internal/settings/postgres_repository_db_test.go` (neu) — DB-Tests fuer alle
     zuvor bei 0,0 % stehenden Repository-Methoden: `IsAdmin` (inkl. Cross-Tenant-Leck-Check
     ueber `u.tenant_id = $2`), `ListModuleLeads`/`GetModuleLead`/`IsModuleLead`/
     `ListLeadModulesForUser` (beide dynamischen WHERE-Zweige `$2`/`$3`),
     `ListModuleGrants` (Join gegen `users` fuer `user_name`, beide Filterzweige,
     Cross-Tenant), `CountGrantsByModule` (Modul ohne Grant fehlt in der Map statt 0),
     `GetTenantSubscription` (eigener Tenant liest Defaults `cosmi`/`standard`/`active`,
     fremder Tenant via RLS auf `tenants` -> `ErrNotFound`).
  2. Zwei Nebenlaeufigkeits-Tests zu Scope-Punkt (3), der erste mit ZWEI echt
     gehaltenen Verbindungen (`pool.Begin` + Goroutine), nicht einer warmen:
     `TestReplaceUserSettings_GenuinelyConcurrentAdditionSurvives` haelt tx1's DELETE
     offen, laesst parallel eine zweite, unabhaengige Verbindung einen neuen Key
     einfuegen und committen, dann tx1 seinen eigenen Insert + Commit — der
     neue Key ueberlebt, weil DELETE nur zum Ausfuehrungszeitpunkt sichtbare
     (committete) Zeilen erfasst. `TestReplaceUserSettings_FullReplaceDropsKeysAddedAfterItsSnapshot`
     zeigt die Kehrseite rein sequenziell: ein VOR dem Replace-Aufruf committeter
     Key wird von dessen ungefiltertem DELETE mitgeloescht — das ist das
     dokumentierte PUT-Vollersatz-Verhalten aus `ReplaceUserSettings`' eigenem
     Kommentar (gespiegelt von `auth.PostgresRepository.SetUserOverrides`), keine
     Lost-Update-Race und daher kein Fix-Bedarf.
  3. `internal/settings/service_test.go` (erweitert) — RBAC-Tests fuer die zuvor
     bei 0,0 % stehenden Service-Methoden `GrantModuleAccess`/`RevokeModuleAccess`/
     `BulkRevokeModuleAccess` (admin-only, `ErrNotAdmin` fuer Nicht-Admins,
     `ErrInvalidModuleID` inkl. eines einzelnen leeren `ModuleID` im Bulk-Batch),
     `ListModuleGrants` (Pass-through), sowie `GetUserSettings`/`PutUserSettings`/
     `ReplaceUserSettings` (Validierung `ErrInvalidModuleID`/`ErrInvalidKey`,
     Replace loescht nicht mitgesendete Keys).
  4. Scope-Punkt (2) (Default-Werte/Steuersaetze) gegengeprueft, kein Fund: dieses
     Paket ist ein generischer Key-Value-Store ohne eigene Default-Logik pro Key —
     `GetResolvedSettings` liefert bei fehlender Einstellung eine leere Liste
     (bereits durch `TestGetResolvedSettings_EmptyWhenNothingSet` belegt), nie
     einen stillen Nullwert. Ein gefaehrlicher Steuersatz-Default waere in dem
     Modul zu suchen, das einen `taxRate`-Key liest und selbst einen Fallback
     anwendet — das ist ausserhalb von `internal/settings`, keine eigene Unit
     angelegt mangels konkretem Fund.
- gate: build ok (`./internal/settings/... ./internal/gateway/... ./cmd/gateway/...`) |
  vet ok | lint ok (`golangci-lint run --config .golangci.yml ./internal/settings/...`,
  0 issues) | test ok (`internal/settings` komplett gruen, 58 Tests, `DATABASE_URL`
  gesetzt, 0 uebersprungene Tests) | migration n.a. (keine neue Tabelle/Policy) |
  rls-smoke ok (in den neuen DB-Tests selbst: Cross-Tenant-Reads fuer IsAdmin,
  ListModuleLeads, ListModuleGrants, CountGrantsByModule, GetTenantSubscription
  liefern durchgehend leer/ErrNotFound)
- coverage: internal/settings 60,3 % -> 82,8 % (selbst gemessen vor/nach per
  `go tool cover -func`, deckt sich mit dem CI-Bezugswert aus der Unit)
- mutations-probe: `IsAdmin`-Query per `cp`-Backup temporaer von
  `WHERE ur.user_id = $1 AND u.tenant_id = $2 AND r.name = 'admin'` auf
  `WHERE ur.user_id = $1 AND r.name = 'admin'` gekuerzt (Tenant-Filter entfernt,
  Scan bekommt dadurch ein Argument zu viel) -> `TestIsAdmin_TrueForAdminRoleFalseOtherwise`
  sofort rot (`expected 1 arguments, got 2`). Backup zurueckgespielt, `git diff --stat`
  danach leer, `internal/settings/...` erneut komplett gruen.
- verify vorgaenger: sauber (`f5b86621` geprueft — reiner Testdateidiff in
  zwei neuen `route_produktion_*_test.go`-Dateien, kein gRPC-Bypass, kein Stub,
  kein `.proto` im Diff, kein `RequirePermission` angefasst, keine neue Tabelle,
  kein Wire-Shape, keine neue Route)
- neue-units: keine
- offen: keine

## Iteration 17 — cov-dialer-service-call-session-paths — done — 2026-08-27 03:29
- commit: 7379f99b
- gebaut:
  1. ECHTER BUG GEFIXT: `Service.refreshCampaignCounts` (`service.go:1274`, aufgerufen von
     `LogCallOutcome` und `CompleteWrapUp`) war ein reiner No-Op — es loggte nur eine
     Debug-Zeile und rief `campaigns.UpdateCampaignCounts` nie auf, obwohl der eigene
     Kommentar genau das als "preferred path" beschrieb. Damit blieben
     `dialer_campaigns.contact_count`/`completed_count` nach dem ersten
     `AddContactsToCampaign`-Aufruf fuer immer stehen, unabhaengig davon, wie viele
     Outcomes danach geloggt wurden. Fix: `refreshCampaignCounts` nimmt jetzt zusaetzlich
     `tenantID`, loest die `campaignID` ueber `GetCampaignContactByID` auf (bestehendes
     Muster aus `checkCampaignCompleteByContact`) und ruft `campaigns.UpdateCampaignCounts`
     mit der echten ID auf. Beide Aufrufer (`LogCallOutcome`, `CompleteWrapUp`) angepasst.
  2. Bug-Suche (1) Idempotenz: Anruf-Ergebnisse selbst sind NICHT idempotent auf
     Service-Ebene (kein Idempotency-Key-Check in `LogCallOutcome`), aber `DialerRoutes`
     wird ueber `cmd/gateway/main.go:329` (`reg.RegisterRoutes(r, authWithIdempotency)`) mit
     der generischen `middleware.Idempotency`-Kette registriert — dieselbe wie alle anderen
     POST-Routen. Produktion faehrt im Default `WarnMode` (`cmd/gateway/main.go:199`, nur
     `IDEMPOTENCY_MODE=hard` schaltet den 400-Block scharf): ein Client, der den
     `Idempotency-Key`-Header sendet, ist dedupliziert, einer der ihn weglaesst, wird nur
     geloggt, nicht geblockt. Das ist eine globale, bereits generisch getestete
     (`internal/middleware/idempotency_test.go`) Produktionsentscheidung, keine
     dialer-spezifische Luecke — keine neue Unit noetig.
  3. Bug-Suche (2) Sitzungslebenszyklus: kein automatischer Timeout/Reaper fuer eine nie
     beendete Sitzung (grep nach stale/abandon/timeout in `internal/dialer/*.go` und
     `cmd/dialer/main.go` liefert nichts). Der einzige Ausweg ist der manuelle
     `RequeueContact`-Aufruf, dessen SQL keinen Status-WHERE-Filter hat und daher jeden
     Status zuruecksetzt — belegt durch neuen DB-Test
     `TestCampaignRepository_RequeueContact_UnsticksAbandonedInProgressContact`
     (`queue_and_list_test.go`). Doc-Kommentar auf `Service.RequeueContact` korrigiert (er
     behauptete faelschlich "skipped or callback"). Fehlender automatischer Reaper als eigene
     Unit ans Backlog-Ende gehaengt (`feat-dialer-stale-in-progress-contact-reaper`) —
     Threshold-Entscheidung gehoert Luke, kein automatischer Fix in dieser Unit.
  4. Bug-Suche (3) Cross-Tenant-Kampagnen-Zuordnung: `AddContactsToCampaign` loest jeden
     importierten Kontakt ueber `crmBridge.GetContactDetails(ctx, cid)` auf; der
     `GRPCCRMBridge` reicht den Caller-`ctx` unveraendert an den CRM-Microservice weiter,
     dessen `GetContact`-Handler tenant-gescoped per RLS ist — ein fremder Kontakt kommt als
     gRPC-NotFound zurueck und wird in der Service-Schleife uebersprungen (skipped++), nie
     hinzugefuegt. Neuer Test `TestAddContactsToCampaign_FailsClosedOnForeignTenantContact`
     dokumentiert dieses Fail-Closed-Verhalten explizit (mechanisch identisch mit der
     bestehenden `BridgeFetchFailSkips`, aber mit klarem Sicherheits-Intent benannt).
  5. Bug-Suche (4) Retention-Abdeckung: `DialerCallRetentionHandler`
     (`internal/security/gdpr/retention_dialer_chat.go`) deckt `dialer_call_sessions`
     zeitbasiert ab (Delete cascadet `dialer_call_events` per FK, Anonymize leert
     notes/next_action). Die beiden in der Backlog-Vorbereitung genannten PII-Spalten
     `dialer_call_sessions.notes/next_action` und `dialer_campaign_contacts.notes` sind
     bereits durch `crm/consent/scrub.go:99-118` (Abhaengigkeit
     `feat-scrub-dependent-pii-dialer-tables`, laut Backlog `done`) beim
     Kontakt-Erasure-Pfad abgedeckt — keine Luecke, kein Fund.
  6. Coverage-Luecken bei 0,0 % geschlossen: `CreateCampaign` (leerer Name, Mode-Default,
     `AssignedAgentIDs`-nil-Normalisierung, `EnsureDefaults`-Fehler non-fatal),
     `GetCampaignForTenant`, `ListCampaigns`, `UpdateCampaign` (Not-Found, Not-Draft,
     leerer Name, Feld-Patches), `SetAgentStatus`/`GetAgentStatus`/`GetCampaignAgents`
     (neuer `newTestHarnessWithAgentStore`-Helper mit echtem miniredis-`AgentStatusStore`,
     da der Typ eine konkrete Struct um `*redis.Client` ist, kein Interface),
     `GetAgentDashboard`, `GetSupervisorOverview` (Totals-Aggregation, Namensaufloesung,
     Kampagnennamen-Resolve, stale-Agent-Default-Offline-Pfad), `GetContactCalls`.
  7. Mocks in `service_test.go` erweitert um konfigurierbare Fehler/Ruecksprungwerte
     (`mockCampaignRepo.listErr/updateErr/agentStatsErr/updateCountsCalls`,
     `mockAgentStatusRepo` mit echten Feldern statt leerer Struct,
     `mockCallRepo.listCallsByContact*/recentCalls*/tenantCallsToday*/agentCallsToday*` etc.)
     — vorher waren diese Pfade *nicht mockbar*, das war der eigentliche Grund fuer die
     0,0 %-Luecken, nicht fehlender Testwille.
- gate: build ok (`./internal/dialer/... ./internal/gateway/... ./cmd/dialer/... ./cmd/gateway/...`) |
  vet ok | lint ok (`golangci-lint run --config .golangci.yml ./internal/dialer/...`, 0 issues) |
  test ok (`internal/dialer` komplett gruen, `DATABASE_URL` gesetzt, 0 uebersprungene Tests,
  RLS-Tests liefen real) | migration n.a. (keine neue Tabelle/Policy) | rls-smoke n.a.
  (keine Schema-Aenderung; Tenant-Isolation der bestehenden Tabellen bereits durch
  `tenant_write_test.go`/`rls_test.go` abgedeckt und in dieser Iteration nicht angefasst)
- coverage: internal/dialer 65,9 % -> 77,5 % (selbst gemessen per `go tool cover -func`
  vor/nach; deckt sich mit dem CI-Bezugswert 65,9 % aus der Unit)
- mutations-probe: `refreshCampaignCounts` per `cp`-Backup auf den alten No-Op-Body
  zurueckgesetzt (kein Aufruf von `GetCampaignContactByID`/`UpdateCampaignCounts` mehr) ->
  `TestLogCallOutcome_RefreshesCampaignCounts` und `TestCompleteWrapUp_RefreshesCampaignCounts`
  sofort rot ("expected UpdateCampaignCounts to be called once, got 0 calls"). Backup
  zurueckgespielt, `git diff --stat` gegen HEAD zeigt wieder nur den beabsichtigten Fix,
  `internal/dialer/...` erneut komplett gruen.
- verify vorgaenger: sauber (`4c6163b4` geprueft — reiner Testdateidiff
  (`postgres_repository_db_test.go` neu, `service_test.go` erweitert) in `internal/settings`,
  kein gRPC-Bypass, kein Stub, kein `.proto` im Diff, kein `RequirePermission` angefasst,
  keine neue Tabelle, kein Wire-Shape, keine neue Route)
- neue-units: feat-dialer-stale-in-progress-contact-reaper (fehlender automatischer
  Aufraeumpfad fuer nie beendete Anrufsitzungen — Threshold-Entscheidung gehoert Luke)
- offen: keine

## Iteration 18 — cov-server-gobd-archive-grpc-handlers — done — 2026-08-27 03:44
- commit: 731eb529
- gebaut:
  1. Neue Testdatei `backend/internal/server/biz_grpc_gobd_archive_test.go` (~700 Zeilen) fuer
     die sechs GoBD-Belegarchiv-Handler, die bei exakt 0,0 % standen. Alle sechs stehen
     jetzt bei 100,0 % Statement-Coverage (`ArchiveDocument`, `ArchiveInvoiceDocument`,
     `GetGobdDocument`, `ListGobdDocuments`, `DownloadGobdDocument`, `AddDocumentAnnotation`),
     ebenso die drei Mapping-Helfer `toProtoGobdDocument`, `toProtoGobdEventList` und
     `mapGobdArchiveError`.
  2. Test-Seam: `gobdArchiveSvc` ist ein konkretes `*gobdarchive.Service`, kein Interface —
     die Doubles sitzen deshalb eine Schicht tiefer (`stubGobdRepo` als
     `gobdarchive.Repository`, `stubGobdStore` als `chatfile.FileStore`) hinter einem echten
     `gobdarchive.NewService`. Damit laufen die Service-Invarianten (SHA-256 via TeeReader,
     Retention = 31.12. des Jahres+10, Event-Append) real im Testpfad mit, statt wegmockt zu
     werden. `stubGobdRepo.GetByID` bildet das RLS-Verhalten der Postgres-Implementierung ab:
     fremder Tenant ist von "existiert nicht" ununterscheidbar.
  3. Bug-Suche (1) Tenant-Bindung jedes Handlers: **kein Fund, alle vier Lesepfade sind
     fail-closed.** Belegt durch je einen Test — `GetGobdDocument` (fremdes Dokument ->
     NotFound, Response nil), `ListGobdDocuments` (nur eigener Tenant in der Liste, `total`
     zaehlt nur eigene), `DownloadGobdDocument` und `AddDocumentAnnotation` (beide NotFound).
  4. Bug-Suche (2) `DownloadGobdDocument`: der presignte MinIO-Link traegt selbst keine
     Autorisierung, wer ihn hat, hat den Beleg. Entscheidend ist deshalb, dass er fuer einen
     fremden Beleg gar nicht erst entsteht. Der Handler laedt vor dem Presignen ueber
     `GetByID` (fuer Dateiname/MIME-Typ) und faellt dort schon aus — Test assertet hart
     `store.presignCalls == 0` und zusaetzlich, dass **kein** `access`-Event auf dem fremden
     Dokument landet. Kein Fund.
  5. Bug-Suche (3) `AddDocumentAnnotation` laesst den archivierten Beleg unberuehrt: belegt.
     Der Test nimmt vor dem Aufruf eine Wertkopie des `models.GobdDocument` und vergleicht sie
     danach feldweise (`assert.Equal(before, *repo.docs[id])`); die Anmerkung landet
     ausschliesslich als `annotation`-Event in `gobd_document_events`. Kein Fund.
  6. Bug-Suche (4) Hash-Pruefung beim Lesen: **FUND, siehe `neue-units`.** Der SHA-256 wird
     beim Schreiben per `io.TeeReader` berechnet (`gobdarchive/service.go:95-113`) und
     danach **nie wieder geprueft**. `GetByID`, `List` und `GetDownloadURL` reichen die
     gespeicherte Spalte nur durch, `toProtoGobdDocument` schreibt sie ins Proto
     (`server/biz_grpc_gobd_archive.go:291`). Ein grep ueber das ganze Backend nach
     `sha256`/`Sha256` findet ausser Schreibpfad, Repository-Spaltenliste und Proto-Mapping
     **keine einzige Vergleichsstelle**. Zweiter Beleg: `models.GobdEventTypeIntegrityCheck`
     = `"integrity_check"` (`internal/models/gobd.go:60`) existiert im Datenmodell und wird
     von **keiner** Zeile Go-Code emittiert (`grep -rn "IntegrityCheck" --include=*.go .`
     liefert genau diese eine Definitionszeile). Das Schema hat die Pruefung vorgesehen, der
     Code hat sie nie bekommen. Wirkung: wird ein Objekt im MinIO-Bucket ausgetauscht, faellt
     das nirgends auf. Migration 000315 (`REVOKE UPDATE, DELETE`) haertet die DB-Zeile, nicht
     die Bytes im Objektspeicher.
  7. Weitere abgedeckte Verhaltensweisen, die vorher unbelegt waren: leeres `content` wird
     abgelehnt, **bevor** Bytes den Store erreichen; `source_invoice_id` kommt als leerer
     String statt als Null-UUID zurueck, wenn kein Bezug besteht; leere Liste serialisiert
     als `[]`, nie `null`; Default `page=1`/`per_page=50` in der Response; alle
     Filter (doc_type, source_invoice_id, date_from/date_to als YYYY-MM-DD, page, per_page)
     erreichen die Repository-Ebene unveraendert; fehlendes `metadata` wird als `{}`
     serialisiert; `ArchiveInvoiceDocument` archiviert nur gesperrte Rechnungen
     (FailedPrecondition sonst) und legt bei Render-Fehler nichts an.
- gate: build ok (`./internal/server/... ./internal/biz/gobdarchive/... ./internal/gateway/...
  ./cmd/gateway/...`) | vet ok | lint ok (`golangci-lint run ./internal/server/...`, 0 issues) |
  test ok (`internal/server` 2,9 s gruen, `internal/biz/gobdarchive` gruen, `internal/gateway`
  gruen; `DATABASE_URL` als `kmuhub_app` gesetzt, `go test -v | grep -c -- "--- SKIP"` = **0**,
  also kein einziger uebersprungener DB-Test) | migration n.a. (reine Testdatei, kein Schema) |
  rls-smoke n.a. (keine Tabelle/Policy angefasst; die Tenant-Isolation ist hier auf
  Handler-Ebene per Test belegt, nicht auf Policy-Ebene veraendert)
- coverage: internal/server 71,4 % -> 72,1 % (selbst gemessen per `go tool cover -func` vor und
  nach der Aenderung; der Bezugswert der Unit war dateibezogen —
  `biz_grpc_gobd_archive.go` 0,0 %, 136 Statements — diese Datei steht jetzt bei 100,0 % ueber
  alle neun Funktionen)
- mutations-probe: `toProtoGobdDocument` per `cp`-Backup auf `Sha256: ""` gesetzt (statt
  `doc.SHA256`) -> `TestArchiveDocument/success_stores_the_document_under_the_caller_tenant_with_a_matching_hash`
  sofort rot (erwartet `52d51394…a7d3`, bekommen `""`) und
  `TestGetGobdDocument/success_returns_document_plus_audit_trail` ebenfalls rot (erwartet
  `deadbeef`). Backup zurueckgespielt, `git diff --stat -- internal/server/biz_grpc_gobd_archive.go`
  leer, Paket erneut gruen. Damit ist belegt, dass die Tests die Hash-Weitergabe wirklich
  pruefen und nicht nur Zeilen ausfuehren.
- verify vorgaenger: sauber (`7379f99b` geprueft — der einzige Nicht-Test-Diff ist
  `internal/dialer/service.go`: `refreshCampaignCounts` war ein dokumentierter No-Op mit
  `slog.Debug` und `return nil` und ruft jetzt real
  `GetCampaignContactByID` + `UpdateCampaignCounts`; beide Aufrufer wurden auf die neue
  Signatur mit `tenantID` gezogen. Das ist die Aufloesung eines Stubs, nicht das Einbauen
  eines neuen. Kein gRPC-Bypass (Service-interner Repository-Aufruf, kein Gateway-Handler),
  kein `.proto` im Diff, kein `RequirePermission` angefasst, keine neue Tabelle, keine neue
  Route, Wire-Shape unveraendert.)
- neue-units: fix-gobd-archive-hash-never-verified (SHA-256 wird geschrieben und nie
  verifiziert; Scope der Unit ist bewusst nur die Service-Methode plus Tests, **nicht** ihr
  Aufrufer)
- offen: **Entscheidung fuer Luke — wo die Integritaetspruefung ausgeloest wird.** Die Bytes
  fliessen heute nirgends durch die App zurueck: `GetDownloadURL` presigned direkt gegen
  MinIO, der Prozess sieht den Inhalt nach dem Upload nie wieder. Ein Pruefpfad braucht daher
  einen eigenen Ausloeser — periodischer Worker (analog `security/gdpr/retention_scheduler.go`
  mit Advisory-Lock), RPC auf Zuruf, oder beim GoBD-Export. Das haengt daran, wie teuer ein
  Vollscan des Buckets sein darf, und ist deshalb bewusst aus der neuen Unit ausgeklammert.

## Iteration 19 — cov-server-websocket-connection-and-message-loop — done — 2026-08-27 03:53
- commit: dc73da76
- gebaut: `backend/internal/server/websocket_connection_test.go` (neu, 18 Tests) plus zwei
  kleine Nebenaenderungen: ein Testfeld `revalidateInterval` an `WebSocketHub` (Null = die
  Produktionskonstante von 5 Minuten, `lean:`-Marker gesetzt) und `userTenants` im
  Test-Helper `newTestHub`, wo die Map bisher fehlte und `registerUserTenant` in eine
  nil-Map geschrieben haette. Abgedeckt sind jetzt:
  1. **Verbindungsaufbau** ueber `HandleWebSocket` end-to-end auf einem echten
     `http.Server`: kein Token -> 401, kaputter Token -> 401, abgelaufener Token -> 401 vor
     `Accept`, `maxConnsPerUser` erreicht -> 429, Gast-Token ohne Gast-Service -> 503,
     ungueltiges Gast-Token -> 401, gueltiges Token -> registriert und `tid`-Claim landet in
     `userTenants`.
  2. **Nachrichtenschleife**: unbekannter Typ -> Error-Frame, leeres Kontingent ->
     "rate limit exceeded", leere `channel_id` -> "channel_id is required".
  3. **Verbindungsabbau**: harter Abbruch, Freigabe von Verbindung, Tenant-Cache,
     Rate-Limiter, Channel-Mitgliedschaft, plus Goroutine-Zaehlung vor/nach.
  4. **Token-Widerruf end-to-end**: laeuft der Token waehrend der Sitzung ab, schliesst der
     Server mit `StatusPolicyViolation` — jetzt ueber den vollen `HandleWebSocket`-Pfad
     belegt, nicht mehr nur an der isolierten Schleife.
  5. **Rate-Limiter-Reichweite**: drei Tests belegen "pro Nutzer" — nicht global (ein
     leerer Bucket bremst keinen zweiten Nutzer aus), nicht pro Verbindung (zwei
     Verbindungen desselben Nutzers teilen einen Bucket, genau einer in der Map).
  6. **Tenant-Trennung beim Broadcast**: der Hub selbst filtert NICHT nach Tenant —
     `broadcastToChannel` faechert an jeden Eintrag in `channelMembers` auf, ohne
     `userTenants` je zu befragen. Der einzige Durchsetzungspunkt ist der
     `GetChannel`-Gate in `handleSubscribeChannel`. Beide Haelften sind jetzt festgenagelt:
     ein Aufrufer aus fremdem Tenant kommt nicht in die Subscriber-Menge und bekommt den
     Broadcast nicht; und wer ohne diesen Gate hineingeschrieben wird, bekommt ihn sehr wohl.
  7. **Gast-Verbindung**: Aufbau, Registrierung unter der Session-ID, Kanal-Zuordnung und
     Freigabe beim Schliessen.
- gate: build ok (`./internal/server/... ./internal/gateway/... ./cmd/gateway/...`) |
  vet ok | lint ok (`golangci-lint run ./internal/server/...`, 0 issues; ein
  `bodyclose`-Befund war echt und wurde behoben, indem der Helfer nur noch den Status-Code
  zurueckgibt und den Body selbst schliesst) | test ok (`internal/server` 7,0 s gruen,
  `internal/server/response` gruen, `internal/gateway` gruen; `DATABASE_URL` als
  `kmuhub_app` gesetzt, `go test -v | grep -c -- "--- SKIP"` = **0** bei 4600 `--- PASS`) |
  migration n.a. (keine) | rls-smoke n.a. (keine Tabelle, keine Policy angefasst)
- coverage: internal/server 72,1 % -> 72,6 % (selbst gemessen, Testdatei einmal
  herausgenommen und einmal drin). Dateibezogen, weil der `coverage_start:`-Wert der Unit
  dateibezogen war: `websocket.go` **43,0 % -> 64,8 %** (215 -> 324 von 500 Statements).
  Der Unit-Bezugswert nannte 38,3 % aus dem CI-Artefakt; mein Vorher-Wert lag bei 43,0 %,
  weil `websocket_presence_subscribe_test.go` nach jenem Artefakt dazugekommen ist.
- mutations-probe: `HandleWebSocket` per `cp`-Backup auf
  `h.registerUserTenant(userID, "")` gesetzt (statt `claims.TenantID`) ->
  `TestHandleWebSocket_ValidToken_RegistersConnectionAndTenant` sofort rot. Backup
  zurueckgespielt, `git diff` auf `websocket.go` zeigt nur noch die elf Zeilen des
  `revalidateInterval`-Felds, Paket erneut gruen.
- verify vorgaenger: sauber (`731eb529` geprueft — der Diff besteht aus genau einer neuen
  Testdatei `internal/server/biz_grpc_gobd_archive_test.go` plus Backlog und Journal. Kein
  Produktionscode, also kein gRPC-Bypass, kein Stub, kein `.proto`, kein
  `RequirePermission`, keine Tabelle, keine Route, kein Wire-Shape. `4c70d358` ist der
  reine Journal-SHA-Nachtrag.)
- neue-units: fix-websocket-chat-rpcs-missing-tenant-context ·
  fix-websocket-hard-disconnect-cleanup-delay · fix-websocket-rate-limit-reset-on-reconnect
- offen: **Drei verifizierte Produktionsbefunde, alle als Unit im Backlog, keiner in dieser
  Iteration gefixt (Coverage-Unit aendert kein Verhalten).**
  1. **Chat ueber WebSocket ist funktionslos.** `/api/v1/ws` ist in
     `cmd/gateway/main.go:453` ohne `authMiddleware` registriert; der Handler validiert das
     JWT selbst, schreibt die Tenant-ID aber nie in den Context. Damit haengt
     `TenantOutboundUnaryInterceptor` kein `x-tenant-id` an, und `ChatGRPCServer.GetChannel`
     (`chat_grpc.go:103`) lehnt mit `Unauthenticated` ab — `channel.subscribe` kann also nie
     gelingen, und ohne Subscribe gibt es keine Live-Zustellung. `channel.mark_read` genauso.
     `message.send` scheitert aus einem zweiten Grund: `SendMessage` liest die Tenant-ID aus
     `req.TenantId` (Client-Payload!) statt aus dem Context, und der WS-Handler setzt das
     Feld nie -> `InvalidArgument`. Kein RLS-Bypass, es faellt fail-closed aus. Der Test
     `TestHandleSubscribeChannel_SendsContextWithoutTenantToChatService` reproduziert es mit
     einem Fake, der exakt den Guard des echten Servers nachbildet. Das Fix-Muster liegt drei
     Bildschirme tiefer im selben File: `tenantAllowsPresenceTarget` (`websocket.go:1203`)
     baut sich seinen `scopedCtx` genau so, wie Chat es braeuchte — fuer Presence
     nachgezogen, fuer Chat vergessen. **Bitte gegen die Produktion gegenpruefen:** wenn dort
     Live-Chat sichtbar funktioniert, habe ich einen Pfad uebersehen.
  2. **Ein harter Verbindungsabbruch raeumt bis zu 5 Minuten lang nichts auf.**
     `handleConnection` wartet nach dem Read-Fehler auf `<-done`, und die
     Revalidierungs-Goroutine kehrt nur bei `ctx.Done()` oder ungueltigem Token zurueck —
     `r.Context()` einer gehijackten Verbindung wird beim Socket-Tod nicht gecancelt. Mit
     `maxConnsPerUser = 5` sperren fuenf Abbrueche in Folge den Nutzer mit 429 aus. Ich
     hatte zuerst das Gegenteil gemessen; der erste Testlauf war irrefuehrend, weil `exp` im
     JWT nur Sekunden-Aufloesung hat und ein 400-ms-Token schon beim Handshake abgelaufen
     war. Mit 2-s-Token ist der Befund reproduzierbar und im Test als Phase 1 festgehalten.
  3. **Der Nachrichten-Rate-Limiter ist per Reconnect zuruecksetzbar** —
     `unregisterConnection` loescht den Bucket mit der letzten Verbindung, der naechste
     Verbindungsaufbau startet mit vollem Burst.
  Nebenbefund ohne eigene Unit: der Docstring von `TenantInboundUnaryInterceptor`
  (`middleware/grpc_tenant.go:73`) behauptet `codes.Unauthenticated` bei fehlendem Tenant,
  der Code tut das nicht — die Korrektur haengt an Unit 1 mit dran.
  `-race` ist auf dieser Maschine nicht gelaufen (Vorgabe der Unit); die Goroutine-Zaehlung
  kommt ohne aus, aber der Race-Beweis bleibt CI vorbehalten.

## Iteration 20 - cov-auth-package-remaining-paths - done - 2026-08-27 04:11
- commit: 6ff4bee2
- gebaut: `internal/auth/login_paths_test.go` (~700 Z., neue Datei, kein Produktionscode
  geaendert). Deckt die vier in der Unit geforderten Flaechen ab:
  1. **Anmelde-Fehlerpfade** als eine Tabelle: unbekannte Adresse, falsches Passwort und
     kaputter Repository-Lookup liefern denselben Sentinel UND dieselbe Meldung; das
     deaktivierte Konto weicht ab (Befund 1, siehe unten) und ist als solcher festgehalten,
     nicht als Zusage. Dazu die Gross-/Kleinschreibungs-Normalisierung.
  2. **Zweiter Faktor**: `Login` gibt bei aktivem 2FA nur einen Pending-Token heraus und
     weder Access- noch Refresh-Token noch das User-Objekt. `CompleteTwoFactorLogin` mit
     TOTP, mit Recovery-Code, und sechs Ablehnungen (Muell, abgelaufener Pending-Token,
     falscher Token-Typ, **Access-Token als Pending-Token**, falscher TOTP-Code, unbekannter
     Nutzer). Recovery-Codes: Einmalverwendung, Replay, nie ausgegebener Code, und der
     erschoepfte Satz mit eigenem Sentinel. Konto-Deaktivierung zwischen den beiden Schritten.
  3. **2FA-Erzwingung**: innerhalb der Karenz laeuft der Login durch, nach Ablauf blockt er
     mit `Err2FAEnforcementRequired`, ein eingeschriebenes Konto erfuellt die Auflage.
     Policy-Update/-Lesen inkl. Tenant-Trennung.
  4. **Token-Lebenszyklus** als drei Pins: Access-Token ueberlebt Passwortaenderung,
     Passwort-Reset und Rollenaenderung (Claims werden gepraegt, nicht nachgeschlagen);
     der Refresh-Token ueberlebt keinen davon. Dazu Refresh-Reuse-Erkennung (ein
     wiederverwendeter Token toetet auch das zweite Geraet) und der abgelaufene Refresh-Token,
     der ausdruecklich KEIN Diebstahlsignal ist.
  Nebenher gedeckt, weil bislang ohne jede Testdatei: `Setup2FA`, `Verify2FA`, `Disable2FA`,
  `RegenerateRecoveryCodes`, `AdminReset2FA`, `CreatePendingToken`.
- gate: build ok (`./internal/auth/... ./internal/middleware/... ./internal/gateway/...
  ./cmd/auth/... ./cmd/gateway/...`) | vet ok | lint ok (`golangci-lint run
  ./internal/auth/...`, 0 issues; drei echte `QF1008`-Befunde auf redundante
  Embedded-Selektoren wurden behoben) | test ok (`internal/auth` 21,7 s gruen,
  `internal/middleware` gruen, `internal/gateway` gruen; `DATABASE_URL` als `kmuhub_app`
  gesetzt, `go test -v | grep -c -- "--- SKIP"` = **0** bei **300** `--- PASS`) |
  migration n.a. (keine) | rls-smoke n.a. (keine Tabelle, keine Policy angefasst)
- coverage: internal/auth **67,9 % -> 77,3 %** (selbst gemessen, `go tool cover -func`
  vor und nach dem Anlegen der Datei; der Vorher-Wert deckt sich exakt mit dem
  `coverage_start:`-Wert der Unit aus CI 32949396303). `totp.go` hatte vorher **keine
  einzige eigene Testdatei** -- dort liegt der groesste Teil des Zugewinns.
- mutations-probe: In `validateRecoveryCode` (`totp.go:374`) den Aufruf
  `s.repo.UseRecoveryCode(ctx, c.ID)` entfernt, den Code also nicht mehr verbraucht ->
  `TestCompleteTwoFactorLogin_RecoveryCodeIsSingleUse` sofort rot an zwei Stellen (Replay
  desselben Codes liefert `nil` statt `ErrInvalidRecoveryCode`; der erschoepfte Satz liefert
  `nil` statt `ErrAllRecoveryCodesUsed`). Backup zurueckgespielt, `git status` zeigt nur noch
  die neue Testdatei, Paket erneut gruen.
- verify vorgaenger: sauber (`dc73da76` geprueft -- eine neue Testdatei
  `websocket_connection_test.go`, ein Feld `revalidateInterval` als Test-Haken in
  `websocket.go` mit `lean:`-Marker und Upgrade-Trigger, eine Zeile in `newTestHub`. Kein
  gRPC-Bypass, kein Stub, kein `.proto`, kein `RequirePermission`, keine Tabelle, keine
  Route, kein Wire-Shape. `530a45a0` ist der reine Journal-SHA-Nachtrag.)
- neue-units: fix-login-inactive-account-is-an-enumeration-oracle ·
  harden-password-reset-invalidate-previous-tokens
- offen: **Zwei verifizierte Befunde, beide als Unit im Backlog, keiner in dieser Iteration
  gefixt (Coverage-Unit aendert kein Verhalten).**
  1. **Ein deaktiviertes Konto ist von aussen erkennbar.** `Login`
     (`service.go:142`) liefert `ErrUserInactive`, alle anderen Fehlpfade
     `ErrInvalidCredentials`; `internal/server/grpc.go:1321` macht daraus **403 gegen 401**.
     Die Anti-Enumerations-Zusage, die der Kommentar in `Login` selbst formuliert, haelt
     also nur fuer drei von vier Pfaden. Die Aktiv-Pruefung laeuft ausserdem VOR dem
     bcrypt-Vergleich -- das Orakel braucht kein richtiges Passwort. Kein Datenleck im
     engeren Sinn, aber der Gegner bekommt eine geprueft existierende Adressliste.
  2. **Ein zweiter Reset-Antrag entwertet den ersten Token nicht.** Beide Links bleiben bis
     zu einer Stunde gueltig; der Test benutzt nachweislich den AELTEREN, nachdem der neuere
     ausgestellt wurde. Wer an die aeltere Mail kommt, uebernimmt das Konto, ohne dass der
     Kontoinhaber etwas merkt.
  **Feststellung ohne eigene Unit (Punkt 4 der Unit):** eine Brute-Force-Bremse existiert
  nicht. Kein Fehlversuchszaehler, keine Kontosperre, keine Migration dafuer -- zehn falsche
  Passwoerter hintereinander hinterlassen keinerlei Zustand
  (`TestLogin_NoBruteForceBrakeInTheService` haelt das fest). Die einzige Bremse vor
  `/api/v1/auth/login` ist der **globale** Per-IP-Limiter mit `RATE_LIMIT_RPS`,
  **Default 100/s** (`cmd/gateway/main.go:162`); der strikte `publicRateLimiter` mit
  Default 10/s haengt an Booking-, Wiki- und der Reset-HTML-Seite, **nicht** am Login. Ob
  das ein Befund oder eine bewusste Entscheidung ist, gehoert Luke -- deshalb keine Unit,
  sondern diese Zeile.
  **Zur Antwortzeit-Frage aus den `notes`:** bewusst NICHT zeitbasiert getestet (ein
  flakender Timing-Test waere wertlos). Statt dessen: unbekannte Adresse und kaputter
  Lookup ueberspringen den bcrypt-Vergleich, ein falsches Passwort nicht -- der
  Zeitunterschied existiert also strukturell. Er ist mit einem `bcryptCost` von 12 gut
  messbar. Das gehoert in dieselbe Betrachtung wie Befund 1 und ist dort in den `notes`
  als Warnung vermerkt, damit ein Fix nicht ein Status-Orakel gegen ein Zeit-Orakel tauscht.

## Iteration 21 - cov-security-gdpr-remaining-paths - done - 2026-08-27 04:23
- commit: f5fdd6d0
- gebaut:
  1. **Tenant-Filter-Audit `dsar_search.go`** (39 Tabellenabfragen, alle einzeln gelesen):
     38 von 39 filtern bereits explizit nach `tenant_id`. Die eine Ausnahme war
     `consentModule` (Zeile 1722) -- `WHERE contact_id = $1` ganz ohne `tenant_id`-Praedikat.
     Da `contact_id` die globale PK von `contacts` ist, war das in der Praxis nicht
     ausnutzbar (kein zweiter Tenant kann dieselbe `contact_id` besitzen) und RLS greift
     ohnehin, aber es widersprach der im Code selbst dokumentierten Konvention
     ("matching every other module here", Kommentar auf `customFieldsModule"). Fix: Parameter
     `tenantID` ergaenzt, `AND tenant_id = $2` ins WHERE, Aufrufstelle in `SearchByQuery`
     angepasst. Dokumentierender Kommentar an der Funktion ergaenzt.
  2. **Retention-Engine, Batch-Teilausfall** (Scope-Punkt 2): neuer Test
     `TestRetentionEngine_Run_PartialBatchFailure_UndercountsButHandlerStaysIdempotent` mit
     einem zustandsbehafteten Fake-Handler, der nach zwei von vier Datensaetzen scheitert.
     Ergebnis: die Handler-eigene Idempotenz haelt (zweiter Lauf plant nur die zwei
     wirklich offenen Datensaetze), aber `runPolicy` (retention.go:400-407) verwirft den von
     `Apply` zurueckgegebenen `affected`-Wert komplett im Fehlerpfad -- Log sagt "0 betroffen",
     obwohl der Handler real zwei Datensaetze verarbeitet hat. Nicht gefixt (Coverage-Unit
     aendert kein Verhalten), als Unit angelegt (siehe unten).
  3. **Scheduler-Test, zwei gehaltene Verbindungen** (Scope-Punkt 3): bereits vorhanden
     und korrekt -- `TestRunScheduledRetention_SkipsWhenLockHeldElsewhere` haelt den
     Advisory-Lock deterministisch auf einer eigenen Connection, waehrend der Testaufruf
     eine zweite benutzt. Nichts zu tun.
  4. **`GetNextAnonymizedLabel`-Zaehler-Race** (Scope-Punkt 4): neuer Test
     `TestGetNextAnonymizedLabel_ConcurrentCallersCollide` in `erasure_log_test.go`.
     `postgres_repository.go:197` ist ein blankes `SELECT COUNT(*) FROM gdpr_erasure_log`
     ohne Sperre und ohne Sequence. Der Test haelt zwei REPEATABLE-READ-Transaktionen (wie
     von den harten Regeln fuer Per-Verbindung-Ressourcen gefordert), zeigt dass beide vor
     jedem Insert denselben Count sehen, und dass zwei parallele Anonymisierungen deshalb
     GARANTIERT dasselbe Label erzeugen. Nicht gefixt, als Unit angelegt (siehe unten).
- gate: build ok (`./internal/security/gdpr/... ./internal/gateway/... ./cmd/gateway/...`) |
  vet ok | lint ok (`golangci-lint run ./internal/security/gdpr/...`, 0 issues) | test ok
  (`internal/security/gdpr` 2,5 s gruen, 199 `--- PASS`, **0** `--- SKIP`, **0** `--- FAIL`,
  `DATABASE_URL` als `kmuhub_app` gesetzt) | migration n.a. (keine noetig fuer den
  `consentModule`-Fix) | rls-smoke n.a. (kein neues Schema, keine neue Policy; der
  Tenant-Filter ist app-seitiges Defense-in-Depth ueber einer bereits per RLS geschuetzten
  Tabelle) | `go test ./internal/gateway/ -run TestOpenAPIRouteDrift` gruen (836 Routen
  gegen 838 Pfade, keine Route angefasst)
- coverage: internal/security/gdpr **72,6 % -> 72,7 %** (selbst gemessen via `git stash`
  auf den drei geaenderten Dateien vor und nach dem Messen; der Vorher-Wert deckt sich
  exakt mit dem `coverage_start:`-Wert der Unit aus CI 32949396303). Kleiner Zugewinn,
  weil das Paket bereits sehr dicht getestet war -- der Wert dieser Iteration liegt in den
  beiden belegten Befunden, nicht in Prozentpunkten.
- mutations-probe (zwei, je Fund eine):
  1. `consentModule`: `tenant_id = $2` zu `tenant_id = $1` veraendert (vergleicht `contact_id`
     mit sich selbst statt mit `tenantID`) -> `TestSearchByQuery_ContactWithAllModules_Integration`
     sofort rot (`ERROR: could not determine data type of parameter $2`, weil `$2` dann
     unbenutzt blieb). Backup zurueckgespielt, `git diff` zeigt wieder nur den urspruenglichen
     Fix.
  2. Retention-Engine: `item.Affected = affected` versuchsweise auch in den Fehlerpfad
     (Zeile 402-406) ergaenzt, um zu pruefen, dass mein neuer Test wirklich die AKTUELLE
     Buggy-Semantik pinnt und nicht zufaellig gruen waere -> beide Affected-Assertions
     sofort rot (erwartet 0, bekommen 2). Backup zurueckgespielt, `git diff` zeigt keinen
     Unterschied zu HEAD mehr.
- verify vorgaenger: sauber (`6ff4bee2` geprueft -- ausschliesslich eine neue Testdatei
  `login_paths_test.go`, kein Produktionscode geaendert, keine Stubs/Skips/TODOs im Diff.
  `3411dded` ist der reine Journal-SHA-Nachtrag der Vorgaenger-Iteration.)
- neue-units: fix-retention-engine-drops-partial-affected-count-on-apply-failure ·
  fix-gdpr-anonymized-label-counter-race
- offen: Beide neuen Units sind reine Verhaltens-Fixes ohne Deploy-Risiko (RETENTION_MODE
  bleibt dry_run, das Label-Problem betrifft nur die manuelle Erasure). Fuer den
  Label-Zaehler ist in den `notes` der Units bereits eine Sequence-basierte Loesung
  skizziert, die eine neue forward-only Migration braucht -- Nummer zur Laufzeit ermitteln,
  nicht aus dieser Zeile uebernehmen. `internal/security/gdpr` bleibt trotz 72,7 % ein
  grosses Paket mit vielen Handlern; die verbleibenden 640 ungedeckten Statements liegen
  ueberwiegend in Handler-Randfaellen, die diese Iteration nicht angefasst hat.

## Bilanz Lauf (2026-08-27 04:37)
- iterationen: 40 im Bereich 1-21, davon 40 done, 0 blocked, 0 ohne auswertbare Kopfzeile
- units nach praefix: cov 20 · fix 10 · feat 6 · harden 3 · scan 1
- commits nach typ: chore 40 · test 13 · fix 7 · docs 1 (61 seit a0ed89e4)
- coverage-delta: internal/email/sync 34,6 -> 64,6 · internal/fuhrpark 54,5 -> 81,3 · internal/settings 60,3 -> 82,8 · (14 weitere)
- offen mit entscheidungsbedarf: 11 von 26 nicht-leeren offen:-Zeilen (Treffer auf "Luke"/"Entscheidung")
- minuten je iteration: 5,3 (214 gesamt)
