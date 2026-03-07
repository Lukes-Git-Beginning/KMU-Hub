---
tags: [security, auth, compliance]
updated: 2026-03-05
---
# Security & Compliance

## Authentifizierung
- JWT Access Token: 15min Expiry, Claims: user_id, roles, permissions
- Opaque Refresh Token: 7 Tage, SHA-256 gehasht in DB, Rotation + Theft Detection
- Passwort: bcrypt cost=12
- 2FA: TOTP (RFC 6238), Grace Period, Enforcement nach Ablauf

## RBAC
- Rollen: admin, manager, member
- Permissions: `resource:action` Pattern (z.B. `contacts:write`, `deals:delete`)
- Middleware: `RequireRole(roles...)`, `RequirePermission(resource, action)`
- 403 Forbidden bei unzureichenden Rechten

## Middleware-Stack (Reihenfolge)
1. CORS → 2. Rate Limiting → 3. Auth (JWT) → 4. RBAC
- Code: `backend/internal/middleware/`

## CORS
- Explizite Allowlist via `CORS_ALLOWED_ORIGINS` (Semikolon-getrennt)
- Erlaubte Headers: Authorization, Content-Type, X-Request-ID
- Methoden: GET, POST, PUT, DELETE, OPTIONS
- Credentials: true, Max-Age: 86400 (1 Tag Preflight-Cache)
- KEINE Wildcards

## Rate Limiting
- Redis-basiert mit In-Memory-Fallback
- Key: `ratelimit:{user_id_or_ip}`, 1-Sekunden Sliding Window
- Default: 100 rps (`RATE_LIMIT_RPS`)
- Response: 429 mit `Retry-After: 1`

## Vault Service (Secrets)
- Verschluesselte Secrets in PostgreSQL
- `VAULT_MASTER_SECRET` (32+ Zeichen) als Env-Var
- Verwendet fuer: OAuth-Tokens (Bexio, DATEV), API-Keys (Lexware), Email-Passwoerter

## Electron Token-Persistence
- `safeStorage.encryptString()` fuer verschluesselte Speicherung
- Datei: `app.getPath('userData')/tokens.enc`
- Fallback: Plaintext auf Linux ohne Keyring
- Geladen beim App-Start → `useAuthStore.initialize()`

## Input Validation
- Prepared Statements durchgehend (kein String-Concatenation)
- Request-Parsing + Validierung pro Endpoint
- Email-Validierung, Passwort-Strength-Checks

## GDPR / Datenschutz
- Audit-Logging: `security_audit_logs` Tabelle
- Erasure-Support implementiert
- **OFFEN (Phase B Blocker):** AVV/DPA, AGB, DSGVO-Pruefung durch Anwalt

## Verwandte Notes
- [[architektur]] — Service-Architektur
- [[api]] — Endpoints & Auth-Flow
- [[deployment]] — Infrastruktur-Security
