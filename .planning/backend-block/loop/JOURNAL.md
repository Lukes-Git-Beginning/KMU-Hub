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

## Iteration 32 — fe-projects-team-utilization — done — 2026-08-02 00:05 (Nachtlauf 3)
- commit: 3f26f98a
- gebaut:
  - `GET /api/v1/projects/{id}/team-utilization` liefert das Auslastungs-Aggregat je Projektmitglied:
    Stunden pro Woche (letzte 6 ISO-Wochen) und pro Monat (letzte 3 Monate), inklusive Mitglieder
    ohne jeden Zeiteintrag (0-gefuellt statt weggelassen — "je Teammitglied" heisst nicht "je
    Mitglied mit Eintraegen"). KORREKTUR zur Backlog-Sources-Notiz: `api/clients` existiert nicht —
    realer FE-Vertrag ist `useProjectTeamUtilization` (`api/hooks/useProjects.ts:522`), Typ
    `MemberUtilization`.
  - **Echter Zielkonflikt gefunden und zugunsten der Sicherheitsvorgabe geloest.** Der FE-Mock
    liefert pro Mitglied `rate` (EUR/h), das `AuslastungReport.tsx` zu einer "Personalkosten"-Kachel
    hochrechnet (`cost += hours * rate`) — exakt die Kostendarstellung, die die Backlog-Notiz
    verbietet. Einzige reale Quelle eines Stundensatzes ist `EmployeeProfile.HourlyRate`
    (`internal/models/hr.go:103`), das hinter `team:salary:view` liegt; diese Route haengt aber am
    bestehenden `projRead`-Guard (`projects:read`/`work:project:read`). `rate` durchzureichen waere
    ein Permission-Bypass — jeder mit Projekt-Lesezugriff saehe Gehaltsdaten ohne HR-Berechtigung.
    Die Antwort traegt deshalb bewusst KEIN `rate`-Feld; Wire-Shape-Test prueft das explizit (String-
    Suche nach `"rate"` im marshalten JSON, nicht nur "Feld X fehlt"). `role` kommt aus
    `ProjectMember.Role` (owner/member/viewer — bereits oeffentlich am selben Guard ueber `GET
    /projects/{id}/members`), NICHT aus den erfundenen Jobtiteln des Mocks ("Frontend Lead" etc.),
    fuer die es keine Datenquelle gibt. `weeklyTarget` ist ein fixer 40h-Default mit `lean:`-Marker:
    die echte Vertragsstundenzahl (`EmployeeProfile.WorkDaysPerWeek` x
    `HRCompanySettings.WorkHoursPerDay`) liegt hinter `team:data_job:view`, ebenfalls ausserhalb
    dieses Route-Scopes. FE bleibt fuers Kosten-Kaertchen Mock-served (gleiche Lage wie alle
    `fe-*`-Units dieses Laufs) — wer es anschliesst, braucht zuerst einen `team:salary:view`-Pfad.
  - Neuer RPC `WorkService.ListProjectTeamUtilization` (`work.proto`, Request nur `project_id`), im
    selben Commit regeneriert. Neue Messages `UtilizationPointProto{label,hours}` und
    `ProjectMemberUtilizationProto{user_id,name,role,weekly_target,weekly_data,monthly_data}`
    (bewusst KEIN `rate`-Feld im Proto selbst — der Schnitt sitzt schon an der Quelle, nicht erst am
    Gateway-Mapping).
  - `internal/work/timeentry`: neue Repository-Methode `AggregateProjectHours(ctx, projectID,
    tenantID, trunc, since)` — EIN SQL-Query fuer Wochen- UND Monats-Buckets (Parameter `trunc`
    "week"/"month" via `date_trunc($1, te.started_at AT TIME ZONE 'UTC')`, explizit UTC-truncated
    statt Session-Timezone, damit die Bucket-Grenzen exakt zu den Go-seitig erzeugten Periods-Keys
    passen), `SUM(duration_seconds) GROUP BY user_id, period` — Aggregation komplett SQL-seitig, kein
    zeilenweises Summieren in Go. `since` grenzt das Zeitfenster ein (5 Wochen bzw. 2 Monate zurueck),
    damit ein altes Projekt nicht jeden je erfassten Eintrag scannt.
  - Service `ProjectUtilization(ctx, projectID, tenantID, members, now)`: ruft `AggregateProjectHours`
    zweimal (week/month), delegiert Zero-Fill + Label-Erzeugung an die reine Funktion
    `buildMemberUtilization` (kein ctx/DB-Zugriff, daher ohne DB unit-testbar). Wochenlabel `KW <n>`
    ueber `time.Time.ISOWeek()` (Stdlib, keine neue Dependency), Monatslabel deutsche
    3-Buchstaben-Abkuerzung aus einer festen Tabelle (`Jan…Dez`) plus Jahr. Bucket-Grenzen ueber
    `startOfISOWeek`/`startOfMonth` (Montag-Anker bzw. Monatserster, beide UTC) — dieselbe Logik
    erzeugt sowohl die SQL-`since`-Grenze als auch die Go-seitigen Vergleichsschluessel, damit beide
    Seiten deckungsgleich bleiben.
  - `WorkGRPCServer.ListProjectTeamUtilization` (`internal/server/work_grpc.go`): gleiches
    Tenant/Projekt-Zugehoerigkeits-Muster wie `ListProjectTimeEntries`
    (`projectService.Get(ctx,…,uuid.Nil,true)` vor jedem Datenzugriff), dann
    `projectService.ListMembers(ctx,…,uuid.Nil,true)` fuer die vollstaendige Mitgliederliste (damit
    auch Mitglieder ohne Eintraege erscheinen), dann `timeEntryService.ProjectUtilization`.
  - Gateway: `WorkRoutes.HandleListProjectTeamUtilization` (`route_work_time.go`) haengt am
    bestehenden `projRead`-Guard unter `/api/v1/projects/{id}/team-utilization` in `route_work.go` —
    kein neuer Permission-Key, keine Seed-Pflicht. Eigenes Wire-Mapping (`toMemberUtilizationWire`)
    statt `response.Proto`, weil `avatarInitial` (aus dem Namen abgeleitet, wie im Mock) im Proto
    nicht existiert und `rate` bewusst fehlt.
  - Tests: `TestBuildMemberUtilization` (rein, keine DB) — Bucket-Platzierung fuer ein Mitglied mit
    Eintraegen (aeltester/aktueller Bucket korrekt, alles dazwischen 0-gefuellt), Wochenlabel
    unabhaengig ueber `time.Time.ISOWeek()` nachgerechnet statt gegen die eigene Funktion zirkulaer
    zu pruefen, Monatslabel exakt ("Jun/Jul/Aug 2026" bei 3 Buckets — Index 0 ist ZWEI Monate
    zurueck, nicht einer; das war ein Bug im ersten Testentwurf, nicht im Produktionscode, per
    fehlgeschlagenem Testlauf gefunden und korrigiert), Mitglied ohne jeden Eintrag komplett
    0-gefuellt statt weggelassen. `TestAggregateProjectHours_TenantIsolation` (echte DB,
    `kmuhub_app`) — Eintrag vor dem `since`-Fenster ausgeschlossen (nicht nur Tenant/Projekt-Filter),
    laufender Timer ausgeschlossen, fremder Tenant mit geratener echter `project_id` sieht 0 Buckets.
    Gateway-Wire-Shape-Test `TestToMemberUtilizationWire_Keys` prueft `member.{id,name,role,
    avatarInitial,weeklyTarget}` vorhanden, `weeklyData[0]`/`monthlyData[0]` mit `label`/`hours`, UND
    per String-Suche im marshalten JSON, dass `"rate"` an KEINER Stelle vorkommt (nicht nur "Feld X
    im erwarteten Objekt fehlt" — eine zukuenftige unbedachte Struct-Erweiterung waere sonst nicht
    abgefangen).
- verify vorgaenger: `db17bb62`/`f204eac3` (fe-projects-time-entries) gegen alle sechs
  Fehlerklassen der Architektur-Regeln geprueft. Handler ueber `getWorkClient()`, kein direkter
  Service-Zugriff; `work.proto` + beide `.pb.go` im selben Commit regeneriert; Repository
  tenant-gescoped ueber den `tasks`/`projects`-Join (`p.tenant_id = $2 AND te.tenant_id = $2`);
  keine neue Tabelle; Guard `projRead` bestehend, kein neuer Key, keine Seed-Pflicht; Wire-Shape
  `{entries:[...]}` mit `id/date/task/person/hours/description` exakt gegen
  `useProjectTimeEntries`/`ProjectTimeEntry` in `useProjects.ts` gegengelesen; openapi.yaml im
  selben Commit. Unabhaengig nachgefahren: `go build -p 2` gruen, `go vet`/`golangci-lint` 0 Issues,
  `go test ./internal/work/... ./internal/gateway/... ./internal/server/...` gruen mit
  `DATABASE_URL` gegen `kmuhub_app`, `TestListByProject_TenantIsolation` gezielt mit `-v` als real
  gelaufen (0,12 s, kein SKIP) verifiziert, `TestOpenAPIRouteDrift` gruen (756 Routen/758 Pfade zu
  diesem Zeitpunkt). Kein Fund.
- gate: build ok (`go build -p 2 ./...`, ganzes Backend) | vet ok (`go vet ./...`) | lint ok
  (`golangci-lint`, 0 issues auf gateway+work+server) | gofmt: KEINE neue Unformatierung — `git
  stash`/gofmt/`stash pop` gegenprobiert, dieselben zehn Dateien (`time_entry.go`,
  `timeentry/errors.go|postgres_repository.go|repository.go|service.go|service_test.go|
  tenant_write_test.go`, `server/work_grpc.go|work_label_test.go`, `gateway/route_work_time.go`)
  waren bereits VOR meinen Aenderungen unformatiert (Bestandsschulden wie in Iteration 8
  dokumentiert) — meine drei neuen/erweiterten Dateien (`route_work.go`,
  `route_work_time_test.go`, `tenant_isolation_phase2_test.go`) tauchen in keiner der beiden Listen
  auf. | test ok mit `DATABASE_URL` gegen `kmuhub_app`: `internal/work/...` (alle Unterpakete inkl.
  `timeentry`: 29 PASS, 0 SKIP per `-v`), `internal/gateway/`, `internal/server/...`,
  `internal/models/...` (keine Testdateien) alle gruen. `TestOpenAPIRouteDrift` gruen: 757
  registrierte Routen gegen 759 dokumentierte Pfade (+1 gegenueber Iteration 31, wie erwartet).
  migration n.a. (keine Schema-Aenderung — reine Aggregation auf `time_entries`/`tasks`/`projects`).
  rls-smoke: `TestAggregateProjectHours_TenantIsolation`, Details oben.
- offen:
  - **Produktentscheidung fuer Luke:** das FE-Personalkosten-Kaertchen in `AuslastungReport.tsx`
    bleibt ohne echten Backend-Wert (Mock-`rate`), weil ein korrekter Wert `team:salary:view`
    braucht, das dieser project-read-gegatete Endpoint nicht hat. Optionen: (a) eigene
    salary-gegatete Zusatzroute, die das FE nur fuer berechtigte User abruft; (b) die Kostenkachel im
    FE ganz entfernen/hinter eine eigene Capability legen; (c) so lassen bis ein Kunde das Feature
    real verlangt. Keine dieser Optionen wurde umgesetzt — reine Beobachtung.
  - `weeklyTarget` (40h-Default) und `role` (Projekt-Rolle statt Jobtitel) weichen bewusst vom
    FE-Mock ab; sollte das FE spaeter von Mock auf diesen Endpoint umgestellt werden, aendert sich
    dort sichtbar die Darstellung (kein "Frontend Lead" mehr, sondern "owner"/"member"/"viewer").
  - Naechste Unit laut Backlog-Reihenfolge: `fe-projects-guest-overview` (deps erfuellt). Deren
    Notes verlangen explizit eine schmale, handgebaute Feldliste ohne interne Notizen/Kosten/
    Mitarbeiterdaten fuer die Rolle `extern` — dieselbe Guard-Kategorie (`projRead`) wie hier, selbes
    Risiko falls versehentlich HR- oder Kostenfelder durchgereicht werden.

## Iteration 33 — fe-projects-guest-overview — blocked — 2026-08-02 00:20 (Nachtlauf 3)
- commit: - (keine Code-Aenderung, nur BACKLOG.yml-Status)
- verify vorgaenger: sauber. `3f26f98a` (fe-projects-team-utilization) gegen alle acht
  Fehlerklassen geprueft: Handler ueber `getWorkClient()`, kein direkter Service-Zugriff;
  `work.proto` + `.pb.go`/`_grpc.pb.go` im selben Commit regeneriert; kein neuer
  `RequirePermission`-Guard (haengt am bestehenden `projRead`), also auch keine Seed-Pflicht und
  kein verlorener Alt-Key; kein neuer Table, Repository tenant-gescoped ueber
  `te.tenant_id = $3 AND p.tenant_id = $3` im JOIN; Wire-Shape `{team:[...]}` mit
  `member.{id,name,role,avatarInitial,weeklyTarget}` + `weeklyData`/`monthlyData` exakt gegen
  `MemberUtilization`/`useProjectTeamUtilization` in `useProjects.ts:26-38,522-533` gegengelesen —
  das bewusst fehlende `rate`-Feld ist dokumentierte Sicherheitsentscheidung (team:salary:view),
  kein uebersehener Drift; openapi.yaml-Eintrag vorhanden. Unabhaengig nachgefahren:
  `go test -count=1 -v ./internal/work/timeentry/... -run
  'TestAggregateProjectHours_TenantIsolation|TestBuildMemberUtilization'` gegen `kmuhub_app` real
  gelaufen (2 PASS, 0 SKIP, 0.24s). Kein Fund.
- gebaut: nichts — Unit als `blocked` markiert, siehe `blocked_reason` in BACKLOG.yml.
- Grund (Kurzfassung, Volltext im Backlog-Eintrag): Fuer `GET
  /api/v1/projects/{id}/guest-overview` erwartet der reale FE-Vertrag
  (`useProjectGuestOverview`, `useProjects.ts:536`) als GESAMTE Antwort
  `{milestones: GuestMilestone[], statusUpdates: GuestStatusUpdate[]}`. Fuer beide Konzepte
  existiert im Backend NICHTS — keine Tabelle, kein Model, kein interner Schreibpfad (verifiziert
  per Volltextsuche ueber `backend/`, inkl. Migrations und `internal/models`). Der MSW-Mock
  (`mocks/handlers/work.ts:1087`) erzeugt "Milestones" aus sechs hartkodierten Platzhaltertiteln
  ("Konzept & Setup" ... "Go-Live"), gleichmaessig ueber Start-/Enddatum verteilt — reine
  Design-Preview-Fiktion, kein Bezug zu echten Projektdaten. Diese Route ehrlich zu bauen braucht
  zuerst ein echtes Datenmodell (mind. zwei neue Tabellen) UND einen internen Schreibpfad, damit
  ein PM-User ueberhaupt Milestones/Status-Updates anlegen kann — beides existiert nirgends, auch
  nicht ausserhalb der Guest-View. Ohne Schreibpfad waere die Route dauerhaft leer (Fehlerklasse 2,
  "leerer Return" hinter einem echten GET). Interne Task-Kommentare als Ersatzquelle zu nehmen
  wuerde die Sicherheitsvorgabe der Unit verletzen ("keine internen Notizen" fuer `extern`). Die
  FE-Platzhalterfiktion serverseitig nachzubauen waere umgekehrt ein hartkodierter Beispieldatensatz
  hinter einer echten Route. Alle drei Auswege (neues Feature aufsetzen / Milestones aus
  vorhandenen Daten ableiten und Status-Updates streichen / Route auf echte Projekt-Kernfelder
  beschraenken) sind Produkt-/Architekturentscheidungen — nicht spontan zu treffen.
- gate: n.a. (kein Code geaendert)
- offen:
  - **Entscheidung fuer Luke:** siehe `blocked_reason` in BACKLOG.yml, drei Optionen zur Auswahl.
  - Naechste Unit laut Reihenfolge: `fe-customization-labels` (deps: [], unabhaengig von dieser
    blockierten Unit).

## Iteration 34 — fe-customization-labels — done — 2026-08-02 00:40 (Nachtlauf 3)
- commit: 528a7868
- verify vorgaenger: `95ff9dbc` (Iteration 33) war reiner Doku-Commit (BACKLOG.yml/JOURNAL.md,
  keine Code-Zeile) — nichts zu verifizieren. Der letzte echte Code-Commit `3f26f98a`
  (fe-projects-team-utilization) wurde bereits in Iteration 33 gegen alle Fehlerklassen geprueft
  und als "Kein Fund" journalisiert; keine erneute Pruefung noetig.
- gebaut:
  - `GET /api/v1/customization/labels?locale=de[&base=1]` und
    `PUT /api/v1/customization/labels` — tenant-eigene Umbenennungen von UI-Begriffen
    (`desktop/src/renderer/src/api/customization-types.ts`). Scope bewusst so eng wie in der
    Backlog-Note verlangt: nur die Tenant-Overlay-Schicht von Label-Overrides, NICHT Value-Sets,
    Drafts/Scheduling oder der Vendor-Resolver — die bleiben unangetastet.
  - **Keine neue Tabelle, keine neue Migration, keine neue Proto-RPC.** `tenant_settings`
    (Migration 000138, RLS seit 000218 per `enable_tenant_rls`) traegt das bereits generisch als
    `(tenant_id, module_id, key) -> JSONB`. Neue Datei `backend/internal/gateway/route_customization.go`
    (`CustomizationRoutes`, eigener `RouteRegistrar`) ruft dafuer ausschliesslich die BEREITS
    vorhandenen `SettingsService.GetTenantSettings`/`PutTenantSettings`-RPCs mit
    `module_id="customization"` auf — ein Row pro Locale, Value ist die sparse
    `{labelKey: overrideText}`-Map (via `structpb.Value`/`AsInterface()` hin- und rueckkonvertiert).
    Damit ist dies der erste Consumer, der die Settings-Foundation-Infrastruktur (Sprint 4/07-xx)
    fuer einen fachfremden Zweck zweitverwertet, statt einen neuen Dienst zu bauen.
  - Whitelist-Filterung (nur `LABEL_WHITELIST`-Keys aus `mocks/data/customization.ts` werden
    angenommen, unbekannte Keys still verworfen — identisch zum MSW-Mock-Verhalten) und die
    Default/Tenant-Provenance-Aufloesung sind pure Funktionen (`resolveLabels`,
    `applyLabelOverrides`, `structValueToStringMap`) im Gateway-Package, nicht im Handler selbst —
    Praezedenzfall dafuer ist `internal/modules`/`toTenantModuleJSON` (statische Katalogdaten direkt
    im Gateway, kein gRPC-Umweg fuer reine Shape-Transformation auf bereits tenant-gescopten Daten).
  - **Fund beim Bau:** `desktop/src/renderer/src/api/hooks/useLabelOverrides.ts` ist KEIN totes
    Type-File, sondern ein echter, bereits verdrahteter Consumer
    (`modules/admin/anpassungen/BegriffeTab.tsx`) — `useLabelOverrides`/`useSetLabelOverrides`/
    `useResetLabelOverride`/`useResetAllLabelOverrides` sind dort live im Editor genutzt. Der Hook
    `useLabelDefaults` (fuer `?base=1`, "Cosmi-Standard"-Baseline-Ansicht) ist im Hook-File definiert,
    aber aktuell in KEINER Seite importiert — trotzdem mitgebaut, weil der Vertrag real existiert und
    ein spaeteres Wieder-Einschalten sonst still falsche Daten (Tenant-Overrides statt Default)
    gezeigt haette. `base=1` überspringt den `GetTenantSettings`-Call komplett (jeder Whitelist-Key
    kommt mit `provenance="default"`/`value=""` zurueck), getestet auch mit absichtlich kaputter
    Settings-Verbindung (`registryWithService`, Adresse loest nie real auf) — beweist, dass der
    Fetch fuer `base=1` wirklich uebersprungen wird und nicht nur zufaellig durchlaeuft.
  - Guard: `RequirePermission("admin:customization","manage")` NUR auf PUT — GET ist fuer jeden
    authentifizierten User offen, weil aufgeloeste Labels UI-Text fuer ALLE User treiben, nicht nur
    fuer die, die ihn editieren duerfen (gleiches Muster wie `/auth/me/permissions`). Kein neuer
    Seed noetig: `admin:customization:manage` steht bereits seit `p1a-migration`/000256 im
    Gesamt-Katalog-Seed und ist an `admin`+`it_admin` vergeben (gegen die lokale DB verifiziert,
    nicht angenommen). Da dies eine BRANDNEUE Route ist (nicht die additive Verschaerfung eines
    Bestandsguards aus Block B), gibt es keinen Alt-Key zu bewahren — der einzelne Fine-Key reicht.
  - Vendor-Layer bewusst abgelehnt: `PUT` mit `layer:"vendor"` liefert 400 statt den Request
    stillschweigend in die Tenant-Schicht zu schreiben. Grund: R-5 (GDAP-Vendor-Access) ist nicht
    verdrahtet, `activeConfigLayer()` im FE-Mock liefert selbst in v1.0 immer nur `"tenant"` — ein
    Vendor-Schreibpfad waere unbenutzt bis dahin und ein falsch benannter Layer haette sonst leise
    im Tenant-Overlay gelandet.
  - Tests (`route_customization_test.go`, neu): pure Funktionstests fuer `resolveLabels` (Default-
    vs. Tenant-Provenance, nicht-Whitelist-Key filtert sich raus), `applyLabelOverrides`
    (unbekannter Key verworfen, leerer Wert loescht, bestehende unberuehrte Keys bleiben stehen),
    `structValueToStringMap` (Objekt-Roundtrip, nicht-String-Werte defensiv verworfen, nil/Array
    liefert nil) sowie ein Wire-Shape-Test (`json.Marshal` gegen den exakten erwarteten String, wie
    bei `TestToEffectivePermissionsBody_WireShape`). HTTP-Verdrahtungstests ueber
    `emptyRegistry()`/`registryWithService()` (Muster aus `testutil_test.go`): GET ohne Guard
    erreicht den Handler auch mit leerem Permission-Slice (503 statt 403), PUT ohne
    `admin:customization:manage` liefert 403, PUT mit dem Key erreicht den Handler (503 an der
    leeren Registry, nicht 403), Vendor-Layer-Ablehnung, ungueltige Locale (400), `base=1`-Erfolg
    trotz kaputter Verbindung.
  - `backend/api/openapi.yaml`: neuer Tag `Customization`, Pfad `/api/v1/customization/labels`
    (GET+PUT) sowie Schemas `ResolvedLabel`/`LabelOverridesResponse`/`UpdateLabelOverridesRequest`
    im selben Commit.
- gate: build ok (`go build -p 2 ./internal/gateway/... ./cmd/gateway/...`) | vet ok | lint ok
  (`golangci-lint`, 0 issues — drei Anfangsfunde selbst behoben: `interface{}` -> `any`,
  Merge-Loop -> `maps.Copy`) | gofmt: `route_customization.go`/`route_customization_test.go` sauber;
  `openapi_drift_test.go`/`cmd/gateway/main.go` per `git stash`/gofmt/`stash pop` gegengeprueft —
  waren bereits VOR meiner Ein-Zeilen-Aenderung unformatiert (Bestandsschulden), meine Zeile selbst
  nicht betroffen. | test ok: `go test -count=1 -v ./internal/gateway/` — 571 PASS, 0 FAIL, 0 SKIP.
  `TestOpenAPIRouteDrift` gruen: 758 registrierte Routen gegen 760 dokumentierte Pfade (+1 gegenueber
  Iteration 32, wie erwartet — GET+PUT teilen sich einen Pfad-Eintrag). `npx @apidevtools/swagger-cli
  validate backend/api/openapi.yaml` gruen (identisch zum CI-Job `openapi-validate`). migration: keine
  (reine Zweitverwertung von `tenant_settings`). rls-smoke: manuell gegen `tenant_settings` mit
  `module_id='customization'` unter `kmuhub_app` gefahren (Muster aus GATE-COMMANDS.md) — eigener
  Tenant 1, fremder Tenant 0, Testzeile danach wieder geloescht. Kein Fund.
- offen:
  - `useLabelDefaults`/`?base=1` ist backend-seitig fertig, aber FE-seitig noch nirgends verbaut
    (kein Import ausserhalb des Hook-Files) — reine Beobachtung, keine Aktion noetig.
  - Value-Sets (`/customization/value-sets[/:id]`), Drafts/Scheduling (`/customization/drafts`) und
    der Vendor-Overlay-Schreibpfad (R-5-Anbindung) sind im FE-Type-File vollstaendig spezifiziert,
    aber laut dieser Unit explizit nicht im Scope — eigene Folge-Units, falls gewuenscht.
  - Naechste Unit laut Reihenfolge: `fe-email-rules` (Block E, deps: [], model: opus).

## Iteration 35 — fe-email-rules — done — 2026-08-02 01:05 (Nachtlauf 3)
- commit: (siehe git log, dieser Lauf)
- verify vorgaenger: `528a7868` (fe-customization-labels) gegen die sechs Fehlerklassen geprueft —
  **kein Fund**. Handler geht ueber `settingsv1.NewSettingsServiceClient(conn)` (kein direkt
  injizierter Service), Tenant kommt aus `middleware.GetTenantID(r.Context())`, Guard
  `RequirePermission("admin:customization","manage")` nur auf PUT mit vorhandenem Seed,
  `openapi.yaml` im selben Commit, kein Proto angefasst, keine Stubs/`TODO`/`panic` in den neuen
  Dateien.
- gebaut:
  - **E-Mail-Regeln (Regeln & Filter)**, fuenf Routen im Gateway, alle ueber den email-gRPC-Client:
    `GET /api/v1/email/rules`, `POST /api/v1/email/rules`, `PATCH /api/v1/email/rules/{id}`,
    `DELETE /api/v1/email/rules/{id}`, `POST /api/v1/email/rules/apply`.
    Vertrag ist `EmailRuleInfo` aus `desktop/src/renderer/src/api/email-types.ts:181` — Felder
    1:1 uebernommen (`field`/`op`/`value`/`action_type`/`action_target`), gegen den FE-Typ und den
    MSW-Handler geprueft, nicht geraten. Konsument ist der bereits verdrahtete
    `modules/mails/RulesDialog.tsx` (`useEmailRules`/`useCreateRule`/`useDeleteRule`/
    `useApplyRules`).
  - **WIEDERVERWENDUNGS-ENTSCHEIDUNG (von der Unit ausdruecklich verlangt):**
    - `internal/automation/condition` **wird wiederverwendet**. `Evaluator.Evaluate` mit
      `ConditionConfig{Mode: simple, Simple: &Condition{Field, Operator, Value}}` deckt das
      Regel-Matching vollstaendig ab, und `OpContains` ist bereits
      `strings.Contains(ToLower(field), ToLower(needle))` — bit-genau die Semantik des FE-Mocks
      (`email-store.ts:569 ruleMatches`). Es entsteht KEIN zweiter Matcher im Mail-Modul. Die
      DB-CHECKs auf `field`/`op` sind bewusst CHECK statt Enum-Typ: der Evaluator kennt schon 15
      Operatoren, das Aufbohren ist damit ein reines ALTER ohne Service-Koordination.
    - `internal/automation/action` **wird NICHT wiederverwendet**. Das `ActionExecutor`-Interface
      arbeitet ueber `json.RawMessage`-Configs plus globale `ActionRegistry`-Registrierung und
      zielt auf modul-uebergreifende Workflow-Schritte; hier gibt es genau zwei fest verdrahtete
      Aktionen (`label`, `move`) im selben Service auf derselben Tabelle. Der Registry-Umweg waere
      mehr Code als der direkte Aufruf und wuerde eine Executor-Verdrahtung in `cmd/email`
      erzwingen, die sonst niemand braucht. Es gibt dort ausserdem nur `email.send`, keine
      Label-/Move-Aktion, die man haette nutzen koennen.
  - Migration **000260** (Repo-Kopf war 259): Tabelle `email_rules` mit `tenant_id NOT NULL`
    (FK auf `tenants`), CHECKs auf `field`/`op`/`action_type` sowie gegen leeren Namen und leeren
    Suchwert (ein leerer Needle matcht JEDE Nachricht), Index `(tenant_id, created_at, id)`,
    `CALL enable_tenant_rls('email_rules')`. Up und down lokal gefahren (`260/u`, `260/d`, `260/u`).
  - `action_target` ist bewusst eine blanke UUID ohne FK: sie zeigt bei `move` auf einen Folder,
    bei `label` auf ein Label. Eine polymorphe Referenz laesst sich nicht als FK ausdruecken; die
    Alternative (zwei nullable Spalten + CHECK) haette Integritaet nur fuer die Folder-Haelfte
    gebracht und eine Shape erzeugt, die der FE nicht nutzt. Stattdessen validiert der Service.
  - **SICHERHEIT — Move-Ziel wird gegen den Tenant geprueft.** Ohne diese Pruefung koennte eine
    Regel Nachrichten in den Ordner eines FREMDEN Tenants schieben, und RLS auf `email_messages`
    faengt das nicht: die Zeile behaelt beim Folder-Wechsel ihre eigene `tenant_id`, die Policy
    sieht also nichts Verdaechtiges. `FolderBelongsToTenant` laeuft direkt gegen
    `email_folders.tenant_id` — **Fund beim Bau:** die Spalte ist entgegen der ersten Annahme
    vorhanden und NOT NULL (Option-B-Retrofit, RLS aktiv), der zuerst gebaute Join ueber
    `email_accounts` war unnoetig und wurde entfernt.
  - `email_messages.label_ids UUID[] NOT NULL DEFAULT '{}'` in derselben Migration. **Begruendung,
    warum das hier und nicht erst in `fe-email-labels` liegt:** `action_type: 'label'` ist die
    Default-Aktion im FE-Formular (`RulesDialog.tsx:37`), eine Label-Regel haette ohne diese Spalte
    kein Ziel und `apply` waere fuer den haeufigsten Fall wirkungslos gewesen. Diese Unit legt nur
    den Speicher an und schreibt ihn; die Label-Stammdaten (`email_labels`) und die READ-Seite
    (`label_ids` im Proto + in den `message`-SELECTs) bleiben bewusst bei `fe-email-labels` — dort
    als Vorarbeit-Block in den Backlog-Notes hinterlegt, inklusive des Hinweises, dass die
    Spaltenliste in `message/postgres_repository.go` an mehreren Stellen steht.
  - Neues Paket `internal/email/rule/` nach dem Muster von `internal/email/signature/`
    (errors/repository/postgres_repository/service). Thick service: Validierung, Merge-Semantik
    des Patches und die gesamte Apply-Logik liegen in `service.go`; das Repository ist reine
    Persistenz, der gRPC-Handler mappt nur.
  - PATCH ist echtes Partial-Update (`optional`-Felder im Proto -> `*string` in `RuleInput`).
    Validiert wird immer die GEMERGTE Regel, nie der Patch allein — sonst koennte man
    `action_type` auf `move` schalten und die Label-UUID als Ordner-Ziel stehen lassen. Dafuer gibt
    es einen eigenen Test.
  - `apply`-Grenze: `applyScanLimit = 2000`, neueste zuerst, Papierkorb ausgenommen, mit
    `lean:`-Marker und Upgrade-Trigger ("Background-Job mit Cursor, sobald ein Postfach das
    regelmaessig reisst"). Die Antwort traegt neben `affected` auch `scanned`, damit "nichts
    gematcht" von "das Limit hat den Lauf abgeschnitten" unterscheidbar bleibt.
  - Regel-Semantik bewusst identisch zum FE-Mock gehalten (damit die Umstellung vom Mock aufs
    Backend das sichtbare Verhalten nicht aendert): Regeln in Anlage-Reihenfolge, Labels
    akkumulieren, ein spaeterer `move` ueberschreibt einen frueheren, `affected` zaehlt GEAENDERTE
    NACHRICHTEN (nicht Treffer) und ein zweiter Lauf ist damit idempotent. Der `from`-Heuhaufen
    ist `from_name + " " + from_email`, weil der FE nur ein einziges "Absender"-Feld anbietet.
  - **WIRE-SHAPE-FALLE (zweimal umschifft, beide Male protojson):**
    (a) `GET /rules` geht ueber `response.ProtoListWrapped(..., "rules", ...)`, damit eine leere
        Regelmenge als `{"rules": []}` und nicht als `{}` rausgeht.
    (b) `POST /rules/apply` baut die Antwort EXPLIZIT als Map. `response.Proto` haette bei
        `affected == 0` wegen `EmitUnpopulated:false` ein blankes `{}` geliefert, der FE liest
        aber `res.affected` und haette `undefined` in den ICU-Plural gereicht
        (`toast.success(t('mails.rules.applied', {count: res.affected}))`).
- gate: build ok (`go build -p 2 ./...`) | vet ok | lint ok (`golangci-lint`, **0 issues**) |
  test ok mit `DATABASE_URL` gesetzt (Rolle `kmuhub_app`, NOSUPERUSER NOBYPASSRLS):
  `./internal/email/... ./internal/gateway/ ./internal/server/... ./internal/models/...` alle `ok`;
  die vier `TestRepository_*`-DB-Tests explizit mit `-v` gegengeprueft, dass sie PASS und nicht
  SKIP melden. `TestOpenAPIRouteDrift` gruen. `npx @apidevtools/swagger-cli validate` gruen
  (identisch zum CI-Job `openapi-validate`).
  gofmt: die vier von mir angefassten Bestandsdateien (`route_email.go`, `email_grpc.go`,
  `models/email.go`, `cmd/email/main.go`) meldet `gofmt -l` als unformatiert — das ist der lokale
  CRLF-Zustand (`core.autocrlf=true`), nicht meine Aenderung: voellig unangetastete Nachbardateien
  (`internal/email/signature/*`, `message/service.go`, `server/crm_grpc.go`) melden dasselbe. Meine
  vier NEUEN Dateien unter `internal/email/rule/` sind sauber.
  migration: 000260 up/down/up lokal gruen. rls-smoke auf `email_rules` als `kmuhub_app`: Tenant A
  sieht 1 Zeile (die eigene), Tenant B sieht 1 Zeile (die eigene) — **keine 0/0-Nullmessung**;
  Cross-Tenant-INSERT scheitert mit `new row violates row-level security policy`,
  Cross-Tenant-UPDATE trifft `UPDATE 0`. Testzeilen danach geloescht (`SELECT count(*)` = 0).
- tests: 11 Service-Unit-Tests (Validierung inkl. 7 Ablehnungsfaelle, Partial-Update-Merge,
  Fremd-Tenant = NotFound, Apply-Semantik: Label+Move, Idempotenz, Akkumulation, Last-Move-Wins,
  Scan-Limit), 4 DB-Tests (`postgres_repository_test.go`, eigene frisch geminzte Tenants statt
  `TenantA`/`TenantB` — die Fixtures laufen in `UNIQUE(user_id)` auf `email_accounts` und
  `UNIQUE(account_id, imap_name)` auf `email_folders`, geteilte Konstanten kollidieren dort unter
  `t.Parallel()`), 12 Gateway-Guard-Faelle plus ein Routing-Test, dass `/rules/apply` nicht von der
  `{id}`-Route verschluckt wird.
- offen:
  - **Der MSW-Mock bleibt aktiv** (`mocks/handlers/email.ts`) — die FE-Umstellung auf das echte
    Backend ist nicht Teil dieser Backend-Unit. Beim Umschalten faellt auf: die Mock-Seeds nutzen
    IDs wie `lbl-rechnung`/`rule-1`, das Backend verlangt echte UUIDs fuer `action_target` (400 bei
    allem anderen). Das ist Absicht, aber der Mock-Datensatz ist so nicht 1:1 uebertragbar.
  - `label`-Ziele werden NICHT auf Existenz geprueft (`lean:`-Marker in `service.go`), weil es
    `email_labels` noch nicht gibt. Trigger steht in den Notes von `fe-email-labels`.
  - Fund im Bestand, ausserhalb dieser Unit: `message.Repository.MoveToFolder(ctx, id, folderID)`
    und `UpdateFlags`/`Delete` tragen **kein** `tenantID`-Argument — der Schutz haengt dort allein
    an RLS. Nicht angefasst (Scope), aber ein Kandidat fuer eine eigene Haertungs-Unit; die neuen
    Rule-Schreibpfade sind bewusst explizit tenant-gescoped gebaut.
  - Naechste Unit laut Reihenfolge: `fe-email-labels` (Block E, deps: [fe-email-rules], model:
    sonnet) — die Vorarbeit-Punkte (a)-(c) stehen jetzt in deren Backlog-Notes.

## Iteration 37 — fe-messages-bookmark — done — 2026-08-02 (Nachtlauf 3)
- commit: `41b7c4e1`
- verify vorgaenger: `50dbc8e3` (fe-email-labels) gegen die acht Fehlerklassen geprueft — **kein
  Fund**. Migration 000261 `tenant_id NOT NULL` + `CALL enable_tenant_rls('email_labels')`, alle
  Repo-Methoden (Create/GetByID/List/Update/Delete/LabelIDsBelongToTenant/AssignToMessage)
  tenant-gescopt gegengelesen. Gateway-Handler gehen ausschliesslich ueber `e.getEmailClient()`
  (kein Fehlerklasse-1-Fund). Proto `optional string name/color` in `UpdateEmailLabelRequest`
  passt exakt zur Service-Signatur `Update(..., name, color *string)` — `.pb.go` im selben Commit
  regeneriert. Guards nutzen ausschliesslich bestehende `email:read/write/delete` (kein neuer Key,
  kein Seed noetig). `openapi.yaml` traegt alle vier Pfade.
- gebaut: **Message-Bookmarks** (persoenliche Lesezeichen an Chat-Nachrichten).
  - SCOPE-KORREKTUR gegen den Backlog-Text (siehe BACKLOG-Notes): kein POST/DELETE-Paar. Der
    bestehende FE-Hook `useBookmarks.ts` (raw fetch, demo-only, MSW-Mock in
    `mocks/handlers/chat.ts:141-150`) legt eindeutig einen **Toggle**-Endpoint fest
    (`POST /api/v1/messages/{id}/bookmark` -> `{bookmarked: boolean}`) plus einen zweiten, im
    Scope-Text nicht erwaehnten Read-Endpoint (`GET /api/v1/messages/bookmarks` ->
    `{messages: MessageInfo[]}`). Gegen den FE-Vertrag gebaut, nicht gegen den Scope-Titel.
  - Migration 000262: `message_bookmarks` (`tenant_id`/`message_id`/`user_id` alle `NOT NULL` +
    FK, zusammengesetzter PK `(user_id, message_id)`, Index auf `message_id`) — Form bewusst
    identisch zu `message_reactions` (Migration 000038/000115/000122), aber `tenant_id NOT NULL`
    von Anfang an, weil es hier kein Retrofit ist. `CALL enable_tenant_rls('message_bookmarks')`.
  - Neues Package `internal/chat/bookmark/` (Repository + PostgresRepository + Service, Muster von
    `internal/email/label/` uebernommen): `Service` komponiert **keine eigene SQL-Query fuers
    Lesen**, sondern ein schlankes `MessageReader`-Interface
    (`GetByID(ctx, id, tenantID, userID) (*models.MessageWithSender, error)`), das
    `*message.Service` strukturell erfuellt. Das ist bewusst dieselbe Zusammensetzung wie
    `label.Service`/`MessageReader` fuer Email, aber mit einem wichtigen Unterschied: die
    wiederverwendete `message.Service.GetByID` prueft nicht nur Tenant-Zugehoerigkeit, sondern
    auch **Channel-Mitgliedschaft** (`ErrNotChannelMember`) — genau die Zugriffspruefung, die ein
    Lesezeichen-Feature ohnehin braucht ("darf dieser User diese Nachricht ueberhaupt sehen").
    `Toggle` ruft `GetByID` VOR jedem Schreiben, `List` ruft `GetByID` pro Bookmark-Treffer und
    ueberspringt `ErrMessageNotFound`/`ErrNotChannelMember` still (Kommentar im Code), damit ein
    Channel-Austritt nach dem Bookmarken nicht die komplette Liste zum 403/404 macht — der
    Mitgliedschafts-Check ist eine lebende Zugriffskontrolle, keine Momentaufnahme beim Bookmarken.
  - Zwei neue RPCs `ToggleBookmark`/`ListBookmarks` in `chat.proto`, `.pb.go`/`_grpc.pb.go` im
    selben Commit regeneriert. `ChatGRPCServer` bekam ein fuenftes Feld `bookmarkService
    *bookmark.Service` (Konstruktor-Signaturbruch dokumentiert wie beim `reactionService`-Vorbild
    aus Welle 8), `cmd/chat/main.go` verdrahtet `bookmarkService := bookmark.NewService(bookmarkRepo,
    messageService)` — derselbe `messageService`, der auch fuer den `MessageInfo`-Chat-Pfad
    verwendet wird, keine zweite Instanz.
  - Gateway: `GET /api/v1/messages/bookmarks` (statischer Pfad VOR `/{id}/thread` registriert,
    gleicher Chi-Ambiguitaets-Grund wie bei `POST /reactions/summary`) und
    `POST /api/v1/messages/{id}/bookmark`, beide ueber `RequirePermission("messages","read"/
    "write")` — bestehender Key, kein neuer Guard, kein Seed. Response-Pfad nutzt
    `response.JSON(w, status, resp)` mit dem rohen Proto-Struct, identisch zu `HandleGetMessages`/
    `HandleToggleReaction` in derselben Datei (die generierten `.pb.go`-json-Tags sind bereits
    snake_case, `response.Proto`/protojson also hier nicht noetig).
  - `openapi.yaml`: zwei neue Pfade (`chat-messages`-Tag) + zwei neue Schemas
    (`ToggleBookmarkResponse`, `ListBookmarksResponse` mit `$ref` auf das bestehende
    `MessageInfo`-Schema). Alle real moeglichen Codes dokumentiert: `POST .../bookmark` ->
    200/400/401/403 (`ErrNotChannelMember`)/404 (`ErrMessageNotFound`) — anders als das
    Reaction-Vorbild, das nur 200/400/401 dokumentiert, weil Reaktionen Existenz/Mitgliedschaft gar
    nicht pruefen; hier sind 403/404 echte, ueber `mapChatError` bereits gemappte Ergebnisse.
- gate: `go build -p 2 ./...` (voller Baum) gruen | vet gruen | `golangci-lint`
  `internal/chat/... internal/gateway/... internal/server/... cmd/chat/...` **0 issues** |
  Migration 000262 up/down/up lokal gruen (Kopf `262`, bit-identisch nach Roundtrip) | test ok mit
  `DATABASE_URL` gesetzt (Rolle `kmuhub_app`): `internal/chat/bookmark` 9 Faelle **0 Skips**
  (3 Service-Unit-Tests mit In-Memory-Mocks fuer Toggle, 3 fuer List inkl. des
  "revoked access wird uebersprungen"-Falls, 3 DB-Tests mit echten Tenant/User/Channel/Message-
  Fixtures — eigene Tenants pro Testfall, keine geteilten Konstanten), `internal/chat/...`
  komplett gruen, `internal/server/...` gruen (kein neuer `mapChatError`-Fall noetig — Toggle
  propagiert die bereits gemappten `message.ErrMessageNotFound`/`ErrNotChannelMember` direkt).
  `go test ./internal/gateway/ -run TestOpenAPIRouteDrift`: PASS, 766 registrierte gegen 768
  dokumentierte Pfade.
  **RLS-Smoke** (manuell, `docker exec -i ... psql`, Transaktion mit `ROLLBACK`, keine
  Datenspuren — Fixtures mit echten `tenants`/`users`/`channels`/`messages`-Zeilen statt der
  GATE-COMMANDS.md-Fixwerte, die bei `email_labels` schon mal an einer fehlenden FK-Zeile
  gescheitert waren): eigener Tenant 1, fremder Tenant 0. Verteilung vor dem Rollenwechsel zeigte
  je 1 Zeile pro Tenant — keine Nullmessung.
- verify eigen: n.a. (kein Vorgaenger-Commit auf dieser Iteration zu pruefen ausser dem oben
  behandelten `50dbc8e3`).
- offen:
  - Der FE-Hook bleibt bewusst unveraendert (raw fetch, kein OpenAPI-generierter Client) — die
    Response-Shape wurde exakt gegen das gebaut, was `useBookmarks.ts`/`useToggleBookmark`
    erwarten, nicht umgekehrt migriert.
  - `message_bookmarks` haengt an `messages.id`, nicht an `channel_memberships` — verlaesst ein
    User einen Channel wieder, bleibt die Bookmark-Zeile bestehen (nur unsichtbar in `List`, weil
    `GetByID` dann `ErrNotChannelMember` wirft). Kein Cleanup-Job dafuer; `lean:`-Marker bewusst
    NICHT gesetzt, weil das keine Vereinfachung ist, sondern die gewaehlte Semantik ("Zugriff ist
    live, nicht eingefroren") — sollte das je gewechselt werden (Bookmark-Sichtbarkeit einfrieren
    zum Zeitpunkt des Bookmarkens), ist das eine bewusste Produktentscheidung, keine Nacharbeit.
  - Naechste Unit laut Reihenfolge: `fe-notifications-snooze` (Block D, deps: [], model: sonnet).

## Iteration 36 — fe-email-labels — done — 2026-08-02 01:35 (Nachtlauf 3)
- commit: (siehe git log, dieser Lauf)
- verify vorgaenger: `087e5e0a` (fe-email-rules) gegen die sechs Fehlerklassen geprueft — **kein
  Fund**. Guards reused `email:read/write/delete` (keine neuen Keys, kein Seed noetig), Handler gehen
  ausschliesslich ueber `e.getEmailClient()`, `.proto` und `.pb.go`/`.grpc.pb.go` im selben Commit
  regeneriert, Migration 000260 `tenant_id NOT NULL` + `enable_tenant_rls`, `openapi.yaml` im selben
  Commit, `TestOpenAPIRouteDrift` laut Journal gruen geprueft, kein `Unimplemented`/`TODO` im neuen
  Pfad.
- gebaut:
  - **E-Mail-Labels**, Migration 000261: `email_labels` (`tenant_id NOT NULL` + FK auf `tenants`,
    `color` mit Hex-CHECK, `UNIQUE(tenant_id, name)` — **pro Tenant, nicht global**, wie in den
    Vorarbeit-Notes verlangt), `CALL enable_tenant_rls('email_labels')`. Die Zuordnungsspalte
    (`email_messages.label_ids`) existiert bereits seit 000260 — keine zweite Zuordnungstabelle.
  - Neues Package `internal/email/label/` (errors/repository/postgres_repository/service, Muster von
    `internal/email/rule/` uebernommen): `Create/GetByID/List/Update/Delete` + `AssignToMessage`.
    `Delete` laeuft in einer Transaktion und entfernt die Label-ID zusaetzlich per
    `array_remove(label_ids, $1)` aus ALLEN Nachrichten des Tenants — `label_ids` hat bewusst keinen
    FK (polymorphe Zuordnung, siehe 000260), sonst blieben tote IDs nach einem Delete stehen (Fund:
    genau das macht der FE-Mock in `email-store.ts:deleteLabel` schon, Backend musste nachziehen).
  - **READ-SEITE NACHGEZOGEN** (Vorarbeit-Punkt b aus `fe-email-rules`): `label_ids` jetzt in
    `EmailMessageInfo` (Proto Feld 26) UND in allen SIEBEN SELECT-Stellen von
    `internal/email/message/postgres_repository.go` (GetByID/GetByFolderUID/ListByFolder/
    ListByThread/Search/GetByMessageIDHeader/FindBySubjectAndParticipants) plus `scanMessage`/
    `collectMessages`. Die INSERT-Spaltenliste (`Create`) bewusst NICHT angefasst — die DB-Default
    `'{}'` deckt neue Nachrichten ab.
  - **AssignToMessage-Komposition**: `label.Service` bekommt neben dem eigenen Repository ein
    schlankes `MessageReader`-Interface (`GetByID(ctx, id, tenantID) (*models.EmailMessage, error)`)
    injiziert — *message.Service erfuellt es strukturell, exakt das Muster, das `send.Service` schon
    fuer `MessageCreator` nutzt (`cmd/email/main.go`). Schreibt label_ids direkt (wie
    `rule.ApplyToMessage` es fuer Regeln tut, kein Umweg ueber das message-Package) und liest danach
    ueber `messageService.GetByID` die frische Nachricht fuer die Response — keine dritte Kopie der
    ~20-Spalten-Liste noetig.
  - **Sicherheit — Assign validiert Label-Eigentuemerschaft vorab**: `LabelIDsBelongToTenant` prueft
    ALLE mitgeschickten IDs gegen `email_labels.tenant_id`, bevor irgendetwas geschrieben wird (Test
    `TestAssignToMessage_RejectsForeignLabelID` beweist: 0 Writes bei Ablehnung). Ohne diese Pruefung
    koennte `label_ids` eine fremde oder nichtexistente ID tragen — die Spalte hat keinen FK, der das
    verhindern wuerde.
  - **NACHGEZOGEN in `internal/email/rule/`** (der `lean:`-Marker aus `fe-email-rules` mit Trigger
    "sobald `email_labels` existiert" ist jetzt eingeloest): `LabelBelongsToTenant` im Rule-Repository
    ergaenzt, `applyInput` prueft jetzt Label-Targets genauso wie Move-Targets (`switch r.ActionType`
    statt `if move`). Vier Bestandstests mussten dafuer `repo.labels[label] = tenant` vorregistrieren
    (sonst waeren sie durch die neue Validierung gefallen), ein neuer Test
    `TestCreate_LabelTargetMustBeOwnLabel` spiegelt `TestCreate_MoveTargetMustBeOwnFolder`. Neuer
    DB-Test `TestRepository_LabelBelongsToTenant` in `rule/postgres_repository_test.go`.
  - Gateway: `GET/POST /api/v1/email/labels`, `PATCH/DELETE /api/v1/email/labels/{id}`,
    `POST /api/v1/email/messages/{id}/labels`. Alle ueber `email:read/write/delete` (kein neuer
    Guard, kein Seed). Wire-Shapes gegen `email-client.ts:308-333` verifiziert: List gewrappt
    `{labels:[]}`, `assign()` erwartet `{message: EmailMessageInfo}` exakt (nicht raten — Full-Replace
    des Label-Sets, kein Add/Remove-Paar, `label_ids` im Body). `update()`/`delete()` sind FE-seitig
    als `Record<string, never>` getypt (liest nichts aus der Antwort); Backend liefert trotzdem
    `{label:...}` bzw. `{success:true}` — konsistent zu `UpdateEmailRuleResponse`/
    `DeleteEmailRuleResponse` aus der Vorgaenger-Unit, harmlos fuer den FE-Client.
  - `openapi.yaml`: 4 neue Pfade + 6 neue Schemas + `label_ids` an `EmailMessageInfo`,
    `TestOpenAPIRouteDrift` gruen (764 Routen / 766 Pfade), `swagger-cli validate` gruen.
- gate: `go build -p 2 ./...` (voller Baum, kein `-p 2`-Timeout diesmal) gruen | vet gruen |
  `golangci-lint` `internal/email/... internal/gateway/... cmd/email/...` **0 issues** | Migration
  000261 up lokal gruen (Kopf `261`) | test ok mit `DATABASE_URL` gesetzt (Rolle `kmuhub_app`):
  `internal/email/... internal/gateway/... internal/server/...` alle `ok`, **0 Skips** ueber
  `internal/email/label`+`rule`+`message` verifiziert (`-v | grep -c SKIP` = 0, 46 PASS).
  `TestOpenAPIRouteDrift` gruen. gofmt: die beiden neuen Testdateien im `label`-Package hatten eine
  echte (nicht CRLF-bedingte) Map-Alignment-Abweichung — mit `gofmt -w` behoben, danach `gofmt -l`
  leer fuer das ganze Package.
  rls-smoke auf `email_labels` (zwei frisch angelegte temporaere Tenants, danach geloescht): eigener
  Tenant 1, fremder Tenant 0 — keine 0/0-Nullmessung. **Fund beim ersten Versuch**: die
  GATE-COMMANDS.md-Fixwerte `...0001`/`...00ff` existieren NICHT in `tenants` (im Gegensatz zu
  `contacts` hat `email_labels.tenant_id` einen FK auf `tenants`), ein Batch-INSERT mit beiden IDs
  schlug komplett fehl und beide Zaehlungen waren 0 — mit echten Tenant-Zeilen wiederholt.
- tests: 11 Service-Unit-Tests (Create/Update-Validierung, Duplicate-Name pro Tenant,
  Assign-Validierung inkl. "0 Writes vor bestandener Validierung"), 6 DB-Tests (`postgres_repository_
  test.go`, eigene Tenants+Mailbox-Fixture wie bei `rule`), 10 Gateway-Guard-Faelle
  (`route_email_labels_test.go`), plus im `rule`-Package: 1 neuer Unit-Test + 1 neuer DB-Test fuer die
  nachgezogene Label-Target-Validierung.
- offen:
  - **Fund im Bestand, ausserhalb dieser Unit (nicht behoben, dokumentiert)**: das Test-Muster
    `defer pool.Close()` gefolgt von `t.Cleanup(func(){ CleanupRow... })` laesst die Cleanup-Zeile
    gegen einen bereits geschlossenen Pool laufen — Go fuehrt Funktions-`defer`s beim Return der
    Testfunktion aus, `t.Cleanup`-Callbacks erst danach. `CleanupRow` loggt den Fehler nur (`t.Logf`),
    faellt der Test nicht. Ergebnis: liegen gebliebene Zeilen nach jedem gruenen Testlauf — lokal
    beobachtet in `email_labels` (34 Zeilen) UND vorbestehend in `email_rules` (7 Zeilen, nicht aus
    diesem Lauf). Manuell bereinigt (`DELETE FROM email_labels`), `email_rules`-Altlast nicht
    angefasst (ausserhalb Scope). Das Muster stammt aus `internal/email/rule`'s Tests und wurde hier
    fuer `label` identisch uebernommen — betrifft vermutlich jedes Package, das diesen Test-Stil
    kopiert hat. Kandidat fuer eine eigene kleine Fix-Unit (`t.Cleanup` fuer den Pool-Close statt
    `defer`, oder Cleanup-Reihenfolge umkehren).
  - `label_id`-Filter in `ListMessagesParams` (FE-Typ, `email-types.ts:162`) bleibt unverdrahtet —
    weder Proto (`ListMessagesRequest`) noch Gateway-Handler kennen ihn. Nicht Teil von `done_when`
    dieser Unit (nur "CRUD + Zuordnung"); Filterung nach Label ist ein sinnvoller Kandidat fuer eine
    eigene kleine Folge-Unit, sobald das FE tatsaechlich danach filtert.
  - `email_labels`-Existenzpruefung fuer Rule-Label-Targets ist jetzt aktiv (siehe oben) — bestehende
    Regeln mit `action_type='label'` und einer (aus der Zeit vor `email_labels`) nie validierten
    Ziel-ID werden dadurch bei einem `UpdateEmailRule`-Aufruf erstmals abgelehnt, falls die ID kein
    echtes Label ist. Reines Anwenden (`ApplyEmailRules`) ist NICHT betroffen (validiert nicht erneut
    beim Apply, nur bei Create/Update) — kein Verhaltensbruch fuer bestehende Regeln im Ruhezustand.

## Iteration 38 — fe-notifications-snooze — done — 2026-08-02 (Nachtlauf 3)
- commit: `1f108b00`
- verify vorgaenger: `41b7c4e1` (fe-messages-bookmark) gegen die acht Fehlerklassen geprueft — **kein
  Fund**. Gateway-Handler gehen ueber `ch.getChatClient()` (kein Fehlerklasse-1-Fund). Migration
  000262 `tenant_id NOT NULL` + `CALL enable_tenant_rls('message_bookmarks')`. Proto zwei neue RPCs
  (`ToggleBookmark`/`ListBookmarks`), `.pb.go`/`_grpc.pb.go` im selben Commit regeneriert. Wire-Shape
  `{bookmarked: boolean}`/`{messages: MessageInfo[]}` exakt gegen `useBookmarks.ts` verifiziert (Zeile
  fuer Zeile gelesen, keine Annahme). Guards nutzen ausschliesslich bestehende `messages:read/write`
  (kein neuer Key, kein Seed noetig). `openapi.yaml` traegt beide Pfade.
- gebaut: **Notification-Snooze** — `POST /api/v1/notifications/{id}/snooze`.
  - FE-Vertrag verifiziert in `notification-client.ts:145-158` (`snoozeApi.snooze`, echt aufgerufen aus
    `NotificationCenter.tsx:199-202,621-633`, kein reiner Mock-Pfad): flacher Response-Shape
    `{id, snoozed_until}`. **Bestandsfund dabei** (nicht Teil dieser Unit, dokumentiert): Pin/Dismiss
    liefern serverseitig `{notification}` (Proto-Wrap), obwohl sowohl der MSW-Mock
    (`mocks/handlers/notifications.ts:481`) als auch der FE-Typ `NotificationActionResponse` denselben
    flachen Shape wie Snooze erwarten. Bleibt bisher unbemerkt, weil `useToggleNotificationPin`/
    `useDismissNotification` die Mutation-Response nie lesen (nur `invalidateQueries`). Nicht repariert
    (ausserhalb Scope dieser Unit) — Kandidat fuer eine eigene kleine Fix-Unit.
  - Migration 000263: `notifications.snoozed_until TIMESTAMPTZ NULL` + Partial-Index
    `idx_notifications_snoozed` (Muster wie 000229 fuer is_pinned/is_dismissed). Keine neue RLS-Policy
    noetig (Tabelle traegt sie bereits seit 000122).
  - `notification.Service.SnoozeNotification` mirrort das Muster von `PinNotification`/
    `DismissNotification` (GetByID, Ownership-Check, dann Repo-Write) plus Zukunfts-Validierung
    (`ErrInvalidSnoozeTime`, woertlich das inbox-Vorbild `message.Service.Snooze`). Repo-`Snooze`
    setzt `snoozed_until` UND `is_read = true` UND `read_at = COALESCE(read_at, NOW())` — bewusst OHNE
    Un-Snooze-Worker (anders als `inbox.StartSnoozeWorker`): `List`/`GetUnreadCount` filtern per
    `(snoozed_until IS NULL OR snoozed_until <= NOW())`, nach Ablauf reicht das reine Filtern, damit
    der Eintrag wieder auftaucht — bleibt aber "gelesen" (kein Badge-Reset), exakt das MSW-Mock-
    Verhalten (`notif.is_read = true` in `mocks/handlers/notifications.ts:492`, keine spaetere
    Rueckstellung). Keine neue Goroutine noetig.
  - Neuer RPC `SnoozeNotification` in `notification.proto`, `.pb.go`/`_grpc.pb.go` im selben Commit
    regeneriert. Response ist bewusst NICHT `{notification: NotificationInfo}` (anders als Pin/
    Dismiss/MarkRead), sondern die schlanke `SnoozeNotificationResponse{id, snoozed_until}` — matcht
    den FE-Vertrag direkt, protojson mit `UseProtoNames: true` liefert automatisch snake_case plus
    RFC3339-String fuer den Well-Known-Timestamp, kein Hand-Mapping noetig.
  - **422 fuer Vergangenheit**: `grpcStatusToHTTP` kennt keinen 422-Fall (nur NotFound/AlreadyExists/
    Unauthenticated/PermissionDenied/InvalidArgument->400/FailedPrecondition->409/...). Statt die
    gemeinsame Mapping-Funktion fuer einen Einzelfall zu erweitern (Risiko fuer alle anderen RPCs),
    validiert der Gateway-Handler `until` VOR dem RPC-Call direkt per
    `response.Error(w, http.StatusUnprocessableEntity, ...)` — Muster aus
    `internal/middleware/idempotency.go`/`internal/server/file_upload.go` uebernommen (einzige
    bestehende 422-Praezedenzfaelle im Repo). Service validiert zusaetzlich (`ErrInvalidSnoozeTime`
    -> `InvalidArgument`) als Defense-in-Depth; bei intaktem Gateway nie erreichbar, schuetzt aber
    jeden kuenftigen zweiten Aufrufer des RPCs.
  - Gateway: `POST /api/v1/notifications/{id}/snooze` ueber `RequirePermission("notifications","write")`
    — bestehender Key (identisch zu Pin/Dismiss), kein neuer Guard, kein Seed.
  - `openapi.yaml`: neuer Pfad + Schema `SnoozeNotificationResponse` mit Beschreibung, warum es NICHT
    das gewrappte Notification-Schema ist. 422 dokumentiert wie beim bestehenden
    `incoming-invoices/{id}`-Precedent (blosse `description`, kein Schema).
- gate: `go build -p 2 ./...` (voller Baum) gruen | `go vet`
  `internal/notification/... internal/gateway/... internal/server/...` gruen | `golangci-lint`
  dieselben Pfade **0 issues** | gofmt sauber fuer alle selbst geaenderten Bloecke (CRLF-bereinigt
  geprueft; zwei Alt-Funde in `internal/models/notification.go` (`NotificationPreference`/
  `QuietHours`) und `notification_grpc.go` (`toQuietHoursInfo`) sind vorbestehende Misalignments in
  Code, den ich nicht angefasst habe — nicht repariert, ausserhalb Scope) | Migration 000263
  up/down/up lokal gruen (Kopf `263`) | swagger-cli validate: `openapi.yaml is valid`. Test ok mit
  `DATABASE_URL` gesetzt (Rolle `kmuhub_app`): `internal/notification/...` + `internal/server/...` +
  `internal/gateway/...` alle `ok`, **0 Skips** (per `-v | grep -c SKIP` in den beiden geaenderten
  Packages verifiziert). `TestOpenAPIRouteDrift`: PASS, 767 registrierte gegen 769 dokumentierte
  Pfade (+1 ggue. Iteration 37, konsistent mit der einen neuen Route).
  **RLS-Smoke** (manuell, `docker exec -i ... psql`, Transaktion mit `ROLLBACK`, keine Datenspuren,
  echte `tenants`/`users`/`notifications`-Zeilen statt der GATE-COMMANDS.md-Fixwerte): eigener Tenant
  1, fremder Tenant 0 — keine Nullmessung. Die Policy selbst ist unveraendert (nur Spalte ergaenzt),
  der Smoke bestaetigt, dass die neue WHERE-Klausel in `List`/`GetUnreadCount` die RLS-Policy nicht
  unterlaeuft.
  **Fund + selbst behoben**: das aus Iteration 36 dokumentierte `defer pool.Close()`-vor-`t.Cleanup`-
  Muster (Pool schliesst vor den Cleanup-Callbacks, `CleanupRow` scheitert still) wurde in der neuen
  `postgres_repository_test.go` NICHT repliziert — `pool.Close()` haengt stattdessen selbst an
  `t.Cleanup` (zuerst registriert, laeuft dank LIFO zuletzt). Verifiziert: erster Testlauf mit dem
  Bug-Muster hinterliess 8 Zeilen in `notifications`/6 in `users`; nach dem Fix und manueller
  Bereinigung liess ein erneuter Lauf **0** Zeilen zurueck (`SELECT count(*) ... WHERE
  title='Test notification'` vor/nach verglichen). Der vorbestehende Fund in `email_labels`/
  `message_bookmarks`/`email_rules` bleibt unangetastet (ausserhalb Scope, weiterhin Kandidat fuer
  eine eigene Fix-Unit).
- tests: 4 neue Service-Unit-Tests (Erfolg inkl. is_read=true, Vergangenheit->ErrInvalidSnoozeTime,
  NotFound, Unauthorized), 3 neue DB-Tests (`postgres_repository_test.go`: Snooze setzt Feld+is_read,
  unbekannte ID->NotFound, List+GetUnreadCount schliessen aktuell-gesnoozte Eintraege aus und zeigen
  abgelaufene wieder an) — eigene Tenants+User pro Testfall, kein Parallel-Kollisionsrisiko.
- offen:
  - **Fund im Bestand, ausserhalb dieser Unit (nicht behoben, dokumentiert)**: Pin/Dismiss liefern
    `{notification}` (gewrappt), FE-Client/Mock erwarten `{id, is_pinned, is_dismissed, actor_name}`
    (flach). Aktuell folgenlos (Mutation-Response wird nie gelesen), waere aber ein echter Bruch,
    sobald jemand die Response tatsaechlich konsumiert. Kandidat fuer eine eigene Fix-Unit, die beide
    RPCs auf den flachen Shape umstellt (analog zu Snooze).
  - Kein Un-Snooze-Worker (bewusst, siehe oben) — ein gesnoozter Eintrag bleibt nach Ablauf dauerhaft
    "gelesen", der Badge zeigt ihn nicht erneut an. Sollte das Produkt-Team spaeter ein "Reminder
    poppt wieder als ungelesen auf" verlangen, ist das eine bewusste Erweiterung (Worker + is_read-
    Reset), keine Nacharbeit an dieser Unit.
  - Zwei vorbestehende gofmt-Misalignments (`NotificationPreference`/`QuietHours` in
    `internal/models/notification.go`, `toQuietHoursInfo` in `notification_grpc.go`) unangetastet
    gelassen — ausserhalb Scope, keine Faelle die ich veraendert habe.
  - Naechste Unit laut Reihenfolge: `fe-documents-activity` (Block D, deps: [], model: sonnet).

## Iteration 39 — fe-documents-activity — done — 2026-08-02 (Nachtlauf 3)
- commit: `6a247186`
- verify vorgaenger: `1f108b00` (fe-notifications-snooze) gegen die acht Fehlerklassen geprueft —
  **kein Fund**. `HandleSnooze` geht ueber `n.getNotificationClient()` (kein Fehlerklasse-1-Fund),
  `SnoozeNotification` im gRPC-Server ist echte Implementierung (ruft `notifService.SnoozeNotification`
  auf, kein Stub). Migration 000263 ist nur `ALTER TABLE notifications ADD COLUMN` — keine neue Tabelle,
  Fehlerklasse 5 entfaellt. Kein neuer Guard (bestehender `notifications:write`-Key wiederverwendet,
  kein Seed noetig, kein Alt-Key verloren). `openapi.yaml` traegt den neuen Pfad + Schema.
- gebaut: **Datei-Aktivitaetsverlauf** — `GET /api/v1/documents/files/{id}/activity`.
  - `document_events` (GoBD-Archiv-Kontext) VERWORFEN als Wiederverwendungskandidat: es ist
    `gobd_document_events`, referenziert `gobd_documents(id)` — eine komplett andere Tabelle fuer den
    Belegarchiv-Kontext, nicht fuer die allgemeinen `document_files`. Deshalb neue Tabelle gebaut, wie
    im Scope als Alternative vorgesehen.
  - Migration 000264: `document_file_activity` — `tenant_id UUID NOT NULL` + FK auf `tenants`,
    `file_id` FK auf `document_files(id) ON DELETE CASCADE`, `action` per CHECK-Constraint auf die
    acht FE-Werte (`uploaded/renamed/moved/copied/downloaded/shared/version_created/reverted`),
    `actor_id` FK auf `users(id)`. `CALL enable_tenant_rls('document_file_activity')` (Standardform,
    keine NULL-Ausnahme noetig — anders als bei RBAC-Praesets ist hier jede Zeile einem echten Tenant
    zugeordnet). Append-only per BEFORE-UPDATE-OR-DELETE-Trigger, identisches Muster zu `audit_log`
    (Migration 000222): `RAISE EXCEPTION` blockt beide Operationen auf DB-Ebene, nicht nur in Go.
  - **Fund + selbst behoben** (Fehlerklasse-2-angrenzend, nicht im Verify-Vorspann sondern beim Bauen
    entdeckt): `CopyFile` und `RevertFileVersion` im gRPC-Server uebergaben seit jeher `uuid.Nil` als
    Akteur an den Service ("Use Nil as userID since gateway handles auth" — der Kommentar war falsch).
    `middleware.GetUserID(ctx)` ist im Dokument-Service tatsaechlich verfuegbar (`registry.go:112`
    verdrahtet `TenantOutboundUnaryInterceptor` global fuer JEDE Gateway-Verbindung, nicht nur fuer
    einzelne Services), wurde aber an diesen beiden Stellen nie gelesen. Neuer Helper
    `actorIDFromContext(ctx)` in `document_grpc.go`, jetzt auch fuer `UpdateFile`/`MoveFile` genutzt.
    Ohne den Fix haetten `copied_by`/`created_by` in den neuen Activity-Zeilen dauerhaft auf einen
    nicht-existenten `uuid.Nil`-User gezeigt (FK-Verletzung bei jedem Copy/Revert) — kein Rand-, sondern
    ein Kernpfad der neuen Unit, deshalb im selben Commit korrigiert statt als offener Punkt notiert.
  - `file.Service.Update`/`.Move` bekamen ein neues `actorID`-Parameter (Signaturaenderung, zwei
    Call-Sites in `document_grpc.go` + zwei Test-Call-Sites angepasst). `Update` loggt `renamed` nur
    wenn `Filename` gesetzt ist und `moved` nur wenn `FolderID` gesetzt ist — ein reines
    `IsFavorite`-Toggle erzeugt bewusst KEINEN Eintrag (Praeferenz, kein Audit-relevanter Vorgang;
    eigener Test `TestUpdate_FavoriteOnly_NoActivity` haelt das fest).
  - `GetDownloadURL` behielt seine bestehende Signatur unveraendert (wird auch vom WOPI-Adapter in
    `cmd/document/main.go` ueber `wopi.FileServiceInterface` aufgerufen, dort ohne Akteur — dieser Pfad
    ruft die Methode aber nachweislich nie real auf, verifiziert per Grep im ganzen `wopi`-Paket). Neue
    separate Methode `Service.LogDownload(ctx, fileID, tenantID, actorID)`, nur vom Gateway-Handler
    `GetFileDownloadURL` nach erfolgreichem Presign aufgerufen — kein Signatur-Bruch fuer den
    unbenutzten WOPI-Pfad.
  - Neuer RPC `ListFileActivity` in `document.proto`, `.pb.go`/`_grpc.pb.go` im selben Commit
    regeneriert (`make proto-document` manuell nachgebaut, `make` fehlt auf dieser Maschine —
    `protoc --go_out=... --go-grpc_out=...` direkt mit denselben Flags wie im Makefile-Target).
    Response `{activities: DocumentFileActivity[]}` per `response.ProtoListWrapped` (protojson,
    `[]` statt `null` bei leerer Liste, snake_case automatisch) — exakt der FE-Vertrag aus
    `ListActivityResponse` in `document-types.ts:268`.
  - Gateway: `GET /api/v1/documents/files/{id}/activity` ueber bestehenden `docRead`-Guard
    (`RequirePermissionAny({"documents","read"}, {"documents:file","read"})`) — kein neuer Key,
    kein Seed. Zwei neue Faelle in `route_capability_guard_test.go` (Katalog-Key allowed, edit-Key
    denied).
  - `openapi.yaml`: neuer Pfad + Schema `DocumentFileActivity` (protojson-Form, `created_at` als
    echter RFC3339-String wie bei `DocumentFileVersion`, nicht als ProtoTimestamp).
  - `.planning/backend-gaps.md` Zeile 215 als erledigt markiert (gleiches Konventions-Praefix `✅ ...
    CODE ERLEDIGT` wie beim Presign-Endpoint-Gap), Original-Beschreibung des Gaps als Kontext belassen.
- gate: `go build -p 2 ./...` (voller Baum) gruen | `go vet` fuer
  `internal/document/... internal/gateway/... internal/server/... internal/models/... cmd/document/...
  cmd/gateway/...` gruen | `golangci-lint run --config .golangci.yml` fuer dieselben Pfade (ausser cmd)
  **0 issues** | Migration 000264 down/up/up lokal gruen (Kopf `264`) | swagger-cli validate:
  `openapi.yaml is valid` | Test mit `DATABASE_URL` gesetzt (Rolle `kmuhub_app`):
  `internal/document/...` (39 Tests, davon 3 neu als echte DB-Tests) + `internal/gateway/...` +
  `internal/server/...` alle `ok`, **0 Skips**. `TestOpenAPIRouteDrift`: PASS, 768 registrierte gegen
  770 dokumentierte Pfade (+1 ggue. Iteration 38, konsistent mit der einen neuen Route).
  **RLS-Smoke** (`docker exec -i ... psql`, echte Zeilen aus den DB-Tests statt synthetischer Werte):
  eigener Tenant 2, fremder Tenant 0 — keine Nullmessung.
  **Append-only verifiziert** (nicht nur behauptet): `TestDocumentFileActivity_AppendOnly` versucht
  echtes UPDATE und DELETE gegen eine geseedete Zeile unter System-Kontext (BYPASSRLS-aequivalent) und
  erwartet beide Male einen Fehler vom DB-Trigger — beide schlagen wie erwartet fehl, die Zeile bleibt
  unveraendert (Count-Check danach).
- tests: 3 neue DB-Tests (`postgres_repository_test.go`: Create+List mit korrekter Sortierung
  newest-first und Actor-Name-JOIN, Tenant-Isolation liefert 0 unter fremder tenant_id, Append-only
  blockt UPDATE+DELETE), 3 neue Mock-basierte Service-Tests (`TestMove_Success`/`TestUpdate_Success`
  pruefen jetzt zusaetzlich den aufgezeichneten Activity-Eintrag, neuer
  `TestUpdate_FavoriteOnly_NoActivity`) — je eigener Tenant/User/Folder/File pro DB-Testfall, kein
  Parallel-Kollisionsrisiko.
- offen:
  - Test-Fixtures in `postgres_repository_test.go` raeumen bewusst NICHT auf (siehe Kommentar am
    `activityFixture`-Typ): sobald ein `document_files`-Testfixture Activity-Kinder traegt, wuerde
    `testutil.CleanupRow`s DELETE per `ON DELETE CASCADE` den Append-only-Trigger der Kinder ausloesen
    und scheitern (nur geloggt, kein Testfehler) — Zeilen bleiben in der lokalen/CI-ephemeren DB liegen,
    exakt der Tradeoff, den die bestehenden `audit_log`-Tests schon eingehen. Keine Produktionsrelevanz.
  - `downloaded` wird nur beim expliziten Gateway-Download-URL-Aufruf geloggt, nicht bei jedem
    OnlyOffice/WOPI-Zugriff (dort ist `GetDownloadURL` nachweislich unbenutzt, s.o.) — falls WOPI
    kuenftig echte Downloads darueber abwickelt, muesste diese Stelle nachgezogen werden.
  - Naechste Unit laut Reihenfolge: `fe-documents-links` (Block D, deps: [fe-documents-activity],
    model: sonnet) — jetzt entsperrt.

## Iteration 40 — fe-documents-links — done — 2026-08-02 (Nachtlauf 3)
- commit: `6a247186` verify vorgaenger, siehe unten
- verify vorgaenger: `6a247186` (fe-documents-activity) gegen die acht Fehlerklassen geprueft — **kein
  Fund**. `HandleListFileActivity` geht ueber `client.ListFileActivity(...)` (gRPC-Client, kein
  Fehlerklasse-1-Fund). Migration 000264 hat `tenant_id NOT NULL` + `CALL enable_tenant_rls(...)` +
  Append-only-Trigger, Repo-INSERT UND -SELECT beide tenant-gescoped (per Diff verifiziert). Kein
  neuer `RequirePermission`-Guard (bestehender `docRead` wiederverwendet), keine Fehlerklasse-4/8.
  `openapi.yaml` traegt den neuen Pfad im selben Commit. Der `actorIDFromContext`-Fix (uuid.Nil ->
  echter Akteur bei CopyFile/RevertFileVersion) ist ein echter, im selben Commit korrigierter Bugfix,
  kein Stub — verifiziert gegen `document_grpc.go`-Diff.
- gebaut: **`DELETE /api/v1/documents/links/{id}`** — Widerruf eines Dokument-Entity-Links per
  eigener Link-ID.
  - **Scope-Korrektur gegenueber der Backlog-Beschreibung:** Die Unit-Notes gehen von externen,
    passwortgeschuetzten "Freigabelinks" aus (Analogie zu `report_share_tokens`). Verifiziert gegen
    den echten FE-Code (`document-client.ts:479-503`, `useDocuments.ts:444-452`,
    `mocks/handlers/documents.ts:760`): `/api/v1/documents/links/{id}` ist tatsaechlich
    `documentLinkApi.unlink(linkId)` — Widerruf eines **CRM/PM-Entity-Links**
    (`DocumentEntityLink`, Datei <-> Kontakt/Deal/Task/Projekt), nicht ein unauthentifizierter
    externer Zugang. Die dialoggetriebene "Freigabelink mit Passwort/Ablauf"-UI
    (`modules/dokumente/ShareLinkDialog.tsx`) ist zu 100% clientseitiger Mock
    (`generateMockLink`, kein einziger API-Call) — ein separates, unwired FE-Feature, nicht das,
    was dieser Pfad bedient. Gebaut wurde die echte Luecke, nicht die vermutete.
  - **Echter Bug gefunden und im selben Commit behoben (Root-Cause, nicht nur die neue Route):**
    `models.DocumentEntityLink` hatte nie ein `TenantID`-Feld, und
    `PostgresRepository.CreateEntityLink` insertierte nie eine `tenant_id` — obwohl die Spalte seit
    Migration 000114 `NOT NULL` ist (RLS seit 000122). Jeder reale
    `POST /documents/files/{id}/links`-Aufruf haette seit der RLS-Retrofit-Welle mit einer
    NOT-NULL-Verletzung 500en muessen; das Feature war seit Sprint 4 tot, nur unsichtbar, weil im FE
    kein einziger Call-Site fuer `useLinkFile`/`useUnlinkFile` existiert (siehe naechster Punkt).
    Fix: `TenantID` aufs Model, `LinkToEntity` (Service) setzt `link.TenantID = tenantID` (Parameter
    war schon da, nur ungenutzt), `CreateEntityLink`-INSERT nimmt die Spalte mit.
  - **Tenant-Scoping nachgezogen, nicht nur fuer die neue Route:** `ListEntityLinks`/`DeleteEntityLink`
    (Repo+Service) haben jetzt `tenantID`-Parameter mit explizitem `WHERE tenant_id = $N` (Konvention
    dieser Datei, siehe `GetByID`) statt sich allein auf RLS zu verlassen. Betrifft auch die
    BESTEHENDEN Handler `UnlinkFileFromEntity`/`ListFileEntityLinks`/`LinkFileToEntity`
    (`document_grpc.go`): `UnlinkFileFromEntity` hatte bisher **gar keinen**
    `middleware.GetTenantID(ctx)`-Aufruf und lief nur ueber RLS — jetzt konsistent mit den
    Nachbar-Handlern. Kein Wire-Shape-Bruch: keiner der drei bestehenden Response-Contracts hat sich
    geaendert, nur die interne Repo-Signatur.
  - **FE-Realitaet, im Journal vermerkt statt geraten:** `useLinkFile`/`useUnlinkFile`
    (`api/hooks/useDocuments.ts:428,444`) haben **keinen einzigen Aufrufer** in einer Komponente —
    nur `useFileLinks` (Liste, read-only Anzeige in `FileDetailPanel.tsx:113`) wird tatsaechlich
    gerendert. Die neue Route bedient also aktuell keinen sichtbaren User-Flow, sondern schliesst die
    Backend-Luecke fuer einen bereits im API-Client committeten, aber UI-seitig noch nicht verdrahteten
    Pfad — dieselbe Kategorie wie die in `p2c-work-documents-crm-finance`/`-2` dokumentierten
    "FE-Aufrufer ist reiner Mock"-Faelle, nur umgekehrt (hier ist es der Erzeuger-Hook, nicht der
    Consumer, der fehlt).
  - **Kein GET-Pendant erfunden:** Die Backlog-Notes nennen "GET/DELETE". Ein `GET
    /documents/links/{id}` (Einzel-Link per ID) hat keinen FE-Aufrufer — Lesen laeuft ausschliesslich
    ueber das bestehende, unveraenderte `GET /documents/files/{id}/links` (Liste pro Datei). Kein
    Katalog-Key betroffen (Entity-Links sind laut Kommentar in `route_document.go:66` bewusst ohne
    Catalogue-Subject) — neue Route bekommt denselben coarse-only `RequirePermission("documents",
    "write")`-Guard wie ihr Geschwister-Endpoint, kein neuer Key, kein Seed.
  - Neuer RPC `DeleteEntityLink(DeleteEntityLinkRequest{link_id}) returns
    (DeleteEntityLinkResponse{})` in `document.proto`, `.pb.go`/`_grpc.pb.go` im selben Commit
    regeneriert (`protoc` direkt, gleiche Flags wie `make proto-document`, `make` fehlt auf dieser
    Maschine).
  - `EntityName` bleibt weiterhin immer leer (kein `entity_name`-Spalte in `document_entity_links`
    seit Migration 000043, nie nachgezogen) — vorbestehende, separate Luecke, NICHT in dieser Unit
    behoben (kein FE-Aufrufer fuer Create/Update betroffen, keine Migration in diesem Commit noetig;
    eine echte Loesung braucht Cross-Entity-Namensaufloesung ueber 5 Entity-Typen, das ist ein eigener
    Zuschnitt).
- gate: `go build -p 2 ./...` (voller Baum) gruen | `go vet` fuer
  `internal/document/... internal/gateway/... internal/server/...` gruen | `golangci-lint run
  --config .golangci.yml` fuer dieselben Pfade **0 issues** | swagger-cli validate: `openapi.yaml is
  valid` | migration: keine (Spalte existierte bereits seit 000114) | Test mit `DATABASE_URL` gesetzt
  (Rolle `kmuhub_app`): `internal/document/...` + `internal/gateway/...` + `internal/server/...` alle
  `ok`, **0 Skips** (per grep verifiziert). `TestOpenAPIRouteDrift`: PASS. Zwei neue Faelle in
  `route_capability_guard_test.go` (`entity link delete by ID`, coarse-only) PASS.
  **RLS-Smoke** (`docker exec -i ... psql`, echte Fixture-Zeilen unter `kmuhub_app`): eigener Tenant 1,
  fremder Tenant 0 — Fixture danach wieder geloescht (keine RLS-Erzwingung noetig, normale Zeilen ohne
  Append-only-Trigger).
- tests: 3 neue echte DB-Tests (`postgres_repository_entity_link_test.go`: Create+List mit
  Tenant-Filter, Tenant-Isolation liefert 0 unter fremder tenant_id, Delete verweigert unter fremdem
  Tenant mit `ErrFileNotFound` UND laesst die Zeile unangetastet, loescht korrekt unter eigenem
  Tenant), 4 neue Mock-Service-Tests (`TestLinkToEntity_SetsTenantID`,
  `TestUnlinkFromEntity_Success`, `TestUnlinkFromEntity_WrongTenant_NotFound`,
  `TestListEntityLinks_TenantIsolation`) — je eigener Tenant/User/Folder/File pro DB-Testfall.
- offen:
  - `useLinkFile`/`useUnlinkFile` sind FE-seitig ungerendert (s.o.) — sobald ein UI-Flow dafuer gebaut
    wird, ist das Backend jetzt bereit (inkl. des Tenant-Bugfixes), keine weitere Backend-Arbeit nötig.
  - `entity_name` bleibt leer (vorbestehende, separate Luecke, s.o.) — eigener Zuschnitt falls die
    Anzeige des verlinkten Entity-Namens im FE gebraucht wird.
  - Naechste Unit laut Reihenfolge: `fe-contacts-files` (Block D, deps: [], model: sonnet).

## Iteration 41 — fe-contacts-files — done — 2026-08-02 (Nachtlauf 3)
- commit: `483ecea7`
- verify vorgaenger: `0685eb68` (fe-documents-links) gegen die acht Fehlerklassen geprueft — **kein
  Fund**. `HandleDeleteEntityLink` geht ueber `client.DeleteEntityLink(...)` (gRPC-Client, kein
  Fehlerklasse-1-Fund). `.proto`+`.pb.go`+`_grpc.pb.go` im selben Commit regeneriert (Fehlerklasse 3
  entfaellt). Kein neuer `RequirePermission`-Guard (bestehender `documents:write` wiederverwendet,
  Fehlerklasse 4/8 entfaellt). `CreateEntityLink`-Tenant-Fix + `ListEntityLinks`/`DeleteEntityLink`
  mit explizitem `tenant_id`-Filter — echte Behebung, kein neuer Tenant-Luecken-Fund. `openapi.yaml`
  traegt den neuen Pfad im selben Commit, `TestOpenAPIRouteDrift` lief.
- gebaut: **`GET/POST /api/v1/contacts/{id}/files`** — Dateianhaenge am CRM-Kontakt.
  - Wie in den Notes gefordert **kein zweiter Dateispeicher**: beide Routen komponieren ausschliesslich
    bestehende Document-Service-RPCs im Gateway (`ListFolders`, `CreateFolder`, `RegisterUploadedFile`,
    `LinkFileToEntity`, plus neu `ListFilesByEntity`) statt eine `crm_contact_files`-Tabelle
    anzulegen — keine neue Migration in diesem Commit.
  - **Upload-Flow ist presign-then-register**, analog zum bestehenden Task-Files-Muster
    (`useTaskFiles.ts`): Client holt sich zuerst eine presigned URL ueber das bereits bestehende
    `POST /api/v1/files/presign-upload` (Scope `kontakte` war in `allowedPresignScopes`
    [`internal/document/file/presign.go:32`] schon zugelassen — nichts daran aendern muessen), laedt
    direkt zu MinIO hoch, ruft danach `POST /api/v1/contacts/{id}/files` mit
    `{filename, mime_type, storage_key, file_size}` auf. Der MSW-Mock
    (`mocks/handlers/crm.ts:660`) sendet **kein** `file_size` — das reale Backend braucht es aber
    zwingend (`document_files.file_size BIGINT NOT NULL CHECK (file_size > 0)`,
    `RegisterUploadedFile` validiert `ErrFileSizeZero`). Vermerkt statt stillschweigend nachgeben: der
    Mock ist duenner als der reale Contract, wie schon in mehreren `p2c-*`-Funden — die zukuenftige
    FE-Anbindung muss `file_size` mitschicken.
  - **`document_files.folder_id` ist `NOT NULL`** (keine Ordner-lose Ablage moeglich) und es gibt
    keinen Ordner-Picker fuer Kontakt-Anhaenge. Geloest mit einem lazily erzeugten, gemeinsamen
    System-Ordner **"CRM-Kontaktanhänge"** (`space_type=team`, `space_id=<tenant_id>` — kein
    Team-Konzept fuer den ganzen Mandanten vorhanden, `space_id` traegt keine FK-Constraint, also
    unproblematisch wiederverwendet). Race-sicher ohne neue DB-Konstrukte: `CreateFolder`
    (`folder.Service.Create`) lehnt einen doppelten Namen im selben Parent selbst mit
    `ErrFolderNameConflict` -> `codes.AlreadyExists` ab; der Verlierer eines Concurrent-First-Calls
    holt sich den Ordner der Gewinner-Anfrage per erneutem `ListFolders` statt zu scheitern.
    `lean:`-markiert in `route_crm_contact_files.go:25` mit Upgrade-Trigger ("wenn Kontakte je einen
    eigenen Anhang-Ordner/Browser brauchen").
  - **Vorbestehende, bisher tote Luecke gefunden und geschlossen:**
    `file.Repository.ListFilesByEntity(ctx, entityType, entityID)` — die noetige Rueckwaerts-Abfrage
    "welche Dateien haengen an Entity X" — existierte bereits vollstaendig (Postgres-Impl + Mock in
    `service_test.go`), war aber **von KEINER Service-Methode, KEINEM RPC und KEINEM Handler jemals
    aufgerufen** worden (verifiziert per grep, nur Interface+Impl+Mock referenzierten den Namen) UND
    filterte **nicht nach `tenant_id`** (reine RLS-Abhaengigkeit, im Widerspruch zur Konvention
    dieser Datei seit dem Fix in `fe-documents-links`). Behoben statt daneben eine zweite Abfrage zu
    bauen: `tenant_id`-Parameter durch Repository-Interface, Postgres-Query
    (`AND del.tenant_id = $3 AND f.tenant_id = $3`) und neue `Service.ListByEntity`
    (validiert `AllowedEntityTypes`, wie `LinkToEntity`) gezogen. Proto bekam den fehlenden RPC
    `ListFilesByEntity(ListFilesByEntityRequest{entity_type, entity_id}) returns
    (ListFilesByEntityResponse{repeated DocumentFile files})`, `document_grpc.go` bekam den
    `DocumentGRPCServer`-Handler (Muster von `ListFileEntityLinks` abgeschaut, `toProtoFile`
    wiederverwendet).
  - **Kontakt-Zugehoerigkeit serverseitig geprueft, nicht nur RLS** (Notes-Pflicht): beide Handler
    rufen zuerst `crmClient.GetContact(ctx, {Id: contactID})` — `CRMGRPCServer.GetContact`
    (`crm_grpc.go:458`) scoped intern per `contactService.GetByID(ctx, id, tenantID)`, ein fremder
    Tenant bekommt `NotFound` -> 404, bevor ueberhaupt ein Document-RPC angefasst wird. Der
    Verbindungsfehler-Pfad (`getCRMClient()`/`getDocumentClient()` scheitert, leere Registry) ist
    bewusst vom RPC-Fehler-Pfad getrennt (`verifyContactOwnership` gibt `(connErr, rpcErr)` zurueck) —
    sonst waere ein Registry-Ausfall via `respondGRPCError` als 500 statt 503 gelandet, im Widerspruch
    zur Guard-Test-Konvention "allowed = 503 bei leerer Registry".
  - **Wire-Shape**: `ContactFile{id, contact_id, filename, mime_type, storage_key, created_at}` exakt
    nach `useContacts.ts:47` (camelCase-Namen im FE-Kommentar sind snake_case im echten Typ), GET
    liefert `{files: []}` (nie `null`), POST `{file: {...}}` gewrappt mit 201 — passt 1:1 zum
    bestehenden `mocks/handlers/crm.ts:655-673`.
  - Kein neuer `RequirePermission`-Guard: `GET` haengt an `contactRead`, `POST` an `contactEdit`
    (beide bereits additiv in `route_crm.go`, wiederverwendet wie bei den Tags-Routen daneben) — kein
    Seed noetig.
- gate: `go build -p 2 ./internal/document/... ./internal/gateway/... ./internal/server/...
  ./cmd/document/... ./cmd/gateway/... ./cmd/crm/...` gruen | `go vet` fuer dieselben Pfade gruen |
  `golangci-lint run --config .golangci.yml` fuer `internal/document/...`, `internal/gateway/...`,
  `internal/server/...` **0 issues** | swagger-cli validate: `openapi.yaml is valid` | migration:
  keine (Spalten existieren seit 000114/000122) | Test mit `DATABASE_URL` gesetzt (Rolle
  `kmuhub_app`): `internal/document/...` (inkl. `file`, `folder`, `search`, `share`, `tag`,
  `virtual`), `internal/gateway/...`, `internal/server/...` alle `ok`, **0 Skips** (per grep
  verifiziert, 54 PASS allein in `internal/document/file`). `TestOpenAPIRouteDrift`: PASS (772
  dokumentierte Pfade). Fuenf neue Faelle in `route_capability_guard_test.go`
  (`contact_files_list_*`, `contact_files_create_*`) einzeln per `-run` verifiziert: alle PASS.
  **RLS-Smoke** (`docker exec -i ... psql`, echte Fixture-Rows unter `SET ROLE kmuhub_app`, in
  Transaktion mit `ROLLBACK` danach): eigener Tenant 1 Zeile, fremder Tenant 0 Zeilen, exakt die
  Query-Form von `ListFilesByEntity`.
- tests: 5 neue echte DB/Service-Tests fuer `ListByEntity`
  (`TestPostgresRepository_ListFilesByEntity`, `TestPostgresRepository_ListFilesByEntity_TenantIsolation`
  gegen echte DB; `TestListByEntity_Success`, `TestListByEntity_TenantIsolation`,
  `TestListByEntity_InvalidType` gegen den Mock-Service), 5 neue Guard-Testfaelle fuer die beiden
  neuen Routen.
- offen:
  - `RegisterUploadedFile`/`file_size`-Pflichtfeld ist im MSW-Mock nicht abgebildet (s.o.) — beim
    Bau des echten FE-Upload-Hooks (`useCreateContactFile`/Ähnliches, existiert bisher nicht)
    `file.size` mitschicken, sonst 400.
  - Der neue System-Ordner "CRM-Kontaktanhänge" taucht nach dem ersten Upload als normaler,
    nicht-System-Ordner (`is_system=false`, `CreateFolder`-RPC setzt das Feld nicht) im allgemeinen
    Dokumente-Modul auf (space_type=team) — potenziell verwirrend fuer Nutzer, die dort durch alle
    Team-Ordner browsen. Kein Backend-Problem, aber FE-seitig evtl. filtern/ausblenden wollen.
  - Naechste Unit laut Reihenfolge: `fe-caldav-test` (Block D, deps: [], model: sonnet).

## Iteration 42 — fe-caldav-test — done — 2026-08-02 (Nachtlauf 3)
- commit: 3b481316
- gebaut:
  - `POST /api/v1/caldav/test` (`route_caldav.go`, `handleTestConnection`/`probeSelf`) hinter
    demselben `authMiddleware` wie die Nachbarrouten `/caldav/status|enable|disable` — kein neuer
    Guard, kein Seed noetig.
  - SCOPE-KLARSTELLUNG (der Backlog-Text "Verbindungstest fuer eine CalDAV-Konfiguration" war
    missverstaendlich): dieses Modul ist kein CalDAV-*Client*, der gegen einen externen Server
    konfiguriert wird — die App IST der CalDAV/CardDAV-Server (Apple Calendar/Thunderbird/etc.
    verbinden sich gegen `/caldav/`, `/carddav/` per App-Passwort). Es gibt keine externe
    "Konfiguration" zu testen. Das schon vorhandene, bisher unverdrahtete FE
    (`caldav-client.ts:testCalDAVConnection`, `CalDAVSettingsTab.tsx` Test-Button, KEIN MSW-Mock
    dafuer) erwartet stattdessen einen echten End-to-End-Beweis, dass der eigene Server-Pfad
    funktioniert.
  - Echter Test statt Lexware-Anti-Pattern (`biz/lexware/service.go:147`, "Feld nicht leer =
    verbunden"): Handler erzeugt ein frisches App-Passwort (`pwService.Create`), macht zwei
    echte, auf 5s timeout-bounded `OPTIONS`-Requests gegen `/caldav/` und `/carddav/` — denselben
    Pfad, den ein echter CalDAV-Client nutzen wuerde — und revoked das Passwort in einem `defer`
    unabhaengig vom Ausgang. `net/http/httptest`-Tests bestaetigen: das Passwort ist nach dem
    Request wirklich revoked (aktive Anzahl 0), nicht nur "sollte".
  - SSRF-Vermeidung als bewusste Designentscheidung (nicht im Backlog gefordert, aber notwendig):
    das Ziel der Probe ist IMMER `http://127.0.0.1<GatewayHTTPPort>` — aus der eigenen Config
    (`cfg.GatewayHTTPPort`, per neuem `selfBaseURL`-Parameter durch `setupCalDAV`/`NewCalDAVRoutes`
    durchgereicht), NIE aus `r.Host`/`X-Forwarded-Host`. Ein authentifizierter Angreifer haette
    sonst jeden beliebigen internen Host per Header ansprechen koennen — jeder eingeloggte User
    darf diesen Endpoint aufrufen, ohne dass CalDAV fuer ihn aktiviert sein muss.
  - Fehlerklassen unterscheidbar: `401` -> "authentication failed", Netzwerkfehler (Verify per
    `errors.Is`/generischer `Do`-Fehler, kein DNS/Connect) -> "network unreachable", Timeout ->
    "timeout", alles andere -> "unexpected status N". Response `{success, message, caldav_reachable,
    carddav_reachable}` — exakt das FE-Interface `CalDAVTestResult` aus `caldav-client.ts:110`.
  - Keine Zugangsdaten in Logs verifiziert (grep auf alle neuen `slog.*`-Aufrufe: nur `user_id`,
    `password_id`, `error` — nie Klartext-Passwort).
  - `openapi.yaml`: neuer Pfad + `CaldavTestResult`-Schema im selben Commit. Fallstrick beim
    Schreiben: eine `description:` als *plain scalar* mit `"CalDAV: ..."` drin bricht den
    YAML-Parser an der `: ` (Mapping-Trenner) — musste die ganze Description doppelt quoten.
    `npx swagger-cli validate` haette das lokal sofort gezeigt (jetzt gruen), stand aber bisher
    nicht in `GATE-COMMANDS.md` als Pflichtschritt fuer *neue* Schemas — nur als Referenz in einem
    frueheren Journal-Eintrag.
- gate: build (`./internal/caldav/... ./internal/gateway/... ./cmd/gateway/...`) ok | vet ok |
  golangci-lint **0 issues** | `npx swagger-cli validate backend/api/openapi.yaml`: valid |
  migration: keine (keine neue Tabelle/Policy angefasst) | RLS-Smoke: n.a. (kein Tabellen-/
  Policy-Zugriff neu, nur bestehender `AppSpecificPassword`-Pfad wiederverwendet) | Test mit
  `DATABASE_URL` gesetzt (Rolle `kmuhub_app`): `internal/caldav/...` UND `internal/gateway/...`
  beide `ok`, **0 Skips**. 3 neue Tests (`TestHandleTestConnection_Success/_AuthFailure/
  _NetworkUnreachable`) gegen einen echten `httptest.Server` (echter TCP-Loopback, echte
  `basicAuthMiddleware`, In-Memory-Fake fuer `CalDAVPasswordService` statt DB) — decken Erfolg,
  Auth-Fehler und "Port tot" (via kurzzeitig reserviertem, sofort wieder freigegebenem Port) ab.
  `TestOpenAPIRouteDrift`: PASS (771 registrierte gegen 773 dokumentierte Pfade).
- verify vorgaenger: sauber (`483ecea7`, contact file attachments) — gRPC-Client durchgaengig
  (`getDocumentClient`/`getCRMClient`), kein Stub, Kontakt-Eigentum vor Document-Zugriff echt
  geprueft (`verifyContactOwnership`), Wire-Shape exakt gegen `useContacts.ts` verifiziert,
  `openapi.yaml` im selben Commit, kein neuer Guard (bestehende `contactRead`/`contactEdit`
  wiederverwendet, kein Seed noetig).
- offen:
  - `GATEWAY_HTTP_PORT` muss in Produktion tatsaechlich der Port sein, auf dem der Gateway-Prozess
    selbst lauscht (Standard `:8080`, `ListenAndServe`-Addr identisch) — sollte das je auseinander
    laufen (z.B. durch einen Reverse-Proxy-Port-Remap direkt am Gateway-Container statt an Caddy),
    liefe der Loopback-Test ins Leere. Aktuell keine Diskrepanz gefunden, nur als Annahme notiert.
  - `swagger-cli validate` steht noch nicht in `GATE-COMMANDS.md` als Pflichtschritt fuer neue
    Schemas/Pfade — waere sinnvoll, da der YAML-Parser-Fehler dieser Iteration sonst erst in CI
    (`TestOpenAPIRouteDrift` haette es vermutlich NICHT gefangen, das ist ein reiner
    Struktur-Test, kein YAML-Validator) aufgefallen waere.
  - Kein Full-Stack-Bringup gefahren (Loop-Policy) — der Loopback-Request gegen den echten,
    laufenden Gateway-Prozess (nicht den Test-Fake) wurde nicht manuell verifiziert; die drei neuen
    Tests decken die Handler-Logik gegen einen `httptest.Server` ab, aber nicht den realen
    Produktions-Bringup mit echtem `caldav.Handler`/`carddav.Handler` (go-webdav) und echtem
    `AppPasswordService` gegen Postgres. Sollte Luke morgens einmal per `curl -u <token>` o.ae.
    gegen einen laufenden Dev-Gateway pruefen.

## Iteration 43 — g-lexware-wiring — done — 2026-08-02 (Nachtlauf 3)
- commit: f2f362f9
- gebaut:
  - Alle fuenf oeffentlichen Methoden von `biz/lexware/service.go` rufen jetzt die seit Monaten
    danebenliegenden Implementierungen auf (Vorbild `bexio/service.go`): Instanzen im Struct, im
    Konstruktor gebaut, in den Methoden aufgerufen. `SyncContacts` -> `ContactSyncer`,
    `PushInvoice`/`PushQuote` -> die Pusher, `HandleWebhookEvent` -> `WebhookHandler`.
    Der doppelte SyncLog-Code in `SyncContacts` ist raus — der Syncer besitzt den Log-Lebenszyklus
    (running -> completed/partial/failed) und die Last-Sync-Zeit; beides zweimal zu schreiben haette
    zwei Log-Zeilen pro Lauf erzeugt und den Zeitstempel auch bei Fehlschlag vorgerueckt.
  - `TestConnection` macht einen echten `GET /v1/profile` (neu: `profile.go`, der billigste
    authentifizierte Endpoint der Lexware-API). Bisher galt "Key im Vault nicht leer" als
    verbunden — ein widerrufener oder vertippter Key war gruen und brach erst beim naechsten Sync.
    Der Client holt den Key selbst aus dem Vault (`lexware_api_key_<tenant>`), also prueft der Test
    genau das Credential, das die Syncs spaeter benutzen, nicht ein daneben gelesenes.
  - VIER FUNDE, ohne die "verdrahtet" nur ein No-op mit mehr Ebenen gewesen waere. Der Reihe nach,
    weil jeder einzelne die Kette in Produktion tot gelassen haette:
    1. `cmd/biz/main.go:383` uebergab `nil` als ContactService (Kommentar: `nil, // ContactService`).
       Verdrahtet haette der erste echte Sync auf `cs.contacts.UpdateForSync` gepanict — vorher fiel
       das nicht auf, weil der Syncer nie lief. Adapter nachgezogen
       (`cmd/biz/lexware_contact_adapter.go`): duenner Wrapper, der den bestehenden
       `crmContactAdapter` auf `lexware.ContactService` uebersetzt (die beiden Interfaces sind
       strukturgleich, nur mit paket-eigenen Typen). ~90 Zeilen Typ-Uebersetzung statt ~200 Zeilen
       duplizierter gRPC-Logik. Zusaetzlich die nil-Guards, die `bexio/contact_sync.go:161,245`
       schon hat — ohne CRM_GRPC_ADDRESS wird der Sync uebersprungen statt zu sterben.
    2. `resolveContactLexwareID` (in beiden Pushern dupliziert) nahm `mappings[0]` — die erstbeste
       Kontakt-Zuordnung, voellig unabhaengig vom Rechnungsempfaenger. Das ist der gefaehrlichste
       Fund des Laufs: verdrahtet haette JEDE Rechnung und JEDES Angebot beim falschen
       Lexware-Kunden gebucht, still und beim Kunden abrechnungsrelevant. Ersetzt durch
       `contact_resolve.go` nach dem Vorbild `bexio/contact_resolve.go` — exakte Aufloesung ueber
       `contacts.GetByEmail` + Mapping auf die konkrete Kontakt-UUID; der Fallback ohne
       ContactService greift nur bei genau EINER vorhandenen Zuordnung und liefert sonst einen
       Fehler statt eines Rateschlusses. Test `TestPushInvoice_RefusesAmbiguousContact` prueft
       zusaetzlich, dass in diesem Fall NICHTS an die API geht (`stub.recorded()` leer).
    3. `HandleWebhookEvent` laeuft pre-JWT (Lexware authentifiziert per HMAC, kein Token) und hatte
       damit keinen Tenant im Context. Der erste Read auf `integration_configs` (RLS seit 000125)
       haette 0 Zeilen geliefert -> jeder Webhook waere als "integration not configured"
       gestorben. `sysctx.With(ctx)` am Eintrittspunkt der Service-Methode, nicht erst in
       `WebhookHandler.HandleEvent` (das wrappt schon, aber zu spaet — der Config-Read liegt davor).
    4. Der Gateway (`route_lexware.go`) dekodierte den Webhook-Body als snake_case
       (`event_type`/`resource_id`), Lexware sendet aber camelCase (`eventType`/`resourceId`, siehe
       die JSON-Tags an `LexwareWebhookEvent` im selben Repo). Ein echter Lexware-Webhook waere mit
       leerem event_type an `InvalidArgument` gescheitert — 400 zurueck an Lexware, bevor der
       Service je erreicht wird. Decode in `parseLexwareWebhookBody` gezogen (testbar ohne
       gRPC-Registry, weil `getLexwareClient` sonst vorher 503 liefert) und beide Schreibweisen
       akzeptiert, camelCase gewinnt.
  - Webhook inhaltlich statt nur delegiert: `contact.created`/`contact.changed` ziehen den
    geaenderten Datensatz (`GetContact`) und wenden ihn ueber denselben Pro-Datensatz-Pfad an wie ein
    Bulk-Sync. Dafuer den Schleifenkoerper von `syncInbound` als `upsertInboundContact` extrahiert
    und `SyncContactByLexwareID` daneben gestellt — Wiederverwendung, keine zweite
    Create/Update/Skip-Logik. Haette ich nur an `HandleEvent` delegiert, waere der Webhook nach wie
    vor ein No-op gewesen (die Methode loggte und emittierte nur), nur mit einer Ebene mehr.
  - `Scheduler.StartAll` nahm `config.CreatedBy` als Tenant-ID. Heute zufaellig identisch (`Connect`
    setzt `CreatedBy: tenantID`), aber der API-Client baut den Vault-Key aus der Tenant-ID —
    eine anders angelegte Config haette einen nicht existierenden Key gesucht. Auf
    `config.TenantID` umgestellt. Relevant erst ab jetzt, weil der Scheduler vorher nichts tat.
- entscheidung: `invoice.status.changed`/`quotation.status.changed` werden quittiert und geloggt,
  aber NICHT angewandt. Dem Lexware-Service fehlt ein `InvoiceStatusUpdater` (Bexio hat einen und
  faehrt darueber sein Payment-Polling); einen einzufuehren haette Interface + Konstruktor-Signatur +
  Wiring in `cmd/biz` bedeutet und damit die Verdrahtungs-Unit gesprengt. `lean:`-Marker
  mit Upgrade-Trigger steht in `webhook_handler.go` ("sobald Paid-Status-Rueckmeldung aus Lexware
  gebraucht wird"), Testfall `TestHandleWebhookEvent_DocumentStatus_IsAcknowledgedOnly` haelt das
  Verhalten fest, damit es nicht unbemerkt zu einem stillen Fehler wird.
- gate: build (`./internal/biz/lexware/... ./internal/gateway/... ./internal/server/... ./cmd/biz/...
  ./cmd/gateway/...`) ok | vet ok | golangci-lint **0 issues** | migration: keine (keine neue
  Tabelle/Policy) | openapi: kein Eintrag noetig (keine neue Route — der Webhook-Pfad existierte
  schon, nur sein Decoder war falsch) | Test mit `DATABASE_URL` (Rolle `kmuhub_app`):
  `internal/biz/lexware` **17 Tests, 0 Skips** (per `-v` gegengeprueft, inkl. der vier
  DB-Tenant-Isolationstests, die real gegen Postgres liefen), `internal/gateway` ok,
  `internal/server` ok. 15 neue Tests: 13 in `service_wiring_test.go` (Stub-Lexware-API per
  `httptest`, jede Operation wird ueber einen ECHTEN HTTP-Request nachgewiesen — ein Rueckgabewert
  von nil beweist bei dieser Unit gar nichts, genau das war ja der Bestand), 1 Table-Test fuer den
  Webhook-Decode, 1 fuer den nicht konfigurierten Webhook.
- verify vorgaenger: sauber (`3b481316`, CalDAV-Testendpoint) — kein Stub, Probe-Ziel aus der
  eigenen Config statt aus `r.Host` (SSRF dicht), App-Passwort im `defer` mit
  `context.Background()` revoked (ueberlebt Client-Abbruch), `openapi.yaml` im selben Commit,
  kein neuer Guard. Ein Detail nachgeprueft: die Route umgeht keinen gRPC-Client — die
  CalDAV-Nachbarrouten (`/status`, `/enable`, `/disable`) arbeiten alle direkt ueber
  `pwService`/`userPrefRepo` am Pool, das ist das Bestandsmuster dieses Moduls und kein
  Direct-Service-Bypass im Gateway-Sinn.
- offen:
  - Kein echter Lauf gegen die Lexware-Sandbox (Loop-Policy: kein externer Netzzugriff, kein
    API-Key). Alle Aussagen ueber die API-Form stuetzen sich auf den bestehenden Client/Parser im
    Repo (`v1/contacts`, `v1/invoices`, `v1/quotations`, Bearer-Auth, page/size-Paginierung) und auf
    `v1/profile` als Standard-Ping. Sollte der Profile-Endpoint in der genutzten API-Version anders
    heissen, faellt genau `TestConnection` auf die Nase — sichtbar und mit klarer Meldung, nicht
    still. Vor dem ersten Kundeneinsatz einmal gegen echte Credentials fahren.
  - `LEXWARE_WEBHOOK_SECRET` + `RegisterWebhooks` sind weiterhin nicht automatisch verdrahtet:
    `Connect` legt die Sync-Config an, registriert aber keine Webhook-Subscriptions bei Lexware
    (`WebhookHandler.RegisterWebhooks` wird von nirgendwo aufgerufen). Ohne diesen Schritt kommen
    real gar keine Events an, egal wie gut der Empfangspfad jetzt ist. Kandidat fuer eine Folge-Unit
    — braucht eine oeffentlich erreichbare Callback-URL aus der Config, also eine Entscheidung
    ueber deren Herkunft (nicht aus Request-Headern ableiten, siehe Iteration 42).
  - `TriggerSync` schluckt weiterhin den Fehler von `SyncContacts` (loggt und gibt nil zurueck) —
    der manuelle "Jetzt synchronisieren"-Knopf meldet damit immer Erfolg. Nicht angefasst, weil es
    ausserhalb der Verdrahtungs-Unit liegt; als eigener Fund notiert.

## Iteration 44 — g-auth-sessions-wiring — done — 2026-08-02 03:5x
- commit: 308fd217
- gebaut: Die Geraeteliste hat jetzt Inhalt. Bisher schrieb kein Pfad je eine
  `user_sessions`-Zeile — `Service.CreateSession` existierte, wurde aber von niemandem gerufen, also
  war `GET /auth/sessions` in Produktion eine garantiert leere Liste und `DELETE /auth/sessions/{id}`
  hatte nie ein Ziel.
  - **Der Backlog-Text ist zur Haelfte veraltet:** die Routen (`GET /auth/sessions`,
    `DELETE /auth/sessions/{id}`, `DELETE /auth/sessions`, `GET /auth/sessions/all`) samt
    gRPC-Handlern und openapi-Eintraegen existierten bereits und gehen sauber ueber
    `getAuthClient()`. Gefehlt hat nur die Schreibseite — und die Terminate-Route war unsicher
    (siehe unten). Neue Routen sind deshalb keine dazugekommen.
  - **Schreibseite an genau einer Stelle:** `createTokenPair` ist der einzige Ort, durch den alle
    fuenf Token-Pfade laufen (Register, Login, 2FA-Abschluss, Refresh, Einladungsannahme). Dort
    haengt jetzt `recordSession`. Ein zweiter Parameter `rotatedFrom *uuid.UUID` unterscheidet
    Neuanmeldung von Rotation: bei Refresh wird die BESTEHENDE Zeile auf den neuen Token
    umgehaengt, nicht eine zweite angelegt. Ohne das waere die Geraeteliste um einen Eintrag pro
    Refresh-Intervall (15 min) gewachsen, jeder davon wie ein eigener Login aussehend.
  - **Transportweg fuer IP/User-Agent ohne Proto-Aenderung:** neues Mini-Paket
    `internal/clientctx` (analog `sysctx`, aus demselben Grund — `middleware` importiert `auth`,
    die Keys koennen also nicht in `middleware` liegen). Kette: neue HTTP-Middleware
    `middleware.ClientInfo(behindProxy)` (nutzt das bestehende `ClientIPTrusted`, also die
    proxy-sichere Variante) -> Context -> die BESTEHENDEN Interceptoren in `grpc_tenant.go` um zwei
    Header erweitert (`x-client-ip`, `x-client-user-agent`) -> Service-Context. Alternative waere
    gewesen, vier Proto-Messages zu erweitern und zu regenerieren; der Interceptor war schon da.
  - **Aufbewahrung:** `DeleteStaleUserSessions` raeumt beim Anmelden die Zeilen weg, deren
    Refresh-Token weg/revoked/abgelaufen ist (nicht mehr erreichbar, tragen aber IP + Geraet).
    `Logout` loescht die Zeile des abgemeldeten Geraets — vorher waere sie als "aktiv" stehen
    geblieben. `lean:`-Marker mit Upgrade-Trigger auf einen Scheduler steht an `recordSession`.
- sicherheitsfund im bestand (behoben, war der eigentliche Fund der Unit): `TerminateSession`
  parste `req.UserId` und **warf ihn weg** — Kommentar im Code: "UserId is available for
  authorization checks but TerminateSession only needs sessionID". Die Session-ID kommt aus der URL
  einer authentifizierten Route, die User-ID aus dem JWT. Jeder eingeloggte User konnte damit jede
  fremde Session beenden, deren ID er kannte oder erriet — inklusive Revoke des fremden
  Refresh-Tokens, also ein Fremd-Logout per Request. RLS haette nur tenant-uebergreifend gebremst,
  innerhalb eines Tenants gar nicht. Jetzt prueft der **Service** die Zugehoerigkeit
  (`TerminateSession(ctx, sessionID, ownerID)`) und meldet `ErrSessionNotFound` statt Forbidden,
  damit die Existenz einer fremden ID nicht abfragbar ist. Test auf gRPC-Ebene ergaenzt, weil dort
  der weggeworfene Parameter sass.
- zweiter fund (behoben, vom DB-Test aufgedeckt): die drei Session-SELECTs lasen
  `COALESCE(ip_address::text, '')` — INET castet MIT Praefixlaenge, die API haette also
  `203.0.113.7/32` als IP-Adresse des Geraets ausgeliefert. Auf `host(ip_address)` umgestellt.
  Waere ohne echten DB-Test nicht aufgefallen: der Mock haelt einen String.
- dritter fund (praeventiv behoben): `ip_address` ist INET, und der INSERT band den Go-String
  direkt. Ein leerer String (kein XFF, kein lesbares RemoteAddr) haette den ganzen INSERT mit
  SQLSTATE 22P02 gekippt — exakt der Fehler aus `security/audit`, Iteration 60 in Lauf 1. Jetzt
  `NULLIF($7,'')::inet`. Eigener Test dafuer (`LoginWithoutClientInfo`): eine unlesbare
  Client-Adresse darf niemanden am Anmelden hindern.
- entscheidung: Session-Schreibfehler werden geloggt, nicht hochgereicht. Die Geraeteliste ist eine
  Komfortansicht; einen gueltigen Login abzulehnen, weil seine Buchhaltungszeile nicht geschrieben
  werden konnte, tauscht ein kosmetisches Problem gegen eine Aussperrung. Testfall haelt das fest.
- gate: build (`./internal/... ./cmd/gateway/... ./cmd/auth/...`, `-p 2` wegen OOM bei vollem
  `./...`) ok | vet ok | golangci-lint **0 issues** | migration: keine (Tabelle, Spalten und RLS
  existieren seit 000039/000114/000120) | openapi: kein neuer Pfad; die 404-Antwort von
  `DELETE /auth/sessions/{id}` deckt jetzt zusaetzlich "gehoert einem anderen User" ab, das steht
  im selben Commit in der Beschreibung | rls-smoke: die vier Bestands-Tests in
  `rls_user_sessions_test.go` liefen real mit (siehe Skip-Zahl) | Test mit `DATABASE_URL`
  (Rolle `kmuhub_app`): `internal/auth` **114 Tests, 0 Skips** (per `-v` gezaehlt),
  `internal/server` ok, `internal/gateway` ok (TestOpenAPIRouteDrift), `internal/middleware` ok.
  14 neue Tests: 4 DB-Lifecycle (Login->Refresh->Logout, Login ohne Client-Info, Fremd-Terminate
  abgelehnt, Terminate macht Refresh-Token ungueltig), 6 Service-Verdrahtung, 1 gRPC-Ownership,
  4 Transportkette (inkl. Interceptor-Paar-Roundtrip).
- verify vorgaenger: sauber (`f2f362f9`, Lexware-Verdrahtung) — keine Stubs, kein Proto-Drift, keine
  neue Route ohne openapi-Eintrag, kein neuer Guard ohne Seed; `sysctx` sitzt am
  HMAC-authentifizierten Webhook-Eintritt und ist dort begruendet. **Ein Nebenfund, nicht meine
  Unit:** `Service.HandleWebhookEvent` liest die Config unter `sysctx` per
  `configRepo.GetByPlatform(ctx, "lexware")` — ohne Tenant-Bedingung. Bei mehreren Tenants mit
  Lexware-Integration bekommt jeder Webhook die zufaellig erste Zeile und bucht auf den falschen
  Tenant. Vor der Verdrahtung war es harmlos (0 Zeilen, alles schlug fehl), jetzt schreibt es.
  Der Webhook traegt eine `organization_id` — darueber waere die Config eindeutig aufloesbar.
  Heute nicht akut (Daten sind Single-Tenant), aber vor Tenant 2 zu schliessen.
- offen:
  - `is_current` bleibt auf allen Zeilen `false`. Die Spalte existiert seit 000039 und das Proto
    liefert sie aus, aber kein Leseweg kennt die eigene Session-ID: der Client sieht sie nirgends,
    weil sie weder im JWT noch in einer Login-Antwort steht. Folge: die UI kann "dieses Geraet"
    nicht markieren, und der Query-Parameter `current_session_id` von
    `DELETE /auth/sessions` ist von aussen nicht befuellbar — "alle anderen abmelden" meldet damit
    heute zwangslaeufig auch das eigene Geraet ab. Sauber loesbar nur mit der Session-ID im
    Token (JWT-Claim oder Login-Response), das ist ein eigener Schnitt.
  - Kein Gateway-HTTP-Test fuer die Session-Routen (die Route-Tests des Pakets pruefen dort nur
    503/400 vor dem gRPC-Call). Die Ownership-Pruefung ist auf gRPC- und Service-Ebene getestet,
    der HTTP-Weg dorthin nicht.

## Iteration 45 — g-rapporte-pdf — done — 2026-08-02

- commit: 7a01590f
- gebaut: `ExportPDF` liefert jetzt ein echtes A4-PDF statt des Text-Stubs. Neue Datei
  `internal/rapporte/pdf.go` (`renderReportPDF`) nutzt `maroto/v2` direkt (dieselbe Dependency wie
  `internal/berichte/export/pdf.go`, keine neue) — Titel, Status (deutsches Label), Projekt,
  Beschreibung, Arbeitszeit/Pause/Wetter/Temperatur, Mitarbeiterliste (Name/Rolle/Stunden),
  Positionstabelle (Pos/Beschreibung/Menge/Einheit/Notiz) und eine Unterschriftszeile.
  `Service.ExportPDF` holt dafuer zusaetzlich `s.repo.ListLines(...)` (vorher ungenutzt fuer den
  Export). `RapporteGRPCServer.ExportPDF` liefert `ContentType: application/pdf` und
  `arbeitsbericht_<id>.pdf` statt `.txt`.
  ENTSCHEIDUNG gegen `internal/biz/pdf.Generator`: dessen `newMaroto()`/`GenerateInvoicePDF` etc.
  sind an `models.CompanySettings` und `ValidateCompanySettingsForPDF` (§14-UStG-Pflichtfelder)
  gebunden — fuer einen Arbeitsbericht (kein Rechnungsdokument) nicht einschlaegig, und
  `internal/rapporte` importierte bisher weder `models` noch `biz/pdf`. Eigene, schlanke
  `maroto/v2`-Nutzung nach dem Vorbild von `internal/berichte/export/pdf.go` (rohe
  row/col/text-Primitive), keine neue Abstraktion.
  Die gespeicherte Kundenunterschrift (`signature_data`, PNG/SVG-Data-URL) wird NICHT als Bild
  eingebettet — nirgends im Repo existiert bisher eine maroto-Bild-Einbettung, und das war nicht
  Teil des `done_when`. Stattdessen Textzeile "Unterschrieben von X am Y" bzw. "ausstehend".
  `lean:`-Marker in `pdf.go` mit Upgrade-Trigger "wenn ein Kunde die Bild-Einbettung im PDF
  verlangt".
- gate: build (`./internal/rapporte/... ./internal/server/... ./cmd/rapporte/... ./cmd/gateway/...`,
  `-p 2`) ok | vet ok | golangci-lint **0 issues** | migration: keine (kein Schema-Wechsel) |
  openapi: kein Eintrag noetig (Content-Type/Filename kommen dynamisch aus der gRPC-Response, die
  Route existierte bereits) | rls-smoke: n.a. (keine Tabelle/Policy angefasst) | Test mit
  `DATABASE_URL` (Rolle `kmuhub_app`): `internal/rapporte` **0 Skips** (Report-Lines/Signature/
  Tenant-Isolation-DB-Tests liefen real mit), `internal/server` ok, `internal/gateway` ok
  (`TestOpenAPIRouteDrift`, vorsorglich gefahren trotz unveraenderter Routen).
  Test umbenannt/erweitert: `TestService_ExportPDF_ReturnsPayload` -> `..._ReturnsRealPDF`, prueft
  jetzt den `%PDF`-Header (vorher nur `NotEmpty`) und laeuft mit Workers + einer Line +
  Signatur-Feldern durch alle neuen Codepfade.
- verify vorgaenger: sauber (`308fd217`, Auth-Sessions-Wiring) — `TerminateSession(sessionID,
  ownerID)`-Ownership-Check korrekt verdrahtet (Mismatch -> `ErrSessionNotFound`, nicht Forbidden),
  INET-Lesart auf `host(ip_address)` umgestellt (statt `::text` mit Praefixlaenge),
  `NULLIF($7,'')::inet` gegen den leeren-String-INSERT-Fehler, kein neues Proto, keine neue
  Tabelle/Migration, keine neue Route (openapi-Diff nur die 404-Beschreibung), kein neuer Guard.
  Sauber.
- offen:
  - Signatur-Bild wird nicht ins PDF eingebettet (siehe oben, `lean:`-Marker in `pdf.go`).
  - `internal/biz/pdf` bleibt der einzige Ort mit `maroto`-Bild-faehiger Infrastruktur
    (`EmbedZUGFeRDXML`/pdfcpu-Attachments) — falls Rapporte-Bild-Einbettung spaeter gebraucht wird,
    dort zuerst nach wiederverwendbaren Bausteinen schauen, nicht neu bauen.

## Iteration 46 — g-vertraege-pdf — done — 2026-08-02

- commit: 2577bc49
- gebaut: `ExportContract` liefert jetzt ein echtes, mehrseitiges A4-PDF statt des Klartext-Dumps.
  Neue Datei `internal/vertraege/pdf.go` (`renderContractPDF`) nutzt `maroto/v2` direkt (dieselbe
  Dependency wie `internal/rapporte/pdf.go`, keine neue) — Titelzeile mit Vertragsnummer, Titel,
  Status/Art (deutsche Labels), Beginn/Ende (bzw. "unbefristet"), Notizen, Vertragsparteien und eine
  Unterschriftszeile.
  Konsequent `text.NewAutoRow` (auto-Hoehe) statt fixer `text.NewRow`-Hoehen fuer alle Inhaltszeilen:
  maroto v2 berechnet die Zeilenhoehe aus dem umgebrochenen Text und `AddRows` paginiert automatisch,
  sobald eine Zeile die Restflaeche der Seite sprengt (siehe `maroto.go:addRow`) — mit fixer Hoehe
  waere langer Text einfach abgeschnitten statt umgebrochen.
  Lange Notizen werden zusaetzlich an Leerzeilen in Absaetze gesplittet (`notesParagraphs`, eigene
  Zeile pro Absatz) statt als ein einziger Riesen-Block: maroto splittet eine einzelne Zeile NICHT
  seitenuebergreifend, nur zwischen Zeilen. Ohne den Split wuerde ein hinreichend langer
  Notizen-Block als eine Zeile behandelt und koennte die Seite ueberragen, statt sauber umzubrechen.
  `lean:`-Marker dafuer in `pdf.go` mit Upgrade-Trigger "wenn ein einzelner Absatz eine Seite sprengt".
  Vertragsparteien: `ContractParty` hat fuer `contact`/`company` nur eine ID, keinen aufgeloesten
  Namen (anders als bei Rapporte-Workern, die den Namen bereits inline tragen) — PDF zeigt fuer diese
  Faelle "Kontakt <ID-Praefix>"/"Firma <ID-Praefix>", fuer `external` den echten `ExternalName`. Kein
  zusaetzlicher Repo-Join eingefuehrt, das war nicht Teil des `done_when`.
  Unterschriftszeile analog Rapporte: `SignedBy`/`SignedAt` vom Contract, sonst "Unterschrift:
  ausstehend". Auch hier `lean:`-Marker: Signatur-Bild (`signature_data`) wird nicht eingebettet.
  `GetContract` liefert `Parties` bereits mit (siehe `postgres_repository.go`), also kein
  zusaetzlicher Repo-Call wie bei Rapporte's `ListLines` noetig.
- gate: build (`./internal/vertraege/... ./internal/server/... ./cmd/vertraege/... ./cmd/gateway/...`,
  `-p 2`) ok | vet ok | golangci-lint **0 issues** | migration: keine (kein Schema-Wechsel) |
  openapi: kein Eintrag noetig (Route existierte bereits, nur der Payload-Inhalt aendert sich) |
  rls-smoke: n.a. (keine Tabelle/Policy angefasst) | Test mit `DATABASE_URL` (Rolle `kmuhub_app`):
  `internal/vertraege` **0 Skips** (inkl. `TestTenantIsolation_Vertraege`,
  `TestVertraegeWrites_LandInCallerTenant`), `internal/server` ok, `internal/gateway` ok
  (`TestOpenAPIRouteDrift`: 771 Routen gegen 773 dokumentierte Pfade, keine Drift).
  Test ersetzt/erweitert: `TestService_ExportContract_TextDump` -> `..._ReturnsRealPDF`, prueft jetzt
  den `%PDF`-Header (vorher nur den Klartext-Inhalt) mit Notizen + einer External-Partei + Signatur.
  Neuer Test `TestService_ExportContract_LongNotesPaginate`: 200 Notiz-Absaetze, parst den `/Pages`-
  Objektkopf aus den rohen PDF-Bytes per Regex (`/Type\s*/Pages.*?/Count\s+(\d+)`) und prueft
  `pageCount > 1`. Verifiziert per Wegwerf-Skript, dass maroto ohne `WithCompression` (Default: aus)
  die Objektstruktur unkomprimiert im Klartext ablegt, `/Count` also direkt grep-bar ist — kein PDF-
  Parser als neue Dependency noetig.
- verify vorgaenger: sauber (`7a01590f`, Rapporte-PDF) — Build der betroffenen Pakete lief hier erneut
  gruen, Text-Stub sauber durch `renderReportPDF` ersetzt, tenant-gescopt ueber `s.repo.GetReport`/
  `ListLines`, keine neue Route/Guard, `lean:`-Marker fuer die ausgelassene Signatur-Bild-Einbettung
  korrekt gesetzt.
- offen:
  - Signatur-Bild wird nicht ins PDF eingebettet (wie bei Rapporte, `lean:`-Marker in `pdf.go`).
  - Sehr lange EINZELNE Notiz-Absaetze (kein Zeilenumbruch im Quelltext ueber eine ganze Seite
    hinweg) wuerden weiterhin nicht seitenuebergreifend umbrechen (`lean:`-Marker, siehe oben) — kein
    reales Szenario heute (Contract-Notizen sind Kurztext), aber falls das Feld je fuer lange
    Fliesstexte genutzt wird, dort ansetzen.
  - Party-Namen fuer `contact`/`company` zeigen nur ein ID-Praefix statt des echten Namens (kein
    Contact-/Company-Join in `ExportContract`, siehe oben) — falls das stoert, Repo-Join ergaenzen.

## Iteration 47 — fix-g-vertraege-pdf — done — 2026-08-02

- commit: (siehe unten)
- verify vorgaenger: **Befund** in Commit `2577bc49` (g-vertraege-pdf, Iteration 46). Der PDF-Renderer
  wurde eingebaut, aber `VertraegeGRPCServer.ExportContract`
  (`internal/server/vertraege_grpc.go:235-239`) blieb unveraendert bei
  `ContentType: "text/plain; charset=utf-8"` und `Filename: "contract.txt"` aus der Text-Dump-Aera —
  ein Client, der der Content-Type-Angabe vertraut (Browser-Download, MIME-Sniffing), speichert/oeffnet
  PDF-Bytes als `.txt`. Fehlerklasse 2 (Stub/veraltete Restdaten im neuen Pfad), keine der anderen
  fuenf Klassen betroffen (kein Proto, keine Migration, kein Guard, keine Route, kein Wire-Shape jenseits
  dieses einen Feldpaars). Exakt dasselbe Muster war schon im Rapporte-Aequivalent (Iteration 45,
  `7a01590f`) korrekt gesetzt (`rapporte_grpc.go:527-531`) — dort als Vorlage uebernommen.
  Fix-Unit `fix-g-vertraege-pdf` ganz vorne in BACKLOG.yml angelegt und sofort als diese Iteration
  abgearbeitet (Prozess-Schritt 1).
- gebaut: `ExportContract` liefert jetzt `ContentType: "application/pdf"` und
  `Filename: "vertrag_<contractID-praefix>.pdf"` (analog `rapporte_grpc.go`'s
  `arbeitsbericht_<id-praefix>.pdf`). Neuer Import `fmt` in `vertraege_grpc.go`.
  KEIN neuer Test: es existiert weder fuer `vertraege_grpc.go` noch fuer `rapporte_grpc.go` ein
  Server-Ebenen-Testfile (kein Mock-Harness fuer `VertraegeGRPCServer`/`RapporteGRPCServer` im Repo) —
  eins nur fuer diese eine Struct-Literal-Korrektur neu aufzuziehen waere Over-Engineering fuer einen
  Ein-Zeilen-Fix. Der PDF-Inhalt selbst ist bereits auf Service-Ebene abgedeckt
  (`TestService_ExportContract_ReturnsRealPDF`, `%PDF`-Header-Check, Iteration 46).
- gate: build (`./internal/vertraege/... ./internal/server/... ./cmd/vertraege/... ./cmd/gateway/...`,
  `-p 2`) ok | vet ok | golangci-lint **0 issues** | migration: keine | openapi: kein Eintrag noetig
  (Route unveraendert, nur Response-Feldwerte) | rls-smoke: n.a. | Test mit `DATABASE_URL` (Rolle
  `kmuhub_app`): `internal/vertraege` **0 Skips** (alle 36 Tests inkl. Tenant-Isolation liefen real),
  `internal/server` ok, `internal/gateway` ok (`TestOpenAPIRouteDrift` vorsorglich gefahren, keine
  Route angefasst).
- offen:
  - Aus Iteration 46 uebernommen (unveraendert): Signatur-Bild nicht eingebettet, einzelne Notiz-
    Absaetze brechen nicht seitenuebergreifend um, Party-Namen fuer contact/company nur ID-Praefix.
  - Kein Server-Ebenen-Test fuer `ExportContract`/`ExportPDF` (weder vertraege noch rapporte) — falls
    an dieser Grenzflaeche kuenftig oefter Fehler auftreten, lohnt sich ein gemeinsames Test-Harness
    fuer `*GRPCServer`-Response-Mapping.

## Iteration 48 — g-vermietung-inspection-upload — blocked — 2026-08-02

- commit: - (keine Code-Aenderung, nur BACKLOG.yml/JOURNAL.md)
- verify vorgaenger: sauber. `811e24de` (fix-g-vertraege-pdf, Iteration 47) gegen alle acht
  Fehlerklassen geprueft: reiner Feldwert-Fix (`ContentType`/`Filename`) in
  `vertraege_grpc.go:235-239`, kein Proto, keine Migration, kein neuer Guard, keine neue Route,
  kein neues Wire-Shape jenseits dieses einen Feldpaars, kein direkter Service-Zugriff im Gateway.
  Kein Fund.
- gebaut: nichts — Unit als `blocked` markiert, siehe `blocked_reason` in BACKLOG.yml.
- Grund (Kurzfassung, Volltext im Backlog-Eintrag): `UploadInspectionPhoto` (bytes-embed im
  Request, Platzhalter-URL im Service, kein Gateway-Handler) hat NIRGENDS einen echten Aufrufer —
  weder FE noch Gateway (Volltextsuche ueber `desktop/src` und `backend/`). Die reale
  Foto-Upload-Funktion fuer Inspektionen ist bereits vollstaendig verdrahtet, aber ueber einen
  ANDEREN Pfad: Browser-direct-Upload nach MinIO via `presignUpload('vermietung', file)` (Scope
  `vermietung` bereits erlaubt in `document/file/presign.go:24-34`, 50-MB-Limit,
  Content-Type-Pflicht) gefolgt von `CreateInspection.photo_urls`/`UpdateInspection.photo_urls` —
  verifiziert in `ZustandsprotokollDialog.tsx:142-162` (Kommentar dort: "the backend already
  persists them end-to-end"). Der zugehoerige FE-Hook `useUploadInspectionPhoto`
  (`useVermietung.ts:275-284`) delegiert selbst nur an `updateInspection` und hat keinerlei
  Call-Site — auch er ungenutzt.
  Die "Gegenstelle bauen" wie urspruenglich gescopt haette eine zweite, parallele
  Upload-Infrastruktur fuer exakt dieselbe Funktion errichtet (Lean-Code/YAGNI-Verstoss), zudem mit
  dem architektonisch schwaecheren Muster (Rohbytes im gRPC-Request vs. presign-basierter
  Browser-Direct-Upload) — vermutlich der Grund, warum die spaetere Implementierung
  (`ZustandsprotokollDialog`) diesen RPC nie benutzt hat.
  Zwei ehrliche Wege, keiner spontan zu entscheiden: (A) `UploadInspectionPhoto` als
  totes/superseded RPC vollstaendig entfernen (Proto, Service, gRPC-Handler, ungenutzter FE-Hook).
  (B) Den RPC fuer einen bislang nicht existierenden Anwendungsfall bauen — ein externer Mieter ohne
  CRM-Login laedt direkt hoch; dafuer fehlt aber im gesamten Repo jedes Mieter-Portal/Auth-Modell
  fuer Nicht-CRM-Nutzer (verifiziert, keine Treffer). Weg B waere kein Upload-Fix, sondern ein neues
  Feature samt eigener Auth-Entscheidung.
- gate: n.a. (kein Code geaendert)
- offen:
  - **Entscheidung fuer Luke:** siehe `blocked_reason` in BACKLOG.yml — Loeschung des toten RPCs
    (A) oder Neubau als echtes Mieter-Portal-Feature (B).
  - Naechste Unit laut Reihenfolge: `g-notification-email-adapter` (deps: [], unabhaengig von
    dieser blockierten Unit).

## Iteration 49 — g-notification-email-adapter — done — 2026-08-02

- commit: (siehe unten)
- verify vorgaenger: sauber. Letzter Commit `398583e9` (Iteration 48) war reine Doku
  (BACKLOG.yml/JOURNAL.md, `g-vermietung-inspection-upload` auf `blocked`), kein Code zu pruefen.
- gebaut: `cmd/notification/main.go` wired `adapter.EmailAdapter` jetzt gegen einen echten
  `emailGRPCClient` (Wrapper-Typ im `main`-Package, Muster wie `crmDealUpdater` in
  `cmd/biz/main.go`): `grpc.NewClient(cfg.EmailGRPCAddress, ..., middleware.
  TenantOutboundUnaryInterceptor())`, `defer emailConn.Close()`, Dial-Fehler nur geloggt (Notification
  startet trotzdem). `SendReply`/`ForwardEmail`/`MarkRead` loesen zuerst per `GetEmailAccount(user_id)`
  die Account-ID auf, dann `ReplyEmail`/`ForwardEmail`/`MarkRead`-RPC. `ListMessages` (fuer das bisher
  nirgends aufgerufene `FetchNewMessages`) mitgebaut, `lean:`-markiert (Einzelseite, kein
  Multi-Ordner-Sweep — Upgrade wenn ein Poller den Pfad tatsaechlich nutzt).
  ROOT-CAUSE-FUND beim Bauen, ohne den `done_when` "Forward und Reply stellen real zu" nicht
  erfuellbar: `message.Service.Reply`/`.Forward` reichten `msg.ID` (Inbox-Message-eigene ID) statt
  `msg.SourceID` (ID im Herkunftssystem, hier email_messages.id) an den Adapter durch — ein echter
  Client haette `ReplyEmail`/`ForwardEmail` deterministisch mit `NotFound` scheitern lassen.
  Fix in der gemeinsamen Funktion (`internal/inbox/message/service.go`), `ChannelAdapter`-Interface
  von `messageID uuid.UUID` auf `sourceID string` umgestellt (passt zu `InboxMessage.SourceID string`),
  alle vier Adapter (Email/Chat/Notification/Guest) + Test-Mock mitgezogen. Chat war vom selben Bug
  betroffen (`CreateMessage` haette ebenfalls die falsche ID bekommen) und ist mit demselben Diff
  mitkorrigiert; Notification/Guest ignorieren den Parameter ohnehin (kein Verhaltensunterschied).
  Neue Tests `TestReply_Success`/`TestReply_NoAdapter` (gab es vorher nicht), `TestForward_Success`
  um `SourceID`-Assertion erweitert.
  Kein Test fuer `emailGRPCClient` selbst — Konvention im Repo: kein einziges `cmd/*/main.go`
  (inkl. der analogen `crmDealUpdater`/`SendEmailAction`-Wrapper) hat je eine `_test.go`-Datei, diese
  duenne RPC-Mapping-Schicht wird nicht auf Package-Ebene getestet.
  GEFUNDEN, NICHT BEHOBEN (ausserhalb Scope): `ChatAdapter`/`NotificationAdapter` bleiben in
  `cmd/notification/main.go` mit `client=nil` registriert — nur Email war gescopt. Kandidat fuer
  Folge-Unit, Notiz in BACKLOG.yml bei dieser Unit hinterlegt.
- gate: build (`internal/inbox/... internal/notification/... internal/gateway/... cmd/notification/...
  cmd/gateway/...`, `-p 2`) ok | vet ok | golangci-lint **0 issues** (inkl. `cmd/notification/...`
  separat geprueft) | migration: keine | openapi: keine neue Route, kein Eintrag noetig |
  rls-smoke: n.a. (kein neues Tabellen-Schema) | Tests mit `DATABASE_URL` (Rolle `kmuhub_app`):
  `internal/inbox/...`, `internal/notification/...`, `internal/gateway/` (inkl.
  `TestOpenAPIRouteDrift`, 771 Routen/773 Pfade, keine Drift) alle **ok**, keine Skips beobachtet.
- offen:
  - Chat-/Notification-Adapter weiterhin `client=nil` (siehe oben) — Folge-Unit-Kandidat
    `g-notification-chat-adapter`.
  - `ListMessages`/`FetchNewMessages` bleibt bis auf Weiteres toter Code (kein Poller-Aufrufer im
    Repo) — falls Luke einen Inbox-Polling-Worker plant, dort die `lean:`-Markierung in
    `cmd/notification/main.go` als Startpunkt nehmen (Multi-Ordner + echte Pagination fehlen noch).

## Iteration 50 — g-berichte-scheduler — done — 2026-08-02

- commit: `51b2f999` feat(berichte): deliver scheduled reports by email
- verify vorgaenger: sauber. `37a8c8a5` (Iteration 49, notification/EmailAdapter) gegen die sechs
  Fehlerklassen geprueft: kein Gateway-Handler beruehrt (also kein Direct-Svc-Bypass), keine
  Migration, kein Proto, keine neue Route (kein openapi-Bedarf). Die beiden neuen `return nil, nil`
  in `cmd/notification/main.go:602/620` sind echte Leerfaelle (kein Email-Account des Users, kein
  Inbox-Ordner), kein Fake-Erfolg. `TenantOutboundUnaryInterceptor` ist am Client gesetzt.
- ORT DER AUSFUEHRUNG — Entscheidung: **Ticker im berichte-Service, nicht pg_cron.** pg_cron kann
  nur SQL ausfuehren; ein Bericht braucht den Go-Executor (Downstream-Repos, JSONB-Params,
  Cache-Lookup), den PDF/XLSX-Renderer (maroto/excelize) und einen SMTP-Versand mit Anhang — nichts
  davon ist aus einer Datenbank-Session heraus erreichbar. pg_cron haette bestenfalls eine
  Job-Tabelle markieren koennen, die derselbe Go-Prozess dann doch pollen muesste; das ist der
  Ticker mit einer zusaetzlichen beweglichen Komponente. Die Idempotenz, wegen der pg_cron
  ueberhaupt zur Debatte stand, liegt ohnehin in der DB: `ClaimSchedule` ist ein Compare-and-Set auf
  `last_run_at` (`UPDATE … WHERE id=$2 AND last_run_at [IS NULL | = $3]`), also fuer beliebig viele
  Replicas gueltig.
- gebaut: Der Scheduler-Kern existierte bereits vollstaendig (`internal/berichte/scheduler`, inkl.
  Cron-Parsing, Claim, Statuspflege, Mailtexte) — was fehlte, waren die konkreten Kollaborateure:
  `cmd/berichte/main.go` uebergab `scheduler.New(repo, svc, nil, nil, …)`, wodurch **jeder** faellige
  Bericht mit `last_run_status=skipped, "exporter not configured"` endete. Vier neue Bausteine:
  1. `internal/email/systemmail` — System-SMTP-Sender (Dial-Timeout, Exchange-Deadline, STARTTLS,
     optionales PLAIN-Auth, MIME ueber den bestehenden `email/send`-Builder). LEAN-ENTSCHEIDUNG:
     nicht als dritte Kopie in `cmd/berichte/mailer.go` geschrieben, sondern aus `cmd/biz/mailer.go`
     hochgezogen; `cmd/biz` ist im selben Commit darauf umgestellt (mechanischer Diff, Verhalten
     identisch inkl. "SMTP nicht konfiguriert -> loggen statt Fehler" fuer Mahnungen).
     `Send` liefert `ErrNotConfigured` statt eines stillen Erfolgs — der Aufrufer entscheidet.
  2. `export.Render(result, name, format)` — rendert in Bytes plus ContentType plus Dateiname
     (`<slug>_<YYYY-MM-DD>.<ext>`, deutsche Umlaute transliteriert, damit "Umsatzuebersicht" nicht
     zu "umsatzbersicht" wird). Faellt auf `bericht_` zurueck, wenn der Name nichts Druckbares hat.
  3. `internal/berichte/delivery` — die beiden Adapter (`scheduler.Exporter`/`scheduler.Mailer`).
     Bewusst ein eigenes Package statt `cmd/berichte`-lokaler Typen: in `package main` waere die
     Zustellkette nicht testbar gewesen, und "wird zugestellt" waere eine Behauptung geblieben.
  4. `internal/testutil/fakesmtp` — In-Process-SMTP-Server (Greeting/EHLO/MAIL/RCPT/DATA/QUIT), von
     systemmail- und delivery-Test gemeinsam genutzt.
  Wiring in `cmd/berichte/main.go`: Exporter immer, Mailer nur wenn `cfg.SystemSMTPHost` gesetzt ist
  — sonst `nil` plus `slog.Warn`, damit der Scheduler weiter ehrlich `skipped` schreibt statt eine
  Zustellung zu behaupten. **Keine neue `config.RequireX`-Assertion**: `RequireSystemSMTP` existiert
  zwar, wurde aber NICHT an `config.Load` dieses Services gehaengt (waere ein Crash-Loop in der
  Produktion, sobald SYSTEM_SMTP_* im berichte-Container fehlt). Compose-Passthrough
  `${SYSTEM_SMTP_*:-}` beim berichte-Service ergaenzt (Muster von biz/auth), prod-Override braucht
  nichts.
- verifiziert, nicht angenommen: `RunReport` holt die Definition tenant-gescoped
  (`GetDefinition(ctx, in.TenantID, …)`) und alle acht Executor-Pfade reichen `def.TenantID` explizit
  durch — der Scheduler laeuft zwar unter `database.WithSystemContext` (er muss tenant-uebergreifend
  listen), die Berichtsdaten selbst bleiben trotzdem am Tenant des Schedules. Das war die Stelle, an
  der ein Leck teuer geworden waere, weil das Ergebnis jetzt real per Mail rausgeht.
- gate: build (`./cmd/... ./internal/...`, `-p 2`) ok | vet ok | golangci-lint **0 issues**
  (`internal/berichte/... internal/email/systemmail/... internal/testutil/... cmd/berichte/...
  cmd/biz/...`) | migration: keine | openapi: keine neue Route | Tests mit `DATABASE_URL`
  (Rolle `kmuhub_app`): `internal/berichte/...` (inkl. neuem DB-Test), `internal/email/...`,
  `internal/testutil/...`, `internal/biz/dunning/...` alle **ok**.
  `-race` lokal NICHT lauffaehig (kein gcc auf dieser Maschine, `cgo: C compiler "gcc" not found`) —
  CI faehrt es. Die zwei neuen Nebenlaeufigkeiten sind bewusst harmlos: der Claim-Test schreibt in
  disjunkte Slice-Indizes und synchronisiert per `WaitGroup`, der fakesmtp-Server publiziert seine
  Session ueber einen Channel.
- Tests zu den done_when-Punkten:
  - `delivery_test.go: TestScheduledReport_RenderedAndDelivered` — faelliger Schedule laeuft durch
    den echten Exporter und den echten Mailer, der Fake-SMTP-Server bekommt beide Empfaenger, der
    Anhang heisst `umsatzuebersicht-q3_2026-08-02.csv` und enthaelt nach Base64-Dekodierung die
    Report-Zeile. Gegenstueck `…_UnconfiguredMailerMarksFailed` beweist, dass der Pfad wirklich
    laeuft und ein fehlender Mailer als `failed` landet, nicht als Erfolg.
  - `internal/berichte/schedule_claim_test.go` — DB-Test gegen echtes Postgres: zwei gleichzeitige
    Claims auf denselben Schedule ergeben **genau einen** Gewinner, ein Replay des veralteten Claims
    verliert, und der Folge-Tick mit aktuellem `last_run_at` kann wieder claimen (sonst wuerde ein
    Schedule genau einmal feuern und dann feststecken). Der bestehende `scheduler_test.go` deckt nur
    die Entscheidungslogik gegen ein Fake-Repo ab und konnte ueber die Atomizitaet der SQL nichts
    aussagen.
- offen:
  - `cmd/berichte` verdrahtet in `executor.Deps` nur `KPI`; finance/crm/helpdesk/inventar/datev
    liefern weiterhin `emptyResult(def, "downstream_not_available")`. Ein Schedule auf so einen Kind
    wird jetzt real zugestellt — mit leerer Tabelle und Warnhinweis im Mailtext. Folge-Unit-Kandidat
    `g-berichte-downstream-wiring`; Notiz in BACKLOG.yml bei dieser Unit hinterlegt.
  - `cmd/auth/mailer.go` bleibt der letzte handgeschriebene System-SMTP-Pfad (eigener MIME-Bau ohne
    Attachments). Umstellung auf `systemmail` waere der Rest des Duplikat-Abbaus.
  - Fuer Luke vor dem Merge: die SYSTEM_SMTP_*-Werte der Produktionsumgebung pruefen — ohne Host
    bleibt die Zustellung aus (sichtbar als `last_run_status=skipped` und einer Warnzeile beim
    Servicestart), der Service laeuft aber normal weiter.

## Iteration 51 — g-dialer-agent-status-log-tenant — done — 2026-08-02

- commit: `324f7df0` fix(dialer): scope GetAgentStats active-campaign lookup by tenant
- verify vorgaenger: sauber. `51b2f999` (Iteration 50, berichte-scheduler) gegen die acht
  Fehlerklassen geprueft: `delivery.go`/`render.go`/`systemmail/sender.go` sind echte
  Implementierungen (kein Stub, kein Fake-Return), kein Gateway-Handler beruehrt (kein
  Direct-Svc-Bypass), keine Migration, kein Proto, keine neue Route. `cmd/berichte/main.go` verdrahtet
  Exporter immer und den Mailer nur bei gesetztem `SYSTEM_SMTP_HOST` — kein stiller Fake-Erfolg.
- PRAEMISSEN-KORREKTUR (wichtigster Befund dieser Iteration): die Unit ging davon aus,
  `dialer_agent_status_log` habe **keine** `tenant_id`-Spalte (Befund aus Iteration 54,
  Lauf 1/2, 2026-07-28). Das stimmt nicht mehr — tatsaechlich stimmte es schon am 28.07. nicht:
  `tenant_id UUID NOT NULL` + FK auf `tenants` + Index + `FORCE ROW LEVEL SECURITY` mit Policy
  `tenant_isolation USING (tenant_id = current_tenant_id() OR is_system_context())` existieren
  bereits seit Migration 000119/000120 (2026-05-10/11, Sprint 4 — zeitlich VOR Iteration 54).
  Verifiziert per `\d dialer_agent_status_log` gegen die lokale DB (Spalte, FK, Index, Policy alle
  vorhanden) und per `git log` auf die Migrationsdateien. Iteration 54 hat den Fund offenbar gegen
  die urspruengliche CREATE-TABLE-Migration 000067 geprueft statt gegen den tatsaechlichen
  Schema-Stand und den veralteten Befund unveraendert ins Backlog uebernommen — dort stand er seit
  drei Nachtlaeufen ungeprueft. Auch `LogStatusChange` (INSERT, setzt tenant_id ueber eine Subquery
  auf `users`) und `GetActiveAgentIDsForTenant` (explizites `WHERE tenant_id = $1`) sind bereits
  korrekt tenant-gescoped — keine dieser Stellen brauchte einen Fix.
- echter Rest-Fund: die Aktive-Kampagne-Subquery in `GetAgentStats` (`postgres_repository.go:503`)
  hatte trotz vorhandener Spalte kein eigenes `tenant_id`-Praedikat und verliess sich allein auf die
  RLS-GUC. Im normalen Request-Pfad bereits ausreichend geschuetzt (`PrepareConn` setzt
  `app.tenant_id` pro Connection-Checkout aus dem ctx), aber ohne Verteidigung gegen einen Aufruf mit
  einem falschen `tenantID`-Funktionsparameter unter System-Kontext (Worker o.ae.) — dort greift
  `is_system_context()` und die RLS-Policy laesst alles durch.
- gebaut: `AND l.tenant_id = $2` in der Aktive-Kampagne-Query ergaenzt, `tenantID` (bereits
  Funktionsparameter von `GetAgentStats`) als zweiten Query-Parameter durchgereicht. Kein
  Migrations-, Proto- oder Route-Bedarf, weil Spalte/RLS/INSERT/anderer-SELECT bereits vorhanden
  waren.
- Test: `TestGetAgentStats_ActiveCampaignTenantScoped` in `tenant_write_test.go` (gleiche Datei/
  gleiches Muster wie die bestehenden `..._LandInCallerTenant`-Tests). Ruft `GetAgentStats` unter
  System-Kontext (RLS komplett durchlaessig) einmal mit fremdem `tenantID` — erwartet
  `ActiveCampaignID == nil` — und einmal mit dem echten `tenantID` — erwartet die echte Kampagne.
  Falsifiziert: Fix testweise per `git stash` auf die Ausgangsversion zurueckgesetzt, Test wird rot
  (`foreign tenantID saw the active campaign: <uuid>`) — der Leak ist also real reproduzierbar ohne
  das Praedikat. Nach `git stash pop` wieder gruen.
- gate: build (`./internal/dialer/... ./internal/gateway/... ./cmd/dialer/... ./cmd/gateway/...`,
  `-p 2`) ok | vet ok | golangci-lint `./internal/dialer/... ./internal/gateway/...` **0 issues** |
  Tests mit `DATABASE_URL` (Rolle `kmuhub_app`): `go test -count=1 -v ./internal/dialer/...` —
  73 PASS, 0 Skip, 0 Fail. `go test ./internal/gateway/` nicht gefahren (keine Route beruehrt).
  Migration: keine (bereits vorhanden). RLS-Smoke: n.a. fuer Tabelle/Policy (unveraendert) — der
  neue DB-Test uebernimmt den Beweis fuer den Read-Pfad-Fix.
- offen:
  - Keine offenen Punkte fuer Luke aus dieser Unit selbst.
  - Hinweis fuers Backlog-Grooming: Befunde aus archivierten Journal-Eintraegen (hier: Iteration 54,
    Lauf 1/2) sollten vor Uebernahme in eine neue Unit gegen den AKTUELLEN Schema-/Code-Stand
    gegengeprueft werden, nicht nur zitiert — dieser lag hier ueber drei Nachtlaeufe unverifiziert im
    Backlog.

## Iteration 52 — g-einvoice-test-hygiene — done — 2026-08-02 05:15

- commit: (folgt direkt auf diesen Eintrag)
- verify vorgaenger: sauber. `324f7df0` (Iteration 51, dialer-agent-status-log-tenant) gepruefte
  Diffs (`postgres_repository.go`, `tenant_write_test.go`): kein Gateway-Handler beruehrt, kein
  Stub, kein `.proto`, kein neuer `RequirePermission`-Guard, keine neue Tabelle/Migration, keine
  Route, kein ersetzter Guard-Key — sauberer Ein-Praedikat-Fix plus Falsifikations-Test wie im
  Iteration-51-Eintrag beschrieben.
- PRAEMISSEN-KORREKTUR (Kernbefund dieser Iteration, analog zu Iteration 51): die Unit ging von
  einem offenen Bug aus (`TestTenantIsolation_IncomingInvoices` nutzt geteilte `TenantA`/`TenantB`,
  bricht jeden zweiten Lauf). Das stimmte bereits nicht mehr — Commit `d321f482`
  ("fix(einvoice): isolate the incoming-invoice RLS test with fresh tenants", 2026-08-01) hat genau
  diesen Kern schon behoben: der Test mint jetzt `tenantA, tenantB := uuid.New(), uuid.New()` statt
  der geteilten Konstanten. Verifiziert per `git log`/`git show d321f482` und durch zweimaligen
  echten Lauf gegen dieselbe lokale DB (siehe gate) — beide gruen, 0 Skips. Der ursprüngliche
  Befund (Iteration 42, Lauf 1/2, 2026-07-28 — archiviert) war zum Zeitpunkt seiner Entstehung
  korrekt, lag danach aber ungeprueft im Backlog, waehrend eine andere Iteration ihn bereits
  behoben hat.
- echter Rest-Fund: `done_when` verlangte zusaetzlich "Cleanup vorhanden" — das fehlte weiterhin.
  `TestTenantIsolation_IncomingInvoices` schloss den Pool nie (`pool.Close()`) und raeumte die
  importierte `finance_incoming_invoices`-Zeile nie auf. Durch die frischen Tenant-UUIDs war das
  keine Korrektheitsluecke mehr (kein Kollisionsrisiko, `UNIQUE(tenant_id, supplier_name,
  invoice_number)` greift nie doppelt), aber echte Hygiene-Schuld: 23 verwaiste Zeilen aus
  Vorlauf-Iterationen liegen noch in der lokalen DB (Tenants mit Namen "Tenant A EInvoice").
  Bewusst NICHT rueckwirkend bereinigt (lokale Dev-DB, keine Repo-Aenderung, kein Correctness-Impact) —
  fuer Luke unten vermerkt.
- gebaut: `t.Cleanup(pool.Close)` direkt nach `PoolFromEnv`, `t.Cleanup(func() {
  testutil.CleanupRow(t, pool, "finance_incoming_invoices", invA.ID) })` direkt nach dem Import.
  Kein Scope-Creep auf `roundtrip_outbound_test.go` (zweite DB-Datei im Paket): die nutzt bereits
  frische Tenants + `t.Cleanup(pool.Close)`, raeumt Zeilen nicht auf, aber ohne Kollisionsrisiko
  (fester Rechnungsname, aber pro Testlauf neuer Tenant) — identisches Muster zum etablierten
  `testutil.AssertWriteCarriesTenant`-Helper, der ebenfalls keine Tenant-Zeilen aufraeumt. Anfassen
  haette Konsistenz mit dem Rest des Backlogs gebrochen, ohne einen echten Bug zu schliessen.
- Falsifikation/Beweis: `go test -count=1 -v ./internal/biz/einvoice/...` zweimal hintereinander
  gegen dieselbe lokale DB gefahren (identischer `DATABASE_URL`) — beide gruen, `0` Skips
  (`TestTenantIsolation_IncomingInvoices` UND `TestRoundtrip_*` liefen real). Dritter isolierter
  Lauf nur dieses Tests bestaetigt zusaetzlich, dass die neue Cleanup-Zeile wirkt: Zeilenzahl in
  `finance_incoming_invoices` fuer Tenants mit dem Namenspraefix "Tenant A EInvoice" blieb bei 23
  vor und nach dem Lauf (kein Wachstum mehr).
- gate: build (`go build -p 2 ./internal/biz/einvoice/... ./internal/gateway/...`) ok | vet ok |
  golangci-lint `./internal/biz/einvoice/...` 0 issues | Tests mit `DATABASE_URL` (Rolle
  `kmuhub_app`): zwei volle Laeufe `go test -count=1 -v ./internal/biz/einvoice/...`, je 0 Skip,
  0 Fail. `go test ./internal/gateway/` nicht gefahren (keine Route beruehrt, reiner Testdatei-Fix).
  Migration: keine. RLS-Smoke: n.a. (keine Tabelle/Policy geaendert, nur Testcode).
- offen:
  - Fuer Luke: 23 verwaiste `finance_incoming_invoices`-Zeilen (Tenants "Tenant A EInvoice*") aus
    Vorlauf-Testlaeufen liegen noch in der lokalen Dev-DB — kosmetisch, optionales manuelles
    `DELETE`, kein Blocker.
  - Backlog-Grooming-Hinweis (wie schon in Iteration 51 notiert): Befunde aus dem archivierten
    Lauf-1/2-Journal vor Uebernahme in eine neue Unit gegen den AKTUELLEN Code-Stand pruefen statt
    nur zu zitieren.

## Iteration 53 — g-work-rest-tenant-writes — done — 2026-08-02 06:05

- commit: (folgt direkt auf diesen Eintrag)
- verify vorgaenger: sauber. `fb0bb7b1` (Iteration 52, einvoice-test-hygiene) gepruefter Diff
  (`tenant_isolation_test.go`): zwei `t.Cleanup`-Zeilen, kein Gateway-Handler, kein Stub, kein
  `.proto`, keine neue Tabelle/Migration, keine Route — reiner Testdatei-Fix wie im
  Iteration-52-Eintrag beschrieben.
- Auftrag: die vier work-Unterpakete calendar, meeting, resource, recording wurden im
  Tenant-Write-Sweep der Laeufe 1/2 nie geprueft — je eine `tenant_write_test.go` nachziehen, echte
  Luecken (Write ohne tenant_id, Read ohne Praedikat) fixen und per Falsifikation belegen.
- Rechercheergebnis vor dem Bauen: alle vier Kern-Repos (calendar, meeting, resource, recording)
  setzen `tenant_id` beim Insert korrekt und tragen — wo ueberhaupt ein `tenantID`-Parameter existiert
  — ein explizites `WHERE tenant_id = ...`-Praedikat auf GetByID/List/Update/Delete. `resource`s
  `ListBookings`/`ListBookingsByEvent` haben zwar kein eigenes Praedikat, werden aber im Service immer
  erst nach einem tenant-gescopten `GetByID` auf die `resourceID` aufgerufen — kein eigenstaendiger
  Angriffspfad. `recording`s Repo-Methoden (`GetRecording`, `UpdateRecording`, `DeleteRecording`, ...)
  nehmen bewusst gar kein `tenantID` entgegen und verlassen sich vollstaendig auf RLS ueber den ctx;
  der einzige System-Kontext-Aufrufer (`CleanupExpiredRecordings`, taegliches Cron in
  `cmd/work/main.go`) iteriert ausschliesslich ueber IDs aus seiner eigenen `ListExpiredRecordings`-
  Abfrage, nie ueber Caller-Input — kein Leseleck.
- echter Fund (Klasse "stiller Erfolg statt Fehler", nicht Klasse "Datenleck"): `calendar.Update`/
  `Delete` und `recording.UpdateRecording`/`DeleteRecording` riefen `pool.Exec` auf, warfen den
  `pgconn.CommandTag` aber weg und gaben nur den SQL-`err` zurueck. Filtert RLS die WHERE-Klausel
  eines Cross-Tenant-Writes auf 0 Zeilen, ist das fuer Postgres kein Fehler — `err` bleibt `nil`, der
  Aufrufer haelt den Write faelschlich fuer erfolgreich. Jeder Nachbar-Repo in diesem Sweep
  (`project`, `resource`, `meeting`, s. `tenant_write_test.go`-Vorlagen) prueft `tag.RowsAffected() ==
  0` und gibt sein `ErrXNotFound` zurueck — nur `calendar.Update`/`Delete` (die beiden sicherheits-
  kritischsten Methoden der Kern-Entitaet) und `recording.UpdateRecording`/`DeleteRecording` fehlten
  in diesem Muster. Im normalen Service-Flow ungefaehrlich (`calendar.Service.Update`/`Delete` und
  praktisch alle `recording.Service`-Methoden rufen vorher ein tenant-gescoptes `GetByID`/
  `GetRecording` unter demselben ctx auf, das einen fremden Datensatz schon davor mit `ErrNotFound`
  abfaengt) — aber genau die Defense-in-Depth-Luecke, die dieser Sweep laut Auftrag schliessen soll:
  ein direkter Repo-Aufruf unter falscher/veralteter Tenant-Annahme waere ein stiller No-op.
- gebaut: `tag, err := r.pool.Exec(...)` + `if tag.RowsAffected() == 0 { return ErrCalendarNotFound }`
  in `calendar/postgres_repository.go` (`Update`, `Delete`); analog `ErrNotFound` in
  `recording/postgres_repository.go` (`UpdateRecording`, `DeleteRecording`). Vier neue
  `tenant_write_test.go` (calendar, meeting, resource, recording) nach dem etablierten
  Create/Update/Delete-Handrolled-Muster (`project`/`timeentry` als Vorlage, nicht der duennere
  `AssertWriteCarriesTenant`-Helper, weil Update/Delete mitgeprueft werden sollen). meeting und
  resource: reine Abdeckung, kein Fund (beide hatten das RowsAffected-Muster schon). recording:
  da die Repo-Signaturen kein `tenantID` fuehren, laeuft der Foreign-Ctx-Nachweis komplett ueber RLS
  (`WithTenantCtx` + `testutil.AssertRowCount`), nicht ueber einen expliziten Parameter — bewusst so
  belassen (Scope-Erweiterung auf "recording-Repo bekommt jetzt ueberall tenantID-Parameter" waere
  eine API-Aenderung ueber alle Aufrufer hinweg gewesen, nicht der Auftrag dieser Unit).
- Falsifikation: beide Fixes per `git stash` auf den Ausgangsstand zurueckgesetzt, gezielt
  `TestCalendarWrites_LandInCallerTenant` und `TestRecordingWrites_LandInCallerTenant` gefahren —
  beide werden rot exakt an der erwarteten Stelle (`Update (foreign ctx): expected an error, got nil`
  bzw. `UpdateRecording (foreign ctx): expected an error, got nil`). Nach `git stash pop` wieder gruen.
- gate: `go build -p 2 ./...` (gesamtes Backend) ok | `go vet ./internal/work/...` ok |
  `golangci-lint run` auf calendar/meeting/resource/recording: 0 issues | Tests mit `DATABASE_URL`
  (Rolle `kmuhub_app`): `go test -count=1 ./internal/work/...` zweimal hintereinander gegen dieselbe
  lokale DB — beide Laeufe alle 17 Unterpakete PASS, 0 Skip, 0 Fail. Zusaetzlich
  `go test -count=1 ./internal/gateway/... ./internal/server/...` als Regressionsschutz (Fehlerpfade
  von Update/Delete geaendert) — beide PASS. Migration: keine (kein Schema-/RLS-Wechsel). RLS-Smoke:
  n.a. fuer Policy-Aenderung (unveraendert) — die vier neuen Tests uebernehmen den Beweis fuer den
  Write-Pfad-Fix.
- offen:
  - Kleinere Geschwister-Methoden mit demselben "kein RowsAffected-Check"-Muster bewusst NICHT
    mitgefixt (Scope-Disziplin): `calendar.UpdateMemberPermission/-Visibility/-ColorOverride`
    (Nebenfelder auf `calendar_members`, nicht die Kern-Entitaet). Fuer eine kuenftige Unit vormerken,
    falls dort mal ein echter Cross-Tenant-Pfad noetig wird — aktuell werden sie ausschliesslich nach
    `GetMember`/Permission-Check aufgerufen.
  - Kein modules.*-Flag, kein neues RequirePermission, keine Migration, keine Route beruehrt.

## Iteration 54 — g-inbox-openapi-shape-drift — done — 2026-08-02 06:12

- commit: (folgt direkt auf diesen Eintrag)
- verify vorgaenger: sauber. `9b24551b` (Iteration 53, work-tenant-write-sweep) gepruefter Diff
  (`calendar/postgres_repository.go`, `recording/postgres_repository.go` + vier neue
  `tenant_write_test.go`): kein `.proto`, keine neue Route, kein neuer `RequirePermission`-Guard,
  keine neue Tabelle. Der `RowsAffected`-Fix ist konsistent mit dem Schwester-Muster
  (project/resource/meeting), und alle produktiven Aufrufer holen den Datensatz vorher per
  tenant-gescoptem `GetByID`/`GetRecording` unter demselben ctx — kein Regressionsrisiko.
- Auftrag: die neun Single-Message-Mutations-Endpoints der Inbox (`/read`, `/unread`, `/star`,
  `/status`, `/archive`, `/unarchive`, `/snooze`, `/unsnooze`, `/assign`) antworten real mit
  `{"message": {...InboxMessageInfo}}` (verifiziert gegen `inboxv1.MarkReadResponse` etc. in
  `proto/inbox/v1/inbox.proto` — jede der neun Response-Messages traegt exakt ein Feld
  `InboxMessageInfo message = 1`, und jeder Handler in `route_inbox.go` liefert das per
  `response.Proto(w, http.StatusOK, resp)` unveraendert durch). `openapi.yaml` dokumentierte fuer
  alle neun stattdessen ein nacktes `InboxMessage`-Schema — Drift aus Nachtlauf 1, vorbestehend.
- FE-Gegenpruefung (Notenauftrag): `desktop/src/renderer/src/api/inbox-client.ts` typt
  `markRead`/`markUnread`/`toggleStar`/`archiveMessage`/`unarchiveMessage`/`snoozeMessage`/
  `unsnoozeMessage`/`assignMessage` durchgehend als `Promise<void>` — der Response-Body wird von
  keinem dieser Aufrufer gelesen. Fuer `/status` existiert im FE ueberhaupt kein Aufrufer. Die reale
  Form widerspricht also keinem FE-Konsumenten; die Spec darf gefahrlos dem Code folgen (Auftrag:
  "die SPEC folgt dem Code, nicht umgekehrt").
- gebaut: neues wiederverwendbares Schema `InboxMessageWrapper` (`{message: InboxMessage}`) direkt
  unter `InboxMessage` in `openapi.yaml` eingefuegt; alle neun `200`-Responses der genannten
  Endpoints von `$ref: InboxMessage` auf `$ref: InboxMessageWrapper` umgestellt. Namensmuster folgt
  dem bereits im Spec vorhandenen `InboxCannedResponseWrapper`/`EmailRuleEnvelope`-Konventionspaar,
  keine neue Konvention erfunden.
- offen (bewusst NICHT mitgefixt, Scope-Disziplin): `GET /api/v1/inbox/messages/{id}` zeigt exakt
  denselben Drift (Handler `HandleGetMessage` liefert ebenfalls die `{message: {...}}`-Form,
  Spec dokumentiert nackt `InboxMessage`) — bestaetigt durch den Kommentar in `inbox-client.ts:104`
  ("GET /messages/{id} wraps the message"). War nicht Teil der neun in dieser Unit benannten
  Endpoints; fuer eine Folge-Unit vormerken, falls die Liste unvollstaendig war.
  `/tags` (add/remove), `/forward`, `/reply` waren ebenfalls nicht in der Neun-Liste und blieben
  unangetastet (letztere zwei sind ohnehin schon korrekt als eigenes `{success: bool}`-Schema
  dokumentiert, nicht als `InboxMessage`).
- gate: build (`go build -p 2 ./internal/gateway/... ./cmd/gateway/...`) ok | vet ok |
  golangci-lint `./internal/gateway/...` 0 issues | `go test -count=1 -v ./internal/gateway/
  -run TestOpenAPIRouteDrift` PASS (771 registrierte Routen gegen 773 dokumentierte Pfade) |
  `go test -count=1 ./internal/gateway/` voll PASS, 0 Skip (per `-v | grep -c SKIP` gegengeprueft).
  Migration: keine. RLS-Smoke: n.a. (reine Spec-Datei, keine Tabelle/Policy/Route-Code beruehrt).

## Iteration 55 — g-featureflag-cleanup — done — 2026-08-02 05:41

- commit: (folgt direkt auf diesen Eintrag)
- verify vorgaenger: sauber. `4b2b9c9a` (Iteration 54, inbox-openapi-shape-drift) gepruefter Diff
  (reine `openapi.yaml`-Aenderung, neun `$ref`-Umstellungen auf `InboxMessageWrapper`): kein Code,
  keine Route, keine Migration. `go test -count=1 ./internal/gateway/ -run TestOpenAPIRouteDrift`
  PASS vor Beginn dieser Unit gegengeprueft.
- Auftrag: zwei Inkonsistenzen in `internal/featureflag/registry.go` aufraeumen —
  (a) `integrations.bexio` wird registriert, aber nirgends per `IsEnabled` abgefragt,
  (b) `plugins.wasm` ist ein toter Flag (Gating passiert ueber den Build-Tag `no_wasm`).
- Befund (a): `grep -rn "integrations.bexio\|COSMI_INTEGRATION_BEXIO_ENABLED"` traf ausser
  `registry.go`/`registry_test.go`/`route_feature_flags_test.go` nur Doku (`.knowledge/`,
  `.planning/`), keinen einzigen Aufrufer. `route_bexio.go` registriert seine Routen bedingungslos
  (kein `flags`-Feld auf `BexioRoutes` — anders als z.B. `BerichteRoutes`), FE-seitig kein
  Feature-Flag-Gate auf der Bexio-Integrationskarte. `.knowledge/integrationen.md` bestaetigt:
  Bexio ist Prod-live (`GET /integrations/bexio/status` → 401, Route live). Entscheidung: NICHT
  scharfschalten. Ein neu enforcter Flag mit `DefaultEnabled: false` haette die live Bexio-Sync
  beim naechsten Deploy stillgelegt, ausser die zugehoerige Prod-Umgebungsvariable ist dort bereits
  gesetzt — das kann von hier aus nicht verifiziert werden (kein Prod-Zugriff im Loop) und waere
  derselbe Deploy-Hazard wie eine neue `config.RequireX`-Assertion, auch wenn es formal kein
  `modules.*`-Flag ist. Flag stattdessen entfernt (`registry.go`), Test-Erwartungen nachgezogen:
  `registry_test.go` `expectedKeys`, `route_feature_flags_test.go` `wantCount` 18 → 17 samt
  Kommentar.
- Befund (b): `plugins.wasm` wird nirgends per `IsEnabled` abgefragt — bestaetigt per grep. Die
  eigentliche Absicherung ist `-tags no_wasm` im `make build-prod`-Target
  (`internal/plugin/wasm/runtime_disabled.go`, Stub mit `NewRuntime` → nil). ABER: der Flag ist
  nicht komplett bedeutungslos, sondern die zweite Verteidigungslinie fuer genau den Fall, dass
  jemand die Plugin-HTTP-API testweise einschaltet (`plugins.api=true`, laut Code-Kommentar in
  `cmd/gateway/main.go:310` explizit fuer Dev vorgesehen) OHNE den `no_wasm`-Tag zu setzen — dann
  waere `HandleCreateManifest` bislang der einzige Ort, der ein `plugin_type=wasm`-Manifest je zu
  Gesicht bekommt, und der hat den Flag nie gefragt. Zusatzfund dabei: der komplette WASM-Hook-
  Ausfuehrungspfad (`hook.NewDispatcher`) ist in KEINEM `cmd/*`-Binary verdrahtet (grep leer) —
  das ist ein eigener, viel groesserer Architektur-Gap (Plugin-Service kennt den Dispatcher gar
  nicht), der bewusst NICHT Teil dieser Aufraeum-Unit ist. Entscheidung: Flag am einzigen Ort
  enforcen, der ihn ueberhaupt lesen kann, ohne neue Cross-Service-Verdrahtung zu bauen — dem
  Gateway, der den `featureflag.Registry` bereits haelt (`cmd/plugin` & Co. kennen ihn bislang gar
  nicht, `grep -rl featureflag ./cmd` traf nur `cmd/gateway`). `PluginRoutes` bekommt ein
  `flags *featureflag.Registry`-Feld (Konstruktor-Signatur wie bei `BerichteRoutes` erweitert),
  `HandleCreateManifest` lehnt `plugin_type=wasm` mit 400 ab, solange `plugins.wasm=false` ist.
  Kein Risiko fuer Bestandsdaten: WASM ist projektweit "OFF bis Phase D", es existiert kein
  legitimer Prod-Anwendungsfall, den das blockieren wuerde.
- gebaut:
  - `internal/featureflag/registry.go`: `integrations.bexio`-Registrierung entfernt.
  - `internal/gateway/route_plugin.go`: `flags *featureflag.Registry` auf `PluginRoutes`,
    `NewPluginRoutes(registry, flags)`, Gate in `HandleCreateManifest` vor dem gRPC-Call.
  - `cmd/gateway/main.go`: Call-Site auf `NewPluginRoutes(registry, flagRegistry)` nachgezogen.
  - `openapi.yaml`: `wasm_binary`-Beschreibung ("inert" war seit diesem Commit falsch) und
    `plugin_type`-Beschreibung (400 bei `plugins.wasm=false`) aktualisiert.
  - Tests: neue `internal/gateway/route_plugin_test.go` (3 Faelle: wasm+flag-off → 400 mit
    "plugins.wasm" im Body, wasm+flag-on → kein 400, config+flag-off → kein 400).
    `internal/gateway/testutil_test.go`: neuer Helper `noFlags()` fuer Tests, die `PluginRoutes`
    nur als Dependency brauchen, ohne Flag-Verhalten zu pruefen. Sieben Call-Sites in
    `tenant_isolation_test.go` und eine in `openapi_drift_test.go` auf die neue Signatur
    nachgezogen. `registry_test.go`/`route_feature_flags_test.go` Flag-Zahl 18 → 17.
- Falsifikation: die neue WASM-Gate-Bedingung testweise mit `if false && ...` deaktiviert,
  `TestHandleCreateManifest_WASM_RejectedWhenFlagOff` gefahren — wird rot exakt an der erwarteten
  Stelle (503 statt 400, Body ohne "plugins.wasm"). Fix zurueckgesetzt, wieder gruen.
- gate: `go build -p 2 ./...` ok | `go build -tags no_wasm ./cmd/gateway/...`
  (Produktions-Build-Tag) ok | `go vet ./internal/gateway/... ./internal/featureflag/...
  ./cmd/gateway/...` ok | `golangci-lint run` auf denselben Paketen: 0 issues |
  `go test -count=1 ./internal/gateway/... ./internal/featureflag/...` PASS, 0 Skip/0 Fail
  (per `-v | grep -c SKIP`/`FAIL` gegengeprueft) | `TestOpenAPIRouteDrift` weiterhin PASS (771
  registrierte Routen, unveraendert — keine Route hinzugefuegt/entfernt, nur Request-Body-Felder
  praezisiert). Migration: keine. RLS-Smoke: n.a. (kein Tabellen-/Policy-Wechsel). Kein
  `modules.*`-Flag scharfgeschaltet, kein `config.RequireX` hinzugefuegt.
- offen:
  - `hook.NewDispatcher` (WASM-Hook-Ausfuehrung) ist in keinem `cmd/*`-Binary verdrahtet —
    eigenstaendiger Architektur-Gap, fuer eine kuenftige Phase-D-Unit vormerken, falls WASM-Plugins
    tatsaechlich ausgefuehrt werden sollen.
  - `integrations.bexio` bleibt unenforced entfernt — falls Luke die Bexio-Integration doch
    hinter einen Flag stellen will, braucht das zuerst eine Pruefung der Prod-Umgebungsvariable
    (Prod-Zugriff, ausserhalb Loop).

## Iteration 56 — g-admin-branding-s3 — done — 2026-08-02 06:00

- commit: (folgt direkt auf diesen Eintrag)
- verify vorgaenger: sauber. `9a6198d8` (Iteration 55, featureflag-cleanup) `git show --stat`
  gepruefter Diff deckt sich mit dem Journal-Eintrag (registry.go, route_plugin.go, main.go,
  openapi.yaml, neue Tests) — keine Business-Logik im Handler, kein direkter Service-Bypass, kein
  neuer `RequirePermission`-Guard ohne Seed, openapi.yaml im selben Commit. Kein Nacharbeitsbedarf.
- Auftrag: Workspace-Branding (Name, Logo, Icon, Akzentfarbe) von "nie im Backlog gewesen" auf
  echte Backend-Persistenz + MinIO-Logo-Upload heben (`.planning/backend-gaps.md` §A-4/admin).
  FE persistiert bisher nur in `localStorage` (`cosmi:brand:*`,
  `desktop/.../admin/branding/BrandingAdminHubTab.tsx`) — das FE-Wiring selbst ist NICHT Teil
  dieses Backend-Loops (kein FE-Code angefasst).
- Architektur-Entscheidung (Begruendung siehe done_when-Kriterien): **kein neuer `internal/admin`-
  Service, keine neue Tabelle, keine neue Migration.** Branding-Metadaten laufen ueber die
  bestehende `tenant_settings`-Infrastruktur (`internal/settings`, Migration 000138, RLS bereits
  seit 000218 aktiv und durch `TestSettingsWrites_LandInCallerTenant` bewiesen) unter
  `module_id="branding"` — vier Keys `name`/`logoObjectKey`/`iconObjectKey`/`accentColor`. Zwei
  neue duenne RPCs `GetBranding`/`PutBranding` im bestehenden `settings.proto`/`settings`-Service
  (selbes Binary wie auth, Port :50051) kapseln Validierung und Wire-Contract, delegieren die
  eigentliche Persistenz aber vollstaendig an `s.repo.GetTenantSettings`/`s.PutTenantSettings` —
  dadurch erbt PutBranding automatisch die bestehende RBAC-Pruefung (admin ODER module-lead fuer
  "branding"; da niemand je Modul-Leiter fuer ein FE-fremdes Modul wird, ist das de-facto
  admin-only, deckungsgleich mit dem FE-Capability-Gate `admin:branding:manage`) UND die bereits
  RLS-bewiesene Tenant-Isolation — kein zusaetzlicher DB-Test noetig, die Schreibpfade sind
  identisch zu denen, die `tenant_write_test.go` schon gegen echte RLS faehrt. Logo/Icon werden als
  MinIO-**Objektschluessel** gespeichert, nie als URL (gleiches Muster wie `User.avatar_url`) —
  Aufloesung zur Downloadable-URL passiert client-seitig ueber den bestehenden
  `POST /api/v1/files/presign-download`.
  Upload laeuft ueber die bereits vorhandene generische Presign-Schicht
  (`internal/document/file/presign.go`) statt eines eigenen Upload-Endpoints: neuer Scope
  `"branding"` in `allowedPresignScopes`. Objektschluessel-Form `{tenant_id}/branding/{uuid}{ext}`
  ist damit automatisch tenant-gescoped (Downloadpfad prueft bereits den `{tenant_id}/`-Praefix).
- Typ-/Groessenbegrenzung + SVG-Entscheidung (done_when-Punkt): SVG **bewusst ausgeschlossen** —
  kann `<script>`/Event-Handler tragen, es gibt keine Sanitizer-Stufe vor der Auslieferung an
  andere Tenant-User. `brandingAllowedContentTypes` erlaubt nur `image/png`/`image/jpeg`/
  `image/webp`, `lean:`-Marker mit Upgrade-Trigger "SVG erlauben, sobald ein Sanitizer vor dem
  Upload sitzt". Zusaetzlich eigener, engerer Groessendeckel `brandingMaxSizeBytes = 2 MB` (statt
  des generischen 50-MB-Limits) — Logo/Icon rendert inline im App-Chrome, ist kein Dokument.
  Server-seitige Validierung in `PutBranding`, nicht nur clientseitig: `accentColor` muss exakt
  einer der 10 Cosmi-Swatch-Werte sein (`desktop/.../lib/swatch-colors.ts`, Liste per Hand
  gespiegelt — kein Codegen-Link, Palette ist klein und aendert sich selten), `name` <= 200
  Zeichen, `logoObjectKey`/`iconObjectKey` (falls gesetzt) muessen mit `{tenant_id}/branding/`
  praefixiert sein — verhindert, dass ein Client einen fremden oder scope-fremden Objektschluessel
  unterschiebt (eigener Testfall: `TestPutBranding_RejectsObjectKeyFromDifferentScope` fuer genau
  diesen Fall, `avatar`-Scope-Key auf `branding` PUT versucht). PUT ist Full-Replace, kein Patch —
  ausgelassenes Logo/Icon loescht es (Testfall `TestPutBranding_FullReplaceClearsPreviousLogo`).
- gebaut:
  - `proto/settings/v1/settings.proto`: `Branding`-Message + `GetBranding`/`PutBranding`-RPCs samt
    Request/Response-Messages, im selben Commit regeneriert (`protoc` direkt, `make` war im
    Bash-Tool nicht verfuegbar — Kommando 1:1 aus dem `proto-settings`-Makefile-Target uebernommen).
  - `internal/settings/branding.go` (neu): `Branding`-Domaintyp, `GetBranding`/`PutBranding` auf
    `*Service`, Validierung (Akzentfarbpalette, Namenslaenge, Objektschluessel-Praefix), vier neue
    Fehlerwerte.
  - `internal/server/settings_grpc.go`: `GetBranding`/`PutBranding`-RPC-Handler +
    `brandingToProto`-Mapper, Fehler-Mapping (ungueltige Farbe/Objektschluessel/Name -> 400,
    `ErrNotModuleLead` -> 403).
  - `internal/gateway/route_settings.go`: `GET`/`PUT /api/v1/admin/branding`, Guard
    `RequirePermission("settings","read"/"write")` wiederverwendet (kein neuer Permission-Key, also
    keine Seed-Migration noetig — deckt sich mit den bestehenden `/settings/{module_id}/tenant`-
    Routen). `response.Proto` (snake_case) wie die Nachbar-Settings-Routen.
  - `internal/document/file/presign.go`: Scope `"branding"` + `brandingAllowedContentTypes` +
    `brandingMaxSizeBytes`, Validierung in `GetPresignedUploadURL` verdrahtet.
  - `backend/api/openapi.yaml`: `/api/v1/admin/branding` (GET+PUT) + `Branding`-Schema, Stil von
    `/api/v1/admin/subscription` abgeschaut.
  - Tests: `internal/settings/branding_test.go` (14 Faelle: Default vor erstem Write, Roundtrip,
    Full-Replace-Clear, Admin/Module-Lead/Non-Lead-RBAC identisch zu `PutTenantSettings` getestet,
    ungueltige Akzentfarbe/fehlende Akzentfarbe/zu langer Name/fremder Tenant-Objektschluessel/
    scope-fremder Objektschluessel/leere Objektschluessel erlaubt, Cross-Tenant-Isolation).
    `internal/document/file/presign_test.go`: 6 neue Faelle (branding erlaubt png/jpeg/webp,
    lehnt SVG + Nicht-Bild-Typ ab, eigener Groessendeckel bei Max und ueber Max).
- gate: `go build -p 2 ./...` ok | `go build -tags no_wasm ./cmd/gateway/... ./cmd/auth/...`
  (Produktions-Build-Tag) ok | `go vet ./...` ok | `golangci-lint run` auf
  `internal/settings/... internal/server/... internal/gateway/... internal/document/...
  proto/settings/...`: 0 issues | `gofmt -l` auf allen touched Files: clean (drei Dateien
  brauchten `gofmt -w`, danach clean) |
  `DATABASE_URL=postgres://kmuhub_app:...@localhost:5432/kmuhub go test -count=1
  ./internal/settings/... ./internal/document/... ./internal/gateway/... ./internal/server/...`
  PASS, `kmuhub_app` (NOSUPERUSER NOBYPASSRLS) — inkl. `TestSettingsWrites_LandInCallerTenant`
  (RLS-Beweis fuer `tenant_settings`, auf dem `Branding` aufsetzt, lief mit echter DB durch, nicht
  nur mit `fakeRepo`) | `go test -count=1 -v ./internal/gateway/ -run TestOpenAPIRouteDrift` PASS
  (772 registrierte Routen gegen 774 dokumentierte Pfade, +1/+1 durch den neuen
  `/api/v1/admin/branding`-Pfad mit zwei Methoden). Migration: keine (bewusst, siehe
  Architektur-Entscheidung oben). RLS-Smoke: n.a. im engeren Sinn — kein neues Schema, aber die
  Schreib-/Lesepfade sind identisch zu den bereits RLS-bewiesenen `tenant_settings`-Pfaden und
  liefen in diesem Lauf gegen echte DB durch. Kein `modules.*`-Flag scharfgeschaltet, kein
  `config.RequireX` hinzugefuegt, kein neuer `RequirePermission`-Key (also auch kein Seed-Bedarf).
- offen:
  - FE-Wiring (echter Upload-Flow ueber `presign-upload`, `useBranding`-Hook, Topbar/Sidebar an
    das gespeicherte Branding statt an das statische Cosmi-Branding anschliessen) ist bewusst nicht
    Teil dieser Unit — reine Backend-Iteration, kein `desktop/`-Code angefasst.
  - Akzentfarbpalette ist von Hand zwischen `desktop/.../lib/swatch-colors.ts` und
    `internal/settings/branding.go` gespiegelt (kein Codegen-Link wie beim RBAC-Katalog). Faellt
    aendert sich die Palette, muss `allowedAccentColors` von Hand nachgezogen werden — bei 10
    Werten bewusst kein Generator gebaut (Lean Code), aber im Auge behalten falls die FE-Palette
    kuenftig haeufiger wechselt.

## Iteration 57 — g-dokumente-comments — done — 2026-08-02 06:20

- commit: (folgt direkt auf diesen Eintrag)
- verify vorgaenger: sauber. `8e21c84b` (Iteration 56, g-admin-branding-s3) `git show --stat`
  gepruefter Diff deckt sich mit dem Journal-Eintrag (settings.proto + Regen im selben Commit,
  `route_settings.go` ruft `sr.getSettingsClient()` statt einer direkten Service-Instanz,
  `settings_grpc.go` hat echte `GetBranding`/`PutBranding`-Implementierungen statt Unimplemented,
  keine neue Tabelle/Migration, kein neuer `RequirePermission`-Key also kein Seed-Bedarf,
  openapi.yaml im selben Commit mit +96 Zeilen). Kein Nacharbeitsbedarf.
- Auftrag: Kommentare an Dokumenten (`backend-gaps.md` §dokumente, "Datei-Kommentare:
  Comment-Tabelle + Endpoints"), erste offene Unit ohne Blocker (`g-dokumente-comments`,
  `deps: []`).
- Vorbild: `internal/work/comment` (task_comments) fuer Service-/Repo-Schnitt, ABER bewusst NICHT
  dessen Autorisierungsmuster uebernommen — siehe Nebenfund unten.
- Design-Entscheidungen:
  - Autor beim Create kommt aus `actorIDFromContext(ctx)` im grpc-Handler (liest
    `middleware.GetUserID(ctx)`), nicht aus dem Request-Body — `x-user-id` propagiert ueber den
    internen gRPC-Hop (`internal/middleware/grpc_tenant.go`), also funktioniert das zuverlaessig.
  - Loeschrecht (Autor ODER Admin) kann NICHT genauso geloest werden: Rollen (fuer `IsAdmin`)
    propagieren ueber den internen gRPC-Hop NICHT, nur `x-tenant-id`/`x-user-id`. Deshalb
    `is_admin` als explizites Proto-Feld auf `DeleteFileCommentRequest`, von der GATEWAY aus
    `middleware.IsAdmin(r.Context())` befuellt (das ist der einzige Ort, an dem die Rolle bekannt
    ist) — service-seitiger Check bleibt `AuthorID == actorID || isAdmin`.
  - Sanitization: kein neuer Sanitizer (kein bluemonday im Modulgraph, Lean-Regel "vorhandene
    Dependency nutzen"). Trim + 10 000-Zeichen-Limit server-seitig, XSS-Schutz bleibt client-seitig
    DOMPurify — exakt das Muster, das `internal/work/comment` bereits im Bestand faehrt.
    `lean:`-Kommentar in `service.go` mit Upgrade-Trigger "non-React-Consumer (E-Mail-Digest,
    Export) rendert Kommentarinhalt als HTML".
  - Wire-Shape: alle vier Endpoints serialisieren einheitlich ueber `response.Proto`/
    `response.ProtoListWrapped` (protojson) statt des alten `response.JSON`-Ad-hoc-Envelopes, den
    `route_document.go` fuer `DocumentFile`/`DocumentFolder` wegen `file_size int64` noch braucht
    — `DocumentFileComment` hat kein int64-Feld, also `created_at`/`updated_at` durchgehend als
    echte RFC3339-Strings (kein `{seconds,nanos}`), gleiches Muster wie das juengste Nachbarfeature
    `ListFileActivity` (Migration 000264).
  - Keine neuen Permission-Keys: die Routen haengen an den schon vorhandenen additiven
    `docRead`/`docEdit`-Guards (`documents`/`documents:file` read/write) aus
    `route_document.go` — kein Katalog-Key `documents:comment:*` im FE, kein Seed-Bedarf.
- gebaut:
  - Migration `000265_document_file_comments` (`.up.sql`/`.down.sql`): Tabelle
    `document_file_comments` (`tenant_id UUID NOT NULL`, FK auf `document_files`/`users`/`tenants`),
    zwei Indizes, `CALL enable_tenant_rls('document_file_comments')`.
  - `internal/models/document.go`: `DocumentFileComment`-Struct.
  - `internal/document/file/{repository.go,errors.go,postgres_repository.go,service.go}`:
    5 neue Repository-Methoden (Create/Get/List/Update/Delete), 5 neue Sentinel-Errors,
    4 neue Service-Methoden (`CreateComment`/`UpdateComment`/`DeleteComment`/`ListComments`) mit
    Autor-/Admin-Pruefung und Content-Validierung.
  - `proto/document/v1/document.proto` + Regen (`protoc` direkt, `make` im Bash-Tool nicht
    verfuegbar — Kommando 1:1 aus dem `proto-document`-Makefile-Target): `DocumentFileComment`-
    Message, 4 neue RPCs (`ListFileComments`/`CreateFileComment`/`UpdateFileComment`/
    `DeleteFileComment`) samt Request/Response-Messages.
  - `internal/server/document_grpc.go`: 4 neue RPC-Handler, `toProtoFileComment`-Mapper,
    5 neue `mapDocumentError`-Faelle (NotFound/InvalidArgument/PermissionDenied).
  - `internal/gateway/route_document.go`: `GET/POST /documents/files/{id}/comments`,
    `PUT/DELETE /documents/comments/{id}` (letztere zwei standalone by ID, analog zum
    bestehenden `/documents/links/{id}`-Muster) mit `docRead`/`docEdit`.
  - `backend/api/openapi.yaml`: `DocumentFileComment`-Schema + 4 Pfad-Eintraege (2 Operationen auf
    `/documents/files/{id}/comments`, 2 auf `/documents/comments/{id}`), Stil von
    `/documents/files/{id}/activity` abgeschaut.
  - Tests: `internal/document/file/service_test.go` (11 neue Faelle: Create Success/trim,
    Content-Required, Content-zu-lang, Cross-Tenant-FileNotFound, Update Autor-darf/Nicht-Autor-
    gesperrt, Delete Autor-darf/Nicht-Autor-Nicht-Admin-gesperrt/Admin-darf-fremde, List
    Tenant-Isolation) + neue `MockRepository`-Methoden.
    `internal/document/file/postgres_repository_comment_test.go` (neu, 4 DB-Tests: Create+List,
    List-Tenant-Isolation, Update inkl. Fremd-Tenant-Ablehnung, Delete inkl. Fremd-Tenant-Ablehnung)
    nach Vorbild `postgres_repository_entity_link_test.go`, wiederverwendet `seedActivityFixture`.
- Nebenfund (nicht in dieser Unit gefixt, eigene Unit `fix-g-work-task-comment-authz` angelegt):
  `WorkGRPCServer.UpdateTaskComment`/`DeleteTaskComment` (`internal/server/work_grpc.go:1028-1055`)
  reichen den Actor falsch durch — `UpdateTaskComment` ruft den Service mit `actorID=uuid.Nil` auf,
  wodurch der Autor-Vergleich (`comment.AuthorID != actorID`) praktisch immer wahr ist und JEDES
  Update mit `ErrCannotEditOthersComment` fehlschlaegt (Feature de-facto kaputt). `DeleteTaskComment`
  ruft mit `isAdmin=true` hardcoded auf — das ist ein kompletter Bypass des Autor-Checks: JEDER
  User mit `work:task_comment:delete` kann JEDEN fremden Kommentar loeschen. Root Cause: Rollen
  propagieren nicht ueber den internen gRPC-Hop (nur `x-tenant-id`/`x-user-id`), und die Gateway-
  Handler dort loesen weder Autor noch Admin auf, bevor sie durchreichen. Das in dieser Unit neu
  gebaute Muster (`actorIDFromContext` fuer den Autor, explizites `is_admin`-Feld von der Gateway
  fuer den Admin-Override) ist der Fix, 1:1 uebertragbar.
- gate: `go build -p 2 ./internal/document/... ./internal/gateway/... ./internal/server/...
  ./proto/document/... ./cmd/document/... ./cmd/gateway/...` ok | `go vet` auf denselben Paketen ok
  | `golangci-lint run --config .golangci.yml` auf `internal/document/... internal/gateway/...
  internal/server/...`: 0 issues |
  `DATABASE_URL=postgres://kmuhub_app:app_dev@localhost:5432/kmuhub?sslmode=disable go test
  -count=1 ./internal/document/...`: PASS, 4 neue DB-Tests liefen real (verifiziert per `-v -run
  Comment`, keine Skips), `kmuhub_app` (NOSUPERUSER NOBYPASSRLS) | `go test -count=1
  ./internal/server/...` PASS | `go test -count=1 ./internal/gateway/` PASS inkl.
  `TestOpenAPIRouteDrift` (774 Routen gegen 776 dokumentierte Pfade, +2/+2 durch die neuen
  Comment-Pfade). Migration lokal angewendet (`migrate ... up` → Kopf 265), RLS-Smoke gegen
  `document_file_comments`: eigener Tenant → 1, fremder Tenant → 0 (bestanden, kein Zwei-Nullen-Fall).
- offen:
  - `fix-g-work-task-comment-authz` (neu angelegt) wartet auf eine kuenftige Iteration.
  - Kein FE-Wiring (kein `desktop/`-Code angefasst) — reine Backend-Iteration, wie im Auftrag der
    Unit vorgesehen (`backend-gaps.md` listet nur "Comment-Tabelle + Endpoints", keine FE-UI).

## Iteration 58 — fix-g-work-task-comment-authz — done — 2026-08-02 06:30

- commit: (folgt direkt auf diesen Eintrag)
- verify vorgaenger: sauber. `65f5918f` (Iteration 57, g-dokumente-comments) `git show --stat`
  gepruefter Diff deckt sich mit dem Journal-Eintrag: `route_document.go` geht durchgehend ueber
  `getDocumentClient()`/`client.<RPC>`, kein direkter Service-Zugriff; `document.proto` +
  `.pb.go`/`_grpc.pb.go` im selben Commit regeneriert; Migration 000265 hat `tenant_id UUID NOT NULL`
  + `CALL enable_tenant_rls('document_file_comments')` verifiziert; `document_grpc.go` nutzt fuer
  Create/Update/Delete durchgehend `actorIDFromContext(ctx)`, nie `uuid.Nil`; `DeleteFileComment`
  bekommt `IsAdmin` explizit von der Gateway (`middleware.IsAdmin(r.Context())`), kein
  Autor-Check-Bypass; keine neuen `RequirePermission`-Keys, also kein Seed-Bedarf; kein neuer
  `/api/v1/*`-Pfad ohne openapi.yaml-Eintrag (`TestOpenAPIRouteDrift` bestand mit +2/+2). Kein
  Nacharbeitsbedarf.
- Auftrag: Nebenfund aus `g-dokumente-comments` beheben — `WorkGRPCServer.UpdateTaskComment`/
  `DeleteTaskComment` (`internal/server/work_grpc.go:1028-1055`) reichten den Actor falsch durch:
  `UpdateTaskComment` rief den Service mit hartkodiertem `actorID=uuid.Nil` auf (Autor-Vergleich
  `AuthorID != actorID` also praktisch immer wahr → JEDES Update schlug mit
  `ErrCannotEditOthersComment` fehl, Feature de-facto kaputt), `DeleteTaskComment` rief mit
  `isAdmin=true` hardcoded auf (kompletter Bypass des Autor-Checks — jeder User mit
  `work:task_comment:delete` konnte jeden fremden Kommentar loeschen).
- gebaut:
  - `proto/work/v1/work.proto`: `DeleteTaskCommentRequest` bekommt `bool is_admin = 2`, Regen
    (`protoc` direkt, exaktes Kommando aus dem `proto`-Makefile-Target uebernommen — kein
    dediziertes `proto-work`-Target vorhanden) im selben Commit.
  - `internal/server/work_grpc.go`: `UpdateTaskComment`/`DeleteTaskComment` nutzen jetzt
    `actorIDFromContext(ctx)` (bereits in `document_grpc.go` definiert, selbes Package `server` —
    keine Dopplung, direkt wiederverwendet) statt `uuid.Nil`; `DeleteTaskComment` reicht
    `req.IsAdmin` statt hartkodiertem `true` an `commentService.Delete`.
  - `internal/gateway/route_work_tasks.go`: `HandleDeleteTaskComment` befuellt
    `IsAdmin: middleware.IsAdmin(r.Context())` — der einzige Ort, an dem die Rolle bekannt ist
    (Rollen propagieren nicht ueber den internen gRPC-Hop, nur `x-tenant-id`/`x-user-id`, exakt das
    Muster aus `g-dokumente-comments`). `HandleUpdateTaskComment` unveraendert, braucht kein
    `is_admin` (nur Autor darf editieren).
  - `internal/server/work_comment_test.go` (neu): 6 Testfaelle auf gRPC-Handler-Ebene (nicht nur
    Service, siehe `done_when`) mit `commentAuthzMockRepo` (echtes `GetByID` statt
    `commentStubRepo`s `ErrNotFound`-Stub) — Autor-darf-editieren, Nicht-Autor-gesperrt (Content
    bleibt unveraendert geprueft), fehlender Actor-Context → Unauthenticated, Autor-darf-loeschen,
    Nicht-Autor-Nicht-Admin-gesperrt (Kommentar bleibt bestehen geprueft), Admin-darf-fremde-loeschen.
    `comment.Service.Update`/`Delete` selbst waren nie das Problem (deren Tests in
    `work/comment/service_test.go` bestanden schon vorher) — der Bug sass ausschliesslich in der
    Verdrahtung des gRPC-Handlers, deshalb testet dieser Commit genau diese Schicht.
- gate: `go build -p 2 ./internal/work/... ./internal/gateway/... ./internal/server/...
  ./proto/work/... ./cmd/work/... ./cmd/gateway/...` ok | `go vet` auf denselben Paketen ok |
  `golangci-lint run --config .golangci.yml ./internal/work/... ./internal/gateway/...
  ./internal/server/...`: 0 issues |
  `DATABASE_URL=postgres://kmuhub_app:app_dev@localhost:5432/kmuhub?sslmode=disable go test
  -count=1 ./internal/work/...` PASS (alle Sub-Pakete, keine Skips) | `go test -count=1 -v
  ./internal/server/... -run TaskComment`: alle 6 neuen Tests PASS | `go test -count=1
  ./internal/server/...` PASS | `go test -count=1 -v ./internal/gateway/ -run
  TestOpenAPIRouteDrift`: PASS, 774 Routen gegen 776 dokumentierte Pfade (unveraendert — keine neue
  Route, nur Handler-Wiring) | `go test -count=1 ./internal/gateway/` PASS. Keine Migration, keine
  Tabelle, kein RLS-Bezug in dieser Unit — RLS-Smoke n.a.
- offen:
  - Keine. Fix ist vollstaendig: Create war nie betroffen (Autor kam dort schon vorher aus
    `middleware.GetUserID` in der Gateway, nicht aus dem gRPC-Handler), List braucht keinen
    Actor-Check.

## Iteration 59 — g-dokumente-share-links — done — 2026-08-02

- commit: (folgt direkt auf diesen Eintrag)
- verify vorgaenger: sauber. `8e7c56d7` (Iteration 58, fix-g-work-task-comment-authz) `git show
  --stat` gepruefter Diff deckt sich mit dem Journal-Eintrag: `route_work_tasks.go` befuellt
  `IsAdmin: middleware.IsAdmin(r.Context())` nur fuer Delete, `work_grpc.go` nutzt fuer
  Update/Delete durchgehend `actorIDFromContext(ctx)` statt `uuid.Nil`; `work.proto` +
  `.pb.go` im selben Commit regeneriert (kein `_grpc.pb.go`-Diff noetig, da nur ein Feld auf einer
  bestehenden Request-Message ergaenzt wurde, keine RPC-Signatur geaendert); 6 neue Tests gegen den
  echten gRPC-Handler (nicht nur den Service); keine Migration, kein RLS-Bezug. Kein
  Nacharbeitsbedarf.
- Auftrag: `g-dokumente-share-links` — externe, unauthentifizierte Freigabelinks fuer
  Dokument-Dateien mit Passwortschutz und Ablaufdatum, serverseitig durchgesetzt, mit Rate-Limit
  und nicht unterscheidbaren Fehlermeldungen (falsches Passwort vs. abgelaufener Link).
- SCOPE-KORREKTUR (Fund beim Bau): `fe-documents-links` (Iteration 40) hatte bereits geklaert, dass
  `/documents/links/{id}` interne CRM/PM-Entity-Links bedient (`DocumentEntityLink`), nicht die hier
  gemeinten externen Freigabelinks — die vom Backlog vermutete "Freigabelink mit
  Passwort/Ablauf"-UI (`ShareLinkDialog.tsx`) ist zu 100% clientseitiger Mock (`generateMockLink`,
  kein API-Call), dokumentiert im Kommentar `route_document.go:63-65` als "documents:share_link:create
  ... zero backend wiring". Diese Unit baut die echte Luecke, nicht die falsch vermutete.
- gebaut:
  - Migration `000266_document_share_links`: neue Tabelle `document_share_links` (id, tenant_id,
    file_id FK `document_files`, token UNIQUE, password_hash NULL, expires_at NULL, revoked_at NULL,
    view_count, created_by, created_at), `CALL enable_tenant_rls(...)` (Standardprozedur inkl.
    `tenant_id IS NULL OR ... OR is_system_context()`-Escape bereits eingebaut — kein manuelles
    Policy-SQL wie im aelteren `report_share_tokens`/000252-Vorbild noetig). Up/Down lokal
    durchgetestet (Kopf 266 -> 265 -> 266 sauber).
  - `internal/models/document.go`: `DocumentShareLink` + `Usable(now)`-Methode (Vorbild:
    `berichte.ShareToken`).
  - `internal/document/file/{repository,postgres_repository,service,errors}.go`: neue Methoden
    direkt auf `file.Repository`/`file.Service` — NICHT als eigenes Sub-Package. Begruendung:
    Comments und Entity-Links (juengste Faelle im selben Modul) liegen ebenfalls direkt im
    `file`-Package, nur das etablierte, aeltere interne Sharing (`document_shares`,
    User-zu-User) hat ein eigenes `share`-Package — externe Freigabelinks sind naeher an
    Comments/Entity-Links (file-scoped Sub-Feature) als an internem Sharing. `CreateShareLink`
    (prueft Datei-Eigentuemerschaft ueber `GetByID` vor dem Minten, wie
    `berichte.CreateShareToken`), `ListShareLinks`, `RevokeShareLink` (Soft-Revoke, zweites Revoke
    liefert `ErrShareLinkNotFound`, kein stiller No-op), `RedeemShareLink` (die public Kernlogik,
    s.u.). Token: 32 Byte `crypto/rand`, base64url. Passwort: bcrypt Cost 12, 72-Byte-Cap
    (`ErrSharePasswordTooLong`), Ablauf: max. 365 Tage (`ErrShareLinkExpiryInvalid`) — beide
    Konstanten/Grenzen identisch zum `berichte`-Vorbild uebernommen.
  - `proto/document/v1/document.proto`: 4 neue RPCs `CreateShareLink`/`ListShareLinks`/
    `RevokeShareLink`/`GetSharedFile` + 6 Messages, `.pb.go`/`_grpc.pb.go` im selben Commit
    regeneriert (`protoc` direkt via `make proto-document`-Flags — Target existiert diesmal, aber
    `make` selbst fehlt weiterhin auf dieser Maschine).
  - `internal/server/document_grpc.go`: 4 neue RPC-Handler + `toProtoShareLink`-Mapper +
    Fehler-Mapping in `mapDocumentError` (`ErrShareLinkNotFound`/`ErrShareLinkInvalid` -> NotFound,
    `ErrShareLinkExpiryInvalid`/`ErrSharePasswordTooLong` -> InvalidArgument). `CreateShareLink`/
    `ListShareLinks`/`RevokeShareLink` ziehen `tenantID` wie ueberall sonst aus
    `middleware.GetTenantID(ctx)`; `GetSharedFile` zieht **keinen** Tenant aus dem Context — der
    Token loest ihn intern in `RedeemShareLink` auf, exakt das berichte-Muster.
  - `internal/gateway/route_document.go`: neuer Guard `docShareLinkCreate` (additiv
    `documents:write` + `documents:share_link:create`, bereits seit `p1a-migration`/000256
    vollstaendig geseedet — verifiziert per psql gegen die lokale DB, kein neuer Seed noetig).
    Authentifiziert: `GET/POST /documents/files/{id}/share-links` (List `docRead`, Create
    `docShareLinkCreate`), `DELETE /documents/share-links/{id}` (Revoke `docShareManage` — derselbe
    Key wie ShareEntity/UnshareEntity, weil es fuer Revoke keinen eigenen Katalog-Key gibt;
    Entscheidung im Kommentar an der Guard-Deklaration begruendet, nicht "sinngemaess" geraten).
    Unauthentifiziert: neue Methode `DocumentRoutes.RegisterPublicRoutes(r, publicRateLimit)`
    (Vorbild `route_berichte.go`), registriert `POST /api/v1/public/documents/share/{token}` hinter
    dem strikten `publicRateLimiter` — in `cmd/gateway/main.go` OUTSIDE der Registrar-Schleife
    verdrahtet, dieselbe `publicRateLimiter`-Instanz wie booking/berichte-public (kein zweiter
    Limiter noetig). `documentRoutes` dafuer von einer Inline-Konstruktion in der Registrar-Liste
    auf eine benannte Variable umgestellt (Vorbild: `berichteRoutes`/`bookingRoutes` sind aus
    demselben Grund schon benannt).
  - SICHERHEITS-ENTSCHEIDUNG, staerker als das eigene `berichte`-Vorbild: `RedeemShareLink`
    liefert fuer unbekannten Token, widerrufenen Link, abgelaufenen Link, fehlendes Passwort UND
    falsches Passwort **denselben** Fehler `ErrShareLinkInvalid` -> ein einziger
    `mapDocumentError`-Case -> HTTP 404 mit identischer Nachricht. `berichte.ShareToken`
    unterscheidet dagegen "not found" (404, `ErrShareNotFound`) von "Passwort fehlt/falsch" (401,
    `ErrSharePasswordRequired`/`ErrSharePasswordInvalid`) per Status-Code — genau das
    Enumerations-Leck, das der Auftragstext ("kein Enumerieren ueber unterschiedliche
    Fehlermeldungen... falsches Passwort und abgelaufener Link antworten gleich") verbietet. Bewusst
    NICHT das bestehende Vorbild 1:1 kopiert, sondern strenger gefasst — im Code
    (`RedeemShareLink`-Doc-Kommentar) und hier begruendet, keine stillschweigende Abweichung.
  - `GetSharedFile`-HTTP-Antwort bleibt bei `response.JSON` mit manueller Map (nicht
    `response.Proto`/protojson wie bei Comments/ShareLink-CRUD): `file_size` ist `int64`, und
    protojson serialisiert 64-Bit-Felder laut `response.protoMarshaler`-Doc-Kommentar als
    JSON-STRING, nicht als Zahl. Shape ist 1:1 identisch zum bestehenden authentifizierten
    `GET .../download-url` (`download_url`/`filename`/`content_type`/`file_size`), das aus
    demselben Grund schon `response.JSON` nutzt.
  - `openapi_drift_test.go` (DB-loses Gegenstueck zu `TestOpenAPIRouteDrift`) baut seinen eigenen
    Test-Router unabhaengig von `main.go` auf — fehlte dort `documentRoutes.RegisterPublicRoutes`,
    waere die neue Public-Route dauerhaft unentdeckt als "dokumentiert aber nie registriert"
    durchgefallen. Im selben Commit nachgezogen (`documentRoutes` dort ebenfalls auf eine benannte
    Variable umgestellt, analog zu `berichteRoutes`).
- gate: `go build -p 2 ./...` (voller Baum) gruen | `go vet ./...` gruen | `golangci-lint run
  --config .golangci.yml` fuer `internal/document/... internal/gateway/... internal/server/...
  cmd/gateway/... cmd/document/...`: 0 issues | swagger-cli validate: `api/openapi.yaml is valid` |
  Migration lokal Up/Down/Up sauber (Kopf 266). `DATABASE_URL` gesetzt (Rolle `kmuhub_app`,
  NOSUPERUSER NOBYPASSRLS): `internal/document/...` PASS (0 Skips, per `-v` verifiziert), darunter
  8 neue DB-Tests in `postgres_repository_sharelink_test.go` (Create+List, Tenant-Isolation
  [fremder Tenant 0 Zeilen], `GetShareLinkByToken` OHNE Tenant im Context — die
  Public-Redemption-Situation, ueber `database.WithSystemContext` aufgeloest —, unbekannter Token,
  Revoke, Doppel-Revoke `ErrShareLinkNotFound`, Revoke unter fremdem Tenant [Zeile bleibt
  unangetastet], View-Count-Increment) + 12 neue Mock-Service-Tests in `service_test.go`, darunter
  `TestRedeemShareLink_IndistinguishableFailures` (5 Unterfaelle: unknown/revoked/expired/
  missing-password/wrong-password, beweist den kollabierten Fehler per `assert.ErrorIs`). `go test
  ./internal/server/...` PASS | `go test ./internal/gateway/...` PASS inkl. `TestOpenAPIRouteDrift`
  (777 Routen gegen 779 dokumentierte Pfade, +3/+3 durch die drei neuen Pfade) und
  `TestOpenAPISpecDrift` (778 dokumentierte Pfade gegen 777 registrierte Routen — Differenz ist die
  bereits vorbestehende `/api/v1/files/upload`-Allowlist-Ausnahme, unveraendert). 6 neue Faelle in
  `route_capability_guard_test.go` (List/Create/Revoke je mit erlaubtem und verweigertem Key) PASS.
  RLS-Smoke: siehe DB-Tests oben (Tenant-Isolation + System-Context-Escape sind der Beweis, kein
  separater psql-Handlauf noetig — die Go-Tests pruefen exakt dieselbe Eigenschaft mit Assertions
  statt Augenschein).
- offen:
  - Kein FE-Wiring (kein `desktop/`-Code angefasst) — reine Backend-Iteration. Der bestehende
    `ShareLinkDialog.tsx`-Mock (`generateMockLink`) bleibt unwired; sobald ein FE-Task die echte
    Route verdrahtet, ist das Backend inkl. Sicherheits-Design bereits vollstaendig.
  - Naechste Unit laut Reihenfolge: `g-helpdesk-contact-link` (Block G, deps: [], model: sonnet).
