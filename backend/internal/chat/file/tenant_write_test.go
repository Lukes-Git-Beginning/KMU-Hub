package file

// Writes into chat_files must land in — and stay visible only to — the
// caller's tenant. chat_files carries tenant_id NOT NULL since migration
// 000115 and was already RLS-enabled in the mig 000122 sweep (unlike
// channels/messages, which were dropped from that sweep — see
// internal/chat/channel/tenant_write_test.go). This test verifies the RLS
// backstop is actually live: CreateFile is visible only to its own tenant,
// and DeleteFile called from a foreign tenant context is a silent no-op —
// RLS scopes the UPDATE, not only the SELECT.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestChatFileWrites_LandInCallerTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Chat File Write Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Chat File Write Other Tenant")

	userA := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantOwn,
		"email":         fmt.Sprintf("chat-file-write-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userA)

	channelID := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "chat-file-write-test",
		"created_by": userA,
	})
	defer testutil.CleanupRow(t, pool, "channels", channelID)

	repo := NewPostgresRepository(pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenantOwn)
	ctxOther := testutil.WithTenantCtx(context.Background(), tenantOther)
	sysCtx := testutil.WithSystemCtx(context.Background())

	// CreateFile
	f := &models.ChatFile{
		ID: uuid.New(), TenantID: tenantOwn, ChannelID: channelID,
		Filename: "report.pdf", MimeType: "application/pdf", FileSize: 1024,
		StorageKey: fmt.Sprintf("channels/%s/files/report.pdf", channelID),
		UploadedBy: userA, IsDeleted: false, CreatedAt: time.Now().UTC(),
	}
	if err := repo.CreateFile(ctxOwn, f); err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "chat_files", f.ID)
	testutil.AssertRowCount(t, pool, ctxOwn, "chat_files", f.ID, 1)
	testutil.AssertRowCount(t, pool, ctxOther, "chat_files", f.ID, 0)

	// DeleteFile from a foreign tenant must be a no-op — RLS scopes the
	// UPDATE, not only the SELECT.
	if err := repo.DeleteFile(ctxOther, f.ID); err != nil {
		t.Fatalf("DeleteFile (foreign tenant): %v", err)
	}
	var isDeleted bool
	if err := pool.QueryRow(sysCtx, "SELECT is_deleted FROM chat_files WHERE id = $1", f.ID).Scan(&isDeleted); err != nil {
		t.Fatalf("read back is_deleted after foreign delete: %v", err)
	}
	if isDeleted {
		t.Fatalf("DeleteFile: foreign tenant context soft-deleted a file it cannot see")
	}

	// DeleteFile from the owning tenant flips is_deleted.
	if err := repo.DeleteFile(ctxOwn, f.ID); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	if err := pool.QueryRow(sysCtx, "SELECT is_deleted FROM chat_files WHERE id = $1", f.ID).Scan(&isDeleted); err != nil {
		t.Fatalf("read back is_deleted: %v", err)
	}
	if !isDeleted {
		t.Fatalf("DeleteFile: expected is_deleted = true, got false")
	}
}
