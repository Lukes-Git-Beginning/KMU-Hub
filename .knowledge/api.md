---
tags: [api, endpoints, openapi]
updated: 2026-04-06
---
# API-Referenz

## OpenAPI Spec
- Datei: `backend/api/openapi.yaml` (14.000+ Zeilen, OpenAPI 3.0.3)
- TypeScript-Generierung: `npm run api:generate` → `desktop/src/renderer/src/api/types.ts` (232KB)
- Gateway: HTTP REST auf Port 8080, leitet intern an gRPC-Services weiter

## Endpoint-Gruppen

| Domain | Praefix | Notizen |
|--------|---------|---------|
| Auth | `/api/v1/auth/…` | Login, Register, 2FA, Refresh, /auth/me |
| CRM | `/api/v1/contacts`, `/companies`, `/deals` | Pipeline-Stages, Activities, Custom-Fields, Tags |
| CRM Extended | `/api/v1/contacts/…/consent`, `/…/duplicates` | Consent-Management, Duplicate Detection (pg_trgm) |
| Chat | `/api/v1/chat/channels`, `/chat/messages` | Reactions, Presence, Files |
| Notifications | `/api/v1/notifications` | List, Read, Preferences, Mutes |
| Dashboard | `/api/v1/dashboard` | Layout-Management pro User/Rolle |
| Work | `/api/v1/projects`, `/tasks`, `/time-entries` | Task-Comments, Files, Time Tracking |
| Calendar | `/api/v1/calendars`, `/events`, `/resources` | Resources, Holidays, Meetings |
| Video | `/api/v1/video/calls`, `/recordings` | LiveKit-basiert |
| Finance | `/api/v1/quotes`, `/invoices`, `/datev/export` | Payments, Credit Notes, Dunning, ZUGFeRD |
| Inbox | `/api/v1/inbox` | Messages, Routing Rules, Teams (Unified Inbox) |
| Automation | `/api/v1/automations` | Workflow CRUD, Execution Logs, Templates |
| HR | `/api/v1/hr/employees`, `/hr/leave`, `/hr/absences` | Teilt "biz" gRPC-Server |
| Security | `/api/v1/security/…` | Audit Logs, 2FA, App-Passwords, GDPR Export — teilt "auth" gRPC |
| Plugin | `/api/v1/plugins/…` | Manifests, Installations, Execution-Logs, Templates |
| Global Search | `/api/v1/search` | Cross-Service (CRM + Dokumente), 500ms Timeout |
| Guest Chat | `/api/v1/guest/…` | Public, kein Auth, eigene Session-Tokens |
| Bexio | `/api/v1/integrations/bexio/…` | OAuth-Flow, Sync Trigger/Status/Logs |
| Lexware | `/api/v1/integrations/lexware/…` | API-Key-basiert |
| DATEV Upload | `/api/v1/datev/upload` | CSV-Upload (Buchungsstapel) |
| CalDAV/CardDAV | `/caldav/…`, `/carddav/…` | go-webdav Proxy, App-Passwords |
| WOPI | `/api/v1/wopi/…` | Document Lock/Unlock, CheckFileInfo (OnlyOffice) |
| Integration Config | `/api/v1/integrations/configs` | Teams/Slack Webhooks + OAuth |
| Registrar | (intern) | Service-Registrierung im Gateway |
| Health | `/health` | Public, kein Auth, Version/Commit/BuildTime |

## Auth-Flow
1. POST `/api/v1/auth/login` (email + password)
2. Falls 2FA aktiv: `requires_two_factor: true` + pending token
3. POST `/api/v1/auth/verify-2fa` (pending token + TOTP code)
4. Response: `access_token` (JWT, 15min) + `refresh_token` (opaque, 7d)
5. Alle geschuetzten Endpoints: `Authorization: Bearer <access_token>`
6. Token-Refresh: POST `/api/v1/auth/refresh` mit refresh_token

## Request/Response Pattern
- Content-Type: `application/json`
- Error-Responses: 400, 401, 403, 409, 429
- Rate Limiting: 100 rps pro User/IP, `429 Too Many Requests` mit `Retry-After: 1`

## Frontend-Integration
- API-Client: `desktop/src/renderer/src/api/client.ts` (openapi-fetch)
- Automatischer Bearer-Header aus Auth-Store
- 401-Interception → transparenter Token-Refresh mit Concurrent De-Duplication
- Offline-Guard: Blockt POST/PUT/DELETE wenn `!navigator.onLine`
- 40+ React Query Hooks in `desktop/src/renderer/src/api/hooks/`

## Verwandte Notes
- [[architektur]] — Service-Architektur & Gateway Routes
- [[datenbank]] — Schema & Tabellen
- [[security]] — Auth & Middleware
- [[integrationen]] — Bexio, Lexware, DATEV Details
