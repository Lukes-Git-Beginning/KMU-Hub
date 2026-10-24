package server

// DB-backed proof that ListSharedWithMe returns the shared files and folders.
// The handler used to answer with empty lists and only a count, so "Mit mir
// geteilt" was empty for every user (fix-document-shared-with-me-and-share-effect).
//
// The tests go through DocumentGRPCServer with real postgres repositories under
// the kmuhub_app role, so RLS is active and the tenant scoping is exercised.

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/document/file"
	"github.com/kmuhub/kmuhub/internal/document/folder"
	"github.com/kmuhub/kmuhub/internal/document/share"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
	documentv1 "github.com/kmuhub/kmuhub/proto/document/v1"
)

func sharedWithMeServerForTest(pool *pgxpool.Pool) *DocumentGRPCServer {
	return NewDocumentGRPCServer(
		folder.NewService(folder.NewPostgresRepository(pool)),
		file.NewService(file.NewPostgresRepository(pool), nil, 0),
		share.NewService(share.NewPostgresRepository(pool)),
		nil, nil, nil, nil, nil,
	)
}

func seedSharedWithMeUser(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, label string) uuid.UUID {
	t.Helper()
	return testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("swm-%s-%s@test.invalid", label, uuid.New().String()[:8]),
		"password_hash": "x",
		"first_name":    "Shared",
		"last_name":     label,
	})
}

func TestDocumentGRPCServer_ListSharedWithMe_ReturnsSharedFilesAndFolders(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Shared With Me Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })

	owner := seedSharedWithMeUser(t, pool, tenantID, "owner")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", owner) })
	recipient := seedSharedWithMeUser(t, pool, tenantID, "recipient")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", recipient) })

	folderID := testutil.SeedRow(t, pool, "document_folders", map[string]any{
		"id": uuid.New(), "tenant_id": tenantID, "name": "Angebote",
		"space_type": "personal", "space_id": owner, "created_by": owner,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "document_folders", folderID) })

	seedFile := func(name string, deleted bool) uuid.UUID {
		id := testutil.SeedRow(t, pool, "document_files", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "folder_id": folderID,
			"filename": name, "mime_type": "application/pdf", "file_size": 1024,
			"storage_key": "documents/swm/" + uuid.NewString(), "owner_id": owner,
			"is_deleted": deleted,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "document_files", id) })
		return id
	}
	sharedFile := seedFile("vertrag.pdf", false)
	unsharedFile := seedFile("privat.pdf", false)
	trashedFile := seedFile("alt.pdf", true)

	srv := sharedWithMeServerForTest(pool)
	ownerCtx := ctxWithTenantAndUser(tenantID, owner)
	for _, in := range []share.ShareInput{
		{EntityType: models.ShareEntityFile, EntityID: sharedFile, Permission: models.SharePermissionWrite},
		{EntityType: models.ShareEntityFile, EntityID: trashedFile, Permission: models.SharePermissionRead},
		{EntityType: models.ShareEntityFolder, EntityID: folderID, Permission: models.SharePermissionRead},
	} {
		in.TenantID, in.SharedBy, in.SharedWithUserID = tenantID, owner, recipient
		_, err := srv.shareService.ShareEntity(ownerCtx, in)
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(testutil.WithSystemCtx(ownerCtx), `DELETE FROM document_shares WHERE tenant_id = $1`, tenantID)
	})

	req := &documentv1.ListSharedWithMeRequest{UserId: recipient.String()}

	t.Run("recipient gets the shared file and folder, not the trashed or unshared file", func(t *testing.T) {
		resp, err := srv.ListSharedWithMe(ctxWithTenantAndUser(tenantID, recipient), req)
		require.NoError(t, err)

		require.Len(t, resp.Files, 1)
		require.Equal(t, sharedFile.String(), resp.Files[0].Id)
		require.Equal(t, "vertrag.pdf", resp.Files[0].Filename)
		require.NotEqual(t, unsharedFile.String(), resp.Files[0].Id)
		require.Len(t, resp.Folders, 1)
		require.Equal(t, folderID.String(), resp.Folders[0].Id)
		require.Equal(t, "Angebote", resp.Folders[0].Name)

		// The share rows carry the permission the desktop reader needs.
		require.Len(t, resp.Shares, 2)
		perms := map[string]documentv1.SharePermission{}
		for _, sh := range resp.Shares {
			perms[sh.EntityId] = sh.Permission
		}
		require.Equal(t, documentv1.SharePermission_SHARE_PERMISSION_WRITE, perms[sharedFile.String()])
		require.Equal(t, documentv1.SharePermission_SHARE_PERMISSION_READ, perms[folderID.String()])

		// total counts what is returned, not the dangling share on the trashed file.
		require.Equal(t, int32(2), resp.Total)
	})

	t.Run("the sharer sees nothing shared with them", func(t *testing.T) {
		resp, err := srv.ListSharedWithMe(ctxWithTenantAndUser(tenantID, owner),
			&documentv1.ListSharedWithMeRequest{UserId: owner.String()})
		require.NoError(t, err)
		require.Empty(t, resp.Files)
		require.Empty(t, resp.Folders)
		require.Empty(t, resp.Shares)
		require.Zero(t, resp.Total)
	})

	t.Run("a foreign tenant sees nothing even for the recipient id", func(t *testing.T) {
		otherTenant := uuid.New()
		testutil.EnsureTenant(t, pool, otherTenant, "Shared With Me Other Tenant")
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", otherTenant) })

		resp, err := srv.ListSharedWithMe(ctxWithTenantAndUser(otherTenant, uuid.New()), req)
		require.NoError(t, err)
		require.Empty(t, resp.Files)
		require.Empty(t, resp.Folders)
		require.Empty(t, resp.Shares)
		require.Zero(t, resp.Total)
	})

	t.Run("invalid user id is rejected", func(t *testing.T) {
		_, err := srv.ListSharedWithMe(ctxWithTenantAndUser(tenantID, recipient),
			&documentv1.ListSharedWithMeRequest{UserId: "nope"})
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})
}
