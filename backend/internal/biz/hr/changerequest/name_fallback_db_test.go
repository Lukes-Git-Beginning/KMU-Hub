package changerequest

// name_fallback_db_test.go is the regression test for the CONCAT_WS blank
// name bug: users.first_name/last_name are NOT NULL DEFAULT '', never NULL,
// so CONCAT_WS(' ', '', '') returns a single space rather than NULL, and
// NULLIF(' ', '') does not recognize that as blank -- the email COALESCE
// fallback in selectColumns' two name joins (user_name, decided_by_name)
// never fired for a blank-named proposer or decider. Fails against the
// pre-fix query (COALESCE(NULLIF(CONCAT_WS(...), ''), ...)) and passes only
// with a TRIM() around the CONCAT_WS before the NULLIF, mirroring
// employee/postgres_repository_db_test.go.

import (
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestGetByID_BlankNamedProposerAndDeciderFallBackToEmail(t *testing.T) {
	pool, tenantID, ctx := newFixture(t)

	proposerEmail := "blank-proposer-" + uuid.NewString() + "@example.test"
	proposerID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantID, "email": proposerEmail,
		"password_hash": "hash", "first_name": "", "last_name": "",
		"is_active": true,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", proposerID) })

	deciderEmail := "blank-decider-" + uuid.NewString() + "@example.test"
	deciderID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantID, "email": deciderEmail,
		"password_hash": "hash", "first_name": "", "last_name": "",
		"is_active": true,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", deciderID) })

	reqID := testutil.SeedRow(t, pool, "hr_profile_change_requests", map[string]any{
		"tenant_id": tenantID, "user_id": proposerID,
		"drawer": "personal", "field": "addressCity", "field_label": "Wohnort",
		"new_value": "Berlin", "status": "approved", "decided_by": deciderID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "hr_profile_change_requests", reqID) })

	repo := NewPostgresRepository(pool)
	got, err := repo.GetByID(ctx, tenantID, reqID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.UserName != proposerEmail {
		t.Errorf("UserName: got %q, want the email fallback %q", got.UserName, proposerEmail)
	}
	if got.DecidedByName != deciderEmail {
		t.Errorf("DecidedByName: got %q, want the email fallback %q", got.DecidedByName, deciderEmail)
	}
}
