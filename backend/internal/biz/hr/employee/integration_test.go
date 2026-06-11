//go:build integration

package employee

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/testsupport/pgtc"
)

// ============================================================================
// Helpers
// ============================================================================

// seedTenant uses the superPool (BYPASSRLS) to insert a tenant row, because
// the tenants table has WITH CHECK (is_system_context()) that prevents
// kmuhub_app from inserting new tenants.
func seedTenant(t *testing.T, superPool *pgxpool.Pool, tenantID uuid.UUID) {
	t.Helper()
	_, err := superPool.Exec(context.Background(),
		`INSERT INTO tenants (id, name, created_at, updated_at)
		 VALUES ($1, $2, NOW(), NOW())
		 ON CONFLICT (id) DO NOTHING`,
		tenantID, fmt.Sprintf("Test Tenant %s", tenantID.String()[:8]),
	)
	if err != nil {
		t.Fatalf("seedTenant: %v", err)
	}
}

// seedUser inserts a user without display_name (the column does not exist — only
// first_name + last_name, as per the schema). Uses superPool to bypass RLS.
// Returns the inserted user ID.
func seedUser(t *testing.T, superPool *pgxpool.Pool, tenantID uuid.UUID, firstName, lastName, email string) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	_, err := superPool.Exec(context.Background(),
		`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, 'hash', $4, $5, TRUE, NOW(), NOW())`,
		userID, tenantID, email, firstName, lastName,
	)
	if err != nil {
		t.Fatalf("seedUser %s: %v", email, err)
	}
	return userID
}

// seedEmployeeProfile inserts an hr_employee_profiles row using superPool.
// All nullable string columns receive explicit empty-string values so pgx
// can scan them into Go string fields without NULL coercion errors.
// Returns the profile ID.
func seedEmployeeProfile(t *testing.T, superPool *pgxpool.Pool, tenantID, userID uuid.UUID) uuid.UUID {
	t.Helper()
	profileID := uuid.New()
	_, err := superPool.Exec(context.Background(),
		`INSERT INTO hr_employee_profiles
			(id, tenant_id, user_id, department, position_title, contract_type,
			 work_days_per_week, annual_leave_days, start_date,
			 emergency_contact_name, emergency_contact_phone,
			 address_street, address_city, address_postal_code, address_country,
			 created_at, updated_at)
		 VALUES ($1, $2, $3, 'Engineering', 'Software Engineer', 'full_time',
		         5, 25, $4,
		         '', '',
		         '', '', '', 'DE',
		         NOW(), NOW())`,
		profileID, tenantID, userID, time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("seedEmployeeProfile: %v", err)
	}
	return profileID
}

// ============================================================================
// Tests
// ============================================================================

// TestIntegrationEmployeeGetByID verifies that GetByID returns the correct
// user_name built from first_name + last_name via CONCAT_WS (regression for
// the display_name bug fixed in c2cc98ad).
func TestIntegrationEmployeeGetByID(t *testing.T) {
	appPool, superURL := pgtc.StartPostgres(t)
	superPool := pgtc.SuperPool(t, superURL)
	t.Cleanup(superPool.Close)

	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)

	seedTenant(t, superPool, tenantID)
	userID := seedUser(t, superPool, tenantID, "Anna", "Müller", "anna@example.com")
	profileID := seedEmployeeProfile(t, superPool, tenantID, userID)

	repo := NewPostgresEmployeeRepo(appPool)
	got, err := repo.GetByID(ctx, profileID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.ID != profileID {
		t.Errorf("ID: got %s, want %s", got.ID, profileID)
	}
	if got.UserName != "Anna Müller" {
		t.Errorf("UserName: got %q, want %q", got.UserName, "Anna Müller")
	}
	if got.UserEmail != "anna@example.com" {
		t.Errorf("UserEmail: got %q, want %q", got.UserEmail, "anna@example.com")
	}
}

// TestIntegrationEmployeeGetByUserID verifies the GetByUserID path and that
// user_name is assembled via CONCAT_WS(first_name, last_name) without display_name.
func TestIntegrationEmployeeGetByUserID(t *testing.T) {
	appPool, superURL := pgtc.StartPostgres(t)
	superPool := pgtc.SuperPool(t, superURL)
	t.Cleanup(superPool.Close)

	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)

	seedTenant(t, superPool, tenantID)
	userID := seedUser(t, superPool, tenantID, "Ben", "Schmidt", "ben@example.com")
	seedEmployeeProfile(t, superPool, tenantID, userID)

	repo := NewPostgresEmployeeRepo(appPool)
	got, err := repo.GetByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("GetByUserID: %v", err)
	}

	if got.UserID != userID {
		t.Errorf("UserID: got %s, want %s", got.UserID, userID)
	}
	if got.UserName != "Ben Schmidt" {
		t.Errorf("UserName: got %q, want %q", got.UserName, "Ben Schmidt")
	}
}

// TestIntegrationEmployeeList verifies that List returns employees with correct
// user_name assembled via CONCAT_WS.
func TestIntegrationEmployeeList(t *testing.T) {
	appPool, superURL := pgtc.StartPostgres(t)
	superPool := pgtc.SuperPool(t, superURL)
	t.Cleanup(superPool.Close)

	tenantID := uuid.New()
	ctx := pgtc.TenantCtx(context.Background(), tenantID)

	seedTenant(t, superPool, tenantID)
	userA := seedUser(t, superPool, tenantID, "Clara", "Wagner", "clara@example.com")
	userB := seedUser(t, superPool, tenantID, "David", "Fischer", "david@example.com")
	seedEmployeeProfile(t, superPool, tenantID, userA)
	seedEmployeeProfile(t, superPool, tenantID, userB)

	repo := NewPostgresEmployeeRepo(appPool)
	employees, total, err := repo.List(ctx, EmployeeFilter{TenantID: tenantID})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total < 2 {
		t.Fatalf("List total: got %d, want >= 2", total)
	}
	if len(employees) < 2 {
		t.Fatalf("List len: got %d, want >= 2", len(employees))
	}

	// All returned employees must have non-empty user_name.
	for _, e := range employees {
		if e.UserName == "" {
			t.Errorf("employee %s: UserName is empty (CONCAT_WS fallback broken)", e.ID)
		}
	}
}

// TestIntegrationEmployeeCrossTenantIsolation verifies that tenant B cannot
// read tenant A's employee via List (RLS / tenant filter enforcement).
func TestIntegrationEmployeeCrossTenantIsolation(t *testing.T) {
	appPool, superURL := pgtc.StartPostgres(t)
	superPool := pgtc.SuperPool(t, superURL)
	t.Cleanup(superPool.Close)

	tenantA := uuid.New()
	tenantB := uuid.New()
	ctxB := pgtc.TenantCtx(context.Background(), tenantB)

	seedTenant(t, superPool, tenantA)
	seedTenant(t, superPool, tenantB)

	userA := seedUser(t, superPool, tenantA, "Eve", "Bauer", "eve@example.com")
	userB := seedUser(t, superPool, tenantB, "Frank", "Klein", "frank@example.com")
	seedEmployeeProfile(t, superPool, tenantA, userA)
	seedEmployeeProfile(t, superPool, tenantB, userB)

	repo := NewPostgresEmployeeRepo(appPool)

	// Tenant B listing its own employees must NOT include tenant A's employee.
	employeesB, _, err := repo.List(ctxB, EmployeeFilter{TenantID: tenantB})
	if err != nil {
		t.Fatalf("List B: %v", err)
	}
	for _, e := range employeesB {
		if e.TenantID == tenantA {
			t.Errorf("cross-tenant leak: tenant B sees employee %s from tenant A", e.ID)
		}
	}
}
