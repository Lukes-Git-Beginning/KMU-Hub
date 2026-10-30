package gdpr

// Integration coverage for the chat_files gap in ChatErasureHandler
// (fix-erasure-chat-files-of-erased-user): uploads of the erased user stay
// visible to the channel unless they are flagged is_deleted.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestChatErasureHandler_ExecuteErasure_HidesUploadedFiles(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantOwn := uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Chat Erasure Files Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantOwn)

	userID := seedExportUser(t, pool, tenantOwn, "chat-erasure-files")
	defer testutil.CleanupRow(t, pool, "users", userID)
	colleagueID := seedExportUser(t, pool, tenantOwn, "chat-erasure-files-colleague")
	defer testutil.CleanupRow(t, pool, "users", colleagueID)

	channelID := testutil.SeedRow(t, pool, "channels", map[string]any{
		"tenant_id":  tenantOwn,
		"name":       "projekt-dateien",
		"created_by": colleagueID,
	})
	defer testutil.CleanupRow(t, pool, "channels", channelID)

	newFile := func(uploader uuid.UUID, name string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "chat_files", map[string]any{
			"tenant_id": tenantOwn, "channel_id": channelID, "uploaded_by": uploader, "filename": name,
			"mime_type": "image/png", "file_size": 10, "storage_key": "c/" + uuid.NewString(),
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "chat_files", id) })
		return id
	}
	own1 := newFile(userID, "baustelle-1.png")
	own2 := newFile(userID, "baustelle-2.png")
	foreign := newFile(colleagueID, "kollege.png")

	h := NewChatErasureHandler(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantOwn)

	preview, err := h.PreviewErasure(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, 2, preview.RecordCount, "two visible uploads")

	affected, err := h.ExecuteErasure(ctx, userID, erasureLabel, ErasureAnonymize)
	require.NoError(t, err)
	assert.Equal(t, 2, affected, "both uploads hidden")

	deleted := func(id uuid.UUID) (bool, bool) {
		var isDeleted, hasStamp bool
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT is_deleted, deleted_at IS NOT NULL FROM chat_files WHERE id = $1`, id,
		).Scan(&isDeleted, &hasStamp))
		return isDeleted, hasStamp
	}
	for _, id := range []uuid.UUID{own1, own2} {
		d, stamp := deleted(id)
		assert.True(t, d, "the subject's upload must be hidden")
		assert.True(t, stamp, "deleted_at must be set")
	}
	d, stamp := deleted(foreign)
	assert.False(t, d, "a colleague's upload must stay visible")
	assert.False(t, stamp)

	affected, err = h.ExecuteErasure(ctx, userID, erasureLabel, ErasureAnonymize)
	require.NoError(t, err)
	assert.Equal(t, 0, affected, "second run finds nothing left to hide")
}
