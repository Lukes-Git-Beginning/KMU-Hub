---
tags: [testing, qualitaet]
updated: 2026-04-28
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
- CI-Job vorhanden (nach Unit Tests), mit Service-Containers
- Tests fuer: CRM, Chat, Work, Document, Dialer Services
- Helpers: `doRequest`, `registerAndLogin`, `requireStatus`, `decodeBody`
- **Makefile:** `make e2e-test`

## Smoke Tests (Pilot-Readiness)
Zwei Varianten mit gleicher Abdeckung:

### Bash (`deploy/scripts/smoke.sh`)
- Curl/jq-basiert, keine Go-Toolchain nötig
- 22 Tests in 7 Kategorien: Infra (5), Auth (3), CRM CRUD (3), Security (3), Performance (3), Cross-Service (2), Berichte (3, gated by `modules.berichte` — 404 akzeptiert wenn Flag OFF)
- Flags: `--base-url`, `--verbose`, `--expect-version`
- Smoke-User Cleanup am Ende
- Wird als Gate im `deploy.sh` nach Health Check ausgefuehrt

### Go (`backend/test/smoke/`)
- Build-Tag: `//go:build smoke`
- `smoke_test.go` (11 Tests) + `helpers_test.go`
- Konfigurierbar via `SMOKE_URL` env var (default: `http://localhost:8080`)
- `SMOKE_EXPECT_VERSION` für Version-Verification
- **Makefile:** `make smoke-test` (lokal), `make smoke-prod` (gegen Prod)
- CI-Job: Laeuft nach E2E in der Pipeline

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
