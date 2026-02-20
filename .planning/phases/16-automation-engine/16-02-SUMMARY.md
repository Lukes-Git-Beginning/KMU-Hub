---
phase: 16-automation-engine
plan: 02
subsystem: automation
tags: [workflow-engine, trigger-registry, action-executors, grpc, gateway, templates, docker, event-bus]

# Dependency graph
requires:
  - phase: 16-01
    provides: "Dual condition evaluator, workflow repository, automation proto, models, binary scaffold"
  - phase: 14-01
    provides: "EventBus with pg_notify, event types, event payload model"
provides:
  - "TriggerRegistry with 14 built-in triggers across 6 modules"
  - "ActionRegistry with 8 action executors using gRPC clients"
  - "WorkflowEngine with trigger->condition->actions pipeline, output chaining, circuit breaker"
  - "AutomationConsumer registered on EventBus wildcard handler"
  - "TimeTriggerPoller for scheduled trigger evaluation (5-minute interval)"
  - "12 pre-built automation templates across 4 categories"
  - "AutomationGRPCServer implementing 16 RPCs"
  - "Gateway HTTP routes: 16 endpoints under /api/v1/automations/"
  - "Docker Compose automation service on :50059/:9099"
affects: [16-03]

# Tech tracking
tech-stack:
  added: []
  patterns: [wildcard event consumer, semaphore-bounded concurrency, circuit breaker auto-disable, output chaining, template resolution with {{key}} replacement, function-reference adapter for import cycle avoidance]

key-files:
  created:
    - backend/internal/automation/trigger/types.go
    - backend/internal/automation/trigger/registry.go
    - backend/internal/automation/trigger/matcher.go
    - backend/internal/automation/trigger/poller.go
    - backend/internal/automation/action/types.go
    - backend/internal/automation/action/registry.go
    - backend/internal/automation/action/executor.go
    - backend/internal/automation/action/crm_actions.go
    - backend/internal/automation/action/work_actions.go
    - backend/internal/automation/action/email_actions.go
    - backend/internal/automation/action/notification_actions.go
    - backend/internal/automation/action/calendar_actions.go
    - backend/internal/automation/action/biz_actions.go
    - backend/internal/automation/engine/engine.go
    - backend/internal/automation/engine/consumer.go
    - backend/internal/automation/engine/logger.go
    - backend/internal/automation/workflow/service.go
    - backend/internal/automation/template/registry.go
    - backend/internal/automation/template/templates.go
    - backend/internal/server/automation_grpc.go
    - backend/internal/gateway/route_automation.go
  modified:
    - backend/cmd/automation/main.go
    - backend/cmd/gateway/main.go
    - backend/internal/notification/event/types.go
    - backend/api/openapi.yaml
    - deploy/docker/docker-compose.yml

key-decisions:
  - "Function references (closures) instead of direct imports to avoid workflow->trigger->workflow import cycle"
  - "Notification action is standalone (slog-based) since notification service has no CreateNotification RPC"
  - "Calendar service co-hosted on work binary -- reuse same gRPC connection"
  - "Semaphore of 20 concurrent executions to prevent resource exhaustion"
  - "Circuit breaker: auto-disable automation at 100 executions/hour"
  - "30-second TTL cache on active automations in TriggerMatcher"
  - "Template {{key}} resolution with dot-notation env flattening for nested access"
  - "Loop prevention: skip events where module_id is automation"

patterns-established:
  - "Function-reference adapter pattern: workflow.Service accepts closures from registries to avoid import cycles"
  - "Wildcard EventBus consumer: automationConsumer.HandleEvent registered as '*' handler"
  - "Output chaining: each action result merges into env as prev_* and step_N_* keys"
  - "Fire-and-forget execution logging: ExecutionLogger methods never return errors to callers"

requirements-completed: [AUTO-02, AUTO-03, AUTO-05, AUTO-06]

# Metrics
duration: ~25min
completed: 2026-02-20
---

# Phase 16 Plan 02: Workflow Execution Engine Summary

**Full automation runtime with 14 triggers, 8 action executors, workflow engine with output chaining and circuit breaker, 12 templates, gRPC server with 16 RPCs, gateway HTTP routes, and Docker integration**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-02-20
- **Completed:** 2026-02-20
- **Tasks:** 4
- **Files modified:** 27

## Accomplishments
- TriggerRegistry with 14 built-in triggers covering CRM (3), Work (3), Email (2), Finance (3), HR (2), Calendar (1)
- ActionRegistry with 8 action executors: UpdateDealField, CreateContact, CreateTask, SendEmail, SendNotification, CreateCalendarEvent, CreateInvoiceDraft, CreateDunning
- WorkflowEngine executes full trigger->condition->actions pipeline with 20-concurrent semaphore, 10s per-action timeout, output chaining, and circuit breaker (100/hour auto-disable)
- AutomationConsumer registered on EventBus wildcard handler with loop prevention (skip automation-sourced events)
- TimeTriggerPoller checks scheduled triggers every 5 minutes with dedup via execution history
- ExecutionLogger records every step with inputs, outputs, errors, and duration (fire-and-forget)
- WorkflowService provides CRUD, enable/disable, catalog, stats, test condition, dry run, and create-from-template
- 12 pre-built templates: Vertrieb (4), Finanzen (3), Personal (3), Kommunikation (2)
- AutomationGRPCServer implements 16 RPCs with proper error mapping to gRPC status codes
- Gateway exposes 16 HTTP endpoints under /api/v1/automations/ with auth and permission middleware
- Automation service added to Docker Compose on :50059/:9099 with health checks
- OpenAPI spec documents all automation endpoints with schemas

## Task Commits

Each task was committed atomically:

1. **Task 1: Trigger registry + action types/registry + CRM/Work/Email executors** - `7bc6fd5` (feat)
2. **Task 2a: Remaining executors + workflow engine + consumer + logger** - `17d44f7` (feat)
3. **Task 2b: Workflow service + templates + main.go wiring** - `a36e7d5` (feat)
4. **Task 3: gRPC server + gateway routes + Docker + OpenAPI** - `6f09dbc` (feat)

## Files Created/Modified

**Trigger package:**
- `backend/internal/automation/trigger/types.go` - TriggerDefinition, TriggerField, ConfigParam, TriggerMatch structs
- `backend/internal/automation/trigger/registry.go` - TriggerRegistry with 14 built-in triggers across 6 modules
- `backend/internal/automation/trigger/matcher.go` - TriggerMatcher with 30s TTL cache for active automations
- `backend/internal/automation/trigger/poller.go` - TimeTriggerPoller with 5-minute interval and dedup

**Action package:**
- `backend/internal/automation/action/types.go` - ActionExecutor interface, ActionResult, ActionDefinition
- `backend/internal/automation/action/registry.go` - ActionRegistry with concurrent-safe executor/definition maps
- `backend/internal/automation/action/executor.go` - Template resolution with {{key}} replacement and env flattening
- `backend/internal/automation/action/crm_actions.go` - UpdateDealFieldAction (stage + field updates), CreateContactAction
- `backend/internal/automation/action/work_actions.go` - CreateTaskAction with project, assignee, priority, due date
- `backend/internal/automation/action/email_actions.go` - SendEmailAction via EmailServiceClient
- `backend/internal/automation/action/notification_actions.go` - SendNotificationAction (standalone, slog-based)
- `backend/internal/automation/action/calendar_actions.go` - CreateCalendarEventAction with relative time parsing
- `backend/internal/automation/action/biz_actions.go` - CreateInvoiceDraftAction and CreateDunningAction

**Engine package:**
- `backend/internal/automation/engine/engine.go` - WorkflowEngine with semaphore, circuit breaker, output chaining
- `backend/internal/automation/engine/consumer.go` - AutomationConsumer with wildcard EventBus handler and loop prevention
- `backend/internal/automation/engine/logger.go` - ExecutionLogger with fire-and-forget methods

**Service + templates:**
- `backend/internal/automation/workflow/service.go` - Service with CRUD, catalog, stats, test, dry-run, create-from-template
- `backend/internal/automation/template/templates.go` - 12 pre-built templates with German names/descriptions
- `backend/internal/automation/template/registry.go` - TemplateRegistry with SeedToDatabase for startup seeding

**Server + gateway:**
- `backend/internal/server/automation_grpc.go` - AutomationGRPCServer implementing 16 RPCs with converters
- `backend/internal/gateway/route_automation.go` - 16 HTTP endpoints with auth/permission middleware

**Wiring + config:**
- `backend/cmd/automation/main.go` - Full wiring: gRPC clients, registries, engine, consumer, EventBus, poller, templates, gRPC server
- `backend/cmd/gateway/main.go` - Automation service registered in gateway with AutomationRoutes
- `backend/internal/notification/event/types.go` - Added ModuleAutomation and automation event constants
- `backend/api/openapi.yaml` - All 16 automation endpoints documented with schemas
- `deploy/docker/docker-compose.yml` - Automation service container with port :50059/:9099

## Decisions Made
- Function-reference adapter pattern avoids import cycle between workflow, trigger, and action packages
- Notification action is standalone (no gRPC client) since notification service lacks a CreateNotification RPC
- Calendar action reuses work gRPC connection since calendar is co-hosted on work binary
- CRM UpdateDealField uses MoveDealToStage RPC for "stage" field updates (no Stage field on UpdateDealRequest)
- Proto helper functions renamed (automationStructToJSON, automationJSONToStruct) to avoid collision with inbox_grpc.go

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Calendar Description *string type mismatch**
- **Found during:** Task 2a
- **Issue:** CreateEventRequest.Description is *string in proto, not string. Direct assignment failed.
- **Fix:** Conditional pointer assignment `req.Description = &description`
- **Files modified:** calendar_actions.go
- **Commit:** 17d44f7

**2. [Rule 3 - Blocking] Import cycle between workflow and trigger packages**
- **Found during:** Task 2b
- **Issue:** workflow/service.go imported trigger.TriggerRegistry, but trigger/matcher.go imports workflow.Repository
- **Fix:** Replaced direct imports with function references (closures) and local adapter interfaces
- **Files modified:** workflow/service.go, cmd/automation/main.go
- **Commit:** a36e7d5

**3. [Rule 1 - Bug] Proto field name mismatches in gRPC server and gateway**
- **Found during:** Task 3
- **Issue:** Multiple proto field names differed from initial implementation (ExecutionStatus enum naming, TestConditionRequest fields, DryRunAutomationResponse structure, optional pointer types)
- **Fix:** Corrected all proto field accesses, renamed helper functions to avoid collision, added pointer conversions
- **Files modified:** automation_grpc.go, route_automation.go
- **Commit:** 6f09dbc

## Issues Encountered
None beyond the auto-fixed deviations above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Full automation runtime ready for Plan 03 (frontend)
- All 16 HTTP endpoints available for React hooks and components
- gRPC server implements all proto RPCs for internal service calls
- Docker Compose updated for local development with automation service
- Template catalog ready for frontend template gallery component

## Self-Check: PASSED

- All 21 created files verified on disk
- All 4 task commits verified in git log (7bc6fd5, 17d44f7, a36e7d5, 6f09dbc)
- `go build ./...` passes for entire backend

---
*Phase: 16-automation-engine*
*Completed: 2026-02-20*
