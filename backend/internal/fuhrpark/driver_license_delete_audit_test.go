package fuhrpark

// Deleting a Fuehrerscheinkontrolle stays allowed (typo correction), but the
// row is the Halterpflicht proof — each delete must leave an audit event with
// the control state that vanished. Real repository, real audit service.

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/security/audit"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestService_DeleteDriverLicense_WritesAuditEvent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantA, tenantB := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantA, "License Audit Tenant A")
	testutil.EnsureTenant(t, pool, tenantB, "License Audit Tenant B")

	sysCtx := testutil.WithSystemCtx(context.Background())
	driver, actor := uuid.New(), uuid.New()
	for _, u := range []struct {
		id    uuid.UUID
		email string
	}{{driver, "fuhrpark-audit-driver-" + driver.String() + "@test.local"}, {actor, "fuhrpark-audit-actor-" + actor.String() + "@test.local"}} {
		if _, err := pool.Exec(sysCtx,
			`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name)
			 VALUES ($1, $2, $3, 'x', 'Fuhrpark', 'Audit')`, u.id, tenantA, u.email); err != nil {
			t.Fatalf("seed user: %v", err)
		}
		// The actor stays behind: audit_log.user_id references it and the log is append-only.
		if u.id == driver {
			defer testutil.CleanupRow(t, pool, "users", u.id)
		}
	}

	repo := NewPostgresRepository(pool)
	svc := NewService(repo).WithAuditLogger(audit.NewService(audit.NewPostgresRepository(pool)))
	ctxA := context.WithValue(testutil.WithTenantCtx(context.Background(), tenantA), middleware.UserIDKey, actor.String())
	ctxB := testutil.WithTenantCtx(context.Background(), tenantB)

	checked := time.Now().UTC().Truncate(24 * time.Hour)
	expiry := checked.AddDate(10, 0, 0)
	notes := "Kontrolle durch Disponent"
	lic, err := repo.CreateDriverLicense(ctxA, DriverLicense{
		TenantID:         tenantA,
		DriverID:         driver,
		LicenseClasses:   []string{"B", "BE"},
		ExpiryDate:       &expiry,
		CheckedAt:        checked,
		NextCheckDueDate: checked.AddDate(0, 6, 0),
		Notes:            &notes,
	})
	if err != nil {
		t.Fatalf("CreateDriverLicense: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "driver_licenses", lic.ID)

	auditRows := func(ctx context.Context) []string {
		t.Helper()
		rows, err := pool.Query(ctx,
			`SELECT details::text FROM audit_log WHERE action = 'fuhrpark.driver_license_deleted' AND target = $1`, lic.ID.String())
		if err != nil {
			t.Fatalf("query audit_log: %v", err)
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var d string
			if err := rows.Scan(&d); err != nil {
				t.Fatalf("scan: %v", err)
			}
			out = append(out, d)
		}
		return out
	}

	t.Run("foreign tenant delete is refused and audits nothing", func(t *testing.T) {
		if err := svc.DeleteDriverLicense(ctxB, tenantB, lic.ID); !errors.Is(err, ErrDriverLicenseNotFound) {
			t.Fatalf("expected ErrDriverLicenseNotFound, got %v", err)
		}
		if got := auditRows(testutil.WithTenantCtx(context.Background(), tenantA)); len(got) != 0 {
			t.Fatalf("expected no audit event, got %v", got)
		}
		testutil.AssertRowCount(t, pool, ctxA, "driver_licenses", lic.ID, 1)
	})

	t.Run("delete writes one event with the full control state", func(t *testing.T) {
		if err := svc.DeleteDriverLicense(ctxA, tenantA, lic.ID); err != nil {
			t.Fatalf("DeleteDriverLicense: %v", err)
		}
		testutil.AssertRowCount(t, pool, ctxA, "driver_licenses", lic.ID, 0)

		got := auditRows(testutil.WithTenantCtx(context.Background(), tenantA))
		if len(got) != 1 {
			t.Fatalf("expected exactly one audit event, got %d", len(got))
		}
		var d map[string]any
		if err := json.Unmarshal([]byte(got[0]), &d); err != nil {
			t.Fatalf("details not JSON: %v", err)
		}
		if d["driver_id"] != driver.String() {
			t.Errorf("driver_id = %v, want %s", d["driver_id"], driver)
		}
		if d["notes"] != notes {
			t.Errorf("notes = %v, want %q", d["notes"], notes)
		}
		if d["expiry_date"] != expiry.Format("2006-01-02") {
			t.Errorf("expiry_date = %v", d["expiry_date"])
		}
		if d["next_check_due_date"] != checked.AddDate(0, 6, 0).Format("2006-01-02") {
			t.Errorf("next_check_due_date = %v", d["next_check_due_date"])
		}
		if classes, _ := d["license_classes"].([]any); len(classes) != 2 {
			t.Errorf("license_classes = %v", d["license_classes"])
		}

		var userID uuid.UUID
		if err := pool.QueryRow(testutil.WithTenantCtx(context.Background(), tenantA),
			`SELECT user_id FROM audit_log WHERE action = 'fuhrpark.driver_license_deleted' AND target = $1`, lic.ID.String()).Scan(&userID); err != nil {
			t.Fatalf("read actor: %v", err)
		}
		if userID != actor {
			t.Errorf("actor = %s, want %s", userID, actor)
		}
	})

	t.Run("second delete is not found and audits nothing more", func(t *testing.T) {
		if err := svc.DeleteDriverLicense(ctxA, tenantA, lic.ID); !errors.Is(err, ErrDriverLicenseNotFound) {
			t.Fatalf("expected ErrDriverLicenseNotFound, got %v", err)
		}
		if got := auditRows(testutil.WithTenantCtx(context.Background(), tenantA)); len(got) != 1 {
			t.Fatalf("expected still one audit event, got %d", len(got))
		}
	})
}
