package gobdarchive

// VerifyIntegrity against the real repository as kmuhub_app: the
// integrity_check event must survive the WORM grants of migration 000315 and
// land with its metadata, and a foreign tenant must not reach the document at
// all. Rows are not cleaned up, see postgres_repository_db_test.go.

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestVerifyIntegrity_DB_MismatchPersistsEventAndForeignTenantSeesNothing(t *testing.T) {
	t.Parallel()
	pool, tenantID, ctx := newGobdFixture(t)
	store := newMockFileStore()
	svc := NewService(NewPostgresRepository(pool), store, nil)

	content := []byte("archived receipt bytes")
	doc, err := svc.ArchiveDocument(ctx, ArchiveInput{
		TenantID:         tenantID,
		DocType:          models.GobdDocTypeReceipt,
		OriginalFilename: "receipt.pdf",
		MimeType:         "application/pdf",
		Content:          bytes.NewReader(content),
		Size:             int64(len(content)),
		ArchivedBy:       uuid.New(),
	})
	require.NoError(t, err)

	foreignTenant := uuid.New()
	testutil.EnsureTenant(t, pool, foreignTenant, "GoBD Foreign Tenant")
	foreignCtx := testutil.WithTenantCtx(context.Background(), foreignTenant)
	require.ErrorIs(t, svc.VerifyIntegrity(foreignCtx, foreignTenant, doc.ID, uuid.New()), ErrDocumentNotFound)
	assert.Zero(t, store.downloads, "a foreign tenant must not trigger a download")

	store.uploaded[doc.StorageKey] = []byte("swapped object")
	require.ErrorIs(t, svc.VerifyIntegrity(ctx, tenantID, doc.ID, uuid.New()), ErrIntegrityMismatch)

	events, err := svc.ListEvents(ctx, tenantID, doc.ID)
	require.NoError(t, err)
	var checks []*models.GobdDocumentEvent
	for _, ev := range events {
		if ev.EventType == models.GobdEventTypeIntegrityCheck {
			checks = append(checks, ev)
		}
	}
	require.Len(t, checks, 1)

	var meta map[string]any
	require.NoError(t, json.Unmarshal(checks[0].Metadata, &meta))
	assert.Equal(t, doc.SHA256, meta["expected_sha256"])
	assert.NotEqual(t, doc.SHA256, meta["actual_sha256"])
	assert.Equal(t, false, meta["match"])
}
