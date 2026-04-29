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
	// maxConcurrentExecutions limits parallel workflow executions.
	maxConcurrentExecutions = 20

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

	semaphore chan struct{}

	// Circuit breaker: tracks executions per automation per hour
	mu              sync.Mutex
	executionCounts map[uuid.UUID]*circuitState
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
		condEvaluator:   evaluator,
		actionRegistry:  registry,
		logger:          logger,
		workflowRepo:    repo,
		execRepo:        execRepo,
		semaphore:       make(chan struct{}, maxConcurrentExecutions),
		executionCounts: make(map[uuid.UUID]*circuitState),
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

	// Acquire semaphore slot (non-blocking: return if full)
	select {
	case we.semaphore <- struct{}{}:
		defer func() { <-we.semaphore }()
	default:
		slog.Warn("execution semaphore full, skipping",
			"automation_id", auto.ID,
			"max_concurrent", maxConcurrentExecutions,
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
	we.logger.LogStart(ctx, executionID, auto.ID, chainID, evt)

	// Parse conditions
	var condConfig models.ConditionConfig
	if len(auto.Conditions) > 0 {
		if err := json.Unmarshal(auto.Conditions, &condConfig); err != nil {
			we.logger.LogConditionError(ctx, executionID, err)
			return fmt.Errorf("unmarshal conditions: %w", err)
		}
	}

	// Evaluate conditions
	condResult, err := we.condEvaluator.Evaluate(ctx, condConfig, env)
	if err != nil {
		we.logger.LogConditionError(ctx, executionID, err)
		return fmt.Errorf("evaluate conditions: %w", err)
	}

	if !condResult {
		we.logger.LogConditionSkipped(ctx, executionID)
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
			we.logger.LogConditionError(ctx, executionID, err)
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
			we.logger.LogStepLimitReached(ctx, executionID, maxSteps)
			return workflow.ErrMaxStepsExceeded
		}

		executor, ok := we.actionRegistry.Get(actionCfg.Type)
		if !ok {
			errMsg := fmt.Sprintf("unknown action type: %s", actionCfg.Type)
			we.logger.LogActionResult(ctx, executionID, i, actionCfg.Type,
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

		we.logger.LogActionResult(ctx, executionID, i, actionCfg.Type, result, execErr)

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
	we.logger.LogComplete(ctx, executionID, duration)

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
