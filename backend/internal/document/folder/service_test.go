package folder

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// --- Mock Repository ---

type MockRepository struct {
	folders     map[uuid.UUID]*models.DocumentFolder
	fileCounts  map[uuid.UUID]int
	descendants map[uuid.UUID]map[uuid.UUID]bool // ancestor -> descendant set

	failCreate       bool
	failGetByID      bool
	failList         bool
	failUpdate       bool
	failDelete       bool
	failGetPath      bool
	failIsDescendant bool
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		folders:     make(map[uuid.UUID]*models.DocumentFolder),
		fileCounts:  make(map[uuid.UUID]int),
		descendants: make(map[uuid.UUID]map[uuid.UUID]bool),
	}
}

func (m *MockRepository) Create(_ context.Context, folder *models.DocumentFolder) error {
	if m.failCreate {
		return errors.New("repo create error")
	}
	m.folders[folder.ID] = folder
	return nil
}

func (m *MockRepository) GetByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*models.DocumentFolder, error) {
	if m.failGetByID {
		return nil, ErrFolderNotFound
	}
	folder, ok := m.folders[id]
	if !ok {
		return nil, ErrFolderNotFound
	}
	return folder, nil
}

func (m *MockRepository) List(_ context.Context, filter ListFilter) ([]*models.DocumentFolder, int, error) {
	if m.failList {
		return nil, 0, errors.New("repo list error")
	}
	var result []*models.DocumentFolder
	for _, f := range m.folders {
		match := true
		if filter.SpaceType != nil && f.SpaceType != *filter.SpaceType {
			match = false
		}
		if filter.SpaceID != nil && f.SpaceID != *filter.SpaceID {
			match = false
		}
		if filter.ParentID != nil {
			if f.ParentID == nil || *f.ParentID != *filter.ParentID {
				match = false
			}
		}
		if match {
			result = append(result, f)
		}
	}
	return result, len(result), nil
}

func (m *MockRepository) Update(_ context.Context, _ uuid.UUID, id uuid.UUID, input UpdateInput) error {
	if m.failUpdate {
		return errors.New("repo update error")
	}
	folder, ok := m.folders[id]
	if !ok {
		return ErrFolderNotFound
	}
	if input.Name != nil {
		folder.Name = *input.Name
	}
	if input.ParentID != nil {
		folder.ParentID = input.ParentID
	}
	if input.Icon != nil {
		folder.Icon = *input.Icon
	}
	return nil
}

func (m *MockRepository) Delete(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	if m.failDelete {
		return errors.New("repo delete error")
	}
	delete(m.folders, id)
	return nil
}

func (m *MockRepository) GetPath(_ context.Context, id uuid.UUID) ([]models.FolderPathSegment, error) {
	if m.failGetPath {
		return nil, errors.New("repo get path error")
	}
	folder, ok := m.folders[id]
	if !ok {
		return nil, ErrFolderNotFound
	}
	return []models.FolderPathSegment{
		{ID: folder.ID, Name: folder.Name},
	}, nil
}

func (m *MockRepository) GetChildren(_ context.Context, _ uuid.UUID, parentID uuid.UUID) ([]*models.DocumentFolder, error) {
	var result []*models.DocumentFolder
	for _, f := range m.folders {
		if f.ParentID != nil && *f.ParentID == parentID {
			result = append(result, f)
		}
	}
	return result, nil
}

func (m *MockRepository) CountFiles(_ context.Context, folderID uuid.UUID) (int, error) {
	return m.fileCounts[folderID], nil
}

func (m *MockRepository) IsDescendant(_ context.Context, folderID, potentialAncestorID uuid.UUID) (bool, error) {
	if m.failIsDescendant {
		return false, errors.New("repo is descendant error")
	}
	if descs, ok := m.descendants[potentialAncestorID]; ok {
		return descs[folderID], nil
	}
	return false, nil
}

// --- Helper ---

func newTestService() (*Service, *MockRepository) {
	repo := NewMockRepository()
	svc := NewService(repo)
	return svc, repo
}

func validCreateInput() CreateInput {
	return CreateInput{
		Name:      "Projects",
		SpaceType: models.FolderSpacePersonal,
		SpaceID:   uuid.New(),
		CreatedBy: uuid.New(),
	}
}

func seedFolder(repo *MockRepository, name string, isSystem bool) *models.DocumentFolder {
	folder := &models.DocumentFolder{
		ID:        uuid.New(),
		Name:      name,
		SpaceType: models.FolderSpacePersonal,
		SpaceID:   uuid.New(),
		IsSystem:  isSystem,
		Icon:      "folder",
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.folders[folder.ID] = folder
	return folder
}

// --- Tests ---

func TestCreate_Success(t *testing.T) {
	svc, repo := newTestService()
	input := validCreateInput()

	folder, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	require.NotNil(t, folder)
	assert.Equal(t, input.Name, folder.Name)
	assert.Equal(t, input.SpaceType, folder.SpaceType)
	assert.Equal(t, input.SpaceID, folder.SpaceID)
	assert.False(t, folder.IsSystem)
	assert.Equal(t, "folder", folder.Icon)

	_, ok := repo.folders[folder.ID]
	assert.True(t, ok)
}

func TestCreate_EmptyName(t *testing.T) {
	svc, _ := newTestService()
	input := validCreateInput()
	input.Name = ""

	folder, err := svc.Create(context.Background(), input)

	assert.Nil(t, folder)
	assert.ErrorIs(t, err, ErrFolderNameRequired)
}

func TestCreate_NameTooLong(t *testing.T) {
	svc, _ := newTestService()
	input := validCreateInput()
	input.Name = strings.Repeat("x", 256)

	folder, err := svc.Create(context.Background(), input)

	assert.Nil(t, folder)
	assert.ErrorIs(t, err, ErrFolderNameTooLong)
}

func TestCreate_InvalidSpaceType(t *testing.T) {
	svc, _ := newTestService()
	input := validCreateInput()
	input.SpaceType = "invalid"

	folder, err := svc.Create(context.Background(), input)

	assert.Nil(t, folder)
	assert.ErrorIs(t, err, ErrInvalidSpaceType)
}

func TestCreate_WithParent(t *testing.T) {
	svc, repo := newTestService()
	parent := seedFolder(repo, "Parent", false)

	input := validCreateInput()
	input.ParentID = &parent.ID
	input.SpaceType = parent.SpaceType
	input.SpaceID = parent.SpaceID

	folder, err := svc.Create(context.Background(), input)

	require.NoError(t, err)
	require.NotNil(t, folder)
	require.NotNil(t, folder.ParentID)
	assert.Equal(t, parent.ID, *folder.ParentID)
}

func TestGetByID_Success(t *testing.T) {
	svc, repo := newTestService()
	seeded := seedFolder(repo, "Test", false)

	folder, err := svc.GetByID(context.Background(), uuid.Nil, seeded.ID)

	require.NoError(t, err)
	assert.Equal(t, seeded.ID, folder.ID)
	assert.Equal(t, seeded.Name, folder.Name)
}

func TestGetByID_NotFound(t *testing.T) {
	svc, _ := newTestService()

	folder, err := svc.GetByID(context.Background(), uuid.Nil, uuid.New())

	assert.Nil(t, folder)
	assert.ErrorIs(t, err, ErrFolderNotFound)
}

func TestUpdate_Success(t *testing.T) {
	svc, repo := newTestService()
	seeded := seedFolder(repo, "Old Name", false)
	newName := "New Name"

	err := svc.Update(context.Background(), uuid.Nil, seeded.ID, UpdateInput{Name: &newName})

	require.NoError(t, err)
	assert.Equal(t, newName, repo.folders[seeded.ID].Name)
}

func TestUpdate_SystemFolderReject(t *testing.T) {
	svc, repo := newTestService()
	seeded := seedFolder(repo, "System Folder", true)
	newName := "Renamed"

	err := svc.Update(context.Background(), uuid.Nil, seeded.ID, UpdateInput{Name: &newName})

	assert.ErrorIs(t, err, ErrCannotRenameSystemFolder)
}

func TestDelete_Success(t *testing.T) {
	svc, repo := newTestService()
	seeded := seedFolder(repo, "To Delete", false)

	err := svc.Delete(context.Background(), uuid.Nil, seeded.ID)

	require.NoError(t, err)
	_, ok := repo.folders[seeded.ID]
	assert.False(t, ok)
}

func TestDelete_SystemFolder(t *testing.T) {
	svc, repo := newTestService()
	seeded := seedFolder(repo, "System", true)

	err := svc.Delete(context.Background(), uuid.Nil, seeded.ID)

	assert.ErrorIs(t, err, ErrCannotDeleteSystemFolder)
	// Folder should still exist
	_, ok := repo.folders[seeded.ID]
	assert.True(t, ok)
}

func TestGetPath_Success(t *testing.T) {
	svc, repo := newTestService()
	seeded := seedFolder(repo, "Deep Folder", false)

	path, err := svc.GetPath(context.Background(), seeded.ID)

	require.NoError(t, err)
	require.Len(t, path, 1)
	assert.Equal(t, seeded.ID, path[0].ID)
	assert.Equal(t, seeded.Name, path[0].Name)
}

func TestInitializeUserSpace_Success(t *testing.T) {
	svc, repo := newTestService()
	userID := uuid.New()

	err := svc.InitializeUserSpace(context.Background(), uuid.Nil, userID)

	require.NoError(t, err)

	// Should create 4 folders: root + 3 defaults (Dokumente, Bilder, Vorlagen)
	assert.Len(t, repo.folders, 4)

	var rootFound bool
	var subCount int
	for _, f := range repo.folders {
		assert.True(t, f.IsSystem)
		assert.Equal(t, models.FolderSpacePersonal, f.SpaceType)
		assert.Equal(t, userID, f.SpaceID)
		if f.ParentID == nil {
			rootFound = true
			assert.Equal(t, "Meine Dateien", f.Name)
		} else {
			subCount++
		}
	}
	assert.True(t, rootFound)
	assert.Equal(t, 3, subCount)
}

func TestInitializeTeamSpace_Success(t *testing.T) {
	svc, repo := newTestService()
	teamID := uuid.New()

	err := svc.InitializeTeamSpace(context.Background(), uuid.Nil, teamID, "Engineering")

	require.NoError(t, err)

	// Should create 4 folders: root + 3 defaults (Allgemein, Projekte, Vorlagen)
	assert.Len(t, repo.folders, 4)

	var rootFound bool
	for _, f := range repo.folders {
		assert.True(t, f.IsSystem)
		assert.Equal(t, models.FolderSpaceTeam, f.SpaceType)
		assert.Equal(t, teamID, f.SpaceID)
		if f.ParentID == nil {
			rootFound = true
			assert.Equal(t, "Engineering", f.Name)
		}
	}
	assert.True(t, rootFound)
}

func TestCircularReference(t *testing.T) {
	svc, repo := newTestService()

	parent := seedFolder(repo, "Parent", false)
	child := seedFolder(repo, "Child", false)
	child.ParentID = &parent.ID
	child.SpaceType = parent.SpaceType
	child.SpaceID = parent.SpaceID

	// Mark child as descendant of parent
	repo.descendants[parent.ID] = map[uuid.UUID]bool{child.ID: true}

	// Try to move parent into child (circular)
	err := svc.Update(context.Background(), uuid.Nil, parent.ID, UpdateInput{ParentID: &child.ID})

	assert.ErrorIs(t, err, ErrCircularParent)
}
