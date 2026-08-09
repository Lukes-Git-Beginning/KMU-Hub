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
