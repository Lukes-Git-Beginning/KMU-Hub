package server

// DB-backed proof for CreateFileVersion (fix-document-version-create-route). The RPC used
// to answer from the request without persisting anything. It now freezes the file's current
// content as a new version: content and size come from the file, the author from the
// caller's JWT, and a file of another tenant is NotFound. Runs under kmuhub_app (RLS active).

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/document/file"
	"github.com/kmuhub/kmuhub/internal/testutil"
	documentv1 "github.com/kmuhub/kmuhub/proto/document/v1"
)

// memFileStore is an in-memory chat/file.FileStore that records what was copied where.
type memFileStore struct {
	mu        sync.Mutex
	objects   map[string]string
	downloads []string
}

func newMemFileStore() *memFileStore { return &memFileStore{objects: map[string]string{}} }

func (m *memFileStore) Upload(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[key] = string(b)
	return nil
}

func (m *memFileStore) Download(_ context.Context, key string) (io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.downloads = append(m.downloads, key)
	return io.NopCloser(strings.NewReader(m.objects[key])), nil
}

func (m *memFileStore) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.objects, key)
	return nil
}

func (m *memFileStore) GetPresignedURL(context.Context, string, time.Duration) (string, error) {
	return "", nil
}

func (m *memFileStore) GetPresignedUploadURL(context.Context, string, time.Duration) (string, error) {
	return "", nil
}

func TestDocumentGRPCServer_CreateFileVersion_FreezesTheCurrentContent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Create Version Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })
	otherTenant := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenant, "Create Version Foreign Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", otherTenant) })

	owner := seedSharedWithMeUser(t, pool, tenantID, "cvowner")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", owner) })
	editor := seedSharedWithMeUser(t, pool, tenantID, "cveditor")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", editor) })
	foreignUser := seedSharedWithMeUser(t, pool, otherTenant, "cvforeign")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", foreignUser) })

	seedFile := func(tenant, creator uuid.UUID, key string, size int64, deleted bool) uuid.UUID {
		folderID := testutil.SeedRow(t, pool, "document_folders", map[string]any{
			"id": uuid.New(), "tenant_id": tenant, "name": "Ordner",
			"space_type": "personal", "space_id": creator, "created_by": creator,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "document_folders", folderID) })
		id := testutil.SeedRow(t, pool, "document_files", map[string]any{
			"id": uuid.New(), "tenant_id": tenant, "folder_id": folderID,
			"filename": "vertrag.pdf", "mime_type": "application/pdf", "file_size": size,
			"storage_key": key, "owner_id": creator, "is_deleted": deleted,
		})
		t.Cleanup(func() {
			_, _ = pool.Exec(testutil.WithSystemCtx(ctxWithTenant(tenant)), `DELETE FROM document_file_versions WHERE file_id = $1`, id)
			_, _ = pool.Exec(testutil.WithSystemCtx(ctxWithTenant(tenant)), `DELETE FROM document_file_activity WHERE file_id = $1`, id)
			testutil.CleanupRow(t, pool, "document_files", id)
		})
		return id
	}

	store := newMemFileStore()
	liveKey := "documents/cv/" + uuid.NewString() + "/vertrag.pdf"
	store.objects[liveKey] = "aktueller Inhalt"
	fileID := seedFile(tenantID, owner, liveKey, 16, false)
	trashedID := seedFile(tenantID, owner, liveKey, 16, true)
	foreignKey := "documents/cv/" + uuid.NewString() + "/fremd.pdf"
	store.objects[foreignKey] = "fremder Inhalt"
	foreignFile := seedFile(otherTenant, foreignUser, foreignKey, 14, false)

	srv := NewDocumentGRPCServer(nil, file.NewService(file.NewPostgresRepository(pool), store, 0), nil, nil, nil, nil, nil, nil)
	editorCtx := ctxWithTenantAndUser(tenantID, editor)

	t.Run("a foreign file is NotFound and nothing is copied", func(t *testing.T) {
		_, err := srv.CreateFileVersion(editorCtx, &documentv1.CreateFileVersionRequest{FileId: foreignFile.String()})
		requireGRPCCode(t, err, codes.NotFound)
		require.NotContains(t, store.downloads, foreignKey)
	})

	t.Run("a trashed file is refused", func(t *testing.T) {
		_, err := srv.CreateFileVersion(editorCtx, &documentv1.CreateFileVersionRequest{FileId: trashedID.String()})
		require.Error(t, err)
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("an over-long label is InvalidArgument", func(t *testing.T) {
		_, err := srv.CreateFileVersion(editorCtx, &documentv1.CreateFileVersionRequest{
			FileId: fileID.String(), VersionLabel: strings.Repeat("x", maxVersionLabelLen+1),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("content, size and author come from the file and the caller, not the request", func(t *testing.T) {
		resp, err := srv.CreateFileVersion(editorCtx, &documentv1.CreateFileVersionRequest{
			FileId:       fileID.String(),
			VersionLabel: "  Vor Abnahme  ",
			// Forged fields that must not matter.
			StorageKey: foreignKey,
			FileSize:   999999,
			CreatedBy:  foreignUser.String(),
		})
		require.NoError(t, err)
		v := resp.Version
		require.Equal(t, fileID.String(), v.FileId)
		require.EqualValues(t, 2, v.VersionNumber)
		require.Equal(t, "Vor Abnahme", v.VersionLabel)
		require.EqualValues(t, 16, v.FileSize)
		require.Equal(t, editor.String(), v.CreatedBy)
		require.NotEqual(t, foreignKey, v.StorageKey)
		require.Equal(t, "aktueller Inhalt", store.objects[v.StorageKey], "the version holds a copy of the file's own object")
		require.NotContains(t, store.downloads, foreignKey)

		list, err := srv.ListFileVersions(editorCtx, &documentv1.ListFileVersionsRequest{FileId: fileID.String()})
		require.NoError(t, err)
		require.Len(t, list.Versions, 1, "the persisted version shows up in the history")
		require.Equal(t, v.Id, list.Versions[0].Id)

		got, err := srv.fileService.GetByID(editorCtx, fileID, tenantID)
		require.NoError(t, err)
		require.Equal(t, 2, got.CurrentVersion)
	})

	t.Run("without a label the version stays unlabelled", func(t *testing.T) {
		resp, err := srv.CreateFileVersion(editorCtx, &documentv1.CreateFileVersionRequest{FileId: fileID.String()})
		require.NoError(t, err)
		require.EqualValues(t, 3, resp.Version.VersionNumber)
		require.Empty(t, resp.Version.VersionLabel)
	})
}
