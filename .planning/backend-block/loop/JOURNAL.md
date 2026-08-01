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
