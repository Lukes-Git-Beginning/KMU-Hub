package customfield

// Covers the read paths (GetByEntityAndName, List) and the options/default
// round trip of the Postgres repository against the app-role database.
// Write-side RLS lives in tenant_write_test.go.

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

type cfFixture struct {
	pool   *pgxpool.Pool
	repo   *PostgresRepository
	tenant uuid.UUID
	other  uuid.UUID
	user   uuid.UUID
	ctx    context.Context
	ctxOth context.Context
}

func newCFFixture(t *testing.T) *cfFixture {
	t.Helper()
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenant, other := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "CustomField Read Tenant")
	testutil.EnsureTenant(t, pool, other, "CustomField Read Other Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", other) })
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenant) })

	user := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenant,
		"email":         fmt.Sprintf("customfield-read-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", user) })

	return &cfFixture{
		pool:   pool,
		repo:   NewPostgresRepository(pool),
		tenant: tenant,
		other:  other,
		user:   user,
		ctx:    testutil.WithTenantCtx(context.Background(), tenant),
		ctxOth: testutil.WithTenantCtx(context.Background(), other),
	}
}

// create inserts a definition for tenantID via the repository and schedules cleanup.
func (f *cfFixture) create(t *testing.T, tenantID uuid.UUID, entity models.EntityType, name string, sortOrder int, createdAt time.Time) *models.CustomFieldDefinition {
	t.Helper()
	ctx := f.ctx
	if tenantID == f.other {
		ctx = f.ctxOth
	}
	field := &models.CustomFieldDefinition{
		ID:         uuid.New(),
		TenantID:   tenantID,
		EntityType: entity,
		FieldName:  name,
		FieldLabel: "Label " + name,
		FieldType:  models.FieldType("text"),
		SortOrder:  sortOrder,
		CreatedBy:  f.user,
		CreatedAt:  createdAt,
		UpdatedAt:  createdAt,
	}
	if err := f.repo.Create(ctx, field); err != nil {
		t.Fatalf("Create %s: %v", name, err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "custom_field_definitions", field.ID) })
	return field
}

func fieldNames(fields []*models.CustomFieldDefinition) []string {
	names := make([]string, 0, len(fields))
	for _, f := range fields {
		names = append(names, f.FieldName)
	}
	return names
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestPostgresRepository_GetByEntityAndName(t *testing.T) {
	t.Parallel()
	f := newCFFixture(t)
	now := time.Now().UTC().Truncate(time.Microsecond)

	own := f.create(t, f.tenant, "contact", "branche", 3, now)
	// Same entity+name in another tenant must not collide and must not be returned.
	foreign := f.create(t, f.other, "contact", "branche", 1, now)

	got, err := f.repo.GetByEntityAndName(f.ctx, f.tenant, "contact", "branche")
	if err != nil {
		t.Fatalf("GetByEntityAndName: %v", err)
	}
	if got.ID != own.ID || got.ID == foreign.ID {
		t.Fatalf("returned wrong definition: got %s, own %s, foreign %s", got.ID, own.ID, foreign.ID)
	}
	if got.SortOrder != 3 || got.TenantID != f.tenant {
		t.Fatalf("unexpected row: sort=%d tenant=%s", got.SortOrder, got.TenantID)
	}

	t.Run("other entity type is not found", func(t *testing.T) {
		if _, err := f.repo.GetByEntityAndName(f.ctx, f.tenant, "deal", "branche"); !errors.Is(err, ErrFieldNotFound) {
			t.Fatalf("expected ErrFieldNotFound, got %v", err)
		}
	})
	t.Run("other name is not found", func(t *testing.T) {
		if _, err := f.repo.GetByEntityAndName(f.ctx, f.tenant, "contact", "other"); !errors.Is(err, ErrFieldNotFound) {
			t.Fatalf("expected ErrFieldNotFound, got %v", err)
		}
	})
	t.Run("foreign tenant id in a foreign session is invisible", func(t *testing.T) {
		if _, err := f.repo.GetByEntityAndName(f.ctxOth, f.tenant, "contact", "branche"); !errors.Is(err, ErrFieldNotFound) {
			t.Fatalf("expected ErrFieldNotFound, got %v", err)
		}
	})
	t.Run("tenant predicate alone separates the same name", func(t *testing.T) {
		got, err := f.repo.GetByEntityAndName(f.ctxOth, f.other, "contact", "branche")
		if err != nil {
			t.Fatalf("GetByEntityAndName (other): %v", err)
		}
		if got.ID != foreign.ID {
			t.Fatalf("other tenant got %s, want %s", got.ID, foreign.ID)
		}
	})
	t.Run("cancelled context is an error, not not-found", func(t *testing.T) {
		ctx, cancel := context.WithCancel(f.ctx)
		cancel()
		_, err := f.repo.GetByEntityAndName(ctx, f.tenant, "contact", "branche")
		if err == nil || errors.Is(err, ErrFieldNotFound) {
			t.Fatalf("expected a non-not-found error, got %v", err)
		}
	})
}

func TestPostgresRepository_List(t *testing.T) {
	t.Parallel()
	f := newCFFixture(t)
	base := time.Now().UTC().Truncate(time.Microsecond)

	// Insertion order differs from sort order on purpose.
	c2 := f.create(t, f.tenant, "contact", "c_second", 2, base)
	c1b := f.create(t, f.tenant, "contact", "c_first_b", 1, base.Add(2*time.Second))
	c1a := f.create(t, f.tenant, "contact", "c_first_a", 1, base.Add(1*time.Second))
	d1 := f.create(t, f.tenant, "deal", "d_first", 0, base)
	co1 := f.create(t, f.tenant, "company", "co_first", 5, base)
	f.create(t, f.other, "contact", "foreign", 0, base)

	t.Run("filtered by entity type sorts by sort_order then created_at", func(t *testing.T) {
		et := models.EntityType("contact")
		fields, total, err := f.repo.List(f.ctx, f.tenant, &et, 0, 50)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 3 {
			t.Fatalf("total = %d, want 3", total)
		}
		want := []string{c1a.FieldName, c1b.FieldName, c2.FieldName}
		if got := fieldNames(fields); !equalStrings(got, want) {
			t.Fatalf("order = %v, want %v", got, want)
		}
	})

	t.Run("unfiltered sorts by entity_type first", func(t *testing.T) {
		fields, total, err := f.repo.List(f.ctx, f.tenant, nil, 0, 50)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 5 {
			t.Fatalf("total = %d, want 5 (foreign tenant must not count)", total)
		}
		// company < contact < deal alphabetically, even though co_first has the largest sort_order.
		want := []string{co1.FieldName, c1a.FieldName, c1b.FieldName, c2.FieldName, d1.FieldName}
		if got := fieldNames(fields); !equalStrings(got, want) {
			t.Fatalf("order = %v, want %v", got, want)
		}
	})

	t.Run("paging keeps the full total", func(t *testing.T) {
		et := models.EntityType("contact")
		fields, total, err := f.repo.List(f.ctx, f.tenant, &et, 1, 1)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 3 || len(fields) != 1 || fields[0].ID != c1b.ID {
			t.Fatalf("filtered page: total=%d names=%v", total, fieldNames(fields))
		}

		fields, total, err = f.repo.List(f.ctx, f.tenant, nil, 3, 2)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		want := []string{c2.FieldName, d1.FieldName}
		if total != 5 || !equalStrings(fieldNames(fields), want) {
			t.Fatalf("unfiltered page: total=%d names=%v", total, fieldNames(fields))
		}
	})

	t.Run("offset behind the end is empty with the real total", func(t *testing.T) {
		fields, total, err := f.repo.List(f.ctx, f.tenant, nil, 10, 10)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(fields) != 0 || total != 5 {
			t.Fatalf("got %d fields, total %d", len(fields), total)
		}
	})

	t.Run("foreign tenant sees only its own", func(t *testing.T) {
		fields, total, err := f.repo.List(f.ctxOth, f.other, nil, 0, 50)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 1 || len(fields) != 1 || fields[0].FieldName != "foreign" {
			t.Fatalf("foreign list: total=%d names=%v", total, fieldNames(fields))
		}
	})

	t.Run("tenant argument is filtered even in a foreign session", func(t *testing.T) {
		fields, total, err := f.repo.List(f.ctxOth, f.tenant, nil, 0, 50)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if total != 0 || len(fields) != 0 {
			t.Fatalf("leak: total=%d names=%v", total, fieldNames(fields))
		}
	})

	t.Run("cancelled context fails both branches", func(t *testing.T) {
		ctx, cancel := context.WithCancel(f.ctx)
		cancel()
		et := models.EntityType("contact")
		if _, _, err := f.repo.List(ctx, f.tenant, &et, 0, 10); err == nil {
			t.Fatal("filtered: expected an error")
		}
		if _, _, err := f.repo.List(ctx, f.tenant, nil, 0, 10); err == nil {
			t.Fatal("unfiltered: expected an error")
		}
	})
}

func TestPostgresRepository_OptionsAndDefaultRoundTrip(t *testing.T) {
	t.Parallel()
	f := newCFFixture(t)
	now := time.Now().UTC().Truncate(time.Microsecond)

	def := "mittel"
	field := &models.CustomFieldDefinition{
		ID:           uuid.New(),
		TenantID:     f.tenant,
		EntityType:   "deal",
		FieldName:    "prio",
		FieldLabel:   "Priorität",
		FieldType:    models.FieldType("select"),
		Options:      []string{"niedrig", "mittel", "hoch"},
		DefaultValue: &def,
		IsRequired:   true,
		SortOrder:    7,
		CreatedBy:    f.user,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := f.repo.Create(f.ctx, field); err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "custom_field_definitions", field.ID) })

	check := func(t *testing.T, got *models.CustomFieldDefinition, label string, opts []string, dflt *string, required bool, sort int) {
		t.Helper()
		if got.FieldLabel != label || got.IsRequired != required || got.SortOrder != sort {
			t.Fatalf("scalar columns: %+v", got)
		}
		if !equalStrings(got.Options, opts) {
			t.Fatalf("options = %v, want %v", got.Options, opts)
		}
		switch {
		case dflt == nil && got.DefaultValue != nil:
			t.Fatalf("default = %q, want nil", *got.DefaultValue)
		case dflt != nil && (got.DefaultValue == nil || *got.DefaultValue != *dflt):
			t.Fatalf("default = %v, want %q", got.DefaultValue, *dflt)
		}
	}

	got, err := f.repo.GetByID(f.ctx, field.ID, f.tenant)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	check(t, got, "Priorität", field.Options, &def, true, 7)
	if got.FieldName != "prio" || got.FieldType != "select" || got.EntityType != "deal" || got.CreatedBy != f.user {
		t.Fatalf("identity columns: %+v", got)
	}
	if !got.CreatedAt.Equal(now) {
		t.Fatalf("created_at = %v, want %v", got.CreatedAt, now)
	}

	// Update writes only the editable columns and leaves identity alone.
	newDef := "hoch"
	later := now.Add(time.Hour)
	upd := *field
	upd.FieldLabel = "Dringlichkeit"
	upd.Options = []string{"a", "b"}
	upd.DefaultValue = &newDef
	upd.IsRequired = false
	upd.SortOrder = 1
	upd.UpdatedAt = later
	upd.FieldName = "must_not_change"
	upd.EntityType = "contact"
	if err := f.repo.Update(f.ctx, &upd); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err = f.repo.GetByEntityAndName(f.ctx, f.tenant, "deal", "prio")
	if err != nil {
		t.Fatalf("GetByEntityAndName after update: %v", err)
	}
	check(t, got, "Dringlichkeit", []string{"a", "b"}, &newDef, false, 1)
	if !got.UpdatedAt.Equal(later) || !got.CreatedAt.Equal(now) {
		t.Fatalf("timestamps: created=%v updated=%v", got.CreatedAt, got.UpdatedAt)
	}

	// A field without options and default comes back with neither.
	plain := f.create(t, f.tenant, "activity", "plain", 0, now)
	got, err = f.repo.GetByID(f.ctx, plain.ID, f.tenant)
	if err != nil {
		t.Fatalf("GetByID plain: %v", err)
	}
	check(t, got, "Label plain", nil, nil, false, 0)

	t.Run("update of an unknown id is not-found", func(t *testing.T) {
		ghost := upd
		ghost.ID = uuid.New()
		if err := f.repo.Update(f.ctx, &ghost); !errors.Is(err, ErrFieldNotFound) {
			t.Fatalf("expected ErrFieldNotFound, got %v", err)
		}
	})
	t.Run("delete of an unknown id is not-found", func(t *testing.T) {
		if err := f.repo.Delete(f.ctx, uuid.New(), f.tenant); !errors.Is(err, ErrFieldNotFound) {
			t.Fatalf("expected ErrFieldNotFound, got %v", err)
		}
	})
	t.Run("explicit tenant predicate holds when RLS is out of the way", func(t *testing.T) {
		// System context bypasses RLS, so only the WHERE tenant_id clause can refuse.
		sysCtx := testutil.WithSystemCtx(context.Background())
		foreign := upd
		foreign.TenantID = f.other
		foreign.FieldLabel = "Hacked"
		if err := f.repo.Update(sysCtx, &foreign); !errors.Is(err, ErrFieldNotFound) {
			t.Fatalf("Update: expected ErrFieldNotFound, got %v", err)
		}
		if err := f.repo.Delete(sysCtx, field.ID, f.other); !errors.Is(err, ErrFieldNotFound) {
			t.Fatalf("Delete: expected ErrFieldNotFound, got %v", err)
		}
		if _, err := f.repo.GetByID(sysCtx, field.ID, f.other); !errors.Is(err, ErrFieldNotFound) {
			t.Fatalf("GetByID: expected ErrFieldNotFound, got %v", err)
		}
		got, err := f.repo.GetByID(f.ctx, field.ID, f.tenant)
		if err != nil || got.FieldLabel != "Dringlichkeit" {
			t.Fatalf("definition was touched: %+v, %v", got, err)
		}
	})
	t.Run("duplicate entity and name in one tenant is rejected", func(t *testing.T) {
		dup := *field
		dup.ID = uuid.New()
		dup.FieldName = "prio"
		dup.EntityType = "deal"
		if err := f.repo.Create(f.ctx, &dup); err == nil {
			t.Fatal("expected a unique violation")
		}
	})

	// Runs last: it corrupts the options column of `plain`.
	t.Run("corrupt options column surfaces an error", func(t *testing.T) {
		if _, err := f.pool.Exec(testutil.WithSystemCtx(context.Background()),
			`UPDATE custom_field_definitions SET options = '{"a":1}'::jsonb WHERE id = $1`, plain.ID); err != nil {
			t.Fatalf("corrupt options: %v", err)
		}
		if _, err := f.repo.GetByID(f.ctx, plain.ID, f.tenant); err == nil || errors.Is(err, ErrFieldNotFound) {
			t.Fatalf("GetByID: expected an unmarshal error, got %v", err)
		}
		et := models.EntityType("activity")
		if _, _, err := f.repo.List(f.ctx, f.tenant, &et, 0, 10); err == nil {
			t.Fatal("List: expected an unmarshal error")
		}
	})
}
