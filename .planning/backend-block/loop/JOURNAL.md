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

## Iteration 9 — a-berichte-kpi-timeseries — done — 2026-08-08 22:15
- commit: bff9c891
- verify vorgaenger: sauber. `836c98c0` (a-users-preferences) geprueft gegen
  alle acht Fehlerklassen: `HandleGetUserPreferences`/`HandlePutUserPreferences`
  gehen ueber `sr.getSettingsClient()` (gRPC-Client), kein direkter
  Service-Zugriff; keine neue Tabelle, kein neuer `RequirePermission`-Guard
  (bestehender `settings:read`/`write`-Key); die Ziel-`user_id` kommt beim PUT
  ausschliesslich aus `middleware.GetUserID(ctx)`, `putUserPreferencesRequest`
  hat strukturell gar kein `user_id`-Feld; `openapi.yaml` im selben Commit
  erweitert; kein Alt-Key ersetzt. `git merge origin/main` war "Already up to
  date", `git fetch` brachte nichts Neues.
- gebaut: `GET /api/v1/berichte/kpis` liefert jetzt `series` (bis zu acht
  Kalendermonats-Perioden, aelteste zuerst) je KPI mit Historie, statt dass
  das FE die Sparkline deterministisch aus `change_percent` synthetisiert.
  Proto: neue Message `KPISeriesPoint` (`period_start`, `value`) +
  `repeated KPISeriesPoint series = 7` an `KPI`, regeneriert
  (`berichte.pb.go`; `berichte_grpc.pb.go` unveraendert, kein neuer RPC).
  Modell/Executor: `berichte.KPISeriesPoint`, `executor.KPISeriesPoint`,
  `KPIRepo.KPISeries(ctx, tenantID, now, periods)` als vierte Methode neben
  `KPISnapshot`. Postgres: `kpiSeriesQuery` in `downstream/kpi_postgres.go` —
  EINE SQL-Abfrage mit einer `periods`-CTE (`generate_series` ueber die
  letzten acht Kalendermonate) und drei Aggregat-Subqueries pro Bucket
  (revenue, pipeline_volume, open_tickets), kein Go-Loop ueber acht
  Snapshot-Aufrufe. Bucket-Grenzen sind exklusiv: fuer Vormonate die
  Monatsgrenze, fuer den laufenden (unvollstaendigen) Monat `now + 1 Tag` —
  der letzte Punkt deckt sich damit exakt mit `KPISnapshot`s "aktuell"
  (Monat bis heute). Stock-Warnings bleiben bewusst ohne Serie — die Zeilen
  werden in-place mutiert (kein `resolved_at`), dieselbe Begruendung wie das
  bestehende Fehlen von `change_percent` dort; das weicht von der woertlichen
  Lesart "series je sichtbarem KPI" in `done_when` ab, ist aber dieselbe
  Einschraenkung, die der bestehende Code fuer `change_percent` bei diesem
  KPI schon dokumentiert (kein neu erfundenes Problem, nur jetzt auch fuer
  die Serie sichtbar gemacht). `DashboardKPIs` ruft `KPISeries` einmal pro
  Aufruf ab (nicht einmal je KPI) und degradiert bei Fehler graceful genau
  wie beim `prev`-Snapshot (Log-Warn, Serie bleibt weg, aktuelle Werte
  bleiben erhalten). Modul-Sichtbarkeit unveraendert und automatisch
  mitgedeckt: `series` haengt am selben `berichte.KPI`-Objekt wie
  `value`/`change_percent`, die bestehende `visibleKPIModules`-Filterung in
  `route_berichte.go` (fail-closed pro Modul) laesst ein nicht sichtbares
  Modul gar nicht erst in den RPC-Aufruf — kein separater Zugriffspfad zum
  Umgehen. openapi.yaml: neues Schema `BerichtKPISeriesPoint`, `series`-Feld
  additiv an `BerichtKPI`.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (berichte,
  berichte/executor, berichte/downstream, server, gateway inkl.
  `TestOpenAPIRouteDrift`, testutil inkl.
  `TestAllPublicTablesHaveRLSOrAreAllowlisted`) | migration n.a. (keine neue
  Tabelle, keine neue Spalte) | openapi `swagger-cli validate` ok
- mutations-probe: zwei Stueck, beide aussagekraeftig.
  (1) SQL: `closed_at >= p.period_end` zu `closed_at > p.period_end` gedreht
  (Grenzfall exakt auf dem Bucket-Rand kippt auf die falsche Seite) → GENAU
  `TestKPISeries_BucketsByCalendarMonthAndExcludesForeignTenant` wurde rot
  (Pipeline-Delta im Vormonats-Bucket 0 statt 654), die beiden
  `TestKPISnapshot_*`-Nachbartests blieben gruen.
  (2) Go: in `DashboardKPIs` die Revenue-Serie versehentlich auf
  `p.PipelineVolume` statt `p.Revenue` verdrahtet → GENAU
  `TestDashboardKPIs_SeriesPerModule` wurde rot, alle acht Nachbartests im
  Paket blieben gruen. Beide Proben zurueckgedreht, Endgate danach erneut
  gelaufen und gruen.
- db-tests: `TestKPISeries_BucketsByCalendarMonthAndExcludesForeignTenant`
  (neu: Foreign-Tenant-Kontrolle + exakte Bucket-Grenze) lief real gegen die
  lokale DB, zusammen mit den zwei bestehenden `TestKPISnapshot_*`-Tests —
  **0 Skips** im gesamten Paket `downstream` (3 PASS, 0 SKIP). `DATABASE_URL`
  war auf `kmuhub_app` gesetzt. Beim ersten Lauf des neuen Tests einen
  echten Zeitzonen-Bug im TEST (nicht im Produktionscode) gefunden und
  behoben: `dbNow(t, pool)` kann eine nicht-UTC `time.Location` tragen
  (lokale Maschinenzeit statt UTC), obwohl der zugrunde liegende Zeitpunkt
  korrekt ist — `time.Date(..., now.Location())` daraus zu bauen verschiebt
  die gebundenen Fixture-Zeitstempel um den Zonen-Offset und laesst sie auf
  der falschen Seite einer exakten Bucket-Grenze landen. Fix: `.UTC()` auf
  `now` vor der Monatsarithmetik im Test. Die Produktions-Query war davon
  nicht betroffen, weil die Bucket-Grenzen vollstaendig in SQL berechnet
  werden (DB-Session-Timezone verifiziert `UTC` per `SHOW timezone`), nicht
  in Go.
- offen: Zwei Punkte fuer Luke.
  (1) Kein FE-Konsument. `DashboardGrid.tsx` synthetisiert die Sparkline im
  Drilldown-Modal weiterhin deterministisch aus `change_percent`
  (`buildDrilldownSeries`) statt die neue `series` zu lesen — der
  Wire-Vertrag steht in `openapi.yaml` unter `BerichtKPI.series`/
  `BerichtKPISeriesPoint`. Eine FE-Session muesste `buildDrilldownSeries`
  durch einen echten `series`-Read ersetzen.
  (2) `stock_warnings_count` traegt bewusst keine `series` (siehe "gebaut").
  Eine woertliche Lesart von `done_when` ("acht Punkte je sichtbarem KPI")
  waere hier nur durch Erfinden von Daten erfuellbar; die Abweichung ist
  dieselbe, die der bestehende Code fuer `change_percent` bei diesem KPI
  schon macht und dokumentiert — kein neues Problem, nur jetzt auch fuer die
  Serie sichtbar.

## Iteration 10 — a-berichte-execute-kind-cross — done — 2026-08-08 17:30
- commit: a86c166c
- gebaut: Neue Report-Art `cross` im Executor
  (`internal/berichte/executor/executor.go`). Sie fuehrt **zwei bestehende
  Arten** ueber einen gemeinsamen Wert zusammen, statt eigene Downstream-
  Abfragen zu bauen — damit erbt sie Tenant-Scoping, Fehlerpfade und die
  Graceful Degradation der Art, die sie wiederverwendet (Lean-Leiter Stufe 2).
  `Run` wurde in `Run` (nur `cross`-Weiche) und `runKind` (die acht
  Einzelquellen-Arten) geteilt; `runKind` kennt `cross` bewusst NICHT, damit
  eine Quelle nicht wieder `cross` heissen kann — die Rekursion faellt
  automatisch in `ErrInvalidQueryConfig` statt in einen Fan-out.
  Config additiv im bestehenden `query_config`-Dialekt:
  `{"kind":"cross","period":"…","join_on":"stage","sources":[{"kind":"pipeline","key":"stage"},{"kind":"conversion","key":"from_stage","period":"last_30_days"}]}`.
  Ergebnis ist dieselbe `Columns/Rows/Totals/Meta`-Form wie bei jeder anderen
  Art — CSV/XLSX/PDF-Export und die FE-Tabelle brauchen keinen Sonderfall.
- gate: build ok | vet ok | lint ok (0 issues) | test ok
  (`./internal/berichte/...` alle sechs Pakete gruen, **0 Skips** bei 185
  Testfaellen, `DATABASE_URL` auf `kmuhub_app` gesetzt) |
  `go test ./internal/gateway/` ok (inkl. `TestOpenAPIRouteDrift`) |
  migration n.a. | openapi n.a. (keine neue Route, keine neue Response-Form —
  `query_config` ist in der Spec ein freies JSON-Feld)
- verify vorgaenger (`bff9c891`, KPI-Series): sauber. `.proto` und `.pb.go` im
  selben Commit regeneriert; die neue `KPISeries`-Query filtert in allen drei
  Sub-Selects auf `tenant_id = $1` (Read-Seite also nicht vergessen);
  `response.Proto` serialisiert mit `UseProtoNames` → `period_start` snake_case
  und Timestamp als RFC3339, deckt sich mit dem neuen openapi-Schema
  `BerichtKPISeriesPoint`; keine neue Route, keine neue Tabelle, kein neuer
  `RequirePermission`-Guard, kein Stub im neuen Pfad.
- entscheidungen (bewusst, nicht geraten):
  1. **Per-Source-Join-Keys statt eines gemeinsamen Spaltennamens.** Beim Bauen
     nachgeschlagen: von den acht bestehenden Arten teilen sich **keine zwei**
     einen Spaltennamen (pipeline nennt die Dimension `stage`, conversion
     `from_stage`, helpdesk `queue`, datev `code`). Ein `join_on`, das in beiden
     Quellen identisch heissen muss, waere mit dem heutigen Bestand unbenutzbar
     gewesen. Deshalb: jede Quelle nennt ihre eigene Schluesselspalte (`key`,
     Default = `join_on`), `join_on` ist der Name, den die Schluesselspalte im
     Ergebnis traegt. Das ist der Punkt, den `notes` als moegliche Vertragsfrage
     markiert hat — er ist im Konfigurationsschema loesbar, ohne dass ein
     Downstream oder ein FE-Vertrag angefasst wird, also kein `blocked`.
  2. **Inner Join, nicht outer.** Der CSV-Exporter rendert fehlende Zellen als
     `fmt.Sprintf("%v", nil)` → `"<nil>"`; ein Outer Join haette also Loecher
     mit `<nil>` in jede Export-Datei geschrieben. Die verworfenen Zeilen sind
     dafuer sichtbar: `Totals["cross_unmatched_<alias>"]` je Seite.
  3. **Kein `Series`.** Welche der zusammengefuehrten Kennzahlen ein Chart
     zeichnen soll, steht in der Config nicht — Raten haette einen falschen
     Chart erzeugt statt gar keinen.
  4. Spalten ausser dem Schluessel bekommen den Alias-Praefix (`pipeline_volume`,
     `conversion_rate`); der Praefix laeuft ueber die Row-Map, nicht ueber die
     deklarierten Spalten, weil Zeilen undeklarierte Extras (`currency`,
     `user_id`) tragen, die zwischen zwei Quellen kollidieren.
  5. Zwei Deckel, beide noetig: `maxCrossSourceRows = 1000` je Quelle (grosse
     Einzelquelle) UND dieselbe Grenze auf der Ausgabe (kartesisches Produkt bei
     wiederholten Schluesseln — 1000 × 1000 waere eine Million Zeilen aus einem
     Aufruf). Getroffener Deckel setzt `Meta.Warning = "cross_truncated"`.
- mutations-probe: drei Stueck, die zweite war der eigentliche Ertrag.
  (1) Join-Seite vertauscht (`joinKey(leftRow, rightKey)`) → vier cross-Tests
  rot, alle Nachbartests gruen. Zurueckgedreht.
  (2) Per-Source-Deckel aufgeweicht (`> maxCrossSourceRows+250`) → **zuerst
  NICHT gefangen**: mein erster Cap-Test (beide Seiten ueberlang, passende
  Schluessel) wurde vom Ausgabe-Deckel gerettet und haette den Per-Source-Deckel
  nie belegt — genau die Zeilenabdeckung ohne Aussage, vor der die Regel warnt.
  Test daraufhin ersetzt durch `TestRun_Cross_SourceRowCapDropsRowsBeyondLimit`:
  der einzige passende Schluessel liegt jenseits des Deckels, der Join findet ihn
  also nur, wenn nicht gekappt wurde (zwei Subtests, links und rechts). Probe
  wiederholt → beide Subtests rot, Rest gruen. Zurueckgedreht.
  (3) Ausgabe-Deckel verdoppelt → GENAU
  `TestRun_Cross_OutputRowCapBoundsCartesianProduct` rot (beide Quellen exakt am
  Per-Source-Limit, damit dieser Deckel stumm bleibt), alle anderen gruen.
  Zurueckgedreht, Endgate danach erneut gelaufen und gruen.
- fehlerpfade im Test: unbekannte Quellen-Art, verschachteltes `cross`, leere
  Quellen-Art, eine/drei Quellen, fehlendes `join_on`, kollidierende Aliase,
  Schluessel ist keine Spalte der Quelle (alle → `ErrInvalidQueryConfig`, kein
  Panic), Downstream nicht verdrahtet (→ `downstream_not_available` statt
  Config-Fehler), Downstream-Fehler propagiert, Per-Source-Period-Override
  erreicht das Downstream wirklich (am Mock gemessen).
- offen: Zwei Punkte fuer Luke.
  (1) **Kein Erzeuger.** `cross` ist serverseitig da, aber niemand schreibt so
  eine Config: der FE-Builder produziert einen ganz anderen Dialekt
  (`{"kind":"builder","sourceId":…}`, siehe `desktop/.../berichte-types.ts`),
  den der Executor **gar nicht kennt** — `runKind` gibt dafuer heute
  `ErrInvalidQueryConfig` zurueck, interpretiert wird er nur vom MSW-Demo-Handler.
  Das ist eine groessere Luecke als diese Unit und steht in keinem Backlog-Eintrag;
  bis sie geschlossen ist, ist `cross` nur ueber die API (oder eine Seed-Zeile)
  erreichbar. Erwaegenswert als Unit fuer Lauf 7.
  (2) Kein Seed-Bericht fuer `cross` angelegt — die Migration 000079 seedet acht
  System-Definitionen, eine neunte haette einen Deploy-relevanten Datenzusatz
  bedeutet, ohne dass das FE ihn heute rendern kann. Sinnvoll erst zusammen mit (1).

## Iteration 11 — a-automation-cron-poller — done — 2026-08-08 17:40
- commit: a6e4665b
- gebaut: Faelligkeitsaufloesung fuer die zeitbasierten Automations-Trigger.
  Neu: `trigger.DueResolver` (+ `DueEntity`, `ErrUnknownTimeTrigger`) in
  `due.go`, `PostgresDueResolver` in `due_postgres.go` mit je einer Abfrage
  fuer `biz.invoice.overdue` (finance_invoices) und `calendar.event.upcoming`
  (calendar_events), Migration 000303 `automation_time_trigger_fires` als
  Per-Entitaet-Dedup, `workflow.Repository.ClaimTimeTriggerFire`, und der
  Poller feuert jetzt pro faelliger Entitaet statt pro Tick.
- gate: build ok | vet ok | lint 0 issues | test ok (18 PASS, **0 SKIP**,
  davon 4 echte DB-Tests gegen die lokale DB als `kmuhub_app`) |
  migration 000303 angewandt, Kopf jetzt 303 | rls-smoke ok
  (fremder Tenant 0 Zeilen, eigener 1) | `TestAllPublicTablesHaveRLSOrAreAllowlisted`
  gruen | `TestOpenAPIRouteDrift` gruen (keine Route angefasst)
- verify vorgaenger (`a86c166c`): sauber. Nur `executor.go` + Tests; keine
  Route, kein Proto, keine Migration, kein Guard, keine neue Tabelle. Auf
  Stub-Marker im neuen Pfad gegrept — nichts.

### Der eigentliche Befund: die Unit-Praemisse war falsch, der Gap ein anderer

Das Backlog sagte „kein Cron, kein `time.Ticker` in `internal/automation/`".
Das stimmt seit Lauf 4 nicht mehr: `trigger/poller.go` existiert, laeuft
(`cmd/automation/main.go:174`), hat einen atomaren Claim auf `last_polled_at`
(Migration 000284) und Tests. Alle vier `done_when` waren nominell erfuellt.

Beim Nachlesen war der Poller aber **schlimmer als keiner**:

1. Er lud aktive zeitbasierte Automationen — nicht faellige Entitaeten. Jede
   aktive `biz.invoice.overdue`-Automation feuerte **alle 5 Minuten**,
   unabhaengig davon, ob eine Rechnung ueberfaellig ist. Der
   `last_polled_at`-Claim verhindert nur Doppelausfuehrung *innerhalb* eines
   Ticks, nicht die Wiederholung *ueber* Ticks: beim naechsten Tick liest der
   Poller den neuen Wert und der Claim gelingt wieder.
2. Das synthetische Event trug **gar keine Payload** (`{Type, ModuleID,
   Timestamp}`). `engine.buildEnvFromPayload` merged `evt.Payload` flach ins
   env — ohne Payload gibt es kein `invoice.days_overdue`. Die mitgelieferte
   Vorlage `invoice-overdue-dunning` filtert auf `invoice.days_overdue >= 14`
   und konnte damit **nie** wahr werden. Netto: die Automation feuerte
   entweder dauernd (ohne Bedingung) oder nie (mit Bedingung).
3. `p.pool` war ein totes Feld — deklariert, im Konstruktor gesetzt, nirgends
   benutzt. Genau der Platz, an den die Faelligkeitsabfrage gehoert hat.
4. Niemand sonst publiziert diese Events: `grep EventInvoiceOverdue` findet
   ausser der Registry und dem Poller keinen Erzeuger.

Ich habe die Unit deshalb nicht als „schon erledigt" abgehakt, sondern den
verbliebenen Kern gebaut. `done_when` Punkt 2 („faellige Automation wird
ausgefuehrt") war vorher nur dem Wortlaut nach erfuellt.

### Entscheidungen (nachgeschlagen, nicht geraten)

- **Payload verschachtelt**, nicht flach. `condition.getFieldValue`
  (`evaluator.go:174`) laeuft `invoice.days_overdue` als **verschachtelte
  Maps** ab; `action/executor.go:32` flacht erst fuer Templates ab. Ein
  flaches `{"invoice.days_overdue": 12}` haette keine einzige Bedingung
  getroffen. Test `..._EventCarriesTenantResourceAndNestedPayload` pinnt das.
- **Dedup-Granularitaet gehoert dem Resolver, nicht der Tabelle.**
  Rechnung: `invoice:<id>:<yyyy-mm-dd>` = einmal pro Tag. Einmal-pro-Rechnung
  waere falsch — die Dunning-Vorlage zuendet erst ab Tag 14, ein einziger
  Schuss an Tag 1 haette die Bedingung `false` ausgewertet und die einzige
  Chance verbraucht. Kalender: `calendar_event:<id>` ohne Datum — ein Termin
  beginnt einmal, eine Erinnerung an zwei Tagen ist ein Bug.
- **Zwei Waechter, zwei verschiedene Fragen.** `last_polled_at` (Bestand):
  „darf DIESE Instanz diese Automation in DIESEM Tick anfassen".
  `automation_time_trigger_fires` (neu): „hat diese Automation fuer DIESE
  Entitaet schon gefeuert". Beide noetig, keiner ersetzt den anderen.
  Beide `INSERT ... ON CONFLICT DO NOTHING` / bedingtes `UPDATE` — also
  atomar, kein check-then-act.
- **Tenant-Scoping explizit in jedem Query.** Der Poller laeuft unter
  `database.WithSystemContext` — RLS ist damit offen, das `WHERE tenant_id =
  $1` ist das einzige, was Tenant As Automation von Tenant Bs Rechnungen
  fernhaelt. Zwei DB-Tests belegen das mit zwei echten Tenants.
- **Deckel 200 Entitaeten je Automation je Tick** (Resolver-`LIMIT`). Der
  Poller startet eine Goroutine pro gefeuerter Entitaet; ein Tenant mit
  fuenfstelligem Mahn-Rueckstand haette beim ersten Tick nach Aktivierung
  genau so viele gleichzeitige Ausfuehrungen gestartet. Getroffener Deckel
  loggt `Warn` und vertagt den Rest auf den naechsten Tick.
- **Mitgenommener Bug (gleiche Wurzel):** `buildEnvFromPayload` setzte nie
  `tenant_id` ins env, obwohl die Dunning-Vorlage `{{tenant_id}}` an
  `biz.create_dunning` durchreicht. Da ich genau diesen Pfad zum Leben
  erwecke, waere er sonst garantiert mit leerem Tenant gescheitert. Fix an
  der gemeinsamen Funktion, nur wenn `evt.TenantID != uuid.Nil` — eine
  gerenderte Null-UUID waere schlimmer als ein unaufgeloester Platzhalter.

### Mutations-Probe: drei Stueck, alle gefangen

1. Tenant-Filter im Overdue-Query durch `($1::uuid IS NOT NULL)` ersetzt →
   **genau** die zwei Tenant-Tests rot
   (`..._ScopesToTenantAndCarriesFields`, `..._IgnoresNotYetDueAndSettled`),
   alle Unit-Tests gruen. Zurueckgedreht.
2. `if !fired { continue }` im Poller entschaerft → **genau**
   `TestFireDueEntities_FireClaimLost_DoesNotExecute` rot. Zurueckgedreht.
3. Vorzustand rekonstruiert (leeres Resolver-Ergebnis feuert trotzdem eine
   Pseudo-Entitaet) → **genau** `TestFireDueEntities_NothingDue_DoesNotExecute`
   rot. Zurueckgedreht, Endgate danach erneut gelaufen und gruen.

### Fehlerpfade im Test
Resolver-Fehler (bricht die uebrigen Automationen nicht ab), unbekannter
Trigger-Typ (`ErrUnknownTimeTrigger` → kein Feuern statt leerer Payload),
Fire-Claim verloren, Fire-Claim-Fehler (die restlichen Entitaeten laufen
weiter), nichts faellig, Deckel-grosse Ergebnismenge, sowie DB-seitig:
zukuenftig faellige / bezahlte / Entwurfs-Rechnung werden ignoriert, Termin
ausserhalb des 15-Minuten-Fensters und bereits begonnener Termin ebenso.

### Offen fuer Luke
1. **Serientermine sind nicht abgedeckt.** `rrule`-Instanzen werden nirgends
   serverseitig materialisiert (die Expansion liegt im Desktop-Client), also
   trifft `calendar.event.upcoming` eine Serie nur zu ihrer Ur-Startzeit.
   `lean:`-Marker mit Upgrade-Trigger steht an der Abfrage.
2. **Keine Retention** auf `automation_time_trigger_fires`. Wachstum ist eine
   Zeile pro echtem Feuern; `lean:`-Marker nennt
   `CleanupOldExecutions` als Anschlussstelle ab sechsstelligen Zeilenzahlen.
3. **Der Poller ist jetzt scharf** in dem Sinn, dass eine aktive
   `biz.invoice.overdue`-Automation ab Deploy real Mahnungen anlegen kann —
   vorher konnte sie das nachweislich nie. Lokal existiert genau eine
   ueberfaellige Rechnung (RE-2026-001, 69 Tage). Vor dem Deploy einmal
   pruefen, wie viele aktive Automationen dieses Typs produktiv stehen.
4. `finance_invoices.status` kennt `'overdue'`, aber niemand setzt es je —
   die Abfrage nimmt `IN ('sent','overdue')`, damit ein spaeterer Setzer den
   Trigger nicht stillschweigend abwuergt.

## Iteration 12 — a-video-caller-identity — done — 2026-08-08 17:27
- commit: e736f324
- verify vorgaenger (`a6e4665b`, a-automation-cron-poller): sauber. Migration
  000303 hat `tenant_id UUID NOT NULL` + FK auf `tenants`/`automations` + `CALL
  enable_tenant_rls('automation_time_trigger_fires')`, up und down beide
  gefuellt. `PostgresDueResolver` scopt beide Queries (`overdueInvoices`,
  `upcomingCalendarEvents`) explizit auf `tenant_id = $1` — der Poller laeuft
  unter `database.WithSystemContext`, RLS ist damit offen, das WHERE ist die
  einzige Schranke, und beide Stellen haben sie. Kein neuer
  `RequirePermission`-Guard, keine Route, kein `.proto` angefasst, also kein
  `openapi.yaml`-Bedarf und `TestOpenAPIRouteDrift` unveraendert. Kein Stub im
  neuen Pfad (`PostgresDueResolver` fuehrt echte SQL aus, keine
  Platzhalter-Rueckgabe). `git merge origin/main` war "Already up to date".
- gebaut: `call.incoming`-Broadcast traegt jetzt `caller_name` und
  `caller_avatar`. Praemisse beim Lesen bestaetigt: die Notiz "FE zeigt eine
  UUID statt eines Namens" stimmt exakt, und `HandleCreateCall`
  (`route_video.go:318`) baute den Broadcast bislang nur mit `caller_id`.
  Fund beim Lesen des FE-Konsumenten (`useIncomingCallListener.ts:43-49`):
  dort liegt bereits ein `lean:`-Kommentar, der genau diese Luecke benennt
  und `caller_name`/`caller_avatar` schon entgegennimmt — die Feldnamen sind
  also nicht neu erfunden, sondern aus dem bestehenden FE-Vertrag
  uebernommen (`caller_avatar`, nicht `caller_avatar_key` — das FE hatte hier
  bereits entschieden).
  Aufloesung an EINER Stelle (wie von `notes` gefordert): neue
  `VideoRoutes.getAuthClient()` (Wrapper analog zu
  `route_customization.go:getSettingsClient`, `registry.GetConnection("auth")`)
  und `resolveCallerIdentity(ctx, userID)` rufen `AuthServiceClient.GetUser`
  auf — ein bereits bestehender RPC, kein neuer. `callerDisplayName` bildet
  denselben Fallback wie das CONCAT_WS-Muster der HR-Lesepfade
  (`postgres_repository.go` in `biz/hr/*`): Vor- und Nachname wenn gesetzt,
  sonst E-Mail. Best-effort: schlaegt `getAuthClient` oder `GetUser` fehl,
  wird `slog.Error` mit `user_id` geloggt und mit leerem Namen/Avatar
  weitergemacht — der Anruf selbst wird nie abgebrochen, exakt der Nil-Mailer-
  artige Pfad, den `notes` verlangt. Die Aufloesung laeuft in DERSELBEN
  Goroutine wie die Broadcasts (vorher: eine Goroutine PRO Teilnehmer ohne
  Aufloesung; jetzt: eine Goroutine gesamt, Identitaet einmal aufgeloest,
  dann sequentiell an alle Teilnehmer verschickt) — verhindert N identische
  `GetUser`-Aufrufe bei einer Gruppen-Einladung und blockiert nicht die
  HTTP-Antwort von `HandleCreateCall` (Broadcast lief schon vorher async).
  `avatar_url` (der Spaltenname traegt laut Kommentar in `auth/service.go:344`
  tatsaechlich einen Objekt-Key, kein volles URL) wird 1:1 als
  `caller_avatar` durchgereicht — kein Presign hier, das FE loest den Key
  selbst auf (unveraendert zu jedem anderen Avatar-Consumer im Repo).
- gate: build ok (`go build -p 1 ./...` — Standard-`go build ./...` starb am
  Linker mit `runtime: cannot allocate memory` beim parallelen Linken
  mehrerer `cmd/*`-Binaries; System hatte >20 GB frei, also ein
  Sandbox-Job-Limit dieser Maschine, kein Code-Problem — mit `-p 1`
  reproduzierbar gruen) | vet ok (`-p 1`) | lint ok (`golangci-lint run
  ./internal/gateway/...` — 0 issues) | test ok (`go test -count=1
  ./internal/gateway/...` inkl. `TestOpenAPIRouteDrift` = 825 Routen gegen
  827 Pfade, unveraendert) | migration n.a. (keine neue Tabelle/Spalte) |
  openapi n.a. (kein neuer Pfad, `call.incoming` ist ein WS-Event, keine
  REST-Route)
- mutations-probe: `if len(parts) > 0` in `callerDisplayName` zu
  `if len(parts) >= 0` gedreht (Email-Fallback wird nie erreicht) → GENAU
  die zwei Subtests `TestCallerDisplayName/no_name_falls_back_to_email` und
  `.../blank_name_fields_fall_back_to_email` wurden rot, die vier anderen
  Subtests (beide Namen, nur Vorname, nur Nachname, nil-User) blieben gruen.
  Zurueckgedreht, Endgate danach erneut gelaufen und gruen.
- fehlerpfade im Test: `TestResolveCallerIdentity_AuthClientUnavailable_ReturnsEmpty`
  (Registry kennt "auth" gar nicht — `getAuthClient` schlaegt sofort fehl)
  und `TestResolveCallerIdentity_LookupFailure_ReturnsEmptyWithoutError`
  (Registry kennt "auth", aber die dahinterliegende Verbindung ist ein
  unerreichbarer Dummy — der RPC selbst schlaegt fehl). Beide liefern
  `("", "")` ohne Panic oder propagierten Fehler — das ist done_when Punkt 3.
- db-tests: keine — reine Gateway-Logik ohne neue Tabelle, kein
  `SkipIfNoDB`-Pfad in `internal/gateway`. `DATABASE_URL` war gesetzt, aber
  fuer dieses Paket ohne Wirkung.
- offen: Zwei Punkte fuer Luke.
  (1) **`./internal/video/...` aus `done_when` existiert nicht** — der
  Video-Businesslogik-Code liegt unter `internal/work/video`, die
  Broadcast-Aufloesung selbst sitzt komplett im Gateway
  (`internal/gateway/route_video_test.go`), wo sie laut `notes` auch
  hingehoert ("dort, wo der Broadcast gebaut wird"). `go test
  ./internal/work/video/... ./internal/server/...` liefen zur Kontrolle
  trotzdem gruen (unveraendert, da nicht angefasst).
  (2) **FE-`lean:`-Marker ist jetzt stale.** `useIncomingCallListener.ts:43-49`
  dokumentiert explizit, dass `caller_name`/`caller_avatar` serverseitig noch
  nicht aufgeloest werden — das stimmt ab diesem Commit nicht mehr. Der
  Marker + sein Fallback-Kommentar sollten in einer FE-Session entfernt
  werden (reines Aufraeumen, keine Logikaenderung noetig: das FE liest die
  Felder bereits korrekt, `?? data.caller_id` bleibt als harmloser
  Sicherheitsnetz-Fallback sinnvoll bestehen).

## Iteration 13 — fix-a-video-caller-identity — done — 2026-08-08 17:36
- commit: 651f7905
- verify vorgaenger: **BEFUND** an `e736f324` (Iteration 12,
  `a-video-caller-identity`). Fehlerklasse 2 (Stub-Wirkung: der neue Pfad kann
  in Produktion nie liefern). Die Broadcast-Goroutine in `HandleCreateCall`
  rief `resolveCallerIdentity(context.Background(), userID)` auf. Damit haengt
  weder Tenant noch User im Kontext. Belegkette, in dieser Reihenfolge
  nachgeschlagen:
  (a) `internal/gateway/registry.go:110-112` haengt an jede ausgehende
      gRPC-Verbindung `middleware.TenantOutboundUnaryInterceptor()`;
  (b) der setzt `x-tenant-id`/`x-user-id` NUR, wenn `GetTenantID(ctx)` bzw.
      `GetUserID(ctx)` etwas liefern (`internal/middleware/grpc_tenant.go:43-48`)
      — bei `context.Background()` also gar nicht;
  (c) auf der Gegenseite antwortet `TenantInboundUnaryInterceptor` ohne
      `tenant_id` mit `codes.Unauthenticated` (`grpc_tenant.go:73-75`, im
      Doc-Kommentar ausdruecklich so zugesichert).
  Folge: `AuthServiceClient.GetUser` schlaegt bei JEDER Anruferstellung fehl,
  `caller_name`/`caller_avatar` sind immer `""`, und der best-effort-Zweig
  loggt bei jedem Anruf ein `slog.Error`. Das FE zeigte weiter die rohe UUID —
  also genau der Zustand, den Iteration 12 beheben wollte. Zweiter Pfad
  desselben Problems (falls die Metadaten je durchkaemen): der Pool setzt die
  RLS-GUCs in `BeforeAcquire` aus `middleware.GetTenantID(ctx)`
  (`internal/database/postgres.go:60-84`) — ohne Tenant bleibt `app.tenant_id`
  leer und der `users`-SELECT liefert 0 Zeilen. Fix-Unit
  `fix-a-video-caller-identity` vorne im Backlog angelegt und in dieser
  Iteration abgearbeitet.
- gebaut: `HandleCreateCall` koppelt den Kontext jetzt mit
  `context.WithoutCancel(r.Context())` ab statt ihn wegzuwerfen — dasselbe
  Muster, das `internal/middleware/idempotency.go:164` fuer denselben Zweck
  schon verwendet (Goroutine ueberlebt den Handler, Werte bleiben, Cancel
  faellt weg). Der anonyme Goroutine-Rumpf wurde zur benannten Methode
  `broadcastCallIncoming(ctx, callID, callType, callerID, participantIDs)` —
  das ist die testbare Naht, an der der Kontext ankommt; ohne sie liesse sich
  die Regression nicht pruefen (`HandleCreateCall` bricht in Tests schon am
  fehlenden Video-Client ab). Verhalten sonst unveraendert: eine
  Identitaets-Aufloesung pro Anruf, danach sequentiell an alle Teilnehmer,
  best-effort bei Fehlschlag.
- gate: build ok (`go build -p 2 ./internal/gateway/... ./cmd/gateway/...`) |
  vet ok | lint ok (`golangci-lint run --config .golangci.yml
  ./internal/gateway/...` = 0 issues) | test ok (`go test -count=1
  ./internal/gateway/` inkl. `TestOpenAPIRouteDrift`) | migration n.a. |
  openapi n.a. (keine neue Route, `call.incoming` ist ein WS-Event) |
  rls-smoke n.a. (keine Tabelle/Policy angefasst)
- mutations-probe: `vr.wsHub.BroadcastCallIncoming(ctx, …)` in
  `broadcastCallIncoming` auf `context.Background()` gedreht →
  `TestBroadcastCallIncoming_PropagatesTenantContext` wurde rot mit
  `broadcast 0: GetTenantID() error = missing or invalid tenant_id in token`.
  Zurueckgedreht, Endgate danach erneut gelaufen und gruen. Der Test faengt
  also genau den Rueckfall, der die Iteration-12-Regression war.
- fehlerpfade im Test: der neue Test laeuft bewusst gegen `emptyRegistry()` —
  die Aufloesung SCHLAEGT FEHL und der Broadcast geht trotzdem an beide
  Teilnehmer raus, mit leerem `caller_name`. Damit ist der best-effort-Vertrag
  mitgeprueft, nicht nur der Happy Path. Zusaetzlich wird der Parent-Kontext
  vor dem Aufruf gecancelt (`cancel()` vor `WithoutCancel`), was die
  Produktionslage nachstellt: der HTTP-Handler ist zurueck, bevor die
  Goroutine laeuft. `ctx.Err() == nil` in der Assertion belegt, dass das
  Abkoppeln wirkt und nicht nur die Werte kopiert wurden.
- db-tests: keine — reine Gateway-Logik. `DATABASE_URL` war gesetzt;
  `go test -count=1 -v ./internal/gateway/` meldete **0 SKIPs** im ganzen
  Paket (gezaehlt via `grep -c -- "--- SKIP"`).
- offen: (1) Der FE-`lean:`-Marker in `useIncomingCallListener.ts:43-49` ist
  weiterhin stale und sollte in einer FE-Session weg — ab diesem Commit
  stimmt seine Aussage tatsaechlich nicht mehr (nach Iteration 12 allein
  stimmte sie faktisch noch, nur der Grund war ein anderer).
  (2) **Ungeprueft ist der echte Ende-zu-Ende-Pfad**: dass `GetUser` mit
  vorhandenem `x-tenant-id` fuer einen Anrufer aus dem eigenen Tenant
  wirklich Namen und Avatar liefert, ist hier nur aus dem Code abgeleitet
  (`AuthGRPCServer.GetUser` in `internal/server/grpc.go:140` hat keinen
  eigenen Permission-Guard, der `users`-SELECT ist rein RLS-gescoped). Ein
  echter Anruf zwischen zwei Accounts nach dem Merge ist die einzige harte
  Bestaetigung — der Aufwand dafuer gehoert nicht in eine Nacht-Iteration.
  (3) Andere `context.Background()`-Stellen im Gateway sind in dieser
  Iteration NICHT durchgesehen worden. Wenn dort weitere Goroutinen ausgehende
  gRPC-Aufrufe machen, haben sie denselben Defekt — das waere eine eigene
  Unit ("Grep `go func` + `context.Background` in `internal/gateway`"), keine
  Nebenbei-Aenderung.
