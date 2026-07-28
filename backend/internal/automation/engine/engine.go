package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/automation/action"
	"github.com/kmuhub/kmuhub/internal/automation/condition"
	"github.com/kmuhub/kmuhub/internal/automation/workflow"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/event"
)

const (
	// maxConcurrentExecutions is the global upper bound on parallel workflow
	// executions across all tenants.
	maxConcurrentExecutions = 20

	// maxConcurrentExecutionsPerTenant is the per-tenant upper bound.
	// A single tenant can hold at most this many slots simultaneously, which
	// prevents noisy-neighbour starvation of other tenants.
	maxConcurrentExecutionsPerTenant = 5

	// actionTimeout is the per-action execution timeout.
	actionTimeout = 10 * time.Second

	// circuitBreakerThreshold auto-disables automations exceeding this rate.
	circuitBreakerThreshold = 100 // executions per hour
)

// WorkflowEngine orchestrates the trigger -> condition -> actions pipeline.
// It evaluates conditions, runs action steps sequentially with output chaining,
// and logs every execution step.
type WorkflowEngine struct {
	condEvaluator  *condition.Evaluator
	actionRegistry *action.ActionRegistry
	logger         *ExecutionLogger
	workflowRepo   workflow.Repository
	execRepo       workflow.ExecutionRepository

	// semaphore is the global concurrency cap across all tenants.
	semaphore chan struct{}

	// mu protects both executionCounts (circuit breaker) and tenantSemaphores.
	mu sync.Mutex

	// Circuit breaker: tracks executions per automation per hour.
	executionCounts map[uuid.UUID]*circuitState

	// tenantSemaphores is a per-tenant concurrency cap (maxConcurrentExecutionsPerTenant).
	// The map is lazily populated and never evicted: with the current single-tenant
	// deployment this is a non-issue; in multi-tenant operation the number of distinct
	// tenants is bounded and small (<<1000), so unbounded growth is acceptable.
	tenantSemaphores map[uuid.UUID]chan struct{}
}

// circuitState tracks execution count and window for circuit breaker.
type circuitState struct {
	count     int
	windowEnd time.Time
}

// NewWorkflowEngine creates a new workflow engine.
func NewWorkflowEngine(
	evaluator *condition.Evaluator,
	registry *action.ActionRegistry,
	logger *ExecutionLogger,
	repo workflow.Repository,
	execRepo workflow.ExecutionRepository,
) *WorkflowEngine {
	return &WorkflowEngine{
		condEvaluator:    evaluator,
		actionRegistry:   registry,
		logger:           logger,
		workflowRepo:     repo,
		execRepo:         execRepo,
		semaphore:        make(chan struct{}, maxConcurrentExecutions),
		executionCounts:  make(map[uuid.UUID]*circuitState),
		tenantSemaphores: make(map[uuid.UUID]chan struct{}),
	}
}

// Execute runs a single automation workflow for a given event.
// This method satisfies the trigger.EngineExecutor interface.
func (we *WorkflowEngine) Execute(ctx context.Context, auto models.Automation, evt models.EventPayload) error {
	// Loop prevention: skip events from the automation module itself
	if evt.ModuleID == event.ModuleAutomation {
		slog.Debug("skipping automation-triggered event (loop prevention)",
			"automation_id", auto.ID,
			"event_type", evt.Type,
		)
		return nil
	}

	// Circuit breaker check
	if we.isCircuitBreakerOpen(auto.ID) {
		slog.Warn("circuit breaker open, auto-disabling automation",
			"automation_id", auto.ID,
			"threshold", circuitBreakerThreshold,
		)
		if err := we.workflowRepo.SetActive(ctx, auto.ID, auto.TenantID, false); err != nil {
			slog.Error("failed to auto-disable automation", "automation_id", auto.ID, "error", err)
		}
		return workflow.ErrCircuitBreakerOpen
	}

	// Acquire global semaphore slot first (non-blocking: drop if full).
	select {
	case we.semaphore <- struct{}{}:
		// acquired; released after the tenant slot is also released (see defer below)
	default:
		slog.Warn("global execution semaphore full, skipping",
			"automation_id", auto.ID,
			"tenant_id", auto.TenantID,
			"max_concurrent", maxConcurrentExecutions,
		)
		return nil
	}

	// Acquire per-tenant semaphore slot (non-blocking: release global and drop).
	tenantID := auto.TenantID
	if tenantID == uuid.Nil {
		slog.Warn("automation has nil tenant_id, using nil bucket",
			"automation_id", auto.ID,
		)
	}
	tenantSem := we.tenantSemaphore(tenantID)
	select {
	case tenantSem <- struct{}{}:
		// acquired both slots; set up deferred release in reverse acquisition order
		defer func() {
			<-tenantSem
			<-we.semaphore
		}()
	default:
		// release global slot we already acquired
		<-we.semaphore
		slog.Warn("per-tenant execution semaphore full, skipping",
			"automation_id", auto.ID,
			"tenant_id", tenantID,
			"max_concurrent_per_tenant", maxConcurrentExecutionsPerTenant,
		)
		return nil
	}

	// Create execution context
	executionID := uuid.New()
	chainID := uuid.New()
	startTime := time.Now()

	// Build initial environment from event payload
	env := buildEnvFromPayload(evt)

	// Log execution start
	we.logger.LogStart(ctx, executionID, auto.TenantID, auto.ID, chainID, evt)

	// Parse conditions
	var condConfig models.ConditionConfig
	if len(auto.Conditions) > 0 {
		if err := json.Unmarshal(auto.Conditions, &condConfig); err != nil {
			we.logger.LogConditionError(ctx, executionID, auto.TenantID, err)
			return fmt.Errorf("unmarshal conditions: %w", err)
		}
	}

	// Evaluate conditions
	condResult, err := we.condEvaluator.Evaluate(ctx, condConfig, env)
	if err != nil {
		we.logger.LogConditionError(ctx, executionID, auto.TenantID, err)
		return fmt.Errorf("evaluate conditions: %w", err)
	}

	if !condResult {
		we.logger.LogConditionSkipped(ctx, executionID, auto.TenantID)
		slog.Debug("condition not met, skipping execution",
			"automation_id", auto.ID,
			"execution_id", executionID,
		)
		return nil
	}

	// Parse actions
	var actions []models.ActionConfig
	if auto.Actions != nil {
		if err := json.Unmarshal(auto.Actions, &actions); err != nil {
			we.logger.LogConditionError(ctx, executionID, auto.TenantID, err)
			return fmt.Errorf("unmarshal actions: %w", err)
		}
	}

	// Determine max steps
	maxSteps := auto.MaxSteps
	if maxSteps <= 0 {
		maxSteps = 10
	}

	// Execute actions sequentially
	for i, actionCfg := range actions {
		if i >= maxSteps {
			we.logger.LogStepLimitReached(ctx, executionID, auto.TenantID, maxSteps)
			return workflow.ErrMaxStepsExceeded
		}

		executor, ok := we.actionRegistry.Get(actionCfg.Type)
		if !ok {
			errMsg := fmt.Sprintf("unknown action type: %s", actionCfg.Type)
			we.logger.LogActionResult(ctx, executionID, auto.TenantID, i, actionCfg.Type,
				action.ActionResult{Success: false, Error: errMsg}, fmt.Errorf("%s", errMsg))

			if actionCfg.OnError == models.OnErrorAbort {
				break
			}
			continue
		}

		// Execute with per-action timeout
		actionCtx, actionCancel := context.WithTimeout(ctx, actionTimeout)
		result, execErr := executor.Execute(actionCtx, actionCfg.Config, env)
		actionCancel()

		we.logger.LogActionResult(ctx, executionID, auto.TenantID, i, actionCfg.Type, result, execErr)

		if execErr != nil || !result.Success {
			slog.Warn("action execution failed",
				"automation_id", auto.ID,
				"execution_id", executionID,
				"step", i,
				"action_type", actionCfg.Type,
				"error", execErr,
			)

			if actionCfg.OnError == models.OnErrorAbort {
				break
			}
			continue
		}

		// Chain output: merge result.Output into env
		if result.Output != nil {
			for k, v := range result.Output {
				env["prev_"+k] = v
				env[fmt.Sprintf("step_%d_%s", i, k)] = v
			}
		}
	}

	// Log completion
	duration := time.Since(startTime)
	we.logger.LogComplete(ctx, executionID, auto.TenantID, duration)

	// Update last triggered timestamp
	now := time.Now()
	if err := we.workflowRepo.UpdateLastTriggered(ctx, auto.ID, now); err != nil {
		slog.Error("failed to update last_triggered_at",
			"automation_id", auto.ID,
			"error", err,
		)
	}

	// Track execution for circuit breaker
	we.trackExecution(auto.ID)

	slog.Info("automation execution completed",
		"automation_id", auto.ID,
		"execution_id", executionID,
		"duration_ms", duration.Milliseconds(),
		"action_count", len(actions),
	)

	return nil
}

// isCircuitBreakerOpen checks if the automation has exceeded the execution threshold.
func (we *WorkflowEngine) isCircuitBreakerOpen(automationID uuid.UUID) bool {
	we.mu.Lock()
	defer we.mu.Unlock()

	state, ok := we.executionCounts[automationID]
	if !ok {
		return false
	}

	// Reset window if expired
	if time.Now().After(state.windowEnd) {
		delete(we.executionCounts, automationID)
		return false
	}

	return state.count >= circuitBreakerThreshold
}

// trackExecution increments the execution counter for circuit breaker tracking.
func (we *WorkflowEngine) trackExecution(automationID uuid.UUID) {
	we.mu.Lock()
	defer we.mu.Unlock()

	state, ok := we.executionCounts[automationID]
	if !ok || time.Now().After(state.windowEnd) {
		we.executionCounts[automationID] = &circuitState{
			count:     1,
			windowEnd: time.Now().Add(1 * time.Hour),
		}
		return
	}

	state.count++
}

// tenantSemaphore returns the buffered channel used as a per-tenant semaphore,
// creating it lazily on first access. mu must NOT be held by the caller.
func (we *WorkflowEngine) tenantSemaphore(tenantID uuid.UUID) chan struct{} {
	we.mu.Lock()
	defer we.mu.Unlock()
	sem, ok := we.tenantSemaphores[tenantID]
	if !ok {
		sem = make(chan struct{}, maxConcurrentExecutionsPerTenant)
		we.tenantSemaphores[tenantID] = sem
	}
	return sem
}

// buildEnvFromPayload converts an EventPayload into a flat environment map
// suitable for template resolution and condition evaluation.
func buildEnvFromPayload(evt models.EventPayload) map[string]any {
	env := map[string]any{
		"event_type":  evt.Type,
		"module_id":   evt.ModuleID,
		"actor_id":    evt.ActorID,
		"resource_id": evt.ResourceID,
		"timestamp":   evt.Timestamp.Format(time.RFC3339),
	}

	// Parse additional fields from the event payload JSON
	if evt.Payload != nil {
		var extra map[string]any
		if err := json.Unmarshal(evt.Payload, &extra); err == nil {
			for k, v := range extra {
				env[k] = v
			}
		}
	}

	return env
}
