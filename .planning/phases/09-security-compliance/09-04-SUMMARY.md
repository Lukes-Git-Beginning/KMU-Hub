---
phase: 09-security-compliance
plan: 04
subsystem: security
tags: [2fa, totp, recovery-codes, pending-token, enforcement, login-flow]
requires:
  - phase: 09-01
    provides: "security migrations, models, pquerna/otp dependency"
provides:
  - "Complete TOTP 2FA service: setup, verify, disable, admin reset"
  - "Recovery code generation and validation (8 codes, SHA-256 hashed)"
  - "Login flow with 2FA pending token (5-min JWT)"
  - "Per-role 2FA enforcement with grace period"
  - "CompleteTwoFactorLogin for 2FA code validation"
affects: [09-05, 09-06, 09-08]
tech-stack:
  added: []
  patterns: [vault-encryptor-interface, pending-token-flow, transactional-enable]
key-files:
  created:
    - backend/internal/auth/totp.go
  modified:
    - backend/internal/auth/errors.go
    - backend/internal/auth/repository.go
    - backend/internal/auth/postgres_repository.go
    - backend/internal/auth/service.go
    - backend/internal/auth/service_test.go
    - backend/internal/auth/token.go
    - backend/internal/server/grpc.go
key-decisions:
  - "VaultEncryptor interface for at-rest encryption (nil = dev fallback, no encryption)"
  - "Login returns LoginResult instead of (User, TokenPair) for 2FA pending flow"
  - "PendingToken is 5-min JWT with type=2fa_pending claim"
  - "Recovery codes: 8 codes, 10 hex chars each, SHA-256 hashed"
  - "2FA enforcement grace period calculated from user.CreatedAt"
  - "AdminReset2FA requires non-empty reason for audit trail"
  - "Enable2FA is transactional (user update + recovery codes insert)"
duration: 6min
completed: 2026-02-11
---

# Phase 9 Plan 04: TOTP Two-Factor Authentication Summary

Complete TOTP 2FA backend: setup wizard, verification, recovery codes, login flow with pending tokens, per-role enforcement, and admin reset.

## One-Liner

TOTP 2FA service with pquerna/otp, 8 recovery codes, 5-min pending token login flow, per-role enforcement with grace period

## Task Commits

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | TOTP service with setup, verification, and recovery codes | c63cefe | totp.go, errors.go, repository.go, postgres_repository.go, service.go, service_test.go |
| 2 | Login flow modification with pending token | 3af6408 | token.go, service.go, service_test.go, grpc.go |

## What Was Built

### TOTP Service (totp.go)

New file with all 2FA service methods on the existing `auth.Service` struct:

| Method | Purpose |
|--------|---------|
| Setup2FA | Generate TOTP key via pquerna/otp, QR code PNG, encrypt & store pending secret |
| Verify2FA | Validate first TOTP code, generate 8 recovery codes, enable 2FA transactionally |
| Validate2FALogin | Validate TOTP code or recovery code during login |
| Disable2FA | Disable 2FA after validating TOTP code confirmation |
| RegenerateRecoveryCodes | Replace all recovery codes (requires TOTP confirmation) |
| AdminReset2FA | Admin-only disable with mandatory reason logging |
| Check2FAEnforcement | Check role policies and grace period calculation |
| GetTwoFactorPolicy | Get 2FA policy for a role |
| UpdateTwoFactorPolicy | Upsert 2FA policy with enforcement and grace period |

**VaultEncryptor interface:** `EncryptTOTPSecret` / `DecryptTOTPSecret` for at-rest encryption. When nil (development), secrets stored as plaintext. Set via `SetVaultService()`.

### Login Flow Modification

**Before:** `Login()` returned `(*models.User, *models.TokenPair, error)`
**After:** `Login()` returns `(*models.LoginResult, error)` with:
- If 2FA disabled: `LoginResult{User, AccessToken, RefreshToken}`
- If 2FA enabled: `LoginResult{RequiresTwoFactor: true, PendingToken: "..."}`
- If 2FA enforcement grace expired: returns `Err2FAEnforcementRequired`

**New method:** `CompleteTwoFactorLogin(ctx, pendingToken, code, isRecoveryCode)` validates the 2FA pending token, verifies the TOTP/recovery code, and issues full token pair.

### Pending Token (token.go)

- `PendingClaims` struct with `type: "2fa_pending"` claim
- 5-minute expiry (hardcoded, short-lived)
- `CreatePendingToken(userID)` and `ValidatePendingToken(tokenStr)` on TokenMaker

### Repository Extensions

**10 new interface methods:**
- 2FA: `StorePending2FASecret`, `GetPending2FASecret`, `Enable2FA`, `Disable2FA`
- Recovery codes: `GetRecoveryCodes`, `UseRecoveryCode`, `ReplaceRecoveryCodes`
- Policy: `GetTwoFactorPolicy`, `ListTwoFactorPolicies`, `UpsertTwoFactorPolicy`

**PostgreSQL implementations:** `Enable2FA` uses `BEGIN/COMMIT` transaction (update user + delete old codes + insert new codes). `UpsertTwoFactorPolicy` uses `INSERT ON CONFLICT DO UPDATE`.

### Error Sentinels

8 new error sentinels: `ErrTwoFactorAlreadyEnabled`, `ErrTwoFactorNotEnabled`, `ErrNo2FASetupPending`, `ErrInvalidTOTPCode`, `ErrInvalidRecoveryCode`, `ErrAllRecoveryCodesUsed`, `ErrInvalidPendingToken`, `ErrPendingTokenExpired`, `Err2FAEnforcementRequired`.

### gRPC Server Updates

- Login handler updated for `LoginResult` return type with 2FA pending response
- 8 new error mappings in `mapError` for all 2FA errors

## Decisions Made

1. **VaultEncryptor as optional interface** -- Allows dev mode without vault, production plugs in vault service via `SetVaultService()`
2. **Login signature change to LoginResult** -- Breaking change from `(*User, *TokenPair, error)` to `(*LoginResult, error)` for clean 2FA flow
3. **5-minute pending token** -- Short-lived JWT prevents replay attacks; user must complete 2FA within 5 minutes
4. **8 recovery codes, 10 hex chars** -- 5 bytes of entropy per code, SHA-256 hashed at rest
5. **Grace period from CreatedAt** -- New users have N days to set up 2FA before enforcement blocks login
6. **TOTP code required for disable/regenerate** -- Prevents unauthorized 2FA manipulation even with valid session

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated mock repository in service_test.go for new interface methods**
- **Found during:** Task 1
- **Issue:** Parallel plan 09-03 added session methods to Repository interface; mock in tests didn't implement them
- **Fix:** Added stub implementations for all session mock methods and all 2FA mock methods
- **Files modified:** service_test.go

**2. [Rule 3 - Blocking] Updated gRPC server Login handler for new signature**
- **Found during:** Task 2
- **Issue:** grpc.go Login handler used old 3-return Login signature
- **Fix:** Updated to use LoginResult, added 2FA pending response path, added 2FA error mappings
- **Files modified:** backend/internal/server/grpc.go

**3. [Rule 3 - Blocking] Updated existing test callers for new Login signature**
- **Found during:** Task 2
- **Issue:** TestService_RefreshToken, TestService_Logout, TestService_RefreshToken_InactiveUser all called Login with old signature
- **Fix:** Updated to use `result, _ := svc.Login(...)` pattern
- **Files modified:** service_test.go

## Next Phase Readiness

- 2FA service methods are ready for gRPC wiring in plan 09-05/09-06
- CompleteTwoFactorLogin ready for gateway integration
- VaultEncryptor interface ready for vault service implementation
- All existing tests pass with updated Login signature

## Self-Check: PASSED
