# Technology Stack

**Analysis Date:** 2026-02-07

## Languages

**Primary:**
- Go 1.25.6 - Backend microservices (auth, CRM, chat, gateway)
- TypeScript 5.7 - Desktop application frontend

**Secondary:**
- SQL - Database migrations and queries
- Protobuf - gRPC service definitions

## Runtime

**Environment:**
- Go 1.25.6 runtime
- Node.js (via Electron 33.0.0)

**Package Manager:**
- Go modules (`go.mod`, `go.sum`)
- npm (lockfile: not detected in repository)

## Frameworks

**Core:**
- chi/v5 v5.2.4 - HTTP routing and middleware for gateway
- gRPC v1.78.0 - Inter-service communication
- React 19.0.0 - Desktop UI framework
- Electron 33.0.0 - Desktop application platform
- React Router DOM 7.0.0 - Desktop navigation

**Testing:**
- testify v1.11.1 - Go test assertions and mocking
- vitest 2.1.0 - Desktop application testing

**Build/Dev:**
- electron-vite 2.4.0 - Desktop build tooling
- Vite (via @vitejs/plugin-react 4.3.0) - Frontend bundling
- protoc-gen-go + protoc-gen-go-grpc - gRPC code generation
- golang-migrate/migrate v4.17.0 - Database migration management
- Tailwind CSS 4.0.0 - Desktop styling framework

## Key Dependencies

**Critical:**
- jackc/pgx/v5 v5.8.0 - PostgreSQL driver and connection pooling
- golang-jwt/jwt/v5 v5.3.1 - JWT token generation and validation
- golang.org/x/crypto v0.47.0 - Bcrypt password hashing
- redis/go-redis/v9 v9.17.3 - Redis client for sessions and caching
- google/uuid v1.6.0 - UUID generation for entity IDs

**Infrastructure:**
- sethvargo/go-envconfig v1.3.0 - Environment variable configuration
- minio/minio-go/v7 v7.0.98 - S3-compatible file storage client
- prometheus/client_golang v1.23.2 - Metrics collection and export
- grpc-ecosystem/go-grpc-prometheus v1.2.0 - gRPC metrics instrumentation
- coder/websocket v1.8.14 - WebSocket for real-time chat features

**Business Logic:**
- shopspring/decimal v1.4.0 - Precise decimal arithmetic for financial calculations
- pemistahl/lingua-go v1.4.0 - Multi-language detection for full-text search
- disintegration/imaging v1.6.2 - Image processing for file thumbnails

## Configuration

**Environment:**
- Configuration via environment variables (parsed with `go-envconfig`)
- Example configuration: `backend/.env.example`
- Required secrets: `JWT_SECRET`, `DATABASE_URL`, `REDIS_URL`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`
- Service ports: Gateway (:8080), Auth (:50051), CRM (:50052), Chat (:50053)
- Health check ports: Auth (9091), CRM (9092), Chat (9093), Gateway metrics (9090)

**Build:**
- Backend: `backend/Makefile` with targets for build, test, lint, migrate, proto
- Desktop: `desktop/package.json` with `electron-vite` scripts
- Linting: `backend/.golangci.yml` (golangci-lint v2.8)
- TypeScript: `desktop/tsconfig.json` (ES2022 target, ESNext modules)

## Platform Requirements

**Development:**
- Go 1.25.6
- Node.js (exact version not pinned)
- PostgreSQL 16
- Redis 7
- MinIO (S3-compatible storage)
- protoc compiler (for gRPC code generation)
- golang-migrate CLI (for database migrations)
- Docker + Docker Compose (for local environment)

**Production:**
- Deployment target: Kubernetes on Hetzner Cloud (SaaS)
- Docker images: Multi-stage builds for gateway, auth, crm, chat services
- Container registry: Not specified
- Self-hosted option: Docker Compose setup in `deploy/docker/`

---

*Stack analysis: 2026-02-07*
