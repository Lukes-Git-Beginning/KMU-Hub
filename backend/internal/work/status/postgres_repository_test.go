package status_test

// Repository coverage against the app-role database (kmuhub_app, RLS in force).
//
// project_statuses carries no explicit `tenant_id = $n` predicate on the read
// side; isolation rests on the RLS policy alone. The tests therefore run under
// the tenant context production uses and assert that a second tenant sees and
// changes nothing.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
	"github.com/kmuhub/kmuhub/internal/work/status"
)

type statusFixture struct {
	tenant  uuid.UUID
	user    uuid.UUID
	project uuid.UUID
}

func newStatusRepo(t *testing.T) (*status.PostgresRepository, *pgxpool.Pool) {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	// Registered first so LIFO closes the pool after the fixture cleanups.
	t.Cleanup(func() { pool.Close() })
	return status.NewPostgresRepository(pool), pool
}

func seedStatusProject(t *testing.T, pool *pgxpool.Pool, tenant, user uuid.UUID, name string) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "projects", map[string]any{
		"tenant_id":   tenant,
		"name":        name,
		"project_key": fmt.Sprintf("S%08d", uuid.New().ID()%100000000),
		"created_by":  user,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "projects", id) })
	return id
}

func seedStatusFixture(t *testing.T, pool *pgxpool.Pool) statusFixture {
	t.Helper()
	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Status Repo Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenant) })
	user := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenant,
		"email":         fmt.Sprintf("wp-status-%s@test.invalid", uuid.New()),
		"password_hash": "x",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", user) })
	return statusFixture{tenant: tenant, user: user, project: seedStatusProject(t, pool, tenant, user, "Status Project")}
}

func newStatus(projectID uuid.UUID, name string, order int) *models.ProjectStatus {
	color := "#336699"
	return &models.ProjectStatus{
		ID: uuid.New(), ProjectID: projectID, Name: name, Color: &color,
		SortOrder: order, CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
}

func createStatus(t *testing.T, repo *status.PostgresRepository, ctx context.Context, s *models.ProjectStatus) {
	t.Helper()
	require.NoError(t, repo.Create(ctx, s))
}

func TestStatusRepo_CreateRoundTripsAndStampsProjectTenant(t *testing.T) {
	repo, pool := newStatusRepo(t)
	fx := seedStatusFixture(t, pool)
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	s := newStatus(fx.project, "In Review", 3)
	s.IsDefault, s.IsClosed = true, true
	createStatus(t, repo, ctx, s)

	got, err := repo.GetByID(ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, fx.project, got.ProjectID)
	assert.Equal(t, "In Review", got.Name)
	require.NotNil(t, got.Color)
	assert.Equal(t, "#336699", *got.Color)
	assert.Equal(t, 3, got.SortOrder)
	assert.True(t, got.IsDefault)
	assert.True(t, got.IsClosed)
	assert.True(t, got.CreatedAt.Equal(s.CreatedAt))

	// tenant_id is derived from the project, not supplied by the caller.
	var tenant uuid.UUID
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT tenant_id FROM project_statuses WHERE id = $1`, s.ID).Scan(&tenant))
	assert.Equal(t, fx.tenant, tenant)
}

func TestStatusRepo_CreateWithNilColorStoresNull(t *testing.T) {
	repo, pool := newStatusRepo(t)
	fx := seedStatusFixture(t, pool)
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	s := newStatus(fx.project, "No Colour", 0)
	s.Color = nil
	createStatus(t, repo, ctx, s)

	got, err := repo.GetByID(ctx, s.ID)
	require.NoError(t, err)
	assert.Nil(t, got.Color)
}

func TestStatusRepo_GetByIDMissingIsNotFound(t *testing.T) {
	repo, pool := newStatusRepo(t)
	fx := seedStatusFixture(t, pool)
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	_, err := repo.GetByID(ctx, uuid.New())
	assert.ErrorIs(t, err, status.ErrNotFound)
}

func TestStatusRepo_ListByProjectOrdersBySortOrderAndScopesToProject(t *testing.T) {
	repo, pool := newStatusRepo(t)
	fx := seedStatusFixture(t, pool)
	other := seedStatusProject(t, pool, fx.tenant, fx.user, "Other Project")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	createStatus(t, repo, ctx, newStatus(fx.project, "Third", 2))
	createStatus(t, repo, ctx, newStatus(fx.project, "First", 0))
	createStatus(t, repo, ctx, newStatus(fx.project, "Second", 1))
	createStatus(t, repo, ctx, newStatus(other, "Elsewhere", 0))

	got, err := repo.ListByProject(ctx, fx.project)
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, []string{"First", "Second", "Third"}, []string{got[0].Name, got[1].Name, got[2].Name})

	empty, err := repo.ListByProject(ctx, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestStatusRepo_UpdatePersistsMutableFieldsOnly(t *testing.T) {
	repo, pool := newStatusRepo(t)
	fx := seedStatusFixture(t, pool)
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	s := newStatus(fx.project, "Open", 4)
	createStatus(t, repo, ctx, s)

	newColor := "#ff0000"
	s.Name, s.Color, s.IsDefault, s.IsClosed = "Renamed", &newColor, true, true
	s.SortOrder = 99 // Update must not touch the position; Reorder owns it.
	require.NoError(t, repo.Update(ctx, s))

	got, err := repo.GetByID(ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, "Renamed", got.Name)
	require.NotNil(t, got.Color)
	assert.Equal(t, "#ff0000", *got.Color)
	assert.True(t, got.IsDefault)
	assert.True(t, got.IsClosed)
	assert.Equal(t, 4, got.SortOrder)
}

func TestStatusRepo_DeleteRemovesRow(t *testing.T) {
	repo, pool := newStatusRepo(t)
	fx := seedStatusFixture(t, pool)
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	s := newStatus(fx.project, "Doomed", 0)
	createStatus(t, repo, ctx, s)

	require.NoError(t, repo.Delete(ctx, s.ID))
	_, err := repo.GetByID(ctx, s.ID)
	assert.ErrorIs(t, err, status.ErrNotFound)
}

func TestStatusRepo_ReorderAssignsPositionsInGivenOrderWithinProject(t *testing.T) {
	repo, pool := newStatusRepo(t)
	fx := seedStatusFixture(t, pool)
	other := seedStatusProject(t, pool, fx.tenant, fx.user, "Other Project")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	a, b, c := newStatus(fx.project, "A", 0), newStatus(fx.project, "B", 1), newStatus(fx.project, "C", 2)
	foreign := newStatus(other, "Foreign", 7)
	for _, s := range []*models.ProjectStatus{a, b, c, foreign} {
		createStatus(t, repo, ctx, s)
	}

	// The foreign status is passed on purpose: the project_id predicate must keep it out.
	require.NoError(t, repo.Reorder(ctx, fx.project, []uuid.UUID{c.ID, a.ID, b.ID, foreign.ID}))

	got, err := repo.ListByProject(ctx, fx.project)
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, []string{"C", "A", "B"}, []string{got[0].Name, got[1].Name, got[2].Name})
	assert.Equal(t, []int{0, 1, 2}, []int{got[0].SortOrder, got[1].SortOrder, got[2].SortOrder})

	untouched, err := repo.GetByID(ctx, foreign.ID)
	require.NoError(t, err)
	assert.Equal(t, 7, untouched.SortOrder)
}

func TestStatusRepo_CountTasksWithStatus(t *testing.T) {
	repo, pool := newStatusRepo(t)
	fx := seedStatusFixture(t, pool)
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	s := newStatus(fx.project, "Busy", 0)
	createStatus(t, repo, ctx, s)

	n, err := repo.CountTasksWithStatus(ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, n)

	elsewhere := newStatus(fx.project, "Elsewhere", 1)
	createStatus(t, repo, ctx, elsewhere)
	seedTask := func(title string, statusID any) {
		id := testutil.SeedRow(t, pool, "tasks", map[string]any{
			"tenant_id":   fx.tenant,
			"title":       title,
			"task_number": int(uuid.New().ID())%900000 + 100000,
			"created_by":  fx.user,
			"project_id":  fx.project,
			"status_id":   statusID,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "tasks", id) })
	}
	seedTask("Task 0", s.ID)
	seedTask("Task 1", s.ID)
	seedTask("Other status", elsewhere.ID)
	seedTask("No status", nil)

	n, err = repo.CountTasksWithStatus(ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
}

func TestStatusRepo_HasDefault(t *testing.T) {
	repo, pool := newStatusRepo(t)
	fx := seedStatusFixture(t, pool)
	other := seedStatusProject(t, pool, fx.tenant, fx.user, "Other Project")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	otherDefault := newStatus(other, "Other Default", 0)
	otherDefault.IsDefault = true
	createStatus(t, repo, ctx, otherDefault)

	has, err := repo.HasDefault(ctx, fx.project)
	require.NoError(t, err)
	assert.False(t, has, "another project's default must not count")

	createStatus(t, repo, ctx, newStatus(fx.project, "Plain", 0))
	has, err = repo.HasDefault(ctx, fx.project)
	require.NoError(t, err)
	assert.False(t, has, "a non-default status must not count")

	def := newStatus(fx.project, "Default", 1)
	def.IsDefault = true
	createStatus(t, repo, ctx, def)
	has, err = repo.HasDefault(ctx, fx.project)
	require.NoError(t, err)
	assert.True(t, has)
}

func TestStatusRepo_GetNextSortOrder(t *testing.T) {
	repo, pool := newStatusRepo(t)
	fx := seedStatusFixture(t, pool)
	other := seedStatusProject(t, pool, fx.tenant, fx.user, "Other Project")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	createStatus(t, repo, ctx, newStatus(other, "Far Away", 40))

	next, err := repo.GetNextSortOrder(ctx, fx.project)
	require.NoError(t, err)
	assert.Equal(t, 0, next, "an empty project starts at 0")

	createStatus(t, repo, ctx, newStatus(fx.project, "A", 0))
	createStatus(t, repo, ctx, newStatus(fx.project, "B", 5))

	next, err = repo.GetNextSortOrder(ctx, fx.project)
	require.NoError(t, err)
	assert.Equal(t, 6, next)
}

func TestStatusRepo_NameExistsInProject(t *testing.T) {
	repo, pool := newStatusRepo(t)
	fx := seedStatusFixture(t, pool)
	other := seedStatusProject(t, pool, fx.tenant, fx.user, "Other Project")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	s := newStatus(fx.project, "Backlog", 0)
	createStatus(t, repo, ctx, s)

	exists, err := repo.NameExistsInProject(ctx, fx.project, "backlog", nil)
	require.NoError(t, err)
	assert.True(t, exists, "comparison is case-insensitive")

	exists, err = repo.NameExistsInProject(ctx, fx.project, "Backlog", &s.ID)
	require.NoError(t, err)
	assert.False(t, exists, "a status never collides with itself")

	exists, err = repo.NameExistsInProject(ctx, fx.project, "BACKLOG", &s.ID)
	require.NoError(t, err)
	assert.False(t, exists, "self-exclusion holds regardless of case")

	otherID := uuid.New()
	exists, err = repo.NameExistsInProject(ctx, fx.project, "Backlog", &otherID)
	require.NoError(t, err)
	assert.True(t, exists, "excluding a different id keeps the collision")

	exists, err = repo.NameExistsInProject(ctx, fx.project, "BACKLOG", &otherID)
	require.NoError(t, err)
	assert.True(t, exists, "the exclusion branch is case-insensitive too")

	exists, err = repo.NameExistsInProject(ctx, fx.project, "Nope", nil)
	require.NoError(t, err)
	assert.False(t, exists)

	exists, err = repo.NameExistsInProject(ctx, other, "Backlog", nil)
	require.NoError(t, err)
	assert.False(t, exists, "names are scoped per project")
}

func TestStatusRepo_CopyStatusesForProjectClonesWithTargetTenant(t *testing.T) {
	repo, pool := newStatusRepo(t)
	fx := seedStatusFixture(t, pool)
	target := seedStatusProject(t, pool, fx.tenant, fx.user, "Target Project")
	ctx := testutil.WithTenantCtx(context.Background(), fx.tenant)

	first, second := newStatus(fx.project, "Todo", 0), newStatus(fx.project, "Done", 1)
	first.IsDefault, second.IsClosed = true, true
	createStatus(t, repo, ctx, first)
	createStatus(t, repo, ctx, second)

	require.NoError(t, repo.CopyStatusesForProject(ctx, fx.project, target))

	got, err := repo.ListByProject(ctx, target)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "Todo", got[0].Name)
	assert.True(t, got[0].IsDefault)
	assert.Equal(t, "Done", got[1].Name)
	assert.True(t, got[1].IsClosed)
	assert.NotEqual(t, first.ID, got[0].ID, "copies get fresh ids")
	assert.Equal(t, target, got[0].ProjectID)

	var tenants int
	require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT COUNT(*) FROM project_statuses WHERE project_id = $1 AND tenant_id = $2`,
		target, fx.tenant).Scan(&tenants))
	assert.Equal(t, 2, tenants)

	// The source stays untouched.
	src, err := repo.ListByProject(ctx, fx.project)
	require.NoError(t, err)
	assert.Len(t, src, 2)
}

// A template copied under the system context (no RLS) must still land in the
// target project's tenant, not the source's.
func TestStatusRepo_CopyStatusesForProjectTakesTenantFromTarget(t *testing.T) {
	repo, pool := newStatusRepo(t)
	src := seedStatusFixture(t, pool)
	dst := seedStatusFixture(t, pool)
	sysCtx := testutil.WithSystemCtx(context.Background())

	createStatus(t, repo, sysCtx, newStatus(src.project, "Template", 0))
	require.NoError(t, repo.CopyStatusesForProject(sysCtx, src.project, dst.project))

	var tenant uuid.UUID
	require.NoError(t, pool.QueryRow(sysCtx,
		`SELECT tenant_id FROM project_statuses WHERE project_id = $1`, dst.project).Scan(&tenant))
	assert.Equal(t, dst.tenant, tenant)
}

func TestStatusRepo_OtherTenantSeesAndChangesNothing(t *testing.T) {
	repo, pool := newStatusRepo(t)
	fx := seedStatusFixture(t, pool)
	foreignTenant := uuid.New()
	testutil.EnsureTenant(t, pool, foreignTenant, "Status Repo Foreign Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", foreignTenant) })

	own := testutil.WithTenantCtx(context.Background(), fx.tenant)
	foreign := testutil.WithTenantCtx(context.Background(), foreignTenant)

	s := newStatus(fx.project, "Secret", 0)
	s.IsDefault = true
	createStatus(t, repo, own, s)

	_, err := repo.GetByID(foreign, s.ID)
	assert.ErrorIs(t, err, status.ErrNotFound)

	list, err := repo.ListByProject(foreign, fx.project)
	require.NoError(t, err)
	assert.Empty(t, list)

	has, err := repo.HasDefault(foreign, fx.project)
	require.NoError(t, err)
	assert.False(t, has)

	exists, err := repo.NameExistsInProject(foreign, fx.project, "Secret", nil)
	require.NoError(t, err)
	assert.False(t, exists)

	// Writes under the foreign tenant must match no row.
	s.Name = "Hijacked"
	require.NoError(t, repo.Update(foreign, s))
	require.NoError(t, repo.Delete(foreign, s.ID))

	got, err := repo.GetByID(own, s.ID)
	require.NoError(t, err)
	assert.Equal(t, "Secret", got.Name)
}

func TestStatusRepo_ClosedPoolSurfacesErrors(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	repo := status.NewPostgresRepository(pool)
	pool.Close()
	ctx := context.Background()
	id := uuid.New()

	assert.Error(t, repo.Create(ctx, newStatus(id, "x", 0)))
	_, err := repo.GetByID(ctx, id)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, status.ErrNotFound, "a connection failure is not a missing row")
	_, err = repo.ListByProject(ctx, id)
	assert.Error(t, err)
	assert.Error(t, repo.Update(ctx, newStatus(id, "x", 0)))
	assert.Error(t, repo.Delete(ctx, id))
	assert.Error(t, repo.Reorder(ctx, id, []uuid.UUID{id}))
	_, err = repo.CountTasksWithStatus(ctx, id)
	assert.Error(t, err)
	_, err = repo.HasDefault(ctx, id)
	assert.Error(t, err)
	_, err = repo.GetNextSortOrder(ctx, id)
	assert.Error(t, err)
	_, err = repo.NameExistsInProject(ctx, id, "x", nil)
	assert.Error(t, err)
	other := uuid.New()
	_, err = repo.NameExistsInProject(ctx, id, "x", &other)
	assert.Error(t, err)
	assert.Error(t, repo.CopyStatusesForProject(ctx, id, other))
}
