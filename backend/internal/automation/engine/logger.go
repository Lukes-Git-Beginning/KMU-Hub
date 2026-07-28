package engine

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/automation/action"
	"github.com/kmuhub/kmuhub/internal/automation/workflow"
	"github.com/kmuhub/kmuhub/internal/models"
)

// ExecutionLogger records automation execution events to the database.
// All methods are fire-and-forget: they log errors internally and never
// return errors to the caller, ensuring the engine is not blocked by logging failures.
type ExecutionLogger struct {
	execRepo workflow.ExecutionRepository
}

// NewExecutionLogger creates a new execution logger.
func NewExecutionLogger(execRepo workflow.ExecutionRepository) *ExecutionLogger {
	return &ExecutionLogger{execRepo: execRepo}
}

// LogStart creates a new execution record with status "running".
func (l *ExecutionLogger) LogStart(ctx context.Context, execID, tenantID, automationID, chainID uuid.UUID, evt models.EventPayload) {
	evtJSON, err := json.Marshal(evt)
	if err != nil {
		slog.Error("failed to marshal event for execution log",
			"execution_id", execID,
			"error", err,
		)
		evtJSON = []byte("{}")
	}

	exec := &models.AutomationExecution{
		ID:           execID,
		TenantID:     tenantID,
		AutomationID: automationID,
		ChainID:      chainID,
		TriggerEvent: evtJSON,
		Status:       models.ExecutionStatusRunning,
		Steps:        json.RawMessage("[]"),
		StartedAt:    time.Now(),
	}

	if err := l.execRepo.CreateExecution(ctx, exec); err != nil {
		slog.Error("failed to create execution log",
			"execution_id", execID,
			"automation_id", automationID,
			"error", err,
		)
	}
}

// LogConditionSkipped updates the execution status to "skipped".
func (l *ExecutionLogger) LogConditionSkipped(ctx context.Context, execID, tenantID uuid.UUID) {
	l.updateExecution(ctx, execID, tenantID, func(exec *models.AutomationExecution) {
		exec.Status = models.ExecutionStatusSkipped
		exec.ConditionResult = false
		now := time.Now()
		exec.CompletedAt = &now
		durationMs := int(time.Since(exec.StartedAt).Milliseconds())
		exec.DurationMs = &durationMs
	})
}

// LogConditionError updates the execution status to "failed" with the error.
func (l *ExecutionLogger) LogConditionError(ctx context.Context, execID, tenantID uuid.UUID, condErr error) {
	l.updateExecution(ctx, execID, tenantID, func(exec *models.AutomationExecution) {
		exec.Status = models.ExecutionStatusFailed
		exec.ConditionResult = false
		errMsg := condErr.Error()
		exec.ErrorMessage = &errMsg
		now := time.Now()
		exec.CompletedAt = &now
		durationMs := int(time.Since(exec.StartedAt).Milliseconds())
		exec.DurationMs = &durationMs
	})
}

// LogActionResult appends an action step to the execution's steps array.
func (l *ExecutionLogger) LogActionResult(ctx context.Context, execID, tenantID uuid.UUID, stepIndex int, actionType string, result action.ActionResult, execErr error) {
	step := models.ExecutionStep{
		ActionType: actionType,
	}

	if result.Output != nil {
		outputJSON, err := json.Marshal(result.Output)
		if err == nil {
			step.Output = outputJSON
		}
	}

	if execErr != nil {
		errMsg := execErr.Error()
		step.Error = &errMsg
	} else if result.Error != "" {
		step.Error = &result.Error
	}

	l.updateExecution(ctx, execID, tenantID, func(exec *models.AutomationExecution) {
		exec.ConditionResult = true

		var steps []models.ExecutionStep
		if exec.Steps != nil {
			_ = json.Unmarshal(exec.Steps, &steps)
		}
		steps = append(steps, step)

		stepsJSON, err := json.Marshal(steps)
		if err == nil {
			exec.Steps = stepsJSON
		}

		if execErr != nil || !result.Success {
			exec.Status = models.ExecutionStatusFailed
			if execErr != nil {
				errMsg := execErr.Error()
				exec.ErrorMessage = &errMsg
			}
		}
	})
}

// LogComplete updates the execution status to "completed" with the duration.
func (l *ExecutionLogger) LogComplete(ctx context.Context, execID, tenantID uuid.UUID, duration time.Duration) {
	l.updateExecution(ctx, execID, tenantID, func(exec *models.AutomationExecution) {
		// Only mark completed if not already failed
		if exec.Status == models.ExecutionStatusRunning {
			exec.Status = models.ExecutionStatusCompleted
		}
		now := time.Now()
		exec.CompletedAt = &now
		durationMs := int(duration.Milliseconds())
		exec.DurationMs = &durationMs
	})
}

// LogStepLimitReached updates the execution status to "aborted" when the step limit is exceeded.
func (l *ExecutionLogger) LogStepLimitReached(ctx context.Context, execID, tenantID uuid.UUID, limit int) {
	l.updateExecution(ctx, execID, tenantID, func(exec *models.AutomationExecution) {
		exec.Status = models.ExecutionStatusAborted
		errMsg := "step limit reached: " + string(rune(limit+'0'))
		exec.ErrorMessage = &errMsg
		now := time.Now()
		exec.CompletedAt = &now
		durationMs := int(time.Since(exec.StartedAt).Milliseconds())
		exec.DurationMs = &durationMs
	})
}

// updateExecution loads an execution, applies the mutation, and persists it.
func (l *ExecutionLogger) updateExecution(ctx context.Context, execID, tenantID uuid.UUID, mutate func(*models.AutomationExecution)) {
	exec, err := l.execRepo.GetExecution(ctx, execID, tenantID)
	if err != nil {
		slog.Error("failed to load execution for update",
			"execution_id", execID,
			"error", err,
		)
		return
	}

	mutate(exec)

	if err := l.execRepo.UpdateExecution(ctx, exec); err != nil {
		slog.Error("failed to update execution log",
			"execution_id", execID,
			"error", err,
		)
	}
}
