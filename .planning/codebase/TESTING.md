# Testing Patterns

**Analysis Date:** 2026-02-07

## Test Framework

**Runner:**
- Go standard library `testing`
- Go 1.25.6

**Assertion Library:**
- testify/assert for non-critical checks
- testify/require for critical checks (stops test on failure)

**Run Commands:**
```bash
go test ./... -v                              # Run all tests
go test ./... -v -race -count=1               # With race detector (CI mode)
go test ./... -coverprofile=coverage.out      # With coverage
make test                                     # Makefile wrapper
make test-coverage                            # Coverage HTML report
```

**E2E Tests:**
```bash
go test -tags=e2e ./test/e2e/... -v          # E2E tests (requires running services)
make e2e-test                                 # Makefile wrapper
```

## Test File Organization

**Location:**
- Co-located with source: `service.go` → `service_test.go`
- E2E tests separate: `backend/test/e2e/*_test.go`

**Naming:**
- Unit tests: `*_test.go` (e.g. `service_test.go`, `token_test.go`, `middleware_test.go`)
- E2E tests: `*_test.go` with build tag `//go:build e2e`

**Structure:**
```
backend/
├── internal/
│   ├── auth/
│   │   ├── service.go
│   │   ├── service_test.go       # Unit tests (100% service coverage)
│   │   ├── token.go
│   │   └── token_test.go         # Unit tests for token logic
│   ├── crm/
│   │   ├── contact/
│   │   │   ├── service.go
│   │   │   └── service_test.go   # Unit tests
│   └── middleware/
│       ├── auth.go
│       └── auth_test.go          # Middleware unit tests
└── test/
    └── e2e/
        └── auth_test.go          # E2E integration tests
```

## Test Structure

**Suite Organization:**
```go
func TestService_MethodName(t *testing.T) {
    tests := []struct {
        name    string
        input   InputType
        setup   func(*MockRepository)
        wantErr error
    }{
        {
            name: "success",
            input: InputType{...},
            setup: func(r *MockRepository) {
                // Arrange mock state
            },
        },
        {
            name: "error case",
            input: InputType{...},
            setup: func(r *MockRepository) {
                // Arrange failing state
            },
            wantErr: ErrExpected,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            svc, repo := newTestService()
            tt.setup(repo)

            result, err := svc.MethodName(context.Background(), tt.input)

            if tt.wantErr != nil {
                assert.ErrorIs(t, err, tt.wantErr)
                assert.Nil(t, result)
            } else {
                require.NoError(t, err)
                assert.NotNil(t, result)
                // Additional assertions
            }
        })
    }
}
```

**Patterns:**
- Table-driven tests with `t.Run()` subtests
- AAA pattern: Arrange (setup) → Act (call method) → Assert (verify)
- Test names: `TestType_Method` for unit, `TestFlowName` for E2E

## Mocking

**Framework:** Manual mocks (no mocking library)

**Patterns:**
```go
// MockRepository implements Repository for testing
type MockRepository struct {
    contacts      map[uuid.UUID]*models.Contact
    contactTags   map[uuid.UUID][]*models.Tag
    customFields  map[uuid.UUID]map[uuid.UUID]any
    createErr     error  // Injectable errors for failure cases
    getErr        error
}

func NewMockRepository() *MockRepository {
    return &MockRepository{
        contacts:     make(map[uuid.UUID]*models.Contact),
        contactTags:  make(map[uuid.UUID][]*models.Tag),
        customFields: make(map[uuid.UUID]map[uuid.UUID]any),
    }
}

func (m *MockRepository) Create(ctx context.Context, contact *models.Contact) error {
    if m.createErr != nil {
        return m.createErr
    }
    m.contacts[contact.ID] = contact
    return nil
}
```

**What to Mock:**
- Repository layer (database access) - ALWAYS mocked in service tests
- External services (not yet present, but pattern established)
- Time (not currently mocked, uses real time.Now())

**What NOT to Mock:**
- Service layer business logic (that's what we're testing)
- Models/DTOs (real structs used)
- Standard library (context, uuid generation in tests)

## Fixtures and Factories

**Test Data:**
```go
func createTestUser(repo *mockRepository, email, password string, active bool) *models.User {
    hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
    user := &models.User{
        ID:           uuid.New(),
        Email:        email,
        PasswordHash: string(hash),
        FirstName:    "Test",
        LastName:     "User",
        IsActive:     active,
        CreatedAt:    time.Now(),
        UpdatedAt:    time.Now(),
    }
    repo.users[user.ID] = user
    repo.usersByEmail[user.Email] = user
    return user
}

func newTestService() (*Service, *mockRepository) {
    repo := newMockRepository()
    tm := NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
    svc := NewService(repo, tm)
    return svc, repo
}
```

**Location:**
- Helper functions in same test file, before test functions
- Naming: `create*`, `new*` prefix for factories
- Pattern: Return both created entity and store reference in mock

## Coverage

**Requirements:**
- Overall: 80%+ minimum (currently ~14.7% due to untested handlers/proto)
- Service layer: 100% achieved in all service packages
- Critical paths (auth, tokens, business logic): 95%+

**View Coverage:**
```bash
make test-coverage                           # Generates coverage.html
go tool cover -func=coverage.out             # Terminal output
```

**CI Coverage:**
- Printed in CI but no hard threshold enforcement
- Coverage artifact uploaded for 30 days
- Race detector enabled: `-race -count=1`

## Test Types

**Unit Tests:**
- Scope: Single service/package in isolation
- Approach: Mock all dependencies via Repository interface
- Example: `internal/auth/service_test.go` tests auth.Service with mockRepository
- Coverage: 100% of service layer methods

**Integration Tests:**
- Scope: Not present yet (could test repository against real Postgres)
- Approach: Would use test database, transactions rolled back

**E2E Tests:**
- Scope: Full HTTP flow through gateway → gRPC → service
- Framework: `test/e2e/auth_test.go` with build tag `//go:build e2e`
- Infrastructure: GitHub Actions service containers (Postgres + Redis)
- Services: Started as binaries, health-checked before tests
- Assertions: HTTP status codes, JSON response structure

## Common Patterns

**Async Testing:**
- Not extensively used (most operations synchronous)
- E2E tests use polling with timeout:
```go
func waitForHealth(t *testing.T, baseURL string) {
    t.Helper()
    deadline := time.Now().Add(60 * time.Second)
    for time.Now().Before(deadline) {
        resp, err := http.Get(baseURL + "/health")
        if err == nil && resp.StatusCode == http.StatusOK {
            resp.Body.Close()
            return
        }
        time.Sleep(2 * time.Second)
    }
    t.Fatal("gateway did not become healthy within 60 seconds")
}
```

**Error Testing:**
```go
if tt.wantErr != nil {
    assert.ErrorIs(t, err, tt.wantErr)
    assert.Nil(t, result)
} else {
    require.NoError(t, err)
    assert.NotNil(t, result)
}
```

**HTTP Helper Functions (E2E):**
```go
func postJSON(t *testing.T, url string, body interface{}, token string) (*http.Response, []byte) {
    t.Helper()
    b, _ := json.Marshal(body)
    req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    if token != "" {
        req.Header.Set("Authorization", "Bearer "+token)
    }
    resp, _ := http.DefaultClient.Do(req)
    defer resp.Body.Close()
    var respBody bytes.Buffer
    respBody.ReadFrom(resp.Body)
    return resp, respBody.Bytes()
}
```

**Context Usage:**
- All service/repository methods take `context.Context` as first param
- Tests use `context.Background()` (no timeouts/cancellation tested yet)

## CI/CD Integration

**Pipeline:** `.github/workflows/ci.yml`

**Jobs:**
1. **lint**: golangci-lint v2.8 (5m timeout)
2. **test**: Unit tests with race detector, coverage report
3. **build**: Compile all services (gateway, auth, crm, chat)
4. **e2e**: E2E tests with service containers + migrations
5. **openapi-validate**: Validate OpenAPI spec

**Test Environment:**
- Ubuntu latest
- Go 1.25.6
- Postgres 16-alpine service container
- Redis 7-alpine service container
- Environment vars: DATABASE_URL, REDIS_URL, JWT_SECRET

**Test Execution:**
```bash
go test ./... -v -race -count=1 -coverprofile=coverage.out -covermode=atomic
```

**E2E Execution:**
- Migrations run via golang-migrate
- Services built and started as background processes
- Health checks with 30s timeout
- Tests run with `-tags=e2e -count=1`
- Service logs dumped on failure

## Test Isolation

**Service Tests:**
- Each test creates fresh mock repository: `repo := newMockRepository()`
- No shared state between tests
- Subtests independent via `t.Run()`

**E2E Tests:**
- Unique email per test: `fmt.Sprintf("e2e-%d@test.com", time.Now().UnixNano())`
- Tests can run in parallel (currently sequential)
- Services cleaned between CI runs

## Performance Testing

**Not currently implemented**, but conventions established:
- Use `testing.B` for benchmarks
- Naming: `Benchmark*` functions
- Would live in `*_test.go` files alongside unit tests

---

*Testing analysis: 2026-02-07*
