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
