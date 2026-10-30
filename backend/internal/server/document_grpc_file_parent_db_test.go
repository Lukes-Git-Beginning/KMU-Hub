package server

// DB-backed proof for fix-document-comment-on-trashed-file: a comment is refused on a file in
// the trash, and registering an uploaded file needs a folder of the caller's own tenant. Both
// used to reach the insert (201 for the comment, FK 500 or a foreign folder for the register).
// Runs under kmuhub_app (RLS active).

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/document/file"
	"github.com/kmuhub/kmuhub/internal/document/folder"
	"github.com/kmuhub/kmuhub/internal/testutil"
	documentv1 "github.com/kmuhub/kmuhub/proto/document/v1"
)

func TestDocumentGRPCServer_FileParents_CommentAndRegister(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "File Parent Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })
	otherTenant := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenant, "File Parent Foreign Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", otherTenant) })

	user := seedSharedWithMeUser(t, pool, tenantID, "fpuser")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", user) })
	foreignUser := seedSharedWithMeUser(t, pool, otherTenant, "fpforeign")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", foreignUser) })

	seedFolder := func(tenant, creator uuid.UUID) uuid.UUID {
		id := testutil.SeedRow(t, pool, "document_folders", map[string]any{
			"id": uuid.New(), "tenant_id": tenant, "name": "Ordner",
			"space_type": "personal", "space_id": creator, "created_by": creator,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "document_folders", id) })
		return id
	}
	seedFile := func(folderID uuid.UUID, deleted bool) uuid.UUID {
		id := testutil.SeedRow(t, pool, "document_files", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "folder_id": folderID,
			"filename": "vertrag.pdf", "mime_type": "application/pdf", "file_size": 16,
			"storage_key": "documents/fp/" + uuid.NewString(), "owner_id": user, "is_deleted": deleted,
		})
		t.Cleanup(func() {
			ctx := testutil.WithSystemCtx(ctxWithTenant(tenantID))
			_, _ = pool.Exec(ctx, `DELETE FROM document_file_comments WHERE file_id = $1`, id)
			_, _ = pool.Exec(ctx, `DELETE FROM document_file_versions WHERE file_id = $1`, id)
			_, _ = pool.Exec(ctx, `DELETE FROM document_file_activity WHERE file_id = $1`, id)
			testutil.CleanupRow(t, pool, "document_files", id)
		})
		return id
	}

	ownFolder := seedFolder(tenantID, user)
	foreignFolder := seedFolder(otherTenant, foreignUser)
	liveFile := seedFile(ownFolder, false)
	trashedFile := seedFile(ownFolder, true)

	srv := NewDocumentGRPCServer(
		folder.NewService(folder.NewPostgresRepository(pool)),
		file.NewService(file.NewPostgresRepository(pool), newMemFileStore(), 1<<20),
		nil, nil, nil, nil, nil, nil)
	ctx := ctxWithTenantAndUser(tenantID, user)

	countComments := func(fileID uuid.UUID) int {
		var n int
		require.NoError(t, pool.QueryRow(testutil.WithSystemCtx(context.Background()),
			`SELECT count(*) FROM document_file_comments WHERE file_id = $1`, fileID).Scan(&n))
		return n
	}
	comment := func(fileID uuid.UUID) error {
		_, err := srv.CreateFileComment(ctx, &documentv1.CreateFileCommentRequest{FileId: fileID.String(), Content: "Hinweis"})
		return err
	}
	register := func(folderID uuid.UUID) error {
		_, err := srv.RegisterUploadedFile(ctx, &documentv1.RegisterUploadedFileRequest{
			FolderId: folderID.String(), Filename: "scan.pdf", MimeType: "application/pdf",
			FileSize: 10, StorageKey: "documents/fp/" + uuid.NewString(), OwnerId: user.String(),
		})
		return err
	}

	t.Run("a comment on a live file is stored", func(t *testing.T) {
		require.NoError(t, comment(liveFile))
		require.Equal(t, 1, countComments(liveFile))
	})
	t.Run("a comment on a file in the trash is refused and nothing is stored", func(t *testing.T) {
		err := comment(trashedFile)
		require.Equal(t, codes.FailedPrecondition, status.Code(err), "got %v", err)
		require.Equal(t, 0, countComments(trashedFile))
	})
	t.Run("register into an unknown folder is NotFound", func(t *testing.T) {
		err := register(uuid.New())
		require.Equal(t, codes.NotFound, status.Code(err), "got %v", err)
	})
	t.Run("register into a folder of another tenant is NotFound", func(t *testing.T) {
		err := register(foreignFolder)
		require.Equal(t, codes.NotFound, status.Code(err), "got %v", err)
	})
	t.Run("register into an own folder succeeds", func(t *testing.T) {
		require.NoError(t, register(ownFolder))
		t.Cleanup(func() {
			_, _ = pool.Exec(testutil.WithSystemCtx(ctxWithTenant(tenantID)), `DELETE FROM document_file_versions WHERE tenant_id = $1`, tenantID)
			_, _ = pool.Exec(testutil.WithSystemCtx(ctxWithTenant(tenantID)), `DELETE FROM document_file_activity WHERE tenant_id = $1`, tenantID)
			_, _ = pool.Exec(testutil.WithSystemCtx(ctxWithTenant(tenantID)), `DELETE FROM document_files WHERE tenant_id = $1 AND filename = 'scan.pdf'`, tenantID)
		})
	})
}
