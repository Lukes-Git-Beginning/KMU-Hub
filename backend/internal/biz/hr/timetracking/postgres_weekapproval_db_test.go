package timetracking_test

// hr_week_approvals is the persistence layer under the week-lock state machine
// in service_week_lock.go: SubmitWeek/ApproveWeek/RejectWeek/ReopenWeek all read
// through GetByEmployeeWeek and write through Upsert. Before this file the
// PostgresWeekApprovalRepo methods had zero DB coverage — every existing test
// ran against the in-memory mock in service_extended_test.go, so a wrong SQL
// predicate here (tenant scoping, the ON CONFLICT target, a COALESCE silently
// keeping a field the service meant to null out on reopen) would ship
// unnoticed.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/biz/hr/timetracking"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// weekApprovalFixture provides a repo plus a ready-to-use system context.
// Run in system context for the same reason postgres_tenant_scope_test.go
// does: it switches RLS off, so a foreign-tenant miss can only come from the
// query's own tenant_id predicate — exactly what these tests pin.
type weekApprovalFixture struct {
	pool *pgxpool.Pool
	repo *timetracking.PostgresWeekApprovalRepo
	ctx  context.Context
}

func newWeekApprovalFixture(t *testing.T) *weekApprovalFixture {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "TenantB")

	return &weekApprovalFixture{
		pool: pool,
		repo: timetracking.NewPostgresWeekApprovalRepo(pool),
		ctx:  testutil.WithSystemCtx(context.Background()),
	}
}

func (f *weekApprovalFixture) cleanup(t *testing.T, tenantID, employeeID uuid.UUID, weekStart time.Time) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = f.pool.Exec(f.ctx,
			`DELETE FROM hr_week_approvals WHERE tenant_id = $1 AND employee_id = $2 AND week_start = $3`,
			tenantID, employeeID, weekStart,
		)
	})
}

func TestWeekApprovalRepo_GetByEmployeeWeek_NotFound(t *testing.T) {
	f := newWeekApprovalFixture(t)

	_, err := f.repo.GetByEmployeeWeek(f.ctx, testutil.TenantA, uuid.New(), time.Now())
	assert.ErrorIs(t, err, timetracking.ErrWeekApprovalNotFound)
}

// TestWeekApprovalRepo_Upsert_InsertThenUpdate_RoundTrips proves the ON CONFLICT
// target is (tenant_id, employee_id, week_start), not id: a second Upsert call
// for the same tenant/employee/week must update the existing row rather than
// violate the unique constraint or insert a duplicate — which is exactly what
// SubmitWeek -> ApproveWeek relies on (both Upsert the same logical week).
func TestWeekApprovalRepo_Upsert_InsertThenUpdate_RoundTrips(t *testing.T) {
	f := newWeekApprovalFixture(t)
	employeeID := uuid.New()
	weekStart := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC) // Monday
	f.cleanup(t, testutil.TenantA, employeeID, weekStart)

	firstID := uuid.New()
	now := time.Now().Truncate(time.Second)
	err := f.repo.Upsert(f.ctx, &models.HRWeekApproval{
		ID:         firstID,
		TenantID:   testutil.TenantA,
		EmployeeID: employeeID,
		WeekStart:  weekStart,
		Status:     models.WeekApprovalOpen,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	require.NoError(t, err)

	inserted, err := f.repo.GetByEmployeeWeek(f.ctx, testutil.TenantA, employeeID, weekStart)
	require.NoError(t, err)
	assert.Equal(t, firstID, inserted.ID)
	assert.Equal(t, models.WeekApprovalOpen, inserted.Status)
	assert.Nil(t, inserted.SubmittedAt)

	// Second Upsert for the same (tenant, employee, week) — a fresh id, as
	// SubmitWeek passes when no record existed in-process yet, must still
	// land on the same row rather than erroring or duplicating.
	submittedAt := now.Add(time.Minute)
	err = f.repo.Upsert(f.ctx, &models.HRWeekApproval{
		ID:          uuid.New(),
		TenantID:    testutil.TenantA,
		EmployeeID:  employeeID,
		WeekStart:   weekStart,
		Status:      models.WeekApprovalSubmitted,
		SubmittedAt: &submittedAt,
		CreatedAt:   now,
		UpdatedAt:   submittedAt,
	})
	require.NoError(t, err)

	updated, err := f.repo.GetByEmployeeWeek(f.ctx, testutil.TenantA, employeeID, weekStart)
	require.NoError(t, err)
	assert.Equal(t, models.WeekApprovalSubmitted, updated.Status)
	require.NotNil(t, updated.SubmittedAt)
	assert.WithinDuration(t, submittedAt, *updated.SubmittedAt, time.Second)

	var count int
	countErr := f.pool.QueryRow(f.ctx,
		`SELECT count(*) FROM hr_week_approvals WHERE tenant_id = $1 AND employee_id = $2 AND week_start = $3`,
		testutil.TenantA, employeeID, weekStart,
	).Scan(&count)
	require.NoError(t, countErr)
	assert.Equal(t, 1, count, "a second Upsert for the same week must update, not duplicate")
}

// TestWeekApprovalRepo_Upsert_NullsClearedFields proves that Upsert actually
// nulls out approved_by/approved_at/rejection_reason on reopen. ReopenWeek
// (service_week_lock.go) sets those to nil on the Go struct and relies on the
// SQL to persist that — an accidental COALESCE(EXCLUDED.x, hr_week_approvals.x)
// would silently keep the stale value and this test would catch it.
func TestWeekApprovalRepo_Upsert_NullsClearedFields(t *testing.T) {
	f := newWeekApprovalFixture(t)
	employeeID := uuid.New()
	weekStart := time.Date(2026, 6, 22, 0, 0, 0, 0, time.UTC) // Monday
	f.cleanup(t, testutil.TenantA, employeeID, weekStart)

	now := time.Now().Truncate(time.Second)
	approverID := uuid.New()
	reason := "missing project codes"
	err := f.repo.Upsert(f.ctx, &models.HRWeekApproval{
		ID:              uuid.New(),
		TenantID:        testutil.TenantA,
		EmployeeID:      employeeID,
		WeekStart:       weekStart,
		Status:          models.WeekApprovalRejected,
		ApprovedBy:      &approverID,
		ApprovedAt:      &now,
		RejectionReason: &reason,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	require.NoError(t, err)

	// Reopen: the service clears all three fields before upserting.
	err = f.repo.Upsert(f.ctx, &models.HRWeekApproval{
		ID:         uuid.New(),
		TenantID:   testutil.TenantA,
		EmployeeID: employeeID,
		WeekStart:  weekStart,
		Status:     models.WeekApprovalOpen,
		CreatedAt:  now,
		UpdatedAt:  now.Add(time.Minute),
	})
	require.NoError(t, err)

	reopened, err := f.repo.GetByEmployeeWeek(f.ctx, testutil.TenantA, employeeID, weekStart)
	require.NoError(t, err)
	assert.Equal(t, models.WeekApprovalOpen, reopened.Status)
	assert.Nil(t, reopened.ApprovedBy, "reopen must clear approved_by, not keep the stale approver")
	assert.Nil(t, reopened.ApprovedAt, "reopen must clear approved_at")
	assert.Nil(t, reopened.RejectionReason, "reopen must clear a stale rejection reason")
}

// TestWeekApprovalRepo_ListByWeek_ScopesToTenantAndWeek seeds three rows —
// two employees in TenantA's target week, one in a different tenant and one
// in a different week — and asserts ListByWeek returns exactly the two that
// match both predicates, ordered by employee_id.
func TestWeekApprovalRepo_ListByWeek_ScopesToTenantAndWeek(t *testing.T) {
	f := newWeekApprovalFixture(t)
	weekStart := time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC) // Monday
	otherWeek := weekStart.AddDate(0, 0, 7)

	empLow := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	empHigh := uuid.MustParse("ffffffff-0000-0000-0000-000000000001")
	empOtherTenant := uuid.New()
	empOtherWeek := uuid.New()

	now := time.Now().Truncate(time.Second)
	for _, seed := range []struct {
		tenant uuid.UUID
		emp    uuid.UUID
		week   time.Time
	}{
		{testutil.TenantA, empHigh, weekStart},
		{testutil.TenantA, empLow, weekStart},
		{testutil.TenantB, empOtherTenant, weekStart},
		{testutil.TenantA, empOtherWeek, otherWeek},
	} {
		require.NoError(t, f.repo.Upsert(f.ctx, &models.HRWeekApproval{
			ID:         uuid.New(),
			TenantID:   seed.tenant,
			EmployeeID: seed.emp,
			WeekStart:  seed.week,
			Status:     models.WeekApprovalOpen,
			CreatedAt:  now,
			UpdatedAt:  now,
		}))
		f.cleanup(t, seed.tenant, seed.emp, seed.week)
	}

	results, err := f.repo.ListByWeek(f.ctx, testutil.TenantA, weekStart)
	require.NoError(t, err)
	require.Len(t, results, 2, "must exclude the other tenant's row and the other week's row")
	assert.Equal(t, empLow, results[0].EmployeeID, "ListByWeek must order by employee_id ascending")
	assert.Equal(t, empHigh, results[1].EmployeeID)
}

// TestWeekApprovalRepo_ForeignTenantNotFound is the cross-tenant read half:
// a caller who learned another tenant's employee id must not be able to read
// that tenant's week-approval status through its own tenant id.
func TestWeekApprovalRepo_ForeignTenantNotFound(t *testing.T) {
	f := newWeekApprovalFixture(t)
	employeeID := uuid.New()
	weekStart := time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC) // Monday
	f.cleanup(t, testutil.TenantA, employeeID, weekStart)

	now := time.Now().Truncate(time.Second)
	require.NoError(t, f.repo.Upsert(f.ctx, &models.HRWeekApproval{
		ID:         uuid.New(),
		TenantID:   testutil.TenantA,
		EmployeeID: employeeID,
		WeekStart:  weekStart,
		Status:     models.WeekApprovalSubmitted,
		CreatedAt:  now,
		UpdatedAt:  now,
	}))

	own, err := f.repo.GetByEmployeeWeek(f.ctx, testutil.TenantA, employeeID, weekStart)
	require.NoError(t, err)
	assert.Equal(t, models.WeekApprovalSubmitted, own.Status)

	_, err = f.repo.GetByEmployeeWeek(f.ctx, testutil.TenantB, employeeID, weekStart)
	assert.ErrorIs(t, err, timetracking.ErrWeekApprovalNotFound,
		"TenantB must not see TenantA's week-approval row for the same employee id")
}
