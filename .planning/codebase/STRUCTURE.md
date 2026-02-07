# Codebase Structure

**Analysis Date:** 2026-02-07

## Directory Layout

```
KMU Hub/
├── .claude/                # Claude Code agent configuration (not committed)
├── .github/workflows/      # CI/CD pipelines (lint, test, build, e2e)
├── .planning/              # GSD planning documents (committed)
│   └── codebase/           # Codebase analysis docs (ARCHITECTURE.md, STRUCTURE.md, etc.)
├── backend/                # Go backend services
│   ├── api/                # OpenAPI specification
│   ├── cmd/                # Service entry points (main.go files)
│   ├── internal/           # Private application code
│   ├── migrations/         # PostgreSQL migrations (golang-migrate)
│   ├── pkg/                # Reusable libraries (currently empty)
│   ├── proto/              # Protobuf definitions for gRPC
│   └── test/               # E2E tests
├── deploy/                 # Deployment configurations
│   ├── docker/             # Docker Compose for local dev
│   └── k8s/                # Kubernetes manifests (future)
├── desktop/                # Electron + React desktop app
│   ├── src/                # TypeScript/React source
│   ├── package.json        # NPM dependencies
│   └── tsconfig.json       # TypeScript configuration
├── docs/                   # Project documentation (ADRs, learnings, roadmap)
├── mobile/                 # React Native mobile app (future)
├── CLAUDE.md               # AI-first development rules and architecture principles
├── README.md               # Project overview
└── .gsdignore              # GSD exclusions
```

## Directory Purposes

**`.claude/`:**
- Purpose: Claude Code agent configuration and local settings
- Contains: Agent configuration, commands, settings
- Not committed to git

**`.github/workflows/`:**
- Purpose: CI/CD pipeline definitions
- Contains: `ci.yml` for lint, test, build, OpenAPI validation, E2E tests
- Triggered on push/PR to main

**`.planning/`:**
- Purpose: GSD (Get Shit Done) planning documents
- Contains: `codebase/` subdirectory with analysis docs (this file), future phase plans
- Committed to git for AI-assisted development workflow

**`backend/api/`:**
- Purpose: API contract definitions
- Contains: `openapi.yaml` (REST API spec)
- Key files: `openapi.yaml` (1000+ lines, all HTTP endpoints documented)

**`backend/cmd/`:**
- Purpose: Service entry points (binaries)
- Contains: `gateway/main.go`, `auth/main.go`, `crm/main.go`, `chat/main.go`
- Each `main.go` initializes service dependencies and starts servers

**`backend/internal/`:**
- Purpose: Private application code (Go convention: not importable by external projects)
- Contains: Domain modules (auth, crm, chat), shared infrastructure (database, middleware, config)
- Organized by domain (vertical slicing)

**`backend/internal/auth/`:**
- Purpose: Authentication and authorization domain
- Contains: `service.go`, `repository.go`, `postgres_repository.go`, `token.go`, errors, tests
- Key files: `service.go` (registration, login, refresh, RBAC), `token.go` (JWT maker)

**`backend/internal/crm/`:**
- Purpose: CRM domain modules
- Contains: Subpackages for each entity (contact, company, deal, activity, tag, customfield, pipelinestage, search, savedfilter, report)
- Pattern: Each subpackage has `service.go`, `repository.go`, `postgres_repository.go`, `errors.go`, `*_test.go`

**`backend/internal/chat/`:**
- Purpose: Chat and messaging domain
- Contains: `channel/`, `message/`, `file/`, `search/`, `langdetect/`
- Key features: DMs, threads, mentions, read receipts, typing indicators, file sharing, full-text search

**`backend/internal/database/`:**
- Purpose: Database connection management
- Contains: `postgres.go` (connection pool), `redis.go` (Redis client)
- Key files: Factory functions `NewPostgresPool()`, `NewRedisClient()`

**`backend/internal/middleware/`:**
- Purpose: HTTP middleware for cross-cutting concerns
- Contains: `auth.go`, `rbac.go`, `ratelimit.go`, `cors.go`, `logging.go`, `metrics.go`, `requestid.go`
- Pattern: Higher-order functions compatible with chi router

**`backend/internal/server/`:**
- Purpose: HTTP/gRPC server handlers and protocol conversion
- Contains: `http.go` (gateway routes), `grpc.go`, `auth_grpc.go`, `crm_grpc.go`, `chat_grpc.go`, `websocket.go`, `file_upload.go`, `response/`
- Pattern: Thin handlers that proxy to services (gateway) or delegate to service layer (gRPC servers)

**`backend/internal/models/`:**
- Purpose: Shared domain entity definitions
- Contains: `user.go`, `contact.go`, `company.go`, `deal.go`, `channel.go`, `chat_message.go`, etc.
- Pattern: Go structs with JSON and DB tags, UUIDs for IDs, pointers for nullable fields

**`backend/internal/config/`:**
- Purpose: Environment-based configuration loading
- Contains: `config.go` with `Config` struct
- Uses: `sethvargo/go-envconfig` for env var parsing with defaults

**`backend/internal/health/`:**
- Purpose: Health check implementations
- Contains: `checker.go`, `postgres.go`, `redis.go`
- Pattern: Interface `Checker`, implementations per dependency

**`backend/internal/metrics/`:**
- Purpose: Prometheus metrics collection
- Contains: `registry.go` with HTTP/gRPC interceptors
- Exposed on dedicated ports (`:9090-9093`)

**`backend/migrations/`:**
- Purpose: Database schema versioning
- Contains: Numbered SQL files (`000001_*.up.sql`, `000001_*.down.sql`)
- Naming: `{sequence}_{description}.{up|down}.sql`
- Executed by: `golang-migrate` CLI or Docker `migrate` service

**`backend/proto/`:**
- Purpose: gRPC service definitions (protobuf)
- Contains: `auth/v1/auth.proto`, `crm/v1/crm.proto`, `chat/v1/chat.proto`
- Generated code: `*pb.go`, `*_grpc.pb.go` (committed)

**`backend/test/e2e/`:**
- Purpose: End-to-end integration tests
- Contains: `auth_test.go` (registration, login, refresh, RBAC flows)
- Run via: `make e2e-test` or CI job

**`deploy/docker/`:**
- Purpose: Local development environment
- Contains: `docker-compose.yml`, `Dockerfile.*` for each service
- Services: postgres, redis, minio, migrate, auth, crm, chat, gateway

**`desktop/`:**
- Purpose: Electron desktop application
- Contains: `src/` (TypeScript/React), `package.json`, `tsconfig.json`
- Stack: Electron 33, React 19, TypeScript 5.7, Vite (via electron-vite), Tailwind 4

**`docs/`:**
- Purpose: Project documentation
- Contains: `ARCHITECTURE.md` (ADRs), `LEARNINGS.md` (past project mistakes), `PRICING.md`, `ROADMAP.md` (archive)
- Note: `ROADMAP.md` is historical; GSD planning in `.planning/` is current source of truth

## Key File Locations

**Entry Points:**
- `backend/cmd/gateway/main.go`: HTTP gateway server
- `backend/cmd/auth/main.go`: Auth gRPC service
- `backend/cmd/crm/main.go`: CRM gRPC service
- `backend/cmd/chat/main.go`: Chat gRPC service

**Configuration:**
- `backend/internal/config/config.go`: Environment variable config struct
- `.env` (local, not committed): Environment variables for development
- `deploy/docker/docker-compose.yml`: Service environment variables for Docker

**Core Logic:**
- `backend/internal/{domain}/{entity}/service.go`: Business logic for each entity
- `backend/internal/{domain}/{entity}/postgres_repository.go`: Database queries
- `backend/internal/server/http.go`: Gateway HTTP routes
- `backend/internal/server/{service}_grpc.go`: gRPC server implementations

**Testing:**
- `backend/internal/{domain}/{entity}/service_test.go`: Unit tests with mock repositories
- `backend/test/e2e/auth_test.go`: E2E integration tests
- Pattern: Mock repositories in `_test.go` files, service layer 100% covered

**API Contracts:**
- `backend/api/openapi.yaml`: REST API specification (validated in CI)
- `backend/proto/{service}/v1/{service}.proto`: gRPC contracts

**Database:**
- `backend/migrations/`: All schema changes
- Index naming: `idx_{table}_{column}` (e.g., `idx_contacts_email`)

**Frontend (Desktop):**
- `desktop/src/`: React components (structure TBD, minimal implementation currently)
- `desktop/package.json`: NPM dependencies

## Naming Conventions

**Files:**
- Go services: `{entity}/service.go`, `{entity}/repository.go`, `{entity}/postgres_repository.go`
- Go tests: `{entity}/service_test.go`, `{entity}/repository_test.go`
- Go models: `models/{entity}.go` (singular)
- Migrations: `{seq}_{description}.{up|down}.sql` (snake_case)
- Protobuf: `{service}.proto` (lowercase)

**Directories:**
- Go packages: lowercase, no underscores (e.g., `customfield`, `pipelinestage`)
- Domain modules: singular noun (e.g., `auth`, `crm`, `chat`)
- Entity packages: singular noun (e.g., `contact`, not `contacts`)

**Go Code:**
- Interfaces: `Repository`, `Service`, `Checker` (noun, no "I" prefix)
- Structs: PascalCase (e.g., `ContactService`, `PostgresRepository`)
- Functions: camelCase for private, PascalCase for exported
- Constants: PascalCase (e.g., `WSMessageSend`)

**Database:**
- Tables: plural snake_case (e.g., `users`, `custom_field_definitions`)
- Columns: snake_case (e.g., `first_name`, `created_at`)
- Indexes: `idx_{table}_{column}` (e.g., `idx_users_email`)
- Foreign keys: `fk_{table}_{ref_table}` pattern used in some migrations

## Where to Add New Code

**New CRM Entity (e.g., "Invoice"):**
- Primary code: `backend/internal/crm/invoice/`
- Files: `service.go`, `repository.go`, `postgres_repository.go`, `errors.go`, `service_test.go`
- Model: `backend/internal/models/invoice.go`
- Migration: `backend/migrations/{next_seq}_create_invoices.up.sql` (run `make migrate-create name=create_invoices`)
- Proto: Add RPCs to `backend/proto/crm/v1/crm.proto`, run `make proto`
- gRPC server: Add methods to `backend/internal/server/crm_grpc.go`
- Gateway routes: Add HTTP routes in `backend/internal/server/http.go` (RegisterRoutes method)
- OpenAPI: Document endpoints in `backend/api/openapi.yaml`
- Initialize: Add to `backend/cmd/crm/main.go` service registry

**New Microservice (e.g., "Billing"):**
- Entry point: `backend/cmd/billing/main.go`
- Domain logic: `backend/internal/billing/`
- Proto: `backend/proto/billing/v1/billing.proto`
- gRPC server: `backend/internal/server/billing_grpc.go`
- Gateway client: Add gRPC client in `backend/cmd/gateway/main.go`, proxy routes in `http.go`
- Docker: `deploy/docker/Dockerfile.billing`, add service to `docker-compose.yml`
- Config: Add `BillingGRPCPort` and `BillingGRPCAddress` to `backend/internal/config/config.go`

**New HTTP Endpoint (Gateway):**
- Route: Add to `backend/internal/server/http.go` in `RegisterRoutes()` method
- Pattern: `r.With(authMiddleware).With(middleware.RequirePermission("resource", "action")).Method("/path", h.Handler)`
- OpenAPI: Document in `backend/api/openapi.yaml`

**New Middleware:**
- Implementation: `backend/internal/middleware/{name}.go`
- Pattern: `func Middleware(deps) func(http.Handler) http.Handler { return func(next http.Handler) http.Handler { ... } }`
- Apply: In `backend/cmd/gateway/main.go` with `r.Use(middleware.Name(...))`

**New Migration:**
- Create: `make migrate-create name=descriptive_name` from `backend/` directory
- Edit: `backend/migrations/{seq}_descriptive_name.up.sql` and `.down.sql`
- Apply: `make migrate-up` (local) or restart Docker stack (runs automatically)

**New Protobuf Message/RPC:**
- Edit: `backend/proto/{service}/v1/{service}.proto`
- Generate: `make proto`
- Implement: Add handler in `backend/internal/server/{service}_grpc.go`

**Shared Utilities:**
- Location: `backend/pkg/{package}/` (currently unused, add as needed)
- Example use cases: Custom validators, string helpers, date utilities
- Import: External projects can import `backend/pkg/*`, but not `backend/internal/*`

**Desktop UI Component:**
- Implementation: `desktop/src/components/{Component}.tsx`
- Pattern: TBD (minimal implementation currently, React 19 conventions)

**E2E Test:**
- Location: `backend/test/e2e/{domain}_test.go`
- Build tag: `// +build e2e` at top of file
- Run: `make e2e-test`

## Special Directories

**`backend/bin/`:**
- Purpose: Compiled service binaries
- Generated: `make build`
- Committed: No (in `.gitignore`)

**`backend/migrations/`:**
- Purpose: Database schema versions
- Generated: `make migrate-create name=xxx`
- Committed: Yes (source of truth for schema)

**`backend/proto/{service}/v1/*.pb.go`:**
- Purpose: Generated Go code from protobuf
- Generated: `make proto` (requires protoc, protoc-gen-go, protoc-gen-go-grpc)
- Committed: Yes (for reproducibility, Go convention)

**`desktop/dist/`:**
- Purpose: Built Electron app
- Generated: `npm run build`
- Committed: No

**`.claude/`:**
- Purpose: Claude Code agent settings and commands
- Generated: GSD installation
- Committed: No (in `.gitignore`)

**`.planning/`:**
- Purpose: GSD planning documents
- Generated: GSD commands (`/gsd:map-codebase`, `/gsd:plan-phase`, etc.)
- Committed: Yes (AI workflow artifacts)

---

*Structure analysis: 2026-02-07*
