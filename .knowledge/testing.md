---
tags: [testing, qualitaet]
updated: 2026-06-08
---
# Test-Strategie

## Backend (Go)
- Framework: Go stdlib `testing` + testify (assert/require/mock)
- 61 `*_test.go` Dateien, 240+ Tests allein im CRM, 193+ Gateway-Handler-Tests
- Race Detector: `-race` Flag in CI aktiviert
- Coverage: Report generiert, als Artifact hochgeladen (30 Tage)

### Mock-Pattern
```go
type MockRepository struct {
    contacts  map[uuid.UUID]*models.Contact
    createErr error  // Error-Injection
}
```
- Eigene Mocks pro Test (keine shared State)
- `uuid.New()` für unique IDs pro Test
- Error-Injection via Mock-Fields

### Spezial-Tests
- **ARBZG** — Arbeitszeitgesetz-Compliance (`biz/hr/compliance/arbzg_test.go`)
- **Steuer** — MWSt-Szenarien, Kleinunternehmer (`biz/tax/calculator_test.go`)
- **Spracherkennung** — Chat-Nachrichten (`chat/langdetect/detector_test.go`)
- **Workflow-Conditions** — `amount > 1000 AND status = 'approved'` (`automation/condition/evaluator_test.go`)
- **Middleware** — RBAC, Rate Limiter, Auth (`middleware/*_test.go`)
- **Dialer Service** — LogCallOutcome (ContactID resolution, fallback, callback), campaign auto-complete (`dialer/service_test.go`, 6 Tests)
- **Dialer Gateway** — 31 Handler-Tests: ServiceUnavailable, InvalidJSON, InvalidUUID (`gateway/route_dialer_test.go`)
- **Dialer E2E** — Full flow: Campaign → Contact → Call → Outcome → Timeline (`test/e2e/dialer_test.go`)
- **Feature-Flag-Registry** (2026-04-18) — 9 Registry-Cases (`featureflag/registry_test.go`) + 4 Handler-Cases (`gateway/route_feature_flags_test.go`)
- **Consent-Asserter** (2026-04-18) — 7 Asserter-Cases (`crm/consent/asserter_test.go`) plus Integration-Tests `email/send/consent_test.go` (4) und `dialer/consent_test.go` (3) — Send-Pfad wird ohne Consent niemals erreicht
- **Prod-Secrets Validator** (2026-04-18) — `config/config_test.go` (TestValidateProductionSecrets)
- **Migration 000075** (2026-04-18) — `migrations/migration_000075_test.go` verifiziert `contact_id ON DELETE SET NULL`
- **Berichte-Modul** (2026-04-19, Sprint 1 Welle 5-6) — Export-Layer `internal/berichte/export/*_test.go` (Golden-File-Tests fuer PDF-Signatur/CSV-BOM+Semikolon/XLSX-Parseable, Coverage 80.2%), gRPC-Server `internal/server/berichte_grpc_test.go` (UUID-Validation + Error-Mapping, 77.6%), Gateway-Routes `gateway/route_berichte_test.go` (Flag-OFF/ON + RBAC, 57%), Scheduler `scheduler/scheduler_test.go` (Clock-Mock + atomic ClaimSchedule, 89.4%), Executor `executor/executor_test.go` (8 Kind-Handler mit nil-toleranten Downstream-Repos, 92.1%)
- **JWT Tenant-Claim** (2026-04-28, Sprint 2 Welle 2D) — `auth/token_test.go` (TenantID-Roundtrip + Empty-Legacy-Case), `middleware/auth_test.go` (`GetTenantID` valid + empty), neu `gateway/tenant_isolation_test.go` (10 Cases: no-tenant/empty-tid → 401, valid-tid → passes), `gateway/testutil_test.go` (`withTenantID`/`withAuth`-Helper). Bestehende Gateway-Tests (route_biz_test, route_berichte_test, …) injizieren `testTenantID` in Context.
- **Tenant-Isolation Erweiterung** (2026-04-29, Sprint 2 Welle 3.5) — `gateway/tenant_isolation_test.go` um 4 Cases fuer `/recordings/{id}/initiator-consent` erweitert (no-tenant, empty-tid, valid-tid, two-tenant-Scenario). Welle 3.5 deckt damit auch den Welle-3-Endpoint mit dem Standard-Tenant-Pattern ab.
- **Idempotency-Middleware** (2026-04-29, Sprint 2 Welle 3.5) — `middleware/idempotency_test.go` validiert `errors.Is`-Matching auf `ErrInFlight`/`ErrConflict`/`ErrKeyMissing` (vorher String-Equality, fail-open auf wrapped errors), atomarer `Reserve`-Pfad via `INSERT ... ON CONFLICT DO UPDATE RETURNING`, `context.WithoutCancel`-Pfad fuer async Complete.
- **Frontend Offline-Queue + CallControls** (2026-04-29, Sprint 2 Welle 3.5) — `api/__tests__/offline-queue.test.ts` deckt 409-Retry ab (in-flight wird als Retry, nicht als Success behandelt; vorher silently dropped). `features/video/__tests__/CallControls.test.tsx` validiert Doppelklick-Guard (`isPending` blockt zweite Mutation) und try/catch-Toast-Pfad bei `confirmInitiatorConsent`-Failure (kein Orphan-Recording-State).
- **Idempotency-Client-Coverage** (2026-04-29, Sprint 2 Welle 4A) — `api/__tests__/idempotency-coverage.test.ts` mit **29 Cases** stellt sicher, dass alle 32 API-Clients ihren Idempotency-Key-Header korrekt durch den neuen `api/utils/authenticatedFetch.ts`-Helper setzen. Voraussetzung fuer den Idempotency-HardMode-Switch in Welle 4B (vorher WarnMode, da Frontend-Rollout incremental war). Pro API-Client-Familie ein Round-Trip ueber den Helper plus Idempotency-Header-Verifikation.
- **Cross-Tenant-Isolation-Tests Welle 4B** (2026-05-07, Sprint 2 Welle 4B Stream 2D) — `gateway/tenant_isolation_test.go` um **12 Sub-Tests** erweitert: `TestTenantIsolation_Pipeline_Stages`, `_CalendarEvents`, `_TimeEntries`, `_Automations`, `_SavedFilters`, `_CustomFields`, `_EmailMessages`, `_InboxMessages`, `_Dialer_Campaigns`, `_AuditLog`, `_Recordings`, `_Channels`. Pattern: Two-Tenant-Smoke (`recA.Code != 401 && recB.Code != 401`). Echte DB-Backed Cross-Tenant-Boundary-Tests sind in `docs/sprint2-welle4b-followups.md` als F6 fuer Sprint 3 deferred. Plus 3 finance JSONB-Subtests in `gateway/route_biz_test.go` (`TestFinanceLineItems_JSONBTenantIsolation`) — bestaetigen dass `finance_line_items` als JSONB in Parent-Tabellen ausreichend per `tenant_id` isoliert sind (kein Schema-Change noetig).
- **Idempotency HardMode-Tests Welle 4B** (2026-05-07, Sprint 2 Welle 4B Stream 2C) — `idempotency/postgres_repository_test.go` mit `TestComplete_TenantFilter` + `TestGet_TenantIsolation` (Composite-PK-Verteidigung gegen Cross-Tenant-Replay), `middleware/idempotency_test.go` mit `TestHardMode_MissingKey_Returns400` + `TestHardMode_CrossTenantKeyRejected` (HardMode-Verhalten). F7 (`time.Sleep(20ms)`-Flakiness) deferred auf Sprint 3.
- **Recording Service-Test Welle 4B** (2026-05-07, Sprint 2 Welle 4B P2-2) — `work/recording/service_test.go` `TestStartRecording_TenantIsolation` mit `tenantAwareRepo`-Wrapper. Vorher pruefte der Test nur String-Existenz; jetzt echter Service-Aufruf mit Mock-Repos: TenantA erstellt Recording → `rec.TenantID == tenantA` ✓ → TenantB-Repo gibt ErrNotFound ✓.
- **Cross-Tenant-Tests Sprint 3 Welle 1A** (2026-05-08) — 4 echte DB-Backed Cross-Tenant-Boundary-Tests (F6) fuer Calendar-Events, Email-Messages, Recordings in `gateway/tenant_isolation_test.go`. Schliessen den Welle-4B-Followup, der HTTP-Smoke durch DB-getriebene Two-Tenant-Scenarios ersetzt. **Gesamt Cross-Tenant-Tests: ~30** (10 W2D + 4 W3.5 + 12 W4B + 4 Sprint3-F6).
- **Idempotency Flaky-Fix Sprint 3 Welle 1A** (2026-05-08) — `assert.Eventually(t, condition, 100*time.Millisecond, 5*time.Millisecond, "msg")` ersetzt `time.Sleep(20ms)` in `TestHardMode_CrossTenantKeyRejected` und `TestIdempotency_Replay_ReturnsCached`. Eliminiert Timing-Abhaengigkeit auf langsamen CI-Agents (F7-Followup).
- **Dialer Concurrent-Test Sprint 3** (2026-05-08) — `TestLogCallOutcome_Concurrent_SameSession` in `dialer/service_test.go`: 5 parallele Goroutinen rufen `LogCallOutcome` auf denselben Session-UUID auf. `mockCallRepo` mit `sync.Mutex` geschuetzt. Verifiziert atomare `UpdateSessionWithEvent`-Transaktion (kein Split-Brain).
- **Cross-Tenant-Tests Sprint 3 Welle 2B/3** (2026-05-08) — 8 neue Cross-Tenant-Tests fuer bexio-EntityMappings, lexware-SyncLogs, message_reactions, chat_files Isolation in `gateway/tenant_isolation_test.go` und den jeweiligen Repository-Tests.
- **Sprint-4-Skeleton-Tests** (2026-05-08, Commit `a1a8d54`) — 3 `t.Skip("ADR-0007: pending")` Tests in `backend/internal/biz/invoice/jsonb_test.go` als Tests-as-Documentation fuer die geplante `finance_line_items`-Normalisierung nach ADR-0007. **Seit 2026-06-08 (Commit `3e4c9055`, ADR-0007) implementiert** — `t.Skip` entfernt, die 3 reinen JSON-Roundtrip/TaxBreakdown/Korrupt-Tests laufen in normalem `go test`; die DB-gestützte Variante deckt der testcontainers-Harness ab (nächster Punkt).
- **testcontainers-go Finance-Integrationstests** (2026-06-08, ADR-0007, Commit `3e4c9055`) — **erster echter DB-Integration-Harness im Repo** (vorher nur In-Memory-Mocks). `internal/testsupport/pgtc` (`//go:build integration`): startet `pgvector/pgvector:pg16`, spielt alle `migrations/*.up.sql` ein, `ALTER ROLE kmuhub_app PASSWORD` + App-Pool via `database.NewPostgresPool` (NOBYPASSRLS → echte RLS-Erzwingung), Tenant via `ctx = context.WithValue(ctx, middleware.TenantIDKey, …)`. `integration_test.go` in invoice/quote/creditnote decken ab: relationales Roundtrip (Decimal-Gleichheit + Position-Reihenfolge), Update-Replace (kein Orphan-Leak), DB-CHECK-Rejection (`quantity<=0`/`tax_rate>100`), Backfill-Idempotenz, **RLS-Tenant-Isolation auf den Line-Tabellen**, Lock-Spalten. Coverage `-tags=integration`: invoice **69.6%** / quote **63.7%** / creditnote **51.3%**. ⚠ Build-Tag-gated → CI muss `-tags=integration` (+Docker-Service) laufen, damit die Coverage CI-seitig den Finance-Gate trifft (offener Follow-up). testcontainers-go ist test-only → normaler `go build`/Service-Binaries ziehen es nicht.
- **Sprint 4 Welle 1 Tenant-Isolation-Erweiterung** (2026-05-10) — pro neuem Wiring-Pfad ein Cross-Tenant-Test im jeweiligen Package: `auth/service_test.go` (`TestService_CreateSession_TenantID`, `TestService_RecoveryCode_TenantID`), neu `caldav/app_password_test.go` (App-Password + Push-Subscription), neu `chat/guest/tenant_isolation_test.go` (3 Tests), neu `chat/message/tenant_isolation_test.go` (2 Tests), neu `work/video/tenant_isolation_test.go` (3 Tests), neu `work/task/tenant_isolation_test.go` (Stream-C Bonus, 7 Tests fuer SetCustomFieldValues + LinkEntity), neu `work/project/tenant_isolation_test.go` (Stream-C, 5 Tests fuer user_project_preferences). Plus `internal/middleware/grpc_tenant_test.go` (4 Unit-Tests fuer Outbound/Inbound-Interceptor). **Gesamt Cross-Tenant-Tests nach Welle 1: ~55** (~30 vorher + ~25 neu).
- **slog-Pattern für gRPC-Error-Mapping** (2026-05-10, Welle 1c) — die 28 default-Branches in `*_grpc.go`-mapXxxError-Funktionen loggen unmappable Errors jetzt mit `slog.Error("unhandled <service> service error", "error", err)` bevor sie `codes.Internal` zurueckgeben. Pattern aus Welle-0.6-Fix `chat_grpc.go:1164`. Ohne Test-Coverage — manueller Test eines beliebigen BAD-Branches mit fake-Error im Fixture-Setup zeigt slog-Output.
- **Dialer-Coverage Welle 2A Sprint 3** (2026-05-08, Commit `1f6c4c0`) — Coverage 12% → **31.8%** (`./internal/dialer/...`). 4 neue Test-Files: `phone_test.go` (140 LOC, table-driven `NormalizePhoneE164` DE/AT/CH/E164/invalid + `FormatDuration`), `redis_agent_store_test.go` (133 LOC, `ValidateTransition`-State-Machine + `parseAgentStatusData`-pure-Funktionen, kein Redis-Fake), `dialer_grpc_test.go` (390 LOC, `mapDialerError` 10 Sentinels inkl. `consent.ErrNoConsent` + `InitiateDialerCall`/`LogCallOutcome`-Handler-Tests via Mocks, KEIN Bufconn). `service_test.go` von 607 → 1169 LOC (StartCampaign/Pause/Archive State-Transitions, CompleteWrapUp Auto-Complete, SaveCallNotes Cross-Tenant, GetNextContact Queue-Erschoepfung, AddContactsToCampaign Filter-Pfad, Create/UpdateCallOutcome nil-pointer-guards). **Bonus-Fix:** `mapDialerError` mappt `consent.ErrNoConsent` jetzt auf `codes.PermissionDenied` (war `codes.Internal`). Race-Detector skipped (Windows ohne CGO/GCC); CI-Linux deckt das ab. Test-Pattern: `testHarness` mit In-Package-Mocks (white-box `package dialer`), kein Bufconn — gesamte server-Test-Suite verwendet keine Bufconn-Server.
- **Mock-Copy-Semantik bei Async-Services** (2026-06-05, `91a3014c`) — **Regel: Wenn der Service Goroutinen spawnt, braucht der Mock DB-Semantik** (Mutex + Snapshot-Copy bei jedem Read/Write), sonst flaky `-race`-Fails. Fall: GDPR-`ApproveExport` startet `ExecuteExportAsync` async; der Mock gab geteilte Map-Pointer heraus → Test-Assertion ("approved") raste gegen die Goroutine-Mutation ("processing"). `gdpr/service_test.go::MockRepository` ist jetzt das Referenz-Pattern (Copy-on-Read/Write, `sync.Mutex`). Tests duerfen nach Service-Calls nur ueber frische Map-Lookups oder Returns asserten, nie ueber vorher gehaltene Pointer.

- **Input-Validation-Framework (S4.1, 2026-06-08, Commits `3937ff2d`+`cb784f79`)** — neue Test-Suiten fuer das `go-playground/validator`-Framework (siehe [[security]] "Validation-Framework"): `internal/dachfmt/dachfmt_test.go` (offizielle IBAN-Testvektoren DE/AT/CH/GB/FR/NL inkl. mod-97-Negativfaelle, BIC 8/11, PLZ je Land, USt-IdNr DACH+EU-VIES, Steuernummer-Laengen/Bundesland, Phone DE/AT/CH), `internal/validation/validation_test.go` (Builtins + Custom-Validatoren + Aggregat-String + `ErrorBody`-Shape). **Gateway-Test-Pattern:** invalid-input-Tests nutzen `assertValidationError(t, rec, "field")` (prueft 400 + `code=="validation_failed"` + Feld in `details`), Malformed-JSON bleibt auf `assertErrorContains(rec, "invalid request body")`. `route_auth_test.go` ist die Referenz; Welle 2 ergaenzte Format-Tests in route_biz/crm/dialer/video/helpdesk/wiki/security/guest. **Lesson (Welle-2-Incident):** parallele Subagenten im **geteilten** Working-Tree duerfen NIE `git stash`/`pull`/`reset`/`checkout` ausfuehren — ein Stash-Pop riss `server/websocket.go` in Konflikt (S4.7-Redis-Konstanten) und verwarf eine Handler-Konversion; Recovery via `git checkout HEAD --` + manuelles Nachziehen. Brief muss git-Mutationen in Subagenten explizit verbieten.

## Desktop (Electron/React)
- Framework: Vitest + jsdom, Setup: `test/setup.ts`
- **Status:** 12+ Test-Dateien (Stand 2026-04-18)
- i18next Mock: `t: (key) => key` — Tests assertieren auf i18n-Keys, nicht übersetzte Strings
- MSW für API-Mocks (ContactsFlow, DashboardPage)
- Bestehende: LoginPage, InvoiceForm, ChatFlow, DealsFlow, TeamFlow, CompaniesFlow, ContactsFlow, DashboardPage
- **Sprint 0 Neuzugaenge:**
  - `lib/__tests__/sanitize.test.ts` — 10 Cases fuer DOMPurify-Wrapper (Script/IFrame/OnClick/JS-URL/OnError stripping, Link-Hook, strict mode)
  - `components/shared/__tests__/FeatureGate.test.tsx` — 4 Cases (Flag on/off, loading-state, fallback)
  - `i18n/__tests__/plural.test.ts` — ICU-Plural-Regressions-Gate

## E2E Tests
- **Datei:** `backend/test/e2e/` (Build-Tag `//go:build e2e`)
- CI-Job vorhanden (nach Unit Tests), mit Service-Containers — startet auth/crm/chat/work/document/**dialer**/gateway, `RATE_LIMIT_RPS=1000` (CI-only)
- Tests fuer: Auth (inkl. InvitationFlow), CRM, Chat, Work, Document, Dialer Services — **komplett gruen seit 2026-06-05** (Modernisierung nach 3 Monaten API-Drift)
- Helpers: `doRequest`, `registerAndLogin`, `requireStatus`, `decodeBody`
- **Admin-Bootstrap (seit 2026-06-05):** `promoteToAdmin(t, userID)` promotet Test-User per Direkt-DB-INSERT in `user_roles` (pgx via `DATABASE_URL`), danach **Re-Login zwingend** (Permissions sind JWT-Snapshot). `registerAndLoginAdmin` kapselt register→login→promote→re-login. Noetig weil `/auth/register` nur die read-only `member`-Rolle vergibt.
- Contact-Emails/project_keys pro Run einzigartig generieren (persistente lokale DB → 409 bei statischen Werten)
- **Makefile:** `make e2e-test`
- **Lokales Rig:** Container `kmuhub-ci-test` (pgvector, Port 55432) + `kmuhub-ci-redis` (56379) + `kmuhub-ci-minio` (9000), Services seriell bauen mit `-ldflags="-w -s"` (Windows-Linker-OOM)

## Smoke Tests (Pilot-Readiness)
Zwei Varianten mit gleicher Abdeckung:

### Bash (`deploy/scripts/smoke.sh`)
- Curl/jq-basiert, keine Go-Toolchain nötig
- 24 Tests (Kategorien: Infra, Auth, CRM CRUD, Security, Performance, Cross-Service, Berichte gated by `modules.berichte`) — **24/24 PASS in Prod seit 2026-06-05**, `--skip-smoke` aus `cd.yml` entfernt (`914a12dd`)
- Token-Bootstrap: `SMOKE_ADMIN_EMAIL`+`PASSWORD` gesetzt ⇒ frischer Login ueberschreibt stales `SMOKE_ADMIN_TOKEN` — JWT-Rotationen brechen den Smoke nicht
- Flags: `--base-url`, `--verbose`, `--expect-version`
- Smoke-User Cleanup am Ende
- Wird als Gate im `deploy.sh` nach Health Check ausgefuehrt
- **Followup:** Test 25 LiveKit-Token-Probe (`/rtc/validate`=200) — siehe `docs/livekit-env-production-followups.md`

### Go (`backend/test/smoke/`)
- Build-Tag: `//go:build smoke`
- `smoke_test.go` (11 Tests) + `helpers_test.go`
- Konfigurierbar via `SMOKE_URL` env var (default: `http://localhost:8080`)
- `SMOKE_EXPECT_VERSION` für Version-Verification
- `SMOKE_CORS_ORIGIN` fuer den CORS-Preflight-Test (default `http://localhost:3000` = Dev-Allowlist; gegen Prod `https://app.zentria.tech` setzen)
- Admin-Bootstrap wie E2E (`registerAndLoginAdmin`), skippt sauber wenn `DATABASE_URL` fehlt (z. B. gegen Production)
- **Makefile:** `make smoke-test` (lokal), `make smoke-prod` (gegen Prod)
- CI-Job: Laeuft nach E2E in der Pipeline — **gruen seit 2026-06-05**

## Coverage-Ziele
- **Gesamt:** 80%+ Minimum
- **Kritische Pfade (Auth, Payments, Data):** 95%+
- **Jeder PR:** Muss Tests für neuen Code enthalten

## Test-Pipeline (CI Reihenfolge)
1. **Lint** (parallel) — golangci-lint
2. **Test** (parallel) — Unit Tests + Race Detector + Coverage
3. **Build** (nach 1+2) — `make build`
4. **E2E** (nach 1+2) — Integration Tests mit laufenden Services
5. **Smoke** (nach 4) — Go Smoke Tests
6. **OpenAPI Validate** (parallel) — Spec-Validierung

## Demo Mode als Testumgebung
- `RENDERER_VITE_DEMO_MODE=true` — Fetch-Interceptor mit realistischen Mock-Daten
- Kein Backend noetig fuer Frontend-Tests
- Mock-Daten: `desktop/src/renderer/src/mocks/data/`
- Handlers: `desktop/src/renderer/src/mocks/handlers/`
- Architektur-Details: [[architektur]] (Demo Mode Abschnitt)

## Verwandte Notes
- [[architektur]] — Service-Architektur, Demo Mode
- [[api]] — Endpoints die getestet werden
- [[deployment]] — Smoke als Deploy-Gate, CD Pipeline
