---
phase: 16-automation-engine
plan: 01
subsystem: automation
tags: [grpc, protobuf, expr-lang, postgres, jsonb, condition-evaluator]

# Dependency graph
requires:
  - phase: 14-event-infrastructure
    provides: "Event bus, pg_notify, event types, Condition/Action models in models/inbox.go"
provides:
  - "AutomationService proto with 18 RPCs"
  - "Migration 000052 with automations, automation_executions, automation_templates tables"
  - "Go models for all automation entities with JSONB support"
  - "Dual condition evaluator (simple AND/OR tree + expr-lang expressions)"
  - "ExprEnv typed environments for CRM, finance, HR, work, email triggers"
  - "Workflow PostgreSQL repository (CRUD for automations, executions, templates)"
  - "Automation binary scaffold on :50059/:9099"
affects: [16-02, 16-03]

# Tech tracking
tech-stack:
  added: [expr-lang/expr v1.17.8]
  patterns: [dual-mode condition evaluation, sync.Map expression cache, typed expr environments]

key-files:
  created:
    - backend/proto/automation/v1/automation.proto
    - backend/migrations/000052_create_automation_tables.up.sql
    - backend/internal/models/automation.go
    - backend/internal/automation/condition/evaluator.go
    - backend/internal/automation/condition/evaluator_test.go
    - backend/internal/automation/condition/expr_env.go
    - backend/internal/automation/condition/types.go
    - backend/internal/automation/workflow/repository.go
    - backend/internal/automation/workflow/postgres_repository.go
    - backend/internal/automation/workflow/errors.go
    - backend/cmd/automation/main.go
    - backend/tools/automation_deps.go
  modified:
    - backend/Makefile
    - backend/internal/config/config.go
    - backend/go.mod
    - backend/go.sum

key-decisions:
  - "expr-lang/expr v1.17.8 for expression evaluation (latest stable, compile+cache pattern)"
  - "Dual condition mode: simple AND/OR tree reuses models.Condition from inbox, expression mode adds expr-lang"
  - "sync.Map cache for compiled expr programs (concurrent safe, no TTL needed for immutable programs)"
  - "Automation service as standalone binary (not co-hosted) on :50059/:9099"
  - "ExprEnv typed environments built from event payload for compile-time field validation in expressions"

patterns-established:
  - "Dual condition evaluator: simple mode for dropdown UI, expression mode for power users"
  - "ExprEnv builder: BuildEnvFromEvent maps raw event payloads to typed structs per module"
  - "Dotted field path resolution in simple mode conditions (e.g., deal.value)"

requirements-completed: [AUTO-01, AUTO-04]

# Metrics
duration: 8min
completed: 2026-02-20
---

# Phase 16 Plan 01: Automation Data Foundation Summary

**Dual condition evaluator with 44 passing tests (simple AND/OR + expr-lang), AutomationService proto with 18 RPCs, 3-table migration, and automation binary scaffold**

## Performance

- **Duration:** 8 min
- **Started:** 2026-02-20T12:30:10Z
- **Completed:** 2026-02-20T12:38:24Z
- **Tasks:** 2
- **Files modified:** 16

## Accomplishments
- AutomationService proto with 18 RPCs covering CRUD, state, executions, catalog, templates, testing, and stats
- Migration 000052 creates automations, automation_executions, and automation_templates tables with proper indexes
- Dual condition evaluator with 44 tests covering all 15 operators in simple mode and expr-lang expressions
- Typed ExprEnv environments for CRM, finance, HR, work, and email trigger contexts
- Full workflow PostgreSQL repository implementation with CRUD, pagination, cleanup, and upsert
- Automation binary scaffold compiles and follows notification service pattern

## Task Commits

Each task was committed atomically:

1. **Task 1: Proto + migration + models + deps** - `8dd1573` (feat)
2. **Task 2: Condition evaluator + workflow repository + binary scaffold** - `f893cd3` (feat)

## Files Created/Modified
- `backend/proto/automation/v1/automation.proto` - AutomationService gRPC contract with 18 RPCs
- `backend/proto/automation/v1/automation.pb.go` - Generated protobuf Go code
- `backend/proto/automation/v1/automation_grpc.pb.go` - Generated gRPC Go code
- `backend/migrations/000052_create_automation_tables.up.sql` - 3 tables with indexes and constraints
- `backend/migrations/000052_create_automation_tables.down.sql` - Reverse migration
- `backend/internal/models/automation.go` - Go domain models with JSONB support
- `backend/internal/automation/condition/evaluator.go` - Dual condition evaluator (simple + expr-lang)
- `backend/internal/automation/condition/evaluator_test.go` - 44 tests covering all operators and modes
- `backend/internal/automation/condition/expr_env.go` - Typed expr-lang environments per module
- `backend/internal/automation/condition/types.go` - Operator constants and type aliases
- `backend/internal/automation/workflow/repository.go` - Repository/ExecutionRepository/TemplateRepository interfaces
- `backend/internal/automation/workflow/postgres_repository.go` - Full PostgreSQL implementation
- `backend/internal/automation/workflow/errors.go` - Domain error types
- `backend/cmd/automation/main.go` - Automation binary scaffold (:50059/:9099)
- `backend/tools/automation_deps.go` - Retains expr-lang/expr in go.mod
- `backend/Makefile` - proto-automation, run-automation, build automation targets
- `backend/internal/config/config.go` - AutomationGRPCPort, AutomationHealthPort entries

## Decisions Made
- Used expr-lang/expr v1.17.8 (latest) instead of v1.17.7 specified in plan (minor version bump, compatible API)
- Automation service as standalone binary (following document service pattern rather than co-hosting)
- sync.Map for expression cache instead of LRU -- expressions are immutable once compiled so eviction is unnecessary
- Dotted field path resolution added to simple mode evaluator for nested environment access (e.g., deal.value)
- Config entries added for automation ports (:50059 gRPC, :9099 health) in config.go (Rule 2 -- missing critical config)

## Deviations from Plan
None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Proto, models, and condition evaluator ready for Plan 02 (workflow execution engine)
- Repository interfaces ready for gRPC server implementation
- Binary scaffold ready for service registration
- All 44 condition evaluator tests passing as regression guard for Plan 02

---
*Phase: 16-automation-engine*
*Completed: 2026-02-20*
