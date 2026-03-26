---
tags: [testing, qualitaet]
updated: 2026-03-08
---
# Test-Strategie

## Backend (Go)
- Framework: Go stdlib `testing` + testify (assert/require/mock)
- 58 `*_test.go` Dateien, 240+ Tests allein im CRM, 162+ Gateway-Handler-Tests
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
- `uuid.New()` fuer unique IDs pro Test
- Error-Injection via Mock-Fields

### Spezial-Tests
- **ARBZG** — Arbeitszeitgesetz-Compliance (`biz/hr/compliance/arbzg_test.go`)
- **Steuer** — MWSt-Szenarien, Kleinunternehmer (`biz/tax/calculator_test.go`)
- **Spracherkennung** — Chat-Nachrichten (`chat/langdetect/detector_test.go`)
- **Workflow-Conditions** — `amount > 1000 AND status = 'approved'` (`automation/condition/evaluator_test.go`)
- **Middleware** — RBAC, Rate Limiter, Auth (`middleware/*_test.go`)

## Desktop (Electron/React)
- Framework: Vitest konfiguriert
- **Status:** Kaum Tests vorhanden — Ausbau noetig

## E2E Tests
- **Datei:** `backend/test/e2e/` (Build-Tag `//go:build e2e`)
- CI-Job vorhanden (nach Unit Tests), mit Service-Containers
- Tests fuer: CRM, Chat, Work, Document Services
- Helpers: `doRequest`, `registerAndLogin`, `requireStatus`, `decodeBody`
- **Makefile:** `make e2e-test`

## Smoke Tests (Pilot-Readiness)
Zwei Varianten mit gleicher Abdeckung:

### Bash (`deploy/scripts/smoke.sh`)
- Curl/jq-basiert, keine Go-Toolchain noetig
- 19 Tests in 6 Kategorien: Infra (5), Auth (3), CRM CRUD (3), Security (3), Performance (3), Cross-Service (2)
- Flags: `--base-url`, `--verbose`, `--expect-version`
- Smoke-User Cleanup am Ende
- Wird als Gate im `deploy.sh` nach Health Check ausgefuehrt

### Go (`backend/test/smoke/`)
- Build-Tag: `//go:build smoke`
- `smoke_test.go` (11 Tests) + `helpers_test.go`
- Konfigurierbar via `SMOKE_URL` env var (default: `http://localhost:8080`)
- `SMOKE_EXPECT_VERSION` fuer Version-Verification
- **Makefile:** `make smoke-test` (lokal), `make smoke-prod` (gegen Prod)
- CI-Job: Laeuft nach E2E in der Pipeline

## Coverage-Ziele
- **Gesamt:** 80%+ Minimum
- **Kritische Pfade (Auth, Payments, Data):** 95%+
- **Jeder PR:** Muss Tests fuer neuen Code enthalten

## Test-Pipeline (CI Reihenfolge)
1. **Lint** (parallel) — golangci-lint
2. **Test** (parallel) — Unit Tests + Race Detector + Coverage
3. **Build** (nach 1+2) — `make build`
4. **E2E** (nach 1+2) — Integration Tests mit laufenden Services
5. **Smoke** (nach 4) — Go Smoke Tests
6. **OpenAPI Validate** (parallel) — Spec-Validierung

## Verwandte Notes
- [[architektur]] — Service-Architektur
- [[api]] — Endpoints die getestet werden
- [[deployment]] — Smoke als Deploy-Gate, CD Pipeline
