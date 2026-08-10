package server

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/document/folder"
	"github.com/kmuhub/kmuhub/internal/models"
	documentv1 "github.com/kmuhub/kmuhub/proto/document/v1"
)

// ============================================================================
// stubDocumentFolderRepo - implements folder.Repository (server-package copy, the
// real one lives in folder/service_test.go which is package-internal there)
// ============================================================================

type stubDocumentFolderRepo struct {
	mu           sync.Mutex
	folders      map[uuid.UUID]*models.DocumentFolder
	pathSegments []models.FolderPathSegment

	createErr  error
	listErr    error
	getPathErr error
}

func newStubDocumentFolderRepo() *stubDocumentFolderRepo {
	return &stubDocumentFolderRepo{folders: make(map[uuid.UUID]*models.DocumentFolder)}
}

func (r *stubDocumentFolderRepo) Create(_ context.Context, f *models.DocumentFolder) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.createErr != nil {
		return r.createErr
	}
	cp := *f
	r.folders[f.ID] = &cp
	return nil
}

func (r *stubDocumentFolderRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*models.DocumentFolder, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	f, ok := r.folders[id]
	if !ok || f.TenantID != tenantID {
		return nil, folder.ErrFolderNotFound
	}
	cp := *f
	return &cp, nil
}

// List mirrors the Postgres repository's filter semantics (postgres_repository.go):
// an explicit ParentID filters on it; with no ParentID, root folders (ParentID nil)
// are matched only when a SpaceType filter is present, otherwise every folder for
// the tenant is a candidate.
func (r *stubDocumentFolderRepo) List(_ context.Context, filter folder.ListFilter) ([]*models.DocumentFolder, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.listErr != nil {
		return nil, 0, r.listErr
	}
	var result []*models.DocumentFolder
	for _, f := range r.folders {
		if f.TenantID != filter.TenantID {
			continue
		}
		if filter.ParentID != nil {
			if f.ParentID == nil || *f.ParentID != *filter.ParentID {
				continue
			}
		} else if filter.SpaceType != nil && f.ParentID != nil {
			continue
		}
		if filter.SpaceType != nil && f.SpaceType != *filter.SpaceType {
			continue
		}
		if filter.SpaceID != nil && f.SpaceID != *filter.SpaceID {
			continue
		}
		cp := *f
		result = append(result, &cp)
	}
	return result, len(result), nil
}

func (r *stubDocumentFolderRepo) Update(_ context.Context, tenantID, id uuid.UUID, input folder.UpdateInput) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	f, ok := r.folders[id]
	if !ok || f.TenantID != tenantID {
		return folder.ErrFolderNotFound
	}
	if input.Name != nil {
		f.Name = *input.Name
	}
	if input.ParentID != nil {
		f.ParentID = input.ParentID
	}
	if input.Icon != nil {
		f.Icon = *input.Icon
	}
	f.UpdatedAt = time.Now()
	return nil
}

func (r *stubDocumentFolderRepo) Delete(_ context.Context, tenantID, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	f, ok := r.folders[id]
	if !ok || f.TenantID != tenantID {
		return folder.ErrFolderNotFound
	}
	delete(r.folders, id)
	return nil
}

func (r *stubDocumentFolderRepo) GetPath(_ context.Context, _ uuid.UUID) ([]models.FolderPathSegment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.getPathErr != nil {
		return nil, r.getPathErr
	}
	return r.pathSegments, nil
}

// GetChildren, CountFiles and IsDescendant are part of folder.Repository but
// unused by folder.Service today (verified: no call site in service.go) -
// trivial implementations are enough to satisfy the interface.
func (r *stubDocumentFolderRepo) GetChildren(_ context.Context, _, _ uuid.UUID) ([]*models.DocumentFolder, error) {
	return nil, nil
}

func (r *stubDocumentFolderRepo) CountFiles(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (r *stubDocumentFolderRepo) IsDescendant(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return false, nil
}

// newFolderTestServer wires a real folder.Service (backed by the stub repo)
// into a DocumentGRPCServer, the other five services stay nil - this unit
// only exercises the folder cluster.
func newFolderTestServer(repo *stubDocumentFolderRepo) *DocumentGRPCServer {
	return NewDocumentGRPCServer(folder.NewService(repo), nil, nil, nil, nil, nil, nil, nil)
}

func seedFolder(t *testing.T, repo *stubDocumentFolderRepo, tenantID uuid.UUID, mutate func(*models.DocumentFolder)) *models.DocumentFolder {
	t.Helper()
	f := &models.DocumentFolder{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      "Seed",
		SpaceType: models.FolderSpacePersonal,
		SpaceID:   uuid.New(),
		Icon:      "folder",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if mutate != nil {
		mutate(f)
	}
	require.NoError(t, repo.Create(context.Background(), f))
	return f
}

// ============================================================================
// CreateFolder
// ============================================================================

func TestCreateFolder_HappyPath(t *testing.T) {
	repo := newStubDocumentFolderRepo()
	srv := newFolderTestServer(repo)
	tenant := uuid.New()
	spaceID := uuid.New()
	createdBy := uuid.New()

	resp, err := srv.CreateFolder(ctxWithTenant(tenant), &documentv1.CreateFolderRequest{
		Name:      "Projekte",
		SpaceType: documentv1.FolderSpaceType_FOLDER_SPACE_TYPE_TEAM,
		SpaceId:   spaceID.String(),
		Icon:      "briefcase",
		CreatedBy: createdBy.String(),
	})

	requireGRPCOK(t, err)
	require.Equal(t, "Projekte", resp.Folder.Name)
	require.Equal(t, documentv1.FolderSpaceType_FOLDER_SPACE_TYPE_TEAM, resp.Folder.SpaceType)
	require.Equal(t, spaceID.String(), resp.Folder.SpaceId)
	require.Empty(t, resp.Folder.ParentId)
	require.False(t, resp.Folder.IsSystem)
}

func TestCreateFolder_DuplicateNameInSameParent(t *testing.T) {
	repo := newStubDocumentFolderRepo()
	srv := newFolderTestServer(repo)
	tenant := uuid.New()
	spaceID := uuid.New()
	createdBy := uuid.New()

	req := &documentv1.CreateFolderRequest{
		Name:      "Vertraege",
		SpaceType: documentv1.FolderSpaceType_FOLDER_SPACE_TYPE_TEAM,
		SpaceId:   spaceID.String(),
		CreatedBy: createdBy.String(),
	}
	_, err := srv.CreateFolder(ctxWithTenant(tenant), req)
	requireGRPCOK(t, err)

	_, err = srv.CreateFolder(ctxWithTenant(tenant), req)
	requireGRPCCode(t, err, codes.AlreadyExists)
}

// ============================================================================
// GetFolder
// ============================================================================

func TestGetFolder_HappyPath(t *testing.T) {
	repo := newStubDocumentFolderRepo()
	srv := newFolderTestServer(repo)
	tenant := uuid.New()
	f := seedFolder(t, repo, tenant, func(f *models.DocumentFolder) { f.Name = "Steuern" })

	resp, err := srv.GetFolder(ctxWithTenant(tenant), &documentv1.GetFolderRequest{Id: f.ID.String()})

	requireGRPCOK(t, err)
	require.Equal(t, "Steuern", resp.Folder.Name)
	require.Equal(t, f.ID.String(), resp.Folder.Id)
}

func TestGetFolder_WrongTenantNotFound(t *testing.T) {
	repo := newStubDocumentFolderRepo()
	srv := newFolderTestServer(repo)
	f := seedFolder(t, repo, uuid.New(), nil)

	_, err := srv.GetFolder(ctxWithTenant(uuid.New()), &documentv1.GetFolderRequest{Id: f.ID.String()})

	requireGRPCCode(t, err, codes.NotFound)
}

// ============================================================================
// ListFolders
// ============================================================================

func TestListFolders_HappyPath(t *testing.T) {
	repo := newStubDocumentFolderRepo()
	srv := newFolderTestServer(repo)
	tenant := uuid.New()
	spaceID := uuid.New()
	seedFolder(t, repo, tenant, func(f *models.DocumentFolder) {
		f.Name = "Root A"
		f.SpaceType = models.FolderSpaceTeam
		f.SpaceID = spaceID
	})
	seedFolder(t, repo, tenant, func(f *models.DocumentFolder) {
		f.Name = "Root B"
		f.SpaceType = models.FolderSpaceTeam
		f.SpaceID = spaceID
	})
	// Different tenant, must not leak into the result.
	seedFolder(t, repo, uuid.New(), func(f *models.DocumentFolder) {
		f.SpaceType = models.FolderSpaceTeam
		f.SpaceID = spaceID
	})

	resp, err := srv.ListFolders(ctxWithTenant(tenant), &documentv1.ListFoldersRequest{
		SpaceType: documentv1.FolderSpaceType_FOLDER_SPACE_TYPE_TEAM,
		SpaceId:   spaceID.String(),
	})

	requireGRPCOK(t, err)
	require.Len(t, resp.Folders, 2)
}

func TestListFolders_NoMatchesReturnsEmptyNotNilSlice(t *testing.T) {
	repo := newStubDocumentFolderRepo()
	srv := newFolderTestServer(repo)
	tenant := uuid.New()

	resp, err := srv.ListFolders(ctxWithTenant(tenant), &documentv1.ListFoldersRequest{})

	requireGRPCOK(t, err)
	require.NotNil(t, resp.Folders)
	require.Empty(t, resp.Folders)
}

func TestListFolders_RepositoryErrorMapsToInternal(t *testing.T) {
	repo := newStubDocumentFolderRepo()
	repo.listErr = errors.New("db unavailable")
	srv := newFolderTestServer(repo)

	_, err := srv.ListFolders(ctxWithTenant(uuid.New()), &documentv1.ListFoldersRequest{})

	requireGRPCCode(t, err, codes.Internal)
}

// ============================================================================
// UpdateFolder
// ============================================================================

func TestUpdateFolder_HappyPath(t *testing.T) {
	repo := newStubDocumentFolderRepo()
	srv := newFolderTestServer(repo)
	tenant := uuid.New()
	f := seedFolder(t, repo, tenant, func(f *models.DocumentFolder) { f.Name = "Alt" })

	resp, err := srv.UpdateFolder(ctxWithTenant(tenant), &documentv1.UpdateFolderRequest{
		Id:   f.ID.String(),
		Name: strPtr("Neu"),
	})

	requireGRPCOK(t, err)
	require.Equal(t, "Neu", resp.Folder.Name)
}

func TestUpdateFolder_RenameSystemFolderFails(t *testing.T) {
	repo := newStubDocumentFolderRepo()
	srv := newFolderTestServer(repo)
	tenant := uuid.New()
	f := seedFolder(t, repo, tenant, func(f *models.DocumentFolder) { f.IsSystem = true })

	_, err := srv.UpdateFolder(ctxWithTenant(tenant), &documentv1.UpdateFolderRequest{
		Id:   f.ID.String(),
		Name: strPtr("Umbenannt"),
	})

	requireGRPCCode(t, err, codes.FailedPrecondition)
}

// ============================================================================
// DeleteFolder
//
// folder.Service.Delete (service.go) only guards against deleting a system
// folder - it does NOT reject a folder that still has files or subfolders
// (that case cascades: PostgresRepository.Delete soft-deletes contained
// files and relies on an FK ON DELETE CASCADE for subfolders, see
// postgres_repository.go:154-172). The backlog's done_when assumed a
// contains-children error path exists; it does not, so the error-path test
// below exercises the guard that is actually implemented (system folder).
// ============================================================================

func TestDeleteFolder_HappyPath(t *testing.T) {
	repo := newStubDocumentFolderRepo()
	srv := newFolderTestServer(repo)
	tenant := uuid.New()
	f := seedFolder(t, repo, tenant, nil)

	_, err := srv.DeleteFolder(ctxWithTenant(tenant), &documentv1.DeleteFolderRequest{Id: f.ID.String()})

	requireGRPCOK(t, err)
	_, getErr := repo.GetByID(context.Background(), tenant, f.ID)
	require.ErrorIs(t, getErr, folder.ErrFolderNotFound)
}

func TestDeleteFolder_SystemFolderFails(t *testing.T) {
	repo := newStubDocumentFolderRepo()
	srv := newFolderTestServer(repo)
	tenant := uuid.New()
	f := seedFolder(t, repo, tenant, func(f *models.DocumentFolder) { f.IsSystem = true })

	_, err := srv.DeleteFolder(ctxWithTenant(tenant), &documentv1.DeleteFolderRequest{Id: f.ID.String()})

	requireGRPCCode(t, err, codes.FailedPrecondition)
}

// ============================================================================
// GetFolderPath
// ============================================================================

func TestGetFolderPath_HappyPath(t *testing.T) {
	repo := newStubDocumentFolderRepo()
	repo.pathSegments = []models.FolderPathSegment{
		{ID: uuid.New(), Name: "Root"},
		{ID: uuid.New(), Name: "Unterordner"},
	}
	srv := newFolderTestServer(repo)

	resp, err := srv.GetFolderPath(context.Background(), &documentv1.GetFolderPathRequest{Id: uuid.New().String()})

	requireGRPCOK(t, err)
	require.Len(t, resp.Segments, 2)
	require.Equal(t, "Root", resp.Segments[0].Name)
	require.Equal(t, "Unterordner", resp.Segments[1].Name)
}

func TestGetFolderPath_RepositoryErrorMapsToInternal(t *testing.T) {
	repo := newStubDocumentFolderRepo()
	repo.getPathErr = errors.New("recursive query failed")
	srv := newFolderTestServer(repo)

	_, err := srv.GetFolderPath(context.Background(), &documentv1.GetFolderPathRequest{Id: uuid.New().String()})

	requireGRPCCode(t, err, codes.Internal)
}

// ============================================================================
// InitializeUserSpace
// ============================================================================

func TestInitializeUserSpace_HappyPath(t *testing.T) {
	repo := newStubDocumentFolderRepo()
	srv := newFolderTestServer(repo)
	tenant := uuid.New()
	userID := uuid.New()

	_, err := srv.InitializeUserSpace(ctxWithTenant(tenant), &documentv1.InitializeUserSpaceRequest{UserId: userID.String()})

	requireGRPCOK(t, err)
	folders, total, listErr := repo.List(context.Background(), folder.ListFilter{TenantID: tenant})
	require.NoError(t, listErr)
	require.Equal(t, 4, total) // root "Meine Dateien" + 3 default subfolders
	var root *models.DocumentFolder
	for _, f := range folders {
		if f.ParentID == nil {
			root = f
		}
	}
	require.NotNil(t, root)
	require.Equal(t, "Meine Dateien", root.Name)
	require.True(t, root.IsSystem)
}

func TestInitializeUserSpace_RepositoryErrorPropagates(t *testing.T) {
	repo := newStubDocumentFolderRepo()
	repo.createErr = errors.New("insert failed")
	srv := newFolderTestServer(repo)

	_, err := srv.InitializeUserSpace(ctxWithTenant(uuid.New()), &documentv1.InitializeUserSpaceRequest{UserId: uuid.New().String()})

	requireGRPCCode(t, err, codes.Internal)
}

// ============================================================================
// InitializeTeamSpace
// ============================================================================

func TestInitializeTeamSpace_HappyPath(t *testing.T) {
	repo := newStubDocumentFolderRepo()
	srv := newFolderTestServer(repo)
	tenant := uuid.New()
	teamID := uuid.New()

	_, err := srv.InitializeTeamSpace(ctxWithTenant(tenant), &documentv1.InitializeTeamSpaceRequest{TeamId: teamID.String()})

	requireGRPCOK(t, err)
	folders, total, listErr := repo.List(context.Background(), folder.ListFilter{TenantID: tenant})
	require.NoError(t, listErr)
	require.Equal(t, 4, total) // root "Team" + 3 default subfolders
	var root *models.DocumentFolder
	for _, f := range folders {
		if f.ParentID == nil {
			root = f
		}
	}
	require.NotNil(t, root)
	require.Equal(t, "Team", root.Name)
	require.Equal(t, models.FolderSpaceTeam, root.SpaceType)
}

func TestInitializeTeamSpace_RepositoryErrorPropagates(t *testing.T) {
	repo := newStubDocumentFolderRepo()
	repo.createErr = errors.New("insert failed")
	srv := newFolderTestServer(repo)

	_, err := srv.InitializeTeamSpace(ctxWithTenant(uuid.New()), &documentv1.InitializeTeamSpaceRequest{TeamId: uuid.New().String()})

	requireGRPCCode(t, err, codes.Internal)
}
