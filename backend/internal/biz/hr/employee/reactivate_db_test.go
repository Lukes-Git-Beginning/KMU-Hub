package employee

// Deactivate and reactivate from the team page, against a real database: the
// deactivation is an offboard with exit type "other" (the CHECK from 000282
// has to accept it), and the reactivation has to undo the whole cascade — an
// active personnel file behind a locked account, or an unlocked account
// without any role, are both states nobody notices until the person cannot
// work.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func deactivateInput() OffboardInput {
	return OffboardInput{ExitDate: time.Now().UTC().Truncate(24 * time.Hour), ExitType: models.ExitTypeOther}
}

func userRoleNames(t *testing.T, pool *pgxpool.Pool, ctx context.Context, userID uuid.UUID) []string {
	t.Helper()
	rows, err := pool.Query(ctx,
		`SELECT r.name || CASE WHEN r.tenant_id IS NULL THEN '' ELSE ':custom' END
		   FROM user_roles ur JOIN roles r ON r.id = ur.role_id
		  WHERE ur.user_id = $1 ORDER BY 1`, userID)
	if err != nil {
		t.Fatalf("read roles: %v", err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatalf("scan role: %v", err)
		}
		names = append(names, n)
	}
	return names
}

func TestDeactivateAndReactivate_RealRepository(t *testing.T) {
	pool, tenantID, ctx := newOffboardFixture(t)
	svc := newOffboardSvc(pool)

	adminID := seedOffboardUser(t, pool, tenantID, "admin-"+uuid.NewString()+"@example.test")
	grantRoleAdmin(t, pool, tenantID, adminID)
	userID := seedOffboardUser(t, pool, tenantID, "member-"+uuid.NewString()+"@example.test")
	grantRoleAdmin(t, pool, tenantID, userID) // a wider role the offboard removes
	profileID := seedOffboardProfile(t, pool, tenantID, userID, nil)

	// Deactivate = offboard with exit type "other".
	off, err := svc.OffboardEmployee(ctx, tenantID, profileID, adminID, deactivateInput())
	if err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	if off.Status != models.EmployeeStatusInactive || off.ExitType != models.ExitTypeOther {
		t.Fatalf("after deactivate: status=%q exit_type=%q, want inactive/other", off.Status, off.ExitType)
	}

	// A foreign tenant cannot reach the profile.
	foreignID := uuid.New()
	testutil.EnsureTenant(t, pool, foreignID, "Reactivate Foreign")
	foreignCtx := testutil.WithTenantCtx(context.Background(), foreignID)
	if _, err := svc.ReactivateEmployee(foreignCtx, foreignID, profileID, adminID); !errors.Is(err, ErrEmployeeNotFound) {
		t.Fatalf("foreign reactivate: err=%v, want ErrEmployeeNotFound", err)
	}

	// No seat left: refused, nothing changes. The admin is the only active
	// user, so limit 1 is full.
	if _, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
		`UPDATE tenants SET seat_limit = 1 WHERE id = $1`, tenantID); err != nil {
		t.Fatalf("set seat limit: %v", err)
	}
	if _, err := svc.ReactivateEmployee(ctx, tenantID, profileID, adminID); !errors.Is(err, ErrSeatLimitReached) {
		t.Fatalf("reactivate without seat: err=%v, want ErrSeatLimitReached", err)
	}
	var status string
	var active bool
	if err := pool.QueryRow(ctx,
		`SELECT ep.status, u.is_active FROM hr_employee_profiles ep JOIN users u ON u.id = ep.user_id
		  WHERE ep.id = $1`, profileID).Scan(&status, &active); err != nil {
		t.Fatalf("read state: %v", err)
	}
	if status != "inactive" || active {
		t.Fatalf("refused reactivate changed state: status=%q is_active=%v", status, active)
	}
	if _, err := pool.Exec(testutil.WithSystemCtx(context.Background()),
		`UPDATE tenants SET seat_limit = 2 WHERE id = $1`, tenantID); err != nil {
		t.Fatalf("raise seat limit: %v", err)
	}

	got, err := svc.ReactivateEmployee(ctx, tenantID, profileID, adminID)
	if err != nil {
		t.Fatalf("reactivate: %v", err)
	}
	if got.Status != models.EmployeeStatusActive || got.ExitDate != nil || got.ExitType != "" || got.ExitReason != "" {
		t.Fatalf("after reactivate: status=%q exit_date=%v exit_type=%q reason=%q", got.Status, got.ExitDate, got.ExitType, got.ExitReason)
	}
	if err := pool.QueryRow(ctx, `SELECT is_active FROM users WHERE id = $1`, userID).Scan(&active); err != nil {
		t.Fatalf("read user: %v", err)
	}
	if !active {
		t.Fatal("login still locked after reactivate")
	}
	// The preset member role, not the wider custom role from before.
	if roles := userRoleNames(t, pool, ctx, userID); len(roles) != 1 || roles[0] != "member" {
		t.Fatalf("roles after reactivate = %v, want [member]", roles)
	}

	if _, err := svc.ReactivateEmployee(ctx, tenantID, profileID, adminID); !errors.Is(err, ErrNotOffboarded) {
		t.Fatalf("second reactivate: err=%v, want ErrNotOffboarded", err)
	}

	// The repository's own guard holds without the service pre-check.
	if _, err := NewPostgresEmployeeRepo(pool).Reactivate(ctx, tenantID, profileID, userID); !errors.Is(err, ErrNotOffboarded) {
		t.Fatalf("repo reactivate on active profile: err=%v, want ErrNotOffboarded", err)
	}
}
