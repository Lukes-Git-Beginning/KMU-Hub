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
- commit: 26388dae
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

## Iteration 10 — g-rls-tenant-id-ohne-policy — done — 2026-08-02 22:40
- commit: bfb89c8b
- gebaut: Migration `000269_rls_email_contact_links_and_plugin_rules` setzt RLS auf die drei
  Tabellen, die `tenant_id` tragen, aber weder RLS noch Policy hatten
  (`email_contact_links`, `validation_rules`, `workflow_rules`), plus zwei Isolationstests und
  einen Schreibpfad-Fix im plugin-gRPC-Server.
  Der Befund ist groesser als "Policy fehlt": `ValidationRuleRepository.GetByID/.Update/.Delete`
  und die `workflow_rules`-Entsprechungen laufen samtlich als `WHERE id = $1` — **ohne**
  Tenant-Praedikat. Wer eine Regel-UUID kannte, konnte bis eben die Regel eines fremden Tenants
  lesen, aendern und loeschen. Der Service darueber (`plugin/service.go`) reicht die ID
  ungefiltert durch. RLS schliesst das an der einzigen Stelle, an der es fuer alle Aufrufer
  zugleich zu schliessen ist; die Repos bleiben unveraendert, weil ein nachtraeglich
  eingebautes Praedikat je Methode genau der groessere Diff mit der kleineren Wirkung waere.
  `email_contact_links.tenant_id` war seit Migration 000110 **nullable und nie gebackfillt**.
  Eine Policy auf einer NULL-Spalte macht die Tabelle unsichtbar statt sicher (Phantom-404), also
  erst Backfill aus `email_messages.tenant_id` (Join ueber `message_id`, das NOT NULL + FK mit
  ON DELETE CASCADE ist, und `email_messages.tenant_id` ist selbst NOT NULL — der Backfill ist
  damit vollstaendig), dann `SET NOT NULL`, dann die Policy.
- entscheidungen:
  1. **Kein DELETE-Fallback fuer nicht aufloesbare Zeilen.** Die naheliegende Zeile
     `DELETE FROM email_contact_links WHERE tenant_id IS NULL` haette die Migration auf jeder
     Datenbank durchlaufen lassen. Genau das ist der Fehler: traegt die FK-Annahme auf Prod nicht,
     soll die Migration stehenbleiben und nicht stillschweigend Verknuepfungen loeschen. Lokal
     ist die Tabelle leer (0 Zeilen), auf Prod entscheidet der Backfill; schlaegt `SET NOT NULL`
     dort fehl, ist das die gewuenschte Bremse.
  2. **Schreibpfad-Fix im Service, nicht nur RLS.** `plugin_grpc.go` nahm die `tenant_id` fuer
     `CreateValidationRule`, `ListValidationRules`, `CreateWorkflowRule`, `ListWorkflowRules` und
     `ApplyIndustryTemplate` **aus dem Request-Body** — die Datei benutzte
     `middleware.GetTenantID` an keiner einzigen Stelle. Ueber HTTP war das gedeckt (das Gateway
     fuellt das Feld aus dem JWT, `route_plugin.go`), am gRPC-Port aber nicht. Neuer Helper
     `ruleTenant(ctx)` liest den propagierten `x-tenant-id`. Ohne ihn waere der Angriffsversuch
     nach dieser Migration zwar geblockt, aber als undurchsichtiger DB-Fehler tief im Repository
     statt als `InvalidArgument` an der Grenze. Das proto-Feld bleibt (Kompatibilitaet), es wird
     nur nicht mehr geglaubt.
  3. **Bewusst nicht angefasst:** die uebrigen Handler derselben Datei (KV-Store, Installationen,
     Manifeste, `ValidateEntity`, `ExecuteHooks`) nehmen den Tenant weiterhin aus dem Body. Ihre
     Tabellen stehen seit Migration 000126 unter RLS, der Vektor ist also gedeckt; ein Umbau
     aller Handler ist eine eigene Unit und haette den Diff dieser verwaessert. **Offener Punkt
     unten.**
  4. **Tests gegen die echten Repo-Methoden, nicht ueber `SeedRow`.** Ein seed-basierter Test
     haette nur bewiesen, dass ein handgeschriebenes INSERT die Policy respektiert — die
     eigentliche Luecke (ungescoptes `GetByID`/`Delete`) waere unsichtbar geblieben. Der Test
     ruft deshalb `repo.GetByID` und `repo.Delete` als fremder Tenant auf und prueft, dass die
     Zeile danach noch steht.
- gate: build ok (`go build -p 2` server/plugin/email + cmd/plugin,email,gateway) | vet ok |
  lint ok (golangci-lint, **0 issues** ueber server+plugin+email) | test ok — `go test -count=1`
  ueber `./internal/plugin/... ./internal/email/... ./internal/server/...` **14 Pakete gruen**,
  darunter die drei neuen `TestTenantIsolation_ValidationRules_DB`,
  `_WorkflowRules_DB`, `_EmailContactLinks_DB` (real gelaufen als `kmuhub_app`, **0 Skips**);
  `go test -count=1 ./internal/gateway/` inkl. `TestOpenAPIRouteDrift` gruen. |
  migration: up/down/up gruen (268 -> 269 -> 268 -> 269), danach per `pg_class`/`pg_policies`
  verifiziert: alle drei `rowsec=true forced=true` mit Policy
  `tenant_id = current_tenant_id() OR is_system_context()`, `email_contact_links.tenant_id`
  `nullable=NO`. | rls-smoke: durch die drei Isolationstests abgedeckt (eigener Tenant sieht 1,
  fremder 0; fremdes DELETE aendert nichts; INSERT auf fremden Tenant wird von WITH CHECK
  abgelehnt).
- verify vorgaenger: sauber. `26388dae` (p1b-guardrail-tests) gegen die acht Klassen geprueft —
  reine Testergaenzung (10 Zeilen in `grpc_test.go`), kein Produktionscode, kein `.proto`, keine
  Route, keine Tabelle, kein neuer Guard. Die drei ergaenzten Tabellenzeilen decken sich mit dem
  tatsaechlichen `mapError`-Switch (`grpc.go:1101-1106`) — nachgelesen, nicht geglaubt.
- offen:
  - **Neuer Fund, eigene Unit wert:** `backend/internal/server/plugin_grpc.go` liest den Tenant in
    allen uebrigen Handlern weiter aus dem Request-Body (`req.GetTenantId()`) statt aus dem
    Context. Nach dieser Iteration ist die Datei uneinheitlich — die Rule-Pfade lesen den Context,
    der Rest nicht. Kandidat fuer eine kleine Aufraeum-Unit in Lauf 5.
  - Testfixture-Fallstrick, den der naechste Isolationstest wissen sollte: `t.Cleanup` laeuft
    **nach** allen `defer`s der Testfunktion. Ein `defer pool.Close()` schliesst den Pool also,
    bevor per `t.Cleanup` registrierte Fixture-Cleanups laufen — sie scheitern still mit
    "closed pool" und lassen die FK-Kette in der DB stehen (im ersten Lauf genau so passiert,
    Reste manuell entfernt). Loesung im contactlink-Test: Pool ebenfalls per `t.Cleanup`
    schliessen, als erstes registriert (LIFO -> laeuft zuletzt).
  - Naechste Unit laut Backlog: `g-rls-custom-field-values` (Block B, opus) — vier
    `*_custom_field_values`-Tabellen ohne eigene `tenant_id`, Absicherung ueber
    `enable_tenant_rls_via_join()`.
  - `SetRolePermissions` ohne Audit-Event, `rbac-format.ts`-Katalogluecke und die nicht
    regenerierte `desktop/src/renderer/src/api/types.ts` bleiben unveraendert offen (FE-seitig
    bzw. Scope-Frage fuer Luke).

## Iteration 11 — g-rls-custom-field-values — done — 2026-08-02 22:20
- commit: 54d1ef7d
- gebaut: Migration `000270_rls_crm_custom_field_values` setzt die vier CRM-Custom-Field-Value-
  Tabellen (`contact_`/`company_`/`deal_`/`activity_custom_field_values`) ueber
  `enable_tenant_rls_via_join()` unter RLS — Join jeweils auf die Eltern-Entitaet.
  Dazu `internal/crm/customfield/value_tenant_isolation_test.go` (external test package) mit einem
  tabellengetriebenen Isolationstest ueber alle vier Repos und einem Merge-Regressionstest.
  **Kein Go-Produktionscode geaendert** — das ist der Punkt der gewaehlten Loesung.
- entscheidungen:
  1. **Join-Policy statt eigener `tenant_id`-Spalte** — bewusst gegen den Praezedenzfall aus
     Migration 000126 (die fuer `ticket_messages` die Spalte ergaenzt und den Join-Helper
     ausdruecklich ablehnt). 000126 argumentiert mit Query-Volumen (Plugin-Execution-Log wird in
     Masse gescannt). Hier trifft das nicht: die vier Tabellen werden ausschliesslich ueber die
     Eltern-ID gelesen, und die ist die fuehrende Spalte ihres Primaerschluessels. Der teurere Weg
     waere die Spalte gewesen: `NOT NULL` haette alle vier Schreibpfade in Go erzwungen,
     inklusive des `INSERT ... SELECT`-Merge in `contact/postgres_repository.go:717` — und ein
     dabei uebersehener Pfad bricht Schreiben komplett (die dokumentierte NULL-tenant_id-Falle
     dieses Repos). Migration 000118 nennt den Join-Helper genau dafuer den vorgesehenen Fallback.
     Er wurde damit **zum ersten Mal produktiv benutzt** — bis jetzt stand er nur definiert im
     Schema.
  2. **`task_custom_field_values` gegengeprueft, nicht geglaubt.** Der Backlog vermutete sie
     ausserhalb der Liste; `pg_class` bestaetigt: `relrowsecurity=t`, eigene `tenant_id`, eine
     Policy. Zu Recht nicht angefasst.
  3. **Negativprobe statt Vertrauen ins gruene Ergebnis.** Migration einmal zurueckgerollt und den
     Test erneut gefahren: alle vier Faelle fallen mit `cross-tenant read ... leaked 1 row(s)`.
     Damit ist belegt, dass der Test die Luecke wirklich misst und nicht nur mitlaeuft.
  4. **Merge-Regressionstest zusaetzlich.** `MergeInto` kopiert Custom-Field-Werte mit
     `INSERT INTO contact_custom_field_values ... SELECT ... FROM contact_custom_field_values`
     in einer Transaktion — nach dieser Migration filtert die Policy dort **beide** Seiten.
     Beide Kontakte gehoeren demselben Tenant, es muss also weiter kopieren; ein still leerer
     Merge waere genau die Phantom-Form, die dieses Repo schon produziert hat. Test beweist, dass
     der Wert ankommt.
- befund (real, kein Blocker): die eigentliche Luecke war groesser als "RLS fehlt". **Saemtliche
  zwoelf Zugriffsstellen filtern nur auf die Eltern-ID** (`WHERE cfv.contact_id = $1`), ohne
  jeden Tenant-Praedikat — Lesen, Batch-Lesen und der Upsert. Wer eine fremde Kontakt-UUID kannte,
  konnte deren Custom-Field-Werte lesen **und ueberschreiben**. Deshalb geht der Test durch die
  echten Repo-Methoden und nicht ueber `SeedRow`: ein Seed-Test haette nur bewiesen, dass ein
  handgeschriebenes INSERT die Policy respektiert.
- query-plan (der im Backlog erbetene Check): bei 12 Kontakten waehlt der Planner fuer die
  Policy-Subquery einen **hashed SubPlan mit Seq Scan auf `contacts`** — bei dieser Groesse die
  billigere Wahl. Der korrelierte Plan existiert und ist erreichbar: mit `enable_seqscan=off`
  zeigt derselbe Query `Index Scan using contacts_pkey ... Index Cond: (id = cfv.contact_id)`.
  Der Planner kippt also kostenbasiert auf den PK-Lookup, sobald `contacts` waechst. Kein
  Handlungsbedarf, aber der Beleg gehoert hierher statt in eine Behauptung.
- gate: build ok (`go build -p 2` ueber `./internal/crm/... ./internal/gateway/... ./cmd/crm/...
  ./cmd/gateway/...`) | vet ok | lint ok (golangci-lint, **0 issues** ueber `./internal/crm/...`) |
  test ok — `go test -count=1 ./internal/crm/...` **12 Pakete gruen, 0 Skips**, darunter die neuen
  `TestTenantIsolation_CRMCustomFieldValues_DB` (4 Subtests) und
  `TestMergeInto_CarriesCustomFieldValues_DB`, real gelaufen als `kmuhub_app`;
  `go test -count=1 ./internal/gateway/ ./internal/server/...` gruen. |
  migration: up/down/up gruen (269 -> 270 -> 269 -> 270), danach per `pg_class`/`pg_policies`
  verifiziert: alle vier `rowsec=true forced=true` mit Policy
  `EXISTS (SELECT 1 FROM <parent> p WHERE p.id = <child>.<fk> AND (p.tenant_id =
  current_tenant_id() OR is_system_context()))`. | rls-smoke: durch die Isolationstests
  abgedeckt (eigener Tenant 1 Zeile, fremder 0; fremder Upsert laesst den Wert des Eigentuemers
  unveraendert).
- verify vorgaenger: sauber. `bfb89c8b` (g-rls-tenant-id-ohne-policy) gegen die acht Klassen
  geprueft. Keine Route, kein `.proto`, kein neuer Guard. Klasse 5 gezielt nachgezogen, weil die
  Migration `email_contact_links.tenant_id` auf `NOT NULL` hebt: der einzige Schreibpfad ist
  `server/email_grpc.go:843 LinkEmailToContact`, er zieht den Tenant aus
  `middleware.GetTenantID(ctx)` und reicht ihn an `contactlink/repository.go:33` durch; die
  Lesepfade sind zusaetzlich per `tenant_id = $2` gescopt. Kein Pfad kann die Spalte leer lassen.
- offen:
  - Der Join-Helper ist mit dieser Migration erstmals im Einsatz. Wenn `g-rls-regression-guard`
    (Block B) den Schema-Scan baut, sollte er Join-geschuetzte Tabellen genauso als "geschuetzt"
    zaehlen wie spalten-geschuetzte — beide zeigen `relrowsecurity=true`, das reicht als Kriterium.
  - `SeedRow`/`CleanupRow` in `testutil` setzen eine `id`-Spalte voraus (`RETURNING id`,
    `DELETE ... WHERE id = $1`). Fuer Tabellen mit zusammengesetztem Primaerschluessel — wie diese
    vier — sind sie unbrauchbar; der Test zaehlt deshalb selbst. Falls weitere Units auf solche
    Tabellen treffen, lohnt ein Helper mit frei waehlbarem Praedikat.
  - Naechste Unit laut Backlog: `g-rls-events-partition` (Block B, opus) — `events` ist
    partitioniert, ohne `tenant_id` und ohne Policy; Entscheidung zwischen Spalte+Backfill (A)
    und begruendetem Allowlist-Eintrag (B).
  - Aus Iteration 10 unveraendert offen: `server/plugin_grpc.go` liest den Tenant in den uebrigen
    Handlern weiter aus dem Request-Body; `SetRolePermissions` ohne Audit-Event;
    `rbac-format.ts`-Katalogluecke; nicht regenerierte `desktop/.../api/types.ts`.

## Iteration 12 — g-rls-events-partition — done — 2026-08-02 23:05
- commit: dd99f2d7
- entscheidung (A), wie vom Backlog verlangt begruendet: **`events` bekommt `tenant_id` + RLS**, kein
  Allowlist-Eintrag. Der Nachweis fuer (B) waere gewesen, dass die Tabelle ein rein technischer
  Event-Bus ohne fachliche Nutzdaten ist. Er scheitert nicht am `payload` — dort steht heute
  tatsaechlich nur `{"calendar_id","event_id"}` (einzige befuellende Stelle:
  `work/event/service.go:684`) — sondern an drei anderen Punkten:
  1. **Der Tenant ist am Schreibpfad bekannt und wird weggeworfen.** `models.EventPayload` fuehrt
     `TenantID` (`models/event.go:42`), jeder Emitter fuellt sie,
     `notification/service.go:52` baut daraus die Zeile und laesst das Feld weg.
  2. **Daraus folgt ein Bestandsbug, kein theoretisches Risiko:** `EventBus.ProcessBacklog`
     (`event/bus.go:157`) rekonstruiert die `EventPayload` aus der gespeicherten Zeile und kann den
     Tenant nicht wiederherstellen. Jedes nach einem Neustart nachgeholte Event erreichte
     `preference.Evaluate` mit `uuid.Nil`.
  3. `actor_id` + `resource_id` sind konstruktionsbedingt tenant-gebunden — genau die
     Korrelationsdaten, die RLS trennen soll. Und die beiden anderen partitionierten Log-Tabellen
     (`automation_executions`, `dialer_call_events`) haben seit 000242 beide `tenant_id` + Policy;
     `events` war der Ausreisser.
- gebaut:
  - `000271_rls_events` — `ADD COLUMN tenant_id`, dreistufiger Backfill (1. `actor_id` ->
    `users.tenant_id`; 2. aktorlose Zeilen nur, wenn genau EIN Tenant existiert — das ist die
    Prod-Form, der Guard verhindert Raten auf einer Multi-Tenant-DB; 3. Rest loeschen, ephemeres
    90-Tage-Log, ohne Tenant ohnehin nicht verarbeitbar), `SET NOT NULL`,
    `idx_events_tenant_id`, `CALL enable_tenant_rls('events')`.
  - `models.Event.TenantID`; Repository-INSERT/SELECT um die Spalte erweitert.
  - `notification/service.go`: `evt.TenantID` **und** `notif.TenantID` aus `payload.TenantID`.
  - `event/bus.go`: `dispatch` stempelt den Tenant in den Handler-Context; `ProcessBacklog` setzt
    `payload.TenantID` aus der Zeile.
  - Tests: 3 DB-Tests (`internal/notification/notification/event_tenant_isolation_test.go`) +
    3 Bus-Tests.
- BEFUND, der die Umsetzung bestimmt hat (verifiziert, nicht vermutet): **der notification-Worker
  konnte seit dem Scharfschalten von RLS gar keine Notification mehr schreiben.** Die Handler
  laufen auf dem Listener-Background-Context (`bus.dispatch`), der weder Tenant noch System-Kontext
  traegt; `PrepareConn` laesst die GUCs dann leer, `current_tenant_id()` ist NULL und die Policy auf
  `notifications` (000122) weist den INSERT ab. `ProcessEvent` loggt den Fehler und macht weiter —
  der Ausfall ist also still. Belegt per psql als `kmuhub_app`:
  `INSERT INTO notifications (...) -> ERROR: new row violates row-level security policy`.
  Deshalb steht der Fix im **Bus** und nicht in `ProcessEvent`: dort greift er fuer jeden Handler,
  auch fuer den zweiten registrierten Konsumenten (`inboxConsumer.HandleEvent`, `cmd/notification/
  main.go:211`), der dasselbe Problem hat. Haette ich nur `events` unter RLS gestellt, waere
  derselbe stille Bruch ein zweites Mal entstanden.
- system-kontext, bewusst asymmetrisch: `ListUnprocessedEvents` und `MarkEventProcessed` laufen als
  System (`sysctx.With`) — der Catch-up ist per Definition tenant-uebergreifend, und ein Event, das
  nicht als verarbeitet markiert werden kann, wird bei jedem Neustart erneut abgespielt.
  `CreateEvent` laeuft **nicht** als System: nur so prueft `WITH CHECK` den Insert wirklich, und ein
  fuer einen fremden Tenant gestempeltes Event wird abgewiesen statt gespeichert (eigener Testfall).
- negativprobe, zweistufig — die erste Fassung war zu schwach und ist deshalb nachgeschaerft
  worden: mit `migrate down` fielen die Tests zwar, aber an *"column tenant_id does not exist"*.
  Ein Test, den jeder beliebige Fehler zufriedenstellt, ueberlebt das Verschwinden der Policy. Jetzt
  prueft `assertRLSRejected` auf **SQLSTATE 42501**. Scharfe Probe (Spalte bleibt, nur
  `DISABLE ROW LEVEL SECURITY`): `TestTenantIsolation_Events_DB` meldet
  `expected 0 row(s), got 1`, `TestEvents_WriteWithoutTenantContext_Rejected_DB` meldet
  `succeeded; the RLS policy is not in force`. Danach RLS wieder aktiviert und `pg_class`
  gegengeprueft (`rowsec=t forced=t`, 0 Restzeilen).
- gate: build ok (`go build -p 2 ./...`, gesamter Baum) | vet ok | lint ok (golangci-lint,
  **0 issues** ueber `./internal/notification/... ./internal/models/... ./cmd/notification/...`) |
  test ok — `go test -count=1 ./internal/notification/... ./internal/models/... ./internal/gateway/
  ./internal/server/...` gruen, die **6 neuen Tests real gelaufen (0 Skips**, per `-v` geprueft),
  DB-Tests als `kmuhub_app`. | migration: up/down/up gruen (270 -> 271 -> 270 -> 271), danach
  `relrowsecurity=t relforcerowsecurity=t`, Policy
  `((tenant_id = current_tenant_id()) OR is_system_context())`, `tenant_id` NOT NULL.
  | openapi: keine Route beruehrt, `TestOpenAPIRouteDrift` als Teil von `./internal/gateway/` gruen.
- stolperstein fuer die naechste Migration: `min(uuid)` gibt es in Postgres nicht — der erste
  Anlauf starb daran (`schema_migrations` blieb `271 dirty`, die Transaktion selbst war sauber
  zurueckgerollt). Recovery: `migrate force 270`, dann regulaer hoch. Fuer "der einzige Tenant"
  also `SELECT count(*)` und `SELECT id` getrennt.
- verify vorgaenger: sauber. `54d1ef7d` (g-rls-custom-field-values) gegen die Fehlerklassen geprueft:
  keine Route, kein `.proto`, kein neuer Guard, kein Stub — die Migration ruft nur den Join-Helper,
  der Rest ist Test. Down-Migration ist symmetrisch zur Up (Policy + FORCE + ENABLE je Tabelle).
  Read-Seite ist durch die Policy selbst abgedeckt, nicht durch handgeschriebene Praedikate.
- offen:
  - **`docs/ARCHITECTURE.md` ergaenzt** um einen Absatz, warum `events` nicht mehr system-global ist.
    Der Kopfkommentar von `000242` behauptet weiterhin "events — NO tenant_id, NO RLS (system-level
    event bus)"; historische Migrationen bleiben unangetastet, aber wer dort liest, liest Veraltetes.
  - **`sentinelTenantID` in `notification/postgres_repository.go:27`** ist jetzt totes Netz: er fing
    genau den `notif.TenantID == uuid.Nil`-Fall ab, dessen Ursache diese Iteration beseitigt.
    Entfernen erst, wenn geprueft ist, dass kein anderer Aufrufer von `Create` ohne Tenant kommt —
    und Bestandszeilen unter dem Sentinel-Tenant `...0001` brauchen dann eine Entscheidung.
  - **`MarkEventProcessed` filtert nur auf `id`**, der PK ist aber `(id, created_at)`: das UPDATE
    scannt jede Partition. Bei 15 Monatspartitionen heute egal, mit `created_at` im WHERE waere es
    ein Partition-Prune. Kein Sicherheitsproblem, eine Zeile Arbeit.
  - Naechste Unit laut Backlog: `g-rls-allowlist-audit` (Block B, sonnet) — elf Tabellen ohne
    `tenant_id` und ohne RLS gegen die Vier-Eintraege-Allowlist stellen. `events` ist dort **nicht**
    zu ergaenzen, es ist ab jetzt geschuetzt.
  - Aus Iteration 10/11 unveraendert offen: `server/plugin_grpc.go` liest den Tenant in den uebrigen
    Handlern weiter aus dem Request-Body; `SetRolePermissions` ohne Audit-Event;
    `rbac-format.ts`-Katalogluecke; nicht regenerierte `desktop/.../api/types.ts`; `SeedRow`/
    `CleanupRow` brauchen eine `id`-Spalte (fuer Tabellen mit zusammengesetztem PK unbrauchbar).

## Iteration 13 — g-rls-allowlist-audit — done — 2026-08-02 22:30

- commit: a732b743
- gebaut:
  - `000272_rls_refresh_tokens_and_plugin_permissions` — `refresh_tokens` bekommt `tenant_id NOT
    NULL` (Backfill per Join ueber `user_id -> users.tenant_id`, FK garantiert 0 Waisen, kein
    DO-Block noetig — anders als bei `events`) + `enable_tenant_rls('refresh_tokens')`. Neuer Index
    `idx_refresh_tokens_tenant_id`. `plugin_permissions` bekommt
    `enable_tenant_rls_via_join('plugin_permissions', 'plugin_installations', 'installation_id')`.
  - `models.RefreshToken.TenantID`; `auth/postgres_repository.go` StoreRefreshToken/
    GetRefreshTokenByHash um die Spalte erweitert; `auth/service.go` `createTokenPair` setzt
    `TenantID: user.TenantID` beim Ausstellen — exakt das Muster von `PasswordResetToken.TenantID`
    (`RequestPasswordReset`), das schon vor dieser Iteration existierte und als Vorlage diente.
  - Tests: `internal/auth/rls_refresh_tokens_test.go` (4 Faelle: Tenant-B sieht nichts, System-
    Kontext sieht alles — deckt die bestehenden `sysctx.With()`-Aufrufe in RefreshToken/Logout ab —,
    Cross-Tenant-Revoke 0 Zeilen betroffen, Cross-Tenant-gestempelter Write via echtem
    `StoreRefreshToken` mit SQLSTATE 42501 abgelehnt) + `internal/plugin/repository/
    permission_rls_test.go` (2 Faelle: Tenant-Trennung beim Lesen, Cross-Tenant-Grant via echtem
    `Grant()` abgelehnt).
  - `docs/ARCHITECTURE.md`: drei neue Allowlist-Zeilen (`automation_templates`, `event_types`,
    `public_holidays` — alle verifiziert schreibfrei zur Laufzeit bzw. tenant-invariant), ein Absatz
    zur Schliessung von refresh_tokens/plugin_permissions, ein Absatz "Offener Befund" fuer die
    fuenf verbleibenden Tabellen (siehe unten).
- rechercheergebnis (der eigentliche Kern dieser Iteration): die elf Tabellen aus dem Backlog-Scope
  zerfallen nicht in zwei, sondern drei Gruppen:
  1. **RLS-Luecke, jetzt geschlossen:** `refresh_tokens`, `plugin_permissions` (oben).
  2. **Echte Katalog-/Seed-Daten, jetzt allowlisted:** `automation_templates` (nur
     `TemplateRegistry.SeedToDatabase()` beim Start, keine HTTP-Route schreibt), `event_types` (nur
     Migrations-Seeds 000027/000048, kein Laufzeit-Writer), `public_holidays` (nur
     `SeedHolidays()` aus der externen Nager.Date-API, Upsert auf `date,country_code,name` — jeder
     Tenant, der den Sync ausloest, schreibt dieselben Zeilen, keine Divergenz moeglich).
     `industry_templates` stand schon in der Allowlist, gegengeprueft: weiterhin korrekt.
  3. **Neuer Fund, nicht geloest:** `dashboard_defaults`, `presence_config`, `two_factor_policy`,
     `storage_quotas`, `plugin_manifests` sind eine dritte Kategorie, die die urspruengliche
     Zwei-Wege-Frage (RLS oder Allowlist) schlicht nicht beantwortet. Alle fuenf sind eine einzige
     globale Zeile (`LIMIT 1` bzw. `UNIQUE(role)`-Lookup) bzw. ein globaler Katalog, aenderbar ueber
     eine Route, die nur `RequireRole("admin")`/`RequirePermission(...,"write")` prueft — ALSO "ist
     irgendein Tenant-Admin", nicht "ist Admin des richtigen Tenants". Verifiziert Zeile fuer Zeile:
     `PUT /admin/dashboard/defaults/{role}` (dashboard_service.go:144, `UNIQUE(role)`), `PUT
     /presence/config` (route_video.go:136, `settings:write`, UPDATE ohne WHERE auf der einen
     Zeile), `PUT /2fa/policy` (route_auth.go:90, `RequireRole("admin")`, `UNIQUE(role_name)`),
     `IncrementUsedBytes`/`DecrementUsedBytes` (chat/file/postgres_repository.go:186ff, UPDATE ohne
     WHERE), `POST /api/v1/plugins/manifests` (route_plugin.go:53, `RequireRole("admin")`, kein
     `tenant_id`, `GET /manifests` ohne Filter — nur `plugin_type=wasm` ist durch den Feature-Flag
     gesondert blockiert, `config`-Manifeste nicht). Weder RLS (kein `tenant_id` zum Filtern) noch
     Allowlist (der Inhalt ist echt admin-mutable, keine Seed-Daten) beantwortet das ehrlich — es
     braucht `tenant_id` + Backfill + tenant-gescopte Unique-Constraints + einen Provisioning-
     Schritt fuer neue Tenants (ohne den liefert der Read nach RLS ein leeres Ergebnis statt eines
     Fallbacks). Groesser als eine Iteration, deshalb bewusst NICHT in 000272 mit hineingezogen.
     Ausgelagert in eine neue, direkt danach eingefuegte Backlog-Unit `g-rls-tenant-scoped-admin-
     writes` (model: opus). `g-rls-regression-guard` haengt jetzt an dieser neuen Unit statt an
     `g-rls-allowlist-audit`, sonst wuerde der Guard live gehen, bevor die Allowlist wirklich
     vollstaendig ist.
  Die urspruengliche Vermutung im Backlog-scope-Text, `dashboard_defaults` sei ein "starker
  Allowlist-Kandidat", hat sich beim Nachpruefen NICHT bestaetigt — der Bewertungsmassstab war
  richtig (Katalog/Seed -> Allowlist, pro-Tenant-divergent -> absichern), die Vorab-Einschaetzung
  dieser einen Tabelle war es nicht: sie SOLLTE divergieren koennen (ist admin-editierbar), kann es
  aber wegen der fehlenden Tenant-Spalte nicht sauber, und genau das ist der Bug.
- gate: build ok (`go build -p 2 ./internal/auth/... ./internal/plugin/... ./internal/models/...
  ./internal/gateway/... ./internal/testutil/... ./cmd/auth/... ./cmd/plugin/... ./cmd/gateway/...`)
  | vet ok | lint ok (golangci-lint, 0 issues) | test ok — `go test -count=1 ./internal/auth/...`
  162 PASS / 0 SKIP, `./internal/plugin/...` gruen (4 neue Tests real gelaufen), `./internal/
  gateway/` gruen (TestOpenAPIRouteDrift unberuehrt, keine Route angefasst) | migration: up/down/up
  gruen (272 -> 271 -> 272), danach `relrowsecurity=t relforcerowsecurity=t` auf beiden Tabellen,
  `tenant_id` auf refresh_tokens NOT NULL | rls-smoke: die vier Cross-Tenant-Faelle sind die
  Isolationstests selbst (kein separates manuelles psql noetig, siehe oben).
- stolperstein: `t.Cleanup` in einem Test-Helper (`seedInstallation`) plus `defer pool.Close()` in
  der aufrufenden Testfunktion feuern in der falschen Reihenfolge — `defer` laeuft beim Return der
  Funktion, `t.Cleanup` erst danach, also war der Pool beim Aufraeumen schon zu (stille
  "closed pool"-Logzeile, Test bleibt gruen, aber die Zeilen blieben in der DB liegen und
  `plugin_manifests.slug` ist UNIQUE — ein zweiter Lauf waere kollidiert). Fix: `pool.Close()`
  selbst ueber `t.Cleanup` registrieren, VOR dem Aufruf des Helpers, dann laeuft die Schliessung in
  der korrekten LIFO-Reihenfolge zuletzt. Verifiziert: zweiter Testlauf hinterlaesst 0 Zeilen.
- verify vorgaenger: sauber. `dd99f2d7` (g-rls-events-partition) gegen die Fehlerklassen geprueft:
  keine Route, kein `.proto`, kein neuer Guard, kein gRPC-Bypass. Tenant-Handling korrekt
  asymmetrisch (Catch-up-Read + processed-Flag als System, CreateEvent bewusst nicht). Migration
  und Code passen zueinander, docs/ARCHITECTURE.md wurde im selben Commit aktualisiert.
- offen:
  - **Neue Unit `g-rls-tenant-scoped-admin-writes`** wartet als naechste in Block B — fuenf Tabellen,
    Kernfrage vor dem Bauen ist, ob ein Tenant-Provisioning-Hook fuer Default-Zeilen ueberhaupt
    existiert (siehe notes der Unit). `plugin_manifests` hat zusaetzlich eine offene Produktfrage
    (Tenant-Admin-Operation vs. Plattform-Operation), keine reine Migrationsfrage.
  - `g-rls-regression-guard` bleibt `todo`, haengt jetzt an der neuen Unit statt an dieser.
  - Aus Iteration 10/11 unveraendert offen (nicht Teil dieser Iteration): `server/plugin_grpc.go`
    liest den Tenant in den uebrigen Handlern weiter aus dem Request-Body; `SetRolePermissions`
    ohne Audit-Event; `rbac-format.ts`-Katalogluecke; nicht regenerierte
    `desktop/.../api/types.ts`; `SeedRow`/`CleanupRow` brauchen eine `id`-Spalte.

## Iteration 14 — g-rls-tenant-scoped-admin-writes (Teil 1: two_factor_policy) — done — 2026-08-02 23:50

- commit: c65d762c
- entscheidung vorab: die Unit umfasste fuenf Tabellen und war, wie ihre eigenen notes vorhergesagt
  hatten, groesser als eine Iteration. Aufgeteilt statt halb gebaut. `two_factor_policy` zuerst —
  genau der Fall, den die notes als "falls die Unit zerfaellt, diese zuerst" benannt hatten.
- gebaut:
  - `000273_tenant_scope_two_factor_policy` — `tenant_id UUID NOT NULL REFERENCES tenants(id) ON
    DELETE CASCADE`, alter `idx_two_factor_policy_role` (global eindeutig auf `role_name`) ersetzt
    durch `idx_two_factor_policy_tenant_role` auf `(tenant_id, role_name)`, `enable_tenant_rls()`.
    Der Index wird VOR dem Backfill gedroppt, sonst kollidiert die Replikation mit sich selbst.
  - Backfill: jede bestehende globale Zeile per `CROSS JOIN tenants` auf jeden Tenant repliziert.
    Das ist die einzige verlustfreie Lesart der Altdaten — die Zeile galt vorher fuer alle, also
    behaelt jeder Tenant exakt die Policy, unter der er stand. `updated_by` wird nur fuer den
    Tenant uebernommen, dem der bearbeitende Nutzer angehoert (`CASE WHEN u.tenant_id = t.id`);
    sonst waere eine fremde User-Referenz ueber genau die Grenze gewandert, die die Migration
    zieht. Lokal 0 Zeilen (Tabelle leer), Prod ist single-tenant -> dort 1:1.
  - `models.TwoFactorPolicy.TenantID`; Repository-Signaturen `GetTwoFactorPolicy(ctx, tenantID,
    roleName)` und `ListTwoFactorPolicies(ctx, tenantID)`; `UpsertTwoFactorPolicy` konfliktet jetzt
    auf `(tenant_id, role_name)`; Service-Methoden durchgereicht; `Check2FAEnforcement` nimmt
    `user.TenantID`; `AuthGRPCServer.GetTwoFactorPolicy`/`UpdateTwoFactorPolicy` loesen den Tenant
    ueber `middleware.GetTenantID(ctx)` auf (`Unauthenticated`, wenn er fehlt).
  - Tests: `internal/auth/rls_two_factor_policy_test.go` (4 Faelle gegen die echte DB) +
    2 neue gRPC-Faelle je Richtung in `internal/server/grpc_test.go`; der `authMockRepo` keyt jetzt
    auf `(tenant, role)` statt nur auf `role`, sonst haette ein Test gruen sein koennen, der die
    Policy eines fremden Tenants liest.
- der eigentliche Fund dieser Iteration (wichtiger als die Migration): **RLS allein haette den Bug
  NICHT geschlossen.** `Login` (auth/service.go:119) wickelt seinen ganzen Rumpf in
  `sysctx.With(ctx)`, weil der User-Lookup vor jedem Tenant-Kontext passiert — und
  `Check2FAEnforcement` erbt diesen Kontext. Im System-Kontext laesst
  `tenant_id = current_tenant_id() OR is_system_context()` jede Zeile durch; `QueryRow` haette
  also nach der Migration die Policy eines beliebigen fremden Tenants geliefert und den Login
  danach beurteilt. Deshalb laeuft der Tenant explizit als Parameter durch Repository und Service
  (dasselbe Muster wie `CreateRole` aus Welle 1b: `internal/auth` darf `middleware` nicht
  importieren, der gRPC-Layer loest auf). Der vierte Test deckt genau das ab.
- bewusst NICHT gebaut: eine Provisioning-Default-Zeile fuer neue Tenants, obwohl das `done_when`
  der Unit sie fuer die Gruppe forderte. Fuer `two_factor_policy` waere sie tot: der Lesepfad wertet
  eine fehlende Policy als "nicht erzwungen" (totp.go:281, `policy == nil -> continue`), und das ist
  identisch zu einer Zeile mit den Spalten-Defaults (`enforced=false`, `grace_period_days=14`). Die
  Zeile entsteht beim ersten Upsert. Fuer die vier verbleibenden Tabellen gilt das nicht — die lesen
  mit `LIMIT 1` und brauchen den Schritt wirklich.
- kernfrage der Unit beantwortet: **der Provisioning-Hook existiert bereits.**
  `Service.ProvisionTenant` (auth/provisioning.go:76) + `PostgresRepository.ProvisionTenant`
  (auth/postgres_repository.go:849) legen Tenant, Modul-Aktivierungen und Erst-Einladung in EINER
  Transaktion unter `sysctx.With()` an. Die Sorge der notes ("falls der fehlt, ist DAS der
  eigentliche fehlende Baustein") hat sich nicht bestaetigt. Steht im `ergebnis:`-Feld der Unit,
  damit die Folge-Units es nicht erneut recherchieren.
- backlog: Ursprungs-Unit auf `done` mit `ergebnis:`-Feld; drei Folge-Units eingefuegt
  (`g-rls-presence-and-dashboard-defaults`, `g-rls-storage-quotas`, `g-rls-plugin-manifests`),
  `g-rls-regression-guard` haengt jetzt an der letzten davon statt an der Ursprungs-Unit. Stand:
  33 offen / 16 done / 2 blocked.
- gate: build ok (`go build -p 2 ./...`) | vet ok | lint ok (golangci-lint, 0 issues nach Fix eines
  SA5011 im neuen Test) | test ok — `go test -count=1 -v ./internal/auth/...` **166 PASS / 0 SKIP**
  (mit gesetztem `DATABASE_URL` auf `kmuhub_app`), `./internal/server/...` gruen,
  `./internal/gateway/` gruen (keine Route angefasst, `TestOpenAPIRouteDrift` unberuehrt) |
  migration: up/down/up gruen (273 -> 272 -> 273), danach verifiziert `relrowsecurity=t
  relforcerowsecurity=t`, Policy `tenant_isolation`, `tenant_id` NOT NULL, nur noch der
  tenant-gescopte Unique-Index | rls-smoke: die vier Isolationsfaelle sind die Go-Tests selbst;
  `two_factor_policy` hinterher wieder bei 0 Zeilen (kein Test-Rueckstand).
- stolperstein: `golangci-lint` meldet SA5011 fuer `if x == nil { t.Fatal(...) }` gefolgt von einer
  Dereferenzierung — `t.Fatal` gilt staticcheck nicht als noreturn. Ein explizites `return` nach
  `t.Fatal` loest es. Derselbe Befund wie in Lauf 1 (dort 5x); offenbar tritt er bei jedem neuen
  Test mit Nil-Check-plus-Zugriff wieder auf.
- verify vorgaenger: sauber. `a732b743` (g-rls-allowlist-audit) gegen die Fehlerklassen geprueft:
  keine Route, kein `.proto`, kein neuer Guard, kein gRPC-Bypass. Migration, Backfill, Down-Pfad und
  die vier Go-Callsites passen zusammen. Zusaetzlich die beiden GDPR-Pfade
  (`security/gdpr/erasure.go:109`, `export.go:196`) gegengeprueft, die `refresh_tokens` per
  `user_id` anfassen — sie laufen im Tenant-Kontext des betroffenen Nutzers, RLS trifft sie also
  korrekt und nicht ueberraschend.
- offen:
  - Die vier Rest-Tabellen (siehe Folge-Units). `g-rls-plugin-manifests` traegt eine echte
    Produktfrage; im Backlog steht ein dritter Weg, der in der Ursprungs-Unit fehlte: `tenant_id`
    NULLable mit NULL = "von Zentria ausgeliefert", also dasselbe Policy-Paar wie `roles` seit
    000256 — das loest Katalog und Tenant-Manifeste in einem Schema.
  - `desktop/src/renderer/src/api/security-types.ts` `TwoFactorPolicy` kennt das neue Feld
    `tenant_id` nicht. Kein Bruch (das FE liest es nicht), aber die generierte `types.ts` ist damit
    einen Tick veraltet — reiht sich in den schon offenen Punkt "nicht regenerierte types.ts" ein.
  - Aus Iteration 10/11/13 unveraendert offen: `server/plugin_grpc.go` liest den Tenant in den
    uebrigen Handlern weiter aus dem Request-Body; `SetRolePermissions` ohne Audit-Event;
    `rbac-format.ts`-Katalogluecke; `SeedRow`/`CleanupRow` brauchen eine `id`-Spalte.

## Iteration 15 — g-rls-presence-and-dashboard-defaults — done — 2026-08-03 00:45

- commit: c0c30e79
- gebaut:
  - `000274_tenant_scope_presence_and_dashboard_defaults` — beide Tabellen bekommen `tenant_id UUID
    NOT NULL REFERENCES tenants(id) ON DELETE CASCADE` + `enable_tenant_rls()`. `presence_config`
    hatte ueberhaupt keinen Unique-Key (nur PK auf `id`), bekommt jetzt `idx_presence_config_tenant`
    auf `tenant_id`; `dashboard_defaults` verliert `dashboard_defaults_role_key` (UNIQUE(role)) und
    das redundante `idx_dashboard_defaults_role` zugunsten von `(tenant_id, role)`.
    Backfill wie in 000273 per `CROSS JOIN tenants`: 1 -> 1655 Zeilen (presence), 3 -> 4965
    (dashboard). Bei `presence_config` nimmt der Backfill bewusst nur die zuletzt bearbeitete
    globale Zeile (`ORDER BY updated_at DESC LIMIT 1`) — die Tabelle konnte mangels Unique-Key
    mehrere tragen, und die haetten sich beim Replizieren gegenseitig auf dem neuen Index blockiert.
  - `presence`: `ConfigRepository.GetConfig/UpdateConfig` nehmen `tenantID` explizit; `UpdateConfig`
    ist ein **Upsert** (`ON CONFLICT (tenant_id)`), weil ein Tenant ohne Zeile sonst einen stillen
    Erfolg ohne Wirkung bekommen haette. `ErrConfigNotFound`/`ErrMissingTenant` neu.
  - `gateway`: `GetDefaultLayout`/`UpsertDefaultLayout` nehmen `tenantID`, filtern bzw. schreiben
    explizit darauf (nicht nur ueber RLS — dieselbe sysctx-Begruendung wie 000273), `ON CONFLICT
    (tenant_id, role)`. Die Admin-Handler loesen den Tenant auf und liefern 401 ohne ihn; beide
    Routen hatten den Tenant vorher gar nicht gebraucht.
- entscheidung (die offene Frage der Unit): **Weg (b), Code-Default statt Provisioning-Zeile.** Nicht
  weil er leaner ist, sondern weil beide Lesepfade ihn schon hatten: `hardcodedDefaultLayout()` ist
  seit jeher die dritte Stufe von `GetDashboard` (dashboard_service.go:24), und
  `DefaultAwayTimeoutSeconds` war bereits der Fallback beider `getAwayTimeout`-Aufrufer — die notes
  verlangten genau diese Pruefung ("nur, wenn der Default wirklich im Code steht"). Ein
  Provisioning-Insert haette dupliziert, was im Code steht, und 1655 Zeilen je Tabelle erzeugt, die
  nichts aussagen. `presence.Service.GetConfig` beantwortet `ErrConfigNotFound` jetzt mit dem
  Default statt mit einem Fehler; damit liefert `GET /presence/config` fuer einen frischen Tenant
  200 statt 500.
- der eigentliche Fund dieser Iteration: **die Migration allein haette die Luecke nicht geschlossen —
  zwei Caches haetten sie im Betrieb wieder geoeffnet.**
  - `presence.Service` hielt den Away-Timeout in einem prozessweiten Feld (`cachedAwayTimeout` +
    `cachedAwayTimeoutAt`, 60 s TTL). Der erste Tenant, der den Cache fuellt, haette seinen Timeout
    fuer die naechste Minute an alle anderen ausgeliefert. Jetzt `map[uuid.UUID]awayTimeoutEntry`,
    und `UpdateConfig` invalidiert nur den eigenen Eintrag.
  - `CachedDashboardRepository` cachte unter `cache:dashboard:defaults:<role>` in Redis, 30 min TTL —
    dasselbe Muster, nur laenger und ueber Prozessgrenzen hinweg. Key traegt jetzt den Tenant
    (`<tenant>:<role>`), analog zum schon vorhandenen `keyDashboardUser`. Regressionstest
    `TestCachedDashboard_GetDefaultLayout_NotSharedAcrossTenants`.
  Das ist die Lehre fuer die zwei Rest-Units: nach der Tabelle die Caches suchen.
- bestandsbug mitgefixt (liegt auf demselben Schreibpfad): `UpdatePresenceConfig`
  (server/video_grpc.go:1350) uebergab die **Tenant-ID** als `updated_by`, obwohl die Spalte
  `users(id)` referenziert. `PUT /presence/config` lief damit in einen Fremdschluesselfehler, sobald
  die Tenant-ID nicht zufaellig auch eine User-ID war — der Endpoint konnte gar nicht erfolgreich
  sein. Jetzt `middleware.GetUserID(ctx)`, mit `Unauthenticated` wenn er fehlt. `GetPresenceConfig`
  gibt seinen Fehler jetzt durch `mapPresenceError` statt pauschal `Internal`.
- tests: `internal/work/presence/rls_config_test.go` (5 Faelle: Isolation, unabhaengige Timeouts,
  Upsert fuer frischen Tenant, Code-Default, fehlender Tenant) und
  `internal/gateway/rls_dashboard_defaults_test.go` (3 Faelle: Isolation, unabhaengige Presets,
  fremder Tenant liest NotFound). Dazu 2 x 401 fuer die Admin-Defaults-Routen in
  `tenant_isolation_test.go` und der Cache-Isolationstest. Beide Mocks keyen jetzt auf den Tenant
  (`mockConfigRepo`, `mockDashboardRepo`) — mit einem globalen Mock waere ein Test gruen geblieben,
  der eine fremde Zeile liest, derselbe Griff wie beim `authMockRepo` in Iteration 14.
- stolperstein (neu, kostet sonst still Daten): **`t.Cleanup` laeuft NACH allen `defer`s**, also auch
  nach `defer pool.Close()`. Ein dort registriertes `CleanupRow` trifft einen geschlossenen Pool,
  `CleanupRow` loggt den Fehler nur (`t.Logf`) — der Test bleibt gruen und laesst Daten liegen. In
  den ersten Laeufen dieser Iteration sind so 22 Tenants + 12 Test-User haengengeblieben (hinterher
  entfernt, lokale DB wieder sauber). Cleanups in diesen Tests laufen jetzt als `defer` in der
  Testfunktion, mit `func()`-Rueckgabe aus den Seed-Helfern.
- gate: build ok (`go build -p 2 ./internal/... ./cmd/gateway/... ./cmd/work/...`) | vet ok | lint ok
  (golangci-lint, 0 issues) | test ok — `go test -count=1 ./internal/gateway/ ./internal/work/...
  ./internal/server/...` alles gruen, mit `DATABASE_URL` auf `kmuhub_app`: **presence 28 PASS /
  0 SKIP**, **gateway 604 PASS / 0 SKIP** | migration: up/down/up gruen (274 -> 273 -> 274), danach
  verifiziert `relrowsecurity=t relforcerowsecurity=t`, Policy `tenant_isolation`, `tenant_id`
  NOT NULL, tenant-gescopte Unique-Indizes, 1655/4965 Zeilen ueber 1655 Tenants | openapi: keine neue
  Route, `401` war fuer alle vier betroffenen Operationen bereits dokumentiert — `TestOpenAPIRouteDrift`
  unberuehrt gruen.
- verify vorgaenger: sauber. `c65d762c` (two_factor_policy) gegen die Fehlerklassen geprueft: keine
  Route, kein `.proto`, kein neuer Guard, kein gRPC-Bypass; Migration, Backfill, Down-Pfad und die
  Callsites in `auth`/`server` passen zusammen. Der Down-Pfad kollabiert per `DISTINCT ON (role_name)`
  bewusst verlustbehaftet — dokumentiert im Migrationskopf.
- backlog: Unit auf `done` mit `ergebnis:`-Feld (Entscheidung + die drei Funde, damit
  `g-rls-storage-quotas` und `g-rls-plugin-manifests` sie nicht erneut herleiten muessen).
  Stand: 32 offen / 17 done / 2 blocked.
- offen:
  - Die zwei Rest-Tabellen der Gruppe. Fuer `storage_quotas` ist die Backfill-Frage offen (der
    globale `used_bytes` laesst sich nicht auf Tenants aufteilen); fuer `plugin_manifests` steht die
    Produktfrage.
  - `desktop/src/renderer/src/api/*`: weder `PresenceConfig` noch `DashboardDefault` kennen das neue
    `tenant_id`-Feld. Kein Bruch (das FE liest es nicht), reiht sich in den offenen Punkt
    "nicht regenerierte types.ts" ein — jetzt drei Typen tief.
  - Aus Iteration 10/11/13/14 unveraendert offen: `server/plugin_grpc.go` liest den Tenant in den
    uebrigen Handlern weiter aus dem Request-Body; `SetRolePermissions` ohne Audit-Event;
    `rbac-format.ts`-Katalogluecke; `SeedRow`/`CleanupRow` brauchen eine `id`-Spalte.

## Iteration 16 — g-rls-storage-quotas — done — 2026-08-02 23:23

- commit: d48eab68
- gebaut:
  - `000275_tenant_scope_storage_quotas` — `tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE
    CASCADE` + `UNIQUE(tenant_id)` (Conflict-Target fuer den Upsert unten) + `enable_tenant_rls()`.
    Anders als `two_factor_policy`/`presence_config`/`dashboard_defaults` liess sich `used_bytes`
    NICHT verlustfrei aus der alten globalen Zeile replizieren — die Zahl war eine Summe ueber alle
    Tenants. Der Backfill berechnet `used_bytes` deshalb je Tenant aus `SUM(file_size)` ueber
    `chat_files` (tenant-gescopt + RLS seit 000115/000122) neu; `max_bytes` wird wie in
    000273/000274 1:1 aus der alten globalen Zeile repliziert. Lokal 0 Zeilen in `chat_files` —
    Backfill also rechnerisch verifiziert (1671 Tenants -> 1671 Zeilen, alle used_bytes=0), nicht
    gegen echte Dateidaten.
  - `chat/file`: `GetStorageQuota(ctx, tenantID)` liefert `ErrQuotaNotFound` statt einer Zeile eines
    fremden Tenants. `IncrementUsedBytes` ist jetzt ein Upsert (INSERT...ON CONFLICT(tenant_id) DO
    UPDATE) — eine reine WHERE-UPDATE-Variante haette einem Tenant ohne Zeile einen stillen Erfolg
    ohne Wirkung gemeldet; der Insert-Zweig zieht `max_bytes` aus dem Spalten-Default.
    `DecrementUsedBytes` bleibt ein WHERE-gescoptes UPDATE (die Zeile existiert immer schon, weil
    Increment sie beim ersten Upload anlegt). `GetFileByID` selektiert jetzt `tenant_id` mit.
  - `Service.Upload` loest den Tenant jetzt VOR der Quota-Pruefung auf (vorher erst kurz vor
    `CreateFile`, ein zweiter Aufruf) und nutzt ihn fuer Quota-Read, Quota-Increment und
    `chatFile.TenantID`. `Service.Delete` liest den Tenant NICHT aus dem Context, sondern aus dem
    bereits geladenen `file.TenantID` — vermeidet eine zweite Tenant-Aufloesungspflicht fuer jeden
    Delete-Caller.
- entscheidung (die offene Frage der Unit): Code-Default statt Provisioning-Zeile, dieselbe Wahl wie
  000274 und aus demselben Grund — `IncrementUsedBytes` legt die Zeile beim ersten Upload an, ein
  Provisioning-Insert haette nur dupliziert, was im Code steht. Bei `ErrQuotaNotFound` faellt
  `Upload` auf den neuen Code-Default `DefaultMaxQuotaBytes` (10 GB, identisch zum Spalten-Default)
  zurueck statt den Upload zu blocken.
- tests: `internal/chat/file/rls_storage_quota_test.go` (3 Faelle: Fremd-Tenant sieht 0 Zeilen,
  unabhaengige Zaehler unter parallelen Increment/Decrement-Aufrufen, Upsert legt Zeile fuer
  frischen Tenant an inkl. Spalten-Default-Pruefung). Cleanups als `defer`, nicht `t.Cleanup` (Lehre
  aus Iteration 15). Bestehende Mocks (`chat/file` und `server`) auf die neue `tenantID`-Signatur
  angepasst; zwei Upload-Testfaelle ("quota exceeded", "scan failed") liefen vorher mit
  `context.Background()`, weil ihr Fehlerpfad vor der alten, spaeteren Tenant-Aufloesung lag — jetzt
  liegt die Aufloesung davor, beide Faelle brauchen darum `tenantCtx(uuid.Nil)` wie die Erfolgsfaelle.
- gate: build ok (`go build -p 2 ./...`) | vet ok | lint ok (golangci-lint, 0 issues auf
  `internal/chat/file`, `internal/server`, `internal/models`) | test ok — `go test -count=1
  ./internal/chat/file/... ./internal/server/... ./internal/gateway/...` mit `DATABASE_URL` auf
  `kmuhub_app`, 0 Skips in den neuen RLS-Tests | migration: up/down/up gruen (274 -> 275 -> 274 ->
  275), Policy `tenant_isolation` mit FORCE ROW LEVEL SECURITY, `tenant_id` NOT NULL,
  `UNIQUE(tenant_id)`, 1671 Zeilen ueber 1671 Tenants nach Wiederherstellung verifiziert | openapi:
  keine neue Route, `TestOpenAPIRouteDrift` unberuehrt gruen (782/784).
- verify vorgaenger: sauber. `c0c30e79` (presence_config/dashboard_defaults) gegen die
  Fehlerklassen geprueft: keine Route, kein `.proto`, kein neuer Guard, kein gRPC-Bypass; Migration,
  Backfill, Down-Pfad, Repository-/Service-Callsites und die Cache-Tenant-Scoping passen zusammen.
- backlog: Unit auf `done` mit `ergebnis:`-Feld. Stand: 31 offen / 18 done / 2 blocked.
- offen:
  - `g-rls-plugin-manifests` bleibt die letzte Tabelle der Fuenfer-Gruppe und hat zusaetzlich eine
    offene Produktfrage (Tenant-Admin- oder Plattform-Operation) — siehe deren eigene notes.
  - `desktop/src/renderer/src/api/*` kennt das neue `tenant_id`-Feld auf `StorageQuota` nicht. Kein
    Bruch (das FE liest es nicht), reiht sich in den offenen Punkt "nicht regenerierte types.ts" ein
    (jetzt vier Typen tief).
  - Aus Iteration 10/11/13/14 unveraendert offen: `server/plugin_grpc.go` liest den Tenant in den
    uebrigen Handlern weiter aus dem Request-Body; `SetRolePermissions` ohne Audit-Event;
    `rbac-format.ts`-Katalogluecke; `SeedRow`/`CleanupRow` brauchen eine `id`-Spalte.

## Iteration 17 — g-rls-plugin-manifests — done — 2026-08-03 00:31

- commit: 15c2ccd6
- entscheidung (die Produktfrage der Unit): **gebaut, nicht blockiert.** Von den drei Wegen der notes
  traegt der dritte: nullable `tenant_id` mit dem Policy-Paar aus 000256 (`roles`/`role_permissions`).
  NULL = "mit dem Produkt ausgeliefert, fuer jeden Tenant lesbar, fuer keinen schreibbar", gesetzter
  Tenant = "eigenes Manifest"; Lesen sieht beides, Schreiben nur das eigene. Weg (A) `tenant_id NOT
  NULL` kann keinen Zentria-Katalog ausdruecken, Weg (B) Plattform-Operation braucht eine Rolle, die
  heute keine Route vergibt, und naehme Tenants die eigenen Config-Plugins. Der dritte loest beides in
  einem Schema und haelt EIN Muster im Schema statt zwei.
- gebaut:
  - `000276_tenant_scope_plugin_manifests` — `tenant_id UUID NULL REFERENCES tenants(id) ON DELETE
    CASCADE`, `plugin_manifests_slug_key` (global unique) ersetzt durch den Ausdrucks-Index
    `(COALESCE(tenant_id,'0000…'::uuid), slug)` byte-identisch zu 000256, RLS ENABLE+FORCE mit
    `tenant_isolation_read` (`tenant_id IS NULL OR = current_tenant_id() OR is_system_context()`) und
    `tenant_isolation_write` (nur eigener Tenant). **Kein `enable_tenant_rls()`** — dessen symmetrische
    Policy haette den Katalog fuer jeden Tenant unsichtbar gemacht. Kein Backfill: bestehende Zeilen
    bleiben NULL, das ist die verlustfreie Lesart (sie waren fuer alle sichtbar und bleiben es).
  - `plugin/repository/manifest.go`: `tenant_id` in INSERT und allen drei SELECTs; `GetBySlug` mit
    `ORDER BY (tenant_id IS NULL), id LIMIT 1`, damit ein spaeter nachgeschobener Katalog-Slug die
    eigene Zeile nicht verdraengt.
  - `plugin/service.go`: `CreateManifest` stempelt `middleware.GetTenantID(ctx)` — aber NACH den
    Eingabepruefungen, damit ein fehlerhaftes Manifest weiterhin 400 statt 500 liefert.
    `DeleteManifest` weist Katalog-Manifeste mit `ErrManifestImmutable` ab (neu, gemappt auf
    `codes.PermissionDenied` -> 403).
- funde ueber die Unit hinaus (beide hier gefixt):
  1. **`DELETE /manifests/{id}` war tenant-uebergreifend destruktiv.** `DeleteManifest` verweigert das
     Loeschen bei aktiven Installationen, zaehlt sie aber ueber `plugin_installations` — das seit
     000122 RLS hat. Fremde Installationen waren fuer die Zaehlung unsichtbar, der Count kam als 0
     zurueck, und `manifest_id ON DELETE CASCADE` riss die Installation des fremden Tenants mit.
     Nach 000276 sind Katalog-Zeilen fuer Tenants nicht mehr loeschbar.
  2. **`wasm_binary_hash` ist NULLable ohne Default gegen ein `string`-Feld im Model.** Eine Zeile ohne
     diesen Wert — genau das, was die jetzt vorgesehene Katalog-Seed-Migration schreiben wuerde — liess
     jeden Read fuer ALLE Tenants am Scan scheitern (`cannot scan NULL into *string`). Die drei
     Lesequeries nutzen jetzt `COALESCE(wasm_binary_hash, '')`. Aufgefallen nur, weil der
     Isolationstest eine Katalog-Zeile per `SeedRow` anlegt.
- gepruefte Lehren der Vorgaenger-Iterationen: **kein Cache** im Plugin-Modul (Iteration 15) und **kein
  `sysctx.With()`-Lesepfad** (Iteration 14) — beide Suchen negativ, hier also nichts zu scopen.
- tests: `internal/plugin/repository/manifest_rls_test.go` (4 Faelle: eigenes Manifest fuer den fremden
  Tenant unsichtbar + Katalog fuer beide sichtbar, Create fuer fremden Tenant und Create einer
  Katalog-Zeile beide mit SQLSTATE 42501 abgewiesen, gleicher Slug in zwei Tenants erlaubt und
  `GetBySlug` liefert die eigene Zeile, DELETE einer Katalog-Zeile trifft still nichts). Eigene Tenants
  per `uuid.New()`, Cleanups als `defer` inklusive der `tenants`-Zeilen (0 liegengeblieben verifiziert).
  Drei Service-Tests neu (Tenant-Stempel, fehlender Tenant-Context, Katalog-Immutability).
  Bestand: `service_test.go` musste von `context.Background()` auf einen Tenant-Context umgestellt
  werden (57 Stellen, ein `tenantCtx()`-Helper) — dieselbe Klasse wie in Iteration 16 bei `chat/file`.
- gate: build ok (`go build -p 2 ./...`) | vet ok | lint ok (golangci-lint, 0 issues auf
  `internal/plugin`, `internal/server`, `internal/models`) | test ok — `go test -count=1
  ./internal/plugin/... ./internal/server/... ./internal/gateway/` mit `DATABASE_URL` auf `kmuhub_app`,
  **71 PASS / 0 SKIP** im Plugin-Baum | migration: up/down/up gruen (275 -> 276 -> 275 -> 276), Policies
  und Ausdrucks-Index nach Wiederherstellung verifiziert. Der **Down-Pfad wurde mit echten Duplikaten
  geprueft**, nicht nur auf der leeren Tabelle: drei Zeilen mit demselben Slug (zwei Tenants + Katalog),
  nach `down` ueberlebt wie dokumentiert die Katalog-Zeile | openapi: keine neue Route,
  `TestOpenAPIRouteDrift` gruen (782/784); 403 war am DELETE bereits dokumentiert, Beschreibung um den
  neuen Fall und die Slug-Semantik ergaenzt.
- verify vorgaenger: sauber. `d48eab68` (storage_quotas) gegen die acht Fehlerklassen geprueft: keine
  neue Route, kein `.proto`, kein neuer Guard, kein gRPC-Bypass. Migration setzt `tenant_id` NOT NULL,
  Unique-Index und RLS; der Upsert-Conflict-Target passt zum Index; `GetStorageQuota` hat genau einen
  Aufrufer (`Upload`), der `ErrQuotaNotFound` behandelt — kein Pfad laeuft in den neuen Fehler.
- backlog: Unit auf `done` mit `ergebnis:`-Feld. Damit ist die Fuenfer-Gruppe aus
  `g-rls-tenant-scoped-admin-writes` (000273-000276) vollstaendig geschlossen.
  Stand: 30 offen / 19 done / 2 blocked.
- offen:
  - Naechste Unit ist `g-rls-regression-guard` — der Test, der genau diesen Block kuenftig verhindert.
    `plugin_manifests` hat jetzt ein Policy-PAAR statt der Standard-`tenant_isolation`; der Guard darf
    also nicht auf den Policy-Namen pruefen, sondern auf `relrowsecurity`.
  - `ErrPluginHasInstallations` faellt in `mapPluginError` in den `default`-Zweig und wird als 500
    beantwortet, obwohl es fachlich ein 409 ist. Bestand, nicht in dieser Unit angefasst.
  - `desktop/src/renderer/src/api/*` kennt `tenant_id` auf keinem der betroffenen Typen — hier ohne
    Wirkung, weil das Proto (`ManifestMsg`) kein Feld dafuer hat und die HTTP-Antwort es nicht traegt.
  - Aus Iteration 10/11/13/14 unveraendert offen: `server/plugin_grpc.go` liest den Tenant in den
    uebrigen Handlern weiter aus dem Request-Body; `SetRolePermissions` ohne Audit-Event;
    `rbac-format.ts`-Katalogluecke; `SeedRow`/`CleanupRow` brauchen eine `id`-Spalte.

## Iteration 18 — g-rls-regression-guard — done — 2026-08-03 01:15

- commit: ead9923e
- gebaut: `backend/internal/testutil/rls_regression_test.go`,
  `TestAllPublicTablesHaveRLSOrAreAllowlisted`. Scannt `pg_class` in `public` fuer
  `relkind IN ('r','p')`, schliesst Partitionen ueber `relispartition` aus (direkter als ein Join
  ueber `pg_inherits`, auf PG16 verifiziert) und verlangt fuer den Rest `relrowsecurity = true`.
  Zwei Ausnahme-Maps: `systemGlobalAllowlist` (die sieben ADR-006-Tabellen aus
  docs/ARCHITECTURE.md) und `knownRLSGaps` (aktuell nur `user_roles`, mit Verweis auf die offene
  Unit `g-user-roles-rls`). Absichtlich getrennt: die erste Map ist dauerhaft legitim, die zweite
  ist bekannte Schuld, die beim Schliessen ihrer Unit wieder rausfliegt — ein neuer, unbenannter
  Fund faellt durch keine von beiden und macht den Test rot.
- scan-ergebnis (Kopf 276, verifiziert gegen die lokale DB): genau 7 Allowlist-Treffer + 1
  bekannter Gap (`user_roles`) = 8 Tabellen ohne RLS ausserhalb von Partitionen. Alle 32
  Partitions-Kinder von `automation_executions`, `dialer_call_events`, `events` zeigen
  `relispartition=true` und werden uebersprungen; ihre drei Eltern haben `relrowsecurity=true`
  und werden korrekt NICHT ausgenommen — waeren sie es, haette der Test diese drei Familien nie
  geprueft, und genau das war das Risiko, das die Unit adressiert.
- gate: build ok (`go build -p 2 ./internal/testutil/...`) | vet ok | lint ok (golangci-lint,
  0 issues) | test ok — `go test -count=1 ./internal/testutil/...`, **4/4 PASS, 0 Skips**
  (`DATABASE_URL` auf `kmuhub_app`). Keine Migration, kein Proto, keine Route angefasst —
  `go test ./internal/gateway/` ist deshalb nicht Teil dieses Gates.
- verify vorgaenger: sauber. `15c2ccd6` (plugin_manifests, Iteration 17) gegen die acht
  Fehlerklassen geprueft: keine neue Route, kein `.proto`, kein neuer Guard, kein gRPC-Bypass,
  Migration (nullable `tenant_id` + asymmetrisches Policy-Paar statt `enable_tenant_rls()`,
  begruendet) und Down-Pfad sauber, `CreateManifest` stempelt den Tenant nach den
  Eingabepruefungen wie beschrieben.
- backlog: Unit auf `done` mit `ergebnis:`-Feld. Damit ist **Block B (Sicherheit/RLS-Reste)
  vollstaendig geschlossen** — der Scan, der ihn ausgeloest hat, ist jetzt Dauerpruefung statt
  Einmalaktion.
  Stand: 29 offen / 20 done / 2 blocked.
- offen:
  - `user_roles` bleibt ein echter, wenn auch heute ungefaehrlicher Gap — `g-user-roles-rls`
    steht weiterhin als eigene Unit im Backlog (Block G/3), nicht in diesem Lauf gezogen.
  - Alle unveraendert offenen Punkte aus Iteration 17 (Plugin-Grpc-Tenant-aus-Body,
    SetRolePermissions ohne Audit-Event, rbac-format.ts-Katalogluecke, SeedRow/CleanupRow ohne
    `id`-Spalte, nicht regenerierte `types.ts`) bleiben unveraendert offen, hier nicht angefasst.

## Iteration 19 — g-crm-contact-timeline — blocked — 2026-08-03 02:00

- commit: 84292279
- gebaut: nichts — Praemisse der Unit widerlegt. `GET /api/v1/contacts/{id}/timeline` existiert
  bereits vollstaendig und korrekt: Proto-RPC (crm.proto:93/1018-1041), Service
  `activity.Service.GetContactTimeline` (activity/service.go:422), Repository-Query unioniert
  `activities`+`deals` tenant-gescopt in beiden Armen (activity/postgres_repository.go:372-403),
  Gateway-Handler geht ueber den gRPC-Client (route_crm_ext.go:146), Route registriert
  (route_crm.go:103, route_crm_ext.go:367), openapi.yaml dokumentiert (Zeile 28390), sogar per
  `test/e2e/dialer_test.go:155` end-to-end geprueft. Kein Stub, kein Bypass, kein Tenant-Leck.
- befund: der reale Fehler ist FE-seitig. `desktop/src/renderer/src/api/hooks/useTimeline.ts:40`
  ruft `/api/v1/crm/contacts/${contactId}/timeline` — ein `/crm/`-Segment, das kein anderer
  CRM-Hook nutzt (`useContacts.ts`, `useContactTags.ts` gehen alle direkt unter
  `/api/v1/contacts/...`). Jeder echte Aufruf bekommt 404. Kein totes FE-Feature: der Hook wird
  von `ContactTimeline.tsx` genutzt, gerendert in `modules/kontakte/ContactDetailPanel.tsx` — die
  Kontakt-Chronik ist in der Detailansicht produktiv nie sichtbar.
  Zweiter, unabhaengiger Vertragsbruch, der erst nach dem Pfad-Fix sichtbar wuerde: openapi.yaml
  dokumentiert den Query-Parameter `offset` explizit als 1-basierte Seitenzahl, nicht als
  Zeilen-Offset (Zeile 28398-28400), der Gateway-Handler reicht ihn so an `Page` durch
  (route_crm_ext.go:158-161). `useTimeline.ts:63` berechnet aber einen echten Zeilen-Offset
  (`(page-1)*pageSize`) — fuer Seite 1 zufaellig 0 (passt), ab Seite 2 waere die angefragte RPC-Seite
  falsch (offset=20 bei pageSize=20 -> Page=20 statt Page=2). Ungetestet, weil der einzige
  bestehende Test nur Seite 1 abruft.
- gate: n.a. — keine Codeaenderung, reine Diagnose. Kein Build/Test noetig.
- verify vorgaenger: sauber. `ead9923e` (RLS-Regression-Guard, Iteration 18) gegen die acht
  Fehlerklassen geprueft: reiner Testzusatz, keine Route/Proto/Migration/Guard angefasst, Test
  selbst korrekt (siehe Code-Review oben in dieser Iteration) — nichts zu beanstanden.
- backlog: Unit auf `blocked` mit `blocked_reason:`. Beide noetigen Fixes liegen in
  `desktop/src/renderer/src/api/hooks/useTimeline.ts` (Pfad korrigieren + Offset/Page-Semantik
  entscheiden) — dieser Backend-Loop fasst `desktop/` nicht an, das ist eine FE-Session- bzw.
  Luke-Entscheidung (welche Seite sich an die andere anpasst).
  Stand: 26 offen / 20 done / 3 blocked.
- offen:
  - **Fuer Luke/eine FE-Session:** `useTimeline.ts:40` von `/api/v1/crm/contacts/{id}/timeline`
    auf `/api/v1/contacts/{id}/timeline` korrigieren, DANACH die Offset/Page-Frage klaeren (FE
    sendet echte Page-Nummer statt berechnetem Zeilen-Offset, ODER Backend interpretiert den
    Parameter als echten Offset um) — sonst ist Seite 1 der Chronik korrekt und jede Folgeseite
    liefert falsche Events.
  - Alle unveraendert offenen Punkte aus Iteration 17/18 (Plugin-Grpc-Tenant-aus-Body,
    SetRolePermissions ohne Audit-Event, rbac-format.ts-Katalogluecke, SeedRow/CleanupRow ohne
    `id`-Spalte, nicht regenerierte `types.ts`, `ErrPluginHasInstallations` faellt auf 500 statt
    409) bleiben unveraendert offen, hier nicht angefasst.

## Iteration 20 — fix-calendar-cancel-booking-actor — done — 2026-08-03 00:20

- commit: 989ff60f
- gebaut: `g-calendar-resource-bookings` gezogen — Praemisse widerlegt, gleiche Fehlerklasse wie
  `g-crm-contact-timeline` (Iteration 19): `POST/DELETE /calendar/bookings` existieren seit
  laengerem vollstaendig (Proto, Service mit Konflikt-Check + Owner-Check, gRPC-gebundener
  Gateway-Handler, openapi.yaml). Das FE ruft ueber `useResources.ts:123,138` ein zusaetzliches
  `/resources/`-Segment (`/api/v1/calendar/resources/bookings`), das im Gateway nicht existiert —
  jeder reale Buchungs-/Stornierungsversuch bekommt 404. Fix liegt in `desktop/`, ausserhalb
  dieses Loops; Unit auf `blocked` mit vollstaendiger Fundstellen-Liste im `blocked_reason`.
  Bei der Recherche dazu ein zweiter, unabhaengiger und schwererer Fund: `CalendarGRPCServer.
  CancelBooking` (calendar_grpc.go:1101) uebergab hartcodiert `uuid.Nil` als Actor an
  `resource.Service.CancelBooking` ("Gateway handles auth; use uuid.Nil as actorID"), dessen
  Owner-Check aber `booking.BookedBy != actorID` prueft — mit `uuid.Nil` schlaegt das fuer JEDEN
  echten Booker fehl. Stornieren war also fuer niemanden moeglich, unabhaengig vom FE-Pfad-Bug.
  Dieselbe Fehlerklasse ist im Repo bereits einmal aufgetreten und behoben worden
  (`fix-g-work-task-comment-authz`, `work_grpc.go`/`work_comment_test.go`) — dort wie hier ein
  hartcodierter `uuid.Nil`-Actor, der einen Owner-Check bricht.
  Root-Cause-Fix statt Proto-Aenderung: `x-user-id` propagiert bereits automatisch vom
  HTTP-Gateway-Kontext in den internen gRPC-Kontext (`TenantOutboundUnaryInterceptor` in
  `internal/gateway/registry.go:112`, `TenantInboundUnaryInterceptor` in `cmd/work/main.go:182`)
  — verifiziert, nicht angenommen. `CancelBooking` liest jetzt `middleware.GetUserID(ctx)`, exakt
  das etablierte Muster aus `document_grpc.go:1652`/`dialer_grpc.go:76`. Als eigene Fix-Unit
  `fix-calendar-cancel-booking-actor` in Block E angelegt und in dieser Iteration sofort
  abgearbeitet (analog zum Verify-Vorspann-Pattern: Fund -> Fix-Unit -> gleiche Iteration).
- gate: build ok (`go build -p 2 ./internal/server/... ./internal/work/resource/...
  ./internal/gateway/... ./cmd/work/... ./cmd/gateway/...`) | vet ok | lint ok (golangci-lint,
  0 issues auf internal/server) | test ok — `go test -count=1 ./internal/server/...`, 210 PASS /
  0 SKIP (`DATABASE_URL` gesetzt; das Paket braucht hier keine echte DB, die drei neuen Tests
  laufen gegen einen In-Memory-`resource.Repository`-Mock). Keine Migration, kein `.proto`, keine
  neue/geaenderte Route, kein neuer Guard — `openapi.yaml` und
  `go test ./internal/gateway/` daher nicht Teil dieses Gates.
- verify vorgaenger: n.a. — `84292279` (Iteration 19) ist eine reine Diagnose ohne Codeaenderung
  (nur `JOURNAL.md`/`BACKLOG.yml`), nichts zu verifizieren.
- offen:
  - **Fuer Luke/eine FE-Session:** `useResources.ts:123,138` von
    `/api/v1/calendar/resources/bookings(/…)` auf `/api/v1/calendar/bookings(/…)` korrigieren —
    sonst bleiben Ressourcen-Buchung und -Stornierung im produktiven FE weiterhin 404, auch nach
    diesem Fix.
  - Alle unveraendert offenen Punkte aus Iteration 17/18/19 (Plugin-Grpc-Tenant-aus-Body,
    SetRolePermissions ohne Audit-Event, rbac-format.ts-Katalogluecke, SeedRow/CleanupRow ohne
    `id`-Spalte, nicht regenerierte `types.ts`, `ErrPluginHasInstallations` faellt auf 500 statt
    409, `useTimeline.ts`-Pfad+Offset-Bug) bleiben unveraendert offen, hier nicht angefasst.

## Iteration 21 — g-admin-billing — blocked — 2026-08-03 (siehe Commit-Zeitstempel)

- commit: - (reine Diagnose, kein Code-Commit — siehe unten)
- gebaut: nichts. `g-admin-billing` gezogen (`GET /admin/billing`), Praemisse geprueft und doppelt
  widerlegt. `useBilling.ts` traegt im Datei-Kopf explizit "Kein Backend-Call. Alle Daten aus
  Mock-Datei oder localStorage." — anders als bei `g-crm-contact-timeline`/
  `g-calendar-resource-bookings` ruft das FE hier nicht mal einen falschen Pfad, sondern GAR
  keinen. `/admin/billing` ist nur ein React-Router-Client-Pfad (`App.tsx:297`).
  Zweiter, wichtigerer Fund: die vermutete fehlende Datenbasis existiert bereits UND hat bereits
  gebaute, getestete Endpoints — nur nicht unter `/admin/billing`. `GET /api/v1/admin/subscription`
  (route_settings.go:875) liefert `planType, supportTier, status, billingPeriodEnd, totalSeats` aus
  `tenants.*` (Migration 000250) — 1:1 die Felder von `MockTenantData`. `GET/PATCH
  /api/v1/admin/license` (route_settings.go:788/823) bedient `tenant_module_activations`
  (ebenfalls 000250). `GET /api/v1/tenant/module-grants` (route_settings.go:74-80) bedient
  `user_module_grants` (Migration 000220) — die Datenbasis fuer `useModuleAssignments()`. Alle drei
  RLS-geschuetzt, permission-gegated, in openapi.yaml dokumentiert. Was der Mock zusaetzlich zeigt
  (Invoice-History, Usage-Stats) braucht Infrastruktur, die es nicht gibt: Invoice-History ein
  echtes Payment-Gateway (neue externe Integration, potenziell ein Deploy-Hazard), Usage-Stats
  `user_module_grants.last_active_at`, das die eigene Migration 000220 als "reserved for a future
  activity-tracking pipeline, stays NULL" dokumentiert — bewusst vertagt, kein uebersehener Gap.
  Volle Herleitung mit Datei:Zeile-Fundstellen im `blocked_reason` der Unit in `BACKLOG.yml`.
- gate: n.a. — keine Codeaenderung, reine Diagnose (BACKLOG.yml/JOURNAL.md sind die einzigen
  geaenderten Dateien dieser Iteration, deshalb auch kein separater Commit — die naechste
  Journal-Record-Iteration des Treibers erfasst den Stand ohnehin).
- verify vorgaenger: sauber. `989ff60f` (fix-calendar-cancel-booking-actor, Iteration 20) geprueft:
  `CancelBooking` liest den Actor jetzt ueber `middleware.GetUserID(ctx)` (Muster aus
  `document_grpc.go`/`dialer_grpc.go` uebernommen, `middleware`-Import war schon vorhanden), kein
  Proto-, Guard- oder Routen-Aenderung, kein Stub, kein gRPC-Bypass. Test
  `calendar_grpc_test.go` deckt den Owner-Check mit einem In-Memory-Repo-Mock ab. Nichts zu
  beanstanden.
- offen:
  - **Fuer Luke/eine FE-Session:** `useBilling.ts` (`useTenant`, `useModuleAssignments`, ggf. die
    Modul-Aktivierung im Billing-Hub) auf die drei existierenden Endpoints
    (`/admin/subscription`, `/admin/license`, `/tenant/module-grants`) umstellen, statt weiter
    gegen `MOCK_*` zu laufen. Invoice-History/Usage-Stats bleiben bewusst Mock, bis
    Payment-Gateway bzw. Activity-Tracking-Pipeline existieren (beides ausserhalb dieses Loops).
  - Alle unveraendert offenen Punkte aus Iteration 17/18/19/20 (Plugin-Grpc-Tenant-aus-Body,
    SetRolePermissions ohne Audit-Event, rbac-format.ts-Katalogluecke, SeedRow/CleanupRow ohne
    `id`-Spalte, nicht regenerierte `types.ts`, `ErrPluginHasInstallations` faellt auf 500 statt
    409, `useTimeline.ts`-Pfad+Offset-Bug, `useResources.ts`-Pfad-Bug) bleiben unveraendert offen,
    hier nicht angefasst.

## Iteration 22 — g-vendor-access — done — 2026-08-03 (siehe Commit-Zeitstempel)

- commit: `d3b7cb01` — feat(security): build vendor access request lifecycle (RBAC R-5 B)
- gebaut: `g-vendor-access` gezogen (`/vendor-access`). Praemisse geprueft und widerlegt: KEIN
  Aussen-Endpoint ohne Login. `desktop/src/renderer/src/api/vendor-access.ts` ruft
  `authenticatedRequest` wie jede andere Admin-API, Header "RBAC R-5 B"; die Typen beschreiben
  GDAP-light v3 — Zentria beantragt zeitlich befristeten Support-Zugang zum Tenant, der Kunde
  (normaler eingeloggter Admin) genehmigt/lehnt ab/schlaegt Alternativtermin vor/entzieht.
  `security:vendor_access:manage` war schon in Migration 000256 geseedet und dem Preset `admin`
  zugeteilt — keine neue Permission-Migration noetig, deutlich kleinerer Deploy-Hazard als die
  urspruengliche Scope-Vermutung.
  Migration 000277: Tabelle `vendor_access_requests` (`tenant_id NOT NULL` + `enable_tenant_rls()`,
  `agents` JSONB, `scope` TEXT[], Status-CHECK ueber alle sieben FE-Zustaende). `security.proto` +
  Regen um 5 RPCs erweitert (List/Approve/Decline/CounterPropose/Revoke), `security_grpc.go` und
  `cmd/auth/main.go` verdrahtet. Neues Paket `internal/security/vendoraccess/` (Repository +
  PostgresRepository + Service) nach dem `gdpr`-Paket-Muster: State-Machine
  (pending/counter_proposed -> active|declined, pending -> counter_proposed, active -> revoked)
  und Sensitive-Scope-Guardrail (`hr_data`/`salary` -> 422 `sensitive_ack_required`, Liste
  haendisch gegen `VENDOR_ACCESS_AREAS` im FE gespiegelt, keine gemeinsame Quelle ueber die
  FE/BE-Grenze — im Code kommentiert). `approved_by`/`revoked_by` werden ueber einen Join auf
  `users` zu Anzeigenamen aufgeloest (Muster aus `p1b-roles-list`), nicht als UUID ausgeliefert.
  Gateway-Routen unter einem EIGENEN Top-Level-Prefix `/api/v1/vendor-access` registriert (nicht
  unter `/api/v1/security/...`), weil das der Pfad ist, den das FE tatsaechlich aufruft — Regel aus
  dem RICHTUNGSENTSCHEID fuer `fix-*-paths`-Units sinngemaess auch hier angewandt (FE ist kanonisch).
  Bewusst NICHT gebaut: kein Create-Endpoint (das FE hat keinen Aufruf dafuer). Die
  Repository-Methode `CreateRequest` existiert fuer Tests/eine kuenftige Zentria-Operator-Anbindung,
  ist aber an keine Route gebunden. Die 15s-Auto-Bestaetigung nach `counter-propose` im MSW-Mock ist
  reine FE-Demo-Simulation; im Backend bleibt eine `counter_proposed`-Anfrage in diesem Zustand, bis
  ein separater Zentria-seitiger Bestaetigungsweg existiert.
- gate: build ok (`go build -p 2 ./internal/models/... ./internal/security/...
  ./internal/gateway/... ./internal/server/... ./cmd/auth/... ./cmd/gateway/...`) | vet ok |
  lint ok (golangci-lint, 0 issues) | test ok — `go test -count=1
  ./internal/security/vendoraccess/...` 15 PASS / 0 SKIP (`DATABASE_URL` gesetzt, Rolle
  `kmuhub_app`), `go test -count=1 ./internal/gateway/` gruen (inkl. TestOpenAPIRouteDrift),
  `go test -count=1 ./internal/server/...` gruen | migration: `migrate up` / `down 1` / `up` alle
  sauber (Kopf 277) | RLS-Smoke (Referenz-Template GATE-COMMANDS.md): eigener Tenant -> 1, fremder
  Tenant -> 0.
  Zwei Bugs beim ersten Testlauf gefunden, beide im TEST-Code (nicht im Repository) und noch in
  dieser Iteration gefixt: (1) ein zweiter `CreateRequest`-Aufruf lief unter `context.Background()`
  ohne Tenant-Kontext und verletzte folgerichtig die WITH-CHECK-Policy (42501) — erwartetes
  Policy-Verhalten, kein Repo-Bug, gefixt via `testutil.WithTenantCtx`. (2) Tenant-Cleanup schlug
  fehl, weil `users.tenant_id` KEIN `ON DELETE CASCADE` hat — der geseedete Pruefer-User muss per
  eigenem `defer` VOR dem Tenant geloescht werden (LIFO beachten; dieselbe Cleanup-Lehre wie
  Iteration 15/16, hier als FK-Reihenfolge- statt Timing-Falle).
- verify vorgaenger: sauber. `e86415ce` (Iteration 21) ist eine reine Diagnose ohne Codeaenderung
  (`g-admin-billing` blocked, nur JOURNAL.md/BACKLOG.yml), nichts zu verifizieren; der letzte
  echte Code-Commit `989ff60f` (Iteration 20) wurde bereits in Iteration 21 verifiziert.
- offen:
  - **Fuer eine spaetere Iteration/Luke:** Zentria-seitiger Bestaetigungsweg fuer
    `counter_proposed` -> `active` fehlt (im Mock nur als 15s-`setTimeout` simuliert); haengt am
    ebenfalls fehlenden Create-Kanal (wie legt Zentria ueberhaupt eine Anfrage an?) — vermutlich ein
    kuenftiges Zentria-Operator-Tool/Platform-API, ausserhalb des Kunden-Vertrags dieser Unit.
  - `sensitiveAreas`-Liste in `internal/security/vendoraccess/service.go` ist eine Handkopie von
    `VENDOR_ACCESS_AREAS` (FE) — bei einer FE-Aenderung an den sensitiven Areas muss diese Liste
    manuell nachgezogen werden, es gibt keine gemeinsame Quelle.
  - Alle unveraendert offenen Punkte aus Iteration 17/18/19/20/21 (Plugin-Grpc-Tenant-aus-Body,
    SetRolePermissions ohne Audit-Event, rbac-format.ts-Katalogluecke, SeedRow/CleanupRow ohne
    `id`-Spalte, nicht regenerierte `types.ts`, `ErrPluginHasInstallations` faellt auf 500 statt
    409, `useTimeline.ts`-Pfad+Offset-Bug, `useResources.ts`-Pfad-Bug, `useBilling.ts` auf reale
    Endpoints umstellen) bleiben unveraendert offen, hier nicht angefasst.

## Iteration 23 — fix-einkauf-po-total — done — 2026-08-03 (siehe Commit-Zeitstempel)

- commit: `45b5331d` — fix(einkauf): backfill stale zero purchase-order totals
- gebaut: `fix-einkauf-po-total` gezogen. Praemisse (Kopfbetrag wird nie berechnet) war VERALTET:
  `e91cdf2a` aus Lauf 3 (2026-07-26) hatte `RecomputePOTotal` bereits gebaut und in
  `AddPOLine`/`UpdatePOLine`/`DeletePOLine` verdrahtet, inklusive Tests fuer alle drei
  Zeilen-Operationen (`service_test.go`) und SQL-seitiger Dezimalrechnung (`SUM(quantity*unit_price)`,
  kein float64). Die Lauf-4-Recherche gegen `backend-gaps.md` hat diesen Fix uebersehen und die Unit
  erneut angelegt — derselbe Fehlklassen-Typ wie die `blocked`-Diagnosen in Iteration 19/21, hier
  aber als `todo`-Dublette statt als offensichtlicher Fehltreffer, weil die Praemisse zum
  urspruenglichen Schreibzeitpunkt (vor dem 26.07.) tatsaechlich richtig war.
  Einzige echte Luecke war das `done_when`-Kriterium "Backfill-Entscheidung begruendet und
  umgesetzt": `RecomputePOTotal` wirkt nur bei kuenftigen Zeilen-Mutationen, Bestellungen deren
  Zeilen vor dem 26.07. angelegt und seither nie wieder angefasst wurden, blieben bei
  `total_amount=0` stehen. Migration `000278_backfill_po_total_amount` (Vorlage
  `000133_backfill_finance_line_tables`) recomputet einmalig ALLE Bestellungen mit derselben Formel,
  idempotent ueber einen `WHERE total_amount <> COALESCE(SUM(...),0)`-Guard. `down.sql` ist bewusst
  ein No-op (Vorlage `000223_validate_tenant_fks.down.sql`) — die alten Werte waren falsch, es gibt
  nichts Sinnvolles, wohin man zurueckkehren koennte.
- gate: build ok (`go build -p 2 ./internal/einkauf/... ./internal/gateway/... ./cmd/einkauf/...
  ./cmd/gateway/...`) | vet ok | lint ok (golangci-lint, 0 issues) | test ok — `go test -count=1
  ./internal/einkauf/...` gruen (Bestandstests aus `e91cdf2a`, unveraendert) | migration:
  `migrate up` (277->278) / `down 1` (278->277) / `up` (278) alle sauber. Lokal 0 Zeilen in
  `purchase_orders`/`po_lines` — Backfill nur rechnerisch verifiziert (SQL-Logik + idempotenter
  Re-Lauf), nicht gegen echte Bestelldaten. Keine Routen-/Proto-Aenderung, deshalb
  `go test ./internal/gateway/` nicht Teil dieses Gates.
- verify vorgaenger: sauber. `d3b7cb01` (g-vendor-access, Iteration 22) geprueft: Handler laufen
  durchgaengig ueber `securityv1.SecurityServiceClient` (kein Gateway-Bypass), Service loest
  `tenant_id` in jeder Methode ueber `middleware.GetTenantID(ctx)` auf, Repository scopet jede
  Query zusaetzlich per `WHERE tenant_id = $1` (Defense-in-depth neben RLS), Permission-Seed
  `security:vendor_access:manage` bestand schon in Migration 000256 und ist dem `admin`-Preset
  zugeteilt (`RequirePermission("security:vendor_access","manage")` passt exakt), DI-Wiring in
  `cmd/auth/main.go` korrekt, `codes.OutOfRange` mappt in `gateway/helpers.go` auf HTTP 422 wie im
  Journal behauptet. Nichts zu beanstanden.
- offen:
  - Alle unveraendert offenen Punkte aus Iteration 17/18/19/20/21/22 (Plugin-Grpc-Tenant-aus-Body,
    SetRolePermissions ohne Audit-Event, rbac-format.ts-Katalogluecke, SeedRow/CleanupRow ohne
    `id`-Spalte, nicht regenerierte `types.ts`, `ErrPluginHasInstallations` faellt auf 500 statt
    409, `useTimeline.ts`-Pfad+Offset-Bug, `useResources.ts`-Pfad-Bug, `useBilling.ts` auf reale
    Endpoints umstellen, Zentria-seitiger Bestaetigungsweg fuer `counter_proposed`,
    `sensitiveAreas`-Handkopie ohne gemeinsame Quelle) bleiben unveraendert offen, hier nicht
    angefasst.

## Iteration 24 — g-fuhrpark-license-check — done — 2026-08-03

- commit: `81212e69` — feat(fuhrpark): add driver license compliance check (Fuehrerscheinkontrolle)
- verify vorgaenger: sauber. `45b5331d` (fix-einkauf-po-total, Iteration 23) gegen die
  Fehlerklassen geprueft: Migration 000278 recomputet mit exakt derselben Formel wie
  `RecomputePOTotal` (`postgres_repository.go:390-405`, `SUM(quantity*unit_price)`, kein float64),
  idempotenter `WHERE total_amount <> COALESCE(...)`-Guard verifiziert, `down.sql` ist ein
  begruendeter No-op (alte Werte waren falsch). Keine neue Route/kein Proto, `go test
  ./internal/gateway/` war zu Recht nicht Teil des Vorgaenger-Gates. Nichts zu beanstanden.
- gebaut: `g-fuhrpark-license-check` gezogen (erste `status: todo`-Unit in Datei-Reihenfolge,
  `deps: []`). Praemisse gegengeprueft: kein FE-Vertrag fuer eine Fuehrerscheinkontrolle
  (`desktop/src/renderer/src` kennt nur `license_plate`, keine Person-bezogene Fuehrerschein-Spur)
  — reine Backend-Unit.
  Design: `driver_licenses` als Historie (append-only, eine Zeile pro Kontrolle — analog
  `vehicle_documents`/`vehicle_services`, nicht ein Status-Feld pro Fahrer), `driver_id`
  referenziert `users(id)`. `CreateDriverLicense` fuegt ueber `INSERT ... SELECT ... FROM users
  WHERE id=$driver AND tenant_id=$tenant` ein (Vorlage: `AssignUserRole`,
  `auth/postgres_repository.go:641`) — verhindert, dass ein Tenant einen fremden User als "Fahrer"
  referenziert, obwohl `driver_licenses` selbst schon `enable_tenant_rls()` traegt. Migration
  000279 (naechste freie Nummer zur Laufzeit ermittelt: Kopf war 278). Vier neue RPCs
  (List/Create/Update/Delete) in `fuhrpark.proto` + Regenerierung im selben Commit (`make`-Target
  fuer Fuhrpark existiert nicht im Makefile, exakter `protoc`-Befehl der Nachbar-Targets manuell
  nachgebaut). Service/Repository/gRPC-Server/Gateway-Route nach dem `VehicleDocument`-Muster,
  Route liegt top-level unter `/api/v1/fuhrpark/driver-licenses` (nicht unter `/vehicles/{id}/`,
  weil ein Fahrer keinem einzelnen Fahrzeug gehoert). Permission `fuhrpark:license:read/write`,
  Seed + Grant an `admin` — gleiches admin-only-Muster wie `fuhrpark:document` (Migration 000196),
  bewusst nicht auf manager/member erweitert (kein FE-Zwang, kein Praezedenzfall dafuer).
  gate: build ok (`go build -p 2 ./internal/fuhrpark/... ./internal/gateway/...
  ./internal/server/... ./cmd/fuhrpark/... ./cmd/gateway/...`) | vet ok | lint ok
  (golangci-lint, 0 issues) | test ok — `go test -count=1 ./internal/fuhrpark/...` 31 PASS /
  0 SKIP (`DATABASE_URL` gesetzt, Rolle `kmuhub_app`), `go test ./internal/gateway/` gruen inkl.
  `TestOpenAPIRouteDrift` (789/791 nach den vier neuen Pfaden), `go test ./internal/server/...`
  gruen | migration: `up`/`down 1`/`up` sauber (278->279->278->279) | RLS-Smoke: eigener Tenant 1,
  fremder Tenant 0, Cross-Tenant-Insert-Versuch (Fahrer aus Tenant A, Aufrufer behauptet Tenant B)
  liefert 0 Zeilen (SQL UND Go-Test) | `TestAllPublicTablesHaveRLSOrAreAllowlisted` bleibt gruen,
  kein neuer Fund durch die neue Tabelle.
  Tests: neue Datei `driver_license_test.go` (6 Subtests: Cross-Tenant-Create-Ablehnung, Isolation,
  List-Filter nach Fahrer, Update, Update/Delete ueber Tenant-Grenze als No-op) +
  `tenant_write_test.go` um den `driver_licenses`-Fall erweitert (echter Schreibpfad statt
  `testutil.SeedRow`, Cleanup-Reihenfolge per `defer`-LIFO: License vor User).
- offen:
  - Bewusst nicht gebaut (steht so in den notes der Unit): keine Ablauf-Erinnerungen/
    Benachrichtigungen fuer faellige Kontrollen — gehoert dem notification-Modul, als
    eigenstaendige Folge-Unit vormerken, nicht Teil dieser Unit.
  - Kein FE-Wiring (Backend-Loop fasst `desktop/` nicht an) — `driver-licenses`-UI im
    Fuhrpark-Modul existiert noch nicht, waere eine Folge-Session.
  - Permission bewusst nur an `admin` vergeben, nicht an `manager` — falls Fuhrpark-Manager das
    im Alltag pflegen sollen, ist das eine Produktentscheidung fuer eine spaetere Migration, kein
    Uebersehen.
  - Alle unveraendert offenen Punkte aus Iteration 17/18/19/20/21/22 (siehe deren Aufzaehlung oben)
    bleiben unveraendert offen, hier nicht angefasst.

## Iteration 25 — fix-security-gdpr-paths — done — 2026-08-03

- commit: `a8ac8fc2` — fix(security): align GDPR export routes with the frontend contract
- verify vorgaenger: sauber. `81212e69` (g-fuhrpark-license-check, Iteration 24) gegen die
  Fehlerklassen geprueft: Handler gehen durchgaengig ueber `fuhrparkv1.FuhrparkServiceClient`
  (kein Direct-Svc-Bypass), Migration 000279 traegt `tenant_id NOT NULL` + `enable_tenant_rls()`,
  `CreateDriverLicense` loest den Fahrer zusaetzlich per `INSERT ... SELECT ... FROM users WHERE
  id=$driver AND tenant_id=$tenant` auf (Defense-in-Depth ueber die RLS-Policy hinaus, verifiziert
  in `postgres_repository.go`), Permission-Seed fuer `fuhrpark:license:read/write` steht in
  derselben Migration. Nichts zu beanstanden.
- gebaut: `fix-security-gdpr-paths` gezogen (erste `status: todo`-Unit in Datei-Reihenfolge,
  `deps: []` — alle Units vor Block "fix-*" waren bereits `done`/`blocked`).
  Vier Client-Pfade aus `security-client.ts` liefen ins Leere:
  `POST /gdpr/export/request` (BE hatte nur `/gdpr/export`), `POST /gdpr/export/{id}/approve|deny`
  (BE hatte `/gdpr/exports/{id}/...`, Plural), `GET /gdpr/export/{id}/download` (BE hatte
  `/gdpr/download/{token}`). Wie in den notes vorgegeben zieht das Gateway auf die FE-Pfade um,
  nicht umgekehrt.
  Route-Umbau in `route_security.go`: `/gdpr/export` ist jetzt eine eigene Route-Gruppe mit
  `POST /request` und `Route("/{id}", ...)` fuer `approve`/`deny`/`download`. `/gdpr/exports`
  (Liste) unveraendert.
  Wichtiger Nebenfund beim Bauen: chi erlaubt an derselben Tree-Position keine zwei
  verschieden benannten Wildcards — empirisch mit einem Wegwerf-Programm verifiziert
  (`chi.Mux.Route` wirft `panic("attempting to Mount() a handler on an existing path")`, wenn
  `/{id}/approve` und `/{token}/download` unter demselben Elternknoten registriert werden). Die
  drei Sub-Routen unter `/export/{id}/...` nutzen deshalb durchgaengig `{id}`, auch fuer den
  Download — das ist inhaltlich korrekt, weil `HandleGetExportDownload` ohnehin ausschliesslich
  ueber `DownloadToken` aufloest, nie ueber eine Export-ID (beantwortet die "id vs. token"-Frage
  aus den notes: es gibt keine Alternative, der Handler kennt nur den Token). Der Golang-Handler
  liest jetzt `chi.URLParam(r, "id")` und behandelt den Wert als Token, mit erklaerendem Kommentar
  im Code gegen Verwechslung.
  Fuenfter FE-Aufruf `PrivacySettingsTab.tsx:139` (`/api/v1/gdpr/exports/${exp.id}/download?token=
  ...`, roher `<a href>` ohne API-Client, kein `/security`-Praefix) wie in den notes vermutet echt
  tot — trifft keine der vier neuen Routen, auch keine alte. NICHT gefixt (Frontend-Datei, siehe
  Grenzen des Loops), bleibt offener FE-Befund unten.
  `openapi.yaml` an allen vier Stellen mitgezogen (Pfad-Rename, Parametername bei `download` von
  `token` auf `id` mit Beschreibung "One-time download token (not the export request ID)").
  Tests: zwei neue Router-Level-Tests (`TestGDPRExportRoutes_MatchFrontendPaths`,
  `TestGDPRExportRoutes_ApproveDenyRequireAdmin`) bauen den echten Chi-Router ueber
  `RegisterRoutes` und schicken echte Requests gegen die vier FE-Pfade — bewusst kein reiner
  Handler-Aufruf, weil ein Handler-Test einen Pfad-Tippfehler nie haette fangen koennen (503 bei
  leerem Registry beweist Pfad-Treffer, 404 waere der Regressionsfall). Dazu ein Unit-Test fuer den
  Token-Vertrag (`TestGDPRExportRoutes_DownloadUsesTokenNotID`). Neuer Helper `withRoles()` in
  `testutil_test.go` fehlte bisher fuer `RequireRole`-Tests (Vorlage `withPermissions` aus
  `route_capability_guard_test.go`, aber ueber `middleware.UserRolesKey`).
  gate: build ok (`go build -p 2 ./...`) | vet ok | lint ok (golangci-lint auf
  `internal/gateway`, 0 issues, nach dem Test-Fix erneut auf dem finalen Stand gelaufen) | test ok
  — `go test -count=1 ./internal/gateway/...` gruen inkl. `TestOpenAPIRouteDrift` und
  `TestOpenAPISpecDrift` (Pfadzahl unveraendert: vier Umbenennungen, keine neue Route). Keine
  Migration, kein `.proto`, kein neuer Permission-Guard (`RequireRole("admin")` bestehend, nur
  mitverschoben) — DB/RLS-Gate daher nicht einschlaegig.
  Eigener Fehlversuch dabei: der erste Entwurf von `TestGDPRExportRoutes_DownloadUsesTokenNotID`
  nutzte `emptyRegistry()` und erwartete 400, bekam aber 503 — `HandleGetExportDownload` prueft den
  gRPC-Client-Connect VOR dem Token, wie alle anderen Handler in der Datei. Mit
  `registryWithService("auth")` behoben; im Journal vermerkt, weil derselbe Fallstrick jedem
  weiteren Test in dieser Datei droht.
- offen:
  - Der praefixlose `PrivacySettingsTab.tsx:139`-Downloadlink (siehe oben) ist ein echter,
    unabhaengiger FE-Bug — eigene Folge-Unit oder FE-Session, nicht Teil des Backend-Loops.
  - Alle unveraendert offenen Punkte aus Iteration 17/18/19/20/21/22/24 (siehe deren Aufzaehlung
    oben) bleiben unveraendert offen, hier nicht angefasst.

## Iteration 26 — fix-hr-document-paths — done — 2026-08-03

- commit: `5359d87b` — fix(hr): move document categories route to the frontend contract
- verify vorgaenger: sauber. `a8ac8fc2` (fix-security-gdpr-paths, Iteration 25) gegen die
  Fehlerklassen geprueft: kein Direct-Svc-Bypass (Route-Umbau bleibt auf Handler-Ebene, ruft weiter
  `HRServiceClient`/`AuthServiceClient` unveraendert), kein Stub, kein `.proto` ohne Regen (keins
  angefasst), kein neuer `RequirePermission` (bestehender `RequireRole("admin")` nur mitverschoben),
  keine neue Tabelle/RLS-Luecke, Wire-Shape passt (Pfad-Rename, kein Response-Formatwechsel), Route
  in openapi.yaml mitgezogen, kein Alt-Key ersetzt. Nichts zu beanstanden.
- gebaut: `fix-hr-document-paths` gezogen (erste `status: todo`-Unit in Datei-Reihenfolge, `deps: []`).
  (a) `GET /hr/document-categories` (nie aufgerufen) ersetzt durch
  `GET /hr/employees/{id}/documents/categories` (FE-Pfad aus `hr-client.ts:937`).
  Beim Recherchieren: `hr:read` (Migration 000129) ist NUR an die Rolle "admin" vergeben — hr_admin/
  manager/member haetten weder die neue Route noch die bestehende `/documents`-Route je erreicht.
  Guard additiv um die produktiv bereits vorgesehene Capability `team:documents:view` erweitert
  (`RequirePermissionAny`, gleiches Muster wie die `zeitTeamView`-Guards weiter oben in
  `route_hr.go` — `hr:read` bleibt gueltig, kein Alt-Key ersetzt).
  Visibility-Filterung nutzt das bereits vorhandene, bislang ungenutzte
  `middleware.PermissionScope(ctx,"team:documents","view")` (own/team/all) — exakt der Scope, den
  `PersonnelDocuments.tsx` bereits clientseitig auswertet (`useCapability('team:documents:view').scope`).
  Migration 000256 belegt scope='all' fuer admin/hr_admin, scope='team' fuer manager, scope='own'
  fuer member — kein neuer Rollen-Mapping-Mechanismus noetig, nur der erste echte Aufrufer der
  schon vorhandenen Bausteine. Filterung laeuft in Go im Service (`filterCategoriesByScope`), nicht
  in SQL (Kategorien sind eine Handvoll Zeilen, ein SQL-Filter waere unnoetige Komplexitaet).
  Unbekannter/leerer Scope faellt restriktiv auf "nur employee-visibility" zurueck, nicht auf "alles".
  (b) `personnel-documents` NICHT gebaut, wie in den notes vorgesehen: `PersonnelDocuments.tsx` ist
  durchgaengig Demo-Zustand — `handleUpload` sendet weder `employeeId` noch ein echtes File (nur
  Groesse/Name als Text), der GET-Query laedt von `/hr/personnel-documents` ohne jeden
  Employee-Bezug. Ein Endpoint dafuer waere blind auf den eingeloggten User zurueckgefallen — genau
  das Datenleck-Risiko, vor dem die notes warnen. Bleibt offener, eigenstaendiger FE-Befund.
  Proto: `ListDocumentCategoriesReq` um `caller_scope` erweitert, `make proto-hr`-Aequivalent
  (protoc direkt, `make` ist auf dieser Maschine nicht im PATH) im selben Commit regeneriert.
  `hr_grpc.pb.go` kam byte-identisch zurueck (nur CRLF/LF-Rauschen) — bewusst NICHT mitcommittet,
  um keinen Nur-Zeilenenden-Diff zu erzeugen.
  Tests: 4 neue Unit-Tests in `service_test.go` (Scope all/team/own/leer — leer/unbekannt faellt
  restriktiv auf employee-only zurueck, nicht auf "alles"), 4 neue Router-Tests in
  `route_hr_test.go` (Pfad erreichbar mit `team:documents:view` ODER `hr:read` allein, 403 ohne
  beides, alte Route liefert jetzt 404).
  gate: build ok (`go build -p 2 ./internal/biz/hr/... ./internal/gateway/... ./internal/server/...
  ./cmd/gateway/... ./cmd/biz/...`) | vet ok | lint ok (golangci-lint, 0 issues) | test ok —
  `go test -count=1 ./internal/biz/hr/... ./internal/gateway/... ./internal/server/...` gruen mit
  `DATABASE_URL` gesetzt auf `kmuhub_app` (u.a. `internal/biz/hr/timetracking` DB-Tests real
  gelaufen, keine Skips beobachtet), inkl. `TestOpenAPIRouteDrift`/`TestOpenAPISpecDrift`. Keine
  Migration (keine neue Tabelle/Spalte), daher kein migrate-Schritt und keine RLS-Smoke noetig.
- offen:
  - FE-Befund: `PersonnelDocuments.tsx` (`handleUpload`/GET-Query) ist komplett demo-/mock-artig
    ohne Employee-Bezug — eigene Folge-Unit oder FE-Session, kein Backend-Gap.
  - Alle unveraendert offenen Punkte aus Iteration 17-22/24/25 (siehe deren Aufzaehlung oben)
    bleiben unveraendert offen, hier nicht angefasst.

## Iteration 27 — fix-finance-notification-paths — done — 2026-08-03

- commit: `fbecfafd` — fix(gateway): move finance and notification routes to frontend contract
- verify vorgaenger: sauber. `5359d87b` (fix-hr-document-paths, Iteration 26) gegen die
  Fehlerklassen geprueft: Handler geht ueber `h.getHRClient()` (kein Direct-Svc-Bypass), kein
  Stub, `hr.proto` + `hr.pb.go` im selben Commit regeneriert, Guard additiv
  (`RequirePermissionAny("hr","read" / "team:documents","view")`, kein Ersetzen des Alt-Keys,
  `team:documents:view` seit Migration 000256 bereits vergeben — kein ungeseedeter Guard), keine
  neue Tabelle/RLS-Luecke, Wire-Shape unveraendert (weiterhin Liste, nur serverseitig gefiltert).
  `filterCategoriesByScope` bestaetigt restriktiv: nur `ScopeAll` sieht alles, `ScopeTeam` addiert
  Manager-Sichtbarkeit, jeder andere/leere Scope faellt auf employee-only zurueck — deckt sich mit
  der Journal-Behauptung der Vorgaenger-Iteration. Nichts zu beanstanden.
- gebaut: `fix-finance-notification-paths` gezogen (naechste `status: todo`-Unit in
  Datei-Reihenfolge, `deps: []`).
  (a) `POST /api/v1/finance/invoices/{id}/pay` -> `/{id}/mark-paid` umbenannt
  (`route_biz.go:116`), Handler `HandleMarkInvoicePaid`/Guard/Service unveraendert — reine
  Pfad-Umbenennung auf den bereits im FE-Client (`finance-client.ts:229`) verwendeten Namen.
  Einzige betroffene Testzeile war `route_capability_guard_test.go` ("invoice pay, catalogue
  edit key maps to it"), auf den neuen Pfad gezogen.
  (b) `DELETE /api/v1/notifications/mutes` (body-basiert, `{module_id,resource_id}`, kein
  FE-Aufrufer) ersetzt durch `DELETE /api/v1/notifications/mutes/{muteId}` — der FE-Client
  (`notification-client.ts:103`) ruft bereits `unmute(muteId)` auf diesen Pfad.
  Proto `UnmuteResourceRequest` von `module_id`+`resource_id` auf ein einziges `mute_id`
  umgebaut (einziger Aufrufer war genau dieser Handler, keine Wire-Kompatibilitaet zu wahren),
  `protoc`-Notification-Block (aus dem generischen `proto:`-Target im Makefile, `make` selbst
  ist auf dieser Maschine nicht im PATH) im selben Commit regeneriert — nur `notification.pb.go`
  geaendert, `notification_grpc.pb.go` byte-identisch, weil die RPC-Signatur gleich blieb.
  Repository/Service/gRPC-Handler durchgaengig auf
  `DeleteMute(ctx, tenantID, userID, muteID uuid.UUID)` umgestellt, SQL jetzt
  `WHERE id = $1 AND tenant_id = $2 AND user_id = $3`. `notification_mutes` hat seit Migration
  000124 bereits RLS (tenant-gescopt) — die zusaetzliche `user_id`-Klausel ist die eigentliche
  Luecke, die RLS NICHT abdeckt (RLS ist tenant-, nicht user-scoped): ohne sie haette ein Nutzer
  per erratener/erschnueffelter Mute-ID einen fremden Mute im selben Tenant loeschen koennen.
  Handler nutzt jetzt `validateUUIDParam(w,r,"muteId")` statt Body-Decode, `unmuteResourceRequest`
  entfernt.
  Zwei Mock-Repos implementieren dasselbe `preference.Repository`-Interface
  (`notification/preference/service_test.go`, `notification/notification/service_test.go`) und
  mussten beide auf die neue `DeleteMute`-Signatur gezogen werden.
  Neuer Test `TestUnmuteResourceWrongUserRejected`: gleicher Tenant, fremde UserID ->
  `ErrMuteNotFound`, Mute bleibt in der Liste — deckt den in den notes geforderten
  Fremdzugriffsfall auf Service-Ebene ab. Cross-Tenant-Abdeckung laeuft ueber die bestehende
  RLS + `TestTenantIsolation_Notifications` (tenant_isolation_phase2_test.go), nicht dupliziert.
  openapi.yaml: Pfad-Rename fuer (a); fuer (b) der alte `delete:`-Block unter
  `/api/v1/notifications/mutes` entfernt und als eigener Pfad
  `/api/v1/notifications/mutes/{muteId}` mit Pfad-Parameter (statt requestBody) neu angelegt,
  inkl. 404-Antwort.
  gate: build ok (`go build -p 2 ./...`) | vet ok | lint ok (golangci-lint auf
  `internal/gateway`, `internal/notification/...`, `internal/server`, 0 issues) | test ok —
  `go test -count=1 ./internal/gateway/... ./internal/notification/... ./internal/server/...`
  gruen mit `DATABASE_URL` gesetzt auf `kmuhub_app`, inkl. `TestOpenAPIRouteDrift`/
  `TestOpenAPISpecDrift`. Keine Migration, keine neue Tabelle/Spalte, kein neuer
  `RequirePermission`-Guard (bestehende `notifications:write`/`finance:invoice:edit` nur
  mitverschoben) — kein DB/RLS-Gate ueber die bestehende Migration 000124 hinaus noetig.
- offen:
  - Alle unveraendert offenen Punkte aus Iteration 17-22/24/25/26 (siehe deren Aufzaehlung oben)
    bleiben unveraendert offen, hier nicht angefasst.

## Iteration 28 — fix-crm-import-company — done — 2026-08-03

- commit: `6ff1509d` — fix(crm): persist company relation on contact import/export
- verify vorgaenger: sauber. `fbecfafd` (fix-finance-notification-paths, Iteration 27) gegen die
  Fehlerklassen geprueft: `HandleUnmuteResource` geht ueber `n.getNotificationClient()` (kein
  Direct-Svc-Bypass), kein Stub, `notification.proto` + `.pb.go` im selben Commit regeneriert
  (`.grpc.pb.go` byte-identisch, RPC-Signatur unveraendert), Guard unveraendert
  (`RequirePermission("notifications","write")` bestand schon vorher — reine Pfad-Aenderung, kein
  neuer Key, kein Seed noetig), `DeleteMute` ist jetzt zusaetzlich zur RLS-Tenant-Policy explizit
  auf `user_id` gescopt (die von RLS nicht abgedeckte Luecke), Wire-Shape unveraendert (kein
  Response-Body-Bruch). openapi.yaml-Diff deckt sich mit dem Code (Pfad-Rename + neuer
  `{muteId}`-Pfad mit 404). Nichts zu beanstanden.
- gebaut: `fix-crm-import-company` gezogen (naechste `status: todo`-Unit in Datei-Reihenfolge,
  `deps: []`).
  Root Cause bestaetigt wie im scope-Text: `importSingleContact`
  (`internal/email/contact/import_service.go`) persistierte `fields["company"]` nie, und der
  CSV-Export (`export_service.go`) gab die Spalte "company" hart als Leerstring aus
  ("Company name is not on the contact model directly, so we leave empty for now"). vCard-Export
  schrieb `vcard.FieldOrganization` ueberhaupt nicht — dieselbe Luecke, nur ohne eigenen Kommentar,
  im selben Zug mitgefixt, sonst waere der Round-Trip nur fuer CSV geschlossen gewesen.
  Firma laeuft ueber die company-Relation (`contacts.company_id` -> `companies`), nicht ueber ein
  neues Freitextfeld. Neu in `internal/crm/company/`: `Repository.GetByName` (case-insensitiv per
  `LOWER(name) = LOWER($2)`, `merged_into_id IS NULL`, tenant-gescopt — eine gemergte Firma darf
  nicht erneut getroffen werden) und `Repository.GetNamesByIDs` (Batch, `id = ANY($1) AND
  tenant_id = $2`), darauf `Service.FindOrCreateByName` und `Service.GetNamesByIDs`. Namensvergleich
  bewusst getrimmt+case-insensitiv (Entscheidung wie in den notes gefordert): `company.Create`
  trimmt den Namen schon vor dem Schreiben, die DB-Seite vergleicht also immer getrimmt gegen
  getrimmt — Trimmen selbst ist Aufgabe der Service-Schicht, nicht des Repositories (analog zu
  `GetByID`, das ebenfalls keine Normalisierung macht).
  `ContactProvider`-Interface (gemeinsam von Import UND Export genutzt) um
  `FindOrCreateCompany(ctx, name, createdBy) (uuid.UUID, error)` und
  `GetCompanyNames(ctx, ids) (map[uuid.UUID]string, error)` erweitert. `TenantScopedAdapter`
  bekommt einen zweiten Konstruktor-Parameter `TenantedCompanyService` — alle vier Aufrufstellen in
  `internal/server/crm_grpc.go` (Import CSV/VCard, Export CSV/VCard) auf
  `NewTenantScopedAdapter(s.contactService, s.companyService, tenantID)` gezogen, `companyService`
  war als Feld auf `CRMGRPCServer` schon vorhanden.
  N+1 beidseitig vermieden: Import cached aufgeloeste Firmennamen in einer lokalen
  `map[string]uuid.UUID`, angelegt PRO LAUF in `ImportCSV`/`ImportVCard` (nicht am `ImportService`
  selbst, der ueber mehrere Requests hinweg als Singleton geteilt sein kann) —
  `TestImportCSV_CompanyDedupedWithinRun` beweist genau 1 `FindOrCreateCompany`-Aufruf fuer 2
  Zeilen derselben Firma trotz Gross-/Kleinschreibungs- und Leerraum-Unterschied. Export holt alle
  benoetigten Namen in einem `GetCompanyNames`-Batch-Aufruf statt pro Kontakt.
  Merge-Pfad (`mergeByEmail=true`) uebernimmt die Firma nachtraeglich nur, wenn der bestehende
  Kontakt noch keine hat (`existing.CompanyID == nil`) — dieselbe "nur leere Felder auffuellen"-
  Semantik wie bei Telefon/Position/Notizen im selben Codepfad.
  Idempotenz (notes-Forderung: zweiter Lauf derselben Datei erzeugt keine Firmen-Dublette) folgt
  direkt aus `FindOrCreateByName`: der zweite Lauf startet mit leerem Cache, findet die beim ersten
  Lauf angelegte Firma aber ueber `GetByName` wieder — bewiesen durch
  `TestService_FindOrCreateByName_FindsCaseInsensitive` (zwei unabhaengige Aufrufe gegen denselben
  Service/Repository, zweiter liefert dieselbe ID, `repo.companies` waechst nicht).
  Tests: `TestExportImportRoundTrip_CompanyCSV` und `TestExportVCard_CompanyRoundTrip` exportieren
  in einen Provider und importieren die Bytes in einen ZWEITEN, frischen Provider (kein geteilter
  Firmen-Cache zwischen den beiden) — das beweist den Fix ueber den reinen Prozess-Cache hinaus,
  echter Export-dann-Import-Beweis wie in done_when gefordert. Dazu 4 weitere Import-Tests
  (Create-Fall, Dedupe-Fall, Merge-Fuellt-Luecke-Fall, Leer-Firma-faellt-nicht-auf) und 8 neue
  Service-Tests fuer `FindOrCreateByName`/`GetNamesByIDs` (Create-wenn-fehlend, case-insensitiv
  finden, gemergte Firma ignorieren, Tenant-Isolation, Pflichtfeld-Validierung) plus ein
  Repository-Test gegen die echte DB (`TestCompanyGetByName_CaseInsensitiveIgnoresMergedAndOtherTenant`,
  neu in `tenant_write_test.go`, demselben Muster wie `TestCompanyWrites_LandInCallerTenant`
  folgend: Merge-Konstellation via echtem `merged_into_id`-FK, Tenant-Fremdzugriff liefert
  `ErrCompanyNotFound` bzw. eine leere Batch-Antwort).
  NEUER FUND, nicht in dieser Unit behoben (ausserhalb des Scopes, eigene Unit noetig): in
  `internal/server/email_grpc.go` rufen `ImportContactsCSV`/`ImportContactsVCard`
  `s.importService.ImportCSV/VCard` DIREKT auf dem in `cmd/email/main.go` mit `nil`-`ContactProvider`
  konstruierten Singleton auf — anders als der CRM-Pfad baut der Email-gRPC-Server KEINEN
  Tenant-scoped Adapter pro Request. Jeder Aufruf ueber `POST /api/v1/email/.../import/csv` bzw.
  `/import/vcard` (`route_email.go:198-199`) waere schon vor dieser Unit bei JEDEM
  `contactProvider`-Zugriff (`GetByEmail`, `CreateForImport`, ...) mit einer Nil-Pointer-Panik
  geendet, nicht erst durch die neuen `FindOrCreateCompany`/`GetCompanyNames`-Aufrufe — ein
  bestehender, durch diese Unit nur sichtbar gewordener Fund. Braucht Tenant-ID-Extraktion +
  Pro-Request-Adapter analog `crm_grpc.go`, eigene Iteration.
  gate: build ok (`go build -p 2 ./...`) | vet ok (`go vet -p 2 ./...`) | lint ok (golangci-lint auf
  `internal/email/contact`, `internal/crm/company`, `internal/crm/contact`, `internal/server`,
  0 issues) | test ok — `go test -count=1 ./internal/crm/company/... ./internal/crm/contact/...
  ./internal/email/contact/... ./internal/server/...` gruen mit `DATABASE_URL` gesetzt auf
  `kmuhub_app`, 0 Skips in den beruehrten Paketen (verifiziert per `-v | grep -c "^--- SKIP"`).
  Keine Migration (`companies` hatte RLS schon aus Migration 000120, keine neue Tabelle/Spalte),
  keine neue Route (kein openapi.yaml-Eintrag noetig), kein neuer `RequirePermission`-Guard.
- offen:
  - NEU: `internal/server/email_grpc.go` `ImportContactsCSV`/`ImportContactsVCard` nutzen den
    nil-Provider-Singleton direkt statt eines Pro-Request-Adapters — Nil-Pointer-Panik bei jedem
    echten Aufruf, unabhaengig von dieser Unit vorbestehend. Siehe Fund oben, eigene Unit.
  - Alle unveraendert offenen Punkte aus Iteration 17-22/24/25/26/27 (siehe deren Aufzaehlung oben)
    bleiben unveraendert offen, hier nicht angefasst.

## Iteration 29 — fix-email-import-nil-provider-panic — done — 2026-08-03

- commit: `37f561ad` — fix(email): stop panicking on nil-provider contact import
- verify vorgaenger: sauber. `6ff1509d` (fix-crm-import-company, Iteration 28) gegen die acht
  Fehlerklassen geprueft: keine `.proto`-Aenderung (N/A Klasse 3), keine neue Route/kein Guard
  (Klasse 4/6/7/8 N/A, `crm_grpc.go`-Diff ruehrt keinen Gateway-Handler an), kein Stub (Klasse 2 —
  `GetByName`/`GetNamesByIDs` vollstaendig implementiert und getestet), kein gRPC-Layer-Bypass
  (Klasse 1 — `CRMGRPCServer` IST die gRPC-Implementierung, `s.contactService`/`s.companyService`
  direkt aufzurufen ist hier die korrekte Schicht, kein Gateway-Handler drumherum). Tenant-Luecke
  (Klasse 5) explizit gegengeprueft: `GetByName` filtert `WHERE tenant_id = $1`, `GetNamesByIDs`
  filtert `id = ANY($1) AND tenant_id = $2`, `TenantScopedAdapter.FindOrCreateCompany`/
  `GetCompanyNames` reichen `a.tenantID` durch — keine ungescopte Query. Nichts zu beanstanden.
- gebaut: `fix-email-import-nil-provider-panic` gezogen (naechste `status: todo`-Unit in
  Datei-Reihenfolge, `deps: []`, direkt der von Iteration 28 selbst angelegte Fund).
  Root Cause bestaetigt wie im scope-Text: `cmd/email/main.go` konstruierte `importService :=
  emailcontact.NewImportService(nil, slog.Default())` und `EmailGRPCServer.ImportContactsCSV`/
  `ImportContactsVCard` riefen diesen Singleton direkt auf — jeder echte Aufruf haette im
  `contactProvider`-Zugriff (`GetByEmail`/`CreateForImport`) eine Nil-Pointer-Panik geworfen, nicht
  erst seit Iteration 28, aber durch deren `FindOrCreateCompany`/`GetCompanyNames`-Aufrufe zuerst
  sichtbar geworden.
  Der Email-Prozess haengt am selben Postgres-Pool wie CRM — kein RPC-Hop noetig.
  `cmd/email/main.go` konstruiert jetzt `contact.NewService(contact.NewPostgresRepository(pool))`
  und `company.NewService(company.NewPostgresRepository(pool))` direkt (Alias-Kollision mit dem
  bereits importierten `emailcontact "internal/email/contact"` durch unqualifizierte Imports fuer
  `internal/crm/contact`/`internal/crm/company` geloest, `crmcontact`/`crmcompany` als Alias in
  `email_grpc.go`) und reicht beide in `NewEmailGRPCServer` durch.
  `ImportContactsCSV`/`ImportContactsVCard` bauen jetzt PRO REQUEST
  `emailcontact.NewTenantScopedAdapter(s.contactService, s.companyService, tenantID)` +
  `emailcontact.NewImportService(provider, nil)`, exakt das Muster aus
  `crm_grpc.go:2092-2144`. `tenantID` kommt aus `middleware.GetTenantID(ctx)` (401 bei Fehlen),
  `ownerID` war wie in den notes vermutet nie ans Auth-Context gekoppelt (`uuid.Nil` mit dem
  Kommentar "Owner determined by auth context", der nie eingeloest wurde) — jetzt
  `uuid.Parse(middleware.GetUserID(ctx))`, ebenfalls 401 bei Fehlen statt der alten Panik.
  Der `importService`-Singleton wurde als Feld/Konstruktor-Param KOMPLETT entfernt statt nur
  umgangen: das Email-Proto hat kein `PreviewContactsCSV`-RPC, also war der Singleton nach dem Fix
  toter Code ohne verbleibenden Aufrufer — Entfernen statt Liegenlassen ist hier der schlankere
  Diff, nicht nur Aufraeumen.
  NEUER FUND, nicht in dieser Unit behoben (identische Fehlerklasse, aber ausserhalb des
  scope-Texts dieser Unit — Import war explizit benannt, Export nicht): `ExportContactsCSV`/
  `ExportContactsVCard` in `email_grpc.go` rufen `s.exportService.ExportCSV`/`ExportVCard` weiterhin
  direkt auf dem `nil`-Provider-Singleton aus `cmd/email/main.go` auf — derselbe Nil-Pointer-Panik-
  Bug, spiegelbildlich. Eigene Unit `fix-email-export-nil-provider-panic` direkt nach dieser
  eingefuegt; sie braucht keine neue Dependency mehr, `contactService`/`companyService` haengen
  bereits am `EmailGRPCServer`.
  Test (neu, `internal/server/email_grpc_import_test.go`, DB-backed nach dem Muster von
  `rbac_audit_events_db_test.go`): `TestEmailImportContactsCSV_DB` und
  `TestEmailImportContactsVCard_DB` fuehren je einen echten Import End-to-End gegen den echten
  `contact.Service`/`company.Service` aus (Kontakt UND Firmenzuordnung landen nachweislich in der
  DB, per Query gegengeprueft — nicht nur ein Mock-Provider wie in den bestehenden
  `import_service_test.go`-Tests), `TestEmailImportContactsCSV_MissingTenant` beweist 401 statt
  Panik bei fehlendem Tenant-Context.
  gate: build ok (`go build -p 2 ./...`) | vet ok (`go vet -p 2 ./...`) | lint ok (golangci-lint auf
  `internal/server`, `cmd/email`, 0 issues) | test ok — `go test -count=1 -v ./internal/server/...
  ./cmd/email/...` gruen mit `DATABASE_URL` gesetzt auf `kmuhub_app`, 0 Skips, 0 Fails (per
  `grep -c "^--- SKIP"`/`"^--- FAIL"` gegengeprueft). Keine Migration, keine neue Route (kein
  openapi.yaml-Eintrag noetig, RPC-Signaturen unveraendert), kein neuer `RequirePermission`-Guard —
  `go test ./internal/gateway/` daher nicht Pflicht, trotzdem kein Routen-Diff im Commit.
- offen:
  - NEU: `ExportContactsCSV`/`ExportContactsVCard` haben denselben nil-Provider-Panik-Bug wie
    Import hatte — eigene Unit `fix-email-export-nil-provider-panic` angelegt, direkt bauen mit
    dem bereits vorhandenen `contactService`/`companyService`.
  - Alle unveraendert offenen Punkte aus Iteration 17-22/24-28 (siehe deren Aufzaehlung oben)
    bleiben unveraendert offen, hier nicht angefasst.
