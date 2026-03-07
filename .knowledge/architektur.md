---
tags: [architektur, backend, frontend]
updated: 2026-03-05
---
# Architektur

## Backend (Go)
- Gateway (HTTP/chi) -> Microservices (gRPC)
- Services mit Ports:
  - auth (:50051), crm (:50052), chat (:50053)
  - notification (:50054), work (:50055), email (:50056)
  - document (:50057), biz (:50058, includes HR+finance+bexio)
  - automation (:50059)

## Frontend
- Electron + React 19 + TypeScript Desktop-App
- Guest Chat: Standalone Vite SPA at /guest/

## Datenbank
- PostgreSQL + Redis (kein Dual-Write!)
- Aenderungen nur via Migrations

## Auth Service
- JWT 15min + opaque refresh 7d, SHA-256, rotation + theft detection
- RBAC: admin/manager/member, resource:action permissions
- 2FA (TOTP), vault, audit logging, GDPR erasure

## CI/CD
- golangci-lint v2.8 (action v7), config: backend/.golangci.yml (version: "2")
- goimports aus formatters entfernt (CI issues)
- Postgres + Redis service containers, E2E with Go binaries
- Claude security review + PR automation

## Regeln
- Thick services, thin handlers
- Structured logging (slog), kein fmt.Println
- 80%+ test coverage, 95%+ fuer kritische Pfade

## Verwandte Notes
- [[stack]] — Strategy Decisions
- [[design]] — Frontend Design System
- [[datenbank]] — Schema & Migrations
- [[api]] — API-Endpoints & OpenAPI
- [[security]] — Auth, RBAC, Middleware
