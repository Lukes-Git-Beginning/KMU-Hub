# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-07)

**Core value:** Every employee completes their entire workday without opening another program
**Current focus:** Phase 5 - Desktop App Shell (Plan 1 of 7 complete)

## Current Position

Phase: 5 of 13 (Desktop App Shell)
Plan: 1 of 7 in current phase
Status: In progress
Last activity: 2026-02-07 -- Completed 05-01-PLAN.md (Electron Shell Foundation)

Progress: [##########░░░░░░░░░░] 12% (4/32 plans across phases 4-13)

## Performance Metrics

**Velocity:**
- Total plans completed: 4
- Average duration: ~13 minutes
- Total execution time: ~0.9 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 04 | 3/3 | ~46min | ~15min |
| 05 | 1/7 | ~9min | ~9min |

**Recent Trend:**
- Last 5 plans: 04-01 (~10min), 04-02 (~16min), 04-03 (~20min), 05-01 (~9min)
- Trend: Foundation plans faster than integration-heavy plans

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Roadmap]: Service consolidation -- 3 new backend services (Work, Biz, Automation) instead of 8 separate ones
- [Roadmap]: Gateway refactoring bundled with Phase 4 (before adding new services)
- [Roadmap]: Notifications first (unblocks all future modules)
- [Roadmap]: Full IMAP+SMTP email in v1 (user decision despite research suggesting deferral)
- [Roadmap]: Automation and Plugins last (need stable APIs from all other modules)
- [04-01]: WebSocket hub stays in main.go (cross-cutting, needs both chat + auth clients)
- [04-01]: HealthHandler kept in server/http.go (used by auth/crm/chat services)
- [04-01]: Notification config fields pre-added to config.go
- [04-02]: Raw pgx over pgxlisten for event bus (pgxlisten pre-v1, unstable)
- [04-02]: 30-second grouping window for smart notification collapse
- [04-02]: 7-stage preference evaluation pipeline
- [04-02]: Dual write (events table + pg_notify) for event durability
- [04-02]: DeliveryCallback pattern decouples notification service from WebSocket delivery
- [04-03]: Dual pg_notify channels: 'events' for notification processing, 'notification_delivery' for gateway WebSocket push
- [04-03]: EventEmitter as optional SetEventEmitter pattern for backward compatibility
- [04-03]: Best-effort event emission (errors logged, don't fail primary operations)
- [04-03]: NotifyDelivery in notification repo signals gateway after storing each notification
- [05-01]: electron-vite v5 with build.externalizeDeps (deprecated plugin replaced)
- [05-01]: TSconfig split: node (bundler resolution) + web (DOM, react-jsx, path aliases)
- [05-01]: electron.vite.config.ts excluded from tsc (electron-vite v5/vite 5 type mismatch)
- [05-01]: CSP unsafe-inline for dev only (Vite HMR), production uses self only
- [05-01]: safeStorage with plaintext fallback for Linux without keyring

### Pending Todos

None yet.

### Blockers/Concerns

- [Phase 10]: GoBD compliance requires Steuerberater consultation before data model design
- [Phase 11]: ArbZG/BUrlG implementation details need labor law expert review
- [Phase 10]: DATEV Buchungsstapel format spec not publicly detailed -- may need DATEV partner access

## Session Continuity

Last session: 2026-02-07
Stopped at: Completed 05-01-PLAN.md (Electron Shell Foundation)
Resume file: None
