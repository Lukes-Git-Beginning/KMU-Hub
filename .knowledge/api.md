---
tags: [api, endpoints, openapi]
updated: 2026-03-05
---
# API-Referenz

## OpenAPI Spec
- Datei: `backend/api/openapi.yaml` (14.000+ Zeilen, OpenAPI 3.0.3)
- TypeScript-Generierung: `npm run api:generate` → `desktop/src/renderer/src/api/types.ts` (232KB)
- Gateway: HTTP REST auf Port 8080, leitet intern an gRPC-Services weiter

## Endpoint-Gruppen

| Domain | Tags | Beispiel-Pfade |
|--------|------|----------------|
| Auth | auth, users | `/api/v1/auth/login`, `/auth/register`, `/auth/refresh` |
| CRM | contacts, companies, deals, pipeline-stages, activities, custom-fields, tags | `/api/v1/contacts`, `/companies`, `/deals` |
| Chat | chat-channels, chat-messages, reactions, presence, chat-files | `/api/v1/chat/channels`, `/chat/messages` |
| Notifications | notifications | `/api/v1/notifications` (list, read, preferences, mutes) |
| Dashboard | dashboard | Layout-Management pro User/Rolle |
| Work | projects, tasks, task-comments, task-files, time-tracking | `/api/v1/projects`, `/tasks`, `/time-entries` |
| Calendar | calendars, events, resources, holidays, meetings | `/api/v1/calendars`, `/events`, `/resources` |
| Video | video-calls, recordings | `/api/v1/video/calls`, `/recordings` |
| Finance | quotes, invoices, payments, credit-notes, dunning, datev | `/api/v1/quotes`, `/invoices`, `/datev/export` |
| Inbox | inbox-messages, inbox-teams, inbox-routing | `/api/v1/inbox/messages`, `/inbox/routing` |
| Automation | automations | Workflow CRUD, Execution Logs, Templates |
| Health | health | `/health` (public, kein Auth) |

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
- 401-Interception → transparenter Token-Refresh
- Concurrent-Refresh De-Duplication
- Offline-Guard: Blockt POST/PUT/DELETE wenn `!navigator.onLine`
- 40+ React Query Hooks in `desktop/src/renderer/src/api/hooks/`

## Verwandte Notes
- [[architektur]] — Service-Architektur
- [[datenbank]] — Schema & Tabellen
- [[security]] — Auth & Middleware
