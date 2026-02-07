# Coding Conventions

**Analysis Date:** 2026-02-07

## Naming Patterns

**Files:**
- Service layer: `service.go` (e.g. `internal/auth/service.go`)
- Tests: `*_test.go` (e.g. `service_test.go`, `token_test.go`)
- Repository interface: `repository.go`
- Repository implementation: `*_repository.go` (e.g. `postgres_repository.go`)
- Errors: `errors.go` (per package)
- Models: lowercase singular (e.g. `contact.go`, `deal.go`)

**Functions:**
- Exported: PascalCase (`Register`, `CreateContact`, `ValidateToken`)
- Unexported: camelCase (`createTokenPair`, `scanContact`, `membershipKey`)
- Constructors: `New*` prefix (`NewService`, `NewPostgresRepository`, `NewTokenMaker`)

**Variables:**
- Local: camelCase (`user`, `refreshToken`, `existingContact`)
- Package-level constants: camelCase (`bcryptCost`, `invitationExpiry`)
- Exported constants: PascalCase or SCREAMING_SNAKE for sets (`ErrUserExists`, `validCurrencies`)

**Types:**
- Structs: PascalCase (`Service`, `Contact`, `TokenPair`)
- Interfaces: PascalCase with implied "er" suffix or explicit naming (`Repository`, `Checker`, `FileStore`)
- Mock implementations: `Mock*` or `mock*` prefix (`MockRepository`, `mockRepository` for unexported)

## Code Style

**Formatting:**
- Tool: goimports (configured in `backend/.golangci.yml`)
- Local prefix: `github.com/kmuhub/kmuhub` (standard library → external → internal)
- Indentation: Tabs (Go standard)

**Linting:**
- Tool: golangci-lint v2.8
- Config: `backend/.golangci.yml` (version: "2")
- Enabled linters: standard set + gosec, bodyclose, misspell, unconvert, govet (with shadow)
- Exclusions: G404 (weak random), G115 (integer overflow), errcheck/gosec in `*_test.go`
- Timeout: 5m

## Import Organization

**Order:**
1. Standard library imports (blank line)
2. External dependencies (blank line)
3. Internal packages (`github.com/kmuhub/kmuhub/internal/*`)

**Example from `internal/auth/service.go`:**
```go
import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "log/slog"
    "time"

    "github.com/google/uuid"
    "golang.org/x/crypto/bcrypt"

    "github.com/kmuhub/kmuhub/internal/models"
)
```

**Path Aliases:**
- Proto packages: Import with alias matching service name (e.g. `authv1 "github.com/kmuhub/kmuhub/proto/auth/v1"`)
- No other aliases used in application code

## Error Handling

**Patterns:**
- Package-level sentinel errors in `errors.go`: `var Err* = errors.New("message")`
- Example: `var ErrUserNotFound = errors.New("user not found")`
- Return errors directly, don't wrap unless adding context
- Check errors with `errors.Is()` in tests: `assert.ErrorIs(t, err, ErrUserExists)`
- Repository errors mapped to service errors: `if err != nil { return nil, ErrContactNotFound }`

**Error Naming:**
- Prefix: `Err` + description in PascalCase
- Examples: `ErrInvalidCredentials`, `ErrTokenExpired`, `ErrChannelNotFound`, `ErrNameRequired`

**Error Messages:**
- Lowercase, no punctuation: `"user not found"`, `"invalid credentials"`, `"token expired"`

## Logging

**Framework:** `log/slog` (standard library)

**Patterns:**
- Structured logging with key-value pairs
- Use Info level for business events, Error for failures, Warn for anomalies
- Always include relevant entity IDs

**Examples:**
```go
slog.Info("user registered", "user_id", user.ID, "email", user.Email)
slog.Error("failed to assign default role", "user_id", user.ID, "error", err)
slog.Warn("revoked refresh token reuse detected, revoking all tokens", "user_id", stored.UserID)
```

**HTTP Request Logging:**
- Middleware: `internal/middleware/logging.go`
- Fields: method, path, status, duration_ms, request_id, remote_addr

**NO fmt.Println or log.Println allowed** - enforced by convention, violations not committed

## Comments

**When to Comment:**
- Exported functions/types: GoDoc comments (full sentence starting with name)
- Complex business logic inline (minimal, prefer self-documenting code)
- Error constant groups

**JSDoc/GoDoc:**
```go
// Service handles deal business logic
type Service struct {
    repo Repository
}

// Create creates a new deal
func (s *Service) Create(ctx context.Context, input CreateInput) (*models.DealWithRelations, error) {
```

**No TODO/FIXME comments observed** - use issue tracker instead

## Function Design

**Size:**
- Service methods: 20-80 lines typical
- Complex flows broken into private helpers (e.g. `createTokenPair`)
- Repository methods: 10-50 lines

**Parameters:**
- Use context.Context as first parameter for all service/repository methods
- Group related parameters into structs (e.g. `CreateInput`, `ListFilter`)
- Use pointers for optional fields (`firstName *string`)

**Return Values:**
- Domain model + error: `(*models.User, error)`
- Model with metadata: `([]*models.Contact, int, error)` (list, total, error)
- Tokens returned separately: `(*models.User, *models.TokenPair, error)`
- Boolean checks: `(bool, error)` not just `bool`

## Module Design

**Exports:**
- Service struct and constructor: `type Service struct`, `func NewService(...) *Service`
- Repository interface: `type Repository interface`
- Input/Filter structs: `type CreateInput struct`, `type ListFilter struct`
- Sentinel errors: `var Err* = errors.New(...)`

**Barrel Files:**
- Not used (Go doesn't have this pattern)
- Each package exports directly from respective files

## Architecture Patterns

**Service Layer:**
- Thick services, thin handlers (business logic ONLY in services)
- Service struct holds dependencies (repository, other services)
- Services are stateless beyond their injected dependencies

**Repository Pattern:**
- Interface defined in service package: `internal/*/repository.go`
- Implementation: `postgres_repository.go` (PostgreSQL)
- Mock for tests: `MockRepository` in `*_test.go` files

**Dependency Injection:**
- Constructor functions: `NewService(repo Repository) *Service`
- No global singletons, no `init()` for application state
- Config loaded once in `main.go`, passed down

**Error Boundary:**
- Service layer returns domain errors (e.g. `ErrUserNotFound`)
- gRPC layer maps to status codes (`internal/server/*_grpc.go`)
- HTTP layer uses `internal/server/response` helper

## JSON Conventions

**Tags:**
- snake_case for JSON: `json:"first_name"` (protobuf compatibility)
- Omit empty for optional fields: `json:"email,omitempty"`
- No camelCase in JSON responses

**HTTP Response Helpers:**
- Success: `response.JSON(w, http.StatusOK, data)`
- Error: `response.Error(w, http.StatusBadRequest, "message")`
- Location: `internal/server/response/response.go`

## Security Practices

**Password Hashing:**
- bcrypt with cost 12: `const bcryptCost = 12`
- `bcrypt.GenerateFromPassword([]byte(password), bcryptCost)`

**Token Security:**
- Refresh tokens: SHA-256 hashed before storage
- Access tokens: JWT (not stored, validated via signature)
- Function: `HashToken()` in `internal/auth/token.go`

**SQL Injection Prevention:**
- ALWAYS use prepared statements (pgx $1, $2 placeholders)
- NEVER string concatenation for queries
- Example: `pool.Exec(ctx, "INSERT INTO users (...) VALUES ($1, $2, $3)", ...)`

**Input Validation:**
- Trim whitespace: `strings.TrimSpace(input.Name)`
- Normalize: `strings.ToLower(email)`, `strings.ToUpper(currency)`
- Validate before DB operations

---

*Convention analysis: 2026-02-07*
