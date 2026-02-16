---
phase: 09-security-compliance
plan: 01
subsystem: security
tags: [proto, migrations, models, otp, vault, gdpr, 2fa, audit, sessions, password-policy, ip-rules]
requires:
  - phase: 01-auth-infrastructure
    provides: "auth.proto base, user model, migrations pattern"
provides:
  - "security.proto with audit/vault/GDPR/password/IP RPCs (17 RPCs)"
  - "auth.proto extended with 2FA, session, and policy RPCs (11 new RPCs)"
  - "10 security database tables with proper indexes"
  - "Go models for all security domain types"
  - "pquerna/otp, go-password-validator dependencies installed"
affects: [09-02, 09-03, 09-04, 09-05, 09-06, 09-07, 09-08, 09-09]
tech-stack:
  added: [pquerna/otp v1.5.0, wagslane/go-password-validator v0.3.0]
  patterns: [security-proto-pattern, hash-chained-audit, vault-encryption]
key-files:
  created:
    - backend/proto/security/v1/security.proto
    - backend/proto/security/v1/security.pb.go
    - backend/proto/security/v1/security_grpc.pb.go
    - backend/internal/models/security.go
    - backend/internal/tools/security_deps.go
    - backend/migrations/000039_create_security_tables.up.sql
    - backend/migrations/000039_create_security_tables.down.sql
    - backend/migrations/000040_extend_users_2fa_locale.up.sql
    - backend/migrations/000040_extend_users_2fa_locale.down.sql
  modified:
    - backend/proto/auth/v1/auth.proto
    - backend/proto/auth/v1/auth.pb.go
    - backend/proto/auth/v1/auth_grpc.pb.go
    - backend/internal/models/user.go
    - backend/go.mod
    - backend/go.sum
    - backend/Makefile
key-decisions:
  - "security.v1.SecurityService as separate proto (not merged into auth) for domain separation"
  - "VaultSecret never exposes encrypted_value in List/Get -- returns decrypted_value only in Get"
  - "Audit log uses BIGSERIAL sequence_num for hash chain ordering (not timestamp)"
  - "tools/security_deps.go holds imports to keep otp+validator in go.mod until service code exists"
  - "GDPR erasure log uses original_user_id (UUID, no FK) since user row gets anonymized"
  - "Default password policy seeded in migration (12 chars, 50 bits entropy, 5 reuse prevention)"
duration: 5min
completed: 2026-02-11
---

# Phase 9 Plan 01: Security Data Foundation Summary

Proto definitions, database migrations, Go models, and dependencies for the entire security and compliance phase.

## One-Liner

Security data foundation: 28 new RPCs across security+auth protos, 10 DB tables, 12 Go model structs, TOTP+password-validator deps

## Task Commits

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Proto definitions for security and auth extensions | aea05f1 | security.proto, auth.proto, generated .pb.go files, Makefile |
| 2 | Database migrations + Go models + dependencies | 0b7e753 | 4 migration files, security.go, user.go, go.mod, go.sum, security_deps.go |

## What Was Built

### Proto Definitions (28 new RPCs total)

**security.v1.SecurityService (17 RPCs):**
- Audit: CreateAuditEntry, ListAuditEntries, ExportAuditLog, VerifyAuditChain
- Vault: GetVaultSecret, SetVaultSecret, ListVaultSecrets, DeleteVaultSecret
- GDPR: RequestDataExport, ListDataExports, ApproveDataExport, DenyDataExport, GetExportDownload, PreviewErasure, ExecuteErasure
- Password: GetPasswordPolicy, UpdatePasswordPolicy, ValidatePassword
- IP Access: ListIPRules, CreateIPRule, DeleteIPRule

**auth.v1.AuthService (11 new RPCs):**
- 2FA: Setup2FA, Verify2FA, Validate2FALogin, Disable2FA, RegenerateRecoveryCodes, AdminReset2FA
- Sessions: ListSessions, TerminateSession, TerminateAllSessions
- Policy: GetTwoFactorPolicy, UpdateTwoFactorPolicy

**LoginResponse extended** with requires_two_factor and pending_token fields for two-step 2FA login flow.
**UserInfo extended** with two_factor_enabled and locale fields.

### Database Tables (10 new tables + user extension)

| Table | Purpose | Key Indexes |
|-------|---------|-------------|
| audit_log | Tamper-evident event log | timestamp DESC, user_id, action, sequence_num DESC, composite timestamp+action |
| recovery_codes | Hashed single-use 2FA backup codes | user_id |
| two_factor_policy | Per-role 2FA enforcement config | UNIQUE role_name |
| user_sessions | Active sessions with device metadata | user_id, last_active_at DESC |
| vault_secrets | Encrypted secret storage | UNIQUE key_name |
| gdpr_export_requests | Data export approval workflow | user_id, status |
| gdpr_erasure_log | Erasure execution records | (no additional indexes, append-only) |
| password_policies | Organization password requirements | (single row, seeded with defaults) |
| password_history | Previous hashes for reuse prevention | user_id |
| ip_access_rules | IP allow/block lists | rule_type |

**Migration 000040:** Added 5 columns to users table (two_factor_enabled, two_factor_secret_encrypted, two_factor_pending_secret, two_factor_enabled_at, locale).

### Go Models (12 structs + constants)

All domain types in `models/security.go`: AuditEntry, AuditFilter, RecoveryCode, TwoFactorPolicy, UserSession, VaultSecret, GDPRExportRequest, GDPRErasureLog, PasswordPolicy, PasswordHistoryEntry, IPAccessRule, LoginResult, ModuleErasurePreview.

User struct in `models/user.go` extended with TwoFactorEnabled, TwoFactorSecretEncrypted (json:"-"), TwoFactorPendingSecret (json:"-"), TwoFactorEnabledAt, Locale.

### Dependencies

- `pquerna/otp` v1.5.0 -- TOTP generation, validation, QR codes
- `wagslane/go-password-validator` v0.3.0 -- NIST-aligned entropy-based password validation
- `golang.org/x/crypto` upgraded v0.47.0 -> v0.48.0 (includes HKDF for key derivation)

## Decisions Made

1. **Separate SecurityService proto** -- Audit, vault, GDPR, password, IP rules in security.v1, while 2FA/sessions stay in auth.v1 (closer to auth domain)
2. **VaultSecret never exposes encrypted_value** -- List returns metadata only, Get returns decrypted_value
3. **BIGSERIAL sequence_num for audit ordering** -- More reliable than timestamp for hash chain integrity
4. **tools/security_deps.go for dependency retention** -- Keeps otp+validator in go.mod before service code imports them
5. **gdpr_erasure_log.original_user_id has no FK** -- User row gets anonymized, so FK would break
6. **Default password policy seeded** -- 12 char min, 50 bits entropy, 5 reuse prevention count

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Created tools/security_deps.go for dependency retention**
- **Found during:** Task 2
- **Issue:** `go mod tidy` removed pquerna/otp and go-password-validator because no Go source files imported them yet
- **Fix:** Created `backend/internal/tools/security_deps.go` with blank imports to retain the dependencies
- **Files created:** backend/internal/tools/security_deps.go

## Next Phase Readiness

All subsequent plans (09-02 through 09-09) can now:
- Import proto-generated types from `securityv1` and `authv1` packages
- Reference Go models from `models.AuditEntry`, `models.VaultSecret`, etc.
- Build service implementations against the defined RPCs
- Run migrations to create all required database tables
- Import `pquerna/otp` and `go-password-validator` for TOTP and password validation

## Self-Check: PASSED
