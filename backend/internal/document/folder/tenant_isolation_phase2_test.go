package folder

// Cross-tenant isolation DB tests for document_folders (Migration 000122).
// document_folders.tenant_id is NOT NULL and RLS was enabled in mig 000122.
// created_by FK → users(id); we seed a minimal user first.

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_DocumentFolders(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a user for TenantA.
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("docfolder-test-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	folderID := testutil.SeedRow(t, pool, "document_folders", map[string]any{
		"tenant_id":  testutil.TenantA,
		"name":       "Team Documents",
		"space_type": "team",
		"space_id":   uuid.New(),
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "document_folders", folderID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	testutil.AssertRowCount(t, pool, ctxA, "document_folders", folderID, 1)
	testutil.AssertRowCount(t, pool, ctxB, "document_folders", folderID, 0)
}
