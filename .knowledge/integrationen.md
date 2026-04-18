---
tags: [integrationen, bexio, lexware, livekit, plugin, wasm]
updated: 2026-04-18
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

## LiveKit (JWT)
- 1:1 und Gruppen-Calls (WebRTC)
- Room-Erstellung via JWT-Tokens
- Recording mit DSGVO-Consent-Management
- Egress-Service fuer Aufnahmen (MinIO-Storage)
- Feature-Flagged: Graceful Disable wenn API-Key/Secret nicht gesetzt
- Code: `backend/internal/work/livekit/`
- Docker: LiveKit Server (7880) + Egress Container

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
