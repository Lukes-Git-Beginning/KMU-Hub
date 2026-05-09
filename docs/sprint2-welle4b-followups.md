# Sprint 2 Welle 4B Followups

> **Closure 2026-05-09:** Pre-Sprint-4-Cleanup hat alle hier gelisteten Items abgeklaert (Code, Tests, oder Verifikation). Details siehe MEMORY `project_followup_cleanup_20260509.md`.

Findings aus Welle 4B.3 Bugfix-Sweep (Stream 3A) — historisch dokumentiert.

## P2

### F6: tenant_isolation_test.go — alle 12 P2-7-Tests sind Smoke-Tests
- File: `backend/internal/gateway/tenant_isolation_test.go`
- Problem: Alle Tests pruefen nur HTTP 401-Verhalten, kein echter Cross-Tenant-Boundary-Assert (Tenant A INSERT → Tenant B GET → 404).
- Status (2026-05-09): **Partial done** — 4 echte DB-Backed Tests fuer Calendar/Email/Recordings/Channels existieren bereits aus Sprint 3 Welle 1A. F6-Vertiefung fuer Pipeline_Stages/Automations/AuditLog/Dialer_Campaigns deferred auf **Sprint 5** (Stretch im Pre-Sprint-4-Cleanup nicht ausgefuehrt — separates DB-Test-Harness-Setup).

### F7: TestHardMode_CrossTenantKeyRejected mit time.Sleep(20ms) — flaky
- File: `backend/internal/middleware/idempotency_test.go:~295`
- Status (2026-05-09): **Done in Sprint 3 Welle 1A** — `assert.Eventually(...)` ersetzt `time.Sleep(20ms)`. Verifiziert: kein `time.Sleep(*Millisecond)` mehr in `idempotency_test.go`.

### F8: email/account ListAllActive — kein Access-Control-Comment
- File: `backend/internal/email/account/postgres_repository.go:105-126`
- Status (2026-05-09): **Done** — Methode hat jetzt expliziten `INTERNAL — cross-tenant system query`-Block mit Caller-Audit-Liste (`internal/email/sync/engine.go::Engine.startSyncLoops`, `Engine.triggerImmediateSync`) plus `MUST NOT be called from HTTP handlers`-Hinweis.

## P3

### F9: ListBrowsable-Calendar-Query — fehlender shared-Filter nach tenant_id-Add
- File: `backend/internal/work/calendar/postgres_repository.go::ListBrowsable`
- Status (2026-05-09): **Done — Code hat den Filter** — Z.227 `AND c.calendar_type = 'shared'` plus Doc-Block (Z.208-220) erlaeutert die Bedingungen. Cross-Tenant-Test in `service_test.go:1569` deckt den Fall (Tenant B sieht Tenant A's shared calendar nicht).

### F10: UpdateActionItem / DeleteActionItem ohne tenant_id — INTEGRAL MIT F3 BEHOBEN
- Status: **Erledigt in Welle 4B.3** — integral mit F3 zusammengefasst.
- Was gefixt wurde: UpdateActionItem-Repo-Sig `(ctx, item, tenantID)` + SQL `WHERE id=$1 AND tenant_id=$6`;
  DeleteActionItem-Repo-Sig `(ctx, id, tenantID)` + SQL `WHERE id=$1 AND tenant_id=$2`;
  Service-Sig-Updates + gRPC-Handler-Updates (UpdateActionItem + DeleteActionItem) mit GetTenantID(ctx)-Extraktion.
