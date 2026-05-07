# Sprint 2 Welle 4B Followups

Findings aus Welle 4B.3 Bugfix-Sweep (Stream 3A) — alle bewusst deferred auf Sprint 3.

## P2 (defer Sprint 3)

### F6: tenant_isolation_test.go — alle 12 P2-7-Tests sind Smoke-Tests
- File: `backend/internal/gateway/tenant_isolation_test.go`
- Problem: Alle Tests pruefen nur HTTP 401-Verhalten, kein echter Cross-Tenant-Boundary-Assert (Tenant A INSERT → Tenant B GET → 404).
- Fix Sprint 3: Fuer mind. 3 kritische Domains (Calendar, Email, Recordings) echte DB-Backed-Tests mit testutil/db.go-Setup.

### F7: TestHardMode_CrossTenantKeyRejected mit time.Sleep(20ms) — flaky
- File: `backend/internal/middleware/idempotency_test.go:~295`
- Problem: 20ms-Sleep auf langsamen CI-Agents potentiell flaky.
- Fix Sprint 3: Sync-Completion durch synchrones Mock-Verhalten oder wg.Wait().

### F8: email/account ListAllActive — kein Access-Control-Comment
- File: `backend/internal/email/account/postgres_repository.go:105-126`
- Problem: ListAllActive liefert cross-tenant Sync-Accounts. Caller-Audit-Comment fehlt.
- Fix Sprint 3: `// INTERNAL: only callable from sync engine worker, not from HTTP handlers`-Kommentar + Caller-Audit.

## P3 (defer Sprint 3)

### F9: ListBrowsable-Calendar-Query — fehlender shared-Filter nach tenant_id-Add
- File: `backend/internal/work/calendar/postgres_repository.go::ListBrowsable`
- Problem: Nach Retrofit fehlt `c.calendar_type = 'shared'`-Filter — alle Calendar-Types sind nun browsable fuer alle Tenant-User.
- Fix Sprint 3: Pruefen ob `calendar_type = 'shared'` als zusaetzliche Bedingung wieder hinein soll (semantik-Frage).

### F10: UpdateActionItem / DeleteActionItem ohne tenant_id — INTEGRAL MIT F3 BEHOBEN
- Status: **Erledigt in Welle 4B.3** — integral mit F3 zusammengefasst.
- Was gefixt wurde: UpdateActionItem-Repo-Sig `(ctx, item, tenantID)` + SQL `WHERE id=$1 AND tenant_id=$6`;
  DeleteActionItem-Repo-Sig `(ctx, id, tenantID)` + SQL `WHERE id=$1 AND tenant_id=$2`;
  Service-Sig-Updates + gRPC-Handler-Updates (UpdateActionItem + DeleteActionItem) mit GetTenantID(ctx)-Extraktion.
