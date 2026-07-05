---
tags: [integrationen, bexio, lexware, livekit, plugin, wasm, brevo, smtp]
updated: 2026-07-05
---
# Externe Integrationen

## Bexio (OAuth 2.0)
- Contact-Sync (Bexio → Cosmi)
- Invoice/Quote-Push (Cosmi → Bexio)
- Payment-Polling (periodisch, kein Webhook)
- Rate-Limiter (respektiert Bexio API-Limits)
- Field-Mapping: BexioContact, BexioInvoice, BexioQuote, BexioPayment
- Gateway: `/api/v1/integrations/bexio/…` (OAuth-Flow, Sync Trigger/Status/Logs)
- Code: `backend/internal/biz/bexio/`

### Härtung 2026-06-17 (5 Scope-Check-Blocker geschlossen)
Quelle: `.planning/bexio-scope-check.md`. Stand: G1–G5 + G10 zu, Integration testbar/demobar.
- **G1** OAuth-`state` HMAC-signiert + Nonce + 10-Min-Expiry (`backend/internal/gateway/bexio_state.go`, `encodeBexioState`/`decodeBexioState`) — kein roher `tenant_id` mehr
- **G5** `cmd/biz` macht ohne `VAULT_MASTER_SECRET` einen `os.Exit(1)` (fail-fast statt nil-Panic)
- **G2** ContactService verdrahtet: `crmContactAdapter` (`backend/cmd/biz/crm_contact_adapter.go`) über bestehenden CRM-gRPC-Client (`ListContacts`/`GetContact`). Degradiert sauber wenn `CRM_GRPC_ADDRESS` fehlt (nil → Fallback)
- **G3** `resolveContactBexioIDByEmail` (`contact_resolve.go`): echter CustomerEmail→Contact→Mapping-Lookup statt blind `mappings[0]`; bei 0/≥2 Mappings ohne CRM expliziter Error statt Raten — verhinderte falsche Rechnungs-/Angebots-Empfänger
- **G10** Sync-Kern getestet (Coverage 28%→42%): ContactSyncer/Invoice-/Quote-Push/PaymentPoller/TokenManager inkl. Idempotenz-Beweisen
- **Offen (minor):** G4 RevokeTokens-Vault-Orphan, G6 kein HTTP-Config-Endpoint, G7 org_name nie befüllt, G8 Scheduler single-tenant, G9 kein Feature-Flag-Registry-Eintrag/First-Full-Sync
- **G12 ENTSCHIEDEN (2026-07-05):** Invoice-Pull = **Read-only Spiegel** (NICHT voll bidirektional), Delta-forward (keine Historie). Details unten unter „Welle 3b". Darien-GoBD-Gegenzeichnung noch offen.
- **Follow-up:** `*Client` ist konkreter Typ → API-Fehler-Tests laufen über echten Retry-Backoff (bexio-Package-Test ~28s); ein Client-Interface würde Mocking erlauben

### Welle 3b — Invoice-Pull Bexio→Cosmi (Read-only Spiegel, ab 2026-07-05)
Produktentscheidung (mit User; Darien-GoBD-Gegenzeichnung offen): aus Bexio importierte Rechnungen sind in Cosmi **unveränderliche Spiegel** (`source='bexio'`), **kein** Rück-Sync; Delta-forward (Cursor = Verbindungszeitpunkt, keine Historie). Grund: GoBD-sauber — Bexio bleibt Buch-der-Wahrheit für seine Rechnungen, kein Zwei-Bücher-Problem, kein Nummernkreis-Konflikt (importierte Rechnungen laufen **nie** durch `Send()`/`NextNumberInTx`; sonst verfälscht `GapsDetected` in `service_gobd.go`).
- **Polling, kein Webhook:** Bexio hat keine Webhook-API (Code-Grep leer). Pull folgt dem `ListContacts(updatedSince)`-Delta-Muster (`updated_from` auf `kb_invoice`), NICHT dem Payment-Poll-Vollscan.
- **Loop-Prevention über `source`-Flag:** der Pull überschreibt nie eine `source='cosmi'`-Rechnung (Skip bei Mapping auf Cosmi-origin). LWW via Timestamp in `bexio_entity_mappings` (`sync_direction='inbound'`, im Schema bereits vorhanden).
- **G6/G9 aus der Härtungs-Liste sind VERALTET:** `PUT /sync/config` existiert (`route_bexio.go`, G6 falsch); `integrations.bexio`-Flag existiert (`featureflag/registry.go:97`, G9 falsch). **G8 real & offen:** Scheduler löst Config via `Service.getConfigID(ctx)` ohne tenantID unter System-Context (RLS-Bypass) auf → bei >1 Bexio-Tenant falsche Config; der Pull-Loop MUSS `configID` explizit durchreichen (Fix in Phase 2).

**Stand Phase 0+1 (✅ gepusht + CD-deployt, Prod-Kopf 243):**
- Migration **000243**: `finance_invoices` +`source`('cosmi'-Default)/`external_id`/`external_number` + Partial-Unique-Dedup-Index `(tenant_id,source,external_id) WHERE external_id IS NOT NULL`; `bexio_sync_configs` +`invoice_pull_enabled`/`_interval_minutes`/`last_invoice_pull_at`; `bexio_sync_log`-CHECK +`invoice_pull`.
- `bexio.Client.ListInvoices(updatedSince)` (`invoices.go`) + Reverse-Mapper `FieldMapper.MapInvoiceToKMUHub` (`field_mapper.go`) → `models.ImportedInvoiceInput` (DTO in `models`, damit `invoice`-Paket es ohne bexio-Import konsumiert). Tax-Rate + Währungs-Reverse `lean:`-markiert.
- `invoice.Service.UpsertImported` — GoBD-konformer Importpfad: umgeht `Create`/`Send`/`NextNumberInTx`, idempotenter Upsert per `(tenant_id,source,external_id)` (ON CONFLICT), Line-Items verbatim (kein Recalc). Read-only-Guard `ErrExternalReadOnly` auf Update/Send/MarkPaid/Cancel/LockInvoice. GoBD-Journal (`CountByFiscalYear`=Gap-Detection, `ListForGoBDExport`, `ListForDATEVExport`, `GetOverdue`) filtert `source='cosmi'`. Repo-Reads über geteilte `invoiceColumns`-Konstante (Scan-Reihenfolge fix). Integrationstest gegen echtes PG16 (Upsert-Roundtrip + Line-Replace + Provenance).

**Offen Phase 2+ (neues Fenster, Worktree-Subagenten-Orchestrierung):** `invoice_pull.go`-Puller (Vorbild `ContactSyncer.syncInbound`), `InvoiceImporter`-Wiring in `cmd/biz/main.go`, Scheduler-3.-Ticker + **G8-Fix**, Cursor-Init bei Toggle-On; Proto-Regen `bexio.proto` (⚠ `make proto-biz` deckt bexio.proto NICHT ab → voller `protoc`-Befehl) + FE-Toggle (`BexioSetupWizard` `StepSyncConfig`) + Dashboard-Karte + Read-only-Badge; FE/BE-Pfad-Drift `bexio-client.ts` (`/auth-url` vs `/oauth/authorize`) mit-fixen. Plan: `~/.claude/plans/wir-machen-uns-an-floating-galaxy.md`.

## Lexware (API Key)
- Contact-Sync (Cosmi → Lexware)
- Invoice/Quote-Push
- Webhook-basierte Realtime-Updates (vs. Bexio Polling)
- Vault-verschluesselter API-Key
- Sync-Config pro Tenant: Contact/Invoice/Quote einzeln aktivierbar
- Gateway: `/api/v1/integrations/lexware/…`
- Code: `backend/internal/biz/lexware/`

## DATEV (OAuth 2.0)
- CSV-Export im Buchungsstapel-Format
- Tax-Mapping: Invoices/Quotes → DATEV-Buchungseintraege
- Deutsche Steuer-Compliance
- **Status:** Export-only, kein Realtime-Sync
- Gateway: `/api/v1/datev/upload` (CSV-Upload)
- Code: `backend/internal/biz/datev/`

## Transaktionale E-Mail — Brevo SMTP (seit 2026-06-16)
- **Provider:** Brevo (EU/Frankreich, GDPR-nativ), Free-Transactional-Tier (~300 Mails/Tag) fuer Pilot-0. Gewaehlt wegen EU-Datensouveraenitaet (Alternativen Mailjet/Scaleway FR, Infomaniak CH; Postmark/SES technisch ok aber US).
- **SMTP:** `smtp-relay.brevo.com:587` (STARTTLS). User = Brevo-SMTP-Login (`…@smtp-brevo.com`), Passwort = **SMTP-Key** (NICHT API-Key). Werte in `/opt/kmuhub/.env.production`: `SYSTEM_SMTP_HOST/PORT/USER/PASSWORD/FROM`.
- **Domain-Auth:** `zentria.tech` via 3 DNS-TXT-Records (Brevo-Code am Root, DKIM `mail._domainkey`, DMARC `_dmarc` mit `rua=mailto:rua@dmarc.brevo.com`). SPF/MX nur bei Dedicated IP noetig.
- **Verbraucher:** (1) `auth`-Service Passwort-Reset-Mailer (`cmd/auth/main.go` `systemMailer`). `PASSWORD_RESET_BASE_URL=https://zentria.tech/reset-password` (zeigt auf die noch zu bauende WP-E Astro-Seite → Mail-Klick erst danach end-to-end). (2) **`biz`-Service Dunning-Notice-Mailer (2026-07-05, `e375b6e4`)** — `cmd/biz/mailer.go` `systemMailer` mit PDF-Attachment (MIME via `email/send.MIMEBuilder`, das manuelle auth-MIME kann keine Attachments). `dunning.Service.Send` UND `SendDunningNotice` gehen jetzt über gemeinsames `sendAndNotify` → beide versenden die Mahnungs-PDF (vorher versendete der reale „Mahnung senden"-Button gar nichts). **fail-closed** wenn Mailer konfiguriert + Versand scheitert (Status bleibt draft; GoBD „sent"=zugestellt), fail-open (log-only) wenn Host leer. Compose-SMTP-Passthrough jetzt an `auth` UND `biz` (`${SYSTEM_SMTP_*:-}`, kein `RequireSystemSMTP` an biz → kein Crash-Loop). Booking-Bestaetigungsmail-Pfad noch zu verifizieren.
- **Assertion:** `config.RequireSystemSMTP` (Commit `0f49fd7f`) macht `auth` in `COSMI_ENV=production` hart abhaengig von `SYSTEM_SMTP_HOST/USER/PASSWORD` → Boot-Crash bei Fehlen (vgl. [[security]] Prod-Secrets-Assertion). Dev/CI: leer → Mailer loggt statt sendet.
- Code: `backend/cmd/auth/main.go`, `backend/internal/config/config.go`.

## Circuit-Breaker fuer Bexio + DATEV (2026-06-05, `5dd862eb`, R2-P1.4)
- Neues Package `backend/internal/circuitbreaker` — 3-State (closed/open/half-open), injectable clock fuer Tests
- Bexio- und DATEV-HTTP-Clients laufen durch den Breaker; DATEV zusaetzlich mit Retry
- Verhindert Kaskaden bei haengenden Drittanbieter-APIs (Graceful-Degradation-Regel #8)

## LiveKit (JWT) — seit 2026-06-05 in Prod end-to-end funktional
- 1:1 und Gruppen-Calls (WebRTC)
- Room-Erstellung via JWT-Tokens
- Recording mit DSGVO-Consent-Management
- Egress-Service fuer Aufnahmen (MinIO-Storage) — Prod-Credentials via gerendertem `egress-secrets.yaml` (siehe [[deployment]])
- Feature-Flagged: Graceful Disable wenn API-Key/Secret nicht gesetzt
- Code: `backend/internal/work/livekit/`
- Docker: LiveKit Server (7880/7881 TCP, 7882 UDP) + Egress Container
- `rtc.use_external_ip: true` seit 2026-04-19 aktiv (bessere direct ICE candidates)

### URL-Split intern/public (2026-06-05, `7d492bb6`)
- **`LIVEKIT_INTERNAL_URL`** (`ws://livekit:7880` in Compose) — Server-API (Room-/Egress-twirp). `cfg.LiveKitServerAPIURL()` faellt auf `LIVEKIT_WS_URL` zurueck wenn leer.
- **`LIVEKIT_WS_URL`** (`wss://app.zentria.tech` in Prod) — PUBLIC Signaling-URL, geht via `ws_url` in Join/Start-Responses an die Clients. Caddy proxyt `/rtc*` → `livekit:7880` (nur Signaling+`/rtc/validate`; twirp bleibt intern).
- **Hintergrund:** Vorher teilten sich beide Zwecke EINE Variable; der Template-Wert `wss://…:7443` war nie erreichbar (Port nirgends gemappt) → CreateRoom connection refused. Zusaetzlich waren `token`/`ws_url` in `JoinCallResponse`/`StartMeetingResponse` IMMER leer ("populated by gateway" = nie existenter Code) — `VideoGRPCServer` bekommt seitdem `tokenGen` (RoomManager) + `publicWSURL` und stellt den Organizer-Token selbst aus. Incident-Historie: `docs/livekit-env-production-followups.md`.
- **Verifikations-Muster:** Login → `POST /api/v1/video/calls` → `POST /calls/{id}/join` → `GET https://app.zentria.tech/rtc/validate?access_token=<token>` → 200.
- **Offen:** Calendar-Event-`GenerateJoinToken` liefert in `ws_url` einen Meeting-LINK (`…/room/{name}`) statt der Signaling-URL — Event-Flow-Verdrahtung pruefen (Sprint 4/5).
- **Webhook-Validierung:** Gateway validiert LiveKit-Webhooks mit dem API-Pair, siehe [[security]] "Realtime-Haertung".
- **✅ Webhook-DELIVERY-Gap BEHOBEN (Wave 0, 2026-06-23, `d1be2bd4`):** war (verifiziert 2026-06-22): Prod sandte GAR KEINE LiveKit-Webhooks. `deploy/docker/livekit-secrets.yaml.tmpl` ist **last-wins** (effektive Prod-Config = letztes `--config`, KEIN key-merge mit Base-`livekit.yaml`) und hat `webhook.api_key`, aber **kein `webhook.urls`** + **keinen `room`-Block** → LiveKit hat kein Sende-Ziel + kein `empty_timeout`. Folge: `egress_ended` feuert nie → **Recordings werden in Prod nie `completed`** (stiller Bestandsbug); `room_finished`/Auto-Close unmöglich. Verifiziert via `docker exec docker-livekit-1 cat /etc/livekit-secrets.yaml | grep -A4 webhook` (nur api_key, NO_ROOM_BLOCK) + `docker logs docker-gateway-1 | grep webhook` (leer = nie empfangen). **Last-wins gilt für den GANZEN Config-Baum** — das Secrets-Template muss self-contained sein (keys + `webhook.urls` + `room` + voller rtc-Block); fehlende Top-Level-Keys werden NICHT aus der Base ergänzt. Webhook-Empfänger im Gateway existiert + validiert bereits (`route_video.go:1220-1288`). **FIX (Wave 0):** `webhook.urls` (→`http://gateway:8080/api/v1/webhooks/livekit`) + `room.empty_timeout:900` ins `livekit-secrets.yaml.tmpl`; `room_finished`→neue gRPC `CompleteMeetingByRoom` verdrahtet; Backstop-Sweeper im `work`-Binary (prod-verifiziert aktiv). **2-Schicht-Bug:** zusätzlich war der Egress-gRPC (`CompleteRecordingByEgress`/`FailRecordingByEgress`) NIE in die `VideoService_ServiceDesc` registriert (Hand-Stub `video_egress_ext.go`, Kommentar „registered by R2-B" nie ausgeführt) → lief in `codes.Unimplemented`; sauber via Proto-Regen behoben (Stub gelöscht). **⚠ DEPLOY-FALLE:** `compose up -d livekit` lädt die geänderte Mount-Config NICHT (Container nicht recreated) → Config-Revision-Label am `livekit`-Service in `docker-compose.prod.yml` bumpen erzwingt Recreate; ohne das greift kein `livekit-secrets.yaml.tmpl`-Change. Siehe MEMORY `feedback_livekit_config_reload`.

## TURN-Server (coturn, self-hosted, seit 2026-04-19)
- **Zweck:** Relay-Fallback fuer WebRTC-Clients hinter symmetric NAT / restriktiven Firewalls
- **Host:** `turn.zentria.tech:3478` (Hetzner CAX11 FSN1, Details [[deployment#turn-server-cax11-seit-2026-04-19]])
- **Protokoll:** plain TURN/UDP (MVP, kein TLS); `lt-cred-mech` mit `use-auth-secret`
- **Secret-Sharing:** `static-auth-secret` in `/etc/turnserver.conf` = `TURN_SECRET` in `.env.production` auf App-Server
- **Client-Auth:** Per-Session-Credentials via HMAC-SHA1, eingebettet als Metadata-JSON im LiveKit `AccessToken` (`RoomManager.GenerateToken` + `Service.TURNIceServers`) — Wiring seit R2-P0.1 (2026-04-26, `e4b98b9`+`ad04191`) LIVE
- **Explizite `ice_servers` in JoinCallResponse (Welle 7b-5, 2026-06-20, `a150fd0f`):** Token-Metadata allein reicht nicht — der deklarative LiveKit-JS-Client (`<LiveKitRoom>`) wendet sie nicht vor dem Connect an. Daher exponiert `JoinCallResponse.ice_servers` (Proto, repeated `IceServer`) die Credentials zusaetzlich explizit; Gateway serialisiert via `response.Proto()` (protojson). Backend: `video.RoomManager.TURNIceServers(userID) []IceServerConfig` (livekit-Adapter mappt internen `IceServer` → Domaenen-Typ, kein Import-Zyklus); gRPC-`JoinCall` befuellt `ice_servers` aus den coturn-Credentials. FE setzt `rtcConfig.iceServers` (+ STUN-Fallback) vor `<LiveKitRoom>`-Mount. ⚠ **Aber:** `VideoCallView`/`<LiveKitRoom>` wird aktuell von keiner Route gerendert (`VideoPage` = Mock-Dashboard) → FE-Wiring ready, kein Runtime-Effekt bis die Call-UI gemountet wird. Backend-Teil voll wirksam. Video = nicht Pilot-0-Scope.
- **Assertion-Guard:** TURN-Symmetrie (`COTURN_HOST`↔`TURN_SECRET` beide oder keiner) wird in Prod beim Start erzwungen, siehe [[security]]
- **Deploy-Doku:** `deploy/turn/livekit-integration.md` (Option B)

## CalDAV/CardDAV (go-webdav)
- App-spezifische Passwoerter fuer Clients (Thunderbird, iOS, macOS)
- Sync-Tokens fuer inkrementelle Updates
- iCalendar ↔ internes Event-Format Konvertierung
- CalDAV (Kalender) + CardDAV (Kontakte)
- Gateway: `/caldav/…`, `/carddav/…`
- Code: `backend/internal/caldav/`

## WOPI/OnlyOffice
- WOPI REST-Protokoll fuer Document-Editing
- JWT-basierter Zugang (file_id + user_id Claims)
- File-Locking (TTL-basiert, Concurrent-Edit-Prevention)
- Auto-Versioning bei Save
- OnlyOffice DocumentServer in Docker (Port 8088) — **aktuell aktiv**
- **Prod-Override (Sprint 0 S0.5):** `deploy/docker/docker-compose.prod.yml` setzt `JWT_ENABLED: "true"` explizit — OnlyOffice akzeptiert in Prod nur JWT-signierte Requests
- **Collabora:** Geplanter Ersatz (MPL 2.0 sicherer als AGPL) — **noch nicht umgesetzt**
- Gateway: `/api/v1/wopi/…`
- Code: `backend/internal/document/wopi/`

## Plugin-System (WASM via wazero) — Feature-Flag OFF bis Phase D (2026-04-18)

- **Status:** Runtime vorhanden, aber bis Launch NICHT aktiv. R2-P1.2 (ehrlicher Pitch — kein WASM-Plugin-Claim zum Launch)
- Plugins als WebAssembly-Module (`.wasm`), Runtime: wazero v1.9.0 (pure Go, kein CGo)
- Sandbox: Kein Filesystem-Zugriff, Netzwerk-Isolation, Capability-basiert
- Manifest-System: Install, Enable/Disable, Permissions, Settings-Schema
- Rate Limiting + Memory Limits pro Plugin
- **Zweifacher Kill-Switch:**
  1. Laufzeit: Flag `plugins.wasm` (Default `false`, Env `COSMI_WASM_PLUGINS_ENABLED`) im Feature-Flag-Registry
  2. Compile-Zeit: Build-Tag `//go:build !no_wasm` auf `runtime.go/sandbox.go/hostapi.go/memory.go/lifecycle.go`; Stub `runtime_disabled.go` mit `//go:build no_wasm` exportiert gleiche API als no-op
- **Prod-Build:** `make build-prod` setzt `-tags no_wasm` → kein WASM-Code im Binary
- Gateway: `/api/v1/plugins/…` (Manifests, Installations, Execution-Logs, Templates) — Config-Plugins bleiben aktiv (`plugins.config`=true)
- Industry-Module (14) laufen ueber eigene Modul-Flags `modules.<name>`, nicht ueber WASM
- Code: `backend/internal/plugin/` (sdk/, wasm/)
- Haertungs-Roadmap: Ed25519-Signing + WASI-Deny-Set nur bei Phase-D-Marktsignal (siehe [[architektur]] Feature-Flag-Subsystem)

## Guest Chat
- Standalone oeffentliche Chat-Oberflaeche (Vite SPA unter `/guest/`)
- Eigene Session-Tokens (kein regulaeres Auth-JWT)
- Gateway: `/api/v1/guest/…` (public endpoints)
- Code: `backend/internal/chat/guest/`

## Teams / Slack (teilweise implementiert)
- Webhook-Handler implementiert in `route_integration.go`
- Teams: OAuth Install Flow (`/slack/oauth/install`, `/slack/oauth/callback`)
- Slack: Bot-Token + Signing Secret
- **Status:** Env-Vars vorhanden, Backend-Code existiert, aber **nicht live/getestet**
- Env: `TEAMS_APP_ID/PASSWORD`, `SLACK_BOT_TOKEN/SLACK_SIGNING_SECRET`

## Verwandte Notes
- [[architektur]] — Service-Architektur & Gateway Routes
- [[security]] — Vault Service, OAuth
- [[deployment]] — Docker-Setup
- [[stack]] — Frontend-Bibliotheken
