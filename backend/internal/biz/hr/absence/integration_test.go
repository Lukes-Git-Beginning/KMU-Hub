//go:build integration

package absence

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/testsupport/pgtc"
)

// ============================================================================
// Helpers
// ============================================================================

// seedTenantAbs uses the superPool (BYPASSRLS) because the tenants table has
// WITH CHECK (is_system_context()) — kmuhub_app cannot insert new tenants.
func seedTenantAbs(t *testing.T, superPool *pgxpool.Pool, tenantID uuid.UUID) {
	t.Helper()
	_, err := superPool.Exec(context.Background(),
		`INSERT INTO tenants (id, name, created_at, updated_at)
		 VALUES ($1, $2, NOW(), NOW())
		 ON CONFLICT (id) DO NOTHING`,
		tenantID, fmt.Sprintf("Test Tenant %s", tenantID.String()[:8]),
	)
	if err != nil {
		t.Fatalf("seedTenantAbs: %v", err)
	}
}

func seedUserAbs(t *testing.T, superPool *pgxpool.Pool, tenantID uuid.UUID, firstName, lastName, email string) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	_, err := superPool.Exec(context.Background(),
		`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, 'hash', $4, $5, TRUE, NOW(), NOW())`,
		userID, tenantID, email, firstName, lastName,
	)
	if err != nil {
		t.Fatalf("seedUserAbs %s: %v", email, err)
	}
	return userID
}

// seedLeaveTypeAbs inserts an hr_leave_types row and returns the type ID.
func seedLeaveTypeAbs(t *testing.T, superPool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	typeID := uuid.New()
	_, err := superPool.Exec(context.Background(),
		`INSERT INTO hr_leave_types
			(id, tenant_id, name, key, color, deducts_from_balance, requires_approval,
			 requires_au_document, is_system, sort_order, created_at)
		 VALUES ($1, $2, 'Urlaub', $3, '#3d8abf', TRUE, TRUE, FALSE, FALSE, 1, NOW())`,
		typeID, tenantID, fmt.Sprintf("urlaub-%s", typeID.String()[:8]),
	)
	if err != nil {
		t.Fatalf("seedLeaveTypeAbs: %v", err)
	}
	return typeID
}

// seedApprovedLeaveRequest inserts an approved hr_leave_requests row.
func seedApprovedLeaveRequest(
	t *testing.T, superPool *pgxpool.Pool,
	tenantID, employeeID, leaveTypeID uuid.UUID,
	start, end time.Time,
) uuid.UUID {
	t.Helper()
	reqID := uuid.New()
	totalDays := decimal.NewFromFloat(1.0)
	_, err := superPool.Exec(context.Background(),
		`INSERT INTO hr_leave_requests
			(id, tenant_id, employee_id, leave_type_id,
			 start_date, end_date,
			 is_half_day_start, is_half_day_end,
			 total_days, status,
			 au_document_required, created_at, updated_at)
		 VALUES ($1, $2, $3, $4,
		         $5, $6,
		         FALSE, FALSE,
		         $7, 'approved',
		         FALSE, NOW(), NOW())`,
		reqID, tenantID, employeeID, leaveTypeID,
		start, end, totalDays,
	)
	if err != nil {
		t.Fatalf("seedApprovedLeaveRequest: %v", err)
	}
	return reqID
}

// ============================================================================
// Tests
// ============================================================================

// TestIntegrationAbsenceCalendarHappyPath verifies that GetAbsenceCalendar
// returns an approved leave request with the correct employee_name assembled via
// CONCAT_WS(first_name, last_name) — regression guard for c2cc98ad.
func TestIntegrationAbsenceCalendarHappyPath(t *testing.T) {
	appPool, superURL := pgtc.StartPostgres(t)
	superPool := pgtc.SuperPool(t, superURL)
	t.Cleanup(superPool.Close)

	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)

	seedTenantAbs(t, superPool, tenantID)
	userID := seedUserAbs(t, superPool, tenantID, "Greta", "Hoffmann", "greta@example.com")
	typeID := seedLeaveTypeAbs(t, superPool, tenantID)

	startDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC)
	seedApprovedLeaveRequest(t, superPool, tenantID, userID, typeID, startDate, endDate)

	repo := NewPostgresAbsenceRepo(appPool)
	filter := AbsenceFilter{
		TenantID:  tenantID,
		StartDate: startDate.AddDate(0, 0, -1),
		EndDate:   endDate.AddDate(0, 0, 1),
	}
	entries, err := repo.GetAbsenceCalendar(ctx, filter)
	if err != nil {
		t.Fatalf("GetAbsenceCalendar: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("GetAbsenceCalendar: got 0 entries, want >= 1")
	}

	var found *AbsenceEntry
	for _, e := range entries {
		if e.EmployeeID == userID {
			found = e
			break
		}
	}
	if found == nil {
		t.Fatalf("GetAbsenceCalendar: employee %s not found in results", userID)
	}
	if found.EmployeeName != "Greta Hoffmann" {
		t.Errorf("EmployeeName: got %q, want %q", found.EmployeeName, "Greta Hoffmann")
	}
}

// TestIntegrationAbsenceCalendarCrossTenant verifies that tenant B's request
// for absence calendar does not include tenant A's approved leave entries.
func TestIntegrationAbsenceCalendarCrossTenant(t *testing.T) {
	appPool, superURL := pgtc.StartPostgres(t)
	superPool := pgtc.SuperPool(t, superURL)
	t.Cleanup(superPool.Close)

	tenantA := uuid.New()
	tenantB := uuid.New()
	ctxB := pgtc.TenantCtx(context.Background(), tenantB)

	seedTenantAbs(t, superPool, tenantA)
	seedTenantAbs(t, superPool, tenantB)

	userA := seedUserAbs(t, superPool, tenantA, "Hans", "Meier", "hans@example.com")
	typeA := seedLeaveTypeAbs(t, superPool, tenantA)

	startDate := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	seedApprovedLeaveRequest(t, superPool, tenantA, userA, typeA, startDate, endDate)

	repo := NewPostgresAbsenceRepo(appPool)

	// Tenant B queries absence calendar — must not see tenant A's entry.
	entriesB, err := repo.GetAbsenceCalendar(ctxB, AbsenceFilter{
		TenantID:  tenantB,
		StartDate: startDate.AddDate(0, 0, -1),
		EndDate:   endDate.AddDate(0, 0, 1),
	})
	if err != nil {
		t.Fatalf("GetAbsenceCalendar B: %v", err)
	}
	for _, e := range entriesB {
		if e.EmployeeID == userA {
			t.Errorf("cross-tenant leak: tenant B sees absence for employee %s from tenant A", userA)
		}
	}
}
