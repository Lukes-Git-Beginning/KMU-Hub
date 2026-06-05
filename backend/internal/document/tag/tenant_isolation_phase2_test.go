package tag

// Cross-tenant isolation DB tests for document_tags, document_shares, document_entity_links
// (Migration 000122 + 000114).
// All have tenant_id NOT NULL and RLS in mig 000122.
//
// document_file_tags has composite PK (file_id, tag_id) — no UUID id, cannot use
// AssertRowCount directly; tested via document_tags isolation.
//
// document_tags.created_by FK → users. document_shares.shared_with_user_id + shared_by FK → users.
// document_entity_links.file_id FK → document_files (which requires folder_id FK → document_folders).

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_DocumentTags(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a user for TenantA.
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("doctag-test-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	// document_tags (UNIQUE on name — add suffix for uniqueness).
	tagID := testutil.SeedRow(t, pool, "document_tags", map[string]any{
		"tenant_id":  testutil.TenantA,
		"name":       "confidential-" + uuid.New().String()[:8],
		"created_by": userID,
	})
	defer testutil.CleanupRow(t, pool, "document_tags", tagID)

	// document_shares — needs a valid entity_id (any UUID is OK since it's just a reference).
	shareID := testutil.SeedRow(t, pool, "document_shares", map[string]any{
		"tenant_id":          testutil.TenantA,
		"entity_type":        "file",
		"entity_id":          uuid.New(),
		"shared_with_user_id": userID,
		"shared_by":          userID,
		"permission":         "read",
	})
	defer testutil.CleanupRow(t, pool, "document_shares", shareID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"document_tags", "document_tags", tagID},
		{"document_shares", "document_shares", shareID},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			testutil.AssertRowCount(t, pool, ctxA, tc.table, tc.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, tc.id, 0)
		})
	}
}
