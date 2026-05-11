package account

// Cross-tenant isolation tests for email tables (Migration 000124).
// email_accounts, email_folders, email_messages, email_attachments, email_signatures
// all have tenant_id NOT NULL after backfill.
//
// email_accounts has FK user_id → users(id). We seed a minimal user in system-context.
// email_messages requires account_id + folder_id FKs.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Email(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a user for TenantA (required by email_accounts FK).
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("email-test-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	// email_accounts.
	accountID := testutil.SeedRow(t, pool, "email_accounts", map[string]any{
		"tenant_id":         testutil.TenantA,
		"user_id":           userID,
		"email_address":     fmt.Sprintf("acct-%s@example.com", uuid.New().String()[:8]),
		"imap_host":         "imap.example.com",
		"smtp_host":         "smtp.example.com",
		"username":          "testuser",
		"password_encrypted": "enc:dummy",
	})
	defer testutil.CleanupRow(t, pool, "email_accounts", accountID)

	// email_folders.
	folderID := testutil.SeedRow(t, pool, "email_folders", map[string]any{
		"tenant_id":  testutil.TenantA,
		"account_id": accountID,
		"name":       "INBOX",
		"imap_name":  "INBOX",
	})
	defer testutil.CleanupRow(t, pool, "email_folders", folderID)

	// email_messages (requires account_id + folder_id + uid + from_email + date).
	msgID := testutil.SeedRow(t, pool, "email_messages", map[string]any{
		"tenant_id":  testutil.TenantA,
		"account_id": accountID,
		"folder_id":  folderID,
		"uid":        42,
		"from_email": "sender@example.com",
		"date":       "2026-05-11T10:00:00Z",
	})
	defer testutil.CleanupRow(t, pool, "email_messages", msgID)

	// email_attachments.
	attachID := testutil.SeedRow(t, pool, "email_attachments", map[string]any{
		"tenant_id":    testutil.TenantA,
		"message_id":   msgID,
		"filename":     "document.pdf",
		"content_type": "application/pdf",
		"minio_key":    "emails/" + uuid.New().String(),
	})
	defer testutil.CleanupRow(t, pool, "email_attachments", attachID)

	// email_signatures (user-scoped).
	sigID := testutil.SeedRow(t, pool, "email_signatures", map[string]any{
		"tenant_id": testutil.TenantA,
		"user_id":   userID,
	})
	defer testutil.CleanupRow(t, pool, "email_signatures", sigID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"email_accounts", "email_accounts", accountID},
		{"email_folders", "email_folders", folderID},
		{"email_messages", "email_messages", msgID},
		{"email_attachments", "email_attachments", attachID},
		{"email_signatures", "email_signatures", sigID},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testutil.AssertRowCount(t, pool, ctxA, tc.table, tc.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, tc.id, 0)
		})
	}
}
