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
