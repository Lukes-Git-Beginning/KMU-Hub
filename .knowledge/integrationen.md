---
tags: [integrationen, bexio, lexware, livekit, plugin, wasm, brevo, smtp]
updated: 2026-06-17
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
- **Verbraucher:** `auth`-Service Passwort-Reset-Mailer (`cmd/auth/main.go` `systemMailer`). `PASSWORD_RESET_BASE_URL=https://zentria.tech/reset-password` (zeigt auf die noch zu bauende WP-E Astro-Seite → Mail-Klick erst danach end-to-end). Compose-SMTP-Passthrough aktuell NUR an `auth` — Booking-Bestaetigungsmail-Pfad noch zu verifizieren.
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

## TURN-Server (coturn, self-hosted, seit 2026-04-19)
- **Zweck:** Relay-Fallback fuer WebRTC-Clients hinter symmetric NAT / restriktiven Firewalls
- **Host:** `turn.zentria.tech:3478` (Hetzner CAX11 FSN1, Details [[deployment#turn-server-cax11-seit-2026-04-19]])
- **Protokoll:** plain TURN/UDP (MVP, kein TLS); `lt-cred-mech` mit `use-auth-secret`
- **Secret-Sharing:** `static-auth-secret` in `/etc/turnserver.conf` = `TURN_SECRET` in `.env.production` auf App-Server
- **Client-Auth:** Per-Session-Credentials via HMAC-SHA1, eingebettet als Metadata-JSON im LiveKit `AccessToken` (`RoomManager.GenerateToken` + `Service.TURNIceServers`) — Wiring seit R2-P0.1 (2026-04-26, `e4b98b9`+`ad04191`) LIVE
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
