# Phase 9: Security & Compliance - Research

**Researched:** 2026-02-11
**Domain:** Security (2FA, audit logging, DSGVO, sessions, vault), Internationalization (i18n)
**Confidence:** HIGH

## Summary

This phase hardens the existing KMU Hub application with enterprise security features and DSGVO compliance, plus adds multi-language support for the Swiss market. The codebase is well-structured with a clean gateway-to-gRPC-microservice pattern, existing auth service with JWT+RBAC, and PostgreSQL as the single source of truth. All new features extend the existing auth service and gateway patterns.

The standard approach for each sub-domain is:
- **2FA:** `pquerna/otp` (Go) for TOTP generation/validation, QR code generation via its built-in image support, encrypted TOTP secret storage in PostgreSQL
- **Audit log:** Append-only PostgreSQL table with SHA-256 hash chaining for tamper evidence, indexed for fast search/filter
- **DSGVO:** Background job pattern for data export (ZIP with JSON per module), cascading anonymization via per-module handlers
- **Sessions:** Extend existing refresh_tokens table with device/IP metadata, admin list/terminate API
- **Vault:** AES-256-GCM encryption using Go's standard `crypto/cipher`, master key from env var, encrypted values in PostgreSQL
- **i18n:** `react-i18next` + `i18next-icu` for ICU message format support (pluralization, gender, ordinals), with `react-intl` as the stronger alternative given the ICU requirement

**Primary recommendation:** Extend the existing auth gRPC service with new RPCs for 2FA, sessions, audit, and DSGVO. Create a new `security` domain package in the backend. Use `react-intl` (FormatJS) for i18n since ICU message format is a first-class citizen there (no plugin needed), which aligns better with the locked decision for full ICU support.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- 2FA: Guided wizard flow (multi-step modal: QR code -> scan -> verify -> recovery codes)
- 8 single-use recovery codes, downloadable/copyable, regeneratable
- Per-role 2FA enforcement with configurable grace period
- Admin can reset 2FA for a user (logged with mandatory reason)
- Login flow: password -> TOTP prompt if 2FA enabled
- Audit log: security events + admin actions + data access events
- Audit presentation: searchable/filterable table (timestamp, user, action, target, IP, result)
- Audit export: CSV and JSON formats
- Audit retention: admin-configurable 1-10 years, default 3 years
- Tamper-evident: append-only with integrity verification
- DSGVO export: ZIP with structured JSON per module, metadata only for files
- DSGVO export trigger: user request -> admin approve/deny -> download link
- Right-to-erasure: two-step with preview, admin confirms with password
- Anonymization: configurable per module (anonymize vs full delete), placeholder "Geloeschter Benutzer #NNN"
- Session management: admin views all sessions (device/IP/location), terminate individual or all
- Secret vault: encrypted storage for API keys, SMTP passwords, integration credentials
- i18n: DE, FR, IT, EN languages
- Language picker in user profile/settings, persists per user
- Browser-detected language on first use, fallback chain: user -> browser -> DE
- Translation scope: UI labels + system content + default values
- Full ICU message format: pluralization, gender, ordinals
- Locale-aware formatting: dates, numbers, currencies

### Claude's Discretion
- Specific TOTP library choice (backend and frontend)
- Audit log database schema and indexing strategy
- ICU/i18n library choice (react-intl, i18next, etc.)
- Exact tamper-evidence implementation (hash chaining, etc.)
- Session storage and cleanup strategy
- Vault encryption algorithm and key management approach
- Password policy specifics (min length, complexity rules)

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `pquerna/otp` | v1.5.0 | TOTP generation, validation, QR code | De facto Go TOTP library, 1333+ importers, RFC 6238 compliant, Google Authenticator compatible |
| `react-intl` (FormatJS) | v8.1.3 | React i18n with ICU message format | ICU message format is a first-class citizen (no plugin needed), smaller bundle (17.8 kB), static message extraction, strong TypeScript |
| `crypto/cipher` (Go stdlib) | Go 1.25 | AES-256-GCM encryption for vault | Standard library, no external dependency, battle-tested, used by HashiCorp Vault internally |
| `wagslane/go-password-validator` | latest | Entropy-based password strength validation | NIST SP 800-63B aligned, entropy-based (not arbitrary complexity rules), lightweight |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `@formatjs/cli` | latest | Static extraction of translation messages from source code | Build step to extract all `<FormattedMessage>` and `intl.formatMessage()` calls |
| `intl-messageformat` | latest | ICU message format parsing (included with react-intl) | Implicit dependency of react-intl for runtime message compilation |
| `encoding/csv` (Go stdlib) | Go 1.25 | CSV export for audit logs | Audit log CSV export for accountants/auditors |
| `archive/zip` (Go stdlib) | Go 1.25 | ZIP archive creation for DSGVO data export | Package user data into downloadable ZIP |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `react-intl` | `react-i18next` + `i18next-icu` | i18next is more flexible and has a larger ecosystem, but ICU is a plugin (not native), requires extra dependency (i18next-icu), and the project specifically requires full ICU format support |
| `wagslane/go-password-validator` | `nbutton23/zxcvbn-go` | zxcvbn provides more detailed feedback (pattern matching) but is heavier; go-password-validator is simpler entropy-based approach aligned with NIST recommendations |
| Custom hash chaining | immudb | immudb provides a separate tamper-proof ledger database, but is a heavy infrastructure dependency for what can be solved with PostgreSQL hash chaining |

**Installation (Backend):**
```bash
cd backend
go get github.com/pquerna/otp@v1.5.0
go get github.com/wagslane/go-password-validator
```

**Installation (Frontend):**
```bash
cd desktop
npm install react-intl @formatjs/cli
```

## Architecture Patterns

### Backend: New Security Domain Package

```
backend/internal/
  auth/
    service.go          # EXTEND: add 2FA methods, session methods
    repository.go       # EXTEND: add 2FA, session, password policy interfaces
    postgres_repository.go  # EXTEND: implement new repository methods
    token.go            # EXTEND: add 2FA-aware claims (twofa_verified flag)
    totp.go             # NEW: TOTP setup, validation, recovery codes
    session.go          # NEW: session tracking, device detection
    errors.go           # EXTEND: new error sentinels
  security/
    audit/
      service.go        # NEW: audit log service
      repository.go     # NEW: audit log repository interface
      postgres_repository.go  # NEW: append-only with hash chaining
      models.go         # NEW: audit entry, filter types
      export.go         # NEW: CSV/JSON export logic
    vault/
      service.go        # NEW: encrypt/decrypt secrets
      repository.go     # NEW: vault repository interface
      postgres_repository.go  # NEW: encrypted storage
      crypto.go         # NEW: AES-256-GCM encrypt/decrypt helpers
    gdpr/
      service.go        # NEW: data export orchestration, erasure orchestration
      export.go         # NEW: per-module export handlers
      erasure.go        # NEW: per-module anonymization handlers
    password/
      policy.go         # NEW: password policy validation, history checking
```

### Gateway: New Route Registrars

```
backend/internal/gateway/
  route_security.go     # NEW: audit log, session management, vault, GDPR routes
```

The security routes follow the established RouteRegistrar pattern (like route_auth.go). Audit/session/vault/GDPR endpoints are admin-protected. 2FA endpoints extend the existing auth routes.

### Frontend: i18n Layer

```
desktop/src/renderer/src/
  i18n/
    index.ts            # IntlProvider setup, language detection, provider wrapper
    messages/
      de.json           # German translations (primary)
      en.json           # English translations
      fr.json           # French translations
      it.json           # Italian translations
    formats.ts          # Locale-specific date/number/currency formats
    useLocale.ts        # Hook for current locale + formatting utilities
  stores/
    locale.ts           # NEW: user language preference (Zustand, persisted)
```

### Proto: Extend Auth Service

```protobuf
// New RPCs in auth.v1.AuthService:
rpc Setup2FA(Setup2FARequest) returns (Setup2FAResponse);
rpc Verify2FA(Verify2FARequest) returns (Verify2FAResponse);
rpc Validate2FA(Validate2FARequest) returns (Validate2FAResponse);
rpc Disable2FA(Disable2FARequest) returns (Disable2FAResponse);
rpc RegenerateRecoveryCodes(RegenerateRecoveryCodesRequest) returns (RegenerateRecoveryCodesResponse);
rpc AdminReset2FA(AdminReset2FARequest) returns (AdminReset2FAResponse);

rpc ListSessions(ListSessionsRequest) returns (ListSessionsResponse);
rpc TerminateSession(TerminateSessionRequest) returns (TerminateSessionResponse);
rpc TerminateAllSessions(TerminateAllSessionsRequest) returns (TerminateAllSessionsResponse);

// New separate service: security.v1.SecurityService
rpc CreateAuditEntry(CreateAuditEntryRequest) returns (CreateAuditEntryResponse);
rpc ListAuditEntries(ListAuditEntriesRequest) returns (ListAuditEntriesResponse);
rpc ExportAuditLog(ExportAuditLogRequest) returns (ExportAuditLogResponse);

rpc RequestDataExport(RequestDataExportRequest) returns (RequestDataExportResponse);
rpc ApproveDataExport(ApproveDataExportRequest) returns (ApproveDataExportResponse);
rpc ExecuteErasure(ExecuteErasureRequest) returns (ExecuteErasureResponse);
rpc PreviewErasure(PreviewErasureRequest) returns (PreviewErasureResponse);

rpc GetVaultSecret(GetVaultSecretRequest) returns (GetVaultSecretResponse);
rpc SetVaultSecret(SetVaultSecretRequest) returns (SetVaultSecretResponse);
rpc ListVaultSecrets(ListVaultSecretsRequest) returns (ListVaultSecretsResponse);
rpc DeleteVaultSecret(DeleteVaultSecretRequest) returns (DeleteVaultSecretResponse);
```

### Pattern 1: Two-Step Login with 2FA

**What:** Login returns a partial auth state when 2FA is enabled, requiring a second TOTP verification step before issuing tokens.
**When to use:** Every login when user has 2FA enabled.

```go
// Login flow modification:
// 1. Validate credentials (existing)
// 2. Check if user has 2FA enabled
// 3. If yes: return a temporary "2fa_pending" token (short-lived, ~5min)
//    instead of access+refresh tokens
// 4. Client submits TOTP code with pending token
// 5. If valid: issue full access+refresh tokens

func (s *Service) Login(ctx context.Context, email, password string) (*models.LoginResult, error) {
    user, err := s.validateCredentials(ctx, email, password)
    if err != nil {
        return nil, err
    }

    if user.TwoFactorEnabled {
        pendingToken, err := s.tokenMaker.CreatePendingToken(user.ID)
        if err != nil {
            return nil, err
        }
        return &models.LoginResult{
            RequiresTwoFactor: true,
            PendingToken:      pendingToken,
        }, nil
    }

    tokens, err := s.createTokenPair(ctx, user)
    return &models.LoginResult{
        User:   user,
        Tokens: tokens,
    }, err
}
```

### Pattern 2: Hash-Chained Audit Log

**What:** Each audit log entry includes a SHA-256 hash of the previous entry, creating a chain that makes tampering detectable.
**When to use:** All audit log inserts.

```go
// Audit entry structure:
type AuditEntry struct {
    ID           uuid.UUID
    Timestamp    time.Time
    UserID       uuid.UUID
    Action       string      // e.g., "user.login", "user.2fa_enabled", "data.export"
    Target       string      // e.g., "user:uuid", "contact:uuid"
    TargetType   string      // e.g., "user", "contact", "session"
    Details      string      // JSON details
    IPAddress    string
    UserAgent    string
    Result       string      // "success" or "failure"
    PreviousHash string      // SHA-256 hash of previous entry
    EntryHash    string      // SHA-256 hash of this entry (computed from all fields + PreviousHash)
}

// Hash computation:
func computeEntryHash(entry *AuditEntry, previousHash string) string {
    data := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s",
        entry.Timestamp.UTC().Format(time.RFC3339Nano),
        entry.UserID,
        entry.Action,
        entry.Target,
        entry.Details,
        entry.IPAddress,
        entry.Result,
        previousHash,
    )
    hash := sha256.Sum256([]byte(data))
    return hex.EncodeToString(hash[:])
}
```

### Pattern 3: AES-256-GCM Vault Encryption

**What:** Secrets stored in the vault are encrypted with AES-256-GCM using a master key derived from an environment variable.
**When to use:** All vault read/write operations.

```go
import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "io"
)

func Encrypt(plaintext []byte, key []byte) (string, error) {
    block, err := aes.NewCipher(key) // key must be 32 bytes for AES-256
    if err != nil {
        return "", err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }

    ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(encoded string, key []byte) ([]byte, error) {
    ciphertext, err := base64.StdEncoding.DecodeString(encoded)
    if err != nil {
        return nil, err
    }

    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonceSize := gcm.NonceSize()
    if len(ciphertext) < nonceSize {
        return nil, fmt.Errorf("ciphertext too short")
    }

    nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
    return gcm.Open(nil, nonce, ciphertext, nil)
}
```

### Pattern 4: react-intl Setup for Multi-Language

**What:** Configure react-intl with ICU message format, locale detection, and runtime switching.
**When to use:** App initialization and all UI text rendering.

```tsx
// i18n/index.tsx
import { IntlProvider } from 'react-intl'
import { useLocaleStore } from '@/stores/locale'
import deMessages from './messages/de.json'
import enMessages from './messages/en.json'
import frMessages from './messages/fr.json'
import itMessages from './messages/it.json'

const messages: Record<string, Record<string, string>> = {
  de: deMessages,
  en: enMessages,
  fr: frMessages,
  it: itMessages,
}

function detectBrowserLocale(): string {
  const browserLang = navigator.language.split('-')[0]
  if (['de', 'en', 'fr', 'it'].includes(browserLang)) return browserLang
  return 'de' // fallback
}

export function I18nProvider({ children }: { children: React.ReactNode }) {
  const locale = useLocaleStore((s) => s.locale) ?? detectBrowserLocale()

  return (
    <IntlProvider
      locale={locale}
      messages={messages[locale]}
      defaultLocale="de"
      onError={(err) => {
        // Suppress missing translation errors in development
        if (err.code !== 'MISSING_TRANSLATION') throw err
      }}
    >
      {children}
    </IntlProvider>
  )
}
```

```tsx
// Usage in components:
import { useIntl, FormattedMessage, FormattedDate, FormattedNumber } from 'react-intl'

function AuditLogEntry({ entry }) {
  const intl = useIntl()

  return (
    <div>
      <FormattedDate value={entry.timestamp} dateStyle="medium" timeStyle="short" />
      <FormattedMessage
        id="audit.action.login"
        defaultMessage="{user} hat sich angemeldet"
        values={{ user: entry.userName }}
      />
    </div>
  )
}

// ICU pluralization:
// messages/de.json:
// "audit.results": "{count, plural, =0 {Keine Ergebnisse} one {# Ergebnis} other {# Ergebnisse}}"
```

### Anti-Patterns to Avoid
- **Storing TOTP secrets in plaintext:** Always encrypt the TOTP secret in the database using the vault encryption pattern. A database breach would compromise all 2FA.
- **Audit log in application memory:** Never buffer audit entries. Write synchronously to PostgreSQL in the same transaction as the action being audited, or immediately after.
- **Rolling your own TOTP algorithm:** Use `pquerna/otp` which handles clock skew, time step windows, and RFC compliance.
- **Hardcoded translations:** Never embed UI text directly in components. Every user-visible string must go through the i18n system from day one.
- **Blocking data export:** DSGVO exports can be large. Use a background job pattern (PostgreSQL-based or goroutine), not synchronous API calls.
- **Single master key for everything:** Vault master key should only decrypt vault entries. TOTP secrets get their own encryption key (or derive from master key with different context).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| TOTP generation/validation | Custom HMAC-SHA1 + time step | `pquerna/otp` | RFC 6238 has subtle edge cases (clock skew, time drift, counter windows), QR code generation included |
| Password strength checking | Regex complexity rules | `wagslane/go-password-validator` entropy check | NIST SP 800-63B recommends entropy-based, not complexity rules. "P@ssw0rd!" passes complexity rules but is weak |
| AES encryption | Custom crypto wrapper | Go stdlib `crypto/cipher` + `crypto/rand` | Standard library is audited, well-tested. Custom crypto wrappers introduce bugs |
| ICU message formatting | Custom pluralization logic | `react-intl` / FormatJS | ICU has complex CLDR rules (Arabic has 6 plural forms, etc.), react-intl handles all CLDR locales |
| CSV generation | Manual string concatenation | Go stdlib `encoding/csv` | Handles escaping, quoting, special characters properly |
| ZIP archive creation | Manual file concatenation | Go stdlib `archive/zip` | Proper ZIP format with compression, file metadata |
| QR code generation | QR code library | `pquerna/otp` Key.Image() | Built into the TOTP library, generates proper otpauth:// URI |
| Browser language detection | Custom user-agent parsing | `navigator.language` + react-intl locale negotiation | Browser API handles regional variants (de-CH, de-AT, etc.) |

**Key insight:** Security features have the highest cost for bugs. Use battle-tested libraries for every crypto/auth primitive. Custom implementations introduce subtle vulnerabilities that are hard to detect in testing.

## Common Pitfalls

### Pitfall 1: 2FA Secret Exposure in API Responses
**What goes wrong:** TOTP secret is returned in API responses after setup, allowing interception.
**Why it happens:** Developers treat the TOTP secret like a regular field.
**How to avoid:** Return the TOTP secret exactly once (during setup), as a QR code image and manual entry string. Never include it in subsequent API responses. Store it encrypted.
**Warning signs:** TOTP secret appearing in GET /user/me or similar profile endpoints.

### Pitfall 2: Audit Log Gaps During Errors
**What goes wrong:** Failed operations are not logged because the error short-circuits before the audit write.
**Why it happens:** Audit logging is added as an afterthought, not as a middleware/interceptor pattern.
**How to avoid:** Use a defer-based pattern or middleware that logs both success and failure outcomes. Log BEFORE returning errors, not in the success path only.
**Warning signs:** Missing "failure" entries in audit log for operations that can fail.

### Pitfall 3: Hash Chain Breaks on Concurrent Writes
**What goes wrong:** Two audit entries are written simultaneously, both using the same "previous hash," breaking the chain.
**Why it happens:** No serialization of audit writes.
**How to avoid:** Use a PostgreSQL advisory lock or serialize writes through a single goroutine channel. Alternatively, use a SERIAL sequence number and compute hashes including the sequence.
**Warning signs:** Two entries with the same `previous_hash` value.

### Pitfall 4: DSGVO Anonymization Misses Foreign Keys
**What goes wrong:** User data is anonymized in the users table but references remain in chat messages, CRM activities, calendar events, etc.
**Why it happens:** Incomplete mapping of user data across modules.
**How to avoid:** Create an explicit registry of every table/column that references user data. Test anonymization by running it and then searching for any trace of the original user data.
**Warning signs:** User's real name still appearing in chat threads or CRM notes after anonymization.

### Pitfall 5: i18n Translation Key Drift
**What goes wrong:** Code references translation keys that don't exist in all locale files, causing runtime fallbacks or empty strings.
**Why it happens:** Developers add keys to one locale file but forget others.
**How to avoid:** Use `@formatjs/cli extract` to generate a canonical list of all message IDs from source code. CI check that all locale files contain all extracted keys. Use `defaultMessage` in source as the German fallback.
**Warning signs:** Empty strings or English text appearing in German UI.

### Pitfall 6: Session Metadata Leaking PII
**What goes wrong:** IP addresses and device information stored in session metadata become subject to DSGVO data export/deletion themselves.
**Why it happens:** Session tracking creates new PII that must be covered by DSGVO compliance.
**How to avoid:** Include session data in DSGVO export. Anonymize session data when a user is deleted. Consider IP hashing after a retention period.
**Warning signs:** Session table not included in DSGVO export or anonymization handler.

### Pitfall 7: Vault Key Rotation Without Re-Encryption
**What goes wrong:** Master key is rotated but existing encrypted values are not re-encrypted with the new key.
**Why it happens:** Key rotation procedure only updates the key, not the data.
**How to avoid:** Store a key version/ID alongside each encrypted value. On key rotation, implement a background re-encryption of all values. Support reading with old key during transition.
**Warning signs:** Decryption failures after key rotation.

## Code Examples

### TOTP Setup Flow (Backend)

```go
// Source: pquerna/otp documentation + KMU Hub patterns
import (
    "crypto/rand"
    "encoding/hex"
    "image/png"
    "bytes"

    "github.com/pquerna/otp/totp"
)

func (s *Service) Setup2FA(ctx context.Context, userID uuid.UUID) (*TwoFactorSetup, error) {
    user, err := s.repo.GetUserByID(ctx, userID)
    if err != nil {
        return nil, ErrUserNotFound
    }

    if user.TwoFactorEnabled {
        return nil, ErrTwoFactorAlreadyEnabled
    }

    // Generate TOTP key
    key, err := totp.Generate(totp.GenerateOpts{
        Issuer:      "KMU Hub",
        AccountName: user.Email,
        Algorithm:   otp.AlgorithmSHA1, // Compatible with most authenticator apps
    })
    if err != nil {
        return nil, fmt.Errorf("generate totp: %w", err)
    }

    // Generate QR code image
    var buf bytes.Buffer
    img, err := key.Image(200, 200)
    if err != nil {
        return nil, fmt.Errorf("generate qr: %w", err)
    }
    if err := png.Encode(&buf, img); err != nil {
        return nil, fmt.Errorf("encode qr: %w", err)
    }

    // Encrypt and store the secret temporarily (not yet verified)
    encryptedSecret, err := s.vault.Encrypt([]byte(key.Secret()))
    if err != nil {
        return nil, fmt.Errorf("encrypt secret: %w", err)
    }

    if err := s.repo.StorePending2FASecret(ctx, userID, encryptedSecret); err != nil {
        return nil, err
    }

    return &TwoFactorSetup{
        QRCodePNG:    buf.Bytes(),
        ManualSecret: key.Secret(),
    }, nil
}

func (s *Service) Verify2FA(ctx context.Context, userID uuid.UUID, code string) ([]string, error) {
    // Get pending secret
    encryptedSecret, err := s.repo.GetPending2FASecret(ctx, userID)
    if err != nil {
        return nil, ErrNo2FASetupPending
    }

    secretBytes, err := s.vault.Decrypt(encryptedSecret)
    if err != nil {
        return nil, fmt.Errorf("decrypt secret: %w", err)
    }

    // Validate the code
    if !totp.Validate(code, string(secretBytes)) {
        return nil, ErrInvalidTOTPCode
    }

    // Generate recovery codes
    recoveryCodes := generateRecoveryCodes(8)
    hashedCodes := make([]string, len(recoveryCodes))
    for i, code := range recoveryCodes {
        hashedCodes[i] = HashToken(code)
    }

    // Enable 2FA: move secret from pending to active, store recovery codes
    if err := s.repo.Enable2FA(ctx, userID, encryptedSecret, hashedCodes); err != nil {
        return nil, err
    }

    slog.Info("2FA enabled", "user_id", userID)
    return recoveryCodes, nil
}

func generateRecoveryCodes(count int) []string {
    codes := make([]string, count)
    for i := range codes {
        b := make([]byte, 5) // 10 hex chars
        rand.Read(b)
        codes[i] = hex.EncodeToString(b)
    }
    return codes
}
```

### Audit Log Repository (Backend)

```go
// Append-only audit log with hash chaining
func (r *PostgresAuditRepository) Create(ctx context.Context, entry *AuditEntry) error {
    // Get the hash of the most recent entry (with advisory lock to serialize)
    tx, err := r.pool.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)

    // Advisory lock to serialize audit writes (prevents hash chain breaks)
    if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", auditLockID); err != nil {
        return fmt.Errorf("advisory lock: %w", err)
    }

    var previousHash string
    err = tx.QueryRow(ctx,
        "SELECT entry_hash FROM audit_log ORDER BY sequence_num DESC LIMIT 1",
    ).Scan(&previousHash)
    if err != nil && err != pgx.ErrNoRows {
        return err
    }

    entry.PreviousHash = previousHash
    entry.EntryHash = computeEntryHash(entry, previousHash)

    _, err = tx.Exec(ctx, `
        INSERT INTO audit_log (id, timestamp, user_id, action, target, target_type,
            details, ip_address, user_agent, result, previous_hash, entry_hash)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
        entry.ID, entry.Timestamp, entry.UserID, entry.Action, entry.Target,
        entry.TargetType, entry.Details, entry.IPAddress, entry.UserAgent,
        entry.Result, entry.PreviousHash, entry.EntryHash,
    )
    if err != nil {
        return err
    }

    return tx.Commit(ctx)
}
```

### DSGVO Data Export Handler Pattern

```go
// Per-module export handler interface
type DataExportHandler interface {
    ModuleName() string
    ExportUserData(ctx context.Context, userID uuid.UUID) ([]byte, error) // Returns JSON
}

// Orchestrator collects data from all modules
func (s *GDPRService) ExecuteExport(ctx context.Context, requestID uuid.UUID) error {
    request, err := s.repo.GetExportRequest(ctx, requestID)
    if err != nil {
        return err
    }

    var buf bytes.Buffer
    zipWriter := zip.NewWriter(&buf)

    for _, handler := range s.exportHandlers {
        data, err := handler.ExportUserData(ctx, request.UserID)
        if err != nil {
            slog.Error("export handler failed", "module", handler.ModuleName(), "error", err)
            continue
        }

        w, err := zipWriter.Create(fmt.Sprintf("%s.json", handler.ModuleName()))
        if err != nil {
            return err
        }
        w.Write(data)
    }

    zipWriter.Close()

    // Store the ZIP and generate a download link
    return s.repo.StoreExportResult(ctx, requestID, buf.Bytes())
}
```

### react-intl ICU Message Examples

```json
// messages/de.json
{
  "common.save": "Speichern",
  "common.cancel": "Abbrechen",
  "common.delete": "Loeschen",
  "common.search": "Suchen",

  "audit.results": "{count, plural, =0 {Keine Ergebnisse} one {# Ergebnis} other {# Ergebnisse}}",
  "audit.action.login": "{user} hat sich angemeldet",
  "audit.action.login_failed": "Fehlgeschlagener Anmeldeversuch fuer {email}",
  "audit.action.2fa_enabled": "{user} hat 2FA aktiviert",

  "session.current": "Aktuelle Sitzung",
  "session.last_active": "Zuletzt aktiv {time}",
  "session.terminate": "Sitzung beenden",
  "session.terminate_all": "Alle anderen Sitzungen beenden",

  "security.2fa.title": "Zwei-Faktor-Authentifizierung",
  "security.2fa.not_enabled": "2FA nicht aktiviert",
  "security.2fa.enable_description": "Schuetze dein Konto mit einem zweiten Faktor",
  "security.2fa.scan_qr": "Scanne den QR-Code mit deiner Authenticator-App",
  "security.2fa.enter_code": "Code eingeben",
  "security.2fa.recovery_codes": "Backup-Codes",
  "security.2fa.recovery_description": "Bewahre diese Codes sicher auf. Jeder Code kann einmalig verwendet werden.",

  "gdpr.export_title": "Datenexport",
  "gdpr.export_description": "Alle deine personenbezogenen Daten herunterladen",
  "gdpr.delete_title": "Datenloesch-Antrag",
  "gdpr.delete_description": "Beantrage die Loeschung deiner personenbezogenen Daten",
  "gdpr.anonymized_user": "Geloeschter Benutzer #{id}",

  "settings.language.title": "Sprache & Region",
  "settings.language.subtitle": "Sprache, Zeitzone und Datumsformat einstellen"
}
```

## Database Schema Design

### Audit Log Table

```sql
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sequence_num BIGSERIAL NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id UUID REFERENCES users(id),
    action VARCHAR(100) NOT NULL,
    target VARCHAR(255),
    target_type VARCHAR(50),
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    result VARCHAR(20) NOT NULL DEFAULT 'success',
    previous_hash VARCHAR(64) NOT NULL DEFAULT '',
    entry_hash VARCHAR(64) NOT NULL
);

-- Primary query patterns
CREATE INDEX idx_audit_log_timestamp ON audit_log (timestamp DESC);
CREATE INDEX idx_audit_log_user_id ON audit_log (user_id);
CREATE INDEX idx_audit_log_action ON audit_log (action);
CREATE INDEX idx_audit_log_result ON audit_log (result);
CREATE INDEX idx_audit_log_sequence ON audit_log (sequence_num DESC);

-- Composite index for filtered queries (date range + action type)
CREATE INDEX idx_audit_log_timestamp_action ON audit_log (timestamp DESC, action);
```

Confidence: HIGH -- Standard PostgreSQL indexing patterns. JSONB for details allows flexible structured data without schema migration for every new event type.

### 2FA Tables

```sql
-- Add to users table
ALTER TABLE users ADD COLUMN two_factor_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN two_factor_secret_encrypted TEXT;
ALTER TABLE users ADD COLUMN two_factor_pending_secret TEXT;
ALTER TABLE users ADD COLUMN two_factor_enabled_at TIMESTAMPTZ;

-- Recovery codes (hashed, single-use)
CREATE TABLE recovery_codes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash VARCHAR(64) NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_recovery_codes_user_id ON recovery_codes (user_id);

-- 2FA enforcement policy
CREATE TABLE two_factor_policy (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    role_name VARCHAR(50) NOT NULL,
    enforced BOOLEAN NOT NULL DEFAULT false,
    grace_period_days INT NOT NULL DEFAULT 14,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES users(id)
);

CREATE UNIQUE INDEX idx_two_factor_policy_role ON two_factor_policy (role_name);
```

### Session Tracking

```sql
-- Extend refresh_tokens or create separate sessions table
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_id UUID REFERENCES refresh_tokens(id) ON DELETE SET NULL,
    device_name VARCHAR(255),
    device_type VARCHAR(50),  -- 'desktop', 'mobile', 'tablet'
    ip_address INET,
    location VARCHAR(255),    -- derived from IP
    user_agent TEXT,
    is_current BOOLEAN NOT NULL DEFAULT false,
    last_active_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_sessions_user_id ON user_sessions (user_id);
CREATE INDEX idx_user_sessions_last_active ON user_sessions (last_active_at DESC);
```

### Vault Secrets

```sql
CREATE TABLE vault_secrets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key_name VARCHAR(255) NOT NULL,
    encrypted_value TEXT NOT NULL,
    key_version INT NOT NULL DEFAULT 1,
    description TEXT,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_vault_secrets_key_name ON vault_secrets (key_name);
```

### DSGVO Export Requests

```sql
CREATE TABLE gdpr_export_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',  -- pending, approved, denied, processing, ready, downloaded, expired
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_by UUID REFERENCES users(id),
    reviewed_at TIMESTAMPTZ,
    review_note TEXT,
    export_data BYTEA,  -- ZIP content (or reference to file storage)
    download_token VARCHAR(64),
    download_expires_at TIMESTAMPTZ,
    downloaded_at TIMESTAMPTZ
);

CREATE INDEX idx_gdpr_exports_user_id ON gdpr_export_requests (user_id);
CREATE INDEX idx_gdpr_exports_status ON gdpr_export_requests (status);

-- Erasure log (separate from audit for DSGVO retention requirements)
CREATE TABLE gdpr_erasure_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    original_user_id UUID NOT NULL,
    anonymized_label VARCHAR(100) NOT NULL,  -- "Geloeschter Benutzer #NNN"
    executed_by UUID NOT NULL REFERENCES users(id),
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    modules_affected JSONB NOT NULL,  -- {"contacts": "anonymized", "chat": "content_deleted", ...}
    confirmation_hash VARCHAR(64) NOT NULL  -- Hash of the erasure action for non-repudiation
);
```

### Password Policy

```sql
CREATE TABLE password_policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    min_length INT NOT NULL DEFAULT 12,
    require_uppercase BOOLEAN NOT NULL DEFAULT false,
    require_lowercase BOOLEAN NOT NULL DEFAULT false,
    require_digit BOOLEAN NOT NULL DEFAULT false,
    require_special BOOLEAN NOT NULL DEFAULT false,
    min_entropy FLOAT NOT NULL DEFAULT 50.0,  -- bits of entropy
    max_age_days INT,                          -- NULL = no expiration
    prevent_reuse_count INT NOT NULL DEFAULT 5,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID REFERENCES users(id)
);

-- Password history for reuse prevention
CREATE TABLE password_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_password_history_user_id ON password_history (user_id);
```

### User Language Preference

```sql
ALTER TABLE users ADD COLUMN locale VARCHAR(5) NOT NULL DEFAULT 'de';
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| SMS-based 2FA | TOTP authenticator apps | 2020+ | SMS is vulnerable to SIM swapping; TOTP is offline and more secure |
| Complex password rules (uppercase/special required) | Entropy-based + breach database check | NIST SP 800-63B (2017, updated 2024) | Complex rules lead to predictable patterns; entropy measures actual strength |
| Custom i18n with string maps | ICU MessageFormat with CLDR data | React ecosystem 2020+ | Proper handling of pluralization rules across all languages (Arabic has 6 forms) |
| Separate audit DB (immudb, etc.) | PostgreSQL with hash chaining | Pragmatic choice for SMB apps | External audit DB adds infrastructure complexity; hash chaining in PostgreSQL is sufficient for KMU-scale compliance |

**Deprecated/outdated:**
- SMS-based 2FA: SIM swapping attacks make it insecure
- Password complexity rules without entropy: NIST explicitly recommends against arbitrary complexity requirements
- gettext/PO files for web i18n: ICU MessageFormat is the modern standard

## Design Branch Integration

### Files to Cherry-Pick

| File | Commit | Integration Approach |
|------|--------|---------------------|
| `InfrastrukturPage.tsx` (security tab) | `8030869` | Cherry-pick, extract SecurityTab, wire to audit/sessions API |
| `VaultSettings.tsx` | `d09bcdd` | Cherry-pick from `desktop/design-reference/`, adapt to actual vault API |
| `SettingsPage.tsx` (security + language + privacy tabs) | `8030869` | Cherry-pick, wire SecurityTab to 2FA/sessions API, LanguageTab to react-intl, PrivacyTab to GDPR API |
| `PrivacySettingsTab.tsx` | `8030869` | Cherry-pick, wire to GDPR export/delete API |
| `stores/settings.ts` | `8030869` | Cherry-pick, replace mock 2FA/security state with actual API hooks |

### Integration Pattern (Same as Phase 7/8)

1. Cherry-pick design files onto main
2. Replace Zustand mock stores with TanStack Query hooks calling real API
3. Wire mock button handlers to actual API calls
4. Add missing Italian language option (design only has DE/EN/FR, requirement includes IT)

## Open Questions

1. **IP Geolocation for Sessions**
   - What we know: Session management requires showing location from IP
   - What's unclear: Which geolocation database/service to use (MaxMind GeoLite2 is common but requires license)
   - Recommendation: Use a simple IP-to-country lookup (free MaxMind GeoLite2 Country database) for MVP. City-level precision is not critical for session management. Could also derive from Hetzner's network info or skip geolocation initially and just show IP + country code.

2. **Audit Log Partitioning Strategy**
   - What we know: Retention is 1-10 years, potentially millions of entries
   - What's unclear: When to implement table partitioning (PostgreSQL native range partitioning by month/year)
   - Recommendation: Start without partitioning. Add PostgreSQL range partitioning by month when table exceeds 10M rows. The schema design (timestamp as primary query dimension) supports adding partitioning later without code changes.

3. **DSGVO Export File Storage**
   - What we know: Export produces a ZIP that user downloads
   - What's unclear: Whether to store the ZIP in PostgreSQL (BYTEA) or MinIO
   - Recommendation: Use PostgreSQL BYTEA for MVP since exports are per-user (small, ~1-10 MB max). Move to MinIO if exports grow large. Add a download expiry (e.g., 7 days) and auto-cleanup.

4. **Vault Master Key Derivation**
   - What we know: Master key comes from environment variable
   - What's unclear: Whether to use the raw env var or derive a key using a KDF (HKDF, scrypt)
   - Recommendation: Use HKDF (Go's `golang.org/x/crypto/hkdf`) to derive separate keys for different purposes (vault encryption, TOTP secret encryption) from a single master secret. This provides key separation without multiple env vars.

## Sources

### Primary (HIGH confidence)
- `pquerna/otp` v1.5.0 - [Go Packages](https://pkg.go.dev/github.com/pquerna/otp), [GitHub](https://github.com/pquerna/otp) - TOTP generation, validation, QR code API
- `react-intl` v8.1.3 - [FormatJS docs](https://formatjs.github.io/docs/react-intl/) - ICU message format, React integration
- Go `crypto/cipher` - [Go stdlib docs](https://pkg.go.dev/crypto/cipher) - AES-256-GCM implementation
- Existing codebase analysis - auth service (service.go, repository.go, token.go), gateway (route_auth.go, route_registrar.go), models, migrations, proto definitions

### Secondary (MEDIUM confidence)
- [Locize blog: react-intl vs react-i18next comparison](https://www.locize.com/blog/react-intl-vs-react-i18next/) - Feature comparison, bundle sizes
- [i18next ICU format docs](https://react.i18next.com/misc/using-with-icu-format) - ICU plugin setup
- [Permify: TOTP in Go](https://permify.co/post/two-factor-authentication-2fa-totp-golang/) - Implementation patterns
- [AES-GCM in Go (Medium)](https://karbhawono.medium.com/encryption-using-aes-gcm-b981bf4890f3) - Encrypt/decrypt code patterns
- [GDPR Right to Erasure guide (Jetico)](https://jetico.com/blog/how-right-erasure-applied-under-gdpr-complete-guide-organizational-compliance/) - Compliance requirements

### Tertiary (LOW confidence)
- `wagslane/go-password-validator` - [GitHub](https://github.com/wagslane/go-password-validator) - Version/API details from GitHub only, not verified via docs
- Audit log hash chaining patterns - Assembled from academic papers and PostgreSQL community discussions, not from a single authoritative implementation guide

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - pquerna/otp is well-established (1333+ importers), react-intl is maintained by FormatJS/Meta, Go stdlib crypto is authoritative
- Architecture: HIGH - Follows established codebase patterns (gateway+gRPC, service/repository layers, RouteRegistrar interface)
- Database schema: HIGH - Standard PostgreSQL patterns, follows existing migration conventions
- Pitfalls: MEDIUM - Compiled from multiple sources and experience, not from a single authoritative guide
- Design integration: HIGH - Verified design branch files exist, checked content structure

**Research date:** 2026-02-11
**Valid until:** 2026-03-11 (stable domain, libraries are mature)
