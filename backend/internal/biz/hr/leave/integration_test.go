//go:build integration

package leave

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

// seedTenantLeave uses the superPool (BYPASSRLS) because the tenants table has
// WITH CHECK (is_system_context()) — kmuhub_app cannot insert new tenants.
func seedTenantLeave(t *testing.T, superPool *pgxpool.Pool, tenantID uuid.UUID) {
	t.Helper()
	_, err := superPool.Exec(context.Background(),
		`INSERT INTO tenants (id, name, created_at, updated_at)
		 VALUES ($1, $2, NOW(), NOW())
		 ON CONFLICT (id) DO NOTHING`,
		tenantID, fmt.Sprintf("Test Tenant %s", tenantID.String()[:8]),
	)
	if err != nil {
		t.Fatalf("seedTenantLeave: %v", err)
	}
}

func seedUserLeave(t *testing.T, superPool *pgxpool.Pool, tenantID uuid.UUID, firstName, lastName, email string) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	_, err := superPool.Exec(context.Background(),
		`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, 'hash', $4, $5, TRUE, NOW(), NOW())`,
		userID, tenantID, email, firstName, lastName,
	)
	if err != nil {
		t.Fatalf("seedUserLeave %s: %v", email, err)
	}
	return userID
}

// seedLeaveType inserts an hr_leave_types row for the given tenant.
func seedLeaveType(t *testing.T, superPool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
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
		t.Fatalf("seedLeaveType: %v", err)
	}
	return typeID
}

// insertLeaveRequest inserts an hr_leave_requests row directly and returns its ID.
// reason and approval_comment are TEXT (nullable) but scanned into Go string, so
// explicit empty strings avoid pgx NULL→string coercion failures.
func insertLeaveRequest(
	t *testing.T, superPool *pgxpool.Pool,
	tenantID, employeeID, leaveTypeID uuid.UUID,
	start, end time.Time, statusStr string,
) uuid.UUID {
	t.Helper()
	reqID := uuid.New()
	totalDays := decimal.NewFromFloat(1.0)
	_, err := superPool.Exec(context.Background(),
		`INSERT INTO hr_leave_requests
			(id, tenant_id, employee_id, leave_type_id,
			 start_date, end_date,
			 is_half_day_start, is_half_day_end,
			 total_days, reason, status, approval_comment,
			 au_document_required, created_at, updated_at)
		 VALUES ($1, $2, $3, $4,
		         $5, $6,
		         FALSE, FALSE,
		         $7, '', $8, '',
		         FALSE, NOW(), NOW())`,
		reqID, tenantID, employeeID, leaveTypeID,
		start, end, totalDays, statusStr,
	)
	if err != nil {
		t.Fatalf("insertLeaveRequest: %v", err)
	}
	return reqID
}

// ============================================================================
// Tests
// ============================================================================

// TestIntegrationLeaveGetByIDHappyPath verifies that GetByID returns the
// correct employee_name assembled via CONCAT_WS(first_name, last_name) —
// regression guard for the display_name bug fixed in c2cc98ad.
func TestIntegrationLeaveGetByIDHappyPath(t *testing.T) {
	appPool, superURL := pgtc.StartPostgres(t)
	superPool := pgtc.SuperPool(t, superURL)
	t.Cleanup(superPool.Close)

	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)

	seedTenantLeave(t, superPool, tenantID)
	userID := seedUserLeave(t, superPool, tenantID, "Ida", "Brandt", "ida@example.com")
	typeID := seedLeaveType(t, superPool, tenantID)

	startDate := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	reqID := insertLeaveRequest(t, superPool, tenantID, userID, typeID, startDate, endDate, "pending")

	repo := NewPostgresLeaveRequestRepo(appPool)
	got, err := repo.GetByID(ctx, reqID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.ID != reqID {
		t.Errorf("ID: got %s, want %s", got.ID, reqID)
	}
	if got.EmployeeName != "Ida Brandt" {
		t.Errorf("EmployeeName: got %q, want %q (display_name regression check)", got.EmployeeName, "Ida Brandt")
	}
	if got.TenantID != tenantID {
		t.Errorf("TenantID: got %s, want %s", got.TenantID, tenantID)
	}
}

// TestIntegrationLeaveListHappyPath verifies that List returns leave requests
// with correct EmployeeName assembled via CONCAT_WS.
func TestIntegrationLeaveListHappyPath(t *testing.T) {
	appPool, superURL := pgtc.StartPostgres(t)
	superPool := pgtc.SuperPool(t, superURL)
	t.Cleanup(superPool.Close)

	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)

	seedTenantLeave(t, superPool, tenantID)
	userID := seedUserLeave(t, superPool, tenantID, "Jakob", "Ernst", "jakob@example.com")
	typeID := seedLeaveType(t, superPool, tenantID)

	startDate := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)
	insertLeaveRequest(t, superPool, tenantID, userID, typeID, startDate, endDate, "pending")

	repo := NewPostgresLeaveRequestRepo(appPool)
	filter := LeaveRequestFilter{
		TenantID: tenantID,
		Page:     1,
		PerPage:  20,
	}
	requests, total, err := repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total < 1 || len(requests) < 1 {
		t.Fatalf("List: got total=%d len=%d, want >=1", total, len(requests))
	}

	var found bool
	for _, r := range requests {
		if r.EmployeeID == userID {
			found = true
			if r.EmployeeName != "Jakob Ernst" {
				t.Errorf("EmployeeName: got %q, want %q", r.EmployeeName, "Jakob Ernst")
			}
		}
	}
	if !found {
		t.Errorf("employee %s not found in List results", userID)
	}
}

// TestIntegrationLeaveCrossTenantIsolation verifies that listing leave requests
// for tenant B does not return tenant A's requests (filter / RLS enforcement).
func TestIntegrationLeaveCrossTenantIsolation(t *testing.T) {
	appPool, superURL := pgtc.StartPostgres(t)
	superPool := pgtc.SuperPool(t, superURL)
	t.Cleanup(superPool.Close)

	tenantA := uuid.New()
	tenantB := uuid.New()
	ctxB := pgtc.TenantCtx(context.Background(), tenantB)

	seedTenantLeave(t, superPool, tenantA)
	seedTenantLeave(t, superPool, tenantB)

	userA := seedUserLeave(t, superPool, tenantA, "Klara", "Stein", "klara@example.com")
	typeA := seedLeaveType(t, superPool, tenantA)

	startDate := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC)
	insertLeaveRequest(t, superPool, tenantA, userA, typeA, startDate, endDate, "approved")

	repo := NewPostgresLeaveRequestRepo(appPool)

	// Tenant B lists — must not see tenant A's request.
	requestsB, _, err := repo.List(ctxB, LeaveRequestFilter{TenantID: tenantB, Page: 1, PerPage: 50})
	if err != nil {
		t.Fatalf("List B: %v", err)
	}
	for _, r := range requestsB {
		if r.TenantID == tenantA {
			t.Errorf("cross-tenant leak: tenant B sees leave request %s from tenant A", r.ID)
		}
	}
}
