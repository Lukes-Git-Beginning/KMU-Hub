package vermietung

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

func TestVermietungWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantWrite, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantWrite, "Vermietung Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Vermietung Write Other Tenant")

	repo := NewPostgresRepository(pool)
	ctxA := testutil.WithTenantCtx(context.Background(), tenantWrite)
	ctxB := testutil.WithTenantCtx(context.Background(), tenantOther)
	now := time.Now().UTC()

	obj := &RentalObject{
		ID:        uuid.New(),
		TenantID:  tenantWrite,
		Name:      "Baugeruest 6m",
		Category:  "geruest",
		DailyRate: 45.0,
		Deposit:   200.0,
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.CreateObject(ctxA, obj); err != nil {
		t.Fatalf("CreateObject: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rental_objects", obj.ID)

	rental := &Rental{
		ID:         uuid.New(),
		TenantID:   tenantWrite,
		ObjectID:   obj.ID,
		RenterName: "Mustermann Bau GmbH",
		StartDate:  now,
		EndDate:    now.Add(48 * time.Hour),
		Status:     RentalStatusReserved,
		TotalPrice: 90.0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := repo.CreateRental(ctxA, rental); err != nil {
		t.Fatalf("CreateRental: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rentals", rental.ID)

	inspection := &RentalInspection{
		ID:        uuid.New(),
		TenantID:  tenantWrite,
		RentalID:  rental.ID,
		Kind:      InspectionKindHandover,
		Notes:     "Keine Schaeden",
		PhotoURLs: []string{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.CreateInspection(ctxA, inspection); err != nil {
		t.Fatalf("CreateInspection: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "rental_inspections", inspection.ID)

	rows := []struct {
		table string
		id    uuid.UUID
	}{
		{"rental_objects", obj.ID},
		{"rentals", rental.ID},
		{"rental_inspections", inspection.ID},
	}

	for _, row := range rows {
		t.Run(row.table, func(t *testing.T) {
			testutil.AssertRowCount(t, pool, ctxA, row.table, row.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, row.table, row.id, 0)
		})
	}
}
