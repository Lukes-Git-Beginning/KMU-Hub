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
- commit: nachgetragen im Folge-Commit
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
- commit: nachgetragen im Folge-Commit
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
