---
tags: [security, auth, compliance, gdpr]
updated: 2026-04-18
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

## IP Filter (2026-04-09)
- Cache-TTL: 60s Refresh, 5min Max-Staleness
- Fail-Close: Blockiert Traffic wenn Auth-Service >5min unerreichbar oder nie geladen
- Fail-Stale: Serviert gecachte Regeln innerhalb 5min Fenster
- `rulesEverLoaded` Flag unterscheidet "nie geladen" von "leere Regelliste"

## gRPC mTLS (2026-04-09)
- Optional via `GRPC_TLS_CERT_FILE`, `GRPC_TLS_KEY_FILE`, `GRPC_TLS_CA_FILE`
- Wenn gesetzt: TLS 1.3 mTLS fuer alle Service-zu-Service gRPC Verbindungen
- Wenn leer: Insecure Credentials (lokale Entwicklung)
- `BuildClientTLSConfig()` in `gateway/tls.go`, injiziert in `ServiceRegistry`

## Tenant Isolation (2026-04-09)
- `contacts`, `companies`, `hr_employee_profiles` haben `tenant_id UUID NOT NULL`
- Default: Nil-UUID fuer Single-Tenant Betrieb
- Alle CRM Repository-Queries filtern nach tenant_id
- Multi-Tenant Support vorbereitet fuer Phase 3

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

## Consent Enforcement (2026-04-18, Sprint 0 S0.2)

- **Package:** `backend/internal/crm/consent/`
- **API:** `Asserter.Assert(ctx, contactID, channel)` — `channel ∈ {ChannelEmail, ChannelPhone}`
- **Hooks:**
  - `email/send/service.go` — vor SMTP-Dispatch
  - `dialer/service.go` — vor Twilio/Dialer-Call
- **Query:** `consent_records WHERE contact_id=$1 AND consent_type=$2 AND granted=true AND revoked_at IS NULL`
- **Transactional Skip:** `ChannelEmail` + Contact ohne E-Mail → `nil` (nichts zu senden, kein Consent noetig)
- **Block-Log:** `slog.Warn("consent_block", "contact_id", id, "channel", ch)` + `ErrNoConsent`
- **Status:** Launch-Blocker R1-P0.2 erledigt (PR #10). Gateway-Wiring via additive `NewServiceWithConsent()`-Constructors — Full-Wiring als separater Schritt im cmd-Paket.

## Prod-Secrets Startup-Assertion (2026-04-18, Sprint 0 S0.3)

- Wenn `COSMI_ENV=production`, erzwingt `backend/internal/config/config.go` nicht-leere Werte fuer:
  - `JWT_SECRET`, `VAULT_MASTER_SECRET`, `WOPI_JWT_SECRET`, `MINIO_SECRET_KEY`
- Dev-Default-Werte (wie `change-me`, `dev-secret`) werden in Prod explizit abgelehnt.
- Tests: `backend/internal/config/config_test.go` (TestValidateProductionSecrets).
- Service-Start ohne Secret → harter Abbruch, keine stillen Fallbacks.

## Frontend HTML Sanitization (2026-04-18, Sprint 0 S0.4)

- **Paket:** `dompurify` v3 + `@types/dompurify` (installiert in `desktop/`)
- **Wrapper:** `desktop/src/renderer/src/lib/sanitize.ts`
  - `sanitizeHtml(raw, config?)` — Standard-Whitelist (p, br, formatting, Links, Tabellen, Bilder), Link-Hook erzwingt `target="_blank" rel="noopener noreferrer"`, blockt `javascript:`/`data:`-URIs
  - `sanitizeHtmlStrict(raw)` — nur Text-Formatierung, keine Links/Bilder (Signature-Preview)
- **Call-Sites:** 5 `dangerouslySetInnerHTML` gehaertet in Mails, Wiki-Artikel, Email-Template, Mail-Settings-Signature, IT-Admin-HTML-Preview
- **i18n-trusted Exceptions:** `features/video/RecordingConsentDialog.tsx:103/:108` — beide rendern `t(...)` aus `messages/`, markiert mit `{/* trusted: i18n-rendered ... */}`
- **Tests:** `lib/__tests__/sanitize.test.ts` (10 Vitest-Cases)

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
