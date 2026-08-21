# Backend-Nachtloop — Journal Lauf 10

Append-only. Ein Eintrag je Iteration, **ans Dateiende**, nie einsortieren. Form und Pflichtzeilen
stehen in `ITERATION.md` Schritt 6.

Frühere Läufe liegen vollständig im Archiv:
`archive/lauf-1-2/` (58 Units) · `archive/lauf-3/` (61) · `archive/lauf-4/` (54) ·
`archive/lauf-5/` (41) · `archive/lauf-6/` (46) · `archive/lauf-7/` (71) ·
`archive/lauf-8/` (94) · `archive/lauf-9/` (37, inkl. `logs/`).

---

## Laufkontext

- **Ausgangspunkt:** `backend-loop` auf `origin/main` gemergt (nicht rebased). `main` steht auf
  dem Stand der Lauf-10-Vorbereitung; CI grün auf `1b49a1f3` (Run 32508843776), CI Desktop grün
  (32508843728), CD grün auf `390a75f5`.
- **Migrationen:** Repo-Kopf = lokaler DB-Kopf = **314**, `schema_migrations` clean. Nächste freie
  Nummer wäre 315 — aber immer zur Laufzeit ermitteln:
  `ls backend/migrations | grep -E '^[0-9]{6}' | sort | tail -1`.
  Genau **zwei** Units dieses Laufs bringen eine Migration mit:
  `fix-gobd-belegarchiv-worm-enforcement` (REVOKE-only) und
  `feat-retention-worker-schema-and-engine` (neue Tabelle + RLS).
- **Lokale DB:** läuft und ist ab diesem Lauf **Startbedingung**. `run-loop.ps1` prüft im Vorflug
  Port 5432, die Anmeldung als `kmuhub_app` und den Migrationskopf und bricht ab, wenn eins davon
  fehlt. Grund: `testutil.SkipIfNoDB` (`backend/internal/testutil/rls.go:24`) fragt nur, **ob**
  `DATABASE_URL` gesetzt ist. Am 2026-08-22 an `internal/security/audit` gemessen — mit DB
  33 PASS / 0 SKIP, ohne die Variable 19 PASS / 14 SKIP, beide Male `ok` und Exit 0.
  Der Treiber exportiert `DATABASE_URL` für die Kindprozesse; der `export` aus `ITERATION.md`
  Schritt 5 bleibt trotzdem richtig und schadet nicht.
- **Rolle:** `kmuhub_app` (NOSUPERUSER NOBYPASSRLS), niemals `kmuhub` — der Superuser hat
  BYPASSRLS und würde jede RLS-Lücke durchwinken.
- **Coverage-Ausgangslage:** gesamt **60,2 %** bei Gate 15 %, `internal/gateway` **46,1 %** als
  schwächstes Kernpaket. Vollständige Paketliste im Kopf von `BACKLOG.yml`.
- **Umfang:** 48 vorab geschriebene Units — Block A (14, Datenschutz-Mechanik), Block B (25,
  Gateway-Härtung), Block C (9, Muster-Scans). Block C legt weitere Units zur Laufzeit an; in
  Lauf 9 haben 11 Scans 16 Zusatz-Units erzeugt.

## Was in diesem Lauf gilt

- Pin-Tests umdrehen, nicht löschen. Mutations-Probe ist Pflicht.
- Root Cause statt Symptom: vor jedem Fix alle Caller greppen.
- Eine Prämisse aus dem Backlog, die sich am Code als falsch erweist, wird **nicht trotzdem
  gebaut** — sie wird hier widerlegt und die Unit auf `blocked` gesetzt. Befund 1 im
  `BACKLOG.yml`-Kopf ist genau so entstanden: der angeblich fehlende Foreign Key auf
  `contact_tags` existiert seit Migration 000007.
- Scan-Units ändern kein Verhalten. `neue-units:` muss IDs nennen, die wirklich in
  `BACKLOG.yml` stehen.
- Gesperrt: Frontend/Desktop, CSAT und Public-Token-Routen, Dependency-Bumps, `deploy/`,
  Migrations-Umnummerierungen, Preise und Modul-Zuschnitt.

---

## Iteration 1 — fix-gobd-belegarchiv-worm-enforcement — done — 2026-08-22 00:23
- commit: 2a27d899
- gebaut: Migration `000315_gobd_archive_worm_privileges` entzieht `kmuhub_app` UPDATE und
  DELETE auf `gobd_documents` und `gobd_document_events` (Begründung je Tabelle im
  Migrationskopf). Neue Datei `worm_privileges_db_test.go` mit zwei Tests: einer archiviert als
  kmuhub_app und bekommt auf UPDATE und DELETE beider Tabellen SQLSTATE 42501, während INSERT
  und SELECT weiter gehen (alles in einer Transaktion mit Savepoints, die am Ende zurückgerollt
  wird — mit entzogenem DELETE gibt es keinen Weg, eine committete Fixture-Zeile aufzuräumen);
  der zweite scannt `pg_tables` nach `gobd\_%` und schlägt fehl, sobald eine Archivtabelle
  UPDATE oder DELETE trägt. Der Catalog-Scan ist die Absicherung gegen 000121 Abschnitt 4:
  ALTER DEFAULT PRIVILEGES gibt jeder künftigen Tabelle die Rechte automatisch zurück, ein
  neues `gobd_*` ohne REVOKE wird damit rot statt still beschreibbar.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (41 PASS / 0 SKIP / 0 FAIL, alle mit
  DATABASE_URL gegen kmuhub_app) | migration ok (315 up und down lokal gefahren, Kopf wieder 315)
  | rls-smoke ok (die beiden `tenant_isolation_test.go`-Tests laufen im selben Paket grün)
- coverage: internal/biz/gobdarchive 89,9 % -> 89,9 % (eigene Messung vor und nach der neuen
  Testdatei, deckt sich mit `coverage_start`). Unverändert ist hier das erwartete Ergebnis:
  die Durchsetzung liegt in der Datenbank, die Tests führen keine zusätzliche Go-Anweisung aus.
- mutations-probe: `migrate down 1` (also die REVOKEs zurückgenommen) → beide WORM-Tests rot,
  `_AllArchiveTables` mit "kmuhub_app holds UPDATE/DELETE" für beide Tabellen und
  `_AppRoleCannotMutate`, weil die Mutation durchging. `migrate up` → wieder grün, Kopf 315,
  Diff sauber. Damit hängt der Beweis an der Migration und nicht an etwas Nebensächlichem.
- verify vorgaenger: n.a. — Iteration 1 dieses Laufs. Der letzte Commit `eb55684a` ist die
  Lauf-10-Vorbereitung (docs), kein Loop-Commit.
- neue-units: keine
- offen:
  - Zwei Annahmen habe ich am laufenden System belegt, nicht nur behauptet: (a) Postgres führt
    die referentiellen Aktionen als Tabelleneigentümer aus, deshalb funktioniert
    `source_invoice_id ON DELETE SET NULL` trotz entzogenem UPDATE weiter — bewiesen durch
    `TestPostgresRepository_List_FiltersBySourceInvoice`, dessen `finance_invoices`-Cleanup
    grün bleibt; (b) im Go-Code gibt es kein UPDATE/DELETE gegen die Archivtabellen (Grep über
    `internal/`, nur INSERT und SELECT in `postgres_repository.go`).
  - Ich habe 14 `testutil.CleanupRow`-Aufrufe auf die Archivtabellen aus
    `postgres_repository_db_test.go` und `tenant_isolation_test.go` entfernt und die Absicht
    im Dateikopf notiert. Grund: `CleanupRow` schluckt Fehler mit `t.Logf`, die Tests wären
    also grün geblieben — aber jeder Lauf hätte 16 "permission denied"-Zeilen ins Log
    geschrieben, die aussehen wie ein Defekt. Die Fixture-Zeilen liegen unter einem
    Wegwerf-Tenant pro Test und sind für alles andere unsichtbar; dass sie liegen bleiben, ist
    genau das, was WORM bedeutet. Der `finance_invoices`-Cleanup ist bewusst geblieben.
  - `go test ./internal/gateway/` habe ich nicht gefahren: keine Route, kein OpenAPI-Eintrag
    berührt. Die Änderung ist auf `internal/biz/gobdarchive` plus Migration begrenzt.
  - Für Production: die Migration ist reines REVOKE, ohne Datenänderung. Beim Deploy läuft sie
    als `kmuhub` (Tabelleneigentümer), in CI als `kmuhub_test` — beide dürfen widerrufen.

## Iteration 2 — feat-contact-deletion-cascade-impact-preview — done — 2026-08-22 00:55
- commit: 473aefa8
- gebaut: `GET /api/v1/contacts/{id}/deletion-preview` (Route + OpenAPI-Eintrag), neue gRPC-RPC
  `PreviewContactDeletion` (Proto regeneriert), `Repository.DeletionImpact` und
  `Service.PreviewDeletion` in `internal/crm/contact`. Repository liest die referenzierenden
  Fremdschluessel LIVE aus `pg_catalog` (pg_constraint/pg_class/pg_attribute) statt einer
  hartcodierten Tabellenliste, zaehlt je Treffer die betroffenen Zeilen und meldet nur Tabellen
  mit mindestens einem Treffer zurueck. Reine Lesefunktion, aendert nichts.
- gate: build ok | vet ok | lint ok (0 issues, crm+gateway+server) | test ok
  (internal/crm/contact 110 PASS / 0 SKIP / 0 FAIL, DATABASE_URL gegen kmuhub_app; alle
  internal/crm/... Pakete gruen mit `-p 1`, Parallel-Lauf ueber alle Pakete sprengt lokal die
  Postgres-Verbindungsgrenze — kein Befund an meinem Code) | migration n.a. (keine
  Schemaaenderung) | `go test ./internal/gateway/` gruen (TestOpenAPIRouteDrift inklusive) |
  `go test ./internal/server/...` gruen | `swagger-cli validate api/openapi.yaml` gruen
- coverage: internal/crm/contact 80,4 % -> 80,4 % (eigene Messung vor/nach via `git stash` auf
  genau die von mir geaenderten Dateien, deckt sich mit `coverage_start`). Neuer Code und neue
  Tests halten sich die Waage; keine Regression.
- mutations-probe: in `DeletionImpact` die WHERE-Klausel der pg_constraint-Query um `AND false`
  ergaenzt → `TestRepository_DeletionImpact_TenantScopedLiveFromCatalog` wird rot (leere
  Impact-Liste). Zurueckgedreht → gruen, `git diff --stat` zeigt nur die urspruengliche
  Aenderung (105 Zeilen, reine Insertion).
- verify vorgaenger: sauber. `2a27d899` (GoBD-WORM-Migration) geprueft gegen alle acht
  Fehlerklassen — reine REVOKE-Migration plus Tests, keine Route/Proto/RBAC/Tenant-Tabelle
  betroffen, Migrationskopf und Begruendung je Tabelle vorhanden.
- neue-units: fix-contact-delete-merged-into-no-action-unchecked
- offen:
  - Root-Cause-Fund waehrend des Bauens: `contacts.merged_into_id` (gesetzt von `MergeInto`)
    traegt `ON DELETE NO ACTION`, nicht CASCADE/SET NULL/RESTRICT. `IsInUse` prueft das nicht —
    loescht man den Primary-Kontakt eines abgeschlossenen Merges, faellt der DELETE nicht mit
    dem sauberen 409 durch, sondern mit einem unbehandelten FK-Fehler direkt aus der DB. Live an
    der lokalen DB verifiziert (`confdeltype = 'a'` fuer `merged_into_id` vs. `'n'` fuer das
    zweite Selbstbezug-Feld `referred_by_contact_id`). Nicht selbst gefixt, da diese Unit als
    reine Lesefunktion beschraenkt war (`darf unter keinen Umstaenden etwas veraendern`) und die
    Wahl zwischen Migrationsfix und `IsInUse`-Erweiterung eine echte Entscheidung ist. Neue Unit
    `fix-contact-delete-merged-into-no-action-unchecked` am Backlog-Ende angelegt.
  - Der Kaskaden-Befund im Laufkopf (Befund 2, "8 SET NULL") zaehlt `merged_into_id` nicht mit —
    die Zahl 8 bleibt richtig fuer SET-NULL-Tabellen, aber `contacts` traegt zwei
    Selbstbezug-FKs, nicht wie dort implizit unterstellt nur einen. Kein Korrekturbedarf am
    Kopf selbst, nur zur Einordnung: die neue Route zaehlt inzwischen alle 15 FK-Faelle, der Kopf
    zaehlt nur die drei benannten Kategorien.
  - RLS-Smoke im engeren Sinn (Tabelle/Policy angefasst) entfaellt, da keine Migration noetig
    war; die eigene DB-Testfunktion deckt Tenant-Isolation fuer die neue Query explizit ab
    (fremder Tenant-Kontext liefert 0 Impacts fuer dieselbe Contact-ID).

## Iteration 3 — feat-dsar-search-contact-custom-fields-and-tags — done — 2026-08-22 00:51
- commit: 7ac232c3
- gebaut: `customFieldsModule` und `tagsModule` in `internal/security/gdpr/dsar_search.go`,
  verkabelt in `SearchByQuery` zwischen der statischen "CRM Kontakte"-Karte und
  `consentModule`. Custom Fields lesen `contact_custom_field_values` mit JOIN auf
  `custom_field_definitions` fuer den `field_label` (Klarname statt UUID) und JOIN auf
  `contacts` fuer den Tenant-Filter (die Wertetabelle traegt selbst keine `tenant_id`,
  ihre RLS-Policy scoped ueber den Join). `formatCustomFieldValue` dekodiert die
  JSONB-Zelle nach Go-Typ (string/bool/float64/[]any/nil), nicht nach dem `field_type`
  der Definition, und deckt damit alle sechs Feldtypen ohne Sonderfall ab. Tags lesen
  `contact_tags` (traegt eine eigene `tenant_id`-Spalte) mit JOIN auf `tags` fuer den
  Namen. Beide folgen der `consentModule`-Vorlage: `nil` bei leerem Ergebnis, kein
  leeres Modul in der Oberflaeche.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (64 PASS / 0 SKIP / 0 FAIL,
  DATABASE_URL gegen kmuhub_app) | migration n.a. (keine Schemaaenderung, Migrationskopf
  bereits bei 315) | rls-smoke n.a. (keine neue Tabelle/Policy; Tenant-Isolation ist Teil
  der neuen DB-Testfunktion) | `go test ./internal/gateway/` n.a. (keine Route/OpenAPI
  beruehrt)
- coverage: internal/security/gdpr 61,2 % -> 61,7 % (eigene Messung per `git stash` auf
  genau die beiden geaenderten Dateien, deckt sich mit `coverage_start`)
- mutations-probe: in `customFieldsModule` der WHERE-Klausel `AND false` angehaengt ->
  `TestSearchByQuery_ContactCustomFieldsAndTags_Integration` wird rot ("Benutzerdefinierte
  Felder" fehlt in der Modulliste). Zurueckgedreht -> gruen, `git diff --stat` zeigt nur
  die drei erwarteten Dateien (BACKLOG.yml, dsar_search.go, dsar_search_test.go).
- verify vorgaenger: sauber. `473aefa8` (Contact-Deletion-Preview) gegen alle acht
  Fehlerklassen geprueft — Handler geht ueber `crmClient.PreviewContactDeletion`, Proto
  und generierte `.pb.go`/`_grpc.pb.go` im selben Commit regeneriert, kein neuer
  `RequirePermission`-Guard (nutzt die bestehende `contacts:read`), reine Lesefunktion
  ohne neue Tabelle, Route in `openapi.yaml` dokumentiert (laut Journal Iteration 2 mit
  `swagger-cli validate` geprueft), kein Alt-Guard ersetzt.
- neue-units: keine
- offen:
  - Der neue Testcase seedet `contact_tags` ueber die echte Repository-Methode
    `contact.PostgresRepository.AddTags`, nicht ueber `testutil.SeedRow` — die Tabelle hat
    eine zusammengesetzte Primaerschluessel (`contact_id, tag_id`) ohne `id`-Spalte, mit
    der `SeedRow`s `RETURNING id` bricht (`column "id" does not exist`). Fuer kuenftige
    Units, die eine Junction-Tabelle ohne eigene `id`-Spalte seeden wollen, ist das der
    Weg: ueber die reale Schreibmethode gehen, nicht `SeedRow` erzwingen.
  - `go tool cover -func` zeigt fuer `internal/security/gdpr` weiterhin mehrere 0,0 %-
    Funktionen aus anderen Dateien (Erasure-Handler-Registrierung, DSAR-Suche fuer
    Formulare/Dokumente/Helpdesk-Nachrichten/Benutzer-Module) — das ist der erwartete
    Rest-Scope der Units A4 bis A9, hier nicht angefasst.

## Iteration 4 — feat-dsar-search-contact-form-submissions — done — 2026-08-22 00:57
- commit: 05457053
- gebaut: `formSubmissionsModule` in `internal/security/gdpr/dsar_search.go`, verkabelt in
  `SearchByQuery` zwischen `deals` und `activities`. Der Entwurf ging von einer Spalte
  `submitted_by` mit der Kontakt-E-Mail aus — das Schema widerlegt das:
  `form_submissions.submitted_by` (TEXT NULL) traegt bei authentifizierten Einreichungen die
  Mitarbeiter-User-ID (`route_formulare.go:451-454`, Fallback auf `userID` wenn nicht explizit
  gesetzt) und bei oeffentlichen Share-Link-Einreichungen ueberhaupt nichts
  (`form_share.go:301-309`, `SubmittedBy` bleibt im Struct-Literal aus). Die Identitaet des
  Einreichenden liegt ausschliesslich im JSONB `answers`-Payload, unter einer vom Formularautor
  gewaehlten Feld-ID. Der Match laeuft daher ueber `form_schemas.fields` (JSONB-Array mit
  `id`/`type`/`label`/optionalem `role`): ein `EXISTS`-Subquery mit `jsonb_array_elements` prueft
  je Einreichung, ob irgendein Feld vom Typ `email` einen Wert traegt, der (case-insensitive)
  der Kontakt-E-Mail entspricht. Treffer werden pro beantwortetem Feld offengelegt (Formular,
  Datum, Feld-Label, Wert), mit `formatCustomFieldValue` (aus Iteration 3) fuer die
  Wert-Rendering — Wiederverwendung statt zweiter JSON-Typumwandlung. Einreichungen, deren Schema
  geloescht wurde (`form_schema_id` durch `ON DELETE SET NULL` auf NULL gesetzt), werden per
  INNER JOIN ausgeschlossen und das ist im Docstring und im Test dokumentiert — ohne Schema ist
  nicht feststellbar, welches Feld eine E-Mail war.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (65 PASS / 0 SKIP / 0 FAIL in
  internal/security/gdpr, DATABASE_URL gegen kmuhub_app; internal/formulare zusaetzlich gruen)
  | migration n.a. (keine Schemaaenderung) | rls-smoke n.a. (keine neue Tabelle/Policy;
  Tenant-Isolation ist Teil der neuen DB-Testfunktion) | `go test ./internal/gateway/` n.a.
  (keine Route/OpenAPI beruehrt)
- coverage: internal/security/gdpr 61,7 % -> 62,2 % (eigene Messung per `git stash` auf genau
  die beiden geaenderten Dateien; deckt sich mit `coverage_start` aus der Unit)
- mutations-probe: in `formSubmissionsModule` der WHERE-Klausel `AND false` angehaengt ->
  `TestSearchByQuery_ContactFormSubmissions_Integration` wird rot ("Formulareinreichungen" fehlt
  in der Modulliste). Zurueckgedreht -> gruen, `git diff --stat` zeigt nur die drei erwarteten
  Dateien (BACKLOG.yml, dsar_search.go, dsar_search_test.go).
- verify vorgaenger: sauber. `7ac232c3` (Custom Fields/Tags DSAR) gegen die acht Fehlerklassen
  geprueft — reine additive Query-Funktionen ohne gRPC-Layer, keine neue Tabelle/Guard/Route,
  alle referenzierten Helfer (`dsarMaxRows`, `dsarTimeLayout`, `boolLabel`, `fieldValueRecord`)
  existierten bereits, kein Stub/TODO im Diff.
- neue-units: keine
- offen:
  - Der Test seedet `form_schemas`/`form_submissions` direkt per `testutil.SeedRow` (JSONB-Spalten
    `fields`/`answers` als `[]byte`) — das folgt demselben Muster wie
    `PostgresRepository.CreateSchema`/`CreateSubmission` und hat keine Besonderheiten wie die
    Junction-Tabelle aus Iteration 3.
  - `role: "requester_email"` (aus `formulare/models.go:79`, fuer Intake-gebundene Formulare)
    wird bewusst NICHT zusaetzlich geprueft — jedes Feld vom Typ `email` zaehlt, unabhaengig von
    seiner Rolle, weil auch reine (nicht Intake-gebundene) Formulare ein E-Mail-Feld ohne Rolle
    haben koennen. Ein Formular mit `role: requester_email` auf einem NICHT als `email` typisierten
    Feld (Schema-seitig eigentlich ausgeschlossen, siehe `service.go` Validierung) wuerde damit
    nicht matchen — laut aktueller Validierung ist das kein realer Fall.
  - Rest-Scope aus A5/A6/A7/A8/A9 (Dokumente, Helpdesk-Nachrichten, User-Chat/Work/Kalender/
    Notification) weiterhin offen, hier nicht angefasst.

## Iteration 5 — feat-dsar-search-contact-documents — done — 2026-08-22 01:04
- commit: fa204b9c
- gebaut: `documentsModule` in `internal/security/gdpr/dsar_search.go`, verkabelt in
  `SearchByQuery` direkt nach `tagsModule`. Liest `document_entity_links` (entity_type='contact',
  Literal wie in `route_crm_contact_files.go`/`document/file`-Paket verwendet, keine eigene
  Konstante vorhanden) gejoint auf `document_files`, gefiltert auf Tenant und `NOT is_deleted`.
  Ausgegeben werden ausschliesslich Dateiname, MIME-Typ, Groesse (Bytes) und Hochladedatum —
  kein `storage_key`, kein `thumbnail_key`, kein Inhalt. Soft-geloeschte (Papierkorb-)Dateien
  sind per `NOT f.is_deleted` ausgeschlossen, mit Begruendung im Docstring (deckt sich mit der
  Trash-Behandlung in der Datei-UI).
- gate: build ok | vet ok | lint ok (0 issues) | test ok (66 PASS / 0 SKIP / 0 FAIL,
  DATABASE_URL gegen kmuhub_app) | migration n.a. (keine Schemaaenderung) | rls-smoke n.a.
  (keine neue Tabelle/Policy; Tenant-Isolation ist Teil der neuen DB-Testfunktion) |
  `go test ./internal/gateway/` n.a. (keine Route/OpenAPI beruehrt)
- coverage: internal/security/gdpr 62,2 % -> 62,7 % (eigene Messung per `git stash` auf genau
  die beiden geaenderten Dateien; deckt sich mit dem kumulativen Stand aus Iteration 4, nicht
  mit dem veralteten `coverage_start` der Unit von 61,2 %, weil A3/A4 dasselbe Paket zuvor schon
  angehoben haben)
- mutations-probe: in `documentsModule` der WHERE-Klausel `AND false` angehaengt ->
  `TestSearchByQuery_ContactDocuments_Integration` wird rot ("Dokumente" fehlt in der
  Modulliste). Zurueckgedreht -> gruen, `git diff --stat` zeigt fuer `dsar_search.go` nur
  57 Zeilen rein additiv (0 Loeschungen).
- verify vorgaenger: sauber. `05457053` (Form-Submissions-DSAR) gegen alle acht Fehlerklassen
  geprueft — kein gRPC-Layer beruehrt (reine additive Query-Funktion im gdpr-Paket selbst, wie
  alle anderen DSAR-Module), kein Stub/TODO, kein `.proto` im Diff, kein neuer
  `RequirePermission`-Guard, keine neue Tabelle/Migration, Wire-Shape unveraendert (DSARModule/
  DSARRecord wie ueberall), keine neue Route, kein Alt-Guard ersetzt. `git show --stat` zeigt
  nur `dsar_search.go`/`dsar_search_test.go`/`BACKLOG.yml`.
- neue-units: keine
- offen:
  - `document_entity_links.entity_type` ist ein freier VARCHAR(50) ohne Enum/Konstante — der
    Literal `'contact'` ist an drei Stellen im Code dupliziert (jetzt vier: hier, `document/file`,
    `route_crm_contact_files.go`). Kein neuer Befund dieser Iteration, aber falls C-Scans eine
    Sammel-Unit fuer "String-Literale vereinheitlichen" anlegen, gehoert diese Stelle dazu.
  - Rest-Scope aus A6/A7/A8/A9 (Helpdesk-Nachrichten, User-Chat/Work/Kalender/Notification)
    weiterhin offen, hier nicht angefasst.
