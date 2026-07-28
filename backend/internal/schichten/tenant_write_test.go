package schichten

// Exercises the real write path (PostgresRepository, not testutil.SeedRow) so a
// forgotten tenant_id column would show up as a genuine INSERT/constraint failure
// instead of being silently bypassed by a system-context fixture. Complements
// tenant_isolation_phase2_test.go, which only proves RLS SELECT filtering on
// hand-seeded rows.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestSchichtenWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantWrite, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantWrite, "Schichten Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Schichten Write Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantWrite)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	shift := &Shift{
		ID:        uuid.New(),
		TenantID:  tenantWrite,
		Title:     "Fruehschicht",
		StartTime: now,
		EndTime:   now.Add(8 * time.Hour),
		Status:    ShiftStatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.CreateShift(ctxA, shift); err != nil {
		t.Fatalf("CreateShift: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "shifts", shift.ID)

	assignment := &ShiftAssignment{
		ID:         uuid.New(),
		TenantID:   tenantWrite,
		ShiftID:    shift.ID,
		EmployeeID: uuid.New(),
		AssignedAt: now,
	}
	if err := repo.CreateAssignment(ctxA, assignment); err != nil {
		t.Fatalf("CreateAssignment: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "shift_assignments", assignment.ID)

	template := &ShiftTemplate{
		ID:              uuid.New(),
		TenantID:        tenantWrite,
		Name:            "Wochentemplate",
		DayOfWeek:       1,
		StartHour:       8,
		StartMinute:     0,
		DurationMinutes: 480,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := repo.CreateTemplate(ctxA, template); err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "shift_templates", template.ID)

	swapReq := &SwapRequest{
		ID:                    uuid.New(),
		TenantID:              tenantWrite,
		AssignmentID:          assignment.ID,
		RequestedByEmployeeID: assignment.EmployeeID,
		SwapWithEmployeeID:    uuid.New(),
		ShiftID:               shift.ID,
		Status:                SwapRequestStatusPending,
		IdempotencyKey:        "swap-" + uuid.New().String(),
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	if err := repo.CreateSwapRequest(ctxA, swapReq); err != nil {
		t.Fatalf("CreateSwapRequest: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "shift_swap_requests", swapReq.ID)

	rows := []struct {
		table string
		id    uuid.UUID
	}{
		{"shifts", shift.ID},
		{"shift_assignments", assignment.ID},
		{"shift_templates", template.ID},
		{"shift_swap_requests", swapReq.ID},
	}

	for _, row := range rows {
		t.Run(row.table, func(t *testing.T) {
			testutil.AssertRowCount(t, pool, ctxA, row.table, row.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, row.table, row.id, 0)
		})
	}
}
