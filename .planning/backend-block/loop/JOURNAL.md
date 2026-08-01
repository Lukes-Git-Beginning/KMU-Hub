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
