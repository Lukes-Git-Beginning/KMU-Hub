# Architecture

**Analysis Date:** 2026-02-07

## Pattern Overview

**Overall:** Microservices Architecture with API Gateway + gRPC Backend

**Key Characteristics:**
- HTTP/REST Gateway fronts internal gRPC microservices
- Service-oriented with clear domain boundaries (Auth, CRM, Chat)
- Repository pattern for data access abstraction
- Event-driven real-time via WebSocket hub
- Stateless services with external state (PostgreSQL, Redis, MinIO)

## Layers

**API Gateway Layer:**
- Purpose: HTTP ingress, authentication, routing, protocol translation
- Location: `backend/cmd/gateway/`, `backend/internal/server/http.go`, `backend/internal/server/*_grpc.go`
- Contains: HTTP handlers, gRPC clients, middleware (auth, rate limit, CORS), WebSocket hub
- Depends on: Auth/CRM/Chat gRPC clients, Redis (rate limiting), PostgreSQL (file upload direct access), MinIO (file upload)
- Used by: Desktop app, future mobile app, external API consumers
- Pattern: Thin handlers that proxy to gRPC services, WebSocket hub for real-time events

**Service Layer (Business Logic):**
- Purpose: Domain logic, validation, orchestration
- Location: `backend/internal/{auth,crm,chat}/*/service.go`
- Contains: Business rules, input validation, transaction coordination, error handling
- Depends on: Repository interfaces, external services (bcrypt, JWT maker, file store)
- Used by: gRPC server handlers
- Pattern: Thick services following "fat service, thin handler" principle

**Repository Layer (Data Access):**
- Purpose: Database abstraction, SQL query execution
- Location: `backend/internal/{auth,crm,chat}/*/repository.go`, `backend/internal/{auth,crm,chat}/*/postgres_repository.go`
- Contains: SQL queries, pgx connection pool usage, database transaction handling
- Depends on: PostgreSQL connection pool, models
- Used by: Service layer
- Pattern: Interface-based with PostgreSQL implementation, enables mocking in tests

**gRPC Server Layer:**
- Purpose: Service exposure via gRPC, protocol serialization
- Location: `backend/internal/server/*_grpc.go` (auth_grpc.go, crm_grpc.go, chat_grpc.go)
- Contains: gRPC handlers, protobuf <-> domain model conversion, error mapping
- Depends on: Service layer, protobuf definitions
- Used by: API Gateway (via gRPC clients)
- Pattern: Thin handlers that delegate to service layer

**Data Model Layer:**
- Purpose: Domain entity definitions shared across layers
- Location: `backend/internal/models/*.go`
- Contains: Go structs for users, contacts, deals, channels, messages, custom fields, etc.
- Depends on: Nothing (pure data)
- Used by: All layers
- Pattern: Shared structs with JSON/DB tags, UUIDs for primary keys, nullable fields via pointers

**Infrastructure Layer:**
- Purpose: Cross-cutting concerns (database, Redis, health, metrics, middleware)
- Location: `backend/internal/{database,health,metrics,middleware,config}/`
- Contains: Connection pools, health checkers, Prometheus metrics, auth/RBAC/rate limiting middleware, config loader
- Depends on: External systems (PostgreSQL, Redis, MinIO)
- Used by: All services and gateway
- Pattern: Factory functions for singletons, middleware as higher-order functions

## Data Flow

**Authenticated Request Flow:**

1. Client sends HTTP request with JWT to Gateway (`:8080`)
2. Gateway middleware validates JWT locally (no gRPC call) via `auth.TokenMaker`
3. Handler extracts user context (ID, roles, permissions) from claims
4. Gateway calls appropriate gRPC service (Auth/CRM/Chat on `:50051-53`)
5. gRPC server handler receives protobuf request
6. gRPC handler converts protobuf to domain models
7. Service layer executes business logic (validation, orchestration)
8. Repository layer queries PostgreSQL
9. Response flows back: Repository -> Service -> gRPC handler (converts to protobuf) -> Gateway -> Client (converts to JSON)

**WebSocket Real-Time Flow:**

1. Client connects to `/api/v1/ws?token=<JWT>` on Gateway
2. WebSocket hub validates JWT, stores connection in `connections[userID]`
3. Client subscribes to channels via `channel.subscribe` message
4. When message created via HTTP POST, handler broadcasts to WebSocket hub
5. Hub looks up channel members, finds connected users, sends `message.new` event
6. Typing indicators and read receipts flow bidirectionally via hub

**File Upload Flow (Special Case):**

1. Client POSTs multipart/form-data to `/api/v1/files/upload` on Gateway
2. Gateway parses multipart directly (not via gRPC due to streaming overhead)
3. Gateway calls `file.Service.Upload()` directly (file service initialized in gateway)
4. File service validates, scans (no-op for now), generates thumbnail if image
5. File service uploads to MinIO, inserts metadata to PostgreSQL
6. Gateway broadcasts `file.uploaded` WebSocket event
7. Response includes file ID and presigned download URL

**State Management:**
- Session state: JWT in client, refresh tokens in PostgreSQL
- Real-time state: WebSocket hub in-memory maps (connections, channel members, user names)
- File storage: MinIO (S3-compatible), metadata in PostgreSQL
- Cache: Redis for rate limiting only (optional, degrades to in-memory if unavailable)

## Key Abstractions

**Repository Interface:**
- Purpose: Decouple business logic from database implementation
- Examples: `backend/internal/crm/contact/repository.go`, `backend/internal/auth/repository.go`
- Pattern: Interface defined in domain package, PostgreSQL implementation in same package, mock implementations in `_test.go` files

**Service:**
- Purpose: Encapsulate domain logic for a bounded context
- Examples: `backend/internal/auth/service.go`, `backend/internal/crm/contact/service.go`, `backend/internal/chat/channel/service.go`
- Pattern: Struct with repository dependencies, public methods for use cases

**Middleware:**
- Purpose: HTTP request/response interception for cross-cutting concerns
- Examples: `backend/internal/middleware/auth.go`, `backend/internal/middleware/ratelimit.go`, `backend/internal/middleware/rbac.go`
- Pattern: Higher-order functions returning `func(http.Handler) http.Handler`, chi-compatible

**TokenMaker:**
- Purpose: JWT creation and validation
- Examples: `backend/internal/auth/token.go`
- Pattern: Struct with secret and expiry config, used by auth service and gateway middleware

**WebSocketHub:**
- Purpose: Manage WebSocket connections and broadcast events
- Examples: `backend/internal/server/websocket.go`
- Pattern: In-memory maps guarded by RWMutex, callback for user info resolution

## Entry Points

**API Gateway:**
- Location: `backend/cmd/gateway/main.go`
- Triggers: HTTP requests on `:8080`, metrics on `:9090`
- Responsibilities: HTTP routing, middleware stack, gRPC client initialization, WebSocket hub, graceful shutdown

**Auth Service:**
- Location: `backend/cmd/auth/main.go`
- Triggers: gRPC calls on `:50051`, health checks on `:9091`
- Responsibilities: User registration/login, JWT issuance, refresh token rotation, RBAC, invitation system

**CRM Service:**
- Location: `backend/cmd/crm/main.go`
- Triggers: gRPC calls on `:50052`, health checks on `:9092`
- Responsibilities: Contact/Company/Deal/Activity management, custom fields, tags, pipeline stages, search, filters, reports

**Chat Service:**
- Location: `backend/cmd/chat/main.go`
- Triggers: gRPC calls on `:50053`, health checks on `:9093`
- Responsibilities: Channel management, message CRUD, DMs, threads, mentions, read receipts, typing indicators, file metadata, full-text search

**Database Migrations:**
- Location: `backend/migrations/*.sql`
- Triggers: `make migrate-up` or Docker Compose `migrate` service
- Responsibilities: Schema versioning, executed by golang-migrate tool

## Error Handling

**Strategy:** Domain errors map to gRPC status codes, which map to HTTP status codes at gateway

**Patterns:**
- Service layer returns sentinel errors (e.g., `ErrUserNotFound`, `ErrInvalidCredentials`)
- gRPC handlers map domain errors to gRPC status codes via `mapError()` function
- Gateway handlers map gRPC errors to HTTP status codes
- Validation errors return immediately without DB access
- Database errors logged with context, returned as internal errors to client
- Graceful degradation: Redis unavailable -> in-memory rate limiting, logged warnings

**Example Flow:**
1. Service: `return nil, contact.ErrEmailExists`
2. gRPC handler: `return nil, status.Error(codes.AlreadyExists, "email already exists")`
3. Gateway handler: Detects `codes.AlreadyExists`, responds `409 Conflict`

## Cross-Cutting Concerns

**Logging:**
- `log/slog` with JSON handler, structured fields
- Context included: `user_id`, `contact_id`, operation names
- No `fmt.Println` allowed per CLAUDE.md

**Validation:**
- Service layer validates inputs (e.g., email format, required fields)
- Early return on validation failure
- Custom field validation via validator engine (`internal/crm/customfield/validator.go`)

**Authentication:**
- JWT access tokens (15m) + opaque refresh tokens (7d)
- Middleware validates JWT locally in gateway, extracts claims
- RBAC via roles (admin/manager/member) and permissions (resource:action)
- gRPC services trust gateway (no re-auth), future: mTLS between services

**Transactions:**
- Repository methods accept `context.Context` for cancellation
- Multi-step operations use `pgx.BeginTx` for atomic commits
- No dual-write (PostgreSQL only, Redis is cache)

**Metrics:**
- Prometheus metrics on dedicated ports (`:9090-9093`)
- gRPC interceptors for request count, latency, errors
- HTTP middleware for request count, response time
- Health checks separate from metrics

---

*Architecture analysis: 2026-02-07*
