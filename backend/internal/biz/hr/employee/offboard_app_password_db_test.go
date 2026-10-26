package employee

// Offboarding revokes the leaver's CalDAV/CardDAV app passwords in the same
// transaction as the login lock (decision 2026-10-25). The lookup behind Basic
// Auth already refuses inactive users, so the test re-activates the account
// afterwards: only a REVOKED password stays dead then.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/caldav"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func seedAppPassword(t *testing.T, pool *pgxpool.Pool, tenantID, userID uuid.UUID, label string) {
	t.Helper()
	id := testutil.SeedRow(t, pool, "app_specific_passwords", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "user_id": userID,
		"label": label, "password_hash": "hash", "password_prefix": "abcd",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "app_specific_passwords", id) })
}

func TestOffboard_RevokesLeaversAppPasswordsOnly(t *testing.T) {
	pool, tenantID, ctx := newOffboardFixture(t)
	svc := newOffboardSvc(pool)
	repo := caldav.NewPostgresAppPasswordRepository(pool)

	adminID := seedOffboardUser(t, pool, tenantID, "admin-"+uuid.NewString()+"@example.test")
	grantRoleAdmin(t, pool, tenantID, adminID)
	colleagueID := seedOffboardUser(t, pool, tenantID, "colleague-"+uuid.NewString()+"@example.test")
	leaverID := seedOffboardUser(t, pool, tenantID, "leaver-"+uuid.NewString()+"@example.test")
	profileID := seedOffboardProfile(t, pool, tenantID, leaverID, nil)

	seedAppPassword(t, pool, tenantID, leaverID, "phone")
	seedAppPassword(t, pool, tenantID, leaverID, "laptop")
	seedAppPassword(t, pool, tenantID, colleagueID, "phone")

	active := func(userID uuid.UUID) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM app_specific_passwords
			  WHERE user_id = $1 AND revoked_at IS NULL`, userID).Scan(&n); err != nil {
			t.Fatalf("count active app passwords: %v", err)
		}
		return n
	}
	if got := active(leaverID); got != 2 {
		t.Fatalf("precondition: leaver should hold 2 active passwords, got %d", got)
	}

	if _, err := svc.OffboardEmployee(ctx, tenantID, profileID, adminID, defaultOffboardInput()); err != nil {
		t.Fatalf("OffboardEmployee: %v", err)
	}

	if got := active(leaverID); got != 0 {
		t.Errorf("leaver still has %d active app passwords", got)
	}
	if got := active(colleagueID); got != 1 {
		t.Errorf("colleague's app password was touched: %d active, want 1", got)
	}

	// Flip the account back on, as a mistaken re-activation would: the revoked
	// passwords must not come back with it.
	if _, err := pool.Exec(ctx, `UPDATE users SET is_active = true WHERE id = $1`, leaverID); err != nil {
		t.Fatalf("re-activate user: %v", err)
	}
	pws, err := repo.FindActiveByUser(context.Background(), leaverID)
	if err != nil {
		t.Fatalf("FindActiveByUser: %v", err)
	}
	if len(pws) != 0 {
		t.Errorf("Basic Auth lookup returns %d passwords for a re-activated leaver, want 0", len(pws))
	}
}
