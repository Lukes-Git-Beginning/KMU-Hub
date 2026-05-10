package tag

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// MockRepository implements Repository for testing.
type MockRepository struct {
	CreateFn       func(ctx context.Context, tag *models.DocumentTag) error
	GetByIDFn      func(ctx context.Context, id uuid.UUID) (*models.DocumentTag, error)
	ListFn         func(ctx context.Context) ([]*models.DocumentTag, error)
	DeleteFn       func(ctx context.Context, id uuid.UUID) error
	TagFileFn      func(ctx context.Context, fileID, tagID uuid.UUID) error
	UntagFileFn    func(ctx context.Context, fileID, tagID uuid.UUID) error
	ListFileTagsFn func(ctx context.Context, fileID uuid.UUID) ([]*models.DocumentTag, error)
	ListFilesByTagFn func(ctx context.Context, tagID uuid.UUID, limit, offset int) ([]*models.DocumentFile, int, error)
}

func (m *MockRepository) Create(ctx context.Context, tag *models.DocumentTag) error {
	return m.CreateFn(ctx, tag)
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DocumentTag, error) {
	return m.GetByIDFn(ctx, id)
}

func (m *MockRepository) List(ctx context.Context) ([]*models.DocumentTag, error) {
	return m.ListFn(ctx)
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.DeleteFn(ctx, id)
}

func (m *MockRepository) TagFile(ctx context.Context, fileID, tagID uuid.UUID) error {
	return m.TagFileFn(ctx, fileID, tagID)
}

func (m *MockRepository) UntagFile(ctx context.Context, fileID, tagID uuid.UUID) error {
	return m.UntagFileFn(ctx, fileID, tagID)
}

func (m *MockRepository) ListFileTags(ctx context.Context, fileID uuid.UUID) ([]*models.DocumentTag, error) {
	return m.ListFileTagsFn(ctx, fileID)
}

func (m *MockRepository) ListFilesByTag(ctx context.Context, tagID uuid.UUID, limit, offset int) ([]*models.DocumentFile, int, error) {
	return m.ListFilesByTagFn(ctx, tagID, limit, offset)
}

func TestCreateTag_Success(t *testing.T) {
	repo := &MockRepository{
		CreateFn: func(_ context.Context, _ *models.DocumentTag) error {
			return nil
		},
	}
	svc := NewService(repo)

	tag, err := svc.CreateTag(context.Background(), "Important", "", uuid.New(), uuid.New())

	require.NoError(t, err)
	require.NotNil(t, tag)
	assert.Equal(t, "Important", tag.Name)
	assert.Nil(t, tag.Color)
	assert.NotEqual(t, uuid.Nil, tag.ID)
}

func TestCreateTag_EmptyName(t *testing.T) {
	repo := &MockRepository{}
	svc := NewService(repo)

	tag, err := svc.CreateTag(context.Background(), "", "", uuid.New(), uuid.New())

	assert.Nil(t, tag)
	assert.ErrorIs(t, err, ErrTagNameRequired)
}

func TestCreateTag_NameTooLong(t *testing.T) {
	repo := &MockRepository{}
	svc := NewService(repo)

	longName := strings.Repeat("a", 101)
	tag, err := svc.CreateTag(context.Background(), longName, "", uuid.New(), uuid.New())

	assert.Nil(t, tag)
	assert.ErrorIs(t, err, ErrTagNameTooLong)
}

func TestCreateTag_InvalidColor(t *testing.T) {
	repo := &MockRepository{}
	svc := NewService(repo)

	tag, err := svc.CreateTag(context.Background(), "Test", "red", uuid.New(), uuid.New())

	assert.Nil(t, tag)
	assert.ErrorIs(t, err, ErrInvalidColor)
}

func TestCreateTag_ValidColor(t *testing.T) {
	repo := &MockRepository{
		CreateFn: func(_ context.Context, _ *models.DocumentTag) error {
			return nil
		},
	}
	svc := NewService(repo)

	tag, err := svc.CreateTag(context.Background(), "Urgent", "#FF0000", uuid.New(), uuid.New())

	require.NoError(t, err)
	require.NotNil(t, tag)
	assert.Equal(t, "Urgent", tag.Name)
	require.NotNil(t, tag.Color)
	assert.Equal(t, "#FF0000", *tag.Color)
}

func TestCreateTag_EmptyColor(t *testing.T) {
	repo := &MockRepository{
		CreateFn: func(_ context.Context, _ *models.DocumentTag) error {
			return nil
		},
	}
	svc := NewService(repo)

	tag, err := svc.CreateTag(context.Background(), "NoColor", "", uuid.New(), uuid.New())

	require.NoError(t, err)
	require.NotNil(t, tag)
	assert.Nil(t, tag.Color)
}

func TestListTags_Success(t *testing.T) {
	expected := []*models.DocumentTag{
		{ID: uuid.New(), Name: "Alpha"},
		{ID: uuid.New(), Name: "Beta"},
		{ID: uuid.New(), Name: "Gamma"},
	}
	repo := &MockRepository{
		ListFn: func(_ context.Context) ([]*models.DocumentTag, error) {
			return expected, nil
		},
	}
	svc := NewService(repo)

	result, err := svc.ListTags(context.Background())

	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, expected, result)
}

func TestListTags_Empty(t *testing.T) {
	repo := &MockRepository{
		ListFn: func(_ context.Context) ([]*models.DocumentTag, error) {
			return []*models.DocumentTag{}, nil
		},
	}
	svc := NewService(repo)

	result, err := svc.ListTags(context.Background())

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestDeleteTag_Success(t *testing.T) {
	repo := &MockRepository{
		DeleteFn: func(_ context.Context, _ uuid.UUID) error {
			return nil
		},
	}
	svc := NewService(repo)

	err := svc.DeleteTag(context.Background(), uuid.New())

	assert.NoError(t, err)
}

func TestTagFile_Success(t *testing.T) {
	repo := &MockRepository{
		TagFileFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
			return nil
		},
	}
	svc := NewService(repo)

	err := svc.TagFile(context.Background(), uuid.New(), uuid.New())

	assert.NoError(t, err)
}

func TestUntagFile_Success(t *testing.T) {
	repo := &MockRepository{
		UntagFileFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
			return nil
		},
	}
	svc := NewService(repo)

	err := svc.UntagFile(context.Background(), uuid.New(), uuid.New())

	assert.NoError(t, err)
}

func TestListFileTags_Success(t *testing.T) {
	fileID := uuid.New()
	expected := []*models.DocumentTag{
		{ID: uuid.New(), Name: "Invoice"},
		{ID: uuid.New(), Name: "Q1"},
	}
	repo := &MockRepository{
		ListFileTagsFn: func(_ context.Context, _ uuid.UUID) ([]*models.DocumentTag, error) {
			return expected, nil
		},
	}
	svc := NewService(repo)

	result, err := svc.ListFileTags(context.Background(), fileID)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, expected, result)
}
