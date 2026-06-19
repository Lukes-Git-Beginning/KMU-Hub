---
tags: [security, auth, compliance, gdpr, rls, multi-tenant]
updated: 2026-06-19
---
# Security & Compliance

## Multi-Tenant Isolation: 4 Schichten

1. **JWT-Claim** (`tid`) — Auth-Layer issued seit Welle 2D. Required, fail-closed.
2. **HTTP-Middleware** `middleware.GetTenantID(ctx)` — extrahiert `tid` aus Request-Context, returnt `ErrMissingTenantID` bei Fehlen.
3. **Repository-Filter** — `WHERE tenant_id = $X` in jedem SELECT/UPDATE/DELETE; `tenant_id`-Spalte+Bind in jedem INSERT. **Read-Side-Pflicht:** Wer `tenant_id` aus einem geladenen Entity in Folge-INSERTs vererbt, MUSS sie im SELECT zuruecklesen — dritter Fund dieser Klasse 2026-06-05 (`call_sessions` → `call_participants`-Join schlug mit RLS 42501 fehl, `5f16f0d9`).
4. **PostgreSQL RLS** (Welle 1+, Aktivierung in Welle 2+) — Foundation seit Migration 118 in [[datenbank]]. Defense-in-Depth: selbst wenn Layer 3 vergessen wird, filtert die DB.

**RLS-Helpers** (Migration 118): `current_tenant_id()`, `current_user_id()`, `current_app_role()`, `is_system_context()`. Standard-Policy-Generator: `enable_tenant_rls(table_name)` setzt `USING (tenant_id = current_tenant_id() OR is_system_context()) WITH CHECK (...)`. Pool-Layer setzt die GUCs via `database.BeginRLSTx(ctx, pool)` LOCAL pro Tx; `WithSystemContext(ctx)` markiert Worker für Bypass.

## gRPC-Tenant-Trust (Welle 0.6 + Welle 1d)

Gateway propagiert `tenant_id` und `user_id` als gRPC-Metadata an Backend-Services:
- **Outbound** (`middleware.TenantOutboundUnaryInterceptor` in `internal/gateway/registry.go:94`) — seit Welle 0.6 GLOBAL für alle Service-Verbindungen.
- **Inbound** (`middleware.TenantInboundUnaryInterceptor`) — seit Welle 0.6 in chat-service, seit Welle 1d in `cmd/auth`, `cmd/crm`, `cmd/dialer`, `cmd/work`. Restliche 16 Services in Welle 5.
- Inbound ist **soft** (handler wird immer aufgerufen, GetTenantID liefert dann ErrMissingTenantID falls Metadata fehlt) — Login/Register/AcceptInvitation funktionieren ohne Whitelist. Falls Welle 5 hardenet: Whitelist-Methods sind `/auth.AuthService/{Login,Register,RefreshToken,AcceptInvitation,Validate2FALogin}`.
- **gRPC-Handler-Sweeps `req.GetTenantId()` → `middleware.GetTenantID(ctx)` (W2D-Muster, `codes.Unauthenticated` statt `InvalidArgument`):** crm/dialer/helpdesk (W2D 2026-04-28), 10 weitere Handler (Welle 4A), **hr_grpc.go komplett (16 Stellen, `6ff7989a`) + work_grpc.go Rest (CreateTask/ListTasks, `772483fd`) — 2026-06-11**. Request-Felder `tenant_id` sind damit in keinem dieser Handler mehr spoofbar.
- **RLS-Nachzug (Migration 142, 2026-06-11, `7317bdc0`):** `enable_tenant_rls` auf den Chain-PILOT-Tabellen `password_reset_tokens`, `booking_pages`, `public_bookings` aktiviert.

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
- **Permission-Seed-Pflicht (Lesson 2026-06-05, Migration 129):** Diese Doktrin wurde fuer 35 Permissions NICHT befolgt — documents/email/finance/formulare/helpdesk/hr/inbox/wiki/automations/search/settings/recording hatten `RequirePermission`-Guards ohne DB-Rows → **403 fuer JEDEN inkl. Admin**, monatelang unbemerkt. Admin bekommt NICHT automatisch neue Permissions (Migration-000002-CROSS-JOIN galt nur fuer damals existierende). Migration 129 seedet die 35 **admin-only**; Manager/Member-Mapping pro Modul ist Produktentscheidung (Followup F1 in `docs/e2e-modernization-followups.md`). Diff-Check Code-vs-DB vor jedem Modul-Launch: `grep -rhoP 'RequirePermission\("\K[^"]+", "[^"]+' --include='*.go' internal/gateway internal/server | sed 's/", "/:/' | sort -u` gegen `select resource||':'||action from permissions`.
- **JWT-Snapshot:** Rollen/Permissions werden beim Login ins Token gebacken (kein DB-Lookup zur Request-Zeit) — nach Rollenaenderung oder Permission-Seed ist Re-Login zwingend. E2E/Smoke nutzen deshalb `registerAndLoginAdmin` (DB-Promote + Re-Login), siehe [[testing]].

## Middleware-Stack (Reihenfolge)
1. Metrics → 2. RequestID → 3. SecurityHeaders → 4. Logging → 5. CORS → 6. IP Filter → 7. Rate Limiting → 8. Audit Logger → 9. Auth (JWT) → 10. RBAC → 11. Idempotency (WarnMode bis Welle 4)
- Code: `backend/internal/middleware/`
- Idempotency-Position bewusst nach Auth+RBAC: nur authentifizierte und autorisierte Mutations werden gehasht/gespeichert. Reihenfolge in Welle 3.5 explizit fixiert (Idempotency war zwischenzeitlich vor Auth registriert — fail-open auf nicht-authentifizierten Mutations). WarnMode loggt fehlende `Idempotency-Key`-Header ohne zu blocken — Hard-Mode greift erst nach Frontend-Rollout in Welle 4.

## Idempotency-Konvention (2026-04-28, Sprint 2 Welle 3 — gehaertet in Welle 3.5)

- **Pflicht-Header `Idempotency-Key: <UUIDv4>`** auf POST/PUT/PATCH/DELETE unter `/api/v1/` (Whitelist: `/auth/login`, `/auth/refresh`, `/auth/2fa` wegen Token-Rotation, GET/HEAD/OPTIONS sowieso skipped).
- Frontend-Auto-Setting via `desktop/src/renderer/src/api/idempotency.ts` (`crypto.randomUUID()`-Wrapper, `withIdempotencyKey()`-Helper). Header wird in `api/client.ts` automatisch gesetzt falls nicht vorhanden.
- Backend `request_hash = sha256("tenant_id|user_id|method|path|body")`. Cases:
  - **Replay** (gleicher Key + gleicher Hash + completed): cached `response_status` + `response_body` zurueck (kein erneuter Service-Call).
  - **Conflict** (gleicher Key + anderer Hash): 422 Unprocessable Entity.
  - **In-Flight** (gleicher Key, noch kein `completed_at`): 409 Conflict + `Retry-After: 2`.
  - **Fresh**: atomare Reserve via `INSERT ... ON CONFLICT DO UPDATE RETURNING ...` (Welle 3.5: vorher `ON CONFLICT DO NOTHING` + zweiter SELECT, was eine TOCTOU-Race zwischen zwei Replicas auf demselben Key auslief). Handler aufrufen, Response cachen, `completed_at=NOW()` mit `RowsAffected==0`-Sentinel → `ErrKeyMissing` (Stale-Row-Schutz).
- Error-Matching ueber `errors.Is(err, idempotency.ErrInFlight|ErrConflict|ErrKeyMissing)`. Welle 3.5 ersetzt String-Equality durch `errors.Is`, damit gewrappte Errors nicht fail-open durchrutschen. Reserve-Failure-Default ist absichtlich fail-open (`next.ServeHTTP` ohne Dedup) — DB-Outage darf keine Mutations blocken.
- Async Complete laeuft in `goroutine` mit `context.WithoutCancel(r.Context())`, damit der Handler-Return nicht das Speichern abbricht (Welle-3.5-Fix: vorher Deadlock auf in-flight Connection bei kurzen Handler-Latenzen).
- Speicher: Tabelle `idempotency_keys` (Migration 000105). PK `(tenant_id, key)` (Welle 3.5, Migration 000108) — Cross-Tenant-Cache-Replay-Schutz fuer HardMode. Tenant-scoped Index `(tenant_id, user_id, created_at DESC)`. TTL 24h via `expires_at`-Spalte.
- Cleanup-Worker: `IdempotencyCleanupWorker` tickt 1h und ruft `repo.CleanupWithLock(ctx, key=0x49444D50)`. Welle 3.5 macht den Lock echt: `pg_try_advisory_xact_lock` fuer Leader-Election (analog `fuhrpark/worker.go`). Nur eine Replica delete-`WHERE expires_at < NOW()` pro Tick.
- Modus-Toggle: `middleware.WarnMode` vs `middleware.HardMode` ueber Env-Var `IDEMPOTENCY_MODE` (Welle 4B, Default `warn`). WarnMode loggt `slog.Warn("idempotency_key_missing", ...)` und gibt der Mutation den freien Lauf. HardMode rejectet 400 Bad Request bei fehlendem Header. **Dev-Default ist Hard** via `deploy/docker/docker-compose.yml` (`IDEMPOTENCY_MODE=hard` im Gateway-Environment) — fuer Production bleibt `IDEMPOTENCY_MODE` unset → WarnMode default. Prod-Cutover ist Sprint-3-Aktion nach Pilot-1.

### Welle 4B Update (2026-05-07): `Complete()` Composite-PK-Fix
- **Problem:** Bis Welle 4B hatte `idempotency.PostgresRepository.Complete(ctx, key, status, body)` nur `WHERE key = $1` — kein tenant_id-Filter im UPDATE-Pfad. Composite-PK aus 000108 war damit nur halb wirksam: Get/Reserve waren sicher, Complete konnte cross-tenant ueberschreiben.
- **Fix:** Neue Sig `Complete(ctx, tenantID, key, status, body)` mit `WHERE tenant_id = $1 AND key = $2`. Middleware-Caller extrahiert tenantID aus Context. Migration 000113 fuegt einen partial Index `idx_idempotency_keys_tenant_completed (tenant_id, key, completed_at) WHERE completed_at IS NULL` fuer Replay-Detection-Performance hinzu.
- **Tests:** 4 neue Cases — `TestComplete_TenantFilter`, `TestGet_TenantIsolation`, `TestHardMode_MissingKey_Returns400`, `TestHardMode_CrossTenantKeyRejected`.

## Pre-Recording-Consent (2026-04-28, Sprint 2 Welle 3, R2-P0.4 — gehaertet in Welle 3.5)

- **Migration 000107:** `recordings.pre_recording_consent_at TIMESTAMPTZ NULL` + `initiator_consent_id UUID NULL`, plus `recording_consents.responded_at`.
- **Service-Gate:** `recording.Service.StartRecording` returniert `ErrPreConsentMissing` (HTTP 412 Precondition Failed) wenn `pre_recording_consent_at IS NULL`. Welle 3.5 ordnet die Pre-Consent-Pruefung VOR `CreateRecording` an — vorher wurde erst die DB-Row angelegt und dann auf Consent geprueft, was bei `ErrConsentPending` Orphan-Rows ohne Egress hinterliess.
- **Endpoint:** `POST /api/v1/video/recordings/{id}/initiator-consent` (gRPC `ConfirmInitiatorConsent` via `proto/video/v1/video_pre_consent_ext.go` — Handfile-Extension-Pattern, kein Proto-Regen). Welle 3.5: `MarkInitiatorConsent` + `GetPreConsentStatus` filtern jetzt `WHERE id=$1 AND tenant_id=$2` mit `RowsAffected==0`-Sentinel; `route_video.go` setzt RBAC-Permission-Middleware vor den Endpoint.
- **Frontend-Flow:** `RecordingInitiatorDialog` (Radix AlertDialog, non-dismissible) wird in `CallControls.handleRecordToggle` gezeigt nachdem `startRecording.mutate()` die DB-Row reserviert hat. `handleConfirmStart` ruft `confirmInitiatorConsent`, was den Stamp setzt — Welle 3.5 wrappt den Call in `try/catch` mit `sonner.toast.error`, damit ein fehlgeschlagener Stamp keinen Orphan-Recording-State ohne sichtbares Feedback erzeugt. `handleRecordToggle` guarded zusaetzlich gegen Doppelklick via `startRecording.isPending || stopRecording.isPending || confirmInitiatorConsent.isPending`. `RecordingActiveBanner` (roter Top-Stripe) ist sichtbar waehrend `recordings.status='active'` — i18n-Keys `features.video.recordingBanner.*` und `features.video.recordingInitiator.*`.
- **Audit-Trail:** Backend-Tests `TestStartRecording_RequiresPreConsent` + Roundtrip-Tests. Frontend-Tests in `CallControls.test.tsx` validieren dass `confirmInitiatorConsent` vor `startRecording` aufgerufen wird. `gateway/tenant_isolation_test.go` hat 4 zusaetzliche Cases (Welle 3.5) fuer `/recordings/{id}/initiator-consent`: no-tenant, empty-tid, valid-tid, two-tenant-Scenario.

## Realtime-Haertung (2026-06-05, R2-P1-Batch + LiveKit-Cluster)

- **LiveKit-Webhook-Signatur-Validierung** (`f5788d8d`, R2-P1.1): Gateway validiert `POST /api/v1/webhooks/livekit` via `lkwebhook.Receive` (Authorization-JWT + SHA-256-Body-Hash) mit dem **API-Pair** — LiveKit signiert mit Key/Secret, NICHT mit einem separaten Webhook-Secret (`LIVEKIT_WEBHOOK_SECRET` wird zur Laufzeit nicht genutzt, nur von der Assertion verlangt). Ohne Pair: graceful Skip + `slog.Warn` — seit dem Compose-Fix bekommt das Gateway das Pair (Skip-Modus in Prod beendet).
- **WS-Token-Revalidierung** (`98337921`, R2-P1.6): 5-min-Ticker revalidiert das JWT in laufenden WS-Sessions, Close mit `StatusPolicyViolation` bei Expiry/Invalidierung.
- **StartMeeting Organizer-only + `meetings:write`** (`98337921`, R2-P1.5): Migration 000131 seedet `meetings:write` fuer admin+manager+member (Lesson: Guard auf BESTANDS-Funktion braucht Seeds fuer alle bisher berechtigten Rollen — Umkehrung der 129er-Lesson).
- **Join-with-Consent** (`19d5adb7`, R2-P0.4, letzter offener P0): MeetingLobby blockt Join bei aktivem Recording bis Consent-Antwort; Decline = Join blurred/muted; `RecordingActiveBanner` in MeetingRoomView (10s-Polling + WS `recording.started`).
- **Automation-Semaphor per-Tenant** (`f5788d8d`, R2-P1.3): 5 parallele Executions/Tenant innerhalb global 20.

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

## gRPC Tenant-Spoof-Sweep (2026-04-29, Sprint 2 Welle 3.5)

Welle-3.5-Verschaerfung der W2D-C-Lehre: gRPC-Server lasen `tenant_id` aus den Proto-Request-Feldern (`req.GetTenantId()`), die das Gateway aus dem JWT durchgereicht hat. Korrekt fuer den Happy-Path — aber bei Service-zu-Service-Calls oder einem kompromittierten Gateway haette ein Caller eine fremde TenantID in den Request schreiben koennen, die Repository-Filter waeren willig hinterhergerannt.

- **Pattern:** gRPC-Handler ziehen `tenant_id` jetzt aus dem gRPC-Context via `middleware.GetTenantID(ctx)`, NICHT aus dem Proto-Request-Feld. Affektiert: `chat_grpc.go`, `crm_grpc.go`, `work_grpc.go`, `video_grpc.go`, `dialer_grpc.go` — 14+ Methoden.
- **Proto bleibt unveraendert:** `tenant_id` ist weiterhin im Request-Wire-Format, wird aber serverseitig ignoriert / nur fuer Logging genutzt. Frontend muss kein Proto-Regen ueber sich ergehen lassen.
- **Repository-Tenant-Filter-Sweep:** `deal/activity/task/pipelinestage/chat-message/recording postgres_repository.go` enforcen `WHERE id=$1 AND tenant_id=$2` auf jedem UPDATE/DELETE/GetByID/Search-Pfad plus `RowsAffected==0`-Sentinel (kein Service-Layer-Trust).
- **Migration 000108:** PK auf `idempotency_keys` von `(key)` → `(tenant_id, key)`. Letzte Verteidigungslinie gegen Cross-Tenant-Cache-Replay sobald HardMode aktiv ist. Details: [[datenbank]] "Sprint 2 Welle 3.5".

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
- Passwort-Strength-Checks (`go-password-validator`)

### Passwort-Reset-Flow (Chain PILOT, 2026-06-09)

- **Endpoints:** `POST /api/v1/auth/forgot-password` (rate-limited per IP+email, immer 200 — kein User-Enumeration-Leak) + `POST /api/v1/auth/reset-password` (single-use Token, Strength-Check via `go-password-validator`, revokiert alle `refresh_tokens` nach erfolgreichem Reset).
- **Tabelle:** `password_reset_tokens` (Migration 000134) — SHA-256-Hash des Tokens, 1h-Expiry, `used_at` als Single-Use-Guard, `tenant_id NOT NULL`. Details [[datenbank]] Chain PILOT.
- **Security-Eigenschaften:** Token wird einmalig ausgespielt (SHA-256-Hash in DB), nach Verbrauch `used_at` gesetzt (keine Reuse), Refresh-Token-Revocation beim Reset (Session-Invalidierung), kein Unterschied in der API-Response ob User existiert oder nicht.

### Validation-Framework (S4.1, R1-P1.7, ab 2026-06-08)

Zentrales `go-playground/validator/v10`-Framework fuer alle HTTP-Mutation-Handler. Folgt "Thick Services, Thin Handlers": Validierung im Handler zwischen Parse und gRPC-Call.

- **`internal/dachfmt/`** (Leaf, dependency-frei) — pure DACH-Format-Funktionen: `NormalizePhoneE164` (DE/AT/CH, **hierher verschoben aus `dialer`** — Single Source of Truth, `dialer` haelt duennen Alias), `ValidateIBAN` (ISO 7064 mod-97 + Laender-Laengen), `ValidateBIC`, `ValidatePLZ`/`ValidatePLZAny`, `ValidateUStIDDACH`/`ValidateUStIDEU` (VIES-Tabelle aller EU + CH; **2026-06-19 F21: echte Pruefziffer fuer DE (ISO 7064 MOD 11,10)/AT/CH in `ustid_checkdigit.go`, gegen python-stdnum-Vektoren verifiziert — nur DACH gegatet, Rest-EU strukturell, damit valide Nummern nie faelschlich abgelehnt werden**), `ValidateSteuernummer`/`ForBundesland`.
- **`internal/validation/`** — Singleton-`validator` (lazy `sync.Once`), `RegisterTagNameFunc` ⇒ `details[].field` = JSON-Name. 7 Custom-Validatoren: `phone_dach`, `iban`, `bic`, `plz_dach`, `ustid_dach`, `ustid_eu`, `steuernr`. Custom-Validatoren scheitern auf Leerstring (wie Builtin `email`) ⇒ optionale Felder brauchen `omitempty,<rule>`. `Validate(s) error` → `*Errors`; `ErrorBody(err) (any, bool)` baut den Response-Body.
- **`gateway.decodeAndValidate[T](w, r) (T, bool)`** (`helpers.go`) — der eine Parse+Validate-Einstieg. Decode-Fehler → `{"error":"invalid request body"}` (Contract-stabil); Validation-Fehler → strukturierte 400.
- **Error-Shape:** `{"error": "<lesbarer Aggregat-String>", "code": "validation_failed", "details": [{"field","rule","param"}]}`. `error`-Feld bleibt (Frontend `authenticatedFetch.ts` liest nur `.error`); `code`+`details` additiv fuer Field-Level-UI + Frontend-i18n-Mapping. **Backend bleibt EN-only** (kein i18n), Lokalisierung Frontend-seitig via `code`/`rule`.
- **Grenze (verschachtelte Proto):** Verschachtelte Proto-Typen (`*bizv1.LineItem`) tragen keine `validate`-Tags → nur `required`/`min=1`; tiefere Line-Validierung bleibt DB-CHECK (ADR-0007 Migr. 132: `quantity>0`, `tax_rate 0–100`) + Service-Layer. **Ausnahme seit F21 (2026-06-19):** Kunden-`ust_id_nr` (`*bizv1.CustomerSnapshot`) wird in Invoice/Quote/CreditNote-Create explizit via `validateCustomerVAT` (Helper in `gateway/helpers.go`, `ustid_eu`-Regel, gleiche `{error,code,details}`-Shape) geprueft.
- **Grenze (dynamisches JSON):** `json.RawMessage`/`[]byte`/`structpb`-Felder (Inbox-Routing-Rules `conditions`/`actions`, Automation `trigger_config`/`conditions`/`actions`, Formular-`answers`/`fields`, Dashboard-`layout`/`active_widgets`, Berichte-`query_config`/`params`) tragen **keine** Content-Tags — nur die Scalar-Felder werden getaggt; bestehende `json.Valid()`-/Cross-Field-Checks bleiben als Code nach `decodeAndValidate`.
- **Grenze (proto-direct):** Handler, die den Body **direkt in einen Proto-Typ** dekodieren (kein lokales DTO), folgen „Wrapper nur wo wertvoll": E-Mail-Versand (`to/cc/bcc` → `dive,email`) + Pflicht-IDs/Enums bekommen ein lokales Wrapper-DTO; reine Passthrough-CRUD (Email-Account-CRUD, Plugin-Manifest/Rules-CRUD) bleibt unveraendert mit Code-Notiz `// proto-direct: no local DTO (S4.1 boundary)`.
- **Test-Helper:** `assertValidationError(t, rec, "field")` (`testutil_test.go`) prueft 400 + `code=="validation_failed"` + Feld in `details`; Malformed-JSON-Tests bleiben auf `assertErrorContains(rec, "invalid request body")`.
- **Status:** ✅ **Alle 4 Wellen live (2026-06-08)** — Welle 1 (Foundation + `route_auth.go` Referenz, `3937ff2d`), Welle 2 (Finance/Integrations/Dialer/CRM/Helpdesk, `cb784f79`), Welle 3 (Collaboration: work/calendar/caldav/chat/email/inbox/document/automation/formulare/notification, `45898f4b`), Welle 4 (Modul-Services rapporte/schichten/fuhrpark/vermietung/inventar/einkauf/produktion/vertraege + hr/plugin/berichte/dashboard, `29e77fb7`). **Abdeckung:** alle JSON-Body-Mutation-Handler ueber ~41 Route-Files. **Bewusst ausgenommen:** Webhook-Handler (LiveKit/Lexware — Raw-Body-HMAC zuerst), WOPI-Raw-Body, CalDAV/CardDAV-Protokoll-Routes (iCal/XML), Multipart-Uploads, proto-direct-Passthrough-CRUD (siehe Grenze oben). `route_search_global`/`route_registrar`/`route_feature_flags` haben 0 Mutation-Handler. Keine Migration. CD-sicher (additiv, gueltige Requests identisch).

## Consent Enforcement (2026-04-18, Sprint 0 S0.2)

- **Package:** `backend/internal/crm/consent/`
- **API:** `Asserter.Assert(ctx, contactID, channel)` — `channel ∈ {ChannelEmail, ChannelPhone}`
- **Hooks:**
  - `email/send/service.go` — vor SMTP-Dispatch
  - `dialer/service.go` — vor Twilio/Dialer-Call
- **Query:** `consent_records WHERE contact_id=$1 AND consent_type=$2 AND granted=true AND revoked_at IS NULL`
- **Transactional Skip:** `ChannelEmail` + Contact ohne E-Mail → `nil` (nichts zu senden, kein Consent noetig)
- **Block-Log:** `slog.Warn("consent_block", "contact_id", id, "channel", ch)` + `ErrNoConsent`
- **Status:** Launch-Blocker R1-P0.2 erledigt (PR #10). Gateway-Wiring via additive `NewServiceWithConsent()`-Constructors. **Dialer-Wiring Chain PILOT (2026-06-09, `1548a067`):** `cmd/dialer/main.go` rief bisher `dialer.NewService(...)` ohne Consent-Asserter — der nil-safe Guard im Service war tot, `AssertConsent(ChannelPhone)` wurde nie aufgerufen. Jetzt `dialer.NewServiceWithConsent(...)` mit Postgres-Consent-Asserter verdrahtet — analog zu email/send. DSGVO-Consent-Check vor `InitiateDialerCall` ist damit scharf in Production.
- **gRPC-Mapping (Sprint 3 Welle 2A, Commit `1f6c4c0`, 2026-05-08):** `mapDialerError` in `backend/internal/server/dialer_grpc.go` mappt `consent.ErrNoConsent` jetzt auf `codes.PermissionDenied` (vorher fiel es durch auf `codes.Internal`, weil keine explizite Sentinel-Klausel). Test-Case in `dialer_grpc_test.go::TestMapDialerError` deckt alle 10 Sentinels ab (`ErrCampaignNotFound`, `ErrCallSessionNotFound`, `ErrOutcomeNotFound`, `ErrCampaignNotDraft`, `ErrCampaignNotActive`, `ErrInvalidStatusTransition`, `ErrNoContactsAvailable`, `ErrContactAlreadyInCampaign`, `ErrAgentNotAvailable`, `ErrCampaignHasNoContacts`, plus `consent.ErrNoConsent` → `PermissionDenied` plus `nil` → `nil` plus unknown → `Internal`).

## Prod-Secrets Startup-Assertion (S0.3, scharf + Requirements-API seit 2026-06-05)

> **Incident 2026-06-05:** Die Assertion lief in Production NIE (`COSMI_ENV` war nirgends
> gesetzt) — und Production lief auf den hartkodierten Dev-Secrets der Basis-Compose
> (Dev-`JWT_SECRET` auf allen 24 Services!). Vollstaendige Befund-Historie (F-A–F-K):
> `docs/livekit-env-production-followups.md`. Seit dem Fix-Cluster ist
> `COSMI_ENV=production` auf dem Prod-Server SCHARF.

- **Aktivierung:** `COSMI_ENV=production` (case-insensitive) in `Load()` → `validateProductionSecrets`. Fehler → `os.Exit(1)` in allen 24 mains.
- **Requirements-API (`78043a63`):** `config.Load(ctx, ...Requirement)` — Services deklarieren konsumierte Secret-Gruppen: `RequireVault` (auth, email, biz), `RequireMinIO` (chat, work, email, document, gateway), `RequireWOPI` (document, gateway), `RequireSystemSMTP` (auth — `SYSTEM_SMTP_HOST/USER/PASSWORD`, seit 2026-06-16 `0f49fd7f`; Brevo-Relay, Detail [[integrationen]]). Schlanke Modul-Services rufen `Load(ctx)` und werden nur an den universellen Checks gemessen (sonst riss der Go-Default-Rueckfall ihren Start — R1-P0.3 war nie mit den Welle-2-Services kompatibel).
- **Zweigeteilte Deny-Lists** in `config.go`:
  - `composeDevSecrets` (immer verboten, auch ohne Requirement): `docker-dev-secret-…`, `docker-dev-wopi-…`, `docker-dev-vault-…`, `minioadmin`, `devkey`, `devsecret` — ein solcher Wert in Prod heisst Compose/Overlay-Gap.
  - `configDefaultSecrets` ("env not set"-Marker, nur bei Requirement verboten): `wopi-dev-secret-change-me`, `kmuhub`/`kmuhub_dev`, leere Strings.
- **Universal:** `JWT_SECRET` (Compose-Dev-Wert + Mindestlaenge 32), LiveKit-Trio wenn Key ODER Secret gesetzt (inkl. non-empty `LIVEKIT_WEBHOOK_SECRET`), TURN-Symmetrie `COTURN_HOST`↔`TURN_SECRET`.
- **Regel fuer neue Services:** Secret-Gruppe konsumiert ⇒ Requirement im main deklarieren; neuer Dev-Default ⇒ in `composeDevSecrets` eintragen + Testfall.
- Tests: `config_test.go` (required-vs-lean-Semantik, Compose-Dev-Werte, JWT-Laenge, TURN-Symmetrie).

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
- **Export-/Erasure-Handler ECHT seit 2026-06-10** (`47d210d9` — vorher waren ALLE 14 Handler Platzhalter-Stubs!): 7 Export-Handler (auth/crm/chat/work/calendar/sessions/notifications) mit tenant+user-gefilterten SQL-Queries und Datensparsamkeit (keine Token-Hashes/2FA-Secrets, Notifications max 90 Tage/1000). Erasure: anonymize (auth/crm/chat/work — PII → Sentinel, `NOT NULL`-FKs wie `created_by` bleiben auf anonymisiertem User-Record), delete (calendar/notifications), retain (audit, Art. 17(3)(e)). Wiring via Konstruktoren `gdpr.New*Handler(pool)` in `cmd/auth/main.go`
- Security-Routen teilen den "auth" gRPC-Server (kein separater Service noetig)
- **Consent Management (Migration 060):**
  - `consent_records`: Einwilligungen pro Kontakt (6 Typen: marketing_email, marketing_phone, profiling, newsletter, data_processing, data_sharing)
  - Legal Basis: consent, legitimate_interest, contract, legal_obligation
  - IP-Adresse, Quelle, Zeitstempel fuer Audit-Trail
  - CRM Extended Routes: `/api/v1/contacts/…/consent`

### Offen (Phase C Blocker)
- **AVV/DPA** (Auftragsverarbeitungsvertrag): Blocker fuer Pilot-Onboarding mit echten Kundendaten — wartet auf UG-Gruendung 01.05.2026
- AGB, DSGVO-Pruefung durch Anwalt

## CI Security-Scans (Sprint 3 S3.2, ab 2026-05-08; ausgelagert nach `scans.yml` 2026-06-09)

> **Seit 2026-06-09 (`8f6aaa32`):** Diese Scans laufen NICHT mehr bei jedem Push in `ci.yml`, sondern im eigenen Workflow `scans.yml` — woechentlich (Mo 04:00 UTC) + bei Aenderung an `go.mod`/`go.sum`/`package-lock.json` + `workflow_dispatch`. Spart per-Push Actions-Minuten (Dependency-Scans aendern sich nur mit Dependencies). Trivy-Cache-Key jetzt wochenbasiert (vorher `github.run_id` = nie ein Cache-Hit). Memory: `project_ci_pipeline_split_20260609.md`.

| Scan | Tool | Wann | Severity-Threshold | Baseline |
|---|---|---|---|---|
| Source-Code Go | `gosec` | woechentlich + Dep-Aenderung | HIGH/CRITICAL | `backend/.gosec.yaml` (G104/G304/G108) |
| Filesystem Vulns | `trivy` (fs-scan) | woechentlich + Dep-Aenderung | HIGH,CRITICAL | `ignore-unfixed: true` |
| NPM-Dependencies | `npm audit --omit=dev` | woechentlich + Dep-Aenderung | high | kein continue-on-error (0 Findings lokal) |

**Baseline-Status:** `gosec` und `trivy` laufen mit `continue-on-error: true` fuer initiale Baseline-Akzeptanz. Verschaerfung (exit-code: 1 fuer gosec, SARIF-Upload zu GitHub Code Scanning) als S3.2-Followup. `npm-audit` ist hart — 0 Vulnerabilities stand 2026-05-08.

**Cross-Tenant-Test-Gesamtstand (Stand 2026-05-08): ~30 Tests**
- W2D: 10 Tests (`tenant_isolation_test.go` no-tenant/empty-tid/valid-tid-Pattern)
- W3.5: 4 Tests (`/recordings/{id}/initiator-consent` Two-Tenant-Scenario)
- W4B: 12 Tests (Pipeline-Stages/Calendar/TimeEntries/Automations/SavedFilters/CustomFields/Email/Inbox/Dialer/AuditLog/Recordings/Channels)
- Sprint 3 F6: 4 DB-Backed Tests (Calendar-Events, Email-Messages, Recordings)
- Sprint 3 Welle 2B/3: 8 Tests (bexio/lexware EntityMappings/SyncLogs, message_reactions, chat_files)

**Konfiguration:**
- gosec-Exclusions: `backend/.gosec.yaml` — G104 (unhandled errors in tests), G304 (file path via variable, reviewed), G108 (profiling endpoint, not prod-facing)
- trivy ignoriert unfixed CVEs (`ignore-unfixed: true`) um Noise durch unactionable upstream-Findings zu reduzieren
- SARIF-Artefakte werden 30 Tage aufbewahrt (Artefakt-Download, kein GitHub Code Scanning erforderlich)

**Local Run:**
```bash
# gosec
cd backend && gosec -conf .gosec.yaml ./...

# trivy
trivy fs --severity HIGH,CRITICAL --ignore-unfixed .

# npm-audit
cd desktop && npm audit --audit-level=high --omit=dev
```

## Verwandte Notes
- [[architektur]] — Service-Architektur
- [[api]] — Endpoints & Auth-Flow
- [[datenbank]] — Consent/GDPR Tabellen
- [[deployment]] — Infrastruktur-Security
