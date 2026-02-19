---
phase: 09-security-compliance
plan: 06
subsystem: api
tags: [grpc, http, gateway, security, 2fa, audit, vault, gdpr, ip-filter, middleware]

# Dependency graph
requires:
  - phase: 09-02
    provides: "Vault encryption, password policy services"
  - phase: 09-03
    provides: "Audit log and session management services"
  - phase: 09-04
    provides: "TOTP 2FA authentication service"
  - phase: 09-05
    provides: "GDPR export/erasure services"
provides:
  - "SecurityService gRPC server (21 RPCs) in auth binary"
  - "HTTP API for all security features (~32 endpoints)"
  - "IP allowlist/blocklist middleware on gateway"
  - "2FA login flow with pending token via HTTP"
  - "Session management HTTP endpoints"
affects: ["09-07", "09-08", "09-09"]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "SecurityGRPCServer bridges SecurityService proto to Go service packages"
    - "SecurityRoutes shares auth gRPC connection (ServiceName 'auth')"
    - "IPFilterMiddleware with 60s cached rules from gRPC"
    - "VaultAdapter bridges context-less vault.Service to ctx-based VaultEncryptor interface"

key-files:
  created:
    - backend/internal/server/security_grpc.go
    - backend/internal/gateway/route_security.go
    - backend/internal/gateway/ip_filter.go
  modified:
    - backend/internal/server/grpc.go
    - backend/internal/gateway/route_auth.go
    - backend/internal/auth/totp.go
    - backend/cmd/auth/main.go
    - backend/cmd/gateway/main.go
    - backend/internal/config/config.go
    - deploy/docker/docker-compose.yml

key-decisions:
  - "SecurityService runs in same binary as AuthService (shared gRPC port)"
  - "VaultAdapter struct bridges vault.Service (no ctx) to auth.VaultEncryptor (with ctx)"
  - "IP filter applied globally before rate limiter (fail-open when rules unavailable)"
  - "IP rules use direct pgx queries in security_grpc.go (no separate service layer)"
  - "2FA endpoints split: validate is public (pending_token), management is protected"

patterns-established:
  - "SecurityRoutes ServiceName returns 'auth' to share gRPC connection"
  - "IPFilterMiddleware caches rules with sync.RWMutex and 60s TTL"
  - "ListTwoFactorPolicies thin service method delegates to repository"

# Metrics
duration: 8min
completed: 2026-02-11
---

# Plan 09-06: gRPC + Gateway Wiring Summary

**SecurityService gRPC server with 21 RPCs, 32 HTTP endpoints for audit/vault/GDPR/password/IP/2FA/sessions, and IP filter middleware**

## Performance

- **Duration:** ~8 min (across context continuation)
- **Started:** 2026-02-11T20:10:00Z (approx)
- **Completed:** 2026-02-11T20:22:54Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments
- SecurityGRPCServer implements all 21 SecurityService RPCs (audit, vault, GDPR, password, IP rules)
- Auth binary initializes vault, audit, password, GDPR services with VAULT_MASTER_SECRET from env
- Gateway exposes ~32 new HTTP endpoints across auth routes (2FA, sessions) and security routes (audit, vault, GDPR, password, IP rules)
- IP filter middleware checks client IPs against configurable allow/block rules with 60s cache
- 2FA login flow returns pending_token for second-factor validation via public endpoint

## Task Commits

Each task was committed atomically:

1. **Task 1: gRPC server + auth service initialization** - `38fa2ad` (feat)
2. **Task 2: Gateway HTTP routes + IP filter middleware** - `3d4ae43` (feat)

## Files Created/Modified
- `backend/internal/server/security_grpc.go` - SecurityServiceServer with 21 RPCs bridging proto to Go services
- `backend/internal/gateway/route_security.go` - 22 HTTP endpoints for audit, vault, GDPR, password, IP rules
- `backend/internal/gateway/ip_filter.go` - IP allow/block middleware with 60s cached rules from gRPC
- `backend/internal/server/grpc.go` - Added 2FA and session gRPC handlers, toSessionInfo/toTwoFactorPolicyProto helpers
- `backend/internal/gateway/route_auth.go` - Extended with 2FA setup/verify/validate/disable, session management endpoints
- `backend/internal/auth/totp.go` - Added ListTwoFactorPolicies service method
- `backend/cmd/auth/main.go` - Vault/audit/password/GDPR service init, SecurityServiceServer registration
- `backend/cmd/gateway/main.go` - SecurityRoutes registrar, IPFilterMiddleware
- `backend/internal/config/config.go` - VAULT_MASTER_SECRET env var
- `deploy/docker/docker-compose.yml` - VAULT_MASTER_SECRET for auth service

## Decisions Made
- SecurityService runs in same binary as AuthService, sharing gRPC port -- avoids a new service for simple CRUD operations
- VaultAdapter struct bridges vault.Service (no context param) to auth.VaultEncryptor interface (with context param) -- clean adapter without modifying vault package
- IP filter middleware applied globally before rate limiter for early rejection of blocked IPs
- IP rules use direct pgx queries in security_grpc.go rather than a separate service layer -- simple CRUD doesn't justify another package
- 2FA validate endpoint is public (uses pending_token), all other 2FA endpoints require JWT auth
- ListTwoFactorPolicies added as thin service method delegating to repository

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed invalid Go construct for entropy calculation**
- **Found during:** Task 1 (security_grpc.go)
- **Issue:** Anonymous function for entropy calculation was unnecessary and incorrectly structured
- **Fix:** Removed the anonymous function block entirely
- **Files modified:** backend/internal/server/security_grpc.go
- **Verification:** go build passes
- **Committed in:** 38fa2ad

**2. [Rule 1 - Bug] Fixed error comparison using string matching instead of errors.Is**
- **Found during:** Task 1 (security_grpc.go)
- **Issue:** mapSecurityError used err.Error() string comparison instead of errors.Is
- **Fix:** Changed all error comparisons to use errors.Is pattern
- **Files modified:** backend/internal/server/security_grpc.go
- **Verification:** go build passes
- **Committed in:** 38fa2ad

**3. [Rule 1 - Bug] Fixed CompleteTwoFactorLogin return type mismatch**
- **Found during:** Task 2 (grpc.go 2FA handlers)
- **Issue:** Used LoginResult return but method returns (*User, *TokenPair, error)
- **Fix:** Updated to use user, tokens, err return pattern
- **Files modified:** backend/internal/server/grpc.go
- **Verification:** go build passes
- **Committed in:** 3d4ae43

**4. [Rule 1 - Bug] Fixed TerminateSession signature mismatch**
- **Found during:** Task 2 (grpc.go session handlers)
- **Issue:** Passed both sessionID and userID but service only takes sessionID
- **Fix:** Only pass sessionID to service, validate userID separately
- **Files modified:** backend/internal/server/grpc.go
- **Verification:** go build passes
- **Committed in:** 3d4ae43

**5. [Rule 1 - Bug] Fixed TerminateAllSessions return type mismatch**
- **Found during:** Task 2 (grpc.go session handlers)
- **Issue:** Expected (count, error) return but service returns only error
- **Fix:** Removed count usage, return empty response
- **Files modified:** backend/internal/server/grpc.go
- **Verification:** go build passes
- **Committed in:** 3d4ae43

**6. [Rule 2 - Missing Critical] Added ListTwoFactorPolicies to auth.Service**
- **Found during:** Task 2 (grpc.go policy handlers)
- **Issue:** Method only existed on repository, not on service layer
- **Fix:** Added thin service method delegating to repo.ListTwoFactorPolicies
- **Files modified:** backend/internal/auth/totp.go
- **Verification:** go build passes
- **Committed in:** 3d4ae43

**7. [Rule 2 - Missing Critical] Added TwoFactorEnabled and Locale to toUserInfo**
- **Found during:** Task 2 (grpc.go user info)
- **Issue:** UserInfo proto has these fields but toUserInfo didn't set them
- **Fix:** Added TwoFactorEnabled and Locale field assignments
- **Files modified:** backend/internal/server/grpc.go
- **Verification:** go build passes
- **Committed in:** 3d4ae43

---

**Total deviations:** 7 auto-fixed (5 bugs, 2 missing critical)
**Impact on plan:** All fixes necessary for correct compilation and runtime behavior. No scope creep.

## Issues Encountered
- Context continuation required due to large scope of changes (21 gRPC RPCs + 32 HTTP endpoints + middleware)
- Multiple method signature mismatches between gRPC handlers and auth service methods discovered at compile time

## User Setup Required

None - no external service configuration required. VAULT_MASTER_SECRET added to docker-compose for development.

## Next Phase Readiness
- All Phase 9 backend security services are now accessible via HTTP API
- Frontend can integrate 2FA setup flow, session management, audit viewer, GDPR export
- Plans 09-07 (i18n), 09-08 (frontend security), 09-09 (integration tests) can proceed
- IP filter middleware is active but allows all traffic when no rules are configured

## Self-Check: PASSED

---
*Phase: 09-security-compliance*
*Completed: 2026-02-11*
