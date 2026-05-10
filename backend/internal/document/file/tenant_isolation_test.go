package file

// Cross-tenant isolation tests for document_file_versions (Migration 000114).
//
// document_file_versions.tenant_id is NOT NULL after migration. Version creation
// now stamps the parent file's tenant_id onto every DocumentFileVersion. These
// tests verify that versions are correctly scoped to their owning tenant.

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	docTenantA = uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001")
	docTenantB = uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000002")
)

// TestCreateVersion_TenantID_Stamped verifies that CreateVersion stamps the file's
// tenant ID onto the newly created DocumentFileVersion.
func TestCreateVersion_TenantID_Stamped(t *testing.T) {
	svc, repo, _ := newTestService()

	// Seed a file belonging to tenant A.
	fileA := seedFileForTenant(repo, docTenantA)

	version, err := svc.CreateVersion(context.Background(), fileA.ID, docTenantA, VersionInput{
		Reader:   strings.NewReader("v2 content"),
		FileSize: 10,
		UserID:   uuid.New(),
	})

	require.NoError(t, err)
	require.NotNil(t, version)
	assert.Equal(t, docTenantA, version.TenantID, "version must carry tenant A's ID")
}

// TestCreateVersion_TenantIsolation_CannotAccessAcrossTenants verifies that a
// GetByID call for a file with tenant A's ID returns ErrFileNotFound when queried
// with tenant B's ID.
func TestCreateVersion_TenantIsolation_CannotAccessAcrossTenants(t *testing.T) {
	_, repo, _ := newTestService()

	// Seed files for two different tenants.
	fileA := seedFileForTenant(repo, docTenantA)
	seedFileForTenant(repo, docTenantB)

	ctx := context.Background()

	// Querying fileA with tenant B's context must return not-found.
	got, err := repo.GetByID(ctx, fileA.ID, docTenantB)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, ErrFileNotFound, "file owned by tenant A must not be visible to tenant B")
}

// TestSoftDelete_TenantIsolation verifies that SoftDelete with the wrong tenant ID
// returns not-found, preventing cross-tenant deletion.
func TestSoftDelete_TenantIsolation(t *testing.T) {
	_, repo, _ := newTestService()

	fileA := seedFileForTenant(repo, docTenantA)

	err := repo.SoftDelete(context.Background(), fileA.ID, docTenantB)
	assert.ErrorIs(t, err, ErrFileNotFound, "tenant B must not be able to soft-delete tenant A's file")

	// Verify the file was NOT deleted.
	f, ok := repo.files[fileA.ID]
	require.True(t, ok)
	assert.False(t, f.IsDeleted, "file must remain undeleted after failed cross-tenant delete")
}

// TestUpload_TenantID_Stamped verifies that a new file upload stamps the tenant ID
// onto the file record and its initial version.
func TestUpload_TenantID_Stamped(t *testing.T) {
	svc, repo, _ := newTestService()

	input := UploadInput{
		FolderID:  uuid.New(),
		TenantID:  docTenantA,
		Filename:  "invoice.pdf",
		MimeType:  "application/pdf",
		FileSize:  512,
		Reader:    strings.NewReader("invoice data"),
		SpaceType: "personal",
		SpaceID:   uuid.New(),
		OwnerID:   uuid.New(),
	}

	uploadedFile, err := svc.Upload(context.Background(), input)
	require.NoError(t, err)
	require.NotNil(t, uploadedFile)
	assert.Equal(t, docTenantA, uploadedFile.TenantID, "uploaded file must carry tenant A's ID")

	// Verify the first version was also stamped.
	versions := repo.versions[uploadedFile.ID]
	require.Len(t, versions, 1, "upload must create exactly one version")
	assert.Equal(t, docTenantA, versions[0].TenantID, "initial version must carry tenant A's ID")
}

// TestUpload_SeparateTenants_NoLeak verifies that files uploaded for different
// tenants are independently accessible without cross-contamination.
func TestUpload_SeparateTenants_NoLeak(t *testing.T) {
	svc, repo, _ := newTestService()

	readerA := io.NopCloser(strings.NewReader("tenant a data"))
	readerB := io.NopCloser(strings.NewReader("tenant b data"))

	fileA, err := svc.Upload(context.Background(), UploadInput{
		FolderID:  uuid.New(),
		TenantID:  docTenantA,
		Filename:  "tenantA.txt",
		MimeType:  "text/plain",
		FileSize:  14,
		Reader:    readerA,
		SpaceType: "personal",
		SpaceID:   uuid.New(),
		OwnerID:   uuid.New(),
	})
	require.NoError(t, err)

	fileB, err := svc.Upload(context.Background(), UploadInput{
		FolderID:  uuid.New(),
		TenantID:  docTenantB,
		Filename:  "tenantB.txt",
		MimeType:  "text/plain",
		FileSize:  14,
		Reader:    readerB,
		SpaceType: "personal",
		SpaceID:   uuid.New(),
		OwnerID:   uuid.New(),
	})
	require.NoError(t, err)

	assert.Equal(t, docTenantA, fileA.TenantID)
	assert.Equal(t, docTenantB, fileB.TenantID)
	assert.NotEqual(t, fileA.ID, fileB.ID)

	// Tenant B cannot access tenant A's file.
	_, err = repo.GetByID(context.Background(), fileA.ID, docTenantB)
	assert.ErrorIs(t, err, ErrFileNotFound, "tenant B must not see tenant A's file")

	// Tenant A cannot access tenant B's file.
	_, err = repo.GetByID(context.Background(), fileB.ID, docTenantA)
	assert.ErrorIs(t, err, ErrFileNotFound, "tenant A must not see tenant B's file")
}
