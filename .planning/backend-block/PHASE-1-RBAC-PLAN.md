# Phase 1 — RBAC-Fundament: Bau-Plan (bau-fertig fürs nächste Fenster)

> **Zweck:** Konkreter, ausführbarer Bauplan für das RBAC-Backend-Fundament. Gehört zu
> `WELLEN-BRIEFING.md` §4 Phase 1. Detail-Backlog: `../backend-gaps.md` §🔴 RBAC. FE-SSOT:
> `desktop/src/renderer/src/config/capability-catalog.ts` + `mocks/data/rbac.ts` (`ROLE_DEFS`) +
> `mocks/handlers/rbac.ts` (Referenz-Endpoints) + `api/rbac-types.ts` (Typen).
> **Stand:** 2026-07-21, verifiziert per Code-Scan. Migrationskopf Repo **000243** → nächste **000244**.
> **Modus:** seriell, Hauptsession selbst (nicht parallelisierbar — ein kohärentes Auth-/RLS-System). Kein Agenten-Fan-out. Opus für die Umsetzung.

---

## 1. Ist-Zustand (verifiziert)

**BE-Authz heute:**
- **Permissions liegen im JWT** (`auth/token.go:20` `Claims{uid, tid, roles []string, perms []string}`). Rollen+Permissions werden **einmalig beim Login/Refresh** aus der DB in den Access-Token gebacken (`auth/service.go:380 createTokenPair`) — **kein Live-DB-Reload pro Request**. Bei Rollenänderung veraltet der Token bis Refresh.
- **`middleware.RequirePermission(resource, action)`** (`middleware/rbac.go:29`) ist ein **reiner In-Memory-Vergleich** `resource:action` gegen den JWT-`perms`-Slice aus dem Context. Kein DB-Query, kein Cache. **812 Vorkommen über 34 `route_*.go`** — der primäre Gate.
- Repo-Methoden (`auth/postgres_repository.go`): `GetUserRoles` (171, `[]string` Rollennamen), `GetUserPermissions` (193, `[]string` `resource:action`-Namen aus `permissions.name`), `UserHasPermission` (216, EXISTS-Query). **Keine Scope-, keine Source-Info.**
- **Kein `/me/permissions`-Endpoint** (bestätigt). Auth-Route-Handler gehen über den **gRPC-Client** (`route_auth.go:50 getAuthClient()` → `authv1.AuthServiceClient`) — House-Style, auch wenn auth co-located ist. `/auth/me` (`HandleGetProfile`) existiert als Muster.
- Rollen-Zuweisung: `assignRoleRequest{RoleName validate:"oneof=admin manager member"}` (`route_auth.go:330`), gated by `RequireRole("admin")`, → `INSERT INTO user_roles ... WHERE r.name=$2 ON CONFLICT DO NOTHING` (kein Fehler bei unbekannter Rolle).
- **Bonus-Fund:** `CheckPermission`-gRPC-RPC existiert (`server/grpc.go:203` → frischer DB-Query), aber nur in Tests verdrahtet — Kandidat für das spätere Propagation-Problem (Rechteänderung ohne Re-Login).

**Schema heute (seit 000002 unverändert, 0 ALTERs):**
- `roles(id, name UNIQUE, description, created_at)` — **kein `tenant_id`, kein `based_on`, kein `color`**.
- `permissions(id, name UNIQUE, resource VARCHAR(50), action VARCHAR(50), created_at)`, unique `(resource, action)`.
- `role_permissions(role_id, permission_id, PK(role_id, permission_id))` — **keine `scope`-Spalte**.
- `user_roles(user_id, role_id, assigned_at)`.
- **Nur 3 Rollen je geseedet:** `admin`/`manager`/`member` (000002). Keine späteren Rollen-Seeds; alle ~40 `*permission*`-Migrationen fügen nur `permissions`+`role_permissions`-Zeilen für diese 3 hinzu.

**FE-Contract (Referenz):**
- **Katalog:** `CapabilityDef {key, fine, scopeable}`, Key = `modul:subject:action` (z.B. `work:task:edit`). ~249 L2/L3-Keys + 30 L1-View-Keys (`<modul>:module:view`, via `moduleViewKey()`, NICHT im Katalog) = **~279 Keys**, 30 Module (3 leer: dashboard/video/notifications).
- **`ROLE_DEFS`:** `Record<RoleId, {grants: Record<key, scope>}>` — **flaches** `{key: 'own'|'team'|'all'}`. 7 Presets, Grant-Zahlen: `admin`~279 (**explizit, kein Wildcard** — neue Module bleiben default-deny), `manager`~190, `member`~105, `it_admin`~103, `hr_admin`~70, `readonly`~70, `extern`~11. Scope-Overrides gezielt (manager Team-Drawer `scope:team`, member `scope:own`, `fuhrpark:gps:read`/`berichte:datev:read` bewusst aus Read-Listen gefiltert). `CUSTOM_ROLE_LIMIT=20`/Tenant.
- **Typen (`rbac-types.ts`):** `CapabilityScope='own'|'team'|'all'`, `SCOPE_ORDER{own:0,team:1,all:2}`, `Role{id,name,description,tenantId:null,basedOn:null,isSystem,color,memberCount,capabilityCount}`, `EffectivePermissions{roles:[{id,name,isSystem,color}], capabilities: Record<key,{scope,sources:string[]}>}`, `EffectivePermissionsResponse{permissions}`.
- **⚠ Wire-Shape-Falle:** interne Seed-Struktur `{key: scope}` (nackt), aber API-Response `RoleGrants = {key: {scope}}` (**Objekt-Wrapper**). BE muss `{key:{scope:"own"}}` liefern, nicht `{key:"own"}`.

**Kern-Einsicht:** Das FE-Gating hängt **allein** an `GET /me/permissions` (liefert die feinen Capability-Keys + Scope). Die **serverseitige `RequirePermission`-Durchsetzung** ist ein **separates, gröberes Thema** — sie fein zu machen ist **Phase 2** (per-Modul). Phase 1 macht das FE gegen echtes BE lauffähig, ohne alle 812 Gates umzuschreiben.

---

## 2. Design-Entscheidungen (getroffen, mit Begründung)

- **D1 — Fine-Key-Mapping:** FE-Key `work:task:edit` → `permissions.name='work:task:edit'`, `resource='work:task'`, `action='edit'`. Unique `(resource, action)` hält. Grobe Alt-Permissions (`files`/`write`) **koexistieren** (die 812 Alt-Gates bleiben unberührt bis Phase 2). `/me/permissions` gibt `permissions.name` als Capability-Key zurück.
- **D2 — `/me/permissions` via neuer auth-gRPC-RPC (Proto-Vor-Welle):** House-Style ist gRPC-Client (verifiziert). Neuer RPC `GetEffectivePermissions(user_id)` am auth-Service → auth.proto ändern + **zentral regen** (`protoc` auth-Target, Toolchain verifiziert versionsgleich). Kein Gateway-Direktzugriff auf `localAuthService` (bräche das Muster).
- **D3 — Seed aus `ROLE_DEFS` generieren, nicht abtippen:** ~828 Grant-Zeilen + ~279 Permission-Zeilen. Ein **Generator-Script** (Node/TS in `desktop/scripts/`, liest `ROLE_DEFS` + `CAPABILITY_CATALOG`) emittiert die `000244_*.up.sql`-INSERTs. Verhindert Abtipp-Drift gegen die SSOT. `lean: einmaliger Snapshot; Codegen-in-CI (FE-Katalog→Seed) wenn der Katalog häufig driftet.`
- **D4 — Preset-Rollen:** `admin`/`manager`/`member` **bleiben** (ihre IDs sind in `user_roles` referenziert). **4 neue ADD:** `it_admin`/`hr_admin`/`readonly`/`extern`. Kein `hr`/`it_support` in der DB (nur 3 existieren) → nichts umzumappen, nur einfügen. Alle Presets `tenant_id=NULL` (System). `isSystem` ableitbar aus `tenant_id IS NULL` (keine eigene Spalte nötig).
- **D5 — Scope-Auflösung:** `role_permissions.scope VARCHAR NOT NULL DEFAULT 'all'`. Bei Multi-Rollen-Union: **breiterer Scope gewinnt** (`SCOPE_ORDER own<team<all`), `sources` kumuliert alle beitragenden Rollen-IDs. Logik im auth-**Service** (thick), nicht im Handler.
- **D6 — RLS auf `roles`/`role_permissions`:** System-Presets (`tenant_id IS NULL`) **global lesbar** für alle Tenants, Custom-Rollen tenant-scoped. Policy: `USING (tenant_id IS NULL OR tenant_id = current_tenant())`. `role_permissions` erbt über den `roles`-Join bzw. eigene Policy. **⚠ `kmuhub_app` (NOSUPERUSER NOBYPASSRLS) muss die NULL-Tenant-Presets lesen können** — sonst 403/leer für alle. RLS-Smoke Pflicht.
- **D7 — Name-Eindeutigkeit pro Tenant:** `idx_roles_name` (global unique) → unique auf `(COALESCE(tenant_id,'00000000-0000-0000-0000-000000000000'), name)` (zwei Tenants dürfen dieselbe Custom-Rolle „Vertrieb" haben; NULL-Presets bleiben global eindeutig).

---

## 3. Welle 1a — Datenmodell + Resolver + Seed (macht das FE echt)

> Ergebnis: `GET /auth/me/permissions` liefert die vollständigen, korrekten feinen Rechte → das komplette RBAC-FE (R-1…R-4) läuft gegen echtes BE statt Client-Fallback.

**Schritt 1 — Proto-Vor-Welle (`backend/proto/auth/v1/auth.proto`):**
```proto
rpc GetEffectivePermissions(GetEffectivePermissionsRequest) returns (GetEffectivePermissionsResponse);

message GetEffectivePermissionsRequest { string user_id = 1; }   // leer = aus Context (me)
message EffectiveRole { string id = 1; string name = 2; bool is_system = 3; string color = 4; }
message EffectiveCapability { string key = 1; string scope = 2; repeated string sources = 3; }
message GetEffectivePermissionsResponse {
  repeated EffectiveRole roles = 1;
  repeated EffectiveCapability capabilities = 2;
}
```
Regen: `protoc --go_out=. --go_opt=module=github.com/kmuhub/kmuhub --go-grpc_out=. --go-grpc_opt=module=github.com/kmuhub/kmuhub proto/auth/v1/auth.proto` (Makefile-`proto`-Muster). protoc-Pfad + `$HOME/go/bin` auf PATH.

**Schritt 2 — Migration 000244** (`make migrate-create name=rbac_foundation`, dann füllen):
- `ALTER TABLE roles ADD COLUMN tenant_id UUID NULL, ADD COLUMN based_on UUID NULL REFERENCES roles(id) ON DELETE SET NULL, ADD COLUMN color VARCHAR(40) NOT NULL DEFAULT '';`
- `DROP INDEX idx_roles_name; CREATE UNIQUE INDEX idx_roles_tenant_name ON roles (COALESCE(tenant_id,'00000000-0000-0000-0000-000000000000'), name);`
- `ALTER TABLE role_permissions ADD COLUMN scope VARCHAR(8) NOT NULL DEFAULT 'all' CHECK (scope IN ('own','team','all'));`
- RLS: `ALTER TABLE roles ENABLE ROW LEVEL SECURITY;` + Policy `USING (tenant_id IS NULL OR tenant_id = current_setting('app.current_tenant',true)::uuid)` (Muster aus dem RLS-Rollout Sprint 4 prüfen — `current_tenant()`-Helper existiert dort). Analog `role_permissions` (join-abgeleitet oder eigene Policy). **System-global-Liste (ADR-006)** ggf. ergänzen.
- Seed (aus D3-Generator): 4 neue Rollen-Rows (tenant_id NULL, color aus ROLE_DEFS) + ~279 `permissions`-Rows (fehlende feine Keys, `ON CONFLICT (resource,action) DO NOTHING`) + `role_permissions`-Rows mit `scope` für alle 7 Presets (INSERT ... SELECT gegen die neuen permission-Ids).

**Schritt 3 — Seed-Generator** (`desktop/scripts/gen-rbac-seed.mjs`): liest `ROLE_DEFS`+`CAPABILITY_CATALOG`, emittiert die INSERT-Blöcke für Schritt 2. Einmal laufen, Output in die `.up.sql` einbetten. `.down.sql`: DELETE der 4 Rollen + der feinen permissions + scope/columns revert.

**Schritt 4 — auth-Service Resolver** (`internal/auth/`):
- Repo: `GetEffectivePermissions(ctx, userID) ([]EffectiveGrantRow, error)` — `SELECT p.name, rp.scope, r.id AS role_id, r.name, r.color, (r.tenant_id IS NULL) AS is_system FROM permissions p JOIN role_permissions rp ON rp.permission_id=p.id JOIN roles r ON r.id=rp.role_id JOIN user_roles ur ON ur.role_id=r.id WHERE ur.user_id=$1`.
- Service: Union-Auflösung — pro Key breitester Scope gewinnt (`SCOPE_ORDER`), `sources` kumuliert Rollen-Ids; sammelt die distinct Rollen für `roles[]`.

**Schritt 5 — auth-gRPC-Server** (`internal/server/grpc.go`): `GetEffectivePermissions`-RPC implementieren → user_id aus Request oder `middleware.GetUserID(ctx)`, Service aufrufen, in Proto mappen.

**Schritt 6 — Gateway** (`internal/gateway/route_auth.go`):
- `HandleGetMyPermissions` → `GET /api/v1/auth/me/permissions` (user aus Context) — Route bei `/auth` registrieren (neben `/me`).
- `HandleGetUserPermissions` → `GET /api/v1/admin/users/{id}/permissions` (path-param, `RequireRole("admin")` o. `admin:user:read`).
- Beide: RPC callen, in FE-Shape mappen: `{"permissions": {"roles":[{id,name,isSystem,color}], "capabilities": {"<key>": {"scope":"own","sources":["manager"]}}}}` (**Objekt-Wrapper pro Key**, D1/Wire-Falle).

**Gate 1a:** `go build -p 2 ...` (⚠ `./...` OOM → gezielt), `go vet`, auth-Unit-Tests, golangci scoped, **RLS-Smoke** (kmuhub_app liest NULL-Tenant-Presets → capabilities nicht leer), lokaler Live-Check `/auth/me/permissions` gegen echtes auth+DB.

---

## 4. Welle 1b — Rollen-Admin-API + Guardrails

> Ergebnis: Custom-Rollen tenant-scoped verwaltbar; Verwaltungs-UI (A-2/RBAC-Builder) läuft echt.

Neue auth-gRPC-RPCs (Proto-Vor-Welle #2) + Gateway-Routen — Contracts exakt aus `mocks/handlers/rbac.ts` (siehe §6):
- `GET /admin/roles` → `{roles:[Role]}` (mit `memberCount`/`capabilityCount`-Aggregat).
- `POST /admin/roles` (Body `{name,description,color,basedOn}`) → **Klon**: `role_permissions` von `basedOn` 1:1 kopieren, `tenant_id=current`. 201 `{role}`. Fehler `role_limit_reached`(409, ≥20/Tenant), `role_name_exists`(409), `not_found`(404, basedOn), 422 (Pflichtfelder).
- `PATCH /admin/roles/{id}` `{name?,description?,color?}` → `{role}`. `preset_immutable`(403, tenant_id NULL), `role_name_exists`(409).
- `DELETE /admin/roles/{id}` → 204. `preset_immutable`(403), `role_has_members`(409, >0 Träger).
- `GET/PUT /admin/roles/{id}/permissions` → `{roleId, grants:{key:{scope}}}` (PUT = Vollersatz).
- `POST /users/{id}/roles` `{roleId}` + `DELETE /users/{id}/roles/{roleId}` → beide `{roles:[...]}`. `last_admin`(409, letzter admin-Träger). **Validator-Entkopplung:** `assignRoleRequest oneof` raus → dynamisch gegen `roles`-Tabelle (tenant-scoped) validieren.
- **Guardrails serverseitig:** Mindestens-1-Admin · Selbst-Aussperr-Schutz · Privilege-Escalation (niemand vergibt Rechte über die eigenen hinaus) · Preset-Immutability · Custom-Limit 20.
- **Audit-Events** für jede Rechteänderung (append-only, auf `audit_log`/000222 aufsetzen; Taxonomie `permission.role_created/updated/deleted`, `permission.assigned/revoked` — R-6 reserviert `permission.override_*`).

**Gate 1b:** wie 1a + Guardrail-Tests (last-admin, escalation, preset-immutable) + Live-Check des Builder-Flows.

---

## 5. Serialisierung + Deploy-Awareness

- **Migrations-Nummer:** 000244 (Repo-Kopf 243). 1b braucht ggf. 000245 (Audit-Taxonomie/Indexe).
- **Proto-Regen:** 2× zentral (1a auth-RPC, 1b admin-RPCs). Versionsgleiche Toolchain verifiziert (`protoc-gen-go v1.36.11` = go.mod).
- **Build:** `go build ./...` OOMt (Parallel-RAM) → immer gezielt `-p 2` auf die betroffenen Pakete (`./proto/auth/... ./internal/auth/... ./internal/server/... ./internal/gateway/... ./cmd/auth/... ./cmd/gateway/...`).
- **Deploy:** additive Endpoints — aber die **RLS-Policies auf `roles`/`role_permissions` sind heikel** (D6): vor Push lokal verifizieren, dass `kmuhub_app` Presets liest. Kein neues `config.RequireX`, kein neuer `modules.*`-Flag → sonst deploy-sicher. CD auto-deployt bei Push.
- **Seed-Migration ist groß** (~1100 Rows) — idempotent halten (`ON CONFLICT DO NOTHING`), damit ein Re-Run/Prod-Apply nicht bricht.

---

## 6. Contract-Referenz (aus FE-Scan)

**`GET /me/permissions` + `/admin/users/{id}/permissions` Response:**
```json
{ "permissions": {
  "roles": [{ "id": "manager", "name": "Team Lead", "isSystem": true, "color": "hsl(217 91% 60%)" }],
  "capabilities": { "work:task:edit": { "scope": "team", "sources": ["manager","hr_admin"] } }
}}
```
Fehlender Key = verboten (Default-Deny). Union: breitester Scope gewinnt, `sources` kumulieren.

**Endpoint-Tabelle (Fehler-Codes):** siehe §4 + `mocks/handlers/rbac.ts` (Referenz-Impl). Status: preset/immutable=403, Konflikte=409, fehlend=404, Pflichtfeld=422.

**7 Presets (Grant-Zahl, Muster):** `admin`~279 (explizit, kein Wildcard) · `it_admin`~103 (alles außer finance; kein HR-Datenladen) · `hr_admin`~70 (team/schichten/zeiterfassung voll; `role:assign` ohne `role:create/edit`) · `manager`~190 (industry voll; Team-Drawer `scope:team`, nie salary; ohne `datev:read`) · `member`~105 (own-Edits; automatisierung unsichtbar) · `readonly`~70 (nur reads + `datev:read` Steuerberater) · `extern`~11 (nur work/documents-Basics). **Wert exakt aus `ROLE_DEFS` (Generator D3).**

---

## 7. Offene Entscheidungen für den Builder (nächstes Fenster)

1. **Seed-Strategie:** generierte Snapshot-Migration (empfohlen, schnell) vs. Codegen-in-CI (FE-Katalog→Seed automatisch, mehr Infra). → Snapshot für 1a, Codegen als Follow-up-Ticket wenn Katalog driftet.
2. **`/me/permissions`-Filter:** nur Katalog-Keys zurückgeben oder auch die groben Alt-Permissions? → Empfehlung: auf Katalog-Keys filtern (FE ignoriert Unbekannte eh, aber sauberer).
3. **RLS-Policy-Wortlaut:** exakt am Sprint-4-Muster (`current_tenant()`/`app.current_tenant`) ausrichten — vor dem Schreiben eine bestehende tenant-Policy als Vorlage lesen (`migrations/0001xx_*rls*`).
4. **Propagation (🟠, kann warten):** Rechteänderung ohne Re-Login. Optionen: kurzlebige Access-Tokens + Refresh zieht neue perms, ODER `RequirePermission` auf den `CheckPermission`-RPC (frischer DB-Query) umstellen für kritische Gates. Nicht Phase-1-blockierend — als 🟠 in `backend-gaps.md`.

---

## 8. Startsequenz nächstes Fenster

```
1. git pull
2. Lies dieses Doc + WELLEN-BRIEFING.md §4 Phase 1 + backend-gaps.md §🔴 RBAC
3. /model opus
4. Welle 1a: Proto-Vor-Welle → Migration 000244 (+ Generator) → Resolver → gRPC → Gateway → Gate → Live-Check → 1 Commit
5. RLS-Smoke NICHT überspringen (kmuhub_app liest NULL-Tenant-Presets)
6. Pause/Review → Welle 1b (Rollen-Admin-API + Guardrails)
```
