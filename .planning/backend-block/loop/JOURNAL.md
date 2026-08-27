# Backend-Nachtloop — Journal Lauf 12

Append-only. Ein Eintrag je Iteration, **ans Dateiende**, nie einsortieren. Form und Pflichtzeilen
stehen in `ITERATION.md` Schritt 6.

Frühere Läufe liegen vollständig im Archiv:
`archive/lauf-1-2/` (58 Units) · `archive/lauf-3/` (61) · `archive/lauf-4/` (54) ·
`archive/lauf-5/` (41) · `archive/lauf-6/` (46) · `archive/lauf-7/` (71) ·
`archive/lauf-8/` (94) · `archive/lauf-9/` (37) · `archive/lauf-10/` (93) ·
`archive/lauf-11/` (131 Einträge, davon 121 done — inkl. Laufbilanz am Dateiende).

---

## Laufkontext

- **Ausgangspunkt:** Lauf 11 gemergt als `acc48aee` und deployt. `main` = `backend-loop`,
  Fast-Forward, **nicht** rebased. CI grün auf `acc48aee` (Run 32735558575, alle fünf Jobs,
  57 Steps), CD grün, Produktion antwortet mit `commit: acc48aee`.
- **Migrationen:** Repo-Kopf = lokaler DB-Kopf = Produktionskopf = **323**, `schema_migrations`
  clean. Nächste freie Nummer wäre 324 — aber immer zur Laufzeit ermitteln:
  `ls backend/migrations | grep -E "^[0-9]{6}" | sort | tail -1`.
  Zwei Units dieses Laufs bringen eine Migration mit (`harden-quote-conversion-unique-index`,
  ggf. `feat-retention-handler-guest-sessions`). Entsteht im Lauf eine weitere, gilt:
  `tenant_id UUID NOT NULL` + RLS-Policy oder ein Eintrag in der System-Global-Liste,
  kein stiller dritter Weg.
- **Lokale DB:** Startbedingung. `run-loop.ps1` prüft im Vorflug Port 5432, die Anmeldung als
  `kmuhub_app` und den Migrationskopf und bricht ab, wenn eins davon fehlt. Grund:
  `testutil.SkipIfNoDB` (`backend/internal/testutil/rls.go:24`) fragt nur, **ob**
  `DATABASE_URL` gesetzt ist — ohne die Variable meldet ein Paket `ok` für Tests, die gar nicht
  gelaufen sind.
- **Rolle:** `kmuhub_app` (NOSUPERUSER NOBYPASSRLS), niemals `kmuhub` — der Superuser hat
  BYPASSRLS und würde jede RLS-Lücke durchwinken.
- **Coverage-Ausgangslage** (CI-Run 32735558575, Artefakt 9523355904): gesamt **64,1 %** bei
  Gate 15 %. Schwächste Kernpakete nach ungedeckten Zeilen: `internal/gateway` **56,6 %**
  (9.882), `internal/server` 70,8 (5.941), `internal/security/gdpr` 72,2 (626),
  `internal/auth` 67,9 (515), `internal/caldav` 54,9 (494), `internal/fuhrpark` 54,5 (464),
  `internal/formulare` 53,6 (381). Vollständige Liste im Kopf von `BACKLOG.yml`.
  **`coverage_start` in einer Unit ist ein Bezugswert, keine Messung** — jede Coverage-Unit
  misst ihr Paket vorher selbst (Lehre aus Lauf 11 Iteration 75).
- **Umfang:** 70 Units vorab, 9 davon auf `opus`. Block A (10 entschiedene) · Block A2 (5, G1-Rest und Funde) ·
  Block B (38, Nicht-Geld-Module end-to-end) · Block C (7, Sicherheits- und Kernflächen) ·
  Block D (10, Muster-Scans). Block D legt weitere Units zur Laufzeit an; in Lauf 10 haben
  9 Scans 45 Zusatz-Units erzeugt, in Lauf 11 waren es 10 Scans.
  Fenster bis 07:30, `-MaxIterations 130`.

## Der rote Faden

**NICHT-GELD-MODULE END-TO-END.** Entscheidung Luke, 2026-08-24. Lauf 11 hat die Geldpfade
abgearbeitet (payment 46,4 → 85,3 · dunning 61,8 → 92,2 · invoice 34,8 → 61,1); übrig geblieben
sind genau die Module, die dabei nicht drankamen — und die sind jetzt die schwächsten Flächen
im Backend. Je Unit **eine Domäne durch alle Schichten** (Route → gRPC → Service → Repo), als
**Bug-Suche** geschnitten, nicht als Coverage-Übung.

Warum nicht G1 oder G2 wie im Entwurf vorgesehen: beide Gates sind backend-seitig fast
abgeräumt. Von 16 G1-Punkten sind 7 erledigt, vom Rest kann der Loop genau einen bauen — alles
Übrige ist Frontend, `deploy/`, Ops oder Legal. Die Gegenprüfung steht als Befunde 1 bis 10 im
Kopf von `BACKLOG.yml`.

## Was in diesem Lauf gilt

- **Zehn Prämissen des Entwurfs haben die Gegenprüfung nicht überstanden** und stehen als
  Befunde 1 bis 10 im Kopf von `BACKLOG.yml`. Vor dem Bauen lesen — fünf davon sind Dinge, die
  bereits fertig im Code stehen (`/health` mit Postgres-Checker, Retention-Worker mit neun
  Handlern, DSAR über 39 Tabellen, GoBD-WORM per `REVOKE`, Passwort-Reset-URL). Wer sie
  trotzdem baut, baut doppelt und merkt es nicht, weil „gebaut" und „grün" dann beide stimmen.
- **Block B ist Bug-Suche, nicht Coverage.** Gesucht wird: Tenant-Scoping auf der READ-Seite,
  fehlende Fehlerpfade, Business-Logik im Handler, blinde Summierungen über Währungen,
  fehlende Indizes auf Join-Spalten. Coverage ist das Nebenprodukt und gehört in die
  `coverage:`-Zeile. Vorbild sind die beiden `idempotency`-Units aus Lauf 10 (`6507e475`,
  `254120eb`) — die erste hat sofort einen Produktionsfehler gefunden.
- **Neue DB-Tests ungetagt schreiben.** Kein `//go:build integration`. Bausteine:
  `testutil.SkipIfNoDB`, `PoolFromEnv`, `EnsureTenant`, `SeedRow`, `CleanupRow`,
  `WithTenantCtx`. Vorlage: `backend/internal/idempotency/postgres_repository_db_test.go`.
  Getaggte Tests laufen weder im lokalen Gate noch im Coverage-Job.
- **Ein DB-Test, der lokal grün ist, weil der Pool nur eine warme Verbindung hatte, beweist
  nichts.** Wer eine Ressource *pro Verbindung* prüft (Advisory Locks, Session-GUCs, temporäre
  Tabellen), hält vorher eine zweite Verbindung fest.
- **Wer ein bestehendes Muster als Vorlage kopiert, kopiert seine Fehler mit.** Vorlage vorher
  prüfen, nicht nur nachbauen.
- **`-race` läuft auf dieser Maschine nicht** (kein `gcc` im PATH). Wo eine Unit `-race`
  verlangt, ist CI der einzige Beweis — das gehört in die `offen:`-Zeile.
- **Lokales Postgres-Verbindungslimit:** `go test` über mehrere DB-Pakete mit vollem
  Parallelismus reißt mit `53300 remaining connection slots are reserved` ab. `-p 1` verwenden
  oder das Paket-Set eingrenzen. Kein Code-Fehler.
- **Die lokale Dev-Postgres trägt 13,8k Müll-Tenants** aus alten Läufen (Produktion hat 1).
  Kein Test darf über `tenants` iterieren, und jeder Test räumt auf, was er seedet.
- **Zwei Nullen sind kein bestandener RLS-Smoke, sondern ein kaputter.** Bestanden heißt eigener
  Tenant grösser 0 UND fremder Tenant gleich 0.
- Root Cause statt Symptom: vor jedem Fix alle Caller greppen. Mutations-Probe ist Pflicht.
- Eine Prämisse aus dem Backlog, die sich am Code als falsch erweist, wird **nicht trotzdem
  gebaut** — sie wird hier widerlegt und die Unit auf `blocked` gesetzt, **im selben Commit**,
  mit `blocked_reason`. Kommentarlos überspringen ist ein Fehlschlag der Iteration.
- Scan-Units ändern kein Verhalten. `neue-units:` muss IDs nennen, die wirklich in
  `BACKLOG.yml` stehen. Ein abgebrochener Scan nennt in `offen:`, **was** tief geprüft, **was**
  nur gegrept und **was** gar nicht angesehen wurde.
- Gesperrt: Frontend/Desktop, CSAT und Public-Token-Routen, `internal/biz/bexio`,
  Dependency-Bumps, Migrations-Umnummerierungen, Preise und Modul-Zuschnitt.
  `RETENTION_MODE` bleibt `dry_run`. `deploy/` ist gesperrt **bis auf zwei namentlich
  freigegebene Ausnahmen** — die Backup-Regel in `alerts.yml` und die SMTP-Vorlagen-Angleichung,
  beide als eigene Unit in Block C, beide ohne Compose-Änderung und ohne neues Env.

---

## Iteration 1 — fix-hr-manual-entry-idempotency-key-not-enforced — done — 2026-08-26 00:25
- commit: (folgt im selben Commit wie dieser Journal-Eintrag)
- gebaut: Zwei DB-gestützte Gateway-Tests
  (`internal/gateway/route_hr_manual_entry_idempotency_db_test.go`), die POST
  `/api/v1/hr/time/entries` durch die echte Kette
  `fakeAuth(idempotencyMW(HandleCreateManualEntry))` schicken (Nachbau von
  `cmd/gateway/main.go:205-206`), gegen einen echten HR-gRPC-Server (loopback
  TCP, `middleware.TenantInboundUnaryInterceptor`) mit echtem
  `timetracking.Service` + `PostgresWorkTimeRepo`. Test 1 belegt: derselbe
  Idempotency-Key erzeugt keinen zweiten `hr_work_time_entries`-Satz, zweite
  Antwort trägt `Idempotency-Replayed: true`. Test 2 belegt: ein anderer Key
  bei identischem Body erzeugt einen zweiten, unabhängigen Eintrag. Dazu ein
  `lean:`-Marker an `timetracking.ManualEntryInput.IdempotencyKey`
  (repository.go:184) — Feld wird durchgereicht, Dedup passiert eine Ebene
  höher in `middleware.Idempotency`.
- gate: build ok | vet ok | lint ok (0 issues) | test ok | migration n.a. (keine Migration) | rls-smoke n.a. (keine Tabelle/Policy angefasst)
- coverage: internal/biz/hr/timetracking 61,9 % -> 61,9 % (unverändert — der neue Test liegt im internal/gateway-Paket, nicht im gemessenen Bezugspaket; die Unit war ein Beleg-Test, kein Coverage-Ziel)
- mutations-probe: `internal/middleware/idempotency.go:145` (`w.Header().Set("Idempotency-Replayed", "true")`) auf einen falschen Header-Namen geändert (cp-Sicherung vorher), `TestHandleCreateManualEntry_SameIdempotencyKey_ReplaysInsteadOfDuplicating` wurde rot ("second request Idempotency-Replayed header = \"\", want \"true\""), per `cp` zurückgedreht, `git diff` danach leer
- verify vorgaenger: n.a. (erste Iteration dieses Laufs, kein Vorgaenger-Commit im Journal)
- neue-units: keine
- offen: Die Praemisse der Unit war widerlegt (kein echter Bug) — gebaut wurde der geforderte Regressionsbeleg. Vollständiger Gate-Lauf `go test -count=1 -p 1 -v ./internal/gateway/ ./internal/biz/hr/timetracking/...`: 2758 PASS, 0 SKIP, 0 FAIL (DATABASE_URL gesetzt, Rolle kmuhub_app). `TestOpenAPIRouteDrift` lief mit (836 Routen gegen 838 Spec-Pfade, PASS) — Unit hat keine Route angefasst, lief trotzdem zur Sicherheit mit.

## Iteration 2 — fix-event-payload-missing-tenant-id — done — 2026-08-26 00:28
- commit: (folgt im selben Commit wie dieser Journal-Eintrag)
- gebaut: Root-Cause-Fix in `event.EmitEvent`
  (`internal/notification/event/emit.go`) statt 20 Einzel-Guards: ist
  `payload.TenantID == uuid.Nil`, zieht die gemeinsame Funktion den Tenant aus
  `middleware.GetTenantID(ctx)`; ist auch dort keiner, wird der Emit mit
  `slog.WarnContext` uebersprungen und `ErrMissingTenant` (neues Sentinel)
  zurueckgegeben, statt tenant-los auf dem Bus zu landen. Dazu zwei
  DB-gestuetzte Tests (`emit_db_test.go`): Test 1 schickt ein Literal ohne
  TenantID durch die echte Kette Request-Ctx -> EmitEvent -> pg_notify ->
  `EventBus.Listen` -> `dispatch` -> Handler und belegt, dass dessen INSERT in
  die RLS-forcierte `events`-Tabelle als `kmuhub_app` durchgeht (plus
  RLS-Smoke: eigener Tenant 1 Zeile, fremder 0). Test 2 belegt den Negativfall
  gegen einen nachweislich lebenden Listener (Kontroll-Event vorher zugestellt).
- gate: build ok | vet ok | lint ok (0 issues) | test ok | migration n.a. (keine Migration) | rls-smoke ok (events, eigener Tenant 1 / fremder 0)
- coverage: internal/notification/event 61,7 % -> 80,6 % (beide selbst gemessen, `go test -count=1 -p 1 -coverprofile`; der Vorher-Wert deckt sich exakt mit `coverage_start` aus CI 32735558575)
- mutations-probe: `emit.go` `payload.TenantID = tenantID` durch `_ = tenantID` ersetzt (cp-Sicherung vorher) — `TestEmitEvent_TenantFromContextSurvivesToRLSGuardedInsert` wurde rot mit exakt dem im Scope beschriebenen Produktionsfehler: `ERROR: new row violates row-level security policy for table "events" (SQLSTATE 42501)`; per `cp` zurueckgedreht, `diff` gegen die Sicherung identisch
- verify vorgaenger: sauber (`6ea944c3` — Test-only plus `lean:`-Kommentar an `ManualEntryInput.IdempotencyKey`; keine Route, kein Proto, kein Guard, keine Tabelle, kein Stub)
- neue-units: keine
- offen: ZAEHLER wie von `done_when` verlangt — 25 Nicht-Test-Literale von `models.EventPayload`, davon setzen weiterhin 5 den Tenant selbst (die vier zeitgesteuerten Pfade + der Backlog-Replay in `bus.go:179`); die restlichen 20 laufen ausnahmslos ueber eine der 13 `PGEventEmitter`-Implementierungen, die alle `event.EmitEvent` aufrufen — nach dem Fix tragen also 25/25 Emits einen Tenant oder werden abgewiesen. Gegengeprueft: die einzigen anderen `pg_notify`-Aufrufe im Code gehen auf den Kanal `notification_delivery`, nicht auf `events`, und sind daher nicht betroffen. Kein Caller bricht ab, wenn der Emit jetzt einen Fehler liefert — alle 20 loggen ihn nur oder verwerfen ihn mit `_ =`. Testlauf ohne uebersprungene Tests: `./internal/notification/... ./internal/biz/... ./internal/crm/... ./internal/work/... ./internal/chat/... ./internal/dialer/... ./internal/document/... ./internal/email/...` und zusaetzlich `./internal/gateway/ ./internal/server/... ./internal/automation/... ./internal/fuhrpark/... ./internal/vertraege/...` alle gruen (DATABASE_URL gesetzt, Rolle `kmuhub_app`, `-p 1`, `-count=1`), inklusive `TestOpenAPIRouteDrift` — die Unit hat keine Route angefasst.

## Iteration 3 — harden-quote-conversion-unique-index — done — 2026-08-26 00:38
- commit: (folgt im selben Commit wie dieser Journal-Eintrag)
- gebaut: Migration 000324 (`finance_invoice_source_quote_id_unique`) legt einen
  partiellen Unique-Index `idx_finance_invoices_source_quote_id_unique` auf
  `finance_invoices (tenant_id, source_quote_id) WHERE source_quote_id IS NOT
  NULL AND status <> 'cancelled'` an — schliesst die Race-Luecke in
  `Service.CreateFromQuote`, deren Read-vor-Write-Check zwei wirklich
  gleichzeitige Konversionen nicht verhindern kann (lean:-Marker in
  service.go). Der Index deckt zugleich den fehlenden Zugriffspfad fuer die
  `LEFT JOIN LATERAL` in `internal/biz/quote/postgres_repository.go` (3
  Stellen) ab — Praedikat und Spalten sind identisch. `postgres_repository.go`:
  neue `isQuoteAlreadyConvertedConflict` (Vorlage `isAccountIBANConflict` in
  `internal/biz/banking`) mappt die Unique-Violation in `Create` auf
  `ErrQuoteAlreadyConverted` statt sie als rohen pg-Fehler durchzureichen —
  `internal/server/biz_grpc.go:2613` mappt den Sentinel bereits auf
  `codes.AlreadyExists` (HTTP 409), keine Aenderung dort noetig.
  `postgres_repository_quote_link_time_tracking_db_test.go`: zwei Bestandstests
  angepasst, weil sie zwei LEBENDE Rechnungen zum selben Quote anlegten (Notiz
  aus der Unit) — `..._MultipleInvoices` storniert die erste vor dem Anlegen
  der zweiten, `..._PrefersLiveInvoiceOverCancelled` legt die zweite direkt als
  `cancelled` an (Status im Model vor `Create`, statt per `UpdateStatus`
  danach), damit die Reihenfolge (juengere Zeile ist die stornierte) erhalten
  bleibt, ohne kurzzeitig zwei lebende Zeilen zu erzeugen. Neuer Test
  `TestPostgresRepository_Create_RejectsSecondLiveInvoiceForSameQuote` haelt
  direkt gegen das Repository fest: zweiter `Create`-Aufruf fuer dieselbe
  `source_quote_id` liefert `ErrQuoteAlreadyConverted`, kein SQLSTATE 23505.
- gate: build ok | vet ok | lint ok (0 issues) | test ok | migration ok
  (000324 angewandt, `migrate ... version` bestaetigt) | rls-smoke n.a. (Index
  auf Bestandstabelle, keine neue Tabelle/Policy/kein neuer tenant-gescopter
  SELECT)
- coverage: internal/biz/invoice 61,1 % -> 61,3 % (beide selbst gemessen,
  `go test -count=1 -p 1 -coverprofile`; Vorher-Messung per `git stash` auf
  den unveraenderten Dateien, danach `stash pop`)
- mutations-probe: `isQuoteAlreadyConvertedConflict` in
  `postgres_repository.go` auf einen falschen Constraint-Namen geaendert
  (cp-Sicherung vorher) — `TestPostgresRepository_Create_RejectsSecondLiveInvoiceForSameQuote`
  wurde rot ("Target error should be in err chain: expected ...
  ErrQuoteAlreadyConverted, in chain: ... duplicate key value violates unique
  constraint \"idx_finance_invoices_source_quote_id_unique\""), per `cp`
  zurueckgedreht, `diff` gegen die Sicherung identisch
- verify vorgaenger: sauber (`4e1ad006` — Root-Cause-Fix in `event.EmitEvent`,
  DB-Tests belegen beide Richtungen inkl. RLS-Smoke auf `events`; keine Route,
  kein Proto, kein Guard, keine neue Tabelle, kein Stub)
- neue-units: keine
- offen: `wire-biz-event-emitters-for-finance-triggers` (naechste Unit in
  Datei-Reihenfolge nach `harden-quote-conversion-unique-index`) zurueck nach
  `BACKLOG-NEXT.yml` verschoben — ihr eigener `done_when`-Punkt verlangt einen
  SELECT gegen die PRODUKTIONS-Automations-Tabellen (pruefen, ob Kunden schon
  Workflows mit `biz.invoice.sent`/`biz.quote.created` angelegt haben, die ab
  dem Deploy live zu feuern anfingen); der Loop hat keinen Produktionszugriff.
  `BACKLOG.yml` darf laut Kopf-Kommentar keine `blocked`-Unit fuehren (Vorflug
  bricht sonst ab) — deshalb Verschiebung statt `status: blocked` inline.
  Grund und Rueckweg stehen als Block am Ende von `BACKLOG-NEXT.yml`. Sobald
  Luke den Produktions-SELECT gefahren hat, kann die Unit unveraendert zurueck
  nach `BACKLOG.yml`. Vollstaendiger Testlauf ohne uebersprungene Tests:
  `internal/biz/invoice` (0 SKIP) und `internal/server` (0 SKIP, 1864 PASS),
  beide DATABASE_URL gegen `kmuhub_app`, `-count=1 -p 1`. Migration 000324
  lokal angewandt (Kopf 324); Produktion hat sie noch nicht.

## Iteration 4 — feat-crm-activity-deal-tag-rpcs — done — 2026-08-26 01:22
- commit: (folgt im selben Commit wie dieser Journal-Eintrag)
- gebaut: Die vier fehlenden gRPC-Hops fuer Deal- und Activity-Tags. `crm.proto`:
  `AddDealTags`/`RemoveDealTags`/`AddActivityTags`/`RemoveActivityTags` plus die
  acht Messages nach dem exakten Vorbild von `AddContactTags` (Request
  `deal_id`/`activity_id` + `repeated string tag_ids`, Response `DealInfo`/
  `ActivityInfo`); `crm.pb.go` und `crm_grpc.pb.go` im selben Commit regeneriert
  (protoc, Kommando aus dem `proto:`-Target, ein `proto-crm`-Target existiert
  nicht). `internal/server/crm_grpc.go`: vier Methoden mit
  `middleware.GetTenantID`, uuid-Parsing und `mapCRMError`, dazu die geteilte
  `parseTagIDs` — `deal.ErrTagNotFound` und `activity.ErrTagNotFound` waren in
  `mapCRMError` bereits eingetragen, deshalb 404 statt Internal ohne Zusatzarbeit.
  Die Service-Signaturen sind `(ctx, tenantID, entityID, tagIDs)`, also anders
  geordnet als bei Contacts. `route_crm_activities.go` und `route_crm_pipeline.go`:
  die vier Handler von `response.Error(501, ...)` auf `getCRMClient` ->
  `client.<RPC>` -> `response.Proto` umgestellt. Keine neue Route, keine neue
  Permission, keine Migration, kein openapi-Diff — die Spec dokumentierte fuer
  alle vier Pfade laengst `200` mit `DealResponse`/`ActivityResponse`, nur der
  Handler hielt nicht Wort. Neu `internal/server/crm_grpc_deal_activity_tags_db_test.go`
  (6 Tests): Add/Remove ueber den gRPC-Weg gegen echte Postgres-Repositories,
  Blick direkt in `deal_tags`/`activity_tags` (Zeile da, `tenant_id` = Tenant des
  Parents, nach Remove weg), Fremd-Tenant auf ein fremdes Deal bekommt
  `codes.NotFound` und schreibt keine Zeile, Tag mit falschem `entity_type` wird
  abgewiesen, plus Parse- und Tenant-Kontext-Pfade.
- gate: build ok | vet ok | lint ok (0 issues auf server+gateway+crm) | test ok
  (8340 PASS, 0 SKIP, 0 FAIL in `./internal/server/ ./internal/gateway/`,
  `./internal/crm/...` gruen) | migration n.a. (keine) | rls-smoke n.a. (keine
  neue Tabelle/Policy; die Tenant-Isolation der bestehenden `deal_tags`-Policy
  ist im Fremd-Tenant-Test mitbelegt)
- coverage: internal/server 70,8 % -> 70,9 % und internal/gateway 56,7 % -> 56,7 %
  (beide selbst gemessen, `go test -count=1 -p 1 -coverprofile`; Vorher-Wert per
  `git stash push -u -- backend/`, danach `stash pop`). Der Gateway-Wert steht
  still, weil die vier Handler von je 6 auf je 14 Statements gewachsen sind —
  die neuen Zeilen sind abgedeckt, aendern aber am Verhaeltnis nichts.
- mutations-probe: in `AddDealTags`/`RemoveDealTags` zuerst `tenantID` durch
  `uuid.New()` ersetzt — das ist KEINE gueltige Probe, es bricht schon den Build
  ("declared and not used"). Stattdessen `RemoveDealTags` auf
  `RemoveTags(ctx, tenantID, dealID, tagIDs[:0])` mutiert (cp-Sicherung vorher):
  `TestCRMGRPCServer_DealTags_AddThenRemoveHitsJoinTable` wurde rot
  ("deal_tags row still present after RemoveDealTags", erwartet uuid.Nil, bekam
  den Tenant), per `cp` zurueckgedreht, `diff` gegen die Sicherung identisch,
  Test danach wieder gruen.
- verify vorgaenger: sauber (`27e29ddb` — Migration 000324 ist neu und forward-only,
  kein `.proto` im Diff, keine Route, kein `RequirePermission`, keine neue Tabelle;
  `isQuoteAlreadyConvertedConflict` mappt eine echte Unique-Violation, kein Stub,
  und `Create` reicht jeden anderen Fehler unveraendert durch)
- neue-units: keine
- offen: (1) Die vier 501-Eintraege sind aus `statusDriftBaseline`
  (`openapi_status_code_drift_test.go`) raus — dort steht jetzt nur noch
  `PUT /api/v1/customization/labels: {500}`. (2) Der Kommentar-Block in
  `route_capability_guard_test.go` und die Erwartung "deal tags add with
  deals:write" mussten von `http.StatusBadRequest` auf `allowed` (503) wechseln:
  der Handler holt sich jetzt zuerst den Client und liest erst danach den Body,
  wie alle anderen verdrahteten Handler auch. Aus demselben Grund brauchen die
  Validierungs-Tests der vier Handler `registryWithService("crm")` statt
  `emptyRegistry()` — mit leerer Registry kommen sie gar nicht mehr bis zur
  Validierung. (3) FE-seitig folgenlos: `openapi.yaml` ist unveraendert, also
  aendern sich die generierten Typen in `desktop/src/renderer/src/api/types.ts`
  nicht; einen Aufrufer fuer die vier Endpunkte gibt es weiterhin nicht (nur
  Kontakte und Firmen haben Tag-UI). Wer die UI nachzieht, findet die Antwort
  jetzt gewrappt als `{"deal": {...}}` bzw. `{"activity": {...}}`.

## Iteration 5 — feat-lexware-store-organization-id-on-connect — done — 2026-08-26 01:01
- commit: (folgt im selben Commit wie dieser Journal-Eintrag)
- gebaut: `Service.Connect` (`internal/biz/lexware/service.go`) ruft nach dem
  Aufbau der `IntegrationConfig` und vor `configRepo.Upsert` best-effort
  `s.client.GetProfile(ctx, tenantID)` auf (nur wenn `s.client != nil` — sonst
  Nil-Pointer-Panic in `client.do`, betrifft `newTestService` mit `client: nil`).
  Liefert die API eine nicht-leere `OrganizationID`, landet sie als
  `metadata["organization_id"]`; schlägt der Aufruf fehl, wird das per
  `slog.Warn` protokolliert und `Connect` läuft normal weiter — ein
  Profil-Timeout darf keine Lexware-Verbindung verhindern. `TestConnection`
  liest zusätzlich das zurückgegebene Profil aus (vorher verworfen) und trägt
  eine fehlende `organization_id` per erneutem `configRepo.Upsert` nach, aber
  nur wenn der Schlüssel wirklich fehlt (kein Upsert bei vorhandenem Wert).
  `lean:`-Marker am Metadata-Schreiben nennt Zweck (Tenant-Auflösung im
  Webhook-Pfad) und Upgrade-Trigger (zweiter aktiver Lexware-Tenant, siehe
  `harden-lexware-webhook-organization-id-scoping` in BACKLOG-PARKED.yml).
  Keine Migration (JSONB-Metadata ist additiv), kein Proto, keine Route.
  Vier neue Tests in `service_wiring_test.go` über die bestehende
  `stubAPI`/`newWiredService`-Infrastruktur (echter `*Client` gegen
  `httptest.NewServer`, kein Mock auf Interface-Ebene, weil `Service.client`
  konkret `*Client` ist): `TestConnect_StoresOrganizationID`,
  `TestConnect_ProfileFetchFailsStillConnects` (401 statt 5xx, sonst brennt
  der Retry/Backoff unnötig Testzeit), `TestTestConnection_BackfillsMissingOrganizationID`,
  `TestTestConnection_DoesNotReupsertExistingOrganizationID`.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (gesamtes Paket
  `./internal/biz/lexware/` grün, keine neuen Fehlschläge) | migration n.a.
  (keine) | rls-smoke n.a. (keine neue Tabelle/Policy, nur eine JSONB-Spalte
  auf einer bestehenden, RLS-geschützten Tabelle beschrieben)
- coverage: internal/biz/lexware 74,5 % -> 74,8 % (beide selbst gemessen,
  `go test -count=1 -coverprofile`; Vorher-Wert per `git stash push -u --
  backend/internal/biz/lexware/`, danach `stash pop` — deckt sich mit
  `coverage_start` aus der Unit)
- mutations-probe: in `Connect` die Zeile `config.Metadata["organization_id"]
  = profile.OrganizationID` durch `_ = profile.OrganizationID` ersetzt (cp
  gesichert vorher). `TestConnect_StoresOrganizationID` wurde rot
  ("expected: org-42, actual: nil"), Datei per `cp` zurückgedreht,
  `git diff` gegen den Ausgangsstand danach leer, kompletter Testlauf wieder
  grün.
- verify vorgaenger: sauber (`fdbeab8a` — Handler gehen über `client.<RPC>`
  aus dem gRPC-Client, nicht über eine direkt injizierte Service-Instanz;
  `.proto`-Änderung mit regenerierten `crm.pb.go`/`crm_grpc.pb.go` im selben
  Commit; Tenant kommt serverseitig aus `middleware.GetTenantID`; keine neue
  Route, keine neue Permission, keine neue Tabelle, kein Stub)
- neue-units: keine
- offen: keine — der geparkte Webhook-Fix
  (`harden-lexware-webhook-organization-id-scoping`) bleibt bewusst liegen,
  bis ein zweiter Lexware-Tenant aktiv wird; diese Unit legt nur den Wert an.

## Iteration 6 — feat-retention-handler-fuhrpark-operational — done — 2026-08-26 01:08
- commit: (folgt im selben Commit wie dieser Journal-Eintrag)
- gebaut: Zwei neue Retention-Handler nach dem Muster von `retention_invitations.go`,
  registriert in `cmd/auth/main.go`. `VehicleBookingRetentionHandler`
  (`retention_vehicle_bookings.go`, resource_type `vehicle_bookings`): delete-only,
  Clock `ends_at` — eine abgelaufene Fahrzeugbuchung hat ohne den Fahrer keinen
  Rest-Nutzen, eine laufende Buchung (altes `starts_at`, aber `ends_at` in der
  Zukunft) landet nie in `Due`. `DriverLicenseRetentionHandler`
  (`retention_driver_licenses.go`, resource_type `driver_licenses`): delete-only,
  Clock `checked_at`, mit Schutz der aktuellen Kontrollzeile — `Plan` laeuft ein
  `ROW_NUMBER() OVER (PARTITION BY driver_id ORDER BY checked_at DESC, id DESC)`
  ueber ALLE Zeilen des Tenants (nicht nur die faelligen), damit auch ein Fahrer
  mit nur einer, jahrzehntealten Kontrollzeile diese behaelt. Die jeweils
  juengste Zeile je `driver_id` landet unabhaengig vom Alter in `Skipped` statt
  `Due`, mit Begruendung ("Nachweis der Halterpflicht"). Beide Handler nur
  `RetentionActionDelete` in `SupportsAction`. Keine Migration (beide Tabellen
  tragen bereits RLS aus ihren eigenen Migrationen 000300/000279), kein Proto,
  keine Route.
- gate: build ok | vet ok | lint ok (0 issues, `./internal/security/gdpr/...
  ./cmd/auth/...`) | test ok (gesamtes Paket `./internal/security/gdpr/` gruen,
  9 neue Tests, keine Skips) | migration n.a. (keine — Migrationskopf lokal
  bereits 324, deckungsgleich mit beiden Zieltabellen) | rls-smoke n.a. (keine
  neue Tabelle/Policy, beide Tabellen bereits RLS-geschuetzt seit ihrer
  jeweiligen Erst-Migration)
- coverage: internal/security/gdpr 72,2 % -> 72,4 % (beide selbst gemessen,
  `go test -count=1 -coverprofile`; Vorher-Wert per `git stash push -u --
  backend/internal/security/gdpr/ backend/cmd/auth/`, danach `stash pop` —
  deckt sich mit `coverage_start` aus der Unit)
- mutations-probe: in `DriverLicenseRetentionHandler.Plan` den
  `if isLatest { ... Skipped ...; continue }`-Block entfernt, sodass jede
  faellige Zeile unabhaengig vom `is_latest`-Flag nach `Due` faellt (Datei
  vorher per `cp` gesichert). `TestDriverLicenseRetentionHandler_
  PlanKeepsLatestPerDriverEvenWhenOnlyRowIsAncient` wurde rot ("Should be
  empty, but was [...]" — die einzige, jahrzehntealte Kontrollzeile eines
  Fahrers landete faelschlich in `Due`), Datei per `cp` zurueckgedreht, `diff`
  gegen die Sicherung identisch, kompletter Testlauf danach wieder gruen.
- verify vorgaenger: sauber (`a0f28af9` — kein gRPC-Bypass, kein Stub, kein
  `.proto` im Diff, keine Route, kein `RequirePermission`, keine neue Tabelle;
  `Connect` ruft `GetProfile` best-effort mit `slog.Warn` bei Fehlschlag,
  `TestConnection` traegt eine fehlende `organization_id` additiv nach)
- neue-units: keine
- offen: keine — `trip_logs` ist bewusst ausgeklammert (siehe `decision` der
  Unit, gehoert zu `decide-retention-policy-hgb-ao-domains`, Etappe 4/Legal).

## Iteration 7 — harden-advisory-protocols-retention-guard — done — 2026-08-26 01:14
- commit: (folgt im selben Commit wie dieser Journal-Eintrag)
- gebaut: `AdvisoryProtocolRetentionHandler` (`retention_advisory_protocols.go`),
  registriert in `cmd/auth/main.go`. `SupportsAction` gibt fuer JEDE Aktion
  `false` zurueck — bewusst kein Loesch- oder Anonymisierungspfad, weil
  `advisory_protocols` einer 10-Jahres-Aufbewahrungspflicht nach §18a
  FinVermV (i.V.m. MiFID II, §64 WpHG, §§16-18 FinVermV, §61 VVG/IDD)
  unterliegt. Damit eine `retention_policies`-Regel auf diesem
  `resource_type` nicht mehr als generisches "beherrscht die Aktion nicht"
  oder gar als "nicht zugeordnet" im Run-Report landet, bekommt die Engine
  einen neuen optionalen Hook: `unsupportedReasoner` (`retention.go`,
  Interface mit `UnsupportedReason(action string) string`). Implementiert ein
  Handler es, ersetzt `runPolicy` die generische Message durch den
  handler-eigenen Text — hier die Rechtsgrundlage plus "10-Jahres"-Frist,
  wortwoertlich pruefbar im Run-Item. Kein anderer der zehn bestehenden
  Handler implementiert den Hook, ihr Verhalten aendert sich nicht.
  `Plan` zaehlt echte Kandidaten (finalisierte Protokolle mit
  `handed_over_at < cutoff`, `tenant_id`-gescoped) fuer eine spaetere
  Admin-Auswertung — die Engine ruft `Plan` fuer diesen Handler in der
  Praxis nie auf, weil `SupportsAction` immer vorher abbricht; die Zaehlung
  ist trotzdem echt und direkt getestet, nicht totgelegter Code, sondern die
  Grundlage fuer einen spaeteren Loeschpfad. Ein Draft (`handed_over_at
  IS NULL`) startet die Frist nicht und taucht nie in `Due` auf. `Apply`
  liefert immer einen Fehler mit derselben Begruendung — Verteidigung falls
  die Engine je ohne den `SupportsAction`-Gate aufgerufen wird.
  `lean:`-Marker im Dateikopf nennt den Upgrade-Trigger (erster Bestand
  aelter als 10 Jahre). Keine Migration (keine neue Tabelle/Policy noetig,
  `advisory_protocols` traegt RLS seit Migration 000137), kein Proto, keine
  Route.
- gate: build ok | vet ok | lint ok (0 issues, `./internal/security/gdpr/...
  ./cmd/auth/...`) | test ok (gesamtes Paket `./internal/security/gdpr/`
  gruen, 177 Tests, 0 Skips, 0 Fails) | migration n.a. (keine) | rls-smoke
  n.a. (keine neue Tabelle/Policy — advisory_protocols ist seit seiner
  Erst-Migration RLS-geschuetzt, der Handler liest/schreibt ausschliesslich
  tenant-gescoped)
- coverage: internal/security/gdpr 72,36 % -> 72,49 % (beide selbst
  gemessen via `go tool cover -func` auf echten Statement-Zaehlern, nicht
  nur der gerundeten Prozentzahl; Vorher-Wert per `git stash push -u --
  backend/internal/security/gdpr backend/cmd/auth/main.go`, danach `stash
  pop` — deckt sich mit `coverage_start` aus der Unit, 72,2 % CI-Stand war
  vor Iteration 6 gemessen). Neue Datei bei 76,9 % (nur der ungetestete
  Scan-Fehlerpfad in Plan bleibt offen, wie bei allen Geschwister-Handlern).
- mutations-probe: in `retention.go:runPolicy` den `if reasoner, ok :=
  handler.(unsupportedReasoner); ok { item.Message = ... }`-Block entfernt
  (Datei vorher per `cp` gesichert). `TestAdvisoryProtocolRetentionHandler_
  Run_ReportsUnsupportedWithLegalReasonNotUnmapped` wurde rot (Message
  enthielt weder "FinVermV" noch "10-Jahres", nur die generische Formel),
  Datei per `cp` zurueckgedreht, `diff` gegen die Sicherung identisch,
  kompletter Testlauf danach wieder gruen.
- verify vorgaenger: sauber (`aa44437a` — kein gRPC-Bypass, kein Stub, kein
  `.proto` im Diff, keine Route, kein `RequirePermission`, keine neue
  Tabelle/Migration; beide neuen Retention-Handler lesen/schreiben
  tenant-gescoped, `DriverLicenseRetentionHandler` schuetzt die jeweils
  juengste Kontrollzeile je `driver_id` nachweislich per Mutations-Probe)
- neue-units: keine
- offen: keine — der `unsupportedReasoner`-Hook ist additiv (Go-Idiom
  optionales Interface, kein Interface-Bruch fuer die zehn bestehenden
  Handler); ein spaeterer aktiver Loeschpfad fuer `advisory_protocols`
  bleibt bewusst ungebaut, siehe `lean:`-Marker im Handler.

## Iteration 8 — feat-retention-handler-guest-sessions — done — 2026-08-26 01:23
- commit: (folgt im selben Commit wie dieser Journal-Eintrag)
- gebaut: `GuestSessionRetentionHandler` (`retention_guest_sessions.go`),
  registriert in `cmd/auth/main.go`. Delete-only nach Entscheidung Luke
  2026-08-24 (A6 Teilentscheidung): 90 Tage nach `last_activity_at`, nicht
  `created_at` (sonst faellig mitten im Gespraech) und nicht `expires_at`
  (das betrifft nur das Token). `Plan` filtert `tenant_id` + `last_activity_at
  < cutoff` UNABHAENGIG von `is_active` — eine aktive, aber seit Monaten
  stille Sitzung ist genauso faellig wie eine inaktive, das ist im Test
  explizit belegt (`old-active` und `old-inactive` beide im Plan, `fresh-active`
  nicht). Migration `000325_guest_sessions_retention_index` legt
  `idx_guest_sessions_tenant_last_activity (tenant_id, last_activity_at)`
  an — der bestehende `idx_guest_sessions_cleanup` (`expires_at WHERE
  is_active = true`, seit Migration 000054 nie von einem Cleanup genutzt)
  passt nicht zur Handler-Query, weil er `is_active` erzwingt und die
  falsche Spalte indiziert. Chat-Nachrichten des Gastes sind explizit NICHT
  Teil dieser Unit (siehe `notes` der Unit) — `messages.guest_session_id`
  hat `ON DELETE SET NULL` (Migration 000054), die Nachricht bleibt also
  bei geloeschter Sitzung erhalten, nur die Zuordnung geht verloren.
- gate: build ok | vet ok | lint ok (0 issues, `./internal/security/gdpr/...
  ./cmd/auth/...`) | test ok (gesamtes Paket `./internal/security/gdpr/`
  gruen, DATABASE_URL gesetzt als kmuhub_app, 0 Skips) | migration ok
  (000325 up gegen lokale DB angewendet, `migrate ... up` lief sauber) |
  rls-smoke n.a. (keine neue Tabelle/Policy — nur ein Index auf einer
  Tabelle, die seit Migration 000122 RLS-geschuetzt ist; Handler
  liest/schreibt ausschliesslich tenant-gescoped per Query-Parameter)
- coverage: internal/security/gdpr 72,5 % -> 72,6 % (beide selbst gemessen
  via `go tool cover -func`, Vorher-Wert per `git stash push -u --
  backend/internal/security/gdpr backend/cmd/auth/main.go` + `stash pop`;
  deckt sich mit `coverage_start` aus der Unit, 72,2 % CI-Stand war vor
  Iteration 7 gemessen)
- mutations-probe: in `retention_guest_sessions.go:Plan` die WHERE-Klausel
  um `AND is_active = true` erweitert (Datei vorher per `cp` gesichert).
  `TestGuestSessionRetentionHandler_PlanUsesLastActivityRegardlessOfIsActiveAndIsTenantScoped`
  wurde rot (die `old-inactive`-Sitzung fehlte im Plan), Datei per `cp`
  zurueckgedreht, `diff` gegen die Sicherung identisch, kompletter
  Testlauf danach wieder gruen.
- verify vorgaenger: sauber (`52a1332b` — kein gRPC-Bypass, kein Stub, kein
  `.proto` im Diff, keine Route, kein `RequirePermission`, keine neue
  Tabelle/Migration; `AdvisoryProtocolRetentionHandler` liest/schreibt
  tenant-gescoped, `unsupportedReasoner`-Hook additiv ohne bestehende
  Handler zu veraendern)
- neue-units: keine
- offen: keine — kein Route-Impact, `go test ./internal/gateway/` daher
  nicht Pflicht und nicht gelaufen.

## Iteration 10 — feat-scrub-dependent-pii-dialer-tables — done — 2026-08-26 01:52
- commit: (folgt im selben Commit wie dieser Journal-Eintrag)
- gebaut: `ScrubDependentPII` (`internal/crm/consent/scrub.go`) um zwei
  tenant-gescopte `tx.Exec`-Bloecke erweitert: `dialer_campaign_contacts.notes`
  (direkter `contact_id`) und `dialer_call_sessions.notes`/`.next_action`
  (ueber Subselect auf `dialer_campaign_contacts`, weil die Tabelle keine
  eigene `contact_id`-Spalte hat, nur `campaign_contact_id`). Beide Bloecke
  zaehlen in `affected` mit, wie die bestehenden. Godoc-Block um die neu
  abgedeckten Tabellen erweitert ("Also scrubbed: ..."). Keine Migration,
  kein Proto, keine Route — reine Erweiterung einer bestehenden Funktion.
  Beim Recherchieren festgestellt: `dialer_campaign_contacts.contact_id` traegt
  `ON DELETE RESTRICT` (Migration 000130) — ein Kontakt mit Dialer-Historie
  kann `contact.PostgresRepository.Delete` (Hard-Delete-Pfad) nie erfolgreich
  durchlaufen, das scheitert an der FK, bevor committet wird (siehe `IsInUse`).
  Der Dialer-Scrub laesst sich deshalb nur ueber `AnonymizeContact` beobachten,
  nicht ueber den Hard-Delete-Pfad — dokumentiert in beiden neuen Testdateien
  und unten unter `offen:`.
- gate: build ok (`./internal/crm/... ./cmd/gateway/...`) | vet ok
  (`./internal/crm/...`) | lint ok (0 issues, `./internal/crm/...`) | test ok
  (siehe unten) | migration n.a. (keine neue Tabelle/Spalte, reine
  UPDATE-Statements auf bestehenden RLS-geschuetzten Tabellen) | rls-smoke
  n.a. (keine neue Tabelle/Policy — beide Tabellen sind seit Migration 000120
  RLS-geschuetzt, beide neuen Statements filtern `tenant_id` explizit, auch
  im Subselect)
- coverage: internal/crm/consent 64,0 % -> 64,5 % (`go tool cover -func`,
  Vorher-Wert deckt sich mit `coverage_start` aus der Unit — CI-Stand
  32735558575, seit Laufbeginn unangetastet)
- mutations-probe: in der `dialer_call_sessions`-UPDATE das `next_action = NULL`
  entfernt (Datei vorher per `cp` gesichert). `TestAnonymizeContact_
  ScrubsDialerCampaignAndCallSessionNotes` wurde rot ("expected
  dialer_call_sessions.notes/.next_action to be scrubbed, got
  notes=<nil> next_action=0xc00038b160"), Datei per `cp` zurueckgedreht,
  `diff` gegen die Sicherung identisch, kompletter Testlauf danach wieder
  gruen.
- verify vorgaenger: sauber (`66f7341f` — kein gRPC-Bypass, kein Stub, kein
  `.proto` im Diff, keine Route, kein `RequirePermission`, Migration 000325
  legt nur einen Index an keine neue Tabelle; `GuestSessionRetentionHandler`
  liest/schreibt ausschliesslich tenant-gescoped ueber Query-Parameter,
  registriert korrekt in `cmd/auth/main.go`)
- neue-units: keine
- offen: Der Hard-Delete-Pfad (`contact.PostgresRepository.Delete`) kann den
  Dialer-Scrub strukturell nicht demonstrieren — ein Kontakt mit
  `dialer_campaign_contacts`-Zeilen loest beim `DELETE FROM contacts` die
  `ON DELETE RESTRICT`-FK aus (Migration 000130) und die ganze Transaktion
  rollt zurueck, inklusive des vorher gelaufenen Scrubs. Statt des in
  `done_when` vorgesehenen "Delete scrubbt sichtbar"-Tests belegt
  `TestDelete_DialerCampaignContactRestrictsHardDeleteWithoutPartialScrub`
  (Paket `contact`) die Grenze: Delete schlaegt sauber fehl, keine
  Teil-Scrubbung bleibt zurueck. `TestAnonymizeContact_
  ScrubsDialerCampaignAndCallSessionNotes` (Paket `consent`) deckt den
  einzigen tatsaechlich erreichbaren Pfad End-to-End inklusive
  Tenant-Negativtest ab. Kein Route-Impact, `go test ./internal/gateway/`
  daher nicht Pflicht und nicht gelaufen.

## Iteration 12 — feat-scrub-dependent-pii-inbox-rentals-contracts — done — 2026-08-26 02:04
- commit: (folgt im selben Commit wie dieser Journal-Eintrag)
- verify vorgaenger: sauber (`232ec86d` — kein gRPC-Bypass, kein Stub, kein
  `.proto` im Diff, keine Route, kein `RequirePermission`, keine neue
  Tabelle/Migration; die beiden neuen `tx.Exec`-Bloecke in `ScrubDependentPII`
  filtern `tenant_id` in JEDEM Statement, auch im Subselect fuer
  `dialer_call_sessions`; DB-Test mit echtem Tenant-Negativfall belegt)
- vorab-befund (nicht der Verify-Vorspann, ein separater Fund beim Ziehen der
  naechsten Unit): `fix-409-double-meaning-on-grpc-conflict-routes` stand seit
  einer nicht journalisierten Iteration (Luecke zwischen den Journal-Eintraegen
  8 und 10) auf `status: in_progress`, ohne jede Code-Aenderung (`git status`
  war beim Start dieser Iteration clean). Recherche zeigt: die Unit-Praemisse
  war falsch. Die 58 `codes.AlreadyExists`-Fundstellen sind keine 58 einzelnen
  RPC-Handler, sondern liegen fast durchgehend in 38 geteilten
  Error-Mapper-Funktionen (`mapCalendarError`, `mapCRMError`, `mapBizError`,
  ...), die jeweils von vielen RPCs aufgerufen werden — gemessen:
  `mapCalendarError(err)` 41x in `calendar_grpc.go`, `mapCRMError(err)` 60x in
  `crm_grpc.go`, `mapBizError(err)` 57x in `biz_grpc.go`. Die vom `done_when`
  verlangte, belegte (nicht geschaetzte) Operationenliste erfordert deshalb je
  Sentinel-Error eine Rueckverfolgung durch die Service-Schicht bis zu den
  RPC-Einstiegspunkten — eine Groessenordnung ueber eine Iteration hinaus. Die
  Unit ist auf `status: blocked` mit ausfuehrlichem `blocked_reason` gesetzt
  (Empfehlung darin: als Block-D-Scan neu zuschneiden statt Recherche+Fix in
  einer Unit).
- gebaut (die eigentliche Unit dieser Iteration): `ScrubDependentPII`
  (`internal/crm/consent/scrub.go`) um zwei weitere tenant-gescopte
  `tx.Exec`-Bloecke erweitert: `inbox_messages.sender_name/.sender_email/
  .preview` (ueber `crm_contact_id` — abweichender Spaltenname, kein
  `contact_id`) und `rentals.renter_name/.notes` (direkter `contact_id`).
  `contract_parties.external_name` NICHT gescrubbt — am Code verifiziert,
  nicht angenommen: `AddParty` (`vertraege/service.go:392-398`) fuellt
  `external_name` ausschliesslich bei `party_type = 'external'`, und eine
  Zeile mit `party_type = 'external'` traegt nie einen `contact_id` (das ist
  der Zweck der Spalte — Platzhalter fuer eine Partei ohne CRM-Kontakt). Ein
  `WHERE contact_id = $1`-Filter trifft also strukturell nie eine Zeile mit
  gefuelltem `external_name`; die in der Unit erwartete Pruefung auf
  Aufbewahrungsfristen (Paragraph 147 AO) erledigt sich dadurch, es gibt
  nichts zu scrubben. Der Godoc-Block nennt jetzt beide neuen Tabellen und die
  `contract_parties`-Ausnahme mit Begruendung.
  Nebenfund beim Verifizieren von `inbox_messages.preview` (laut Unit-Notes
  ein abgeleitetes Feld aus `email_messages`): `email_messages`
  (`from_name`, `from_email`, `body_text`, `body_html`, `raw_headers`) bleibt
  beim Anonymisieren unangetastet, verlinkt ueber `email_contact_links`
  (`ON DELETE CASCADE`, bleibt beim Anonymisieren aber bestehen, weil die
  Kontaktzeile nicht geloescht wird). Das ist keine Iterations-Entscheidung,
  sondern beruehrt moeglicherweise Paragraph 257 HGB
  (Aufbewahrungspflicht Handelsbriefe) — als `decide-`-Unit angelegt, siehe
  `neue-units`.
- gate: build ok (`./internal/crm/... ./cmd/gateway/...`) | vet ok
  (`./internal/crm/...`) | lint ok (0 issues, `./internal/crm/...`) | test ok
  (32 Tests im Paket `consent`, 0 Skips, `DATABASE_URL` gesetzt als
  `kmuhub_app`; gesamtes `./internal/crm/...` mit `-p 1` seriell gruen — mit
  Default-Parallelitaet schlugen mehrere fremde Pakete mit
  "remaining connection slots are reserved for roles with the SUPERUSER
  attribute" fehl, reine lokale `max_connections`-Erschoepfung durch
  gleichzeitig geoeffnete Pools, keine Regression durch diese Aenderung —
  seriell bestaetigt gruen) | migration n.a. (keine neue Tabelle/Spalte, reine
  UPDATE-Statements auf bestehenden RLS-geschuetzten Tabellen) | rls-smoke
  n.a. (keine neue Tabelle/Policy — `inbox_messages` RLS seit Migration
  000122, `rentals` seit 000122 via `enable_tenant_rls('rentals')`; beide
  neuen Statements filtern `tenant_id` explizit)
- coverage: internal/crm/consent 64,5 % -> 64,9 % (`go tool cover -func`,
  Vorher-Wert per `git stash push -u -- backend/internal/crm/consent/` +
  `stash pop` selbst gemessen — deckt sich mit dem Stand nach Iteration 10,
  `coverage_start` der Unit war der aeltere CI-Stand 64,0 % vor Iteration 10)
- mutations-probe: in der `rentals`-UPDATE `renter_name = ''` entfernt (Datei
  vorher per `cp` gesichert). `TestAnonymizeContact_
  ScrubsInboxMessageAndRentalIdentity` wurde rot ("expected rentals renter
  identity to be scrubbed, got renter_name=\"Inbox ScrubTest\""), Datei per
  `cp` zurueckgedreht, `diff` gegen die Sicherung identisch, kompletter
  Testlauf danach wieder gruen.
- neue-units: `decide-email-messages-contact-pii-on-anonymize` (in
  BACKLOG-NEXT.yml, nicht BACKLOG.yml — echte Produktentscheidung mit
  moeglicher HGB-Beruehrung, kein reiner Bugfix)
- offen: `fix-409-double-meaning-on-grpc-conflict-routes` blockiert, siehe
  `vorab-befund` oben — braucht entweder Lukes Entscheidung, die Unit als
  Block-D-Scan neu zuzuschneiden, oder eine deutlich groessere Iteration.
  `decide-email-messages-contact-pii-on-anonymize` wartet auf Lukes
  Entscheidung (a)/(b)/(c) in BACKLOG-NEXT.yml.

## Iteration 13 — harden-caldav-rest-routes-missing-idempotency — done — 2026-08-26 02:17
- commit: (folgt im selben Commit wie dieser Journal-Eintrag; Subject `fix(gateway): run CalDAV REST routes through the idempotency middleware`)
- gebaut: Root Cause war genau die in der Unit vermutete — `setupCalDAV`
  (`cmd/gateway/setup.go:121`) bekam nur die halbe Kette. `main.go:353` reichte
  `authMiddleware` durch, waehrend jeder andere Registrar `authWithIdempotency`
  (`main.go:205-206`) bekommt; `c.authMiddleware` wird in `route_caldav.go`
  ausschliesslich von den beiden `/api/v1`-Bloecken benutzt (`:147`, `:162`),
  die Protokollrouten haengen an `basicAuthMiddleware`. `NewCalDAVRoutes` nimmt
  jetzt `idempotencyMW` als eigenen Parameter (NICHT die vorkomponierte Kette:
  diese Datei registriert drei Auth-Geschmacksrichtungen auf einem Router, und
  nur die JWT-REST-Bloecke duerfen Idempotency tragen — die vorkomponierte
  Kette ist genau der Grund, warum es untergehen konnte). Neuer Helper
  `useAPIChain(sub)` setzt `authMiddleware` dann `idempotencyMW` — dieselbe
  Reihenfolge wie `authWithIdempotency`, damit Tenant/User im Context stehen,
  bevor die Middleware sie liest. Damit laufen alle sieben mutierenden Routen
  (`POST /passwords`, `DELETE /passwords/{id}`, `PUT /enable`, `PUT /disable`,
  `POST /test`, `PUT /admin/settings`, `DELETE /admin/users/{userId}/passwords`)
  durch dieselbe Kette wie der Rest von `/api/v1`. GET-Routen sind unberuehrt,
  die Middleware ueberspringt nicht-mutierende Methoden selbst
  (`middleware/idempotency.go:65-69`). `IDEMPOTENCY_MODE` NICHT angefasst —
  bleibt WarnMode, die Unit schafft die Moeglichkeit zur Deduplizierung, sie
  erzwingt sie nicht.
  Neuer DB-Test `route_caldav_idempotency_db_test.go` (3 Tests): registriert
  die echte `RegisterRoutes`-Verdrahtung auf einem echten chi-Router mit einer
  echten `idempotency.NewPostgresRepository`. Der Passwort-Service bleibt
  bewusst der bestehende Fake — die Behauptung betrifft die Middleware-Kette,
  nicht `caldav.AppPasswordService`. Vier bestehende `NewCalDAVRoutes`-Aufrufe
  in `route_caldav_test.go` plus einer in `openapi_drift_test.go` auf den neuen
  Parameter nachgezogen (Pass-Through statt `nil` — ein `nil` wuerde in chi
  erst beim ersten Request panicken, ein explizites Pass-Through macht an jeder
  Aufrufstelle sichtbar, dass hier eine Entscheidung getroffen wird).
- gate: build ok (`./internal/gateway/... ./cmd/gateway/...`) | vet ok | lint ok
  (0 issues) | test ok (`./internal/gateway/` komplett gruen inkl.
  `TestOpenAPIRouteDrift`; `./internal/caldav/...` gruen; **0 Skips** bei den
  neuen Tests verifiziert, `DATABASE_URL` als `kmuhub_app` gesetzt) |
  migration n.a. (reine Middleware-Verdrahtung, keine Tabelle, keine Route neu —
  `openapi.yaml` unveraendert, alle sieben Pfade stehen bereits drin) |
  rls-smoke n.a. (keine Tabelle/Policy angefasst; `idempotency_keys` hat RLS
  seit Bestehen, der Test schreibt tenant-gescoped ueber den PoolFromEnv-Hook)
- coverage: internal/gateway 56,7 % -> 56,8 % (selbst gemessen, Vorher-Wert per
  `git stash push -u -- backend/internal/gateway/ backend/cmd/gateway/`).
  Achtung: `coverage_start` der Unit nennt `internal/caldav 54,9 %` — das ist
  das falsche Paket, die Aenderung liegt vollstaendig in `internal/gateway`
  bzw. `cmd/gateway`. `internal/caldav` ist von dieser Unit gar nicht beruehrt.
- mutations-probe: `sub.Use(c.idempotencyMW)` in `useAPIChain` entfernt (Datei
  vorher per `cp` gesichert). `TestCalDAVCreatePassword_SameIdempotencyKey_
  CreatesExactlyOnePassword` wurde rot ("idempotency key ... did not complete
  within deadline" — ohne die Middleware wird gar kein Key reserviert), Datei
  per `cp` zurueckgedreht, `git diff --stat` wieder identisch (24+/2-),
  kompletter Testlauf danach gruen.
- verify vorgaenger: sauber. `73aa6f1d` aendert nur `internal/crm/consent/
  scrub.go` + einen Test; beide neuen `tx.Exec` filtern `tenant_id = $2`
  explizit, keine neue Route, kein `.proto`, keine Migration, kein
  `RequirePermission`, keine gRPC-Umgehung (der Scrub laeuft innerhalb der
  bestehenden Service-Transaktion).
- neue-units: keine
- offen: (1) NEBENFUND aus den Unit-Notes, wie gefordert hier genannt und NICHT
  gefixt: `basicAuthMiddleware` (`route_caldav.go:178`, `:189`) sendet
  `WWW-Authenticate: Basic realm="KMU Hub CalDAV"` — ein user-sichtbarer String
  mit dem alten Produktnamen, den der Browser im Anmeldedialog zeigt. Gehoert
  in die Inventur von Scan D10, nicht in diesen Fix.
  (2) NEUER NEBENFUND beim Testschreiben: die replayte Antwort ist NICHT
  byte-identisch mit der ersten. Der gecachte Body macht eine Runde durch eine
  `jsonb`-Spalte und kommt mit anderer Schluesselreihenfolge und Whitespace
  zurueck (`{"id":"…","label":…}` vs `{"id": "…", "label": …}`). Fuer JSON-Clients
  irrelevant, aber jeder Client, der Antworten byte-weise vergleicht oder
  signiert, sieht bei einem Replay etwas anderes. Der Test vergleicht deshalb
  feldweise. Falls das jemals stoeren sollte, waere der Ort
  `middleware/idempotency.go` (Body als `bytea` statt `jsonb` speichern) — als
  Unit habe ich es nicht angelegt, weil kein Client im Repo so vergleicht.
  (3) Der Admin-Block bekommt jetzt `auth -> idempotency -> RequireRole`. Das
  heisst: ein 403 eines Nicht-Admins wird unter seinem Idempotency-Key gecacht.
  Bewusst so gewaehlt, weil `RequirePermission`-Guards ueberall sonst im Gateway
  ebenfalls INNERHALB von `authWithIdempotency` sitzen — die Unit verlangt
  "dieselbe Kette wie die uebrigen Routen", nicht eine bessere.

## Iteration 14 — fix-datev-token-refresh-thundering-herd — done — 2026-08-26 02:25
- commit: (folgt im selben Commit wie dieser Journal-Eintrag; Subject
  `fix(biz): serialize DATEV OAuth refresh per tenant with singleflight`)
- gebaut: `OAuthManager.GetAccessToken` (`internal/biz/datev/oauth.go:70`) las
  den Cache unter `RLock`, gab das Lock frei und rief bei Ablauf
  `RefreshAccessToken` ohne jedes Lock auf — zwei gleichzeitige Aufrufer fuer
  denselben Tenant loesten damit zwei unabhaengige HTTP-Roundtrips gegen den
  DATEV-Token-Endpunkt aus, beide mit demselben (noch nicht rotierten)
  Refresh-Token. Fix: `refreshGroup singleflight.Group` (neues Feld auf
  `OAuthManager`, `golang.org/x/sync/singleflight` — Paket war bereits
  indirekte Dependency, `go mod tidy` hat es auf direkt gehoben, keine neue
  Dependency). `GetAccessToken` prueft den Cache zuerst ausserhalb jeder
  Sperre (`cachedToken`, reiner Lesezugriff wie bisher), faellt bei Ablauf
  aber jetzt durch `refreshGroup.Do(tenantID.String(), ...)` statt direkt
  `RefreshAccessToken` aufzurufen — gleichzeitige Aufrufer fuer denselben
  Tenant teilen sich EINE Ausfuehrung, verschiedene Tenants blockieren sich
  nicht (verschiedene Keys). Die eigentliche Arbeit passiert in
  `refreshLocked`, das den Cache ERNEUT prueft (double-checked), bevor es
  `RefreshAccessToken` aufruft — sonst wuerde ein Aufrufer, der erst NACH
  einem bereits abgeschlossenen Refresh in die Gruppe eintritt, unnoetig
  nochmal refreshen. `RefreshAccessToken` selbst ist unveraendert (immer ein
  erzwungener Refresh) — bestehende direkte Aufrufer/Tests (`ExchangeCode`
  ruft es nicht auf, aber sieben Tests rufen `om.RefreshAccessToken` direkt
  und erwarten IMMER einen Serverhit) haetten sonst ihre Semantik verloren.
  Caller-Grep (done_when-Pflicht): `GetAccessToken` wird nur von
  `uploader.go:101,175` und `belegbilder.go:31` aufgerufen — beide ueber den
  jetzt gesicherten Pfad. `RefreshAccessToken` hat ausserhalb der Tests KEINEN
  direkten Caller (nur intern von `GetAccessToken`/`refreshLocked`) — kein
  Reconnect-Pfad umgeht das Lock. Schwesterpruefung: `internal/biz/bexio/
  auth.go` (`TokenManager.GetAccessToken`/`RefreshAccessToken`) hat exakt
  dasselbe Muster (RLock lesen, ohne Lock refreshen) — NICHT mitgezogen, weil
  `internal/biz/bexio` in diesem Lauf explizit gesperrt ist (BACKLOG.yml
  Kopf, "GESPERRT IN DIESEM LAUF"). `internal/biz/lexware` hat kein
  aequivalentes Muster: `APIKeyManager` (`lexware/auth.go`) haelt einen
  statischen API-Key ohne Cache/Refresh-Zyklus.
- gate: build ok (`./internal/biz/datev/...`) | vet ok | lint ok (0 issues,
  `golangci-lint run --config .golangci.yml ./internal/biz/datev/...`) |
  test ok (`./internal/biz/datev/` komplett gruen, 104 Tests, **0 Skips**
  verifiziert per `go test -v | grep -i SKIP` — nur zwei Testnamen enthalten
  das Wort "Skip", keine uebersprungenen Faelle; `DATABASE_URL` als
  `kmuhub_app` gesetzt, die beiden DB-gaeteten Testdateien liefen also echt) |
  migration n.a. (keine Tabelle, kein Schema angefasst) | rls-smoke n.a.
  (keine Tabelle/Policy beruehrt) | route-drift n.a. (kein Gateway-Code, keine
  Route angefasst, `./internal/gateway/` daher nicht Pflicht-Gate hier)
- coverage: internal/biz/datev 80,8 % -> 81,1 % (selbst gemessen per
  `git stash push -u -- internal/biz/datev/ go.mod go.sum` / `pop`).
  `coverage_start` der Unit nennt 79,7 % aus einem aelteren CI-Lauf
  (32570176303) — die eigene Vorher-Messung weicht leicht ab, vermutlich weil
  eine der beiden vorangehenden `feat-scrub-dependent-pii-*`- oder
  `harden-caldav-*`-Iterationen dieses Laufs das Paket nicht beruehrt hat und
  der Bezugswert schlicht aelter ist als CI-Lauf 32735558575 (64,1 %-Gesamt),
  auf den der Backlog-Kopf sich sonst bezieht.
- mutations-probe: (1) `refreshLocked` auf `return om.RefreshAccessToken(ctx,
  tenantID)` ohne den vorgeschalteten Cache-Check reduziert (Datei vorher per
  `cp` gesichert) — `TestRefreshLocked_ReturnsCacheHitWithoutHittingVaultOrServer`
  wurde rot ("must not refresh: the cache is already warm", der Vault-Stub
  wurde entgegen der Erwartung angefasst), Datei per `cp` zurueckgedreht,
  `git diff --stat` wieder identisch zum vorherigen Stand, kompletter
  Testlauf danach gruen. (2) Vor dem Fix, am UNVERAENDERTEN Vorgaengerstand
  (vor dieser Iteration), lief der alte Test
  `TestGetAccessToken_ConcurrentColdCacheCallsBothHitTokenEndpoint` bereits
  gruen bei `calls == 2` — das war der dokumentierte Bug selbst, nicht die
  Probe. Als Beleg fuer den Fix wurde derselbe Test umbenannt und auf
  `calls == 1` umgestellt; er waere am alten Code rot gewesen (dort ist es
  ja gerade der Beleg, dass der Fix vorher fehlte).
- verify vorgaenger: sauber. `6cdb9ab0` aendert `cmd/gateway/main.go`,
  `cmd/gateway/setup.go`, `internal/gateway/route_caldav.go` sowie die
  zugehoerigen Tests — `NewCalDAVRoutes` nimmt jetzt `authMiddleware` und
  `idempotencyMW` als getrennte Parameter, `useAPIChain` legt beide in
  derselben Reihenfolge wie `authWithIdempotency` an. Kein `.proto`
  angefasst, keine neue Route (`openapi.yaml` unveraendert, alle sieben Pfade
  standen bereits drin), keine neue Tabelle, kein neuer
  `RequirePermission`-Guard, keine gRPC-Umgehung (reine Middleware-
  Verdrahtung). Der im Journal genannte Nebenfund (Basic-Auth-Realm-String
  "KMU Hub CalDAV") ist als Beobachtung fuer Scan D10 vermerkt, nicht
  gefixt — korrekt gegen die Unit-Notes.
- neue-units: keine
- offen: (1) Dieselbe Race existiert strukturidentisch in
  `internal/biz/bexio/auth.go` (`TokenManager`) — NICHT gefixt, weil
  `internal/biz/bexio` in diesem Lauf explizit gesperrt ist. Wird die Sperre
  aufgehoben, ist der Fix hier eine direkte Vorlage. (2) Die
  Refresh-Token-Rotationsfrage bei DATEV bleibt unbeantwortet (braucht
  DATEV-Portalzugang) — der Fix ist unabhaengig davon richtig, reduziert aber
  nur die Haeufigkeit des Problems, loest die Rotationsfrage selbst nicht.
  (3) `-race`-Bestaetigung bleibt CI vorbehalten (kein gcc lokal); die neuen
  Tests beweisen die Deduplizierung ueber Aufrufzaehlung am Fake-Server, nicht
  ueber den Race-Detector.

## Iteration 15 — fix-invoice-pdf-missing-buyer-vat-id — done — 2026-08-26 02:32
- commit: (siehe unten, wird im selben Schritt erstellt)
- gebaut: `buildRecipient` in `internal/biz/pdf/templates.go` druckt jetzt die
  Kaeufer-USt-IdNr. (`USt-IdNr.: <wert>`) im Empfaenger-Block, wenn
  `CustomerUStIDNr` gesetzt ist — konditional, keine leere Zeile bei leerem
  Feld. Alle vier Aufrufstellen in `generator.go` (Quote, Invoice, CreditNote,
  Dunning) geben das Feld jetzt durch. `GenerateCreditNotePDF` wurde nach dem
  Muster von `buildInvoiceDoc` in `buildCreditNoteDoc` (gibt `core.Maroto`
  zurueck, kein PDF-Byte-Diff noetig) + duennen `GenerateCreditNotePDF`-Wrapper
  aufgeteilt, damit ein Test die Gutschrift-Struktur direkt inspizieren kann.
  Entscheidung (siehe `notes:` der Unit, "steht mit Begruendung im Journal"):
  die Nummer wird NICHT nur im Reverse-Charge-Fall gedruckt, sondern in JEDEM
  Modus, sobald `CustomerUStIDNr` nicht leer ist — konsistent mit dem
  bestehenden Header/Footer-Muster (eigene USt-IdNr. wird ebenfalls
  unconditional gedruckt, sobald gesetzt) und weil die XML-Seite
  (`buildBuyerParty`, `einvoice/generator_doc.go:257`) das Feld ebenfalls ohne
  Ruecksicht auf `TaxMode` uebernimmt — eine Beschraenkung auf Reverse Charge
  haette die Inkonsistenz nur teilweise behoben.
  Gutschrift-Frage aus den Notes beantwortet: `creditnote/service.go:137`
  kopiert `CustomerUStIDNr` unveraendert von der Rechnung, und
  `GenerateCreditNotePDF` nutzt exakt dasselbe `buildRecipient` — belegt durch
  `TestCreditNotePDF_BuyerVATIDPrinted`.
- gate: build ok (`./internal/biz/pdf/...`) | vet ok
  (`./internal/biz/pdf/...`) | lint ok (0 issues,
  `golangci-lint run --config .golangci.yml ./internal/biz/pdf/...
  ./internal/biz/einvoice/...`) | test ok (`./internal/biz/pdf/`,
  `./internal/biz/einvoice/`, `./internal/biz/creditnote/...` alle gruen,
  keine Skips — reine Unit-Tests ohne DB-Anbindung in diesem Paket) |
  migration n.a. (keine Tabelle/Schema angefasst) | rls-smoke n.a. (kein
  DB-Zugriff in diesem Paket) | route-drift n.a. (kein Gateway-Code, keine
  Route angefasst — `internal/gateway` daher kein Pflicht-Gate hier; trotzdem
  `go build ./internal/gateway/... ./cmd/gateway/...` zur Sicherheit gruen)
- coverage: internal/biz/pdf 52,0 % -> 60,8 % (selbst gemessen per
  `git stash push -u -- generator.go invoice_ustg14_test.go templates.go` /
  `pop`, DATABASE_URL gesetzt obwohl das Paket keine DB-Tests hat).
  `coverage_start` der Unit nennt "Paket vorher selbst messen" — keine
  Diskrepanz.
- mutations-probe: `if vatID != ""` in `buildRecipient` auf
  `if false && vatID != ""` reduziert (Datei vorher per `cp` gesichert) —
  `TestInvoicePDF_BuyerVATIDPrintedInReverseCharge` UND
  `TestCreditNotePDF_BuyerVATIDPrinted` wurden beide rot (erwarteter Text
  fehlte im gerenderten Output), `TestInvoicePDF_BuyerVATIDOmittedWhenEmpty`
  blieb erwartungsgemaess gruen (Gegenprobe: bei leerem Feld aendert die
  Mutation nichts). Datei per `cp` zurueckgedreht, `git diff --stat` danach
  wieder identisch zum Stand vor der Mutation, kompletter Testlauf
  (`./internal/biz/pdf/`) danach erneut gruen.
- verify vorgaenger: sauber. `3b63d2c1` (DATEV-OAuth-Singleflight) aendert
  nur `internal/biz/datev/oauth.go` + Test + `go.mod`; reine Business-Logik
  ohne gRPC-Aufruf, kein `.proto`, keine neue Tabelle/Route/Guard, keine
  Wire-Shape-Aenderung. Diff selbst nachvollzogen (`git show`), Journal-Eintrag
  der Vor-Iteration deckt sich mit dem Code.
- neue-units: keine
- offen: `GenerateQuotePDF` und `GenerateDunningPDF` drucken die Kaeufer-
  USt-IdNr. jetzt ebenfalls (ueber denselben `buildRecipient`-Aufruf), sind
  aber keine GoBD-Belege im engeren Sinn und wurden dafuer NICHT extra
  getestet — nur die beiden `done_when`-relevanten Faelle (Invoice,
  CreditNote) tragen eigene Tests. Wer das fuer Angebot/Mahnung ebenfalls
  belegt haben will, kann das als eigene, sehr kleine Coverage-Unit anlegen.

## Iteration 16 — cov-fuhrpark-postgres-repository-real-sql — done — 2026-08-26 02:41
- commit: (siehe unten, wird im selben Schritt erstellt)
- gebaut: Neue Datei `internal/fuhrpark/postgres_repository_gap_test.go` mit sieben
  DB-Tests (82 PASS inkl. Subtests) gegen echtes Postgres, die den Rest von
  `postgres_repository.go` abdecken, der bislang von keinem Test angefasst wurde:
  Services (`GetService`/`ListServices`/`UpdateService`/`DeleteService`,
  Filter auf VehicleID/Status/ScheduledBefore), Damages (`GetDamage`/`ListDamages`/
  `UpdateDamage` — kein `DeleteDamage` im Repository-Interface, also nicht getestet),
  `GetVehicleHistory` (UNION aus vehicle_services + vehicle_damages, DESC-Reihenfolge,
  Tenant-Scope), Fuel Logs (`ListFuelLogs` mit/ohne Vehicle-Filter, `UpdateFuelLog`,
  `DeleteFuelLog`), Trip Logs (`ListTripLogs`, `UpdateTripLog` inkl. EndKm<StartKm-Guard,
  `DeleteTripLog`, generierte `km`-Spalte), Vehicle Documents (`ListVehicleDocuments`,
  `DeleteVehicleDocument`) und GPS (`IngestGpsPositions`, `GetGpsPositions`,
  `GetVehicleRoutes` — der einzige echte SQL-JOIN in dieser Datei, gps_positions zu
  vehicles). Vehicle-Kern, Booking, Driver-License und Trip-Log-Export waren bereits durch
  `postgres_repository_core_test.go`, `booking_conflict_test.go`, `driver_license_test.go`
  und `triplog_export_test.go` abgedeckt und wurden nicht dupliziert.
  Alle Lese- und Schreibpfade sind entweder durch die neuen Tests belegt oder — bei
  `DeleteDamage` — weil es die Methode im Repository schlicht nicht gibt.
- gate: build ok (`./internal/fuhrpark/...`) | vet ok (`./internal/fuhrpark/...`) | lint ok
  (0 issues, `golangci-lint run --config .golangci.yml ./internal/fuhrpark/...`) | test ok
  (`./internal/fuhrpark/`, komplettes Paket gruen, 82 PASS, 0 SKIP) | migration n.a. (keine
  Tabelle/Policy angefasst, alle sechs betroffenen Tabellen hatten RLS bereits seit ihrer
  jeweiligen Erst-Migration) | rls-smoke ok (vier Methoden mit eigenem Tenant > 0 UND
  fremdem Tenant = 0 belegt: `GetVehicleHistory`, `ListFuelLogs`, `ListVehicleDocuments`,
  `GetVehicleRoutes` — mehr als die geforderten drei) | route-drift n.a. (kein
  Gateway-Code, keine Route angefasst — `go build ./internal/gateway/... ./cmd/gateway/...`
  trotzdem zur Sicherheit gruen)
- coverage: internal/fuhrpark 54,5 % -> 81,3 % (selbst gemessen: neue Testdatei kurz nach
  /tmp verschoben, Paket ohne sie mit `-coverprofile` laufen lassen — 54,5 %, deckt sich
  exakt mit dem CI-Bezugswert der Unit —, Datei zurueckgeholt, erneut gemessen — 81,3 %).
  `postgres_repository.go` selbst war laut `coverage_start` bei 40,4 % (370/621 ungedeckt);
  Datei-genaue Nachmessung nicht separat durchgefuehrt, die Paketzahl ist der belastbare
  Beleg.
- mutations-probe: `ORDER BY occurred_at DESC` in `GetVehicleHistory`
  (`postgres_repository.go:445`) auf `ASC` geaendert (Datei vorher per `cp` gesichert).
  `TestGetVehicleHistory_MergesServicesAndDamagesOrderedAndTenantScoped` wurde sofort rot
  (`expected the newer damage entry first ... got ... service`). Datei per `cp`
  zurueckgedreht, `git diff` danach leer, kompletter Paketlauf (`./internal/fuhrpark/`)
  anschliessend wieder gruen (82 PASS).
- verify vorgaenger: sauber. `6ae3a10a` (Kaeufer-USt-IdNr. auf PDFs) aendert ausschliesslich
  `internal/biz/pdf/generator.go` + `templates.go` + Test; kein gRPC-Aufruf, kein `.proto`,
  keine neue Tabelle/Route/Guard, keine Wire-Shape-Aenderung. Diff mit `git show`
  nachvollzogen, deckt sich mit der Journal-Beschreibung der Vor-Iteration.
- neue-units: fix-fuhrpark-vehicle-routes-daily-km-always-zero (Backlog-Ende) —
  `GetVehicleRoutes` liefert `DailyKm` fuer jede Route hart als 0 aus, obwohl das Feld bis
  ins Frontend (`fuhrparkv1.VehicleRoute.DailyKm`, `desktop/.../types.ts`) durchgereicht
  wird und aus den vorliegenden Lat/Lng-Fixes berechenbar waere. Kein Verhalten in dieser
  Coverage-Unit geaendert, nur dokumentiert und als eigene Fix-Unit angelegt.
- offen: `postgres_repository.go`-dateigenaue Vorher/Nachher-Coverage wurde nicht separat
  gemessen (nur die Paketzahl) — bei Bedarf mit `go tool cover -func` gegen ein
  datei-gefiltertes Profil nachholen. `fix-fuhrpark-vehicle-routes-daily-km-always-zero`
  ist ein echter Produktionsfehler (falsche Kilometeranzeige im Fuhrpark-Modul), keine
  reine Coverage-Luecke — sollte fuer den naechsten Lauf priorisiert werden.

## Iteration 17 — cov-gateway-fuhrpark-licenses-and-documents — done — 2026-08-26 02:58
- commit: (siehe unten, wird im selben Schritt erstellt)
- gebaut: Neue Datei `internal/gateway/route_fuhrpark_licenses_documents_test.go` mit
  Tests fuer alle elf im Scope genannten Handler (HandleListDriverLicenses,
  HandleCreateDriverLicense, HandleUpdateDriverLicense, HandleDeleteDriverLicense,
  HandleListVehicleDocuments, HandleCreateVehicleDocument, HandleDeleteVehicleDocument,
  HandleListDamages, HandleListVehicleDamages, HandleUpdateDamage, HandleResolveDamage),
  je Handler mindestens ServiceUnavailable/MissingTenant/ReachesRPC, plus Validierungsfaelle
  fuer alle `validate`-Tags (driver_id required+uuid, license_classes min=1, next_check_due_date
  required, doc_type required+oneof, name/object_key required) und InvalidIDUUID fuer jede
  Route mit Pfad-Parameter. Folgt dem in `route_fuhrpark_crud_test.go` etablierten Muster
  (dummy `localhost:0`-Registry, Assertion auf 503 als Beleg, dass der Handler die
  Validierung passiert und die RPC erreicht hat).
  Unterschied HandleListDamages vs. HandleListVehicleDamages geklaert und im Testfile
  dokumentiert: HandleListDamages ist die flottenweite Uebersicht unter `/damages`, filterbar
  per Query nach `status` UND `vehicle_id`; HandleListVehicleDamages haengt unter
  `/vehicles/{id}/damages`, ist strikt auf die Pfad-ID gescoped und hat keinen Status-Filter.
  Beide rufen dieselbe `ListDamages`-RPC mit unterschiedlichem Request auf — keiner ist tote
  Flaeche.
  Zwei Befunde beim Bauen, beide als Fix-Units angelegt statt hier gefixt (Coverage-Unit
  aendert kein Verhalten): (1) HandleDeleteDriverLicense/Service.DeleteDriverLicense loescht
  jede Kontrollzeile ungeprueft, auch die neueste/einzige je driver_id — der im Scope
  benannte Befund war real; (2) updateDamageRequest hat auf Severity/Status gar kein
  `validate`-Tag (anders als reportDamageRequest), Service.UpdateDamage schreibt beide Felder
  ungeprueft durch — belegt durch
  `TestHandleUpdateDamage_ArbitrarySeverityReachesRPCUnvalidated`, die zeigt, dass ein
  Nonsense-Wert die RPC statt einer 400 erreicht.
- gate: build ok (`./internal/gateway/... ./cmd/gateway/...`) | vet ok
  (`./internal/gateway/...`) | lint ok (0 issues, `golangci-lint run --config .golangci.yml
  ./internal/gateway/...`) | test ok (`./internal/gateway/`, komplettes Paket gruen, 2741
  PASS, 0 SKIP, `DATABASE_URL` gesetzt) | migration n.a. (keine Tabelle/Policy angefasst) |
  rls-smoke n.a. (reine Handler-Unit-Tests gegen Dummy-Registry, keine echte DB-Verbindung —
  wie die bestehenden `route_fuhrpark_crud_test.go`-Tests) | route-drift ok
  (`TestOpenAPIRouteDrift`: 836 registrierte Routen gegen 838 dokumentierte Pfade, PASS —
  keine neue Route in dieser Unit)
- coverage: internal/gateway 56,8 % -> 57,6 % (selbst gemessen: neue Testdatei kurz nach
  /tmp verschoben, Paket ohne sie mit `-coverprofile` laufen lassen — 56,8 %, deckt sich mit
  dem CI-Bezugswert 56,6 % der Unit —, Datei zurueckgeholt, erneut gemessen — 57,6 %). Alle
  elf Ziel-Handler liegen jetzt bei 82,6-96,2 % (vorher 0 %), Datei-Detailwerte im Diff-Kontext
  der Iteration.
- mutations-probe: `oneof=registration insurance tuev other` in `createVehicleDocumentRequest`
  auf `oneof=registration insurance other` verkuerzt (Datei vorher per `cp` gesichert).
  `TestHandleCreateVehicleDocument_ReachesRPC` (benutzt `doc_type: tuev`) wurde sofort rot
  (400 `validation_failed` statt 503). Datei per `cp` zurueckgedreht, `git diff` danach leer,
  kompletter Paketlauf (`./internal/gateway/`) anschliessend wieder gruen.
- verify vorgaenger: sauber. `f0976050` (Fuhrpark-Repository-DB-Tests) aendert nur
  `internal/fuhrpark/postgres_repository_gap_test.go` + Backlog/Journal; kein gRPC-Aufruf im
  Gateway umgangen, kein `.proto`, keine neue Tabelle/Route/Guard, keine Wire-Shape-Aenderung.
  Diff mit `git show --stat` nachvollzogen, deckt sich mit der Journal-Beschreibung.
- neue-units: fix-fuhrpark-delete-driver-license-no-last-check-guard (Backlog-Ende, deps: []) —
  DeleteDriverLicense schuetzt die letzte/neueste Kontrollzeile eines driver_id nicht, obwohl
  sie der Halterhaftungs-Nachweis ist; done_when[0] verlangt bewusst KEINE Luke-Entscheidung
  (Preflight-Regel), sondern nennt beide moeglichen Wege (409-Block oder Audit-Event) und
  bittet die naechste Iteration, die Wahl vorab mit Luke zu klaeren, bevor sie baut.
  fix-fuhrpark-update-damage-missing-enum-validation (Backlog-Ende, deps: []) —
  updateDamageRequest validiert Severity/Status nicht gegen dieselbe Wertemenge wie beim
  Erzeugen, kleiner klar umrissener Fix (ein validate-Tag je Feld).
- offen: Preflight (`hooks/backlog-check.py --preflight`) meldet weiterhin die bereits vor
  dieser Iteration bestehende `fix-409-double-meaning-on-grpc-conflict-routes` mit
  `status: blocked` + `blocked_reason` direkt in BACKLOG.yml (gehoert nach
  BACKLOG-PARKED.yml/BACKLOG-NEXT.yml) — mit `git stash` gegen den Stand vor dieser Iteration
  gegengeprueft, nicht durch diese Iteration verursacht, hier nur dokumentiert statt
  angefasst (ausserhalb des Scopes dieser Coverage-Unit).

## Iteration 18 — cov-gateway-fuhrpark-triplogs-gps-exports — done — 2026-08-26 03:00
- commit: (siehe unten, wird im selben Schritt erstellt)
- gebaut: Neue Datei `internal/gateway/route_fuhrpark_triplogs_gps_test.go` mit Tests fuer
  alle dreizehn im Scope genannten Handler (HandleListTripLogs, HandleListVehicleTripLogs,
  HandleExportTripLogs, HandleListFuelLogs, HandleListVehicleFuelLogs, HandleListServices,
  HandleDeleteService, HandleListUpcomingServices, HandleListVehicleBookings,
  HandleIngestGpsPositions, HandleGetVehicleRoutes, HandleGetGpsPositions,
  HandleExportVehicleReport), je Handler mindestens ServiceUnavailable/MissingTenant/
  ReachesRPC plus die vorhandenen Validierungs-/UUID-Faelle, im selben `localhost:0`-Dummy-
  Registry-Muster wie die Vorgaenger-Iterationen.
  Die drei im Scope benannten Verdachtsstellen geprueft:
  (1) `HandleIngestGpsPositions`/`Service.IngestGpsPositions`/`PostgresRepository.
  IngestGpsPositions` — Batch-Obergrenze: es gibt keine, nur `required,min=1`. Mit
  `TestHandleIngestGpsPositions_LargeBatchReachesRPC` (5000 Positionen) belegt, dass kein
  lokaler Reject stattfindet; laut Scope-Notiz eine Ressourcenfrage, keine eigene Fix-Unit.
  Tenant-Zugehoerigkeit des Fahrzeugs wird dagegen NICHT geprueft — weder Service noch
  Repository verifizieren, dass `vehicle_id` dem aufrufenden Tenant gehoert, bevor inserted
  wird (`gps_positions.vehicle_id` hat nur eine FK auf `vehicles(id)`, keine tenant-scoped
  Pruefung). Als eigene Fix-Unit angelegt.
  (2) `trip_logs.is_private` wird in `ListTripLogs` UND `ListTripLogsForExport`
  (`postgres_repository.go:639`/`:1191`) selektiert, aber in keiner WHERE-Klausel gefiltert —
  private Fahrten erscheinen ununterscheidbar im arbeitgeberseitigen Fahrtenbuch-Export. Als
  eigene Fix-Unit angelegt (Entscheidung opt-in vs. immer ausgeblendet gehoert Luke).
  (3) Tenant-Filter der Export-Handler: `HandleExportTripLogs` und
  `HandleExportVehicleReport` setzen beide `TenantId` im Request, `ExportTripLogsParams`/
  `GetVehicleRoutesParams`/`GetGpsPositionsParams` sind alle tenant-gescoped bis in die SQL
  (`WHERE tenant_id=$1`) — vollstaendig, kein Befund.
- gate: build ok (`./internal/gateway/... ./cmd/gateway/...`) | vet ok
  (`./internal/gateway/...`) | lint ok (0 issues, `golangci-lint run --config .golangci.yml
  ./internal/gateway/...`) | test ok (`./internal/gateway/`, komplettes Paket gruen, 0 FAIL,
  0 SKIP, `DATABASE_URL` gesetzt) | migration n.a. (keine Tabelle/Policy angefasst) |
  rls-smoke n.a. (reine Handler-Unit-Tests gegen Dummy-Registry, keine echte DB-Verbindung) |
  route-drift ok (`TestOpenAPIRouteDrift`: 836 registrierte Routen gegen 838 dokumentierte
  Pfade, PASS — keine neue Route in dieser Unit)
- coverage: internal/gateway 57,6 % -> 58,6 % (selbst gemessen mit `-coverprofile`: neue
  Testdatei kurz nach `/tmp` verschoben, Paket ohne sie gemessen — 57,6 %, deckt sich mit dem
  Bezugswert der Vor-Iteration 57,6 % — Datei zurueckgeholt, erneut gemessen — 58,6 %)
- mutations-probe: `ingestGpsPositionsRequest.Positions` von `validate:"required,min=1"` auf
  kein Tag verkuerzt (Datei vorher per `cp` gesichert). `TestHandleIngestGpsPositions_
  MissingPositions` sofort rot (503 + leerer error-string statt 400/validation_failed).
  Datei per `cp` zurueckgedreht, `git diff` danach leer, kompletter Paketlauf
  (`./internal/gateway/`) anschliessend wieder gruen.
- verify vorgaenger: sauber. `ba94e8de` (Fuhrpark-Fuehrerschein/Dokument/Schaden-Gateway-
  Tests) aendert nur `internal/gateway/route_fuhrpark_licenses_documents_test.go` +
  Backlog/Journal; kein gRPC-Aufruf im Gateway umgangen, kein `.proto`, keine neue Tabelle/
  Route/Guard, keine Wire-Shape-Aenderung. `git show --stat` gegengeprueft, deckt sich mit der
  Journal-Beschreibung der Vor-Iteration.
- neue-units: fix-fuhrpark-trip-logs-private-flag-not-filtered (Backlog-Ende, deps: []) —
  is_private wird in Liste UND Export von trip_logs nirgends gefiltert, private Fahrten
  landen ununterscheidbar im Arbeitgeber-Bericht; done_when[0] nennt bewusst beide moeglichen
  Wege (opt-in sichtbar vs. immer ausgeblendet) statt eine Entscheidung zu verlangen.
  fix-fuhrpark-gps-ingest-no-vehicle-tenant-check (Backlog-Ende, deps: []) — IngestGpsPositions
  prueft nicht, ob vehicle_id dem aufrufenden Tenant gehoert, bevor es GPS-Positionen dagegen
  inserted; Vorbild fuer den Fix ist der bestehende Ownership-Check in GetVehicleHistory.
- offen: Batch-Obergrenze bei HandleIngestGpsPositions bewusst nicht als Fix-Unit angelegt
  (Scope-Notiz: Ressourcenfrage, kein Testloch) — mit `TestHandleIngestGpsPositions_
  LargeBatchReachesRPC` nur belegt, nicht gefixt; falls das je zum Thema wird, gehoert dazu
  auch eine Antwort auf den fehlenden globalen `MaxBytesReader` fuer JSON-POST-Bodies (andere
  Endpunkte wie route_document.go/route_biz_banking.go setzen ihn nur lokal, nicht generisch).

## Iteration 19 — cov-gateway-hr-employees-and-documents — done — 2026-08-26 03:09
- commit: (siehe unten, wird im selben Schritt erstellt)
- gebaut: Neue Datei `internal/gateway/route_hr_employees_documents_test.go` mit Tests fuer
  alle zwoelf im Scope genannten Handler (HandleCreateEmployee, HandleUpdateEmployee,
  HandleGetEmployee, HandleListEmployees, HandleGetSelfProfile, HandleCreatePersonnelDocument,
  HandleListPersonnelDocuments, HandleUploadEmployeeDocument, HandleListEmployeeDocuments,
  HandleRecordSickLeave, HandleGetEmployeeLeaveBalance, HandleGetAbsenceCalendar), je Handler
  ServiceUnavailable/ReachesRPC plus Validierungsfaelle, im selben Dummy-Registry-Muster wie
  die Vorgaenger-Iterationen.
  Schwerpunkt "kann ein Mitarbeiter an die Akte eines Kollegen kommen" geprueft: `HandleGetEmployee`,
  `HandleListEmployees` und `HandleListEmployeeDocuments` haben KEINEN eigenen
  Tenant-/Ownership-Check im Gateway — `{id}` geht direkt in die RPC. Der Schutz gegen einen
  Kollegen-Zugriff sitzt komplett im Guard: `/employees/{id}`, `/employees/` (Liste) und
  `/employees/{id}/documents` sind alle mit `RequirePermission("hr","read")` bewacht, und
  Migration 000129 vergibt diesen Key NUR an `admin` — ein normaler Mitarbeiter hat den Key
  gar nicht und kommt nie bis zum Handler. Mit drei Router-Level-Tests belegt
  (`TestHandleGetEmployee_RequiresPermission`, `TestHandleListEmployees_RequiresPermission`,
  `TestHandleListEmployeeDocuments_RequiresPermission`, je 403 ohne den Key,
  `TestHandleGetEmployee_WithPermissionReachesRPC` als Gegenprobe mit dem Key). Fuer
  `HandleRecordSickLeave` gilt dasselbe Muster indirekt: die Route nimmt gar kein `{id}`
  entgegen, sie schreibt immer auf `middleware.GetUserID(ctx)` — ein Kollege kann schon
  strukturell keine fremde Krankmeldung anlegen; zusaetzlich mit `hr:write`-Permission-Guard-Test
  belegt. `HandleGetEmployeeLeaveBalance` (`/balance/{userId}`) und `HandleUploadEmployeeDocument`/
  `HandleCreatePersonnelDocument` (Personalakte-Schreibpfade) pruefen Tenant explizit
  (`getTenantID`), tragen aber ebenfalls "hr:read"/"hr:write"/"hr:admin"-Guards — kein neuer
  Befund, deckt sich mit dem bereits bekannten Muster.
  Kein Berechtigungsfehler gefunden (kein Guard fehlt, keiner haengt eine Ebene zu tief) —
  daher keine Fix-Unit noetig.
- gate: build ok (`./internal/gateway/... ./cmd/gateway/...`) | vet ok
  (`./internal/gateway/...`) | lint ok (0 issues, `golangci-lint run --config .golangci.yml
  ./internal/gateway/...`) | test ok (`./internal/gateway/`, komplettes Paket gruen, 0 FAIL,
  0 SKIP, `DATABASE_URL` gesetzt) | migration n.a. (keine Tabelle/Policy angefasst) |
  rls-smoke n.a. (reine Handler-Unit-Tests gegen Dummy-Registry, keine echte DB-Verbindung) |
  route-drift ok (Teil des vollen Paketlaufs, `TestOpenAPIRouteDrift` PASS — keine neue Route)
- coverage: internal/gateway 58,6 % -> 59,3 % (selbst gemessen mit `-coverprofile`: neue
  Testdatei kurz nach `/tmp` verschoben, Paket ohne sie gemessen — 58,6 %, deckt sich mit dem
  Bezugswert der Vor-Iteration 58,6 % — Datei zurueckgeholt, erneut gemessen — 59,3 %)
- mutations-probe: `createEmployeeHTTPReq.UserID` von `validate:"required,uuid"` auf
  `validate:"omitempty,uuid"` verkuerzt (Datei vorher per `cp` gesichert).
  `TestHandleCreateEmployee_MissingUserID` sofort rot (503 statt 400/validation_failed, da
  ohne Pflichtfeld die leere UserID unvalidiert an die RPC geht). Datei per `cp`
  zurueckgedreht, `git diff` danach leer, kompletter Paketlauf (`./internal/gateway/`)
  anschliessend wieder gruen.
- verify vorgaenger: sauber. `d2026b9e` (Fuhrpark-Fahrtenbuch/GPS/Fuel-Log-Gateway-Tests)
  aendert nur `internal/gateway/route_fuhrpark_triplogs_gps_test.go` + Backlog/Journal; kein
  gRPC-Aufruf im Gateway umgangen, kein `.proto`, keine neue Tabelle/Route/Guard, keine
  Wire-Shape-Aenderung. Zwei echte Fuhrpark-Bugs wurden korrekt als eigene Fix-Units angelegt
  statt nebenbei gefixt (`fix-fuhrpark-trip-logs-private-flag-not-filtered`,
  `fix-fuhrpark-gps-ingest-no-vehicle-tenant-check`) — `git show --stat` gegengeprueft, deckt
  sich mit der Journal-Beschreibung der Vor-Iteration.
- neue-units: keine
- offen: keine

## Iteration 20 — cov-gateway-hr-time-tracking-paths — done — 2026-08-26 03:16
- commit: (siehe unten, wird im selben Schritt erstellt)
- gebaut: Neue Datei `internal/gateway/route_hr_time_tracking_test.go` mit Tests fuer alle
  sechzehn im Scope genannten Handler (HandleCreateManualEntry, HandleApproveCorrection,
  HandleListWorkTimeEntries, HandleGetWorkTimeStatus, HandleGetMyWeekStatus,
  HandleGetTimeBalance, HandleGetTeamTime, HandleCreateTimeCategory, HandleUpdateTimeCategory,
  HandleDeleteTimeCategory, HandleListTimeCategories, HandleCreateTimeProject,
  HandleListTimeProjects, HandleCreateTimeTemplate, HandleDeleteTimeTemplate,
  HandleListTimeTemplates), je ServiceUnavailable/MissingTenant/ReachesRPC plus
  Validierungsfaelle, im selben Dummy-Registry-Muster wie die HR-Vorgaenger-Iteration.
  Permission-Guard-Wiring fuer HandleApproveCorrection und HandleGetTeamTime ist bereits durch
  `TestCapabilityGuards_AdditiveWiring` (route_capability_guard_test.go) belegt und nicht
  dupliziert worden.
  Echter Berechtigungsfund zu Frage 1 ("kann ein Mitarbeiter seine eigene Korrektur
  genehmigen?"): JA. `Service.ApproveTimeCorrection` (service.go:441) vergleicht `approverID`
  nie mit `correction.EmployeeID` — der Guard (`hr:write` ODER
  `zeiterfassung:corrections:approve`) regelt nur WER genehmigen darf, nicht WESSEN eigene
  Korrektur. Mit `TestApproveTimeCorrection_EmployeeCanApproveOwnCorrection` in
  `internal/biz/hr/timetracking/service_test.go` belegt (Test ist gruen, weil der Bug real
  ist) und als eigene Fix-Unit `fix-hr-approve-correction-self-approval` (model: opus) ans
  Backlog-Ende gehaengt statt nebenbei gefixt.
  Frage 2 ("sieht ein Teamleiter nur sein Team?"): NEIN, aber kein Bug — `GetTeamTime`
  traegt den Kommentar "returns aggregated weekly time for all employees of the tenant" und
  `GetTeamTimeReq` hat kein Feld zum Team-/Manager-Scoping (Data-Model hat ueberhaupt keinen
  manager_id/team_id, nur ein freies Department-Textfeld). Explizit dokumentiertes Verhalten,
  keine stille Luecke — nicht als Fix-Unit angelegt, siehe offen: unten.
  Saldentest mit krummen Werten: `TestGetTimeBalance_FractionalWorkWeek` (6-Tage-Woche,
  2647 von 2880 Zielminuten, erwartet exakt -233) plus `TestWeekStartOf_AcrossDSTSpringForward`
  (Europe/Berlin, Woche mit der Sommerzeitumstellung 2026-03-29, prueft Montag 00:00 lokal).
  Nebenbefund gepruefte, aber NICHT als Bug gewertet: `DeleteTimeCategory`/`DeleteTimeTemplate`/
  `UpdateTimeCategory`-GetByID setzen kein `tenant_id` in der SQL — aber `hr_time_categories`/
  `hr_time_templates` laufen ueber `enable_tenant_rls` mit `FORCE ROW LEVEL SECURITY` und die
  Policy deckt per Default alle Commands (SELECT/UPDATE/DELETE), `internal/database/postgres.go`
  stempelt `app.tenant_id` pro Connection-Acquire aus dem Request-Context. Cross-Tenant-Zugriff
  wird also von RLS abgefangen, nicht von der Anwendung — konsistent mit dem sonstigen Muster
  im Repo, deshalb keine Fix-Unit.
- gate: build ok (`./internal/gateway/... ./internal/biz/hr/timetracking/... ./cmd/gateway/...`)
  | vet ok (beide Pakete) | lint ok (0 issues, `golangci-lint run --config .golangci.yml
  ./internal/gateway/... ./internal/biz/hr/timetracking/...`) | test ok (beide Pakete komplett
  gruen, 4142 PASS, 0 FAIL, 0 SKIP, `DATABASE_URL` gesetzt) | migration n.a. (keine
  Tabelle/Policy angefasst) | rls-smoke n.a. (reine Unit-Tests, keine echte DB-Verbindung fuer
  die neuen Tests) | route-drift ok (Teil des vollen `./internal/gateway/`-Laufs,
  `TestOpenAPIRouteDrift` PASS — keine neue Route)
- coverage: internal/gateway 59,3 % -> 60,1 % (`-coverprofile` vor/nach der neuen Testdatei
  gemessen); internal/biz/hr/timetracking 61,9 % (Bezugswert Unit) -> 63,4 % (selbst gemessen,
  neue Tests fuer ApproveTimeCorrection-Selbstgenehmigung, GetTimeBalance und weekStartOf)
- mutations-probe: `GetTimeBalance` in `internal/biz/hr/timetracking/service.go` von
  `workDays * 8 * 60` auf `workDays * 7 * 60` verkuerzt (Datei vorher per `cp` gesichert).
  `TestGetTimeBalance_FractionalWorkWeek` sofort rot (target 2520 statt 2880, balance 127 statt
  -233). Datei per `cp` zurueckgedreht, `git diff` danach leer, kompletter Paketlauf
  (`./internal/biz/hr/timetracking/`) anschliessend wieder gruen.
- verify vorgaenger: sauber. `7a2ab393` (HR-Employee-Profile/Personnel-Document-Gateway-Tests)
  aendert nur `internal/gateway/route_hr_employees_documents_test.go` + Backlog/Journal; kein
  gRPC-Aufruf im Gateway umgangen, kein `.proto`, keine neue Tabelle/Route/Guard, keine
  Wire-Shape-Aenderung. Kein Berechtigungsfehler gefunden, daher korrekt keine Fix-Unit
  angelegt — mit `git show --stat` gegengeprueft, deckt sich mit der Journal-Beschreibung.
- neue-units: fix-hr-approve-correction-self-approval (model: opus, Block-Ende)
- offen: `python3 .planning/backend-block/loop/hooks/backlog-check.py --preflight` schlaegt
  weiterhin fehl auf der VORBESTEHENDEN Unit `fix-409-double-meaning-on-grpc-conflict-routes`
  (status: blocked, stammt aus `18b85a4e` vom 2026-08-24, nicht aus dieser Iteration) — gehoert
  nach BACKLOG-PARKED.yml oder BACKLOG-NEXT.yml, siehe
  feedback_loop_backlog_yaml_never_parsed.md in memory. Von mir weder erzeugt noch veraendert.

## Iteration 21 — cov-gateway-hr-analytics-and-settings — done — 2026-08-26 03:29
- commit: 1e6691f1
- gebaut: Neue Datei `internal/gateway/route_hr_analytics_settings_test.go` mit Tests fuer
  alle fuenf im Scope genannten Handler (HandleDailySummary, HandleWeeklySummary,
  HandleGetTimeAnalytics, HandleGetHRSettings, HandleUpdateHRSettings), je
  ServiceUnavailable/MissingTenant/ReachesRPC plus Validierungsfaelle
  (work_hours_per_day > 24, au_threshold_days < 0), im selben Dummy-Registry-Muster wie die
  beiden HR-Vorgaenger-Iterationen. Zwei neue Service-Tests in
  `internal/biz/hr/timetracking/service_test.go`: `TestGetTimeAnalytics_EmptyPeriod` (leerer
  Zeitraum liefert 0 Minuten, 0 statt negativer Overtime, DayTrend mit 7 genullten Eintraegen,
  ByProject als `[]` statt `nil`) und eine Ergaenzung zu `TestServicePassesCallerTenantToRepo`
  fuer GetDailySummary/GetWeeklySummary/GetTimeAnalytics.
  Frage (1) Tenant-Scoping: JA belegt — alle drei Aggregations-Queries in
  `postgres_repository.go` (`GetDailySummary`, `aggregateDailyBuckets`, das Fundament von
  GetWeeklySummary/GetDailySummaryRange) filtern `tenant_id = $5`/`$5`, und der Service reicht
  den Aufrufer-Tenant durch (neuer Subtest, `assertAllTenants` gruen).
  Frage (2) leerer Zeitraum: kein Crash, keine Division durch Null — `aggregateDailyBuckets`
  seedet jeden Tag mit einem genullten `DailySummary` VOR der SQL-Antwort, `AvgDailyMinutes`
  teilt durch `numDays`, das aus der Route immer >=1 kommt (7 fest oder `now.Day()`).
  Frage (3) HR-Settings rueckwirkend: NEIN, aber schlimmer als das — `s.settingsRepo` wird in
  `internal/biz/hr/timetracking/service.go` konstruiert (Zeile 32/53), aber IM GANZEN PAKET
  nie gelesen (`grep -n "s\.settingsRepo" service.go` = 0 Treffer). `GetTimeBalance` und
  `GetTimeAnalytics` rechnen hart `workDays * 8 * 60`, der ArbZG-10h-Block in `ClockIn`
  (`errors.go:21`) ist eine feste Konstante. `HandleUpdateHRSettings` persistiert
  `work_hours_per_day`/`max_daily_hours`/`break_after_hours` korrekt (bestaetigt per
  Read von `hr_grpc.go:1581`), aber kein Konsument liest sie je zurueck — die Einstellung
  wirkt nie, weder rueckwirkend noch prospektiv. Als eigene Fix-Unit
  `fix-hr-settings-never-consumed-by-timetracking` (model: opus) ans Backlog-Ende gehaengt,
  nicht nebenbei gefixt (Verhaltensaenderung, keine Coverage-Unit).
  Verbleibender ungetesteter Rest von `route_hr.go`: NULL. Gegengeprueft mit
  `grep -oP 'func \(h \*HRRoutes\) \KHandle\w+' route_hr.go | sort -u` (55 Handler) gegen alle
  `route_hr*_test.go`-Dateien — vor dieser Unit waren genau diese fuenf ungetestet, danach
  keiner mehr. Keine Folge-Unit noetig.
- gate: build ok (`./internal/gateway/... ./internal/biz/hr/timetracking/... ./cmd/gateway/...`)
  | vet ok (beide Pakete) | lint ok (0 issues, `golangci-lint run --config .golangci.yml
  ./internal/gateway/... ./internal/biz/hr/timetracking/...`) | test ok (beide Pakete komplett
  gruen, 2986 PASS, 0 FAIL, 0 SKIP, `DATABASE_URL` gesetzt) | migration n.a. (keine
  Tabelle/Policy angefasst) | rls-smoke n.a. (reine Unit-Tests) | route-drift ok
  (`TestOpenAPIRouteDrift`: 836 registrierte gegen 838 dokumentierte Pfade, PASS — keine
  neue Route)
- coverage: internal/gateway 60,1 % -> 60,4 % (`-coverprofile` vor/nach der neuen Testdatei
  gemessen, Vorher-Lauf mit temporär entfernter Datei); internal/biz/hr/timetracking 63,4 %
  (Bezugswert aus Vorgaenger-Iteration) -> 63,7 % (selbst gemessen)
- mutations-probe: `overtimeMinutes < 0 { overtimeMinutes = 0 }`-Clamp in `GetTimeAnalytics`
  (`internal/biz/hr/timetracking/service.go`) entfernt (Datei vorher per `cp` gesichert).
  `TestGetTimeAnalytics_EmptyPeriod` sofort rot (erwartet 0, bekam -3360). Datei per `cp`
  zurueckgedreht, `git diff` danach leer, `./internal/biz/hr/timetracking/` anschliessend
  wieder komplett gruen.
- verify vorgaenger: sauber. `28a5d652` (HR-Time-Tracking-Gateway-Tests + Fix-Unit
  fuer Selbstgenehmigung) aendert nur Testdateien + Backlog/Journal; kein gRPC-Aufruf
  umgangen, kein `.proto`, keine neue Tabelle/Route/Guard, keine Wire-Shape-Aenderung. Der
  gefundene Berechtigungsfund wurde korrekt als eigene Fix-Unit angelegt statt nebenbei
  gefixt — mit `git show --stat` gegen die Journal-Beschreibung gegengeprueft, deckt sich.
- neue-units: fix-hr-settings-never-consumed-by-timetracking (model: opus, Block-Ende)
- offen: `route_hr.go` ist jetzt vollstaendig durch Tests referenziert (55/55 Handler-Namen
  treffen in mind. einer `route_hr*_test.go`-Datei). Die vorbestehende Preflight-Meldung zu
  `fix-409-double-meaning-on-grpc-conflict-routes` (aus `18b85a4e`, nicht dieser Iteration)
  besteht unveraendert fort, siehe Iteration 20.

## Iteration 22 — cov-gateway-video-breakout-and-cohosts — done — 2026-08-26 03:37
- commit: dabf02be
- gebaut: Neue Datei `internal/gateway/route_video_breakout_cohost_test.go` mit Tests fuer
  alle 15 im Scope genannten Handler (HandleCreateBreakoutRooms, HandleCloseBreakoutRooms,
  HandleListBreakoutRooms, HandleJoinBreakoutRoom, HandleReturnToMainRoom,
  HandleAssignBreakoutParticipant, HandleGetBreakoutAssignment, HandlePromoteCoHost,
  HandleDemoteCoHost, HandleListCoHosts, HandleMuteMeetingParticipant, HandleJoinCall,
  HandleEndCall, HandleGetCall, HandleListActiveCalls), je ServiceUnavailable/InvalidUUID/
  ReachesRPC plus Validierungsfaelle (fehlende/ungueltige UUID-Felder, Count-Grenzen 1..20),
  im selben Dummy-Registry-Muster wie die HR-Vorgaenger-Iterationen (kein Fake
  VideoServiceClient in diesem Paket vorhanden).
  Schwerpunkt-Fragen der Unit: (1) Koennen PromoteCoHost/DemoteCoHost/MuteMeetingParticipant
  von einem normalen Teilnehmer ausgeloest werden? NEIN, belegt — aber nicht am Gateway
  pruefbar (kein Fake-Client, das Gateway sieht nur den durchgereichten gRPC-Fehler). Die
  Autorisierung sitzt vollstaendig in `internal/work/meeting/service.go` (`isHostOrCoHost`,
  aufgerufen aus PromoteCoHost/DemoteCoHost/MuteMeetingParticipant/CreateBreakoutRooms/
  AssignBreakoutParticipant/ReturnToMainRoom/CloseBreakoutRooms) und dort bereits durch
  bestehende Tests bewiesen (gegengeprueft, alle sechs liefen gruen:
  TestPromoteCoHost_NonOrganizerDenied, TestDemoteCoHost_NonOrganizerDenied,
  TestMuteMeetingParticipant_NonHostDenied, TestCreateBreakoutRooms_HostOnly,
  TestAssignBreakoutParticipant_NonHost, TestJoinBreakoutRoom_NoAssignment).
  (2) Prueft HandleJoinBreakoutRoom, dass der Raum zum Meeting des Aufrufers gehoert? Das
  Request-Design macht die Frage gegenstandslos, nicht falsch beantwortbar: die Route nimmt
  KEINE Room-ID entgegen (weder Body noch URL-Param, siehe `route_video.go:2295-2311`), der
  Service loest ueber `GetBreakoutAssignmentForUser(meetingID, callerID, tenantID)` exakt die
  eigene Zuweisung des Aufrufers fuer GENAU dieses Meeting auf
  (`internal/work/meeting/service.go:1241-1253`) — ein Cross-Meeting-Zugriff auf einen fremden
  Raum ist mit diesem Vertrag strukturell nicht adressierbar, nicht bloss durch eine Pruefung
  verhindert. Kein Fund, kein Fix noetig.
  Alle 15 Handler nutzen keinen expliziten `getTenantID`/`middleware.GetTenantID`-Aufruf im
  Gateway (gegengeprueft per Zeilen-Scan) — die Aufrufer-Identitaet reist ausschliesslich ueber
  den Outbound-gRPC-Interceptor, deshalb entfaellt fuer diese Unit der sonst uebliche
  "MissingTenant"-Testfall.
- gate: build ok (`./internal/gateway/... ./cmd/gateway/...`) | vet ok | lint ok (0 issues,
  `golangci-lint run --config .golangci.yml ./internal/gateway/...`) | test ok (komplettes
  Paket gruen, `DATABASE_URL` gesetzt, keine Skips) | migration n.a. (keine Tabelle/Policy
  angefasst) | rls-smoke n.a. (reine Unit-Tests, kein neuer DB-Zugriff) | route-drift ok
  (`TestOpenAPIRouteDrift`: 836 registrierte gegen 838 dokumentierte Pfade, PASS — keine neue
  Route)
- coverage: internal/gateway 60,4 % -> 61,2 % (`-coverprofile` vor/nach der neuen Testdatei
  gemessen, Vorher-Lauf mit temporaer entfernter Datei)
- mutations-probe: `Count` im `createBreakoutRoomsHTTPRequest`-Validate-Tag von `max=20` auf
  `max=25` geaendert (Datei vorher per `cp` gesichert). `TestHandleCreateBreakoutRooms_
  CountOutOfBounds` sofort rot (erwartete 400/validation_failed/Feld "count", bekam 503 vom
  Transportfehler, weil Count=21 jetzt gueltig war und bis zur RPC durchlief). Datei per `cp`
  zurueckgedreht, `git diff` danach leer, `./internal/gateway/` anschliessend wieder komplett
  gruen.
- verify vorgaenger: sauber. `1e6691f1` (HR-Analytics/Settings-Gateway-Tests) und `3853a5d6`
  (Journal-SHA-Nachtrag) aendern ausschliesslich Testdateien + Backlog/Journal — kein
  gRPC-Aufruf umgangen, kein `.proto`, keine neue Tabelle/Route/Guard, keine
  Wire-Shape-Aenderung; `git show --stat` gegen beide Commits gegengeprueft.
- neue-units: keine
- offen: Der Backlog-Kopf nennt fuer `route_video.go` "Drei Units teilen sie auf" — die dritte
  ist `cov-video-recording-service-and-repository` (deckt die verbleibenden fuenf
  Recording-Handler + das `internal/work/recording`-Paket ab), noch `status: todo`. Damit sind
  nach dieser Iteration alle drei Video-Units im Backlog vorhanden, keine fehlt.

## Iteration 23 — cov-gateway-video-notes-and-action-items — done — 2026-08-26 03:53
- commit: 5b2bcf94
- gebaut: `backend/internal/gateway/route_video_notes_and_action_items_test.go` — 33 Tests
  fuer die zehn Handler der zweiten Video-Unit (Meeting-Notizen, Aktionspunkte,
  Meeting-Chat, AI-Summary): HandleGetMeetingNotes, HandleSaveMeetingNotes,
  HandleGetPreviousMeetingNotes, HandleGenerateMeetingSummary, HandleCreateActionItem,
  HandleUpdateActionItem, HandleDeleteActionItem, HandleListActionItems,
  HandleSaveMeetingChatMessage, HandleListMeetingChatMessages — je ServiceUnavailable/
  InvalidUUID/ReachesRPC plus Validierungsfaelle (fehlender Content/Description/Message,
  ungueltige assignee_id), im selben Dummy-Registry-Muster wie die Vorgaenger-Iteration
  (kein Fake VideoServiceClient in diesem Paket).
  Schwerpunkt-Fragen der Unit, mit Beleg beantwortet:
  (1) HandleGetPreviousMeetingNotes-Scope: `Service.GetPreviousMeetingNotes`
  (service.go:708) prueft NUR Tenant (ueber `repo.GetMeeting`) und Rekurrenz, KEIN
  Teilnehmer-Kriterium — aber das ist konsistent mit dem Rest des Moduls:
  `Service.GetMeeting` (service.go:186) hat ebenfalls keinen Attendee-Check, Meetings und
  ihre OEFFENTLICHEN Notizen (is_private=false, per SQL-Filter in
  postgres_repository.go:342/372) sind tenant-weit sichtbar by design; nur private Notizen
  sind per author_id gescoped (`GetNotes`, Zeile 323-337). Kein Fund, kein Fix noetig —
  gegengeprueft durch Lesen von service.go und postgres_repository.go.
  (2) HandleGenerateMeetingSummary: die LLM-Zusammenfassung laeuft in
  `Service.GenerateAISummary` (service.go:556), NICHT im Handler — Architekturregel 1
  bereits eingehalten, kein Fund.
  (3) DSAR/Retention fuer Meeting-Notizen/-Chat: NICHT abgedeckt. `dsar_search.go`
  `meetingsModule` (Zeile 2239) liest nur `meetings`-Metadaten, nie `meeting_notes`/
  `meeting_chat_messages`. Kein `retention_*.go`-Handler referenziert eine der beiden
  Tabellen (alle 15 bestehenden Handler gegengeprueft). `crm/consent/scrub.go:51` nennt nur
  `meetings.title/description/agenda` als "accepted residual risk" — fuer Notizen/Chat
  wurde diese Entscheidung nie getroffen. Als Unit angelegt (siehe neue-units).
  Zusaetzlicher, schwerwiegender Fund beim Lesen von `video_grpc.go`:
  `VideoGRPCServer.GetMeetingNotes` (Zeile 1047-1083) liest NIE echte Notizen — es
  simuliert den Read per `SaveNotes(ctx, meetingID, userID, tenantID, "", false)`
  (leerer Content), was `Service.SaveNotes` unbedingt mit `ErrNotesContentRequired`
  ablehnt (service.go:678-681, vor jedem Repo-Zugriff), sodass der Fehlerzweig immer einen
  leeren Stub zurueckgibt. `HandleGetMeetingNotes` am Gateway kann folglich NIEMALS die
  tatsaechlichen Notizen eines Nutzers liefern, unabhaengig davon, was zuvor gespeichert
  wurde. Der Code kommentiert die eigene Verlegenheit selbst ("This is a deviation",
  "We need a different approach"). Als Fix-Unit angelegt (siehe neue-units) — Fix ist
  klein (bestehende `PostgresRepository.GetNotes` fehlt nur eine Service-Methode
  darueber), aber ausserhalb des Gateway-Scopes dieser Unit und eine Verhaltensaenderung,
  die eine Coverage-Unit laut Vorgabe nicht selbst vornimmt.
- gate: build ok (`./internal/gateway/... ./cmd/gateway/...`) | vet ok | lint ok (0 issues,
  `golangci-lint run --config .golangci.yml ./internal/gateway/...`) | test ok (komplettes
  Paket gruen, `DATABASE_URL` gesetzt, 0 Skips) | migration n.a. (keine Tabelle/Policy
  angefasst) | rls-smoke n.a. (reine Unit-Tests, kein neuer DB-Zugriff) | route-drift ok
  (`TestOpenAPIRouteDrift`: 836 registrierte gegen 838 dokumentierte Pfade, PASS — keine
  neue Route)
- coverage: internal/gateway 61,2 % -> 61,8 % (`-coverprofile` vor/nach der neuen
  Testdatei gemessen, Datei temporaer nach /tmp verschoben fuer den Vorher-Lauf)
- mutations-probe: `validate:"required"` von `Message` in `saveMeetingChatMessageRequest`
  entfernt (Datei vorher per `cp` gesichert). `TestHandleSaveMeetingChatMessage_
  MissingMessage` sofort rot (erwartete 400/validation_failed/Feld "message", bekam 503
  vom Transportfehler, weil die leere Message jetzt gueltig war und bis zur RPC durchlief).
  Datei per `cp` zurueckgedreht, `git diff` danach leer, `./internal/gateway/`
  anschliessend wieder komplett gruen.
- verify vorgaenger: sauber. `dabf02be` (Video-Breakout/CoHost-Gateway-Tests) aendert
  ausschliesslich eine neue Testdatei + Backlog/Journal — kein gRPC-Aufruf umgangen, kein
  `.proto`, keine neue Tabelle/Route/Guard, keine Wire-Shape-Aenderung; `git show --stat`
  gegen den Commit gegengeprueft.
- neue-units: fix-video-get-meeting-notes-returns-empty-stub,
  feat-meeting-notes-and-chat-dsar-and-retention-coverage
- offen: Beide neuen Units sind `sonnet`, `deps: []`, direkt baubar. Die DSAR/Retention-
  Unit verlangt vorab eine Entscheidung (a) DSAR/Retention nachziehen oder (b) bewusst als
  "accepted residual risk" in scrub.go dokumentieren — beides ist im `done_when` als
  gleichwertige Option beschrieben, damit der naechste Lauf nicht auf Luke warten muss.

## Iteration 24 — cov-video-recording-service-and-repository — done — 2026-08-26 03:55
- commit: c13a1b14
- gebaut: Dritte Video-Coverage-Unit (`internal/work/recording`, 33,2 % -> 78,1 %).
  Drei neue Testdateien:
  1. `backend/internal/gateway/route_video_recording_lifecycle_test.go` — 14 Tests fuer
     die fuenf im Scope genannten Gateway-Handler: HandleGetRecordingStatus,
     HandleGetRecordingDownloadURL, HandleUpdateRecordingMetadata,
     HandleCleanupExpiredRecording, HandleListRecordingsByMeeting (je
     ServiceUnavailable/InvalidUUID/ReachesRPC, UpdateRecordingMetadata zusaetzlich
     InvalidJSON, da alle Metadata-Felder optional sind und es keinen `validate:"required"`
     gibt).
  2. `backend/internal/work/recording/postgres_repository_real_sql_test.go` — 7 Tests
     gegen echtes Postgres fuer die zuvor 0-%-Repository-Methoden: ListRecordingsByCall/
     ListRecordingsByMeeting, GetRecordingByEgressID, TagRecordingWithConsents
     (inkl. "[]"-Sentinel statt NULL), GetConsents/GetConsentsWithUser/
     CountPendingConsents, ListExpiredRecordings (Status-Filter beweisen: expired+completed
     ja, expired+active nein), ListRecordingsWithAccess + GetRecordingParticipants
     (Teilnehmer sieht, Fremder nicht), MarkInitiatorConsent/GetPreConsentStatus
     (falscher Tenant UND falscher User schlagen beide mit ErrNotFound fehl, nicht mit
     stillem No-Op).
  3. `backend/internal/work/recording/service_lifecycle_test.go` — 24 Tests (Mock-Repo,
     kein DB-Zugriff noetig) fuer die zuvor 0-%-Service-Methoden: GetRecordingStatus,
     ListRecordingsByMeeting (Pagination inkl. letzte Seite und Seite-jenseits-Total),
     UpdateRecordingMetadata (Patch-Semantik, ungueltiger Status abgelehnt),
     TagRecordingWithConsents, GetRecordingConsents (AllConsented-Logik inkl. "kein
     Live-Consent aber Snapshot vorhanden" -> false), CompleteRecordingByEgressID/
     FailRecordingByEgressID, CleanupExpiredRecording (Retention-Gate, NotFound) sowie
     GetRecordingDownloadURL-Vorpruefungen (not-completed, file-url fehlt,
     PermissionDenied fuer Nicht-Teilnehmer) und `fileURLToObjectKey` als
     Tabellentest (6 URL-Formen: https+internal, https+public, s3://, bucket-relativ,
     nackter Key, fuehrender Slash).
  Schwerpunkt-Fragen der Unit, mit Beleg beantwortet:
  (1) HandleGetRecordingDownloadURL Laufzeit/Tenant/Nach-Loeschung: Laufzeit ist fest 1h
  (`downloadExpiry` in service.go:652). Tenant-Bindung laeuft NICHT ueber eine
  WHERE-Klausel im Go-Code (GetRecording filtert nicht nach tenant_id), sondern
  ausschliesslich ueber RLS — verifiziert: `recordings` UND `recording_consents` haben
  `CALL enable_tenant_rls(...)` in Migration 000120, die Policy erzwingt
  `tenant_id = current_tenant_id()`. Das ist die von ADR-006 vorgesehene Architektur
  (RLS statt expliziter Tenant-Parameter), kein Fund. "Nach Loeschung noch gueltig?" —
  JA, mit Einschraenkung: ein bereits ausgestellter Presigned-URL bleibt bis Ablauf
  gueltig, unabhaengig vom DB-Zustand (MinIO kennt keine DB-Transaktion), ABER wenn das
  zugrundeliegende Objekt aus MinIO entfernt wurde (wie im Bulk-Cleanup), liefert der
  Download einen 404 vom Objectstore. Das fuehrt zu Fund (2).
  (2) `Service.CleanupExpiredRecording` (Singular, service.go:555-575) loescht NUR die
  DB-Zeile, ruft nie `s.objectStore()`/`RemoveObject` — im Gegensatz zum Bulk-Pfad
  `CleanupExpiredRecordings` (service.go:396-445), der das Objekt best-effort vor dem
  DB-Delete entfernt. Konsequenz: Video-Dateien bleiben nach Single-Cleanup unbefristet
  in MinIO liegen (DSGVO-Retention-Verletzung + Storage-Leck). Scope-Fehler mit
  `model: opus` als Fix-Unit angelegt (siehe neue-units) — keine Nebenbei-Korrektur laut
  Notiz der Unit.
  (3) DSAR/Retention fuer `recordings`/`recording_consents`: NICHT abgedeckt.
  `dsar_search.go` hat keinen `recordingsModule` (gegengeprueft: kein Modul referenziert
  eine der beiden Tabellen). Kein `retention_*.go`-Handler referenziert sie. `scrub.go`
  nennt sie nicht als "accepted residual risk". Als Unit angelegt (siehe neue-units),
  analog zu `feat-meeting-notes-and-chat-dsar-and-retention-coverage` aus Iteration 23.
  Mutations-Probe (siehe unten) bestaetigt zusaetzlich, dass die Participant-ACL in
  `GetRecordingDownloadURL` wirklich greift und nicht nur durch Zufall gruen ist.
- gate: build ok (`./internal/work/recording/... ./internal/gateway/... ./cmd/work/...
  ./cmd/gateway/...`) | vet ok | lint ok (0 issues, beide Pakete einzeln gegen
  `.golangci.yml`) | test ok (`internal/work/recording` UND `internal/gateway`,
  `DATABASE_URL` gesetzt, 0 Skips in beiden) | migration n.a. (keine Tabelle/Policy
  angefasst, RLS bereits seit Migration 000120 auf `recordings`+`recording_consents`
  aktiv) | rls-smoke ok (ueber die neuen DB-Tests: MarkInitiatorConsent/
  GetPreConsentStatus schlagen bei falschem Tenant mit ErrNotFound fehl, kein stiller
  No-Op; bestehende `rls_test.go` deckt bereits TenantB-sieht-nichts/TenantA-sieht-eigene
  Zeile fuer beide Tabellen ab) | route-drift ok (`TestOpenAPIRouteDrift`: 836 registrierte
  gegen 838 dokumentierte Pfade, PASS — keine neue Route)
- coverage: internal/work/recording 33,2 % -> 78,1 % (`-coverprofile` vor Aenderung
  [nur die drei neuen Testdateien nach /tmp verschoben] und danach gemessen; verbleibende
  Luecken sind `objectStore` 0 % und `GetRecordingDownloadURL` 43,8 % — der Presign-
  Erfolgspfad braucht einen erreichbaren MinIO/S3-Endpunkt bzw. eine Interface-Seam auf
  `*minio.Client`, die es heute nicht gibt; bewusst nicht simuliert, siehe Kommentar in
  service_lifecycle_test.go)
- mutations-probe: In `Service.GetRecordingDownloadURL` (service.go) die ACL-Bedingung
  `if !allowed { return ... PermissionDenied }` zu `if !allowed && false { ... }`
  veraendert (Datei vorher per `cp` gesichert, `allowed` blieb dadurch referenziert und
  der Build kompilierte weiter). `TestGetRecordingDownloadURL_DeniesNonParticipant` sofort
  rot (erwartete codes.PermissionDenied, bekam einen anderen Code, weil der
  Nicht-Teilnehmer jetzt bis zum MinIO-Zugriff durchlief). Datei per `cp` zurueckgedreht,
  `git diff --stat` danach leer, `go test ./internal/work/recording/...` wieder komplett
  gruen.
- verify vorgaenger: sauber. `5b2bcf94` (Video-Notes/ActionItems-Gateway-Tests) aendert
  ausschliesslich eine neue Testdatei + Backlog/Journal — kein gRPC-Aufruf umgangen, kein
  `.proto`, keine neue Tabelle/Route/Guard, keine Wire-Shape-Aenderung; `git show --stat`
  gegen den Commit gegengeprueft.
- neue-units: fix-cleanup-single-recording-orphans-minio-object (opus),
  feat-recording-dsar-and-retention-coverage (sonnet)
- offen: Beide neuen Units sind `deps: []`, direkt baubar. Die Fix-Unit ist ein echter
  Produktionsfehler (DSGVO-Retention + Storage-Leck), keine reine Test-Nacharbeit — sollte
  zeitnah drankommen. Commit-SHA dieser Iteration wird wie ueblich im naechsten
  chore-Commit nachgetragen.


## Iteration 25 — cov-formulare-postgres-repository-real-sql — done — 2026-08-26 04:10
- commit: 2c225f9e
- gebaut: `backend/internal/formulare/postgres_repository_db_test.go` — 18 neue DB-Tests
  gegen PostgresRepository mit echter Postgres-Instanz (kein Mock): UpdateSchema,
  SoftDeleteSchema, ListSchemas (Status/IsTemplate/Search, einzeln und kombiniert),
  DuplicateSchema, GetSubmission, ListSubmissions (Schema+Status), UpdateSubmissionStatus,
  ListSubmissionsForExport (Status+SubmittedAfter+SubmittedBefore kombiniert), GetWebhook,
  UpdateWebhook, DeleteWebhook, ListWebhooks, ClaimPendingDeliveries (System-Kontext
  ueberschreitet Tenants wie im Worker vorgesehen, Tenant-Kontext nicht), MarkDeliveryResult,
  ListWebhookDeliveries, GetWebhookDelivery, GetFormStats, RevokeShareLink. Jede
  Update/Delete/Get-Methode hat einen expliziten Cross-Tenant-Fall (ErrXNotFound statt
  stillschweigendem No-Op), nicht nur den Erfolgspfad.
  Befund (1): `form_webhook_deliveries` hat KEINEN Retention-/Cleanup-Pfad — gegengeprueft
  per Grep ueber `backend/internal/` und `backend/cmd/`, nur INSERT/SELECT/UPDATE in
  `postgres_repository.go`, kein Eintrag in `cmd/auth/main.go`s `retentionRegistry` (die
  dortige `NewFormSubmissionRetentionHandler` deckt nur `form_submissions`). Das `payload`-
  Feld jeder Delivery-Zeile enthaelt eine volle Kopie der Submission-Answers
  (`insertSubmissionTx`, postgres_repository.go:232-241) — derselbe personenbezogene Inhalt
  wie `form_submissions`, aber ohne dessen Retention-Abdeckung, waechst unbegrenzt. Als Unit
  angelegt (siehe neue-units), NICHT selbst gefixt — eine Coverage-Unit aendert kein
  Verhalten, und die Wahl zwischen generischem Retention-Framework und dediziertem Cleanup-
  Job ist eine Entscheidung fuer die bauende Iteration, nicht hier vorwegzunehmen.
  Befund (2, negativ/entkraeftet): `ClaimPendingDeliveries` filtert nicht nach `tenant_id`
  im SQL — sah zunaechst nach derselben Fehlerklasse wie der behobene
  `ListActiveWebhooksForSchema`-Fall (tenant_write_test.go) aus. Gegengeprueft: `worker.go`
  Run() wrapped den Kontext explizit mit `database.WithSystemContext` (worker.go:70) — das
  ist die eine beabsichtigte RLS-Ausnahme auf dieser Tabelle, analog zu
  `GetShareLinkByToken`. Per Test bestaetigt: unter einem gewoehnlichen
  Tenant-Kontext sieht `ClaimPendingDeliveries` nur die eigenen pending Zeilen (RLS greift),
  unter System-Kontext beide Tenants (Worker-Verhalten wie vorgesehen). Kein Fund, aber die
  Grenze ist jetzt durch einen Test belegt statt nur durch Lesen des Codes behauptet.
- gate: build ok (`./internal/formulare/... ./internal/gateway/... ./cmd/gateway/...
  ./cmd/auth/...`) | vet ok | lint ok (0 issues) | test ok (`internal/formulare`,
  `DATABASE_URL` gesetzt, 0 Skips, 95 Tests gesamt inkl. der 18 neuen) | migration n.a.
  (keine Tabelle/Policy angefasst) | rls-smoke ok (bestehende `tenant_isolation_phase2_test.go`
  deckt alle vier Tabellen inkl. `form_submissions` bereits ab; die neuen Cross-Tenant-
  NotFound-Faelle in dieser Iteration pruefen dieselbe Grenze zusaetzlich ueber den
  Repository-Aufruf statt nur per Roh-SQL) | route-drift n.a. (keine Route angefasst,
  `go test ./internal/gateway/` daher nicht Pflicht — trotzdem mitgebaut, siehe gate build)
- coverage: internal/formulare 53,6 % -> 83,2 % (`-coverprofile` vor und nach den drei
  neuen Testdateien gemessen); `postgres_repository.go` im Speziellen: jede zuvor 0,0 %
  Methode (UpdateSchema, SoftDeleteSchema, ListSchemas, DuplicateSchema, GetSubmission,
  ListSubmissions, UpdateSubmissionStatus, ListSubmissionsForExport, GetWebhook,
  UpdateWebhook, DeleteWebhook, ListWebhooks, ClaimPendingDeliveries, MarkDeliveryResult,
  ListWebhookDeliveries, GetWebhookDelivery, GetFormStats, RevokeShareLink) liegt jetzt
  zwischen 80,0 % und 100,0 % (`go tool cover -func` einzeln gegengeprueft)
- mutations-probe: In `ListSchemas` (postgres_repository.go) die Search-Bedingung von
  `"%"+strings.ToLower(filter.Search)+"%"` auf `strings.ToLower(filter.Search)` geaendert
  (Wildcards entfernt, LIKE wird zum Exact-Match). `TestPostgresListSchemas_FiltersByStatus
  TemplateAndSearch` sofort rot ("expected only the support schema, got total=0"). Datei per
  `cp` aus Backup zurueckgedreht, `git diff --stat` auf `postgres_repository.go` danach leer,
  `go test ./internal/formulare/...` wieder komplett gruen. (Ein erster Versuch, die
  Tenant-Filterung aus `GetSubmission`s WHERE-Klausel zu entfernen, war KEINE brauchbare
  Probe: RLS auf `kmuhub_app` blockt den Cross-Tenant-Read unabhaengig vom WHERE, der Test
  blieb gruen — Erkenntnis, nicht nur Ruestzeug, siehe `tenant_write_test.go`-Kommentar
  "muss nicht allein auf RLS vertrauen": hier ist RLS als zweite Verteidigungslinie genau
  das, was die App-Ebene absichert, also war die Probe an der falschen Stelle angesetzt.)
- verify vorgaenger: sauber. `c13a1b14` (Video-Recording-Service/Repository/Gateway-Tests)
  aendert ausschliesslich drei neue Testdateien + Backlog/Journal; `git show --stat`
  gegengeprueft — kein `.proto`, keine neue Route, kein neuer `RequirePermission`-Guard,
  kein direkter Service-Aufruf im Gateway statt Client.
- neue-units: fix-form-webhook-deliveries-unbounded-retention (sonnet)
- offen: Die neue Unit ist `deps: []`, direkt baubar, aber kein akuter Produktionsfehler
  (nur unbegrenztes Wachstum, kein Datenleck ueber die Tenant-Grenze) — kann regulaer in
  Dateireihenfolge drankommen. Commit-SHA dieser Iteration wird wie ueblich im naechsten
  chore-Commit nachgetragen.

## Iteration 26 — cov-gateway-formulare-webhooks-and-submissions — done — 2026-08-26 04:22
- commit: bc322733
- gebaut: 74 neue Handler-Tests in `internal/gateway/route_formulare_test.go` fuer die 14 im
  Scope genannten Handler (HandleListFormSchemas, HandleGetFormSchema, HandleGetSubmission,
  HandleListSubmissions, HandleExportSubmissions, HandleGetFormStats, HandleListWebhooks,
  HandleCreateWebhook, HandleGetWebhook, HandleUpdateWebhook, HandleDeleteWebhook,
  HandleListWebhookDeliveriesForWebhook, HandleListWebhookDeliveries, HandleSubmitByShareToken),
  je Missing-Tenant/Invalid-UUID/Invalid-JSON/Service-Unavailable/ReachesRPC nach dem
  bestehenden Muster der Datei.
  Befund (1, negativ/entkraeftet): SSRF-Absicherung fuer Webhook-Ziele EXISTIERT bereits —
  `internal/formulare/worker.go:61` baut den `http.Client` des Webhook-Workers ueber
  `safehttp.New(...)` (`internal/security/safehttp`), derselbe Guard wie bei der
  Automation-`http.request`-Action. Kein eigener opus-Sicherheitsbefund noetig. Getestet mit
  `TestHandleCreateWebhook_MalformedURL`, das dokumentiert: die Gateway-Validierung
  (`validate:"required,url"`) prueft nur Syntax, die SSRF-Pruefung sitzt bewusst eine Ebene
  tiefer beim Worker (Zustellzeitpunkt), nicht im Gateway (Anfragezeitpunkt) — das Gateway kann
  zur Anfragezeit keine DNS-Aufloesung/Netzklassifizierung leisten.
  Befund (2, negativ/entkraeftet): das Verhalten von `HandleSubmitByShareToken` bei
  abgelaufenem und widerrufenem Token ist bereits vollstaendig auf Service-Ebene abgedeckt
  (`internal/formulare/form_share_test.go`,
  `TestSubmitByShareToken_EveryTokenVerdictIsTheSameNotFound`, deckt unknown/empty/over-long/
  revoked/expired/quota-used-up/schema-private/schema-archived/schema-deleted — alle auf
  denselben `ErrShareLinkNotFound`). Das Gateway ist ein duenner Durchreicher
  (`respondGRPCError` mappt `codes.NotFound` auf 404); eine Dopplung dieser Faelle auf
  Gateway-Ebene wuerde nur den unerreichbaren-Dummy-Client testen, nicht das echte Verhalten.
  Stattdessen zwei NEUE gateway-eigene Faelle getestet: die Token-Laengen-/Leer-Pruefung, die
  VOR jedem RPC-Aufruf im Handler selbst laeuft (`HandleSubmitByShareToken`, Zeile ~1124), und
  der `maxPublicSubmitBody`-Body-Cap (256 KiB) via `http.MaxBytesReader`.
  Befund (3, negativ/entkraeftet): `HandleExportSubmissions` reicht `TenantId` vollstaendig an
  `client.ExportSubmissions` durch; `Service.ExportSubmissions` ruft
  `repo.ListSubmissionsForExport(ctx, FormSchemaID, TenantID, filter)` mit beiden IDs auf —
  die Methode ist laut Vorgaenger-Iteration (`2c225f9e`) bereits 100 % DB-getestet. Kein Fund.
- gate: build ok (`./internal/gateway/... ./cmd/gateway/...`) | vet ok | lint ok (0 issues,
  `.golangci.yml`) | test ok (`internal/gateway`, `DATABASE_URL` gesetzt, 0 Skips, 3073 Tests
  gesamt inkl. `TestOpenAPIRouteDrift`) | migration n.a. (keine Tabelle/Policy angefasst) |
  rls-smoke n.a. (keine neue Tabelle) | route-drift ok (keine Route angefasst, trotzdem
  mitgelaufen als Teil von `go test ./internal/gateway/`)
- coverage: internal/gateway 62,0 % -> 63,1 % (`-coverprofile` vor der Aenderung per
  `git stash`/nach der Aenderung gemessen, Baseline weicht vom `coverage_start`-Bezugswert
  56,6 % ab — fruehere Iterationen dieses Laufs haben das Paket schon angehoben, siehe
  `offen:`). `route_formulare.go` im Speziellen: alle 14 im Scope genannten Handler von 0,0 %
  bzw. teilweise (HandleExportSubmissions 17,6 %, HandleListWebhooks 37,5 %) auf 64,7–96,2 %
  (`go tool cover -func` einzeln gegengeprueft vor/nach).
- mutations-probe: In `HandleSubmitByShareToken` (route_formulare.go) die Token-Laengengrenze
  von `len(token) > 128` auf `len(token) > 256` geaendert. `TestHandleSubmitByShareToken_
  OverLongToken` sofort rot ("status = 503, want 404"). Datei per `cp` aus Backup
  zurueckgedreht, `git diff --stat` auf `route_formulare.go` danach leer, volles
  `go test ./internal/gateway/` wieder komplett gruen (3073 Tests, 0 Skips).
- verify vorgaenger: sauber. `2c225f9e` (Postgres-Repository-Formulare-DB-Tests) aendert
  ausschliesslich eine neue Testdatei + Backlog/Journal; `git show --stat` gegengeprueft —
  kein `.proto`, keine neue Route, kein neuer `RequirePermission`-Guard, kein direkter
  Service-Aufruf im Gateway statt Client.
- neue-units: keine
- offen: Die `coverage_start`-Bezugswerte fuer `internal/gateway` (56,6 %) und
  `route_formulare.go` (32,2 %) im Backlog-Kopf sind vom 2026-08-24-CI-Lauf und liegen
  spuerbar unter der selbst gemessenen Vorher-Zahl (62,0 % / route_formulare.go bereits
  teilweise getestet) — mehrere Iterationen dieses Laufs (u. a. die Video- und
  Postgres-Formulare-Units) haben das Gateway-Paket seit dem CI-Referenzlauf schon angehoben.
  Kein Handlungsbedarf, nur Hinweis fuer die naechste `coverage_start`-Pflege im Backlog-Kopf.
  Commit-SHA dieser Iteration wird wie ueblich im naechsten chore-Commit nachgetragen.

## Iteration 27 — cov-caldav-backend-real-protocol-paths — done — 2026-08-26 05:10
- commit: a1973183
- gebaut: zwei neue Testdateien in `internal/caldav/` fuer `caldav_backend.go` (259/316
  Statements ungedeckt, 18,0 %):
  `caldav_backend_grpc_test.go` (kein DB-Zugriff): `CurrentUserPrincipal`/`CalendarHomeSetPath`
  fuer Nil-User (401) und authentifizierten User (korrekter Pfad); `CreateCalendar` (immer 403);
  `calendarClient()`-Fehlerpropagation ueber `ListCalendars`/`GetCalendar`/`GetCalendarObject`/
  `ListCalendarObjects`/`QueryCalendarObjects`/`PutCalendarObject`/`DeleteCalendarObject` bei
  nicht registriertem bzw. nicht erreichbarem "work"-Service (503), nach demselben
  `emptyRegistry()`/`registryWithService()`-Muster wie `internal/gateway` (kein bufconn-Stub fuer
  `CalendarServiceClient` in diesem Repo — bestaetigt per Grep ueber alle
  `internal/gateway/route_*_test.go`, keine Ausnahme gefunden). Dabei die Zeitfenster-Extraktion
  aus `QueryCalendarObjects` (Top-Level-CompFilter vs. verschachtelter VEVENT-Filter,
  Teil-Override) als reine Funktion `queryTimeRange` herausgezogen (verhaltensgleicher Refactor,
  kein neuer Codepfad) und mit sechs Faellen einzeln getestet — vorher 0 % abgedeckte
  Verzweigungslogik.
  `caldav_backend_db_test.go` (echte Postgres, `kmuhub_app`): `checkCalendarWritePermission`
  fuer Owner/Edit-Mitglied/Admin-Mitglied/View-Mitglied (403)/fremder User (403)/nicht
  existierender Kalender (404) — alle unter `testutil.WithTenantCtx`, um die
  Berechtigungs-Verzweigung selbst zu belegen; `listEventExceptions` fuer sortierte
  Mehrfach-Treffer, leere Liste (nie `nil`).
  Befund (VERIFIZIERTER PRODUKTIONSFEHLER, nicht entkraeftet): `checkCalendarWritePermission`
  und `listEventExceptions` markieren ihren Context nirgends mit `sysctx.With` und erhalten nie
  einen Tenant — anders als `resolveTenantID` in derselben Datei, das genau dafuer einen
  eigenen erklaerenden Kommentar traegt. Der reale CalDAV-Request-Context (Basic Auth,
  `basicAuthMiddleware` -> `CtxWithUser`, nie JWT) traegt nie einen Tenant. Empirisch gegen die
  lokale Postgres bestaetigt (psql als `kmuhub_app`, GUCs leer: 0 Zeilen fuer eine
  nachweislich existierende, dem Owner gehoerende `calendars`-Zeile) UND per Test
  (`TestCheckCalendarWritePermission_RealCalDAVContext_OwnerBlocked`,
  `TestListEventExceptions_RealCalDAVContext_SilentlyEmpty`, beide gruen — sie dokumentieren den
  Bug, aendern ihn nicht). Folge in Produktion: JEDES CalDAV-PUT/DELETE (Apple Kalender,
  Thunderbird/Lightning, DAVx5) scheitert mit 404 "calendar not found", auch fuer den
  tatsaechlichen Kalenderbesitzer — passt zum in der Unit beschriebenen Symptom "mein Kalender
  synchronisiert nicht". Und: Ausnahmen (abgesagte/verschobene Vorkommen wiederkehrender
  Termine) verschwinden lautlos aus dem CalDAV-Export, ohne Fehler. Dasselbe Bug-Muster wurde im
  selben Package fuer `FindActiveByUser`/`UpdateLastUsed` und die `SyncTokenService`-Writes
  bereits gefunden und behoben (`tenant_write_test.go:10-22`) — hier war es nicht mit erledigt.
  Nach den Grenzen dieses Laufs aendert eine cov-Unit kein Verhalten: Fix als eigene Unit ans
  Backlog-Ende gehaengt (siehe `neue-units:`), nicht hier mitgefixt.
  Befund (entkraeftet, kein eigener Test noetig): das in der Unit-Scope geforderte RRULE/DST-Ziel
  ("wiederkehrender Termin ueber einen Sommerzeitwechsel") ist bereits vollstaendig abgedeckt —
  `internal/work/event/rrule_test.go`,
  `TestExpandRecurrence_DSTSpringForward_ShiftsPastMissingHour`, prueft die tatsaechliche
  Recurrence-Expansion (CET vor, CEST nach dem Wechsel). `caldav_backend.go` selbst expandiert
  keine RRULEs — es konsumiert bereits expandierte `ExpandedEventProto`-Listen vom
  Calendar-Service per gRPC; die RRULE-Mathematik gehoert fachlich dorthin, nicht hierher. Eine
  Dopplung des Tests in diesem Package wuerde nur denselben Produktionscode ueber einen anderen
  Pfad erneut treffen.
- gate: build ok (`./internal/caldav/... ./internal/gateway/... ./cmd/gateway/...`) | vet ok |
  lint ok (0 issues, `.golangci.yml`) | test ok (`internal/caldav`, `DATABASE_URL` gesetzt, 118
  Tests, 0 Skips, 0 Fails) | migration n.a. (keine neue Tabelle/Spalte) | rls-smoke n.a. (keine
  Policy geaendert — die Tests BELEGEN eine bestehende Policy-Luecke, aendern sie nicht) |
  route-drift n.a. (keine Route angefasst)
- coverage: internal/caldav 54,9 % -> 64,5 % (`git stash`/`go test -coverprofile` vor und nach
  der Aenderung gemessen; Baseline deckt sich mit `coverage_start`). `caldav_backend.go` im
  Speziellen (per `go tool cover -func` einzeln geprueft): `checkCalendarWritePermission` 0,0 %
  -> 100 %, `listEventExceptions` 0,0 % -> 84,6 %, `queryTimeRange` (neu extrahiert) -> 100 %,
  `CurrentUserPrincipal`/`CalendarHomeSetPath`/`CreateCalendar` 0,0 % -> 100 %,
  `ListCalendars`/`GetCalendar`/`GetCalendarObject`/`ListCalendarObjects`/`QueryCalendarObjects`
  je von 0,0 % auf 37,5–78,6 % (Fehlerpfad + Validierung abgedeckt, Erfolgspfad braucht einen
  bufconn-Stub, den dieses Repo nirgends hat). `PutCalendarObject`/`DeleteCalendarObject`/
  `expandedEventsToObjects` bleiben ueberwiegend ungedeckt (5,7 % / 20,0 % / 0,0 %) — ihr
  Kernpfad haengt vollstaendig am fehlenden RPC-Stub, nur die Pfad-Validierung vorn ist
  erreichbar.
- mutations-probe: in `checkCalendarWritePermission` `permission == "edit" || permission ==
  "admin"` zu `permission == "edit"` verengt.
  `TestCheckCalendarWritePermission_MemberWithAdminAllowed_WithTenantCtx` sofort rot ("received
  unexpected error: 403 Forbidden"). Datei per `cp` aus Backup zurueckgedreht, `git diff --stat`
  auf `caldav_backend.go` zeigt danach nur noch den `queryTimeRange`-Refactor (33
  Einfuegungen/21 Loeschungen ggue. dem committeten Stand), volles `go test
  ./internal/caldav/...` wieder komplett gruen (0 Skips, 0 Fails).
- verify vorgaenger: sauber. `bc322733` (Formulare-Webhook/Submission-Handler-Tests) fuegt
  ausschliesslich eine neue Testdatei hinzu (682 Zeilen); `git show --stat` gegengeprueft — kein
  `.proto`, keine neue Route, kein neuer `RequirePermission`-Guard, kein direkter
  Service-Aufruf im Gateway statt Client (Grep auf `Unimplemented|TODO|FIXME|directly` in der
  Testdatei: kein Treffer).
- neue-units: fix-caldav-write-and-exceptions-blocked-by-missing-tenant-ctx (verifizierter
  Produktionsfehler, siehe Befund oben — status: todo, ans Backlog-Ende gehaengt)
- offen: Die neue Fix-Unit ist bewusst noch NICHT priorisiert eingeordnet (deps: []) — sie steht am
  Backlog-Ende und wird erst gezogen, wenn alle vorherigen `todo`-Units mit erfuellten `deps`
  durch sind. Gegeben die Schwere (CalDAV-Schreiben ist production-weit kaputt) waere eine
  manuelle Vorziehung durch Luke sinnvoll, statt auf die natuerliche Reihenfolge zu warten.

## Iteration 28 — cov-carddav-backend-real-protocol-paths — done — 2026-08-26 04:45
- commit: df4b86a1
- gebaut: Zwei neue Testdateien in `internal/caldav/` fuer `carddav_backend.go` (155/201
  Statements ungedeckt, 22,9 %).
  `carddav_backend_grpc_db_test.go`: echter CRM-gRPC-Server (Loopback-TCP, exakt die
  `middleware.TenantInboundUnaryInterceptor`-Kette aus Production, `contact.Service` +
  `contact.PostgresRepository` gegen echte Postgres), Vorlage
  `internal/gateway/route_hr_manual_entry_idempotency_db_test.go` (Iteration 1). Damit erstmals
  ein echter End-to-End-Roundtrip fuer CardDAV statt nur der Fehlerpfade aus
  `caldav_backend_grpc_test.go` (503 bei nicht registriertem Service).
  Befund 1 (VERIFIZIERTER PRODUKTIONSFEHLER, groesserer Bruder von
  `fix-caldav-write-and-exceptions-blocked-by-missing-tenant-ctx` aus Iteration 27, gleicher Root
  Cause): `CardDAVRoutes.basicAuthMiddleware` setzt nur die User-ID in den Context
  (`caldavpkg.CtxWithUser`), nie `middleware.TenantIDKey`. `registry.GetConnection`s
  `TenantOutboundUnaryInterceptor` haengt `x-tenant-id` deshalb nie an, und `crm_grpc.go`s
  `ListContacts`/`GetContact`/`CreateContact`/`UpdateContact`/`DeleteContact`/
  `UpdateContactVisibility` verlangen ALLE `middleware.GetTenantID(ctx)` und liefern sonst
  `codes.Unauthenticated` → `grpcToWebDAVError` → HTTP 401. Bewiesen (nicht nur hergeleitet) mit
  dem echten CardDAV-Request-Context: `TestListAddressObjects_RealCardDAVContext_NoTenant_Returns401`,
  `TestGetAddressObject_RealCardDAVContext_NoTenant_Returns401`. Folge in Produktion: JEDE
  CardDAV-Operation (Adressbuch-Auflistung, Einzelkontakt, Anlegen/Aendern/Loeschen) schlaegt mit
  401 fehl, auch fuer den echten Kontobesitzer — CardDAV ist in Produktion vollstaendig tot, nicht
  nur teilweise degradiert. Fix-Unit `fix-carddav-missing-tenant-context-blocks-all-operations`
  (opus) ans Backlog-Ende gehaengt, mit Verweis auf den vermutlich gemeinsamen Root-Cause-Fix in
  `basicAuthMiddleware` (Tenant per `resolveTenantID` aufloesen und als echten
  `middleware.TenantIDKey`-Wert setzen — das wuerde beide Units gleichzeitig loesen).
  Um die eigentliche Scope-Frage (done_when #1) trotzdem zu beantworten, injizieren die uebrigen
  Tests den Tenant manuell (`asIfTenantMiddlewareFixed`, so als waere der Root-Cause-Fix bereits
  gelandet) und pruefen die tatsaechliche Listing-Logik:
  `TestListAddressObjects_PersonalBook_OnlyOwnPersonalContacts` (personal = nur eigene, NICHT
  tenant-weit — beabsichtigtes Verhalten, in `postgres_repository.go`s Visibility-Klausel
  korrekt), `TestListAddressObjects_CompanyBook_ReturnsAllSharedRegardlessOfOwner` (shared = alle
  im Tenant, unabhaengig vom Owner).
  Befund 2 (VERIFIZIERTER PRODUKTIONSFEHLER, unabhaengig von Befund 1):
  `TestListAddressObjects_PersonalBook_SilentlyTruncatesPast20` — `ListAddressObjects` fragt mit
  `PageSize: 1000` an ("Reasonable limit for CalDAV sync"), aber
  `contact.Service.ListWithVisibility` klemmt jede PageSize > 100 still auf 20
  (`service.go:722-724`); `ListAddressObjects` liest `resp.Total` nie und paginiert nicht nach.
  25 geseedete Kontakte → 20 zurueckgegeben, kein Fehler. Jedes Adressbuch mit > 20 Kontakten
  synchronisiert dauerhaft unvollstaendig. Fix-Unit
  `fix-carddav-list-address-objects-silently-truncates-past-page-size-limit` (sonnet) ans
  Backlog-Ende gehaengt.
  Befund 3 (done_when #3, anonymisierter Kontakt): `TestGetAddressObject_AnonymizedContact_NoRealNameOrPIIInVCard`
  — `consent.PostgresRepository.AnonymizeContact` setzt `first_name`/`last_name` auf die
  Platzhalter `"Gelöschte"`/`"Person"` und nullt Email/Phone/Notes; `GetContact` liest diese
  Zeile unveraendert, `contactInfoToVCard` mappt sie 1:1 — die vCard traegt danach nachweislich
  keinen Klarnamen und kein Email/Phone/Notes mehr. Kein Bug, Verhalten belegt.
  Feld-Mapping (done_when #2) war bereits vollstaendig durch `carddav_backend_test.go`
  (`TestContactInfoToVCard_*`, Vorlauf-Iterationen) abgedeckt — keine Dopplung noetig.
  `carddav_backend_ownership_db_test.go`: `checkPersonalContactOwnership` (0,0 % vorher, von
  `DeleteAddressObject` fuer "personal" genutzt) — Owner erlaubt, fremder User 403, nicht
  existierender Kontakt 404, kaputter Pfad 400.
- gate: build ok (`./internal/caldav/... ./internal/crm/... ./cmd/gateway/...`) | vet ok | lint ok
  (0 issues, `./internal/caldav/... ./internal/crm/...`) | test ok (`internal/caldav`,
  `DATABASE_URL` gesetzt als `kmuhub_app`, 128 Tests, 0 Skips, 0 Fails) | migration n.a. (keine
  neue Tabelle/Spalte) | rls-smoke n.a. (keine Policy geaendert — die Tests BELEGEN eine
  bestehende Luecke im Tenant-Context, aendern sie nicht) | route-drift n.a. (keine Route
  angefasst, `go test ./internal/gateway/` daher nicht Pflicht)
- coverage: internal/caldav 64,5 % → 68,6 % (`git stash push -u -- backend/internal/caldav/carddav_backend_grpc_db_test.go
  backend/internal/caldav/carddav_backend_ownership_db_test.go` fuer die Vorher-Messung, danach
  `stash pop`; Vorher-Wert deckt sich mit dem Coverage-Stand nach Iteration 27, nicht mit dem
  CI-Referenzwert 54,9 % aus dem Backlog-Kopf — der ist vom 2026-08-24-Lauf und wurde seither von
  mehreren Iterationen angehoben). `carddav_backend.go` im Speziellen (`go tool cover -func`):
  `NewCardDAVBackend` 0,0 % → 100 %, `crmClient` 0,0 % → 75 %, `GetAddressObject` 0,0 % → 86,7 %,
  `ListAddressObjects` 0,0 % → 94,4 %, `checkPersonalContactOwnership` 0,0 % → 100 %.
  `CreateAddressBook`/`DeleteAddressBook`/`ListAddressBooks`/`GetAddressBook`/
  `QueryAddressObjects`/`PutAddressObject`/`DeleteAddressObject` bleiben bei 0,0 % — Put/Delete
  brauchen denselben echten-Server-Ansatz wie hier, aber fuer Schreibpfade (naechster
  natuerlicher Schnitt, nicht Teil dieser Unit).
- mutations-probe: in `contact/postgres_repository.go`s `ListWithVisibility`-Query die
  Bedingung `if filter.VisibilityFilter != ""` zu `if false && filter.VisibilityFilter != ""`
  mutiert (Backup vorher per `cp`). `TestListAddressObjects_PersonalBook_OnlyOwnPersonalContacts`
  blieb gruen (die Owner-Bedingung allein reicht fuer dieses Szenario), aber
  `TestListAddressObjects_CompanyBook_ReturnsAllSharedRegardlessOfOwner` wurde korrekt rot ("has
  3 item(s)" statt 2 — der eigene personal-Kontakt leckte ins company-Listing). Datei per `cp`
  zurueckgedreht, `git diff --stat` auf `postgres_repository.go` danach leer.
- verify vorgaenger: sauber (`a1973183` — `git show --stat` gegengeprueft: `caldav_backend.go`
  liefert nur den angekuendigten verhaltensgleichen `queryTimeRange`-Refactor, kein `.proto`,
  keine Route, kein `RequirePermission`, keine neue Tabelle, kein gRPC-Bypass, kein Stub; der
  dokumentierte Tenant-Bug ist ein BEFUND der beiden `*_db_test.go`-Dateien, keine Code-Aenderung
  an der Produktionslogik)
- neue-units: fix-carddav-missing-tenant-context-blocks-all-operations (opus, VERIFIZIERTER
  Produktionsfehler — CardDAV vollstaendig tot, gleicher Root Cause wie
  `fix-caldav-write-and-exceptions-blocked-by-missing-tenant-ctx`), sowie
  fix-carddav-list-address-objects-silently-truncates-past-page-size-limit (sonnet, VERIFIZIERTER
  Produktionsfehler — stille Truncation bei > 20 Kontakten)
- offen: (1) `BACKLOG.yml`s eigener Vorflug (`hooks/backlog-check.py --preflight`) meldet
  UNABHAENGIG von dieser Iteration einen Bestandsbefund: `fix-409-double-meaning-on-grpc-conflict-routes`
  (aus Iteration 12, siehe dortiger Journal-Eintrag) traegt `status: blocked` +
  `blocked_reason` DIREKT in `BACKLOG.yml`, obwohl der Datei-Kopf das ausdruecklich verbietet
  ("muss vor dem naechsten Lauf hier raus", nach `BACKLOG-NEXT.yml` oder `BACKLOG-PARKED.yml`
  verschieben). Nicht in dieser Iteration behoben (ausserhalb des Scopes von
  `cov-carddav-backend-real-protocol-paths`, und eine Verschiebung ist eine Backlog-Kuration,
  keine Coverage-Aenderung) — Luke sollte das vor dem naechsten Lauf-Start bereinigen, sonst
  bricht `run-loop.ps1`s Preflight (falls er `--preflight` tatsaechlich aufruft) ab. (2) Beide
  neuen Fix-Units enthielten anfangs bare `` `Text` `` YAML-Plain-Scalars direkt nach `- ` ohne
  `>-` — das ist kein gueltiges YAML (Backtick kann kein Token oeffnen), `--preflight` hat das
  sofort gefangen (zwei Runden). Alle vier betroffenen `done_when`-Zeilen auf `>-`-Block-Scalare
  umgestellt und erneut gegen `--preflight` verifiziert (nur noch der oben unter (1) genannte,
  unabhaengige Befund bleibt). (3) `PutAddressObject`/`DeleteAddressObject` bleiben bei 0,0 % —
  natuerlicher Anschluss fuer eine Folge-Unit mit demselben Loopback-Server-Ansatz, aber erst
  sinnvoll NACH `fix-carddav-missing-tenant-context-blocks-all-operations`, sonst 401en auch
  diese Tests nur den bereits bekannten Bug erneut.

## Iteration 29 — cov-gateway-inventar-locations-and-picking — done — 2026-08-26 05:02
- commit: f41baa01
- gebaut: Handler-Tests fuer alle 13 zuvor ungetesteten Inventar-Handler
  (`internal/gateway/route_inventar_test.go`, Muster der bestehenden Datei: Validation,
  MissingTenant, ServiceUnavailable, ReachesRPC): HandleListLocations, HandleCreateLocation,
  HandleGetLocation, HandleUpdateLocation, HandleDeleteLocation, HandleListPickingLists,
  HandleCreatePickingList, HandleGetPickingList, HandleUpdatePickingList,
  HandleDeletePickingList, HandleUpsertPickingListItem, HandleDeletePickingListItem,
  HandleBookPickingList. Zusaetzlich ein DB-gestuetzter Test in
  `internal/inventar/postgres_repository_test.go`
  (`TestSoftDeleteLocation_ItemStillReferencingIt_LeavesDanglingLocationID`) fuer den
  Referenz-Integritaets-Teil von `done_when`. Idempotenz von HandleBookPickingList (kein
  Doppel-Abzug) und Atomaritaet einer fehlgeschlagenen Kommissionierliste (keine Teilbuchung)
  waren bereits VOR dieser Iteration vollstaendig durch Bestandstests belegt — service-seitig
  (`internal/inventar/picking_service_test.go`:
  TestBookPickingList_SecondBookingDoesNotMoveStockAgain,
  TestBookPickingList_InsufficientStockLeavesEverythingUntouched) und repo-seitig gegen echtes
  Postgres (`internal/inventar/picking_booking_tx_test.go`:
  TestBookPickingListTx_UpsertConflictAndClaimAgainstRealSchema,
  TestBookPickingListTx_PartialFailureRollsBackClaimAndStock) — keine Dopplung gebaut, stattdessen
  in einem Doc-Kommentar an den neuen HandleBookPickingList-Tests referenziert.
  Befund beim Bauen: `Service.DeleteLocation` -> `PostgresRepository.SoftDeleteLocation` prueft nie,
  ob Items noch per `location_id` auf den Lagerort verweisen. Die FK-Klausel `ON DELETE SET NULL`
  (`migrations/000184_inventory_locations.up.sql`) greift nur bei einem echten `DELETE`, das hier
  nie passiert — nach dem Soft-Delete bleibt `item.location_id` unveraendert auf einen Lagerort
  zeigen, den `GetLocation`/`ListLocations` als nicht mehr existent behandeln. Belegt durch den
  neuen Repo-Test, als Fix-Unit ans Backlog-Ende gehaengt (siehe neue-units).
- gate: build ok (`./internal/gateway/... ./internal/inventar/... ./cmd/gateway/...`) | vet ok
  | lint ok (0 issues, `./internal/gateway/... ./internal/inventar/...`) | test ok
  (`internal/inventar` 67 PASS/0 SKIP/0 FAIL, `internal/gateway` gruen inkl.
  TestOpenAPIRouteDrift — 836 Routen gegen 838 Spec-Pfade, keine Route in dieser Unit angefasst,
  Pflichtlauf trotzdem gruen) | migration n.a. (keine neue Tabelle/Spalte) | rls-smoke n.a.
  (keine neue Tabelle/Policy — der neue Repo-Test dokumentiert eine bestehende
  Referenz-Integritaets-Luecke, aendert aber keine Policy)
- coverage: internal/gateway 63,1 % -> 63,9 % (`git stash push -u -- backend/internal/gateway/route_inventar_test.go
  backend/internal/inventar/postgres_repository_test.go` fuer die Vorher-Messung, danach `stash pop`;
  route_inventar.go im Speziellen, `go tool cover -func`: alle 13 Ziel-Handler von 0,0 % auf 80-95 %,
  z. B. HandleBookPickingList 0,0 % -> 87,0 %, HandleDeleteLocation 0,0 % -> 81,2 %,
  HandleListPickingLists 0,0 % -> 95,5 %). internal/inventar 72,9 % -> 72,9 % (unveraendert —
  der neue Test deckt SoftDeleteLocation/GetItem, die bereits vollstaendig abgedeckt waren; der
  Test belegt Verhalten, kein neues Coverage-Ziel, siehe `gebaut:`).
- mutations-probe: in `HandleListPickingLists` (route_inventar.go) den `default:`-Zweig des
  Status-Filter-Switch von `response.Error(400, ...)` auf `grpcReq.Status = &raw` geaendert
  (cp-Sicherung vorher). `TestHandleListPickingLists_InvalidStatusFilter` wurde rot (503 statt
  400, "connection error" statt "invalid status filter"), per `cp` zurueckgedreht, `git diff`
  gegen `route_inventar.go` danach leer, betroffener Test wieder gruen.
- verify vorgaenger: sauber (`df4b86a1` — `git show --stat` gegengeprueft: ausschliesslich zwei
  neue DB-Testdateien gegen einen echten CardDAV/CRM-gRPC-Server, kein `.proto` im Diff, keine
  Route, kein `RequirePermission`, keine neue Tabelle, kein gRPC-Bypass, kein Stub; die beiden
  dokumentierten Produktionsfehler (fehlender Tenant-Context, stille 20er-Truncation) wurden
  bereits in Iteration 28 korrekt als `neue-units:` angehaengt)
- neue-units: fix-inventar-location-soft-delete-leaves-dangling-item-references (sonnet,
  VERIFIZIERTER Befund — SoftDeleteLocation laesst `location_id` an referenzierenden Items
  unveraendert stehen statt sie zu bereinigen oder das Loeschen abzulehnen; Blast-Radius aktuell
  begrenzt, weil `location_id` ueber REST derzeit nicht setzbar ist, siehe scope der Unit)
- offen: (1) Der bereits aus Iteration 28 bekannte, unabhaengige Preflight-Befund
  (`fix-409-double-meaning-on-grpc-conflict-routes` traegt `status: blocked` DIREKT in
  `BACKLOG.yml` statt in `BACKLOG-NEXT.yml`/`BACKLOG-PARKED.yml`) besteht unveraendert fort —
  ausserhalb des Scopes dieser Unit, Luke sollte ihn vor dem naechsten Lauf-Start bereinigen.
  (2) `HandleGetStockReport`, `HandleExportInventory`, `HandleListInventurSessions`,
  `HandleGetInventurSession`, `HandleListItemAttachments` bleiben bei 0,0 % in route_inventar.go —
  nicht Teil dieser Unit (Scope war ausdruecklich auf Locations+Picking begrenzt), natuerlicher
  Anschluss fuer die zweite der "zwei Inventar-Units" aus dem `scope`-Text.


## Iteration 30 — cov-gateway-inventar-inventur-and-reports — done — 2026-08-26 05:11
- commit: e9c316b0
- gebaut: Handler-Tests (ServiceUnavailable, MissingTenant/InvalidIDUUID, ReachesRPC, plus
  Validierungs-Faelle) fuer die acht in dieser Unit genannten Handler: HandleGetStockReport,
  HandleExportInventory, HandleListInventurSessions, HandleGetInventurSession,
  HandleDeleteInventurSession, HandleListItemAttachments, HandleCreateItemAttachment,
  HandleDeleteItemAttachment. Zusaetzlich zwei Service-Tests, die reale Produktionsluecken
  belegen statt nur Zeilen abzudecken:
  `TestDeleteInventurSession_CompletedSessionIsNotProtected`
  (`internal/inventar/inventur_booking_test.go`) und
  `TestService_GetStockReport_SumsAcrossIncompatibleUnits` (`internal/inventar/service_test.go`).
  Befund 1 (HGB Paragraph 240): `Service.DeleteInventurSession` (service.go:878) prueft
  `session.Status` nicht, im Unterschied zu `UpdateInventurSessionStatus` und
  `UpsertInventurCount`, die beide korrekt gegen `ErrInventurAlreadyCompleted` absichern — eine
  abgeschlossene Inventur ist per Test nachweislich vollstaendig loeschbar. Befund 2 (Mengeneinheiten):
  `Service.GetStockReport` summiert `item.Quantity` ueber alle Items ohne `item.Unit` zu pruefen
  (50 Stk + 30 kg -> `TotalQuantity: 80`), dieselbe Fehlerklasse wie die drei blinden
  Waehrungssummierungen aus Lauf 11. `HandleExportInventory`/`ExportInventory` ist NICHT betroffen
  (CSV-Export schreibt pro Zeile mit eigener `unit`-Spalte, keine Summierung). Beide Befunde als
  eigene Fix-Units ans Backlog-Ende gehaengt (siehe neue-units), nicht selbst gefixt — eine
  Coverage-Unit aendert kein Verhalten.
- gate: build ok (`./internal/inventar/... ./internal/gateway/... ./cmd/gateway/...`) | vet ok
  | lint ok (0 issues, `./internal/inventar/... ./internal/gateway/...`) | test ok
  (`internal/inventar` gruen 0 Skip/0 Fail, `internal/gateway` gruen inkl. TestOpenAPIRouteDrift —
  836 Routen gegen 838 Spec-Pfade, keine Route in dieser Unit angefasst, Pflichtlauf trotzdem
  gruen) | migration n.a. (keine neue Tabelle/Spalte) | rls-smoke n.a. (keine neue Tabelle/Policy)
- coverage: internal/gateway 63,9 % -> 64,3 % (`go tool cover -func` vor/nach den neuen Tests in
  route_inventar_test.go; alle acht Ziel-Handler von 0,0 % auf 75-100 %, z. B.
  HandleGetStockReport 0,0 % -> 100 %, HandleDeleteInventurSession 0,0 % -> 85,7 %,
  HandleCreateItemAttachment 0,0 % -> 85,0 %). internal/inventar 72,9 % -> 73,1 % (die beiden
  neuen Befund-Tests treffen bereits erreichten Code in GetStockReport/DeleteInventurSession,
  Zuwachs kommt vor allem aus den neuen Assertions selbst).
- mutations-probe: in `Service.GetStockReport` (service.go:588) `totalQty += item.Quantity` auf
  `totalQty += item.Quantity * 2` geaendert (cp-Sicherung vorher). Beide GetStockReport-Tests
  wurden rot (`TestService_GetStockReport_Success`: erwartet 53, bekam 106;
  `TestService_GetStockReport_SumsAcrossIncompatibleUnits`: erwartet 80, bekam 160), per `cp`
  zurueckgedreht, `git diff` gegen `service.go` danach leer, beide Tests wieder gruen.
- verify vorgaenger: sauber (`f41baa01` — `git show --stat` gegengeprueft: ausschliesslich
  `route_inventar_test.go`/`postgres_repository_test.go` plus BACKLOG.yml-Statuszeile, kein
  `.proto` im Diff, keine Route, kein `RequirePermission`, keine neue Tabelle, kein gRPC-Bypass
  im neuen Testcode — Muster `registryWithService`/`emptyRegistry` erreicht den echten gRPC-Client
  ueber einen unerreichbaren Port, kein Stub)
- neue-units: fix-inventur-delete-completed-session-unprotected (opus, wie im scope-Text
  gefordert — VERIFIZIERTER Befund, HGB-Aufbewahrungspflicht verletzt),
  fix-inventar-stock-report-blind-unit-sum (sonnet, VERIFIZIERTER Befund — blinde Summe ueber
  Mengeneinheiten, Proto-Aenderung an StockReportResponse vermutlich noetig)
- offen: (1) `fix-409-double-meaning-on-grpc-conflict-routes` traegt weiterhin `status: blocked`
  direkt in BACKLOG.yml statt in BACKLOG-NEXT.yml/BACKLOG-PARKED.yml — `--preflight` bricht
  deswegen mit Exit 1 ab (nicht fatal fuer den Lauf, aber Luke sollte es vor dem naechsten
  Lauf-Start bereinigen, seit Iteration 28 unveraendert offen). (2) Anhaenge (`HandleCreateItemAttachment`)
  koennen laut scope-Text personenbezogene Daten tragen (Fotos mit Personen) — geprueft:
  `inventory_item_attachments` (migration 000240) hat keine FK auf einen Nutzer/Mitarbeiter,
  nur `item_id`; kein Treffer in `internal/security/gdpr` fuer `item_attachment`/`inventur_session`.
  Strukturell also kein DSAR/Retention-Anschlusspunkt vorhanden (Inhalt der Fotos selbst ist
  ausserhalb automatisierter SQL-Scrubbing-Reichweite) — kein Fund, kein Fix-Unit angelegt.

## Iteration 31 — cov-gateway-rapporte-measurements — done — 2026-08-26 05:20
- commit: 3f151a26
- gebaut: Handler-Tests (MissingTenant/InvalidIDUUID/ServiceUnavailable/ReachesRPC plus
  Validierungs-Faelle) fuer alle sieben in dieser Unit genannten Handler: HandleListMeasurements,
  HandleCreateMeasurement, HandleGetMeasurement, HandleUpdateMeasurement, HandleDeleteMeasurement,
  HandleAddMeasurementPosition, HandleDeleteMeasurementPosition. Zusaetzlich drei Repo-Tests, die
  reale Befunde belegen statt nur Zeilen abzudecken (`internal/rapporte/postgres_repository_test.go`):
  `TestDeleteMeasurement_CascadesToPositions` (Punkt 2 aus dem scope-Text: KEIN Fund — die
  ON-DELETE-CASCADE-FK aus Migration 000163 greift nachweislich, keine verwaisten Zeilen),
  `TestAddMeasurementPosition_PreservesFractionalQuantityAndRoundsUnitPrice` (krumme Menge
  12,3456 exakt bei NUMERIC(12,4), unit_price 45,999 korrekt auf 46,00 bei NUMERIC(12,2)
  gerundet — kein Fund, Rundung verhaelt sich wie erwartet) und
  `TestAddMeasurementPosition_AcceptsMeasurementIDFromAnotherTenant` (VERIFIZIERTER Befund).
  Befund: `PostgresRepository.AddMeasurementPosition` (postgres_repository.go:672) prueft nicht,
  ob `measurementID` zu `tenantID` gehoert — der FK auf `measurements(id)` prueft nur Existenz,
  die RLS-Policy (`enable_tenant_rls`, migration 000118) prueft nur den `tenant_id`-Wert der NEUEN
  Zeile selbst. Tenant A kann damit eine Position (mit `quantity`/`unit_price`, abrechnungsrelevant)
  an jede bekannte `measurement_id` haengen, auch an eine Messung von Tenant B — das Insert gelingt
  fehlerfrei, Tenant B sieht die fremde Zeile nie ueber den normalen Lesepfad, sie wird aber beim
  Loeschen von Tenant B's Messung per CASCADE stillschweigend mitgeloescht. Als eigene opus-Unit
  ans Backlog-Ende gehaengt (siehe neue-units), nicht selbst gefixt — eine Coverage-Unit aendert
  kein Verhalten. Punkt (3) aus dem scope-Text (bereits abgerechnetes Aufmass aenderbar?) ist
  gegenstandslos: `grep -rn measurement internal/biz/invoice/*.go internal/rapporte/service.go`
  findet keine Verbindung zwischen Rapport-Aufmassen und Rechnungen — es existiert kein
  `internal/rapporte/service.go`-Layer fuer Measurements ueberhaupt, `RapporteGRPCServer` ruft
  `s.repo.*Measurement*` direkt (kein Business-Logik-Bruch, da fuer diese sieben RPCs schlicht
  keine Business-Logik jenseits Tenant-Scoping noetig ist ausser dem jetzt gefundenen Fehlen davon).
- gate: build ok (`./internal/rapporte/... ./internal/gateway/... ./cmd/gateway/...`) | vet ok
  | lint ok (0 issues, `./internal/rapporte/... ./internal/gateway/...`) | test ok
  (`internal/rapporte` gruen 0 Skip/0 Fail, `internal/gateway` gruen inkl. TestOpenAPIRouteDrift,
  keine Route in dieser Unit angefasst, Pflichtlauf trotzdem gruen)
  | migration n.a. (keine neue Tabelle/Spalte) | rls-smoke n.a. (keine neue Policy — der Befund
  betrifft eine bestehende Policy, wird in der Fix-Unit behandelt)
- coverage: internal/gateway 64,3 % -> 64,7 % (`go tool cover -func` vor/nach, gemessen per
  `git stash` auf den Ausgangsstand; alle sieben Ziel-Handler von 0,0-37,5 % auf 68,8-94,1 %,
  z. B. HandleListMeasurements 0,0 % -> 94,1 %, HandleGetMeasurement 0,0 % -> 81,2 %,
  HandleAddMeasurementPosition 0,0 % -> 84,2 %). internal/rapporte 76,1 % -> 76,1 % (keine
  Verschiebung auf Paketebene — die drei neuen Repo-Tests treffen bereits erreichten Code in
  AddMeasurementPosition/GetMeasurement/DeleteMeasurement, Zuwachs kommt aus den neuen
  Assertions selbst, nicht aus neuen Statements).
- mutations-probe: in `AddMeasurementPosition` (postgres_repository.go:694) die INSERT-Parameter
  `p.Quantity, p.UnitPrice` zu `p.UnitPrice, p.Quantity` vertauscht (Backup vorher per `cp`).
  `TestAddMeasurementPosition_PreservesFractionalQuantityAndRoundsUnitPrice` wurde sofort rot
  ("expected quantity 12.3456 preserved at 4 decimals, got 45.9990"), per `cp` zurueckgedreht,
  `git diff --stat` gegen `postgres_repository.go` danach leer, Testlauf beider Pakete wieder gruen.
- verify vorgaenger: sauber (`e9c316b0` — `git show --stat` gegengeprueft: nur
  `route_inventar_test.go`/`inventur_booking_test.go`/`service_test.go` plus BACKLOG.yml, kein
  `.proto` im Diff, keine Route, kein `RequirePermission`, keine neue Tabelle, kein gRPC-Bypass —
  Muster `registryWithService`/`emptyRegistry` erreicht den echten gRPC-Client ueber einen
  unerreichbaren Port, kein Stub, Grep auf Unimplemented/TODO/FIXME im Diff leer)
- neue-units: fix-rapporte-measurement-position-cross-tenant-insert (opus — VERIFIZIERTER Befund,
  Tenant-Isolationsluecke auf dem Schreibpfad, abrechnungsrelevante Felder betroffen)
- offen: (1) `fix-409-double-meaning-on-grpc-conflict-routes` traegt weiterhin `status: blocked`
  direkt in BACKLOG.yml (seit Iteration 28 unveraendert, siehe dortige offen-Zeilen) —
  `--preflight` bricht deswegen weiter mit Exit 1 ab, nicht fatal fuer den Lauf. (2) Die neue
  Fix-Unit setzt voraus, dass ein aehnliches Muster NICHT bei `DeleteMeasurementPosition` vorliegt
  (dort wird nur per `id=$1 AND tenant_id=$2` geloescht, kein Fremd-Tenant-Zugriff moeglich, weil
  das Loeschen selbst tenant-gescoped ist und keine fremde Zeile trifft) — beim Fixen trotzdem
  gegenpruefen, ob dasselbe Copy-Paste-Muster (Insert mit fremder FK ohne Tenant-Check) noch
  woanders im `rapporte`-Paket vorkommt (z. B. `AddLine`, `CreateAttachment` an eine fremde
  `report_id`) — nicht Teil dieser Unit untersucht, nur bei den Measurement-Handlern.

## Iteration 32 — cov-gateway-rapporte-lines-attachments-export — done — 2026-08-26 05:31
- commit: 94076a6d
- gebaut: Handler-Tests (MissingTenant/InvalidIDUUID/ServiceUnavailable/ReachesRPC plus
  Validierungsfaelle) fuer alle zwoelf in dieser Unit genannten Handler: HandleUpdateLine,
  HandleDeleteLine, HandleListLines, HandleDeleteAttachment, HandleListAttachments,
  HandleGetTemplate, HandleUpdateTemplate, HandleDeleteTemplate, HandleListPendingApprovals,
  HandleGetReportStats, HandleSaveReportSignature, HandleExportPDF
  (`internal/gateway/route_rapporte_test.go`). Dazu ein Repo-Test gegen die echte DB
  (`internal/rapporte/postgres_repository_test.go`), der Punkt (1) aus dem scope-Text
  beantwortet: `TestSaveSignature_OverwritesExistingSignatureWithoutGuard` — VERIFIZIERTER
  Befund. Punkt (2) aus dem scope-Text (Scope von HandleListPendingApprovals) beantwortet:
  `TestHandleListPendingApprovals_OwnScopeWithoutUserIsRejected` +
  `TestHandleListPendingApprovals_ReachesRPCWithAuthorFilterAtAllScope` — kein Fund, der
  Handler nutzt denselben `ownerFilterForScope("rapporte:report","read")`-Pfad wie
  HandleListReports korrekt. Punkt (3) (HandleExportPDF Tenant-Filter) — kein Fund,
  `ExportPDFRequest.TenantId` kommt ausschliesslich aus dem authentifizierten Context, nie aus
  der URL.
  Signatur-Muster fuer die vertraege/vermietung-Units (HandleSaveContractSignature,
  HandleSaveRentalSignature): der Gateway-Handler selbst hat KEINEN Overwrite-Guard (thin
  Parse/Call/Respond) — die Entscheidung liegt vollstaendig im Service/Repo. Bei rapporte
  fehlt sie dort ebenfalls: `PostgresRepository.SaveSignature` (postgres_repository.go:911)
  hat kein "AND signature_data IS NULL" und keinen Status-Check, `Service.SaveSignature`
  (service.go:599) validiert nur Format/Groesse/Pflichtfelder der neuen Signatur. Ein bereits
  signierter Report akzeptiert klaglos eine zweite Signatur und ueberschreibt sie ohne Spur der
  ersten — als eigene Fix-Unit angelegt (siehe neue-units), nicht selbst gefixt, eine
  Coverage-Unit aendert kein Verhalten. Die vertraege/vermietung-Units sollten explizit
  gegenpruefen, ob sie dasselbe Muster teilen.
- gate: build ok (`./internal/rapporte/... ./internal/gateway/... ./cmd/gateway/...`) | vet ok
  | lint ok (0 issues, `./internal/rapporte/... ./internal/gateway/...`) | test ok
  (`internal/rapporte` gruen 0 Skip/0 Fail, `internal/gateway` gruen inkl.
  TestOpenAPIRouteDrift — keine Route in dieser Unit angefasst, Pflichtlauf trotzdem gruen,
  836 registrierte Routen gegen 838 dokumentierte Pfade geprueft)
  | migration n.a. (keine neue Tabelle/Spalte) | rls-smoke n.a. (keine neue Policy angefasst)
- coverage: internal/gateway 64,7 % -> 65,4 % (`go tool cover -func` vor/nach, Vorher-Wert per
  `git stash`/`git stash pop` auf den Ausgangsstand gemessen; alle zwoelf Ziel-Handler von
  0,0-32,2 % auf 61,5-100,0 %, z. B. HandleListTemplates/HandleCreateTemplate bleiben 0,0 %
  ausserhalb dieser Unit, HandleUpdateLine 0,0 % -> 79,2 %, HandleSaveReportSignature
  0,0 % -> 84,2 %, HandleListPendingApprovals 0,0 % -> 94,1 %). internal/rapporte 76,1 % ->
  76,8 % (der neue Signature-Repo-Test trifft ueberwiegend bereits erreichten Code, Zuwachs
  kommt aus der zweiten `GetReport`-Assertion am Testende).
- mutations-probe: in `SaveSignature` (postgres_repository.go:915) die SQL-Platzhalter
  `$3, $4` (signed_by, signature_data) vertauscht zu `$4, $3` (Backup vorher per `cp`).
  `TestSaveSignature_OverwritesExistingSignatureWithoutGuard` wurde sofort rot ("expected
  first signature persisted, got ..." — SignatureData/SignedBy vertauscht), per `cp`
  zurueckgedreht, `git diff --stat` gegen `postgres_repository.go` danach leer, beide Pakete
  (`internal/rapporte`, `internal/gateway`) wieder gruen.
- verify vorgaenger: sauber (`3f151a26` — `git show --stat` gegengeprueft: nur
  `route_rapporte_test.go`/`postgres_repository_test.go` plus BACKLOG.yml, kein `.proto` im
  Diff, keine neue Route, kein `RequirePermission`, keine neue Tabelle, kein gRPC-Bypass; Grep
  auf RequirePermission/Unimplemented/TODO/FIXME/.proto im Testdiff leer)
- neue-units: fix-rapporte-signature-overwritable-after-signing (sonnet — VERIFIZIERTER
  Befund, Signatur unbegrenzt ueberschreibbar nach dem Signieren, kein Guard in Service noch
  Repo)
- offen: (1) `fix-409-double-meaning-on-grpc-conflict-routes` traegt weiterhin `status: blocked`
  direkt in BACKLOG.yml (seit Iteration 28 unveraendert) — `--preflight` bricht deswegen weiter
  mit Exit 1 ab, nicht fatal fuer den Lauf. (2) Die neue Fix-Unit laesst offen, ob es einen
  legitimen "Unsign"-Workflow (Signatur-Widerruf durch Reviewer) geben soll statt eines
  einfachen Ablehnungs-Guards — das ist eine Entscheidung von Luke, nicht Teil der Fix-Unit
  selbst. (3) Beim Fixen der neuen Unit gegenpruefen, ob `HandleSaveContractSignature`
  (vertraege) und `HandleSaveRentalSignature` (vermietung) dasselbe Overwrite-Muster teilen —
  in dieser Iteration nur rapporte selbst untersucht.

## Iteration 33 — cov-document-file-repository-and-service — done — 2026-08-26 05:38
- commit: 409850d3
- gebaut: neue DB-Testdatei `internal/document/file/postgres_repository_file_test.go` (28 Tests)
  fuer die bislang komplett ungetestete Kern-CRUD/Versionierung von `postgres_repository.go`:
  `Create`, `GetByID` (inkl. Cross-Tenant), `List` (Default-Ausschluss geloeschter Dateien,
  `IsDeleted`-Filter, Folder/Owner/Favorite-Filter, Sortierung, Tag-Filter mit
  `HAVING COUNT(DISTINCT tag_id) = n`-Semantik — "alle Tags", nicht "irgendein Tag"), `Update`
  (Rename+Move, Cross-Tenant-Ablehnung, No-Op ohne Felder), `SoftDelete` (setzt Flag+Timestamp,
  zweiter Aufruf auf bereits geloeschter Datei liefert `ErrFileNotFound` durch
  `AND NOT is_deleted`), `CreateVersion`/`ListVersions`/`GetVersion`/`GetVersionByID`
  (fremde Datei darf fremde Version nicht ueber die eigene File-ID erreichen)/
  `UpdateCurrentVersion`/`UpdateSearchContent`. Ergaenzend in `service_test.go`: die bislang
  0,0-%-Funktionen `Register`, `ListVersions`/`RevertVersion`/`ListActivity`/`LogDownload`
  (Service-Wrapper), `SetEventEmitter`.
  VERIFIZIERTER BEFUND (scope-Punkt 2, Freigabe-Pfade): `CreateShareLink` und `RedeemShareLink`
  sind die einzigen beiden Lesepfade in `service.go`, die NICHT `file.IsDeleted` pruefen — jeder
  andere tenant-gescopte Lesepfad (GetDownloadURL, LinkToEntity, Move, Copy, CreateVersion,
  RevertVersion, GetVersionDownloadURL) tut das. Da `Delete` bewusst nicht aus MinIO loescht,
  bedeutet das: eine bereits geloeschte Datei kann weiterhin einen neuen oeffentlichen Share-Link
  bekommen, UND ein vor dem Loeschen erzeugter Link liefert den Download unveraendert weiter —
  die Sichtbarkeit endet nicht mit dem Loeschen der Datei. Dokumentiert durch
  `TestCreateShareLink_DeletedFile_NotRejected` und
  `TestRedeemShareLink_DeletedFile_StillDownloadable` (beide pruefen aktuell `err == nil`, siehe
  Testkommentar "gap:"). Nicht selbst gefixt (Coverage-Unit aendert kein Verhalten) — als
  Fix-Unit angelegt, siehe neue-units.
  Scope-Punkt 1 (Ordner-Zyklus in `HandleGetFolderPath`/`GetPath`, `internal/document/folder`,
  separates Package): `folder/service.go:170-185` blockt jeden Zyklus bereits VOR dem
  SQL-Aufruf — `Update` prueft `IsDescendant(newParentID, id)` und lehnt mit `ErrCircularParent`
  ab, `Create` verlangt einen bereits existierenden Parent (kann also nie auf sich selbst
  zeigen). Ein Zyklus ist ueber den regulaeren Service-Pfad nicht erzeugbar. Die zugrundeliegende
  `WITH RECURSIVE`-Query in `postgres_repository.go:GetPath`/`IsDescendant` bleibt trotzdem ohne
  eigene Tiefenbegrenzung (Verteidigung nur auf Service-Ebene, nicht auf DB-Ebene) — kein DB-Test
  mit kuenstlich injiziertem Zyklus gebaut, da `internal/document/folder` ausserhalb des
  `done_when`-Zielpakets dieser Unit liegt (dort misst `go test ./internal/document/file/`) und
  der Service-Guard den praktischen Fall bereits abdeckt. Siehe offen.
- gate: build ok (`./internal/document/... ./internal/gateway/... ./cmd/gateway/...`) | vet ok
  | lint ok (0 issues, `./internal/document/file/...`) | test ok (`internal/document/file` gruen,
  126 Tests, 0 Skip/0 Fail, `DATABASE_URL` gesetzt) | migration n.a. (keine neue Tabelle/Spalte)
  | rls-smoke ok (`document_files`, Tenant `164e9ab5-...`: eigener Tenant 2, fremder Tenant 0)
- coverage: internal/document/file 55,8 % -> 84,0 % (`go tool cover -func` vor/nach). Datei-Ebene:
  `postgres_repository.go` 38,2 % -> ca. 90 % (Create/GetByID/CreateVersion/UpdateSearchContent
  100 %, List/Update/SoftDelete/Versioning-Methoden 80-90 %), `service.go` 66,3 % -> ca. 85 %
  (SetEventEmitter/LogDownload/ListActivity/ListVersions 100 %, Register 87,0 %,
  RevertVersion 78,6 %).
- mutations-probe: in `SoftDelete` (postgres_repository.go:206) die Klausel `AND NOT is_deleted`
  entfernt (Backup vorher per `cp`). `TestPostgresRepository_SoftDelete_AlreadyDeleted_NotFound`
  wurde sofort rot ("Expected error with 'file not found' in chain but got nil"), per `cp`
  zurueckgedreht, `git diff --stat` gegen `postgres_repository.go` danach leer.
- verify vorgaenger: sauber (`94076a6d` — `git show --stat` gegengeprueft: nur
  `route_rapporte_test.go`/`postgres_repository_test.go` plus BACKLOG.yml/JOURNAL.md im Diff,
  kein `.proto`, keine neue Route, kein `RequirePermission`, keine neue Tabelle, kein
  gRPC-Bypass)
- neue-units: fix-document-share-link-survives-file-deletion (sonnet — VERIFIZIERTER Befund,
  CreateShareLink/RedeemShareLink pruefen `file.IsDeleted` nicht, Share-Link ueberlebt das
  Loeschen der Datei)
- offen: (1) `fix-409-double-meaning-on-grpc-conflict-routes` traegt weiterhin `status: blocked`
  direkt in BACKLOG.yml (seit Iteration 28 unveraendert), `--preflight` bricht deswegen weiter
  mit Exit 1 ab, nicht fatal fuer den Lauf. (2) Die `WITH RECURSIVE`-Abfragen in
  `internal/document/folder/postgres_repository.go` (`GetPath`, `IsDescendant`) haben keine
  eigene Tiefenbegrenzung auf DB-Ebene; der Service-Layer verhindert einen Zyklus ueber die
  normale API bereits vollstaendig (siehe gebaut-Zeile), ein DB-Test mit kuenstlich injiziertem
  Zyklus wurde nicht gebaut, da ausserhalb des Zielpakets dieser Unit. Falls es einen anderen
  Schreibpfad auf `document_folders.parent_id` gibt (Migration, Admin-Tool, direktes SQL), waere
  das ein Verfuegbarkeitsrisiko — nicht verifiziert, nur die Service-Guard-Kette gegengeprueft.
  (3) Coverage-Rest in `service.go`: `Upload` 88,0 %, `CreateVersion` 70,0 %, `Move`/`Copy`
  70-72 % — ueberwiegend `slog.Error`-Zweige bei simuliertem Repo-Fehler nach erfolgreichem
  MinIO-Schreiben, nicht weiter verfolgt (Best-Effort-Pfade, kein Bug-Verdacht).

## Iteration 34 — cov-gateway-document-wopi-comments-shares — done — 2026-08-26 05:47
- commit: 151aefd3
- gebaut: `backend/internal/gateway/route_document_wopi_comments_shares_test.go` — 62 Tests
  fuer die 19 im Scope genannten, zuvor ungetesteten Handler in `route_document.go`:
  HandleGetFolderPath, HandleInitializeUserSpace, HandleInitializeTeamSpace,
  HandleUploadFile, HandleListFileActivity, HandleListFileComments,
  HandleCreateFileComment, HandleUpdateFileComment, HandleDeleteFileComment,
  HandleGetFileVersionDownloadURL, HandleDeleteEntityLink, HandleUnshareEntity,
  HandleListShares, HandleTagFile, HandleUntagFile, HandleSearchFiles,
  HandleListVirtualFiles, HandleGenerateWOPIToken, HandleGetWOPIDiscovery — je
  ServiceUnavailable/InvalidJSON/Validierungsluecken/ReachesRPC, im selben
  Dummy-Registry-Muster wie die uebrigen Gateway-Coverage-Units (kein Fake
  DocumentServiceClient in diesem Paket). `HandleUploadFile` zusaetzlich mit echtem
  Multipart-Body (`multipartUploadBody`, neuer lokaler Helper mit `CreatePart` statt
  `CreateFormFile`, da Letzteres den Content-Type nicht steuerbar macht): fehlendes
  `folder_id`, fehlende Datei, verbotener MIME-Typ (inkl. `image/svg+xml`, explizit im
  Code-Kommentar ausgeschlossen), leere Datei, ueberschrittenes 50-MiB-Limit
  (`maxDocumentUploadBytes`), gueltiger Request bis zur RPC.
  Schwerpunkt-Fragen der Unit, mit Beleg beantwortet:
  (1) WOPI-Token — Laufzeit/Datei-/Tenant-Bindung/Verhalten nach Freigabe-Entzug: TTL
  fest 10 h (`wopi/token.go:11`), `file_id`+`tenant_id` werden als Claims eingebettet
  (`document_grpc.go:1397`) und bei jedem WOPI-Content-Handler gegengeprueft
  (`wopi/handler.go`: `claims.FileID` gegen URL-Parameter, `tenantID` aus Claims scopt
  `fileService.GetByID` — ein fremder Tenant scheitert an RLS, nicht an einem
  Go-Filter, gleiches Muster wie bereits fuer `internal/work/recording` in
  Iteration 24 verifiziert). Es gibt aber KEINE per-Datei widerrufbare Freigabe fuer
  WOPI-Bearbeitung — anders als External-Share-Links (die einen echten `RevokeShareLink`
  haben), ist WOPI-Schreibzugriff nur ueber die grobe tenant-weite
  "documents:write"-Berechtigung gegated (`route_document.go:187`), dieselbe Guard wie
  jede andere Schreibaktion im Modul. "Gilt der Token nach Entzug der Freigabe noch?"
  ist damit keine anwendbare Frage — es gibt nichts Feinkoerniges zu entziehen. Kein
  Fund, kein Fix-Unit.
  (2) `HandleSearchFiles`-Tenant-Filter: `document/search/postgres_repository.go` setzt
  KEINEN `tenant_id`-Filter in SQL, sondern verlaesst sich vollstaendig auf RLS —
  verifiziert: `document_files` traegt `CALL enable_tenant_rls(...)` seit Migration
  000122. Konsistent mit der bereits akzeptierten Architektur (ADR-006), kein Fund.
  (3) `HandleUploadFile`-Groessenbegrenzung/Typpruefung: bereits im Handler vorhanden
  (`maxDocumentUploadBytes` = 50 MiB, `allowedDocumentUploadMimeTypes`-Allowlist) — durch
  die neuen Tests jetzt belegt statt nur gelesen.
- gate: build ok (`./internal/gateway/... ./cmd/gateway/...`) | vet ok | lint ok
  (0 issues, `golangci-lint run --config .golangci.yml ./internal/gateway/...`) | test ok
  (komplettes Paket gruen, `DATABASE_URL` gesetzt, 0 Skips) | migration n.a. (keine
  Tabelle/Policy angefasst) | rls-smoke n.a. (reine Handler-Unit-Tests, kein neuer
  DB-Zugriff) | route-drift ok (`TestOpenAPIRouteDrift`: 836 registrierte gegen 838
  dokumentierte Pfade, PASS — keine neue Route)
- coverage: internal/gateway 65,4 % -> 66,3 % (`-coverprofile` mit/ohne die neue
  Testdatei gemessen). `route_document.go` (Funktions-Durchschnitt ueber
  `go tool cover -func`, da kein Datei-Summenwert existiert): 40,2 % -> 66,0 %.
- mutations-probe: in `createFileCommentRequest.Content` das `validate:"required"`
  entfernt (Backup vorher per `cp`). `TestHandleCreateFileComment_MissingContent`
  sofort rot (erwartete 400/validation_failed/Feld "content", bekam 503 vom
  Transportfehler, weil der leere Content jetzt gueltig war und bis zur RPC durchlief).
  Datei per `cp` zurueckgedreht, `git diff --stat` gegen `route_document.go` danach
  leer, `go test ./internal/gateway/...` wieder komplett gruen.
- verify vorgaenger: sauber (`409850d3` — `git show --stat` gegengeprueft: nur
  `postgres_repository_file_test.go`/`service_test.go` (neue Testdateien) plus
  BACKLOG.yml/JOURNAL.md im Diff, kein `.proto`, keine neue Route, kein
  `RequirePermission`, keine neue Tabelle, kein gRPC-Bypass)
- neue-units: keine
- offen: (1) `UnshareEntity` (`document_grpc.go:978`) holt anders als die
  Nachbar-Handler `ShareEntity`/`TagFile`/`DeleteEntityLink` KEIN `tenantID` aus dem
  Context und reicht keins an `shareService.Delete`/`ListByEntity` durch — verifiziert
  als stilistische Inkonsistenz, nicht als Sicherheitsluecke: `document_shares` traegt
  RLS seit Migration 000122 (`enable_tenant_rls`), die Isolation greift also trotzdem.
  Nicht als Fix-Unit angelegt, da kein tatsaechlicher Cross-Tenant-Zugriff moeglich ist.
  (2) `GenerateWOPIToken` (`document_grpc.go:1382`) prueft nie, ob die `file_id`
  tatsaechlich existiert, bevor ein Token ausgestellt wird — scheitert erst beim ersten
  WOPI-Zugriff (`CheckFileInfo`/`GetFile`). Niedrige Schwere (kein Datenzugriff, nur ein
  nutzloser Token), nicht als Fix-Unit angelegt. (3) Der bereits mehrfach dokumentierte
  Bestandsbefund gilt weiterhin: viele `id`-Pfadparameter in `route_document.go`
  erreichen die RPC-Schicht ohne lokale UUID-Validierung (`validateUUIDParam` fehlt),
  gleiche Klasse wie in Iteration 33 (`route_document_test.go`) dokumentiert — nicht neu,
  nicht hier gefixt.

## Iteration 35 — cov-gateway-calendar-lists-and-categories — done — 2026-08-26 05:57
- commit: 8091d15b
- gebaut: neue Testdatei `route_calendar_lists_categories_test.go` deckt die elf
  bislang ungetesteten Handler aus `route_calendar.go` ab: HandleListCalendars,
  HandleListBrowsableCalendars, HandleListCalendarMembers,
  HandleCreateEventCategory, HandleUpdateCalendarPreferences,
  HandleGetCalendarPreferences, HandleDeleteEventCategory,
  HandleListEventCategories, HandleListEventAttendees, HandleListHolidays,
  HandleListTaskDeadlinesInRange — jeweils ServiceUnavailable-, Validierungs-
  und ReachesRPC-Pfad, dazu Query-Parameter-Varianten (include_hidden, search,
  subdivision_code+Range, alle Preferences-Felder).
  Schwerpunkt-Fragen der Unit, mit Beleg beantwortet (Code gelesen, nicht per
  neuem Test erreichbar, da kein fake CalendarServiceClient in diesem Paket):
  (1) `HandleListBrowsableCalendars` — welche Information ueber fremde
  Kalender wird sichtbar? `internal/work/calendar/postgres_repository.go
  ListBrowsable` selektiert ausschliesslich Kalender-Metadaten (id, tenant_id,
  name, description, calendar_type, color, owner_id, is_default, timezone,
  Zeitstempel) — keine Termine, keine Termintitel. Name/Beschreibung eines
  geteilten Kalenders zu zeigen ist der Zweck des "shared"-Discovery-Views,
  kein Datenschutzbefund.
  (2) `HandleListEventAttendees` — sieht ein Teilnehmer die Liste aller
  anderen? `calendar_grpc.go ListEventAttendees` -> `event/service.go
  ListAttendees` prueft nur `repo.GetByID(eventID, tenantID)` (tenant-scoped
  Existenz), NIE ob der Aufrufer selbst Teilnehmer, Ersteller oder
  Kalender-Mitglied ist. Jeder Tenant-User mit der groben, tenant-weiten
  Berechtigung "calendars:read" kann die Teilnehmerliste JEDES Events im
  Tenant abrufen. Kein neuer Befund dieser Unit: `GetEvent`
  (calendar_grpc.go:461, `eventService.Get`) hat exakt dieselbe Form — wer
  schon Event-Details lesen kann, kam an dieselbe Information bereits vorher.
  `SetReminders` (event/service.go:504) hat dagegen fuer Nicht-Ersteller ein
  `requireCalendarEditPermission` — der Lesepfad hat kein Aequivalent.
  Bestehende, durchgaengige grobe Berechtigungsarchitektur (calendars:read
  tenant-weit, keine Pro-Kalender-ACL beim Lesen), keine neue Abweichung durch
  diesen Handler — keine Fix-Unit, da Architekturentscheidung und nicht durch
  eine Coverage-Unit aenderbar; im Journal festgehalten, weil die Backlog-Scope
  die Frage ausdruecklich stellte.
  (3) `HandleListTaskDeadlinesInRange` — reichen fremde Aufgaben durch?
  `event/postgres_repository.go ListTaskDeadlinesInRange` filtert per SQL nur
  auf `assignee_id = userID OR project_members.user_id = userID` (eigene
  Aufgaben/Projekte), OHNE expliziten `tenant_id`-Filter im SQL — aber die
  Tabelle `tasks` traegt RLS seit Migration 000120
  (`CALL enable_tenant_rls('tasks')`), konsistent mit der bereits akzeptierten
  ADR-006-Architektur (RLS statt App-Filter, gleiches Muster wie fuer die
  Dokumentensuche in Iteration 34 verifiziert). Kein Fund, keine Fix-Unit.
- gate: build ok (`./internal/gateway/... ./cmd/gateway/...`) | vet ok | lint ok
  (0 issues, `golangci-lint run --config .golangci.yml ./internal/gateway/...`)
  | test ok (komplettes Paket gruen, `DATABASE_URL` gesetzt, 0 Skips) |
  migration n.a. (keine Tabelle/Policy angefasst) | rls-smoke n.a. (reine
  Handler-Unit-Tests, kein neuer DB-Zugriff) | route-drift ok
  (`TestOpenAPIRouteDrift`: 836 registrierte gegen 838 dokumentierte Pfade,
  PASS — keine neue Route)
- coverage: internal/gateway 66,3 % -> 67,0 % (`-coverprofile` vor/nach der
  neuen Testdatei gemessen). Die elf Ziel-Handler in `route_calendar.go`
  (`go tool cover -func`): alle vorher 0,0 %, nachher 90,0-96,6 % (Uebersicht:
  HandleListCalendars 90,9 %, HandleListBrowsableCalendars 92,3 %,
  HandleListCalendarMembers 90,0 %, HandleCreateEventCategory 92,3 %,
  HandleUpdateCalendarPreferences 96,2 %, HandleGetCalendarPreferences 90,0 %,
  HandleDeleteEventCategory 91,7 %, HandleListEventCategories 90,0 %,
  HandleListEventAttendees 91,7 %, HandleListHolidays 96,6 %,
  HandleListTaskDeadlinesInRange 95,8 %).
- mutations-probe: in `createEventCategoryRequest.Name` das
  `validate:"required"` entfernt (Backup vorher per `cp`).
  `TestHandleCreateEventCategory_MissingName` sofort rot (erwartete
  400/validation_failed/Feld "name", bekam 503 vom Transportfehler, weil der
  leere Name jetzt gueltig war und bis zur RPC durchlief). Datei per `cp`
  zurueckgedreht, `git diff --stat` gegen `route_calendar.go` danach leer,
  `go test ./internal/gateway/...` wieder komplett gruen.
- verify vorgaenger: sauber (`151aefd3` — `git show --stat` gegengeprueft: nur
  eine neue Testdatei `route_document_wopi_comments_shares_test.go` plus
  BACKLOG.yml/JOURNAL.md im Diff, kein `.proto`, keine neue Route, kein
  `RequirePermission`, keine neue Tabelle, kein gRPC-Bypass)
- neue-units: keine
- offen: (1) `HandleListCalendarMembers` liest "id" wie mehrere
  Membership-Handler in `route_calendar_membership_test.go` per rohem
  `chi.URLParam` ohne `validateUUIDParam` — bereits katalogisierte Bestandslage
  (Iteration 6, `fix-gateway-id-validation-consistency`), nicht neu, nicht hier
  gefixt. (2) Die unter Schwerpunkt-Frage (2) dokumentierte grobe
  `calendars:read`-Berechtigung (kein Pro-Kalender-ACL beim Lesen von Events/
  Attendees) ist eine Architekturfrage fuer Luke, keine Coverage-Aenderung —
  falls das je geschaerft werden soll, gehoert das als eigene Design-Unit ins
  Backlog, nicht in diesen Lauf.

## Iteration 36 — cov-gateway-calendar-resources-and-reminders — done — 2026-08-26 06:06
- commit: 22aad711cb74bb95f7eda07ead06bbd778d6a03e
- gebaut: `route_calendar_resources_reminders_test.go` (Gateway, 25 Tests fuer die sieben
  Handler HandleListResources, HandleGetResource, HandleDeleteResource,
  HandleListResourceAvailability, HandleSetEventReminders, HandleListEventReminders,
  HandleGenerateJoinToken) + `postgres_repository_db_test.go` (internal/work/resource, 7
  DB-Testfunktionen gegen echtes SQL: List mit allen Filterkombinationen inkl. Tenant-Scope,
  SetTags-Replace, CreateBooking Erfolg+Konflikt, CancelBooking inkl. Fremd-Tenant-Versuch,
  ListBookings/ListBookingsByEvent Overlap-Fenster, FindAvailableResources/FindAlternatives
  inkl. Ausschluss gebuchter/inaktiver Ressourcen).
  Schwerpunkt (1) Race-Bedingung Verfuegbarkeit/Buchung: KEINE Luecke gefunden.
  `resource.Service.Book` prueft Verfuegbarkeit nicht selbst per Read-vor-Write, sondern
  verlaesst sich auf die `resource_bookings`-EXCLUDE-USING-GIST-Constraint
  (postgres_repository.go:213-217, pg-Fehler 23P01 -> ErrBookingConflict) — atomar auf
  DB-Ebene, exakt das Muster, das `harden-quote-conversion-unique-index` als sicher
  etabliert hat. Durch Mutations-Probe UND einen echten DB-Test
  (TestPostgresCreateBooking_SuccessAndConflict) belegt. Keine Fix-Unit noetig.
  Schwerpunkt (2) HandleGenerateJoinToken: ECHTER PRODUKTIONSFEHLER gefunden, siehe
  neue-units. Weder Handler noch gRPC-Server pruefen, ob event_id zu einem existierenden,
  zum aufrufenden Tenant gehoerenden Event gehoert — jeder User mit "calendars:write" kann
  ein gueltiges 24h-LiveKit-Join-Token fuer eine beliebige (auch fremd-tenante) UUID
  erzeugen. Terminabsage aendert nichts, weil das Event nie geladen wird.
- gate: build ok (`./internal/work/resource/... ./internal/gateway/... ./cmd/gateway/...`)
  | vet ok (beide Pakete) | lint ok (0 issues, beide Pakete, golangci-lint run --config
  .golangci.yml) | gofmt: eine Formatierungsabweichung in postgres_repository_db_test.go
  (intPtr/strPtr-Ausrichtung) gefunden und mit `gofmt -w` behoben, danach `gofmt -l` leer
  | test ok (beide Pakete komplett gruen, DATABASE_URL gesetzt, 0 Skips) | migration n.a.
  (keine Tabelle/Policy angefasst) | rls-smoke ok (ueber die echten Repo-Methoden: List
  mit fremdem TenantID-Filter liefert 0 Zeilen trotz physisch vorhandener Fremd-Tenant-Row;
  CancelBooking mit fremder tenantID liefert ErrBookingNotFound statt den Booking zu
  canceln — beides in postgres_repository_db_test.go) | route-drift ok
  (`TestOpenAPIRouteDrift`: 836 registrierte gegen 838 dokumentierte Pfade, PASS — keine
  neue Route)
- coverage: internal/gateway 67,0 % -> 67,5 % (die sieben Ziel-Handler laut `go tool cover
  -func`: HandleSetEventReminders 93,3 %, HandleListEventReminders 91,7 %,
  HandleListResources 95,8 %, HandleGetResource 91,7 %, HandleDeleteResource 91,7 %,
  HandleListResourceAvailability 96,2 %, HandleGenerateJoinToken 93,8 %, alle vorher 0,0 %).
  internal/work/resource 42,6 % -> 85,9 % (postgres_repository.go je Methode: List 0,0 % ->
  95,0 %, SetTags 0,0 % -> 76,9 %, CreateBooking 0,0 % -> 87,5 %, CancelBooking 0,0 % ->
  85,7 %, ListBookings 0,0 % -> 83,3 %, ListBookingsByEvent 0,0 % -> 83,3 %, GetBooking
  0,0 % -> 75,0 %, FindAvailableResources 0,0 % -> 73,0 %, FindAlternatives 0,0 % -> 83,3 %,
  scanBookings 0,0 % -> 85,7 %).
- mutations-probe: in `CreateBooking` (postgres_repository.go:215) den pg-Fehlercode-Vergleich
  von `"23P01"` auf `"00000"` geaendert (Backup vorher per `cp`).
  `TestPostgresCreateBooking_SuccessAndConflict` sofort rot (erwartete ErrBookingConflict,
  bekam den rohen pg-Fehler "conflicting key value violates exclusion constraint... SQLSTATE
  23P01" durchgereicht). Datei per `cp` zurueckgedreht, `git diff --stat` gegen
  postgres_repository.go danach leer, `go test ./internal/work/resource/...` wieder
  komplett gruen.
- verify vorgaenger: sauber (`8091d15b` — `git show --stat` gegengeprueft: nur eine neue
  Testdatei `route_calendar_lists_categories_test.go` plus BACKLOG.yml/JOURNAL.md im Diff,
  kein `.proto`, keine neue Route, kein `RequirePermission`, keine neue Tabelle, kein
  gRPC-Bypass)
- neue-units: `fix-generatejointoken-missing-event-tenant-check` (opus) — der unter
  Schwerpunkt (2) beschriebene Produktionsfehler. Nicht in dieser Coverage-Unit gefixt, weil
  eine Coverage-Unit kein Verhalten aendern darf; ans Backlog-Ende angehaengt mit vollem
  scope/sources/notes/done_when.
- offen: (1) Die neue Fix-Unit `fix-generatejointoken-missing-event-tenant-check` ist ein
  echtes Sicherheitsproblem (Cross-Tenant-Zugriff auf LiveKit-Meetings) — sollte in einer der
  naechsten Iterationen priorisiert werden, nicht bis zum Blockende liegen bleiben. (2) Wie
  in Iteration 35 vermerkt bleibt die grobe `calendars:read`/`calendars:write`-Berechtigung
  (keine Pro-Event-ACL) eine Architekturfrage fuer Luke — der neue Fix behebt nur das
  fehlende Tenant-Scoping, nicht die fehlende Teilnehmer/Ersteller-Pruefung.

## Iteration 37 — cov-berichte-repository-and-grpc — done — 2026-08-26 06:18
- commit: c92b0ec7
- gebaut: `postgres_repository_db_test.go` (internal/berichte, 10 DB-Testfunktionen gegen
  echtes SQL: ListDefinitions/ListSchedules/ListDocuments mit allen Filtern + Pagination +
  Sortierung + Tenant-Scope, Get/Update/Delete-Tenant-Scoping fuer Definitions und
  Schedules, Cache-CRUD inkl. ON-CONFLICT-Upsert und DeleteExpiredCacheEntries-Sweep,
  ListDueSchedules, UpdateScheduleLastRun, InsertRun) + Erweiterung von
  `berichte_grpc_test.go` (internal/server) um die zuvor 0 % gedeckten Document- und
  Share-Token-RPCs (CreateDocument, GetDocument, UpdateDocument, DeleteDocument,
  ListDocuments inkl. Leer-Ergebnis-als-`[]`-Test, CreateShareToken, ListShareTokens,
  RevokeShareToken, GetSharedDocument) plus documentToProto/reportShareTokenToProto.
  Bug-Hypothese laut Scope (weitere blinde Waehrungs-/Einheiten-Aggregationen): das
  Repository (`postgres_repository.go`) enthaelt KEINE einzige Aggregation (kein SUM/AVG/
  MAX) — reine CRUD-Methoden fuer report_definitions/report_cache/report_schedules/
  report_runs/report_documents/report_share_tokens. Die tatsaechliche Aggregation liegt in
  `internal/berichte/executor/executor.go` (unit-getestet, kein DB-Bezug) und
  `internal/berichte/downstream/kpi_postgres.go` (die einzige PRODUKTIV VERDRAHTETE
  Aggregation, `cmd/berichte/main.go:59` — Finance/CRMReports/Helpdesk/Inventar/DatevBridge
  sind dort nirgends verdrahtet, bleiben also nil und `emptyResult`). `kpi_postgres.go` ist
  bereits vollstaendig DB-getestet inkl. Waehrungs-Scoping (`fix-dashboard-metrics-blind-
  currency-sum`-Muster) und Leer-Zeitraum-Verhalten — keine weitere blinde Aggregation
  gefunden, siehe Grep-Beleg unten. `executor.go` dokumentiert selbst (Kommentar Zeile
  547-554), dass `revenueByMonth`/`invoicesOpen`/`pipeline` denselben Bug wie die laengst
  gefixte Dashboard-KPI-Aggregation tragen wuerden, aber aktuell unerreichbar sind (nil
  Downstreams) — verifiziert gegen `cmd/berichte/main.go`, kein Neufund, bereits
  dokumentiert.
  Schwerpunkt (2), echter Fund: `PostgresRepository.DeleteExpiredCacheEntries` ist
  vollstaendig implementiert und jetzt DB-getestet, wird aber im gesamten Code NIRGENDS
  aufgerufen (`grep -rn DeleteExpiredCacheEntries` traf nur die Implementierung, das
  Interface und Test-Stubs) — `report_cache` waechst unbegrenzt, da abgelaufene Zeilen nie
  geloescht werden (sie werden nur nie zurueckgeliefert, `GetCacheEntry` + service.go
  pruefen `expires_at` selbst). Kein Korrektheits-Bug, aber ein Verhaltens-Fix (periodischer
  Sweep noetig) — gehoert nicht in eine Coverage-Unit, daher als eigene Fix-Unit angelegt
  (siehe neue-units).
  Ob `berichte_grpc.go` Aggregation nachbaut, die in den Service gehoert: nein — jeder RPC
  ist Parse/Call-Service/Respond, keine einzige Summierung oder Gruppierung im gRPC-Server
  selbst (verifiziert durch vollstaendiges Lesen der Datei).
  Leerer Zeitraum: `ListDefinitions`/`ListSchedules`/`ListDocuments` liefern bei keinem
  Treffer `nil`-Slice + `total=0` (kein Crash, kein Fehler) — durch die neuen DB-Tests fuer
  die "other tenant"-Faelle mitbelegt; die gRPC-Schicht wrapped das korrekt in eine leere
  Proto-Liste (`ListDocuments_Empty`-Test belegt `resp.Documents != nil` als `[]`, nicht
  `null`).
- gate: build ok (`./internal/berichte/... ./internal/server/... ./cmd/berichte/...`)
  | vet ok (beide Pakete) | lint ok (0 issues, golangci-lint run --config .golangci.yml)
  | gofmt: eine Formatierungsabweichung in berichte_grpc_test.go gefunden und mit
  `gofmt -w` behoben, danach `gofmt -l` leer | test ok (`./internal/berichte/...` alle
  Unterpakete gruen inkl. downstream/executor/scheduler/export/delivery,
  `./internal/server/` komplett gruen, DATABASE_URL gesetzt, 0 Skips)
  | migration n.a. (keine Tabelle/Policy angefasst) | rls-smoke ok (ueber die echten
  Repo-Methoden unter `kmuhub_app`: ListDefinitions/ListSchedules/ListDocuments liefern
  fuer den fremden Tenant total=0 trotz physisch vorhandener Fremd-Tenant-Zeile;
  Get/Update/Delete unter Fremd-Tenant-Kontext liefern durchgaengig ErrDefinitionNotFound/
  ErrScheduleNotFound statt eine Mutation zuzulassen) | route-drift ok
  (`TestOpenAPIRouteDrift`: 836 registrierte gegen 838 dokumentierte Pfade, PASS — keine
  neue Route)
- coverage: internal/berichte 62,0 % -> 86,3 % (postgres_repository.go je Methode:
  UpdateDefinition/DeleteDefinition 0,0->83,3 %, GetDefinition 0,0->100 %,
  ListDefinitions 0,0->90,7 %, GetCacheEntry 0,0->85,7 %, UpsertCacheEntry/InvalidateCache/
  DeleteExpiredCacheEntries 0,0->75,0 %, UpdateSchedule/DeleteSchedule 0,0->83,3 %,
  ListSchedules 0,0->89,3 %, ListDueSchedules 0,0->83,3 %, UpdateScheduleLastRun/InsertRun
  0,0->75,0 %, ListDocuments 0,0->88,2 %). internal/server 70,9 % -> 71,3 %
  (berichte_grpc.go: CreateDocument 0,0->84,6 %, GetDocument 0,0->100 %, UpdateDocument
  0,0->85,7 %, DeleteDocument 0,0->83,3 %, ListDocuments 0,0->90,0 %, CreateShareToken
  0,0->86,7 %, ListShareTokens 0,0->90,0 %, RevokeShareToken 0,0->88,9 %,
  GetSharedDocument/reportShareTokenToProto/documentToProto 0,0->100 %).
- mutations-probe: zwei unabhaengige Mutationen in postgres_repository.go (Backup vorher
  per `cp`, danach zurueckgespielt und `git diff --stat` gegen die Datei leer): (1) Tenant-
  Filter aus `DeleteDefinition`s WHERE-Klausel entfernt — blieb GRUEN, weil RLS unter
  `kmuhub_app` (NOSUPERUSER NOBYPASSRLS) den fehlenden WHERE-Filter selbst abfaengt; als
  Beleg fuer die Wirksamkeit von RLS notiert, aber KEIN Beweis fuer den eigenen Test.
  (2) `ListDefinitions`-Sortierrichtung invertiert (`if filter.SortDesc` statt
  `if !filter.SortDesc`) — `TestPostgresListDefinitions_...` sofort ROT ("sort asc by name:
  unexpected order [Gamma Tickets Beta Pipeline Alpha Umsatz]"), zurueckgedreht, danach
  wieder gruen. Zeigt: die Sortier-/Filter-/Pagination-Assertions sind wirksam, RLS allein
  haette sie nicht gefangen.
- verify vorgaenger: sauber (`22aad711` — `git show --stat` gegengeprueft: nur zwei neue
  Testdateien (route_calendar_resources_reminders_test.go,
  internal/work/resource/postgres_repository_db_test.go) plus BACKLOG.yml/JOURNAL.md im
  Diff, kein `.proto`, keine neue Route, kein `RequirePermission`, keine neue Tabelle, kein
  gRPC-Bypass)
- neue-units: `fix-berichte-report-cache-never-purged` (sonnet) — der unter Schwerpunkt (2)
  beschriebene Fund (DeleteExpiredCacheEntries wird nie aufgerufen, report_cache waechst
  unbegrenzt). Nicht in dieser Coverage-Unit gefixt, weil eine Coverage-Unit kein Verhalten
  aendern darf (neuer periodischer Sweep waere ein Verhaltens-Fix); ans Backlog-Ende
  angehaengt mit vollem scope/sources/notes/done_when.
- offen: (1) `fix-generatejointoken-missing-event-tenant-check` (Iteration 36, echte
  Sicherheitsluecke) steht weiterhin unbearbeitet am Backlog-Kopf-nahen Bereich — sollte
  vor den uebrigen Coverage-Units drankommen, der Treiber zieht aber strikt nach
  Dateireihenfolge. (2) `fix-berichte-report-cache-never-purged` ist ein reines
  Betriebsproblem (unbegrenztes Tabellenwachstum), kein akuter Bug — kann regulaer in
  Dateireihenfolge einsortiert bleiben.

## Iteration 38 — cov-work-customfield-and-presence-zero-coverage — done — 2026-08-26 06:30
- commit: 221e9347
- gebaut: DB-Tests fuer `internal/work/customfield/postgres_repository.go` (Create/GetByID/
  List/Update/Delete, Tenant-Scoping ueber RLS-Smoke fuer jede Methode, (tenant_id, name)
  Unique-Constraint inkl. "gleicher Name, anderer Tenant funktioniert", Sortierung
  position ASC/name ASC, leere Liste, sowie der in scope verlangte Fall "Definition mit
  vorhandenen Task-Werten loeschen"). Redis-Tests fuer
  `internal/work/presence/redis_store.go` mit miniredis (SetPresence/GetPresence/
  RemovePresence/UpdateLastActivity/GetBulkPresence, TTL-Pruefung per `mr.TTL`,
  korrupter JSON-Wert, leere/fehlende Keys).
  Scope-Fragen beantwortet: (1) Delete mit vorhandenen `task_custom_field_values`: kein Bug,
  `ON DELETE CASCADE` seit Migration 000320 — Testfall
  `TestPostgresDelete_WithExistingTaskValues_CascadesValues` belegt das. (2) DSAR:
  `internal/security/gdpr/dsar_search.go` deckt Task-Custom-Field-Werte NICHT ab —
  `customFieldsModule` (Zeile 1771) ist ausschliesslich fuer `contact_custom_field_values`
  (CRM-Kontakte); `tasksModule` (Zeile 619, benutzerbezogener DSAR-Pfad) liefert Titel/Status/
  Rolle, aber keine Custom-Field-Werte der Aufgabe — echte Luecke, als Unit angelegt (siehe
  neue-units). (3) Redis-Testaufbau: `miniredis` existiert bereits im Repo (u. a.
  `internal/dialer/redis_agent_store_io_test.go`), kein neuer Aufbau noetig. (4) TTL/Wachstum:
  `presenceTTL = 90 * time.Second` auf jedem `SetPresence`/`UpdateLastActivity` — waechst
  nicht unbegrenzt, per Test gegen `mr.TTL` gepinnt. (5) Tenant im Redis-Schluessel: NEIN —
  `presenceKey(userID) = "presence:" + userID` (redis_store.go:51-53), userID ist eine
  global-eindeutige UUID, daher keine Kollisionsgefahr durch Zufall — aber der Schluessel
  selbst beweist keine Tenant-Zugehoerigkeit. Der wertvolle Fund liegt eine Schicht hoeher:
  `WebSocketHub.handlePresenceSubscribe` (internal/server/websocket.go:1116) nimmt eine
  `user_ids`-Liste direkt vom Client entgegen und traegt sie ungeprueft in
  `presenceSubscribers` ein — kein Tenant-Check. Ein User kann sich damit auf die Presence
  eines Users aus einem FREMDEN Tenant abonnieren, wenn er dessen UUID kennt — echtes
  Cross-Tenant-Informationsleck. Nicht in dieser Coverage-Unit gefixt (Verhaltensaenderung),
  als Fix-Unit angelegt (siehe neue-units).
- gate: build ok (`./internal/work/...`) | vet ok (`./internal/work/...`) | lint ok
  (0 issues, golangci-lint run --config .golangci.yml ./internal/work/...) | test ok
  (`./internal/work/customfield/...` und `./internal/work/presence/...` einzeln UND
  zusammen gruen, DATABASE_URL gesetzt, 65 Tests in den beiden Paketen, 0 Skips, 0 Fails;
  `./internal/work/...` gesamt zeigte beim ersten -count=1-Lauf drei Fails durch
  Connection-Pool-Erschoepfung ("too many clients already", "remaining connection slots
  reserved for SUPERUSER") in `resource`/`timeentry` — mit `-p 2` reproduzierbar gruen,
  also Infra-Artefakt der hohen Parallelitaet ueber alle work-Pakete gleichzeitig, keine
  Regression durch diese Aenderung) | migration n.a. (keine Tabelle/Policy angefasst) |
  rls-smoke ok (ueber die echten Repo-Methoden unter `kmuhub_app`: GetByID/Update/Delete
  liefern fuer fremden Tenant durchgaengig ErrNotFound trotz physisch vorhandener Zeile,
  List liefert fuer fremden Tenant nur eigene 3 von 4 Definitionen) | route-drift n.a.
  (keine Route angefasst, `go test ./internal/gateway/` daher nicht Pflicht)
- coverage: internal/work/customfield 0,0 % (postgres_repository.go, 71 Statements)
  -> 82,6 % Paket-gesamt (Create 83,3 %, GetByID 88,9 %, List 84,6 %, Update 85,7 %,
  Delete 83,3 %, marshalOptions 100 %, unmarshalOptions 66,7 %, isDuplicateError 71,4 %).
  internal/work/presence 0,0 % (redis_store.go, 68 Statements) -> 81,2 % Paket-gesamt
  (SetPresence 75 %, GetPresence 83,3 %, GetBulkPresence 70,4 %, RemovePresence 66,7 %,
  UpdateLastActivity 80 %, parsePresenceData 85,7 %).
- mutations-probe: zwei unabhaengige Mutationen (Backup vorher per `cp`, danach
  zurueckgespielt, `git status --short` auf beiden Dateien leer): (1) in
  `customfield/postgres_repository.go` List-Query von `ORDER BY position ASC, name ASC`
  auf `ORDER BY position DESC, name ASC` geaendert — `TestPostgresList_TenantScoped_
  OrderedByPositionThenName` sofort ROT ("expected order [Prefix Alpha Zeta], got
  [Alpha Zeta Prefix]"). (2) in `presence/redis_store.go` `SetPresence`s TTL-Parameter
  von `presenceTTL` auf `0` geaendert — `TestRedisStore_SetPresence_SetsTTL` sofort ROT
  ("expected: 1m30s, actual: 0s"). Beide zurueckgedreht, Pakete danach wieder gruen.
- verify vorgaenger: sauber (`c92b0ec7` — `git show --stat` gegengeprueft: nur zwei neue
  Testdateien plus BACKLOG.yml/JOURNAL.md im Diff, kein `.proto`, keine neue Route, kein
  `RequirePermission`, keine neue Tabelle, kein gRPC-Bypass)
- neue-units: `fix-websocket-presence-subscribe-missing-tenant-check` (sonnet) — Presence-
  Subscribe ohne Tenant-Check, echtes Cross-Tenant-Info-Leck (siehe gebaut). `fix-dsar-
  tasks-module-missing-custom-field-values` (sonnet) — Task-Custom-Field-Werte fehlen im
  benutzerbezogenen DSAR-Export (siehe gebaut). Beide mit vollem scope/sources/notes/
  done_when ans Backlog-Ende angehaengt.
- offen: (1) `fix-generatejointoken-missing-event-tenant-check` (Iteration 36) steht
  weiterhin unbearbeitet, der Treiber zieht strikt nach Dateireihenfolge. (2)
  `fix-berichte-report-cache-never-purged` (Iteration 37) ebenso. (3) Die beiden neuen
  Units aus dieser Iteration sind reine Verhaltens-Fixes ohne Deploy-Hazard (kein neuer
  RequirePermission-Guard, keine neue Config-Assertion) und koennen regulaer in
  Dateireihenfolge einsortiert bleiben. (4) Connection-Pool-Limit fuer `kmuhub_app`/
  Superuser-reservierte Slots: `go test ./internal/work/...` als Gesamtlauf mit voller
  Parallelitaet ist auf dieser lokalen DB nicht zuverlaessig gruen (siehe gate) — kein
  Fund dieser Iteration, aber falls das oefter auftritt, waere `max_connections` oder die
  Pool-Groesse pro Testpaket ein Thema fuer Luke.

## Iteration 39 — cov-work-project-repository-real-sql — done — 2026-08-26 06:41
- commit: 34d3eba3
- gebaut: Real-SQL-DB-Tests fuer `internal/work/project/postgres_repository.go` (7 neue
  Tests: List admin/member/archived-Verzweigung, GetProjectKey+KeyExists inkl. der
  gewollten Archiv-Freigabe des Keys, volle Member-Management-Lifecycle, AddMember gegen
  ein fremdes projectID als Verteidigung-in-der-Tiefe, SaveAsTemplate+GetForTemplate inkl.
  Cross-Tenant-Fall, UserPreference-Upsert gegen die echte Tabelle, Delete-Kaskade
  Projekt->Task->TimeEntry) und fuer `internal/work/timeentry/postgres_repository.go`
  (5 neue Tests: Doppel-Start ueber Service+Repo — beweist die "nur ein laufender Timer"
  Invariante real in SQL, Stop-ohne-laufenden-Timer auf Repo- UND Service-Ebene,
  GetActiveTimer tenant-gescoped, ListByTask/ListByUser Pagination-Clamping+Tenant-Scope,
  GetTaskTimeSummary inkl. laufendem Timer per NOW() und Tenant-Scope). Dazu acht neue
  Gateway-Handler-Tests fuer route_work_time.go (HandleStartTimer, HandleStopTimer,
  HandleGetActiveTimer, HandleAddManualTimeEntry, HandleUpdateTimeEntry,
  HandleGetTaskTimeSummary, HandleListProjectTimeEntries, HandleListProjectTeamUtilization)
  — Validierungsfehler, ServiceUnavailable und Reaches-RPC-Pfade, inkl. Beleg, dass
  billed=true den gRPC-Client trotzdem holt (nur die RPC selbst wird uebersprungen).
  Kein Produktionsbug gefunden: AddMember's Tenant-Subquery ist durch RLS abgesichert
  (Insert schlaegt mit NOT-NULL-Verletzung fehl statt zu leaken), SaveAsTemplate gegen
  fremde sourceID kopiert nichts (INSERT...SELECT matched 0 Zeilen unter RLS). Projekt ->
  Task -> TimeEntry Cascade-Delete ist beabsichtigtes Verhalten (Migrationen 000024/
  000025/000030), kein Datenleck.
- gate: build ok (`./internal/work/... ./internal/gateway/... ./cmd/work/... ./cmd/gateway/...`
  mit `-p 2`) | vet ok | lint ok (0 issues, golangci-lint ./internal/work/...
  ./internal/gateway/...) | test ok (DATABASE_URL gesetzt, project 7/7 neue Tests gruen,
  timeentry 5/5 neue Tests gruen, gateway 18 neue Handler-Tests gruen nach zwei
  Korrekturen — Feldnamen in assertValidationError mussten json-Keys sein
  (`started_at`/`duration_seconds`, nicht `StartedAt`/`DurationSeconds`), und der
  billed=true-Test brauchte `registryWithService` statt `emptyRegistry`, weil
  HandleListProjectTimeEntries den gRPC-Client vor der billed-Weiche holt; komplettes
  `./internal/work/...` mit `-p 2` diesmal ohne Connection-Pool-Erschoepfung durchgelaufen)
  | migration n.a. (keine Tabelle/Policy angefasst) | rls-smoke ok (durchgaengig ueber
  die echten Repo-Methoden: fremder Tenant liefert 0 Zeilen/nil/ErrNotFound in List,
  GetProjectKey, KeyExists, GetActiveTimer, ListByTask/ListByUser, GetTaskTimeSummary;
  AddMember gegen fremdes projectID schlaegt fehl statt zu leaken) | route-drift ok
  (`TestOpenAPIRouteDrift`: 836 registrierte Routen gegen 838 dokumentierte Pfade, keine
  neue Route in dieser Unit)
- coverage: internal/work/project (Datei postgres_repository.go, Referenzwert 19,4 %,
  100 von 124 Statements ungedeckt) -> Paket-gesamt 78,2 %, Datei fast durchgaengig
  71-100 % je Funktion (schwaechste: GetMember 71,4 %, Archive 66,7 % — beide durch
  bereits vorhandene tenant_write_test.go/service_test.go Pfade mitabgedeckt, nicht
  Luecken dieser Unit). internal/work/timeentry (Datei postgres_repository.go,
  Referenzwert 38,8 %, 71 von 116 Statements ungedeckt) -> Paket-gesamt 82,8 %, alle
  13 Funktionen zwischen 62,5 % und 100 %. internal/gateway/route_work_time.go: alle
  acht benannten Handler jetzt 73-96 % (vorher nur die zwei reinen Wire-Shape-Helfer
  getestet, 0 % auf den Handlern selbst).
- mutations-probe: drei Mutationen (Backup vorher per `cp`, danach zurueckgespielt,
  `git status --short` auf beiden Dateien am Ende leer): (1) in
  `project/postgres_repository.go` KeyExists' `AND archived_at IS NULL` entfernt ->
  `TestGetProjectKey_And_KeyExists_TenantScopedAndArchiveAware` sofort ROT ("KeyExists
  must return false once the holder is archived"). (2) in
  `timeentry/postgres_repository.go` StopActiveTimer's `AND ended_at IS NULL` entfernt ->
  KEIN Test wurde rot (Lehre: die vorhandenen Szenarien haben nie mehr als einen
  Zeiteintrag pro User gleichzeitig, daher deckt kein Test diese Zeile scharf — als
  Erkenntnis hier vermerkt, nicht als neue Unit, da reine Testschaerfe-Frage ohne
  Produktionsrisiko: das WHERE bleibt durch RLS UND durch die Service-Invariante
  "immer nur ein offener Timer" in der Praxis unkritisch). (3) Ersatzmutation in
  `ListByTask`: Page-Clamp `page = 1` auf `page = 2` geaendert ->
  `TestListByTask_And_ListByUser_PaginationAndTenantScoping` sofort ROT ("total=3 len=0,
  want 3, 3"). Zusaetzlich AM SELBEN Ort separat probiert: `GetTaskTimeSummary`s
  `AND tenant_id = $2` durch eine Tautologie ersetzt -> KEIN Test wurde rot, weil RLS
  die fremde Zeile ohnehin nicht sichtbar macht (Verteidigung-in-der-Tiefe faengt den
  Bug ab, bevor der Test ihn sehen koennte) — beide Nicht-Treffer sind Belege dafuer,
  dass RLS als zweite Schicht greift, nicht ein Loch in der Testsuite.
- verify vorgaenger: sauber (`221e9347` — nur zwei neue Testdateien
  (`customfield/postgres_repository_db_test.go`, `presence/redis_store_test.go`) plus
  BACKLOG.yml/JOURNAL.md im Diff, kein `.proto`, keine neue Route, kein
  `RequirePermission`, keine neue Tabelle, kein gRPC-Bypass, kein Stub-Marker)
- neue-units: keine (kein Fund, der nicht selbst behoben werden durfte oder auesserhalb
  des Scopes lag)
- offen: (1) `fix-generatejointoken-missing-event-tenant-check` (seit Iteration 36) und
  `fix-berichte-report-cache-never-purged` (seit Iteration 37) stehen weiterhin weiter
  hinten im File und wurden entsprechend Dateireihenfolge korrekt uebersprungen — sie
  laufen erst dran, wenn alle frueher im File stehenden `todo`-Units mit erfuellten
  `deps` abgearbeitet sind. (2) Projekt- vs. HR-Zeiterfassung: keine Doppel-Buchung —
  `internal/work/timeentry` schreibt in `time_entries` (Projekt-/Task-Stunden fuer
  "Stunden -> Rechnung"), `internal/biz/hr/timetracking` schreibt in
  `hr_work_time_entries`/`hr_break_entries` (Kommen/Gehen fuer Lohnabrechnung) — zwei
  getrennte Tabellen, zwei getrennte fachliche Zwecke, keine gemeinsame Rechnung. (3)
  `go test ./internal/work/...` lief in dieser Iteration mit `-p 2` durchgaengig gruen
  (kein Wiederauftreten des Connection-Pool-Problems aus Iteration 38).

## Iteration 40 — cov-vermietung-repository-lowest-coverage-in-backend — done — 2026-08-26 06:55
- commit: a6d6a187
- gebaut: `internal/vermietung/postgres_repository_db_test.go` (neu, 17 Testfunktionen)
  deckt jede Repository-Methode gegen echtes SQL ab (Objects/Rentals/Inspections CRUD,
  Listen mit Filtern, HasOverlap, SaveSignature) inkl. Tenant-Grenze auf jedem Lesepfad.
  Echter Fund dabei behoben: `postgres_repository.go` CreateRental/UpdateRental gaben bei
  einer durch die GIST-Exclusion-Constraint (`uq_rentals_no_overlap`, Migration 000101)
  abgelehnten Race den rohen Postgres-Fehler zurueck statt `ErrRentalConflict` — der
  Service prueft `HasOverlap` VOR dem INSERT, aber Pre-Check und INSERT liegen nicht in
  derselben Transaktion, zwei gleichzeitige Buchungen fuer denselben Zeitraum koennen also
  beide den Pre-Check passieren und um die Constraint racen. Ohne Mapping landete der
  Verlierer via `mapVermietungError`s `default`-Zweig als `codes.Internal` ("internal
  error") statt als `codes.AlreadyExists` mit brauchbarer Meldung. Neue Funktion
  `asRentalConflict` mappt SQLSTATE 23P01 (exclusion_violation) auf `ErrRentalConflict`,
  angewendet in beiden Schreibpfaden. Kein Proto/keine Route/kein RequirePermission/keine
  neue Tabelle angefasst.
- gate: build ok (`go build -p 2 ./internal/vermietung/... ./internal/gateway/...
  ./cmd/gateway/...`) | vet ok | lint ok (0 issues) | test ok (DATABASE_URL gesetzt,
  17 neue Tests + alle bestehenden vermietung-Tests gruen, 0 uebersprungen) | migration
  n.a. (keine Tabelle/Policy angefasst, Migrationskopf 325 lokal = Repo, keine neue
  Migration noetig) | rls-smoke ok (durchgaengig ueber echte Repo-Methoden:
  GetObject/GetRental/ListObjects/ListRentals/ListInspections liefern fuer fremden
  Tenant ErrObjectNotFound/ErrRentalNotFound bzw. 0 Treffer, UpdateObject/UpdateRental/
  SoftDeleteObject/DeleteRental gegen fremden Tenant schlagen mit demselben NotFound
  fehl statt zu leaken) | route-drift n.a. (keine Route angefasst, `route_vermietung.go`
  war nur als Kontext-Quelle gelistet, nicht Ziel dieser Unit — die Gateway-Seite ist
  die separate `cov-gateway-vermietung-objects-and-inspections`)
- coverage: internal/vermietung/postgres_repository.go vorher 3,9 % (173 von 180
  Statements ungedeckt, gemessen im CI-Referenzlauf) -> Paket-gesamt jetzt 82,6 %
  (vorher `internal/vermietung` gesamt 48,2 % laut coverage_start). Je Funktion via
  `go tool cover -func`: alle 22 Funktionen zwischen 66,7 % (UpdateInspection) und
  100 % (NewPostgresRepository, CreateObject, CreateRental, HasOverlap,
  CreateInspection), keine mehr bei 0.
- mutations-probe: zwei Mutationen (Backup vorher per `cp`, danach zurueckgespielt,
  `git diff --stat` am Ende zeigt nur die beabsichtigte `asRentalConflict`-Aenderung):
  (1) in `GetRental` `AND tenant_id = $2` aus der WHERE-Klausel entfernt (Query-Platzhalter
  blieben, Argumentzahl blieb bei zwei) -> sofortiger Laufzeitfehler "expected 1
  arguments, got 2" in `TestPostgresRental_GetUpdateDelete_CrossTenant_NotFound` statt
  eines stillen Cross-Tenant-Leaks — der Test haette auch bei einer subtileren Mutation
  (z. B. Tautologie) denselben Bug gefangen, da er auf `ErrRentalNotFound` prueft statt
  nur auf "kein Fehler". (2) `pgErrCodeExclusionViolation` von "23P01" auf "23505"
  geaendert (falscher SQLSTATE-Code) -> `TestPostgresCreateRental_ConcurrentOverlap_
  OneWinnerOneConflict` sofort ROT: "expected nil or ErrRentalConflict, got ERROR:
  conflicting key value violates exclusion constraint ... (SQLSTATE 23P01) (not the raw
  pg error)" — bestaetigt, dass der Test genau die eben gefixte Luecke haette gefangen,
  waere sie nicht gefixt worden.
- verify vorgaenger: sauber (`34d3eba3` — nur drei neue Testdateien
  (`route_work_time_handlers_test.go`, `work/project/postgres_repository_db_test.go`,
  `work/timeentry/postgres_repository_db_test.go`) plus BACKLOG.yml/JOURNAL.md im Diff,
  kein `.proto`, keine neue Route, kein `RequirePermission`, keine neue Tabelle, kein
  gRPC-Bypass, kein Stub-Marker; Journal-Eintrag selbst dokumentiert bereits eine saubere
  Mutations-Probe mit Backup/Restore)
- neue-units: keine (der einzige echte Fund — fehlendes Conflict-Mapping bei
  Exclusion-Violation — war innerhalb dieser Iteration selbst behebbar, kein Proto/keine
  Migration/kein Deploy-Hazard, also direkt gefixt statt als Fix-Unit verschoben)
- offen: (1) Frage aus `done_when` beantwortet: `renter_name` in `rentals` ist ein reines
  Freitextfeld aus dem Request (`route_vermietung.go:147`, `validate:"required"`),
  NICHT aus dem optionalen `contact_id` abgeleitet und wird beim Anlegen/Aktualisieren nie
  automatisch mit dem Kontaktnamen synchronisiert — Service- und Repository-Ebene
  bestaetigen das (`service.go` CreateRental/UpdateRental setzen `RenterName` nur aus
  `input.RenterName`). Das ist vermutlich Absicht (Mieter muss keinen CRM-Kontakt haben,
  z. B. Laufkundschaft), aber falls doch aus dem Kontakt vorbefuellt werden soll, ist das
  eine Produktentscheidung, kein Bug — daher keine Fix-Unit angelegt. (2) Doppelvermietung
  (`done_when` Punkt 3): technisch NICHT moeglich dank der GIST-Exclusion-Constraint
  `uq_rentals_no_overlap` (Migration 000101, bereits vor diesem Lauf vorhanden) — der
  einzige Bug war die fehlende Fehler-Uebersetzung im Race-Fall, jetzt gefixt und mit
  einem echten Concurrency-Test belegt. (3) Kleine Inkonsistenz beobachtet, nicht als Bug
  gewertet: `HasOverlap`s Pre-Check schliesst nur `status = 'cancelled'` aus, die
  DB-Constraint zusaetzlich auch `'completed'` — ein abgeschlossener historischer
  Zeitraum blockiert im App-Layer strenger als noetig eine neue Buchung mit denselben
  Daten. Sehr unwahrscheinlicher Fall (niemand bucht typischerweise denselben
  Datumsbereich einer bereits abgeschlossenen Miete erneut), daher nur vermerkt, keine
  Fix-Unit. (4) `route_vermietung.go`/`vermietung_grpc.go`/`service.go` sind fachlich
  weiterhin ungetestet auf Gateway-/gRPC-Ebene (0,0/48,4 % laut `coverage_start`) — das
  ist die separate `cov-gateway-vermietung-objects-and-inspections`, absichtlich nicht
  Teil dieser Unit.

## Iteration 41 — cov-gateway-vermietung-objects-and-inspections — done — 2026-08-26 07:03
- commit: 4802caf3ad15e70c0a12f6eec067ea727836824f
- gebaut: Gateway-Tests fuer die neun ungetesteten vermietung-Handler
  (HandleGetObject, HandleUpdateObject, HandleDeleteObject, HandleListRentals,
  HandleGetInspection, HandleListInspections, HandleUpdateInspection,
  HandleSaveRentalSignature, HandleExportRentalReport) plus HandleGetRental (nicht im
  scope-Text genannt, aber ebenfalls 0,0 % und trivial mitgenommen) — je Handler
  ServiceUnavailable, UUID-/JSON-/Validierungsfehlerpfad und ein ReachesRPC-Test.
  Zusaetzlich zwei dokumentierende Fund-Tests auf Service-Ebene (mockRepository):
  `TestSaveRentalSignature_OverwritesExistingSignatureWithoutGuard`
  (internal/vermietung/signature_test.go) und
  `TestService_DeleteObject_ActiveRental_NoReferentialIntegrityGuard`
  (internal/vermietung/service_test.go).
- gate: build ok (`go build -p 2 ./internal/vermietung/... ./internal/gateway/...
  ./cmd/gateway/...`) | vet ok | lint ok (0 issues) | test ok (DATABASE_URL gesetzt,
  ./internal/vermietung/... und ./internal/gateway/... beide gruen, 0 uebersprungen,
  TestOpenAPIRouteDrift lief mit, keine Route angefasst also erwartungsgemaess ok) |
  migration n.a. (keine Tabelle/Policy angefasst) | rls-smoke n.a. (keine Tabelle/Policy
  angefasst, nur Tests)
- coverage: internal/gateway/route_vermietung.go — informeller Funktions-Mittelwert
  (26 Funktionen/Methoden laut `go tool cover -func`) 50,9 % -> 81,7 %; die neun
  scope-Handler + HandleGetRental liegen jetzt alle zwischen 81,2 % und 94,7 % (vorher
  0,0-37,5 %). Paket `internal/gateway` gesamt 67,9 % -> 68,6 % (vorher-Wert selbst
  gemessen, coverage_start der Unit nannte den CI-Referenzwert 56,6 % — Differenz durch
  die vielen zwischen CI-Lauf und dieser Iteration bereits done gebauten Coverage-Units
  in diesem Paket, siehe methodische Warnung im Backlog-Kopf).
- mutations-probe: n.a. — reine Coverage-Unit ohne Produktionscode-Aenderung
  (route_vermietung.go, service.go, postgres_repository.go unveraendert). Die beiden
  neuen Fund-Tests SIND selbst die Probe in dem Sinn, dass sie den jeweiligen Gap
  demonstrieren (Signatur-Overwrite gelingt mit err == nil; Objekt-Loeschung trotz
  aktiver Vermietung gelingt mit err == nil und die Vermietung bleibt danach unveraendert
  auffindbar) — kein Verhalten geaendert, nur belegt.
- verify vorgaenger: sauber (`a6d6a187` geprueft — Fix beschraenkt sich auf
  `pgErrCodeExclusionViolation`/`asRentalConflict` in postgres_repository.go plus
  DB-Testdatei; kein `.proto`, keine neue Route, kein `RequirePermission`, keine neue
  Tabelle, kein gRPC-Bypass, kein Stub-Marker; Diff sauber gegen die acht
  Fehlerklassen)
- neue-units: fix-vermietung-rental-signature-overwritable-after-signing (SaveSignature
  ueberschreibt eine bestehende Unterschrift kommentarlos — derselbe Fehler wie bei
  `fix-rapporte-signature-overwritable-after-signing`, hier fuer vermietung),
  fix-vermietung-delete-object-with-active-rental-dangling-reference (DeleteObject
  loescht ein Objekt weich, ohne laufende Vermietungen zu pruefen — die Vermietung
  bleibt danach mit toter Objektreferenz auffindbar), feat-vermietung-dsar-and-retention-coverage
  (rentals/rental_inspections fehlen komplett in dsar_search.go und haben keinen
  Retention-Handler, obwohl rentals.contact_id auf CRM-Kontakte zeigt und
  rental_inspections Fotos/Freitext personenbezogener Uebergaben traegt)
- offen: (1) done_when-Frage "Ob Uebergabeprotokolle in DSAR und Retention auftauchen"
  ist beantwortet: NEIN, weder DSAR noch Retention decken vermietung ab — als eigene
  Unit `feat-vermietung-dsar-and-retention-coverage` angelegt statt hier mitgebaut
  (Umfang: neues DSAR-Modul + Registrierung, klar ueber reine Gateway-Coverage hinaus).
  (2) `HandleExportRentalReport` wurde nur mit Format `csv` (Default) und `json`
  (unbekanntes Format, landet ungeprueft im ContentType-Fallback) getestet — die
  gRPC-Ebene (`vermietung_grpc.go:563`) ignoriert `format` faktisch komplett und liefert
  immer CSV; das ist vermutlich Absicht (nur ein Exportformat implementiert), aber falls
  JSON/PDF-Export erwartet wird, ist das eine Produktentscheidung, kein Bug — nicht als
  Fix-Unit angelegt. (3) `HandleListRentals`/`HandleGetRentalCalendar` teilen dasselbe
  Silently-Ignore-Muster fuer unparsbare Datumsangaben — dokumentiert, nicht als Fund
  gewertet (konsistent mit dem bereits bestehenden Calendar-Verhalten).

## Iteration 42 — cov-einkauf-service-extended-real-paths — done — 2026-08-26 07:13
- commit: 21879179
- gebaut: Tests fuer die Nullcoverage-Funktionen in service.go/service_extended.go:
  GetSupplier, UpdatePO (inkl. Draft-Only-Guard, Duplicate-PONumber,
  Supplier-Wechsel-NotFound, ExpectedDeliveryDate-Clear), ListPOLines,
  UpdateCatalogItem/DeleteCatalogItem/GetCatalogItem, DeleteSupplierRating/
  ListSupplierRatings, UpdateFrameworkContract/DeleteFrameworkContract/
  GetFrameworkContract/ListFrameworkContracts, CreateContractItem/
  UpdateContractItem/DeleteContractItem, WithInventarAdjuster. Zusaetzlich zwei
  gezielte Status-Tests (ReceiveGoods/PartialReceive auf stornierter PO ->
  ErrPONotReceivable) und ein DB-Test mit krummen Werten
  (TestRecomputePOTotal_SumsRawWithoutPerLineRounding in tenant_write_test.go),
  der die Bug-Hypothese aus dem Scope widerlegt: RecomputePOTotal summiert
  quantity*unit_price roh in Postgres (total_amount ist NUMERIC(15,4), exakt
  wie quantity/unit_price) statt je Zeile zu runden — kein Bug gefunden.
  Ausserdem ein Fund-Test (TestExtended_ReceiveGoods_LinkedToFrameworkContract_
  DoesNotRecordCallOff), der belegt, dass der Docstring-Kommentar an
  PurchaseOrder.FrameworkContractID ("ReceiveGoods will automatically record a
  contract call-off") nicht der Realitaet entspricht — als eigene Unit angelegt.
- gate: build ok (`go build -p 2 ./internal/einkauf/...`) | vet ok | lint ok
  (`golangci-lint run ./internal/einkauf/...`, 0 issues) | test ok
  (DATABASE_URL gesetzt, `go test -count=1 ./internal/einkauf/...`, 0
  uebersprungen, DB-Test lief real gegen kmuhub_app-Rolle, kein `t.Skip`) |
  migration n.a. (keine neue Migration, nur bestehende Spalte
  framework_contract_id gelesen) | rls-smoke n.a. (keine neue Tabelle/Policy) |
  gateway-test n.a. (keine Route angefasst)
- coverage: internal/einkauf 63,9 % -> 79,1 % (selbst gemessen vor/nach,
  deckt sich mit coverage_start der Unit). Funktions-Feinbild
  (`go tool cover -func`): service_extended.go hatte 13 Funktionen bei 0,0 %
  (UpdateCatalogItem, DeleteCatalogItem, GetCatalogItem, DeleteSupplierRating,
  ListSupplierRatings, UpdateFrameworkContract, DeleteFrameworkContract,
  GetFrameworkContract, ListFrameworkContracts, CreateContractItem,
  UpdateContractItem, DeleteContractItem, getContractItemByID) — jetzt liegt
  jede zwischen 66,7 % und 100 %. service.go: GetSupplier 0,0->100, UpdatePO
  0,0->78,8, ListPOLines 0,0->100, WithInventarAdjuster 0,0->100.
- mutations-probe: RecomputePOTotal-SQL testweise auf
  `SUM(ROUND(quantity * unit_price, 2))` je Zeile geaendert (statt
  `SUM(quantity * unit_price)` roh) -> TestRecomputePOTotal_
  SumsRawWithoutPerLineRounding wurde rot (erwartet 0.25, bekam 0.26 bei zwei
  Zeilen a 1x0.125) -> Diff zurueckgedreht (`git diff --stat` danach leer),
  Test wieder gruen. Fuer den Fund-Test (FrameworkContractID) keine separate
  Mutations-Probe noetig — er dokumentiert fehlendes Verhalten, kein
  vorhandenes; sein "Rot-Zustand" waere das Fehlen des Tests selbst.
- verify vorgaenger: sauber (`4802caf3` geprueft — reiner Test-Commit, fuenf
  neue/geaenderte Dateien, alles `_test.go` oder Journal/Backlog; kein `.proto`,
  keine neue Route, kein `RequirePermission`, keine neue Tabelle, kein
  gRPC-Bypass, kein Stub-Marker im Diff)
- neue-units: feat-einkauf-po-framework-contract-call-off-wiring
  (PurchaseOrder.FrameworkContractID wird von CreatePOInput/UpdatePOInput nie
  gesetzt und von ReceiveGoods/PartialReceive nie gelesen, obwohl Modell-
  Kommentar und Migration 000208 genau dafuer angelegt wurden — die
  versprochene automatische Rahmenvertrags-Abrufbuchung existiert nicht)
- offen: (1) total_amount ist ein Netto-Betrag ohne Steuer — tax_rate wird pro
  Zeile gespeichert, aber RecomputePOTotal ignoriert es vollstaendig
  (SUM(quantity*unit_price) ohne tax_rate-Faktor). Keine eigene Steuer-
  arithmetik nachgebaut (done_when-Punkt 3 beantwortet: kein Bug, aber auch
  keine Brutto-Summe verfuegbar — falls das erwartet wird, ist das eine
  Produktentscheidung, keine Fix-Unit). (2) CreateContractCall hat in der
  Postgres-Implementierung denselben Budget-Race-Schutz wie die
  bereits-getesteten Faelle (Transaktion in der Repo-Ebene) — nicht erneut
  gepruecft, da ausserhalb des service_extended.go-Scopes dieser Unit.
  (3) Die neue Unit feat-einkauf-po-framework-contract-call-off-wiring braucht
  eine Produktentscheidung zum Verhalten bei ErrContractBudgetExceeded waehrend
  eines physischen Wareneingangs — im Zweifel `blocked` mit Frage an Luke.

## Iteration 43 — cov-gateway-einkauf-purchase-order-lifecycle — done — 2026-08-26 07:22
- commit: 4107e6b8
- gebaut: Neue Datei `route_einkauf_lifecycle_test.go` deckt die 13 zuvor
  ungetesteten Gateway-Handler ab (HandleGetPO, HandleListPOs, HandleUpdatePO,
  HandleDeletePO, HandleCancelPO, HandleSubmitPO, HandleListPOLines,
  HandleUpdatePOLine, HandleDeletePOLine, HandleReceiveGoods,
  HandlePartialReceive, HandleGetSupplier, HandleUpdateSupplier) — je
  ServiceUnavailable, Invalid-UUID-Validierung, Body-Validierung wo vorhanden,
  und ein "ReachesRPC"-Dokutest (Muster aus route_vermietung_test.go: die
  Gateway-Schicht hat keine eigene Statemachine, sie reicht jede syntaktisch
  gueltige Anfrage direkt an die RPC durch). Zusaetzlich ein neuer Service-Test
  `TestService_ReceiveGoods_DoubleReceive_SecondCallRejected` in
  internal/einkauf/service_test.go, der die Idempotenz-Frage aus dem Scope
  beantwortet: ein zweiter ReceiveGoods-Aufruf auf eine bereits empfangene PO
  wird vom Status-Guard (service.go:612) mit ErrPONotReceivable abgewiesen,
  BEVOR received_quantity oder Inventar-Anpassung ein zweites Mal laufen —
  kein Doppel-Buchen. Die Statusuebergaenge (Draft-only Delete/Update,
  Nicht-stornierbar nach Empfangsbeginn, Submit nur ab Draft mit Zeilen)
  waren bereits vollstaendig durch Service-Tests aus fruaeheren Iterationen
  belegt (TestService_UpdatePO_ClosedRejected, _CancelledRejected,
  TestService_DeletePO_NonDraftRejected, TestService_CancelPO_
  NotCancellableRejected, TestService_SubmitPO_NonDraftRejected/_NoLines) —
  die neuen Gateway-Tests referenzieren sie per Kommentar statt sie zu
  duplizieren. DSAR-Frage zu Lieferanten beantwortet: `grep -n
  "einkauf|supplier|Supplier" internal/security/gdpr/dsar_search.go` liefert
  null Treffer — Lieferanten (potenziell Einzelunternehmer, personenbezogene
  Daten) tauchen in KEINEM DSAR-Ergebnis auf, auch nicht ueber ihre
  `contact_id`-Verknuepfung. Als eigene Unit angelegt (siehe neue-units).
- gate: build ok (`go build -p 2 ./internal/einkauf/... ./internal/gateway/...
  ./cmd/einkauf/... ./cmd/gateway/...`) | vet ok | lint ok (`golangci-lint run
  ./internal/einkauf/... ./internal/gateway/...`, 0 issues) | test ok
  (DATABASE_URL gesetzt, `go test -count=1 ./internal/einkauf/...` 110 Tests,
  0 uebersprungen; `go test -count=1 ./internal/gateway/` gruen, inkl.
  TestOpenAPIRouteDrift da keine Route geaendert) | migration n.a. (keine) |
  rls-smoke n.a. (keine neue Tabelle/Policy) | gateway-test ok (siehe oben,
  Pflicht war ohnehin erfuellt, da Gateway-Paket direkt betroffen)
- coverage: internal/gateway 68,6 % -> 69,3 % (selbst gemessen vor/nach mit
  Datei rausgenommen/reingenommen; weicht vom `coverage_start` der Unit
  (56,6 %, CI-Stand 2026-08-24) deutlich ab, weil zwischenzeitliche
  Iterationen dasselbe Paket bereits angehoben haben — eigene Messung gilt).
  Funktions-Feinbild fuer route_einkauf.go: alle 13 Ziel-Handler jetzt
  zwischen 77,8 % (HandleListSuppliers, unveraendert, nicht im Scope) und
  87,0 % (HandleUpdatePOLine); vorher 0,0 % fuer alle 13. internal/einkauf
  79,1 % -> 79,1 % unveraendert (der neue Idempotenz-Test durchlaeuft bereits
  von TestService_ReceiveGoods_Success/_WrongStatus abgedeckte Zeilen kein
  zweites Mal neu — sein Wert ist der Beweis, nicht neue Coverage).
- mutations-probe: Status-Guard in ReceiveGoods testweise mit `if false && ...`
  deaktiviert -> TestService_ReceiveGoods_DoubleReceive_SecondCallRejected
  wurde rot (`Expected nil, but got: &einkauf.PurchaseOrder{...Status:"received"...}`,
  zweiter Aufruf lief durch statt ErrPONotReceivable zurueckzugeben) ->
  zurueckgedreht, `git diff --stat backend/internal/einkauf/service.go` danach
  leer, Test wieder gruen.
- verify vorgaenger: sauber (`21879179` geprueft — fuenf Dateien, alles
  `_test.go`/Journal/Backlog, kein `.proto`, keine neue Route, kein
  `RequirePermission`, keine neue Tabelle, kein gRPC-Bypass, kein
  Stub-Marker)
- neue-units: feat-gdpr-dsar-search-missing-einkauf-suppliers (Lieferanten
  fehlen komplett im DSAR-Suchergebnis, auch verknuepfte)
- offen: (1) `hooks/backlog-check.py --preflight` meldet weiterhin die
  vorbestehende `blocked`-Unit `fix-409-double-meaning-on-grpc-conflict-routes`
  (Zeile 806, seit Block A der Lauf-12-Vorbereitung so eingetragen) — nicht
  durch diese Iteration verursacht, aber noch nicht nach BACKLOG-NEXT.yml
  verschoben. (2) HandleCreateSupplier/HandleDeleteSupplier/HandleCreatePO/
  HandleAddPOLine/HandleListSuppliers blieben bei ihrer alten Coverage (waren
  nicht im 13er-Scope dieser Unit, hatten teils schon Tests). (3) Das neue
  `feat-gdpr-dsar-search-missing-einkauf-suppliers` unterscheidet zwei Faelle
  (Lieferant mit/ohne `contact_id`) — welcher davon in der Unit selbst gebaut
  wird, ist im Backlog-Eintrag als Bau-Entscheidung offengelassen, nicht als
  Luke-Entscheidung.

## Bilanz Lauf (2026-08-26 07:31)
- iterationen: 41 im Bereich 1-43, davon 41 done, 0 blocked, 0 ohne auswertbare Kopfzeile
- units nach praefix: cov 28 · feat 6 · fix 4 · harden 3
- commits nach typ: chore 51 · test 28 · feat 7 · fix 6 (92 seit 8f3b92b3)
- coverage-delta: internal/work/customfield 0,0 -> 82,6 · vorher 3,9 -> 82,6 · Referenzwert 19,4 -> 78,2 · (17 weitere)
- offen mit entscheidungsbedarf: 15 von 36 nicht-leeren offen:-Zeilen (Treffer auf "Luke"/"Entscheidung")
- minuten je iteration: 10,4 (425 gesamt)

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
- commit: <pending>
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
