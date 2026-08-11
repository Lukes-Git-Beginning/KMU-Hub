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
	"github.com/kmuhub/kmuhub/internal/automation/condition"
	"github.com/kmuhub/kmuhub/internal/automation/workflow"
	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// Test-only ActionExecutor implementations
// ============================================================================

// countingExecutor records how many times it ran; used to verify a step
// either ran exactly once or was skipped entirely.
type countingExecutor struct {
	typ string
	mu  sync.Mutex
	n   int
}

func (c *countingExecutor) Type() string { return c.typ }
func (c *countingExecutor) Execute(_ context.Context, _ json.RawMessage, _ map[string]any) (action.ActionResult, error) {
	c.mu.Lock()
	c.n++
	c.mu.Unlock()
	return action.ActionResult{Success: true}, nil
}
func (c *countingExecutor) callCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

// failingExecutor returns a caller-configured result/error, used to exercise
// both the on_error=abort and on_error=continue branches of the action loop.
type failingExecutor struct {
	typ    string
	result action.ActionResult
	err    error
}

func (f *failingExecutor) Type() string { return f.typ }
func (f *failingExecutor) Execute(_ context.Context, _ json.RawMessage, _ map[string]any) (action.ActionResult, error) {
	return f.result, f.err
}

// outputExecutor always succeeds and returns a fixed Output map, used to
// verify Execute's output-chaining into env.
type outputExecutor struct{ typ string }

func (o *outputExecutor) Type() string { return o.typ }
func (o *outputExecutor) Execute(_ context.Context, _ json.RawMessage, _ map[string]any) (action.ActionResult, error) {
	return action.ActionResult{Success: true, Output: map[string]any{"foo": "bar"}}, nil
}

// chainingExecutor asserts that the env it receives already contains the
// prior step's chained output, reporting the verdict on resultCh since
// assertions can't cross into the engine's internal loop directly.
type chainingExecutor struct {
	typ      string
	resultCh chan bool
}

func (c *chainingExecutor) Type() string { return c.typ }
func (c *chainingExecutor) Execute(_ context.Context, _ json.RawMessage, env map[string]any) (action.ActionResult, error) {
	prevFoo, prevOk := env["prev_foo"]
	stepFoo, stepOk := env["step_0_foo"]
	ok := prevOk && stepOk && prevFoo == "bar" && stepFoo == "bar"
	c.resultCh <- ok
	return action.ActionResult{Success: true}, nil
}

// ============================================================================
// recordingWorkflowRepo: mockWorkflowRepo plus a SetActive call log, used for
// the circuit-breaker test to verify the automation gets auto-disabled.
// ============================================================================

type setActiveCall struct {
	id       uuid.UUID
	tenantID uuid.UUID
	active   bool
}

type recordingWorkflowRepo struct {
	mockWorkflowRepo
	mu    sync.Mutex
	calls []setActiveCall
}

func (r *recordingWorkflowRepo) SetActive(_ context.Context, id, tenantID uuid.UUID, active bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, setActiveCall{id: id, tenantID: tenantID, active: active})
	return nil
}

func (r *recordingWorkflowRepo) setActiveCalls() []setActiveCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]setActiveCall(nil), r.calls...)
}

// ============================================================================
// helper
// ============================================================================

// newEngineWith builds a WorkflowEngine with a real condition.Evaluator and
// an action.ActionRegistry populated with the given executors, wired to the
// provided repos so tests can inspect logged execution state and repo calls.
func newEngineWith(wfRepo workflow.Repository, execRepo workflow.ExecutionRepository, executors ...action.ActionExecutor) *WorkflowEngine {
	evaluator := condition.NewEvaluator()
	registry := action.NewActionRegistry()
	for _, exec := range executors {
		registry.Register(exec, &action.ActionDefinition{Type: exec.Type()})
	}
	logger := NewExecutionLogger(execRepo)
	return NewWorkflowEngine(evaluator, registry, logger, wfRepo, execRepo)
}

// singleExec returns the one execution logged during a test, failing it if
// there isn't exactly one. Execute() generates its executionID internally, so
// this is how engine-level (as opposed to logger-level) tests must observe
// logged state.
func singleExec(t *testing.T, repo *fakeExecRepo) *models.AutomationExecution {
	t.Helper()
	execs := repo.allExecs()
	require.Len(t, execs, 1, "expected exactly one logged execution")
	return execs[0]
}

// ============================================================================
// Circuit breaker
// ============================================================================

func TestExecute_CircuitBreakerOpen_DisablesAutomationAndReturnsError(t *testing.T) {
	wfRepo := &recordingWorkflowRepo{}
	execRepo := newFakeExecRepo()
	eng := newEngineWith(wfRepo, execRepo)

	auto := makeAutomation(uuid.New())
	eng.executionCounts[auto.ID] = &circuitState{
		count:     circuitBreakerThreshold,
		windowEnd: time.Now().Add(time.Hour),
	}

	err := eng.Execute(context.Background(), auto, makeEvent())
	require.ErrorIs(t, err, workflow.ErrCircuitBreakerOpen)

	calls := wfRepo.setActiveCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, auto.ID, calls[0].id)
	assert.Equal(t, auto.TenantID, calls[0].tenantID)
	assert.False(t, calls[0].active)
}

// ============================================================================
// Condition parsing / evaluation errors
// ============================================================================

func TestExecute_InvalidConditionsJSON_LogsConditionErrorAndFails(t *testing.T) {
	execRepo := newFakeExecRepo()
	eng := newEngineWith(&mockWorkflowRepo{}, execRepo)

	auto := makeAutomation(uuid.New())
	auto.Conditions = []byte("{not json")

	err := eng.Execute(context.Background(), auto, makeEvent())
	require.Error(t, err)

	exec := singleExec(t, execRepo)
	assert.Equal(t, models.ExecutionStatusFailed, exec.Status)
}

func TestExecute_UnknownConditionMode_LogsConditionErrorAndFails(t *testing.T) {
	execRepo := newFakeExecRepo()
	eng := newEngineWith(&mockWorkflowRepo{}, execRepo)

	auto := makeAutomation(uuid.New())
	auto.Conditions = []byte(`{"mode":"bogus"}`)

	err := eng.Execute(context.Background(), auto, makeEvent())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown condition mode")

	exec := singleExec(t, execRepo)
	assert.Equal(t, models.ExecutionStatusFailed, exec.Status)
}

func TestExecute_ConditionNotMet_LogsSkippedAndRunsNoAction(t *testing.T) {
	execRepo := newFakeExecRepo()

	gate := make(chan struct{})
	close(gate)
	startedCh := make(chan struct{}, 1)
	blocker := &blockingExecutor{started: startedCh, gate: gate}
	eng := newEngineWith(&mockWorkflowRepo{}, execRepo, blocker)

	auto := makeAutomation(uuid.New())
	auto.Conditions = []byte(`{"mode":"simple","simple":{"field":"nonexistent_field","operator":"exists"}}`)

	err := eng.Execute(context.Background(), auto, makeEvent())
	require.NoError(t, err)

	select {
	case <-startedCh:
		t.Fatal("action executed despite condition evaluating to false")
	default:
	}

	exec := singleExec(t, execRepo)
	assert.Equal(t, models.ExecutionStatusSkipped, exec.Status)
	assert.False(t, exec.ConditionResult)
}

// ============================================================================
// Action loop: unknown action type
// ============================================================================

func TestExecute_UnknownActionType_AbortStopsBeforeNextStep(t *testing.T) {
	execRepo := newFakeExecRepo()
	next := &countingExecutor{typ: "known_action"}
	eng := newEngineWith(&mockWorkflowRepo{}, execRepo, next)

	auto := makeAutomation(uuid.New())
	auto.Actions = []byte(`[
		{"type":"unknown_action","on_error":"abort","config":{}},
		{"type":"known_action","on_error":"continue","config":{}}
	]`)

	err := eng.Execute(context.Background(), auto, makeEvent())
	require.NoError(t, err, "an aborted action loop is not itself an Execute error")
	assert.Equal(t, 0, next.callCount(), "on_error=abort must stop the loop before the next step")
}

func TestExecute_UnknownActionType_ContinueRunsNextStep(t *testing.T) {
	execRepo := newFakeExecRepo()
	next := &countingExecutor{typ: "known_action"}
	eng := newEngineWith(&mockWorkflowRepo{}, execRepo, next)

	auto := makeAutomation(uuid.New())
	auto.Actions = []byte(`[
		{"type":"unknown_action","on_error":"continue","config":{}},
		{"type":"known_action","on_error":"continue","config":{}}
	]`)

	err := eng.Execute(context.Background(), auto, makeEvent())
	require.NoError(t, err)
	assert.Equal(t, 1, next.callCount(), "on_error=continue must proceed to the next step")
}

// ============================================================================
// Action loop: failing action (execErr and/or result.Success=false)
// ============================================================================

func TestExecute_FailingAction_OnErrorBehavior(t *testing.T) {
	cases := []struct {
		name        string
		onError     string
		result      action.ActionResult
		execErr     error
		wantNextRan bool
	}{
		{
			name:        "result failure, abort",
			onError:     models.OnErrorAbort,
			result:      action.ActionResult{Success: false},
			wantNextRan: false,
		},
		{
			name:        "result failure, continue",
			onError:     models.OnErrorContinue,
			result:      action.ActionResult{Success: false},
			wantNextRan: true,
		},
		{
			name:        "exec error, abort",
			onError:     models.OnErrorAbort,
			result:      action.ActionResult{Success: true},
			execErr:     fmt.Errorf("boom"),
			wantNextRan: false,
		},
		{
			name:        "exec error, continue",
			onError:     models.OnErrorContinue,
			result:      action.ActionResult{Success: true},
			execErr:     fmt.Errorf("boom"),
			wantNextRan: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			execRepo := newFakeExecRepo()
			next := &countingExecutor{typ: "next_action"}
			failer := &failingExecutor{typ: "failing_action", result: tc.result, err: tc.execErr}
			eng := newEngineWith(&mockWorkflowRepo{}, execRepo, next, failer)

			auto := makeAutomation(uuid.New())
			auto.Actions = fmt.Appendf(nil, `[
				{"type":"failing_action","on_error":%q,"config":{}},
				{"type":"next_action","on_error":"continue","config":{}}
			]`, tc.onError)

			err := eng.Execute(context.Background(), auto, makeEvent())
			require.NoError(t, err)

			wantCount := 0
			if tc.wantNextRan {
				wantCount = 1
			}
			assert.Equal(t, wantCount, next.callCount())
		})
	}
}

// ============================================================================
// Output chaining
// ============================================================================

func TestExecute_OutputChaining_PopulatesEnvForNextStep(t *testing.T) {
	execRepo := newFakeExecRepo()
	resultCh := make(chan bool, 1)
	out := &outputExecutor{typ: "output_action"}
	chainer := &chainingExecutor{typ: "chain_action", resultCh: resultCh}
	eng := newEngineWith(&mockWorkflowRepo{}, execRepo, out, chainer)

	auto := makeAutomation(uuid.New())
	auto.Actions = []byte(`[
		{"type":"output_action","on_error":"continue","config":{}},
		{"type":"chain_action","on_error":"continue","config":{}}
	]`)

	err := eng.Execute(context.Background(), auto, makeEvent())
	require.NoError(t, err)

	select {
	case ok := <-resultCh:
		assert.True(t, ok, "second step's env must contain prev_foo and step_0_foo chained from the first step's output")
	case <-time.After(2 * time.Second):
		t.Fatal("chaining executor never ran")
	}
}

// ============================================================================
// MaxSteps
// ============================================================================

func TestExecute_MaxStepsReached_StopsAtLimitAndLogsAborted(t *testing.T) {
	execRepo := newFakeExecRepo()
	counter := &countingExecutor{typ: "capped_action"}
	eng := newEngineWith(&mockWorkflowRepo{}, execRepo, counter)

	auto := makeAutomation(uuid.New())
	auto.MaxSteps = 1
	auto.Actions = []byte(`[
		{"type":"capped_action","on_error":"continue","config":{}},
		{"type":"capped_action","on_error":"continue","config":{}}
	]`)

	err := eng.Execute(context.Background(), auto, makeEvent())
	require.ErrorIs(t, err, workflow.ErrMaxStepsExceeded)

	assert.Equal(t, 1, counter.callCount(), "step 0 must run before the MaxSteps=1 limit stops step 1")

	exec := singleExec(t, execRepo)
	assert.Equal(t, models.ExecutionStatusAborted, exec.Status)
	require.NotNil(t, exec.ErrorMessage)
}
