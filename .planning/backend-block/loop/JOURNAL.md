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
