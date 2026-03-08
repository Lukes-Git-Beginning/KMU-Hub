package share

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// MockRepository implements Repository for testing.
type MockRepository struct {
	CreateFn            func(ctx context.Context, share *models.DocumentShare) error
	DeleteFn            func(ctx context.Context, id uuid.UUID) error
	ListByEntityFn      func(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.DocumentShare, error)
	ListSharedWithUserFn func(ctx context.Context, userID uuid.UUID, entityType string) ([]*models.DocumentShare, error)
	GetUserPermissionFn func(ctx context.Context, entityType string, entityID uuid.UUID, userID uuid.UUID) (*models.DocumentShare, error)
	HasAccessFn         func(ctx context.Context, entityType string, entityID uuid.UUID, userID uuid.UUID) (bool, string, error)
}

func (m *MockRepository) Create(ctx context.Context, share *models.DocumentShare) error {
	return m.CreateFn(ctx, share)
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return m.DeleteFn(ctx, id)
}

func (m *MockRepository) ListByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*models.DocumentShare, error) {
	return m.ListByEntityFn(ctx, entityType, entityID)
}

func (m *MockRepository) ListSharedWithUser(ctx context.Context, userID uuid.UUID, entityType string) ([]*models.DocumentShare, error) {
	return m.ListSharedWithUserFn(ctx, userID, entityType)
}

func (m *MockRepository) GetUserPermission(ctx context.Context, entityType string, entityID uuid.UUID, userID uuid.UUID) (*models.DocumentShare, error) {
	return m.GetUserPermissionFn(ctx, entityType, entityID, userID)
}

func (m *MockRepository) HasAccess(ctx context.Context, entityType string, entityID uuid.UUID, userID uuid.UUID) (bool, string, error) {
	return m.HasAccessFn(ctx, entityType, entityID, userID)
}

func TestShareEntity_Success(t *testing.T) {
	repo := &MockRepository{
		GetUserPermissionFn: func(_ context.Context, _ string, _ uuid.UUID, _ uuid.UUID) (*models.DocumentShare, error) {
			return nil, ErrShareNotFound
		},
		CreateFn: func(_ context.Context, _ *models.DocumentShare) error {
			return nil
		},
	}
	svc := NewService(repo)

	input := ShareInput{
		EntityType:       models.ShareEntityFile,
		EntityID:         uuid.New(),
		SharedWithUserID: uuid.New(),
		Permission:       models.SharePermissionRead,
		SharedBy:         uuid.New(),
	}

	result, err := svc.ShareEntity(context.Background(), input)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, input.EntityType, result.EntityType)
	assert.Equal(t, input.EntityID, result.EntityID)
	assert.Equal(t, input.SharedWithUserID, result.SharedWithUserID)
	assert.Equal(t, input.Permission, result.Permission)
	assert.Equal(t, input.SharedBy, result.SharedBy)
	assert.NotEqual(t, uuid.Nil, result.ID)
}

func TestShareEntity_InvalidEntityType(t *testing.T) {
	repo := &MockRepository{}
	svc := NewService(repo)

	input := ShareInput{
		EntityType:       "invalid",
		EntityID:         uuid.New(),
		SharedWithUserID: uuid.New(),
		Permission:       models.SharePermissionRead,
		SharedBy:         uuid.New(),
	}

	result, err := svc.ShareEntity(context.Background(), input)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrInvalidEntityType)
}

func TestShareEntity_InvalidPermission(t *testing.T) {
	repo := &MockRepository{}
	svc := NewService(repo)

	input := ShareInput{
		EntityType:       models.ShareEntityFile,
		EntityID:         uuid.New(),
		SharedWithUserID: uuid.New(),
		Permission:       "admin",
		SharedBy:         uuid.New(),
	}

	result, err := svc.ShareEntity(context.Background(), input)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrInvalidPermission)
}

func TestShareEntity_CannotShareWithSelf(t *testing.T) {
	repo := &MockRepository{}
	svc := NewService(repo)

	userID := uuid.New()
	input := ShareInput{
		EntityType:       models.ShareEntityFolder,
		EntityID:         uuid.New(),
		SharedWithUserID: userID,
		Permission:       models.SharePermissionWrite,
		SharedBy:         userID,
	}

	result, err := svc.ShareEntity(context.Background(), input)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrCannotShareWithSelf)
}

func TestShareEntity_AlreadyShared(t *testing.T) {
	existingShare := &models.DocumentShare{ID: uuid.New()}
	repo := &MockRepository{
		GetUserPermissionFn: func(_ context.Context, _ string, _ uuid.UUID, _ uuid.UUID) (*models.DocumentShare, error) {
			return existingShare, nil
		},
	}
	svc := NewService(repo)

	input := ShareInput{
		EntityType:       models.ShareEntityFile,
		EntityID:         uuid.New(),
		SharedWithUserID: uuid.New(),
		Permission:       models.SharePermissionRead,
		SharedBy:         uuid.New(),
	}

	result, err := svc.ShareEntity(context.Background(), input)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrAlreadyShared)
}

func TestUnshareEntity_Success(t *testing.T) {
	repo := &MockRepository{
		DeleteFn: func(_ context.Context, _ uuid.UUID) error {
			return nil
		},
	}
	svc := NewService(repo)

	err := svc.UnshareEntity(context.Background(), uuid.New())

	assert.NoError(t, err)
}

func TestUnshareEntity_NotFound(t *testing.T) {
	repo := &MockRepository{
		DeleteFn: func(_ context.Context, _ uuid.UUID) error {
			return ErrShareNotFound
		},
	}
	svc := NewService(repo)

	err := svc.UnshareEntity(context.Background(), uuid.New())

	assert.ErrorIs(t, err, ErrShareNotFound)
}

func TestListByEntity_Success(t *testing.T) {
	entityID := uuid.New()
	expected := []*models.DocumentShare{
		{ID: uuid.New(), EntityType: models.ShareEntityFile, EntityID: entityID},
		{ID: uuid.New(), EntityType: models.ShareEntityFile, EntityID: entityID},
	}
	repo := &MockRepository{
		ListByEntityFn: func(_ context.Context, _ string, _ uuid.UUID) ([]*models.DocumentShare, error) {
			return expected, nil
		},
	}
	svc := NewService(repo)

	result, err := svc.ListByEntity(context.Background(), models.ShareEntityFile, entityID)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, expected, result)
}

func TestListByEntity_Empty(t *testing.T) {
	repo := &MockRepository{
		ListByEntityFn: func(_ context.Context, _ string, _ uuid.UUID) ([]*models.DocumentShare, error) {
			return []*models.DocumentShare{}, nil
		},
	}
	svc := NewService(repo)

	result, err := svc.ListByEntity(context.Background(), models.ShareEntityFolder, uuid.New())

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestListSharedWithMe_Success(t *testing.T) {
	userID := uuid.New()
	expected := []*models.DocumentShare{
		{ID: uuid.New(), SharedWithUserID: userID, EntityType: models.ShareEntityFile},
	}
	repo := &MockRepository{
		ListSharedWithUserFn: func(_ context.Context, _ uuid.UUID, _ string) ([]*models.DocumentShare, error) {
			return expected, nil
		},
	}
	svc := NewService(repo)

	result, err := svc.ListSharedWithMe(context.Background(), userID, models.ShareEntityFile)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, expected, result)
}

func TestCheckAccess_HasAccess(t *testing.T) {
	repo := &MockRepository{
		HasAccessFn: func(_ context.Context, _ string, _ uuid.UUID, _ uuid.UUID) (bool, string, error) {
			return true, models.SharePermissionWrite, nil
		},
	}
	svc := NewService(repo)

	hasAccess, perm, err := svc.CheckAccess(context.Background(), models.ShareEntityFile, uuid.New(), uuid.New())

	require.NoError(t, err)
	assert.True(t, hasAccess)
	assert.Equal(t, models.SharePermissionWrite, perm)
}

func TestCheckAccess_NoAccess(t *testing.T) {
	repo := &MockRepository{
		HasAccessFn: func(_ context.Context, _ string, _ uuid.UUID, _ uuid.UUID) (bool, string, error) {
			return false, "", nil
		},
	}
	svc := NewService(repo)

	hasAccess, perm, err := svc.CheckAccess(context.Background(), models.ShareEntityFile, uuid.New(), uuid.New())

	require.NoError(t, err)
	assert.False(t, hasAccess)
	assert.Empty(t, perm)
}
