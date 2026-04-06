---
tags: [testing, qualitaet]
updated: 2026-04-06
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
- `uuid.New()` für unique IDs pro Test
- Error-Injection via Mock-Fields

### Spezial-Tests
- **ARBZG** — Arbeitszeitgesetz-Compliance (`biz/hr/compliance/arbzg_test.go`)
- **Steuer** — MWSt-Szenarien, Kleinunternehmer (`biz/tax/calculator_test.go`)
- **Spracherkennung** — Chat-Nachrichten (`chat/langdetect/detector_test.go`)
- **Workflow-Conditions** — `amount > 1000 AND status = 'approved'` (`automation/condition/evaluator_test.go`)
- **Middleware** — RBAC, Rate Limiter, Auth (`middleware/*_test.go`)

## Desktop (Electron/React)
- Framework: Vitest konfiguriert
- **Status:** Kaum Tests vorhanden — Ausbau nötig

## E2E Tests
- **Datei:** `backend/test/e2e/` (Build-Tag `//go:build e2e`)
- CI-Job vorhanden (nach Unit Tests), mit Service-Containers
- Tests fuer: CRM, Chat, Work, Document Services
- Helpers: `doRequest`, `registerAndLogin`, `requireStatus`, `decodeBody`
- **Makefile:** `make e2e-test`

## Smoke Tests (Pilot-Readiness)
Zwei Varianten mit gleicher Abdeckung:

### Bash (`deploy/scripts/smoke.sh`)
- Curl/jq-basiert, keine Go-Toolchain nötig
- 19 Tests in 6 Kategorien: Infra (5), Auth (3), CRM CRUD (3), Security (3), Performance (3), Cross-Service (2)
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

## Playwright MCP (E2E / Visuelle Verifikation)
- Protocol: Chrome DevTools Protocol (CDP), Port 9222
- Konfiguration: `.mcp.json` im Projekt-Root
- Start: `npm run dev:test` — Electron im Demo-Modus mit CDP-Port
  - Befehl: `electron-vite dev --mode demo -- --remote-debugging-port=9222 --remote-allow-origins=*`
- Use-Cases: Visuelle Verifikation nach Aenderungen, Screenshot-Sessions (B10: 36 Screenshots), Crash-Detection (B9: alle Module auf 0 JS-Errors geprueft)

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
