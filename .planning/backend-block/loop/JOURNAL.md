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
