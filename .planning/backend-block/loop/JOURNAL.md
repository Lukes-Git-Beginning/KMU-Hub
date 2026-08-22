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

## Iteration 6 — feat-dsar-search-contact-helpdesk-messages — done — 2026-08-22 01:15
- commit: de357fa1
- gebaut: `helpdeskMessagesModule` in `internal/security/gdpr/dsar_search.go`, verkabelt in
  `SearchByQuery` direkt nach `helpdeskModule`. Liest `ticket_messages` gejoint auf `tickets`
  (Filter `t.contact_id`, `tm.tenant_id`, `NOT tm.internal`), chronologisch aufsteigend
  dargestellt. Interne Notizen (`internal = true`) sind ausdruecklich ausgeschlossen — sie sind
  Arbeitsmaterial des Bearbeiters und tragen haeufig eine Einschaetzung UEBER die betroffene
  Person, nicht an sie gerichtete Kommunikation. Kuerzung: Query holt `dsarMaxRows+1` Zeilen
  neueste-zuerst; wird die Grenze ueberschritten, faellt die AELTESTE der geholten Zeilen weg
  (nicht die neueste), das Ergebnis wird fuer die Anzeige umgedreht und ein sichtbarer
  Hinweissatz ("gekuerzt auf die 50 neuesten Nachrichten") als erster Record vorangestellt —
  kein stilles Abschneiden.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (68 PASS / 0 SKIP / 0 FAIL in
  internal/security/gdpr, DATABASE_URL gegen kmuhub_app; internal/helpdesk zusaetzlich gruen)
  | migration n.a. (keine Schemaaenderung) | rls-smoke n.a. (keine neue Tabelle/Policy;
  Tenant-Isolation ist Teil der neuen DB-Testfunktion) | `go test ./internal/gateway/` n.a.
  (keine Route/OpenAPI beruehrt)
- coverage: internal/security/gdpr 62,7 % -> 63,9 % (eigene Messung per `git stash` auf genau
  die beiden geaenderten Dateien; weicht vom `coverage_start` der Unit von 61,2 % ab, weil A3-A5
  dasselbe Paket in diesem Lauf bereits angehoben haben — siehe Iteration 5)
- mutations-probe: in `helpdeskMessagesModule` die WHERE-Klausel um `AND NOT tm.internal`
  gekuerzt -> `TestSearchByQuery_ContactHelpdeskMessages_Integration` wird rot (interne Notiz
  taucht zusaetzlich im Export auf). Zurueckgedreht -> gruen, `git diff --stat` zeigt fuer
  `dsar_search.go` 75 Zeilen rein additiv (0 Loeschungen).
- verify vorgaenger: sauber. `fa204b9c` (Documents-DSAR) gegen alle acht Fehlerklassen geprueft
  (`git show --stat` und Volltext) — reine additive Query-Funktion im gdpr-Paket, kein gRPC-
  Layer, kein `.proto`, kein neuer `RequirePermission`-Guard, keine neue Tabelle/Migration,
  Wire-Shape (DSARModule/DSARRecord) unveraendert, keine neue Route, kein Alt-Guard ersetzt.
- neue-units: keine
- offen:
  - Fuer die Kuerzungs-Probe habe ich einen eigenen Test mit 55 Nachrichten geschrieben
    (`TestSearchByQuery_ContactHelpdeskMessages_Truncation_Integration`), der ueber `pool.Exec`
    direkt seedet statt ueber `testutil.SeedRow` (55 Einzelinserts mit `RETURNING id` waeren
    unnoetiger Overhead) und per `make_interval` gestaffelte `created_at`-Werte erzeugt. Cleanup
    laeuft ueber das Ticket-CASCADE, keine 55 einzelnen `CleanupRow`-Aufrufe noetig.
  - Rest-Scope aus A7/A8/A9 (User-Chat/Work/Kalender/Notification) weiterhin offen, hier nicht
    angefasst.

## Iteration 7 — feat-dsar-search-user-chat-module — done — 2026-08-22 01:19
- commit: 784f252d
- gebaut: `matchUsers` in `internal/security/gdpr/dsar_search.go` von "scannen und sofort
  DSARPerson bauen" auf "erst alle User-Zeilen sammeln, `rows.Close()`, dann pro User anreichern"
  umgestellt — dieselbe Reihenfolge wie beim Kontaktpfad (`matchContacts` liefert zuerst eine
  Slice, `SearchByQuery` haengt danach die Module an), damit keine zweite `pool.Query` waehrend
  eines offenen `rows.Next()`-Durchlaufs laeuft. Zwei neue Module fuer einen gematchten Benutzer:
  `chatMessagesModule` (Tabelle `messages`, gefiltert auf `created_by = userID` UND `tenant_id`,
  `NOT is_deleted`, mit derselben Kuerzungs-/Umkehr-Logik wie `helpdeskMessagesModule` aus
  Iteration 6 — dsarMaxRows+1 holen, aeltestes fallen lassen, sichtbarer Kuerzungshinweis) und
  `chatMembershipsModule` (Tabelle `channel_memberships`, gefiltert auf `user_id` UND
  `tenant_id`, liefert Kanal/Rolle/Beigetreten). Beide Tabellen sind exakt die, die
  `ChatErasureHandler.ExecuteErasure` anfasst (`erasure.go:296` `messages.created_by`,
  `erasure.go:308` `channel_memberships.user_id`) — die Tabellenliste deckt sich, kein
  Befund fuer C2. Nachrichten anderer Teilnehmer im selben Kanal werden NICHT exportiert
  (Kernanforderung der Unit), per Test mit Zwei-Personen-Kanal belegt.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (70 PASS / 0 SKIP / 0 FAIL in
  internal/security/gdpr, DATABASE_URL gegen kmuhub_app) | migration n.a. (keine
  Schemaaenderung, `messages`/`channels`/`channel_memberships` haben bereits tenant_id seit
  Migration 000112) | rls-smoke n.a. (keine neue Tabelle/Policy; Tenant-Isolation ist Teil
  beider neuer DB-Testfunktionen, RLS traegt sie) | `go test ./internal/gateway/` n.a.
  (keine Route/OpenAPI beruehrt)
- coverage: internal/security/gdpr 63,9 % -> 64,6 % (eigene Messung, volles Paket nach dem
  Umbau; weicht vom `coverage_start` der Unit von 61,2 % ab, weil A3-A6 dasselbe Paket in
  diesem Lauf bereits angehoben haben — siehe Iterationen 3-6)
- mutations-probe: `chatMessagesModule`-WHERE von `m.created_by = $2 AND NOT m.is_deleted` auf
  `(m.created_by = $2 OR TRUE) AND NOT m.is_deleted` geaendert ->
  `TestSearchByQuery_UserChatMessages_Integration` wird rot (Nachricht des zweiten
  Kanalteilnehmers taucht zusaetzlich im Export auf, Diff zeigt genau das). Zurueckgedreht ->
  gruen (70/70), `git diff --stat` zeigt fuer `dsar_search.go` einen Umbau von `matchUsers`
  (18 Zeilen geloescht, siehe gebaut:) plus rein additive neue Funktionen — kein
  unbeabsichtigter Verlust, Volltext-Diff geprueft.
- verify vorgaenger: sauber. `de357fa1` (Helpdesk-Nachrichten-DSAR) gegen alle acht
  Fehlerklassen geprueft (`git show --stat` und Volltext) — reine additive Query-Funktion im
  gdpr-Paket, kein gRPC-Layer, kein `.proto`, kein neuer `RequirePermission`-Guard, keine neue
  Tabelle/Migration, Wire-Shape (DSARModule/DSARRecord) unveraendert, keine neue Route, kein
  Alt-Guard ersetzt.
- neue-units: keine
- offen:
  - `channel_memberships` hat keinen Surrogat-`id` (Composite-PK `channel_id, user_id`) —
    `testutil.SeedRow` funktioniert dort nicht (`RETURNING id` schlaegt fehl), der Test seedet
    per rohem `pool.Exec`/`QueryRow`. Kein neuer Befund, aber relevant fuer kuenftige Tests auf
    dieser Tabelle.
  - Rest-Scope aus A8/A9 (User-Work/Kalender/Notification) weiterhin offen, hier nicht
    angefasst.

## Iteration 8 — feat-dsar-search-user-work-module — done — 2026-08-22 01:24
- commit: bbcaf38c
- gebaut: Drei neue Module in `internal/security/gdpr/dsar_search.go`, verkabelt in `matchUsers`
  direkt nach den Chat-Modulen aus Iteration 7. `tasksModule` liest `tasks` gefiltert auf
  `tenant_id` UND (`assignee_id` ODER `created_by` = Subjekt), mit `Rolle`-Spalte
  ("Ersteller", "Zugewiesen" oder beides) und Status-Namen per LEFT JOIN auf
  `project_statuses` (COALESCE '' bei fehlendem Status). `taskCommentsModule` liest
  `task_comments` gefiltert auf `author_id`, mit derselben Kuerzungs-/Umkehr-Logik wie die
  Chat-Nachrichten aus Iteration 7 (dsarMaxRows+1 holen, aeltestes fallen lassen, sichtbarer
  Kuerzungshinweis). `timeEntriesModule` liest `time_entries` gefiltert auf `user_id`, aber
  AGGREGIERT per SQL `GROUP BY date_trunc('month', started_at)` statt einzelne Zeilen zu
  listen (Notiz der Unit: "falls die Menge gross wird, aggregieren statt weglassen") — die
  Aggregation steht im Modultitel ("Zeiterfassung (aggregiert pro Monat)"), damit sie nie als
  rohe Vollstaendigkeit missverstanden wird. Alle drei Tabellen sind exakt die, die
  `WorkErasureHandler.ExecuteErasure` anfasst (`erasure.go:385` tasks, `erasure.go:394`
  time_entries, `erasure.go:406` task_comments) — die Tabellenliste deckt sich, kein Befund
  fuer C2.
  Bug waehrend des Bauens gefunden und in derselben Iteration behoben (kein Produktionscode
  vorher betroffen, da die Funktion neu ist): `(t.assignee_id = $2)` liefert SQL-`NULL` statt
  `false`, wenn `assignee_id` NULL ist (Task nur ueber `created_by` gematcht) — das Scannen in
  `*bool` schlug dann mit "cannot scan NULL into *bool" fehl. Fix: `IS NOT DISTINCT FROM`.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (73 PASS / 0 SKIP / 0 FAIL in
  internal/security/gdpr, DATABASE_URL gegen kmuhub_app; internal/security/... komplett gruen)
  | migration n.a. (keine Schemaaenderung; tasks/time_entries/task_comments haben tenant_id +
  RLS-Policy bereits seit Migration 000109/106, gegen die lokale DB per `\d` bestaetigt)
  | rls-smoke n.a. (keine neue Tabelle/Policy; Tenant-Isolation ist Teil aller drei neuen
  DB-Testfunktionen, RLS traegt sie) | `go test ./internal/gateway/` n.a. (keine Route/OpenAPI
  beruehrt)
- coverage: internal/security/gdpr 64,6 % -> 65,6 % (eigene Messung per `git worktree add
  HEAD~1` gegen den Vorgaenger-Commit, danach `git worktree remove`; deckt sich mit dem in
  Iteration 7 protokollierten Nachher-Wert — kein Drift durch parallele Iterationen)
- mutations-probe: in `tasksModule` die Rollen-Zuweisung `if isCreator { roles = append(roles,
  "Ersteller") }` auf `if isAssignee` geaendert -> `TestSearchByQuery_UserTasks_Integration`
  wird rot (erwartet "Zugewiesen" fuer den nur-zugewiesenen Task, tatsaechlich
  "Ersteller, Zugewiesen"). Zurueckgedreht -> gruen (alle security/gdpr-Tests), `git diff
  --stat` zeigt fuer `dsar_search.go` 176 Zeilen rein additiv (0 Loeschungen).
- verify vorgaenger: sauber. `784f252d` (User-Chat-DSAR, Iteration 7) gegen alle acht
  Fehlerklassen geprueft (`git show --stat` und Volltext) — reine additive Query-Funktionen im
  gdpr-Paket (kein gRPC-Layer, kein Stub/TODO), kein `.proto` im Diff, kein neuer
  `RequirePermission`-Guard, keine neue Tabelle/Migration, Wire-Shape (DSARModule/DSARRecord)
  unveraendert, keine neue Route, kein Alt-Guard ersetzt. `matchUsers`-Umbau (Sammeln vor
  Anreicherung, `rows.Close()` vor der zweiten Query) ist beabsichtigt und in der Commit-Message
  begruendet.
- neue-units: keine
- offen:
  - `work/reaction` (message_reactions) haengt an CHAT-Nachrichten, nicht am Work-Modul,
    und wird von KEINEM Erasure-Handler angefasst (weder Chat- noch WorkErasureHandler) —
    reine Beobachtung, kein Befund dieser Unit (der Scope war explizit "Tabellenliste deckt
    sich mit WorkErasureHandler", der reactions nicht kennt). Falls ein C-Scan die
    Erasure-Handler-Vollstaendigkeit prueft, gehoert diese Luecke dorthin.
  - Rest-Scope aus A9 (User-Kalender/Notification) weiterhin offen, hier nicht angefasst.

## Iteration 9 — feat-dsar-search-user-calendar-notification-modules — done — 2026-08-22 01:35
- commit: d50754d5
- gebaut: Sechs neue Module in `internal/security/gdpr/dsar_search.go`, verkabelt in
  `matchUsers` direkt nach den Work-Modulen aus Iteration 8. `calendarEventsModule` liest
  `calendar_events` LEFT JOIN `event_attendees` gefiltert auf `created_by = $2 OR
  ea.user_id = $2`, mit `Rolle`-Spalte ("Ersteller"/"Teilnehmer (rsvp_status)") — bei
  Terminen mit mehreren Teilnehmern wird NUR die eigene RSVP der betroffenen Person
  aufgenommen, nicht die Teilnehmerliste (per Mutations-Probe und Test belegt).
  `calendarPreferencesModule` liest `user_calendar_preferences` (PK `user_id`, kein
  Surrogatschluessel) als Feld/Wert-Modul wie "Benutzerkonto". `notificationsModule` liest
  `notifications` mit derselben Kuerzungs-/Umkehr-Logik wie Chat-Nachrichten/Aufgaben-
  Kommentare (dsarMaxRows+1 holen, aeltestes fallen lassen, sichtbarer Kuerzungshinweis).
  `notificationPreferencesModule`, `notificationQuietHoursModule` (Feld/Wert, PK `user_id`
  bzw. UNIQUE(user_id)) und `notificationMutesModule` runden die vier Tabellen des
  `NotificationErasureHandler` ab. Alle sieben Tabellen sind exakt die, die
  `CalendarErasureHandler.ExecuteErasure` (erasure.go:461/490) und
  `NotificationErasureHandler.ExecuteErasure` (erasure.go:560/567/574/581) anfassen — die
  Tabellenlisten decken sich, kein Befund fuer C2.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (79 PASS / 0 SKIP / 0 FAIL in
  internal/security/gdpr, DATABASE_URL gegen kmuhub_app; internal/security/... komplett
  gruen) | migration n.a. (keine Schemaaenderung; alle sieben Tabellen haben tenant_id +
  RLS-Policy bereits seit Migration 000106/109/110, gegen die lokale DB per `\d` bestaetigt)
  | rls-smoke n.a. (keine neue Tabelle/Policy; Tenant-Isolation ist Teil aller sechs neuen
  DB-Testfunktionen, RLS traegt sie) | `go test ./internal/gateway/ -run
  TestOpenAPIRouteDrift` ok (n.a. fuer diese Unit, keine Route/OpenAPI beruehrt, trotzdem
  zur Sicherheit gelaufen)
- coverage: internal/security/gdpr 65,6 % -> 67,4 % (eigene Messung per `git worktree add
  bbcaf38c` gegen den Vorgaenger-Commit, danach `git worktree remove`; deckt sich mit dem in
  Iteration 8 protokollierten Nachher-Wert — kein Drift durch parallele Iterationen)
- mutations-probe: in `calendarEventsModule` `if isCreator { roles = append(roles,
  "Ersteller") }` auf `if !isCreator` geaendert -> `TestSearchByQuery_UserCalendarEvents_Integration`
  wird rot (zwei Assertions: "Ersteller" fehlt beim eigenen Termin, "Teilnehmer (accepted)"
  bekommt zusaetzlich "Ersteller" beim fremden Termin). Zurueckgedreht -> gruen (alle
  security/...-Tests, 79/79 in gdpr), `git diff --stat` zeigt fuer beide Dateien rein
  additive Aenderungen (309 bzw. 353 Zeilen neu, 0 Loeschungen ausser der
  Backlog-Statuszeile).
- verify vorgaenger: sauber. `bbcaf38c` (User-Work-DSAR, Iteration 8) gegen alle acht
  Fehlerklassen geprueft (`git show --stat` und Volltext) — reine additive Query-Funktionen
  im gdpr-Paket (kein gRPC-Layer, kein Stub/TODO), kein `.proto` im Diff, kein neuer
  `RequirePermission`-Guard, keine neue Tabelle/Migration, Wire-Shape
  (DSARModule/DSARRecord) unveraendert, keine neue Route, kein Alt-Guard ersetzt. Der
  waehrend des Bauens gefundene und in derselben Iteration behobene NULL-Scan-Bug
  (`IS NOT DISTINCT FROM`) betraf keinen vorher bestehenden Produktionscode.
- neue-units: keine
- offen:
  - Damit ist Block A der DSAR-Luecken (A3-A9) vollstaendig abgearbeitet — die Auskunft
    fuer Kontakte UND Benutzer deckt jetzt alle Module ab, die die jeweiligen
    Erasure-Handler kennen. Naechste offene Units in Block A sind die Retention-Worker
    (feat-retention-worker-schema-and-engine und ihre vier Folge-Units).
  - `event_exceptions` und `event_reminders` (haengen an `calendar_events` per
    `event_id`) werden von KEINEM Erasure-Handler angefasst und sind auch hier nicht
    disclosed — reine Beobachtung wie die `work/reaction`-Luecke aus Iteration 8, kein
    Befund dieser Unit (Scope war explizit "Tabellenliste deckt sich mit
    CalendarErasureHandler", der beide nicht kennt). Gehoert in einen C-Scan zur
    Erasure-Handler-Vollstaendigkeit, falls einer eingeplant wird.

## Iteration 10 — feat-retention-worker-schema-and-engine — done — 2026-08-22 01:36
- commit: c1edcb15
- gebaut: Migration `000316_create_retention_runs` legt `retention_runs` (ein Lauf) und
  `retention_run_items` (ein Eintrag je Policy) an, beide mit `tenant_id UUID NOT NULL` und
  `CALL enable_tenant_rls(...)`; `run_id` kaskadiert, `policy_id` steht auf ON DELETE SET NULL,
  damit das Protokoll lesbar bleibt, wenn die Policy spaeter geaendert oder geloescht wird.
  `internal/security/gdpr/retention.go` bringt den Motor: `RetentionHandler` (ResourceType,
  Table, DateColumn, SupportsAction, Plan, Apply), `RetentionRegistry` mit EXAKTEM Lookup
  (kein Trim, kein Case-Folding — "Contacts " ist nicht "contacts"), und `RetentionEngine.Run`,
  der die aktivierten Policies des Tenants durchgeht und je Policy eine Zeile schreibt.
  Dry-Run ist der Default und zwar strukturell: `Run` normalisiert JEDEN Wert ausser
  `RetentionModeEnforce` auf `dry_run`, der Nullwert von `RetentionMode` ist also nie scharf.
  `ParseRetentionMode` liefert zusaetzlich ein `ok=false` bei Tippfehlern ("enfroce", "on"),
  damit ein falsch geschriebener Schalter geloggt und nicht stumm entschaerft wird.
  Ein `resource_type` ohne Handler wird als `unmapped` mit Klartext "nicht zugeordnet ..."
  protokolliert (plus `slog.Warn`), ein Handler ohne die verlangte Aktion als `unsupported`,
  ein werfender Handler als `failed` — der Lauf laeuft in allen drei Faellen weiter, weil er
  sonst an der ersten kaputten Policy haengen bleibt und die guten nie erreicht. Das
  Anonymisierungs-Label kommt aus `repo.GetNextAnonymizedLabel` (dieselbe Mechanik wie
  `service.go:314`), einmal je Lauf und nur wenn ein scharfer Lauf es wirklich braucht —
  keine zweite Label-Implementierung. Die Default-Registry ist ABSICHTLICH LEER: welche
  Ressourcen zugeordnet sind, entscheiden die vier Folge-Units; bis dahin erscheint jede
  angelegte Policy ehrlich als "nicht zugeordnet" statt als erfuellte Loeschpflicht.
- gate: build ok | vet ok | lint ok (0 issues, `golangci-lint run ./internal/security/gdpr/...`)
  | test ok (100 PASS / 0 SKIP / 0 FAIL in internal/security/gdpr, DATABASE_URL gegen
  `kmuhub_app`; `./internal/security/...` komplett gruen; `./internal/gateway/` inkl.
  TestOpenAPIRouteDrift gruen, obwohl keine Route beruehrt) | migration ok (000316 up
  angewandt, `down 1` entfernt beide Tabellen wieder — gegen `information_schema.tables`
  auf 0 geprueft — danach erneut `up`; `schema_migrations` = 316, dirty=f)
  | rls-smoke ok (Vorlage aus GATE-COMMANDS.md gegen beide neuen Tabellen: eigener Tenant
  runs 1 / items 1, fremder Tenant runs 0 / items 0; Testzeile danach geloescht, CASCADE
  raeumte das Item mit ab)
- coverage: internal/security/gdpr 67,4 % -> 68,3 % (eigene Messung: Nachher im Arbeitsbaum,
  Vorher per `git worktree add /tmp/covbase HEAD` gegen `210140f4`, danach
  `git worktree remove`; deckt sich mit dem Nachher-Wert aus Iteration 9)
- mutations-probe: in `runPolicy` den Dry-Run-Riegel `if mode != RetentionModeEnforce {
  item.Status = RetentionItemDryRun; return item }` ersatzlos entfernt —
  `TestRetentionEngine_Run_Integration` wird rot mit vier Assertions: `RecordsAffected` 2
  statt 0, Item-Status "applied" statt "dry_run", Item-Affected 2 statt 0, und
  `AssertRowCount` meldet, dass der Probelauf die Benachrichtigung wirklich geloescht hat.
  Zurueckgedreht -> gruen (build, vet, `./internal/security/...`, `./internal/gateway/`);
  `git diff --stat` zeigt ausser der Backlog-Statuszeile nur neue Dateien.
- verify vorgaenger: sauber. `d50754d5` (Kalender-/Notification-DSAR, Iteration 9) gegen alle
  acht Fehlerklassen geprueft: nur `dsar_search.go` + Test + Backlog-Zeile, rein additive
  Query-Funktionen (kein gRPC-Handler, kein Stub/TODO/Unimplemented), kein `.proto`, kein
  neuer `RequirePermission`-Guard und kein ersetzter Alt-Guard, keine neue Tabelle/Migration,
  Wire-Shape (DSARModule/DSARRecord) unveraendert, keine neue Route. Alle sechs neuen SELECTs
  filtern auf `tenant_id = $1 AND user_id = $2` bzw. `created_by/ea.user_id` — keine
  Tenant-Luecke.
- neue-units: keine
- offen:
  - Der Motor ist gebaut, aber NOCH NIRGENDS VERDRAHTET: `NewRetentionEngine` wird von keinem
    Produktionspfad aufgerufen. Das ist so vorgesehen — Ausloeser, Sperre gegen
    Mehrfachstart und Admin-Sicht liegen in `feat-retention-worker-scheduling-and-admin-
    visibility`, die Handler in den drei Handler-Units. Bis dahin aendert dieser Commit am
    Laufzeitverhalten der Anwendung exakt nichts (nur zwei leere Tabellen mehr).
  - Bewusst NICHT gebaut: ein Feld `RetentionMode` in `internal/config`. Der Schalter
    gehoert zur Verdrahtungs-Unit; `ParseRetentionMode` ist die fertige Nahtstelle dafuer.
    Ein jetzt eingefuegtes, von niemandem gelesenes Config-Feld waere toter Code — und ein
    neuer Env-Zwang ist in diesem Lauf ohnehin gesperrt. Wichtig fuer A14: der Default MUSS
    `dry_run` bleiben.
  - Ebenfalls bewusst NICHT gebaut: ein generischer `TableRetentionHandler` (Tabelle +
    Datumsspalte, alles per Konfiguration). Er sieht sparsam aus, ist aber bei `anonymize`
    nicht idempotent zu bekommen — das Label wechselt je Lauf, also kann `Plan` die bereits
    anonymisierten Zeilen nicht wiedererkennen und jeder Lauf schriebe dieselben Zeilen neu.
    Die Idempotenz-Pflicht steht deshalb als Vertrag im Interface-Kommentar; der Testhandler
    ueber `notifications` (nur `delete`) belegt sie mit einem zweiten scharfen Lauf, der 0
    Treffer findet. Die drei Handler-Units brauchen ohnehin je eigene Logik
    (IsInUse, Kaskaden, `closed_at`).
  - `retention_run_items.skip_reasons` ist auf `retentionMaxSkipReasons = 50` Gruende je
    Policy gekappt (`lean:`-Marker im Code); die Zaehler `skipped`/`matched` bleiben exakt.
    Sollte A14 die vollstaendige Liste in der Admin-Sicht brauchen, ist das eine eigene
    Tabelle wert.

## Iteration 11 — feat-retention-worker-handler-contacts — done — 2026-08-22 02:10
- commit: ebc4a43b
- gebaut: `ContactRetentionHandler` implementiert `gdpr.RetentionHandler` fuer
  `resource_type "contacts"` — der erste echte Handler auf dem Motor aus A10.
  Liegt bewusst NICHT in `internal/crm/contact` (wie im Backlog-Scope notiert), sondern in
  `internal/security/gdpr/retention_contacts.go`: `dsar_search_test.go` (Paket `gdpr`,
  interner Testfile) importiert bereits `crm/contact` fuer Fixtures; haette der Handler in
  `crm/contact` gelegen und `security/gdpr` importiert (fuer `RetentionPlan`/`RetentionSkip`),
  waere das ein Importzyklus gewesen (`go build` bestaetigt: "import cycle not allowed in
  test"). Mit dem Handler in `gdpr` importiert nur `gdpr` -> `crm/contact` + `crm/consent`,
  keine Rueckrichtung, kein Zyklus. `RetentionHandler.Plan` hat dafuer eine vierte Signatur-
  Aenderung bekommen (`action string` zusaetzlich zu ctx/tenantID/cutoff) — noetig, weil nur
  bei `action=delete` die zwei RESTRICT-FKs (`dialer_campaign_contacts`, `advisory_protocols`)
  ueberhaupt eine Rolle spielen: `repo.IsInUse` (postgres_repository.go:549, unveraendert
  wiederverwendet) schiebt blockierte Kontakte bei `delete` in `plan.Skipped` mit Klartext-
  Grund statt sie zu werfen; bei `action=anonymize` sind sie ganz normal `Due`, weil
  `consent.AnonymizeContact` die RESTRICT-Tabellen nie anfasst. Zwei Test-Doubles in
  `retention_test.go` (aus Iteration 10) mussten auf die neue Signatur nachgezogen werden.
  Neue Repository-Methode `contact.Repository.ListRetentionCandidates(ctx, tenantID, cutoff)`
  waehlt bewusst NICHT `created_at`, sondern `GREATEST(contacts.updated_at,
  MAX(activities.created_at))` — ein Kontakt mit juengerer Aktivitaet gilt nicht als reif,
  auch wenn er vor Jahren angelegt wurde (Datumswahl wie im Scope gefordert kommentiert und
  per Test belegt: `TestRepository_ListRetentionCandidates` in
  `postgres_repository_db_test.go`). Idempotenz kommt strukturell mit: `AnonymizeContact`
  setzt `updated_at = NOW()`, also faellt ein frisch anonymisierter Kontakt aus dem GREATEST-
  Filter, bis eine volle Aufbewahrungsfrist erneut verstrichen ist — kein zusaetzlicher
  Anonymisiert-Marker noetig. Dafuer wurden `consent.AnonymizedFirstName`/`AnonymizedLastName`
  ("Gelöschte"/"Person") als exportierte Konstanten aus dem bisherigen String-Literal in
  `AnonymizeContact` gezogen (jetzt per Query-Parameter statt String-Konkatenation in SQL),
  damit der Test denselben Wert referenziert statt ihn zu duplizieren.
- gate: build ok (`-p 2` ueber crm/security/server/gateway/cmd) | vet ok | lint ok (0 issues,
  `golangci-lint run` ueber contact, consent, security/gdpr, server) | test ok
  (`internal/crm/...` 12/12 Pakete gruen bei `-p 1` — ein erster `-p 4`-Lauf riss mit
  "too many clients already" ab, reine Verbindungspool-Erschoepfung durch parallele Pakete,
  kein Codefehler, siehe `offen:`; `internal/security/...` 7/7 gruen; `internal/server/` gruen
  inkl. `crm_grpc_fields_tags_contacts_test.go`, wo `stubContactRepo` die neue Interface-
  Methode nachziehen musste; `internal/gateway/` inkl. `TestOpenAPIRouteDrift` gruen, obwohl
  keine Route beruehrt) | migration n.a. (keine neue Tabelle) | rls-smoke n.a. (keine Tabelle/
  Policy angefasst — RLS auf `contacts` bereits aktiv, `ListRetentionCandidates` filtert
  zusaetzlich explizit auf `tenant_id`)
- coverage: internal/security/gdpr 68,3 % -> 68,7 % (eigene Messung: Nachher im Arbeitsbaum,
  Vorher per `git worktree add /tmp/covbase11 HEAD` gegen `982690b0`; deckt sich mit dem
  Endwert aus Iteration 10). internal/crm/contact 80,4 % -> 80,4 % (Netto null: die neue
  `ListRetentionCandidates`-Query fuegt Zeilen hinzu, aber `TestRepository_
  ListRetentionCandidates` deckt sie im selben Paket ab — ohne diesen Test waere der Wert auf
  79,4 % gefallen, weil `go test ./internal/crm/contact/` die Aufrufe aus den gdpr-Tests nicht
  sieht; genau die "falsches Bezugspaket"-Falle aus dem Backlog-Kopf, hier durch einen echten
  Repository-Test statt eines `n.a.` geloest).
- mutations-probe: in `ContactRetentionHandler.Plan` die Bedingung `if inUse {` zu
  `if false && inUse {` verfaelscht — `TestContactRetentionHandler_Plan_DeleteSkipsBlocked`
  wird rot (blockierter Kontakt taucht in `Due` auf, `Skipped` bleibt leer statt 1 Eintrag).
  Zurueckgedreht -> `go build ./internal/...` und der Test wieder gruen; `git diff --stat`
  zeigt nur die beabsichtigten Dateien.
- verify vorgaenger: sauber. `c1edcb15` (Retention-Motor, Iteration 10) gegen alle acht
  Fehlerklassen geprueft: kein gRPC-Handler und damit kein Layer-Bypass, kein Stub/TODO/
  Unimplemented (Grep auf `retention.go` liefert nichts), kein `.proto`, kein neuer
  `RequirePermission`-Guard, Migration 000316 legt `tenant_id UUID NOT NULL` + `enable_
  tenant_rls` auf beiden neuen Tabellen an, keine Wire-Shape-Aenderung (Engine ist noch
  nirgends an eine Route gebunden), keine neue Route.
- neue-units: keine
- offen:
  - Der erste `go test ./internal/crm/...` mit dem Standard-Parallelismus (`-p 4`, mehrere
    Pakete gleichzeitig, jedes mit eigenem `t.Parallel()`-Pool) riss mit "remaining connection
    slots are reserved for roles with the SUPERUSER attribute" ab. Nach Entfernen der
    Baseline-Worktree und einer kurzen Verschnaufpause lief derselbe Lauf mit `-p 1` sauber
    durch. Kein Befund an meinem Code — aber ein Hinweis, dass die lokale Postgres-Instanz
    unter voller Parallelitaet plus einer offenen Zusatz-Worktree an ihre `max_connections`
    stoesst. Falls das oefter auftritt, waere `-p 1` oder ein hoeheres `max_connections` fuer
    den lokalen Compose-Postgres ein Thema fuer Luke, kein Code-Fix.
  - `ContactRetentionHandler` ist gebaut und getestet, aber wie der Motor selbst NOCH NICHT
    verdrahtet: keine Produktionsstelle registriert ihn in einer `RetentionRegistry`. Das ist
    Absicht — die Verdrahtung liegt in `feat-retention-worker-scheduling-and-admin-
    visibility`, die auf `feat-retention-worker-schema-and-engine` deppt, nicht auf diese
    Unit. Bis dahin aendert dieser Commit am Laufzeitverhalten der Anwendung nichts.
  - Bewusst NICHT behoben: `consent.Repository.AnonymizeContact` nimmt kein
    `anonymizedLabel`-Argument, obwohl `RetentionHandler.Apply` eines durchreicht (aus
    `GetNextAnonymizedLabel`, Iteration 10). Der Kontakt-Handler ignoriert es (`_` im
    Funktionskopf) und verlaesst sich auf die fest verdrahteten Platzhalter "Gelöschte"/
    "Person" aus dem bestehenden GDPR-Loeschantrags-Pfad (`consent.Service.ProcessDeletion`).
    Zwei Anonymisierungs-Mechaniken (fester Platzhalter vs. laufender Zaehler) leben damit
    nebeneinander im selben Package — kein Bug, aber ein Normalisierungs-Kandidat fuer einen
    spaeteren C-Scan, falls weitere Retention-Handler denselben Zaehler brauchen.

## Iteration 12 — feat-retention-worker-handler-dialer-chat — done — 2026-08-22 02:08
- commit: 4c0da577
- gebaut: `DialerCallRetentionHandler` und `ChatMessageRetentionHandler`
  (`internal/security/gdpr/retention_dialer_chat.go`), zweiter und dritter Handler an der
  Registry aus A10. Beide arbeiten roh gegen `*pgxpool.Pool`, im selben Stil wie
  `ChatErasureHandler` in `erasure.go` — keine neue Abstraktion, kein Repository-Interface
  angefasst. Dialer: `resource_type="dialer_calls"`, Tabelle `dialer_call_sessions`, Uhr
  `updated_at`; delete raeumt ueber die bestehende `ON DELETE CASCADE` auf
  `dialer_call_events` mit auf, anonymize leert `notes`/`next_action` und laesst Dauer/Outcome/
  Zeitstempel stehen. Chat: `resource_type="chat_messages"`, Tabelle `messages`, Uhr
  `GREATEST(created_at, edited_at)`; anonymize repliziert exakt das Muster aus
  `ChatErasureHandler.ExecuteErasure` (Inhalt -> `[Label]`, `is_deleted=true`,
  `edited_at=NOW()`), delete loescht hart und korrigiert `reply_count` des Elternposts ueber
  ein `RETURNING parent_message_id` plus Best-Effort-Update (Fehlschlag wird geloggt, bricht
  den Lauf nicht ab — dieselbe Nichtfatal-Regel wie im bestehenden manuellen Loeschpfad in
  `message.Service.Delete`).
- gebaut (Recherche-Befund, kein Code): die im Unit-Scope verlangte Pruefung "Datei im
  Objektspeicher" ist negativ — `dialer_call_sessions.call_session_id` zeigt nur AUF
  `call_sessions`, keine Tabelle zeigt zurueck auf `dialer_call_sessions` ausser dem
  CASCADE-Kind `dialer_call_events`. `recordings.call_id` haengt an `call_sessions`, nicht an
  `dialer_call_sessions` — ein geloeschter/anonymisierter Dialer-Call beruehrt also nie eine
  Aufnahme. Aufnahmen haben zudem laengst einen eigenen, unabhaengigen Ablaufmechanismus
  (`recording.Service.CleanupExpiredRecordings`, MinIO-Object-Delete inklusive) — keine zweite
  Unit noetig, kein verwaister Fall gefunden.
- gate: build ok (`-p 2` ueber security/gdpr, gateway, dialer, chat, cmd/gateway) | vet ok |
  lint ok (0 issues, `internal/security/gdpr`) | test ok (`internal/security/gdpr` 8/8 neue
  Tests gruen, ganzes Paket gruen inkl. bestehender Suite, 0 SKIP per `-v | grep -c SKIP`;
  `internal/security/...` 7/7 Pakete gruen; `internal/gateway/` inkl.
  `TestOpenAPIRouteDrift` gruen, obwohl keine Route beruehrt) | migration n.a. (keine neue
  Tabelle/Policy — beide Handler laufen gegen bestehende, bereits RLS-gesicherte Tabellen) |
  rls-smoke n.a. (RLS auf `dialer_call_sessions` seit Migration 000120, auf `messages` seit
  000253 aktiv; beide Handler filtern zusaetzlich explizit auf `tenant_id`, wie
  `ContactRetentionHandler`)
- coverage: internal/security/gdpr 68,7 % -> 69,0 % (eigene Messung: Nachher im Arbeitsbaum,
  Vorher per `git worktree add /tmp/covbase12 ebc4a43b`)
- mutations-probe: in `ChatMessageRetentionHandler.Plan` `GREATEST(created_at, ...)` durch
  `created_at` ersetzt (Edited-Ausschluss deaktiviert) —
  `TestChatMessageRetentionHandler_PlanExcludesFreshAndEditedMessages` wird rot (die kuerzlich
  bearbeitete, aber alte Nachricht taucht faelschlich in `Due` auf). Zurueckgedreht -> Test
  wieder gruen, `git diff --stat` zeigt nur die beabsichtigten Dateien.
- verify vorgaenger: sauber. `ebc4a43b` (Iteration 11, ContactRetentionHandler) gegen alle acht
  Fehlerklassen geprueft: kein gRPC-Handler/Layer-Bypass (Handler liegt in security/gdpr, ist
  nirgends an eine Route gebunden), kein Stub/TODO/Unimplemented, kein `.proto`, kein neuer
  `RequirePermission`-Guard, keine neue Tabelle also kein Tenant-/RLS-Thema, keine
  Wire-Shape-Aenderung (kein Response-Pfad), keine neue Route, kein Guard ersetzt.
- neue-units: keine
- offen:
  - Wie schon bei A11 (Contacts): `DialerCallRetentionHandler` und
    `ChatMessageRetentionHandler` sind gebaut und getestet, aber noch NICHT in einer
    `RetentionRegistry` produktiv registriert — das ist Absicht und liegt bei
    `feat-retention-worker-scheduling-and-admin-visibility`. Bis dahin aendert dieser Commit
    am Laufzeitverhalten nichts.
  - Nicht behoben, weil ausserhalb des Scopes: `message.Repository.Delete` (der bestehende
    manuelle Soft-Delete-Pfad) setzt `is_deleted=true`, ohne `content` zu anonymisieren — eine
    manuell "geloeschte" Nachricht traegt ihren Klartext also unveraendert weiter in der DB.
    Kein Bug in dieser Unit (mein Handler geht ueber `GREATEST(created_at, edited_at)`, nicht
    ueber `is_deleted`, und erreicht solche Zeilen deshalb trotzdem), aber ein Normalisierungs-
    Kandidat: zwei verschiedene Bedeutungen von "geloescht" (Flag ohne Inhaltsloeschung vs.
    Flag mit Inhaltsloeschung) leben im selben Feld.
