package trigger

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/automation/workflow"
	"github.com/kmuhub/kmuhub/internal/models"
)

// fakeListRepo implements workflow.Repository with only List wired -- every
// other method is an unused no-op stub, since TriggerMatcher never calls them.
type fakeListRepo struct {
	mu sync.Mutex

	listResult []*models.Automation
	listErr    error
	calls      int
}

func (f *fakeListRepo) List(_ context.Context, _ workflow.ListFilter) ([]*models.Automation, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.listErr != nil {
		return nil, 0, f.listErr
	}
	return f.listResult, len(f.listResult), nil
}

func (f *fakeListRepo) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *fakeListRepo) Create(context.Context, *models.Automation) error { return nil }
func (f *fakeListRepo) Update(context.Context, *models.Automation) error { return nil }
func (f *fakeListRepo) Delete(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}
func (f *fakeListRepo) GetByID(context.Context, uuid.UUID, uuid.UUID) (*models.Automation, error) {
	return nil, nil
}
func (f *fakeListRepo) GetByIDUnscoped(context.Context, uuid.UUID) (*models.Automation, error) {
	return nil, nil
}
func (f *fakeListRepo) ListActiveByTriggerType(context.Context, string) ([]*models.Automation, error) {
	return nil, nil
}
func (f *fakeListRepo) ListActiveTimeBased(context.Context, []string) ([]*models.Automation, error) {
	return nil, nil
}
func (f *fakeListRepo) SetActive(context.Context, uuid.UUID, uuid.UUID, bool) error {
	return nil
}
func (f *fakeListRepo) UpdateLastTriggered(context.Context, uuid.UUID, time.Time) error {
	return nil
}
func (f *fakeListRepo) ClaimTimeTrigger(context.Context, uuid.UUID, *time.Time, time.Time) (bool, error) {
	return true, nil
}
func (f *fakeListRepo) ClaimTimeTriggerFire(context.Context, uuid.UUID, uuid.UUID, string, time.Time) (bool, error) {
	return true, nil
}

func newAutomation(triggerType string) *models.Automation {
	return &models.Automation{
		ID:          uuid.New(),
		TriggerType: triggerType,
		IsActive:    true,
		Actions:     []byte(`[]`),
	}
}

// ============================================================================
// FindMatching
// ============================================================================

func TestFindMatching_ReturnsOnlyAutomationsWhoseRegisteredEventTypeMatches(t *testing.T) {
	registry := NewTriggerRegistry()
	matching := newAutomation("crm.deal.stage_changed") // registry.Get succeeds, EventType == "crm.deal.stage_changed"
	unregistered := newAutomation("no.such.trigger")    // registry.Get fails -> skipped, no panic
	other := newAutomation("work.task.created")         // registered, but a different EventType

	repo := &fakeListRepo{listResult: []*models.Automation{matching, unregistered, other}}
	matcher := NewTriggerMatcher(repo, registry)

	matches, err := matcher.FindMatching(context.Background(), "crm.deal.stage_changed")
	require.NoError(t, err)
	require.Len(t, matches, 1)
	assert.Equal(t, matching.ID, matches[0].Automation.ID)
	assert.Equal(t, "crm.deal.stage_changed", matches[0].TriggerDef.EventType)
}

func TestFindMatching_NoMatches_ReturnsEmptyWithoutError(t *testing.T) {
	registry := NewTriggerRegistry()
	repo := &fakeListRepo{listResult: []*models.Automation{newAutomation("work.task.created")}}
	matcher := NewTriggerMatcher(repo, registry)

	matches, err := matcher.FindMatching(context.Background(), "crm.deal.stage_changed")
	require.NoError(t, err)
	assert.Empty(t, matches)
}

// ============================================================================
// TTL cache
// ============================================================================

func TestFindMatching_CachesWithinTTL_SecondCallSkipsRepo(t *testing.T) {
	registry := NewTriggerRegistry()
	repo := &fakeListRepo{listResult: []*models.Automation{newAutomation("crm.deal.stage_changed")}}
	matcher := NewTriggerMatcher(repo, registry)

	_, err := matcher.FindMatching(context.Background(), "crm.deal.stage_changed")
	require.NoError(t, err)
	_, err = matcher.FindMatching(context.Background(), "crm.deal.stage_changed")
	require.NoError(t, err)

	assert.Equal(t, 1, repo.callCount(), "second call within the TTL must be served from cache, not hit the repo again")
}

func TestFindMatching_StaleCacheAfterTTLExpiry_FallsBackOnRepoError(t *testing.T) {
	registry := NewTriggerRegistry()
	firstAuto := newAutomation("crm.deal.stage_changed")
	repo := &fakeListRepo{listResult: []*models.Automation{firstAuto}}

	// Bypass NewTriggerMatcher's fixed 30s TTL so the test doesn't need to sleep.
	matcher := &TriggerMatcher{
		workflowRepo: repo,
		registry:     registry,
		cacheTTL:     20 * time.Millisecond,
	}

	matches, err := matcher.FindMatching(context.Background(), "crm.deal.stage_changed")
	require.NoError(t, err)
	require.Len(t, matches, 1)
	assert.Equal(t, firstAuto.ID, matches[0].Automation.ID)

	time.Sleep(30 * time.Millisecond) // let the TTL expire
	repo.mu.Lock()
	repo.listErr = assertAnError
	repo.mu.Unlock()

	matches, err = matcher.FindMatching(context.Background(), "crm.deal.stage_changed")
	require.NoError(t, err, "a refresh failure after the cache was already populated must fall back to the stale cache, not error")
	require.Len(t, matches, 1)
	assert.Equal(t, firstAuto.ID, matches[0].Automation.ID, "stale-cache fallback must still be the original, pre-error data")
}

func TestFindMatching_ListErrorOnFirstCall_ReturnsError(t *testing.T) {
	registry := NewTriggerRegistry()
	repo := &fakeListRepo{listErr: assertAnError}
	matcher := NewTriggerMatcher(repo, registry)

	matches, err := matcher.FindMatching(context.Background(), "crm.deal.stage_changed")
	require.Error(t, err, "with no cache populated yet, a repo error must propagate")
	assert.Nil(t, matches)
}

// ============================================================================
// InvalidateCache
// ============================================================================

func TestInvalidateCache_ForcesRefreshDespiteUnexpiredTTL(t *testing.T) {
	registry := NewTriggerRegistry()
	repo := &fakeListRepo{listResult: []*models.Automation{newAutomation("crm.deal.stage_changed")}}
	matcher := NewTriggerMatcher(repo, registry) // default 30s TTL, plenty of headroom

	_, err := matcher.FindMatching(context.Background(), "crm.deal.stage_changed")
	require.NoError(t, err)
	assert.Equal(t, 1, repo.callCount())

	matcher.InvalidateCache()

	_, err = matcher.FindMatching(context.Background(), "crm.deal.stage_changed")
	require.NoError(t, err)
	assert.Equal(t, 2, repo.callCount(), "InvalidateCache must force the next FindMatching to hit the repo even though the 30s TTL has not elapsed")
}

// assertAnError is a stand-in repo failure for tests that only care that an
// error occurred, not its identity.
var assertAnError = &staticError{"list failed"}

type staticError struct{ msg string }

func (e *staticError) Error() string { return e.msg }
