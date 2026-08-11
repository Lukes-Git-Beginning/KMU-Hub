package bexio

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// schedulerTestConfigRepo is a mockConfigRepo variant with a configurable
// ListActiveByPlatform for scheduler-specific StartAll tests.
type schedulerTestConfigRepo struct {
	mockConfigRepo
	listActiveFn func(ctx context.Context, platform string) ([]*IntegrationConfig, error)
}

func (r *schedulerTestConfigRepo) ListActiveByPlatform(ctx context.Context, platform string) ([]*IntegrationConfig, error) {
	if r.listActiveFn != nil {
		return r.listActiveFn(ctx, platform)
	}
	return nil, nil
}

// schedulerTestRepo wraps mockRepository with a configurable GetSyncConfig
// that supports call counting for AddTenant tests.
type schedulerTestRepo struct {
	mu               sync.Mutex
	getSyncConfigFn  func(ctx context.Context, configID uuid.UUID) (*models.BexioSyncConfig, error)
	getSyncConfigErr error
}

func (r *schedulerTestRepo) GetSyncConfig(ctx context.Context, configID uuid.UUID) (*models.BexioSyncConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.getSyncConfigFn != nil {
		return r.getSyncConfigFn(ctx, configID)
	}
	if r.getSyncConfigErr != nil {
		return nil, r.getSyncConfigErr
	}
	// Realistic positive intervals so a background runTenantScheduler
	// goroutine never calls time.NewTicker with a non-positive duration.
	return &models.BexioSyncConfig{
		ContactSyncIntervalMin: 1440,
		PaymentPollIntervalMin: 1440,
		InvoicePullIntervalMin: 1440,
	}, nil
}

func (r *schedulerTestRepo) UpsertSyncConfig(context.Context, *models.BexioSyncConfig) error { return nil }
func (r *schedulerTestRepo) UpdateLastSyncTime(context.Context, uuid.UUID, string, time.Time) error {
	return nil
}
func (r *schedulerTestRepo) GetEntityMapping(context.Context, uuid.UUID, string, uuid.UUID) (*models.BexioEntityMapping, error) {
	return nil, nil
}
func (r *schedulerTestRepo) GetEntityMappingByBexioID(context.Context, uuid.UUID, string, int) (*models.BexioEntityMapping, error) {
	return nil, nil
}
func (r *schedulerTestRepo) UpsertEntityMapping(context.Context, *models.BexioEntityMapping) error {
	return nil
}
func (r *schedulerTestRepo) ListEntityMappings(context.Context, uuid.UUID, string) ([]models.BexioEntityMapping, error) {
	return nil, nil
}
func (r *schedulerTestRepo) DeleteEntityMapping(context.Context, uuid.UUID) error { return nil }
func (r *schedulerTestRepo) GetFieldMappings(context.Context, uuid.UUID, string) (*models.BexioFieldMapping, error) {
	return nil, nil
}
func (r *schedulerTestRepo) UpsertFieldMappings(context.Context, *models.BexioFieldMapping) error {
	return nil
}
func (r *schedulerTestRepo) CreateSyncLog(context.Context, *models.BexioSyncLog) error { return nil }
func (r *schedulerTestRepo) UpdateSyncLog(context.Context, *models.BexioSyncLog) error { return nil }
func (r *schedulerTestRepo) ListSyncLogs(context.Context, uuid.UUID, int) ([]models.BexioSyncLog, error) {
	return nil, nil
}
func (r *schedulerTestRepo) GetLatestSyncLog(context.Context, uuid.UUID, string) (*models.BexioSyncLog, error) {
	return nil, nil
}

// --- NewScheduler ---

func TestNewScheduler(t *testing.T) {
	svc := newTestService(nil, nil, nil)
	repo := &schedulerTestRepo{}
	cr := &schedulerTestConfigRepo{}

	s := NewScheduler(svc, repo, cr)

	require.NotNil(t, s)
	assert.Same(t, svc, s.service)
	assert.Same(t, repo, s.repo)
	assert.Same(t, cr, s.configRepo)
	assert.NotNil(t, s.tenants)
	assert.Empty(t, s.tenants)
}

// --- StartAll ---

func TestStartAll_NoActiveConfigs(t *testing.T) {
	cr := &schedulerTestConfigRepo{
		listActiveFn: func(context.Context, string) ([]*IntegrationConfig, error) {
			return nil, nil
		},
	}
	s := NewScheduler(newTestService(nil, nil, nil), &schedulerTestRepo{}, cr)

	err := s.StartAll(context.Background())

	require.NoError(t, err)
	assert.Empty(t, s.tenants)
}

func TestStartAll_ListError_SkipsGracefully(t *testing.T) {
	cr := &schedulerTestConfigRepo{
		listActiveFn: func(context.Context, string) ([]*IntegrationConfig, error) {
			return nil, errors.New("db unreachable")
		},
	}
	s := NewScheduler(newTestService(nil, nil, nil), &schedulerTestRepo{}, cr)

	err := s.StartAll(context.Background())

	// StartAll treats a listing error as "nothing to start", not a fatal error —
	// a Bexio outage at gateway boot must not block gateway startup.
	require.NoError(t, err)
	assert.Empty(t, s.tenants)
}

func TestStartAll_MixedSuccessAndFailure(t *testing.T) {
	okConfigID := uuid.New()
	okTenantID := uuid.New()
	failConfigID := uuid.New()
	failTenantID := uuid.New()

	cr := &schedulerTestConfigRepo{
		listActiveFn: func(context.Context, string) ([]*IntegrationConfig, error) {
			return []*IntegrationConfig{
				{ID: okConfigID, TenantID: okTenantID, IsActive: true},
				{ID: failConfigID, TenantID: failTenantID, IsActive: true},
			}, nil
		},
	}
	repo := &schedulerTestRepo{
		getSyncConfigFn: func(_ context.Context, configID uuid.UUID) (*models.BexioSyncConfig, error) {
			if configID == failConfigID {
				return nil, errors.New("sync config missing")
			}
			return &models.BexioSyncConfig{
				ContactSyncIntervalMin: 1440,
				PaymentPollIntervalMin: 1440,
				InvoicePullIntervalMin: 1440,
			}, nil
		},
	}
	s := NewScheduler(newTestService(nil, nil, nil), repo, cr)

	err := s.StartAll(context.Background())
	require.NoError(t, err)

	// Only the tenant whose GetSyncConfig succeeded should remain registered;
	// AddTenant must clean up its own map entry on failure.
	s.mu.Lock()
	assert.Len(t, s.tenants, 1)
	_, ok := s.tenants[okTenantID]
	assert.True(t, ok)
	_, failOk := s.tenants[failTenantID]
	assert.False(t, failOk)
	s.mu.Unlock()

	s.StopAll()
}

// --- AddTenant ---

func TestAddTenant_GetSyncConfigError_CleansUpAndReturnsErr(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()
	repo := &schedulerTestRepo{getSyncConfigErr: errors.New("row not found")}
	s := NewScheduler(newTestService(nil, nil, nil), repo, &schedulerTestConfigRepo{})

	err := s.AddTenant(context.Background(), configID, tenantID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "row not found")
	s.mu.Lock()
	_, ok := s.tenants[tenantID]
	s.mu.Unlock()
	assert.False(t, ok, "a tenant whose sync config failed to load must not remain in the map")
}

func TestAddTenant_Success_RegistersTenant(t *testing.T) {
	configID := uuid.New()
	tenantID := uuid.New()
	s := NewScheduler(newTestService(nil, nil, nil), &schedulerTestRepo{}, &schedulerTestConfigRepo{})

	err := s.AddTenant(context.Background(), configID, tenantID)
	require.NoError(t, err)

	s.mu.Lock()
	ts, ok := s.tenants[tenantID]
	s.mu.Unlock()
	require.True(t, ok)
	assert.Equal(t, configID, ts.configID)
	assert.Equal(t, tenantID, ts.tenantID)

	// Clean up the background goroutine spawned by AddTenant.
	s.RemoveTenant(tenantID)
}

func TestAddTenant_ReplacesExistingScheduler(t *testing.T) {
	tenantID := uuid.New()
	firstConfigID := uuid.New()
	secondConfigID := uuid.New()
	s := NewScheduler(newTestService(nil, nil, nil), &schedulerTestRepo{}, &schedulerTestConfigRepo{})

	require.NoError(t, s.AddTenant(context.Background(), firstConfigID, tenantID))

	s.mu.Lock()
	firstTS := s.tenants[tenantID]
	s.mu.Unlock()
	require.NotNil(t, firstTS)

	// Adding the same tenant again must cancel the previous scheduler and
	// replace it — a stale goroutine must never keep running unmonitored.
	require.NoError(t, s.AddTenant(context.Background(), secondConfigID, tenantID))

	// firstTS.cancel is the context.CancelFunc AddTenant created for the first
	// registration; calling it a second time here is a no-op if AddTenant
	// already canceled it (proves the swap happened, doesn't panic either way).
	firstTS.cancel()

	s.mu.Lock()
	secondTS, ok := s.tenants[tenantID]
	s.mu.Unlock()
	require.True(t, ok)
	assert.Equal(t, secondConfigID, secondTS.configID)
	assert.NotSame(t, firstTS, secondTS)

	s.RemoveTenant(tenantID)
}

// --- RemoveTenant ---

func TestRemoveTenant_Existing_CancelsAndDeletes(t *testing.T) {
	tenantID := uuid.New()
	configID := uuid.New()
	s := NewScheduler(newTestService(nil, nil, nil), &schedulerTestRepo{}, &schedulerTestConfigRepo{})
	require.NoError(t, s.AddTenant(context.Background(), configID, tenantID))

	s.RemoveTenant(tenantID)

	s.mu.Lock()
	_, ok := s.tenants[tenantID]
	s.mu.Unlock()
	assert.False(t, ok)
}

func TestRemoveTenant_Unknown_NoOp(t *testing.T) {
	s := NewScheduler(newTestService(nil, nil, nil), &schedulerTestRepo{}, &schedulerTestConfigRepo{})

	// Must not panic on an unregistered tenant.
	s.RemoveTenant(uuid.New())

	assert.Empty(t, s.tenants)
}

// --- StopAll ---

func TestStopAll_StopsAllAndClearsMap(t *testing.T) {
	s := NewScheduler(newTestService(nil, nil, nil), &schedulerTestRepo{}, &schedulerTestConfigRepo{})
	require.NoError(t, s.AddTenant(context.Background(), uuid.New(), uuid.New()))
	require.NoError(t, s.AddTenant(context.Background(), uuid.New(), uuid.New()))
	require.Len(t, s.tenants, 2)

	s.StopAll()

	assert.Empty(t, s.tenants)
}

func TestStopAll_Empty_NoOp(t *testing.T) {
	s := NewScheduler(newTestService(nil, nil, nil), &schedulerTestRepo{}, &schedulerTestConfigRepo{})

	// Must not panic when nothing was ever started.
	s.StopAll()

	assert.Empty(t, s.tenants)
}

// --- runTenantScheduler ---

// TestRunTenantScheduler_ReturnsImmediatelyOnCanceledContext exercises the
// interval computation (including the InvoicePullIntervalMin<=0 guard) and
// the ctx.Done() shutdown path without waiting for a real tick — the ticker
// intervals are minute-granularity in production, so no test can wait for an
// actual tick to fire; canceling ctx before entry proves the select picks
// the shutdown branch and every ticker.Stop() defer runs without panicking.
func TestRunTenantScheduler_ReturnsImmediatelyOnCanceledContext(t *testing.T) {
	svc := newTestService(nil, nil, nil)
	s := NewScheduler(svc, &schedulerTestRepo{}, &schedulerTestConfigRepo{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ts := &tenantScheduler{cancel: cancel, configID: uuid.New(), tenantID: uuid.New()}
	syncConfig := &models.BexioSyncConfig{
		ContactSyncEnabled:      true,
		PaymentPollEnabled:      true,
		InvoicePullEnabled:      true,
		ContactSyncIntervalMin:  60,
		PaymentPollIntervalMin:  60,
		InvoicePullIntervalMin:  0, // exercises the <=0 guard defaulting to 15min
	}

	done := make(chan struct{})
	go func() {
		s.runTenantScheduler(ctx, ts, syncConfig)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("runTenantScheduler did not return after ctx cancellation")
	}
}

// TestRunTenantScheduler_RecoversFromPanic proves the deferred recover()
// keeps a panicking tenant scheduler from taking down the process — a
// non-positive ticker interval (e.g. a corrupt row with IntervalMin<=0 for
// contact/payment, which has no defaulting guard unlike invoice pull) would
// otherwise panic inside the goroutine.
func TestRunTenantScheduler_RecoversFromPanic(t *testing.T) {
	svc := newTestService(nil, nil, nil)
	s := NewScheduler(svc, &schedulerTestRepo{}, &schedulerTestConfigRepo{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ts := &tenantScheduler{cancel: cancel, configID: uuid.New(), tenantID: uuid.New()}
	syncConfig := &models.BexioSyncConfig{
		ContactSyncIntervalMin: 0, // time.NewTicker(0) panics
		PaymentPollIntervalMin: 60,
		InvoicePullIntervalMin: 60,
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		// If runTenantScheduler's own recover() did not catch the panic,
		// this goroutine would crash the test binary instead of failing
		// this test cleanly.
		s.runTenantScheduler(ctx, ts, syncConfig)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("runTenantScheduler goroutine did not return after panic")
	}
}
