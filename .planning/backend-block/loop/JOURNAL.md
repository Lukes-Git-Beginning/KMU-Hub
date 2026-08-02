# Backend-Nachtloop — Journal (Lauf 4)

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
- offen: <was Luke morgens pruefen muss>
```

Uhrzeiten im Journal sind geraten — der Agent hat keine Uhr. Die Wahrheit steht in `logs/run.log`.

Journale der Vorlaeufe: `archive/lauf-1-2/JOURNAL.md` (58 Units, PR #15),
`archive/lauf-3/JOURNAL.md` (61 Units, PR #16).

---

## Ausgangslage Lauf 4 (2026-08-02, vor der ersten Iteration)

Kein Vorgaenger-Commit zu verifizieren — Lauf 3 ist abgenommen, durch CI und gemergt. Die erste
Iteration ueberspringt den Verify-Vorspann daher zu Recht und beginnt direkt mit `p1b-proto`.

Stand, gegen den gebaut wird:

- Migrationskopf Repo **268**. Naechste freie Nummer zur Laufzeit ermitteln, nicht annehmen:
  `ls backend/migrations | grep -E '^[0-9]{6}' | sort | tail -1`
- Registrierte Routen **778** gegen **780** dokumentierte Pfade (`TestOpenAPIRouteDrift` gruen).
- Backend-Gates zuletzt vollstaendig gruen: `golangci-lint ./...` 0 issues,
  `go test ./...` 130 ok / 0 FAIL gegen echte DB, Coverage 28,3 %.
- RBAC-Fundament aus Welle 1a steht und ist empirisch geprueft: als `kmuhub_app` 8 Preset-Rollen,
  1179 Grants, admin = 454 Capabilities. Presets sind lesbar, aber fuer Tenants nicht
  schreib- oder loeschbar (Schreib-Policy ohne die NULL-Klausel; DELETE wertet nur USING aus).
- **Neu freigegeben:** RBAC Welle 1b (Block A). Gesperrt bleiben Phase 4 (Branchen-BE), neue
  `config.RequireX`-Assertionen, das Scharfschalten neuer `modules.*`-Flags sowie Merge/Deploy.

Schwerpunkte dieses Laufs: **Sicherheit/RLS-Reste** (Block B — die Allowlist in ADR-006 kennt vier
Ausnahmen, ungeschuetzt sind deutlich mehr), **Automatisierung fertigbauen** (Block C) und die vier
verifizierten FE-Luecken (Block D).

Ein Hinweis zur Erwartung: der Backlog hat **29 offene Units**, nicht 40. Die FE-Client-Pfade
aller Module wurden gegen die registrierten Routen gediffed — die duennen Module (fuhrpark,
inventar, vermietung, einkauf, produktion, schichten, rapporte) haben keine Routen-Luecken mehr.
Sind alle Units abgearbeitet, ist `ALLE UNITS ABGEARBEITET` ins Journal zu schreiben, `STOP`
anzulegen und der Lauf zu beenden — nicht nach Arbeit zu suchen.

---

## Iteration 1 — p1b-proto — done — 2026-08-02 20:15

- commit: dbe58528
- gebaut: 8 Rollen-Admin-RPCs am `AuthService` (`ListRoles`, `CreateRole`, `UpdateRole`,
  `DeleteRole`, `GetRolePermissions`, `SetRolePermissions`, `AssignUserRole`, `RevokeUserRole`)
  plus die Messages `Role`, `RoleGrant` und die acht Request/Response-Paare in
  `backend/proto/auth/v1/auth.proto`. Regenerat (`auth.pb.go`, `auth_grpc.pb.go`) liegt im
  selben Commit. **Nur Proto + Regen — keine Implementierung**, so wie die Unit es vorsieht.
- gate: build ok (`go build ./...` gesamtes Backend, Exit 0) | vet ok | lint ok
  (`golangci-lint run ./proto/auth/... ./internal/auth/... ./internal/server/...` → 0 issues)
  | test n.a. (kein Testcode in dieser Unit) | migration n.a. | rls-smoke n.a.
- verify vorgaenger: entfaellt — Lauf 3 ist abgenommen, gemergt und deployt; es gibt keinen
  Vorgaenger-Commit dieses Laufs.

### Vertrags-Entscheidungen (bindend fuer p1b-roles-list ff.)

Die Feldnamen sind aus dem FE-Vertrag gezogen (`desktop/src/renderer/src/api/rbac-types.ts`,
Verhalten gegengeprueft an `mocks/handlers/rbac.ts`). Drei Stellen weichen bewusst ab und
muessen im Gateway gemappt werden — wer das vergisst, baut genau den
Nested-Proto-vs-flacher-FE-Typ-Bruch, der dieses Repo schon mehrfach gekostet hat:

1. **`preset_id` (Proto) ↔ `basedOn` (FE).** Die DB-Spalte aus Welle 1a heisst `preset_id`;
   der FE-Typ nennt dasselbe Feld `basedOn`. Im Request heisst es `based_on` (so wie das FE
   es beim Anlegen schickt), im `Role`-Objekt `preset_id` (so wie die DB es haelt).
2. **`tenant_id` leer ⇒ JSON `null`.** proto3 kennt kein NULL fuer Strings. `tenant_id == ""`
   bedeutet System-Preset; der Gateway-Handler MUSS daraus `null` machen, weil der FE-Typ
   `tenantId: string | null` ist und `isSystem` daran haengt.
3. **`role_ids` (Proto) ↔ `roles` (FE).** `AssignUserRoleResponse`/`RevokeUserRoleResponse`
   tragen `repeated string role_ids`; der FE-Vertrag `UserRolesResponse` heisst `roles` und ist
   ebenfalls eine reine ID-Liste — **keine** Objekte. Nicht versehentlich `Role`-Objekte
   zurueckgeben.

Weiter: `RoleGrant` ist bewusst `repeated {key, scope}` und keine Proto-`map`. Der FE-Vertrag
ist eine Map (`Record<string, {scope}>`), die Uebersetzung Liste→Map gehoert in den Handler.
Eine Proto-Map haette hier keine stabile Reihenfolge und laesst sich schlechter validieren.

`UpdateRoleRequest` nutzt `optional` fuer `name`/`description`/`color` — PATCH-Semantik: nicht
gesetzt heisst "unveraendert", nicht "auf leer setzen". Muster uebernommen von
`UpdateUserRequest`.

### Hinweise fuer die naechste Iteration

- Der Auftritt der acht neuen Methoden in `UnimplementedAuthServiceServer` ist erwartete
  gRPC-Boilerplate und **kein** Stub-Fund im Sinne des Verify-Vorspanns. Der Diff im
  Regenerat (285 Loeschungen) besteht ausschliesslich aus verschobenen `msgTypes`-Indizes;
  der Generator-Versions-Header ist unveraendert, es gab also keinen Werkzeug-Drift.
- Fehlercodes stehen noch nicht im Proto. Der Vertrag aus `mocks/handlers/rbac.ts` ist:
  `preset_immutable` (403), `role_limit_reached` (409), `role_name_exists` (409),
  `role_has_members` (409), `last_admin` (409), `not_found` (404/422). Die Abbildung
  gRPC-Status → HTTP-Code gehoert in die Handler der Folge-Units.
- offen: nichts zu pruefen ausser dem Regenerat selbst.

## Iteration 2 — p1b-roles-list — done — 2026-08-02 20:29

- commit: 3dac5856
- gebaut: `GET /admin/roles` Ende-zu-Ende. `auth.Role` + `Service.ListRoles` (internal/auth/roles_admin.go),
  `PostgresRepository.ListRoles` mit zwei korrelierten Subqueries fuer member_count/capability_count
  (kein Cross-Join zweier LEFT JOINs — haette beide Zaehler aufgeblaeht), gRPC-Implementierung in
  `AuthGRPCServer.ListRoles` (internal/server/grpc.go), Gateway-Handler `HandleListRoles` mit
  `roleBody`/`rolesBody`-Mapping (camelCase, `tenantId`/`basedOn` als JSON `null` auf Presets — proto3
  liefert dafuer nur den leeren String, `nullIfEmpty` macht die Rueckuebersetzung). Route-Guard
  `RequirePermissionAny({"roles","manage"}, {"admin:role","read"})`: `roles:manage` ist die seit
  Migration 000002 bestehende grobe Admin-Permission, `admin:role:read` die feine Nachfolgerin aus
  Migration 000256 (dort schon fuer admin/it_admin/hr_admin geseedet — kein neuer Seed in dieser Unit
  noetig). `openapi.yaml`: Pfad `/api/v1/admin/roles` + Schemas `Role`/`RolesResponse`.
- member_count-Fund (aus den Unit-Notes uebernommen und umgesetzt): `user_roles` hat weder `tenant_id`
  noch RLS. Die Subquery joint deshalb ueber `users` (RLS-gescopt), sonst wuerde der member_count eines
  Presets (dieselbe role_id fuer alle Tenants) tenant-uebergreifend zaehlen.
- gate: build ok (`go build -p 2` fuer gateway/auth/server/cmd/auth/cmd/gateway) | vet ok | lint ok
  (golangci-lint, 0 issues auf allen drei Paketen) | test ok — `go test -count=1 ./internal/gateway/`
  gruen (inkl. `TestOpenAPIRouteDrift`: 779 registrierte Routen gegen 781 dokumentierte Pfade) und
  `go test -count=1 -v ./internal/auth/... ./internal/server/...` mit gesetzter `DATABASE_URL`
  (kmuhub_app): 319 PASS, 0 SKIP, 0 FAIL | migration n.a. (roles.tenant_id/based_on/color existieren
  schon seit 000256) | rls-smoke n.a. (keine Tabelle/Policy angefasst, RLS der Vorgaenger-Migration
  traegt die Query bereits)
- verify vorgaenger: sauber. `dbe58528` (p1b-proto) gegen die sechs Fehlerklassen geprueft — reiner
  Proto+Regen-Commit, `go build ./proto/auth/... ./internal/auth/... ./internal/server/...` gruen, keine
  Service-Implementierung im Diff (kein Stub-Fund im Sinne des Vorspanns).
- offen: Vier bestehende Mock-Repositories (`mockRepository` in internal/auth, `authMockRepo` in
  internal/server) mussten um eine No-Op-`ListRoles`-Methode ergaenzt werden, damit sie weiterhin das
  `Repository`-Interface erfuellen — reiner Interface-Fixup, keine fachliche Aenderung, aber falls eine
  spaetere Unit echte Rollback-Logik testen will, liefert dieser Mock aktuell immer `nil, nil`.

## Iteration 3 — p1b-roles-create — done — 2026-08-02 21:05

- commit: 797dfdb4
- gebaut: `POST /admin/roles` Ende-zu-Ende als **Klon**. `Service.CreateRole(ctx, tenantID, CreateRoleInput)`
  in `internal/auth/roles_admin.go` mit den drei Geschaeftsregeln (Budget `CustomRoleLimit = 20`,
  Namenskollision, sichtbare Klon-Quelle); Repository-Methoden `CountCustomRoles`, `RoleNameExists`,
  `CreateRole` (Transaktion: INSERT roles + `INSERT ... SELECT` auf `role_permissions`, `RowsAffected`
  ist der `capabilityCount` der neuen Rolle). gRPC `AuthGRPCServer.CreateRole` + neuer Mapper
  `toProtoRole` (den `ListRoles` jetzt mitbenutzt, statt das Mapping ein zweites Mal zu schreiben).
  Gateway-Handler `HandleCreateRole` mit `roleResponseBody` (`{role:{…}}`, weil das FE `resp.role` liest).
  Guard `RequirePermissionAny({roles,manage},{admin:role,create})` — `admin:role:create` ist seit
  Migration 000256 fuer admin und it_admin geseedet (in der DB nachgezaehlt), **kein neuer Seed noetig**.
  `openapi.yaml`: POST-Pfad + Schemas `RoleResponse`/`CreateRoleRequest`, alle Status-Codes des Handlers
  (400/401/403/404/409/422).
- entscheidungen:
  1. **Fehler-Sentinels tragen die FE-Codes als Message** (`role_limit_reached`, `role_name_exists`,
     `not_found`). `mapError` reicht `err.Error()` als gRPC-Message durch, `response.Error` schreibt
     `{"error": "<message>"}`, und `rbac-format.ts` schlaegt genau diesen String in `RBAC_ERROR_CODES`
     nach. Eine "schoenere" Prosameldung wuerde im Builder still zu "Unbekannter Fehler" —
     `TestRoleAdminErrorsCarryFrontendCodes` nagelt die drei Strings fest.
  2. **`ErrBaseRoleNotFound` ist ein eigener Sentinel**, nicht das bestehende `ErrRoleNotFound`: letzteres
     ist der Alt-Fehler von `AssignRole` und mappt auf FailedPrecondition → 409. Eine fehlende Klon-Quelle
     muss aber als 404 ankommen.
  3. **Namenskollision explizit geprueft statt ueber `ON CONFLICT`** (wie in den Unit-Notes angeraten).
     Der Ausdrucks-Index kann den FE-Vertrag ohnehin nicht abbilden: er vergleicht case-sensitiv und legt
     die Presets in einen eigenen COALESCE-Eimer, waehrend das FE case-insensitiv **und** gegen die
     Presets prueft. Die Unique-Violation (23505) wird trotzdem auf `ErrRoleNameExists` gemappt — als
     Netz fuer das Rennen zwischen Pruefung und INSERT.
  4. **Kein `database.BeginRLSTx`, sondern `pool.Begin`**: `internal/database` importiert `middleware`,
     und `middleware` importiert `internal/auth` — der Import waere ein Zyklus (der Compiler hat es
     sofort gezeigt). Aus demselben Grund kommt die Tenant-ID als Parameter aus der gRPC-Schicht statt
     via `middleware.GetTenantID` im Service, genau wie bei `CreateInvitation`. Die GUCs stampt
     `PrepareConn` beim Acquire, die Transaktion laeuft auf derselben Connection — RLS traegt.
  5. **422 im Handler statt ueber den Validator**: `decodeAndValidate` antwortet mit 400,
     der FE-Vertrag (`mocks/handlers/rbac.ts`) verlangt fuer fehlenden Namen/`basedOn` aber 422. Deshalb
     tragen `name`/`basedOn` kein `required`-Tag, sondern werden nach dem Decode explizit geprueft. Die
     Laengen-Tags (`max=50`/`max=40`) bleiben, sonst wird ein zu langer Name zum 22001 → 500.
- gate: build ok (`go build -p 2` auth/server/gateway/cmd) | vet ok | lint ok (golangci-lint, 0 issues) |
  test ok — `go test -count=1 ./internal/gateway/` gruen (inkl. `TestOpenAPIRouteDrift`),
  `go test -count=1 -v ./internal/auth/... ./internal/server/...` mit `DATABASE_URL` als `kmuhub_app`:
  **639 PASS, 0 SKIP, 0 FAIL** | migration n.a. (keine noetig — `roles`/`role_permissions` tragen seit
  000256 alles) | rls-smoke: keine Policy angefasst, die Isolation ist aber in zwei DB-Tests belegt
  (fremde Rolle als `basedOn` → `not_found`; gleicher Rollenname in zwei Tenants → beide erlaubt).
- verify vorgaenger: sauber. `3dac5856` (p1b-roles-list) gegen die acht Fehlerklassen geprueft — Handler
  ruft `client.ListRoles` ueber `getAuthClient()` (keine Layer-Umgehung), keine Stubs, kein `.proto` im
  Diff, Guard additiv (`RequirePermissionAny`), `openapi.yaml` im selben Commit, leere Liste als `[]`.
- offen:
  - Sechs neue DB-Tests in `internal/auth/roles_admin_db_test.go` raeumen ihre beiden eigenen Tenants
    (`40123a00-…-0001/0002`) vor **und** nach dem Lauf aus `roles` — sonst frisst ein abgebrochener
    Lauf ueber das 20er-Budget den naechsten. Wer die Tenant-UUIDs wiederverwendet, muss das wissen.
  - Der Klon kopiert Grants **inklusive scope**. Ein Aufrufer kann damit heute eine Rolle klonen, die
    mehr Rechte traegt als er selbst — der Escalation-Schutz ist bewusst erst `p1b-guardrails` (c).
  - `RoleNameExists` hat schon den `exceptID`-Parameter, den erst `p1b-roles-update-delete` fuer den
    Rename braucht; heute wird `uuid.Nil` uebergeben.

## Iteration 4 — p1b-roles-update-delete — done — 2026-08-02 22:10

- commit: 1f0f7c66
- gebaut: `PATCH /admin/roles/{id}` (Teil-Update: nur mitgeschickte Felder aendern sich) und
  `DELETE /admin/roles/{id}` (204). Service `UpdateRole`/`DeleteRole` in `internal/auth/roles_admin.go`
  pruefen ERST per neuem `Repository.GetRoleByID` (RLS-Read: Preset oder eigener Tenant, alles andere
  `ErrBaseRoleNotFound`), ob die Zielrolle ein Preset ist (`ErrRolePresetImmutable`), bevor irgendetwas
  geschrieben wird — die Schreib-Policy wuerde ein Preset ohnehin nur mit 0 Zeilen treffen, das waere ein
  stilles No-Op statt eines Fehlers. `UpdateRole` prueft danach `RoleNameExists(..., exceptID=roleID)`
  (Rename auf den eigenen aktuellen Namen ist keine Kollision), `DeleteRole` prueft `RoleHasMembers`
  (Join `user_roles` -> `users`, da `user_roles` weder `tenant_id` noch RLS hat — dasselbe Muster wie
  `ListRoles.member_count`). Repository: `GetRoleByID`, `UpdateRole` (eine Query, CTE-UPDATE + RETURNING
  mit denselben Member-/Capability-Count-Subqueries wie `ListRoles`), `RoleHasMembers`, `DeleteRole`
  (verlaesst sich auf die FK-Kaskaden von `role_permissions`/`user_roles` auf `role_id`, migration 000002).
  gRPC `AuthGRPCServer.UpdateRole`/`DeleteRole` + zwei neue `mapError`-Faelle
  (`ErrRolePresetImmutable` -> PermissionDenied/403, `ErrRoleHasMembers` -> FailedPrecondition/409).
  Gateway-Handler `HandleUpdateRole`/`HandleDeleteRole`, Guards additiv
  (`RequirePermissionAny({roles,manage},{admin:role,edit|delete})` — beide Keys seit Migration 000256 fuer
  admin und it_admin geseedet, in der DB nachgezaehlt, **kein neuer Seed noetig**). `openapi.yaml`: neuer
  Pfad `/api/v1/admin/roles/{id}` mit PATCH/DELETE + Schema `UpdateRoleRequest`.
- entscheidungen:
  1. **`GetRoleByID` vor jedem Schreibzugriff, nicht die Schreib-Policy allein entscheiden lassen.** Ein
     UPDATE/DELETE auf ein Preset traefe wegen `tenant_id = current_tenant_id()` in der `WITH CHECK`/
     `USING`-Klausel ohnehin 0 Zeilen — aber 0 betroffene Zeilen ist im Code nicht von "Rolle existiert
     nicht" unterscheidbar, und der FE-Vertrag (`mocks/handlers/rbac.ts:150,175`) verlangt fuer ein Preset
     explizit `preset_immutable` (403), nicht `not_found` (404). Deshalb liest der Service die Rolle zuerst.
  2. **`GetRoleByID` liefert `ErrBaseRoleNotFound` sowohl fuer unbekannte IDs als auch fuer Rollen anderer
     Tenants** — die RLS-Read-Policy macht Fremdrollen unsichtbar, ein Aufrufer darf nicht lernen, dass sie
     existieren. Getestet in `TestUpdateRole_DB_ForeignRoleIsNotFound`/`TestDeleteRole_DB_NotFound`.
  3. **`RoleHasMembers` und der `member_count` in `UpdateRole`s RETURNING joinen ueber `users`**, weil
     `user_roles` selbst weder `tenant_id` noch RLS traegt (siehe Block-B-Befund `g-user-roles-rls`) — ohne
     den Join wuerde `role_has_members` tenant-uebergreifend zaehlen. Exakt dasselbe Muster wie
     `ListRoles.member_count` (Iteration 2), hier wiederverwendet statt neu erfunden.
  4. **`UpdateRole`-Repo-Query ist eine einzige `WITH updated AS (UPDATE ... RETURNING ...) SELECT ...`
     Anweisung**, keine zwei Roundtrips: die Member-/Capability-Counts kommen als korrelierte Subqueries im
     selben Statement, damit der Aufrufer nicht zwischen UPDATE und Re-Read eine inkonsistente Zwischenzeile
     sehen kann.
  5. **`DeleteRole`s `RowsAffected()==0`-Check ist ein Race-Backstop, kein Primaerschutz** — Preset- und
     Members-Pruefung laufen bereits im Service davor; der Check faengt nur die theoretische Luecke
     zwischen Pruefung und DELETE ab (kein `SELECT ... FOR UPDATE`, bewusst: Rollen-Admin ist kein
     Hochlast-Pfad, ein zusaetzlicher Lock waere Overhead ohne echten Nutzen hier).
- gate: build ok (`go build -p 2` auth/gateway/server/cmd) | vet ok | lint ok (golangci-lint, 0 issues) |
  test ok — `go test -count=1 ./internal/auth/... ./internal/gateway/... ./internal/server/...` mit
  `DATABASE_URL` als `kmuhub_app`: alle vier Pakete gruen, 649 PASS (334 Top-Level + 315 Subtests laut
  `--- PASS`-Zeilen), **0 SKIP, 0 FAIL** (gezaehlt ueber `grep -c`). `TestOpenAPIRouteDrift`: 780
  registrierte gegen 782 dokumentierte Pfade, gruen. | migration n.a. (keine noetig — `roles`/
  `role_permissions`/Guards tragen seit 000256 alles) | rls-smoke: keine neue Policy/Tabelle angefasst,
  nur neue tenant-gescopte SELECTs/UPDATE/DELETE auf `roles`; Isolation ueber vier neue DB-Tests belegt
  (`TestUpdateRole_DB_ForeignRoleIsNotFound`, `TestDeleteRole_DB_NotFound`, `TestUpdateRole_DB_NameCollision`
  ueber zwei eigene Tenants, `TestDeleteRole_DB_RoleHasMembers` inkl. echtem User+Assignment) statt eines
  manuellen psql-Snippets.
- verify vorgaenger: sauber. `797dfdb4` (p1b-roles-create) gegen die acht Fehlerklassen geprueft — Handler
  geht ueber `client.CreateRole` (keine Layer-Umgehung), keine Stubs, kein `.proto`-Drift in diesem Commit
  (bereits in Iteration 1 regeneriert), Guard additiv, `openapi.yaml` im selben Commit, Fehlercode
  `not_found` bei fehlendem Namen/`basedOn` deckt sich exakt mit `mocks/handlers/rbac.ts:133`.
- offen:
  - Vier Mock-Repositories (`mockRepository` in `internal/auth`, `authMockRepo` in `internal/server`)
    mussten um No-Op-`GetRoleByID`/`UpdateRole`/`RoleHasMembers`/`DeleteRole` ergaenzt werden, damit sie das
    `Repository`-Interface weiter erfuellen — reiner Interface-Fixup, kein fachlicher Test dahinter (echte
    Abdeckung liegt in `roles_admin_db_test.go` gegen die echte DB).
  - `p1b-role-permissions` (naechste Unit) baut `GET/PUT /admin/roles/{id}/permissions` — die
    Preset-Immutability-Pruefung dort sollte denselben `GetRoleByID`-Weg gehen statt eine dritte Variante zu
    erfinden.
  - Escalation-Schutz beim Rename/Grant-Wechsel ist weiterhin bewusst nicht Teil dieser Unit
    (`p1b-guardrails`, Regel c) — `UpdateRole` aendert nur Name/Beschreibung/Farbe, keine Grants.

## Iteration 5 — p1b-role-permissions — done — 2026-08-02 22:35

- commit: cd5e8a79
- gebaut: `GET /admin/roles/{id}/permissions` -> `{roleId, grants:{key:{scope}}}` (Presets lesbar — der
  Builder zeigt das Grant-Set eines Presets waehrend er noch auswaehlt, wovon geklont wird) und
  `PUT /admin/roles/{id}/permissions` (Vollersatz: fehlende Keys entzogen, neue eingefuegt, geaenderte
  Scopes aktualisiert, alles in einer Transaktion). Service `GetRolePermissions`/`SetRolePermissions` in
  `internal/auth/roles_admin.go` gehen denselben `GetRoleByID`-Weg wie `UpdateRole`/`DeleteRole` (Vorschlag
  aus Iteration 4 aufgegriffen): GET erlaubt Presets, PUT prueft `TenantID == nil` -> `ErrRolePresetImmutable`
  vor jedem Schreiben. Neuer Scope-Check im Service (jeder Grant muss `own|team|all` sein, sonst
  `ErrCapabilityKeyUnknown`) faengt zu lange/falsche Werte ab, BEVOR sie die DB erreichen — ohne ihn waere
  ein Scope laenger als `VARCHAR(8)` ein rohes 500 statt des sauberen 422 (real beim ersten Testlauf
  aufgefallen: `22001 value too long` statt der erwarteten Sentinel). Repository:
  `GetRolePermissions` (Read ueber `role_permissions`' eigene RLS-Policy, kein expliziter Tenant-Filter
  noetig), `SetRolePermissions` (eine Transaktion: erst alle Keys per `unnest($1::text[])` LEFT-JOIN-Check
  gegen `permissions.name` validieren — ein unbekannter Key bricht den GESAMTEN Schreibvorgang ab, bevor
  irgendetwas geloescht wird —, dann `DELETE FROM role_permissions WHERE role_id=$1`, dann Bulk-`INSERT
  ... SELECT ... FROM unnest($2::text[], $3::text[]) AS g(key, scope) JOIN permissions`; der
  `role_permissions_scope_check`-Constraint (000256) ist ein zweiter Backstop, den ein direkter
  Repo-Aufruf mit ungueltigem Scope tatsaechlich mid-transaction ausloest). gRPC `AuthGRPCServer.
  GetRolePermissions`/`SetRolePermissions` (Proto-RPCs/Messages waren seit Iteration 1 fertig generiert,
  keine `.proto`-Aenderung noetig) + `mapError`-Fall `ErrCapabilityKeyUnknown` -> `codes.OutOfRange`.
  Gateway-Handler `HandleGetRolePermissions`/`HandleSetRolePermissions`, Guards additiv
  (`RequirePermissionAny({roles,manage},{admin:role,read|edit})` — beide Keys seit Migration 000256
  geseedet, **kein neuer Seed noetig**). `grpcStatusToHTTP` bekam einen neuen Fall
  `codes.OutOfRange -> http.StatusUnprocessableEntity` (422) — vorher unbenutzt im Repo, verifiziert per
  Grep. `openapi.yaml`: neuer Pfad `/api/v1/admin/roles/{id}/permissions` mit GET/PUT + Schemas
  `RoleGrant`/`RolePermissionsResponse`/`UpdateRolePermissionsRequest`.
- entscheidungen:
  1. **GET erlaubt Presets, nur PUT ist gesperrt.** Weder Backlog-Notes noch FE-Mock verlangen eine
     Preset-Sperre beim Lesen (`rbac.ts` hat keinen `isPresetRole`-Check auf der GET-Route), und der Builder
     braucht das Preset-Grant-Set als Ausgangspunkt beim Klonen. Nur `PUT` prueft `TenantID == nil`.
  2. **Scope-Validierung im Service, nicht nur der DB-Constraint ueberlassen.** Der ADR-Notiz-Gedanke
     "der Wert kommt vom FE-Typunion, kann also nicht falsch sein" haelt nicht fuer jeden Aufrufer der API —
     ein zu langer String durchbricht die Kette VOR dem CHECK-Constraint mit einem Laengenfehler
     (`22001`), der als 500 durchgereicht worden waere. Der Service-Check faengt beides (unbekannter Wert
     UND falsche Laenge) einheitlich als `unknown_capability_key`/422 ab; der DB-Constraint bleibt Backstop
     fuer den direkten Repo-Aufruf (siehe Transaktionalitaets-Test).
  3. **Unbekannte Keys werden VOR jedem Schreiben geprueft (SELECT vor DELETE), nicht erst beim INSERT
     erkannt.** Ein LEFT-JOIN-basierter INSERT wuerde einen unbekannten Key stillschweigend fallenlassen
     (`JOIN permissions` liefert fuer ihn keine Zeile) statt den ganzen Aufruf abzulehnen — genau das
     verbietet der Backlog explizit ("nicht still verschluckt"). Die separate Validierungsquery lohnt sich:
     ein `COUNT(*)`-Check gegen `unnest($1::text[])` ist billiger als das Risiko eines Teil-Schreibens.
  4. **Delete-dann-Insert statt Upsert+Differenz**, weil die Semantik "Vollersatz" ist (jeder fehlende Key
     wird entzogen) — ein Upsert muesste die verschwundenen Keys separat per Diff loeschen, was zwei
     Statements plus eine Zwischenmenge braucht; Delete-dann-Insert ist ein Statement pro Richtung und
     dieselbe Transaktion deckt beide ab.
  5. **`toProtoRoleGrants`/`toRoleGrantsBody` bauen IMMER eine nicht-nil Struktur** (leere Grants ->
     `[]`/`{}`, nie `null`) — dasselbe Wire-Shape-Muster wie `toEffectivePermissionsBody` aus Welle 1a,
     hier neu fuer eine Map statt einer Liste angewendet und mit einem eigenen Test gepinnt
     (`TestToRoleGrantsBody_EmptyIsContainerNotNull`).
- gate: build ok (`go build -p 2` auth/gateway/server/cmd) | vet ok | lint ok (golangci-lint, 0 issues,
  auth+gateway+server) | test ok — `go test -count=1 ./internal/auth/... ./internal/gateway/...
  ./internal/server/...` mit `DATABASE_URL` als `kmuhub_app`: alle vier Pakete gruen, **1762 PASS, 0 SKIP,
  0 FAIL** (gezaehlt ueber `grep -c -- "--- PASS/SKIP/FAIL"`). `TestOpenAPIRouteDrift`: 781 registrierte
  gegen 783 dokumentierte Pfade, gruen. | migration n.a. (keine noetig — `role_permissions` traegt Scope
  und RLS seit 000256) | rls-smoke: keine neue Policy/Tabelle angefasst, nur neue tenant-gescopte
  SELECT/DELETE/INSERT auf `role_permissions` ueber dessen bestehende RLS-Policy; Isolation ueber
  `TestGetRolePermissions_DB_NotFound` (fremde Rolle -> `not_found`, RLS macht sie unsichtbar) belegt statt
  eines manuellen psql-Snippets.
- verify vorgaenger: sauber. `1f0f7c66` (p1b-roles-update-delete) gegen die acht Fehlerklassen geprueft —
  Handler gehen ueber `client.UpdateRole`/`client.DeleteRole` (keine Layer-Umgehung), keine Stubs, kein
  `.proto`-Drift (bereits Iteration 1), Guards additiv mit bereits geseedeten `admin:role:edit`/`delete`
  (in der Migration nachgezaehlt), `openapi.yaml` im selben Commit, `GetRoleByID`-Vorcheck verhindert den
  stillen No-Op auf Presets, den die Schreib-Policy allein erzeugt haette.
- offen:
  - `p1b-user-roles` (naechste Unit in der Kette) muss das `oneof` in `assignRoleRequest`/dem Validator
    entkoppeln, damit Custom-Rollen ueberhaupt zuweisbar werden — diese Unit hat daran nichts geaendert.
  - Der neue `codes.OutOfRange -> 422`-Fall in `grpcStatusToHTTP` ist bisher nur von
    `ErrCapabilityKeyUnknown` belegt; falls ein spaeterer Fund einen echten OutOfRange-Anwendungsfall
    braucht (z. B. Pagination), diesen Praezedenzfall zuerst lesen statt eine zweite 422-Konvention
    einzufuehren.
  - `unknown_capability_key` steht noch nicht in `RBAC_ERROR_CODES` (`rbac-format.ts:50`) — der Fehler
    kommt beim FE aktuell als generische Meldung an (`rbac.builder.errors.generic`), nicht als spezifischer
    Text. Kein Blocker (Backend-Scope dieser Unit), aber ein offener FE-Punkt fuer das naechste Antasten
    von `rbac-format.ts`.

## Iteration 6 — p1b-user-roles — done — 2026-08-02 23:05
- commit: 60fc6dae
- gebaut: Rollen-Zuweisung auf IDs umgestellt — `POST /api/v1/users/{id}/roles` nimmt jetzt
  `{roleId}` (statt `{role_name}` mit `oneof=admin manager member`) und `DELETE
  /api/v1/users/{id}/roles/{roleId}` traegt die Rolle im Pfad; beide antworten mit
  `{roles:[<role-ids>]}` (FE-Contract `UserRolesResponse`, `rbac-client.ts`).
  Service `AssignUserRole`/`RevokeUserRole` in `roles_admin.go`: Ziel-User ueber `GetUserByID`
  (users-RLS) und Rolle ueber `GetRoleByID` (roles-RLS) aufloesen, dann schreiben, dann die neue
  Liste lesen. Repo-Trio `AssignUserRole`/`RevokeUserRole`/`GetUserRoleIDs` in
  `postgres_repository.go`: das INSERT selektiert beide Seiten aus `users` CROSS JOIN `roles`
  zurueck, das DELETE laeuft ueber `USING users, roles` — weil `user_roles` weder `tenant_id` noch
  RLS hat (Block B), sind diese Joins die Tenant-Grenze des Schreibens, nicht Komfort.
  gRPC `AssignUserRole`/`RevokeUserRole` (Proto-RPCs seit Iteration 1 generiert, **keine**
  `.proto`-Aenderung noetig). `openapi.yaml`: neuer Pfad `/api/v1/users/{id}/roles/{roleId}`,
  neuer Parameter `RoleId`, Schemas `AssignRoleRequest`/`UserRolesResponse` ersetzen `RoleRequest`.
- entscheidungen:
  1. **Das `oneof` ist ersatzlos weg, die Validierung liegt in der DB-Schicht.** Der Tag konnte
     strukturell nur die drei Seed-Presets nennen — eine Custom-Rolle von Welle 1b war damit nicht
     zuweisbar, egal wie der Builder sie anlegt. Statisch geht das nicht zu reparieren: welche
     Rollen ein Tenant hat, weiss nur die `roles`-Tabelle. Der Handler prueft jetzt nur noch
     UUID-Form (`validate:"required,uuid"`), die Existenz-/Sichtbarkeitsfrage beantwortet der
     Service gegen RLS.
  2. **Presets bleiben zuweisbar.** `admin`/`member` SIND Presets — der Preset-Check aus
     `UpdateRole`/`DeleteRole` hier zu kopieren haette die haeufigste Zuweisung ueberhaupt
     gesperrt. Immutability betrifft die Rollen-Definition, nicht ihre Traegerschaft.
  3. **Guard von `RequireRole("admin")` auf `RequirePermissionAny({roles,manage},
     {admin:role,assign})`** — das ist eine Ausweitung, keine Ersetzung: jedes Admin-Token traegt
     `roles:manage` (000002 gibt dem admin-Preset per CROSS JOIN den ganzen Katalog, in der
     Migration nachgelesen), also verliert niemand Zugriff. Gewinnen tun `it_admin`/`hr_admin`,
     die `admin:role:assign` seit 000256 geseedet haben — genau das Muster aus
     PHASE-1-RBAC-PLAN (hr_admin darf zuweisen, aber nicht editieren). Kein neuer Seed noetig.
  4. **DELETE traegt die Rolle im Pfad statt im Body.** Das FE adressiert die Zuweisung als eigene
     Ressource (`removeUserRole` in `rbac-client.ts`, Mock-Handler `:userId/roles/:roleId`), und
     ein DELETE mit Payload ist fuer Clients unangenehm. Die alte Body-Variante ist damit weg —
     verifiziert, dass sie ausser der (generierten) `types.ts` keinen Aufrufer hatte.
  5. **Ein Revoke einer nicht gehaltenen Rolle ist ein No-Op, kein 404.** Der Aufrufer verlangt
     einen Zustand, keine Transition; ein 404 wuerde den Doppelklick im Builder zum Fehler machen.
  6. `withChiURLParam` (Gateway-Testhelfer) haengt einen Parameter jetzt an einen vorhandenen
     RouteContext an, statt jedes Mal einen neuen anzulegen — vorher verschluckte der zweite
     Aufruf den ersten Parameter, was bei einer Route mit zwei IDs still das Falsche testet.
- gate: build ok (`go build -p 2` auth/gateway/server/cmd) | vet ok | lint ok (golangci-lint,
  0 issues, auth+gateway+server) | test ok — `go test -count=1 -v ./internal/auth/...
  ./internal/gateway/ ./internal/server/...` mit `DATABASE_URL` als `kmuhub_app`:
  **1777 PASS, 0 SKIP, 0 FAIL**; die 10 neuen `*_DB_*`-Tests namentlich als PASS verifiziert.
  `TestOpenAPIRouteDrift` gruen, `swagger-cli validate backend/api/openapi.yaml` = valid.
  | migration n.a. (keine noetig — `user_roles` und `roles` stehen seit 000002/000256, beide
  Guard-Keys geseedet) | rls-smoke: keine Policy/Tabelle geaendert; die Isolation ist stattdessen
  in vier Tests gepinnt (Fremd-Tenant-User -> `user not found`, Fremd-Tenant-Rolle -> `not_found`,
  jeweils mit Nachzaehlen, dass **null** Zeilen in `user_roles` landen, plus ein Test, der die
  Repository-Methode direkt unter fremdem Tenant aufruft und beweist, dass die Joins allein
  schon nichts schreiben).
- verify vorgaenger: sauber. `cd5e8a79` (p1b-role-permissions) gegen die acht Fehlerklassen
  geprueft — Gateway-Handler gehen ueber `client.GetRolePermissions`/`client.SetRolePermissions`
  (keine Layer-Umgehung), die vier `return nil, nil` im Diff liegen ausschliesslich in
  Test-Mocks (`service_test.go`, `testhelpers_test.go`), kein `.proto` im Commit, Guards additiv
  mit `admin:role:read|edit` (Seed in 000256 nachgezaehlt), `openapi.yaml` im selben Commit,
  Wire-Shape `{roleId, grants}` deckt sich mit `rbac-types.ts`.
- offen:
  - **`desktop/src/renderer/src/api/types.ts` ist nicht neu generiert** (Zeile ~297 beschreibt
    weiterhin `RoleRequest`/`StatusResponse` und ein DELETE ohne `{roleId}`). Kein CI-Gate (der
    `openapi-validate`-Job validiert nur die Spec, er generiert nicht), und `rbac-client.ts`
    nutzt eigene Typen — aber wer die generierten Typen liest, sieht den alten Vertrag.
  - `p1b-guardrails` (naechste Unit) setzt vor `RevokeUserRole` an: last_admin (409),
    Selbst-Aussperrung, Privilege-Escalation, Preset-Immutability. Die Stelle ist im Service
    kommentiert, absichtlich noch leer — nicht doppelt bauen.
  - `AssignUserRole` liefert bei unsichtbarem Ziel-Account `user not found`, nicht `not_found`;
    `RBAC_ERROR_CODES` in `rbac-format.ts` kennt den String nicht, das FE zeigt dort also die
    generische Meldung. Bewusst so gelassen: `ErrUserNotFound` ist repo-weit in Auth-Pfaden im
    Einsatz, sein Wortlaut ist kein RBAC-Detail.

## Iteration 7 — p1b-guardrails — done — 2026-08-02 21:15
- commit: 95ce32f0
- gebaut: Die vier Guardrails aus PHASE-1-RBAC-PLAN §4, zentral in
  `internal/auth/guardrails.go` (neu) und von `roles_admin.go` aus verdrahtet — kein Handler kennt
  eine davon. (a) **last_admin**: `CountRoleAdminsExcluding` zaehlt die aktiven Accounts des Tenants,
  die Rollen vergeben duerfen, unter Ausschluss genau der Zuweisung, die entzogen werden soll; 0 ->
  `ErrLastAdmin` (409). (b) **self_lockout**: beim Revoke an sich selbst und beim Umschreiben einer
  Rolle, die der Aufrufer traegt (`assertKeepsOwnRoleAdmin`). (c) **privilege_escalation**: die
  aufgeloeste Grant-Menge des Aufrufers (`callerReach`, Union ueber seine Rollen, breitester Scope)
  ist die Obergrenze fuer `SetRolePermissions`, den `CreateRole`-Klon und die **Selbst**-Zuweisung in
  `AssignUserRole`. (d) **Preset-Immutability** stand schon auf allen drei Schreibpfaden und ist jetzt
  in einem Test ueber alle drei gepinnt. Actor kommt als Parameter (auth darf `middleware` nicht
  importieren — Cycle), aufgeloest in `server/grpc.go` via neuem `callerID(ctx)` aus dem
  propagierten `x-user-id`; fehlt der, ist es `Unauthenticated`, nie `uuid.Nil`.
- entscheidungen:
  1. **"Admin" = wer Rollen vergeben darf** (`roles:manage` ODER `admin:role:assign`), nicht "wer
     das admin-Preset traegt". Ein Tenant, der das Preset in "Geschaeftsfuehrung" klont und das
     Original ablegt, hat weiterhin Administratoren — ein Namensvergleich saehe sie nicht und
     last_admin wuerde auf einem gesunden Tenant feuern. Es ist exakt das Paar, das die
     Assign-Routen seit Iteration 6 bewachen.
  2. **Fremd-Zuweisung bleibt frei, Selbst-Zuweisung ist gedeckelt.** Der Plan gibt hr_admin
     `admin:role:assign` ohne `role:create/edit` — es SOLL Rollen besetzen koennen, die mehr
     duerfen als es selbst (member-Preset ⊄ hr_admin). Haenge ich (c) an jede Zuweisung, ist das
     Preset wertlos. Der reale Eskalationsweg ist "ich gebe sie MIR", und genau der ist zu.
  3. **Der Klon faellt unter (c).** Sonst ist er der Weg um `SetRolePermissions` herum: Rechte
     kommen ueber die Kopie herein statt ueber eine Bearbeitung. Konsequenz, im Test festgehalten:
     nur wessen Reichweite ein Preset abdeckt, kann es klonen — it_admin scheitert schon an
     `extern` (documents:file:upload/download fehlen ihm). Praktisch klont nur admin frei.
  4. **`SetRolePermissions` prueft den Katalog VOR der Reichweite** (neue Repo-Methode
     `CountUnknownPermissionKeys`). Ohne das kommt ein Tippfehler als 403 zurueck statt als 422:
     ein Key, den es nicht gibt, liegt in niemandes Grant-Menge, und (c) kann ihn nicht von einem
     Griff nach einem echten Recht unterscheiden. Der Check im Repo bleibt als Backstop (Race).
  5. **`GetUserGrants` neben `GetEffectivePermissions`** statt eines Flags: die Bildschirm-Ansicht
     filtert die groben Alt-Keys (`p.resource LIKE '%:%'`) bewusst weg, die Autorisierung darf das
     nicht — die groben Keys sind die Waehrung der `RequirePermission`-Gates, wer sie vergeben
     darf, oeffnet jedes Gate.
- gate: build ok (`go build -p 2` auth/server/gateway/cmd) | vet ok | lint ok (golangci-lint,
  0 issues ueber auth+server+gateway) | test ok — `go test -count=1 -v ./internal/auth/...
  ./internal/gateway/ ./internal/server/...` mit `DATABASE_URL` als `kmuhub_app`:
  **1787 PASS, 0 SKIP, 0 FAIL** (1777 vorher + 10 neue `TestGuardrail_DB_*`, alle namentlich als
  PASS verifiziert). `TestOpenAPIRouteDrift` gruen, `swagger-cli validate` = valid.
  | migration n.a. — keine neue Tabelle, kein neuer `RequirePermission`-Guard (die Routen-Guards
  aus Iteration 6 bleiben unveraendert, also auch kein Seed noetig) | rls-smoke: keine Policy
  angefasst; die Tenant-Grenze der neuen Zaehlung ist stattdessen im last-admin-Test gepinnt —
  er seedet einen Admin in einem **fremden** Tenant, der nicht mitzaehlen darf. Ohne den
  `users`-Join in `CountRoleAdminsExcluding` waere der Test rot.
- verify vorgaenger: sauber. `60fc6dae` (p1b-user-roles) gegen die acht Klassen geprueft — Handler
  gehen ueber `client.AssignUserRole`/`client.RevokeUserRole`, kein Stub, kein `.proto` im Diff,
  Wire-Shape `{roles:[...]}` mit `[]`-Garantie via `toUserRolesBody`, `openapi.yaml` im selben
  Commit. Klasse 8 gezielt nachgezaehlt: der Wechsel `RequireRole("admin")` ->
  `RequirePermissionAny({roles,manage},{admin:role,assign})` verliert niemanden, weil 000002 dem
  admin-Preset per `CROSS JOIN permissions` den ganzen Katalog gibt und `roles:manage` (Zeile 51)
  darin liegt — `RequirePermissionAny` hat keinen Rollen-Fallback, das war der Punkt zum Pruefen.
- offen:
  - **`rbac-format.ts` kennt `self_lockout` und `privilege_escalation` nicht.** `RBAC_ERROR_CODES`
    listet fuenf Codes, meine beiden neuen sind nicht dabei — der Builder zeigt dort die generische
    Meldung. Zwei Zeilen dort plus vier i18n-Kataloge; bewusst nicht im Backend-Commit gemacht
    (i18n ×4 + Screenshot-QA ist ein eigener Workflow).
  - **Produktentscheidung fuer Luke:** ist "nur admin kann Presets klonen" zu streng? Die
    Alternative waere, den Klon frei zu lassen und stattdessen JEDE Zuweisung zu deckeln — dann
    verliert hr_admin sein `role:assign` praktisch. Ich halte die gewaehlte Seite fuer die
    richtige, aber es ist eine Produktfrage, keine technische.
  - `desktop/src/renderer/src/api/types.ts` weiterhin nicht regeneriert (Befund aus Iteration 6,
    unveraendert).
  - `p1b-audit-events` (naechste Unit) setzt genau auf diesen vier Pfaden auf; die Guardrails
    laufen alle VOR dem Write, ein abgelehnter Versuch darf also kein Event schreiben.

## Iteration 8 — p1b-audit-events — done — 2026-08-02 21:40
- commit: aa5fcca8
- gebaut: Die fuenf Audit-Events aus PHASE-1-RBAC-PLAN §4 (`permission.role_created/_updated/
  _deleted/assigned/revoked`) auf dem bestehenden `audit_log` (Migration 000039, append-only seit
  000222) — keine zweite Audit-Infrastruktur, sondern der schon vorhandene `audit.Service` aus
  `internal/security/audit`, der im auth-Prozess laengst konstruiert wird (`cmd/auth/main.go:87f`,
  bislang nur an `SecurityGRPCServer` durchgereicht). Neu in `internal/server/grpc.go`:
  `AuthGRPCServer` bekommt `auditService` als zweiten Konstruktor-Parameter, `tenantAndCaller(ctx)`
  buendelt die Tenant-/Akteur-Aufloesung (bislang nur `callerID`, UpdateRole/DeleteRole brauchten
  bisher keine von beiden), `logPermissionEvent` haengt EIN `LogEvent`-Aufruf direkt hinter den
  erfolgreichen Service-Call in allen fuenf Handlern (CreateRole, UpdateRole, DeleteRole,
  AssignUserRole, RevokeUserRole). Ziel-Ressource ist die Rolle (role_created/_updated/_deleted)
  bzw. der betroffene Account (assigned/revoked, `role_id` in `details`) — nie der Akteur selbst,
  der steht in `user_id`. `SetRolePermissions` bleibt bewusst ohne eigenes Event: das ist keine der
  fuenf Aenderungsarten aus dem Scope, sondern der Weg, ueber den role_updated fachlich hinausgeht
  (Namensaenderung) bzw. hinausginge, wenn spaeter Grant-Aenderungen mitprotokolliert werden sollen
  — als offener Punkt unten, nicht mitgebaut.
- entscheidungen:
  1. **Kein gRPC-Hop zur security.** `audit_log` gehoert der security-Domaene, aber `cmd/auth`
     hostet AuthGRPCServer UND SecurityGRPCServer im selben Prozess (Kommentar dort: "co-located in
     auth process") — genau wie SecurityGRPCServer selbst tut, haelt AuthGRPCServer jetzt eine
     direkte `*audit.Service`-Referenz. Ein RPC zur eigenen Security-Instanz waere ein
     Netzwerk-Hop fuer einen In-Prozess-Aufruf gewesen und keine echte Service-Grenze — die einzige
     bestehende Cross-Service-Nutzung von `SecurityServiceClient` sitzt im Gateway (Middleware),
     nicht zwischen zwei Domain-Services.
  2. **Event NACH dem Service-Call, nie davor.** Die vier Guardrails aus Iteration 7 lehnen einen
     unrechtmaessigen Versuch VOR dem Schreiben ab (`mapError` gibt zurueck, `logPermissionEvent`
     wird nie erreicht) — pytest-artig durch `TestPermissionAuditEvents_DB_RejectedWriteLeavesNoEvent`
     belegt (DeleteRole auf eine Rolle mit Mitglied: 409, Audit-Log-Zeilenzahl unveraendert).
  3. **`tenantAndCaller` statt getrennter Aufrufe** in UpdateRole/DeleteRole: beide brauchten vorher
     weder Tenant noch Akteur (RLS scopt Read+Write allein), jetzt brauchen sie beides nur fuer das
     Event. Ein Bundling-Helfer statt zwei separate Boilerplate-Zeilen pro Handler, konsistent mit
     dem bereits vorhandenen `callerID`.
  4. **IP-Adresse/User-Agent bleiben leer.** Diese Events entstehen im internen auth<->auth-Aufruf,
     nicht im HTTP-Request-Pfad wie `middleware/audit.go`s generisches CRUD-Logging — der
     Client-Kontext (X-Forwarded-For etc.) existiert an dieser Stelle nicht. `audit.Repository.Create`
     normalisiert die leere IP bereits auf SQL NULL (bestehender Code, nicht neu).
  5. **Fester Akteur-Testfixture statt `uuid.New()` pro Lauf.** Erster Testlauf zeigte: das
     `t.Cleanup`-DELETE auf den seedenden `users`-Datensatz schlaegt fehl, sobald ein Audit-Event
     ihn als `user_id` referenziert — `audit_log_user_id_fkey` plus Append-Only-Trigger machen den
     Fixture-User dauerhaft unloeschbar. Umgestellt auf eine feste UUID, idempotent geseedet
     (`ON CONFLICT DO NOTHING`), nie geloescht — analog dazu, wie `testutil.EnsureTenant` Tenants
     dauerhaft stehen laesst. Der `target`-User (nur im `target`-String, nie `user_id`) bleibt
     `uuid.New()` + normalem Cleanup, das funktioniert unveraendert.
- gate: build ok (`go build -p 2` auth/server/gateway/cmd/auth/cmd/gateway) | vet ok | lint ok
  (golangci-lint, 0 issues ueber server + cmd/auth) | test ok — `go test -count=1 -v
  ./internal/server/...` mit `DATABASE_URL` als `kmuhub_app`: **207 PASS, 0 SKIP, 0 FAIL**
  (196 vorher + 11 neue: 5 Erfolgspfade als Subtests von `TestPermissionAuditEvents_DB` plus
  `TestPermissionAuditEvents_DB_RejectedWriteLeavesNoEvent`, alle real gegen die DB gelaufen, nicht
  uebersprungen). `go test ./internal/gateway/` nicht erneut gelaufen — keine Route, kein
  `.proto`, kein `openapi.yaml` in diesem Diff, der Drift-Test ist also nicht beruehrt. | migration
  n.a. — `audit_log.action` ist ein freies VARCHAR(100) ohne CHECK/ENUM (gegen die Migrationen
  verifiziert), die neue Taxonomie braucht kein Schema. | rls-smoke n.a. — keine Tabelle/Policy
  angefasst, `audit_log`s RLS (000120) und Append-Only-Trigger (000222) unveraendert; die
  Nicht-Loeschbarkeit selbst ist im Test sichtbar geworden (siehe Entscheidung 5) statt eigens
  geprueft zu werden.
- verify vorgaenger: sauber. `95ce32f0` (p1b-guardrails) gegen die acht Klassen geprueft — kein
  gRPC-Bypass (Guardrails sitzen im `auth.Service`, keine Handler-Logik), kein Stub, kein `.proto`
  im Diff (reine Service-Aenderung, keine RPC-Signatur neu), kein neuer `RequirePermission`-Guard
  (wiederverwendet `roles:manage`/`admin:role:assign`), Tenant-Scoping von
  `CountRoleAdminsExcluding` ueber den `users`-Join korrekt und im Code kommentiert, `openapi.yaml`
  nur um Fehlerfall-Doku auf bestehenden Routen erweitert (keine neue Route, Drift-Test also nicht
  betroffen). `go build -p 2` auth/server/gateway lief zusaetzlich gruen als Sanity-Check.
- offen:
  - **`p1b-guardrail-tests` (naechste Unit)** deckt laut Backlog dieselben sieben Faelle nochmal
    explizit ab (last-admin, Selbst-Aussperrung, Escalation, Preset-Immutability x4, Custom-Limit,
    Tenant-Isolation) — die Guardrail-Logik selbst ist durch `guardrails_db_test.go` aus Iteration 7
    schon abgedeckt, diese Unit muesste also pruefen, ob echte Luecken bleiben, statt blind zu
    duplizieren.
  - `SetRolePermissions` (Grant-Aenderung an einer Rolle) schreibt bewusst kein Audit-Event — war
    nicht in den fuenf geforderten Aenderungsarten. Falls das GoBD-relevant werden soll (Rechte
    geaendert ohne Rollen-Rename), ist das ein separater Scope-Punkt fuer Luke, keine Nachlaessigkeit.
  - `rbac-format.ts`/`RBAC_ERROR_CODES`-Luecke aus Iteration 7 weiterhin offen (self_lockout,
    privilege_escalation fehlen im FE-Katalog).
  - `desktop/src/renderer/src/api/types.ts` weiterhin nicht regeneriert (Befund aus Iteration 6).

## Iteration 9 — p1b-guardrail-tests — done — 2026-08-02 22:05
- commit: <wird nach dem Commit unten ergaenzt>
- gebaut: Die sieben im Backlog geforderten Faelle (last-admin, Selbst-Aussperrung, Escalation
  inkl. scope-Aufwertung, Preset-Immutability, Custom-Limit 20, Tenant-Isolation der
  Rollenzuweisung) haben sich bei der Recherche als bereits vollstaendig durch echte DB-Tests
  abgedeckt herausgestellt — verteilt ueber `guardrails_db_test.go` (Iteration 7),
  `roles_admin_db_test.go` und `user_roles_db_test.go` (Iterationen 3-6). Blind duplizieren haette
  nur Testmasse ohne neuen Erkenntniswert erzeugt.
  Stattdessen die tatsaechliche Luecke geschlossen: `TestMapError` in
  `backend/internal/server/grpc_test.go` deckte `ErrRoleNameExists`, `ErrRoleLimitReached`,
  `ErrBaseRoleNotFound`, `ErrRolePresetImmutable`, `ErrRoleHasMembers` ab, aber NICHT die drei
  eigentlichen Guardrail-Sentinels `ErrLastAdmin`/`ErrSelfLockout`/`ErrPrivilegeEscalation` — obwohl
  `mapError` (grpc.go:1101-1106) alle drei behandelt. Ein vertauschter Case in diesem Switch (z.B.
  `ErrSelfLockout` faellt auf `codes.NotFound`) waere von keinem bestehenden Test bemerkt worden:
  die Service-Tests pruefen nur, dass `auth.Service` den richtigen Go-Fehler zurueckgibt, nicht dass
  `mapError` ihn auf den richtigen gRPC-Code abbildet und `respondGRPCError` daraus den richtigen
  HTTP-Code macht. Drei Tabellenzeilen ergaenzt (last_admin -> FailedPrecondition/409, self_lockout
  -> FailedPrecondition/409, privilege_escalation -> PermissionDenied/403), keine neue Test-Infra.
- entscheidungen:
  1. **Kein neuer Gateway-Mock-Harness.** `route_auth_test.go` hat kein Fake fuer
     `authv1.AuthServiceClient` — die bestehenden Route-Tests pruefen nur ServiceUnavailable/
     Validierung, nie einen simulierten gRPC-Fehler durchs HTTP. Ein solcher Harness allein fuer
     drei Fehlerfaelle waere die groessere Abstraktion fuer den kleineren Nutzen gewesen — der
     Luecken-Schluss sitzt bewusst auf der Ebene, auf der er entstehen kann (der `mapError`-Switch),
     nicht eine Ebene hoeher noch einmal neu gebaut. `grpcStatusToHTTP` selbst ist reiner, bereits
     durch bestehende Gateway-Tests belegter Code und war nicht die Luecke.
  2. **Keine weiteren DB-Tests ergaenzt.** Alle sieben fachlichen Faelle liefen beim Nachmessen real
     und ungeskippt; ein achter, ergaenzender DB-Test haette nur eine bereits bewiesene Eigenschaft
     ein zweites Mal bewiesen.
- gate: build ok (`go build -p 2` auth/gateway/server/cmd/auth/cmd/gateway) | vet ok | lint ok
  (golangci-lint, 0 issues ueber auth+gateway+server) | test ok — `go test -count=1 -v
  ./internal/auth/...` als `kmuhub_app`: **158 PASS, 0 SKIP, 0 FAIL**. `go test -count=1 -v
  ./internal/server/...`: **207 PASS, 0 SKIP, 0 FAIL**, darunter die drei neuen
  `TestMapError/last_admin`, `/self_lockout`, `/privilege_escalation`. `go test -count=1
  ./internal/gateway/` (inkl. `TestOpenAPIRouteDrift`) gruen. | migration n.a. — keine Tabelle,
  keine Route, kein `.proto` angefasst. | rls-smoke n.a. — keine Policy beruehrt.
- verify vorgaenger: sauber. `aa5fcca8` (p1b-audit-events) gegen die acht Klassen geprueft — kein
  gRPC-Bypass (AuthGRPCServer haelt `auditService` direkt, weil Auth- und SecurityGRPCServer im
  selben Prozess co-located sind, kein Netzwerk-Hop noetig; keine Gateway-Umgehung), kein Stub,
  kein `.proto` im Diff, kein neuer `RequirePermission`-Guard, keine neue Tabelle, Events entstehen
  nachweislich nur NACH erfolgreichem Schreiben (Test `..._RejectedWriteLeavesNoEvent`), Test seedet
  eigenen Tenant/Actor statt TenantA/B. `go build -p 2` zusaetzlich als Sanity-Check gruen.
- offen:
  - Block A (RBAC Welle 1b) ist damit **vollstaendig abgearbeitet** — naechste Unit laut Backlog ist
    `g-rls-tenant-id-ohne-policy` (Block B, Sicherheit/RLS-Reste, model: opus).
  - `SetRolePermissions` schreibt weiterhin kein eigenes Audit-Event (siehe Iteration 8, offener
    Punkt fuer Luke — kein Blocker).
  - `rbac-format.ts`/`RBAC_ERROR_CODES`-Luecke im FE (self_lockout, privilege_escalation fehlen im
    Katalog) weiterhin offen — FE-seitig, dieser Loop fasst das Frontend nicht an.
  - `desktop/src/renderer/src/api/types.ts` weiterhin nicht regeneriert (Befund aus Iteration 6).
