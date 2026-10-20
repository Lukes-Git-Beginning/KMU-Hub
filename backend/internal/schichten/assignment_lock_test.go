package schichten

// Concurrent AssignEmployee calls against the real PostgresRepository (as
// kmuhub_app, RLS active). The capacity count and the rest-period read are
// check-then-act: without WithAssignmentLock two callers both see "room left"
// and both insert.

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestAssignEmployee_ConcurrentAssignmentsRespectInvariants(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Schichten Assignment Lock Tenant")

	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	// Weekday daytime window so the minor-protection rules never interfere.
	base := time.Date(2031, 3, 4, 8, 0, 0, 0, time.UTC)

	newShift := func(t *testing.T, start, end time.Time, capacity *int) *Shift {
		t.Helper()
		now := time.Now().UTC()
		s := &Shift{
			ID: uuid.New(), TenantID: tenantID, Title: "Lock-Test",
			StartTime: start, EndTime: end, Status: ShiftStatusDraft,
			Capacity: capacity, CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.CreateShift(ctx, s); err != nil {
			t.Fatalf("CreateShift: %v", err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "shifts", s.ID) })
		return s
	}

	// race runs every call at once and returns the errors in call order.
	race := func(calls []func() error) []error {
		errs := make([]error, len(calls))
		start := make(chan struct{})
		var wg sync.WaitGroup
		for i, call := range calls {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				errs[i] = call()
			}()
		}
		close(start)
		wg.Wait()
		return errs
	}

	t.Run("capacity is not exceeded", func(t *testing.T) {
		capacity := 2
		shift := newShift(t, base, base.Add(8*time.Hour), &capacity)

		const callers = 8
		calls := make([]func() error, callers)
		for i := range calls {
			employeeID := uuid.New()
			calls[i] = func() error {
				_, err := svc.AssignEmployee(ctx, AssignEmployeeInput{
					TenantID: tenantID, ShiftID: shift.ID, EmployeeID: employeeID,
				})
				return err
			}
		}

		ok, full := 0, 0
		for _, err := range race(calls) {
			switch {
			case err == nil:
				ok++
			case errors.Is(err, ErrShiftFull):
				full++
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}
		if ok != capacity || full != callers-capacity {
			t.Errorf("assigned=%d full=%d, want %d and %d", ok, full, capacity, callers-capacity)
		}
		count, err := repo.CountAssignments(ctx, tenantID, shift.ID)
		if err != nil {
			t.Fatalf("CountAssignments: %v", err)
		}
		if count != capacity {
			t.Errorf("stored assignments = %d, want %d", count, capacity)
		}
	})

	t.Run("rest period holds across parallel assignments of one employee", func(t *testing.T) {
		// Two shifts one hour apart: each alone is fine, both together break ArbZG §5.
		early := newShift(t, base.Add(24*time.Hour), base.Add(32*time.Hour), nil)
		late := newShift(t, base.Add(33*time.Hour), base.Add(40*time.Hour), nil)
		employeeID := uuid.New()

		assign := func(shiftID uuid.UUID) func() error {
			return func() error {
				_, err := svc.AssignEmployee(ctx, AssignEmployeeInput{
					TenantID: tenantID, ShiftID: shiftID, EmployeeID: employeeID,
				})
				return err
			}
		}

		ok, violations := 0, 0
		for _, err := range race([]func() error{assign(early.ID), assign(late.ID)}) {
			switch {
			case err == nil:
				ok++
			case errors.Is(err, ErrArbzgViolation):
				violations++
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}
		if ok != 1 || violations != 1 {
			t.Errorf("assigned=%d violations=%d, want 1 and 1", ok, violations)
		}
	})

	t.Run("unknown shift is still not found", func(t *testing.T) {
		_, err := svc.AssignEmployee(ctx, AssignEmployeeInput{
			TenantID: tenantID, ShiftID: uuid.New(), EmployeeID: uuid.New(),
		})
		if !errors.Is(err, ErrShiftNotFound) {
			t.Errorf("err = %v, want ErrShiftNotFound", err)
		}
	})

	t.Run("failed check leaves nothing behind", func(t *testing.T) {
		capacity := 1
		shift := newShift(t, base.Add(72*time.Hour), base.Add(80*time.Hour), &capacity)
		if _, err := svc.AssignEmployee(ctx, AssignEmployeeInput{TenantID: tenantID, ShiftID: shift.ID, EmployeeID: uuid.New()}); err != nil {
			t.Fatalf("first assign: %v", err)
		}
		if _, err := svc.AssignEmployee(ctx, AssignEmployeeInput{TenantID: tenantID, ShiftID: shift.ID, EmployeeID: uuid.New()}); !errors.Is(err, ErrShiftFull) {
			t.Fatalf("second assign err = %v, want ErrShiftFull", err)
		}
		// The lock is released: a third call returns promptly instead of hanging.
		done := make(chan error, 1)
		go func() {
			_, err := svc.AssignEmployee(ctx, AssignEmployeeInput{TenantID: tenantID, ShiftID: shift.ID, EmployeeID: uuid.New()})
			done <- err
		}()
		select {
		case err := <-done:
			if !errors.Is(err, ErrShiftFull) {
				t.Errorf("third assign err = %v, want ErrShiftFull", err)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("third assign hung: lock not released after a rejected assignment")
		}
	})
}
