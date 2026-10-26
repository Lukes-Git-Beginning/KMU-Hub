package label_test

// postgres_repository_crud_test.go covers the remaining repository paths
// (Create/Update/List/Delete errors, batch loading, assignment replacement)
// against the real schema as kmuhub_app, with RLS enforced.

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
	"github.com/kmuhub/kmuhub/internal/work/label"
)

type labelFixture struct {
	pool   *pgxpool.Pool
	repo   *label.PostgresRepository
	tenant uuid.UUID
	user   uuid.UUID
	ctx    context.Context
}

func newLabelFixture(t *testing.T) *labelFixture {
	t.Helper()
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Label CRUD Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenant) })

	return &labelFixture{
		pool:   pool,
		repo:   label.NewPostgresRepository(pool),
		tenant: tenant,
		user:   seedLabelUser(t, pool, tenant),
		ctx:    testutil.WithTenantCtx(context.Background(), tenant),
	}
}

func (f *labelFixture) mustCreate(t *testing.T, name, color string) *label.Label {
	t.Helper()
	l, err := f.repo.Create(f.ctx, f.tenant.String(), name, color)
	require.NoError(t, err)
	return l
}

func (f *labelFixture) task(t *testing.T) string {
	t.Helper()
	return seedLabelTask(t, f.pool, f.tenant, f.user).String()
}

func (f *labelFixture) otherTenant(t *testing.T, name string) uuid.UUID {
	t.Helper()
	other := uuid.New()
	testutil.EnsureTenant(t, f.pool, other, name)
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "tenants", other) })
	return other
}

func TestLabelRepo_CreateRoundtrip(t *testing.T) {
	f := newLabelFixture(t)

	l := f.mustCreate(t, "Bug", "#ef4444")
	assert.NotEmpty(t, l.ID)
	assert.Equal(t, f.tenant.String(), l.TenantID)
	assert.Equal(t, "Bug", l.Name)
	assert.Equal(t, "#ef4444", l.Color)
	assert.False(t, l.CreatedAt.IsZero())
	assert.Equal(t, l.CreatedAt, l.UpdatedAt)

	got, err := f.repo.GetByID(f.ctx, f.tenant.String(), l.ID)
	require.NoError(t, err)
	assert.Equal(t, l.ID, got.ID)
	assert.Equal(t, "Bug", got.Name)
	assert.Equal(t, "#ef4444", got.Color)
}

func TestLabelRepo_CreateDuplicateNameInTenant(t *testing.T) {
	f := newLabelFixture(t)
	f.mustCreate(t, "Dupe", "#111111")

	_, err := f.repo.Create(f.ctx, f.tenant.String(), "Dupe", "#222222")
	assert.ErrorIs(t, err, label.ErrDuplicateName)
}

func TestLabelRepo_SameNameInOtherTenantIsAllowed(t *testing.T) {
	f := newLabelFixture(t)
	other := f.otherTenant(t, "Label Other Tenant")
	otherCtx := testutil.WithTenantCtx(context.Background(), other)

	f.mustCreate(t, "Shared", "#111111")
	_, err := f.repo.Create(otherCtx, other.String(), "Shared", "#111111")
	assert.NoError(t, err)
}

func TestLabelRepo_CreateRejectsForeignTenant(t *testing.T) {
	f := newLabelFixture(t)
	other := f.otherTenant(t, "Label Foreign Tenant")

	// RLS WITH CHECK refuses a row for a tenant that is not the caller's;
	// the error is not a duplicate, so it must surface unmapped.
	_, err := f.repo.Create(f.ctx, other.String(), "Smuggled", "#111111")
	require.Error(t, err)
	assert.False(t, errors.Is(err, label.ErrDuplicateName))
}

func TestLabelRepo_GetByIDNotFound(t *testing.T) {
	f := newLabelFixture(t)

	_, err := f.repo.GetByID(f.ctx, f.tenant.String(), uuid.NewString())
	assert.ErrorIs(t, err, label.ErrNotFound)
}

func TestLabelRepo_ListOrderedByNameAndScoped(t *testing.T) {
	f := newLabelFixture(t)
	f.mustCreate(t, "charlie", "#111111")
	f.mustCreate(t, "alpha", "#111111")
	f.mustCreate(t, "bravo", "#111111")

	got, err := f.repo.List(f.ctx, f.tenant.String())
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, []string{"alpha", "bravo", "charlie"}, []string{got[0].Name, got[1].Name, got[2].Name})

	// A tenant without labels gets an empty result and no error.
	empty := f.otherTenant(t, "Label Empty Tenant")
	none, err := f.repo.List(testutil.WithTenantCtx(context.Background(), empty), empty.String())
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestLabelRepo_Update(t *testing.T) {
	f := newLabelFixture(t)
	l := f.mustCreate(t, "Before", "#111111")

	got, err := f.repo.Update(f.ctx, f.tenant.String(), l.ID, "After", "#222222")
	require.NoError(t, err)
	assert.Equal(t, l.ID, got.ID)
	assert.Equal(t, "After", got.Name)
	assert.Equal(t, "#222222", got.Color)
	assert.Equal(t, l.CreatedAt, got.CreatedAt)
	assert.True(t, got.UpdatedAt.After(l.UpdatedAt))

	persisted, err := f.repo.GetByID(f.ctx, f.tenant.String(), l.ID)
	require.NoError(t, err)
	assert.Equal(t, "After", persisted.Name)
}

func TestLabelRepo_UpdateNotFound(t *testing.T) {
	f := newLabelFixture(t)

	_, err := f.repo.Update(f.ctx, f.tenant.String(), uuid.NewString(), "X", "#222222")
	assert.ErrorIs(t, err, label.ErrNotFound)
}

func TestLabelRepo_UpdateDuplicateName(t *testing.T) {
	f := newLabelFixture(t)
	f.mustCreate(t, "Taken", "#111111")
	l := f.mustCreate(t, "Mine", "#111111")

	_, err := f.repo.Update(f.ctx, f.tenant.String(), l.ID, "Taken", "#111111")
	assert.ErrorIs(t, err, label.ErrDuplicateName)
}

func TestLabelRepo_DeleteAndNotFound(t *testing.T) {
	f := newLabelFixture(t)
	l := f.mustCreate(t, "Doomed", "#111111")

	require.NoError(t, f.repo.Delete(f.ctx, f.tenant.String(), l.ID))
	_, err := f.repo.GetByID(f.ctx, f.tenant.String(), l.ID)
	assert.ErrorIs(t, err, label.ErrNotFound)

	assert.ErrorIs(t, f.repo.Delete(f.ctx, f.tenant.String(), l.ID), label.ErrNotFound)
}

func TestLabelRepo_GetLabelsByTaskIDs(t *testing.T) {
	f := newLabelFixture(t)
	tid := f.tenant.String()
	taskA, taskB, taskC := f.task(t), f.task(t), f.task(t)
	red := f.mustCreate(t, "red", "#ff0000")
	blue := f.mustCreate(t, "blue", "#0000ff")

	require.NoError(t, f.repo.SetTaskLabels(f.ctx, tid, taskA, []string{red.ID, blue.ID}))
	require.NoError(t, f.repo.SetTaskLabels(f.ctx, tid, taskB, []string{red.ID}))

	got, err := f.repo.GetLabelsByTaskIDs(f.ctx, tid, []string{taskA, taskB, taskC})
	require.NoError(t, err)
	require.Len(t, got, 2, "task without labels has no entry")
	require.Len(t, got[taskA], 2)
	assert.Equal(t, "blue", got[taskA][0].Name, "ordered by name within a task")
	assert.Equal(t, "red", got[taskA][1].Name)
	require.Len(t, got[taskB], 1)
	assert.Equal(t, red.ID, got[taskB][0].ID)
	assert.Equal(t, "#ff0000", got[taskB][0].Color)
	assert.Equal(t, tid, got[taskB][0].TenantID)
	_, hasC := got[taskC]
	assert.False(t, hasC)
}

func TestLabelRepo_GetLabelsByTaskIDsEmptyInput(t *testing.T) {
	f := newLabelFixture(t)

	got, err := f.repo.GetLabelsByTaskIDs(f.ctx, f.tenant.String(), nil)
	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.Empty(t, got)
}

func TestLabelRepo_SetTaskLabelsReplacesAssignments(t *testing.T) {
	f := newLabelFixture(t)
	tid := f.tenant.String()
	task := f.task(t)
	a := f.mustCreate(t, "a", "#111111")
	b := f.mustCreate(t, "b", "#111111")
	c := f.mustCreate(t, "c", "#111111")

	require.NoError(t, f.repo.SetTaskLabels(f.ctx, tid, task, []string{a.ID, b.ID}))
	ids, err := f.repo.GetTaskLabelIDs(f.ctx, tid, task)
	require.NoError(t, err)
	assert.Equal(t, []string{a.ID, b.ID}, ids, "ordered by label name")

	require.NoError(t, f.repo.SetTaskLabels(f.ctx, tid, task, []string{c.ID, b.ID, b.ID}))
	ids, err = f.repo.GetTaskLabelIDs(f.ctx, tid, task)
	require.NoError(t, err)
	assert.Equal(t, []string{b.ID, c.ID}, ids, "a dropped, c added, duplicate id tolerated")

	require.NoError(t, f.repo.SetTaskLabels(f.ctx, tid, task, nil))
	ids, err = f.repo.GetTaskLabelIDs(f.ctx, tid, task)
	require.NoError(t, err)
	assert.Empty(t, ids, "empty set clears all assignments")
}

func TestLabelRepo_SetTaskLabelsLeavesOtherTasksAlone(t *testing.T) {
	f := newLabelFixture(t)
	tid := f.tenant.String()
	taskA, taskB := f.task(t), f.task(t)
	l := f.mustCreate(t, "shared", "#111111")

	require.NoError(t, f.repo.SetTaskLabels(f.ctx, tid, taskA, []string{l.ID}))
	require.NoError(t, f.repo.SetTaskLabels(f.ctx, tid, taskB, []string{l.ID}))
	require.NoError(t, f.repo.SetTaskLabels(f.ctx, tid, taskA, nil))

	ids, err := f.repo.GetTaskLabelIDs(f.ctx, tid, taskB)
	require.NoError(t, err)
	assert.Equal(t, []string{l.ID}, ids)
}

func TestLabelRepo_SetTaskLabelsIgnoresForeignTenantLabel(t *testing.T) {
	f := newLabelFixture(t)
	tid := f.tenant.String()
	task := f.task(t)

	other := f.otherTenant(t, "Label Foreign Label Tenant")
	foreign, err := f.repo.Create(testutil.WithTenantCtx(context.Background(), other), other.String(), "foreign", "#111111")
	require.NoError(t, err)

	own := f.mustCreate(t, "own", "#111111")

	// The foreign id is silently skipped (INSERT ... WHERE EXISTS in own
	// tenant); the own label is still assigned.
	require.NoError(t, f.repo.SetTaskLabels(f.ctx, tid, task, []string{foreign.ID, own.ID}))
	ids, err := f.repo.GetTaskLabelIDs(f.ctx, tid, task)
	require.NoError(t, err)
	assert.Equal(t, []string{own.ID}, ids)
}

// The explicit tenant predicates must hold on their own, independent of the
// RLS policy: under a system context RLS admits every row, so only the SQL
// WHERE clause keeps tenant B's arguments away from tenant A's data.
func TestLabelRepo_ExplicitTenantPredicatesUnderSystemContext(t *testing.T) {
	f := newLabelFixture(t)
	sys := testutil.WithSystemCtx(context.Background())
	tid := f.tenant.String()
	otherTid := uuid.NewString()
	task := f.task(t)

	l, err := f.repo.Create(sys, tid, "Scoped", "#111111")
	require.NoError(t, err)
	require.NoError(t, f.repo.SetTaskLabels(sys, tid, task, []string{l.ID}))

	_, err = f.repo.GetByID(sys, otherTid, l.ID)
	assert.ErrorIs(t, err, label.ErrNotFound)

	other, err := f.repo.List(sys, otherTid)
	require.NoError(t, err)
	assert.Empty(t, other)

	_, err = f.repo.Update(sys, otherTid, l.ID, "Hijacked", "#222222")
	assert.ErrorIs(t, err, label.ErrNotFound)

	assert.ErrorIs(t, f.repo.Delete(sys, otherTid, l.ID), label.ErrNotFound)

	byTask, err := f.repo.GetLabelsByTaskIDs(sys, otherTid, []string{task})
	require.NoError(t, err)
	assert.Empty(t, byTask)

	ids, err := f.repo.GetTaskLabelIDs(sys, otherTid, task)
	require.NoError(t, err)
	assert.Empty(t, ids)

	// A foreign-tenant SetTaskLabels must neither remove nor add assignments.
	require.NoError(t, f.repo.SetTaskLabels(sys, otherTid, task, nil))
	require.NoError(t, f.repo.SetTaskLabels(sys, otherTid, task, []string{l.ID}))
	ids, err = f.repo.GetTaskLabelIDs(sys, tid, task)
	require.NoError(t, err)
	assert.Equal(t, []string{l.ID}, ids)

	intact, err := f.repo.GetByID(sys, tid, l.ID)
	require.NoError(t, err)
	assert.Equal(t, "Scoped", intact.Name)

	// A label of another tenant must not be assignable to this tenant's task
	// even when RLS is open.
	foreignTenant := f.otherTenant(t, "Label Foreign Assign Tenant")
	foreign, err := f.repo.Create(sys, foreignTenant.String(), "foreign", "#111111")
	require.NoError(t, err)
	require.NoError(t, f.repo.SetTaskLabels(sys, tid, task, []string{foreign.ID}))
	// Read the join table directly: the repository's own getters would hide a
	// foreign label behind their tenant predicate and mask a wrong insert.
	var assigned int
	require.NoError(t, f.pool.QueryRow(sys,
		`SELECT count(*) FROM task_labels WHERE task_id = $1`, task).Scan(&assigned))
	assert.Zero(t, assigned, "set replaced the assignment and skipped the foreign label")
}

func TestLabelRepo_ClosedPoolSurfacesErrors(t *testing.T) {
	f := newLabelFixture(t)
	tid := f.tenant.String()
	pool := testutil.PoolFromEnv(t)
	repo := label.NewPostgresRepository(pool)
	pool.Close()

	_, err := repo.Create(f.ctx, tid, "x", "#111111")
	assert.Error(t, err)
	assert.False(t, errors.Is(err, label.ErrDuplicateName))
	_, err = repo.GetByID(f.ctx, tid, uuid.NewString())
	assert.Error(t, err)
	assert.False(t, errors.Is(err, label.ErrNotFound))
	_, err = repo.List(f.ctx, tid)
	assert.Error(t, err)
	_, err = repo.Update(f.ctx, tid, uuid.NewString(), "x", "#111111")
	assert.Error(t, err)
	assert.False(t, errors.Is(err, label.ErrNotFound))
	assert.Error(t, repo.Delete(f.ctx, tid, uuid.NewString()))
	_, err = repo.GetLabelsByTaskIDs(f.ctx, tid, []string{uuid.NewString()})
	assert.Error(t, err)
	assert.Error(t, repo.SetTaskLabels(f.ctx, tid, uuid.NewString(), nil))
	_, err = repo.GetTaskLabelIDs(f.ctx, tid, uuid.NewString())
	assert.Error(t, err)
}
