# Backend-Nachtloop — Journal (Lauf 3)

Append-only. Eine Iteration = ein Eintrag. Vorlage:

```markdown
## Iteration <n> — <unit-id> — <done|blocked> — <YYYY-MM-DD HH:MM>
- commit: <sha oder ->
- gebaut: <ein bis drei Zeilen, was real existiert>
- gate: build ok | vet ok | lint ok | test ok | migration ok | rls-smoke ok|n.a.
- verify vorgaenger: <sauber | Befund + angelegte Fix-Unit>
- offen: <was Luke morgens pruefen muss>
```

Uhrzeiten im Journal sind geraten — der Agent hat keine Uhr. Die Wahrheit steht in `logs/run.log`.

**Laeufe 1 und 2** (26.–28.07., 60 Iterationen, 58 Units, alles ueber PR #15 auf `main`):
`archive/lauf-1-2/JOURNAL.md` und `archive/lauf-1-2/BACKLOG.yml`.

---

## Lauf 3 — Ausgangslage (2026-08-01)

- Branch `backend-loop` == `main` (`2ce86506`), Repo-Migrationskopf **000255**.
- Schwerpunkt laut Absprache mit Luke: **RBAC Phase 1 Welle 1a** und **E-Rechnung-Ausgang (Welle 5)**,
  danach RBAC Phase 2 (additiv), drei FE-only-Cluster, zuletzt die Code-Scan-Funde.
- **Neu freigegeben:** Phase-1-Units der **Welle 1a**. Welle **1b** (Rollen-CRUD + Guardrails) und
  **Phase 4** (Branchen-BE) bleiben gesperrt.
- Zwei Korrekturen gegenueber `PHASE-1-RBAC-PLAN.md`, verifiziert gegen
  `migrations/000118_rls_foundation.up.sql:42-79` — stehen in den Notes von `p1a-migration`:
  1. Das RLS-Setting heisst **`app.tenant_id`**, die Helper **`current_tenant_id()`** /
     **`is_system_context()`**. Der Plan schreibt `app.current_tenant` — existiert in keiner Migration.
  2. `roles` darf **nicht** ueber `CALL enable_tenant_rls('roles')` laufen: das Standardmuster blendet die
     System-Presets mit `tenant_id IS NULL` aus und erzeugt 403 fuer alle.

---

## Iteration 1 — p1a-proto — done — 2026-08-01 21:40

- commit: f2b98a7b
- gebaut: RPC `GetEffectivePermissions` in `proto/auth/v1/auth.proto` (direkt hinter
  `CheckPermission`, weil naechste Verwandtschaft) plus die vier Messages
  `GetEffectivePermissionsRequest{user_id}`, `EffectiveRole{id,name,is_system,color}`,
  `EffectiveCapability{key,scope,repeated sources}` und
  `GetEffectivePermissionsResponse{repeated roles, repeated capabilities}`.
  `.pb.go` und `_grpc.pb.go` im selben Commit regeneriert.
- gate: build ok (`./proto/auth/... ./internal/auth/...` **und** `./...` komplett) | vet ok |
  lint ok (0 issues) | test ok (`./internal/auth/...` 7,6 s mit gesetzter `DATABASE_URL` gegen
  `kmuhub_app` — also real gelaufen, nicht ueber `SkipIfNoDB` weggeskippt) |
  migration n.a. | rls-smoke n.a. (reine Schnittstellendefinition, kein Datenpfad)
- verify vorgaenger: n.a. — Vorgaenger-Commit `7e0b1a4a` ist der Backlog-Reset fuer Lauf 3, enthaelt
  keinen Code. Erste Code-Iteration des Laufs.
- offen: Der neue RPC steht erwartungsgemaess in `UnimplementedAuthServiceServer`
  (`auth_grpc.pb.go:530`) — das ist Codegen-Boilerplate und **kein** Stub-Fund, so auch in den Notes
  von `p1a-proto` vorab festgehalten. Die Implementierung kommt in `p1a-resolver` (Service) und
  `p1a-grpc` (Handler); erst danach ist der RPC ueber die Leitung nutzbar.
  Kein handgeschriebener Fake des `AuthServiceClient`/`AuthServiceServer`-Interfaces im Repo
  (geprueft) — die Interface-Erweiterung bricht also keine Testdoubles.
  Der 819-Zeilen-Diff in `auth.pb.go` ist harmlos: alle Loeschungen sind
  `msgTypes[n]`-Index-Verschiebungen und `rawDesc`-Bytes, die protoc-gen-go-Version hat sich nicht
  geaendert (Header-Diff leer).
  Naechste Unit: `p1a-seed-generator` (deps erfuellt).

## Iteration 2 — p1a-seed-generator — done — 2026-08-01 22:10

- commit: (siehe unten)
- gebaut: `desktop/scripts/gen-rbac-seed.mjs` liest `mocks/data/rbac.ts` (`ROLE_DEFS`) und
  `config/capability-catalog.ts` (`CAPABILITY_CATALOG` + `MODULE_KEYS`/`moduleViewKey` aus
  `config/capabilities.ts`) und emittiert auf stdout: (a) `INSERT INTO permissions` fuer alle 282
  Keys (30 L1-View-Keys + 252 feine Katalog-Keys), (b) `INSERT INTO roles` fuer die 4 fehlenden
  Presets (it_admin/hr_admin/readonly/extern, tenant_id NULL) plus `UPDATE roles SET color=...` fuer
  die 3 bestehenden (admin/manager/member), (c) `INSERT INTO role_permissions ... SELECT` je Rolle
  gruppiert nach Scope (own/team/all) mit IN-Liste — 14 Bloecke fuer 7 Rollen. Alle INSERTs mit
  `ON CONFLICT DO NOTHING` wie in den Notes gefordert.
  Toolchain-Fund: `tsx` (bereits Dependency) muss mit `--tsconfig tsconfig.web.json` laufen, sonst
  loest es den `@/`-Alias aus `rbac.ts` nicht auf. Zweiter Fund, unabhaengig vom Alias: auf dieser
  Maschine (tsx 4.21.0/Node 22.19.0/Windows) landen benannte Exports von `.ts`-Dateien beim Import via
  `tsx` unter `.default` statt als Top-Level-Named-Exports (reproduziert auch mit einer Datei ohne
  jeden `@/`-Import) — der Generator destructured deshalb defensiv `pkg.default ?? pkg`. Lauffaehiger
  Befehl steht im Script-Header: `cd desktop && npx tsx --tsconfig tsconfig.web.json
  scripts/gen-rbac-seed.mjs`.
- gate: Script-Lauf exit 0, stderr meldet die Grant-Zahlen: admin 282, it_admin 104, hr_admin 71,
  manager 205, member 110, readonly 74, extern 11 (Summe 857 — Notes-Schaetzung war "~828", plausibel).
  282 permissions-Zeilen, das erwartete Groessenmuster "~279". Alle 30 L1-View-Keys ueber
  `moduleViewKey()`/`MODULE_KEYS` mitgeneriert, geprueft per Diff gegen die Katalog-Summe (30+252=282).
  Struktur-Check des SQL-Outputs (kein DB-Zugriff moeglich, da `roles.tenant_id/color` und
  `role_permissions.scope` erst in `p1a-migration` entstehen): 16 `INSERT`, 3 `UPDATE`, 19
  Zeilen auf `;`, Anfuehrungszeichen- und Klammernzahl gerade/balanciert. `extern`-Block stichprobenhaft
  gegen `rbac.ts` gegengelesen (6 all + 3 team + 2 own = 11, exakt der ROLE_DEFS-Wert).
  go build/vet/test: n.a. — reine Frontend-Tooling-Datei, kein Go-Code angefasst.
- verify vorgaenger: sauber. `f2b98a7b` (p1a-proto) gegen alle acht Fehlerklassen geprueft:
  Proto-Diff matcht exakt die Notes-Spezifikation (RPC + 4 Messages), `.proto`-Aenderung UND
  `_grpc.pb.go`-Regen im selben Commit vorhanden und inhaltlich konsistent (grep bestaetigt
  `GetEffectivePermissions` an allen erwarteten Stellen: Client-Interface, Server-Interface,
  Unimplemented-Stub, Handler-Registrierung). Kein Gateway/Guard/Tenant/Wire-Shape-Code in diesem
  Commit, diese Klassen n.a.
- offen: **Sicherheits-relevante Beobachtung fuer `p1a-migration` (naechste Unit, opus):** die
  `own`/`team`-Scope-Grants in `ROLE_DEFS` (z.B. member `work:task:edit`→own,
  manager `team:data_personal:view`→team, extern `work:task:read`→team) betreffen ausschliesslich
  Keys, die VOR dieser Migration noch KEINE `role_permissions`-Zeile hatten — verifiziert per Grep
  gegen alle bestehenden `*_permissions*.up.sql`-Migrationen (work:task, crm:contact,
  team:data_personal/data_job/salary/documents/absence_data, schichten:swap:read/create fuer member,
  rapporte:report fuer member, helpdesk:ticket: keine Treffer ausser admin-only in 000161/000164).
  Das `ON CONFLICT (role_id, permission_id) DO NOTHING` in Block (c) kann deshalb bei DIESER Migration
  keinen bestehenden Scope stillschweigend auf `all` (Spalten-Default) haengen lassen — aber bei
  spaeteren Katalog-Aenderungen mit neuen Scope-Downgrades auf einen Key mit bereits existierender
  Zeile wuerde genau das passieren (DO NOTHING gewinnt gegen den neuen, engeren Scope). Kein Fund fuer
  HEUTE, aber ein Muster, das `p1a-migration` kennen sollte, falls es den generierten Block (c)
  irgendwann per Diff gegen eine bereits teilmigrierte DB laufen laesst.
  `p1a-migration` muss aus den Notes zusaetzlich sicherstellen, dass die neue
  `idx_roles_tenant_name`-Indexdefinition EXAKT `COALESCE(tenant_id, '00000000-0000-0000-0000-000000000000'::uuid)`
  lautet — das ist die Ausdrucksform, gegen die der generierte `ON CONFLICT`-Klausel in Block (b)
  matcht; weicht der Index-Ausdruck (z.B. ohne `::uuid`-Cast oder andere Klammerung) davon ab, findet
  Postgres keinen Arbiter-Index und die INSERT schlaegt fehl.
  Keine Migration in dieser Iteration angefasst (das ist explizit `p1a-migration`s Scope) — DB-Gate
  daher nicht gelaufen, es gibt noch keine anzuwendende `.sql`-Datei.

## Iteration 3 — p1a-migration — done — 2026-08-01 15:54

- commit: (siehe unten)
- gebaut: Migration `000256_rbac_phase1a_role_scope_and_presets` (lokaler Kopf war 255 = Repo-Kopf,
  zur Laufzeit ermittelt). `.up.sql`: (a) `roles` bekommt `tenant_id UUID NULL REFERENCES tenants(id)
  ON DELETE CASCADE`, `based_on UUID NULL REFERENCES roles(id) ON DELETE SET NULL`,
  `color VARCHAR(40) NOT NULL DEFAULT ''`; (b) `idx_roles_name` (global unique) ersetzt durch
  `idx_roles_tenant_name ON roles (COALESCE(tenant_id,'00000000-0000-0000-0000-000000000000'::uuid), name)`
  — Ausdruck byte-identisch zur ON-CONFLICT-Klausel des Generators, sonst fehlt der Arbiter-Index;
  (c) `role_permissions.scope VARCHAR(8) NOT NULL DEFAULT 'all'` mit benanntem CHECK
  (`own`/`team`/`all`); (d) RLS auf beiden Tabellen; (e) Generator-Output von
  `desktop/scripts/gen-rbac-seed.mjs` woertlich eingebettet. `.down.sql` gefuellt.
  **RLS-Form (Abweichung vom Standardmuster, bewusst):** statt einer Policy zwei — `tenant_isolation_read`
  (`FOR SELECT`, `tenant_id IS NULL OR tenant_id = current_tenant_id() OR is_system_context()`) und
  `tenant_isolation_write` (`FOR ALL`, ohne die NULL-Bedingung). Grund: DELETE wertet **ausschliesslich**
  USING aus, nie WITH CHECK. Eine einzelne permissive Policy, die die Presets lesbar macht, macht sie
  damit auch fuer jeden Tenant loeschbar. Mit der Trennung gilt: lesen ja, schreiben/aendern/loeschen
  nein. `role_permissions` analog ueber den `roles`-Join (die Subquery laeuft selbst unter der
  roles-Read-Policy, deshalb greift sie).
  `permissions` und `user_roles` bleiben ohne RLS — `permissions` ist echter System-Katalog und wurde
  dafuer in die ADR-006-Liste in `docs/ARCHITECTURE.md` eingetragen, `user_roles` ist eine echte Luecke
  (siehe unten), die dort als offen dokumentiert und als Backlog-Unit `g-user-roles-rls` angelegt ist.
- **Korrektur am Generator aus Iteration 2 (echter Fund, nicht kosmetisch):** die
  `role_permissions`-Bloecke liefen mit `ON CONFLICT (role_id, permission_id) DO NOTHING`. Die
  Iteration-2-Pruefung dieses Risikos war ein Grep gegen die Migrations-DATEIEN und hat deshalb nichts
  gefunden; der Abgleich gegen die laufende DB zeigt zwei Treffer: `member` hatte
  `schichten:swap:create` und `schichten:swap:read` bereits als Grant, der Katalog fuehrt beide auf
  `own`. Mit DO NOTHING waeren beide auf dem Spaltendefault `all` haengen geblieben — ein `member`
  haette alle Schichttausch-Anfragen des Tenants gesehen und angelegt statt nur die eigenen. Der
  Generator emittiert fuer die role_permissions-Bloecke jetzt
  `DO UPDATE SET scope = EXCLUDED.scope`; der Katalog ist SSOT fuer System-Presets, ein Re-Run
  konvergiert damit statt zu driften. Bloecke (a) permissions und (b) roles bleiben auf DO NOTHING.
  Verifiziert nach dem Lauf: beide Keys stehen auf `own`.
- gate: `migrate up` 256 gruen, `down 1` -> 255 gruen, `up` erneut gruen — Zustand danach
  bit-identisch zum ersten Lauf (8 Rollen, 1179 Grants, 40 davon `own`/`team`, `user_roles`
  unveraendert 12). Die 3 bestehenden Presets und `platform_admin` behalten ihre IDs, nur `color`
  kam dazu; 7 von 8 Rollen tragen eine Farbe (`platform_admin` bewusst nicht, steht nicht im
  FE-Katalog). permissions 212 -> 456, role_permissions 394 -> 1179.
  **RLS-Smoke als `kmuhub_app` (NOSUPERUSER NOBYPASSRLS), Lesepfad:** Tenant A sieht 9 Rollen
  (8 Presets + 1 Fixture-Custom-Rolle), Tenant B sieht 8 und die fremde Custom-Rolle **0**-mal, ihre
  Grants ebenfalls 0-mal. Kein Tenant-Kontext gesetzt (Login vor Tenant-Aufloesung): 8 Rollen und
  1179 Grants weiterhin lesbar — genau der Fall, den das Standardmuster kaputtgemacht haette.
  Aufgeloeste Capability-Menge fuer `member` unter fremdem Tenant: **159**, nicht leer.
  **Schreibpfad:** `DELETE FROM roles WHERE name='admin'` -> 0 Zeilen, `UPDATE` auf ein Preset ->
  0 Zeilen, `DELETE` der fremden Custom-Rolle -> 0 Zeilen, `DELETE` der Preset-Grants -> 0 Zeilen,
  INSERT mit `tenant_id = NULL` (Preset-Faelschung) und INSERT auf fremden Tenant -> beide
  `new row violates row-level security policy`. INSERT auf den eigenen Tenant funktioniert.
  Fixtures danach entfernt, `admin.color` unveraendert.
  `go build -p 2` + `go vet` auf auth/gateway/settings gruen. `go test -count=1` mit gesetztem
  `DATABASE_URL` (Rolle `kmuhub_app`): `internal/auth` ok 6.7s (Laufzeit belegt, dass die DB-Tests
  wirklich liefen und nicht per SkipIfNoDB uebersprungen wurden), `internal/settings` ok,
  `internal/gateway` ok (TestOpenAPIRouteDrift). Keine Go-Datei in dieser Unit geaendert, lint daher
  ohne neuen Angriffspunkt.
- verify vorgaenger: `29172d54` (p1a-seed-generator) — Dateiliste sauber (nur BACKLOG, JOURNAL,
  `desktop/scripts/gen-rbac-seed.mjs`, kein Go/Proto/Route/Migration), Fehlerklassen 1-6 daher n.a.
  Der Script-Lauf ist reproduziert: exit 0, dieselben Zahlen wie im Journal behauptet (282
  permissions; admin 282 / it_admin 104 / hr_admin 71 / manager 205 / member 110 / readonly 74 /
  extern 11). Ein Fund: die DO-NOTHING-Klausel der role_permissions-Bloecke, oben korrigiert.
- offen: **`user_roles` hat weder `tenant_id` noch RLS** — als einzige Tabelle des RBAC-Kerns. Heute
  kein Leck, weil `GetUserRoles`/`GetUserPermissions`/`UserHasPermission` alle ueber eine `user_id`
  aus dem JWT filtern, aber der Datenbank-Backstop fehlt. Unit `g-user-roles-rls` angelegt.
  Zweiter Fund an derselben Datei: `AssignRole` (`internal/auth/postgres_repository.go:157`) sucht die
  Rolle per `WHERE r.name = $2` ohne Tenant-Bedingung. Solange nur System-Presets existieren, ist der
  Name eindeutig — sobald Welle 1b tenant-eigene Rollen zulaesst, kann derselbe Name zweimal
  auftreten und die Subquery liefert zwei Zeilen. Steht in `g-user-roles-rls` mit drin und gehoert
  spaetestens in 1b geprueft.
  Fuer `p1a-resolver` (naechste Unit): die Union-Aufloesung muss auf die 3-segmentigen Katalog-Keys
  filtern; in `permissions` liegen jetzt 456 Zeilen, davon 102 grobe Alt-Keys (`resource` ohne
  Doppelpunkt) — `resource LIKE '%:%'` trennt beide Welten zuverlaessig (geprueft: `name` ist in
  allen 456 Zeilen exakt `resource || ':' || action`).

## Iteration 4 — p1a-resolver — done — 2026-08-01 16:04

- commit: 7b6cfb2e
- gebaut: die Aufloesung der feinen Capabilities, Repo-Query + Service-Union.
  **Repo** (`internal/auth/postgres_repository.go`): `GetEffectivePermissions(ctx, userID)` liefert
  `[]EffectiveGrantRow` — eine Zeile je (Rolle × feine Capability). Join
  `user_roles -> roles LEFT JOIN (role_permissions JOIN permissions WHERE p.resource LIKE '%:%')`.
  Der Filter steckt in der Subquery, nicht in der WHERE-Klausel: sonst faellt eine Rolle, die
  ausschliesslich grobe Alt-Grants haelt, ganz aus dem Ergebnis und taucht in `roles[]` nicht auf,
  obwohl der User sie nachweislich hat. Mit dem LEFT JOIN kommt sie mit leerem `Key` durch.
  `p.resource LIKE '%:%'` trennt die beiden Permission-Welten, die seit 000256 koexistieren
  (feiner Katalog `work:task:edit` -> resource `work:task`, grobe Alt-Keys `files:write` ->
  resource `files`); der Vorgaenger-Journaleintrag hatte das als tragfaehig verifiziert (`name` ist
  in allen 456 Zeilen exakt `resource || ':' || action`).
  **Service** (`internal/auth/effective_permissions.go`, neue Datei): Union nach Regel
  `own < team < all`, breitester Scope gewinnt, `sources` kumuliert **alle** beitragenden Rollen —
  auch die, die den Scope-Vergleich verloren haben. Ausgabe deterministisch sortiert
  (Capabilities nach Key, Rollen nach Name, Sources alphabetisch), beide Slices nie nil.
  Unbekannter Scope faellt auf `own` zurueck statt durchgereicht zu werden: der CHECK auf
  `role_permissions.scope` macht das heute unerreichbar, aber die Fehlerrichtung "zu viel gewaehrt"
  ist die einzige, die man sich hier nicht leisten kann.
  `GetUserRoles`/`GetUserPermissions`/`UserHasPermission` unangetastet — Login und die 812
  bestehenden Gates haengen daran.
- gate: build ok (`./internal/auth/... ./internal/server/... ./internal/gateway/...`) | vet ok |
  lint ok (0 issues, auth + server) | test ok: `./internal/auth/...` 7,1 s und `./internal/server/...`
  0,9 s und `./internal/gateway/` 0,3 s, alle mit `DATABASE_URL` gegen `kmuhub_app` gesetzt.
  Die 11 neuen Tests einzeln nachgefahren, DB-Faelle mit 0,06–0,15 s Laufzeit — also real gelaufen
  und nicht per `SkipIfNoDB` uebersprungen. migration n.a. (kein Schema-Eingriff).
  **RLS-relevant:** zwei der DB-Tests pruefen genau die Policy-Asymmetrie aus 000256 — ein `member`
  eines zweiten Tenants loest zu einer nicht-leeren Menge auf (Presets mit `tenant_id IS NULL`
  bleiben lesbar), und eine tenant-eigene Custom-Rolle mit `fuhrpark:gps:read` ist fuer ihren
  Besitzer sichtbar, unter fremdem Tenant-Kontext dagegen weder in `roles[]` noch in den
  Capabilities. Der Fremd-Fall assertet zusaetzlich, dass die Presets dort **nicht** verschwunden
  sind — zwei Nullen waeren ein kaputter Test statt eines Beweises.
- tests: 5 Unit-Tests ueber `mockRepository` (breitester Scope gewinnt in beiden Reihenfolgen,
  sources kumuliert, unbekannter Scope, Rolle ohne feine Grants bleibt gelistet, leeres Ergebnis
  ist `[]` und nicht nil, Sortierung) + 5 DB-Tests gegen die echten Seeds
  (`effective_permissions_db_test.go`, package `auth_test`). Zahlen aus der laufenden DB:
  extern **11** Keys exakt, admin **354** (Assertion `>250`, plus: jeder Key ist 3-segmentig, kein
  grober Alt-Key leakt), member **117**; `member+extern` loest auf **117** auf, also genau die
  member-Menge, und die drei Keys, die extern enger fuehrt (`documents:file:read` team,
  `work:task:read` team, `work:task:comment` own), stehen auf `all` mit `sources = [extern, member]`.
  Eigene Tenants (`e4fec700-…-0001/-0002`), nicht die geteilten `TenantA/TenantB` — die Kollision
  aus Nachtlauf 1 ist so ausgeschlossen.
  `user_roles` wird per direktem INSERT gesetzt, nicht ueber `AssignRole`: dessen
  `WHERE r.name = $2` ist ohne Tenant-Bedingung (bekannter Fund aus Iteration 3, Unit
  `g-user-roles-rls`) und wuerde den Test an einer Stelle festnageln, die sich in Welle 1b aendert.
- verify vorgaenger: `214d2931` (p1a-migration) — Dateiliste sauber und deckungsgleich mit der Unit
  (2 Migrationsdateien, Generator, ARCHITECTURE.md, BACKLOG, JOURNAL; kein Go, kein Proto, keine
  Route), Fehlerklassen 1/3/4 damit n.a. Die im Journal behaupteten Zahlen gegen die laufende DB
  nachgezaehlt: 456 permissions (354 davon feine), 8 Rollen, 1179 Grants, 40 mit Scope != all,
  12 `user_roles` — stimmt. Die RLS-Form der Migration gegengelesen: Read-Policy mit
  `tenant_id IS NULL OR …`, Write-Policy ohne die NULL-Bedingung; die Begruendung (DELETE wertet nur
  USING aus) traegt. Kein Fund.
- offen: nichts blockierend. Zwei Beobachtungen fuer spaeter:
  (1) Der Fremd-Tenant-Test belegt nebenbei, dass ein fremder Tenant-Kontext die **Preset**-Rollen
  eines fremden Users aufloesen kann — die Custom-Rollen sind dicht, aber der Backstop fehlt, weil
  `user_roles` weder `tenant_id` noch RLS hat. Genau die schon angelegte Unit `g-user-roles-rls`;
  in der Praxis heute nicht erreichbar, weil jeder Aufrufer die `user_id` aus dem eigenen JWT nimmt.
  Sobald `p1a-gateway` die Route `GET /admin/users/{id}/permissions` baut, wird daraus ein echter
  Pfad mit fremder `user_id` — dort MUSS der Handler zusaetzlich pruefen, dass der Ziel-User im
  eigenen Tenant liegt (`GetUserByID` steht unter der users-RLS und leistet genau das).
  Das gehoert in die Notes von `p1a-gateway`, bevor die Route gebaut wird.
  (2) `EffectiveGrantRow` liegt bewusst im Repo-Vertrag (Rohzeilen), die Faltung im Service — falls
  `p1a-grpc` in Versuchung kommt, die Union im Handler zu wiederholen: nicht tun, sie ist fertig.

## Iteration 5 — p1a-grpc — done — 2026-08-01 16:20

- commit: d6f3a1f6
- gebaut: `GetEffectivePermissions` im auth-gRPC-Server (`internal/server/grpc.go`), direkt nach
  `CheckPermission` eingefuegt (gleiches Muster: `uuid.Parse` -> `InvalidArgument`, `mapError` fuer
  Service-Fehler). Leeres `user_id` faellt auf `middleware.GetUserID(ctx)` zurueck — verifiziert, dass
  das trotz echtem Netzwerk-gRPC (nicht in-process) funktioniert: `TenantOutboundUnaryInterceptor`
  (Gateway-Client) haengt `x-user-id`/`x-tenant-id` als Metadata an, `TenantInboundUnaryInterceptor`
  (Service-Server) liest sie zurueck in den Context unter denselben Keys wie die HTTP-Auth-Middleware —
  bestehendes Muster (`grpc_tenant.go`, Sprint-4-Welle-5), nicht neu erfunden. Kein nicht-parsebares
  `user_id` faellt still auf `uuid.Nil` durch — beide Faelle (kaputte UUID im Request, leeres Feld ohne
  Caller-Context) enden bei `InvalidArgument`.
  Ergebnis-Mapping ueber `make([]*authv1.X, len(...))` statt `append` an nil-Slice, damit eine leere
  Antwort ein nicht-nil `[]` bleibt (Resolver-Vertrag aus `p1a-resolver`: "beide Slices nie nil").
  Kein eigener DB-Zugriff im Handler, keine Union-Logik wiederholt — nur Service-Aufruf + Mapping,
  exakt wie in den Notes gefordert.
- gate: build ok (`./internal/auth/... ./internal/gateway/... ./internal/server/... ./cmd/auth/...
  ./cmd/gateway/...`) | vet ok | lint ok (0 issues, auth+gateway+server) | test ok: `./internal/auth/...`
  7,4 s (DB-Tests real gelaufen, nicht uebersprungen), `./internal/server/...` 0,9 s (Mock-Repo, keine
  DB-Abhaengigkeit), `./internal/gateway/` 0,3 s (TestOpenAPIRouteDrift, vorsorglich mitgelaufen obwohl
  keine Route angefasst — die naechste Unit `p1a-gateway` baut die Routen). Alle mit `DATABASE_URL`
  gegen `kmuhub_app` gesetzt. migration n.a. (kein Schema-Eingriff), rls-smoke n.a. (keine Tabelle/Policy
  angefasst).
  5 neue Testfaelle in `TestAuthGRPC_GetEffectivePermissions` (explizite user_id, leere user_id mit
  Caller-Context, keine Rollen -> leere aber nicht-nil Slices, kaputte UUID, leere user_id ohne
  Caller-Context) — alle gegen den echten `auth.Service` ueber `newTestAuthGRPCServer()`, nicht gegen
  einen isolierten Mock des Handlers. `authMockRepo` (in `testhelpers_test.go`, package `server`) bekam
  dafuer ein `effectiveGrants`-Feld; der bisherige Stub-Return (`nil, nil`) war nur ein
  Interface-Erfueller fuer unbeteiligte Tests im selben Package und ist jetzt real nutzbar.
- verify vorgaenger: `7b6cfb2e` (p1a-resolver) — Dateiliste sauber und deckungsgleich mit der Unit
  (Repo-Methode + Service-Datei + drei Testdateien, kein Proto, keine Route, keine Migration),
  Fehlerklassen 1/3/4/5/7 damit n.a. Code gegengelesen: Union-Regel (breitester Scope gewinnt, sources
  kumuliert auch Verlierer-Rollen), `p.resource LIKE '%:%'`-Filter sitzt in der LEFT-JOIN-Subquery statt
  der WHERE-Klausel (rollen mit nur groben Grants bleiben in `roles[]`), Sortierung deterministisch,
  beide Slices explizit als leere (nicht nil) Literale initialisiert. Kein Fund.
- offen: nichts blockierend. `p1a-gateway` (naechste Unit) baut jetzt auf einem RPC auf, der laut Test
  auch bei leerem `user_id` den Caller aus dem Context zieht — die Gateway-Route fuer
  `/auth/me/permissions` kann den Request also mit leerem `user_id` an den auth-Client durchreichen,
  MUSS aber fuer `/admin/users/{id}/permissions` weiterhin den bereits in `p1a-resolver` festgehaltenen
  Tenant-Check via `GetUserByID` selbst machen — der neue RPC prueft NICHT, ob eine explizit uebergebene
  fremde `user_id` zum aufrufenden Tenant gehoert (das ist laut Notes bewusst Aufgabe des Gateway-Handlers,
  nicht des RPC).

## Iteration 6 — p1a-gateway — done — 2026-08-01 16:22

- commit: 0d84142a
- gebaut: Beide Routen in `internal/gateway/route_auth.go`.
  `GET /api/v1/auth/me/permissions` haengt in der bestehenden protected auth-Group (kein neuer Guard —
  jeder eingeloggte User darf seine eigenen Rechte lesen, und ein neuer Guard haette Seed-Pflicht).
  `GET /api/v1/admin/users/{id}/permissions` in einer neuen `/api/v1/admin/users`-Gruppe mit dem
  BESTEHENDEN `RequireRole("admin")` wie die Nachbar-Routen — kein neuer Permission-Key, kein Seed.
  Beide Handler gehen ueber `getAuthClient()`, keine direkt injizierte Service-Instanz.
  Der Tenant-Check auf der Admin-Route ist verdrahtet: `client.GetUser` VOR dem Resolver. Verifiziert,
  dass das traegt — `users` hat seit Migration 000120 die Policy
  `tenant_id = current_tenant_id() OR id = current_user_id() OR is_system_context()`, ein fremder
  Tenant-User faellt dort raus und `GetUserByID` liefert `ErrUserNotFound` -> 404. Ohne diesen Schritt
  waere die Route ein tenant-uebergreifendes Leseleck, weil `user_roles` weder `tenant_id` noch RLS hat
  (Fund aus `p1a-resolver`).
  Wire-Shape in einem eigenen Mapping (`toEffectivePermissionsBody`), nicht `response.Proto`: die
  Antwort traegt `capabilities` als MAP (key -> {scope,sources}), das Proto als repeated list.
  `roles[]` in camelCase (`isSystem`), weil der FE-Typ das so definiert.
- FUND, korrigiert (Wire-Shape, haette das FE still degradiert): der Resolver aus `p1a-resolver`
  fuellte `Sources` mit Rollen-**Namen**. Der FE-Vertrag verlangt Rollen-**IDs** — `rbac-types.ts`
  sagt es im Kommentar ("Role ids (Role.id)"), der MSW-Mock emittiert `sources: [roleId]`
  (`mocks/data/rbac.ts:597`), und `EffectivePermissionsView.tsx:180` loest per
  `roles.find((r) => r.id === src)` gegen die `roles[]`-Liste derselben Response auf. Mit Namen greift
  dort der Fallback `label = src`: Chip ohne Rollenfarbe und ohne i18n-Label, kein Fehler, kein Crash —
  genau die Sorte Defekt, die erst im Pilotbetrieb auffaellt. Die Backlog-Note zu dieser Unit behauptete
  ebenfalls Namen (`["manager","hr_admin"]`) und war damit falsch; sie ist im BACKLOG mit Begruendung
  korrigiert, damit Welle 1b und R-6 nicht darauf aufbauen.
  Umgestellt in `effective_permissions.go` (`g.RoleID.String()` statt `g.RoleName`, Sortierung bleibt,
  Antwort bleibt byte-stabil), Tests in allen drei betroffenen Dateien nachgezogen. Der DB-Test
  `..._OnlyCatalogueKeys` prueft jetzt zusaetzlich, dass JEDE Source gegen `roles[]` aufloest — das ist
  der eigentliche Vertrag und faellt beim naechsten Rueckfall auf Namen sofort auf.
- openapi.yaml im selben Commit: beide Pfade plus vier Schemas (`EffectivePermissionsResponse`,
  `EffectivePermissions`, `EffectiveRole`, `EffectiveCapability`). `capabilities` als
  `additionalProperties`-Map modelliert, `scope` als Enum `own|team|all`. Status-Codes wie der Handler
  sie wirklich liefert: me -> 200/401; admin -> 200/400/401/403/404. Der `?base=1`-Query-Param, den
  `rbac-client.ts` schickt, ist dokumentiert und als "aktuell akzeptiert und ignoriert" beschrieben —
  er gehoert zu R-6 (per-user Overrides), die serverseitig nicht existieren; `lean:`-Marker mit
  Upgrade-Trigger sitzt am Handler.
- gate: build ok | vet ok | lint ok (0 issues, auth+gateway+server) | test ok:
  `./internal/gateway/` 0,34 s (TestOpenAPIRouteDrift gruen — beide Richtungen, die Routen sind also
  registriert UND dokumentiert), `./internal/auth/...` 9,2 s, `./internal/server/...` 1,0 s.
  Alle mit `DATABASE_URL` gegen `kmuhub_app`. Gegengeprueft, dass die DB-Tests wirklich liefen und
  nicht `SkipIfNoDB` durchrutschte: `-run TestEffectivePermissions_DB -v` -> 5× PASS, 0× SKIP.
  migration n.a. (kein Schema-Eingriff), rls-smoke n.a. (keine Tabelle/Policy angefasst — die
  Policy-Verifikation ist die naechste Unit `p1a-rls-verify`).
  4 neue Gateway-Tests in `route_auth_permissions_test.go`: 503 ohne Backend, 400 bei kaputter UUID,
  und zwei, die die Wire-Shape als JSON-String festnageln (Map-Form + camelCase `isSystem`; leere
  Antwort ist `{"roles":[],"capabilities":{}}`, nil-Sources marshallt als `[]` nicht `null`).
- verify vorgaenger: `d6f3a1f6` (p1a-grpc) — Dateiliste deckungsgleich mit der Unit (gRPC-Handler +
  zwei Testdateien + BACKLOG, kein Proto, keine Route, keine Migration), Fehlerklassen 3/4/5 damit n.a.
  Handler gegengelesen: thin (Parse -> Service -> Mapping), kein DB-Zugriff, keine wiederholte
  Union-Logik, `uuid.Parse`-Fehler -> `InvalidArgument` statt stillem `uuid.Nil`, Slices ueber
  `make(..., len)` also nie nil. Kein Fund im Commit selbst — der Sources-Defekt oben stammt aus
  `7b6cfb2e` (p1a-resolver) und war in dessen eigenem Verify nicht auffindbar, weil er sich erst gegen
  den FE-Typ zeigt und dieser erst in dieser Unit zum Pruefgegenstand wurde.
- offen: nichts blockierend.
  (1) `p1a-rls-verify` (naechste Unit) hat jetzt einen echten Endpunkt zum Gegenpruefen. Die dort
  geforderte Aussage "member bekommt eine nicht-leere Capability-Menge" ist auf Resolver-Ebene bereits
  durch `TestEffectivePermissions_DB_UnionAcrossRoles` belegt; was noch fehlt, ist der explizite
  RLS-Smoke auf `roles`/`role_permissions` als `kmuhub_app` mit dokumentierten Zeilenzahlen.
  (2) R-6 (per-user Overrides) bleibt die einzige Luecke gegenueber dem FE-Typ: `hasOverrides` und
  `deniedByOverride` sind dort optional und werden bewusst nicht geliefert, `?base=1` ist deshalb
  wirkungslos. Sobald die Override-Tabelle existiert, sind das drei Stellen: Resolver (Overrides
  anwenden + verdraengte Keys sammeln), Mapping (`hasOverrides`/`deniedByOverride` fuellen), Handler
  (auf `base` verzweigen).
  (3) `g-user-roles-rls` bleibt sinnvoll als Backstop, ist durch den `GetUser`-Vorschalt-Check auf der
  Admin-Route aber nicht mehr akut.

## Iteration 7 — p1a-rls-verify — done — 2026-08-01 (Nachtlauf 3)

- commit: (nur BACKLOG.yml, kein Quellcode — reine Verifikations-Unit)
- gebaut: nichts Neues. Abschluss-Verifikation von Block A (RBAC Phase 1 Welle 1a): RLS-Smoke gegen
  `roles`/`role_permissions` als `kmuhub_app` plus Bestaetigung, dass die geforderte DB-Test-Aussage
  bereits existiert.
- RLS-Smoke (manuell, Transaktion mit `ROLLBACK`, keine Datenspuren):
  Das Standard-Smoke-Muster aus `GATE-COMMANDS.md` passt hier nicht 1:1 — die Policy lautet
  `tenant_id IS NULL OR tenant_id = current_tenant_id() OR is_system_context()`, Presets (tenant_id
  NULL) sind also bewusst unter JEDEM Tenant sichtbar. Getestet wurde deshalb zusaetzlich mit einer
  temporaeren Custom-Rolle unter Tenant A (`aaaa0000-...0001`), die vor dem Commit zurueckgerollt wurde:
  - Presets (superuser, sanity): 8 Zeilen (`tenant_id IS NULL`)
  - Eigener Tenant (A) unter `kmuhub_app`: `roles` liefert 9 (8 Presets + 1 eigene Custom-Rolle),
    `role_permissions` fuer die Custom-Rolle liefert 1
  - Fremder Tenant (B) unter `kmuhub_app`: `roles` liefert 8 (nur Presets, Custom-Rolle ausgeblendet),
    direkte Abfrage der Custom-Rolle liefert 0, ihre `role_permissions`-Zeile liefert 0
  - Write-Policy: `DELETE FROM roles WHERE name='member' AND tenant_id IS NULL` unter fremdem
    Tenant-Context (B) betrifft 0 Zeilen, Preset danach weiterhin vorhanden (1) — die
    Read/Write-Split-Policy aus der Migration haelt wie begruendet.
  Ergebnis: Presets tenant-uebergreifend lesbar (Design-Absicht), Custom-Rollen strikt
  tenant-isoliert (Lesen UND Schreiben), keine der beiden Nullen ein kaputter Test.
- DB-Test "member bekommt nicht-leere Capabilities": existiert bereits, nicht neu gebaut.
  `TestEffectivePermissions_DB_UnionAcrossRoles` (`require.NotEmpty(t, memberGot.Capabilities, ...)`)
  und `TestEffectivePermissions_DB_PresetsVisibleUnderForeignTenant` (member-Rolle unter fremdem
  Tenant, `assert.NotEmpty(t, got.Capabilities, ...)`) belegen das bereits aus `p1a-resolver`/
  `p1a-gateway`. Zusaetzlich per SQL bestaetigt: `role_permissions` fuer die Rolle `member` traegt 159
  Zeilen (`admin` 454, `manager` 305, `it_admin` 104, `readonly` 74, `hr_admin` 71, `extern` 11,
  `platform_admin` 1 — Summe 1179), also strukturell nicht leer.
- gate: build ok (`./internal/auth/... ./internal/gateway/... ./cmd/auth/... ./cmd/gateway/...`) |
  vet ok | lint ok (0 issues) |
  test ok: `./internal/auth/...` 9,02 s, `./internal/gateway/` 0,35 s (inkl.
  `TestOpenAPIRouteDrift` PASS). Gezielt `-run TestEffectivePermissions_DB -v`: 5× PASS, 0× SKIP.
  Alle mit `DATABASE_URL` gegen `kmuhub_app` (lokaler Migrationskopf 256 = Repo-Kopf, keine Migration
  in dieser Unit noetig). migration n.a. (keine Schema-Aenderung in dieser Unit).
- verify vorgaenger: `0d84142a` (p1a-gateway) — Dateiliste passt zur Unit (Gateway-Routen + Mapping +
  zwei Testdateien, `effective_permissions.go`-Korrektur, openapi.yaml, kein Proto-Diff ohne Regen).
  Gateway-Handler gegengelesen: beide Routen gehen ueber `getAuthClient()`, kein direkter
  Service-Zugriff (Fehlerklasse 1 n.a.). `/auth/me/permissions` ohne neuen Guard (korrekt, jeder
  eingeloggte User darf eigene Rechte lesen), `/admin/users/{id}/permissions` nutzt den bestehenden
  `RequireRole("admin")` (Fehlerklasse 4 n.a., kein neuer Key). Tenant-Check via `client.GetUser` vor
  dem Resolver-Aufruf verifiziert vorhanden. Wire-Shape (Fehlerklasse 6) gegen `rbac-types.ts`
  gegengelesen: `capabilities` als Map, `roles[].isSystem` camelCase, leere Antwort marshallt als
  `{"roles":[],"capabilities":{}}` nicht `null`. Sources-Korrektur (Rollen-IDs statt Namen) im Diff
  bestaetigt (`g.RoleID.String()`). openapi.yaml traegt beide Pfade plus vier Schemas mit den Handler-
  Statuscodes (me: 200/401; admin: 200/400/401/403/404), `TestOpenAPIRouteDrift` bestaetigt gruen.
  Kein Fund.
- offen: **Block A (RBAC Phase 1 Welle 1a) ist damit vollstaendig abgeschlossen** — alle sieben Units
  (`p1a-proto` bis `p1a-rls-verify`) `done`. Welle 1b (Rollen-CRUD) und Phase 2 (Guard-Verfeinerung je
  Modul) bleiben wie im Backlog vermerkt gesperrt bzw. warten auf `p2-guard-compat`. R-6 (per-user
  Overrides) bleibt offen, siehe Iteration 6. Naechste ziehbare Unit laut Backlog-Reihenfolge:
  `w5-ubl-generator` (Block C, E-Rechnung-Ausgang, keine deps).

## Iteration 8 — w5-ubl-generator — done — 2026-08-01 (Nachtlauf 3)

- commits: `d321f482` (fix, Test-Isolation) + Feature-Commit dieser Iteration
- gebaut: XRechnung-UBL-2.1-Ausgang (EN 16931). Zwei neue Dateien im einvoice-Paket:
  - `generator_doc.go` — neutrales Zwischenmodell `invoiceDoc` + `buildInvoiceDoc`. Traegt die
    gesamte Betrags- und Steuergruppenlogik, damit UBL und der kommende CII-Weg sich ueber die
    Zahlen nie uneinig werden koennen (so vom Backlog fuer `w5-cii-generator` gefordert).
  - `generator_ubl.go` — `GenerateUBL(invoice, settings, buyerReference)`.
  - `errors.go` um `ErrGenerateFailed` und `ErrTotalsMismatch` ergaenzt.
- ABWEICHUNG VON DER UNIT-NOTE (bewusst, technisch erzwungen): Die Note verlangte, die
  UBL-Structs aus `parser.go:318ff` fuer die Schreibrichtung wiederzuverwenden. Das geht mit
  `encoding/xml` nicht. Die Lesestructs tragen `xml:"ID"` **ohne** Namespace und matchen damit
  jeden Absender-Namespace; XRechnung schreiben verlangt die Praefixe `cbc:`/`cac:` woertlich.
  Ein Feld kann nur eines von beidem: mit namespace-qualifiziertem Tag wuerde der Eingangs-Parser
  Dokumente fremder Namespaces ablehnen (Regression im Bestand), mit Praefix-Tag matcht er beim
  Decoden nie. Also getrennte Write-Structs, mit dem Grund als Kommentarblock im Code, und der
  Roundtrip-Test als Klammer. Vorher verifiziert (Scratch-Programm): Go schreibt bei
  `xml:"cbc:ID"` tatsaechlich `<cbc:ID>`, escaped automatisch, und der namespace-agnostische
  Parser liest es zurueck.
- Format gegen `testdata/ubl_minimal.xml` gespiegelt (Default-NS am Root + cac/cbc-Praefixe),
  Element-Reihenfolge nach den UBL-2.1-XSD-Sequences. Erzeugtes Dokument einmal vollstaendig
  ausgegeben und Zeile fuer Zeile gegengelesen, nicht nur die Assertions geglaubt.
- Entscheidungen, die nicht offensichtlich waren:
  (1) **Summen werden aus den Zeilen neu berechnet**, nicht aus der Rechnung uebernommen — so
      halten BR-CO-10/13/14/15 und BR-S-08 immer. Damit dabei aber nie still ein anderer Betrag
      rausgeht als auf dem PDF steht, prueft `assertTotalsMatch` gegen die gespeicherten Summen
      und bricht bei mehr als 1 Cent Abweichung mit `ErrTotalsMismatch` ab.
  (2) **Kaeuferadresse:** `models.Invoice` speichert sie als einen Freitext, XRechnung macht
      BT-50/52/53 zur Pflicht. `splitPostalAddress` trennt die uebliche Form "Strasse\nPLZ Ort";
      was nicht passt, bleibt unveraendert in der Strassenzeile. Ohne das waere JEDES erzeugte
      Dokument bei oeffentlichen Empfaengern unzustellbar — deshalb kein Nice-to-have.
      `lean:`-Marker gesetzt, Trigger: sobald `finance_invoices` die Adressteile getrennt fuehrt.
  (3) Steuerkategorien: `S` / `Z` bei 0 % / `AE` reverse charge / `E` Kleinunternehmer, bei den
      steuerfreien Modi Zeilensatz auf 0 (BR-AE-05/BR-E-05) und Befreiungsgrund gesetzt.
  (4) BT-10 (Leitweg-ID) als Parameter — kein Feld im Datenmodell. Bei leerem Wert entfaellt das
      Element. Die Pflichtfeld-Meldung dazu gehoert in `w5-en16931-validation`.
- keine neue Dependency (`go.mod`/`go.sum` unveraendert), reine Stdlib `encoding/xml`.
- FUNDE IM BESTAND (beide ins BACKLOG uebertragen, weil sie den Zuschnitt der Folge-Units aendern):
  (a) **Ein CII-Generator existiert bereits**: `pdf.GenerateZUGFeRDXML`
      (`internal/biz/pdf/zugferd.go:27`), komplett mit EN16931-Profil-URN und BG-23-Aufschluesselung.
      `w5-cii-generator` ist damit **Zusammenfuehrung, kein Neubau** — Unit-Notes entsprechend
      umgeschrieben. Zwei Abweichungen dort sind beim Zusammenfuehren zu bereinigen: Steuerbasis
      aus `Quantity*UnitPrice` gegen Zeilenbetrag aus `LineTotal` (verletzt BR-S-08, sobald sie
      divergieren) und CategoryCode `S` auch bei 0 %.
  (b) `pdf.EmbedZUGFeRDXML` **verschluckt jeden Fehler** und liefert das PDF ohne Anhang zurueck;
      der einzige Test dafuer skippt still, weil das Stub-PDF fuer pdfcpu unlesbar ist. Die
      Einbettung ist faktisch ungetestet. In die Notes von `w5-pdfa3-embed` uebertragen, samt der
      Info, dass pdfcpu v0.6.0 (`api.AddAttachments`) bereits im Modulgraph liegt — fuer den
      Anhang-Teil braucht die Unit also keine neue Dependency.
- NEBENFIX (eigener Commit `d321f482`): `TestTenantIsolation_IncomingInvoices` importierte das
  Fixture-XML mit konstanter Rechnungsnummer in die geteilten `testutil.TenantA/TenantB` und lief
  deshalb nur beim ersten Mal pro Datenbank gruen — beim zweiten Lauf schlug der
  Duplikat-Check zu statt der Isolation. Jetzt frische Tenants pro Lauf. Der Fehlschlag reproduziert
  auf unveraendertem Tree (per `git stash` verifiziert, bevor ich ihn angefasst habe) und war
  vorher unsichtbar, weil das Paket ohne `DATABASE_URL` still skippt.
- gate: build ok (`go build -p 2 ./...`, ganzes Backend) | vet ok | lint ok (0 issues) |
  gofmt ok (nur eigene Dateien angefasst; `parser.go`/`service_test.go` sind im Bestand
  unformatiert und blieben unberuehrt) |
  test ok: `./internal/biz/einvoice/...` zweimal hintereinander gruen (Wiederholbarkeit nach dem
  Isolationsfix belegt), zusaetzlich `./internal/biz/pdf/...` und `./internal/biz/invoice/...` gruen.
  Alles mit `DATABASE_URL` gegen `kmuhub_app`. 30 neue Testfaelle, 0 SKIP unter den neuen;
  einziger SKIP im Paket ist der bestehende `TestExtractXMLFromPDF_HappyPath` (Fund (b) oben).
  migration n.a. (keine Schema-Aenderung). openapi n.a. (keine Route — die kommt in `w5-route`).
- verify vorgaenger: `f069b086` (p1a-rls-verify) — Commit aendert ausschliesslich `BACKLOG.yml`
  (Statuswechsel auf `done`), kein Quellcode. Deckt sich mit dem Journaleintrag, der die Unit als
  reine Verifikation ohne Codeaenderung ausweist; der geforderte DB-Test existierte bereits aus
  `p1a-resolver`/`p1a-gateway`. Der Prefix `test(auth):` ohne Testdatei im Diff ist unsauber, aber
  kein Fund im Sinne der sechs Fehlerklassen — kein Stub, kein Proto-Diff ohne Regen, kein
  Guard ohne Seed, keine Service-Direktinjektion, keine Wire-Shape-Aenderung. Kein Fund.
- offen: `w5-cii-generator` als naechste Unit (jetzt Zusammenfuehrung, siehe Fund (a)). Die
  Ausgangsroute (`w5-route`) haengt weiterhin hinter Validierung und PDF-Entscheidung. Fuer die
  Route offen: woher die Leitweg-ID kommt — sie hat kein Feld in `finance_invoices` und muss
  entweder als Request-Parameter durchgereicht oder am Kontakt/Tenant hinterlegt werden.

## Iteration 9 — w5-cii-generator — done — 2026-08-01 (Nachtlauf 3)

- commit: Feature-Commit dieser Iteration
- gebaut: `internal/biz/einvoice/generator_cii.go` — `GenerateCII(invoice, settings,
  buyerReference)` rendert ZUGFeRD 2.1 / Factur-X (CII, Profil EN16931) ueber `buildInvoiceDoc`,
  also ueber dieselbe Betrags- und Steuergruppenlogik wie `GenerateUBL`. Write-Structs mit
  `rsm:`/`ram:`/`udt:`-Praefixen, getrennt von den Lesestructs in `parser.go` — identische
  Begruendung wie beim UBL-Writer (der Parser matcht namespace-agnostisch, das Schreiben braucht
  woertliche Praefixe; ein Feld kann nur eines von beidem).
- ZUSAMMENFUEHRUNG statt Neubau, wie die Unit-Note es nach dem Fund aus Iteration 8 verlangte:
  `pdf.GenerateZUGFeRDXML` ist jetzt ein Delegator auf `einvoice.GenerateCII`. Der alte
  `fmt.Sprintf`-Template-Writer samt `buildHeaderTradeTax`, `xmlEscape` und `countryCode` ist
  entfallen (rund 200 Zeilen). Die beiden §14-UStG-Vorbedingungen des PDF-Pfades — Faelligkeits-
  datum und vollstaendiger Ausstellerblock (`ValidateCompanySettingsForPDF`) — bleiben in
  `GenerateZUGFeRDXML`, weil sie nur fuer die PDF-Auslieferung gelten, nicht fuer reines XML.
- die zwei Abweichungen aus der Unit-Note sind damit beseitigt, jede mit eigenem Test:
  (a) BR-S-08: Der Bestand nahm `Quantity*UnitPrice` als Steuerbasis, schrieb als Zeilenbetrag
      aber `LineTotal`. Bei einer rabattierten Position (3 x 100,00 -> 270,00) ergab das ein
      Dokument, dessen Kategorie-Basis 300,00 und dessen Zeilensumme 270,00 sagte.
      `TestGenerateCII_TaxBasisMatchesLineAmounts` faehrt genau diesen Fall und prueft nach dem
      Roundtrip `TaxBreakdown[0].TaxableNet == LineItems[0].LineTotal`.
  (b) CategoryCode `S` auch bei 0 % -> jetzt `Z` (`TestGenerateCII_ZeroRatedLineUsesCategoryZ`).
- DRITTER FUND beim Bau, nicht in der Note: der alte Writer schrieb die
  `IncludedSupplyChainTradeLineItem`-Bloecke ans ENDE von `SupplyChainTradeTransaction`. Die
  CII-XSD-Sequenz verlangt sie VOR den drei Header-Gruppen. Das parst hier durch und faellt
  erst beim Empfaenger in der Schema-Validierung auf — also unsichtbar, bis eine Rechnung
  draussen abgelehnt wird. `TestGenerateCII_LineItemsPrecedeHeaderGroups` haelt die Reihenfolge
  fest. Erzeugtes Dokument einmal vollstaendig ausgegeben und gegengelesen, nicht nur die
  Assertions geglaubt.
- IMPORT-ZYKLUS (der eigentliche Umbau-Aufwand): `pdf` importiert jetzt `einvoice`. Die
  einvoice-Tests lagen alle in `package einvoice` und zwei davon importierten `pdf` — das
  schliesst im Test-Build einen Zyklus. Aufgeloest ohne Testabdeckung zu verlieren:
  `parser_test.go` erzeugt sein CII jetzt direkt ueber `GenerateCII` (der `pdf`-Import faellt
  weg), `pdf_extract_test.go` liegt in `package einvoice_test`, weil es `EmbedZUGFeRDXML`
  wirklich braucht. Die Richtung `pdf -> einvoice` ist damit fest; als Randbedingung in die
  Notes von `w5-pdfa3-embed` uebernommen.
- VERHALTENSAENDERUNG AM PDF-PFAD, bewusst: `buildInvoiceDoc` lehnt eine Rechnung ohne Positionen
  ab (BG-25). Der Bestand erzeugte dafuer ein zeilenloses Dokument. Zwei bestehende Tests
  (`_CurrencyFromInvoice`, `_CurrencyDefaultsToEUR`) nutzten positionslose Rechnungen, um die
  Waehrung zu pruefen — die haben jetzt eine Position, und `TestGenerateZUGFeRDXML_NoLineItems`
  haelt die Verschaerfung ausdruecklich fest. Ebenso erbt der PDF-Pfad `assertTotalsMatch`.
  Beides ist die gewollte Folge der Zusammenfuehrung, kein Kollateralschaden.
- keine neue Dependency (`go.mod`/`go.sum` unveraendert), reine Stdlib `encoding/xml`.
- FUNDE IM BESTAND (in die Notes der Folge-Units uebertragen):
  (a) `pdf.GenerateZUGFeRDInvoicePDF` hat **keinen einzigen Test** — nicht nur die Einbettung ist
      ungetestet, der ganze PDF-plus-XML-Pfad ist es. -> `w5-pdfa3-embed`.
  (b) `server/biz_grpc.go:1365` faengt JEDEN Generierungsfehler ab und liefert still das PDF ohne
      XML aus. Durch die BG-25-Verschaerfung trifft das jetzt auch positionslose Rechnungen: der
      Aufrufer bekommt ein PDF und kann nicht erkennen, dass kein E-Rechnungs-XML entstanden ist.
      Fuer `w5-route` festgehalten (dort 422 statt stiller Degradation), fuer `w5-pdfa3-embed` als
      Mit-zu-aendernder Aufrufer.
  (c) Fuer `w5-en16931-validation`: die Validierung gehoert auf `invoiceDoc`, nicht auf
      `models.Invoice` — dort gilt sie fuer beide Formate zugleich. In die Notes uebernommen.
- gate: build ok (`go build -p 2 ./...`, ganzes Backend) | vet ok (`./...`) | lint ok
  (golangci-lint auf einvoice + pdf: 0 issues) | gofmt ok (alle sechs beruehrten Dateien, gegen
  LF-normalisierten Inhalt geprueft — `gofmt -l` meldet auf diesem Windows-Tree sonst reine
  CRLF-Treffer; der Index ist LF) |
  test ok: `./internal/biz/...` und `./internal/server/...` vollstaendig gruen, mit
  `DATABASE_URL` gegen `kmuhub_app`. 66 Testfaelle im einvoice-Paket, 15 davon neu; einziger SKIP
  im Lauf ist der bestehende `TestExtractXMLFromPDF_HappyPath` (Fund (b) aus Iteration 8).
  migration n.a. (keine Schema-Aenderung). openapi n.a. (keine Route — die kommt in `w5-route`).
- verify vorgaenger: `58b27948` (w5-ubl-generator) gegen die sechs Fehlerklassen geprueft.
  Diff umfasst nur `errors.go`, `generator_doc.go`, `generator_ubl.go`, `generator_ubl_test.go`
  plus Loop-Dateien: keine Migration, keine Route, kein Proto, kein neuer Guard, kein DB-Zugriff,
  kein gRPC-Layer betroffen — die vier darauf zielenden Klassen sind hier gegenstandslos. Kein
  Stub und kein `Unimplemented` im Diff; `buildInvoiceDoc`/`renderUBL` sind vollstaendig
  implementiert und durch 30 Testfaelle gedeckt. Die Wire-Shape-Klasse trifft nicht zu (kein
  JSON-Handler), das erzeugte XML habe ich in dieser Iteration ohnehin gegen `ParseUBL` und gegen
  den CII-Zwilling gegengeprueft. Kein Fund. Beim Weiterbauen bestaetigt: `buildInvoiceDoc` traegt
  die Betragslogik tatsaechlich vollstaendig, der CII-Writer brauchte keine eigene Zahl.
- offen: `w5-en16931-validation` als naechste Unit (Ansatzpunkt `invoiceDoc`, siehe oben).
  Weiterhin offen fuer `w5-route`: woher die Leitweg-ID (BT-10) kommt — sie hat kein Feld in
  `finance_invoices`.

## Iteration 10 — w5-en16931-validation — done — 2026-08-01 (Nachtlauf 3)

- Validierung auf `invoiceDoc` gelegt (neue `internal/biz/einvoice/validation.go`), wie in der
  Unit-Note vorgegeben: dort gilt sie fuer XRechnung und ZUGFeRD zugleich, weil beide Formate durch
  dasselbe neutrale Modell laufen. Auf `models.Invoice` haette man die Ableitungen (Steuerkategorien,
  BG-23-Gruppen, Summen) ein zweites Mal nachbauen muessen — genau das, was `w5-cii-generator`
  gerade beseitigt hat.
- `*ValidationError` traegt ALLE Verstoesse (`Violations[]{Rule, Term, Message}`) und unwrappt auf
  `ErrValidationFailed`, also `errors.Is` fuer den 422-Fall und `errors.As` fuer die Liste. Kein
  Abbruch beim ersten Fund — eine E-Rechnung wird beim Empfaenger abgelehnt, der Kunde erfaehrt es
  Wochen spaeter ueber seine Buchhaltung, und pro Runde ein Feld nachzureichen ist der eigentliche
  Schaden. `TestValidate_ReportsEveryViolationAtOnce` faehrt sechs Verstoesse in einem Durchlauf.
- Geprueft wird nur, was `buildInvoiceDoc` NICHT schon ablehnt (BT-1, BT-2, BG-25, Summen-Drift,
  unparsebare Positionen sind dort und bleiben dort): Verkaeufer-Name/Land (BR-06/BR-09),
  Kaeufer-Name/Land (BR-07/BR-11), Steueridentifikation des Verkaeufers (BR-S-02/BR-E-02/BR-Z-02/
  BR-AE-02 je nach Kategorie), Kaeufer-USt-ID bei Reverse Charge (BR-AE-03), Positionsbezeichnung
  (BR-25), negativer Einzelpreis (BR-27), und BR-CO-25 (Faelligkeitsdatum ODER Zahlungsbedingungen —
  nicht beides, sonst wuerde eine Rechnung abgelehnt, die das Zahlungsziel in Worten nennt).
- ZWEI PROFILE, und die Trennung ist der eigentliche Entwurfsentscheid. `ProfileEN16931` ist die
  Untergrenze und wird von `GenerateUBL`/`GenerateCII` selbst erzwungen — ab jetzt verlaesst kein
  halb gueltiges Dokument den Generator. `ProfileXRechnung` (BR-DE-15 Leitweg-ID, BR-DE-1
  Zahlungsangaben, zerlegte Postanschriften BT-35/37/38 + BT-50/52/53) liegt NICHT im Generator:
  ob der Empfaenger ein oeffentlicher Auftraggeber ist, weiss nur der Aufrufer. Haette ich BR-DE-15
  hart in `GenerateUBL` gezogen, waere jede B2B-XRechnung unerzeugbar geworden — der bestehende
  `TestGenerateUBL_OmitsBuyerReferenceWhenEmpty` haelt genau dieses Verhalten fest, und er hat
  recht. Deshalb exportiert `Validate(invoice, settings, buyerReference, profile)` die strengere
  Stufe; `w5-route` ruft sie, wenn eine Leitweg-ID im Request mitkommt. In die Notes uebertragen.
- FUND BEIM BAUEN, mit behoben (sonst waere die eigene Regel eine Sackgasse): Das Dokument schrieb
  nur BT-31 (USt-ID). `ValidateCompanySettingsForPDF` akzeptiert aber seit jeher Steuernummer ODER
  USt-ID, und ein Kleinunternehmer hat per Definition keine USt-ID. Ergebnis: dessen ZUGFeRD-
  Rechnungen verletzten BR-E-02 immer, und die Meldung "USt-ID fehlt" waere fuer ihn nicht
  behebbar gewesen. `docParty.TaxRegID` traegt jetzt BT-32 (Steuernummer) als Rueckfall, in UBL als
  TaxScheme `FC`, in CII als `schemeID="FC"`. Bewusst nur EINES von beiden: der Eingangs-Parser
  liest ein einzelnes `SpecifiedTaxRegistration`/`PartyTaxScheme` und wuerde bei zwei Elementen das
  letzte nehmen, also die Steuernummer fuer die USt-ID halten. `TestGenerate_VATIDWinsOverTaxNumber`
  haelt das inklusive Parser-Roundtrip fest.
- VERHALTENSAENDERUNG AM PDF-PFAD, bewusst und gleicher Art wie in Iteration 9: `GenerateZUGFeRDXML`
  erbt die Validierung. Vier bestehende `pdf`-Tests fielen, alle vier zu Recht — ihre Fixtures
  hatten keinen Kundennamen (BT-44), der Reverse-Charge-Fall zusaetzlich keine Kaeufer-USt-ID
  (BR-AE-03). Fixtures ergaenzt statt Regel entschaerft: eine Rechnung ohne Kundennamen ist ein
  Datendefekt, kein Testfall. `server/biz_grpc.go:1365` verschluckt den neuen Fehler weiterhin still
  und liefert das PDF ohne XML (nur `slog.Error`) — nicht angefasst, das ist `w5-pdfa3-embed`; der
  Nachtrag steht dort in den Notes, samt dem Hinweis, dass jetzt die Verstossliste hochzureichen ist.
- Kommentar in `buildBuyerParty` korrigiert: er versprach, diese Unit melde BT-55. Tut sie nicht und
  kann sie nicht — `isoCountryCode` liefert immer einen Wert, das geratene Kaeuferland hinterlaesst
  keine Luecke. Steht jetzt als `lean:`-Marker mit Trigger "sobald Rechnungen die Grenze
  ueberschreiten". Zweiter `lean:`-Marker an `xrechnungViolations` fuer BR-DE-2/5..7 (Kontaktdaten
  des Verkaeufers) — das Datenmodell hat keine Kontaktfelder, also gibt es nichts zu pruefen;
  Trigger "sobald ein Kunde einen oeffentlichen Auftraggeber beliefert, der sie erzwingt".
- keine neue Dependency, keine Migration, keine Route, kein Proto, kein Guard.
- gate: build ok (`./internal/biz/... ./internal/server/... ./cmd/gateway/...`) | vet ok |
  lint ok (golangci-lint auf einvoice + pdf: 0 issues) | gofmt ok (alle sieben beruehrten Dateien
  gegen LF-normalisierten Inhalt geprueft; zwei echte Treffer gefunden und behoben — Struct-Feld-
  Ausrichtung, keine CRLF-Artefakte) | test ok: `./internal/biz/...` und `./internal/server/...`
  vollstaendig gruen mit `DATABASE_URL` gegen `kmuhub_app`, 12 neue Testfaelle. openapi n.a.,
  migration n.a., gateway n.a. (keine Route).
- verify vorgaenger: `3b45e1e1` (w5-cii-generator) gegen die sechs Fehlerklassen geprueft. Diff
  umfasst `generator_cii.go` + Test, `parser_test.go`, `pdf_extract_test.go`, `pdf/zugferd.go` +
  Test, plus Loop-Dateien — keine Migration, keine Route, kein Proto, kein Guard, kein JSON-Handler,
  kein DB-Zugriff, kein gRPC-Layer. Kein `Unimplemented`, kein Stub. Gezielt nachgesehen, ob der
  Umbau die Delivery-Gruppe verliert: der alte Writer schrieb `ApplicableHeaderTradeDelivery` immer
  mit Rueckfall auf das Rechnungsdatum, der neue schreibt sie unter `if !doc.DeliveryDate.IsZero()`.
  Kein Fund — `buildInvoiceDoc` setzt denselben Rueckfall (`generator_doc.go:127`), und da BT-2
  vorher erzwungen wird, ist das Feld nie leer. Beim Weiterbauen bestaetigt: `buildInvoiceDoc` war
  der richtige Ansatzpunkt, die Validierung brauchte keine einzige Zahl neu.
- offen: `w5-pdfa3-embed` als naechste Unit. Weiterhin offen fuer `w5-route`: woher die Leitweg-ID
  (BT-10) kommt — sie hat kein Feld in `finance_invoices`. Mit den zwei Profilen ist die Frage jetzt
  scharf gestellt: Leitweg-ID mitgeliefert = oeffentlicher Auftraggeber = strenges Profil.

## Iteration 11 — w5-pdfa3-embed — done — 2026-08-01 (Nachtlauf 3)
- commit: c12467c8
- ENTSCHEIDUNG (die Unit verlangte sie ausdruecklich): **kein PDF/A-3b.** Nachgesehen statt geraten —
  maroto/v2 v2.3.3 zeichnet mit den nicht eingebetteten Standard-14-Fonts (Einbettung nur ueber
  `AddUTF8Font` + TTF-Asset im Repo), pdfcpu v0.6.0 kennt `AFRelationship` und `OutputIntent`
  ausschliesslich im Validierungspfad (`pkg/pdfcpu/validate/`) und hat keinen XMP-Writer. PDF/A-3b
  braucht alle drei: eingebettete Fonts, sRGB-OutputIntent, XMP mit pdfaid- und Factur-X-Schema. Das
  waere eine neue Dependency plus ein ICC-Profil-Asset gewesen — beides ausgeschlossen.
  Der Ausschlag gab aber nicht der Aufwand: XMP-Konformitaet zu behaupten, ohne sie zu erfuellen,
  waere schlechter als sie wegzulassen. Ein pruefender Empfaenger lehnt eine Datei ab, die ueber den
  eigenen Standard luegt; die Datei ohne Anspruch nimmt er an. Als `lean:`-Marker an
  `EmbedZUGFeRDXML` mit Trigger "sobald die Empfangssoftware eines Kunden PDF/A-3b verlangt".
- Statt Konformitaets-Etikett die Ebene gebaut, die ueber Auffindbarkeit entscheidet — und dort lag
  der eigentliche Fund: **der Anhang hiess `factur-x_RE-2026-0001_20260801.xml`.** Die Factur-X-Spec
  fixiert `factur-x.xml`, Empfangssoftware matcht literal. Die Rechnung trug das XML also mit sich
  und kam beim Empfaenger trotzdem als Rechnung ohne strukturierte Daten an. Dazu fehlten `/AF` im
  Catalog und `/AFRelationship /Alternative` an der Filespec — ohne die ist der Anhang eine Beilage,
  keine E-Rechnung. Alle drei sitzen jetzt (`declareFacturXAssociatedFile`), plus `/Subtype
  /text#2Fxml` am Stream (`#2F` ist die Escape-Form des Solidus im PDF-Name-Token; pdfcpu reicht
  Name-Bytes unveraendert durch, `Name.Value()` dekodiert zurueck — im Test als `text/xml` geprueft).
- Kein Temp-File mehr: `NewEmbeddedStreamDict` nimmt einen `io.Reader`, also `bytes.NewReader`. Damit
  fallen `MkdirTemp` und `WriteFile` als Fehlerquellen ersatzlos weg. Keine neue Dependency (go.mod
  unveraendert), pdfcpu war laengst im Graph.
- **Stille Degradierung entfernt, an beiden Stellen.** `EmbedZUGFeRDXML` gab bei jedem Fehler das PDF
  OHNE Anhang zurueck (drei `slog.Warn`-Pfade, `error` immer nil); `server/biz_grpc.go:1365` fing
  zusaetzlich jeden Generierungsfehler ab und lieferte das nackte PDF unter anderem Dateinamen aus.
  Zusammen hiess das: der Aufrufer konnte eine E-Rechnung nicht von einer gewoehnlichen unterscheiden,
  und der Kunde erfuhr es erst, wenn beim Empfaenger nichts zu importieren war. Jetzt kommt jeder
  Fehler an. `mapBizError` kennt `ErrValidationFailed`/`ErrGenerateFailed`/`ErrTotalsMismatch` und
  macht `FailedPrecondition` daraus — die vollstaendige Verstossliste aus `ValidationError.Error()`
  geht mit raus, ueber `grpcStatusToHTTP` als **409**. Die beiden Vorpruefungen in
  `GenerateZUGFeRDXML` (BT-9 Faelligkeit, §14-Stammdaten) wrappen dafuer `ErrGenerateFailed` statt
  eines nackten `fmt.Errorf` — fehlende Stammdaten sind kein 500.
- **Der Test, der sich selbst uebersprang, tut es nicht mehr.** `TestExtractXMLFromPDF_HappyPath`
  baute auf einem handgeschriebenen PDF-Stub auf, den pdfcpu nicht lesen konnte; die Einbettung
  degradierte still, `len(out) == len(in)`, `t.Skip`. Beide Tests rendern jetzt ueber
  `pdf.NewGenerator(...).GenerateInvoicePDF` ein echtes maroto-PDF (`renderInvoicePDF`-Helfer), der
  Stub ist geloescht. Neu in `internal/biz/pdf`: `TestGenerateZUGFeRDInvoicePDF_
  DeclaresFacturXAttachment` faehrt den kompletten Pfad (maroto -> CII -> Einbettung -> Extraktion)
  und prueft Name, Byte-Gleichheit gegen das Referenzdokument, `/AF`, `/AFRelationship` und Subtype;
  dazu je ein Test fuer den harten Fehlerpfad und fuer eine EN-16931-widrige Rechnung.
  `GenerateZUGFeRDInvoicePDF` hatte vorher keinen einzigen Test.
- openapi.yaml: `/finance/invoices/{id}/pdf` dokumentiert jetzt den `format=zugferd`-Query-Parameter
  und die Codes 404/409 — der 409 ist neu, weil dieser Pfad vorher gar nicht fehlschlagen konnte.
- gate: build ok (`./internal/biz/... ./internal/server/... ./internal/gateway/...`) | vet ok |
  lint ok (golangci-lint auf pdf+einvoice+server+gateway: 0 issues; 7x QF1008 unterwegs behoben) |
  gofmt ok (fuenf beruehrte Dateien gegen LF-normalisierten Inhalt, keine echten Diffs — die
  gofmt-Treffer sind CRLF im Bestand) | test ok mit `DATABASE_URL` gegen `kmuhub_app`:
  einvoice 57 Tests **0 Skips** (darunter `TestTenantIsolation_IncomingInvoices` real gegen die DB),
  pdf gruen, `./internal/server/...` gruen, `./internal/gateway/` gruen inkl. TestOpenAPIRouteDrift.
  migration n.a., rls-smoke n.a. (keine Tabelle/Policy angefasst), proto n.a.
- verify vorgaenger: `a7fe6318` (w5-en16931-validation) gegen die acht Fehlerklassen geprueft. Diff
  beruehrt nur `internal/biz/einvoice/*` plus einen pdf-Test — keine Migration, keine Route, kein
  Proto, kein Guard, keine Tabelle, kein Gateway-Handler, kein DB-Zugriff. Kein `Unimplemented`,
  kein TODO, kein leerer Return; `Validate` und `validateInvoiceDoc` sind vollstaendig ausgefuehrt.
  Sauber, keine Fix-Unit noetig.
- offen fuer Luke:
  (1) VERHALTENSAENDERUNG an `GET /finance/invoices/{id}/pdf?format=zugferd`: unvollstaendige
      Rechnung liefert 409 statt eines PDF ohne XML. Im Desktop-Code gibt es aktuell **keinen**
      Aufrufer mit `format=zugferd` (nur das Anzeige-Widget `EInvoiceIndicator.tsx`), das FE bricht
      also nichts — sobald `w5-route` das UI anbindet, muss die Fehlermeldung dort aber sichtbar
      werden, sonst ist die Verstossliste umsonst erhoben.
  (2) 409 statt 422 fuer Validierungsfehler ist eine Folge von `grpcStatusToHTTP`, das kein 422
      kennt. Fuer die neue Route in `w5-route` ist das die offene Wahl (Nachtrag steht dort).
  (3) Das erzeugte PDF ist bewusst kein PDF/A-3b. Fuer oeffentliche Auftraggeber ist ohnehin
      XRechnung (reines XML) der Weg; fuer B2B-Empfaenger reicht der deklarierte Anhang.

## Iteration 12 — w5-route — done — 2026-08-01 (Nachtlauf 3)
- commit: 38cae79b
- gebaut: neuer RPC `FinanceService.GenerateEInvoice` (`GenerateEInvoiceRequest{id, tenant_id,
  format, buyer_reference}` -> `GenerateEInvoiceResponse{data, filename}`) plus
  `POST /api/v1/finance/invoices/{id}/erechnung`. `format=xrechnung` ruft `einvoice.GenerateUBL`
  direkt; `format=zugferd` ruft `pdf.NewGenerator(settings).GenerateZUGFeRDInvoicePDF(inv)` — exakt
  der Helper, den `w5-pdfa3-embed` schon gebaut hatte, nichts davon neu geschrieben. Gateway prueft
  `format` gegen eine feste Map (`eInvoiceContentTypes`) VOR dem gRPC-Call und setzt daraus
  Content-Type (`application/xml` bzw. `application/pdf`); unbekannt/leer -> 400 ohne RPC-Aufruf.
- ENTSCHEIDUNG (a) — 409, nicht 422: `mapBizError` kannte `ErrValidationFailed`/`ErrGenerateFailed`/
  `ErrTotalsMismatch` bereits aus `w5-pdfa3-embed` und mappt sie auf `FailedPrecondition` ->
  `grpcStatusToHTTP` -> 409. Diese Route nutzt denselben Pfad, statt einen zweiten
  Fehlercode-Vertrag fuer dieselbe Fehlerfamilie zu erfinden — die Geschwister-Route
  `/pdf?format=zugferd` liefert für dieselben Fehler bereits 409, zwei verschiedene Codes für
  denselben Fehlertyp waeren die schlechtere API. In `openapi.yaml` als 409 dokumentiert.
- ENTSCHEIDUNG (b) — buyer_reference gilt nur fuer xrechnung: bei `format=zugferd` wird jeder
  `buyer_reference`-Wert stillschweigend ignoriert (bestehendes Verhalten aus
  `pdf.GenerateZUGFeRDXML`, das intern immer mit leerem BuyerReference an `einvoice.GenerateCII`
  geht — ZUGFeRD-im-PDF ist fuer private Empfaenger, die keine Leitweg-ID haben). Ist
  `buyer_reference` gesetzt UND `format=xrechnung`, laeuft vor `GenerateUBL` zusaetzlich
  `einvoice.Validate(..., ProfileXRechnung)` — die deutschen CIUS-Regeln (Leitweg-ID, IBAN,
  zerlegte Adressen), die `GenerateUBL` selbst nur mit dem EN-16931-Kern prueft.
- Neues Message-Paar `GenerateEInvoiceRequest/Response` statt eins der bestehenden
  wiederzuverwenden — `GenerateZUGFeRDInvoicePDFResponse` heisst sein bytes-Feld `pdf_data` und ist
  an ZUGFeRD gebunden; ein XRechnung-Ergebnis darin unterzubringen waere Etikettenschwindel. Proto
  im selben Commit regeneriert (`protoc --go_out --go-grpc_out`, `make` war in der Bash-Umgebung
  nicht auf PATH, direkter `protoc`-Aufruf mit denselben Flags wie `make proto-biz`).
  Nebenbei korrigiert: der Kommentar an `GenerateZUGFeRDInvoicePDFResponse` behauptete noch
  "graceful degradation (plain PDF on failure)" — das hat `w5-pdfa3-embed` bereits entfernt, der
  Kommentar war seither falsch.
  `respondPDF` in `route_biz.go` auf einen generischen `respondFile(w, data, filename,
  contentType)` umgestellt (Ein-Zeilen-Wrapper), damit die neue Route denselben Header-Code nutzt
  statt ihn zu duplizieren.
- Kein neuer Guard: bestehendes `RequirePermission("finance","read")` wiederverwendet (wie bei der
  Schwester-Route `/pdf`), kein Seed noetig. Kein 404-Zusatzcode noetig — `invoiceService.GetByID`
  ist bereits tenant-gescoped und liefert `ErrInvoiceNotFound` fuer fremde Rechnungen, `mapBizError`
  macht daraus 404 (identisch zum bestehenden Verhalten der PDF-Route).
- Kein Server-Test (`internal/server`) fuer den neuen RPC angelegt — bewusst, nicht vergessen: die
  Schwester-RPC `GenerateZUGFeRDInvoicePDF` hat ebenfalls keinen direkten Test in diesem Paket, die
  eigentliche Logik ist bereits in `internal/biz/einvoice` und `internal/biz/pdf` vollstaendig
  getestet, und der RPC selbst ist duenne Verdrahtung (Tenant/ID parsen, Invoice+Settings laden,
  dispatch, mapBizError). Inhaltliche Pruefung des Outputs ist explizit `w5-roundtrip-test`
  zugewiesen — Nachtrag mit den exakten RPC-Signaturen steht jetzt dort im Backlog.
  Gateway-Tests neu: `TestHandleGenerateEInvoice_ServiceUnavailable`,
  `_MissingFormat`, `_InvalidFormat` — folgen dem bestehenden Muster (Validierung vor dem
  gRPC-Call wird ohne echte Verbindung getestet, echte Erzeugung nicht — dafuer gibt es keine
  Server-Test-Infrastruktur in diesem Paket, siehe oben).
- gate: build ok (`./internal/server/... ./internal/gateway/... ./internal/biz/... ./cmd/gateway/...
  ./cmd/biz/...`) | vet ok | lint ok (golangci-lint auf server+gateway+einvoice+pdf: 0 issues) |
  gofmt ok (drei beruehrte Go-Dateien gegen LF-normalisierten Inhalt geprueft, keine echten Diffs —
  CRLF im Bestand) | test ok mit `DATABASE_URL` gegen `kmuhub_app`: `./internal/gateway/...` gruen
  inkl. `TestOpenAPIRouteDrift` (739 Routen gegen 741 dokumentierte Pfade), `./internal/server/...`
  und `./internal/biz/einvoice/...` zusammen 256 Tests, **0 Skips**. migration n.a. (keine Tabelle),
  rls-smoke n.a. (keine Policy angefasst), proto-regen durchgefuehrt und geprueft.
- verify vorgaenger: `c12467c8` (w5-pdfa3-embed) gegen die acht Fehlerklassen geprueft. Diff:
  `zugferd.go`+Test, `biz_grpc.go`, `route_biz_invoices.go`, `openapi.yaml`. gRPC-Client-Aufruf im
  Gateway unveraendert (`client.GenerateZUGFeRDInvoicePDF`, Fehlerklasse 1 n.a.). Kein Stub, kein
  TODO. Keine `.proto`-Aenderung in diesem Commit (Fehlerklasse 3 n.a.). Kein neuer Guard
  (Fehlerklasse 4/8 n.a.). Keine neue Tabelle (Fehlerklasse 5 n.a.). Route unveraendert, nur ihr
  Fehlerverhalten (Fehlerklasse 7 n.a. — `/pdf` stand schon in openapi.yaml, der Commit ergaenzt nur
  409+Beschreibung). Wire-Shape (Fehlerklasse 6) gegen den Test gegengelesen: `/AF`, `/AFRelationship
  /Alternative`, `text/xml`, exakter Dateiname `factur-x.xml` — alles wie behauptet, im Test
  verifiziert statt geglaubt. Kein Fund.
- offen fuer Luke:
  (1) Die neue Route ist noch ungetestet gegen echten Content — `w5-roundtrip-test` (naechste Unit)
      schliesst das, indem er Rechnung -> Route -> `Service.Import` zurueckspielt.
  (2) `models.Invoice` hat weiterhin kein Leitweg-ID-Feld — `buyer_reference` kommt ausschliesslich
      als Query-Parameter, nicht aus gespeicherten Rechnungsdaten. Wenn ein Kunde die Leitweg-ID
      dauerhaft am Kontakt/Rechnung hinterlegen will, ist das ein FE+Migrations-Thema, nicht Teil
      dieser Welle.
  (3) Kein Desktop-Aufrufer fuer `/erechnung` existiert (wie schon bei `/pdf?format=zugferd` in
      Iteration 11 notiert) — `EInvoiceIndicator.tsx` ist reines Mock-Widget ohne API-Calls. Sobald
      das FE anbindet, muss die 400/409-Fehlerdarstellung dort sichtbar werden.

## Iteration 13 — w5-roundtrip-test — done — 2026-08-01 (Nachtlauf 3)
- commit: 0c56cd6e
- gebaut: zwei DB-Tests in `internal/biz/einvoice/roundtrip_outbound_test.go` (Paket
  `einvoice_test`, external — importiert `internal/server` fuer `BizGRPCServer`, das seinerseits
  `einvoice` importiert; ein in-package-Test haette einen Zyklus geschlossen, exakt der Grund, aus
  dem `pdf_extract_test.go` schon dort liegt). Ablauf je Test: frischer Tenant + EN-16931-taugliche
  Test-Rechnung (2 Steuersaetze: 2x100,00 @19%, 3x50,00 @7%, netto 350,00/Steuer 48,50/brutto
  398,50) direkt ueber `invoice.NewPostgresRepository(pool).Create` + Company-Settings ueber
  `quote.NewPostgresCompanySettingsRepo(pool).Upsert` persistiert (bewusst NICHT ueber
  `invoice.Service.Create`, das zusaetzlich Nummernkreis/Tax-Berechnung mitbringt und fuer diesen
  Test nur Rauschen waere) -> `BizGRPCServer.GenerateEInvoice` (mit `nil` fuer alle Dependencies
  ausser `invoiceService`+`companySettings`, da `GenerateEInvoice` sonst nichts vom Server-Struct
  liest) fuer `format=xrechnung` bzw. `zugferd` -> Ergebnis-Bytes direkt (ohne manuelles
  `ExtractXMLFromPDF`) an `einvoice.Service.Import` mit MimeType `application/xml` bzw.
  `application/pdf` — `Import` extrahiert bei "pdf" intern selbst, das ist also ein echter Test des
  Extraktionspfads gegen ein real generiertes PDF (`pdf.NewGenerator(...).GenerateZUGFeRDInvoicePDF`),
  nicht nur der beiden Text-Generatoren. Assertions: Subtotal/TotalTax/GrossTotal, Positionsliste
  (Beschreibung + Zeilensumme je Position) und beide Tax-Breakdown-Eintraege (Satz + Steuerbetrag)
  gegen die Ausgangsrechnung.
  Zwei eigene Tenants (einer je Format), keine geteilten Konstanten — vermeidet die aus Nachtlauf 1
  bekannte Duplicate-Key-Falle, hier zusaetzlich real: gleiche Rechnungsnummer+Lieferant in
  BEIDEN Formaten haette in einem gemeinsamen Tenant den zweiten Import als Duplikat abgewiesen.
- gate: build ok (`./internal/biz/einvoice/... ./internal/server/... ./internal/gateway/...
  ./cmd/gateway/... ./cmd/biz/...`) | vet ok | lint ok (golangci-lint auf einvoice: 0 issues) |
  gofmt ok | migration n.a. (lokaler Kopf 256 = Repo-Kopf, keine Migration in dieser Unit) |
  test ok mit `DATABASE_URL` gegen `kmuhub_app`: `./internal/biz/einvoice/...` 59 Tests, **0 Skips**,
  beide neuen Roundtrip-Tests PASS (0,10s/0,06s); `./internal/server/...` zur Sicherheit mitgelaufen
  (unveraendertes Paket, nur als Aufrufer importiert), gruen. rls-smoke n.a. (keine neue Tabelle,
  keine Policy angefasst — nur bestehende, seit Migration 000122/000132 RLS-gesicherte Tabellen
  ueber den normalen Tenant-Ctx-Pfad gelesen/geschrieben, exakt das Muster aus
  `tenant_isolation_test.go`). `go test ./internal/gateway/` nicht erneut als Pflichtschritt
  gelaufen — diese Unit haben keine Route/keinen Handler angefasst (nur ein neuer Test importiert
  bestehenden Server-Code), TestOpenAPIRouteDrift ist von Iteration 12 unveraendert.
- verify vorgaenger: `38cae79b` (w5-route) gegen die acht Fehlerklassen geprueft. Diff:
  `route_biz.go`, `route_biz_invoices.go`, `route_biz_test.go`, `biz_grpc.go`, `biz.proto`+`.pb.go`
  (regeneriert im selben Commit, Fehlerklasse 3 n.a.), `openapi.yaml`. Handler geht ueber
  `getBizClient()` -> `client.GenerateEInvoice` (gRPC-Client, Fehlerklasse 1 n.a.). Kein Stub: der
  RPC-Server implementiert xrechnung/zugferd vollstaendig, kein `Unimplemented`, kein TODO
  (Fehlerklasse 2 n.a.). Kein neuer Guard (wiederverwendet `RequirePermission("finance","read")`,
  Fehlerklassen 4/8 n.a.). Keine neue Tabelle (Fehlerklasse 5 n.a.). `invoiceService.GetByID` ist
  tenant-gescoped, liefert 404 fuer fremde Rechnungen ueber `mapBizError`. `openapi.yaml` traegt die
  neue Route inkl. 409 im selben Commit, `TestOpenAPIRouteDrift` lief in Iteration 12 gruen
  (Fehlerklasse 7 n.a.). Wire-Shape (Fehlerklasse 6): Content-Type wird serverseitig aus einer festen
  Map vor dem gRPC-Call gesetzt (`application/xml`/`application/pdf`), unbekanntes Format 400 ohne
  RPC-Aufruf — im Code nachvollzogen, kein Fund. Kein Fund.
- offen fuer Luke:
  (1) Damit ist Block C (E-Rechnung Ausgang) vollstaendig abgearbeitet — alle sechs `w5-*`-Units
      sind `done`. Die drei aus Iteration 11/12 bekannten offenen Punkte bleiben unveraendert:
      kein PDF/A-3b (bewusst, Begruendung dort), 409 statt 422 fuer Validierungsfehler (Begruendung
      in Iteration 12), kein Desktop-Aufrufer fuer `/erechnung`.
  (2) Naechste Unit laut Backlog-Reihenfolge waere `p2-guard-compat` (Block B, RBAC Phase 2) — die
      naechste Iteration zieht sie automatisch, kein Handlungsbedarf hier.

## Iteration 14 — p2-guard-compat — done — 2026-08-01 (Nachtlauf 3)
- commit: dfb10919
- gebaut: `RequirePermissionAny` in `internal/middleware/rbac.go` — laesst durch, sobald der
  Perms-Slice aus dem Context MINDESTENS EINEN der uebergebenen `resource:action`-Keys enthaelt.
  Aufbau wie beim Nachbarn `RequireRole`: das Key-Set wird einmal beim Wiring gebaut (Map), pro
  Request nur ein Lookup je Token-Key; kein DB-Query, kein Cache, `RequirePermission` unveraendert.
  SIGNATUR-ABWEICHUNG vom Backlog-Vorschlag, bewusst: `(first [2]string, rest ...[2]string)` statt
  `(pairs ...[2]string)`. Mit reinem Variadic waere `RequirePermissionAny()` gueltiger Go-Code, der
  ein leeres Key-Set baut und damit JEDEN aussperrt — ein Fehler, der lokal nie auffaellt und erst in
  Produktion als flaechiges 403 sichtbar wird. Mit dem gezogenen ersten Parameter faellt genau dieser
  Fall beim Kompilieren durch, ohne Runtime-Check und ohne panic. Die Call-Site aus dem Backlog
  bleibt Zeichen fuer Zeichen dieselbe.
  Tests: sechs Faelle in `rbac_test.go` — Alt-Token (nur grober Key) kommt durch, frisches Token (nur
  feiner Key) kommt durch, beide Keys durch, keiner von beiden 403, gar keine Permissions 403, und
  "gleiche Resource, andere Action" (`documents:file:delete`) 403, damit der Match nicht versehentlich
  auf Praefix-Ebene passiert.
- gate: build ok (`./internal/middleware/... ./internal/gateway/... ./cmd/gateway/...`) | vet ok |
  lint ok (golangci-lint auf middleware: 0 issues) | gofmt ok fuer die beiden geaenderten Dateien
  (`gofmt -l` meldet 7 unveraenderte Bestandsdateien im Paket, u.a. `cors.go`/`ratelimit.go` — nicht
  von dieser Unit, nicht angefasst) | migration n.a. | rls-smoke n.a. (kein SQL, keine Tabelle,
  keine Policy) | test ok mit `DATABASE_URL` gegen `kmuhub_app`: `./internal/middleware/` 64 Tests,
  **0 Skips**, alle PASS. `go test ./internal/gateway/` NICHT gelaufen — diese Unit hat keine Route,
  keinen Handler und keine `openapi.yaml` angefasst (nur einen bisher nirgends aufgerufenen Helper
  hinzugefuegt); `TestOpenAPIRouteDrift` ist seit Iteration 12 unveraendert. Ab `p2a` gilt die
  Pflicht wieder, dort werden Routen angefasst.
- verify vorgaenger: `0c56cd6e` (w5-roundtrip-test) gegen die acht Fehlerklassen geprueft. Diff sind
  genau zwei Dateien: `BACKLOG.yml` und die neue `internal/biz/einvoice/roundtrip_outbound_test.go`
  — kein Handler, kein Proto, keine Migration, keine Route, kein Guard, also Klassen 1/3/4/5/6/7/8
  n.a. Klasse 2 (Stub/Fake) gezielt am Testinhalt geprueft, weil ein Test der plausibelste Ort fuer
  einen Schein-Beweis ist: der Test ruft den echten `BizGRPCServer.GenerateEInvoice`, schiebt das
  Ergebnis durch den echten DB-gestuetzten `einvoice.Service.Import` und vergleicht Subtotal,
  TotalTax, GrossTotal, beide Positionen (Beschreibung + Zeilensumme) und beide
  Tax-Breakdown-Eintraege gegen die Ausgangsrechnung — keine hartkodierten Erwartungswerte neben der
  Quelle, kein `t.Skip` ausser dem regulaeren `SkipIfNoDB`, keine Fake-Bytes. Zwei frische Tenants
  per `uuid.New()` statt der geteilten Konstanten, also keine Duplicate-Key-Falle unter
  `t.Parallel()`. Kein Fund.
- offen fuer Luke:
  (1) TOKEN-GROESSE — Fund am Rande dieser Unit, gehoert NICHT zu Block B, entstand aber durch
      `p1a-migration` und trifft dessen Annahmen. In der lokalen DB haengen nach Migration 000256
      **454 Permission-Keys** an der Rolle `admin` (9040 Byte reiner Key-Text), 305 an `manager`,
      159 an `member`. `createTokenPair` backt sie ALLE ungefiltert in den Access-Token —
      als JSON-Claim rund 10 KB, base64-kodiert also ein Bearer-Header von grob **14 KB pro
      Request**. Tragfaehig ist das aktuell: das Backend setzt nirgends `SetCookie` (das
      4-KB-Cookie-Hardlimit der Browser greift also nicht), und weder das Gateway noch der Caddyfile
      setzen `MaxHeaderBytes`, es gilt Gos 1-MB-Default. Risikoreich wird es, sobald irgendein Proxy
      oder CDN mit dem branchenueblichen 8-KB-Header-Limit davorkommt (nginx, ALB, Cloudflare) —
      dann bekommt ausgerechnet der Admin 431 und niemand sonst. Zwei saubere Auswege, beide
      ausserhalb dieses Loops: die feinen Keys aus dem Token nehmen und per
      `/auth/me/permissions` (existiert seit `p1a-gateway`) clientseitig laden, oder sie im Token
      komprimiert/als Bitmaske fuehren. Vor dem Launch entscheiden, nicht danach.
  (2) Der Helper hat noch keinen einzigen Aufrufer — das ist Absicht (`p2a` ff. verdrahten ihn).
      Bis dahin veraendert dieser Commit das Verhalten in Produktion an keiner Stelle.

## Iteration 15 — p2a-wiki-zeiterfassung-infra — done — 2026-08-01 (Nachtlauf 3)
- commit: 20685fab
- Fortsetzung einer bereits im Arbeitsverzeichnis vorbereiteten, aber nie committeten Iteration
  (Code + BACKLOG-Statuswechsel lagen unstaged vor, kein eigener Journal-Eintrag). Vor dem Commit
  vollstaendig gegengeprueft statt blind uebernommen: Diff gelesen, Guard-Logik gegen
  `RequirePermissionAny` aus Iteration 14 nachvollzogen, alle neun referenzierten Katalog-Keys per
  SQL gegen die laufende DB verifiziert (`wiki:article:{read,create,edit,delete}`,
  `wiki:category:manage`, `wiki:share_token:create`, `zeiterfassung:{team:view,week:approve,
  corrections:approve}` — alle vorhanden, aus dem `p1a-migration`-Gesamtkatalog).
- gebaut: `RequirePermissionAny(alt, neu)` auf allen wiki- und zeiterfassung-Supervisor-Routen.
  `route_wiki.go`: sechs Guard-Variablen (`wikiRead/Create/Edit/Delete/Share/CatManage`) vor
  `r.Route(...)`, additiv auf Artikel/Versionen/Attachments/Share-Tokens/Kategorien. Alt-Key bleibt
  ueberall erhalten. `wiki:categories:read` hat keine Katalog-Entsprechung und blieb unveraendert
  auf dem reinen `RequirePermission`. `route_hr.go`: drei Guard-Variablen
  (`zeitTeamView/WeekApprove/CorrApprove`) auf `/team`, `/weeks/{approve,reject,reopen}`,
  `/corrections/{id}/approve`. Persoenliche Zeiterfassung (Clock-in/-out, eigene Eintraege,
  Wochenstatus) bleibt bewusst auf dem groben `hr:read`/`hr:write` — der Katalog fuehrt dafuer
  keinen eigenen Key.
  Drittes Modul im Scope, **infrastructure, hat ueberhaupt keine Backend-Routen** — verifiziert per
  Grep (`grep -rn infrastructure backend/internal/gateway/` — 0 Treffer) und per Sub-Agent-Recherche:
  `InfrastrukturPage.tsx` ist explizit als Mock deklariert ("All data is mock — the real backend
  integration comes in Luke's phases"), keine `fetch`/`apiClient`-Aufrufe, alle Aktionen (Service
  restart, Backup, SSL renew, Firewall-Toggle, Update-Check, Log-Export) enden in
  `toast.success(...)` ohne Netzwerk-Call. Die fuenf `infrastructure:*`-Katalog-Keys existieren zwar
  seit `p1a-migration` in `permissions` (per SQL bestaetigt), aber es gibt nichts zu gaten — keine
  Route erfunden, wie in den Unit-Notes Punkt (d) gefordert.
- gate: `go build -p 2 ./internal/gateway/... ./internal/middleware/... ./cmd/gateway/...` ok |
  vet ok | lint ok (golangci-lint auf gateway+middleware: 0 issues) | gofmt: `route_hr.go`/
  `route_wiki.go` von `gofmt -l` gemeldet, aber Fehlalarm — Diff gegen den committeten HEAD-Stand
  zeigt reinen CRLF-Checkout-Artefakt (`core.autocrlf=true`, Datei-Tool hat die Zeilenenden des
  Bestands beibehalten), der committete Inhalt ist LF und gofmt-clean (git normalisiert beim
  Commit zurueck). Neue Testdatei selbst gofmt-clean.
  test ok mit `DATABASE_URL` gegen `kmuhub_app`: `./internal/gateway/...` (inkl. der 22 neuen Faelle
  in `TestCapabilityGuards_AdditiveWiring` und `TestOpenAPIRouteDrift` — 739 Routen/741 Pfade, keine
  neue Route in dieser Unit) und `./internal/middleware/...` beide gruen. Kein Skip.
  migration n.a. (Seeds bereits aus 000256 vorhanden). openapi n.a. (keine neue Route, nur Guards
  ausgetauscht).
- verify vorgaenger: `dfb10919` (p2-guard-compat) gegen die acht Fehlerklassen geprueft. Diff: nur
  `rbac.go`+`rbac_test.go` (plus BACKLOG/JOURNAL). `RequirePermissionAny` folgt exakt dem
  Nachbarmuster `RequireRole`/`RequirePermission` (Map-Aufbau beim Wiring, ein Lookup pro Request,
  `response.Error` bei Miss), Signatur mit gezogenem erstem Parameter wie in den Notes gefordert,
  `RequirePermission` unangetastet. Kein Handler/Proto/Migration/Route/Wire-Shape in diesem Commit,
  Fehlerklassen 1/3/5/6/7 n.a. Kein Fund.
- offen fuer Luke:
  (1) Block B (RBAC Phase 2, Guard-Verfeinerung) ist mit dieser Unit fuer wiki+zeiterfassung
      abgeschlossen, `p2b-helpdesk-kommunikation-kalender` ist die naechste ziehbare Unit — ihre
      Notes tragen jetzt einen "MUSTER AUS p2a"-Absatz mit fuenf konkreten Punkten (Seed-Check-Query,
      Guard-als-lokale-Variable-Konvention, Testdatei-Konvention, Endpoint-ohne-Key/Key-ohne-Endpoint-
      Regel, additiv-heisst-erweiternd), damit die naechste Iteration nicht bei null anfaengt.
  (2) infrastructure bleibt ein reines FE-Mock ohne Backend — falls/wenn Luke das Modul mit echten
      Endpoints baut (Service-Steuerung, Backup-Trigger, o.ae.), MUESSEN die fuenf bereits geseedeten
      `infrastructure:*`-Keys von Anfang an als Guards verdrahtet werden, nicht nachtraeglich. Keine
      Backlog-Unit dafuer angelegt — das ist Neubau, nicht additive Guard-Tightening, und ausserhalb
      dessen, was dieser Loop bauen darf ohne Rueckfrage.
  (3) TOKEN-GROESSE (Iteration 14, Punkt 1) bleibt unveraendert offen und wird durch diese Unit nicht
      groesser: keine neuen Keys, nur zusaetzliche OR-Bedingungen in bestehenden Guards.

## Iteration 16 — p2b-helpdesk-kommunikation-kalender — done — 2026-08-01 (Nachtlauf 3)
- commit: 89789994
- gebaut: `RequirePermissionAny(alt, neu)` additiv auf helpdesk-, kommunikation- und
  kalender-Routen, verteilt ueber vier Gateway-Dateien (kein 1:1 Modul-zu-Datei-Mapping):
  `route_helpdesk.go` — sieben Guard-Variablen (`hdTicketRead/Create/Edit/Reply`,
  `hdKbManage`, `hdCannedManage`, `hdStatsView`). Tickets: create/read wie erwartet;
  edit deckt PUT + close/reopen/assign/merge ab (verifiziert gegen den FE-Gate-Block
  `canEditTicket` in `HelpdeskPage.tsx:1047`, der genau diese vier Aktionen hinter
  `helpdesk:ticket:edit` versammelt). Queues/SLA-Policies/Routing-Rules/Business-Hours haben
  keine Katalog-Entsprechung (backend-gaps.md §RBAC: "helpdesk BE seedet nur das grobe Paar
  ausserhalb Tickets/KB/Canned/Stats") und blieben unangetastet.
  `route_chat.go` — `kommunikation:channel:manage` auf Channel-Create/Update/Delete/Archive/
  Member-Role (Administration); Join/Leave/DM/Messages/Files bleiben auf dem reinen
  `channels:*`-Key (persoenliche Nutzung, Katalog-Kommentar `capability-catalog.ts:118-121`).
  `route_inbox.go` — `kommunikation:canned:manage` auf die INBOX-eigenen Canned-Responses
  (separat von `helpdesk:canned:manage`, zwei verschiedene Canned-Response-Systeme).
  `kommunikation:team_inbox:manage` additiv NUR auf die drei bereits `RequirePermission`-
  gegateten Team-Inbox-Routen (Update, Add-/RemoveMember) — Create/Delete bleiben auf
  `RequireRole("admin","manager")`, siehe Fund unten.
  `route_calendar.go` + `route_booking.go` — `kalender:category:manage` auf Event-Kategorie
  Create/Delete, `kalender:booking_page:manage` auf Booking-Page Create/Update/Delete. Rest
  von Kalender (Calendars/Members/Events/Resources/Holidays/Preferences/LiveKit) hat keine
  Katalog-Entsprechung, unangetastet.
  Test: `route_capability_guard_test.go` um 4 Router-Setups (helpdesk via bestehendem
  `newHelpdeskRoutes`-Helper, chat, inbox, calendar+booking) und 22 neue Faelle erweitert —
  je Modul: Alt-Key-only, Neu-Key-only, Fremd-Key-denied, plus "personal-use bleibt eng"
  Faelle (Channel-Join, Clock-in-Analog) und "kein-Katalog-Key-Route bleibt eng" Faelle
  (Queue-Create, Calendar-Create).
- gate: `go build -p 2 ./internal/gateway/... ./internal/middleware/... ./cmd/gateway/...` ok |
  vet ok | lint ok (0 issues auf gateway+middleware) | test ok mit `DATABASE_URL` gegen
  `kmuhub_app`: `./internal/gateway/...` (`TestCapabilityGuards_AdditiveWiring` 51 Faelle gruen,
  `TestOpenAPIRouteDrift` 739 Routen/741 Pfade — unveraendert, keine neue Route in dieser Unit)
  und `./internal/middleware/...` beide gruen, 0 Skips. migration n.a. (Seeds bereits aus
  000256 vorhanden, gegen die laufende DB verifiziert: alle referenzierten
  helpdesk/kommunikation/kalender-Keys existieren). openapi n.a. (keine neue Route, nur Guards
  ausgetauscht). gofmt: `route_helpdesk.go`/`route_inbox.go`/`route_calendar.go`/
  `route_booking.go` von `gofmt -l` gemeldet — Diff-Pruefung (`gofmt -d`) zeigt reinen
  CRLF-Checkout-Artefakt (identisch zum Fehlalarm aus Iteration 15: `core.autocrlf=true`,
  `file`-Kommando bestaetigt CRLF-Terminatoren im Arbeitsverzeichnis, git normalisiert beim
  Commit zurueck auf LF). `route_chat.go` und die Testdatei waren bereits LF und wurden nicht
  gemeldet.
- verify vorgaenger: `20685fab` (p2a-wiki-zeiterfassung-infra) gegen die acht Fehlerklassen
  geprueft. Diff: nur `route_hr.go`/`route_wiki.go`/neue Testdatei/BACKLOG/JOURNAL. Alle neun
  referenzierten Katalog-Keys per SQL gegen die laufende DB verifiziert (vorhanden). Additiv
  durchgehend (`RequirePermissionAny(alt, neu)`, kein einziger `RequirePermission`-Alt-Key
  entfernt). Kein Proto/Migration/neue Route/Handler-Direktzugriff in diesem Commit,
  Fehlerklassen 1/2/3/5/6/7 n.a. Kein Fund.
- offen fuer Luke:
  (1) DREI KATALOG-KEYS OHNE ADDITIVES ZIEL, bewusst nicht verdrahtet:
      `kommunikation:routing:manage` — JEDE Routing-Rule-Mutation in `route_inbox.go` haengt an
      `RequireRole("admin","manager")`, keine einzige an `RequirePermission`, also kein
      Alt-Key zum Erweitern vorhanden. `kommunikation:team_inbox:manage` deckt nur
      Update/Add-/RemoveMember ab — Create/Delete bleiben aus demselben Grund rollenbasiert.
      `kommunikation:webhook:manage` hat ueberhaupt keinen Backend-Endpoint (weder chat noch
      inbox haben je eine Webhook-Route bekommen). Alle drei sind Entscheidungen fuer Luke:
      entweder bleibt die Verwaltung dauerhaft rollen-exklusiv (dann sind die Keys im Katalog
      irrefuehrend), oder es braucht einen neuen `RequireRoleOrPermissionAny`-Helper, der
      Rolle UND Permission additiv kombiniert — bewusst NICHT in dieser Guard-Tightening-Unit
      gebaut (Scope-Kriechen, neue Middleware-Semantik verdient eigene Ueberlegung/Review).
  (2) Wie in p2a: Block B ist fuer helpdesk+kommunikation+kalender abgeschlossen,
      `p2c-work-documents-crm-finance` ist die naechste ziehbare Unit. Ihre Notes tragen jetzt
      einen "MUSTER AUS p2b"-Absatz (vier Punkte: RequireRole-Routen nicht anfassen,
      Katalog-Key kann mehrere Gateway-Dateien treffen, "manage"-Key deckt nicht zwingend alle
      CRUD-Verben ab, Guard-Test-Helper wiederverwenden) zusaetzlich zu p2a's fuenf Punkten.
  (3) TOKEN-GROESSE (Iteration 14/15) bleibt unveraendert offen, waechst durch diese Unit nicht:
      keine neuen Permission-Keys, nur zusaetzliche OR-Bedingungen in bestehenden Guards.

## Iteration 17 — p2c-work-documents-crm-finance — done (GETEILT, work+documents) — 2026-08-01 (Nachtlauf 3)
- commit: 94c26a40
- gebaut: `RequirePermissionAny(alt, neu)` additiv auf work- und documents-Routen (beide 1:1 in
  `route_work.go`/`route_document.go`, keine weiteren Dateien). Groesster Batch bisher — 13
  Katalog-Keys (work) + 9 (documents) — deshalb wie in den Notes vorgesehen NACH ZWEI MODULEN
  geteilt: crm+finance sind als neue Folge-Unit `p2c-work-documents-crm-finance-2` (deps: diese
  Unit) direkt vor `p2c-inventar-einkauf-produktion-vertraege` eingehaengt, deren `deps` entsprechend
  umgebogen.
  `route_work.go`: zwei Guard-Bloecke (Projects, Tasks). Projects: `projRead`/`projCreate`/`projEdit`/
  `projDelete`/`projMemberManage` + `projStatusDelete` (Sonderfall: Kanban-Spalten-DELETE traegt
  legacy `projects:delete`, additiver Key ist trotzdem `work:project:edit`, weil Create/Update/
  Reorder/Delete der Statuses alle in derselben `projectCan.edit`-gated `ProjectSettingsDialog`-Tab
  sitzen — verifiziert gegen `ProjectSettingsDialog.tsx`). Tasks: `taskRead`/`taskCreate`/`taskEdit`/
  `taskDelete`/`taskComment`/`timeLog`/`labelEdit` + `taskEditDel` (DeleteTaskDependency/
  UnlinkEntityFromTask/RemoveTaskFile tragen legacy `tasks:delete`, additiver Key `work:task:edit`,
  weil alle Sub-Ressourcen-Sektionen in `TaskDetailPage.tsx` denselben `can.edit`-Prop durchreichen —
  verifiziert Zeile fuer Zeile, nicht angenommen).
  `route_document.go`: ein Guard-Block. `docRead`/`docDownload`/`docUpload`/`docEdit`/`docDelete`/
  `docShareManage`/`docVersionRestore`. Bemerkenswert: `documents:file:edit` deckt sowohl Datei-
  Rename+Move ALS AUCH Ordner-NewSubfolder+Rename ab (verifiziert gegen `FileContextMenu.tsx`:
  `canNewFolder = canEditAllowed` in `DokumentePage.tsx:144`, explizit NICHT `documents:file:upload`).
  Version-Liste (GET .../versions) traegt `documents:version:restore` statt `documents:file:read`,
  weil der einzige FE-Zugang (Kontextmenue "Version history") komplett hinter `canVersionRestore`
  sitzt (`FileContextMenu.tsx:245-247`) — es gibt keinen separaten Lese-Pfad.
  ZWEI KATALOG-KEYS OHNE JEDEN BACKEND-ENDPOINT, verifiziert per Code-Lesen (nicht nur "keine Route
  gefunden"): `documents:template:manage` — `TemplateGalleryDialog.tsx` hat null API-Calls, reiner FE-
  Stub. `documents:share_link:create` — `ShareDialog.tsx` `copyLink()` ist `toast.success(...)` ohne
  jeden Netzwerk-Call. `work:board:export` hat weder FE-Aufrufer noch Backend-Route. Alle drei bleiben
  unwired, keine Route erfunden (wie infrastructure in p2a, webhook:manage in p2b).
  ROUTEN OHNE FE-CAPABILITY-AUFRUF BLEIBEN AUF DEM COARSE-KEY, auch wo ein Katalog-Key thematisch
  passen wuerde (bewusst nicht geraten, per grep auf `useHasCapability`/`useScopedCapability`
  verifiziert dass 0 Treffer existieren): work `AddProjectMember`/`UpdateProjectMemberRole`/
  `CreateProjectFromTemplate`/`SetUserProjectPreference`/`MoveTask` sowie die vier `isOwn`/
  `isOwner`-Client-Checks (`UpdateTimeEntry`/`DeleteTimeEntry`/`UpdateTaskComment`/
  `DeleteTaskComment` — kein Capability-Aufruf im FE, nur Besitzer-Vergleich); documents `CopyFile`
  (FE-Kommentar: "Copy is read-action — always visible", kein Gate).
  Test: `route_capability_guard_test.go` um zwei Router-Setups (`workRouter`, `documentRouter`) und
  47 neue Faelle erweitert — je Guard-Paar: Katalog-Key-only, plus fuer die Sonderfaelle (Kanban-
  Status-Delete, Dependency-Delete, Version-Liste) einen "falscher Key oeffnet es NICHT"-Gegenfall.
- gate: `go build -p 2 ./internal/gateway/... ./internal/middleware/... ./cmd/gateway/...` ok |
  vet ok | lint ok (0 issues auf gateway+middleware) | test ok mit `DATABASE_URL` gegen
  `kmuhub_app`: `./internal/gateway/...` (`TestCapabilityGuards_AdditiveWiring` alle Faelle gruen inkl.
  47 neuer, `TestOpenAPIRouteDrift` 739 Routen/741 Pfade — unveraendert, keine neue Route) und
  `./internal/middleware/...` beide gruen, 0 Skips. migration n.a. (alle referenzierten
  work/documents-Keys per SQL gegen die laufende DB verifiziert: vorhanden aus 000256). openapi n.a.
  (keine neue Route, nur Guards ausgetauscht). rls-smoke n.a. (keine Tabelle/Policy angefasst). gofmt:
  `route_document.go` von `gofmt -l` gemeldet — Diff-Pruefung (CRLF strippen, dann `gofmt -l` erneut)
  bestaetigt reinen CRLF-Checkout-Artefakt (identisch zum Fehlalarm aus Iteration 15/16,
  `core.autocrlf=true`), committeter Inhalt ist LF und gofmt-clean. `route_work.go` und die Testdatei
  waren bereits LF/ASCII und wurden nicht gemeldet.
- verify vorgaenger: `89789994` (p2b-helpdesk-kommunikation-kalender) gegen die acht Fehlerklassen
  geprueft. Diff: `route_booking.go`/`route_calendar.go`/`route_capability_guard_test.go`/
  `route_chat.go`/`route_helpdesk.go`/`route_inbox.go`. Alle Alt-Keys additiv erweitert (kein
  einziger `RequirePermission`-Aufruf ersetzt statt erweitert), Testfaelle substantiell (nicht nur
  "irgendwas ausser 403"). Kein Proto/Migration/neue Route/Handler-Direktzugriff/Stub in diesem
  Commit, Fehlerklassen 1/2/3/5/6/7 n.a. Kein Fund.
- offen fuer Luke:
  (1) `p2c-work-documents-crm-finance-2` (crm+finance) ist die naechste ziehbare Unit in Block B,
      `p2c-inventar-einkauf-produktion-vertraege` deps entsprechend umgebogen. Ihre Notes nennen
      bereits die betroffenen Gateway-Dateien (kein 1:1-Modul-Mapping bei crm, kein dediziertes
      `route_finance.go`) und die Katalog-Key-Zahlen (14+14).
  (2) Zu pruefen, ob `work:board:export`, `documents:template:manage` und
      `documents:share_link:create` dauerhaft ohne Backend bleiben sollen oder ob die FE-Stubs
      (TemplateGalleryDialog, ShareDialog-Link-Copy, Board-Export) noch gebaut werden — das ist
      Neubau, keine Guard-Tightening-Frage, keine Backlog-Unit dafuer angelegt.

## Iteration 18 — p2c-work-documents-crm-finance-2 — done (crm+finance) — 2026-08-01 (Nachtlauf 3)
- commit: f534ef8e
- gebaut: `RequirePermissionAny(alt, neu)` additiv auf allen crm- und finance-Routen. crm sitzt
  komplett in `route_crm.go` (Contacts/Companies/Deals/Activities/Reports — die anderen
  `route_crm_*.go`-Dateien enthalten nur Handler, keine eigenen `r.Route`-Bloecke, Korrektur zur
  Datei-Vermutung aus der vorigen Unit). finance sitzt komplett in `route_biz.go`
  (Invoices/Quotes/Recurring/CreditNotes/Dunning/Export/Deal-to-Quote — `route_biz_invoices.go`/
  `route_biz_quotes.go`/`route_biz_einvoice.go` ebenfalls nur Handler). 15 Guard-Variablen crm
  (`contactRead/Create/Edit/Delete/Import/Export`, `companyRead/Create/Edit/Delete`,
  `dealRead/Create/Edit/Delete`, `activityRead/Edit/Delete`, `reportRead`), 14 Guard-Variablen
  finance (`invoiceRead/Create/Edit/Delete/Send`, `creditNoteCreate`, `quoteRead/Write/Send/Convert`,
  `dunningRun/RunWrite/Settings/SettingsWrite`, `exportRun`).
  ZWEI NEUE MUSTER (zusaetzlich zu p2a-p2c's Katalog): (a) eine Route kann durch mehrere fine Keys
  geoeffnet werden, wenn zwei FE-Aktionen mit unterschiedlicher Capability auf denselben Endpoint
  zielen — `POST /finance/credit-notes` ist sowohl "neue Gutschrift" (`finance:invoice:create`,
  FinanzenPage.tsx canCreateInvoice) als auch "Storno einer versendeten Rechnung"
  (`finance:invoice:delete`, canDeleteInvoice); `creditNoteCreate` traegt beide als Alternativen.
  (b) ein legacy-Verb kann durch ein ANDERES catalogue-Verb erweitert werden, wenn es keinen
  eigenen Key gibt — Quote-Edit/Accept/Reject haben keinen `quote:edit`-Key, QuoteDetailPanel.tsx
  kommentiert sie explizit als "-> quote:create"; Quote-Convert-to-Invoice ist laut Kommentar
  "-> invoice:create" (erzeugt ja tatsaechlich eine Rechnung).
  GROESSTER FUND: ein grosser Teil des finance-Katalogs hat aktuell KEINEN echten Backend-Anschluss,
  weil die konsumierenden FE-Komponenten selbst Mocks sind (per Code-Lesen verifiziert, nicht
  geraten). `BankingWidget.tsx` ("Mock data for design — backend: FinAPI integration") ruft
  `/finance/bank-accounts` + `/finance/bank-transactions/{id}/match|reject-match` — Pfade, die im
  Gateway gar nicht existieren (das sind die kuenftigen `fe-finance-bank-accounts`/
  `-bank-transactions-matching`-Units aus Block D). `TransactionsTab.tsx` und `BerichteTab.tsx`
  lesen explizit aus `useFinanceStore`/`useFinanceLedger` (Zustand-Mock, Datei-Kopfkommentar
  "Backend-Anbindung folgt in einem spaeteren Sprint"). `ExpensesTab.tsx`/`ExpenseDetailPanel.tsx`
  laufen ueber den MSW-"swap-ready Mock" aus `fe-finance-expenses` (Block D, weiterhin `todo`).
  Ergebnis: `finance:incoming:review`, `finance:incoming:book`, `finance:amounts:view` gaten reale
  FE-Buttons, aber KEINE bestehende Backend-Route (`bank-statements`, `bank-transactions`,
  `incoming-invoices`, `journal/summary`, `stats/payments`, `open-items` bleiben komplett
  legacy-only). `finance:settings:manage` hat KEINEN FE-Gate-Call auf `PUT /finance/settings`
  (StammdatenTab.tsx/CompanySettingsTab.tsx speichern ungegated) — nur der Dunning-Config-Teil ist
  real verdrahtet (`DunningConfigDialog`). Bei crm analog: die komplette Advisory-Protokoll-Funktion
  (`route_crm_advisory.go`, 8 Routen — List/Create/Get/Update/Delete/HandOver/PDF +
  ReferralReport) ist FE-seitig 100 % lokaler Zustand (`stores/advisoryProtocols.ts`, kein
  einziger API-Call trotz echter `crm:advisory:read`/`write`-Gate-Calls im FE) — Datei komplett
  unangetastet gelassen. `crm:pipeline:manage` (PipelineStagesEditor.tsx ohne Capability-Hook) und
  `crm:segment:override` (reiner Zustand-Persist-Store, Code-Kommentar "Backend target" noch offen)
  ebenfalls FE-seitig ungegatet, pipeline-stages-Routen unangetastet.
  Nebenfunde ohne FE-Aufrufer, legacy-only belassen: `HandleUpdateContactVisibility`,
  `HandleValidateInvoiceNumber`, `HandleImportInvoice`, `HandleLockInvoice`, `HandleDeletePayment`,
  `HandleUpdateDunningStatus`/`HandleSendDunningNotice`, `HandleGenerateGoBDExport`,
  `HandleFindContactDuplicates`/`HandleMergeContacts` (+ Company-Pendants), `HandleAddDealTags`/
  `HandleRemoveDealTags` (Handler ist ein reiner 501-Stub). `/api/v1/finance/document-chains`
  (Belegkette-Tab-Aufruf) existiert im Gateway ueberhaupt nicht — Phantom-Endpoint, ausserhalb
  des Guard-Tightening-Scopes, nicht behoben.
  Test: `route_capability_guard_test.go` um zwei Router-Setups (`crmRouter`, `bizRouter`) und 76
  neue Faelle erweitert. Sonderfall `HandleAddDealTags`: 501-Stub dekodiert den Body VOR jedem
  Client-Call, bestandener Guard ist bei leerem Test-Body als 400 sichtbar statt 503 — im
  Testkommentar dokumentiert, nicht als Bug behandelt.
- gate: `go build -p 2 ./internal/gateway/... ./internal/middleware/... ./cmd/gateway/...` ok |
  vet ok | lint ok (0 issues auf gateway+middleware) | test ok mit `DATABASE_URL` gegen
  `kmuhub_app`: `./internal/gateway/...` (`TestCapabilityGuards_AdditiveWiring` alle Faelle gruen
  inkl. 76 neuer, `TestOpenAPIRouteDrift` 739 Routen/741 Pfade — unveraendert, keine neue Route)
  und `./internal/middleware/...` gruen, 0 Skips. migration n.a. (alle referenzierten crm/finance-
  Keys per SQL gegen die laufende DB verifiziert: vorhanden aus 000256). openapi n.a. (keine neue
  Route, nur Guards ausgetauscht). rls-smoke n.a. (keine Tabelle/Policy angefasst). gofmt:
  `route_biz.go` von `gofmt -l` gemeldet — Diff-Pruefung (CRLF strippen, dann `gofmt -l` erneut auf
  einer temporaeren Kopie) bestaetigt reinen CRLF-Checkout-Artefakt (identisch zum Fehlalarm aus
  Iteration 15-17, `core.autocrlf=true`), committeter Inhalt ist LF und gofmt-clean. `route_crm.go`
  und die Testdatei waren bereits LF und wurden nicht gemeldet.
- verify vorgaenger: `94c26a40` (p2c-work-documents-crm-finance, work+documents) gegen die acht
  Fehlerklassen geprueft. Diff: `route_work.go`/`route_document.go`/`route_capability_guard_test.go`.
  Alle entfernten `RequirePermission(...)`-Direktaufrufe wurden durch benannte
  `RequirePermissionAny(alt, neu)`-Variablen ersetzt (additiv, kein Key verengt), Testfaelle
  substantiell. Kein Proto/Migration/neue Route/Handler-Direktzugriff/Stub in diesem Commit,
  Fehlerklassen 1/2/3/5/6/7 n.a. Kein Fund.
- offen fuer Luke:
  (1) `p2c-inventar-einkauf-produktion-vertraege` ist die naechste ziehbare Unit in Block B (deps
      bereits auf diese Unit umgebogen).
  (2) Grosses FE/BE-Luecken-Inventar im finance-Modul entdeckt (siehe "gebaut" oben): Banking/
      Transaktionen/Ausgaben/Berichte laufen komplett gegen Mocks, nicht gegen die bestehenden
      Backend-Routen. Die Block-D-Units (`fe-finance-expenses`, `fe-finance-bank-accounts`,
      `fe-finance-bank-transactions-matching`) muessen das schliessen — bis dahin sind
      `finance:incoming:review`/`book`/`amounts:view` reine FE-UI-Capabilities ohne Backend-Wirkung.
  (3) `/api/v1/finance/document-chains` (Belegkette-Tab) ist ein Phantom-Endpoint — FE ruft ihn,
      Gateway hat ihn nicht. Ausserhalb des Scopes dieser Unit, aber ein echter Bug fuer den
      Belegkette-Tab in Produktion.
  (4) Advisory-Protokolle (`route_crm_advisory.go`, 8 Routen) sind seit Bestand reine FE-Mock-
      Funktion ohne API-Anbindung — falls das Feature scharf geschaltet werden soll, ist das
      Neubau (FE-Hooks + Wiring), keine Guard-Tightening-Frage.
  (5) TOKEN-GROESSE bleibt unveraendert offen, waechst durch diese Unit nicht (keine neuen
      Permission-Keys, nur zusaetzliche OR-Bedingungen in bestehenden Guards).
  (3) TOKEN-GROESSE (Iteration 14-16) bleibt unveraendert offen, waechst durch diese Unit nicht.

## Iteration 19 — p2c-inventar-einkauf-produktion-vertraege — done — 2026-08-01 (Nachtlauf 3)
- commit: 07004000
- gebaut: `RequirePermissionAny(alt, neu)` additiv auf inventar-, einkauf-, produktion- und
  vertraege-Routen. Alle vier Module hatten aus fruehren Migrationen (000084/000185/000241
  inventar, 000086/000209 einkauf, 000088/000191 produktion, 000090 vertraege) bereits
  granulare `modul:subjekt:read/write`-Guards statt eines einzigen groben Legacy-Keys — anders
  als bei wiki/zeiterfassung/helpdesk (Iteration 15/16) oder crm/finance (Iteration 17/18). Reads
  entsprechen deshalb schon 1:1 dem Katalog-Key und blieben unveraendert; getightened wurden nur
  die Writes, wo der Katalog eine feinere Aufteilung als das bestehende "write" vorschreibt.
  Per SQL gegen die laufende DB verifiziert (nicht geraten): saemtliche 30 referenzierten
  Katalog-Keys der vier Module existieren bereits in `permissions` (aus `p1a-migration`, die
  den gesamten FE-Katalog seedet) — keine eigene Migration noetig.
  inventar (`route_inventar.go`): `item:create/edit/delete` additiv auf das bestehende
  `item:write`; `movement:adjust` (AdjustStock) getrennt von `movement:create` (RecordMovement) —
  beide liefen bisher unter demselben `item:write` bzw. `movement:write`, aber das FE gated die
  "Korrektur"-Bewegungsart per eigenem Fine-Switch (`InventarPage.tsx:403`, Kommentar "corrections
  rewrite stock without a movement reason"); `attachment:manage` auf Attachment-Create+Delete;
  `inventur:create` auf CreateInventurSession; `inventur:count` auf BEIDE Endpoints, die
  `InventurSessionCard` gated (Status-Patch fuer die Zaehlen-Uebergaenge UND Counts-Upsert);
  `inventur:book` auf BookInventurDifferences. `export:run` hat KEINEN Backend-Anker — alle drei
  FE-Export-Buttons (Artikel/Bewegungen/Inventur) bauen die CSV client-seitig aus bereits
  geladenen Daten (`inventar-export.ts`), `useStockReport`/`getExportUrl` sind tote Hooks;
  `GetStockReport`/`ExportInventory`-Routen bleiben deshalb unveraendert. Legacy-only (kein
  FE-Aufrufer, verifiziert per Grep): TransferStock, alle vier Warning-Handler (Katalog hat
  gar kein `inventar:warning:*`), DeleteInventurSession, Location-Writes (Katalog hat nur
  `location:read`).
  einkauf (`route_einkauf.go`/`route_einkauf_extended.go`): `po:create/edit/delete` additiv;
  `po:send` UND `po:approve` oeffnen denselben Submit-Endpoint (OrderDetailModal ruft
  `submitPOMutation` sowohl fuers "Senden"-Button als auch den Genehmigungs-Banner — dritte
  Instanz des "zwei FE-Aktionen, ein Endpoint"-Musters aus Iteration 18); `po:cancel`;
  `po:receive` oeffnet BEIDE Endpoints, die `WareneingangDialog` nutzt (ReceiveGoods +
  PartialReceive); `supplier:create/edit`; `supplier:deactivate` auf DeleteSupplier (FE-Label
  "Deaktivieren" ist ein Soft-Delete-Wortlaut ueber dem echten DELETE-Call, per Code verifiziert);
  `rating:create` (Ressourcen-Sprung: legacy war `supplier:write`, Katalog ist `rating:create`,
  wie bei den crm/finance-Kreuz-Faellen aus Iteration 18); `contract:call` auf CreateContractCall
  (ebenfalls Ressourcen-Sprung von `contract:write`). Legacy-only: PO-Lines (kein `line`-Subjekt
  im Katalog), Catalog-CRUD und Framework-Contract-CRUD (Katalog hat nur `catalog:read`/
  `contract:read`, kein write), Rating-Delete (Hook `useDeleteSupplierRating` existiert, hat aber
  keinen FE-Aufrufer). `einkauf:export:run` ebenfalls ohne Backend-Anker (PDF-Export in
  `EinkaufDetailModals.tsx` ist client-seitig via `buildPOPdf`).
  produktion (`route_produktion.go`/`route_produktion_ext.go`): `order:create/edit/delete/
  start/complete/cancel` additiv; `bom:create/edit` (BOM-Delete hat keinen FE-Aufrufer);
  `machine:manage` oeffnet SOWOHL CreateMachine ALS AUCH UpdateMachine (Katalog hat keine
  separate create/edit-Aufteilung fuer Maschinen, ein einziger Fine-Switch fuer beides, verifiziert
  gegen `ProduktionPage.tsx` Neu-Button UND `MachineDetailModal`-Statuswechsel); `quality:create`;
  `workstep:edit` auf UpdateWorkStep (Schritt vorwaerts schalten in `ProduktionDetailModals.tsx`).
  Legacy-only: Machine Bookings und Production Plans (Katalog kennt beide Subjekte gar nicht),
  WorkStep-Create/Delete, Machine-Delete, BOM-Delete. `produktion:export:run` wieder ohne
  Backend-Anker (Laufkarten-PDF client-seitig via `buildOrderPdf`, BOM-CSV via
  `buildBomItemsCsv`) — durchgaengiges Muster ueber alle drei Module: kein einziges
  `<modul>:export:run` im Katalog hat je einen Backend-Endpoint gegated, weil jeder Export-Button
  im FE aus bereits geladenen React-Query-Daten baut statt einen eigenen Endpoint aufzurufen.
  vertraege (`route_vertraege.go`): `contract:create` additiv; `contract:edit` UND
  `contract:terminate` oeffnen denselben UpdateContract-Endpoint (Kuendigungsdialog setzt nur
  `status=terminated` + Grund ueber denselben RPC wie ein normales Edit, viertes Beispiel des
  Mehrfach-Key-Musters); `contract:delete`. Legacy-only: Parties, Reminders, Export, Signature
  (Katalog kennt fuer `vertraege` ausschliesslich das `contract`-Subjekt).
  Test: `route_capability_guard_test.go` um vier neue Router-Setups (`inventarRouter`,
  `einkaufRouter`, `produktionRouter`, `vertraegeRouter`) und 71 neue Faelle erweitert. Da alle
  vier Module hinter einem Feature-Flag registrieren (`flags.IsEnabled("modules.<x>")`), brauchten
  sie — anders als crm/biz — eigene Test-Konstruktoren (`newInventarRoutes` etc. in den jeweiligen
  `route_<modul>_test.go`, Muster von `newHelpdeskRoutes`/`newWikiRoutes` uebernommen), die den
  Flag ueber `featureflag.NewRegistry().Load(...)` erzwingen — sonst haette `RegisterRoutes` still
  gar keine Route registriert und jeder Testfall waere an einem 404 vorbeigelaufen, nicht am Guard.
- gate: `go build -p 2 ./internal/gateway/... ./internal/middleware/... ./cmd/gateway/...` ok |
  vet ok | lint ok (0 issues auf gateway+middleware) | test ok mit `DATABASE_URL` gegen
  `kmuhub_app`: `./internal/gateway/...` (`TestCapabilityGuards_AdditiveWiring` alle Faelle gruen
  inkl. 71 neuer, `TestOpenAPIRouteDrift` 739 Routen/741 Pfade — unveraendert, keine neue Route)
  und `./internal/middleware/...` gruen, 0 Skips. migration n.a. (alle referenzierten Keys per
  SQL gegen die laufende DB verifiziert: vorhanden aus fruehen Modul-Migrationen + p1a-migration).
  openapi n.a. (keine neue Route, nur Guards ausgetauscht). rls-smoke n.a. (keine Tabelle/Policy
  angefasst). gofmt: alle sechs geaenderten Route-Dateien von `gofmt -l` gemeldet — Diff-Pruefung
  (CRLF strippen auf temporaeren Kopien, dann `gofmt -l` erneut) bestaetigt reinen
  CRLF-Checkout-Artefakt (identisch zum Fehlalarm aus Iteration 15-18, `core.autocrlf=true`),
  committeter Inhalt ist LF und gofmt-clean. Testdateien waren bereits LF und wurden nicht
  gemeldet.
- verify vorgaenger: `f534ef8e` (p2c-work-documents-crm-finance-2, crm+finance) gegen die acht
  Fehlerklassen geprueft. Diff: `route_crm.go`/`route_biz.go`/`route_capability_guard_test.go`.
  Alle entfernten `RequirePermission(...)`-Direktaufrufe wurden durch benannte
  `RequirePermissionAny(alt, neu)`-Variablen ersetzt (additiv, kein Key verengt), Testfaelle
  substantiell (76 neue Faelle inkl. Kreuz-Ressourcen- und Mehrfach-Key-Faelle). Kein Proto/
  Migration/neue Route/Handler-Direktzugriff/Stub in diesem Commit, Fehlerklassen 1/2/3/5/6/7
  n.a. Kein Fund.
- offen fuer Luke:
  (1) `p2c-schichten-fuhrpark-vermietung-rapporte` ist die naechste ziehbare Unit in Block B
      (deps bereits erfuellt).
  (2) Durchgaengiger Fund ueber jetzt sieben Module (crm, finance, work, documents, inventar,
      einkauf, produktion): das `<modul>:export:run`-Fine-Switch-Muster im Katalog hat in KEINEM
      der drei Industrie-Module (inventar/einkauf/produktion) einen Backend-Endpoint hinter sich —
      alle Export-Buttons bauen client-seitig aus geladenen Daten. Falls das absichtlich so bleibt
      (Server-Export nie gebraucht), waere Aufraeumen des toten `useStockReport`-Hooks und der
      toten `getExportUrl`-Funktion in `inventar-client.ts` ein Kandidat fuer `/lean-debt` — kein
      Blocker, nur Karteileiche.
  (3) TOKEN-GROESSE (Iteration 14-16) bleibt unveraendert offen, waechst durch diese Unit nicht
      (keine neuen Permission-Keys, nur zusaetzliche OR-Bedingungen).

## Iteration 20 — p2c-schichten-fuhrpark-vermietung-rapporte — done — 2026-08-01 20:17 (Nachtlauf 3)
- commit: (siehe naechster `git log`, wird nach diesem Eintrag erstellt)
- gebaut: additive `RequirePermissionAny(legacy, catalogue)`-Guards fuer schichten (shift:publish,
  assignment:manage, template:manage), fuhrpark (vehicle:manage, service/damage/fuel/trip:create),
  vermietung (object create/edit/delete, rental create/edit/cancel/handover, inspection:create) und
  rapporte (report create/edit/delete). Alle 19 referenzierten Keys vorab per SQL gegen die lokale
  DB verifiziert (aus `p1a-migration`/000256 geseedet) — keine neue Migration noetig.
  schichten (`route_schichten.go`): `shift:publish` auf PublishShifts; `assignment:manage` auf
  AssignEmployee UND UnassignEmployee (ShiftDetailModal.tsx `onUnassign`); `template:manage` auf
  Create/Update/DeleteTemplate (alle drei haengen an `getTemplateActions`, komplett hinter
  `canTemplateManage` versteckt). ZWEI CROSS-RESOURCE-FAELLE (Legacy-Verb bleibt, Zusatz-Key kommt
  von einer ANDEREN Ressource, analog zum Kanban-Status-Fund aus Iteration 17): CreateShift ist nur
  ueber den "Schicht zuweisen"-Dialog erreichbar (`handleAssignSubmit` legt bei fehlender
  Datum+Vorlage-Kombination erst die Shift an, dann die Zuweisung) — additiver Key ist
  `schichten:assignment:manage`, nicht ein `shift:*`-Key; ApplyTemplate liegt komplett hinter
  `canTemplateManage`, additiver Key ist `schichten:template:manage` bei Legacy `schichten:shift:write`.
  Swap-Requests (Create/Approve/Reject) nutzen bereits die feinen Katalog-Keys direkt (`create`/
  `approve`, kein grobes `write` je existiert) — unangetastet.
  fuhrpark (`route_fuhrpark.go`): `vehicle:manage` NUR auf CreateVehicle (kein FE-Aufrufer fuer
  Update/Delete, verifiziert — `FuhrparkDetailModals.tsx` hat keine Mutation-Hooks, `FuhrparkPage.tsx`
  ruft nie `useUpdateVehicle`/`useDeleteVehicle`); `service:create`/`damage:create`/`fuel:create`/
  `trip:create` je auf den einzigen Create-Endpoint ihrer Ressource. Update/Delete/Complete/Resolve
  fuer alle vier Ressourcen legacy-only, weil der Katalog dafuer gar keinen Key hat (nicht weil kein
  FE-Aufrufer existiert — bei service/fuel/trip GIBT es Update/Delete-Aufrufer im FE, aber keinen
  passenden Katalog-Key, also Guard unveraendert per Vorgabe (d)). `fuhrpark:document:*` hat ueberhaupt
  keinen Katalog-Key, GPS-Guards bleiben unangetastet (gps:read matcht 1:1, gps:write hat keinen
  Katalog-Key, gps:read bewusst NICHT grosszuegig erweitern lt. Backlog-Notiz). `fuhrpark:export:run`
  wieder ohne Backend-Anker (CSV/PDF laufen client-seitig aus geladenen Daten).
  **FUND (Routing-Bug, nicht RBAC):** `/vehicles/{id}` war ZWEIMAL gemountet — einmal verschachtelt
  unter `r.Route("/vehicles", ...)` (GetVehicle/UpdateVehicle/DeleteVehicle/History/Services/Damages)
  und ein zweites Mal als eigener Top-Level-Block "Vehicle sub-resources (extended)"
  (Fuel-Logs/Trip-Logs/Documents). chi dispatcht bei zwei Mounts auf demselben Pattern nur zum
  ZULETZT registrierten — dadurch waren GetVehicle, UpdateVehicle, DeleteVehicle, GetVehicleHistory,
  ListVehicleServices, ScheduleService, ListVehicleDamages und ReportDamage in Produktion komplett
  unerreichbar (404), obwohl `chi.Walk` sie brav als registriert auflistete (Dispatch != Walk-Baum).
  Gefunden durch den neuen Capability-Guard-Test (erster Test, der die echte chi-Route ueber
  `ServeHTTP` statt direkten Handler-Aufruf ausloest — die Bestandstests riefen `routes.HandleX()`
  immer direkt auf und haben das Mounting nie geprueft). Bug ist vorbestehend (verifiziert: identische
  Struktur bereits im Commit vor dieser Iteration), nicht durch RBAC-Arbeit eingefuehrt. Behoben im
  selben Commit durch Zusammenlegen beider Bloecke in EINEN `r.Route("/{id}", ...)`-Mount — noetig,
  weil sonst `service:create`/`damage:create` (meine eigenen neuen Guards) unerreichbar geblieben
  waeren UND weil es ein launch-kritischer Funktionsbug ist (Fahrzeug-Detail/-Update/-Loeschen,
  Wartungshistorie, Schadensliste alle kaputt). Routenzahl unveraendert (739/741 vor und nach dem
  Merge — der Merge aendert nur, WELCHER Mount gewinnt, nicht welche Pfade existieren).
  vermietung (`route_vermietung.go`): `object:create/edit/delete` additiv, matcht 1:1 auf
  `canCreateObject`/`canEditObject`/`canDeleteObject` in `VermietungPage.tsx`; `rental:create/edit`
  additiv; `rental:cancel` auf DeleteRental (RentalDetailModal.tsx "Stornieren"-Button ist ein
  Soft-Label ueber dem echten DELETE-Call, per Code verifiziert — viertes Beispiel dieses Musters
  nach einkauf/supplier, vertraege/contract); `rental:handover` oeffnet SOWOHL StartRental ALS AUCH
  EndRental (beide Buttons in `RentalDetailModal.tsx` teilen sich den einen `canHandoverRental`-Gate,
  kein separates Katalog-Verb fuer Start vs. Ende); `inspection:create` auf CreateInspection
  (ZustandsprotokollDialog.tsx). Signature hat keinen Katalog-Key, legacy-only. `vermietung:export:run`
  wieder ohne Backend-Anker (client-seitig).
  rapporte (`route_rapporte.go`): `report:create/edit/delete` additiv (edit/delete sind FE-seitig
  `useScopedCapability`-gated auf `report.authorId` — der eigentliche own-Scope-Listenfilter ist
  separate Zukunfts-Unit `p2-own-scope-list-filter`, hier nur die Guard-Verdrahtung). Approve/Reject
  nutzen bereits den feinen `approve`-Key direkt, unangetastet. Submit/Lines/Attachments/Signature
  haben keinen Katalog-Key, legacy-only. **FUND:** `rapporte:measurement:read`/`:manage` haben TROTZ
  existierendem Katalog-Key KEINEN Backend-Anker — der komplette "Aufmass"-Tab (`RapportePage.tsx`)
  laeuft auf einem rein lokalen Zustand-Store (`stores/rapporte.ts`, `MOCK_MEASUREMENTS`, kein
  einziger API-Call fuer Create/Delete/AddPosition) — analog zum Advisory-Protokoll-Fund aus
  Iteration 18 (Katalog-Key real, FE-Consumer ist ein No-op-Mock). Measurement-Routen bleiben
  komplett legacy-only. `rapporte:export:run` ebenfalls ohne Backend-Anker (PDF client-seitig via
  `buildReportPdf`, obwohl ein echter `HandleExportPDF`/`GET /export/pdf`-Endpoint existiert — der
  wird vom FE schlicht nie aufgerufen).
  Test: `route_capability_guard_test.go` um vier neue Router-Setups (`schichtenRouter`,
  `fuhrparkRouter`, `vermietungRouter`, `rapporteRouter`) und 58 neue Faelle erweitert. Alle vier
  Module sind feature-flag-gated, brauchten also eigene Test-Konstruktoren (`newSchichtenRoutes` etc.
  in den jeweiligen `route_<modul>_test.go`, Muster von `newInventarRoutes` uebernommen).
- gate: `go build -p 2 ./internal/gateway/... ./internal/middleware/... ./cmd/gateway/...` ok |
  vet ok | lint ok (0 issues) | test ok mit `DATABASE_URL` gegen `kmuhub_app`:
  `./internal/gateway/...` komplett gruen (`TestCapabilityGuards_AdditiveWiring` alle Faelle inkl.
  58 neuer, `TestOpenAPIRouteDrift` 739 Routen/741 Pfade — unveraendert), 0 Skips. migration n.a.
  (alle Keys per SQL gegen die laufende DB verifiziert). openapi n.a. (keine neue Route, nur Guards
  + der Mount-Merge, der KEINE neuen Pfade registriert). rls-smoke n.a. (keine Tabelle/Policy
  angefasst). gofmt: `route_schichten.go`/`route_fuhrpark.go`/`route_vermietung.go`/`route_rapporte.go`
  von `gofmt -l` gemeldet — CRLF-Checkout-Artefakt (identisch zum Fehlalarm aus Iteration 15-19),
  nach CRLF-Strip clean. `route_schichten_test.go` bleibt auch NACH CRLF-Strip gemeldet
  (Map-Literal-Spaltenausrichtung ab Zeile 111) — verifiziert VORBESTEHEND im Commit vor dieser
  Iteration (`git show HEAD:...` + CRLF-Strip zeigt denselben Drift), nicht durch diese Iteration
  eingefuehrt, nicht angefasst (ausserhalb Scope).
- verify vorgaenger: `07004000` (p2c-inventar-einkauf-produktion-vertraege) gegen die Fehlerklassen
  geprueft. Diff: sechs Route-Dateien + `route_capability_guard_test.go`, kein Proto/Migration/neue
  Route/Handler-Direktzugriff/Stub. Alle entfernten `RequirePermission(...)`-Direktaufrufe wurden
  additiv durch `RequirePermissionAny(alt, neu)` ersetzt. Stichprobe: alle 36 referenzierten
  Capability-Keys per SQL gegen die laufende DB verifiziert (vorhanden). Kein Fund.
- offen fuer Luke:
  (1) `p2c-dialer-berichte-formulare-automatisierung` ist die naechste ziehbare Unit in Block B
      (deps bereits erfuellt).
  (2) ROUTING-BUG-MUSTER: der `/vehicles/{id}`-Doppel-Mount in fuhrpark koennte sich in anderen
      Modulen wiederholen, wenn dort ebenfalls ein "erweiterter" Routen-Block spaeter unter
      demselben Prefix wie ein frueherer verschachtelter Block gemountet wurde. Bisher nur in
      fuhrpark gefunden (durch den ersten echten `ServeHTTP`-Dispatch-Test in diesem Modul) — ein
      gezielter Grep/Audit auf "gleiches Mount-Pattern zweimal in derselben `RegisterRoutes`" ueber
      alle Gateway-Route-Dateien waere ein guter Kandidat fuer eine eigene Unit, da Bestandstests
      (die immer `routes.HandleX()` direkt aufrufen) so etwas grundsaetzlich nicht faengen.
  (3) `rapporte:measurement:*` und `rapporte:export:run` haben trotz Katalog-Keys keinen
      Backend-Anker (Mock-Store bzw. Client-Export) — analog zu den bereits bekannten toten
      Katalog-Keys aus Iteration 17/18/19. Kein Blocker, nur zur Kenntnis.
  (4) TOKEN-GROESSE (Iteration 14-16) bleibt unveraendert offen, waechst durch diese Unit kaum
      (nur zusaetzliche OR-Bedingungen, keine neuen Permission-Keys).

## Iteration 21 — p2c-dialer-berichte-formulare-automatisierung — done — 2026-08-01 20:35 (Nachtlauf 3)
- commit: (siehe naechster `git log`, wird nach diesem Eintrag erstellt)
- gebaut: additive `RequirePermissionAny(legacy, catalogue)`-Guards fuer dialer, berichte, formulare
  und automatisierung. Alle 19 neu referenzierten Katalog-Keys vorab per SQL gegen die lokale DB
  verifiziert (aus `p1a-migration`/000256 geseedet) — keine neue Migration noetig. Damit ist der
  komplette p2c-Modul-Sweep aus Block B (12 Industrie-/Querschnitt-Module) abgeschlossen.
  dialer (`route_dialer.go`): einziges Modul, dessen Legacy-Ressourcen schon in `modul:subject`-Form
  seedeten (`dialer:campaigns`, `dialer:calls`, `dialer:agent`, `dialer:outcomes`) — read/calls:write/
  agent:manage/outcomes:manage konkatenieren dadurch 1:1 auf ihren Katalog-Key und blieben unangetastet.
  Einzige echte Luecke: `dialer:campaigns:write` (Legacy) vs. `dialer:campaigns:manage` (Katalog) —
  additiv auf alle Campaign-Management-Routen. **FUND (Cross-Resource):** Skip/Requeue teilen sich
  EINEN Handler zwischen zwei FE-Oberflaechen mit unterschiedlichen Capabilities — dem Agenten-Live-
  Workspace (`DialerWorkspacePage.tsx`, `dialer:calls:write`, unveraendert) und der Supervisor-Queue-
  Tabelle im Kampagnen-Detail (`ContactQueueTable.tsx`, Kommentar "Derived from dialer:campaigns:manage",
  verifiziert bis `CampaignDetailPage.tsx:247 canManage={canManageCampaigns}`). Ohne Erweiterung haette
  ein Manager mit `campaigns:manage` aber ohne `calls:write` 403 auf Skip/Requeue aus der eigenen
  Kampagnen-Ansicht bekommen — additiv `dialer:campaigns:manage` als zweite Alternative ergaenzt.
  `GetNextContact` bleibt unangetastet (nur vom Workspace aufgerufen, kein Supervisor-Pfad).
  berichte (`route_berichte.go`): `reports:create/edit/delete` additiv nach HTTP-Verb (POST/PATCH/
  DELETE) auf definitions UND documents. `documents/{id}` PATCH traegt sowohl Content-Edits als auch
  den Status-Lifecycle (`ReportDocumentEditor.tsx transitionTo()` ruft denselben UpdateDocument-Call
  wie ein normaler Save) — additiv DREI Alternativen (write/edit/publish). `schedule:manage` additiv
  auf alle vier Schedule-Routen, `share:manage` additiv auf Share-Create/-Revoke (Kommentar im Code
  bestaetigt: Minting = write-aequivalent). **FUND:** `/definitions/{id}/export` wird von ZWEI FE-
  Stellen aufgerufen — `DatevView.tsx` mit explizitem `canExport = useHasCapability('berichte:export:
  run')`-Gate, `MyReportsLibrary.tsx` (`exportAs()`) OHNE jedes Capability-Gate. Der Legacy-Guard
  (`reports:read`) ist damit ein striktes Superset von `export:run` — additiv beide Keys ergaenzt
  (macht praktisch aktuell nichts, da read bereits alles abdeckt), aber die Additiv-Regel verbietet
  ausdruecklich das Verengen von `read` auf `export:run` (JWT-Bake-in, Aussperr-Risiko) — ein
  `readonly`-User mit `reports:read`, aber ohne `export:run`, kann den Export-Endpoint aktuell trotzdem
  direkt per API aufrufen, obwohl die FE-Absicht ("readonly sieht, laedt nie herunter", Kommentar in
  DatevView.tsx) das verhindern will. Kein Fix in dieser Unit moeglich (additiv-only), nur dokumentiert.
  `datev:read` hat weiterhin keinen eigenen Backend-Anker (reiner Tab-Gate, Bestand aus Iteration-
  Notiz). Cache-Invalidierung und der Dokument-PDF-Export haben keinen Katalog-Verb-Aequivalent — die
  PDF-Route hat zusaetzlich GAR KEINEN FE-Aufrufer (nur `window.print()` ist verdrahtet, laut
  Code-Kommentar in `route_berichte.go` selbst) — beide legacy-only.
  formulare (`route_formulare.go`): `schemas:create/edit/delete` additiv nach Verb. PATCH traegt wie
  bei berichte sowohl Content-Edit als auch den Draft/Active/Closed/Archived-Toggle (`canSchemasPublish`
  ruft denselben `updateSchema`-Call wie `canSchemasEdit`) — additiv edit+publish. Duplicate ist laut
  FE `canSchemasCreate`-gated (nicht edit) — additiv `schemas:create`. `submissions/export` additiv
  `export:run` (echter Backend-Anker, `FormularePage.tsx canExportRun`), gleiche Read-ist-Superset-
  Einschraenkung wie bei berichte. **FUND (fehlende Route, kein Wiring-Fall):** `formulare:share:manage`
  gated in `FormularePage.tsx` einen echten Button ("Share opens the link-create dialog — that IS share
  management"), der `useCreateShareLink`/`useUpdateShareLink`/`useDeleteShareLink` aufruft — diese
  Hooks rufen `{BASE}/schemas/{id}/share-links` und `{BASE}/share-links/{id}`
  (`formulare-client.ts:183-209`), und KEINE dieser Routen existiert in `route_formulare.go`. Anders
  als die bisherigen "toter Katalog-Key"-Funde (FE-Aufrufer fehlt) ist hier der FE-Aufrufer real und
  die Route fehlt — ein Feature-Gap, kein additiver Guard-Fall, also nichts zu verdrahten. Webhooks
  bleiben komplett legacy-only (Katalog-Notiz: FE-UI ist ein unmontierter Stub, noch kein Key).
  automatisierung (`route_automation.go`): einziges Modul, dessen Legacy-Ressource `automations` OHNE
  Modul-Praefix seedete (vor 000129) — dadurch konkateniert KEIN einziger Legacy-Key zufaellig auf
  seinen Katalog-Key (anders als bei allen anderen elf p2c-Modulen), jede Route brauchte den additiven
  Wrapper. Triggers/Actions/Templates haben kein eigenes Katalog-Verb: additiv auf `automations:read`
  (Triggers/Actions, aus `AutomationDetailModal.tsx`, nur nach Oeffnen aus der bereits-`read`-gated
  Liste erreichbar) bzw. `automations:create` (Templates-Tab, `AutomatisierungPage.tsx` Zeile 52:
  `templates: 'automatisierung:automations:create'` — Vorlagen durchstoebern ist nur mit
  Instanziierungsrecht sinnvoll). Test-Condition/Dry-Run laufen aus `AutomationWizard.tsx`, der laut
  eigenem Doc-Kommentar in `AutomationDetailModal.tsx` sowohl fuer Neu-Anlage als auch fuer den
  "Bearbeiten"-Hand-off wiederverwendet wird — additiv `create` UND `edit`. Stats bleibt auf
  `RequireRole("admin")`, ein komplett anderer Mechanismus ohne Katalog-Aequivalent — unangetastet.
  Test: `route_capability_guard_test.go` um vier neue Router-Setups (`dialerRouter`, `berichteRouter`,
  `formulareRouter`, `automationRouter`) und 59 neue Faelle erweitert. `formulare` brauchte einen
  neuen Test-Konstruktor (`newFormulareRoutes` in neuer `route_formulare_test.go`, Muster von
  `newSchichtenRoutes` uebernommen — bislang existierten fuer formulare nur Handler-Unit-Tests, kein
  eigener Router-Konstruktor). `berichte` nutzt den bereits vorhandenen `berichteFlagsON()`-Helper aus
  `route_berichte_test.go`. `dialer`/`automatisierung` sind nicht feature-flag-gated, direkter
  Konstruktor-Aufruf ohne Wrapper.
- gate: `go build -p 2 ./internal/gateway/... ./internal/middleware/... ./cmd/gateway/...` ok |
  vet ok | lint ok (0 issues) | test ok mit `DATABASE_URL` gegen `kmuhub_app`:
  `./internal/gateway/...` komplett gruen (`TestCapabilityGuards_AdditiveWiring` alle Faelle inkl.
  59 neuer, `TestOpenAPIRouteDrift` 739 Routen/741 Pfade — unveraendert), 0 Skips. migration n.a.
  (alle 19 neuen Keys per SQL gegen die laufende DB verifiziert). openapi n.a. (keine neue Route, nur
  Guards). rls-smoke n.a. (keine Tabelle/Policy angefasst). gofmt: `route_dialer.go`/
  `route_berichte.go`/`route_formulare.go`/`route_automation.go` von `gofmt -l` gemeldet —
  CRLF-Checkout-Artefakt (identisch zum Fehlalarm aus Iteration 15-20, `git show HEAD:...` liefert
  reines ASCII/LF fuer alle vier Dateien, git diff --stat zeigt gezielte 31-53-Zeilen-Diffs statt
  vollstaendiger Umschreibung — kein echter Format-Fund).
- verify vorgaenger: `3778c05d` (p2c-schichten-fuhrpark-vermietung-rapporte) gegen die acht
  Fehlerklassen geprueft. Diff: vier Route-Dateien + `route_capability_guard_test.go`, kein Proto/
  Migration/Handler-Direktzugriff/Stub. Alle entfernten `RequirePermission(...)`-Direktaufrufe wurden
  additiv durch `RequirePermissionAny(alt, neu)` ersetzt (Fehlerklasse 8 n.a.). Der im selben Commit
  behobene Doppel-Mount-Bug (`/vehicles/{id}`) ist als vorbestehend dokumentiert und veraendert die
  Routenzahl nicht (739/741 vor und nach dem Merge, Fehlerklasse 7 n.a.). Kein Fund.
- offen fuer Luke:
  (1) Damit ist der komplette p2c-Modul-Sweep (12 Industrie-/Querschnitt-Module: work/documents/crm/
      finance/inventar/einkauf/produktion/vertraege/schichten/fuhrpark/vermietung/rapporte/dialer/
      berichte/formulare/automatisierung) abgeschlossen. Naechste ziehbare Unit in Block B ist
      `p2-owner-fk-crm` (deps `[p1a-migration]`, bereits erfuellt) — Voraussetzung fuer
      `p2-own-scope-list-filter` danach.
  (2) `berichte:export:run`/`formulare:export:run` sind additiv verdrahtet, aber wirkungslos: der
      Legacy-Guard (`reports:read`/`submissions:read`) ist ein striktes Superset, additiv-only kann
      das nicht verengen. Ein `readonly`-User kann den Export-Endpoint aktuell per direktem API-Call
      umgehen, obwohl die FE-Absicht das verhindern will (DatevView.tsx-Kommentar "readonly sieht,
      laedt nie herunter"). Braucht einen bewussten harten Cutover (Token-Rotation abwarten), kein
      additiver Fix.
  (3) `formulare:share:manage` hat einen echten FE-Aufrufer (`FormularePage.tsx` "Teilen"-Aktion ->
      `useCreateShareLink`/`useUpdateShareLink`/`useDeleteShareLink`), aber die aufgerufenen Routen
      (`/schemas/{id}/share-links`, `/share-links/{id}`) existieren nicht in `route_formulare.go` —
      ein Feature-Gap (Endpoint fehlt komplett), kein additiver Guard-Fall. Kandidat fuer eine eigene
      `g-formulare-share-links`-Unit, falls das Feature launch-relevant ist.
  (4) TOKEN-GROESSE (Iteration 14-16) bleibt unveraendert offen, waechst durch diese Unit kaum (nur
      zusaetzliche OR-Bedingungen + 6 neue Basis-Keys `automatisierung:automations:read/create/edit/
      delete/toggle` + `executions:read`, die aber schon seit `p1a-migration` im Katalog-Seed stehen).

## Iteration 22 — p2-owner-fk-crm — done — 2026-08-01 21:05 (Nachtlauf 3)
- commit: (siehe naechster `git log`)
- gebaut: kein Produktionscode geaendert — die Unit war beim Recherchieren bereits erfuellt.
  `created_by UUID NOT NULL REFERENCES users(id)` existiert an `contacts` (000007) und `deals`
  (000009) seit den CRM-Basis-Migrationen, lange vor RBAC. Die Befuellung aus dem Auth-Context ist
  bereits durchgaengig verdrahtet: `HandleCreateContact`/`HandleCreateDeal`
  (`route_crm_contacts.go`/`route_crm_pipeline.go`) setzen `CreatedBy: middleware.GetUserID(r.
  Context())` auf den gRPC-Request, NIE aus dem Body — kein Client-Spoofing moeglich. Von dort laeuft
  es unveraendert durch `CreateInput.CreatedBy` -> `models.{Contact,Deal}.CreatedBy` -> Repository-
  INSERT. `backend-gaps.md:54` ("CRM contact/deal tragen KEIN owner_id/created_by") bezog sich
  offenbar auf fehlende Wire-Exposure ans FE, nicht auf die DB-Spalte — fuer serverseitiges
  `own`-Filtern (naechste Unit) reicht die WHERE-Klausel auf der Repo-Query, ein FE-sichtbares Feld
  ist dafuer nicht Voraussetzung.
  Einzige echte Luecke gegen `done_when`: kein Test bewies den Round-Trip. Geschlossen mit vier
  Assertions in bestehenden Tests (kein neuer Testfall, keine neue Infrastruktur): `TestContactWrites_
  LandInCallerTenant`/`TestDealWrites_LandInCallerTenant` (`tenant_write_test.go`, echte DB als
  `kmuhub_app`) pruefen nach Create+GetByID jetzt zusaetzlich `got.CreatedBy == userID`;
  `TestService_Create_Success` in beiden Paketen prueft zusaetzlich `contact.CreatedBy`/`deal.
  CreatedBy == input.CreatedBy` auf Service-Ebene (Mock-Repo).
- gate: `go build -p 2 ./internal/crm/... ./internal/gateway/... ./cmd/gateway/...` ok | vet ok |
  lint ok (0 issues) | test ok mit `DATABASE_URL` gegen `kmuhub_app`: alle 12 `internal/crm/*`-Pakete
  gruen, die beiden geaenderten DB-Tests einzeln verbose gegengeprueft (PAUSE/CONT sichtbar, kein
  SKIP). migration n.a. (keine Schemaaenderung). openapi n.a. (keine Route). rls-smoke n.a. (keine
  Tabelle/Policy angefasst, bestehende RLS-Write-Tests nur um eine Feldassertion erweitert). gofmt:
  `contact/tenant_write_test.go`/`deal/tenant_write_test.go` von `gofmt -l` gemeldet — CRLF-Checkout-
  Artefakt (Muster aus Iteration 15-21, `git show HEAD:...` liefert reines LF). `deal/service_test.go`
  ebenfalls gemeldet, aber verifiziert VORBESTEHEND: `git show HEAD:...` durch `gofmt -l` gejagt
  meldet denselben Import-Reihenfolge-Fund (`errors` nach den externen Imports) unabhaengig von diesem
  Commit — nicht in dieser Iteration eingefuehrt, nicht angefasst (Scope-Disziplin).
- verify vorgaenger: `c3721884` (p2c-dialer-berichte-formulare-automatisierung) gegen die acht
  Fehlerklassen geprueft. Diff: vier Route-Dateien + `route_capability_guard_test.go` + neue
  `route_formulare_test.go`, kein Proto/Migration/Handler-Direktzugriff/Stub. Alle ersetzten
  `RequirePermission(...)`-Direktaufrufe wurden additiv durch `RequirePermissionAny(alt, neu[,
  neu2])` erweitert, kein Alt-Key verloren (Fehlerklasse 8 n.a.). Kein Fund.
- offen fuer Luke:
  (1) `p2-own-scope-list-filter` (naechste Unit, deps erfuellt) muss fuer deals entscheiden, ob
      `own`-Scope gegen `created_by` (Ersteller) oder `owner_id` (Business-Owner, vom Client im
      Request settable) filtert — das ist eine Produktentscheidung, keine Code-Luecke. Bei contact
      gibt es nur `created_by`, keine Ambiguitaet.
  (2) `crm:pipeline:manage`/`crm:segment:override` (Iteration 18-Fund) und die komplette
      Advisory-Protokoll-Funktion bleiben FE-seitig ungegatet/Mock — unveraendert offen, nur zur
      Erinnerung, betrifft diese Unit nicht direkt.
  (3) TOKEN-GROESSE (Iteration 14-16) unveraendert offen, durch diese Unit nicht beruehrt (keine
      neuen Permission-Keys).

## Iteration 23 — p2-own-scope-list-filter — done — 2026-08-01 22:10 (Nachtlauf 3)
- commit: `78985af4` feat(rbac): filter list endpoints server-side when a grant is scoped to own
- ausgangslage: Der Scope war **zur Laufzeit nirgends lesbar**. Der JWT traegt `perms []string`
  (nur Keys), `role_permissions.scope` lag ausschliesslich in der DB, und `GetEffectivePermissions`
  wurde nur von der `/auth/me/permissions`-Route fuers Admin-UI benutzt. Die Unit war damit zu
  ~70 % Transportweg-Bau und nur zu ~30 % Filter.
- gebaut, drei Schichten:
  (1) TRANSPORT. `Service.NarrowedScopes(ctx, userID)` (`internal/auth/effective_permissions.go`)
      faehrt den vorhandenen Union-Resolver aus `p1a-resolver` und behaelt nur die Keys mit Scope
      < `all`. `createTokenPair` legt die Map in den neuen JWT-Claim `scopes` (`omitempty`).
      `middleware.PermissionScope(ctx, resource, action)` liest sie zurueck.
      **Abwesenheit = `all`** — bewusst, zweifach begruendet: (a) in einem neu ausgestellten Token
      ist ein fehlender Key per Konstruktion unbeschraenkt, (b) ein Alt-Token ohne den Claim
      verhaelt sich exakt wie bisher. Die Gegenrichtung (`own` als Default) haette JEDEM User mit
      gueltigem Alt-Token alle Listen geleert. Der Guard entscheidet weiterhin das Ob, der Scope nur
      das Wieviel — ohne Guard-Pass erreicht niemand den Handler.
      Token-Groesse: nur verengte Keys reisen. Gemessen an der lokalen DB traegt `member` 17 solche
      Keys, `admin` **null** (alle 454 Grants stehen auf `all`) — der Claim entfaellt dort ganz.
      Fehler beim Scope-Lookup ist im Gegensatz zu `GetUserRoles`/`GetUserPermissions` **fatal**
      (Login schlaegt fehl): eine leere Map liest sich als "alles unbeschraenkt".
  (2) SCOPE-KORREKTUR — CRM war der falsche Ort. Die Unit-deps zeigten auf `p2-owner-fk-crm`, aber
      **kein einziger CRM-Key traegt im Seed 000256 Scope `own`**; ein Filter auf `contacts`/`deals`
      waere toter Code gewesen (und die offene `created_by`-vs-`owner_id`-Frage aus Iteration 22
      damit gegenstandslos — sie stellt sich erst, wenn jemand einem CRM-Key `own` gibt).
      Die einzigen READ-Keys mit `own` im Seed sind `rapporte:report:read` und
      `helpdesk:ticket:read`, beide an Rolle `member`. Die uebrigen 15 `own`-Keys sind Edit-/
      Create-Keys (`work:task:edit`, `wiki:article:edit`, …) — die verengen eine Liste nicht, weil
      der Listen-Guard der Read-Key ist. `schichten:swap:read` hat `own`, ist aber **nicht
      umsetzbar**: `shift_swap_requests` fuehrt `requested_by_employee_id` (employee-ID), und ein
      user-zu-employee-Mapping existiert in der DB nicht (es gibt gar keine `employees`-Tabelle).
      Die `team:*:view`-Keys sind FE-only, im Backend nirgends verdrahtet.
  (3) FILTER an drei Endpoints. Gemeinsamer Helper `ownerFilterForScope(w, r, resource, action)
      (*string, bool)` in `internal/gateway/helpers.go` — Hausstil analog `validateUUIDParam`,
      schreibt im Fehlerfall selbst 401. Angeschlossen:
      - `GET /rapporte/reports` — `author_id`-Filter existierte bereits samt Index
        `idx_work_reports_author (tenant_id, author_id)`; **kein** Proto/Repo-Change noetig.
        Bei `own` ueberschreibt die Caller-ID ein vom Client mitgegebenes `author_id`.
      - `GET /rapporte/pending` — neues Proto-Feld `author_id`; `Service.ListPendingApprovals`
        delegierte schon an `ListReports`, also nur durchreichen, kein Repo-Change.
      - `GET /helpdesk/tickets` — neues Proto-Feld `participant_id`; Repo-Condition
        `(requester_id = $n OR assignee_id = $n)`. Ownership ist bewusst requester-ODER-assignee:
        ein Agent, der sein zugewiesenes Ticket nicht sieht, kann es nicht bearbeiten.
      Filter sitzt ueberall in der WHERE-Klausel, nie als Nachfilter — `total` und Seitengrenzen
      beschreiben dieselbe Menge wie die Zeilen.
- gate: build ok (`-p 2`, `./internal/... ./cmd/{gateway,auth,helpdesk,rapporte}/...`; `go build ./...`
  ohne `-p 2` stirbt weiterhin an OOM) | vet ok | lint ok (0 issues) | test ok mit `DATABASE_URL`
  gegen `kmuhub_app`: `./internal/{auth,middleware,helpdesk,rapporte,server}/...` + `./internal/
  gateway/` gruen, TestOpenAPIRouteDrift gruen. openapi n.a. — **keine neue Route**, nur
  Verhaltensaenderung bestehender; keine neuen Query-Parameter (der Filter ist serverseitig
  erzwungen, nicht anfragbar). migration n.a. gofmt: neun Dateien von `gofmt -l` gemeldet, alle
  CRLF-Checkout-Artefakt (LF-Kopien durch `gofmt -l` gejagt: sauber) bis auf
  `server/rapporte_grpc.go` — dort ein vorbestehender Alignment-Fund im Measurement-Mapping
  (Z. ~972), gegen `git show HEAD:` gegengeprueft, nicht von dieser Iteration, nicht angefasst.
- neue tests: `helpdesk/own_scope_list_test.go` (echte DB — 4 Tickets, own-Scope liefert 3 Zeilen
  UND total 3, Fremdticket fehlt; zusaetzlich Status+Participant kombiniert, weil der
  Platzhalter dabei von `$2` auf `$3` wandert), `rapporte/own_scope_list_test.go` (echte DB, ueber
  den Service wegen page-zu-offset-Umrechnung; deckt Liste + Genehmigungsqueue ab und prueft, dass
  die ungefilterte Queue unveraendert 2 liefert), `middleware/scope_test.go` (vier Faelle durch ein
  echtes signiertes Token: own, team, Key-nicht-verengt, Legacy-Token ohne Claim),
  `gateway/own_scope_filter_test.go` (Helper inkl. 401 ohne User-ID),
  zwei Faelle fuer `NarrowedScopes` (nur Verengtes; nil wenn nichts verengt ist).
  Beide DB-Tests seeden **eigene** Tenants (sie zaehlen alle Zeilen des Tenants) und raeumen per
  `defer`, nicht `t.Cleanup` — letzteres laeuft nach dem deferten `pool.Close()` und liess die
  Zeilen beim ersten Lauf mit "closed pool" liegen.
- rls-verifikation (per psql als `kmuhub_app`, ohne Tenant-Kontext — so laeuft der Login):
  `current_tenant_id()` ist NULL, `is_system_context()` false, und die Preset-Scopes sind trotzdem
  lesbar (17 verengte Keys fuer `member`, darunter `rapporte:report:read` und
  `helpdesk:ticket:read`, beide `own`). Die `roles`-Policy `tenant_id IS NULL OR tenant_id =
  current_tenant_id() OR is_system_context()` traegt also auch den Token-Pfad.
- verify vorgaenger: `9e29cbe8` (p2-owner-fk-crm) gegen die acht Fehlerklassen geprueft. Diff sind
  ausschliesslich vier Testdateien plus BACKLOG/JOURNAL, kein Produktionscode, kein Proto, keine
  Migration, keine Route. Kein Fund.
- offen fuer Luke:
  (1) CUSTOM-ROLLEN UND DER LOGIN-PFAD (aus der psql-Verifikation oben, betrifft Welle 1b): der
      Login laeuft ohne Tenant-Kontext, die `roles`-Policy zeigt dort nur `tenant_id IS NULL`.
      Sobald es Custom-Rollen mit gesetztem `tenant_id` gibt, sind die beim Token-Bau unsichtbar —
      ihre Permissions fehlen dann im Token (fail-closed, faellt sofort auf) und ihre Scopes auch
      (fail-open, faellt NICHT auf: der Key steht dann auf `all`). Das ist Bestandsverhalten von
      `GetUserPermissions`, kein Regress dieser Unit, aber vor Welle 1b zu loesen — sonst ist die
      erste Custom-Rolle mit `own`-Scope wirkungslos. Loesung vermutlich: Token-Bau im
      System-Kontext oder mit dem Tenant des Users fahren.
  (2) NUR LISTEN, NICHT DETAILS. `GET /rapporte/reports/{id}`, `/export/pdf`, `/stats` und der
      Ticket-Detailzugriff bleiben tenant-weit — ein `member` mit `own` kann einen fremden Rapport
      weiterhin per direkter ID oeffnen. Die Unit-Beschreibung ging davon aus, dass der
      Detailzugriff bereits 403 liefert; das tut er nicht, es gab vor dieser Unit ueberhaupt keine
      Scope-Auswertung. Braucht eine eigene Unit (Muster: Scope im Handler lesen, ID-Read gegen die
      Owner-Spalte pruefen, 404 statt 403 — sonst verraet der Statuscode die Existenz).
  (3) `schichten:swap:read` traegt `own` im Seed, ist aber mangels user-zu-employee-Mapping nicht
      erfuellbar (s.o.). Entweder Mapping nachziehen oder den Scope im Seed auf `all` korrigieren —
      aktuell verspricht die Rechteverwaltung dort eine Einschraenkung, die es nicht gibt.
  (4) `team`-Scope verhaelt sich weiterhin wie `all` (Reporting-Line-Resolver fehlt). Betrifft
      heute nur `crm:contact:create/edit` bei `member` und `wiki:article:edit` bei `manager`.
  (5) TOKEN-GROESSE (Iteration 14-16) unveraendert offen; diese Unit erhoeht sie fuer `member` um
      17 Map-Eintraege, fuer `admin` um nichts.

## Iteration 24 — fe-finance-expenses — done — 2026-08-01 21:30 (Nachtlauf 3)
- commit: 8e3ab489
- gebaut: Ausgaben-Modul komplett von der Tabelle bis zur Route. Migration 000257
  `finance_expenses` (tenant_id NOT NULL + FK auf tenants, RLS mit USING **und** WITH CHECK,
  drei Indizes, CHECK auf status und auf amount > 0); `internal/biz/expense/` (Repository-
  Interface, PostgresRepository, Service, Errors); sechs RPCs am FinanceService
  (`CreateExpense`/`ListExpenses`/`UpdateExpense`/`DeleteExpense`/`DecideExpense`/
  `AttachExpenseReceipt`) mit Regen im selben Commit; `internal/server/biz_grpc_expense.go`;
  sieben Gateway-Routen in `route_biz_expenses.go` plus openapi.yaml-Pfade und drei Schemas.
- fachliche entscheidungen (keine davon aus dem Backlog uebernommen, alle begruendet):
  - VIER-AUGEN serverseitig: `decide()` lehnt den Einreicher ab (`ErrSelfApproval`) und
    laesst nur `pending` zu. Deshalb traegt die Tabelle `submitted_by` UND `decided_by`
    getrennt — aus einem "letzter Akteur"-Feld waere die Regel nachtraeglich nicht mehr
    pruefbar. Beide Guards liegen in `decide()`, nicht in Approve/Reject, damit keiner in
    einem der beiden vergessen werden kann.
  - EDIT NACH DER ENTSCHEIDUNG: `account` (Kontierung) und `receipt_name` bleiben offen,
    Betrag/Datum/Beschreibung/Kategorie/Lieferant/Projekt nicht (`ErrDecided`). Kontierung
    passiert fachlich NACH der Genehmigung — die Seed-Daten des Mocks zeigen genau das
    (approved + account gesetzt). Ein aenderbarer Betrag haette die Genehmigung entwertet.
  - DELETE nur solange `pending`; eine getroffene Entscheidung ist ein Datensatz, kein Entwurf.
  - BELEG-UPLOAD ist bewusst NUR ein Dateiname. Der FE-Vertrag (`financeExpenseApi.
    attachReceipt`) postet `{receiptName}` als JSON, es gibt kein Upload-Widget und keinen
    Download-Link. Das Presign-Muster aus der Unit-Note waere Infrastruktur ohne Aufrufer.
    `lean:`-Marker mit Upgrade-Trigger steht im Paketkommentar von
    `internal/biz/expense/service.go`, der Verweis auf `internal/biz/gobdarchive` als
    zukuenftiger Ablageort ebenfalls.
- WIRE-SHAPE-FUND (wichtigster Punkt fuer die restlichen Block-D-Units): `response.Proto` und
  `ProtoListWrapped` sind fuer diese Endpunkte unbrauchbar. Der repo-weite protoMarshaler
  laeuft mit `UseProtoNames: true` → snake_case, und Decimals reisen im Repo als String. Der
  FE-Typ (`stores/finance.ts:304` + `finance-client.ts:434ff`) verlangt aber `receiptName`
  in camelCase und `amount` als JSON-**Zahl**. Geloest mit einer hand-gemappten
  `expenseWire`-Struct im Gateway plus zwei Tests darauf
  (`route_biz_expenses_test.go`) — inklusive der Gegenprobe, dass `tenant_id`/`submitted_by`
  NICHT in die Antwort lecken. `receipt` wird aus `receiptName` abgeleitet statt gespeichert,
  damit Liste und Detail nicht auseinanderlaufen koennen. Antwortform exakt wie der Mock:
  Liste `{expenses,total}` (leer als `[]`, nicht `null`), Einzelentitaet `{expense}`,
  DELETE `{}`.
- guards: bestehende `RequirePermission("finance", "read"/"write"/"delete")`, kein neuer Key
  und damit keine Seed-Pflicht. `capability-catalog.ts` hat ueberhaupt keinen
  `finance:expense:*`-Key — ein additives `RequirePermissionAny` haette einen Key versprochen,
  den niemand bekommt.
- gate: build ok (`-p 2`, `./internal/biz/expense/... ./internal/server/... ./internal/gateway/...
  ./cmd/biz/... ./cmd/gateway/...`) | vet ok | lint ok (0 issues) | test ok mit `DATABASE_URL`
  gegen `kmuhub_app`: `./internal/biz/expense/...` (19 Testfaelle, davon 3 echte DB-Tests mit
  9 Unterfaellen — **null Skips**, im `-v`-Lauf verifiziert), `./internal/gateway/` und
  `./internal/server/...` gruen. TestOpenAPIRouteDrift gruen: 744 registrierte Routen gegen
  746 dokumentierte Pfade. gofmt sauber auf allen neuen Dateien (die drei gemeldeten
  Bestandsdateien `cmd/biz/main.go`, `route_biz.go`, `biz_grpc.go` sind das bekannte
  CRLF-Checkout-Artefakt — ganze Datei als Diff, kein inhaltlicher Fund).
- migration: 000257 up und `down 1` und wieder up lokal sauber durchgelaufen, Kopf zurueck
  auf 257. Prod steht auf 243 — dieser Commit erhoeht den Nachhol-Stapel um eine Migration.
- rls-smoke (als `kmuhub_app`, NOSUPERUSER NOBYPASSRLS): eigener Tenant **1** Zeile, fremder
  Tenant **0**, und ein INSERT mit fremder `tenant_id` wurde von der WITH-CHECK-Klausel
  **abgelehnt** (nicht still in den falschen Tenant geschrieben). Zusaetzlich im Go-Test:
  fremde ID liest als ErrNotFound, fremde ID laesst sich nicht loeschen, und die Zeile ist
  danach fuer ihren Besitzer noch da.
- verify vorgaenger: `78985af4` (p2-own-scope-list-filter) gegen die acht Fehlerklassen
  geprueft. Handler gehen ueber `client.ListTickets`/`ListReports`/`ListPendingApprovals`
  (kein Layer-Bypass); `.proto` und `.pb.go` beider Dienste im selben Commit; kein neuer
  Guard, keine neue Route, keine neue Tabelle; der `whereExtra`-Umbau in
  `helpdesk/postgres_repository.go` numeriert die Platzhalter jetzt korrekt ueber
  `len(args)+1` statt hart `$2`. Kein Fund.
- offen fuer Luke:
  (1) KEIN FEINER CAPABILITY-KEY. `finance:expense:*` fehlt in `capability-catalog.ts`. Der
      Scope-Filter der Liste ist trotzdem verdrahtet (`ownerFilterForScope(w, r,
      "finance:expense", "read")`, Repo-Filter `submitted_by`, Index dafuer angelegt) und
      loest heute immer auf `all` auf. Sobald der Key katalogisiert und geseedet ist, greift
      die Einschraenkung ohne weitere Code-Aenderung. Bewusst so herum: ein Scope, der still
      NICHT wirkt, ist die Richtung, die Zeilen leakt.
  (2) GENEHMIGUNG HAENGT AN `finance:write`. Ein eigener `finance:expense:approve`-Key
      gehoert in den Katalog — heute darf jeder, der Ausgaben anlegen darf, auch fremde
      genehmigen. Die Vier-Augen-Regel greift trotzdem (niemand genehmigt sich selbst), aber
      das ist eine schwaechere Aussage als "nur Vorgesetzte genehmigen".
  (3) KEIN GET-BY-ID UND KEIN DETAIL-ENDPOINT. Das FE hat keinen (`financeExpenseApi` kennt
      nur list/create/update/approve/reject/receipt/delete), also wurde keiner gebaut. Der
      Service kann es (`Service.Get`), die Route fehlt bewusst.
  (4) BELEG IST METADATEN. Wer eine echte Beleg-Ablage will (GoBD §147), braucht Upload +
      Dokument-ID; Trigger und Ablageort stehen als `lean:`-Marker im Paketkommentar.
  (5) `finance_expenses.submitted_by` ist `NULL`-bar mit `ON DELETE SET NULL` — ein
      geloeschter Mitarbeiter reisst seine Ausgaben nicht mit. Folge: nach dem Loeschen des
      Einreichers greift die Vier-Augen-Pruefung fuer diese Zeile nicht mehr (sie kann
      niemanden mehr vergleichen). Bei `pending`-Zeilen eines geloeschten Users ist das
      vertretbar, wenn es auffaellt — sonst muesste die Zeile beim User-Delete auf
      `rejected` laufen.

## Iteration 25 — fe-finance-bank-accounts — done — 2026-08-01 21:50 (Nachtlauf 3)
- gebaut: Bankkonten-Stammdaten von der Tabelle bis zur Route. Migration 000258
  `finance_bank_accounts` (tenant_id NOT NULL + FK auf tenants, RLS mit USING **und** WITH CHECK,
  `UNIQUE (tenant_id, iban)`, CHECK dass die IBAN kanonisch ist, CHECK auf ISO-4217-Form der
  Waehrung) plus ein Index auf `finance_bank_statements (tenant_id, account_iban, statement_date
  DESC, created_at DESC)`; `models.BankAccount`; Repo- und Service-Methoden **im bestehenden
  Paket `internal/biz/banking`** (nicht daneben); fuenf RPCs am FinanceService
  (`ListBankAccounts`/`CreateBankAccount`/`UpdateBankAccount`/`DeleteBankAccount`/
  `ConnectBankAccount`) mit Regen im selben Commit; `internal/server/biz_grpc_bankaccount.go`;
  fuenf Gateway-Routen in `route_biz_bank_accounts.go` plus vier openapi.yaml-Pfade und drei
  Schemas.
- lean-fund vorab: das Backlog liest sich wie ein Neubau, aber `internal/biz/banking` existiert
  bereits (Migration 000247: Auszugs-Import CAMT.053/MT940 + Matcher). Es fehlte nur der
  Kontenstamm — `finance_bank_statements.account_iban` ist Freitext aus der hochgeladenen Datei.
  Deshalb kein neues Paket: die Saldo-Ableitung liest die Statements-Tabelle, die diesem Paket
  gehoert; ein Fremdpaket haette quer hineingegriffen. Der IBAN-Validator kam aus
  `internal/dachfmt` (S4.1), nicht neu geschrieben — nur `NormalizeIBAN`/`FormatIBAN` daneben
  ergaenzt (je ~8 Zeilen, gehoeren fachlich genau dorthin).
- fachliche entscheidungen (alle begruendet, keine aus dem Backlog uebernommen):
  - KEINE `balance`-SPALTE. Der FE-Typ verlangt `balance: number`, aber eine gespeicherte Zahl
    waere nur im Moment des Schreibens wahr und wuerde danach still zur Luege altern — und der
    Buchhalter kann eine gealterte Zahl nicht von einer echten unterscheiden. Der Saldo kommt per
    `LEFT JOIN LATERAL` aus dem `closing_balance` des juengsten importierten Auszugs derselben
    IBAN, `last_sync` aus dessen `statement_date`. Ohne Import: `0.00` und `lastSync: null` — das
    ist, was das System ehrlich weiss. Der Test dazu treibt genau das (kein Auszug -> 0, juengerer
    Auszug gewinnt, fremder Tenant mit derselben IBAN bleibt unsichtbar).
  - IBAN KANONISCH GESPEICHERT (Upper-Case, ohne Trennzeichen), gruppiert erst beim Ausliefern
    (`dachfmt.FormatIBAN`). Grund: ein Mensch tippt "DE89 3704 …", eine CAMT.053-Datei traegt
    "DE89370400440532013000". Duerften beide Formen in die Tabelle, existierte dasselbe Konto
    zweimal und der Saldo verteilte sich auf die Kopien. Ein CHECK in der Migration haelt die
    Spalte kanonisch, auch gegen einen kuenftigen Direkt-INSERT. Test:
    `TestCreateAccount_SameIBANDifferentSpellingCollides`.
  - MOD-97 SERVERSEITIG, nicht nur am Rand. Eine IBAN mit einer vertauschten Ziffer ist formal
    speicherbar, findet aber nie einen ihrer Auszuege — und nichts sagte, warum. Deshalb im
    Service (`ErrInvalidIBAN`) und zusaetzlich als `validate:"iban"` am Gateway, damit der Nutzer
    einen Feldfehler statt eines nackten 400 aus dem RPC bekommt.
  - IBAN-KORREKTUR ERLAUBT und sie verschiebt bewusst den Saldo: ein mit Tippfehler angelegtes
    Konto bliebe sonst fuer immer von seinen eigenen Importen getrennt. `UpdateAccount` liest nach
    dem Schreiben neu, damit die Antwort den Saldo der NEUEN IBAN traegt statt den der alten.
  - `connect` IST EIN FLAG, keine Bankverbindung. Der PSD2-Handshake ist im Desktop-Client
    simuliert (`BankConnectDialog.tsx`, "echte Anbindung folgt mit FinAPI (P5)"). Ein zweiter
    Aufruf haelt den urspruenglichen Zeitstempel (der Dialog ist wieder oeffenbar); ein Disconnect
    loescht `connected_at`, sonst liesse sich ein spaeteres Reconnect als "war die ganze Zeit
    verbunden" lesen. `lean:`-Marker mit Upgrade-Trigger steht an `Service.ConnectAccount`.
  - KEIN LOGGING VON KONTODATEN. Weder Service noch gRPC-Handler schreiben IBAN, BIC oder
    Bankname in ein Log, auch nicht auf Debug — nur `account_id` und `tenant_id`. Als Kommentar am
    Paketkopf beider Dateien festgehalten, damit es nicht beim naechsten Anfassen zurueckkommt.
- scope-abweichung, bewusst: der MSW-Mock kennt nur `GET /bank-accounts` und
  `POST /bank-accounts/{id}/connect` — es gibt im FE keinen Anlege-Dialog. Gebaut wurden trotzdem
  POST/PATCH/DELETE (Backlog-Scope), denn ohne sie waere die Liste in Produktion fuer immer leer
  und Konten kaemen nur per SQL in die Datenbank. `connect` ist mitgebaut, weil das FE sonst
  gegen 404 laeuft.
- wire-shape (gegen `types/finance-types.ts:386` geprueft, nicht geraten): Liste
  `{accounts: [...]}` (leer als `[]`, nicht `null`), Einzelentitaet `{account}`, DELETE `{}`.
  Felder camelCase (`bankName`, `lastSync`), `balance` als JSON-**Zahl**. `lastSync` ist ein
  NICHT-`omitempty` Pointer: der FE-Typ ist `string | null` und rendert den
  Nie-synchronisiert-Fall aus dem `null` — ein fehlender Key liefe dort in `undefined`. Zwei
  Tests darauf (`route_biz_bank_accounts_test.go`), inklusive der Gegenprobe, dass
  `tenant_id`/`tenantId` nicht in die Antwort lecken.
- guards: bestehende `RequirePermission("finance","read"/"write"/"delete")`, kein neuer Key und
  damit keine Seed-Pflicht. `capability-catalog.ts` hat keinen `finance:bank-account:*`-Key —
  ein additives `RequirePermissionAny` haette einen Key versprochen, den niemand bekommt.
  `connect` liegt auf `write`, nicht auf `read`: es aendert gespeicherten Zustand.
- gate: build ok (`-p 2`, `./internal/biz/banking/... ./internal/server/... ./internal/gateway/...
  ./internal/dachfmt/... ./cmd/biz/... ./cmd/gateway/...`) | vet ok | lint ok (0 issues) | test ok
  mit `DATABASE_URL` gegen `kmuhub_app`: `./internal/biz/banking/...` (11 Testfunktionen, davon
  3 echte DB-Tests — im `-v`-Lauf verifiziert, **null Skips**), `./internal/dachfmt/...`,
  `./internal/gateway/` und `./internal/server/...` gruen. TestOpenAPIRouteDrift gruen:
  747 registrierte Routen gegen 749 dokumentierte Pfade. gofmt sauber auf allen neuen Dateien
  (die zehn gemeldeten Bestandsdateien in `internal/biz/banking` sind das bekannte
  CRLF-Checkout-Artefakt — mit `cat -A` verifiziert, ganze Datei als `^M`-Diff, kein inhaltlicher
  Fund).
- migration: 000258 up, `down 1` und wieder up lokal sauber durchgelaufen, Kopf zurueck auf 258.
  Prod steht auf 243 — dieser Commit erhoeht den Nachhol-Stapel um eine Migration.
- rls-smoke (als `kmuhub_app`, NOSUPERUSER NOBYPASSRLS, alles in einer zurueckgerollten
  Transaktion): eigener Tenant **1** Konto, fremder Tenant **0**, der abgeleitete Saldo lieferte
  **1234.56** (den eigenen Auszug) und nicht die **999999.99** des Fremd-Tenants mit derselben
  IBAN — das ist der eigentliche Beweis, dass die laterale Statement-Query nicht ueber die
  Tenant-Grenze liest. Ein INSERT mit fremder `tenant_id` wurde von der WITH-CHECK-Klausel
  **abgelehnt**. Zusaetzlich im Go-Test: fremde ID liest als ErrAccountNotFound, fremde ID laesst
  sich nicht loeschen, die Zeile ist danach fuer ihren Besitzer noch da, und dieselbe IBAN unter
  zwei Tenants ist erlaubt (Unique ist per Tenant).
- test-hygiene: `t.Cleanup(pool.Close)` statt `defer pool.Close()`. Beim ersten Lauf meldeten die
  Zeilen-Cleanups "closed pool" — `t.Cleanup` der Fixtures laeuft NACH einem `defer` der
  Testfunktion, die Testdaten waeren also liegen geblieben. Mit `t.Cleanup` greift LIFO und der
  Pool lebt noch, wenn aufgeraeumt wird. (Der Expense-Test loest dasselbe andersherum per
  `defer` an den Fixtures — beides geht, mischen nicht.)
- verify vorgaenger: `8e3ab489` (fe-finance-expenses) gegen die Fehlerklassen geprueft. Alle
  sieben Handler gehen ueber `b.getBizClient()` (kein Layer-Bypass); jeder SELECT/UPDATE/DELETE in
  `expense/postgres_repository.go` traegt `tenant_id = $1`; `.proto` und beide `.pb.go` im selben
  Commit; openapi.yaml im selben Commit; Migration mit up UND down; einziger `Unimplemented` ist
  der `requireExpense()`-nil-Guard, also erwartete Boilerplate. Kein Fund.
- offen fuer Luke:
  (1) KEIN FEINER CAPABILITY-KEY, gleiche Lage wie bei den Ausgaben: `finance:bank-account:*`
      fehlt in `capability-catalog.ts`. Wer `finance:read` hat, sieht alle Konten des Tenants —
      inklusive IBAN und Saldo. Fuer ein KMU mit getrennter Buchhaltung ist das vermutlich zu
      grob; ein eigener Key waere hier wertvoller als bei den Ausgaben.
  (2) DER MOCK BLEIBT AKTIV. `handlers/finance.ts` beantwortet `/finance/bank-accounts` weiterhin
      aus `mockBanking.accounts`. Das Umschalten des FE auf das echte Backend war nicht Teil
      dieser Unit — es faellt auf, sobald jemand die Liste real leer sieht.
  (3) KEIN ANLEGE-DIALOG IM FE. POST/PATCH/DELETE existieren jetzt serverseitig, aber das
      Frontend hat kein Formular dafuer (`BankConnectDialog` verbindet nur ein bestehendes
      Konto). Bis das nachkommt, kommen Konten nur per API-Aufruf in die Datenbank.
  (4) SALDO NUR SO FRISCH WIE DER LETZTE IMPORT. Das ist Absicht (siehe oben), aber es heisst:
      wer nie einen Auszug hochlaedt, sieht dauerhaft `0,00 €` — auch bei einem verbundenen
      Konto. Wenn das im Pilot als Fehler gelesen wird, ist die Antwort ein Hinweis in der UI
      ("noch kein Auszug importiert"), nicht eine manuell gepflegte Saldo-Spalte.
  (5) `connected` OHNE FOLGEN. Ein verbundenes Konto tut heute nichts anderes als ein
      unverbundenes — kein Polling, kein automatischer Abruf. Der Upgrade-Trigger steht als
      `lean:`-Marker an `Service.ConnectAccount`.

## Iteration 26 — fe-finance-bank-transactions-matching — done — 2026-08-01 22:15 (Nachtlauf 3)
- ausgangslage bestaetigt: die Scope-Korrektur der Unit stimmte. Tabelle (000247), Import, Matcher,
  `Reconcile`/`Ignore` und drei Gateway-Routen waren da. **Keine Migration**, kein neuer Service —
  die Unit war Namensgebung, Wire-Shape und eine echte Luecke (siehe unten).
- namensfrage entschieden: `POST .../{id}/reconcile` **umbenannt** zu `.../match`; `.../ignore`
  BLEIBT. Begruendung: `match` und `reconcile` sind dieselbe Operation (bestaetigen + buchen), zwei
  Namen dafuer driften auseinander — die Backlog-Vorgabe "nicht beide Namenspaare" trifft genau
  dieses Paar. `ignore` dagegen ist eine ANDERE Entscheidung als `reject-match`: ablehnen heisst
  "nicht diese Rechnung" (zurueck in die Queue als `unmatched`), ignorieren heisst "gar keine
  Kundenzahlung" (raus aus der Queue als `ignored`). Beide zu einem Endpoint zu verschmelzen haette
  Eintraege aus der Queue genommen, die noch eine Entscheidung brauchen. Kein Alias auf den alten
  Pfad: das FE ruft `reconcile` nirgends (grep ueber `desktop/src` leer), es gibt keinen externen
  Konsumenten, und ein Alias waere genau die Doppelung, die vermieden werden soll.
- echte luecke gefunden: **`reject-match` existierte im Backend gar nicht.** Der Bestand konnte
  einen Vorschlag nur buchen (`Reconcile`) oder aus der Queue nehmen (`Ignore`) — "diesen Vorschlag
  verwerfen, Eintrag bleibt offen" war nicht abbildbar, obwohl der MSW-Mock (`handlers/finance.ts`)
  genau das seit P2.5e tut. Neu: `Service.RejectMatch` + RPC `RejectBankTransactionMatch`.
  Idempotent auf `unmatched` (zweimal geklickt = dasselbe Ergebnis, kein 409), `matched`/`ignored`
  dagegen `ErrNothingToReject` -> FailedPrecondition -> **409**: beides sind Umkehrungen, die eine
  Zahlung bzw. eine bewusste Ablage aufloesen muessten, und das gehoert nicht in diesen Endpoint.
- wire-shape (gegen `types/finance-types.ts:397` und `api/finance-client.ts:581ff` geprueft, nicht
  geraten): flach + camelCase (`matchStatus`, `matchedInvoice`, `counterpart`), `amount` als
  JSON-**Zahl** mit erhaltenem Vorzeichen, `type` als `credit|debit` aus dem Vorzeichen abgeleitet,
  `date` = `value_date`, `description` = `remittance_info` mit Fallback auf `entry_ref` (eine
  Bankgebuehr hat oft keinen Verwendungszweck, eine leere Zeile in der Queue ist nicht
  entscheidbar). Liste `{transactions:[...], total}` (leer als `[]`), Einzelentitaet
  `{transaction}` — auch bei `ignore`, das vorher als nacktes Proto antwortete.
  Zwei Tests darauf (`route_biz_bank_transactions_test.go`), inkl. Gegenprobe, dass
  `statement_id`/`counterparty_iban`/`matched_invoice_id` nicht in die Antwort lecken und
  `matchedInvoice` bei nichts-gematcht **fehlt** (der FE-Typ hat es optional, nicht nullable).
- `ignored` wird aus der Queue gefiltert: der FE-Typ `BankMatchStatus` kennt nur
  `matched|suggested|unmatched`; ein durchgereichtes `ignored` liefe in
  `BankTransactionDetailPanel.tsx:18` in `STATUS_LABEL_KEYS[undefined]` und damit in `t(undefined)`.
  Deshalb `?status=` (ersetzt `?match_status=`): leer oder `all` = Queue ohne `ignored`,
  `status=ignored` holt sie explizit. Umgesetzt ueber `BankTransactionFilter.ExcludeMatchStatus` +
  Proto-Feld `exclude_match_status` — im Gateway nachzufiltern haette `total` und die Seitengroesse
  loechrig gemacht. Unbekannter Wert = 400, nicht stillschweigend "alles".
- `matchedInvoice` ist die Rechnungs**nummer** und kommt aus einem LEFT JOIN auf `finance_invoices`
  im Repo (Backlog-Vorgabe c), nicht aus einem zweiten Roundtrip. Dafuer sind `transactionColumns`
  jetzt auf `t` aliasiert und `transactionFrom` traegt den Join — **wer dort eine WHERE-Bedingung
  ergaenzt, muss praefixen**, sonst ist sie mehrdeutig. Der Join fuehrt `tenant_id` mit, obwohl RLS
  `finance_invoices` schon scoped: unter einem System-Context waere die Nummer sonst
  tenant-uebergreifend lesbar.
- zwei stillere funde, beide gefixt, weil das FE `matchedInvoice` DIREKT anzeigt:
  (1) `Reconcile` setzte nach einem Override `matched_invoice_id` neu, liess die Nummer aber stehen
      — die Antwort haette den Namen der ERSETZTEN Rechnung getragen. Jetzt liest `Reconcile` nach
      dem Update einmal nach (der Join ist die einzige Wahrheit); schlaegt der Read fehl, kostet das
      die Nummer, nicht die Buchung.
  (2) `Ignore` liess die Nummer ebenfalls stehen, obwohl es `matched_invoice_id` auf NULL setzt.
  Ausserdem traegt `MatchResult` jetzt die Nummer mit, damit die Import-Antwort den Vorschlag
  genauso benennt wie ein spaeterer Read der Queue.
- manuelle zuordnung per nummer: das FE sendet `{invoice_number}` (nicht die id). Aufloesung im
  Repo (`FindInvoiceIDByNumber`, tenant-gescopt) und im **Service**, nicht im Handler. Eine Nummer,
  die nichts trifft, ist `ErrInvoiceNotFound` -> **404** und faellt NICHT auf den Vorschlag zurueck
  — das haette Geld gegen eine Rechnung gebucht, die niemand gewaehlt hat. `invoice_id` schlaegt
  `invoice_number`, wenn beides kommt (die id ist eindeutig, die Nummer nur pro Tenant).
- openapi: `reconcile`-Pfad raus, `match` und `reject-match` rein, GET-Parameter auf `status`
  umgestellt, neues Schema `BankTransactionQueueEntry`. Zwei Schemas fuer dieselbe Tabelle ist
  Absicht: die Statement-Routen liefern weiter das volle proto-`BankTransaction` (entry_ref,
  end_to_end_id, counterparty_iban, payment_id …), die Queue nur das, worueber entschieden wird.
  Alle Codes dokumentiert, die der Handler wirklich liefert — inkl. 400 (unbekannter Status) und
  409 (bereits gebucht / Debit / nichts zu verwerfen).
- guards unveraendert: bestehende `RequirePermission("finance","read"/"write")`, kein neuer Key,
  keine Seed-Pflicht. `capability-catalog.ts` hat weiterhin keinen `finance:bank-transaction:*`-Key.
- gate: build ok (`-p 2`) | vet ok | lint ok (0 issues) | test ok mit `DATABASE_URL` gegen
  `kmuhub_app`: `./internal/biz/banking/...` **47 PASS, 0 SKIP, 0 FAIL** (im `-v`-Lauf gezaehlt),
  `./internal/gateway/`, `./internal/server/...`, `./internal/biz/payment/...`,
  `./internal/biz/invoice/...` gruen. TestOpenAPIRouteDrift gruen: 748 registrierte Routen gegen
  750 dokumentierte Pfade. Proto im selben Commit regeneriert (biz.pb.go + biz_grpc.pb.go).
  gofmt: die acht gemeldeten Bestandsdateien sind das bekannte CRLF-Checkout-Artefakt (mit `cat -A`
  verifiziert, ganze Datei als `^M`-Diff); die drei neuen Dateien sind sauber.
- CRLF-FALLE (kostet sonst eine Iteration): ein `python`-Replace mit LF-Suchstring greift auf diesen
  Dateien **still nicht** — es meldet Erfolg und aendert nichts. Erst der naechste Build zeigt es.
  Fuer Bestandsdateien in `internal/biz/banking` das Edit-Tool nehmen.
- rls-smoke (als `kmuhub_app`, alles in einer zurueckgerollten Transaktion, zwei Tenants mit je
  einer Rechnung + gematchter Transaktion): Tenant A sieht **2** Zeilen (seine gematchte + seine
  ignorierte) und als Nummer **nur** `RE-SMOKE-MINE`; die Queue-Bedingung `match_status <> ALL`
  liefert dort **1**; Tenant B sieht **1** Zeile mit `RE-SMOKE-THEIRS`. Der Join leckt also keine
  fremde Rechnungsnummer. Zusaetzlich im Go-Test: fremde Transaktion liest als
  ErrTransactionNotFound, `FindInvoiceIDByNumber` findet die Nummer eines fremden Tenants nicht.
- verify vorgaenger: `689149dd` (fe-finance-bank-accounts) gegen die Fehlerklassen geprueft. Alle
  fuenf Handler ueber `b.getBizClient()`; jeder SELECT/UPDATE/DELETE in
  `postgres_repository_accounts.go` traegt `tenant_id = $1`, der laterale Statement-Read joint ueber
  `s.tenant_id = a.tenant_id`; `.proto` + beide `.pb.go` im selben Commit; openapi.yaml im selben
  Commit; Migration 000258 mit up UND down. Kein Stub, kein Fund.
- offen fuer Luke:
  (1) DER MOCK BLEIBT AKTIV. `handlers/finance.ts` beantwortet `/finance/bank-transactions*`
      weiterhin aus `mockBanking.transactions` — das Umschalten des FE aufs echte Backend war nicht
      Teil dieser Unit. Fuer die Queue heisst das: erst wenn der Mock faellt, sieht jemand, dass die
      Liste ohne importierten Kontoauszug leer ist.
  (2) KEIN FEINER CAPABILITY-KEY, dritte Unit in Folge mit derselben Lage. Wer `finance:read` hat,
      sieht jede Kontobewegung des Tenants inklusive Gegenpartei und Betrag. Bei Gehaltszahlungen
      ("Gehaelter Februar" steht so im Mock) ist das mehr als "Buchhaltung darf Rechnungen sehen" —
      hier waere ein eigener Key wertvoller als bei Ausgaben und Konten.
  (3) `reject-match` SETZT `reconciled_at`/`reconciled_by`. Die Spalten heissen nach dem Buchen,
      tragen jetzt aber auch "wer hat wann abgelehnt". Das ist bewusst (die Information ist es wert)
      und `match_reason = 'rejected'` unterscheidet die Faelle — aber wer die Spalte fuer eine
      Buchungs-Statistik auswertet, muss auf `match_status` filtern, nicht auf `reconciled_at IS
      NOT NULL`.
  (4) EINE ABGELEHNTE ZUORDNUNG KOMMT SOFORT ZURUECK. `RejectMatch` setzt `unmatched`, der naechste
      Import-Lauf matcht dieselbe Rechnung womoeglich erneut vor. Ein "nicht mehr vorschlagen"-
      Merker existiert nicht; ob er gebraucht wird, zeigt der Pilot.

## Iteration 27 — fe-finance-document-chains — done — 2026-08-01 22:40 (Nachtlauf 3)
- commit: f7fd0ce3f0fcbec8a31877b8384e9f5372bb5c24
- gebaut:
  - `GET /api/v1/finance/document-chains` liefert alle Belegketten des Tenants (mandantenweite
    Uebersicht, kein Parameter — FE filtert/sucht client-seitig). Keine neue Tabelle: Ketten
    entstehen als Read-Time-Assembly ueber fuenf tenant-gescopte Queries in
    `internal/biz/invoice/postgres_document_chains.go`, verkettet ueber bestehende FKs
    (`finance_invoices.source_quote_id`, `finance_payments`/`finance_dunning_records.invoice_id`,
    `finance_credit_notes.original_invoice_id`).
  - Proto: `ChainNode`/`DocumentChain`/`ListDocumentChainsRequest/Response` in `biz.proto` + RPC
    `ListDocumentChains`, im selben Commit regeneriert (`biz.pb.go`, `biz_grpc.pb.go`).
  - `BizGRPCServer.ListDocumentChains` (`internal/server/biz_grpc_document_chains.go`) ruft
    `invoiceService.ListDocumentChains` (duenner Passthrough zum Repo, wie `List`/`ListForDATEVExport`
    im selben Service).
  - Gateway-Handler `route_biz_document_chains.go`: Hand-Mapping wie bei Bankkonten/-transaktionen
    (repo-weiter protojson-Marshaller ist `UseProtoNames: true`, FE braucht aber `totalValue`/
    `isComplete` camelCase), plus DACH-Betragsformatierung ("CHF 12.450,00") — die Komponente
    rendert `amount`/`totalValue` unveraendert, kein `parseFloat` im Client (verifiziert).
  - `models.DocumentChain`/`ChainNode` in `internal/models/finance.go`, Status-/Typ-Konstanten
    passend zum FE-Union-Typ.
  - Test: `internal/biz/tenant_isolation_document_chains_test.go` — frische Tenants, `kmuhub_app`.
- verify vorgaenger: `d18c622e` (fe-finance-bank-transactions-matching) gegen alle acht
  Fehlerklassen geprueft. Alle Handler ueber `b.getBizClient()`; Proto+beide `.pb.go` im selben
  Commit; Guards unveraendert (`RequirePermission("finance", "write"/"read")`, keine Ersetzung,
  kein neuer Key -> keine Seed-Pflicht); jeder Query/Join im Repo traegt `tenant_id`; openapi.yaml
  im selben Commit, `TestOpenAPIRouteDrift` lokal nachgefahren: gruen (748/750 vor dem Commit).
  Kein Fund.
- SCOPE-KORREKTUR (Fund beim Bau): Backlog-Scope-Text ("Kette zu einem Beleg") stimmte nicht mit dem
  bereits verdrahteten FE-Vertrag ueberein — `useDocumentChains()` ruft die Route ganz ohne Parameter,
  `BelegketteTab.tsx` ist eine mandantenweite Liste mit Client-seitiger Suche. Gebaut wie der reale
  Vertrag es verlangt (siehe Backlog-Notiz "ERGEBNIS" fuer Details: Sortierung, Platzhalter-Logik,
  `isComplete`-Regel, abgeleitete Nummern fuer Zahlung/Mahnung ohne eigene Sequenz).
- gate: build ok (`-p 2`, `internal/biz/...`, `internal/gateway/...`, `internal/server/...`,
  `cmd/gateway/...`, `cmd/biz/...`) | vet ok | lint ok (`golangci-lint`, 0 issues) | test ok mit
  `DATABASE_URL` gegen `kmuhub_app`: `internal/biz/...` + `internal/biz/invoice/...` **603 PASS,
  0 SKIP** (per `-v`-Lauf gezaehlt); `internal/gateway/...`, `internal/server/...` gruen.
  `TestOpenAPIRouteDrift` gruen: 749 registrierte Routen gegen 751 dokumentierte Pfade.
  rls-smoke: `TestTenantIsolation_DocumentChains` — Eigentuemer sieht 2 Ketten (Rechnungs-Kette mit
  6 Knoten in chronologischer Reihenfolge, Restbetrag 500 von 1000 EUR nach 400 Zahlung + 100
  Gutschrift; Solo-Angebot-Kette, abgelehnt, `isComplete=true`), fremder Tenant sieht exakt 1 eigene
  Kette ohne jede Spur der Kunden-/Betragsdaten des anderen Tenants.
- offen:
  - `finance-chains.ts`-Mock bleibt aktiv im FE — Umschalten auf das echte Backend war nicht Teil
    dieser Unit (gleiche Lage wie bei den anderen `fe-finance-*`-Units in Block D).
  - Kein eigener Capability-Key: die Route haengt am bestehenden `RequirePermission("finance","read")`
    wie ihre Nachbarn. Belegketten zeigen Kundennamen und Betraege quer durch alle Belegarten — falls
    Block-B-artige Verfeinerung noch kommt, waere das ein Kandidat.
  - Mahnungs- und Zahlungs-"Nummer" sind abgeleitet (`{Rechnungsnummer}/M{level}` bzw. Fallback auf
    die Rechnungsnummer), da beide Belegarten keine eigene Nummernsequenz haben — falls das je stoert,
    ist das der Punkt, an dem eine echte Sequenz eingefuehrt werden muesste.

## Iteration 28 — fe-finance-time-entries — done — 2026-08-01 23:05 (Nachtlauf 3)
- commit: c414b61d
- gebaut:
  - `GET /api/v1/finance/time-entries` liefert abrechenbare Zeiteintraege des Tenants (Filter
    `?billed=false`/`true`, matched das FE `financeTimeEntryApi.listUnbilled()`). KORREKTUR zur
    Backlog-Notiz: nicht die HR-Aggregation (`CreateInvoiceFromTimeEntries`/`hr_work_time_entries`),
    sondern die WORK-Tabelle `time_entries` (Migration 000030, `task_id -> tasks -> projects`) ist
    die Datenquelle — der FE-Vertrag (`FinanceTimeEntry`: project/task/employee/description) passt
    nur zu dieser, nicht zur ArbZG-Clock-in/out-Tabelle.
  - Neuer RPC `WorkService.ListBillableTimeEntries` (`work.proto`, tenant aus RLS-Context wie alle
    Nachbar-RPCs in dieser Datei, keine explizite `tenant_id` im Request), im selben Commit
    regeneriert (`work.pb.go`, `work_grpc.pb.go`).
  - `internal/work/timeentry`: `Repository.ListBillable` (neue Interface-Methode) + Postgres-Impl
    (JOIN `time_entries`/`tasks`/`projects`/`users`, alle drei Tabellen tenant-gescoped, nur
    `ended_at IS NOT NULL AND duration_seconds IS NOT NULL` zaehlt als abrechenbar) +
    `Service.ListBillable`-Passthrough. `models.BillableTimeEntry` neu.
  - `WorkGRPCServer.ListBillableTimeEntries` (`internal/server/work_grpc.go`) mapped auf Proto,
    `hours` als `duration_seconds/3600` gerundet auf 2 Nachkommastellen.
  - Gateway: `getWorkClient()` neu auf `BizRoutes` (`route_biz_time_entries.go`) — die Route haengt
    unter `/finance/` (FE-Pfad), die Daten gehoeren aber Work; Precedent fuer mehrere Routes-Structs
    mit demselben Client-Typ ist `BizExtRoutes`, das ebenfalls eine eigene `getBizClient()`-Kopie
    neben `BizRoutes` haelt. Route haengt am bestehenden `RequirePermission("finance","read")`, kein
    neuer Key, keine Seed-Pflicht.
  - `billed` ist im Gateway-Handler hartkodiert `false` (`lean:`-Marker) — die Invoice-Erstellung aus
    `HoursToInvoiceDialog` laeuft ueber die generische Create-Invoice-Route mit frei gebauten
    Line-Items und persistiert keine Zeiteintrag-IDs, anders als der HR-Pfad ueber
    `finance_invoices.time_tracking_source`. `?billed=true` liefert deshalb konsequent `[]`.
  - Test: `TestListBillable_TenantIsolation` (`internal/work/timeentry/tenant_isolation_phase2_test.go`)
    — frische Tenants (nicht die geteilten `TenantA`/`TenantB`), beweist: fremder Tenant sieht 0
    Eintraege, Eigentuemer sieht genau 1 (die abgeschlossene, mit korrekt gejointen Projekt-/Task-/
    Mitarbeiternamen), ein laufender Timer ohne `ended_at`/`duration_seconds` erscheint nicht.
- verify vorgaenger: `f7fd0ce3` (fe-finance-document-chains) gegen alle acht Fehlerklassen geprueft.
  Handler ueber `b.getBizClient()`; Proto+beide `.pb.go` im selben Commit; Guard unveraendert
  (`RequirePermission("finance","read")`, kein neuer Key); alle fuenf Repo-Queries tenant-gescoped
  (`tenant_id = $1` bzw. Join-Bedingung); openapi.yaml im selben Commit. Unabhaengig nachgefahren:
  `go build`/`go test ./internal/biz/...` (alle Pakete inkl. `invoice` gruen, `TestTenantIsolation_
  DocumentChains` PASS ohne Skip), `go test ./internal/gateway/... ./internal/server/...` gruen.
  Kein Fund.
- gate: build ok (`-p 2`, `internal/work/...`, `internal/models/...`, `internal/server/...`,
  `internal/gateway/...`, `cmd/work/...`, `cmd/gateway/...`) | vet ok | lint ok (`golangci-lint`,
  0 issues) | test ok mit `DATABASE_URL` gegen `kmuhub_app`: `internal/work/...` komplett gruen
  (alle Unterpakete inkl. `timeentry`), `internal/server/...` + `internal/gateway/...` gruen.
  `TestOpenAPIRouteDrift` gruen: 750 registrierte Routen gegen 752 dokumentierte Pfade (+1
  gegenueber Iteration 27, wie erwartet).
  rls-smoke: `TestListBillable_TenantIsolation` — Details oben unter "gebaut".
- offen:
  - `finance-time-entries.ts`-Mock (`mockFinanceTimeEntries`) bleibt aktiv im FE — Umschalten auf
    das echte Backend war nicht Teil dieser Unit (gleiche Lage wie bei den anderen `fe-finance-*`-
    Units in Block D).
  - `billed` ist dauerhaft `false`, bis die Invoice-Erstellung Zeiteintrag-IDs persistiert (siehe
    `lean:`-Marker in `route_biz_time_entries.go`). Ohne diese Verknuepfung erscheint ein bereits
    abgerechneter Eintrag beim naechsten Aufruf wieder in der Liste — funktional identisch zum
    heutigen FE-Mock-Verhalten (der Mock hat dieselbe Luecke: `billed` steht dort statisch fest),
    aber ein Kandidat fuer eine Folge-Unit, sobald die reale Verknuepfung gebraucht wird.
  - Kein eigener Capability-Key: die Route haengt am bestehenden `RequirePermission("finance","read")`.
    Zeiteintraege zeigen Mitarbeiternamen und Projektzuordnung quer durchs Team — falls Block-B-
    artige Verfeinerung noch kommt, waere das ein Kandidat (aehnlich der Notiz bei
    `fe-finance-bank-transactions-matching`).

## Iteration 29 — fe-finance-transactions — done — 2026-08-01 23:15 (Nachtlauf 3)
- commit: 120b32a4
- gebaut:
  - `GET /api/v1/finance/transactions` liefert die konsolidierte Zahlungsbewegungs-Liste des
    Tenants (`{transactions:[...], total}`), `DELETE /api/v1/finance/transactions/{id}` entfernt
    einen Eintrag. SCOPE-KORREKTUR zur Backlog-Notiz: nicht Banktransaktionen, sondern
    `finance_payments` + `finance_expenses` — der reale FE-Vertrag ist `financeTransactionApi`
    (`finance-client.ts`) + `useTransactions`/`useDeleteTransaction` (`useFinanceLedger.ts`),
    konsumiert von `TransactionsTab.tsx`/`BerichteTab.tsx`/`TransactionDetailPanel.tsx`. Die
    Legacy-`useFinanceStore` (`stores/finance.ts`, "Mock-Store fuer BuchhaltungPage — nicht im
    Router") liefert nur die geteilte `Transaction`-Typdefinition, keine Daten.
  - Kein neuer Tisch: `internal/biz/invoice/postgres_transactions.go` (neue Datei, Muster wie
    `postgres_document_chains.go`) liest `finance_payments` JOIN `finance_invoices` (Type
    `income`, Description "Rechnung {number} {customer}", Category hartkodiert "Umsatzerlöse"
    mangels besserer Spalte, Reference = Rechnungsnummer) und `finance_expenses WHERE
    status='approved'` (Type `expense`) und merged sortiert nach Datum absteigend. Pending/
    rejected Ausgaben bewegen nie Geld und fehlen bewusst aus der Liste.
  - Neue RPCs `ListFinanceTransactions`/`DeleteFinanceTransaction` (`biz.proto`, im selben Commit
    regeneriert: `biz.pb.go`/`biz_grpc.pb.go`), Server-Seite in
    `internal/server/biz_grpc_transactions.go`, delegiert an `invoiceService.ListTransactions`
    (neue Repository-/Service-Methode) fuer den Read.
  - IDs sind praefixiert (`pay-<uuid>`/`exp-<uuid>`), damit Delete weiss, welche Tabelle gemeint
    ist — Loeschen bedeutet je Seite etwas anderes: eine Zahlung loeschen revertiert ggf. den
    Rechnungsstatus (bestehendes `payment.Service.Delete`, unveraendert wiederverwendet), eine
    bereits entschiedene Ausgabe zu loeschen ist von `expense.Service.Delete` verboten
    (`ErrDecided` -> 409) — bei dieser Ansicht IMMER, weil jede Expense-Zeile hier per Konstruktion
    approved ist. Echtes, bestehendes Geschaeftsverhalten, kein Stub.
  - Root-Cause-Fix im selben Commit (kein separates Scope-Kriechen — `DeleteFinanceTransaction`
    ruft diesen Pfad direkt auf): `payment.Service`/`Repository.GetByID` gab bei "nicht gefunden"
    einen rohen `errors.New("payment not found")` statt eines Sentinels zurueck, `mapBizError` traf
    keinen Fall und liess die bestehende `DeletePayment`-Route bei unbekannter ID auf 500 statt 404
    laufen. `payment.ErrNotFound`-Sentinel ergaenzt, Postgres-Repo + Mock-Repo (`service_test.go`)
    darauf umgestellt, `mapBizError`-Fall ergaenzt — behebt denselben Bug auch fuer die bestehende
    Payment-Route.
  - Guard: `RequirePermission("finance","read"/"delete")` wie die Nachbarrouten (kein Katalog-Key
    fuer transactions, gleiche Begruendung wie bei expenses/bank-accounts in `p2c-work-documents-
    crm-finance-2`).
  - Tests: `TestTenantIsolation_Transactions` (`internal/biz/`, echte DB) — Eigentuemer sieht genau
    2 Eintraege (1 Payment + 1 approved Expense; pending/rejected Ausgaben fehlen), sortiert
    neueste zuerst, Description/Category/Reference/InvoiceID wie oben. Fremder Tenant sieht exakt
    seine eigenen 2, keine Spur der Tenant-A-Daten. `TestToFinanceTransactionWire_Income`/
    `_ExpenseOmitsAbsentOptionals` (`internal/gateway/`) pinnen den Wire-Shape (camelCase
    `invoiceId`, `amount` als JSON-Zahl, optionale Felder bei Expense-Zeilen abwesend).
- verify vorgaenger: `c414b61d` (fe-finance-time-entries) gegen alle acht Fehlerklassen geprueft.
  Handler ueber `b.getWorkClient()`; Proto+beide `.pb.go` im selben Commit; Guard unveraendert
  (`RequirePermission("finance","read")`, kein neuer Key); Repo-Query tenant-gescoped ueber alle
  drei gejointen Tabellen; openapi.yaml im selben Commit. Unabhaengig nachgefahren: `go build`,
  `go vet`, `golangci-lint` (0 issues), `go test ./internal/work/... ./internal/server/...
  ./internal/gateway/...` mit `DATABASE_URL` gegen `kmuhub_app` — alles gruen, 0 SKIP (per `-v`
  gezaehlt), `TestListBillable_TenantIsolation` PASS. `TestOpenAPIRouteDrift` gruen. Kein Fund.
- gate: build ok (`-p 2`, `internal/biz/...`, `internal/gateway/...`, `internal/server/...`,
  `internal/models/...`, `proto/biz/v1/...`, `cmd/biz/...`, `cmd/gateway/...`) | vet ok | lint ok
  (`golangci-lint`, 0 issues) | test ok mit `DATABASE_URL` gegen `kmuhub_app`:
  `internal/biz/...` (alle Unterpakete inkl. `invoice`, `payment`), `internal/gateway/...`,
  `internal/server/...` gruen, 0 SKIP (per `-v` gezaehlt auf den neuen/beruehrten Paketen).
  `TestOpenAPIRouteDrift` gruen: 752 registrierte Routen gegen 754 dokumentierte Pfade (+2
  gegenueber Iteration 28, wie erwartet: GET + DELETE).
  rls-smoke: `TestTenantIsolation_Transactions` — Details oben unter "gebaut".
- offen:
  - `finance-ledger.ts`-Mock (`mockTransactions`) bleibt aktiv im FE — Umschalten auf das echte
    Backend war nicht Teil dieser Unit (gleiche Lage wie bei den anderen `fe-finance-*`-Units in
    Block D).
  - Kein eigener Capability-Key: die Route haengt am bestehenden `RequirePermission("finance",
    "read"/"delete")`. Der reale FE-Gate-Call fuer den Loeschen-Button ist `finance:incoming:book`
    (verifiziert in `TransactionsTab.tsx`) — analog zur Notiz bei `p2c-work-documents-crm-finance-2`
    gaten `finance:incoming:*` reale Buttons ohne eigenen Backend-Anschluss; ein Kandidat fuer eine
    kuenftige Block-B-artige Verfeinerung.
  - Das Loeschen eines Expense-Eintrags aus dieser Ansicht schlaegt IMMER mit 409 fehl (jede
    Expense-Zeile hier ist per Konstruktion `approved`, und `expense.Service.Delete` verbietet das
    Loeschen entschiedener Ausgaben). Der FE-Loeschen-Button unterscheidet nicht zwischen Payment-
    und Expense-Zeilen — das ist eine FE-Produktentscheidung ausserhalb des Backend-Scopes dieser
    Unit, aber ein sichtbares Verhalten, das Luke kennen sollte.

## Iteration 30 — fe-leads-lifecycle — done — 2026-08-01 23:35
- commit: 6b570fcc
- gebaut:
  - **Leads sind Kontakte, keine zweite Tabelle.** Migration `000259_contact_lead_lifecycle`
    haengt sechs Spalten an `contacts`: `lifecycle_stage VARCHAR(20) NOT NULL DEFAULT 'customer'`
    (CHECK lead|qualified|customer — Bestandszeilen bekommen per Default den Wert, der dem
    heutigen Verhalten entspricht, kein NULL-Loch), `lead_source`, `lead_score SMALLINT` (CHECK
    0–100), `lead_temperature` (NUR der manuelle Override, NULL = aus dem Score ableiten),
    `lead_status`, `lead_company`. Dazu ein partieller Index
    `(tenant_id, lifecycle_stage, created_at DESC) WHERE lifecycle_stage <> 'customer'`.
    `contacts` traegt `tenant_id NOT NULL` + Policy schon seit Migr. 000070/RLS-Welle — neue
    Spalten erben beides, keine neue Policy noetig. Down-Migration getestet (259/d dann 259/u).
  - `lead_company` ist keine Bequemlichkeit: der FE-`Lead`-Typ fuehrt `company` als Pflicht-String,
    und CSV-/Dialer-Intake nennt einen Arbeitgeber lange bevor jemand eine `companies`-Zeile
    anlegt. Ohne die Spalte waere der Wert beim Anlegen still verloren gegangen. Read loest ueber
    `COALESCE(companies.name, contacts.lead_company)` auf.
  - Vier Routen — mehr als der Scope woertlich nannte (`GET` + Convert), aber ohne Schreibpfad
    waere `GET /leads` per Konstruktion immer leer und `lead_status` eine Spalte, die man nie
    setzen kann: `GET /api/v1/leads` (`{items,total}`, Filter `stage`/`status`/`search`,
    Pagination), `POST /api/v1/leads`, `PATCH /api/v1/leads/{id}` (status/temperature; leerer
    String loescht den Override), `POST /api/v1/leads/{id}/convert`.
  - Service-Layer in `internal/crm/contact/lead.go` (`CreateLead`/`ListLeads`/`UpdateLead`/
    `ConvertLead` + `ComputeLeadScore`/`ScoreToTemperature`/`EffectiveTemperature`), Persistenz in
    `postgres_lead.go` (2 neue Repository-Methoden `ListLeads`/`UpdateLead`, beide tenant-gescoped;
    Mock im `service_test.go` nachgezogen). `CreateInput` bekam ein optionales `Lead *LeadIntake`
    statt eines zweiten Create-Pfads — bestehende Aufrufer bleiben unveraendert.
  - Scoring serverseitig, regelbasiert, kein ML: spiegelt `DEFAULT_LEAD_SCORING` aus
    `stores/leadScoring.ts` exakt (dialer 35 / manual 25 / csv 10, unbekannte Quelle 15; +email 20
    +phone 15 +company 20 +notes 10; Schwellen hot 66 / warm 33). Der Score wird nie vom Client
    uebernommen — ein Client-Score waere eine Client-gewaehlte Prioritaet. `lean:`-Marker mit
    Upgrade-Trigger auf `tenant_settings (module_id='crm', key='leadScoring.*')`.
  - Neue RPCs `ListLeads`/`CreateLead`/`UpdateLead`/`ConvertLead` in `crm.proto`, im selben Commit
    regeneriert (`crm.pb.go`/`crm_grpc.pb.go`), Server-Seite in `internal/server/crm_grpc_leads.go`
    ueber `contactService`. Gateway-Handler gehen ausschliesslich ueber `c.getCRMClient()`.
  - Guards: bestehende `contactRead`/`contactCreate`/`contactEdit` (`RequirePermissionAny`,
    `contacts`+`crm:contact`). **Kein neuer Permission-Key, also kein Seed noetig** — und kein
    bestehender Guard ersetzt.
  - Sicherheitsdetail: der UPDATE-Pfad hat `AND lifecycle_stage <> 'customer'` in der WHERE-Klausel.
    Der Lead-Endpoint darf keine Seitentuer zum Editieren gewoehnlicher Kontakte sein; ein Test
    deckt genau das ab (Update auf eine Customer-Zeile -> `ErrLeadNotFound`).
  - Tests: `TestLeads_TenantIsolation` (echte DB, `kmuhub_app`) — Eigentuemer sieht genau seine 2
    Leads, die gewoehnliche Kontaktzeile taucht NICHT im Inbox auf; derselbe `ListLeads`-Aufruf aus
    der Fremd-Session mit der *echten* fremden `tenantID` liefert 0 Zeilen (nur RLS kann das
    stoppen), waehrend der fremde Tenant seinen eigenen Lead sehr wohl sieht; Status-Filter,
    Override-Setzen und -Loeschen. `TestCreateLead_PersistsComputedScore` — Score 100 persistiert,
    Lead ist ueber `GetByID` auch als Kontakt lesbar, Convert aendert die Stage **derselben** Zeile
    (`COUNT(*) = 1` nach Convert: eine Person, eine Zeile). Dazu Scoring-Tabellentests,
    Validierungstests und zwei Gateway-Wire-Shape-Tests (`route_crm_leads_test.go`), die camelCase
    pinnen und dass `temperatureOverride` ohne Pin abwesend bleibt.
- verify vorgaenger: `120b32a4` (fe-finance-transactions) gegen alle acht Fehlerklassen geprueft.
  Handler ueber `b.getBizClient()`; `biz.proto` + beide `.pb.go` im selben Commit; keine
  `Unimplemented`/`TODO` im neuen Pfad (`postgres_transactions.go`, `biz_grpc_transactions.go`);
  Guards `RequirePermission("finance","read"/"delete")` sind bestehende Keys der Nachbarrouten
  (Zeilen 98/155/225/245 in `route_biz.go`) — additiv ergaenzt, keiner ersetzt, also kein Seed
  faellig; keine neue Tabelle (Read-Zeit-Union); Wire-Shape `{transactions, total}` deckt sich
  exakt mit `financeTransactionApi.list()` in `desktop/src/renderer/src/api/finance-client.ts:494`;
  openapi.yaml im selben Commit (+123 Zeilen), `TestOpenAPIRouteDrift` gruen. Kein Fund.
- gate: build ok (`-p 2`: `internal/crm/...`, `internal/gateway/...`, `internal/server/...`,
  `internal/models/...`, `proto/crm/...`, `cmd/crm/...`, `cmd/gateway/...`) | vet ok | lint ok
  (`golangci-lint`, 0 issues) | test ok mit `DATABASE_URL` gegen `kmuhub_app`
  (verifiziert NOSUPERUSER/NOBYPASSRLS via `pg_roles`): `internal/crm/...` (alle 12 Unterpakete),
  `internal/gateway/`, `internal/server/...` gruen, **0 SKIP** (per `-v` gezaehlt auf
  `internal/crm/contact` + `internal/gateway`). migration ok (259 up, down 1, up — sauberer
  Roundtrip). rls-smoke ok: `TestLeads_TenantIsolation`, Details oben.
  `TestOpenAPIRouteDrift` gruen (4 neue Routen, 4 neue Pfad-Eintraege).
- offen:
  - **`useLeads.ts` bleibt Mock-first.** Der FE-Hook arbeitet weiter auf seinem In-Memory-Array;
    das Umstellen auf `/api/v1/leads` war nicht Teil dieser Unit (gleiche Lage wie bei den
    `fe-finance-*`-Units). Beim Umbau beachten: `useLeads()` liefert heute ein *nacktes* Array,
    der Endpoint liefert `{items,total}` (Backlog-Vorgabe).
  - **Produktfrage fuer Luke: Leads erscheinen jetzt auch in `GET /api/v1/contacts`.** Das folgt
    zwingend aus der Modell-Entscheidung (ein Lead *ist* ein Kontakt) und ich habe die bestehende
    Route bewusst NICHT angefasst — eine stille Verhaltensaenderung an einer Bestandsroute waere
    schlimmer als die offene Frage. Fuer Bestandsdaten aendert sich faktisch nichts (alle sind
    `customer`). Sobald echte Leads angelegt werden, ist zu entscheiden, ob die Kontaktliste
    `lifecycle_stage = 'lead'` per Default ausblendet.
  - **Batch-Anlage fehlt** (`useCreateLeadsBatch`, CSV-Import von Leads). `ImportContactsCSV`
    existiert, kennt aber die Lead-Spalten nicht. Eigene Unit wert.
  - `lead_temperature` speichert ausschliesslich den manuellen Override; die effektive Temperatur
    wird serverseitig in `toLeadInfo` abgeleitet, damit sie nicht an zwei Orten driftet.

## Iteration 31 — fe-projects-time-entries — done — 2026-08-01 23:45 (Nachtlauf 3)
- commit: db17bb62
- gebaut:
  - `GET /api/v1/projects/{id}/time-entries` liefert die abgeschlossenen Zeiteintraege eines
    Projekts (`{entries:[...]}`, Felder `id/date/task/person/hours/description`). KORREKTUR zur
    Backlog-Sources-Notiz: `desktop/src/renderer/src/api/clients` existiert nicht — der reale
    FE-Vertrag ist `useProjectTimeEntries` (`api/hooks/useProjects.ts:507`), Typ `ProjectTimeEntry`.
    Keine Migration noetig: `tasks.project_id` existiert bereits seit Migration 000025 — die andere
    Kandidaten-Spalte `hr_work_time_entries.project_id` (000212) gehoert zur ArbZG-Clock-in/out-
    Tabelle und ist der falsche Join fuer diesen FE-Vertrag (gleiche Verwechslungsgefahr wie bei
    `fe-finance-time-entries`, Iteration 28).
  - Neuer RPC `WorkService.ListProjectTimeEntries` (`work.proto`, Request nur `project_id`), im
    selben Commit regeneriert (`work.pb.go`, `work_grpc.pb.go`). `billed` bleibt bewusst ausserhalb
    des RPC-Vertrags und wird im Gateway gefiltert — Precedent `ListBillableTimeEntries`.
  - `internal/work/timeentry`: neue Interface-Methode `Repository.ListByProject` + Postgres-Impl
    (JOIN `time_entries`/`tasks`/`projects`/`users`, alle drei Tabellen tenant-gescoped ueber
    `p.tenant_id = $2 AND te.tenant_id = $2`, nur `ended_at IS NOT NULL AND duration_seconds IS
    NOT NULL` zaehlt) + `Service.ListByProject`-Passthrough. `models.ProjectTimeEntry` neu.
  - `WorkGRPCServer.ListProjectTimeEntries` (`internal/server/work_grpc.go`) prueft
    Projekt-Zugehoerigkeit SERVERSEITIG, nicht nur ueber RLS: ruft zuerst `projectService.Get(ctx,
    projectID, tenantID, uuid.Nil, true)` (dasselbe Muster wie die bestehende `GetProject`-RPC) —
    ein fremder Tenant oder eine falsche ID landet als `NotFound`, bevor ueberhaupt eine Zeitzeile
    abgefragt wird. Die Repository-Query filtert zusaetzlich selbst ueber `p.tenant_id` (defense
    in depth, kein Verlass allein auf RLS).
  - Gateway: `WorkRoutes.HandleListProjectTimeEntries` (`route_work_time.go`) haengt am bestehenden
    `projRead`-Guard (`RequirePermissionAny(projects:read, work:project:read)`) unter
    `/api/v1/projects/{id}/time-entries` in `route_work.go` — kein neuer Permission-Key, keine
    Seed-Pflicht. `billed=true` liefert konsequent `[]entries` (`lean:`-Marker mit demselben
    Upgrade-Trigger wie `route_biz_time_entries.go`: Invoice-Erstellung persistiert bisher keine
    Zeiteintrag-IDs, also kann kein Eintrag als tatsaechlich abgerechnet erkannt werden).
  - Tests: `TestListByProject_TenantIsolation` (echte DB, `kmuhub_app`) — zwei Projekte DESSELBEN
    Tenants duerfen sich nicht vermischen (project_id grenzt ein, nicht nur tenant_id: Projekt A
    sieht nur seinen abgeschlossenen Eintrag, nicht den von Projekt B), ein laufender Timer ohne
    `ended_at` erscheint nicht, ein fremder Tenant mit der geratenen echten Projekt-ID von A sieht
    0 Zeilen. Gateway-Wire-Shape-Test `TestToProjectTimeEntryWire_Keys` pinnt die sechs erwarteten
    Felder und beweist, dass weder `project` noch `billed` noch `employee` (das Finance-Pendant-Feld
    aus `FinanceTimeEntry`) im Body landen.
- verify vorgaenger: `344f17de`/`6b570fcc` (fe-leads-lifecycle) gegen alle acht Fehlerklassen
  geprueft. Alle vier Lead-Routen ueber `c.getCRMClient()`; `crm.proto` + beide `.pb.go` im selben
  Commit regeneriert; Repository tenant-gescoped (`ct.tenant_id = $1` bzw. `$2`, UPDATE/GetByID mit
  `AND ct.tenant_id = $2`); keine neue Tabelle (Spalten an `contacts`, bereits RLS-geschuetzt); Guards
  bestehende `contactRead`/`contactCreate`/`contactEdit`, kein neuer Key, keine Seed-Pflicht.
  Unabhaengig nachgefahren: `go build -p 2` auf `internal/crm/...`, `internal/gateway/...`,
  `internal/server/...`, `internal/models/...`, `proto/crm/...`, `cmd/crm/...`, `cmd/gateway/...`
  gruen. Kein Fund.
- gate: build ok (`-p 2`: `internal/gateway/...`, `internal/work/...`, `internal/server/...`,
  `internal/models/...`, `proto/work/...`, `cmd/gateway/...`, `cmd/work/...`) | vet ok | lint ok
  (`golangci-lint`, 0 issues) | test ok mit `DATABASE_URL` gegen `kmuhub_app` (verifiziert
  NOSUPERUSER/NOBYPASSRLS via `pg_roles`): `internal/work/...` (alle 17 Unterpakete inkl.
  `timeentry`), `internal/gateway/`, `internal/server/...` gruen, **0 SKIP** (per `-v` gezaehlt auf
  `internal/work/timeentry`: 26 PASS). `TestOpenAPIRouteDrift` gruen: 756 registrierte Routen gegen
  758 dokumentierte Pfade (+1 gegenueber Iteration 30, wie erwartet).
  rls-smoke: `TestListByProject_TenantIsolation`, Details oben.
- offen:
  - `useProjectTimeEntries`/`useProjectTeamUtilization`/`useProjectGuestOverview` bleiben laut
    FE-Kommentar "Mock-served (design preview)" — Umstellung auf das echte Backend war nicht Teil
    dieser Unit (gleiche Lage wie bei den `fe-finance-*`-Units).
  - `billed` ist dauerhaft `false` bis zur selben Invoice-Zeiteintrag-Verknuepfung, die schon bei
    `fe-finance-time-entries` (Iteration 28) offen blieb — eine Folge-Unit koennte beide Endpoints
    in einem Rutsch fixen, sobald die Verknuepfung existiert.
  - Naechste Unit `fe-projects-team-utilization` haengt an dieser (deps: [fe-projects-time-entries])
    und braucht eine SQL-Aggregation ueber Zeiteintraege + Sollarbeitszeit — `hr_work_time_entries`
    (ArbZG) vs. `time_entries` (PM) ist dort erneut zu klaeren, vermutlich ist wieder `time_entries`
    gemeint (Team-Auslastung nach Projekt, nicht nach Anwesenheit).
