---
phase: 09-security-compliance
plan: 02
subsystem: security
tags: [vault, encryption, aes-256-gcm, hkdf, password-policy, entropy, nist, bcrypt]
requires:
  - phase: 09-01
    provides: "security models, migrations, proto definitions, go-password-validator dep"
provides:
  - "AES-256-GCM vault encryption with HKDF key derivation"
  - "Vault service with Get/Set/List/Delete and dedicated TOTP encrypt/decrypt"
  - "Password policy service with entropy-based NIST SP 800-63B validation"
  - "Password history checking with bcrypt for reuse prevention"
affects: [09-03, 09-04, 09-05, 09-06]
tech-stack:
  added: []
  patterns: [hkdf-key-derivation, aes-256-gcm-at-rest, entropy-based-password-validation]
key-files:
  created:
    - backend/internal/security/vault/crypto.go
    - backend/internal/security/vault/service.go
    - backend/internal/security/vault/repository.go
    - backend/internal/security/vault/postgres_repository.go
    - backend/internal/security/password/policy.go
    - backend/internal/security/password/repository.go
    - backend/internal/security/password/postgres_repository.go
  modified: []
key-decisions:
  - "HKDF-SHA256 with nil salt and context-string info for key derivation (KeyContextVault, KeyContextTOTP)"
  - "AES-256-GCM nonce prepended to ciphertext, base64 encoded for storage"
  - "Vault service derives two independent keys from single master secret (min 32 chars)"
  - "Password history uses bcrypt.CompareHashAndPassword for reuse checking"
  - "Default policy returns NIST-aligned defaults when no DB row exists"
  - "go-password-validator GetEntropy for entropy calculation (not Validate, for custom threshold)"
duration: 4min
completed: 2026-02-11
---

# Phase 9 Plan 02: Vault Encryption & Password Policy Summary

AES-256-GCM vault service with HKDF key derivation and NIST-aligned entropy-based password policy with bcrypt history checking.

## One-Liner

Vault encryption (AES-256-GCM + HKDF dual-key derivation) and password policy service (entropy validation + bcrypt history reuse prevention)

## Task Commits

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Vault encryption service with AES-256-GCM and HKDF | ed8e196 | crypto.go, service.go, repository.go, postgres_repository.go |
| 2 | Password policy service with entropy validation | 11d4a21 | policy.go, repository.go, postgres_repository.go |

## What Was Built

### Vault Encryption Service (4 files, 267 lines)

**crypto.go** -- Core cryptographic operations:
- `DeriveKey(masterSecret, context)` using HKDF-SHA256 to produce 32-byte AES-256 keys
- `Encrypt(plaintext, key)` with AES-256-GCM, random 12-byte nonce, base64 output
- `Decrypt(encoded, key)` reverses encrypt: base64 decode, split nonce, GCM open
- Constants: `KeyContextVault` and `KeyContextTOTP` for key purpose separation

**service.go** -- Business logic layer:
- `NewService(repo, masterSecret)` derives vaultKey + totpKey from single master secret
- `GetSecret` / `SetSecret` / `ListSecrets` / `DeleteSecret` for general vault CRUD
- `EncryptTOTPSecret` / `DecryptTOTPSecret` for 2FA secret encryption (separate key)
- SetSecret auto-detects create vs update based on existing key name
- Structured logging (slog) throughout, never logs plaintext or keys

**repository.go** -- Interface with GetByKeyName, List, Create, Update, Delete

**postgres_repository.go** -- PostgreSQL implementation:
- List intentionally omits encrypted_value column (metadata only)
- Delete returns ErrSecretNotFound if no row affected

### Password Policy Service (3 files, 146 lines)

**policy.go** -- Business logic layer:
- `ValidatePassword` checks min length, entropy (via go-password-validator), and optional complexity
- `CheckPasswordHistory` compares new password against last N bcrypt hashes
- `RecordPassword` stores a hash in history for future reuse checks
- `GetPolicy` / `UpdatePolicy` for policy management
- Helper functions for character class detection (uppercase, lowercase, digit, special)

**repository.go** -- Interface with GetPolicy, UpdatePolicy, AddPasswordHistory, GetPasswordHistory

**postgres_repository.go** -- PostgreSQL implementation:
- GetPolicy returns NIST-aligned defaults if no row exists (12 chars, 50 bits, 5 reuse)
- GetPasswordHistory returns hashes ordered newest first with LIMIT

## Decisions Made

1. **HKDF with nil salt** -- Salt is unnecessary when master secret has sufficient entropy (min 32 chars enforced); context string provides key separation
2. **Nonce prepended to ciphertext** -- Standard pattern for AES-GCM; nonce is 12 bytes (GCM standard), ciphertext follows
3. **Dual-key derivation** -- Vault and TOTP keys derived from same master but are cryptographically independent via different HKDF contexts
4. **bcrypt for history comparison** -- Password history stores bcrypt hashes (same as auth service), CompareHashAndPassword for timing-safe comparison
5. **GetEntropy over Validate** -- Using GetEntropy directly allows custom threshold comparison with descriptive error messages
6. **Default policy fallback** -- PostgresRepository returns in-memory defaults when no DB row exists, ensuring service works before migration seed runs

## Deviations from Plan

None -- plan executed exactly as written.

## Next Phase Readiness

- Vault service ready for 2FA TOTP secret encryption (09-03: 2FA setup)
- Password policy service ready for registration/password-change validation (09-04: session management)
- Both services follow standard repository+service pattern for gRPC wiring (09-06)

## Self-Check: PASSED
