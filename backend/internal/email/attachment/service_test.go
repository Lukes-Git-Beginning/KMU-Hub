package attachment

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

// MockRepository implements Repository for testing.
type MockRepository struct {
	CreateFn          func(ctx context.Context, att *models.EmailAttachment) error
	GetByMessageFn    func(ctx context.Context, messageID uuid.UUID, tenantID uuid.UUID) ([]*models.EmailAttachment, error)
	GetByIDFn         func(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.EmailAttachment, error)
	GetMinIOKeyByIDFn func(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (string, error)
}

func (m *MockRepository) Create(ctx context.Context, att *models.EmailAttachment) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, att)
	}
	return nil
}

func (m *MockRepository) GetByMessage(ctx context.Context, messageID uuid.UUID, tenantID uuid.UUID) ([]*models.EmailAttachment, error) {
	if m.GetByMessageFn != nil {
		return m.GetByMessageFn(ctx, messageID, tenantID)
	}
	return nil, nil
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.EmailAttachment, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id, tenantID)
	}
	return nil, ErrAttachmentNotFound
}

func (m *MockRepository) GetMinIOKeyByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (string, error) {
	if m.GetMinIOKeyByIDFn != nil {
		return m.GetMinIOKeyByIDFn(ctx, id, tenantID)
	}
	return "", ErrAttachmentNotFound
}

// MockObjectStore implements ObjectStore for testing.
type MockObjectStore struct {
	UploadFn          func(ctx context.Context, accountID uuid.UUID, messageUID uint32, filename string, reader io.Reader, size int64, contentType string) (string, error)
	GetPresignedURLFn func(ctx context.Context, minioKey string, expiry time.Duration) (string, error)
	DeleteFn          func(ctx context.Context, minioKey string) error
}

func (m *MockObjectStore) Upload(ctx context.Context, accountID uuid.UUID, messageUID uint32, filename string, reader io.Reader, size int64, contentType string) (string, error) {
	if m.UploadFn != nil {
		return m.UploadFn(ctx, accountID, messageUID, filename, reader, size, contentType)
	}
	return "attachments/" + accountID.String() + "/" + filename, nil
}

func (m *MockObjectStore) GetPresignedURL(ctx context.Context, minioKey string, expiry time.Duration) (string, error) {
	if m.GetPresignedURLFn != nil {
		return m.GetPresignedURLFn(ctx, minioKey, expiry)
	}
	return "https://minio.example.com/" + minioKey + "?presigned=true", nil
}

func (m *MockObjectStore) Delete(ctx context.Context, minioKey string) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, minioKey)
	}
	return nil
}

var testTenantID = uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

func TestCreateFromStream_Success(t *testing.T) {
	accountID := uuid.New()
	messageID := uuid.New()
	var messageUID uint32 = 42

	repo := &MockRepository{}
	store := &MockObjectStore{}
	svc := NewService(repo, store)

	reader := strings.NewReader("test content")
	att, err := svc.CreateFromStream(
		context.Background(),
		messageID, accountID, messageUID,
		"report.pdf", "application/pdf",
		int64(reader.Len()), reader,
		"", false,
		testTenantID,
	)

	require.NoError(t, err)
	require.NotNil(t, att)
	assert.Equal(t, messageID, att.MessageID)
	assert.Equal(t, testTenantID, att.TenantID)
	assert.Equal(t, "report.pdf", att.Filename)
	assert.Equal(t, "application/pdf", att.ContentType)
	assert.NotEmpty(t, att.MinIOKey)
	assert.False(t, att.IsInline)
}

func TestCreateFromStream_UploadError(t *testing.T) {
	uploadErr := errors.New("storage unavailable")
	store := &MockObjectStore{
		UploadFn: func(_ context.Context, _ uuid.UUID, _ uint32, _ string, _ io.Reader, _ int64, _ string) (string, error) {
			return "", uploadErr
		},
	}
	svc := NewService(&MockRepository{}, store)

	att, err := svc.CreateFromStream(
		context.Background(),
		uuid.New(), uuid.New(), 1,
		"file.txt", "text/plain",
		12, strings.NewReader("test content"),
		"", false,
		testTenantID,
	)

	assert.Nil(t, att)
	assert.Error(t, err)
	assert.ErrorIs(t, err, uploadErr)
}

func TestCreateFromStream_RepoError_CleansUpMinIO(t *testing.T) {
	repoErr := errors.New("db insert failed")
	deletedKey := ""

	store := &MockObjectStore{
		DeleteFn: func(_ context.Context, minioKey string) error {
			deletedKey = minioKey
			return nil
		},
	}
	repo := &MockRepository{
		CreateFn: func(_ context.Context, _ *models.EmailAttachment) error {
			return repoErr
		},
	}
	svc := NewService(repo, store)

	att, err := svc.CreateFromStream(
		context.Background(),
		uuid.New(), uuid.New(), 1,
		"file.txt", "text/plain",
		12, strings.NewReader("test content"),
		"", false,
		testTenantID,
	)

	assert.Nil(t, att)
	assert.Error(t, err)
	assert.ErrorIs(t, err, repoErr)
	assert.NotEmpty(t, deletedKey, "MinIO object should be cleaned up on DB failure")
}

func TestGetByMessage_Success(t *testing.T) {
	messageID := uuid.New()
	expected := []*models.EmailAttachment{
		{ID: uuid.New(), TenantID: testTenantID, MessageID: messageID, Filename: "a.pdf"},
		{ID: uuid.New(), TenantID: testTenantID, MessageID: messageID, Filename: "b.png"},
	}
	repo := &MockRepository{
		GetByMessageFn: func(_ context.Context, mid uuid.UUID, tid uuid.UUID) ([]*models.EmailAttachment, error) {
			if mid == messageID && tid == testTenantID {
				return expected, nil
			}
			return nil, nil
		},
	}
	svc := NewService(repo, &MockObjectStore{})

	result, err := svc.GetByMessage(context.Background(), messageID, testTenantID)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestGetByMessage_Empty(t *testing.T) {
	repo := &MockRepository{
		GetByMessageFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]*models.EmailAttachment, error) {
			return []*models.EmailAttachment{}, nil
		},
	}
	svc := NewService(repo, &MockObjectStore{})

	result, err := svc.GetByMessage(context.Background(), uuid.New(), testTenantID)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetDownloadURL_Success(t *testing.T) {
	attID := uuid.New()
	minioKey := "email-attachments/acc/42/file.pdf"

	repo := &MockRepository{
		GetMinIOKeyByIDFn: func(_ context.Context, id uuid.UUID, tid uuid.UUID) (string, error) {
			if id == attID && tid == testTenantID {
				return minioKey, nil
			}
			return "", ErrAttachmentNotFound
		},
	}
	store := &MockObjectStore{
		GetPresignedURLFn: func(_ context.Context, key string, _ time.Duration) (string, error) {
			return "https://minio.example.com/" + key + "?token=abc", nil
		},
	}
	svc := NewService(repo, store)

	url, err := svc.GetDownloadURL(context.Background(), attID, testTenantID)
	require.NoError(t, err)
	assert.Contains(t, url, minioKey)
	assert.Contains(t, url, "token=abc")
}

func TestGetDownloadURL_NotFound(t *testing.T) {
	repo := &MockRepository{
		GetMinIOKeyByIDFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (string, error) {
			return "", ErrAttachmentNotFound
		},
	}
	svc := NewService(repo, &MockObjectStore{})

	url, err := svc.GetDownloadURL(context.Background(), uuid.New(), testTenantID)
	assert.Empty(t, url)
	assert.ErrorIs(t, err, ErrAttachmentNotFound)
}

func TestCreateFromStream_InlineAttachment(t *testing.T) {
	repo := &MockRepository{}
	store := &MockObjectStore{}
	svc := NewService(repo, store)

	att, err := svc.CreateFromStream(
		context.Background(),
		uuid.New(), uuid.New(), 10,
		"logo.png", "image/png",
		256, strings.NewReader("png-data"),
		"cid:logo@company", true,
		testTenantID,
	)

	require.NoError(t, err)
	assert.True(t, att.IsInline)
	assert.Equal(t, "cid:logo@company", att.ContentID)
}

func TestGetByMessage_TenantIsolation(t *testing.T) {
	tenantA := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	tenantB := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	messageID := uuid.New()
	expected := []*models.EmailAttachment{
		{ID: uuid.New(), TenantID: tenantA, MessageID: messageID, Filename: "secret.pdf"},
	}
	repo := &MockRepository{
		GetByMessageFn: func(_ context.Context, mid uuid.UUID, tid uuid.UUID) ([]*models.EmailAttachment, error) {
			if mid == messageID && tid == tenantA {
				return expected, nil
			}
			return nil, nil
		},
	}
	svc := NewService(repo, &MockObjectStore{})

	// Tenant A sees their attachment
	result, err := svc.GetByMessage(context.Background(), messageID, tenantA)
	require.NoError(t, err)
	assert.Len(t, result, 1)

	// Tenant B sees nothing
	result, err = svc.GetByMessage(context.Background(), messageID, tenantB)
	require.NoError(t, err)
	assert.Empty(t, result)
}
