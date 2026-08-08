# Backend-Nachtloop — Journal (Lauf 6)

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
`archive/lauf-5/JOURNAL.md` (Lauf 5 haengt dort am Ende des Lauf-4-Journals).

---

## Lauf 6 — Ausgangslage (2026-08-07, vor der ersten Iteration)

- Branch `backend-loop`, auf `origin/main` gemergt (Fast-Forward ueber 3 Commits).
- Migrationskopf lokal wie produktiv **297**, `dirty=false`.
- Lokale DB laeuft und ist verifiziert: `docker-postgres-1`, Rolle `kmuhub_app` mit Passwort
  `app_dev`, die RLS-Integrationstests in `internal/crm/contact` laufen real durch (0 Skips).
  `DATABASE_URL` ist damit kein Alibi mehr — wer ohne sie testet, hat kein Gate.
- Backlog: **70 offene Units** in drei Bloecken — A (20, verifizierte Luecken), C (16, Coverage
  auf den kritischen Pfaden biz/crm), B (34, Coverage-Breite server/gateway). Dazu 7 `blocked`
  aus Lauf 5, die bewusst liegen bleiben.
- **Fenster verschoben.** Der Lauf am 07.08. (21:00–08:45) fiel aus, der Rechner war belegt.
  Echtes Fenster: **2026-08-08 14:30 bis 00:45** (`-UntilTime "00:45"`; danach nur noch Push und
  CI-Warten). Das sind rund zehn Stunden. Beim Median aus Lauf 5 — 35 Iterationen in 5 h 40,
  also ~10 min — deckt das etwa 60 der 70 Units. Was von Block B liegen bleibt, startet Lauf 7:
  eingeplant, kein Versaeumnis. Ein Loop, der um 02:00 leerlaeuft, waere der teurere Fehler.
- Coverage-Ausgangswerte, am 2026-08-07 lokal ohne `DATABASE_URL` gemessen (untere Schranke):
  `internal/server` 6,2 % · `internal/gateway` 24,1 %. Mit DB in CI: 8,1 / 27,2.

## Iteration 1 — a-dunning-send-non-fatal — done — 2026-08-08 14:45
- commit: a3bcfda0 (nachgetragen im Folge-Commit — ein Journal kann seine eigene SHA nicht enthalten)
- gebaut: Neues Sentinel `ErrCompanySettingsMissing` in `internal/biz/dunning/errors.go`;
  `emailNotice` wrappt es beim `settings == nil`-Zweig statt eines anonymen Fehlers, und
  `sendAndNotify` prueft per `errors.Is` — fehlende Firmen-Stammdaten sind jetzt non-fatal
  (Status flippt auf `sent`, `sent_at` gesetzt, Miss als `slog.Error` mit `dunning_id` und
  `tenant_id`), waehrend jeder echte Zustell-, PDF- oder Settings-Ladefehler weiterhin
  fail-closed bleibt und den Datensatz in `draft` haelt.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (dunning 53/53, 0 Skips;
  `./internal/gateway/` ok inkl. TestOpenAPIRouteDrift) | migration n.a. | rls-smoke n.a.
- mutations-probe: `if !errors.Is(sendErr, ErrCompanySettingsMissing)` zu
  `if !errors.Is(sendErr, ErrDunningNotFound)` gedreht (macht den Settings-Fall wieder fatal).
  Ergebnis: genau `TestSend_WithNoticeMailer_MissingCompanySettings_StillSends` wurde rot, die
  drei Nachbartests blieben gruen. Zurueckgedreht, Endgate danach erneut gelaufen und gruen.
  Anmerkung fuer kuenftige Proben: die naive Mutation `if true {` faellt nicht als Test-Rot
  auf, sondern als Compile-Fehler (`"errors" imported and not used`) — dann ist die Probe
  wertlos. Immer eine Mutation waehlen, die kompiliert.
- db-tests: 0 real gelaufene DB-Tests, 0 Skips — `internal/biz/dunning` ist reines
  Mock-Testing, es gibt in dem Paket keinen `SkipIfNoDB`-Pfad. `DATABASE_URL` war gesetzt.
- verify vorgaenger: n.a. — erste Iteration des Laufs, letzter Commit `535ea306` ist
  docs/chore am Loop-Verzeichnis. `git merge origin/main` war "Already up to date".
- offen: Bewusste Abweichung vom `scope`-Wortlaut, gedeckt durch `notes`: entschaerft ist NUR
  `settings == nil`, nicht jeder Fehler des konfigurierten Mailers. Ein SMTP-Refuse laesst die
  Mahnung weiter in `draft` (GoBD: "sent" = zugestellt). Zweiter Punkt fuer Luke: bei
  fehlenden Stammdaten gilt die Mahnung jetzt als `sent`, obwohl keine Mail rausging — die
  Fachlage will das so (der Nil-Mailer-Zweig macht es seit je genauso), aber die UI hat dafuer
  keinen Hinweis. Falls gewuenscht, gehoert eine Warnung in die Send-Response, das waere eine
  eigene Unit inkl. Proto-Feld.

## Iteration 2 — a-vertraege-contract-events — done — 2026-08-08 15:40
- commit: 04278996
- gebaut: Migration **000298** legt `contract_events` an (`tenant_id UUID NOT NULL` +
  `contract_id` -> contracts ON DELETE CASCADE, `action TEXT`, `user_id UUID NULL` ->
  users ON DELETE SET NULL, `payload JSONB NOT NULL DEFAULT '{}'`, `created_at`), Index
  `(tenant_id, contract_id, created_at DESC)`, `CALL enable_tenant_rls('contract_events')`.
  Repository bekommt `CreateContractEvent` + `ListContractEvents` (append-only: kein UPDATE,
  kein DELETE). Der Service schreibt an Create (`created`), Update (`updated` mit der Liste
  der real geaenderten Feldnamen), Kuendigung (`terminated` + `from_status`, nur auf dem
  Uebergang), Signatur (`signed`) und beiden Party-Pfaden (`party_added`/`party_removed`).
  Neuer RPC `ListContractEvents` (Proto regeneriert) und
  `GET /api/v1/vertraege/contracts/{id}/events` hinter `RequirePermission("vertraege:contract",
  "read")` — bestehender Key, deshalb **keine** Seed-Migration noetig.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (vertraege 57/57, gateway ok inkl.
  TestOpenAPIRouteDrift, server ok, testutil-RLS-Standing-Guard PASS) | migration ok (298
  angewandt, `298/u contract_events`) | rls-smoke ok
- rls-smoke: zwei Vertraege + je ein Event in Tenant `0000…0001` und `aaaa…0001` gesetzt,
  dann als `kmuhub_app` (NOSUPERUSER NOBYPASSRLS): eigener Tenant -> 1, anderer Tenant -> 1,
  fremder Tenant (`…00ff`) -> 0. `relforcerowsecurity = true`, Policy `tenant_isolation` da.
  Anschliessend `DELETE FROM contracts WHERE contract_number IN (…)` — die Events sind per
  Cascade mitgegangen (`rest: 0`), die lokale DB ist wieder sauber.
- mutations-probe: `previousStatus != ContractStatusTerminated` zu
  `previousStatus != ContractStatusDraft` gedreht (macht aus jedem Speichern eines bereits
  gekuendigten Vertrags eine zweite Kuendigung). Ergebnis: genau
  `TestService_UpdateContract_TerminationIsRecordedOnTransitionOnly` wurde rot,
  `…RecordsTerminationWithPreviousStatus` und die uebrigen blieben gruen. Zurueckgedreht,
  Endgate danach erneut gelaufen und gruen.
- db-tests: 0 DB-Tests im Paket `internal/vertraege` (reines Mock-Testing, kein
  `SkipIfNoDB`-Pfad), **0 Skips** bei 57 Tests. `DATABASE_URL` war gesetzt; die DB-Seite ist
  ueber die angewandte Migration + den RLS-Smoke + den testutil-Standing-Guard abgedeckt.
- verify vorgaenger: sauber. `a3bcfda0` (dunning) fasst keine Route, keine Migration und kein
  `.proto` an, ruft nichts am gRPC-Client vorbei, hat keinen neuen `RequirePermission`-Guard
  und keinen Stub; die Sentinel-Unterscheidung per `errors.Is` ist praezise. `git merge
  origin/main` war "Already up to date".
- offen: Zwei Punkte fuer Luke.
  (1) **Signatur-Aenderung an `Repository.RemoveParty`**: gibt jetzt `(uuid.UUID, error)`
  zurueck (`DELETE … RETURNING contract_id`), damit das Party-Entfernen ueberhaupt einem
  Vertrag zugeordnet werden kann. Loeschen einer nicht existierenden Partei bleibt ein
  stiller No-Op (`uuid.Nil, nil`) und schreibt bewusst keinen Eintrag.
  (2) **`//nolint:staticcheck` in `vertraege_grpc.go`**: die Proto-Regeneration hat die
  `Deprecated:`-Kommentare zu `UploadDocument` erstmals in die `.pb.go` gebracht — die
  eingecheckte Datei war gegenueber der `.proto` veraltet. Der Linter meldet damit zu Recht
  SA1019 auf dem absichtlich behaltenen Unimplemented-Stub. Wenn der RPC ganz weg soll, ist
  das eine eigene Unit (analog zum toten Vermietung-RPC im Backlog).
  Kein FE-Konsument: `{items,total}` ist die im Backlog vorgegebene Form, das
  Vertraege-Modul im Frontend liest den Endpoint noch nicht.

## Iteration 3 — a-dokumente-version-download — done — 2026-08-08 15:55
- commit: fb6893d3
- gebaut: Neuer RPC `GetFileVersionDownloadURL` im document-Proto (Request:
  `file_id`+`version_id`, Response wie `GetFileDownloadURLResponse`). Neue
  Repository-Methode `GetVersionByID(ctx, fileID, versionID, tenantID)` filtert
  in SQL auf `id = $1 AND file_id = $2 AND tenant_id = $3` (nicht nur auf die
  version_id, wie in den `notes` gefordert). Neue Service-Methode
  `GetVersionDownloadURL` prueft zuerst `GetByID(fileID, tenantID)` (File
  gehoert dem Tenant, nicht geloescht), dann die Version scoped auf
  `(fileID, versionID, tenantID)`, liefert dann eine Presign-URL ueber
  `store.GetPresignedURL` (Wiederverwendung des chat/file-Presign-Musters,
  keine neue Storage-Logik). Filename/Content-Type kommen vom File-Datensatz
  (Version traegt beides nicht), file_size von der Version. Neue Route
  `GET /api/v1/documents/files/{id}/versions/{versionId}/download` hinter dem
  bestehenden `docDownload`-Guard (`documents:read` ODER
  `documents:file:download`, additiv, keine neue Guard-Kombination) — Handler
  geht ueber `documentv1.DocumentServiceClient`, keine direkte Service-Instanz.
  Pfad in `api/openapi.yaml` nachgezogen (Parameter-Stil vom
  `wiki/articles/{id}/versions/{versionId}/restore`-Pfad uebernommen).
- gate: build ok | vet ok | lint ok (0 issues) | test ok (document/file 
  inkl. 4 neuer Tests, server ok, gateway ok inkl. TestOpenAPIRouteDrift) |
  migration n.a. (keine neue Tabelle) | rls-smoke n.a. (keine Tabelle/Policy
  angefasst)
- mutations-probe: `if f.IsDeleted { return ..., ErrFileDeleted }` in
  `GetVersionDownloadURL` entfernt. Ergebnis: genau
  `TestGetVersionDownloadURL_DeletedFile` wurde rot, die drei
  Nachbartests (`_Success`, `_ForeignFile`, `_UnknownVersion`) blieben gruen.
  Zurueckgedreht, Endgate danach erneut gelaufen und gruen. Eine zweite Probe
  (tenantID beim `GetVersionByID`-Aufruf durch `uuid.Nil` ersetzt) wurde NICHT
  rot — die Mock-Repo-Isolation in den Tests laeuft ueber die
  `map[fileID][]version`-Struktur, nicht ueber den tenantID-Parameter selbst,
  weil im Mock ohnehin nie zwei Tenants dieselbe file_id teilen. Die
  tenant_id-Bedingung in der SQL-Query ist damit nur durch die Postgres-Seite
  (Migrationskopf 297, RLS auf `document_files`/`document_file_versions`)
  belegt, nicht durch einen Unit-Test — vermerkt statt verschwiegen.
- db-tests: 0 DB-Integrationstests im Paket `internal/document/file` (reines
  Mock-Testing, kein `SkipIfNoDB`-Pfad), **0 Skips** bei allen gelaufenen
  Tests. `DATABASE_URL` war gesetzt.
- verify vorgaenger: sauber. `04278996` (vertraege-contract-events) geht im
  Handler ueber `vr.getClient()` (gRPC-Client), Migration 298 hat
  `tenant_id UUID NOT NULL` + `CALL enable_tenant_rls(...)`, up und down
  beide gefuellt, `.proto` und `.pb.go`/`_grpc.pb.go` beide im Commit, kein
  neuer `RequirePermission`-Guard (nutzt den bestehenden
  `vertraege:contract`/`read`-Key), Pfad in `openapi.yaml` vorhanden.
- offen: Zwei Punkte fuer Luke.
  (1) Die zweite Mutations-Probe (tenantID-Parameter) hat keine rote Zeile
  erzeugt, siehe oben — der Unit-Test-Beweis fuer die
  Tenant-Scopung dieses spezifischen Aufrufs fehlt, die Verteidigung steht
  nur in der SQL-Query und im DB-Schema. Waere eine eigene, kleinere Unit
  wert, wenn das lueckenlos belegt werden soll (z.B. ein Repository-Test mit
  echter DB, zwei Tenants, gleiche file_id kollidiert nicht).
  (2) Kein FE-Konsument geprueft — der Download-Knopf in der Versionshistorie
  (`FileContextMenu.tsx`) ist nicht Teil dieser Unit; das FE muesste den
  neuen Endpoint noch verdrahten.

## Iteration 4 — a-schichten-minor-compliance — done — 2026-08-08 17:20
- commit: 7493cf0f
- gebaut: Migration 000299 ergaenzt `is_minor BOOLEAN NOT NULL DEFAULT FALSE`
  an `hr_employee_profiles` (up + down gefuellt, Partial-Index
  `(tenant_id, is_minor) WHERE is_minor` fuer den Lookup). Keine neue Tabelle,
  also keine neue RLS-Policy noetig — die Tabelle steht seit Migration 123
  unter RLS. Wire durchgereicht: `models.EmployeeProfile.IsMinor`, beide
  Scan-Helfer, alle drei SELECT-Listen, INSERT und UPDATE im
  `PostgresEmployeeRepo`; `CreateEmployeeInput.IsMinor bool` und
  `UpdateEmployeeInput.IsMinor *bool` (Pointer = "nil heisst keine
  Aenderung", wie die uebrigen Update-Felder), zusaetzlich in
  `hasRestrictedFields` aufgenommen, damit ein Mitarbeiter das Flag nicht
  per Self-Service an sich selbst loeschen kann. Proto: `is_minor` an
  `EmployeeProfile` (26), `CreateEmployeeReq` (16) und als `optional bool`
  an `UpdateEmployeeReq` (10) — `optional`, weil ein blankes `bool` "auf
  false setzen" nicht von "nicht Teil dieses Updates" trennen kann;
  `hr.pb.go` im selben Commit regeneriert (`hr_grpc.pb.go` unveraendert,
  keine Version-Drift). Gateway: `is_minor` in `createEmployeeHTTPReq` (bool)
  und `updateEmployeeHTTPReq` (`*bool`).
  Compliance: die bestehende Pruefung wurde ERWEITERT, keine zweite Engine
  daneben. `CheckArbzgCompliance` ruft nach `validateRestPeriod` neu
  `validateMinorProtection` auf; die Verstoesse kommen als derselbe
  `(compliant bool, reason string)` zurueck wie bisher, das FE lernt nichts
  Neues. Drei Sentinels in `errors.go` (`ErrJArbSchGNightWork`,
  `…DailyHours`, `…Weekend`), alle drei auf `FailedPrecondition` gemappt wie
  `ErrArbzgViolation`. Regeln in der reinen, DB-freien Funktion
  `checkMinorShiftLimits`: §16/§17 kein Sa/So (beide Enden geprueft, damit
  Freitagnacht in den Samstag mitgefangen wird), §14 Fenster 06:00–20:00
  (beide Grenzen am Starttag verankert, damit eine Schicht ueber Mitternacht
  hinter der Obergrenze landet statt wie ein frueher Morgen auszusehen),
  §8 hoechstens acht Stunden. Zeiten werden vorher nach `Europe/Berlin`
  konvertiert (Muster von `ApplyTemplate` uebernommen) — der UTC-Stundenwert
  wuerde genau die Nachtschichten durchlassen, um die es geht.
  Der Guard sitzt zusaetzlich in `AssignEmployee` als Guard 3: eine Schicht,
  die ein Minderjaehriger nicht arbeiten darf, soll gar nicht erst zur
  Zuweisung werden. Fuer alle ohne Flag aendert sich nichts (Default false).
  Der Lookup `IsMinorEmployee` liest `hr_employee_profiles` direkt aus dem
  schichten-Repo — dasselbe Muster wie
  `PostgresEmployeeRepo.CountOtherActiveRoleAdmins`, das eine auth-Tabelle
  liest ("there is no service-to-service gRPC in this repository", Kommentar
  dort). Tenant-Praedikat explizit, RLS greift zusaetzlich. Kein Profil =
  kein Verstoss (fehlende HR-Daten sind keine Aussage ueber das Alter).
- gate: build ok | vet ok | lint ok (0 issues) | test ok (schichten, biz/hr
  komplett, server, gateway inkl. TestOpenAPIRouteDrift, testutil inkl.
  `TestAllPublicTablesHaveRLSOrAreAllowlisted`) | migration ok (299 lokal
  angewendet, Kopf jetzt 299) | openapi `swagger-cli validate` ok
- mutations-probe: zwei Stueck, beide aussagekraeftig.
  (1) `end.After(latest)` aus der §14-Bedingung entfernt → genau
  `TestService_CheckArbzgCompliance_Minor_NightWork` und
  `TestService_AssignEmployee_MinorNightShiftRejected` wurden rot, die
  uebrigen blieben gruen. Genau die Zeile, genau die Tests.
  (2) `if !isMinor` zu `if isMinor` gedreht (Flag-Steuerung invertiert) →
  zehn Tests rot, darunter vier bestehende ArbZG-Tests und der neue
  `_Adult_NightShiftAllowed`. Das belegt beide Richtungen: das Flag schaltet
  die Regeln ein UND haelt sie von allen anderen fern. Beide Proben
  zurueckgedreht, Endgate danach erneut gelaufen und gruen.
- db-tests: `internal/biz/hr/employee` hat DB-Integrationstests
  (`integration_test.go`, `offboard_db_test.go`), die real gegen die lokale
  DB liefen — **0 Skips** in `hr/employee` und `schichten`. `DATABASE_URL`
  war gesetzt (`kmuhub_app`), Migration 299 vorher angewendet.
- verify vorgaenger: `fb6893d3` (a-dokumente-version-download) sauber. Handler
  geht ueber `documentv1.DocumentServiceClient`, kein direkter Service-Zugriff;
  `GetVersionByID` filtert `id AND file_id AND tenant_id`; kein neuer
  `RequirePermission`-Guard (nutzt den bestehenden `docDownload`); Pfad in
  `openapi.yaml` im selben Commit; keine neue Tabelle, also nichts an RLS.
- offen: Drei Punkte fuer Luke.
  (1) Die `(id = $2 OR user_id = $2)`-Bedingung im Lookup ist bewusst breit
  und mit `lean:` markiert: `shift_assignments.employee_id` hat keinen
  Foreign Key, das Desktop-Raster fuettert dort Profil-IDs
  (`useEmployees() → e.id`), waehrend derselbe Wert im Swap-Pfad `userId`
  heisst. Eine Spalte zu raten waere fail-open gewesen (unerkannte
  Minderjaehrige). Sobald die Spalte einen FK bekommt, gehoert die Bedingung
  auf eine Spalte verengt. Ein Test deckt diese Verzweigung NICHT ab — der
  Mock liefert den Bool direkt, belegt ist sie nur durch die SQL-Query.
  (2) Zeitzone ist fest `Europe/Berlin`, nicht `hr_company_settings.timezone`
  — schichten hat auf die HR-Settings keinen Zugriff, und ein zweiter
  Cross-Modul-Lookup pro Compliance-Pruefung waere teurer als der Nutzen bei
  aktuell einem Land. Mit `lean:` markiert.
  (3) Kein FE-Konsument: das Team-Modul hat keinen Schalter fuer `is_minor`,
  und `SchichtenPage.tsx` rechnet seine Warnungen weiterhin im Client
  (`type: 'max_hours' | 'rest_period' | 'consecutive_days'`), ohne den
  Compliance-Endpoint zu fragen. Das Backend kann jetzt mehr als das FE
  abholt — eine Frontend-Session waere der naechste Schritt.

## Iteration 5 — a-fuhrpark-vehicle-booking — done — 2026-08-08 17:35
- commit: 4383e7b2
- gebaut: Poolfahrzeug-Buchung mit Konfliktpruefung. Neue Tabelle
  `vehicle_bookings` (Migration 000300, `tenant_id NOT NULL` + FK auf
  `tenants`/`vehicles`/`users`, `CALL enable_tenant_rls`, CHECK auf
  `ends_at > starts_at` und den Status-Wortschatz von `machine_bookings`),
  vier RPCs (List/Create/Update/Delete), vier Routen unter
  `/api/v1/fuhrpark/bookings` hinter der neuen Ressource `fuhrpark:booking`.
  Die Ueberschneidungspruefung ist NICHT neu entworfen: sie ist das Muster aus
  `produktion/postgres_repository.go` — halboffenes Intervall
  `starts_at < $ends AND ends_at > $starts`, Konflikt-Check und INSERT in EINER
  Transaktion unter `pg_advisory_xact_lock(hashtext(tenant||':'||vehicle))`,
  damit zwei gleichzeitige Anfragen nicht beide ein freies Fahrzeug sehen.
  Konflikt = `ErrBookingConflict` → `codes.AlreadyExists` → 409 (belegt durch
  `helpers_test.go:19`), nicht 400. Zwei bewusste Abweichungen von
  `machine_bookings`: `vehicle_id` ist ein echter FK (die Tabelle `vehicles`
  existiert, `produktion` hat keine Maschinentabelle), und `user_id` ist
  Pflicht — eine Reservierung, fuer die niemand verantwortlich ist, laesst sich
  am Schluesselschrank nicht durchsetzen. Der INSERT geht als
  `INSERT ... SELECT FROM users WHERE u.id=$ AND u.tenant_id=$` (Muster von
  `CreateDriverLicense`): der blosse FK auf `users` wuerde eine Buchung fuer
  einen fremden Tenant-Nutzer durchlassen. `created_by` kommt aus
  `middleware.GetUserID(ctx)`, nicht aus dem Request-Body — sonst koennte ein
  Aufrufer im Namen eines anderen buchen. Beim Update wird nur
  nachgeprueft, solange die Buchung aktiv bleibt: Stornieren ist genau das, was
  einen Slot freigibt, und darf nie an einem Konflikt scheitern.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (fuhrpark, gateway inkl.
  `TestOpenAPIRouteDrift`, server, testutil inkl.
  `TestAllPublicTablesHaveRLSOrAreAllowlisted`) | migration ok (300 lokal
  angewendet, Kopf jetzt 300) | rls-smoke ok (Cross-Tenant-Lesen beider
  Richtungen als `kmuhub_app` = 0 Zeilen, im DB-Test) | openapi
  `swagger-cli validate` ok
- mutations-probe: zwei Stueck, beide aussagekraeftig.
  (1) `bookingOverlapPredicate` von `<`/`>` auf `<=`/`>=` gedreht (halboffen →
  geschlossen) → GENAU `TestBookingConflict_.../adjacent_is_accepted` wurde rot,
  `overlapping_is_rejected` blieb gruen. Das ist der Punkt der Unit: die
  Uebergabe um 12:00 bleibt moeglich.
  (2) `if isActiveBookingStatus(booking.Status)` im Update auf `if true`
  gesetzt → `TestService_UpdateVehicleBooking_ReleaseSkipsConflictCheck` wurde
  rot, die uebrigen blieben gruen. Beide Proben zurueckgedreht, Endgate danach
  erneut gelaufen und gruen.
- db-tests: `TestBookingConflict_HalfOpenAndTenantScoped` mit sechs Subtests
  lief real gegen die lokale DB — **0 Skips** im gesamten Paket `fuhrpark`
  (67 PASS, 0 SKIP). `DATABASE_URL` war auf `kmuhub_app` gesetzt, Migration 300
  vorher angewendet.
- verify vorgaenger: `7493cf0f` (a-schichten-minor-compliance) sauber. Beide
  Handler in `route_hr.go` gehen ueber `hrv1.HRServiceClient`, kein direkter
  Service-Zugriff; `is_minor` sitzt auf der bestehenden RLS-Tabelle
  `hr_employee_profiles`, also keine neue Policy noetig; Migration 000299 hat
  up UND down; kein neuer `RequirePermission`-Guard; openapi-Aenderung im
  selben Commit. Der Pointer-Typ im Update-Request ist korrekt gewaehlt
  (ausgelassenes Feld laesst das Flag stehen).
- offen: Drei Punkte fuer Luke.
  (1) Die Migration seedet `fuhrpark:booking:read/write` NUR fuer die Rolle
  `admin` (Muster von 000196). Andere Rollen sehen die Routen bis zur
  Zuweisung nicht — falls das Modul an eine Fuhrpark-Rolle soll, ist das eine
  eigene Seed-Migration.
  (2) Kein FE-Konsument. `FuhrparkPage.tsx` hat keinen Buchungskalender, und
  `capability-catalog.ts` kennt `fuhrpark:booking` noch nicht — das Backend
  kann jetzt mehr, als das Frontend abholt. Eine Frontend-Session waere der
  naechste Schritt; der Wire-Vertrag steht in `openapi.yaml` unter
  `FuhrparkVehicleBooking`.
  (3) Es gibt bewusst kein `GET /bookings/{id}`: `produktion` hat fuer
  `machine_bookings` auch keins, und die Liste mit `vehicle_id`-Filter deckt
  jeden bekannten Lesefall. Falls das FE eine Detailansicht braucht, ist das
  eine Zeile Handler plus ein openapi-Pfad.

## Iteration 6 — a-fuhrpark-triplog-export — done — 2026-08-08 17:55
- commit: 0c9d5afb
- gebaut: Finanzamtkonformer Fahrtenbuch-Export unter
  `GET /api/v1/fuhrpark/trip-logs/export?vehicle_id=&from=&to=&format=csv|pdf`.
  Praemisse zuerst geprueft: `/fuhrpark/export` (HandleExportVehicleReport)
  liefert nur eine CSV der Fahrzeugflotte, keine Fahrten — die Unit war also
  nicht durch Ueberschneidung blockiert. Zweite Praemisse, die beim Lesen von
  `TripLog` aufgefallen ist: die geforderte Pflichtangabe "Geschaeftspartner"
  hatte kein Feld im Modell (nur `purpose`, `start_location`, `end_location`,
  `driver_name`). Statt sie wegzulassen oder in `purpose` zu verstecken (wo
  sie fuer Bestandsdaten unrekonstruierbar waere), Migration 000301:
  `trip_logs.business_partner TEXT NOT NULL DEFAULT ''`, durchgezogen durch
  Proto (`business_partner` an `TripLog`/`Create`/`UpdateTripLogRequest`,
  neu generiert), Repository, Service, gRPC-Server und Gateway-Wire-Typen
  (`createTripLogRequest`/`updateTripLogRequest`) — bestehende Zeilen bleiben
  lesbar mit leerem String.
  Export selbst ist NEU implementiertes Format-Handling, aber KEIN neuer
  PDF/CSV-Code: `Service.ExportTripLogs` baut ein `berichte.ReportResult`
  (10 Spalten: Lfd. Nr., Datum, Fahrer, Start, Ziel, Zweck, Geschaeftspartner,
  KM Beginn, KM Ende, Gefahrene KM) und reicht es an
  `export.CSVExporter`/`export.PDFExporter` aus `internal/berichte/export`
  durch — dieselben Typen, die `berichte`/`cmd/berichte` schon benutzen, hier
  zum ersten Mal aus einem anderen Service heraus (kein Zyklus, `berichte`
  importiert `fuhrpark` nirgends). Neue Repository-Methode
  `ListTripLogsForExport` filtert Tenant, optional Fahrzeug und Datumsbereich
  vollstaendig in SQL (kein Laden+Filtern in Go), sortiert aufsteigend nach
  Datum/created_at — genau diese Reihenfolge liefert die luecken-sichere
  fortlaufende Nummerierung, denn die Zeilennummer 1..N entsteht rein aus der
  Aufzaehlung der Ergebnisreihenfolge, nicht aus einer gespeicherten Sequenz.
  Cap bei 5000 Zeilen pro Export (`exportTripLogsCap`, Kommentar mit
  Upgrade-Pfad), analog zum bestehenden `PageSize:10000` in
  `ExportVehicleReport`. Leerer Zeitraum liefert ein gueltiges Dokument mit
  nur der Kopfzeile, kein Fehler. `format=` ausserhalb `csv|pdf` liefert 400
  am Gateway, nicht am Service (Service degradiert defensiv auf CSV, falls er
  je direkt aufgerufen wird).
- gate: build ok | vet ok | lint ok (0 issues, nach einem
  ineffassign-Fund selbst behoben) | test ok (fuhrpark 47/47 PASS inkl. 6
  neuer Service-Tests + 5 neuer DB-Subtests, gateway inkl.
  `TestOpenAPIRouteDrift` = 824 Routen gegen 826 Pfade, server, testutil
  inkl. `TestAllPublicTablesHaveRLSOrAreAllowlisted`) | migration ok (301
  lokal angewendet) | openapi `swagger-cli validate` ok
- mutations-probe: eine, an der SQL-Zeitraumgrenze (Block-A-Unit, keine
  Coverage-Pflicht laut Kopf der Datei, trotzdem gemacht wie in Iteration 5).
  `date >= $n` auf `date > $n` gedreht → GENAU der Subtest
  `date_range_narrows_in_SQL,_not_in_Go` wurde rot, die anderen vier Subtests
  blieben gruen. Zurueckgedreht, Endgate danach erneut gelaufen und gruen.
- db-tests: `TestListTripLogsForExport_FiltersRunInSQL` mit fuenf Subtests
  lief real gegen die lokale DB (Tenant-Scoping, Fahrzeugfilter,
  Datumsbereich, chronologische Reihenfolge, leerer Bereich) — **0 Skips**
  im gesamten Paket `fuhrpark` (47 PASS, 0 SKIP). `DATABASE_URL` war auf
  `kmuhub_app` gesetzt, Migration 301 vorher angewendet.
- verify vorgaenger: `4383e7b2` (a-fuhrpark-vehicle-booking) sauber. Handler
  in `route_fuhrpark.go` gehen ueber `fuhrparkv1.FuhrparkServiceClient`, kein
  direkter Service-Zugriff. Migration 000300 hat `tenant_id NOT NULL` + FK +
  `CALL enable_tenant_rls('vehicle_bookings')` + Permission-Seed fuer
  `fuhrpark:booking:read/write` auf die Rolle `admin`, up UND down gefuellt.
  `.proto` wurde geaendert UND `fuhrpark.pb.go`/`fuhrpark_grpc.pb.go` im
  selben Commit regeneriert (1334 Zeilen Diff, keine Stub-Rueckgabe). Kein
  ersetzter `RequirePermission`-Guard, nur additiv neue Ressource
  `fuhrpark:booking`. Route in `api/openapi.yaml` im selben Commit.
- offen: Zwei Punkte fuer Luke.
  (1) Kein FE-Konsument fuer `business_partner` und den Export-Button.
  `FuhrparkPage.tsx`/Trip-Log-Formular kennen das Feld noch nicht — die
  Spalte ist additiv und bestehende Eintraege lesen mit leerem String, aber
  ohne FE-Aenderung bleibt "Geschaeftspartner" in jedem Fahrtenbuch-Export
  leer, bis das Formular es abfragt.
  (2) Export-Cap bei 5000 Zeilen ist ungetestet am oberen Rand (kein Test mit
  >5000 Zeilen erzeugt, waere ein teurer DB-Seed fuer wenig Aussage) — falls
  ein Tenant je mehr Fahrten zwischen zwei Exports sammelt, greift die Grenze
  still (keine Fehlermeldung, nur ein unvollstaendiges Dokument). Kommentar
  mit Upgrade-Pfad steht im Code (`exportTripLogsCap`).

## Iteration 7 — a-vermietung-inspection-signature — done — 2026-08-08 19:10
- commit: ec2dd22a
- verify vorgaenger: sauber. `0c9d5afb` (a-fuhrpark-triplog-export) geprueft
  gegen alle acht Fehlerklassen: Handler geht ueber
  `fuhrparkv1.FuhrparkServiceClient.ExportTripLogs` (kein direkter
  Service-Zugriff), keine Stub-Rueckgabe (echte Repository-Query +
  `berichte.ReportResult` + `export.CSVExporter`/`PDFExporter`), `.proto`
  UND `fuhrpark.pb.go`/`fuhrpark_grpc.pb.go` im selben Commit regeneriert,
  kein neuer `RequirePermission`-Guard (nutzt den bestehenden
  `fuhrpark:trip`/`read`-Key), Migration 000301 nur eine Spalte an einer
  bereits RLS-gescopten Tabelle (kein neuer Guard noetig), Wire additiv
  (`business_partner` optional), Route `/fuhrpark/trip-logs/export` steht in
  `api/openapi.yaml` im selben Commit, kein Altkey ersetzt. `git merge
  origin/main` war "Already up to date".
- gebaut: Unterschrift und strukturierte Checkliste am
  Zustandsprotokoll (`RentalInspection`). Migration 000302 ergaenzt
  `rental_inspections.signature_data TEXT NULL` und
  `rental_inspections.checklist JSONB NULL` — beide bewusst NULLABLE statt
  NOT NULL DEFAULT: `TestVermietungWrites_LandInCallerTenant`
  (`tenant_write_test.go`) baut eine `RentalInspection` direkt gegen das
  Repository ohne den Service-Normalisierungspfad, und ein NOT-NULL-Default
  haette dort mit dem gebundenen `nil`-Parameter einen Constraint-Verstoss
  ausgeloest, weil ein explizit gebundener NULL-Parameter den DEFAULT
  umgeht. Modell: neuer Typ `ChecklistItem{Label, Condition, Remark}`,
  `RentalInspection.Checklist []ChecklistItem` und `.SignatureData
  *string`. Service: `normalizeChecklist` validiert nicht-leere Labels und
  macht `nil` bei einem Create explizit zu `[]ChecklistItem{}`;
  `validateInlineSignature` ist die aus `SaveSignature` extrahierte
  Validierung (gleiche Praefix-/Groessenpruefung, jetzt von Renal- UND
  Inspektions-Signatur geteilt, Verhalten unveraendert — alle sieben
  bestehenden `signature_test.go`-Tests blieben gruen). `UpdateInspection`
  bekommt `ReplaceChecklist bool` nach demselben Muster wie das
  bestehende `ReplacePhotos` (ein bare `repeated`-Feld hat auf dem
  gRPC-Wire kein Presence-Bit, also braucht "ersetzen" ein explizites Flag)
  — anders als beim photo_urls-Pfad ist das neue Flag hier vom Gateway bis
  zum Service durchgaengig verdrahtet, nicht nur im Proto vorgesehen.
  Proto: `ChecklistItem`-Message, `RentalInspection.checklist`+
  `.signature_data`, `CreateInspectionRequest.checklist`,
  `UpdateInspectionRequest.checklist`+`.replace_checklist`+
  `.signature_data` — regeneriert im selben Commit. Gateway:
  `checklistItemRequest`, beide Request-Structs erweitert, neue
  Helfer `checklistRequestToProto`/`checklistToProto`/`checklistFromProto`
  (Serverseite) normalisieren `nil` zu `[]` fuers Wire, damit ein
  Bestand ohne Checkliste nie als `null` serialisiert. Route/Guard
  unveraendert (bestehender `vermietung:inspection`/`write`-Key, kein
  neuer Seed). `api/openapi.yaml`: neues Schema
  `VermietungChecklistItem`, Felder an `VermietungRentalInspection`, Request-
  Bodies von POST .../inspections und PATCH .../inspections/{id}.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (vermietung 51/51,
  0 Skips; gateway ok inkl. `TestOpenAPIRouteDrift`; server ok inkl. neuer
  `vermietung_checklist_test.go`) | migration ok (302 lokal angewendet,
  Kopf jetzt 302) | openapi `swagger-cli validate` ok | rls-smoke n.a.
  (keine neue Tabelle/Policy — `rental_inspections` steht seit Migration
  000122 unter RLS; `relrowsecurity`/`relforcerowsecurity` nach der
  `ALTER TABLE` verifiziert weiterhin `t|t`, aber die Tabelle hat lokal
  keine Zeilen fuer einen aussagekraeftigen Positiv/Negativ-Vergleich)
- mutations-probe: zwei Stueck, beide aussagekraeftig.
  (1) `normalizeChecklist`s Leerlabel-Check von `== ""` auf `!= ""`
  gedreht → alle fuenf neuen Checklist-Tests (Create Happy/Reject, Update
  Replace/Ignore/Reject) wurden rot, keine Nachbartests betroffen.
  (2) `if input.ReplaceChecklist` in `UpdateInspection` auf `if true`
  gesetzt → GENAU `TestService_UpdateInspection_ChecklistIgnoredWithoutReplaceFlag`
  wurde rot, alle 50 uebrigen Tests im Paket (inklusive der beiden echten
  DB-Tests) blieben gruen. Beide Proben zurueckgedreht, Endgate danach
  erneut gelaufen und gruen.
- db-tests: `TestVermietungWrites_LandInCallerTenant` und
  `TestTenantIsolation_Vermietung` liefen real gegen die lokale DB — **0
  Skips** im gesamten Paket `vermietung` (51 PASS, 0 SKIP). `DATABASE_URL`
  war auf `kmuhub_app` gesetzt, Migration 302 vorher angewendet. Kein
  eigener DB-Test fuer die neuen Spalten noetig, weil
  `TestVermietungWrites_LandInCallerTenant` bereits eine Inspektion ohne
  Checklist/Signature ueber den echten Repository-Pfad anlegt und damit
  den Nullable-Fall abdeckt — hat das Fehlen der zweiten Migrationsspalte
  im gebundenen INSERT sofort aufgedeckt (siehe "gebaut").
- offen: Drei Punkte fuer Luke.
  (1) Kein FE-Konsument. Weder ein Checklisten-Editor noch ein
  Signatur-Pad fuer Inspektionen existieren im Vermietung-Modul — der
  Wire-Vertrag steht in `openapi.yaml` unter `VermietungRentalInspection`/
  `VermietungChecklistItem`.
  (2) `Condition` (Zustand) ist bewusst Freitext, kein Enum — die Aufgabe
  nannte keine feste Werteliste, und `production_orders`-artige
  Value-Sets waeren eine eigene Entscheidung. Falls das FE ein Dropdown
  will (z. B. intakt/beschaedigt/fehlt), ist das eine kleine Folge-Unit.
  (3) `signature_data` an der Inspektion ist NUR ueber PATCH setzbar, nicht
  beim Create — Signaturen entstehen in der Praxis erst nach der
  Begehung, nicht beim Anlegen des Datensatzes. Falls das FE einen
  Create-mit-Signatur-Flow braucht, ist das additiv nachruestbar
  (`optional string signature_data` an `CreateInspectionRequest`).

## Iteration 8 — a-users-preferences — done — 2026-08-08 20:35
- commit: 836c98c0
- verify vorgaenger: sauber. `ec2dd22a` (a-vermietung-inspection-signature)
  geprueft gegen alle acht Fehlerklassen: `HandleCreateInspection`/
  `HandleUpdateInspection` gehen ueber `client.CreateInspection`/
  `client.UpdateInspection` (`vermietungv1.VermietungServiceClient`), keine
  direkte Service-Instanz; keine Stub-Rueckgabe; `.proto` UND `vermietung.pb.go`
  im selben Commit regeneriert (kein neuer RPC, also kein `_grpc.pb.go`-Diff
  noetig); kein neuer `RequirePermission`-Guard (bestehender
  `vermietung:inspection`/`write`-Key); Migration 000302 fuegt nur zwei
  NULLABLE Spalten an einer seit Migration 000122 RLS-gescopten Tabelle an,
  keine neue Policy noetig; `checklistToProto` liefert immer ein
  Non-Nil-Slice, also `[]` statt `null`; Pfad-Erweiterungen in
  `api/openapi.yaml` im selben Commit; kein Alt-Key ersetzt. War bereits als
  Iteration 7 im Journal protokolliert, dieser Lauf hat nur die SHA
  nachgetragen (`1c283333`, chore-Commit) und dann selbst verifiziert. `git
  merge origin/main` war "Already up to date".
- gebaut: `GET`/`PUT /api/v1/users/preferences` — duenner Endpunkt ueber der
  bestehenden `user_settings`-Tabelle (Migration 000138), `module_id =
  "profile"`, KEINE neue Tabelle. Praemisse beim Lesen widerlegt: die
  Backlog-Notiz "FE haelt Praeferenzen nur im localStorage" stimmt fuer
  Sprache nicht — `desktop/.../stores/locale.ts` synct das Locale bereits
  serverseitig ueber `PUT /settings/language/user`; Theme ist in
  `stores/ui.ts` bewusst NICHT synct (`lean:`-Marker mit explizitem
  Upgrade-Pfad auf `/settings/appearance/user`) und "Region" existiert im FE
  ueberhaupt nicht als allgemeine Praeferenz. Der Endpunkt ist trotzdem
  gebaut, weil `done_when` ihn explizit fordert und er echten Mehrwert hat:
  anders als der generische `/settings/{module_id}/user` lehnt er unbekannte
  Schluessel mit 400 ab und ersetzt beim PUT den ganzen Satz (Keys, die im
  Body fehlen, werden GELOESCHT, nicht nur unangetastet gelassen) — echte
  Full-Replace-Semantik statt Patch. Dafuer eine neue RPC
  `ReplaceUserSettings` am bestehenden `SettingsService` ergaenzt (Proto +
  `settings.pb.go`/`_grpc.pb.go` regeneriert), Repository-Methode nach dem
  Delete-dann-Insert-Transaktionsmuster aus
  `auth.PostgresRepository.SetUserOverrides` (nicht neu erfunden). Erlaubte
  Schluessel: `language`, `theme`, `region` (`userPreferenceKeys`-Allowlist
  im Gateway). Handler ignorieren jedes `user_id`-Feld im Body strukturell —
  `putUserPreferencesRequest` hat gar kein solches Feld, der Zielnutzer kommt
  ausschliesslich aus `middleware.GetUserID(ctx)`. Guard: bestehender
  `settings:read`/`settings:write`-Key (Migration 000138 seedet ihn schon
  fuer admin/manager/member) — KEINE neue Seed-Migration noetig, KEINE neue
  Migration ueberhaupt. Response-Form identisch zu
  `GET/PUT /settings/{module_id}/user` (`{entries:[{key,value}]}` via
  `response.Proto`), damit das Wire-Muster konsistent bleibt.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (settings 42/42 PASS
  0 SKIP inkl. neuem DB-Test, server ok, gateway ok inkl.
  `TestOpenAPIRouteDrift` = 826 Routen gegen 827 Pfade) | migration n.a.
  (keine neue Tabelle, keine neue Permission) | openapi `swagger-cli
  validate` ok
- mutations-probe: zwei Stueck, beide aussagekraeftig.
  (1) Gateway: `if !userPreferenceKeys[key]` zu `if false` gedreht (Allowlist
  wirkungslos) → GENAU `TestHandlePutUserPreferences_UnknownKeyRejected`
  wurde rot (503 statt 400 — der Request kam ungeprueft bis zum toten
  gRPC-Dial durch), die vier Nachbartests blieben gruen.
  (2) Repository: `DELETE FROM user_settings WHERE ... module_id = $3` um
  `AND FALSE` erweitert (Full-Replace liefert kein Delete mehr) → GENAU
  `TestSettingsWrites_LandInCallerTenant` wurde rot (`ERROR: duplicate key
  value violates unique constraint "user_settings_pk"` — der zweite INSERT
  auf denselben Schluessel `language` kollidierte mit der nicht geloeschten
  Altzeile). Beide Proben zurueckgedreht, Endgate danach erneut gelaufen und
  gruen (42/42, 0 Skips).
- db-tests: `TestSettingsWrites_LandInCallerTenant` (erweitert um den
  `ReplaceUserSettings`-Block: Full-Replace loescht `region`, behaelt nur das
  neu gesetzte `language`, Cross-Tenant liefert 0 Zeilen) lief real gegen die
  lokale DB — **0 Skips** im gesamten Paket `settings` (42 PASS, 0 SKIP).
  `DATABASE_URL` war auf `kmuhub_app` gesetzt.
- offen: Zwei Punkte fuer Luke.
  (1) Kein FE-Konsument. `stores/locale.ts` und `stores/ui.ts` schreiben
  weiterhin gegen `/settings/language/user` bzw. gar nicht serverseitig
  (Theme) — eine Umstellung auf `/users/preferences` waere ein
  FE-Vertragswechsel und liegt ausserhalb dieser Unit. Der neue Endpunkt ist
  bewusst additiv, kollidiert mit nichts Bestehendem.
  (2) Architektur-Entscheidung zur Bestaetigung: das Full-Replace-Verhalten
  (fehlender Key = geloescht) weicht vom Rest der Settings-Familie ab (dort
  ueberall Patch-Semantik). Wenn das FE spaeter einzelne Praeferenzen partiell
  aktualisieren will (z. B. nur Theme aendern, ohne Sprache/Region
  mitzuschicken), muesste es bei jedem PUT den vollstaendigen Satz
  mitschicken (erst GET, dann PUT mit allen dreien) — sonst gehen die
  fehlenden Keys verloren. Das ist Absicht laut `notes` im Backlog ("PUT
  ersetzt den ganzen Satz"), aber ein FE-Implementierer muss das wissen.
