package thread

// name_fallback_db_test.go is the regression test for the CONCAT_WS blank
// name bug: users.first_name/last_name are NOT NULL DEFAULT '', never NULL,
// so CONCAT_WS(' ', '', '') returns a single space rather than NULL, and
// NULLIF(' ', '') does not recognize that as blank -- the email COALESCE
// fallback in ListThreadMessages' author_name join never fired for a
// blank-named author. Fails against the pre-fix query
// (COALESCE(NULLIF(CONCAT_WS(...), ''), ...)) and passes only with a TRIM()
// around the CONCAT_WS before the NULLIF.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestListThreadMessages_BlankNamedAuthorFallsBackToEmail(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Inbox Thread Name Fallback Tenant")

	ownerID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantID, "email": "owner-" + uuid.NewString() + "@example.test",
		"password_hash": "hash", "first_name": "Owner", "last_name": "Inbox",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", ownerID) })

	authorEmail := "blank-author-" + uuid.NewString() + "@example.test"
	authorID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id": tenantID, "email": authorEmail,
		"password_hash": "hash", "first_name": "", "last_name": "",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", authorID) })

	messageID := testutil.SeedRow(t, pool, "inbox_messages", map[string]any{
		"tenant_id": tenantID, "user_id": ownerID, "channel": "email",
		"source_id": "src-" + uuid.NewString(), "received_at": time.Now().UTC(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inbox_messages", messageID) })

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	if err := repo.AppendThreadMessage(ctx, &ThreadMessage{
		TenantID: tenantID, MessageID: messageID, AuthorID: &authorID,
		Direction: DirectionOutbound, Body: "Antwort ohne Namen",
	}); err != nil {
		t.Fatalf("AppendThreadMessage: %v", err)
	}

	got, err := repo.ListThreadMessages(ctx, tenantID, messageID)
	if err != nil {
		t.Fatalf("ListThreadMessages: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListThreadMessages: got %d message(s), want 1", len(got))
	}
	if got[0].AuthorName != authorEmail {
		t.Errorf("AuthorName: got %q, want the email fallback %q", got[0].AuthorName, authorEmail)
	}
}
