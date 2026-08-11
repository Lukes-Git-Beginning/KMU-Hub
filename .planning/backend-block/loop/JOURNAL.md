# Backend-Nachtloop — Journal Lauf 9

Append-only. Ein Eintrag je Iteration, **ans Dateiende**, nie einsortieren. Form und Pflichtzeilen
stehen in `ITERATION.md` Schritt 6.

Frühere Läufe liegen vollständig im Archiv:
`archive/lauf-1-2/` (58 Units) · `archive/lauf-3/` (61) · `archive/lauf-4/` (54) ·
`archive/lauf-5/` (41) · `archive/lauf-6/` (46) · `archive/lauf-7/` (71) ·
`archive/lauf-8/` (94, inkl. `logs/`).

---

## Laufkontext

- **Ausgangspunkt:** `backend-loop` auf `origin/main` gemergt (nicht rebased), Fast-Forward auf
  `10a1a26e`. `main` = `10a1a26e`. Produktion: Migrationskopf **310 clean**, 36 Container laufend /
  30 healthy / 0 unhealthy, `/health` mit 23 Services.
- **Migrationen:** Repo-Kopf = lokaler Kopf = Produktionskopf = **310**. Nächste freie **311** —
  aber immer zur Laufzeit ermitteln.
- **Lokale DB:** vor dem Start prüfen. Rolle muss `kmuhub_app` sein — `kmuhub` hat BYPASSRLS und
  würde jede RLS-Lücke durchwinken. `go test` ohne `DATABASE_URL` ist **kein** Gate.
- **Backlog:** `BACKLOG.yml`, Block A (10 Fix-Units) + Block B (11 Scan-Units). Null `blocked`- und
  null `done`-Units zum Laufbeginn — die Datei ist für diesen Lauf frisch aufgebaut, Lauf 8 liegt
  vollständig in `archive/lauf-8/`. `BACKLOG-NEXT.yml` ist leer: die zehn Fix-Units sind
  **verschoben**, nicht kopiert.
- **Fenster:** ein Lauf ab 16:00 (`-StartNotBefore "16:00"`), Deadline `-UntilTime "09:00"` als
  Sicherheitsnetz. Kein Pausenfenster. Ein Prozess, ein Push, ein CI-Lauf.
- **Workflow-Zustand beim Start:** `Claude PR Review` `disabled_manually`, `Security Review` vor dem
  Anlegen des Draft-PRs disabled (beide haben kein Draft-Gate und würden beim `opened`-Event zünden).

### Was dieser Lauf ist — und was nicht

Lauf 9 ist ein reiner **Fix- und Scan-Lauf**. **Keine Coverage-Units.**

Der Coverage-Engpass ist zu: Lauf 8 hat 47,7 → **60,0 %** gehoben, bei einem Gate von 15 %. Wichtiger
ist, was dabei sichtbar wurde: die vier Pakete mit den **schlimmsten** Bugs haben die **höchste**
Coverage — `notification/preference` 87,2 % (`UpsertQuietHours` schlägt bei jedem Aufruf fehl),
`document/virtual` 83,1 % (vier Queries auf eine gelöschte Spalte), `schichten` 79,7 % (Schichttausch
hat keinen funktionierenden Pfad), `biz/datev` 79,3 % (Upload seit zwei Monaten totalausgefallen).
Coverage misst keine Korrektheit. Mehr Prozente auf `gateway` (46,0 %, schwächstes Kernpaket) würden
dieselbe Bug-Klasse *erzeugen*, nicht finden — deshalb steht `gateway` in diesem Lauf nicht auf der
Liste.

**Block A** sind die zehn verifizierten Produktionsbugs aus Lauf 8, nach Schwere sortiert.
**Block B** sind elf Muster-Scans: die zehn Bugs teilen vier mechanisch auffindbare Muster
(Typ-Scan-Mismatch, `ON CONFLICT` gegen einen nicht existierenden Index, SQL auf gelöschte Spalten,
INSERT ohne `tenant_id`, `nil`-Slice als JSON `null`). Jede Scan-Unit legt ihre Funde als neue
Fix-Units am Backlog-Ende an — **Block C** entsteht damit zur Laufzeit und füllt den Rest des
Fensters.

### Neu in diesem Lauf

- **`neue-units:` ist Pflichtzeile** (seit `d6d80fcc`). Ein Fund ohne angelegte Unit ist kein Fund.
  In früheren Läufen sind verifizierte Bugs verlorengegangen, weil sie nur im Journal standen —
  drei der zehn Block-A-Units mussten bei der Nachbereitung von Lauf 8 nachgetragen werden.
- **`coverage:` ist ein Delta auf dem Paket der Unit**, nicht gegen den Laufstart. `coverage_start`
  nennt jetzt bei jeder Unit **exakt** das Paket, das sie anfasst — in der Lauf-8-Fassung standen
  bei fünf Units Elternpakete oder veraltete Werte (`internal/security 76,9 %` statt
  `internal/security/audit 66,1 %`), womit die Iteration nur `n.a.` schreiben konnte.
- **Der Drift-Check vergleicht nur noch den Kalendertag** (`d6d80fcc`). Vorher feuerte er
  minutengenau und damit in 90 von 94 Iterationen, obwohl die Nummer 94/94 stimmte — eine Warnung,
  die fast immer feuert, wird nicht gelesen.
- **Pin-Tests werden UMGEDREHT, nicht gelöscht.** Jede Block-A-Unit bringt einen Test mit, der heute
  das *kaputte* Verhalten assertiert. Nach dem Fix muss er das *korrekte* assertieren. Ein
  gelöschter Pin-Test ist eine verlorene Regression.
- **Die Schichttausch-Semantik ist vorab entschieden** (Luke, 2026-08-11): ein Tausch ist nur
  zwischen zwei bereits zugeordneten Mitarbeitern gültig, Fall 2 wird ein Validierungsfehler. An
  genau dieser offenen Frage hat Iteration 94 in Lauf 8 gehalten — sie ist beantwortet und nicht
  erneut aufzumachen.
- **Neu auf `main` seit Lauf 8:** eine öffentliche Route `GET/POST /reset-password` im Gateway
  (`10a1a26e`, embedded HTML-Seite für den Passwort-Reset-Mail-Link). Beim Anfassen von
  `cmd/gateway/main.go` nicht versehentlich ausbauen.

---

## Iteration 1 — fix-audit-list-verifychain-ip-address-scan — done — 2026-08-11 16:06
- commit: 12be7c3f
- gebaut: `List()` und `VerifyChain()` in `internal/security/audit/postgres_repository.go` casten
  `ip_address` jetzt mit `COALESCE(host(ip_address), '')` statt es roh zu scannen (Vorlage:
  `internal/auth/postgres_repository.go`). Testdatei umgebaut: `TestPostgresRepository_List_
  IPAddressScanBug` (pinnte den Fehler) ist ersetzt durch fünf echte Assertions (IP gesetzt, IP
  NULL, Result-Filter, Pagination Limit/Offset) plus zwei neue `VerifyChain`-Tests (valide
  Single-Row-Range, erkannte Manipulation). Die in Iteration 40 (Lauf 8) zurückgestellten
  Filter-/Pagination-/Chain-Assertions sind damit nachgezogen.
- gate: build ok | vet ok | lint ok (0 issues) | test ok | migration n.a. (reiner Query-Fix,
  kein Schema-Änderung) | rls-smoke ok (TestRLS_AuditLog_* liefen als Teil des Pakettests grün,
  keine Policy/Tabelle angefasst)
- coverage: internal/security/audit 66,1 % -> 80,0 % (lokal gemessen, `go tool cover -func`)
- mutations-probe: `COALESCE(host(ip_address), '')` in `List()`s dataQuery zurück auf rohes
  `ip_address` gesetzt -> `TestPostgresRepository_List_ReturnsSeededEntry_WithIPAddress` wurde
  rot mit exakt dem ursprünglich dokumentierten pgx-Scanfehler ("cannot scan inet (OID 869) in
  binary format into *string"), zurückgedreht, `git diff --stat` zeigt wieder nur die
  ursprünglichen 2 Zeilen (List + VerifyChain je 1 Zeile).
- verify vorgaenger: n.a. (Iteration 1 dieses Laufs, kein Vorgänger-Commit)
- neue-units: fix-audit-verifychain-timestamp-precision-mismatch (Block C, ans Backlog-Ende
  gehängt)
- offen: Beim Schreiben der beiden neuen `VerifyChain`-Tests kam ein zweiter, unabhängiger Bug
  zum Vorschein: `Create()` hasht `entry.Timestamp` mit voller (potenziell Sub-Mikrosekunden-)
  Go-Präzision via `RFC3339Nano`, aber `audit_log.timestamp` ist `TIMESTAMPTZ` und speichert nur
  Mikrosekunden — jeder spätere `VerifyChain`-Aufruf rechnet den Hash aus dem gekappten,
  zurückgelesenen Timestamp neu und bekommt bei nicht-null Sub-Mikrosekunden-Ziffern (auf dieser
  Windows-Maschine in ~4 von 5 Stichproben der Fall) ein anderes Ergebnis als beim Insert — eine
  intakte Kette wird als manipuliert gemeldet. Nicht Teil dieser Unit (andere Ursache, andere
  Zeile), daher NICHT mitgefixt, sondern als eigene Unit
  `fix-audit-verifychain-timestamp-precision-mismatch` angelegt (inkl. Fix-Vorschlag und
  Hinweis auf die Nicht-rückwirkende Grenze für bereits geschriebene Einträge). Die beiden neuen
  `VerifyChain`-Tests in dieser Iteration umgehen das Problem bewusst durch einen auf
  Mikrosekunden gekappten Test-Timestamp — das ist im Testkommentar referenziert, damit die
  nächste Iteration den Workaround nicht für den eigentlichen Fix hält.
  Zusätzlich (kein eigener Bug, nur Testartefakt): `SeedRow`-Aufrufe auf `audit_log`, die
  `target`/`target_type`/`user_agent` weglassen, erzeugen NULL in diesen nullable
  VARCHAR-Spalten und lösen denselben NULL-in-`string`-Scanfehler aus wie `ip_address` — aber
  nur über den Test-Bypass von `Create()`, nicht über einen echten Aufrufer (der einzige
  Produktionsschreibpfad in `Create()` setzt diese Felder immer auf `""`, nie NULL). Deshalb
  keine eigene Unit, nur hier vermerkt.

## Iteration 2 — fix-datev-upload-repo-missing-tenant-id — done — 2026-08-11 16:29
- commit: 50e89714
- gebaut: `UpsertUploadConfig` und `CreateUploadLog` in
  `internal/biz/datev/postgres_upload_repo.go` schreiben `tenant_id` jetzt in beide INSERTs.
  Neuer `tenantForWrite(ctx)`-Helper loest die Tenant-ID direkt aus dem Context (Vorlage:
  `internal/notification/integration/postgres_repository.go`s gleichnamiges Muster fuer
  denselben NOT-NULL+FORCE-RLS-Fall) — kein Modellfeld, keine Handler-Aenderung noetig, weil
  der Interceptor jede reale gRPC-Anfrage schon mit der Tenant-ID im Context ausstattet. Neuer
  Sentinel `ErrTenantMissing`. `UpsertUploadConfig`s `ON CONFLICT (config_id)`-Ziel war bereits
  korrekt (echter `UNIQUE(config_id)`-Constraint aus Migration 000056), kein MUSTER-A-Fund an
  dieser Stelle.
- gebaut (Tests): `TestUpsertUploadConfig_FailsNotNullTenantID` und
  `TestCreateUploadLog_FailsNotNullTenantID` sind durch
  `TestUpsertUploadConfig_InsertsThenUpdatesOnConflict` (Insert- und ON-CONFLICT-Update-Pfad)
  und `TestCreateUploadLog_InsertsRow` ersetzt. Neu dazu:
  `TestUpsertUploadConfigAndCreateUploadLog_WriteLandsInCallerTenant` (RLS-Nachweis nach dem
  `AssertRowCount`-Muster aus `crm/consent/tenant_write_test.go` — eigener Tenant sieht die
  Zeile, ein fremder Tenant nicht) sowie zwei `ErrTenantMissing`-Tests fuer den Context-ohne-
  Tenant-Fall.
- gate: build ok (`./internal/biz/...`) | vet ok | lint ok (0 issues) | test ok (0 Skips,
  `DATABASE_URL` gegen `kmuhub_app`) | migration n.a. (kein Schema-Fund, reiner Repo-Fix) |
  rls-smoke ok (neuer Test + bestehender `TestTenantIsolation_Datev_DB` beide gruen) |
  `go test -count=1 ./internal/biz/...` gesamt gruen (alle Unterpakete)
- coverage: internal/biz/datev 79,3 % -> 79,7 % (lokal gemessen, `go tool cover -func`)
- mutations-probe: `tenant_id` aus Spaltenliste und VALUES von `UpsertUploadConfig`s INSERT
  entfernt (Parameterindizes zurueckgesetzt, `tenantID` mit `_ = tenantID` totgelegt) ->
  `TestUpsertUploadConfig_InsertsThenUpdatesOnConflict` und
  `TestUpsertUploadConfigAndCreateUploadLog_WriteLandsInCallerTenant` wurden rot, beide mit
  `ERROR: new row violates row-level security policy for table "datev_upload_configs"
  (SQLSTATE 42501)` — RLS greift jetzt VOR dem alten NOT-NULL-Fehler, weil `FORCE RLS`
  (Migration 000122) bereits ein `tenant_id`-loses Insert ablehnt. Zurueckgedreht,
  `git diff --stat` zeigt wieder nur die ursprueliche Aenderung.
- verify vorgaenger: sauber (Commit `12be7c3f`, Iteration 1 — reiner Query-Cast-Fix, kein
  gRPC-Bypass, kein Proto/Migrations-Drift, kein Guard, kein Wire-Shape-, Routen- oder
  Tenant-Fund; neue Unit aus Iteration 1 korrekt am Backlog-Ende angelegt und verifiziert)
- neue-units: keine
- offen: `UpdateDatevUploadConfig`/`ListDatevUploadLogs` im Gateway-Handler
  (`internal/server/datev_upload_grpc.go`) validieren weiterhin nur `req.GetTenantId() != ""`
  ohne den Wert zu nutzen — das ist unveraendert (Ctx-Tenant zaehlt, nicht das Request-Feld),
  aber falls das Feld in der Response-Semantik je gebraucht wird, gehoert eine Pruefung
  Request-Tenant == Ctx-Tenant dorthin. Kein Fund dieser Iteration, nur als Beobachtung
  vermerkt. Produktions-Zeilenzahl in `datev_upload_configs`/`datev_upload_log` (vermutlich 0)
  wurde nicht geprueft — das ist ein reiner Produktions-Read und nicht Teil dieses Laufs.

## Iteration 3 — fix-schichten-swap-assignments-unique-violation — done — 2026-08-11 16:38
- commit: f379b30a
- gebaut: `SwapAssignmentsForRequest` (`internal/schichten/postgres_repository.go`) beheben
  beide Faelle des Root Causes in einem Rutsch. Neue Migration 000311 macht
  `uq_shift_assignments_tenant` `DEFERRABLE INITIALLY IMMEDIATE` (Drop+Re-Add, jeder andere
  Schreibpfad bleibt unveraendert, weil der Check nur innerhalb dieser einen Transaktion per
  `SET CONSTRAINTS ... DEFERRED` aufgeschoben wird). Die Funktion sperrt jetzt zuerst die Zeile
  des Tauschpartners per `SELECT ... FOR UPDATE` (liefert 0 Zeilen -> neuer Sentinel
  `ErrSwapPartnerNotAssigned`, gemappt auf `codes.FailedPrecondition` in
  `internal/server/schichten_grpc.go`, analog zu `ErrArbzgViolation`) und scopet BEIDE UPDATEs
  auf die jeweilige Zeilen-ID statt auf `(shift_id, employee_id)` — genau die Scoping-Luecke,
  die den stillen No-Op verursacht hat. Per vorab entschiedener Semantik (siehe Unit-`notes`)
  ist ein Tausch mit einem noch nicht zugeordneten Partner jetzt ein abgelehnter, kein stiller
  Erfolg. `ApproveSwapRequest` (service.go) brauchte keine Aenderung — es reicht Repo-Fehler
  bereits unveraendert durch, bevor es `UpdateSwapRequestStatus` aufruft.
- gebaut (Tests): `TestSwapAssignmentsForRequest_SilentlyNoOpsWhenTargetNotYetOnShift` ->
  `TestSwapAssignmentsForRequest_RejectsPartnerNotYetOnShift` (erwartet jetzt
  `ErrSwapPartnerNotAssigned`, prueft dass nichts geschrieben wurde);
  `TestSwapAssignmentsForRequest_FailsWhenBothEmployeesAlreadyAssignedToShift` ->
  `TestSwapAssignmentsForRequest_SwapsBothEmployeesAlreadyAssignedToShift` (erwartet jetzt den
  tatsaechlichen Tausch, prueft dass die Zeilen-IDs mit den employee_id-Werten wandern).
- gate: build ok (`./internal/schichten/... ./internal/gateway/... ./cmd/schichten/...
  ./cmd/gateway/...`) | vet ok | lint ok (0 issues, `internal/schichten` + `internal/server`) |
  migration ok (`migrate up` gegen lokale DB, Kopf jetzt 311) | test ok (0 Skips,
  `DATABASE_URL` gegen `kmuhub_app`, `./internal/schichten/...` und `./internal/server/` gruen)
  | `go test ./internal/gateway/ -run TestOpenAPIRouteDrift` gruen (834 Routen gegen 836
  Spec-Pfade — keine neue Route, reine Vorsichtspruefung wegen `schichten_grpc.go`-Anfassung)
  | rls-smoke n.a. (keine RLS-Policy angefasst, nur die Deferrability einer bestehenden
  UNIQUE-Constraint auf `shift_assignments`; bestehende Tenant-Isolationstests aus
  `tenant_write_test.go`/`tenant_isolation_phase2_test.go` liefen im vollen Paketlauf mit und
  waren gruen)
- coverage: internal/schichten 79,7 % -> 79,4 % (beide lokal gemessen, `go tool cover -func`,
  vor/nach per `git stash`/`pop` isoliert — kein Coverage-Ziel dieser Unit, der leichte
  Ruecklauf kommt vom neuen Fehlerpfad (`FOR UPDATE`-Miss -> `ErrSwapPartnerNotAssigned`), der
  von den neuen Tests nur teilweise durchlaufen wird)
- mutations-probe: zweites UPDATE zurueck auf `WHERE shift_id = $2 AND employee_id = $3 AND
  tenant_id = $4` gesetzt (alte Scoping-Luecke) -> `TestSwapAssignmentsForRequest_
  SwapsBothEmployeesAlreadyAssignedToShift` wurde rot (`duplicate key value violates unique
  constraint "uq_shift_assignments_tenant"`, weil das mutierte UPDATE jetzt BEIDE Zeilen trifft
  und beide auf `employee_id = requester` setzt). Zurueckgedreht, `git diff --stat` zeigt wieder
  nur die urspruengliche Aenderung (33 insertions, 4 deletions in `postgres_repository.go`).
- verify vorgaenger: sauber (Commit `50e89714`, Iteration 2 — reiner Repository-Fix mit
  Context-abgeleiteter `tenant_id`, kein gRPC-Bypass, kein Stub, kein Proto/Migrations-Drift,
  kein Guard-Fund, kein Wire-Shape- oder Routen-Fund; `tenantForWrite` nutzt Ctx statt
  Request-Feld, deckt sich mit der eigenen `offen:`-Beobachtung aus Iteration 2)
- neue-units: keine
- offen: Die Semantik-Entscheidung "Tausch nur zwischen zwei bereits Zugeordneten" war bereits
  in den Unit-`notes` fixiert (Luke, 2026-08-11) — hier nicht neu aufgemacht. Kein sonstiger
  Fund. Block-B-Scan-Units koennten dieselbe Deferrable-Constraint-Klasse (ON-CONFLICT- bzw.
  UNIQUE-Scoping-Probleme bei Mehrzeilen-Swaps) an anderer Stelle im Repo finden — nicht
  vorab durchsucht, das ist Aufgabe der jeweiligen Scan-Unit selbst.

## Iteration 4 — fix-notification-quiet-hours-conflict-index — done — 2026-08-11 16:40
- commit: f30636b7
- gebaut (Migration 000312): `notification_quiet_hours` verliert den inline
  `UNIQUE(user_id)` aus 000022 und bekommt `idx_notification_quiet_hours_user
  (tenant_id, user_id)`; `idx_notification_preferences_module_default` wird von
  `(user_id, module_id)` auf `(tenant_id, user_id, module_id)` verbreitert (Praedikat
  unveraendert). Beide neuen Indexe sind Supersets, kollidieren also mit keiner bestehenden
  Zeile. Down-Migration stellt beide Altzustaende her (Roundtrip `down 1` + `up` gegen die
  lokale DB gefahren, Kopf danach wieder 312 clean).
- gebaut (Code): `UpsertQuietHours` brauchte wie vermutet KEINE Aenderung — geprueft, nicht
  angenommen (ON CONFLICT nennt nur Spalten, keinen Constraint-Namen; Test gruen nach der
  Migration). Der ZWEITE, im Backlog unverifizierte Verdacht an `UpsertPreference` ist REAL,
  aber anders als vermutet: er schreibt KEINE Dublette, sondern schlaegt fehl. Neue Funktion
  `upsertPreferenceConflict(pref)` waehlt den Arbiter passend zur Zeile — Event-Type-Zeilen
  bekommen den Event-Type-Index, Modul-Defaults (`event_type_key IS NULL`) den
  Modul-Default-Index, eine Zeile ohne beides einen reinen INSERT (kein Index deckt sie ab,
  ein Konflikt ist konstruktiv unmoeglich).
- gate: build ok (`./internal/notification/... ./internal/gateway/... ./cmd/notification/...
  ./cmd/gateway/...`) | vet ok | lint ok (0 issues) | migration ok (`up`, `down 1`, `up`,
  Kopf 312) | test ok (0 Skips bei 222 Tests, `DATABASE_URL` gegen `kmuhub_app`;
  `./internal/notification/...` und `./internal/server/` gruen) | rls-smoke ok (beide
  angefasste Tabellen: eigener Tenant 1 / fremder Tenant 0, als `kmuhub_app`) |
  `go test ./internal/gateway/ -run TestOpenAPIRouteDrift` gruen (keine Route angefasst,
  reine Vorsichtspruefung)
- coverage: internal/notification/preference 87,2 % -> 87,4 % (beide selbst gemessen,
  `go tool cover -func`; Vorher-Wert per `migrate down 1` + `git stash -u` isoliert und
  identisch mit `coverage_start:`)
- mutations-probe: ZWEI Proben, weil der Fix zwei Traeger hat.
  (1) Migration: `migrate down 1` (Altzustand der Indexe) -> `TestPostgresRepository_
  QuietHoursRoundTrip` UND `TestPostgresRepository_UpsertModuleDefaultTwice` beide rot mit
  SQLSTATE 42P10; `up` -> beide wieder gruen.
  (2) Code: den `ModuleID != nil`-Zweig von `upsertPreferenceConflict` auf den
  Event-Type-Arbiter zurueckgesetzt -> `TestPostgresRepository_UpsertModuleDefaultTwice` rot
  mit SQLSTATE 23505 auf `idx_notification_preferences_module_default` (genau der Fehler, den
  der Fix beseitigt). Zurueckgedreht, `git diff --stat` zeigt wieder nur die urspruengliche
  Aenderung (3 Dateien, 184 insertions / 23 deletions, plus die zwei untracked
  Migrationsdateien).
- verify vorgaenger: sauber (Commit `f379b30a`, Iteration 3 — kein gRPC-Bypass (`schichten_grpc.go`
  bekommt nur eine Fehler-Mapping-Zeile), kein Stub, kein `.proto`, kein neuer
  `RequirePermission`-Guard, keine neue Tabelle, keine Wire-Shape-Aenderung, keine neue Route.
  Migration 000311 ist neu angelegt, keine ausgerollte Migration angefasst;
  `SET CONSTRAINTS ... DEFERRED` gilt nur fuer die eigene Transaktion und der neue
  `FOR UPDATE`-Lookup laeuft vor beiden UPDATEs)
- neue-units: fix-notification-mutes-unique-missing-tenant (am Backlog-Ende)
- offen: (1) Der Backlog-Verdacht zu `UpsertPreference` war in der Wirkung falsch beschrieben —
  eine Dublette ist unmoeglich, weil `idx_notification_preferences_module_default` UNIQUE ist;
  der zweite Upsert eines Modul-Defaults schlug mit 23505 fehl. Ergebnis ist derselbe Fix, aber
  wer die Unit-`notes` als Beleg liest, bekommt die falsche Fehlerklasse.
  (2) `notification_mutes` traegt dieselbe Index-Drift (`UNIQUE(user_id, module_id,
  resource_id)` ohne `tenant_id`) — hier NICHT mitgefixt, weil ausserhalb des `done_when` und
  eine eigene Migration noetig; als Unit angelegt (siehe `neue-units:`). Damit sind alle vier
  Unique-Indexe des Notification-Schemas erfasst.
  (3) `GetQuietHours` scannt die `time`-Spalten `start_time`/`end_time` in Go-`string` —
  funktioniert mit pgx v5 (jetzt erstmals mit einer echten Zeile bewiesen, vorher konnte nie
  eine geschrieben werden), liefert aber `"18:00:00"`, nicht `"18:00"`. Die Tests vergleichen
  deshalb `[:5]`. Kein Fehler, aber das Frontend bekommt ein anderes Format, als es beim
  Schreiben schickt — pruefen, ob der FE-Typ das vertraegt.

## Iteration 5 — fix-document-virtual-users-display-name — done — 2026-08-11 16:50
- commit: (siehe git log nach diesem Eintrag)
- gebaut: alle vier SELECT-Listen in `internal/document/virtual/postgres_repository.go`
  (ListChatFiles, ListTaskFiles, beide chat/task-Zweige der ListAll-UNION) lesen jetzt
  `COALESCE(NULLIF(TRIM(u.first_name || ' ' || u.last_name), ''), u.email) AS
  uploaded_by_name` statt des nichtexistenten `u.display_name` (users hat seit Migration
  000001 kein display_name, first_name/last_name sind NOT NULL DEFAULT ''). Keine Migration
  noetig, reiner SQL-Text-Fix. `ListEmailAttachments` (liest `eacc.display_name` von
  `email_accounts`, existiert dort wirklich) unveraendert.
- gebaut (Tests): die drei "ColumnBug"-Tests aus Iteration 86 (Lauf 8) auf das korrekte
  Verhalten umgestellt statt geloescht — `ListChatFiles_ColumnBug` ->
  `ListChatFiles_UploadedByName`, `ListTaskFiles_ColumnBugViaProjectMembership` ->
  `ListTaskFiles_UploadedByNameViaProjectMembership`,
  `ListAll_UnionColumnBugTriggeredByEmailOnlyAccess` ->
  `ListAll_UnionAcrossAllSourcesWithEmailOnlyAccess`; alle drei pruefen jetzt Erfolg + den
  korrekten `uploaded_by_name`. Neuer Test
  `ListChatFiles_UploadedByNameFallsBackToEmail` deckt den Email-Fallback bei leeren
  first_name/last_name ab (belegt beide done_when-Punkte: Name-Anzeige UND Email-Fallback).
  Der erklaerende Datei-Header-Kommentar zum "unfixed bug" ist entfernt, da er den jetzt
  gefixten Zustand falsch beschrieben haette.
- gate: build ok (`./internal/document/... ./internal/gateway/...`) | vet ok | lint ok
  (0 issues) | test ok (0 Skips bei 17 Tests in `internal/document/virtual`, `DATABASE_URL`
  gegen `kmuhub_app`; `./internal/document/...` komplett gruen) | migration n.a. (keine
  Migration) | rls-smoke n.a. (keine Tabelle/Policy angefasst, reiner Query-Textfix)
- coverage: internal/document/virtual 83,1 % -> 81,7 % (beide selbst gemessen,
  `go tool cover -func`; Vorher-Wert per `git stash push -u` auf genau diese zwei Dateien
  isoliert, deckt sich mit `coverage_start:`. Der leichte Ruecklauf kommt vom neuen
  Email-Fallback-Zweig der COALESCE/NULLIF/TRIM-Kette, der von den bestehenden Tests nur
  teilweise durchlaufen wird — kein Coverage-Ziel dieser Unit)
- mutations-probe: Zeile 61 (ListChatFiles-SELECT) zurueck auf
  `COALESCE(u.display_name, u.email)` gesetzt -> `TestPostgresRepository_
  ListChatFiles_UploadedByName` UND `TestPostgresRepository_
  ListChatFiles_UploadedByNameFallsBackToEmail` beide rot mit
  `ERROR: column u.display_name does not exist (SQLSTATE 42703)`. Zurueckgedreht,
  `git diff --stat` zeigt wieder nur die urspruengliche Aenderung (4 Zeilen geaendert in
  postgres_repository.go).
- verify vorgaenger: sauber (Commit `f30636b7`, Iteration 4 — kein gRPC-Bypass, kein Stub,
  Migration 000312 hat up UND down, keine ausgerollte Migration angefasst, kein neuer
  `RequirePermission`-Guard, keine neue Tabelle, keine Wire-Shape-Aenderung, keine neue Route)
- neue-units: keine
- offen: keine neuen Funde. Reiner Query-Textfix ohne Migrations-/RLS-Beruehrung.

## Iteration 6 — fix-crm-erasure-double-count — done — 2026-08-11 16:55
- commit: (siehe git log nach diesem Eintrag)
- gebaut: `CRMErasureHandler.ExecuteErasure` (internal/security/gdpr/erasure.go) zieht die
  bisherigen zwei sequenziellen UPDATEs auf `activities` (erst assigned_to=NULL WHERE
  assigned_to=$1, danach description=NULL WHERE created_by=$1 AND (assigned_to IS NULL OR
  assigned_to != $1)) zu einem einzigen UPDATE zusammen: `SET assigned_to = CASE WHEN
  assigned_to = $1 THEN NULL ELSE assigned_to END, description = NULL WHERE assigned_to = $1
  OR created_by = $1`. Eine Aktivitaet, die sowohl assigned_to als auch created_by = user war,
  matcht jetzt genau einmal statt zweimal (das zweite UPDATE traf sie vorher erneut, weil ihr
  assigned_to durch das erste UPDATE bereits NULL war). Datenverhalten unveraendert, nur die
  zurueckgegebene Zahl ist jetzt korrekt.
- gebaut (Tests): `TestCRMErasureHandler_ExecuteErasure_Integration` erwartet jetzt 4 statt 5,
  `TestCRMErasureHandler_ExecuteErasure_IgnoresAction` jetzt 2 statt 3 -- beide Kommentare auf
  den alten Doppelzaehl-Bug entfernt, Assertions unveraendert ansonsten (Daten waren immer
  schon korrekt).
- gate: build ok (`./internal/security/... ./cmd/gateway/...`) | vet ok | lint ok (0 issues)
  | test ok (`./internal/security/gdpr/` und `./internal/security/...` komplett gruen,
  DATABASE_URL gegen kmuhub_app) | migration n.a. (keine Migration, keine Tabellenaenderung)
  | rls-smoke n.a. (kein RLS-/Policy-Bezug, reiner Query-Logik-Fix) | TestOpenAPIRouteDrift ok
  (keine Route beruehrt, nur zur Sicherheit mitgelaufen)
- coverage: internal/security/gdpr 60,6 % -> 60,5 % (beide selbst gemessen per
  `go tool cover -func`, Vorher-Wert per `git stash push -u` auf genau die zwei geaenderten
  Dateien isoliert; deckt sich mit `coverage_start:`. Leichter Ruecklauf ist Rauschen aus dem
  entfernten zweiten Codepfad, kein Coverage-Ziel dieser Unit)
- mutations-probe: `assigned_to = CASE WHEN ... END` durch das alte bedingungslose
  `assigned_to = NULL` ersetzt -> `TestCRMErasureHandler_ExecuteErasure_Integration` sofort
  rot ("an activity assigned to somebody else must keep its assignee"). Zurueckgedreht,
  `git diff --stat internal/security/gdpr/erasure.go` zeigt wieder nur den urspruenglichen
  Fix-Diff (7 insertions, 17 deletions durch die Zusammenlegung zu einem UPDATE).
- verify vorgaenger: sauber (Commit `31e22ac4`, Iteration 5 — reiner Query-Textfix in
  document/virtual, kein gRPC-Bypass, kein Stub, kein Proto/Migration/Guard/Tenant/Wire-Shape/
  Route-Bezug)
- neue-units: fix-work-erasure-task-double-count (am Backlog-Ende) — identische
  Doppelzaehl-Bugklasse in der Schwesterfunktion `WorkErasureHandler.ExecuteErasure` (tasks
  statt activities), gefunden beim Root-Cause-Grep nach demselben Muster; nicht mitgefixt, weil
  ausserhalb des `done_when` dieser Unit und die Funktion bisher gar keinen Execute-Test hat.
- offen: keine. Reiner Zaehl-Logik-Fix ohne Migrations-/RLS-Beruehrung.

## Iteration 7 — fix-email-attachment-download-metadata-wrong-message-id — done — 2026-08-11 16:59
- commit: (siehe git log nach diesem Eintrag)
- gebaut: `attachment.Repository` bekommt eine neue Methode `GetByID(ctx, id, tenantID)
  (*models.EmailAttachment, error)` — Implementierung in `PostgresRepository.GetByID`
  (SELECT auf `email_attachments WHERE id = $1 AND tenant_id = $2`, mappt `pgx.ErrNoRows`
  auf `ErrAttachmentNotFound`, exakt das Muster von `GetMinIOKeyByID` daneben).
  `attachment.Service.GetByID` reicht das durch. `EmailGRPCServer.GetAttachmentDownloadURL`
  (internal/server/email_grpc.go:1133) ruft jetzt `s.attachmentService.GetByID(ctx, id,
  tenantID)` statt des fest verdrahteten `GetByMessage(ctx, uuid.Nil, tenantID)` — damit
  werden filename/content_type/size_bytes fuer Anhaenge an echten (gesendeten/empfangenen)
  Nachrichten korrekt befuellt; das Pre-Send-Upload-Verhalten (MessageID=uuid.Nil) bleibt
  unveraendert, weil die neue Query direkt ueber die Attachment-ID sucht statt ueber die
  MessageID.
- gebaut (Tests/Mocks): `stubAttachmentRepo.GetByID` (internal/server, Testfixture) und
  `MockRepository.GetByID` (internal/email/attachment/service_test.go) ergaenzt, beide fuer
  Interface-Konformitaet. `TestGetAttachmentDownloadURL_MetadataEmptyForRealMessageAttachment`
  in `TestGetAttachmentDownloadURL_MetadataPresentForRealMessageAttachment` umbenannt und auf
  das korrekte Verhalten umgestellt (erwartet jetzt gefuellte filename/content_type/
  size_bytes statt leerer Felder), Kommentar auf den gefixten Zustand angepasst.
  `TestGetAttachmentDownloadURL_MetadataPresentForPreSendAttachment` unveraendert (deckte den
  funktionierenden Pfad schon vorher ab, bleibt gruen).
- gate: build ok (`./internal/email/... ./internal/server/... ./cmd/email/... ./cmd/gateway/...`)
  | vet ok | lint ok (0 issues, `./internal/email/... ./internal/server/...`) | test ok
  (`./internal/server/` 0 Skips bei DATABASE_URL gegen kmuhub_app; `./internal/email/...` alle
  12 Unterpakete gruen; `./internal/gateway/` inkl. TestOpenAPIRouteDrift gruen — keine Route
  angefasst, nur zur Sicherheit mitgelaufen) | migration n.a. (keine Tabellenaenderung)
  | rls-smoke n.a. (kein RLS-/Policy-Bezug, reiner Repository-Methoden-Fix)
- coverage: internal/server 70,0 % -> 70,0 % (selbst gemessen per `go tool cover -func`;
  deckt sich mit coverage_start. Unveraendert, weil dies eine Fix-Unit ist, keine
  Coverage-Unit — der neue Code (`GetByID` in Repository/Service/PostgresRepository) liegt
  ausserhalb von internal/server)
- mutations-probe: Zeile 1133 (email_grpc.go) testweise auf
  `s.attachmentService.GetByID(ctx, uuid.Nil, tenantID)` zurueckgesetzt ->
  `TestGetAttachmentDownloadURL_MetadataPresentForRealMessageAttachment` UND
  `TestGetAttachmentDownloadURL_MetadataPresentForPreSendAttachment` beide rot (leere statt
  gefuellter Metadaten). Zurueckgedreht, `git diff --stat internal/server/email_grpc.go`
  zeigt wieder nur den urspruenglichen Fix-Diff (4 insertions, 8 deletions).
- verify vorgaenger: sauber (Commit `b04c41ad`, Iteration 6 — reiner UPDATE-Zusammenlegungsfix
  in security/gdpr, kein gRPC-Bypass, kein Stub, kein Proto/Migration/Guard/Tenant/
  Wire-Shape/Route-Bezug)
- neue-units: keine
- offen: keine. Reiner Repository-/Handler-Fix ohne Migrations-/RLS-Beruehrung.

## Iteration 8 — fix-email-send-missing-tenant-id — done — 2026-08-11 17:12
- commit: (siehe git log nach diesem Eintrag)
- gebaut: `send.Service.Send` und `.SaveDraft`
  (`internal/email/send/service.go`) setzen `TenantID: input.TenantID` jetzt auf dem
  konstruierten `models.EmailMessage`, bevor `messageCreator.Create` aufgerufen wird —
  beide Felder existierten auf `SendInput`/`DraftInput`, wurden aber nie gelesen.
  Root-Cause-Grep (`grep -rn "models.EmailMessage{" internal/`) fand eine dritte,
  unabhaengige Fundstelle mit demselben Bug: `Worker.envelopeToMessage`
  (`internal/email/sync/worker.go:422`, IMAP-Sync fuer eingehende Mails) konstruiert
  ebenfalls ohne `TenantID`, obwohl `w.account.TenantID` bereits an zwei anderen Stellen
  in derselben Datei genutzt wird (`syncCycle`, Zeilen 141/180) — mitgefixt in derselben
  Unit, weil identischer Root Cause (Struct-Konstruktion vor `Create`/`CreateSynced`, die
  beide auf dasselbe `message.PostgresRepository.Create` mit `tenant_id NOT NULL` +
  `FORCE ROW LEVEL SECURITY` (Migration 000122, `enable_tenant_rls('email_messages')`)
  laufen).
- gebaut (Tests): `TestSendEmail`/`TestSaveDraft` (`internal/server/
  email_grpc_messages_send_test.go`) pruefen jetzt zusaetzlich, dass die gespeicherte
  Nachricht ueber einen tenant-gescopten `GetByID` mit dem Tenant des Senders auffindbar
  ist (der Stub-Repo verweigert bei Tenant-Mismatch, exakt das Muster von echtem RLS).
  `TestSaveDraft` im `send`-Paket selbst (`service_test.go`) setzt jetzt `input.TenantID`
  und prueft `msg.TenantID`. `TestEnvelopeToMessage`
  (`internal/email/sync/helpers_test.go`) bekommt eine zusaetzliche Assertion
  `assert.Equal(t, w.account.TenantID, msg.TenantID)`.
- gate: build ok (`./internal/email/... ./internal/server/... ./cmd/email/...
  ./cmd/gateway/...`) | vet ok | lint ok (0 issues, `./internal/email/...
  ./internal/server/...`) | test ok (0 Skips, `DATABASE_URL` gegen `kmuhub_app`;
  `./internal/email/...` alle 12 Unterpakete gruen, `./internal/server/` gruen) |
  migration n.a. (kein Schema-/Policy-Fund, reiner Code-Fix) |
  `go test ./internal/gateway/ -run TestOpenAPIRouteDrift` gruen (834/836, keine Route
  angefasst, reine Vorsichtspruefung) | rls-smoke n.a. im engeren Sinn (keine Tabelle/
  Policy angefasst) — der reale RLS-INSERT-Pfad fuer `email_messages` bleibt vollstaendig
  durch das bestehende, unveraenderte `TestPostgresRepository_CreateAndGetByID`
  (`internal/email/message/postgres_repository_test.go`, echte DB, echter
  `testutil.WithTenantCtx`) abgedeckt und lief gruen mit
- coverage: internal/email/send 57,6 % -> 57,6 % (selbst gemessen, `go tool cover -func`;
  deckt sich mit `coverage_start:`. Unveraendert, weil die zwei neuen Zeilen bereits von
  bestehenden Testpfaden durchlaufen wurden)
- mutations-probe: ZWEI Proben, weil der Fix zwei unabhaengige Traeger hat.
  (1) `TenantID: input.TenantID,` aus `Send()`s Struct-Literal entfernt ->
  `TestSendEmail` (`internal/server`) wurde rot mit "email message not found" (der neue
  tenant-gescopte `GetByID`-Nachweis schlaegt exakt wie erwartet fehl). Zurueckgedreht,
  `git diff --stat internal/email/send/service.go` zeigt wieder nur die urspruengliche
  Aenderung (2 insertions).
  (2) `TenantID: w.account.TenantID,` in `envelopeToMessage` auf `uuid.Nil` gesetzt ->
  `TestEnvelopeToMessage/maps_addresses,_flags_and_in-reply-to`
  (`internal/email/sync`) wurde rot (Nil-UUID statt der erwarteten Tenant-ID).
  Zurueckgedreht, `git diff --stat internal/email/sync/worker.go` zeigt wieder nur die
  urspruengliche Aenderung (1 insertion).
- verify vorgaenger: sauber (Commit `8e00f87a`, Iteration 7 — Handler geht ueber
  `attachmentService.GetByID` (Service-Layer, kein gRPC-Bypass), kein Stub, kein
  Proto/Migrations-Drift, kein neuer Guard, keine neue Tabelle, keine Wire-Shape- oder
  Routen-Aenderung; Diff deckt sich mit dem Journal-Eintrag aus Iteration 7)
- neue-units: keine
- offen: keine neuen Funde ausserhalb des root-cause-gefixten `envelopeToMessage`. Reiner
  Service-/Worker-Fix ohne Migrations-Beruehrung.

## Iteration 9 — fix-rapporte-template-empty-default-lines-crashes — done — 2026-08-11 17:12
- commit: (siehe git log nach diesem Eintrag)
- gebaut: `PostgresRepository.CreateTemplate` und `.UpdateTemplate`
  (`internal/rapporte/postgres_repository.go`) normalisieren jetzt am Funktionsanfang ein
  leeres `defaultLinesJSON` explizit auf `"[]"`, bevor die Spalte `default_lines` (JSONB,
  `NOT NULL DEFAULT '[]'`) geschrieben wird — statt weiterhin ein SQL-NULL zu setzen, das
  den Spalten-Default ueberschreibt und den NOT-NULL-Constraint verletzt. `CreateTemplate`
  setzt `t.DefaultLinesJSON` jetzt direkt beim Struct-Literal (`Valid: true`), `UpdateTemplate`
  uebergibt den normalisierten String direkt statt eines `sql.NullString{Valid: false}`.
  Kein Handler-Aenderungsbedarf: `RapporteGRPCServer.CreateTemplate`/`.UpdateTemplate`
  (`internal/server/rapporte_grpc.go`) reichen `req.GetDefaultLinesJson()` unveraendert
  durch, der Fix sitzt an der einzigen gemeinsamen Stelle (Repository), ueber die beide
  RPCs laufen.
- gebaut (Tests): `TestCreateAndUpdateTemplate_EmptyDefaultLinesJSONDefaultsToEmptyArray`
  (`internal/rapporte/postgres_repository_test.go`) deckt den leeren Request-Pfad end-to-end
  auf Repository-Ebene ab (kein separater Service-Layer fuer Templates vorhanden — der
  Handler ruft `s.repo.CreateTemplate`/`.UpdateTemplate` direkt): `CreateTemplate` mit `""`
  legt eine Vorlage mit `default_lines_json = "[]"` an, `GetTemplate` liest denselben Wert
  zurueck, `UpdateTemplate` mit `""` aktualisiert auf denselben Default statt zu scheitern.
- gate: build ok (`./internal/rapporte/... ./internal/server/... ./cmd/rapporte/...
  ./cmd/gateway/...`) | vet ok | lint ok (0 issues, `./internal/rapporte/...
  ./internal/server/...`) | test ok (0 Skips, `DATABASE_URL` gegen `kmuhub_app`;
  `./internal/rapporte/...` gruen, `./internal/server/` gruen, `./internal/gateway/`
  inkl. TestOpenAPIRouteDrift gruen — keine Route angefasst, nur zur Sicherheit
  mitgelaufen) | migration n.a. (keine Schemaaenderung, reiner Repository-Code-Fix)
  | rls-smoke n.a. (keine Tabelle/Policy angefasst)
- coverage: internal/rapporte 76,0 % -> 76,1 % (selbst gemessen per `go tool cover -func`,
  deckt sich mit `coverage_start:`)
- mutations-probe: Zeilen 743-745 (`if defaultLinesJSON == "" { defaultLinesJSON = "[]" }`
  in `CreateTemplate`) entfernt -> `TestCreateAndUpdateTemplate_EmptyDefaultLinesJSONDefaultsToEmptyArray`
  wurde rot mit "ERROR: invalid input syntax for type json (SQLSTATE 22P02)" (leerer String
  statt normalisiertem `"[]"` landet als ungueltiges JSON in der Spalte, weil das
  Struct-Literal `DefaultLinesJSON` jetzt unbedingt mit `Valid: true` setzt). Zurueckgedreht,
  `git diff --stat internal/rapporte/postgres_repository.go` zeigt wieder nur den
  urspruenglichen Fix-Diff (14 insertions, 10 deletions).
- verify vorgaenger: sauber (Commit `070df174`, Iteration 8 — zwei zusaetzliche
  Struct-Feld-Zuweisungen (`TenantID`) in `send.Service.Send`/`.SaveDraft` und
  `Worker.envelopeToMessage`, kein gRPC-Bypass, kein Stub, kein Proto/Migrations-Drift,
  kein neuer Guard, keine neue Tabelle, keine Wire-Shape- oder Routen-Aenderung)
- neue-units: keine
- offen: keine. Reiner Repository-Fix ohne Migrations-/RLS-Beruehrung; `default_lines_json`
  als "keine Zeilen"-Signal bleibt `"[]"` (nicht NULL) — konsistent mit dem bereits
  bestehenden `GetTemplate`/`ListTemplates`-Lesepfad, der `default_lines::text` direkt in
  `sql.NullString` liest.

## Iteration 10 — fix-crm-list-nil-slice-wire-shape — done — 2026-08-11 17:17
- commit: (siehe git log nach diesem Eintrag)
- gebaut: Fuenf Stellen in zwei Dateien auf das Wire-Shape-Muster "leere Liste = [] statt
  null" umgestellt. `crm_grpc.go`: `ListCustomFields` (`infos := make([]*crmv1.CustomFieldInfo,
  0, len(fields))`), `ListTags` (analog fuer `TagInfo`), `ListContacts` in BEIDEN Zweigen
  (visibility-aware ueber `ListWithVisibility` und der Fallback ueber `.List`) statt jeweils
  eines nackten `var infos []*T`. Root Cause fuer `ListContacts` sass eine Ebene tiefer:
  `contact.Service.enrichWithRelationsBatch` (internal/crm/contact/service.go:400-403) gab bei
  `len(contacts) == 0` explizit `nil, nil` zurueck, bevor der Handler ueberhaupt eine Slice zum
  Iterieren bekam -- jetzt `[]*models.ContactWithRelations{}, nil`. Diese Funktion ist der
  gemeinsame Pfad fuer BEIDE Aufrufer (`Service.List` Zeile 214 und `Service.ListWithVisibility`
  Zeile 642), ein Fix an dieser einen Stelle deckt beide `ListContacts`-Zweige des Handlers ab.
- gebaut (Tests): die drei bestehenden Pin-Tests `TestListCustomFields_EmptyIsNilNotEmptySlice`,
  `TestListTags_EmptyIsNilNotEmptySlice`, `TestListContacts_EmptyIsNilNotEmptySlice`
  (crm_grpc_fields_tags_contacts_test.go) von `require.Nil` auf `require.NotNil` + Laengenpruefung
  umgedreht, Kommentare auf den jetzt korrekten Zustand aktualisiert. Neuer Test
  `TestListContacts_WithVisibility_EmptyIsNilNotEmptySlice` deckt den zweiten, bisher
  ungetesteten Zweig (UserId gesetzt -> ListWithVisibility) ab. In
  `internal/crm/contact/service_test.go` `TestService_List_Empty` um `assert.NotNil(t, contacts)`
  ergaenzt -- die vorherige `assert.Empty` allein unterscheidet nicht zwischen nil und leerer
  Slice und haette die Root-Cause-Aenderung nicht bewiesen.
- gate: build ok (`./internal/server/... ./internal/crm/... ./internal/gateway/...
  ./cmd/gateway/...`) | vet ok | lint ok (0 issues) | test ok (0 Skips, `DATABASE_URL` gegen
  `kmuhub_app`; `./internal/server/` gruen, `./internal/crm/...` gruen bei `-p 1` -- bei
  paralleler Ausfuehrung schlugen mehrere `internal/crm/deal`-DB-Tests mit SQLSTATE 53300
  "remaining connection slots"/"too many clients already" fehl, reproduzierbar unabhaengig
  von diesem Diff (unberuehrtes Paket) und bei `-p 1` durchgehend gruen -- Verbindungspool-
  Erschoepfung des lokalen Postgres, kein Befund dieser Unit; `./internal/gateway/` inkl.
  TestOpenAPIRouteDrift gruen -- keine Route/Spec angefasst, reine Wire-Shape-Aenderung)
  | migration n.a. (keine Schemaaenderung) | rls-smoke n.a. (keine Tabelle/Policy angefasst)
- coverage: internal/server 70,0 % -> 70,0 % (unveraendert, Fix beruehrt bereits durchlaufene
  Zeilen) | internal/crm/contact 80,6 % -> 80,6 % (unveraendert) -- beide selbst gemessen per
  `go tool cover -func`, deckt sich mit `coverage_start:`
- mutations-probe: `return []*models.ContactWithRelations{}, nil` in
  `enrichWithRelationsBatch` (service.go:402) zurueck auf `return nil, nil` gesetzt ->
  `TestService_List_Empty` wurde rot ("Expected value not to be nil"). Zurueckgedreht,
  `git diff --stat internal/crm/contact/service.go` zeigt wieder nur den urspruenglichen
  Ein-Zeilen-Fix (1 insertion, 1 deletion).
- verify vorgaenger: sauber (Commit `152fa892`, Iteration 9 — reiner Repository-Fix in
  `internal/rapporte/postgres_repository.go`, kein gRPC-Bypass, kein Stub, kein
  Proto/Migrations-Drift, kein neuer Guard, keine neue Tabelle, keine Wire-Shape- oder
  Routen-Aenderung; Diff deckt sich mit dem Journal-Eintrag aus Iteration 9)
- neue-units: keine
- offen: keine. Alle fuenf Fundstellen aus der Unit-Beschreibung abgedeckt (drei Handler-Stellen
  plus beide `ListContacts`-Zweige plus der eine Root-Cause-Ort in `enrichWithRelationsBatch`).
  Die connection-slot-Fehler in `internal/crm/deal` bei paralleler Testausfuehrung sind ein
  Lokal-DB-Kapazitaetsthema (Postgres `max_connections` bzw. viele parallele Test-Pools), kein
  Code-Befund -- falls das oefter auftritt, waere ein `max_connections`-Bump auf der lokalen
  Docker-Postgres oder ein niedrigeres `-p` als Default im Gate sinnvoll.

## Iteration 11 — scan-inet-cidr-scanned-into-go-string — done — 2026-08-11 17:25
- commit: (siehe git log nach diesem Eintrag)
- gebaut: Muster D (INET/CIDR-Spalte roh in Go-string statt netip/Cast gescannt) ueber alle
  6 in der Unit genannten Spalten geprueft. Ergebnis: 2 Funde, 4 Spalten bereits korrekt.
  Geprueft (Tabelle.Spalte -> Fundstelle(n) -> Ergebnis):
  - audit_log.ip_address -> internal/security/audit/postgres_repository.go (List, VerifyChain)
    -> bereits gefixt in fix-audit-list-verifychain-ip-address-scan (Lauf 9, It. 1), nicht
    erneut bearbeitet.
  - user_sessions.ip_address -> internal/auth/postgres_repository.go:1574/1586/1614 (3 SELECTs)
    -> korrekt (`COALESCE(host(ip_address), '')`); zusaetzlich internal/security/gdpr/export.go:172
    (DSAR-Export) -> korrekt (`CASE WHEN ip_address IS NOT NULL THEN host(ip_address) ELSE NULL END`).
  - ip_access_rules.ip_cidr -> internal/server/security_grpc.go:711 (ListIPRules) -> FUND. Roh
    gescannt in models.IPAccessRule.IPCIDR (string, kein Cast). Reproduziert per DB-gestuetztem
    Wegwerf-Test (nicht committet, nach Reproduktion geloescht): Zeile mit ip_cidr='10.0.0.0/8'
    eingefuegt, exakt dieselbe Query wie ListIPRules abgesetzt -> "cannot scan cidr (OID 650) in
    binary format into *string". Neue Unit: fix-security-ip-access-rules-cidr-scan.
  - guest_sessions.ip_address -> internal/chat/guest/postgres_repository.go:44/67/87
    (GetSessionByTokenHash, GetSessionByID, ListSessionsByChannel) -> FUND, alle drei Stellen.
    Roh gescannt in GuestSession.IPAddress (*string, kein Cast). Bei NULL scannt pgx anstandslos
    (deshalb in bestehenden Tests unsichtbar -- seedGuestSession dort setzt ip_address nie).
    Reproduziert per DB-gestuetztem Wegwerf-Test (nicht committet, nach Reproduktion geloescht):
    Session mit ip_address='203.0.113.5' eingefuegt, GetSessionByID aufgerufen -> "cannot scan
    inet (OID 869) in binary format into **string". Neue Unit:
    fix-chat-guest-sessions-ip-address-scan.
  - consent_records.ip_address -> internal/crm/consent/postgres_repository.go:37/64/196 (3
    SELECTs) -> korrekt (`ip_address::text`).
  - form_submissions.ip_address -> internal/formulare/postgres_repository.go:259/300/362 (3
    SELECTs) -> korrekt (`ip_address::text`).
  Zusaetzlich per Grep geprueft, dass keine weiteren Lesestellen fuer ip_cidr/guest_sessions.
  ip_address existieren (internal/gateway/route_security.go traegt IPCIDR nur als Wire-Typ,
  keine SQL-Query; internal/chat/message/postgres_repository.go liest nur display_name aus
  guest_sessions, keine ip_address-Spalte).
- gate: n.a. -- reine Backlog-Recherche, kein Produktionscode geaendert. Die beiden
  Wegwerf-Reproduktionstests (internal/server/zzscan_repro_test.go,
  internal/chat/guest/zzscan_repro_test.go) wurden nach dem Verifizieren wieder geloescht;
  `git status --short` zeigt nur BACKLOG.yml/JOURNAL.md.
- coverage: n.a. (Scan-Unit, kein Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein Verhalten geaendert)
- verify vorgaenger: sauber (Commit `7d4b6571`, Iteration 10 — reiner Wire-Shape-Fix in
  crm_grpc.go + contact/service.go, kein gRPC-Bypass, kein Stub, kein Proto/Migrations-Drift,
  kein neuer Guard, keine neue Tabelle, keine Routenaenderung; Diff deckt sich mit dem
  Journal-Eintrag aus Iteration 10)
- neue-units: fix-security-ip-access-rules-cidr-scan, fix-chat-guest-sessions-ip-address-scan
- offen: keine. Alle 6 Spalten aus der Unit-Beschreibung abschliessend geprueft, beide Funde
  als vollstaendige Fix-Units mit reproduziertem Fehler und Pin-Test-Vorschlag angelegt.

## Iteration 12 — scan-on-conflict-target-vs-real-index-a — done — 2026-08-11 17:45
- commit: (siehe git log nach diesem Eintrag)
- gebaut: Muster A (ON-CONFLICT-Ziel vs. tatsaechlicher Unique-Index) ueber die Teil-A-Flaeche
  (`internal/{auth,security,notification,chat,document,email,inbox,work,crm}`) geprueft.
  Ergebnis: 0 neue Funde. `internal/security` und `internal/inbox` haben ueberhaupt kein
  `ON CONFLICT` (per Grep bestaetigt). Alle 28 explizit-getargeteten `ON CONFLICT`-Klauseln
  (Spaltenliste angegeben) in den restlichen sieben Paketen matchen exakt einen bestehenden
  Unique-Index/PK -- gegen die laufende lokale DB per `pg_index`/`pg_get_indexdef` verifiziert
  (Kopf 310), nicht nur gegen Migrationen (Drift-Gefahr, siehe Vorlage). Geprueft:
  - notification/preference/postgres_repository.go: 3 Klauseln (event_type_key- und
    module_id-Partial-Index, notification_quiet_hours) -- alle drei matchen; quiet_hours und
    module_default wurden bereits in Iteration 4 (Migration 000312) korrigiert, event_type_key
    war seit Migration 000305 (Lauf 8) bereits korrekt.
  - work/calendar/postgres_repository.go: user_calendar_preferences(user_id) matcht PK.
  - document/wopi/lock.go: wopi_locks(file_id) matcht PK.
  - crm/contact/postgres_repository.go: contact_custom_field_values(contact_id, field_id) x2
    matcht PK.
  - crm/company/postgres_repository.go: company_custom_field_values(company_id, field_id) x2
    matcht PK.
  - crm/deal/postgres_repository.go: deal_custom_field_values(deal_id, field_id) matcht PK.
  - crm/activity/postgres_repository.go: activity_custom_field_values(activity_id, field_id)
    matcht PK.
  - chat/file/postgres_repository.go: storage_quotas(tenant_id) matcht idx_storage_quotas_tenant.
  - chat/bookmark/postgres_repository.go: message_bookmarks(user_id, message_id) matcht PK.
  - notification/integration/postgres_repository.go:
    integration_account_links(platform, external_user_id) matcht
    integration_account_links_platform_external_user_id_key.
  - work/meeting/postgres_repository.go: meeting_attendees(meeting_id, user_id) matcht PK;
    meeting_cohosts(meeting_id, user_id) matcht meeting_cohosts_unique;
    meeting_breakout_assignments(meeting_id, user_id) matcht meeting_breakout_assignments_unique;
    meeting_notes(meeting_id, author_id) WHERE is_private = $5 (parametrisiertes Praedikat auf
    einem partiellen Index, migration 000309) zusaetzlich per echtem DB-Testlauf
    (TestNotes_SeriesIsolationAndSaveNotesUpsert) bestaetigt, nicht nur per Indexvergleich --
    Postgres kann den Bind-Parameter im Arbiter-Praedikat bei jedem Exec neu gegen den
    passenden der beiden partiellen Indizes (is_private = true / false) aufloesen, funktioniert
    nachweislich.
  - work/task/postgres_repository.go: task_custom_field_values(task_id, field_id) matcht PK.
  - work/recording/postgres_repository.go: recording_consents(recording_id, user_id) matcht PK.
  - work/presence/postgres_config_repository.go: presence_config(tenant_id) matcht
    idx_presence_config_tenant.
  - document/tag/postgres_repository.go: document_file_tags(file_id, tag_id) matcht PK.
  - auth/postgres_repository.go: two_factor_policy(tenant_id, role_name) matcht
    idx_two_factor_policy_tenant_role.
  - work/project/postgres_repository.go: project_members(project_id, user_id) matcht PK;
    user_project_preferences(user_id, project_id) matcht PK.
  - work/video/postgres_repository.go: call_participants(call_id, user_id) matcht PK.
  - email/contactlink/repository.go: email_contact_links(message_id, contact_id) matcht
    email_contact_links_message_id_contact_id_key.
  - work/holiday/postgres_repository.go: public_holidays(date, country_code, name) matcht
    public_holidays_date_country_code_name_key (System-Global-Tabelle, kein tenant_id).
  Zusaetzlich 12 `ON CONFLICT DO NOTHING`-Klauseln OHNE Spaltenliste geprueft (contact_tags,
  company_tags, activity_tags, deal_tags je bis zu 2x, calendars, message_mentions,
  work/label task_labels, work/reaction message_reactions, auth user_roles x2) -- diese sind
  per Definition sicher (kein Arbiter-Ziel, greift bei jeder Constraint-Verletzung) und damit
  kein Muster-A-Kandidat.
- gate: n.a. -- reine Backlog-Recherche, kein Produktionscode geaendert. Ein Wegwerf-Insert
  in `users` (Testzeile, nicht Teil einer Test-Suite) wurde waehrend der Recherche versehentlich
  angelegt und sofort wieder geloescht (`DELETE FROM users WHERE email =
  'scan-repro@test.local'`), `git status --short` zeigt nur BACKLOG.yml/JOURNAL.md.
- coverage: n.a. (Scan-Unit, kein Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein Verhalten geaendert)
- verify vorgaenger: sauber (Commit `b07a6b54`, Iteration 11 — reine Backlog/Journal-Aenderung,
  kein Produktionscode; kein gRPC-Bypass, kein Stub, kein Proto/Migrations-Drift, kein neuer
  Guard, keine neue Tabelle, keine Wire-Shape- oder Routenaenderung moeglich, da kein Code
  angefasst wurde)
- neue-units: keine
- offen: keine. Teil B (`scan-on-conflict-target-vs-real-index-b`, Module-Services) ist die
  naechste Haelfte derselben Musterflaeche und noch offen.

## Iteration 13 — scan-on-conflict-target-vs-real-index-b — done — 2026-08-11 17:36
- commit: (siehe git log nach diesem Eintrag)
- gebaut: Muster A (ON-CONFLICT-Ziel vs. tatsaechlicher Unique-Index), Teil B — restliche
  Haelfte der Flaeche (`internal/{biz,hr,inventar,einkauf,produktion,vertraege,rapporte,
  schichten,vermietung,fuhrpark,helpdesk,wiki,formulare,berichte,dialer,automation,plugin,
  settings,gateway,caldav,idempotency}`) per Subagent geprueft. Ergebnis: 0 neue Funde.
  41 explizit-getargetete ON-CONFLICT-Klauseln (Spaltenliste oder ON CONSTRAINT) in 29
  Nicht-Test-Dateien matchen alle exakt einen real existierenden Unique-Index/PK/Constraint
  gegen die laufende lokale DB (nicht nur Migrationen). Geprueft:
  - schichten/postgres_repository.go:490 shift_swap_requests(idempotency_key)
  - biz/datev/postgres_upload_repo.go:79 datev_upload_configs(config_id)
  - settings/postgres_repository.go: tenant_module_leads(tenant_id,user_id,module_id);
    user_module_grants(tenant_id,user_id,module_id); tenant_settings(tenant_id,module_id,key);
    user_settings(tenant_id,user_id,module_id,key); tenant_module_activations(tenant_id,
    module_id); customization_value_sets(tenant_id,set_key)
  - plugin/repository/kv_store.go:51 plugin_kv_store(installation_id,key)
  - inventar/postgres_repository.go: inventur_counts(session_id,item_id);
    picking_list_items(picking_list_id,item_id)
  - helpdesk/postgres_repository.go: helpdesk_ticket_counters(tenant_id);
    ticket_csat_responses(tenant_id,ticket_id) x2; helpdesk_business_hours(tenant_id)
  - biz/lexware/postgres_repository.go: lexware_sync_configs(config_id);
    lexware_entity_mappings(config_id,entity_type,kmuhub_id);
    lexware_field_mappings(config_id,entity_type);
    lexware_webhook_subscriptions(config_id,event_type)
  - biz/hr/leave/postgres_repository.go: hr_leave_balances(tenant_id,employee_id,year);
    hr_company_settings(tenant_id)
  - biz/bexio/postgres_repository.go: bexio_sync_configs(config_id);
    bexio_entity_mappings(config_id,entity_type,kmuhub_id);
    bexio_field_mappings(config_id,entity_type)
  - automation/workflow/postgres_repository.go:
    automation_time_trigger_fires(automation_id,entity_key); automation_templates(id)
  - caldav/push_subscription.go:65
    caldav_push_subscriptions(user_id,collection_type,collection_id,push_url)
  - caldav/app_password.go:183 caldav_settings(key)
  - caldav/sync_token.go:80 caldav_sync_versions(collection_type,collection_id); :46
    arbiterloses DO NOTHING (sicher, nicht Muster-A-Kandidat)
  - dialer/postgres_repository.go:177 ON CONSTRAINT uq_campaign_contact ->
    dialer_campaign_contacts(campaign_id,contact_id), per EXPLAIN real-verifiziert
  - biz/recurring/postgres_repository.go:172
    finance_recurring_runs(tenant_id,recurring_id,period_date)
  - biz/hr/timetracking/postgres_extended_repository.go:256
    hr_week_approvals(tenant_id,employee_id,week_start)
  - biz/quote/postgres_repository.go:500 company_settings(tenant_id)
  - gateway/dashboard_repository.go:67 dashboard_defaults(tenant_id,role); :103 ON CONSTRAINT
    uq_user_dashboard_layouts_user_id -> user_dashboard_layouts(user_id), per EXPLAIN
    real-verifiziert
  - biz/payment/postgres_repository.go:56
    finance_payments(tenant_id,idempotency_key) WHERE idempotency_key IS NOT NULL, per EXPLAIN
    real-verifiziert (partieller Index)
  - biz/invoice/postgres_repository.go:127
    finance_invoices(tenant_id,source,external_id) WHERE external_id IS NOT NULL, per EXPLAIN
    real-verifiziert (partieller Index)
  - berichte/postgres_repository.go:187 report_cache(definition_id,params_hash)
  - biz/datev/postgres_config_repo.go, biz/bexio/postgres_config_repo.go,
    biz/lexware/postgres_config_repo.go: alle drei integration_configs(platform,tenant_id)
  - biz/dunning/postgres_repository.go:313 finance_dunning_config(tenant_id)
  - idempotency/postgres_repository.go:93 idempotency_keys(tenant_id,key) (= PK)
  - plugin/repository/permission.go:27 plugin_permissions(installation_id,permission)
  - plugin/repository/industry_template.go:29 industry_templates(slug)
  - helpdesk/repository.go:151 reiner Kommentar, kein Code-ON-CONFLICT (nicht gezaehlt)
  Zusaetzlich per Backlog-Scope angefordert geprueft: `internal/hr` (top-level, getrennt von
  `internal/biz/hr`) existiert nicht; `internal/video` (top-level, getrennt von
  `internal/work/video`) existiert nicht; `einkauf`, `produktion`, `vertraege`, `rapporte`,
  `vermietung`, `fuhrpark`, `wiki`, `formulare` existieren alle, enthalten aber KEIN einziges
  `ON CONFLICT` (per Grep bestaetigt) -- nichts zu pruefen. Damit sind Teil A + Teil B
  gemeinsam die vollstaendige Musterflaeche aus scan-on-conflict-target-vs-real-index-a/-b.
- gate: n.a. -- reine Backlog-Recherche, kein Produktionscode geaendert. Vier riskante Muster
  (zwei partielle Indizes, zwei ON-CONSTRAINT-Ziele) zusaetzlich per EXPLAIN gegen die echte
  Query geplant (kein ANALYZE, keine Ausfuehrung, keine Datenaenderung). `git status --short`
  zeigt vor dem Commit nur BACKLOG.yml/JOURNAL.md.
- coverage: n.a. (Scan-Unit, kein Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein Verhalten geaendert)
- verify vorgaenger: sauber (Commit `2b99ac09`, Iteration 12 — reine Backlog/Journal-Aenderung,
  kein Produktionscode; kein gRPC-Bypass, kein Stub, kein Proto/Migrations-Drift, kein neuer
  Guard, keine neue Tabelle, keine Wire-Shape- oder Routenaenderung moeglich, da kein Code
  angefasst wurde)
- neue-units: keine
- offen: keine. Muster A (ON-CONFLICT-Ziel vs. Index) ist mit Teil A + Teil B vollstaendig
  abgearbeitet, 0 Funde ueber die gesamte Flaeche.

## Iteration 15 — scan-phantom-columns-crm-biz-work-email — done — 2026-08-11 17:46
- commit: (siehe git log nach diesem Eintrag)
- gebaut: Muster A2 (SQL referenziert Spalte/Tabelle/Alias, die es nicht gibt), Teil 1 von 3
  — Repositories unter `internal/crm`, `internal/biz` (inkl. aller Unterpakete: bexio,
  lexware, datev, hr/*, invoice, creditnote, quote, payment, expense, recurring, banking,
  einvoice, dunning, gobdarchive, dashboard), `internal/work` (inkl. video) und `internal/email`
  per drei parallelen Subagenten geprueft (max. 3 gleichzeitig, wie vorgeschrieben). Vorgehen
  pro Agent: SQL-tragende Go-Dateien per Grep auf SELECT/INSERT/UPDATE/RETURNING identifiziert
  (nicht nur `*postgres_repository*.go`), Aliase aus JOIN-Klauseln aufgeloest, jede referenzierte
  Spalte gegen `information_schema.columns` der laufenden lokalen DB (Migrationskopf 312, nicht
  gegen die Migrationsdateien) geprueft.
  ERGEBNIS: 0 Funde ueber alle vier Pakete. 68 Produktions-Repository-Dateien vollstaendig
  geprueft:
  - crm (14): report, contact/postgres_repository, contact/postgres_lead, company, advisoryprotocol,
    deal, activity, consent/postgres_repository, consent/assert_repo, search, tag, customfield,
    savedfilter, pipelinestage
  - email (8): attachment, message, template, rule, label, account, contactlink, signature
  - biz (29): dunning, gobdarchive, invoice/postgres_repository, quote, creditnote, payment,
    expense, recurring, banking/postgres_repository_accounts, banking/postgres_repository,
    einvoice, invoice/postgres_open_items, invoice/postgres_document_chains,
    invoice/postgres_transactions, bexio/postgres_repository, bexio/postgres_config_repo,
    lexware/postgres_repository, lexware/postgres_config_repo, datev/postgres_upload_repo,
    datev/postgres_config_repo, dashboard, hr/leave, hr/absence, hr/employee, hr/changerequest,
    hr/timetracking/postgres_repository, hr/timetracking/postgres_extended_repository,
    invoice/service.go (kein Roh-SQL), invoice/repository.go (kein Roh-SQL)
  - work (17): calendar/postgres_repository, calendar/booking_postgres_repository, event,
    meeting, task, timeentry, recording, project, label, customfield, video, status, comment,
    resource, reaction, holiday, presence/postgres_config_repository
  Der Vorlage-Bug (`u.display_name` auf `users`, existiert nur in `document/virtual`) tritt in
  keinem der vier gescannten Pakete auf -- alle `users`-JOINs verwenden durchgaengig korrekt
  `first_name`/`last_name` (in `hr/*` teils bereits mit explizitem Regressionsschutz-Kommentar
  auf den historischen Incident, z. B. `hr/absence/postgres_repository_db_test.go:130`). Ein
  auffaelliger Fund (`m.references` als unquotierter Alias auf ein reserviertes Wort in
  `email/contactlink/repository.go:80`) wurde gegen die DB verifiziert und laeuft fehlerfrei --
  kein Bug, da qualifiziert referenziert.
- gate: n.a. -- reine Backlog-Recherche, kein Produktionscode geaendert. `git status --short`
  zeigt vor dem Commit nur BACKLOG.yml/JOURNAL.md.
- coverage: n.a. (Scan-Unit, kein Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein Verhalten geaendert)
- verify vorgaenger: Iteration 14 hat KEINEN Commit hinterlassen -- lediglich den
  `status: in_progress`-Flag dieser Unit unkommitiert in BACKLOG.yml gesetzt (git diff bestaetigt:
  ausschliesslich diese eine Zeile geaendert, kein Code angefasst) und ist ohne Journal-Eintrag
  gestorben. Nichts zu verifizieren, da keine inhaltliche Arbeit vorlag -- diese Iteration hat die
  Unit vollstaendig selbst bearbeitet.
- neue-units: keine
- offen: Teil 2 von 3 (`scan-phantom-columns-platform-services`: auth, security, document, chat,
  calendar, notification, inbox, hr-top-level) und Teil 3 von 3
  (`scan-phantom-columns-module-services`) stehen noch aus und sind bereits als todo im Backlog.

## Iteration 16 — scan-phantom-columns-platform-services — done — 2026-08-11 17:52
- commit: (siehe git log nach diesem Eintrag)
- gebaut: Muster A2 (SQL referenziert Spalte/Tabelle/Alias, die es nicht gibt), Teil 2 von 3 —
  Repositories unter `internal/auth`, `internal/security` (inkl. audit, gdpr, password, vault,
  vendoraccess), `internal/document`, `internal/chat`, `internal/notification` (inkl.
  preference, integration, notification), `internal/inbox` (inkl. thread, message, routing,
  team, adapter) per drei parallelen Subagenten geprueft (max. 3 gleichzeitig). `internal/hr`
  und `internal/calendar` existieren als Top-Level-Pakete NICHT (verifiziert per `ls
  backend/internal/`) -- die tatsaechlichen Pakete `internal/biz/hr` und `internal/work/calendar`
  wurden bereits in Iteration 15 (Teil 1) gescannt, hier also korrekt ausgelassen statt doppelt
  geprueft.
  Vorgehen pro Agent: SQL-tragende Go-Dateien per Grep auf SELECT/INSERT/UPDATE/RETURNING
  identifiziert (nicht nur `*postgres_repository*.go`), Aliase aus JOIN-Klauseln aufgeloest,
  jede referenzierte Spalte gegen `information_schema.columns` der laufenden lokalen DB
  (Migrationskopf 312) geprueft.
  ERGEBNIS: 0 Funde ueber alle sechs Pakete. 150 Nicht-Test-Go-Dateien vollstaendig geprueft
  (38 auth+security, 60 document+chat, 52 notification+inbox), davon 30 mit tatsaechlichem
  Roh-SQL (Rest sind Interfaces/Services/Adapter ohne direkten DB-Zugriff):
  - auth (1 SQL-Datei): postgres_repository.go
  - security (8): audit/postgres_repository.go, password/postgres_repository.go,
    vault/postgres_repository.go, vendoraccess/postgres_repository.go,
    gdpr/postgres_repository.go, gdpr/erasure.go, gdpr/dsar_search.go, gdpr/export.go
  - document (7): virtual, file, search, folder, tag, share, wopi/lock.go
    (postgres_repository.go je Paket, wopi als lock.go)
  - chat (6 mit echtem SQL): channel, message, file, search, bookmark, guest
    (postgres_repository.go je Paket)
  - notification (3): preference, integration, notification (postgres_repository.go je Paket)
  - inbox (5): thread, message, routing, team (postgres_repository.go je Paket),
    adapter/guest_adapter.go
  Der Vorlage-Bug (`u.display_name` auf `users`) tritt in keinem der sechs gescannten Pakete
  auf. Bestaetigt korrekt (Gegenbeispiel, kein Fund): `chat/message` und `inbox/adapter`
  nutzen `gs.display_name` aus `guest_sessions` -- diese Tabelle TRAEGT tatsaechlich eine
  `display_name`-Spalte, im Unterschied zu `users`. `auth`, `security/vendoraccess` und
  `security/gdpr` per zusaetzlichem `EXPLAIN` gegen komplexere Queries (CTEs, Self-Joins,
  korrelierte Subqueries) verifiziert -- alle planen fehlerfrei.
- gate: n.a. -- reine Backlog-Recherche, kein Produktionscode geaendert. `git status --short`
  zeigt vor dem Commit nur BACKLOG.yml/JOURNAL.md.
- coverage: n.a. (Scan-Unit, kein Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein Verhalten geaendert)
- verify vorgaenger: sauber (Commit `49c7dc42`, Iteration 15 — reine Backlog/Journal-Aenderung,
  kein Produktionscode; kein gRPC-Bypass, kein Stub, kein Proto/Migrations-Drift, kein neuer
  Guard, keine neue Tabelle, keine Wire-Shape- oder Routenaenderung moeglich, da kein Code
  angefasst wurde)
- neue-units: keine
- offen: Teil 3 von 3 (`scan-phantom-columns-module-services`) steht noch aus und ist bereits
  als todo im Backlog.

## Iteration 17 — scan-phantom-columns-module-services — done — 2026-08-11 17:57
- commit: (siehe git log nach diesem Eintrag)
- gebaut: Muster A2 (SQL referenziert Spalte/Tabelle/Alias, die es nicht gibt), Teil 3 von 3 —
  Repositories unter `internal/{plugin,inventar,einkauf,produktion,caldav,vertraege,rapporte,
  schichten,vermietung,fuhrpark,helpdesk,wiki,formulare,berichte,dialer,automation,settings}`
  per drei parallelen Subagenten geprueft (max. 3 gleichzeitig). `internal/video` existiert
  nicht als Top-Level-Paket -- die tatsaechliche Video-Funktionalitaet liegt unter
  `internal/work/video` und wurde bereits in Iteration 15 (Teil 1, Paketliste `work`) geprueft,
  hier also korrekt ausgelassen statt doppelt bearbeitet.
  Vorgehen pro Agent: alle Nicht-Test-Go-Dateien mit echtem Roh-SQL (SELECT/INSERT/UPDATE/
  RETURNING) identifiziert, Aliase aus JOIN-Klauseln aufgeloest, jede referenzierte Spalte/
  Tabelle gegen `information_schema.columns` bzw. `\d` der laufenden lokalen DB (Container
  `docker-postgres-1`, Migrationskopf 312) geprueft, bei Unsicherheit zusaetzlich per `EXPLAIN`
  gegen die DB verifiziert.
  Insgesamt 37 SQL-tragende Nicht-Test-Dateien geprueft:
  - Gruppe 1 (13 Dateien): plugin/repository/{execution_log,industry_template,installation,
    kv_store,manifest,permission,validation_rule,workflow_rule}.go (8),
    inventar/postgres_repository.go (1), einkauf/{postgres_repository,
    postgres_repository_extended}.go (2), produktion/{postgres_repository,
    postgres_repository_ext}.go (2) — 0 Funde.
  - Gruppe 2 (12 Dateien): caldav/{push_subscription,app_password,caldav_backend,sync_token,
    postgres_app_password,carddav_backend,user_preferences}.go (7),
    vertraege/postgres_repository.go (1), rapporte/{postgres_repository,service}.go (2, service.go
    ohne Roh-SQL, trotzdem geprueft), schichten/postgres_repository.go (1),
    vermietung/postgres_repository.go (1) — 1 FUND (siehe unten).
  - Gruppe 3 (12 Dateien): fuhrpark/postgres_repository.go (1),
    helpdesk/postgres_repository.go (1), wiki/postgres_repository.go (1),
    formulare/postgres_repository.go (1), berichte/{postgres_repository,downstream/
    kpi_postgres}.go (2), dialer/postgres_repository.go (1),
    automation/{trigger/due_postgres,workflow/postgres_repository}.go (2),
    settings/postgres_repository.go (1) — 0 Funde (erste Grep-Treffer auf weitere Dateien in
    diesen Paketen waren Fehlalarme: Go-`select{}`-Statements bzw. das Wort "returning" in
    Kommentaren, verifiziert und ausgeschlossen).
  FUND: `backend/internal/caldav/carddav_backend.go:400-407`,
  `checkCompanyContactPermission` selektiert `SELECT role FROM users WHERE id = $1` roh in
  eine `role string`-Variable. `users` hat keine `role`-Spalte (verifiziert per `\d users`
  gegen Kopf 312 und per `EXPLAIN` -> `ERROR: column "role" does not exist`, SQLSTATE 42703).
  Rollen liegen ausschliesslich in `user_roles`/`roles` (RBAC-Modell, vgl.
  `internal/auth/postgres_repository.go:191 GetUserRoles`). Wirkung: der `err != nil`-Zweig
  faengt den SQL-Fehler ab und liefert HTTP 403 -- fail-closed, aber das Feature (CardDAV-Schreib-
  zugriff auf das geteilte `company`-Adressbuch) ist fuer JEDEN Nutzer, auch echte Admins/
  Manager, vollstaendig unbenutzbar. Als neue Fix-Unit angelegt:
  `fix-caldav-carddav-company-permission-role-column`.
- gate: n.a. -- reine Backlog-Recherche, kein Produktionscode geaendert. `git status --short`
  zeigt vor dem Commit nur BACKLOG.yml/JOURNAL.md.
- coverage: n.a. (Scan-Unit, kein Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein Verhalten geaendert)
- verify vorgaenger: sauber (Commit `2c2c0a01`, Iteration 16 — reine Backlog/Journal-Aenderung,
  kein Produktionscode; kein gRPC-Bypass, kein Stub, kein Proto/Migrations-Drift, kein neuer
  Guard, keine neue Tabelle, keine Wire-Shape- oder Routenaenderung moeglich, da kein Code
  angefasst wurde)
- neue-units: fix-caldav-carddav-company-permission-role-column
- offen: Muster A2 (phantom columns) ist damit ueber alle drei Teile abgeschlossen. Block B hat
  noch offene Scan-Units (scan-insert-missing-tenant-id-a/b/c, scan-nil-slice-wire-shape-large-
  services, weitere) fuer die naechste Iteration.

## Iteration 18 — scan-insert-missing-tenant-id-a — done — 2026-08-11 18:26
- commit: (siehe git log nach diesem Eintrag)
- gebaut: Muster B (INSERT setzt tenant_id nicht, obwohl NOT NULL ohne Default), Teil A von 3 —
  Repositories unter `internal/biz` (inkl. banking, bexio, creditnote, datev, dunning, einvoice,
  expense, gobdarchive, hr/{changerequest,employee,leave,timetracking}, invoice, lexware,
  payment, quote, recurring), `internal/crm` (activity, advisoryprotocol, company, consent,
  contact, customfield, deal, pipelinestage, savedfilter, tag) und `internal/work` (calendar,
  comment, customfield, event, holiday, label, meeting, presence, project, reaction, recording,
  resource, status, task, timeentry, video) per drei parallelen Subagenten geprueft (max. 3
  gleichzeitig).
  Vorgehen pro Agent: alle `INSERT INTO`-Statements in Nicht-Test-Go-Dateien gefunden, Zieltabelle
  bestimmt, gegen die laufende lokale DB (Container `docker-postgres-1`, Kopf 312) per `\d
  <tabelle>` geprueft ob `tenant_id NOT NULL` ohne Default ist, System-Global-Liste (ADR-006:
  schema_migrations, caldav_settings, industry_templates, permissions, automation_templates,
  event_types, public_holidays) ausgenommen, dann Spaltenliste + Werteherkunft (Struct-Feld aus
  Kontext vs. hartkodiert/uuid.Nil/fehlend) verifiziert.
  Insgesamt 116 INSERT-Statements in 48 Nicht-Test-Dateien geprueft:
  - biz (21 Dateien, 46 INSERTs, Referenzdatei postgres_upload_repo.go bereits gefixt und nicht
    erneut gezaehlt) — 0 Funde, alle tenant_id aus *.TenantID-Feld bzw. tenantID-Parameter.
    Einzige Ausnahme: `hr_employee_profiles` hat einen Zero-UUID-Default (kein NOT-NULL-Problem,
    kein Fund).
  - crm (10 Dateien, 23 INSERTs) — 0 Funde. *_custom_field_values-Junction-Tabellen haben gar
    keine tenant_id-Spalte (RLS via EXISTS-Policy auf Elterntabelle), alle uebrigen setzen
    tenant_id korrekt aus Struct-Feld oder Subquery gegen die tenant-tragende Elterntabelle.
  - work (17 Dateien, 47 INSERTs) — 1 FUND (siehe unten), Rest sauber (Feld, tenantID-Parameter
    oder Subquery gegen Elterntabelle; `task_labels` hat wie die CRM-Junction-Tabellen keine
    eigene tenant_id-Spalte).
  FUND: `backend/internal/work/recording/postgres_repository.go:154 SetConsent` inserted in
  `recording_consents` (tenant_id NOT NULL ohne Default, FORCE RLS, verifiziert per `\d
  recording_consents`) ohne tenant_id in der Spaltenliste. `RecordingConsent`-Modell hat kein
  TenantID-Feld, `Service.SetConsent` bekommt keine tenantID uebergeben — der Aufrufer
  (`VideoGRPCServer.SetRecordingConsent`) hat sie nicht zur Hand. Jeder Aufruf schlaegt am
  NOT-NULL-Constraint fehl: DSGVO-relevante Recording-Consent-Zustimmung laesst sich nie
  speichern. Selbst gegen die DB verifiziert (`\d recording_consents`, Kopf 312) und Modell/
  Service/Caller-Kette gegengelesen, bevor die Unit angelegt wurde. Als neue Fix-Unit angelegt:
  `fix-work-recording-consent-missing-tenant-id`.
- gate: n.a. -- reine Backlog-Recherche, kein Produktionscode geaendert. `git status --short`
  zeigt vor dem Commit nur BACKLOG.yml/JOURNAL.md.
- coverage: n.a. (Scan-Unit, kein Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein Verhalten geaendert)
- verify vorgaenger: sauber (Commit `2d6017a2`, Iteration 17 — reine Backlog/Journal-Aenderung,
  kein Produktionscode; kein gRPC-Bypass, kein Stub, kein Proto/Migrations-Drift, kein neuer
  Guard, keine neue Tabelle, keine Wire-Shape- oder Routenaenderung moeglich, da kein Code
  angefasst wurde)
- neue-units: fix-work-recording-consent-missing-tenant-id
- offen: Teil B (`scan-insert-missing-tenant-id-b`, auth/security/document/chat/calendar/email/
  notification/inbox/hr) und Teil C (`scan-insert-missing-tenant-id-c`, Modul-Services) stehen
  noch aus und sind bereits als todo im Backlog.

## Iteration 19 — scan-insert-missing-tenant-id-b — done — 2026-08-11 18:09
- commit: (siehe git log nach diesem Eintrag)
- gebaut: Muster B (INSERT setzt tenant_id nicht, obwohl NOT NULL ohne Default), Teil B von 3 —
  Pakete auth, security/{audit,gdpr,password,vendoraccess,vault}, document/{file,folder,share,
  tag,wopi}, chat/{bookmark,channel,file,guest,message}, calendar-Domaene (work/{calendar,event,
  meeting}, ergaenzend auch caldav), email/{account,attachment,contactlink,label,message,rule,
  signature,template} (ausser der bereits gefixten send.Send/SaveDraft-Ausnahme), notification/
  {preference,integration,notification}, inbox/{thread,team,routing,message} per drei parallelen
  Explore-Subagenten geprueft (max. 3 gleichzeitig).
  Scope-Klarstellung vorab: `internal/hr` existiert nicht eigenstaendig — das HR-Modul liegt unter
  `internal/biz/hr/{leave,employee,changerequest,timetracking,absence,compliance}` und wurde dort
  geprueft (leave/employee/changerequest/timetracking waren durch Teil A/Iteration 18 bereits
  abgedeckt und wurden hier zur Bestaetigung erneut gegengelesen — 0 Abweichung; absence und
  compliance sind neu abgedeckt: absence hat kein INSERT, compliance hat kein Repository mit
  INSERT). `internal/calendar` existiert ebenfalls nicht eigenstaendig — als fachliches Aequivalent
  wurden `internal/work/{calendar,event,meeting}` sowie ergaenzend `internal/caldav` geprueft.
  Vorgehen pro Agent: alle `INSERT INTO`-Statements in Nicht-Test-Go-Dateien gefunden, Zieltabelle
  bestimmt, gegen die laufende lokale DB (Container `docker-postgres-1`, Kopf 312) per `\d
  <tabelle>` geprueft ob `tenant_id NOT NULL` ohne Default ist, System-Global-Liste (ADR-006:
  schema_migrations, caldav_settings, industry_templates, permissions, automation_templates,
  event_types, public_holidays; zusaetzlich `roles`/`role_permissions` mit bewusst NULLable
  tenant_id) ausgenommen, dann Spaltenliste + Werteherkunft bis zum gRPC-Handler/Service
  zurueckverfolgt (Struct-Feld/Parameter aus `middleware.GetTenantID(ctx)`, tenant-tragendem
  Event-Payload, oder sicherer SQL-Subquery-Ableitung aus einer bereits tenant-validierten
  Elternzeile — vs. hartkodiert/uuid.Nil/fehlend).
  Insgesamt 107 INSERT-Statements in 33 Nicht-Test-Dateien geprueft, 0 FUNDE:
  - auth/security/hr (10 Dateien, 37 INSERTs) — 0 Funde. Alle tenant_id aus *.TenantID-Feld,
    tenantID-Parameter oder `SELECT tenant_id FROM users WHERE id = ...`-Subquery.
    `role_permissions` hat gar keine tenant_id-Spalte (Isolation ueber Join auf roles.tenant_id).
    Drei Tabellen (users, audit_log, hr_employee_profiles) haben zusaetzlich einen Spalten-
    Default und faellen damit per Definition ohnehin nicht unter Muster B — der Code setzt den
    Wert aber in allen drei Faellen ohnehin explizit.
  - document/chat/calendar-Domaene (13 Dateien + 3 caldav-Dateien, 49 INSERTs) — 0 Funde. Vier
    Tabellen (document_files, channels, messages, calendar_events) haben einen fixen System-
    Tenant-Default (Sentinel-UUID ...0001) und faellen damit nicht unter Muster B; der Code setzt
    tenant_id trotzdem explizit aus dem Modell. Alle Subquery-Ableitungen (z. B. calendar_members
    aus calendars, event_attendees aus calendar_events, meeting_attendees aus meetings) stammen
    von Elternzeilen, die der Service vorher tenant-scoped aufgeloest hat.
  - email/notification/inbox (15 Dateien, 26 INSERTs) — 0 Funde (ausser der bereits gefixten
    send.Send/SaveDraft-Ausnahme, nicht erneut geprueft). email_messages, notifications und
    inbox_messages haben denselben Sentinel-Default und faellen damit nicht unter Muster B.
    `inbox/adapter/*.go` bauen InboxMessage ohne TenantID, sind aber laut Code-Kommentar ein
    toter Pfad ohne aktiven Aufrufer (kein Fund). `integration_configs.CreateConfig` verlaesst
    sich (anders als die Schwester-Methoden in derselben Datei) allein auf `cfg.TenantID` statt
    zusaetzlich `tenantForWrite(ctx)` zu pruefen — der einzige produktive Aufrufer setzt den Wert
    korrekt, also kein Fund, aber als Konsistenz-Beobachtung vermerkt (keine eigene Unit, da kein
    reproduzierbarer Bug, nur fehlende Verteidigungslinie gegen einen hypothetischen zweiten
    Aufrufer).
- gate: n.a. -- reine Backlog-Recherche, kein Produktionscode geaendert. `git status --short`
  zeigt vor dem Commit nur BACKLOG.yml/JOURNAL.md.
- coverage: n.a. (Scan-Unit, kein Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein Verhalten geaendert)
- verify vorgaenger: sauber (Commit `98874542`, Iteration 18 — reine Backlog/Journal-Aenderung,
  kein Produktionscode; kein gRPC-Bypass, kein Stub, kein Proto/Migrations-Drift, kein neuer
  Guard, keine neue Tabelle, keine Wire-Shape- oder Routenaenderung moeglich, da kein Code
  angefasst wurde)
- neue-units: keine (0 Funde in allen neun Zielpaketen, siehe Belegliste oben)
- offen: Teil C (`scan-insert-missing-tenant-id-c`, Modul-Services inventar/einkauf/produktion/
  vertraege/rapporte/schichten/vermietung/fuhrpark/helpdesk/wiki/formulare/berichte/dialer/
  automation/plugin/settings/video/caldav — caldav ist durch diese Iteration bereits mitgeprueft
  und kann in Teil C uebersprungen werden) steht noch aus und ist bereits als todo im Backlog.
  Muster B ist damit fuer 2 von 3 Teilen ohne weitere Funde abgeschlossen.

## Iteration 20 — scan-insert-missing-tenant-id-c — done — 2026-08-11 18:17
- commit: (siehe git log nach diesem Eintrag)
- gebaut: Muster B (INSERT setzt tenant_id nicht, obwohl NOT NULL ohne Default), Teil C von 3 —
  die restlichen 17 Modul-Service-Pakete `internal/{inventar,einkauf,produktion,vertraege,
  rapporte,schichten,vermietung,fuhrpark,helpdesk,wiki,formulare,berichte,dialer,automation,
  plugin,settings,video}` per drei parallelen Explore-Subagenten geprueft (max. 3 gleichzeitig).
  `caldav` war bereits durch Iteration 19 mitgeprueft und wurde hier ausgelassen. `video` liegt
  faktisch nicht unter `internal/video`, sondern unter `internal/work/video` (Feature-Flag-Modul)
  — dort geprueft, trotz ausgeschaltetem Flag (Modul lief nie unter Produktionslast, ein
  fehlendes tenant_id faellt dort erst auf, wenn das Flag angeht).
  Vorgehen pro Agent identisch zu Teil A/B: alle `INSERT INTO`-Statements in Nicht-Test-Go-Dateien
  gefunden, Zieltabelle bestimmt, gegen die laufende lokale DB (Container `docker-postgres-1`,
  Kopf 312) per `\d <tabelle>` geprueft ob `tenant_id NOT NULL` ohne Default ist (mit Default
  automatisch kein Fund gemaess Vorgabe), System-Global-Liste (ADR-006) ausgenommen, dann
  Spaltenliste + Werteherkunft bis zum Struct-Feld/Service-Layer zurueckverfolgt.
  Gruppe 1 (inventar/einkauf/produktion/vertraege/rapporte/schichten): 42 INSERTs, 39 Tabellen,
  0 Funde. Alle tenant_id aus tenant-tragendem Struct-Feld (z. B. item.TenantID, order.TenantID),
  das im Service-Layer aus middleware.GetTenantID(ctx) befuellt wird.
  Gruppe 2 (vermietung/fuhrpark/helpdesk/wiki/formulare/berichte): 39 INSERTs, 0 Funde. 6 Tabellen
  (rental_objects, rentals, rental_inspections, vehicles, vehicle_services, vehicle_damages) haben
  einen Sentinel-Default und faellen damit ohnehin nicht unter Muster B; die restlichen 33 setzen
  tenant_id explizit korrekt, u. a. per tenant-validierter Subquery (vehicle_bookings JOIN users
  WHERE u.tenant_id = $2; ticket_csat_responses SELECT t.tenant_id FROM tickets WHERE t.tenant_id
  = $1 — tenant wird aus bereits validierter Elternzeile uebernommen statt vom Aufrufer erwartet).
  Gruppe 3 (dialer/automation/plugin/settings/work-video): 30 INSERTs, 0 Funde. dialer_call_sessions
  und automations haben einen Sentinel-Default (trotzdem explizit gesetzt); plugin_manifests ist
  bewusst NULLable (Systemkatalog-Eintraege ohne Tenant); plugin_permissions hat gar keine
  tenant_id-Spalte (Isolation ueber Join auf plugin_installations.tenant_id, analog zum
  roles-Muster aus Teil B — kein Fund); dialer_agent_status_log leitet tenant_id korrekt per
  Subquery aus users.tenant_id anhand der UserID ab.
  Insgesamt 111 INSERT-Statements in 17 Paketen geprueft, 0 Funde. Muster B (fehlende tenant_id
  bei INSERT) ist damit fuer alle 3 Teile des Scans (Iteration 18/19/20) ohne einen einzigen
  weiteren Fund abgeschlossen — der einzige Fund des gesamten Musters bleibt der bereits in
  Block A gefixte `fix-datev-upload-repo-missing-tenant-id`.
- gate: n.a. -- reine Backlog-Recherche, kein Produktionscode geaendert. `git status --short`
  zeigt vor dem Commit nur BACKLOG.yml/JOURNAL.md.
- coverage: n.a. (Scan-Unit, kein Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein Verhalten geaendert)
- verify vorgaenger: sauber (Commit `e5b29de8`, Iteration 19 — reine Backlog/Journal-Aenderung,
  kein Produktionscode; kein gRPC-Bypass, kein Stub, kein Proto/Migrations-Drift, kein neuer
  Guard, keine neue Tabelle, keine Wire-Shape- oder Routenaenderung moeglich, da kein Code
  angefasst wurde)
- neue-units: keine (0 Funde in allen 17 Zielpaketen, siehe Belegliste oben)
- offen: Muster B (INSERT ohne tenant_id) ist jetzt vollstaendig gescannt (Teil A+B+C, alle
  Service-Pakete). Block B des Lauf-9-Backlogs hat noch drei weitere Muster-Scan-Units offen:
  scan-nil-slice-wire-shape-large-services, scan-nil-slice-wire-shape-remaining-services
  (Muster C) sowie die bereits laufenden fix-*-Units aus Funden frueherer Scans
  (fix-audit-verifychain-timestamp-precision-mismatch, fix-notification-mutes-unique-missing-
  tenant, fix-work-erasure-task-double-count, fix-security-ip-access-rules-cidr-scan,
  fix-chat-guest-sessions-ip-address-scan, fix-caldav-carddav-company-permission-role-column,
  fix-work-recording-consent-missing-tenant-id).

## Iteration 21 — scan-nil-slice-wire-shape-large-services — done — 2026-08-11 18:21
- commit: (siehe git log nach diesem Eintrag)
- gebaut: Muster C (List-RPC gibt bei leerem Ergebnis nil-Slice statt make([]*T,0,n) zurueck,
  ueber protojson wird daraus JSON `null` statt `[]`), Teil 1 von 2 — die acht groessten
  gRPC-Server in `internal/server`: crm_grpc.go, biz_grpc.go, work_grpc.go, video_grpc.go,
  hr_grpc.go, calendar_grpc.go, email_grpc.go, document_grpc.go. Ueber drei parallele
  Explore-Subagenten geprueft (max. 3 gleichzeitig): Gruppe 1 crm/biz/work, Gruppe 2
  video/hr/calendar, Gruppe 3 email/document. Vorgehen pro Datei: alle `List`-praefigierten
  RPC-Funktionen gegrept, pro Treffer geprueft ob das Response-Slice vor der Append-Schleife
  mit `make(..., 0, n)` initialisiert wird oder als nackte `var xs []*T` stehen bleibt; wo
  plausibel zusaetzlich eine Ebene tiefer in den Service-Layer geschaut (Vorlage:
  `enrichWithRelationsBatch`).
  Ergebnis: 32 List-RPCs in video_grpc.go (9) + hr_grpc.go (10) + biz_grpc.go (5, ohne bereits
  gefixte) + email_grpc.go (7) + document_grpc.go (12) geprueft — 0 Funde, durchgaengig
  korrektes `make(...)`. document_grpc.go hat auf Repository-Ebene
  (`internal/document/file/postgres_repository.go`, 7 Funktionen) zwar dasselbe nil-Var-Muster,
  ist aber wirkungslos, weil jeder aufrufende Handler das Ergebnis bereits durch eine eigene,
  korrekt mit `make()` gepufferte Konvertierungsschleife reicht — kein Fund, da kein
  JSON-Response betroffen.
  30 Treffer in den restlichen zwei Dateien: calendar_grpc.go 13 von 13 geprueften List-RPCs
  betroffen (ListCalendars, ListCalendarMembers, ListBrowsableCalendars, ListEventsInRange,
  ListEventAttendees, ListEventCategories, ListEventReminders, ListResources,
  ListResourceAvailability, ListResourceBookings, ListHolidays, ListTaskDeadlinesInRange,
  ListBookingPages) plus Randfund GetAvailability (kein List-Praefix, gleiches Muster).
  crm_grpc.go 5 von 8 geprueften List-RPCs betroffen (ListCompanies, ListPipelineStages,
  ListDeals, ListActivities, ListSavedFilters — ListCustomFields/ListTags/ListContacts waren
  bereits durch fix-crm-list-nil-slice-wire-shape gefixt) plus Randfunde GetCompanyContacts,
  ReorderPipelineStages, Search. work_grpc.go 12 von 17 geprueften List-RPCs betroffen
  (ListProjects, ListProjectMembers, ListProjectStatuses, ListTasks, ListSubtasks,
  ListTaskDependencies, ListTaskComments, ListTaskEntityLinks, ListEntityTasks,
  ListTaskActivities, ListTaskFiles, ListTimeEntries) plus Randfunde ReorderProjectStatuses,
  SearchTasks.
  30 Funde in drei Dateien sind kein "wenige" im Sinne des done_when dieser Unit — Direkt-Fix
  in dieser Iteration haette Umfang und Testaufwand weit ueber eine Iteration hinaus getrieben.
  Stattdessen drei neue Fix-Units angelegt, GRUPPIERT PRO DATEI (Praezedenzfall
  fix-chat-guest-sessions-ip-address-scan aus Lauf 9 Iteration 11, das 3 Fundstellen im selben
  File in einer Unit zusammenfasst): fix-calendar-grpc-nil-slice-wire-shape (13 List-RPCs + 1
  Randfund), fix-crm-grpc-nil-slice-wire-shape-remaining (5 List-RPCs + 3 Randfunde),
  fix-work-grpc-nil-slice-wire-shape (12 List-RPCs + 2 Randfunde). Jede Unit ist derselbe
  mechanische Zwei-Zeilen-Diff pro Stelle (Vorlage Commit `c3f0c46f`), aber root-cause-identisch
  im selben File — ein Commit pro Datei ist angemessen granular, 30 Einzel-Units waeren reines
  Backlog-Rauschen fuer denselben Fix.
- gate: n.a. -- reine Backlog-Recherche, kein Produktionscode geaendert. `git status --short`
  zeigt vor dem Commit nur BACKLOG.yml/JOURNAL.md.
- coverage: n.a. (Scan-Unit, kein Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein Verhalten geaendert)
- verify vorgaenger: sauber (Commit `7fd2a4e3`, Iteration 20 — reine Backlog/Journal-Aenderung,
  kein Produktionscode; kein gRPC-Bypass, kein Stub, kein Proto/Migrations-Drift, kein neuer
  Guard, keine neue Tabelle, keine Wire-Shape- oder Routenaenderung moeglich, da kein Code
  angefasst wurde)
- neue-units: fix-calendar-grpc-nil-slice-wire-shape, fix-crm-grpc-nil-slice-wire-shape-remaining,
  fix-work-grpc-nil-slice-wire-shape
- offen: Teil 2 von 2 (`scan-nil-slice-wire-shape-remaining-services`, die restlichen 23
  `*_grpc.go`-Dateien: fuhrpark, inbox, helpdesk, chat, inventar, security, notification,
  dialer, rapporte, berichte, formulare, plugin, automation, settings, vermietung, wiki,
  schichten, einkauf, vertraege, produktion, bexio, lexware, datev_upload) steht noch aus und
  ist bereits als todo im Backlog. Die drei neuen Fix-Units sind ungewoehnlich gross (bis zu 14
  Fundstellen je Unit) -- falls eine davon in einer Iteration nicht vollstaendig passt, steht in
  ihren notes bereits eine explizite Anweisung, den Rest ehrlich als `offen:` zu vermerken statt
  als erledigt zu melden.

## Iteration 22 — scan-nil-slice-wire-shape-remaining-services — done — 2026-08-11 18:39
- commit: (siehe git log nach diesem Eintrag)
- gebaut: Muster C, Teil 2 von 2 — die restlichen 23 `*_grpc.go`-Dateien in `internal/server`:
  automation, berichte, bexio, chat, datev_upload, dialer, einkauf, formulare, fuhrpark,
  helpdesk, inbox, inventar, lexware, notification, plugin, produktion, rapporte, schichten,
  security, settings, vermietung, vertraege, wiki. Ueber drei parallele Explore-Subagenten
  geprueft (max. 3 gleichzeitig): Gruppe 1 automation/berichte/bexio/chat/datev_upload/dialer/
  einkauf/formulare, Gruppe 2 fuhrpark/helpdesk/inbox/inventar/lexware/notification/plugin/
  produktion, Gruppe 3 rapporte/schichten/security/settings/vermietung/vertraege/wiki. Vorgehen
  pro Datei wie Iteration 21: alle `List`-praefigierten RPC-Funktionen (plus auffaellige
  Get-/Search-Randfunde mit Listenfeld) gegrept, pro Treffer geprueft ob das Response-Slice vor
  der Append-Schleife mit `make(..., 0/len, n)` initialisiert wird oder als nackte
  `var xs []*T` stehen bleibt bzw. direkt auf ein unvorbelegtes Struct-Feld appended wird; bei
  Durchreiche-Mustern eine Ebene tiefer in Service-/Repository-Layer geschaut.
  Ergebnis: 107 List-RPCs (+ diverse Randfunde) ueber 23 Dateien geprueft — 21 Funde in 6
  Dateien, 17 Dateien vollstaendig sauber:
  - automation_grpc.go: 5/5 sauber. berichte_grpc.go: 5/5 sauber. bexio_grpc.go: 1/1 sauber.
    datev_upload_grpc.go: 1/1 sauber. dialer_grpc.go: 3/3 + 2 Randfunde sauber.
    einkauf_grpc.go: 3/3 sauber. formulare_grpc.go: 5/5 sauber.
  - chat_grpc.go: 5 Funde (ListChannels, ListDMs, ListChannelFiles, ListReactions, SearchChat),
    alle Handler-Ebene; Randfund ToggleReaction (Mutation, gleiches Muster).
  - fuhrpark_grpc.go: 9/9 sauber. helpdesk_grpc.go: 7/7 sauber. inbox_grpc.go: 6/6 sauber
    (make(...,0,n) durchgaengig). lexware_grpc.go: 1/1 sauber. produktion_grpc.go: 2/2 sauber.
  - inventar_grpc.go: 6/7 sauber, 1 Fund (ListItemAttachments, append auf unvorbelegtes
    Struct-Feld statt lokaler Slice-Variable — Bug-Variante ohne `var`-Deklaration).
  - notification_grpc.go: 5 Funde (ListNotifications, ListMutedResources, ListEventTypes,
    ListIntegrationConfigs, ListChannelMappings) + 2 Randfunde ohne List-Praefix
    (GetNotificationPreferences, GetAccountLinkStatus — Letzterer trifft den Normalfall, da
    die meisten Nutzer keinen verknuepften Account haben).
  - plugin_grpc.go: 6/7 sauber, 1 Fund (ListGrantedPermissions) — Handler ist reiner
    Durchreicher, die Nil-Quelle sitzt eine Ebene tiefer im Repository
    (internal/plugin/repository/permission.go:34-44).
  - rapporte_grpc.go: 7/7 sauber. settings_grpc.go: 3/3 + Randfund sauber (GetMyModuleLeads
    ist bereits explizit gegen nil abgesichert). vermietung_grpc.go: 3/3 sauber.
    vertraege_grpc.go: 4/4 sauber. wiki_grpc.go: 5/5 + Randfund sauber.
  - schichten_grpc.go: 4/4 Funde (ListShifts, ListAssignments, ListTemplates,
    ListSwapRequests), alle Handler-Ebene.
  - security_grpc.go: 5/7 Funde (ListAuditEntries, ListVaultSecrets, ListDataExports,
    ListIPRules, ListRetentionPolicies — Letztere zwei aus direkten pgx-Query-Loops statt
    Service-Aufrufen), ListVendorAccessRequests und DSARSearch sauber; Randfunde PreviewErasure/
    ExecuteErasure (Mutation, teilen dieselbe nil-Variable).
  21 Funde in 6 Dateien sind kein "wenige" im Sinne des done_when — sechs neue Fix-Units
  angelegt, GRUPPIERT PRO DATEI (gleiche Begruendung wie Iteration 21, Praezedenzfall
  fix-work-grpc-nil-slice-wire-shape): fix-chat-grpc-nil-slice-wire-shape (5 RPCs + 1
  Randfund), fix-notification-grpc-nil-slice-wire-shape (5 RPCs + 2 Randfunde),
  fix-inventar-grpc-list-item-attachments-nil-slice (1 RPC, eigene Unit da eigenes Modul),
  fix-plugin-grpc-granted-permissions-nil-slice (1 RPC, Root-Cause im Repository, daher eigene
  Unit mit service: plugin statt server), fix-schichten-grpc-nil-slice-wire-shape (4 RPCs,
  Hinweis auf Merge-Naehe zu fix-schichten-swap-assignments-unique-violation im selben File),
  fix-security-grpc-nil-slice-wire-shape (5 RPCs + 2 Randfunde).
- gate: n.a. -- reine Backlog-Recherche, kein Produktionscode geaendert. `git status --short`
  zeigt vor dem Commit nur BACKLOG.yml/JOURNAL.md.
- coverage: n.a. (Scan-Unit, kein Coverage-Ziel)
- mutations-probe: n.a. (Scan-Unit, kein Verhalten geaendert)
- verify vorgaenger: sauber (Commit `185d9ac7`, Iteration 21 — reine Backlog/Journal-Aenderung,
  kein Produktionscode; `git show --stat` zeigt ausschliesslich BACKLOG.yml/JOURNAL.md; kein
  gRPC-Bypass, kein Stub, kein Proto/Migrations-Drift, kein neuer Guard, keine neue Tabelle,
  keine Wire-Shape- oder Routenaenderung moeglich, da kein Code angefasst wurde)
- neue-units: fix-chat-grpc-nil-slice-wire-shape, fix-notification-grpc-nil-slice-wire-shape,
  fix-inventar-grpc-list-item-attachments-nil-slice, fix-plugin-grpc-granted-permissions-nil-slice,
  fix-schichten-grpc-nil-slice-wire-shape, fix-security-grpc-nil-slice-wire-shape
- offen: Muster C (nil-slice wire-shape) ist jetzt fuer alle 30 `*_grpc.go`-Dateien in
  internal/server vollstaendig gescannt (Teil 1 Iteration 21 + Teil 2 hier). Block B des
  Lauf-9-Backlogs ist damit VOLLSTAENDIG abgearbeitet (alle vier Muster A/A2/B/C gescannt).
  Block C hat jetzt insgesamt neun offene fix-*-Units aus fruehren Scans plus die sechs neuen
  aus dieser Iteration — naechste Iterationen ziehen normal von vorne nach hinten ab.

## Iteration 23 — fix-audit-verifychain-timestamp-precision-mismatch — done — 2026-08-11 18:32
- commit: (siehe git log nach diesem Eintrag)
- gebaut: `PostgresRepository.Create` (internal/security/audit/postgres_repository.go) truncatet
  `entry.Timestamp` jetzt auf Mikrosekunden-Praezision, BEVOR `computeEntryHash` aufgerufen wird
  -- audit_log.timestamp ist TIMESTAMPTZ (nur Mikrosekunden-Aufloesung), computeEntryHash
  formatiert aber mit RFC3339Nano. Dazu musste die ID-/Timestamp-Default-Bloecke (Zeile 57-62)
  vor die Hash-Berechnung gezogen werden (vorher lag computeEntryHash VOR den Defaults) -- als
  Nebeneffekt schliesst das auch die latente Luecke, dass ein Aufruf mit Zero-Timestamp mit dem
  Zero-Wert gehasht, aber mit time.Now() persistiert worden waere. Einziger echte Caller
  (`Service.LogEvent`, service.go:49) setzt Timestamp allerdings ohnehin immer selbst -- reine
  Haertung, kein zweiter aktiver Bug. Neuer DB-gestuetzter Test
  `TestPostgresRepository_VerifyChain_SubMicrosecondTimestampStillValid` erzwingt einen
  Timestamp mit expliziter Sub-Mikrosekunden-Ziffer (+123ns) und beweist valid=true danach. Der
  Truncate-Workaround-Kommentar in `TestPostgresRepository_VerifyChain_ValidSingleEntryRange`
  ist entfernt (Timestamp dort wieder normales `time.Now().UTC()` ohne Truncate), weil Create()
  das jetzt selbst uebernimmt.
- gate: build ok (`./internal/security/... ./internal/gateway/...`) | vet ok | lint ok
  (golangci-lint 0 issues) | test ok (`./internal/security/...` komplett gruen, 0 SKIP in
  `./internal/security/audit/ -v`) | migration n.a. (keine Migration) | rls-smoke n.a. (keine
  Tabellen-/Policy-Aenderung)
- coverage: internal/security/audit 80,0 % (Iteration-1-Messung) -> 80,1 % (lokal gemessen,
  `go tool cover -func`)
- mutations-probe: Truncate-Zeile per Kommentar deaktiviert -> neuer Test
  TestPostgresRepository_VerifyChain_SubMicrosecondTimestampStillValid wurde rot
  (`VerifyChain(...) = invalid ... want valid`), Truncate-Zeile zurueckgedreht, `git diff` zeigt
  wieder den sauberen Fix-Diff.
- verify vorgaenger: sauber (Commit `4507ec5d`, Iteration 22 — reine Backlog/Journal-Aenderung,
  `git show --stat` zeigt ausschliesslich BACKLOG.yml/JOURNAL.md, kein Produktionscode; kein
  gRPC-Bypass, kein Stub, kein Proto/Migrations-Drift, kein neuer Guard, keine neue Tabelle,
  keine Wire-Shape- oder Routenaenderung moeglich)
- neue-units: keine
- offen: Die im Unit-Scope dokumentierte Nicht-rueckwirkende-Grenze bleibt bestehen -- alle vor
  diesem Commit geschriebenen audit_log-Eintraege mit einem ungekappten Original-Hash bleiben
  mit ihrem alten (potenziell spurious-broken) Hash bestehen, append-only, nicht reparierbar.
  `go test ./internal/gateway/` nicht gelaufen, da keine Route angefasst wurde (keine Pflicht
  laut Schritt 5).

## Iteration 24 — fix-notification-mutes-unique-missing-tenant — done — 2026-08-11 18:36
- commit: (siehe git log nach diesem Eintrag)
- gebaut: Migration 000313 verbreitert den Unique-Constraint von `notification_mutes` von
  `UNIQUE(user_id, module_id, resource_id)` (Inline-Constraint aus 000022, nie mit 000110/000124
  mitgezogen) auf den Index `idx_notification_mutes_resource(tenant_id, user_id, module_id,
  resource_id)`; down stellt den alten Constraint her. `postgres_repository.go` blieb wie im
  Unit-Scope vermutet unveraendert -- `CreateMute` nennt keinen ON-CONFLICT-Arbiter, der Bug
  schlug daher nicht als 42P10 zu, sondern als 23505: `Service.MuteResource` prueft vorab
  tenant-gescopt per `IsResourceMuted`, sieht die Zeile des fremden Tenants nicht und laesst
  `CreateMute` in den tenant-losen Constraint laufen (roher 500 statt Mute). Neuer
  DB-gestuetzter Test `TestPostgresRepository_MutePerTenant` geht ueber den Service (nicht das
  Repo), beweist Mute fuer zwei Tenants auf derselben (user_id, module_id, resource_id), haelt
  `ErrMuteAlreadyExists` fuer das Duplikat innerhalb eines Tenants fest und pruefen beide
  ListMutedResources-Sichten auf Tenant-Isolation. Kommentar in
  `TestPostgresRepository_MuteLifecycle` auf die neue Index-Spaltenliste angeglichen.
- gate: build ok (`./internal/notification/... ./internal/gateway/... ./cmd/notification/...
  ./cmd/gateway/...`) | vet ok | lint ok (golangci-lint 0 issues) | test ok
  (`./internal/notification/...` alle 7 Pakete gruen, 0 SKIP in `./internal/notification/
  preference/ -v`) | migration ok (313/u angewendet, Kopf 313, Index per `\d
  notification_mutes` als UNIQUE btree (tenant_id, user_id, module_id, resource_id) verifiziert)
  | rls-smoke ok (eigener Tenant 1, unbeteiligter Tenant 0; die beiden Seed-Zeilen mit
  identischem (user_id, module_id, resource_id) in zwei Tenants gingen unter dem neuen Index
  ueberhaupt erst durch, Zeilen danach wieder geloescht)
- coverage: internal/notification/preference 87,4 % -> 87,4 % (beide selbst gemessen mit
  `go tool cover -func`; der neue Test faehrt ausschliesslich schon abgedeckte Zeilen --
  MuteResource, CreateMute, IsResourceMuted, ListMutedResources, ListMutes waren alle bereits
  durch TestPostgresRepository_MuteLifecycle erreicht. Der Wert deckt sich mit dem
  `coverage_start:` der Unit; Gewinn liegt hier im Schema-Fix, nicht in neuen Zeilen)
- mutations-probe: statt einer Code-Zeile die Migration selbst gebrochen --
  `migrate down 1` auf Kopf 312 gedreht, `TestPostgresRepository_MutePerTenant` wurde rot
  (`duplicate key value violates unique constraint
  "notification_mutes_user_id_module_id_resource_id_key" (SQLSTATE 23505)` genau an der
  Zeile des zweiten Tenants), danach `migrate up` zurueck auf 313, Test wieder gruen, Kopf und
  Arbeitsbaum sauber.
- verify vorgaenger: sauber (Commit `ef032f24`, Iteration 23 — Produktionsdiff ist der
  Timestamp-Truncate in `internal/security/audit/postgres_repository.go` plus Test; kein
  gRPC-Bypass (kein Handler beruehrt), kein Stub, kein `.proto`/kein Migrations-Drift, kein
  neuer RequirePermission-Guard, keine neue Tabelle, Wire-Shape unveraendert (nur der intern
  gehashte Timestamp wird gekappt, keine Response-Form), keine Route)
- neue-units: keine
- offen: Die drei Migrationen 000305 / 000312 / 000313 schliessen den Notification-Schema-Drift
  vollstaendig — im Schema bleibt keine Unique-Constraint mehr ohne `tenant_id`. Ob dieselbe
  Drift in ANDEREN Schemata liegt, ist bislang nicht systematisch gescannt; die drei Funde
  stammen alle aus dem Anlassfall Notification. Ein Scan `alle UNIQUE-Constraints/Indexe auf
  RLS-Tabellen ohne tenant_id als fuehrende Spalte` gegen die laufende DB waere eine
  lohnende Unit fuer einen naechsten Lauf (nicht angelegt, weil sie ausserhalb der
  Block-C-Fix-Liste dieses Laufs liegt). `go test ./internal/gateway/` nicht gelaufen, da keine
  Route angefasst wurde.
