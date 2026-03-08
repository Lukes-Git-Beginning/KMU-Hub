package virtual

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// MockRepository implements Repository for testing.
type MockRepository struct {
	ListChatFilesFn        func(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.VirtualFile, int, error)
	ListEmailAttachmentsFn func(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.VirtualFile, int, error)
	ListTaskFilesFn        func(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.VirtualFile, int, error)
	ListAllFn              func(ctx context.Context, userID uuid.UUID, sourceType string, limit, offset int) ([]*models.VirtualFile, int, error)
}

func (m *MockRepository) ListChatFiles(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.VirtualFile, int, error) {
	return m.ListChatFilesFn(ctx, userID, limit, offset)
}

func (m *MockRepository) ListEmailAttachments(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.VirtualFile, int, error) {
	return m.ListEmailAttachmentsFn(ctx, userID, limit, offset)
}

func (m *MockRepository) ListTaskFiles(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.VirtualFile, int, error) {
	return m.ListTaskFilesFn(ctx, userID, limit, offset)
}

func (m *MockRepository) ListAll(ctx context.Context, userID uuid.UUID, sourceType string, limit, offset int) ([]*models.VirtualFile, int, error) {
	return m.ListAllFn(ctx, userID, sourceType, limit, offset)
}

func makeVirtualFile(sourceType string) *models.VirtualFile {
	return &models.VirtualFile{
		ID:             uuid.New(),
		Filename:       "test-file.pdf",
		MimeType:       "application/pdf",
		FileSize:       1024,
		StorageKey:     "files/" + uuid.New().String(),
		SourceType:     sourceType,
		SourceID:       uuid.New(),
		SourceName:     "Test Source",
		UploadedBy:     uuid.New(),
		UploadedByName: "Test User",
		CreatedAt:      time.Now(),
	}
}

func TestListVirtualFiles_Chat(t *testing.T) {
	chatFiles := []*models.VirtualFile{makeVirtualFile(models.VirtualFileSourceChat)}
	repo := &MockRepository{
		ListAllFn: func(_ context.Context, _ uuid.UUID, sourceType string, _ int, _ int) ([]*models.VirtualFile, int, error) {
			assert.Equal(t, "chat", sourceType)
			return chatFiles, 1, nil
		},
	}
	svc := NewService(repo)

	files, total, err := svc.ListVirtualFiles(context.Background(), uuid.New(), "chat", 20, 0)

	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, 1, total)
	assert.Equal(t, models.VirtualFileSourceChat, files[0].SourceType)
}

func TestListVirtualFiles_Email(t *testing.T) {
	emailFiles := []*models.VirtualFile{makeVirtualFile(models.VirtualFileSourceEmail)}
	repo := &MockRepository{
		ListAllFn: func(_ context.Context, _ uuid.UUID, sourceType string, _ int, _ int) ([]*models.VirtualFile, int, error) {
			assert.Equal(t, "email", sourceType)
			return emailFiles, 1, nil
		},
	}
	svc := NewService(repo)

	files, total, err := svc.ListVirtualFiles(context.Background(), uuid.New(), "email", 20, 0)

	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, 1, total)
	assert.Equal(t, models.VirtualFileSourceEmail, files[0].SourceType)
}

func TestListVirtualFiles_Task(t *testing.T) {
	taskFiles := []*models.VirtualFile{makeVirtualFile(models.VirtualFileSourceTask)}
	repo := &MockRepository{
		ListAllFn: func(_ context.Context, _ uuid.UUID, sourceType string, _ int, _ int) ([]*models.VirtualFile, int, error) {
			assert.Equal(t, "task", sourceType)
			return taskFiles, 1, nil
		},
	}
	svc := NewService(repo)

	files, total, err := svc.ListVirtualFiles(context.Background(), uuid.New(), "task", 20, 0)

	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, 1, total)
	assert.Equal(t, models.VirtualFileSourceTask, files[0].SourceType)
}

func TestListVirtualFiles_All(t *testing.T) {
	allFiles := []*models.VirtualFile{
		makeVirtualFile(models.VirtualFileSourceChat),
		makeVirtualFile(models.VirtualFileSourceEmail),
		makeVirtualFile(models.VirtualFileSourceTask),
	}
	repo := &MockRepository{
		ListAllFn: func(_ context.Context, _ uuid.UUID, sourceType string, _ int, _ int) ([]*models.VirtualFile, int, error) {
			assert.Equal(t, "", sourceType)
			return allFiles, 3, nil
		},
	}
	svc := NewService(repo)

	files, total, err := svc.ListVirtualFiles(context.Background(), uuid.New(), "", 20, 0)

	require.NoError(t, err)
	assert.Len(t, files, 3)
	assert.Equal(t, 3, total)
}

func TestListVirtualFiles_Empty(t *testing.T) {
	repo := &MockRepository{
		ListAllFn: func(_ context.Context, _ uuid.UUID, _ string, _ int, _ int) ([]*models.VirtualFile, int, error) {
			return []*models.VirtualFile{}, 0, nil
		},
	}
	svc := NewService(repo)

	files, total, err := svc.ListVirtualFiles(context.Background(), uuid.New(), "chat", 20, 0)

	require.NoError(t, err)
	assert.Empty(t, files)
	assert.Equal(t, 0, total)
}

func TestListVirtualFiles_RepoError(t *testing.T) {
	repoErr := errors.New("database connection failed")
	repo := &MockRepository{
		ListAllFn: func(_ context.Context, _ uuid.UUID, _ string, _ int, _ int) ([]*models.VirtualFile, int, error) {
			return nil, 0, repoErr
		},
	}
	svc := NewService(repo)

	files, total, err := svc.ListVirtualFiles(context.Background(), uuid.New(), "", 20, 0)

	assert.ErrorIs(t, err, repoErr)
	assert.Nil(t, files)
	assert.Equal(t, 0, total)
}
