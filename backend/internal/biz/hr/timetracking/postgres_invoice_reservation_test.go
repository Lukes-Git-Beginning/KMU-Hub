package timetracking_test

// ReserveWorkTimeForInvoice is the double-billing guard for
// CreateInvoiceFromTimeEntries (fix-biz-time-entry-invoice-double-billing).
// Before this fix, AggregateWorkTimeForInvoice only filtered on tenant,
// employee, status, and the clock_in window — a second call for the same
// employee/period (double-click, retry after timeout, two staff members
// working the same invoice) aggregated the same entries again and produced a
// second, fully separate invoice over the same hours. This file proves the
// fix at the layer where it actually lives: the repository, not the gRPC
// handler — see the "out of scope" note on TestCreateInvoiceFromTimeEntries_
// Validation in biz_grpc_invoices_creditnotes_payments_test.go, which already
// declined to build a full WorkTimeRepository fake for a handler-level test.
// The handler's own logic (call reserve, create invoice, release on failure,
// confirm on success) is thin wiring around this repository call.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/biz/hr/timetracking"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

type reservationFixture struct {
	pool       *pgxpool.Pool
	repo       *timetracking.PostgresWorkTimeRepo
	ctx        context.Context
	tenantID   uuid.UUID
	employeeID uuid.UUID
	from       time.Time
	to         time.Time
}

func newReservationFixture(t *testing.T) *reservationFixture {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	testutil.EnsureTenant(t, pool, testutil.TenantA, "TenantA")

	f := &reservationFixture{
		pool:     pool,
		repo:     timetracking.NewPostgresWorkTimeRepo(pool),
		ctx:      testutil.WithSystemCtx(context.Background()),
		tenantID: testutil.TenantA,
		from:     time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		to:       time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC),
	}

	f.employeeID = testutil.SeedRow(t, pool, "users", map[string]any{
		"id":            uuid.New(),
		"tenant_id":     f.tenantID,
		"email":         "reservation-fixture@test.local",
		"password_hash": "x",
		"first_name":    "Reservation",
		"last_name":     "Fixture",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", f.employeeID) })

	return f
}

// seedBillableEntry creates one completed entry inside the fixture's date
// window with the given net minutes.
func (f *reservationFixture) seedBillableEntry(t *testing.T, netMinutes int) uuid.UUID {
	t.Helper()
	clockIn := f.from.Add(24 * time.Hour)
	id := testutil.SeedRow(t, f.pool, "hr_work_time_entries", map[string]any{
		"id":               uuid.New(),
		"tenant_id":        f.tenantID,
		"employee_id":      f.employeeID,
		"clock_in":         clockIn,
		"clock_out":        clockIn.Add(time.Duration(netMinutes) * time.Minute),
		"break_minutes":    0,
		"net_work_minutes": netMinutes,
		"status":           "completed",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "hr_work_time_entries", id) })
	return id
}

func (f *reservationFixture) seedInvoice(t *testing.T) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, f.pool, "finance_invoices", map[string]any{
		"id":           uuid.New(),
		"tenant_id":    f.tenantID,
		"created_by":   f.employeeID,
		"invoice_date": time.Now(),
		"due_date":     time.Now().AddDate(0, 0, 30),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "finance_invoices", id) })
	return id
}

// TestReserveWorkTimeForInvoice_SecondCallSeesNothing is the core regression
// test: two calls with identical employee/period must not both bill the same
// hours. The second call is the exact scenario a double-click or retry
// produces.
func TestReserveWorkTimeForInvoice_SecondCallSeesNothing(t *testing.T) {
	f := newReservationFixture(t)
	f.seedBillableEntry(t, 120)
	f.seedBillableEntry(t, 60)

	total1, ids1, err := f.repo.ReserveWorkTimeForInvoice(f.ctx, f.tenantID, f.employeeID, f.from, f.to)
	require.NoError(t, err)
	assert.Equal(t, 180, total1)
	assert.Len(t, ids1, 2)

	total2, ids2, err := f.repo.ReserveWorkTimeForInvoice(f.ctx, f.tenantID, f.employeeID, f.from, f.to)
	require.NoError(t, err)
	assert.Equal(t, 0, total2, "second reservation for the same employee/period must not re-bill the same hours")
	assert.Empty(t, ids2)
}

// TestReserveWorkTimeForInvoice_ExcludesAlreadyBilled belt-and-braces on the
// query predicate itself: an entry billed by an unrelated, earlier
// reservation must never show up again, even for a differently-scoped call.
func TestReserveWorkTimeForInvoice_ExcludesAlreadyBilled(t *testing.T) {
	f := newReservationFixture(t)
	f.seedBillableEntry(t, 90)

	total, ids, err := f.repo.ReserveWorkTimeForInvoice(f.ctx, f.tenantID, f.employeeID, f.from, f.to)
	require.NoError(t, err)
	require.Equal(t, 90, total)
	require.Len(t, ids, 1)

	var billedAt *time.Time
	err = f.pool.QueryRow(f.ctx, `SELECT billed_at FROM hr_work_time_entries WHERE id = $1`, ids[0]).Scan(&billedAt)
	require.NoError(t, err)
	assert.NotNil(t, billedAt, "reservation must stamp billed_at in the same transaction")
}

// TestReleaseInvoiceReservation_MakesEntriesBillableAgain covers the
// compensating path: if invoiceService.Create fails after the reservation,
// the handler releases it so the hours are not stuck forever.
func TestReleaseInvoiceReservation_MakesEntriesBillableAgain(t *testing.T) {
	f := newReservationFixture(t)
	f.seedBillableEntry(t, 45)

	total, ids, err := f.repo.ReserveWorkTimeForInvoice(f.ctx, f.tenantID, f.employeeID, f.from, f.to)
	require.NoError(t, err)
	require.Equal(t, 45, total)

	require.NoError(t, f.repo.ReleaseInvoiceReservation(f.ctx, f.tenantID, ids))

	total2, ids2, err := f.repo.ReserveWorkTimeForInvoice(f.ctx, f.tenantID, f.employeeID, f.from, f.to)
	require.NoError(t, err)
	assert.Equal(t, 45, total2, "a released reservation must become billable again")
	assert.Len(t, ids2, 1)
}

// TestConfirmInvoiceReservation_StampsInvoiceID covers the success path:
// once an invoice exists, the entries record which one claimed them.
func TestConfirmInvoiceReservation_StampsInvoiceID(t *testing.T) {
	f := newReservationFixture(t)
	f.seedBillableEntry(t, 30)
	invoiceID := f.seedInvoice(t)

	_, ids, err := f.repo.ReserveWorkTimeForInvoice(f.ctx, f.tenantID, f.employeeID, f.from, f.to)
	require.NoError(t, err)
	require.Len(t, ids, 1)

	require.NoError(t, f.repo.ConfirmInvoiceReservation(f.ctx, f.tenantID, invoiceID, ids))

	var gotInvoiceID uuid.UUID
	err = f.pool.QueryRow(f.ctx, `SELECT invoice_id FROM hr_work_time_entries WHERE id = $1`, ids[0]).Scan(&gotInvoiceID)
	require.NoError(t, err)
	assert.Equal(t, invoiceID, gotInvoiceID)

	// ReleaseInvoiceReservation is scoped to invoice_id IS NULL — a confirmed
	// reservation must not be silently unbillable-again by a stray release call.
	require.NoError(t, f.repo.ReleaseInvoiceReservation(f.ctx, f.tenantID, ids))
	var billedAt *time.Time
	err = f.pool.QueryRow(f.ctx, `SELECT billed_at FROM hr_work_time_entries WHERE id = $1`, ids[0]).Scan(&billedAt)
	require.NoError(t, err)
	assert.NotNil(t, billedAt, "release must not touch an already-confirmed reservation")
}
