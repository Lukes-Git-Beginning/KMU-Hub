package file

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// --- Mock FileStore ---

type MockFileStore struct {
	uploaded map[string]bool
	failUpload   bool
	failDownload bool
	failDelete   bool
}

func NewMockFileStore() *MockFileStore {
	return &MockFileStore{uploaded: make(map[string]bool)}
}

func (m *MockFileStore) Upload(_ context.Context, key string, _ io.Reader, _ int64, _ string) error {
	if m.failUpload {
		return errors.New("store upload error")
	}
	m.uploaded[key] = true
	return nil
}

func (m *MockFileStore) Download(_ context.Context, key string) (io.ReadCloser, error) {
	if m.failDownload {
		return nil, errors.New("store download error")
	}
	return io.NopCloser(strings.NewReader("test content")), nil
}

func (m *MockFileStore) Delete(_ context.Context, key string) error {
	if m.failDelete {
		return errors.New("store delete error")
	}
	delete(m.uploaded, key)
	return nil
}

func (m *MockFileStore) GetPresignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://example.com/download/" + key, nil
}

// --- Mock Repository ---

type MockRepository struct {
	files         map[uuid.UUID]*models.DocumentFile
	versions      map[uuid.UUID][]*models.DocumentFileVersion
	entityLinks   map[uuid.UUID][]*models.DocumentEntityLink

	failCreate             bool
	failGetByID            bool
	failList               bool
	failUpdate             bool
	failSoftDelete         bool
	failCreateVersion      bool
	failCreateEntityLink   bool
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		files:       make(map[uuid.UUID]*models.DocumentFile),
		versions:    make(map[uuid.UUID][]*models.DocumentFileVersion),
		entityLinks: make(map[uuid.UUID][]*models.DocumentEntityLink),
	}
}

func (m *MockRepository) Create(_ context.Context, file *models.DocumentFile) error {
	if m.failCreate {
		return errors.New("repo create error")
	}
	m.files[file.ID] = file
	return nil
}

func (m *MockRepository) GetByID(_ context.Context, id uuid.UUID) (*models.DocumentFile, error) {
	if m.failGetByID {
		return nil, ErrFileNotFound
	}
	file, ok := m.files[id]
	if !ok {
		return nil, ErrFileNotFound
	}
	return file, nil
}

func (m *MockRepository) List(_ context.Context, _ ListFilter) ([]*models.DocumentFile, int, error) {
	if m.failList {
		return nil, 0, errors.New("repo list error")
	}
	var result []*models.DocumentFile
	for _, f := range m.files {
		result = append(result, f)
	}
	return result, len(result), nil
}

func (m *MockRepository) Update(_ context.Context, id uuid.UUID, input UpdateInput) error {
	if m.failUpdate {
		return errors.New("repo update error")
	}
	file, ok := m.files[id]
	if !ok {
		return ErrFileNotFound
	}
	if input.Filename != nil {
		file.Filename = *input.Filename
	}
	if input.FolderID != nil {
		file.FolderID = *input.FolderID
	}
	if input.IsFavorite != nil {
		file.IsFavorite = *input.IsFavorite
	}
	return nil
}

func (m *MockRepository) SoftDelete(_ context.Context, id uuid.UUID) error {
	if m.failSoftDelete {
		return errors.New("repo soft delete error")
	}
	file, ok := m.files[id]
	if !ok {
		return ErrFileNotFound
	}
	file.IsDeleted = true
	return nil
}

func (m *MockRepository) CreateVersion(_ context.Context, version *models.DocumentFileVersion) error {
	if m.failCreateVersion {
		return errors.New("repo create version error")
	}
	m.versions[version.FileID] = append(m.versions[version.FileID], version)
	return nil
}

func (m *MockRepository) ListVersions(_ context.Context, fileID uuid.UUID) ([]*models.DocumentFileVersion, error) {
	return m.versions[fileID], nil
}

func (m *MockRepository) GetVersion(_ context.Context, fileID uuid.UUID, versionNumber int) (*models.DocumentFileVersion, error) {
	for _, v := range m.versions[fileID] {
		if v.VersionNumber == versionNumber {
			return v, nil
		}
	}
	return nil, ErrVersionNotFound
}

func (m *MockRepository) UpdateCurrentVersion(_ context.Context, fileID uuid.UUID, versionNumber int) error {
	file, ok := m.files[fileID]
	if !ok {
		return ErrFileNotFound
	}
	file.CurrentVersion = versionNumber
	return nil
}

func (m *MockRepository) UpdateSearchContent(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

func (m *MockRepository) CreateEntityLink(_ context.Context, link *models.DocumentEntityLink) error {
	if m.failCreateEntityLink {
		return errors.New("repo create entity link error")
	}
	m.entityLinks[link.FileID] = append(m.entityLinks[link.FileID], link)
	return nil
}

func (m *MockRepository) DeleteEntityLink(_ context.Context, id uuid.UUID) error {
	for fileID, links := range m.entityLinks {
		for i, link := range links {
			if link.ID == id {
				m.entityLinks[fileID] = append(links[:i], links[i+1:]...)
				return nil
			}
		}
	}
	return nil
}

func (m *MockRepository) ListEntityLinks(_ context.Context, fileID uuid.UUID) ([]*models.DocumentEntityLink, error) {
	return m.entityLinks[fileID], nil
}

func (m *MockRepository) ListFilesByEntity(_ context.Context, _ string, _ uuid.UUID) ([]*models.DocumentFile, error) {
	return nil, nil
}

// --- Helper ---

func newTestService() (*Service, *MockRepository, *MockFileStore) {
	repo := NewMockRepository()
	store := NewMockFileStore()
	svc := NewService(repo, store, 10*1024*1024) // 10 MB max
	return svc, repo, store
}

func validUploadInput() UploadInput {
	return UploadInput{
		FolderID:  uuid.New(),
		Filename:  "report.pdf",
		MimeType:  "application/pdf",
		FileSize:  1024,
		Reader:    strings.NewReader("test content"),
		SpaceType: "personal",
		SpaceID:   uuid.New(),
		OwnerID:   uuid.New(),
	}
}

// seedFile inserts a file directly into the mock repo and returns it.
func seedFile(repo *MockRepository) *models.DocumentFile {
	file := &models.DocumentFile{
		ID:             uuid.New(),
		FolderID:       uuid.New(),
		Filename:       "existing.pdf",
		MimeType:       "application/pdf",
		FileSize:       2048,
		StorageKey:     "documents/personal/" + uuid.New().String() + "/existing.pdf",
		CurrentVersion: 1,
		OwnerID:        uuid.New(),
		IsDeleted:      false,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	repo.files[file.ID] = file
	return file
}

// --- Tests ---

func TestUpload_Success(t *testing.T) {
	svc, repo, store := newTestService()
	input := validUploadInput()

	file, err := svc.Upload(context.Background(), input)

	require.NoError(t, err)
	require.NotNil(t, file)
	assert.Equal(t, input.Filename, file.Filename)
	assert.Equal(t, input.MimeType, file.MimeType)
	assert.Equal(t, input.FileSize, file.FileSize)
	assert.Equal(t, input.OwnerID, file.OwnerID)
	assert.Equal(t, 1, file.CurrentVersion)
	assert.False(t, file.IsDeleted)

	// Verify stored in repo
	_, ok := repo.files[file.ID]
	assert.True(t, ok)

	// Verify uploaded to store
	assert.True(t, store.uploaded[file.StorageKey])
}

func TestUpload_EmptyFilename(t *testing.T) {
	svc, _, _ := newTestService()
	input := validUploadInput()
	input.Filename = ""

	file, err := svc.Upload(context.Background(), input)

	assert.Nil(t, file)
	assert.ErrorIs(t, err, ErrFilenameRequired)
}

func TestUpload_FilenameTooLong(t *testing.T) {
	svc, _, _ := newTestService()
	input := validUploadInput()
	input.Filename = strings.Repeat("a", 256)

	file, err := svc.Upload(context.Background(), input)

	assert.Nil(t, file)
	assert.ErrorIs(t, err, ErrFilenameTooLong)
}

func TestUpload_ZeroSize(t *testing.T) {
	svc, _, _ := newTestService()
	input := validUploadInput()
	input.FileSize = 0

	file, err := svc.Upload(context.Background(), input)

	assert.Nil(t, file)
	assert.ErrorIs(t, err, ErrFileSizeZero)
}

func TestUpload_TooLarge(t *testing.T) {
	svc, _, _ := newTestService()
	input := validUploadInput()
	input.FileSize = 20 * 1024 * 1024 // 20 MB, exceeds 10 MB max

	file, err := svc.Upload(context.Background(), input)

	assert.Nil(t, file)
	assert.ErrorIs(t, err, ErrFileTooLarge)
}

func TestUpload_StoreError(t *testing.T) {
	svc, _, store := newTestService()
	store.failUpload = true
	input := validUploadInput()

	file, err := svc.Upload(context.Background(), input)

	assert.Nil(t, file)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "store upload error")
}

func TestGetByID_Success(t *testing.T) {
	svc, repo, _ := newTestService()
	seeded := seedFile(repo)

	file, err := svc.GetByID(context.Background(), seeded.ID)

	require.NoError(t, err)
	assert.Equal(t, seeded.ID, file.ID)
	assert.Equal(t, seeded.Filename, file.Filename)
}

func TestGetByID_NotFound(t *testing.T) {
	svc, _, _ := newTestService()

	file, err := svc.GetByID(context.Background(), uuid.New())

	assert.Nil(t, file)
	assert.ErrorIs(t, err, ErrFileNotFound)
}

func TestDelete_Success(t *testing.T) {
	svc, repo, _ := newTestService()
	seeded := seedFile(repo)

	err := svc.Delete(context.Background(), seeded.ID)

	require.NoError(t, err)
	assert.True(t, repo.files[seeded.ID].IsDeleted)
}

func TestMove_Success(t *testing.T) {
	svc, repo, _ := newTestService()
	seeded := seedFile(repo)
	targetFolder := uuid.New()

	err := svc.Move(context.Background(), seeded.ID, targetFolder)

	require.NoError(t, err)
	assert.Equal(t, targetFolder, repo.files[seeded.ID].FolderID)
}

func TestCopy_Success(t *testing.T) {
	svc, repo, _ := newTestService()
	seeded := seedFile(repo)
	targetFolder := uuid.New()
	userID := uuid.New()

	copied, err := svc.Copy(context.Background(), seeded.ID, targetFolder, userID)

	require.NoError(t, err)
	require.NotNil(t, copied)
	assert.NotEqual(t, seeded.ID, copied.ID)
	assert.Equal(t, targetFolder, copied.FolderID)
	assert.Equal(t, seeded.Filename, copied.Filename)
	assert.Equal(t, userID, copied.OwnerID)

	// Verify both files exist in repo
	assert.Len(t, repo.files, 2)
}

func TestGetDownloadURL_Success(t *testing.T) {
	svc, repo, _ := newTestService()
	seeded := seedFile(repo)

	url, err := svc.GetDownloadURL(context.Background(), seeded.ID)

	require.NoError(t, err)
	assert.Contains(t, url, "https://example.com/download/")
	assert.Contains(t, url, seeded.StorageKey)
}

func TestGetDownloadURL_FileNotFound(t *testing.T) {
	svc, _, _ := newTestService()

	url, err := svc.GetDownloadURL(context.Background(), uuid.New())

	assert.Empty(t, url)
	assert.ErrorIs(t, err, ErrFileNotFound)
}

func TestLinkToEntity_Success(t *testing.T) {
	svc, repo, _ := newTestService()
	seeded := seedFile(repo)
	entityID := uuid.New()
	userID := uuid.New()

	err := svc.LinkToEntity(context.Background(), seeded.ID, "contact", entityID, userID)

	require.NoError(t, err)
	links := repo.entityLinks[seeded.ID]
	require.Len(t, links, 1)
	assert.Equal(t, "contact", links[0].EntityType)
	assert.Equal(t, entityID, links[0].EntityID)
}

func TestLinkToEntity_InvalidType(t *testing.T) {
	svc, repo, _ := newTestService()
	seeded := seedFile(repo)

	err := svc.LinkToEntity(context.Background(), seeded.ID, "invalid_type", uuid.New(), uuid.New())

	assert.ErrorIs(t, err, ErrInvalidEntityType)
}

func TestUpdate_Success(t *testing.T) {
	svc, repo, _ := newTestService()
	seeded := seedFile(repo)
	newName := "renamed.pdf"

	err := svc.Update(context.Background(), seeded.ID, UpdateInput{Filename: &newName})

	require.NoError(t, err)
	assert.Equal(t, newName, repo.files[seeded.ID].Filename)
}

func TestList_Success(t *testing.T) {
	svc, repo, _ := newTestService()
	seedFile(repo)
	seedFile(repo)

	files, total, err := svc.List(context.Background(), ListFilter{})

	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, files, 2)
}

func TestCreateVersion_Success(t *testing.T) {
	svc, repo, _ := newTestService()
	seeded := seedFile(repo)
	userID := uuid.New()

	version, err := svc.CreateVersion(context.Background(), seeded.ID, VersionInput{
		Reader:   strings.NewReader("new version content"),
		FileSize: 512,
		MimeType: "application/pdf",
		UserID:   userID,
	})

	require.NoError(t, err)
	require.NotNil(t, version)
	assert.Equal(t, 2, version.VersionNumber)
	assert.Equal(t, seeded.ID, version.FileID)
	assert.Equal(t, userID, version.CreatedBy)
}
