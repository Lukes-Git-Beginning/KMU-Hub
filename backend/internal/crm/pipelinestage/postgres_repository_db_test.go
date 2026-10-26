package pipelinestage

// cov-crm-pipelinestage: exercises the read, reorder and check queries of the
// PostgreSQL repository against the real schema as kmuhub_app. The write
// surface is pinned in tenant_write_test.go, the stats currency rule in
// list_with_stats_currency_db_test.go.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// stageFixture is a repository bound to one fresh tenant. Rows are removed by
// the tenant-level cleanup, so tests only create what they need.
type stageFixture struct {
	repo     *PostgresRepository
	tenantID uuid.UUID
	ctx      context.Context
}

func newStageFixture(t *testing.T) *stageFixture {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Stage Repo Tenant "+tenantID.String()[:8])
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })

	return &stageFixture{
		repo:     NewPostgresRepository(pool),
		tenantID: tenantID,
		ctx:      testutil.WithTenantCtx(context.Background(), tenantID),
	}
}

func (f *stageFixture) create(t *testing.T, name string, sortOrder int, won, lost bool, prob string) *models.PipelineStage {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	s := &models.PipelineStage{
		ID:          uuid.New(),
		TenantID:    f.tenantID,
		Name:        name,
		Color:       "#112233",
		SortOrder:   sortOrder,
		IsWon:       won,
		IsLost:      lost,
		Probability: decimal.RequireFromString(prob),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := f.repo.Create(f.ctx, s); err != nil {
		t.Fatalf("Create %q: %v", name, err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, f.repo.pool, "pipeline_stages", s.ID) })
	return s
}

func TestPostgresRepository_List(t *testing.T) {
	t.Parallel()

	t.Run("orders by sort_order and scans every column", func(t *testing.T) {
		t.Parallel()
		f := newStageFixture(t)
		// Created out of order so only ORDER BY can produce the result.
		c := f.create(t, "Closing", 3, true, false, "100")
		a := f.create(t, "Lead", 1, false, false, "12.5")
		b := f.create(t, "Offer", 2, false, false, "60")

		got, err := f.repo.List(f.ctx, f.tenantID)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("len = %d, want 3", len(got))
		}
		for i, want := range []*models.PipelineStage{a, b, c} {
			if got[i].ID != want.ID {
				t.Fatalf("position %d = %q, want %q", i, got[i].Name, want.Name)
			}
		}
		first := got[0]
		if first.TenantID != f.tenantID || first.Name != "Lead" || first.Color != "#112233" ||
			first.SortOrder != 1 || first.IsWon || first.IsLost {
			t.Errorf("scanned stage = %+v", first)
		}
		if !first.Probability.Equal(decimal.RequireFromString("12.5")) {
			t.Errorf("probability = %s, want 12.5", first.Probability)
		}
		if !first.CreatedAt.Equal(a.CreatedAt) || !first.UpdatedAt.Equal(a.UpdatedAt) {
			t.Errorf("timestamps = %v / %v, want %v / %v", first.CreatedAt, first.UpdatedAt, a.CreatedAt, a.UpdatedAt)
		}
		if !got[2].IsWon {
			t.Errorf("won flag of %q was not scanned", got[2].Name)
		}
	})

	t.Run("a tenant without stages gets an empty list", func(t *testing.T) {
		t.Parallel()
		f := newStageFixture(t)
		got, err := f.repo.List(f.ctx, f.tenantID)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("len = %d, want 0", len(got))
		}
	})

	t.Run("never returns another tenant's stages", func(t *testing.T) {
		t.Parallel()
		own := newStageFixture(t)
		other := newStageFixture(t)
		own.create(t, "Own", 1, false, false, "10")
		foreign := other.create(t, "Foreign", 1, false, false, "10")

		got, err := own.repo.List(own.ctx, own.tenantID)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 1 || got[0].ID == foreign.ID {
			t.Fatalf("own list = %+v, want only the own stage", got)
		}

		// Explicit tenant predicate without RLS help: system ctx sees both
		// tenants, so only the WHERE clause can keep the foreign row out.
		sys := testutil.WithSystemCtx(context.Background())
		got, err = own.repo.List(sys, own.tenantID)
		if err != nil {
			t.Fatalf("List (system ctx): %v", err)
		}
		if len(got) != 1 || got[0].ID == foreign.ID {
			t.Fatalf("system-ctx list = %+v, want only the own stage", got)
		}
	})

	t.Run("a cancelled context is reported", func(t *testing.T) {
		t.Parallel()
		f := newStageFixture(t)
		ctx, cancel := context.WithCancel(f.ctx)
		cancel()
		if _, err := f.repo.List(ctx, f.tenantID); err == nil {
			t.Fatal("List with cancelled ctx: want error, got nil")
		}
	})
}

func TestPostgresRepository_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	f := newStageFixture(t)
	s := f.create(t, "Known", 1, false, false, "5")

	if _, err := f.repo.GetByID(f.ctx, uuid.New(), f.tenantID); !errors.Is(err, ErrStageNotFound) {
		t.Fatalf("unknown id: err = %v, want ErrStageNotFound", err)
	}
	// Right id, wrong tenant under system ctx (RLS off): only the predicate stops it.
	sys := testutil.WithSystemCtx(context.Background())
	if _, err := f.repo.GetByID(sys, s.ID, uuid.New()); !errors.Is(err, ErrStageNotFound) {
		t.Fatalf("foreign tenant predicate: err = %v, want ErrStageNotFound", err)
	}
	ctx, cancel := context.WithCancel(f.ctx)
	cancel()
	if _, err := f.repo.GetByID(ctx, s.ID, f.tenantID); err == nil || errors.Is(err, ErrStageNotFound) {
		t.Fatalf("cancelled ctx: err = %v, want a non-NotFound error", err)
	}
}

func TestPostgresRepository_Delete_UnknownIsNotFound(t *testing.T) {
	t.Parallel()
	f := newStageFixture(t)
	s := f.create(t, "Keep", 1, false, false, "5")

	if err := f.repo.Delete(f.ctx, uuid.New(), f.tenantID); !errors.Is(err, ErrStageNotFound) {
		t.Fatalf("unknown id: err = %v, want ErrStageNotFound", err)
	}
	// Wrong tenant predicate under system ctx must not delete the row.
	sys := testutil.WithSystemCtx(context.Background())
	if err := f.repo.Delete(sys, s.ID, uuid.New()); !errors.Is(err, ErrStageNotFound) {
		t.Fatalf("foreign tenant predicate: err = %v, want ErrStageNotFound", err)
	}
	if _, err := f.repo.GetByID(f.ctx, s.ID, f.tenantID); err != nil {
		t.Fatalf("stage vanished after a foreign-tenant delete: %v", err)
	}
	if err := f.repo.Delete(f.ctx, s.ID, f.tenantID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := f.repo.GetByID(f.ctx, s.ID, f.tenantID); !errors.Is(err, ErrStageNotFound) {
		t.Fatalf("after delete: err = %v, want ErrStageNotFound", err)
	}
}

func TestPostgresRepository_Reorder(t *testing.T) {
	t.Parallel()

	names := func(stages []*models.PipelineStage) []string {
		out := make([]string, len(stages))
		for i, s := range stages {
			out[i] = fmt.Sprintf("%s=%d", s.Name, s.SortOrder)
		}
		return out
	}

	t.Run("assigns 1..n in the given order and touches updated_at", func(t *testing.T) {
		t.Parallel()
		f := newStageFixture(t)
		a := f.create(t, "A", 1, false, false, "10")
		b := f.create(t, "B", 2, false, false, "20")
		c := f.create(t, "C", 3, false, false, "30")

		if err := f.repo.Reorder(f.ctx, f.tenantID, []uuid.UUID{c.ID, a.ID, b.ID}); err != nil {
			t.Fatalf("Reorder: %v", err)
		}
		got, err := f.repo.List(f.ctx, f.tenantID)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		want := []string{"C=1", "A=2", "B=3"}
		have := names(got)
		for i := range want {
			if have[i] != want[i] {
				t.Fatalf("order = %v, want %v", have, want)
			}
		}
		if !got[0].UpdatedAt.After(c.UpdatedAt) {
			t.Errorf("updated_at %v did not move past %v", got[0].UpdatedAt, c.UpdatedAt)
		}
	})

	t.Run("an empty id list is a no-op", func(t *testing.T) {
		t.Parallel()
		f := newStageFixture(t)
		a := f.create(t, "A", 7, false, false, "10")
		if err := f.repo.Reorder(f.ctx, f.tenantID, nil); err != nil {
			t.Fatalf("Reorder(nil): %v", err)
		}
		got, err := f.repo.GetByID(f.ctx, a.ID, f.tenantID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.SortOrder != 7 {
			t.Fatalf("sort_order = %d, want 7", got.SortOrder)
		}
	})

	t.Run("a foreign tenant's stage id is left untouched", func(t *testing.T) {
		t.Parallel()
		own := newStageFixture(t)
		other := newStageFixture(t)
		mine := own.create(t, "Mine", 1, false, false, "10")
		theirs := other.create(t, "Theirs", 5, false, false, "10")

		// System ctx (RLS off) isolates the explicit tenant_id predicate.
		sys := testutil.WithSystemCtx(context.Background())
		if err := own.repo.Reorder(sys, own.tenantID, []uuid.UUID{theirs.ID, mine.ID}); err != nil {
			t.Fatalf("Reorder: %v", err)
		}
		got, err := other.repo.GetByID(other.ctx, theirs.ID, other.tenantID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.SortOrder != 5 {
			t.Fatalf("foreign stage sort_order = %d, want 5 (untouched)", got.SortOrder)
		}
		gotMine, err := own.repo.GetByID(own.ctx, mine.ID, own.tenantID)
		if err != nil {
			t.Fatalf("GetByID own: %v", err)
		}
		if gotMine.SortOrder != 2 {
			t.Fatalf("own stage sort_order = %d, want 2 (index in the list)", gotMine.SortOrder)
		}
	})

	t.Run("a cancelled context is reported and nothing changes", func(t *testing.T) {
		t.Parallel()
		f := newStageFixture(t)
		a := f.create(t, "A", 4, false, false, "10")
		ctx, cancel := context.WithCancel(f.ctx)
		cancel()
		if err := f.repo.Reorder(ctx, f.tenantID, []uuid.UUID{a.ID}); err == nil {
			t.Fatal("Reorder with cancelled ctx: want error, got nil")
		}
		got, err := f.repo.GetByID(f.ctx, a.ID, f.tenantID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.SortOrder != 4 {
			t.Fatalf("sort_order = %d, want 4", got.SortOrder)
		}
	})
}

func TestPostgresRepository_HasDeals(t *testing.T) {
	t.Parallel()
	f := newStageFixture(t)
	pool := f.repo.pool

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"id": uuid.New(), "email": fmt.Sprintf("stage-deals-%s@test.invalid", uuid.New()),
		"password_hash": "x", "tenant_id": f.tenantID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	withDeal := f.create(t, "With deal", 1, false, false, "10")
	empty := f.create(t, "Empty", 2, false, false, "10")
	dealID := testutil.SeedRow(t, pool, "deals", map[string]any{
		"id": uuid.New(), "tenant_id": f.tenantID, "name": "Deal-" + uuid.NewString()[:8],
		"value": "100.00", "currency": "EUR", "stage_id": withDeal.ID, "created_by": userID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "deals", dealID) })

	has, err := f.repo.HasDeals(f.ctx, withDeal.ID, f.tenantID)
	if err != nil || !has {
		t.Fatalf("stage with a deal: has=%v err=%v, want true", has, err)
	}
	has, err = f.repo.HasDeals(f.ctx, empty.ID, f.tenantID)
	if err != nil || has {
		t.Fatalf("stage without deals: has=%v err=%v, want false", has, err)
	}
	// Under system ctx only the tenant predicate separates the tenants.
	sys := testutil.WithSystemCtx(context.Background())
	has, err = f.repo.HasDeals(sys, withDeal.ID, uuid.New())
	if err != nil || has {
		t.Fatalf("foreign tenant: has=%v err=%v, want false", has, err)
	}
	ctx, cancel := context.WithCancel(f.ctx)
	cancel()
	if _, err := f.repo.HasDeals(ctx, withDeal.ID, f.tenantID); err == nil {
		t.Fatal("cancelled ctx: want error, got nil")
	}
}

func TestPostgresRepository_HasWonAndLostStage(t *testing.T) {
	t.Parallel()
	f := newStageFixture(t)
	sys := testutil.WithSystemCtx(context.Background())
	cancelled, cancel := context.WithCancel(f.ctx)
	cancel()

	type check struct {
		name string
		call func(ctx context.Context, tenantID uuid.UUID, exclude *uuid.UUID) (bool, error)
	}
	checks := []check{
		{"won", f.repo.HasWonStage},
		{"lost", f.repo.HasLostStage},
	}

	// Neither flag exists yet.
	plain := f.create(t, "Plain", 1, false, false, "10")
	for _, c := range checks {
		if has, err := c.call(f.ctx, f.tenantID, nil); err != nil || has {
			t.Errorf("%s on a tenant without flagged stages: has=%v err=%v, want false", c.name, has, err)
		}
	}

	won := f.create(t, "Won", 2, true, false, "100")
	lost := f.create(t, "Lost", 3, false, true, "0")
	flagged := map[string]*models.PipelineStage{"won": won, "lost": lost}
	otherFlag := map[string]*models.PipelineStage{"won": lost, "lost": won}

	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			own, other := flagged[c.name], otherFlag[c.name]

			if has, err := c.call(f.ctx, f.tenantID, nil); err != nil || !has {
				t.Errorf("without exclude: has=%v err=%v, want true", has, err)
			}
			// Excluding the only flagged stage leaves none.
			if has, err := c.call(f.ctx, f.tenantID, &own.ID); err != nil || has {
				t.Errorf("excluding the flagged stage: has=%v err=%v, want false", has, err)
			}
			// Excluding a different stage keeps the flagged one visible.
			if has, err := c.call(f.ctx, f.tenantID, &other.ID); err != nil || !has {
				t.Errorf("excluding another stage: has=%v err=%v, want true", has, err)
			}
			if has, err := c.call(f.ctx, f.tenantID, &plain.ID); err != nil || !has {
				t.Errorf("excluding the plain stage: has=%v err=%v, want true", has, err)
			}
			// Tenant predicate with RLS off.
			if has, err := c.call(sys, uuid.New(), nil); err != nil || has {
				t.Errorf("foreign tenant: has=%v err=%v, want false", has, err)
			}
			if has, err := c.call(sys, uuid.New(), &own.ID); err != nil || has {
				t.Errorf("foreign tenant with exclude: has=%v err=%v, want false", has, err)
			}
			if _, err := c.call(cancelled, f.tenantID, nil); err == nil {
				t.Error("cancelled ctx without exclude: want error, got nil")
			}
			if _, err := c.call(cancelled, f.tenantID, &own.ID); err == nil {
				t.Error("cancelled ctx with exclude: want error, got nil")
			}
		})
	}
}

// A tenant that owns only one of the two terminal stages must not report the other.
func TestPostgresRepository_HasWonAndLostStage_FlagsAreIndependent(t *testing.T) {
	t.Parallel()
	onlyWon := newStageFixture(t)
	onlyLost := newStageFixture(t)
	onlyWon.create(t, "Won", 1, true, false, "100")
	onlyLost.create(t, "Lost", 1, false, true, "0")

	// Excluded id is a different, non-existing stage, so the exclude branch runs without hiding the row.
	nobody := uuid.New()
	for _, exclude := range []*uuid.UUID{nil, &nobody} {
		if has, err := onlyWon.repo.HasLostStage(onlyWon.ctx, onlyWon.tenantID, exclude); err != nil || has {
			t.Errorf("only a won stage, HasLostStage(exclude=%v): has=%v err=%v, want false", exclude, has, err)
		}
		if has, err := onlyWon.repo.HasWonStage(onlyWon.ctx, onlyWon.tenantID, exclude); err != nil || !has {
			t.Errorf("only a won stage, HasWonStage(exclude=%v): has=%v err=%v, want true", exclude, has, err)
		}
		if has, err := onlyLost.repo.HasWonStage(onlyLost.ctx, onlyLost.tenantID, exclude); err != nil || has {
			t.Errorf("only a lost stage, HasWonStage(exclude=%v): has=%v err=%v, want false", exclude, has, err)
		}
		if has, err := onlyLost.repo.HasLostStage(onlyLost.ctx, onlyLost.tenantID, exclude); err != nil || !has {
			t.Errorf("only a lost stage, HasLostStage(exclude=%v): has=%v err=%v, want true", exclude, has, err)
		}
	}
}

func TestPostgresRepository_NextSortOrderAndCount(t *testing.T) {
	t.Parallel()
	f := newStageFixture(t)
	other := newStageFixture(t)
	sys := testutil.WithSystemCtx(context.Background())

	next, err := f.repo.GetNextSortOrder(f.ctx, f.tenantID)
	if err != nil || next != 1 {
		t.Fatalf("empty tenant: next=%d err=%v, want 1", next, err)
	}
	count, err := f.repo.CountStages(f.ctx, f.tenantID)
	if err != nil || count != 0 {
		t.Fatalf("empty tenant: count=%d err=%v, want 0", count, err)
	}

	f.create(t, "A", 2, false, false, "10")
	f.create(t, "B", 9, false, false, "10")
	other.create(t, "Foreign", 40, false, false, "10")

	next, err = f.repo.GetNextSortOrder(f.ctx, f.tenantID)
	if err != nil || next != 10 {
		t.Fatalf("next=%d err=%v, want 10 (max own sort_order + 1)", next, err)
	}
	count, err = f.repo.CountStages(f.ctx, f.tenantID)
	if err != nil || count != 2 {
		t.Fatalf("count=%d err=%v, want 2", count, err)
	}

	// System ctx sees both tenants, so only the predicate keeps them apart.
	if next, err = f.repo.GetNextSortOrder(sys, f.tenantID); err != nil || next != 10 {
		t.Fatalf("system ctx: next=%d err=%v, want 10", next, err)
	}
	if count, err = f.repo.CountStages(sys, f.tenantID); err != nil || count != 2 {
		t.Fatalf("system ctx: count=%d err=%v, want 2", count, err)
	}

	ctx, cancel := context.WithCancel(f.ctx)
	cancel()
	if _, err := f.repo.GetNextSortOrder(ctx, f.tenantID); err == nil {
		t.Error("GetNextSortOrder with cancelled ctx: want error, got nil")
	}
	if _, err := f.repo.CountStages(ctx, f.tenantID); err == nil {
		t.Error("CountStages with cancelled ctx: want error, got nil")
	}
}

func TestPostgresRepository_ListWithStats_EmptyStageAndCancelled(t *testing.T) {
	t.Parallel()
	f := newStageFixture(t)
	b := f.create(t, "B", 2, false, false, "40")
	a := f.create(t, "A", 1, false, false, "7.5")

	got, err := f.repo.ListWithStats(f.ctx, f.tenantID)
	if err != nil {
		t.Fatalf("ListWithStats: %v", err)
	}
	if len(got) != 2 || got[0].ID != a.ID || got[1].ID != b.ID {
		t.Fatalf("stats order = %+v, want A then B", got)
	}
	if got[0].DealCount != 0 || !got[0].TotalValue.IsZero() {
		t.Errorf("empty stage: count=%d total=%s, want 0 / 0", got[0].DealCount, got[0].TotalValue)
	}
	if !got[0].Probability.Equal(decimal.RequireFromString("7.5")) {
		t.Errorf("probability = %s, want 7.5", got[0].Probability)
	}

	ctx, cancel := context.WithCancel(f.ctx)
	cancel()
	if _, err := f.repo.ListWithStats(ctx, f.tenantID); err == nil {
		t.Error("cancelled ctx: want error, got nil")
	}
}
