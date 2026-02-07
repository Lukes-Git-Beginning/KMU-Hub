# External Integrations

**Analysis Date:** 2026-02-07

## APIs & External Services

**File Storage:**
- MinIO (S3-compatible) - Object storage for chat file uploads and attachments
  - SDK/Client: `minio/minio-go/v7` v7.0.98
  - Auth: `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY` env vars
  - Endpoint: `MINIO_ENDPOINT` (default: localhost:9000)
  - Implementation: `backend/internal/chat/file/minio_store.go`
  - Features: Presigned URLs, file upload/download, thumbnail generation, quota enforcement

**Real-time Communication:**
- WebSocket - Native WebSocket server for chat real-time features
  - SDK/Client: `coder/websocket` v1.8.14
  - Auth: JWT token validation on connection
  - Implementation: `backend/internal/server/websocket.go`
  - Features: Message broadcasting, typing indicators, read receipts, presence

**Monitoring:**
- Prometheus - Metrics collection and monitoring
  - SDK/Client: `prometheus/client_golang` v1.23.2, `grpc-ecosystem/go-grpc-prometheus` v1.2.0
  - Endpoint: Gateway exposes metrics on port 9090
  - Implementation: `backend/internal/metrics/metrics.go`
  - Metrics: HTTP requests, gRPC calls, custom business metrics

## Data Storage

**Databases:**
- PostgreSQL 16
  - Connection: `DATABASE_URL` env var (format: `postgres://user:pass@host:port/db?sslmode=disable`)
  - Client: pgx/v5 v5.8.0 (native Go driver with connection pooling)
  - Migrations: golang-migrate (38 migration files in `backend/migrations/`)
  - Schema: Users, roles, permissions, contacts, companies, deals, activities, channels, messages, files

**File Storage:**
- MinIO (local/docker: minio:9000, production: configurable)
  - Bucket: `MINIO_BUCKET` (default: kmuhub-files)
  - SSL: Configurable via `MINIO_USE_SSL` (default: false for local dev)

**Caching:**
- Redis 7
  - Connection: `REDIS_URL` env var (format: `redis://host:port`)
  - Client: go-redis/v9 v9.17.3
  - Usage: Session storage, rate limiting (with in-memory fallback), caching
  - Implementation: `backend/internal/database/redis.go`

## Authentication & Identity

**Auth Provider:**
- Custom JWT-based authentication
  - Implementation: `backend/internal/auth/` service
  - Access tokens: 15min expiry (configurable via `ACCESS_TOKEN_EXPIRY`)
  - Refresh tokens: 7 days expiry (configurable via `REFRESH_TOKEN_EXPIRY`)
  - Secret: `JWT_SECRET` env var (minimum 32 characters)
  - Features: User registration, login, token refresh, logout, password change, user invitations
  - RBAC: Role-based access control with admin/manager/member roles
  - Permissions: Resource:action format stored in PostgreSQL

## Monitoring & Observability

**Error Tracking:**
- None (currently relies on structured logging)

**Logs:**
- Structured JSON logging via Go stdlib `log/slog`
  - Format: JSON with fields for context, errors, request IDs
  - Output: stdout (captured by container orchestration)
  - Request tracing: Request ID middleware in `backend/internal/middleware/requestid.go`

**Metrics:**
- Prometheus metrics
  - HTTP middleware: `backend/internal/middleware/metrics.go`
  - gRPC interceptor: go-grpc-prometheus
  - Custom metrics: `backend/internal/metrics/metrics.go`

**Health Checks:**
- HTTP health endpoints on each service
  - Gateway: :8080/health
  - Auth: :9091/health
  - CRM: :9092/health
  - Chat: :9093/health
  - Implementation: `backend/internal/health/`

## CI/CD & Deployment

**Hosting:**
- Development: Docker Compose (`deploy/docker/docker-compose.yml`)
- Production (planned): Kubernetes on Hetzner Cloud (EU-only)

**CI Pipeline:**
- GitHub Actions (`.github/workflows/ci.yml`)
  - Jobs: lint, test, build, e2e, openapi-validate
  - Triggers: Push/PR to main or develop branches
  - Go version: 1.25.6
  - Linter: golangci-lint v2.8
  - Coverage: Atomic mode with race detector, report uploaded as artifact
  - E2E: Integration tests with PostgreSQL and Redis service containers

**Container Registry:**
- Not specified (likely GitHub Container Registry or private registry)

**Docker Images:**
- Multi-service architecture with separate Dockerfiles:
  - `backend/Dockerfile.gateway` - API Gateway (HTTP + WebSocket)
  - `backend/Dockerfile.auth` - Auth Service (gRPC)
  - `backend/Dockerfile.crm` - CRM Service (gRPC)
  - `backend/Dockerfile.chat` - Chat Service (gRPC)
  - `backend/Dockerfile.migrate` - Migration runner

## Environment Configuration

**Required env vars:**
- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_URL` - Redis connection string
- `JWT_SECRET` - JWT signing secret (32+ chars)
- `MINIO_ENDPOINT` - MinIO server address
- `MINIO_ACCESS_KEY` - MinIO access key
- `MINIO_SECRET_KEY` - MinIO secret key
- `MINIO_BUCKET` - S3 bucket name
- Service addresses: `AUTH_GRPC_ADDRESS`, `CRM_GRPC_ADDRESS`, `CHAT_GRPC_ADDRESS`

**Optional configuration:**
- `ACCESS_TOKEN_EXPIRY` (default: 15m)
- `REFRESH_TOKEN_EXPIRY` (default: 168h)
- `CORS_ALLOWED_ORIGINS` (default: http://localhost:3000, semicolon-delimited)
- `RATE_LIMIT_RPS` (default: 100)
- `FILE_SIZE_LIMIT_MB` (default: 50)
- `MINIO_USE_SSL` (default: false)

**Secrets location:**
- Environment variables only (never in code or committed files)
- `.env.example` provides template in `backend/` directory
- Production: Container orchestration secrets management (K8s secrets)

## Webhooks & Callbacks

**Incoming:**
- None detected (no webhook endpoints in OpenAPI spec)

**Outgoing:**
- None detected (no external webhook calls in codebase)
- Future consideration: Virus scanning integration point exists in `backend/internal/chat/file/service.go` (VirusScanFunc callback, currently no-op)

## Inter-Service Communication

**gRPC Services:**
- Protocol: gRPC over HTTP/2 (insecure credentials for internal communication)
- Service definitions:
  - `backend/proto/auth/v1/auth.proto` - 18 RPCs (auth, users, roles, invitations)
  - `backend/proto/crm/v1/crm.proto` - CRM entity management
  - `backend/proto/chat/v1/chat.proto` - Chat channels, messages, files, search
- Code generation: protoc with protoc-gen-go and protoc-gen-go-grpc
- Service discovery: Static configuration via environment variables

**API Gateway Pattern:**
- Gateway (`backend/cmd/gateway/`) exposes unified HTTP/REST API
- Proxies requests to backend gRPC services
- OpenAPI specification: `backend/api/openapi.yaml` (v3.0.3)
- Rate limiting: Redis-backed with in-memory fallback
- CORS: Configurable allowed origins
- Middleware: Auth, RBAC, logging, metrics, request ID

---

*Integration audit: 2026-02-07*
