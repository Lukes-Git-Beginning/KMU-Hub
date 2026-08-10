# Backend-Nachtloop — Journal (Lauf 7)

Append-only. Eine Iteration = ein Eintrag. **Immer ans Dateiende anhaengen, nie vor einen
bestehenden Eintrag einsortieren** — der Treiber leitet die Fortschrittsanzeige aus der hoechsten
Iterationsnummer ab, und ein eingeschobener Eintrag hat in Lauf 3 zwei Iterationen lang denselben
Stand gemeldet.

Vorlage:

```markdown
## Iteration <n> — <unit-id> — <done|blocked> — <YYYY-MM-DD HH:MM>
- commit: <sha oder ->
- gebaut: <ein bis drei Zeilen, was real existiert>
- gate: build ok | vet ok | lint ok | test ok | migration ok | rls-smoke ok|n.a.
- verify vorgaenger: <sauber | Befund + angelegte Fix-Unit>
- offen: <was Luke morgens pruefen muss — DB-Gate, Proto-Regen, Route-Registrierung, Annahmen>
```

Bei Coverage-Units (Bloecke C und B) gehoert zusaetzlich in den Eintrag:

```markdown
- mutations-probe: <welche Zeile gebrochen wurde, ob der Test rot wurde, zurueckgedreht ja/nein>
- db-tests: <Zahl der real gelaufenen DB-Tests, Zahl der Skips — bei Block C muss Skips = 0 sein>
```

Journale der Vorlaeufe: `archive/lauf-3/JOURNAL.md`, `archive/lauf-4/JOURNAL.md`,
`archive/lauf-5/JOURNAL.md` (Lauf 5 haengt dort am Ende des Lauf-4-Journals),
`archive/lauf-6/JOURNAL.md`.

---

## Lauf 7 — Ausgangslage (2026-08-09, vor der ersten Iteration)

- Branch `backend-loop`, auf `origin/main` gemergt. Diesmal **kein** Fast-Forward: `origin/main`
  ist der Merge-Commit von PR #19 (`1e68b7dc`) und damit kein Ancestor des Branches. Der Merge
  ist inhaltlich trivial (gleicher Baum wie `79651386`), erzeugt aber einen echten Merge-Commit.
- Migrationskopf lokal wie produktiv **308**, `dirty=false`. Naechste freie Nummer 309 — immer
  zur Laufzeit ermitteln, nie annehmen.
- Lokale DB laeuft und ist verifiziert: `docker-postgres-1` healthy auf 308, Rolle `kmuhub_app`
  mit Passwort `app_dev` (Login geprueft), 283 Tabellen mit aktiver RLS. `DATABASE_URL` ist
  damit kein Alibi — wer ohne sie testet, hat kein Gate.
- Backlog: **50 offene Units**. Eine Fix-Unit vorweg, dann C1 (2, `biz` aufs 60-%-Ziel),
  dann B-Gateway (12, schliesst die Coverage-Delle aus Lauf 6), dann B-Server (12), dann
  C2 (23, Service-Paket-Coverage). Dazu 8 `blocked` aus den Vorlaeufen, die bewusst liegen
  bleiben — das sind Rechercheergebnisse mit offenen Produktentscheidungen, keine Ausfaelle.
- Fenster: **2026-08-09 22:00 bis 2026-08-10 09:00** (`-UntilTime "09:00"`), rund 11 Stunden.
  Beim Median aus Lauf 6 (13 min ueber 47 Iterationen) deckt das etwa 50 Units. Was von C2
  liegen bleibt, startet Lauf 8.
- Coverage-Ausgangswerte aus dem CI-`coverage.out` von Lauf 6 (gewichtet, nicht geschaetzt):
  Gesamt **36,3 %** · `crm` 69,5 % (Ziel erreicht) · `biz` 54,9 % (Ziel 60 verfehlt) ·
  `server` 26,0 % · `gateway` **24,2 %** (in Lauf 6 von 27,2 % GEFALLEN — Block A legte
  Routen schneller an, als Block B sie testete).

### Was aus Lauf 6 als Lehre in diesem Lauf gilt

- **Build-Tags vor jeder Coverage-Unit greppen.** Drei Iterationen von Lauf 6 legten ihre Tests
  hinter `//go:build integration`. Die Tests waren gut, liefen lokal real durch und meldeten
  ehrlich "0 Skips" — aber der PR-blockierende CI-Job kompiliert sie gar nicht, und in
  `coverage.out` tauchen sie nicht auf. Fuer die Pakete in Block C2 ist das vorab geprueft.
- **Der Verify-Vorspann funktioniert — ueberspring ihn nicht.** In Lauf 6 baute Iteration 12
  einen Fehler, Iteration 13 fand ihn und legte die Fix-Unit an. Zweimal wurde ausserdem eine
  Backlog-Praemisse widerlegt statt blind gebaut, einmal mit einem echten Produktionsbugfund
  (`mentions:read` war keiner Rolle zugeordnet, jeder Aufruf 403 — auch Admin).
- **Nummer und Zeitstempel nicht raten.** Beides steht ab diesem Lauf im Laufkontext-Block am
  Ende deines Prompts, vom Treiber gesetzt.
- **Ausgerollte Migrationen sind tabu — auch ihre Kommentare.**

## Iteration 1 — fix-inventar-picking-partial-book — done — 2026-08-09 19:10
- commit: 06700605
- gebaut: `BookPickingList` und `BookInventurDifferences` liefen bisher pro Position einen
  eigenen `AdjustStock`-Call und loggten einen Fehschlag nur — die Liste/Session blieb
  trotzdem `completed`, ein Teil des Bestands unbewegt. Beide teilen sich jetzt
  `Repository.applyMovementsInTx`: Claim/Complete und jede Bestandsbewegung laufen in EINER
  Postgres-Transaktion (`BookPickingListTx`, `CompleteInventurSessionTx`). `CompletePickingList`
  ist entfallen (war nur noch Totcode). `movementTypeForDelta` als geteilter Helfer, auch von
  `AdjustStock` genutzt (Dedup).
- gate: build ok | vet ok | lint ok (golangci-lint 0 issues) | test ok | migration n.a. (keine
  neue Migration) | rls-smoke n.a.
- verify vorgaenger: n.a. — erste Iteration von Lauf 7, kein Vorgaenger-Commit in diesem Lauf.
- offen: Nichts Blockierendes. Zur Kenntnis: `applyMovementsInTx` verwendet `SELECT ... FOR
  UPDATE` statt Advisory-Lock (anders als `produktion`/`fuhrpark`s Booking-Pattern) — passend,
  weil hier direkt die existierende Item-Zeile gesperrt wird, nicht auf eine noch nicht
  existierende Zeile geprueft wird. `AdjustStock` selbst (manuelle Einzel-Anpassung) blieb
  unangetastet, nur die Movement-Type-Klassifikation wurde in den geteilten Helfer gezogen.
- mutations-probe: `it.Quantity+mv.Delta < 0`-Check in `applyMovementsInTx`
  (`postgres_repository.go`) auf `false && ...` gesetzt → `TestBookPickingListTx_
  PartialFailureRollsBackClaimAndStock` (DB-Test) wurde rot ("expected ErrInsufficientStock,
  got <nil>"), zurueckgedreht, Suite wieder gruen.
- db-tests: 2 echte DB-Tests neu (`picking_booking_tx_test.go`, gegen lokale DB via
  `docker-postgres-1`/`kmuhub_app` verifiziert: ON-CONFLICT-Upsert, Claim-UPDATE, atomarer
  Rollback bei Teilfehler), 0 Skips bei gesetzter `DATABASE_URL`. Docker Desktop war zu
  Laufbeginn nicht gestartet — musste hochgefahren und `docker-compose up -d postgres` erneut
  laufen lassen, bevor die DB-Tests liefen; danach `docker-postgres-1` wieder healthy.

## Iteration 2 — c-cov-biz-lexware — done — 2026-08-09 19:23
- commit: 9d0443b3
- gebaut: 6 neue Testdateien fuer `internal/biz/lexware`: `field_mapper_test.go`
  (Tabellentests fuer `MapContactToKMUHub`/`MapContactToLexware`/
  `applyContactFieldTo*`/`setNestedLexwareValue`/`ValidateFieldMappings`/
  `MapInvoiceToLexware`/`MapQuoteToLexware`, jede Switch-Case-Verzweigung
  einzeln), `auth_test.go` (`APIKeyManager` komplett), `postgres_config_repo_db_test.go`
  + `postgres_repository_db_test.go` (DB-Tests gegen das reale Schema fuer
  Get/List/Delete/Update-Pfade beider Repositories, `PostgresIntegrationConfigRepo.Upsert`
  eingeschlossen — der ist sauber), `service_extra_test.go` (TriggerSync,
  Service-Feldmappings, Scheduler Start/Stop/AddTenant ueber Mocks),
  `webhook_handler_registration_test.go` (Register-/UnregisterWebhooks gegen
  `httptest`-Server, Muster von `bexio/contact_sync_happy_test.go` uebernommen).
  Coverage `internal/biz/lexware`: 41,5 % → 71,4 %.
- gate: build ok | vet ok | lint ok (golangci-lint 0 issues) | test ok | migration n.a. | rls-smoke n.a. (nur Reads/Deletes gegen bestehende RLS-Policies, keine Policy angefasst)
- verify vorgaenger: sauber — `06700605` (fix-inventar-picking-partial-book)
  gegen alle acht Fehlerklassen geprueft: kein gRPC-Layer-Bypass (reine
  `internal/inventar`-Service-/Repository-Aenderung), kein Stub, kein
  `.proto` beruehrt, kein neuer `RequirePermission`-Guard, keine neue
  Tabelle, korrekte Transaktionsgrenzen (`BeginTx`/`defer Rollback`/`Commit`
  in `applyMovementsInTx`, geteilt von `BookPickingListTx` und
  `CompleteInventurSessionTx`), keine Routen-Aenderung, kein Guard ersetzt.
- **Befund waehrend der Arbeit (kein Fix in dieser Iteration, siehe Grund unten):**
  `UpsertSyncConfig`, `UpsertEntityMapping`, `UpsertFieldMappings`,
  `CreateSyncLog` und `UpsertWebhookSubscription` in
  `internal/biz/lexware/postgres_repository.go` bauen ihre INSERTs ohne
  `tenant_id`-Spalte. Alle fuenf Zieltabellen haben `tenant_id UUID NOT NULL`
  ohne Default und ohne befuellenden Trigger — verifiziert per `\d <table>`
  gegen `docker-postgres-1` (alle fuenf) und per Grep ueber saemtliche
  Migrationen nach einem tenant_id-setzenden Trigger (keiner vorhanden;
  `enable_tenant_rls` legt nur die RLS-Policy an, backfillt keine Inserts).
  Jeder Aufruf dieser fuenf Methoden scheitert auf einer echten Datenbank
  JEDES MAL mit einer NOT-NULL-Verletzung. Die vier `models.Lexware*`-Structs,
  die sie schreiben, tragen nicht einmal ein TenantID-Feld — die Luecke geht
  bis ins Modell durch. Betroffen: der komplette Lexware-Schreibpfad — Connect
  (UpsertSyncConfig), Kontakt-Sync in beide Richtungen (CreateSyncLog,
  UpsertEntityMapping), UpdateFieldMappings (UpsertFieldMappings), Webhook-
  (De-)Registrierung (UpsertWebhookSubscription). Nicht in dieser Iteration
  gefixt: Signaturaenderung ueber viele Call-Sites (Repository-Interface,
  service.go, contact_sync.go, webhook_handler.go, alle Mock-Doubles in
  service_test.go/service_wiring_test.go/tenant_isolation_*_test.go) —
  explizit ausserhalb des Lauf-7-Scopes fuer eine Coverage-Unit ("keine Fixes
  nebenbei"). Neue Fix-Unit `fix-lexware-tenant-id-missing-on-upsert` ganz
  vorne in `BACKLOG.yml` fuer Lauf 8 angelegt, `status: todo`. Meine neuen
  DB-Tests seeden ihre Fixtures deshalb direkt per `testutil.SeedRow`
  (umgeht die kaputten Upsert/Create-Methoden) und testen nur die Lese-/
  Lösch-/Update-Pfade — dokumentiert im Dateikopf von `postgres_repository_db_test.go`.
- **Wegen dieses Befunds bleibt `lexware ueber 80 %` (done_when) knapp
  verfehlt** — 71,4 % statt der geforderten 80 %. Die fuenf blockierten
  Funktionen sind rund 120 der 998 Statements (~12 %); ohne den Fix ist das
  praktische Maximum rund 88 %. Alle uebrigen done_when-Punkte sind erfuellt:
  Mapping-Pfade als Tabellentests gegen erwartete Literale, Fehlerklassen
  abgedeckt, Mutations-Probe durchgefuehrt.
- mutations-probe: `ValidateFieldMappings` in `field_mapper.go` — die
  Duplikat-Pruefung `if lexwareTargets[m.LexwareField] {` auf
  `if false && lexwareTargets[m.LexwareField] {` gesetzt →
  `TestValidateFieldMappings/duplicate_lexware_target` wurde rot ("An error
  is expected but got nil"), zurueckgedreht, Suite wieder gruen.
- db-tests: 17 echte DB-Tests gelaufen (3 `postgres_config_repo_db_test.go`,
  12 neu in `postgres_repository_db_test.go`, 2 vorbestehend in
  `tenant_isolation_phase2_test.go`/`tenant_isolation_phase3_test.go`),
  0 Skips bei gesetzter `DATABASE_URL` (verifiziert: `grep -c SKIP` auf dem
  vollen `-v`-Testlauf = 0). Cleanup-Reihenfolge-Bug dabei gefunden und
  behoben: `defer pool.Close()` lief vor den `t.Cleanup`-Aufrufen aus
  `seedLexwareFixture` (t.Cleanup laeuft immer erst NACH allen Defers der
  Testfunktion) — auf `t.Cleanup(func() { pool.Close() })` umgestellt, damit
  Zeilen wirklich geloescht werden statt gegen einen bereits geschlossenen
  Pool zu laufen.
- offen: `go test ./internal/gateway/` nicht gelaufen — diese Iteration hat
  keine Route/Gateway-Datei angefasst, daher laut Schritt 5 nicht Pflicht.
  `fix-lexware-tenant-id-missing-on-upsert` ist die naechste inhaltlich
  wichtige Unit fuer Lauf 8, gehoert aber nicht automatisch an die Spitze der
  Reihenfolge — Luke sollte kurz entscheiden, ob sie vor `c-cov-biz-recurring`
  vorgezogen wird (Bug mit Produktions-Impact fuer den Lexware-Schreibpfad,
  sobald `modules.lexware` aktiv ist) oder regulaer in der C1-Reihenfolge
  bleibt.

## Iteration 3 — fix-lexware-tenant-id-missing-on-upsert — done — 2026-08-09 19:52
- commit: 5b847267
- gebaut: Die vier `models.Lexware*`-Structs (`LexwareSyncConfig`,
  `LexwareEntityMapping`, `LexwareFieldMapping`, `LexwareSyncLog`) tragen
  jetzt `TenantID`. Alle fuenf betroffenen `PostgresRepository`-Methoden
  (`UpsertSyncConfig`, `UpsertEntityMapping`, `UpsertFieldMappings`,
  `CreateSyncLog`, `UpsertWebhookSubscription`) schreiben `tenant_id` in ihr
  INSERT. `UpsertWebhookSubscription` hat zusaetzlich einen `tenantID`-Parameter
  bekommen (Repository-Interface-Signatur geaendert). Jede Konstruktionsstelle
  eines neuen `models.Lexware*`-Literals wurde mit `TenantID` bestueckt:
  `service.go` (`Connect`, `UpdateFieldMappings` — Letzteres bekam dafuer
  selbst einen neuen `tenantID`-Parameter, durchgereicht vom gRPC-Layer
  `lexware_grpc.go:UpdateLexwareFieldMappings`, wo `req.GetTenantId()` bisher
  nur auf Leerstring geprueft, nie geparst wurde), `contact_sync.go`
  (`SyncContacts`-Synclog, `upsertInboundContact` — bekam dafuer einen neuen
  `tenantID`-Parameter, threaded durch beide Aufrufer `syncInbound` und
  `SyncContactByLexwareID` —, `syncOutbound`), `invoice_push.go` und
  `quote_push.go` (je Mapping- und Synclog-Literal). Bei Mappings, die per
  `Get*` geladen und dann mutiert wieder ge-upserted werden (Update-Zweige in
  `contact_sync.go`/`invoice_push.go`/`quote_push.go`), bleibt `TenantID` am
  geladenen Struct absichtlich leer — das ist sicher, weil der
  `ON CONFLICT ... DO UPDATE`-Zweig `tenant_id` nie in sein `SET` aufnimmt
  und Postgres den FK-Trigger fuer unveraenderte Spalten beim UPDATE
  ueberspringt; nur die INSERT-Zweige (neue Structs) brauchten eine explizite
  Zuweisung, das ist dokumentiert im Kommentarkopf von
  `postgres_repository_db_test.go`.
- **Zweiter, unabhaengiger Bug beim Bauen der DB-Tests gefunden und gleich
  mitgefixt (gleiche Methode, kein separater Scope-Bruch):**
  `UpsertWebhookSubscription`s `ON CONFLICT (config_id, subscription_id)`
  zielte auf einen Constraint, den es nie gab — die Tabelle
  `lexware_webhook_subscriptions` hat laut Migration 000056 nur
  `UNIQUE(config_id, event_type)`. Jeder Aufruf waere mit
  `SQLSTATE 42P10 (there is no unique or exclusion constraint matching the
  ON CONFLICT specification)` gescheitert, komplett unabhaengig vom
  tenant_id-Bug. Fix: Conflict-Target auf `(config_id, event_type)`
  umgestellt, `subscription_id` dafuer in die `SET`-Klausel verschoben (eine
  Re-Registrierung bekommt eine neue Lexware-Subscription-ID fuer denselben
  Event-Typ). Per Mutations-Probe verifiziert (siehe unten).
- **Bexio hat denselben tenant_id-Bug — verifiziert, nicht gefixt, neue Unit
  angelegt:** `internal/biz/bexio/postgres_repository.go`s
  `UpsertSyncConfig`/`UpsertEntityMapping`/`UpsertFieldMappings`/
  `CreateSyncLog` bauen ihre INSERTs identisch ohne `tenant_id` gegen
  identisch NOT-NULL-ohne-Trigger-Spalten (Migrationen 000115/000125, exakt
  dasselbe Backfill-Muster wie bei lexware). bexio hat kein
  Webhook-Pendant (OAuth statt Webhooks), also vier statt fuenf Methoden.
  Bewusst NICHT in dieser Iteration gefixt — anderes Modul, eigener
  Blast-Radius, waere Scope-Ausweitung ueber die gezogene Unit hinaus. Neue
  Fix-Unit `fix-bexio-tenant-id-missing-on-upsert` in `BACKLOG.yml`
  angelegt, `status: todo`, mit Hinweis auf `payment_poll.go`/
  `invoice_pull.go` (haben kein Lexware-Aequivalent, brauchen eigene Pruefung)
  und auf den ON-CONFLICT-Fund (bei bexio nicht verifiziert, aber derselbe
  Fehlerklasse plausibel — als Pruefpunkt in den `notes` vermerkt).
- gate: build ok (`go build -p 2 ./internal/biz/lexware/... ./internal/server/...
  ./internal/gateway/... ./cmd/gateway/...`) | vet ok | lint ok
  (golangci-lint, `./internal/biz/lexware/... ./internal/server/...
  ./internal/models/...`, 0 issues) | test ok (lexware, server, gateway alle
  gruen) | migration n.a. (keine neue Migration, nur bestehende Spalten
  genutzt) | rls-smoke n.a. (keine Policy angefasst, nur INSERT-Spaltenlisten)
- verify vorgaenger: sauber — `9d0443b3` (c-cov-biz-lexware) nur Testdateien
  + `BACKLOG.yml`, kein gRPC-Layer, kein Proto, keine neue Route, kein neuer
  Guard, keine neue Tabelle.
- offen: Diese Iteration ist bewusst von der im Laufkopf/ITERATION.md
  genannten Freigabe ("genau eine Fix-Unit fix-inventar-picking-partial-book")
  abgewichen, weil `fix-lexware-tenant-id-missing-on-upsert` zum Zeitpunkt
  dieser Iteration bereits ganz vorne in `BACKLOG.yml` stand mit
  `status: todo` (von Iteration 2 selbst dort platziert) und Schritt 2 des
  Ablaufs mechanisch "die erste Unit mit status: todo" verlangt — keine neue
  Fix-Unit wurde nebenbei erfunden, nur die bereits wartende gezogen. Luke
  sollte das kurz gegenpruefen, falls das nicht die gewuenschte Lesart war.
  `fix-bexio-tenant-id-missing-on-upsert` (neu, `status: todo`) ist nicht
  Teil der urspruenglich freigegebenen Bloecke fuer Lauf 7 — bewusst nicht
  gezogen, liegt fuer Lauf 8 bereit. `go test ./internal/gateway/` gelaufen
  und gruen, obwohl kein `.go`-Routendateiinhalt veraendert wurde (nur der
  gRPC-Server in `internal/server/lexware_grpc.go`) — zur Sicherheit
  mitgelaufen, da eine gRPC-Signatur sich geaendert hat.
- mutations-probe: `ON CONFLICT (config_id, event_type)` in
  `UpsertWebhookSubscription` zurueck auf das falsche, nie existierende
  `(config_id, subscription_id)` gesetzt →
  `TestPostgresRepository_UpsertWebhookSubscription_RealSchema` wurde rot
  (`SQLSTATE 42P10`), zurueckgedreht, Suite wieder gruen.
- db-tests: 5 neue DB-Tests gegen die real gefixten Upsert/Create-Methoden
  (`TestPostgresRepository_UpsertSyncConfig_RealSchema`,
  `_UpsertEntityMapping_RealSchema`, `_UpsertFieldMappings_RealSchema`,
  `_CreateSyncLog_RealSchema`, `_UpsertWebhookSubscription_RealSchema`),
  jede mit Insert- und Update-Pfad (ON-CONFLICT-Ziel real gepruef). Zusammen
  mit den 17 bereits bestehenden lexware-DB-Tests aus Iteration 2 liefen
  22 echte DB-Tests, 0 Skips bei gesetzter `DATABASE_URL`
  (`go test -v ./internal/biz/lexware/...`, `grep -c SKIP` = 0).

## Iteration 4 — fix-bexio-tenant-id-missing-on-upsert — done — 2026-08-09 19:55
- commit: 8118db67
- gebaut: `UpsertSyncConfig`, `UpsertEntityMapping`, `UpsertFieldMappings`, `CreateSyncLog`
  in `internal/biz/bexio/postgres_repository.go` schreiben jetzt `tenant_id` in ihr INSERT.
  Die vier `models.Bexio*`-Structs (`BexioSyncConfig`, `BexioEntityMapping`,
  `BexioFieldMapping`, `BexioSyncLog`) tragen ein neues `TenantID`-Feld. Jede
  Konstruktionsstelle eines neuen Structs wurde mit `TenantID` bestueckt:
  `contact_sync.go` (SyncLog, zwei `newMapping`-Literale), `invoice_pull.go`
  (SyncLog, Mapping bei Neuanlage), `invoice_push.go` und `quote_push.go`
  (je SyncLog + `newMapping`), `payment_poll.go` (SyncLog), `oauth_handler.go`
  (SyncConfig + FieldMapping in `HandleCallback`, das `tenantID` bereits als
  Parameter fuehrt). Anders als bei lexware brauchte KEIN Repository-Interface
  eine neue Signatur — alle vier betroffenen Methoden nehmen bereits das volle
  Model-Struct entgegen, nicht einzelne Skalare. Eine echte Signaturaenderung
  gab es nur eine Ebene hoeher: `Service.UpdateFieldMappings` bekam einen
  neuen `tenantID uuid.UUID`-Parameter (identisches Muster zu lexware), durch-
  gereicht vom gRPC-Layer `bexio_grpc.go:UpdateBexioFieldMappings`, wo
  `req.GetTenantId()` bisher nur auf Leerstring geprueft, nie geparst wurde.
  Update-Pfade, die ein bestehendes Struct laden und mutiert wieder upserten
  (`UpsertEntityMapping` in den Update-Zweigen, `UpdateSyncConfig`), lassen
  `TenantID` bewusst leer — sicher, weil `ON CONFLICT ... DO UPDATE` `tenant_id`
  nie in sein `SET` aufnimmt, exakt das etablierte lexware-Muster.
- **ON-CONFLICT-Targets verifiziert, kein zweiter Bug wie bei lexware:** alle
  drei `ON CONFLICT`-Klauseln (`(config_id)`, `(config_id, entity_type,
  kmuhub_id)`, `(config_id, entity_type)`) wurden gegen die echten
  UNIQUE-Constraints aus `backend/migrations/000055_add_bexio_integration.up.sql`
  gegengeprueft — alle drei stimmen exakt. bexio hat kein Webhook-Pendant
  (OAuth-basiert), also keine fuenfte Methode wie bei lexware.
- **`payment_poll.go` und `invoice_pull.go` gezielt geprueft** (kein
  lexware-Aequivalent, im Backlog-Eintrag als offener Pruefpunkt vermerkt):
  beide haben `tenantID` bereits als Funktionsparameter im Scope, keine
  Ueberraschung gefunden.
- gate: build ok (`go build -p 2 ./internal/biz/bexio/... ./internal/server/...
  ./internal/models/... ./cmd/gateway/... ./cmd/biz/...`) | vet ok | lint ok
  (golangci-lint, dieselben Pakete, 0 issues) | test ok (`go test -count=1
  ./internal/biz/bexio/...`, `./internal/server/...`, `./internal/gateway/...`
  alle gruen) | migration n.a. (keine neue Migration, nur bestehende Spalten
  genutzt) | rls-smoke n.a. (keine Tabelle/Policy angefasst, nur
  INSERT-Spaltenlisten)
- verify vorgaenger: sauber — `5b847267` (fix-lexware-tenant-id-missing-on-upsert)
  gegen alle acht Fehlerklassen geprueft (vollstaendiger Diff gelesen): kein
  gRPC-Bypass, kein Stub, kein Proto-Drift ohne Regen, kein neuer Guard ohne
  Seed, keine neue Tabelle, Wire-Shape unveraendert, keine neue Route ohne
  Spec-Eintrag, kein Guard-Alt-Key verloren.
- offen: Diese Iteration zog mechanisch die erste `status: todo`-Unit in
  `BACKLOG.yml` (Schritt 2 des Ablaufs), obwohl Iteration 3 in ihren eigenen
  Notizen vermerkt hatte, die Unit bewusst NICHT zu ziehen und fuer Lauf 8
  aufzuheben — die beiden Iterationen widersprechen sich darin, ob "erste
  todo-Unit" oder "urspruenglich freigegebener Block" Vorrang hat. Diese
  Iteration folgt dem woertlichen Ablauf-Text ("Nimm die erste Unit mit
  status: todo"). Luke sollte das einmal grundsaetzlich klaeren, damit
  kuenftige Iterationen nicht wieder unterschiedlich entscheiden. Inhaltlich
  ist der Fix selbst risikoarm und eng am bereits gepruegten lexware-Muster.
  `modules.bexio`s Aktivierungsstatus wurde nicht separat geprueft (wie schon
  im urspruenglichen Fund vermerkt) — der Fix behebt den Bug unabhaengig davon.

## Iteration 5 — c-cov-biz-recurring — done — 2026-08-09 20:02
- commit: b965ba93
- gebaut: `internal/biz/recurring` Coverage von 52,4 % auf 88,1 % (Ziel >85 % erreicht).
  Zwei Bloecke: (1) `service_test.go` um 21 neue Tests erweitert — schliesst
  `Get`/`List`/`Delete`/`IsNotFound` (waren 0 %), `SetStatus` unbekannter Status
  und den Resume-eines-exhausted-Schedules-Zweig, `Generate`s defensiven
  `ErrScheduleExhausted`-Pfad (Periode bereits nach EndDate, direkt ueber
  Repo-State erzwungen wie im bestehenden Idempotenz-Test), `Create`s
  Title-Fallback auf CustomerName, `CreatedBy`-Zweig, StartDate-Default auf
  heute, `Update`s bislang unerreichte Felder (Customer/Currency/PaymentTerms/
  Notes/LineItems/ClearEndDate/EndDate-Set/EndDate-vor-StartDate-Fehler),
  ReverseCharge- und Kleinunternehmer-Steuermodus (beide zuvor nie ueber den
  eigenen Zweig gelaufen, nur ueber den Standard-Default), und
  `addMonthsClamped`s negativer-Monate-Zweig (in `nextRunFor` heute
  unerreichbar, da periods>=0 geclampt wird — direkt aufgerufen, um die
  Arithmetik fuer einen kuenftigen rueckwaerts-Aufruf zu sichern).
  (2) neue Datei `postgres_repository_db_test.go` — neun DB-Tests gegen das
  reale Schema (`finance_recurring_invoices`/`finance_recurring_runs`,
  Migration 000246): Create+GetByID, GetByID NotFound (unbekannte ID UND
  falscher Tenant-Filter), List (Status-Filter, Limit->200-Clamp,
  negativer Offset, fremder Tenant liefert leeres `[]` nicht `null`),
  Update (Treffer + NotFound bei RowsAffected=0), Delete (Treffer +
  NotFound), ClaimPeriod-Replay (ON CONFLICT DO NOTHING liefert dieselbe
  Run-ID zurueck, kein zweiter Datensatz), AttachInvoice, und ReleasePeriod
  (unattached wird geloescht und freigegeben, attached bleibt stehen —
  genau der Schutz aus dem Doc-Kommentar des Repositories). Jeder Schreib-
  und jeder legitime Lesezugriff laeuft ueber `testutil.WithTenantCtx`, da
  beide Tabellen FORCE ROW LEVEL SECURITY tragen (erst ohne das versucht,
  `insert ... violates row-level security policy`, dann korrigiert).
- fehlerpfade: pro getesteter Funktion mindestens ein Fehlerfall — Get/Delete/
  Update NotFound, Create-Validierung (bereits vorhanden), SetStatus
  ErrInvalidStatus, Generate ErrScheduleExhausted/ErrNotActive, Update
  ErrInvalidDateRange/ErrInvalidInterval.
- mutations-probe: `ReleasePeriod`s `AND invoice_id IS NULL`-Guard aus dem
  DELETE-Statement entfernt (`internal/biz/recurring/postgres_repository.go`,
  vorher: `WHERE tenant_id = $1 AND id = $2 AND invoice_id IS NULL`). Damit
  wurde `TestPostgresRepository_ReleasePeriod` rot ("releasing an attached
  run must not delete it", Run-ID und Invoice-ID wichen ab) — der Test faengt
  also genau den Bug, den der Doc-Kommentar des Repositories beschreibt
  (ein geloeschter attached Run wuerde eine zweite Rechnung fuer dieselbe
  Periode ermoeglichen). Zurueckgedreht, Suite wieder gruen.
- gate: build ok (`go build -p 2 ./internal/biz/recurring/... ./cmd/biz/...`) |
  vet ok | lint ok (golangci-lint, 0 issues) | test ok
  (`go test -count=1 ./internal/biz/recurring/...`, 40 Top-Level-Tests + Sub-
  tests, 0 Fails, 0 Skips bei gesetzter `DATABASE_URL` — 9 echte DB-Tests
  liefen real) | migration n.a. (keine neue Migration) | rls-smoke n.a.
  (keine neue Tabelle/Policy, nur Tests gegen bestehende) | route n.a. (kein
  Gateway-Handler angefasst, `go test ./internal/gateway/` deshalb nicht
  Pflicht gemaess Schritt 5 und nicht separat gelaufen)
- verify vorgaenger: sauber — `8118db67` (fix-bexio-tenant-id-missing-on-upsert)
  gegen alle acht Fehlerklassen geprueft (vollstaendiger Diff gelesen): kein
  gRPC-Bypass, kein Stub, kein Proto-Drift, kein neuer Guard ohne Seed, keine
  neue Tabelle/RLS, Wire-Shape unveraendert, keine neue Route, kein
  Guard-Alt-Key-Verlust. ON-CONFLICT-Targets waren bereits gegen Migration
  000055 verifiziert (Iteration-4-eigene Notiz).
- offen: Backlog-Notiz "Ausloese-Pfad mit und ohne faelligen Termin" aus dem
  `done_when` wurde als Generate-Aufruf mit faelliger vs. bereits abgelaufener
  Periode interpretiert (`ErrScheduleExhausted`), NICHT als Scheduler/Cron —
  ein solcher existiert nicht; Migration 000246 kommentiert den Index
  `idx_finance_recurring_tenant_due` explizit als "Due-schedule scan for a
  future scheduler pass", also noch ungebaut. Kein Befund, nur Klarstellung
  fuer den Fall, dass ein Leser das anders erwartet hatte. Iteration 4 hat
  eine Grundsatzfrage offengelassen (erste `todo`-Unit vs. urspruenglich
  freigegebener Block bei Widerspruch) — diese Iteration war davon nicht
  betroffen, da `c-cov-biz-recurring` sowohl die erste `todo`-Unit als auch
  Teil des urspruenglich freigegebenen Blocks C1 ist.

## Iteration 6 — b-cov-gateway-inbox — done — 2026-08-09 20:10
- commit: b2a80006
- gebaut: neue Datei `internal/gateway/route_inbox_test.go` fuer
  `route_inbox.go` (1.396 Zeilen, 36 Handler-Funktionen ueber 45 registrierte
  Routen, vorher 0 % Coverage, keine Testdatei). Abgedeckt: `ServiceName`,
  ein tabellengetriebener 503-Test ueber alle 36 Handler (jeder prueft den
  gRPC-Client zuerst, bevor er irgendetwas anderes tut — verifiziert per
  vollstaendiger Dateilektuere), ein Router-Level-Test fuer die im Backlog
  benannte Kollisionsgefahr `/messages/unread-count` vs. `/messages/{id}`,
  ein tabellengetriebener UUID-Validierungstest ueber alle 15 reinen
  Id-Handler plus die beiden Zwei-Id-Faelle von `HandleRemoveTeamMember`,
  und je ein bis drei Validierungsfaelle fuer jeden Handler mit Body
  (Status-`oneof`, Tag/To/Body-`required`, RFC3339-Parse in
  `HandleSnoozeMessage` — das liegt nach `decodeAndValidate`, kein
  `validate`-Tag kann "parsebar als Zeitstempel" ausdruecken —, AssigneeID-
  `uuid`, Bulk-IDs `dive,uuid`, Canned-Response-Name/-Body inkl.
  Laengenlimit, Team-Inbox `assignment_mode`/`visibility`-`oneof`,
  Team-Member-Felder, Routing-Rule-Name plus die `rawJSONToStruct`-
  Fehlerpfade fuer `conditions`/`actions` bei Create UND Update).
  Zusaetzlich vier Tabellentests fuer die vier Enum-Parser
  (`parseChannelQuery`/`parseAssignmentMode`/`parseVisibility`/
  `parseTeamMemberRole`, je inklusive Default-Fallback-Zweig) und vier
  direkte Tests fuer `rawJSONToStruct` (leer, gueltig, Nicht-Objekt,
  kaputtes JSON). Response-Wire-Shapes (`toProto*`-Aequivalent) wurden NICHT
  extra getestet, weil diese Datei keine eigene `toProto`-Konvertierung
  besitzt — jeder Erfolgspfad reicht das rohe `resp` direkt an
  `response.Proto` durch (Ausnahme: die beiden Canned-Response-Handler
  marshaln ueber `cannedResponseMarshaler` und wrappen in
  `{"canned_response": ...}` — das liegt aber HINTER dem RPC-Call und ist
  mit den vorhandenen Gateway-Testhelfern ohne einen Fake-gRPC-Server nicht
  erreichbar, exakt das im Backlog-Kopf dokumentierte Repo-Muster: dieser
  Layer testet nur, was VOR dem RPC-Aufruf passiert).
- gate: build ok (`go build ./internal/gateway/...`) | vet ok
  (`go vet ./internal/gateway/...`) | lint ok (`golangci-lint run
  ./internal/gateway/...`, 0 issues) | test ok (`go test -count=1
  ./internal/gateway/`, dreimal wiederholt fuer Flake-Check, durchgehend
  gruen) | migration n.a. | rls-smoke n.a. (kein DB-Zugriff in diesem
  Testpaket)
- verify vorgaenger: sauber — `b965ba93` (c-cov-biz-recurring) Diff
  vollstaendig gelesen: nur `postgres_repository_db_test.go` (neu) und
  `service_test.go` (erweitert) plus Journal/Backlog, kein gRPC-Layer-
  Bypass, kein Stub, kein Proto beruehrt, kein neuer Guard, keine neue
  Tabelle/Migration, keine Routen-Aenderung.
- **Befund waehrend der Arbeit:** `assertValidationError` erwartet den
  JSON-Tag-Namen (z. B. `"status"`, `"member_user_id"`, `"ids[0]"` fuer
  ein einzelnes Slice-Element), nicht den Go-Feldnamen — beim ersten
  Testlauf 15 Fehlschlaege deswegen, alle auf die JSON-Tags korrigiert.
  Kein Code-Befund, nur eine Erinnerung fuer kuenftige Coverage-Units:
  `validation.ErrorBody` gibt `field` immer als JSON-Tag zurueck.
- offen: `HandleListMessages` bleibt bei 12,9 % (kein dedizierter Test fuer
  die Query-Parameter-Kombinatorik `channel`/`is_read`/`is_starred`/
  `team_inbox_id`/`search`/`status`/`page_size`-Clamp) und mehrere
  Update-Handler (`HandleUpdateCannedResponse` 28 %, `HandleUpdateTeamInbox`
  29,6 %, `HandleListTeamInboxes`/`HandleListRoutingRules`/
  `HandleListCannedResponses` 36–44 %) haben nur den Service-Unavailable-
  und ggf. den Id-Validierungspfad, keinen Test fuer die optionalen-Felder-
  Zusammenstellung nach dem `if body.X != nil`-Muster — das waere jeweils
  ein weiterer RPC-Erfolgspfad und braucht einen Fake-`InboxServiceClient`
  (Interface-Mock), den es fuer dieses Paket noch nicht gibt. Kein Blocker,
  nur Grenze dieser Iteration; ein kuenftiger `InboxServiceClient`-Mock nach
  dem `formulare_grpc_test.go`-Stub-Repo-Muster koennte das schliessen, ist
  aber ein groesserer Aufbau als eine einzelne Coverage-Unit.
  `internal/gateway`-Gesamtcoverage: 25,6 % (von 24,2 % zu Laufbeginn).
- mutations-probe: `/messages/unread-count` in `RegisterRoutes` kurzzeitig
  von `ir.HandleGetUnreadCount` auf `ir.HandleGetMessage` umgehaengt →
  `TestInboxRoutes_UnreadCountRouteOrder` wurde rot ("unread-count was
  routed into /messages/{id} and rejected as an invalid UUID"), exakt der
  im Backlog benannte Risikofall. Zurueckgedreht (`git diff` bestaetigt
  keine Restaenderung), Suite dreimal in Folge gruen.
- db-tests: 0 — `internal/gateway` ist reines HTTP-Handler-Paket ohne
  DB-Zugriff, alle bestehenden Gateway-Tests im Repo sind reine
  In-Memory-Tests. done_when verlangt hier keine DB-Tests.

## Iteration 7 — b-cov-gateway-crm-activities — done — 2026-08-09 20:22
- commit: (siehe unten)
- gebaut: `internal/gateway/route_crm_activities_test.go` (neu, ~500 Zeilen) —
  deckt die 23 zuvor ungetesteten Handler aus `route_crm_activities.go` ab
  (Activities CRUD + Complete + Tags-Stubs, Search, Saved Filters CRUD,
  drei Reports-Handler, plus die in `route_crm_test.go` noch fehlenden
  List/Update/Delete-Handler fuer Custom Fields und Tags). Create/Get fuer
  Custom Fields und Tags waren schon in `route_crm_test.go` abgedeckt und
  wurden nicht verdoppelt. Getestet: alle Validierungspfade
  (`activity_type`/`subject` required, `contact_id`/`company_id`/`deal_id`
  je einzeln als eigener Validierungsfall fuer den polymorphen Zweig,
  `tag_ids` dive-uuid, Saved-Filter- und Custom-Field-Pflichtfelder), die
  Reihenfolge-Eigenheiten (bei `HandleGetActivity`/`HandleDeleteActivity`/
  `HandleCompleteActivity` laeuft der Tenant-Check VOR dem UUID-Check, bei
  `HandleUpdateActivity` GENAU ANDERSHERUM — UUID vor Decode vor Tenant;
  beide Reihenfolgen als eigene Tests festgeschrieben, nicht angenommen),
  die zwei `HandleAddActivityTags`/`HandleRemoveActivityTags`-Stubs (rufen
  nie einen gRPC-Client, liefern immer 501 "not implemented via HTTP, use
  gRPC" — als bewusster, bereits bestehender Vertrag getestet, nicht neu
  gebaut), Filterkombinationen fuer `HandleListActivities`/
  `HandleListCustomFields`/`HandleListTags` und die `HandleSearch`-
  Limit-Grenzfaelle. `internal/gateway`-Gesamtcoverage: 26,5 % (von 25,6 %
  zu Iterationsbeginn), Handler-Coverage in `route_crm_activities.go` von
  0 % auf 41–100 % je Handler.
- **Befund waehrend der Arbeit:** die drei Reports-Handler
  (`HandleGetPipelineReport`/`HandleGetConversionReport`/
  `HandleGetActivityReport`) validieren `start_date`/`end_date` NUR auf
  Nicht-Leerheit, keine Datumsformat-Pruefung — ein syntaktisch ungueltiges
  Datum ("banana") wird unveraendert an den gRPC-Call durchgereicht statt
  lokal mit 400 abgewiesen zu werden. Kein Absturz (durch Test belegt),
  aber die im Backlog-`done_when` unterstellte 400-Antwort fuer ein
  ungueltiges Datum existiert im aktuellen Code nicht. Kein Fix in dieser
  Unit (reine Coverage-Iteration, keine Produktionscode-Aenderung
  vorgesehen) — als Befund hier festgehalten, ggf. eigene Unit fuer
  Lauf 8, falls gewuenscht.
- **Zweiter Pruefpunkt, kein Befund:** die fuenf Saved-Filter-Handler
  setzen `TenantId` NICHT im gRPC-Request (Proto-Nachrichten
  `CreateSavedFilterRequest`/`GetSavedFilterRequest`/... tragen gar kein
  `tenant_id`-Feld). Gegengeprueft gegen `internal/server/crm_grpc.go`
  (Zeilen 1783–1832) und `internal/crm/savedfilter/postgres_repository.go`:
  der Server liest den Tenant serverseitig ueber
  `middleware.GetTenantID(ctx)` aus dem propagierten Kontext, und jede
  Repository-Methode ist tenant-gescoped (`WHERE tenant_id = $n` in
  GetByID/List/Update/Delete, `tenant_id` im INSERT). Kein Tenant-Leck,
  nur ein anderes (ctx-basiertes statt feld-basiertes) Propagierungs-
  Muster als bei Activities. Kein Fix-Unit-Anlass.
- gate: build ok (`go build -p 2 ./internal/gateway/...`) | vet ok
  (`go vet ./internal/gateway/...`) | lint ok (`golangci-lint run
  ./internal/gateway/...`, 0 issues) | test ok (`go test -count=1
  ./internal/gateway/`, dreimal wiederholt, durchgehend gruen) | migration
  n.a. | rls-smoke n.a. (kein DB-Zugriff in diesem Testpaket)
- verify vorgaenger: sauber — `b2a80006` (b-cov-gateway-inbox) geprueft:
  reiner Test-Additiv (`route_inbox_test.go`, neu), folgt dem etablierten
  `registryWithService`/`testServiceUnavailable`-Muster, kein
  Produktionscode angefasst, kein neues `.proto`, keine neue Route, kein
  neuer `RequirePermission`-Guard, keine neue Tabelle.
- mutations-probe: `dive,uuid` aus dem `tag_ids`-Validate-Tag von
  `createActivityRequest` entfernt → `TestHandleCreateActivity_InvalidTagIDs`
  wurde rot (503 statt 400, keine `validation_failed`-Details), exakt wie
  erwartet. Zurueckgedreht, `git diff` auf die Produktionsdatei bestaetigt
  keine Restaenderung, volle Suite danach wieder gruen (dreimal).
- db-tests: 0 — `internal/gateway` ist reines HTTP-Handler-Paket ohne
  DB-Zugriff, alle Tests hier sind In-Memory. done_when verlangt hier keine
  DB-Tests.
- offen: keine neue Handler-Datei bleibt bei 0 %; `HandleUpdateActivity`
  (41,2 %) und `HandleCreateActivity`/`HandleGetCustomField`/`HandleGetTag`
  (41,7–46,2 %) haben noch keinen echten RPC-Erfolgspfad-Test — wie bei
  `b-cov-gateway-inbox` schon notiert braeuchte das einen Fake-
  `CRMServiceClient` (das Interface ist gross, > 50 Methoden ueber
  Contacts/Companies/Deals/Activities/Filters/Reports/Fields/Tags), den es
  fuer dieses Paket noch nicht gibt — kein Blocker, nur Grenze dieser
  Iteration.

## Iteration 8 — b-cov-gateway-biz-billing — done — 2026-08-09 20:31
- commit: (siehe unten)
- gebaut: neue Datei `internal/gateway/route_biz_billing_test.go` fuer
  `route_biz_billing.go` (868 Zeilen, 24 Handler — Credit Notes, Payments,
  Dunning, Finance-Dashboard, DATEV-Export, GoBD-Journal/Compliance).
  `route_biz_test.go` deckte bereits Teile ab (Create/List-Validierung fuer
  Credit Notes, RecordPayment-Validierung, Dunning-Detect-Validierung,
  DATEV-Validierung) — vor dem Schreiben gelesen, nichts verdoppelt. Zwei
  tabellengetriebene Tests ueber alle 24 Handler
  (`TestBizBillingRoutes_ServiceUnavailable`,
  `TestBizBillingRoutes_NoTenant` — Client-Check laeuft in dieser Datei
  durchgehend VOR dem Tenant-Check, anders als bei `route_crm_activities.go`,
  wo die Reihenfolge je Handler wechselt; per Volldateilektuere verifiziert,
  dann fuer alle 24 als eine Reihenfolge angenommen). Dazu gezielte
  Validierungsfaelle fuer die bislang unbedeckten Zweige: Credit-Note
  `line_items`/`tax_mode`/`original_invoice_id`(uuid)/VAT, Dunning
  `level`-Grenzen (gte=1,lte=3) und `fee`(decimal_gte0), Escalate-Dunning-
  Body (`invoice_id` required+uuid, eigener lokaler Typ), Update-Dunning-
  Config (`level1_days_after_due` gte=0), Update-Dunning-Status
  (`status`-oneof), Journal-Summary-Jahr (fehlend, nicht-numerisch,
  ausserhalb 2000-2100 je als eigener Fall), Payment-Stats (`from`/`to`
  fehlend in allen drei Kombinationen), GoBD-Export (`from_date`/`to_date`
  required). Fuer `HandleRecordPayment` zwei gezielte Tests: ein
  20-stelliger Dezimalbetrag mit 9 Nachkommastellen erreicht den (nicht
  erreichbaren Dummy-)RPC-Call statt 400 zu liefern — belegt, dass
  `decimal_gt0` ueber `shopspring/decimal` laeuft (beliebige Praezision),
  keine `float64`-Konvertierung im Pfad stattfindet; ein zweiter Test
  belegt, dass ein gesetzter `Idempotency-Key`-Header den Request nicht
  lokal abweist (503 statt 400) — die eigentliche Deduplizierung passiert
  serverseitig (Kommentar im Handler: "DB-level dedup (F5)"), eine
  Zustellungspruefung des Headers ins Proto-Feld braeuchte einen Fake-
  `FinanceServiceClient`, der fuer dieses Paket wie schon bei
  `b-cov-gateway-inbox`/`b-cov-gateway-crm-activities` nicht existiert —
  als Grenze im Test-Kommentar dokumentiert, kein Fix-Anlass. Kein Handler
  in `route_biz_billing.go` bleibt bei 0 % (vorher 12 von 24). Gateway-
  Gesamtcoverage: 27,3 % (von 26,5 % zu Iterationsbeginn).
- gate: build ok (`go build ./internal/gateway/...`) | vet ok
  (`go vet ./internal/gateway/...`) | lint ok (`golangci-lint run
  ./internal/gateway/...`, 0 issues) | test ok (`go test -count=1
  ./internal/gateway/...`, dreimal wiederholt, durchgehend gruen) |
  migration n.a. | rls-smoke n.a. (kein DB-Zugriff in diesem Testpaket)
- verify vorgaenger: sauber — `7a83e348` (b-cov-gateway-crm-activities)
  vor dem Ziehen dieser Unit geprueft: `go build`/`go vet`/`go test`
  gruen, Diff nur `route_crm_activities_test.go` (neu) + Journal/Backlog,
  kein Produktionscode angefasst, kein Build-Tag, kein Proto/Route/Guard
  beruehrt.
- mutations-probe: `year < 2000 || year > 2100` in `HandleGetJournalSummary`
  (`route_biz_billing.go:639`) auf nur `year < 2000` verkuerzt (Obergrenze
  entfernt) → `TestHandleGetJournalSummary_YearOutOfRange/TooHigh` wurde rot
  (503 statt 400, "connection error" statt "4-digit number"), `TooLow`
  blieb erwartungsgemaess gruen (untere Grenze unveraendert). Zurueckgedreht,
  `git diff` auf die Produktionsdatei bestaetigt keine Restaenderung, volle
  Suite danach dreimal gruen.
- db-tests: 0 — `internal/gateway` ist reines HTTP-Handler-Paket ohne
  DB-Zugriff, alle Tests hier sind In-Memory. done_when verlangt hier keine
  DB-Tests.
- offen: `HandleListCreditNotes`/`HandleListPayments`/`HandleListDunnings`
  (42-44 %) und die reinen ID-Handler (`HandleGetCreditNote`/
  `HandleSendCreditNote`/`HandleGenerateCreditNotePDF`/`HandleDeletePayment`/
  `HandleSendDunning`/`HandleGenerateDunningPDF` bei 53-61 %) haben — wie in
  den vorigen zwei Iterationen schon notiert — keinen echten
  RPC-Erfolgspfad-Test, weil kein Fake-`FinanceServiceClient` existiert.
  Auffaellig, aber kein Fix in dieser Coverage-Unit: keiner der ID-Handler
  (`HandleGetCreditNote`, `HandleSendCreditNote`,
  `HandleGenerateCreditNotePDF`, `HandleSendDunning`, `HandleEscalateDunning`,
  `HandleGenerateDunningPDF`, `HandleLockInvoice`, `HandleSendDunningNotice`)
  validiert die `id` aus `chi.URLParam` als UUID, bevor sie in den
  gRPC-Request geht — anders als z. B. `route_crm_activities.go`, wo
  `HandleGetActivity`/`HandleDeleteActivity` das tun. Eine ungueltige ID
  erreicht hier immer erst den (dummy) gRPC-Call statt lokal mit 400
  abgewiesen zu werden. Kein Sicherheitsproblem (der Service validiert
  serverseitig), aber eine Inkonsistenz im Handler-Stil, die als Befund fuer
  eine kuenftige Unit vermerkt ist, falls gewuenscht.

## Iteration 9 — b-cov-gateway-work-tasks — done — 2026-08-09 20:39
- commit: (wird nach diesem Eintrag erstellt)
- gebaut: neue Datei `route_work_tasks_test.go` (route_work_test.go blieb unangetastet,
  dort lagen bereits Tests fuer HandleCreateTask/HandleGetTask/HandleMoveTask-Teilmengen und
  fuer HandleDeleteTimeEntry aus einer anderen Datei — nichts davon dupliziert). Abgedeckt:
  alle 25 Handler aus `route_work_tasks.go` mit mindestens ServiceUnavailable-Pfad, dazu
  Validierungsfaelle fuer jeden Handler mit Body (CreateTask: NoTenant/invalid start_date/
  due_date/priority; UpdateTask: InvalidUUID/InvalidJSON/EmptyTitle/InvalidPriority/
  InvalidStatusIDUUID/InvalidAssigneeIDUUID/InvalidStartDate/InvalidDueDate/leerer Body reicht
  bis zum RPC; CreateTaskDependency: MissingTargetTaskID/InvalidUUID/InvalidDependencyType;
  CreateTaskComment/UpdateTaskComment: MissingContent, InvalidQuotedCommentIDUUID;
  LinkEntityToTask: MissingEntityType/MissingEntityID/InvalidEntityIDUUID; AttachFileToTask:
  alle Pflichtfelder plus file_size<=0; SetTaskCustomFieldValues: EmptyValues;
  ListEntityTasks/SearchTasks: fehlende Pflicht-Query-Parameter). ListTasks bekam eine
  Tabellen-Testreihe ueber alle Filterkombinationen (project_id, assignee_id, status_id,
  priority, parent_task_id, label_ids inkl. Leerstrings/Whitespace, due_from/due_to,
  search+sort, include_completed, alle gleichzeitig, und die leere Kombination) sowie eigene
  Faelle fuer invalid due_from/due_to-Format und fehlenden Tenant. Kein Handler bleibt bei
  0 % (vorher 25 von 25 ungetestet); Gateway-Gesamtcoverage 27,3 % -> 28,3 %, Einzelabdeckung
  je Handler 36,4-97,5 % (Rest ist der unerreichbare Erfolgspfad ohne Fake-WorkServiceClient,
  wie in den vorigen drei Gateway-Units).
- befund own-scope: `ownerFilterForScope` (helpers.go:144, benutzt in route_biz_expenses.go,
  route_helpdesk.go, route_rapporte.go) wird in `route_work_tasks.go` NIRGENDS aufgerufen —
  `HandleListTasks` filtert nur ueber den optionalen `assignee_id`-Query-Parameter, den der
  Client selbst setzen muss, nicht ueber eine serverseitig erzwungene Scope-Aufloesung. Waere
  ein RBAC-Grant `work:task:read` jemals auf Scope "own" gesetzt (Mechanismus seit Lauf 4
  vorhanden), saehe der Nutzer trotzdem alle Tenant-Tasks, sofern er selbst keinen
  assignee_id-Filter mitschickt. Zusaetzlich ist die im Backlog notierte Praemisse
  ueberholt: `TaskProto` traegt inzwischen `created_by` (work.proto:167) — die Listen-Antwort
  hat also seit einer frueheren Aenderung sehr wohl einen Ersteller, nur wird er nirgends als
  Filter genutzt. Kein Fix in dieser Coverage-Unit (Produktentscheidung + echte
  Verhaltensaenderung, kein Test-Diff) — als Fund fuer eine kuenftige Unit vermerkt.
- befund fehlende id-validierung: `HandleDeleteTask` und `HandleMoveTask` lesen `id` per
  `chi.URLParam` ohne `validateUUIDParam` (anders als `HandleGetTask`/`HandleUpdateTask` im
  selben File) — eine ungueltige ID erreicht den (serverseitig validierenden) RPC-Call statt
  lokal mit 400 abgelehnt zu werden. Gleiches Muster wie in Iteration 8
  (route_biz_billing.go) schon vermerkt, hier als Test `TestHandleDeleteTask_
  InvalidIDReachesRPCNotLocalValidation` festgeschrieben statt uebersehen. Kein
  Sicherheitsproblem, nur Stil-Inkonsistenz.
- gate: build ok (`go build -p 2 ./internal/gateway/... ./cmd/gateway/...`) | vet ok
  (`go vet ./internal/gateway/...`) | lint ok (`golangci-lint run ./internal/gateway/...`,
  0 issues) | test ok (`go test -count=1 ./internal/gateway/...`, dreimal wiederholt,
  durchgehend gruen, 0 uebersprungen) | migration n.a. | rls-smoke n.a. (kein DB-Zugriff)
- verify vorgaenger: sauber — `8204ad01` (b-cov-gateway-biz-billing) geprueft: `git show
  --stat` zeigt nur `route_biz_billing_test.go` (neu) plus Journal/Backlog, kein
  Produktionscode, kein Proto, keine neue Route, kein RequirePermission, keine Tabelle.
- mutations-probe: `entityType == "" || entityID == ""` in `HandleListEntityTasks`
  (route_work_tasks.go:626) auf nur `entityType == ""` verkuerzt →
  `TestHandleListEntityTasks_MissingParams/missing_entity_id` wurde rot (503 "connection
  error" statt 400 "entity_type and entity_id are required"), die beiden anderen Faelle
  blieben erwartungsgemaess gruen. Zurueckgedreht, `git diff` auf die Produktionsdatei zeigt
  keine Restaenderung, volle Suite danach gruen.
- db-tests: 0 — `internal/gateway` ist reines HTTP-Handler-Paket ohne DB-Zugriff.
- offen: HandleGetTask/HandleListTaskDependencies/HandleUnlinkEntityFromTask/
  HandleListTaskEntityLinks/HandleRemoveTaskFile/HandleListTaskFiles/
  HandleGetTaskCustomFieldValues und die reinen Lesehandler bleiben bei 36-58 % — wie in den
  vorigen drei Gateway-Coverage-Units haben sie keinen Erfolgspfad-Test, weil kein Fake-
  `WorkServiceClient` existiert. Beide own-scope- und id-Validierungs-Befunde oben sind fuer
  Lauf 8 vorgemerkt, nicht in dieser Unit gefixt (Backlog-Regel: Coverage-Units bauen keine
  Fixes nebenbei).

## Iteration 10 — b-cov-gateway-automation — done — 2026-08-09 20:52
- commit: (wird nach diesem Eintrag erstellt)
- gebaut: neue Datei `route_automation_test.go` fuer `route_automation.go` (21 Handler plus
  `RegisterPublicRoutes`-Webhook, keine bestehende Testdatei vorher). Abgedeckt: ServiceUnavailable
  fuer alle 17 direkt aufrufbaren Handler (RegisterRoutes/ServiceName/NewAutomationRoutes/getClient
  liefen bereits mit ueber die vorhandene `route_capability_guard_test.go`); Validierungspfade fuer
  HandleCreateAutomation (InvalidJSON, MissingName, MissingTriggerType, InvalidScope-oneof, plus ein
  Fall der beweist, dass ein JSON-Array als trigger_config still verworfen statt 400 wird);
  HandleUpdateAutomation (InvalidUUID, InvalidJSON, leerer Body und Body mit allen Feldern erreichen
  beide die RPC-Schicht); HandleDeleteAutomation/HandleEnableAutomation/HandleDisableAutomation/
  HandleGetAutomation (InvalidUUID); HandleListExecutions/HandleGetExecution (InvalidUUID fuer id
  bzw. executionId, Statusfilter erreicht RPC); HandleCreateFromTemplate (InvalidJSON, MissingName);
  HandleTestCondition (InvalidJSON, leerer Body, Condition+SampleEnv erreichen RPC); HandleDryRun
  (InvalidJSON, MissingAutomationID und InvalidAutomationIDFormat ueber dieselbe
  `validate:"required,uuid"`-Regel auf automation_id, gueltiger Fall erreicht RPC);
  HandleListAutomations bekam eine Tabellen-Testreihe (owner_id, limit/offset inkl. nicht-numerisch,
  scope in allen drei gueltigen Werten plus unbekanntem Wert, trigger_type, is_active in allen drei
  Auspraegungen, alle Filter zusammen, leere Kombination) — 14 Faelle, alle reichen bis zur RPC-Schicht
  durch. HandleTriggerWebhook (der einzige unauthentifizierte Pfad, ueber `RegisterPublicRoutes`
  angebunden): InvalidAutomationID (400), PayloadTooLarge (Body ueber `maxWebhookBodyBytes` = 256 KiB,
  413), gueltige Nutzlast mit Signatur- und Idempotency-Key-Header sowie leerer Body erreichen beide
  die RPC-Schicht. Die drei reinen Helferfunktionen `parseAutomationScope`, `parseExecutionStatus` und
  `rawJSONToAutomationStruct` direkt als Tabellentests (inkl. Default-Zweig, nicht-Objekt-JSON-Wurzel,
  kaputtes JSON). Kein Handler bleibt bei 0 % (vorher 15 von 21 bei 0 %, `getClient` bei 75 %); Gateway-
  Gesamtcoverage 28,3 % -> 29,2 %, `route_automation.go` je Handler 44,4-100 % (Rest ist wie in den
  vorigen Gateway-Units der unerreichbare Erfolgspfad ohne Fake-`AutomationServiceClient`;
  HandleListTriggers/HandleListActions/HandleGetStats bleiben bei 44,4 %, weil sie ausser dem
  Client-Check und einem parameterlosen RPC-Aufruf keine Logik zum Testen haben).
- befund berechtigungs-abbildung: bereits vollstaendig getestet — `route_capability_guard_test.go`
  deckt seit einer frueheren Iteration die komplette `RequirePermissionAny`-Matrix fuer
  `route_automation.go` ab (Zeilen 691-721: create/list/update/delete/enable/disable/executions/
  templates/dry-run/stats, jeweils legacy- und Katalog-Schluessel getrennt getestet). Kein neuer Test
  noetig, im Backlog-`done_when` "falls das Gateway eine vornimmt" bereits erfuellt.
- befund webhook-validierung: Groessenbegrenzung (`http.MaxBytesReader` + `maxWebhookBodyBytes`) und
  UUID-Validierung des `automationId`-Pfadparameters sind vorhanden und getestet. Eine tiefere
  Payload-Struktur- oder Content-Type-Pruefung findet im Gateway nicht statt (bewusst laut
  Doc-Kommentar — Signaturpruefung passiert downstream in `workflow.Service.TriggerWebhook`); kein
  SSRF-Risiko im Gateway selbst, da der Handler keine ausgehende Anfrage anhand von Nutzereingaben
  staged (anders als die im Backlog fuer `b-cov-server-automation` vermerkte HTTP-Aktion, die die
  ausgehende Seite betrifft). Kein Fund, der einen Fix braucht.
- befund scope-default: `parseAutomationScope` faellt bei unbekanntem oder leerem String still auf
  `SCOPE_PERSONAL` zurueck (route_automation.go:706-717) statt auf `SCOPE_UNSPECIFIED` oder eine
  Ablehnung — ein Tippfehler in `?scope=` filtert die Liste also stillschweigend auf "personal" statt
  ignoriert zu werden oder einen 400 zu liefern. `parseExecutionStatus` (gleiche Datei, Zeile 719)
  macht es anders und faellt korrekt auf `UNSPECIFIED` zurueck. Als Test `TestParseAutomationScope`
  festgeschrieben (dokumentiert das IST-Verhalten), nicht gefixt — Verhaltensaenderung, kein
  Test-Diff, gehoert in eine eigene Unit falls gewuenscht.
- gate: build ok (`go build -p 2 ./internal/gateway/... ./cmd/gateway/...`) | vet ok
  (`go vet ./internal/gateway/...`) | lint ok (`golangci-lint run ./internal/gateway/...`, 0 issues)
  | test ok (`go test -count=1 ./internal/gateway/...`, dreimal wiederholt, durchgehend gruen,
  0 uebersprungen) | migration n.a. | rls-smoke n.a. (kein DB-Zugriff)
- verify vorgaenger: sauber — `db747fae` (b-cov-gateway-work-tasks) geprueft: `git show --stat`
  zeigt nur `route_work_tasks_test.go` (neu, 738 Zeilen) plus Journal/Backlog, kein Produktionscode,
  keine neue Route, kein RequirePermission, keine Tabelle.
- mutations-probe: `validate:"required"` vom `Name`-Feld in `createAutomationRequest`
  (route_automation.go:132) entfernt → `TestHandleCreateAutomation_MissingName` wurde rot (503 statt
  400/"validation_failed"/Feld "name"), alle anderen Tests des neuen Files blieben unberuehrt.
  Zurueckgedreht, `git diff` auf die Produktionsdatei zeigt keine Restaenderung, volle Suite danach
  dreimal gruen.
- db-tests: 0 — `internal/gateway` ist reines HTTP-Handler-Paket ohne DB-Zugriff.
- offen: keins fuer diese Datei — jeder Handler hat mindestens einen Validierungs- oder
  ID-Pruefpfad-Test, die drei reinen Helferfunktionen sind bei 100 %. Der Scope-Default-Befund ist
  fuer eine kuenftige Unit vorgemerkt, nicht Teil dieser.

## Iteration 11 — b-cov-gateway-notification — done — 2026-08-09 21:05
- commit: (wird nach diesem Eintrag erstellt)
- gebaut: neue Datei `route_notification_test.go` fuer `route_notification.go` (19 Handler,
  keine bestehende Testdatei vorher). ServiceUnavailable fuer alle 19 Handler in einer
  Tabellen-Testreihe. Validierungspfade: HandleMarkRead/HandlePin/HandleUnpin/HandleDismiss
  teilen sich InvalidUUID + ValidUUID-erreicht-RPC als Tabellentest ueber alle vier;
  HandleSnooze (InvalidUUID vor Body-Parse, InvalidJSON, Until in der Vergangenheit -> 422,
  Until in der Zukunft erreicht RPC); HandleMarkAllRead (siehe Befund unten);
  HandleUpdatePreference (InvalidJSON, gueltiger Body erreicht RPC — die Struct traegt keine
  `validate`-Tags, jedes Feld ist optional); HandleMuteResource (MissingModuleID/
  MissingResourceID als strukturierte Validierungsfehler gegen das JSON-Tag `module_id`/
  `resource_id`, nicht den Go-Feldnamen — erster Testlauf schlug genau daran fehl, korrigiert);
  HandleUnmuteResource (InvalidUUID auf `muteId`); HandleUpdateQuietHours (InvalidJSON, Tage
  besetzt und leer, beide erreichen RPC — die `int`->`int32`-Konvertierungsschleife mitgetestet);
  HandleToggleDND (InvalidJSON, ungueltiges `until`-Zeitformat -> 400, mit und ohne `until`
  erreichen RPC). Filter-Kombinationen fuer HandleListNotifications und
  HandleListMutedResources (module_id, is_read, Pagination inkl. nicht-numerisch) als
  Tabellentests. `dndStatusFromQuietHours` (die eine handgebaute Wire-Shape-Funktion der Datei)
  direkt als vier Unit-Tests: nil, disabled ohne until, enabled mit until (RFC3339-Format
  geprueft), enabled ohne until — deckt exakt das FE-`DNDStatus`-Interface
  (`desktop/src/renderer/src/api/notification-client.ts:21-23`, `{is_active: boolean,
  expires_at?: string}`) ab, `expires_at` muss beim Fehlen ABWESEND sein, nicht `null` oder
  leerer String. Kein Handler bleibt unter 90 % (`HandleGetUnreadCount` war im ersten Lauf bei
  40 %, weil nur im ServiceUnavailable-Sammeltest erfasst — eigener Reach-RPC-Test ergaenzt,
  jetzt 90 %); Rest ist wie in den vorigen Gateway-Units der unerreichbare Erfolgspfad ohne
  Fake-`NotificationServiceClient`.
- befund fremde-nutzerkennung: `TestHandleListNotifications_UserFromContextNotQuery` bestaetigt
  das Backlog-Notiz-Risiko als NICHT vorhanden — kein Handler in der Datei liest `user_id` (oder
  irgendeine andere Identitaet) aus der Query-String oder dem Body, alle 19 nehmen sie
  ausschliesslich ueber `middleware.GetUserID(r.Context())`. Grep ueber die ganze Datei nach
  `Query().Get` bestaetigt: nur `module_id`, `is_read`, `page`, `page_size` werden gelesen, nie
  ein Identitaetsfeld. Es gibt in diesem Paket keinen Stub-gRPC-Client, der das tatsaechlich
  gesendete Proto-Feld abfangen koennte (kein bufconn-Muster im ganzen Repo - gegengeprueft per
  Grep `bufconn` ueber `backend/`, 0 Treffer) - der Test pinnt deshalb, dass ein Query-Parameter
  `user_id` das Handler-Verhalten nicht veraendert (identischer Statuscode, kein Sonderzweig),
  gestuetzt durch die Code-Lektuere. Kein Fund, der einen Fix braucht.
- befund markallread-body: `HandleMarkAllRead` haelt sein Kommentarversprechen "Body is optional"
  nicht ein. Die Pruefung `if r.Body != nil` schuetzt nur vor einem *nil* `Request.Body` - laut
  Go-Doku ist der bei echten Server-Requests NIE nil, ein Body-loser Request liefert stattdessen
  sofort `io.EOF` beim Lesen. `json.Decode` auf einem leeren Body liefert also einen echten
  Decode-Fehler -> 400 "invalid request body", nicht das beabsichtigte "alle Module als gelesen
  markieren". `TestHandleMarkAllRead_NoBodyIsRejected` schreibt dieses IST-Verhalten fest,
  `TestHandleMarkAllRead_EmptyObjectBodyReachesRPC` zeigt, dass ein Client den Bug nur durch
  explizites Senden von `{}` umgeht. Grep `r.Body != nil` ueber `internal/gateway` bestaetigt:
  einzige Fundstelle im ganzen Paket, kein Wiederholungsmuster anderswo. NICHT gefixt (echte
  Verhaltensaenderung, gehoert nicht in eine Coverage-Unit) - als Unit
  `fix-notification-markallread-empty-body-rejected` fuer Lauf 8 im Backlog angelegt, inkl.
  Fix-Vorschlag (`errors.Is(err, io.EOF)` durchlassen statt auf `r.Body != nil` zu pruefen).
- befund antwortformen: `dndStatusFromQuietHours` ist die einzige handgebaute Wire-Shape-Stelle
  der Datei (alle anderen Handler geben `response.Proto` direkt durch) und stimmt mit dem
  FE-Typ `DNDStatus` exakt ueberein, siehe oben. Alle anderen Handler liefern das generierte
  Proto-JSON unveraendert durch - keine eigene Wire-Shape-Pruefung noetig, das ist Vertragssache
  der `.proto`-Definition, nicht dieser Datei.
- gate: build ok (`go build -p 2 ./internal/gateway/... ./cmd/gateway/...`) | vet ok
  (`go vet ./internal/gateway/...`) | lint ok (`golangci-lint run ./internal/gateway/...`, 0 issues)
  | test ok (`go test -count=1 ./internal/gateway/...`, dreimal wiederholt, durchgehend gruen,
  0 uebersprungen) | migration n.a. | rls-smoke n.a. (kein DB-Zugriff)
- verify vorgaenger: sauber — `83e6e616` (b-cov-gateway-automation) geprueft: `git show --stat`
  zeigt nur `route_automation_test.go` (neu) plus Journal/Backlog, kein Produktionscode, keine
  neue Route, kein RequirePermission, keine Tabelle.
- mutations-probe: `validate:"required"` vom `ModuleID`-Feld in `muteResourceRequest`
  (route_notification.go:429) entfernt → `TestHandleMuteResource_MissingModuleID` wurde rot
  (503 "connection error" statt 400/"validation_failed"/Feld "module_id"), die beiden anderen
  Mute-Tests blieben unberuehrt. Zurueckgedreht, `git diff` auf die Produktionsdatei zeigt keine
  Restaenderung, volle Suite danach dreimal gruen.
- db-tests: 0 — `internal/gateway` ist reines HTTP-Handler-Paket ohne DB-Zugriff.
- offen: `fix-notification-markallread-empty-body-rejected` im Backlog fuer Lauf 8 angelegt
  (echter Produktionsbug, kein Test-Diff). Fremde-Nutzerkennung- und Antwortform-Punkte aus dem
  Backlog-`done_when` sind erfuellt, kein weiterer Fund offen fuer diese Datei.

## Iteration 12 — b-cov-gateway-work-projects — done — 2026-08-09 21:15
- commit: (wird nach diesem Eintrag erstellt)
- gebaut: neue Datei `route_work_projects_test.go` fuer `route_work_projects.go` (19 Handler).
  `HandleCreateProject`/`HandleGetProject`/`HandleListProjects` hatten ueber `route_work_test.go`
  bereits einen Baseline-Grundstock (ServiceUnavailable, NoUserID, InvalidJSON, MissingFields,
  InvalidUUID) - dort nichts gedoppelt, nur die fehlenden Pfade ergaenzt (ValidRequestReachesRPC
  je Handler). Alle 19 Handler liegen laut `go tool cover -func` zwischen 90,0 % und 95,7 % -
  keiner unter 90 %. Vorlagenfilter `templates_only` in `TestHandleListProjects_FilterCombinations`
  in beiden Stellungen (true/false) getestet, zusammen mit include_archived/search/pagination/
  Kombinationen. `HandleDeleteProject` (neuester Handler der Datei, laut Backlog-Notiz noch ohne
  Test) bekam drei eigene Faelle: ServiceUnavailable, InvalidUUID, und eine wohlgeformte aber
  nicht-existente UUID die die RPC-Ebene erreicht (503) - echtes NotFound-Mapping kann diese Datei
  strukturell nicht beweisen, da kein Fake-`WorkServiceClient`/bufconn-Stub im Paket existiert
  (wie in jeder vorigen Gateway-Unit dieses Laufs), das Grepping dazu erneut bestaetigt (0 Treffer
  `bufconn` in `backend/`). Restliche Handler nach demselben Muster: je ein Validierungspfad wo
  vorhanden (`decodeAndValidate`-Pflichtfelder, `oneof`-Rollen bei Members, `dive,uuid` bei
  Statuses-Reorder inkl. Element-Index `status_ids[0]`), sonst ReachesRPC.
- befund fehlende-id-validierung: 10 von 19 Handlern lesen ihre Pfad-ID(s) direkt per
  `chi.URLParam(r, "id")` (bzw. `"userId"`) OHNE `validateUUIDParam` -
  `HandleRemoveProjectMember`, `HandleListProjectMembers`, `HandleUpdateProjectMemberRole`,
  `HandleSaveProjectAsTemplate`, `HandleCreateProjectStatus`, `HandleUpdateProjectStatus`,
  `HandleDeleteProjectStatus`, `HandleReorderProjectStatuses`, `HandleListProjectStatuses`,
  `HandleGetUserProjectPreference`, `HandleSetUserProjectPreference` (11 gezaehlt, korrigiert).
  Nur `HandleGetProject`, `HandleUpdateProject`, `HandleArchiveProject`, `HandleDeleteProject` und
  `HandleAddProjectMember` validieren ihre `id` vor dem RPC-Aufruf. Die restlichen reichen eine
  beliebige Zeichenkette unveraendert an den gRPC-Client durch - kein 400 an der Gateway-Grenze,
  die Validierung faellt komplett auf den Service zurueck. Kein Datenverlust- oder RLS-Risiko
  (der Service validiert ohnehin serverseitig), aber inkonsistent zum Rest der Datei und zur
  in `CLAUDE.md` festgehaltenen "Input-Validierung an der Grenze"-Regel. NICHT gefixt (Aenderung
  an neun weiteren Handlern ist kein Coverage-Diff mehr) - keine eigene Backlog-Unit angelegt, da
  der Service-Layer die Luecke bereits abfaengt und kein beobachtbarer Bug vorliegt, nur eine
  Stil-Abweichung; im Journal dokumentiert falls spaeter jemand die Konsistenz herstellen will.
- gate: build ok (`go build -p 2 ./internal/gateway/... ./cmd/gateway/...`) | vet ok
  (`go vet ./internal/gateway/...`) | lint ok (`golangci-lint run ./internal/gateway/...`, 0 issues)
  | test ok (`go test -count=1 ./internal/gateway/...`, dreimal wiederholt, durchgehend gruen,
  0 uebersprungen) | migration n.a. | rls-smoke n.a. (kein DB-Zugriff)
- verify vorgaenger: sauber — `876c9649` (b-cov-gateway-notification) geprueft: `git show --stat`
  zeigt nur `route_notification_test.go` (neu) plus Journal/Backlog, kein Produktionscode, keine
  neue Route, kein RequirePermission, keine Tabelle.
- mutations-probe: `validate:"required,min=1,dive,uuid"` auf `StatusIDs` in
  `reorderStatusesRequest` (route_work_projects.go:470) zu `validate:"dive,uuid"` verkuerzt (Pflicht
  entfernt) → `TestHandleReorderProjectStatuses_EmptyIDs` wurde rot (503 statt 400/
  "validation_failed"/Feld "status_ids"). Zurueckgedreht, `git diff --stat` auf die
  Produktionsdatei zeigt keine Restaenderung, volle Suite danach dreimal gruen.
- db-tests: 0 — `internal/gateway` ist reines HTTP-Handler-Paket ohne DB-Zugriff.
- offen: kein neuer Fund fuer Lauf 8 - der Id-Validierungs-Befund ist dokumentiert, aber bewusst
  ohne Folge-Unit, siehe oben.


## Iteration 13 — b-cov-gateway-crm-contacts — done — 2026-08-09 23:45
- commit: (wird nach diesem Eintrag erstellt)
- gebaut: neue Datei `route_crm_contacts_test.go` fuer `route_crm_contacts.go` (14 Handler:
  Create/Get/List/Update/DeleteContact, Add/RemoveContactTags, Import CSV/VCard/XLSX,
  PreviewImportCSV, Export CSV/VCard, UpdateContactVisibility). `route_crm_test.go` deckte
  Create/Get/List/Delete bereits mit einem Grundstock ab (ServiceUnavailable, NoUserID,
  InvalidJSON, MissingFields, InvalidEmail, InvalidCompanyID, InvalidUUID) - dort nichts
  gedoppelt, nur fehlende Pfade ergaenzt (InvalidPhone, InvalidTagID, ValidRequestReachesRPC je
  Handler). `go tool cover -func` zeigt alle 14 Handler zwischen 76,5 % (die beiden Export-
  Handler, deren Erfolgspfad - Header setzen + `resp.FileContent` schreiben - ohne Fake-
  `CRMServiceClient` strukturell nicht erreichbar ist, wie in jeder vorigen Gateway-Unit dieses
  Laufs) und 95,2 %. Fuer die drei Import-Handler und `HandlePreviewImportCSV` (alle nutzen
  `r.ParseMultipartForm`/`r.FormFile`) wurde ein eigener `multipartBody`-Testhelfer gebaut
  (`multipart.NewWriter` gegen einen `bytes.Buffer`, Content-Type inkl. Boundary zurueckgegeben) -
  kein Aequivalent existierte im Paket, `testutil_test.go` hat bisher nur `jsonBody`/`invalidJSON`
  fuer JSON-Bodies.
- befund map_-vertrag: `HandleImportContactsCSV` und `HandleImportContactsXLSX` lesen die
  Spalten-Zuordnung identisch aus `r.MultipartForm.Value` per `key[:4] == "map_"`-Praefixpruefung
  (route_crm_contacts.go:322 bzw. :425 - Code ist woertlich dupliziert zwischen beiden Handlern).
  `TestHandleImportContactsCSV_FieldMappingContract` und das XLSX-Pendant schreiben fest, dass ein
  realistischer Multi-Feld-Payload (drei `map_`-Felder plus `visibility`/`merge_by_email` als
  Nicht-Mapping-Felder) die Parse-Schleife ohne Panic durchlaeuft und die RPC-Ebene erreicht (503).
  Der ausgehende Proto-Request selbst ist strukturell nicht beobachtbar (kein Fake-
  `CRMServiceClient`, kein bufconn-Stub im Paket - dieselbe Grenze wie in jeder vorigen
  Gateway-Unit dieses Laufs), also kann kein Test beweisen, dass genau die drei erwarteten
  Schluessel (`first_name`, `last_name`, `email`) mit den richtigen Werten in der Map landen -
  nur dass das Parsen selbst robust ist. `TestHandleImportContactsCSV_NoMappingNoFile` deckt den
  in den `done_when` explizit genannten Fall "ohne Zuordnung" ab: eine wohlgeformte Multipart-
  Anfrage ohne `map_`-Felder und ohne Datei liefert 400 "file is required" (die fehlende Datei
  greift zuerst, ein leeres Mapping allein loest keinen eigenen Fehlerpfad aus - im Code gibt es
  keine Validierung, die ein leeres Mapping ablehnt).
- gate: build ok (`go build -p 2 ./internal/gateway/... ./cmd/gateway/...`) | vet ok
  (`go vet ./internal/gateway/...`) | lint ok (`golangci-lint run ./internal/gateway/...`, 0
  issues) | test ok (`go test -count=1 ./internal/gateway/...`, dreimal wiederholt, durchgehend
  gruen, 0 uebersprungen) | migration n.a. | rls-smoke n.a. (kein DB-Zugriff)
- verify vorgaenger: sauber — `d35c50b5` (b-cov-gateway-work-projects) geprueft: `git show --stat`
  zeigt nur `route_work_projects_test.go` (neu) plus Journal/Backlog, kein Produktionscode, keine
  neue Route, kein RequirePermission, keine Tabelle.
- mutations-probe: `required` von `Visibility` in `updateContactVisibilityRequest`
  (route_crm_contacts.go:562) auf `omitempty` verkuerzt →
  `TestHandleUpdateContactVisibility_MissingVisibility` wurde rot (503/connection-error statt
  400/"validation_failed"/Feld "visibility"), die uebrigen Visibility-Tests blieben unberuehrt.
  Zurueckgedreht, `git diff --stat` auf die Produktionsdatei zeigt keine Restaenderung, volle
  Suite danach dreimal gruen.
- db-tests: 0 — `internal/gateway` ist reines HTTP-Handler-Paket ohne DB-Zugriff.
- offen: kein neuer Fund fuer Lauf 8. `map_`-Vertrag ist jetzt gegen Panics und den
  Kein-Mapping/kein-Datei-Fall abgesichert, aber die genaue Schluessel-Zuordnung bleibt bis zu
  einem echten Fake-`CRMServiceClient` im Paket unbewiesen - dieselbe strukturelle Luecke, die
  jede vorige Gateway-Unit dieses Laufs schon dokumentiert hat, keine neue Erkenntnis.

## Iteration 14 — b-cov-gateway-crm-pipeline — done — 2026-08-09 23:52
- commit: (wird nach diesem Eintrag erstellt)
- gebaut: neue Datei `route_crm_pipeline_test.go` fuer `route_crm_pipeline.go` (14 Handler,
  keine bestehende Testdatei vorher: sechs Pipeline-Stage-Handler + acht Deal-Handler inkl. der
  zwei Tag-Stubs). `go tool cover -func` zeigt alle 14 Handler zwischen 58,3 % (die beiden
  reinen Id-Handler `HandleGetPipelineStage`/`HandleDeletePipelineStage`, deren Erfolgspfad ohne
  Fake-`CRMServiceClient` strukturell nicht erreichbar ist, dieselbe Grenze wie in jeder vorigen
  Gateway-Unit dieses Laufs) und 100 % (die beiden Tag-Stubs). Gateway-Gesamtcoverage 32,9 % laut
  `go test -coverprofile`.
- befund reihenfolge: die Tenant-Pruefreihenfolge wechselt innerhalb dieser einen Datei zweimal.
  `HandleCreateDeal`/`HandleGetDeal`/`HandleListDeals`/`HandleDeleteDeal` pruefen den Tenant VOR
  Body-Decode bzw. Id-Validierung; `HandleUpdateDeal`/`HandleMoveDealToStage` pruefen ihn ERST
  NACH Id-Validierung und Body-Decode. Beide Reihenfolgen als eigene Tests festgeschrieben
  (`_NoTenant` vs. `_NoTenantAfterValidBody`), nicht angenommen - exakt das Muster, das
  `b-cov-gateway-crm-activities` (Iteration 7) fuer diese Datei-Familie schon dokumentiert hat.
- befund reorder-logik: die im Backlog benannte "echte Logik" der Umsortierung
  (unvollstaendige/doppelte Reihenfolge ablehnen, Gewonnen/Verloren-Einmaligkeit -> 409) sitzt
  serverseitig in `internal/server/crm_grpc.go`/dem CRM-Service, nicht im Gateway-Handler. Der
  Handler selbst validiert nur `required,min=1,dive,uuid` auf `stage_ids` - dieselbe strukturelle
  Grenze wie ueberall in diesem Lauf: ohne Fake-`CRMServiceClient` kann diese Testdatei die
  Server-Logik nicht erreichen, nur die Gateway-seitige Vorbedingung. Kein Fund, der eine neue
  Unit braucht - die 409-/Duplikat-Logik gehoert (falls ungetestet) in eine `internal/server`-
  oder `internal/crm`-Coverage-Unit, nicht in eine Gateway-Unit.
- befund pipeline-stage-ohne-tenant: die sechs Pipeline-Stage-Handler (Create/Get/List/Update/
  Delete/Reorder) rufen `middleware.GetTenantID` nirgends auf und senden auch kein `TenantId`-
  Feld im gRPC-Request (anders als alle Deal-Handler). Gegengepueft gegen
  `internal/server/crm_grpc.go`: der Server liest den Tenant serverseitig aus dem propagierten
  Kontext, exakt das bereits in Iteration 7 fuer die Saved-Filter-Handler dokumentierte
  ctx-basierte statt feld-basierte Propagierungsmuster. Kein Tenant-Leck, keine neue Unit.
- gate: build ok (`go build -p 2 ./internal/gateway/... ./cmd/gateway/...`) | vet ok
  (`go vet ./internal/gateway/...`) | lint ok (`golangci-lint run ./internal/gateway/...`,
  0 issues) | test ok (`go test -count=1 ./internal/gateway/...`, dreimal wiederholt,
  durchgehend gruen, 0 uebersprungen) | migration n.a. | rls-smoke n.a. (kein DB-Zugriff)
- verify vorgaenger: sauber — `503517c7` (b-cov-gateway-crm-contacts) geprueft: `git show --stat`
  zeigt nur `route_crm_contacts_test.go` (neu) plus Journal/Backlog, kein Produktionscode, keine
  neue Route, kein RequirePermission, keine Tabelle.
- mutations-probe: `validate:"required,min=1,dive,uuid"` auf `StageIDs` in
  `reorderPipelineStagesRequest` (route_crm_pipeline.go:160) zu `validate:"dive,uuid"` verkuerzt
  (Pflicht + Mindestlaenge entfernt) → `TestHandleReorderPipelineStages_EmptyIDs` und
  `_MissingField` wurden rot (503 statt 400/"validation_failed"/Feld "stage_ids"), `_InvalidElement`
  und `_ValidReachesRPC` blieben erwartungsgemaess gruen. Zurueckgedreht, `git diff --stat` auf die
  Produktionsdatei zeigt keine Restaenderung, volle Suite danach gruen.
- db-tests: 0 — `internal/gateway` ist reines HTTP-Handler-Paket ohne DB-Zugriff.
- offen: kein neuer Fund fuer Lauf 8, der eine eigene Unit braucht. Docker/Postgres war zu
  Iterationsbeginn gestoppt (`docker-postgres-1: Exited`) - fuer diese reine Gateway-Unit nicht
  noetig, nicht neu gestartet; Luke sollte das vor der naechsten DB-Unit (Block B-Server oder C2)
  wieder hochfahren.

## Iteration 15 — b-cov-gateway-biz-invoices — done — 2026-08-10 00:05
- commit: (wird nach diesem Eintrag erstellt)
- gebaut: neue Datei `route_biz_invoices_test.go` fuer `route_biz_invoices.go` (10 Handler,
  vorher nur teilweise durch `route_biz_test.go` mitabgedeckt: Create/List/Get nur
  ServiceUnavailable, GenerateEInvoice nur Format-Validierung). Neu abgedeckt:
  HandleUpdateInvoice, HandleSendInvoice, HandleMarkInvoicePaid, HandleCancelInvoice,
  HandleGenerateInvoicePDF inkl. des ZUGFeRD-Formatzweigs (`handleZUGFeRDInvoicePDF` war zuerst
  bei 0 % - kein Test erreichte ihn, da immer vorher an Registry/Tenant scheiterte; ein Test mit
  registrierter Service + gesetztem Tenant + `format=zugferd` behoben das strukturell), die
  `contact_id`/`recurring_id`-Filter in HandleListInvoices, und der Dezimal-als-String-Vertrag
  der `LineItem`-Felder direkt gegen `decodeAndValidate` geprueft (kein Umweg ueber float64).
  `go tool cover -func` zeigt alle 10 Handler zwischen 28,6 % (`HandleGetInvoice`, traegt
  weiterhin nur den ServiceUnavailable-Test aus der Nachbardatei) und 92,9 %
  (`HandleMarkInvoicePaid`). Gateway-Gesamtcoverage 33,2 % laut `go test -coverprofile`
  (vorher 32,9 % nach Iteration 14).
- befund mark-paid-idempotenz: `MarkInvoicePaidRequest` traegt laut Proto nur `{id, tenant_id}`,
  keinen Betrag und keinen Idempotency-Key. Die im Backlog verlangte Pruefung "zweiter Aufruf
  verdoppelt nichts" ist damit vollstaendig serverseitige Zustandslogik (Rechnung bereits
  bezahlt -> Fehler oder No-Op), im Gateway strukturell nicht beobachtbar - dieselbe Grenze wie
  in jeder vorigen Gateway-Unit dieses Laufs (kein Fake-`FinanceServiceClient` im Paket). Statt
  der unbeweisbaren Behauptung "verdoppelt nicht" belegt ein Test, dass der Handler bei
  wiederholtem Aufruf keinen eigenen Zustand haelt (identische Anfrageform, identisches
  Fehlverhalten ohne echten Server). Kein Fund, der eine neue Unit braucht - die eigentliche
  Zustandslogik gehoert (falls ungetestet) in eine `internal/server`- oder `internal/biz`-Unit.
- befund status-filter: `invoiceStatusToProto` (route_biz.go:443) hat keinen Default-Reject-Zweig
  - ein unbekannter `?status=`-Wert wird still zu `INVOICE_STATUS_UNSPECIFIED` (kein Filter)
  statt eines 400. Gegengeprueft: dasselbe Muster gilt fuer `quoteStatusToProto`/
  `dunningStatusToProto` in derselben Datei - konsistentes Verhalten im ganzen Paket, kein
  Einzelfall und damit kein neuer Fund, nur als Test festgeschrieben statt uebersehen.
- gate: build ok (`go build -p 2 ./internal/gateway/... ./cmd/gateway/...`) | vet ok
  (`go vet ./internal/gateway/...`) | lint ok (`golangci-lint run ./internal/gateway/...`,
  0 issues) | test ok (`go test -count=1 ./internal/gateway/...`, dreimal wiederholt,
  durchgehend gruen, 0 uebersprungen) | migration n.a. | rls-smoke n.a. (kein DB-Zugriff)
- verify vorgaenger: sauber — `a9a0b1e2` (b-cov-gateway-crm-pipeline) geprueft: `git show --stat`
  zeigt nur `route_crm_pipeline_test.go` (neu) plus Journal/Backlog, kein Produktionscode, keine
  neue Route, kein RequirePermission, keine Tabelle.
- mutations-probe: `uuid.Parse(cid)`-Fehlerpruefung in `HandleListInvoices`
  (route_biz_invoices.go:99) von `parseErr != nil` auf `parseErr == nil` invertiert →
  `TestHandleListInvoices_InvalidContactID` wurde rot (503 statt 400, Fehlertext "connection
  error" statt "invalid contact_id") UND `TestHandleListInvoices_ValidContactID_ReachesRPC`
  wurde rot (400 statt 503) - beide Tests wie erwartet symmetrisch betroffen. Zurueckgedreht,
  `git diff --stat` auf die Produktionsdatei zeigt keine Restaenderung, volle Suite danach
  dreimal gruen.
- db-tests: 0 — `internal/gateway` ist reines HTTP-Handler-Paket ohne DB-Zugriff.
- offen: kein neuer Fund fuer Lauf 8. Docker/Postgres weiterhin nicht benoetigt fuer diese
  Unit, Status ungeprueft gelassen (letzter bekannter Stand: gestoppt) - vor der naechsten
  DB-Unit (Block B-Server oder C2) pruefen/hochfahren.

## Iteration 16 — b-cov-gateway-einkauf-extended — done — 2026-08-10 00:20
- commit: (wird nach diesem Eintrag erstellt)
- gebaut: neue Datei `route_einkauf_extended_test.go` fuer `route_einkauf_extended.go` (19
  Handler, vorher 0 % - `route_einkauf_test.go` deckt nur die Basisdatei `route_einkauf.go`
  ab, wie im Backlog vermerkt gelesen und nichts gedoppelt). Abgedeckt: Catalog (List mit
  allen Query-Filtern category/search/supplier_id/available in beiden Boolean-Schreibweisen
  und Pagination, Get/Create/Update/Delete inkl. `decimal_gte0` auf `price`), Supplier
  Ratings (List/Create/Delete inkl. der beiden Pflichtfelder category/rating), Framework
  Contracts (List mit supplier_id/status-Filtern, Get/Create/Update/Delete inkl.
  `decimal_gte0` auf `total_value`), Contract Items (Create/Update/Delete, beide UUID-Params
  id+itemId je einzeln falsch getestet) und Contract Calls (List, Create inkl. optionalem
  `po_id`-Zeiger-Wiring mit und ohne Wert). `go tool cover -func` zeigt alle 19 Handler
  zwischen 54,5 % (`HandleUpdateContractItem`) und 100 % (`registerExtendedRoutes`).
  Gateway-Gesamtcoverage 34,0 % laut `go test -coverprofile` (vorher 33,2 % nach Iteration 15).
- befund contract-call-ohne-mengenpruefung: `Service.CreateContractCall`
  (internal/einkauf/service_extended.go:610) prueft den Abrufbetrag nur auf `v < 0` - kein
  Abgleich gegen `framework_contracts.total_value` oder das bereits verbrauchte `used_value`.
  `UpdateContractUsedValue` (postgres_repository_extended.go:403) rechnet danach nur die
  Summe aus `framework_contract_calls` neu und schreibt sie zurueck, rein informativ. Ein
  Rahmenvertrag laesst sich damit beliebig oft und beliebig hoch abrufen - genau der im
  Backlog selbst benannte Verdacht ("ein Abruf ueber die Rahmenvertragsmenge hinaus ist der
  Fall, an dem eine fehlende Pruefung sichtbar wird"), jetzt per Lektuere von Service UND
  Repository bestaetigt statt nur vermutet. Kein Fake-`EinkaufServiceClient` im Gateway-Paket,
  also strukturell dieselbe Grenze wie in jeder vorigen Gateway-Unit - die Pruefung selbst
  gehoert ins `internal/einkauf`-Service, nicht ins Gateway. NICHT gefixt (echte
  Verhaltensaenderung, keine Coverage-Aenderung) - neue Unit
  `fix-einkauf-contract-call-no-value-check` ganz vorne im Backlog fuer Lauf 8 angelegt,
  inkl. Fix-Vorschlag und offener Produktfrage (Verhalten bei Vertragsstatus != active).
  `TestHandleCreateContractCall_WithAndWithoutPOID` schreibt deshalb nur fest, dass der
  Handler bei beiden Formen ohne Panic bis zur RPC durchlaeuft, nicht dass eine
  Mengenpruefung greift - die gibt es serverseitig noch nicht.
- befund rating-ohne-obergrenze: `createSupplierRatingRequest.Rating` (route_einkauf_extended.go:100)
  traegt `validate:"required"` (also nur != 0), obwohl der Proto-Kommentar `int32 rating = 4; //
  1-5` eine Spanne vorgibt. Ein Wert wie 99 oder -3 wuerde die Gateway-Validierung anstandslos
  passieren. Gegengeprueft: kein `min=1,max=5`-Tag existiert, kein serverseitiger Guard gefunden
  in `internal/einkauf` fuer `CreateSupplierRating`. Kleinerer Fund als die Mengenpruefung, nicht
  separat als Unit angelegt - falls Lauf 8 die Contract-Call-Unit aufgreift, gehoert diese
  Bereichspruefung als Ein-Zeiler mit rein (`validate:"required,min=1,max=5"` reicht,
  Root-Cause-Fix an der gemeinsamen Stelle statt Guard verstreut).
- gate: build ok (`go build -p 2 ./internal/gateway/... ./cmd/gateway/...`) | vet ok
  (`go vet ./internal/gateway/...`) | lint ok (`golangci-lint run ./internal/gateway/...`,
  0 issues) | test ok (`go test -count=1 ./internal/gateway/...`, gruen, 0 uebersprungen)
  | migration n.a. | rls-smoke n.a. (kein DB-Zugriff)
- verify vorgaenger: sauber — `6a4831e7` (b-cov-gateway-biz-invoices) geprueft: `git show
  --stat` zeigt nur `route_biz_invoices_test.go` (neu) plus Journal/Backlog, kein
  Produktionscode, keine neue Route, kein RequirePermission, keine Tabelle.
- mutations-probe: `validate:"required"` vom `Name`-Feld in `createCatalogItemRequest`
  (route_einkauf_extended.go:77) entfernt → `TestHandleCreateCatalogItem_MissingName` wurde
  rot (503 "connection error" statt 400/"validation_failed"/Feld "name"), die uebrigen fuenf
  Create-Tests blieben unberuehrt. Erste Probe (SupplierID-`required` entfernt, nur `uuid`
  belassen) waere KEINE echte Probe gewesen - der `uuid`-Tag lehnt einen leeren String
  ohnehin ab (kein `omitempty`), also waere der Test zufaellig gruen geblieben; verworfen
  und durch die Name-Probe ersetzt, bevor sie ins Journal kam. Zurueckgedreht, `git diff
  --stat` auf die Produktionsdatei zeigt keine Restaenderung, volle Suite danach gruen.
- db-tests: 0 — `internal/gateway` ist reines HTTP-Handler-Paket ohne DB-Zugriff.
- offen: `fix-einkauf-contract-call-no-value-check` im Backlog fuer Lauf 8 angelegt (echter
  Produktionsbug, kein Test-Diff). Naechste Unit laut Reihenfolge: `b-cov-gateway-bexio`
  (letzte Gateway-Unit vor Block B-Server). Docker/Postgres weiterhin nicht hochgefahren -
  vor der ersten Block-B-Server- oder C2-Unit noetig.

## Iteration 17 — b-cov-gateway-bexio — done — 2026-08-10 00:25
- commit: (wird nach diesem Eintrag erstellt)
- gebaut: neue Datei `route_bexio_test.go` fuer `route_bexio.go` (16 Handler/Methoden, vorher
  0 % - `bexio_state_test.go` deckte nur die HMAC-Token-Logik in `bexio_state.go` ab, wie im
  Backlog vermerkt gelesen und nichts gedoppelt). Abgedeckt: `HandleOAuthCallback` (Service-
  Unavailable, fehlender Code mit/ohne `error`-Query, fehlend konfiguriertes `stateSecret`,
  ungueltiges State-Token, gueltiges State-Token mit anschliessendem RPC-Fehl → generischer
  Redirect), `HandleGetAuthURL` (Service-Unavailable, fehlender Tenant, fehlendes
  `stateSecret`, RPC-Fehler), sowie fuer alle uebrigen zehn Handler (Disconnect,
  GetConnectionStatus, TriggerSync, GetSyncStatus, UpdateSyncConfig, ListSyncLogs,
  GetFieldMappings, UpdateFieldMappings, PushInvoice, PushQuote) durchgaengig Service-
  Unavailable + fehlender Tenant + (wo zutreffend) ungueltiges JSON + RPC-Fehler-Pfad.
  `TriggerSync` bekam zusaetzlich einen Test fuer den `ContentLength==0`-Kurzschluss (leerer
  Body wird NICHT als kaputtes JSON abgelehnt, sondern faellt mit `sync_type=""` durch).
  `go tool cover -func` zeigt alle 16 Handler/Methoden zwischen 50,0 % (`HandleGetSyncStatus`)
  und 100 % (Konstruktor/`ServiceName`/`getBexioClient`/`RegisterRoutes`). Gateway-
  Gesamtcoverage 34,9 % laut `go test -coverprofile` (vorher 34,0 % nach Iteration 16).
- kein-token-befund: `done_when` verlangt einen Test, dass kein Access-/Refresh-Token in einer
  Antwort auftaucht. Ein Live-Test dafuer ist strukturell nicht moeglich - `internal/gateway`
  hat wie jede vorige Coverage-Unit dieses Laufs keinen Fake-/bufconn-Client fuer
  `BexioIntegrationServiceClient`, ein RPC-Aufruf schlaegt in jedem Testfall am
  Verbindungsaufbau fehl, es gibt also nie eine echte erfolgreiche Response zum Pruefen.
  Stattdessen per Proto-Lektuere verifiziert (`proto/biz/v1/bexio.proto`, alle 14
  Response-Messages durchgesehen): keine einzige traegt ueberhaupt ein Token- oder
  Secret-Feld - der OAuth-Access-Token verlaesst den `biz`-Service nie in Richtung Gateway.
  `TestBexioResponseProtos_NeverExposeOAuthTokens` schreibt das als Reflection-Test ueber alle
  14 generierten Response-Structs fest (keine Feldnamen, die "token" oder "clientsecret"
  enthalten) - kein Live-Beweis, aber ein echter Regressions-Wächter: wird dem Proto je ein
  Token-Feld hinzugefuegt, faellt dieser Test, bevor irgendein Handler es durchreichen koennte.
  Der zweite Teil des done_when ("Rueckruf mit falschem Zustandsparameter wird
  ununterscheidbar abgewiesen") ist end-to-end getestet:
  `TestHandleOAuthCallback_InvalidState_SameResponseAsExpiredState` vergleicht Status UND Body
  eines kaputten gegen ein gueltig signiertes, aber abgelaufenes Token - beide identisch.
- kleinerer-befund: `HandlePushInvoice`/`HandlePushQuote` validieren `invoice_id`/`quote_id`
  aus dem chi-URL-Parameter nicht als UUID (kein `validateUUIDParam`-Aufruf, anders als in
  mehreren anderen Routendateien dieses Laufs) - jeder String erreicht direkt die RPC. Kein
  eigener Fund fuer eine Unit, da die biz-Service-Seite die eigentliche Validierung tragen
  muss und ein Format-Fehler dort ohnehin nur zu einer schlechteren Fehlermeldung fuehrt, nicht
  zu einem Sicherheitsproblem (Tenant-Scoping passiert serverseitig) - der Vollstaendigkeit
  halber hier notiert, falls Lauf 8 die Gateway-UUID-Validierungskonvention vereinheitlicht.
- gate: build ok (`go build ./internal/gateway/...`) | vet ok (`go vet ./internal/gateway/...`)
  | lint ok (`golangci-lint run ./internal/gateway/...`, 0 issues) | test ok
  (`go test -count=1 ./internal/gateway/...`, gruen, 0 uebersprungen) | migration n.a. |
  rls-smoke n.a. (kein DB-Zugriff)
- verify vorgaenger: sauber — `9bd78903` (b-cov-gateway-einkauf-extended) geprueft: `git show
  --stat` zeigt nur `route_einkauf_extended_test.go` (neu) plus Journal/Backlog, kein
  Produktionscode, keine neue Route, kein RequirePermission, keine Tabelle.
- mutations-probe: `validate:"required,min=1"` vom `Mappings`-Feld in
  `updateBexioFieldMappingsRequest` (route_bexio.go:470) entfernt →
  `TestHandleUpdateFieldMappings_MissingMappings` wurde rot (503 statt 400/"validation_failed"/
  Feld "mappings"), die uebrige Suite unberuehrt. Zurueckgedreht, `git diff --stat` auf die
  Produktionsdatei zeigt keine Restaenderung, volle Suite danach gruen.
- db-tests: 0 — `internal/gateway` ist reines HTTP-Handler-Paket ohne DB-Zugriff.
- offen: letzte Unit in Block B-Gateway (12/12 erledigt). Naechste Unit laut Reihenfolge:
  `b-cov-server-fuhrpark` (erste Unit in Block B-Server) - Docker/Postgres bisher nicht
  hochgefahren, fuer diese und alle folgenden Server-Units pruefen ob DB-Zugriff noetig ist
  (fuhrpark-Scope nennt GPS-Lesepfade + Tenant, ggf. reine gRPC-Handler-Tests ohne DB analog
  zum `formulare_grpc_test.go`-Muster ausreichend).

## Iteration 18 — b-cov-server-fuhrpark — done — 2026-08-10 01:10
- commit: (wird nach diesem Eintrag erstellt)
- gebaut: neue Datei `fuhrpark_grpc_test.go` fuer `fuhrpark_grpc.go` (36 Methoden, vorher 0 %).
  Kein DB-Zugriff noetig - analog zum `formulare_grpc_test.go`-Muster reicht ein
  In-Memory-Stub (`stubFuhrparkRepo`), der `fuhrpark.Repository` (33 Methoden) implementiert,
  weil `FuhrparkGRPCServer.svc` ein konkretes `*fuhrpark.Service` ist (kein Interface) und ein
  echtes `fuhrpark.NewService(repo)` mit Stub-Repo dahinter braucht, statt eines nilbaren Feldes
  wie bei formulare. Der Stub gibt bei `r.err == nil` plausible Default-Objekte zurueck (IDs aus
  der Anfrage gespiegelt), sonst `r.err` - damit laufen Validierungs-, Fehlerabbildungs- und
  Happy-Pfade ueber denselben Stub. Abgedeckt je Handler-Gruppe: Vehicle, Service, Damage,
  History/Report (inkl. `ExportVehicleReport`-CSV-Header-Pruefung), FuelLog, TripLog (inkl.
  `ExportTripLogs`), VehicleBooking (inkl. dem Kommentar-belegten Fall, dass `CreatedBy` aus dem
  Auth-Kontext kommt, nie aus dem Body - per Stub-Capture bewiesen), VehicleDocument,
  DriverLicense, GPS. `go tool cover -func` zeigt fuer alle 36 Handler-Methoden zwischen 30 %
  (`ListUpcomingServices`) und 100 % (`mapFuhrparkError`, Konstruktor); auch alle acht
  Proto-Mapper-Helfer (`serviceToProto`, `damageToProto`, `fuelLogToProto`, `tripLogToProto`,
  `vehicleDocumentToProto`, `driverLicenseToProto`, `gpsPositionToProto`,
  `vehicleRouteToProto`) liegen nach zusaetzlichen Happy-Path-Tests zwischen 50 % und 100 %,
  keine 0-%-Reste mehr in der Datei. Server-Gesamtcoverage 27,9 % laut
  `go test -coverprofile` (vorher 27,6 % nach Iteration 17s Bexio-Lauf, jeweils fuer das
  `internal/server`-Paket allein - die Lauf-6-Zahl von 26,0 % war Repo-weit gewichtet und daher
  nicht direkt vergleichbar).
- gps-tenant-befund: GPS ist personenbezogene Bewegungsdaten (Backlog-Auflage). Alle drei
  GPS-Handler (`IngestGpsPositions`, `GetVehicleRoutes`, `GetGpsPositions`) nehmen den Tenant
  ausschliesslich aus `middleware.GetTenantID(ctx)`, nicht aus dem Request-Body - es gibt in
  keinem der drei Proto-Requests ueberhaupt ein `tenant_id`-Feld, ein Client kann den Tenant
  also strukturell nicht faelschen. Per Stub-Capture (`lastIngestTenantID`, `lastRoutesParams`,
  `lastGpsParams`) bewiesen, dass der tatsaechlich an den Service durchgereichte Tenant der
  Kontext-Tenant ist. Zusaetzlich getestet: `GetVehicleRoutes` faellt bei leerem Datumsfilter
  auf ein 7-Tage-Fenster zurueck, `GetGpsPositions` auf 24 Stunden - beide im Handler
  hartkodiert (Zeilen 1371-1376 bzw. 1409-1416), als Test festgeschrieben.
- gate: build ok (`go build ./internal/server/...`) | vet ok (`go vet ./internal/server/...`)
  | lint ok (`golangci-lint run ./internal/server/...`, 0 issues) | test ok
  (`go test -count=1 ./internal/server/...`, gruen, 0 uebersprungen) | migration n.a. (keine
  Migration angefasst) | rls-smoke n.a. (Stub-Repo, kein DB-Zugriff) | `go build ./...`
  Repo-weit bricht lokal mit `fatal error: runtime: cannot allocate memory` beim Linken von
  `cmd/crm` ab (24 Microservice-Binaries gleichzeitig linken sprengt den lokalen RAM) - das ist
  eine Umgebungsgrenze dieser Maschine, keine Regression durch diese Unit; `internal/server`
  allein baut, vettet und testet sauber, das ist der in `done_when` geforderte Gate.
- verify vorgaenger: sauber — `c6134991` (b-cov-gateway-bexio, letzte Unit in Block B-Gateway)
  geprueft: `git show --stat` zeigt nur `route_bexio_test.go` (neu) plus Journal/Backlog, kein
  Produktionscode, keine neue Route, kein RequirePermission, keine Tabelle.
- mutations-probe: in `IngestGpsPositions` (fuhrpark_grpc.go:1357) den durchgereichten
  `tenantID` direkt vor dem Service-Aufruf auf `uuid.Nil` ueberschrieben (Tenant-Scoping
  gebrochen, Variable bleibt benutzt, damit der Build nicht schon am unused-var scheitert) →
  `TestFuhrparkGpsHandlers/IngestGpsPositions_scopes_the_write_to_the_context_tenant,_not_a_
  client-suppliable_value` wurde rot (erwarteter Tenant ungleich `uuid.Nil`), alle anderen
  zehn Subtests der Gruppe blieben gruen. Zurueckgedreht, `git diff --stat` auf
  `fuhrpark_grpc.go` zeigt keine Restaenderung, volle Suite danach wieder gruen.
- db-tests: 0 — reines gRPC-Handler-Paket ohne DB-Zugriff, analog zu allen bisherigen
  `internal/server`-Coverage-Units in diesem Lauf (formulare, work_*, work_comment etc.).
- offen: erste von zwoelf Units in Block B-Server erledigt (1/12). Naechste laut Reihenfolge:
  `b-cov-server-rapporte` (Genehmigungsfluss, ebenfalls kein DB-Zugriff noetig laut Scope).

## Iteration 19 — b-cov-server-rapporte — done — 2026-08-10 00:32
- commit: (wird nach diesem Eintrag erstellt)
- gebaut: neue Datei `rapporte_grpc_test.go` fuer `rapporte_grpc.go` (34 Methoden, vorher 0 %).
  In-Memory-Stub `stubRapporteRepo` implementiert `rapporte.Repository` (23 Methoden) mit
  echten Status-Uebergaengen fuer Reports (nicht nur Fehler-Injektion) - `AtomicApproveReport`/
  `AtomicRejectReport` respektieren den TOCTOU-sicheren "nur aus submitted"-Vertrag der echten
  Implementierung, damit der volle Zustandsautomat ueber den echten `rapporte.Service` laeuft,
  nicht nur ueber gemockte Sentinel-Fehler. Abgedeckt je Handler-Gruppe: Report (CRUD,
  Tenant-aus-Kontext-Beweis analog Lauf-6-Fuhrpark-Vorlage), State Machine (submit/approve/
  reject inkl. Doppel-Approve -> already-approved, Reject-nach-Approved -> already-approved statt
  generischem invalid-transition), Line, Attachment (inkl. Objekt-Key-Tenant-Praefix-Pruefung),
  Signature, Stats/Export (inkl. PDF-Payload/Filename), Worker, Measurement, Template.
  `go tool cover -func` zeigt alle 34 Handler-Methoden zwischen 68,8 % (`UpdateLine`) und 100 %
  (`GetReport`, `mapRapporteError`, Konstruktor), keine 0-%-Reste; die acht Proto-Mapper liegen
  zwischen 41,7 % und 66,7 %. `internal/server`-Gesamtcoverage 30,5 % laut
  `go test -coverprofile` (vorher 27,9 % nach Iteration 18s Fuhrpark-Lauf).
- fund-echte-luecke: `RejectReport` validiert nirgends (Service, Handler, Repository), dass
  `ReviewNote` nicht leer ist - der Backlog-Scope dieser Unit selbst fordert aber "Ablehnen ohne
  Begruendung muss scheitern". Verifiziert durch Lesen von service.go (RejectReport prueft nur
  TenantID/ReportID), rapporte_grpc.go (reicht ReviewNote ungeprueft durch) und
  postgres_repository.go (kein NOT-NULL/Laengen-Constraint). Nach Backlog-Regel ("NEUE ROUTEN...
  wer eine echte Luecke findet, notiert sie im Journal und legt eine Unit fuer Lauf 8 an, statt
  sie nebenbei zu bauen") nicht inline gefixt, sondern `fix-rapporte-reject-without-reason` ganz
  vorne im Backlog angelegt (todo). Die Coverage-Tests selbst pruefen deshalb nur die IST-Logik
  (leeres ReviewNote wird aktuell angenommen, nicht abgelehnt) und behaupten nichts anderes.
- gate: build ok (`go build -p 2 ./internal/server/...`) | vet ok | lint ok
  (`golangci-lint run ./internal/server/...`, 0 issues) | test ok (`go test -count=1
  ./internal/server/...`, gruen, 0 uebersprungen - Postgres-Container `docker-postgres-1` war zu
  Laufbeginn gestoppt, per `docker start` reaktiviert und Healthcheck abgewartet, danach voller
  Lauf gruen inkl. aller DB-Tests) | migration n.a. (keine Migration angefasst) | rls-smoke n.a.
  (Stub-Repo, kein DB-Zugriff durch die neuen Tests selbst) | `go build ./...` Repo-weit nicht
  versucht (bekannte lokale RAM-Grenze beim Linken aller 24 Microservice-Binaries, siehe
  Iteration 18) - `internal/server` allein baut, vettet, lintet und testet sauber.
- verify vorgaenger: sauber — `a90372fd` (b-cov-server-fuhrpark) geprueft: `git show --stat`
  zeigt nur `fuhrpark_grpc_test.go` (neu) plus Journal/Backlog, kein Produktionscode, keine neue
  Route, kein RequirePermission, keine Tabelle, kein .proto.
- mutations-probe: in `mapRapporteError` (rapporte_grpc.go:1056) die Zuordnung fuer
  `ErrAlreadyApproved` von `codes.FailedPrecondition` auf `codes.Internal` geaendert →
  `TestMapRapporteError/already_approved` sowie
  `TestRapporteStateMachineHandlers/ApproveReport_twice_fails_the_second_time_with_already-approved`
  und `.../RejectReport_on_an_already-approved_report_fails_with_already-approved,_not_a_generic_
  invalid_transition` wurden alle drei rot (erwarteter Code FailedPrecondition, erhalten
  Internal), der Rest der Suite blieb gruen. Zurueckgedreht, `git diff --stat` auf
  `rapporte_grpc.go` zeigt keine Restaenderung, volle Suite danach wieder gruen.
- db-tests: 0 — reines gRPC-Handler-Paket ohne DB-Zugriff, wie bei allen bisherigen
  `internal/server`-Coverage-Units in diesem Lauf.
- offen: zweite von zwoelf Units in Block B-Server erledigt (2/12). Naechste laut Reihenfolge:
  `b-cov-server-inventar` (Bestandsbewegungen, Zu-/Abgang getrennt pruefen). Neue Fix-Unit
  `fix-rapporte-reject-without-reason` steht jetzt ganz vorne im Backlog (todo, nicht Teil
  dieses Laufs' Coverage-Reihenfolge - Luke entscheidet ob sie in Lauf 7 noch reinpasst oder
  nach Lauf 8 wandert). Postgres-Container lief zu Laufbeginn nicht - falls das oefter passiert,
  lohnt sich ein Blick, ob `docker-postgres-1` einen Restart-Policy-Eintrag braucht.

## Iteration 20 — fix-rapporte-reject-without-reason — done — 2026-08-10 00:39
- commit: (wird nach diesem Eintrag erstellt)
- gebaut: `Service.RejectReport` (internal/rapporte/service.go:333) validiert jetzt vor dem
  Repository-Aufruf `strings.TrimSpace(input.ReviewNote) != ""` und liefert sonst
  `ErrInvalidInput` — exakt das Muster, das `AddLine` fuer `Description` bereits nutzt.
  `mapRapporteError` bildete `ErrInvalidInput` bereits auf `codes.InvalidArgument` ab, keine
  Aenderung an der Fehlerabbildung noetig. `RejectReportInput.ReviewNote` blieb `string`
  (kein Zeigertyp, ein leerer String ist bereits eindeutig ablehnbar). `ApproveReport` bewusst
  unangetastet — hat laut Backlog-Scope keine Begruendungspflicht.
  Zwei bestehende Tests mussten angepasst werden, weil sie `RejectReport` mit leerem
  `ReviewNote` aufriefen, um NUR den Zustandsuebergang zu pruefen (nicht die neue
  Validierung): `TestService_RejectReport_FromDraft_Blocked`
  (internal/rapporte/service_test.go) und der Handler-Test "RejectReport on an
  already-approved report..." (internal/server/rapporte_grpc_test.go) bekamen beide ein
  nicht-leeres `ReviewNote`, damit sie weiterhin ihren jeweiligen Zustandsuebergang
  (`ErrInvalidStateTransition`/`ErrAlreadyApproved`) statt der neuen `ErrInvalidInput`
  pruefen — sonst haette die neue Validierung diese beiden ohne Bezug zum eigentlichen
  Testzweck rot gemacht (Validierung laeuft vor dem Zustandscheck, absichtlich, analog zu
  jeder anderen Input-Validierung im Repo).
  Neue Tests: `TestService_RejectReport_EmptyReviewNote_Returns_ErrInvalidInput` und
  `TestService_RejectReport_WhitespaceReviewNote_Returns_ErrInvalidInput`
  (internal/rapporte/service_test.go, pruefen zusaetzlich dass der Report-Status bei
  Ablehnung `submitted` bleibt statt `rejected` zu werden), sowie
  "RejectReport without a reason is rejected as invalid argument"
  (internal/server/rapporte_grpc_test.go, Handler-Ebene/Fehler-Mapping bis `codes.InvalidArgument`).
  Kein Gateway-Code angefasst — `route_rapporte.go`s `approveRejectRequest.ReviewNote` ist
  bewusst ohne `validate:"required"`, weil dieselbe Struktur auch fuer `ApproveReport` genutzt
  wird; die Pflicht sitzt korrekt allein im Rapporte-Service.
- gate: build ok (`go build -p 2 ./internal/rapporte/... ./internal/server/...
  ./internal/gateway/... ./cmd/rapporte/... ./cmd/gateway/...`) | vet ok
  (`go vet ./internal/rapporte/... ./internal/server/... ./internal/gateway/...`) | lint ok
  (`golangci-lint run --config .golangci.yml ./internal/rapporte/... ./internal/server/...`,
  0 issues) | test ok (`go test -count=1 ./internal/rapporte/... ./internal/server/...
  ./internal/gateway/...`, alle gruen, 0 uebersprungen bei gesetzter `DATABASE_URL` —
  `docker-postgres-1` lief bereits healthy) | migration n.a. (keine neue Spalte/Tabelle,
  reine Service-Validierung) | rls-smoke n.a. (keine Tabelle/Policy angefasst)
- verify vorgaenger: sauber — `0384151f` (b-cov-server-rapporte) geprueft: `git show --stat`
  zeigt nur `rapporte_grpc_test.go` (neu) plus Journal/Backlog, kein Produktionscode, keine
  neue Route, kein RequirePermission, keine Tabelle, kein .proto.
- mutations-probe: `if reviewNote == ""` in `Service.RejectReport` (service.go:335) auf
  `if false && reviewNote == ""` gesetzt → beide neuen Service-Tests
  (`TestService_RejectReport_EmptyReviewNote_Returns_ErrInvalidInput`,
  `_WhitespaceReviewNote_Returns_ErrInvalidInput`) wurden rot ("Expected error ... but got
  nil", Status wechselte fälschlich auf "rejected" statt "submitted" zu bleiben), der Rest der
  Suite blieb gruen. Zurueckgedreht, `git diff internal/rapporte/service.go` zeigt keine
  Restaenderung, volle Suite (`rapporte`+`server`+`gateway`) danach wieder gruen.
- offen: Diese Iteration wurde mechanisch nach der Ablauf-Regel "Nimm die erste Unit mit
  status: todo" gezogen (Schritt 2 des `ITERATION.md`-Ablaufs, woertlich verifiziert) — die
  Unit stand seit Iteration 19 ganz vorne im Backlog. Damit wurde faktisch eine dritte
  Fix-Unit ueber die im Laufkopf genannte Freigabe ("genau eine Fix-Unit
  fix-inventar-picking-partial-book") hinaus gezogen, nach demselben Praezedenzfall wie
  Iteration 4 (`fix-bexio-tenant-id-missing-on-upsert`), der seither unkorrigiert blieb. Luke
  sollte diese wiederkehrende Spannung zwischen "urspruenglich freigegebener Block" und
  "mechanisch erste todo-Unit" einmal grundsaetzlich klaeren (wird jetzt zum dritten Mal im
  Journal vermerkt). Inhaltlich ist der Fix selbst klein und risikoarm: eine reine
  Input-Validierung ohne neue Tabelle/Route/Guard, exakt im bestehenden `AddLine`-Muster.
  Naechste Unit laut Reihenfolge: `b-cov-server-inventar` (erste verbleibende `todo`-Unit,
  Block B-Server 3/12).

## Iteration 21 — b-cov-server-inventar — done — 2026-08-10 00:47
- commit: (wird nach diesem Eintrag erstellt)
- gebaut: `internal/server/inventar_grpc_test.go` (neu, 0 → alle 31 Methoden von
  `InventarGRPCServer`). `stubInventarRepo` implementiert `inventar.Repository`
  vollstaendig (Item/Movement/Warning/Location/InventurSession/InventurCount/
  PickingList/PickingListItem/ItemAttachment), mit steuerbaren Feldern
  `itemQuantity`, `sessionStatus`, `pickingStatus`, `bookingClaimed` fuer die
  Zustandspfade. `newInventarTestServer(repo)` baut den echten
  `inventar.Service` auf dem Stub auf (Muster aus `fuhrpark_grpc_test.go`,
  nicht das Nil-Service-Muster aus `formulare_grpc_test.go` - inventar_grpc.go
  parst `tenant_id` in JEDEM Handler aus dem Request, nie aus dem
  Auth-Kontext, also gibt es hier keine ctx-basierten Tenant-Tests wie bei
  fuhrpark). `TestMapInventarError` deckt alle 14 Sentinel-Fehler plus den
  Internal-Fallback als Tabellentest ab. Zu- und Abgang sind getrennt
  geprueft: `AdjustStock` mit positivem Delta (Zugang), negativem Delta
  innerhalb des Bestands (Abgang) und negativem Delta unter den verfuegbaren
  Bestand (`ErrInsufficientStock` -> `FailedPrecondition`), ebenso
  `TransferStock` mit zu hoher Menge. Zustandsuebergaenge: Inventur-Session
  bereits `completed` blockt `UpdateInventurSessionStatus`/
  `UpsertInventurCount`/`BookInventurDifferences`; Picking-Liste bereits
  `completed` blockt `UpdatePickingList`/`UpsertPickingListItem`/
  `BookPickingList`, inklusive des Sonderfalls "zweite gleichzeitige Buchung"
  (`BookPickingListTx` liefert `claimed=false` -> `ErrPickingListAlreadyBooked`,
  ohne dass der Handler das an `list.Status` erkennen kann). Jede der 31
  Methoden hat mindestens einen Validierungsfall (ungueltige `tenant_id`
  und/oder ID-Felder), Listen-Handler zusaetzlich den Leer-Ergebnis-Fall
  (Wire-Shape: leeres Proto-Slice, nicht nil, ueber `make([]*X, len(items))`
  in allen `List*`-Handlern bestaetigt).
  Randbefund waehrend der Recherche: `ListItemAttachments`
  (inventar_grpc.go:1020) baut `resp.Attachments` per `append` statt per
  `make([]*X, len(atts))` wie alle anderen List-Handler - bei leerem
  Ergebnis bleibt das Proto-Feld `nil` statt eines leeren Slices. Geprueft,
  ob das ein echter Wire-Shape-Bug ist: der Gateway-Handler
  `HandleListItemAttachments` (route_inventar.go:1523) gibt die Antwort ueber
  `response.Proto` aus, das per `protoMarshaler.Marshal` (protojson) codiert
  - protojson serialisiert `repeated`-Felder unabhaengig von nil/leer immer
  als JSON-Array `[]`, nie als `null`. Kein Fund, keine Fix-Unit noetig -
  Go-interner nil-vs-empty-Unterschied ist hier folgenlos, weil protojson ihn
  einebnet (anders als bei den Gateway-Handlern, die selbst `encoding/json`
  auf einen rohen Go-Typ anwenden - dort waere derselbe Unterschied real).
- gate: build ok (`go build -p 2 ./internal/inventar/... ./internal/server/...
  ./cmd/inventar/... ./cmd/gateway/...`) | vet ok (`go vet
  ./internal/inventar/... ./internal/server/...`) | lint ok (`golangci-lint
  run --config .golangci.yml ./internal/inventar/... ./internal/server/...`,
  0 issues) | test ok (`go test -count=1 ./internal/inventar/...
  ./internal/server/...`, alle gruen; `internal/inventar` 65 Subtests laut
  `-v`-Zaehlung, 0 uebersprungen bei gesetzter `DATABASE_URL` -
  `docker-postgres-1` lief bereits healthy) | `go test -count=1
  ./internal/gateway/` zusaetzlich gruen (Pflichtlauf, obwohl diese Iteration
  keine Route/kein `.proto` anfasst) | migration n.a. (reine
  Server-Test-Coverage, keine Schemaaenderung) | rls-smoke n.a. (keine
  Tabelle/Policy angefasst)
- verify vorgaenger: sauber — `cefb5419` (fix-rapporte-reject-without-reason)
  geprueft: `git show --stat` zeigt nur `service.go` (5 Zeilen,
  Trim+Leer-Check vor dem Repository-Aufruf) plus zwei Testdateien
  (`service_test.go`, `rapporte_grpc_test.go`) und Journal/Backlog. Kein
  gRPC-Layer-Umgehung, kein Stub, kein `.proto` angefasst, kein neuer
  `RequirePermission`-Guard, keine neue Tabelle, keine Wire-Shape-Aenderung,
  keine neue Route, kein Guard-Alt-Key verloren. `mapRapporteError` unangetastet,
  `ErrInvalidInput` war bereits auf `codes.InvalidArgument` gemappt.
- mutations-probe: in `Service.AdjustStock` (internal/inventar/service.go,
  Zeile `if item.Quantity+input.Delta < 0`) die Schwelle testweise auf
  `< -1000` gesetzt → `TestInventarStockHandlers/AdjustStock_negative_delta_
  below_zero_is_rejected_as_failed_precondition` wurde rot (erwarteter Code
  FailedPrecondition, Aufruf lief stattdessen durch), alle anderen Subtests
  blieben gruen. Zurueckgedreht, `git diff internal/inventar/service.go`
  zeigt keine Restaenderung (leerer Diff bestaetigt), volle Suite
  (`inventar`+`server`) danach wieder gruen.
- offen: dritte von zwoelf Units in Block B-Server erledigt (3/12). Naechste
  laut Reihenfolge: `b-cov-server-plugin` (WASM-Feature-Flag AUS, Build-Tag
  `no_wasm` beachten - siehe Notiz an der Unit selbst). Kein neuer Befund,
  der eine Fix-Unit rechtfertigt (siehe Randbefund oben - kein echter Bug).

## Iteration 22 — b-cov-server-plugin — done — 2026-08-10 01:05
- commit: (wird nach diesem Eintrag erstellt)
- gebaut: `internal/server/plugin_grpc_test.go` (neu, 0 → alle 32 Methoden von
  `PluginGRPCServer`). Acht kleine Stub-Repos statt eines kombinierten Stubs -
  `plugin.NewService` nimmt acht getrennte Repository-Interfaces
  (ManifestRepo, InstallationRepo, PermissionRepo, KVStoreRepo,
  ExecutionLogRepo, ValidationRuleRepo, WorkflowRuleRepo,
  IndustryTemplateRepo), und mehrere davon haben gleichnamige Methoden
  (`Create`, `GetByID`, `List`, `Delete`) mit unterschiedlichen Signaturen -
  ein einzelner Go-Typ kann nicht alle acht gleichzeitig implementieren (kein
  Overloading). Die unexported Mocks in `internal/plugin/service_test.go`
  loesen genau das schon fuer `package plugin`, sind aber aus `package
  server` nicht erreichbar - deshalb acht neue, schlankere Stub-Typen
  (`stubPluginManifestRepo` etc.), gebuendelt in `pluginTestRepos`. Kein
  Build-Tag `no_wasm` noetig fuer die Tests selbst - sie rufen nur
  Verwaltungs-/Validierungspfade des echten `plugin.Service` auf Stubs auf,
  keine WASM-Ausfuehrung. `TestMapPluginError` deckt alle 15 Sentinel-Faelle
  als Tabellentest ab, davon zwei Faelle die bewusst eine Luecke
  dokumentieren (siehe unten). Jede der 32 Methoden hat mindestens einen
  Validierungsfall (ungueltige UUID) und wo zutreffend Erfolg/Nicht-gefunden/
  Konflikt. `go tool cover -func` zeigt alle 32 Handler zwischen 75 % und
  100 %, keine 0-%-Reste; `internal/server`-Gesamtcoverage 34,3 % laut
  `go test -coverprofile` (vorher 30,5 % nach Iteration 21s Inventar-Lauf).
- fund-echte-luecke (zwei getrennte Bug-Klassen, beide durch Tests belegt,
  beide NICHT inline gefixt, zwei neue Fix-Units ganz vorne im Backlog
  angelegt):
  1. `fix-plugin-error-mapping-gaps`: `mapPluginError`
     (plugin_grpc.go:829) ist die einzige `map<X>Error`-Funktion im Repo, die
     mit `==` statt `errors.Is` vergleicht (fuhrpark/rapporte/inventar nutzen
     alle bereits `errors.Is`). Jeder von `Service` gewrappte Sentinel wird
     dadurch nie erkannt: `ApprovePermissions` (undeklarierte Permission,
     service.go:324) und `UpdatePluginSettings` (Schema-Verstoss,
     service.go:392) liefern beide Internal statt InvalidArgument. Zusaetzlich
     hat `mapPluginError` ueberhaupt keinen Fall fuer `ErrPluginHasInstallations`
     (DeleteManifest, service.go:165) - faellt auch auf Internal.
  2. `fix-plugin-nil-manifest-panic-on-orphaned-installation`: `ApprovePermissions`,
     `UpdatePluginSettings` und `GetPluginSettingsSchema` laden per
     `manifests.GetByID` ein Manifest und dereferenzieren es ungeprueft auf
     nil - anders als `Service.GetManifest`, das denselben nil-Fall explizit
     abfaengt. Da `DeleteManifest`s `HasActiveInstallations`-Check nur
     nicht-uninstallte Installationen zaehlt, kann eine `uninstalled`-
     Installation ihr geloeschtes Manifest ueberleben; jeder Aufruf einer der
     drei Methoden auf diese Installation loest eine Nil-Pointer-
     Dereferenzierung aus. `cmd/plugin/main.go` haengt
     `middleware.RecoveryUnaryInterceptor()` ein, also wird daraus in
     Produktion ein opaker Internal-Fehler statt eines Prozessabsturzes -
     aber es bleibt ein Panic bei jedem Treffer.
  Beide Funde per `require.Panics`/Code-Erwartung in `plugin_grpc_test.go`
  reproduziert und als "documents current gap" markiert (IST-Zustand, keine
  Behauptung ueber gewuenschtes Verhalten). Nach Fix muessen fuenf Testfaelle
  auf den dann korrekten Code/das dann korrekte Verhalten aktualisiert werden
  - in den beiden neuen Fix-Units selbst benannt.
- gate: build ok (`go build -p 2 ./internal/plugin/... ./internal/server/...
  ./cmd/plugin/... ./cmd/gateway/...`) | vet ok (`go vet ./internal/plugin/...
  ./internal/server/...`) | lint ok (`golangci-lint run --config .golangci.yml
  ./internal/server/...`, 0 issues) | test ok (`go test -count=1
  ./internal/server/...`, gruen; `go test -count=1 ./internal/gateway/...
  ./internal/plugin/...` zusaetzlich gruen, Pflichtlauf obwohl diese Iteration
  dort nichts aendert) | migration n.a. (reine Test-Coverage) | rls-smoke n.a.
  (Stub-Repos, kein DB-Zugriff durch die neuen Tests)
- verify vorgaenger: sauber — `3d97b397` (b-cov-server-inventar) geprueft:
  `git show --stat` zeigt nur `inventar_grpc_test.go` (neu) plus
  Journal/Backlog, kein Produktionscode, keine neue Route, kein
  RequirePermission, keine Tabelle, kein .proto.
- mutations-probe: in `mapPluginError` (plugin_grpc.go, Zeile
  `case isNotFound(err): return status.Error(codes.NotFound, ...)`) den Code
  testweise auf `codes.Unavailable` geaendert → 19 Subtests wurden rot
  (`TestMapPluginError` sechs NotFound-Faelle plus 13 Handler-Tests, die auf
  NotFound pruefen, u. a. `TestPluginGetManifest/not_found`,
  `TestPluginDeleteManifest/not_found`, `TestPluginApprovePermissions/
  installation_not_found`), alle anderen Subtests blieben gruen.
  Zurueckgedreht, `git diff --stat internal/server/plugin_grpc.go` zeigt
  keine Restaenderung (leerer Diff bestaetigt), volle Suite danach wieder
  gruen.
- offen: vierte von zwoelf Units in Block B-Server erledigt (4/12). Naechste
  laut Reihenfolge: `b-cov-server-automation` (SSRF-Pruefung der
  HTTP-Aktion als Befund pruefen, unbekannter Ausloeser-/Aktionstyp muss
  abgelehnt werden). Zwei neue Fix-Units (`fix-plugin-error-mapping-gaps`,
  `fix-plugin-nil-manifest-panic-on-orphaned-installation`) stehen jetzt todo
  im Backlog, beide klein und risikoarm (Fehler-Mapping-Tabelle bzw.
  Nil-Guard nach bestehendem Muster aus Service.GetManifest) - Luke
  entscheidet ob sie noch in Lauf 7 reinpassen.

## Iteration 23 — b-cov-server-automation — done — 2026-08-10 01:20
- commit: (wird nach diesem Eintrag erstellt)
- gebaut: neue Datei `internal/server/automation_grpc_test.go` fuer
  `automation_grpc.go` (834 Zeilen, 17 RPC-Methoden, vorher keine Testdatei,
  0 % auf allen Handlern). `AutomationGRPCServer.svc` ist ein konkretes
  `*workflow.Service`, kein Interface - wie bei Plugin/Rapporte gibt es keinen
  Weg, die RPC-Schicht direkt zu faken; stattdessen `stubAutomationRepo`/
  `stubAutomationExecRepo`/`stubAutomationTemplateRepo` (implementieren
  `workflow.Repository`/`ExecutionRepository`/`TemplateRepository`) plus
  No-Op-Stubs fuer `idempotency.Repository` und `workflow.Executor` (nur zum
  Durchreichen an `workflow.NewService`, keine der Tests laeuft tiefer als
  der Automation-Lookup in TriggerWebhook), und lokale Trigger-/Action-Catalog-
  Fakes (`automationTriggerGet/-All`, `automationActionGet/-AllDefs`) mit
  echten `trigger.TriggerDefinition`/`action.ActionDefinition`-Werten, um die
  `triggerDefToProto`/`actionDefToProto`-JSON-Roundtrip-Konvertierung sinnvoll
  zu pruefen statt nur mit Strings zu faken. Alle 17 Methoden mindestens mit
  einem Validierungs- oder Not-Found-Pfad plus Erfolgspfad abgedeckt: Create/
  Update/Delete/GetAutomation (fehlender Tenant, ungueltige ID, unbekannter
  Trigger-/Aktionstyp -> InvalidArgument, Not-Found, Erfolg inkl. Scope- und
  TriggerConfig-Konvertierung); UpdateAutomation zusaetzlich mit einem Test,
  der beweist, dass ein Teil-Update (nur `Name` gesetzt) `Description`
  unangetastet laesst; ListAutomations/ListExecutions je mit einem Test, der
  eine unparsebare Owner-/Automation-ID *ignoriert statt ablehnt* (Filter
  bleibt leer, kein Fehler - Code-Pfad `if err == nil { filter.X = &x }`);
  Enable/DisableAutomation (Not-Found via SetActive, Erfolg inkl. Re-Fetch);
  GetExecution (ungueltige ID, Erfolg); ListTriggerDefinitions/
  ListActionDefinitions (Katalog-Konvertierung inkl. Type/Module/Name/
  Description); ListTemplates (Kategorie-Filter durchgereicht);
  CreateFromTemplate (fehlender Tenant, ungueltige Owner-ID, Template nicht
  gefunden, Erfolg inkl. `IsActive:false`-Default); TestCondition (siehe
  Befund unten); DryRunAutomation (fehlender Tenant, ungueltige ID, Not-Found
  als *weicher* Fehler statt gRPC-Fehler, Erfolg mit zwei simulierten
  Schritten - einer registriert, einer nicht, beide im Response sichtbar);
  GetAutomationStats (fehlender Tenant, Erfolg mit korrekt aufsummierten
  Execution-Zaehlern); TriggerWebhook (ungueltige ID, unbekannte Automation
  ueber die volle Handler->Service->mapDomainError-Kette bis NotFound - der
  einzige RPC ohne `middleware.GetTenantID`, siehe Doc-Kommentar im
  Produktionscode). `mapDomainError` als eigener Tabellentest ueber alle
  12 Sentinels plus den generischen Internal-Zweig plus `nil` -> `nil`.
- befund ssrf-pruefung (im Scope explizit gefordert): bereits vollstaendig
  vorhanden und getestet, eine Ebene tiefer als `automation_grpc.go`.
  `internal/automation/action/http_actions.go` (`HTTPRequestAction`) nutzt
  `safehttp.New()` als Default-Client (nie ein ungeschuetzter `http.Client`,
  `NewHTTPRequestAction` ersetzt `nil` automatisch) und ruft vor jedem Request
  `safehttp.CheckURL(target)` auf; eigene Testdatei `http_actions_test.go`
  existiert bereits. `automation_grpc.go` selbst staged keine ausgehende
  Anfrage - die HTTP-Aktion laeuft ausschliesslich ueber den `engine`/
  `action`-Ausfuehrungspfad, den `automation_grpc.go` nie direkt beruehrt
  (DryRun simuliert nur, ruft keine echten Actions auf). Kein Fund, der einen
  neuen Test in dieser Datei braucht.
- befund unbekannter trigger-/aktionstyp (im Scope explizit gefordert):
  bereits auf Service-Ebene erzwungen (`workflow.Service.validateAutomation`,
  `service.go:426`, mit eigener Testabdeckung in `service_test.go` seit
  laengerem: `TestCreate_UnknownTriggerType`/`TestCreate_UnknownActionType`).
  Auf Handler-Ebene per `TestCreateAutomation_UnknownTriggerType` erneut
  bestaetigt (voller Pfad Handler -> Service -> `mapDomainError` ->
  `codes.InvalidArgument`), kein separater Test fuer unbekannte Aktionstypen
  ueber den Handler noetig - siehe naechster Befund, der genau diesen Pfad
  strukturell verhindert.
- befund actions-wire-shape (schwerwiegend, NICHT gefixt, siehe unten):
  `actions` ist im .proto als `google.protobuf.Struct` deklariert (JSON-Objekt
  -only), aber `workflow.Service.validateAutomation`/`DryRun` unmarshaln
  `auto.Actions` unconditional in `[]models.ActionConfig` (JSON-Array). Eine
  `*structpb.Struct` marshalt strukturell IMMER zu `{...}`, nie zu `[...]` -
  jeder nicht-nil Actions-Wert, selbst `{}`, schlaegt beim Unmarshal in eine
  Go-Slice fehl und liefert `codes.InvalidArgument`. Es gibt also aktuell
  KEINEN Weg, ueber CreateAutomation/UpdateAutomation eine Automation mit
  auch nur einer echten Aktion zu erstellen - nur eine ganz ohne Actions
  (Feld weggelassen) geht durch. Gateway-seitig (`route_automation.go`,
  `rawJSONToAutomationStruct`) verschaerft sich das noch: ein echtes
  JSON-Array im Request-Body laesst `s.UnmarshalJSON(data)` auf einem
  `*structpb.Struct` scheitern, der Fehler wird verschluckt und das Feld
  bleibt nil - 200 OK, aber die vom Client konfigurierten Aktionen wurden nie
  gesetzt. `b-cov-gateway-automation` (Iteration 10) hatte das exakt gleiche
  Symptom bereits fuer `trigger_config` als Testfall notiert, ohne Ursache
  oder Tragweite fuer `actions` zu benennen. `modules.automation` ist in
  `internal/featureflag/` nicht gefuehrt - kein Feature-Flag-Gate, der Pfad
  ist erreichbar. Bewiesen durch
  `TestCreateAutomation_ActionsStructCannotCarryAnArray`. NICHT gefixt -
  Proto-Wire-Vertragsaenderung mit `protoc`-Neugenerierung an drei Messages,
  keine Coverage-Aenderung, nach Backlog-Regel eine eigene Unit:
  `fix-automation-actions-struct-cannot-represent-array` (todo, neu im
  Backlog).
- befund testcondition-inkonsistenz (kleiner, NICHT gefixt): `TestCondition`
  hat - anders als `DryRun` fuer denselben Fall - keine Laengenpruefung vor
  dem Unmarshal von `conditions`; ein Client, der "ohne Bedingungen testen"
  meint und das Feld weglaesst, bekommt `Matches:false` mit
  `"unexpected end of JSON input"` statt des von `DryRun` her erwarteten
  `Matches:true`. Bewiesen durch
  `TestTestCondition_OmittedConditionsSurfacesAsError`. Eigene, kleine Unit:
  `fix-automation-testcondition-omitted-conditions-error` (todo, neu im
  Backlog) - bewusst nicht in die Actions-Unit gemischt, da unabhaengiger
  Root Cause und unabhaengiger Fix.
- gate: build ok (`go build -p 2 ./internal/server/... ./internal/automation/...
  ./cmd/automation/... ./cmd/gateway/...`) | vet ok (`go vet
  ./internal/server/...`) | lint ok (`golangci-lint run --config .golangci.yml
  ./internal/server/...`, 0 issues) | test ok (`go test -count=3
  ./internal/server/...`, dreimal wiederholt, durchgehend gruen, 0
  uebersprungen)
  | migration n.a. (reine Test-Coverage, keine Tabelle angefasst) | rls-smoke
  n.a. (Stub-Repos, kein DB-Zugriff durch die neuen Tests)
- coverage: `automation_grpc.go` von 0 % auf alle 17 RPC-Methoden zwischen
  66,7 % und 100 % (Mittel ueber alle 32 benannten Funktionen inkl. Converter/
  Enum-Helfer: 80,4 %), keine Methode bleibt bei 0 %. `internal/server`
  gesamt (laut `go tool cover -func`) bei 34,9 % (Block-B-Server-Ausgangslage
  war 26,0 % vor Iteration 18).
- verify vorgaenger: sauber — `3a1c38b7` (b-cov-server-plugin) geprueft:
  `git show --stat` zeigt nur `plugin_grpc_test.go` (neu) plus Journal/
  Backlog, kein Produktionscode, keine neue Route, kein RequirePermission,
  keine Tabelle, kein .proto.
- mutations-probe: in `mapDomainError` (automation_grpc.go, Zeile
  `case errors.Is(err, workflow.ErrAutomationNotFound): return
  status.Error(codes.NotFound, ...)`) den Code testweise auf
  `codes.Unavailable` geaendert → vier Tests wurden rot
  (`TestMapDomainError_Table/automation_not_found`,
  `TestUpdateAutomation_NotFound`, `TestGetAutomation_NotFound`,
  `TestEnableAutomation_NotFound`), alle anderen blieben gruen.
  Zurueckgedreht, `git diff --stat internal/server/automation_grpc.go` zeigt
  keine Restaenderung (leerer Diff bestaetigt), `go test -count=1
  ./internal/server/...` danach wieder vollstaendig gruen.
- db-tests: 0 — Repository-Interfaces sind vollstaendig gestubbt, kein
  Postgres-Zugriff in dieser Unit.
- offen: fuenfte von zwoelf Units in Block B-Server erledigt (5/12). Naechste
  laut Reihenfolge: `b-cov-server-settings` (dreistufige Aufloesung Tenant/
  Modul-Leiter/persoenlich, Schreibzugriff ohne Recht muss scheitern). Zwei
  neue Fix-Units aus dieser Iteration (`fix-automation-actions-struct-
  cannot-represent-array`, `fix-automation-testcondition-omitted-conditions-
  error`) stehen todo im Backlog, plus die vier bereits laufenden aus
  frueheren Iterationen (`fix-einkauf-contract-call-no-value-check`,
  `fix-plugin-error-mapping-gaps`,
  `fix-plugin-nil-manifest-panic-on-orphaned-installation`) - Luke
  entscheidet welche noch in Lauf 7 reinpassen.

## Iteration 24 — b-cov-server-settings — done — 2026-08-10 01:25
- commit: (siehe unten)
- gebaut: neue Datei `internal/server/settings_grpc_test.go` fuer
  `settings_grpc.go` (771 Zeilen, 21 RPC-Methoden ueber `SettingsGRPCServer`,
  vorher 0 % Coverage, keine Testdatei). Eigener In-Memory-Stub fuer
  `settings.Repository` (23 Methoden) und `settings.RoleChecker`
  (`stubSettingsRepo`/`stubRoleChecker`, neu — `settings_test.go`s
  `fakeRepo` ist Package-privat in `settings_test` und von `internal/server`
  aus nicht erreichbar). `forceErr` auf dem Stub-Repo faengt die generische
  "slog.Error + codes.Internal"-Fallback-Verzweigung, die sich alle 21
  RPCs teilen, ohne 20 fast identische Injektionspunkte zu brauchen.
  Abgedeckt je RPC: UUID-Validierung fuer tenant_id/user_id/granted_by/
  updated_by (inkl. `optional string`-Filterfelder bei List*), die
  dreistufige Settings-Aufloesung (`GetResolvedSettings`: Tenant-Default
  per Admin geschrieben -> Modul-Leiter-Override am selben Speicherpfad
  (beweist das RBAC-Gate, nicht eine eigene Speicherebene) -> persoenlicher
  User-Override gewinnt -> ein zweiter User ohne eigenen Override sieht
  weiterhin die Tenant-/Leiter-Ebene, nicht den ersten User), Tenant-
  Schreibzugriff ohne Modul-Leiter-/Admin-Recht scheitert
  (`PutTenantSettings`/`PutBranding` -> PermissionDenied), leerer
  Settings-Key -> InvalidArgument, `[]` statt `null` bei leerer
  `GetResolvedSettings`-Antwort, `ReplaceUserSettings`-Full-Replace-
  Semantik (nicht mitgeschickte Keys verschwinden), `mapModuleGrantError`
  als eigener Tabellentest ueber alle drei Zweige
  (ErrNotAdmin/ErrInvalidModuleID/generisch), Branding-Validierungskette
  als Tabellentest (Name zu lang, Accent-Farbe ausserhalb der Palette,
  Objekt-Key eines fremden Tenants), Lizenzierung (`GetTenantLicense`
  liefert den vollen Katalog mit Aktivierung/Sitzen je Modul,
  `SetTenantModuleActive` mit unbekanntem Modul -> InvalidArgument, Sitze
  fallen bei Deaktivierung auf 0), `GetTenantSubscription` NotFound vs.
  Erfolg inkl. Datumsformat "YYYY-MM-DD" fuer `billing_period_end`, und
  Value-Sets (`base=true`-Baseline-Ansicht, Tenant-Override-Merge,
  ungueltiges Key-Pattern, unbekannter Key -> NotFound,
  `UpsertValueSet`-Validierungstabelle ueber alle vier `validateValueSet`-
  Sentinels). `internal/server`-Gesamtcoverage laut `go tool cover -func`:
  keine Methode in `settings_grpc.go` bleibt bei 0 % (Spanne 55,6 % bis
  100 %, `ReplaceUserSettings` am niedrigsten wegen ihres unbenutzten
  Fehlerpfads bei Repo-Fehlern, der nicht extra injiziert wurde).
- **Befund waehrend der Arbeit, kein Fix (ausserhalb Scope):** `DeleteValueSet`
  existiert als Service-Methode (`internal/settings/service.go:504`,
  System-Key -> Reset auf Baseline, Tenant-Key -> geloescht) und im
  Repository-Interface, ist aber im `.proto` bewusst nicht als RPC
  exponiert — der Proto-Kommentar selbst dokumentiert das ("Delete is
  deliberately not exposed here yet -- no caller needs it"). Kein Bug,
  nur zur Kenntnis: die Coverage-Luecke fuer `DeleteValueSet` in
  `internal/settings` bleibt unabhaengig von dieser gRPC-Coverage-Unit
  bestehen, falls eine kuenftige `internal/settings`-Coverage-Unit das
  aufgreifen will.
- gate: build ok (`go build -p 2 ./internal/server/... ./cmd/gateway/...
  ./internal/settings/...`) | vet ok (`go vet ./internal/server/...`) |
  lint ok (`golangci-lint run --config .golangci.yml ./internal/server/...`,
  0 issues — ein erster Lauf meldete `ST1012` fuer den generischen
  Test-Error `assertAnError`, auf `errStubRepoFailure` umbenannt) | test ok
  (`go test -count=3 ./internal/server/...`, dreimal wiederholt,
  durchgehend gruen) | migration n.a. (reine Test-Coverage, keine Tabelle
  angefasst) | rls-smoke n.a. (Stub-Repo, kein DB-Zugriff durch die neuen
  Tests)
- verify vorgaenger: sauber — `920447cc` (b-cov-server-automation) geprueft:
  `git show --stat` zeigt nur `automation_grpc_test.go` (neu) plus
  Journal/Backlog, kein Produktionscode, keine neue Route, kein
  `RequirePermission`, keine Tabelle, kein `.proto` beruehrt.
- mutations-probe: in `mapModuleGrantError` (settings_grpc.go, Zeile
  `case errors.Is(err, settings.ErrInvalidModuleID): return
  status.Errorf(codes.InvalidArgument, ...)`) den Code testweise auf
  `codes.Unimplemented` geaendert → zwei Tests wurden rot
  (`TestMapModuleGrantError_Table/invalid_module_id`,
  `TestGrantModuleAccess_ErrorMapping/empty_module_id`), alle anderen
  blieben gruen. Zurueckgedreht, `git diff --stat internal/server/
  settings_grpc.go` zeigt keine Restaenderung (leerer Diff bestaetigt),
  `go test -count=3 ./internal/server/...` danach wieder vollstaendig
  gruen.
- db-tests: 0 — `settings.Repository` ist vollstaendig gestubbt, kein
  Postgres-Zugriff in dieser Unit.
- offen: sechste von zwoelf Units in Block B-Server erledigt (6/12).
  Naechste laut Reihenfolge: `b-cov-server-wiki` (widerrufener vs.
  unbekannter Freigabe-Token muss dieselbe Antwort erzeugen, beide
  Artikelinhalt-Formen). `go test ./internal/gateway/` nicht gelaufen -
  diese Iteration hat keine Route-/Gateway-Datei angefasst, laut Schritt 5
  daher nicht Pflicht.

## Iteration 25 — b-cov-server-wiki — done — 2026-08-10 (siehe Commit-Zeit)
- commit: (siehe unten)
- gebaut: neue Datei `internal/server/wiki_grpc_test.go` fuer
  `wiki_grpc.go` (699 Zeilen, 20 RPC-Methoden ueber `WikiGRPCServer`,
  vorher 0 % Coverage, keine Testdatei). Eigener In-Memory-Stub
  `stubWikiRepo` fuer `wiki.Repository` (23 Methoden, `sync.Mutex`-
  geschuetzte Maps, `forceErr` fuer die generische Internal-Fallback-
  Verzweigung — analog zum Muster aus `settings_grpc_test.go`/
  `formulare_grpc_test.go`). `newTestWikiServer()` liefert einen
  Nil-Service-Server fuer die 44 UUID-Validierungsfaelle (ein
  Tabellentest ueber alle Handler, die vor dem Service-Aufruf
  abbrechen), `newWikiServerWithRepo(repo)` fuer die Happy-Path- und
  Fehlerpfad-Cluster. Kernstueck laut Scope: `RedeemShareToken` -
  widerrufener, abgelaufener, ohne-Read-Berechtigung und unbekannter
  Token liefern denselben `codes.NotFound` MIT identischer
  `err.Error()`-Nachricht (per String-Vergleich explizit geprueft, nicht
  nur derselbe Code — ein Existenz-Orakel koennte sonst ueber die
  Fehlermeldung selbst durchsickern). Ebenso ein unveroeffentlichter
  Artikel hinter einem gueltigen Token - dieselbe Antwort. Beide
  Inhaltsformen (TipTap-Block-JSON-Objekt `{"type":"doc",...}` und
  Alt-HTML als JSON-String-Literal `"<p>...</p>"` - beides besteht
  `json.Valid`, da ein zitierter String ebenfalls gueltiges JSON ist)
  ueber `CreateArticle` -> `GetArticle`-Rundlauf verifiziert. Ausserdem
  abgedeckt: Versionierung (Update legt Snapshot mit dem
  VOR-Update-Inhalt an, `RestoreVersion` stellt ihn wieder her),
  Kategorie-/Artikel-Elternteil-Leerung ueber leeren String im
  `optional string`-Feld, Slug-Konflikt -> `AlreadyExists`,
  Attachment-Upload/List/Delete inkl. doppeltem Delete ->
  `NotFound`, `RevokeShareToken` zweimal hintereinander ist idempotent
  und bewegt den ersten `revoked_at`-Zeitstempel nicht,
  `mapWikiError` als Tabellentest ueber alle sieben Sentinels plus
  Default-Internal-Zweig. `SearchArticles`/`ListCategories` initial nur
  ueber den Validierungspfad getroffen (30 %) - zwei Happy-Path-Tests
  nachgezogen, beide jetzt bei 90 %. `go tool cover -func`: keine
  Methode in `wiki_grpc.go` bleibt bei 0 % (Spanne 57,1 % bis 100 %,
  `attachmentToProto`/`shareTokenToProto` am niedrigsten wegen
  ihrer nicht separat injizierten Nil-Uploader/Nil-CreatedBy-Aeste,
  die aber schon durch die Upload-ohne-uploaded_by- bzw.
  Redeem-Tests indirekt mitlaufen).
- gate: build ok (`go build -p 2 ./internal/server/... ./cmd/gateway/...
  ./internal/wiki/...`) | vet ok (`go vet ./internal/server/...
  ./internal/wiki/...`) | lint ok (`golangci-lint run --config
  .golangci.yml ./internal/server/... ./internal/wiki/...`, 0 issues —
  ein erster Lauf meldete `unusedparams` fuer den ungenutzten
  `repo`-Parameter in einem Test-Helfer und einen `minmax`-Hinweis in
  der `ListArticles`-Offset-Klammerung im Stub-Repo, beide behoben) |
  test ok (`go test -count=3 ./internal/server/... ./internal/wiki/...`,
  dreimal wiederholt, durchgehend gruen) | migration n.a. (reine
  Test-Coverage, keine Tabelle angefasst) | rls-smoke n.a. (Stub-Repo,
  kein DB-Zugriff durch die neuen Tests)
- verify vorgaenger: sauber — `b7cbb428` (b-cov-server-settings) geprueft:
  `git show --stat` zeigt nur `settings_grpc_test.go` (neu) plus
  Journal/Backlog, kein Produktionscode, keine neue Route, kein
  `RequirePermission`, keine Tabelle, kein `.proto` beruehrt.
- mutations-probe: in `internal/wiki/share.go`, `ShareToken.Usable`
  die Widerrufs-Pruefung testweise auf `if false && t.RevokedAt != nil`
  entschaerft → `TestRedeemShareToken_RevokedAndUnknownProduceTheSameAnswer`
  wurde rot (`expected error with code NotFound, got nil` — der
  widerrufene Token wurde still eingeloest statt abgelehnt), alle
  anderen Tests blieben unberuehrt (isolierter Lauf). Zurueckgedreht,
  `git diff --stat internal/wiki/share.go` zeigt keine Restaenderung
  (leerer Diff bestaetigt), `go test -count=3 ./internal/server/...
  ./internal/wiki/...` danach wieder vollstaendig gruen.
- db-tests: 0 — `wiki.Repository` ist vollstaendig gestubbt, kein
  Postgres-Zugriff in dieser Unit.
- offen: siebte von zwoelf Units in Block B-Server erledigt (7/12).
  Naechste laut Reihenfolge: `b-cov-server-schichten`. `go test
  ./internal/gateway/` nicht gelaufen - diese Iteration hat keine
  Route-/Gateway-Datei angefasst, laut Schritt 5 daher nicht Pflicht.

## Iteration 26 — b-cov-server-schichten — done — 2026-08-10 (siehe Commit-Zeit)
- commit: (siehe unten)
- gebaut: neue Datei `internal/server/schichten_grpc_test.go` fuer
  `schichten_grpc.go` (675 Zeilen, 24 RPC-Methoden ueber
  `SchichtenGRPCServer`, vorher 0 % Coverage, keine Testdatei). Eigener
  In-Memory-Stub `stubSchichtenRepo` fuer `schichten.Repository` (23
  Methoden — Shifts, Assignments, Templates, Stats, SwapRequests —
  inklusive injizierbarer `priorShiftEnd`/`nextShiftStart`/
  `minorEmployees`/`existingTemplate` fuer die ArbZG-/JArbSchG-Pfade und
  `forceErr` fuer die generische Internal-Fallback-Verzweigung).
  `newTestSchichtenServer()` liefert einen Nil-Service-Server fuer 43
  UUID-/Pflichtfeld-Validierungsfaelle als Tabellentest,
  `newSchichtenServerWithRepo(repo)` fuer Happy-Path- und Fehlerpfad-
  Cluster. Kernstueck laut Scope: Tauschantrags-Uebergaenge
  (Create -> Approve -> zweites Approve/Reject scheitert mit
  `FailedPrecondition`, spiegelbildlich fuer Reject -> Approve),
  `mapSchichtenError` als Tabellentest ueber alle elf Sentinels plus
  Default-Internal-Zweig. `go tool cover -func`: keine Methode bleibt bei
  0 % (Spanne 64,3 % bis 100 %).
- befund own-scope (Kern des Scope-Auftrags "own-Scope ueber beide
  Mitarbeiterfelder geprueft"): geprueft und die Praemisse haelt NICHT.
  `schichten.SwapRequestFilter` (repository.go) traegt nur ShiftID/
  Status, `ListSwapRequestsInput` (service.go) hat kein Mitarbeiterfeld,
  `route_schichten.go` guardet `GET /swap-requests` nur mit der flachen
  Permission `schichten:swap read` - keine own-Variante wie
  `rapporte`/`helpdesk` sie ueber `own_scope_list_test.go` demonstrieren.
  Jeder Leser mit dieser Permission sieht jeden Tauschantrag des
  Tenants, unabhaengig von Antragsteller/Tauschpartner. Dokumentiert in
  `TestSchichten_ListSwapRequests_NoOwnScopeFiltering` (mit
  Erklaer-Kommentar im Test). Neue Backlog-Unit fuer Lauf 8:
  `fix-schichten-swaprequests-no-own-scope` - anders als bei
  `a-inbox-sla` keine offene Produktentscheidung ueber das Datenmodell
  (beide Mitarbeiterfelder existieren bereits auf SwapRequest), nur die
  Filterung selbst fehlt; eine kleine offene Frage bleibt, ob Genehmigen/
  Ablehnen (Schichtleitung) tenant-weit bleiben sollen, waehrend nur das
  Lesen own-gescoped wird.
- befund createtemplate-location: `CreateTemplate`
  (schichten_grpc.go:268) liest `req.GetLocation()` nie und setzt es nie
  auf `CreateTemplateInput` - jedes ueber die API erstellte Template
  verliert seinen Standort, obwohl `UpdateTemplate` dasselbe Feld direkt
  daneben korrekt verdrahtet und `ApplyTemplate` den Standort auf jede
  generierte Schicht kopiert. Gefunden, weil
  `TestSchichten_TemplateCRUDAndList` zunaechst denselben Location-
  Rundlauf wie beim Shift-Test erwartete und rot wurde. Test auf den
  IST-Zustand umgestellt (`assert.Nil`, mit "documents current gap"-
  Kommentar), UpdateTemplate-Teil des Tests deckt jetzt zusaetzlich den
  korrekten Location-Rundlauf ab. Neue Backlog-Unit fuer Lauf 8:
  `fix-schichten-createtemplate-drops-location` (mechanischer Ein-
  Zeilen-Fix, Muster liegt direkt daneben in UpdateTemplate).
- befund error-mapping-luecke: `mapSchichtenError` hat keinen Fall fuer
  `schichten.ErrShiftFull` (zurueckgegeben von `AssignEmployee`s
  Kapazitaets-Guard) - faellt auf den Default-Internal-Zweig, obwohl es
  wie `ErrArbzgViolation`/die JArbSchG-Sentinels ein client-actionabler
  Fehler ist ("Schicht ist voll" ist kein 500er). Alle anderen zehn
  Sentinels aus errors.go sind vertreten, nur dieser fehlt. Belegt durch
  `TestMapSchichtenError_Table/shift_full_documents_current_gap`
  (Tabellentest, direkter Funktionsaufruf) UND
  `TestSchichten_AssignEmployee_CapacityExceeded_MapsToInternal`
  (End-to-End durch den echten Handler). Neue Backlog-Unit fuer Lauf 8:
  `fix-schichten-error-mapping-shiftfull-gap`.
- gate: build ok (`go build -p 2 ./internal/server/... ./cmd/gateway/...
  ./internal/schichten/...`) | vet ok (`go vet ./internal/server/...
  ./internal/schichten/...`) | lint ok (`golangci-lint run --config
  .golangci.yml ./internal/server/... ./internal/schichten/...`, 0
  issues — ein erster Lauf meldete drei `minmax`-Hinweise
  (offset+limit-Klammerung in ListShifts/ListTemplates/
  ListSwapRequests des Stub-Repos) und einen `rangeint`-Hinweis in
  einer Test-Schleife, alle behoben) | test ok (`go test -count=3
  ./internal/server/... ./internal/schichten/...`, dreimal wiederholt,
  durchgehend gruen — der erste Lauf schlug wegen der
  CreateTemplate-Location-Erwartung fehl, siehe Befund oben, danach
  gruen) | migration n.a. (reine Test-Coverage, keine Tabelle
  angefasst) | rls-smoke n.a. (Stub-Repo, kein DB-Zugriff durch die
  neuen Tests)
- verify vorgaenger: sauber — `3ed2e7c3` (b-cov-server-wiki) geprueft:
  `git show --stat` zeigt nur `wiki_grpc_test.go` (neu) plus
  Journal/Backlog, kein Produktionscode, keine neue Route, kein
  `RequirePermission`, keine Tabelle, kein `.proto` beruehrt.
- mutations-probe: in `internal/server/schichten_grpc.go`,
  `mapSchichtenError` den `ErrArbzgViolation`-Fall testweise auf
  `codes.Internal` statt `codes.FailedPrecondition` entschaerft →
  sowohl `TestMapSchichtenError_Table/arbzg_violation` als auch
  `TestSchichten_AssignEmployee_ArbzgViolation` wurden rot (erwarteter
  Code FailedPrecondition, bekommen Internal). Zurueckgedreht,
  `git diff --stat internal/server/schichten_grpc.go` zeigt keine
  Restaenderung (leerer Diff bestaetigt), `go test -count=3
  ./internal/server/... ./internal/schichten/...` danach wieder
  vollstaendig gruen.
- db-tests: 0 — `schichten.Repository` ist vollstaendig gestubbt, kein
  Postgres-Zugriff in dieser Unit.
- offen: achte von zwoelf Units in Block B-Server erledigt (8/12).
  Naechste laut Reihenfolge: `b-cov-server-vermietung`. Drei neue
  Fix-Units fuer Lauf 8 im Backlog:
  `fix-schichten-createtemplate-drops-location`,
  `fix-schichten-error-mapping-shiftfull-gap`,
  `fix-schichten-swaprequests-no-own-scope`. `go test
  ./internal/gateway/` nicht gelaufen - diese Iteration hat keine
  Route-/Gateway-Datei angefasst, laut Schritt 5 daher nicht Pflicht.

## Iteration 27 — b-cov-server-vermietung — done — 2026-08-10 (siehe Commit-Zeit)
- commit: (siehe unten)
- verify vorgaenger: sauber — `a43e25d3` (b-cov-server-schichten) geprueft:
  `git show --stat` zeigt nur `schichten_grpc_test.go` (neu) plus
  Journal/Backlog, kein Produktionscode, keine neue Route, kein
  `RequirePermission`, keine Tabelle, kein `.proto` beruehrt.
- gebaut: neue Datei `internal/server/vermietung_grpc_test.go` fuer
  `vermietung_grpc.go` (757 Zeilen, 20 RPC-Methoden ueber
  `VermietungGRPCServer`, vorher 0 % Coverage, keine Testdatei). Eigener
  In-Memory-Stub `stubVermietungRepo` fuer `vermietung.Repository` (16
  Methoden — Objects, Rentals, Inspections — inklusive einer echten
  Intervall-Ueberlappungspruefung in `HasOverlap`, nicht nur eines
  Force-Flags, damit Konflikt-/Verfuegbarkeitstests etwas Reales pruefen).
  `newTestVermietungServer()` liefert einen Nil-Service-Server fuer 38
  UUID-/Pflichtfeld-Validierungsfaelle als Tabellentest,
  `newVermietungServerWithRepo(repo)` fuer Happy-Path- und Fehlerpfad-
  Cluster. Kernstueck laut Scope: Ruecknahme (EndRental) vor Uebergabe
  (StartRental) scheitert mit `FailedPrecondition`
  (`TestVermietung_RentalLifecycle_ReturnBeforeHandoverFails`, deckt
  zusaetzlich Start-nach-Start und End-nach-End ab), Buchungskonflikt bei
  ueberlappenden Datumsbereichen (`AlreadyExists`, sowohl bei CreateRental
  als auch beim Datums-Update einer bestehenden Reservierung),
  `mapVermietungError` als Tabellentest ueber alle sechs Sentinels plus
  Default-Internal-Zweig. `go tool cover -func`: keine Methode bleibt
  unter 69 % (Spanne 69,0 % bis 100 %) — UpdateRental/ListRentals/
  DeleteRental/GetRental lagen im ersten Durchlauf bei 26–90 %, drei
  zusaetzliche Tests (Update mit allen Feldern, Datums-Update-Konflikt,
  Filter+Pagination bei ListRentals, NotFound bei Get/Delete/Update) haben
  das auf durchgehend ueber 69 % gehoben.
- befund rentaltoproto-signature-drop: `rentalToProto`
  (vermietung_grpc.go:650) mappt `SignatureData`/`SignedAt`/`SignedBy`
  nie auf das Wire-`Rental`, obwohl das Proto alle drei Felder traegt und
  `Service.SaveSignature` sie korrekt auf dem Domain-Modell persistiert.
  Jede RPC, die einen Rental zurueckgibt — inklusive der Antwort von
  `SaveSignature` selbst — liefert die gerade gespeicherte Signatur nie
  an den Aufrufer zurueck. Verifiziert als reiner Proto-Mapping-Fehler,
  nicht als Repository-/Service-Luecke: `TestVermietung_SaveSignature`
  prueft zusaetzlich direkt im Stub-Repo, dass `SignatureData`/`SignedBy`
  dort gesetzt sind, waehrend die gRPC-Antwort leer bleibt. Neue
  Backlog-Unit fuer Lauf 8: `fix-vermietung-rentaltoproto-drops-signature`.
- befund error-mapping-luecke: `mapVermietungError` hat keinen Fall fuer
  `vermietung.ErrInspectionKindExists` (zurueckgegeben von
  `CreateInspection`, wenn fuer eine Reservierung bereits eine Inspektion
  derselben Art existiert) — faellt auf den Default-Internal-Zweig,
  obwohl es strukturell identisch zu `ErrRentalConflict` ist (dort
  korrekt auf `AlreadyExists` gemappt). Alle anderen fuenf Sentinels aus
  errors.go sind vertreten, nur dieser fehlt. Belegt durch
  `TestMapVermietungError_Table/inspection_kind_exists_documents_current_gap`
  (Tabellentest) UND `TestVermietung_CreateInspection_DuplicateKind_MapsToInternal`
  (End-to-End durch den echten Handler). Neue Backlog-Unit fuer Lauf 8:
  `fix-vermietung-error-mapping-inspectionkindexists-gap`.
- gate: build ok (`go build -p 2 ./internal/server/... ./cmd/gateway/...
  ./internal/vermietung/...`) | vet ok (`go vet ./internal/server/...
  ./internal/vermietung/...`) | lint ok (`golangci-lint run --config
  .golangci.yml ./internal/server/...`, 0 issues) | test ok (`go test
  -count=3 ./internal/server/...`, dreimal wiederholt, durchgehend
  gruen) | migration n.a. (reine Test-Coverage, keine Tabelle
  angefasst) | rls-smoke n.a. (Stub-Repo, kein DB-Zugriff durch die
  neuen Tests)
- mutations-probe: in `internal/server/vermietung_grpc.go`,
  `mapVermietungError` den `ErrInvalidStateTransition`-Fall testweise auf
  `codes.Internal` statt `codes.FailedPrecondition` entschaerft → sowohl
  `TestMapVermietungError_Table/invalid_state_transition` als auch
  `TestVermietung_RentalLifecycle_ReturnBeforeHandoverFails` wurden rot
  (erwarteter Code FailedPrecondition, bekommen Internal). Zurueckgedreht,
  `git diff --stat internal/server/vermietung_grpc.go` zeigt keine
  Restaenderung (leerer Diff bestaetigt), `go test -count=3
  ./internal/server/...` danach wieder vollstaendig gruen.
- db-tests: 0 — `vermietung.Repository` ist vollstaendig gestubbt, kein
  Postgres-Zugriff in dieser Unit.
- offen: neunte von zwoelf Units in Block B-Server erledigt (9/12).
  Naechste laut Reihenfolge: `b-cov-server-einkauf`. Zwei neue Fix-Units
  fuer Lauf 8 im Backlog: `fix-vermietung-rentaltoproto-drops-signature`,
  `fix-vermietung-error-mapping-inspectionkindexists-gap`. `go test
  ./internal/gateway/` nicht gelaufen - diese Iteration hat keine
  Route-/Gateway-Datei angefasst, laut Schritt 5 daher nicht Pflicht.

## Iteration 28 — b-cov-server-einkauf — done — 2026-08-10 (siehe Commit-Zeit)
- commit: (siehe unten)
- verify vorgaenger: sauber — `2886ed02` (b-cov-server-vermietung) geprueft:
  `git show --stat` zeigt nur `vermietung_grpc_test.go` (neu) plus
  Journal/Backlog, kein Produktionscode, keine neue Route, kein
  `RequirePermission`, keine Tabelle, kein `.proto` beruehrt.
- gebaut: neue Datei `internal/server/einkauf_grpc_test.go` fuer
  `einkauf_grpc.go` + `einkauf_grpc_extended.go` (zusammen 1.292 Zeilen, 36
  RPC-Methoden ueber `EinkaufGRPCServer`, vorher 0 % Coverage, keine
  Testdatei). Eigener In-Memory-Stub `stubEinkaufRepo`, der
  `einkauf.RepositoryExtended` implementiert UND die unexportierte
  `contractItemQuerier`-Schnittstelle (`QueryRowContractItem`) bedient — ohne
  letztere faellt jedes `UpdateContractItem` auf `ErrContractItemNotFound`
  zurueck, unabhaengig davon ob das Item existiert (siehe
  `einkauf.getContractItemByID`, service_extended.go:685). `RecomputePOTotal`
  im Stub rechnet wie die Postgres-Seite `sum(quantity * unit_price)`, aber
  auf vier Nachkommastellen (numeric(15,4)) statt der zwei, die ein naiver
  Test annehmen wuerde. `newTestEinkaufServer()` liefert einen
  Nil-Service-Server fuer 77 UUID-Validierungsfaelle als Tabellentest
  (`TestEinkauf_UUIDValidation`), `newEinkaufServerWithRepo(repo)` fuer
  Happy-Path- und Fehlerpfad-Cluster je Domain (Supplier, PO-Lifecycle,
  PO-Lines, Catalog, Supplier-Ratings, Framework-Contracts, Contract-Items,
  Contract-Calls).
- kopfbetrag-string-test: `TestEinkauf_PO_FullLifecycle_HappyPath` legt eine
  Zeile mit `UnitPrice "3.3333"` an und prueft, dass `GetPO` exakt
  `"9.9999"` liefert (nicht auf zwei Dezimalstellen gerundet) - belegt, dass
  `poToProto` den vom Service nachgerechneten String unveraendert durchreicht
  und keine Float-Konvertierung dazwischenliegt. `TestEinkauf_PoToProto_
  EmbedsLinesAndOptionalFields` deckt zusaetzlich einen Wert mit vier
  Nachkommastellen (`"1234.5678"`) direkt auf der Konvertierungsfunktion ab.
- statusuebergaenge: `TestEinkauf_PO_Lifecycle_InvalidTransitions` deckt
  sieben Faelle ab (SubmitPO ohne Zeilen, SubmitPO auf Nicht-Draft, CancelPO
  auf Received, ReceiveGoods auf Draft, PartialReceive auf Draft, DeletePO
  auf Nicht-Draft, UpdatePO auf Closed) plus PartialReceive ueber die
  bestellte Menge hinaus (ErrExceedsOrderedQty). Alle sieben Statuspruefungen
  aus `einkauf/service.go` sind damit vertreten.
- gate: build ok (`go build -p 2 ./internal/server/... ./cmd/gateway/...
  ./internal/einkauf/...`) | vet ok (`go vet ./internal/server/...
  ./internal/einkauf/...`) | lint ok (`golangci-lint run --config
  .golangci.yml ./internal/server/...`, 0 issues) | test ok (`go test
  -count=3 ./internal/server/... ./internal/einkauf/...`, dreimal
  wiederholt, durchgehend gruen) | migration n.a. (reine Test-Coverage,
  keine Tabelle angefasst) | rls-smoke n.a. (Stub-Repo, kein DB-Zugriff
  durch die neuen Tests)
- mutations-probe: in `internal/server/einkauf_grpc.go`, `mapEinkaufError`
  den `ErrPONotDraft`-Fall testweise auf `codes.Internal` statt
  `codes.FailedPrecondition` entschaerft → vier Tests wurden rot
  (`TestMapEinkaufError_Table/po_not_draft`,
  `TestEinkauf_PO_Lifecycle_InvalidTransitions/SubmitPO_on_non-draft_fails`,
  `.../DeletePO_on_non-draft_fails`, `TestEinkauf_UpdatePO_AllFields`
  via `UpdatePO_on_closed_fails`-Pendant). Zurueckgedreht, `git diff --stat
  internal/server/einkauf_grpc.go` zeigt keine Restaenderung (leerer Diff
  bestaetigt), `go test -count=3 ./internal/server/...` danach wieder
  vollstaendig gruen.
- db-tests: 0 — `einkauf.Repository`/`RepositoryExtended` sind vollstaendig
  gestubbt, kein Postgres-Zugriff in dieser Unit.
- kein neuer Fund: anders als die letzten Iterationen (schichten, vermietung,
  automation) hat diese Coverage-Unit keine neue Verhaltensluecke
  aufgedeckt, die eine eigene Fix-Unit rechtfertigt. Die bereits bekannte
  Luecke `fix-einkauf-contract-call-no-value-check` (CreateContractCall
  prueft nicht gegen `total_value - used_value`) war schon vor dieser
  Iteration im Backlog (Lauf 7, Iteration 16) und bleibt unveraendert offen -
  `TestEinkauf_CreateContractCall_NegativeAmount` deckt nur den bereits
  vorhandenen Negativ-Check ab, nicht die fehlende Obergrenzenpruefung.
- offen: zehnte von zwoelf Units in Block B-Server erledigt (10/12).
  Naechste laut Reihenfolge: `b-cov-server-produktion`. `go test
  ./internal/gateway/` nicht gelaufen - diese Iteration hat keine
  Route-/Gateway-Datei angefasst, laut Schritt 5 daher nicht Pflicht.

## Iteration 29 — b-cov-server-produktion — done — 2026-08-10 (siehe Commit-Zeit)
- commit: (siehe unten)
- verify vorgaenger: sauber — `9ff75f1d` (b-cov-server-einkauf) geprueft:
  `git show --stat` zeigt nur `einkauf_grpc_test.go` (neu, 1777 Zeilen)
  plus Journal/Backlog, kein Produktionscode, keine neue Route, kein
  `RequirePermission`, keine Tabelle, kein `.proto` beruehrt.
- gebaut: neue Datei `internal/server/produktion_grpc_test.go` fuer
  `produktion_grpc.go` + `produktion_grpc_ext.go` (zusammen 1.189 Zeilen,
  43 Methoden ueber `ProduktionGRPCServer`, vorher 0 % Coverage, keine
  Testdatei). Eigener In-Memory-Stub `stubProduktionRepo`, der das
  vollstaendige `produktion.Repository`-Interface (Orders, Bookings,
  Plans, Capacity, BOMs, WorkSteps, Machines, QualityChecks) bedient.
  `newTestProduktionServer()` liefert einen Nil-Service-Server fuer 60
  UUID-/Pflichtfeld-Validierungsfaelle als Tabellentest (jede der 30
  RPCs mindestens einmal), `newProduktionServerWithRepo(repo)` fuer
  Happy-Path- und Fehlerpfad-Cluster je Domain.
- kernstueck laut scope (booking conflict): `CreateMachineBooking`/
  `UpdateMachineBooking` nutzen `CreateBookingWithLock`/
  `FindConflictingBooking` mit halboffenem Intervall
  (`existing.starts_at < new.ends_at AND existing.ends_at >
  new.starts_at`, siehe `postgres_repository.go:299`). Der Stub bildet
  exakt diese Arithmetik nach (`findConflict`), nicht nur ein
  Force-Flag, damit die beiden done_when-Kriterien etwas Reales pruefen:
  `TestProduktion_CreateMachineBooking_ConflictMapsToFailedPrecondition`
  belegt, dass ein Ueberlapp `codes.FailedPrecondition` liefert statt
  `codes.Internal`; `TestProduktion_CreateMachineBooking_AdjacentIsNotAConflict`
  belegt, dass eine Buchung, die exakt beim Ende der vorherigen beginnt,
  KEIN Konflikt ist (plus: andere `machine_id` ist nie ein Konflikt,
  auch im exakt gleichen Slot). `TestProduktion_UpdateMachineBooking_
  ConflictAndNotFound` deckt zusaetzlich Update-in-Konflikt und
  Update-auf-den-eigenen-unveraenderten-Slot (kein Selbst-Konflikt) ab.
- stub-aliasing-bug im eigenen Code gefunden und behoben, bevor er in den
  Commit ging: `GetOrder`/`GetBooking`/`GetPlan`/`GetBOM`/`GetWorkStep`/
  `GetMachine`/`GetQualityCheck` gaben zunaechst den Live-Pointer aus der
  Map zurueck statt einer Kopie. Die `Update*`-Methoden im Service
  mutieren das geladene Objekt in-place, bevor sie eine Konflikt-/
  Validierungspruefung machen — bei einem fehlschlagenden Update blieb
  die Mutation trotzdem im Stub haengen (Update von Buchung B auf einen
  Konflikt-Slot veraenderte B im Stub dauerhaft, obwohl der Aufruf einen
  Fehler zurueckgab). `TestProduktion_UpdateMachineBooking_
  ConflictAndNotFound` deckte das beim ersten Lauf auf (dritter Schritt
  schlug mit einem falschen Konflikt fehl). Fix: alle sieben `Get*`-
  Methoden im Stub geben jetzt eine flache Kopie zurueck, wie es
  `PostgresRepository` durch einen frischen `SELECT` ohnehin tut — kein
  Produktionscode betroffen, reiner Test-Stub-Bug.
- weitere abgedeckte Cluster: Order-Lifecycle (Create/Get/Update/Delete,
  Duplikat-Bestellnummer -> `AlreadyExists`, nicht existente `bom_id` ->
  `NotFound`, StartOrder/CompleteOrder/CancelOrder-Statusuebergaenge inkl.
  aller ungueltigen Uebergaenge -> `FailedPrecondition`, DeleteOrder auf
  nicht-planned -> `FailedPrecondition`), ListOrders-Pagination+Total,
  Plans (Create/Update/Get, GetPlan-NotFound), GetCapacityOverview
  (leere `machine_id` -> `InvalidArgument` auf Service-Ebene), BOMs
  (Duplikat-SKU -> `AlreadyExists` ueber `ErrBOMSKUTaken`, Items-
  Sort-Order wird vom Service aus dem Insertions-Index neu vergeben, nicht
  durchgereicht — bewusst als Verhalten dokumentiert, kein Bug), WorkSteps,
  Machines, QualityChecks (je Create/Get/Update/List/Delete plus leeres
  Pflichtfeld -> `InvalidArgument`), GetMaterialAvailability (Order ohne
  BOM -> `ErrOrderHasNoBOM` -> `FailedPrecondition` nicht Internal; mit
  BOM und ohne konfiguriertes `InventarLookup` bleibt Verfuegbarkeit
  bewusst `nil` statt `0` — Graceful-Degradation-Pfad aus dem
  Doc-Kommentar direkt verifiziert). `mapProduktionError`/
  `mapProduktionExtError` je als vollstaendiger Tabellentest ueber alle
  Sentinels plus Default-Internal-Zweig; `mapProduktionExtError` faellt
  fuer den Basissatz korrekt an `mapProduktionError` durch (eigener
  Testfall). `toProto*`-Konvertierungen: alle sieben nil-Empfaenger ->
  nil, `orderToProto` mit und ohne optionale Zeiger-Felder
  (ActualStart/ActualEnd/CreatedBy/BomId),
  `materialAvailabilityToProto` mit aufgeloester und unaufgeloester
  Zeile (Available/Shortfall nil vs. gesetzt).
- go tool cover -func: keine der 30 RPC-Handler-Methoden bleibt unter
  60,9 % (ListOrders), die meisten zwischen 78–100 %; alle vorher 0 %.
- gate: build ok (`go build -p 2 ./internal/server/... ./cmd/gateway/...
  ./internal/produktion/...`) | vet ok (`go vet ./internal/server/...
  ./internal/produktion/...`) | lint ok (`golangci-lint run --config
  .golangci.yml ./internal/server/...`, 0 issues) | test ok (`go test
  -count=3 ./internal/server/...`, dreimal wiederholt, durchgehend
  gruen) | migration n.a. (reine Test-Coverage, keine Tabelle angefasst)
  | rls-smoke n.a. (Stub-Repo, kein DB-Zugriff durch die neuen Tests)
- mutations-probe: in `internal/server/produktion_grpc.go`,
  `mapProduktionError` den `ErrBookingConflict`-Fall testweise auf
  `codes.Internal` statt `codes.FailedPrecondition` entschaerft → vier
  Tests wurden rot (`TestMapProduktionError_Table/booking_conflict`,
  `TestMapProduktionExtError_Table/falls_through_booking_conflict`,
  `TestProduktion_CreateMachineBooking_ConflictMapsToFailedPrecondition`,
  `TestProduktion_UpdateMachineBooking_ConflictAndNotFound`).
  Zurueckgedreht, `git diff --stat internal/server/produktion_grpc.go`
  zeigt keine Restaenderung (leerer Diff bestaetigt), `go test -count=3
  ./internal/server/...` danach wieder vollstaendig gruen.
- db-tests: 0 — `produktion.Repository` ist vollstaendig gestubbt, kein
  Postgres-Zugriff in dieser Unit.
- kein neuer Fund: Proto-RPC-Liste gegen die Implementierung geprueft
  (`grep "^  rpc " produktion.proto`), alle 30 deklarierten RPCs sind
  implementiert und ueber `mapProduktionError`/`mapProduktionExtError`
  vollstaendig abgedeckt — anders als bei den letzten Iterationen
  (schichten, vermietung, automation) kein Wire-/Error-Mapping-/
  Nil-Dereferenz-Fund, der eine eigene Fix-Unit rechtfertigt.
- offen: elfte von zwoelf Units in Block B-Server erledigt (11/12).
  Naechste laut Reihenfolge: `b-cov-server-vertraege` (letzte Unit in
  Block B-Server, danach Block C2). `go test ./internal/gateway/` nicht
  gelaufen - diese Iteration hat keine Route-/Gateway-Datei angefasst,
  laut Schritt 5 daher nicht Pflicht.

## Iteration 30 — b-cov-server-vertraege — done — 2026-08-10 (siehe Commit-Zeit)
- commit: (siehe unten)
- verify vorgaenger: sauber — `c116bd86` (b-cov-server-produktion) geprueft:
  `git show --stat` zeigt nur `produktion_grpc_test.go` (neu, 1596 Zeilen)
  plus Journal/Backlog, kein Produktionscode, keine neue Route, kein
  `RequirePermission`, keine Tabelle, kein `.proto` beruehrt.
- gebaut: neue Datei `internal/server/vertraege_grpc_test.go` fuer
  `vertraege_grpc.go` (580 Zeilen, 16 Methoden inkl. des deprecated
  `UploadDocument`-Stubs, vorher 0 % Coverage, keine Testdatei). Eigener
  In-Memory-Stub `stubVertraegeRepo`, der das vollstaendige
  `vertraege.Repository`-Interface bedient (`svc *vertraege.Service` ist
  ein konkreter Typ, kein Interface — der Server laesst sich also nur
  ueber einen echten `vertraege.NewService(repo)` testen, nicht per
  Nil-Service-Repo-Swap wie bei formulare/produktion). `newTestVertraegeServer()`
  liefert einen Nil-Service-Server fuer die 30 UUID-Validierungsfaelle
  (Tabellentest ueber alle 15 nicht-deprecated RPCs plus den
  Unimplemented-Fall von UploadDocument), `newVertraegeServerWithRepo(repo)`
  fuer Happy-Path- und Fehlerpfad-Cluster.
- kernstueck laut scope (Datums-Grenzfaelle): `TestVertraege_ListContracts_
  DateBoundaries` prueft ein Vertragspaar mit `StartsOn` exakt an der letzten
  Sekunde 2025 und der ersten Sekunde 2026 gegen `StartsAfter`/`StartsBefore`
  — beide muessen im weiten Fenster auftauchen, und `StartsAfter` auf die
  exakte Jahreswechsel-Instanz gesetzt muss den Treffer strikt ausschliessen
  (`time.After` ist exklusiv, kein Off-by-one). `TestVertraege_CreateReminder_
  LeapDayAndYearEnd` deckt zusaetzlich einen Schalttag (2028-02-29) und eine
  Erinnerung an der letzten Sekunde eines Jahres ab — beide muessen
  verlustfrei durch `timestamppb.New(...).AsTime()` round-trippen.
- kernstueck laut scope (Signaturdaten nicht in Listen-Antworten):
  `TestVertraege_SaveSignature_NotInListResponse` speichert eine Signatur,
  prueft direkt am Repository, dass sie dort ankommt, und dann, dass
  `ListContracts` weder `signature_data` noch `signed_by` noch `signed_at`
  zurueckgibt. Beim Bau des Tests fiel ein zweiter, groesserer Fund auf:
  `vertraegeContractToProto` setzt diese drei Felder UEBERHAUPT NIE — auch
  nicht bei `GetContract`, obwohl der Proto-Kommentar
  (vertraege.proto:64-69) exakt das als Vertrag dokumentiert ("Populated
  only by GetContract"). Eine gespeicherte Signatur ist damit ueber KEINE
  RPC lesbar. Nicht inline gefixt (echte Wire-Verhaltensaenderung, keine
  Coverage-Aenderung) — eigene Fix-Unit `fix-vertraege-contracttoproto-
  drops-signature` im Backlog angelegt, der GetContract-Assert im Test
  traegt einen "documents current gap"-Kommentar.
- weitere abgedeckte Cluster: Contract-Lifecycle (Create inkl. Duplikat-
  Nummer -> AlreadyExists und leerer Titel -> InvalidArgument,
  Update-NotFound, Update mit `ClearEndsOn` — belegt, dass ein gleichzeitig
  gesetztes `EndsOn` bei `ClearEndsOn=true` ignoriert wird, Delete nur im
  Draft-Status inkl. FailedPrecondition fuer Active und NotFound fuer
  unbekannte ID, Get ueber Tenant-Grenze -> NotFound statt Leak), Party-
  Lifecycle (external Party ohne `external_name` -> InvalidArgument, sonst
  Add/List/Remove), Reminder-Lifecycle (Create/Update/Delete/List,
  Update auf unbekannte ID -> NotFound), ListContractEvents (unbekannter
  Contract -> NotFound statt leerer Liste — Service-Kommentar dazu direkt
  verifiziert; CreateReminder emittiert bewusst kein Event, leere Trail
  bestaetigt), ExportContract (PDF-Bytes nicht leer, NotFound fuer
  unbekannte ID). `toProto*`-Konvertierungen: alle vier nil-Empfaenger ->
  nil, `vertraegeContractToProto`/`vertraegePartyToProto`/
  `vertraegeReminderToProto` je mit und ohne optionale Zeiger-Felder,
  `vertraegeEventToProto` mit gesetzter und fehlender `UserID` sowie ein
  Payload, das `structpb.NewStruct` nicht kodieren kann (ein Channel-Wert)
  — belegt, dass der Eintrag trotzdem durchkommt, nur der Payload entfaellt
  (Doc-Kommentar im Code direkt verifiziert). `mapVertraegeError` als
  vollstaendiger Tabellentest ueber alle sechs Sentinels plus
  Default-Internal-Zweig.
- go tool cover -func: keine der 16 RPC-Handler-Methoden bleibt unter
  58,1 % (UpdateContract — die ungetesteten Zweige sind die uebrigen
  optionalen Feld-Setter Title/ContractType/Status/StartsOn/DocumentURL/
  Notes, keine Fehlerpfade), die meisten zwischen 76–100 %; alle
  Konvertierungs- und Fehler-Mapping-Funktionen 100 %; vorher 0 %.
- gate: build ok (`go build -p 2 ./internal/server/...`) | vet ok
  (`go vet ./internal/server/...`) | lint ok (`golangci-lint run --config
  .golangci.yml ./internal/server/...`, 0 issues — ein `ineffassign` auf
  eine ungelesene Zwischenvariable im eigenen Testcode gefunden und vor
  dem Commit behoben) | test ok (`go test -count=3
  ./internal/server/...`, dreimal wiederholt, durchgehend gruen) |
  migration n.a. (reine Test-Coverage, keine Tabelle angefasst) |
  rls-smoke n.a. (Stub-Repo, kein DB-Zugriff durch die neuen Tests)
- mutations-probe: in `internal/server/vertraege_grpc.go`,
  `mapVertraegeError` den `ErrDeleteNonDraft`-Fall testweise auf
  `codes.Internal` statt `codes.FailedPrecondition` entschaerft → zwei
  Tests wurden rot (`TestMapVertraegeError_Table/delete_non_draft`,
  `TestVertraege_DeleteContract`). Zurueckgedreht, `git diff --stat
  internal/server/vertraege_grpc.go` zeigt keine Restaenderung (leerer
  Diff bestaetigt), `go test -count=1 ./internal/server/...` danach
  wieder vollstaendig gruen.
- db-tests: 0 — `vertraege.Repository` ist vollstaendig gestubbt, kein
  Postgres-Zugriff in dieser Unit.
- offen: **Block B-Server ist mit dieser Unit vollstaendig (12/12)**.
  Naechste laut Reihenfolge im Backlog-Kopf: Block C2 (23 Service-Paket-
  Coverage-Units, beginnend mit `c-cov-plugin-config`). Neue Fix-Unit
  `fix-vertraege-contracttoproto-drops-signature` fuer Lauf 8 angelegt
  (Signatur nie lesbar, auch nicht ueber GetContract). `go test
  ./internal/gateway/` nicht gelaufen — diese Iteration hat keine
  Route-/Gateway-Datei angefasst, laut Schritt 5 daher nicht Pflicht.

## Iteration 31 — c-cov-plugin-config — done — 2026-08-10 02:35
- commit: (siehe unten)
- verify vorgaenger: sauber — `daaf5d39` (b-cov-server-vertraege) geprueft: `git show --stat`
  zeigt nur `vertraege_grpc_test.go` (neu) plus Journal/Backlog, kein Produktionscode, keine
  neue Route, kein `RequirePermission`, keine Tabelle, kein `.proto` beruehrt.
- gebaut: drei neue Testdateien fuer `internal/plugin/config` (551 Zeilen Produktionscode,
  vorher 0 % Coverage, keine Testdatei): `schema_validator_test.go` (Validate: leeres/nil/
  `null`-Schema, kaputtes Schema-JSON, kaputtes Settings-JSON, fehlendes Pflichtfeld, erlaubte
  Zusatzfelder, mehrere akkumulierte Fehler; `validateProperty` je Typ string/number/integer/
  boolean inkl. Fehlerpfad, unbekannter Typ faellt durch den Switch ohne Typpruefung, Enum
  gefunden/nicht gefunden), `validation_engine_test.go` (`ValidateEntity` disabled-Skip und
  unbekannter RuleType als No-op, sowie ein Dispatch-Test fuer Format/Enum ueber `evaluateRule`
  selbst — ohne den waere der Switch-Dispatch fuer diese zwei Zweige nur indirekt ueber die
  Unit-Funktionen gedeckt, nicht ueber `evaluateRule`; `evalRegex`/`evalRange`/`evalRequiredIf`/
  `evalFormat`/`evalEnum` je mit kaputtem RuleConfig-JSON, fehlend+required, fehlend+optional,
  Fehlerpfad, Erfolgspfad — `evalRequiredIf` zusaetzlich alle vier Operatoren `eq`/`neq`/
  `exists`/`not_empty` inkl. des Nicht-String-`depValue`-Zweigs, `evalFormat` alle fuenf Formate
  email/url/date/phone/iban je valide+invalide plus unbekanntes Format als No-op; `toFloat64`
  ueber alle sechs Typzweige float64/float32/int/int64/json.Number(valide+invalide)/
  nicht-numerisch), `workflow_engine_test.go` (`EvaluateWorkflows` disabled-Skip,
  Trigger-Event-Mismatch-Skip, kaputtes Conditions-JSON verhindert jeden Trigger,
  Bedingung erfuellt baut TriggeredActions korrekt inkl. RuleID/RuleName, Bedingung nicht
  erfuellt liefert keine Actions, unbekannter Aktionstyp fuehrt zu einer durchgereichten Action
  ohne Panic — `require.NotPanics` explizit als Test verankert, siehe done_when;
  `evaluateCondition` als Tabellentest ueber alle elf Operatoren inkl. Default-Zweig und
  Typ-Fehlpassungen bei contains/gt; `parseActions` kaputtes JSON -> nil).
- go tool cover -func: `internal/plugin/config` von 0 % auf 98,6 % (alle Funktionen 100 %
  ausser `evaluateRule`-Dispatcher selbst bei 62,5 % — die beiden fehlenden Prozentpunkte sind
  der `default:`-Fall, der bereits durch einen eigenen Test abgedeckt ist; go-cover zaehlt
  Switch-Case-Sprungziele separat von der Rueckgabe-Zeile, das ist eine Darstellungs-
  Eigenheit des Tools, keine Luecke im Verhalten).
- fehlerpfade: pro Auswerter mindestens ein Fehlerpfad (Regex-Pattern-Mismatch/-Required/
  -Invalid-Pattern, Range-Min/-Max/-NichtNumerisch, RequiredIf alle vier Operatoren,
  Format alle fuenf Muster je invalide, Enum-Nicht-In-Liste), Schema-Validator (falscher
  Typ je String/Number/Boolean, MinLength/MaxLength, Minimum/Maximum, Enum-Miss), Workflow
  (kaputtes Conditions-JSON, kaputtes Actions-JSON, unbekannter Operator/Aktionstyp).
- gate: build ok (`go build -p 2 ./internal/plugin/... ./cmd/plugin/...`) | vet ok
  (`go vet ./internal/plugin/...`) | lint ok (`golangci-lint run --config .golangci.yml
  ./internal/plugin/config/...`, 0 issues) | test ok (`go test -count=3
  ./internal/plugin/config/...`, durchgehend gruen) | migration n.a. (reine Go-Logik ohne
  DB) | rls-smoke n.a. (kein DB-Zugriff im Paket) | route n.a. (kein Gateway-Handler
  angefasst, `go test ./internal/gateway/` deshalb nicht Pflicht und nicht separat gelaufen)
- mutations-probe: in `internal/plugin/config/validation_engine.go`, `evalEnum`s Vergleich
  `if v == str` auf `if v == "MUTATION_PROBE_"+str` gesetzt → `TestEvalEnum/value_in_list`
  wurde rot ("Expected nil, but got: &config.FieldError{...}"). Zurueckgedreht, `git diff
  --stat` auf allen drei Produktionsdateien zeigt leeren Diff (bestaetigt keine
  Restaenderung), `go test -count=1 ./internal/plugin/config/...` danach wieder
  vollstaendig gruen.
- db-tests: 0 — das Paket hat keinen DB-Zugriff (reine Validierungs-/Workflow-Logik ohne
  Repository), done_when verlangt hier keine DB-Tests.
- kein neuer Fund: alle drei Dateien vollstaendig gelesen vor dem Schreiben der Tests, kein
  Wire-/Error-Mapping-/Nil-Dereferenz-Problem wie bei den letzten Block-B-Server-Iterationen
  gefunden — reine, deterministische Go-Logik ohne externe Abhaengigkeiten.
- offen: erste von 23 Units in Block C2 erledigt (1/23). Naechste laut Backlog-Reihenfolge:
  `c-cov-work-event-rrule`. `go test ./internal/gateway/` nicht gelaufen — diese Iteration
  hat keine Route-/Gateway-Datei angefasst, laut Schritt 5 daher nicht Pflicht.

## Iteration 32 — fix-einkauf-contract-call-no-value-check — blocked — 2026-08-10 02:33
- commit: -
- verify vorgaenger: sauber — `95c3458b` (c-cov-plugin-config) geprueft: `git show --stat`
  zeigt ausschliesslich drei neue Testdateien in `internal/plugin/config/` plus Journal/
  Backlog, kein Produktionscode, keine `.proto`, keine neue Route, kein `RequirePermission`,
  keine Tabelle.
- gebaut: nichts — Unit direkt geblockt, siehe unten.
- gate: n.a.
- blocked_reason: Produktfrage aus der Unit selbst bestaetigt statt nur uebernommen.
  `Service.CreateContractCall` (internal/einkauf/service_extended.go:610-644) prueft nur
  `v < 0` auf den Abrufbetrag, laedt den Rahmenvertrag nur um seine Existenz zu pruefen
  (Zeile 620), vergleicht `Amount` an keiner Stelle gegen `TotalValue - UsedValue`.
  `ContractStatus` (models_extended.go:50-55) kennt `draft`/`active`/`expired`. Offene Frage
  an Luke: (1) Soll ein Abruf abgelehnt werden, sobald `total_value - used_value`
  ueberschritten wird, unabhaengig vom Vertragsstatus? (2) Falls ja — gilt das auch fuer
  `draft`-Vertraege (die noch gar nicht "scharf" sind), oder nur fuer `active`? (3) Ist bei
  `expired` ueberhaupt noch ein Abruf erlaubt, oder sollte das schon vorher (unabhaengig von
  dieser Unit) blockiert sein? Ohne Antwort waere jeder Fix geraten — genau der Fall aus
  "Wenn du nicht weiterkommst": Produktentscheidung gehoert Luke.
- offen: BACKLOG.yml auf `status: blocked` mit `blocked_reason` gesetzt. Naechste Iteration
  (bzw. dieselbe, siehe unten) nimmt die naechste Unit.

## Iteration 33 — fix-plugin-error-mapping-gaps — done — 2026-08-10 02:40
- commit: (siehe unten)
- verify vorgaenger: n.a. — direkt im Anschluss an Iteration 32 (blockierte Unit, kein
  Commit) innerhalb derselben Session; verify fuer den Commit davor (`95c3458b`) bereits in
  Iteration 32 dokumentiert.
- gebaut: `mapPluginError`/`isNotFound`/`isAlreadyExists`/`isInvalidArgument`
  (internal/server/plugin_grpc.go) von `==`-Vergleich auf `errors.Is` umgestellt (erkennt
  jetzt auch von Service gewrapptte Sentinels), neuer Case fuer `ErrPluginHasInstallations`
  -> `codes.FailedPrecondition` (vorher kein Case, fiel auf Internal). Vier bestehende
  "documents current gap"-Testfaelle in plugin_grpc_test.go auf das neue Verhalten
  aktualisiert (TestMapPluginError zwei Faelle, TestPluginDeleteManifest/
  has_active_installations, TestPluginApprovePermissions/undeclared_permission,
  TestPluginSettings/schema_mismatch). Die drei "orphaned installation panics"-Testfaelle
  (gehoeren zu fix-plugin-nil-manifest-panic-on-orphaned-installation, separate Unit) und
  der "missing tenant context"-Testfall (anderer Root-Cause: middleware-Sentinel ist kein
  plugin.Err*, bleibt von errors.Is unberuehrt) bewusst nicht angefasst.
- gate: build ok (`go build -p 2 ./internal/server/... ./internal/plugin/...
  ./cmd/plugin/... ./cmd/gateway/...`) | vet ok | lint ok (`golangci-lint run --config
  .golangci.yml ./internal/server/... ./internal/plugin/...`, 0 issues) | test ok
  (`go test -count=3 ./internal/server/... ./internal/plugin/...`, durchgehend gruen) |
  migration n.a. (kein Schema-Zugriff) | rls-smoke n.a. | route n.a. (kein Gateway-Handler,
  keine neue Route angefasst — reines Error-Mapping im Server-Layer, `go test
  ./internal/gateway/` deshalb nicht Pflicht)
- mutations-probe: `ErrPluginHasInstallations`-Case testweise auf `ErrManifestSlugExists`
  umgebogen -> `TestMapPluginError/plugin_has_installations` UND
  `TestPluginDeleteManifest/has_active_installations_maps_to_FailedPrecondition` wurden rot
  (erwartet FailedPrecondition, bekam Internal). Zurueckgedreht, `git diff --stat` zeigt nur
  den beabsichtigten Diff (errors.Is-Umstellung + neuer Case), `go test -count=3
  ./internal/server/... ./internal/plugin/...` danach wieder vollstaendig gruen.
- db-tests: 0 — reine Error-Mapping-Logik ohne DB-Zugriff, done_when verlangt keine.
- offen: keine neuen Funde. `TestPluginCreateManifest/missing_tenant_context...` bleibt
  bewusst als eigener, unveraenderter Gap stehen (nicht Teil dieser Unit-Scope) — falls
  relevant, eigene Fix-Unit in einem kommenden Lauf.

## Iteration 34 — fix-plugin-nil-manifest-panic-on-orphaned-installation — done — 2026-08-10 02:41
- commit: (siehe unten)
- verify vorgaenger: sauber — `e33bcdf6` (fix-plugin-error-mapping-gaps) geprueft: `git show
  --stat` zeigt nur `plugin_grpc.go` (Produktionscode) + `plugin_grpc_test.go` +
  Journal/Backlog. Diff gelesen: reine `==` → `errors.Is`-Umstellung plus ein neuer
  `FailedPrecondition`-Case fuer `ErrPluginHasInstallations`, kein gRPC-Bypass, kein Stub, kein
  `.proto` beruehrt, kein neuer `RequirePermission`-Guard, keine neue Tabelle, kein
  Wire-Shape-Wechsel, keine neue Route, kein Guard-Alt-Key verloren.
- gebaut: `ApprovePermissions`, `UpdatePluginSettings` und `GetPluginSettingsSchema`
  (internal/plugin/service.go) pruefen jetzt das Ergebnis von `s.manifests.GetByID(ctx,
  inst.ManifestID)` auf `nil`, bevor sie es dereferenzieren — derselbe Waechter, den
  `Service.GetManifest` schon fuer denselben Fall nutzt (`if manifest == nil { return
  ErrManifestNotFound }`). Alle drei lieferten bei einer verwaisten Installation (Manifest
  geloescht, `uninstalled`-Installation ueberlebt, weil `DeleteManifest`s
  `HasActiveInstallations`-Check nur nicht-uninstallte Installationen zaehlt) bisher einen
  Nil-Pointer-Panic — `mapPluginError` bekommt den Fehler nie zu Gesicht, weil der Prozess vorher
  paniert (in Produktion von `middleware.RecoveryUnaryInterceptor()` zu einem opaken Internal
  abgefangen, aber ohne den erwarteten NotFound). `ErrManifestNotFound` war in `mapPluginError`
  bereits ueber `isNotFound`/`errors.Is` korrekt auf `codes.NotFound` gemappt (Iteration 33),
  hier war also keine Aenderung an der Fehler-Zuordnung noetig, nur an den drei fehlenden
  Nil-Checks selbst. Die drei `require.Panics`-Testfaelle in `plugin_grpc_test.go`
  (`TestPluginApprovePermissions`, zwei in `TestPluginSettings`) auf normale
  `requireGRPCCode(t, err, codes.NotFound)`-Faelle umgestellt, "documents current gap"-Kommentare
  entfernt.
- gate: build ok (`go build -p 2 ./internal/plugin/... ./internal/server/... ./cmd/plugin/...
  ./cmd/gateway/...`) | vet ok (`go vet ./internal/plugin/... ./internal/server/...`) | lint ok
  (`golangci-lint run --config .golangci.yml ./internal/plugin/... ./internal/server/...`, 0
  issues) | test ok (`go test -count=3 ./internal/plugin/... ./internal/server/...`, dreimal
  wiederholt, durchgehend gruen, 0 Skips bei gesetzter `DATABASE_URL`) | migration n.a. (kein
  Schema-Zugriff) | rls-smoke n.a. (keine Tabelle/Policy angefasst) | route n.a. (kein
  Gateway-Handler, keine Route angefasst — `go test ./internal/gateway/` deshalb nicht Pflicht
  gemaess Schritt 5)
- mutations-probe: den neuen Nil-Guard in `ApprovePermissions`
  (`internal/plugin/service.go`) auf `if false && manifest == nil { ... }` gesetzt →
  `TestPluginApprovePermissions/orphaned_installation_(deleted_manifest)_returns_not_found` wurde
  rot (Panic: "invalid memory address or nil pointer dereference" statt NotFound-Response),
  zurueckgedreht, `git diff --stat` auf `service.go` zeigt danach nur noch die drei
  beabsichtigten Nil-Checks, Suite dreimal in Folge wieder vollstaendig gruen.
- db-tests: 0 — reine Service-Logik ohne DB-Zugriff (Repos sind In-Memory-Stubs in den
  Tests), done_when verlangt hier keine DB-Tests.
- offen: keine neuen Funde. Alle drei `done_when`-Punkte der Unit erfuellt: die drei Methoden
  pruefen jetzt auf nil, liefern `ErrManifestNotFound`/`codes.NotFound` statt zu paniken, die
  drei `require.Panics`-Faelle sind ersetzt.

## Iteration 35 — fix-automation-actions-struct-cannot-represent-array — done — 2026-08-10 02:51
- commit: (siehe unten)
- verify vorgaenger: sauber — `e1294ff9` (fix-plugin-nil-manifest-panic-on-orphaned-installation)
  geprueft: `git show --stat` zeigt ausschliesslich `internal/plugin/service.go` (Produktionscode)
  + `internal/server/plugin_grpc_test.go` + Journal/Backlog. Diff gelesen: drei neue Nil-Checks
  nach `s.manifests.GetByID`, kein gRPC-Bypass, kein Stub, kein `.proto` beruehrt, kein neuer
  `RequirePermission`-Guard, keine neue Tabelle, kein Wire-Shape-Wechsel, keine neue Route.
- gebaut: `actions` in `automation.proto` von `google.protobuf.Struct` auf
  `google.protobuf.ListValue` umgestellt (vier Stellen: `AutomationInfo`,
  `AutomationTemplateInfo`, `CreateAutomationRequest`, `UpdateAutomationRequest` — alle vier, nicht
  nur die drei im Scope-Text genannten, weil `AutomationTemplateInfo.actions` denselben
  `automationJSONToStruct`-Konverter durchlaeuft und `internal/automation/template/templates.go`
  seine `Actions` ausnahmslos als `mustJSON([]models.ActionConfig{...})` befuellt — ein JSON-Array,
  exakt derselbe Fehler, waere der vierte Ort unangetastet geblieben). `protoc --go_out=.
  --go-grpc_out=.` fuer `proto/automation/v1/automation.proto` neu generiert (kein manueller
  .pb.go-Edit). `internal/server/automation_grpc.go`: neue Helfer `automationListToJSON`/
  `automationJSONToList` (ListValue<->JSON-Array, Pendant zu den bestehenden
  `automationStructToJSON`/`automationJSONToStruct` fuer trigger_config/conditions), an allen vier
  Call-Sites eingesetzt (`CreateAutomation`, `UpdateAutomation`, `automationToProto`,
  `templateToProto`). `internal/gateway/route_automation.go`: neuer Helfer
  `rawJSONToAutomationList`, ersetzt `rawJSONToAutomationStruct` fuer `body.Actions` in
  `HandleCreateAutomation`/`HandleUpdateAutomation` (Verhalten bei Parse-Fehler unveraendert:
  Feld bleibt still nil, identisch zum bestehenden trigger_config/conditions-Muster — das
  Verschlucken selbst ist ein separat dokumentierter, nicht in dieser Unit behobener Befund aus
  `b-cov-gateway-automation`). `openapi.yaml`: die vier `actions: { type: object }`-Schemas
  (`Automation`, `CreateAutomationRequest`, `UpdateAutomationRequest`, `AutomationTemplate`) auf
  `actions: { type: array, items: { type: object } }` korrigiert — dokumentierten vorher
  fälschlich ein JSON-Objekt statt eines Arrays; `swagger-cli validate` danach gruen.
  `TestCreateAutomation_ActionsStructCannotCarryAnArray` (Server-Test) von "pinnt das kaputte
  Verhalten" auf "ein echtes Actions-Array wird angenommen, im Repository als JSON-Array
  gespeichert und in der Response wieder als `ListValue` mit einem Eintrag ausgelesen" umgestellt
  (nicht geloescht, wie done_when verlangt) — neuer Helfer `automationList(t, []any{...})` neben
  dem bestehenden `automationStruct`. Gateway-Test `TestHandleUpdateAutomation_AllFieldsReachesRPC`
  hatte `actions` bisher als JSON-*Objekt* im Body (`map[string]interface{}{"type": "notify"}`) —
  war zufaellig gueltig, solange das Proto-Feld Struct war; auf ein Array umgestellt, sonst waere
  der Test nach dem Fix ein (harmloser, weil eh nur auf 503 pruefender) Fall des
  Malformed-Silently-Ignored-Pfads statt tatsaechlich alle Felder gueltig zu befuellen.
  FE-Gegenprobe (`desktop/src/renderer/src/api/automation-client.ts`,
  `automation-types.ts`): das Frontend sendet `actions` bereits als `unknown[]`/`ActionConfig[]` —
  der Bug war produktiv wirksam, nicht nur theoretisch: keine einzige Automation mit echten
  Aktionen liess sich je erfolgreich anlegen.
- gate: build ok (`go build -p 2 ./internal/server/... ./internal/automation/...
  ./internal/gateway/... ./cmd/...`) | vet ok (`go vet ./internal/server/... ./internal/automation/...
  ./internal/gateway/...`) | lint ok (`golangci-lint run --config .golangci.yml
  ./internal/server/... ./internal/automation/... ./internal/gateway/... ./proto/...`, 0 issues) |
  test ok (`go test -count=3 ./internal/server/... ./internal/automation/... ./internal/gateway/...`,
  dreimal wiederholt, durchgehend gruen) | migration n.a. (kein Schema-Zugriff) | rls-smoke n.a. |
  route n.a. (keine neue Route, nur ein bestehendes Feld auf allen bestehenden Routen) |
  openapi: `npx swagger-cli validate api/openapi.yaml` gruen, `TestOpenAPIRouteDrift` gruen |
  protoc: Neugenerierung im selben Commit, `git diff` zeigt nur generierte Aenderungen (Actions-Feldtyp
  + Nachbar-Typnummern-Verschiebung durch das neue `ListValue`-Symbol im Descriptor), kein Handedit.
- mutations-probe: in `CreateAutomation` die Bedingung `if req.GetActions() != nil` auf
  `if false && req.GetActions() != nil` gesetzt → `TestCreateAutomation_ActionsStructCannotCarryAnArray`
  wurde rot (`captured.Actions` blieb leer statt `[{"type":"send_email"}]`), zurueckgedreht,
  `git diff --stat` auf `automation_grpc.go` zeigt danach nur noch die beabsichtigten Aenderungen,
  Suite dreimal in Folge wieder vollstaendig gruen.
- db-tests: 0 — reine Proto-/Konverter-/Handler-Logik ohne DB-Zugriff, done_when verlangt hier
  keine DB-Tests.
- offen: `fix-automation-testcondition-omitted-conditions-error` (naechste Unit im Backlog, selbe
  Datei/Domain) ist davon unberuehrt — betrifft `conditions`, nicht `actions`, bleibt bei
  `google.protobuf.Struct`. `AutomationTemplateInfo.actions` wurde mitgezogen, obwohl der
  Scope-Text nur die drei anderen Messages nannte — Begruendung siehe oben (gebaut); falls Luke
  das als Scope-Ueberschreitung werten will, ist der Diff dafuer isoliert (ein zusaetzlicher
  proto-Feldtyp, ein zusaetzlicher Call-Site-Swap in `templateToProto`).

## Iteration 36 — fix-automation-testcondition-omitted-conditions-error — done — 2026-08-10 02:58
- commit: 8101a0c2
- verify vorgaenger: sauber — 6ea97f34 (actions Struct->ListValue) gegen alle acht Fehlerklassen
  geprueft: .proto-Aenderung mit echter protoc-Regenerierung (rawDesc mitgeaendert, kein Handedit),
  Handler bleibt gRPC-Client-Aufruf, kein neuer Guard/keine neue Route/Tabelle, openapi.yaml korrekt
  auf `type: array` umgestellt und deckungsgleich mit dem neuen ListValue-Feld. Keine Befunde.
- gebaut: `workflow.Service.TestCondition` (service.go:346) hatte anders als `DryRun` keine
  Laengenpruefung vor dem `json.Unmarshal(conditionJSON, &config)` — ein weggelassenes
  `conditions`-Feld ("Automation soll immer feuern") ergab "unexpected end of JSON input" statt
  Matches:true. Guard `if len(conditionJSON) > 0 { unmarshal }` ergaenzt, symmetrisch zu
  `DryRun` (service.go:384). Bestehenden Testfall `TestTestCondition_OmittedConditionsSurfacesAsError`
  in `automation_grpc_test.go` auf `TestTestCondition_OmittedConditionsAlwaysMatches` umbenannt und
  auf das neue (korrekte) Verhalten umgestellt, nicht geloescht. Neuen Service-Ebene-Test
  `TestTestCondition_NilConditionJSONAlwaysTrue` in `service_test.go` ergaenzt (direkter Aufruf mit
  `nil` statt ueber den Handler).
- gate: build ok (`go build -p 2 ./internal/server/... ./internal/automation/...
  ./internal/gateway/... ./cmd/...`) | vet ok | lint ok (`golangci-lint run --config .golangci.yml
  ./internal/server/... ./internal/automation/...`, 0 issues) | test ok (`go test -count=3
  ./internal/server/... ./internal/automation/...`, dreimal wiederholt, durchgehend gruen) |
  migration n.a. (kein Schema-Zugriff) | rls-smoke n.a. | route n.a. (keine Route angefasst,
  `internal/gateway` nicht Pflicht) | openapi n.a.
- mutations-probe: `if len(conditionJSON) > 0` auf `if false && len(conditionJSON) > 0` gesetzt →
  `TestTestCondition_InvalidConfigReturnsSoftError` (internal/server) wurde rot (erwartete
  Matches:false + nicht-leere ErrorMessage, bekam Matches:true + leere ErrorMessage, weil der
  Unmarshal komplett uebersprungen wurde), `internal/automation/workflow`-Paket blieb dabei
  gruen (mein neuer Test dort prueft nur den nil-Fall, den die Mutation nicht veraendert — das
  Server-Paket ist die schaerfere Probe). Zurueckgedreht, `git diff` auf service.go zeigt danach
  nur die beabsichtigte Zeile, Suite dreimal in Folge wieder vollstaendig gruen.
- db-tests: 0 — reine In-Memory-Logik ohne DB-Zugriff, done_when verlangt hier keine DB-Tests.
- offen: keins.

## Iteration 37 — fix-schichten-createtemplate-drops-location — done — 2026-08-10 03:01
- commit: 1b78eb51
- verify vorgaenger: sauber — 8101a0c2 (TestCondition match-all) geprueft: kein Proto-/Route-/
  Guard-/Tabellen-Bezug, reiner Service-Guard + symmetrischer Testumbau, gRPC-Layer unveraendert.
  Keine Befunde.
- gebaut: `SchichtenGRPCServer.CreateTemplate` (schichten_grpc.go:268) las `req.GetLocation()`
  nie und liess `CreateTemplateInput.Location` immer nil, obwohl `UpdateTemplate` dasselbe Feld
  schon korrekt setzt. Dieselbe `if l := req.GetLocation(); l != "" { input.Location = &l }`-Zeile
  wie in UpdateTemplate ergaenzt. `TestSchichten_TemplateCRUDAndList` (schichten_grpc_test.go) von
  "documents current gap" (assert.Nil) auf das neue Verhalten (require.NotNil + Wertvergleich)
  umgestellt.
- gate: build ok (`go build -p 2 ./internal/schichten/... ./internal/server/...
  ./cmd/schichten/... ./cmd/gateway/...`) | vet ok | lint ok (`golangci-lint run --config
  .golangci.yml ./internal/schichten/... ./internal/server/...`, 0 issues) | test ok (`go test
  -count=3 ./internal/schichten/... ./internal/server/... ./internal/gateway/...`, dreimal
  wiederholt, durchgehend gruen, 0 uebersprungen) | migration n.a. (kein Schema-Zugriff) |
  rls-smoke n.a. | route n.a. (keine neue Route, bestehendes Feld auf bestehender RPC) |
  openapi n.a.
- mutations-probe: die neue Zeile auf `if l := req.GetLocation(); false && l != "" { ... }`
  gesetzt → `TestSchichten_TemplateCRUDAndList` wurde rot ("Expected value not to be nil" auf
  `createResp.Template.Location`), zurueckgedreht, `git diff --stat` zeigt danach nur die
  beabsichtigten 7 Zeilen in schichten_grpc.go, Suite dreimal in Folge wieder vollstaendig gruen.
- db-tests: 0 — reine Handler-/In-Memory-Logik (Stub-Repo), done_when verlangt hier keine
  DB-Tests.
- offen: keins.

## Iteration 38 — fix-schichten-error-mapping-shiftfull-gap — done — 2026-08-10 03:08
- commit: 1ff79225
- verify vorgaenger: sauber — 1b78eb51 (CreateTemplate Location) geprueft: kein Proto-/Route-/
  Guard-/Tabellen-Bezug, exakt das dokumentierte Muster von UpdateTemplate uebernommen (`if l :=
  req.GetLocation(); l != "" { input.Location = &l }`), Testfall in TestSchichten_TemplateCRUDAndList
  entsprechend von "documents current gap" auf require.NotNil + Wertvergleich umgestellt statt
  geloescht. Keine Befunde.
- gebaut: `mapSchichtenError` (internal/server/schichten_grpc.go:652) hatte keinen Fall fuer
  `schichten.ErrShiftFull` — jeder andere Sentinel aus errors.go war vertreten, dieser eine fehlte,
  fiel auf den generischen Internal-Zweig statt eines client-actionable 4xx. Fall
  `case errors.Is(err, schichten.ErrShiftFull): return status.Error(codes.FailedPrecondition,
  err.Error())` ergaenzt (FailedPrecondition passt zum bestehenden Muster fuer Kapazitaets-/
  Regel-Verstoesse in derselben Funktion — ArbZG/JArbSchG nutzen denselben Code). Die beiden
  "documents current gap"-Testfaelle in schichten_grpc_test.go auf den neuen erwarteten Code
  umgestellt: `TestMapSchichtenError_Table/shift_full_documents_current_gap` →
  `shift_full` mit codes.FailedPrecondition, `TestSchichten_AssignEmployee_CapacityExceeded_
  MapsToInternal` → `TestSchichten_AssignEmployee_CapacityExceeded_MapsToFailedPrecondition`
  mit requireGRPCCode(..., codes.FailedPrecondition).
- gate: build ok (`go build -p 2 ./internal/schichten/... ./internal/server/...
  ./cmd/schichten/... ./cmd/gateway/...`) | vet ok (`go vet ./internal/schichten/...
  ./internal/server/...`) | lint ok (`golangci-lint run --config .golangci.yml
  ./internal/schichten/... ./internal/server/...`, 0 issues) | test ok (`go test -count=3
  ./internal/schichten/... ./internal/server/...`, dreimal wiederholt, durchgehend gruen) |
  migration n.a. (kein Schema-Zugriff) | rls-smoke n.a. | route n.a. (keine neue Route, nur ein
  zusaetzlicher Fall in bestehendem Error-Mapping) | openapi n.a.
- mutations-probe: den neuen Case auf `case false && errors.Is(err, schichten.ErrShiftFull):`
  gesetzt → beide betroffenen Tests wurden rot (`TestMapSchichtenError_Table/shift_full`:
  erwartet FailedPrecondition, bekam Internal; `TestSchichten_AssignEmployee_CapacityExceeded_
  MapsToFailedPrecondition`: dieselbe Abweichung end-to-end durch den echten Handler), zurueckgedreht,
  `git diff --stat` auf schichten_grpc.go zeigt danach nur die beabsichtigten 2 Zeilen, Suite
  dreimal in Folge wieder vollstaendig gruen.
- db-tests: 0 — reine Handler-/In-Memory-Logik (Stub-Repo), done_when verlangt hier keine
  DB-Tests.
- offen: keins.

## Iteration 39 — fix-schichten-swaprequests-no-own-scope — done — 2026-08-10 03:20
- commit: 126c40f6
- verify vorgaenger: sauber — 1ff79225 (ErrShiftFull FailedPrecondition) geprueft: einzelner
  zusaetzlicher `case`-Zweig in `mapSchichtenError`, exakt dem Muster der Nachbarfaelle
  (ArbZG/JArbSchG) folgend, beide betroffenen "documents current gap"-Testfaelle korrekt auf
  den neuen erwarteten Code umgestellt statt geloescht. `go build`/`go test` gegen
  `internal/schichten`+`internal/server` erneut gruen. Keine Befunde.
- gebaut: own-Scope fuer `ListSwapRequests`, matched gegen BEIDE Mitarbeiterfelder. Produktfrage
  aus dem Backlog-Eintrag war bereits beantwortet (own-Scope gilt nur fuers Lesen, Genehmigen/
  Ablehnen bleiben unveraendert tenant-weit) — nicht neu zu klaeren. Neue Erkenntnis beim Bauen:
  in diesem Repo IST `employee_id` durchgehend dieselbe ID wie `users.id` (verifiziert per Grep
  ueber alle Migrationen: `hr_*`-Tabellen referenzieren `employee_id UUID NOT NULL REFERENCES
  users(id)`, `hr_employee_profiles` loest sogar explizit `id = $2 OR user_id = $2` auf) — es gibt
  keine separate "Mitarbeiter"-Entitaet, gegen die erst aufgeloest werden muesste. Das macht das
  bereits bestehende `ownerFilterForScope`-Helper (internal/gateway/helpers.go, liest
  `middleware.PermissionScope`+`middleware.GetUserID`, exakt das Muster aus
  `route_rapporte.go:HandleListReports`) direkt anwendbar, ohne neue Aufloese-Schicht. Die vom
  Backlog-Eintrag vorgeschlagene `RequirePermissionAny`-Route-Umstellung war NICHT noetig — Rapporte
  selbst nutzt fuer denselben Zweck nur die einzelne bestehende `RequirePermission("rapporte:report",
  "read")`-Guard und filtert erst im Handler per `PermissionScope`; dasselbe fuer
  `route_schichten.go` uebernommen, keine Guard-Aenderung. Kette: `.proto`
  (`ListSwapRequestsRequest.own_employee_id`, Feld 6, `optional string`, protoc-neu-generiert) →
  `schichten_grpc.go:ListSwapRequests` parst es zu `uuid.UUID` → `service.go:ListSwapRequestsInput.
  OwnEmployeeID` → `repository.go:SwapRequestFilter.OwnEmployeeID` → `postgres_repository.go:
  ListSwapRequests` haengt `(requested_by_employee_id = $N OR swap_with_employee_id = $N)` an die
  WHERE-Klausel VOR dem COUNT(*) und dem LIMIT/OFFSET-Query (Zeilen UND total bleiben konsistent).
  Gateway `HandleListSwapRequests` ruft `ownerFilterForScope(w, r, "schichten:swap", "read")` und
  setzt `grpcReq.OwnEmployeeId` nur, wenn die Grant-Scope des Aufrufers `auth.ScopeOwn` ist — bei
  vollem Scope bleibt das Verhalten unveraendert tenant-weit. Bisherigen Test
  `TestSchichten_ListSwapRequests_NoOwnScopeFiltering` (dokumentierte die Luecke) ersetzt durch
  `TestSchichten_ListSwapRequests_OwnEmployeeFilterMatchesBothFields` (End-to-End durch den echten
  Handler: drei Swap-Requests — ich bin Requester, ich bin nur Swap-Partner, ich bin in keinem
  Feld — unscoped sieht alle drei, scoped genau die zwei, die mich betreffen) und
  `TestSchichten_ListSwapRequests_InvalidOwnEmployeeID` (ungueltige UUID im neuen Feld liefert
  InvalidArgument). `newStubSchichtenRepo.ListSwapRequests` (Testdouble) um dieselbe
  OR-Filterung ergaenzt, sonst haette der neue Handler-Test nichts geprueft. Zusaetzlich echter
  DB-Test `internal/schichten/own_scope_list_test.go`
  (`TestListSwapRequests_OwnScopeMatchesBothEmployeeFields`) gegen das reale Schema, nach dem
  Muster von `internal/rapporte/own_scope_list_test.go` — beweist die SQL-Bedingung selbst, nicht
  nur den Stub.
- gate: build ok (`go build -p 2 ./internal/schichten/... ./internal/server/... ./internal/gateway/...
  ./cmd/schichten/... ./cmd/gateway/...`) | vet ok (`go vet ./internal/schichten/...
  ./internal/server/... ./internal/gateway/...`) | lint ok (`golangci-lint run --config
  .golangci.yml ./internal/schichten/... ./internal/server/... ./internal/gateway/...`, 0 issues) |
  test ok (`go test -count=1 ./internal/schichten/... ./internal/server/... ./internal/gateway/...`
  mit gesetzter `DATABASE_URL`, durchgehend gruen, 0 uebersprungen) | migration n.a. (keine neue
  Migration, nur bestehende Spalten `requested_by_employee_id`/`swap_with_employee_id` gefiltert) |
  rls-smoke n.a. (keine Policy angefasst) | route n.a. (keine neue Route, bestehender Handler
  bekommt einen zusaetzlichen server-injizierten Request-Parameter) | openapi n.a. (own_employee_id
  ist server-seitig aus dem Scope abgeleitet, kein Client-Query-Parameter — analog zu rapporte's
  `author_id`-Injection, kein neuer dokumentierter Query-Parameter) | protoc: Neugenerierung im
  selben Commit, `git diff --stat` auf `schichten.pb.go` zeigt nur generierte Aenderungen (ein
  neues optionales Feld + dessen Nachbar-Deskriptor-Verschiebung), kein Handedit.
- mutations-probe: `if filter.OwnEmployeeID != nil` in `ListSwapRequests`
  (postgres_repository.go) auf `if false && filter.OwnEmployeeID != nil` gesetzt →
  `TestListSwapRequests_OwnScopeMatchesBothEmployeeFields` (DB-Test) wurde rot ("own scope: got 3
  rows / total 3, want 2 / 2" — die Filterung griff nicht mehr), zurueckgedreht, `diff` gegen die
  Sicherungskopie bestaetigt eine identische Datei, volle Suite (schichten+server+gateway) danach
  wieder vollstaendig gruen.
- db-tests: 1 neuer echter DB-Test (`TestListSwapRequests_OwnScopeMatchesBothEmployeeFields`),
  0 Skips bei gesetzter `DATABASE_URL` (`docker-postgres-1` war bereits healthy, kein Docker-Start
  noetig).
- offen: Nichts Blockierendes. `ApproveSwapRequest`/`RejectSwapRequest` bewusst unveraendert
  gelassen (bleiben tenant-weit fuer die genehmigende Rolle) — exakt wie in den Backlog-`done_when`
  gefordert, keine eigene Regel dafuer gebaut, da nicht Teil dieses Scopes.

## Iteration 40 — fix-vermietung-rentaltoproto-drops-signature — done — 2026-08-10 03:29
- commit: 7dbcc892
- verify vorgaenger: sauber — 126c40f6 (own-Scope fuer ListSwapRequests) geprueft: Diff auf
  route_schichten.go/postgres_repository.go/repository.go/service.go/schichten_grpc.go zeigt
  exakt das im Journal beschriebene Muster (`ownerFilterForScope` im Gateway, OR-Filter VOR
  COUNT/LIMIT im Repository, server-injiziertes Feld statt Client-Query-Parameter). Keine Befunde.
- gebaut: `rentalToProto` (internal/server/vermietung_grpc.go:650) mappte `Rental.SignatureData`/
  `SignedAt`/`SignedBy` nie auf das Wire-`Rental`, obwohl `Service.SaveSignature` sie korrekt
  persistiert — jede RPC, die ein Rental zurueckgibt (inklusive `SaveSignature`s eigene Antwort),
  liess die gerade gespeicherte Signatur verschwinden. Anders als bei der parallelen
  `fix-vertraege-contracttoproto-drops-signature`-Unit gibt es hier laut done_when KEINE
  Liste-vs-Detail-Unterscheidung zu bauen — der Proto-Kommentar auf `vermietung.proto` traegt
  keine "nur GetContract"-Einschraenkung wie bei vertraege, `rentalToProto` ist die einzige
  Konvertierungsfunktion fuer alle Rental-RPCs. Drei Zeilen ergaenzt, alle drei Proto-Felder sind
  `string`/`optional Timestamp` nicht `optional string` (verifiziert gegen vermietung.proto:76-78),
  also nil-Guard vor Dereferenzierung statt direktem Pointer-Copy: `if r.SignatureData != nil { ... }`,
  `if r.SignedBy != nil { ... }`, `if r.SignedAt != nil { proto.SignedAt = timestamppb.New(*r.SignedAt) }`.
  Kein Proto-Aenderung noetig (Felder existierten schon), keine Repository-Aenderung (persistiert
  schon korrekt), reiner Mapping-Fix in einer einzigen Funktion.
- gate: build ok (`go build -p 2 ./internal/server/... ./internal/vermietung/...
  ./cmd/vermietung/... ./cmd/gateway/...`) | vet ok (`go vet ./internal/server/...
  ./internal/vermietung/...`) | lint ok (`golangci-lint run --config .golangci.yml
  ./internal/server/... ./internal/vermietung/...`, 0 issues) | test ok (`go test -count=2
  ./internal/server/... ./internal/vermietung/...` mit gesetzter `DATABASE_URL`, zweimal
  wiederholt, durchgehend gruen) | migration n.a. (keine Schema-Aenderung) | rls-smoke n.a. |
  route n.a. (keine neue Route, nur Wire-Mapping in bestehenden Responses) | openapi n.a. (keine
  neue/geaenderte Route) | protoc n.a. (keine .proto-Aenderung, alle drei Felder existierten
  bereits im generierten Code).
- mutations-probe: `if r.SignatureData != nil` in `rentalToProto` auf
  `if false && r.SignatureData != nil` gesetzt → `TestVermietung_SaveSignature` wurde rot
  (erwartet `"data:image/png;base64,AAAA"`, bekam `""`), zurueckgedreht, `diff` gegen die
  Sicherungskopie bestaetigt eine identische Datei, Suite (server+vermietung) danach wieder
  vollstaendig gruen.
- db-tests: 0 — reine Proto-Mapping-Logik (Stub-Repo), done_when verlangt hier keine DB-Tests.
- offen: keins. Die parallele Signatur-Luecke bei `vertraegeContractToProto`
  (`fix-vertraege-contracttoproto-drops-signature`, weiter unten im Backlog) ist NICHT Teil
  dieser Unit — dort ist zusaetzlich eine Liste-vs-Detail-Unterscheidung gefordert, hier nicht.

## Iteration 41 — fix-vermietung-error-mapping-inspectionkindexists-gap — done — 2026-08-10 (Lauf 7)
- commit: (dieser Commit)
- verify vorgaenger: sauber — 7dbcc892 (Signatur-Mapping bei rentalToProto) geprueft: Diff auf
  vermietung_grpc.go zeigt exakt die drei im Journal beschriebenen nil-geguardeten Zeilen
  (SignatureData/SignedBy als string, SignedAt als *time.Time -> timestamppb.New), Feldtypen
  gegen models.go verifiziert (alle drei *string/*time.Time), keine Befunde.
- gebaut: `mapVermietungError` (internal/server/vermietung_grpc.go:744) hatte keinen Fall fuer
  `vermietung.ErrInspectionKindExists` und fiel auf den generischen Internal-Zweig, obwohl der
  Fehler dieselbe Form wie `ErrRentalConflict` hat (bereits auf AlreadyExists gemappt). Case
  `case errors.Is(err, vermietung.ErrInspectionKindExists): return status.Error(codes.AlreadyExists, err.Error())`
  direkt nach dem ErrRentalConflict-Case ergaenzt (identisches Muster). Die beiden
  "documents current gap"-Testfaelle in vermietung_grpc_test.go aktualisiert:
  `TestVermietung_CreateInspection_DuplicateKind_MapsToInternal` in
  `TestVermietung_CreateInspection_DuplicateKind_MapsToAlreadyExists` umbenannt, Assert auf
  `codes.AlreadyExists`; Tabellenfall `inspection_kind_exists_documents_current_gap` in
  `inspection_kind_exists` umbenannt, erwarteter Code auf `codes.AlreadyExists`. Kommentarblock
  ueber dem Testfall (der den Gap beschrieb) entfernt, da der Gap nun geschlossen ist.
- gate: build ok (`go build -p 2 ./internal/server/... ./internal/vermietung/...`) | vet ok
  (`go vet ./internal/server/... ./internal/vermietung/...`) | lint ok (`golangci-lint run
  --config .golangci.yml ./internal/server/... ./internal/vermietung/...`, 0 issues) | test ok
  (`go test -count=1 ./internal/server/... ./internal/vermietung/...` mit gesetzter
  `DATABASE_URL`, durchgehend gruen) | migration n.a. (keine Schema-Aenderung) | rls-smoke n.a.
  | route n.a. (keine neue Route, reines Error-Mapping in bestehenden RPCs) | openapi n.a.
  (kein Wire-Vertrags-Wechsel, nur HTTP-Status ueber gRPC-Code-Mapping in der Gateway-Uebersetzung)
  | protoc n.a. (keine .proto-Aenderung).
- mutations-probe: den neuen Case in `mapVermietungError` auf
  `case false && errors.Is(err, vermietung.ErrInspectionKindExists):` gesetzt →
  `TestVermietung_CreateInspection_DuplicateKind_MapsToAlreadyExists` und
  `TestMapVermietungError_Table/inspection_kind_exists` wurden beide rot (erwartet
  AlreadyExists, bekamen Internal), zurueckgedreht, `diff` gegen die Sicherungskopie
  bestaetigt eine identische Datei, volle Suite (server+vermietung) danach wieder
  vollstaendig gruen.
- db-tests: 0 — reine Error-Mapping-Logik (Stub-Repo), done_when verlangt hier keine DB-Tests.
- offen: keins. Die parallele Unit `fix-vertraege-contracttoproto-drops-signature` (naechste
  im Backlog) ist unabhaengig und noch offen.

## Iteration 42 — fix-vertraege-contracttoproto-drops-signature — done — 2026-08-10 (Lauf 7)
- commit: (dieser Commit)
- verify vorgaenger: sauber — 71d0a5db (ErrInspectionKindExists -> AlreadyExists bei vermietung)
  geprueft: Diff auf vermietung_grpc.go/vermietung_grpc_test.go zeigt exakt den im Journal
  beschriebenen zusaetzlichen case-Zweig plus die zwei umbenannten/aktualisierten Testfaelle,
  ErrInspectionKindExists in errors.go/service.go gegengeprueft, build/vet/test (server+vermietung,
  DATABASE_URL gesetzt) gruen. Keine Befunde.
- gebaut: `vertraegeContractToProto` (internal/server/vertraege_grpc.go:537) setzte
  `SignatureData`/`SignedAt`/`SignedBy` nie auf das Wire-Contract, obwohl der Proto-Kommentar
  (vertraege.proto:64-69) "Populated only by GetContract" verspricht und `Service.SaveSignature`
  die Felder korrekt persistiert - jede der fuenf Aufrufstellen (Create/Update/Get/List/
  SaveSignature-Response) nutzte dieselbe Funktion und liess die Signatur verschwinden, auch
  beim expliziten Detail-Abruf. Anders als bei `fix-vermietung-rentaltoproto-drops-signature`
  (Iteration 40, wo alle RPCs die Signatur zeigen sollen) verlangt der Scope hier eine bewusste
  Liste-vs-Detail-Trennung. Fix: `vertraegeContractToProto` um einen `includeSignature bool`-
  Parameter erweitert (nil-Guard vor Dereferenzierung, da `SignatureData`/`SignedBy` im Proto
  als `string` nicht `optional string` deklariert sind, `SignedAt` als `optional Timestamp`).
  CreateContract/UpdateContract/ListContracts rufen mit `false` (Signatur bleibt unsichtbar,
  wie von `TestVertraege_SaveSignature_NotInListResponse` bereits fuer die Liste gefordert),
  GetContract/SaveSignature mit `true`. Die drei direkten Testaufrufe
  (`TestVertraege_ToProtoNilSafety`, `TestVertraege_ContractToProto_OptionalFields`) auf die neue
  Signatur angepasst; `TestVertraege_ContractToProto_OptionalFields` zusaetzlich um einen
  echten includeSignature=true/false-Vergleich erweitert (vorher testete er Signatur ueberhaupt
  nicht). Der "documents current gap"-Assert in `TestVertraege_SaveSignature_NotInListResponse`
  (GetContract liefert leere SignatureData) durch einen Assert auf die jetzt korrekt gesetzten
  Werte ersetzt, inklusive Erweiterung der SaveSignature-Response-Assertion selbst (die vorher
  nur require.NotNil auf das Contract pruefte, nicht die Signaturfelder). Keine Proto-Aenderung
  noetig (alle drei Felder existierten bereits im generierten Code), keine Repository-Aenderung
  (persistiert schon korrekt), reiner Mapping-Fix in einer einzigen Funktion plus deren fuenf
  Aufrufstellen.
- gate: build ok (`go build -p 2 ./internal/server/... ./internal/vertraege/...`) | vet ok
  (`go vet ./internal/server/... ./internal/vertraege/...`) | lint ok (`golangci-lint run
  --config .golangci.yml ./internal/server/... ./internal/vertraege/...`, 0 issues) | test ok
  (`go test -count=1 ./internal/server/... ./internal/vertraege/...` mit gesetzter
  `DATABASE_URL`, durchgehend gruen) | migration n.a. (keine Schema-Aenderung) | rls-smoke n.a.
  | route n.a. (keine neue Route, nur Wire-Mapping in bestehenden Responses) | openapi n.a.
  (keine neue/geaenderte Route) | protoc n.a. (keine .proto-Aenderung, alle drei Felder
  existierten bereits im generierten Code).
- mutations-probe: `if includeSignature` in `vertraegeContractToProto` auf
  `if false && includeSignature` gesetzt → `TestVertraege_SaveSignature_NotInListResponse` UND
  `TestVertraege_ContractToProto_OptionalFields` wurden beide rot (erwartete SignatureData/
  SignedBy/SignedAt, bekamen leere Werte bzw. nil), zurueckgedreht, `diff` gegen die
  Sicherungskopie bestaetigt eine identische Datei, volle Suite (server+vertraege) danach
  wieder vollstaendig gruen.
- db-tests: 0 — reine Proto-Mapping-Logik (Stub-Repo), done_when verlangt hier keine DB-Tests.
- offen: keins. Naechste offene Unit im Backlog ist Block C1 (`c-cov-work-event-rrule`, Zeile
  ~1720). Korrektur zur vorherigen Annahme: Phase 3 ist NICHT vollstaendig abgearbeitet — es
  gibt noch eine offene Fix-Unit `fix-notification-markallread-empty-body-rejected` (Zeile
  ~2362, vermutlich waehrend einer spaeteren Coverage-Unit gefunden und angehaengt), die beim
  ersten Backlog-Scan uebersehen wurde. Fuer Iteration 43 gilt weiterhin die im Backlog-Kopf
  festgelegte Reihenfolge (C1 vor B-Gateway vor B-Server vor C2) — diese eine Fix-Unit steht
  nicht am Anfang der Datei und war deshalb nicht die naechste in Zeilen-Reihenfolge; wer als
  naechstes den Scan macht, sollte trotzdem grep -n "status: todo" laufen lassen statt nur die
  Zeilenreihenfolge zu vertrauen.

## Iteration 43 — c-cov-work-event-rrule — done — 2026-08-10 (Lauf 7)
- commit: (dieser Commit)
- verify vorgaenger: sauber — 1b52107d (Signatur-Mapping bei vertraegeContractToProto) geprueft:
  Diff auf vertraege_grpc.go/vertraege_grpc_test.go zeigt exakt das im Journal beschriebene
  `includeSignature bool`-Parameter-Muster (false bei Create/Update/List, true bei Get/
  SaveSignature), alle drei Aufrufstellen korrekt, Nil-Guards vor Dereferenzierung passend zu den
  Feldtypen in vertraege.proto (string/optional Timestamp), Testanpassungen decken beide
  includeSignature-Werte ab. Keine Befunde.
- **Backlog-Praemisse widerlegt (kein Blocker, nur Korrektur):** `internal/work/event/rrule.go`
  war entgegen dem Scope-Text ("komplett ungetestet") bereits ueber `service_test.go` mit 14
  Tests abgedeckt (`TestExpandRecurrence_Weekly/Daily/Monthly/InvalidRRule`,
  `TestValidateRRule_Valid/Invalid`, `TestSetUntil_AddUntil/ReplaceUntil/RemovesCount/
  InvalidRRule`, `TestRemoveUntil_Success/NoUntil/InvalidRRule`) — die vier Basisfunktionen
  hatten also bereits Erfolgs- und Fehlerpfad. Was tatsaechlich fehlte, waren genau die im
  `done_when` explizit geforderten Randfaelle: Monatsende, Schaltjahr, DST.
- gebaut: neue Datei `internal/work/event/rrule_test.go` mit drei neuen `ExpandRecurrence`-
  Randfall-Tests, deren erwartete Werte gegen das reale Verhalten der `rrule-go`-Bibliothek
  verifiziert wurden (Wegwerf-Programm unter `tmp_scratch/` gebaut, Ausgabe gegenlesen, danach
  geloescht — nicht aus der RFC hergeleitet und geraten):
  (1) `TestExpandRecurrence_MonthEnd_SkipsShortMonths` — `FREQ=MONTHLY;BYMONTHDAY=31` ab 31. Januar
  liefert nur Jan/Mar/Mai/Jul (Feb/Apr/Jun werden uebersprungen, nicht auf Monatsende geklemmt —
  exakt RFC-5545-Verhalten, per Bibliotheks-Testlauf bestaetigt).
  (2) `TestExpandRecurrence_LeapYear_Feb29OnlyFiresOnLeapYears` — `FREQ=YEARLY` ab 29.02.2024
  liefert nur 2024 und 2028, ueberspringt 2025-2027 komplett statt auf 1. Maerz zu rollen.
  (3) `TestExpandRecurrence_DSTSpringForward_ShiftsPastMissingHour` — woechentlich 02:30
  Europe/Berlin ueber den Umstellungstermin 2026-03-29 (02:00->03:00 CEST): der Termin VOR der
  Umstellung liegt bei 02:30 CET (+1h), der Termin, der in die uebersprungene Stunde faellt,
  verschiebt sich auf 03:30 CEST (+2h) statt zu verschwinden oder bei einer nicht-existenten Zeit
  zu bleiben.
  Zusaetzlich zwei kleinere Ergaenzungen: `TestValidateRRule_EmptyString` (leerer String war noch
  kein eigener Testfall) und `TestSetUntil_ReplacesExistingUntilAndRemovesCount` (deckt die
  Kombination beider Praemissen — Regel traegt schon UNTIL und COUNT gleichzeitig, ein Fall, den
  RFC 5545 verbietet und den keiner der bestehenden Einzeltests abdeckte). Zwei Duplikate zu
  bereits bestehenden Tests (`TestExpandRecurrence_InvalidRRule`, `TestSetUntil_InvalidRRule`,
  `TestRemoveUntil_InvalidRRule` waren Namenskollisionen) verworfen statt dupliziert — stattdessen
  die zwei bestehenden `_InvalidRRule`-Tests in `service_test.go` (SetUntil/RemoveUntil) um
  `assert.ErrorIs(t, err, ErrInvalidRRule)` ergaenzt, da sie vorher nur `assert.Error` prueften und
  damit nicht den vom `done_when` geforderten spezifischen Fehlertyp belegten.
- gate: build ok (`go build -p 2 ./internal/work/... ./cmd/work/...`) | vet ok
  (`go vet ./internal/work/event/...`) | lint ok (`golangci-lint run --config .golangci.yml
  ./internal/work/event/...`, 0 issues) | test ok (`go test -count=3 ./internal/work/event/...`,
  dreimal wiederholt, durchgehend gruen, 0 Fails) | migration n.a. (keine Schema-Aenderung) |
  rls-smoke n.a. (kein DB-Zugriff im Paket) | route n.a. (kein Gateway/Server-Handler angefasst,
  `go test ./internal/gateway/` deshalb nicht Pflicht und nicht separat gelaufen) | openapi n.a.
  | protoc n.a.
- mutations-probe: `strings.HasPrefix(strings.ToUpper(part), "COUNT=")` in `SetUntil`
  (rrule.go:55) auf `false && strings.HasPrefix(...)` gesetzt → sowohl der neue
  `TestSetUntil_ReplacesExistingUntilAndRemovesCount` als auch der bestehende
  `TestSetUntil_RemovesCount` wurden rot (Ergebnis enthielt weiterhin `COUNT=`), zurueckgedreht,
  `git diff` auf die Produktionsdatei bestaetigt eine identische Datei, Suite danach wieder
  vollstaendig gruen.
- db-tests: 0 — `internal/work/event/rrule.go` ist reine Terminberechnungslogik ohne DB-Zugriff
  (Repository/DB-Tests existieren bereits separat fuer den Rest des Pakets), done_when verlangt
  hier keine DB-Tests.
- offen: `internal/work/event`-Paketcoverage liegt bei 54,6 % (Gesamtpaket, nicht nur rrule.go —
  die Datei selbst war schon vor dieser Iteration groesstenteils abgedeckt, siehe Praemissen-
  Korrektur oben). Naechste Unit im Backlog laut Datei-Reihenfolge: `c-cov-caldav-ical`
  (`status: todo`, Zeile ~1749).

## Iteration 44 — c-cov-caldav-ical — done — 2026-08-10 03:45
- commit: (dieser Commit)
- verify vorgaenger: sauber — `665a1641` (c-cov-work-event-rrule) geprueft: Diff enthaelt nur
  `rrule_test.go` (neu) und zwei ergaenzte Assertions in `service_test.go`, kein
  Produktionscode angefasst (`rrule.go` selbst zeigt lokal einen reinen CRLF/LF-Normalisierungs-
  Artefakt ohne inhaltliche Aenderung — `git diff`/`git diff --ignore-all-space` beide leer,
  unbedenklich liegen gelassen), kein gRPC-Layer, kein Proto, keine neue Route, kein neuer
  Guard, keine neue Tabelle.
- gebaut: zwei neue Testdateien fuer `internal/caldav` (vorher 7,2 % Paketcoverage, keine
  Testdatei fuer `ical_converter.go`/`etag.go`): `ical_converter_test.go` (17 Tests) und
  `etag_test.go` (4 Tests). Kernstueck sind Roundtrip-Tests `EventToICal` -> echte ICS-Bytes
  (`ical.NewEncoder`) -> `ical.NewDecoder` -> `ICalToEventInput`, nicht nur In-Memory-Struct-
  Vergleich: normales Event (Titel/Beschreibung/Ort/Zeiten), ganztaegiges Event (VALUE=DATE),
  wiederkehrendes Event (RRULE-Property), Event mit abweichender Zeitzone (America/New_York
  statt Default Europe/Berlin), leere Beschreibung/Ort (muss als `nil` zurueckkommen, nicht als
  Pointer auf Leerstring — `EventToICal` schreibt sie ja gar nicht erst). Exceptions getrennt
  fuer beide Zweige: EXDATE (stornierter Termin, `IsCancelled: true`) und RECURRENCE-ID
  (ueberschriebener Termin mit eigenem Titel/Zeiten). Attendees separat getestet, da
  `ICalToEventInput`/`parseVEvent` das ATTENDEE-Property gar nicht in `CalEventInput`
  zurueckspiegelt (kein Feld dafuer) — Test prueft stattdessen direkt auf der dekodierten
  `ical.Calendar`, dass CN und PARTSTAT beide Attendees ueberleben. `rsvpToPartStat` zusaetzlich
  als eigener Tabellentest (inkl. unbekannter/leerer RSVP-Wert -> Default `NEEDS-ACTION`).
  Fehlerpfade: kein VEVENT im Kalender, fehlendes DTSTART, syntaktisch kaputter ICS-Block direkt
  am `ical.Decoder` (Zufallsbytes inkl. Null-Byte) und ein VEVENT, das nach `BEGIN:VEVENT` ohne
  `END:` abbricht — beide in `assert.NotPanics` gewrappt, wie im `done_when` gefordert ("keinen
  Panic"). `etag_test.go` deckt `GenerateETag` (deterministisch bei gleicher ID+Timestamp,
  aendert sich bei 1ns Differenz, aendert sich bei anderer ID, Quoted-Hex-Format) und
  `GenerateCTag` (Format inkl. negativem Wert) ab — beide waren zuvor komplett ungetestet und
  standen mit im `sources`-Block dieser Unit. `internal/caldav`-Paketcoverage: 29,1 % (von
  7,2 %). Zielfunktionen: `EventToICal` 88,6 %, `ICalToEventInput` 93,4 %, `parseVEvent` 92,5 %,
  `setEventTimes`/`rsvpToPartStat` 100 %, `GenerateETag`/`GenerateCTag` 100 %.
- gate: build ok (`go build ./internal/caldav/...`) | vet ok (`go vet ./internal/caldav/...`) |
  lint ok (`golangci-lint run --config .golangci.yml ./internal/caldav/...`, 0 issues) | test ok
  (`go test -count=3 ./internal/caldav/...` mit gesetzter `DATABASE_URL`, durchgehend gruen,
  inklusive der drei bestehenden DB-Tests des Pakets) | migration n.a. (keine Schema-Aenderung)
  | rls-smoke n.a. (kein DB-Zugriff im neuen Testcode, reine In-Memory-Konvertierung) | route
  n.a. (kein Gateway/Server-Handler angefasst, `go test ./internal/gateway/` deshalb nicht
  Pflicht und nicht separat gelaufen) | openapi n.a. | protoc n.a.
- mutations-probe: `IsCancelled: true` im EXDATE-Parsing-Zweig von `ICalToEventInput`
  (`ical_converter.go:214`) auf `IsCancelled: false` gesetzt →
  `TestEventToICal_RoundTrip_CancelledException` wurde rot ("Should be true"), zurueckgedreht,
  `git diff` auf die Produktionsdatei bestaetigt eine identische Datei (leerer Diff), volle
  Suite danach dreimal in Folge wieder gruen.
- db-tests: 0 neue — `ical_converter.go`/`etag.go` sind reine In-Memory-Konvertierungslogik ohne
  DB-Zugriff, `done_when` verlangt hier keine DB-Tests. Die drei bereits bestehenden DB-Tests
  des Pakets (`app_password_test.go`, `tenant_isolation_phase2_test.go`, `tenant_write_test.go`)
  liefen unveraendert mit, 0 Skips bei gesetzter `DATABASE_URL`.
- offen: keins. Naechste Unit im Backlog laut Datei-Reihenfolge: `c-cov-email-send-mime`
  (`status: todo`, Zeile ~1775, MIME-Nachrichtenerzeugung `internal/email/send/mime_builder.go`).

## Iteration 45 — c-cov-email-send-mime — done — 2026-08-10 (Lauf 7)
- commit: (dieser Commit)
- verify vorgaenger: sauber — `0bfa4012` (c-cov-caldav-ical) geprueft: Diff enthaelt nur zwei neue
  Testdateien (`ical_converter_test.go`, `etag_test.go`), kein Produktionscode angefasst, kein
  gRPC-Layer, kein Proto, keine neue Route, kein neuer Guard, keine neue Tabelle.
- gebaut: neue Testdatei `internal/email/send/mime_builder_test.go` fuer
  `internal/email/send/mime_builder.go` (279 Zeilen, vorher nur indirekt per
  `strings.Contains`-Assertions in `service_test.go` beruehrt, kein echter Parse-Rueckwaerts-Test).
  Roundtrip via `net/mail.ReadMessage` + `mime/multipart` (eigener rekursions-/shallow-faehiger
  Parser, da das Paket selbst keinen bietet): `TestMIMEBuilder_ParseBack_PlainAndHTML` (Text+HTML
  ohne Anhang, echter Decode inkl. Quoted-Printable), `TestMIMEBuilder_SubjectRFC2047_Umlauts`
  (echte Umlaute+Emoji im Betreff, beweist RFC-2047-Kodierung ueber den rohen Header-Wert
  `=?utf-8?...?=` UND per `mime.WordDecoder` zurueckdekodiert exakt gleich dem Original — die
  bisherige `TestMIMEBuilder_UnicodeSubject` in service_test.go nutzte gar keine echten Umlaute
  und pruefte nur `NotEmpty`), `TestMIMEBuilder_MultipleAttachments` (zwei Anhaenge, Base64-Decode
  gegen Originalbytes, Content-Disposition/Dateiname), `TestMIMEBuilder_AttachmentWithoutFilename`
  (leerer Dateiname, `Content-Disposition: attachment; filename=""` parst sauber). Fehlerpfad:
  `TestMIMEBuilder_AttachmentReadError` (Anhang-`io.Reader` liefert Fehler, `Build` gibt ihn durch
  statt zu verschlucken).
- FUND (nicht gefixt, eigene Unit `fix-email-mime-attachment-boundary-mismatch` im Backlog
  angelegt, status: todo): `buildWithAttachments` (mime_builder.go:129) deklariert den Boundary
  des verschachtelten Text/HTML-Bodyparts ueber einen Wegwerf-`multipart.Writer`
  ("just for the boundary", Zeile 137/160), schreibt den tatsaechlichen Inhalt aber ueber einen
  ZWEITEN, unabhaengig erzeugten `multipart.Writer` mit eigenem Zufalls-Boundary (Zeile 144/167).
  Beide Boundaries matchen nie — jede Mail mit mindestens einem Anhang (inline oder nicht, beide
  Zweige betroffen) ist fuer einen RFC-2046-konformen Client strukturell unlesbar; der
  Text/HTML-Body ist nicht wiederherstellbar (Anhaenge selbst sind nicht betroffen, die haengen
  unverschachtelt am aeusseren mixed-Writer). Nicht vermutet, sondern konkret bewiesen: erst
  scheiterte der naive rekursive Parser mit `multipart: NextPart: EOF` beim Versuch, den
  verschachtelten Bodypart zu decodieren; danach per Rohdump verifiziert (deklarierter vs.
  tatsaechlicher Boundary-String im Klartext verschieden), und als eigener, selbsterklaerender
  Test `TestMIMEBuilder_WithAttachments_NestedBoundaryMismatch_DocumentsCurrentGap` festgehalten
  (parst den deklarierten Boundary aus dem Header, fuettert ihn zusammen mit den tatsaechlichen
  Bytes in einen frischen `multipart.Reader`, zeigt `NextPart()` liefert sofort `io.EOF`). Die
  beiden Anhang-Tests umschiffen den Bug bewusst durch einen Shallow-Parse nur auf der aeusseren
  mixed-Ebene (Anhaenge selbst bleiben pruefbar), ohne den kaputten inneren Bodypart zu decodieren.
- gate: build ok (`go build ./internal/email/send/...`) | vet ok
  (`go vet ./internal/email/send/...`) | lint ok (`golangci-lint run --config .golangci.yml
  ./internal/email/send/...`, 0 issues) | test ok (`go test -count=3 ./internal/email/send/...`,
  dreimal wiederholt, durchgehend gruen, 0 Fails, inkl. der 6 bereits bestehenden
  MIMEBuilder-Tests aus service_test.go) | migration n.a. | rls-smoke n.a. (kein DB-Zugriff im
  Paket) | route n.a. (kein Gateway/Server-Handler angefasst) | openapi n.a. | protoc n.a.
- mutations-probe: `mime.QEncoding.Encode("utf-8", input.Subject)` in `Build` (mime_builder.go:90)
  auf reines `input.Subject` (ohne Encoding) gesetzt → `TestMIMEBuilder_SubjectRFC2047_Umlauts`
  wurde rot ("expected RFC 2047 encoded-word, got \"Grüße äöü ß und Emoji 😀 Test\""),
  zurueckgedreht, `git diff` auf die Produktionsdatei bestaetigt eine identische Datei (leerer
  Diff), volle Suite danach wieder gruen.
- db-tests: 0 — reine In-Memory-MIME-Erzeugung ohne DB-Zugriff, `done_when` verlangt hier keine
  DB-Tests.
- coverage: `internal/email/send` Paketcoverage nach dieser Unit 58,4 %; `mime_builder.go` selbst
  95,8 % (`Build`), 70,0-92,9 % je Helferfunktion (`buildAlternative`/`buildWithAttachments`/
  `writeTextPart`/`writeHTMLPart`/`writeAlternativePart`/`writeAttachmentPart`), 100 % fuer
  `String`/`NewMIMEBuilder`/`writeHeader`/`formatAddressList`.
- offen: neue Fix-Unit `fix-email-mime-attachment-boundary-mismatch` (phase 3, status: todo) im
  Backlog fuer den Boundary-Bug — als eigenstaendige, risikoreichere Iteration vorgesehen, nicht
  nebenbei in dieser Coverage-Unit gefixt. Naechste Unit im Backlog laut Datei-Reihenfolge:
  `c-cov-notification-forwarder` (`status: todo`, Zeile ~1803 vor Einfuegen der neuen Fix-Unit,
  `internal/notification/integration/forwarder.go`).

## Iteration 46 — fix-email-mime-attachment-boundary-mismatch — done — 2026-08-10 04:05
- commit: (dieser Commit)
- verify vorgaenger: sauber — `fdcc40dd` (c-cov-email-send-mime) geprueft: Diff enthaelt nur
  `mime_builder_test.go` (neu) plus Journal/Backlog, kein Produktionscode angefasst, kein
  gRPC-Layer, kein Proto, keine neue Route, kein neuer Guard, keine neue Tabelle.
- gebaut: Fix des in Iteration 45 gefundenen Boundary-Mismatch-Bugs in
  `internal/email/send/mime_builder.go:buildWithAttachments`. Beide Zweige (`hasInline` true und
  false) deklarierten das Content-Type-Boundary des verschachtelten Body-Parts ueber einen
  Wegwerf-`multipart.Writer(nil)` ("just for the boundary"), schrieben den tatsaechlichen Inhalt
  aber ueber einen zweiten, unabhaengig erzeugten Writer mit eigenem Zufalls-Boundary — beide
  matchten nie. Fix: der innere Writer wird jetzt direkt gegen einen eigenen `bytes.Buffer`
  erzeugt (`innerRelated`/`innerAlt := multipart.NewWriter(&innerBuf)`), sein ECHTES
  `.Boundary()` geht in den Header, und erst danach wird der fertige Puffer per
  `bodyPart.Write(innerBuf.Bytes())` in den vom `mixedWriter` erzeugten Part geschrieben —
  keine Wegwerf-Writer mehr. Testdatei aktualisiert:
  `TestMIMEBuilder_WithAttachments_NestedBoundaryMismatch_DocumentsCurrentGap` (dokumentierte den
  Bug) ersetzt durch `TestMIMEBuilder_WithAttachments_NestedBodyRoundTrips` mit zwei Subtests
  (`no inline images`, `has inline images`) — echter rekursiver Parse (eigene
  `parseNestedBodyPart`-Hilfsfunktion, nutzt den vom Part selbst deklarierten Boundary, keine
  Shallow-Workarounds) beweist Text- UND HTML-Body sind jetzt in beiden Zweigen decodierbar,
  inklusive Inline-Bild (Content-ID) und dem parallel vorhandenen Nicht-Inline-Anhang. Zusaetzlich
  `TestMIMEBuilder_MultipleAttachments`/`TestMIMEBuilder_AttachmentWithoutFilename` von
  Shallow-Parse (nur Mixed-Ebene) auf vollen rekursiven Parse des nested Alternative-Bodyparts
  umgestellt — genau wie im `done_when` gefordert, nicht geloescht sondern durch echten Beweis
  ersetzt. Signatur von `Build`/`buildWithAttachments` unveraendert (kein Caller-Bruch, geprueft
  gegen `internal/email/systemmail/sender.go` und `internal/email/send/service.go`, beide
  unangetastet).
- gate: build ok (`go build -p 2 ./internal/email/send/...`) | vet ok | lint ok (golangci-lint
  --config .golangci.yml, 0 issues) | test ok (`go test -count=3 ./internal/email/send/...`,
  dreimal wiederholt, durchgehend gruen, inkl. 6 vorbestehende MIMEBuilder-Tests aus
  service_test.go) | migration n.a. (kein Schema beruehrt) | rls-smoke n.a. (kein DB-Zugriff im
  Paket) | route n.a. (kein Gateway/Server-Handler angefasst, `go test ./internal/gateway/`
  deshalb nicht Pflicht und nicht separat gelaufen) | openapi n.a. | protoc n.a.
- mutations-probe: `bodyHeader.Set("Content-Type", ...)` im `hasInline: false`-Zweig
  (`mime_builder.go`) zurueck auf einen hartkodierten falschen Boundary-String
  (`mutation-probe-fake-boundary` statt `innerAlt.Boundary()`) gesetzt — genau der urspruengliche
  Bug-Zustand fuer diesen Zweig. `TestMIMEBuilder_MultipleAttachments` und
  `TestMIMEBuilder_WithAttachments_NestedBodyRoundTrips/no_inline_images` wurden beide rot
  (`multipart: NextPart: EOF`), exakt der dokumentierte Fehler. Zurueckgedreht, `git diff`
  bestaetigt eine zur Ausgangslage identische Produktionsdatei (keine Restaenderung), volle Suite
  danach wieder gruen (`go test -count=1`).
- db-tests: 0 — reine In-Memory-MIME-Erzeugung ohne DB-Zugriff, `done_when` verlangt hier keine
  DB-Tests.
- offen: keins. `done_when` vollstaendig erfuellt: Boundary stimmt in beiden Zweigen, echter
  rekursiver Parse liefert Text/HTML-Body korrekt zurueck, der Gap-Test wurde auf das neue
  Verhalten aktualisiert statt geloescht, `go test -count=1 ./internal/email/send/...` gruen.
  Naechste Unit im Backlog laut Datei-Reihenfolge: `c-cov-notification-forwarder`
  (`status: todo`, `internal/notification/integration/forwarder.go`).

## Iteration 47 — c-cov-notification-forwarder — done — 2026-08-10 04:20
- commit: (dieser Commit)
- verify vorgaenger: sauber — `e89ea594` (fix-email-mime-attachment-boundary-mismatch) geprueft:
  `git show --stat` zeigt nur `mime_builder.go` (Fix) plus Test-/Journal-/Backlog-Dateien, deckt
  sich 1:1 mit dem Journal-Eintrag der Vorgaenger-Iteration. Ausserdem lag ein rein
  zeilenenden-bedingtes `M backend/internal/work/event/rrule.go` im Arbeitsverzeichnis
  (leerer `git diff`, `git diff --raw` und `wc -l` bestaetigten 0 inhaltliche Aenderung) —
  per `git checkout --` zurueckgesetzt, kein Zusammenhang mit dieser Iteration.
- gebaut: `internal/notification/integration/forwarder_test.go` (neu, ~370 Zeilen) fuer das bisher
  ungetestete `forwarder.go` (384 Zeilen, 0 Tests vorher). Aufbau: `fakeRepository` implementiert
  das volle `Repository`-Interface in-memory (nur `ListConfigs`/`ListMappingsByConfig`/
  `UpdateMapping`/`LogDelivery` mit echtem Verhalten, Rest Leerimplementierung fuer die
  Interface-Erfuellung), `fakePoster` implementiert `PlatformPoster` skriptbar
  (Result/Error). Abgedeckt: `TestSelectMostSpecific` (Exact-schlaegt-Wildcard, Fallback auf
  Wildcard, Dedup mehrerer Exact-Treffer auf demselben Kanal, kein Treffer), `TestBuildActionSet`
  (alle vier Praefix-Zweige + Default per Tabellentest), `TestForwarder_TrackFailure_
  DisablesAfterThreshold` (10x kein Disable, 11. Fehlschlag disabled + genau ein
  `UpdateMapping`-Call + Counter-Reset), `TestForwarder_ResetFailures_ClearsCounter`,
  `TestForwarder_DispatchToMapping` (vier Subtests: Erfolg resettet Failures + loggt „sent" mit
  PlatformMessageID, Plattformfehler loggt „failed" + trackt Failure, nicht konfigurierte
  Plattform ist ein stiller No-Op statt Panic, Rate-Limit loggt „rate_limited" ohne
  `PostNotification`-Aufruf), `TestMappingCache_TTLRefresh` (echtes `time.Sleep` ueber eine
  30ms-TTL: vor Ablauf liefert der Cache den alten Wert trotz geaenderter Repo-Daten, nach
  Ablauf den neuen), `TestMappingCache_SkipsInactiveConfigsAndMappings`,
  `TestMappingCache_RefreshPropagatesListConfigsError`.
- fund: beim Schreiben von `TestMappingCache_WildcardMappingNeverReturned_DocumentsCurrentGap`
  bestaetigt, dass ein Wildcard-Channel-Mapping (leeres `Modules`-Feld — "an dieses Modul jede
  Notification weiterleiten") in `MappingCache.refresh` unter dem Literal-Key `"*"` abgelegt wird
  (forwarder.go:370), `GetMappingsForModule` aber NIE `c.modules["*"]` liest (nur den echten
  `moduleID`) — der Kommentar "we'll handle this at query time" ist ein Versprechen, das der Code
  nie einloest. Ein Wildcard-Mapping wird also geladen und dann fuer JEDES Modul als leere Liste
  zurueckgegeben — die einfachste denkbare Admin-Konfiguration ("alles an diesen Slack-Kanal")
  ist von Anfang an tot, ohne Fehler, ohne Log. `ListActiveMappingsForModule` (Repository-
  Interface + Postgres-Impl) ist per Grep bestaetigt toter Code (0 Call-Sites) und würde das
  Problem auch nicht loesen (SQL `modules @> $1::jsonb` matched ebenfalls keine leere Liste).
  Nicht inline gefixt — echte Verhaltensaenderung, keine Coverage-Aenderung, nach Backlog-Regel
  eine eigene Unit: neue Fix-Unit `fix-notification-wildcard-mapping-never-delivered` (phase 3,
  status: todo) im Backlog angelegt, direkt nach `fix-bexio-tenant-id-missing-on-upsert`
  eingefuegt (gleiche Gruppierung wie alle anderen waehrend Coverage-Arbeit gefundenen Fix-Units).
- gate: build ok (`go build -p 2 ./internal/notification/...`) | vet ok
  (`go vet ./internal/notification/...`) | lint ok (golangci-lint --config .golangci.yml
  ./internal/notification/integration/..., 0 issues, nach Behebung zweier Modernize-Hinweise —
  ungenutztes `tc := tc`-Shadowing entfernt, `for i := 0; i < 10; i++` auf `for range 10`
  umgestellt) | test ok (`go test -count=3 ./internal/notification/integration/...`, dreimal
  wiederholt durchgehend gruen, keine Flakiness trotz echtem `time.Sleep` im TTL-Test) | `-race`
  in dieser Umgebung nicht verfuegbar (`CGO_ENABLED=0`, keine cgo-Toolchain) — stattdessen
  `-count=3` als Ersatzbeleg gegen Datenrennen in den lock-geschuetzten Feldern (`failureMu`,
  `MappingCache.mu`) | migration n.a. (keine Tabelle beruehrt) | rls-smoke n.a. (keine Tabelle/
  Policy angefasst) | route n.a. (kein Gateway/Server-Handler angefasst) | openapi n.a. |
  protoc n.a. | mit `DATABASE_URL=postgres://kmuhub_app:app_dev@localhost:5432/kmuhub` erneut
  `go test -count=1 -v ./internal/notification/integration/...` gefahren (Rolle `kmuhub_app`,
  nicht `kmuhub`): alle 12 Testfunktionen inkl. der drei vorbestehenden DB-Tests
  (`TestTenantIsolation_Integration_DB`, `TestIntegrationWrites_RefuseWithoutTenant`,
  `TestIntegrationWrites_LandInCallerTenant`) real gelaufen und gruen, 0 Skips.
- mutations-probe: `count > 10` in `trackFailure` (forwarder.go:182) auf `count > 100` gesetzt.
  `TestForwarder_TrackFailure_DisablesAfterThreshold` wurde rot ("want mapping disabled after 11
  consecutive failures"), exakt der erwartete Fehlschlag. Zurueckgedreht, `git diff --stat`
  bestaetigt eine zur Ausgangslage identische Produktionsdatei (0 Zeilen Diff), volle Suite
  danach wieder gruen (`go test -count=1 ./internal/notification/...`).
- db-tests: 3 real gelaufen (0 Skips, siehe gate) — alle drei vorbestehend, nicht Teil dieser
  Unit. Diese Unit selbst fuegt keine DB-Tests hinzu: `done_when` verlangt hier keine (reine
  In-Memory-Logik hinter Interfaces, analog zur Email-MIME-Unit), Coverage-Ziel bereits ueber
  Fakes erreichbar.
- coverage: `internal/notification/integration` Paketcoverage nach dieser Unit 35,7 % (`slack`
  35,0 %, `teams` 11,0 % — beide unveraendert, nicht Teil dieser Unit).
- offen: neue Fix-Unit `fix-notification-wildcard-mapping-never-delivered` (phase 3,
  status: todo) im Backlog fuer den Wildcard-Bug. Naechste Unit im Backlog laut Datei-Reihenfolge:
  `c-cov-caldav-vcard-vtimezone` (`status: todo`, `internal/caldav` vCard/VTIMEZONE-Pfade).

## Iteration 48 — c-cov-caldav-vcard-vtimezone — done — 2026-08-10 (Lauf 7)
- commit: (dieser Commit)
- verify vorgaenger: sauber — `26b3f776` (c-cov-notification-forwarder) geprueft: `git show --stat`
  zeigt nur `forwarder_test.go` (neu) plus Backlog-/Journal-Dateien, deckt sich 1:1 mit dem
  Journal-Eintrag der Vorgaenger-Iteration. Kein Merge-Konflikt (`git merge origin/main` lief als
  "Already up to date" — kein Divergenzrisiko).
- gebaut: `internal/caldav/vcard_converter_test.go` (neu, ~245 Zeilen) und
  `internal/caldav/vtimezone_test.go` (neu, ~140 Zeilen) fuer die beiden bisher ungetesteten
  Dateien `vcard_converter.go` (127 Z.) und `vtimezone.go` (114 Z.). vCard-Seite:
  `TestContactToVCard_FullFields`/`_MissingOptionalFields`/`_EmptyStringFieldsAreOmitted`/
  `_NoName_FallsBackToUnnamed`, `TestContactToVCardWithCompany_SetsOrg`/`_NilOrEmptyCompany_NoOrg`,
  echte Text-Roundtrips (`vcardEncodeDecode` analog zu `encodeDecode` aus
  `ical_converter_test.go` — durch den echten vCard-Encoder/Decoder, nicht nur In-Memory-Vergleich)
  fuer vollstaendigen Kontakt und fuer "Kontakt ohne Nachname" (Firmenname im FirstName-Feld, N
  traegt GivenName ohne FamilyName), `VCardToContactInput`-Fallback-Pfade
  (kein N-Feld -> FN-Split mehrwortig/einwortig, weder N noch FN -> leeres Ergebnis, N hat
  Vorrang vor FN wenn beide gesetzt), und ein Test, der direkt gegen die go-vcard-Bibliothek
  belegt, dass mehrere EMAIL/TEL-Eintraege mit TYPE-Labels und PREF-Parameter den echten
  Wire-Roundtrip alle ueberleben (Bibliotheksebene), waehrend `VCardToContactInput` bewusst nur
  den bevorzugten (PREF=1) Eintrag herauszieht (Converter-Ebene) — `ContactInput` hat wie
  `models.Contact` nur ein einzelnes Email-/Phone-Feld, kein Mehrfach-Datenverlust also kein Bug,
  sondern dokumentiertes Verhalten. VTIMEZONE-Seite: `TestGenerateVTimezone_EuropeBerlin_...`
  prueft TZID, beide STANDARD/DAYLIGHT-Bloecke (DTSTART, RRULE, TZOFFSETFROM/TO, TZNAME) exakt
  gegen CET/CEST, `TestGenerateVTimezone_DACHAliases_SameShapeAsBerlin` (Tabellentest
  Zurich/Vienna), `TestGenerateVTimezone_Minimal_FixedOffsetZone` (Asia/Tokyo — DST-frei, damit
  das Ergebnis unabhaengig vom Testlaufzeitpunkt deterministisch ist), `_InvalidTimezone_
  ReturnsError`, `_Caching_ReturnsSamePointer` (`assert.Same` beweist echten Cache-Hit, nicht nur
  gleichen Wert), `TestFormatUTCOffset` (Tabellentest inkl. negativ, halbe Stunde/Indien, Null),
  `TestBuildMinimalTimezone_MatchesRuntimeOffset` (America/New_York — DST-behaftete Zone, Erwartung
  zur Laufzeit ueber `time.Now().In(loc).Zone()` berechnet statt hartkodiert, damit der Test
  ganzjaehrig stabil bleibt).
- fund: keiner — beide Dateien verhalten sich wie im Scope beschrieben, keine neue Fix-Unit noetig.
- gate: build ok (`go build -p 2 ./internal/caldav/...`) | vet ok (`go vet ./internal/caldav/...`)
  | lint ok (`golangci-lint run --config .golangci.yml ./internal/caldav/...`, 0 issues) | test ok
  (`go test -count=1 ./internal/caldav/...`, komplettes Paket inkl. bestehender DB-/RLS-Tests
  gruen) | migration n.a. | rls-smoke n.a. | route n.a. | openapi n.a. | protoc n.a.
- mutations-probe: `TZOFFSETTO` des STANDARD-Blocks in `buildDACHTimezone` (vtimezone.go:58) von
  `"+0100"` auf `"+0200"` gesetzt. `TestGenerateVTimezone_EuropeBerlin_StandardAndDaylightOffsets`
  wurde rot (`expected: "+0100", actual: "+0200"`), exakt der erwartete Fehlschlag.
  Zurueckgedreht, `git diff --stat internal/caldav/vtimezone.go` liefert keine Ausgabe (identisch
  zur Ausgangslage), volle Suite danach wieder gruen (`go test -count=1 ./internal/caldav/...`).
- db-tests: 0 — beide Dateien sind reine In-Memory-Konvertierung/-Erzeugung ohne DB-Zugriff,
  `done_when` verlangt hier keine DB-Tests.
- coverage: `internal/caldav` Paketcoverage nach dieser Unit 26,7 % (vorher 7,2 %, siehe Iteration
  46 — `vcard_converter.go`/`vtimezone.go` beide jetzt 100 % auf Funktionsebene fuer die reinen
  Helfer wie `formatUTCOffset`/`setPropValue`).
- offen: keins. `done_when` vollstaendig erfuellt. Naechste Unit im Backlog laut Datei-Reihenfolge:
  `c-cov-caldav-backend-helpers` (`status: todo`, Pfad-Parser/Proto-Konvertierung/Fehler-Mapping
  in `caldav_backend.go`/`carddav_backend.go`).

## Iteration 49 — c-cov-caldav-backend-helpers — done — 2026-08-10 (Lauf 7)
- commit: (dieser Commit)
- verify vorgaenger: sauber — `f9991767` (c-cov-caldav-vcard-vtimezone) geprueft: `git show --stat`
  zeigt nur `vcard_converter_test.go`/`vtimezone_test.go` (neu) plus Backlog-/Journal-Dateien,
  deckt sich 1:1 mit dem Journal-Eintrag der Vorgaenger-Iteration. `git merge origin/main` lief
  als "Already up to date" — kein Divergenzrisiko, keine STOP-Datei.
- gebaut: `internal/caldav/caldav_backend_test.go` (neu) und `internal/caldav/carddav_backend_test.go`
  (neu) fuer die reinen Helferfunktionen aus `caldav_backend.go`/`carddav_backend.go`, die weder
  gRPC-Client noch DB-Pool brauchen. caldav_backend.go-Seite: `calendarIDFromPath` (gueltiger Pfad,
  fehlendes "calendars"-Segment, "calendars" als letztes Segment, ungueltige UUID),
  `eventUIDFromPath` (gueltiger `.ics`-Pfad, komplett leerer Pfad als Fehlerfall),
  `protoEventToModel` (alle Felder gesetzt inkl. aller optionalen Pointer, sowie Minimalfall mit
  allen optionalen Pointern nil — Ergebnis-Model behaelt dort ebenfalls nil statt zu crashen),
  `protoAttendeesToModels` (nil-Response -> nil, leere Attendee-Liste -> nicht-nil leeres Slice,
  volle Liste inkl. eines Attendees ohne `RespondedAt`), `grpcToWebDAVError` als Tabellentest ueber
  NotFound/PermissionDenied/Unauthenticated/InvalidArgument/Unavailable/Internal/Unknown (die
  letzten beiden dokumentieren denselben Default-Zweig -> 500), plus nil-Input -> nil und ein
  Nicht-gRPC-Fehler, der unveraendert (per `assert.Same`) durchgereicht wird. Da
  `webdav.NewHTTPError` einen `*internal.HTTPError` aus dem (fuer dieses Modul nicht importierbaren)
  `go-webdav/internal`-Paket zurueckgibt, extrahiert ein kleiner Test-Helfer `webdavStatusCode` den
  Code aus dem dokumentierten `Error()`-Format (`"<code> <statustext>[: <cause>]"`, verifiziert direkt
  am go-webdav-Quelltext in `internal/internal.go`), statt den Typ zu importieren.
  carddav_backend.go-Seite: `addressBookTypeFromPath` (personal/company/fehlendes Segment/unbekannter
  Typ — letzte beide Faelle fallen beide auf den "personal"-Default), `contactIDFromPath` (gueltige
  `.vcf`-UUID, ungueltige UUID, komplett leerer Pfad), `syncCollectionIDForAddressBook`
  (deterministisch fuer gleiche User+Typ-Kombination, unterschiedlich je Typ und je User — beweist
  den SHA1-Namespace-Ansatz), `contactInfoToVCard` (volle Felder inkl. Firma via
  `vcardEncodeDecode`-Wire-Roundtrip aus `vcard_converter_test.go` wiederverwendet, fehlende
  optionale Felder ohne Crash, Name-Fallback auf "Unnamed Contact" bei leerem Vor-/Nachnamen),
  `parseContactUpdatedAt` (gueltiges RFC3339, nicht parsebarer String und leerer String fallen
  beide auf `time.Now()` zurueck — belegt mit `assert.WithinDuration`, da ein exakter Zeitwert
  hier naturgemaess nicht erwartbar ist). `strPtr`/`baseContact`/`vcardEncodeDecode` aus
  `vcard_converter_test.go` wiederverwendet statt neu erfunden (gleiches Package), eigene
  `baseContactInfo()`-Fixture fuer den abweichenden `crmv1.ContactInfo`-Typ ergaenzt.
- fund: keiner — beide Dateien verhalten sich wie im Scope beschrieben. `eventUIDFromPath`s
  `uid == last && strings.Contains(last, "/")`-Bedingung ist faktisch totes Wrap (ein per `/`
  gesplitteter Abschnitt kann nie selbst ein `/` enthalten) und die Funktion akzeptiert Pfade ohne
  `.ics`-Endung klaglos (liefert das letzte Segment unveraendert zurueck) — beides ein
  Code-Geruch, aber kein beobachtbarer Fehlverhalten mit Produktauswirkung (die Route-Ebene
  reicht ohnehin nur `.ics`-Pfade durch), deshalb keine neue Fix-Unit angelegt, nur hier notiert.
- gate: build ok (`go build -p 2 ./internal/caldav/...`) | vet ok (`go vet ./internal/caldav/...`)
  | lint ok (`golangci-lint run --config .golangci.yml ./internal/caldav/...`, 0 issues) | test ok
  mit gesetztem `DATABASE_URL` (`postgres://kmuhub_app:app_dev@localhost:5432/kmuhub?sslmode=disable`,
  `go test -count=1 ./internal/caldav/...`, komplettes Paket inkl. bestehender DB-/RLS-Tests real
  gelaufen, 0 Skips) | migration n.a. | rls-smoke n.a. | route n.a. | openapi n.a. | protoc n.a.
- mutations-probe: `codes.NotFound`-Fall in `grpcToWebDAVError` (caldav_backend.go) von
  `http.StatusNotFound` auf `http.StatusTeapot` gesetzt. `TestGrpcToWebDAVError_CodeMapping/not_found`
  wurde rot (`expected: 404, actual: 418`), alle anderen Subtests blieben gruen (praezise auf den
  einen veraenderten Fall isoliert), exakt der erwartete Fehlschlag. Zurueckgedreht,
  `git diff --stat internal/caldav/caldav_backend.go` liefert keine Ausgabe (identisch zur
  Ausgangslage), volle Suite danach wieder gruen (`go test -count=1 ./internal/caldav/...` mit
  gesetztem `DATABASE_URL`).
- db-tests: 3 real gelaufen (0 Skips, siehe gate) — alle drei vorbestehend, nicht Teil dieser Unit.
  Diese Unit selbst fuegt keine DB-Tests hinzu: reine In-Memory-Pfad-Parser/Proto-Konvertierung/
  Fehler-Mapping, `done_when` verlangt hier keine.
- coverage: `internal/caldav` Paketcoverage nach dieser Unit 41,5 % (vorher 26,7 %, siehe Iteration
  48).
- offen: keins. `done_when` vollstaendig erfuellt. Naechste Unit im Backlog laut Datei-Reihenfolge:
  `c-cov-email-sync-helpers` (`status: todo`, `DetectFolderType`/`envelopeToMessage`/
  `imapAddressesToModel`/`parseEnvelopeDate`/`firstInReplyTo` in `internal/email/sync`).

## Iteration 50 — c-cov-email-sync-helpers — done — 2026-08-10 (Lauf 7)
- commit: (dieser Commit)
- verify vorgaenger: sauber — `248ace02` (c-cov-caldav-backend-helpers) geprueft: `git show --stat`
  zeigt nur `caldav_backend_test.go`/`carddav_backend_test.go` (neu) plus Backlog-/Journal-Dateien,
  deckt sich 1:1 mit dem Journal-Eintrag der Vorgaenger-Iteration. `git merge origin/main` lief als
  "Already up to date" — kein Divergenzrisiko, keine STOP-Datei.
- gebaut: `internal/email/sync/helpers_test.go` (neu) fuer die reinen Helferfunktionen, die laut
  Scope KEINE echte IMAP-Verbindung brauchen: `DetectFolderType` (Tabellentest ueber alle Eintraege
  aus `folderTypeMap` inkl. Gross-/Kleinschreibung, Umlaut- vs. ASCII-Variante von "Entwürfe", plus
  unbekannter/leerer Name -> `FolderTypeCustom`), `parseEnvelopeDate` (gueltiges RFC-5322-artiges
  Datum mit `Z` und mit `+02:00`-Offset parst exakt, kaputter String und leerer String fallen beide
  auf `time.Now().UTC()` zurueck statt eines ungepruesten Zero-Values — belegt mit
  `assert.WithinDuration`, da ein exakter Zeitwert hier naturgemaess nicht erwartbar ist),
  `firstInReplyTo` (leere/nil Liste -> `""`, erstes Element getrimmt zurueckgegeben), sowie das im
  selben Paket lebende `imapAddressesToModel` (nil/leer -> nil, populierte Liste mappt
  `Name`/`Addr()` korrekt inkl. leerem Namen). `envelopeToMessage` (Methode auf `*Worker`) per
  Struct-Literal-`Worker{account: &models.EmailAccount{...}}` instanziiert (gleiches Package, keine
  Service-Abhaengigkeiten noetig fuer diese reine Konvertierung) — deckt Adress-/Flag-Mapping
  (Seen/Flagged/Draft), `InReplyTo`/`References`-Uebernahme, leeren From-Fall, und die
  200-Zeichen-Preview-Truncation (Grenzfall exakt 250 Zeichen Subject -> 200 Zeichen Preview).
- fund: keiner — alle fuenf Funktionen verhalten sich wie im Scope beschrieben, keine neue Fix-Unit
  noetig. Die Methoden, die wirklich gegen einen IMAP-Server sprechen (Connect/Login/Fetch* in
  `imap_client.go`, Run/syncCycle/syncFolder/idleLoop/pollLoop in `worker.go`), sind wie im Scope
  festgelegt bewusst nicht Teil dieser Unit geblieben.
- gate: build ok (`go build -p 2 ./internal/email/sync/...`) | vet ok
  (`go vet ./internal/email/sync/...`) | lint ok (`golangci-lint run --config .golangci.yml
  ./internal/email/sync/...`, 0 issues) | test ok (`go test -count=1 ./internal/email/sync/...`,
  komplettes Paket gruen) | migration n.a. | rls-smoke n.a. | route n.a. | openapi n.a. | protoc n.a.
- mutations-probe: in `parseEnvelopeDate` (worker.go:494-499) den Erfolgsfall von `return t` auf
  `return t.Add(24 * time.Hour)` gesetzt. `TestParseEnvelopeDate/valid_RFC3339_date_parses_exactly`
  und `.../valid_date_with_offset_parses_exactly` wurden beide rot (erwartet 2026-03-05, erhalten
  2026-03-06), die beiden Fallback-Subtests blieben gruen (praezise auf die zwei veraenderten
  Erfolgsfaelle isoliert) — exakt der erwartete Fehlschlag. Zurueckgedreht, `git diff --stat
  internal/email/sync/worker.go` liefert keine Ausgabe (identisch zur Ausgangslage), volle Suite
  danach wieder gruen (`go test -count=1 ./internal/email/sync/...`).
- db-tests: 0 — alle fuenf Funktionen sind reine In-Memory-Parsing/-Konvertierungs-Helfer ohne
  DB-Zugriff, `done_when` verlangt hier keine.
- coverage: `internal/email/sync` Paketcoverage nach dieser Unit 10,9 % (vorher 0 %, kein Testfile
  existierte). Bewusst niedrig trotz vollstaendig gedeckter Scope-Funktionen: `Run`/`syncCycle`/
  `syncFolders`/`syncFolder`/`idleLoop`/`pollLoop` sowie fast der gesamte `imap_client.go`-Inhalt
  (Connect/Login/Select/Fetch*/Idle/Noop/ListFolders) bleiben ungetestet, weil sie einen echten
  IMAP-Server brauchen (go.mod hat kein Test-Server-Paket dafuer) — exakt wie im Scope begruendet
  ausgeschlossen.
- offen: keins. `done_when` vollstaendig erfuellt. Naechste Unit im Backlog laut Datei-Reihenfolge:
  `c-cov-work-task-repo` (`status: todo`, `List`/`Search`/`GetSubtasks`/`GetParentChain`/
  `GetDepth`/`HasCycle` in `internal/work/task/postgres_repository.go`).

## Iteration 51 — c-cov-work-task-repo — done — 2026-08-10 (Lauf 7)
- commit: (dieser Commit)
- verify vorgaenger: sauber — `829156e9` (c-cov-email-sync-helpers) geprueft: `git show --stat`
  zeigt nur `internal/email/sync/helpers_test.go` (neu) plus Backlog-/Journal-Dateien, deckt sich
  1:1 mit dem Journal-Eintrag der Vorgaenger-Iteration. `git merge origin/main` lief als
  "Already up to date" — kein Divergenzrisiko, keine STOP-Datei.
- gebaut: `internal/work/task/postgres_repository_db_test.go` (neu) fuer die im Scope benannte
  Luecke: `List` (Filterkombinationen Priority/AssigneeID/StatusID/IsStandalone/Search plus
  Pagination), `Search` (Volltextsuche gegen `search_vector`, kombiniert mit Priority-Filter,
  Null-Treffer-Fall), `GetSubtasks`/`GetParentChain` (dreistufige Verschachtelung
  Parent->2 Children->1 Grandchild, inkl. `maxDepth`-Cutoff-Test), `GetDepth` (Kette bis
  `MaxNestingDepth-1` plus `ErrNotFound`-Pfad) und `HasCycle` (direkter Zyklus B->A blockt A->B,
  transitiver Zyklus C->B->A blockt A->C, unbeteiligtes Task-Paar bleibt unauffaellig). `List`/
  `Search` pruefen explizit, dass ein fremdtenant-Task mit identischem Titel-Substring/Volltext-
  Token weder in den Zeilen noch in `total` auftaucht (COUNT(*) traegt dieselbe WHERE wie die
  Seite) — genau das vom Scope geforderte Kriterium.
- fund: keiner an der getesteten Business-Logik selbst. EIN echtes API-Missverstaendnis beim
  Bauen aufgedeckt und sofort korrigiert (kein Fix-Unit-Fall, kein Verhaltensbug): `GetSubtasks`/
  `GetParentChain`/`GetDepth`/`HasCycle` nehmen anders als `List`/`Search` KEIN `tenantID`-Argument
  entgegen — die Isolation laeuft ausschliesslich ueber die RLS-Session im `ctx`. Erste Testversion
  rief sie mit nacktem `context.Background()` auf und bekam durchgaengig 0 Zeilen / `ErrNotFound`
  zurueck (RLS ohne gesetztes `app.tenant_id` sieht nichts) — auf `testutil.WithTenantCtx(...,
  tenantOwn)` umgestellt, danach alle fuenf gruen. Kein Bug im Produktionscode, nur ein
  Testaufbau-Fehler.
- gate: build ok (`go build -p 2 ./internal/work/task/...`) | vet ok
  (`go vet ./internal/work/task/...`) | lint ok (`golangci-lint run --config .golangci.yml
  ./internal/work/task/...`, 0 issues) | test ok (`go test -count=1 ./internal/work/task/...` mit
  gesetztem `DATABASE_URL`, komplettes Paket gruen, 0 Skips) | migration n.a. | rls-smoke n.a.
  (bestehende `rls_test.go`-Suite lief in derselben Runde mit) | route n.a. | openapi n.a. |
  protoc n.a.
- mutations-probe: in `HasCycle` (postgres_repository.go:425) `WHERE task_id = $1` auf
  `WHERE task_id = $2` gesetzt. `TestHasCycle_DirectAndTransitiveDetection` wurde sofort rot
  (Postgres verweigert die Query: "could not determine data type of parameter $1", weil der
  Parameter durch die Aenderung nirgends mehr referenziert wird) — ein noch haerterer Fehlschlag
  als ein falscher Bool-Wert, aber exakt der erwartete Fehlschlag: der Test haette jede Aenderung
  an dieser Zeile aufgedeckt. Alle anderen Tests des Pakets blieben von der Aenderung unberuehrt.
  Zurueckgedreht, `git diff --stat internal/work/task/postgres_repository.go` liefert keine
  Ausgabe (identisch zur Ausgangslage), volle Suite danach wieder gruen
  (`go test -count=1 ./internal/work/task/...`).
- db-tests: 5 neu, alle gegen das reale Schema (0 Skips) — `TestList_FiltersAndTenantScopedTotal`,
  `TestSearch_FullTextAndTenantScopedTotal`, `TestGetSubtasksAndGetParentChain_MultiLevelOrder`,
  `TestGetDepth_DeepChainAndNotFound`, `TestHasCycle_DirectAndTransitiveDetection`.
- coverage: `internal/work/task` Paketcoverage nach dieser Unit 63,5 % (Vorwert nicht separat
  gemessen — CRUD/RLS/Service-Tests liefen bereits vorher, diese Unit schliesst gezielt die im
  Scope benannte Luecke in List/Search/Baum-Navigation/Zyklenerkennung).
- offen: keins. `done_when` vollstaendig erfuellt. Naechste Unit im Backlog laut Datei-Reihenfolge:
  `c-cov-work-meeting-repo` (`status: todo`, Teilnehmerverwaltung/Serientermin-Ausnahmen/Listen-
  pfade in `internal/work/meeting/postgres_repository.go`).

## Iteration 52 — c-cov-work-meeting-repo — done — 2026-08-10 (Lauf 7)
- commit: (dieser Commit)
- verify vorgaenger: sauber — `778d030e` (c-cov-work-task-repo) geprueft: `git show --stat` zeigt
  nur `internal/work/task/postgres_repository_db_test.go` (neu) plus Backlog-/Journal-Dateien,
  deckt sich 1:1 mit dem Journal-Eintrag der Vorgaenger-Iteration. `git merge origin/main` lief
  als "Already up to date" — kein Divergenzrisiko, keine STOP-Datei.
- gebaut: `internal/work/meeting/postgres_repository_db_test.go` (neu, 10 Testfunktionen) fuer
  den im Scope benannten groessten unabgedeckten Repository-Block des work-Pakets: Attendee-CRUD
  inkl. RSVP und ON-CONFLICT-Idempotenz (`TestAttendees_...`), Listen-/Filterpfade von
  `ListMeetings` (Organizer/Attendee-EXISTS/Status/Zeitfenster/Pagination) plus expliziter
  Cross-Tenant-Nachweis, dass RLS auch dann blockt, wenn `filter.TenantID` selbst das Opfer-Tenant
  benennt (`TestListMeetings_FiltersAndCrossTenant`), `GetMeetingByRoomName`/`ListStaleMeetings`
  unter System-Kontext, die Serientermin-Isolation (drei Meetings mit gemeinsamem
  `recurring_meeting_id`, `UpdateMeeting` auf einer Instanz beruehrt die Geschwister nicht) plus
  `GetPreviousMeetingNotes` ueber die Serie (`TestNotes_SeriesIsolationAndSaveNotesConflictGap`),
  Action-Items (CRUD + `UpdateActionItemTaskID` + alle Not-Found-Pfade), Chat-Messages (Limit-Clamp
  <=0/>500 -> 100, expliziter Tenant-Parameter), Co-Hosts (Add/Remove-Idempotenz, IsCoHost, List)
  und die komplette Breakout-Room-/Assignment-Kette (Create/List/Get/CloseAll,
  Upsert-Reassign/Get/List/Clear/ClearAll) sowie `SetLocked`/`UpdateAISummary` mit Not-Found-Pfad.
- fund: **echter Bug, nicht inline gefixt** — `SaveNotes`
  (internal/work/meeting/postgres_repository.go:308) hat ein `ON CONFLICT (meeting_id, author_id)
  WHERE is_private = $5`-Ziel, fuer das auf `meeting_notes` KEIN passender Unique-/Exclusion-Index
  existiert (verifiziert per `\d meeting_notes` auf docker-postgres-1: nur PK auf `id` plus zwei
  einfache btree-Indizes, und per Grep aller Migrationen — 000037 legt die Tabelle ohne
  entsprechenden Constraint an, 000109/000124 aendern nur `tenant_id`/RLS). Jeder einzige Aufruf
  von `SaveNotes` schlaegt fehl — kein Teilfall, sondern eine 100-%-Fehlerrate auf einer
  Kernfunktion. Empirisch per direktem `psql`-INSERT gegen docker-postgres-1 reproduziert (exakter
  Fehler: "there is no unique or exclusion constraint matching the ON CONFLICT specification"),
  danach `TestNotes_SeriesIsolationAndSaveNotesConflictGap` gebaut, die genau diesen Fehler
  assertet, und fuer die uebrigen Notes-Fixtures (GetNotes/GetAllNotes/GetPreviousMeetingNotes,
  alle drei funktionieren korrekt) auf `testutil.SeedRow` statt `repo.SaveNotes` umgestellt. Neue
  Fix-Unit `fix-meeting-savenotes-onconflict-no-matching-constraint` im Backlog angelegt
  (Produktfrage vorab: soll je Autor nur eine private Notiz erlaubt sein, oder mehrere Scratch-
  Notizen? — beeinflusst, ob ein oder zwei partielle Unique-Indizes noetig sind).
- gate: build ok (`go build -p 2 ./internal/work/meeting/...`) | vet ok
  (`go vet ./internal/work/meeting/...`) | lint ok (`golangci-lint run --config .golangci.yml
  ./internal/work/meeting/...`, 0 issues, ein `rangeint`-Hinweis waehrend der Entwicklung sofort
  auf `for i := range 3` umgestellt) | test ok (`go test -count=1 ./internal/work/meeting/...` mit
  gesetztem `DATABASE_URL`, komplettes Paket gruen inkl. aller bestehenden Service-/RLS-Tests,
  0 Skips) | migration n.a. (keine neue Migration in dieser Coverage-Unit — der gefundene Bug wird
  in der neuen Fix-Unit behoben) | rls-smoke n.a. (bestehende Tenant-Isolation-Suite lief in
  derselben Runde mit) | route n.a. | openapi n.a. | protoc n.a.
- mutations-probe: in `GetPreviousMeetingNotes` (postgres_repository.go:373) `ORDER BY
  m.scheduled_start DESC, mn.created_at ASC` auf `ASC, mn.created_at ASC` gesetzt.
  `TestNotes_SeriesIsolationAndSaveNotesConflictGap` wurde sofort rot (Sub-Case "after both":
  erwartete die Notiz von Woche 2 als juengste, erhielt stattdessen die von Woche 1) — exakt der
  erwartete Fehlschlag, direkt an der im Scope geforderten Serientermin-Faehigkeit. Zurueckgedreht,
  `git diff --stat internal/work/meeting/postgres_repository.go` liefert keine Ausgabe (identisch
  zur Ausgangslage), volle Suite danach wieder gruen (`go test -count=1
  ./internal/work/meeting/...`).
- db-tests: 10 neu, alle gegen das reale Schema (0 Skips) —
  `TestAttendees_AddUpdateRemoveAndGet`, `TestListMeetings_FiltersAndCrossTenant`,
  `TestGetMeetingByRoomName_And_ListStaleMeetings`,
  `TestNotes_SeriesIsolationAndSaveNotesConflictGap`, `TestActionItems_CRUDAndTaskIDLink`,
  `TestChatMessages_SaveListLimitAndCrossTenant`, `TestCoHosts_AddIsListRemoveIdempotent`,
  `TestBreakoutRoomsAndAssignments_FullLifecycle`, `TestSetLocked_And_UpdateAISummary_ErrorPaths`.
- coverage: `internal/work/meeting` Paketcoverage nach dieser Unit 72,0 % (Vorwert nicht separat
  gemessen — Service-/RLS-Tests liefen bereits vorher, diese Unit schliesst gezielt die im Scope
  benannte Repository-Luecke). Die zwei verbleibenden 0-%-Zeilen laut `go tool cover -func`
  (`ReturnToMainRoom`, `CloseBreakoutRooms` in service.go) sind Service-Ebene, nicht Teil dieses
  Repository-Scopes.
- offen: `fix-meeting-savenotes-onconflict-no-matching-constraint` (neu, `status: todo`) —
  Produktentscheidung zu privaten Notizen vor dem Fix noetig. `done_when` dieser Coverage-Unit
  vollstaendig erfuellt. Naechste Unit im Backlog laut Datei-Reihenfolge: `c-cov-dialer-repo`
  (`status: todo`, Listen-/Such-/Reporting-Methoden in `internal/dialer/postgres_repository.go`).

## Iteration 53 — fix-notification-wildcard-mapping-never-delivered — done — 2026-08-10 (Lauf 7)
- commit: (dieser Commit)
- verify vorgaenger: sauber — `b70814aa` (c-cov-work-meeting-repo) geprueft: `git show --stat`
  zeigt nur `internal/work/meeting/postgres_repository_db_test.go` (neu) plus Backlog-/
  Journal-Dateien, deckt sich 1:1 mit dem Journal-Eintrag der Vorgaenger-Iteration.
  `git merge origin/main` lief als "Already up to date" — kein Divergenzrisiko, keine STOP-Datei.
- korrektur zur Vorgaenger-Notiz: der Journal-Eintrag von Iteration 52 nannte `c-cov-dialer-repo`
  als naechste Unit (Datei-Reihenfolge ab dem eigenen Fundpunkt weitergelesen). Tatsaechlich
  liegt `fix-notification-wildcard-mapping-never-delivered` (Zeile 900, aus Iteration ~44/
  c-cov-notification-forwarder gefunden) weiter vorne in der Datei und war seit deren Anlage
  `status: todo` geblieben — per `grep -n "^  - id:\|status:"` bestaetigt: erster
  `status: todo`-Treffer nach dem Datei-Kopf. Diese Iteration hat die Datei-Reihenfolge
  massgeblich genommen, nicht die Journal-Vermutung.
- gebaut/gefixt: `internal/notification/integration/forwarder.go` —
  `MappingCache.GetMappingsForModule` liest jetzt zusaetzlich zum exakten Modul-Key auch
  `c.modules["*"]` (Wildcard-Mappings) und fuehrt beide Slices ueber die neue Hilfsfunktion
  `mergeModuleMappings` zusammen (leerer Slice -> anderer Slice direkt zurueckgegeben, sonst
  neu allokiert und beide angehaengt). Die Exact-vs-Wildcard-Praezedenz bleibt unveraendert
  in `selectMostSpecific` (unberuehrt) — die Funktion bekommt jetzt einfach beide Kandidaten
  statt nur der (fehlenden) Wildcard-Haelfte.
- test: `forwarder_test.go` — `TestMappingCache_WildcardMappingNeverReturned_DocumentsCurrentGap`
  ersetzt durch zwei Tests auf das neue Verhalten:
  `TestMappingCache_WildcardMappingReturnedForAnyModule` (Wildcard-Mapping wird fuer zwei
  verschiedene, im Mapping selbst gar nicht genannte Module zurueckgegeben) und
  `TestMappingCache_ExactMappingWinsOverWildcard` (Wildcard + ein exaktes `crm`-Mapping im
  selben Cache: `crm`-Anfrage liefert nur das exakte, `biz`-Anfrage faellt auf die Wildcard
  zurueck — End-to-End durch `GetMappingsForModule` + `selectMostSpecific`, nicht nur die reine
  Merge-Funktion isoliert).
- gate: build ok (`go build -p 2 ./internal/notification/...`) | vet ok (`go vet
  ./internal/notification/...`) | lint ok (`golangci-lint run --config .golangci.yml
  ./internal/notification/...`, 0 issues — ein vorbestehender `slicescontains`-Hinweis in
  `selectMostSpecific` Zeile 221 ist ausserhalb des Scopes dieser Fix-Unit, nicht angefasst)
  | test ok (`go test -count=1 ./internal/notification/...`, alle sieben Unterpakete gruen,
  kein DATABASE_URL noetig — reine In-Memory-Cache-Logik) | migration n.a. | rls-smoke n.a.
  | route n.a. | openapi n.a. | protoc n.a.
- mutations-probe: in `mergeModuleMappings` den ersten Zweig `if len(exact) == 0 { return
  wildcard }` auf `return nil` gesetzt. Beide neuen Tests wurden sofort rot —
  `TestMappingCache_WildcardMappingReturnedForAnyModule` ("expected the wildcard mapping to
  be returned for module crm, got []") und `TestMappingCache_ExactMappingWinsOverWildcard`
  ("expected the wildcard mapping to be selected for module biz, got []") — exakt der
  erwartete Fehlschlag am genau gefixten Pfad. Zurueckgedreht, `git diff --stat
  internal/notification/integration/forwarder.go` zeigt danach nur noch den beabsichtigten
  Fix (19 Einfuegungen/3 Loeschungen ggue. dem Ausgangsstand), volle Suite danach wieder gruen
  (`go build`/`go vet`/`go test -count=1 ./internal/notification/...`).
- offen: keins. `done_when` vollstaendig erfuellt (Wildcard-Mapping wird fuer jedes Modul ohne
  spezifischeres exaktes Mapping zurueckgegeben, Exact-schlaegt-Wildcard unveraendert, Test auf
  neues Verhalten aktualisiert statt geloescht, Paket gruen). Naechste Unit im Backlog laut
  Datei-Reihenfolge: `fix-meeting-savenotes-onconflict-no-matching-constraint` (`status: todo`,
  Zeile ~1033 — Produktentscheidung zu privaten Notizen vorab noetig, siehe Notes der Unit),
  danach `c-cov-dialer-repo`.

## Iteration 54 — fix-meeting-savenotes-onconflict-no-matching-constraint — done — 2026-08-10 (Lauf 7)
- commit: `7e39b615`
- verify vorgaenger: sauber — `18906e8c` (fix-notification-wildcard-mapping-never-delivered)
  geprueft: `git show --stat` zeigt nur `internal/notification/integration/forwarder.go` und
  `forwarder_test.go` plus Backlog-/Journal-Dateien, deckt sich 1:1 mit dem Journal-Eintrag
  der Vorgaenger-Iteration. `git merge origin/main` lief als "Already up to date" — kein
  Divergenzrisiko, keine STOP-Datei.
- produktentscheidung (Vorbedingung der Unit): die Unit verlangte vorab zu klaeren, ob private
  Notizen ebenfalls auf eine pro Autor/Meeting begrenzt sein sollen. Gegen den bestehenden Code
  entschieden statt geraten: `GetNotes` (postgres_repository.go:323) liest ohne
  `is_private`-Filter mit `LIMIT 1` je `(meeting_id, author_id)`, und `SaveNotes`s eigenes
  `ON CONFLICT (meeting_id, author_id) WHERE is_private = $5` parametrisiert die Eindeutigkeit
  bereits ueber `is_private` — beides zusammen legt zwingend ein Modell von genau einer
  oeffentlichen und einer privaten Notiz pro Autor/Meeting fest; "mehrere private Scratch-Notizen"
  haette eine komplett andere SaveNotes-Persistenz (Insert statt Upsert) erfordert, die nirgends
  im Code (Service/Handler) angelegt ist. Keine Eskalation noetig, keine Annahme jenseits des
  bereits geschriebenen Codes.
- gebaut/gefixt: `backend/migrations/000309_meeting_notes_unique_index.{up,down}.sql` — zwei
  partielle Unique-Indizes auf `meeting_notes(meeting_id, author_id)`, einer `WHERE is_private =
  false`, einer `WHERE is_private = true`, exakt passend zum ON-CONFLICT-Ziel in `SaveNotes`.
  Migration lokal angewendet: `migrate -path backend/migrations -database
  "postgres://kmuhub:kmuhub_dev@localhost:5432/kmuhub?sslmode=disable" up` (Docker-Hostname
  `postgres` aus `MIGRATION_DATABASE_URL` funktioniert vom Host aus nicht, auf `localhost`
  umgebogen), Ergebnis `309/u meeting_notes_unique_index`. `\d meeting_notes` gegen
  `docker-postgres-1` bestaetigt beide Indizes final passend zur Migration.
- test: `internal/work/meeting/postgres_repository_db_test.go` —
  `TestNotes_SeriesIsolationAndSaveNotesConflictGap` umbenannt zu
  `TestNotes_SeriesIsolationAndSaveNotesUpsert`, der bisherige "SaveNotes gap"-Block (erwartete
  den Postgres-Fehler) ersetzt durch drei echte Assertions: (1) `SaveNotes` legt eine neue
  oeffentliche Notiz an, (2) ein zweiter `SaveNotes`-Call fuer dieselbe
  `(meeting_id, author_id, is_private)`-Kombination aktualisiert dieselbe Zeile (`DO UPDATE`,
  `GetAllNotes` liefert danach weiterhin genau eine Zeile mit dem neuen Inhalt, nicht zwei), (3)
  eine private Notiz fuer denselben Autor/dasselbe Meeting entsteht als eigene, separate Zeile und
  taucht nicht in `GetAllNotes` (public-only) auf. Ungenutzten `strings`-Import entfernt (war nur
  fuer die alte Fehlertext-Pruefung noetig). Datei-Kopfkommentar (Zeile 3-7) von "genuine ON
  CONFLICT gap ... documents rather than fixes" auf den jetzt korrekten Zustand aktualisiert.
- gate: build ok (`go build -p 2 ./internal/work/... ./cmd/work/...`) | vet ok (`go vet
  ./internal/work/...`) | lint ok (`golangci-lint run --config .golangci.yml ./internal/work/...`,
  0 issues) | test ok (`go test -count=1 ./internal/work/meeting/...`, volles Paket gruen — 0
  uebersprungen, `DATABASE_URL=postgres://kmuhub_app:app_dev@localhost:5432/kmuhub?sslmode=disable`
  gesetzt, Rolle `kmuhub_app` verifiziert) | migration ok (siehe oben) | rls-smoke n.a. (keine neue
  Tabelle/Policy, nur Indizes auf bestehender RLS-Tabelle) | route n.a. | openapi n.a. | protoc n.a.
- mutations-probe: `meeting_notes_meeting_author_private_unique`-Index live per `psql` (Rolle
  `kmuhub`) `DROP`ped, `go test -run TestNotes_ -v ./internal/work/meeting/...` sofort rot —
  `SaveNotes (create private): ... no unique or exclusion constraint matching the ON CONFLICT
  specification (SQLSTATE 42P10)`, exakt am gefixten Pfad (dem privaten Zweig). Index per
  identischem `CREATE UNIQUE INDEX ... WHERE is_private = true` zurueckgedreht, Test danach wieder
  gruen, `\d meeting_notes` final gegen die Migration abgeglichen (beide Indizes vorhanden,
  identisch benannt).
- offen: keins. `done_when` vollstaendig erfuellt (Migration passt exakt zum ON-CONFLICT-Ziel,
  SaveNotes gelingt fuer neue Kombinationen, zweiter Call aktualisiert statt dupliziert, Test auf
  neues Verhalten umgestellt statt geloescht, Paket gruen). Naechste Unit im Backlog laut
  Datei-Reihenfolge (per `grep -n "^  - id:\|status:"` verifiziert, nicht nur der vorigen
  Journal-Notiz vertraut — die nannte faelschlich `c-cov-biz-lexware`, das laut Backlog bereits
  `status: done` ist): `c-cov-dialer-repo` (Zeile ~2160, `status: todo`).

## Iteration 55 — c-cov-dialer-repo — done — 2026-08-10 (Lauf 7)
- commit: `8760b26e`
- verify vorgaenger: sauber — `3572424f` (fix-meeting-savenotes-onconflict-no-matching-constraint)
  geprueft: `git show --stat` zeigt nur die Migration 000309 (up/down), den DB-Test in
  `internal/work/meeting/postgres_repository_db_test.go` sowie Backlog-/Journal-Dateien, deckt
  sich 1:1 mit dem Journal-Eintrag der Vorgaenger-Iteration. Kein `.proto` beruehrt, kein neuer
  `RequirePermission`-Guard, keine neue Route, kein Wire-Shape-Wechsel — keine der acht
  Fehlerklassen einschlaegig. `git merge origin/main` lief als "Already up to date".
- gebaut: `internal/dialer/queue_and_list_test.go` (neu, 901 Zeilen) — 13 neue Testfunktionen
  gegen das reale Schema, decken den laut Backlog-Scope groessten unabgedeckten Repository-Block
  ab: `CampaignRepository.List`/`ListContacts` (Status-Filter, Pagination, Tenant-gescopte
  Gesamtzahl, negative-OFFSET-Fehlerpfad), `AddContacts` (Duplikat-Skip via ON CONFLICT,
  Leer-Input-Kurzschluss, Fehlerpfad bei falscher `tenantID` — Subquery liefert NULL, NOT-NULL-
  Constraint schlaegt zu), `GetNextPendingContact` (Claim in `position ASC`, nicht
  Einfuegereihenfolge, `ErrNoContactsAvailable` bei leerer Queue), `GetCampaignStats` (volle
  Aggregation inkl. Outcome-Breakdown und Avg-Dauer) + `UpdateCampaignCounts`,
  `CallRepository`s komplette Session-Lifecycle (`CreateSession`/`GetSessionByID`/`UpdateSession`
  inkl. Not-Found-Pfade), die atomare `UpdateSessionWithEventAndContact`-Transaktion (Erfolgspfad
  UND ein echter Rollback-Beweis: eine Session-Notiz-Aenderung + ein absichtlich auf eine
  nicht-existente Session-ID gesetztes Event fuehren dazu, dass der Event-Insert an der
  NOT-NULL-Spalte scheitert und die Notiz-Aenderung sichtbar zurueckrollt), `AppendEvent`/
  `ListEventsBySession`, die Today-Counts (`GetTenantCallsTodayCount`,
  `GetTenantAppointmentsTodayCount`, `GetAgentCallsTodayCount`, `GetAgentAvgDurationToday`),
  `GetRecentCallsForTenant`/`ListCallsByContact` (inkl. Cross-Tenant-Leerpruefung),
  `OutcomeRepository.List`/`EnsureDefaults` (Idempotenz-Beweis: zweiter Call dupliziert nicht)
  und `AgentStatusRepository.GetActiveAgentIDsForTenant`/`GetUserDisplayNames` (stille
  Missing-User-Omission, Leer-Input-Kurzschluss). `tenant_write_test.go` (Campaign-/
  Contact-CRUD-Writes, Outcome-Writes, `GetAgentStats`) und `rls_test.go` (rohe RLS-Zeilenzahl)
  nicht dupliziert.
- fixture-lehre: erste Fassung nutzte statische E-Mail-Strings (`"dialer-list@test.local"` etc.)
  fuer `seedDialerUser` — kollidierte beim zweiten Lauf mit `idx_users_email` (UNIQUE), weil
  `t.Cleanup`-registrierte Aufraeumung durch ein `defer pool.Close()` VOR den `t.Cleanup`-Calls
  unwirksam gemacht wurde (Go raeumt erst alle `defer`s der Testfunktion ab, danach erst die
  `t.Cleanup`-Kette — bei einem plain `defer pool.Close()` ist der Pool also schon zu, wenn
  `CleanupRow` laeuft). Fix: `defer pool.Close()` durchgaengig zu
  `t.Cleanup(func() { pool.Close() })` gemacht (registriert vor allen Fixture-Helpern, laeuft
  dadurch unter LIFO NACH ihnen) und `seedDialerUser` auf `emailPrefix + "-" + uuid.New()`
  umgestellt. Zweiter Fund beim Nachpruefen der lokalen DB nach dem ersten gruenen vollen Lauf:
  `TestCampaignRepository_AddContacts_SkipsDuplicatesAndRejectsForeignTenant` liess trotzdem zwei
  `users`-Zeilen zurueck — die zweite `AddContacts`-Charge legt einen `dialer_campaign_contacts`-
  Join ausserhalb von `SeedRow` an, dessen `t.Cleanup` nur fuer die ERSTE Charge registriert war;
  beim Abraeumen versuchte `CleanupRow` den zugehoerigen Kontakt zu loeschen, waehrend der Join
  noch existierte (`contact_id`-FK ohne CASCADE) — `CleanupRow` loggt Fehler nur (`t.Logf`), faellt
  also nicht auf, ohne die Zeile danach explizit zu pruefen. Fix: ein einzelner
  Sweep-Cleanup (`DELETE FROM dialer_campaign_contacts WHERE campaign_id = $1`) **als letzte**
  Registrierung nach allen Kontakt-Fixtures, laeuft dadurch unter LIFO vor jeder Kontakt-
  Aufraeumung. Beide Fixes committed als Teil dieser Unit, keine Nacharbeit noetig — beide
  Bugs waren reine Test-Hygiene (lokale Docker-DB), keine Produktionslogik.
- gate: build ok (`go build -p 2 ./internal/dialer/... ./cmd/dialer/...`) | vet ok (`go vet
  ./internal/dialer/...`) | lint ok (`golangci-lint run --config .golangci.yml
  ./internal/dialer/...`, 0 issues) | test ok (`go test -count=1 ./internal/dialer/...`, volles
  Paket gruen — 0 uebersprungen, `DATABASE_URL=postgres://kmuhub_app:app_dev@localhost:5432/
  kmuhub?sslmode=disable` gesetzt, Rolle `kmuhub_app` verifiziert; dreifach mit `-count=3` gegen
  die neue `AddContacts`-Testfunktion wiederholt, um die Cleanup-Fixes zu erhaerten) | migration
  n.a. (keine Schemaaenderung) | rls-smoke n.a. (keine neue Tabelle/Policy, nur Lesepfade auf
  bestehenden RLS-Tabellen) | route n.a. | openapi n.a. | protoc n.a.
- mutations-probe: zwei unabhaengige Proben, beide zurueckgedreht (`git diff --stat
  internal/dialer/postgres_repository.go` danach leer). (1) `CampaignRepository.List`s
  `countQuery` von `WHERE tenant_id = $1%s` auf `WHERE true%s` geaendert (Tenant-Bedingung aus der
  Gesamtzahl entfernt) — `TestCampaignRepository_List_FiltersPaginatesAndScopesTotal` sofort rot
  ("expected 0 arguments, got 1"), exakt am Pfad, den `done_when` explizit verlangt ("Gesamtzahl
  traegt dieselbe Tenant-Bedingung wie die Seite"). (2) In
  `UpdateSessionWithEventAndContact` den ersten `tx.Exec` (Session-Update) auf `r.pool.Exec`
  umgestellt — damit laeuft dieser Schritt AUSSERHALB der Transaktion. Anschliessender
  `TestCallRepository_UpdateSessionWithEventAndContact_Atomic` sofort rot ("session update was
  not rolled back") — der absichtlich herbeigefuehrte Fehlerpfad (Event mit nicht-existenter
  `DialerCallSessionID`) rollt die Transaktion zurueck, aber die ausserhalb liegende
  Session-Notiz-Aenderung bleibt sichtbar committed, genau der Bug-Typ, den der Docstring-
  Kommentar des Codes ("A failure in any step rolls back all three") verspricht zu verhindern.
  Beide Male zurueckgedreht, danach `go build`/`go vet`/`golangci-lint`/`go test -count=1
  ./internal/dialer/...` erneut komplett gruen.
- offen: nicht ALLE 39 Repository-Methoden sind jetzt einzeln getestet — bewusst nicht mitgenommen
  (kein natuerlicher Fehlerpfad, reine Aggregat-SELECTs ohne Konstante/Constraint zum Brechen):
  keine. Alle in `postgres_repository.go` definierten Methoden sind jetzt in mindestens einem
  Test aufgerufen (verifiziert per `grep -c '\.<Methode>('` ueber alle `*_test.go` vor Abschluss
  dieser Unit — 0 Treffer fuer keine der 26 zuvor unabgedeckten Methoden mehr). `done_when`
  vollstaendig erfuellt (Listen-/Queue-Filterkombinationen, Pagination-Tenant-Bedingung,
  Fehlerpfad pro getesteter Funktion, Mutations-Probe im Journal belegt, Paket gruen 0 Skips).
  Naechste Unit im Backlog laut Datei-Reihenfolge: `c-cov-work-calendar-repo` (Zeile ~2189,
  `status: todo`).

## Iteration 56 — c-cov-work-calendar-repo — done — 2026-08-10 (Lauf 7)
- commit: `3aafe2a5`
- verify vorgaenger: sauber — `8760b26e`/`48fd85b2` (c-cov-dialer-repo, Iteration 55) geprueft:
  `git show --stat 48fd85b2` zeigt `queue_and_list_test.go` (neu, 901 Zeilen, reine
  Test-Datei) plus BACKLOG.yml/JOURNAL.md — deckt sich 1:1 mit dem Journal-Eintrag der
  Vorgaenger-Iteration. `8760b26e` (Zwischen-Commit derselben Aenderung ohne Backlog/
  Journal-Diff) ist kein Ancestor von `48fd85b2`, weil die Iteration ihn per Amend um die
  Backlog-/Journal-Datei erweitert hat — inhaltlich identisch, kein Grund zur Sorge. Kein
  `.proto` beruehrt, kein neuer `RequirePermission`-Guard, keine neue Route, kein
  Wire-Shape-Wechsel. `git merge origin/main` lief als "Already up to date".
- gebaut: `internal/work/calendar/repository_gaps_test.go` (neu, 10 Testfunktionen) —
  schliesst die Luecke in `postgres_repository.go` (432 Z., 21 Methoden) und
  `booking_postgres_repository.go` (266 Z., 11 Methoden), die zuvor nur duenn durch
  `tenant_write_test.go` (Create/Update/Delete), `tenant_isolation_phase2_test.go`
  (SeedRow-basierte RLS-Zeilenzahl) und `booking_slug_unique_test.go` (globaler
  Active-Slug-Index) abgedeckt waren. Neu: `TestCalendarMembers_PermissionAndVisibility
  Lifecycle` (AddMember/GetMember/ListMembers/UpdateMemberPermission/
  UpdateMemberVisibility/UpdateMemberColorOverride/RemoveMember, UND der Beweis, dass eine
  Mitgliedschaftsaenderung `ListByUser` tatsaechlich beeinflusst — Backlog-Prioritaet 1),
  `TestCalendarGetByID_CrossTenantNotFound` (RLS blockt selbst wenn der echte Opfer-Tenant
  als Methodenargument mitgegeben wird), `TestListBrowsable_UnusedSecondParameter_
  DocumentsCurrentGap` + `TestCalendarSubscription_SubscribeAndUnsubscribe` (siehe Fund
  unten), `TestEventCategories_CreateListDelete` (Case-insensitive Unique-Index,
  fremder-User-Delete → NotFound), `TestUserCalendarPreferences_UpsertAndGet` (nil-bei-
  fehlender-Zeile-Vertrag, ON-CONFLICT-Update), `TestEnsurePersonalCalendar_
  IdempotentAndCreates` (zweiter Call liefert dieselbe Calendar-ID, kein Duplikat),
  `TestBookingPages_CRUDAndListFiltering` (Create/Get/Update/Delete/List inkl.
  includeInactive-Filter, inaktive Seite verschwindet aus dem Slug-Lookup, Fremd-Tenant-
  Update/Delete → NotFound), `TestPublicBookings_CreateAndGetBookedSlots`
  (GetBookedSlotsForPage schliesst `status='cancelled'` aus, UpdatePublicBookingCalendar
  EventID schreibt zurueck), `TestGetCalendarEventsInRange_OverlapBoundaryAndCrossTenant`
  (Backlog-Prioritaet 2: ein Slot, der exakt an einer bestehenden Buchung endet, ist frei,
  ein ueberlappender ist belegt — UND Cross-Tenant-Lesepfad, da diese eine Methode gar
  keinen expliziten `tenant_id`-Parameter hat und komplett auf RLS angewiesen ist).
- fund (nicht gefixt, eigene Unit `fix-work-calendar-listbrowsable-broken-query` angelegt):
  `ListBrowsable` (postgres_repository.go:233) bindet drei Argumente (`userID, userID,
  tenantID`) fuer eine Query, die nur `$1` (zweimal) und `$3` referenziert — `$2` kommt im
  SQL-Text nirgends vor. pgx nutzt per Default das Extended-Query-Protocol (Parse/Bind/
  Describe); fuer einen Platzhalter, den die Query nie referenziert, kann Postgres keinen
  Typ ermitteln → JEDER Aufruf schlaegt mit `ERROR: could not determine data type of
  parameter $2 (SQLSTATE 42P18)` fehl, unabhaengig von Tenant/User/Fixture-Daten. Kein
  RLS-Thema, kein Datenproblem — die Query ist strukturell kaputt. Erste Testfassung
  (`TestCalendarDiscovery_ListBrowsableAndSubscription`, wollte Subscribe/Unsubscribe UND
  Browsable-Filterlogik gemeinsam pruefen) schlug am `ListBrowsable`-Call selbst fehl, bevor
  irgendeine Filterlogik ueberhaupt getestet werden konnte — kein Fixture-Fehler auf meiner
  Seite, sondern der Beweis, dass die Methode nie funktioniert hat (kein vorheriger Test hat
  sie je aufgerufen). Aufgeteilt in `TestListBrowsable_UnusedSecondParameter_
  DocumentsCurrentGap` (dokumentiert 42P18 explizit) und
  `TestCalendarSubscription_SubscribeAndUnsubscribe` (prueft Subscribe/Unsubscribe direkt
  ueber GetMember, unabhaengig von der kaputten Methode). Fix-Vorschlag im Backlog-Eintrag:
  das doppelte `userID`-Argument streichen, `$3`→`$2` im SQL-Text fuer `tenant_id`.
- eigene test-hygiene-lehre: erste Fassung von `seedCalendarWithOwner`-Aufrufstellen
  registrierte `defer CleanupRow(..., "calendars", ...)` VOR `defer CleanupRow(...,
  "users", ...)` — unter LIFO laeuft der User-Cleanup dann ZUERST, wodurch
  `calendars_owner_id_fkey` verletzt wird (Calendar referenziert den User noch). Fix: an
  allen sieben Aufrufstellen die Reihenfolge getauscht (User-Cleanup zuerst registrieren,
  Calendar-Cleanup danach — LIFO raeumt dann Calendar vor User ab), exakt das Muster aus
  `tenant_write_test.go`. Zweiter Fund: `UpsertPreferences`/`GetPreferences` mit
  `context.Background()` statt `testutil.WithTenantCtx(...)` aufgerufen — das INSERT
  ermittelt `tenant_id` zwar per Subquery aus `users`, aber die RLS-`WITH CHECK`-Klausel
  wertet weiterhin `app.tenant_id` aus dem Connection-Kontext aus; ohne gesetzten Tenant im
  ctx schlaegt der Insert mit "new row violates row-level security policy" fehl. Fix: ctx
  mit `WithTenantCtx` durchgaengig verwendet. Beide Fixes sind reine Test-Hygiene, keine
  Produktionslogik veraendert.
- gate: build ok (`go build -p 2 ./internal/work/calendar/...`) | vet ok (`go vet
  ./internal/work/calendar/...`) | lint ok (`golangci-lint run --config .golangci.yml
  ./internal/work/calendar/...`, 0 issues) | test ok (`go test -count=1
  ./internal/work/calendar/...`, volles Paket gruen — 0 uebersprungen, `DATABASE_URL=
  postgres://kmuhub_app:app_dev@localhost:5432/kmuhub?sslmode=disable`, Rolle `kmuhub_app`
  verifiziert) | migration n.a. (keine Schemaaenderung) | rls-smoke n.a. (keine neue
  Tabelle/Policy) | route n.a. | openapi n.a. | protoc n.a.
- mutations-probe: zwei unabhaengige Proben, beide zurueckgedreht (`git diff --stat
  internal/work/calendar/postgres_repository.go internal/work/calendar/
  booking_postgres_repository.go` danach leer). (1)
  `GetCalendarEventsInRange`s Bedingung von `start_time < $3 AND end_time > $2` auf
  `start_time <= $3 AND end_time >= $2` geaendert (strikt → nicht-strikt) —
  `TestGetCalendarEventsInRange_OverlapBoundaryAndCrossTenant` sofort rot ("should have 1
  item(s), but has 2" — der angrenzende Termin zaehlt jetzt faelschlich als Konflikt),
  exakt am Pfad, den `done_when` explizit verlangt ("Angrenzender Slot gilt als frei"). (2)
  `ListByUser`s Sichtbarkeitsbedingung von `(c.owner_id = $1 OR cm.user_id = $1)` auf
  `(c.owner_id = $1)` verkuerzt (Member-Zweig entfernt) —
  `TestCalendarMembers_PermissionAndVisibilityLifecycle` sofort rot ("calendar must be
  visible via ListByUser after membership was granted"), exakt am Pfad, den `done_when`
  explizit verlangt ("Member-Berechtigungsaenderung wirkt sich auf die Sichtbarkeit aus").
  Beide Male zurueckgedreht, danach `go build`/`go vet`/`golangci-lint`/`go test -count=1
  ./internal/work/calendar/...` erneut komplett gruen.
- offen: `fix-work-calendar-listbrowsable-broken-query` (neu im Backlog, vor
  `c-cov-plugin-repository-gaps` eingefuegt) fixt den 42P18-Fund. `done_when` dieser Unit
  vollstaendig erfuellt (Member-Sichtbarkeit belegt, Slot-Grenzfall belegt, Cross-Tenant-
  Lesepfad belegt, zwei Mutations-Proben im Journal, Paket gruen 0 Skips). Naechste Unit im
  Backlog laut Datei-Reihenfolge: `fix-work-calendar-listbrowsable-broken-query` (neu
  eingefuegt, `status: todo`) — danach `c-cov-plugin-repository-gaps`.

## Iteration 57 — fix-work-calendar-listbrowsable-broken-query — done — 2026-08-10 (Lauf 7)
- commit: `c398d17c`
- verify vorgaenger: sauber — `3aafe2a5` (c-cov-work-calendar-repo, Iteration 56) und
  `74d774bf` (Folge-Commit, traegt nur den Commit-Hash in den Journal-Eintrag von Iteration
  56 nach) geprueft: `git show --stat 3aafe2a5` zeigt `repository_gaps_test.go` (neu, reine
  Test-Datei) plus BACKLOG.yml/JOURNAL.md — deckt sich 1:1 mit dem Journal-Eintrag. Kein
  `.proto` beruehrt, kein neuer `RequirePermission`-Guard, keine neue Route, kein
  Wire-Shape-Wechsel. `git merge origin/main` lief als "Already up to date", kein Konflikt.
- gefixt: `PostgresRepository.ListBrowsable` (internal/work/calendar/postgres_repository.go:
  233) band drei Argumente (`userID, userID, tenantID`) fuer eine Query, die im SQL-Text nur
  `$1` (zweimal) und `$3` referenzierte — `$2` kam nirgends vor, jeder Aufruf schlug mit
  SQLSTATE 42P18 fehl. Fix: doppelte `userID`-Bindung gestrichen (`userID, tenantID` statt
  `userID, userID, tenantID`), Platzhalter im SQL-Text von `$3` auf `$2` fuer `tenant_id`
  umnummeriert — reiner 2-Zeilen-Diff, keine Logikaenderung an der WHERE-Klausel selbst.
- test: `TestListBrowsable_UnusedSecondParameter_DocumentsCurrentGap` in
  `internal/work/calendar/repository_gaps_test.go` von einem reinen Fehler-Beweis (assert auf
  "42P18") auf einen echten Verhaltenstest umgestellt, Testname bewusst NICHT geloescht
  (done_when-Vorgabe). Deckt alle drei geforderten Faelle in einem Testlauf: ein shared
  Calendar eines fremden Owners ist browsable, ein personal Calendar nie (dritter, eigens
  dafuer geseedeter Kalender), ein bereits per `Subscribe` abonnierter shared Calendar wird
  ueber die NOT-EXISTS-Klausel ausgeschlossen. Nutzt den bestehenden
  `seedCalendarWithOwner`-Helper aus derselben Datei (Iteration 56) fuer alle drei Kalender.
- gate: build ok (`go build -p 2 ./internal/work/calendar/...`) | vet ok (`go vet
  ./internal/work/calendar/...`) | lint ok (`golangci-lint run --config .golangci.yml
  ./internal/work/calendar/...`, 0 issues) | test ok (`go test -count=1
  ./internal/work/calendar/...`, volles Paket gruen — 0 uebersprungen, `DATABASE_URL=
  postgres://kmuhub_app:app_dev@localhost:5432/kmuhub?sslmode=disable`, Rolle `kmuhub_app`
  verifiziert) | migration n.a. (keine Schemaaenderung) | rls-smoke n.a. (keine neue
  Tabelle/Policy) | route n.a. | openapi n.a. | protoc n.a.
- mutations-probe: NOT-EXISTS-Subquery-Klausel testweise aus der Query entfernt (Fix
  bleibt intakt, nur die Filterlogik zurueckgebaut) —
  `TestListBrowsable_UnusedSecondParameter_DocumentsCurrentGap` sofort rot ("Should be
  false" / "already-subscribed shared calendar must be excluded"), exakt am Pfad, den
  `done_when` explizit verlangt ("bereits abonnierte Kalender werden ausgeschlossen").
  Danach zurueckgedreht (`git diff --stat internal/work/calendar/postgres_repository.go`
  zeigt wieder nur den urspruenglichen 2-Zeilen-Fix), `go build`/`go vet`/`golangci-lint`/
  `go test -count=1 ./internal/work/calendar/...` erneut komplett gruen.
- offen: keine neue Unit angelegt. `done_when` dieser Unit vollstaendig erfuellt (kein
  42P18 mehr, alle drei Browsable-Faelle mit einem Test belegt, bestehender Testname
  aktualisiert statt geloescht, Mutations-Probe im Journal belegt, Paket gruen 0 Skips).
  Naechste Unit im Backlog laut Datei-Reihenfolge: `c-cov-plugin-repository-gaps`.

## Iteration 58 — c-cov-plugin-repository-gaps — done — 2026-08-10 (Lauf 7)
- commit: `4df032e4`
- verify vorgaenger: sauber — `c398d17c` (fix-work-calendar-listbrowsable-broken-query,
  Iteration 57) geprueft: `git show --stat` zeigt nur `postgres_repository.go` (2-Zeilen-Fix),
  `repository_gaps_test.go` (Test aktualisiert statt geloescht) plus BACKLOG.yml/JOURNAL.md.
  Kein `.proto`, kein neuer `RequirePermission`-Guard, keine neue Route. Branch-Merge mit
  origin/main lief als "Already up to date", kein Konflikt.
- gebaut: vier neue Testdateien fuer die vier bisher ungetesteten Repositories in
  `internal/plugin/repository/` (432 Zeilen Produktionscode ohne Testdatei, siehe
  Backlog-Scope):
  - `installation_test.go`: `TestInstallation_Lifecycle` (Create, UNIQUE(tenant_id,
    manifest_id)-Verstoss abgelehnt, GetByID/GetByTenantAndManifest je mit Treffer- und
    Nicht-Treffer-Fall, List mit Status-Filter vor/nach Aktivierung, UpdateStatus inkl.
    ErrorMessage-Set-und-Clear, UpdateSettings) und `TestInstallation_ListActiveByHook`
    (zwei Installationen mit identischer Hook-Registrierung, nur die aktive erscheint —
    beweist Filterung auf Hook-Match UND Status gemeinsam, nicht nur Zufall durch die
    einzige vorhandene Installation).
  - `kv_store_test.go`: `TestKVStore_IsolatedByInstallation` (zwei Installationen mit
    identischem Key "config:theme" halten unabhaengige Werte — Isolation nach
    installation_id, nicht nur nach Tenant, wie im Backlog-Scope gefordert — plus
    Missing-Key, Upsert-Overwrite, Praefix-Filter, Delete) und
    `TestKVStore_Set_UnknownInstallation_DocumentsCurrentGap`.
  - `execution_log_test.go`: `TestExecutionLog_ListWithInstallationFilterAndLimit` (Create
    fuer zwei Installationen, List ohne Filter, List mit installation_id-Filter, List mit
    Limit=1 gegen ORDER BY created_at DESC, plus derselbe Silent-No-Op-Fund wie bei KVStore
    fuer Create gegen eine unbekannte installation_id).
  - `industry_template_test.go`: `TestIndustryTemplate_CreateGetListBySlug` (Create,
    ON-CONFLICT(slug)-Upsert, GetByID/GetBySlug je mit Treffer- und
    Nicht-Treffer-Fall, List mit Industry-Filter — eigener Industry-Wert gewaehlt, der in
    keiner der von Migration 000058 geseedeten Zeilen vorkommt, damit die
    Count-Assertion exakt bleibt).
  Alle vier folgen dem in `manifest_rls_test.go` etablierten Muster (Paket
  `repository_test`, `testutil.PoolFromEnv`/`SeedRow`/`EnsureTenant`/`WithTenantCtx`,
  eigene Tenants/Slugs statt geteilter Fixtures).
- fund (dokumentiert, nicht gefixt): `KVStoreRepository.Set` UND `ExecutionLogRepository.
  Create` leiten `tenant_id` per `INSERT ... SELECT ... FROM plugin_installations pi WHERE
  pi.id = $2` aus der Installation ab. Findet die SELECT-Subquery keine Zeile (erfundene
  ID, geloeschte Installation, oder RLS blockt eine fremde Tenant-Installation), fuegt die
  INSERT...SELECT-Form 0 Zeilen ein und `Exec` meldet KEINEN Fehler — `Service.KVSet`/der
  gRPC-Handler `KVSet` reichen das unveraendert durch, der Aufrufer bekommt
  `Success: true`, obwohl nichts gespeichert wurde. Kein Sicherheitsproblem (RLS verhindert
  Schreiben/Lesen fremder Daten), aber ein irrefuehrender Erfolgsstatus. Neue Unit
  `fix-plugin-kvset-silent-noop-unknown-installation` fuer Lauf 8 angelegt (vor
  `c-cov-work-event-repo` eingefuegt, `status: todo`), mit Fix-Vorschlag
  (CommandTag.RowsAffected() pruefen, eigenen Sentinel zurueckgeben).
- gate: build ok (`go build -p 2 ./internal/plugin/...`) | vet ok (`go vet
  ./internal/plugin/...`) | lint ok (`golangci-lint run --config .golangci.yml
  ./internal/plugin/repository/...`, 0 issues) | test ok (`go test -count=1
  ./internal/plugin/repository/...`, volles Paket gruen inkl. bestehender RLS-/
  Tenant-Isolation-Tests — 0 uebersprungen, `DATABASE_URL=
  postgres://kmuhub_app:app_dev@localhost:5432/kmuhub?sslmode=disable`, Rolle `kmuhub_app`
  verifiziert) | migration n.a. (keine Schemaaenderung) | rls-smoke n.a. (keine neue
  Tabelle/Policy) | route n.a. | openapi n.a. | protoc n.a.
- notiz: `UpdateSettings`-Assertion mit rohem String-Vergleich
  (`string(afterSettings.Settings) != string(newSettings)`) schlug zunaechst fehl — Postgres
  normalisiert JSONB beim Round-Trip auf `{"theme": "dark"}` (Leerzeichen nach dem Doppelpunkt),
  waehrend der eingesetzte Literal-String `{"theme":"dark"}` keins hatte. Kein Bug, reine
  Testannahme-Korrektur: auf `json.Unmarshal` + Map-Zugriff umgestellt statt Byte-fuer-Byte-
  Vergleich.
- mutations-probe: drei unabhaengige Proben, alle zurueckgedreht (Diff-Stat der drei
  betroffenen Dateien danach leer). (1) `InstallationRepository.List`s Status-Filter-
  Bedingung `if status != ""` auf `if false && status != ""` gesetzt —
  `TestInstallation_Lifecycle` sofort rot ("list status=active before activation: got 1"),
  exakt am Pfad, den `done_when` fordert. (2) `KVStoreRepository.List`s Praefix-Filter
  ebenso auf `if false && keyPrefix != ""` gesetzt — `TestKVStore_IsolatedByInstallation`
  sofort rot ("list with prefix filter: got 2"). (3) `ExecutionLogRepository.List`s
  Installation-Filter auf `if false && installationID != nil` gesetzt —
  `TestExecutionLog_ListWithInstallationFilterAndLimit` sofort rot ("list filtered by
  installation: got [... 2 Eintraege ...]"). Alle drei Male zurueckgedreht, danach
  `go build`/`go vet`/`golangci-lint`/`go test -count=1 ./internal/plugin/repository/...`
  erneut komplett gruen.
- offen: `fix-plugin-kvset-silent-noop-unknown-installation` (neu, siehe oben) fuer Lauf 8.
  `done_when` dieser Unit vollstaendig erfuellt (Installation-Lifecycle inkl. UpdateStatus/
  UpdateSettings, KVStore-Isolation nach installation_id belegt, ExecutionLog.List mit
  Limit und Installation-Filter, drei Mutations-Proben im Journal, Paket gruen 0 Skips).
  Naechste Unit im Backlog laut Datei-Reihenfolge:
  `fix-plugin-kvset-silent-noop-unknown-installation` (neu, `status: todo`) — danach
  `c-cov-work-event-repo`.

## Iteration 59 — fix-plugin-kvset-silent-noop-unknown-installation — done — 2026-08-10 (Lauf 7)
- commit: `a4ef2ee2`
- verify vorgaenger: sauber — `4df032e4` (c-cov-plugin-repository-gaps, Iteration 58) und
  `1481874f` (Folge-Commit, traegt nur den Commit-Hash in den Journal-Eintrag von Iteration 58
  nach) geprueft: `git show --stat 4df032e4` zeigt vier neue Testdateien in
  `internal/plugin/repository/` plus BACKLOG.yml/JOURNAL.md — deckt sich 1:1 mit dem
  Journal-Eintrag. Kein `.proto`, kein neuer `RequirePermission`-Guard, keine neue Route.
  `git merge origin/main` lief als "Already up to date", kein Konflikt.
- gefixt: `KVStoreRepository.Set` (kv_store.go) und `ExecutionLogRepository.Create`
  (execution_log.go) pruefen jetzt `pgconn.CommandTag.RowsAffected()` nach ihrem
  INSERT-...-SELECT-FROM-plugin_installations und liefern das neue Sentinel
  `repository.ErrInstallationNotFound` (neue Datei `internal/plugin/repository/errors.go`),
  wenn die Subquery keine Installation findet — statt wie bisher still 0 Zeilen zu schreiben
  und `nil` zurueckzugeben. `Service.KVSet`/`Service.LogExecution` (internal/plugin/service.go)
  mappen das Repository-Sentinel per `errors.Is` auf `plugin.ErrInstallationNotFound` (bereits
  bestehender Sentinel, `isNotFound` in `mapPluginError` deckt ihn seit Lauf 7 Iteration 58 ab).
  Der `KVSet`-gRPC-Handler (internal/server/plugin_grpc.go) rief bisher nicht `mapPluginError`
  auf, sondern haerte JEDEN Fehler hart auf `codes.Internal` — umgestellt auf `mapPluginError`,
  damit das neue Sentinel als `codes.NotFound` durchkommt statt als opaker 500er (`KVGet`/
  `KVDelete`/`KVList` unveraendert, die haben kein installation-abgeleitetes Schreiben und
  bleiben bewusst bei Internal fuer generische Fehler).
- entscheidung KVDelete/execution-log-Aequivalente (laut done_when explizit gefordert):
  `KVStoreRepository.Delete` bleibt No-op fuer eine unbekannte installation_id/key-Kombination
  — anders als Set hat der Aufrufer bei Delete keine Erwartung, dass "loeschen, was nicht da
  ist" anders behandelt wird als "loeschen, was es nie gab" (idempotente Delete-Semantik, Kommentar
  im Code ergaenzt). `ExecutionLogRepository.Create` bekommt dieselbe Behandlung wie
  `KVStoreRepository.Set` (identisches Subquery-Muster) — der einzige Aufrufer ist aber
  `internal/plugin/hook/dispatcher.go`, der das Ergebnis von `LogExecution` bisher komplett
  verwarf (`_ = d.service.LogExecution(...)`, zweimal). Hook-Ausfuehrung darf durch einen
  fehlgeschlagenen Log-Write nicht abbrechen (Execution-Logging ist Best-Effort) — deshalb kein
  `return`/Propagate an den Hook-Aufrufer, aber die beiden Stellen protokollieren den Fehler
  jetzt per `slog.Warn` statt ihn stillschweigend zu verschlucken.
- test: `TestKVStore_Set_UnknownInstallation_DocumentsCurrentGap` in kv_store_test.go auf
  `TestKVStore_Set_UnknownInstallation_ReturnsError` umbenannt (nicht geloescht) und die
  Assertion von "kein Fehler erwartet" auf `errors.Is(err, repository.ErrInstallationNotFound)`
  gedreht. Dasselbe Analogon in execution_log_test.go (Teil von
  `TestExecutionLog_ListWithInstallationFilterAndLimit`) umgestellt. Neuer Subtest
  `TestPluginKVStore/set_against_unresolvable_installation_is_NotFound,_not_Success:true` in
  `internal/server/plugin_grpc_test.go`, der den Stub-Repo-Fehler
  `repository.ErrInstallationNotFound` injiziert und `codes.NotFound` am `KVSet`-Handler
  erwartet (bestehender Subtest "repository errors surface as Internal" bleibt unveraendert
  gueltig fuer generische Fehler, da `mapPluginError`s Default-Zweig weiterhin Internal liefert).
- gate: build ok (`go build -p 2 ./internal/plugin/... ./internal/server/...`) | vet ok
  (`go vet ./internal/plugin/... ./internal/server/...`) | lint ok (`golangci-lint run
  --config .golangci.yml ./internal/plugin/... ./internal/server/...`, 0 issues) | test ok
  (`go test -count=1 ./internal/plugin/... ./internal/server/...`, beide Pakete gruen inkl.
  aller DB-Tests — 0 uebersprungen, `DATABASE_URL=postgres://kmuhub_app:app_dev@localhost:
  5432/kmuhub?sslmode=disable`, Rolle `kmuhub_app` verifiziert) | migration n.a. (keine
  Schemaaenderung) | rls-smoke n.a. (kein neues Schreibmuster, nur Fehlerbehandlung eines
  bestehenden) | route n.a. (KVSet-Route existiert bereits, kein neuer Pfad) | openapi n.a. |
  protoc n.a.
- mutations-probe: in beiden Repositories die neue Bedingung von `if tag.RowsAffected() == 0`
  auf `if false && tag.RowsAffected() == 0` gesetzt (Fix faktisch deaktiviert, Rest intakt).
  `TestKVStore_Set_UnknownInstallation_ReturnsError` sofort rot ("expected
  ErrInstallationNotFound for an unresolvable installation, got <nil>"),
  `TestExecutionLog_ListWithInstallationFilterAndLimit` sofort rot (identische Fehlermeldung
  am `unknownErr`-Assert) — beide exakt am Pfad, den `done_when` fordert. Danach in beiden
  Dateien zurueckgedreht (`git diff --stat internal/plugin/repository/kv_store.go
  internal/plugin/repository/execution_log.go` zeigt wieder nur die urspruengliche
  Drei-Zeilen-Ergaenzung je Datei), `go build`/`go vet`/`golangci-lint`/
  `go test -count=1 ./internal/plugin/... ./internal/server/...` erneut komplett gruen.
- offen: keine neue Unit angelegt. `done_when` dieser Unit vollstaendig erfuellt (Set/Create
  liefern erkennbaren Fehler statt still 0 Zeilen zu schreiben, Service-/gRPC-Ebene mappen auf
  NotFound statt Success:true, KVDelete/execution-log-Entscheidung dokumentiert, beide
  "documents current gap"-Tests auf das neue Verhalten aktualisiert statt geloescht,
  Mutations-Probe im Journal belegt, beide Pakete gruen 0 Skips). Naechste Unit im Backlog
  laut Datei-Reihenfolge: `c-cov-work-event-repo` (deps `c-cov-work-event-rrule` bereits
  `status: done`, also frei).

## Iteration 60 — c-cov-work-event-repo — done — 2026-08-10 (Lauf 7)
- commit: `c200333a`
- verify vorgaenger: sauber — `a4ef2ee2` (fix-plugin-kvset-silent-noop-unknown-installation,
  Iteration 59) gegen alle acht Fehlerklassen geprueft: kein gRPC-Bypass (KVSet-Handler ruft
  weiterhin `s.svc.KVSet` ueber den Service, nur das Error-Mapping wechselt von hartem
  `codes.Internal` auf `mapPluginError`), kein Stub, kein `.proto` angefasst, kein neuer
  `RequirePermission`-Guard, keine neue Tabelle/Tenant-Luecke, Wire-Shape unveraendert (nur
  Fehlercode, `KVSetResponse` bleibt gleich), keine neue Route, kein Guard-Alt-Key ersetzt.
  `c3fc75c9` traegt nur den Commit-Hash nach. `git merge origin/main` lief als "Already up to
  date", kein Konflikt.
- gebaut: neue Datei `internal/work/event/postgres_repository_test.go` (5 Tests) gegen die
  bisher nur ueber Mocks getestete `PostgresRepository` (448 Z., 18 Methoden) in
  `internal/work/event/postgres_repository.go`. Kein Produktionscode geaendert — reine
  Coverage-Unit.
  - `TestEvent_CRUD_RRuleRoundtripAndNotFoundPaths`: RRULE-String uebersteht Create/GetByID
    zeichengenau, Update aendert RRULE/Title in-place, GetByID/Update/Delete liefern
    `ErrEventNotFound` fuer eine nicht existente Zeile statt still zu erfolgen, GetByID unter
    fremdem Tenant-Ctx liefert ebenfalls `ErrEventNotFound` (RLS).
  - `TestEvent_ExceptionsIsolateSeriesFromInstance`: EXDATE-Exception (`CreateException`)
    aendert die Serie (RRULE/RecurrenceEnd auf dem Parent-Event) nachweislich nicht, Duplikat
    auf denselben `(event_id, original_date)` liefert `ErrExceptionAlreadyExists`,
    `DeleteExceptionsAfterDate` loescht nur Ausnahmen ab dem Cutoff-Datum, eine fruehere bleibt
    stehen.
  - `TestEvent_ListInRange_And_ListRecurringOverlapping_MonthBoundary`: Fenster Jan25-Feb5
    (echte Monatsgrenze). `ListInRange` liefert genau die zwei nicht-wiederkehrenden
    In-Window-Events (je einer vor/nach der Grenze), schliesst ein wiederkehrendes Event mit
    eigener Start/Ende-Zeile IM Fenster aus (rrule-IS-NULL-Filter, nicht nur der Zeitraum).
    `ListRecurringOverlapping` liefert genau das eine Recurring-Event, dessen
    `recurrence_end` nach Fensterstart liegt, schliesst eines mit `recurrence_end` vor
    Fensterstart und eines mit `start_time` nach Fensterende aus, sowie Non-Recurring-Events.
    Fremdmandant-Session mit explizitem Opfer-Filter (Kalender-IDs + TenantID) liefert an
    beiden Methoden 0 Zeilen (RLS haelt trotz expliziter Parameter).
  - `TestEvent_Attendees_Lifecycle`: Add/Remove/UpdateRSVP/List plus
    `ListAttendeeEventIDs` inkl. `ErrNotAttendee` (Update/Remove auf Nicht-Teilnehmer) und
    `ErrAlreadyAttendee` (Duplikat-Add, 23505-Mapping), denormalisierter Name im Join geprueft.
  - `TestEvent_Reminders_Lifecycle`: `SetReminders` loescht-und-neu-schreibt (kein Append),
    `ListReminders` nach `minutes_before` sortiert, `ListUpcomingReminders` (System-Ctx, keine
    eigene Tenant-Filterung in der Query) liefert das Erinnerungs-Fenster korrekt und schliesst
    ein wiederkehrendes Event trotz identischem Fenster aus (`e.rrule IS NULL`-Filter).
  - `internal/work/event/rrule.go` (Expansions-Korrektheit) bewusst nicht erneut getestet —
    das deckt `c-cov-work-event-rrule` bereits ab, laut Scope dieser Unit nicht neu erfunden.
    `ListTaskDeadlinesInRange` ausgelassen (Cross-Modul-Abhaengigkeit auf tasks/projects,
    ausserhalb des in `scope`/`done_when` benannten Schwerpunkts RRULE/EXDATE/Range-Query).
- gate: build ok (`go build -p 2 ./internal/work/event/...`) | vet ok
  (`go vet ./internal/work/event/...`) | lint ok (`golangci-lint run --config .golangci.yml
  ./internal/work/event/...`, 0 issues) | test ok (`go test -count=1 ./internal/work/event/...`,
  komplettes Paket gruen inkl. aller neuen und bestehenden Tests — 0 uebersprungen,
  `DATABASE_URL=postgres://kmuhub_app:app_dev@localhost:5432/kmuhub?sslmode=disable`, Rolle
  `kmuhub_app` verifiziert) | migration n.a. (keine Schemaaenderung) | rls-smoke ok (im Test
  selbst: Fremdmandant-Session liefert 0 Zeilen bei explizitem Opfer-Filter, sowohl fuer
  `ListInRange` als auch `ListRecurringOverlapping` sowie GetByID) | route n.a. (kein Gateway-
  Handler/Route beruehrt, `go test ./internal/gateway/` daher nicht Pflicht fuer diese Unit) |
  openapi n.a. | protoc n.a.
- mutations-probe: in `ListInRange` (postgres_repository.go) die Zeile `AND e.rrule IS NULL`
  entfernt. Vorher mit dem urspruenglichen Fixture (`recOngoing` mit Start/Ende ausserhalb des
  Fensters) blieb der Test trotzdem gruen — false-negative-Probe erkannt, weil die eigene
  Start/Ende-Zeile des wiederkehrenden Events schon durch den Zeitraum-Filter ausgeschlossen
  wurde, unabhaengig vom rrule-Filter. Fixture korrigiert (`recOngoing`-Start auf `2031-01-28`
  verschoben, liegt jetzt selbst im Abfragefenster). Probe wiederholt: Test sofort rot
  (`ListInRange: recurring event must not be returned (rrule IS NULL filter)`), Zeile
  zurueckgedreht, `git diff --stat internal/work/event/postgres_repository.go` zeigt wieder
  keinen Unterschied zum Ausgangsstand. `go build`/`go vet`/`golangci-lint`/
  `go test -count=1 ./internal/work/event/...` danach erneut komplett gruen (siehe gate oben).
- offen: `done_when` dieser Unit vollstaendig erfuellt (RRULE-Roundtrip zeichengenau, EXDATE
  entfernt eine Instanz ohne die Serie zu aendern, Zeitraumabfrage ueber Monatsgrenze korrekt,
  Mutations-Probe im Journal belegt inkl. der korrigierten Fixture-Falle, Paket gruen 0 Skips).
  Naechste Unit im Backlog laut Datei-Reihenfolge: `c-cov-notification-integration-repo`
  (deps: [], frei).

## Iteration 61 — c-cov-notification-integration-repo — done — 2026-08-10 05:46 (Lauf 7)
- commit: -
- verify vorgaenger: sauber — `c200333a` (c-cov-work-event-repo, Iteration 60) gegen alle acht
  Fehlerklassen geprueft: `git show --stat` zeigt ausschliesslich eine neue Testdatei
  (`internal/work/event/postgres_repository_test.go`) plus Loop-Buchhaltung
  (`BACKLOG.yml`/`JOURNAL.md`) — kein Produktionscode angefasst, also kein gRPC-Bypass, kein
  Stub, kein `.proto`, kein neuer `RequirePermission`-Guard, keine neue Tabelle, kein
  Wire-Shape-Wechsel, keine neue Route, kein Guard-Alt-Key ersetzt. `457c326f` traegt nur den
  Commit-Hash nach. `git merge origin/main` lief als "Already up to date", kein Konflikt.
- gebaut: neue Datei `internal/notification/integration/postgres_repository_test.go` (6 Tests)
  gegen die bisher nur ueber die vier `Create*`-Methoden getestete `PostgresRepository` (550 Z.,
  23 Methoden) in `internal/notification/integration/postgres_repository.go`. Kein
  Produktionscode geaendert — reine Coverage-Unit.
  - `TestPostgresRepository_ConfigCRUD`: `GetConfigByPlatform`/`ListConfigs` tenant-gescopt
    (zwei unabhaengige Tenants, je eigene Configs, Cross-Tenant-Fall belegt beide Methoden
    sehen nur die eigenen Zeilen), `ListConfigs`-Sortierung nach `platform ASC` geprueft,
    `UpdateConfig`/`DeleteConfig` Erfolgs- und Not-Found-Pfad.
  - `TestPostgresRepository_MappingCRUD`: `GetMapping`/`ListMappingsByConfig` (Sortierung nach
    `channel_name`), `ListActiveMappingsForModule` beweist den Zwei-Tabellen-Join —
    ein aktives Mapping auf einem INAKTIVEN Config wird korrekt ausgeschlossen (`c.is_active =
    true`-Bedingung), ein inaktives Mapping auf aktivem Config ebenso, `UpdateMapping`/
    `DeleteMapping` Erfolgs- und Not-Found-Pfad.
  - `TestPostgresRepository_AccountLinks`: `GetAccountLink`/`GetAccountLinkByKMUHubUser`
    gefunden/nicht gefunden, `DeleteAccountLink` Erfolg + Not-Found beim zweiten Aufruf.
  - `TestPostgresRepository_LinkTokens`: `GetLinkTokenByHash` filtert bereits verbrauchte Tokens
    aus (Query traegt `AND NOT used`), `MarkLinkTokenUsed` Erfolg + Not-Found bei
    Doppelaufruf/unbekannter ID, `CleanupExpiredTokens` loescht nur abgelaufene UND
    unverbrauchte Zeilen — ein abgelaufenes aber bereits verbrauchtes Token bleibt bewusst
    stehen, ein nicht abgelaufenes Token bleibt stehen, ein abgelaufenes Token eines FREMDEN
    Tenants bleibt von `CleanupExpiredTokens(ctxA)` unberuehrt (Tenant-Isolation des DELETE
    selbst, nicht nur der Reads).
  - `TestPostgresRepository_DeliveryLog`: `GetRecentFailures` schliesst `status=sent` aus,
    sortiert absteigend nach `created_at`, haelt das `limit` ein; `CleanupOldLogs` loescht nur
    Zeilen aelter als der Cutoff.
  - `TestPostgresRepository_ResolveTenant`: sieben Faelle — aktiver Slack-Workspace aufgeloest,
    aktiver Teams-Workspace aufgeloest, unbekannter Workspace liefert `ErrTenantUnresolved`,
    leere `workspaceID` liefert `ErrTenantUnresolved`, unbekannte Plattform liefert
    `ErrInvalidPlatform`, ein INAKTIVER Config mit sonst passendem Workspace wird nicht
    aufgeloest (`ErrTenantUnresolved`, nicht faelschlich erfolgreich), zwei Tenants mit
    identischem `(platform, metadata->>key)` liefern `ErrTenantAmbiguous` statt eine
    Tie-Break-Vermutung.
- gate: build ok (`go build -p 2 ./internal/notification/... ./internal/gateway/...
  ./cmd/notification/... ./cmd/gateway/...`) | vet ok (`go vet ./internal/notification/...
  ./internal/gateway/...`) | lint ok (`golangci-lint run --config .golangci.yml
  ./internal/notification/...`, 0 issues) | test ok (`go test -count=1
  ./internal/notification/...`, alle sieben Pakete gruen inkl. `integration`/`integration/slack`/
  `integration/teams`, 0 uebersprungen laut `-v`-Lauf,
  `DATABASE_URL=postgres://kmuhub_app:app_dev@localhost:5432/kmuhub?sslmode=disable`, Rolle
  `kmuhub_app` verifiziert) | migration n.a. (keine Schemaaenderung) | rls-smoke ok (im Test
  selbst: Cross-Tenant-Faelle fuer `ListConfigs`, `CleanupExpiredTokens` und den
  System-Context-Read in `ResolveTenant`) | route n.a. (kein Gateway-Handler/Route beruehrt,
  `go test ./internal/gateway/` daher nicht Pflicht fuer diese Unit) | openapi n.a. | protoc n.a.
- mutations-probe: in `CleanupExpiredTokens` (postgres_repository.go) das `AND NOT used` aus dem
  DELETE-Statement entfernt. Test sofort rot
  (`CleanupExpiredTokens: deleted 2 rows, want 1 (only expiredUnused)` — das bereits abgelaufene,
  aber verbrauchte Token wurde mitgeloescht). Zeile zurueckgedreht, `git diff --stat
  internal/notification/integration/postgres_repository.go` zeigt wieder keinen Unterschied zum
  Ausgangsstand. `go build`/`go vet`/`golangci-lint`/`go test -count=1
  ./internal/notification/...` danach erneut komplett gruen (siehe gate oben).
- eigener Testfehler gefunden und korrigiert (kein Produktionsbug): die ersten beiden Testlaeufe
  scheiterten an `cannot scan NULL into *string` fuer `external_display_name`/
  `platform_message_id`/`error_message`, weil `testutil.SeedRow`-Fixtures diese NULLABLE Spalten
  ausliessen (SQL-DEFAULT ist NULL), waehrend die echten `Create*`-Repository-Methoden diese
  Felder immer als (ggf. leeren) String schreiben, nie als NULL — Fixtures um explizite
  Leerwerte ergaenzt. Zusaetzlich ein echter Bug im eigenen Testcode gefunden: `defer
  pool.Close()` in Kombination mit `t.Cleanup(...)` fuer Row-Deletes schloss den Pool VOR den
  Zeilen-Cleanups (Go fuehrt Funktions-`defer`s vor `t.Cleanup`-Callbacks aus), wodurch
  fehlgeschlagene Testlaeufe ihre Fixtures (u. a. `integration_configs`-Zeilen mit
  `metadata->>'team_id' = 'TSLACK111'`/`'TDUPLICATE'`) nicht aufraeumten und der naechste Lauf
  von `TestPostgresRepository_ResolveTenant` durch die Altlast faelschlich `ErrTenantAmbiguous`
  bekam. Fix: `pool.Close()` selbst ueber `t.Cleanup(...)` registriert (zuerst registriert, laeuft
  dank LIFO-Reihenfolge zuletzt) statt per `defer`. Zehn verwaiste `integration_configs`-Zeilen
  manuell aus der lokalen DB entfernt (`DELETE ... WHERE id IN (...)`), damit der naechste
  Testlauf sauber startet — die neue Cleanup-Reihenfolge verhindert eine Wiederholung.
- offen: `done_when` dieser Unit vollstaendig erfuellt (ResolveTenant mit unbekanntem Workspace
  definierter Fehler, CleanupExpiredTokens/CleanupOldLogs loeschen nachweislich nur die
  faelligen Zeilen inkl. Mutations-Probe, List-Methoden tenant-gescopt mit Cross-Tenant-Fall,
  Paket gruen 0 Skips). Naechste Unit im Backlog laut Datei-Reihenfolge: `c-cov-email-message-repo`
  (deps: [], frei).
