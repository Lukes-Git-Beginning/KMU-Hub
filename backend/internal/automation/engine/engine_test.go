package engine

import (
	"context"
	"encoding/json"
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
	"github.com/kmuhub/kmuhub/internal/notification/event"
)

// ============================================================================
// Minimal mock implementations for WorkflowEngine dependencies
// ============================================================================

// mockWorkflowRepo satisfies workflow.Repository with no-op stubs.
type mockWorkflowRepo struct{}

func (m *mockWorkflowRepo) Create(_ context.Context, _ *models.Automation) error  { return nil }
func (m *mockWorkflowRepo) Update(_ context.Context, _ *models.Automation) error  { return nil }
func (m *mockWorkflowRepo) Delete(_ context.Context, _, _ uuid.UUID) error        { return nil }
func (m *mockWorkflowRepo) GetByID(_ context.Context, _, _ uuid.UUID) (*models.Automation, error) {
	return nil, nil
}
func (m *mockWorkflowRepo) GetByIDUnscoped(_ context.Context, _ uuid.UUID) (*models.Automation, error) {
	return nil, nil
}
func (m *mockWorkflowRepo) List(_ context.Context, _ workflow.ListFilter) ([]*models.Automation, int, error) {
	return nil, 0, nil
}
func (m *mockWorkflowRepo) ListActiveByTriggerType(_ context.Context, _ string) ([]*models.Automation, error) {
	return nil, nil
}
func (m *mockWorkflowRepo) ListActiveTimeBased(_ context.Context) ([]*models.Automation, error) {
	return nil, nil
}
func (m *mockWorkflowRepo) SetActive(_ context.Context, _, _ uuid.UUID, _ bool) error {
	return nil
}
func (m *mockWorkflowRepo) UpdateLastTriggered(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}

// mockExecRepo satisfies workflow.ExecutionRepository with no-op stubs.
type mockExecRepo struct{}

func (m *mockExecRepo) CreateExecution(_ context.Context, _ *models.AutomationExecution) error {
	return nil
}
func (m *mockExecRepo) UpdateExecution(_ context.Context, _ *models.AutomationExecution) error {
	return nil
}
func (m *mockExecRepo) GetExecution(_ context.Context, _, _ uuid.UUID) (*models.AutomationExecution, error) {
	return &models.AutomationExecution{Steps: []byte("[]")}, nil
}
func (m *mockExecRepo) ListExecutions(_ context.Context, _ workflow.ExecutionFilter) ([]*models.AutomationExecution, int, error) {
	return nil, 0, nil
}
func (m *mockExecRepo) CleanupOldExecutions(_ context.Context, _ time.Duration) (int64, error) {
	return 0, nil
}

// ============================================================================
// blockingExecutor is an ActionExecutor that blocks until the provided gate
// channel is closed. Used to hold semaphore slots open for concurrency tests.
// ============================================================================

type blockingExecutor struct {
	// started is sent on once Execute has begun (before blocking on gate).
	started chan<- struct{}
	// gate is closed to unblock all in-flight Execute calls.
	gate <-chan struct{}
}

func (b *blockingExecutor) Type() string { return "test_block" }

func (b *blockingExecutor) Execute(_ context.Context, _ json.RawMessage, _ map[string]any) (action.ActionResult, error) {
	// Signal that we have entered Execute (slots are held at this point).
	if b.started != nil {
		select {
		case b.started <- struct{}{}:
		default:
		}
	}
	<-b.gate
	return action.ActionResult{Success: true}, nil
}

// ============================================================================
// helpers
// ============================================================================

// newTestEngine returns a WorkflowEngine wired with minimal mocks and the
// given action executor registered as "test_block".
func newTestEngine(executor action.ActionExecutor) *WorkflowEngine {
	evaluator := condition.NewEvaluator()
	registry := action.NewActionRegistry()
	if executor != nil {
		registry.Register(executor, &action.ActionDefinition{
			Type:        "test_block",
			Name:        "Block for test",
			Description: "Blocks until gate is closed",
		})
	}
	execRepo := &mockExecRepo{}
	logger := NewExecutionLogger(execRepo)

	return NewWorkflowEngine(evaluator, registry, logger, &mockWorkflowRepo{}, execRepo)
}

// makeAutomation builds a minimal Automation with one "test_block" action and
// the given tenantID.
func makeAutomation(tenantID uuid.UUID) models.Automation {
	actions := []byte(`[{"type":"test_block","on_error":"continue","config":{}}]`)
	return models.Automation{
		ID:       uuid.New(),
		TenantID: tenantID,
		Actions:  actions,
		MaxSteps: 10,
	}
}

// makeEvent returns a minimal EventPayload that bypasses loop prevention.
func makeEvent() models.EventPayload {
	return models.EventPayload{
		Type:      "contact.created",
		ModuleID:  "crm",
		Timestamp: time.Now(),
	}
}

// ============================================================================
// Loop-prevention ModuleID test
// ============================================================================

// TestExecute_SkipsAutomationModuleEvent verifies the intended loop guard:
// an event tagged as originating from the automation module itself is
// dropped before any action runs.
func TestExecute_SkipsAutomationModuleEvent(t *testing.T) {
	gate := make(chan struct{})
	close(gate) // executor would return immediately if ever entered
	startedCh := make(chan struct{}, 1)
	executor := &blockingExecutor{started: startedCh, gate: gate}
	eng := newTestEngine(executor)

	evt := models.EventPayload{Type: "x", ModuleID: event.ModuleAutomation, Timestamp: time.Now()}
	err := eng.Execute(context.Background(), makeAutomation(uuid.New()), evt)
	require.NoError(t, err)

	select {
	case <-startedCh:
		t.Fatal("action ran for a ModuleAutomation-tagged event; loop prevention did not skip it")
	default:
	}
}

// TestExecute_SchedulerOriginedEventIsNotSkipped is a regression test for the
// poller ModuleID collision: trigger.TimeTriggerPoller used to tag its
// synthetic time-based-trigger events with ModuleID "automation", which this
// exact loop-prevention check then silently discarded before condition
// evaluation -- biz.invoice.overdue and calendar.event.upcoming never
// actually executed. The poller now tags them "scheduler"; this asserts that
// value is not itself caught by the guard.
func TestExecute_SchedulerOriginedEventIsNotSkipped(t *testing.T) {
	gate := make(chan struct{})
	close(gate)
	startedCh := make(chan struct{}, 1)
	executor := &blockingExecutor{started: startedCh, gate: gate}
	eng := newTestEngine(executor)

	evt := models.EventPayload{Type: "x", ModuleID: "scheduler", Timestamp: time.Now()}
	err := eng.Execute(context.Background(), makeAutomation(uuid.New()), evt)
	require.NoError(t, err)

	select {
	case <-startedCh:
	case <-time.After(time.Second):
		t.Fatal("action did not run for a scheduler-originated event")
	}
}

// ============================================================================
// Per-tenant semaphore isolation test
// ============================================================================

// TestPerTenantSemaphore_TenantADoesNotStarveB verifies that when Tenant A
// has exhausted its per-tenant slot budget (maxConcurrentExecutionsPerTenant),
// Tenant B can still acquire a slot and execute successfully.
//
// Sequence:
//  1. Launch N goroutines for Tenant A — each blocks inside Execute (holds a slot).
//  2. Wait until all N are confirmed inside Execute (slots held).
//  3. Attempt one more Execute for Tenant A → must be dropped (nil error, no panic).
//  4. Execute for Tenant B → must succeed (nil error) despite A's saturation.
func TestPerTenantSemaphore_TenantADoesNotStarveB(t *testing.T) {
	const n = maxConcurrentExecutionsPerTenant

	// started receives one token for each Execute call that has entered the
	// blocking executor and is now holding a semaphore slot.
	startedCh := make(chan struct{}, n*2)
	gate := make(chan struct{})

	executor := &blockingExecutor{started: startedCh, gate: gate}
	eng := newTestEngine(executor)

	tenantA := uuid.New()
	tenantB := uuid.New()

	// Launch N goroutines for Tenant A.
	var wgA sync.WaitGroup
	for i := 0; i < n; i++ {
		wgA.Add(1)
		go func() {
			defer wgA.Done()
			_ = eng.Execute(context.Background(), makeAutomation(tenantA), makeEvent())
		}()
	}

	// Wait until all N have signalled that they entered Execute (slots confirmed held).
	deadline := time.After(2 * time.Second)
	for received := 0; received < n; {
		select {
		case <-startedCh:
			received++
		case <-deadline:
			close(gate) // unblock so goroutines can exit cleanly
			t.Fatalf("timeout: only %d/%d tenant-A goroutines entered Execute", received, n)
		}
	}

	// An additional Execute for Tenant A must be dropped (per-tenant limit reached).
	droppedResult := make(chan error, 1)
	go func() {
		droppedResult <- eng.Execute(context.Background(), makeAutomation(tenantA), makeEvent())
	}()
	select {
	case err := <-droppedResult:
		assert.NoError(t, err, "dropped execution must return nil (skip), not an error")
	case <-time.After(500 * time.Millisecond):
		close(gate)
		t.Fatal("timeout: extra Tenant A execution did not return promptly")
	}

	// Tenant B must execute successfully — its semaphore bucket is independent.
	// Close the gate first so the blocking executor returns immediately for B.
	close(gate)

	err := eng.Execute(context.Background(), makeAutomation(tenantB), makeEvent())
	assert.NoError(t, err, "Tenant B must execute successfully when Tenant A's bucket is saturated")

	wgA.Wait()
}

// ============================================================================
// Global semaphore cap test
// ============================================================================

// TestGlobalSemaphore_DropsWhenFull verifies that when the global semaphore is
// fully occupied, further executions are dropped (return nil) without panic.
func TestGlobalSemaphore_DropsWhenFull(t *testing.T) {
	gate := make(chan struct{})
	executor := &blockingExecutor{gate: gate}
	eng := newTestEngine(executor)

	// Pre-fill the global semaphore to simulate maxConcurrentExecutions in-flight.
	for i := 0; i < maxConcurrentExecutions; i++ {
		eng.semaphore <- struct{}{}
	}

	err := eng.Execute(context.Background(), makeAutomation(uuid.New()), makeEvent())
	assert.NoError(t, err, "execution when global semaphore is full must return nil (skip)")

	// Drain the manually filled slots and close gate.
	for i := 0; i < maxConcurrentExecutions; i++ {
		<-eng.semaphore
	}
	close(gate)
}

// ============================================================================
// Nil TenantID bucket test
// ============================================================================

// TestNilTenantID_UsesNilBucket confirms that an automation with uuid.Nil as
// tenant_id is placed into its own semaphore bucket and executes without panic.
func TestNilTenantID_UsesNilBucket(t *testing.T) {
	gate := make(chan struct{})
	close(gate) // non-blocking: executor returns immediately
	executor := &blockingExecutor{gate: gate}
	eng := newTestEngine(executor)

	err := eng.Execute(context.Background(), makeAutomation(uuid.Nil), makeEvent())
	assert.NoError(t, err)

	// Verify the nil-UUID bucket was created, not mixed with any tenant bucket.
	eng.mu.Lock()
	_, exists := eng.tenantSemaphores[uuid.Nil]
	eng.mu.Unlock()
	assert.True(t, exists, "nil tenant_id must create its own semaphore bucket")
}

// ============================================================================
// Slot release correctness test
// ============================================================================

// TestSemaphoreReleasedAfterExecution verifies that slots are released after
// successful execution, so that more than maxConcurrentExecutionsPerTenant
// sequential executions can all complete for the same tenant.
func TestSemaphoreReleasedAfterExecution(t *testing.T) {
	gate := make(chan struct{})
	close(gate) // unblocked immediately — fast executions
	executor := &blockingExecutor{gate: gate}
	eng := newTestEngine(executor)

	tenantID := uuid.New()

	// Run maxConcurrentExecutionsPerTenant+2 executions sequentially for the
	// same tenant. All should complete without slots leaking.
	for i := 0; i < maxConcurrentExecutionsPerTenant+2; i++ {
		err := eng.Execute(context.Background(), makeAutomation(tenantID), makeEvent())
		require.NoError(t, err, "sequential execution %d must not error", i)
	}

	// After all executions both semaphores must be empty (no leaked slots).
	assert.Equal(t, 0, len(eng.semaphore),
		"global semaphore must be fully released after sequential executions")

	eng.mu.Lock()
	tenantSem := eng.tenantSemaphores[tenantID]
	eng.mu.Unlock()
	require.NotNil(t, tenantSem, "tenant semaphore bucket must exist")
	assert.Equal(t, 0, len(tenantSem),
		"tenant semaphore must be fully released after sequential executions")
}
