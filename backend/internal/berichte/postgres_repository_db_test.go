package berichte

// DB-backed tests for the postgres_repository.go paths that carry no
// aggregation (this module has none — every SUM/AVG lives in
// internal/berichte/executor and internal/berichte/downstream, both already
// DB-tested) but were still 0% covered: definition/schedule listing with real
// filters and pagination, the cache CRUD path, and the run/schedule-run
// bookkeeping the scheduler depends on. Bug hypothesis for this pass: a
// missing tenant_id predicate on a read, or a filter/pagination/sort bug that
// silently returns the wrong slice.

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedDefinition(t *testing.T, repo *PostgresRepository, ctx context.Context, tenantID uuid.UUID, name, module, kind string, published bool) *Definition {
	t.Helper()
	now := time.Now().UTC()
	def := &Definition{
		ID:            uuid.New(),
		TenantID:      tenantID,
		Name:          name,
		Module:        module,
		Kind:          kind,
		QueryConfig:   []byte(`{}`),
		DefaultFormat: "pdf",
		IsPublished:   published,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repo.CreateDefinition(ctx, def); err != nil {
		t.Fatalf("seed definition %q: %v", name, err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, repo.pool, "report_definitions", def.ID) })
	return def
}

// TestPostgresListDefinitions_FiltersPaginationSortAndTenantScope covers the
// dynamic WHERE/ORDER BY builder in ListDefinitions end to end: each filter in
// isolation, combined pagination + sort, and that a foreign tenant's rows
// never leak into either the page or the total count.
func TestPostgresListDefinitions_FiltersPaginationSortAndTenantScope(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant, other := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Berichte List Def Tenant")
	testutil.EnsureTenant(t, pool, other, "Berichte List Def Other Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	otherCtx := testutil.WithTenantCtx(context.Background(), other)

	seedDefinition(t, repo, ctx, tenant, "Alpha Umsatz", "finanzen", "custom", true)
	seedDefinition(t, repo, ctx, tenant, "Beta Pipeline", "crm", "system", false)
	seedDefinition(t, repo, ctx, tenant, "Gamma Tickets", "helpdesk", "custom", true)
	// A same-named row in another tenant must not inflate this tenant's count
	// or leak into a Search hit.
	seedDefinition(t, repo, otherCtx, other, "Alpha Umsatz", "finanzen", "custom", true)

	// Module filter.
	mod := "crm"
	rows, total, err := repo.ListDefinitions(ctx, tenant, ListDefinitionsFilter{Module: &mod}, 0, 20)
	if err != nil {
		t.Fatalf("ListDefinitions (module filter): %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].Name != "Beta Pipeline" {
		t.Fatalf("module filter: got %d rows (total %d), want 1 (Beta Pipeline)", len(rows), total)
	}

	// IsPublished filter.
	published := true
	rows, total, err = repo.ListDefinitions(ctx, tenant, ListDefinitionsFilter{IsPublished: &published}, 0, 20)
	if err != nil {
		t.Fatalf("ListDefinitions (published filter): %v", err)
	}
	if total != 2 {
		t.Fatalf("published filter: total = %d, want 2 (Alpha, Gamma)", total)
	}
	for _, d := range rows {
		if !d.IsPublished {
			t.Fatalf("published filter returned an unpublished definition: %s", d.Name)
		}
	}

	// Kind filter.
	kind := "system"
	rows, total, err = repo.ListDefinitions(ctx, tenant, ListDefinitionsFilter{Kind: &kind}, 0, 20)
	if err != nil {
		t.Fatalf("ListDefinitions (kind filter): %v", err)
	}
	if total != 1 || rows[0].Name != "Beta Pipeline" {
		t.Fatalf("kind filter: got total %d, want 1 (Beta Pipeline)", total)
	}

	// Search is case-insensitive and must not match the other tenant's row.
	rows, total, err = repo.ListDefinitions(ctx, tenant, ListDefinitionsFilter{Search: "alpha"}, 0, 20)
	if err != nil {
		t.Fatalf("ListDefinitions (search): %v", err)
	}
	if total != 1 || rows[0].TenantID != tenant {
		t.Fatalf("search: got total %d, want 1 scoped to own tenant", total)
	}

	// Sort by name ascending vs descending.
	rows, _, err = repo.ListDefinitions(ctx, tenant, ListDefinitionsFilter{SortBy: "name", SortDesc: false}, 0, 20)
	if err != nil {
		t.Fatalf("ListDefinitions (sort asc): %v", err)
	}
	if len(rows) != 3 || rows[0].Name != "Alpha Umsatz" || rows[2].Name != "Gamma Tickets" {
		t.Fatalf("sort asc by name: unexpected order %v", namesOf(rows))
	}
	rows, _, err = repo.ListDefinitions(ctx, tenant, ListDefinitionsFilter{SortBy: "name", SortDesc: true}, 0, 20)
	if err != nil {
		t.Fatalf("ListDefinitions (sort desc): %v", err)
	}
	if len(rows) != 3 || rows[0].Name != "Gamma Tickets" || rows[2].Name != "Alpha Umsatz" {
		t.Fatalf("sort desc by name: unexpected order %v", namesOf(rows))
	}

	// Pagination: limit 2 offset 1, sorted by name, must return the middle and
	// last row while total still reports the tenant's full set.
	rows, total, err = repo.ListDefinitions(ctx, tenant, ListDefinitionsFilter{SortBy: "name"}, 1, 2)
	if err != nil {
		t.Fatalf("ListDefinitions (paged): %v", err)
	}
	if total != 3 {
		t.Fatalf("paged total = %d, want 3 (total ignores offset/limit)", total)
	}
	if len(rows) != 2 || rows[0].Name != "Beta Pipeline" || rows[1].Name != "Gamma Tickets" {
		t.Fatalf("paged rows: unexpected page %v", namesOf(rows))
	}

	// Tenant scope: the foreign tenant's own list must not see this tenant's rows.
	_, otherTotal, err := repo.ListDefinitions(otherCtx, other, ListDefinitionsFilter{}, 0, 20)
	if err != nil {
		t.Fatalf("ListDefinitions (other tenant): %v", err)
	}
	if otherTotal != 1 {
		t.Fatalf("other tenant total = %d, want 1 (only its own Alpha Umsatz)", otherTotal)
	}
}

func namesOf(defs []*Definition) []string {
	out := make([]string, len(defs))
	for i, d := range defs {
		out[i] = d.Name
	}
	return out
}

// TestPostgresGetUpdateDeleteDefinition_TenantScoped proves the single-row
// definition mutations are bounded by tenant_id, not just by id — a foreign
// tenant's Get/Update/Delete against a real id must report ErrDefinitionNotFound
// rather than silently reaching or mutating the row.
func TestPostgresGetUpdateDeleteDefinition_TenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant, other := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Berichte Def Mutate Tenant")
	testutil.EnsureTenant(t, pool, other, "Berichte Def Mutate Other Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	otherCtx := testutil.WithTenantCtx(context.Background(), other)

	def := seedDefinition(t, repo, ctx, tenant, "Mutate Me", "finanzen", "custom", false)

	got, err := repo.GetDefinition(ctx, tenant, def.ID)
	if err != nil {
		t.Fatalf("GetDefinition (own tenant): %v", err)
	}
	if got.Name != "Mutate Me" {
		t.Fatalf("GetDefinition: name = %q", got.Name)
	}
	if _, err := repo.GetDefinition(otherCtx, tenant, def.ID); err != ErrDefinitionNotFound {
		t.Fatalf("GetDefinition (foreign ctx): got %v, want ErrDefinitionNotFound", err)
	}

	// Update under the foreign tenant must not touch the row.
	stolen := *def
	stolen.Name = "Stolen"
	if err := repo.UpdateDefinition(otherCtx, &stolen); err != ErrDefinitionNotFound {
		t.Fatalf("UpdateDefinition (foreign ctx): got %v, want ErrDefinitionNotFound", err)
	}
	unchanged, err := repo.GetDefinition(ctx, tenant, def.ID)
	if err != nil {
		t.Fatalf("GetDefinition (post foreign update attempt): %v", err)
	}
	if unchanged.Name != "Mutate Me" {
		t.Fatalf("UpdateDefinition (foreign ctx) mutated the row: name = %q", unchanged.Name)
	}

	// Update under the owning tenant succeeds.
	def.Name = "Mutated"
	def.IsPublished = true
	if err := repo.UpdateDefinition(ctx, def); err != nil {
		t.Fatalf("UpdateDefinition (own tenant): %v", err)
	}
	updated, err := repo.GetDefinition(ctx, tenant, def.ID)
	if err != nil {
		t.Fatalf("GetDefinition (after update): %v", err)
	}
	if updated.Name != "Mutated" || !updated.IsPublished {
		t.Fatalf("UpdateDefinition: fields not persisted, got %+v", updated)
	}

	// Delete under the foreign tenant is a no-op, not a cross-tenant delete.
	if err := repo.DeleteDefinition(otherCtx, tenant, def.ID); err != ErrDefinitionNotFound {
		t.Fatalf("DeleteDefinition (foreign ctx): got %v, want ErrDefinitionNotFound", err)
	}
	if _, err := repo.GetDefinition(ctx, tenant, def.ID); err != nil {
		t.Fatalf("definition vanished after a foreign-tenant delete attempt: %v", err)
	}

	if err := repo.DeleteDefinition(ctx, tenant, def.ID); err != nil {
		t.Fatalf("DeleteDefinition (own tenant): %v", err)
	}
	if _, err := repo.GetDefinition(ctx, tenant, def.ID); err != ErrDefinitionNotFound {
		t.Fatalf("GetDefinition (after delete): got %v, want ErrDefinitionNotFound", err)
	}
}

// TestPostgresCacheEntry_CRUDAndExpiry exercises Get/Upsert/Invalidate plus
// DeleteExpiredCacheEntries's cross-tenant sweep — it intentionally has no
// tenant_id predicate (it is a maintenance sweep, not a tenant read), so this
// proves it purges only entries past `before` and leaves fresh ones and
// other-tenant fresh ones alone.
func TestPostgresCacheEntry_CRUDAndExpiry(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant, other := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Berichte Cache Tenant")
	testutil.EnsureTenant(t, pool, other, "Berichte Cache Other Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	otherCtx := testutil.WithTenantCtx(context.Background(), other)
	sysCtx := testutil.WithSystemCtx(context.Background())

	def := seedDefinition(t, repo, ctx, tenant, "Cache Def", "finanzen", "custom", false)
	otherDef := seedDefinition(t, repo, otherCtx, other, "Cache Def Other", "finanzen", "custom", false)

	now := time.Now().UTC()

	// Miss before any entry exists.
	if _, err := repo.GetCacheEntry(ctx, tenant, def.ID, "hash-a"); err != ErrCacheMiss {
		t.Fatalf("GetCacheEntry (no entry): got %v, want ErrCacheMiss", err)
	}

	fresh := &CacheEntry{
		ID: uuid.New(), TenantID: tenant, DefinitionID: def.ID, ParamsHash: "hash-a",
		Result: []byte(`{"rows":[]}`), RowCount: 0, ComputedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	if err := repo.UpsertCacheEntry(ctx, fresh); err != nil {
		t.Fatalf("UpsertCacheEntry (insert): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_cache", fresh.ID) })

	got, err := repo.GetCacheEntry(ctx, tenant, def.ID, "hash-a")
	if err != nil {
		t.Fatalf("GetCacheEntry (hit): %v", err)
	}
	// jsonb round-trips through Postgres's own canonical text form (a space
	// after ':'), so compare parsed structure rather than raw bytes.
	var resultRows map[string]any
	if err := json.Unmarshal(got.Result, &resultRows); err != nil {
		t.Fatalf("GetCacheEntry: result is not valid JSON: %v", err)
	}
	if got.RowCount != 0 || resultRows["rows"] == nil {
		t.Fatalf("GetCacheEntry: unexpected payload %+v", got)
	}

	// Upsert on the same (definition_id, params_hash) updates in place rather
	// than duplicating — the ON CONFLICT target.
	updatedPayload := &CacheEntry{
		ID: uuid.New(), TenantID: tenant, DefinitionID: def.ID, ParamsHash: "hash-a",
		Result: []byte(`{"rows":[{"x":1}]}`), RowCount: 1, ComputedAt: now, ExpiresAt: now.Add(2 * time.Hour),
	}
	if err := repo.UpsertCacheEntry(ctx, updatedPayload); err != nil {
		t.Fatalf("UpsertCacheEntry (update): %v", err)
	}
	got, err = repo.GetCacheEntry(ctx, tenant, def.ID, "hash-a")
	if err != nil {
		t.Fatalf("GetCacheEntry (after upsert-update): %v", err)
	}
	if got.RowCount != 1 {
		t.Fatalf("GetCacheEntry (after upsert-update): row_count = %d, want 1 (ON CONFLICT should have replaced, not duplicated)", got.RowCount)
	}

	// A soon-to-expire entry for the cross-tenant sweep below.
	expiring := &CacheEntry{
		ID: uuid.New(), TenantID: tenant, DefinitionID: def.ID, ParamsHash: "hash-expiring",
		Result: []byte(`{}`), RowCount: 0, ComputedAt: now, ExpiresAt: now.Add(-time.Minute),
	}
	if err := repo.UpsertCacheEntry(ctx, expiring); err != nil {
		t.Fatalf("UpsertCacheEntry (expiring): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_cache", expiring.ID) })

	otherExpiring := &CacheEntry{
		ID: uuid.New(), TenantID: other, DefinitionID: otherDef.ID, ParamsHash: "hash-other-expiring",
		Result: []byte(`{}`), RowCount: 0, ComputedAt: now, ExpiresAt: now.Add(-time.Minute),
	}
	if err := repo.UpsertCacheEntry(otherCtx, otherExpiring); err != nil {
		t.Fatalf("UpsertCacheEntry (other tenant expiring): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_cache", otherExpiring.ID) })

	evicted, err := repo.DeleteExpiredCacheEntries(sysCtx, now)
	if err != nil {
		t.Fatalf("DeleteExpiredCacheEntries: %v", err)
	}
	if evicted < 2 {
		t.Fatalf("DeleteExpiredCacheEntries: evicted %d, want at least 2 (both tenants' expiring rows)", evicted)
	}
	// The still-fresh entry must survive the sweep.
	if _, err := repo.GetCacheEntry(ctx, tenant, def.ID, "hash-a"); err != nil {
		t.Fatalf("GetCacheEntry (fresh entry after sweep): %v", err)
	}
	if _, err := repo.GetCacheEntry(ctx, tenant, def.ID, "hash-expiring"); err != ErrCacheMiss {
		t.Fatalf("GetCacheEntry (expired entry after sweep): got %v, want ErrCacheMiss", err)
	}

	evictedByInvalidate, err := repo.InvalidateCache(ctx, tenant, def.ID)
	if err != nil {
		t.Fatalf("InvalidateCache: %v", err)
	}
	if evictedByInvalidate != 1 {
		t.Fatalf("InvalidateCache: evicted %d, want 1 (only the still-fresh hash-a entry)", evictedByInvalidate)
	}
	if _, err := repo.GetCacheEntry(ctx, tenant, def.ID, "hash-a"); err != ErrCacheMiss {
		t.Fatalf("GetCacheEntry (after InvalidateCache): got %v, want ErrCacheMiss", err)
	}
}

// TestPostgresListSchedules_FiltersPaginationAndTenantScope covers the
// DefinitionID/Active filters and pagination on ListSchedules, and that a
// foreign tenant's schedules never appear in either the page or the count.
func TestPostgresListSchedules_FiltersPaginationAndTenantScope(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant, other := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Berichte List Sch Tenant")
	testutil.EnsureTenant(t, pool, other, "Berichte List Sch Other Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	otherCtx := testutil.WithTenantCtx(context.Background(), other)

	defA := seedDefinition(t, repo, ctx, tenant, "Def A", "finanzen", "custom", false)
	defB := seedDefinition(t, repo, ctx, tenant, "Def B", "crm", "custom", false)
	otherDef := seedDefinition(t, repo, otherCtx, other, "Other Def", "finanzen", "custom", false)

	now := time.Now().UTC()
	mkSchedule := func(defID uuid.UUID, tenantID uuid.UUID, active bool, name string, c context.Context) *Schedule {
		sch := &Schedule{
			ID: uuid.New(), TenantID: tenantID, DefinitionID: defID, Name: name,
			CronExpression: "0 8 * * *", Recipients: []string{}, Params: []byte(`{}`), Format: "pdf", Active: active, CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.CreateSchedule(c, sch); err != nil {
			t.Fatalf("seed schedule %q: %v", name, err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_schedules", sch.ID) })
		return sch
	}

	schA1 := mkSchedule(defA.ID, tenant, true, "A active", ctx)
	mkSchedule(defA.ID, tenant, false, "A inactive", ctx)
	mkSchedule(defB.ID, tenant, true, "B active", ctx)
	mkSchedule(otherDef.ID, other, true, "Other active", otherCtx)

	// DefinitionID filter.
	rows, total, err := repo.ListSchedules(ctx, tenant, ListSchedulesFilter{DefinitionID: &defA.ID}, 0, 20)
	if err != nil {
		t.Fatalf("ListSchedules (definition filter): %v", err)
	}
	if total != 2 {
		t.Fatalf("definition filter: total = %d, want 2", total)
	}
	for _, s := range rows {
		if s.DefinitionID != defA.ID {
			t.Fatalf("definition filter leaked schedule for a different definition: %+v", s)
		}
	}

	// Active filter.
	active := true
	rows, total, err = repo.ListSchedules(ctx, tenant, ListSchedulesFilter{Active: &active}, 0, 20)
	if err != nil {
		t.Fatalf("ListSchedules (active filter): %v", err)
	}
	if total != 2 {
		t.Fatalf("active filter: total = %d, want 2 (A active, B active)", total)
	}
	for _, s := range rows {
		if !s.Active {
			t.Fatalf("active filter returned an inactive schedule: %+v", s)
		}
	}

	// Pagination: 3 schedules total for this tenant, limit 2.
	rows, total, err = repo.ListSchedules(ctx, tenant, ListSchedulesFilter{}, 0, 2)
	if err != nil {
		t.Fatalf("ListSchedules (paged): %v", err)
	}
	if total != 3 {
		t.Fatalf("paged total = %d, want 3", total)
	}
	if len(rows) != 2 {
		t.Fatalf("paged rows = %d, want 2 (limit)", len(rows))
	}

	// Tenant scope.
	_, otherTotal, err := repo.ListSchedules(otherCtx, other, ListSchedulesFilter{}, 0, 20)
	if err != nil {
		t.Fatalf("ListSchedules (other tenant): %v", err)
	}
	if otherTotal != 1 {
		t.Fatalf("other tenant total = %d, want 1", otherTotal)
	}
	if _, err := repo.GetSchedule(ctx, tenant, schA1.ID); err != nil {
		t.Fatalf("sanity GetSchedule (own tenant): %v", err)
	}
}

// TestPostgresUpdateDeleteSchedule_TenantScoped mirrors the definition
// tenant-scope test for the schedule mutations.
func TestPostgresUpdateDeleteSchedule_TenantScoped(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant, other := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Berichte Sch Mutate Tenant")
	testutil.EnsureTenant(t, pool, other, "Berichte Sch Mutate Other Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	otherCtx := testutil.WithTenantCtx(context.Background(), other)

	def := seedDefinition(t, repo, ctx, tenant, "Sch Mutate Def", "finanzen", "custom", false)
	now := time.Now().UTC()
	sch := &Schedule{
		ID: uuid.New(), TenantID: tenant, DefinitionID: def.ID, Name: "Original",
		CronExpression: "0 8 * * *", Recipients: []string{}, Params: []byte(`{}`), Format: "pdf", Active: false, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateSchedule(ctx, sch); err != nil {
		t.Fatalf("CreateSchedule: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_schedules", sch.ID) })

	stolen := *sch
	stolen.Name = "Stolen"
	if err := repo.UpdateSchedule(otherCtx, &stolen); err != ErrScheduleNotFound {
		t.Fatalf("UpdateSchedule (foreign ctx): got %v, want ErrScheduleNotFound", err)
	}
	if err := repo.DeleteSchedule(otherCtx, tenant, sch.ID); err != ErrScheduleNotFound {
		t.Fatalf("DeleteSchedule (foreign ctx): got %v, want ErrScheduleNotFound", err)
	}
	unchanged, err := repo.GetSchedule(ctx, tenant, sch.ID)
	if err != nil {
		t.Fatalf("GetSchedule (post foreign attempts): %v", err)
	}
	if unchanged.Name != "Original" {
		t.Fatalf("a foreign-tenant mutation attempt changed the row: name = %q", unchanged.Name)
	}

	sch.Name = "Renamed"
	sch.Active = true
	if err := repo.UpdateSchedule(ctx, sch); err != nil {
		t.Fatalf("UpdateSchedule (own tenant): %v", err)
	}
	updated, err := repo.GetSchedule(ctx, tenant, sch.ID)
	if err != nil {
		t.Fatalf("GetSchedule (after update): %v", err)
	}
	if updated.Name != "Renamed" || !updated.Active {
		t.Fatalf("UpdateSchedule: fields not persisted, got %+v", updated)
	}

	if err := repo.DeleteSchedule(ctx, tenant, sch.ID); err != nil {
		t.Fatalf("DeleteSchedule (own tenant): %v", err)
	}
	if _, err := repo.GetSchedule(ctx, tenant, sch.ID); err != ErrScheduleNotFound {
		t.Fatalf("GetSchedule (after delete): got %v, want ErrScheduleNotFound", err)
	}
}

// TestPostgresListDueSchedules_ReturnsOnlyActive proves ListDueSchedules'
// active=TRUE predicate: the scheduler polls this across every tenant (it has
// no tenant filter by design — see the repo's doc comment), so this pins that
// an inactive schedule never comes back regardless of tenant.
func TestPostgresListDueSchedules_ReturnsOnlyActive(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Berichte Due Sch Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	sysCtx := testutil.WithSystemCtx(context.Background())

	def := seedDefinition(t, repo, ctx, tenant, "Due Def", "finanzen", "custom", false)
	now := time.Now().UTC()

	active := &Schedule{
		ID: uuid.New(), TenantID: tenant, DefinitionID: def.ID, Name: "Due Active " + uuid.New().String(),
		CronExpression: "0 8 * * *", Recipients: []string{}, Params: []byte(`{}`), Format: "pdf", Active: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateSchedule(ctx, active); err != nil {
		t.Fatalf("CreateSchedule (active): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_schedules", active.ID) })

	inactive := &Schedule{
		ID: uuid.New(), TenantID: tenant, DefinitionID: def.ID, Name: "Due Inactive " + uuid.New().String(),
		CronExpression: "0 8 * * *", Recipients: []string{}, Params: []byte(`{}`), Format: "pdf", Active: false, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateSchedule(ctx, inactive); err != nil {
		t.Fatalf("CreateSchedule (inactive): %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_schedules", inactive.ID) })

	due, err := repo.ListDueSchedules(sysCtx, now)
	if err != nil {
		t.Fatalf("ListDueSchedules: %v", err)
	}
	var sawActive, sawInactive bool
	for _, s := range due {
		if s.ID == active.ID {
			sawActive = true
		}
		if s.ID == inactive.ID {
			sawInactive = true
		}
	}
	if !sawActive {
		t.Fatal("ListDueSchedules did not return the active schedule")
	}
	if sawInactive {
		t.Fatal("ListDueSchedules returned an inactive schedule")
	}
}

// TestPostgresUpdateScheduleLastRun_PersistsStatusAndError covers the write
// the scheduler makes after every dispatch, including that a nil error
// pointer clears a previously stored error rather than leaving it stale.
func TestPostgresUpdateScheduleLastRun_PersistsStatusAndError(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Berichte LastRun Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	def := seedDefinition(t, repo, ctx, tenant, "LastRun Def", "finanzen", "custom", false)
	// Postgres timestamptz has microsecond precision; truncate so the
	// round-tripped value compares equal instead of failing on the
	// nanosecond remainder Go's clock carries.
	now := time.Now().UTC().Truncate(time.Microsecond)
	sch := &Schedule{
		ID: uuid.New(), TenantID: tenant, DefinitionID: def.ID, Name: "LastRun Sch",
		CronExpression: "0 8 * * *", Recipients: []string{}, Params: []byte(`{}`), Format: "pdf", Active: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.CreateSchedule(ctx, sch); err != nil {
		t.Fatalf("CreateSchedule: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_schedules", sch.ID) })

	runErr := "smtp: connection refused"
	firstRun := now.Add(time.Minute)
	if err := repo.UpdateScheduleLastRun(ctx, sch.ID, "failed", &runErr, firstRun); err != nil {
		t.Fatalf("UpdateScheduleLastRun (failed): %v", err)
	}
	got, err := repo.GetSchedule(ctx, tenant, sch.ID)
	if err != nil {
		t.Fatalf("GetSchedule: %v", err)
	}
	if got.LastRunStatus == nil || *got.LastRunStatus != "failed" {
		t.Fatalf("last_run_status = %v, want failed", got.LastRunStatus)
	}
	if got.LastRunError == nil || *got.LastRunError != runErr {
		t.Fatalf("last_run_error = %v, want %q", got.LastRunError, runErr)
	}
	if got.LastRunAt == nil || !got.LastRunAt.Equal(firstRun) {
		t.Fatalf("last_run_at = %v, want %v", got.LastRunAt, firstRun)
	}

	secondRun := now.Add(2 * time.Minute)
	if err := repo.UpdateScheduleLastRun(ctx, sch.ID, "success", nil, secondRun); err != nil {
		t.Fatalf("UpdateScheduleLastRun (success): %v", err)
	}
	got, err = repo.GetSchedule(ctx, tenant, sch.ID)
	if err != nil {
		t.Fatalf("GetSchedule (after success): %v", err)
	}
	if got.LastRunStatus == nil || *got.LastRunStatus != "success" {
		t.Fatalf("last_run_status = %v, want success", got.LastRunStatus)
	}
	if got.LastRunError != nil {
		t.Fatalf("last_run_error = %v, want nil (a successful run must clear a stale failure)", *got.LastRunError)
	}
}

// TestPostgresInsertRun_PersistsFullRecord covers the audit-trail write:
// every RunReport call inserts one report_runs row regardless of outcome.
func TestPostgresInsertRun_PersistsFullRecord(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Berichte Run Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	def := seedDefinition(t, repo, ctx, tenant, "Run Def", "finanzen", "custom", false)
	now := time.Now().UTC()
	duration := 128
	rowCount := 7
	errStr := "downstream timeout"

	run := &Run{
		ID: uuid.New(), TenantID: tenant, DefinitionID: def.ID, Trigger: "manual",
		Params: []byte(`{}`), DurationMs: &duration, RowCount: &rowCount,
		Status: "failed", Error: &errStr, StartedAt: now, CompletedAt: &now,
	}
	if err := repo.InsertRun(ctx, run); err != nil {
		t.Fatalf("InsertRun: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_runs", run.ID) })

	testutil.AssertRowCount(t, pool, ctx, "report_runs", run.ID, 1)
	otherTenantCtx := testutil.WithTenantCtx(context.Background(), uuid.New())
	testutil.AssertRowCount(t, pool, otherTenantCtx, "report_runs", run.ID, 0)
}

// TestPostgresListDocuments_FiltersPaginationAndTenantScope covers the
// Module/Status/Search filters and pagination on ListDocuments, mirroring the
// definition-listing test above.
func TestPostgresListDocuments_FiltersPaginationAndTenantScope(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant, other := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Berichte List Doc Tenant")
	testutil.EnsureTenant(t, pool, other, "Berichte List Doc Other Tenant")

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	otherCtx := testutil.WithTenantCtx(context.Background(), other)

	now := time.Now().UTC()
	mkDoc := func(title, module, status string, tenantID uuid.UUID, c context.Context) *Document {
		doc := &Document{
			ID: uuid.New(), TenantID: tenantID, Title: title, Module: module, Status: status,
			Rows: []byte(`[]`), Settings: []byte(`{}`), CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.CreateDocument(c, doc); err != nil {
			t.Fatalf("seed document %q: %v", title, err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "report_documents", doc.ID) })
		return doc
	}

	mkDoc("Quartalsbericht Q1", "finanzen", "draft", tenant, ctx)
	mkDoc("Pipeline Review", "crm", "final", tenant, ctx)
	mkDoc("Quartalsbericht Q2", "finanzen", "released", tenant, ctx)
	mkDoc("Quartalsbericht Q1", "finanzen", "draft", other, otherCtx)

	mod := "finanzen"
	rows, total, err := repo.ListDocuments(ctx, tenant, ListDocumentsFilter{Module: &mod}, 0, 20)
	if err != nil {
		t.Fatalf("ListDocuments (module filter): %v", err)
	}
	if total != 2 {
		t.Fatalf("module filter: total = %d, want 2", total)
	}
	for _, d := range rows {
		if d.Module != "finanzen" {
			t.Fatalf("module filter leaked a document from another module: %+v", d)
		}
	}

	status := "final"
	rows, total, err = repo.ListDocuments(ctx, tenant, ListDocumentsFilter{Status: &status}, 0, 20)
	if err != nil {
		t.Fatalf("ListDocuments (status filter): %v", err)
	}
	if total != 1 || rows[0].Title != "Pipeline Review" {
		t.Fatalf("status filter: got total %d, want 1 (Pipeline Review)", total)
	}

	rows, total, err = repo.ListDocuments(ctx, tenant, ListDocumentsFilter{Search: "quartalsbericht"}, 0, 20)
	if err != nil {
		t.Fatalf("ListDocuments (search): %v", err)
	}
	if total != 2 {
		t.Fatalf("search: total = %d, want 2 (own tenant's two Quartalsbericht docs, not the other tenant's)", total)
	}
	for _, d := range rows {
		if d.TenantID != tenant {
			t.Fatalf("search leaked a foreign-tenant document: %+v", d)
		}
	}

	rows, total, err = repo.ListDocuments(ctx, tenant, ListDocumentsFilter{}, 0, 2)
	if err != nil {
		t.Fatalf("ListDocuments (paged): %v", err)
	}
	if total != 3 {
		t.Fatalf("paged total = %d, want 3", total)
	}
	if len(rows) != 2 {
		t.Fatalf("paged rows = %d, want 2 (limit)", len(rows))
	}

	_, otherTotal, err := repo.ListDocuments(otherCtx, other, ListDocumentsFilter{}, 0, 20)
	if err != nil {
		t.Fatalf("ListDocuments (other tenant): %v", err)
	}
	if otherTotal != 1 {
		t.Fatalf("other tenant total = %d, want 1", otherTotal)
	}
}
