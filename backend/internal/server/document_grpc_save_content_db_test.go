package server

// DB-backed proof for an editor's save through CreateFileVersion with content
// (fix-wopi-putfile-discards-the-saved-body). The saved bytes must land in the new version AND
// become the file's current content (storage_key/file_size follow the version, otherwise the
// next open serves the old document again); an unchanged save adds no version; a foreign
// tenant's file is NotFound with nothing stored. Runs under kmuhub_app (RLS active).

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/document/file"
	"github.com/kmuhub/kmuhub/internal/testutil"
	documentv1 "github.com/kmuhub/kmuhub/proto/document/v1"
)

func TestDocumentGRPCServer_CreateFileVersion_SavesTheEditorsContent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Save Content Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })
	otherTenant := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenant, "Save Content Foreign Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", otherTenant) })

	owner := seedSharedWithMeUser(t, pool, tenantID, "scowner")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", owner) })
	editor := seedSharedWithMeUser(t, pool, tenantID, "sceditor")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", editor) })
	foreignUser := seedSharedWithMeUser(t, pool, otherTenant, "scforeign")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", foreignUser) })

	store := newMemFileStore()
	seedFile := func(tenant, creator uuid.UUID, content string) (uuid.UUID, string) {
		key := "documents/sc/" + uuid.NewString() + "/angebot.docx"
		store.objects[key] = content
		folderID := testutil.SeedRow(t, pool, "document_folders", map[string]any{
			"id": uuid.New(), "tenant_id": tenant, "name": "Ordner",
			"space_type": "personal", "space_id": creator, "created_by": creator,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "document_folders", folderID) })
		id := testutil.SeedRow(t, pool, "document_files", map[string]any{
			"id": uuid.New(), "tenant_id": tenant, "folder_id": folderID,
			"filename": "angebot.docx", "mime_type": "application/octet-stream", "file_size": len(content),
			"storage_key": key, "owner_id": creator,
		})
		// The upload's version 1, as file.Service.Upload records it.
		testutil.SeedRow(t, pool, "document_file_versions", map[string]any{
			"id": uuid.New(), "tenant_id": tenant, "file_id": id, "version_number": 1,
			"storage_key": key, "file_size": len(content), "created_by": creator,
		})
		t.Cleanup(func() {
			_, _ = pool.Exec(testutil.WithSystemCtx(ctxWithTenant(tenant)), `DELETE FROM document_file_versions WHERE file_id = $1`, id)
			_, _ = pool.Exec(testutil.WithSystemCtx(ctxWithTenant(tenant)), `DELETE FROM document_file_activity WHERE file_id = $1`, id)
			testutil.CleanupRow(t, pool, "document_files", id)
		})
		return id, key
	}

	fileID, originalKey := seedFile(tenantID, owner, "Entwurf")
	foreignFile, foreignKey := seedFile(otherTenant, foreignUser, "fremd")

	srv := NewDocumentGRPCServer(nil, file.NewService(file.NewPostgresRepository(pool), store, 0), nil, nil, nil, nil, nil, nil)
	editorCtx := ctxWithTenantAndUser(tenantID, editor)

	t.Run("a foreign tenant's file is NotFound and nothing is stored", func(t *testing.T) {
		before := len(store.objects)
		_, err := srv.CreateFileVersion(editorCtx, &documentv1.CreateFileVersionRequest{
			FileId: foreignFile.String(), Content: []byte("eingeschleust"),
		})
		requireGRPCCode(t, err, codes.NotFound)
		require.Len(t, store.objects, before)
		require.Equal(t, "fremd", store.objects[foreignKey])
	})

	t.Run("the saved content becomes the new version and the file's current content", func(t *testing.T) {
		resp, err := srv.CreateFileVersion(editorCtx, &documentv1.CreateFileVersionRequest{
			FileId: fileID.String(), VersionLabel: "Auto-saved via WOPI", Content: []byte("Endfassung mit Preisen"),
			CreatedBy: foreignUser.String(), // forged, must not matter
		})
		require.NoError(t, err)
		v := resp.Version
		require.EqualValues(t, 2, v.VersionNumber)
		require.EqualValues(t, len("Endfassung mit Preisen"), v.FileSize)
		require.Equal(t, editor.String(), v.CreatedBy)
		require.Equal(t, "Endfassung mit Preisen", store.objects[v.StorageKey], "the version holds the saved bytes")
		require.Equal(t, "Entwurf", store.objects[originalKey], "version 1 keeps the original")

		got, err := srv.fileService.GetByID(editorCtx, fileID, tenantID)
		require.NoError(t, err)
		require.Equal(t, 2, got.CurrentVersion)
		require.Equal(t, v.StorageKey, got.StorageKey, "download and the editor now serve the saved content")
		require.EqualValues(t, len("Endfassung mit Preisen"), got.FileSize)
	})

	t.Run("an unchanged save adds no version", func(t *testing.T) {
		resp, err := srv.CreateFileVersion(editorCtx, &documentv1.CreateFileVersionRequest{
			FileId: fileID.String(), VersionLabel: "Auto-saved via WOPI", Content: []byte("Endfassung mit Preisen"),
		})
		require.NoError(t, err)
		require.EqualValues(t, 2, resp.Version.VersionNumber, "the current version is answered")

		list, err := srv.ListFileVersions(editorCtx, &documentv1.ListFileVersionsRequest{FileId: fileID.String()})
		require.NoError(t, err)
		require.Len(t, list.Versions, 2)
	})

	t.Run("a same-size change is still a new version", func(t *testing.T) {
		resp, err := srv.CreateFileVersion(editorCtx, &documentv1.CreateFileVersionRequest{
			FileId: fileID.String(), Content: []byte("Endfassung ohne Preisen"[:len("Endfassung mit Preisen")]),
		})
		require.NoError(t, err)
		require.EqualValues(t, 3, resp.Version.VersionNumber)
	})

	t.Run("a revert makes the reverted content current again", func(t *testing.T) {
		resp, err := srv.RevertFileVersion(editorCtx, &documentv1.RevertFileVersionRequest{FileId: fileID.String(), VersionNumber: 1})
		require.NoError(t, err)

		got, err := srv.fileService.GetByID(editorCtx, fileID, tenantID)
		require.NoError(t, err)
		require.Equal(t, 4, got.CurrentVersion)
		require.Equal(t, got.StorageKey, resp.File.StorageKey)
		require.NotEqual(t, originalKey, got.StorageKey, "the revert is a fresh copy")
		require.Equal(t, "Entwurf", store.objects[got.StorageKey])
		require.EqualValues(t, len("Entwurf"), got.FileSize)
	})
}
