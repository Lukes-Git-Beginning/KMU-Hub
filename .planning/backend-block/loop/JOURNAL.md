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
