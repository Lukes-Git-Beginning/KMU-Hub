---
tags: [security, auth, compliance, gdpr]
updated: 2026-04-28
---
# Security & Compliance

## Authentifizierung
- JWT Access Token: 15min Expiry, Claims: `uid`, `tid` (Tenant), `roles`, `perms`
- Opaque Refresh Token: 7 Tage, SHA-256 gehasht in DB, Rotation + Theft Detection
- Passwort: bcrypt cost=12
- 2FA: TOTP (RFC 6238), Grace Period, Enforcement nach Ablauf

## RBAC
- Rollen: admin, manager, member
- Permissions: `resource:action` Pattern (z.B. `contacts:write`, `deals:delete`, `dialer:campaigns:read`, `berichte:reports:read|write` — Migration 080)
- Middleware: `RequireRole(roles...)`, `RequirePermission(resource, action)`
- 403 Forbidden bei unzureichenden Rechten
- Neue Modul-Permissions landen als eigene Seed-Migration (`08x_seed_<modul>_permissions.up.sql`) mit Admin-Default-Grant

## Middleware-Stack (Reihenfolge)
1. Metrics → 2. RequestID → 3. SecurityHeaders → 4. Logging → 5. CORS → 6. IP Filter → 7. Rate Limiting → 8. Audit Logger → 9. Auth (JWT) → 10. RBAC → 11. Idempotency (WarnMode bis Welle 4)
- Code: `backend/internal/middleware/`
- Idempotency-Position bewusst nach RBAC: nur authentifizierte und autorisierte Mutations werden gehasht/gespeichert. WarnMode loggt fehlende `Idempotency-Key`-Header ohne zu blocken — Hard-Mode greift erst nach Frontend-Rollout in Welle 4.

## Idempotency-Konvention (2026-04-28, Sprint 2 Welle 3)

- **Pflicht-Header `Idempotency-Key: <UUIDv4>`** auf POST/PUT/PATCH unter `/api/v1/` (Whitelist: `/auth/login`, `/auth/refresh`, `/auth/2fa` wegen Token-Rotation, GET/HEAD/OPTIONS sowieso skipped).
- Frontend-Auto-Setting via `desktop/src/renderer/src/api/idempotency.ts` (`crypto.randomUUID()`-Wrapper, `withIdempotencyKey()`-Helper). Header wird in `api/client.ts` automatisch gesetzt falls nicht vorhanden.
- Backend `request_hash = sha256("tenant_id|user_id|method|path|body")`. Cases:
  - **Replay** (gleicher Key + gleicher Hash + completed): cached `response_status` + `response_body` zurueck (kein erneuter Service-Call).
  - **Conflict** (gleicher Key + anderer Hash): 422 Unprocessable Entity.
  - **In-Flight** (gleicher Key, noch kein `completed_at`): 409 Conflict + `Retry-After: 2`.
  - **Fresh**: Reserve via `ON CONFLICT DO NOTHING`, Handler aufrufen, Response cachen, `completed_at=NOW()`.
- Speicher: Tabelle `idempotency_keys` (Migration 000105). Tenant-scoped Index `(tenant_id, user_id, created_at DESC)`. TTL 24h via `expires_at`-Spalte; Cleanup-Goroutine im Gateway tickt 1h und delete `WHERE expires_at < NOW()`.
- Modus-Toggle: `middleware.WarnMode` vs `middleware.HardMode`. WarnMode loggt `slog.Warn("idempotency_key_missing", ...)` und gibt der Mutation den freien Lauf — gewollt fuer Phase-1-Rollout. Hard-Mode rejectet 400 Bad Request.

## Pre-Recording-Consent (2026-04-28, Sprint 2 Welle 3, R2-P0.4)

- **Migration 000107:** `recordings.pre_recording_consent_at TIMESTAMPTZ NULL` + `initiator_consent_id UUID NULL`, plus `recording_consents.responded_at`.
- **Service-Gate:** `recording.Service.StartRecording` returniert `ErrPreConsentMissing` (HTTP 412 Precondition Failed) wenn `pre_recording_consent_at IS NULL`. Verhindert dass das Egress startet ohne dass der Initiator aktiv zugestimmt hat — bisher wurden Empfaenger-Consents geprueft (`CountPendingConsents`), aber Initiator-Consent war implizit.
- **Endpoint:** `POST /api/v1/video/recordings/{id}/initiator-consent` (gRPC `ConfirmInitiatorConsent` via `proto/video/v1/video_pre_consent_ext.go` — Handfile-Extension-Pattern, kein Proto-Regen) stempelt das Feld nachdem der Initiator den Pre-Dialog bestaetigt.
- **Frontend-Flow:** `RecordingInitiatorDialog` (Radix AlertDialog, non-dismissible) wird in `CallControls.handleRecordToggle` gezeigt VOR `startRecording.mutate()`. `handleConfirmStart` ruft erst `confirmInitiatorConsent`, dann `startRecording`. `RecordingActiveBanner` (roter Top-Stripe) ist sichtbar waehrend `recordings.status='active'` — i18n-Keys `features.video.recordingBanner.*` und `features.video.recordingInitiator.*`.
- **Audit-Trail:** Backend-Tests `TestStartRecording_RequiresPreConsent` + Roundtrip-Tests. Frontend-Tests in `CallControls.test.tsx` validieren dass `confirmInitiatorConsent` vor `startRecording` aufgerufen wird.

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

## JWT Tenant-Claim & Cross-Layer-Hardening (2026-04-28, Sprint 2 Welle 2D)

Schliesst die Welle-1-Altlast: vor dieser Welle hatten 11 Gateway-Routes (rapporte/schichten/fuhrpark/vermietung/inventar/einkauf/produktion/berichte/formulare/wiki/vertraege) hardcoded `<modul>PlaceholderTenantID = "00000000-...-000000000001"`. Cross-Tenant-Isolation auf HTTP-Ebene war effektiv defekt.

- **Migration 000104** (`backend/migrations/000104_users_tenant_id.up.sql`): `users.tenant_id UUID NOT NULL DEFAULT '00000000-...-000000000001'` + `idx_users_tenant`. Defensives `IF NOT EXISTS` (Lesson aus erstem Patch ohne Idempotenz).
- **JWT-Claim:** `auth.Claims` um `TenantID string \`json:"tid"\`` erweitert (`backend/internal/auth/token.go`). `CreateAccessToken(userID, tenantID, roles, perms)` signiert `tid` in jedes Access-Token. `auth/postgres_repository.go` SELECTed jetzt `tenant_id` (Hotfix `c421fac` — vorher leeres Feld → nie ausgespielt).
- **Middleware:** `middleware.TenantIDKey` Context-Key. `middleware.GetTenantID(ctx) (uuid.UUID, error)` returniert `ErrMissingTenantID` bei leerem oder Nicht-UUID-Wert (**fail-closed, kein Placeholder-Fallback**). `Auth()`-Middleware schreibt `claims.TenantID` in Context neben UserID/Roles/Perms.
- **11 Gateway-Routes refactored:** Erste Aktion in jedem Handler ist `tenantID, err := middleware.GetTenantID(r.Context())`. Bei Fehler `401 Unauthorized` bevor der gRPC-Client erreicht wird. Kein Default-Tenant.
- **5 Cross-Layer-Holes geschlossen** (`8f055e3`):
  - `dialer_grpc.go` + `helpdesk_grpc.go`: `extract*TenantID()`-Helper mit hardcoded UUIDs entfernt. Proto erweitert um `tenant_id` Field auf 13 Requests (Dialer 5: CreateCampaign/ListCampaigns/SetAgentStatus/ListCallOutcomes/CreateCallOutcome; Helpdesk 8: CreateTicket/ListTickets/CreateQueue/ListQueues/CreateCannedResponse/ListCannedResponses/CreateSLAPolicy/ListSLAPolicies). gRPC-Server validiert via `req.GetTenantId()` mit `InvalidArgument`-Guard.
  - `route_wiki.go`: 4 Handler (ListVersions/ListAttachments/UploadAttachment/DeleteAttachment) verwarfen `tenantID` aus Context — jetzt durchgereicht.
  - `route_biz.go::getTenantID(r)` bug: rief `middleware.GetUserID` (string) statt `middleware.GetTenantID` (UUID) auf, d.h. **UserID wurde als TenantID-Surrogate benutzt** in 90 Call-Sites quer durch biz/billing/invoices/quotes/ext/hr/lexware/bexio/datev. Return-Signatur jetzt `(string, error)`, alle Callsites prüfen den Error.
- **Tests:**
  - `auth/token_test.go` — TenantID-Roundtrip + Empty-Legacy-Case
  - `middleware/auth_test.go` — `GetTenantID` valid/empty
  - `gateway/tenant_isolation_test.go` (10 neue Tests) — no-tenant/empty-tid → 401, valid-tid → passes
  - Bestehende Gateway-Tests aktualisiert mit `withTenantID`/`withAuth` Helpers
- **Regel:** Neue Routes MUESSEN `middleware.GetTenantID(ctx)` als erste Aktion aufrufen und 401 bei Fehler zurueckgeben. Kein `<modul>PlaceholderTenantID` mehr.

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
