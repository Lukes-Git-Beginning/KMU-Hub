package trigger

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/automation/workflow"
	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// Fakes
// ============================================================================

// fakeTimeBasedRepo implements workflow.Repository with only
// ListActiveTimeBased and ClaimTimeTrigger wired -- every other method is an
// unused no-op stub, since checkTimeTriggers never calls them.
type fakeTimeBasedRepo struct {
	mu sync.Mutex

	listResult []*models.Automation
	listErr    error
	listCalls  [][]string // captured triggerTypes argument per call

	claimResult map[uuid.UUID]bool // per-automation-id claim outcome; default true
	claimErr    error
	claimCalls  []uuid.UUID
}

func (f *fakeTimeBasedRepo) ListActiveTimeBased(_ context.Context, triggerTypes []string) ([]*models.Automation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listCalls = append(f.listCalls, triggerTypes)
	return f.listResult, f.listErr
}

func (f *fakeTimeBasedRepo) ClaimTimeTrigger(_ context.Context, id uuid.UUID, _ *time.Time, _ time.Time) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.claimCalls = append(f.claimCalls, id)
	if f.claimErr != nil {
		return false, f.claimErr
	}
	if f.claimResult == nil {
		return true, nil
	}
	claimed, ok := f.claimResult[id]
	if !ok {
		return true, nil
	}
	return claimed, nil
}

func (f *fakeTimeBasedRepo) claimedAutomations() []uuid.UUID {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]uuid.UUID(nil), f.claimCalls...)
}

func (f *fakeTimeBasedRepo) Create(context.Context, *models.Automation) error { return nil }
func (f *fakeTimeBasedRepo) Update(context.Context, *models.Automation) error { return nil }
func (f *fakeTimeBasedRepo) Delete(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}
func (f *fakeTimeBasedRepo) GetByID(context.Context, uuid.UUID, uuid.UUID) (*models.Automation, error) {
	return nil, nil
}
func (f *fakeTimeBasedRepo) GetByIDUnscoped(context.Context, uuid.UUID) (*models.Automation, error) {
	return nil, nil
}
func (f *fakeTimeBasedRepo) List(context.Context, workflow.ListFilter) ([]*models.Automation, int, error) {
	return nil, 0, nil
}
func (f *fakeTimeBasedRepo) ListActiveByTriggerType(context.Context, string) ([]*models.Automation, error) {
	return nil, nil
}
func (f *fakeTimeBasedRepo) SetActive(context.Context, uuid.UUID, uuid.UUID, bool) error {
	return nil
}
func (f *fakeTimeBasedRepo) UpdateLastTriggered(context.Context, uuid.UUID, time.Time) error {
	return nil
}

// fakeEngineExecutor records every Execute call on a buffered channel so
// tests can wait for the poller's async goroutine without sleeping.
type fakeEngineExecutor struct {
	executed chan uuid.UUID
}

func newFakeEngineExecutor() *fakeEngineExecutor {
	return &fakeEngineExecutor{executed: make(chan uuid.UUID, 16)}
}

func (f *fakeEngineExecutor) Execute(_ context.Context, auto models.Automation, _ models.EventPayload) error {
	f.executed <- auto.ID
	return nil
}

func waitExecutedIDs(t *testing.T, executor *fakeEngineExecutor, n int) []uuid.UUID {
	t.Helper()
	var got []uuid.UUID
	for i := range n {
		select {
		case id := <-executor.executed:
			got = append(got, id)
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for execution %d/%d", i+1, n)
		}
	}
	return got
}

func assertNoExecution(t *testing.T, executor *fakeEngineExecutor) {
	t.Helper()
	select {
	case id := <-executor.executed:
		t.Fatalf("expected no execution, got one for automation %s", id)
	case <-time.After(200 * time.Millisecond):
	}
}

// ============================================================================
// timeBasedTriggerTypes
// ============================================================================

func TestTimeBasedTriggerTypes_MatchesRegistryTimeBasedFlag(t *testing.T) {
	registry := NewTriggerRegistry()

	got := timeBasedTriggerTypes(registry)

	gotSet := make(map[string]bool, len(got))
	for _, ty := range got {
		gotSet[ty] = true
	}

	for _, def := range registry.All() {
		if def.TimeBased && !gotSet[def.Type] {
			t.Errorf("registry entry %q has TimeBased=true but is missing from timeBasedTriggerTypes", def.Type)
		}
		if !def.TimeBased && gotSet[def.Type] {
			t.Errorf("registry entry %q has TimeBased=false but was included in timeBasedTriggerTypes", def.Type)
		}
	}

	// Pin the known built-ins so a future registry change that silently drops
	// TimeBased=true from both is caught, not just a divergence between the
	// helper and the registry.
	if !gotSet["biz.invoice.overdue"] || !gotSet["calendar.event.upcoming"] {
		t.Fatalf("expected biz.invoice.overdue and calendar.event.upcoming, got %v", got)
	}
}

// ============================================================================
// checkTimeTriggers
// ============================================================================

func newTestAutomation(triggerType string) *models.Automation {
	return &models.Automation{
		ID:          uuid.New(),
		TriggerType: triggerType,
		IsActive:    true,
		Actions:     []byte(`[]`),
	}
}

func TestCheckTimeTriggers_ExecutesOnlyClaimedAutomations(t *testing.T) {
	claimedAuto := newTestAutomation("biz.invoice.overdue")
	lostAuto := newTestAutomation("calendar.event.upcoming")

	repo := &fakeTimeBasedRepo{
		listResult: []*models.Automation{claimedAuto, lostAuto},
		claimResult: map[uuid.UUID]bool{
			claimedAuto.ID: true,
			lostAuto.ID:    false, // lost the race to another instance/tick
		},
	}
	executor := newFakeEngineExecutor()

	poller := NewTimeTriggerPoller(repo, executor, nil, NewTriggerRegistry())
	poller.checkTimeTriggers(context.Background())

	executed := waitExecutedIDs(t, executor, 1)
	if executed[0] != claimedAuto.ID {
		t.Fatalf("expected claimed automation %s to execute, got %s", claimedAuto.ID, executed[0])
	}
	assertNoExecution(t, executor)

	claimed := repo.claimedAutomations()
	if len(claimed) != 2 {
		t.Fatalf("expected a claim attempt for both listed automations, got %d", len(claimed))
	}
}

func TestCheckTimeTriggers_PassesRegistryTimeBasedTypesToRepo(t *testing.T) {
	repo := &fakeTimeBasedRepo{}
	executor := newFakeEngineExecutor()
	registry := NewTriggerRegistry()

	poller := NewTimeTriggerPoller(repo, executor, nil, registry)
	poller.checkTimeTriggers(context.Background())

	if len(repo.listCalls) != 1 {
		t.Fatalf("expected exactly one ListActiveTimeBased call, got %d", len(repo.listCalls))
	}
	got := repo.listCalls[0]
	want := timeBasedTriggerTypes(registry)
	if len(got) != len(want) {
		t.Fatalf("ListActiveTimeBased called with %v, want %v", got, want)
	}
}

func TestCheckTimeTriggers_NoTimeBasedTriggers_SkipsRepoEntirely(t *testing.T) {
	// A registry with zero TimeBased=true entries must short-circuit before
	// ever calling the repository -- exercises the len(p.timeBasedTypes)==0
	// guard added alongside the ListActiveTimeBased signature change.
	empty := &TriggerRegistry{triggers: map[string]*TriggerDefinition{}}
	repo := &fakeTimeBasedRepo{}
	executor := newFakeEngineExecutor()

	poller := NewTimeTriggerPoller(repo, executor, nil, empty)
	poller.checkTimeTriggers(context.Background())

	if len(repo.listCalls) != 0 {
		t.Fatalf("expected ListActiveTimeBased not to be called, got %d calls", len(repo.listCalls))
	}
	assertNoExecution(t, executor)
}

func TestCheckTimeTriggers_ClaimError_SkipsExecutionWithoutFailingOthers(t *testing.T) {
	failsClaim := newTestAutomation("biz.invoice.overdue")
	succeedsClaim := newTestAutomation("calendar.event.upcoming")

	repo := &fakeTimeBasedRepo{
		listResult: []*models.Automation{failsClaim, succeedsClaim},
		claimResult: map[uuid.UUID]bool{
			succeedsClaim.ID: true,
		},
	}
	// claimErr applies to every ClaimTimeTrigger call in this fake, so verify
	// the all-error path separately: no automation should execute, and the
	// poller must not panic walking the rest of the list.
	repo.claimErr = context.Canceled
	executor := newFakeEngineExecutor()

	poller := NewTimeTriggerPoller(repo, executor, nil, NewTriggerRegistry())
	poller.checkTimeTriggers(context.Background())

	assertNoExecution(t, executor)
	if len(repo.claimedAutomations()) != 2 {
		t.Fatalf("expected a claim attempt for both automations despite the error, got %d", len(repo.claimedAutomations()))
	}
}
