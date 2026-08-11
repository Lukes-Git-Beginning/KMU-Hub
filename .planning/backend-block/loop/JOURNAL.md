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
