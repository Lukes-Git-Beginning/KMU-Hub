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

## Iteration 14 — a-video-meeting-recurrence — done — 2026-08-08 18:20
- commit: 6f5779ec
- vorspann: `651f7905` (Iteration 13) nachgeprueft. `git show --stat` = 4 Dateien,
  davon zwei Loop-Dateien; Code-Diff nur `route_video.go` (+
  `context.WithoutCancel`, Goroutine als benannte Methode) und ein neuer Test.
  Keine neue Route (`call.incoming` ist ein WS-Event) → OpenAPI zu Recht
  unberuehrt; keine Migration, keine Tabelle, kein `RequirePermission`. Handler
  ruft weiter ueber den gRPC-Client. Sauber, nichts nachzuziehen.
- praemisse (das Kernstueck dieser Unit): die Vermutung im Backlog — „vielleicht
  haengt ein Meeting schon am Kalendereintrag und die Serie ist damit geloest" —
  ist **halb** wahr, und die Haelfte entscheidet alles:
  - Die Spalte existiert: `meetings.calendar_event_id UUID` seit
    `000037_create_meetings.up.sql:14` (kein FK, nur Spalte + Index), im Proto
    als `Meeting.calendar_event_id = 12` und `CreateMeetingRequest.…= 7`, vom
    Client durchgereicht bis `postgres_repository.go:29`.
  - Sie ist aber **inert**: nichts liest sie. Kein Codepfad erzeugt beim
    Meeting-Anlegen einen Kalendereintrag oder umgekehrt, und die Rueckrichtung
    (`calendar_events.meeting_id`) existiert gar nicht.
  - `meetings.recurring_meeting_id` ist **keine** Seriendefinition, sondern eine
    Self-FK als Gruppierungsschluessel; einziger Leser ist
    `GetPreviousMeetingNotes` (`service.go:713`). Kein `rrule`/`recurrence` auf
    `meetings` oder im Meeting-Proto.
  Ergebnis: **nicht blocked.** Die Serie war nicht geloest — aber die
  Verknuepfung, ueber die sie zu loesen ist, lag fertig da. Also nicht `blocked`
  und auch nicht „Serie am Meeting neu definieren", sondern: die vorhandene
  Verknuepfung zum Tragen bringen.
- entscheidung gegen den Scope-Wortlaut: der Scope sagt „Serien-Definition am
  Meeting ergaenzen". Das haette `rrule` + `recurrence_end` auf `meetings`
  bedeutet — eine **zweite** Quelle fuer dieselbe Serie, direkt neben der
  Spalte, die schon auf den Kalendereintrag zeigt. Die Notiz derselben Unit
  verbietet genau das („nicht eine zweite daneben stellen"). Gebaut ist deshalb
  die Lesart, die beide Saetze erfuellt: die Regel bleibt am Kalendereintrag,
  das Meeting erbt sie ueber `calendar_event_id`. Spart Migration 298, spart
  Repository-Umbau, und Meeting- und Kalenderserie koennen nicht auseinander
  driften.
- gebaut:
  - `internal/work/meeting/occurrences.go` — `SeriesSource`-Interface (`Get` +
    `ListExceptions`, erfuellt von `*event.Service`), `WithSeriesSource`,
    `ListOccurrences(ctx, meetingID, tenantID, start, end) ([]MeetingOccurrence,
    truncated bool, error)`. Expandiert mit **`event.ExpandRecurrence`** —
    dieselbe Funktion, die `ListEventsInRange` benutzt, keine zweite
    Implementierung.
  - `internal/work/event/service.go` — `ListExceptions` als Passthrough auf das
    Repo, damit das Meeting dieselben Ausnahmen sieht wie der Kalender
    (Absagen + verschobene Starts) und nicht am Service vorbei ins Repository
    greifen muss.
  - Proto: `ListMeetingOccurrences` + `MeetingOccurrence`/Request/Response
    (`items`/`total`/`truncated`), regeneriert. Der `.pb.go`-Diff hat 332
    Loeschungen — das sind ausschliesslich verschobene `msgTypes`-Indizes durch
    die drei mittig eingefuegten Messages plus gofmt-Ausrichtung, kein
    Generator-Wechsel (geprueft: keine `protoc-gen-go v…`-Zeile im Diff).
  - `video_grpc.go`: RPC + `mapMeetingError` um `ErrInvalidRecurrence` →
    `FailedPrecondition` und `ErrSeriesUnavailable` → `Unavailable`.
  - Gateway: `GET /api/v1/meetings/{id}/occurrences?start&end`, Validierung wie
    beim Kalender-Pendant (`HandleListEventsInRange`), ueber den gRPC-Client.
- obergrenze: `maxMeetingOccurrences = 200`, `truncated` im Response. **Befund
  nebenbei:** der Kalender selbst hat *keine* Kappung — `ExpandRecurrence` wird
  in `ListEventsInRange` nur durch das angefragte Fenster begrenzt, und
  `HandleListEventsInRange` prueft die Fensterbreite nicht. `FREQ=HOURLY` +
  Zehn-Jahres-Fenster ist dort heute erlaubt. Nicht in dieser Unit gefixt (der
  Kalender war nicht ihr Gegenstand); gehoert als eigene Unit nachgelegt.
- semantik, die eine Entscheidung war: Zeitpunkt der Instanz kommt vom Kalender,
  **Laenge** vom Meeting (`ScheduledEnd - ScheduledStart`), weil das Meeting das
  Wiederholte ist. Ausserdem wird `recurrence_end` des Events als harte
  Fensterklammer gezogen, auch wenn der Aufrufer weiter fragt.
- gate: build ok (`go build ./...`) | vet ok | lint ok (`golangci-lint run` ueber
  gateway/meeting/event/server/cmd-work = 0 issues) | test ok
  (`./internal/gateway/` inkl. `TestOpenAPIRouteDrift`, `./internal/work/meeting/`,
  `./internal/work/event/`, `./internal/server/`) | migration n.a. (keine neue
  Tabelle/Spalte) | permission-seed n.a. (Route nur `authMiddleware`, wie die
  uebrigen Meeting-Reads) | openapi: Pfad ergaenzt **und**
  `swagger-cli validate` = valid.
- statuscode-falle, fast reingelaufen: die OpenAPI-Antwort stand zuerst auf 412.
  `respondGRPCError` mappt `FailedPrecondition` in diesem Repo aber auf **409**
  (`internal/gateway/helpers.go:53`, festgehalten in `helpers_test.go:23`).
  Korrigiert, bevor die Spec das Gegenteil des Handlers behauptet haette.
- mutations-proben (drei, alle rot geworden, alle zurueckgedreht):
  (A) Kappung deaktiviert (`if len(occurrences) == max` → `if false`) →
  `TestListOccurrences_CapsInstances` rot („the caller must learn the window was
  cut short"). (B) Absage-Skip deaktiviert → `…_SkipsCancelledAndHonoursMovedStart`
  rot mit „should have 3 item(s), but has 4". (C) Pflichtparameter-Guard im
  Gateway deaktiviert → `…_Validation` rot in drei Subtests (Fehlertext kippte
  auf „invalid start time format", d. h. ohne Guard laeuft ein leerer Parameter
  bis in den Parser). Endgate nach dem Zurueckdrehen erneut gelaufen und gruen;
  `grep -c "if false"` = 0 in beiden Dateien.
- fehlerpfade im Test: neun Stueck als Subtests — Fenster nicht positiv, Series
  Source nicht verdrahtet, unbekanntes Meeting, **fremder Tenant** (muss
  `ErrNotFound` sein, nicht „nicht wiederkehrend" — sonst verraet die Antwort
  die Existenz), Meeting ohne Kalender-Link, Event ohne Regel (auch `"   "`),
  unparsebare Regel, Event-Lookup-Fehler, Exception-Lookup-Fehler. Der letzte
  bricht bewusst ab statt best-effort weiterzumachen: eine halb geladene
  Ausnahmenliste zeigt abgesagte Termine als lebende Meetings — der Kalender
  loggt dort nur eine Warnung, das ist fuer eine Meeting-Liste zu wenig.
- db-tests: die neuen Tests sind stub-basiert (kein DB-Bedarf). Gegenprobe
  trotzdem gemacht: ohne `DATABASE_URL` melden `./internal/work/meeting/` und
  `./internal/work/event/` zusammen **2 SKIPs**, mit gesetzter
  `DATABASE_URL` (`kmuhub_app`) **0** — die DB-gebundenen Tests der beruehrten
  Pakete laufen real und sind gruen.
- offen: (1) Die fehlende Kappung im Kalender selbst (oben). (2) Es gibt weiterhin
  keinen Weg, `calendar_event_id` **aus dem Meeting heraus** zu setzen — der
  Client muss es beim Anlegen mitgeben, `UpdateMeeting` kann es nicht nachtragen.
  Wer ein bestehendes Meeting nachtraeglich zur Serie machen will, kann das ueber
  die API nicht. Eigene Unit, keine Nebenbei-Aenderung. (3) Frontend kennt die
  Route noch nicht (Loop baut Backend).

## Iteration 15 — a-notifications-channel-exposure — done — 2026-08-08 19:05

- commit: 3e094778
- verify vorgaenger: Commit `6f5779ec` (Iteration 14, Meeting-Occurrences) gegen
  die sechs Fehlerklassen geprueft. `git show --stat` = 15 Dateien: Handler geht
  ueber `client.ListMeetingOccurrences` (gRPC-Client, kein Direct-Svc-Bypass),
  `.proto` UND `.pb.go`/`_grpc.pb.go` im selben Commit regeneriert, keine neue
  Tabelle/Migration (also keine Tenant-/RLS-Frage), kein neuer
  `RequirePermission`-Guard, neuer Pfad `/meetings/{id}/occurrences` steht in
  `openapi.yaml` im selben Commit. Sauber, nichts nachzuziehen.
- praemisse (Notiz verlangte das zuerst): `internal/notification/preference/`
  UND `/api/v1/notifications/preferences` existieren bereits — aber die
  Kanalwahl selbst fehlte nachweislich. `models.NotificationPreference` /
  `notification_preferences` (Migration 000022) kannten nur `in_app` und
  `desktop_push`; `DeliveryDecision` im Service ebenso. Tiefer nachgesehen als
  die Notiz verlangte, weil die Praemisse sonst nur halb gilt: der Dispatcher
  (`internal/notification/delivery/dispatcher.go`) liefert bis heute NUR
  in_app/desktop_push (WS) und Teams/Slack-Integration-Forwarding aus
  `cmd/notification/main.go` — keine Zeile Email- oder SMS-Versand fuer
  Notifications. Trotzdem nicht `blocked`: die Notiz selbst sagt explizit
  "Zustellung selbst bleibt beim Dispatcher" — die Kanalwahl ist als reines
  Praeferenz-Feld gescopt, die Zustellung ist bewusst eine spaetere Unit. Ein
  Treiber fuer eine echte Email-Zustellung existiert inzwischen sogar
  (`internal/email/systemmail`, aus Lauf 3 Iteration 50 fuer den
  Berichte-Scheduler gebaut) — aber ihn hier anzuschliessen waere Scope-Creep
  ueber die Notiz hinaus gewesen, nicht Mechanik.
- gebaut: `email`/`sms` als zusaetzliche Spalten auf `notification_preferences`
  (Migration 000304, `email DEFAULT true`, `sms DEFAULT false` — Default
  spiegelt den bisherigen faktischen Zustand: In-App/Push liefen immer, Email
  war nie aktiv verweigert, SMS gibt es nicht). Additiv durch die gesamte
  Kette gezogen: `models.NotificationPreference`, `postgres_repository.go`
  (alle vier Queries), `preference.Service.DeliveryDecision` +
  `Evaluate()`, `notification.proto`
  (`NotificationPreferenceInfo`/`UpdateNotificationPreferenceRequest`, Felder
  10/11 bzw. 7/8, `make proto` fuer notification.proto neu generiert — Diff nur
  Feldanhang, keine Generator-Versionszeile), `notification_grpc.go`
  (`toPreferenceInfo`, `UpdateNotificationPreference`), Gateway
  `updatePreferenceRequest` + `route_notification.go`, `openapi.yaml`
  (`NotificationPreference`/`UpdateNotificationPreferenceRequest`-Schemas).
  KEINE neue Route — genau wie die Notiz verlangt.
  ENTSCHEIDUNG die ueber ein reines Copy-Paste hinausgeht: die "alle Kanaele
  deaktiviert -> Deliver=false"-Pruefung in `Evaluate()` pruefte bisher nur
  `!InApp && !DesktopPush`. Unveraendert gelassen haette das bedeutet, dass ein
  Nutzer, der ausschliesslich Email waehlt (in_app=false, desktop_push=false,
  email=true), von der Entscheidungsebene als "nicht zuzustellen" markiert
  wird — sobald irgendwann ein Email-Callback drangehaengt wird, wuerde diese
  Preference ihn sofort wieder stumm schalten. Die Pruefung um `!Email &&
  !SMS` erweitert, damit die Kanalwahl ab dem ersten Tag korrekt in die
  Deliver-Entscheidung einfliesst, auch wenn heute noch niemand `decision.Email`
  liest.
- ROOT-CAUSE-FUND beim Testen, nicht geraten: `UpsertPreference`s
  `ON CONFLICT (tenant_id, user_id, event_type_key) WHERE event_type_key IS
  NOT NULL` passte seit Migration 000124 (tenant_id NOT NULL + RLS) auf KEINEN
  echten Unique-Index mehr — der Index aus Migration 000022 kennt nur
  `(user_id, event_type_key)`, tenant_id wurde nie nachgezogen. Postgres:
  "no unique or exclusion constraint matching the ON CONFLICT specification".
  Das bedeutet: jedes Upsert einer event-type-spezifischen Preference ist seit
  000124 in Produktion mit einem 500er gescheitert — nie aufgefallen, weil es
  vor dieser Iteration keinen einzigen DB-gestuetzten Test fuer
  `UpsertPreference` gab (nur den Mock-basierten `service_test.go`). Fix in
  derselben Datei, die ich ohnehin anfasse: Migration 000305 ersetzt den Index
  durch `(tenant_id, user_id, event_type_key)` — echte Obermenge des alten
  Index, kann also keine bestehende Zeile verletzen. Root-Cause-Fix statt Guard
  um den Aufrufer, wie von der Lean-Regel verlangt.
  GEFUNDEN, NICHT BEHOBEN (ausserhalb Scope): derselbe Tenant-Blindfleck
  besteht auch bei `idx_notification_preferences_module_default`
  (`(user_id, module_id) WHERE event_type_key IS NULL`) — dort loest er aber
  aktuell keinen Fehler aus, weil `UpsertPreference` fuer den
  Modul-Default-Fall gar kein `ON CONFLICT` definiert (ein zweites Upsert
  desselben Moduls wuerde also mit einem rohen Duplicate-Key-Fehler statt
  einem Update enden — ein eigener, vorbestehender Bug, nicht durch diese
  Iteration ausgeloest). Kandidat fuer eine eigene Fix-Unit.
- abweichung vom woertlichen done_when: "Unbekannter Kanal liefert 400" liess
  sich nicht sinnvoll umsetzen, weil dieses Repo Kanaele als benannte
  Booleans fuehrt (`in_app`, `desktop_push`, jetzt `email`, `sms`), nicht als
  generischer String/Array — eine Kanal-Enum-Validierung haette eine zweite,
  inkonsistente Wire-Form neben dem bestehenden Muster erzwungen. Bewusst
  NICHT gebaut (Lean-Code-Leiter: Muster wiederverwenden statt neu erfinden).
  Bestehendes Verhalten fuer unbekannte JSON-Felder gilt unveraendert
  (werden von `decodeAndValidate` ignoriert, kein 400 — wie bei allen anderen
  Preference-Feldern auch).
- risiko, dokumentiert statt versteckt: Das Preference-PUT ersetzt den ganzen
  Satz (bestehendes Muster, siehe `a-users-preferences`-Notiz). Solange das FE
  `email`/`sms` nicht mitschickt, faellt jedes PUT von einer noch unwissenden
  FE-Version auf `email=false, sms=false` zurueck (Go-Zero-Value bei
  JSON-Absenz) — ein stiller Abfall vom DB-Default `email=true`. Wirkt sich
  aktuell auf NICHTS aus (kein Dispatcher-Callback liest `decision.Email`),
  wird aber real in dem Moment, in dem sowohl FE-Formular als auch
  Email-Zustellung nachgezogen sind. Luke sollte das beim FE-Anschluss wissen.
- fehlerpfade: `GetEventTypePreference`/`GetModuleDefault` liefern weiterhin
  `ErrPreferenceNotFound` unveraendert (keine Verhaltensaenderung an diesen
  Pfaden ausser den zwei zusaetzlichen Spalten).
- mutations-proben (zwei, beide rot geworden, beide zurueckgedreht): (A) die
  neue `!Email && !SMS`-Klausel im Deliver-Gate entfernt ->
  `TestEvaluateEmailOnlyChannelStillDelivers` rot ("Should be true"). (B) in
  `GetEventTypePreference` die Scan-Reihenfolge von `&pref.Email, &pref.SMS`
  auf `&pref.SMS, &pref.Email` vertauscht (Spaltennamen in der SELECT-Liste
  unveraendert) -> `TestPostgresRepository_EmailSMSRoundTrip` rot ("email must
  round-trip as false, not the column default"). Beide zurueckgedreht, Endgate
  danach erneut gruen gelaufen.
- gate: build (`internal/notification/... internal/server/... internal/gateway/...
  cmd/notification/... cmd/gateway/...`, `-p 2`) ok | vet ok | golangci-lint
  **0 issues** | migration: 000304 + 000305 angewendet
  (`migrate -path backend/migrations -database
  postgres://kmuhub:kmuhub_dev@localhost:5432/kmuhub?sslmode=disable up`,
  lokaler Hostname-Ersatz fuer `postgres` aus `.env`, siehe GATE-COMMANDS.md) |
  Tests mit `DATABASE_URL` (Rolle `kmuhub_app`): `internal/notification/...`
  (7 Pakete) und `internal/gateway/` (inkl. `TestOpenAPIRouteDrift`, 826
  Routen/828 Pfade, keine Drift) alle **ok**, 0 Skips beobachtet | openapi:
  Felder ergaenzt (keine neue Route) **und** `swagger-cli validate` = valid |
  rls-smoke: kein neues Tabellenschema, aber ein bestehender RLS-Index auf
  `notification_preferences` wurde per Migration ersetzt ->
  `TestTenantIsolation_Notifications` nach beiden Migrationen erneut gruen,
  plus der neue `TestPostgresRepository_EmailSMSRoundTrip` laeuft explizit
  unter `testutil.WithTenantCtx(..., TenantA)` (RLS-INSERT haette sonst schon
  am Schreiben abgelehnt — hat es beim ersten Versuch mit
  `context.Background()` auch getan, SQLSTATE 42501, dann korrigiert).
- offen:
  - Email-/SMS-Zustellung selbst bleibt unwired (Notiz-Vorgabe). Sobald ein
    Dispatcher-Callback fuer Email gebaut wird (Kandidat: `internal/email/
    systemmail.Sender`, bereits fuer Berichte/Dunning im Einsatz), zusaetzlich
    Nutzer-E-Mail-Adresse aufloesen und `decision.Email` konsumieren.
  - FE-Formular kennt `email`/`sms` noch nicht — siehe "risiko" oben, vor dem
    FE-Anschluss lesen.
  - `idx_notification_preferences_module_default` hat denselben
    Tenant-Blindfleck wie der jetzt gefixte Index, aber ohne aktiven Fehler
    (siehe "GEFUNDEN, NICHT BEHOBEN" oben) — eigene Fix-Unit-Kandidat.

## Iteration 16 — a-kontakte-xlsx-import — done — 2026-08-08 19:35
- commit: 4baf1304
- verify vorgaenger: sauber. `3e094778` (Iteration 15, notification channel
  exposure) gegen die Fehlerklassen geprueft: `route_notification.go` fasst
  keine neue Route an (nur additive Felder auf dem bestehenden
  Update-Preference-Request), kein neuer `RequirePermission`-Guard, beide
  Migrationen (000304, 000305) haben up UND down gefuellt, Migration 000305
  ist eine reine Index-Verbreiterung (Obermenge des Altindex, kann keine
  Bestandszeile verletzen), `.proto` UND `.pb.go` im selben Commit
  regeneriert (kein `.grpc.pb.go`-Diff noetig, da kein neuer RPC). `git merge
  origin/main` war "Already up to date".
- gebaut: XLSX-Import fuer Kontakte, viertes Format neben CSV/vCard/Preview.
  Neuer RPC `ImportContactsXLSX` im `crm.proto` (Request-Form identisch zu
  `ImportContactsCSVRequest`: `file_content`, `field_mapping`, `visibility`,
  `merge_by_email`, `user_id`), `crm.pb.go`/`crm_grpc.pb.go` im selben Commit
  regeneriert (Diff gross, aber wie in Iteration 14 belegt: nur verschobene
  `msgTypes`-Indizes durch die mittig eingefuegte Message, keine
  Generator-Versionszeile geaendert). Route `POST /api/v1/contacts/import/xlsx`
  hinter dem bestehenden `contactImport`-Guard (`contacts:write` ODER
  `crm:import:run`, additiv, keine neue Guard-Kombination) — Handler geht
  ueber `crmv1.CRMServiceClient`, keine direkte Service-Instanz.
  `ImportService.ImportXLSX` (neu in `internal/email/contact/import_service.go`)
  liest NUR das erste Arbeitsblatt via `excelize.Rows()`-Streaming-Iterator
  (kein `GetRows()`, das haette das ganze Blatt vorab in den Speicher
  geladen), erste Zeile als Kopfzeile — ab da laeuft exakt derselbe Pfad wie
  CSV: `extractFieldsFromRow`, `importSingleContact`, `resolveCompany` mit
  demselben `companyCache`. Keine zweite Auto-Erkennung, keine zweite
  Firmen-Aufloesung, wie von `notes` verlangt.
  Zeilenobergrenze `maxXLSXImportRows = 5000` (`lean:`-Marker mit
  Upgrade-Pfad): anders als CSV/vCard, deren Zeilenzahl implizit durch die
  10-MB-Multipart-Grenze im Gateway gedeckelt ist, entkoppelt die
  XLSX-Kompression Dateigroesse und Zeilenzahl fast vollstaendig — genau der
  Fall, den die Notiz mit "eine 50-MB-Arbeitsmappe darf den Dienst nicht
  umbringen" meint. Ueberschrittene Zeilen werden still uebersprungen (kein
  Fehler, ein `slog.Warn`), der Import laeuft mit dem Rest weiter.
  Neuer Sentinel `ErrInvalidXLSX` (`errors.go`), im `crm_grpc.go`-Handler
  `ImportContactsXLSX` explizit auf `codes.InvalidArgument` gemappt (anders
  als die bestehenden CSV/VCard-Handler, die JEDEN Importfehler pauschal auf
  `codes.Internal` legen — siehe "offen"). `respondGRPCError` setzt
  `InvalidArgument` auf 400 (`helpers.go:51`, bestehendes Muster). Pfad in
  `api/openapi.yaml` inklusive `ContactsImportXLSXForm`-Schema.
- gate: build ok (`-p 2`, `./internal/... ./cmd/crm/... ./cmd/gateway/...`) |
  vet ok | lint ok (`golangci-lint run` ueber
  contact/server/gateway = 0 issues) | test ok (`internal/email/contact`
  25 PASS/0 SKIP inkl. 6 neuer XLSX-Tests, `internal/server` gruen,
  `internal/gateway` gruen inkl. `TestOpenAPIRouteDrift` = 827 Routen gegen
  829 Pfade) | migration n.a. (keine neue Tabelle/Spalte, `excelize` steht
  bereits in `go.mod`) | openapi `swagger-cli validate` = valid | rls-smoke
  n.a. (kein Tabellenschema angefasst)
- mutations-probe: `dataRows >= maxXLSXImportRows` auf
  `dataRows >= maxXLSXImportRows+1000` aufgeweicht (Deckel wirkungslos) →
  GENAU `TestImportXLSX_RowCapTruncatesImport` wurde rot (5050 statt 5000
  importiert), die vier XLSX-Nachbartests und alle CSV/vCard-Tests blieben
  gruen. Zurueckgedreht, Endgate danach erneut gelaufen und gruen.
- fehlerpfade im Test: korrupte Datei (kein gueltiges ZIP/XLSX) →
  `ErrInvalidXLSX`, ungueltige E-Mail in einer Zeile → 1 Skip mit korrektem
  Workbook-Zeilenindex im Fehlerobjekt (Zeile 2 = erste Datenzeile, Kopfzeile
  ist Zeile 1 — bewusst die echte Excel-Zeilennummer, nicht die
  CSV-eigene "Header = Zeile 0"-Zaehlweise, die in diesem Repo ohnehin
  zwischen Lese- und Importfehler-Pfad um eins auseinanderlaeuft).
- offen: Drei Punkte fuer Luke.
  (1) **Kein FE-Konsument.** Der Import-Wizard im Kontakte-Modul kennt XLSX
  als Dateityp noch nicht — der Wire-Vertrag steht in `openapi.yaml` unter
  `ContactsImportXLSXForm`, identisch zum CSV-Formular bis auf den Dateityp.
  (2) **Vorbestehende Inkonsistenz gefunden, nicht angefasst:** die
  Geschwister-RPCs `ImportContactsCSV`/`ImportContactsVCard` in `crm_grpc.go`
  mappen JEDEN Fehler (auch eine kaputte Datei) auf `codes.Internal` statt
  `codes.InvalidArgument` — der neue XLSX-Pfad macht es richtig
  (`ErrInvalidXLSX` -> `InvalidArgument` -> 400), die beiden aelteren Pfade
  liefern bei derselben Fehlerklasse weiterhin 500. Root-Cause-Fix waere eine
  Zeile in beiden bestehenden Handlern, aber ausserhalb des Scopes dieser
  Unit (nicht die von mir angefasste Funktion) — Kandidat fuer eine eigene,
  sehr kleine Fix-Unit.
  (3) **Zweiter, unabhaengiger Import-Pfad existiert bereits:** `internal/server/email_grpc.go`
  spiegelt `ImportContactsCSV`/`ImportContactsVCard` fuer den `email`-Service
  unter `/api/v1/email/contacts/import/*` (eigenes Proto, eigener Handler,
  gleiche `ImportService` dahinter). Diese Unit hat dort bewusst NICHTS
  ergaenzt — die Backlog-`sources` nennen nur die CRM-Seite
  (`route_crm_contacts.go`), und ein zweiter XLSX-Pfad dort waere Scope-Creep
  gewesen. Falls XLSX auch fuer den E-Mail-Kontaktimport gewuenscht ist, ist
  das eine eigene, sehr aehnliche Unit.

## Iteration 17 — a-crm-dialer-lead-link — done — 2026-08-08 20:10
- commit: 66afd509
- verify vorgaenger: sauber. `4baf1304` (Iteration 16, XLSX-Import) gegen die
  Fehlerklassen geprueft: Handler geht ueber `crmv1.CRMServiceClient`, keine
  direkte Service-Instanz; `.proto`, `.pb.go` UND `.grpc.pb.go` im selben
  Commit regeneriert; kein neuer `RequirePermission`-Guard (wiederverwendet
  den bestehenden `contactImport`-Guard); keine Migration noetig; Route in
  `route_crm.go` UND `api/openapi.yaml` registriert. `git merge origin/main`
  war "Already up to date".
- gebaut: Dialer-Ergebnisse mit Rueckrufwunsch heben den verknuepften
  Kontakt in den CRM-Lead-Funnel. Neue RPC `PromoteContactToLead`
  (`crm.proto`, service-zu-service, bewusst OHNE HTTP-Gateway-Route — kein
  FE-Konsument, done_when verlangt keinen openapi.yaml-Eintrag), Handler in
  `internal/server/crm_grpc_leads.go` (gleiche Datei wie `ConvertLead` etc.,
  gleiches Muster). Business-Logik in einer neuen Service-Methode
  `contact.Service.PromoteFromDialerCallback` (`lead.go`): laedt den
  Kontakt, rechnet bei Rueckrufwunsch die Bewertung ueber `ComputeLeadScore`
  mit `LeadSourceDialer` neu, setzt `lifecycle_stage=lead`,
  `lead_source=dialer` per erweitertem `LeadPatch{Source,Score}` auf dem
  bestehenden `UpdateLead`-Repository-Pfad (`postgres_lead.go`, zwei neue
  SET-Klauseln, keine neue Query).
  **Der zentrale Designentscheid dieser Unit:** die Note im Backlog verlangt
  explizit "ein Kontakt, der schon `customer` ist, wird NICHT auf `lead`
  zurueckgestuft". Das Datenmodell (`models/contact.go:29-31`) macht
  `customer` zum STANDARDWERT jedes gewoehnlichen, nie ueber die Lead-Inbox
  angelegten Kontakts — es gibt im Code keine Unterscheidung zwischen "echter
  zahlender Kunde" und "nie angefasster Standardkontakt". Die Guard-Klausel
  ist trotzdem woertlich umgesetzt (Kontakt bei `customer` unangetastet
  zurueckgegeben, kein Fehler). Zusaetzlich, ueber die explizite Anforderung
  hinaus, denselben Schutz auf `qualified` ausgedehnt: `ConvertLead`/
  `UpdateLead` bewegen die Stage in diesem Repo nirgends rueckwaerts, und ein
  Rueckruf sollte eine bereits qualifizierte Sales-Stage nicht stillschweigend
  auf `lead` zuruecksetzen. Praktische Konsequenz, siehe "offen".
  Verdrahtung im Dialer (`service.go`, `LogCallOutcome`): direkt neben dem
  bestehenden `CreateCallActivity`-Bestcase-Aufruf, nur wenn
  `contactStatus == ContactStatusCallback` UND die echte Kontakt-ID ueber die
  Campaign-Contact-Verknuepfung tatsaechlich aufgeloest werden konnte (neues
  Flag `realContactResolved` — bei Aufloesungsfehler faellt der Code auf die
  Campaign-Contact-ID zurueck, und genau DIE darf niemals als CRM-Kontakt
  befoerdert werden, das waere ein falscher Kontakt). `CRMBridge`-Interface
  um `PromoteToLead(ctx, contactID)` erweitert, `GRPCCRMBridge`-Implementierung
  ruft die neue RPC. Aufruf best-effort/non-fatal wie `CreateCallActivity` —
  ein Anrufergebnis muss loggbar bleiben, auch wenn die Lead-Promotion
  fehlschlaegt.
  Idempotenz (done_when-Kriterium): die Bewertung wird immer als absolutes
  SET geschrieben, nie inkrementell — zweimal denselben Callback verarbeiten
  liefert denselben Score, dieselbe Source, dieselbe Stage. Test
  `TestPromoteFromDialerCallback_IdempotentOnReplay` belegt das explizit.
- gate: build ok (`-p 2`, dialer/crm/server/gateway/cmd-dialer/cmd-crm/cmd-gateway)
  | vet ok | lint ok (`golangci-lint run` ueber dialer/crm/server = 0 issues)
  | test ok, DATABASE_URL gesetzt, 0 Skips (`internal/dialer` inkl. 2 neuer
  Tests, `internal/crm/contact` inkl. 3 neuer Tests, alle uebrigen
  `internal/crm/*`-Pakete, `internal/server`, `internal/gateway` inkl.
  `TestOpenAPIRouteDrift` — unveraendert, da keine neue HTTP-Route) |
  migration n.a. (keine neue Tabelle/Spalte) | openapi n.a. (service-zu-
  service-RPC, kein Gateway-Pfad) | rls-smoke n.a. (kein Tabellenschema
  angefasst)
- mutations-probe: Guard-Zeile
  `if current.LifecycleStage == LifecycleQualified || ... == LifecycleCustomer`
  auf `if false` verkuerzt (Guard wirkungslos) →
  `TestPromoteFromDialerCallback_NeverDemotesQualifiedOrCustomer` wurde rot
  (qualified- UND customer-Kontakt wurden faelschlich auf `lead` gehoben),
  die beiden Nachbartests (`_LiftsLeadAndRecomputesScore`,
  `_IdempotentOnReplay`) blieben gruen. Zurueckgedreht, Endgate danach
  erneut gelaufen und gruen.
- fehlerpfade im Test: Rueckruf-Outcome ohne aufgeloeste Kontakt-ID (Fallback
  auf Campaign-Contact-ID) → `PromoteToLead` wird NICHT aufgerufen
  (`realContactResolved=false`); Nicht-Rueckruf-Outcome → `PromoteToLead`
  wird NICHT aufgerufen; Kontakt bei `qualified`/`customer` → Aufruf liefert
  keinen Fehler, aber keine Aenderung.
- offen: Zwei Punkte fuer Luke.
  (1) **Praktische Reichweite der Guard-Klausel.** Weil `customer` der
  Standardwert jedes gewoehnlichen Kontakts ist (nicht nur echter Kunden),
  greift die Promotion in der Praxis vor allem bei Kontakten, die bereits
  ueber die Lead-Inbox angelegt wurden (Stage `lead`, z. B. aus einem
  CSV-Import) — ein Dialer-Ziel, das nie durch die Lead-Inbox lief, bleibt
  trotz Rueckrufwunsch bei `customer` und wird NICHT befoerdert. Das ist die
  woertliche Umsetzung der Backlog-Note, aber falls der eigentliche
  Produktwunsch ist, dass JEDER Kaltakquise-Kontakt mit Rueckrufwunsch zum
  Lead wird, braucht es zuerst eine Entscheidung, wie "echter Kunde" von
  "nie qualifizierter Standardkontakt" unterschieden werden soll — das ist
  eine Datenmodell-Frage, keine Iteration.
  (2) **`qualified`-Schutz ueber die Anforderung hinaus ergaenzt** (siehe
  oben) — falls das nicht gewuenscht ist, ist es eine einzelne Zeile in
  `lead.go` (`PromoteFromDialerCallback`), leicht rueckgaengig zu machen.

## Iteration 18 — a-produktion-material-availability — done — 2026-08-08 19:45
- commit: 2e0f3bea
- verify vorgaenger: sauber. 66afd509 (a-crm-dialer-lead-link, Iteration 17)
  gegen alle acht Fehlerklassen geprueft: GRPCCRMBridge.PromoteToLead geht
  ueber b.client.PromoteContactToLead (gRPC-Client); der neue RPC-Handler in
  crm_grpc_leads.go ruft s.contactService.PromoteFromDialerCallback innerhalb
  des CRM-Dienstes selbst (kein Layer-Bypass); .proto, .pb.go UND
  .grpc.pb.go regeneriert; kein neuer RequirePermission-Guard, keine
  Migration; bewusst kein openapi.yaml-Eintrag (service-zu-service-RPC ohne
  Gateway-Route); Kontext wird durchgereicht (ctx, nicht Background()). git
  merge origin/main war "Already up to date".
- gebaut: GET /api/v1/produktion/orders/{id}/material-availability.
  Praemisse widerlegt und dabei eine tiefere Luecke gefunden:
  production_orders.bom_id existiert seit Migration 000187 als DB-Spalte,
  aber der Go-Stack hat sie nie gelesen/geschrieben. Ohne diese Luecke waere
  der Endpunkt fuer jeden Auftrag leer gewesen, also durchgezogen statt
  blocked: ProductionOrder.BomID *uuid.UUID, INSERT/UPDATE/SELECT in
  postgres_repository.go, CreateOrderInput.BomID/UpdateOrderInput.BomID
  (validieren BOM-Existenz via repo.GetBOM, sonst ErrBOMNotFound, jetzt auch
  in mapProduktionError gemappt), Proto bom_id an ProductionOrder/
  CreateOrderRequest/UpdateOrderRequest (optional, additiv), Gateway-Wire an
  beiden Order-Bodies. Keine neue Migration noetig.
  Zweite Praemisse widerlegt: production_bom_items hat KEIN SKU-Feld, nur
  material_name. Bewusste Abweichung: Abgleich laeuft ueber material_name
  gegen den inventar-Item-Namen (case-insensitiv/getrimmt,
  normalizeMaterialName), gleiche Kernsemantik wie der SKU-Abgleich in
  einkauf/inventar_adjuster.go, nur auf dem Feld das wirklich existiert.
  Neu: InventarLookup-Interface + GRPCInventarLookup
  (internal/produktion/inventar_lookup.go, Muster von
  einkauf.GRPCInventarAdjuster) — EIN ListItems-Aufruf (PageSize:200, die
  von inventar.Service.ListItems selbst erzwungene Obergrenze) statt N
  Einzelaufrufen, danach Client-seitiges Matching. Service.WithInventarLookup
  in cmd/produktion/main.go mit demselben lazy-connect/non-fatal-Muster wie
  cmd/einkauf/main.go verdrahtet.
  Neue Service-Methode GetMaterialAvailability
  (internal/produktion/material_availability.go): verlangt BomID != nil
  (sonst ErrOrderHasNoBOM, FailedPrecondition/409), loest alle
  Positionsnamen in einem Batch auf, RequiredQuantity = item.Quantity *
  order.Quantity. Nicht gematchte Position ODER fehlgeschlagener/nicht
  konfigurierter Lookup -> Available/ShortfallQuantity bleiben nil (kein
  Fehler fuer die gesamte Anfrage, Architekturregel 8). Shortfall bei
  Ueberschuss auf 0 geklemmt. Gateway-Handler geht ueber
  ProduktionServiceClient, Route hinter dem bestehenden
  produktion:order/read-Key.
- gate: build ok (-p 2) | vet ok | lint ok (0 issues) | test ok (produktion
  40/40 PASS 0 SKIP inkl. TestTenantIsolation_Produktion real gegen DB,
  server ok, gateway ok inkl. TestOpenAPIRouteDrift = 828 Routen gegen 830
  Pfade) | migration n.a. (Spalte existierte bereits) | openapi
  swagger-cli validate ok | rls-smoke n.a. (kein Schema angefasst)
- mutations-probe: zwei Stueck. (1) Shortfall-Klemme aufgeweicht
  (if shortfall < -999999 statt < 0) -> GENAU
  TestService_GetMaterialAvailability_SufficientStockHasZeroShortfall wurde
  rot, sechs Nachbartests blieben gruen. (2) `if info, ok := stock[...]; ok`
  auf immer-wahr gedreht -> GENAU die drei Tests rot, die den Unknown-Pfad
  pruefen, vier andere blieben gruen. Beide zurueckgedreht, Endgate danach
  erneut gruen (40/40, 0 Skips).
- db-tests: TestTenantIsolation_Produktion (unveraendert) lief real gegen
  die lokale DB — 0 Skips im Paket produktion (40 PASS). Kein eigener
  DB-Test fuer bom_id-Persistenz (kein DB-Test fuer BOM existierte vorher im
  Paket ueberhaupt); Spaltenverdrahtung ist durch Build+Vet+Lint sowie die
  neuen Mock-Repo-Tests belegt, nicht durch einen echten Repository-Roundtrip.
- offen: Vier Punkte fuer Luke. (1) Kein FE-Konsument — weder Order-Formular
  noch ProduktionDetailModals.tsx kennen bom_id oder den neuen Endpoint,
  getMaterialAvailability im FE bleibt bis zu einer FE-Session der
  deterministische Pseudo-Wert. (2) BOM-CRUD hatte vorher keinerlei
  Testabdeckung im Paket produktion — der Mock-Kommentar behauptete
  "covered by service_ext tests", eine Datei die es nicht gibt; CreateBOM/
  GetBOM als Service-RPC-Pfad bleiben ungetestet, Kandidat fuer eine eigene
  Coverage-Unit. (3) Namensabgleich statt SKU-Abgleich ist bewusste
  Abweichung vom Backlog-Wortlaut mangels SKU-Feld an BOM-Positionen — bei
  spaeterem SKU-Feld an production_bom_items ist inventar_lookup.go leicht
  umzustellen. (4) GRPCInventarLookup.ResolveByNames selbst hat keinen
  Unit-Test (Service-Tests laufen gegen fakeInventarLookup); der
  200er-Deckel aus inventar.Service.ListItems ist dokumentiert aber
  ungetestet — ein Tenant mit >200 Inventar-Positionen kann Materialien
  haben, die der eine Batch-Call nicht mehr sieht (unbekannt statt
  gefunden, kein Fehler, aber ein stiller Deckel).

## Iteration 19 — a-chat-permission-seeds — done — 2026-08-08 21:10
- commit: dd9da170
- verify vorgaenger: sauber. 2e0f3bea (a-produktion-material-availability,
  Iteration 18) geprueft: HandleGetMaterialAvailability geht ueber
  pr.getClient() -> client.GetMaterialAvailability (gRPC-Client, kein
  Layer-Bypass), holt tenantID aus middleware.GetTenantID und validiert den
  Pfad-Parameter via validateUUIDParam; Route haengt am bestehenden
  produktion:order/read-Key, also kein Seed noetig; .proto, .pb.go und
  .grpc.pb.go regeneriert; openapi.yaml im selben Commit ergaenzt; keine
  Migration, keine neue Dependency. git merge origin/main war "Already up to
  date".
- praemisse: WIDERLEGT, und zwar vollstaendig. Die Unit behauptete, chat-
  bzw. channels-Permissions fehlten in den Seed-Migrationen. Gegengeprueft
  per Mengenvergleich statt per Stichprobe: alle 252 Keys aus
  capability-catalog.ts gegen alle 628 Permission-Namen aus backend/migrations/*.sql
  -> Differenz LEER. Die fuenf kommunikation:*-Keys stehen seit Migration
  000256 (Zeilen 254-259) drin und sind den Preset-Rollen zugeordnet. Die
  Notiz hatte nach dem Praefix `chat` gesucht; das FE benutzt `kommunikation:`.
  Gegenrichtung ebenfalls geprueft (die sicherheitskritischere): alle 174 Keys
  aus RequirePermission-Aufrufen in backend/internal/ gegen dieselbe
  Seed-Menge -> ebenfalls leer, sowohl im 3-Segment- als auch im
  Legacy-2-Segment-Format.
- gefunden: Der von der Notiz beschriebene Lockout existierte trotzdem, eine
  Ebene tiefer als vermutet. Nicht "Permission fehlt", sondern "Permission
  existiert und gehoert niemandem". DB-Abfrage gegen die lokale DB (Stand
  305, clean): genau EINE Permission hatte keinen einzigen Grant an eine
  System-Rolle — `mentions:read`. Migration 000017 legt die Zeile an und
  ordnet sie keiner Rolle zu; route_chat.go:83 erzwingt sie seit jeher fuer
  GET /api/v1/messages/mentions. Ergebnis: 403 fuer JEDEN Nutzer, Admin
  eingeschlossen. Genau der Fehlermodus, den die Unit-Notiz als "in diesem
  Repo schon einmal schiefgegangen" beschreibt.
- gebaut: (1) Migration 000306_grant_mentions_read_to_presets. Ordnet
  mentions:read den drei Preset-Rollen zu, die messages:read schon halten —
  admin, manager, member, scope 'all', per DB abgefragt statt geraten. Die
  Nachbarrouten im selben /api/v1/messages-Block haengen alle an
  messages:read, also dieselbe Zielgruppe; readonly/it_admin/hr_admin halten
  messages:read nicht und bleiben bewusst unveraendert. Idempotent
  (ON CONFLICT DO NOTHING), down loescht exakt diese drei Grants und laesst
  tenant-eigene Rollen in Ruhe. KEIN Guard angefasst, wie von der Unit
  gefordert.
  (2) TestEveryRouteGuardHasAUsablePermission in
  internal/testutil/permission_seed_regression_test.go — Standing-Guard nach
  dem Muster von TestAllPublicTablesHaveRLSOrAreAllowlisted. Parst per
  go/ast (Stdlib, keine neue Dependency) alle .go-Dateien unter internal/,
  sammelt jeden RequirePermission- und RequirePermissionAny-Aufruf mit
  String-Literalen und prueft gegen die DB, ob mindestens ein Key des
  Guards geseedet UND einer System-Rolle zugeordnet ist. AST statt Regex,
  weil route_auth.go mehrzeilige RequirePermissionAny-Aufrufe hat, die eine
  Zeilen-Regex verschluckt. Any-Guards werden bewusst als Gruppe gewertet
  (ODER-Semantik): die Legacy-Coarse-Aliase in diesen Aufrufen sind
  absichtlich teils nicht mehr geseedet, eine Pro-Key-Forderung waere falsch
  rot. 784 Call-Sites erfasst.
- gate: build ok (go build -p 2 ./...) | vet ok (testutil, gateway) | lint ok
  (golangci-lint testutil, 0 issues) | test ok (testutil 0 SKIP real gegen
  die lokale DB, gateway ok inkl. TestOpenAPIRouteDrift, middleware ok) |
  migration 306 up+down+up real gegen die lokale DB gefahren, danach
  Grants per psql verifiziert (admin/manager/member je scope 'all'), down
  liess 0 Zeilen zurueck | openapi n.a. (keine neue Route) | rls-smoke n.a.
  (kein Schema angefasst, nur Daten; TestAllPublicTablesHaveRLSOrAreAllowlisted
  lief im selben Paket gruen mit)
- mutations-probe: zwei Stueck, beide Diagnosezweige des neuen Tests belegt.
  (1) Der staerkste Beweis kam gratis: der Test lief VOR Migration 306
  gegen dieselbe DB und war rot mit exakt einem Treffer —
  "route_chat.go:83:10: mentions:read (seeded, granted to no preset role)".
  Nach der Migration gruen, ohne Testaenderung. (2) Fuer den zweiten Zweig
  RequirePermission("mentions","read") temporaer auf "mentions_bogus"
  gedreht -> Meldung wechselte korrekt auf "(no permission row)", 783
  andere Guards blieben gruen. Zurueckgedreht, Endgate danach erneut gruen.
- db-tests: Der neue Test IST ein DB-Test (SkipIfNoDB + PoolFromEnv, Rolle
  kmuhub_app also NOBYPASSRLS). Vorab geprueft, dass die Abfrage unter
  kmuhub_app ohne Tenant-Kontext ueberhaupt liest: role_permissions und
  roles haben Lesepolicies fuer tenant_id IS NULL, 8 System-Rollen und 459
  gegrantete Permissions sichtbar. Paket testutil: 0 Skips.
- offen: Drei Punkte fuer Luke. (1) PRODUKTIONSWIRKUNG: mentions:read ist
  auf Prod genauso ungegrantet — die Mentions-Route ist dort seit jeher tot
  (403 fuer alle). Migration 306 heilt das beim naechsten Deploy; ob es je
  jemandem aufgefallen ist, waere ein Blick ins FE wert (der Aufruf koennte
  clientseitig weggefangen worden sein und die Luecke deshalb still bleiben).
  (2) Der Standing-Guard prueft nur System-Presets (tenant_id IS NULL). In
  auth/postgres_repository.go:451 gibt es ein INSERT INTO roles mit
  `based_on` — kopiert eine tenant-eigene Rolle beim Anlegen die
  Preset-Grants, hat keine bestehende Kopie mentions:read je bekommen. Das
  ist Bestandsdaten-Nacharbeit, nicht Migrations-Scope, und lokal nicht
  pruefbar (0 tenant-eigene Rollen in der DB). (3) Der AST-Scanner
  ueberspringt Guards mit nicht-literalen Argumenten und loggt sie nur —
  heute gibt es keine (784 von 784 erfasst), aber der `lean:`-Marker im Code
  nennt den Upgrade-Trigger, falls je eine Route ihren Key zur Laufzeit
  berechnet.

## Iteration 20 — a-inbox-sla — blocked — 2026-08-08 19:15
- commit: -
- verify vorgaenger: sauber. `dd9da170` (a-chat-permission-seeds, Iteration
  19) geprueft: Migration 000306 ordnet `mentions:read` per
  `WHERE r.tenant_id IS NULL AND r.name IN ('admin','manager','member')`
  ausschliesslich den drei Presets zu, `ON CONFLICT DO NOTHING` macht sie
  idempotent, down loescht exakt dieselben drei Grants und laesst
  tenant-eigene Rollen unberuehrt; kein `RequirePermission`-Guard geaendert,
  keine neue Route, kein Handler angefasst — reine Daten-Migration plus ein
  neuer Standing-Guard-Test. Diff-Umfang (`git show --stat`) bestaetigt genau
  das: Migration + Test + Backlog/Journal, keine Handler-/Proto-Dateien. `git
  merge origin/main` war "Already up to date".
- praemisse: WACKELT, wie von der Unit selbst vorhergesagt — nur eine Ebene
  tiefer als der Notiz-Wortlaut nahelegt. `sla_policies` existiert
  tatsaechlich (Migration 000077), aber es ist keine freistehende,
  modul-unabhaengige Tabelle: sie haengt an `ticket_queues.sla_policy_id`,
  traegt `first_response_mins`/`resolution_mins`/`business_hours` fuer
  Helpdesk-Queues und wird ausschliesslich von `internal/helpdesk` gelesen
  (verifiziert per Grep, 0 Treffer in `internal/inbox`). Inbox
  (`inbox_messages`, `team_inboxes`, `team_inbox_members`, `routing_rules`,
  Migration 000047 + Folgemigrationen 000110/000124) hat keine Spalte und
  keinen Code-Pfad, der je auf `sla_policies` verweist oder verwiesen hat.
  Es gibt also keine Antwort auf die Frage "welche Policy gilt fuer welchen
  Team-Inbox/Channel" — exakt die Produktentscheidung, die die Notiz fuer
  den Fall "keine Richtlinien-Quelle" vorgesehen hat.
- gebaut: nichts im Code. `BACKLOG.yml`: `a-inbox-sla` auf `status: blocked`
  mit `blocked_reason` (Fundstelle: Migration 000077 fuer die Tabelle,
  Migrationen 000047/000110/000124 fuer die vollstaendige Inbox-Historie
  ohne `sla_policy_id`-Spalte, Grep-Ergebnis fuer `internal/inbox/**`) und
  drei konkreten Entscheidungsoptionen (eigene Inbox-Policy-Tabelle,
  Cross-Modul-FK auf Helpdesks `sla_policies`, oder Scope-Streichung).
- gate: n.a. — keine Code-/Migrations-Aenderung in diesem Schritt.
- mutations-probe: n.a. — Block-A-Unit ohne Coverage-Pflicht, und es gibt
  keinen neuen Code, an dem eine Probe etwas beweisen koennte.
- db-tests: n.a.
- offen: Fuer Luke — SLA-Modell-Entscheidung fuer Inbox treffen (siehe
  `blocked_reason`/`notes` in `BACKLOG.yml`). Sobald entschieden, ist die
  Berechnung selbst laut Notiz mechanisch (serverseitig, Filter in SQL) und
  eine kleine Folge-Unit.

## Iteration 21 — a-inventar-picking — done — 2026-08-08 19:25
- commit: 9445f8c8
- verify vorgaenger: sauber. Iteration 20 hat keinen Code angefasst; ihr
  Commit `2373880c` enthaelt exakt zwei Dateien (`BACKLOG.yml`,
  `JOURNAL.md`), `git show --stat` bestaetigt das. Der davorliegende
  Code-Commit `dd9da170` war in Iteration 20 bereits gegen die sechs
  Fehlerklassen geprueft und ist unveraendert. `git merge origin/main` war
  "Already up to date".
- gebaut: Kommissionierung, vollstaendig durch alle Schichten.
  - Migration 000307: `picking_lists` (tenant_id, reference, status
    open|picking|completed, assigned_to, created_by) und
    `picking_list_items` (tenant_id, picking_list_id, item_id,
    quantity_requested > 0, quantity_picked >= 0, location, UNIQUE
    (picking_list_id, item_id)), beide mit `tenant_id UUID NOT NULL` +
    `CALL enable_tenant_rls(...)`, up und down gefuellt.
  - Migration 000308: Seeds `inventar:picking:read|write|book`, an die
    System-Presets (`tenant_id IS NULL`) admin+manager voll, member nur read.
  - `internal/inventar`: Models, Errors, Repository-Interface,
    Postgres-Implementierung, Service (CRUD, Positions-Upsert, Buchen).
  - Proto: 8 RPCs + Messages, mit protoc regeneriert
    (`inventar.pb.go`/`inventar_grpc.pb.go`).
  - `internal/server/inventar_grpc.go`: 8 RPC-Methoden, `pickingListToProto`
    (Items als `[]`, nicht `null`), Fehler-Mapping.
  - `internal/gateway/route_inventar.go`: 7 Routen ueber den gRPC-Client,
    alle in `api/openapi.yaml` (`swagger-cli validate` gruen), plus zwei
    Schemas `InventarPickingList`/`InventarPickingListItem`.
- buchungs-semantik (die eigentliche Entscheidung dieser Unit): Bestand
  bewegt sich ausschliesslich ueber `AdjustStock` — denselben Pfad, den
  Anpassung und Inventur-Buchung nehmen, also mit `stock_movements`-Eintrag
  und Warnungs-Trigger; kein zweites UPDATE auf `inventory_items`.
  Reihenfolge: erst ALLE Positionen gegen den Bestand vorpruefen, dann die
  Liste per bedingtem UPDATE (`WHERE status <> 'completed'`, Rows-Affected
  als Rueckgabe) beanspruchen, erst dann buchen. Damit kann eine Liste, die
  nicht vollstaendig buchbar ist, nicht halb gebucht liegen bleiben, und ein
  zweiter Aufruf — auch ein gleichzeitiger — findet null betroffene Zeilen
  und bekommt 409 statt einer zweiten Bestandsbewegung.
- nebenbefund, mitgefixt: `ErrInsufficientStock` fiel in `mapInventarError`
  in den `default`-Zweig und wurde damit zu `codes.Internal` = HTTP 500.
  Es ist eine Aufrufer-Vorbedingung, nicht ein Serverfehler — jetzt
  `FailedPrecondition` = 409. Das betrifft auch die Bestandsrouten
  `/items/{id}/adjust` und `/transfer`; beide haben den 409 in
  `openapi.yaml` nachgetragen bekommen.
- gate: `go build` / `go vet` / `golangci-lint` (0 issues) /
  `go test ./internal/inventar/... ./internal/gateway/ ./internal/server/
  ./internal/testutil/` — alle gruen, auf dem finalen Tree wiederholt.
  `swagger-cli validate backend/api/openapi.yaml` gruen (der Drift-Test
  prueft nur Pfade, nicht die Spec).
  `TestEveryRouteGuardHasAUsablePermission` mit DB: 792 Guard-Fundstellen
  (vorher 784, also die 8 neuen erfasst), PASS —
  `TestAllPublicTablesHaveRLSOrAreAllowlisted` PASS.
- mutations-probe: zwei Stueck, beide rot, beide zurueckgedreht.
  (1) Doppelbuchungs-Schutz entfernt (Status-Guard + `!claimed`-Zweig
  raus) → `TestBookPickingList_SecondBookingDoesNotMoveStockAgain` FAIL
  ("stock must not move twice"). (2) Bestandsvorpruefung auf `if false`
  gesetzt → `TestBookPickingList_InsufficientStockLeavesEverythingUntouched`
  FAIL, und der Lauf zeigt genau den Schaden, den die Vorpruefung
  verhindert: die erste Position war schon gebucht, als die zweite an
  `ErrInsufficientStock` starb. Danach `go test ./internal/inventar/` wieder
  gruen.
- rls-smoke: `picking_lists` und `picking_list_items` beide
  `relrowsecurity=true`/`forced=true`, Policy `tenant_isolation`
  (`tenant_id = current_tenant_id() OR is_system_context()`). Real gefahren
  unter `SET LOCAL ROLE kmuhub_app` mit zwei Tenants: als A 1 von 2 Listen
  und 1 von 2 Positionen sichtbar, als B die eigene 1 / die fremde 0 /
  1 Position; Cross-Tenant-INSERT von der WITH-CHECK-Policy abgelehnt.
  Alles in einer zurueckgerollten Transaktion.
- db-tests: die Service-Tests sind Stub-Tests (mockRepository), kein
  `SkipIfNoDB` — dafuer lief das Gate mit gesetztem `DATABASE_URL`, sodass
  die beiden Standing-Guards in `internal/testutil` real gegen die DB
  geprueft haben. Migrationen 307/308 lokal angewandt, Kopf jetzt **308**,
  `dirty=false`.
- offen: Fuer Luke — (1) Es gibt noch kein Frontend fuer die
  Kommissionierung; die Routen sind da, `InventarPage.tsx` kennt sie nicht.
  Das ist eine Frontend-Session, kein Nachtlauf. (2) Der 500→409-Wechsel bei
  `ErrInsufficientStock` aendert das Verhalten der bestehenden
  Adjust-/Transfer-Routen. Wenn im FE irgendwo auf 500 gesondert reagiert
  wird, muesste das mitgezogen werden — ein Grep im Desktop-Client waere
  billig. (3) Die Buchung schreibt pro Position eine Bewegung mit Grund
  "Kommissionierung: <reference>"; eine echte Referenz-Spalte
  (`stock_movements.reference`) haette sie feiner verknuepft, das
  Inventur-Vorbild nutzt sie aber ebenfalls nicht — bewusst gleich gelassen
  statt zwei Muster nebeneinander zu haben.

## Iteration 22 — c-cov-biz-hr-changerequest — done — 2026-08-08 22:35
- commit: 353d557a
- verify vorgaenger: sauber. `9445f8c8` (a-inventar-picking, Iteration 21)
  gegen alle acht Fehlerklassen geprueft: alle sieben Handler in
  `route_inventar.go` gehen ueber `client.<RPC>` (`InventarServiceClient`,
  gRPC-Client, kein direkter Service-Zugriff); Migration 000307
  (`picking_lists`/`picking_list_items`) hat `tenant_id UUID NOT NULL` +
  `CALL enable_tenant_rls(...)` fuer beide Tabellen, up und down gefuellt;
  Migration 000308 seedet `inventar:picking:read/write/book` VOR jedem neuen
  `RequirePermission`-Aufruf (Reihenfolge stimmt); `.proto` und
  `inventar.pb.go`/`inventar_grpc.pb.go` im selben Commit regeneriert; alle
  sieben Routen in `api/openapi.yaml` (`picking`-Grep bestaetigt 20+
  Fundstellen inkl. Pfade und Schemas); kein Alt-Guard ersetzt, nur additiv.
  `git merge origin/main` war "Already up to date".
- praemisse: WIDERLEGT. Die Backlog-Notiz behauptete 0,0 % Coverage,
  "vollstaendig ungetestet". Tatsaechlich existiert
  `integration_db_test.go` bereits seit dem Feature-Commit `dbcf2493`
  ("feat(hr): add employee profile change requests") mit sieben
  DB-Integrationstests (Doppelantrag-Ablehnung, unbekanntes Feld,
  Genehmigung inkl. Zweifachgenehmigung, Team-Scope-Reichweite, Ablehnung
  mit Pflichtgrund, Ruecknahme, Cross-Tenant-Unsichtbarkeit). Gemessen vor
  dieser Iteration: **79,9 %** (`go test ./internal/biz/hr/changerequest/...
  -cover`), nicht 0,0 %. Trotzdem nicht als erledigt/blocked abgehakt —
  `go tool cover -func` zeigte reale, sicherheitsrelevante Luecken in
  `approveScopeAllows` (der laut Paket-Doku "eigentliche Grund, warum der
  Flow existiert") und in der Transaktionslogik von `ApproveAndApply`, also
  ergaenzt statt neu erfunden.
- gebaut: Fuenf neue DB-Tests in `integration_db_test.go`, keine
  Quelldatei-Aenderung (reine Testergaenzung).
  (1) `TestCreate_DrawerMismatchIsRefused` — Feld unter falschem Drawer
  eingereicht (`addressCity` unter `"banking"` statt `"personal"`) →
  `ErrFieldNotProposable`, nichts gespeichert.
  (2) `TestCreate_BlankOrOversizedValueIsRefused` — Leerwert (nur
  Whitespace) und 501-Zeichen-Wert beide abgelehnt.
  (3) `TestApprove_OwnScopeOnlyReachesTheApproversOwnRequest` — deckt
  `auth.ScopeOwn` in `approveScopeAllows` ab, vorher 0 % Zeilenabdeckung:
  ein Genehmiger mit `own`-Scope darf nur seinen eigenen Antrag entscheiden,
  einen fremden liefert `ErrOutOfScope`.
  (4) `TestApprove_TeamScope_ProposerWithoutProfileIsOutOfScope` — Antragsteller
  ohne `hr_employee_profiles`-Zeile (z. B. schon offgeboardet); `ManagerOf`
  liefert `ErrProfileNotFound`, `approveScopeAllows` macht daraus
  `false, nil` statt eines Serverfehlers → `ErrOutOfScope`.
  (5) `TestApprove_ProfileRemovedBetweenSubmitAndDecideRollsBack` — das
  Profil verschwindet zwischen Einreichung und Entscheidung; `ApproveAndApply`
  claimt die Zeile (`status='approved'`), das nachfolgende
  `UPDATE hr_employee_profiles` trifft 0 Zeilen → `ErrProfileNotFound`,
  UND die Transaktion rollt zurueck (Status bleibt `pending`, kein
  `decided_at`/`decided_by`) — das ist der Fall, der nur gegen eine echte
  DB beweisbar ist, kein Mock haette die Rollback-Semantik gepruefe.
- gate: build ok (`-p 2`, Windows-Linker-Workaround wie in fruehreren
  Iterationen) | vet ok | lint ok (`golangci-lint run
  ./internal/biz/hr/changerequest/...`, 0 issues) | test ok
  (`./internal/biz/hr/changerequest/` 12/12 PASS 0 SKIP,
  `./internal/gateway/` inkl. `TestOpenAPIRouteDrift`,
  `./internal/testutil/`, `./internal/server/` alle gruen) | migration n.a.
  (keine Schema-Aenderung) | openapi n.a. (keine neue Route)
- mutations-probe: zwei Stueck, beide aussagekraeftig, beide zurueckgedreht.
  (1) `case auth.ScopeOwn: return actorID == proposerID, nil` zu
  `return true, nil` gedreht (Own-Scope wirkungslos) → GENAU
  `TestApprove_OwnScopeOnlyReachesTheApproversOwnRequest` wurde rot, die
  drei Nachbartests (`_WritesTheValueToTheProfile`,
  `_TeamScopeReachesOnlyDirectReports`,
  `_TeamScope_ProposerWithoutProfileIsOutOfScope`) blieben gruen.
  (2) In `ApproveAndApply` die `if tag.RowsAffected() == 0`-Pruefung auf
  `if false` gesetzt (Profil-Verschwinden wird nicht mehr erkannt) → GENAU
  `TestApprove_ProfileRemovedBetweenSubmitAndDecideRollsBack` wurde rot, die
  drei Nachbar-Approve-Tests blieben gruen. Beide Proben zurueckgedreht,
  Endgate danach erneut gelaufen und gruen (12/12, 0 Skips).
- db-tests: alle zwoelf Tests im Paket sind DB-Integrationstests
  (`testutil.SkipIfNoDB` + `PoolFromEnv`, jeder Test seedet seinen eigenen
  Tenant) — **0 Skips**. `DATABASE_URL` war auf `kmuhub_app` gesetzt
  (NOSUPERUSER NOBYPASSRLS). Coverage vorher 79,9 %, nachher **85,2 %**
  (`go test ./internal/biz/hr/changerequest/... -cover`).
- offen: Ein Punkt fuer Luke. `ApproveAndApply` bleibt bei 68,4→hoeher aber
  nicht 100 % Coverage: `tx.Commit`-Fehler und der generische
  `tx.Exec`-Fehler (z. B. Verbindungsabbruch mitten in der Transaktion)
  sind weiterhin ungetestet — beides ist gegen die lokale DB nicht
  provozierbar ohne einen Fault-Injection-Mechanismus, den es in diesem
  Repo nicht gibt. Kein aktiver Befund, nur eine Grenze der
  DB-Integrationstest-Methode selbst.

## Iteration 23 — c-cov-biz-dashboard — done — 2026-08-08 19:42
- commit: ac3f9178
- verify vorgaenger: sauber. `353d557a` (c-cov-biz-hr-changerequest, Iteration
  22) erneut gegen den finalen Baum laufen lassen: `go test
  ./internal/biz/hr/changerequest/... -count=1 -v -cover` mit
  `DATABASE_URL` auf `kmuhub_app` — 12/12 PASS, 0 Skips, 85,2 % Coverage,
  identisch zum im Journal behaupteten Ergebnis. Diff nur Testdatei
  (`integration_db_test.go`, +140 Zeilen), keine Quelldatei-Aenderung, keine
  Migration, keine neue Route. `git merge origin/main` war "Already up to
  date".
- praemisse: bestaetigt. `internal/biz/dashboard/` hatte VOR dieser Iteration
  0 DB-Integrationstests — nur `service_test.go` (Service gegen
  MockRepository) und `cached_repository_test.go` (Cache-Wrapper gegen
  Miniredis). Die eigentliche Aggregationslogik in
  `postgres_repository.go::GetDashboardMetrics` (sechs handgeschriebene
  SQL-Queries: Umsatz, Pipeline, Statusverteilung, letzte Rechnungen,
  ablaufende Angebote, offene Mahnungen) lief nie gegen echtes Schema.
  Gemessen vor dieser Iteration: 17,2 % (`go test
  ./internal/biz/dashboard/... -cover` ohne die neue Datei).
- gebaut: Neue Datei `internal/biz/dashboard/postgres_repository_db_test.go`,
  neun Tests, keine Quelldatei-Aenderung. Muster von
  `internal/biz/hr/changerequest/integration_db_test.go` uebernommen
  (`testutil.SkipIfNoDB` + `PoolFromEnv`, kein Build-Tag, jeder Test seedet
  eigene Tenants).
  (1) `TestGetDashboardMetrics_RevenueScopedByTenantAndDateRange` — Query 1,
  zwei Tenants, Rechnung ausserhalb des Zeitraums, prueft alle vier
  Umsatzsummen exakt.
  (2) `TestGetDashboardMetrics_EmptyDateRangeReturnsZeroNotError` — leerer
  Tenant, alle Summen 0, kein Fehler, `MonthsInRange >= 1`.
  (3) `TestGetDashboardMetrics_PipelineScopedByTenant` — zwei Tenants,
  Entwurfs-Angebot darf den Durchschnittswert nicht verzerren,
  Konversionsrate 33,3 %.
  (4) `TestGetDashboardMetrics_StatusBreakdownIsAllTimeButTenantScoped` —
  Query 3 ignoriert bewusst den Zeitraum (Kommentar im Quellcode bestaetigt);
  Testzeitraum schliesst alle Rechnungen aus, Breakdown zaehlt trotzdem
  korrekt, zweiter Tenant zaehlt nicht mit.
  (5) `TestGetDashboardMetrics_RecentInvoicesScopedByTenantAndLimitedToTen`
  — elf Rechnungen, nur zehn kommen zurueck, neueste zuerst, fremder Tenant
  nicht dabei.
  (6) `TestGetDashboardMetrics_ExpiringQuotesWithinSevenDays` — Status,
  `valid_until`-Fenster und Tenant als Dreifachfilter.
  (7) `TestGetDashboardMetrics_PendingDunningRecordsScopedByTenant` — nur
  `status=draft`, nur der eigene Tenant.
  (8) `TestGetDashboardMetrics_CanceledContextReturnsError` — Fehlerpfad:
  abgebrochener Kontext liefert einen Fehler statt Panic oder leerem
  Ergebnis.
- gate: build ok (`go build -p 2 ./...`, komplettes Repo) | vet ok
  (`go vet ./internal/biz/dashboard/...`) | lint ok (`golangci-lint run
  ./internal/biz/dashboard/...`, 0 issues, nach Aufraeumen zweier
  Modernizer-Hinweise im neuen Testcode: `maps.Copy` statt manueller
  Merge-Schleife, `for i := range 11` statt `for i := 0; i < 11; i++`) |
  test ok (`./internal/biz/dashboard/` 21/21 PASS 0 SKIP,
  `./internal/gateway/` inkl. `TestOpenAPIRouteDrift` gruen) | migration
  n.a. (keine Schema-Aenderung) | openapi n.a. (keine neue Route, reine
  Testergaenzung)
- befund unterwegs: RLS ist hier Defense-in-Depth, nicht nur die
  Anwendungsschicht. Testweise `tenant_id = $1` aus der Umsatz-Query
  entfernt (`WHERE ($1::uuid IS NOT NULL) AND invoice_date...`, Parameter
  bewusst referenziert belassen, um keinen reinen SQL-Typfehler zu erzeugen)
  — der Zwei-Tenant-Test blieb GRUEN, weil die RLS-Policy auf
  `finance_invoices` unter `kmuhub_app` mit `app.tenant_id` aus dem
  `WithTenantCtx`-Kontext bereits auf den eigenen Tenant einschraenkt, bevor
  die explizite WHERE-Bedingung ueberhaupt greift. Kein aktiver Fund (die
  App-Ebene filtert trotzdem korrekt, siehe Mutations-Probe unten), aber
  bemerkenswert: ein vergessenes `tenant_id = $1` in einer neuen
  `postgres_repository.go`-Query dieser Art waere hier durch RLS abgefangen,
  nicht zwangslaeufig ein Produktionsleck.
- mutations-probe: zwei Versuche, zurueckgedreht.
  (1) `tenant_id = $1` durch `($1::uuid IS NOT NULL)` ersetzt (s.o.) — Test
  blieb gruen (RLS faengt es ab, siehe Befund oben). Als Mutations-Probe
  fuer DIESEN Test also nicht aussagekraeftig, deshalb Versuch (2).
  (2) `status IN ('sent','paid','overdue')` in der `total_invoiced`-Summe zu
  `status IN ('sent','paid')` verkuerzt (overdue faellt raus) — GENAU
  `TestGetDashboardMetrics_RevenueScopedByTenantAndDateRange` wurde rot
  (`TotalInvoiced = 1500, want 1700`), alle anderen Tests blieben gruen.
  Beide Aenderungen zurueckgedreht, `git diff --stat` auf die Quelldatei
  bestaetigt 0 Zeilen Differenz, Endgate danach erneut gelaufen und gruen.
- db-tests: alle neun neuen Tests sind DB-Integrationstests (`SkipIfNoDB` +
  `PoolFromEnv`), **0 Skips** unter `DATABASE_URL` auf `kmuhub_app`
  (NOSUPERUSER NOBYPASSRLS). Paket-Coverage 17,2 % -> **79,3 %**.
- offen: keins. Naechste Unit im Backlog: `c-cov-biz-gobdarchive`.

## Iteration 24 — c-cov-biz-gobdarchive — done — 2026-08-08 20:15
- commit: 94f5aa05
- verify vorgaenger: sauber. `ac3f9178` (c-cov-biz-dashboard, Iteration 23)
  gegen den finalen Baum erneut gelaufen: `go test ./internal/biz/dashboard/...
  -count=1 -cover` mit `DATABASE_URL` auf `kmuhub_app` — 21/21 PASS, 0 Skips,
  79,3 % Coverage wie behauptet. Diff nur die neue Testdatei
  `postgres_repository_db_test.go`, keine Quelldatei-Aenderung, keine
  Migration. `git merge origin/main` war "Already up to date".
- praemisse: teilweise widerlegt. Die `notes` im Backlog verlangen einen
  Test, der beweist, dass "ein Aenderungsversuch am archivierten Dokument
  scheitert". Beim Lesen von `repository.go` und Migration 000139 zeigt sich:
  das `Repository`-Interface hat strukturell KEIN `Update`/`Delete` — es gibt
  also gar keine Methode, ueber die ein Aenderungsversuch überhaupt
  ausgefuehrt werden koennte. Die Migration sagt das selbst so ("immutability
  by design — service-side"). Auf DB-Ebene gibt es dagegen KEINEN Schutz:
  kein Trigger, keine Rule, und `kmuhub_app` hat aus Migration 000121
  uneingeschraenkt `GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES` —
  auch auf `gobd_documents`/`gobd_document_events`. "Unveraenderlich" gilt
  also nur, solange kein Code jemals ein UPDATE/DELETE auf diese Tabellen
  ausfuehrt, nicht weil die DB es verweigern wuerde. Der einzige Schreibpfad
  im Repository ist `Create` — der wertvollste, ehrliche Test ist deshalb:
  ein zweiter `Create`-Aufruf mit derselben Dokument-ID (der einzige
  "Aenderungsversuch", den diese API ueberhaupt zulaesst) muss am Primary
  Key scheitern, und der Originaldatensatz muss unveraendert bleiben. Das ist
  real DB-erzwungen (PK-Unique-Constraint), im Unterschied zur reinen
  Interface-Abwesenheit. Diese Erkenntnis ist ein Befund fuer Luke, siehe
  "offen" — kein Fix in dieser Unit (Grant/Trigger-Aenderung ist Schema, nicht
  Test, und eine bewusste eigene Entscheidung wert).
  "Manipulierter Inhalt faellt an der Pruefsumme auf" ist ebenfalls keine
  eigene Verify-Funktion (es gibt keine `VerifyIntegrity`-RPC, nur den toten
  ENUM-Wert `integrity_check`) — getestet ist stattdessen die eigentliche
  Garantie: der SHA-256 wird per `io.TeeReader` waehrend des Uploads
  berechnet, landet unveraenderlich im Datensatz, und weicht jede
  nachtraegliche Manipulation der gespeicherten Bytes zwangslaeufig von
  diesem fixen Soll-Wert ab.
- gebaut: 24,8 % -> **89,9 %** Coverage, keine Quelldatei-Verhaltensaenderung.
  (1) `service_test.go` (+30 Tests): Fehlerpfade fuer `ArchiveDocument`
  (Upload-Fehler bricht VOR `repo.Create` ab, `repo.Create`-Fehler
  propagiert), `ArchiveInvoiceDocument` (kein InvoiceReader konfiguriert,
  Invoice-Lookup-Fehler, Dateiname mit/ohne Rechnungsnummer),
  `GetByID`/`List`/`GetDownloadURL`/`ListEvents` (vorher alle vier bei 0 %:
  Passthrough + NotFound + `GetPresignedURL`-Fehler +
  Access-Event-Append-Fehler ist non-fatal), `AddAnnotation`
  (Append-Fehler propagiert). Dazu
  `TestArchiveDocument_ChecksumDetectsStorageCorruption` (siehe oben).
  (2) NEUE Datei `postgres_repository_db_test.go` (10 DB-Tests, das
  eigentliche 0-%-Loch): `Create`+`GetByID`-Roundtrip inkl. NULL
  `source_invoice_id` -> nil-Pointer-Roundtrip,
  `Create`-Duplikat-ID-Ablehnung (PK-Constraint, siehe Praemisse),
  `GetByID` cross-tenant/unbekannte ID -> `ErrDocumentNotFound`,
  `List`-Filter (DocType, SourceInvoiceID inkl. echter FK-Zeile in
  `finance_invoices`, DateFrom), `List`-Pagination+Sortierung
  (`archived_at DESC`, Seite 1/2), `AppendEvent`+`ListEvents` Sortierung
  (`created_at DESC`), `ListEvents` fuer unbekanntes Dokument -> leer, kein
  Fehler.
  (3) Nebenbei-Fund beim Lesen: drei Dokumentationskommentare
  (`service.go`-Package-Doc, `models/gobd.go`-Struct-Doc,
  Migration-000139-Kopfkommentar) behaupteten "archivedAt.Year + 8", der
  tatsaechliche Code UND alle drei bestehenden `computeRetentionUntil`-Tests
  (Normal/Dec31/Jan1) belegen seit je "+10" (§147 AO: 10 Jahre). Reiner
  Kommentar-Fix auf allen drei Stellen, keine Verhaltensaenderung, keine neue
  Migration (nur Kommentartext in der bereits angewandten 000139 korrigiert
  — `schema_migrations` traegt keinen Content-Hash, unkritisch).
- gate: build ok | vet ok (`./...`) | lint ok (`golangci-lint run
  ./internal/biz/gobdarchive/...`, 0 issues) | test ok
  (`./internal/biz/gobdarchive/` 39/39 PASS 0 SKIP,
  `./internal/server/`+`./internal/gateway/`+`./internal/testutil/` alle
  gruen) | migration n.a. (nur Kommentar in 000139, Kopf lokal weiter 308,
  `dirty=false`) | openapi n.a. (kein RPC/Route angefasst)
- mutations-probe: drei Versuche, zwei davon aussagekraeftig, einer verworfen.
  (1) `ArchiveInvoiceDocument`s Leerstring-Check `if inv.InvoiceNumber == ""`
  zu `!= ""` gedreht (Dateiname-Fallback invertiert) -> GENAU
  `TestArchiveInvoiceDocument_FilenameUsesInvoiceNumberWhenPresent` und
  `..._FilenameFallsBackToIDWhenNumberBlank` wurden rot (beide, weil sie sich
  gegenseitig ergaenzen), alle uebrigen blieben gruen.
  (2) `GetByID`s `WHERE tenant_id = $1 AND id = $2` auf `WHERE id = $2`
  verkuerzt (Tenant-Scoping in der SQL entfernt) -> verworfen: Postgres kann
  bei ungenutztem `$1` den Parametertyp nicht ableiten (SQLSTATE 42P18,
  Compile-Fehler in ALLEN vier betroffenen Tests statt eines gezielten Rots
  — dieselbe Falle wie in Iteration 1 vermerkt). Ersetzt durch
  `WHERE ($1::uuid IS NOT NULL) AND id = $2` (syntaktisch gueltig, Tenant-
  Bedingung trotzdem wirkungslos) -> blieb GRUEN, inklusive
  `TestPostgresRepository_GetByID_CrossTenantReturnsNotFound`. Grund: RLS
  greift als zweite, unabhaengige Schicht (`app.tenant_id`-GUC aus
  `WithTenantCtx`) und faengt den Cross-Tenant-Read trotzdem ab — derselbe
  Befund wie in Iteration 23 bei `dashboard` ("RLS ist Defense-in-Depth,
  nicht nur die Anwendungsschicht"). Kein aktiver Fund, aber die Probe war
  fuer DIESEN Test nicht aussagekraeftig.
  (3) `scanDocument`s `if sourceInvoiceID != uuid.Nil` auf `if true` gesetzt
  (NULL-Behandlung ausgehebelt) -> GENAU
  `TestPostgresRepository_CreateAndGetByID_RoundTrip` wurde rot (die
  `assert.Nil(t, got.SourceInvoiceID, ...)`-Zeile), alle 38 uebrigen Tests
  blieben gruen. Alle drei Aenderungen zurueckgedreht, `git diff --stat` auf
  `postgres_repository.go` bestaetigt 0 Zeilen Differenz, Endgate danach
  erneut gelaufen und gruen (39/39, 0 Skips).
- db-tests: 10 neue DB-Integrationstests in `postgres_repository_db_test.go`
  (`SkipIfNoDB` + `PoolFromEnv`), **0 Skips** im gesamten Paket (39 PASS, 0
  SKIP) unter `DATABASE_URL` auf `kmuhub_app`. Migrationskopf 308 vorher
  bereits angewendet (keine neue Migration in dieser Unit).
- offen: Ein Befund fuer Luke, kein Fix in dieser Unit. Die "GoBD-Immutability"
  von `gobd_documents`/`gobd_document_events` ist NICHT auf DB-Ebene erzwungen
  — `kmuhub_app` hat aus der generischen Rollen-Migration 000121 volles
  `UPDATE`/`DELETE` auf ALLE Tabellen, es gibt weder Trigger noch Rule noch
  einen expliziten `REVOKE` fuer diese beiden. Der einzige Schutz ist, dass
  aktuell kein Code-Pfad ein UPDATE/DELETE auf diese Tabellen ausfuehrt (das
  Repository-Interface bietet keine Methode dafuer) — das ist Schutz durch
  Abwesenheit, kein DB-Constraint. Ein Angreifer mit direktem DB-Zugriff
  (oder ein zukuenftiger Bugfix, der versehentlich ein UPDATE ergaenzt) waere
  nicht blockiert. Fuer ein §147-AO-Archiv, das explizit mit
  "revisionssicher" wirbt, waere ein `REVOKE UPDATE, DELETE ON gobd_documents,
  gobd_document_events FROM kmuhub_app` (plus ggf. ein expliziter
  Ausnahmepfad fuer `AppendEvent`, falls INSERT-only gewuenscht ist) die
  naheliegende Haertung. Bewusst nicht in dieser Coverage-Unit umgesetzt,
  weil das ein Schema-/Grant-Eingriff mit Sicherheitsauswirkung ist, kein
  Test — waere eine eigene Block-A-artige Unit wert. Ausserdem: `mockRepository`s
  `createErr`/`getErr`/`appendEventErr`/`uploadErr`-Felder in
  `service_test.go` existierten vorher unbenutzt (toter Scaffold-Code) und
  sind jetzt alle real durch mindestens einen Test bespielt.
  Naechste Unit im Backlog: `c-cov-biz-creditnote`.

## Iteration 25 — c-cov-biz-creditnote — done — 2026-08-08 21:15
- commit: df6a7398
- verify vorgaenger: sauber. `94f5aa05` (c-cov-biz-gobdarchive) geprueft: nur
  `service_test.go`/`postgres_repository_db_test.go` (neu) plus drei reine
  Kommentarkorrekturen (`service.go`-Package-Doc, `models/gobd.go`,
  Migration-000139-Kopfkommentar: `+8` -> `+10`) — `git show` gegen alle drei
  Nicht-Test-Dateien bestaetigt Kommentar-only, keine Logikaenderung. Kein
  Handler, keine Route, kein RPC, keine neue Tabelle, kein
  `RequirePermission`-Guard angefasst. `git merge origin/main` war "Already
  up to date".
- gebaut: Praemisse beim Lesen korrigiert, bevor irgendetwas geschrieben
  wurde — `internal/biz/creditnote` hat KEINE 0-DB-Test-Luecke wie die
  vorigen Bloecke: `integration_test.go` und
  `send_atomic_integration_test.go` existierten bereits (Muster
  `//go:build integration` + `testsupport/pgtc` + testcontainers-go, laeuft
  nachts in `.github/workflows/nightly.yml` als eigener "Finance Integration
  Tests"-Job fuer invoice/quote/creditnote — NICHT Teil von `go test ./...`
  in `ci.yml`, deshalb hatte die im Backlog notierte 28,2 % nur die
  Mock-basierten Tests aus `service_test.go` gezaehlt). Gemessen MIT
  `-tags=integration` lag der echte Ausgangswert bei 50,2 %, nicht 28,2 %.
  Der verbliebene 0-%-Rest war trotzdem real: `List`, `GetByInvoiceID` und
  `ListForDATEVExport` in `postgres_repository.go` liefen nie gegen
  echtes SQL — nur `Create`/`GetByID`/`Update`/`UpdateInTx` waren durch die
  bestehenden Tests abgedeckt.
  Neue Datei `repository_coverage_integration_test.go`
  (package `creditnote_test`, externes Testpaket wie
  `send_atomic_integration_test.go`, damit `internal/biz/invoice` als echter
  `InvoiceReader` importiert werden kann ohne einen Zyklus in Package
  `creditnote` selbst zu erzeugen) mit fuenf Tests:
  (1) `TestList_FiltersPaginationAndTenantScoping` — Status-Filter,
  `OriginalInvID`-Filter, Pagination (Limit/Offset, Total bleibt ueber beide
  Seiten korrekt), Tenant-Scopung (Tenant B nie in Tenant As Liste).
  (2) `TestGetByInvoiceID_TenantScopedAndOrderedDescending` — genau die
  Abfrage, auf die `StornoInvoice`s Doppel-Storno-Schutz sich verlaesst;
  `ORDER BY created_at DESC`, unbekannte Invoice-ID liefert leer statt
  Fehler, Cross-Tenant-Kontext gegen eine fremde Invoice-ID liefert leer.
  (3) `TestListForDATEVExport_DateRangeStatusAndKeysetPaging` —
  Status-Filter (nur `sent`), die inklusive Tagesgrenze
  (`created_at < $3::date + INTERVAL '1 day'`), Keyset-Paging ueber zwei
  Seiten per `(created_at, id)`-Cursor.
  (4) `TestCreate_AmountNotValidatedAgainstInvoiceOpenBalance` — die im
  Backlog geforderte "ueberzogene Gutschrift" als Ist-Verhalten
  festgeschrieben, NICHT als neue Validierung gebaut: `Service.Create`
  prueft den Gutschriftsbetrag an keiner Stelle gegen den offenen
  Rechnungsbetrag der Originalrechnung, eine Gutschrift ueber das
  Zehnfache der Rechnungssumme wird anstandslos angelegt. Das ist eine
  Coverage-Unit, keine Block-A-Luecke — eine neue Pruefung waere
  Scope-Erweiterung ueber `done_when` hinaus ("wird abgelehnt ODER ihr
  Verhalten ist als Test festgeschrieben" erlaubt ausdruecklich Letzteres).
  (5) `TestCreate_DecimalAmountsSurviveDBRoundTripAsExactStrings` — belegt
  die zweite `done_when`-Vorgabe ("Betragspruefungen auf der Zeichenkette,
  nicht auf gerundeten Floats") mit dem klassischen Float64-Fall 0,10+0,20:
  Subtotal wird als exakte Dezimalzahl "0.30" im Go-Objekt UND nach dem
  DB-Roundtrip erwartet, zusaetzlich wird die NUMERIC-Spalte per `::text`
  roh gelesen (umgeht jede Go-seitige Formatierung) und ebenfalls gegen
  "0.30" verglichen — waere irgendwo im Pfad float64 im Spiel, stuende dort
  "0.30000000000000004" oder eine Rundung.
- gate: build ok | vet ok (`./...`) | lint ok (`golangci-lint run
  ./internal/biz/creditnote/...` UND mit `--build-tags=integration`, beide
  0 issues) | test ok (`go test -tags=integration ./internal/biz/creditnote/...`
  27/27 PASS 0 SKIP inkl. aller fuenf neuen Tests;
  `./internal/gateway/`+`./internal/server/`+`./internal/testutil/` mit
  `DATABASE_URL` auf `kmuhub_app` alle gruen) | migration n.a. (keine
  Migration, kein Schema angefasst) | openapi n.a. (kein RPC/Route
  angefasst)
- mutations-probe: zwei Stueck, beide aussagekraeftig, an den zwei
  vorher 0-%-Abfragen mit der hoechsten Business-Relevanz.
  (1) `GetByInvoiceID`s `ORDER BY created_at DESC` auf `ASC` gedreht ->
  GENAU `TestGetByInvoiceID_TenantScopedAndOrderedDescending` wurde rot,
  alle 26 uebrigen Tests blieben gruen.
  (2) `ListForDATEVExport`s `created_at < ($3::date + INTERVAL '1 day')`
  auf `created_at < $3::date` verkuerzt (inklusive Tagesgrenze aufgehoben)
  -> GENAU `TestListForDATEVExport_DateRangeStatusAndKeysetPaging` wurde
  rot (der Datensatz vom letzten Tag im Bereich fiel raus), alle uebrigen
  blieben gruen. Beide Aenderungen zurueckgedreht, `git diff --stat` auf
  `postgres_repository.go` bestaetigt 0 Zeilen Differenz, Endgate danach
  erneut gelaufen und gruen (27/27, 0 Skips, 82,1 % Coverage).
- db-tests: 27 Tests im gesamten Paket unter `-tags=integration`
  (testcontainers-go, kein `SkipIfNoDB`/`DATABASE_URL`-Pfad in diesem
  Paket — jeder Testlauf startet seinen eigenen `pgvector/pgvector:pg16`-
  Container und wendet alle 277 `.up.sql`-Migrationen an), **0 Skips**.
  Docker Desktop war lokal verfuegbar.
- offen: Zwei Befunde fuer Luke, keiner in dieser Unit behoben.
  (1) Die im Backlog beschriebene Luecke "Ueberzogene Gutschrift wird nicht
  geprueft" ist real und jetzt durch einen Test dauerhaft sichtbar gemacht
  (`TestCreate_AmountNotValidatedAgainstInvoiceOpenBalance`) statt nur in
  `backend-gaps.md` zu stehen. Ob das eine gewollte Freiheit ist (der
  Sachbearbeiter entscheidet den Betrag manuell) oder eine echte Luecke, ist
  eine Fachentscheidung — waere als eigene Block-A-artige Unit sauber
  nachruestbar (Guard in `Service.Create`, vergleicht `GrossTotal` gegen
  `inv.GrossTotal` minus Summe bereits gesendeter Gutschriften zu dieser
  Rechnung, ueber `GetByInvoiceID`).
  (2) Waehrend der Recherche aufgefallen, nicht Teil dieser Unit:
  `internal/biz/hr/absence/integration_test.go`,
  `internal/biz/hr/employee/integration_test.go` und
  `internal/biz/hr/leave/integration_test.go` tragen ebenfalls
  `//go:build integration`, tauchen aber in KEINEM Workflow unter
  `.github/workflows/*.yml` auf (nur invoice/quote/creditnote sind im
  "Finance Integration Tests"-Job von `nightly.yml` verdrahtet) — diese drei
  Dateien laufen nirgends in CI, weder taeglich noch nachts. Ob das
  historisch tote Testdateien sind oder ein fehlender Nightly-Eintrag, ist
  unklar; ein Grep zeigt, dass sie kompilieren (`go build -tags=integration
  ./...` war Teil des BUILD_OK oben), aber ob sie inhaltlich noch zum Schema
  passen, wurde in dieser Unit nicht geprueft. Waere ein guter Kandidat fuer
  eine eigene kleine Rechercheeinheit ("laufen die drei HR-Integrationstests
  noch durch, und falls ja, warum sind sie nicht in nightly.yml?").
  Naechste Unit im Backlog: `c-cov-crm-report`.

## Iteration 26 — c-cov-crm-report — done — 2026-08-08 20:22
- commit: bcf8b28e
- verify vorgaenger: teilweise unsauber, aber am Code selbst nichts
  auszusetzen. `df6a7398` (c-cov-biz-creditnote, Iteration 25) inhaltlich
  geprueft: nur die neue Testdatei `repository_coverage_integration_test.go`,
  keine Quelldatei angefasst, `go test -tags=integration
  ./internal/biz/creditnote/...` lief erneut 27/27 gruen. Der Fehler steckte
  im selben Commit an anderer Stelle: der `BACKLOG.yml`-Hunk, der den neuen
  `result:`-Block an die `c-cov-biz-creditnote`-Unit anhaengte, hat dabei die
  Zeile `- id: c-cov-crm-report` der DARAUFFOLGENDEN Unit geloescht (sichtbar
  im Diff: `-  - id: c-cov-crm-report` ohne Ersatz). Ergebnis: kein gueltiges
  YAML-Listenelement mehr — `phase`/`service`/`model`/... der crm-report-Unit
  haengten als zusaetzliche (Duplikat-)Keys an der creditnote-Unit. Der
  Treiber selbst waere davon vermutlich nicht gestolpert (er zaehlt offene
  Units per Zeilen-Regex auf `status: todo`, nicht per YAML-Parser — siehe
  Kopfkommentar), aber jedes Tool, das die Datei als YAML laedt (dieser
  Verify-Schritt eingeschlossen), waere auf einen Parse-Fehler oder
  stille Duplicate-Key-Ueberschreibung gelaufen. Reparatur: `- id:
  c-cov-crm-report` wieder eingefuegt (ein Zeilen-Insert, siehe Commit-Diff).
  `python3 -c "import yaml; ..."` bestaetigt danach 78 Units, keine
  doppelten IDs. `git merge origin/main` war "Already up to date".
- praemisse: bestaetigt. `internal/crm/report` hatte 0 DB-Tests
  (`service_test.go` nur gegen `MockRepository`), alle drei
  Repository-Methoden liefen nie gegen echtes SQL.
- gebaut: 28,4 % -> **90,8 %** Coverage. Echter Fund dabei, kein reiner
  Test-Zuwachs: `GetPipelineReport` gruppierte und sortierte nach
  `ps.position` — diese Spalte hat in `pipeline_stages` nie existiert
  (Migration 000008 nennt sie `sort_order`). Jeder produktive Aufruf dieser
  Methode waere mit `SQLSTATE 42703` gescheitert; unentdeckt, weil die
  einzige bestehende Testdatei die Mock-Repository testete, nie die echte
  SQL-Query. Fix: `ps.position` -> `ps.sort_order` in `GROUP BY`/`ORDER BY`
  (`postgres_repository.go:48-49`), selber Commit. Neue Datei
  `postgres_repository_db_test.go` (Paket `report`, DATABASE_URL-Pattern wie
  `internal/crm/contact/rls_test.go`, `SkipIfNoDB`+`PoolFromEnv`) mit elf
  Tests: je Aggregation (Pipeline, Conversion, Activity) ein Zwei-Tenant-Fall,
  ein Leerer-Tenant-Fall (Stages/Metrics kommen als leerer Slice, nicht nil,
  zurueck — der Repository-Code allokiert das schon korrekt vor, die Tests
  pinnen es fest) und ein Fehlerpfad (canceled context). Der Pipeline-Test
  prueft zusaetzlich `sort_order`-Sortierung und die Weighted-Value-Formel
  (`value * probability / 100`) mit exakten Decimal-Werten; der
  Conversion-Test seedet `deal_stage_history` direkt (Trigger umgangen, um
  `changed_at` frei zu kontrollieren) und prueft die Average-Days-Berechnung
  auf ~4 Tage genau; der Activity-Test deckt beide Zweige des
  `created_by OR assigned_to`-Userfilters einzeln ab.
- gate: build ok (`go build ./internal/...` — `go build ./...` scheitert auf
  dieser Maschine systembedingt beim Linken aller 24 Service-Binaries mit
  "fatal error: runtime: cannot allocate memory", reproduzierbar unabhaengig
  von dieser Aenderung, kein Regressionsindiz) | vet ok (`./internal/...`)
  | lint ok (`golangci-lint run ./internal/crm/report/...`, 0 issues)
  | test ok (`go test -count=1 -cover ./internal/crm/report/...` 25/25 PASS
  0 SKIP, 90,8 % Coverage; `./internal/gateway/`+`./internal/server/`+
  `./internal/testutil/`+`./internal/crm/...` mit DATABASE_URL auf
  `kmuhub_app` alle gruen) | migration n.a. (keine Migration, kein Schema
  angefasst) | openapi n.a. (kein RPC/Route angefasst)
- mutations-probe: zwei Stueck, beide aussagekraeftig.
  (1) Den `sort_order`-Fix zurueck auf `ps.position` gedreht -> alle drei
  Tests, die `GetPipelineReport` tatsaechlich aufrufen, wurden rot mit
  "column ps.position does not exist" (nur der Canceled-Context-Test blieb
  gruen, weil er vor dem Query abbricht) — belegt, dass die Tests real gegen
  die SQL-Query laufen, nicht gegen eine Mock-Attrappe.
  (2) Im Userfilter von `GetActivityReport` `(created_by = $4 OR
  assigned_to = $4)` auf `created_by = $4` verkuerzt -> GENAU
  `TestGetActivityReport_UserFilterMatchesCreatedByOrAssignedTo` wurde rot
  (die nur-zugewiesene, nicht selbst erstellte Aktivitaet fiel aus der
  Zaehlung), alle uebrigen Tests blieben gruen. Beide Aenderungen
  zurueckgedreht, `git diff --stat` auf `postgres_repository.go` bestaetigt
  nur die zwei beabsichtigten Fix-Zeilen als Differenz zum Ausgangsstand,
  Endgate danach erneut gelaufen und gruen (25/25, 0 Skips, 90,8 %).
- db-tests: 11 neue DB-Integrationstests unter dem DATABASE_URL-Pattern
  (`SkipIfNoDB`+`PoolFromEnv`, lokale `docker-postgres-1` verfuegbar),
  **0 Skips** im gesamten Paket (25 PASS, 0 SKIP).
- offen: Kein Befund fuer Luke aus dieser Unit selbst — der einzige Fund
  (`ps.position`) wurde direkt gefixt, nicht nur dokumentiert, weil die
  Korrektur eindeutig und risikofrei war (Spaltenname, keine Schema-
  Aenderung, keine Verhaltensaenderung ausser "funktioniert jetzt statt
  SQL-Fehler"). Erwaehnenswert fuer den naechsten Verify-Vorspann: Achtung
  beim Anhaengen von `result:`-Bloecken an bestehende Backlog-Units mit
  dem Edit-Tool — der Bug in dieser Iteration entstand dadurch, dass die
  neue `result:`-Sektion versehentlich die `- id:`-Zeile der naechsten Unit
  mit ueberschrieben hat. Nach jedem BACKLOG.yml-Edit lohnt sich ein
  YAML-Parse-Check (`python3 -c "import yaml; yaml.safe_load(open(...))"`),
  nicht nur ein visueller Diff-Blick.
  Naechste Unit im Backlog: `c-cov-biz-invoice-repo`.

## Iteration 28 — c-cov-biz-invoice-repo — done — 2026-08-08 21:40
- commit: f22ac820
- gebaut: `internal/biz/invoice` war bei 47,7 % (mit `-tags=integration`) bzw.
  30,9 % ohne DB. Der bestehende `integration_test.go` (587 Zeilen) deckt
  Zeilen-Roundtrip, Lock-Spalten, Overdue-Filter und den Bexio-Import-Pfad
  bereits gut ab. Komplett ungetestet waren `postgres_open_items.go`
  (`ListOpenItems`, `SummarizeOpenItems`, `bucketCondition`,
  `bucketIndexCase`) und `postgres_document_chains.go` (`ListDocumentChains`
  plus die vier `*NodeStatus`-Helfer und `paymentNodeNumber`) — genau die von
  den Backlog-notes benannten "lohnendsten Luecken" (Offene Posten,
  Belegketten). Zwei neue Dateien:
  - `open_items_chains_helpers_test.go` (kein Build-Tag, reine
    Funktionstests): `bucketCondition` fuer alle vier Bucket-Keys plus
    unbekannter Key -> `ErrUnknownAgingBucket`, `bucketIndexCase`-Bounds
    gegen `models.AgingBucketUpperDays()`, die vier Node-Status-Helfer und
    `paymentNodeNumber` (Referenz vs. Fallback auf Rechnungsnummer).
  - `postgres_open_items_and_chains_integration_test.go` (`//go:build
    integration`, testcontainers wie die bestehende Datei, wiederverwendet
    `seedTenant`/`makeInvoice`/`twoLines` aus `integration_test.go` desselben
    Pakets). Drei Testfunktionen mit Subtests:
    - `TestListOpenItems_FiltersPaginationAndDunning`: alle vier Aging-Buckets
      belegt, `OverdueOnly` filtert "noch nicht faellig" raus, Bucket-Filter
      isoliert genau eine Rechnung, unbekannter Bucket-Key ist ein Fehler,
      Pagination begrenzt die Seite aber nicht `total`, Teilzahlung reduziert
      `OpenAmount` exakt um den gezahlten Betrag (Decimal-Vergleich, nicht
      Float), Volltilgung entfernt die Rechnung komplett aus der Liste
      (`gross_total - paid > 0`-Filter), zwei Mahnstufen auf derselben
      Rechnung liefern die HOECHSTE (nicht die zuletzt eingefuegte) ueber den
      LATERAL-Join, Cross-Tenant-Isolation, abgebrochener Kontext.
    - `TestSummarizeOpenItems_AggregatesByCurrencyAndBucket`: Zwei-Rechnungen-
      Bucket-Summe (Count, Amount, DaysOverdueSum), leerer Tenant liefert `[]`
      nicht `nil`, Cross-Tenant-Isolation, abgebrochener Kontext.
    - `TestListDocumentChains_FullLifecycleAndBranches`: sechs Ketten-
      Varianten in einem Testlauf — volle Kette Angebot(accepted)->Rechnung->
      Teilzahlung->Mahnung->Pending-Restbetrag-Knoten (inkl. Datums-Sortierung
      der Knoten), vollstaendig bezahlt->`IsComplete=true` ohne Pending-Knoten,
      stornierte Rechnung->`IsComplete=true` unabhaengig vom Restbetrag,
      Gutschrift im Entwurf zaehlt weder zum Restbetrag noch zur
      Vollstaendigkeit, versendete Gutschrift ueber den vollen Betrag schliesst
      die Kette ab, alleinstehendes abgelehntes Angebot als Ein-Knoten-Kette
      mit `IsComplete=true`, Cross-Tenant-Isolation, abgebrochener Kontext.
  Seed-Helfer fuer `finance_quotes`/`finance_payments`/
  `finance_dunning_records`/`finance_credit_notes` sind rohes SQL im Testfile
  (Spalten aus den jeweiligen `postgres_repository.go` der Nachbarpakete
  abgeschrieben) statt eines Imports dieser Pakete — vermeidet jedes
  Zyklusrisiko und bleibt im Stil der bestehenden Tests dieses Pakets.
- gate: `go build ./internal/...` ok | `go vet -tags=integration
  ./internal/biz/invoice/...` ok | `golangci-lint run
  ./internal/biz/invoice/...` 0 issues | `go test -count=1 -tags=integration
  -cover ./internal/biz/invoice/...` — alle Tests gruen (neue + alle
  bestehenden Service-/Import-/JSONB-/Atomic-Rollback-Tests), 0 Skips,
  **68,1 % Coverage** (vorher 47,7 % mit `-tags=integration`, 30,9 % ohne DB).
  Migration n.a. (kein Schema angefasst), openapi n.a. (kein RPC/Route
  angefasst).
- mutations-probe: zwei Stueck, beide aussagekraeftig.
  (1) In `postgres_open_items.go` `ORDER BY level DESC` auf
  `ORDER BY level ASC` gedreht (im LATERAL-Join fuer den hoechsten
  Mahnstatus) -> genau der Subtest `bucket_filter_isolates_d60` wurde rot
  ("dunning level: got 1, want 2"), alle anderen Subtests blieben gruen.
  (2) In `postgres_document_chains.go` `invoiceNodeStatus()` den
  `InvoiceStatusPaid`-Zweig von `ChainNodeCompleted` auf `ChainNodeActive`
  gedreht -> `TestInvoiceNodeStatus` wurde rot. Beide Aenderungen
  zurueckgedreht, `git diff --stat` auf beiden Dateien bestaetigt keine
  Differenz zum Ausgangsstand.
- db-tests: alle neuen DB-Integrationstests laufen unter dem
  testcontainers-Muster dieses Pakets (frischer Container pro
  `pgtc.StartPostgres(t)`-Aufruf, kein `DATABASE_URL`/`SkipIfNoDB`), **0
  Skips**.
- offen: Kein Befund fuer Luke aus dieser Unit. Bemerkenswert fuer den
  naechsten Verify-Vorspann: dieses Paket nutzt testcontainers
  (`//go:build integration`, `pgtc.StartPostgres`) statt des
  `DATABASE_URL`+`SkipIfNoDB`-Musters, das `c-cov-crm-report` und
  `c-cov-biz-dashboard` nutzen — beide Muster existieren nebeneinander im
  Repo, je nach Paket nachsehen, welches dort schon etabliert ist, statt zu
  raten.
  Naechste Unit im Backlog: `c-cov-biz-quote`.

## Iteration 29 — c-cov-biz-quote — done — 2026-08-08 21:55
- commit: 7ead6260
- praemisse widerlegt: Backlog behauptete 33,3 % Coverage fuer
  `internal/biz/quote`; gemessen waren 62,8 % (`-tags=integration`, testcontainers-
  Muster wie bei `c-cov-biz-invoice-repo`). Alle vier `done_when`-Kriterien zu
  Statuswechsel-Guards, Zeichenketten-Summen und Cross-Tenant waren bereits durch
  `service_test.go` (Mock-Ebene, 41 Tests) und `integration_test.go`
  (Repository-Ebene: Create/Update/List/GetByDealID/RLS) abgedeckt.
- gebaut: Die echte Luecke lag komplett in `postgres_repository.go` bei 0,0 %
  gemessener Funktionsabdeckung: `Delete`, `UpdateStatus`, der gesamte
  `PostgresNumberSequenceRepo` (`NextNumber`, `NextNumberInTx`,
  `GetSequenceInfo`) und der gesamte `PostgresCompanySettingsRepo`
  (`GetByTenantID`, `Upsert`) liefen noch nie gegen echtes SQL — obwohl
  `PostgresNumberSequenceRepo` die GoBD-luecken-freie Dokumentennummerierung
  fuer Angebote, Rechnungen UND Gutschriften traegt (geteilter Code, ueber
  `invoicebiz.SequenceInfo` auch von `internal/biz/invoice` genutzt). Neue
  Datei `postgres_repository_db_test.go` (`//go:build integration`, package
  `quote`, wiederverwendet `seedTenantQ`/`makeQuote`/`twoQuoteLines`/
  `countQuoteLineRows` aus `integration_test.go` desselben Pakets), neun
  Testfunktionen:
  - `TestRepository_Delete_RemovesQuoteAndLines` — Delete raeumt die
    `finance_quote_lines`-Zeilen mit auf (Cascade), Cross-Tenant-Delete
    aendert 0 Zeilen statt eines Fehlers.
  - `TestRepository_UpdateStatus_IsTenantScoped` — Cross-Tenant-Aufruf laesst
    den Status unveraendert, eigener Tenant kann ihn setzen.
  - `TestNumberSequenceRepo_SequentialWithinTenantAndYear` — drei
    aufeinanderfolgende Nummern, `GetSequenceInfo` liefert `CurrentNumber=3`.
  - `TestNumberSequenceRepo_IndependentPerFiscalYearAndTenant` — neues
    Fiskaljahr UND fremder Tenant starten beide wieder bei 0001, statt den
    Zaehler des anderen fortzusetzen.
  - `TestNumberSequenceRepo_GetSequenceInfo_NilForUnseenSequence` — der
    `pgx.ErrNoRows`-Zweig liefert `(nil, nil)`, keinen Fehler.
  - `TestSend_RealDB_AssignsNumberAndStatusAtomically` — `Service.Send` mit
    echtem Pool (nicht Mock/noopTxBeginner) committet Nummer + Status in
    einem Zug.
  - `TestSend_RealDB_FailedUpdateRollsBackNumberAssignment` — der
    wertvollste Test der Unit: repliziert genau die Tx-Kopplung, die
    `service.go`s Send-Kommentar als historischen GoBD-Bugfix beschreibt
    (NextNumberInTx + UpdateInTx in EINER Transaktion, damit ein
    fehlgeschlagenes Update die Nummer nicht verbrennt). `Send()` selbst
    schreibt immer einen gueltigen Status und laesst sich von aussen nicht in
    den Fehlerfall zwingen — der Test baut deshalb dieselbe Tx-Sequenz manuell
    nach und erzwingt den Fehler ueber die CHECK-Constraint
    `chk_finance_quotes_status` (Status `"bogus-status"`), rollt zurueck und
    belegt per zweitem `NextNumber`-Aufruf, dass die naechste Nummer wieder
    0001 ist statt 0002.
  - `TestCompanySettingsRepo_UpsertAndGetByTenantID_RoundtripsDecimalPrecision`
    — `Basiszinssatz` als NUMERIC(5,2) mit Vorzeichen (-0,88; deutsche
    Basiszinssaetze waren real ueber Jahre negativ) uebersteht den
    String-Roundtrip; zweiter Upsert auf denselben Tenant geht ueber
    `ON CONFLICT (tenant_id)` (kein Duplikat); der B6-Guard bestaetigt: ein
    leerer `DefaultCurrency` faellt auf den System-Default `EUR` zurueck,
    NICHT auf den zuvor gespeicherten Tenant-Wert — meine erste
    Testerwartung ("bleibt CHF") war falsch und wurde vom Testlauf selbst
    widerlegt, korrigiert auf die tatsaechliche Guard-Semantik.
- gate: `go build ./internal/...` ok | `go vet -tags=integration
  ./internal/biz/quote/...` ok | `golangci-lint run --build-tags=integration
  ./internal/biz/quote/...` 0 issues | `go test -tags=integration -count=1
  -cover ./internal/biz/quote/...` — alle 41 Tests gruen (13 Integration- +
  28 Service-Tests), 0 Skips, **73,5 % Coverage** (vorher 62,8 %).
  Migration n.a. (kein Schema angefasst), openapi n.a. (kein RPC/Route
  angefasst).
- mutations-probe: zwei Stueck.
  (1) In `NextNumberInTx`s SELECT die Bedingung `AND fiscal_year = $3`
  entfernt (Query behaelt aber den dritten Bind-Parameter) ->
  `TestNumberSequenceRepo_IndependentPerFiscalYearAndTenant` wurde sofort rot
  (`select sequence: expected 2 arguments, got 3`). Zurueckgedreht,
  `git diff --stat` bestaetigt keine Differenz.
  (2) In `PostgresCompanySettingsRepo.Upsert` den B6-Leerstring-Guard
  entfernt (`defaultCurrency := settings.DefaultCurrency` ohne Fallback) ->
  `TestCompanySettingsRepo_UpsertAndGetByTenantID_RoundtripsDecimalPrecision`
  wurde rot (`DefaultCurrency after empty-string Upsert: got "   ", want
  "EUR"`). Zurueckgedreht, `git diff --stat` bestaetigt keine Differenz.
- db-tests: testcontainers-Muster (frischer Container pro
  `pgtc.StartPostgres(t)`), **0 Skips**. Laufzeit des Gesamtpakets ~66s
  (13 neue DB-Tests je ein eigener Container, ~4-6s je Test).
- offen: Kein Befund fuer Luke aus dieser Unit. Bemerkenswert fuer folgende
  Iterationen: `PostgresNumberSequenceRepo` wird von quote, invoice UND
  creditnote genutzt (geteilter Code in `internal/biz/quote/postgres_repository.go`,
  ueber `invoicebiz.NumberSequenceRepo`-Interface) — die hier gebauten
  Sequence-Tests decken damit indirekt auch die Nummerierungs-Grundlage der
  anderen beiden Pakete ab, falls dort noch Coverage-Units offen sind.
  Naechste Unit im Backlog: `c-cov-crm-company`.

## Iteration 30 — c-cov-crm-company — done — 2026-08-08 21:05
- commit: a05208e2
- verify vorgaenger: sauber (7ead6260 ist ein reiner Testdatei-Commit, keine
  der sechs Fehlerklassen betroffen — kein Gateway/Route/Proto/Guard/Tabelle
  angefasst).
- gebaut: Neue Datei `postgres_repository_db_test.go` mit sechs DB-Integrationstests
  gegen `internal/crm/company` (Konvention der bestehenden `rls_test.go`/
  `tenant_write_test.go` uebernommen: `testutil.PoolFromEnv` gegen die lokale
  Compose-DB, NICHT testcontainers — das ist crm-Konvention, biz-Pakete nutzen
  `pgtc`). Coverage 42,7 % -> 79,7 % (Backlog nannte 35,6 %, war durch
  fruehere Iterationen bereits hoeher).
  - `TestRepository_List_FiltersScopesAndPaginates` — Suche, Industrie-Filter,
    Tag-AND-Filter, Pagination (Offset ueber Total liefert leer statt Fehler),
    Tenant-Scopung.
  - `TestService_Delete_HangingContactsBlockDeletionButForeignTenantContactDoesNot`
    — der Kern der Unit: ein Kontakt im GLEICHEN Tenant blockiert (`ErrCompanyInUse`),
    ein absichtlich fremd-tenant-verseuchter `company_id`-Verweis (direkt
    geseedet, kein Pfad den die App je erzeugen wuerde) blockiert NICHT, weil
    `HasContacts`/`GetContactCount` tenant-scoped filtern. Plus
    `ErrCompanyNotFound` fuer eine nicht existente Firma.
  - `TestRepository_Tags_AddGetRemoveRoundtripAndTagExistsIsTenantScoped` —
    `TagExists` traegt KEIN explizites `tenant_id` im SQL, RLS ist die einzige
    Schranke; belegt fuer eigenen und fremden Tag. `Service.AddTags` mit
    fremdem Tag liefert `ErrTagNotFound` (RLS versteckt die Zeile, der Service
    kann "existiert nicht" von "gehoert jemand anderem" nicht unterscheiden —
    by design).
  - `TestRepository_CustomFields_SetGetRoundtripAndEmptyReturnsNoRows` —
    Roundtrip inkl. Upsert-Overwrite, leere Firma liefert leeren Slice statt
    Fehler.
  - `TestRepository_FindDuplicateCandidates_DomainExactAndFuzzyNameExcludeMergedAndForeignTenant`
    — domain_exact, name_fuzzy (Trigram), bereits gemergte Firma und
    Fremd-Tenant-Domain-Treffer beide ausgeschlossen.
  - `TestRepository_MergeInto_ReassignsRelationsMergesTagsAndCustomFieldsThenSoftDeletes`
    — Kontakte/Aktivitaeten/Deals umgehaengt, Tags und Custom Fields gemergt,
    Dublette soft-deleted (`merged_into_id`), Cross-Tenant-Guard (Dublette aus
    fremdem Tenant) liefert Fehler statt Merge, zweiter Merge-Versuch auf eine
    bereits gemergte Firma liefert `ErrAlreadyMerged`.
- echter-bug: `MergeInto`s Tag-Merge-INSERT (`postgres_repository.go:432-439`
  vor dem Fix) setzte `company_tags.tenant_id` NICHT — die Spalte ist seit
  Migration 000124 NOT NULL + RLS-bewehrt (`enable_tenant_rls('company_tags')`,
  kein Default, kein Trigger). Jeder Merge einer Dublette, die mindestens
  einen Tag trug, brach die gesamte Merge-Transaktion mit einer
  RLS-Policy-Verletzung (SQLSTATE 42501) ab — mein erster Testlauf hat das
  live reproduziert, bevor ich es gefixt habe. Fix: INSERT uebernimmt
  `tenant_id` aus der Quellzeile (`SELECT $1, tag_id, tenant_id FROM
  company_tags WHERE company_id = $2`), exakt das Muster, das `AddTags`
  bereits fuer denselben Sachverhalt benutzt. Root-Cause-Fix, kein Workaround
  (Regel `shared-lean-code.md`: Bug-Fix an der gemeinsamen Funktion).
- testfixture-falle (fuer folgende Iterationen relevant): `custom_field_definitions`,
  `activities` und `deals` haben `tenant_id NOT NULL DEFAULT
  '00000000-...001'` (System-Tenant) — ein `testutil.SeedRow`-Aufruf ohne
  explizites `tenant_id` landet also NICHT im Test-Tenant, sondern im
  System-Tenant, und die Zeile ist unter einem echten Tenant-Kontext per RLS
  unsichtbar (kein Fehler, nur leere Ergebnisse — genau das ist mir zweimal
  passiert und hat die Tests zunaechst mit falschem Befund rot laufen lassen,
  bis ich es auf ein fehlendes Testfixture-Feld statt einen Repo-Bug
  zurueckgefuehrt habe). `pipeline_stages`/`deals`/`contacts` explizit
  `tenant_id` setzen, nicht auf den Default verlassen.
- gate: `go build -p 2 ./internal/crm/... ./internal/gateway/...` ok |
  `go vet` ok | `golangci-lint run --config .golangci.yml
  ./internal/crm/company/...` 0 issues | `go test -count=1
  ./internal/crm/company/...` — alle Tests gruen (bestehende + 6 neue),
  **0 Skips**, DATABASE_URL gesetzt (Rolle kmuhub_app). Keine Route/kein
  Proto/keine Migration angefasst, daher `TestOpenAPIRouteDrift` nicht
  betroffen und kein separater `go test ./internal/gateway/`-Lauf noetig.
- mutations-probe: `HasContacts`s `AND tenant_id = $2` entfernt (Query behielt
  den zweiten Bind-Parameter) -> `TestService_Delete_HangingContactsBlockDeletionButForeignTenantContactDoesNot`
  wurde sofort rot (`expected 1 arguments, got 2`). Zurueckgedreht, `git diff
  --stat` bestaetigt keine Differenz. Der Fund fuer den echten Bug
  (`MergeInto`-Tag-Merge) diente als zweite, unabsichtliche Mutations-Probe:
  der Test war rot VOR dem Fix (RLS-Verletzung) und gruen danach — dieselbe
  Beweislast wie eine absichtliche Probe, nur in umgekehrter Reihenfolge
  entstanden.
- offen: Kein DB-Gate-Ausfall. Naechste Unit im Backlog: `c-cov-crm-activity`.

## Iteration 31 — c-cov-crm-activity — done — 2026-08-08 22:10
- commit: 259ab227
- verify vorgaenger: sauber (a05208e2 aendert nur die Testdatei plus einen
  gezielten 7-Zeilen-Fix in MergeInto's Tag-Merge-INSERT, exakt wie im
  Journal-Eintrag der Iteration 30 beschrieben — kein Gateway/Route/Proto/
  Guard/neue-Tabelle betroffen).
- gebaut: Neue Datei `postgres_repository_db_test.go` mit acht
  DB-Integrationstests gegen `internal/crm/activity` (vorher 0 DB-Tests im
  Paket — nur Mock-basierte `service_test.go` sowie die bestehenden
  `rls_test.go`/`tenant_write_test.go` fuer die reine CRUD-RLS-Flaeche).
  Coverage 40,1 % (lokal gemessen; Backlog nannte 36,2 %) -> 77,8 %.
  - `TestRepository_List_FiltersScopesAndPaginates` — alle sieben Filter
    (ActivityType, ContactID, CompanyID, DealID, AssignedTo, CreatedBy,
    IsCompleted) einzeln, Search, Sortierung nach subject, Pagination
    (Offset ueber Total liefert leer statt Fehler), Tenant-Scopung ueber
    eine fremde Aktivitaet, die nirgends auftaucht.
  - `TestRepository_List_TagFilterRequiresAllTags` — die
    `HAVING COUNT(DISTINCT tag_id) = N`-AND-Semantik: eine doppelt getaggte
    Aktivitaet matcht bei zwei angefragten Tags, eine einfach getaggte nicht.
  - `TestRepository_GetRelationNames_FoundAndNotFound` — GetContactName/
    GetCompanyName/GetDealName/GetUserName je Treffer- und
    Nicht-Treffer-Zweig (leerer String + nil-Error bei pgx.ErrNoRows).
  - `TestExistenceChecks_AreTenantScopedByRLS` — der Kern der Unit:
    ContactExists/CompanyExists/DealExists/UserExists tragen KEIN
    tenant_id-Praedikat im SQL. Unter der eigenen Tenant-Session sind die
    seeded Zeilen sichtbar, unter einer fremden Session verschwinden sie —
    ausschliesslich RLS haelt diese Grenze.
  - `TestService_Create_RejectsActivityLinkedToForeignTenantEntity` — dieselbe
    Eigenschaft ueber den echten `Service.Create`-Pfad bewiesen, alle drei
    Zweige (Contact/Company/Deal): eine Aktivitaet, die an eine fremde
    Entitaet verlinkt werden soll, liefert ErrContactNotFound/
    ErrCompanyNotFound/ErrDealNotFound, weil die Exists-Checks unter der
    aufrufenden Session die fremde Zeile gar nicht erst sehen. Das ist genau
    die im Backlog benannte Luecke ("eine fehlende Vorpruefung") — sie
    existiert nicht als Bug, weil RLS sie schliesst, aber das stand bislang
    nirgends als Test.
  - `TestRepository_Tags_AddGetRemoveRoundtripAndTagExistsIsTenantScoped` —
    Roundtrip plus TagExists ebenfalls ohne SQL-Tenant-Praedikat (RLS-getragen),
    GetTags aus fremder Session liefert keine Zeilen.
  - `TestRepository_CustomFields_SetGetRoundtripUpsertAndEmptyReturnsNoRows`
    — Upsert-Overwrite, leere Aktivitaet liefert leeren Slice statt Fehler.
  - `TestRepository_GetContactTimeline_CombinesActivitiesAndDealsScopedAndPaginated`
    — UNION aus Activities und Deal-Links, Sortierung neueste-zuerst,
    Pagination, ein absichtlich fremd-tenant-verseuchter Deal mit derselben
    contact_id bleibt in der eigenen Sicht aussen vor, und eine
    Angreifer-Session mit der GESTOHLENEN echten Tenant-ID als Parameter
    sieht trotzdem nichts (RLS, nicht die WHERE-Klausel, blockiert).
- praemisse widerlegt (Teilaspekt): "Zeitraumfilter" aus dem Backlog-Scope
  existiert nicht als Datumsbereichs-Parameter — weder in `ListFilter` noch
  im Gateway-Handler (`route_crm_activities.go`) gibt es einen From/To-Filter
  auf due_date oder created_at. Interpretiert als das, was tatsaechlich im
  Code steckt: Sortierung nach due_date (bereits vorhanden, mitgetestet)
  und der Pagination-Grenzfall leerer Zeitraum/leere Seite (Offset ueber
  Total). Kein Blocker, kein neuer Endpunkt gebaut — reine Praezisierung.
- testfixture-falle (wie iteration 30, hier erneut bestaetigt):
  `custom_field_definitions.tenant_id` hat `DEFAULT
  '00000000-...001'::uuid` (System-Tenant). Ein `SeedRow`-Aufruf ohne
  explizites `tenant_id` landet unsichtbar ausserhalb des echten
  Test-Tenants. Diesmal vorab bekannt und in der Testdatei per Kommentar
  festgehalten, damit es nicht erneut zu einem falschen Rot-Befund fuehrt.
- korrigierter erster Entwurf (relevant fuer folgende Iterationen): Die
  "Fremd-Tenant-Session sieht nichts"-Assertion in
  `TestRepository_GetContactTimeline_CombinesActivitiesAndDealsScopedAndPaginated`
  pruefte im ersten Entwurf mit der ECHTEN `tenantOther`-ID als Parameter.
  Da absichtlich eine fremd-tenant-verseuchte Test-Deal-Zeile mit exakt
  `tenant_id=tenantOther` UND derselben `contact_id` existiert, haette diese
  Zeile unter ihrer EIGENEN Tenant-ID legitim zurueckkommen muessen — der
  Test haette also mit einer falschen Erwartung gearbeitet und nichts
  bewiesen (waere aber zufaellig gruen gelaufen, weil ich das vor dem
  ersten Testlauf beim Nachdenken ueber die RLS-Policy-Kombination aus
  Session-Tenant und WHERE-Klausel gefunden und korrigiert habe, nicht weil
  der Testlauf es aufgedeckt haette). Korrigiert auf die gestohlene ECHTE
  `tenantOwn`-ID aus der Angreifer-Session (`ctxOther`) heraus — das ist der
  Fall, in dem RLS (Session-Tenant = tenantOther) und die WHERE-Klausel
  (Parameter = tenantOwn) einander widersprechen und daher garantiert 0
  Zeilen liefern, unabhaengig vom Dateninhalt.
- gate: `go build -p 2 ./internal/crm/... ./internal/gateway/...` ok |
  `go vet ./internal/crm/activity/...` ok | `golangci-lint run --config
  .golangci.yml ./internal/crm/activity/...` 0 issues | `go test -count=1
  ./internal/crm/activity/...` — alle 41 Tests gruen (33 bestehende + 8
  neue), **0 Skips**, DATABASE_URL gesetzt (Rolle kmuhub_app). Keine
  Route/kein Proto/keine Migration angefasst, daher `TestOpenAPIRouteDrift`
  nicht betroffen und kein separater `go test ./internal/gateway/`-Lauf
  noetig.
- mutations-probe: In `List` die `CompanyID`-Filterbedingung
  (`conditions = append(...)`) entfernt, aber den Bind-Parameter
  (`args = append(...)`, `argNum++`) belassen -> `TestRepository_List_FiltersScopesAndPaginates`
  wurde sofort rot (`expected 1 arguments, got 2` — pgx-Placeholder-Mismatch,
  weil ein ungenutzter Parameter im Slice blieb). Zurueckgedreht, `git diff
  --stat internal/crm/activity/postgres_repository.go` bestaetigt keine
  Differenz.
- offen: Kein DB-Gate-Ausfall. Naechste Unit im Backlog: `c-cov-biz-datev`.

## Iteration 32 — c-cov-biz-datev — done — 2026-08-08 21:35
- commit: 95650432
- verify vorgaenger: sauber. `259ab227` (c-cov-crm-activity) ist ein reiner
  Test-Commit (neue Datei `postgres_repository_db_test.go` + Backlog/Journal),
  kein Gateway/Route/Proto/Guard/neue-Tabelle betroffen, `git show --stat`
  bestaetigt nur die drei erwarteten Dateien. `git merge origin/main` war
  "Already up to date".
- praemisse widerlegt: Der Backlog-Scope verlangte "Zeichensatz (Windows-1252)
  belegt" per Testdaten. Das stimmt nicht gegen den echten Code:
  `NewStreamWriter` schreibt eine UTF-8-BOM (`0xEF 0xBB 0xBF`), und
  `encoding/csv` emittiert grundsaetzlich nur UTF-8 — es gibt in diesem Paket
  keinen cp1252/charmap-Transcoding-Schritt (Grep ueber `internal/biz/datev`
  auf `1252`/`charmap`/`golang.org/x/text/encoding`: 0 Treffer; nur
  Planungsdateien wie `backend-gaps.md` erwaehnen Windows-1252 als
  Erwartung, kein Code). Statt eine Windows-1252-Kodierung zu erfinden (das
  waere ein Fach-/Architektur-Entscheid mit neuer Dependency, keine
  Coverage-Unit), belegt der neue Golden-Test genau das, was real
  ausgeliefert wird: UTF-8 mit BOM, Umlaute/Eszett als ihre echten
  Mehrbyte-UTF-8-Sequenzen (`c3bc` = "ü", `c39f` = "ß", per Byte-Vergleich
  bewiesen, nicht nur per String-Contains). Vorab per Probe-Test verifiziert
  (Hex-Dump der realen Export-Bytes angeschaut, dann als literale
  Erwartung uebernommen) statt blind generiert.
- gebaut: Coverage fuer den EXTF-Formatvertrag (`exporter.go`, `mapping.go`,
  `buchungsstapel.go`) — 36,6 % -> 40,8 % Paket-Coverage (der Rest des
  Pakets — oauth.go, uploader.go, postgres_config_repo.go,
  postgres_upload_repo.go, belegbilder.go — ist DATEV-API-Anbindung, nicht
  der Formatvertrag, und blieb bewusst ausserhalb des Scopes). Neu:
  - `TestExport_GoldenBytesWithUmlauts` — Byte-fuer-Byte-Vergleich der
    kompletten Export-Ausgabe (BOM, EXTF-Header, Spaltenkoepfe, eine
    Buchungszeile) gegen eine festgeschriebene erwartete Zeichenkette,
    inkl. expliziter Byte-Assertion auf die UTF-8-Sequenzen von ü/ß.
  - `mapping_test.go` (neu) — `RevenueAccountForRate`,
    `RevenueAccountForRateAndMode` (alle vier Verzweigungen inkl.
    reverse_charge vs. Kleinunternehmer) und `BUSchluesselForRate` auf
    100 % Coverage.
  - Skip-Zweige: `TestWriteInvoices_SkipsNonExportableStatuses` (draft +
    ein unbekannter Status "cancelled") und
    `TestWriteCreditNotes_SkipsNonSentStatus` — belegen, dass nicht
    exportierbare Belege weder eine Buchungszeile noch einen
    DocumentCount-Eintrag erzeugen.
  - Fehlerpfade: kaputtes `line_items`-JSON in Invoice UND CreditNote wird
    als Fehler mit Beleg-Nummer/ID durchgereicht (`parseLineItems`), und
    `Export()` reicht beide Fehler nach oben durch statt eine
    Teil-Datei zurueckzugeben.
  - Schreibfehler: `alwaysFailWriter` (schlaegt bei jedem Write fehl) deckt
    den BOM-Schreibfehler in `NewStreamWriter` (sofortiger Fehlschlag, kein
    Puffer-Trick noetig) und, per direkter Konstruktion eines
    `StreamWriter` mit eigenem `*csv.Writer` (selbes Package, Zugriff auf
    das unexportierte Feld `w`), den `Close()`-Flush-Fehlerpfad
    ("csv flush: ...").
  - `buchungsstapel_test.go` (neu, schlank): NUR das, was in
    `upload_service_test.go` fehlte (dort bereits vorhanden:
    Pagination-Cursor, DocumentCount-schliesst-Drafts-aus, invertierter
    Zeitraum — wiederverwendet statt dupliziert). Neu:
    `ErrBuilderNotConfigured` bei nil-Builder und bei fehlendem
    Invoice-/CreditNote-Reader, `TestBuild_SettingsErrorIsBestEffortTolerated`
    (ein fehlschlagender `CompanySettingsReader` lässt den Export trotzdem
    durch, Berater-/Mandantennummer bleiben leer), sowie
    Invoice-/CreditNote-Reader-Fehlerpropagation (dafuer
    `creditNotePagerStub` in `upload_service_test.go` um ein `err`-Feld
    ergaenzt, analog zum bereits vorhandenen `invoicePagerStub.err`).
- gate: `go build -p 2 ./internal/biz/datev/... ./internal/gateway/...
  ./cmd/biz/... ./cmd/gateway/...` ok | `go vet ./internal/biz/datev/...`
  ok | `golangci-lint run --config .golangci.yml ./internal/biz/datev/...`
  0 issues | `go test -count=1 ./internal/biz/datev/...` — 36/36 PASS,
  **0 Skips**, `DATABASE_URL` gesetzt (Rolle kmuhub_app). Keine
  Route/kein Proto/keine Migration angefasst, daher kein separater
  `go test ./internal/gateway/`-Lauf noetig (Build-Check reicht).
- mutations-probe: zwei Stueck.
  (1) In `WriteInvoices`s Status-Filter `&&` zu `||` gedreht (macht die
  Bedingung tautologisch wahr -> ALLE Invoices werden uebersprungen, auch
  `sent`). Ergebnis: `go vet` (Teil von `go test`) schlug SOFORT mit
  `suspect or` auf beiden betroffenen Zeilen an, bevor ueberhaupt ein Test
  lief — der Build-Gate wurde rot. Aussagekraeftig, aber kein
  Test-Rot im engeren Sinn, deshalb zweite Probe.
  (2) `docDate.Format("0201")` (DDMM) zu `docDate.Format("0102")` (MMDD)
  gedreht -> GENAU `TestExport_SingleInvoice19Percent` (bestehend) und
  `TestExport_GoldenBytesWithUmlauts` (neu) wurden rot (erwartet
  Belegdatum "1503"/"2003", bekommen "0315"/"0320"), alle uebrigen 34
  Tests blieben gruen. Beide Proben zurueckgedreht, `git diff --stat
  internal/biz/datev/exporter.go` bestaetigt keine Differenz, Endgate
  danach erneut gelaufen und gruen (36/36, 0 Skips).
- db-tests: 0 echte DB-Integrationstests in den neuen/geaenderten Dateien
  (reines Mock-/In-Memory-Testing — der EXTF-Export braucht keine DB). Das
  bestehende `TestTenantIsolation_Datev_DB` im Paket lief unveraendert mit
  **0 Skips** (2 Subtests), `DATABASE_URL` war gesetzt.
- offen: Ein Punkt fuer Luke. Die Windows-1252-Erwartung aus
  `backend-gaps.md`/dem Backlog-Scope ist gegen den Code falsifiziert (siehe
  oben) — falls DATEV-Importe in der Praxis cp1252 brauchen, ist das ein
  eigener Fach-Entscheid (neue Dependency `golang.org/x/text/encoding/
  charmap`, ein echter Transcoding-Schritt zwischen `csv.Writer` und dem
  Ziel-`io.Writer`), keine Coverage-Nacharbeit. Aktuell exportiert das
  System nachweislich UTF-8 mit BOM, DATEV liest laut eigener Spec beides.

## Iteration 32 — c-cov-crm-contact-repo — done — 2026-08-08 21:35
- commit: f27f1c77
- gebaut: zwei neue DB-Testdateien fuer internal/crm/contact
  (postgres_repository_db_test.go, postgres_lead_db_test.go). Coverage
  36,6% -> 80,6%. Abgedeckt: List/ListWithVisibility (Filter, Sortierung,
  Pagination inkl. Tenant-Bedingung auf der Gesamtzahl, Owner/Admin/
  Visibility-Filter), ListByIDs/ListAll, GetCompanyName(s), Tags inkl.
  Batch, CustomFields inkl. Batch, FindDuplicateCandidates (phone_exact +
  name_fuzzy + merged-/foreign-tenant-Ausschluss), MergeInto (Reassign
  Activities/Deals, Tag-/CustomField-Merge, Soft-Delete, Cross-Tenant-Guard,
  ErrAlreadyMerged), ListLeads (Default-Stage schliesst customer aus,
  Status-/Search-Filter, Pagination) und UpdateLead (Partial-Patch,
  Customer-Guard, Temperature-Clear-vs-Set, fremde/fehlende ID).
- fund + fix: MergeInto setzte beim Tag-Merge kein tenant_id auf
  contact_tags (NOT NULL + RLS seit Migration 000124) — jeder
  Contact-Merge mit Tags auf dem Duplikat schlug in Produktion mit einer
  RLS-Verletzung fehl und rollte die gesamte Merge-Transaktion zurueck
  (per Test reproduziert, nicht spekuliert). Gefixt analog zum bereits
  korrekten company_tags-Muster (internal/crm/company/postgres_repository.go:435):
  `INSERT INTO contact_tags (contact_id, tag_id, tenant_id) SELECT $1,
  tag_id, tenant_id FROM contact_tags WHERE contact_id = $2`.
- fund, nicht gefixt (Schema-Frage fuer Luke): `idx_contacts_email`
  (Migration 000007) ist ein GLOBALER Unique-Index auf LOWER(email) ohne
  tenant_id — zwei Kontakte koennen system-weit, nicht nur pro Tenant, nie
  dieselbe E-Mail tragen. Macht den email_exact-Zweig in
  FindDuplicateCandidates durch keinen aktuellen Schreibpfad erreichbar
  (repo.Create prueft zusaetzlich GetByEmail). Kein Coverage-Thema, eine
  Architekturfrage — als eigene Fix-Unit oder bewusste Doku-Entscheidung
  vorzumerken.
- gate: `go build -p 2 ./internal/crm/... ./internal/gateway/...
  ./cmd/gateway/...` ok | `go vet ./internal/crm/...` ok | `golangci-lint
  run --config .golangci.yml ./internal/crm/contact/...` 0 issues |
  `go test -count=1 -p 1 ./internal/crm/...` — alle Pakete gruen
  (inkl. contact, report), 0 Skips, `DATABASE_URL` gesetzt (Rolle
  kmuhub_app). `-p 1` noetig: der volle `./internal/crm/...`-Lauf ohne
  `-p 1` erschoepft lokal die Postgres-Verbindungen (mehrere
  Paket-Pools gleichzeitig, "remaining connection slots are reserved for
  roles with the SUPERUSER attribute") — eine lokale Umgebungsgrenze,
  reproduzierbar unabhaengig von diesem Diff, kein Fund im Code. Keine
  Route/kein Proto angefasst, daher kein separater
  `go test ./internal/gateway/`-Lauf noetig (Build-Check reicht;
  trotzdem mitgebaut da crm oft von Gateway-Handlern importiert wird).
- mutations-probe: `ct.lifecycle_stage <> '`+LifecycleCustomer+`'` in
  `ListLeads` (postgres_lead.go) zu `=` gedreht — macht den
  Default-Stage-Filter zum genauen Gegenteil (nur customer statt alles
  ausser customer). Ergebnis: `TestRepository_ListLeads_
  DefaultStageStatusSearchAndTenantScopedCount` sofort rot (total=1 statt
  3, der customer-Kontakt tauchte im offenen Inbox-Filter auf, die beiden
  echten Leads fehlten). Zurueckgedreht, `git diff --stat
  internal/crm/contact/postgres_lead.go` bestaetigt keine Differenz,
  Endgate danach erneut gelaufen und gruen. Der Tags-Fix in MergeInto
  hatte de facto bereits seine eigene Mutations-Probe: der Test schlug mit
  dem urspruenglichen (kaputten) Code real fehl (RLS-Verletzung), bestand
  erst nach dem Fix.
- db-tests: 15 neue DB-Integrationstests in den beiden neuen Dateien, alle
  real gegen `kmuhub_app` gelaufen, 0 Skips.
- verify vorgaenger: sauber (Iteration 31, DATEV-Export — nur *_test.go
  geaendert, kein Gateway-/Proto-/Permission-/Tenant-Bezug, Build-Diff
  bestaetigt reine Testdateien).
- offen: Die beiden oben genannten Funde fuer Luke — (1) idx_contacts_email
  ist global statt tenant-scoped, eine Architekturfrage, kein Bug in
  diesem Diff. (2) Lokale Postgres-Verbindungsgrenze bei parallelen
  crm-Paket-Laeufen — betrifft nur lokale Testlaeufe, nicht CI (dort
  laufen Pakete vermutlich mit anderer Parallelitaet/Connection-Limits).

## Iteration 33 — c-cov-crm-advisoryprotocol — done — 2026-08-08 21:50
- commit: 2da5d08d
- verify vorgaenger: sauber (c98d4f9d ist ein reiner chore-Commit, der nur
  Commit-SHAs in bestehende Journal-Eintraege nachtraegt — kein Gateway/
  Route/Proto/Guard/neue-Tabelle betroffen). `git merge origin/main` war
  "Already up to date".
- gebaut: Neue Datei `postgres_repository_db_test.go` mit zehn
  DB-Integrationstests fuer `internal/crm/advisoryprotocol` (vorher 0
  DB-Tests im Paket — nur Mock-basierte `service_test.go`). Coverage
  40,0% -> 65,5% (der Rest ist ueberwiegend `GeneratePDF`, PDF-Rendering,
  kein Schreibpfad).
  Dabei DREI echte Produktionsbugs gefunden und gefixt, alle per
  DB-Test reproduziert statt spekuliert:
  (1) **Der eigentliche Zweck der Unit — Immutability-Race.** `Update`
  und `Delete` haengen ihre Query an `AND status='draft'`, pruefen aber
  nie `RowsAffected()`. Ein Race zwischen dem Service-Precondition-Check
  (`GetByID` + `status=="finalized"`) und dem eigentlichen Write haette
  ein zehn Jahre aufzubewahrendes, ausgehaendigtes Protokoll still
  ueberschreiben/loeschen koennen — `nil`-Error, 0 tatsaechlich
  geaenderte Zeilen. Fix: `RowsAffected()==0` -> `ErrProtocolFinalized`
  in beiden Methoden. Kein Mapping-Code noetig, `crm_grpc.go:2837`
  kennt den Sentinel bereits aus dem Service-Pfad.
  (2) **DATE-Spalten crashen jeden Read mit gesetztem Datum.** `date`,
  `birth_date`, `document_delivered_date`, `followup_date` sind Postgres
  `DATE` und werden im Binaerformat (OID 1082) uebertragen — pgx kann
  das nicht direkt in `*string` scannen (`cannot scan date ... in binary
  format into **string`). `GetByID`/`ListByContact` crashten also bei
  JEDEM Protokoll mit gesetztem Datum, nicht nur einem Edge-Case. Fix:
  `::text`-Cast auf alle vier Spalten in beiden SELECTs (Go-Feld bleibt
  `*string`, ISO-Format `yyyy-mm-dd` bleibt erhalten — Kommentar im
  Modell stimmt weiterhin).
  (3) **NOT-NULL-Verletzung auf dem Create-Pfad selbst.**
  `known_asset_classes`/`investment_purpose`/`warnings_given` sowie
  `horizon`/`delivery_form` sind `NOT NULL DEFAULT ...`-Spalten, aber
  `nullableTextArray` wandelte leere Arrays und `emptyToNil` wandelte
  leere Strings in NULL. `Service.Create` baut jedes neue Draft-Protokoll
  GENAU mit leeren Arrays und leerem `Horizon` — der allererste
  `repo.Create`-Aufruf nach jedem `Service.Create` schlug mit
  `null value in column "known_asset_classes" violates not-null
  constraint` fehl. Fix: neue Funktion `textArrayOrEmpty` (nil -> `[]`
  statt nil -> NULL) ersetzt `nullableTextArray` an allen sechs
  Stellen; `horizon`/`delivery_form` binden jetzt direkt den String statt
  durch `emptyToNil` zu laufen (beide Spalten haben ohnehin ein
  DB-seitiges `DEFAULT`, brauchen also nie NULL).
  Zusammengenommen: die Advisory-Protocol-Erstellung war vor diesem
  Fix fuer praktisch jedes reale Protokoll ungetestet und kaputt — der
  Happy Path "Draft anlegen, dann ansehen" ist per neuem Test
  `TestRepository_CreateGetByID_BlankDraftRoundTrips` jetzt exakt so
  abgedeckt, wie `Service.Create` das Objekt tatsaechlich baut.
  Weitere Tests: Create/GetByID-Roundtrip mit allen Feldern (Arrays,
  JSONB-Products, Pointer-Felder), ListByContact-Sortierung
  (`COALESCE(date, created_at::date) DESC, created_at DESC`) inkl.
  RLS-Leak-Probe mit gestohlener Tenant-ID, Update auf Draft (inkl.
  Products-JSON), HandOver setzt Status+Zeitstempel, ContactExists in
  beide Richtungen tenant-gescoped (WHERE-Parameter UND RLS-Session
  getrennt geprueft), GetReferralReport aggregiert korrekt und
  tenant-scoped.
- gate: `go build -p 2 ./internal/crm/... ./internal/gateway/...
  ./internal/server/... ./cmd/gateway/... ./cmd/crm/...` ok | `go vet
  ./internal/crm/advisoryprotocol/...` ok | `golangci-lint run --config
  .golangci.yml ./internal/crm/advisoryprotocol/...` 0 issues | `go test
  -count=1 ./internal/crm/advisoryprotocol/... ./internal/gateway/...`
  — beide Pakete gruen, **0 Skips**, `DATABASE_URL` gesetzt (Rolle
  kmuhub_app). Keine Route/kein Proto/keine Migration angefasst
  (Fix ist reines SQL/Go im Repository), daher kein separater
  `TestOpenAPIRouteDrift`-Lauf noetig — trotzdem mitgetestet, da das
  Paket von Gateway-Handlern importiert wird.
- mutations-probe: `if tag.RowsAffected() == 0` im `Update`-Pfad zu
  `!= 0` gedreht (macht die neue Race-Absicherung zum genauen Gegenteil:
  ein erfolgreiches Draft-Update liefert jetzt faelschlich
  `ErrProtocolFinalized`, ein Update auf ein finalisiertes Protokoll
  faelschlich `nil`). Ergebnis: GENAU
  `TestRepository_Update_DraftAppliesAllFieldsIncludingProductsJSON`
  und `TestRepository_Update_OnFinalizedReturnsErrProtocolFinalizedAndDoesNotMutate`
  wurden rot, alle 23 uebrigen Tests (inklusive der analogen
  Delete-Finalized-Probe, deren eigener RowsAffected-Check unveraendert
  blieb) bestanden weiter. Zurueckgedreht, `git diff --stat
  internal/crm/advisoryprotocol/postgres_repository.go` bestaetigt danach
  wieder den urspruenglichen (kumulierten) Fix-Diff, Endgate erneut
  gelaufen und gruen. Die drei echten Bugfunde hatten de facto bereits
  ihre eigene Mutations-Probe: jeder Test schlug mit dem urspruenglichen
  Code real fehl (NOT-NULL-Verletzung bzw. Scan-Fehler), bestand erst
  nach dem jeweiligen Fix.
- db-tests: 10 neue DB-Integrationstests, alle real gegen `kmuhub_app`
  gelaufen, **0 Skips**.
- offen: Kein DB-Gate-Ausfall. Ein Punkt fuer Luke: Punkt (3) oben wirft
  die Frage auf, ob das Beratungsprotokoll-Feature in Produktion je
  erfolgreich ein Draft angelegt hat — falls ja (z. B. durch einen
  Frontend-Pfad, der abweichend von `Service.Create` nicht-leere Arrays
  mitschickt), waere das ein Datenpunkt wert, das zu verifizieren; falls
  nein, war das Feature bislang komplett unbenutzbar. Naechste Unit im
  Backlog: `c-cov-biz-hr-employee`.
