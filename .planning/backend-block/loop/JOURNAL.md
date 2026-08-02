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
