package contactlink

// Cross-tenant isolation for email_contact_links after migration 000269 made
// tenant_id NOT NULL (it was added nullable in 000110 and never backfilled)
// and put the table under RLS.
//
// The repository already passes a tenant to every read and to the delete, so
// this test exercises the write path through the real Create method: that is
// where a silently omitted tenant_id would show up, and it is the reason the
// column can be NOT NULL at all.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// seedMessageAndContact builds the FK chain email_contact_links needs:
// users -> email_accounts -> email_folders -> email_messages, plus a contact.
// All rows belong to the given tenant; the seeds run in system context so RLS
// does not reject the fixture itself.
func seedMessageAndContact(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) (uuid.UUID, uuid.UUID) {
	t.Helper()

	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("link-test-%s@example.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", userID) })

	accountID := testutil.SeedRow(t, pool, "email_accounts", map[string]any{
		"tenant_id":          tenantID,
		"user_id":            userID,
		"email_address":      fmt.Sprintf("acct-%s@example.com", uuid.New().String()[:8]),
		"imap_host":          "imap.example.com",
		"smtp_host":          "smtp.example.com",
		"username":           "testuser",
		"password_encrypted": "enc:dummy",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_accounts", accountID) })

	folderID := testutil.SeedRow(t, pool, "email_folders", map[string]any{
		"tenant_id":  tenantID,
		"account_id": accountID,
		"name":       "INBOX",
		"imap_name":  "INBOX",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_folders", folderID) })

	messageID := testutil.SeedRow(t, pool, "email_messages", map[string]any{
		"tenant_id":  tenantID,
		"account_id": accountID,
		"folder_id":  folderID,
		"uid":        42,
		"from_email": "sender@example.com",
		"date":       time.Now().UTC(),
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "email_messages", messageID) })

	contactID := testutil.SeedRow(t, pool, "contacts", map[string]any{
		"tenant_id":  tenantID,
		"first_name": "Link",
		"last_name":  "Target",
		"created_by": userID,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "contacts", contactID) })

	return messageID, contactID
}

func TestTenantIsolation_EmailContactLinks_DB(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	// Registered before the fixture cleanups so it runs after them: t.Cleanup
	// is LIFO, and seedMessageAndContact's cleanups still need a live pool.
	// A plain `defer pool.Close()` would close it first and leave the whole
	// FK chain behind in the database.
	t.Cleanup(func() { pool.Close() })

	owner, other := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, owner, "contactlink owner")
	testutil.EnsureTenant(t, pool, other, "contactlink other")

	messageID, contactID := seedMessageAndContact(t, pool, owner)

	repo := NewPostgresRepository(pool)
	ownerCtx := testutil.WithTenantCtx(context.Background(), owner)
	otherCtx := testutil.WithTenantCtx(context.Background(), other)

	link := &models.EmailContactLink{
		ID:        uuid.New(),
		TenantID:  owner,
		MessageID: messageID,
		ContactID: contactID,
		LinkType:  models.LinkTypeManual,
		CreatedAt: time.Now().UTC(),
	}
	if err := repo.Create(ownerCtx, link); err != nil {
		t.Fatalf("create as owner: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "email_contact_links", link.ID)

	// Visible to its own tenant...
	testutil.AssertRowCount(t, pool, ownerCtx, "email_contact_links", link.ID, 1)
	// ...and to nobody else.
	testutil.AssertRowCount(t, pool, otherCtx, "email_contact_links", link.ID, 0)

	links, err := repo.GetByMessageID(ownerCtx, messageID, owner)
	if err != nil {
		t.Fatalf("get by message as owner: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("owner expected 1 link for its message, got %d", len(links))
	}

	// The foreign tenant sees nothing even when it asks for the exact
	// message and passes its own tenant — RLS and the predicate agree.
	foreign, err := repo.GetByMessageID(otherCtx, messageID, other)
	if err != nil {
		t.Fatalf("get by message as other tenant: %v", err)
	}
	if len(foreign) != 0 {
		t.Fatalf("cross-tenant read leaked %d link(s)", len(foreign))
	}
}
