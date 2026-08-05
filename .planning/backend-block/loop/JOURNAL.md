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

## Iteration 30 — fix-email-export-nil-provider-panic — done — 2026-08-03
- commit: 4a02c5e76778ad69956cb39f53c3a4bb71f4bc26
- verify vorgaenger: sauber. `37f561ad` (fix-email-import-nil-provider-panic, Iteration 29) gegen
  die acht Fehlerklassen geprueft: kein Proto-Diff (Klasse 3 N/A), keine neue Route/kein Guard
  (Klasse 4/6/7/8 N/A), kein Stub (Klasse 2 — beide Import-RPCs vollstaendig implementiert und
  DB-getestet), kein gRPC-Layer-Bypass (Klasse 1 — `EmailGRPCServer` IST die gRPC-Implementierung,
  `s.contactService`/`s.companyService` direkt aufzurufen ist hier korrekt). Tenant-Luecke (Klasse 5)
  explizit gegengeprueft: `TenantScopedAdapter` wird pro Request mit `middleware.GetTenantID(ctx)`
  gebaut, `ownerID` kommt aus `middleware.GetUserID(ctx)`, beide 401 bei Fehlen. Testfalle aus
  Iteration 15 (`t.Cleanup` nach `defer pool.Close()`) nicht wiederholt — der neue Test nutzt
  durchgaengig `t.Cleanup` fuer Pool UND Zeilen, korrekte LIFO-Reihenfolge. Nichts zu beanstanden.
- gebaut: `fix-email-export-nil-provider-panic` gezogen (naechste `status: todo`-Unit in
  Datei-Reihenfolge, `deps: []`, der von Iteration 29 selbst angelegte Spiegel-Fund).
  Root Cause wie im scope-Text: `ExportContactsCSV`/`ExportContactsVCard` in
  `backend/internal/server/email_grpc.go` riefen `s.exportService.ExportCSV`/`ExportVCard` direkt
  auf einem Singleton aus `cmd/email/main.go` auf, konstruiert mit
  `emailcontact.NewExportService(nil, slog.Default())` — jeder echte Aufruf haette im
  `contactProvider.ListByIDs`-Zugriff (`internal/email/contact/export_service.go`) eine
  Nil-Pointer-Panik geworfen.
  Beide RPCs bauen jetzt pro Request `tenantID, err := middleware.GetTenantID(ctx)` (401 bei
  Fehlen) und `emailcontact.NewTenantScopedAdapter(s.contactService, s.companyService, tenantID)` +
  `emailcontact.NewExportService(provider, nil)`, exakt das Muster aus `crm_grpc.go:2146-2224` und
  aus den Import-RPCs (Iteration 29). Anders als der CRM-Export-Pfad hat
  `emailv1.ExportContactsRequest` kein `is_admin`/kein "export ohne IDs"-Feld — `ExportCSV`/
  `ExportVCard` bleiben deshalb bei `req.ContactIds` wie in den notes vorgesehen, kein
  `ExportAllCSV`-Zweig.
  Der `exportService`-Singleton (nil-Provider) wurde komplett entfernt statt nur umgangen: Feld auf
  `EmailGRPCServer`, Konstruktor-Parameter in `NewEmailGRPCServer` UND die Konstruktion in
  `cmd/email/main.go` (inkl. des jetzt ungenutzten `emailcontact`-Imports dort) — kein Aufrufer
  blieb uebrig, Liegenlassen waere totes Konstrukt gewesen.
  Test (neu, `internal/server/email_grpc_export_test.go`, DB-backed, spiegelt
  `email_grpc_import_test.go` aus Iteration 29): `TestEmailExportContactsCSV_DB` und
  `TestEmailExportContactsVCard_DB` seeden je einen Kontakt ueber den bereits bewiesenen
  Import-Pfad, exportieren ihn ueber den Email-RPC und pruefen E-Mail UND Firmenname im
  Export-Output (nicht nur Erfolg ohne Panik). `TestEmailExportContactsCSV_MissingTenant` beweist
  401 statt Panik bei fehlendem Tenant-Context.
  FUND waehrend des Testens (kein Produktionsbug, reiner Test-Leak): der beim Import via
  Find-or-Create angelegte Firma-Datensatz ("Acme GmbH") wurde in der ersten Testfassung nie
  aufgeraeumt und blockierte das User-Cleanup per FK (`companies_created_by_fkey`) —
  `seedEmailExportContact` liest jetzt `company_id` mit aus und registriert dessen Cleanup VOR dem
  Cleanup der Kontakt-Zeile im Testkoerper, damit die LIFO-Reihenfolge (Kontakt -> Firma -> User ->
  Pool) FK-sauber ist. Vor dem Fix lief der Test trotzdem gruen (der Fehler landet nur als geloggte
  Cleanup-Warnung, nicht als Testfehler) — beim naechsten Mal auf genau solche stillen
  Cleanup-Zeilen im Testoutput achten, nicht nur auf PASS/FAIL.
  gate: build ok (`go build -p 2 ./...`) | vet ok (`go vet -p 2 ./...`) | lint ok (golangci-lint auf
  `internal/server`, `cmd/email`, 0 issues) | test ok — `go test -count=1 -v ./internal/server/...`
  mit `DATABASE_URL` auf `kmuhub_app`: 216 PASS, 0 SKIP, 0 FAIL (per `grep -c` gegengeprueft). Keine
  Migration, keine neue Route, kein neuer Guard — `openapi.yaml`-Diff nicht Pflicht, keiner
  vorgenommen.
- offen:
  - Alle unveraendert offenen Punkte aus Iteration 17-22/24-29 (siehe deren Aufzaehlung oben)
    bleiben unveraendert offen, hier nicht angefasst. Damit ist die Email-Import/Export-Nil-Provider-
    Fehlerklasse (beide Haelften) vollstaendig geschlossen.

## Iteration 31 — fix-berichte-kpi-module-scope — done — 2026-08-03
- commit: f3c65ad5
- verify vorgaenger: sauber. `4a02c5e7` (fix-email-export-nil-provider-panic, Iteration 30) gegen
  die Fehlerklassen geprueft: kein Proto-Diff, keine neue Route, kein neuer Guard, kein Stub.
  Klasse 1 (gRPC-Bypass) N/A — `EmailGRPCServer` IST die gRPC-Implementierung. Klasse 5
  (Tenant-Luecke) sauber: beide Export-RPCs holen `middleware.GetTenantID(ctx)` und bauen den
  `TenantScopedAdapter` daraus, 401 statt Panik bei fehlendem Context. Zusaetzlich gegengeprueft,
  was der Journal-Eintrag nicht erwaehnte: `NewExportService(provider, nil)` uebergibt einen
  nil-Logger — `NewExportService` faengt das ab (`export_service.go:25-27`, faellt auf
  `slog.Default()` zurueck), also keine zweite Nil-Panik an derselben Stelle. Der entfernte
  Singleton hatte keinen weiteren Aufrufer (grep). Nichts zu beanstanden.
- gebaut: `fix-berichte-kpi-module-scope` gezogen (naechste `status: todo`-Unit mit `deps: []`).
  Befund bestaetigt: `HandleGetDashboardKPIs` reichte `?modules=` ungeprueft an den RPC durch,
  einziger Guard war `berichte:reports:read` ("darf das Berichte-Modul oeffnen"). Die Kacheln
  tragen aber Umsatz, Pipeline-Volumen und Bestandszahlen fremder Module — ein `member` ohne
  `finance:module:view` las ueber `?modules=finanzen` den Monatsumsatz.
  Fix im Gateway (`route_berichte.go`): `kpiModuleVisibility` mappt Modul-ID -> Level-1-Capability
  und ist die serverseitige Kopie von `REPORT_MODULE_KEY` in
  `desktop/.../report-module-visibility.ts`. Wichtig und bewusst NICHT abgeleitet: die Berichte-
  Modul-ID heisst `finanzen`, der RBAC-Key `finance:module:view` — jede String-Ableitung waere beim
  naechsten Drift ein stilles Loch. `cross` bleibt ohne Modul-Capability (aggregierte Zahlen ohne
  Modulzuordnung), exakt wie `cross: null` im Frontend; unbekannte IDs fallen fail-closed raus.
  `visibleKPIModules(r, requested)` schneidet die Anfrage gegen `middleware.GetUserPermissions`.
  Leere Anfrage heisst jetzt "alle Module, die dieser Nutzer sehen darf" und wird **explizit**
  expandiert (sortiert — Map-Iteration ist zufaellig, das RPC-Argument darf es nicht sein).
  Kernfalle dieser Unit: der Executor liest eine leere Modulliste weiterhin als "alle Module"
  (`executor.go:337-340`). Ein Schnitt, der auf die leere Liste faellt, waere deshalb keine
  Verschaerfung, sondern eine Oeffnung. Deshalb antwortet der Handler bei leerem Ergebnis selbst mit
  `DashboardKPIsResponse{GeneratedAt: ...}` (200, keine Kacheln) und schickt nichts an den Service.
  Der Early-Return sitzt **vor** `getClient()`: eine Autorisierungsentscheidung darf nicht davon
  abhaengen, ob berichte gerade erreichbar ist.
  Der Route-Guard `berichte:reports:read` wurde nicht angefasst (nicht getauscht, kein
  `RequirePermissionAny` noetig) — der Schnitt sitzt hinter dem Guard, also kann kein Nutzer mit
  gueltigem Alt-Token ausgesperrt werden; er sieht schlimmstenfalls Kacheln weniger.
  `openapi.yaml`: keine neue Route, aber die Parameter-Beschreibung sagte "empty means all modules"
  und war damit falsch — auf die neue Semantik korrigiert.
  Test (neu, `internal/gateway/route_berichte_kpi_scope_test.go`, 7 Faelle, kein DB/gRPC noetig):
  Kern ist `TestHandleGetDashboardKPIs_MemberAskingForFinanzenGetsNothing` (member-Permissions aus
  Migration 000256, ohne `finance:module:view`, `?modules=finanzen` -> 200 ohne `kpis`) plus der
  Kontrast `..._GrantedModuleReachesTheService` (derselbe Request mit `finance:module:view` -> 503
  aus der leeren Registry, d.h. der Call ging wirklich raus). Ohne dieses Paar waere "200 leer"
  nicht von "Handler antwortet immer leer" zu unterscheiden. Dazu die Filterfaelle: Drop, Reihenfolge
  bleibt Request-Reihenfolge, Leer-Expansion ohne `finanzen`, unbekanntes Modul fail-closed,
  `cross` ohne Capability.
  FUND beim Testen: der bestehende `TestHandleGetDashboardKPIs_ServiceUnavailable` blieb nur
  deshalb gruen, weil `cross` immer sichtbar ist — ohne den Eintrag haette ein Request ohne
  Permissions eine leere Expansion und damit 200 statt 503 ergeben. Wer `cross` spaeter
  capability-pflichtig macht, muss diesen Test mitziehen.
  gate: build ok (`go build -p 2 ./...`) | vet ok (`./internal/gateway/...`) | lint ok
  (golangci-lint auf `./internal/gateway/...`, 0 issues) | test ok — `go test -count=1 -v
  ./internal/gateway/` mit `DATABASE_URL` auf `kmuhub_app`: 618 PASS, 0 FAIL, 0 SKIP (inkl.
  `TestOpenAPIRouteDrift`); `go test -count=1 ./internal/berichte/...` alle 6 Pakete ok.
  Keine Migration, keine neue Route, kein neuer Guard, kein neuer Permission-Key.
- offen:
  - Der Executor-Default "leere Modulliste = alle Module" (`internal/berichte/executor/executor.go`)
    steht unveraendert. Heute unschaedlich, weil das Gateway der einzige HTTP-Eingang ist und nie
    mehr eine leere Liste schickt (verifiziert per grep: genau ein Aufrufer). Wird der RPC je von
    einem zweiten Aufrufer genutzt, muss die Semantik dort mit umgestellt werden.
  - `kpiModuleVisibility` deckt die sechs `validModules` des Backends ab; das Frontend-Mapping kennt
    zusaetzlich work/kommunikation/hr/zeiterfassung/vertraege/einkauf/fuhrpark/rapporte. Solange
    `sanitizeModules` diese ohnehin verwirft, ist das deckungsgleich — wer `validModules` erweitert,
    muss `kpiModuleVisibility` im selben Zug erweitern, sonst faellt das neue Modul still weg.
  - Alle unveraendert offenen Punkte aus Iteration 17-22/24-30 bleiben offen, hier nicht angefasst.

## Iteration 32 — g-berichte-document-upload — done — 2026-08-03
- commit: dcf98a3e
- verify vorgaenger: sauber. `f3c65ad5` (fix-berichte-kpi-module-scope, Iteration 31) gegen die
  Fehlerklassen geprueft: Klasse 1 (gRPC-Bypass) N/A — Handler ruft weiterhin `client.GetDashboardKPIs`
  ueber den gRPC-Client, kein direkter Service-Zugriff. Klasse 2/3 N/A (kein Stub, kein .proto-Diff).
  Klasse 4 N/A — keine neuen `RequirePermission`-Guards, die verwendeten Capability-Keys
  (`finance:module:view` etc.) sind bereits in Migration 000256 geseedet (gegengeprueft per grep).
  Klasse 5 N/A (keine neue Tabelle). Klasse 6 sauber — Response bleibt dasselbe Proto, nur die
  Modulliste wird server-seitig geschnitten. Klasse 7 sauber — kein neuer Pfad, nur eine
  Parameter-Beschreibung in openapi.yaml korrigiert (keine neue Route noetig). Klasse 8 N/A — der
  Route-Guard `berichte:reports:read` wurde nicht angefasst. Nichts zu beanstanden.
- gebaut: `g-berichte-document-upload` gezogen (einzige `status: todo`-Unit mit `deps: []`).
  Befund bestaetigt: `desktop/.../berichte-client.ts:286` sendet `POST
  /api/v1/documents/files/upload` (multipart: file, folder_id, tag_id), die Route existierte nicht —
  "Bericht als PDF in Dokumente ablegen" war tot.
  Neue RPC `UploadFile` in `document.proto` (Request: folder_id, filename, mime_type, file_size,
  content bytes, owner_id, tag_id; Response: file) + Regen im selben Commit
  (`protoc --go_out --go-grpc_out proto/document/v1/document.proto`, da `make` auf dieser Maschine
  fehlt — Kommando aus dem Makefile-Target `proto-document` 1:1 uebernommen).
  Server-Implementierung (`document_grpc.go`) loest `folder_id` **vor** jedem Schreiben ueber
  `folderService.GetByID(ctx, tenantID, folderID)` auf — das ist der tenant-scope Check, der laut
  Unit-Notes Pflicht war ("ungeprueft waere das ein Weg, in fremde Ordner zu schreiben"); ein Ordner
  ausserhalb des eigenen Tenants liefert `folder.ErrFolderNotFound` -> 404, bevor `fileService.Upload`
  ueberhaupt aufgerufen wird. Die Space-Daten des aufgeloesten Ordners (`SpaceType`/`SpaceID`) fuettern
  den Storage-Key, wie es das bereits vorhandene (bisher komplett aufruferlose!) `file.Service.Upload`
  ueberall sonst im Modul erwartet — `Upload` selbst existierte fertig getestet, hatte aber schlicht
  keine RPC, die sie erreicht; diese Unit ist damit primaer Verdrahtung, keine neue Business-Logik.
  FUND + Root-Cause-Fix ueber die Unit hinaus: `tag_id` ist in `document_tags` genauso eine
  FK-Referenz wie `folder_id`, aber die FK-Pruefung laeuft ueber einen Trigger mit Owner-Rechten und
  umgeht damit RLS komplett — `TagFile` haette einen `tag_id` aus einem FREMDEN Tenant klaglos
  akzeptiert (die einzige Sicherung war RLS's WITH CHECK auf die `tenant_id`-Spalte der neuen
  `document_file_tags`-Zeile selbst, nicht auf den referenzierten Tag). Statt das nur in der neuen
  RPC ad-hoc abzufangen, wurde `tag.PostgresRepository.TagFile` (`internal/document/tag/
  postgres_repository.go`) um einen expliziten `SELECT EXISTS(...tenant_id=$2)`-Check vor dem INSERT
  erweitert (`ErrTagNotFound` bei Miss) — das schliesst dieselbe Luecke auch fuer die bestehende
  `HandleTagFile`-Route (`/api/v1/documents/tags/file`) mit, nicht nur fuer den neuen Pfad. Neuer
  Test `TestTagFile_RejectsTagFromAnotherTenant` (`tenant_write_test.go`) belegt das: eigener
  Tenant-Context, eigene Datei, aber `tag_id` eines fremden Tenants -> `ErrTagNotFound`, 0 Zeilen in
  `document_file_tags`. Der bestehende Test `TestDocumentTagWrites_LandInCallerTenant` blieb gruen
  (die dort gepruefte RLS-Session-Fehlpassung schlaegt weiterhin fehl, nur jetzt eine Zeile frueher
  ueber den neuen Ownership-Check statt ueber den rohen RLS-Fehler).
  `tag_id` ist im neuen Endpunkt bewusst optional (leerer String = kein Tag) und nicht hart
  Pflichtfeld: das FE (`saveReportToDocuments`) sendet aktuell den mock-typischen String-Literal
  `'t-bericht'` statt einer echten UUID (siehe `mocks/handlers/documents.ts:40`, wo Demo-Tags
  String-IDs statt UUIDs tragen) — `uuid.Parse` auf diesem Wert liefert einen 400
  `invalid tag id`, bis entweder das FE eine echte Tag-UUID schickt oder pro Tenant ein "Bericht"-Tag
  provisioniert wird. Das ist eine offene FE/Produkt-Frage, keine Backend-Luecke, siehe unten.
  Gateway-Handler (`route_document.go`, `HandleUploadFile`) folgt exakt dem Multipart-Muster aus
  `route_biz_gobd_archive.go` (55 MiB Formular-Cap = 50 MiB Datei + 5 MiB Felder,
  `grpc.MaxCallSendMsgSize(60<<20)` fuer den RPC-Call). Content-Type-Allowlist
  (`allowedDocumentUploadMimeTypes`) und Groessenlimit (`maxDocumentUploadBytes = 50<<20`, deckungsgleich
  mit dem bestehenden Presign-Limit in `presign.go`) sitzen bewusst im Gateway-Handler, nicht im
  generischen `file.Service.Upload` (der bleibt unveraendert wiederverwendbar fuer kuenftige Aufrufer
  mit anderen Anforderungen) — Allowlist ist dieselbe Dokumente/Bilder/Office/Archiv-Menge wie
  `internal/chat/file`s `allowedMimeTypes`, MINUS `image/svg+xml` (dieselbe XSS-Begruendung wie
  `brandingAllowedContentTypes` in `presign.go`: Dokumente werden potenziell inline ueber WOPI
  geoeffnet, ohne Sanitizer davor).
  Route registriert unter der bestehenden `docUpload`-Guard-Variable (`documents:file:upload` /
  `documents:write`, additiv, kein neuer Permission-Key noetig, kein Seed noetig).
  openapi.yaml: neuer Pfad `/api/v1/documents/files/upload` (multipart/form-data, 201/400/401/403/
  404/413/415) nach dem Vorbild von `/finance/gobd-archive`.
  gate: build ok (`go build -p 2` auf document/gateway/server/cmd/document/cmd/gateway) | vet ok |
  lint ok (golangci-lint auf `internal/document/...`, `internal/gateway/...`, `internal/server/...`,
  0 issues) | migration: keine (kein Schema-Change) | test ok — `go test -count=1 -v
  ./internal/document/... ./internal/server/...` mit `DATABASE_URL` auf `kmuhub_app`: 368 PASS, 0
  SKIP, 0 FAIL; `go test -count=1 ./internal/gateway/`: 618 PASS, 0 FAIL, 0 SKIP, inkl.
  `TestOpenAPIRouteDrift` (791 registrierte gegen 793 dokumentierte `/api/v1/*`-Pfade, gruen).
- offen:
  - `tag_id` aus dem FE ist aktuell der literale Mock-String `'t-bericht'`, keine echte UUID —
    der neue Endpunkt liefert dafuer 400 `invalid tag id`, das Speichern des Files selbst (ohne Tag)
    waere aber unbenommen, wenn das FE `tag_id` wegliesse. Zwei Wege fuer eine spaetere Unit: (a) FE
    schickt keine `tag_id` mehr, bis es echte Tag-UUIDs kennt, oder (b) ein "Bericht"-Tag wird pro
    Tenant provisioniert (im `ProvisionTenant`-Hook, `auth/provisioning.go:76`, derselbe Ort wie bei
    `g-rls-presence-and-dashboard-defaults`) und das FE erhaelt dessen echte UUID. Reine
    FE/Produkt-Entscheidung, keine Backend-Blockade.
  - `RegisterUploadedFile` (der bestehende Presign-Register-Pfad, `document_grpc.go:301`) hat
    denselben ungeprueften `folder_id`-Fund wie die neue `UploadFile`-Route hatte, VOR dem Fix: es
    validiert nicht, dass `folder_id` dem aufrufenden Tenant gehoert, bevor es die Datei-Metadaten
    darunter registriert. Bewusst NICHT in diesem Commit mitgezogen (andere Route, ausserhalb des
    Unit-Scopes, kein Test dafuer vorbereitet) — fuer eine kuenftige Fix-Unit vormerken, Vorlage ist
    der `folderService.GetByID`-Check aus dieser Iteration.
  - DB-Gate lief lokal vollstaendig (Postgres in Docker erreichbar), kein Nachlauf noetig.

## Iteration 33 — g-email-messages-bulk — done — 2026-08-03
- commit: 06f1447c
- verify vorgaenger: sauber. `dcf98a3e` (g-berichte-document-upload, Iteration 32) gegen die acht
  Fehlerklassen geprueft: Klasse 1 N/A — `HandleUploadFile` ruft `client.UploadFile` ueber den
  gRPC-Client, kein direkter Service-Zugriff. Klasse 2 N/A — `UploadFile` ist eine echte
  Implementierung (loest `folder_id` ueber `folderService.GetByID` auf, schreibt echte Bytes via
  `fileService.Upload`), kein Stub. Klasse 3 sauber — `.proto` und `.pb.go`/`.grpc.pb.go` im selben
  Commit (959 Zeilen Regen). Klasse 4 N/A — Route haengt am bestehenden `docUpload`-Guard
  (`documents:file:upload`/`documents:write`), kein neuer Permission-Key, kein Seed noetig. Klasse 5
  N/A — keine neue Tabelle. Klasse 6 sauber — Response `{file: DocumentFile}`, deckungsgleich mit den
  Nachbarrouten. Klasse 7 sauber — `/api/v1/documents/files/upload` steht in openapi.yaml, alle
  Status-Codes (400/401/403/404/413/415) dokumentiert. Klasse 8 N/A — kein bestehender Guard ersetzt.
  Zusatzfund im selben Commit sauber behandelt: `TagFile` bekam einen expliziten Tenant-Ownership-Check
  fuer `tag_id` (schliesst eine RLS-Bypass-Luecke ueber einen Owner-Trigger, betrifft auch die
  bestehende `/tags/file`-Route), mit eigenem Regressionstest belegt. Nichts zu beanstanden.
- gebaut: `g-email-messages-bulk` gezogen (erste `status: todo`-Unit mit erfuellten `deps: []` in
  Datei-Reihenfolge unter Block G).
  Verifiziert: `desktop/.../email-client.ts:191` (`bulk(ids, action, target)`) und
  `useEmail.ts:224` (`useBulkMessageAction`) riefen `POST /api/v1/email/messages/bulk`, das im Gateway
  nicht existierte — die Mehrfachauswahl-Toolbar in `MailsPage.tsx:706-718` (read/star/archive/spam/
  delete-Buttons) lief ins Leere.
  Neue RPC `BulkMessageAction` in `email.proto` (Request: `ids[]`, `action`, `target`; Response:
  `affected`) + Regen im selben Commit (`protoc --go_out --go-grpc_out proto/email/v1/email.proto`,
  kein dediziertes `proto-email`-Makefile-Target existierte — Kommando 1:1 aus dem generischen
  `proto:`-Ziel uebernommen und dokumentiert, dass `email.proto` dort bislang fehlte).
  Aktionsmenge bewusst NICHT die drei in den Backlog-Notes vorgeschlagenen ("read, unread, star, move,
  delete") — die stimmten nicht mit der real verdrahteten UI ueberein (dort: read/star/archive/spam/
  delete, kein move/unread als Button). Stattdessen die volle `BulkAction`-Menge aus der stateful
  MSW-Referenz `mocks/data/email-store.ts:424` uebernommen: read/unread/star/unstar/archive/spam/move/
  delete (dieselbe SSOT-Konvention wie bei RBAC Welle 1b gegen `mocks/handlers/rbac.ts`). `label`
  bewusst ausgelassen (kein FE-Aufrufer, eigene Ownership-Semantik) — jeder unbekannte Wert inkl.
  `label` liefert 422 (`codes.OutOfRange`, dieselbe Konvention wie der RBAC-Capability-Key-Check).
  Kein neuer Business-Code fuer die einfachen Aktionen: `BulkAction` im `message.Service` bildet jede
  Aktion auf bestehende Methoden ab (`MarkRead`/`MarkUnread`/`MoveToFolder`/`Delete`). Neu: `SetStarred`
  (duenner `UpdateFlags`-Wrapper, noetig weil `ToggleStar` togglet statt setzt — bulk star/unstar
  braucht einen deterministischen Zielzustand unabhaengig vom Ausgangszustand jeder Nachricht) und
  `FolderRepository.GetByAccountAndType` (neu, fuer archive/spam-Ordnerauflösung).
  Tenant-Isolation wie von den Notes gefordert PRO ID: `GetByID(ctx,id,tenantID)` vor jeder Mutation,
  keine fremde oder nicht-existente ID zaehlt in `affected`, und keine einzelne fremde ID bricht den
  Rest des Batches ab (Teil-Erfolg). `email_messages`/`email_folders` haben ohnehin aktives RLS
  (Migration 000122/000124) — die per-ID-Pruefung ist zusaetzlich zur DB-Ebene, nicht ihr Ersatz, und
  macht `affected` exakt zaehlbar (die bestehenden Single-Op-Repos geben nur `error` zurueck, keine
  Rows-Affected).
  `archive`/`spam` loesen den Zielordner PRO NACHRICHT ueber deren eigenen `account_id` auf (eine
  Unified-Inbox-Auswahl kann mehrere Konten umfassen) und cachen das Ergebnis pro Konto innerhalb eines
  Aufrufs, um nicht pro Nachricht erneut zu queryen. Hat ein Konto keinen Archiv-/Spam-Ordner (nicht
  jedes gesyncte Konto hat einen — siehe `sync/worker.go:50-54`), wird NUR diese Nachricht
  uebersprungen, nicht der ganze Aufruf verworfen.
  FUND ueber die Unit hinaus: die Route ist mit `email:write` gegated (das Minimum, das jede Aktion
  braucht), aber `action=delete` haette damit `email:delete` umgangen — die Einzel-Route
  `DELETE /api/v1/email/messages/{id}` verlangt dieses Recht separat. `HandleBulkMessageAction` prueft
  bei `action=="delete"` zusaetzlich explizit `email:delete` aus `middleware.GetUserPermissions(ctx)`
  und liefert 403 ohne — sonst waere Bulk eine breitere Tuer gewesen als das Einzel-Loeschen.
  Bestandsbeobachtung, NICHT in dieser Iteration behoben (ausserhalb des Scopes): `delete` in bulk ruft
  die bestehende `Service.Delete` (hartes `DELETE FROM email_messages`), waehrend die MSW-Mock-Referenz
  ein Soft-Delete in den Papierkorb simuliert. Diese Abweichung existierte schon in der Einzel-DELETE-
  Route vor dieser Unit — bulk reproduziert nur dasselbe Verhalten, fuehrt keine neue Inkonsistenz ein.
  `ids`-Obergrenze 500 gateway-seitig (`validate:"required,min=1,max=500,dive,uuid"`), vor jedem
  gRPC-Call geprueft.
  gate: build ok (`go build -p 2` auf email/gateway/server/cmd/email/cmd/gateway) | vet ok | lint ok
  (golangci-lint, 0 issues) | migration: keine (kein Schema-Change) | test ok —
  `go test -count=1 -v ./internal/email/... ./internal/gateway/ ./internal/server/...` mit
  `DATABASE_URL` auf `kmuhub_app`: 1797 PASS, 0 SKIP, 0 FAIL, inkl. `TestOpenAPIRouteDrift` (792
  registrierte gegen 794 dokumentierte `/api/v1/*`-Pfade, gruen). 8 neue Tests in
  `internal/email/message/service_test.go` fuer `BulkAction`: unbekannte Aktion, move ohne target,
  read/star/delete, unread/unstar, Cross-Tenant (fremde/fehlende ID zaehlt nicht, Rest laeuft durch),
  move mit gueltigem/unbekanntem Ziel, archive-Aufloesung pro Konto inkl. Teil-Skip.
- offen:
  - `delete` in bulk ist ein hartes Delete, die MSW-Referenz simuliert Soft-Delete-in-Papierkorb — diese
    Diskrepanz besteht schon in der Einzel-DELETE-Route und ist keine neue Luecke dieser Unit. Falls das
    Produktverhalten je auf Soft-Delete umgestellt wird, gehoert das in eine eigene Unit, die BEIDE
    Routen gleichzeitig aendert (sonst laufen Einzel- und Bulk-Loeschen wieder auseinander).
  - `label` als Bulk-Aktion ist NICHT gebaut (422). Falls das FE das je an einen Button haengt, braucht
    es eine eigene Entscheidung: taggen (add) oder ersetzen (replace), plus Tenant-Ownership-Check auf
    das Label wie bei `AssignMessageLabels`.
  - DB-Gate lief lokal vollstaendig (Postgres in Docker erreichbar), kein Nachlauf noetig.

## Iteration 34 — g-admin-users-list — done — 2026-08-03
- commit: 55f439fe
- verify vorgaenger: sauber. `06f1447c` (g-email-messages-bulk, Iteration 33) stichprobenartig gegen
  die Fehlerklassen geprueft (Iteration 33 hatte sich selbst bereits ausfuehrlich gegen Iteration 32
  geprueft und dokumentiert — hier nur der unabhaengige Gegen-Check dieser Iteration): Klasse 1 sauber
  — `HandleBulkMessageAction` ruft `client.BulkMessageAction`, kein direkter Service-Zugriff. Klasse 2
  sauber — `Service.BulkAction` ist eine echte Implementierung (Per-ID-Tenant-Check via `GetByID`,
  echtes Mapping auf bestehende Methoden, Teil-Erfolg statt Abbruch), kein Stub. Klasse 3 sauber —
  `email.pb.go`/`email_grpc.pb.go` im selben Commit regeneriert. Klasse 4 N/A — kein neuer Permission-
  Key, nur eine zusaetzliche Pruefung auf bereits bestehendes `email:delete`. Klasse 5 N/A — keine neue
  Tabelle. Klasse 7 sauber — `/api/v1/email/messages/bulk` in openapi.yaml. Klasse 8 sauber — der
  zusaetzliche `email:delete`-Check verschaerft die bestehende Route, ersetzt sie nicht. Nichts zu
  beanstanden.
- gebaut: `g-admin-users-list` gezogen (naechste `status: todo`-Unit mit erfuellten `deps: []`).
  Verifiziert wie in den Notes beschrieben: `useAdminUsers.ts:21` ruft `GET /api/v1/admin/users`, im
  Gateway existierte dort nur `/{id}/permissions`. Die Substanz war vorhanden (`HandleListUsers` unter
  `/api/v1/users`, `user_roles`, `invitations`) — diese Unit ist eine reine Zusammenfuehrung, kein
  neuer Datenbestand, deshalb KEINE Migration in diesem Commit.
  Neue RPC `ListAdminUsers` in `auth.proto` + Regen im selben Commit. Komposition lebt in
  `PostgresRepository.ListAdminUsers` (zwei Queries, keine pro Nutzer): (1) `users` LEFT JOIN
  `user_roles` LEFT JOIN `user_sessions`, `GROUP BY u.id` — Postgres erlaubt dank funktionaler
  Abhaengigkeit vom Primary Key, `first_name`/`last_name`/`email`/`is_active` im SELECT zu fuehren, ohne
  sie in GROUP BY aufzunehmen; (2) offene Einladungen (`accepted_at IS NULL AND expires_at > NOW()`)
  gegen eine einmalige Preset-Name→ID-Lookup (`roles WHERE tenant_id IS NULL`) aufgeloest. `Service.
  ListAdminUsers` ist reiner Pass-Through wie `ListRoles` — die Komposition sitzt in der Repository-
  Query, nicht verstreut ueber Service/Handler. Tenant-Scoping laeuft fuer beide Quellen vollstaendig
  ueber RLS, kein expliziter `tenantID`-Parameter (wie bei `ListUsers`/`ListRoles`) — sicher, weil die
  Route immer unter authentifiziertem Kontext laeuft, nie unter `sysctx.With()`.
  DREI Entscheidungen bewusst getroffen und hier dokumentiert (die Notes hatten nur zwei vorausgesehen):
  (1) `lastLoginAt` = `MAX(user_sessions.created_at)` — `created_at` markiert den Moment der
  Session-Erzeugung (`Service.CreateSession` laeuft direkt nach erfolgreichem Login), nicht
  `last_active_at`, das mit jeder Aktivitaet weiterlaeuft und damit vom Wortsinn "letzter Login"
  wegdriften wuerde. Bekannte Grenze: eine per Logout/Terminate geloeschte Session zaehlt nicht mehr
  mit — dokumentiert in openapi.yaml, nicht versteckt.
  (2) `roles` sind Rollen-IDs direkt aus `user_roles.role_id` — kein Join auf `roles` noetig, weil
  `AdminUser.roles` laut `admin-types.ts` bereits IDs erwartet (Preset- oder Custom-Rollen-IDs), keine
  Namen.
  (3) NEU, in den Notes nicht erwaehnt: `jobTitle` hat KEINE Datenquelle im Backend (`users` hat keine
  Spalte dafuer, das Feld lebt nur in der HR-Employee-Tabelle eines anderen Service mit eigener ID) —
  liefert bewusst immer `""` statt eines erfundenen Werts, dokumentiert in openapi.yaml. `hasOverrides`
  bleibt `omitempty` (R-6 Permission-Overrides existieren serverseitig noch nicht, siehe
  `g-rbac-user-overrides-model`, weiterhin offen).
  Invited-Zeilen kommen NICHT aus `users` — ein Invite legt laut `Service.CreateInvitation` keine
  Nutzerzeile an, nur eine `invitations`-Zeile. Deshalb firstName/lastName dort bewusst leer statt aus
  der E-Mail-Adresse geraten (`nameFromEmail`-Stil waere Erfindung, kein Datum) — derselbe Massstab wie
  bei lastLoginAt. Abgelaufene Einladungen zaehlen bewusst NICHT als "invited" (`expires_at > NOW()` im
  Query), sonst haette der Builder eine tote Einladung als aktiven Vorgang angezeigt.
  gate: build ok (`go build -p 2 ./...`) | vet ok (`go vet -p 2 ./...`) | migration: keine (kein
  Schema-Change) | lint ok (golangci-lint, 0 Issues auf den geaenderten Packages) | test ok — `go test
  -count=1 ./internal/auth/... ./internal/gateway/... ./internal/server/...` mit `DATABASE_URL` auf
  `kmuhub_app`: alle gruen, `TestOpenAPIRouteDrift` 793 gegen 795 dokumentierte Pfade. Zwei neue
  DB-Tests in `admin_users_db_test.go` (Merge aktiv/deaktiviert/eingeladen inkl. ausgeschlossener
  abgelaufener Einladung; Tenant-Isolation fuer User UND Invitation, eigene Tenants nicht TenantA/B),
  ein neuer Gateway-Test `TestHandleListAdminUsers_ServiceUnavailable`.
- offen:
  - `jobTitle`/`hasOverrides` bleiben leer/omitted, bis es eine echte Backend-Quelle gibt (HR-Verknuepfung
    bzw. `g-rbac-user-overrides-model`). Kein Blocker fuer diese Unit, aber relevant fuer `g-admin-users-
    invite` (naechste Unit): falls die Schreibrouten je Vor-/Nachnamen bei Invite erfassen sollen, braucht
    `invitations` dafuer neue Spalten.
  - `lastLoginAt` unterscheidet nicht zwischen "nie eingeloggt" und "eingeloggt, aber jede Session seither
    geloescht" — beides liefert `null`. Nur relevant, falls das FE das je unterscheiden will.
  - DB-Gate lief lokal vollstaendig (Postgres in Docker erreichbar), kein Nachlauf noetig.

## Iteration 35 — g-admin-users-invite — done — 2026-08-03
- commit: 6c937c7a
- verify vorgaenger: sauber. `55f439fe` (g-admin-users-list, Iteration 34) gegen die Fehlerklassen
  geprueft. Klasse 1 sauber — `HandleListAdminUsers` holt sich `a.getAuthClient()` und ruft
  `client.ListAdminUsers`, keine direkt injizierte Service-Instanz. Klasse 2 sauber —
  `PostgresRepository.ListAdminUsers` ist eine echte Zwei-Query-Komposition, kein Stub; die drei
  Datenluecken (jobTitle/hasOverrides/lastLoginAt) sind bewusst leer statt erfunden und im Code sowie
  in openapi.yaml als solche dokumentiert. Klasse 3 sauber — `auth.pb.go`/`auth_grpc.pb.go` im selben
  Commit regeneriert. Klasse 4 N/A — kein neuer Permission-Key, `RequireRole("admin")` wie die
  Nachbarroute. Klasse 5 N/A — keine neue Tabelle. Klasse 7 sauber — Route in openapi.yaml,
  `TestOpenAPIRouteDrift` gruen. Klasse 8 sauber — die neue `Get("/")` steht neben der bestehenden
  `/{id}/permissions`, ersetzt nichts. Nichts zu beanstanden.
- gebaut: `g-admin-users-invite` (naechste `status: todo`-Unit, `deps: [g-admin-users-list]` erfuellt).
  Drei Routen: `POST /api/v1/admin/users/invite` (201), `PATCH /api/v1/admin/users/{id}`,
  `POST /api/v1/admin/users/{id}/resend-invite`, alle `RequireRole("admin")` wie die Nachbarroute —
  kein neuer Permission-Key, also keine Seed-Migration noetig. Antwortform `{user, inviteToken?}`.
  Migration **000280** (`invitations`: `role_ids UUID[]`, `first_name`, `last_name`; up/down/up gruen).
  Keine neue Tabelle — `invitations` hat tenant_id + RLS seit 000249, die Spalten erben das.
  **Die roles-Array-Frage (zentrale Entscheidung der Unit): sauber modelliert, nicht abgeschnitten.**
  `role_ids` wird autoritativ fuer das, was eine angenommene Einladung gewaehrt; die Legacy-Spalte
  `role` bleibt NOT NULL, ist aber ab jetzt reine Anzeige (GET /api/v1/invitations liefert sie weiter,
  ProvisionTenant schreibt sie weiter) und entscheidet nichts mehr. Grund gegen die von den notes
  ebenfalls erlaubte Variante "erste Rolle nehmen, Rest dokumentieren": ein NAME kann eine
  Custom-Rolle gar nicht identifizieren — `roles` ist per `(COALESCE(tenant_id,zero), name)` eindeutig,
  zwei Tenants duerfen dieselbe "Buchhaltung" besitzen. Und das FE schickt ohnehin Rollen-IDs, weil der
  Roster seit Iteration 34 IDs liefert. Ein Kompromiss haette also nicht weniger Arbeit bedeutet,
  sondern nur eine Luecke.
- **Zwei Bestandsdefekte auf demselben Pfad, beide in dieser Iteration geschlossen:**
  (1) SICHERHEITSRELEVANT: `PostgresRepository.AcceptInvitation` loeste die Rolle mit
  `INSERT ... SELECT ... WHERE r.name = $2` auf — und der ganze Accept-Pfad laeuft unter
  `sysctx.With()`, wo RLS aus ist. Sobald irgendein Tenant eine Custom-Rolle namens "admin" besitzt,
  weist dieses INSERT dem neuen Konto **jede** gleichnamige Zeile zu, fremde Tenants eingeschlossen.
  Jetzt Aufloesung per ID mit `AND (r.tenant_id IS NULL OR r.tenant_id = <invitation.tenant>)`.
  Regressionstest `TestAcceptInvitation_DB_ResolvesRolesByID` seedet genau diese Namenskollision ueber
  zwei Tenants und prueft, dass nur die eigene Rolle ankommt.
  (2) `ProvisionTenant` hat einen EIGENEN INSERT in `invitations` (nicht ueber `CreateInvitation`) —
  der haette `role_ids` auf dem Spalten-Default `'{}'` stehen lassen, und der erste Admin-Invite eines
  frisch provisionierten Tenants waere nicht mehr annehmbar gewesen (`role_not_found`) — der Tenant
  waere gestrandet. Loest den Preset jetzt inline auf. Der Name muss dabei zweimal als Parameter
  wandern ($4 Spaltenwert, $9 Vergleich), sonst weigert sich Postgres, den Typ zu deduzieren
  ("inconsistent types deduced for parameter", varchar(50) vs text). Test
  `TestProvisionTenant_DB_InvitationCarriesRoleIDs`.
  Beide fand das Gate, nicht die Planung — (2) haette ohne den vollen `./internal/auth/...`-Lauf
  niemand bemerkt. Drittes Detail derselben Klasse: ein nil-Go-Slice kommt als SQL NULL an, nicht als
  Spalten-Default, deshalb `COALESCE($7::uuid[],'{}')` in `CreateInvitation`.
- **Seat-Limit: existiert bereits, greift automatisch.** Die notes fragten, woher die Platzzahl kommt.
  Antwort: `tenants.seat_limit` + `Service.assertSeatAvailable` + `repo.CountSeatsInUse` seit
  Migration 000249, und die Zaehlung schliesst offene, unabgelaufene Einladungen bereits ein. Der neue
  Invite laeuft ueber dieselbe interne `createInvitation` und erbt die Pruefung; die REAKTIVIERUNG
  eines deaktivierten Kontos laeuft ebenfalls dagegen, weil ein reaktiviertes Konto einen Platz belegt.
  Kein erfundener Default noetig.
- **Guardrails wiederverwendet statt kopiert, mit einer bewussten Abweichung von den notes.** Der
  Rollen-Vollersatz im PATCH laeuft ueber `AssignUserRole`/`RevokeUserRole` (Revokes zuerst, sonst
  triggert der Zwischenzustand last_admin), also greifen last_admin, self_lockout und die
  Escalation-Kappung unveraendert. Die notes forderten zusaetzlich "niemand vergibt per Invite/PATCH
  eine Rolle, die er selbst nicht haelt" — das waere INKONSISTENT zu `AssignUserRole`, das die
  Delegation an andere absichtlich erlaubt (hr_admin darf Rollen besetzen, die reicher sind als seine
  eigene) und nur die Selbstvergabe kappt. Eine strengere Regel nur hier waere ueber
  `POST /users/{id}/roles` trivial umgehbar gewesen, haette also Sicherheit vorgetaeuscht statt sie zu
  schaffen. Neu dazugekommen sind drei Regeln, die es vorher nicht gab: `self_deactivation` (409, ein
  Admin, der sich selbst abschaltet, merkt es erst beim naechsten Login — die laufende Session bleibt
  gueltig), die Deaktivierungs-Variante von `last_admin`, und `status_not_assignable` (400) fuer
  "invited". Fuer die Deaktivierungs-Variante braucht es einen neuen Repo-Zaehler
  `CountActiveRoleAdminsExcludingUser`: der bestehende `CountRoleAdminsExcluding` ignoriert ein
  user/role-PAAR und beantwortet damit "darf diese Rolle weg", nicht "darf dieses Konto ganz dunkel
  werden" — ein Admin mit der Faehigkeit ueber zwei Rollen waere von der Paar-Variante mitgezaehlt
  worden.
- **`inviteToken` in der Antwort.** Die beiden Invite-Routen liefern den Einmal-Token mit
  (`{user, inviteToken}`). Grund: **nichts im Backend versendet Einladungsmails** — `auth.Mailer` hat
  genau eine Methode (`SendPasswordResetEmail`), und `POST /api/v1/invitations` gibt den Token seit
  jeher in der Antwort zurueck. Ohne dieses Feld waere die neue Route unbenutzbar gewesen (Einladung
  angelegt, niemand kann sie annehmen). Das FE liest `data.user` und ignoriert das Feld — kein
  Vertragsbruch, in openapi.yaml dokumentiert.
- **Nicht-atomar, bewusst:** aendert ein PATCH Rollen UND Status, laufen die Status-Guards vorab, aber
  ein Fehler beim Status-Write laesst eine bereits geschriebene Rollenaenderung stehen. Der ehrliche
  Preis dafuer, die geschuetzten Methoden wiederzuverwenden statt user_roles direkt zu schreiben; das
  FE schickt laut `useAdminUsers.ts` ohnehin eins von beidem. In openapi.yaml benannt, nicht versteckt.
- gate: build ok (`go build -p 2 ./...`) | vet ok | migration 000280 up/down/up gruen | lint ok
  (golangci-lint auf auth/gateway/server/models, 0 Issues) | test ok — `go test -count=1
  ./internal/auth/... ./internal/gateway/... ./internal/server/...` mit `DATABASE_URL` auf
  `kmuhub_app`: alle gruen, **177 PASS / 0 SKIP** in `internal/auth`, `TestOpenAPIRouteDrift` gruen.
  8 neue DB-Tests (`admin_users_write_db_test.go`) + 6 neue Gateway-Tests. Der last-admin-Test bekam
  einen EIGENEN Tenant: "letzter Administrator" ist eine Eigenschaft der Tenant-Population, ein
  Nachbartest im geteilten Tenant haette ueber sein Ergebnis entschieden (erster Versuch endete
  genau deshalb in einem Skip — der dann rausgeflogen ist, weil ein Skip nichts beweist).
  Zur Validator-Konvention: `decodeAndValidate` antwortet mit **400** `validation_failed`, nicht 422 —
  die openapi-Eintraege dieser Unit sind entsprechend korrigiert worden (erste Fassung dokumentierte
  faelschlich 422; die Tests haben es aufgedeckt).
- offen:
  - **MSW-Mock passt nicht mehr zum echten Backend:** die Invite-Mocks im Desktop arbeiten mit
    Rollen-NAMEN (`'admin'`), das echte Backend verlangt UUIDs (`dive,uuid` -> 400). Kein
    Backend-Blocker (der Roster liefert seit Iteration 34 UUIDs, das FE reicht im Echtbetrieb also
    UUIDs zurueck), aber der Mock maskiert den Unterschied — genau das Muster aus
    `feedback_nested_proto_flat_type.md`. Gehoert in eine FE-Unit.
  - **Einladungsmail wird weiterhin nirgends versendet.** `auth.Mailer` kennt nur Passwort-Resets; der
    Token geht an den Aufrufer. Das ist Bestand, kein Regress, aber A-1 ist erst wirklich fertig, wenn
    jemand die Zustellung baut (eigene Unit, braucht `SYSTEM_SMTP_*`-Anbindung wie der berichte-Service).
  - `invitations.role` ist jetzt tote Anzeige-Denormalisierung. Wenn `GET /api/v1/invitations` irgendwann
    auf `role_ids` umgestellt wird, kann die Spalte weg — nicht in dieser Unit, weil das den
    Antwortvertrag der Legacy-Route bricht.
  - DB-Gate lief lokal vollstaendig (Postgres in Docker erreichbar), kein Nachlauf noetig.

## Iteration 36 — g-hr-change-requests — done — 2026-08-03
- commit: dbcf2493
- gebaut: Migration 000281 (`hr_profile_change_requests`, tenant_id + RLS + partieller
  Unique-Index auf offene Antraege), Package `internal/biz/hr/changerequest` (Service, Repository,
  Postgres-Impl, Fehler), 5 RPCs auf `HRService` inkl. Regen, 5 Routen unter
  `/api/v1/hr/change-requests` + openapi-Eintraege, Wiring in `cmd/biz/main.go`.
- verify vorgaenger (`6c937c7a`, admin-users-write): **sauber**. Handler gehen ueber
  `client.InviteAdminUser`/`UpdateAdminUser`/`ResendAdminUserInvite`, kein Direct-Svc. Kein Stub
  (das einzige `return nil, nil` in `admin_users.go:185` heisst "Status unveraendert, nichts zu
  tun"). `.proto` + beide `.pb.go` im selben Commit. Kein neuer `RequirePermission` — die drei
  Routen haengen am bestehenden `RequireRole("admin")`, also auch kein Guard-Ersatz (Klasse 8).
  Keine neue Tabelle; 000280 ergaenzt Spalten auf der bereits RLS-geschuetzten `invitations`,
  INSERT/UPDATE tragen den Tenant. Die zwei ungescopten Reads (`GetAdminUser`,
  `GetInvitationAsAdminUser`) laufen NICHT unter sysctx, also filtert RLS — dasselbe Muster, das
  `ListRoles` dokumentiert. `CountActiveRoleAdminsExcludingUser` holt die Tenant-Grenze ueber den
  `JOIN users`, wie der bestehende Paar-Zaehler. openapi +236 Zeilen.

- **Der HR-Service liegt nicht, wo der Backlog ihn vermutet.** `sources` nennt `backend/internal/hr/`
  — das Verzeichnis existiert nicht. HR wohnt in `internal/biz/hr/{leave,employee,timetracking,
  absence}` und wird auf dem **biz**-gRPC-Server registriert (`HRRoutes.ServiceName()` gibt "biz"
  zurueck, um die vorhandene Verbindung wiederzuverwenden). Wiring in `cmd/biz/main.go`, nicht in
  einem eigenen `cmd/hr`.
- **Feld-Allowlist statt freiem Feldnamen — der Kern der Unit.** Der MSW-Handler schreibt
  `emp[req.field] = req.newValue`, also einen beliebigen Property-Namen aus dem Request. Serverseitig
  waere das eine Spalte, und `field` kommt vom Client: das ist eine Injection und ausserdem ein Weg,
  `salary` oder `manager_user_id` per "Antrag" zu setzen. Deshalb `proposableFields` mit sechs
  Eintraegen (address_street/city/postal_code/country, emergency_contact_name/phone) — der
  Spaltenname kommt aus dieser Konstante, nie aus dem Payload.
  Folge davon, und der einzige echte Vertragsbruch gegen die UI: `SelfServiceView.tsx` bietet
  **`phone` und `mobile`** an. Beide existieren weder auf `hr_employee_profiles` noch auf `users`
  (das FE liest sie aus `raw.phone`/`raw.mobile`, die das echte Backend nie fuellt). Ein Antrag
  darauf ist jetzt 400 `field cannot be proposed`, statt eine Zeile zu speichern, die approve
  niemals ausfuehren koennte — genau die Fake-Erfolg-Klasse. Wer die Felder will, braucht zuerst die
  Spalten; das ist eine eigene Unit, keine Zeile hier.
- **Erster echter Team-Scope-Resolver im Repo.** `middleware.PermissionScope` dokumentiert selbst,
  dass `ScopeTeam` mangels Resolver wie `ScopeAll` wirkt. Die Unit verlangt aber "ein Manager ohne
  Scope auf diese Person darf nicht genehmigen". `approveScopeAllows` loest den Scope deshalb gegen
  `hr_employee_profiles.manager_user_id` auf: `all` = jeder, `team` = eigene Reports (plus man
  selbst), `own` = nur der eigene Antrag. Fehlt dem Antragsteller das Profil, greift `team` nicht —
  eine Berichtslinie, die man nicht belegen kann, gibt es nicht.
  Dieselbe Regel steckt in der LISTE (`Filter.ManagerID`), nicht als Post-Filter, sondern als
  Praedikat. Sonst zeigt die Inbox Zeilen, die beim Klick 403 antworten.
- **Sichtbarkeit der Liste ist Handler-Sache, nicht Guard-Sache.** Der Guard laesst Proposer UND
  Entscheider auf `GET /change-requests` (ein Mitarbeiter muss seine eigenen Antraege sehen). Wer
  `team:data_personal:edit` NICHT haelt, bekommt `own_only` erzwungen — unabhaengig vom
  `?scope=`-Parameter. `PermissionScope` kann das nicht beantworten: ein fehlender Key liefert dort
  absichtlich `all`, ein reiner Proposer haette also volle Reichweite gemeldet. Dafuer gibt es
  `callerHasPermission` (exakter Key-Treffer). Zwei Gateway-Tests sichern beide Richtungen ab.
- **422 laeuft ueber `codes.OutOfRange`, nicht ueber den Validator.** Die Unit fordert 422 fuer
  "reject ohne Grund", die Repo-Konvention `decodeAndValidate` antwortet aber mit **400**
  (`validation_failed`, siehe Iteration 35). Deshalb ist `reason` im HTTP-Struct `omitempty` und die
  Pflicht liegt im Service (`ErrReasonRequired`); `grpcStatusToHTTP` mappt `OutOfRange` -> 422.
- **Approve ist eine Transaktion, kein Zweischritt.** Erst der bedingte UPDATE auf den Antrag
  (`WHERE status='pending'` ist zugleich der Concurrency-Guard: der zweite Approver trifft null
  Zeilen), dann der Profil-UPDATE, beides in einer Tx. Kein Profil vorhanden -> Rollback und 404,
  statt einer Genehmigung, die nichts geaendert hat. Die Alternative (ueber `employee.Service`) waere
  wiederverwendet gewesen, haette aber genau den halben Zustand erlaubt, den die Offboard-Unit als
  schlimmsten Ausgang beschreibt.
- **Der 409 sitzt im partiellen Unique-Index**, nicht nur im Service: `UNIQUE (tenant_id, user_id,
  field) WHERE status='pending'`. Zwei gleichzeitige POSTs bestehen beide einen vorgeschalteten
  SELECT. Partiell, damit ein zurueckgezogener oder abgelehnter Antrag denselben Wunsch nicht fuer
  immer sperrt — dafuer gibt es einen Test.
- **Kein Permission-Seed noetig, und das ist geprueft, nicht vermutet:** `team:self:propose` und
  `team:data_personal:edit` stehen seit Migration 000256 im Katalog; `self:propose` haben alle vier
  Presets, `data_personal:edit` admin + hr_admin (`manager` hat nur `view` — ein Manager entscheidet
  also erst, wenn ein Tenant ihm per Custom-Rolle `edit` gibt, und dann greift der Team-Scope).
- **`cancel` schreibt keinen Entscheider.** Ein Rueckzug ist nicht die Entscheidung eines anderen;
  der MSW-Handler setzt dort ebenfalls weder `decidedAt` noch `decidedByName`. Test sichert es.
- Wire-Shape gegen `api/hr-change-requests.ts` geprueft, nicht geraten: `{requests,total}` bzw.
  `{request}`, leere Liste `[]`, und **camelCase** — anders als der Rest von HR. Deshalb ist
  `changeRequestBody` handgeschrieben statt protojson-marshalled: `cannedResponseMarshaler` setzt
  `UseProtoNames: true` und haette `user_id` geliefert, wo das FE `userId` liest. Ein Test
  vergleicht das gerenderte JSON Feld fuer Feld gegen den FE-Typ.
- gate: build ok (`go build -p 2 ./...`) | vet ok | lint ok (golangci-lint auf biz/hr, gateway,
  server, models, cmd/biz — 0 Issues) | migration 000281 up/down/up gruen | rls-smoke ok (eigener
  Tenant 1, fremder 0, als `kmuhub_app`; Testzeile danach entfernt) | test ok — `go test -count=1
  ./internal/biz/hr/... ./internal/gateway/... ./internal/server/... ./internal/testutil/...` mit
  `DATABASE_URL` auf `kmuhub_app`: alles gruen. **7 neue DB-Tests, 0 SKIP** (mit `-v` geprueft), 5
  neue Gateway-Tests, `TestOpenAPIRouteDrift` gruen, der RLS-Regressionstest in `internal/testutil`
  sieht die neue Tabelle und ist zufrieden. Jeder DB-Test seedet einen EIGENEN Tenant — "wer meldet
  an wen" und "welche Antraege sind offen" sind Eigenschaften der Tenant-Population, ein geteilter
  Tenant haette unter `-parallel` ueber fremde Ergebnisse entschieden.
- offen:
  - **`phone`/`mobile` sind FE-seitig tot** (siehe oben). Bis die Spalten existieren, laufen zwei
    der sieben angebotenen Felder in der Self-Service-Ansicht in ein 400. Eigene Unit: entweder
    Spalten nachziehen (dann auch im Employee-Update lesen/schreiben) oder die beiden Eintraege aus
    `PROPOSABLE_FIELDS` im FE entfernen.
  - **MSW-Mock weicht in drei Punkten ab** und maskiert das echte Verhalten: er akzeptiert jeden
    Feldnamen, erlaubt approve/reject auf bereits entschiedenen Antraegen (Backend: 409) und prueft
    beim Genehmigen keinen Scope. Gehoert in eine FE-Unit — der Mock ist sonst das, woran die UI
    entwickelt wird.
  - Der `drawer` "job" hat noch kein beantragbares Feld. Der Wert bleibt im Schema und im Vertrag,
    weil das FE ihn kennt; sobald ein Job-Feld beantragbar wird, ist es ein Eintrag in
    `proposableFields`.
  - DB-Gate lief lokal vollstaendig (Docker-Postgres erreichbar), kein Nachlauf noetig.

## Iteration 37 — g-hr-offboard — done — 2026-08-03 04:40
- commit: 23ceede6
- gebaut: Migration 000282 (status/last_work_day/exit_date/exit_type/exit_reason auf
  `hr_employee_profiles`, drei CHECKs, Index auf (tenant_id,status)), RPC `OffboardEmployee` am
  hr-Proto inkl. Regen, `Service.OffboardEmployee` + `PostgresEmployeeRepo.Offboard` in
  `internal/biz/hr/employee`, Route `POST /api/v1/hr/employees/{id}/offboard` hinter
  `RequirePermission("team:employee","offboard")`, OpenAPI-Pfad + zwei Schemas.
- verify vorgaenger (dbcf2493): sauber. Gateway-Handler gehen alle ueber `h.getHRClient()`, kein
  Direct-Svc; `.proto` + beide `.pb.go` im selben Commit; jeder SELECT in
  `changerequest/postgres_repository.go` traegt `tenant_id`; kein neuer Guard ohne Katalog-Eintrag;
  openapi.yaml mit 185 Zeilen im selben Commit. Kein Fund, keine Fix-Unit.

**Die Architektur-Entscheidung, weil die notes hier eine falsche Praemisse hatten.** Die Unit
schreibt vor: "Liegt HR-Profil und Auth-Konto in verschiedenen Services, ist eine verteilte
Transaktion nicht baubar." Das stimmt fuer die Prozessgrenze, aber nicht fuer die Datengrenze:
`users`, `user_roles` und `hr_employee_profiles` liegen in derselben Postgres-Instanz, und der
biz-Pool sieht alle drei (er liest `users` heute schon in jedem Employee-SELECT). Die Kaskade
laeuft deshalb als EINE Transaktion, nicht als geordnete Schrittfolge mit Nachprotokollierung —
der "Konto kann noch einloggen, Akte sagt ausgeschieden"-Zustand ist damit unerreichbar statt nur
unwahrscheinlich. Kein Fallback-Pfad noetig, keine Kompensationslogik.
- **Warum HR und nicht auth**, obwohl `ErrSelfDeactivation`, `ErrLastAdmin` und `RevokeUserRole`
  dort schon stehen: der Vertrag ist HR-foermig (`POST /hr/employees/{id}/offboard` -> `{employee}`).
  Ein auth-RPC muesste `hr.v1.EmployeeProfile` importieren (Proto-Kopplung zweier Services) oder
  das Gateway zu einem Zweischritt zwingen. Umgekehrt kostet der gewaehlte Weg genau ein Duplikat:
  die COUNT-Query hinter dem Last-Admin-Guard, kommentiert mit Verweis auf
  `auth.PostgresRepository.CountActiveRoleAdminsExcludingUser` und `auth.roleAdminKeys` (dort
  unexportiert). Die Alternative haette die sicherheitskritische Logik dupliziert statt eines
  COUNTs. Es gibt in diesem Repo **keinen** Service-zu-Service-gRPC-Client (verifiziert:
  `grpc.NewClient` kommt ausschliesslich in `internal/gateway/` vor) — eine dritte Option war das
  also nicht.
- **"Platz freigeben" ist kein eigener Kaskadenschritt.** auth zaehlt Seats als
  `COUNT(users WHERE is_active)` (`CountSeatsInUse`, postgres_repository.go:1114) — `is_active =
  false` gibt den Platz von selbst zurueck. Und das Sperren ist vollstaendig: Login (service.go:197)
  **und** Refresh (service.go:248) pruefen beide `!user.IsActive`. Refresh-Tokens zu loeschen waere
  wirkungslose Zusatzarbeit gewesen. Was bleibt: ein bereits ausgestelltes Access-Token laeuft bis
  zum Ablauf weiter — dasselbe Verhalten wie bei `UpdateAdminUser`, kein Sonderfall dieser Route.
- **Der Zyklus-Schutz ist eine rekursive CTE, kein Skip des Nachfolgers.** Der naheliegende Fix
  ("Nachfolger von der Umhaenge-Menge ausnehmen") deckt nur den direkten Fall. Sitzt der Nachfolger
  zwei Ebenen tief — Teamleiter scheidet aus, jemand aus einem Unterteam uebernimmt —, dann wird
  dessen eigener Vorgesetzter auf ihn umgehaengt und die beiden managen sich gegenseitig. Die CTE
  laeuft vom Nachfolger nach oben (Tiefenlimit 64) und gibt allen Knoten auf dieser Kette den
  Vorgesetzten des Ausscheidenden — was eine Befoerderung fachlich ohnehin bedeutet. Beide Faelle
  haben einen eigenen DB-Test.
- **Fuenf Exit-Types, nicht die vier aus dem scope-Text.** `OffboardEmployeeDialog.tsx:46-50` bietet
  zusaetzlich `fixed_term_expired`. Ein Backend, das nur die vier genannten kennt, haette einen
  Wert, den die UI produziert, an der Grenze abgelehnt. Ein Gateway-Test geht alle fuenf durch.
- **Der Body ist camelCase**, anders als der Rest der HR-Routen: `hr-client.ts:943` postet die
  Formularwerte des Dialogs unveraendert. Ein Test schickt bewusst einen snake_case-Body und
  erwartet 400 — sonst wandert diese Abweichung unbemerkt.
- **Response gewrappt.** `response.Proto(w, …, resp.Employee)` liefert das Profil nackt; der Client
  liest `{employee}`. Der Handler schreibt deshalb die ganze Resp-Message. **Fund im Bestand:** die
  Nachbar-Handler `HandleCreateEmployee`/`HandleUpdateEmployee` liefern nackt, waehrend
  `hrEmployeeApi.create/update` `raw.employee` liest — dort duerfte `adaptEmployee` heute
  `undefined` bekommen. Nicht in dieser Unit gefixt (Bestandsvertrag, keine Testabdeckung); als
  Unit-Kandidat notiert.
- **Kein Permission-Seed** — geprueft, nicht vermutet: `team:employee:offboard` steht in
  `permissions` und ist `admin` + `hr_admin` zugewiesen.
- **422 fuer "Reports ohne Nachfolger"** laeuft ueber `codes.OutOfRange` (der Repo-Weg seit
  Iteration 36), nicht ueber den Validator — der antwortet mit 400.
- **Zweiter Fund im Bestand:** `scanEmployeeProfile` liest `emergency_contact_name` und
  Geschwister in plain `string`; eine Zeile mit NULL dort sprengt jeden Read dieses Profils. Die App
  schreibt immer `""`, handgeschriebene Zeilen nicht — der erste Testlauf ist genau darueber
  gefallen. Fixture angeglichen (mit Kommentar), Produktionscode bewusst nicht angefasst.
- gate: build ok (`go build -p 2` ueber biz/hr, gateway, server, models, cmd/biz, cmd/gateway) |
  vet ok | lint ok (golangci-lint, 0 issues) | migration 000282 up/down/up gruen | rls-smoke n.a.
  (keine neue Tabelle; `TestAllPublicTablesHaveRLSOrAreAllowlisted` gruen) | test ok — `go test
  -count=1 ./internal/biz/hr/... ./internal/gateway/ ./internal/server/... ./internal/testutil/...`
  mit `DATABASE_URL` auf `kmuhub_app`: alles gruen. **9 neue DB-Tests, 0 SKIP** (mit `-v` und
  `grep -c '^--- SKIP'` = 0 geprueft), 4 neue Gateway-Tests, `TestOpenAPIRouteDrift` gruen. Jeder
  DB-Test seedet einen EIGENEN Tenant — "wer meldet an wen" und "wer ist der letzte
  Rollen-Administrator" sind Eigenschaften der Tenant-Population.
- offen:
  - **`{employee}`-Wrapping bei create/update** (Fund oben). Eigene Unit: entweder die beiden
    Handler auf die Resp-Message umstellen oder den FE-Client auf die nackte Antwort. Ein
    Alleingang auf einer Seite bricht die andere.
  - **Der MSW-Mock kennt die Guards nicht** (`handlers/team.ts:366`): er offboardet ohne
    Nachfolger-Pflicht, ohne Last-Admin-Pruefung und ohne Selbst-Schutz, und er haengt Reports
    unbesehen auf den Nachfolger um — inklusive des Zyklus, den das Backend jetzt verhindert. Der
    Mock ist das, woran die UI entwickelt wird; gehoert in eine FE-Unit.
  - **`backfill` wird nur protokolliert**, nicht ausgewertet — es gibt im Backend kein
    Stellen-/Recruiting-Modell, an das es andocken koennte. Bewusst so, im Proto kommentiert.
  - DB-Gate lief lokal vollstaendig (Docker-Postgres erreichbar), kein Nachlauf noetig.

## Iteration 38 — g-hr-salary-statements — blocked — 2026-08-03 04:42
- commit: -
- gebaut: nichts — Unit auf `blocked` gesetzt, Begruendung im `blocked_reason:`-Feld der Unit.
- gate: n.a. (keine Code-Aenderung)
- verify vorgaenger (23ceede6, g-hr-offboard): sauber. `client.OffboardEmployee` laeuft ueber den
  gRPC-Client (kein Direct-Svc); `.proto` + `.pb.go`/`.grpc.pb.go` im selben Commit regeneriert;
  `team:employee:offboard` steht seit Migration 000256 im Katalog und ist admin+hr_admin
  zugewiesen (kein fehlender Seed); jede neue/geaenderte SELECT-Query traegt `tenant_id`
  (offboard-Transaktion scoped users/hr_employee_profiles konsistent, user_roles ueber die
  Subquery auf users); `OffboardEmployeeResp{employee}` matched `response.Proto(w, ..., resp)` —
  Wire-Shape `{employee}` stimmt mit dem FE-Client ueberein; openapi.yaml dokumentiert alle fuenf
  Statuscodes (400/401/403/404/409) im selben Commit; kein TODO/Unimplemented/Fake-Return im
  neuen Pfad (`UnimplementedHRServiceServer`-Treffer ist der erwartete Boilerplate-Embed). Kein
  Fund, keine Fix-Unit.
- Der Befund, der `g-hr-salary-statements` blockiert: die notes der Unit gingen von einer
  fehlenden Dokument-Kategorie aus (`hrcat-payroll` in `hr_document_categories`), aber der reale
  FE-Vertrag (`SelfServiceView.tsx:60-66`, `downloadStatement()` Zeile 316) verlangt gar keine
  Dokumente — `SalaryStatement` ist `{id, month, label, gross, net}` ohne `fileId`, der Download
  wird client-seitig aus den Zahlen als Text-Blob gebaut. Das Backend hat aber keine Quelle fuer
  echte Monats-Brutto-/Netto-Betraege: `EmployeeProfile` traegt nur ein optionales `HourlyRate`,
  kein Monats-/Jahresgehalt, keinen Payroll-Lauf, keine persistierten Abrechnungen. Der MSW-Mock
  erfindet `net` als `gross * 0.675` — eine geratene Steuer-/SV-Naeherung; dasselbe Muster hinter
  einer echten Route waere Fehlerklasse 2 (hartkodierte Beispieldaten). DATEV im Repo ist reiner
  Buchungsstapel-Export fuer Finance, keine Lohnquelle. `team:salary:view/edit` stehen im Katalog,
  sind aber an keiner Backend-Route verdrahtet — diese Route waere die erste. Drei ehrliche Wege
  (echtes Gehalts-/Netto-System, HR laedt echte Abrechnungs-PDFs hoch, oder Feature vorerst aus
  dem Scope) sind Produktentscheidungen fuer Luke, keine Loop-Wahl. Volle Herleitung im
  `blocked_reason:`-Feld der Unit in BACKLOG.yml.
- offen:
  - Luke: Entscheidung zwischen den drei Wegen in `blocked_reason:` treffen, dann Unit
    reaktivieren bzw. neu scopen.
  - Naechste Iteration zieht die naechste `todo`-Unit mit erfuellten deps
    (`g-rbac-user-overrides-model`, opus, deps `p1b-audit-events` erfuellt).

## Iteration 39 — g-rbac-user-overrides-model — done — 2026-08-03
- commit: de6eb85a
- gebaut: Persistenz + CRUD fuer Per-User-Rechte-Overrides (RBAC R-6). Migration 000283 legt
  `user_permission_overrides` an (`tenant_id NOT NULL` + `enable_tenant_rls`, unique ueber
  `(tenant_id, user_id, permission_key)`, `permission_key` als FK auf `permissions(name)` mit
  ON DELETE CASCADE — ein aus dem Katalog entfernter Key nimmt seine Overrides mit). Service
  `internal/auth/user_overrides.go`, Repo-Methoden, drei RPCs, drei Routen
  `GET/PUT/DELETE /api/v1/admin/users/{id}/overrides` hinter
  `RequirePermission("admin:user_override","manage")`.
- **Kein neuer Permission-Seed noetig, und das ist verifiziert, nicht angenommen:**
  `admin:user_override:manage` steht seit Migration 000256 im Katalog (Zeile 116) und in der
  admin-Grant-Liste (Zeile 402); DB-Query gegen die lokale Instanz bestaetigt: nur `admin` haelt
  ihn. `hr_admin` haelt ihn bewusst nicht (R6-Briefing §0.3). Ein zusaetzlicher Seed waere ein
  No-Op gewesen — im Migrationskopf als Kommentar begruendet.
- **Fund, der einen Test zuerst falsch gruen aussehen liess:** `hr_admin` haelt selbst
  `admin:role:assign` und ist damit nach `roleAdminKeys` ein Rollen-Administrator. Der erste
  Last-Admin-Test benutzte einen hr_admin als Caller und erreichte den Guard nie (Count blieb 1).
  Von den Presets halten **admin, it_admin und hr_admin** `admin:role:assign` — nur `manager` und
  darunter nicht. Wer den Last-Admin-Guard testen will, braucht einen Caller unterhalb davon; ein
  deny braucht keinen Reach, deshalb geht das ueberhaupt.
- Guardrails serverseitig (das FE erzwingt sie nur als UI):
  - **Selbst-Bearbeitung komplett gesperrt** (PUT und DELETE, `actorID == userID` →
    `self_lockout`/409). "Eigener Account nicht editierbar" aus dem Briefing woertlich genommen —
    ein allow auf sich selbst waere der direkte Weg von "darf feinjustieren" zu "hat alles".
  - **Eskalation** nur auf der allow-Haelfte, ueber das bestehende `assertWithinReach` (nicht
    kopiert). Ein `deny` auf einen Key, den der Caller selbst nicht haelt, ist erlaubt — es
    vergroessert niemanden, und ein Admin muss hinter einem Kollegen mit anderer Rolle aufraeumen
    koennen.
  - **Last-Admin** feuert nur, wenn die neue Map *beide* `roleAdminKeys` des Ziels denied. Die
    Zaehlung der Uebrigen ist **override-aware** (`CountEffectiveRoleAdminsExcluding`): sobald
    Overrides existieren, kann jemand allein per allow Administrator sein, und eine rein
    rollenbasierte Zaehlung wuerde ein voellig sicheres deny verweigern.
  - **422** fuer unbekannten Katalog-Key *und* fuer unspellbares mode/scope — dieselbe
    Entscheidung wie bei `SetRolePermissions` (ein eigenes Sentinel kaeme im FE als "Unbekannter
    Fehler" an, weil `rbac-format.ts` es nicht kennt). Ein `deny` ohne scope wird auf `all`
    normalisiert statt abgelehnt: der scope ist dort bedeutungslos.
- Audit: ein Event pro **tatsaechlich geaendertem** Key (`permission.override_set` /
  `permission.override_removed`) ueber `logPermissionEvent`, kein zweiter Pfad. Erneutes Speichern
  derselben Map schreibt nichts — sonst waere der Trail eine Historie von Save-Klicks statt von
  Entscheidungen. Der Diff kommt aus dem Service (`OverrideChange`), nicht aus einer zweiten
  Abfrage im gRPC-Server.
- Wire-Shape gegen `rbac-types.ts` geprueft: `{userId, overrides: {key: {mode, scope}}}`, leere
  Map als `{}` nicht `null` (zwei Marshal-Tests pinnen es). DELETE antwortet 204 wie der
  MSW-Handler.
- gate: build ok (`go build -p 2` ueber auth, server, gateway, cmd/auth, cmd/gateway) | vet ok |
  lint ok (golangci-lint, 0 issues) | migration 000283 up/down/up gruen (Kopf 282 → 283) |
  **rls-smoke bestanden: eigener Tenant 1, fremder Tenant 0** (Smoke-Zeile danach entfernt);
  `TestAllPublicTablesHaveRLSOrAreAllowlisted` gruen | test ok — `go test -count=1
  ./internal/auth/... ./internal/gateway/ ./internal/server/... ./internal/testutil/...` mit
  `DATABASE_URL` auf `kmuhub_app`: alles gruen. **12 neue DB-Tests + 4 Audit-Subtests, 0 SKIP**
  (mit `-v` und `grep -c '^--- SKIP'` = 0 geprueft), 5 neue Gateway-Tests,
  `TestOpenAPIRouteDrift` gruen. Eigene Tenants (`6b0e0000-…`), nie TenantA/B — diese Tests
  zaehlen die Administratoren *eines Tenants*.
- openapi.yaml: ein Pfad mit drei Operationen + drei Schemas, alle real gelieferten Codes
  (400/401/403/404/409/422 bzw. 204). Zusaetzlich drei **veraltete Aussagen** korrigiert, die
  jetzt gelogen haetten ("per-user overrides do not exist server-side yet" bei `?base=1`, bei
  `hasOverrides` im Roster-Schema und in der Roster-Beschreibung) — die Speicherung existiert
  jetzt, die Aufloesung nicht.
- offen:
  - `g-rbac-user-overrides-resolver` (naechste Unit, deps jetzt erfuellt): Aufloesung,
    `hasOverrides`, `deniedByOverride`, `sources: ['override', …]`, `?base=1`. Erst danach
    aendert ein Override tatsaechlich ein Gate — bis dahin schreibt und liest der Editor Zeilen,
    die nichts schalten. Das ist Absicht und in der Migration so kommentiert.
  - `hasOverrides` im Roster (`/admin/users`) joint die Tabelle noch nicht — das FE-Badge
    "Angepasst" und der Filter "Nur angepasste Benutzer" (Briefing §0.4) haengen daran. Eigene
    kleine Unit, gehoert nicht in den Resolver.
  - Der MSW-Mock (`handlers/rbac.ts:257-318`) kennt keine der vier Guardrails: er laesst
    Selbst-Bearbeitung zu, prueft weder Eskalation noch Last-Admin noch den Katalog. Die UI wird
    daran entwickelt — gehoert in eine FE-Unit, sonst faellt der Editor erst gegen das echte
    Backend um.
  - DB-Gate lief lokal vollstaendig (Docker-Postgres erreichbar), kein Nachlauf noetig.

## Iteration 40 — g-rbac-user-overrides-resolver — done — 2026-08-03 05:45
- commit: 18bf5c18
- gebaut: Die Per-User-Overrides sind in der Rechte-Aufloesung angekommen.
  `Service.GetEffectivePermissions` = Rollen-Union + `applyOverrides()`, **eine** Naht wie die
  MSW-Referenz. Der alte Union-Code heisst jetzt `Service.GetRoleUnion` und ist unveraendert.
  `GET /admin/users/{id}/permissions?base=1` liefert diese Union (neues Proto-Feld `base`), ohne
  `base` die Overrides. Beide Endpoints tragen `hasOverrides` und `deniedByOverride`.
- Semantik, exakt nach `applyUserOverrides` (mocks/data/rbac.ts), nicht nach dem scope-Text:
  - `allow` **setzt** den Scope, es hebt ihn nicht nur. Der Referenzcode tut das (`capabilities[key]
    = { scope: override.scope, ... }`), und Runterregeln fuer eine Person ohne Rollen-Klon ist die
    halbe Daseinsberechtigung des Features. Der Backlog-Text sagte "setzt oder hebt".
  - `deny` entfernt den Key immer, aber `deniedByOverride` bekommt nur einen Eintrag, wenn eine
    Rolle ihn wirklich gab — die Liste zeigt, was WEGGENOMMEN wurde. Ein deny auf einen nie
    gehaltenen Key waere sonst als "Recht entzogen" sichtbar, das es nie gab.
  - Provenance: der Sentinel `override` steht als LETZTER Eintrag in `sources`, hinter den Rollen,
    die den Key auch geben. `EffectivePermissionsView` filtert ihn heraus und rendert den Rest als
    Chips — Position egal, Reihenfolge aber lesbar.
  - Unbekannter `mode` faellt in den deny-Zweig, nicht in den allow-Zweig. Dieselbe Richtung, in
    die `narrowestScope` irrt: zu viel Recht ist der Fehler, den man nicht machen darf.
- **Kernentscheidung: `NarrowedScopes` bleibt bewusst auf `GetRoleUnion`.** Der scope-Claim kann
  kein "denied" ausdruecken — ein fehlender Key liest sich dort als `all` (steht so in der
  Funktion). Ein deny-Override wuerde den Key aus der Map werfen und damit seine WEITESTE
  Reichweite freigeben, statt sie zu entziehen. Das Gate, das das auffangen muesste, ist der
  permissions-Claim, und der wird aus `GetUserPermissions` gebacken, die von Overrides nichts
  weiss. Solange die eine override-blind ist, muss die andere es auch sein. Zweiter, unabhaengiger
  Grund: `createTokenPair` laeuft unter `sysctx.With()` (service.go:79 u.a.) — dort filtert RLS
  nicht, ein Override-Read braeuchte dort einen expliziten Tenant-Filter (Fund aus Iteration
  14/15). Beides steht als Kommentar an `NarrowedScopes` und als Unit `g-rbac-user-overrides-token`
  im Backlog. Ein Regressionstest haelt es fest.
- Wire: beide neuen Felder mit `omitempty`. Fuer `?base=1` und fuer jedes Konto ohne Override
  faellt beides aus dem JSON — die Antwort ist damit **byte-identisch** zu der vor R-6, was der
  bestehende `TestToEffectivePermissionsBody_WireShape` unveraendert beweist. Das FE defaultet
  beide (`deniedByOverride = []`, `hasOverrides = false`, EffectivePermissionsView:35 /
  permissions.ts:113). `sources` in der Spec ist nicht mehr `format: uuid` — der Sentinel ist
  keine UUID.
- `?base=1` wertet nur die exakte "1"; jeder andere Wert liefert die effektive Menge. Wer sich
  vertippt, bekommt die sicherere der beiden Antworten.
- gate: build ok (`go build -p 2` ueber auth, server, gateway, cmd/auth, cmd/gateway) | vet ok |
  lint ok (golangci-lint, 0 issues) | proto regeneriert im selben Commit (`auth.pb.go`;
  `auth_grpc.pb.go` unveraendert — keine neuen RPCs) | migration n.a. (kein Schema-Change) |
  rls-smoke n.a. | test ok: `go test -count=1 ./internal/auth/... ./internal/gateway/
  ./internal/server/...` mit `DATABASE_URL` auf `kmuhub_app` — alles gruen. **7 neue DB-Tests,
  einzeln mit `-v` als PASS verifiziert, 0 SKIP im ganzen auth-Paket** (`grep -c '^--- SKIP'` = 0),
  2 neue Gateway-Marshal-Tests, `TestOpenAPIRouteDrift` gruen.
- verify vorgaenger (`de6eb85a`, g-rbac-user-overrides-model): **sauber**. Handler gehen ueber
  `client.<RPC>` statt einer Service-Instanz, kein Stub/TODO im neuen Pfad, `.proto` mit `.pb.go`
  und `.grpc.pb.go` im selben Commit, Tabelle mit `tenant_id NOT NULL` + `enable_tenant_rls`,
  Wire-Shape gegen `rbac-types.ts` gepinnt. Die Seed-Behauptung des Commits gegen die DB geprueft
  statt geglaubt: `admin:user_override:manage` existiert im Katalog und haengt am `admin`-Preset
  (und nur dort) — der Guard sperrt also niemanden aus.
- **FUND ausserhalb des Vorgaenger-Commits, CI-blockierend:** `backend/api/openapi.yaml` ist seit
  den beiden HR-Units schema-invalide. `npx @apidevtools/swagger-cli validate` (CI-Job "Validate
  OpenAPI", ci.yml:387) meldet **75 Fehler** — identisch vor und nach meiner Aenderung, also
  Bestand, nicht neu. Ursache: Responses im YAML-**Flow-Stil** mit Kommas in der Beschreibung
  (`"400": { description: Malformed body, unknown exit type, ... }`) — das Komma trennt im
  Flow-Mapping Schluessel-Paare, also wird `unknown exit type` als weiterer Key gelesen.
  Betroffen: `POST /hr/employees/{id}/offboard` (400/403/409), `/hr/change-requests` (GET/POST)
  und `ProfileChangeRequest`. `TestOpenAPIRouteDrift` faengt das nicht — er prueft Pfad-Existenz,
  nicht Validitaet. Als `fix-openapi-flow-mapping-invalid` **ganz vorne** im Backlog angelegt,
  Ursache dort notiert, damit die naechste Iteration nicht erneut sucht.
- offen:
  - Ein Override aendert weiterhin **kein Gate** — nur die Anzeige. Das ist Absicht und begruendet
    (siehe Kernentscheidung); `g-rbac-user-overrides-token` schliesst es.
  - `hasOverrides` im Roster (`/admin/users`) joint die Tabelle immer noch nicht — das FE-Badge
    "Angepasst" und der Filter haengen daran. Weiterhin eigene kleine Unit, gehoert nicht hierher.
  - Die OpenAPI-Aussage am Roster ("hasOverrides is omitted") bleibt korrekt und wurde bewusst
    nicht angefasst.
  - DB-Gate lief lokal vollstaendig (Docker-Postgres erreichbar), kein Nachlauf noetig.

## Iteration 41 — fix-openapi-flow-mapping-invalid — done — 2026-08-03
- commit: 935c07b8
- verify vorgaenger (`18bf5c18`, g-rbac-user-overrides-resolver): **sauber**. Handler in
  `route_auth.go` geht ueber `client.GetEffectivePermissions`, kein Stub. `GetUserOverrides`
  (postgres_repository.go:710) ist RLS-gescopt, nicht `sysctx` — richtig, weil dieser Pfad (anders
  als `createTokenPair`) nicht unter Systemkontext laeuft. `NarrowedScopes` bewusst auf
  `GetRoleUnion` belassen, mit Begruendung im Code und Backlog-Unit `g-rbac-user-overrides-token`
  verlinkt. Proto: `auth.pb.go` regeneriert, `auth_grpc.pb.go` unveraendert (keine neuen RPCs,
  konsistent mit dem Commit-Text). Eigenstaendig nachgerechnet statt geglaubt: `go build -p 2 ./...`,
  `go vet ./...`, `golangci-lint run` (0 issues), `go test -count=1 ./internal/auth/...
  ./internal/gateway/... ./internal/server/...` — alle gruen, 0 SKIP im auth-Paket
  (`grep -c '^--- SKIP'` = 0, unabhaengig nachgezaehlt). Der CI-blockierende openapi.yaml-Fund aus
  Iteration 40 bestaetigt: `swagger-cli validate` liefert weiterhin 75 Fehler, identisch vor und
  nach `18bf5c18` — Bestand, nicht durch diesen Commit verursacht.
- gebaut: `fix-openapi-flow-mapping-invalid`. Zwei Fund-Orte, nicht nur der aus Iteration 40
  vermutete: der Offboard-Block (400/403/409/409, Kommas in der Beschreibung, wie erwartet) UND das
  eigentliche Wurzelproblem fuer die GET/POST-200/201-Fehler auf `/hr/change-requests` — die
  Fehlermeldungen zeigten dort scheinbar auf die (bereits Block-Stil) Response-Objekte selbst, aber
  `swagger-cli` mit vollem Fehlerpfad zeigte: der eigentliche Bruch sass in
  `components/schemas/ProfileChangeRequest`, dessen Feld `reason` im Flow-Stil mit Komma in der
  Beschreibung geschrieben war (`reason: { type: string, description: Rejection reason, absent
  otherwise }`). Weil GET/POST per `$ref` auf dieses Schema zeigen, propagierte
  "must NOT have additional properties" auf `reason` bis in beide Routen hoch. Ohne den vollen
  Fehlerpfad (`#/paths/.../schema/properties/requests/items/properties/reason`) waere das leicht mit
  einem echten Route-Bug verwechselt worden.
  Alle betroffenen Flow-Mappings — Offboard, change-requests GET/POST, approve/reject/cancel, das
  Schema — in Block-Stil aufgeloest. Jede Aussage woertlich uebernommen, nur die YAML-Form geaendert
  (per `git diff` gegengeprueft: keine Wortaenderung, nur Einrueckung).
- gate: `npx @apidevtools/swagger-cli validate backend/api/openapi.yaml` -> "is valid" (vorher 75
  Fehler) | `go build -p 2 ./...` ok | `go vet ./...` ok | `go test -count=1 ./internal/gateway/`
  gruen, `TestOpenAPIRouteDrift` + `TestOpenAPIRouteDriftParserSanity` beide PASS (802 registrierte
  Routen gegen 804 dokumentierte Pfade, unveraendert gegenueber vorher).
- offen: keins Neues. Die von Iteration 40 offen gelassenen Punkte
  (`g-rbac-user-overrides-token`, Roster-`hasOverrides`) bleiben unberuehrt — reine Docs-Unit ohne
  Code-Aenderung in `internal/`.
- DB-Gate lief lokal vollstaendig (Docker-Postgres erreichbar), kein Nachlauf noetig.

## Iteration 42 — g-rbac-user-overrides-token — done — 2026-08-03
- commit: a7ae3f56
- verify vorgaenger (`935c07b8`, fix-openapi-flow-mapping-invalid): **sauber**. Reine YAML-Form,
  keine Code-Aenderung in `internal/`. Diff eigenstaendig durchgesehen: alle sechs Bloecke
  (Offboard, change-requests GET/POST, approve/reject/cancel) plus `ProfileChangeRequest.reason`
  von Flow- auf Block-Stil, kein Wort und kein Status-Code veraendert. Nicht geglaubt, sondern
  nachgerechnet: `swagger-cli validate` -> "is valid" (vorher 75 Fehler), und danach das ganze
  File nach verbliebenen Flow-Mappings mit Komma in der Beschreibung durchsucht — genau ein
  Treffer (`openapi.yaml:24124`, ein Parameter-Objekt, in dem das Komma korrekt `description` von
  `schema` trennt), also kein Rest. `TestOpenAPIRouteDrift` unveraendert gruen.
- gebaut: `g-rbac-user-overrides-token`.
- **Kernfund, der die Unit umgeschrieben hat:** die Praemisse "deny = Key aus dem
  permissions-Claim entfernen" traegt nicht. Gemessen statt vermutet: 154 der 164
  `RequirePermissionAny`-Aufrufe im Gateway paaren einen groben Legacy-Key mit dem feinen
  Katalog-Key, der ihn ersetzen soll (der Doc-Kommentar der Funktion verlangt das ausdruecklich:
  "extend, never swap"). Faellt bei einem deny nur der feine Key weg, haelt der grobe daneben die
  Tuer offen — der Override waere in 94 % dieser Gates folgenlos geblieben, und zwar unsichtbar.
  Deshalb ein dritter Claim `Denied` (`den`, omitempty): ein grober Key zaehlt nicht mehr als
  Platzhalter, sobald IRGENDEIN feiner Key DESSELBEN Guards explizit denied ist. Zwei feine Keys
  in einem Guard bleiben dagegen eigenstaendig — `Any(berichte:reports:read, berichte:export:run)`
  sind zwei verschiedene Rechte an derselben Route, und ein deny auf den Export darf das Lesen
  nicht mitnehmen. Die Trennlinie grob/fein ist dieselbe, die
  `PostgresRepository.GetEffectivePermissions` mit `resource LIKE '%:%'` zieht.
- **Beide Fallen der Unit-notes adressiert:**
  (1) *Nur zusammen, nie einzeln.* `NarrowedScopes` ist in `Service.ResolveTokenPermissions`
  (neue Datei `token_permissions.go`) aufgegangen; die drei Claim-Teile entstehen aus einem Read
  in einer Funktion, was ein Auseinanderlaufen strukturell ausschliesst. Die Signatur von
  `CreateAccessToken` nimmt jetzt ein `*TokenPermissions` statt drei lose Parameter — dieselbe
  Absicht, im Typsystem. Ein denied Key wird im scope-Claim auf `own` GESCHRIEBEN, nicht entfernt:
  Abwesenheit liest sich dort als `all`, ein Entfernen haette also die weiteste Reichweite des
  gerade entzogenen Rechts ausgeliefert. Verteidigung in der Tiefe fuer den Fall, dass ein
  Lesepfad den Scope hinter einem grob bewachten Gate abfragt.
  (2) *sysctx.* Neue Repo-Methode `GetUserOverridesForTenant` mit explizitem
  `WHERE tenant_id = $1 AND user_id = $2` (nutzt `idx_user_permission_overrides_user`); die
  RLS-Variante bleibt fuer die Request-Pfade unveraendert. Regressionstest seedet eine
  Override-Zeile mit FREMDEM `tenant_id` auf denselben User — genau die Form, die ein Read ohne
  Tenant-Bedingung unter sysctx aufgesammelt haette — und prueft, dass sie den Token nicht
  erreicht.
- **Formfrage (die eigentliche Arbeit laut notes):** der Allow-Satz wird aus `GetUserPermissions`
  (grob UND fein) fortgeschrieben, nie aus dem Union neu gebaut. Ein Neubau aus den feinen
  Katalog-Keys haette alle 226 `RequirePermission`-Gates mit grobem Key beim ersten Override eines
  Accounts auf 403 gesetzt. Eigener DB-Test dafuer.
- entschieden: `RequirePermission` (Einzelkey) liest die Deny-Liste NICHT. Ein denied Key fehlt
  dort ohnehin im Allow-Satz, und ein Einzelkey-Guard hat keinen zweiten Key, der einspringen
  koennte — der Check waere reine Redundanz. Als Kommentar an der Funktion begruendet, damit die
  Asymmetrie zu `RequirePermissionAny` nicht wie ein Versehen aussieht.
- entschieden: "wirkt erst mit dem naechsten Token" steht in openapi.yaml an PUT und DELETE
  `/admin/users/{id}/overrides`, nicht als Feld in der Antwort. Die Aussage ist eine Konstante
  ueber den Mechanismus, kein Datum dieser Anfrage; ein Response-Feld haette in jeder Antwort
  denselben Wert. Der Text nennt auch den sichtbaren Widerspruch, der daraus folgt:
  `/auth/me/permissions` loest live auf und zeigt den neuen Stand sofort, der Token nicht.
- ersetzt: `TestNarrowedScopes_DB_StaysOnTheRoleUnion` hielt bewusst das ALTE Verhalten fest
  ("der Claim darf sich nicht bewegen, solange der permissions-Claim nichts von Overrides weiss").
  Diese Bedingung ist mit dieser Iteration erfuellt, der Test daher durch
  `TestResolveTokenPermissions_DB_DenyNarrowsInsteadOfDropping` ersetzt, der dieselbe Gefahr aus
  der neuen Richtung pruft (Key bleibt in der Map, aber auf `own`).
- gate: `go build -p 2 ./...` ok | `go vet ./...` ok | `golangci-lint run` -> 0 issues |
  `swagger-cli validate backend/api/openapi.yaml` -> "is valid" |
  `go test -count=1 ./internal/auth/... ./internal/middleware/... ./internal/gateway/
  ./internal/server/...` alle gruen | `go test -count=1 -v ./internal/auth/`: **207 PASS, 0 SKIP**
  (unabhaengig gezaehlt, `DATABASE_URL` auf `kmuhub_app` gesetzt — ohne sie waeren die vier neuen
  DB-Tests still uebersprungen worden und der Lauf haette trotzdem `ok` gemeldet).
- offen:
  - `hasOverrides` im Roster (`/admin/users`) joint die Tabelle weiterhin nicht — FE-Badge
    "Angepasst" und Filter haengen daran. Unveraendert eine eigene kleine Unit, nicht Teil dieser.
  - Ein Override auf einen GROBEN Legacy-Key ist mechanisch moeglich (der Katalog-Join laesst ihn
    durch) und funktioniert auch korrekt, aber der Override-Editor im FE bietet nur feine Keys an.
    Kein Bug, nur eine Asymmetrie, die jemand kennen sollte, der die Tabelle direkt befuellt.
- DB-Gate lief lokal vollstaendig (Docker-Postgres erreichbar), kein Nachlauf noetig.

## Iteration 43 — g-automation-http-action — done — 2026-08-03

- verify-vorspann auf `a7ae3f56` (Iteration 42, per-user Overrides im Token): 21 Dateien geprueft,
  keine der sechs Fehlerklassen. Keine Stubs/`Unimplemented`, kein Gateway-Bypass (nur
  `internal/auth`, `internal/middleware`, `internal/server` — keine Route-Datei), kein Proto
  angefasst, kein neuer `RequirePermission`-Guard (also kein Seed faellig), Wire-Shape unberuehrt.
  Der Kern der Aenderung — `RequirePermissionAny` laesst einen groben Key nicht mehr als Platzhalter
  zaehlen, sobald ein feiner Key desselben Guards denied ist — im Diff nachgelesen und schluessig.
  `go build -p 2 ./...` gruen.
- gebaut: `g-automation-http-action`.
- **Der Schutz sitzt im Dialer, nicht in einer URL-Pruefung.** Das ist die eine Entscheidung, an der
  diese Unit haengt. Eine Pruefung der konfigurierten URL beantwortet die falsche Frage: sie sieht
  einen Namen, verbunden wird aber mit einer Adresse, und zwischen beiden liegt eine zweite
  DNS-Aufloesung, die ein Angreifer kontrolliert (Rebinding). Ausserdem muesste sie bei jedem
  Redirect wiederholt werden. `net.Dialer.Control` laeuft nach der Aufloesung und vor `connect()`,
  bekommt das Literal, mit dem der Kernel spricht — und weil jeder Redirect-Hop eine eigene
  Verbindung aufmacht, ist er ohne Zusatzcode mitgeprueft. Die URL-Pruefung (`safehttp.CheckURL`)
  bleibt als Komfort fuer eine praezise Fehlermeldung, ist aber ausdruecklich nicht die Grenze —
  steht so im Doc-Kommentar, damit niemand sie spaeter fuer den Schutz haelt.
- **Zweiter Fundort, gleiche Luecke, bereits produktiv.** Vor dem Bauen nach vorhandenen ausgehenden
  Clients gesucht (Lean-Leiter Stufe 2). `internal/formulare/worker.go:56` verschickt
  Formular-Webhooks an `webhook.URL` — tenant-konfiguriert, aus der DB — mit einem blanken
  `&http.Client{Timeout: 10s}`, ohne Adressfilter, mit Go-Default-Redirects. Das ist dieselbe
  SSRF-Falle wie die neue Action, nur seit Monaten scharf. Deshalb wurde der Guard als eigenes Paket
  gebaut statt in die Action gelegt, und der Worker ist auf eine Zeile umgestellt. Root Cause statt
  Symptom, und der Diff ist kleiner als zwei getrennte Guards.
  Wichtig dabei: ALLE bestehenden Erfolgstests des Workers injizieren `server.Client()` und haetten
  einen Totalausfall der Zustellung durch den Guard nicht bemerkt. Deshalb zwei neue Tests — Sperre
  greift (`169.254.169.254` -> failed, Grund im `last_error`) und Zustellung laeuft durch den
  Guard weiterhin. Der bestehende `TestWorker_NetworkError_IncrementsAttempt` faengt jetzt zwar
  weiterhin, aber aus einem anderen Grund (Adressfilter statt Connection-Refused); inhaltlich
  unveraendert richtig, hier nur vermerkt, damit es niemanden spaeter verwirrt.
- **Test-Seam ohne Produktionsrisiko.** Ein `httptest`-Server lebt auf Loopback, also braucht es eine
  Ausnahme — und eine exportierte Ausnahme ist genau die Zeile, die spaeter in Produktionscode
  kopiert wird. `safehttp.AllowLoopback()` setzt sein Flag deshalb ueber `testing.Testing()`: unter
  `go test` wirkt es, im gebauten Binary ist es tot. Kein Env-Var, keine Config, nichts, das im
  Betrieb umgelegt werden koennte. `TestNew_LoopbackBlockedWithoutOption` haelt fest, dass der
  Konstruktor ohne Optionen — der, den `cmd/` benutzt — Loopback ablehnt.
- ueber die notes hinaus gefunden und mitgeschlossen:
  - Go entfernt beim Redirect auf einen fremden Host nur `Authorization`/`Cookie`. Ein
    tenant-konfigurierter `X-Api-Key` waere an das Redirect-Ziel weitergereicht worden — der
    naheliegendste Weg, einem Angreifer den API-Schluessel des Tenants zuzustellen. Beim Hostwechsel
    fliegen jetzt alle Header ausser einer harmlosen Liste raus (Test fuer beide Richtungen: fremder
    Host verliert ihn, gleicher Host behaelt ihn).
  - `Proxy: nil` am Transport. `http.DefaultTransport` liest `HTTP_PROXY` aus der Prozessumgebung;
    ein Proxy wuerde am Dial-Filter vorbeirouten, weil die Verbindung dann zum Proxy geht und das
    Ziel im CONNECT steht. Im Repo ist keiner konfiguriert (`deploy/` gegengeprueft).
  - Header-Werte sind Templates mit Datensatzinhalten. Die CR/LF-Pruefung laeuft deshalb NACH der
    Aufloesung — sonst schmuggelt ein Kontakt-Notizfeld einen zweiten Header ein. Eigener Test; der
    abgelehnte Wert wird nicht in die Fehlermeldung zurueckgespiegelt (die landet im Execution-Log).
  - CONNECT und TRACE verboten. TRACE spiegelt die Request-Header zurueck und ist zusammen mit einem
    konfigurierten Auth-Header eine Credential-Disclosure gegen den eigenen Endpunkt.
- entschieden: ein 4xx/5xx ist `Success: false` **ohne** Go-Error, mit Status und Body im Output. Der
  `on_error`-Zweig der Automation kann nur entscheiden, wenn der Status ehrlich durchgereicht wird;
  ein verschluckter 403 saehe im Execution-Log aus wie ein Netzproblem.
- entschieden: Body-Cap zweistufig. 1 MiB am Client (Speicher), zusaetzlich 4096 Zeichen im Output —
  der Env wandert bei JEDEM Lauf nach `automation_executions`, eine 900-KB-Antwort waere also nicht
  einmal, sondern pro Ausfuehrung in der DB. Ueberschreitung ist als `body_truncated` sichtbar, nicht
  still. Der Reader wirft `ErrResponseTooLarge` statt zu kuerzen, damit ein abgeschnittenes JSON
  nicht als Parse-Fehler weit weg von der Ursache auftaucht — mit einem Byte Kopffreiheit, sonst
  waere ein Body von exakt Cap-Groesse fehlgeschlagen (eigener Regressionstest fuer beide Faelle).
- nicht gebaut, bewusst: JSON-Parsing der Antwort in einzelne Output-Keys (`resolveTemplate` kann nur
  flache Keys — eigene Unit, wenn jemand es braucht) und eine per-Tenant-Allowlist erlaubter
  Zielhosts (Produktentscheidung, kein Sicherheitsloch: der Filter haelt auch ohne sie).
- keine neue Route, keine Migration, kein neuer Guard -> `openapi.yaml` unveraendert.
- gate: `go build -p 2 ./...` ok | `go vet` (safehttp, automation, formulare, cmd/automation) ok |
  `golangci-lint run` auf denselben -> **0 issues** | `swagger-cli validate backend/api/openapi.yaml`
  -> "is valid" | `go test -count=1 ./internal/gateway/` gruen (TestOpenAPIRouteDrift) |
  `go test -count=1 -v ./internal/security/safehttp/ ./internal/automation/action/`:
  **66 PASS, 0 SKIP** (unabhaengig gezaehlt) | `go test -count=1 -v ./internal/formulare/`: 58 gruen |
  `./internal/automation/...` vollstaendig gruen.
- offen fuer die naechste Iteration: `g-automation-webhook-trigger` (haengt an dieser Unit) ist der
  eingehende Gegenpart. Das dort geforderte konstantzeitige Signatur-Pruefen gibt es im Repo schon
  in `internal/biz/lexware/webhook_handler.go` und `internal/formulare/worker.go` (HMAC-SHA256,
  `X-Cosmi-Signature: sha256=…`) — dieselbe Form wiederverwenden statt eine dritte zu erfinden.
- commit: `cac9f73d` (feat(automation): add an SSRF-guarded http.request action)

## Iteration 44 — g-automation-webhook-trigger — done — 2026-08-03

- verify-vorspann auf `cac9f73d` (Iteration 43, SSRF-guarded http.request action): Diff gelesen
  (`internal/security/safehttp/`, `internal/automation/action/http_actions.go`,
  `internal/formulare/worker.go`, Tests). Keine der sechs Fehlerklassen: kein Stub, kein
  Gateway-Bypass (reines Aktions-/Worker-Paket, keine Route), kein Proto/Migration angefasst, kein
  neuer Guard. `go build -p 2 ./...` gruen.
- gebaut: `g-automation-webhook-trigger`.
- Neuer Trigger `webhook.received` in der Registry, neue Route
  `POST /api/v1/public/automations/webhooks/{automationId}` (kein Auth, publicRateLimiter wie
  booking/berichte/document), neue RPC `AutomationService.TriggerWebhook`.
  **Tenant-Aufloesung ohne JWT** ist die zentrale Entscheidung: neue Repo-Methode
  `GetByIDUnscoped` (kein `WHERE tenant_id`) hinter `sysctx.With`, Trigger-Type/`is_active` danach
  im Service geprueft — dieselbe Grenze wie bei `two_factor_policy`/`refresh_tokens`. Signatur
  HMAC-SHA256, Header `X-Cosmi-Signature: sha256=<hex>` (Vorgabe aus Iteration 43, dieselbe Form wie
  das ausgehende `internal/formulare/worker.go`), Vergleich konstantzeitig. Secret liegt in
  `trigger_config.secret`, wird beim Erstellen/Aendern einer `webhook.received`-Automation
  automatisch generiert (`ensureWebhookSecret`), sonst existiert die Automation nie ohne Secret.
  Idempotenz wiederverwendet den bestehenden `internal/idempotency`-Store statt einer eigenen
  Dedup-Tabelle: expliziter `Idempotency-Key`-Header wenn vorhanden, sonst SHA-256 des Bodies als
  Fallback — ein reiner Resend dedupt also auch ohne Header. Ausfuehrung laeuft asynchron
  (Goroutine, 30s-Timeout), Muster wie `trigger.TimeTriggerPoller` — schneller 202 fuer den
  Absender, eine haengende Action haelt die Verbindung nicht offen.
- **Root-Cause-Fund unterwegs, im selben Commit behoben:** `engine.Execute`s Loop-Prevention
  verwirft jedes Event mit `ModuleID == "automation"` — genau das setzte `trigger.TimeTriggerPoller`
  fuer seine eigenen synthetischen Events. `biz.invoice.overdue` und `calendar.event.upcoming` sind
  dadurch seit ihrem Bau nie wirklich gelaufen (stiller Drop vor jeder Condition-Auswertung, nur ein
  Debug-Log). Waere ich demselben Muster fuer den neuen Webhook-Trigger gefolgt, waere derselbe
  Fehler entstanden und "Trigger startet die Automation" schlicht falsch gewesen. Fix: Poller-Event
  traegt jetzt `ModuleID: "scheduler"`, mein Webhook-Event `event.ModuleIntegration`
  ("integration" — vorher deklariert, nie genutzt). Zwei neue Regressionstests in `engine_test.go`
  (`TestExecute_SkipsAutomationModuleEvent`, `TestExecute_SchedulerOriginedEventIsNotSkipped`)
  belegen beide Seiten.
- **Gefunden, bewusst nicht angefasst:** `internal/idempotency.PostgresRepository.Reserve`
  unterscheidet unter der echten Postgres-Implementierung "frisch" und "gleichzeitig in-flight,
  noch nicht completed" nicht — beide liefern `(nil, nil)`. Der `mockRepository` in
  `internal/idempotency/repository_test.go` (gegen den `TestReserve_RaceCondition_InFlight` laeuft)
  tut es korrekt; die echte Postgres-Variante ist gegen dieses Szenario nie getestet. Sehr enges
  Zeitfenster, betrifft aber Infrastruktur, die Dutzende bereits produktive Endpunkte nutzen —
  gehoert als eigene Sicherheits-Unit auf den Backlog, nicht in diesen Commit gezogen. Fundstelle:
  `internal/idempotency/postgres_repository.go`, `Reserve`, `CompletedAt == nil`-Zweig.
- Neue Repo-Methode `GetByIDUnscoped` DB-getestet (`webhook_db_test.go`, Rolle `kmuhub_app`, je
  Testlauf ein eigener frischer Tenant statt der geteilten `TenantA`/`TenantB`): unter sysctx ohne
  jeden Tenant im Context aufloesbar, das normale tenant-gescopte `GetByID` verweigert weiterhin
  einen fremden Tenant. Zusaetzlicher DB-Test `TestTriggerWebhook_RealRepositories` faehrt den
  kompletten Pfad gegen die echten Postgres-Repos (nicht Mocks) inkl. Dedup ueber zwei echte Calls.
- Body-Limit zweistufig: `http.MaxBytesReader` (256 KiB) im Gateway-Handler UND derselbe Cap im
  Service (Verteidigung in der Tiefe — Service-Tests laufen ohne Gateway und muessen den Cap
  trotzdem pruefen koennen).
- keine neue `config.RequireX`-Assertion, kein neues `modules.*`-Flag scharfgeschaltet.
- gate: `go build -p 2 ./...` ok | `go vet -p 2 ./...` ok | `golangci-lint run` auf
  `internal/automation/...`, `internal/gateway/...`, `internal/server/...`, `cmd/automation/...`,
  `cmd/gateway/...` -> **0 issues** | `swagger-cli validate backend/api/openapi.yaml` -> "is valid" |
  `go test -count=1 ./internal/gateway/...` gruen (`TestOpenAPIRouteDrift` + `TestOpenAPISpecDrift`) |
  mit `DATABASE_URL` gegen `kmuhub_app` (lokaler Docker-Postgres, Migrationskopf 283):
  `go test -count=1 ./internal/automation/... ./internal/gateway/... ./internal/server/...
  ./internal/idempotency/...` vollstaendig gruen, die beiden neuen DB-Tests liefen tatsaechlich
  (nicht uebersprungen). Proto regeneriert im selben Commit (`protoc` direkt, `make` nicht verfuegbar
  unter Git Bash).
- offen fuer die naechste Iteration: `g-automation-cron-poller` (haengt an dieser Unit) — der
  Poller-Bug ist bereits gefixt (siehe oben), nicht erneut suchen. Dort geht es nur noch um die
  Doppelausfuehrungssperre analog `g-berichte-scheduler`.

## Iteration 45 — g-automation-cron-poller — done — 2026-08-03 06:55

- verify-vorspann auf `45e0dc91` (Iteration 44, webhook.received-Trigger): Diff gelesen
  (`route_automation.go`, `automation_grpc.go`, `webhook.go`, `poller.go`, proto + Regen,
  openapi.yaml, Tests). Handler geht ueber `client.TriggerWebhook` (kein Gateway-Bypass), gRPC-Server
  liest bewusst keinen Tenant aus dem Context (dokumentiert, Tenant kommt aus der Automation-Zeile
  unter sysctx), `.proto` + `.pb.go`/`.grpc.pb.go` im selben Commit regeneriert, keine neue
  `RequirePermission` (Route ist bewusst unauthenticated, kein Seed noetig), keine Tenant-Luecke
  (Webhook-Secret + sysctx-Read + explizite Trigger-Type/Active-Pruefung), Wire-Shape
  `{duplicate: bool}` passt zum simplen Accepted-Response, Route in openapi.yaml eingetragen
  (`TestOpenAPIRouteDrift` gruen). Kein Fund. Sauber.
- gebaut: `g-automation-cron-poller`.
- **Zweiter, tieferer Bug gefunden, der den in Iteration 44 gefixten ModuleID-Bug wirkungslos
  gehalten haette:** `ListActiveTimeBased` filterte `WHERE trigger_type = 'time_based'` — dieser
  String ist in der 14-Eintraege-Trigger-Registry NIRGENDS registriert (die echten Typen heissen
  `biz.invoice.overdue`, `calendar.event.upcoming` usw.), und `validateAutomation` lehnt beim
  Anlegen jeden nicht-registrierten `trigger_type` ab — die Query lieferte also strukturell IMMER
  null Zeilen. Der Poller lief brav alle 5 Minuten, fand aber nie etwas zu tun. Ohne diesen Fix waere
  "Faellige Automation laeuft" aus dem `done_when` schlicht falsch gewesen, unabhaengig von einer
  Ausfuehrungssperre.
- Fix: `Repository.ListActiveTimeBased(ctx, triggerTypes []string)` nimmt jetzt die konkrete
  Typliste vom Aufrufer entgegen statt selbst zu entscheiden, was "zeitbasiert" heisst.
  `trigger.TimeTriggerPoller` berechnet sie einmalig bei Konstruktion aus der Registry
  (`timeBasedTriggerTypes()`, filtert `TriggerDefinition.TimeBased == true`) — bewusst nicht
  umgekehrt (Registry-Zugriff aus `workflow` heraus), das haette einen Importzyklus erzeugt
  (`trigger` importiert bereits `workflow`, nicht umgekehrt).
- **Ausfuehrungssperre** wie von der Unit gefordert, 1:1 gespiegelt von
  `berichte/scheduler.ClaimSchedule`: neue Methode `ClaimTimeTrigger` macht ein
  Optimistic-Concurrency-`UPDATE` auf einen Zeitstempel, `RowsAffected()==1` entscheidet claimed
  ja/nein. Neue Spalte `automations.last_polled_at` (Migration 000284) statt Wiederverwendung von
  `last_triggered_at` — Letzteres ist FE-sichtbar ("letzte Ausfuehrung",
  `AutomatisierungPage.tsx`/`automation-types.ts`) und wird nur nach erfolgreicher
  Actions-Ausfuehrung gesetzt; ein Claim-Versuch ist semantisch etwas anderes (kann gewinnen, obwohl
  die Condition danach false auswertet) und haette dort einen falschen Zeitstempel gezeigt.
  Die racy History-Pruefung `wasRecentlyExecuted` (Check-then-Act-Query gegen
  `automation_executions`) ist komplett entfernt, `execRepo` damit aus dem Poller-Konstruktor raus
  (war nur dafuer da) — ein Mechanismus statt zwei ueberlappenden, von denen einer racy war.
- "Verpasste Zeitfenster nicht nachholen-stampeden" war bereits durch das bestehende Design erfuellt
  (`Start()` feuert einmal sofort, danach fester 5-Minuten-Tick, kein Aufholen mehrerer verpasster
  Ticks) — keine Aenderung noetig, nur verifiziert.
- **Gefunden, bewusst nicht angefasst (ausserhalb des Scopes dieser Unit):** die synthetischen
  Poller-Events tragen kein `Payload` (`evt := models.EventPayload{Type, ModuleID, Timestamp}`),
  wodurch `engine.buildEnvFromPayload` ein leeres Environment baut. Eine Condition wie
  "invoice.days_overdue > 3" sieht dieses Feld nie — die Automation feuert bei jedem Tick
  unconditional statt nur beim tatsaechlichen Erreichen der Schwelle, weil nirgends einzelne
  ueberfaellige Rechnungen/anstehende Termine aufgeloest und pro Ressource ein Event gebaut wird.
  Eigenstaendiges, deutlich groesseres Feature (Resource-Level-Polling pro Trigger-Typ), stand nicht
  in den Notes dieser Unit — Kandidat fuer einen neuen Backlog-Eintrag
  `g-automation-resource-level-time-triggers`, nicht in diesem Commit geloest (analog zur
  Idempotency-Reserve-Race aus Iteration 44, die ebenfalls nur im Journal vermerkt und nicht selbst
  als Unit angelegt wurde).
- Tests: 2 neue DB-Tests (`workflow/time_trigger_db_test.go`,
  `TestListActiveTimeBased_FiltersByProvidedTypes` + `TestClaimTimeTrigger_AtomicClaim`, echte
  Postgres-Rolle `kmuhub_app`, je ein frischer Tenant statt TenantA/B), 5 neue Unit-Tests
  (`trigger/poller_test.go`: Registry-Flag-Abgleich, Claim-gated-Execute inkl. verlorenem Claim,
  leere Registry ueberspringt den Repo-Call komplett, Claim-Fehler blockt Ausfuehrung ohne die
  restliche Schleife abzubrechen).
- Migration 000284 (`automations.last_polled_at`, nullable, kein Backfill noetig) — up/down/up
  lokal gruen. RLS auf `automations` nach der Migration weiterhin `relrowsecurity=t` mit
  unveraenderter Policy `tenant_isolation` (per psql verifiziert, keine Policy angefasst, keine
  neue Tabelle).
- keine neue `config.RequireX`-Assertion, kein neues `modules.*`-Flag scharfgeschaltet, keine neue
  Route (openapi.yaml unveraendert, `swagger-cli validate` -> "is valid").
- gate: `go build -p 2` + `go vet` + `golangci-lint run` auf `internal/automation/...`,
  `internal/gateway/...`, `internal/models/...`, `cmd/automation/...`, `cmd/gateway/...` ->
  0 issues. Mit `DATABASE_URL` gegen `kmuhub_app` (lokaler Docker-Postgres, Migrationskopf 284):
  `go test -count=1 -v ./internal/automation/... ./internal/gateway/...` -> 768 PASS, 0 SKIP,
  0 FAIL, die beiden neuen DB-Tests liefen tatsaechlich (nicht uebersprungen).
- offen fuer Luke: Triage von `g-automation-resource-level-time-triggers` (siehe oben) als neue
  Backlog-Unit, falls gewuenscht — nicht in dieser Iteration angelegt.

## Iteration 46 — g-mails-multi-account — done — 2026-08-03

- verify-vorspann auf `d1efda2d` (Iteration 45, Cron-Poller): `git show --stat` gelesen, Diff deckt
  sich mit der Journal-Beschreibung. Keine Route/kein `.proto`/kein `RequirePermission` angefasst,
  Migration 000284 nur additiv (nullable Spalte), RLS unveraendert. Kein Fund. Sauber.
- gebaut: `g-mails-multi-account`. `email_accounts` hatte bereits `tenant_id NOT NULL` + RLS seit
  Migration 000124 — die einzige Sperre gegen Mehrfachkonten war `UNIQUE(user_id)` aus Migration
  000041. Migration 000285: Constraint faellt, neue Spalte `is_default BOOLEAN NOT NULL DEFAULT
  true` (jedes Bestandskonto wird automatisch sein eigenes Default, kein Backfill noetig, da vorher
  exakt 1 Konto/User existierte), Ausdrucks-Unique-Index
  `idx_email_accounts_user_default ON email_accounts(user_id) WHERE is_default` erzwingt "genau ein
  Default pro User" als DB-Invariante.
- **Kritischer Fund vor dem Bauen:** `cmd/notification/main.go:resolveAccountID` verlaesst sich
  laut eigenem Kommentar auf "at most one account per (user, tenant)" via
  `GetEmailAccount`/`GetByUserIDAndTenant`. Ohne Gegenmassnahme haette die Umstellung diesen und
  jeden aehnlichen Caller lautlos auf eine beliebige DB-Zeile statt "das eine Konto" umgestellt —
  kein Fehler, kein rotes Gate, einfach das falsche Konto. Fix: `GetByUserIDAndTenant` filtert
  zusaetzlich `is_default = true`; "das eine Konto" heisst ab jetzt "das Default-Konto", bestehende
  Caller bleiben unveraendert kompatibel.
- Zweiter Fund: `Delete` brauchte eine Promotion-Regel. Das Default-Konto bei vorhandenen weiteren
  Konten zu loeschen, haette sonst den User ohne Default zurueckgelassen und
  `resolveAccountID` haette 404 geliefert, bis jemand manuell ein neues Default setzt.
  `Service.Delete` promotet jetzt das aelteste verbleibende Konto (`ListByUserAndTenant`,
  `ORDER BY created_at ASC`) ueber dieselbe `SetDefault`-Methode, die auch der neue
  `SetDefaultEmailAccount`-RPC nutzt. Das letzte Konto zu loeschen laesst den User bewusst ohne
  Default zurueck (kein Konto = kein Default noetig).
- `Repository.SetDefault` ist EIN atomares `UPDATE ... SET is_default = (id = $1) WHERE
  tenant_id=$2 AND user_id=$3` statt eines Clear-dann-Set-Zweischritts (das Muster, das
  `email_signature.Service.SetDefault` verwendet) — vermeidet dessen Race bei einem Doppel-Klick,
  kein Mehraufwand gegenueber der Vorlage.
- `ErrAccountExists` entfernt: Create erlaubt jetzt beliebig viele Konten je User, die alte
  1:1-Pruefung war der einzige Aufrufer. Zugehoerige `mapEmailError`-Klausel mitentfernt.
- Proto: `ListEmailAccounts`, `SetDefaultEmailAccount` neu, `EmailAccountInfo.is_default` (Feld
  15), `.pb.go`/`.grpc.pb.go` im selben Commit regeneriert.
- Gateway: `GET /accounts/list` (User aus `middleware.GetUserID(ctx)`, NICHT aus einem
  client-gelieferten `?user_id=` wie die Nachbarrouten — das FE ruft `.list()` ohnehin ohne
  Parameter, also die Chance auf eine IDOR-freie Variante statt das bestehende Muster zu kopieren)
  und `POST /accounts/{id}/default` (kein Body, User wird serverseitig aus dem Konto aufgeloest).
  Beide Pfade + Schemas in openapi.yaml, `swagger-cli validate` -> "is valid",
  `TestOpenAPIRouteDrift` gruen.
- Wire-Shape gegen den Bestand geprueft, nicht geraten: FE-Client (`email-client.ts:97-99`) und
  MSW-Mock (`mocks/handlers/email.ts:56-58`) erwarten `GET /api/v1/email/accounts/list` ->
  `{accounts:[...]}` bereits seit einer frueheren FE-Iteration ("Account-Switcher/Unified-Inbox").
  Backend zieht jetzt nach — keine Frontend-Aenderung in diesem Commit (Backend-Nachtloop-Scope).
- RLS: keine neue Policy noetig (Tabelle war schon geschuetzt), aber Tabelle angefasst -> Smoke
  gelaufen ueber `TestTenantIsolation_Email/email_accounts` (echter `kmuhub_app`-DB-Test, eigener
  Tenant vs. fremder Tenant), PASS.
- keine neue `config.RequireX`-Assertion, kein neues `modules.*`-Flag scharfgeschaltet, kein neuer
  `RequirePermission`-Guard (beide neuen Routen haengen unter dem bestehenden
  `RequirePermission("email", ...)`-Muster der Nachbarrouten, kein Seed noetig).
- gate: `go build -p 2` + `go vet` + `golangci-lint run` auf `internal/email/...`,
  `internal/gateway/...`, `internal/server/...`, `internal/models/...`, `cmd/email/...`,
  `cmd/gateway/...`, `cmd/notification/...` -> 0 issues. Mit `DATABASE_URL` gegen `kmuhub_app`
  (lokaler Docker-Postgres, Migrationskopf 285): `go test -count=1 -v
  ./internal/email/account/...` -> 10 Testfunktionen mit Subtests, 0 SKIP (inkl. echtem DB-Test
  `TestTenantIsolation_Email`); `go test -count=1 ./internal/email/... ./internal/gateway/...
  ./internal/server/...` komplett gruen. migrate up/down/up lokal gruen (Kopf 285).
- offen fuer Luke: keine FE-Anbindung an `.list()`/`.default` in diesem Commit (Backend-Scope) —
  FE-Client/Mocks erwarten die Wire-Shape bereits, ein FE-seitiger Smoke-Test gegen das echte
  Backend steht noch aus.

## Iteration 47 — g-user-roles-rls — done — 2026-08-03

- verify-vorspann auf `364b5756` (Iteration 46, Mail-Multi-Account): `git show --stat` gelesen,
  Diff deckt sich mit der Journal-Beschreibung. Migration 000285 additiv, kein neues
  `RequirePermission`, RLS auf `email_accounts` unveraendert. Kein Fund. Sauber.
- gebaut: `g-user-roles-rls`, die letzte ungeschuetzte Tabelle des RBAC-Kerns. Migration 000286:
  `tenant_id` per Backfill aus `users.tenant_id`, NOT NULL, FK auf `tenants`,
  `CALL enable_tenant_rls('user_roles')` — Einzel-Policy reicht, weil eine Zuweisung immer einem
  echten Tenant gehoert (anders als bei `roles` gibt es hier keine System-Preset-Zeile, die eine
  Split-Policy noetig macht).
- **Kritischer Fund vor dem Bauen, wie in den Notes vermutet und real ausnutzbar:** Welle 1b (jetzt
  abgeschlossen) erlaubt tenant-eigenen Rollen denselben Namen wie ein Preset (Arbiter-Index liegt
  auf `COALESCE(tenant_id, sentinel)`, ein echter `tenant_id` kollidiert nie mit dem Sentinel der
  Presets). `AssignRole(ctx, userID, "member")` — der Pfad, den JEDE Registrierung durchlaeuft
  (`service.go:107`, unter `sysctx.With()`, RLS also wirkungslos) — matchte per `WHERE r.name = $2`
  beide Zeilen und haette jeden neuen Registranten, tenant-uebergreifend, zusaetzlich in die
  gleichnamige Custom-Rolle eines fremden Tenants gehaengt — eine Rechte-Uebertragung ueber die
  Tenant-Grenze, auslösbar durch nichts weiter als eine Registrierung. Fix sitzt in der Query, nicht
  in RLS (die unter `sysctx` ohnehin nicht filtert): `AssignRole`/`RemoveRole` loesen jetzt
  ausschliesslich System-Presets auf (`r.tenant_id IS NULL`); Custom-Rollen laufen ausschliesslich
  ueber den ID-basierten `AssignUserRole`/`RevokeUserRole`-Pfad aus 1b. Neuer Test
  `TestAssignRole_DB_PresetOnlyDespiteNameCollision` seedet eine Tenant-Rolle namens "member" und
  beweist: genau eine Zuweisung entsteht, die des Presets, im Tenant des Opfers — nicht die des
  Angreifer-Tenants.
- 13 Bestandsstellen mussten `tenant_id` beim Insert nachziehen (Repository: `AssignRole`,
  `AssignUserRole`, `AcceptInvitation`; dazu 10 Test-/Seed-Dateien), sonst waere die neue
  NOT-NULL-Spalte ein harter Bruch gewesen. Alle bereits vorhandenen Kommentare, die "user_roles hat
  keine RLS" behaupteten (postgres_repository.go, repository.go, roles_admin.go, route_auth.go,
  hr/employee/postgres_repository.go, mehrere *_db_test.go), auf den neuen Stand korrigiert statt
  stehen gelassen — die Joins ueber `users`/`roles` bleiben trotzdem drin, weil mehrere Pfade unter
  `sysctx.With()` laufen, wo RLS nicht greift (Doppelabsicherung, kein Widerspruch).
- Zwei Nebenbefunde beim Testfixen, beide Test-Artefakte, keine Produktionsluecken:
  (1) `roles_admin_db_test.go` und `user_roles_db_test.go` liessen denselben festen Akteur ueber
  ZWEI Tenants hinweg als Aufrufer agieren (nur moeglich, weil `user_roles` vorher RLS-frei war —
  real waere das nie erreichbar, ein JWT-Tenant stimmt immer mit der eigenen Zeile des Halters
  ueberein). Sechs betroffene Stellen bekommen jetzt einen tenant-eigenen Akteur
  (`roleAdminForeignActor`, `userRoleCustomOwner`) statt des geteilten.
  (2) `TestEffectivePermissions_DB_ForeignCustomRoleStaysInvisible` stand auf einer jetzt
  UEBERHOLTEN Annahme (Presets bleiben tenant-uebergreifend sichtbar, nur Custom-Rollen nicht) —
  echte Verschaerfung, keine Regression: `HandleGetUserPermissions` loest das Ziel schon vorher ueber
  `GetUser` (RLS-geschuetzt) auf, ein fremder User war ueber HTTP nie erreichbar. Erwartung im Test
  angepasst, der zugehoerige (jetzt falsche) Kommentar an der Route korrigiert.
  (3) Ein drittes Test-Artefakt in derselben Runde, kein Produktionscode: der eigene
  `rls_user_roles_test.go`-Test `TestAssignRole_DB_PresetOnlyDespiteNameCollision` mischte
  `defer pool.Close()` mit `t.Cleanup(...)` fuer Row-Aufraeumung — `t.Cleanup` laeuft NACH dem
  eigenen `defer` der Funktion, die Aufraeum-Querys liefen also gegen einen bereits geschlossenen
  Pool und hinterliessen eine Leiche, die den naechsten Testlauf mit einem Unique-Constraint-Fehler
  kollidieren liess. Auf durchgaengige `defer` umgestellt (mirrors `rls_refresh_tokens_test.go`).
- keine neue `config.RequireX`-Assertion, kein neues `modules.*`-Flag scharfgeschaltet, keine neue
  Route (kein `.proto` angefasst, `api/openapi.yaml` unveraendert).
- gate: `go build -p 2 ./...` + `go vet ./...` + `golangci-lint run ./...` -> 0 issues (voller
  Repo-Scope, weil die Query-Aenderung ausserhalb von `internal/auth` Bestandscode beruehrt hat:
  `internal/gateway/route_auth.go`, `internal/biz/hr/employee/postgres_repository.go`). Mit
  `DATABASE_URL` gegen `kmuhub_app` (lokaler Docker-Postgres, Migrationskopf 286):
  `go test -count=1 ./internal/auth/... ./internal/server/... ./internal/biz/hr/employee/...
  ./internal/gateway/...` -> alles PASS, 0 SKIP, inkl. 3 neuer RLS-Tests fuer `user_roles`.
  migrate up/down/up lokal gruen (Kopf 286).

## Iteration 48 — g-mails-templates — done — 2026-08-03

- verify-vorspann auf `6d6799b2` (Iteration 47, user_roles-RLS): `git show --stat` + Diff der
  Kern-Dateien (`postgres_repository.go`, `roles_admin.go`, `route_auth.go`) gelesen, deckt sich
  mit der Journal-Beschreibung. Migration additiv+Backfill, kein neuer Guard, keine neue Route.
  Kein Fund. Sauber.
- gebaut: `g-mails-templates`. Neues Package `internal/email/template/` (4-Datei-Muster: repository,
  postgres_repository, service, errors — mirrored von `signature/`). Migration 000287:
  `email_templates` mit `tenant_id NOT NULL`, `owner_id UUID REFERENCES users(id)` (NULL = shared),
  `visibility` CHECK IN ('personal','shared'), `CALL enable_tenant_rls('email_templates')`.
- Sichtbarkeits-Pattern ist EINE WHERE-Klausel, geteilt von `GetByID`/`ListVisible`:
  `tenant_id = $t AND (visibility='shared' OR owner_id=$user OR $isAdmin=true)` — mirrored von
  `crm/contact`s owner_id/visibility-Spalten, nicht neu erfunden (per Recherche als einziger
  echter Analog im Bestand identifiziert; `signature` ist rein user-scoped und taugte nicht als
  Vorlage). Eine fremde personal-Vorlage ist damit ueberall (Get/Update/Delete) unauffindbar statt
  403 — dieselbe "invisible not forbidden"-Konvention wie bei den RBAC-Presets aus Welle 1b.
  Update/Delete rufen GetByID zuerst; ein Fremdzugriff erreicht `repo.Delete` dadurch nie
  (testbewiesen: `DeleteFn` wird nicht aufgerufen, wenn GetByID `ErrTemplateNotFound` liefert).
- Placeholder-Substitution (`Service.Render`) iteriert NUR ueber die feste 6-Key-Liste
  `AllowedPlaceholders` (contact_first_name/_last_name/_email, company_name, sender_name, today)
  und schlaegt fuer jeden Key `{{key}}` im Body nach — ein vom Aufrufer gelieferter Key ausserhalb
  der Liste wird nie gelesen. Kein `text/template`, keine Reflection-basierte Feldaufloesung.
  Testbeweis: ein `unknown_key`-Wert mit Script-Payload bleibt im Output als literales
  `{{unknown_key}}` stehen statt ersetzt zu werden — genau die in den Notes verlangte
  Daten-Abfluss-Vermeidung.
- Keine neue Permission-Migration: recherchiert, dass JEDES email-Feature (signatures, rules,
  labels, accounts) dieselben drei `email:read/write/delete` teilt, keine Feature-eigenen Keys
  existieren. Templates reihen sich dort ein statt eine Ausnahme zu bauen — spart eine
  Migration und das Seed-Risiko komplett.
- Gateway: `user_id`/`is_admin` werden serverseitig aus `middleware.GetUserID`/`IsAdmin` aufgeloest,
  nie aus dem Client-Body uebernommen — dieselbe IDOR-Vermeidung wie `HandleListAccounts` aus
  Iteration 46. Die DTOs (`createEmailTemplateDTO` etc.) haben deshalb bewusst keine
  `user_id`/`is_admin`-Felder.
- Proto: 6 neue RPCs (List/Get/Create/Update/Delete/Render) + `EmailTemplateInfo`,
  `.pb.go`/`_grpc.pb.go` per direktem `protoc`-Aufruf regeneriert (kein `make proto-email`-Target
  im Makefile vorhanden, generischer `proto`-Target-Befehl 1:1 uebernommen).
  `EmailGRPCServer`-Struct/Konstruktor, `cmd/email/main.go`-Wiring und der bestehende
  `email_grpc_import_test.go` (11. Konstruktor-Argument) an die neue Service-Instanz angepasst.
- 6 Endpunkte in `route_email.go` unter `/api/v1/email/templates` + `/{id}/render`, alle unter dem
  bestehenden `email:read/write/delete`-Muster. `api/openapi.yaml`: 3 Pfad-Bloecke + 10 Schemas,
  `swagger-cli validate` -> "is valid", `TestOpenAPIRouteDrift` gruen (808 Routen / 810 Pfade).
- Test-Artefakt gefunden und noch vor dem Commit gefixt: die neuen DB-Testfunktionen mischten
  zunaechst `defer pool.Close()` mit `t.Cleanup(CleanupRow)` — derselbe Fund wie in Iteration 47
  bei `user_roles`. Cleanup lief gegen einen bereits geschlossenen Pool (nicht fatal, `t.Logf`
  statt `t.Fatalf`, aber Zeilen blieben in der DB stehen). Auf
  `t.Cleanup(func(){ pool.Close() })` umgestellt (LIFO: zuerst registriert, zuletzt aufgerufen).
- migrate up/down/up lokal gruen (Kopf 287). RLS per psql verifiziert: `relrowsecurity=t,
  relforcerowsecurity=t`, Policy `tenant_isolation`.
- keine neue `config.RequireX`-Assertion, kein neues `modules.*`-Flag scharfgeschaltet, kein neuer
  `RequirePermission`-Guard (bestehendes `email`-Trio wiederverwendet, kein Seed noetig).
- gate: `go build -p 2 ./...`, `go vet -p 2 ./...`, `golangci-lint run` auf
  `internal/email/template/...`, `internal/gateway/...`, `internal/server/...`,
  `internal/models/...`, `cmd/email/...` -> 0 issues. Mit `DATABASE_URL` gegen `kmuhub_app`
  (lokaler Docker-Postgres, Migrationskopf 287): `go test -count=1 ./internal/gateway/...
  ./internal/server/... ./internal/email/... ./internal/models/...` -> alles PASS, 0 SKIP
  (12 neue Tests in `internal/email/template`, davon 2 echte DB-Tests fuer Tenant-/Visibility-
  Isolation). Ein einmaliger Flake in `internal/gateway` (FAIL ohne erkennbaren Testnamen im
  Diff) verschwand beim zweimaligen Wiederholen — reproduzierbar gruen, kein Fund, keine
  Aenderung noetig.
- offen fuer Luke: keine FE-Anbindung in diesem Commit (Backend-Nachtloop-Scope) — recherchiert
  bestaetigt greenfield, kein FE-Client/Mock erwartete vorher eine bestimmte Wire-Shape.
- commit: `7d381016` (feat(mails): add email templates with placeholder substitution)

## Iteration 49 — csat-schema — done — 2026-08-05 23:10
- verify vorgaenger: `7d381016` (Iteration 48, email templates) geprueft — `git show --stat` plus
  gezielt `route_email.go` (Template-Handler), `000287_email_templates.up/down.sql`. Handler gehen
  ueber `e.getEmailClient()` -> `client.ListEmailTemplates/RenderEmailTemplate`, kein direkter
  Service-Zugriff; `user_id`/`is_admin` aus `middleware.GetUserID`/`IsAdmin`, nicht aus dem Body.
  Guards nutzen ausschliesslich das bestehende `email:read/write/delete`-Trio (kein neuer Key,
  kein Seed noetig). Tabelle hat `tenant_id UUID NOT NULL` + `CALL enable_tenant_rls`, down
  gefuellt. `.proto` und beide generierten Dateien im selben Commit. Kein Fund. Sauber.
- gebaut: Migration `000288_helpdesk_csat` (up+down). Tabelle `ticket_csat_responses`
  (id, tenant_id -> tenants ON DELETE CASCADE, ticket_id -> tickets ON DELETE CASCADE, rating,
  comment, submitted_at, token, token_expires_at, created_at, updated_at) plus die zwei
  denormalisierten Spalten `tickets.csat_rating` (SMALLINT, CHECK 1..5) und `tickets.csat_comment`.
  Reine Migration, kein Go-Code — Proto/Service ist A2, Route ist A3.
- ENTSCHEIDUNG rating NULLable (weicht vom Backlog-Wortlaut "rating SMALLINT CHECK 1..5" ab,
  begruendet): A6 legt die Zeile beim Ticket-Close mit Token an, BEVOR eine Bewertung existiert.
  Mit `rating NOT NULL` haette A6 entweder eine zweite Tabelle oder eine Pseudo-Bewertung
  gebraucht. Stattdessen `rating SMALLINT NULL CHECK (rating IS NULL OR rating BETWEEN 1 AND 5)`
  und der Kopplungs-CHECK `chk_ticket_csat_rating_submitted`:
  `(rating IS NULL) = (submitted_at IS NULL)`. Damit ist eine Zeile ohne Rating eindeutig eine
  offene Umfrage, `submitted_at IS NOT NULL` ist die autoritative "wurde bewertet"-Abfrage, und
  halb-gesetzte Zwischenzustaende laesst die DB gar nicht erst zu (per psql verifiziert: 23514).
- Zwei Unique-Indizes mit unterschiedlicher Reichweite, das ist Absicht:
  `uq_ticket_csat_responses_ticket (tenant_id, ticket_id)` ist tenant-gescopt (ein Ticket, eine
  Bewertung, Upsert-Ziel fuer A2); `uq_ticket_csat_responses_token (token) WHERE token IS NOT NULL`
  ist GLOBAL, weil der oeffentliche Endpunkt aus A8 keinen Tenant-Kontext hat und den Tenant erst
  AUS der Token-Zeile aufloest. Ein tenant-gescopter Token-Index waere dort nutzlos.
- Token-Spalten (`token`, `token_expires_at`) bewusst schon hier, damit A6 ohne zweite Migration
  auf dieselbe Tabelle auskommt (so im Backlog verlangt). Klartext, gleiche Begruendung wie
  `report_share_tokens` (000252) — der Token IST das Credential und wird nur per Gleichheit
  gesucht. Dispatch-Spalten (Versandzeitpunkt, Claim-Stempel) NICHT vorweggenommen: deren Form
  entscheidet A7, ein geratener Spaltenname waere teurer als eine Migration. Beide Punkte als
  Notiz an A2 und A6 im BACKLOG hinterlegt.
- migrate up (287 -> 288), down 1 (-> 287), up (-> 288) lokal gruen. RLS per psql:
  `relrowsecurity=true relforcerowsecurity=true`, Policy `tenant_isolation`.
- RLS-Smoke als `kmuhub_app` mit je einer geseedeten Zeile in zwei Tenants: eigener Tenant -> 1,
  fremder Tenant -> 0. Seed-Zeilen danach wieder geloescht.
- Constraint-Smoke per psql, alle vier Faelle wie erwartet: rating=6 -> CHECK-Verletzung;
  rating ohne submitted_at -> `chk_ticket_csat_rating_submitted`; zweite Zeile zum selben Ticket
  -> `uq_ticket_csat_responses_ticket`; Zeile mit Token und ohne Rating -> INSERT erfolgreich
  (das ist der A6-Fall); zweiter identischer Token in einem ANDEREN Tenant ->
  `uq_ticket_csat_responses_token`, also global eindeutig wie beabsichtigt.
- gate: build/vet/lint n.a. (kein Go-Code geaendert) | migration ok | rls-smoke ok |
  `go test -count=1 ./internal/testutil/... ./internal/helpdesk/...` mit gesetztem `DATABASE_URL`
  gegen `kmuhub_app` -> PASS, **0 SKIP, 66 gelaufene Tests**, darunter der Standing-Guard
  `TestAllPublicTablesHaveRLSOrAreAllowlisted` (gruen mit der neuen Tabelle, keine
  Allowlist-Ausnahme noetig).
- offen: Docker Desktop lief zu Iterationsbeginn nicht und wurde von dieser Iteration gestartet;
  lokaler Migrationskopf steht jetzt auf 288 (Prod-Kopf bleibt 287, kein Deploy aus dem Loop).
  Der Read-Pfad liefert `csat_rating`/`csat_comment` noch NICHT — die Spalten existieren, aber
  `postgres_repository.go` selektiert sie nicht und `postgres_repository.go:789` setzt
  `customer_satisfaction` weiterhin hart auf "–". Das ist A2 bzw. A4, kein Versaeumnis hier.
- commit: HEAD von backend-loop nach dieser Iteration, Subject "feat(helpdesk): add CSAT
  schema with survey token columns". SHA hier bewusst nicht eingetragen: die Zeile steht IN
  diesem Commit, jede Nachtragung per amend erzeugt eine neue SHA. `git log --oneline -1`
  bzw. Suche nach dem Subject findet ihn.

## Iteration 50 — csat-proto-service — done — 2026-08-05 23:40
- verify vorgaenger: `54305b08` (Iteration 49, CSAT-Schema) geprueft — `git show --stat` plus
  beide Migrations-Dateien vollstaendig gelesen. Tabelle hat `tenant_id UUID NOT NULL` +
  `CALL enable_tenant_rls`, `.down.sql` droppt Tabelle UND beide tickets-Spalten, kein Go-Code
  im Diff (also kein Proto-/Handler-Risiko). Die im Journal beschriebene NULLable-`rating`-
  Entscheidung samt `chk_ticket_csat_rating_submitted` steht so in der Datei. Kein Fund.
- gebaut: Proto + Service-Schicht fuer die Bewertung.
  - `.proto`: RPC `SubmitCsat(SubmitCsatRequest) returns (Ticket)`, Message
    `SubmitCsatRequest{ticket_id=1, rating=2 int32, optional comment=3}` — Tenant bewusst NICHT
    im Body, er kommt wie bei CloseTicket aus `middleware.GetTenantID(ctx)`. `Ticket` bekommt
    `optional int32 csat_rating = 24` und `optional string csat_comment = 25`. Beide
    generierten Dateien (`helpdesk.pb.go`, `helpdesk_grpc.pb.go`) im SELBEN Commit regeneriert
    (protoc direkt, `make` existiert auf dieser Maschine nicht — Kommando aus dem Makefile-
    Target `proto-helpdesk` uebernommen).
  - `Repository.SubmitCsatTx(ctx, tenantID, ticketID, rating int16, comment *string, submittedAt)`
    plus Postgres-Implementierung. EINE Transaktion: erst `UPDATE tickets SET csat_rating,
    csat_comment, updated_at WHERE id AND tenant_id`, dann Upsert in `ticket_csat_responses`
    mit `ON CONFLICT (tenant_id, ticket_id) DO UPDATE`.
  - `Service.SubmitCsat` validiert 1..5 serverseitig, laedt das Ticket (Tenant-Scope + 404) und
    ruft die Transaktion.
  - `ticketSelectColumns` um `t.csat_rating, t.csat_comment` erweitert — dadurch liefern
    GetTicketByID, ListTickets UND FindOpenTicketsByRequester die Werte, ohne dass drei Queries
    einzeln angefasst werden mussten. Beide Scan-Funktionen entsprechend erweitert.
- ENTSCHEIDUNG Reihenfolge in der Transaktion (Ticket-UPDATE VOR Response-Upsert): der
  Fremdschluessel `ticket_id -> tickets(id)` beweist nur, dass das Ticket existiert, NICHT dass
  es dem uebergebenen Tenant gehoert. Ein Insert mit fremdem `ticket_id` und eigenem `tenant_id`
  wuerde den FK passieren. Der zeilenlose `UPDATE tickets ... AND tenant_id` ist damit die
  eigentliche Tenant-Wache und muss zuerst laufen -> `ErrTicketNotFound`, nichts geschrieben.
  Gegen die echte DB verifiziert (Test unten, Fremd-Tenant-Fall).
- ENTSCHEIDUNG `DO UPDATE` setzt bewusst NUR rating/comment/submitted_at/updated_at und laesst
  `token`/`token_expires_at` in Ruhe: ein beim Ticket-Close ausgegebener Umfrage-Link (A6/A8)
  darf durch eine Agenten-seitige Bewertung nicht entwertet werden.
- ENTSCHEIDUNG `int16` statt `int` im Model (`Ticket.CsatRating *int16`): die Spalte ist
  SMALLINT, pgx scannt int2 verlustfrei in *int16. Die Verengung int32 -> int16 passiert im
  gRPC-Server und ist dort durch eine eigene Bereichspruefung abgesichert, damit ein
  Wire-Wert wie 65540 nicht in eine gueltige Bewertung wrappen kann. Die fachliche Pruefung
  bleibt zusaetzlich im Service (Handler duerfen nicht die einzige Wache sein).
- Kommentar wird im gRPC-Server getrimmt, ein leerer Kommentar wird zu NULL statt zu "" —
  sonst haette `csat_comment` zwei Bedeutungen fuer "kein Kommentar".
- `ErrInvalidCsatRating` neu + Mapping auf `codes.InvalidArgument` in `mapHelpdeskError`.
  Kein neuer Permission-Key, kein neues Flag, keine `config.RequireX`.
- gate (alle mit gesetztem `DATABASE_URL` gegen `kmuhub_app`):
  `go build ./...` ok | `go vet ./internal/helpdesk/... ./internal/server/...` ok |
  `golangci-lint run ./internal/helpdesk/... ./internal/server/...` -> **0 issues** |
  `go test -count=1 ./internal/helpdesk/... ./internal/server/... ./internal/gateway/...
  ./internal/testutil/...` -> PASS. Helpdesk verbose: **61 PASS, 0 SKIP, 0 FAIL**.
- Tests neu in `internal/helpdesk/csat_test.go`: vier Service-Tests gegen den Mock
  (Erfolg inkl. Nachweis, dass die Persistenz wirklich gerufen wurde; Rating 0/-1/6/127 ->
  `ErrInvalidCsatRating` UND null Repository-Aufrufe; zweite Bewertung gewinnt statt zu
  scheitern; fremder Tenant -> `ErrTicketNotFound` ohne Schreibversuch) und ein
  Repository-Test gegen die echte Datenbank (`TestSubmitCsatTx_UpsertsAndStaysInTenant`,
  0.07s, nicht geskippt): Fremd-Tenant-Kontext mit dem ECHTEN tenantID als Parameter wird von
  RLS gestoppt; eigener Kontext schreibt; `GetTicketByID` liefert Rating und Kommentar zurueck
  (das ist der Nachweis, dass der Read-Pfad die neuen Spalten wirklich mitbringt); zweite
  Bewertung erzeugt genau EINE Zeile mit dem neuen Wert; `submitted_at` ist gesetzt (der
  CHECK aus A1 haette einen halben Zustand sonst abgelehnt); fremder Tenant sieht 0 Zeilen.
  Eigene Tenants geseedet, nicht die geteilten TenantA/B.
- offen fuer die naechsten Iterationen (auch im BACKLOG bei A3/A4 notiert):
  - A3 (Route): 400 und 404 sind die real gelieferten Codes; die Ticket-JSON-Serialisierung
    des Gateways braucht `csat_rating`/`csat_comment`, sonst sieht das FE die eigene
    Bewertung nach dem POST nicht.
  - A4 (Stats): ueber `ticket_csat_responses` mit `WHERE submitted_at IS NOT NULL` aggregieren,
    nicht ueber `tickets.csat_rating` (nur Spiegel) — offene Umfrage-Zeilen haben
    `rating IS NULL` und duerfen weder zaehlen noch den Schnitt verfaelschen.
- kein FE-Teil in diesem Commit (Backend-Loop-Scope). Das FE-Flag `CSAT_FEATURE_ENABLED` bleibt
  false, bis A3 die Route liefert.
- commit: HEAD von backend-loop nach dieser Iteration, Subject "feat(helpdesk): submit and read
  customer satisfaction ratings".

## Iteration 51 — csat-route — done — 2026-08-05 23:58
- verify vorgaenger: `7bf12c18` (Iteration 50, csat-proto-service) gegen `git show --stat`
  und den vollen Diff geprueft (helpdesk_grpc.go, service.go, postgres_repository.go).
  Tenant-Guard-Reihenfolge (Ticket-UPDATE zuerst, FK beweist nur Existenz nicht Tenant) ist
  wie im Journal beschrieben im Code, `DO UPDATE` laesst token/token_expires_at unangetastet,
  Rating-Bereichspruefung existiert doppelt (gRPC-Server gegen int32->int16-Ueberlauf, Service
  fachlich). `go build ./...` lief zusaetzlich durch — gruen, kein Fund.
- gebaut: `POST /api/v1/helpdesk/tickets/{id}/csat` im Gateway.
  - `submitCsatRequest{rating int32 validate:"required,min=1,max=5", comment *string}` —
    Body-Form deckt sich mit `helpdesk-client.ts:134` (`{rating, comment}`).
  - `HandleSubmitCsat` folgt exakt dem Close/Reopen-Muster: `validateUUIDParam`,
    `decodeAndValidate`, Aufruf ueber `helpdeskClient.SubmitCsat` (kein direkt injizierter
    Service), `response.Proto` auf 200. Kein neuer Permission-Key: Route haengt an
    `hdTicketEdit` (bestehender additiver Guard `helpdesk:write` ODER `helpdesk:ticket:edit`) —
    Bewertung-Erfassung ist im Desktop-Client eine Ticket-Bearbeitung durch den Agenten, kein
    neuer fachlicher Bereich.
  - `openapi.yaml`: Pfad `/api/v1/helpdesk/tickets/{id}/csat` direkt vor `/assign` eingefuegt,
    Request-Schema (rating 1..5 required, comment optional), Responses 200/400/401/404 — alle
    vier sind die real gelieferten Codes (400 aus decodeAndValidate ODER dem gRPC-InvalidArgument-
    Mapping von A2, 404 aus dem gRPC-NotFound-Mapping bei fremdem/unbekanntem Ticket).
- gate: `go build ./...` ok | `go vet ./internal/helpdesk/... ./internal/gateway/...` ok |
  `golangci-lint run ./internal/gateway/...` -> 0 issues | mit
  `DATABASE_URL=postgres://kmuhub_app:...@localhost:5432/kmuhub` (kmuhub_app, nicht kmuhub):
  `go test -count=1 ./internal/helpdesk/... ./internal/gateway/...` -> beide PASS, inklusive
  `TestOpenAPIRouteDrift` (809 Routen gegen 811 dokumentierte Pfade) und
  `TestOpenAPISpecDrift` (810 dokumentierte Pfade gegen 809 registrierte — die eine bekannte
  Allowlist-Luecke `/api/v1/files/upload` bleibt unveraendert, kein neuer Drift).
- kein neuer Test in dieser Unit: die Rating-Grenzfaelle (0/-1/6/127) und der Fremd-Tenant-Fall
  sind bereits in `csat_test.go` (Iteration 50) auf Service-/Repository-Ebene abgedeckt: der
  Handler fuegt nur Parse/Validate/Dispatch hinzu und aendert daran nichts. `decodeAndValidate`
  selbst ist generisch und an anderer Stelle getestet.
- offen fuer die naechsten Iterationen: A4 (csat-stats-aggregation) haengt weiterhin am
  Literal in `postgres_repository.go:789`; die Route hier liest/schreibt nur `csat_rating`/
  `csat_comment` am Ticket, nicht die Stats-Aggregation.
- kein FE-Teil in diesem Commit (Backend-Loop-Scope). Das FE-Flag `CSAT_FEATURE_ENABLED`
  bleibt false — das ist eine FE-seitige Entscheidung, die Route ist jetzt real erreichbar.
- commit: HEAD von backend-loop nach dieser Iteration, Subject "feat(helpdesk): add ticket
  CSAT submission route".

## Iteration 52 — csat-stats-aggregation — done — 2026-08-06 00:15
- verify vorgaenger: `28efdc9e` (Iteration 51, csat-route) geprueft (`git show --stat` +
  vollstaendiger Diff route_helpdesk.go + openapi.yaml). `HandleSubmitCsat` geht ueber
  `client.SubmitCsat` (kein direkt injizierter Service), kein neuer Permission-Key
  (wiederverwendet `hdTicketEdit`), openapi.yaml dokumentiert 200/400/401/404 — alle vier
  sind die real gelieferten Codes. Sauber, kein Fund.
- gebaut: `GetHelpdeskStats` (postgres_repository.go) ersetzt das Literal
  `stats.CustomerSatisfaction = "–"` durch eine echte Aggregation ueber
  `ticket_csat_responses`: `AVG(rating) WHERE tenant_id = $1 AND submitted_at IS NOT NULL`,
  Format `"%.1f/5"` bzw. `"–"` bei NULL (keine Bewertungen). Pending-Survey-Zeilen
  (rating NULL, token gesetzt) werden ueber den `submitted_at`-Filter ausgeschlossen,
  nicht ueber `rating IS NOT NULL` — deckt sich mit der A2-Konvention.
- ABWEICHUNG vom Backlog-`done_when`: der bindende Wire-Vertrag (`WireHelpdeskStats` in
  `mocks/handlers/helpdesk.ts:115` UND `HelpdeskStats` in `helpdesk-types.ts:264`, konsumiert
  einzig in `HelpdeskPage.tsx:955` als ein StatCard-String) hat KEIN Verteilungs- oder
  Antwortzahl-Feld — nur `customer_satisfaction: string`. Die im Backlog-Scope beschriebene
  "Sterne-Verteilung (1..5 mit Anzahl)" existiert weder im MSW-Mock noch im FE-Typ noch in
  irgendeinem Konsumenten. Der Scope-Kopf sagt explizit "Dagegen pruefen, nicht raten" —
  also nur den echten Durchschnitt gebaut, keine spekulative Verteilungsstruktur ohne
  Abnehmer (Lean/YAGNI). Falls das FE spaeter eine Verteilung braucht, ist das eine neue
  Unit mit eigenem FE-Vertrag, kein Nachbau hier.
- gebaut (Test): `stats_test.go`, DB-Test `TestGetHelpdeskStats_CsatAverage` — drei
  Bewertungen (5,3,4) ergeben exakt 4.0/5 (rundungsfrei gewaehlt), eine Pending-Survey-Zeile
  (Token gesetzt, kein Rating) verfaelscht den Schnitt nicht, eine Bewertung in einem
  fremden Tenant leakt nicht in den eigenen Schnitt, ein Tenant ohne Bewertungen liefert
  definiert "–" statt NULL/Fehler.
- Fallstrick beim Bauen: `t.Cleanup` in einer Helper-Closure lief NACH dem `defer pool.Close()`
  der Testfunktion (t.Cleanup feuert nach allen Defers) — Rows blieben stehen, sichtbar an
  "cleanup ... closed pool"-Logzeilen. Auf einen einzigen `defer` am Ende der Testfunktion
  umgestellt (registriert nach `defer pool.Close()`, laeuft also per LIFO davor), wie in
  `csat_test.go` bereits vorgemacht. 5 verwaiste Test-Ticket-Zeilen aus dem fehlgeschlagenen
  ersten Versuch manuell in der lokalen DB bereinigt (`DELETE FROM tickets WHERE subject =
  'CSAT Stats Ticket'`), betrifft nur die lokale Dev-DB, keine Migration noetig.
- gate: `go build -p 2 ./internal/helpdesk/... ./internal/gateway/... ./cmd/gateway/...` ok |
  `go vet` ok | `golangci-lint run` -> 0 issues | mit
  `DATABASE_URL=postgres://kmuhub_app:...@localhost:5432/kmuhub` (kmuhub_app, nicht kmuhub):
  `go test -count=1 ./internal/helpdesk/... ./internal/gateway/...` -> beide PASS, 0 SKIP.
  Keine neue Tabelle/Policy in dieser Unit, RLS-Cross-Tenant-Isolation ist Teil des neuen
  Tests selbst (fremder Tenant liefert 0 Beitrag zum Schnitt) statt eines separaten Smoke.
- offen fuer naechste Iterationen: A5 (csat-tenant-config) und A6 (csat-survey-token) haengen
  weiterhin an csat-schema/-proto-service, beide `todo`. Falls spaeter eine Sterne-Verteilung
  im FE gebraucht wird, ist der Ansatzpunkt hier dokumentiert (SQL waere eine zweite Abfrage
  `GROUP BY rating` mit denselben Filtern), aber bewusst nicht vorgebaut.
- kein FE-Teil in diesem Commit (Backend-Loop-Scope).
- commit: HEAD von backend-loop nach dieser Iteration, Subject "feat(helpdesk): aggregate real
  CSAT average into helpdesk stats".

## Iteration 53 — csat-tenant-config — done — 2026-08-06 00:40
- verify vorgaenger: `edbc1cdf` (Iteration 52, csat-stats-aggregation) geprueft (`git show`
  + vollstaendiger Diff postgres_repository.go). Aggregation tenant-gescopt (`WHERE tenant_id
  = $1 AND submitted_at IS NOT NULL`), kein neuer Guard, kein neuer RPC/Route, kein
  gRPC-Layer-Umgehung. Sauber, kein Fund.
- gebaut: CSAT-Tenant-Konfiguration (an/aus, Verzoegerung in Minuten, Umfragetext) ueber die
  generische `tenant_settings`-Ablage, KEINE neue Tabelle — Muster wie `internal/settings.Branding`
  (module_id, ein JSON-Objekt als einzelner Entry), Zugriff ueber den bestehenden generischen
  `GetTenantSettings`/`PutTenantSettings`-RPC, kein neues dediziertes RPC-Paar.
  - `internal/helpdesk/csat_config.go` (neu): `CsatConfig{Enabled, SurveyDelayMinutes,
    SurveyQuestion}`, `DefaultCsatConfig()` (Enabled=true, 24h, deutscher Default-Fragetext —
    gespiegelt von `stores/helpdesk.ts` Initialzustand, damit der Server-Default zum
    FE-Store-Default passt, sobald das Panel spaeter verdrahtet wird), `ValidateCsatConfig`
    (Delay 0..20160 Minuten [14 Tage Deckel], Frage <=500 Zeichen). Reine Domain-Logik, kein
    gRPC-Client im Paket — folgt exakt der bestehenden Konvention aus `CreateTicketFromMessage`
    ("this method holds no cross-service client"; channel/subject/preview werden vom Aufrufer
    uebergeben). Zwei neue Sentinel-Errors in `errors.go`: `ErrInvalidCsatDelay`,
    `ErrCsatQuestionTooLong`.
  - `internal/server/helpdesk_grpc.go`: `HelpdeskGRPCServer` bekommt ein optionales
    `settingsClient settingsv1.SettingsServiceClient` (nil-sicher, exakt das gleiche Muster wie
    `inboxClient`). `GetCsatConfig`/`SetCsatConfig` als Go-Methoden (KEIN neues Proto-RPC — es
    gibt noch keinen FE-Vertrag, der eine Route bindet; die Konsumenten sind vorerst
    `csat-survey-token` (naechste Unit, Ticket-Close-Pfad) und spaeter ein Settings-Panel-Endpoint,
    falls das FE-Store `stores/helpdesk.ts` je an den Server gebunden wird). Get faellt bei
    fehlendem/unerreichbarem Settings-Client oder fehlender Zeile auf `DefaultCsatConfig()`
    zurueck (CSAT-Konfiguration ist ein Nebenaspekt, darf Ticket-Flows nie brechen). Set validiert
    zuerst (client-unabhaengig), dann schreibt es einen einzelnen Entry unter
    `helpdesk.CsatConfigEntryKey` als volles JSON-Objekt-Replace (kein Sparse-Merge — es gibt
    genau eine CSAT-Konfiguration pro Tenant, nicht eine Zeile pro Locale wie bei den
    Label-Overrides). RBAC (admin oder module-lead fuer "helpdesk_csat") wird bereits von
    `PutTenantSettings` im Settings-Service erzwungen, hier nicht dupliziert.
  - `cmd/helpdesk/main.go`: optionale `settingsv1.SettingsServiceClient`-Verbindung gegen
    `cfg.AuthGRPCAddress` (Settings ist bei auth co-located), 1:1 das gleiche
    Verbindungs-/Fallback-Muster wie der bestehende `inboxServiceClient` (verbindungsfehler ->
    Warn-Log, `NewHelpdeskGRPCServer` bekommt nil, Helpdesk startet trotzdem).
- ABWEICHUNG von der woertlichen Scope-Beschreibung: es gibt bewusst KEINE neue
  `/api/v1/helpdesk/csat-config`-Route und keinen openapi.yaml-Eintrag in dieser Unit. Das
  FE (`HelpdeskSettingsPanel.tsx`, `stores/helpdesk.ts`) haelt Enabled/Delay/Frage heute
  ausschliesslich in einem lokalen Zustand-Store, `helpdesk-client.ts` hat keinen einzigen
  Aufruf fuer diese Werte — es gibt keinen bindenden Wire-Vertrag, an dem sich eine Route
  ausrichten koennte (Lean/YAGNI: keine spekulative Route ohne Abnehmer). Der `done_when`-Text
  im Backlog verlangt explizit nur "Config liegt in tenant_settings" + "Defaults" + "Validierung",
  keine Route — anders als A3/A4/A8, deren `done_when` ausdruecklich `openapi.yaml` nennt.
  Falls/wenn das Panel ans Backend gebunden wird, ist das eine eigene Unit mit eigenem
  FE-Vertrag zum Pruefen, kein Nachbau hier.
- gebaut (Tests, alle rein, keine DB/kein gRPC noetig):
  - `internal/helpdesk/csat_config_test.go`: Default ist valide; Delay-Grenzen (0 und 20160
    valide, -1 und 20161 `ErrInvalidCsatDelay`); Frage bei 500 Zeichen valide, bei 501
    `ErrCsatQuestionTooLong`.
  - `internal/server/helpdesk_csat_config_test.go`: leere Entries -> Default; fremder Key wird
    ignoriert -> Default; voller Roundtrip `csatConfigToValue` -> `csatConfigFromEntries` liefert
    exakt den Ausgangswert zurueck; fehlerhafte Feldtypen (z. B. `enabled` als String statt Bool,
    `delay_minutes` fehlt) fallen einzeln auf den Default zurueck statt Nullwert oder Fehler,
    ein korrektes Feld (`question`) bleibt dabei erhalten.
- gate (alle mit gesetztem `DATABASE_URL` gegen `kmuhub_app`):
  `go build -p 2 ./internal/helpdesk/... ./internal/server/... ./internal/gateway/...
  ./cmd/helpdesk/... ./cmd/gateway/...` ok | `go vet ./internal/helpdesk/... ./internal/server/...
  ./cmd/helpdesk/...` ok | `golangci-lint run ./internal/helpdesk/... ./internal/server/...
  ./cmd/helpdesk/...` -> 0 issues | `go test -count=1 ./internal/helpdesk/...` -> PASS, 66
  PASS, 0 SKIP | `go test -count=1 ./internal/server/...` -> PASS | `go test -count=1
  ./internal/gateway/...` -> PASS (keine Route angefasst in dieser Unit, trotzdem mitgelaufen).
  Kein RLS-Smoke noetig: keine neue Tabelle, keine neue Policy — `tenant_settings` traegt
  bereits RLS, unveraendert.
- offen fuer die naechste Iteration: `csat-survey-token` (deps: csat-schema, csat-tenant-config
  — beide jetzt done) ist die naechste Unit im Block. Sie ruft `HelpdeskGRPCServer.GetCsatConfig`
  im Ticket-Close-Pfad auf, um zu entscheiden ob ein Umfrage-Token erzeugt wird; das ist der
  erste echte Aufrufer der hier gebauten Get/Set-Methoden.
- kein FE-Teil in diesem Commit (Backend-Loop-Scope).
- commit: HEAD von backend-loop nach dieser Iteration, Subject "feat(helpdesk): add tenant CSAT
  survey configuration".

## Iteration 54 — csat-survey-token — done — 2026-08-05 23:46 (lokale Uhr; die
Zeitstempel der Iterationen 50–53 liegen rund eine Stunde vor der Systemzeit)
- verify vorgaenger (913d540f, csat-tenant-config): sauber. Keine neue Route (also kein
  openapi-Drift moeglich), keine Migration, kein .proto, kein RequirePermission, keine neue
  Tabelle. Der Settings-Zugriff laeuft ueber `settingsv1.SettingsServiceClient`, also ueber den
  gRPC-Client und nicht an einer direkt injizierten Service-Instanz vorbei — keine
  Layer-Umgehung. Ein Doc-Detail stimmt nicht ganz: der Kommentar an `GetCsatConfig` sagt
  "defaults when the settings service is unreachable", der Code gibt bei einem RPC-Fehler
  aber den Fehler zurueck (nur `settingsClient == nil` faellt auf Default). Kein Fehlverhalten
  (der Aufrufer entscheidet), deshalb keine Fix-Unit — in dieser Iteration ist der Aufrufer
  gebaut und faengt den Fehler ab.
- gebaut (Code):
  - `internal/helpdesk/csat_survey.go` (neu): `CsatSurvey{TicketID, Token, ExpiresAt}`,
    `newCsatSurveyToken()` = 32 Byte `crypto/rand` + `base64.RawURLEncoding` (43 Zeichen,
    1:1 das Muster von `berichte.newShareSecret`, service.go:1132 — kein math/rand, keine UUID,
    nicht die Ticket-ID), `CsatSurveyTokenTTL = 30 Tage` und
    `Service.IssueCsatSurveyToken(ctx, *Ticket, CsatConfig) (*CsatSurvey, error)`.
    Ablauf = `now + cfg.SurveyDelayMinutes + TTL`: der Link soll erst nach dem geplanten
    Versand (A7) ablaufen, deshalb liegt die Verzoegerung VOR der Gueltigkeitsdauer. Ablauf
    steckt in `token_expires_at`, nicht im Link kodiert.
    Rueckgabe `(nil, nil)` fuer jeden legitimen Grund, nicht zu befragen: Tenant hat CSAT aus,
    oder das Ticket traegt schon eine Bewertung (close -> reopen -> close fragt nicht erneut).
    Kein Fehler, weil der Aufrufer daraus keine Konsequenz zieht.
  - `internal/helpdesk/repository.go` + `postgres_repository.go`: neue Interface-Methode
    `IssueCsatSurveyTokenTx(ctx, tenantID, ticketID, token, expiresAt, now) (bool, error)`.
    Das INSERT nimmt `tenant_id` NICHT vom Aufrufer, sondern `SELECT t.tenant_id FROM tickets t
    WHERE t.id = $2 AND t.tenant_id = $1` — ein fremdes Ticket liefert null Zeilen statt einer
    CSAT-Zeile unter dem falschen Tenant (gleiche Ueberlegung wie der Ticket-UPDATE-Guard in
    `SubmitCsatTx`, FK auf ticket_id beweist den Tenant nicht). Der Conflict-Zweig ist mit
    `WHERE ticket_csat_responses.submitted_at IS NULL` bewacht: das ist die autoritative
    "schon bewertet"-Pruefung, der Service-Check davor spart nur den Write im Normalfall und
    kann mit einer zwischenzeitlichen Abgabe rennen. `RowsAffected()==1` = ausgestellt.
    Ein bestehender *pending* Token wird bewusst ersetzt (neuer Close = neue Verzoegerung,
    der alte Link liefe auf einem veralteten Zeitplan) — der aeltere Link wird damit ungueltig.
  - `internal/server/helpdesk_grpc.go`: `CloseTicket` ruft nach erfolgreichem Close
    `issueCsatSurvey(ctx, tenantID, t)`. Diese Hilfsmethode holt die Tenant-Config ueber den
    Settings-Client (`GetCsatConfig`, aus der Vor-Iteration) und ruft `IssueCsatSurveyToken`.
    JEDER Fehler darin wird geloggt (`slog.WarnContext`) und verschluckt — der Close ist zu
    diesem Zeitpunkt bereits persistiert, ein Fehler wuerde einen erfolgreichen Close in eine
    fehlgeschlagene RPC verwandeln. Kein neuer RPC, keine neue Route: der Token wird nirgends
    ausgeliefert, er wartet auf A7 (Dispatch) und A8 (oeffentliche Einloesung).
  - KEINE Migration noetig (`token`, `token_expires_at` und der globale Partial-Unique-Index
    kamen mit 000288 aus A1) und keine angelegt.
- gebaut (Tests):
  - `internal/helpdesk/csat_survey_test.go` (Mock, rein): Token dekodiert zu genau 32 Byte
    base64url; Ablauf >= now+Delay+TTL; persistierter Token == zurueckgegebener; 16 Ausstellungen
    ergeben 16 verschiedene Tokens (Entropie, kein Determinismus); Tenant hat CSAT aus -> kein
    Token, kein Write, kein Fehler; Ticket schon bewertet -> dito; fremdes Ticket -> kein Token;
    Delay ausserhalb der Grenzen -> Fehler UND kein Write.
  - `internal/helpdesk/csat_survey_db_test.go` (echte DB als `kmuhub_app`, NOSUPERUSER
    NOBYPASSRLS, eigene frisch geseedete Tenants): Aufruf aus fremdem Ctx mit dem ECHTEN
    tenantID der Zeile stellt nichts aus (nur RLS kann das stoppen, nicht die WHERE-Klausel);
    eigener Ctx stellt aus; Cross-Tenant-Read auf `ticket_csat_responses` liefert 0 Zeilen
    (RLS-Smoke); zweiter Close ersetzt den pending Token; nach `SubmitCsatTx` stellt ein
    weiterer Aufruf NICHTS mehr aus und der vorhandene Token bleibt unveraendert.
  - `service_test.go`: `mockRepo` um `IssueCsatSurveyTokenTx` + `csatTokens`-Map erweitert
    (spiegelt das SQL-Verhalten: fremd/unbekannt und bereits bewertet -> false ohne Fehler).
- gate (alle mit `DATABASE_URL` gegen `kmuhub_app`):
  `go build -p 2 ./internal/helpdesk/... ./internal/server/... ./internal/gateway/...
  ./cmd/helpdesk/... ./cmd/gateway/...` ok | `go vet` (helpdesk, server, cmd/helpdesk) ok |
  `golangci-lint run` ueber dieselben Pakete -> 0 issues | `go test -count=1 ./internal/helpdesk/...`
  PASS, **73 PASS / 0 SKIP** (der neue DB-Test lief real, 0.06s, nicht uebersprungen) |
  `go test -count=1 ./internal/server/...` PASS | `go test -count=1 ./internal/gateway/` PASS
  (keine Route angefasst, TestOpenAPIRouteDrift trotzdem mitgelaufen).
  RLS-Smoke im DB-Test enthalten (siehe oben), keine neue Tabelle/Policy.
- offen fuer die naechste Iteration: A7 `csat-survey-dispatch` entscheidet die Dispatch-Spalten
  (Versandzeitpunkt + Claim) — die gibt es noch nicht, der Poller braucht eine eigene Migration.
  Beim Entwurf beachten: der hier gesetzte Ablauf ist bereits `Versandzeitpunkt + 30 Tage`, der
  Poller muss also nach `token IS NOT NULL AND submitted_at IS NULL AND <faellig>` filtern und
  nicht nach `token_expires_at`. A8 loest den Token oeffentlich ein und muss ihn dabei entwerten.
- kein FE-Teil in diesem Commit (Backend-Loop-Scope).
- commit: HEAD von backend-loop nach dieser Iteration, Subject "feat(helpdesk): issue CSAT survey
  token when a ticket closes".

## Iteration 55 — csat-survey-dispatch — done — 2026-08-06 01:10
- verify vorgaenger (606e94fb, csat-survey-token): **sauber**. Kein `.proto` im Diff (also kein
  Regen faellig), keine neue Route (kein openapi-Drift), keine neue Tabelle, kein
  `RequirePermission`, kein Stub/`Unimplemented`. Der einzige gRPC-nahe Punkt ist
  `internal/server/helpdesk_grpc.go` — das ist die gRPC-Server-Seite selbst, keine
  Layer-Umgehung im Gateway. Das INSERT in `IssueCsatSurveyTokenTx` nimmt `tenant_id` aus der
  Ticket-Zeile statt vom Aufrufer, Read-Seite unveraendert. Nichts anzulegen.
- gebaut (Schema): Migration **000289_helpdesk_csat_survey_dispatch** — drei Spalten an
  `ticket_csat_responses`: `survey_send_after` (Faelligkeit = Close + Tenant-Delay),
  `survey_sent_at` (ist GLEICHZEITIG der Claim) und `survey_dispatch_attempts SMALLINT NOT NULL
  DEFAULT 0` (Retry-Deckel 3). Partial-Index `idx_ticket_csat_responses_due` deckt exakt die
  Faelligkeits-Query. Backfill: Bestandszeilen mit Token bekommen `survey_send_after = created_at`
  (sofort faellig statt nie). down droppt Index + Spalten. Keine RLS-Arbeit noetig, die Policy
  kam mit 000288 — `TestAllPublicTablesHaveRLSOrAreAllowlisted` bleibt gruen (keine neue Tabelle).
  ENTSCHEIDUNG Faelligkeit: `survey_send_after` wird explizit gespeichert, NICHT aus
  `token_expires_at - TTL` zurueckgerechnet. Die Ableitung waere still falsch, sobald die TTL
  sich aendert.
- gebaut (Repo): `IssueCsatSurveyTokenTx` bekommt `sendAfter` und setzt beim Ersetzen eines
  pending Tokens `survey_sent_at = NULL, survey_dispatch_attempts = 0` zurueck (neuer Close =
  neuer Zeitplan; ohne das Reset waere ein zweiter Close nie zustellbar). Neu:
  `ListDueCsatSurveys` (JOIN tickets + users, liefert Empfaenger und Ticketnummer mit),
  `ClaimCsatSurveyDispatch` (Optimistic-UPDATE, `RowsAffected()==1` gewinnt),
  `ReleaseCsatSurveyDispatch` (gibt NUR den eigenen Claim zurueck, `WHERE survey_sent_at =
  claimedAt`), `CancelCsatSurvey` (entwertet Token, laesst ein vorhandenes Rating stehen).
  Diese vier haengen bewusst NICHT am fetten `Repository`-Interface, sondern an der neuen engen
  `CsatDispatchRepository` (csat_dispatch.go) — gleiche Trennung wie
  `berichte/scheduler.ScheduleRepository`, dadurch braucht `mockRepo` sie nicht.
- gebaut (Dispatcher): `internal/helpdesk/csat_dispatch.go`, `CsatSurveyDispatcher`.
  `Run` laeuft unter `database.WithSystemContext` (kein eingeloggter User, Query ist
  tenant-uebergreifend), Tick 5 min, Batch 100, erster Tick sofort. `ProcessTick`: listen →
  je Zeile Config → claim → mailen → bei Fehler Claim freigeben.
  **Die Falle, die hier fast zugeschnappt waere:** `GetCsatConfig` liest `tenant_settings` im
  Settings-Service unter RLS mit dem Tenant aus dem **Call-Kontext**, nicht mit der `tenant_id`
  im Request-Body. Aus dem tenantlosen System-Kontext des Pollers gerufen haette es 0 Zeilen
  gefunden und still `DefaultCsatConfig()` geliefert — die Tenant-Einstellung waere wirkungslos
  gewesen und niemand haette es gemerkt. Deshalb geht genau dieser eine Call ueber
  `withTenant(ctx, s.TenantID)` (1:1 `berichte.WithTenant`); alles andere bleibt System-Kontext.
  Tenant hat CSAT zwischenzeitlich abgeschaltet → `CancelCsatSurvey` statt Ueberspringen, sonst
  taucht die Zeile 30 Tage lang in jedem Tick wieder auf und der Link bliebe einloesbar.
  Config-Fehler → Zeile bleibt unangetastet und unclaimed (verbraucht keinen Versuch).
  Config-Lookup memoisiert pro Tenant pro Tick.
- gebaut (Mail): `csat_mailer.go`, `SystemMailCsatMailer` ueber `email/systemmail` (dieselbe
  Transaktions-SMTP-Strecke wie die geplanten Berichte). Neue Config `CSAT_SURVEY_BASE_URL`
  (default `https://app.zentria.tech/csat`) — **defaulted, keine `config.RequireX`-Assertion**,
  also keine Startgefahr fuer den laufenden Betrieb. Verdrahtung in `cmd/helpdesk/main.go` exakt
  nach dem Muster von `cmd/berichte`: ohne konfiguriertes System-SMTP wird der Dispatcher gar
  nicht erst gestartet (statt Umfragen zu claimen, die kein Transport zustellen kann) + Warn-Log.
  ABWEICHUNG vom Backlog-Text: Rendering laeuft NICHT ueber `email/template.Service.Render`.
  Das ist ein CRUD-Store fuer tenant-eigene Templates, `Render` schlaegt eine Template-Zeile per
  ID mit Sichtbarkeitsregeln nach — fuer eine System-Mail gibt es keine solche Zeile, und eine
  pro Tenant zu seeden waere eine eigene Baustelle. Stattdessen feste Body-Bauform mit der
  Tenant-Frage aus `CsatConfig`, 1:1 wie `berichte/scheduler.buildTextBody`. Kein text/template,
  keine freie Platzhalter-Aufloesung. HTML-Escaping via `html.EscapeString` (Stdlib).
- gebaut (Tests): `csat_dispatch_db_test.go` (echte DB als `kmuhub_app`) — die Faelligkeits-Query
  liefert fuer geseedete Daten **wirklich eine Zeile** (das ist der Punkt, nicht "laeuft
  fehlerfrei"; Lehre aus Iteration 45), eine noch nicht faellige Umfrage kommt nicht mit,
  Empfaenger/Name/Ticketnummer sind gefuellt, Cross-Tenant-Read der CSAT-Zeile = 0 Zeilen,
  zweiter Claim scheitert, Release macht faellig und verbraucht genau einen Versuch,
  erschoepfte Versuche fallen dauerhaft raus, Cancel entwertet den Token und laesst ein Rating
  stehen. `csat_dispatch_test.go` (Fakes) — zwei Ticks = eine Mail, nicht faellig = keine Mail,
  Sendefehler gibt den Claim frei und der naechste Tick stellt zu, abgeschalteter Tenant wird
  entwertet statt gemailt, Config-Ausfall verbraucht keinen Versuch, ein Config-Lookup pro
  Tenant pro Tick, Link/Frage/Subject im Body und Subject im HTML escaped.
- gate (alle mit `DATABASE_URL` gegen `kmuhub_app`, NOSUPERUSER NOBYPASSRLS):
  `migrate up` bis 289 ok | `go build -p 2` (helpdesk, server, gateway, cmd/helpdesk, cmd/gateway)
  ok | `go vet` (helpdesk, cmd/helpdesk) ok | `golangci-lint run` (helpdesk, cmd/helpdesk, config)
  → **0 issues** | `go test -count=1 ./internal/helpdesk/` **82 PASS / 0 SKIP / 0 FAIL** (die vier
  neuen DB-Tests liefen real) | `./internal/server/` ok | `./internal/gateway/` ok
  (TestOpenAPIRouteDrift mitgelaufen; keine Route angefasst, openapi.yaml unveraendert) |
  `./internal/testutil/` ok (RLS-Standing-Guard).
- offen fuer die naechste Iteration / fuer Luke:
  - **Externe Requester bekommen heute keine Umfrage.** `ListDueCsatSurveys` holt die Adresse
    ueber `JOIN users ON users.id = tickets.requester_id`; ohne User-Konto gibt es keine Zeile.
    Sobald B1 (`intake-ticket-columns`) `requester_email` an `tickets` bringt, muss der JOIN auf
    `COALESCE(t.requester_email, req.email)` erweitert werden — sonst bleibt genau die
    Kundengruppe stumm, fuer die CSAT gedacht ist.
  - A8 (`csat-public-response`) loest den Token oeffentlich ein und muss ihn dabei entwerten
    (`token = NULL`), sonst ist der Link mehrfach einloesbar. `CancelCsatSurvey` ist dafuer schon
    die passende Formulierung.
  - Ohne `SYSTEM_SMTP_HOST` laeuft der Dispatcher gar nicht — Umfragen sammeln sich als pending
    Tokens an und gehen raus, sobald SMTP konfiguriert ist (Nachhol-Effekt beim ersten Start
    bedenken). `CSAT_SURVEY_BASE_URL` muss auf die FE-Route zeigen, die A8 bedient.
- kein FE-Teil in diesem Commit (Backend-Loop-Scope).
- commit: HEAD von backend-loop nach dieser Iteration, Subject "feat(helpdesk): mail out CSAT
  surveys after their configured delay".

## Iteration 48 — csat-public-response — done — 2026-08-06 00:50

- commit: HEAD von backend-loop nach dieser Iteration, Subject "feat(helpdesk): redeem CSAT
  survey links without a login".
- verify vorgaenger (66253a22, csat-survey-dispatch): **sauber**. Keine Route, kein Proto,
  keine neue Tabelle (nur drei Spalten an `ticket_csat_responses`, RLS kam mit 000288);
  neue Config `CSAT_SURVEY_BASE_URL` ist **defaulted**, keine `config.RequireX`-Assertion,
  also kein Deploy-Hazard; kein `Unimplemented`/TODO/Fake im neuen Pfad; kein
  RequirePermission-Guard, kein Gateway-Handler, damit keine gRPC-Umgehung moeglich.
- gebaut (Proto): `SubmitCsatByToken(token, rating, optional comment) -> {ticket_number, rating}`.
  Antwort bewusst **kein** `Ticket`: der Aufrufer ist ein nicht angemeldeter Kunde und bekommt
  nur die Ticketnummer zurueck, die in seiner Einladungsmail ohnehin stand — kein Betreff,
  kein Bearbeiter, kein interner Zustand. `.pb.go` + `_grpc.pb.go` im selben Commit (protoc
  direkt aufgerufen, `make` gibt es in dieser Shell nicht).
- gebaut (Service): `Service.SubmitCsatByToken` in `csat_survey.go`, Schrittfolge 1:1 nach
  `berichte.Service.GetSharedDocument`: Token-Lookup unter `database.WithSystemContext` (die
  eine Zeile, die RLS verlassen MUSS, weil sie erst beantwortet, welcher Tenant gemeint ist) →
  Usable-Pruefung → `withTenant(ctx, survey.TenantID)` → ab da alles normal tenant-gescopt.
  Der System-Kontext wird nirgends weitergereicht.
  **Reihenfolge ist Absicht:** Rating-Range und Kommentarlaenge werden VOR dem Lookup geprueft.
  Ein ungueltiges Rating ist der Fehler des Aufrufers und darf nichts ueber den Token verraten;
  ein Test (`RejectsRatingBeforeTouchingToken`) haelt das fest, indem er zaehlt, dass der
  Lookup bei ungueltiger Eingabe **null Mal** laeuft.
- gebaut (Repo): `GetCsatSurveyByToken` (Equality auf die global eindeutige token-Spalte, nie
  eine Liste, nie ein Filter) und `RedeemCsatSurveyTx`. Die Transaktion setzt
  rating/comment/submitted_at, **entwertet den Token** (`token = NULL`, expires/send_after
  NULL) und spiegelt auf `tickets`. Der Guard `token IS NOT NULL AND submitted_at IS NULL`
  im UPDATE ist die autoritative Einmal-Pruefung — die Service-Pruefung davor ist gegen eine
  zeitgleiche zweite Einloesung rennbar, das SQL nicht. Null Zeilen → ErrCsatSurveyNotFound.
- gebaut (Gateway): `HelpdeskRoutes.RegisterPublicRoutes` →
  `POST /api/v1/public/helpdesk/csat/{token}`, registriert in `cmd/gateway/main.go` am
  **Root-Router ausserhalb der Registrar-Schleife**, einziges Middleware der strenge
  `publicRateLimiter` (`ratelimit:public`). Body-Deckel 8 KiB per `http.MaxBytesReader` **vor**
  dem Decode. Handler geht ueber `helpdeskClient.SubmitCsatByToken`, nicht ueber eine
  Service-Instanz. `openapi_drift_test.go` registriert die Route jetzt ebenfalls (sonst
  faellt sie durch die Drift-Pruefung).
- ein Verdikt fuer alle toten Links: unbekannt, missgebildet, ueberlang, leer, abgelaufen,
  entwertet und bereits eingeloest → **alle** `ErrCsatSurveyNotFound` → 404. Nie 403, nie eine
  unterscheidbare Meldung. 400 gibt es nur fuer Rating/Body, die nichts ueber den Token sagen.
- openapi.yaml: Pfad im selben Commit, dokumentiert 200/400/404/429/503.
- gate (alle mit `DATABASE_URL` gegen `kmuhub_app`, NOSUPERUSER NOBYPASSRLS):
  `go build -p 2` (helpdesk, server, gateway, cmd/gateway, cmd/helpdesk) ok | `go vet` ok |
  `golangci-lint run` (helpdesk, gateway, server) → **0 issues** |
  `go test -count=1 -v ./internal/helpdesk/` **85 PASS / 0 SKIP** (die drei neuen liefen real
  gegen die DB) | `./internal/gateway/` ok inkl. **TestOpenAPIRouteDrift PASS** |
  `./internal/server/` ok | `./internal/testutil/` ok (RLS-Standing-Guard).
  Keine Migration in dieser Unit — Spalten und Policy stammen aus 000288/000289.
- Testfalle, die hier zuschnappte und fuer kuenftige DB-Tests gilt: `t.Cleanup` laeuft **nach**
  dem `defer pool.Close()` der Testfunktion und findet dann einen geschlossenen Pool; die
  Zeilen bleiben liegen, und der global eindeutige Token-Index laesst den naechsten Lauf
  auflaufen. Aufraeumen per `defer`, und Test-Tokens mit einer Lauf-ID praefixen statt fester
  Literale.
- offen fuer die naechste Iteration / fuer Luke:
  - **Kein FE-Teil** (Loop-Scope). `CSAT_SURVEY_BASE_URL` (Default zeigt auf die
    `/csat`-Route der App) muss auf eine FE-Seite zeigen, die den Token aus dem Pfad nimmt
    und **POST**et — ein GET loest nichts ein. Die Seite bekommt `{ticket_number, rating}`
    zurueck und sollte 404 als "Link abgelaufen oder bereits genutzt" formulieren, ohne die
    Faelle zu unterscheiden.
  - Der Kommentar-Deckel (2000 Zeichen, `csatCommentMaxLength`) gilt vorerst nur auf dem
    oeffentlichen Weg; `submitCsatRequest` (Agenten-Route) hat weiterhin keinen. Bewusst nicht
    mitgeaendert, um den bestehenden Vertrag nicht nachts zu verengen.
  - Weiterhin offen aus Iteration 47: externe Requester bekommen keine Umfrage, weil
    `ListDueCsatSurveys` die Adresse ueber `JOIN users` holt. B1 bringt `requester_email` an
    `tickets`, dann muss der JOIN auf `COALESCE(t.requester_email, req.email)`.

## Iteration 49 — intake-ticket-columns — done — 2026-08-06 02:05

- commit: HEAD von backend-loop nach dieser Iteration, Subject "feat(helpdesk): give tickets
  their intake origin and extra fields".
- verify vorgaenger (18e58bf7, csat-public-response): **sauber**. `.proto` und beide `.pb.go`
  im selben Commit; openapi.yaml und `openapi_drift_test.go` mitgezogen; der Handler geht ueber
  `helpdeskClient.SubmitCsatByToken`, keine direkt injizierte Service-Instanz. Der System-Kontext
  ist auf **eine** Query eingesperrt (`postgres_repository.go:381`, Equality auf die eindeutige
  token-Spalte) und wird nirgends weitergereicht — `csat_survey.go:194` setzt unmittelbar danach
  `withTenant`. Kein `Unimplemented`, kein TODO, kein neuer RequirePermission-Guard, keine
  `config.RequireX`.
- gebaut (Migration 000290, `helpdesk_ticket_intake_fields`): fuenf Spalten an `tickets` —
  `channel TEXT NOT NULL DEFAULT 'agent'` mit eigenem CHECK auf ('agent','selfservice','external'),
  `requester_email TEXT NULL`, `requester_name TEXT NULL`,
  `requester_is_external BOOLEAN NOT NULL DEFAULT false`, `custom_fields JSONB NOT NULL DEFAULT '{}'`.
  Keine neue Tabelle, also keine RLS-Arbeit: `tickets` hat `tenant_id NOT NULL` und die
  `tenant_isolation`-Policy laengst.
- `source_channel` (000268) **nicht angefasst**, weder Spalte noch CHECK. In der DB verifiziert:
  `tickets_channel_check` und `tickets_source_channel_check` stehen als zwei getrennte
  Constraints nebeneinander. Die beiden beantworten verschiedene Fragen — source_channel sagt,
  aus welcher Inbox eine Nachricht konvertiert wurde (nur fuer Adapter-Tickets gesetzt),
  `channel` sagt, wie die Anfrage ueberhaupt in den Helpdesk kam (fuer jedes Ticket gesetzt).
- **Backfill-Entscheidung, bewusst konservativ:** Bestandszeilen nehmen die Spalten-Defaults
  ('agent', false, '{}'). `source_channel` wird NICHT auf `channel` gemappt. Ein aus einer Mail
  konvertiertes Ticket ist damit nicht als extern eingereicht belegt — ein Agent kann es genauso
  gut aus der Inbox geoeffnet haben. 'agent' heisst "im Modul angelegt" und ist fuer alle
  Altzeilen wahr; eine Ableitung aus source_channel wuerde Herkunft erfinden. Begruendung steht
  als Kommentar in der Migration.
- **VORRANGREGEL requester_name (die Entscheidung, die B1 verlangt):** der `users`-JOIN schlaegt
  die neue Spalte, die Spalte ist nur Fallback. `ticketSelectColumns` liest jetzt
  `COALESCE(JOIN-Name, req.email, NULLIF(t.requester_name,''), '')`. Grund: ein interner
  Requester ist ueber `requester_id` identifiziert, seine aktuelle Anzeige gehoert zur
  User-Zeile — eine persistierte Kopie veraltet beim ersten Rename und niemand merkt es. Die
  Spalte traegt genau die Requester, die keine User-Zeile zum Joinen haben. Das deckt sich mit
  dem FE-Typ, der es woertlich so beschreibt (`helpdesk-types.ts:37-39`). Regel steht als
  Doc-Kommentar an `ticketSelectColumns`, als Kommentar an der Spalte in der Migration, und —
  wichtiger — als Test.
- getestet (`ticket_requester_name_db_test.go`, real gegen die DB als `kmuhub_app`):
  `TestTicketRead_RequesterNamePrecedence` legt beide Faelle nebeneinander und gibt dem internen
  Ticket **absichtlich** einen widersprechenden Spaltenwert ("Falscher Name") — der Test kann
  also nur gruen werden, wenn der JOIN wirklich gewinnt. Der externe Fall (requester_id ohne
  User-Zeile, LEFT JOIN liefert NULL) faellt ohne den neuen COALESCE-Zweig auf `""` zurueck und
  waere rot. `TestTicketRead_IntakeColumnDefaults` haelt fest, dass ein ohne Intake-Felder
  angelegtes Ticket mit 'agent'/false/'{}' aus der DB kommt statt mit NULLs.
- gate (`DATABASE_URL` gesetzt, Rolle `kmuhub_app`, NOSUPERUSER NOBYPASSRLS):
  `migrate up` 289 -> 290 ok, `down 1` -> 289 ok, `up` -> 290 ok (Roundtrip, down ist wirklich
  gefuellt) | `go build -p 2 ./...` ok | `go vet` ok | `golangci-lint run` (helpdesk, gateway)
  **0 issues** | `go test -count=1 -v ./internal/helpdesk/ ./internal/testutil/ ./internal/gateway/`
  **732 PASS / 0 FAIL / 0 SKIP** — inklusive `TestAllPublicTablesHaveRLSOrAreAllowlisted` und
  `TestOpenAPIRouteDrift`. Null Skips heisst: die DB-Tests liefen wirklich.
- **keine openapi.yaml-Aenderung, und das ist Absicht:** diese Unit legt Spalten an, sie bringt
  noch kein Feld auf den Wire. Die Ticket-Response aendert sich erst mit B2/B3, dann gehoert der
  Schema-Nachtrag in denselben Commit.
- offen fuer die naechste Iteration:
  - B2 muss `requester_name` fuer **interne** Requester gar nicht erst schreiben (sonst baut man
    die veraltende Kopie, die die Vorrangregel gerade vermeidet). Schreiben nur, wenn
    `requester_is_external`.
  - Die fuenf Spalten sind noch in keinem SELECT ausser `requester_name` — `ticketSelectColumns`
    und `scanTicket` muessen in B2 gemeinsam wachsen, sonst laeuft die Scan-Reihenfolge auseinander.
  - Aus Iteration 48 uebernommen und jetzt entsperrt: `ListDueCsatSurveys` holt die Adresse ueber
    `JOIN users` und uebergeht damit externe Requester. `tickets.requester_email` existiert ab
    jetzt, der JOIN kann auf `COALESCE(t.requester_email, req.email)` — gehoert sinnvoll zu B5.

## Iteration 50 — intake-proto-create (B2)

- commit: HEAD von backend-loop nach dieser Iteration, Subject "feat(helpdesk): carry intake
  origin and extra fields through create".
- verify vorgaenger (ca6b516a, intake-ticket-columns): **sauber**. Reine Migration + ein
  COALESCE-Zweig in `ticketSelectColumns` + ein DB-Test. Kein `.proto` beruehrt, also auch kein
  fehlendes Regen; kein `Unimplemented`, kein TODO, kein neuer RequirePermission-Guard, keine
  `config.RequireX`, kein Direct-Svc-Aufruf. Die dort behauptete Vorrangregel ist im Code und im
  Test wirklich so verdrahtet, wie das Journal es beschreibt — nachgelesen, nicht geglaubt.
- gebaut: die fuenf Intake-Felder gehen jetzt durch Proto, Service und Repository und kommen beim
  Read zurueck. `CreateTicketRequest` +5 Felder (11–15), `Ticket` +4 (26–29; `requester_name`
  gab es als Feld 16 laengst). `helpdesk.pb.go` im selben Commit regeneriert;
  `helpdesk_grpc.pb.go` ist unveraendert und das ist korrekt — keine RPC-Signatur hat sich
  geaendert, protoc erzeugt dieselbe Datei.
- **custom_fields ist `google.protobuf.Struct`, nicht `map<string,string>`.** Der FE-Typ ist
  `Record<string, string | number | boolean>` (helpdesk-types.ts:49) — eine String-Map haette
  jede Zahl und jedes Boolean in seine Textdarstellung gedrueckt, und beim Zurueckschreiben
  waere aus `4711` das Wort "4711" geworden. `structpb` ist im Repo bereits im Einsatz
  (automation, inbox, settings, und helpdesk_grpc.go importiert es schon fuer die CSAT-Config),
  also keine neue Abhaengigkeit. Der Round-Trip-Test prueft ausdruecklich, dass die Zahl als
  float64 zurueckkommt und nicht als String.
- **Signatur-Entscheidung:** `Service.CreateTicket` hatte bereits 11 Positionsparameter. Fuenf
  weitere haetten 16 ergeben — eine Aufruf-Zeile, in der `nil, nil, "", "", nil, nil` steht und
  niemand mehr sieht, welches nil was ist. Stattdessen ein trailing `TicketIntake`-Wert
  (`ticket_intake.go`). Die fuenf Felder gehoeren fachlich zusammen (sie beschreiben die
  Herkunft), und der Diff an den ~28 bestehenden Testaufrufen ist mechanisch `, TicketIntake{})`.
  Die 11 Altparameter blieben absichtlich unangetastet: ein Params-Struct-Refactor haette
  denselben Nutzen zu 30 handgeschriebenen Aufrufumbauten gehabt und gehoert nicht in dieselbe
  Unit wie ein Vertragsbruch-Fix.
- **Validierung sitzt in `TicketIntake.normalize()`, nicht im Handler** — eine Stelle fuer den
  gRPC-Handler, den spaeteren Formular-Dispatch (B7) und den oeffentlichen Intake (B8):
  - channel leer -> "agent", channel unbekannt -> `ErrInvalidChannel`. **Kein stiller Default
    fuer einen unbekannten Wert** — das haette den Datenverlust nur eine Schicht tiefer gelegt.
  - requester_email ueber `net/mail.ParseAddress` plus 320-Zeichen-Deckel (Stdlib, keine Regex,
    keine Dependency).
  - requester_name wird fuer INTERNE Requester **verworfen statt gespeichert**. Genau das
    verlangt die Vorrangregel aus B1: der users-JOIN gewinnt beim Lesen, eine persistierte Kopie
    wuerde beim ersten Rename veralten. Die Regel wird hier durchgesetzt, nicht nur gelesen.
  - custom_fields: nur Skalare (string | number | bool). Verschachtelte Objekte, Arrays, null und
    leere Keys -> `ErrInvalidCustomFields`. Deckel bei 100 Keys, 128 Zeichen Key, 4096 Zeichen
    Wert — die oeffentlichen Intake-Pfade schreiben spaeter durch dieselbe Funktion, und ohne
    Deckel waere das eine unbegrenzt wachsende Zeile pro Einreichung.
  Alle vier Sentinels sind in `mapHelpdeskError` auf `InvalidArgument` verdrahtet. Das war fast
  die Luecke dieser Iteration: ohne den Eintrag faellt ein sauber validierter Tippfehler durch
  den default-Zweig auf `Internal` und das Modul zeigt "Serverfehler" fuer eine 400.
  `helpdesk_intake_error_test.go` belegt die Abbildung bare und wrapped.
- **`scanTicket` und `scanTicketFromRows` waren zwei Kopien derselben Ziel-Liste** und der
  Kommentar an `ticketSelectColumns` warnte genau davor, dass sie auseinanderlaufen. Sie gehen
  jetzt beide durch `ticketScanDest()`; eine Spalte kann nicht mehr in nur einer der beiden
  landen und alle Folgefelder verschieben.
- **`CreateTicketFromMessage` setzt channel bewusst auf 'agent'**, nicht auf den source_channel.
  Wie eine Nachricht in die INBOX kam, ist eine andere Frage als wie die Anfrage in den Helpdesk
  kam — dieselbe Unterscheidung, die schon der 000290-Backfill trifft. `TestCreateTicketFromMessage_SetsAgentChannel`
  haelt beide Spalten nebeneinander fest, damit sie niemand zusammenlegt.
- **Repo-Guard:** `channel` ist NOT NULL mit CHECK, ein leerer Wert waere eine 23514 mitten in
  einer Nutzeraktion. Das Repository setzt daher 'agent', wenn ein interner Aufrufer ohne
  `normalize()` einen Ticket-Struct baut. Das ist KEIN stilles Defaulten eines Nutzerwerts —
  unbekannte Werte werden an der Trust-Boundary abgelehnt, hier geht es nur um einen Aufrufer,
  der das Feld gar nicht kennt.
- gate (`DATABASE_URL` gesetzt, Rolle `kmuhub_app`, NOSUPERUSER NOBYPASSRLS): `go build ./...` ok
  | `go vet ./...` ok | `golangci-lint run` (helpdesk, server, gateway) **0 issues** |
  `go test -count=1 -v ./internal/helpdesk/ ./internal/server/ ./internal/gateway/ ./internal/testutil/`
  **958 PASS / 0 FAIL / 0 SKIP** — inklusive `TestAllPublicTablesHaveRLSOrAreAllowlisted` und
  `TestOpenAPIRouteDrift`. Null Skips heisst: die DB-Tests liefen wirklich.
- **keine openapi.yaml-Aenderung, geprueft statt angenommen:** diese Unit legt keine Route an,
  und die Ticket-Antworten sind in der Spec als `schema: { type: object }` beschrieben
  (openapi.yaml:13894, :13920) — es gibt kein Ticket-Schema, das nachzuziehen waere. Der
  Request-Body-Block bei `post /api/v1/helpdesk/tickets` (:13905) listet die Felder dagegen
  einzeln auf; **dort gehoeren die fuenf neuen Felder in B3 hinein**, zusammen mit dem DTO.
- offen fuer die naechste Iteration:
  - B3 (intake-route-create) ist jetzt reine Gateway-Arbeit: DTO um die fuenf Felder erweitern,
    `custom_fields` als `map[string]any` dekodieren und ueber `structpb.NewStruct` in die
    gRPC-Request heben (Fehler dort -> 400), requester_id weiter aus der Session. Validierung
    NICHT doppeln, die liegt im Service.
  - `UpdateTicket` im Repository schreibt die Intake-Spalten nicht mit — fuer B4 heisst das:
    custom_fields-Merge im Service bauen, nicht die SET-Liste aufblaehen.
  - Weiter offen aus Iteration 48/49: `ListDueCsatSurveys` holt die Adresse ueber `JOIN users`
    und uebergeht externe Requester. `tickets.requester_email` ist jetzt nicht nur da, sondern
    auch befuellt — der JOIN kann auf `COALESCE(t.requester_email, req.email)`. Gehoert zu B5.

## Iteration 51 — intake-route-create (B3)

- commit: f4a6cd31, "feat(helpdesk): carry intake origin through the create route".
- verify vorgaenger (f4a6cd31 selbst geprueft nach dem Bauen, siehe unten; das davor liegende
  eac91ac9/intake-proto-create war schon in Iteration 50 verifiziert). Kein neuer Vorgaenger-Fund.
- gebaut: `createTicketRequest` im Gateway um `channel`, `requester_email`, `requester_name`,
  `requester_is_external`, `custom_fields` erweitert und in `grpcReq` durchgereicht.
  `custom_fields` kommt als `map[string]any` aus `decodeAndValidate`, wird ueber
  `structpb.NewStruct` in ein `*structpb.Struct` gehoben — schlaegt das fehl, 400
  "invalid custom_fields", kein 500 (Vorgabe aus den Notes, nicht wie
  `route_automation.go`/`route_inbox.go`, die den Fehler heute still schlucken; hier bewusst
  NICHT diesem Muster gefolgt, weil die Notes explizit 400 statt 500 verlangen).
  `openapi.yaml` im selben Commit: die fuenf Felder unter `post /api/v1/helpdesk/tickets`
  ergaenzt (Zeile ~13910). `description`/`category` fehlten dort schon vorher im Request-Body-
  Schema — vorbestehende Luecke, nicht in dieser Unit angefasst (out of scope).
- **Keine Validierung im Handler dupliziert** — kein `validate`-Tag auf den vier neuen
  String-/Struct-Feldern, die Regeln (unbekannter channel, kaputte Mail, verschachtelte
  custom_fields) laufen wie vorgeschrieben ausschliesslich durch `TicketIntake.normalize` im
  Service (B2). `requester_is_external` hat kein `omitempty`-Validate noetig, ist ein Plain-Bool.
- **requester_id bleibt serverseitig aus der Session** (`middleware.GetUserID`), nicht aus dem
  Body — im DTO gibt es dafuer gar kein Feld, IDOR-Linie aus Iteration 46 unveraendert weiter
  gezogen.
- **FE-Vertrag abgeglichen, nicht geraten:** `helpdesk-types.ts` (`CreateTicketInput`) und
  `helpdesk-ticket-target.ts` (der Intake-Engine-Zielpunkt, der `createTicket()` schon mit
  `channel`/`requester_id`/`requester_name`/`requester_email`/`requester_is_external`/
  `custom_fields` aufruft) nutzen exakt diese fuenf Feldnamen und `TicketChannel =
  'agent'|'selfservice'|'external'`, `custom_fields?: Record<string, string|number|boolean>` —
  passt 1:1 auf `map[string]any` + `structpb.NewStruct`.
- **"Round-Trip-Test" im Gateway-Package ist strukturell nicht als Live-Call moeglich:** dieses
  Package hat keine bufconn/echten-Server-Testinfrastruktur (grep bestaetigt: kein `bufconn` im
  ganzen Repo), `registryWithService()` verbindet immer auf eine Dummy-Adresse
  (`localhost:0`), jeder echte RPC-Call schlaegt mit `Unavailable` fehl → 503. Der tatsaechliche
  Proto→Service→Repository→Read-Round-Trip ist bereits in B2 (Iteration 50, `internal/helpdesk`)
  getestet. Fuer B3 stattdessen zwei Tests, die die Gateway-eigene Verantwortung pruefen:
  `TestHandleCreateTicket_IntakeFieldsPassValidation` (alle fuenf Felder gesetzt, Subject dazu →
  503 statt 400 beweist, dass decode+validate die neuen Felder NICHT ablehnt, bevor sie den
  RPC-Call erreichen) und `TestHandleCreateTicket_CustomFieldsNotObject` (custom_fields als
  JSON-Array → 400 "invalid request body", weil `json.Decode` in `map[string]any` beim Typ
  scheitert, bevor `structpb.NewStruct` ueberhaupt aufgerufen wird).
- gate (`DATABASE_URL` gesetzt, Rolle `kmuhub_app`): `go build ./...` ok | `go vet ./...` ok |
  `golangci-lint run ./internal/gateway/...` **0 issues** |
  `go test -count=1 ./internal/helpdesk/... ./internal/server/... ./internal/gateway/... ./internal/testutil/...`
  **grün** (inkl. `TestOpenAPIRouteDrift`, `TestOpenAPISpecDrift`,
  `TestAllPublicTablesHaveRLSOrAreAllowlisted`). Ein Lauf zuvor zeigte 1 FAIL
  (`TestDecodeBexioState_ManipulatedSignature`, `internal/gateway/bexio_state_test.go` — flippt
  einen Base64-Zeichen der Signatur und erwartet einen Fehler; ~1/64 Chance, dass das getroffene
  Zeichen zufaellig gleich bleibt). 20 isolierte Wiederholungen liefen sauber durch, ein zweiter
  Voll-Lauf war gruen — bestaetigt als vorbestehender Flake in einem Bexio-Test, unberuehrt von
  dieser Unit, nicht behoben (auesserhalb des Scopes B3).
- offen fuer die naechste Iteration:
  - B4 (intake-route-update) ist die naechste in der deps-Kette: `updateTicketRequest` um
    `status` und `custom_fields` erweitern (HelpdeskPage.tsx:515 sendet beides, beides geht
    heute verloren). `UpdateTicket` im Repository schreibt die Intake-Spalten noch nicht mit —
    custom_fields-Merge im Service bauen, nicht die SET-Liste aufblaehen (Hinweis aus B2/B3-Notes
    unveraendert gueltig).
  - Der vorbestehende Bexio-Flake (`TestDecodeBexioState_ManipulatedSignature`) ist keine Backlog-
    Unit — falls er wieder auftaucht und stoert, waere ein deterministischer Test (festes
    manipuliertes Byte statt "X" an fixer Position, oder pruefen, ob sich das Zeichen tatsaechlich
    geaendert hat) der saubere Fix, aber das ist ausserhalb von Helpdesk/Intake.

## Iteration 52 — intake-route-update (B4)

- commit: 4ada610d, "feat(helpdesk): persist status and custom_fields on ticket update".
- verify vorgaenger: f4a6cd31/intake-route-create (Iteration 51) selbst geprueft nach dem Bauen,
  kein neuer Vorgaenger-Fund. Kein Commit lag zwischen Iteration 51 und dieser Iteration.
- gebaut: `UpdateTicketRequest` im Proto um `optional string status = 8` und
  `google.protobuf.Struct custom_fields = 9` erweitert, `make proto-helpdesk`-Aequivalent
  (protoc direkt, `make` fehlt in der bash) im selben Commit regeneriert. `Service.UpdateTicket`
  nimmt jetzt `statusVal *string` und `customFields map[string]any` zusaetzlich zu den
  bestehenden Parametern. `PostgresRepository.UpdateTicket` schreibt `custom_fields` jetzt mit
  (vorher fehlte die Spalte komplett in der SET-Liste — ein Read-Modify-Write ueber den Service
  haette sie also schon vorher nie persistiert, unabhaengig vom Gateway-Fix). Gateway-DTO um
  `Status *string` und `CustomFields map[string]any` erweitert, `custom_fields` ueber
  `structpb.NewStruct` in die gRPC-Request gehoben — schlaegt das fehl, 400 "invalid
  custom_fields", exakt das Muster aus B3 (Iteration 51). `openapi.yaml` im selben Commit.
- **custom_fields ist ein MERGE-Patch, kein Replace** — Vorgabe aus den B4-Notes, Vorlage ist
  der MSW-Handler (`handlers/helpdesk.ts:754`): `ticket.custom_fields = { ...alt, ...patch }`.
  `Service.UpdateTicket` laedt das bestehende `t.CustomFields` (kommt aus `GetTicketByID` ->
  `applyCustomFields`, nie nil), validiert den eingehenden Patch ueber das bestehende
  `normalizeCustomFields` (dieselbe Funktion wie beim Create-Intake, B2) und mergt mit
  `maps.Copy` in eine Kopie. Zwei aufeinanderfolgende Updates mit je einem Key loeschen sich
  nicht gegenseitig (Test `TestUpdateTicket_CustomFieldsMergeAcrossUpdates`).
- **status: geprueft, aber NICHT alle fuenf Werte durchgelassen.** Die Notes verlangten
  "erlaubte Werte UND Uebergaenge serverseitig pruefen" und hielten fest, dass close/reopen
  ihre eigenen Endpunkte bleiben und dieses Feld "die uebrigen Uebergaenge" abdeckt. Entscheidung:
  `status=closed` und `status=merged` werden ueber den generischen Pfad mit `ErrInvalidStatus`
  abgelehnt (400), weil `CloseTicket` zusaetzlich `resolved_at` setzt und die CSAT-Umfrage
  ausloest (`issueCsatSurvey`), und `MergeTickets` Nachrichten umhaengt — ein direktes
  `status=closed` ueber PUT haette den Ticket-Status ohne diese Nebenwirkungen umgestellt und
  genau die Art von stillem Datendrift erzeugt, die dieser ganze Block beheben soll. `open`,
  `pending`, `solved` bleiben ueber das Feld erreichbar, unabhaengig vom aktuellen Ist-Status
  (kein State-Machine-Umbau, das FE routet `closed` und `closed/resolved -> open` ohnehin nicht
  hierher, siehe HelpdeskPage.tsx:497-514). `openapi.yaml`-Enum entsprechend auf
  `[open, pending, solved]` verengt statt aller fuenf Werte, mit Begruendung im
  `description`-Feld. Test `TestUpdateTicket_RejectsClosedAndMergedStatus` deckt beide
  Ablehnungen ab.
- **DisallowUnknownFields NICHT eingefuehrt.** `decodeAndValidate[T]` ist ein einziger generischer
  Helper fuer JEDES DTO im gesamten Gateway-Package (Grep bestaetigt: keine Pro-Typ-Overrides).
  Eine Option waere gewesen, es global anzuschalten, um den naechsten stillen Feldverlust laut zu
  machen — aber das ist eine Verhaltensaenderung fuer saemtliche bestehenden Routen auf einmal,
  nicht etwas das man nebenbei in einer einzelnen Helpdesk-Unit entscheidet. Bleibt aus, wie in
  den Notes als Option vorgesehen ("falls es bestehende Clients brechen wuerde, ebenfalls
  begruenden und lassen") — hier ist die Sorge nicht Client-Bruch, sondern Blast-Radius weit
  ausserhalb des Scopes. Falls das noch gewuenscht ist: eigene Unit mit eigenem Audit ueber alle
  DTOs, nicht Beifang hier.
- **`category` bleibt ein bekannter, unberuehrter Datenverlust.** `UpdateTicketInput.category`
  existiert im FE-Typ (`helpdesk-types.ts:136`), aber weder `updateTicketRequest` im Gateway noch
  `UpdateTicketRequest` im Proto noch `Service.UpdateTicket` kennen das Feld — derselbe Fehler-
  Typ wie status/custom_fields vor dieser Unit, aber ausserhalb des B4-Scopes (`done_when` nennt
  nur status und custom_fields) und nicht in dieser Iteration angefasst. Naechste passende
  Backlog-Unit oder Ergaenzung von B4, falls gewuenscht.
- vier bestehende Aufrufer von `Service.UpdateTicket` (zwei in `contact_org_link_test.go`, zwei in
  `service_test.go`) an die neue Signatur angepasst (zwei neue Parameter, `nil` an den passenden
  Stellen) — reine Positionsanpassung, keine Verhaltensaenderung an diesen Tests.
- gate (`DATABASE_URL` gesetzt, Rolle `kmuhub_app`): `go build -p 2 ./internal/helpdesk/...
  ./internal/gateway/... ./internal/server/...` ok | `go vet` ok | `golangci-lint run` (helpdesk,
  gateway, server) **0 issues** | `go test -count=1 -v ./internal/helpdesk/...
  ./internal/gateway/... ./internal/server/... ./internal/testutil/...` **970 PASS / 0 FAIL / 0
  SKIP** (inkl. `TestOpenAPIRouteDrift`, `TestOpenAPISpecDrift`,
  `TestAllPublicTablesHaveRLSOrAreAllowlisted`) — der Bexio-Flake aus Iteration 51
  (`TestDecodeBexioState_ManipulatedSignature`) trat in diesem Lauf nicht auf, unberuehrt von
  dieser Unit. `go build -p 2 ./...` zusaetzlich fuer den Gesamt-Build geprueft, gruen.
- offen fuer die naechste Iteration:
  - `intake-external-requester` (B5) ist laut deps-Kette als naechstes dran: `requester_id`
    NULLable machen, CHECK fuer intern-oder-extern-mit-Mail, `scope=own`-Verhalten fuer Externe
    entscheiden und dokumentieren. Aus Iteration 50 weiterhin gueltig: `ListDueCsatSurveys` holt
    die Adresse ueber `JOIN users` und uebergeht externe Requester — mit befuelltem
    `tickets.requester_email` (seit B1) kann der JOIN auf
    `COALESCE(t.requester_email, req.email)` wechseln, das gehoert in B5 mit hinein wenn dort der
    externe Requester ohnehin angefasst wird.
  - `category` beim Ticket-Update (siehe oben) ist kein Backlog-Eintrag, aber derselbe
    Fehler-Typ wie B4 vor dieser Iteration — Kandidat fuer eine kleine Folge-Unit oder eine
    Erweiterung, falls das Team das priorisiert.

## Iteration 53 — intake-external-requester — done — 2026-08-06 01:2x
- commit: aa815b47
- gebaut: Migration 000291 macht `tickets.requester_id` NULLable und setzt
  `chk_tickets_requester_identity` darueber (`requester_id IS NOT NULL` ODER
  `requester_is_external AND requester_email <> ''`), dazu ein Partial-Index auf
  `(tenant_id, requester_email)`. `Ticket.RequesterID` ist jetzt `*uuid.UUID` durch Model,
  Service, Repository, gRPC-Server und Tests; `Service.CreateTicket` nimmt einen Pointer und
  spiegelt den CHECK vorab als `ErrMissingRequester` (→ InvalidArgument), der gRPC-Server
  akzeptiert ein LEERES `requester_id` als "externer Requester" und faellt bei einem
  vorhandenen, aber kaputten Wert weiterhin hart auf InvalidArgument.
- gate (DATABASE_URL gesetzt, Rolle `kmuhub_app`): `go build -p 2 ./...` ok | `go vet` ok |
  `golangci-lint run` (helpdesk, gateway, server) **0 issues** | `go test -count=1 -v
  ./internal/helpdesk/... ./internal/gateway/... ./internal/server/... ./internal/testutil/...`
  **1795 PASS / 0 FAIL / 0 SKIP** (inkl. `TestOpenAPIRouteDrift`, `TestOpenAPISpecDrift`,
  `TestAllPublicTablesHaveRLSOrAreAllowlisted`). Migration lokal up **und** down **und** wieder
  up gefahren (Kopf 291) — die down-Richtung loescht bewusst Zeilen ohne requester_id, weil
  `SET NOT NULL` sonst nicht laufen kann; das steht als Kommentar in der `.down.sql`.
- verify vorgaenger (4ada610d): sauber. Handler geht ueber `helpdeskClient.UpdateTicket`,
  `.proto` und `.pb.go` liegen im selben Commit, `openapi.yaml` ebenfalls, keine neue Tabelle
  und kein neuer Permission-Guard. Zusaetzlich geprueft, was der neue `custom_fields = $13`
  im UPDATE fuer die uebrigen Schreiber bedeutet: alle sechs `repo.UpdateTicket`-Aufrufer in
  `service.go` laden das Ticket vorher ueber `GetTicketByID`, schreiben den Wert also
  unveraendert zurueck — kein stiller Wipe.
- **scope=own fuer Externe: entschieden, unsichtbar zu lassen** (done_when verlangt die
  Entscheidung samt Begruendung). `scope=own` ist ein RBAC-Datenscope fuer eingeloggte
  Mitarbeiter und vergleicht gegen die User-UUID; ein externes Ticket hat keine. Die
  naheliegende Alternative — ueber `requester_email` matchen — waere ein Identitaets-Match auf
  einem Wert, der ungeprueft aus dem Intake-Body kommt: wer die Adresse eines Kollegen
  eintippt, schoebe sein Ticket in dessen own-Scope (und mit dem oeffentlichen Submit aus B8
  koennte das jeder von aussen). Externe Tickets erreichen einen Agenten also ueber die
  Zuweisung (`assignee_id` matcht weiterhin) oder ueber einen weiteren Scope. Begruendung steht
  als Kommentar am Filter in `ListTickets`, gepinnt von
  `TestExternalRequester_InvisibleToOwnScope` (zugewiesen sichtbar, unzugewiesen nicht).
- **Read-Pfade:** `ticketSelectColumns` nutzte bereits `LEFT JOIN users req`, die Read-Seite
  verliert also nichts — `TestExternalRequester_ReadableThroughEveryTicketRead` belegt das fuer
  `GetTicketByID` UND `ListTickets` (zwei verschiedene Queries auf derselben Spaltenliste), inkl.
  `requester_name` aus der Spalte statt aus dem JOIN.
- **`ListDueCsatSurveys` mitgenommen** (Hinweis aus Iteration 50/52): der `JOIN users` war
  ein INNER JOIN und liess genau die externen Requester fallen, fuer die die Umfrage gedacht
  ist. Jetzt `LEFT JOIN` + `COALESCE(NULLIF(t.requester_email,''), req.email)` fuer die Adresse
  und dieselbe Vorrangregel wie beim Ticket-Read fuer den Namen; das WHERE filtert auf die
  kombinierte Adresse statt auf `req.email <> ''`. Test
  `TestListDueCsatSurveys_ReachesExternalRequester`.
- **Duplikat-Erkennung fuer Externe abgeschaltet, nicht umgebaut:** `DetectDuplicates` gruppiert
  ueber `requester_id`; mit NULL waere die SQL-Bedingung strukturell immer leer (NULL = NULL),
  also ein stiller Nulltreffer statt eines sichtbaren Verhaltens. Jetzt frueher Return mit
  Begruendung im Code — ueber die unverifizierte Mailadresse zu gruppieren waere derselbe
  Identitaetsfehler wie beim own-Scope.
- **Proto NICHT angefasst.** `requester_id` bleibt `string`; ein externer Requester liefert "".
  Der FE-Typ (`helpdesk-types.ts:36`) deklariert `requester_id: string` nicht-optional und
  rendert den Anzeigenamen ohnehin aus `requester_name` (der MSW-Mock legt dort teilweise sogar
  Klartextnamen ab), ein weggelassenes Feld waere also die groessere Abweichung. Kein Regen,
  kein `optional`-Umbau — bewusst, damit B7/B8 nicht auf einem Proto-Wechsel sitzen.
- Testfixtures: `Ticket.RequesterID` ist jetzt ein Pointer, dadurch mechanische Anpassung von
  16 Struct-Literalen und ~30 `CreateTicket`-Aufrufen (`uuidPtr`-Helper in
  `external_requester_db_test.go`). Keine Verhaltensaenderung an diesen Tests; der Fake-Repo-
  Own-Scope-Filter in `service_test.go:75` spiegelt jetzt zusaetzlich die NULL-Semantik der
  SQL-Variante.
- offen fuer die naechste Iteration:
  - `intake-form-target` (B6) ist als naechstes dran (deps leer).
  - Das Gateway setzt `RequesterId` weiterhin hart aus der Session (`route_helpdesk.go:309`) —
    richtig fuer den authentifizierten Pfad. Der NULL-Fall ist damit heute nur ueber gRPC
    erreichbar und wird erst von B7/B8 (Formular-Dispatch, oeffentlicher Submit) benutzt; bis
    dahin ist die Lockerung Fundament, kein aktiver Pfad.
  - `category` beim Ticket-Update bleibt der bekannte, unberuehrte Datenverlust (siehe
    Iteration 52).
