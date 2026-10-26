package savedfilter

// DB tests for the saved_filters repository, run as kmuhub_app (RLS enforced).
// Every test creates its own tenants and users so the parallel tests never share
// the per-user default-filter unique index.

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

type filterFixture struct {
	pool      *pgxpool.Pool
	repo      *PostgresRepository
	tenant    uuid.UUID
	other     uuid.UUID
	userA     uuid.UUID
	userB     uuid.UUID
	otherUser uuid.UUID
	ctx       context.Context
	ctxOther  context.Context
	sysCtx    context.Context
}

func newFilterFixture(t *testing.T) *filterFixture {
	t.Helper()
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	fx := &filterFixture{pool: pool, repo: NewPostgresRepository(pool), tenant: uuid.New(), other: uuid.New()}
	testutil.EnsureTenant(t, pool, fx.tenant, "SavedFilter Repo Tenant")
	testutil.EnsureTenant(t, pool, fx.other, "SavedFilter Repo Other Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", fx.tenant) })
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", fx.other) })

	seedUser := func(tenant uuid.UUID, tag string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "users", map[string]any{
			"tenant_id":     tenant,
			"email":         fmt.Sprintf("sf-repo-%s-%s@savedfilter.local", tag, uuid.New().String()[:8]),
			"password_hash": "$argon2id$v=19$test",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
		return id
	}
	fx.userA = seedUser(fx.tenant, "a")
	fx.userB = seedUser(fx.tenant, "b")
	fx.otherUser = seedUser(fx.other, "o")

	fx.ctx = testutil.WithTenantCtx(context.Background(), fx.tenant)
	fx.ctxOther = testutil.WithTenantCtx(context.Background(), fx.other)
	fx.sysCtx = testutil.WithSystemCtx(context.Background())
	return fx
}

func (fx *filterFixture) create(t *testing.T, ctx context.Context, tenant, user uuid.UUID, name string, et models.EntityType, def bool) *models.SavedFilter {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	f := &models.SavedFilter{
		ID: uuid.New(), TenantID: tenant, Name: name, EntityType: et,
		FilterJSON: `{"status":"open"}`, IsDefault: def, CreatedBy: user,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := fx.repo.Create(ctx, f); err != nil {
		t.Fatalf("Create %q: %v", name, err)
	}
	return f
}

func filterNames(fs []*models.SavedFilter) []string {
	out := make([]string, 0, len(fs))
	for _, f := range fs {
		out = append(out, f.Name)
	}
	return out
}

func cancelled(ctx context.Context) context.Context {
	c, cancel := context.WithCancel(ctx)
	cancel()
	return c
}

func TestRepoList_OrderingFiltersAndTenantScope(t *testing.T) {
	fx := newFilterFixture(t)
	contact, company := models.EntityType("contact"), models.EntityType("company")

	// Created out of order: default first in the result, then name ASC.
	fx.create(t, fx.ctx, fx.tenant, fx.userA, "Charlie", contact, false)
	fx.create(t, fx.ctx, fx.tenant, fx.userA, "Alpha", contact, false)
	def := fx.create(t, fx.ctx, fx.tenant, fx.userA, "Zulu-Default", contact, true)
	fx.create(t, fx.ctx, fx.tenant, fx.userA, "Bravo-Company", company, false)
	fx.create(t, fx.ctx, fx.tenant, fx.userB, "Delta-B", contact, false)
	fx.create(t, fx.ctxOther, fx.other, fx.otherUser, "Foreign", contact, false)

	t.Run("all of the tenant sorted default first then name", func(t *testing.T) {
		got, err := fx.repo.List(fx.ctx, fx.tenant, ListFilter{})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		want := []string{"Zulu-Default", "Alpha", "Bravo-Company", "Charlie", "Delta-B"}
		if !slices.Equal(filterNames(got), want) {
			t.Fatalf("order = %v, want %v", filterNames(got), want)
		}
		first := got[0]
		if first.ID != def.ID || first.TenantID != fx.tenant || first.EntityType != contact ||
			!first.IsDefault || first.CreatedBy != fx.userA || first.FilterJSON == "" ||
			first.CreatedAt.IsZero() || first.UpdatedAt.IsZero() {
			t.Fatalf("columns not scanned into the right fields: %+v", first)
		}
	})

	t.Run("entity type filter", func(t *testing.T) {
		got, err := fx.repo.List(fx.ctx, fx.tenant, ListFilter{EntityType: &company})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if !slices.Equal(filterNames(got), []string{"Bravo-Company"}) {
			t.Fatalf("got %v", filterNames(got))
		}
	})

	t.Run("user filter", func(t *testing.T) {
		got, err := fx.repo.List(fx.ctx, fx.tenant, ListFilter{UserID: &fx.userB})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if !slices.Equal(filterNames(got), []string{"Delta-B"}) {
			t.Fatalf("got %v", filterNames(got))
		}
	})

	t.Run("entity type and user combine", func(t *testing.T) {
		got, err := fx.repo.List(fx.ctx, fx.tenant, ListFilter{EntityType: &contact, UserID: &fx.userA})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if !slices.Equal(filterNames(got), []string{"Zulu-Default", "Alpha", "Charlie"}) {
			t.Fatalf("got %v", filterNames(got))
		}
	})

	t.Run("tenant predicate holds even under the system context", func(t *testing.T) {
		got, err := fx.repo.List(fx.sysCtx, fx.other, ListFilter{})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if !slices.Equal(filterNames(got), []string{"Foreign"}) {
			t.Fatalf("other tenant sees %v", filterNames(got))
		}
	})

	t.Run("foreign session sees nothing of this tenant", func(t *testing.T) {
		got, err := fx.repo.List(fx.ctxOther, fx.tenant, ListFilter{})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("RLS leak: %v", filterNames(got))
		}
	})

	t.Run("unknown user yields an empty result", func(t *testing.T) {
		nobody := uuid.New()
		got, err := fx.repo.List(fx.ctx, fx.tenant, ListFilter{UserID: &nobody})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("got %v", filterNames(got))
		}
	})

	t.Run("cancelled context errors", func(t *testing.T) {
		if _, err := fx.repo.List(cancelled(fx.ctx), fx.tenant, ListFilter{}); err == nil {
			t.Fatal("expected an error")
		}
	})
}

func TestRepoClearDefault(t *testing.T) {
	fx := newFilterFixture(t)
	contact, company := models.EntityType("contact"), models.EntityType("company")

	target := fx.create(t, fx.ctx, fx.tenant, fx.userA, "A-contact-default", contact, true)
	keepType := fx.create(t, fx.ctx, fx.tenant, fx.userA, "A-company-default", company, true)
	keepUser := fx.create(t, fx.ctx, fx.tenant, fx.userB, "B-contact-default", contact, true)
	keepTenant := fx.create(t, fx.ctxOther, fx.other, fx.otherUser, "O-contact-default", contact, true)
	plain := fx.create(t, fx.ctx, fx.tenant, fx.userA, "A-contact-plain", contact, false)

	before, err := fx.repo.GetByID(fx.ctx, target.ID, fx.tenant)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if err := fx.repo.ClearDefault(fx.ctx, fx.tenant, fx.userA, contact); err != nil {
		t.Fatalf("ClearDefault: %v", err)
	}

	isDefault := func(ctx context.Context, f *models.SavedFilter) bool {
		t.Helper()
		got, err := fx.repo.GetByID(ctx, f.ID, f.TenantID)
		if err != nil {
			t.Fatalf("GetByID %s: %v", f.Name, err)
		}
		return got.IsDefault
	}
	if isDefault(fx.ctx, target) {
		t.Fatal("the user's default of that entity type was not cleared")
	}
	if !isDefault(fx.ctx, keepType) {
		t.Fatal("another entity type lost its default")
	}
	if !isDefault(fx.ctx, keepUser) {
		t.Fatal("another user's default was cleared")
	}
	if !isDefault(fx.ctxOther, keepTenant) {
		t.Fatal("another tenant's default was cleared")
	}
	if isDefault(fx.ctx, plain) {
		t.Fatal("a non-default filter became default")
	}

	after, err := fx.repo.GetByID(fx.ctx, target.ID, fx.tenant)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if !after.UpdatedAt.After(before.UpdatedAt) {
		t.Fatalf("updated_at not bumped: %v -> %v", before.UpdatedAt, after.UpdatedAt)
	}

	// With the old default gone, a new one can be set for the same user and type.
	fx.create(t, fx.ctx, fx.tenant, fx.userA, "A-contact-new-default", contact, true)

	t.Run("no defaults to clear is not an error", func(t *testing.T) {
		if err := fx.repo.ClearDefault(fx.ctx, fx.tenant, uuid.New(), contact); err != nil {
			t.Fatalf("ClearDefault: %v", err)
		}
	})

	t.Run("foreign tenant argument leaves the rows alone", func(t *testing.T) {
		if err := fx.repo.ClearDefault(fx.sysCtx, fx.other, fx.userA, contact); err != nil {
			t.Fatalf("ClearDefault: %v", err)
		}
		got, err := fx.repo.List(fx.ctx, fx.tenant, ListFilter{UserID: &fx.userA, EntityType: &contact})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) == 0 || !got[0].IsDefault {
			t.Fatalf("default of this tenant was touched: %v", filterNames(got))
		}
	})

	t.Run("cancelled context errors", func(t *testing.T) {
		if err := fx.repo.ClearDefault(cancelled(fx.ctx), fx.tenant, fx.userA, contact); err == nil {
			t.Fatal("expected an error")
		}
	})
}

func TestRepoCreateEnforcesOneDefaultPerUserAndType(t *testing.T) {
	fx := newFilterFixture(t)
	contact := models.EntityType("contact")

	fx.create(t, fx.ctx, fx.tenant, fx.userA, "first", contact, true)

	now := time.Now().UTC()
	dup := &models.SavedFilter{
		ID: uuid.New(), TenantID: fx.tenant, Name: "second", EntityType: contact,
		FilterJSON: `{}`, IsDefault: true, CreatedBy: fx.userA, CreatedAt: now, UpdatedAt: now,
	}
	if err := fx.repo.Create(fx.ctx, dup); err == nil {
		t.Fatal("a second default for the same user and type must violate the unique index")
	}

	dup.ID, dup.IsDefault = uuid.New(), false
	if err := fx.repo.Create(fx.ctx, dup); err != nil {
		t.Fatalf("a non-default sibling must be fine: %v", err)
	}
}

func TestRepoGetUpdateDelete_NotFoundAndErrorPaths(t *testing.T) {
	fx := newFilterFixture(t)
	contact := models.EntityType("contact")
	f := fx.create(t, fx.ctx, fx.tenant, fx.userA, "Roundtrip", contact, true)

	t.Run("GetByID returns every column", func(t *testing.T) {
		got, err := fx.repo.GetByID(fx.ctx, f.ID, fx.tenant)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.Name != f.Name || got.EntityType != contact || !got.IsDefault ||
			got.CreatedBy != fx.userA || got.TenantID != fx.tenant ||
			!got.CreatedAt.Equal(f.CreatedAt) || !got.UpdatedAt.Equal(f.UpdatedAt) {
			t.Fatalf("got %+v, want %+v", got, f)
		}
	})

	t.Run("GetByID with the wrong tenant argument is not found even under the system context", func(t *testing.T) {
		if _, err := fx.repo.GetByID(fx.sysCtx, f.ID, fx.other); !errors.Is(err, ErrFilterNotFound) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("GetByID unknown id", func(t *testing.T) {
		if _, err := fx.repo.GetByID(fx.ctx, uuid.New(), fx.tenant); !errors.Is(err, ErrFilterNotFound) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("GetByID cancelled context is a real error, not not-found", func(t *testing.T) {
		_, err := fx.repo.GetByID(cancelled(fx.ctx), f.ID, fx.tenant)
		if err == nil || errors.Is(err, ErrFilterNotFound) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("Update unknown id is not found", func(t *testing.T) {
		ghost := *f
		ghost.ID = uuid.New()
		if err := fx.repo.Update(fx.ctx, &ghost); !errors.Is(err, ErrFilterNotFound) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("Update with the wrong tenant on the filter keeps the row even under the system context", func(t *testing.T) {
		foreign := *f
		foreign.TenantID, foreign.Name = fx.other, "Hacked"
		if err := fx.repo.Update(fx.sysCtx, &foreign); !errors.Is(err, ErrFilterNotFound) {
			t.Fatalf("got %v", err)
		}
		got, err := fx.repo.GetByID(fx.ctx, f.ID, fx.tenant)
		if err != nil || got.Name != f.Name {
			t.Fatalf("row changed: %+v, %v", got, err)
		}
	})

	t.Run("Update with a cancelled context is a real error", func(t *testing.T) {
		err := fx.repo.Update(cancelled(fx.ctx), f)
		if err == nil || errors.Is(err, ErrFilterNotFound) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("Delete unknown id is not found", func(t *testing.T) {
		if err := fx.repo.Delete(fx.ctx, uuid.New(), fx.tenant); !errors.Is(err, ErrFilterNotFound) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("Delete with a cancelled context is a real error", func(t *testing.T) {
		err := fx.repo.Delete(cancelled(fx.ctx), f.ID, fx.tenant)
		if err == nil || errors.Is(err, ErrFilterNotFound) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("Delete with the wrong tenant argument keeps the row", func(t *testing.T) {
		if err := fx.repo.Delete(fx.sysCtx, f.ID, fx.other); !errors.Is(err, ErrFilterNotFound) {
			t.Fatalf("got %v", err)
		}
		testutil.AssertRowCount(t, fx.pool, fx.sysCtx, "saved_filters", f.ID, 1)
	})

	// Update last: it rewrites the row the cases above rely on.
	t.Run("Update persists name, json, default flag and timestamp", func(t *testing.T) {
		upd := *f
		upd.Name, upd.FilterJSON, upd.IsDefault = "Renamed", `{"status":"won"}`, false
		upd.UpdatedAt = f.UpdatedAt.Add(time.Hour)
		if err := fx.repo.Update(fx.ctx, &upd); err != nil {
			t.Fatalf("Update: %v", err)
		}
		got, err := fx.repo.GetByID(fx.ctx, f.ID, fx.tenant)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.Name != "Renamed" || got.IsDefault || !got.UpdatedAt.Equal(upd.UpdatedAt) {
			t.Fatalf("update did not land: %+v", got)
		}
		if got.FilterJSON != `{"status": "won"}` {
			t.Fatalf("filter_json = %s", got.FilterJSON)
		}
		if got.CreatedBy != fx.userA || got.EntityType != contact {
			t.Fatalf("immutable columns changed: %+v", got)
		}
	})
}
