---
tags: [integrationen, bexio, lexware, livekit]
updated: 2026-03-05
---
# Externe Integrationen

## Bexio (OAuth 2.0)
- Contact-Sync (Bexio → KMU Hub)
- Invoice/Quote-Push (KMU Hub → Bexio)
- Payment-Polling (periodisch, kein Webhook)
- Rate-Limiter (respektiert Bexio API-Limits)
- Field-Mapping: BexioContact, BexioInvoice, BexioQuote, BexioPayment
- Code: `backend/internal/biz/bexio/`

## Lexware (API Key)
- Contact-Sync (KMU Hub → Lexware)
- Invoice/Quote-Push
- Webhook-basierte Realtime-Updates (vs. Bexio Polling)
- Vault-verschluesselter API-Key
- Sync-Config pro Tenant: Contact/Invoice/Quote einzeln aktivierbar
- Code: `backend/internal/biz/lexware/`

## DATEV (OAuth 2.0)
- CSV-Export im Buchungsstapel-Format
- Tax-Mapping: Invoices/Quotes → DATEV-Buchungseintraege
- Deutsche Steuer-Compliance
- **Status:** Export-only, kein Realtime-Sync
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
- Code: `backend/internal/caldav/`

## WOPI/OnlyOffice
- WOPI REST-Protokoll fuer Document-Editing
- JWT-basierter Zugang (file_id + user_id Claims)
- File-Locking (TTL-basiert, Concurrent-Edit-Prevention)
- Auto-Versioning bei Save
- OnlyOffice DocumentServer in Docker (Port 8088)
- Code: `backend/internal/document/wopi/`

## Geplant (nicht implementiert)
- Microsoft 365/Teams — Env-Vars vorhanden (TEAMS_APP_ID/PASSWORD)
- Slack — Env-Vars vorhanden (SLACK_BOT_TOKEN/SLACK_SIGNING_SECRET)

## Verwandte Notes
- [[architektur]] — Service-Architektur
- [[security]] — Vault Service, OAuth
- [[deployment]] — Docker-Setup
