package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/berichte"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

type fakeSweeper struct {
	calls  []time.Time
	result int
	err    error
}

func (f *fakeSweeper) DeleteExpiredCacheEntries(_ context.Context, before time.Time) (int, error) {
	f.calls = append(f.calls, before)
	return f.result, f.err
}

func newSweepScheduler(sw CacheSweeper, every time.Duration) *Scheduler {
	return New(newMockRepo(), &mockRunner{}, &mockExporter{}, &mockMailer{},
		Config{CacheSweeper: sw, CacheSweepEvery: every})
}

func TestMaybeSweepCache_ThrottledToInterval(t *testing.T) {
	sw := &fakeSweeper{}
	s := newSweepScheduler(sw, time.Hour)
	t0 := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)

	s.maybeSweepCache(context.Background(), t0)
	s.maybeSweepCache(context.Background(), t0.Add(59*time.Minute))
	if len(sw.calls) != 1 {
		t.Fatalf("sweeps before the interval elapsed = %d, want 1 (first tick only)", len(sw.calls))
	}
	s.maybeSweepCache(context.Background(), t0.Add(time.Hour))
	if len(sw.calls) != 2 {
		t.Fatalf("sweeps after the interval elapsed = %d, want 2", len(sw.calls))
	}
	if !sw.calls[1].Equal(t0.Add(time.Hour)) {
		t.Errorf("cutoff = %v, want the tick time %v", sw.calls[1], t0.Add(time.Hour))
	}
}

func TestMaybeSweepCache_NoSweeperIsNoop(t *testing.T) {
	s := newSweepScheduler(nil, time.Hour)
	s.maybeSweepCache(context.Background(), time.Now()) // must not panic
}

func TestMaybeSweepCache_ErrorWaitsForNextInterval(t *testing.T) {
	sw := &fakeSweeper{err: errors.New("db down")}
	s := newSweepScheduler(sw, time.Hour)
	t0 := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)

	s.maybeSweepCache(context.Background(), t0)
	s.maybeSweepCache(context.Background(), t0.Add(time.Minute))
	if len(sw.calls) != 1 {
		t.Fatalf("a failed sweep was retried after 1 min: calls = %d, want 1", len(sw.calls))
	}
}

func TestMaybeSweepCache_DefaultIntervalIsOneHour(t *testing.T) {
	sw := &fakeSweeper{}
	s := newSweepScheduler(sw, 0)
	t0 := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	s.maybeSweepCache(context.Background(), t0)
	s.maybeSweepCache(context.Background(), t0.Add(30*time.Minute))
	s.maybeSweepCache(context.Background(), t0.Add(61*time.Minute))
	if len(sw.calls) != 2 {
		t.Fatalf("sweeps = %d, want 2 (default interval 1h)", len(sw.calls))
	}
}

// TestMaybeSweepCache_RemovesOnlyExpiredRows runs the sweep against the real
// repository as kmuhub_app: the expired row of one tenant disappears, the
// fresh one survives, and a second tenant's expired row goes too (the sweep is
// cross-tenant, so it depends on the system context Run installs).
func TestMaybeSweepCache_RemovesOnlyExpiredRows(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close) // registered first, so row cleanups (LIFO) still have a pool

	tenant, other := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Berichte Sweep Tenant")
	testutil.EnsureTenant(t, pool, other, "Berichte Sweep Other Tenant")

	repo := berichte.NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	otherCtx := testutil.WithTenantCtx(context.Background(), other)
	sysCtx := testutil.WithSystemCtx(context.Background())

	now := time.Now().UTC()
	seedDef := func(c context.Context, tid uuid.UUID) uuid.UUID {
		def := &berichte.Definition{
			ID: uuid.New(), TenantID: tid, Name: "Sweep Def", Module: "finanzen", Kind: "custom",
			QueryConfig: []byte(`{}`), DefaultFormat: "pdf", CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.CreateDefinition(c, def); err != nil {
			t.Fatalf("seed definition: %v", err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_definitions", def.ID) })
		return def.ID
	}
	seedEntry := func(c context.Context, tid, defID uuid.UUID, hash string, expires time.Time) uuid.UUID {
		e := &berichte.CacheEntry{
			ID: uuid.New(), TenantID: tid, DefinitionID: defID, ParamsHash: hash,
			Result: []byte(`{}`), ComputedAt: now, ExpiresAt: expires,
		}
		if err := repo.UpsertCacheEntry(c, e); err != nil {
			t.Fatalf("seed cache entry %q: %v", hash, err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_cache", e.ID) })
		return e.ID
	}

	def, otherDef := seedDef(ctx, tenant), seedDef(otherCtx, other)
	seedEntry(ctx, tenant, def, "expired", now.Add(-time.Minute))
	seedEntry(ctx, tenant, def, "fresh", now.Add(time.Hour))
	seedEntry(otherCtx, other, otherDef, "expired-other", now.Add(-time.Minute))

	s := New(newMockRepo(), &mockRunner{}, &mockExporter{}, &mockMailer{},
		Config{CacheSweeper: repo})
	s.maybeSweepCache(sysCtx, now)

	if _, err := repo.GetCacheEntry(ctx, tenant, def, "expired"); !errors.Is(err, berichte.ErrCacheMiss) {
		t.Errorf("expired entry after sweep: err = %v, want ErrCacheMiss", err)
	}
	if _, err := repo.GetCacheEntry(otherCtx, other, otherDef, "expired-other"); !errors.Is(err, berichte.ErrCacheMiss) {
		t.Errorf("other tenant's expired entry after sweep: err = %v, want ErrCacheMiss", err)
	}
	if _, err := repo.GetCacheEntry(ctx, tenant, def, "fresh"); err != nil {
		t.Errorf("fresh entry after sweep: err = %v, want it kept", err)
	}
}
