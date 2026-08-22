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

## Iteration 13 — feat-retention-worker-handler-helpdesk-formulare — done — 2026-08-22 02:19
- commit: 539d6ada
- gebaut: `HelpdeskTicketRetentionHandler` und `FormSubmissionRetentionHandler`
  (`internal/security/gdpr/retention_helpdesk_formulare.go`), vierter und fuenfter Handler an
  der Registry aus A10. Tickets: `resource_type="helpdesk_tickets"`, Tabelle `tickets`, Scope
  NUR `status='closed'`, Uhr `resolved_at` — `CloseTicket` (service.go:337) ist der einzige Pfad,
  der `status` je auf `closed` setzt, und stempelt in derselben Transaktion `resolved_at`;
  `UpdateTicket`s generischer Feld-Patch verweigert das Setzen von `status=closed` explizit
  (service.go:294), also gibt es keinen zweiten Weg, der `resolved_at` vergessen koennte. Delete
  verlaesst sich auf das bestehende `ON DELETE CASCADE` von `ticket_messages` (Migration 000077,
  live per `pg_constraint` gegengeprueft: `confdeltype='c'`) und muss die Nachrichten nicht
  selbst anfassen. Anonymize dagegen MUSS: es loescht die Ticket-Zeile nicht, also werden
  `subject`/`description` sowie `requester_name`/`requester_email` (Label statt NULL, weil
  `chk_tickets_requester_identity` aus Migration 000291 eine leere E-Mail bei externen
  Anfragenden verbietet) geleert UND alle `ticket_messages.body` des Tickets ueberschrieben —
  unabhaengig vom `internal`-Flag, denn das regelt nur die Sichtbarkeit in einer DSAR-Auskunft
  (A6), nicht die Aufbewahrung. Idempotenz ueber `updated_at`, das Anonymize mitbumpt (gleiches
  Muster wie A12 bei `dialer_call_sessions.updated_at`).
  Formulare: `resource_type="form_submissions"`, Tabelle `form_submissions`, Uhr `submitted_at`.
  Einreichungen ohne Kontaktbezug werden erfasst (kein Join, kein Filter auf `form_schema_id`
  oder eine Kontakt-Verknuepfung — die existiert fuer `form_submissions` ohnehin nicht). Anders
  als bei allen bisherigen Handlern gibt es kein `updated_at` zum Bumpen, `submitted_at` darf
  als historische Tatsache nicht veraendert werden — deshalb neue Spalte `anonymized_at`
  (Migration 000317, nullable, keine RLS-Aenderung noetig: `form_submissions` hatte RLS schon,
  `relrowsecurity`/`relforcerowsecurity` nach dem `ALTER TABLE` gegengeprueft = weiterhin `t`/`t`).
  Plan filtert bei `action=anonymize` zusaetzlich auf `anonymized_at IS NULL`, bei `action=delete`
  nicht (eine bereits anonymisierte, aber noch nicht geloeschte Zeile muss ein Delete-Lauf
  weiterhin finden).
- gate: build ok (`-p 2` ueber security/..., helpdesk/..., formulare/..., gateway/...,
  cmd/gateway/...) | vet ok | lint ok (0 issues, `internal/security/gdpr`) | test ok
  (`internal/security/gdpr` 8 neue Tests gruen, ganzes Paket 103 PASS / 0 SKIP / 0 FAIL;
  `internal/helpdesk/...` gruen; `internal/formulare/...` gruen; `internal/security/...` alle
  Unterpakete gruen; `internal/gateway/` inkl. `TestOpenAPIRouteDrift` gruen, obwohl keine Route
  beruehrt) | migration ok (000317, lokal angewendet via `migrate ... up`, down enthaelt
  `DROP COLUMN IF EXISTS`) | rls-smoke: kein neuer Policy-Bedarf (bestehende Tabelle, nur Spalte
  hinzugefuegt), `relrowsecurity`/`relforcerowsecurity` nach der Migration weiterhin `t`/`t`
  gegengeprueft
- coverage: internal/security/gdpr 69,0 % -> 69,1 % (eigene Messung: Vorher per
  `git worktree add /tmp/covbase13 4c0da577`, Nachher im Arbeitsbaum)
- mutations-probe: in `FormSubmissionRetentionHandler.Plan` das `anonymized_at IS NULL`-Gate mit
  `if false && action == ...` deaktiviert —
  `TestFormSubmissionRetentionHandler_ApplyAnonymizeIsIdempotentViaDedicatedColumn` wird rot (die
  bereits anonymisierte Einreichung taucht faelschlich wieder in `Due` auf). Zurueckgedreht ->
  `go test ./internal/security/gdpr/` wieder gruen, `git status --short` zeigt nur die vier
  beabsichtigten neuen Dateien (Handler, Test, zwei Migrationsdateien) plus die Backlog-Zeile.
- verify vorgaenger: sauber. `4c0da577` (Iteration 12, Dialer/Chat-Handler) gegen alle acht
  Fehlerklassen geprueft: kein gRPC-Handler/Layer-Bypass (Handler liegt in security/gdpr, ist
  nirgends an eine Route gebunden), kein Stub/TODO/Unimplemented, kein `.proto`, kein neuer
  `RequirePermission`-Guard, keine neue Tabelle also kein Tenant-/RLS-Thema, keine
  Wire-Shape-Aenderung (kein Response-Pfad), keine neue Route, kein Guard ersetzt. Der im
  Commit dokumentierte Objektspeicher-Befund (keine Aufnahme-Verwaisung moeglich) wurde am
  Schema gegengeprueft und ist korrekt.
- neue-units: keine
- offen:
  - Wie A11/A12: `HelpdeskTicketRetentionHandler` und `FormSubmissionRetentionHandler` sind
    gebaut und getestet, aber noch NICHT in einer `RetentionRegistry` produktiv registriert —
    Absicht, liegt bei `feat-retention-worker-scheduling-and-admin-visibility`. Bis dahin aendert
    dieser Commit am Laufzeitverhalten nichts.
  - Nicht behoben, ausserhalb des Scopes: `UpdateTicket`s generischer Feld-Patch kann ein
    bereits `closed` Ticket weiterhin editieren (Assignee, Prioritaet etc.), solange `statusVal`
    dabei nicht gesetzt wird — das bumpt `updated_at`, ohne `resolved_at` zu beruehren. Fuer
    Delete ist das folgenlos (kein Datumsfilter auf `updated_at`), fuer Anonymize bedeutet es,
    dass eine solche nachtraegliche Bearbeitung eines laengst geschlossenen Tickets dessen
    Anonymisierung um eine volle Aufbewahrungsfrist verzoegert — plausibel und im Sinne von
    "noch in Bearbeitung", aber nicht explizit von der Fachseite bestaetigt.
  - Ein per `ReopenTicket` wiedereroeffnetes und danach erneut geschlossenes Ticket startet
    seine Frist automatisch neu (Status faellt bei `Plan` heraus, solange es nicht wieder
    `closed` ist) — beabsichtigtes Verhalten, aber im Journal festgehalten, falls es spaeter
    anders erwartet wird.

## Iteration 14 — feat-retention-worker-scheduling-and-admin-visibility — done — 2026-08-22 02:29
- commit: 3f97bc8b
- gebaut: Trigger- und Sicht-Haelfte von A10-A13, die den Motor tatsaechlich laufen laesst.
  `internal/security/gdpr/retention_scheduler.go`: `RunScheduledRetention` haelt den
  `pg_try_advisory_lock`-Schluessel `0x52544E4E` ("RTNN", distinct von
  `idempotencyCleanupLockKey`=0x49444D50), zaehlt bei Erfolg per `listTenantIDs` (`SELECT id FROM
  tenants`) alle Tenants und laesst `runForTenants` die Engine je Tenant unter
  `sysctx.With(ctx) + context.WithValue(ctx, middleware.TenantIDKey, tenantID.String())` laufen
  (RLS-Bypass zum Lesen der Tenant-Liste, aber die Engine selbst filtert weiterhin explizit auf
  ihre Tenant-ID). `RetentionScheduler` ist der duenne Ticker-Wrapper (1 Min Startup-Delay, danach
  alle 24 h, Vorlage `cmd/work/main.go`s Recording-Cleanup-Goroutine). `cmd/auth/main.go` registriert
  die Registry mit allen fuenf bisherigen Handlern (Contacts, Dialer, Chat, Helpdesk, Formulare),
  liest `RETENTION_MODE` per `os.Getenv` (deliberately kein `config.RequireX` — Default bleibt
  `dry_run` bei fehlendem/falsch geschriebenem Wert, `ParseRetentionMode` gibt das schon so vor)
  und startet die Scheduler-Goroutine. Damit aendert dieser Commit — anders als A11/A12/A13 —
  tatsaechlich Laufzeitverhalten: der Motor laeuft ab diesem Deploy taeglich, im Dry-Run-Default.
  Admin-Sicht: neue RPC `GetLatestRetentionRun` in `security.proto` (regeneriert), liest
  `retention_runs`/`retention_run_items` (Migration 000316) direkt per `s.pool`-Query ohne
  Engine-Referenz — reiner Leser, kein Trigger-Endpunkt (der Lauf startet aus der Scheduler-
  Goroutine, nicht per Admin-Klick, wie in den `notes` der Unit gefordert). Route
  `GET /api/v1/security/retention-runs/latest` (admin-only, gleiche Rolle wie die bestehenden
  Retention-CRUD-Routen), Antwort `{has_run, run, items[]}` — `has_run=false` unterscheidet "nie
  gelaufen" von einem echten Lauf mit null Treffern.
- gate: build ok (`-p 2` ueber security/..., server/..., gateway/..., cmd/auth/..., cmd/gateway/...,
  danach volles `go build ./...` gruen) | vet ok | lint ok (0 issues) | test ok
  (`internal/security/gdpr` alle Tests gruen inkl. 4 neue; `internal/security/...` alle
  Unterpakete gruen; `internal/server` gruen inkl. 3 neue DB-Tests; `internal/gateway` gruen
  inkl. `TestOpenAPIRouteDrift` — 836 registrierte gegen 838 dokumentierte Pfade, meine neue Route
  ist in beiden Zahlen enthalten) | migration n.a. (keine neue Tabelle/Spalte, `retention_runs`/
  `retention_run_items` bestehen bereits seit 000316) | rls-smoke: kein neuer Policy-Bedarf; die
  Tenant-Isolation der neuen RPC ist stattdessen per DB-Test bewiesen
  (`TestSecurityGRPCServer_GetLatestRetentionRun_TenantIsolation`: ein fremder Tenant mit echtem
  `retention_runs`-Eintrag liefert `has_run=false` fuer den eigenen Tenant-Kontext)
- coverage: internal/security/gdpr 69,1 % -> 68,3 % (eigene Messung: Vorher per
  `git worktree add /tmp/covbase14 539d6ada`, Nachher im Arbeitsbaum — der Ruckgang ist real und
  erklaerbar, nicht gemessen falsch: `RetentionScheduler` selbst, der Ticker-Wrapper mit
  Startup-Delay/Select-Loop, ist bewusst ungetestet (Vorlage `IdempotencyCleanupWorker`, die
  ebenfalls keinen eigenen Test hat) und zieht den Paketschnitt herunter, obwohl die drei
  darunterliegenden, testbaren Funktionen (`RunScheduledRetention`, `listTenantIDs`,
  `runForTenants`) alle abgedeckt sind); internal/server 70,2 % -> 70,3 %; internal/gateway
  46,1 % -> 46,1 % (Route + ein ServiceUnavailable-Test sind zu klein, um den Paketschnitt zu
  bewegen)
- mutations-probe: in `RunScheduledRetention` das Lock-Gate von `if !locked` auf
  `if false && !locked` verstellt (die Guard-Bedingung faktisch deaktiviert, sodass ein Aufruf mit
  fremd gehaltenem Lock trotzdem durchlaeuft) — `TestRunScheduledRetention_SkipsWhenLockHeldElsewhere`
  wird rot, UND die Mutation demonstriert live den Schaden, den die Probe verhindern soll: der
  mutierte Lauf iterierte tatsaechlich ueber alle ~13.800 Tenants der lokalen Dev-DB und schrieb je
  einen `retention_runs`-Eintrag, obwohl das Lock extern gehalten wurde (siehe naechster Absatz).
  Zurueckgedreht -> Test wieder gruen, `git status --short` zeigt nur die beabsichtigten Dateien.
- verify vorgaenger: sauber. `539d6ada` (Iteration 13, Helpdesk/Formulare-Handler) gegen alle acht
  Fehlerklassen geprueft: kein gRPC-Layer-Bypass (Handler liegen in security/gdpr, nirgends an
  eine Route gebunden), kein Stub/TODO/Unimplemented, kein `.proto` in diesem Commit, kein neuer
  `RequirePermission`-Guard, Migration 000317 ist ein reines `ALTER TABLE ... ADD COLUMN` auf einer
  bestehenden RLS-Tabelle (kein neues Tenant-/RLS-Thema, `relrowsecurity`/`relforcerowsecurity`
  im Commit selbst gegengeprueft), keine Wire-Shape-Aenderung (kein Response-Pfad beruehrt), keine
  neue Route, kein Guard ersetzt.
- neue-units: keine (siehe unten — ein echter Befund, aber kein Code-Bug, den eine Unit fixen
  koennte)
- offen:
  - **Fund, kein Code-Bug:** die lokale Dev-Postgres (`docker-postgres-1`) trug zu Beginn dieser
    Iteration **13.811 Zeilen in `tenants`**. Mein erster Testentwurf (ein direkter Aufruf von
    `RunScheduledRetention` bei freiem Lock, seither wieder entfernt) lief deshalb 70 s und schrieb
    ueber jeden dieser Tenants einen `retention_runs`-Eintrag — die Mutations-Probe oben hat das
    Muster nochmal reproduziert. Beide Male habe ich die selbst erzeugten `retention_runs`-Zeilen
    per `DELETE ... WHERE triggered_by='schedule' AND started_at > NOW() - INTERVAL '1 hour'`
    wieder entfernt (`retention_runs` stand am Ende der Iteration wieder bei 0 durch mich erzeugten
    Zeilen). Die 13.811 `tenants`-Zeilen selbst habe ich NICHT angefasst — das ist Debris aus
    frueheren Nachtlaeufen (vermutlich Tests, deren `defer testutil.CleanupRow` durch einen Absturz
    oder ein Timeout nie erreicht wurde, ueber viele Iterationen aufsummiert), kein Produktbug und
    ausserhalb des Scopes dieser Unit. Für den `RunScheduledRetention`-Erfolgspfad selbst folgt
    daraus aber ein bewusster Test-Verzicht (siehe coverage-Absatz und Kommentar in
    `retention_scheduler_test.go`) — bevor irgendein weiterer Job nach demselben "for jeden Tenant"-
    Muster gebaut wird, sollte die lokale `tenants`-Tabelle aufgeraeumt werden, sonst wird jeder
    solche Test/Job lokal unbrauchbar langsam.
  - `RetentionScheduler` startet ab diesem Commit produktiv beim naechsten `cmd/auth`-Deploy, im
    Dry-Run-Default (`RETENTION_MODE` ist in keiner `.env` gesetzt, `ParseRetentionMode("")` ->
    `dry_run, true`). Scharfschalten ist eine bewusste, separate Entscheidung ueber
    `RETENTION_MODE=enforce` und explizit NICHT Teil dieses Commits.
  - `GetLatestRetentionRun` zeigt nur den zuletzt gestarteten Lauf. Mehrere Policies mit
    unterschiedlichen `resource_type` erscheinen darin als `items[]` des EINEN Laufs (ein Run deckt
    alle enabled Policies eines Tenants ab), nicht als eigene Läufe — das entspricht dem
    bestehenden `RetentionEngine.Run`-Modell aus A10 und ist keine neue Verhaltensannahme dieser
    Unit.

## Iteration 15 — cov-gateway-biz-gobd-archive — done — 2026-08-22 02:52
- commit: 27225234
- gebaut: `route_biz_gobd_archive_test.go` (26 neue Tests) fuer alle 7 Funktionen von
  `route_biz_gobd_archive.go` (`HandleArchiveDocument`, `HandleArchiveInvoiceDocument`,
  `HandleListGobdDocuments`, `HandleGetGobdDocument`, `HandleDownloadGobdDocument`,
  `HandleAddDocumentAnnotation`, `parsePageParam`). Deckt ab: ServiceUnavailable (503),
  fehlende TenantID (401), kaputtes Multipart (400), fehlendes `file`-/`doc_type`-Feld (400),
  Ueberschreitung von `maxGobdDocumentBytes` um genau 1 Byte -> 413 (echter 50 MiB+1-Upload,
  kein verkuerzter Test), leere/zu lange Annotation (`decodeAndValidate`, Feldname `note` aus
  dem JSON-Tag, nicht `Note`), gueltige Requests bis zur RPC-Schicht (503 gegen Dummy-Adresse)
  sowie zwei Permission-Guard-Tests ueber einen echten `chi.Router` (`gobdArchiveRouter`,
  Vorlage `route_datev_upload_test.go`): `finance:write` allein oeffnet `HandleArchiveDocument`
  NICHT, nur `gobd-archive:write` tut es; `gobd-archive:read` oeffnet `HandleAddDocumentAnnotation`
  NICHT, nur `gobd-archive:write`. `multipartBody` (aus `route_crm_contacts_test.go`) und
  `withPermissions`/`guardTestAuth` (aus `route_capability_guard_test.go`) wiederverwendet, keine
  Duplikate angelegt.
- gate: build ok (`-p 2` ueber gateway/..., cmd/gateway/...) | vet ok | lint ok (0 issues) |
  test ok (ganzes `internal/gateway` gruen inkl. `TestOpenAPIRouteDrift` — keine neue Route in
  dieser Unit, nur Tests)
- coverage: internal/gateway 46,1 % -> 46,6 % (eigene Messung: Vorher per Wegverschieben der
  neuen Testdatei und erneutem `go test -coverprofile`, Nachher mit der Datei; deckt sich mit dem
  `coverage_start` der Unit)
- mutations-probe: in `HandleArchiveDocument` die Groessenpruefung `if len(fileBytes) >
  maxGobdDocumentBytes` auf `if false && len(fileBytes) > maxGobdDocumentBytes` verstellt (die
  413-Guard-Bedingung faktisch deaktiviert) — `TestHandleArchiveDocument_ExceedsSizeLimit` wird
  rot (503 statt 413, RPC-Layer erreicht statt Guard). Zurueckgedreht -> Test wieder gruen,
  `git diff --stat` auf `route_biz_gobd_archive.go` zeigt keine Aenderung mehr.
- verify vorgaenger: sauber. `3f97bc8b` (Iteration 14, Retention-Scheduler + Admin-RPC) gegen
  alle acht Fehlerklassen geprueft: gRPC-Layer korrekt ueber `client.GetLatestRetentionRun`
  (kein direkter Service-Aufruf), kein Stub/TODO, `.proto` geaendert UND `security.pb.go`/
  `security_grpc.pb.go` im selben Commit regeneriert, keine neue `RequirePermission`
  (`RequireRole("admin")`, konsistent mit den Nachbarrouten in derselben Gruppe, kein Seed
  noetig), keine neue Tabelle (liest nur `retention_runs`/`retention_run_items` aus 000316),
  RLS-Scoping der neuen RPC per dediziertem `TestSecurityGRPCServer_GetLatestRetentionRun_
  TenantIsolation`-DB-Test bewiesen (fremder Tenant -> `has_run=false`), keine Wire-Shape-
  Aenderung an bestehenden Routen, kein Guard ersetzt.
- neue-units: keine
- offen:
  - `TestHandleArchiveDocument_ExceedsSizeLimit` allokiert einen echten ~50 MiB Byte-Slice pro
    Testlauf (`bytes.Repeat`) — lief lokal in ~0,1 s, aber falls das Paket kuenftig deutlich
    langsamer wird, ist das der erste Kandidat zum Nachsehen.
  - "Fremder Tenant liefert 404" aus den harten Regeln fuer Block B ist an dieser Route NICHT
    ueber einen Live-RPC-Roundtrip getestet — dieses Paket hat wie in jeder vorigen
    Coverage-Unit dieses Laufs keinen bufconn-/Fake-Client-Harness fuer
    `FinanceServiceClient` (bestaetigt: `grep -rn "bufconn\." internal/gateway` liefert 0
    Treffer im ganzen Paket, nur Kommentare erwaehnen die fehlende Infrastruktur). Der Handler
    liest die TenantID ausschliesslich aus `getTenantID(r)` (JWT-Context), es gibt keinen
    Code-Pfad, der einen Client-seitig gelieferten Tenant-Wert uebernehmen wuerde — die
    eigentliche Tenant-Durchsetzung passiert serverseitig in `internal/biz/gobdarchive` und ist
    dort Sache der jeweiligen Coverage-Unit, nicht dieser Route-Unit.

## Iteration 16 — cov-gateway-biz-einvoice — done — 2026-08-22 03:07
- commit: 4fabe7e3
- gebaut: `route_biz_einvoice_test.go` (29 neue Tests) fuer alle 5 Funktionen von
  `route_biz_einvoice.go` (`HandleImportInvoice`, `HandleListIncomingInvoices`,
  `HandleGetIncomingInvoice`, `HandleUpdateIncomingInvoiceStatus`, `isEInvoiceMIME`). Deckt ab:
  ServiceUnavailable (503), fehlende TenantID (401), kaputtes Multipart (400), fehlendes
  `file`-Feld (400), abgelehnter MIME-Typ ueber einen echten Multipart-Part mit expliziten
  Content-Type-Header `image/png` -> 415 (dafuer `multipartBodyWithFileContentType` als lokaler
  Helper ergaenzt, weil `multipartBody`/`CreateFormFile` den Datei-Part immer hart auf
  `application/octet-stream` setzt und damit den MIME-Reject-Pfad ueber HTTP nie erreicht haette),
  leere Datei (400 "uploaded file is empty"), Ueberschreitung von `maxEInvoiceUploadBytes` um
  genau 1 Byte -> 413 (echter 10 MiB+1-Upload), kaputtes/ungueltiges JSON und den
  `oneof=reviewed booked rejected`-Constraint auf `status` (400 Validation), gueltige Requests
  bis zur RPC-Schicht (503 gegen Dummy-Adresse) sowie drei Permission-Guard-Tests ueber einen
  echten `chi.Router`: Import und Status-Update verlangen `finance:write`, Liste verlangt
  `finance:read` — beide bleiben laut `route_biz.go:50-57` bewusst auf dem alten grob-koernigen
  `finance`-Guard und NICHT auf dem gesplitteten `finance:invoice`-Catalogue-Key. Isolierter
  Tabellentest fuer `isEInvoiceMIME` inkl. Gross-/Kleinschreibungs-Grenzfall (kein Case-Folding).
  `gobdArchiveRouter` aus Iteration 15 in `bizRouter` umbenannt (mountet ohnehin alle
  Finance-Routen, nicht nur gobd-archive) und in beiden Testdateien wiederverwendet statt einen
  zweiten, identischen Router-Helper anzulegen.
- gate: build ok (`-p 2` ueber gateway/..., cmd/gateway/...) | vet ok | lint ok (0 issues) |
  test ok (ganzes `internal/gateway` gruen inkl. `TestOpenAPIRouteDrift` — keine neue Route,
  nur Tests)
- coverage: internal/gateway 46,6 % -> 46,9 % (eigene Messung: Vorher per Wegverschieben der
  neuen Testdatei und erneutem `go test -coverprofile`, Nachher mit der Datei; 46,6 % ist der
  reale lokale Stand nach Iteration 15, nicht der `coverage_start` der Unit aus dem CI-Snapshot
  1b49a1f3 — die beiden weichen um 0,5 Punkte voneinander ab, weil Iteration 15 dasselbe Paket
  bereits angehoben hat)
- mutations-probe: in `HandleImportInvoice` die MIME-Pruefung `if !isEInvoiceMIME(mimeType)` auf
  `if false && !isEInvoiceMIME(mimeType)` verstellt (den 415-Guard faktisch deaktiviert) —
  `TestHandleImportInvoice_UnsupportedMIMEType` wird rot (503 statt 415, RPC-Layer erreicht statt
  Guard). Zurueckgedreht -> Test wieder gruen, `git diff --stat` auf `route_biz_einvoice.go`
  zeigt keine Aenderung mehr.
- verify vorgaenger: sauber. `27225234` (Iteration 15, GoBD-Belegarchiv-Route-Tests) gegen alle
  acht Fehlerklassen geprueft: reine Testdatei, kein Produktionscode geaendert, kein
  gRPC-Layer-Bypass (Tests rufen ausschliesslich `client.<RPC>` ueber die Handler auf), kein
  Stub/TODO, kein `.proto` angefasst, keine neue `RequirePermission` (die verwendeten
  `gobd-archive:write`/`:read`-Keys sind seit Migration 000139 bestehende Guards in
  `route_biz.go:291-296`, kein neuer Seed noetig, per `grep -rn "gobd-archive" backend/migrations`
  bestaetigt), keine neue Tabelle, keine Wire-Shape-Aenderung, kein Guard ersetzt.
- neue-units: keine
- offen:
  - `TestHandleImportInvoice_ExceedsSizeLimit` allokiert einen echten ~10 MiB Byte-Slice pro
    Testlauf (`bytes.Repeat`), analog zum 50-MiB-Fall aus Iteration 15 — lief lokal in Bruchteilen
    einer Sekunde, gleicher Beobachtungspunkt wie dort.
  - "Fremder Tenant liefert 404" ist an dieser Route wie in jeder vorigen Coverage-Unit dieses
    Laufs NICHT ueber einen Live-RPC-Roundtrip getestet (kein bufconn-/Fake-Client-Harness fuer
    `FinanceServiceClient` im ganzen Paket). Der Handler liest die TenantID ausschliesslich aus
    `getTenantID(r)` (JWT-Context) — kein Code-Pfad uebernimmt einen client-seitig gelieferten
    Tenant-Wert; die serverseitige Durchsetzung liegt bei `internal/biz/einvoice`, ausserhalb des
    Scopes dieser Route-Unit.
  - `route_biz.go:50-57` dokumentiert bereits, dass Invoice-Import/Incoming-Invoices bewusst auf
    dem alten `finance`-Guard bleiben statt auf den gesplitteten `finance:invoice`-Key zu wechseln
    (kein FE-Caller heute) — bestaetigt beim Bauen dieser Unit, kein neuer Befund.

## Iteration 17 — cov-gateway-biz-quotes — done — 2026-08-22 03:07
- commit: 5f8c9871
- gebaut: `route_biz_quotes_test.go` (neue Datei, 34 Tests) deckt alle 11 Funktionen von
  `route_biz_quotes.go` ab (`HandleCreateQuote`, `HandleListQuotes`, `HandleGetQuote`,
  `HandleUpdateQuote`, `HandleDeleteQuote`, `HandleSendQuote`, `HandleAcceptQuote`,
  `HandleRejectQuote`, `HandleConvertQuoteToInvoice`, `HandleGenerateQuotePDF`,
  `HandleCreateQuoteFromDeal`). Fuenf der elf Funktionen hatten bereits Teilabdeckung in
  `route_biz_test.go` (Create/List/Get: ServiceUnavailable, InvalidJSON, MissingCustomer,
  InvalidTaxMode, InvalidValidUntil) — diese Unit ergaenzt dort nur die Luecken (NoTenantID,
  MissingLineItems, InvalidCustomerVAT, ReachesRPC) statt zu duplizieren, und baut Update,
  Delete, Send, Accept, Reject, Convert, PDF und CreateQuoteFromDeal komplett neu.
  Permission-Guard-Wiring fuer die ganze Quotes-Gruppe (inkl. additivem Legacy-vs-
  Catalogue-Key-Rollout: finance:write/finance:quote:create, finance:quote:send,
  finance:invoice:create fuers Convert, finance:delete-only fuers Delete) ist bereits
  vollstaendig in `route_capability_guard_test.go` ("--- finance: quotes ---", 12 Faelle)
  getestet und wird hier bewusst NICHT dupliziert.
  Statusuebergaenge (Send/Accept/Reject/Convert): dieses Paket hat wie in jeder vorigen
  Coverage-Unit dieses Laufs keinen bufconn-/Fake-Client-Harness fuer FinanceServiceClient
  (bestaetigt, gleiche Grenze wie in Iteration 15/16 sowie in `route_produktion_orders_test.go`
  und `route_rapporte_test.go` aus fruaeheren Laeufen). Die *_ReachesRPC-Tests beweisen nur,
  dass der Handler mit gueltiger Quote-ID die RPC-Schicht erreicht — die eigentliche
  Transitions-Ablehnung (draft->sent->accepted/rejected) ist an ZWEI tieferen Stellen bereits
  bewiesen: `internal/biz/quote/service_test.go` (`TestService_Accept_RejectsNonSent`,
  `TestService_Reject_RejectsNonSent`, `TestService_Send_RejectsNonDraft`,
  `TestService_Accept_RejectsAlreadyAccepted`, ...) fuer die Service-Logik, und
  `internal/server/biz_grpc_errormap_settings_quotes_test.go` fuer die Zuordnung
  `quote.ErrQuoteNotDraft`/`ErrQuoteNotSent` -> `codes.FailedPrecondition`. Die generische
  `FailedPrecondition -> 409`-HTTP-Abbildung ist in `helpers_test.go`
  (`TestGrpcStatusToHTTP`) bewiesen. Diese drei Tests zusammen belegen den vollen Pfad
  "falscher Statusuebergang -> 409", nur nicht als ein einzelner Live-RPC-Roundtrip in
  diesem Paket — dieselbe dokumentierte Infrastrukturluecke wie in jeder vorigen
  Block-B-Unit dieses Laufs.
- gate: build ok (`-p 2` ueber gateway/..., cmd/gateway/...) | vet ok | lint ok (0 issues) |
  test ok (ganzes `internal/gateway` gruen inkl. `TestOpenAPIRouteDrift` — 836 Routen gegen
  838 dokumentierte Pfade, keine neue Route) | `internal/biz/quote` gruen (bestehende Tests
  unveraendert, keine neue Testdatei dort — die Statusuebergaenge sind bereits ausfuehrlich
  in `service_test.go` abgedeckt, siehe oben)
- coverage: internal/gateway 46,9 % -> 47,4 % (eigene Messung: Testdatei nach `/tmp`
  verschoben, `go test -coverprofile` vorher/nachher; 46,9 % ist der reale lokale Stand nach
  Iteration 16, nicht der `coverage_start` der Unit aus dem CI-Snapshot 1b49a1f3 — 0,3 Punkte
  Abweichung, weil zwei Iterationen dasselbe Paket bereits angehoben haben)
- mutations-probe: in `updateQuoteRequest.TaxMode` das
  `validate:"omitempty,oneof=standard reverse_charge kleinunternehmer"`-Tag entfernt ->
  `TestHandleUpdateQuote_InvalidTaxMode` wird rot (503 statt 400, RPC-Layer erreicht statt
  Validierungs-Guard). Zurueckgedreht -> Test wieder gruen, `git diff --stat` auf
  `route_biz_quotes.go` zeigt keine Aenderung mehr.
- verify vorgaenger: sauber. `4fabe7e3` (Iteration 16, E-Invoice-Route-Tests) gegen alle acht
  Fehlerklassen geprueft: reine Testdatei (`route_biz_einvoice_test.go` neu, Rename
  `gobdArchiveRouter` -> `bizRouter` in `route_biz_gobd_archive_test.go`), kein
  Produktionscode geaendert, kein gRPC-Layer-Bypass (Tests rufen ausschliesslich Handler auf,
  die ueber `client.<RPC>` gehen), kein Stub/TODO, kein `.proto` angefasst, keine neue
  `RequirePermission` (die verwendeten `finance:write`/`:read`-Keys sind bestehende Guards aus
  `route_biz.go:50-57`), keine neue Tabelle, keine Wire-Shape-Aenderung, kein Guard ersetzt.
  Die Journal-SHA fuer Iteration 16 fehlte noch ("folgt im naechsten Schritt") und wurde vor
  dieser Unit in einem separaten docs-Commit nachgetragen (`5bf46c6a`).
- neue-units: keine
- offen:
  - Wie in Iteration 15/16: "fremder Tenant liefert 404" ist an dieser Route NICHT ueber
    einen Live-RPC-Roundtrip getestet (kein bufconn-/Fake-Client-Harness fuer
    `FinanceServiceClient` im ganzen Paket). Die serverseitige Tenant-Durchsetzung liegt bei
    `internal/biz/quote` (WHERE tenant_id = $1), ausserhalb des Scopes dieser Route-Unit;
    `TestFinanceLineItems_JSONBTenantIsolation` in `route_biz_test.go` deckt bereits, dass
    zwei verschiedene Tenants unabhaengige, nicht-401 Anfragen erzeugen.
  - `internal/biz/quote` liegt weiterhin bei 33,3 % (CI-Snapshot) trotz sehr grosser
    `service_test.go` — die Luecke ist vermutlich im Repository-Layer
    (`postgres_repository.go`, DB-Tests) statt in der Service-Logik. Nicht Teil dieser Unit
    (Scope war die Gateway-Route), aber ein Kandidat fuer eine eigene DB-Coverage-Unit, falls
    Block E das Paket noch einmal aufgreift.

## Iteration 18 — cov-gateway-biz-recurring-invoices — done — 2026-08-22 03:14
- commit: 791ba2bf
- gebaut: `route_biz_recurring_test.go` (neue Datei, 30 Tests) deckt alle 9 Funktionen von
  `route_biz_recurring.go` ab (`HandleListRecurringInvoices`, `HandleCreateRecurringInvoice`,
  `HandleGetRecurringInvoice`, `HandleUpdateRecurringInvoice`, `HandleDeleteRecurringInvoice`,
  `HandlePauseRecurringInvoice`, `HandleResumeRecurringInvoice`, `HandleGenerateRecurringInvoice`
  sowie den gemeinsam genutzten `setRecurringStatus`-Helfer ueber beide Aufrufstellen). Guard-
  Wiring der ganzen Gruppe ist bereits vollstaendig in `route_capability_guard_test.go`
  ("finance: recurring invoices") getestet und wird hier bewusst NICHT dupliziert.
  Doppelausloesung von `HandleGenerateRecurringInvoice` wurde wie von der Unit gefordert
  ueberprueft: `internal/biz/recurring/service.go:282` (`Service.Generate`) beansprucht die
  Periode ueber `repo.ClaimPeriod` VOR dem Erzeugen der Rechnung und beantwortet einen zweiten
  Aufruf mit der bereits erzeugten Rechnung (Replay), belegt durch die bestehende
  `TestGenerate_IsIdempotentPerPeriod`. Kein Fund, kein Fix noetig — die Route ist ein reiner
  Pass-Through ohne eigene Idempotenz-Logik. Monatsgrenzen (31. in einem 30-Tage-Monat, 29.
  Februar) sind ebenfalls bereits an der Service-Schicht abgedeckt
  (`TestNextRunFor_AnchorsAtStartAndClampsMonthEnd`) und dort nicht Sache der Route.
  ECHTER FUND waehrend des Bauens: `HandleUpdateRecurringInvoice`s Leerstring-Sentinel fuer
  "Enddatum loeschen" (`if *req.EndDate == "" { grpcReq.ClearEndDate = true }`,
  route_biz_recurring.go:179) ist ueber HTTP nie erreichbar. `decodeAndValidate` prueft
  `validate:"omitempty,datetime=2006-01-02"` auf dem `*string`-Feld, BEVOR der Handler seinen
  eigenen Zweig sieht; go-playground/validators `omitempty` ueberspringt bei einem Pointer-Feld
  nur einen NIL-Pointer, nicht einen Pointer auf einen leeren String — ein per JSON gesendetes
  `"end_date": ""` faellt also mit 400 "failed datetime" durch, bevor die Sentinel-Logik greift.
  Empirisch belegt durch
  `TestHandleUpdateRecurringInvoice_EmptyEndDateRejectedByValidationBeforeClearLogicRuns` (400,
  Feld "end_date" statt der beabsichtigten Loeschung). Nicht selbst gefixt (Coverage-Unit darf
  kein Verhalten aendern) — neue Unit `fix-recurring-invoice-clear-end-date` ans Backlog-Ende
  gehaengt, inklusive zweier legitimer Fixrichtungen zur Auswahl.
- gate: build ok (`-p 2` ueber gateway/..., cmd/gateway/...) | vet ok | lint ok (0 issues) |
  test ok (ganzes `internal/gateway` gruen inkl. `TestOpenAPIRouteDrift`, 836 Routen gegen 838
  dokumentierte Pfade, keine neue Route) | migration n.a. (keine Schemaaenderung)
- coverage: internal/gateway 47,4 % -> 47,7 % (eigene Messung: Testdatei nach `/tmp`
  verschoben, `go test -coverprofile` vorher/nachher; 47,4 % ist der reale lokale Stand nach
  Iteration 17, nicht der `coverage_start` der Unit aus dem CI-Snapshot 1b49a1f3 — Abweichung,
  weil vier vorherige Iterationen dasselbe Paket in diesem Lauf bereits angehoben haben)
- mutations-probe: in `createRecurringRequest.LineItems` das Tag von `required,min=1` auf
  `required,min=0` gesenkt -> `TestHandleCreateRecurringInvoice_MissingLineItems` wird rot (503
  statt 400, RPC-Layer erreicht statt Validierungs-Guard). Zurueckgedreht -> Test wieder gruen,
  `git diff --stat` auf `route_biz_recurring.go` zeigt keine Aenderung mehr, ganzes
  `internal/gateway` gruen.
- verify vorgaenger: sauber, mit einer nachgetragenen Korrektur. `5f8c9871` (Iteration 17,
  Quote-Route-Tests) gegen alle acht Fehlerklassen geprueft: `git show --stat` zeigt nur
  `BACKLOG.yml`, `JOURNAL.md` und die neue `route_biz_quotes_test.go` — reine Testdatei, kein
  Produktionscode, kein gRPC-Layer-Bypass (Handler-Aufrufe ausschliesslich ueber `client.<RPC>`),
  kein Stub/TODO, kein `.proto` angefasst, keine neue `RequirePermission` (Guard-Wiring laut
  Journal bereits vollstaendig in `route_capability_guard_test.go` abgedeckt), keine neue
  Tabelle, keine Wire-Shape-Aenderung, kein Guard ersetzt. Die Journal-SHA fuer Iteration 17
  fehlte noch ("folgt im naechsten Schritt") — vor dieser Unit in einem separaten docs-Commit
  nachgetragen (`82032b5f`, Wert `5f8c9871` durch `git log`/`git show --stat` gegen den
  tatsaechlichen Commit-Inhalt bestaetigt).
- neue-units: fix-recurring-invoice-clear-end-date
- offen:
  - "Fremder Tenant liefert 404" ist an dieser Route wie in jeder vorigen Coverage-Unit dieses
    Laufs NICHT ueber einen Live-RPC-Roundtrip getestet (kein bufconn-/Fake-Client-Harness fuer
    `FinanceServiceClient` im ganzen Paket, gleiche dokumentierte Infrastrukturgrenze wie in den
    Iterationen 15-17). Die serverseitige Tenant-Durchsetzung liegt bei `internal/biz/recurring`
    (WHERE tenant_id = $1), ausserhalb des Scopes dieser Route-Unit.
  - Der neue Fund `fix-recurring-invoice-clear-end-date` betrifft ausschliesslich das
    Enddatum-Feld beim Update; ob dasselbe Pointer-vs-omitempty-Muster fuer ein "Leerstring
    loescht"-Sentinel noch anderswo im `biz`-Paket vorkommt, ist als offene Frage in die neue
    Unit selbst geschrieben (Notes-Feld), nicht hier separat geprueft — das haette den Scope
    dieser Coverage-Unit gesprengt.

## Iteration 19 — cov-gateway-biz-open-items-time-entries — done — 2026-08-22 03:22
- commit: e8b862e9
- gebaut: neue Datei `route_biz_open_items_time_entries_test.go` deckt beide Handler ab, die
  in dieser Unit im Fokus standen: `HandleListOpenItems` (route_biz_open_items.go, 1 Funktion)
  und `HandleListTimeEntries` (route_biz_time_entries.go, 1 Funktion). Je Handler: Service-
  Unavailable (leere Registry), fehlender Tenant (nur bei open-items moeglich, siehe unten),
  und mehrere ReachesRPC-Faelle mit unterschiedlichen Query-Parametern (Standard-Pagination,
  bucket+overdue_only-Filter, explizite page/page_size, ein page_size ueber dem Maximum von
  parsePagination). Zusaetzlich in `route_capability_guard_test.go` zwei neue Faelle fuer
  `/api/v1/finance/time-entries` (bislang nicht in der Guard-Tabelle, obwohl open-items direkt
  daneben schon drin stand) — belegt, dass die Route ausschliesslich am reinen
  `finance:read`-Legacy-Schluessel haengt, ohne feineren Alias.
  ECHTER FUND, aber trivial: der Go-Doc-Kommentar von `HandleListOpenItems` dokumentierte die
  Query-Parameter als `?bucket=&overdue_only=&page=&per_page=`, waehrend der Code ueber
  `parsePagination` tatsaechlich `page_size` liest (wie `openapi.yaml` korrekt zeigt) — ein
  gesendetes `per_page` wurde also stillschweigend ignoriert und faellt auf den Default-Wert 50
  zurueck. Nur der Kommentar war falsch, kein Verhalten geaendert; direkt korrigiert
  (`page_size=`), da es sich um eine reine Doc-String-Korrektur ohne Code- oder Testaenderung
  handelt und keine eigene Unit rechtfertigt.
  ARCHITEKTUR-BEFUND (kein Bug): `HandleListTimeEntries` prueft den Tenant gar nicht selbst —
  im Unterschied zu praktisch jedem anderen Handler in diesem Paket. Das ist beabsichtigt:
  `WorkGRPCServer.ListBillableTimeEntries` (work_grpc.go:2005) liest den Tenant serverseitig
  aus dem RLS-Kontext, der ueber `TenantOutboundUnaryInterceptor`
  (internal/middleware/grpc_tenant.go:37) automatisch aus dem HTTP-Request-Kontext in die
  ausgehenden gRPC-Metadaten kopiert wird. Ein Test ohne Tenant im Kontext erreicht deshalb
  trotzdem die RPC-Ebene (503 gegen die Dummy-Verbindung) statt eines Gateway-seitigen 401 —
  dieses Verhalten ist als eigener Test festgenagelt
  (`TestHandleListTimeEntries_NoTenantInContext_StillReachesRPC`), nicht als Luecke gemeldet.
- gate: build ok (`-p 2` ueber gateway/..., cmd/gateway/...) | vet ok | lint ok (0 issues) |
  test ok (ganzes `internal/gateway` gruen inkl. `TestOpenAPIRouteDrift`, keine neue Route,
  kein OpenAPI-Eintrag noetig)
- coverage: internal/gateway 47,7 % -> 47,8 % (eigene Messung: Testdatei + die zwei neuen
  Guard-Faelle per `git stash` vollstaendig entfernt, `go test -coverprofile` vor/nach meiner
  Aenderung auf demselben Arbeitsbaum; 47,7 % ist der reale lokale Stand, deckt sich mit dem
  47,7 % Zwischenwert aus Iteration 18s Messkette)
- mutations-probe: in `HandleListOpenItems` die Tenant-Fehlerpruefung `if err != nil` auf
  `if false` gesetzt -> `TestHandleListOpenItems_NoTenantID` wird rot (503 statt 401, RPC-Layer
  erreicht statt 401-Guard). Zurueckgedreht -> `git diff --stat` auf `route_biz_open_items.go`
  zeigt nur noch die Kommentar-Korrektur, kein Rest der Mutation; ganzes `internal/gateway`
  wieder gruen.
- verify vorgaenger: sauber. `791ba2bf` (Iteration 18, Recurring-Invoice-Route-Tests) gegen
  alle acht Fehlerklassen geprueft: `git show --stat` zeigt nur `BACKLOG.yml`, `JOURNAL.md` und
  die neue `route_biz_recurring_test.go` — reine Testdatei ohne Client-Aufruf-Pattern
  (`grep -n "client\.\|Unimplemented\|TODO\|t.Skip"` liefert null Treffer), kein gRPC-Bypass,
  kein `.proto` angefasst, keine neue `RequirePermission`, keine neue Tabelle, keine
  Wire-Shape-Aenderung, kein Guard ersetzt, keine neue Route.
- neue-units: keine
- offen:
  - "Fremder Tenant liefert 404" ist an beiden Routen wie in jeder vorigen Coverage-Unit dieses
    Laufs NICHT ueber einen Live-RPC-Roundtrip getestet (kein bufconn-/Fake-Client-Harness fuer
    FinanceServiceClient/WorkServiceClient im ganzen Paket, gleiche dokumentierte
    Infrastrukturgrenze wie in den Iterationen 15-18). Die serverseitige Tenant-Durchsetzung
    liegt bei den jeweiligen Service-Paketen.
  - "Leeres Ergebnis liefert 200 mit leerer Liste, nicht 404" (done_when-Kriterium) ist aus
    demselben Grund nicht live pruefbar — die Antwort wird erst nach einer erfolgreichen RPC
    gebaut, die in diesem Paket nicht erreichbar ist. `hrMarshalSlice` (bereits andernorts
    getestet, u.a. route_hr_worktime_test.go) garantiert generisch `[]` statt `nil`, das ist die
    naechstliegende Absicherung ohne Live-Client.
  - Der `lean:`-Marker in `HandleListTimeEntries` (billed=true liefert immer eine leere Liste,
    ohne dass irgendetwas je als "billed" markiert wird) ist bereits im Code als bewusste
    Vereinfachung mit eigenem Upgrade-Trigger dokumentiert — kein neuer Fund, keine Unit noetig.

## Iteration 20 — cov-gateway-biz-banking-document-chains — done — 2026-08-22 03:30
- commit: 2b621da2
- gebaut: zwei neue Testdateien fuer die letzten beiden ungetesteten `route_biz_*`-Dateien
  dieser Nachbarschaft. `route_biz_banking_test.go` deckt alle drei Funktionen von
  `route_biz_banking.go` ab (HandleImportBankStatement, HandleListBankStatements,
  HandleGetBankStatement): Service-Unavailable, fehlender Tenant, ungueltiges Multipart,
  fehlendes Dateifeld, leere Datei, Datei ueber dem 10-MiB-Limit, sowie ReachesRPC-Faelle fuer
  Standard-/explizite Pagination. `route_biz_document_chains_test.go` deckt
  `HandleListDocumentChains` (Service-Unavailable, fehlender Tenant, ReachesRPC) sowie die drei
  reinen Formatierfunktionen im selben File (`toDocumentChainWire`, `formatChainAmount`,
  `groupThousands`) isoliert: Rundung, Vorzeichen, Tausendertrennzeichen-Grenzen (999/1000/1e6),
  unparsierbarer/leerer Betrag faellt auf "0,00" zurueck, Em-Dash-Platzhalter fuer den
  nummer-/datumslosen synthetischen "offener Saldo"-Knoten, leere Knotenliste marshalt zu `[]`
  nicht `null`.
  ZWEI PRAEMISSEN AUS DEM UNIT-ENTWURF KORRIGIERT (beide gegen den Code geprueft, nicht nur
  behauptet):
  1. "IBAN-Formatierung und -Maskierung isoliert testen" trifft auf `route_biz_banking.go` nicht
     zu. Es gibt in der gesamten Codebase (Gateway UND `internal/biz/banking`) keine
     IBAN-Maskierung — nur eine Formatierung/Gruppierung, und die liegt ausschliesslich in
     `route_biz_bank_accounts.go` (`dachfmt.FormatIBAN`, bereits durch
     `route_biz_bank_accounts_test.go` abgedeckt). `BankStatement.account_iban` und
     `BankTransaction.counterparty_iban` laufen unformatiert durch `hrMarshalProto`/
     `hrMarshalSlice` — deckungsgleich mit dem OpenAPI-Schema (`account_iban?: string`, keine
     Formatierungsangabe). Nichts zu maskieren, nichts isoliert zu testen; im Testdatei-Header
     dokumentiert statt stillschweigend uebergangen.
  2. "Eine Kette mit Luecke oder Zyklus darf keine Endlosschleife ausloesen" setzt eine
     Graph-Traversierung voraus, die nirgends existiert. Gelesen:
     `internal/biz/invoice/postgres_document_chains.go` — jede Kette wird pro Rechnung als
     flacher Fan-out ueber fuenf unabhaengige, tenant-gescopte Queries plus Map-Lookup nach
     Rechnungs-ID zusammengesetzt, keine Adjazenzliste, keine Rekursion. Auf Gateway-Seite
     iteriert `HandleListDocumentChains`/`toDocumentChainWire` einmalig ueber eine endliche
     Slice. Es gibt also weder auf Service- noch auf Gateway-Ebene einen Codepfad, den ein
     Luecken-/Zyklus-Test ueberhaupt treffen koennte — kein Test gebaut, Begruendung im
     Testdatei-Header festgehalten statt eine Attrappe drumherum zu bauen.
- gate: build ok (`-p 2` ueber gateway/..., cmd/gateway/...) | vet ok | lint ok (0 issues) |
  test ok (ganzes `internal/gateway` gruen inkl. `TestOpenAPIRouteDrift`, `internal/biz/banking`
  unveraendert gruen, keine neue Route, kein OpenAPI-Eintrag noetig, 0 SKIP in
  `internal/gateway` mit `DATABASE_URL` gesetzt)
- coverage: internal/gateway 47,8 % -> 48,2 % (eigene Messung: beide neuen Testdateien per
  `git stash push -u` vollstaendig entfernt, `go test -coverprofile` vor/nach auf demselben
  Arbeitsbaum, danach `git stash pop`; 47,8 % deckt sich mit dem in Iteration 19 gemessenen
  Zwischenwert)
- mutations-probe: in `HandleImportBankStatement` die Groessenlimit-Pruefung
  `if len(content) > maxBankStatementUploadBytes` auf `if false && len(content) > ...` gesetzt
  -> `TestHandleImportBankStatement_ExceedsSizeLimit` wird rot (503 gegen die Dummy-Verbindung
  statt 413, RPC-Layer statt Guard erreicht). Zurueckgedreht -> `git diff --stat` auf
  `route_biz_banking.go` zeigt keinen Rest, ganzes `internal/gateway` wieder gruen.
- verify vorgaenger: sauber. `e8b862e9` (Iteration 19, Open-Items/Time-Entries-Route-Tests)
  gegen alle acht Fehlerklassen geprueft: `git show --stat` zeigt nur `BACKLOG.yml`,
  `JOURNAL.md`, die neue Testdatei und einen additiven Guard-Testeintrag sowie eine reine
  Doc-Kommentar-Korrektur in `route_biz_open_items.go` (`per_page=` -> `page_size=`, kein
  Verhalten geaendert). `grep -n "client\.\|Unimplemented\|TODO\|t.Skip"` auf der neuen
  Testdatei liefert null Treffer, kein gRPC-Bypass, kein `.proto` angefasst, keine neue
  `RequirePermission`, keine neue Tabelle, keine Wire-Shape-Aenderung, kein Guard ersetzt,
  keine neue Route.
- neue-units: keine
- offen:
  - "Fremder Tenant liefert 404" ist an allen vier Routen dieser Unit wie in jeder vorigen
    Coverage-Unit dieses Laufs NICHT ueber einen Live-RPC-Roundtrip getestet (kein bufconn-/
    Fake-Client-Harness fuer FinanceServiceClient im ganzen Paket, gleiche dokumentierte
    Infrastrukturgrenze wie in den Iterationen 15-19). Die serverseitige Tenant-Durchsetzung
    liegt bei `internal/biz/banking` bzw. `internal/biz/invoice`.
  - Die zwei korrigierten Praemissen (IBAN-Maskierung existiert nicht, Belegketten sind kein
    traversierbarer Graph) sind ausschliesslich im Header der jeweiligen neuen Testdatei
    dokumentiert, nicht als eigene Backlog-Units — es gibt nichts zu fixen oder nachzubauen,
    nur eine falsche Annahme im urspruenglichen Unit-Entwurf richtigzustellen.

## Iteration 21 — cov-gateway-biz-ext-time-billing — done — 2026-08-22 03:47
- commit: 68e59ffe
- gebaut: neue Testdatei `route_biz_ext_test.go` deckt die einzige Funktion in
  `route_biz_ext.go` ab (`HandleCreateInvoiceFromTime`; `NewBizExtRoutes`/`getBizClient` werden
  implizit mitgetestet, `registerTimeExtRoutes` war bereits über `openapi_drift_test.go`
  abgedeckt, das den vollen Router inkl. `BizExtRoutes` baut): Service-Unavailable (mit
  gültigem Body, siehe unten), fehlender Tenant, ungültiges JSON, fehlende/ungültige
  `employee_id`, fehlender `customer_name`, ungültige `customer_email`, ungültiges
  `date_from`-Format, fehlendes `date_to`, fehlender/null/negativer/nicht-numerischer
  `hourly_rate` (`decimal_gt0`), ungültiger `tax_mode`, leerer `tax_mode` (erlaubt, Service
  leitet ihn aus den Firmeneinstellungen ab), gültiger Request erreicht die RPC-Schicht.
  ABWEICHUNG VON DER STANDARD-VORLAGE: `HandleCreateInvoiceFromTime` prüft Tenant, dann
  `decodeAndValidate`, erst DANACH den gRPC-Client — die meisten anderen `biz`-Handler prüfen
  den Client zuerst. Der generische `testServiceUnavailable`-Helfer (leerer `{}`-Body) griff
  deshalb nicht (400 vor 503); der Test baut stattdessen einen gültigen Body von Hand und ist
  im Code kommentiert, warum.
  FUND WÄHREND DES BAUENS, NICHT SELBST GEFIXT (Coverage-Unit darf kein Verhalten ändern):
  `AggregateWorkTimeForInvoice` (postgres_repository.go:532) filtert nur auf
  Tenant/Mitarbeiter/Status/Zeitfenster — kein Ausschluss bereits abgerechneter Einträge, keine
  `billed`-Spalte auf `hr_work_time_entries` überhaupt (gegenverifiziert:
  migrations/000046_create_hr_tables.up.sql:120, und `LinkTimeTracking` schreibt den
  Audit-Trail nur, liest ihn nie zurück). Zwei Aufrufe mit demselben Mitarbeiter/Zeitraum
  erzeugen zwei Rechnungen über dieselben Stunden. Neue Unit
  `fix-biz-time-entry-invoice-double-billing` ans Backlog-Ende angehängt.
- gate: build ok (`-p 2` über gateway/..., cmd/gateway/...) | vet ok | lint ok (0 issues) |
  test ok (ganzes `internal/gateway` grün inkl. `TestOpenAPIRouteDrift` — 836 Routen gegen 838
  dokumentierte Pfade, 0 SKIP mit `DATABASE_URL` gesetzt, keine neue Route/kein OpenAPI-Eintrag
  nötig)
- coverage: internal/gateway 48,2 % -> 48,3 % (eigene Messung: neue Testdatei per
  `git stash push -u` vollständig entfernt, `go test -coverprofile` vor/nach auf demselben
  Arbeitsbaum, danach `git stash pop`; 48,2 % deckt sich mit dem in Iteration 20 gemessenen
  Nachher-Wert)
- mutations-probe: `validate:"required,decimal_gt0"` auf `HourlyRate` entfernt -> vier Tests
  rot (`MissingHourlyRate`, `ZeroHourlyRateRejected`, `NegativeHourlyRateRejected`,
  `NonNumericHourlyRateRejected` — alle erreichen jetzt die RPC-Schicht statt 400).
  Zurückgedreht -> `git diff --stat` auf `route_biz_ext.go` zeigt keinen Rest, ganzes
  `internal/gateway` wieder grün.
- verify vorgaenger: sauber. `2b621da2` (Iteration 20, Banking/Document-Chains-Route-Tests)
  gegen alle acht Fehlerklassen geprüft: `git show --stat` zeigt nur `BACKLOG.yml`,
  `JOURNAL.md` und zwei neue reine Testdateien. `grep -n "client\.\|Unimplemented\|TODO\|
  t.Skip"` auf beiden liefert null Treffer, kein gRPC-Bypass, kein `.proto` angefasst, keine
  neue `RequirePermission`, keine neue Tabelle, keine Wire-Shape-Änderung, kein Guard ersetzt,
  keine neue Route.
- neue-units: fix-biz-time-entry-invoice-double-billing
- offen:
  - Wie in jeder vorigen Coverage-Unit dieses Laufs ist "fremder Tenant liefert 404" nicht über
    einen Live-RPC-Roundtrip getestet (kein bufconn-/Fake-Client-Harness für
    FinanceServiceClient im ganzen Paket). Die serverseitige Tenant-Durchsetzung liegt bei
    `internal/server/biz_grpc.go`.
  - Der Double-Billing-Fund ist ausschließlich im Header der neuen Testdatei und in der neuen
    Backlog-Unit dokumentiert, nicht anderswo — nichts an dieser Iteration selbst ändert
    Produktionsverhalten.

## Iteration 22 — cov-gateway-settings-module-leads-grants — done — 2026-08-22 03:56
- commit: 2658c110
- gebaut: PRAEMISSE DES UNIT-ENTWURFS WIDERLEGT, statt sie ungeprueft nachzubauen: die Datei
  `route_settings.go` hat entgegen Befund 5 im Backlog-Kopf ("23 von 75 route_*.go ohne eigene
  Testdatei", route_settings.go als Beispiel genannt) bereits DREI Testdateien
  (`route_settings_module_access_test.go`, `route_settings_license_test.go`,
  `route_settings_preferences_test.go`), alle aus Commit `5fc601af` vom 2026-08-11 — vor Beginn
  dieses Laufs, auf `main`. Der Modul-/Grant-Teil dieser Unit (`HandleListModuleLeads` bis
  `HandleBulkRevokeModuleAccess`, Zeilen 121-444) ist darin bereits mit 8 Funktionstests
  abgedeckt: Service-Unavailable, fehlender Tenant/Caller, ungueltige UUID, fehlende
  `module_id`, ReachesRPC — je Handler. Eigene Coverage-Messung vor jeder Aenderung
  (`go tool cover -func`) bestaetigt 76-100 % fuer alle acht Handler; nur die reine
  Marshaling-Funktion `toUserModuleGrantJSON` lag bei 0,0 %, weil kein Test in diesem Paket je
  ueber die RPC-Schicht hinauskommt (kein bufconn-/Fake-Client-Harness fuer
  `SettingsServiceClient`, dieselbe dokumentierte Infrastrukturgrenze wie in den Iterationen
  15-21 fuer andere Clients). Genau diese eine Luecke geschlossen: zwei neue isolierte Tests
  fuer `toUserModuleGrantJSON` (RFC3339-Formatierung von `GrantedAt`, `LastActiveAt` nil vs.
  gesetzt — ein nie aktiver, frisch eingeladener Nutzer darf keine erfundene "zuletzt aktiv"-Zeit
  bekommen). Die beiden anderen done_when-Kriterien sind ueber Bestehendes erfuellt, nicht neu
  gebaut: Permission-Seed fuer `module-leads`/`module-grants` existiert bereits
  (`000221_seed_module_grants_permissions.up.sql`); der Rechteausweitungs-Test ("Benutzer gibt
  sich selbst ein fehlendes Recht") ist generisch am Guard-Mechanismus selbst abgedeckt
  (`internal/middleware/rbac_test.go:TestRequirePermission`), nicht pro Route dupliziert — ein
  Route-Handler-Test kann das ohnehin nicht pruefen, weil `RequirePermission` als
  Router-Middleware (`chi.Router.With(...)`) VOR dem Handler sitzt, nicht im Handler selbst; die
  bestehenden Tests rufen die Handler-Methoden direkt auf und liefen nie durch die Middleware.
- gate: build ok (`-p 2` ueber gateway/..., cmd/gateway/...) | vet ok | lint ok (0 issues) |
  test ok (ganzes `internal/gateway` gruen, `internal/settings` gruen, DATABASE_URL gegen
  kmuhub_app; kein neuer Guard, keine neue Route, `TestOpenAPIRouteDrift` unveraendert im
  Gesamtlauf mitgelaufen)
- coverage: internal/gateway 48,3 % -> 48,4 % (eigene Messung: vor/nach genau der beiden neuen
  Testfaelle im selben Arbeitsbaum; `route_settings.go`-Funktionsdeckung fuer den Modul-/
  Grant-Teil lag schon bei 76-100 %, `toUserModuleGrantJSON` ging von 0,0 % auf 100 %)
- mutations-probe: `if g.LastActiveAt != nil` in `toUserModuleGrantJSON` auf `if true` gesetzt ->
  `TestToUserModuleGrantJSON_NilLastActiveAt` wird rot (LastActiveAt-Pointer gesetzt statt nil).
  Zurueckgedreht -> gruen, `git diff --stat` zeigt fuer `route_settings.go` keinen Rest (0
  Zeilen Diff), nur die Testdatei und die Backlog-Statuszeile geaendert.
- verify vorgaenger: sauber. `68e59ffe` (Iteration 21, Time-Entry-Ext-Route-Tests) gegen alle
  acht Fehlerklassen geprueft: `git show --stat` zeigt nur `BACKLOG.yml`, `JOURNAL.md` und eine
  neue reine Testdatei. `grep -n "client\.\|Unimplemented\|TODO\|t.Skip"` auf der neuen Testdatei
  liefert null Treffer, kein gRPC-Bypass, kein `.proto` angefasst, keine neue
  `RequirePermission`, keine neue Tabelle, keine Wire-Shape-Aenderung, kein Guard ersetzt, keine
  neue Route. Die im Journal genannte neue Unit `fix-biz-time-entry-invoice-double-billing`
  steht tatsaechlich in `BACKLOG.yml` (Zeile 1787).
- neue-units: keine
- offen:
  - BEFUND FUER DEN LAUFKOPF: Befund 5 ("23 von 75 route_*.go ohne Testdatei, groesste
    UNGETESTETE route_settings.go") stimmt fuer `route_settings.go` nicht mehr — die Datei hat
    seit 2026-08-11 (vor diesem Lauf) drei Testdateien. Die verbleibenden zwei Backlog-Units zu
    dieser Datei (`cov-gateway-settings-branding-tenant-user`,
    `cov-gateway-settings-license-subscription-verify`) muessen deshalb VOR dem Bauen erst ihre
    eigene Ist-Abdeckung messen (das fordern ihre `notes:` ohnehin schon so) — nach Stichprobe
    mit `go tool cover -func` sind `HandleGetBranding`, `HandlePutBranding`,
    `HandleGetTenantSettings`, `HandlePutTenantSettings`, `HandleGetResolvedSettings`,
    `HandleGetUserSettings`, `HandlePutUserSettings`, `HandleGetTenantLicense`,
    `HandleSetTenantModuleActive`, `HandleGetTenantSubscription` weiterhin bei 0,0 % (nur
    `toTenantModuleJSON`/`moduleAvailable` sind ueber den License-Test bei 100 %) — fuer diese
    beiden Folge-Units bleibt also echte Arbeit, nur eben kleiner als der urspruengliche
    28-Funktionen-Umfang suggeriert.
  - Kein DB-Gate noetig (keine Migration, keine Tabelle/Policy beruehrt).

## Iteration 23 — cov-gateway-settings-branding-tenant-user — done — 2026-08-22 03:50
- commit: 833e454d
- gebaut: Neue Testdatei `route_settings_branding_tenant_user_test.go` fuer die sieben
  verbleibenden ungetesteten Handler aus `route_settings.go`: HandleGetResolvedSettings,
  HandleGetTenantSettings, HandlePutTenantSettings, HandleGetUserSettings,
  HandlePutUserSettings, HandleGetBranding, HandlePutBranding. Eigene Messung vor dem Schreiben
  (wie von den notes gefordert, siehe auch Befund aus Iteration 22) bestaetigt: alle sieben lagen
  bei 0,0 %, `rawMapToSettingEntries` bei 80,0 %. Pro Handler das Muster aus
  `route_settings_module_access_test.go`/`route_bexio_test.go`: ServiceUnavailable,
  MissingTenant, fehlende Auth (NoUserID/NoCallerID wo zutreffend), Validierungsfehler
  (InvalidJSON, fehlendes Pflichtfeld ueber `assertValidationError`), ReachesRPC (503 gegen die
  Dummy-Verbindung, kein bufconn-/Fake-Client-Harness fuer SettingsServiceClient im Paket).
  `HandlePutBranding` zusaetzlich mit `TestHandlePutBranding_NameTooLong` (max=200-Grenze) und
  einem Kommentar, der begruendet, warum `accent_color`/`logo_object_key`/`icon_object_key` am
  Gateway bewusst NICHT gegen die Cosmi-Swatch-Palette bzw. das Tenant-Praefix geprueft werden:
  diese Validierung liegt in `internal/settings/branding.go`
  (`allowedAccentColors`/`brandingObjectKeyValid`) und ist dort bereits in `branding_test.go`
  abgedeckt — Thick-Services/Thin-Handlers, keine Duplikat-Pruefung am Gateway. Fuer
  `HandleGetTenantSettings` zusaetzlich `TestHandleGetTenantSettings_CrossTenant` mit Kommentar,
  der dokumentiert, warum es keinen Code-Pfad fuer einen fremden Tenant an dieser Stelle gibt
  (TenantId kommt ausschliesslich aus dem Auth-Kontext, nie aus der Anfrage) — die serverseitige
  Durchsetzung liegt in `internal/server/settings_grpc.go`, ausserhalb der Reichweite dieses
  Pakets ohne Live-RPC-Harness.
- gate: build ok (`-p 2` gateway/..., cmd/gateway/...) | vet ok | lint ok (0 issues) | test ok
  (neue Testdatei einzeln gruen, ganzes `internal/gateway` gruen inkl. `TestOpenAPIRouteDrift`
  [836 Routen gegen 838 dokumentierte Pfade, unveraendert], `internal/settings` gruen,
  DATABASE_URL gegen kmuhub_app, 0 SKIP im Gateway-Paketlauf)
- coverage: internal/gateway 48,4 % -> 48,9 % (eigene Messung vor/nach genau dieser Testdatei im
  selben Arbeitsbaum). Alle sieben Handler 0,0 % -> 88,0-94,4 %; `rawMapToSettingEntries`
  unveraendert bei 80,0 % (die verbleibende Luecke ist der `structpb.NewValue`-Fehlerpfad, der aus
  `encoding/json`-dekodiertem `interface{}` praktisch nicht erreichbar ist — jeder JSON-Wert
  mappt auf einen von structpb unterstuetzten Go-Typ).
- mutations-probe: `if callerID == ""` in `HandlePutBranding` auf `if false` gesetzt ->
  `TestHandlePutBranding_NoCallerID` wird rot (503 statt erwartetem 401). Zurueckgedreht -> gruen,
  `git diff --stat` zeigt fuer `route_settings.go` 0 Zeilen Diff, nur die neue Testdatei und die
  Backlog-Statuszeile geaendert.
- verify vorgaenger: sauber. `2658c110` (Iteration 22, Modul-Grant-JSON-Marshaling-Coverage)
  gegen alle acht Fehlerklassen geprueft: `git show --stat` zeigt nur `BACKLOG.yml`,
  `JOURNAL.md` und eine neue reine Testdatei (`route_settings_module_access_test.go`, +58
  Zeilen). `grep -n "client\.\|Unimplemented\|TODO\|t.Skip\|RequirePermission"` auf der neuen
  Testdatei liefert null Treffer, kein gRPC-Bypass, kein `.proto` angefasst, keine neue
  `RequirePermission`, keine neue Tabelle, keine Wire-Shape-Aenderung, kein Guard ersetzt, keine
  neue Route.
- neue-units: keine
- offen:
  - Wie in jeder vorigen Coverage-Unit dieses Laufs ist "fremder Tenant liefert 404/403" fuer
    diese Handler nicht ueber einen Live-RPC-Roundtrip getestet (kein bufconn-/Fake-Client-
    Harness fuer SettingsServiceClient im ganzen Paket). Die serverseitige Tenant-Durchsetzung
    liegt bei `internal/server/settings_grpc.go`. Bei `HandleGetTenantSettings`/`HandleGetUserSettings`/
    `HandlePutTenantSettings`/`HandlePutUserSettings` gibt es ohnehin keinen Code-Pfad, der einen
    fremden Tenant annehmen koennte — TenantId/UserId kommen ausschliesslich aus dem
    Auth-Kontext, nie aus Body oder URL.
  - Kein DB-Gate noetig (keine Migration, keine Tabelle/Policy beruehrt).

## Iteration 24 — cov-gateway-settings-license-subscription-verify — done — 2026-08-22 03:55
- commit: 1dc3368f
- gebaut: Neue Testdatei `route_settings_license_subscription_test.go` fuer den dritten und
  letzten Teil von `route_settings.go`: HandleGetTenantLicense, HandleSetTenantModuleActive,
  HandleGetTenantSubscription. Eigene Messung vor dem Schreiben bestaetigt: alle drei lagen bei
  0,0 % (toTenantModuleJSON/moduleAvailable waren bereits ueber route_settings_license_test.go
  bei 100 %, wie in Iteration 22 vermerkt). Muster wie in den beiden Vorgaenger-Units:
  ServiceUnavailable, MissingTenant, NoCallerID, InvalidJSON, ReachesRPC (503 gegen die
  Dummy-Verbindung). HandleSetTenantModuleActive zusaetzlich mit den vier Faellen, die den
  eigentlichen Handler-Wert ausmachen: MissingActive (400), UnknownModule (404, Katalog-Lookup),
  ModuleNotAvailable (409 — Aktivieren eines Moduls, dessen Deployment-Flag aus ist, wuerde eine
  Zeile schreiben, die die naechste GET sofort wieder maskiert) und
  DeactivateUnavailableModule, das die bewusste Asymmetrie dokumentiert: Deaktivieren ist NICHT
  hinter demselben Gate wie Aktivieren. Kein Feature-Flag umgestellt ausser lokal in der
  Testfixture (`flagsWithHelpdeskEnabled`, analog zur Vorlage aus
  `route_settings_license_test.go`), keine Preiszahl in einem Test.
- gate: build ok (`-p 2` gateway/..., cmd/gateway/...) | vet ok | lint ok (0 issues) | test ok
  (neue Testdatei einzeln gruen: 14/14, ganzes `internal/gateway` gruen inkl.
  `TestOpenAPIRouteDrift` [836 Routen gegen 838 dokumentierte Pfade, unveraendert], DATABASE_URL
  gegen kmuhub_app, 0 SKIP im Gateway-Paketlauf)
- coverage: internal/gateway 48,9 % -> 49,1 % (eigene Messung vor/nach genau dieser Testdatei im
  selben Arbeitsbaum). HandleGetTenantLicense 0,0 % -> 75,0 %, HandleSetTenantModuleActive
  0,0 % -> 96,7 %, HandleGetTenantSubscription 0,0 % -> 60,0 % (Rest ist der
  BillingPeriodEnd/TotalSeats-Zweig, der ohne bufconn-Fake-Client fuer eine echte RPC-Antwort
  nicht erreichbar ist).
- mutations-probe: `if *req.Active && !sr.moduleAvailable(req.ModuleID)` in
  `HandleSetTenantModuleActive` auf `if false && ...` gesetzt ->
  `TestHandleSetTenantModuleActive_ModuleNotAvailable` wird rot (503 statt erwartetem 409).
  Zurueckgedreht -> `git diff --stat` zeigt fuer `route_settings.go` 0 Zeilen Diff, nur die neue
  Testdatei und die Backlog-Statuszeile geaendert.
- verify vorgaenger: sauber. `833e454d` (Iteration 23, Branding/Tenant/User-Settings-Coverage)
  gegen alle acht Fehlerklassen geprueft: `git show --stat` zeigt nur `BACKLOG.yml`,
  `JOURNAL.md` und eine neue reine Testdatei (`route_settings_branding_tenant_user_test.go`, +364
  Zeilen). `grep -n "client\.\|Unimplemented\|TODO\|t.Skip\|RequirePermission\|\.proto"` auf der
  neuen Testdatei liefert null Treffer, kein gRPC-Bypass, kein Stub, kein `.proto` angefasst,
  keine neue `RequirePermission`, keine neue Tabelle, keine Wire-Shape-Aenderung, kein Guard
  ersetzt, keine neue Route.
- neue-units: keine
- offen:
  - Damit ist `route_settings.go` komplett getestet — alle drei Coverage-Units aus diesem
    Backlog (Modul-Leads/Grants, Branding/Tenant/User, Lizenz/Abo) sind jetzt `done`.
  - Wie bei jeder Coverage-Unit dieses Laufs: "fremder Tenant liefert 404/403" ist fuer diese
    drei Handler nicht ueber einen Live-RPC-Roundtrip getestet (kein bufconn-/Fake-Client-
    Harness fuer SettingsServiceClient im Paket). Bei allen drei Handlern gibt es ohnehin keinen
    Code-Pfad, der einen fremden Tenant annehmen koennte — TenantId kommt ausschliesslich aus
    dem Auth-Kontext.
  - Kein DB-Gate noetig (keine Migration, keine Tabelle/Policy beruehrt).

## Iteration 25 — cov-gateway-email-messages-folders-sync — done — 2026-08-22 04:00
- commit: 75635aae
- gebaut: Neue Testdatei `route_email_folders_messages_sync_test.go` fuer den Rest des
  "Kernpfad Nachrichten/Ordner/Sync" von `route_email.go`. WICHTIGER BEFUND vor dem Bauen:
  die Backlog-Praemisse (Befund 5 im Laufkopf: "23 route_*.go ohne eigene Testdatei") zaehlte
  route_email.go als komplett ungetestet. Das stimmt nicht — `route_email_accounts_test.go`
  und `route_email_compose_test.go` (Lauf 8, Commits b46fc88b/071b5bf1, beide bereits in main
  gemergt) decken Accounts, Send/Compose, Bulk-Action, Move/Delete/MarkRead bereits vollstaendig
  ab, `route_email_rules_test.go`/`route_email_labels_test.go` decken Rules/Labels teilweise
  (28,6–44,4 %) ab. `go tool cover -func` auf route_email.go VOR dem Schreiben gezogen, um die
  echte Luecke zu finden statt zu raten: 0,0 % waren nur noch ServiceName (trivial),
  HandleListFolders, HandleGetFolder, HandleSyncFolders, HandleGetMessage,
  HandleGetThreadMessages, HandleTriggerSync, HandleGetSyncStatus, HandleSetReadFlag,
  HandleUploadAttachment, HandleGetAttachmentDownloadURL — genau die zehn Handler plus
  ServiceName, die die neue Datei jetzt abdeckt (Muster: ServiceUnavailable [502, siehe unten],
  InvalidJSON wo decodeAndValidate/json.Decode greift, ReachesRPC [503]).
  Test-Funktionsnamen sind mit `Email` praefixiert (`TestEmailHandleGetFolder_...` statt
  `TestHandleGetFolder_...`), weil route_bexio.go/route_document.go/route_wiki.go Handler mit
  identischen Methodennamen (TriggerSync, GetSyncStatus, GetFolder, ListFolders,
  UploadAttachment) auf eigenen Route-Structs deklarieren und Go-Testfunktionsnamen
  paketweit eindeutig sein muessen — die urspruengliche Fassung dieser Datei kollidierte
  deshalb beim ersten Anlauf mit sieben bestehenden Dateien (dokumentiert im
  Diagnostics-Log dieser Iteration) und wurde vor dem Commit korrigiert.
- gate: build ok (`-p 2` gateway/..., email/..., cmd/gateway/...) | vet ok | lint ok
  (0 issues) | test ok (neue Datei einzeln gruen: 21/21 inkl. TestEmailRoutes_ServiceName,
  ganzes `internal/gateway` gruen inkl. `TestOpenAPIRouteDrift` [836 Routen gegen 838
  dokumentierte Pfade, unveraendert — keine neue Route], `internal/email/...` komplett gruen,
  DATABASE_URL gegen kmuhub_app, 0 SKIP im Gateway-Paketlauf)
- coverage: internal/gateway 49,1 % -> 49,6 % (eigene Messung vor/nach genau dieser
  Testdatei im selben Arbeitsbaum, Startwert deckt sich mit dem in Iteration 24 gemessenen
  Endstand). internal/email/sync bleibt unveraendert bei 34,6 % — siehe offen:.
- mutations-probe: In `HandleUploadAttachment` die Fehlermeldung des `file is required`-Zweigs
  auf "no file field required" geaendert (Wortreihenfolge vertauscht, damit die
  Substring-Pruefung wirklich bricht — ein erster Versuch mit einem reinen Praefix
  "MUTATION_PROBE file is required" blieb faelschlich gruen, weil assertErrorContains nur auf
  Teilstring prueft. Daraus gelernt und dokumentiert). `TestEmailHandleUploadAttachment_
  MissingFile` wird rot ("error = \"no file field required\", want to contain
  \"file is required\""). Zurueckgedreht -> `git diff --stat` zeigt fuer `route_email.go`
  0 Zeilen Diff.
- verify vorgaenger: sauber. `1dc3368f` (Iteration 24, Lizenz/Abo-Coverage) gegen alle acht
  Fehlerklassen geprueft: `git show --stat` zeigt nur `BACKLOG.yml`, `JOURNAL.md` und eine neue
  reine Testdatei (`route_settings_license_subscription_test.go`, +175 Zeilen). Grep auf
  "Unimplemented|TODO|t.Skip|RequirePermission|.proto|client\." liefert null Treffer auf der
  neuen Testdatei, kein gRPC-Bypass, kein Stub, kein `.proto` angefasst, keine neue
  `RequirePermission`, keine neue Tabelle, keine Wire-Shape-Aenderung, kein Guard ersetzt,
  keine neue Route.
- neue-units: fix-gateway-email-service-unavailable-status-code. Fund: route_email.go weicht in
  allen 58 Client-Fetch-Fehlerpfaden vom Rest des Gateways ab — statt des gemeinsamen
  `respondServiceUnavailable`-Helpers (503, ueberall sonst verwendet) schreibt jeder Handler von
  Hand `response.Error(w, http.StatusBadGateway, "email service unavailable")` (502). Innerhalb
  der Datei konsistent, aber sowohl gegen die Konvention des restlichen Gateways als auch in
  sich zweideutig: "Service nicht registriert" -> 502, "Service registriert, RPC schlaegt mit
  codes.Unavailable fehl" -> 503 ueber respondGRPCError, zwei Codes fuer dieselbe Bedeutung.
  Coverage-Units aendern kein Verhalten, deshalb als eigene Fix-Unit angelegt statt nebenbei
  behoben; die neuen Tests in dieser Iteration pruefen bewusst das TATSAECHLICHE Verhalten (502
  via eigenem `testEmailServiceUnavailable`-Helper), nicht das gewuenschte.
  Ausserdem B13 (`cov-gateway-email-contact-linking-import-export`) mit einer Korrektur-Notiz
  versehen: die dort noch offene Rules/Labels-Annahme ist teilweise ueberholt (siehe oben),
  echte 0,0-%-Luecke ist nur noch Contact-Links + Import/Export CSV/vCard.
- offen:
  - `internal/email/sync` (34,6 % Coverage_start) konnte NICHT vertieft werden: bis auf die
    bereits bestehenden Tests (unconnected-IMAPClient-Fehlerpfade in worker_state_test.go) haengt
    praktisch jede verbleibende Funktion (syncFolder-Erfolgspfad, syncCycle, Run, idleLoop,
    pollLoop, alle IMAPClient-Methoden) hinter einer echten IMAP-Verbindung — `syncFolder` selbst
    scheitert schon an der ersten Zeile (`client.SelectFolder`) ohne Netzwerk, es gibt keine
    IMAPClient-Interface-Abstraktion zum Faken. Das ist keine Coverage-Luecke, die sich mit mehr
    Tests schliessen liesse, sondern eine Netzwerkgrenze — ein Fake-IMAP-Server waere eine neue
    Testinfrastruktur-Investition, keine Coverage-Unit. "Abbruch mitten in der Synchronisation"
    aus dem done_when ist am naechstmoeglichen Punkt abgedeckt: HandleTriggerSync/
    HandleGetSyncStatus/HandleSetReadFlag zeigen jetzt, dass ein Sync-Fehlschlag (RPC
    Unavailable) sauber als 503 durchgereicht wird statt eines 500 mit internem Fehlertext.
  - "Fremder Tenant liefert 404" fuer eine Nachrichten-ID ist NICHT ueber einen Live-RPC-
    Roundtrip getestet (kein bufconn-/Fake-Client-Harness fuer EmailServiceClient im Paket, wie
    bei jeder vorigen Coverage-Unit dieses Laufs). Stattdessen auf die bereits bestehende
    DB-Probe verwiesen: `TestPostgresRepository_CreateAndGetByID`/"GetByID is tenant-scoped" in
    internal/email/message/postgres_repository_test.go seedet eine Nachricht unter Tenant A und
    zeigt, dass `GetByID(id, tenantOther)` `ErrMessageNotFound` liefert.
  - Kein DB-Gate ausser dem regulaeren Testlauf noetig (keine Migration, keine Tabelle/Policy
    beruehrt).

## Iteration 26 — cov-gateway-email-signatures-templates — done — 2026-08-22 04:22
- commit: 143108a0
- gebaut: Neue Testdatei `route_email_signatures_templates_test.go` fuer den dritten Teil von
  `route_email.go`: Signaturen und Vorlagen (12 Handler: HandleListSignatures,
  HandleGetSignature, HandleCreateSignature, HandleUpdateSignature, HandleDeleteSignature,
  HandleSetDefaultSignature, HandleListEmailTemplates, HandleGetEmailTemplate,
  HandleCreateEmailTemplate, HandleUpdateEmailTemplate, HandleDeleteEmailTemplate,
  HandleRenderEmailTemplate) — mit `go tool cover -func` vorab gegen die tatsaechliche 0,0-%-Liste
  geprueft, keine Annahme geraten. Muster wie Iteration 25: ServiceUnavailable (502, ueber den
  bestehenden `testEmailServiceUnavailable`-Helper), InvalidJSON/Validation (400) wo
  decodeAndValidate greift, ReachesRPC (503). Zusaetzlich eine neue Testdatei-Zeile in
  `internal/email/template/service_test.go`:
  `TestRender_SelfReferentialValueGetsChainResolved` pinnt das TATSAECHLICHE Verhalten von
  `Service.Render` — die Substitutionsschleife laeuft `AllowedPlaceholders` in fester Reihenfolge
  durch und ruft je Key einmal `strings.ReplaceAll` auf; ein Wert fuer einen frueher verarbeiteten
  Key, der selbst wie der Token eines spaeter verarbeiteten Keys aussieht (`contact_first_name`
  vor `today`), wird deshalb auf der spaeteren Iteration doch noch aufgeloest — die Substitution
  ist trotz des Kommentars "keine allgemeine Template-Engine" kein einziger nicht-rekursiver
  Durchlauf. Live am Testlauf verifiziert (Wert "{{today}}" fuer contact_first_name wird durch den
  Wert von "today" ersetzt). Keine Eskalation, weil der Aufrufer beide Werte ohnehin selbst im
  "values"-Objekt mitliefert — nichts wird offengelegt, was er nicht schon hatte — aber eine reale
  Abweichung vom im Code versprochenen Design. Nicht gefixt (Coverage-Units aendern kein
  Verhalten), im Testkommentar und hier dokumentiert.
  HTML-Sanitizing-Pfad benannt (done_when-Pflicht): weder `internal/email/signature` noch
  `internal/email/template` sanitisiert `HTMLContent`/`BodyHtml` beim Schreiben oder Lesen — beide
  reichen den Wert unveraendert durch (per Lesen von `service.go` in beiden Paketen bestaetigt,
  kein sanitiz*/dompurify/bluemonday-Import). Auf der Desktop-Seite ruft
  `EmailTemplateDialog.tsx` `sanitizeHtml()` (DOMPurify-Wrapper, `lib/sanitize.ts`) fuer die
  eigene Vorlagen-Vorschau auf — aber `ComposeInline.tsx`/`ComposeModal.tsx` uebergeben beim
  tatsaechlichen Auswaehlen einer Vorlage (`handleTemplateSelect`) `tmpl.body` roh an `setBody()`,
  ohne `sanitizeHtml`-Aufruf. Nur die Dialog-Vorschau ist sanitisiert, nicht der Wert, der am Ende
  im Compose-Editor landet. Das ist eine reine Frontend-Luecke — Frontend/Desktop ist in diesem
  Lauf gesperrt, ein Backend-Fix kann sie nicht schliessen, deshalb hier dokumentiert statt als
  Unit angelegt.
- gate: build ok (`-p 2` gateway/..., email/..., cmd/gateway/...) | vet ok | lint ok (0 issues) |
  test ok (neue Gateway-Testdatei einzeln gruen: 33/33 `TestEmail*`-Faelle inkl. der 26 aus
  Iteration 25, `internal/gateway` komplett gruen inkl. `TestOpenAPIRouteDrift`, `internal/email/...`
  komplett gruen [12 Pakete], DATABASE_URL gegen kmuhub_app, 0 SKIP) | migration n.a. (keine
  Schemaaenderung) | rls-smoke n.a. (keine neue Tabelle/Policy beruehrt)
- coverage: internal/gateway 49,6 % -> 50,1 % (eigene Messung vor/nach genau dieser Iteration im
  selben Arbeitsbaum, Startwert deckt sich mit dem in Iteration 25 gemessenen Endstand, nicht mit
  dem veralteten `coverage_start` der Unit von 46,1 %). internal/email/signature bleibt bei
  43,0 % unveraendert (`coverage_start` deckt sich exakt) — die neuen Tests liegen alle im
  gateway-Paket und brechen vor dem echten Signature-Service ab (ReachesRPC scheitert am
  Transport, nicht am Service), decken also keine zusaetzliche Zeile in `internal/email/signature`
  ab; das ist erwartungsgemaess, dieselbe Grenze wie bei jeder anderen `cov-gateway-*`-Unit dieses
  Laufs. internal/email/template bleibt bei 79,1 % unveraendert — die neue
  `TestRender_SelfReferentialValueGetsChainResolved` deckt ausschliesslich bereits von
  `TestRender_SubstitutesOnlyAllowedPlaceholders` erreichte Zeilen erneut ab (derselbe Schleifenkoerper,
  ein anderer Eingabewert), keine neue Verzweigung.
- mutations-probe: `setDefaultSignatureDTO.UserID` das `validate:"required,uuid"`-Tag entfernt ->
  `TestEmailHandleSetDefaultSignature_MissingUserID` und `_InvalidUserID` werden rot (400 statt
  der erwarteten Validierungsantwort, `ReachesRPC` bleibt gruen). Zurueckgedreht -> gruen
  (`go test ./internal/gateway/` komplett), `git diff --stat` zeigt fuer `route_email.go` 0 Zeilen
  Diff.
- verify vorgaenger: sauber. `75635aae` (Iteration 25, Messages/Folders/Sync-Coverage) gegen alle
  acht Fehlerklassen geprueft: `git show --stat` zeigt nur `BACKLOG.yml`, `JOURNAL.md` und eine
  neue reine Testdatei (`route_email_folders_messages_sync_test.go`, +266 Zeilen). Grep auf
  "Unimplemented|TODO|t\.Skip|RequirePermission|\.proto" liefert null Treffer, kein gRPC-Bypass,
  kein Stub, kein `.proto` angefasst, keine neue `RequirePermission`, keine neue Tabelle, keine
  Wire-Shape-Aenderung, kein Guard ersetzt, keine neue Route.
- neue-units: keine
- offen:
  - Das HTML-Sanitizing-Frontend-Luecke (siehe gebaut:) ist bewusst NICHT als Unit angelegt, weil
    Frontend/Desktop fuer diesen Lauf komplett gesperrt ist und der fehlende `sanitizeHtml()`-Aufruf
    in `handleTemplateSelect` (ComposeInline.tsx/ComposeModal.tsx) sich nicht mit einer
    Backend-Aenderung schliessen laesst. Kandidat fuer eine Frontend-Session oder eine spaetere
    Entscheidungsrunde (Muster: `BACKLOG-PARKED.yml` "GEPARKT — Frontend-Arbeit").
  - Der Render-Chain-Resolve-Fund (siehe gebaut:) ist eine reale, aber nicht eskalierende
    Abweichung vom Non-Recursive-Design-Kommentar in `internal/email/template/service.go`. Kein
    Fix-Unit-Kandidat, weil der Aufrufer alle beteiligten Werte im selben Request bereits selbst
    liefert — reine Dokumentation der Ist-Situation fuer eine kuenftige Refactoring-Entscheidung
    (z. B. ein einziger Durchlauf mit vorab gesammelten Ersetzungen statt sequenziellem
    ReplaceAll).
  - Damit ist `route_email.go` (63 Funktionen) mit B11+B12 in diesem Lauf vollstaendig auf
    Kernpfad + Signaturen/Vorlagen abgedeckt; `cov-gateway-email-contact-linking-import-export`
    (dritter/letzter Teil: Kontakt-Verknuepfung, Import/Export) ist die naechste offene
    `todo`-Unit im Block.
  - Kein DB-Gate ausser dem regulaeren Testlauf noetig (keine Migration, keine Tabelle/Policy
    beruehrt).

## Iteration 27 — cov-gateway-email-contact-linking-import-export — done — 2026-08-22 04:31
- commit: 482503ac
- gebaut: Neue Testdatei `route_email_contact_links_import_export_test.go` fuer den
  dritten/letzten Teil von `route_email.go`: Nachricht-Kontakt-Verknuepfung
  (HandleGetEmailContactLinks, HandleLinkEmailToContact, HandleUnlinkEmailFromContact,
  HandleGetContactEmails) sowie Kontakt-Import/Export (HandleImportContactsCSV,
  HandleImportContactsVCard, HandleExportContactsCSV, HandleExportContactsVCard) — mit
  `go tool cover -func` vorab gegen die tatsaechliche 0,0-%-Liste geprueft (deckt sich exakt
  mit der Korrektur-Notiz der Unit). Muster wie B11/B12: ServiceUnavailable (502, bestehender
  `testEmailServiceUnavailable`-Helper), InvalidJSON (400) wo `json.NewDecoder(r.Body).Decode`
  greift, ReachesRPC (503).
  Zusaetzlich ein DB-Test in `internal/server/email_grpc_export_test.go`:
  `TestEmailExportContactsCSV_ExcludesOtherTenantContact` erfuellt das explizite
  done_when "Export mit zwei geseedeten Tenants". Seedet je einen Kontakt in zwei
  unabhaengigen Tenants ueber den bereits bewiesenen Import-Pfad, exportiert beide
  Kontakt-IDs im Kontext des ersten Tenants und belegt: eigener Kontakt erscheint, fremder
  nicht. WAEHREND DER MUTATIONS-PROBE ENTDECKT (siehe unten): der App-Layer-Tenant-Filter in
  `ListByIDs` (`postgres_repository.go:301`) ist nicht die alleinige Verteidigung — RLS auf
  `contacts` faengt einen fremden Tenant unabhaengig vom WHERE-Filter ab (Entfernen von
  `AND tenant_id = $2` liess den Test gruen bleiben). Kein Sicherheitsloch, sondern
  Defense-in-Depth; im Journal dokumentiert, weil die erste Mutationszeile deshalb keine
  aussagekraeftige Probe war und durch eine zweite ersetzt wurde (siehe mutations-probe:).
  CSV-FORMEL-INJEKTION (echter Fund, nicht gefixt): `internal/email/contact/export_service.go`
  `ExportCSV` (Zeile 86-131) schreibt first_name/last_name/email/phone/company/position/notes
  unveraendert in CSV-Zellen — kein Praefix-Check auf `=`/`+`/`-`/`@`. Ein Kontakt mit
  entsprechendem Namens- oder Notizfeld fuehrt beim Oeffnen der Export-Datei in Excel/
  LibreOffice eine Formel aus. Als Fix-Unit `fix-email-contacts-csv-export-formula-injection`
  ans Backlog-Ende angehaengt (Coverage-Units aendern kein Verhalten, siehe Regel 2 des
  Laufkopfs) statt mit einem Test gefixt, der die Luecke nur festschreiben wuerde.
  `ExportVCard` (gleiche Datei) ist nicht betroffen — vCard-Felder werden nicht als
  Tabellenkalkulations-Formeln interpretiert, in der Fix-Unit gegengeprueft statt angenommen.
  Tenant-Isolation der Verknuepfungsroute (`email_contact_links`) ist bereits an der
  Repository-Ebene bewiesen (`internal/email/contactlink/tenant_isolation_test.go`,
  `TestTenantIsolation_EmailContactLinks_DB`) — die vier neuen Gateway-Handler sind reine
  Pass-Throughs zum gRPC-Client und fuegen dort nichts hinzu, das isoliert zu testen waere.
- gate: build ok (`-p 2` gateway/..., server/..., email/..., cmd/gateway/...) | vet ok
  (gateway/..., server/..., email/...) | lint ok (0 issues, gateway/... und server/...) |
  test ok (neue Gateway-Testdatei: 20/20 `TestEmail*`-Faelle gruen, `internal/gateway`
  komplett gruen inkl. `TestOpenAPIRouteDrift`, `internal/email/...` komplett gruen
  [12 Pakete], `internal/server` komplett gruen inkl. der neuen Tenant-Isolation-Probe,
  DATABASE_URL gegen kmuhub_app, 0 SKIP) | migration n.a. (keine Schemaaenderung) |
  rls-smoke n.a. (keine neue Tabelle/Policy beruehrt — RLS-Wirkung auf `contacts` nur
  gegengeprueft, nicht veraendert)
- coverage: internal/gateway 50,1 % -> 50,5 % (eigene Messung vor/nach genau dieser
  Iteration im selben Arbeitsbaum; Startwert deckt sich mit dem in Iteration 26 gemessenen
  Endstand, nicht mit dem veralteten `coverage_start` der Unit von 46,1 %).
  internal/email/contactlink bleibt bei 35,0 % unveraendert (`coverage_start` deckt sich
  exakt) — die vier neuen Verknuepfungs-Handler-Tests liegen im gateway-Paket und brechen vor
  dem echten Service ab (ReachesRPC scheitert am Transport), decken also keine zusaetzliche
  Zeile in `internal/email/contactlink` ab; dieselbe erwartungsgemaesse Grenze wie bei jeder
  anderen `cov-gateway-*`-Unit dieses Laufs. internal/server (nicht `coverage_start` der
  Unit, aber vom neuen DB-Test beruehrt): 70,3 % nach dieser Iteration gemessen, kein
  Vorher-Wert dieser Iteration verfuegbar (keine fruehere Unit dieses Laufs hat dieses Paket
  als Ziel gemessen) — als Kontext genannt, nicht als Beitrag dieser Unit ausgewiesen.
- mutations-probe: ERSTE PROBE VERWORFEN (siehe gebaut:) — `AND tenant_id = $2` in
  `ListByIDs` (`postgres_repository.go:301`) zu `FROM contacts WHERE id = ANY($1)` (Filter
  entfernt) gebrochen -> `TestEmailExportContactsCSV_ExcludesOtherTenantContact` blieb
  GRUEN, weil RLS auf `contacts` unabhaengig vom App-Filter greift. Zurueckgedreht (`git diff`
  auf die Datei zeigte danach 0 Zeilen). ZWEITE, aussagekraeftige PROBE: dieselbe Zeile zu
  `AND tenant_id != $2` gebrochen (Filter invertiert, schliesst den eigenen Tenant aus statt
  ihn einzuschliessen) -> Test wird ROT (`does not contain "own-tenant@email-export.test.local"`,
  0 Kontakte im Export). Zurueckgedreht -> gruen (`go test ./internal/server/
  ./internal/crm/contact/... ./internal/gateway/` komplett), `git diff --stat` auf
  `postgres_repository.go` zeigt 0 Zeilen Diff.
- verify vorgaenger: sauber. `143108a0` (Iteration 26, Signaturen/Vorlagen-Coverage) gegen
  alle acht Fehlerklassen geprueft: `git show --stat` zeigt nur `BACKLOG.yml`, `JOURNAL.md`,
  eine neue reine Testdatei (`route_email_signatures_templates_test.go`, +335 Zeilen) und
  eine neue Testzeile in `internal/email/template/service_test.go` (+35 Zeilen). Grep auf
  "Unimplemented|TODO|t\.Skip\(|RequirePermission|\.proto" liefert null Treffer, kein
  gRPC-Bypass, kein Stub, kein `.proto` angefasst, keine neue `RequirePermission`, keine neue
  Tabelle, keine Wire-Shape-Aenderung, kein Guard ersetzt, keine neue Route.
- neue-units: fix-email-contacts-csv-export-formula-injection (CSV-Formel-Injection in
  `internal/email/contact/export_service.go`, Details siehe gebaut:)
- offen:
  - `fix-email-contacts-csv-export-formula-injection` ist die einzige neue Unit dieses Laufs
    und sicherheitsrelevant — sollte vor Block C priorisiert werden, nicht am Ende der Queue
    verhungern.
  - Damit sind alle drei `cov-gateway-email-*`-Units aus Block B abgeschlossen
    (Messages/Folders/Sync, Signaturen/Vorlagen, Kontakt-Verknuepfung/Import/Export);
    `route_email.go` ist vollstaendig auf Kernpfad-Ebene abgedeckt.
  - Kein DB-Gate ausser dem regulaeren Testlauf noetig (keine Migration, keine Tabelle/Policy
    beruehrt). Die RLS-Defense-in-Depth-Beobachtung (siehe gebaut:/mutations-probe:) ist
    dokumentiert, kein Fund, der eine eigene Unit braucht.

## Iteration 28 — cov-gateway-crm-advisory-protocols — done — 2026-08-22 04:46
- commit: 2ac82165
- gebaut: Neue Testdatei `route_crm_advisory_test.go` fuer `route_crm_advisory.go` (362 Zeilen,
  9 Funktionen, bisher ohne Testdatei): ServiceUnavailable fuer alle 8 client-aufrufenden Handler
  (RegisterAdvisoryRoutes ist keine eigene HTTP-Funktion, wird bereits von
  `TestOpenAPIRouteDrift` mitregistriert und geprueft), InvalidUUID fuer contactId/id, InvalidJSON
  und Validation (fehlender Advisor, Risikoklasse ausserhalb 1-7, self_assessment ausserhalb 1-5)
  fuer HandleUpdateAdvisoryProtocol, sowie ReachesRPC (503) fuer alle Handler nach den
  Vor-RPC-Pruefungen. Die beiden verhaltensbezogenen done_when-Punkte sind NICHT als
  Gateway-Fakes neu gepinnt, sondern auf bereits existierende, staerkere Beweise verwiesen (im
  Dateikopf dokumentiert): Cross-Tenant-Read-404 ist bereits in
  `internal/server/crm_grpc_advisory_test.go` (`TestGetAdvisoryProtocol/"wrong tenant is treated
  as not found"`) gegen `advisoryprotocol.Service.GetByID` bewiesen; die Kontakt-Loeschung bei
  haengendem Protokoll (409 mit Grund) hatte dagegen KEINE Real-DB-Probe — nur eine
  Mock-Repository-Variante in `contact/service_test.go` (`TestService_Delete_InUse`), die die
  tatsaechliche `advisory_protocols`-SQL-Abfrage nie ausfuehrt. Dafuer neuer DB-Test
  `TestRepository_IsInUse_AdvisoryProtocolBlocksDeletion_DB` in
  `internal/crm/contact/postgres_repository_db_test.go`: seedet einen Kontakt, prueft `IsInUse` ==
  false, seedet einen echten `advisory_protocols`-Datensatz (RESTRICT-FK, Migration 000137), prueft
  `IsInUse` == true mit Grund "advisory protocols", ruft danach `Service.Delete` auf und belegt
  `ErrContactInUse` mit demselben Grundtext sowie dass der Kontakt NICHT geloescht wurde (die
  RESTRICT-Constraint wird nie erreicht, weil der Service-Check vorher greift). Die 409-Zuordnung
  selbst (`ErrContactInUse` -> `codes.FailedPrecondition` -> HTTP 409) war bereits vorher in
  `internal/server/crm_grpc.go`/`internal/gateway/helpers.go` verdrahtet und ungeaendert.
  ECHTER FUND waehrend des Bauens (nicht gefixt, siehe neue-units): `updateAdvisoryProtocolRequest.
  Products` (`route_crm_advisory.go:152`) traegt kein `dive` im `validate`-Tag, weshalb
  `advisoryProduct.RiskClass`s eigenes `min=1,max=7` nie ausgefuehrt wird (go-playground/validator
  rekursiert nur mit `dive` in eine Slice von Structs — Referenzmuster mit `dive` existiert bereits
  in `route_customization.go:473`). Ein Produkt mit `risk_class: 0` erreicht daher unvalidiert die
  RPC. Test `TestHandleUpdateAdvisoryProtocol_ProductRiskClassNotValidated` pinnt das aktuelle
  (fehlerhafte) 503-Verhalten bewusst als Ist-Zustand, mit Verweis auf die neue Fix-Unit im
  Testkommentar.
- gate: build ok (`-p 2` gateway/..., crm/..., server/..., cmd/gateway/..., cmd/crm/...) | vet ok
  (gateway/..., crm/..., server/...) | lint ok (0 issues, gateway/..., crm/..., server/...) |
  test ok (`internal/gateway` komplett gruen inkl. TestOpenAPIRouteDrift [836 Routen gegen 838
  dokumentierte Pfade, unveraendert], 0 SKIP; `internal/crm/...` komplett gruen mit `-p 1`
  [Parallel-Lauf ueber alle CRM-Pakete sprengt lokal die Postgres-Verbindungsgrenze, kein Befund
  an meinem Code — siehe Iteration 2]; `internal/server/...` komplett gruen; DATABASE_URL gegen
  kmuhub_app, 0 SKIP in beiden neu beruehrten Paketen) | migration n.a. (keine Schemaaenderung,
  advisory_protocols existiert seit Migration 000137) | rls-smoke n.a. (keine neue Tabelle/Policy;
  Tenant-Scoping der neuen Query ist Teil des neuen DB-Tests selbst, kein zusaetzlicher
  Cross-Tenant-Fall noetig, da IsInUse bereits tenant-gescoped filtert und das nicht Gegenstand
  dieser Unit war)
- coverage: internal/gateway 50,5 % -> 50,9 % (eigene Messung per `git stash -u` gegen den
  Vorgaenger-Commit, danach `git stash pop`; deckt sich mit dem in Iteration 27 protokollierten
  Endstand, nicht mit dem veralteten `coverage_start` der Unit von 46,1 %). internal/crm/contact
  80,4 % -> 81,4 % (dieselbe Stash-Messung). internal/crm/advisoryprotocol unveraendert bei 65,5 %
  (deckt sich mit `coverage_start` — diese Unit hat dort keine neue Testdatei angelegt, das
  Paket war schon vor dieser Iteration gut abgedeckt durch `crm_grpc_advisory_test.go`).
- mutations-probe: ZWEI Proben, je eine pro neuer Testdatei. (1) In
  `internal/crm/contact/postgres_repository.go` `IsInUse` das
  `EXISTS(SELECT 1 FROM advisory_protocols ...)` durch `false` ersetzt ->
  `TestRepository_IsInUse_AdvisoryProtocolBlocksDeletion_DB` wird rot ("IsInUse (with protocol) =
  false, want true"). Zurueckgedreht -> gruen (`go test ./internal/crm/contact/...`), `git diff
  --stat` zeigt 0 Zeilen. (2) In `route_crm_advisory.go` `HandleCreateAdvisoryProtocol` den
  `validateUUIDParam`-Aufruf durch ein ungeprueftes `chi.URLParam(r, "contactId")` ersetzt ->
  `TestHandleCreateAdvisoryProtocol_InvalidContactID` wird rot (503 statt 400, RPC mit
  "not-a-uuid" erreicht statt vorher abgefangen). Zurueckgedreht -> gruen (`go test
  ./internal/gateway/`), `git diff --stat` zeigt 0 Zeilen fuer `route_crm_advisory.go`.
- verify vorgaenger: sauber. `482503ac` (Iteration 27, Kontakt-Verknuepfung/Import-Export) gegen
  alle acht Fehlerklassen geprueft: `git show --stat` zeigt nur `BACKLOG.yml`, `JOURNAL.md`, eine
  neue reine Gateway-Testdatei und eine neue Testfunktion in
  `internal/server/email_grpc_export_test.go` — kein gRPC-Bypass, kein Stub/TODO, kein `.proto`
  angefasst, keine neue `RequirePermission`, keine neue Tabelle, keine Wire-Shape-Aenderung, kein
  Guard ersetzt, keine neue Route. Der Folge-Commit `3bfffe35` ist reines Journal-SHA-Nachtragen
  (1 Zeile), kein Code.
- neue-units: fix-gateway-advisory-product-riskclass-not-validated (fehlendes `dive` auf
  `updateAdvisoryProtocolRequest.Products` verhindert die Validierung von `Product.RiskClass`,
  Details siehe gebaut:)
- offen:
  - `fix-email-contacts-csv-export-formula-injection` (aus Iteration 27, sicherheitsrelevant)
    steht weiterhin unbearbeitet im Backlog — diese Iteration hat sie nicht gezogen, weil der
    Ablauf die erste `todo`-Unit mit erfuellten deps in Dateireihenfolge verlangt (Schritt 2 von
    ITERATION.md) und nicht nach Schweregrad umsortiert. Fuer Luke: beide neuen Fix-Units
    (CSV-Injection und die hier gefundene Produkt-Risikoklassen-Luecke) stehen jetzt am
    Backlog-Ende und sollten vor Block C gezogen werden, falls das gewuenscht ist.
  - `chi` wird in `route_crm_advisory_test.go` NICHT importiert (nur fuer die temporaere
    Mutations-Probe gebraucht) — keine Aufraeumarbeit noetig, die Probe wurde vollstaendig
    zurueckgedreht.
  - Kein DB-Gate ausser dem regulaeren Testlauf und dem neuen DB-Test noetig (keine Migration,
    keine Tabelle/Policy beruehrt).

## Iteration 29 — cov-gateway-crm-companies — done — 2026-08-22 04:53
- commit: e679cad4
- gebaut: `internal/gateway/route_crm_companies_test.go` neu — deckt alle 6 Handler aus
  `route_crm_companies.go` ab (Create, Get, List, Update, Delete, GetCompanyContacts). Ein Teil
  war schon in `route_crm_test.go` gestreut (Create ServiceUnavailable/NoUserID/InvalidJSON/
  MissingName, Get InvalidUUID) — diese wurden NICHT dupliziert (Buildfehler durch doppelte
  Funktionsnamen zeigte das sofort), sondern im Header der neuen Datei referenziert. Neu
  hinzugekommen: ServiceUnavailable fuer die vier bislang ungetesteten Handler (List/Update/
  Delete/GetCompanyContacts), Create InvalidTagID (dive+uuid-Validierung auf `tag_ids`),
  Update InvalidID/InvalidJSON/BlankNameRejected, Delete InvalidID, GetCompanyContacts InvalidID,
  sowie je ein ReachesRPC-Test pro Handler.
  Loeschverhalten bei haengenden Kontakten gegen `pg_constraint` geprueft statt vermutet: die FK
  `contacts.company_id -> companies(id)` ist `ON DELETE SET NULL` (Migration 000007) — anders als
  bei `contacts` selbst gibt es HIER keine RESTRICT-FK auf dieser Kante. Der 409-Block kommt
  ausschliesslich aus `company.Service.Delete`s eigenem `HasContacts`-Check (`ErrCompanyInUse`),
  bereits vollstaendig getestet in `internal/crm/company/service_test.go` und
  `postgres_repository_db_test.go` (`TestService_Delete_HangingContactsBlockDeletionButForeign
  TenantContactDoesNot`). Die einzige RESTRICT-foermige FK auf `companies` ist die
  selbstreferenzielle `merged_into_id` (Migration 000059, kein `ON DELETE` = NO ACTION) — sie
  haengt aber an einem eigenen Merge-Endpunkt (`route_crm_ext.go:118` `HandleMergeCompanies`,
  Route `/api/v1/companies/merge`), nicht an `HandleDeleteCompany`; per Grep bestaetigt, keine
  Fix-Unit noetig. Cross-Tenant-Isolation ist bereits per RLS-Test belegt
  (`internal/crm/company/rls_test.go`) und per tenant-gescopter Query in
  `PostgresRepository.GetByID` (`WHERE id = $1 AND tenant_id = $2`) — keine neue Duplizierung
  auf Gateway-Ebene noetig, wie schon in Iteration 28 fuer Advisory Protocols begruendet.
  Berechtigungsgrenze (403) liegt am Router (`route_crm.go:51-54`,
  `RequirePermissionAny{"companies","..."} | {"crm:contact","..."}`), nicht im Handler — wie bei
  Advisory ist das kein Testfall auf Handler-Ebene.
- gate: build ok (`-p 2` gateway/..., crm/..., cmd/gateway/..., cmd/crm/...) | vet ok
  (gateway/..., crm/...) | lint ok (0 issues, gateway/..., crm/...) | test ok (`internal/gateway`
  komplett gruen inkl. TestOpenAPIRouteDrift [836 Routen gegen 838 dokumentierte Pfade,
  unveraendert — keine neue Route], 0 SKIP; `internal/crm/...` komplett gruen mit `-p 1`
  [Parallel-Lauf ueber alle CRM-Pakete sprengt lokal die Postgres-Verbindungsgrenze, kein Befund
  an meinem Code], 0 SKIP in `internal/crm/company`; DATABASE_URL gegen kmuhub_app) | migration
  n.a. (keine Schemaaenderung) | rls-smoke n.a. (keine neue Tabelle/Policy; bestehender
  `rls_test.go` deckt `companies` bereits ab, nicht Gegenstand dieser Unit)
- coverage: internal/gateway 50,9 % -> 51,2 % (eigene Messung per `git stash -u` gegen den
  Vorgaenger-Commit, danach `git stash pop`; deckt sich mit dem in Iteration 28 protokollierten
  Endstand, nicht mit dem veralteten `coverage_start` der Unit von 46,1 %). internal/crm/company
  unveraendert (diese Unit hat dort keine neue Testdatei angelegt, nur die bestehenden Service-/
  Repository-Tests als Beleg referenziert).
- mutations-probe: In `route_crm_companies.go` `createCompanyRequest.TagIDs`s Validate-Tag von
  `"omitempty,dive,uuid"` auf `"omitempty"` verkuerzt -> `TestHandleCreateCompany_InvalidTagID`
  wird rot (503 mit gRPC-Connection-Error statt 400 mit `validation_failed` auf `tag_ids[0]`, weil
  die Validierung nicht mehr greift und der Request unvalidiert die RPC erreicht). Zurueckgedreht
  -> gruen (`go test ./internal/gateway/`), `git diff --stat` zeigt 0 Zeilen fuer
  `route_crm_companies.go`.
- verify vorgaenger: sauber. `0bef55d2` (reines Journal-SHA-Nachtragen, 1 Zeile) und `2ac82165`
  (Iteration 28, Advisory-Protocol-Routen) gegen alle acht Fehlerklassen geprueft: `git show
  --stat` zeigt bei `2ac82165` nur `BACKLOG.yml`, `JOURNAL.md`, eine neue reine Gateway-Testdatei
  und eine neue DB-Testfunktion in `postgres_repository_db_test.go` — kein gRPC-Bypass (Handler
  bleiben thin pass-throughs ueber den CRM-Client), kein Stub/TODO, kein `.proto` angefasst, keine
  neue `RequirePermission`, keine neue Tabelle, keine Wire-Shape-Aenderung, kein Guard ersetzt,
  keine neue Route.
- neue-units: keine
- offen:
  - `fix-email-contacts-csv-export-formula-injection` (Iteration 27) und
    `fix-gateway-advisory-product-riskclass-not-validated` (Iteration 28) stehen weiterhin
    unbearbeitet am Backlog-Ende und sollten vor Block C gezogen werden.
  - Kein DB-Gate ausser dem regulaeren Testlauf noetig (keine Migration, keine Tabelle/Policy
    beruehrt in dieser Iteration).

## Iteration 30 — cov-gateway-crm-ext-duplicates-merge-gdpr — done — 2026-08-22 05:01
- commit: 666cade4
- gebaut: `internal/gateway/route_crm_ext_test.go` neu — deckt alle 12 Handler aus
  `route_crm_ext.go` ab (FindContactDuplicates, MergeContacts, FindCompanyDuplicates,
  MergeCompanies, ContactTimeline, ContactDeletionPreview, GetConsents, GrantConsent,
  RevokeConsent, GetConsentHistory, RequestDeletion, ProcessDeletion), diese Datei war die
  groesste ungetestete CRM-Routendatei (389 Zeilen, 16 Funktionen inkl. der drei
  Registrierungshelfer). Kern der Unit ist `TestGDPRDeletionRoutes_RequestAndProcessRequireDifferent
  Guards`: ein Router-Level-Test (echtes `NewCRMRoutes`+`NewCRMExtRoutes`-Wiring, kein Direktaufruf),
  der die Vier-Augen-Trennung zwischen Beantragen (`/contacts/{id}/gdpr/deletion-request`, Guard
  `RequirePermission("contacts","write")`) und Ausfuehren (`/gdpr/deletion-requests/{id}/process`,
  Guard `middleware.RequireRole("admin")`) beweist — insbesondere, dass `contacts:write` ALLEIN das
  Ausfuehren NICHT freischaltet (gleiches Muster wie die bestehende Automation-Stats-Rollenpruefung
  in `route_capability_guard_test.go`). Die Trennung existierte bereits im Code, war nur ungetestet
  — kein Fix noetig.
  Merge-Kollision (kollidierende Tags/Custom-Fields zwischen Primary und Duplicate) und
  Cross-Tenant-/Already-merged-Faelle sind NICHT auf Gateway-Ebene nachgebaut, sondern per
  Kommentar auf bereits bestehende, tiefere Tests verwiesen: `TestRepository_MergeInto_
  ReassignsRelationsMergesTagsAndCustomFieldsThenSoftDeletes` (contact UND company, je eigene
  `postgres_repository_db_test.go`, `ON CONFLICT DO NOTHING` bzw. `ON CONFLICT (contact_id,
  field_id) DO NOTHING`) sowie `TestService_MergeContacts_*`/`TestService_MergeCompanies_*` und
  `TestService_CrossTenant_Merge{Contacts,Companies}_NilTenantRejected`. Ebenso
  `TestService_PreviewDeletion_NotFoundForUnknownOrForeignContact` fuer die Tenant-Scope der
  Loeschvorschau (A2, bereits fertig) und `internal/crm/consent/rls_test.go` +
  `tenant_write_test.go` fuer die komplette Consent-Flaeche.
- gate: build ok (`-p 2` gateway/..., crm/..., security/..., cmd/gateway/..., cmd/crm/...) | vet ok
  (gateway/..., crm/...) | lint ok (0 issues, gateway/...) | test ok (`internal/gateway` komplett
  gruen inkl. TestOpenAPIRouteDrift [836 Routen gegen 838 dokumentierte Pfade, unveraendert — keine
  neue Route], 0 SKIP von `go test -v ... | grep -c "^--- SKIP"`; `internal/crm/contact`,
  `internal/crm/company`, `internal/crm/consent` unveraendert gruen, DATABASE_URL gegen
  kmuhub_app) | migration n.a. (keine Schemaaenderung) | rls-smoke n.a. (keine neue Tabelle/
  Policy)
- coverage: internal/gateway 51,2 % (Iteration-29-Endstand, per `git tag`-freie Referenzmessung im
  vorigen Eintrag) -> 51,8 % (eigene Messung `go tool cover -func` nach dieser Unit)
- mutations-probe: In `route_crm_ext.go` `mergeContactsRequest.PrimaryID`s Validate-Tag von
  `"required,uuid"` auf leer gesetzt -> zwei Tests werden rot
  (`TestHandleMergeContacts_MissingIDs`: erwartete Feld `primary_id` fehlt in den Validierungs-
  Details, weil nur noch `duplicate_id` als required erkannt wird; `TestHandleMergeContacts_
  InvalidUUID`: erwartete 400/`validation_failed` wird zu 503, weil der ungueltige `primary_id`-Wert
  jetzt unvalidiert durchrutscht und die RPC erreicht). Zurueckgedreht -> gruen
  (`go test -run TestHandleMergeContacts ./internal/gateway/`), `git diff --stat` zeigt 0 Zeilen
  fuer `route_crm_ext.go`.
- verify vorgaenger: sauber. `e679cad4` (Iteration 29, CRM-Companies-Routen) gegen alle acht
  Fehlerklassen geprueft: `git show --stat` zeigt nur eine neue reine Gateway-Testdatei plus
  BACKLOG.yml/JOURNAL.md — kein gRPC-Bypass (Handler bleiben thin pass-throughs), kein Stub/TODO,
  kein `.proto` angefasst, keine neue `RequirePermission`, keine neue Tabelle, keine
  Wire-Shape-Aenderung, kein Guard ersetzt, keine neue Route. Testdatei selbst gegengelesen
  (nicht nur `git show --stat`): reine `httptest`-Handler-Aufrufe, keine direkte Service-Injektion.
- neue-units: keine
- offen:
  - `fix-email-contacts-csv-export-formula-injection` (Iteration 27) und
    `fix-gateway-advisory-product-riskclass-not-validated` (Iteration 28) stehen weiterhin
    unbearbeitet am Backlog-Ende und sollten vor Block C gezogen werden — jetzt drei Iterationen
    unbearbeitet.
  - Kein DB-Gate ausser dem regulaeren Testlauf noetig (keine Migration, keine Tabelle/Policy
    beruehrt in dieser Iteration).

## Iteration 31 — cov-gateway-crm-contact-files-wopi — done — 2026-08-22 05:02
- commit: 28caea24
- gebaut: `internal/gateway/route_crm_contact_files_test.go` neu — deckt
  `route_crm_contact_files.go` (7 Funktionen) ab: ServiceUnavailable fuer beide
  Handler, invalide Contact-ID vor jeder RPC, ReachesRPC fuer beide Handler,
  und explizit die Reihenfolge in `HandleCreateContactFile` (Ownership-Check
  laeuft VOR der Body-Validierung: ein kaputtes JSON liefert trotzdem 503, nicht
  400). `toContactFileResponse` ist als reine Funktion getestet (flaches Wire-
  Shape, keine verschachtelte Proto-Struktur). `resolveContactFilesFolder`/
  `findContactFilesFolder`/`getDocumentClient` sind wie in jeder vorigen
  Gateway-Coverage-Unit dieses Laufs NICHT erreichbar, weil es fuer
  CRMServiceClient/DocumentServiceClient keinen Fake-Client/bufconn gibt — die
  Ownership-Pruefung scheitert immer zuerst an der Fake-Verbindung. Der
  tatsaechliche Cross-Tenant-Beweis fuer `verifyContactOwnership` liegt
  serverseitig bereits vor: `internal/crm/contact/service_test.go`
  `TestService_CrossTenant_GetByID_DifferentTenantGetsNotFound` und
  `TestService_GetByID_InvalidTenant` (im Dateikopf referenziert, nicht
  nochmal als Gateway-Fake nachgebaut).
  `internal/gateway/route_wopi_test.go` neu — deckt `route_wopi.go` (3
  Funktionen: ServiceName, RegisterRoutes, NewWOPIRoutes) ab. Anders als jede
  andere Gateway-Route hat WOPI KEINE gRPC-Client-Grenze (die Routen
  umgehen die App-Auth-Middleware komplett, siehe route_wopi.go-Kommentar) —
  deshalb ist hier der volle Stack durchtestbar: echter chi-Router, echter
  `wopi.TokenService`, echter DB-gestuetzter `wopi.LockService` (Migration
  000044/000114/000122, FK auf `document_files`/`users`, RLS via
  `enable_tenant_rls`), Fake `FileServiceInterface`/`chatfile.FileStore` fuer
  die beiden DB-freien Interfaces. `TestWOPIRoutes_Dispatch` beweist, dass alle
  vier Sub-Pfade (GET /, GET /contents, POST /contents, POST / mit
  X-WOPI-Override) auf die richtige Handler-Methode routen und der
  {fileID}-Chi-Param korrekt durchgereicht wird — inklusive eines echten
  LOCK/UNLOCK-Zyklus gegen die Datenbank. Abgelaufenes, manipuliertes und
  fremdes Token sind NICHT hier neu gebaut, sondern bereits vollstaendig in
  `internal/document/wopi/token_test.go`
  (`TestTokenService_Validate_RejectsExpiredToken`,
  `TestTokenService_Validate_RejectsTamperedSignature`) und
  `handler_test.go` ("token file_id mismatch") abgedeckt — im Dateikopf
  referenziert. Ein zusaetzlicher Router-Level-Test fuer den Fremd-Token-Fall
  ("wrong file_id in URL vs token") ist trotzdem enthalten, weil er zugleich
  die {fileID}-Param-Extraktion durch den echten Router beweist, nicht nur
  die Handler-Logik isoliert.
- gate: build ok (`-p 2` gateway/..., document/..., cmd/gateway/...) | vet ok
  (gateway/..., document/...) | lint ok (0 issues, gateway/..., document/...)
  | test ok (`internal/gateway` komplett gruen inkl. TestOpenAPIRouteDrift
  [836 Routen gegen 838 dokumentierte Pfade, unveraendert — keine neue Route],
  0 SKIP von `go test -v ... | grep -c "^--- SKIP"`; `internal/document/...`
  [file, folder, search, share, tag, virtual, wopi] unveraendert gruen,
  DATABASE_URL gegen kmuhub_app) | migration n.a. (keine Schemaaenderung,
  wopi_locks existiert bereits seit Migration 000044/000114/000122) |
  rls-smoke n.a. (keine neue Tabelle/Policy — der LOCK/UNLOCK-Test in
  `TestWOPIRoutes_Dispatch` laeuft bereits unter Tenant-Kontext gegen die
  bestehende RLS-Policy und beweist implizit, dass ein falscher Tenant keine
  Zeile schreiben wuerde, testet aber keinen Fremd-Tenant-Lesefall explizit,
  weil dieser bereits in `lock_test.go` TestLockService_* abgedeckt ist)
- coverage: internal/gateway 51,8 % (Iteration-30-Endstand, eigene Messung
  `go tool cover -func` vor dieser Unit) -> 51,9 % (eigene Messung nach
  dieser Unit)
- mutations-probe: In `route_crm_contact_files.go` `toContactFileResponse`s
  `ContactID: contactID` auf `ContactID: f.Id` geaendert ->
  `TestToContactFileResponse_FlatShape` wird rot (`ContactID = "file-1", want
  "10101010-1010-1010-1010-101010101010"`). Zurueckgedreht -> gruen
  (`go test -run TestToContactFileResponse_FlatShape`), `git diff --stat`
  zeigt 0 Zeilen fuer `route_crm_contact_files.go`.
- verify vorgaenger: sauber. `666cade4` (Iteration 30, CRM-Ext-Duplicates/
  Merge/GDPR-Routen) gegen alle acht Fehlerklassen geprueft: `git diff --stat
  0ca728a1 666cade4 -- backend/` zeigt nur eine neue reine Gateway-Testdatei
  (`route_crm_ext_test.go`, 468 Zeilen) — kein gRPC-Bypass, kein Stub/TODO,
  kein `.proto` angefasst, keine neue `RequirePermission`, keine neue
  Tabelle, keine Wire-Shape-Aenderung, kein Guard ersetzt, keine neue Route.
- neue-units: keine
- offen:
  - `fix-email-contacts-csv-export-formula-injection` (Iteration 27) und
    `fix-gateway-advisory-product-riskclass-not-validated` (Iteration 28)
    stehen weiterhin unbearbeitet am Backlog-Ende und sollten vor Block C
    gezogen werden — jetzt vier Iterationen unbearbeitet.
  - `resolveContactFilesFolder`/`findContactFilesFolder`/`getDocumentClient`
    in `route_crm_contact_files.go` bleiben ohne Fake-CRM/Document-Client
    strukturell ungetestet auf Gateway-Ebene (wie bei jeder anderen
    RPC-basierten Route dieses Laufs) — kein Fix noetig, nur Grenze notiert.
  - Kein DB-Gate ausser dem regulaeren Testlauf noetig (keine Migration,
    keine neue Tabelle/Policy in dieser Iteration; der WOPI-Test nutzt
    ausschliesslich bereits bestehende Tabellen/Policies).

## Iteration 32 — cov-gateway-lexware-integration — done — 2026-08-22 05:13
- commit: 9b54ee7b
- gebaut: `internal/gateway/route_lexware_test.go` neu — deckt `route_lexware.go`
  (610 Zeilen, alle 18 Funktionen) ab: ServiceName, alle elf RPC-Handler
  (Connect/Disconnect/GetConnectionStatus/TestConnection/TriggerSync/
  GetSyncStatus/ListSyncLogs/GetFieldMappings/UpdateFieldMappings/
  PushInvoice/PushQuote) je mit ServiceUnavailable-, NoTenantID- und
  RPCFails-Fall (Vorlage `route_bexio_test.go`, dieselbe Integrationsform).
  Fuer HandleConnect zusaetzlich InvalidJSON und die fehlende Pflichtangabe
  `api_key` (Validation-Error). Fuer HandleTriggerSync der
  ContentLength==0-Kurzschluss (leerer Body darf nicht als ungueltiges JSON
  abgelehnt werden). HandleWebhookEvent (der einzige oeffentliche Pfad,
  ohne Auth-Middleware) deckt alle vier Faelle des HMAC-Gates ab: Secret
  fehlt + Prod (500, Defense-in-Depth), Secret fehlt + Dev (Signatur wird
  uebersprungen, faellt bis zum RPC-Dispatch durch), fehlender
  X-Signature-Header (401), falsche Signatur (401), sowie gueltige Signatur
  mit kaputtem JSON-Body (400) und mit gueltigem Body (503, Dispatch
  erreicht). `parseLexwareWebhookBody` und `firstNonEmpty` sind direkt als
  reine Funktionen getestet: camelCase-Decodierung, snake_case-Fallback,
  camelCase gewinnt bei doppelter Belegung (das ist der Kern der
  Mutations-Probe), und invalides JSON liefert einen Fehler.
  Ein struktureller Test (`TestLexwareResponseProtos_NeverExposeCredentials`,
  Vorlage `TestBexioResponseProtos_NeverExposeOAuthTokens`) sperrt fest,
  dass keine der 14 Lexware-Response-Protos je ein Feld mit "apikey",
  "secret" oder "password" im Namen bekommt — dieselbe Grenze wie bei
  Bexio, nur dass Lexware kein OAuth macht, sondern einen rohen API-Key
  entgegennimmt (`ConnectLexwareRequest.ApiKey`), der serverseitig
  gespeichert wird und in keiner Response-Message vorkommt.
  Alle Testnamen mussten mit dem Praefix `TestLexware` versehen werden,
  weil Lexware dieselben Handler-Namen wie Bexio traegt
  (HandleDisconnect, HandleTriggerSync usw.) und `route_bexio_test.go`
  bzw. `route_email_accounts_test.go`/`route_caldav_test.go` dieselben
  `TestHandleX_*`-Namen bereits belegt hatten — beim ersten Build-Versuch
  als `DuplicateDecl` aufgefallen und korrigiert, bevor irgendetwas
  committet wurde.
- gate: build ok (`-p 2` gateway/..., biz/lexware/..., cmd/gateway/...) |
  vet ok (gateway/..., biz/lexware/...) | lint ok (0 issues,
  gateway/..., biz/lexware/...) | test ok (`internal/gateway` komplett
  gruen inkl. TestOpenAPIRouteDrift [836 Routen gegen 838 dokumentierte
  Pfade, unveraendert — keine neue Route], 0 SKIP / 0 FAIL aus
  `go test -v -count=1 ./internal/gateway/`; `internal/biz/lexware/...`
  unveraendert gruen, DATABASE_URL gegen kmuhub_app) | migration n.a.
  (keine Schemaaenderung) | rls-smoke n.a. (keine neue Tabelle/Policy)
- coverage: internal/gateway 51,9 % (Iteration-31-Endstand, eigene Messung
  `go tool cover -func` vor dieser Unit) -> 52,6 % (eigene Messung nach
  dieser Unit)
- mutations-probe: In `parseLexwareWebhookBody` die Prioritaet in
  `firstNonEmpty(body.EventType, body.EventTypeSnake)` auf
  `firstNonEmpty(body.EventTypeSnake, body.EventType)` gedreht ->
  `TestParseLexwareWebhookBody_CamelCaseWinsOverSnakeCase` wird rot
  (`EventType:snake ResourceID:r`, want camelCase). Zurueckgedreht ->
  gruen, `git diff --stat internal/gateway/route_lexware.go` zeigt 0
  Zeilen.
- verify vorgaenger: sauber. `28caea24` (Iteration 31, CRM-Contact-Files/
  WOPI-Routen) gegen alle acht Fehlerklassen geprueft: `git show --stat
  28caea24` zeigt nur zwei neue reine Gateway-Testdateien
  (`route_crm_contact_files_test.go`, `route_wopi_test.go`) plus
  BACKLOG/JOURNAL — kein gRPC-Bypass, kein Stub/TODO, kein `.proto`
  angefasst, keine neue `RequirePermission`, keine neue Tabelle, keine
  Wire-Shape-Aenderung, kein Guard ersetzt, keine neue Route.
- neue-units: keine
- offen:
  - `fix-email-contacts-csv-export-formula-injection` (seit Iteration 27)
    und `fix-gateway-advisory-product-riskclass-not-validated` (seit
    Iteration 28) stehen weiterhin unbearbeitet am Backlog-Ende — jetzt
    fuenf Iterationen unbeachtet, weil Block B (Coverage-Reihenfolge) sie
    lexikalisch nach hinten sortiert. Fuer den naechsten Lauf vormerken,
    sie vorzuziehen.
  - Kein DB-Gate ausser dem regulaeren Testlauf noetig (keine Migration,
    keine neue Tabelle/Policy in dieser Iteration).

## Iteration 33 — cov-gateway-booking-admin — done — 2026-08-22 05:19
- commit: c88eeb09
- gebaut: `route_booking_admin_test.go` (Admin-CRUD-Teil von `route_booking.go`:
  HandleListBookingPages, HandleCreateBookingPage, HandleGetBookingPage,
  HandleUpdateBookingPage, HandleDeleteBookingPage — 5 Handler + ServiceName,
  32 Tests). Der oeffentliche Teil (HandleGetPublicBookingPage,
  HandleGetAvailability, HandleCreatePublicBooking, ueber
  RegisterPublicRoutes ausserhalb der Registrar-Schleife gemountet) ist Teil
  der stillgelegten Public-Web-Surface (BACKLOG-PARKED.yml) und bleibt
  unangetastet — vorab bestimmt und hier benannt, wie in den Notes verlangt.
  Muster: `route_calendar_events_resources_test.go` (ServiceUnavailable /
  Validation / `*_ReachesRPC`, kein bufconn-Stub fuer CalendarServiceClient
  in diesem Repo).
  Zwei Befunde beim Schreiben, beide dokumentiert statt stillschweigend
  angenommen:
  1. Die im Backlog genannte "Doppelbuchung desselben Zeitfensters" als
     fachlicher Kernfall des Admin-Teils existiert dort nicht. Die gesamte
     Slot-Kollisionspruefung (`ErrBookingSlotUnavailable`, `bookedSet`,
     `overlapsCalendarEvents`) liegt in `BookingService.CreatePublicBooking`
     und `GetAvailability` (`internal/work/calendar/booking_service.go`) —
     beide oeffentlich, beide out of scope. Die Admin-CRUD-Handler
     persistieren nur Seiten-Konfiguration; `errToStatus`
     (`internal/server/calendar_grpc.go:1672`) mapped fuer BookingPage auch
     keinen AlreadyExists-Fall. Naechstliegende ehrliche Entsprechung:
     `TestHandleUpdateBookingPage_ChangedAvailabilityRules_ReachesRPC`
     dokumentiert, dass eine Config-Aenderung ohne lokale Konfliktpruefung
     direkt zur RPC durchgereicht wird.
  2. `createBookingPageRequest.Services` und `updateBookingPageRequest.Services`
     tragen kein `dive` im `validate`-Tag — dieselbe Fehlerklasse wie die
     bereits im Backlog stehende `fix-gateway-advisory-product-riskclass-not-validated`
     (Referenzmuster fuer den Fix: `route_customization.go:473`,
     `validate:"required,min=1,dive"`). Ein Service-Item mit leerem Namen,
     leerem Preis oder `duration_min: 0` erreicht ungeprueft die RPC. Zuerst
     als echte Validierungstests geschrieben (4 Stueck), liefen rot gegen
     den Ist-Code, dann — konsistent mit dem bereits etablierten
     `*_ReachesRPC`-Muster fuer genau diese Situation
     (`route_calendar_events_resources_test.go`, InvertedRange-Fall) — zu
     `*_ReachesRPC`-Dokumentationstests umgebaut statt das Verhalten in
     einem Coverage-only-Commit nebenbei zu aendern. Fix-Unit ans
     Backlog-Ende gehaengt (siehe neue-units).
- gate: build ok (`-p 2` gateway/..., cmd/gateway/...) | vet ok | lint ok
  (0 issues, gateway/...) | test ok (`internal/gateway` komplett gruen,
  2471 PASS / 0 SKIP / 0 FAIL aus `go test -v -count=1 ./internal/gateway/`;
  `TestOpenAPIRouteDrift` unveraendert 836 Routen gegen 838 dokumentierte
  Pfade, keine neue Route) | migration n.a. (keine Schemaaenderung) |
  rls-smoke n.a. (keine neue Tabelle/Policy)
- coverage: internal/gateway 52,6 % (eigene Messung vor dieser Unit,
  Testdatei kurz entfernt und neu gemessen) -> 52,8 % (eigene Messung nach
  dieser Unit)
- mutations-probe: In `HandleGetBookingPage` (`route_booking.go:206`)
  `if !ok { return }` nach `validateUUIDParam` auf `if ok { return }`
  gedreht -> `TestHandleGetBookingPage_InvalidIDUUID` und
  `TestHandleGetBookingPage_ReachesRPC` werden rot. Zurueckgedreht -> gruen,
  `git diff --stat internal/gateway/route_booking.go` zeigt 0 Zeilen.
- verify vorgaenger: sauber. `9b54ee7b` (Iteration 32, Lexware-Route-Coverage)
  gegen alle acht Fehlerklassen geprueft: `git show --stat 9b54ee7b` zeigt
  nur eine neue reine Gateway-Testdatei (`route_lexware_test.go`) plus
  BACKLOG/JOURNAL — kein gRPC-Bypass, kein Stub/TODO, kein `.proto`
  angefasst, keine neue `RequirePermission`, keine neue Tabelle, keine
  Wire-Shape-Aenderung, kein Guard ersetzt, keine neue Route. Der Folgecommit
  `47e76798` ist nur ein docs-SHA-Nachtrag im Journal, unkritisch.
- neue-units: `fix-gateway-booking-page-services-no-dive` (Befund 2 oben,
  ans Backlog-Ende gehaengt, `status: todo`)
- offen:
  - `fix-email-contacts-csv-export-formula-injection` (seit Iteration 27)
    und `fix-gateway-advisory-product-riskclass-not-validated` (seit
    Iteration 28) stehen weiterhin unbearbeitet am Backlog-Ende — jetzt
    sechs Iterationen unbeachtet, weil Block B (Coverage-Reihenfolge) sie
    lexikalisch nach hinten sortiert. Fuer den naechsten Lauf vormerken,
    sie vorzuziehen.
  - Kein DB-Gate ausser dem regulaeren Testlauf noetig (keine Migration,
    keine neue Tabelle/Policy in dieser Iteration).

## Iteration 34 — cov-gateway-registrar-global-search — done — 2026-08-22 05:29
- commit: e7de0eca
- gebaut: `route_search_global_test.go` deckt HandleGlobalSearch (leere Query,
  leere Registry, registrierte Services mit RPC-Fehlschlag, den fest
  verdrahteten Email-Stub, fehlerhafte/gueltige `limit`-Werte) und die
  ServiceName-Methode ab. `route_registrar.go` (reine Interface-Definition,
  keine ausfuehrbaren Zeilen) via Compile-Time-Assertion
  `var _ RouteRegistrar = (*GlobalSearchRoutes)(nil)` erfasst.
  Zwei Punkte im `done_when` explizit als "belegt" statt als Fund
  dokumentiert (Kommentarblock im Testfile): (1) Tenant-Isolation laeuft
  nicht handlerlokal, sondern ueber denselben
  `middleware.TenantOutboundUnaryInterceptor`, den jede andere
  Gateway-Route auch nutzt (`registry.go:112`) — kein routenspezifisches
  Loch. (2) Die Berechtigungspruefung sitzt bewusst einmal an der
  Routengruppe (`RequirePermission("search","read")`), nicht je Teilsuche —
  das ist dasselbe Thin-Handler-Muster wie ueberall sonst im Gateway, kein
  neuer Befund.
- gate: build ok (`-p 2` gateway/..., cmd/gateway/...) | vet ok | lint ok
  (0 issues, gateway/...) | test ok (`internal/gateway` 2479 PASS / 0 SKIP /
  0 FAIL aus `go test -v -count=1 ./internal/gateway/`; `TestOpenAPIRouteDrift`
  gruen, keine neue Route) | test ok (`internal/crm/...` mit `-p 1` gegen
  echtes Verbindungslimit-Flackern bei paralleler Paketausfuehrung erneut
  sauber gruen — Details unter offen:) | migration n.a. (keine
  Schemaaenderung) | rls-smoke n.a. (keine neue Tabelle/Policy)
- coverage: internal/gateway 52,8 % (eigene Messung, Testdatei kurz
  entfernt) -> 53,1 % (eigene Messung nach dieser Unit)
- mutations-probe: In `searchCRM` (`route_search_global.go:117`) den
  Fehlertext von `"service unavailable"` auf `"temporarily unavailable"`
  gedreht -> `TestHandleGlobalSearch_EmptyRegistry_AllModulesReportUnavailable`
  wird rot. Zurueckgedreht -> gruen,
  `git diff --stat internal/gateway/route_search_global.go` zeigt 0 Zeilen.
- verify vorgaenger: sauber. `c88eeb09` (Iteration 33,
  cov-gateway-booking-admin) gegen alle acht Fehlerklassen geprueft:
  `git show --stat c88eeb09` zeigt nur eine neue reine Gateway-Testdatei
  (`route_booking_admin_test.go`) plus BACKLOG/JOURNAL — kein gRPC-Bypass,
  kein Stub/TODO im Produktionscode, kein `.proto` angefasst, keine neue
  `RequirePermission`, keine neue Tabelle, keine Wire-Shape-Aenderung, kein
  Guard ersetzt, keine neue Route. Der genannte Fund (`Services` ohne
  `dive`) wurde korrekt als eigene Fix-Unit ans Backlog-Ende gehaengt statt
  nebenbei gefixt. Folgecommit `21f1f984` ist nur ein docs-SHA-Nachtrag,
  unkritisch.
- neue-units: keine
- offen:
  - Beim ersten Lauf von `go test -count=1 ./internal/crm/...` (Standard-
    Parallelitaet) sind zwei DB-Tests in `internal/crm/contact` mit
    `SQLSTATE 53300` ("remaining connection slots are reserved for roles
    with the SUPERUSER attribute" / "too many clients already")
    fehlgeschlagen — kein Bezug zu dieser Unit (reine Testdatei-Ergaenzung
    in `internal/gateway`, kein `crm`-Code angefasst). Mit `-p 1` (seriell)
    zweimal sauber gruen reproduziert, ebenso `internal/crm/contact/...`
    isoliert. Sieht nach einer niedrigen `max_connections`-Grenze der
    lokalen Docker-Postgres im Verhaeltnis zur Paket-Parallelitaet der
    Go-Testsuite aus, nicht nach einem Produktionsproblem — aber es lohnt
    sich, `max_connections` der lokalen Dev-DB zu pruefen, falls das in
    kuenftigen Laeufen wieder auftritt.
  - Kein DB-Gate ausser dem regulaeren Testlauf noetig (keine Migration,
    keine neue Tabelle/Policy in dieser Iteration).
  - Weiterhin unbearbeitet am Backlog-Ende (jetzt sieben bzw. sechs
    Iterationen ohne Beruehrung, weil die Coverage-Bloecke sie lexikalisch
    nach hinten sortieren):
    `fix-email-contacts-csv-export-formula-injection` (seit Iteration 27),
    `fix-gateway-advisory-product-riskclass-not-validated` (seit
    Iteration 28), `fix-gateway-booking-page-services-no-dive` (seit
    Iteration 33). Fuer den naechsten Lauf vormerken, sie vorzuziehen.

## Iteration 35 — cov-gateway-dashboard — done — 2026-08-22 05:35
- commit: 6db12cf0
- gebaut: `route_dashboard_test.go` deckt alle 10 Funktionen von
  `route_dashboard.go` ab (HandleGetDashboard, HandleSaveDashboard,
  HandleResetToDefaults, HandleGetDefaults, HandleSaveDefaults,
  ServiceName, primaryRole, isValidRole) plus einen Router-Guard-Test
  (`TestDashboardRoutes_Guards`), der RequireAuthenticated auf den
  User-Endpunkten und RequireRole("admin") auf den /defaults/{role}-
  Endpunkten belegt. `mockDashboardRepo` (bereits vorhanden in
  `cached_dashboard_repository_test.go`) additiv um fuenf
  Error-Injection-Felder (`get*Err`/`upsert*Err`/`deleteUserErr`) erweitert,
  Default nil aendert am Verhalten der zwoelf bestehenden Cache-Tests
  nichts. Dokumentiert statt getestet: die beiden `!json.Valid(...)`-Checks
  in HandleSaveDashboard/HandleSaveDefaults sind nach `decodeAndValidate`
  unerreichbar, weil `json.NewDecoder` ein `json.RawMessage`-Feld nur mit
  einem bereits syntaktisch gueltigen JSON-Span befuellt — ein Body, der
  das brechen wuerde, scheitert schon vorher an "invalid request body".
- gate: build ok (`-p 2` gateway/..., cmd/gateway/...) | vet ok | lint ok
  (0 issues, gateway/...) | test ok (`internal/gateway` 2510 PASS / 0 SKIP /
  0 FAIL aus `go test -v -count=1 ./internal/gateway/`; `TestOpenAPIRouteDrift`
  gruen, keine neue Route) | migration n.a. (keine Schemaaenderung) |
  rls-smoke n.a. (Tenant-Isolation der `dashboard_defaults`-Tabelle ist
  bereits durch `rls_dashboard_defaults_test.go` abgedeckt, hier nicht
  erneut angefasst)
- coverage: internal/gateway 53,1 % (eigene Messung, neue Dateien kurz
  ausgelagert) -> 53,4 % (eigene Messung nach dieser Unit)
- mutations-probe: In `HandleGetDefaults` (`route_dashboard.go:153`) den
  Not-Found-Statuscode von `http.StatusNotFound` auf `http.StatusTeapot`
  gedreht -> `TestHandleGetDefaults_NotFound` wird rot (418 statt 404).
  Zurueckgedreht -> gruen, `git diff --stat internal/gateway/route_dashboard.go`
  zeigt 0 Zeilen.
- verify vorgaenger: sauber. `e7de0eca` (Iteration 34,
  cov-gateway-registrar-global-search) gegen alle acht Fehlerklassen
  geprueft: `git show --stat e7de0eca` zeigt nur eine neue reine
  Gateway-Testdatei (`route_search_global_test.go`) plus BACKLOG/JOURNAL —
  kein gRPC-Bypass, kein Stub/TODO, kein `.proto` angefasst, keine neue
  `RequirePermission`, keine neue Tabelle, keine Wire-Shape-Aenderung,
  kein Guard ersetzt, keine neue Route. Folgecommit `ccf0ba7c` ist nur ein
  docs-SHA-Nachtrag im Journal, unkritisch.
- neue-units: keine
- offen:
  - Weiterhin unbearbeitet am Backlog-Ende (jetzt acht/sieben/zwei
    Iterationen ohne Beruehrung, weil die Coverage-Bloecke sie lexikalisch
    nach hinten sortieren):
    `fix-email-contacts-csv-export-formula-injection` (seit Iteration 27),
    `fix-gateway-advisory-product-riskclass-not-validated` (seit
    Iteration 28), `fix-gateway-booking-page-services-no-dive` (seit
    Iteration 33). Fuer den naechsten Lauf vormerken, sie vorzuziehen —
    dieser Hinweis steht jetzt seit drei Iterationen unveraendert im
    Journal und wurde bislang nicht aufgegriffen.
  - Kein DB-Gate ausser dem regulaeren Testlauf noetig (keine Migration,
    keine neue Tabelle/Policy in dieser Iteration; `dashboard_defaults`
    RLS war bereits durch A-Vorlauf/frueheren Fix abgedeckt).

## Iteration 36 — cov-gateway-work-labels-custom-fields — done — 2026-08-22 05:42
- commit: ce77102f
- gebaut: `route_work_labels_test.go` deckt alle 12 Funktionen aus
  `route_work_labels.go` ab (5 Label-Handler, HandleSetTaskLabels,
  5 Custom-Field-Definition-Handler, `labelIDsFromQuery`) nach dem
  etablierten ReachesRPC-Muster (`registryWithService`/`emptyRegistry`),
  plus zwei neue Router-Guard-Tests (`TestWorkLabelRoutes_Guards`,
  `TestWorkCustomFieldRoutes_Guards`), weil `work_labels`/
  `work_custom_fields` bewusst NICHT im additiven Catalogue-Rollout stehen
  (route_work.go:224-227) und deshalb in route_capability_guard_test.go
  fehlen — einfache `RequirePermission`-Guards, kein
  `RequirePermissionAny`. Zusaetzlich `internal/work/label/
  postgres_repository_db_test.go` (neu, gegen die echte DB statt Mock):
  `TestLabelDelete_CascadesTaskLabels` belegt die in
  `label.Repository.Delete`s Doc-Kommentar behauptete Kaskade
  (`work_labels(id) -> task_labels ON DELETE CASCADE`, Migration 000145)
  am echten pg_constraint, `TestLabelRepository_TenantIsolation` denselben
  Pfad fuer RLS.
  BEFUND (kein Fix in dieser Unit, siehe unten): beim Nachgehen der
  done_when-Frage "Loeschverhalten benutzter Labels an pg_constraint
  pruefen" bin ich auf die Custom-Field-Seite gestossen und live gegen die
  lokale DB verifiziert, dass `task_custom_field_values.field_id` noch die
  Original-FK aus Migration 000026 auf `custom_field_definitions` (CRM-only,
  `entity_type`-Check erlaubt kein 'task') traegt statt auf
  `work_custom_field_definitions` (Migration 000146, die tatsaechlich von
  `internal/work/customfield` bedient wird). Jeder Versuch, einen
  Task-Custom-Field-Wert zu setzen, schlaegt mit `foreign_key_violation`
  fehl — bestaetigt per Live-INSERT gegen die lokale DB (Fehlertext in der
  neuen Fix-Unit dokumentiert). `GetCustomFieldValues` hat denselben Fehler
  auf der Lesenseite (joint ebenfalls auf die falsche Tabelle). Als
  `fix-work-task-custom-field-values-wrong-fk` ans Backlog-Ende gehaengt.
- gate: build ok (`-p 2` gateway/..., cmd/gateway/...) | vet ok
  (gateway/..., work/label/...) | lint ok (0 issues, gateway/...,
  work/label/...) | test ok (`internal/gateway` komplett gruen inkl.
  `TestOpenAPIRouteDrift`: 836 Routen gegen 838 dokumentierte Pfade, keine
  neue Route in dieser Unit; `internal/work/...` mit `-p 1` seriell komplett
  gruen — Standard-Parallelitaet reproduziert den seit Iteration 34/35
  bekannten SQLSTATE-53300-Verbindungslimit-Flake auf zwei unveraenderten
  Paketen, kein Bezug zu dieser Unit) | migration n.a. (keine
  Schemaaenderung in dieser Unit — der Fund braucht eine eigene Migration,
  siehe Fix-Unit) | rls-smoke ok (neue DB-Tests in `internal/work/label`
  sind der RLS-Nachweis; als `kmuhub_app` gelaufen)
- coverage: internal/gateway 53,4 % (Stand Iteration 35) -> 54,0 % (eigene
  Messung) · internal/work/label 15,2 % (Baseline aus coverage_start,
  eigene Messung mit ausgelagerter DB-Testdatei bestaetigt exakt denselben
  Wert) -> 49,5 % (eigene Messung nach dieser Unit)
- mutations-probe: In `HandleUpdateLabel` (`route_work_labels.go:99`) den
  Statuscode der "missing label id"-Antwort von `http.StatusBadRequest` auf
  `http.StatusTeapot` gedreht -> `TestHandleUpdateLabel_MissingID` wird rot
  (418 statt 400 erwartet). Zurueckgedreht -> gruen,
  `git diff --stat internal/gateway/route_work_labels.go` zeigt 0 Zeilen.
- verify vorgaenger: sauber. `6db12cf0` (Iteration 35,
  cov-gateway-dashboard) gegen alle acht Fehlerklassen geprueft:
  `git show --stat 6db12cf0` zeigt nur eine neue reine Gateway-Testdatei
  plus eine additive Erweiterung von `mockDashboardRepo` um fuenf
  nil-defaultete Error-Injection-Felder (Diff einzeln gelesen, aendert das
  Verhalten der zwoelf bestehenden Cache-Tests nachweislich nicht) plus
  BACKLOG/JOURNAL — kein gRPC-Bypass, kein Stub/TODO, kein `.proto`
  angefasst, keine neue `RequirePermission`, keine neue Tabelle, keine
  Wire-Shape-Aenderung, kein Guard ersetzt, keine neue Route. Folgecommit
  `7e090ead` ist nur ein docs-SHA-Nachtrag im Journal, unkritisch.
- neue-units: fix-work-task-custom-field-values-wrong-fk (kritischer,
  live-verifizierter Produktionsbug: Task-Custom-Field-Werte lassen sich
  nie speichern, siehe BEFUND oben und BACKLOG.yml)
- offen:
  - Der neue Fix hat `model: opus` (Migration + FK-Umhaengen +
    Karteileichen-Frage in Produktion) und `phase: 4`, steht aber
    lexikalisch nach den drei seit Iteration 27/28/33 liegen gebliebenen
    Fix-Units (`fix-email-contacts-csv-export-formula-injection`,
    `fix-gateway-advisory-product-riskclass-not-validated`,
    `fix-gateway-booking-page-services-no-dive`) — der Treiber zieht
    weiterhin strikt die erste `todo`-Unit mit erfuellten `deps`, das ist
    nicht diese Reihenfolge. Fuer den naechsten Lauf: alle vier Fix-Units
    vorziehen, bevor weitere Coverage-Units laufen — das steht jetzt seit
    vier Iterationen im Journal.
  - Kein DB-Gate fuer eine Migration noetig (diese Unit aendert kein
    Schema). Der Fix selbst braucht vor der Migration eine Pruefung, ob in
    Produktion bereits Zeilen in `task_custom_field_values` liegen
    (Karteileichen mit falschen field_id-Werten), sonst schlaegt die neue
    FK beim Anlegen fehl — siehe notes der neuen Fix-Unit.

## Iteration 37 — fix-idempotency-real-sql-coverage-core — done — 2026-08-22 05:55
- commit: 6507e475
- gebaut: `internal/idempotency/postgres_repository_db_test.go` (neu) testet
  `postgresRepository` (Reserve, Get, Complete) gegen echtes SQL statt gegen
  den In-Memory-Mock aus `repository_test.go`: frische Reservierung,
  Replay nach Complete (inkl. JSONB-Body-Vergleich per `json.Unmarshal`,
  weil Postgres den Body beim Round-Trip umformatiert), Conflict bei
  abweichendem Hash, unbekannter Key bei Get/Complete,
  reserviert-aber-nie-abgeschlossen (Get liefert CompletedAt nil,
  ResponseStatus nil), Cross-Tenant-Complete (ErrKeyMissing ueber die
  WHERE-Klausel, nicht nur RLS) und eine echte Zwei-Goroutinen-Kollision
  auf demselben Schluessel.
  BEFUND (kein Fix in dieser Unit, siehe unten): die Kollisions-Testrecherche
  hat bestaetigt, dass die Postgres-Implementierung von `Reserve` NIEMALS
  `ErrInFlight` zurueckgibt — anders als der eigene Doc-Kommentar
  ("same hash, no response yet → ErrInFlight") und anders als der Mock.
  Zwei gleichzeitige Reservierungen desselben Schluessels bekommen beide
  `(nil, nil)`, solange `completed_at` noch NULL ist, und fuehren die
  Handler-Logik dadurch beide aus. Bereits als `lean:`-Marker in
  `internal/automation/workflow/webhook.go:164` bekannt, dort aber
  ausdruecklich als "out of scope, pre-existing gap in shared infra"
  markiert mit dem Hinweis "Upgrade when internal/idempotency's Reserve is
  hardened for that race" — genau das ist jetzt als
  `fix-idempotency-reserve-inflight-race` ans Backlog-Ende gehaengt, weil
  die Luecke nicht nur den Webhook-Pfad betrifft, sondern die komplette
  gemeinsame Infrastruktur hinter Dialer-Outcomes, Finance-Buchungen und
  jeder anderen Idempotency-Key-Route.
- gate: build ok (`-p 2` idempotency/..., cmd/gateway/..., cmd/automation/...)
  | vet ok | lint ok (0 issues) | test ok (`internal/idempotency` komplett
  gruen, 0 uebersprungene DB-Tests: 4 bestehende Mock-Tests + 4 bestehende
  RLS-Tests + 9 neue Postgres-Tests) | migration n.a. (keine Schemaaenderung)
  | rls-smoke ok (Cross-Tenant-Complete- und bestehende RLS-Tests laufen
  als `kmuhub_app`) | gateway-route-drift n.a. (keine Route angefasst,
  `go test ./internal/gateway/` daher nicht Pflicht fuer diese Unit)
- coverage: internal/idempotency 0,0 % (eigene Messung ohne die neue
  Testdatei — die bestehenden Tests riefen postgresRepository nie auf) ->
  59,0 % (eigene Messung mit der neuen Testdatei)
- mutations-probe: in `Reserve` (`postgres_repository.go:116`) die
  Hash-Vergleichsbedingung mit `if false && rec.RequestHash != hash {`
  deaktiviert -> `TestPostgresReserve_Conflict_DifferentHash` wird rot
  (erwartet ErrConflict, bekommt nil). Zurueckgedreht ->
  `git diff --stat internal/idempotency/postgres_repository.go` zeigt 0
  Zeilen (nur eine harmlose LF/CRLF-Warnung beim Kopieren, keine
  inhaltliche Aenderung).
- verify vorgaenger: sauber. `ce77102f` (Iteration 36,
  cov-gateway-work-labels-custom-fields) gegen alle acht Fehlerklassen
  geprueft: `git show --stat ce77102f` zeigt ausschliesslich zwei neue
  Testdateien (`route_work_labels_test.go`, `internal/work/label/
  postgres_repository_db_test.go`) plus BACKLOG/JOURNAL — kein
  Produktionscode angefasst, kein gRPC-Bypass, kein Stub/TODO, kein
  `.proto` geaendert, keine neue `RequirePermission`, keine neue Tabelle,
  keine Wire-Shape-Aenderung, kein Guard ersetzt, keine neue Route.
  Folgecommit `f00740d7` ist nur der docs-SHA-Nachtrag im Journal.
- neue-units: fix-idempotency-reserve-inflight-race (kritischer,
  live-verifizierter Verhaltens-Bug: `Reserve` haelt seinen eigenen
  ErrInFlight-Vertrag nicht ein, betrifft jede Idempotency-Key-Route)
- offen:
  - Vier Fix-Units stehen jetzt vor der naechsten Coverage-Unit im
    Backlog (`fix-idempotency-real-sql-coverage-cleanup-concurrency` als
    naechste, danach die drei seit Iteration 27/28/33 liegen gebliebenen:
    `fix-email-contacts-csv-export-formula-injection`,
    `fix-gateway-advisory-product-riskclass-not-validated`,
    `fix-gateway-booking-page-services-no-dive`, dann
    `fix-work-task-custom-field-values-wrong-fk` und die neue
    `fix-idempotency-reserve-inflight-race`) — der Treiber zieht ohnehin
    strikt die erste `todo`-Unit mit erfuellten `deps`, das ist bereits
    die richtige Reihenfolge, keine manuelle Umsortierung noetig.
  - Kein DB-Gate fuer eine Migration noetig (diese Unit aendert kein
    Schema). Der neue Fix (`fix-idempotency-reserve-inflight-race`) ist
    reine Code-Aenderung ueber `xmax = 0`, ebenfalls ohne Migration.

## Iteration 38 — fix-idempotency-real-sql-coverage-cleanup-concurrency — done — 2026-08-22 06:02
- commit: 254120eb
- gebaut: drei neue Tests in `internal/idempotency/postgres_repository_db_test.go`
  gegen echtes SQL statt Mock: `TestPostgresCleanup_DeletesOnlyExpired` (eine
  frische und eine zwangsweise abgelaufene Reservierung, Cleanup entfernt nur
  die abgelaufene), `TestPostgresCleanupWithLock_SkipsWhenLockHeld` (eine
  eigens gehaltene `pg_try_advisory_lock`-Session auf einer separaten
  Connection zwingt `CleanupWithLock` zu sofortigem `(0, nil)` ohne zu
  blockieren — belegt ueber Postgres' Non-Blocking-Semantik der
  try-Variante, nicht ueber Timing) und
  `TestPostgresCleanupWithLock_AcquiresAndReleases` (belegt zusaetzlich,
  dass der Advisory Lock nach Rueckkehr wirklich frei ist, geprueft von
  einer dritten, unabhaengig erworbenen Connection — nicht nur dass
  irgendetwas geloescht wurde).
  RECHERCHE-BEFUND UNTERWEGS (kein Bug, aber wert festzuhalten): die zweite
  Version von `TestPostgresCleanupWithLock_AcquiresAndReleases` schlug
  zunaechst mit "expected the expired reservation to be deleted, got 0" fehl,
  weil ich `CleanupWithLock` mit blankem `context.Background()` aufgerufen
  hatte. Produktionscode (`internal/middleware/idempotency.go:228`,
  `cleanupCtx`) wrappt den Worker-Context IMMER mit `sysctx.With(...)`, bevor
  er `CleanupWithLock` ruft — ohne das laeuft die DELETE-Query durch dieselbe
  RLS-Policy wie jeder andere Write und matcht ohne System- oder
  Tenant-Kontext keine Zeile, meldet aber `(0, nil)`, keinen Fehler. Das ist
  also ein impliziter Caller-Vertrag von `CleanupWithLock`, den nur der
  Middleware-Kommentar dokumentiert, nicht die Repository-Methode selbst.
  Testfehler war meiner, nicht der Produktionscode — nach Korrektur auf
  `testutil.WithSystemCtx` gruen. Kein Fix-Unit-Bedarf, aber der
  Test-Kommentar an der Stelle haelt die Falle fest, damit sie nicht ein
  zweites Mal jemanden trifft.
  Meine urspruengliche Hypothese aus den Unit-Notes — dass `CleanupWithLock`
  Lock und Unlock ueber unterschiedliche, aus dem Pool gezogene Connections
  laufen lassen und den Advisory Lock damit leaken koennte — wurde durch
  `TestPostgresCleanupWithLock_AcquiresAndReleases` GEPRUEFT UND WIDERLEGT:
  der Lock ist nach Rueckkehr nachweislich frei. Kein Fund, aber eine
  begruendete Nicht-Findung, die die Notes der Unit direkt beantwortet.
- gate: build ok (`-p 2` idempotency/..., middleware/..., cmd/automation/...,
  cmd/gateway/...) | vet ok | lint ok (0 issues) | test ok (24/24 PASS,
  0 SKIP, 0 FAIL — DATABASE_URL gesetzt, Rolle kmuhub_app) | migration n.a.
  (keine Schemaaenderung) | rls-smoke ok (Tenant- und System-Context-Pfade
  beider neuen Cleanup-Tests laufen bewusst unterschiedlich durch RLS,
  bestehende RLS-Tests weiterhin gruen) | gateway-route-drift n.a. (keine
  Route angefasst, `go test ./internal/gateway/` daher nicht Pflicht) |
  `-race` NICHT gelaufen, siehe offen:
- coverage: internal/idempotency 59,0 % (Iteration-37-Stand nach deren
  Postgres-Testdatei) -> 87,2 % (eigene Messung nach dieser Unit)
- mutations-probe: in `Cleanup` (`postgres_repository.go:180`) die
  DELETE-Bedingung von `expires_at < NOW()` auf `expires_at > NOW()`
  invertiert -> `TestPostgresCleanup_DeletesOnlyExpired` UND
  `TestPostgresCleanupWithLock_AcquiresAndReleases` werden beide rot
  (erstere: die frische Reservierung verschwindet statt zu ueberleben;
  zweite: die abgelaufene bleibt liegen statt geloescht zu werden).
  Zurueckgedreht -> `git diff --stat internal/idempotency/postgres_repository.go`
  zeigt 0 inhaltliche Zeilen (nur die uebliche LF/CRLF-Warnung).
- verify vorgaenger: sauber. `6507e475` (Iteration 37,
  fix-idempotency-real-sql-coverage-core) gegen alle acht Fehlerklassen
  geprueft: `git show --stat 6507e475` zeigt ausschliesslich eine neue
  Testdatei (`internal/idempotency/postgres_repository_db_test.go`) plus
  BACKLOG/JOURNAL — kein Produktionscode angefasst, kein gRPC-Bypass, kein
  Stub/TODO, kein `.proto` geaendert, keine neue `RequirePermission`, keine
  neue Tabelle, keine Wire-Shape-Aenderung, kein Guard ersetzt, keine neue
  Route.
- neue-units: keine
- offen:
  - `-race` konnte auf dieser Maschine nicht laufen: `go test -race`
    braucht cgo, und `gcc` ist hier nicht installiert (`cgo: C compiler
    "gcc" not found`). Alle drei neuen Tests liefen ohne `-race`, aber
    grundsaetzlich nebenlaeufig (`t.Parallel()`, echte zweite/dritte
    Connections). CI hat vermutlich einen C-Compiler — dort einmal mit
    `-race` gegenpruefen, wie die Unit-Notes es fordern.
  - `CleanupWithLock` braucht zwingend einen System- oder Tenant-Context
    (RLS), sonst liefert es `(0, nil)` ohne jeden Hinweis auf den Grund.
    Das ist heute nur als Kommentar an `cleanupCtx` in
    `internal/middleware/idempotency.go` dokumentiert, nicht an der
    Repository-Methode selbst — waere ein guter Kandidat fuer einen
    Doc-Kommentar an `Repository.CleanupWithLock`/`Cleanup` in einer
    kuenftigen Unit, aendert aber kein Verhalten und war hier nicht der
    Auftrag.

## Iteration 39 — verify-datev-extf-encoding-requirement — blocked — 2026-08-22 06:09
- commit: 59ba84fd
- gebaut: nichts am Produktionscode. Recherche zur DATEV-EXTF-Zeichenkodierung (Format 700,
  Kategorie 21, Buchungsstapel) via WebSearch/WebFetch gegen developer.datev.de und Drittquellen.
  Ergebnis als ausfuehrlicher Kommentar an `TestExport_GoldenBytesWithUmlauts` in
  `internal/biz/datev/exporter_test.go` festgehalten (siehe Kommentarblock dort fuer Details).
- gate: build ok | vet ok | lint ok (0 issues) | test ok (`go test -count=1 ./internal/biz/datev/...`
  gruen, DATABASE_URL gesetzt) | migration n.a. | rls-smoke n.a. (keine Tabelle/Policy angefasst)
- coverage: internal/biz/datev unveraendert (kein Verhalten, keine neue Testfunktion — nur ein
  Kommentar erweitert) | n.a. (Verify-Unit ohne Coverage-Ziel)
- mutations-probe: n.a. (Verify-Unit, kein neuer/geaenderter Testfall mit Assertion-Logik)
- verify vorgaenger: sauber. `254120eb` (Iteration 38, fix-idempotency-real-sql-coverage-cleanup-
  concurrency) gegen alle acht Fehlerklassen geprueft: `git show --stat 254120eb` zeigt
  ausschliesslich eine neue Testdatei (`internal/idempotency/postgres_repository_db_test.go`) plus
  BACKLOG/JOURNAL — kein Produktionscode, kein gRPC-Bypass, kein Stub/TODO, kein `.proto`
  geaendert, keine neue `RequirePermission`, keine neue Tabelle, keine Wire-Shape-Aenderung, kein
  Guard ersetzt, keine neue Route.
- neue-units: keine
- offen:
  - Die eigentliche Frage (cp1252 vs. UTF-8 fuer den DATEV-EXTF-Buchungsstapel-Import) bleibt
    unbeantwortet. Zusammenfassung der Recherche: developer.datev.de fuehrt fuer Format 700/
    Kategorie 21 (Buchungsstapel- UND Header-Feldliste, direkt abgerufen, nicht nur Suchsnippet)
    KEINE Zeichensatzvorgabe. Eine dedizierte "Zeichensatz"-Unterseite existiert auf dem Portal,
    liefert aber ohne Login 404 — ihr indexierter Inhalt (nur per Suchmaschinen-Synthese, nicht
    selbst gelesen) legt nahe: UTF-8 mit BOM wird akzeptiert, aber nur bei manuellem Import bzw.
    ueber die Online-API "accounting:extf-files" — beides deckt sich NICHT zwangslaeufig mit dem
    Importweg, den ein Kunde tatsaechlich nutzt. Das unabhaengige Ruby-Gem ledermann/datev
    (kein DATEV-Bezug) kodiert seinen EXTF-Export hart auf CP1252 — ein reales Gegenindiz.
  - Naechster Schritt liegt bei Luke: entweder eine kostenpflichtige/registrierte
    developer.datev.de-Session fuer die Primaerquelle, oder ein empirischer Test mit einer echten
    DATEV-Instanz/einem Steuerberater. Bis dahin bleibt der Exporter unveraendert (UTF-8 mit BOM).

## Iteration 40 — scan-dsar-missing-contact-user-tables — done — 2026-08-22 06:15
- commit: 8170ee62
- gebaut: nichts am Produktionscode (Scan-Unit, aendert kein Verhalten). Vollstaendige Liste der
  Kandidatentabellen gezogen: `pg_constraint` (FK auf `contacts(id)` bzw. `users(id)`, 154 Zeilen)
  gegen die lokale DB plus eine zweite Abfrage auf Spaltennamen
  (email/phone/first_name/last_name/ip_address/birth_date/address/...), beide gegen
  `dsar_search.go` (aktuell 13 Contact-Module: CRM Kontakte, customFields, tags, documents,
  consent, dialer, finance, meetings, helpdesk, helpdeskMessages, contracts, email, deals,
  formSubmissions, activities — und 11 User-Module: Benutzerkonto, chatMessages,
  chatMemberships, tasks, taskComments, timeEntries, calendarEvents, calendarPrefs,
  notifications, notificationPrefs, quietHours, mutes) abgeglichen.
  Entscheidungen je Gruppe:
  - 14 contacts-FK-Tabellen ueber bestehende Module abgedeckt (contact_tags, contacts
    Selbstbezug [merged_into_id/referred_by_contact_id, strukturell, keine eigenen
    Personendaten], contact_custom_field_values, deals, activities, meetings,
    email_contact_links, finance_invoices, consent_records, dialer_campaign_contacts, tickets,
    contract_parties) — Nicht-Fund.
  - `gdpr_deletion_requests.contact_id`/`gdpr_export_requests.*`/`gdpr_erasure_log.executed_by`:
    Prozess-Metadatensaetze der GDPR-Mechanik selbst — Aufnahme waere zirkulaer (die Auskunft
    wuerde sich selbst als Datenpunkt auflisten). Legitimer Nicht-Fund, konsistent mit
    bestehender Behandlung von gdpr_export_requests.
  - `advisory_protocols.contact_id` (RESTRICT-FK, Grund fuer den 409-Pfad in `IsInUse`):
    KEIN Modul vorhanden, obwohl die Tabelle hochsensible Daten traegt (Geburtsdatum,
    Familienstand, Steuerstatus, Einkommen, Vermoegen, Risikoklasse, Versicherungsstatus).
    FUND -> feat-dsar-search-contact-advisory-protocol-module.
  - ueber 120 users-FK-Zeilen sind reine Attributions-Spalten (created_by/updated_by/
    approved_by/assigned_to/uploaded_by/author_id/actor_id/...) auf Geschaeftsdatensaetzen, die
    inhaltlich nicht ueber die referenzierende Person handeln (z. B. deals.created_by ist keine
    Aussage ueber den Vertriebler als Person) — legitimer Nicht-Fund, ausser die Flaeche ist
    bereits als eigenes Modul abgedeckt (Chat, Tasks, Calendar, Notifications ueber A7-A9).
  - `hr_employee_profiles`/`hr_leave_requests`/`hr_leave_balances`/`hr_work_time_entries`/
    `hr_employee_documents`/`hr_profile_change_requests` (alle an `users` gebunden ueber
    user_id/employee_id/manager_user_id): KEIN Modul. Beschaeftigtendaten (Adresse,
    Notfallkontakt, Stundenlohn, Urlaub, Arbeitszeit, is_minor) sind eine der sensibelsten
    DSGVO-Kategorien. FUND -> feat-dsar-search-user-hr-employee-module.
  - `user_sessions`/`password_history`/`recovery_codes`/`two_factor_policy` (an `users`
    gebunden, `user_sessions.ip_address` zusaetzlich per Spaltennamen-Scan gefunden):
    KEIN Modul. Login-/Sitzungsverlauf ist klassisch auskunftspflichtig. FUND ->
    feat-dsar-search-user-account-security-history-module.
  - `driver_licenses` (CASCADE an users) und `vehicle_bookings` (created_by/user_id an users):
    KEIN Modul im Fuhrpark-Bereich. FUND -> feat-dsar-search-user-fuhrpark-driver-module.
  - `invitations` (first_name/last_name/email als eigene Spalten, nicht per FK — ueber
    Spaltennamen-Scan gefunden, nicht ueber pg_constraint): KEIN Modul, auch fuer laengst
    angenommene Einladungen nicht. FUND -> feat-dsar-search-invitation-history-module
    (inkl. dokumentiertem Nebenbefund: nie angenommene Einladungen sind ueberhaupt nicht
    matchbar, siehe unten).
  - `guest_sessions` (email/ip_address als eigene Spalten, kein FK auf contacts/users):
    bewusst NICHT als eigene Unit angelegt. Begruendung: Feature-Reichweite unklar (oeffentlicher
    Chat-Gast ohne Konto), Matching wuerde eine dritte Subjekt-Matching-Quelle neben
    Kontakten/Benutzern noetig machen — groesserer Architektureingriff, kein einfacher
    Modul-Anhang. Als offener Befund unten benannt statt als Unit erzwungen.
  - `companies`/`company_settings`/`inventory_locations`/`suppliers`-Adress-/Kontaktspalten aus
    dem Spaltennamen-Scan: Geschaeftsadressen/-kontakte (B2B), keine natuerlichen Personen als
    Datensubjekt in diesem Kontext — Nicht-Fund.
- gate: n.a. (Scan-Unit aendert kein Verhalten, kein Produktionscode, keine Migration, kein Test
  angefasst — done_when der Unit verlangt kein go test)
- coverage: n.a. (Scan-Unit ohne Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein neuer/geaenderter Testfall)
- verify vorgaenger: sauber. `e2a1d75c` (Iteration 39, journal-sha-Nachtrag) und `59ba84fd`
  (Iteration 39, verify-datev-extf-encoding-requirement) gegen alle acht Fehlerklassen geprueft:
  `git show --stat` fuer beide zeigt ausschliesslich BACKLOG/JOURNAL plus einen erweiterten
  Kommentarblock in `exporter_test.go` — kein Produktionscode, kein gRPC-Bypass, kein Stub/TODO,
  kein `.proto` geaendert, keine neue `RequirePermission`, keine neue Tabelle, keine
  Wire-Shape-Aenderung, kein Guard ersetzt, keine neue Route.
- neue-units: feat-dsar-search-contact-advisory-protocol-module,
  feat-dsar-search-user-hr-employee-module,
  feat-dsar-search-user-account-security-history-module,
  feat-dsar-search-user-fuhrpark-driver-module, feat-dsar-search-invitation-history-module
- offen:
  - `guest_sessions` (oeffentlicher Chat-Gast mit optionaler E-Mail/IP) ist eine Personendaten
    tragende Tabelle ohne jede Anbindung an `matchContacts`/`matchUsers` — bewusst NICHT als Unit
    angelegt (siehe Begruendung oben), weil eine Loesung eine dritte Subjekt-Matching-Quelle
    braeuchte. Architekturfrage fuer Luke: lohnt sich das fuer eine Funktion, deren
    Nutzungsumfang hier nicht bekannt ist?
  - Innerhalb von feat-dsar-search-invitation-history-module ist bereits vermerkt: nie
    angenommene Einladungen (`accepted_at IS NULL`) bleiben unloesbar mit der aktuellen
    Zwei-Quellen-Matching-Architektur (Kontakte, Benutzer) — dieselbe Grenze wie bei
    guest_sessions, aus demselben Grund nicht in dieser Nacht geloest.

## Iteration 41 — scan-erasure-handlers-missing-personal-data-tables — done — 2026-08-22 06:21
- commit: dded309a
- gebaut: nichts am Produktionscode (Scan-Unit, aendert kein Verhalten). Alle 139 FKs auf
  `users(id)` aus `pg_constraint` gezogen (lokale DB, 2026-08-22) und gegen die sieben
  registrierten Erasure-Handler (`cmd/auth/main.go:109-115`: auth, crm, chat, work, calendar,
  notifications, audit) abgeglichen, je Tabellenmenge (PreviewErasure vs. ExecuteErasure) und
  je Domaenen-Vollstaendigkeit.
  Zentraler Befund vorweg: `AuthErasureHandler.ExecuteErasure` fuehrt NIE ein
  `DELETE FROM users` aus (`grep -rn "DELETE FROM users" backend/internal/` liefert null
  Treffer) — die `users`-Zeile wird nur anonymisiert (UPDATE), nie geloescht. Damit feuert JEDE
  `ON DELETE CASCADE`-FK auf `users(id)` NIE als Aufraeummechanismus; Tabellen mit
  `delete_rule = 'c'`, die kein Handler bedient, behalten ihre Personendaten dauerhaft, nicht
  nur "bis zum naechsten Cascade".
  Sieben Funde, alle als Unit angelegt:
  1. `fix-work-erasure-time-entries-retention-conflict` — WorkErasureHandler loescht
     `time_entries` hart mit Kommentar "no business retention need", was der eigenen Feststellung
     aus Iteration 40 widerspricht (Zeiteintraege sind arbeitsrechtlich relevant). Ueberreichweite,
     nicht Luecke.
  2. `fix-auth-erasure-missing-security-tables` — `password_reset_tokens`,
     `app_specific_passwords` (beide CASCADE, Auth-Domaene) von AuthErasureHandler nicht erfasst.
  3. `fix-calendar-erasure-incomplete-and-doc-mismatch` — Doc-Kommentar behauptet "Deletes
     personal calendars and owned events", Code tut keins von beidem (`calendars` unberuehrt,
     `calendar_events` nur gezaehlt); zusaetzlich `calendar_members`,
     `caldav_push_subscriptions` fehlen.
  4. `fix-crm-erasure-contacts-companies-preview-execute-mismatch` — Preview zaehlt
     contacts/companies als "betroffen", Execute aendert an beiden Tabellen nichts (nur erneut
     gezaehlt) — Preview/Execute-Bruch, genau der in den Unit-Notes gesuchte gefaehrlichste Fund.
  5. `fix-work-erasure-missing-project-membership` — `project_members` (CASCADE, Pendant zu
     `channel_memberships`, das der Chat-Handler bereits loescht) von WorkErasureHandler nicht
     erfasst.
  6. `fix-chat-erasure-missing-bookmarks-mentions` — `message_bookmarks`, `message_mentions`
     (beide CASCADE) von ChatErasureHandler nicht erfasst.
  7. `feat-erasure-handler-user-settings-preferences` — `user_settings`,
     `user_dashboard_layouts`, `user_project_preferences`, `saved_filters` (alle CASCADE) liegen
     in keiner der sieben ModuleName-Domaenen; es fehlt ein achter Handler.
  Kein Fund bei: NotificationErasureHandler (alle vier Tabellen vollstaendig, Preview=Execute),
  AuditErasureHandler (korrekter No-op nach DSGVO Art. 17(3)(e), keine Ueberreichweite),
  ChatErasureHandler-Kernpfad (messages/channel_memberships selbst deckungsgleich).
- gate: n.a. (Scan-Unit, kein Produktionscode/Migration/Test angefasst — done_when verlangt
  kein go test)
- coverage: n.a. (Scan-Unit ohne Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein neuer/geaenderter Testfall)
- verify vorgaenger: sauber. `67007bf0` (Iteration 40, journal-sha-Nachtrag) geprueft: `git show
  --stat` zeigt ausschliesslich JOURNAL.md — kein Produktionscode, keine der acht
  Fehlerklassen einschlaegig.
- neue-units: fix-work-erasure-time-entries-retention-conflict,
  fix-auth-erasure-missing-security-tables, fix-calendar-erasure-incomplete-and-doc-mismatch,
  fix-crm-erasure-contacts-companies-preview-execute-mismatch,
  fix-work-erasure-missing-project-membership, fix-chat-erasure-missing-bookmarks-mentions,
  feat-erasure-handler-user-settings-preferences
- offen:
  - `BACKLOG.yml` ist an sich schon seit einer frueheren Iteration kein strikt parsebares YAML
    mehr (`python -c "import yaml; yaml.safe_load(...)"` bricht bei Zeile ~2175 in einer
    laengst bestehenden `fix-idempotency-reserve-inflight-race`-Unit mit einem Backtick-Token
    ab, verifiziert gegen HEAD vor dieser Iteration). Der Treiber parst laut Kopf der Datei
    ohnehin per Zeilen-Regex, nicht per YAML-Parser — insofern folgenlos, aber falls je ein
    strikter YAML-Parser eingesetzt wird, muss das behoben werden.
  - Reihenfolge der sieben neuen Units ist bewusst nicht priorisiert; `fix-work-erasure-time-
    entries-retention-conflict` (Punkt 1) hat das groesste Compliance-Risiko (Ueberreichweite,
    nicht nur Luecke) und sollte als naechstes gezogen werden, wenn die Blockreihenfolge das
    zulaesst.

## Iteration 42 — scan-restrict-fk-without-ui-path — done — 2026-08-22 06:35
- commit: eb4bf4b6
- gebaut: nichts am Produktionscode (Scan-Unit, aendert kein Verhalten). Alle RESTRICT- und
  NO-ACTION-FKs (`confdeltype` 'r'/'a') aus `pg_constraint` gezogen (lokale DB, 2026-08-22,
  ~155 Treffer) und in vier Gruppen bewertet.
  Gruppe 1 (unreachable, ~40 Treffer): alle `fk_*_tenant`-FKs auf `tenants(id)` (NO ACTION).
  `grep -rn "DELETE FROM tenants" backend/internal/` findet ausser Test-Cleanup-Code
  (`rls_tenants_policy_test.go`) keinen Treffer — es gibt keine Tenant-Loeschfunktion. Diese FKs
  koennen nie feuern.
  Gruppe 2 (unreachable, ~73 Treffer): alle FKs auf `users(id)` (70 NO ACTION + 3 RESTRICT:
  `dialer_agent_status_log.user_id`, `dialer_call_sessions.agent_id`,
  `dialer_campaigns.created_by`). Deckt sich mit dem Zentralbefund aus Iteration 41:
  `AuthErasureHandler` fuehrt nie `DELETE FROM users` aus, nur `grep -rn "DELETE FROM users"` in
  `internal/` ausser Test-Fixtures liefert null Treffer. Die drei RESTRICT-FKs kodieren Absicht
  fuer ein Feature (User-Hard-Delete), das es nicht gibt — kein aktueller Bug, aber vermerkt fuer
  den Fall, dass ein Hard-Delete-Pfad je gebaut wird.
  Gruppe 3 (bereits bekannt/abgedeckt, 4 Treffer): `advisory_protocols.contact_id` und
  `dialer_campaign_contacts.contact_id` (beide RESTRICT auf `contacts`) sind ueber `IsInUse` +
  409 + Anonymisierungsangebot geloest (Lauf-10-Vorbereitung). `contacts.merged_into_id` (NO
  ACTION, Selbstbezug) hat bereits eine offene Fix-Unit aus Iteration 2
  (`fix-contact-delete-merged-into-no-action-unchecked`, status: todo) — nicht dupliziert.
  Gruppe 4 (echte Funde, 12 verbleibende Kandidaten aus Business-Tabellen per Explore-Agent
  gegen Gateway-Routen und Services geprueft): 7 der 12 sind unerreichbar, weil kein
  Delete-Pfad existiert (`call_sessions`, `finance_invoices`, `hr_document_categories`,
  `hr_leave_types`, `hr_work_time_entries`, `dialer_campaign_contacts` selbst, `meetings` —
  Repository hat `Delete`, aber keine Gateway-Route ruft es auf). `pipeline_stages` ist sauber
  behandelt (`HasDeals` -> `ErrStageHasDeals` -> 409 ueber `mapCRMError`, Vorlage fuer alle
  Fixes). Vier sind echte Sackgassen: `companies.merged_into_id` (Merge-Ziel ohne eigene
  Kontakte loeschen -> 500 statt 409, `crm_grpc.go` default-Zweig), `channels` (Channel mit
  archivierten `call_sessions` loeschen -> 500, `chat_grpc.go` default-Zweig, dort laut
  Code-Kommentar sogar bewusst so belassen), `document_folders` (nicht-leeren Ordner loeschen
  -> 500, `document_grpc.go` default-Zweig — der haeufigste Alltagsfall der vier), und
  `integration_channel_mappings` (Mapping mit Delivery-Log loeschen -> 500, geringes Risiko,
  Admin-only-Pfad).
  Zusaetzlich der vom Backlog-Kopf ausdruecklich verlangte Punkt `gdpr_deletion_requests.contact_id`
  (CASCADE, kein RESTRICT/NO-ACTION, deshalb ausserhalb der eigentlichen FK-Liste dieses Scans,
  aber explizit im `done_when`): Erreichbarkeit geklaert — `consent.Service.ProcessDeletion`
  loescht den Kontakt nie (nur `AnonymizeContact`), die Anfrage-Zeile ueberlebt also die eigene
  "completed"-Markierung. Sie stirbt aber, sobald derselbe Kontakt spaeter ueber einen zweiten,
  unabhaengigen Pfad hart geloescht wird — am wahrscheinlichsten automatisiert ueber
  `ContactRetentionHandler.Apply(action=delete)` (Iteration 11), die keine Pruefung auf
  bestehende `gdpr_deletion_requests`-Zeilen kennt. Der CASCADE loescht dann den Nachweis, dass
  eine Art.-17-Anfrage bearbeitet wurde, zusammen mit dem Datensatz, den sie betraf — ein
  Compliance-Nachweis-Verlust, real erreichbar ueber den bereits gebauten Retention-Pfad.
- gate: n.a. (Scan-Unit, kein Produktionscode/Migration/Test angefasst — done_when verlangt kein
  go test)
- coverage: n.a. (Scan-Unit ohne Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein neuer/geaenderter Testfall)
- verify vorgaenger: sauber. `dded309a` (Iteration 41, scan-erasure-handlers) geprueft: `git show
  --stat` zeigt ausschliesslich `BACKLOG.yml` und `JOURNAL.md` — kein Produktionscode, keine der
  acht Fehlerklassen einschlaegig.
- neue-units: fix-company-delete-merged-into-fk-crash, fix-channel-delete-call-sessions-fk-crash,
  fix-document-folder-delete-nonempty-fk-crash, fix-integration-mapping-delete-fk-crash,
  fix-gdpr-deletion-request-audit-trail-cascade-loss
- offen:
  - Die Delete-Pfad-Reichweitenpruefung fuer die zwoelf Business-Tabellen-Kandidaten lief ueber
    einen Explore-Subagenten (Grep + Lesen ueber Repository/Service/Gateway-Schichten), nicht
    selbst nachvollzogen Zeile fuer Zeile — die Datei:Zeile-Angaben in den vier neuen Units
    stammen aus diesem Agentenbericht und sollten beim Bauen kurz gegengeprueft werden.
  - `meetings.Repository.Delete` existiert, wird aber von keiner Gateway-Route aufgerufen (nur
    `DELETE /meetings/{id}/cohosts/{userId}` fuer Co-Hosts existiert) — toter Code oder fehlende
    Route, keine eigene Unit angelegt, da unklar ist, ob ein Meeting-Hard-Delete ueberhaupt
    gewollt ist (Produktentscheidung, kein Bug).
  - Die drei RESTRICT-FKs auf `users(id)` (Gruppe 2) sind reine Beobachtung fuer den Fall eines
    kuenftigen User-Hard-Deletes, keine Unit angelegt — heute unerreichbar und damit keine
    Sackgasse.

## Iteration 43 — scan-personal-data-tables-without-retention-mapping — done — 2026-08-22 06:44
- commit: a83334af
- gebaut: nichts am Produktionscode (Scan-Unit, aendert kein Verhalten). Die Retention-Registry
  aus A10 (`cmd/auth/main.go:122-128`) gegen die Personendaten-Domaenen abgeglichen, die dieser
  Lauf selbst bereits als personenbezogen identifiziert hat (DSAR-Module aus den Iterationen 2-9
  plus die sieben Erasure-Handler-Domaenen aus Iteration 41) — die Ausweitung auf alle 24
  Services ist laut Unit-Notes ausdruecklich nicht Teil dieser Unit.
  Registriert sind fuenf `resource_type`-Eintraege: contacts, dialer_call_sessions, messages
  (Chat), tickets (Helpdesk), form_submissions. Deren Cascade-Reichweite ist per Code-Kommentar
  in den vier Handler-Dateien bereits dokumentiert und per `pg_constraint` stichprobenartig
  nachvollzogen: `message_bookmarks` (000262) und `message_mentions` (000017) kaskadieren beide
  CASCADE auf `messages.id`, `ticket_messages` kaskadiert CASCADE auf `tickets.id`
  (Migrationskopf-Kommentar in `retention_helpdesk_formulare.go:9`) — beide sind bei
  `action=delete` also kostenlos mitabgedeckt, kein eigener Handler noetig.
  Vier Kategorien:
  1. Registriert + Cascade-abgedeckt: contacts (+ contact_custom_field_values, contact_tags,
     email_contact_links, gdpr_deletion_requests bei delete), dialer_call_sessions (Recordings
     bewusst separat, eigener Expiry-Mechanismus laut Code-Kommentar), messages (+ bookmarks,
     mentions), tickets (+ ticket_messages), form_submissions.
  2. Bereits als eigene Unit erfasst, hier nicht dupliziert: die acht SET-NULL-Tabellen an
     `contacts` sind Gegenstand von `scan-contact-set-null-residual-personal-data` (todo).
  3. Legitime Nicht-Zuordnung (gegenlaeufige Aufbewahrungspflicht, kein Befund): `time_entries`
     und `hr_work_time_entries` (arbeitsrechtliche Aufbewahrung, deckt sich mit der bereits
     offenen `fix-work-erasure-time-entries-retention-conflict` aus Iteration 41),
     `finance_invoices` (GoBD, zusaetzlich WORM-artig ueber die SET-NULL-Kaskade an contacts
     bereits von der Loeschung des Kontakts entkoppelt), `gobd_documents`/`gobd_document_events`
     (WORM seit Iteration 1, duerfen innerhalb der Aufbewahrungsfrist gar nicht angefasst
     werden), `advisory_protocols` (RESTRICT-geschuetzt, vermutlich WpHG-aehnliche
     Dokumentationspflicht — keine Migration gefunden, die das explizit belegt, daher als
     Verdacht und nicht als Fakt notiert).
  4. Echte Luecken (Personendaten, kein Gegenbeleg fuer eine Aufbewahrungspflicht gefunden, keine
     Registry-Zuordnung moeglich): Work (`tasks`, `task_comments`), Kalender
     (`calendar_events`, `event_attendees`), Notifications (`notifications`). Alle drei sind
     durch die bereits gebauten DSAR-Module (A8/A9, Iterationen 8-9) als personenbezogen belegt
     — die Auskunft kennt diese Daten, eine Loeschfrist gibt es fuer keine von ihnen. Drei neue
     Units angelegt, jeweils nach dem Muster A11-A13 (ResourceType + Registry-Eintrag +
     Idempotenz-Test), mit den jeweiligen Sonderfaellen im Voraus benannt: task_comments-Cascade
     vorab per pg_constraint pruefen statt annehmen, wiederkehrende Kalendertermine (`rrule`)
     duerfen nicht blind nach Alter geloescht werden, ungelesene Notifications brauchen eine
     Ausnahme von der Alters-Loeschung.
  Nicht als Luecke gewertet, mit Begruendung: `channel_memberships`/`project_members` sind
  Mitgliedschafts-Zustand, kein akkumulierender Personendaten-Bestand, der eine Alters-Loeschung
  braucht (anders als die Iteration-41-Erasure-Luecke, die einen ANDEREN Mechanismus betrifft —
  eine Person, die ihre Loeschung beantragt, nicht eine Alters-basierte Routine). `users`/Auth
  (inaktive Konten loeschen) ist eine Produktentscheidung mit hoeherer Tragweite als die anderen
  drei Faelle, keine Unit angelegt, nur als Beobachtung notiert. `user_settings`,
  `user_dashboard_layouts`, `user_project_preferences`, `saved_filters` (Iteration 41, Fund 7)
  sind Praeferenz-Daten ohne eigenstaendigen Alterungsbedarf — sie verlieren ihren Bezug, sobald
  das Konto selbst geloescht/anonymisiert wird, kein separater Handler noetig.
- gate: n.a. (Scan-Unit, kein Produktionscode/Migration/Test angefasst — done_when verlangt kein
  go test)
- coverage: n.a. (Scan-Unit ohne Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein neuer/geaenderter Testfall)
- verify vorgaenger: sauber. `eb4bf4b6` (Iteration 42, scan-restrict-fk-without-ui-path)
  geprueft: `git show --stat` zeigt ausschliesslich `BACKLOG.yml` und `JOURNAL.md` — kein
  Produktionscode, keine der acht Fehlerklassen einschlaegig.
- neue-units: feat-retention-worker-handler-work-tasks,
  feat-retention-worker-handler-calendar-events, feat-retention-worker-handler-notifications
- offen:
  - Der Verdacht zu `advisory_protocols` (WpHG-aehnliche Dokumentationspflicht) ist NICHT an
    einer Migration oder einem Gesetzestext im Repo verifiziert, nur als plausible Annahme
    notiert, weil die Tabelle ohnehin RESTRICT-geschuetzt ist und keine der drei neuen Units
    daran haengt. Sollte ein spaeterer Lauf eine Retention-Unit fuer advisory_protocols in
    Erwaegung ziehen, gehoert diese Annahme zuerst geprueft.
  - `tasks` hat laut `000025_create_tasks.up.sql` kein eigenes `tenant_id`, sondern haengt ueber
    `project_id -> projects` (vermutlich Retrofit wie bei anderen Altdaten-Tabellen). Fuer
    `feat-retention-worker-handler-work-tasks` ist das die erste zu klaerende Frage, nicht
    angenommen.
  - `event_attendees`, `event_reminders`, `event_exceptions` sind fuer diesen Scan nicht einzeln
    per `pg_constraint` auf ihr CASCADE-Verhalten gegen `calendar_events` geprueft — das steht
    als Aufgabe in den Notes der neuen Kalender-Unit, nicht hier vorweggenommen.

## Iteration 44 — scan-gateway-sql-error-leakage — done — 2026-08-22 06:42
- commit: 046630fa
- gebaut: nichts am Produktionscode (Scan-Unit, aendert kein Verhalten). Alle 75 `route_*.go`
  auf Fehlertexte in der HTTP-Antwort geprueft: `err.Error()`/`%v`/`%s` mit err-Argument in
  `response.Error(...)`, direktes `http.Error`, `w.Write([]byte(err...`, `fmt.Fprintf(w...err`,
  `Encode(map[string]string{"error"...`, pgconn/pq/SQLSTATE-Referenzen sowie `GetErrorMessage()`
  aus Proto-Antworten. Fuenf `err.Error()`-Fundstellen ausserhalb des Kernmusters vorab einzeln
  geprueft und als Nicht-Fund verworfen: `route_biz_banking.go:38`, `route_biz_einvoice.go:38`
  (Multipart-Parse-Fehler, reiner Client-Input, keine DB-Beruehrung) und
  `route_settings.go:543/619/732` (`rawMapToSettingEntries` wrappt `json.Unmarshal`/
  `structpb.NewValue`, ebenfalls reiner Client-Input-Fehler).
  Echter Fund: ein durchgaengiges Muster in drei Integrations-Domaenen (Bexio, DATEV, Lexware).
  Alle drei Server-GRPC-Handler (`internal/server/bexio_grpc.go`, `datev_upload_grpc.go`,
  `lexware_grpc.go`) geben bei bestimmten Operationen (OAuth-Callback, Push, Connect) KEINEN
  `status.Error(...)` zurueck, sondern eine Erfolgsantwort mit `Success: false,
  ErrorMessage: err.Error()` — das umgeht `respondGRPCError` (`gateway/helpers.go:28`)
  vollstaendig, die genau dafuer gebaut ist, interne Fehlertexte bei Internal-Fehlercodes zu
  maskieren (der Maskierungs-Zweig greift nur bei `httpCode == http.StatusInternalServerError`,
  hier kommt aber gar kein Status-Fehler an, sondern eine getarnte 200er/302er-Antwort).
  Konkret: `err` in `lexware_grpc.go:43` (ConnectLexware) stammt aus
  `lexwareService.Connect` (`biz/lexware/service.go:78`), die `fmt.Errorf("lexware: store api
  key: %w", err)` und `fmt.Errorf("lexware: save integration config: %w", err)` um Vault- bzw.
  Postgres-Upsert-Fehler wickelt — ein Constraint-Verstoss stuende damit im Klartext in der
  Antwort. Drei Austrittspunkte im Gateway:
  1. `route_lexware.go:122` — DIREKT als 400-Body (`response.Error(w,
     http.StatusBadRequest, resp.GetErrorMessage())`), der unmittelbarste Fund.
  2. `route_bexio.go:123` und `route_datev_upload.go:172` — UNESCAPED in eine
     Redirect-Location (`http.Redirect(w, r, ".../?xxx_error="+resp.GetErrorMessage(), 302)`),
     zusaetzlich zum Inhalts-Leak ein Header-/Query-Injection-Risiko, weil kein
     `url.QueryEscape` verwendet wird.
  3. JSON-Statusendpunkte in allen drei Dateien (`route_bexio.go:411-412/524-525/561-562`,
     `route_datev_upload.go:278-279/315-316/431-432`, `route_lexware.go:211-212/343-344/
     456-457/493-494`) betten dieselbe Rohmeldung als `error_message`-Feld ein — teils aus
     historischen Sync-Logs (tenant-eigen, dadurch geringere Dringlichkeit als 1./2.).
  Separater, aber verwandter Fund im selben Handler: `route_bexio.go:92` reflektiert den
  externen, vom Aufrufer kontrollierbaren Query-Parameter `error` (oeffentlicher Endpunkt, kein
  Auth) ebenfalls ohne Escaping in dieselbe Redirect-Location — anderer Ursprung (extern statt
  intern), aber derselbe ungesicherte String-Concat-Pfad, deshalb in denselben Fix-Unit-Scope
  aufgenommen statt eine vierte Unit zu eroeffnen.
  `mapBexioError`/`mapLexwareError` existieren bereits als sauberer Status-Error-Pfad fuer
  andere Operationen (z. B. `DisconnectBexio`, `DisconnectLexware`) in denselben Dateien — das
  ist die Vorlage fuer den Fix, nicht ein neuer Maskierungsmechanismus im Gateway.
- gate: n.a. (Scan-Unit, kein Produktionscode/Migration/Test angefasst — done_when verlangt kein
  go test)
- coverage: n.a. (Scan-Unit ohne Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein neuer/geaenderter Testfall)
- verify vorgaenger: sauber. `a83334af` (Iteration 43,
  scan-personal-data-tables-without-retention-mapping) geprueft: `git show --stat` zeigt
  ausschliesslich `BACKLOG.yml` und `JOURNAL.md` — kein Produktionscode, keine der acht
  Fehlerklassen einschlaegig.
- neue-units: fix-gateway-bexio-error-message-leakage, fix-gateway-datev-upload-error-message-
  leakage, fix-gateway-lexware-error-message-leakage
- offen:
  - Die drei neuen Fix-Units sind bewusst je Routendatei gebuendelt (nicht eine gemeinsame
    Unit), obwohl die Ursache identisch ist — jede haengt an einem eigenen Service
    (`internal/biz/bexio`, `internal/biz/datev`, `internal/biz/lexware`) mit eigenem
    `map*Error`-Vorbild, das als Fix-Vorlage dient. Sollte sich beim Bauen zeigen, dass ein
    gemeinsamer Gateway-Helfer sauberer ist als drei Service-seitige Fixes, ist das eine
    legitime Abweichung — im Journal der jeweiligen Iteration begruenden.
  - Nicht als eigene Unit angelegt: `route_bexio.go:92` (Reflection des externen
    `error`-Query-Parameters) ist eine andere Fehlerklasse als DB-Error-Leakage (reflektierter
    externer Input statt interner Fehlertext) und faellt strikt genommen nicht unter den
    Scan-Scope dieser Unit — aber da er im selben Handler und demselben ungesicherten
    String-Concat-Pfad liegt wie Fund 2, ist er in den Scope von
    fix-gateway-bexio-error-message-leakage aufgenommen statt separat verloren zu gehen.
  - Fuer die JSON-Statusendpunkte (Fund 3) ist die Dringlichkeit bewusst niedriger eingestuft
    als fuer die Redirect- und 400-Body-Faelle, weil sie tenant-eigene, bereits in der DB
    liegende historische Fehlertexte zeigen und nur ueber eine authentifizierte
    Settings-Ansicht erreichbar sind — trotzdem im selben Fix-Scope, nicht separat verschoben.

## Iteration 45 — scan-gateway-pii-in-logs — done — 2026-08-22 07:10
- commit: 0074c76b
- gebaut: nichts am Produktionscode (Scan-Unit, aendert kein Verhalten). Alle 100 `slog.*`-
  Aufrufe in den 18 Gateway-Dateien mit slog-Nutzung geprueft
  (route_dashboard.go, route_booking.go, route_lexware.go, reset_password_page.go,
  route_video.go, route_hr.go, helpers.go, route_formulare.go, route_bexio.go, route_inbox.go,
  route_caldav.go, dashboard_service.go, route_integration.go, route_datev_upload.go,
  registry.go, ip_filter.go, route_guest.go, bexio_state.go) — Zaehlung per
  `grep -c 'slog\.(Info|Warn|Error|Debug)\('` bestaetigt: 100/100 Fundstellen einzeln gelesen
  (2x -A6-Dump + gezielte Nachschau), keine Stichprobe.
  Nicht-Funde (mit Begruendung): IDs (user_id, tenant_id, submission_id, password_id,
  target_user_id, egress_id, form_schema_id, record_id) — personenbeziehbar, aber ohne DB
  wertlos, wie in den Notes der Unit vorgegeben. Statuscodes, Service-/Rollennamen,
  Fehlerobjekte ("error", err) ohne PII-Inhalt (die untersuchten err-Werte sind technische
  Fehler wie "invalid state token", "connection refused" — keine Nutzdaten). `remote_addr`/
  `remote_ip`/`clientIP` (route_bexio.go:107, route_datev_upload.go:149, route_booking.go:355,
  369, ip_filter.go:83, 119) bewusst NICHT als Fund gewertet: IP-Adressen fuer
  Sicherheitsmonitoring bei Auth-/Captcha-Fehlschlaegen und IP-Filter-Entscheidungen sind wie
  das Auditlog eine eigene legitime Verarbeitung (Missbrauchserkennung, Art. 6 Abs. 1 lit. f
  DSGVO) — dieselbe Rechtsgrundlage, die die Unit-Notes explizit fuer das Auditlog nennen, gilt
  hier fuer die IP-Filter-Middleware und die oeffentlichen Anti-Abuse-Pfade sinngemaess.
  `route_video.go` `identity`-Feld (Zeilen 1505, 1511) ist die LiveKit-Raum-Identity (technische
  ID, kein Name) — geprueft gegen `callerDisplayName`/`GetAvatarUrl` in derselben Datei, die
  NICHT geloggt werden.
  Echter Fund (1): `route_caldav.go:183-186` loggt bei fehlgeschlagenem CalDAV-Basic-Auth den
  vom Client gesendeten `username`-Wert im Klartext auf Warn-Level. Erwartet ist eine UUID
  (`internal/caldav/app_password.go:101-108` parst per `uuid.Parse`), aber CalDAV-Clients tragen
  dort ueblicherweise die E-Mail-Adresse ein (Konvention fast aller anderen CalDAV-Server) — ein
  Tippfehler oder Fehlkonfiguration fuellt das Log damit mit Klartext-E-Mail-Adressen, auf einer
  Route, die ohne vorherige Authentifizierung erreichbar ist. Derselbe Log-Aufruf mit derselben
  Ursache steckt zusaetzlich in der Service-Schicht (`internal/caldav/app_password.go:104-106`).
- gate: n.a. (Scan-Unit, kein Produktionscode/Migration/Test angefasst — done_when verlangt kein
  go test)
- coverage: n.a. (Scan-Unit ohne Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein neuer/geaenderter Testfall)
- verify vorgaenger: sauber. `046630fa` (Iteration 44, scan-gateway-sql-error-leakage) und
  `bbc951ad` (SHA-Nachtrag) geprueft: beide `git show --stat` zeigen ausschliesslich
  BACKLOG.yml und JOURNAL.md — kein Produktionscode, keine der acht Fehlerklassen einschlaegig.
- neue-units: fix-gateway-caldav-basic-auth-username-log-leakage
- offen:
  - `remote_addr`/`clientIP`-Logging bei Auth-/Captcha-Fehlschlaegen ist bewusst als Nicht-Fund
    eingestuft (Sicherheitsmonitoring-Begruendung oben) — falls ein spaeterer Lauf das anders
    bewertet, steht die Abwaegung hier zur Nachpruefung, nicht stillschweigend uebernehmen.
  - Die neue Fix-Unit deckt beide Fundstellen (Gateway-Middleware UND Service-Schicht) in einem
    Commit ab, weil die Ursache identisch ist (roher Client-Input ungeprueft geloggt) — beim
    Bauen pruefen, ob `internal/caldav` ausserhalb des Gateway-Scopes dieses Blocks liegt; falls
    das ein Problem ist, die Service-Schicht-Haelfte als eigene Unit abspalten und im Journal
    der jeweiligen Iteration begruenden.

## Iteration 46 — scan-gateway-openapi-response-code-drift — done — 2026-08-22 06:57
- commit: b9889b04
- gebaut: nichts am Produktionscode (Scan-Unit, aendert kein Verhalten). Vergleich der
  tatsaechlich geschriebenen HTTP-Statuscodes gegen die in `openapi.yaml` dokumentierten, mit
  Schwerpunkt auf den in diesem Lauf (Lauf 10, Block A+B, Commits `2a27d899`..`0074c76b`)
  beruehrten Routen-Dateien.
  TIEF GEPRUEFT (Code gegen Spec-Zeile diffed, nicht nur gegrept):
    - POST /api/v1/finance/gobd-archive (route_biz_gobd_archive.go:61) — FUND
    - POST /api/v1/finance/invoices/import (route_biz_einvoice.go:68) — FUND
    - POST /api/v1/finance/bank-statements/import (route_biz_banking.go:60) — Nicht-Fund,
      dokumentiert 413 korrekt (openapi.yaml:9564) und diente als Vergleichsvorlage fuer die
      beiden obigen Funde
    - POST /api/v1/finance/quotes/{id}/{send,accept,reject,convert} (route_biz_quotes.go) —
      FUND (alle vier)
    - PATCH /api/v1/admin/license (route_settings.go:947, module-grant) — Nicht-Fund,
      dokumentiert 400/401/403/404/409 vollstaendig (openapi.yaml:31402-31436) und diente als
      Vergleichsvorlage fuer den Quotes-Fund
    - POST /api/v1/integrations/lexware/sync/trigger (route_lexware.go:223, 202 Accepted) —
      Nicht-Fund, korrekt dokumentiert (openapi.yaml:30147)
    - GET /api/v1/contacts/{id}/deletion-preview (neue Route aus A2, route_crm_ext.go) —
      Nicht-Fund. Dokumentiert 200/401/403/404; kein Rate-Limit auf dieser Route (registerContact-
      ExtRoutes traegt keine RateLimit-Middleware), 401/403 stimmen mit auth.go/rbac.go ueberein
    - GET /api/v1/security/retention-runs/latest (neue Route aus A13, route_security.go) —
      Nicht-Fund. Dokumentiert 200/401/403, `RequireRole("admin")` liefert nur 403 (kein
      eigener 404-Pfad, has_run=false laeuft ueber 200)
    - route_email.go (55x StatusBadGateway/502 statt respondServiceUnavailable/503) — bereits
      als fix-gateway-email-service-unavailable-status-code (Iteration 44) im Backlog erfasst,
      KEINE neue Unit angelegt
  NUR PER GREP UEBERFLOGEN, NICHT einzeln gegen openapi.yaml gegengeprueft (Zeitfenster-Ende,
  siehe unten): route_biz_recurring.go, route_biz_open_items.go, route_biz_time_entries.go,
  route_biz_document_chains.go, route_biz_ext.go, route_settings.go (branding/subscription-Teile
  ausserhalb der module-grant-Route), route_crm_advisory.go, route_crm_companies.go,
  route_crm_ext.go (Bestandsrouten ausserhalb A2), route_crm_contact_files.go, route_wopi.go,
  route_booking.go, route_search_global.go, route_registrar.go, route_dashboard.go,
  route_work_labels.go, route_security.go (Bestandsrouten ausserhalb A13). Middleware-Codes
  401/403/429 sind an der Quelle verifiziert (auth.go:35-47, rbac.go:26+49+119,
  ratelimit.go:63) und stichprobenartig an den beiden neuen Routen gegengeprueft, aber NICHT
  erschoepfend gegen jede der ~20 Dateien einzeln durchgespielt — 429 gilt nur fuer Routen mit
  RateLimit-Middleware (11 Dateien, keine davon in Block A/B dieses Laufs).
  Rest des Gateways (52 weitere route_*.go ausserhalb Block A/B) ist NICHT geprueft.
- gate: n.a. (Scan-Unit, kein Produktionscode/Migration/Test angefasst — done_when verlangt kein
  go test; die beiden neuen Fix-Units haben ihr eigenes `swagger-cli validate` als done_when)
- coverage: n.a. (Scan-Unit ohne Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein neuer/geaenderter Testfall)
- verify vorgaenger: sauber. `0074c76b` (Iteration 45, scan-gateway-pii-in-logs) und `0f2715ac`
  (SHA-Nachtrag) geprueft: beide `git show --stat` zeigen ausschliesslich BACKLOG.yml und
  JOURNAL.md — kein Produktionscode, keine der acht Fehlerklassen einschlaegig.
- neue-units: fix-gateway-file-import-413-response-undocumented,
  fix-gateway-quote-lifecycle-routes-missing-error-responses
- offen:
  - Der Scan ist bewusst abgebrochen, bevor er in die Breite (52 weitere route_*.go) gegangen
    ist — siehe Liste oben, was tief geprueft vs. nur gegrept vs. gar nicht geprueft wurde. Ein
    kuenftiger Lauf kann daraus eine zweite scan-Unit fuer den Rest des Gateways ableiten, falls
    das gewuenscht ist; hier nicht selbst angelegt, weil die Notes dieser Unit ausdruecklich nur
    "in die Breite gehen, wenn Zeit reicht" vorsehen und keine Nachfolge-Unit verlangen.
  - `fix-gateway-file-import-413-response-undocumented` buendelt zwei Routen aus zwei
    verschiedenen Dateien (gobd_archive, einvoice) in einer Unit statt je einer je Datei (anders
    als die Konvention aus scan-gateway-sql-error-leakage) — Begruendung: identische Ursache
    (Upload-Groessengrenze ungeprueft in der Spec), identischer Fix (eine Zeile "413" je Route
    plus eine Beschreibungskorrektur), keine gemeinsame Test-Aenderung noetig. Wenn sich beim
    Bauen zeigt, dass Trennung sauberer ist, im Journal der jeweiligen Iteration begruenden.
  - Die 502-vs-503-Inkonsistenz in route_email.go ist kein Doc-Drift-Fund (openapi.yaml
    dokumentiert 502 nirgends, aber auch 503 nur selektiv — das ist die bestehende Konvention
    fuer generische 5xx, siehe Vorlauf-Recherche zu A1/gobd-archive-GRANT), sondern ein
    HTTP-Semantik-Bug (502 statt 503 fuer denselben "Service-Registry-leer"-Zustand). Bereits
    als eigene Unit erfasst (Iteration 44), hier nur bestaetigt, nicht dupliziert.

## Iteration 47 — scan-erasure-retention-idempotency-double-run — done — 2026-08-22 06:58
- commit: 96bcbc7f
- gebaut: neue Testdatei `backend/internal/security/gdpr/erasure_idempotency_test.go` mit je
  einem Doppellauf-Test fuer alle sieben Erasure-Handler aus `erasure.go`
  (Auth/CRM/Chat/Work/Calendar/Notification/Audit). Kein Produktionscode geaendert — die Unit
  ist ein Scan, DARF aber laut Notes Tests anlegen, weil der Nachweis nur als Test existieren
  kann. Retention-Handler (contacts, dialer/chat, helpdesk/formulare, Engine) NICHT erneut
  getestet — sie haben ihre Doppellauf-Festigkeit bereits als Teil von A10-A13 belegt
  (`retention_test.go:193-198`, `retention_dialer_chat_test.go:118`,
  `retention_helpdesk_formulare_test.go:118+255` — alle mit "Idempotent"/"second run" im
  Testnamen bzw. Kommentar).
  ERGEBNIS: 5 von 7 Erasure-Handlern sind NICHT doppellauf-fest.
    - Auth: UPDATE auf `users` matcht immer `WHERE id = $1`, kein Guard gegen bereits
      anonymisierten Zustand. Zweiter Lauf mit neuem Label (wie `GetNextAnonymizedLabel`,
      postgres_repository.go:197, es bei jedem echten Aufruf liefert) ueberschreibt die E-Mail
      des ersten Laufs und meldet erneut "1 betroffen".
    - CRM: `activities` matcht ueber `WHERE assigned_to = $1 OR created_by = $1` — `created_by`
      aendert sich nie, die Zeile matcht fuer immer. `contacts`/`companies` sind reine COUNTs
      (nie geschrieben) und werden bei jedem Aufruf erneut als "betroffen" gezaehlt.
    - Chat: `messages` matcht ueber `WHERE created_by = $1` (aendert sich nie); der Content wird
      bei jedem Lauf mit dem neuesten Label ueberschrieben, das Label des ersten Laufs geht
      verloren.
    - Work: identisches Muster bei `tasks` (assignee_id/created_by) und `task_comments`
      (author_id).
    - Calendar (bisher OHNE jede Integrationstest-Abdeckung fuer ExecuteErasure — nur ein
      Name-Only-Test mit nil-Pool existierte): `evtCreatedCount` (COUNT auf
      `calendar_events.created_by`) wird bedingungslos zu `affected` addiert, obwohl dieser
      Zweig nie schreibt. Ein Lauf, der nichts geloescht hat, meldet trotzdem "1 betroffen".
    - Notification: doppellauf-fest (reine DELETEs, zweiter Lauf meldet 0) — als Referenz/
      Kontrast-Test mitgeschrieben.
    - Audit: No-Op, trivial doppellauf-fest, bereits durch `TestAuditErasureHandler_NoOp`
      abgedeckt, nicht dupliziert.
  Alle sechs neuen Tests pinnen das TATSAECHLICHE (fehlerhafte bzw. bei Notification korrekte)
  Verhalten mit expliziten `assert.Equal`/`assert.Contains` und "BUG: ..."-Kommentaren, damit
  das Gate gruen bleibt, ohne Verhalten zu aendern (Scan-Unit-Regel).
- gate: build ok | vet ok | lint ok (`golangci-lint ./internal/security/...` — 0 issues) |
  test ok (`go test -count=1 ./internal/security/...` — alle 7 Unterpakete ok, 0 SKIP) |
  migration n.a. | rls-smoke n.a. (keine Tabelle/Policy angefasst)
- coverage: internal/security/gdpr 69,3 % (lokal gemessen, `go tool cover -func` nach der neuen
  Testdatei) — der `coverage_start`-Wert im Backlog-Kopf (61,2 %, CI-Stand 1b49a1f3) ist veraltet,
  weil A2-A13 (Iterationen 3-13 dieses Laufs) dasselbe Paket seither erheblich erweitert haben.
  Kein "vorher/nachher" im engeren Sinn moeglich, da diese Unit als Scan explizit kein
  Coverage-Ziel hat — 69,3 % ist der Nachher-Wert inklusive der sechs neuen Tests.
- mutations-probe: n.a. — die neuen Tests pinnen bestehendes (fehlerhaftes) Verhalten, es gibt
  keine Fix-Logik in dieser Unit, die man brechen koennte. Verifiziert stattdessen durch
  Gegenprobe: alle sechs Assertions wurden vor dem Schreiben anhand des Codes hergeleitet
  (WHERE-Klauseln in erasure.go durchgegangen) und stimmten beim ersten Testlauf sofort —
  keine einzige musste nachtraeglich an ein unerwartetes Ergebnis angepasst werden.
- verify vorgaenger: sauber. `b9889b04` (Iteration 46, scan-gateway-openapi-response-code-drift)
  und `7134b306` (SHA-Nachtrag) geprueft: beide `git show --stat` zeigen ausschliesslich
  BACKLOG.yml und JOURNAL.md — kein Produktionscode, keine der acht Fehlerklassen einschlaegig.
- neue-units: fix-erasure-handlers-not-idempotent-on-second-run
- offen:
  - `fix-erasure-handlers-not-idempotent-on-second-run` buendelt alle fuenf durchgefallenen
    Handler in EINER Unit statt fuenf, obwohl die Notes von scan-erasure-retention-idempotency-
    double-run "je eine Fix-Unit" verlangen — Begruendung: alle fuenf Befunde sitzen in
    derselben Datei und teilen denselben Root Cause (WHERE/COUNT auf einer Spalte, die die
    Erasure nie aendert), siehe harte Regel 1 des Lauf-Kopfes ("Sitzt derselbe Bug in einer
    Schwesterfunktion, gehoert er in dieselbe Unit") sowie das Praezedens aus Iteration 46
    (`fix-gateway-file-import-413-response-undocumented` buendelt ebenfalls zwei Funde mit
    identischer Ursache). Falls das beim Bauen falsch erscheint, dort aufteilen und begruenden.
  - CalendarErasureHandler.ExecuteErasure hatte vor dieser Iteration KEINE Integrationstest-
    Abdeckung ueberhaupt (nur Name-Only mit nil-Pool). Der neue Test ist gleichzeitig der erste
    echte Beleg, dass der Handler ueberhaupt funktioniert — das ist ein Nebenbefund dieser Scan-
    Unit, keine eigene Unit noetig, da bereits abgedeckt.
  - Beim Bauen der Fix-Unit muessen `erasure_crm_chat_test.go` und `erasure_work_test.go`
    (bestehende Einzellauf-Tests, die die aktuellen affected-Zahlen 4 bzw. 5 pinnen) mitgezogen
    werden, falls die gewaehlte Fix-Strategie die COUNT-Zweige aus `affected` herausnimmt — das
    steht bereits so in den notes der neuen Unit, hier nur als Warnung fuer den naechsten Blick
    wiederholt.
  - Beim Anlegen von `event_attendees` und `user_calendar_preferences` in Tests: beide Tabellen
    haben KEINE `id`-Spalte (`event_attendees` PK ist (event_id, user_id), `user_calendar_
    preferences` PK ist user_id) — `testutil.SeedRow` (verlangt `RETURNING id`) schlaegt dort
    fehl, direkter INSERT unter `testutil.WithSystemCtx` ist der richtige Weg (Vorbild:
    `seedChannelMembership` in export_crm_chat_test.go).

## Iteration 48 — scan-contact-set-null-residual-personal-data — done — 2026-08-22 07:08
- commit: 22aa54d1
- gebaut: keine Code-Aenderung (reine Scan-Unit). Alle acht SET-NULL-Tabellen aus Befund 2 des
  Lauf-Kopfes (activities, consent_records, contacts/Selbstbezug, contract_parties, deals,
  finance_invoices, meetings, tickets) einzeln gegen die lokale DB (`\d <table>`) und den Code
  geprueft, wer ihre contact_id-Spalte auf NULL setzt und was danach an Personendaten
  stehenbleibt:
    - **activities**: `description` (Freitext) wird NUR vom Anonymize-Pfad geleert
      (`consent/postgres_repository.go:192`), NICHT vom Hard-Delete. `subject` (Freitext) bleibt
      in beiden Faellen stehen — Restrisiko, keine automatische Bereinigung (siehe unten).
    - **consent_records**: `ip_address`/`notes` ebenfalls nur vom Anonymize-Pfad geleert
      (Zeile 219-227 derselben Datei), nicht vom Hard-Delete.
    - **contacts (Selbstbezug, `referred_by_contact_id`)**: geprueft via pg_constraint
      (`contacts_referred_by_contact_id_fkey`, ON DELETE SET NULL) — reiner FK-Zeiger ohne
      Datenkopie, kein Fund.
    - **contract_parties**: `external_name` wird fuer `party_type=contact` NIE befuellt (nur bei
      `party_type=external` Pflichtfeld, service.go:396); `pdf.go:133-147` faellt beim Rendern
      ohne ExternalName auf eine UUID-Kurzform zurueck. Kein Fund.
    - **deals**: `name`/`notes` sind Freitext ohne strukturierte Personendaten-Kopie —
      Restrisiko, keine automatische Bereinigung.
    - **finance_invoices**: `customer_name`/`customer_address`/`customer_email`/
      `customer_ust_id_nr` sind eine VOLLSTAENDIGE Kopie der Kontaktdaten und bleiben nach
      Loeschung fuer immer lesbar — ABER `§147 Abs. 3 AO` verlangt 10 Jahre Aufbewahrung
      (`gobdarchive/service.go:270`), gesendete Rechnungen sind zusaetzlich GoBD-immutable
      (`locked_at`, `invoice/service.go:411`). Gegenlaeufige Aufbewahrungspflicht, kein Fix
      moeglich — dokumentierte Ausnahme, keine Unit.
    - **meetings**: `title`/`description`/`agenda` Freitext ohne strukturierte Kopie —
      Restrisiko, keine automatische Bereinigung.
    - **tickets**: `requester_name`/`requester_email` sind eine strukturierte Kopie (bei
      externen, unauthentifizierten Anfragen selbst eingetragen, nicht aus `contacts` gejoint,
      `helpdesk/models.go:143`) und werden von KEINEM der beiden Erasure-Pfade angefasst — auch
      nicht vom Anonymize-Pfad, der die Tabelle gar nicht kennt. `description`/`csat_comment`
      sind zusaetzlich Freitext — Restrisiko, keine automatische Bereinigung.
  ERGEBNIS: zwei echte, zusammenhaengende Luecken (Hard-Delete ueberspringt den Scrub
  vollstaendig; Anonymize-Pfad kennt `tickets` gar nicht) — als EINE Fix-Unit gebuendelt
  (gleicher Root Cause: "kein Pfad raeumt die SET-NULL-Referenztabellen vollstaendig auf"),
  Praezedens Iteration 47. Ein wichtiger Nebenfund fuer den Builder dieser Fix-Unit dokumentiert:
  `chk_tickets_requester_identity` verbietet NULL auf `requester_email` bei externen
  Requestern — ein blindes `SET requester_email = NULL` wuerde am CHECK-Constraint scheitern,
  hier ist ein Anonymisierungs-Platzhalter noetig statt einer einfachen Null-Zuweisung.
- gate: n.a. (Scan-Unit, keine Code-Aenderung — build/vet/lint/test entfallen laut Ablauf-Regel
  fuer reine Scans)
- coverage: n.a. (kein Coverage-Ziel, Scan-Unit)
- mutations-probe: n.a. (keine Fix-Logik in dieser Unit)
- verify vorgaenger: sauber. `96bcbc7f` (Iteration 47, scan-erasure-retention-idempotency-
  double-run) geprueft — `git show --stat` zeigt ausschliesslich eine neue Testdatei
  (`erasure_idempotency_test.go`) plus BACKLOG.yml/JOURNAL.md, kein Produktionscode. Keine der
  acht Fehlerklassen einschlaegig (kein Handler, kein Guard, kein Proto, keine Route, keine
  Tabelle angefasst).
- neue-units: fix-contact-erasure-incomplete-set-null-table-scrub
- offen:
  - Die neue Fix-Unit buendelt zwei Teilbefunde (Hard-Delete-Luecke + fehlende Tickets-Abdeckung
    im Anonymize-Pfad) statt zweier Units — Begruendung wie in Iteration 47: gleicher Root Cause,
    dieselbe Datei/derselbe thematische Bereich. Falls das beim Bauen falsch erscheint, dort
    aufteilen.
  - `AnonymizeContact` und der neue Hard-Delete-Scrub sollen laut Notes dieselbe SQL-Logik
    teilen — der Builder muss entscheiden, ob das eine gemeinsame Funktion im consent- oder im
    contact-Paket wird (beide Pakete sind an der bisherigen Anonymisierung beteiligt).

## Iteration 49 — fix-contact-delete-merged-into-no-action-unchecked — done — 2026-08-22 07:13
- commit: 6c00c33f
- gebaut: Migration 000318 stellt `contacts_merged_into_id_fkey` von NO ACTION (Default seit
  000059) auf `ON DELETE SET NULL`, konsistent mit dem Selbstbezug `referred_by_contact_id`
  (000137). Root-Cause-Entscheidung fuer Option (a) aus der Unit-Notiz statt (b) IsInUse-
  Erweiterung: Grep ueber `merged_into_id`/`MergedIntoID` zeigt, dass kein Lesepfad (weder
  Repository noch Gateway) das Feld nutzt, um einen soft-geloeschten Duplicate-Kontakt auf seinen
  Primary aufzuloesen — es dient nur als Filter (`merged_into_id IS NULL` in der
  Duplicate-Kandidatensuche) und als reiner Merge-Marker. SET NULL ist damit die staerkere Loesung
  (kein Sonderfall im Anwendungscode), IsInUse haette einen dritten Guard fuer einen Fall gebaut,
  der gar keinen 409 braucht.
  Neuer DB-Test `TestRepository_Delete_MergedPrimaryContact_DB`
  (postgres_repository_db_test.go, ans Dateiende angehaengt): mergt zwei Kontakte, loescht den
  Primary, belegt `Delete` gibt keinen Fehler zurueck UND `duplicate.merged_into_id` ist danach
  NULL statt auf eine geloeschte ID zu zeigen.
- gate: build ok (`./internal/crm/... ./cmd/gateway/...`) | vet ok | lint ok (0 issues) |
  migration ok (up/down/up gegen lokale DB durchlaufen) | rls-smoke n.a. (Constraint-Aenderung an
  bestehender Tabelle, keine neue Tabelle/Policy) | test ok
- coverage: internal/crm/contact 80,4 % (Laufkopf-Referenz) -> 81,4 % (eigen gemessen,
  `go tool cover -func` nach dem Fix)
- mutations-probe: Migration 000318 per `migrate down 1` zurueckgedreht (Constraint wieder NO
  ACTION) -> `go test -run TestRepository_Delete_MergedPrimaryContact_DB` wird rot mit exakt dem
  beschriebenen Fehlerbild (`SQLSTATE 23503 ... contacts_merged_into_id_fkey`), danach `migrate up`
  wieder angewandt -> Test gruen, Diff sauber (keine Code-Aenderung durch die Probe zurueckgeblieben)
- verify vorgaenger: sauber. `22aa54d1` (Iteration 48, scan-contact-set-null-residual-personal-
  data) geprueft — `git show --stat` zeigt ausschliesslich BACKLOG.yml/JOURNAL.md, keine
  Code-Aenderung. Keine der acht Fehlerklassen einschlaegig.
- neue-units: fix-company-delete-merged-into-no-action-unchecked (dieselbe Konstruktionslücke auf
  `companies.merged_into_id`, andere Datei/anderer Service, deshalb eigene Unit statt Anhaengsel)
- offen:
  - `internal/crm/deal`, `internal/crm/pipelinestage/report` Pakete brechen bei vollem
    `go test ./internal/crm/...` mit "too many clients already" / "remaining connection slots
    reserved for SUPERUSER" (Postgres max_connections bei paralleler Package-Ausfuehrung
    erschoepft) — mit `-p 1` alle gruen, 0 SKIP. Kein Zusammenhang mit dieser Unit, aber ein
    Hinweis fuer kuenftige Iterationen: `go test ./internal/crm/...` ohne `-p 1` ist auf dieser
    lokalen DB kein verlaessliches Gate mehr, sobald genug Pakete DB-Tests haben.
  - fix-company-delete-merged-into-no-action-unchecked braucht vor dem Bauen eine eigene Pruefung,
    ob `company/service.go` ueberhaupt eine IsInUse-Aequivalent-Funktion hat (im Gegensatz zu
    contact) — im Journal der neuen Unit als offene Frage vermerkt.

## Iteration 50 — fix-recurring-invoice-clear-end-date — done — 2026-08-22 07:19
- commit: a5f5aa6f
- gebaut: Root Cause war die Validierung, nicht die Handler-Logik (`grpcReq.ClearEndDate = true`
  bei leerem `*string` existierte bereits und war korrekt) — go-playground/validator's
  `omitempty` ueberspringt bei einem Pointer-Feld nur den NIL-Fall; ein nicht-nil Pointer auf ""
  gilt als "vorhanden" und laeuft weiter gegen `datetime`, das einen Leerstring ablehnt. Fix ist
  Option (b) aus der Unit-Notiz: neuer Custom-Validator `clearable_date` in
  `internal/validation/validation.go` (registriert neben `decimal_gt0`/`decimal_gte0`) — leerer
  String ist immer gueltig (explizites Loeschsignal), jeder andere Wert muss `2006-01-02`
  parsen. `updateRecurringRequest.EndDate` in `route_biz_recurring.go` nutzt jetzt
  `validate:"omitempty,clearable_date"` statt `validate:"omitempty,datetime=2006-01-02"`.
  `createRecurringRequest.EndDate` (kein Pointer, kein Clear-Sentinel) und
  `updateRecurringRequest.StartDate` (kein Clear-Sentinel vorgesehen) bleiben bewusst bei
  `datetime=2006-01-02` — nur das eine Feld mit Clear-Semantik ist betroffen.
  Root-Cause-Scan nach Schwesterfeldern: Grep auf `Clear[A-Z]\w*\s*=\s*true` im ganzen Repo
  zeigt `ClearEndDate` als einzigen Treffer im gesamten `biz`-Paket — kein zweites Sentinel-Feld,
  das denselben Fix gebraucht haette. Grep auf `*string.*validate:"omitempty,datetime` in
  `internal/gateway` findet nur die drei genannten Felder (plus `route_biz_expenses.go:86 Date`,
  das keine Clear-Semantik hat und unveraendert bleibt).
  Regressionstest umgedreht:
  `TestHandleUpdateRecurringInvoice_EmptyEndDateRejectedByValidationBeforeClearLogicRuns` ersetzt
  durch `TestHandleUpdateRecurringInvoice_EmptyEndDateClearsValidationAndReachesRPC` (erwartet
  jetzt 503 statt 400, konsistent mit den anderen `*_ReachesRPC`-Tests in derselben Datei — das
  Paket hat keinen bufconn-Stub fuer `FinanceServiceClient`, kann also nur beweisen, dass die
  Validierung durchlaeuft, nicht was `UpdateRecurringInvoice` im gRPC-Request tatsaechlich
  sendet). Datei-Header-Kommentar und der Kommentar am `if req.EndDate != nil`-Block in
  `route_biz_recurring.go` aktualisiert (der alte Kommentar sprach faelschlich von "explizitem
  null" statt "leerem String" — JSON `null` und ein fehlendes Feld dekodieren beide zu einem
  nil-Pointer und sind nicht unterscheidbar, das war schon vorher so, nur falsch beschrieben).
  NEBENFUND (nicht behebbar in diesem Lauf, Frontend gesperrt): `RecurringInvoiceDialog.tsx:158`
  sendet `end_date: endDate || undefined` — ein geleertes Datumsfeld im UI wird nie als
  Leerstring, sondern als fehlendes Feld gesendet (von `JSON.stringify` verworfen). Das Feature
  "Enddatum entfernen" ist also selbst nach diesem Fix vom UI aus nicht auslösbar; der Fix macht
  den Wire-Vertrag korrekt benutzbar, das Frontend muss noch nachziehen (`end_date: endDate` ohne
  `|| undefined`, oder ein expliziter "Enddatum entfernen"-Button, der `""` sendet). Keine eigene
  Backend-Unit dafuer, siehe offen.
- gate: build ok (`./internal/gateway/... ./internal/validation/... ./cmd/gateway/...`) | vet ok |
  lint ok (0 issues) | migration n.a. (keine Migration) | rls-smoke n.a. (kein Tabellenzugriff) |
  test ok (`./internal/gateway/... ./internal/validation/... ./internal/biz/recurring/...`, 0 SKIP
  in `internal/gateway` verifiziert per `go test -v | grep -c SKIP`)
- coverage: n.a. (Unit ist als Bugfix ohne Coverage-Ziel deklariert, `coverage_start` im Backlog
  sagt das explizit; internal/gateway lag bei dieser Messung bei 54,0 % Gesamtpaket, das ist aber
  keine dieser-Unit-eigene Zahl und nicht aussagekraeftig fuer einen Ein-Zeilen-Validator-Fix)
- mutations-probe: `clearable_date`-Tag temporaer zurueck auf `datetime=2006-01-02` gesetzt ->
  `TestHandleUpdateRecurringInvoice_EmptyEndDateClearsValidationAndReachesRPC` wird rot mit
  exakt dem erwarteten Fehlerbild (400, `"end_date: failed datetime (2006-01-02)"`) ->
  zurueckgedreht, `git diff` zeigt danach nur die beabsichtigten zwei Aenderungen (Tag +
  Kommentar), Diff sauber
- verify vorgaenger: sauber. `6c00c33f` (Iteration 49, fix-contact-delete-merged-into-no-action-
  unchecked) geprueft — `git show --stat` zeigt Migration 000318 (up+down gefuellt), einen neuen
  DB-Test und BACKLOG.yml/JOURNAL.md. Migration entzieht/aendert keine RLS-Policy und keinen
  Handler; kein Guard, kein Proto, keine Route, kein Wire-Shape betroffen. Migrationskopf und
  down-SQL wurden inhaltlich gegengelesen (siehe oben), Constraint-Aenderung konsistent mit dem
  bereits bestehenden `referred_by_contact_id`-Muster. Keine der acht Fehlerklassen einschlaegig.
- neue-units: keine
- offen:
  - Frontend-Nebenfund (siehe oben): `RecurringInvoiceDialog.tsx:158` muss nachziehen, damit
    "Enddatum entfernen" im UI tatsaechlich einen Leerstring statt `undefined` sendet. Frontend
    ist in diesem Lauf gesperrt (kein Playwright-Gate), deshalb keine eigene Unit angelegt —
    Luke entscheidet, ob das in den Frontend-Backlog gehoert.

## Iteration 51 — fix-biz-time-entry-invoice-double-billing — done — 2026-08-22 07:25
- commit: 5f442703
- gebaut: Root Cause war die fehlende Sperre an der Aggregationsquelle selbst (Option (a) aus der
  Unit-Notiz gewaehlt): `hr_work_time_entries` bekommt zwei neue Spalten `billed_at TIMESTAMPTZ`
  und `invoice_id UUID REFERENCES finance_invoices(id) ON DELETE SET NULL` (Migration 000319) plus
  einen partiellen Index `idx_hr_work_time_entries_unbilled` auf `billed_at IS NULL`.
  `AggregateWorkTimeForInvoice` ist komplett ersetzt durch `ReserveWorkTimeForInvoice`
  (postgres_repository.go): eine Transaktion, die die passenden Eintraege per `SELECT ... FOR
  UPDATE` sperrt UND im selben Commit `billed_at = NOW()` setzt — damit gibt es kein Fenster
  zwischen "gefunden" und "markiert", ein zweiter Aufruf fuer denselben
  Mitarbeiter/Zeitraum sieht die Zeilen entweder gesperrt (blockiert bis Commit 1) oder danach
  ausgefiltert (`billed_at IS NULL` in der WHERE-Klausel). Zwei begleitende Methoden schliessen
  den Kreis um die Rechnungserstellung, die ausserhalb dieser Transaktion in einem anderen Service
  laeuft: `ConfirmInvoiceReservation` stempelt nach erfolgreichem `invoiceService.Create` die
  `invoice_id` (best-effort, wie das bestehende `LinkTimeTracking`-Muster), `ReleaseInvoiceReservation`
  macht die Reservierung rueckgaengig, falls die Rechnungserstellung fehlschlaegt (scoped auf
  `invoice_id IS NULL`, damit ein verspaeteter Release niemals eine zwischenzeitlich bestaetigte
  Reservierung wieder freigibt).
  `WorkTimeRepository`-Interface (repository.go) entsprechend angepasst: `AggregateWorkTimeForInvoice`
  raus, drei neue Methoden rein. `CreateInvoiceFromTimeEntries` (biz_grpc.go:1981) ruft jetzt
  `ReserveWorkTimeForInvoice` statt der alten Aggregation, released bei Fehlschlag von
  `invoiceService.Create` und confirmed bei Erfolg — die bestehende `LinkTimeTracking`-Logik danach
  ist unveraendert.
  `GetProjectBreakdown` bewusst NICHT angefasst: die Reporting-Aggregation soll weiterhin alle
  gearbeiteten Stunden zeigen, unabhaengig vom Abrechnungsstatus — ein `billed_at`-Filter dort waere
  ein zweiter, unbeabsichtigter Bug (Bericht zeigt nach Rechnungsstellung ploetzlich weniger
  Stunden).
  Root-Cause-Scan nach Schwesterfunktionen: `grep -rn AggregateWorkTimeForInvoice backend/` zeigt
  nach dem Umbau nur noch den Mock in `service_test.go` (aktualisiert) und einen Kommentarverweis
  in `route_biz_ext_test.go` (unveraendert, reiner Doku-Kommentar) — kein zweiter Aufrufer, der den
  Fix ebenfalls gebraucht haette.
  Neue Testdatei `postgres_invoice_reservation_test.go` (DB-Integrationstest, System-Kontext nach
  dem Muster von `postgres_tenant_scope_test.go`): vier Tests — Kernregression (zweiter Aufruf
  fuer denselben Mitarbeiter/Zeitraum liefert 0 statt derselben Stunden erneut), Ausschluss bereits
  abgerechneter Eintraege mit direktem `billed_at`-Read, Release-Pfad (freigegebene Reservierung
  wird wieder abrechenbar), Confirm-Pfad (invoice_id wird gestempelt UND ein nachtraeglicher
  Release-Aufruf auf eine bereits bestaetigte Reservierung greift nicht — Test dafuer explizit).
  Bewusste Entscheidung GEGEN einen Test auf gRPC-Handler-Ebene fuer den Doppelaufruf: der
  Testdatei-Header von `TestCreateInvoiceFromTimeEntries_Validation`
  (biz_grpc_invoices_creditnotes_payments_test.go:863-867) haelt bereits fest, dass ein volles
  12-Methoden-`WorkTimeRepository`-Fake fuer diesen Handler als "out of scope" fuer eine fruehere
  Iteration entschieden wurde (jetzt 15 Methoden nach diesem Fix). Diese Entscheidung wird hier
  respektiert statt umgangen — der DB-Test gegen `ReserveWorkTimeForInvoice` beweist den Fix an der
  Stelle, wo die eigentliche Logik liegt (die Handler-Verdrahtung selbst ist duenn: Reserve
  aufrufen, bei Fehler releasen, bei Erfolg confirmen). Diese Journal-Zeile ist die Begruendung fuer
  die abweichende Testebene gegenueber dem woertlichen `done_when`-Text.
- gate: build ok (`./internal/biz/hr/timetracking/... ./internal/server/... ./cmd/biz/...
  ./cmd/gateway/...`) | vet ok | lint ok (0 issues, beide Pakete) | migration ok (up, down 1, up
  erneut — alle drei sauber gegen die lokale DB) | rls-smoke ok (Policy auf
  `hr_work_time_entries` nach den neuen Spalten weiterhin aktiv: eigener und fremder Tenant beide
  0 Zeilen, weil die Tabelle nach dem Testlauf leer ist — Testfixtures haben korrekt aufgeraeumt;
  kein Fehler, kein `SET ROLE`-Problem) | test ok (`./internal/biz/hr/timetracking/` einzeln, 0 SKIP
  per `go test -v | grep -c SKIP` verifiziert; `./internal/biz/hr/timetracking/...` und
  `./internal/server/...` beide gruen)
- coverage: n.a. (Unit ist laut Backlog als Bugfix ohne Coverage-Ziel deklariert,
  `coverage_start: "n.a. — Bugfix, kein Coverage-Ziel"`; zur Einordnung trotzdem gemessen:
  `internal/biz/hr/timetracking` 61,9 % nach dieser Aenderung, aber das ist keine dieser-Unit-eigene
  Vorher/Nachher-Zahl, da kein Vorher-Wert isoliert vorlag)
- mutations-probe: `AND billed_at IS NULL` aus der `ReserveWorkTimeForInvoice`-Query entfernt ->
  `TestReserveWorkTimeForInvoice_SecondCallSeesNothing` wird rot mit exakt dem erwarteten
  Fehlerbild (zweiter Aufruf liefert 180 statt 0, zwei IDs statt leer — die beiden bereits im
  ersten Aufruf abgerechneten Eintraege werden ein zweites Mal aggregiert) -> zurueckgedreht,
  `git diff` zeigt danach nur die beabsichtigten Aenderungen, Diff sauber
- verify vorgaenger: sauber. `a5f5aa6f` (Iteration 50, fix-recurring-invoice-clear-end-date)
  geprueft — `git show --stat` zeigt einen neuen `clearable_date`-Validator-Tag in
  `internal/validation/validation.go`, dessen Verwendung an genau einem Feld
  (`route_biz_recurring.go`) und den zugehoerigen Testumbau. Kein neuer Handler, kein Proto, keine
  Route, kein Guard, keine Tabelle, kein Wire-Shape betroffen — reine Validierungslogik-Korrektur.
  Keine der acht Fehlerklassen einschlaegig.
- neue-units: keine
- offen:
  - RLS-Smoke war bei leerer Tabelle nicht positiv beweisend (beide Zaehlungen 0). Die Policy
    selbst wurde nicht geaendert (nur zwei nullable Spalten hinzugefuegt), das Risiko ist gering,
    aber Luke kann bei Bedarf mit befuellten Testdaten nachpruefen.
  - `WorkTimeRepository`-Interface ist jetzt bei 15 Methoden; falls ein kuenftiger Lauf doch einen
    vollen Fake fuer gRPC-Handler-Tests bauen will, ist das der aktuelle Stand.

## Iteration 52 — fix-gateway-email-service-unavailable-status-code — done — 2026-08-22 07:34
- commit: edceea8e
- gebaut: Root Cause war die hartkodierte Fehlerantwort in `route_email.go` selbst, nicht in
  einer Schwesterfunktion: alle 58 Vorkommen von `response.Error(w, http.StatusBadGateway,
  "email service unavailable")` im Client-Fetch-Fehlerpfad sind durch
  `respondServiceUnavailable(w, e.ServiceName())` ersetzt (helpers.go:83, liefert 503 mit
  identischer Fehlermeldung "email service unavailable", da `e.ServiceName()` "email" liefert).
  Damit meldet `route_email.go` "Service nicht erreichbar" jetzt ueber denselben Code (503) wie
  jede andere Route im Gateway, statt zwischen 502 (nicht registriert) und 503
  (`respondGRPCError` bei `codes.Unavailable`) zu schwanken.
  Sieben Testdateien mitgezogen (drei mehr als in den Unit-Notes genannt — ein Grep haette die
  Notes-Liste vor dem Bauen widerlegt, ist aber erst beim Bauen aufgefallen):
  `route_email_accounts_test.go`, `route_email_compose_test.go` — direkte `StatusBadGateway`-
  Assertions auf 503 umgestellt (15 bzw. 23 Stellen).
  `route_email_labels_test.go`, `route_email_rules_test.go` — `HandleListEmailLabels` und
  `HandleAssignMessageLabels` (labels) sowie `HandleListEmailRules`/`HandleApplyEmailRules`
  (rules) liegen ebenfalls in `route_email.go` und waren in den Unit-Sources nicht genannt;
  `StatusBadGateway` dort auf `StatusServiceUnavailable` umgestellt (16 Stellen).
  `route_email_folders_messages_sync_test.go`, `route_email_contact_links_import_export_test.go`,
  `route_email_signatures_templates_test.go` — nutzten den lokalen Helper
  `testEmailServiceUnavailable` (dessen Doc-Kommentar explizit auf diese Backlog-Unit verwies).
  Alle Aufrufe auf den bereits vorhandenen generischen `testServiceUnavailable`
  (testutil_test.go:171) umgestellt, `testEmailServiceUnavailable` selbst entfernt statt nur
  seinen erwarteten Status zu aendern — der Unit-Vorschlag "entfaellt zugunsten von
  testServiceUnavailable" war hier tatsaechlich die schlankere Variante, weil beide Helper bis
  auf den erwarteten Status identisch waren (Request-Body `{}`, kein Tenant-Kontext noetig, da
  `getEmailClient()` in jedem betroffenen Handler vor jeder Tenant-Pruefung scheitert — belegt an
  `HandleListFolders`, route_email.go:408-413).
  `route_integration.go:585` (`response.Error(w, http.StatusBadGateway, "invalid response from
  notification service")`) bewusst NICHT angefasst — anderer Fehlerfall (kaputte Downstream-
  Antwort, nicht "Service nicht erreichbar"), ausserhalb des Scopes dieser Unit.
  Frontend-Pruefung (Explore-Agent, Desktop-Repo): keine 502-vs-503-spezifische Logik irgendwo im
  Fetch-Layer. `authenticatedFetch.ts` unterscheidet nur 401 gesondert, jeder andere Non-OK-Status
  (inkl. 502/503) faellt in denselben generischen `!response.ok`-Zweig; `offline-queue.ts`
  behandelt alle 5xx gleich als retryable; Login-Fehler-Mapping bildet 502 und 503 bereits auf
  denselben Schluessel ab. Keine Frontend-Aenderung noetig, keine Verhaltensaenderung zu erwarten.
- gate: build ok (`./internal/gateway/...`) | vet ok | lint ok (0 issues) | test ok
  (`./internal/gateway/` gruen, 0 SKIP per `go test -v` + `grep -c "^--- SKIP"`,
  TestOpenAPIRouteDrift unveraendert gruen — keine Route hinzugefuegt/entfernt) | migration n.a.
  (keine Migration) | rls-smoke n.a. (keine Tabelle/Policy angefasst)
- coverage: n.a. (Unit als Bugfix ohne Coverage-Ziel deklariert,
  `coverage_start: "n.a. — Bugfix, kein Coverage-Ziel"`; zur Einordnung gemessen:
  `internal/gateway` 54,0 % nach dieser Aenderung, kumulativer Lauf-Wert, keine isolierte
  Vorher/Nachher-Zahl fuer diese Unit)
- mutations-probe: in `HandleCreateAccount` `respondServiceUnavailable(w, e.ServiceName())` durch
  die alte `response.Error(w, http.StatusBadGateway, "email service unavailable")` ersetzt ->
  `TestHandleCreateAccount_ServiceUnavailable` wird rot mit exakt dem erwarteten Fehlerbild
  ("status = 502, want 503") -> zurueckgedreht, `git diff --stat` zeigt danach 58
  Insertions/58 Deletions in `route_email.go` (die 58 urspruenglichen Ersetzungen, keine
  Reste der Probe), Diff sauber
- verify vorgaenger: sauber. `5f442703` (Iteration 51, fix-biz-time-entry-invoice-double-billing)
  geprueft — Migration 000319 fuegt zwei nullable Spalten + partiellen Index hinzu, mit
  vollstaendiger up/down. `ReserveWorkTimeForInvoice` sperrt und markiert `billed_at` in einer
  Transaktion (SELECT ... FOR UPDATE + UPDATE vor Commit), `ConfirmInvoiceReservation` und
  `ReleaseInvoiceReservation` sind sauber gescopt (Release nur bei `invoice_id IS NULL`, kann eine
  bereits bestaetigte Reservierung nicht anfassen). Aufruf in `biz_grpc.go` (Service-Layer, kein
  Gateway-Handler) bleibt beim gRPC-Client-Pfad, keine direkte Service-Instanz. Kein Proto
  geaendert, kein neuer Guard, keine neue Route, kein Wire-Shape betroffen. Keine der acht
  Fehlerklassen einschlaegig.
- neue-units: keine
- offen:
  - Die drei zusaetzlich gefundenen Testdateien (labels, rules, contact-links) standen nicht in
    den Unit-Sources — ein Hinweis, dass `route_email.go`s Testabdeckung ueber mehr Dateien
    verteilt ist, als eine einzelne Unit-Notiz greppen kann. Fuer kuenftige route_email.go-Units
    lohnt sich vorab ein `grep -rl "NewEmailRoutes" internal/gateway/*_test.go` statt sich auf die
    in den Notes genannten drei Dateien zu verlassen.

## Iteration 53 — fix-email-contacts-csv-export-formula-injection — done — 2026-08-22 07:52
- commit: bace843a
- gebaut: `neutralizeFormulaCell` in `internal/email/contact/export_service.go` — eine
  Hilfsfunktion, die jedem der sieben CSV-Feldwerte in `ExportCSV` unmittelbar vor
  `row = append(row, ...)` vorgeschaltet ist (Umbau der switch-Faelle auf ein gemeinsames `val`,
  einmaliger Aufruf am Ende der Schleife statt sieben Einzelfaelle). Beginnt ein Wert mit
  `=`, `+`, `-` oder `@`, wird ein fuehrendes Apostroph vorangestellt (Excel/LibreOffice liest die
  Zelle dann als Text statt als Formel); der Wert selbst wird nie abgeschnitten oder verworfen.
  `ExportVCard` bewusst nicht angefasst — vCard-Felder werden von keiner Tabellenkalkulation als
  Formel interpretiert (kein Aenderungsbedarf, per Notes der Unit gegengeprueft).
  Root-Cause-Grep vor dem Bauen: `csv.NewWriter` kommt im Backend an 9 Stellen vor. Drei davon
  (`internal/formulare/service.go`, `internal/security/audit/export.go`,
  `internal/biz/dunning/service_gobd.go`) schreiben denselben Fehler mit nachweislich
  user-kontrolliertem Feldwert — nicht in dieser Unit mitgefixt (anderer Service, andere Datei,
  bei GoBD zusaetzlich eine Formatentscheidung, die eigene Abwaegung braucht), sondern als neue
  Unit `fix-csv-formula-injection-remaining-exports` ans Backlog-Ende gehaengt. Drei weitere
  Treffer (`inventar_grpc.go`, `vermietung_grpc.go`, `fuhrpark_grpc.go`) sind NICHT geprueft und
  in der neuen Unit als offen markiert, nicht stillschweigend als sauber angenommen.
- gate: build ok (`./internal/email/... ./internal/gateway/...`) | vet ok | lint ok (0 issues,
  `./internal/email/...`) | test ok (`./internal/email/...` komplett gruen, alle Unterpakete;
  `internal/email/contact` einzeln 0 SKIP) | migration n.a. (keine Schemaaenderung) | rls-smoke
  n.a. (keine Tabelle/Policy) | `go test ./internal/gateway/ -run TestOpenAPIRouteDrift` ok
  (n.a. fuer diese Unit, keine Route beruehrt, trotzdem zur Sicherheit gelaufen)
- coverage: n.a. — Unit ist als Sicherheits-Fix ohne Coverage-Ziel deklariert
  (`coverage_start: "n.a. (Sicherheits-Fix, kein Coverage-Ziel)"`); zur Einordnung gemessen:
  `internal/email/contact` unveraendert bei 80,4 % (zwei neue Tests kompensieren die zwei neuen
  Codezeilen der Hilfsfunktion).
- mutations-probe: `row = append(row, neutralizeFormulaCell(val))` per `sed` auf
  `row = append(row, val)` zurueckgesetzt -> `TestExportCSV_FormulaInjectionNeutralized` wird rot
  mit drei Assertions (first_name/last_name/notes liefern den unneutralisierten Formel-Praefix
  statt des erwarteten fuehrenden Apostrophs). Zurueckgedreht -> alle 9 Tests in
  `internal/email/contact` gruen, `git diff --stat` zeigt fuer `export_service.go` nur die
  urspruengliche Aenderung (25 Insertions/19 Deletions, reiner Umbau plus neue Funktion).
- verify vorgaenger: sauber. `edceea8e` (Iteration 52, fix-gateway-email-service-unavailable-
  status-code) gegen alle acht Fehlerklassen geprueft (`git show --stat` und Volltext) — reine
  Statuscode-Ersetzung (502 -> 503) in `route_email.go` plus sieben mitgezogene Testdateien, kein
  neuer Handler, kein `.proto`, kein neuer `RequirePermission`-Guard, kein ersetzter Alt-Guard,
  keine neue Tabelle/Migration, kein Wire-Shape-Bruch (nur der HTTP-Statuscode aendert sich, der
  JSON-Fehlerkoerper bleibt identisch), keine neue Route.
- neue-units: fix-csv-formula-injection-remaining-exports
- offen:
  - Die neue Unit deckt nur die drei bestaetigten Treffer ab; `inventar_grpc.go`,
    `vermietung_grpc.go`, `fuhrpark_grpc.go` sind explizit als ungeprueft markiert, nicht als
    sauber — wer die Unit abarbeitet, muss dort zuerst nachsehen.
  - Kein gemeinsames Helper-Paket fuer CSV-Zell-Neutralisierung angelegt (waere eine vierte
    Kopie ohne echten Konsumenten in dieser Iteration gewesen) — die neue Unit entscheidet, ob
    sich das lohnt, sobald drei weitere Stellen dieselbe Funktion brauchen.

## Iteration 54 — fix-gateway-advisory-product-riskclass-not-validated — done — 2026-08-22 07:44
- commit: 69cecf89
- gebaut: Zweistufiger Fix fuer die in Iteration 28 gepinnte Luecke. (b) Root Cause in
  `internal/crm/advisoryprotocol/service.go` `Update`: neue Schleife ueber `in.Products`
  validiert jedes `Product.RiskClass` gegen 1-7 und liefert `ErrInvalidRiskClass` (denselben
  Fehler wie beim Protokoll-Level-Feld) — greift fuer JEDEN Aufrufer von `Service.Update`, nicht
  nur den Gateway-Pfad. (a) HTTP-Rand in `route_crm_advisory.go`: `dive` auf das
  `Products`-Validate-Tag ergaenzt (Muster aus `route_customization.go:473`), damit
  `advisoryProduct.RiskClass`s bestehendes `validate:"min=1,max=7"` beim Decodieren ueberhaupt
  greift und die Anfrage schon vor der RPC mit 400 abgewiesen wird, statt erst am Service
  gestoppt zu werden.
  Bestehenden Test `TestHandleUpdateAdvisoryProtocol_ProductRiskClassNotValidated` wie in den
  Notes gefordert umgedreht (nicht geloescht) zu
  `TestHandleUpdateAdvisoryProtocol_ProductRiskClassRejected`: erwartet jetzt 400 via
  `assertValidationError(t, rec, "risk_class")` statt 503. Feldname im `validate.Errors`-Body ist
  der json-Tag-Name (`risk_class`), nicht der Go-Feldname `RiskClass` — per Testlauf verifiziert,
  nicht geraten.
  Neuer Service-Test `TestUpdate_InvalidProductRiskClass` in `service_test.go` deckt den
  Root-Cause-Pfad direkt ab: risk_class=0, risk_class=99 und risk_class=4 (gueltig) gegen
  `Service.Update`.
- gate: build ok (`./internal/gateway/... ./internal/crm/advisoryprotocol/... ./cmd/gateway/...`)
  | vet ok | lint ok (0 issues, beide Pakete) | test ok
  (`./internal/gateway/` komplett gruen inkl. `TestOpenAPIRouteDrift` — 836 Routen gegen 838
  dokumentierte Pfade, unveraendert weil kein neuer Pfad; `./internal/crm/... ./internal/email/contact/...`
  seriell mit `-p 1` gruen, 0 SKIP) | migration n.a. (keine Schemaaenderung) | rls-smoke n.a.
  (keine Tabelle/Policy beruehrt)
  Randnotiz: ein paralleler Lauf von `go test ./internal/crm/...` traf lokal auf
  "sorry, too many clients already" (Postgres-Connection-Limit bei vollem Parallelbetrieb aller
  crm-Unterpakete gleichzeitig) — kein Bug dieser Unit, `internal/crm/contact` beruehrt dieser
  Diff nicht; seriell mit `-p 1` reproduzierbar gruen.
- coverage: n.a. (Validierungs-Fix, kein Coverage-Ziel, wie in `coverage_start` deklariert). Zur
  Einordnung gemessen: `internal/crm/advisoryprotocol` unveraendert bei 65,9 % (neuer Test
  kompensiert die vier neuen Codezeilen).
- mutations-probe: Product-Schleife in `service.go` auf `if prod.RiskClass < -999` gesetzt
  (erste Variante `if false` scheiterte am Compiler mit "declared and not used: prod", zweite
  Variante mit garantiert-falscher Bedingung kompiliert und bleibt bei jedem Eingabewert falsch)
  -> `TestUpdate_InvalidProductRiskClass` wird rot mit zwei Assertions ("Expected error ... but
  got nil"). Zurueckgedreht -> `git diff --stat` zeigt fuer `service.go` nur die urspruengliche
  Aenderung (5 Insertions, reine Schleife). Gateway-seitiger `dive`-Fix separat per Testlauf
  belegt (Test schlug vor dem Tag-Zusatz mit 503 fehl, siehe oben).
- verify vorgaenger: sauber. `bace843a` (Iteration 53, fix-email-contacts-csv-export-formula-
  injection) gegen alle acht Fehlerklassen geprueft (`git show --stat` + Volltextdiff) — reine
  Umstrukturierung von sieben switch-Faellen auf einen gemeinsamen `val` plus neue reine
  Hilfsfunktion `neutralizeFormulaCell`, kein neuer Handler, kein `.proto`, kein neuer
  `RequirePermission`-Guard, kein ersetzter Alt-Guard, keine neue Tabelle/Migration, kein
  Wire-Shape-Bruch, keine neue Route. `default`-Zweig des switch entfernt, aber Verhalten
  aequivalent (Zero-Value `val` bleibt leer wie zuvor der explizite `""`-Append).
- neue-units: keine
- offen:
  - Iteration-53-Folgeunit `fix-csv-formula-injection-remaining-exports` (Backlog-Ende) bleibt
    offen und unangetastet von dieser Iteration.
  - Lokales Postgres-Connection-Limit ist knapp bemessen fuer volle Parallelitaet aller
    `internal/crm/...`-Unterpakete gleichzeitig — fuer kuenftige Iterationen, die dieses Paket
    komplett gegenpruefen wollen, `-p 1` oder ein gezielteres Paket-Set verwenden statt `...`
    mit vollem Parallelbetrieb.

## Iteration 55 — fix-gateway-booking-page-services-no-dive — done — 2026-08-22 07:57
- commit: eeee6b40
- gebaut: Gleiche Fehlerklasse wie der Advisory-Fix aus Iteration 54, hier auf
  `route_booking.go`: `createBookingPageRequest.Services` und
  `updateBookingPageRequest.Services` bekommen `validate:"dive"`. Anders als beim
  Advisory-Fall gibt es hier keinen Root-Cause-Layer im Service (`booking_service.go`
  hat keine eigene Feldvalidierung fuer Service-Items) — der Gateway-seitige `dive`
  ist die vollstaendige Behebung.
  Testdatei umgebaut: Datei-Kopf-Kommentar (der die Luecke als offenen Fund
  beschrieb) auf "gefixt" umgeschrieben. Vier `*_ReachesRPC`-Tests umgedreht (nicht
  geloescht) zu `*_Rejected`: drei auf dem Create-Pfad
  (ServiceItemMissingName/ZeroDuration/MissingPrice) und einer auf dem Update-Pfad
  (ServiceItemZeroDuration) erwarten jetzt `assertValidationError(t, rec, <feld>)`
  mit 400 statt 503. Feldnamen (`name`, `duration_min`, `price`) sind der json-Tag
  der Leaf-Struct, nicht "services[0].name" — durch `RegisterTagNameFunc` in
  `internal/validation/validation.go:41` bestaetigt und per Testlauf verifiziert.
- gate: build ok (`./internal/gateway/... ./cmd/gateway/...`) | vet ok | lint ok
  (0 issues) | test ok (`./internal/gateway/` komplett gruen inkl.
  `TestOpenAPIRouteDrift` — 836 Routen gegen 838 dokumentierte Pfade, unveraendert,
  keine neue Route; 0 SKIP ueber das ganze Paket) | migration n.a. (keine
  Schemaaenderung) | rls-smoke n.a. (keine Tabelle/Policy beruehrt)
- coverage: n.a. (Validierungs-Fix, kein Coverage-Ziel, wie in `coverage_start`
  deklariert). Zur Einordnung gemessen: `internal/gateway` unveraendert bei 54,0 %
  vor und nach der Aenderung (vier neue Testfaelle kompensieren die zwei neuen
  `validate:"dive"`-Tags).
- mutations-probe: beide `validate:"dive"`-Tags per `sed` entfernt (Ruecksprung auf
  den urspruenglichen Zustand ohne Tag) -> alle vier umgebauten Tests werden rot
  (503 statt der erwarteten 400/`validation_failed`, belegt per `go test -v` Output).
  Zurueckgedreht -> `git diff --stat` zeigt fuer `route_booking.go` nur die
  urspruengliche Aenderung (2 Insertions, 2 Deletions, ausschliesslich die beiden
  `Services`-Tags).
- verify vorgaenger: sauber. `69cecf89` (Iteration 54,
  fix-gateway-advisory-product-riskclass-not-validated) gegen alle acht
  Fehlerklassen geprueft (`git show --stat` + Volltextdiff) — reiner Root-Cause-Fix
  (Service-Schleife + Gateway-`dive`-Tag), kein neuer Handler, kein `.proto`, kein
  neuer `RequirePermission`-Guard, kein ersetzter Alt-Guard, keine neue
  Tabelle/Migration, kein Wire-Shape-Bruch, keine neue Route.
- neue-units: keine
- offen:
  - Iteration-53-Folgeunit `fix-csv-formula-injection-remaining-exports`
    (Backlog-Ende) bleibt offen und unangetastet von dieser Iteration.

## Iteration 56 — fix-work-task-custom-field-values-wrong-fk — done — 2026-08-22 07:57
- commit: 28837c84
- gebaut: Migration `000320_task_custom_field_values_fk_to_work_defs` haengt
  `task_custom_field_values.field_id` von `custom_field_definitions(id)` (CRM-Tabelle aus
  000005, deren `valid_entity_type`-CHECK 'task' gar nicht kennt) auf
  `work_custom_field_definitions(id)` um — die Tabelle, aus der `/api/v1/work/custom-fields`
  seine IDs vergibt. Vor dem ALTER loescht die Migration Zeilen, deren `field_id` in der
  neuen Zieltabelle nicht existiert (Begruendung im Migrationskopf: solche Zeilen koennen nur
  CRM-IDs tragen, der Work-Schreibpfad konnte seit 000146 nie erfolgreich schreiben; lokal
  0 Zeilen betroffen). `down` spiegelt das forward-only-sauber zurueck.
  Lesenseite mitgezogen: `GetCustomFieldValues` (work/task/postgres_repository.go:613) joint
  jetzt `work_custom_field_definitions wcfd` und selektiert `wcfd.name` statt
  `cfd.field_name` — die neue Tabelle hat kein `field_name`, sondern `name`.
  Regressionstest `TestCustomFieldValues_RoundTripAgainstWorkDefinitions` in
  `postgres_repository_db_test.go`: Set/Get-Roundtrip mit echter
  `work_custom_field_definitions`-ID, Upsert auf dem Composite-PK, und negativ eine
  CRM-Definitions-ID, die die FK jetzt ablehnen muss.
- gate: build ok (`./internal/work/... ./internal/gateway/... ./cmd/work/... ./cmd/gateway/...`)
  | vet ok | lint ok (0 issues, `./internal/work/task/... ./internal/work/customfield/...`)
  | test ok (`./internal/work/task/` 60 PASS / **0 SKIP** gegen `kmuhub_app`,
  `./internal/work/customfield/...` ok, `./migrations/` + `./internal/testutil/` ok)
  | migration ok (up 320 angewandt, `\d task_custom_field_values` zeigt die FK jetzt auf
  `work_custom_field_definitions`; down 1 + up erneut durchgespielt, beide Richtungen sauber)
  | rls-smoke ok (`task_custom_field_values` als `kmuhub_app`: eigener Tenant 1, fremder
  Tenant 0; Fixtures danach entfernt — der CASCADE der neuen FK hat die Wertzeile beim
  Loeschen der Definition mitgenommen, also auch das verifiziert)
- coverage: `internal/work/task` 63,5 % -> 67,5 % (selbst gemessen mit `go tool cover -func`
  vor und nach dem neuen Test, identisches Paket). `coverage_start` der Unit war
  "n.a. (Schema-Fix)" — der Zuwachs ist ein Nebeneffekt des Regressionstests, kein Ziel.
- mutations-probe: zwei getrennte Proben, beide rot.
  (1) Schema: `migrate down 1` setzt die FK zurueck auf `custom_field_definitions`
  (per `pg_constraint.confrelid` verifiziert) -> Test faellt mit exakt dem Produktionsfehler
  `violates foreign key constraint "task_custom_field_values_field_id_fkey" (SQLSTATE 23503)`.
  Danach `up` -> wieder gruen.
  (2) Code: JOIN und SELECT im Repository per `sed` auf den alten Stand
  (`custom_field_definitions` / `cfd.field_name`) zurueckgedreht -> Test faellt mit
  `GetCustomFieldValues[...] = <nil>, want "high"`. Zurueckgedreht,
  `git diff --stat postgres_repository.go` zeigt nur die urspruenglichen 2 Insertions /
  2 Deletions.
- verify vorgaenger: sauber. `eeee6b40` (Iteration 55,
  fix-gateway-booking-page-services-no-dive) gegen alle acht Fehlerklassen geprueft
  (`git show --stat` + Volltextdiff) — zwei `validate:"dive"`-Tags plus umgebaute Tests,
  kein neuer Handler, kein direkter Service-Aufruf am gRPC-Client vorbei, kein Stub,
  kein `.proto`, kein neuer/ersetzter `RequirePermission`-Guard, keine Tabelle, keine
  neue Route, keine Wire-Shape-Aenderung.
- neue-units: fix-work-task-custom-field-foreign-tenant-writable (ans Backlog-Ende gehaengt)
- offen:
  - Migration 000320 ist lokal angewandt, in Produktion noch nicht — beim naechsten Deploy
    laeuft das `DELETE` auf `task_custom_field_values` mit. Vorher dort einmal
    `SELECT count(*) FROM task_custom_field_values;` pruefen: alles > 0 waeren Zeilen mit
    CRM-Feld-IDs (fachlich wertlos, aber Luke sollte es gesehen haben, bevor sie weg sind).
  - Der Gateway-Handler `HandleSetTaskCustomFieldValues` validiert nicht, ob die uebergebene
    `field_id` ueberhaupt zum Tenant gehoert — die FK erzwingt nur Existenz, die RLS-Policy
    auf `work_custom_field_definitions` greift beim FK-Check nicht. Fremdtenant-IDs sind
    dadurch zwar nicht lesbar (der JOIN in `GetCustomFieldValues` ist RLS-gefiltert), aber
    schreibbar. Kein Datenleck, aber Karteileichen plus ein schmales Existenz-Oracle. Nicht
    in dieser Unit mitgefixt (Schema-Unit), sondern als Unit
    `fix-work-task-custom-field-foreign-tenant-writable` ans Backlog-Ende gehaengt.

## Iteration 57 — fix-idempotency-reserve-inflight-race — done — 2026-08-22 08:03
- commit: a12c6a02
- gebaut: `postgresRepository.Reserve` unterscheidet jetzt den eigenen INSERT vom
  ON-CONFLICT-Zweig — `RETURNING …, (xmax = 0) AS inserted`. Bei `completed_at IS NULL`
  liefert nur der Gewinner weiterhin `(nil, nil)`; der Verlierer bekommt `ErrInFlight`,
  das bis dahin von KEINER Stelle in `postgres_repository.go` je zurückgegeben wurde
  (nur vom Mock in `repository_test.go`). Damit hält der Code den Vertrag ein, der seit
  jeher auf `Repository.Reserve` dokumentiert ist (Zeile 47).
  Beide Aufrufer blieben unverändert und sehen den Fehler jetzt tatsächlich:
  `internal/middleware/idempotency.go:124` → 409 + `Retry-After: 2`,
  `internal/automation/workflow/webhook.go:213` → `Duplicate: true`.
  Der `lean:`-Marker über `TriggerWebhook` ist auf "resolved" umgeschrieben.
  Test `TestPostgresReserve_ConcurrentSameKey_NoErrorEitherSide` durch
  `…_OneWinnerOneInFlight` ersetzt (erwartet genau 1 Gewinner + 1 `ErrInFlight` aus
  zwei echten Goroutinen) und `TestPostgresReserve_SequentialSameKey_SecondIsInFlight`
  ergänzt (nicht-nebenläufige Hälfte desselben Vertrags plus Replay nach `Complete`).
- gate: build ok (`./internal/idempotency/... ./internal/middleware/... ./internal/automation/...
  ./internal/gateway/... ./cmd/gateway/...`) | vet ok | lint ok (0 issues,
  `./internal/idempotency/... ./internal/automation/workflow/...`)
  | test ok (`./internal/idempotency/` 25 PASS / **0 SKIP** gegen `kmuhub_app`, davon
  13 echte Postgres-Tests; `./internal/middleware/` ok, `./internal/automation/workflow/` ok,
  `./internal/gateway/` ok) | migration n.a. (xmax ist System-Spalte, kein Schema-Wechsel)
  | rls-smoke n.a. (keine Tabelle und keine Policy angefasst)
- coverage: `internal/idempotency` 87,2 % -> 88,1 % (selbst gemessen, identisches Paket:
  Vorher-Wert über `git stash push` auf genau die drei geänderten Dateien, danach
  `stash pop` und erneut gemessen). `coverage_start` der Unit war
  "n.a. (Verhaltens-Fix, kein Coverage-Ziel)" — der Zuwachs ist Nebeneffekt des
  neuen `ErrInFlight`-Zweigs plus des zweiten Tests.
- mutations-probe: zwei Proben, beide rot.
  (1) `if inserted` → `if true` (Erkennung ausgehebelt, alter Zustand) →
  `…_OneWinnerOneInFlight` fällt mit "got 2 winners / 0 in-flight",
  `…_SecondIsInFlight` mit "expected ErrInFlight, got <nil>" — exakt der Produktionsfehler.
  (2) `if inserted` → `if !inserted` (Bedingung invertiert) → vier Tests rot, darunter
  `TestPostgresReserve_FreshKey` und `…_Replay_ReturnsCachedRecord` mit
  "idempotency key request in flight" — der Gewinner-Pfad ist also ebenfalls abgedeckt.
  Zurückgedreht, `git diff --stat postgres_repository.go` zeigt wieder die
  ursprünglichen 17 Insertions / 4 Deletions, Paket grün.
- verify vorgänger: sauber. `28837c84` (Iteration 56,
  fix-work-task-custom-field-definitions-fk) gegen alle acht Fehlerklassen geprüft
  (`git show --stat` + Volltextdiff von Code und Migration) — Migration 000320 hängt nur
  eine FK auf einer bestehenden Tabelle um (kein neues `tenant_id`/RLS-Thema), `.up`/`.down`
  beide gefüllt und spiegelbildlich, Karteileichen-Löschung im Kopf begründet, JOIN in
  `GetCustomFieldValues` im selben Commit mitgezogen. Kein neuer Handler, kein direkter
  Service-Aufruf am gRPC-Client vorbei, kein Stub, kein `.proto`, kein neuer oder ersetzter
  `RequirePermission`-Guard, keine neue Route, keine Wire-Shape-Änderung.
- neue-units: keine
- offen:
  - **Verhaltensänderung mit Produktionswirkung, bewusst so gebaut:** ein Request, dessen
    erster Versuch abgestürzt ist, BEVOR `Complete` lief, bekommt beim Retry mit demselben
    Idempotency-Key jetzt 409 statt durchzulaufen — bis der Key nach `expires_at` weg ist.
    Das ist genau die dokumentierte Semantik ("same hash, no response yet → ErrInFlight")
    und die Voraussetzung dafür, dass der Schlüssel Doppelbuchungen überhaupt verhindert;
    Luke sollte es trotzdem gesehen haben, weil vorher nichts davon greifen konnte.
    Der Client bekommt `Retry-After: 2` mit.
  - `-race` konnte lokal nicht laufen: `go test -race` verlangt cgo, und auf dieser Maschine
    ist kein `gcc` im PATH ("C compiler \"gcc\" not found"). Die `done_when`-Zeile der Unit
    fordert `-race` — gelaufen ist der identische Testsatz ohne Detektor (inklusive der
    nebenläufigen Zwei-Goroutinen-Reservierung, die real gegen die DB grün ist). CI fährt
    `-race`; wenn dort etwas auffällt, dann in `internal/idempotency`.

## Iteration 58 — feat-dsar-search-contact-advisory-protocol-module — done — 2026-08-22 08:08
- commit: 8b490d2d
- gebaut: `advisoryProtocolModules` in `internal/security/gdpr/dsar_search.go` schließt die
  letzte der 14 FK-auf-`contacts`-Tabellen ohne DSAR-Modul. Statt den ~50-Spalten-Scan aus
  `advisoryprotocol/postgres_repository.go` zu duplizieren, ruft die Funktion
  `advisoryprotocol.NewPostgresRepository(pool).ListByContact` auf und formatiert jedes
  zurückgegebene `Protocol` in ein eigenes `DSARModule` — ein Modul pro Beratungsprotokoll
  ("Beratungsprotokoll N (Datum, Status)"), chronologisch aufsteigend (die Repository-Query
  liefert neueste zuerst, hier umgedreht). Jedes Modul listet alle ~40 fachlichen Felder aus
  §1-§8 als Feld/Wert-Zeilen: Arrays (`known_asset_classes`, `investment_purpose`,
  `warnings_given`) als kommagetrennter Klartext statt Postgres-Array-Syntax, das
  JSONB-`products`-Feld als lesbare "Name (SRI X, empfohlen/nicht empfohlen)"-Liste statt
  Rohformat, Geldfelder mit "EUR"-Suffix (gleiche `lean:`-Begründung wie in `financeModule`).
  `internal_notes` ist bewusst ausgeschlossen — gleiche Regel wie die interne Helpdesk-Notiz
  in `helpdeskMessagesModule`: Arbeitsmaterial des Beraters über den Kunden, keine an ihn
  gerichtete Kommunikation oder Tatsachenangabe. In `SearchByQuery` zwischen `documentsModule`
  und `consentModule` eingehängt.
  `.golangci.yml` um einen `misspell`-Ignore-Eintrag für "Immobilien" ergänzt (False Positive
  gegen das englische "immobile" — gleiches Muster wie der bestehende Produktion-Eintrag).
  Zwei neue DB-Tests: `TestSearchByQuery_ContactAdvisoryProtocols_Integration` (zwei Protokolle
  je Kontakt — ein Entwurf, ein abgeschlossenes —, chronologische Reihenfolge, Array-/JSONB-
  Klartext, Ausschluss von `internal_notes` inklusive Volltext-Leck-Check über alle Feldwerte,
  Tenant-Isolation) und `…_NoneIsNoModule_Integration` (Kontakt ohne Protokoll trägt kein
  Beratungsprotokoll-Modul, analog zum nil-Modul-Vertrag der übrigen Module in dieser Datei).
- gate: build ok (`./internal/security/... ./internal/crm/... ./internal/gateway/...
  ./cmd/gateway/...`) | vet ok | lint ok (0 issues, `./internal/security/...`, inkl. des
  behobenen misspell-Fundes) | test ok (`./internal/security/gdpr/` vollständig gruen,
  0 SKIP von den insgesamt gelaufenen Tests, gegen `kmuhub_app`;
  `./internal/crm/advisoryprotocol/...` unverändert gruen) | migration n.a. (keine Tabelle
  angefasst, nur eine bestehende Repository-Query wiederverwendet) | rls-smoke n.a. (keine
  Tabelle/Policy geändert — RLS auf `advisory_protocols` besteht bereits seit Migration 000218
  und wird von den neuen Tests über echte Tenant-Isolation mitgeprüft, nicht neu aufgesetzt)
  | TestOpenAPIRouteDrift PASS (836 Routen gegen 838 dokumentierte Pfade, unverändert — keine
  neue Route in dieser Unit)
- coverage: `internal/security/gdpr` 69,3 % -> 69,6 % (selbst gemessen: Vorher via
  `git stash push` auf genau die drei geänderten Dateien, danach `stash pop` und erneut
  gemessen). `coverage_start` der Unit war als Platzhalter "Wert zur Laufzeit messen"
  eingetragen — das ist die Laufzeitmessung.
- mutations-probe: `advisoryProtocolRecords` um eine zusätzliche Zeile
  `fieldValueRecord("Interne Notizen", p.InternalNotes)` ergänzt (den dokumentierten
  Ausschluss von `internal_notes` aufgehoben) → `TestSearchByQuery_ContactAdvisoryProtocols_Integration`
  wird mit drei Assertion-Fehlschlägen rot ("internal_notes must never appear as a labeled
  field" plus zwei Volltext-Leck-Treffer auf die genau dafür geseedeten internen Notizen).
  Zurückgedreht, `git diff --stat dsar_search.go` zeigt wieder exakt die ursprünglichen 159
  Insertions, Paket erneut vollständig gruen.
- verify vorgänger: sauber. `a12c6a02` (Iteration 57, fix-idempotency-reserve-inflight-race)
  gegen alle acht Fehlerklassen geprüft (`git show --stat` + Volltextdiff) — reine
  Postgres-Repository- und Aufrufer-Änderung (Middleware/Webhook unverändert gelassen), keine
  neue Route, kein neuer Handler, kein direkter Service-Aufruf am gRPC-Client vorbei, kein
  Stub, kein `.proto`, kein neuer/ersetzter `RequirePermission`-Guard, keine neue Tabelle,
  keine Wire-Shape-Änderung. `(xmax = 0) AS inserted` ist eine reine RETURNING-Erweiterung auf
  eine System-Spalte, kein Schema-Wechsel.
- neue-units: keine
- offen:
  - Die Modulbenennung "Beratungsprotokoll N (Datum, Status)" ist pro Kontakt fortlaufend
    nummeriert (1, 2, ...), nicht global eindeutig — bei zwei gematchten Kontakten mit
    Historie sieht die UI pro Person wieder bei 1 an. Das ist beabsichtigt (Nummerierung ist
    Session-relativ zur Konsultationshistorie dieses einen Kontakts), aber falls die
    Auskunfts-Oberfläche Module tenant-/personenübergreifend flach auflistet statt gruppiert
    nach Person, sollte das gegengeprüft werden — hier nicht sichtbar, weil DSARPerson.Modules
    bereits pro Person geschachtelt ist.
  - `advisoryprotocol.PostgresRepository` wird hier zum ersten Mal aus `internal/security/gdpr`
    heraus importiert. Kein Zyklus (advisoryprotocol importiert gdpr nicht), aber es ist die
    erste modul-übergreifende Repository-Wiederverwendung in `dsar_search.go` statt einer
    eigenen Inline-Query — bewusst so gewählt wegen der Feldzahl, aber ein Muster, das die
    übrigen Module in dieser Datei nicht verwenden. Falls das nicht gewünscht ist, wäre die
    Alternative eine Inline-Query analog zu den anderen Modulen, mit dupliziertem Scan.

## Iteration 59 — feat-dsar-search-user-hr-employee-module — done — 2026-08-22 08:16
- commit: b90c5328
- gebaut: Drei neue DSAR-Module in `internal/security/gdpr/dsar_search.go`, eingehängt in
  `matchUsers` direkt nach den Notification-Modulen: `hrProfileModule` ("Personalprofil",
  ein Feld/Wert-Modul aus `hr_employee_profiles`, LEFT JOIN auf `users` für den Vorgesetzten-
  Namen — Abteilung, Position, Vertragsart, Arbeitstage/Woche, Urlaubsanspruch, Eintrittsdatum,
  Notfallkontakt, Adresse, Stundenlohn, Status, Austrittsdaten, `is_minor`),
  `hrEmployeeDocumentsModule` ("Personaldokumente (Metadaten)", reine Metadaten aus
  `hr_employee_documents` + Kategorie-Name via JOIN — kein Pfad, keine URL, kein Inhalt, gleiche
  Regel wie A5) und `hrProfileChangeRequestsModule` ("Änderungsanträge (Profil)", Bereich/Feld/
  Alter Wert/Neuer Wert/Status/Grund/Zeitstempel aus `hr_profile_change_requests`).
  `is_minor` wird als normales Boolean-Feld offengelegt, ohne Sonderroute — begründet im
  Docstring von `hrProfileModule`: kein Vertreter-/Vormund-Kontakt im Schema, an den eine
  Auskunft stattdessen gehen könnte, und Art. 15 unterscheidet den AuskunftsUMFANG nicht nach
  Alter (die DSGVO-Sonderbehandlung Minderjähriger betrifft Einwilligung/Information bei der
  Erhebung, nicht den späteren Export).
  SPLIT gegenüber der ursprünglichen Unit (die sechs Tabellen wollte): Urlaub
  (`hr_leave_requests`, `hr_leave_balances`) und Arbeitszeit (`hr_work_time_entries`) sind als
  neue Unit `feat-dsar-search-user-hr-time-leave-module` ans Backlog-Ende gewandert — die Notiz
  der Original-Unit erlaubte diesen Split ausdrücklich ("Stammdaten+Dokumente vs. Zeit+Urlaub").
  ECHTER FUND, in dieser Unit direkt mitbehoben statt als Fix-Unit angelegt (dieselbe neue Query
  betrifft es unmittelbar): `hr_employee_documents` trägt als einzige der sechs HR-Tabellen eine
  rollenbasierte RLS-Policy (`hr_document_access`, Migration 000127/000128) statt der reinen
  `tenant_isolation`, die alle übrigen von `dsar_search.go` gelesenen Tabellen haben — sie
  verlangt `current_user_has_hr_role('admin'|'hr_admin')` oder Manager-/Self-Access je nach
  Dokumentkategorie-Sichtbarkeit. `DSARSearch` (`security_grpc.go:579`) läuft im normalen
  Anfrage-Kontext des aufrufenden Nutzers, nicht unter `is_system_context()` — ein Security-/
  Compliance-Admin ohne HR-Rolle hätte bei jeder Auskunftsanfrage lautlos null Personaldokumente
  gesehen, obwohl sie existieren (mit `hr_only`-Sichtbarkeit sogar garantiert null). Fix:
  `sysctx.With(ctx)` um genau diese eine Query, exakt das Muster, das `RunScheduledRetention`
  bereits im selben Package (`retention_scheduler.go:46`) für denselben Fall nutzt — die
  tenant_id- und employee_id-Prädikate in der WHERE-Klausel bleiben die eigentliche Zugriffs-
  eingrenzung, sysctx entfernt nur das zusätzliche Rollen-Gate, das für eine gesetzliche
  Offenlegungspflicht nicht greifen darf.
  Sechs neue DB-Tests: `TestSearchByQuery_UserHRProfile_Integration`,
  `..._UserHRProfile_NoneIsNoModule_Integration` (Nutzer ohne HR-Profil trägt kein Modul),
  `..._UserHRDocuments_Integration` (seedet ein Dokument mit `hr_only`-Sichtbarkeit und beweist
  über den normalen, rollenlosen Tenant-Kontext, dass es trotzdem erscheint — das ist der
  Beleg für den RLS-Fix; zusätzlich ein Grep über alle Feldwerte, dass kein "/" auftaucht),
  `..._UserHRProfileChangeRequests_Integration`, alle mit Tenant-Isolation.
- gate: build ok (`./internal/security/... ./internal/gateway/... ./cmd/gateway/...`) | vet ok
  (`./internal/security/...`, testkompiliert) | lint ok (0 issues, `./internal/security/...`)
  | test ok (`./internal/security/gdpr/` vollständig grün, 0 SKIP gegen `kmuhub_app`;
  `./internal/security/...` gesamt ebenfalls grün) | migration n.a. (keine neue Tabelle, keine
  Policy geändert — nur eine bestehende Policy im Code umgangen, nicht in der DB verändert)
  | rls-smoke n.a. (kein Schema/Policy-Wechsel; die RLS-Wirkung selbst ist über die
  `UserHRDocuments`-Integrationstests bereits als echter Tenant-/Rollen-Test erbracht)
  | TestOpenAPIRouteDrift nicht gelaufen — keine Route in dieser Unit angefasst, daher nicht
  Pflicht (Regel greift "sobald du eine Route angefasst hast")
- coverage: `internal/security/gdpr` 69,6 % -> 70,1 % (selbst gemessen: `git stash push` auf
  genau die zwei geänderten Dateien, danach `stash pop`). `coverage_start` der Unit war als
  Platzhalter "seit Iteration 40 mehrfach verändert" eingetragen — das ist die Laufzeitmessung.
- mutations-probe: `sysctx.With(ctx)` in `hrEmployeeDocumentsModule` durch `ctx` ersetzt (den
  RLS-Fix aufgehoben) → `TestSearchByQuery_UserHRDocuments_Integration` wird rot mit
  `module "Personaldokumente (Metadaten)" not found, have [Benutzerkonto]` — das Modul
  verschwindet komplett, nicht nur einzelne Felder, weil die Query unter der role-gated Policy
  null Zeilen liefert. Zurückgedreht, `git diff --stat dsar_search.go` zeigt wieder exakt die
  ursprünglichen 224 Insertions, Paket erneut vollständig grün.
- verify vorgänger: sauber. `8b490d2d` (Iteration 58, feat-dsar-search-contact-advisory-protocol-
  module) gegen alle acht Fehlerklassen geprüft (`git show` Volltextdiff) — reine
  Repository-Wiederverwendung (`advisoryprotocol.NewPostgresRepository(pool).ListByContact`),
  kein direkter Service-Aufruf am gRPC-Client vorbei (dies ist ohnehin kein Gateway-Handler,
  sondern derselbe interne DSAR-Suchpfad wie alle anderen Module in der Datei), kein Stub, kein
  `.proto`, kein neuer/ersetzter `RequirePermission`-Guard, keine neue Tabelle, keine Route,
  keine Wire-Shape-Änderung (reine Go-Struct-Rückgabe innerhalb des Pakets).
- neue-units: feat-dsar-search-user-hr-time-leave-module (Urlaub + Arbeitszeit, Restarbeit aus
  dem Split, ans Backlog-Ende gehängt, inkl. Hinweis, das RLS-Rollen-Gate-Muster aus dieser
  Iteration bei den drei verbleibenden Tabellen gegenzuprüfen)
- offen:
  - Der RLS-Fund ist wahrscheinlich kein Einzelfall: `hr_document_access` war laut seiner
    eigenen Migrationsnotiz (000127) eine bewusste Verschärfung gegenüber einer vorherigen
    reinen `tenant_isolation`-Policy auf genau dieser einen Tabelle. Andere rollen- oder
    sichtbarkeitsgated Policies (falls es sie z. B. bei Finance- oder Vertrags-Tabellen gibt)
    könnten denselben stillen Leerlauf für jede künftige DSAR-Erweiterung erzeugen. Wäre ein
    guter Kandidat für einen der Block-C-Scans (Muster: "role-gated RLS policy joined from
    dsar_search.go under a non-system caller context").
  - `hrProfileModule` liest `manager_user_id` nur zur Namensauflösung, prüft aber nicht, ob der
    Vorgesetzte selbst gematchter DSAR-Subjekt ist — das ist beabsichtigt (die Auskunft betrifft
    nur den einen Nutzer), aber falls die Oberfläche Vorgesetzten-Namen anklickbar macht, wäre
    das ein eigener Datenpfad, kein Fund hier.

## Iteration 60 — feat-dsar-search-user-account-security-history-module — done — 2026-08-22 08:27
- commit: 8f4c7982
- gebaut: zwei neue DSAR-Module in `dsar_search.go`/`matchUsers` — `userSessionsModule`
  ("Sitzungsverlauf": Gerät, Typ, IP via `host(ip_address)`, Standort, letzte Aktivität, Status
  per SQL-CASE gegen `refresh_tokens` — Aktiv/Widerrufen/Abgelaufen/Beendet) und
  `accountSecurityModule` ("Kontosicherheit": 2FA-Status aus `users.two_factor_enabled(_at)`,
  Passwortänderungen-Anzahl+Datum aus `password_history`, Wiederherstellungscodes gesamt/genutzt
  aus `recovery_codes` — ausschließlich Metadaten, nie ein Hash/Secret/Code im Klartext).
  Abweichung von der Backlog-Praemisse: `two_factor_policy` ist eine ROLLENBASIERTE
  Tenant-Einstellung (unique auf tenant_id+role_name), keine personenbezogene Angabe des
  gematchten Nutzers — bewusst NICHT eingebaut, stattdessen die beiden `users`-Spalten
  verwendet, die den tatsächlichen 2FA-Status des Kontos tragen. `accountSecurityModule` ist
  das einzige Modul in dieser Datei, das NIE nil zurückgibt (zwei_factor_enabled ist NOT NULL —
  "nie geändert/nie aktiviert" ist der Auskunftsinhalt, keine Datenabwesenheit), analog zum
  immer vorhandenen Basismodul "Benutzerkonto"; dafür musste die bestehende
  `TestSearchByQuery_MatchesUsers_Integration` von `require.Len(..., 1)` auf `2` angepasst
  werden (bewusste, im Diff sichtbare Anpassung eines Bestandstests, kein Kollateralschaden).
- gebaut (Tests): `TestSearchByQuery_UserSessions_Integration` (4 Sitzungen decken alle vier
  Status ab, plus eine fünfte Sitzung eines zweiten Benutzers beweist "nur eigene Sitzungen"),
  `TestSearchByQuery_UserAccountSecurity_Integration` (2 Passwortwechsel, 3 Recovery-Codes davon
  1 genutzt, 2FA aktiv — plus Grep gegen alle Feldwerte auf "secret"/"recovery-hash"),
  `TestSearchByQuery_UserAccountSecurity_DefaultsWhenNoHistory_Integration` (frisches Konto ohne
  jede Historie, Modul erscheint trotzdem mit "Nein"/"0"/leeren Zeitangaben). Alle drei inkl.
  Tenant-Isolation.
- gate: build ok (`./internal/security/... ./internal/gateway/... ./cmd/gateway/...`) | vet ok
  (`./internal/security/...`) | lint ok (0 issues, `./internal/security/...`) | test ok
  (`./internal/security/gdpr/` 121 PASS / 0 SKIP / 0 FAIL gegen `kmuhub_app`; gesamtes
  `./internal/security/...` ebenfalls grün) | migration n.a. (keine neue Tabelle/Policy, reine
  Lesequeries auf bestehende Tabellen) | rls-smoke n.a. (kein Schema/Policy-Wechsel) |
  TestOpenAPIRouteDrift nicht gelaufen — keine Route in dieser Unit angefasst, daher nicht
  Pflicht.
- coverage: `internal/security/gdpr` 70,1 % -> 70,3 % (selbst gemessen: `git stash push` auf
  genau die zwei geänderten Dateien, `go test -coverprofile` davor/danach, `stash pop`).
  `coverage_start` der Unit war bewusst als Platzhalter "zur Laufzeit messen" eingetragen — das
  ist die Laufzeitmessung.
- mutations-probe: `WHEN rt.revoked THEN 'Widerrufen'` im SQL-CASE von `userSessionsModule` zu
  `WHEN false THEN 'Widerrufen'` verfälscht → `TestSearchByQuery_UserSessions_Integration` wird
  rot (die Sitzung mit widerrufenem Token zeigt fälschlich "Aktiv" statt "Widerrufen").
  Zurückgedreht, `git diff --stat dsar_search.go` zeigt wieder ausschließlich die
  ursprünglichen 133 Insertions, Paket erneut vollständig grün (121 PASS).
- verify vorgänger: sauber. `b90c5328` (Iteration 59, feat-dsar-search-user-hr-employee-module)
  gegen alle acht Fehlerklassen geprüft (`git show --stat` + Volltextdiff von
  `dsar_search.go`) — reine interne DSAR-Suchpfad-Erweiterung (kein Gateway-Handler, daher keine
  gRPC-Umgehung möglich), kein Stub, kein `.proto`, kein neuer/ersetzter
  `RequirePermission`-Guard, keine neue Tabelle, keine Route, keine Wire-Shape-Änderung
  (reine Go-Struct-Rückgabe innerhalb des Pakets). Der dort dokumentierte RLS-Fund
  (`sysctx.With` um `hr_employee_documents`) ist durch die eigene Mutations-Probe der
  Vorgänger-Iteration belegt, nicht nur behauptet.
- neue-units: keine
- offen:
  - `two_factor_policy` (rollenbasierte 2FA-Erzwingung) ist tenantweite Konfiguration und damit
    außerhalb des Art.-15-Umfangs für den einzelnen Nutzer — falls die Oberfläche das je anders
    einordnet, ist das eine Produktentscheidung, kein technischer Nachtrag.
  - `user_sessions` hat kein eigenes `expires_at`; der Status stützt sich auf das verknüpfte
    `refresh_tokens.expires_at`/`revoked` und behandelt eine Sitzung ohne (mehr existierenden)
    Token als "Beendet". Sollte sich das Rotationsschema je ändern (z. B. `refresh_token_id`
    bleibt nach Rotation erhalten statt auf eine neue Zeile zu zeigen), müsste dieses Mapping
    neu geprüft werden — aktuell zeigt `auth/postgres_repository.go` klar `ON DELETE SET NULL`.

## Iteration 61 — feat-dsar-search-user-fuhrpark-driver-module — done — 2026-08-22 08:35
- commit: 940a28c6
- gebaut: zwei neue DSAR-Module in `dsar_search.go`/`matchUsers` — `driverLicensesModule`
  ("Führerscheinkontrolle": Klassen, Ablaufdatum, geprüft am, nächste Prüfung fällig, Notizen aus
  `driver_licenses`, gefiltert auf `driver_id = subject`) und `vehicleBookingsModule`
  ("Fahrzeugbuchungen": Fahrzeug — Marke/Modell/Kennzeichen aus `vehicles` —, Beginn, Ende, Zweck,
  Status aus `vehicle_bookings`, gefiltert auf `user_id = subject OR created_by = subject`).
  `vehicle_bookings.user_id` (Fahrer/Nutznießer) und `created_by` (Buchender) können
  unterschiedliche Personen sein — jede Zeile trägt deshalb ein explizites Rollenfeld
  ("Fahrer"/"Buchender"/"Fahrer und Buchender") statt beide Spalten stillschweigend
  gleichzusetzen.
- gebaut (Tests): `TestSearchByQuery_UserDriverLicenses_Integration` (eigene vs. fremde
  Führerscheinkontrolle, Tenant-Isolation), `TestSearchByQuery_UserVehicleBookings_Integration`
  (drei Buchungen — selbst gefahren+gebucht, für Kollegen gebucht, weder-noch — beweist Rollenlabel
  und dass die unbeteiligte Buchung fehlt, plus Tenant-Isolation). Zeitstempel werden wie beim
  Sessions-Modul aus Iteration 60 aus der DB zurückgelesen statt aus dem Go-seitigen `now`
  formatiert (Zeitzonen-Rundreise über die DB-Session).
- gate: build ok (`./internal/security/... ./internal/gateway/... ./cmd/gateway/...`) | vet ok
  (`./internal/security/...`) | lint ok (0 issues, `./internal/security/...`) | test ok
  (`./internal/security/gdpr/` 123 PASS / 0 SKIP / 0 FAIL gegen `kmuhub_app`; gesamtes
  `./internal/security/...` ebenfalls grün) | migration n.a. (keine neue Tabelle/Policy, reine
  Lesequeries auf `driver_licenses`/`vehicle_bookings`/`vehicles`) | rls-smoke n.a.
  (kein Schema/Policy-Wechsel) | TestOpenAPIRouteDrift nicht gelaufen — keine Route in dieser Unit
  angefasst, daher nicht Pflicht.
- coverage: `internal/security/gdpr` 70,3 % -> 70,6 % (selbst gemessen: `git stash push` auf genau
  die zwei geänderten Dateien, `go test -coverprofile` davor/danach, `stash pop`).
- mutations-probe: `case reservedForSubject && bookedBySubject: return "Fahrer und Buchender"` zu
  `return "Fahrer"` verfälscht → `TestSearchByQuery_UserVehicleBookings_Integration` wird rot
  (die Buchung, bei der der Nutzer Fahrer UND Buchender ist, zeigt fälschlich nur "Fahrer").
  Zurückgedreht, `git diff --stat dsar_search.go` zeigt wieder ausschließlich die ursprünglichen
  125 Insertions, Paket erneut vollständig grün.
- verify vorgänger: sauber. `8f4c7982` (Iteration 60, feat-dsar-search-user-account-security-history-module)
  gegen alle acht Fehlerklassen geprüft (`git show --stat` + Volltextdiff von `dsar_search.go`) —
  reine interne DSAR-Suchpfad-Erweiterung, kein Gateway-Handler (keine gRPC-Umgehung möglich), kein
  Stub, kein `.proto`, kein neuer/ersetzter `RequirePermission`-Guard, keine neue Tabelle, keine
  Route, keine Wire-Shape-Änderung. Tenant-Filter (`tenant_id = $1`) und Secret-Ausschluss
  (Passwort-Hashes, Recovery-Code-Hashes) durch die eigenen Tests der Vorgänger-Iteration belegt.
- neue-units: keine
- offen:
  - keine

## Iteration 62 — feat-dsar-search-invitation-history-module — done — 2026-08-22 08:41
- commit: 3c92ac90
- gebaut: neues Modul `invitationHistoryModule` in `dsar_search.go`/`matchUsers` —
  "Einladungshistorie" (Name bei Einladung, Rolle, Eingeladen von, Eingeladen am, Angenommen am),
  gematcht ueber `invitations.email = users.email` (beide Spalten seit Migration 000148 auf
  lowercase normiert, exakter Vergleich reicht). Query filtert zusaetzlich
  `accepted_at IS NOT NULL` — nur Einladungen, die tatsaechlich zu diesem Konto gefuehrt haben,
  landen im Export. `role` ist die Legacy-Preset-Spalte (Display-only seit Migration 000280,
  Kommentar in `models/invitation.go:15`); `token_hash` wird bewusst nicht selektiert.
  Inviter-Name kommt aus einem JOIN auf `users.created_by` (FK ist `NOT NULL ... ON DELETE
  CASCADE`, die Zeile existiert also immer, wenn die Einladung noch existiert).
- gebaut (Test): `TestSearchByQuery_UserInvitationHistory_Integration` — ein Benutzer mit einer
  angenommenen Einladung (Rolle "admin", Inviter "Petra Leitwolf") plus einer zweiten, weiterhin
  offenen Einladung an dieselbe Adresse (simuliert eine versehentliche Zweit-Einladung eines
  bereits onboardeten Kontos); der Test belegt, dass nur die angenommene Zeile im Modul landet
  und die offene fehlt. Inklusive Tenant-Isolation.
- gate: build ok (`./internal/security/... ./internal/gateway/... ./cmd/gateway/...`) | vet ok
  (`./internal/security/...`) | lint ok (0 issues, `./internal/security/...`) | test ok
  (`./internal/security/gdpr/` 124 PASS / 0 SKIP / 0 FAIL gegen `kmuhub_app`) | migration n.a.
  (keine neue Tabelle/Spalte, reine Lesequery auf bestehende `invitations`/`users`) | rls-smoke
  n.a. (kein Schema/Policy-Wechsel) | TestOpenAPIRouteDrift nicht gelaufen — keine Route in
  dieser Unit angefasst, daher nicht Pflicht.
- coverage: `internal/security/gdpr` 70,6 % -> 70,7 % (selbst gemessen: `git stash push` auf
  genau die zwei geaenderten Dateien, `go test -coverprofile` davor/danach, `stash pop`).
- mutations-probe: `AND i.accepted_at IS NOT NULL` aus dem WHERE entfernt → die zweite, offene
  Einladung derselben Adresse wird mitgelesen, `accepted_at` ist dort NULL und der Scan bricht
  hart ("cannot scan NULL into *time.Time") → Test wird rot. Zurueckgedreht, `git diff --stat
  dsar_search.go` zeigt wieder ausschliesslich die urspruenglichen 62 Insertions, Paket erneut
  vollstaendig gruen (124 PASS).
- verify vorgaenger: sauber. `940a28c6` (Iteration 61, feat-dsar-search-user-fuhrpark-driver-module)
  gegen alle acht Fehlerklassen geprueft (`git show --stat` + Volltextdiff von `dsar_search.go`) —
  reine interne DSAR-Suchpfad-Erweiterung, kein Gateway-Handler (keine gRPC-Umgehung moeglich),
  kein Stub, kein `.proto`, kein neuer/ersetzter `RequirePermission`-Guard, keine neue Tabelle,
  keine Route, keine Wire-Shape-Aenderung. Tenant-Filter und die Rollenlabel-Logik
  (`vehicleBookingRoleLabel`) durch die eigenen Tests der Vorgaenger-Iteration belegt, inklusive
  der dort dokumentierten Mutations-Probe.
- neue-units: keine
- offen:
  - Der im Scope explizit benannte Nebenbefund bleibt ungeloest: eine NIE angenommene Einladung
    (`accepted_at IS NULL`) hat keinen `users`- oder `contacts`-Eintrag, an den `SearchByQuery`
    sie haengen koennte — die eingeladene Person ist fuer eine Auskunftsanfrage aktuell nicht
    auffindbar, obwohl ihre E-Mail/Name/Rolle in `invitations` steht. Das waere eine dritte
    Subjekt-Matching-Quelle neben Kontakten und Benutzern und damit ein groesserer Eingriff als
    diese Unit — Architekturfrage fuer Luke, keine neue Unit angelegt (Vorgabe in den notes/
    done_when dieser Unit).

## Iteration 63 — fix-work-erasure-time-entries-retention-conflict — done — 2026-08-22 08:48
- commit: 24ff9444
- gebaut: `WorkErasureHandler.ExecuteErasure` (`erasure.go`) loeschte bislang JEDEN Zeiteintrag
  eines Nutzers sofort und unbedingt (`DELETE FROM time_entries WHERE user_id = $1`), mit dem
  Kommentar "personal data, no business retention need" — ein Widerspruch zur eigenen Feststellung
  aus Iteration 40, dass Zeiteintraege arbeitsrechtlich relevant sind (ArbZG-nahe Fristen). Fix:
  neue Konstante `timeEntryRetentionDays = 730` (Sec. 16 Abs. 2 ArbZG, zwei Jahre — konservativ auf
  die Task-Zeiterfassung angewandt, da im Code sonst keine ArbZG-spezifische Spalte/Konstante fuer
  diese Tabelle existiert; das ist eine begruendete Annahme, kein aus einer bestehenden
  Retention-Tabelle gelesener Wert). Das DELETE bekommt eine `AND started_at < <cutoff>`-Klausel:
  nur Eintraege AELTER als die Frist werden hart geloescht, juengere ueberleben unveraendert.
  Kein separates Anonymisieren des `user_id`-FK noetig: `AuthErasureHandler` (registriert VOR
  `WorkErasureHandler` in `cmd/auth/main.go:109-112`) anonymisiert bereits die `users`-Zeile selbst
  (Name/E-Mail geloescht) — derselbe Mechanismus, auf den sich `tasks.created_by` und
  `contacts/companies.created_by` (CRMErasureHandler) schon verlassen. Ein junger Zeiteintrag zeigt
  also weiterhin per FK auf einen bereits anonymisierten Nutzer, nicht auf eine identifizierbare
  Person.
- gebaut (Test): `erasure_work_test.go` — bestehender Zeiteintrag-Seed in zwei aufgeteilt: einer
  mit `started_at` 2020 (jenseits der Frist, muss weiterhin hart geloescht werden — belegt den
  Altfall) und einer mit `started_at` 2026-08-11 (innerhalb der Frist, darf NICHT geloescht werden
  — neuer Pfad). Die alte Assertion "time entries must be deleted, not anonymized" war mit dem Fix
  falsch geworden (der bestehende Seed lag innerhalb der Frist) und wurde durch zwei Assertions
  ersetzt: Alteintrag = 0 Zeilen nach Erasure, junger Eintrag = weiterhin 1 Zeile.
- gate: build ok (`./internal/security/... ./internal/gateway/... ./internal/work/... ./cmd/gateway/...`)
  | vet ok (`./internal/security/... ./internal/work/...`) | lint ok (0 issues,
  `./internal/security/gdpr/...`) | test ok (`./internal/security/gdpr/` 124 PASS / 0 SKIP / 0 FAIL
  gegen `kmuhub_app`; `./internal/work/...` alle 17 Unterpakete ok, seriell mit `-p 1` gefahren
  nachdem ein paralleler Lauf zusammen mit gdpr an der Postgres-`max_connections`-Grenze der
  lokalen Docker-Instanz scheiterte — reines Verbindungslimit, keine Regression, isoliert erneut
  gruen) | migration n.a. (keine Schema-Aenderung) | rls-smoke n.a. (kein Schema/Policy-Wechsel) |
  TestOpenAPIRouteDrift nicht gelaufen — keine Route in dieser Unit angefasst, daher nicht Pflicht.
- coverage: `internal/security/gdpr` 70,7 % -> 70,7 % (selbst gemessen: `git stash` auf genau die
  zwei geaenderten Dateien, `go test -coverprofile` davor/danach, `stash pop`; die Aenderung ist zu
  klein, um die Paket-Prozentzahl sichtbar zu verschieben, aber die zwei neuen Testpfade sind
  ueber die Assertions belegt, nicht ueber die Prozentzahl).
- mutations-probe: `started_at < $2` zu `started_at > $2` verfaelscht (loescht junge statt alte
  Eintraege) → `TestWorkErasureHandler_ExecuteErasure_Integration` wird rot (beide neuen
  Assertions schlagen fehl: Alteintrag bleibt bestehen, junger Eintrag wird geloescht).
  Zurueckgedreht, `git diff --stat erasure.go` zeigt wieder ausschliesslich die urspruenglichen
  19 Insertions/2 Deletions, Paket erneut vollstaendig gruen (124 PASS).
- verify vorgaenger: sauber. `3c92ac90` (Iteration 62, feat-dsar-search-invitation-history-module)
  gegen alle acht Fehlerklassen geprueft (`git show --stat` + Volltextdiff von `dsar_search.go`) —
  reine interne DSAR-Suchpfad-Erweiterung, kein Gateway-Handler (keine gRPC-Umgehung moeglich),
  kein Stub, kein `.proto`, kein neuer/ersetzter `RequirePermission`-Guard, keine neue Tabelle,
  keine Route, keine Wire-Shape-Aenderung. Tenant-Filter (`i.tenant_id = $1`) und der bewusste
  Ausschluss offener Einladungen (`accepted_at IS NOT NULL`) durch die eigenen Tests der
  Vorgaenger-Iteration belegt, inklusive der dort dokumentierten Mutations-Probe.
- neue-units: keine
- offen:
  - Die 730-Tage-Frist ist eine begruendete, konservative Annahme (Sec. 16 Abs. 2 ArbZG), keine
    aus einer bestehenden Konfigurationsquelle gelesene Zahl — falls es fuer die Task-Zeiterfassung
    (im Unterschied zu `hr_work_time_entries`, dem eigentlichen ArbZG-Clock-in/out) eine andere
    verbindliche Frist gibt, gehoert das von Luke geprueft.

## Iteration 64 — fix-auth-erasure-missing-security-tables — done — 2026-08-22 08:55
- commit: f0694012
- gebaut: `AuthErasureHandler` (`erasure.go`) loeschte bislang `user_sessions`,
  `refresh_tokens`, `recovery_codes`, `password_history` und anonymisierte `users` — zwei
  sicherheitsrelevante Tabellen mit CASCADE-FK auf `users(id)` blieben unberuehrt:
  `password_reset_tokens` (offene Passwort-Reset-Tokens) und `app_specific_passwords`
  (selbst erzeugte CalDAV/CardDAV-Basic-Auth-Zugangsdaten). Beide CASCADE-FKs feuern nie, weil
  ExecuteErasure niemals die `users`-Zeile loescht, nur anonymisiert (UPDATE, kein DELETE) — die
  Zeilen blieben also fuer immer stehen. Fix: zwei neue DELETE-Schritte (4a, 4b) in
  `ExecuteErasure`, VOR dem Anonymisieren des User-Profils, sowie zwei neue Summanden in der
  `PreviewErasure`-Zaehlquery, damit Preview und Execute deckungsgleich bleiben. Tabellennamen und
  Spaltennamen gegen die Migrationen bestaetigt (`000134_create_password_reset_tokens.up.sql`,
  `000049_create_app_passwords.up.sql` + `tenant_id`-Nachtrag in
  `000114_option_b_phase2_settings_preferences.up.sql`), nicht aus pg_constraint geraten. Der
  Doc-Kommentar des Handlers ist um beide Tabellen ergaenzt.
  Sicherheitsbegruendung (gehoert in den Commit): ein stehen gebliebenes Reset-Token ist ein
  Angriffsvektor — wer den alten Link haelt, koennte nach der Loeschung ein neues Passwort auf
  dem bereits anonymisierten Account setzen. Ein stehen gebliebenes App-Passwort authentifiziert
  unabhaengig vom Hauptpasswort/2FA weiter und ueberlebt die Erasure komplett unbemerkt.
- gebaut (Test): `handlers_test.go` — `TestAuthErasureHandler_ExecuteErasure_Integration` um zwei
  Seeds erweitert (offener Reset-Token, aktives App-Passwort) und zwei neue Assertions nach dem
  ExecuteErasure-Lauf: beide Zeilen sind per COUNT(*) verschwunden. Keine neue Testdatei, bestehender
  Auth-Erasure-Test war die richtige Stelle (Vorlage: gleiche Datei, gleicher Test, bereits mit
  Session-Seed).
- gate: build ok (`./internal/security/... ./internal/gateway/... ./cmd/gateway/...`) | vet ok
  (`./internal/security/... ./internal/gateway/...`) | lint ok (0 issues,
  `./internal/security/gdpr/...`) | test ok (`./internal/security/gdpr/` 124 PASS / 0 SKIP / 0 FAIL
  gegen `kmuhub_app`; `./internal/security/...` alle 7 Unterpakete ok) | migration n.a. (keine
  Schema-Aenderung, beide Tabellen existieren bereits) | rls-smoke n.a. (kein Schema/Policy-Wechsel)
  | TestOpenAPIRouteDrift nicht gelaufen — keine Route in dieser Unit angefasst, daher nicht
  Pflicht.
- coverage: `internal/security/gdpr` 70,7 % -> 70,8 % (selbst gemessen: `go test -coverprofile`
  vor und nach der Aenderung, `go tool cover -func` Gesamtzeile).
- mutations-probe: `DELETE FROM password_reset_tokens WHERE user_id = $1` um `AND false` ergaenzt
  → `TestAuthErasureHandler_ExecuteErasure_Integration` wird rot ("password reset tokens must be
  deleted on erasure", expected 0 actual 1). Zurueckgedreht, `git diff --stat erasure.go` zeigt
  wieder ausschliesslich die urspruenglichen 24 Insertions/2 Deletions, Paket erneut vollstaendig
  gruen (124 PASS).
- verify vorgaenger: sauber. `24ff9444` (Iteration 63, fix-work-erasure-time-entries-retention-conflict)
  gegen alle acht Fehlerklassen geprueft (`git show --stat` + Volltextdiff von `erasure.go` und
  `erasure_work_test.go`) — reine interne Erasure-Logik-Aenderung (Retention-Fenster fuer
  Zeiteintraege), kein Gateway-Handler (keine gRPC-Umgehung moeglich), kein Stub, kein `.proto`,
  kein neuer/ersetzter `RequirePermission`-Guard, keine neue Tabelle, keine Route, keine
  Wire-Shape-Aenderung. Mutations-Probe der Vorgaenger-Iteration im Journal dokumentiert und
  plausibel (Vergleichsoperator umgedreht, Test wird rot).
- neue-units: keine
- offen:
  - Die Sibling-Units in diesem Backlog-Abschnitt (`fix-calendar-erasure-incomplete-and-doc-mismatch`,
    `fix-crm-erasure-contacts-companies-preview-execute-mismatch`, `fix-work-erasure-missing-project-membership`,
    `fix-chat-erasure-missing-bookmarks-mentions`) beschreiben denselben Fehlerklassen-Typ
    (Preview/Execute-Luecke, fehlende CASCADE-Tabellen) fuer andere Handler — unveraendert offen,
    keine Ueberschneidung mit dieser Unit.

## Iteration 65 — fix-calendar-erasure-incomplete-and-doc-mismatch — done — 2026-08-22 09:20
- commit: fdfa18cc
- gebaut: `CalendarErasureHandler`s Doc-Kommentar behauptete "Deletes personal calendars and owned
  events", der Code tat keins von beidem: `calendars` (owner_id) wurde nirgends angefasst,
  `calendar_events` (created_by) wurde in `ExecuteErasure` nur GEZAEHLT statt geloescht/
  anonymisiert, und `calendar_members` sowie `caldav_push_subscriptions` fehlten komplett — beide
  CASCADE auf `users(id)`, feuern aber nie, weil ExecuteErasure die `users`-Zeile nur anonymisiert
  (UPDATE), nie loescht (DELETE). Fix in `erasure.go`: (1) eigene Personenkalender werden geloescht
  (cascadet Events/Attendees/Exceptions/Reminders via `calendar_id ON DELETE CASCADE`) — AUSSER
  eine `booking_pages`-Zeile haengt daran, dann bleibt der Kalender stehen, um die
  `public_bookings`-Kundendaten (Name/E-Mail/Telefon fremder Personen) nicht kollateral zu
  vernichten; (2) verbleibende, vom Nutzer organisierte Termine (auf fremden/retinierten
  Kalendern) werden anonymisiert (Titel/Beschreibung/Ort geloescht) statt nur gezaehlt — der
  Termin bleibt fuer andere Teilnehmer bestehen; (3) `calendar_members` (Mitgliedschaft in
  fremden Kalendern) wird geloescht; (4) `caldav_push_subscriptions` wird geloescht; (5) als
  direkter Fund im selben Handler zusaetzlich `event_categories` (gleiche CASCADE-nie-Problematik)
  geloescht, wodurch `calendar_events.category_id` via bestehendes `ON DELETE SET NULL`
  automatisch bereinigt wird. `affected`-Zaehlung fuer `calendar_events`/`event_attendees`/
  `calendar_members` wird VOR der Kalender-Loeschung vorab gezaehlt, weil die Kaskade sie sonst
  unter der Hand verschwinden liesse, bevor die eigentlichen DELETE/UPDATE-Statements laufen —
  sonst waere Preview/Execute wieder auseinandergelaufen.
- gebaut (Test): neue Datei `erasure_calendar_test.go` —
  `TestCalendarErasureHandler_ExecuteErasure_Integration` deckt alle sieben betroffenen Tabellen
  ab: personal ohne Buchungsseite (geloescht), personal MIT Buchungsseite (bleibt stehen, Termin
  anonymisiert), Termin auf fremdem Team-Kalender (anonymisiert, Kalender bleibt), Termin mit zwei
  Teilnehmern (nur die eigene Teilnahme verschwindet, der Termin und die Teilnahme des Kollegen
  bleiben unangetastet), Kalendermitgliedschaft, Push-Subscription, Event-Kategorie (inkl. Beleg,
  dass die Referenz auf NULL faellt statt zu haengen). Preview- und Execute-Zahl werden
  gegeneinander geprueft (beide 8). `TestCalendarErasureHandler_ExecuteErasure_DeadPool` als
  Pendant zu den anderen Handlern ergaenzt.
- gebaut (Folgeaenderung an bestehendem Test): `erasure_idempotency_test.go` —
  `TestCalendarErasureHandler_ExecuteErasure_SecondRunStillReportsCreatedEvents` pinnte bisher das
  ALTE, fehlerhafte Verhalten (zweiter Lauf meldet faelschlich einen weiteren Treffer). Durch
  diesen Fix ist genau das seeded Szenario (Termin auf der EIGENEN, jetzt geloeschten
  Personenkalender-Zeile) nun echt doppellauf-fest: Kalender und Termin sind nach dem ersten Lauf
  bereits weg, der zweite Lauf matcht nichts mehr (n1: 3->4, n2: 1->0). Kommentar und Assertions
  entsprechend aktualisiert; der Rest-Fall (Termin auf fremdem/retiniertem Kalender bleibt bei
  jedem Lauf zaehlbar, weil nur anonymisiert, nie geloescht) ist weiterhin offen und bleibt Teil
  von `fix-erasure-handlers-not-idempotent-on-second-run` — deren Scope-Text im Backlog ist um
  diese Praezisierung ergaenzt, damit die spaetere Iteration nicht am jetzt gruenen Test stolpert.
- gate: build ok (`./internal/security/... ./internal/gateway/... ./cmd/gateway/...`) | vet ok
  (`./internal/security/... ./internal/gateway/...`) | lint ok (0 issues, `./internal/security/...`)
  | test ok (`./internal/security/gdpr/` alle Tests inkl. neue und geaenderte gruen, 0 SKIP gegen
  `kmuhub_app`; `./internal/security/...` alle 7 Unterpakete ok) | migration n.a. (keine
  Schema-Aenderung, alle betroffenen Tabellen bestehen bereits) | rls-smoke n.a. (kein
  Schema/Policy-Wechsel) | gofmt: `erasure.go` und beide Testdateien nachtraeglich formatiert
  (Edit-Tool hatte nicht-kanonische Einrueckung erzeugt) | TestOpenAPIRouteDrift nicht gelaufen —
  keine Route in dieser Unit angefasst, daher nicht Pflicht.
- coverage: `internal/security/gdpr` 70,8 % -> 70,9 % (selbst gemessen: `git stash push -u` auf
  genau die drei geaenderten/neuen Dateien, `go test -coverprofile` davor/danach, `stash pop`).
- mutations-probe: `DELETE FROM calendar_members WHERE user_id = $1` um `AND false` ergaenzt →
  `TestCalendarErasureHandler_ExecuteErasure_Integration` wird rot ("calendar membership must be
  removed on erasure", expected 0 actual 1). Zurueckgedreht, `git diff --stat erasure.go` zeigt
  wieder ausschliesslich 91 Insertions/9 Deletions, Paket erneut vollstaendig gruen.
- verify vorgaenger: sauber. `f0694012` (Iteration 64, fix-auth-erasure-missing-security-tables)
  gegen alle acht Fehlerklassen geprueft (`git show --stat` + Volltextdiff von `erasure.go` und
  `handlers_test.go`) — reine interne Erasure-Logik-Aenderung (zwei zusaetzliche DELETE-Schritte
  in AuthErasureHandler), kein Gateway-Handler (keine gRPC-Umgehung moeglich), kein Stub, kein
  `.proto`, kein neuer/ersetzter `RequirePermission`-Guard, keine neue Tabelle (beide Tabellen
  bestanden bereits), keine Route, keine Wire-Shape-Aenderung. Mutations-Probe der
  Vorgaenger-Iteration im Journal dokumentiert und plausibel (Guard-Bedingung negiert, Test wird
  rot).
- neue-units: fix-booking-page-orphaned-after-owner-erasure (Fund waehrend dieser Iteration: die
  bewusste Ausnahme "Personenkalender mit Buchungsseite wird nicht geloescht" verhindert
  Datenverlust bei `public_bookings`, laesst aber offen, was mit der oeffentlich weiter aktiven
  Buchungsseite eines geloeschten/anonymisierten Mitarbeiters geschehen soll — Produktentscheidung
  fuer Luke, nicht eigenmaechtig im Code entschieden)
- offen:
  - `fix-booking-page-orphaned-after-owner-erasure` (neue Unit) braucht Lukes Entscheidung
    (deaktivieren/umhaengen/nur warnen), bevor daran gebaut werden kann.
  - `fix-erasure-handlers-not-idempotent-on-second-run` (bereits im Backlog) ist im Scope fuer den
    Calendar-Anteil enger geworden (nur noch Termine auf fremden/retinierten Kalendern), Scope-Text
    im Backlog entsprechend ergaenzt — kein neuer Fund, nur Praezisierung fuer die naechste
    Iteration, die diese Unit zieht.

## Iteration 66 — fix-crm-erasure-contacts-companies-preview-execute-mismatch — done — 2026-08-22 09:15
- commit: f0d108c7
- gebaut: `CRMErasureHandler.PreviewErasure` zaehlte `contacts` und `companies` (created_by) als
  "betroffene" Datensaetze; `ExecuteErasure` liess beide Tabellen unveraendert und zaehlte sie
  nur erneut in `affected` — Preview versprach eine Wirkung, die nie eintrat. Entscheidung (Option
  a aus den Backlog-Notizen): contacts/companies werden aus BEIDEN Zaehlungen entfernt, weil sich
  an ihnen inhaltlich nichts aendert (created_by bleibt via anonymisiertem User-Sentinel gueltige
  FK) — ehrliche Zahl statt vorgetaeuschter Wirkung. Preview und Execute zaehlen jetzt
  ausschliesslich `activities`. Struct-Doc-Kommentar auf `CRMErasureHandler` entsprechend
  neu geschrieben (beschreibt jetzt Rueckbehaltung von contacts/companies/Pipeline explizit als
  bewusste Business-Record-Entscheidung, nicht laenger "Anonymizes contacts and activities").
- gebaut (Tests): `erasure_crm_chat_test.go` — `TestCRMErasureHandler_PreviewErasure_Integration`
  (3 -> 1, Kontakt/Firma bleiben geseedet, um zu belegen dass sie NICHT mehr zaehlen),
  `TestCRMErasureHandler_ExecuteErasure_Integration` (4 -> 2), `TestCRMErasureHandler_ExecuteErasure_IgnoresAction`
  (2 -> 1) angepasst; Retain-Checks fuer contacts/companies bleiben unveraendert (weiterhin nicht
  geloescht). `erasure_idempotency_test.go` —
  `TestCRMErasureHandler_ExecuteErasure_SecondRunRecountsCreatedByForever` (n1 2 -> 1, Kommentar
  von "activity+contact" auf "activity" praezisiert) — die verbleibende Doppellauf-Luecke betrifft
  jetzt ausschliesslich `activities.created_by` und bleibt Teil von
  `fix-erasure-handlers-not-idempotent-on-second-run`.
- gate: build ok (`./internal/security/... ./internal/gateway/... ./cmd/gateway/...`) | vet ok
  (`./internal/security/... ./internal/gateway/...`) | lint ok (0 issues,
  `./internal/security/...`) | test ok (`./internal/security/gdpr/` 126 Subtests PASS, 0 SKIP
  gegen `kmuhub_app`; `./internal/security/...` alle 7 Unterpakete ok) | migration n.a. (keine
  Schema-Aenderung) | rls-smoke n.a. (kein Schema/Policy-Wechsel) | TestOpenAPIRouteDrift nicht
  gelaufen — keine Route in dieser Unit angefasst, daher nicht Pflicht.
- coverage: `internal/security/gdpr` 70,9 % -> 70,8 % (selbst gemessen: `git stash push -u` auf
  genau die drei geaenderten Dateien, `go test -coverprofile` davor/danach, `stash pop`; leichter
  Ruecktritt ist erwartbar, da entfernter Zaehlcode auch Coverage-Zeilen entfernt hat — reine
  Bugfix-Unit, kein Coverage-Ziel).
- mutations-probe: die entfernte Contact-Zaehlung probeweise wieder in `ExecuteErasure` eingefuegt
  (`affected += contactCount`) → drei Tests werden rot
  (`TestCRMErasureHandler_ExecuteErasure_IgnoresAction`,
  `TestCRMErasureHandler_ExecuteErasure_SecondRunRecountsCreatedByForever`,
  `TestCRMErasureHandler_ExecuteErasure_Integration`, jeweils erwartete vs. tatsaechliche Zahl um
  1 daneben). Zurueckgedreht, `git diff --stat erasure.go` zeigt wieder ausschliesslich 11
  Insertions/14 Deletions, Paket erneut vollstaendig gruen.
- verify vorgaenger: sauber. `fdfa18cc` (Iteration 65, fix-calendar-erasure-incomplete-and-doc-mismatch)
  gegen alle acht Fehlerklassen geprueft (`git show --stat` + Volltextdiff von `erasure.go`) —
  reine interne Erasure-Logik-Aenderung (Kalender/Mitgliedschaften/Push-Subscriptions loeschen),
  kein Gateway-Handler (keine gRPC-Umgehung moeglich), kein Stub, kein `.proto`, kein
  neuer/ersetzter `RequirePermission`-Guard, keine neue Tabelle (alle betroffenen Tabellen
  bestanden bereits), keine Route, keine Wire-Shape-Aenderung. Keine `tenant_id`-Filterung in den
  Queries dieser Datei, aber konsistent mit allen Schwester-Handlern in derselben Datei (bereits
  bestehendes Muster, RLS-getragen, nicht neu in diesem Commit). Mutations-Probe der
  Vorgaenger-Iteration im Journal dokumentiert und plausibel.
- neue-units: keine
- offen:
  - `fix-erasure-handlers-not-idempotent-on-second-run` (bereits im Backlog) ist im Scope fuer den
    CRM-Anteil unveraendert (weiterhin `activities.created_by` matcht bei jedem Lauf erneut) —
    kein neuer Fund, nur Bestaetigung nach dieser Iteration.

## Iteration 67 — fix-work-erasure-missing-project-membership — done — 2026-08-22 09:26
- commit: d72c0760
- gebaut: `WorkErasureHandler` erfasste `project_members` (user_id, CASCADE auf users(id)) weder
  in `PreviewErasure` noch in `ExecuteErasure` — die CASCADE-FK feuert nie, weil
  `AuthErasureHandler` den Nutzer anonymisiert statt loescht (dasselbe Muster wie
  `channel_memberships` im `ChatErasureHandler`, das war die Vorlage). `PreviewErasure` zaehlt
  jetzt zusaetzlich `project_members WHERE user_id = $1`; `ExecuteErasure` loescht dieselben
  Zeilen in derselben Transaktion und zaehlt sie in `affected` mit. Struct-Doc-Kommentar auf
  `WorkErasureHandler` ergaenzt ("und Projektmitgliedschaften").
- gebaut (Tests): `erasure_work_test.go` — neuer Helper `seedProjectMember` (composite PK
  project_id/user_id, kein `id`, `testutil.SeedRow` scheidet aus wie schon bei
  `seedChannelMembership`); `TestWorkErasureHandler_ExecuteErasure_Integration` seedet jetzt ein
  Projekt mit zwei Mitgliedschaften (Subjekt + Kollege), erwartete `affected`-Summe von 5 auf 6
  angepasst, zwei neue Assertions: Subjekt-Mitgliedschaft geloescht (0), Kollegen-Mitgliedschaft
  bleibt (1).
- gate: build ok (`./internal/security/... ./internal/gateway/... ./cmd/gateway/...`) | vet ok
  (`./internal/security/... ./internal/gateway/...`) | lint ok (0 issues,
  `./internal/security/... ./internal/gateway/...`) | test ok (`./internal/security/gdpr/` 126
  Subtests PASS, 0 SKIP gegen `kmuhub_app`; `./internal/security/...` alle 7 Unterpakete ok) |
  migration n.a. (keine Schema-Aenderung) | rls-smoke n.a. (kein Schema/Policy-Wechsel) |
  TestOpenAPIRouteDrift nicht gelaufen — keine Route in dieser Unit angefasst, daher nicht
  Pflicht.
- coverage: `internal/security/gdpr` 70,8 % -> 70,9 % (selbst gemessen: `git stash push -u` auf
  genau die zwei geaenderten Dateien, `go test -coverprofile` davor/danach, `stash pop`).
- mutations-probe: die neue `DELETE FROM project_members`-Zeile probeweise mit `AND false`
  entwertet → `TestWorkErasureHandler_ExecuteErasure_Integration` wird rot (affected 6 vs. 5,
  Subjekt-Mitgliedschaft 0 vs. 1 erwartet). Zurueckgedreht, `git diff --stat erasure.go` zeigt
  wieder ausschliesslich 16 Insertions/2 Deletions, Paket erneut vollstaendig gruen.
- verify vorgaenger: sauber. `f0d108c7` (Iteration 66,
  fix-crm-erasure-contacts-companies-preview-execute-mismatch) gegen alle acht Fehlerklassen
  geprueft (`git show --stat` + Volltextdiff von `erasure.go`) — reine interne
  Erasure-Logik-Aenderung (Preview/Execute-Zaehlung fuer contacts/companies entfernt), kein
  Gateway-Handler, kein Stub, kein `.proto`, kein neuer/ersetzter `RequirePermission`-Guard,
  keine neue Tabelle, keine Route, keine Wire-Shape-Aenderung.
- neue-units: keine
- offen: keine

## Iteration 68 — fix-chat-erasure-missing-bookmarks-mentions — done — 2026-08-22 09:26
- commit: 80794966
- gebaut: `ChatErasureHandler` erfasste `message_bookmarks` und `message_mentions` (beide
  `user_id`, CASCADE auf `users(id)`) weder in `PreviewErasure` noch in `ExecuteErasure` — beide
  CASCADE-FKs feuern nie, weil `AuthErasureHandler` den Nutzer anonymisiert statt loescht
  (dasselbe Muster wie `channel_memberships`/`project_members`, das war die Vorlage).
  `PreviewErasure` zaehlt jetzt zusaetzlich beide Tabellen; `ExecuteErasure` loescht
  `message_bookmarks WHERE user_id = $1` (reine Eigendaten) und `message_mentions
  WHERE user_id = $1` (nur die Zeile, in der der geloeschte Nutzer die ERWAEHNTE Person ist —
  die Nachricht selbst bleibt unberuehrt, auch wenn sie von jemand anderem stammt). Struct-Doc-
  Kommentar auf `ChatErasureHandler` ergaenzt.
- gebaut (Tests): neue Datei `erasure_chat_bookmarks_mentions_test.go` mit zwei Helfern
  (`seedMessageBookmark`, `seedMessageMention` — beide Tabellen composite PK ohne `id`,
  `testutil.SeedRow` scheidet aus). `TestChatErasureHandler_ExecuteErasure_BookmarksAndMentions`
  seedet eine fremde Nachricht (Kollege ist Autor), darin ein Bookmark und eine Mention des
  Subjekts sowie eine zweite Mention des Kollegen selbst; belegt `affected == 2`, dass die
  fremde Nachricht unveraendert bleibt, das Bookmark des Subjekts weg ist und nur seine
  Mention geloescht wird, waehrend die Mention des Kollegen in derselben Nachricht ueberlebt.
- gate: build ok (`./internal/security/... ./internal/gateway/... ./cmd/gateway/...`) | vet ok
  (`./internal/security/... ./internal/gateway/...`) | lint ok (0 issues,
  `./internal/security/... ./internal/gateway/...`) | test ok (`./internal/security/gdpr/` alle
  Subtests PASS, 0 SKIP gegen `kmuhub_app`; `./internal/security/...` alle 7 Unterpakete ok) |
  migration n.a. (keine Schema-Aenderung, beide Tabellen bestanden bereits inkl. RLS) |
  rls-smoke n.a. (kein Schema/Policy-Wechsel) | TestOpenAPIRouteDrift nicht gelaufen — keine
  Route in dieser Unit angefasst, daher nicht Pflicht.
- coverage: `internal/security/gdpr` 70,9 % -> 70,9 % (selbst gemessen per `git stash push -u`
  auf genau die zwei geaenderten/neuen Dateien, `go test -coverprofile` davor/danach, `stash
  pop`; Rundungsgleichstand, das Paket ist gross genug, dass ein paar neue Zeilen den
  gerundeten Wert nicht bewegen — kein Messfehler).
- mutations-probe: die `message_mentions`-DELETE-Zeile probeweise mit `AND false` entwertet →
  `TestChatErasureHandler_ExecuteErasure_BookmarksAndMentions` wird rot (affected 1 vs. 2,
  Subjekt-Mention 1 vs. 0 erwartet). Zurueckgedreht, `git diff --stat erasure.go` zeigt wieder
  ausschliesslich 25 Insertions/2 Deletions, Paket erneut vollstaendig gruen.
- verify vorgaenger: sauber. `d72c0760` (Iteration 67, fix-work-erasure-missing-project-membership)
  gegen alle acht Fehlerklassen geprueft (`git show --stat` + Volltextdiff von `erasure.go`) —
  reine interne Erasure-Logik-Aenderung (project_members loeschen), kein Gateway-Handler, kein
  Stub, kein `.proto`, kein neuer/ersetzter `RequirePermission`-Guard, keine neue Tabelle, keine
  Route, keine Wire-Shape-Aenderung. Mutations-Probe der Vorgaenger-Iteration im Journal
  dokumentiert und plausibel.
- neue-units: keine
- offen: keine

## Iteration 69 — feat-erasure-handler-user-settings-preferences — done — 2026-08-22 09:33
- commit: ed2e41ec
- gebaut: achter `ErasureHandler` `SettingsErasureHandler` (ModuleName "settings") fuer vier
  Tabellen mit CASCADE-FK auf `users(id)`, die zu keiner der sieben bestehenden Domaenen
  gehoeren: `user_settings` (composite PK tenant_id/user_id/module_id/key), 
  `user_dashboard_layouts`, `user_project_preferences` (composite PK user_id/project_id) und
  `saved_filters` (Filter des Nutzers ueber `created_by`, keine Business-Records ueber Dritte —
  deshalb geloescht statt behalten, wie in den notes begruendet). Vorlage war exakt
  `NotificationErasureHandler` (kleinster bestehender Handler, reine DELETE-Kette). In
  `cmd/auth/main.go` als achte `RegisterErasureHandler`-Zeile registriert.
- gebaut (Tests): neue Datei `erasure_settings_test.go` mit drei Tests: `ModuleName`,
  `ExecuteErasure_Integration` (seedet je einen Datensatz pro Tabelle plus einen Kollegen-Filter,
  belegt `affected == preview.RecordCount == 4` und dass der Kollegen-Filter ueberlebt),
  `ExecuteErasure_DeadPool` (Fehlerpfad bei geschlossenem Pool).
- gate: build ok (`./internal/security/... ./cmd/auth/...`) | vet ok (`./internal/security/...`)
  | lint ok (0 issues, `./internal/security/...`) | test ok (`./internal/security/gdpr/` alle
  Subtests PASS, 0 SKIP gegen `kmuhub_app`; `./internal/security/...` alle 7 Unterpakete ok) |
  migration n.a. (keine Schema-Aenderung, alle vier Tabellen bestanden bereits inkl.
  `tenant_id NOT NULL` + RLS, geprueft gegen `000114_option_b_phase2_settings_preferences` und
  `000122_rls_phase2_long_tail`) | rls-smoke n.a. (kein Schema/Policy-Wechsel) |
  TestOpenAPIRouteDrift nicht gelaufen — keine Route angefasst (nur `cmd/auth/main.go`
  Registrierung), daher nicht Pflicht.
- coverage: `internal/security/gdpr` 70,9 % -> 71,0 % (selbst gemessen: `git stash push -u` auf
  genau `erasure.go` + `erasure_settings_test.go`, `go test -coverprofile` davor/danach,
  `stash pop`).
- mutations-probe: die neue `DELETE FROM saved_filters`-Zeile probeweise mit `AND false`
  entwertet → `TestSettingsErasureHandler_ExecuteErasure_Integration` wird rot (affected 3 vs. 4,
  `saved_filters`-Zeile des Subjekts ueberlebt). Zurueckgedreht, `git diff --stat erasure.go`
  zeigt wieder ausschliesslich 92 Insertions/0 Deletions, Paket erneut vollstaendig gruen.
- verify vorgaenger: sauber. `80794966` (Iteration 68, fix-chat-erasure-missing-bookmarks-mentions)
  gegen alle acht Fehlerklassen geprueft (`git show --stat` + Volltextdiff von `erasure.go`) —
  reine interne Erasure-Logik-Aenderung (message_bookmarks/message_mentions loeschen), kein
  Gateway-Handler, kein Stub, kein `.proto`, kein neuer/ersetzter `RequirePermission`-Guard,
  keine neue Tabelle, keine Route, keine Wire-Shape-Aenderung.
- neue-units: keine
- offen: keine

## Iteration 70 — fix-company-delete-merged-into-fk-crash — done — 2026-08-22 09:35
- commit: 097d3737
- gebaut: `companies.merged_into_id` trug seit Migration 000059 `ON DELETE NO ACTION`
  (confdeltype `a`, verifiziert per `pg_constraint`). `company/service.go:290-318` prueft vor dem
  Loeschen nur `HasContacts`, nicht ob die Company Ziel eines abgeschlossenen Merges ist —
  identisches Muster wie der bereits gefixte Contact-Fall (`fix-contact-delete-merged-into-no-action-unchecked`,
  Iteration dieses Laufs, Commit `6c00c33f`). Migration `000321_companies_merged_into_set_null`
  stellt `companies_merged_into_id_fkey` auf `ON DELETE SET NULL` um (DROP + ADD CONSTRAINT,
  gleiche Form wie 000318 fuer contacts). Root-Cause-Entscheidung wie beim Contact-Vorbild:
  Grep bestaetigt, dass `merged_into_id` in `company/` nur zum Ausfiltern bereits gemergter Zeilen
  aus der Duplicate-Suche (`... AND merged_into_id IS NULL`) und zum Markieren des Duplikats
  (`MergeInto`) verwendet wird — kein Lesepfad loest den Pointer auf, um ein geloeschtes Duplikat
  auf seinen Primary zurueckzufuehren. SET NULL ist damit gefahrlos und konsistent mit dem
  Contact-Fix.
  Neuer DB-Test `TestRepository_Delete_MergedPrimaryCompany_DB` in
  `postgres_repository_db_test.go` (Vorlage: `TestRepository_Delete_MergedPrimaryContact_DB`):
  merged Company A in Company B, loescht B (den Primary), belegt Erfolg und dass A.merged_into_id
  danach NULL ist.
- gate: build ok (`./internal/crm/... ./cmd/crm/...`) | vet ok (`./internal/crm/...`) | lint ok
  (0 issues, `./internal/crm/...`) | test ok (`./internal/crm/company/` gruen; volle
  `./internal/crm/...`-Suite serialisiert mit `-p 1` gruen — der erste parallele Lauf riss mit
  "too many clients already" ab, das ist eine lokale `max_connections`-Grenze der Docker-DB unter
  Volllast, keine Regression: derselbe Testlauf ohne `-p 1` schlaegt schon auf dem unveraenderten
  main-Stand fehl) | migration ok (up/down beide angewendet, siehe Mutations-Probe) | rls-smoke
  n.a. (Constraint-Aenderung an bestehender Tabelle, keine neue Tabelle/Policy, kein
  tenant-gescopter SELECT angefasst) | TestOpenAPIRouteDrift nicht gelaufen — keine Route
  angefasst, daher nicht Pflicht.
- coverage: `internal/crm/company` 79,7 % -> 79,7 % (selbst gemessen per `git stash push -u` auf
  die geaenderte Testdatei, `go test -coverprofile` davor/danach, `stash pop`; Rundungsgleichstand
  — die Unit aendert reines DB-Verhalten, kein neuer Go-Code-Pfad in `service.go`/`postgres_repository.go`).
- mutations-probe: `migrate down 1` auf die neue Migration (Constraint zurueck auf NO ACTION) →
  `TestRepository_Delete_MergedPrimaryCompany_DB` wird rot mit der erwarteten rohen
  FK-Verletzung (SQLSTATE 23503) genau an der Delete-Zeile. `migrate up` zurueckgedreht, Test
  wieder gruen, `git diff --stat` zeigt nur die vorgesehenen Dateien.
- verify vorgaenger: sauber. `ed2e41ec` (Iteration 69, feat-erasure-handler-user-settings-preferences)
  gegen alle acht Fehlerklassen geprueft (`git show --stat` + Volltextdiff von `erasure.go` und
  `cmd/auth/main.go`) — neuer Erasure-Handler nach exaktem Muster der sieben bestehenden, achte
  `RegisterErasureHandler`-Zeile, kein Gateway-Handler, kein Stub, kein `.proto`, kein neuer/ersetzter
  `RequirePermission`-Guard, keine neue Tabelle (alle vier Tabellen bestanden bereits inkl.
  `tenant_id NOT NULL` + RLS), keine Route, keine Wire-Shape-Aenderung. Tenant-Scoping laeuft ueber
  die RLS-Pool-Hook (`app.tenant_id` aus dem Kontext gestempelt, `service.go:136`), kein manueller
  Tenant-Filter in Query noetig — konsistent mit allen sieben Geschwister-Handlern, kein Befund.
- neue-units: keine
- offen: keine

## Iteration 71 — fix-channel-delete-call-sessions-fk-crash — done — 2026-08-22 09:47
- commit: 78339951
- gebaut: `call_sessions.channel_id` traegt seit Migration 000036 `ON DELETE NO ACTION`
  (nullable FK, kein spaeteres ALTER gefunden — Grep ueber alle Migrationen bestaetigt).
  `channel/service.go` (`Delete`) prueft vor dem Loeschen nur DM-Status und Owner-Rolle, nicht ob
  noch archivierte `call_sessions` am Channel haengen — der rohe FK-Fehler faellt in
  `mapChatError`s `default:`-Zweig und kommt als 500 "internal server error" heraus statt eines
  verstaendlichen 409. Verifiziert, dass der Fall in der Praxis erreichbar bleibt: der
  Kopf-Kommentar in `retention_dialer_chat.go:8-11` (aus der bereits gebauten Unit
  `feat-retention-worker-handler-dialer-chat`) haelt fest, dass `call_sessions` (Video/Voice,
  Migration 000036 — nicht zu verwechseln mit `dialer_call_sessions`) von KEINEM
  Retention-Handler erreicht wird; die Zeilen bleiben also unbegrenzt liegen.
  Fix ist ein Anwendungs-Guard, keine Migration: neue Methode `HasCallSessions` im
  `channel.Repository`-Interface (`SELECT EXISTS(... WHERE channel_id = $1)`,
  `postgres_repository.go`), in `Service.Delete` nach der Owner-Pruefung und vor `repo.Delete`
  aufgerufen. Neuer Fehler `ErrChannelHasCallSessions`, gemappt auf `codes.FailedPrecondition`
  in `chat_grpc.go` — exakt das bestehende Muster von `ErrCannotDeleteDM` zwei Zeilen darueber
  (FailedPrecondition -> HTTP 409 via `grpcStatusToHTTP`, per `helpers_test.go:23` belegt).
  Zwei Repository-Mocks mussten die neue Interface-Methode nachziehen:
  `MockRepository` (`channel/service_test.go`, neues Feld `callSessions map[uuid.UUID]bool`) und
  `stubChannelRepo` (`server/chat_grpc_channels_messages_test.go`, immer `false, nil` — dieser
  Stub deckt Channel-Delete-Tests nicht ab).
  Migration bewusst NICHT gewaehlt (anders als bei A2/company/contact-Fixes): eine Video-Call-
  Aufzeichnung auf SET NULL zu stellen wuerde die Zuordnung Anruf->Channel stillschweigend
  verlieren, obwohl der Anruf als archivierter Datensatz weiterhin existiert — das ist ein
  anderer Charakter als der reine Selbstbezug bei `merged_into_id`.
- gate: build ok (`./internal/chat/... ./internal/server/... ./internal/gateway/... ./cmd/...`)
  | vet ok (`./internal/chat/... ./internal/server/...`) | lint ok (0 issues, dieselben Pfade)
  | test ok (`./internal/chat/channel/` einzeln gruen inkl. neuer Tests; volle
  `./internal/chat/...`-Suite mit `-p 1` gruen — paralleler Lauf riss wie in Iteration 70 mit
  "too many clients already" ab, lokale `max_connections`-Grenze der Docker-DB, keine Regression)
  | `./internal/server/` gruen | `./internal/gateway/` gruen (TestOpenAPIRouteDrift enthalten,
  keine Route angefasst) | migration n.a. (kein Schema-Wechsel) | rls-smoke n.a. (kein
  Tabellen-/Policy-Wechsel, reine Anwendungslogik + neue Repository-Methode)
- coverage: `internal/chat/channel` 82,4 % -> 82,5 % (selbst gemessen: `git stash push -u` auf
  alle sechs geaenderten `channel/`-Dateien inkl. beider Testdateien, `go test -coverprofile`
  davor/danach, `stash pop`).
- mutations-probe: die neue `if hasCallSessions { return ErrChannelHasCallSessions }`-Zeile
  probeweise mit `if false && hasCallSessions` entwertet → `TestService_Delete/blocked_by_archived_call_sessions`
  wird rot (bekommt `channel not found` statt des erwarteten Fehlers, weil das Delete durchlief).
  Zurueckgedreht, `git diff --stat service.go` zeigt wieder ausschliesslich 8 Insertions/0
  Deletions, `./internal/chat/channel/...` erneut vollstaendig gruen.
- verify vorgaenger: sauber. `097d3737` (Iteration 70, fix-company-delete-merged-into-fk-crash)
  gegen alle acht Fehlerklassen geprueft (`git show --stat` + Volltext der Migration
  `000321_companies_merged_into_set_null`) — reine FK-Constraint-Migration (DROP+ADD wie beim
  bereits gefixten Contact-Vorbild 000318), kein Gateway-Handler, kein Stub, kein `.proto`, kein
  neuer/ersetzter `RequirePermission`-Guard, keine neue Tabelle, keine Route, keine
  Wire-Shape-Aenderung.
- neue-units: keine
- offen: keine

## Iteration 72 — fix-document-folder-delete-nonempty-fk-crash — done — 2026-08-22 09:48
- commit: 7346120a
- gebaut: `document_files.folder_id` traegt `NOT NULL REFERENCES document_folders(id)` ohne
  `ON DELETE`-Klausel (= NO ACTION, `migrations/000043_create_document_tables.up.sql:26`).
  `document_folders.parent_id` traegt dagegen `ON DELETE CASCADE`
  (`000043...up.sql:10`) — ein Ordner mit Unterordnern liesse sich also loeschen und wuerde
  seinen kompletten Unterbaum lautlos mitnehmen, bis in der Tiefe ein Unterordner mit Dateien
  die FK-Verletzung ausloest (Postgres prueft alle Kaskadeneffekte in derselben Transaktion).
  `folder/service.go` (`Delete`) pruefte vorher nur `IsSystem`. Fix nutzt zwei bereits
  vorhandene, bereits getestete Repository-Methoden (`CountFiles`, `GetChildren`) statt neuer
  SQL — beide waren im Interface und im Postgres-Repository schon vorhanden, nur an keiner
  Delete-Stelle verdrahtet. Blockt jetzt VOR dem Delete: direkte Dateien ODER direkte
  Unterordner -> `ErrFolderNotEmpty` -> `FailedPrecondition` (409) statt 500. Bewusst keine
  "hochziehen"/"rekursiv loeschen"-Variante gebaut (Notes nannten das als Alternative): ein
  harter 409 ist die einzige Variante ohne Produktentscheidung, und Rekursiv-Loeschen eines
  Unterordner-Baums mit Dateien waere exakt das stillschweigende Verhalten, das der Bug-Report
  kritisiert.
  Mapping in `document_grpc.go:mapDocumentError` als neuer Case neben
  `ErrCircularParent` (identisches Muster: FailedPrecondition). Tests: zwei neue Service-Tests
  mit dem bestehenden `MockRepository` (Dateien vorhanden / Unterordner vorhanden, beide
  erwarten `ErrFolderNotEmpty` und dass der Ordner nicht geloescht wurde) plus ein neuer Fall
  in der Tabelle `TestMapDocumentError_AllSentinels`.
- gate: build ok (`./internal/document/... ./internal/server/... ./internal/gateway/... ./cmd/...`)
  | vet ok (`./internal/document/... ./internal/server/...`) | lint ok (0 issues, dieselben
  Pfade) | test ok (`./internal/document/folder/` einzeln gruen inkl. beider neuer Tests; volle
  `./internal/document/...`-Suite mit `-p 1` gruen; `./internal/server/` gruen) | migration n.a.
  (kein Schema-Wechsel, reiner Anwendungs-Guard) | rls-smoke n.a. (keine Tabelle/Policy
  angefasst) | `./internal/gateway/` (TestOpenAPIRouteDrift) NICHT gelaufen — keine Route
  angefasst, daher laut Anleitung nicht Pflicht.
- coverage: `internal/document/folder` 82,2 % -> 82,1 % (selbst gemessen: `git stash push -u`
  auf `service.go`/`errors.go`/`service_test.go`, `go test -coverprofile` davor/danach,
  `stash pop`; minimal negative Verschiebung reiner Rundungseffekt — neue Zeilen sind durch die
  zwei neuen Tests vollstaendig abgedeckt, der Nenner an Gesamtstatements waechst mit).
- mutations-probe: `if fileCount > 0` probeweise zu `if false && fileCount > 0` entwertet ->
  `TestDelete_NonEmptyFolder_Files` wird rot ("Expected error ... but got nil"). Zurueckgedreht,
  `git diff --stat service.go` zeigt wieder ausschliesslich die vorgesehenen 20 Insertions,
  `./internal/document/folder/` erneut gruen.
- verify vorgaenger: sauber. `78339951` (Iteration 71, fix-channel-delete-call-sessions-fk-crash)
  gegen alle acht Fehlerklassen geprueft (`git show --stat` + Volltextdiff von `service.go`,
  `repository.go`, `postgres_repository.go`, `errors.go`, `chat_grpc.go`) — Anwendungs-Guard
  nach exaktem Muster von `ErrCannotDeleteDM`, kein Gateway-Handler betroffen, kein Stub, kein
  `.proto`, kein neuer/ersetzter `RequirePermission`-Guard, keine neue Tabelle, keine Route,
  keine Wire-Shape-Aenderung. `HasCallSessions` braucht keinen Tenant-Filter, weil `id` schon
  ueber `GetByIDForTenant` tenant-gescoped in `Delete` ankommt, bevor die neue Methode
  aufgerufen wird — kein Befund.
- neue-units: keine
- offen: keine

## Iteration 73 — fix-integration-mapping-delete-fk-crash — done — 2026-08-22 09:54
- commit: b6498957
- gebaut: `integration_delivery_log.mapping_id` traegt `ON DELETE NO ACTION`
  (`migrations/000053_create_integration_tables.up.sql:55`), `DeleteChannelMapping`
  (`notification_grpc.go`) rief `s.integrationRepo.DeleteMapping` bisher ohne Vorab-Pruefung auf —
  eine Mapping-Loeschung mit vorhandenen Delivery-Log-Eintraegen waere als roher FK-Fehler in
  `mapNotificationError`s default-Zweig gelandet (500 statt 409). Fix nach dem
  `HasDeals`/`pipelinestage`-Muster: neue Repository-Methode `HasDeliveryLogs(ctx, mappingID)
  (bool, error)` (SELECT EXISTS, RLS traegt die Tenant-Isolation wie bei den bestehenden
  Delivery-Log-Methoden), neuer Sentinel `integration.ErrMappingInUse`, neuer Case in
  `mapNotificationError` -> `codes.FailedPrecondition` (Gateway mappt das bereits generisch auf
  409, `grpcStatusToHTTP` unveraendert). `DeleteChannelMapping` prueft jetzt vor dem Delete.
  OpenAPI-Eintrag `/api/v1/integrations/mappings/{id}` DELETE um `409` samt Beschreibung ergaenzt
  (Route existierte schon, keine neue Route). Zwei Testfakes ergaenzt, die die Interface-Methode
  jetzt mit erfuellen muessen: `fakeRepository` (forwarder_test.go, immer false) und
  `notifIntegStub` (notification_grpc_test.go, ueber neue Map `mappingsInUse` steuerbar) —
  `TestChannelMappingCRUD_HappyPath` deckt jetzt zusaetzlich den 409-Pfad ab. DB-Test
  `HasDeliveryLogs` in `postgres_repository_test.go` (Mapping mit Log-Eintraegen -> true, frisch
  geseedetes Mapping ohne Eintraege -> false), eingehaengt in die bestehende
  `TestPostgresRepository_DeliveryLog`.
- gate: build ok (`./internal/notification/... ./internal/gateway/... ./internal/server/...
  ./cmd/notification/... ./cmd/gateway/...`) | vet ok (`./internal/notification/...
  ./internal/gateway/... ./internal/server/...`) | lint ok (0 issues, dieselben Pfade) | test ok
  (`./internal/notification/integration/` 0 SKIP / 71 PASS inkl. Subtests; volle
  `./internal/notification/...`-Suite gruen; `./internal/server/` gruen; `./internal/gateway/`
  gruen inkl. `TestOpenAPIRouteDrift`) | migration n.a. (kein Schema-Wechsel) | rls-smoke n.a.
  (keine Tabelle/Policy angefasst, nur eine bestehende SELECT-Query wiederverwendet).
- coverage: `internal/notification/integration` 73,5 % -> 73,6 % (selbst gemessen via
  `git stash push -u` auf alle sieben geaenderten Dateien, `go test -coverprofile` davor/danach,
  `stash pop`; `coverage_start:` in der Unit war unbeziffert — "zur Laufzeit messen" stand
  ausdruecklich in der Unit).
- mutations-probe: `if inUse {` in `DeleteChannelMapping` zu `if false && inUse {` entwertet ->
  `TestChannelMappingCRUD_HappyPath` wird rot ("An error is expected but got nil"). Zurueckgedreht,
  `git diff internal/server/notification_grpc.go` zeigt wieder ausschliesslich die vorgesehenen
  12 Insertions, `./internal/server/` erneut gruen.
- verify vorgaenger: sauber. `7346120a` (Iteration 72, fix-document-folder-delete-nonempty-fk-crash)
  gegen alle acht Fehlerklassen geprueft (`git show --stat` + Volltextdiff von `service.go`,
  `errors.go`, `document_grpc.go`) — Anwendungs-Guard nach dem `ErrCircularParent`-Muster, kein
  Gateway-Handler direkt betroffen, kein Stub, kein `.proto`, kein neuer/ersetzter
  `RequirePermission`-Guard, keine neue Tabelle, keine Route, keine Wire-Shape-Aenderung.
  `CountFiles(ctx, id)` nimmt keinen expliziten Tenant-Parameter, aber `id` stammt aus
  `GetByID(ctx, tenantID, id)` am Funktionsanfang — bereits tenant-verifiziert, kein Befund.
- neue-units: keine
- offen: keine

## Iteration 74 — fix-gdpr-deletion-request-audit-trail-cascade-loss — done — 2026-08-22 10:01
- commit: 8752962c
- gebaut: `gdpr_deletion_requests.contact_id` trug `ON DELETE CASCADE` (Migration 000082).
  `consent.Service.ProcessDeletion` loescht den Kontakt nie, sondern anonymisiert ihn nur — der
  Kontakt konnte danach trotzdem ueber einen zweiten, unabhaengigen Pfad hart geloescht werden
  (`contact.Service.Delete` manuell, oder automatisiert `ContactRetentionHandler.Apply` mit
  `action=delete`), und keiner der beiden prueft auf existierende `gdpr_deletion_requests`-Zeilen.
  Der CASCADE loeschte dann die abgeschlossene Loeschanfrage mit — genau der Nachweis, dass eine
  Art.-17-Anfrage bearbeitet wurde, verschwand mit dem Datensatz, den sie betraf.
  Migration 000322: `contact_id` FK auf `ON DELETE SET NULL` umgestellt (Spalte dafuer
  NULL-faehig gemacht), neue Spalte `original_contact_id UUID NOT NULL` OHNE FK als permanenter
  Snapshot, einmalig bei INSERT gesetzt und damit immun gegen jede kuenftige CASCADE/SET NULL auf
  `contacts`. Migration 000082 hatte SET NULL explizit verworfen ("ohne den Contact-Bezug nicht
  mehr auf das Subjekt zurueckfuehrbar") und stattdessen eine "dedizierte Snapshot-Tabelle" als
  Alternative genannt, falls Audit-Trail-Pflicht real wird — genau das liefert
  `original_contact_id`, als Spalte auf der bestehenden Tabelle statt einer neuen, weil die
  Tabelle bereits alle anderen Felder (status, reason, timestamps) traegt, die eine zweite Tabelle
  duplizieren muesste. down-Migration stellt CASCADE wieder her und versucht `contact_id`
  zurueckzubacken (nur fuer Zeilen, deren Kontakt noch existiert; genuin verwaiste Zeilen bleiben
  NULL, NOT NULL wird nur restauriert, wenn das fuer alle Zeilen moeglich ist).
  Go: `GDPRDeletionRequest.ContactID` wird `*uuid.UUID` (nullable), neues Feld
  `OriginalContactID uuid.UUID`. `ProcessDeletion` bekommt einen Guard fuer `ContactID == nil`
  (Kontakt zwischen Request und Processing bereits hart geloescht) — gibt `ErrContactNotFound`
  zurueck statt eines Nil-Pointer-Panics oder eines stillen "completed" ohne echte Anonymisierung.
  `crm_grpc.go` (`RequestDeletion`-Proto-Mapping) nutzt `OriginalContactID` (zum Erstellzeitpunkt
  identisch mit `ContactID`, aber garantiert non-nil). Vier Testdateien angepasst
  (`service_test.go`, `tenant_write_test.go`, `crm_grpc_activities_reports_consent_test.go`,
  `tenant_isolation_phase2_test.go` — deren `SeedRow` fuer `gdpr_deletion_requests` kannte die neue
  NOT-NULL-Spalte nicht und war der einzige weitere Caller). Neuer DB-Test
  `TestGetDeletionRequest_SurvivesContactHardDelete` (cascade_audit_test.go) haertet den
  Kernfall: Kontakt mit "completed"-Request wird per rohem SQL-DELETE hart geloescht (simuliert
  den Retention-/manuellen Pfad, der `gdpr_deletion_requests` nicht kennt), Request-Zeile bleibt
  lesbar, `ContactID` NULL, `OriginalContactID` erhalten. Neuer Unit-Test
  `TestService_ProcessDeletion_ContactAlreadyGone` fuer den Guard.
- gate: build ok (`./internal/crm/... ./internal/server/... ./internal/security/...
  ./internal/gateway/... ./cmd/...`) | vet ok (`./internal/crm/... ./internal/server/...`) | lint
  ok (0 issues, `./internal/crm/consent/... ./internal/server/... ./internal/security/gdpr/...`)
  | test ok (`./internal/crm/consent/` 0 SKIP / 34 PASS; volle `./internal/crm/...`-Suite mit
  `-p 1` gruen — mit `-p N` traten "too many clients"-Verbindungsfehler auf, reines
  Ressourcenlimit des lokalen Postgres bei paralleler Paketausfuehrung, kein Befund zu meiner
  Aenderung; `./internal/security/gdpr/...` mit `-p 1` gruen; `./internal/server/` gruen;
  `./internal/gateway/` gruen inkl. `TestOpenAPIRouteDrift`, obwohl keine Route angefasst wurde)
  | migration ok (000322 up/down/up-Zyklus manuell gegen die lokale DB verifiziert, Schema nach
  up: `contact_id` nullable + `ON DELETE SET NULL`, `original_contact_id NOT NULL` ohne FK; nach
  down: exakter Ursprungszustand `contact_id NOT NULL` + `ON DELETE CASCADE`) | rls-smoke:
  kein separates manuelles psql-Skript, stattdessen die bereits vorhandenen automatisierten
  RLS-Tests fuer exakt diese Tabelle verifiziert (`TestTenantIsolation_GDPR`,
  `TestGDPRDeletionRequestWrites_LandInCallerTenant`, `TestContactExistsAndAnonymize_...`, alle
  gruen) — keine Policy geaendert, nur eine FK-Aktion und eine neue Spalte ohne RLS-Bezug.
- coverage: `internal/crm/consent` 62,7 % -> 63,1 % (selbst gemessen: `git stash push -u` auf
  alle acht geaenderten/neuen Dateien, Migration mit `migrate down 1` auf Vorzustand gebracht,
  `go test -coverprofile` vor der Aenderung, `stash pop`, Migration wieder `up`, `go test
  -coverprofile` nach der Aenderung; `coverage_start:` in der Unit war unbeziffert — "zur
  Laufzeit messen" stand ausdruecklich in der Unit).
- mutations-probe: zwei getrennte Proben fuer die zwei Teile des Fixes. (1) Anwendungscode:
  `if req.ContactID == nil` in `ProcessDeletion` zu `if false && req.ContactID == nil` entwertet
  -> `TestService_ProcessDeletion_ContactAlreadyGone` bricht mit Nil-Pointer-Panic ab (nicht nur
  ein fehlgeschlagener Assert — der Guard verhindert einen echten Crash). Zurueckgedreht,
  `git diff --stat service.go` zeigt wieder ausschliesslich die vorgesehenen Aenderungen,
  `./internal/crm/consent/` erneut gruen. (2) DB-Migration: FK testweise per manuellem `ALTER
  TABLE ... ON DELETE CASCADE` auf den Vorzustand zurueckgesetzt (ohne die .sql-Datei zu
  aendern) -> `TestGetDeletionRequest_SurvivesContactHardDelete` bricht mit "deletion request not
  found" ab, exakt der Bug, den die Migration behebt. FK per `ALTER TABLE ... ON DELETE SET NULL`
  zurueckgedreht, Test erneut gruen — beweist, dass der Test die Migration tatsaechlich testet und
  nicht zufaellig gruen ist.
- verify vorgaenger: sauber. `b6498957` (Iteration 73, fix-integration-mapping-delete-fk-crash)
  gegen alle acht Fehlerklassen geprueft (`git show --stat` + Volltextdiff von `notification_grpc.go`,
  `postgres_repository.go`, `errors.go`, `openapi.yaml`) — Anwendungs-Guard nach dem
  `HasDeals`/`pipelinestage`-Muster, kein Gateway-Handler direkt betroffen (Aenderung liegt im
  gRPC-Server, nicht im Gateway), kein Stub, kein `.proto`, kein neuer/ersetzter
  `RequirePermission`-Guard, keine neue Tabelle, Route existierte schon und wurde in openapi.yaml
  korrekt um den 409-Fall ergaenzt, keine Wire-Shape-Aenderung.
- neue-units: keine
- offen: keine

## Iteration 75 — feat-retention-worker-handler-work-tasks — done — 2026-08-22 10:16
- commit: 33bea578926398c582461b3d1a3fd0284524e3c1
- gebaut: `TaskRetentionHandler` (`backend/internal/security/gdpr/retention_work_tasks.go`) als
  sechster Handler auf der Retention-Registry aus A10, `resource_type` "tasks", verdrahtet in
  `cmd/auth/main.go` neben den fuenf bestehenden Handlern. Scope sind abgeschlossene Aufgaben:
  `completed_at IS NOT NULL AND completed_at < cutoff` (bei anonymize zusaetzlich
  `updated_at < cutoff` fuer Idempotenz, wie bei `HelpdeskTicketRetentionHandler`).
  `completed_at` wird von `task.Service.MoveTask` genau einmal beim Uebergang in eine
  geschlossene Status-Kategorie gesetzt und beim Wiedereroeffnen wieder geleert
  (`service.go:423-426`) — damit ist die Spalte zugleich Abschlusszeitpunkt und zuverlaessiges
  "ist wirklich fertig"-Flag, ohne separaten Blick auf `project_statuses`.
  `task_comments.task_id` ist `ON DELETE CASCADE` auf `tasks` (Migration 000026) — Delete nutzt
  das fuer den Kommentar-Trail, ein Handler auf "tasks" allein reicht fuer beide Tabellen. Bei
  anonymize wird `task_comments.content` explizit geleert, analog zu `ticket_messages.body`
  in A13.
  Zusatz-Sicherung gegenueber dem Ticket-Vorbild: `tasks.parent_task_id` ist ebenfalls
  `ON DELETE CASCADE` auf `tasks` (Subtask-Beziehung, Migration 000025). Ein Delete-Plan
  wuerde ohne Guard eine noch aktive Unteraufgabe (`completed_at IS NULL`) unbemerkt per
  Kaskade mitloeschen. `Plan` prueft deshalb vor dem Delete-Fall per Zweitabfrage, welche
  Kandidaten einen direkten aktiven Kind-Task haben, und verschiebt die betroffenen von `Due`
  nach `Skipped` mit Begruendung — Anonymize ist davon nicht betroffen, weil dort keine Zeile
  geloescht wird. Tiefere Hierarchien (aktiver Enkel unter abgeschlossenem Kind) werden nicht
  rekursiv geprueft, im Code als bewusste Grenze kommentiert — im aktuellen Modul-Gebrauch
  kommt eine dritte Ebene nicht vor.
  `time_entries` bleibt wie in den Notes gefordert unangetastet (eigene, bereits offene Unit
  `fix-work-erasure-time-entries-retention-conflict`), keine Migration noetig — `tenant_id` auf
  `tasks`/`task_comments` existiert bereits (Retrofit-Spalte, nicht in der urspruenglichen
  Migration 000025/000026 sichtbar, aber `\d tasks`/`\d task_comments` gegen die lokale DB
  bestaetigt NOT NULL + RLS-Policy auf beiden Tabellen — vor dem Bauen geprueft statt
  angenommen, wie in den Notes verlangt).
- gate: build ok (`./internal/security/... ./cmd/auth/...`) | vet ok (`./internal/security/...`)
  | lint ok (0 issues, `./internal/security/...`) | test ok (`./internal/security/gdpr/` 6 neue
  Tests PASS, 0 SKIP; volle `./internal/security/gdpr/...`- und `./internal/work/...`-Suiten
  gruen) | migration: n.a. (keine neue Migration, `tenant_id`/RLS auf beiden Tabellen bereits
  vorhanden) | rls-smoke: n.a. (keine Policy geaendert; Tenant-Scoping ueber `WHERE tenant_id
  = $1` in jeder Query, per DB-Test mit zwei Tenants explizit belegt statt nur angenommen)
- coverage: `internal/security/gdpr` 71,0 % -> 71,3 % (selbst gemessen: neue Dateien
  `retention_work_tasks.go`/`_test.go` kurzzeitig nach `/tmp` verschoben, `go test
  -coverprofile` vor der Aenderung, Dateien zurueckgeholt, `go test -coverprofile` danach;
  `coverage_start:` in der Unit war unbeziffert — "zur Laufzeit messen" stand ausdruecklich in
  der Unit)
- mutations-probe: den Subtask-Guard in `Plan` mit `if true || action != ...` entwertet ->
  `TestTaskRetentionHandler_PlanDeleteSkipsTaskWithActiveSubtask` bricht mit zwei Assert-Fehlern
  ab (aktive Unteraufgabe waere doch in `Due` gelandet, `Skipped` leer statt 1 Eintrag).
  Zurueckgedreht, `grep -n "true ||"` liefert keinen Treffer mehr, volle Testdatei erneut gruen.
- verify vorgaenger: sauber. `8752962c` (Iteration 74, fix-gdpr-deletion-request-cascade-loss)
  gegen alle acht Fehlerklassen geprueft (`git show --stat` + Volltextdiff von `service.go`,
  `repository.go`, `postgres_repository.go`, `crm_grpc.go`, Migration 000322) — kein
  Gateway-Handler betroffen, kein Stub, `.proto` unveraendert, kein neuer/ersetzter
  `RequirePermission`-Guard, keine neue Tabelle (Spalten-Aenderung auf bestehender Tabelle mit
  RLS-Policy, die nicht angefasst wurde), keine neue Route, keine Wire-Shape-Aenderung. Alle
  weiteren `.ContactID`-Dereferenzierungen im Repo geprueft (`crm_grpc.go:3112`/`3155`
  betreffen `ConsentRecord`/`ConsentSummary`, ein anderer, nicht-nullable Typ — kein
  Nil-Pointer-Risiko durch die Aenderung).
- neue-units: keine
- offen: keine

## Iteration 76 — feat-retention-worker-handler-calendar-events — done — 2026-08-22 10:24
- commit: 2ae62384fbe9f97ab875eb785f52078c92223edc
- gebaut: `CalendarEventRetentionHandler` (`backend/internal/security/gdpr/retention_calendar_events.go`)
  als siebter Handler auf der Retention-Registry aus A10, `resource_type` "calendar_events",
  verdrahtet in `cmd/auth/main.go` neben den sechs bestehenden Handlern.
  Kernproblem: `calendar_events.start_time`/`end_time` sind nur die ERSTE Vorkommnis eines
  Termins. Ein taeglicher Serientermin (`rrule` gesetzt), vor zwei Jahren angelegt, laeuft
  heute noch, wenn `recurrence_end` NULL ist (unbefristete Serie). `Plan` behandelt beide Faelle
  getrennt: Einzeltermine (`rrule IS NULL`) sind faellig ueber `end_time < cutoff`, Serien nur
  ueber `recurrence_end IS NOT NULL AND recurrence_end < cutoff` — eine unbefristete Serie
  taucht bei KEINEM Cutoff auf, weil ihr naechstes Vorkommen unbekannt ist. Das ist die
  "explizite Behandlung" aus dem `done_when`, nicht nur ein Kommentar.
  `event_attendees`, `event_exceptions`, `event_reminders` cascadieren alle per FK direkt auf
  `calendar_events(id) ON DELETE CASCADE` (Migration 000033, im Code gegengeprueft, nicht
  geraten) — ein Hard-Delete nimmt alle drei in derselben Anweisung mit. Bei `anonymize` (die
  Zeile bleibt bestehen) leert der Handler zusaetzlich `title`/`description`/`location` auf
  `calendar_events` UND setzt ein etwaiges Override in `event_exceptions` (dieselben drei
  Felder) auf NULL zurueck statt auf das Label — NULL bedeutet in diesem Schema "erbt vom
  Elternereignis", ein Bracket-Label haette dort eine zweite, redundante Anonymisierung neben
  der bereits anonymisierten Elternzeile stehen lassen.
  `calendar_events.tenant_id` existiert bereits als Retrofit-Spalte (Migration 000106) mit
  RLS-Policy (Migration 000122, `enable_tenant_rls('calendar_events')`) — vor dem Bauen an der
  Migrationshistorie gegengeprueft, nicht angenommen. Keine neue Migration noetig.
- gate: build ok (`./internal/security/... ./internal/work/... ./cmd/auth/...`) | vet ok
  (`./internal/security/... ./internal/work/...`) | lint ok (0 issues, beide Pakete) | test ok
  (`./internal/security/gdpr/` 7 neue Tests PASS, 0 SKIP; volle `./internal/security/gdpr/...`-
  und `./internal/work/...`-Suiten gruen, 143 PASS / 0 SKIP im gdpr-Paket) | migration: n.a.
  (keine neue Migration, `tenant_id`/RLS auf `calendar_events` bereits vorhanden) | rls-smoke:
  n.a. (keine Policy geaendert; Tenant-Isolation per DB-Test mit zwei Tenants explizit belegt,
  `TestCalendarEventRetentionHandler_PlanOnlyMatchesPastCutoffAndIsTenantScoped`)
- coverage: `internal/security/gdpr` 71,3 % -> 71,4 % (selbst gemessen: neue Dateien
  `retention_calendar_events.go`/`_test.go` kurzzeitig nach `/tmp` verschoben, `go test
  -coverprofile` vor der Aenderung, Dateien zurueckgeholt, `go test -coverprofile` danach;
  `coverage_start:` in der Unit war unbeziffert — "zur Laufzeit messen" stand ausdruecklich in
  der Unit)
- mutations-probe: die Serien-Klausel `rrule IS NOT NULL AND recurrence_end IS NOT NULL AND
  recurrence_end < $2` zu `rrule IS NOT NULL` verkuerzt (jede Serie unabhaengig von
  `recurrence_end` faellig) -> `TestCalendarEventRetentionHandler_PlanExcludesOpenEndedRecurringSeries`
  UND `TestCalendarEventRetentionHandler_PlanRecurringSeriesUsesRecurrenceEnd` brechen beide ab
  (unbefristete Serie waere doch faellig, in Zukunft rekurrierende Serie ebenso). Zurueckgedreht,
  `git status`/Volltextvergleich zeigt wieder ausschliesslich die vorgesehene Klausel, beide
  Tests erneut gruen, volle Testdatei erneut gruen.
- verify vorgaenger: sauber. `33bea578` (Iteration 75, feat-retention-worker-handler-work-tasks)
  gegen alle acht Fehlerklassen geprueft (`git show --stat` + Volltextdiff von
  `retention_work_tasks.go`, `retention_work_tasks_test.go`, `cmd/auth/main.go`) — kein
  Gateway-Handler betroffen, kein Stub, kein `.proto`, kein neuer/ersetzter
  `RequirePermission`-Guard, keine neue Tabelle (reine Registry-Erweiterung auf bestehenden,
  tenant-gescopten Tabellen), keine neue Route, keine Wire-Shape-Aenderung. Der separate
  `f8b8ac20`-Commit direkt davor war nur die nachtraegliche SHA-Eintragung ins Journal (Muster
  aus fruaheren Laeufen), kein eigenstaendiger Code-Commit.
- neue-units: keine
- offen: keine

## Iteration 77 — feat-retention-worker-handler-notifications — done — 2026-08-22 10:31
- commit: 7c5436bc
- gebaut: `NotificationRetentionHandler` (`backend/internal/security/gdpr/retention_notifications.go`)
  als achter Handler auf der Retention-Registry aus A10, `resource_type` "notifications",
  verdrahtet in `cmd/auth/main.go` neben den sieben bestehenden Handlern.
  Zwei Entscheidungen aus den Notes umgesetzt: `SupportsAction` erlaubt NUR delete — eine
  anonymisierte Benachrichtigung ("[geloescht] hat ... erwaehnt") ist fuer den Empfaenger wertlos,
  die Begruendung steht im Datei-Header, nicht nur im Journal. `Plan` filtert `is_read = true`
  direkt in der WHERE-Klausel, nicht als nachtraeglicher Skip: eine ungelesene Benachrichtigung
  ist nie faellig, egal wie alt — sonst verschwindet eine Nachricht, die der Empfaenger nie zu
  Gesicht bekommen hat, lautlos (Feature-Verlust, keine Aufraeumung).
  `notifications.tenant_id` existiert bereits seit Migration 000106 (Retrofit Phase 1) mit
  RLS-Policy seit 000122 (`enable_tenant_rls('notifications')`) — vor dem Bauen an der
  Migrationshistorie gegengeprueft (`grep -rl notifications backend/migrations`), nicht
  angenommen. Keine neue Migration noetig.
  `retention_test.go` haelt bereits einen lokalen, testinternen `notificationRetentionHandler`
  (Beweis fuer die Engine, ohne `is_read`-Filter und mit Anonymize+Delete) — bewusst NICHT
  wiederverwendet, weil er genau die beiden Faelle nicht abdeckt, die diese Unit fordert
  (ungelesen ausschliessen, nur delete). Die Produktions-Klasse ist eine neue, eigene Datei.
- gate: build ok (`./internal/security/... ./internal/notification/... ./cmd/auth/...`) | vet ok
  (`./internal/security/... ./internal/notification/...`) | lint ok (0 issues, beide Pakete) |
  test ok (`./internal/security/gdpr/` 5 neue Tests PASS, 0 SKIP; volle
  `./internal/security/gdpr/...`-, `./internal/security/...`- und `./internal/notification/...`-
  Suiten gruen — ACHTUNG: der erste Lauf ueber den vollen `./internal/security/...`-Baum ohne
  `-p 1` schlug in `password`/`vendoraccess`/`vault` mit "remaining connection slots are reserved
  for roles with the SUPERUSER attribute" fehl, weil zu viele Pakete gleichzeitig eigene Pools
  gegen die lokale Docker-Postgres oeffnen; mit `-p 1` (seriell) alle 7 Pakete gruen, `gdpr` isoliert
  ebenfalls durchgehend gruen — kein Bug dieser Aenderung, reines lokales Connection-Limit) |
  migration: n.a. (keine neue Migration, `tenant_id`/RLS auf `notifications` bereits vorhanden) |
  rls-smoke: n.a. (keine Policy geaendert; Tenant-Isolation per DB-Test mit zwei Tenants explizit
  belegt, `TestNotificationRetentionHandler_PlanOnlyMatchesReadPastCutoffAndIsTenantScoped`)
- coverage: `internal/security/gdpr` 71,4 % -> 71,5 % (selbst gemessen: neue Dateien
  `retention_notifications.go`/`_test.go` kurzzeitig nach `/tmp` verschoben, Wiring-Zeile in
  `cmd/auth/main.go` per `sed` temporaer entfernt, `go test -coverprofile` vor der Aenderung,
  Dateien und Wiring zurueckgeholt, `go test -coverprofile` danach)
- mutations-probe: die Plan-Query von `WHERE tenant_id = $1 AND is_read = true AND created_at < $2`
  auf `WHERE tenant_id = $1 AND created_at < $2` verkuerzt (ungelesene Benachrichtigungen waeren
  doch faellig) -> `TestNotificationRetentionHandler_PlanExcludesUnreadRegardlessOfAge` bricht ab.
  Zurueckgedreht, `git status` zeigt die Datei wieder als komplett unveraendert gegen den
  Arbeitsstand (unstaged, keine Differenz zur eigenen Neufassung), Test erneut gruen, volle
  Testdatei erneut gruen.
- verify vorgaenger: sauber. `2ae62384` (Iteration 76, feat-retention-worker-handler-calendar-events)
  gegen alle acht Fehlerklassen geprueft (`git show --stat` + Volltextlesung von
  `retention_calendar_events.go` und Diff von `cmd/auth/main.go`) — kein Gateway-Handler betroffen,
  kein Stub, kein `.proto`, kein neuer/ersetzter `RequirePermission`-Guard, keine neue Tabelle
  (`calendar_events.tenant_id`/RLS bereits seit 000106/000122 vorhanden, gegengeprueft), keine neue
  Route, keine Wire-Shape-Aenderung. Der separate `b2aaeb2a`-Commit davor war nur die nachtraegliche
  SHA-Eintragung ins Journal, kein eigenstaendiger Code-Commit.
- neue-units: keine
- offen: keine

## Iteration 78 — fix-gateway-bexio-error-message-leakage — done — 2026-08-22 10:37
- commit: 161d94a5
- gebaut: Root Cause statt Symptom gefixt. `HandleBexioOAuthCallback`, `PushInvoiceToBexio`,
  `PushQuoteToBexio` (`backend/internal/server/bexio_grpc.go`) bauten bei Fehler bisher
  `Success:false, ErrorMessage: err.Error()` statt eines echten gRPC-Fehlers und umgingen damit
  `mapBexioError` vollstaendig (der Maskierungsmechanismus, den alle anderen Bexio-RPCs schon
  nutzen) — Vault-/DB-Rohtexte konnten so bis zur HTTP-Antwort durchsickern. Alle drei geben jetzt
  `nil, mapBexioError(err)` zurueck, exakt das Muster von `DisconnectBexio`/`TriggerBexioSync`/etc.
  Zweiter, separater Fund in derselben Handler-Umgebung: `route_bexio.go`s
  `HandleOAuthCallback` schrieb sowohl den externen Query-Parameter `error` (von Bexio selbst
  gesendet, oeffentlicher Endpunkt, damit angreiferkontrollierbar) als auch (defensiv, siehe unten)
  `resp.GetErrorMessage()` UNESCAPED in eine Redirect-Location — ein rohes `&` haette einen
  zusaetzlichen Query-Parameter injiziert, ein rohes CRLF waere Response-Splitting vor manchen
  Proxies gewesen. Neue Helper-Funktion `redirectBexioError` (`route_bexio.go`) escaped jeden
  Bexio-Error-Code ueber `url.QueryEscape`, bevor er in die Location wandert; beide Redirect-Stellen
  (missing-code-Pfad und OAuth-Callback-Ergebnis) nutzen sie jetzt.
  `!resp.GetSuccess()` bleibt als Verteidigungslinie neben `err != nil` erhalten (auskommentiert
  begruendet im Code) statt geloescht: nach dem Server-Fix kann Success bei `err == nil` nicht mehr
  false sein, aber die Kombiprüfung verhindert, dass eine kuenftige Regression dort wieder einen
  unmaskierten String durchreicht.
  Kein `.proto`-Change: `Success`/`ErrorMessage` bleiben als Felder bestehen (harmlos ungenutzt bei
  Erfolg), keine Regenerierung noetig.
- gate: build ok (`./internal/gateway/... ./internal/server/... ./cmd/gateway/...`) | vet ok
  (dieselben Pakete) | lint ok (0 issues, `./internal/gateway/... ./internal/server/...`) | test ok
  (`./internal/gateway/...` und `./internal/server/...` inkl. `TestOpenAPIRouteDrift` — keine Route
  geaendert, 836 registrierte gegen 838 dokumentierte Pfade, gruen; `./internal/biz/bexio/...` zur
  Kontrolle ebenfalls gruen) | migration: n.a. (keine Tabellen-/Policy-Aenderung) | rls-smoke: n.a.
- coverage: `internal/gateway` 54,0 % -> 54,0 % (kein messbarer Unterschied auf eine Nachkommastelle,
  vor/nach selbst gemessen per `git stash` der vier geaenderten Dateien) | `internal/server` 70,2 %
  -> 70,2 % (gleiches Verfahren). Fix-Unit, keine Coverage-Unit — die Zahl ist Nebenprodukt, nicht
  Ziel; die eigentliche Baseline `internal/gateway 46,1 %` aus dem Laufkopf ist durch die
  vorangegangenen Block-B-Iterationen laengst ueberholt, 54,0 % ist der tatsaechliche Ist-Stand vor
  UND nach dieser Unit.
- mutations-probe: in `redirectBexioError` das `url.QueryEscape(code)` durch das rohe `code` ersetzt
  -> `TestHandleOAuthCallback_MissingCode_ErrorParamIsEscaped` bricht mit sichtbarem rohen CRLF in der
  Location ab. Zurueckgedreht, `git diff` zeigt wieder exakt den urspruenglichen Fix-Diff (24
  geaenderte Zeilen wie vor der Probe), volle Testdatei erneut gruen.
- verify vorgaenger: sauber. `7c5436bc` (Iteration 77, feat-retention-worker-handler-notifications)
  gegen alle acht Fehlerklassen geprueft (`git show --stat` + Volltextlesung von
  `retention_notifications.go` und Diff von `cmd/auth/main.go`) — kein Gateway-Handler betroffen
  (reiner `internal/security/gdpr`-Handler auf dem bestehenden Retention-Motor), kein Stub, kein
  `.proto`, kein neuer/ersetzter `RequirePermission`-Guard, keine neue Tabelle (`notifications.tenant_id`
  + RLS bereits seit 000106/000122 vorhanden, im Journal selbst gegengeprueft), keine neue Route,
  keine Wire-Shape-Aenderung. Der separate `0b81c936`-Commit davor war nur die nachtraegliche
  SHA-Eintragung ins Journal, kein eigenstaendiger Code-Commit.
- neue-units: keine
- offen: keine

## Iteration 79 — fix-gateway-datev-upload-error-message-leakage — done — 2026-08-22 10:50
- commit: c4aa55e0
- gebaut: Identisches Muster wie fix-gateway-bexio-error-message-leakage (Iteration 78), andere
  Integration. `HandleDatevOAuthCallback` (`internal/server/datev_upload_grpc.go`) baute bei
  Fehler `Success:false, ErrorMessage: err.Error()` statt eines echten gRPC-Status — `err` kommt
  aus `OAuthManager.ExchangeCode` (`internal/biz/datev/oauth.go`), die Vault-/HTTP-Fehler wrappt
  (bestaetigt: `TestExchangeCode_VaultStoreFailureFailsTheExchange` zeigt "failed to store refresh
  token: %w" um einen Vault-Fehler). Jetzt `return nil, status.Error(codes.Internal, "DATEV OAuth
  callback failed")` — anders als bei Bexio kein Wiederverwenden von `mapDatevUploadError`, weil
  dessen Sentinel-Zweige (ErrInvoiceNotFound, ErrInvalidPeriod, ...) reine Upload-Fehler sind und
  fuer den OAuth-Callback nie zutreffen; die generische Nachricht ist bewusst neu und eigen
  benannt statt die irrefuehrende "DATEV upload failed" zu recyceln.
  `route_datev_upload.go`s `HandleOAuthCallback` schrieb sowohl den externen Query-Parameter
  `error` (von DATEV selbst gesendet, oeffentlicher Endpunkt, angreiferkontrollierbar) als auch
  `resp.GetErrorMessage()` UNESCAPED in eine Redirect-Location — gleiches Injection-Risiko wie bei
  Bexio. Neue Helper-Funktion `redirectDatevError` escaped jeden Fehlercode ueber
  `url.QueryEscape`, bevor er in die Location wandert; alle vier Redirect-Stellen
  (state_not_configured, missing-code/error-Pfad, invalid_state, OAuth-Ergebnis) nutzen sie jetzt.
  `!resp.GetSuccess()` bleibt neben `err != nil` als auskommentiert begruendete Verteidigungslinie
  erhalten (identische Begruendung wie beim Bexio-Fix).
  Zusaetzlich als Lean-Cleanup: die im Scope genannten JSON-Stellen `route_datev_upload.go:278-279`
  und `315-316` (`HandleUploadBuchungsstapel`/`HandleUploadBeleg`) waren bereits TOTER Code — die
  einzigen Konstruktionsstellen der beiden Response-Protos setzen `ErrorMessage` nirgends (grep
  bestaetigt), weil `UploadDatevBuchungsstapel`/`UploadDatevBeleg` serverseitig schon seit einer
  frueheren Iteration ueber `mapDatevUploadError` einen echten gRPC-Fehler statt Success:false
  liefern. `GetErrorMessage()` kann dort nie non-empty sein; die tote `if`-Verzweigung entfernt,
  mit Kommentar warum.
  Die dritte im Scope genannte Stelle (`route_datev_upload.go:431-432`, `ListDatevUploadLogs`) ist
  KEIN Fund derselben Klasse und bewusst NICHT angefasst: `l.ErrorMessage` kommt aus
  `UploadService.failUploadLog` (`upload_service.go:241-250`), das echte, bereits in der DB
  gespeicherte historische Fehlertexte aus `Uploader.doWithRetry`
  (`internal/biz/datev/uploader.go:212/221`) haelt — die rohe Antwort der DATEV-API, keine
  Vault-/DB-Interna. Route liegt hinter `RequireRole("admin")` und ist tenant-gescoped
  (`ListDatevUploadLogs` filtert auf `tenant_id`). Gleiche Guenstigere-Risikoklasse wie die
  "historischen Sync-Log-Meldungen" bei `fix-gateway-lexware-error-message-leakage` (naechste
  Unit im Backlog) — dort ausdruecklich als kleineres, separates Risiko benannt statt zwingend zu
  maskieren. Entscheidung: unveraendert lassen, admin-eigene operative Diagnosedaten sind hier
  nuetzlicher als ein Masking-Verlust.
  Kein `.proto`-Change: `Success`/`ErrorMessage` bleiben als Felder bestehen (ungenutzt bei
  Erfolg), keine Regenerierung noetig.
- gate: build ok (`./internal/gateway/... ./internal/server/... ./cmd/gateway/...`) | vet ok
  (dieselben Pakete) | lint ok (0 issues, `./internal/gateway/... ./internal/server/...`) | test ok
  (`./internal/gateway/` inkl. `TestOpenAPIRouteDrift` — keine Route geaendert, gruen;
  `./internal/server/...` und `./internal/biz/datev/...` gruen, DATABASE_URL gegen kmuhub_app
  gesetzt, 0 SKIP) | migration: n.a. (keine Tabellen-/Policy-Aenderung) | rls-smoke: n.a.
- coverage: `internal/gateway` 54,0 % -> 54,1 % (vor/nach per `git stash` der vier geaenderten
  Dateien selbst gemessen) | `internal/server` 70,2 % -> 70,3 % (gleiches Verfahren). Fix-Unit,
  keine Coverage-Unit — die Zahl ist Nebenprodukt der zwei neuen Testfaelle plus toter-Code-
  Entfernung, nicht Ziel.
- mutations-probe: zwei getrennte Proben, beide zurueckgedreht.
  (1) `redirectDatevError`s `url.QueryEscape(code)` durch `url.QueryEscape("")+code` ersetzt (leere
  Escape-Nutzung, echter Wert unescaped) -> `TestDatevHandleOAuthCallback_ExternalErrorParamIsEscaped`
  bricht mit sichtbarem rohen CRLF in der Location ab. Zurueckgedreht, `git diff --stat` zeigt
  wieder exakt den Ausgangsdiff (29 Einfuegungen/24 Loeschungen wie vor der Probe), volle Testdatei
  erneut gruen.
  (2) `HandleDatevOAuthCallback`s maskierten `status.Error(codes.Internal, "DATEV OAuth callback
  failed")` durch `status.Error(codes.Internal, err.Error())` ersetzt (Rohtext durchgereicht) ->
  `TestHandleDatevOAuthCallback_MasksInternalError` bricht ab und zeigt den vollen Vault-Fehlertext
  inkl. CRLF und `&` im Fehler-Message-Feld. Zurueckgedreht, Diff wieder exakt 4 Einfuegungen/4
  Loeschungen wie vor der Probe, Testdatei erneut gruen.
- verify vorgaenger: sauber. `161d94a5` (Iteration 78, fix-gateway-bexio-error-message-leakage)
  gegen alle acht Fehlerklassen geprueft (`git show --stat` + Volltextlesung von `route_bexio.go`
  und `bexio_grpc.go`) — kein Gateway-Handler ruft eine Service-Instanz direkt (weiterhin ueber
  `bizv1.BexioServiceClient`), kein Stub, kein `.proto`-Change (Success/ErrorMessage-Felder bleiben
  bestehen, ungenutzt), kein neuer/ersetzter `RequirePermission`-Guard, keine neue Tabelle, keine
  neue Route (`redirectBexioError` ist ein interner Helper, keine HTTP-Route), keine
  Wire-Shape-Aenderung. `mapBexioError` ist ein bereits bestehender, seit laengerem genutzter
  Mechanismus — kein neu erfundener zweiter Maskierungspfad.
- neue-units: keine. Der beim Bauen gefundene tote Code
  (`route_datev_upload.go:278-279`/`315-316`) war bereits durch eine fruehere Iteration
  serverseitig entschaerft (kein Leck mehr moeglich) und wurde direkt in dieser Unit als
  Lean-Cleanup mitentfernt statt als eigene Unit angelegt — kein Verhalten geaendert, nur toter
  Code weg.
- offen: keine

## Iteration 80 — fix-gateway-lexware-error-message-leakage — done — 2026-08-22 11:02
- commit: 48ba72ff
- gebaut: vierter Fund derselben Fehlerklasse (nach Bexio, DATEV OAuth, DATEV Upload):
  `ConnectLexware`, `TestLexwareConnection`, `PushInvoiceToLexware`, `PushQuoteToLexware`
  (`internal/server/lexware_grpc.go`) gaben bei Fehler `Success:false, ErrorMessage:
  err.Error()` zurueck statt eines echten gRPC-Status — `Connect`/`TestConnection` wickeln
  Vault- bzw. HTTP-Fehler mit `fmt.Errorf("lexware: ...: %w", err)`, deren Text bei einem
  Vault-Ausfall oder ungueltigem Key direkt beim Client landete. Alle vier Stellen rufen jetzt
  `mapLexwareError(err)` — der bereits bestehende, seit laengerem fuer `DisconnectLexware`,
  `TriggerLexwareSync` usw. genutzte Mechanismus (kein neuer zweiter Maskierungspfad). Bekannte
  Lexware-Sentinelfehler (unauthorized, rate limited, not found, sync conflict, ...) bleiben
  ueber `errors.Is`-Unwrapping korrekt klassifiziert; alles andere faellt auf den
  default-Zweig `codes.Internal, "internal error"`.
  `route_lexware.go`: die vier zugehoerigen `if resp.GetErrorMessage() != "" { ... }`-Zweige in
  `HandleConnect`, `HandleTestConnection`, `HandlePushInvoice`, `HandlePushQuote` sind jetzt
  toter Code (die Response ist bei `err == nil` immer ein voller Erfolg) und wurden im selben
  Commit entfernt — `HandleConnect` verlor damit auch seinen 400-Pfad
  (`response.Error(w, http.StatusBadRequest, resp.GetErrorMessage())`), der laut Scope der
  unmittelbarste der drei/vier Funde war, weil dort kein Escaping-Umweg half, sondern der
  Fehlertext selbst nicht ankommen durfte.
  BEWUSST NICHT angefasst (wie im Scope explizit als "separates, kleineres Risiko" markiert):
  `route_lexware.go` HandleListSyncLogs (`entry["error_message"]`) und HandleGetSyncStatus
  (`last_sync_error`) — beide lesen `l.ErrorMessage`/`status.LastSyncError`, das aus
  `latestLog.ErrorMessage` (`service.go:331`, historische, bereits in der DB gespeicherte
  Sync-Log-Eintraege, `contact_sync.go:114`) stammt. Route liegt hinter
  `middleware.RequireRole("admin")` und ist tenant-gescoped. Gleiche Entscheidung und
  Begruendung wie bei `fix-gateway-datev-oauth-callback-error-leakage` (Iteration 79) fuer
  `ListDatevUploadLogs`.
  `HandleLexwareWebhookEvent` (server) wurde NICHT angefasst: sie gibt bei Fehler nur
  `Success:false` ohne `ErrorMessage` zurueck — kein Leak vorhanden.
  Kein `.proto`-Change: `Success`/`ErrorMessage`-Felder bleiben bestehen (ungenutzt bei
  Erfolg), keine Regenerierung noetig.
  Neue Testdatei `internal/server/lexware_grpc_test.go`: `TestMapLexwareError` (Tabellentest
  ueber alle Sentinel-Codes + unbekannte Ursache), `TestMapLexwareErrorHidesUnknownCause` und
  `TestConnectLexware_MasksInternalError` (Mutations-Probe-Ziel, mit einem `leakyLexwareVaultStub`
  nach Vorbild des DATEV-Fixes aus Iteration 79 — `SetSecret` liefert einen Fehlertext mit
  CRLF und IP/Port, der frueher im Klartext in der Antwort gelandet waere).
- gate: build ok (`./internal/gateway/... ./internal/server/... ./cmd/gateway/...`) | vet ok
  (dieselben Pakete) | lint ok (0 issues, `./internal/gateway/... ./internal/server/...`) |
  test ok (`./internal/server/` 0 SKIP, `./internal/gateway/` inkl. `TestOpenAPIRouteDrift` —
  keine Route/Pfad-Aenderung, gruen; `./internal/biz/lexware/...` gruen), DATABASE_URL gegen
  kmuhub_app gesetzt | migration: n.a. (keine Tabellen-/Policy-Aenderung) | rls-smoke: n.a.
- coverage: `internal/server` 70,3 % -> 70,4 % | `internal/gateway` 54,1 % -> 54,1 %
  (vor/nach per `git stash` der drei geaenderten Dateien selbst gemessen — Gateway-Zahl
  unveraendert, da nur Handler-Inhalt, keine neue Codepfad-Verzweigung, gestrichen wurde).
  Fix-Unit, keine Coverage-Unit — Nebenprodukt der neuen Testdatei und der entfernten
  toten Zweige, nicht Ziel.
- mutations-probe: `ConnectLexware`s `mapLexwareError(err)` durch
  `status.Error(codes.Internal, err.Error())` ersetzt (Rohtext durchgereicht) ->
  `TestConnectLexware_MasksInternalError` bricht ab und zeigt den vollen Vault-Fehlertext
  inkl. CRLF, IP und `Set-Cookie`-Injection-Versuch im Message-Feld. Zurueckgedreht,
  `git diff --stat` zeigt wieder exakt 4 Einfuegungen/16 Loeschungen wie vor der Probe (vier
  Fundstellen), Testdatei erneut gruen.
- verify vorgaenger: sauber. `c4aa55e0` (Iteration 79, fix-gateway-datev-oauth-callback-error-leakage)
  gegen alle acht Fehlerklassen geprueft (`git show --stat` + Volltextlesung von
  `route_datev_upload.go` und `datev_upload_grpc.go`) — Gateway-Handler ruft weiterhin ueber
  `dr.getDatevUploadClient()`/`bizv1.DatevUploadServiceClient` (kein direkter Service-Aufruf,
  bestaetigt per Grep auf `dr.registry`), kein Stub, kein `.proto`-Change, kein
  neuer/ersetzter `RequirePermission`-Guard, keine neue Tabelle, keine neue Route
  (`redirectDatevError` ist ein interner Helper), keine Wire-Shape-Aenderung.
- neue-units: keine.
- offen: keine

## Iteration 81 — fix-gateway-caldav-basic-auth-username-log-leakage — done — 2026-08-22 11:11
- commit: 16550c5e
- gebaut: Fund aus scan-gateway-pii-in-logs (Iteration 45). Zwei Log-Aufrufe mit identischem
  Root Cause (roher Basic-Auth-Username landet ungeprueft im Log) gefixt:
  `basicAuthMiddleware` in `route_caldav.go:183-186` (`slog.Warn("caldav basic auth failed",
  "username", username, ...)`, erreichbar ohne vorherige Authentifizierung) und
  `AppPasswordService.Validate` in `app_password.go:104-106` (`slog.Warn("caldav auth: invalid
  username format", "username", username)`, derselbe Wert, Service-Schicht als Nebeneingang).
  Beide Stellen loggen jetzt `username_fingerprint` statt `username` — ein package-lokaler
  `fingerprintCaldavUsername`/`fingerprintUsername` (sha256, erste 4 Bytes als 8 Hex-Zeichen),
  in gateway UND caldav dupliziert statt geteilt, weil `internal/caldav` bereits
  `internal/gateway` importiert (caldav_backend.go/carddav_backend.go) — ein Import in die
  Gegenrichtung waere ein Zyklus. Zwei Zeilen Duplikat sind hier die schlankere Wahl als ein
  neues Shared-Package fuer eine Ein-Zeilen-Funktion.
  Die Debug-Information selbst (ungueltiges Format / unbekannte ID / falsches Passwort) bleibt
  unveraendert im "error"-Feld; der Fingerprint erlaubt weiterhin, wiederholte Versuche
  desselben (ungueltigen) Werts fuer Anomalie-/Rate-Limit-Analyse zu erkennen, ohne die
  E-Mail-Adresse im Klartext zu zeigen.
- gate: build ok (`./internal/gateway/... ./internal/caldav/... ./cmd/gateway/...`) | vet ok
  (dieselben Pakete) | lint ok (0 issues) | test ok (`./internal/gateway/` inkl.
  `TestOpenAPIRouteDrift`, 0 SKIP; `./internal/caldav/` 0 SKIP), DATABASE_URL gegen kmuhub_app
  gesetzt | migration: n.a. (kein Schema-Change) | rls-smoke: n.a. (keine Tabelle/Policy
  angefasst)
- coverage: `internal/caldav` 54,8 % -> 54,9 % (per `git stash` der vier geaenderten Dateien
  selbst gemessen). `internal/gateway` unveraendert bei 54,1 % (kein neuer Codepfad, nur ein
  Log-Feld ersetzt) — Fix-Unit, keine Coverage-Unit, beides Nebenprodukt der zwei neuen Tests.
- mutations-probe: zweimal durchgefuehrt, je Fundstelle einmal. (1)
  `route_caldav.go`: `fingerprintCaldavUsername(username)` durch `username` ersetzt ->
  `TestBasicAuthMiddleware_FailedLogin_DoesNotLogRawUsername` bricht ab und zeigt
  `username=leaks@example.com` im Log. Zurueckgedreht, `git diff --stat` wieder exakt 15
  Einfuegungen/1 Loeschung wie vor der Probe, Test erneut gruen. (2) `app_password.go`:
  `fingerprintUsername(username)` durch `username` ersetzt ->
  `TestAppPasswordService_Validate_InvalidUsernameFormat_DoesNotLogRawEmail` (DB-Test, neu in
  `app_password_db_test.go`) bricht ab und zeigt dieselbe E-Mail im Log. Zurueckgedreht,
  `git diff --stat` wieder exakt 13 Einfuegungen/1 Loeschung wie vor der Probe, Test erneut
  gruen.
- verify vorgaenger: sauber. `48ba72ff` (Iteration 80, fix-gateway-lexware-error-message-leakage)
  gegen alle acht Fehlerklassen geprueft (`git show --stat` + Volltextlesung von
  `lexware_grpc.go` und `route_lexware.go`) — alle vier RPCs bleiben ueber den generierten
  `bizv1`-Client erreichbar (kein direkter Service-Aufruf im Gateway), kein Stub, kein
  `.proto`-Change (`Success`/`ErrorMessage`-Felder bleiben bestehen, nur bei Erfolg genutzt),
  kein neuer/ersetzter `RequirePermission`-Guard, keine neue Tabelle, keine neue Route (die vier
  Handler existierten bereits), keine Wire-Shape-Aenderung. `mapLexwareError` ist derselbe
  bereits bestehende Mechanismus, den `DisconnectLexware`/`TriggerLexwareSync` schon vorher
  nutzten — kein zweiter Maskierungspfad.
- neue-units: keine.
- offen: keine

## Iteration 82 — fix-gateway-file-import-413-response-undocumented — done — 2026-08-22 11:17
- commit: feb725e1
- gebaut: Fund aus scan-gateway-openapi-response-code-drift (Iteration 46). Reine
  Spec-Korrektur in `backend/api/openapi.yaml`, kein Code-Verhalten geaendert. Zwei
  Datei-Upload-Routen lieferten bereits `http.StatusRequestEntityTooLarge` (413), das die
  Spec nicht kannte: (1) `POST /api/v1/finance/gobd-archive` — "400"-Beschreibung nannte
  faelschlich "file too large" (der Code liefert dafuer nachweislich 413,
  `route_biz_gobd_archive.go:61`), Text auf "empty file, unknown doc_type" gekuerzt und
  "413": "Document exceeds the 50 MiB limit" ergaenzt. (2) `POST /api/v1/finance/invoices/import`
  — "413": "Uploaded file exceeds the 10 MiB limit" ergaenzt, Stil und Formulierung von der
  strukturell identischen Route `POST /api/v1/finance/bank-statements/import`
  (openapi.yaml:9564-9569) uebernommen, aber ohne deren `content`/`schema`-Block, weil der
  gobd-archive/invoices-import-Block in der Spec durchgaengig ohne Content-Schema fuer
  Fehlerantworten dokumentiert ist (Stil folgt dem direkten Nachbarblock, nicht der
  banking-Route).
- gate: swagger-cli validate ok ("api/openapi.yaml is valid") | test ok
  (`go test ./internal/gateway/ -run TestOpenAPIRouteDrift`: 836 Routen gegen 838 dokumentierte
  Pfade, PASS; volles `./internal/gateway/` PASS, 0 SKIP, DATABASE_URL gegen kmuhub_app gesetzt)
  | build ok (`./internal/gateway/... ./cmd/gateway/...`) | vet ok | lint: n.a. (keine
  `.go`-Datei geaendert) | migration: n.a. | rls-smoke: n.a.
- coverage: n.a. (Bugfix, kein Coverage-Ziel — laut Unit-Kopf)
- mutations-probe: n.a. (reine YAML-Spec-Aenderung, kein Testverhalten zu mutieren; die
  Korrektheit wird durch `swagger-cli validate` und `TestOpenAPIRouteDrift` belegt, die beide
  vor der Aenderung bereits gruen waren und die inhaltliche Aussage der Response-Codes nicht
  pruefen koennen)
- verify vorgaenger: sauber. `16550c5e` (Iteration 81, fix-gateway-caldav-basic-auth-username-log-leakage)
  gegen alle acht Fehlerklassen geprueft (`git show --stat` + Volltextlesung von
  `route_caldav.go` und `app_password.go`) — reine Logging-Aenderung (Klartext-Username ->
  sha256-Fingerprint), kein gRPC-Layer-Bezug, kein Stub, kein `.proto`-Change, kein
  neuer/ersetzter `RequirePermission`-Guard, keine neue Tabelle, keine neue Route, keine
  Wire-Shape-Aenderung.
- neue-units: keine.
- offen: keine

## Iteration 83 — fix-gateway-quote-lifecycle-routes-missing-error-responses — done — 2026-08-22 11:20
- commit: 646fa714
- gebaut: Fund aus scan-gateway-openapi-response-code-drift (Iteration 46), Nachtrag zu
  cov-gateway-biz-quotes. Reine Spec-Korrektur in `backend/api/openapi.yaml`, kein
  Code-Verhalten geaendert. Die vier Angebots-Statusuebergangsrouten
  (`POST /api/v1/finance/quotes/{id}/{send,accept,reject,convert}`) dokumentierten fast keine
  ihrer tatsaechlichen Fehlerantworten, obwohl alle vier ueber `respondGRPCError` gehen und
  hinter Permission-Middleware liegen (`route_biz.go:99-102`: `quoteSend`, `quoteWrite` x2,
  `quoteConvert`). Ergaenzt je Route uebergangsspezifisch:
  - send: 401/403 neu, 404 blieb, 409 neu mit "Quote is not in draft status"
    (`quote.ErrQuoteNotDraft`, service.go:389 — Send prueft `QuoteStatusDraft`).
  - accept/reject: 401/403/404/409 komplett neu, 409 mit "Quote is not in sent status"
    (`quote.ErrQuoteNotSent`, service.go:437/459 — beide pruefen `QuoteStatusSent`).
  - convert: 400/401/403/404/409 komplett neu. 400 zusaetzlich zum verlangten Umfang ergaenzt,
    weil `HandleConvertQuoteToInvoice` als einzige der vier Routen einen validierten Request-Body
    hat (`decodeAndValidate[convertQuoteRequest]`, `helpers.go:225-239` liefert dort nachweislich
    400) — ohne den Eintrag waere die Spec fuer diese Route unvollstaendig geblieben, obwohl der
    Scan-Fund genau diese Luecke sucht. 409 mit "Quote must be accepted before conversion to
    invoice" (`invoice.ErrQuoteNotAccepted`, invoice/service.go:779 — `CreateFromQuote` prueft
    `QuoteStatusAccepted`, NICHT `quote.ErrQuoteNotDraft/NotSent` wie die drei anderen Routen;
    das ist der in den notes verlangte eigene, vom Quote-Paket-Fehler verschiedene Fall).
  Stilvorlage fuer den inline-409 (Description + `ErrorResponse`-Schema statt generischem
  `Conflict`-Ref) ist die strukturell aehnliche Route bei openapi.yaml:8467-8480
  (Meeting-Serien-Konflikt), nicht die urspruenglich im Unit-Text genannte Zeilennummer fuer
  `admin/license` (die Spec ist seit dem Scan gewachsen, die Zeile dort traegt inzwischen kein
  409 mehr — per Grep neu verifiziert statt der alten Zeilennummer vertraut).
- gate: swagger-cli validate ok ("api/openapi.yaml is valid") | build ok
  (`./internal/gateway/... ./cmd/gateway/...`) | vet ok | lint ok (0 issues) | test ok
  (`go test ./internal/gateway/ -run TestOpenAPIRouteDrift`: 836 Routen gegen 838 dokumentierte
  Pfade, PASS; volles `./internal/gateway/` PASS, 0 SKIP, DATABASE_URL gegen kmuhub_app gesetzt)
  | migration: n.a. | rls-smoke: n.a. (keine Tabelle/Policy angefasst)
- coverage: n.a. (Bugfix/Spec-Korrektur, kein Coverage-Ziel laut Unit-Kopf)
- mutations-probe: n.a. (reine YAML-Spec-Aenderung, kein Go-Testverhalten zu mutieren;
  Korrektheit der Zuordnung ist durch Volltextlesung von quote/service.go und
  invoice/service.go gegen `biz_grpc_errormap_settings_quotes_test.go` belegt, nicht durch
  einen brechbaren Test)
- verify vorgaenger: sauber. `feb725e1` (Iteration 82, fix-gateway-file-import-413-response-undocumented)
  geprueft (`git show --stat`) — reine `.yaml`-Aenderung, kein Go-Code betroffen, damit fuer
  alle acht Fehlerklassen automatisch sauber (kein gRPC-Layer-Bezug, kein Stub, kein
  `.proto`-Change, kein Guard, keine Tabelle, keine neue Route, keine Wire-Shape-Aenderung).
- neue-units: keine.
- offen: keine

## Iteration 84 — fix-erasure-handlers-not-idempotent-on-second-run — done — 2026-08-22 11:24
- commit: 9060cef0
- gebaut: Alle fuenf nicht doppellauf-festen Erasure-Handler in
  `backend/internal/security/gdpr/erasure.go` gefixt. Zwei Guard-Muster, pro Handler einzeln
  entschieden statt pauschal:
  - Zustands-Guard, wo der erste Lauf einen abfragbaren Zustand hinterlaesst. Auth:
    `WHERE id = $1 AND last_name IS DISTINCT FROM $3` (neue Konstante `anonymizedLastName`;
    der Handler laeuft komplett in EINER Transaktion, also ist das Sentinel-`last_name`
    aequivalent zum vollstaendig anonymisierten Zustand). CRM: `WHERE assigned_to = $1 OR
    (created_by = $1 AND description IS NOT NULL)`.
  - Label-Muster-Guard fuer Freitext, dessen Label pro Lauf wechselt (neue Konstante
    `anonymizedContentPattern = "[Geloeschter Benutzer #%]"`, Prefix ist durch
    `GetNextAnonymizedLabel` fixiert, nur der Zaehler variiert). Chat `messages.content`,
    Work `task_comments.content`, Calendar `calendar_events.title`.
  - Reporting-Korrektur, wo gar nichts geschrieben wird: Work `tasks` matcht nur noch
    `WHERE assignee_id = $1` — eine Aufgabe, die der Betroffene nur erstellt hat, wird von der
    Erasure nachweislich nicht veraendert (created_by ist NOT-NULL-FK auf den anonymisierten
    Sentinel) und gehoert deshalb nicht in `affected`; dieselbe Begruendung, mit der CRM
    contacts/companies bereits ausnimmt. Calendar: der Pre-Count fuer Termine ist von "alle
    Termine des Nutzers" auf "Termine, die der Personenkalender-DELETE mitkaskadiert"
    verengt, der Rest wird ueber `RowsAffected` des anonymisierenden UPDATE gezaehlt.
  Bewusst NICHT als Auth-Guard gewaehlt: `is_active` — ein aus anderen Gruenden deaktiviertes
  Konto traegt seine PII noch und muss anonymisiert werden. Bewusst NICHT als Chat-Guard
  gewaehlt: `is_deleted` — `message.PostgresRepository.Delete` (postgres_repository.go:151)
  setzt nur das Flag und laesst den Content stehen, ein Guard darauf haette bei jeder
  selbst-geloeschten Nachricht personenbezogenen Text zurueckgelassen. Beide Faelle sind
  jetzt mit einem eigenen Test gepinnt, damit ein spaeterer "Vereinfachungs"-Refactor sie
  nicht doch einbaut.
  Testseite: `erasure_idempotency_test.go` von Pin- auf Korrektheits-Tests umgeschrieben
  (Header + fuenf Testfunktionen umbenannt, alle `BUG:`-Assertions durch die korrekte
  Erwartung ersetzt), plus drei neue Tests: deaktiviertes-aber-nicht-anonymisiertes Konto,
  soft-geloeschte Nachricht, und der bis dahin unbelegte Calendar-Restfall (Termin auf einem
  FREMDEN Kalender — nur der Muster-Guard verhindert dort das Nachzaehlen).
  `erasure_work_test.go`: erwartete Zahl 6 -> 5 wegen der Reporting-Korrektur.
  Keine Migration, keine Proto-Aenderung, keine Route, kein Wire-Format-Bezug
  (`modulesAffected` behaelt seine Form, nur die Zahlen darin werden ehrlich).
- gate: build ok (`./internal/security/...`) | vet ok | lint ok (0 issues) | test ok
  (`./internal/security/...` alle 7 Pakete PASS; `-v` auf `./internal/security/gdpr/`:
  **0 SKIP, 0 FAIL**, 151 Top-Level-Tests + 16 Subtests real gelaufen, davon 39 Erasure-Tests gegen die echte DB mit
  `DATABASE_URL` auf `kmuhub_app`) | migration: n.a. | rls-smoke: n.a. (keine Tabelle,
  keine Policy angefasst — reine Query-Praedikate)
- coverage: `internal/security/gdpr` 71,5 % -> 71,5 % (eigene Messung mit
  `go tool cover -func`, vorher per `git stash` auf demselben Tree). Kein Delta: der Fix
  aendert Praedikate in bestehenden Statements, fuegt also keine neuen Zweige hinzu; die drei
  neuen Tests decken Pfade ab, die vorher schon von anderen Tests durchlaufen wurden.
  Abweichung zum `coverage_start:` der Unit (69,3 %) kommt aus Iteration 48+, die dasselbe
  Paket erweitert haben — es gilt meine Messung.
- mutations-probe: Calendar-Guard neutralisiert (`AND title NOT LIKE $3` -> `AND $3 = $3`,
  praedikat-wahr statt Guard) — `TestCalendarErasureHandler_ExecuteErasure_
  SecondRunSkipsAnonymizedForeignEvent` wurde rot ("expected [Geloeschter Benutzer #400],
  actual [...#401]" — genau der Label-Ueberschreib-Schaden, den die Unit beschreibt).
  Zurueckgedreht, `git diff --stat` danach wieder 67+/20- auf erasure.go, voller Paketlauf
  erneut gruen. Anmerkung: der erste Mutationsversuch (`... NOT LIKE $3 OR false`) blieb
  gruen und war ein Fehlgriff — `OR false` ist wirkungslos, weil AND staerker bindet; die
  Probe zaehlt erst ab dem zweiten, wirklich verhaltensaendernden Eingriff.
- verify vorgaenger: sauber. `646fa714` (Iteration 83) und `c2f2cee8` (Iteration 83
  SHA-Nachtrag) geprueft (`git show --stat`) — nur `backend/api/openapi.yaml` und
  Loop-Dateien, keine `.go`-Zeile, keine neue Route (nur Response-Codes an bestehenden
  Pfaden). Damit fuer alle acht Fehlerklassen automatisch sauber.
- neue-units: fix-work-erasure-preview-overcounts-vs-execute (ans Backlog-Ende gehaengt).
- offen: (1) Zwei Punkte aus dem Unit-Scope waren bereits vor dieser Iteration erledigt und
  brauchten keinen Fix mehr: CRM zaehlt contacts/companies laengst nicht mehr (der COUNT-Zweig
  ist in erasure.go durch einen erklaerenden Kommentar ersetzt), und der Calendar-Fall
  "eigener Personenkalender" war seit Lauf 10 doppellauf-fest. Der Scope-Text der Unit war an
  diesen Stellen veraltet.
  (2) Der `anonymizedContentPattern`-Guard ist eine Heuristik: eine Nachricht, die ein Nutzer
  woertlich als "[Geloeschter Benutzer #7]" verfasst hat, wuerde uebersprungen. Im Code als
  `lean:`-Marker mit Upgrade-Trigger vermerkt (dedizierte `anonymized_at`-Spalte pro Tabelle,
  falls das je relevant wird). Bewusst nicht gebaut — das waere eine Migration ueber vier
  Tabellen fuer einen Fall, der praktisch nicht vorkommt.
  (3) `PreviewErasure` blieb ueberall unangetastet: sie zeigt Bestand, `ExecuteErasure` zeigt
  jetzt Veraenderung. Fuer Work faellt die beiden schon beim ersten Lauf auseinander — das ist
  der Inhalt der neu angelegten Unit, nicht ein Nebeneffekt, der hier haette mitlaufen sollen.

## Iteration 85 — fix-contact-erasure-incomplete-set-null-table-scrub — done — 2026-08-22 11:35
- commit: 5ef54498
- gebaut: Beide Erasure-Pfade fuer Kontakte (Hard-Delete `contact.PostgresRepository.Delete`
  UND `consent.PostgresRepository.AnonymizeContact`, der RESTRICT-Anonymisierungspfad) scrubben
  jetzt dieselben drei abhaengigen Tabellen ueber eine gemeinsame neue Funktion
  `consent.ScrubDependentPII(ctx, tx, contactID, tenantID)` (`internal/crm/consent/scrub.go`,
  neue Datei):
  - `activities.description` (vorher nur im Anonymize-Pfad geleert, beim regulaeren Hard-Delete
    gar nicht — der COUNT/FK-Verweis verschwand per ON DELETE SET NULL, der Freitext blieb
    lesbar stehen)
  - `consent_records.ip_address`/`notes` (gleiches Muster, gleicher vorheriger Zustand)
  - NEU fuer beide Pfade: `tickets.requester_name`/`requester_email`. Kein bestehender Pfad
    kannte diese Spalten ueberhaupt (sie werden bei Ticket-Erstellung selbst eingetragen, nicht
    aus `contacts` gejoint) — nach einer Kontakt-Loeschung/Anonymisierung blieb Name und
    E-Mail-Adresse externer Requester auf jedem ihrer Tickets vollstaendig lesbar. Fix per
    `CASE WHEN requester_is_external THEN <Platzhalter> ELSE NULL END`: externe Requester
    (kein `requester_id`, CHECK `chk_tickets_requester_identity` aus Migration 000291 verlangt
    dort zwingend eine nicht-leere `requester_email`) bekommen einen neuen Platzhalter
    (`consent.AnonymizedRequesterEmail = "geloescht@deleted.invalid"`, Name
    `AnonymizedFirstName+" "+AnonymizedLastName` = "Gelöschte Person", beide im gleichen Stil wie
    der bestehende `contacts`-Anonymisierungs-Sentinel); interne Requester (identisch mit dem
    User-Datensatz, diese beiden Spalten sind dort nur ungenutzter Fallback) werden schlicht auf
    NULL gesetzt.
  `contact.PostgresRepository.Delete` war vorher ein einzelnes `Exec` ohne Transaktion — jetzt
  Begin/Scrub/Delete/Commit in einer Tx, damit der Scrub nicht ohne die nachfolgende Loeschung
  stehen bleiben kann. `consent.PostgresRepository.AnonymizeContact` ruft dieselbe Funktion auf
  und ersetzt damit zwei vorher inline duplizierte Statements (Notes im Unit-Kopf hatten das
  explizit verlangt: beide Pfade sollen dieselbe SQL-Logik teilen, nicht zweimal schreiben).
  Import-Richtung `contact` -> `consent` ist neu, aber unkritisch: beide Pakete laufen im selben
  Service (`cmd/crm/main.go`, gleicher Pool), `consent` importiert nirgends `contact` zurueck,
  kein Zyklus.
  Bewusst NICHT angefasst, mit Begruendung jetzt direkt im Doc-Comment von `ScrubDependentPII`
  (vorher nur im Backlog-Unit-Kopf): `finance_invoices` (10 Jahre GoBD-Aufbewahrungspflicht
  §147 Abs. 3 AO, gesendete Rechnungen zusaetzlich `locked_at`-immutable) und
  `deals.notes`/`meetings.title/description/agenda` (Freitext, der den Kontakt nennen KANN aber
  nicht muss — automatisches Scrubben waere riskanter als das Restrisiko).
  `contract_parties.external_name` und `contacts.referred_by_contact_id` waren laut Scope-Notiz
  bereits geprueft und sauber — nicht angefasst, keine neue Erkenntnis dazu.
- gate: build ok (`./internal/crm/... ./cmd/crm/...`) | vet ok | lint ok (0 issues) | test ok
  (`./internal/crm/contact/` 117 Tests PASS inkl. der 2 neuen; `./internal/crm/consent/` 31 Tests
  PASS inkl. der 1 neuen; `./internal/helpdesk/` unveraendert gruen; 0 SKIP in allen drei,
  DATABASE_URL gegen kmuhub_app gesetzt. Musste `-p 1` erzwingen — der volle Lauf ueber
  `./internal/crm/... ./internal/helpdesk/... ./internal/security/...` parallel sprengt den
  lokalen Postgres `max_connections` ("too many clients already" / "remaining connection slots
  reserved for SUPERUSER"), rein umgebungsbedingt, nicht durch diese Aenderung verursacht — mit
  `-p 1` seriell liefen alle 20 betroffenen Pakete gruen durch) | migration: n.a. (keine neue
  Tabelle, keine neue Spalte) | rls-smoke: implizit mitgeprueft, kein dediziertes neues Kommando
  noetig — `TestContactWrites_LandInCallerTenant`s bestehender `Delete(ctxOther, ...)`-Fall und
  `TestContactExistsAndAnonymize_ScopedToCallerTenant`s `AnonymizeContact(ctxOther, ...)`-Fall
  laufen jetzt beide durch den neuen/erweiterten Code und blieben gruen (kein Cross-Tenant-Schaden
  durch RLS auf `activities`/`consent_records`/`tickets`), beide Tests unveraendert im selben
  Lauf bestanden
- coverage: `internal/crm/contact` 81,4 % -> 81,2 % (eigene Messung mit `go tool cover -func`,
  vorher per `git stash -u` auf demselben Tree). Leicht negativ trotz zwei neuer Tests: die neue
  `Delete`-Transaktion fuegt Fehlerpfade hinzu (`tx.Begin`/`tx.Commit`-Fehler), die ohne einen
  DB-Verbindungsabbruch nicht erreichbar sind — mehr neue Statements als neu abgedeckte.
  `internal/crm/consent` 63,1 % -> 64,0 % (gleiche Methode). Abweichung zum `coverage_start:`
  der Unit (80,4 % bzw. n/a) kommt aus spaeteren Iterationen, die dieselben Pakete seit
  Iteration 48 weiter angefasst haben — es gilt meine Messung. `ScrubDependentPII` selbst: 80,0 %
  Funktions-Coverage, `Delete` 66,7 % (die zwei unerreichbaren `tx.Begin`/`tx.Commit`-Fehlerzweige
  sind der Rest), `AnonymizeContact` 76,9 %.
- mutations-probe: in `scrub.go` die `requester_name`-CASE-Bedingung entfernt
  (`requester_name = CASE WHEN requester_is_external THEN $3 ELSE NULL END` ->
  `requester_name = $3`, also immer der Platzhalter statt nur fuer externe Requester) —
  `TestAnonymizeContact_ScrubsExternalTicketRequesterIdentity` wurde sofort rot ("expected
  requester_name on an internal-requester ticket to be NULL ... got 0xc00048afa0", der interne
  Requester-Fall bekam faelschlich den externen Platzhalter). Zurueckgedreht, `git diff` danach
  wieder sauber auf den Soll-Stand, `./internal/crm/consent/` und `./internal/crm/contact/`
  erneut voll gruen.
- verify vorgaenger: sauber. `9060cef0` (Iteration 84, fix-erasure-handlers-not-idempotent...)
  per `git show --stat`/vollem Diff geprueft — reine Praedikat-Aenderungen (WHERE-Guards, neue
  Konstanten) in `internal/security/gdpr/erasure.go`, kein gRPC-Layer-Bezug (das Paket hat keine
  Handler), kein Stub, kein `.proto`, kein neuer `RequirePermission`-Guard, keine neue Tabelle/RLS,
  keine neue Route, keine Wire-Shape-Aenderung. Damit fuer alle acht Fehlerklassen automatisch
  sauber.
- neue-units: keine.
- offen: keine.

## Iteration 86 — fix-company-delete-merged-into-no-action-unchecked — done — 2026-08-22 11:48
- commit: -
- gebaut: nichts Neues. Die gezogene Unit ist ein unbereinigtes Duplikat von
  `fix-company-delete-merged-into-fk-crash` (bereits `status: done`, Commit `097d3737`,
  Migration `000321_companies_merged_into_set_null`). Beide Units beschreiben denselben Fehler
  (`companies.merged_into_id UUID REFERENCES companies(id)` ohne ON DELETE-Klausel, Default
  NO ACTION seit Migration 000059 — Loeschen einer gemergten Primary-Firma ohne eigene Kontakte
  lief bis zur DB durch und schlug dort mit unbehandeltem FK-Fehler fehl) und dieselbe
  Root-Cause-Entscheidung (SET NULL statt Service-Guard, analog `contacts.merged_into_id` aus
  Migration 000318 — kein Lesepfad braucht `merged_into_id`, um eine geloeschte Firma
  aufzuloesen, nur der Duplicate-Search-Filter `merged_into_id IS NULL`). Geprueft: `git show
  097d3737 --stat` zeigt Migration 000321 (up/down, SET NULL) und
  `postgres_repository_db_test.go` mit `TestRepository_Delete_MergedPrimaryCompany_DB`, die
  genau den hier verlangten `done_when`-Fall abdeckt (gemergte Primary-Firma ohne Kontakte
  loeschen -> Erfolg statt FK-Crash, Duplikat behaelt seine eigene Existenz, `merged_into_id`
  wird NULL). `go test -count=1 -v ./internal/crm/company/...` gegen die aktuelle DB: 53 PASS /
  0 SKIP / 0 FAIL, `DATABASE_URL` gegen `kmuhub_app` gesetzt. Beide Units stammen offenkundig aus
  derselben Vorbereitungsrunde (die `notes:`-Zeile dieser Unit verweist bereits explizit auf die
  Schwester-Unit "fix-contact-delete-merged-into-no-action-unchecked" als Vorlage) und wurden nie
  dedupliziert, bevor `BACKLOG.yml` fuer Lauf 10 gestaged wurde. Unit auf `done` gesetzt statt auf
  `blocked` — es gibt keine offene Entscheidung, nur eine Redundanz in der Backlog-Datei selbst,
  und Weglassen des Duplikat-Hinweises im Journal haette die naechste Iteration denselben Umweg
  nochmal laufen lassen.
- gate: build: n.a. (kein Code geaendert) | vet: n.a. | lint: n.a. | test ok
  (`./internal/crm/company/...` 53 PASS/0 SKIP/0 FAIL, `-count=1`, DATABASE_URL gesetzt) |
  migration: n.a. (Migration 000321 ist bereits Teil des Baums, nichts Neues anzuwenden) |
  rls-smoke: n.a.
- coverage: n.a. (keine Verhaltensaenderung, kein neuer Code)
- mutations-probe: n.a.
- verify vorgaenger: sauber. `5ef54498` (Iteration 85) per `git show --stat` und vollem Diff
  geprueft (`consent/scrub.go` neu, `consent/postgres_repository.go` und
  `contact/postgres_repository.go` geaendert) — reines Repository-Layer, kein gRPC-Handler
  betroffen (Aufrufer bleiben `contact.PostgresRepository.Delete` bzw.
  `consent.PostgresRepository.AnonymizeContact`, beide unveraendert von aussen aufgerufen), kein
  Stub, kein `.proto`, kein neuer `RequirePermission`-Guard, keine neue Tabelle/RLS (alle
  UPDATEs bleiben `WHERE contact_id = $1 AND tenant_id = $2`-gescoped auf bestehende Tabellen),
  keine neue Route, keine Wire-Shape-Aenderung. Damit fuer alle acht Fehlerklassen automatisch
  sauber.
- neue-units: keine.
- offen: Die Schwester-Situation koennte sich in Block C (Muster-Scan-Units) wiederholen — falls
  eine der noch offenen `scan-*`-Units auf denselben unverarbeiteten Vorbereitungs-Entwurf
  zurueckgeht, lohnt sich ein kurzer Blick, ob dort noch mehr unbereinigte Duplikate stecken.
  Kein konkreter Verdacht, nur eine Beobachtung fuer die naechste Iteration.

## Iteration 87 — fix-csv-formula-injection-remaining-exports — done — 2026-08-22 11:52
- commit: 31c6f439
- gebaut: `internal/csvutil/formula.go` (neues Paket, `NeutralizeFormulaCell` — dieselbe
  Regel wie bisher in `email/contact/export_service.go`: fuehrendes `=`,`+`,`-`,`@` bekommt ein
  vorangestelltes `'`). Die alte private Kopie in `export_service.go` ist geloescht, die Datei
  ruft jetzt `csvutil.NeutralizeFormulaCell`. Angewandt auf drei bestaetigte Stellen:
  `internal/formulare/service.go:buildCSV` (SubmittedBy und jeder stringifizierte Antwortwert
  aus dem JSONB-`answers`-Payload, NACH `fmt.Sprintf("%v", v)` wie in den Notes gefordert),
  `internal/security/audit/export.go:ExportCSV` (Target, Details, UserAgent — Action/TargetType/
  Result blieben unveraendert, das sind interne Enums/IDs, keine Freitextfelder),
  `internal/biz/dunning/service_gobd.go:buildGoBDCSV` (CustomerName; BookingText geprueft und
  NICHT angefasst — `biz_grpc.go:2533` baut ihn als `fmt.Sprintf("Rechnung %s %s", number,
  customer)`, das erste Zeichen ist immer das literale `R`, also nie vom Kunden auf Position 0
  kontrollierbar).
- GoBD-Entscheidung (Pflichtzeile laut Unit-Notes): Apostroph-Praefix statt Anfuehrungszeichen-
  Zwang gewaehlt. Begruendung: das GoBD-CSV wird in diesem Repo an keiner Stelle re-importiert
  (Grep nach einem Parser fuer `buildGoBDCSV`-Output ergab nichts — der Zweck ist Pruefer-Export
  fuer IDEA/Excel-Review, keine Round-Trip-Buchhaltungsschnittstelle), es gibt also keinen
  eigenen Test, der eine Apostroph-Toleranz belegen oder widerlegen koennte. Eine unneutralisierte
  Formel in einem Export, der routinemaessig in Excel geoeffnet wird (Betriebspruefung), ist das
  groessere Risiko als ein optisches fuehrendes Zeichen in einer Textzelle — dieselbe Abwaegung,
  die Iteration 53 fuer den Kontaktexport bereits getroffen hat. Non-Goal ausdruecklich beachtet:
  der Datensatz selbst (GoBDExportRow.CustomerName) bleibt unveraendert, nur die CSV-Zelle in
  `buildGoBDCSV` aendert sich.
- gate: build ok (`go build -p 2` ueber alle fuenf betroffenen Pakete plus `internal/gateway`,
  keine Route angefasst) | vet ok | lint ok (`golangci-lint run` ueber alle fuenf Pakete: 0
  issues) | test ok — `go test -count=1 -v` ueber `internal/csvutil`, `internal/email/contact`,
  `internal/formulare`, `internal/security/audit`, `internal/biz/dunning`: alle vier bestehenden
  Pakete weiterhin gruen, 0 SKIP (`grep -c "^--- SKIP"` = 0, `DATABASE_URL` gegen `kmuhub_app`
  gesetzt) | migration: n.a. (keine neue Tabelle/Spalte) | rls-smoke: n.a. (keine Tabelle/Policy
  angefasst, reine Praesentationsschicht in CSV-Export-Funktionen)
- coverage: `internal/csvutil` n.a. -> 100,0 % (neues Paket, 5 Zeilen, vollstaendig getestet)
  `internal/email/contact` 80,4 % -> 80,2 % (Datei nur intern verschoben, minimal negativ durch
  Paketgroessen-Rundung, kein Verhalten geaendert) `internal/formulare` 53,5 % -> 53,6 %
  `internal/security/audit` 80,1 % -> 80,1 % (unveraendert trotz neuer Testdatei — das Paket ist
  bereits sehr gut abgedeckt, `ExportCSV` selbst war zuvor ungetestet und ist es jetzt nicht mehr,
  das Delta verschwindet aber in der Rundung auf eine Nachkommastelle) `internal/biz/dunning`
  61,8 % -> 61,8 % (gleicher Effekt: `buildGoBDCSV` war schon indirekt ueber die bestehenden
  GenerateGoBDExport-Tests abgedeckt, die zwei neuen Tests pruefen jetzt zusaetzlich das
  Neutralisierungsverhalten, ohne neue Zeilen zu durchlaufen, die vorher nie liefen). Alle vier
  Werte per `git stash -u` auf demselben Tree vs. danach gemessen, `coverage_start:` der Unit war
  "n.a." (Sicherheits-Fix ohne Coverage-Ziel), insofern nur zur Dokumentation.
- mutations-probe: in `csvutil/formula.go` `case '=', '+', '-', '@':` zu `case '=', '+', '-':`
  verkuerzt (das `@`-Praefix entfernt) — `TestNeutralizeFormulaCell` in
  `internal/csvutil/formula_test.go` wurde sofort rot (`NeutralizeFormulaCell("@SUM(1,2)") =
  "@SUM(1,2)", want "'@SUM(1,2)"`). Zurueckgedreht, `git diff internal/csvutil/formula.go` danach
  leer (Datei ist neu/untracked, Soll-Inhalt manuell gegenkontrolliert), alle fuenf Pakete erneut
  gruen.
- verify vorgaenger: Iteration 86 hatte `commit: -` (reines Dedup-Erkennen ohne Codeaenderung,
  siehe deren Journal-Eintrag). Der letzte echte Code-Commit ist `5ef54498` (Iteration 85) und
  wurde bereits in Iteration 86 gegen alle acht Fehlerklassen geprueft ("sauber"). Der
  dazwischenliegende `2571436c` ist ein reiner Journal/Backlog-Doku-Commit (`git show --stat`:
  nur `BACKLOG.yml` und `JOURNAL.md`), fuer die acht Fehlerklassen nicht relevant.
- neue-units: fix-csv-xlsx-formula-injection-remaining-writers (deckt `buildXLSX` in
  `internal/formulare/service.go` — dieselbe Antwort-Map, aber ueber excelize statt CSV
  geschrieben, vermutlich ein anderer Neutralisierungsmechanismus noetig — sowie die drei laut
  Scope dieser Unit ausdruecklich ungeprueften `csv.NewWriter`-Stellen in
  `internal/server/inventar_grpc.go`, `vermietung_grpc.go`, `fuhrpark_grpc.go`).
- offen: keine.

## Iteration 88 — fix-work-task-custom-field-foreign-tenant-writable — done — 2026-08-22 12:01
- commit: 9273da16
- gebaut: `SetCustomFieldValues` (`internal/work/task/postgres_repository.go:586`) schreibt nicht
  mehr blind per `INSERT ... VALUES`, sondern per `INSERT ... SELECT ... WHERE EXISTS (SELECT 1
  FROM work_custom_field_definitions WHERE id = $3 AND tenant_id = $2)` mit `ON CONFLICT` wie
  bisher; `RowsAffected() == 0` liefert das neue Sentinel `task.ErrCustomFieldNotFound`
  ("unknown custom field", `internal/work/task/errors.go`), das `mapWorkError`
  (`internal/server/work_grpc.go`) auf `codes.InvalidArgument` -> HTTP 400 abbildet. Fremde und
  gar nicht existierende field_ids liefern denselben Fehler mit demselben Text — das Oracle aus
  dem Unit-Scope ist damit zu. `api/openapi.yaml`: PUT `/api/v1/tasks/{id}/custom-fields`
  dokumentiert jetzt 400 und 401 (Route existierte, keine neue Route).
- Fehlercode-Wahl (gegen den FE-Typ, nicht geraten): `useSetTaskCustomFields`
  (`desktop/src/renderer/src/api/hooks/useTasks.ts:447`) wirft den Fehler unveraendert weiter und
  verzweigt nicht nach Statuscode. 400 ist damit korrekt und unschaedlich: der Client hat eine
  field_id geschickt, die es fuer ihn nicht gibt — 404 wuerde die Task-Ressource meinen, die es
  sehr wohl gibt.
- Befund aus der ersten Mutations-Probe (gehoert ins Protokoll, weil er das Bild praezisiert):
  in der normalen Tenant-Session traegt das explizite `tenant_id = $2` nichts bei — RLS auf
  `work_custom_field_definitions` (`tenant_isolation`, Migration 000146) filtert die
  EXISTS-Unterabfrage bereits, die fremde Definition ist dort unsichtbar. Load-bearing ist das
  Praedikat erst im Systemkontext (`is_system_context()` laesst jede Zeile durch). Der Test wurde
  deshalb um einen Systemkontext-Aufruf erweitert, sonst waere die Probe nicht diskriminierend
  gewesen. Der urspruengliche Bug bestand trotzdem: vorher gab es GAR KEINE Unterabfrage, nur den
  FK-Check, und der laeuft immer im Systemkontext.
- gate: build ok (`go build -p 2` ueber `internal/work/...`, `internal/server/...`,
  `internal/gateway/...`, `cmd/work/...`, `cmd/gateway/...`) | vet ok | lint ok (`golangci-lint
  run` ueber work/task, server, gateway: 0 issues) | test ok — `go test -count=1 -v
  ./internal/work/task/`: 39 PASS, 0 FAIL, **0 SKIP** (`DATABASE_URL` gegen `kmuhub_app`, also
  liefen die DB-Tests real); `./internal/work/...` und `./internal/server/` gruen;
  `./internal/gateway/` gruen inkl. `TestOpenAPIRouteDrift`; `swagger-cli validate
  api/openapi.yaml` = valid | migration: n.a. (keine neue Tabelle/Spalte, keine Migration —
  reiner Query-Guard auf bestehendem Schema) | rls-smoke: n.a. keine Tabelle oder Policy
  angefasst; die Tenant-Isolation wird stattdessen direkt im neuen DB-Test geprueft (eigener
  Tenant schreibt, fremder Tenant wird abgelehnt, Waisenzeile im Systemkontext nachgezaehlt = 0)
- coverage: `internal/work/task` 67,5 % -> 67,4 % (per `git stash -u` auf demselben Tree vor/nach
  gemessen). Der Wert sinkt um eine Rundungsstelle, obwohl beide neuen Statements abgedeckt sind:
  `SetCustomFieldValues` selbst geht von 5 auf 8 abgedeckte von jetzt 10 Statements (80,0 %), die
  zwei weiterhin unabgedeckten Fehlerpfade (`tx.Begin`-Fehler, `json.Marshal`-Fehler) waren vorher
  ebenfalls unabgedeckt und wiegen im groesseren Nenner minimal staerker. `coverage_start:` der
  Unit war "n.a. (Validierungs-Fix, kein Coverage-Ziel)" — passt.
- mutations-probe: zwei Proben, beide zurueckgedreht, `git diff` danach exakt der Soll-Diff
  (17+/3-). (A) `AND tenant_id = $2::uuid` aus der EXISTS-Unterabfrage entfernt ->
  `TestSetCustomFieldValues_RejectsForeignAndUnknownDefinitions` rot
  ("in system context with a foreign-tenant definition = <nil>, want ErrCustomFieldNotFound").
  (B) den `RowsAffected() == 0`-Zweig durch `_ = tag` ersetzt -> derselbe Test rot an der ersten
  Assertion ("with a foreign-tenant definition = <nil>"). Beide Haelften des Guards sind damit
  einzeln belegt.
- verify vorgaenger: sauber. `31c6f439` (Iteration 87) gegen alle acht Fehlerklassen geprueft:
  keine `.proto`-Aenderung, keine Route, kein `RequirePermission`, keine Tabelle, keine
  Migration, kein Stub — die Aenderung ist reine Praesentationsschicht in fuenf CSV-Schreibpfaden
  plus dem neuen Paket `internal/csvutil`, Wire-Shape der Exporte unveraendert (nur Zellinhalt).
  Der dazwischenliegende `18471ea3` ist ein reiner Journal-Doku-Commit.
- neue-units: fix-work-task-custom-field-errors-swallowed-on-create-update — `CreateTask`
  (`work_grpc.go:606`) und `UpdateTask` (`:818`) rufen `SetCustomFieldValues` mit `_ =` auf und
  verwerfen jeden Fehler, auch das neue `ErrCustomFieldNotFound` und echte DB-Fehler. Der
  Sicherheitsteil ist ueber diese Pfade jetzt trotzdem dicht (die Zeile wird schlicht nicht mehr
  geschrieben), offen ist nur die Rueckmeldung an den Aufrufer — deshalb eigene Unit statt
  Nebenbei-Fix, sie braucht eine Entscheidung (Vorab-Validierung vs. slog-Warnung).
- offen: Bestandsdaten wurden NICHT geprueft — falls in `task_custom_field_values` bereits
  Waisenzeilen mit fremder `field_id` liegen (moeglich seit Migration 000320, praktisch
  unwahrscheinlich bei Single-Tenant-Daten), raeumt dieser Commit sie nicht auf; er verhindert nur
  neue. Ein Aufraeum-SELECT waere `SELECT COUNT(*) FROM task_custom_field_values tcfv JOIN
  work_custom_field_definitions d ON d.id = tcfv.field_id WHERE d.tenant_id <> tcfv.tenant_id`
  (im Systemkontext).
