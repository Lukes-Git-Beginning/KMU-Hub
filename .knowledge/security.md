---
tags: [security, auth, compliance, gdpr]
updated: 2026-04-09
---
# Security & Compliance

## Authentifizierung
- JWT Access Token: 15min Expiry, Claims: user_id, roles, permissions
- Opaque Refresh Token: 7 Tage, SHA-256 gehasht in DB, Rotation + Theft Detection
- Passwort: bcrypt cost=12
- 2FA: TOTP (RFC 6238), Grace Period, Enforcement nach Ablauf

## RBAC
- Rollen: admin, manager, member
- Permissions: `resource:action` Pattern (z.B. `contacts:write`, `deals:delete`, `dialer:campaigns:read`)
- Middleware: `RequireRole(roles...)`, `RequirePermission(resource, action)`
- 403 Forbidden bei unzureichenden Rechten

## Middleware-Stack (Reihenfolge)
1. Metrics → 2. RequestID → 3. SecurityHeaders → 4. Logging → 5. CORS → 6. IP Filter → 7. Rate Limiting → 8. Audit Logger → 9. Auth (JWT) → 10. RBAC
- Code: `backend/internal/middleware/`

## Audit Logger (2026-04-08)
- Buffered Channel (Kapazitaet 1000) + Worker Pool (10 Worker)
- Non-blocking: Events werden bei vollem Channel gedroppt (mit Warning-Log)
- Erfasst: mutating requests (POST/PUT/PATCH/DELETE) auf Security-relevanten Pfaden
- Sendet via gRPC an Security-Service (`CreateAuditEntry`)
- `Start(10)` + `defer Close()` in `gateway/main.go`

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

### Implementiert
- Audit-Logging: `security_audit_logs` Tabelle — vollstaendig aktiv
- Erasure-Support: GDPR-Loeschbegehren via `gdpr_deletion_requests` Tabelle (status: pending/completed)
- GDPR-Dateiexport: `/api/v1/security/gdpr/export` + `/gdpr/exports` + `/gdpr/download/{token}`
- Security-Routen teilen den "auth" gRPC-Server (kein separater Service noetig)
- **Consent Management (Migration 060):**
  - `consent_records`: Einwilligungen pro Kontakt (6 Typen: marketing_email, marketing_phone, profiling, newsletter, data_processing, data_sharing)
  - Legal Basis: consent, legitimate_interest, contract, legal_obligation
  - IP-Adresse, Quelle, Zeitstempel fuer Audit-Trail
  - CRM Extended Routes: `/api/v1/contacts/…/consent`

### Offen (Phase C Blocker)
- **AVV/DPA** (Auftragsverarbeitungsvertrag): Blocker fuer Pilot-Onboarding mit echten Kundendaten — wartet auf UG-Gruendung 01.05.2026
- AGB, DSGVO-Pruefung durch Anwalt

## Verwandte Notes
- [[architektur]] — Service-Architektur
- [[api]] — Endpoints & Auth-Flow
- [[datenbank]] — Consent/GDPR Tabellen
- [[deployment]] — Infrastruktur-Security
