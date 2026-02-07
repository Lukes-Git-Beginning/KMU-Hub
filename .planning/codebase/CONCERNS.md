# Codebase Concerns

**Analysis Date:** 2026-02-07

## Tech Debt

**Sprint 4 (Chat File Sharing + Search) Code-Complete but Uncommitted:**
- Issue: Migrations 000018-000019, file service, search service, MinIO integration all implemented but not committed to git
- Files: `backend/migrations/000018_chat_files.up.sql`, `backend/migrations/000019_chat_search.up.sql`, `backend/internal/chat/file/`, `backend/internal/chat/search/`, `backend/internal/chat/langdetect/`
- Impact: Risk of losing work, can't be deployed, CI doesn't validate it, collaboration blocked
- Fix approach: Commit Sprint 4 work immediately with proper conventional commit message

**Desktop App is Empty Scaffold:**
- Issue: Desktop app has `package.json` with dependencies but `desktop/src/` directory is empty
- Files: `desktop/src/` (empty), `desktop/package.json`
- Impact: Electron + React 19 stack configured but zero implementation, no UI for end users
- Fix approach: Phase 4 implementation (not yet planned), prioritize based on go-to-market strategy (desktop vs web-first)

**NoOpScanner Instead of Real Virus Scanning:**
- Issue: File uploads have no malware protection, `NoOpScanner` always returns nil
- Files: `backend/internal/chat/file/scanner.go` (line 18)
- Impact: Users can upload malicious files, security risk for file sharing feature
- Fix approach: Integrate ClamAV scanner (daemon mode), add configuration for `CLAMAV_HOST`, fallback behavior on scanner unavailability
- Current mitigation: MIME type whitelist, file size limits (50MB default)
- Recommendations: Deploy ClamAV in docker-compose, implement `ClamAVScanner` struct, add scanner health checks

**WebSocket Authentication via Query Parameter:**
- Issue: JWT passed as `?token=...` in WebSocket URL, tokens leak in logs and browser history
- Files: `backend/internal/server/websocket.go` (line 89)
- Impact: Access tokens exposed in server logs, browser history, proxy logs
- Fix approach: Switch to `Sec-WebSocket-Protocol` header-based auth or connection upgrade with cookie
- Current mitigation: Access tokens expire after 15 minutes
- Recommendations: Implement secure WebSocket auth pattern before production

**Redis Failure Falls Back to In-Memory Rate Limiter:**
- Issue: Rate limiter silently falls back to in-memory map when Redis is down, no persistence across gateway restarts
- Files: `backend/internal/middleware/ratelimit.go` (line 45, 88-119), `backend/internal/database/redis.go` (line 20)
- Impact: Rate limits reset on gateway restart during Redis outage, DDoS vulnerability during failover
- Fix approach: Add alerting when fallback is active, consider rejecting requests if Redis is critical
- Current mitigation: Best-effort Redis ping logged as warning, fallback prevents total service outage
- Recommendations: Add Prometheus metric for `ratelimit_fallback_active`, alert on sustained fallback

**Large Generated Protobuf Files:**
- Issue: Generated `.pb.go` files are massive (8k+ lines for CRM, 4k+ for Chat)
- Files: `backend/proto/crm/v1/crm.pb.go` (8246 lines), `backend/proto/chat/v1/chat.pb.go` (4093 lines)
- Impact: Slow IDE performance, large git diffs, review difficulty
- Fix approach: Split proto definitions into smaller logical files (e.g., `crm/v1/contacts.proto`, `crm/v1/deals.proto`)
- Current mitigation: Files are generated, not hand-written
- Recommendations: Refactor proto definitions in Phase 4, consider breaking CRM service into smaller services

**No Monitoring/Alerting Implementation:**
- Issue: Prometheus metrics port configured (`METRICS_PORT=:9090`) but no actual metrics exported
- Files: `backend/internal/config/config.go` (line 30)
- Impact: No observability into production systems, can't diagnose performance issues or outages
- Fix approach: Add prometheus/client_golang, instrument critical paths (request duration, DB query times, gRPC call duration, rate limit hits)
- Recommendations: Implement before production deployment, add Grafana dashboards for key metrics

## Known Bugs

**None detected** - No open bugs found in code comments or error handling. Test coverage suggests major flows work correctly.

## Security Considerations

**JWT Secret Required but No Rotation Mechanism:**
- Risk: Compromised JWT secret requires redeployment of all services, no key rotation support
- Files: `backend/internal/config/config.go` (line 14), `backend/internal/auth/token.go`
- Current mitigation: Secret required at startup (fails if missing), SHA-256 hashed refresh tokens in DB
- Recommendations: Implement JWT key rotation with kid (key ID) support, multiple active keys for zero-downtime rotation

**No SQL Injection Protection Audit:**
- Risk: Most queries use pgx parameterized queries, but 618 instances of `if err != nil` indicate complex query construction
- Files: All `*_postgres_repository.go` files across 9+ packages
- Current mitigation: Repository pattern uses pgx with `$1` placeholders throughout
- Recommendations: Add `gosec` SQL injection scan to CI (currently excludes G404, G115 only), code review all dynamic query builders

**CORS Wildcards in Development:**
- Risk: Default `CORS_ALLOWED_ORIGINS=http://localhost:3000` safe, but no validation for production wildcards
- Files: `backend/internal/config/config.go` (line 26)
- Current mitigation: Must explicitly set env var, no `*` default
- Recommendations: Add validation to reject `*` in production environments, document allowed origins per deployment

**No Rate Limiting on WebSocket Connections:**
- Risk: HTTP endpoints have rate limiting (100 RPS default), but WebSocket connections have no per-user limits
- Files: `backend/internal/server/websocket.go` (no rate limit check), `backend/internal/middleware/ratelimit.go` (HTTP only)
- Current mitigation: WebSocket requires valid JWT, auth service rate-limits token issuance
- Recommendations: Add per-user WebSocket connection limit, message send rate limit via Redis counter

**MinIO Credentials in Plaintext Config:**
- Risk: MinIO access/secret keys stored in environment variables, no secrets manager integration
- Files: `backend/internal/config/config.go` (lines 36-38), `deploy/docker/docker-compose.yml` (lines 32-33, 142-143)
- Current mitigation: Default credentials only for local development, production requires env override
- Recommendations: Integrate HashiCorp Vault or cloud secrets manager before production, rotate credentials regularly

## Performance Bottlenecks

**PostgreSQL Connection Pool Hardcoded to 25:**
- Problem: Max 25 connections regardless of deployment size or load
- Files: `backend/internal/database/postgres.go` (line 16)
- Cause: No environment variable for pool size configuration
- Improvement path: Add `DATABASE_MAX_CONNS` and `DATABASE_MIN_CONNS` env vars, scale based on deployment (5-50 range)

**Full-Text Search Without Materialized View:**
- Problem: Search queries scan all messages/files with `to_tsvector()` on each request
- Files: `backend/internal/chat/search/postgres_repository.go`, `backend/internal/crm/search/postgres_repository.go`, migrations `000013` (CRM), `000019` (Chat)
- Cause: GIN indexes on `ts_vector` generated columns, but no caching of search results
- Improvement path: Add materialized view for frequent searches, consider Elasticsearch for >1M messages, add search result pagination

**No Caching Layer for Frequently-Read Data:**
- Problem: Every request hits PostgreSQL, no Redis caching for hot data (user profiles, channel memberships, pipeline stages)
- Files: All repository layers (`*_postgres_repository.go`)
- Cause: No cache-aside pattern implementation
- Improvement path: Add Redis cache with TTL for read-heavy endpoints (GetUser, GetChannel, ListPipelineStages), implement cache invalidation on writes

**WebSocket Hub Uses In-Memory Maps:**
- Problem: `connections`, `channelMembers`, `userNames` maps in `WebSocketHub` don't scale horizontally
- Files: `backend/internal/server/websocket.go` (lines 28-34)
- Cause: Single-gateway assumption, no distributed pubsub
- Improvement path: Migrate to Redis pubsub for multi-gateway deployments, use Redis sets for channel membership tracking
- Current capacity: ~10,000 concurrent connections per gateway (estimated)
- Scaling path: Add NATS or Redis pubsub, deploy multiple gateways behind load balancer

**Language Detection on Every Message:**
- Problem: `lingua-go` library runs on every message insert to detect German/English/French/Italian/Spanish
- Files: `backend/internal/chat/langdetect/detector.go`, `backend/internal/chat/message/service.go`
- Cause: Search requires language-specific `ts_config` but detection is CPU-heavy
- Improvement path: Cache detected language per channel (most channels are single-language), use simpler heuristic for short messages

## Fragile Areas

**Gateway HTTP Handler Monolith:**
- Files: `backend/internal/server/http.go` (3353 lines)
- Why fragile: Single file with 100+ endpoints across auth/CRM/chat domains, hard to navigate and test
- Safe modification: Use grep to find specific handler before editing, add integration tests for changed routes
- Test coverage: HTTP handlers have NO unit tests (only E2E for auth endpoints)
- Recommendations: Split into `http_auth.go`, `http_crm.go`, `http_chat.go`, add handler unit tests with mocked gRPC clients

**gRPC Error Mapping Across 3 Services:**
- Files: `backend/internal/server/auth_grpc.go`, `backend/internal/server/crm_grpc.go` (2215 lines, 101 error conversions), `backend/internal/server/chat_grpc.go` (1035 lines, 66 error conversions)
- Why fragile: Manual error code mapping from service errors to gRPC status codes, easy to forget or mismatch
- Safe modification: Add test case for new error type in service layer, verify error code in E2E test
- Test coverage: gRPC server implementations have NO unit tests
- Recommendations: Create error mapping table test, consider centralized error converter

**Migration Ordering:**
- Files: `backend/migrations/` (38 migration files, 19 up/down pairs)
- Why fragile: 19 migrations applied sequentially, any failure leaves DB in broken state
- Safe modification: Always test migration up+down locally before commit, never edit old migrations
- Test coverage: Migrations tested in CI via `migrate up`, no down testing
- Recommendations: Add `migrate down && migrate up` to CI, test migrations on production-like dataset before deploy

**Custom Field Validation Logic:**
- Files: `backend/internal/crm/customfield/service.go`, `backend/internal/models/custom_fields.go`
- Why fragile: Complex validation for 7 field types (text, number, date, boolean, select, multiselect, url) with type-specific rules
- Safe modification: Add test case for new field type or validation rule, ensure JSON schema validation passes
- Test coverage: Service layer 100% tested, but edge cases (timezone handling, number overflow) not covered

**WebSocket Message Type Dispatch:**
- Files: `backend/internal/server/websocket.go` (lines 50-68, message type constants)
- Why fragile: String-based message type switching, typo = silent failure or wrong handler
- Safe modification: Add new message type to constants, handle in switch statement, add integration test
- Test coverage: WebSocket hub has NO tests
- Recommendations: Add WebSocket integration tests, consider protobuf messages instead of JSON + string types

## Scaling Limits

**Single Gateway Instance:**
- Current capacity: ~10,000 concurrent WebSocket connections (estimated based on in-memory hub)
- Limit: In-memory connection map, no distributed pubsub
- Scaling path: Add Redis pubsub or NATS, deploy multiple gateways behind Nginx/HAProxy, session affinity not required

**PostgreSQL Primary-Only:**
- Current capacity: 25-connection pool, single primary, no read replicas
- Limit: Write throughput ~5k TPS (depends on hardware), read throughput limited by connection pool
- Scaling path: Add read replicas for search/reporting queries, use pgBouncer for connection pooling, consider Aurora PostgreSQL for auto-scaling

**MinIO Single-Node:**
- Current capacity: Docker single-node MinIO, no replication or erasure coding
- Limit: Single disk failure = data loss, no high availability
- Scaling path: Deploy MinIO in distributed mode (4+ nodes), enable erasure coding, or migrate to S3-compatible cloud storage (AWS S3, Cloudflare R2)

**No CDN for File Downloads:**
- Current capacity: All file downloads proxied through gateway -> MinIO
- Limit: Gateway bandwidth becomes bottleneck for large files (videos, archives)
- Scaling path: Use MinIO presigned URLs directly (already implemented), add CDN (CloudFront, Cloudflare) in front of MinIO/S3

## Dependencies at Risk

**React 19 is Bleeding Edge:**
- Risk: React 19.0.0 released recently, ecosystem libraries may not be compatible
- Impact: Desktop app development blocked by incompatible UI libraries
- Migration plan: Monitor React 19 adoption, downgrade to React 18 if needed, wait for ecosystem maturity

**golang-migrate CLI Dependency:**
- Risk: Migrations require external `migrate` CLI tool, not part of Go build
- Impact: Deployment requires `migrate` binary in PATH, version mismatch risk
- Migration plan: Embed migrations in Go binary using `golang-migrate/migrate/v4` library, run migrations from service startup

**No Dependabot or Dependency Scanning:**
- Risk: 100+ Go dependencies, 20+ npm dependencies, no automated security updates
- Impact: Vulnerable dependencies go unnoticed until incident
- Migration plan: Enable GitHub Dependabot, add `govulncheck` to CI, run `npm audit` in desktop CI

## Missing Critical Features

**LiveKit Video Calls Not Implemented:**
- Problem: CLAUDE.md and docs mention LiveKit for video, but no code exists
- Blocks: Video calling, screen sharing, conferencing (Phase 5+ feature)
- Files: Config stubs in `CLAUDE.md` lines 246-248, no actual integration
- Recommendation: Defer to Phase 5 (Chat Video/Calls), not blocking for beta

**No Plugin/WASM System:**
- Problem: USP mentions "Config/WASM-Plugin-System" for customization, but no plugin loader exists
- Blocks: Customer-specific customizations without forking codebase
- Recommendation: Phase 6+ feature, build after core product validated

**No Email Notifications:**
- Problem: No email service, no SMTP configuration, no notification templates
- Blocks: User invitations require manual token sharing, no @mention notifications, no deal stage change alerts
- Recommendation: Add email service in Sprint 5 (Chat Notifications) or Phase 4, integrate SendGrid/AWS SES

**No File Preview/Thumbnails for Documents:**
- Problem: Thumbnail generation only for images (JPEG/PNG/GIF), no PDF/DOCX preview
- Blocks: User experience for document-heavy workflows
- Files: `backend/internal/chat/file/thumbnail.go` (stub for image thumbnails only)
- Recommendation: Add PDF thumbnail generation (first page), consider third-party preview service (Filestack, Cloudinary)

**No Audit Logs:**
- Problem: No audit trail for sensitive actions (user role changes, data deletion, permission changes)
- Blocks: Compliance requirements (GDPR audit trail, security incident investigation)
- Recommendation: Add audit log table, log all mutations with (user_id, action, entity_type, entity_id, timestamp, changes_json)

## Test Coverage Gaps

**HTTP Handlers Have Zero Unit Tests:**
- What's not tested: All 100+ HTTP handlers in `backend/internal/server/http.go`
- Files: `backend/internal/server/http.go` (3353 lines), no corresponding `*_test.go`
- Risk: Breaking changes in HTTP layer not caught until E2E, JSON serialization bugs, error response format drift
- Priority: High - add handler tests with mocked gRPC clients

**gRPC Server Implementations Have Zero Unit Tests:**
- What's not tested: Auth gRPC (733 lines), CRM gRPC (2215 lines), Chat gRPC (1035 lines)
- Files: `backend/internal/server/auth_grpc.go`, `backend/internal/server/crm_grpc.go`, `backend/internal/server/chat_grpc.go`
- Risk: Error mapping bugs, protobuf conversion errors, validation bypasses
- Priority: High - add gRPC server tests with mocked service layers

**WebSocket Hub Has Zero Tests:**
- What's not tested: WebSocket connection management, message broadcasting, channel subscriptions, typing indicators
- Files: `backend/internal/server/websocket.go` (487 lines)
- Risk: Subscription leaks, broadcast failures, connection cleanup bugs
- Priority: Medium - add integration tests with WebSocket client

**Repository Layers Have Low Coverage:**
- What's not tested: Most repository implementations (only service layers tested)
- Files: `backend/internal/*/postgres_repository.go` (14 repository files)
- Risk: SQL query bugs, transaction handling errors, constraint violations not handled
- Priority: Medium - integration tests with testcontainers-go for database

**E2E Tests Only Cover Auth Flow:**
- What's not tested: CRM endpoints, chat endpoints, file uploads, WebSocket connections
- Files: `backend/test/e2e/auth_test.go` (556 lines), no other E2E tests
- Risk: Integration failures between services not caught in CI
- Priority: Medium - add E2E tests for critical user flows (create contact, send message, upload file)

**No Load/Performance Tests:**
- What's not tested: System behavior under load, concurrent user limits, database query performance
- Files: No load test scripts exist
- Risk: Production performance issues discovered by users, scaling problems not identified
- Priority: Low - add before beta launch, use k6 or vegeta for load testing

---

*Concerns audit: 2026-02-07*
