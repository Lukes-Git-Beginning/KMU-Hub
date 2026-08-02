package file

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
)

// tenantCtx returns a context with the given tenant_id set, mirroring what
// the Auth middleware does in production.
func tenantCtx(tenantID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), middleware.TenantIDKey, tenantID.String())
}

// ============================================================================
// Mocks
// ============================================================================

type MockRepository struct {
	files       map[uuid.UUID]*models.ChatFile
	memberships map[string]models.ChannelRole // key: channelID-userID
	channels    map[uuid.UUID]struct{ archived bool }
	users       map[uuid.UUID]struct{ firstName, lastName string }
	quota       *models.StorageQuota
	createErr   error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		files:       make(map[uuid.UUID]*models.ChatFile),
		memberships: make(map[string]models.ChannelRole),
		channels:    make(map[uuid.UUID]struct{ archived bool }),
		users:       make(map[uuid.UUID]struct{ firstName, lastName string }),
		quota: &models.StorageQuota{
			ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			MaxBytes:  10737418240, // 10 GB
			UsedBytes: 0,
			UpdatedAt: time.Now(),
		},
	}
}

func mockKey(channelID, userID uuid.UUID) string {
	return channelID.String() + "-" + userID.String()
}

func (m *MockRepository) CreateFile(_ context.Context, file *models.ChatFile) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.files[file.ID] = file
	return nil
}

func (m *MockRepository) GetFileByID(_ context.Context, id uuid.UUID) (*models.ChatFile, error) {
	f, ok := m.files[id]
	if !ok {
		return nil, ErrFileNotFound
	}
	return f, nil
}

func (m *MockRepository) ListChannelFiles(_ context.Context, channelID uuid.UUID, limit, offset int) ([]*models.ChatFileWithUploader, int, error) {
	var result []*models.ChatFileWithUploader
	for _, f := range m.files {
		if f.ChannelID != channelID || f.IsDeleted {
			continue
		}
		user := m.users[f.UploadedBy]
		result = append(result, &models.ChatFileWithUploader{
			ChatFile:          *f,
			UploaderFirstName: user.firstName,
			UploaderLastName:  user.lastName,
		})
	}
	total := len(result)
	if offset >= total {
		return []*models.ChatFileWithUploader{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return result[offset:end], total, nil
}

func (m *MockRepository) GetFilesByMessageIDs(_ context.Context, messageIDs []uuid.UUID) (map[uuid.UUID][]models.ChatFileWithUploader, error) {
	idSet := make(map[uuid.UUID]bool)
	for _, id := range messageIDs {
		idSet[id] = true
	}
	result := make(map[uuid.UUID][]models.ChatFileWithUploader)
	for _, f := range m.files {
		if f.MessageID != nil && idSet[*f.MessageID] && !f.IsDeleted {
			user := m.users[f.UploadedBy]
			result[*f.MessageID] = append(result[*f.MessageID], models.ChatFileWithUploader{
				ChatFile:          *f,
				UploaderFirstName: user.firstName,
				UploaderLastName:  user.lastName,
			})
		}
	}
	return result, nil
}

func (m *MockRepository) DeleteFile(_ context.Context, id uuid.UUID) error {
	if f, ok := m.files[id]; ok {
		f.IsDeleted = true
		now := time.Now()
		f.DeletedAt = &now
	}
	return nil
}

func (m *MockRepository) IsChannelMember(_ context.Context, channelID, userID uuid.UUID) (bool, error) {
	_, ok := m.memberships[mockKey(channelID, userID)]
	return ok, nil
}

func (m *MockRepository) IsChannelArchived(_ context.Context, channelID uuid.UUID) (bool, error) {
	ch, ok := m.channels[channelID]
	if !ok {
		return false, ErrFileNotFound
	}
	return ch.archived, nil
}

func (m *MockRepository) GetChannelRole(_ context.Context, channelID, userID uuid.UUID) (models.ChannelRole, error) {
	role, ok := m.memberships[mockKey(channelID, userID)]
	if !ok {
		return "", ErrNotChannelMember
	}
	return role, nil
}

func (m *MockRepository) GetStorageQuota(_ context.Context, _ uuid.UUID) (*models.StorageQuota, error) {
	return m.quota, nil
}

func (m *MockRepository) IncrementUsedBytes(_ context.Context, _ uuid.UUID, bytes int64) error {
	m.quota.UsedBytes += bytes
	return nil
}

func (m *MockRepository) DecrementUsedBytes(_ context.Context, _ uuid.UUID, bytes int64) error {
	m.quota.UsedBytes -= bytes
	if m.quota.UsedBytes < 0 {
		m.quota.UsedBytes = 0
	}
	return nil
}

func (m *MockRepository) GetUserInfo(_ context.Context, userID uuid.UUID) (string, string, error) {
	u, ok := m.users[userID]
	if !ok {
		return "", "", nil
	}
	return u.firstName, u.lastName, nil
}

// Helpers
func (m *MockRepository) AddChannel(id uuid.UUID, archived bool) {
	m.channels[id] = struct{ archived bool }{archived}
}

func (m *MockRepository) AddMember(channelID, userID uuid.UUID, role models.ChannelRole) {
	m.memberships[mockKey(channelID, userID)] = role
}

func (m *MockRepository) AddUser(id uuid.UUID, firstName, lastName string) {
	m.users[id] = struct{ firstName, lastName string }{firstName, lastName}
}

func (m *MockRepository) AddFile(f *models.ChatFile) {
	m.files[f.ID] = f
}

// MockFileStore
type MockFileStore struct {
	files     map[string][]byte
	uploadErr error
	deleteErr error
}

func NewMockFileStore() *MockFileStore {
	return &MockFileStore{files: make(map[string][]byte)}
}

func (s *MockFileStore) Upload(_ context.Context, key string, reader io.Reader, _ int64, _ string) error {
	if s.uploadErr != nil {
		return s.uploadErr
	}
	data, _ := io.ReadAll(reader)
	s.files[key] = data
	return nil
}

func (s *MockFileStore) Download(_ context.Context, key string) (io.ReadCloser, error) {
	data, ok := s.files[key]
	if !ok {
		return nil, ErrFileNotFound
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (s *MockFileStore) Delete(_ context.Context, key string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	delete(s.files, key)
	return nil
}

func (s *MockFileStore) GetPresignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://minio.local/" + key, nil
}

func (s *MockFileStore) GetPresignedUploadURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://minio.local/upload/" + key, nil
}

// MockScanner
type MockScanner struct {
	scanErr error
}

func (s *MockScanner) Scan(_ context.Context, _ io.Reader, _ string) error {
	return s.scanErr
}

// MockThumbnailGenerator
type MockThumbnailGenerator struct {
	canGen    bool
	genErr    error
	genResult io.Reader
}

func (g *MockThumbnailGenerator) CanGenerate(_ string) bool {
	return g.canGen
}

func (g *MockThumbnailGenerator) Generate(_ context.Context, _ io.Reader, _ string) (io.Reader, error) {
	if g.genErr != nil {
		return nil, g.genErr
	}
	if g.genResult != nil {
		return g.genResult, nil
	}
	return bytes.NewReader([]byte("thumbnail-data")), nil
}

// ============================================================================
// Test helpers
// ============================================================================

func setupTestService() (*Service, *MockRepository, *MockFileStore, *MockScanner, *MockThumbnailGenerator) {
	repo := NewMockRepository()
	store := NewMockFileStore()
	scanner := &MockScanner{}
	thumbGen := &MockThumbnailGenerator{canGen: false}
	svc := NewService(repo, store, scanner, thumbGen, 50) // 50 MB limit
	return svc, repo, store, scanner, thumbGen
}

func setupChannel(repo *MockRepository) (uuid.UUID, uuid.UUID) {
	channelID := uuid.New()
	userID := uuid.New()
	repo.AddChannel(channelID, false)
	repo.AddMember(channelID, userID, models.ChannelRoleMember)
	repo.AddUser(userID, "John", "Doe")
	return channelID, userID
}

// ============================================================================
// Upload Tests
// ============================================================================

func TestService_Upload(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, repo, _, _, _ := setupTestService()
		channelID, userID := setupChannel(repo)

		result, err := svc.Upload(tenantCtx(uuid.Nil), UploadInput{
			ChannelID: channelID,
			Filename:  "document.pdf",
			MimeType:  "application/pdf",
			FileSize:  1024,
			Reader:    bytes.NewReader([]byte("pdf-content")),
			UserID:    userID,
		})

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, result.ID)
		assert.Equal(t, "document.pdf", result.Filename)
		assert.Equal(t, "application/pdf", result.MimeType)
		assert.Equal(t, int64(1024), result.FileSize)
		assert.Equal(t, "John", result.UploaderFirstName)
		assert.Equal(t, "Doe", result.UploaderLastName)
		assert.Equal(t, int64(1024), repo.quota.UsedBytes)
	})

	t.Run("success with thumbnail", func(t *testing.T) {
		svc, repo, store, _, thumbGen := setupTestService()
		thumbGen.canGen = true
		channelID, userID := setupChannel(repo)

		result, err := svc.Upload(tenantCtx(uuid.Nil), UploadInput{
			ChannelID: channelID,
			Filename:  "photo.jpg",
			MimeType:  "image/jpeg",
			FileSize:  2048,
			Reader:    bytes.NewReader([]byte("jpeg-content")),
			UserID:    userID,
		})

		require.NoError(t, err)
		assert.NotNil(t, result.ThumbnailKey)
		// Verify thumbnail was uploaded to store
		thumbUploaded := false
		for key := range store.files {
			if len(key) > 10 && key[len(key)-10:] == ".thumb.jpg" {
				thumbUploaded = true
				break
			}
		}
		assert.True(t, thumbUploaded)
	})

	t.Run("file too large", func(t *testing.T) {
		svc, repo, _, _, _ := setupTestService()
		channelID, userID := setupChannel(repo)

		_, err := svc.Upload(context.Background(), UploadInput{
			ChannelID: channelID,
			Filename:  "huge.pdf",
			MimeType:  "application/pdf",
			FileSize:  100 * 1024 * 1024, // 100 MB
			Reader:    bytes.NewReader([]byte("data")),
			UserID:    userID,
		})

		assert.Equal(t, ErrFileTooLarge, err)
	})

	t.Run("invalid MIME type", func(t *testing.T) {
		svc, repo, _, _, _ := setupTestService()
		channelID, userID := setupChannel(repo)

		_, err := svc.Upload(context.Background(), UploadInput{
			ChannelID: channelID,
			Filename:  "script.exe",
			MimeType:  "application/x-msdownload",
			FileSize:  1024,
			Reader:    bytes.NewReader([]byte("data")),
			UserID:    userID,
		})

		assert.Equal(t, ErrInvalidMimeType, err)
	})

	t.Run("quota exceeded", func(t *testing.T) {
		svc, repo, _, _, _ := setupTestService()
		channelID, userID := setupChannel(repo)
		repo.quota.UsedBytes = repo.quota.MaxBytes - 100 // Almost full

		_, err := svc.Upload(tenantCtx(uuid.Nil), UploadInput{
			ChannelID: channelID,
			Filename:  "doc.pdf",
			MimeType:  "application/pdf",
			FileSize:  1024,
			Reader:    bytes.NewReader([]byte("data")),
			UserID:    userID,
		})

		assert.Equal(t, ErrQuotaExceeded, err)
	})

	t.Run("not a member", func(t *testing.T) {
		svc, repo, _, _, _ := setupTestService()
		channelID := uuid.New()
		repo.AddChannel(channelID, false)
		nonMemberID := uuid.New()

		_, err := svc.Upload(context.Background(), UploadInput{
			ChannelID: channelID,
			Filename:  "doc.pdf",
			MimeType:  "application/pdf",
			FileSize:  1024,
			Reader:    bytes.NewReader([]byte("data")),
			UserID:    nonMemberID,
		})

		assert.Equal(t, ErrNotChannelMember, err)
	})

	t.Run("channel archived", func(t *testing.T) {
		svc, repo, _, _, _ := setupTestService()
		channelID := uuid.New()
		userID := uuid.New()
		repo.AddChannel(channelID, true) // archived
		repo.AddMember(channelID, userID, models.ChannelRoleMember)

		_, err := svc.Upload(context.Background(), UploadInput{
			ChannelID: channelID,
			Filename:  "doc.pdf",
			MimeType:  "application/pdf",
			FileSize:  1024,
			Reader:    bytes.NewReader([]byte("data")),
			UserID:    userID,
		})

		assert.Equal(t, ErrChannelArchived, err)
	})

	t.Run("scan failed", func(t *testing.T) {
		svc, repo, _, scanner, _ := setupTestService()
		channelID, userID := setupChannel(repo)
		scanner.scanErr = ErrScanFailed

		_, err := svc.Upload(tenantCtx(uuid.Nil), UploadInput{
			ChannelID: channelID,
			Filename:  "virus.pdf",
			MimeType:  "application/pdf",
			FileSize:  1024,
			Reader:    bytes.NewReader([]byte("data")),
			UserID:    userID,
		})

		assert.Equal(t, ErrScanFailed, err)
	})
}

// ============================================================================
// GetDownloadURL Tests
// ============================================================================

func TestService_GetDownloadURL(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, repo, _, _, _ := setupTestService()
		channelID, userID := setupChannel(repo)

		fileID := uuid.New()
		repo.AddFile(&models.ChatFile{
			ID:         fileID,
			ChannelID:  channelID,
			StorageKey: "test/key",
			UploadedBy: userID,
		})

		url, err := svc.GetDownloadURL(context.Background(), fileID, userID)

		require.NoError(t, err)
		assert.Contains(t, url, "test/key")
	})

	t.Run("file deleted", func(t *testing.T) {
		svc, repo, _, _, _ := setupTestService()
		channelID, userID := setupChannel(repo)

		fileID := uuid.New()
		repo.AddFile(&models.ChatFile{
			ID:         fileID,
			ChannelID:  channelID,
			StorageKey: "test/key",
			UploadedBy: userID,
			IsDeleted:  true,
		})

		_, err := svc.GetDownloadURL(context.Background(), fileID, userID)

		assert.Equal(t, ErrFileDeleted, err)
	})

	t.Run("not a member", func(t *testing.T) {
		svc, repo, _, _, _ := setupTestService()
		channelID := uuid.New()
		uploaderID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, uploaderID, models.ChannelRoleMember)

		fileID := uuid.New()
		repo.AddFile(&models.ChatFile{
			ID:         fileID,
			ChannelID:  channelID,
			StorageKey: "test/key",
			UploadedBy: uploaderID,
		})

		nonMemberID := uuid.New()
		_, err := svc.GetDownloadURL(context.Background(), fileID, nonMemberID)

		assert.Equal(t, ErrNotChannelMember, err)
	})
}

// ============================================================================
// GetThumbnailURL Tests
// ============================================================================

func TestService_GetThumbnailURL(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, repo, _, _, _ := setupTestService()
		channelID, userID := setupChannel(repo)

		thumbKey := "test/key.thumb.jpg"
		fileID := uuid.New()
		repo.AddFile(&models.ChatFile{
			ID:           fileID,
			ChannelID:    channelID,
			StorageKey:   "test/key",
			ThumbnailKey: &thumbKey,
			UploadedBy:   userID,
		})

		url, err := svc.GetThumbnailURL(context.Background(), fileID, userID)

		require.NoError(t, err)
		assert.Contains(t, url, "thumb.jpg")
	})

	t.Run("no thumbnail", func(t *testing.T) {
		svc, repo, _, _, _ := setupTestService()
		channelID, userID := setupChannel(repo)

		fileID := uuid.New()
		repo.AddFile(&models.ChatFile{
			ID:         fileID,
			ChannelID:  channelID,
			StorageKey: "test/key",
			UploadedBy: userID,
		})

		_, err := svc.GetThumbnailURL(context.Background(), fileID, userID)

		assert.Equal(t, ErrFileNotFound, err)
	})
}

// ============================================================================
// ListChannelFiles Tests
// ============================================================================

func TestService_ListChannelFiles(t *testing.T) {
	t.Run("success with pagination", func(t *testing.T) {
		svc, repo, _, _, _ := setupTestService()
		channelID, userID := setupChannel(repo)

		for i := 0; i < 3; i++ {
			repo.AddFile(&models.ChatFile{
				ID:         uuid.New(),
				ChannelID:  channelID,
				Filename:   "file.pdf",
				UploadedBy: userID,
				CreatedAt:  time.Now(),
			})
		}

		files, total, err := svc.ListChannelFiles(context.Background(), channelID, userID, 1, 20)

		require.NoError(t, err)
		assert.Len(t, files, 3)
		assert.Equal(t, 3, total)
	})

	t.Run("empty channel", func(t *testing.T) {
		svc, repo, _, _, _ := setupTestService()
		channelID, userID := setupChannel(repo)

		files, total, err := svc.ListChannelFiles(context.Background(), channelID, userID, 1, 20)

		require.NoError(t, err)
		assert.Empty(t, files)
		assert.Equal(t, 0, total)
	})

	t.Run("not a member", func(t *testing.T) {
		svc, repo, _, _, _ := setupTestService()
		channelID := uuid.New()
		repo.AddChannel(channelID, false)
		nonMemberID := uuid.New()

		_, _, err := svc.ListChannelFiles(context.Background(), channelID, nonMemberID, 1, 20)

		assert.Equal(t, ErrNotChannelMember, err)
	})
}

// ============================================================================
// Delete Tests
// ============================================================================

func TestService_Delete(t *testing.T) {
	t.Run("success as uploader", func(t *testing.T) {
		svc, repo, _, _, _ := setupTestService()
		channelID, userID := setupChannel(repo)
		repo.quota.UsedBytes = 5000

		fileID := uuid.New()
		repo.AddFile(&models.ChatFile{
			ID:         fileID,
			ChannelID:  channelID,
			StorageKey: "test/key",
			UploadedBy: userID,
			FileSize:   1000,
		})

		err := svc.Delete(context.Background(), fileID, userID)

		require.NoError(t, err)
		assert.True(t, repo.files[fileID].IsDeleted)
		assert.Equal(t, int64(4000), repo.quota.UsedBytes)
	})

	t.Run("success as admin", func(t *testing.T) {
		svc, repo, _, _, _ := setupTestService()
		channelID := uuid.New()
		uploaderID := uuid.New()
		adminID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, uploaderID, models.ChannelRoleMember)
		repo.AddMember(channelID, adminID, models.ChannelRoleAdmin)

		fileID := uuid.New()
		repo.AddFile(&models.ChatFile{
			ID:         fileID,
			ChannelID:  channelID,
			StorageKey: "test/key",
			UploadedBy: uploaderID,
			FileSize:   500,
		})

		err := svc.Delete(context.Background(), fileID, adminID)

		require.NoError(t, err)
		assert.True(t, repo.files[fileID].IsDeleted)
	})

	t.Run("not authorized - other member", func(t *testing.T) {
		svc, repo, _, _, _ := setupTestService()
		channelID := uuid.New()
		uploaderID := uuid.New()
		otherID := uuid.New()
		repo.AddChannel(channelID, false)
		repo.AddMember(channelID, uploaderID, models.ChannelRoleMember)
		repo.AddMember(channelID, otherID, models.ChannelRoleMember)

		fileID := uuid.New()
		repo.AddFile(&models.ChatFile{
			ID:         fileID,
			ChannelID:  channelID,
			StorageKey: "test/key",
			UploadedBy: uploaderID,
		})

		err := svc.Delete(context.Background(), fileID, otherID)

		assert.Equal(t, ErrNotAuthorized, err)
	})

	t.Run("idempotent - already deleted", func(t *testing.T) {
		svc, repo, _, _, _ := setupTestService()
		channelID, userID := setupChannel(repo)

		fileID := uuid.New()
		repo.AddFile(&models.ChatFile{
			ID:         fileID,
			ChannelID:  channelID,
			StorageKey: "test/key",
			UploadedBy: userID,
			IsDeleted:  true,
		})

		err := svc.Delete(context.Background(), fileID, userID)

		require.NoError(t, err) // idempotent
	})

	t.Run("file not found", func(t *testing.T) {
		svc, _, _, _, _ := setupTestService()

		err := svc.Delete(context.Background(), uuid.New(), uuid.New())

		assert.Equal(t, ErrFileNotFound, err)
	})
}
