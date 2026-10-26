package tag

// DB-backed tests for the read side of PostgresRepository (List, GetByEntityAndName,
// GetByID, IsInUse). Run as kmuhub_app, so RLS is in force; every test works in its own
// freshly created tenants and never touches the shared testutil.Tenant* rows.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

type tagDBEnv struct {
	pool *pgxpool.Pool
	repo *PostgresRepository
	ctx  context.Context
	sys  context.Context
	t    uuid.UUID
}

// newTagDBEnv creates one tenant and a context scoped to it.
func newTagDBEnv(t *testing.T) tagDBEnv {
	t.Helper()
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Tag DB Tenant "+tenant.String()[:8])
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenant) })

	return tagDBEnv{
		pool: pool,
		repo: NewPostgresRepository(pool),
		ctx:  testutil.WithTenantCtx(context.Background(), tenant),
		sys:  testutil.WithSystemCtx(context.Background()),
		t:    tenant,
	}
}

// seedTag creates a tag through the repository under the tenant's own context.
func (e tagDBEnv) seedTag(t *testing.T, entity models.EntityType, name string) *models.Tag {
	t.Helper()
	tg := &models.Tag{
		ID:         uuid.New(),
		TenantID:   e.t,
		Name:       name,
		Color:      "#3d8abf",
		EntityType: entity,
		CreatedAt:  time.Now().UTC(),
	}
	if err := e.repo.Create(e.ctx, tg); err != nil {
		t.Fatalf("seed tag %q: %v", name, err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, e.pool, "tags", tg.ID) })
	return tg
}

func names(tags []*models.Tag) []string {
	out := make([]string, len(tags))
	for i, tg := range tags {
		out[i] = tg.Name
	}
	return out
}

func TestPostgresRepository_List(t *testing.T) {
	e := newTagDBEnv(t)

	other := newTagDBEnv(t)
	other.seedTag(t, models.EntityType("contact"), "Fremd")

	e.seedTag(t, models.EntityType("deal"), "Aaa-Deal")
	e.seedTag(t, models.EntityType("contact"), "Beta")
	e.seedTag(t, models.EntityType("contact"), "Alpha")

	t.Run("all entity types ordered by entity_type then name", func(t *testing.T) {
		got, total, err := e.repo.List(e.ctx, e.t, nil, 0, 50)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 3 {
			t.Fatalf("total = %d, want 3 (foreign tenant must not count)", total)
		}
		// "Aaa-Deal" sorts first by name alone, so only the entity_type key keeps it last.
		want := []string{"Alpha", "Beta", "Aaa-Deal"}
		for i, n := range names(got) {
			if n != want[i] {
				t.Fatalf("order = %v, want %v", names(got), want)
			}
		}
	})

	t.Run("entity type filter narrows rows and total", func(t *testing.T) {
		et := models.EntityType("contact")
		got, total, err := e.repo.List(e.ctx, e.t, &et, 0, 50)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 2 || len(got) != 2 {
			t.Fatalf("total=%d len=%d, want 2/2", total, len(got))
		}
		if got[0].Name != "Alpha" || got[1].Name != "Beta" {
			t.Fatalf("order = %v, want [Alpha Beta]", names(got))
		}
		for _, tg := range got {
			if tg.EntityType != et {
				t.Fatalf("tag %q has entity type %q, want contact", tg.Name, tg.EntityType)
			}
		}
	})

	t.Run("paging keeps the total of the full set", func(t *testing.T) {
		got, total, err := e.repo.List(e.ctx, e.t, nil, 1, 2)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 3 {
			t.Fatalf("total = %d, want 3", total)
		}
		if len(got) != 2 || got[0].Name != "Beta" || got[1].Name != "Aaa-Deal" {
			t.Fatalf("page = %v, want [Beta Aaa-Deal]", names(got))
		}

		et := models.EntityType("contact")
		got, total, err = e.repo.List(e.ctx, e.t, &et, 1, 1)
		if err != nil {
			t.Fatalf("List filtered: %v", err)
		}
		if total != 2 || len(got) != 1 || got[0].Name != "Beta" {
			t.Fatalf("filtered page total=%d rows=%v, want 2/[Beta]", total, names(got))
		}
	})

	t.Run("offset past the end is empty but keeps the total", func(t *testing.T) {
		got, total, err := e.repo.List(e.ctx, e.t, nil, 10, 10)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 3 || len(got) != 0 {
			t.Fatalf("total=%d len=%d, want 3/0", total, len(got))
		}
	})

	t.Run("rows carry every column", func(t *testing.T) {
		et := models.EntityType("deal")
		got, _, err := e.repo.List(e.ctx, e.t, &et, 0, 10)
		if err != nil || len(got) != 1 {
			t.Fatalf("List: rows=%d err=%v", len(got), err)
		}
		tg := got[0]
		if tg.ID == uuid.Nil || tg.TenantID != e.t || tg.Name != "Aaa-Deal" || tg.Color != "#3d8abf" || tg.CreatedAt.IsZero() {
			t.Fatalf("incomplete row: %+v", tg)
		}
	})

	t.Run("a tenant never lists another tenant's tags", func(t *testing.T) {
		// Own ctx, foreign tenantID: the explicit predicate and RLS both exclude the rows.
		got, total, err := other.repo.List(other.ctx, e.t, nil, 0, 50)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 0 || len(got) != 0 {
			t.Fatalf("foreign tenant sees total=%d rows=%v", total, names(got))
		}
	})

	t.Run("empty tenant", func(t *testing.T) {
		empty := newTagDBEnv(t)
		got, total, err := empty.repo.List(empty.ctx, empty.t, nil, 0, 50)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 0 || len(got) != 0 {
			t.Fatalf("total=%d len=%d, want 0/0", total, len(got))
		}
	})

	t.Run("canceled context is an error, not an empty page", func(t *testing.T) {
		ctx, cancel := context.WithCancel(e.ctx)
		cancel()
		if _, _, err := e.repo.List(ctx, e.t, nil, 0, 50); err == nil {
			t.Fatal("List with canceled ctx: expected error")
		}
		et := models.EntityType("contact")
		if _, _, err := e.repo.List(ctx, e.t, &et, 0, 50); err == nil {
			t.Fatal("filtered List with canceled ctx: expected error")
		}
	})
}

func TestPostgresRepository_GetByEntityAndName(t *testing.T) {
	e := newTagDBEnv(t)
	other := newTagDBEnv(t)
	other.seedTag(t, models.EntityType("contact"), "VIP")

	seeded := e.seedTag(t, models.EntityType("contact"), "VIP")

	t.Run("match is case-insensitive", func(t *testing.T) {
		got, err := e.repo.GetByEntityAndName(e.ctx, e.t, models.EntityType("contact"), "vIp")
		if err != nil {
			t.Fatalf("GetByEntityAndName: %v", err)
		}
		if got.ID != seeded.ID || got.Name != "VIP" {
			t.Fatalf("got %+v, want tag %s", got, seeded.ID)
		}
	})

	t.Run("other entity type is not found", func(t *testing.T) {
		_, err := e.repo.GetByEntityAndName(e.ctx, e.t, models.EntityType("deal"), "VIP")
		if !errors.Is(err, ErrTagNotFound) {
			t.Fatalf("err = %v, want ErrTagNotFound", err)
		}
	})

	t.Run("unknown name is not found", func(t *testing.T) {
		_, err := e.repo.GetByEntityAndName(e.ctx, e.t, models.EntityType("contact"), "Nope")
		if !errors.Is(err, ErrTagNotFound) {
			t.Fatalf("err = %v, want ErrTagNotFound", err)
		}
	})

	t.Run("same name in another tenant resolves to that tenant's own tag", func(t *testing.T) {
		got, err := other.repo.GetByEntityAndName(other.ctx, other.t, models.EntityType("contact"), "VIP")
		if err != nil {
			t.Fatalf("GetByEntityAndName: %v", err)
		}
		if got.ID == seeded.ID || got.TenantID != other.t {
			t.Fatalf("leaked tag %+v", got)
		}
	})

	t.Run("foreign tenant id with own context finds nothing", func(t *testing.T) {
		_, err := other.repo.GetByEntityAndName(other.ctx, e.t, models.EntityType("contact"), "VIP")
		if !errors.Is(err, ErrTagNotFound) {
			t.Fatalf("err = %v, want ErrTagNotFound", err)
		}
	})

	t.Run("canceled context surfaces the driver error, not ErrTagNotFound", func(t *testing.T) {
		ctx, cancel := context.WithCancel(e.ctx)
		cancel()
		_, err := e.repo.GetByEntityAndName(ctx, e.t, models.EntityType("contact"), "VIP")
		if err == nil || errors.Is(err, ErrTagNotFound) {
			t.Fatalf("err = %v, want a non-NotFound error", err)
		}
	})
}

func TestPostgresRepository_GetByID(t *testing.T) {
	e := newTagDBEnv(t)
	other := newTagDBEnv(t)
	seeded := e.seedTag(t, models.EntityType("company"), "Key Account")

	got, err := e.repo.GetByID(e.ctx, seeded.ID, e.t)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Key Account" || got.EntityType != models.EntityType("company") || got.Color != "#3d8abf" || got.TenantID != e.t {
		t.Fatalf("got %+v", got)
	}

	if _, err := e.repo.GetByID(e.ctx, uuid.New(), e.t); !errors.Is(err, ErrTagNotFound) {
		t.Fatalf("unknown id: err = %v, want ErrTagNotFound", err)
	}
	if _, err := other.repo.GetByID(other.ctx, seeded.ID, other.t); !errors.Is(err, ErrTagNotFound) {
		t.Fatalf("foreign tenant: err = %v, want ErrTagNotFound", err)
	}
	// Right row, wrong tenant predicate: the explicit tenant filter alone must reject it.
	if _, err := e.repo.GetByID(e.ctx, seeded.ID, other.t); !errors.Is(err, ErrTagNotFound) {
		t.Fatalf("wrong tenant predicate: err = %v, want ErrTagNotFound", err)
	}

	ctx, cancel := context.WithCancel(e.ctx)
	cancel()
	if _, err := e.repo.GetByID(ctx, seeded.ID, e.t); err == nil || errors.Is(err, ErrTagNotFound) {
		t.Fatalf("canceled ctx: err = %v, want a non-NotFound error", err)
	}
}

// TestPostgresRepository_IsInUse links one tag to a parent row through each of the four
// junction tables in turn and expects IsInUse to flip exactly then.
func TestPostgresRepository_IsInUse(t *testing.T) {
	e := newTagDBEnv(t)
	other := newTagDBEnv(t)

	userID := testutil.SeedRow(t, e.pool, "users", map[string]any{
		"id":            uuid.New(),
		"email":         fmt.Sprintf("tag-inuse-%s@test.invalid", uuid.New()),
		"password_hash": "x",
		"tenant_id":     e.t,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, e.pool, "users", userID) })
	stageID := testutil.SeedRow(t, e.pool, "pipeline_stages", map[string]any{
		"id": uuid.New(), "tenant_id": e.t, "name": "Stage-" + uuid.NewString()[:8],
	})
	t.Cleanup(func() { testutil.CleanupRow(t, e.pool, "pipeline_stages", stageID) })

	contactID := testutil.SeedRow(t, e.pool, "contacts", map[string]any{
		"id": uuid.New(), "tenant_id": e.t, "first_name": "Ada", "last_name": "Tag", "created_by": userID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, e.pool, "contacts", contactID) })
	companyID := testutil.SeedRow(t, e.pool, "companies", map[string]any{
		"id": uuid.New(), "tenant_id": e.t, "name": "Tag GmbH", "created_by": userID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, e.pool, "companies", companyID) })
	dealID := testutil.SeedRow(t, e.pool, "deals", map[string]any{
		"id": uuid.New(), "tenant_id": e.t, "name": "Tag Deal", "stage_id": stageID, "created_by": userID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, e.pool, "deals", dealID) })
	activityID := testutil.SeedRow(t, e.pool, "activities", map[string]any{
		"id": uuid.New(), "tenant_id": e.t, "activity_type": "note", "subject": "Tag Note", "created_by": userID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, e.pool, "activities", activityID) })

	cases := []struct {
		table, column string
		parent        uuid.UUID
	}{
		{"contact_tags", "contact_id", contactID},
		{"company_tags", "company_id", companyID},
		{"deal_tags", "deal_id", dealID},
		{"activity_tags", "activity_id", activityID},
	}

	for _, tc := range cases {
		t.Run(tc.table, func(t *testing.T) {
			tg := e.seedTag(t, models.EntityType("contact"), "InUse-"+tc.table)

			inUse, err := e.repo.IsInUse(e.ctx, tg.ID, e.t)
			if err != nil {
				t.Fatalf("IsInUse (unused): %v", err)
			}
			if inUse {
				t.Fatal("unused tag reported as in use")
			}

			if _, err := e.pool.Exec(e.sys,
				fmt.Sprintf(`INSERT INTO %s (%s, tag_id, tenant_id) VALUES ($1, $2, $3)`, tc.table, tc.column),
				tc.parent, tg.ID, e.t,
			); err != nil {
				t.Fatalf("link tag via %s: %v", tc.table, err)
			}

			inUse, err = e.repo.IsInUse(e.ctx, tg.ID, e.t)
			if err != nil {
				t.Fatalf("IsInUse (linked): %v", err)
			}
			if !inUse {
				t.Fatalf("tag linked via %s not reported as in use", tc.table)
			}

			// Another tenant asking about this tag id must not see the link.
			inUse, err = other.repo.IsInUse(other.ctx, tg.ID, other.t)
			if err != nil {
				t.Fatalf("IsInUse (foreign tenant): %v", err)
			}
			if inUse {
				t.Fatal("foreign tenant sees the link")
			}
		})
	}

	t.Run("an unknown tag is not in use", func(t *testing.T) {
		inUse, err := e.repo.IsInUse(e.ctx, uuid.New(), e.t)
		if err != nil || inUse {
			t.Fatalf("inUse=%v err=%v, want false/nil", inUse, err)
		}
	})

	t.Run("canceled context is an error, not false", func(t *testing.T) {
		ctx, cancel := context.WithCancel(e.ctx)
		cancel()
		inUse, err := e.repo.IsInUse(ctx, uuid.New(), e.t)
		if err == nil || inUse {
			t.Fatalf("inUse=%v err=%v, want false/error", inUse, err)
		}
	})
}
