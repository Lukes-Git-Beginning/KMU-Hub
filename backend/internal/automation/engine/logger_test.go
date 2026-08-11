package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/automation/action"
	"github.com/kmuhub/kmuhub/internal/automation/workflow"
	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// fakeExecRepo: an honest in-memory workflow.ExecutionRepository.
//
// Unlike mockExecRepo (engine_test.go), which returns a fresh, empty
// AutomationExecution from every GetExecution call regardless of what was
// previously stored, fakeExecRepo actually keeps state keyed by execution ID
// so tests can assert on accumulated mutations (status transitions, appended
// steps) across multiple Logger calls, and can force GetExecution/
// UpdateExecution errors to exercise ExecutionLogger.updateExecution's error
// paths.
// ============================================================================

type fakeExecRepo struct {
	mu    sync.Mutex
	execs map[uuid.UUID]*models.AutomationExecution

	getErr    error
	updateErr error

	createCalls int
	getCalls    int
	updateCalls int
}

func newFakeExecRepo() *fakeExecRepo {
	return &fakeExecRepo{execs: make(map[uuid.UUID]*models.AutomationExecution)}
}

func (f *fakeExecRepo) CreateExecution(_ context.Context, exec *models.AutomationExecution) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createCalls++
	cp := *exec
	f.execs[exec.ID] = &cp
	return nil
}

func (f *fakeExecRepo) UpdateExecution(_ context.Context, exec *models.AutomationExecution) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updateCalls++
	if f.updateErr != nil {
		return f.updateErr
	}
	cp := *exec
	f.execs[exec.ID] = &cp
	return nil
}

func (f *fakeExecRepo) GetExecution(_ context.Context, id, _ uuid.UUID) (*models.AutomationExecution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getCalls++
	if f.getErr != nil {
		return nil, f.getErr
	}
	exec, ok := f.execs[id]
	if !ok {
		return nil, fmt.Errorf("execution %s not found", id)
	}
	cp := *exec
	return &cp, nil
}

func (f *fakeExecRepo) ListExecutions(_ context.Context, _ workflow.ExecutionFilter) ([]*models.AutomationExecution, int, error) {
	return nil, 0, nil
}

func (f *fakeExecRepo) CleanupOldExecutions(_ context.Context, _ time.Duration) (int64, error) {
	return 0, nil
}

// get returns a defensive copy of the stored execution, or nil if absent.
func (f *fakeExecRepo) get(id uuid.UUID) *models.AutomationExecution {
	f.mu.Lock()
	defer f.mu.Unlock()
	exec, ok := f.execs[id]
	if !ok {
		return nil
	}
	cp := *exec
	return &cp
}

// allExecs returns defensive copies of every stored execution. Intended for
// engine-level tests that never see the executionID Execute generates
// internally, but know exactly one execution was logged.
func (f *fakeExecRepo) allExecs() []*models.AutomationExecution {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*models.AutomationExecution, 0, len(f.execs))
	for _, exec := range f.execs {
		cp := *exec
		out = append(out, &cp)
	}
	return out
}

func (f *fakeExecRepo) counts() (create, get, update int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.createCalls, f.getCalls, f.updateCalls
}

// ============================================================================
// ExecutionLogger tests
// ============================================================================

func TestExecutionLogger_LogStart_CreatesRunningExecution(t *testing.T) {
	t.Parallel()
	repo := newFakeExecRepo()
	logger := NewExecutionLogger(repo)

	execID := uuid.New()
	tenantID := uuid.New()
	automationID := uuid.New()
	chainID := uuid.New()
	evt := models.EventPayload{Type: "crm.deal.stage_changed", ModuleID: "crm", Timestamp: time.Now()}

	logger.LogStart(context.Background(), execID, tenantID, automationID, chainID, evt)

	stored := repo.get(execID)
	require.NotNil(t, stored)
	assert.Equal(t, models.ExecutionStatusRunning, stored.Status)
	assert.Equal(t, tenantID, stored.TenantID)
	assert.Equal(t, automationID, stored.AutomationID)
	assert.Equal(t, chainID, stored.ChainID)
}

func TestExecutionLogger_LogConditionSkipped_SetsSkippedStatus(t *testing.T) {
	t.Parallel()
	repo := newFakeExecRepo()
	logger := NewExecutionLogger(repo)

	execID := uuid.New()
	tenantID := uuid.New()
	logger.LogStart(context.Background(), execID, tenantID, uuid.New(), uuid.New(), models.EventPayload{Timestamp: time.Now()})

	logger.LogConditionSkipped(context.Background(), execID, tenantID)

	stored := repo.get(execID)
	require.NotNil(t, stored)
	assert.Equal(t, models.ExecutionStatusSkipped, stored.Status)
	assert.False(t, stored.ConditionResult)
	require.NotNil(t, stored.CompletedAt)
	require.NotNil(t, stored.DurationMs)
}

func TestExecutionLogger_LogConditionError_SetsFailedStatusWithMessage(t *testing.T) {
	t.Parallel()
	repo := newFakeExecRepo()
	logger := NewExecutionLogger(repo)

	execID := uuid.New()
	tenantID := uuid.New()
	logger.LogStart(context.Background(), execID, tenantID, uuid.New(), uuid.New(), models.EventPayload{Timestamp: time.Now()})

	logger.LogConditionError(context.Background(), execID, tenantID, fmt.Errorf("bad condition"))

	stored := repo.get(execID)
	require.NotNil(t, stored)
	assert.Equal(t, models.ExecutionStatusFailed, stored.Status)
	require.NotNil(t, stored.ErrorMessage)
	assert.Equal(t, "bad condition", *stored.ErrorMessage)
}

func TestExecutionLogger_LogActionResult_SuccessAppendsStepWithoutFailing(t *testing.T) {
	t.Parallel()
	repo := newFakeExecRepo()
	logger := NewExecutionLogger(repo)

	execID := uuid.New()
	tenantID := uuid.New()
	logger.LogStart(context.Background(), execID, tenantID, uuid.New(), uuid.New(), models.EventPayload{Timestamp: time.Now()})

	logger.LogActionResult(context.Background(), execID, tenantID, 0, "crm.update_deal_field",
		action.ActionResult{Success: true, Output: map[string]any{"foo": "bar"}}, nil)

	stored := repo.get(execID)
	require.NotNil(t, stored)
	assert.Equal(t, models.ExecutionStatusRunning, stored.Status, "a successful step must not flip the status away from running")

	var steps []models.ExecutionStep
	require.NoError(t, json.Unmarshal(stored.Steps, &steps))
	require.Len(t, steps, 1)
	assert.Equal(t, "crm.update_deal_field", steps[0].ActionType)
	assert.Nil(t, steps[0].Error)
}

func TestExecutionLogger_LogActionResult_FailureSetsFailedStatus(t *testing.T) {
	t.Parallel()
	repo := newFakeExecRepo()
	logger := NewExecutionLogger(repo)

	execID := uuid.New()
	tenantID := uuid.New()
	logger.LogStart(context.Background(), execID, tenantID, uuid.New(), uuid.New(), models.EventPayload{Timestamp: time.Now()})

	logger.LogActionResult(context.Background(), execID, tenantID, 0, "crm.update_deal_field",
		action.ActionResult{Success: false, Error: "boom"}, fmt.Errorf("execution exploded"))

	stored := repo.get(execID)
	require.NotNil(t, stored)
	assert.Equal(t, models.ExecutionStatusFailed, stored.Status)
	require.NotNil(t, stored.ErrorMessage)
	assert.Equal(t, "execution exploded", *stored.ErrorMessage)
}

func TestExecutionLogger_LogActionResult_ResultFailureWithoutExecErr_StillFails(t *testing.T) {
	t.Parallel()
	repo := newFakeExecRepo()
	logger := NewExecutionLogger(repo)

	execID := uuid.New()
	tenantID := uuid.New()
	logger.LogStart(context.Background(), execID, tenantID, uuid.New(), uuid.New(), models.EventPayload{Timestamp: time.Now()})

	// execErr is nil, but result.Success is false: status must still flip to
	// failed, but ErrorMessage is only ever set from execErr, not result.Error.
	logger.LogActionResult(context.Background(), execID, tenantID, 0, "crm.update_deal_field",
		action.ActionResult{Success: false, Error: "soft failure"}, nil)

	stored := repo.get(execID)
	require.NotNil(t, stored)
	assert.Equal(t, models.ExecutionStatusFailed, stored.Status)
	assert.Nil(t, stored.ErrorMessage, "ErrorMessage is only populated from execErr, not result.Error")
}

func TestExecutionLogger_LogActionResult_AccumulatesStepsAcrossCalls(t *testing.T) {
	t.Parallel()
	repo := newFakeExecRepo()
	logger := NewExecutionLogger(repo)

	execID := uuid.New()
	tenantID := uuid.New()
	logger.LogStart(context.Background(), execID, tenantID, uuid.New(), uuid.New(), models.EventPayload{Timestamp: time.Now()})

	logger.LogActionResult(context.Background(), execID, tenantID, 0, "step_one", action.ActionResult{Success: true}, nil)
	logger.LogActionResult(context.Background(), execID, tenantID, 1, "step_two", action.ActionResult{Success: true}, nil)
	logger.LogActionResult(context.Background(), execID, tenantID, 2, "step_three", action.ActionResult{Success: true}, nil)

	stored := repo.get(execID)
	require.NotNil(t, stored)

	var steps []models.ExecutionStep
	require.NoError(t, json.Unmarshal(stored.Steps, &steps))
	require.Len(t, steps, 3, "each LogActionResult call must append, not overwrite, the steps array")
	assert.Equal(t, "step_one", steps[0].ActionType)
	assert.Equal(t, "step_two", steps[1].ActionType)
	assert.Equal(t, "step_three", steps[2].ActionType)
}

func TestExecutionLogger_LogComplete_MarksCompletedWhenStillRunning(t *testing.T) {
	t.Parallel()
	repo := newFakeExecRepo()
	logger := NewExecutionLogger(repo)

	execID := uuid.New()
	tenantID := uuid.New()
	logger.LogStart(context.Background(), execID, tenantID, uuid.New(), uuid.New(), models.EventPayload{Timestamp: time.Now()})

	logger.LogComplete(context.Background(), execID, tenantID, 5*time.Millisecond)

	stored := repo.get(execID)
	require.NotNil(t, stored)
	assert.Equal(t, models.ExecutionStatusCompleted, stored.Status)
	require.NotNil(t, stored.CompletedAt)
	require.NotNil(t, stored.DurationMs)
}

func TestExecutionLogger_LogComplete_PreservesNonRunningStatus(t *testing.T) {
	t.Parallel()
	repo := newFakeExecRepo()
	logger := NewExecutionLogger(repo)

	execID := uuid.New()
	tenantID := uuid.New()
	logger.LogStart(context.Background(), execID, tenantID, uuid.New(), uuid.New(), models.EventPayload{Timestamp: time.Now()})
	logger.LogConditionError(context.Background(), execID, tenantID, fmt.Errorf("already failed"))

	logger.LogComplete(context.Background(), execID, tenantID, 5*time.Millisecond)

	stored := repo.get(execID)
	require.NotNil(t, stored)
	assert.Equal(t, models.ExecutionStatusFailed, stored.Status, "LogComplete must not overwrite a status other than running")
}

func TestExecutionLogger_LogStepLimitReached_SetsAbortedStatus(t *testing.T) {
	t.Parallel()
	repo := newFakeExecRepo()
	logger := NewExecutionLogger(repo)

	execID := uuid.New()
	tenantID := uuid.New()
	logger.LogStart(context.Background(), execID, tenantID, uuid.New(), uuid.New(), models.EventPayload{Timestamp: time.Now()})

	logger.LogStepLimitReached(context.Background(), execID, tenantID, 10)

	stored := repo.get(execID)
	require.NotNil(t, stored)
	assert.Equal(t, models.ExecutionStatusAborted, stored.Status)
	require.NotNil(t, stored.ErrorMessage)
}

func TestExecutionLogger_UpdateExecution_GetErrorSkipsUpdate(t *testing.T) {
	t.Parallel()
	repo := newFakeExecRepo()
	repo.getErr = fmt.Errorf("db unavailable")
	logger := NewExecutionLogger(repo)

	execID := uuid.New()
	tenantID := uuid.New()

	require.NotPanics(t, func() {
		logger.LogConditionSkipped(context.Background(), execID, tenantID)
	})

	_, _, updateCalls := repo.counts()
	assert.Equal(t, 0, updateCalls, "UpdateExecution must not be called when GetExecution fails")
}

func TestExecutionLogger_UpdateExecution_UpdateErrorDoesNotPanic(t *testing.T) {
	t.Parallel()
	repo := newFakeExecRepo()
	logger := NewExecutionLogger(repo)

	execID := uuid.New()
	tenantID := uuid.New()
	logger.LogStart(context.Background(), execID, tenantID, uuid.New(), uuid.New(), models.EventPayload{Timestamp: time.Now()})

	repo.updateErr = fmt.Errorf("write failed")

	require.NotPanics(t, func() {
		logger.LogConditionSkipped(context.Background(), execID, tenantID)
	})
}
