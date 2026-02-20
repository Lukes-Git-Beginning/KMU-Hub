# Phase 16: Automation Engine - Research

**Researched:** 2026-02-20
**Domain:** Workflow automation, expression evaluation, event-driven trigger-action systems
**Confidence:** HIGH

## Summary

Phase 16 builds the "killer feature" of KMU Hub: cross-module workflow automation that standalone tools cannot match. The architecture is straightforward because Phase 14 already established the event infrastructure (EventBus, pg_notify, events table, PGEventEmitter pattern) that automation triggers subscribe to. The automation engine is essentially a new EventBus consumer that, instead of creating inbox messages, evaluates workflow conditions and executes cross-module actions via gRPC.

The core challenge is the condition evaluation engine. The user decided on a hybrid approach: simple dropdown-based conditions for most users plus a raw expression language fallback for power users. The expr-lang/expr library (v1.17.7) is the ideal choice for the expression language -- it is memory-safe, side-effect-free, always terminates, provides static type checking, and supports the operators needed (comparison, string functions, date math, regex via `matches`, ternary/if-else). Its compile-once-run-many model is perfect for cached automation rules. For the simple dropdown conditions, the existing `inbox/routing/evaluator.go` pattern (AND/OR tree with leaf comparisons) can be directly reused or generalized.

The frontend has two modes per the user decision: a 4-step wizard (Trigger, Condition, Action(s), Review) for basic automations and a full react-flow visual node editor for complex multi-branch flows. React Flow (@xyflow/react v12.10.1) supports React 19, has Tailwind CSS 4 compatibility, and provides the node/edge/handle primitives needed for a visual workflow builder.

**Primary recommendation:** Build the automation service as a new binary (`cmd/automation/main.go`) on its own gRPC port (:50059), NOT co-hosted in the notification binary. Unlike the inbox (which is tightly coupled to notifications), automation has its own EventBus listener, needs gRPC clients to ALL other services (CRM, Work, Email, Biz, HR, Calendar, Chat, Notification), and has distinct scaling characteristics. It listens on the same `events` pg_notify channel but processes events through a workflow matching engine rather than the notification pipeline.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **Hybrid approach**: Simple wizard for basic automations, full visual node editor (react-flow) for complex multi-branch flows
- **Wizard model**: 4-step linear flow -- Trigger, Condition, Action(s), Review. Allows multiple actions per trigger. Review step shows full summary before saving
- **Advanced mode**: Full react-flow node editor with draggable nodes, connectors, zoom/pan. User can switch between wizard and canvas views
- **Access model**: Role-based with scope -- admins create org-wide automations, managers create for their team, members create personal automations (only affecting their own items)
- **Full cross-module coverage from v1**: All four module groups (CRM, Email/Inbox, Finance, HR/Calendar) get trigger and action support
- **Priority CRM triggers**: Deal stage change, new contact/company created
- **Cross-module actions**: Always allowed -- a CRM trigger can fire an Email action, a Finance action, a Calendar action, etc. No restrictions on combinations
- **Event mechanism**: Hybrid -- most triggers subscribe to Phase 14 LISTEN/NOTIFY events directly; time-based triggers (e.g., "invoice overdue for 7 days") use scheduled polling checks
- **10-15 templates** shipping in v1, fully editable -- broad coverage across all modules
- **Discovery**: Both a browsable template gallery AND contextual suggestions (on first use, on empty state, and contextually when relevant)
- **Must-have templates**: "Invoice overdue -> Dunning workflow" and "Leave approved -> Calendar event + team notification" are essential
- **Organization**: Default grouping by module/use case (Vertrieb, Personal, Finanzen, Kommunikation) with toggle to switch to complexity-based grouping (Einfach, Mittel, Fortgeschritten)
- **Expression language support** (e.g., expr-lang) -- full power including string operations, date math, regex
- **UI-first with expression fallback**: Simple conditions use dropdown UI (field, operator, value). Power users can toggle to a raw expression text field with autocomplete and syntax highlighting
- **Chained actions**: Action output feeds into next action (e.g., created invoice ID becomes input to next step). Enables powerful multi-step workflows
- **Soft step limit**: Default maximum of 10 actions per automation chain, admins can raise the limit. Prevents runaway workflows while allowing power users flexibility

### Claude's Discretion
- Exact react-flow node types and edge styling
- Expression language choice (expr-lang vs alternatives)
- Execution log retention and cleanup policy
- Error handling and retry behavior for failed actions
- Template content for the 10-15 pre-built automations beyond the two must-haves
- Polling interval for time-based triggers

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| AUTO-01 | User can create automations using trigger-action model ("When X happens, do Y") | Automation CRUD via AutomationService proto; workflow storage in `automations` table with JSONB trigger/conditions/actions; wizard and react-flow UIs both produce the same workflow JSON |
| AUTO-02 | System provides 10-15 pre-built triggers across all modules | TriggerRegistry with typed trigger definitions; event-based triggers subscribe to EventBus; time-based triggers use polling goroutine; triggers defined per-module covering CRM, Work, Email, Finance, HR, Calendar |
| AUTO-03 | System provides 8-10 pre-built actions across all modules | ActionRegistry with typed action executors; each action calls target service via gRPC; action output returned as map[string]interface{} for chaining to next step |
| AUTO-04 | User can add conditional logic (if/else with field-value conditions, AND/OR operators) | Dual condition model: simple dropdown conditions reuse inbox routing evaluator pattern (AND/OR tree); expr-lang/expr for power-user raw expressions with compile-once caching |
| AUTO-05 | User can view automation execution logs | `automation_executions` table with per-step logging; stores trigger event, condition result, each action result/error, total duration; queryable by automation_id, date range, status |
| AUTO-06 | User can enable/disable automations without deleting them | `is_active` boolean on automations table; inactive workflows skip trigger matching but retain all configuration for re-enabling |
</phase_requirements>

## Standard Stack

### Core (Already in Project)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| pgx/v5 | Latest | PostgreSQL LISTEN/NOTIFY, workflow storage | Already used across all services for pg_notify and DB access |
| EventBus | Internal | Event consumption from pg_notify | Built in Phase 4, wildcard handler dispatching, backlog processing |
| gRPC + protobuf | Latest | Inter-service action execution | All services expose gRPC APIs; automation calls them to execute actions |
| chi/v5 | Latest | HTTP routing for gateway automation routes | Established gateway router |
| TanStack Query | v5 | Frontend data fetching + caching | All modules use TanStack Query hooks |
| Zustand | Latest | UI-only state (wizard step, selected nodes, editor mode) | Consistent with existing store patterns |

### New Dependencies
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| [expr-lang/expr](https://github.com/expr-lang/expr) | v1.17.7 | Expression evaluation for power-user conditions | Memory-safe, side-effect-free, always terminates, static type checking. Compile-once-run-many. Used by Google Cloud, Uber, ByteDance for business rules. |
| [@xyflow/react](https://reactflow.dev) | v12.10.1 | Visual node editor for complex workflows | React 19 compatible, Tailwind CSS 4 support. Widely adopted (500+ npm dependents). Replaces old `reactflow` package. |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| expr-lang/expr | Custom evaluator (inbox routing pattern) | Custom is simpler for AND/OR but cannot handle date math, regex, string functions, nested expressions. expr-lang gives full power without security risk. |
| expr-lang/expr | Grule (full rule engine) | Grule is overkill -- full RETE algorithm, Drools-like rule files. We need expression evaluation, not a rule engine. |
| expr-lang/expr | cel-go (Common Expression Language) | cel-go is Google's alternative but heavier, designed for policy enforcement. expr-lang is more Go-idiomatic and simpler API. |
| @xyflow/react | Custom canvas with SVG | react-flow handles zoom, pan, minimap, connection validation, keyboard shortcuts, touch support. Building custom is months of work. |
| @xyflow/react | Rete.js | Rete is framework-agnostic but complex setup. react-flow integrates natively with React 19 component model. |
| Separate automation binary | Co-hosted in notification binary | Automation needs gRPC clients to ALL services (6+), has its own EventBus listener, and distinct memory/CPU profile. Co-hosting would overload the notification binary and couple unrelated concerns. |

**Installation:**
```bash
# Backend
cd backend && go get github.com/expr-lang/expr@v1.17.7

# Frontend
cd desktop && npm install @xyflow/react
```

## Architecture Patterns

### Recommended Project Structure

```
backend/
├── proto/automation/v1/automation.proto    # AutomationService proto (~20 RPCs)
├── cmd/automation/main.go                  # Automation binary (port :50059, health :9099)
├── internal/automation/
│   ├── workflow/
│   │   ├── repository.go                   # Automation CRUD interface
│   │   ├── postgres_repository.go          # PostgreSQL implementation
│   │   ├── service.go                      # Workflow CRUD, enable/disable, template instantiation
│   │   └── errors.go                       # Domain errors
│   ├── trigger/
│   │   ├── registry.go                     # TriggerRegistry: maps event types to trigger definitions
│   │   ├── matcher.go                      # TriggerMatcher: checks if event matches any active trigger
│   │   ├── poller.go                       # TimeTriggerPoller: scheduled checks for time-based triggers
│   │   └── types.go                        # TriggerDefinition, TriggerConfig structs
│   ├── condition/
│   │   ├── evaluator.go                    # Dual evaluator: simple (AND/OR tree) + expr-lang
│   │   ├── evaluator_test.go               # TDD for condition evaluation
│   │   ├── expr_env.go                     # Expr environment definition with event/entity fields
│   │   └── types.go                        # ConditionConfig structs
│   ├── action/
│   │   ├── registry.go                     # ActionRegistry: maps action types to executors
│   │   ├── executor.go                     # ActionExecutor interface + base implementation
│   │   ├── crm_actions.go                  # CRM actions (update field, create contact, etc.)
│   │   ├── work_actions.go                 # Work actions (create task, etc.)
│   │   ├── email_actions.go                # Email actions (send email, etc.)
│   │   ├── notification_actions.go         # Notification actions (send notification)
│   │   ├── calendar_actions.go             # Calendar actions (create event, etc.)
│   │   ├── biz_actions.go                  # Finance actions (create invoice draft, etc.)
│   │   ├── hr_actions.go                   # HR actions (none in v1, placeholder)
│   │   └── types.go                        # ActionDefinition, ActionResult structs
│   ├── engine/
│   │   ├── engine.go                       # WorkflowEngine: orchestrates trigger->condition->actions
│   │   ├── consumer.go                     # AutomationConsumer: EventBus handler
│   │   └── logger.go                       # ExecutionLogger: writes to automation_executions table
│   └── template/
│       ├── registry.go                     # TemplateRegistry: pre-built automation definitions
│       └── templates.go                    # 10-15 template definitions
├── internal/gateway/
│   └── route_automation.go                 # AutomationRoutes (gateway HTTP routes)
└── migrations/
    ├── 000052_create_automation_tables.up.sql
    └── 000052_create_automation_tables.down.sql

desktop/src/renderer/src/
├── api/
│   ├── automation-client.ts                # Automation API client (fetch wrapper)
│   ├── automation-types.ts                 # TypeScript types
│   └── hooks/
│       └── useAutomation.ts                # TanStack Query hooks
├── modules/automatisierung/
│   ├── AutomatisierungPage.tsx             # Main page (list + template gallery)
│   ├── AutomationWizard.tsx                # 4-step wizard (Trigger->Condition->Action->Review)
│   ├── AutomationEditor.tsx                # react-flow visual editor
│   ├── TriggerSelector.tsx                 # Trigger selection with module grouping
│   ├── ConditionBuilder.tsx                # Dropdown conditions + expr toggle
│   ├── ActionConfigurator.tsx              # Action selection and parameter configuration
│   ├── ExecutionLogViewer.tsx              # Execution log table with filters
│   ├── TemplateGallery.tsx                 # Browsable template gallery with grouping toggle
│   └── nodes/                              # react-flow custom node types
│       ├── TriggerNode.tsx
│       ├── ConditionNode.tsx
│       ├── ActionNode.tsx
│       └── nodeTypes.ts
└── stores/automatisierung.ts               # UI state (wizard step, editor mode, selected automation)
```

### Pattern 1: Automation as EventBus Consumer

**What:** The automation engine subscribes to the same pg_notify `events` channel as the notification service, but processes events through a workflow matching pipeline instead of creating notifications.
**When to use:** For all event-based triggers.

```go
// AutomationConsumer listens on EventBus and triggers matching workflows.
type AutomationConsumer struct {
    matcher *trigger.TriggerMatcher
    engine  *WorkflowEngine
}

func (ac *AutomationConsumer) HandleEvent(ctx context.Context, evt models.EventPayload) error {
    // Skip automation-originated events to prevent loops
    if evt.ModuleID == "automation" {
        return nil
    }

    // Find all active automations whose trigger matches this event
    matches, err := ac.matcher.FindMatching(ctx, evt)
    if err != nil {
        slog.Error("trigger matching failed", "event_type", evt.Type, "error", err)
        return nil // Non-fatal: don't block the event bus
    }

    // Execute each matching automation asynchronously
    for _, automation := range matches {
        go func(auto models.Automation) {
            execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
            defer cancel()
            if err := ac.engine.Execute(execCtx, auto, evt); err != nil {
                slog.Error("automation execution failed",
                    "automation_id", auto.ID,
                    "event_type", evt.Type,
                    "error", err,
                )
            }
        }(automation)
    }

    return nil
}
```

### Pattern 2: Dual Condition Evaluation (Simple + expr-lang)

**What:** Conditions have two modes: simple (AND/OR tree from inbox routing) and expression (expr-lang). The workflow JSON stores which mode is used.
**When to use:** All condition evaluation in the automation engine.

```go
// ConditionConfig stores the condition configuration for a workflow step.
type ConditionConfig struct {
    Mode       string          `json:"mode"`       // "simple" or "expression"
    Simple     *models.Condition `json:"simple,omitempty"`     // AND/OR tree
    Expression string          `json:"expression,omitempty"` // expr-lang expression string
}

// Evaluator handles both simple and expression-based condition evaluation.
type Evaluator struct {
    exprCache sync.Map // map[string]*vm.Program -- compiled expr programs
}

func (e *Evaluator) Evaluate(ctx context.Context, config ConditionConfig, env map[string]any) (bool, error) {
    switch config.Mode {
    case "simple":
        return e.evaluateSimple(config.Simple, env), nil
    case "expression":
        return e.evaluateExpr(config.Expression, env)
    default:
        return true, nil // No condition = always true
    }
}

func (e *Evaluator) evaluateExpr(expression string, env map[string]any) (bool, error) {
    // Check cache for compiled program
    cached, ok := e.exprCache.Load(expression)
    if !ok {
        program, err := expr.Compile(expression, expr.Env(env), expr.AsBool())
        if err != nil {
            return false, fmt.Errorf("compile expression: %w", err)
        }
        e.exprCache.Store(expression, program)
        cached = program
    }

    output, err := expr.Run(cached.(*vm.Program), env)
    if err != nil {
        return false, fmt.Errorf("evaluate expression: %w", err)
    }

    result, ok := output.(bool)
    if !ok {
        return false, fmt.Errorf("expression did not return bool, got %T", output)
    }
    return result, nil
}
```

### Pattern 3: Action Execution via gRPC with Output Chaining

**What:** Each action type has an executor that calls the target service via gRPC and returns a result map. The result map feeds into the next action's environment.
**When to use:** All action execution in the workflow engine.

```go
// ActionExecutor executes a single action and returns output for chaining.
type ActionExecutor interface {
    // Type returns the action type identifier (e.g., "create_task", "send_email").
    Type() string
    // Execute runs the action and returns output variables for chaining.
    Execute(ctx context.Context, config json.RawMessage, env map[string]any) (ActionResult, error)
}

// ActionResult holds the output of an action execution.
type ActionResult struct {
    Success bool              `json:"success"`
    Output  map[string]any    `json:"output,omitempty"`  // e.g., {"created_invoice_id": "uuid..."}
    Error   string            `json:"error,omitempty"`
}

// WorkflowEngine orchestrates the full trigger->condition->actions pipeline.
func (we *WorkflowEngine) Execute(ctx context.Context, auto models.Automation, evt models.EventPayload) error {
    execID := uuid.New()
    startedAt := time.Now()
    env := we.buildInitialEnv(auto, evt)

    // Log execution start
    we.logger.LogStart(ctx, execID, auto.ID, evt)

    // Evaluate conditions
    condResult, err := we.condEvaluator.Evaluate(ctx, auto.Conditions, env)
    if err != nil {
        we.logger.LogConditionError(ctx, execID, err)
        return err
    }
    if !condResult {
        we.logger.LogConditionSkipped(ctx, execID)
        return nil // Condition not met, skip
    }

    // Execute actions in order, chaining outputs
    for i, actionConfig := range auto.Actions {
        if i >= auto.MaxSteps {
            we.logger.LogStepLimitReached(ctx, execID, i)
            break
        }

        executor, ok := we.actionRegistry.Get(actionConfig.Type)
        if !ok {
            we.logger.LogActionError(ctx, execID, i, fmt.Errorf("unknown action type: %s", actionConfig.Type))
            continue
        }

        result, err := executor.Execute(ctx, actionConfig.Config, env)
        we.logger.LogActionResult(ctx, execID, i, actionConfig.Type, result, err)

        if err != nil {
            // Continue or abort based on action's error_handling setting
            if actionConfig.OnError == "abort" {
                return err
            }
            continue
        }

        // Chain output into env for next action
        for k, v := range result.Output {
            env["prev_"+k] = v
            env["step_"+strconv.Itoa(i)+"_"+k] = v
        }
    }

    we.logger.LogComplete(ctx, execID, time.Since(startedAt))
    return nil
}
```

### Pattern 4: Time-Based Trigger Polling

**What:** Some triggers are not event-driven (e.g., "invoice overdue for 7 days"). These use a polling goroutine that periodically checks database state.
**When to use:** Time-based/scheduled triggers only.

```go
// TimeTriggerPoller checks for time-based trigger conditions at regular intervals.
type TimeTriggerPoller struct {
    interval   time.Duration
    workflowRepo workflow.Repository
    engine     *WorkflowEngine
    pool       *pgxpool.Pool
}

func (p *TimeTriggerPoller) Start(ctx context.Context) {
    ticker := time.NewTicker(p.interval) // Recommended: 5 minutes
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            p.checkTimeTriggers(ctx)
        }
    }
}

func (p *TimeTriggerPoller) checkTimeTriggers(ctx context.Context) {
    // Query active automations with time-based triggers
    automations, err := p.workflowRepo.ListActiveByTriggerType(ctx, "time_based")
    if err != nil {
        slog.Error("time trigger poll failed", "error", err)
        return
    }

    for _, auto := range automations {
        // Each time trigger defines a query (e.g., "overdue invoices not yet dunned")
        // The poller runs the query and fires the automation for each matching row
        matches, err := p.runTimeTriggerQuery(ctx, auto.Trigger)
        if err != nil {
            slog.Error("time trigger query failed",
                "automation_id", auto.ID, "error", err)
            continue
        }

        for _, match := range matches {
            evt := models.EventPayload{
                Type:       auto.Trigger.EventType,
                ModuleID:   "automation",
                ResourceID: match.ResourceID,
                Timestamp:  time.Now(),
                Payload:    match.Payload,
            }
            go p.engine.Execute(ctx, auto, evt)
        }
    }
}
```

### Pattern 5: Trigger and Action Registries

**What:** Centralized registries define all available triggers and actions with metadata (name, description, module, parameter schema) for both backend matching and frontend UI rendering.
**When to use:** For building the trigger/action catalogs and for frontend type generation.

```go
// TriggerDefinition describes an available trigger.
type TriggerDefinition struct {
    Type        string                 `json:"type"`         // e.g., "crm.deal.stage_changed"
    Module      string                 `json:"module"`       // e.g., "crm"
    Name        string                 `json:"name"`         // e.g., "Deal-Phase geaendert"
    Description string                 `json:"description"`  // Human-readable
    EventType   string                 `json:"event_type"`   // Matching event.Types constant
    Fields      []TriggerField         `json:"fields"`       // Available fields for conditions
    Config      []ConfigParam          `json:"config,omitempty"` // Trigger-specific config params
}

// ActionDefinition describes an available action.
type ActionDefinition struct {
    Type        string        `json:"type"`         // e.g., "create_task"
    Module      string        `json:"module"`       // e.g., "work"
    Name        string        `json:"name"`         // e.g., "Aufgabe erstellen"
    Description string        `json:"description"`
    Params      []ConfigParam `json:"params"`       // Required/optional parameters
    OutputFields []OutputField `json:"output_fields"` // Fields available for chaining
}

// TriggerField describes a field available for conditions on a trigger.
type TriggerField struct {
    Key       string `json:"key"`        // e.g., "deal.value", "deal.stage_name"
    Label     string `json:"label"`      // e.g., "Deal-Wert"
    Type      string `json:"type"`       // "string", "number", "boolean", "date"
    Operators []string `json:"operators"` // Applicable operators
}
```

### Anti-Patterns to Avoid

- **Synchronous action execution blocking the EventBus:** Execute automations in goroutines. The EventBus handler must return quickly. If an action takes 10 seconds (e.g., sending an email), it blocks all other event handlers. Use `go engine.Execute(...)` with a timeout context.
- **Storing compiled expr programs in the database:** Compile on first use and cache in memory (sync.Map). Compiled programs are safe for concurrent use. Invalidate cache when the automation is updated.
- **Calling services via HTTP instead of gRPC:** The automation service runs in the backend network. All inter-service calls use gRPC directly. HTTP is only for the gateway-to-frontend path.
- **Automation triggering itself (infinite loops):** Skip events where `module_id == "automation"`. Also add a circuit breaker: if an automation fires more than N times per minute, auto-disable it and notify the admin.
- **Building a custom visual editor instead of react-flow:** react-flow handles zoom, pan, minimap, connection validation, keyboard shortcuts, touch support, and accessibility. Building custom is months of work.
- **Putting the automation service in the notification binary:** Automation needs gRPC clients to 6+ services and has independent scaling needs. Co-hosting would couple unrelated concerns and overload the notification binary.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Expression evaluation | Custom parser/interpreter | expr-lang/expr | Handles operator precedence, type coercion, date math, regex, string functions. Custom parser is 1000+ LOC for half the features. |
| Visual workflow editor | Custom canvas + SVG | @xyflow/react | Zoom, pan, minimap, connection validation, keyboard shortcuts, touch support, accessibility. 6+ months of work to build custom. |
| Event consumption | New event listener | Existing EventBus + pg_notify | Phase 4 infrastructure works. Just register another handler on the same bus. |
| Event emission | New event transport | Existing event.EmitEvent() + pg_notify | Already built, battle-tested across 9+ services. |
| Simple conditions (AND/OR) | New condition evaluator | Adapt inbox/routing/evaluator.go pattern | The evaluator pattern exists and is tested. Generalize it for automation use. |
| Retry/backoff for failed actions | Custom retry logic | Simple retry with exponential backoff | 3 retries with 1s/2s/4s backoff. No external library needed -- ~20 LOC in Go. |

**Key insight:** The automation engine is primarily an orchestration layer. It consumes events (existing), evaluates conditions (expr-lang + existing AND/OR pattern), and calls services (existing gRPC APIs). The new code is the glue: trigger matching, workflow storage, action dispatching, execution logging, and the frontend builder.

## Common Pitfalls

### Pitfall 1: Infinite Automation Loops
**What goes wrong:** Automation A fires action "update CRM deal", which emits `crm.deal.stage_changed`, which triggers Automation A again. Infinite loop.
**Why it happens:** Actions emit events, and those events can trigger the same or related automations.
**How to avoid:** Three defenses: (1) Skip events where `module_id == "automation"`. (2) Track execution chains: each execution carries a `chain_id`. If the same automation appears twice in a chain, abort. (3) Circuit breaker: if an automation fires > 100 times per hour, auto-disable and notify admin.
**Warning signs:** CPU spike on automation service, exponential growth in execution logs, same automation_id appearing repeatedly.

### Pitfall 2: Action Execution Timeout Cascades
**What goes wrong:** An action calls a service via gRPC that is slow or down. The 30-second timeout blocks the goroutine. 50 events arrive simultaneously, spawning 50 goroutines all waiting on the same slow service.
**Why it happens:** No concurrency limiting on automation execution.
**How to avoid:** Use a semaphore (channel of fixed size, e.g., 20) to limit concurrent automation executions. Queue excess executions. Each action call has its own 10-second timeout. The overall execution has a 30-second timeout.
**Warning signs:** High goroutine count, increasing execution latency, service timeouts in logs.

### Pitfall 3: expr-lang Environment Type Mismatches
**What goes wrong:** User writes expression `deal.value > 10000` but `deal.value` is a string in the environment, not a number. The expression fails at runtime.
**Why it happens:** Event payloads carry JSON, which may deserialize numbers as float64 or string.
**How to avoid:** Define a strongly-typed environment struct for each trigger type. Use expr.Env() with the struct type at compile time so expr-lang can type-check. Convert event payload fields to the correct Go types before building the environment map.
**Warning signs:** Runtime type errors in execution logs, user confusion about why conditions don't match.

### Pitfall 4: Stale Trigger Match Cache
**What goes wrong:** Admin creates a new automation, but events don't trigger it because the trigger matcher has a cached list of active automations.
**Why it happens:** In-memory caching of active automations without invalidation.
**How to avoid:** Use a short TTL cache (30-60 seconds) for active automations. Alternatively, pg_notify on automation CRUD to invalidate the cache immediately. The 30-60 second delay is acceptable for most use cases.
**Warning signs:** New automations not firing, delay between creation and first execution.

### Pitfall 5: Time-Based Trigger Double-Execution
**What goes wrong:** The 5-minute poller fires twice for the same overdue invoice because the first execution hasn't updated the invoice status yet.
**Why it happens:** Polling checks a condition (e.g., "overdue > 7 days AND no dunning sent") that is true until the action completes and updates state.
**How to avoid:** Use a `last_checked_at` timestamp per time-based automation. The poller query includes `AND resource_id NOT IN (SELECT resource_id FROM automation_executions WHERE automation_id = $1 AND created_at > $2)`. This deduplicates based on recent execution history.
**Warning signs:** Duplicate dunning emails, double task creation.

### Pitfall 6: Frontend Wizard/Editor State Divergence
**What goes wrong:** User switches from wizard to node editor mid-edit. The node editor doesn't reflect the wizard changes, or vice versa.
**Why it happens:** Two different state representations not kept in sync.
**How to avoid:** Single canonical workflow JSON format stored in Zustand. Both wizard and editor read from and write to the same store. The wizard is a structured view of the JSON. The editor is a visual view. Switching modes re-renders from the same source.
**Warning signs:** Lost changes when switching modes, inconsistent automation behavior between creation modes.

## Code Examples

### Automation Database Schema

```sql
-- 000052_create_automation_tables.up.sql

-- Automation workflows
CREATE TABLE automations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    description TEXT,
    -- Scope: who can this automation affect?
    scope VARCHAR(20) NOT NULL DEFAULT 'personal'
        CHECK (scope IN ('personal', 'team', 'organization')),
    owner_id UUID NOT NULL REFERENCES users(id),
    -- Trigger configuration (what event starts this?)
    trigger_type VARCHAR(100) NOT NULL,  -- e.g., "crm.deal.stage_changed"
    trigger_config JSONB NOT NULL DEFAULT '{}',
    -- Conditions (when should it fire?)
    conditions JSONB NOT NULL DEFAULT '{}',  -- ConditionConfig JSON
    -- Actions (what should it do?)
    actions JSONB NOT NULL DEFAULT '[]',  -- []ActionConfig JSON
    -- State
    is_active BOOLEAN NOT NULL DEFAULT true,
    max_steps INTEGER NOT NULL DEFAULT 10,
    -- Template metadata
    template_id VARCHAR(100),  -- NULL if custom, template key if from template
    -- Timestamps
    last_triggered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Active automations by trigger type (most common query)
CREATE INDEX idx_automations_trigger_active ON automations(trigger_type)
    WHERE is_active = true;

-- Owner's automations
CREATE INDEX idx_automations_owner ON automations(owner_id, created_at DESC);

-- Automation execution logs
CREATE TABLE automation_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    automation_id UUID NOT NULL REFERENCES automations(id) ON DELETE CASCADE,
    chain_id UUID NOT NULL,  -- For loop detection across chained executions
    trigger_event JSONB NOT NULL,  -- The event that triggered this execution
    condition_result BOOLEAN NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'running'
        CHECK (status IN ('running', 'completed', 'failed', 'skipped', 'aborted')),
    steps JSONB NOT NULL DEFAULT '[]',  -- [{action_type, input, output, error, duration_ms}]
    error_message TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    duration_ms INTEGER
);

-- Query executions by automation (for log viewer)
CREATE INDEX idx_automation_executions_automation ON automation_executions(automation_id, started_at DESC);

-- Query executions by status (for monitoring)
CREATE INDEX idx_automation_executions_status ON automation_executions(status, started_at DESC)
    WHERE status IN ('failed', 'running');

-- Cleanup: drop old execution logs (retention policy)
CREATE INDEX idx_automation_executions_cleanup ON automation_executions(started_at)
    WHERE completed_at IS NOT NULL;

-- Automation templates (pre-built, immutable reference)
CREATE TABLE automation_templates (
    id VARCHAR(100) PRIMARY KEY,  -- e.g., "deal-won-invoice"
    name VARCHAR(200) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(50) NOT NULL,  -- "vertrieb", "personal", "finanzen", "kommunikation"
    complexity VARCHAR(20) NOT NULL DEFAULT 'einfach'
        CHECK (complexity IN ('einfach', 'mittel', 'fortgeschritten')),
    trigger_type VARCHAR(100) NOT NULL,
    trigger_config JSONB NOT NULL DEFAULT '{}',
    conditions JSONB NOT NULL DEFAULT '{}',
    actions JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed templates are inserted via migration or application startup
```

### expr-lang Environment for Automation Conditions

```go
// ExprEnv defines the typed environment available to expr-lang expressions.
// Fields are populated from the trigger event and resolved entity data.
type ExprEnv struct {
    // Event metadata
    EventType string    `expr:"event_type"`
    Module    string    `expr:"module"`
    Timestamp time.Time `expr:"timestamp"`

    // CRM fields (populated when trigger is CRM-related)
    Deal    *DealEnv    `expr:"deal"`
    Contact *ContactEnv `expr:"contact"`
    Company *CompanyEnv `expr:"company"`

    // Finance fields
    Invoice *InvoiceEnv `expr:"invoice"`
    Quote   *QuoteEnv   `expr:"quote"`

    // HR fields
    Leave  *LeaveEnv  `expr:"leave"`
    Shift  *ShiftEnv  `expr:"shift"`

    // Work fields
    Task    *TaskEnv    `expr:"task"`
    Project *ProjectEnv `expr:"project"`

    // Previous step outputs (for chaining)
    Prev map[string]any `expr:"prev"`
}

type DealEnv struct {
    ID        string  `expr:"id"`
    Name      string  `expr:"name"`
    Value     float64 `expr:"value"`
    StageName string  `expr:"stage_name"`
    StageOld  string  `expr:"stage_old"`
    OwnerID   string  `expr:"owner_id"`
}

type InvoiceEnv struct {
    ID         string  `expr:"id"`
    Number     string  `expr:"number"`
    Total      float64 `expr:"total"`
    Status     string  `expr:"status"`
    DueDate    time.Time `expr:"due_date"`
    DaysOverdue int    `expr:"days_overdue"`
}

// Usage in condition evaluation:
// "deal.value > 10000 and deal.stage_name == 'Won'"
// "invoice.days_overdue > 7 and invoice.status == 'sent'"
// "leave.days > 3 and leave.type == 'vacation'"
```

### Pre-Built Automation Templates (Examples)

```go
var Templates = []AutomationTemplate{
    // MUST-HAVE: Invoice overdue -> Dunning workflow
    {
        ID:          "invoice-overdue-dunning",
        Name:        "Ueberfaellige Rechnung -> Mahnlauf",
        Description: "Erstellt automatisch eine Mahnung wenn eine Rechnung ueberfaellig wird",
        Category:    "finanzen",
        Complexity:  "mittel",
        TriggerType: "time_based",
        TriggerConfig: TriggerConfig{
            Query: "overdue_invoices_without_dunning",
            Interval: "5m",
        },
        Conditions: ConditionConfig{
            Mode: "expression",
            Expression: "invoice.days_overdue >= 14",
        },
        Actions: []ActionConfig{
            {Type: "create_dunning", Config: map[string]any{"level": 1}},
            {Type: "send_notification", Config: map[string]any{
                "title": "Mahnung erstellt",
                "body":  "Rechnung {{invoice.number}} ist seit {{invoice.days_overdue}} Tagen ueberfaellig",
            }},
        },
    },
    // MUST-HAVE: Leave approved -> Calendar event + team notification
    {
        ID:          "leave-approved-calendar",
        Name:        "Urlaub genehmigt -> Kalender + Benachrichtigung",
        Description: "Erstellt einen Kalendereintrag und benachrichtigt das Team wenn Urlaub genehmigt wird",
        Category:    "personal",
        Complexity:  "einfach",
        TriggerType: event.EventLeaveApproved,
        Conditions:  ConditionConfig{Mode: "simple", Simple: nil}, // Always fire
        Actions: []ActionConfig{
            {Type: "create_calendar_event", Config: map[string]any{
                "title":    "{{leave.employee_name}} - Urlaub",
                "all_day":  true,
                "calendar": "team_absence",
            }},
            {Type: "send_notification", Config: map[string]any{
                "target": "team_members",
                "title":  "Urlaub genehmigt",
                "body":   "{{leave.employee_name}} ist vom {{leave.start_date}} bis {{leave.end_date}} im Urlaub",
            }},
        },
    },
    // Deal stage changed -> Create task
    {
        ID:          "deal-won-task",
        Name:        "Deal gewonnen -> Aufgabe erstellen",
        Description: "Erstellt eine Follow-up Aufgabe wenn ein Deal gewonnen wird",
        Category:    "vertrieb",
        Complexity:  "einfach",
        TriggerType: event.EventCRMDealStageChanged,
        Conditions: ConditionConfig{
            Mode:       "expression",
            Expression: "deal.stage_name == 'Won' or deal.stage_name == 'Gewonnen'",
        },
        Actions: []ActionConfig{
            {Type: "create_task", Config: map[string]any{
                "title":    "Onboarding: {{deal.name}}",
                "assignee": "{{deal.owner_id}}",
                "priority": "high",
                "due_days": 3,
            }},
        },
    },
    // Deal won -> Create invoice draft
    {
        ID:          "deal-won-invoice",
        Name:        "Deal gewonnen -> Rechnung erstellen",
        Description: "Erstellt automatisch einen Rechnungsentwurf wenn ein Deal gewonnen wird",
        Category:    "vertrieb",
        Complexity:  "mittel",
        TriggerType: event.EventCRMDealStageChanged,
        Conditions: ConditionConfig{
            Mode:       "expression",
            Expression: "deal.stage_name == 'Won' and deal.value > 0",
        },
        Actions: []ActionConfig{
            {Type: "create_invoice_draft", Config: map[string]any{
                "from_deal": "{{deal.id}}",
            }},
            {Type: "send_notification", Config: map[string]any{
                "target": "deal_owner",
                "title":  "Rechnung erstellt",
                "body":   "Rechnungsentwurf fuer Deal '{{deal.name}}' ({{deal.value}} EUR) erstellt",
            }},
        },
    },
}
```

### AutomationService Proto (Sketch)

```protobuf
service AutomationService {
    // Workflow CRUD
    rpc CreateAutomation(CreateAutomationRequest) returns (CreateAutomationResponse);
    rpc UpdateAutomation(UpdateAutomationRequest) returns (UpdateAutomationResponse);
    rpc DeleteAutomation(DeleteAutomationRequest) returns (DeleteAutomationResponse);
    rpc GetAutomation(GetAutomationRequest) returns (GetAutomationResponse);
    rpc ListAutomations(ListAutomationsRequest) returns (ListAutomationsResponse);

    // Enable/disable
    rpc EnableAutomation(EnableAutomationRequest) returns (EnableAutomationResponse);
    rpc DisableAutomation(DisableAutomationRequest) returns (DisableAutomationResponse);

    // Execution logs
    rpc ListExecutions(ListExecutionsRequest) returns (ListExecutionsResponse);
    rpc GetExecution(GetExecutionRequest) returns (GetExecutionResponse);

    // Trigger/Action catalog (for frontend UI rendering)
    rpc ListTriggerDefinitions(ListTriggerDefinitionsRequest) returns (ListTriggerDefinitionsResponse);
    rpc ListActionDefinitions(ListActionDefinitionsRequest) returns (ListActionDefinitionsResponse);

    // Templates
    rpc ListTemplates(ListTemplatesRequest) returns (ListTemplatesResponse);
    rpc CreateFromTemplate(CreateFromTemplateRequest) returns (CreateFromTemplateResponse);

    // Testing
    rpc TestCondition(TestConditionRequest) returns (TestConditionResponse);
    rpc DryRunAutomation(DryRunAutomationRequest) returns (DryRunAutomationResponse);

    // Statistics
    rpc GetAutomationStats(GetAutomationStatsRequest) returns (GetAutomationStatsResponse);
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Polling for state changes | Event-driven via pg_notify | Phase 4 (2026-02-07) | Real-time trigger matching without polling |
| Custom condition parser | expr-lang compile+run | Industry standard since ~2023 | Full expression power without security risk |
| reactflow (v11) | @xyflow/react (v12) | 2024 | React 19 support, better TypeScript types |
| Inbox routing evaluator (string-only) | Dual: simple + expr-lang | Phase 16 (new) | Power users get date math, regex, nested logic |

**Already established patterns in this project:**
- Dual-write: events table + pg_notify (durability + real-time)
- EventBus with wildcard + type-specific handlers
- PGEventEmitter per service package with nil-safe SetEventEmitter()
- RouteRegistrar pattern for gateway route modules
- Condition evaluator (AND/OR tree) in inbox/routing/evaluator.go
- gRPC lazy client pattern in gateway (ServiceRegistry.GetConnection)

## Open Questions

1. **Should the automation service have its own EventBus or share with notification?**
   - What we know: The notification binary already runs an EventBus. The automation binary would need its own.
   - What's unclear: Whether two independent pg_notify LISTEN connections on the same channel cause issues.
   - Recommendation: Each binary runs its own EventBus instance. PostgreSQL supports multiple LISTEN connections on the same channel -- all connected clients receive every notification. This is by design and is the correct approach. The two services process events independently (notification creates inbox items; automation runs workflows).

2. **How should template variable substitution work (e.g., `{{deal.name}}`)?**
   - What we know: Templates use mustache-style placeholders. Actions need resolved values from the trigger event and prior action outputs.
   - What's unclear: Whether to use a template engine or simple string replacement.
   - Recommendation: Simple strings.NewReplacer() for action config rendering. The variables are known at execution time (from the environment map). No need for a full template engine -- the patterns are flat key-value substitutions. For complex rendering (e.g., email bodies), delegate to the target service.

3. **Execution log retention policy**
   - What we know: Execution logs can grow large with high-frequency automations.
   - What's unclear: How long to retain and how to clean up.
   - Recommendation: 90-day retention for all logs. Background goroutine runs daily: `DELETE FROM automation_executions WHERE completed_at < NOW() - INTERVAL '90 days'`. Configurable per-tenant via a setting. Failed executions retained for 180 days for debugging.

4. **Retry behavior for failed actions**
   - What we know: Actions can fail (service unavailable, invalid data, etc.).
   - What's unclear: Whether to retry immediately, with backoff, or not at all.
   - Recommendation: Default is no retry (fail-fast). Each action can optionally specify `retry: {max: 3, backoff: "exponential"}` in its config. Retry is per-action, not per-workflow. If an action fails after retries, log the failure and continue to the next action (unless `on_error: "abort"` is set).

## Sources

### Primary (HIGH confidence)
- Project codebase analysis: `backend/internal/notification/event/bus.go` -- EventBus with LISTEN/NOTIFY, handler registration, backlog processing
- Project codebase analysis: `backend/internal/notification/event/types.go` -- All event type constants across modules
- Project codebase analysis: `backend/internal/notification/event/emit.go` -- EmitEvent via pg_notify
- Project codebase analysis: `backend/internal/inbox/routing/evaluator.go` -- AND/OR condition evaluator (reusable pattern)
- Project codebase analysis: `backend/internal/inbox/routing/evaluator_test.go` -- Condition evaluator test suite
- Project codebase analysis: `backend/internal/models/inbox.go` -- Condition/Action JSONB models (designed for reuse)
- Project codebase analysis: `backend/cmd/notification/main.go` -- EventBus wiring, InboxConsumer pattern, co-hosting pattern
- Project codebase analysis: `backend/internal/gateway/route_inbox.go` -- RouteRegistrar pattern, gRPC client lazy loading
- [expr-lang/expr documentation](https://expr-lang.org/) -- Expression language API, operators, functions
- [expr-lang/expr releases](https://github.com/expr-lang/expr/releases/tag/v1.17.7) -- v1.17.7 release (latest)
- [@xyflow/react](https://reactflow.dev) -- React Flow v12 documentation, React 19 compatibility
- [@xyflow/react npm](https://www.npmjs.com/package/@xyflow/react) -- v12.10.1 (latest)

### Secondary (MEDIUM confidence)
- [PostgreSQL NOTIFY documentation](https://www.postgresql.org/docs/current/sql-notify.html) -- Multiple listeners on same channel supported
- [expr-lang use cases](https://wundergraph.com/blog/expr-lang-go-centric-expression-language) -- Google Cloud, Uber, ByteDance production usage
- [React Flow React 19 update](https://reactflow.dev/whats-new/2025-10-28) -- Tailwind CSS 4 + React 19 compatibility confirmed

### Tertiary (LOW confidence)
- None -- all findings verified with primary or secondary sources

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- expr-lang is well-documented, @xyflow/react actively maintained, all other libraries already in project
- Architecture: HIGH -- event infrastructure exists, patterns well-established across 15 prior phases, automation is a natural extension
- Pitfalls: HIGH -- infinite loop prevention, timeout handling, and cache invalidation are well-understood patterns in event-driven systems
- Frontend (react-flow): MEDIUM -- React 19 compatibility confirmed but exact integration with existing Tailwind/Radix setup needs validation during implementation

**Research date:** 2026-02-20
**Valid until:** 2026-03-20 (stable domain, no fast-moving dependencies)
