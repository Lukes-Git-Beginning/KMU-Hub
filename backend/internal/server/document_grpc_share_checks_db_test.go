package server

// DB-backed proof for the entity checks on ShareEntity / UnshareEntity: the shared
// entity must exist in the caller's tenant and be owned by the caller, and only the
// sharer or the owner may withdraw a share (fix-document-share-entity-checks).
// Runs under the kmuhub_app role, so RLS is active.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
	documentv1 "github.com/kmuhub/kmuhub/proto/document/v1"
)

func TestDocumentGRPCServer_ShareEntity_ChecksEntityAndOwner(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Share Checks Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })
	otherTenant := uuid.New()
	testutil.EnsureTenant(t, pool, otherTenant, "Share Checks Foreign Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", otherTenant) })

	owner := seedSharedWithMeUser(t, pool, tenantID, "owner")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", owner) })
	recipient := seedSharedWithMeUser(t, pool, tenantID, "recipient")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", recipient) })
	third := seedSharedWithMeUser(t, pool, tenantID, "third")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", third) })
	foreignUser := seedSharedWithMeUser(t, pool, otherTenant, "foreign")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", foreignUser) })

	seedFolder := func(tenant, creator uuid.UUID) uuid.UUID {
		id := testutil.SeedRow(t, pool, "document_folders", map[string]any{
			"id": uuid.New(), "tenant_id": tenant, "name": "Ordner",
			"space_type": "personal", "space_id": creator, "created_by": creator,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "document_folders", id) })
		return id
	}
	seedFile := func(tenant, folderID, ownerID uuid.UUID, deleted bool) uuid.UUID {
		id := testutil.SeedRow(t, pool, "document_files", map[string]any{
			"id": uuid.New(), "tenant_id": tenant, "folder_id": folderID,
			"filename": "vertrag.pdf", "mime_type": "application/pdf", "file_size": 1024,
			"storage_key": "documents/swm/" + uuid.NewString(), "owner_id": ownerID,
			"is_deleted": deleted,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "document_files", id) })
		return id
	}
	folderID := seedFolder(tenantID, owner)
	fileID := seedFile(tenantID, folderID, owner, false)
	trashedID := seedFile(tenantID, folderID, owner, true)
	foreignFolder := seedFolder(otherTenant, foreignUser)
	foreignFile := seedFile(otherTenant, foreignFolder, foreignUser, false)
	t.Cleanup(func() {
		_, _ = pool.Exec(testutil.WithSystemCtx(ctxWithTenant(tenantID)), `DELETE FROM document_shares WHERE tenant_id = $1`, tenantID)
	})

	srv := sharedWithMeServerForTest(pool)
	ownerCtx := ctxWithTenantAndUser(tenantID, owner)
	shareReq := func(entityType string, entityID, with uuid.UUID) *documentv1.ShareEntityRequest {
		return &documentv1.ShareEntityRequest{
			EntityType: entityType, EntityId: entityID.String(), SharedWithUserId: with.String(),
			Permission: documentv1.SharePermission_SHARE_PERMISSION_READ,
			// A forged shared_by must not matter.
			SharedBy: third.String(),
		}
	}

	t.Run("unknown, trashed and foreign-tenant entities are NotFound", func(t *testing.T) {
		for name, req := range map[string]*documentv1.ShareEntityRequest{
			"unknown file":   shareReq(models.ShareEntityFile, uuid.New(), recipient),
			"unknown folder": shareReq(models.ShareEntityFolder, uuid.New(), recipient),
			"trashed file":   shareReq(models.ShareEntityFile, trashedID, recipient),
			"foreign file":   shareReq(models.ShareEntityFile, foreignFile, recipient),
			"foreign folder": shareReq(models.ShareEntityFolder, foreignFolder, recipient),
		} {
			_, err := srv.ShareEntity(ownerCtx, req)
			require.Error(t, err, name)
			requireGRPCCode(t, err, codes.NotFound)
		}
	})

	t.Run("unknown entity type is InvalidArgument", func(t *testing.T) {
		_, err := srv.ShareEntity(ownerCtx, shareReq("tag", fileID, recipient))
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("a non-owner cannot share, not even a recipient passing it on", func(t *testing.T) {
		_, err := srv.ShareEntity(ctxWithTenantAndUser(tenantID, third), shareReq(models.ShareEntityFile, fileID, recipient))
		requireGRPCCode(t, err, codes.PermissionDenied)

		resp, err := srv.ShareEntity(ownerCtx, shareReq(models.ShareEntityFile, fileID, recipient))
		require.NoError(t, err)
		// shared_by is the caller, not the forged request field.
		require.Equal(t, owner.String(), resp.Share.SharedBy)

		_, err = srv.ShareEntity(ctxWithTenantAndUser(tenantID, recipient), shareReq(models.ShareEntityFile, fileID, third))
		requireGRPCCode(t, err, codes.PermissionDenied)
	})

	t.Run("the folder owner shares a folder, a non-owner does not", func(t *testing.T) {
		_, err := srv.ShareEntity(ctxWithTenantAndUser(tenantID, third), shareReq(models.ShareEntityFolder, folderID, recipient))
		requireGRPCCode(t, err, codes.PermissionDenied)
		_, err = srv.ShareEntity(ownerCtx, shareReq(models.ShareEntityFolder, folderID, recipient))
		require.NoError(t, err)
	})

	t.Run("the foreign tenant cannot reach the entity either", func(t *testing.T) {
		_, err := srv.ShareEntity(ctxWithTenantAndUser(otherTenant, foreignUser), shareReq(models.ShareEntityFile, fileID, foreignUser))
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("unshare: a third party is refused, the sharer may withdraw", func(t *testing.T) {
		unshare := &documentv1.UnshareEntityRequest{
			EntityType: models.ShareEntityFile, EntityId: fileID.String(), SharedWithUserId: recipient.String(),
		}
		_, err := srv.UnshareEntity(ctxWithTenantAndUser(tenantID, third), unshare)
		requireGRPCCode(t, err, codes.PermissionDenied)
		// the recipient cannot drop it either
		_, err = srv.UnshareEntity(ctxWithTenantAndUser(tenantID, recipient), unshare)
		requireGRPCCode(t, err, codes.PermissionDenied)

		list, err := srv.ListShares(ownerCtx, &documentv1.ListSharesRequest{EntityType: models.ShareEntityFile, EntityId: fileID.String()})
		require.NoError(t, err)
		require.Len(t, list.Shares, 1, "the refused attempts must not delete the share")

		// the foreign tenant sees no share at all
		_, err = srv.UnshareEntity(ctxWithTenantAndUser(otherTenant, foreignUser), unshare)
		requireGRPCCode(t, err, codes.NotFound)

		_, err = srv.UnshareEntity(ownerCtx, unshare)
		require.NoError(t, err)
		_, err = srv.UnshareEntity(ownerCtx, unshare)
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("unshare: the owner may withdraw a share somebody else created", func(t *testing.T) {
		// A share created by someone else (shared_by differs from the owner).
		shareID := testutil.SeedRow(t, pool, "document_shares", map[string]any{
			"id": uuid.New(), "tenant_id": tenantID, "entity_type": models.ShareEntityFile,
			"entity_id": fileID, "shared_with_user_id": third, "permission": models.SharePermissionRead,
			"shared_by": recipient,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "document_shares", shareID) })

		unshare := &documentv1.UnshareEntityRequest{
			EntityType: models.ShareEntityFile, EntityId: fileID.String(), SharedWithUserId: third.String(),
		}
		_, err := srv.UnshareEntity(ctxWithTenantAndUser(tenantID, third), unshare)
		requireGRPCCode(t, err, codes.PermissionDenied)
		_, err = srv.UnshareEntity(ownerCtx, unshare) // owner, not the sharer
		require.NoError(t, err)
	})

	t.Run("unshare by share_id: same rules, tenant-scoped lookup", func(t *testing.T) {
		seedShare := func(with, by uuid.UUID) uuid.UUID {
			id := testutil.SeedRow(t, pool, "document_shares", map[string]any{
				"id": uuid.New(), "tenant_id": tenantID, "entity_type": models.ShareEntityFile,
				"entity_id": fileID, "shared_with_user_id": with, "permission": models.SharePermissionRead,
				"shared_by": by,
			})
			t.Cleanup(func() { testutil.CleanupRow(t, pool, "document_shares", id) })
			return id
		}
		byID := func(id uuid.UUID) *documentv1.UnshareEntityRequest {
			return &documentv1.UnshareEntityRequest{ShareId: id.String()}
		}
		shareCount := func() int {
			list, err := srv.ListShares(ownerCtx, &documentv1.ListSharesRequest{EntityType: models.ShareEntityFile, EntityId: fileID.String()})
			require.NoError(t, err)
			return len(list.Shares)
		}

		// created by the recipient (not the owner), shared with the third user
		sharerShare := seedShare(third, recipient)
		_, err := srv.UnshareEntity(ctxWithTenantAndUser(tenantID, third), byID(sharerShare))
		requireGRPCCode(t, err, codes.PermissionDenied)
		_, err = srv.UnshareEntity(ctxWithTenantAndUser(otherTenant, foreignUser), byID(sharerShare))
		requireGRPCCode(t, err, codes.NotFound)
		require.Equal(t, 1, shareCount(), "refused attempts must not delete the share")
		_, err = srv.UnshareEntity(ctxWithTenantAndUser(tenantID, recipient), byID(sharerShare)) // the sharer
		require.NoError(t, err)
		require.Equal(t, 0, shareCount())

		ownerShare := seedShare(third, recipient)
		_, err = srv.UnshareEntity(ownerCtx, byID(ownerShare)) // the owner, not the sharer
		require.NoError(t, err)
		_, err = srv.UnshareEntity(ownerCtx, byID(ownerShare))
		requireGRPCCode(t, err, codes.NotFound)

		_, err = srv.UnshareEntity(ownerCtx, byID(uuid.New()))
		requireGRPCCode(t, err, codes.NotFound)
		_, err = srv.UnshareEntity(ownerCtx, &documentv1.UnshareEntityRequest{ShareId: "nope"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}
