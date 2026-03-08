package signature

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
	CreateFn              func(ctx context.Context, sig *models.EmailSignature) error
	GetByIDFn             func(ctx context.Context, id uuid.UUID) (*models.EmailSignature, error)
	GetDefaultFn          func(ctx context.Context, userID uuid.UUID) (*models.EmailSignature, error)
	ListByUserFn          func(ctx context.Context, userID uuid.UUID) ([]*models.EmailSignature, error)
	UpdateFn              func(ctx context.Context, sig *models.EmailSignature) error
	DeleteFn              func(ctx context.Context, id uuid.UUID) error
	ClearDefaultForUserFn func(ctx context.Context, userID uuid.UUID) error
	CountByUserFn         func(ctx context.Context, userID uuid.UUID) (int, error)
}

func (m *MockRepository) Create(ctx context.Context, sig *models.EmailSignature) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, sig)
	}
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.EmailSignature, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, ErrSignatureNotFound
}

func (m *MockRepository) GetDefault(ctx context.Context, userID uuid.UUID) (*models.EmailSignature, error) {
	if m.GetDefaultFn != nil {
		return m.GetDefaultFn(ctx, userID)
	}
	return nil, ErrSignatureNotFound
}

func (m *MockRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.EmailSignature, error) {
	if m.ListByUserFn != nil {
		return m.ListByUserFn(ctx, userID)
	}
	return nil, nil
}

func (m *MockRepository) Update(ctx context.Context, sig *models.EmailSignature) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, sig)
	}
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}

func (m *MockRepository) ClearDefaultForUser(ctx context.Context, userID uuid.UUID) error {
	if m.ClearDefaultForUserFn != nil {
		return m.ClearDefaultForUserFn(ctx, userID)
	}
	return nil
}

func (m *MockRepository) CountByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	if m.CountByUserFn != nil {
		return m.CountByUserFn(ctx, userID)
	}
	return 0, nil
}

func TestCreate_Success(t *testing.T) {
	repo := &MockRepository{
		CountByUserFn: func(_ context.Context, _ uuid.UUID) (int, error) {
			return 1, nil
		},
	}
	svc := NewService(repo)

	sig, err := svc.Create(context.Background(), uuid.New(), "Work Sig", "<p>Regards</p>")
	require.NoError(t, err)
	require.NotNil(t, sig)
	assert.Equal(t, "Work Sig", sig.Name)
	assert.Equal(t, "<p>Regards</p>", sig.HTMLContent)
	assert.False(t, sig.IsDefault)
}

func TestCreate_FirstSignatureAutoDefault(t *testing.T) {
	repo := &MockRepository{
		CountByUserFn: func(_ context.Context, _ uuid.UUID) (int, error) {
			return 0, nil
		},
	}
	svc := NewService(repo)

	sig, err := svc.Create(context.Background(), uuid.New(), "First Sig", "<p>Hi</p>")
	require.NoError(t, err)
	assert.True(t, sig.IsDefault)
}

func TestCreate_SubsequentNotDefault(t *testing.T) {
	repo := &MockRepository{
		CountByUserFn: func(_ context.Context, _ uuid.UUID) (int, error) {
			return 1, nil
		},
	}
	svc := NewService(repo)

	sig, err := svc.Create(context.Background(), uuid.New(), "Second Sig", "<p>Bye</p>")
	require.NoError(t, err)
	assert.False(t, sig.IsDefault)
}

func TestCreate_RepoError(t *testing.T) {
	repoErr := errors.New("db failure")
	repo := &MockRepository{
		CountByUserFn: func(_ context.Context, _ uuid.UUID) (int, error) {
			return 0, nil
		},
		CreateFn: func(_ context.Context, _ *models.EmailSignature) error {
			return repoErr
		},
	}
	svc := NewService(repo)

	sig, err := svc.Create(context.Background(), uuid.New(), "Fail", "<p>x</p>")
	assert.Nil(t, sig)
	assert.Error(t, err)
	assert.ErrorIs(t, err, repoErr)
}

func TestGetByID_Success(t *testing.T) {
	sigID := uuid.New()
	expected := &models.EmailSignature{
		ID:   sigID,
		Name: "My Sig",
	}
	repo := &MockRepository{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*models.EmailSignature, error) {
			if id == sigID {
				return expected, nil
			}
			return nil, ErrSignatureNotFound
		},
	}
	svc := NewService(repo)

	sig, err := svc.GetByID(context.Background(), sigID)
	require.NoError(t, err)
	assert.Equal(t, "My Sig", sig.Name)
}

func TestGetByID_NotFound(t *testing.T) {
	repo := &MockRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*models.EmailSignature, error) {
			return nil, ErrSignatureNotFound
		},
	}
	svc := NewService(repo)

	sig, err := svc.GetByID(context.Background(), uuid.New())
	assert.Nil(t, sig)
	assert.ErrorIs(t, err, ErrSignatureNotFound)
}

func TestGetDefault_Success(t *testing.T) {
	userID := uuid.New()
	expected := &models.EmailSignature{
		ID:        uuid.New(),
		UserID:    userID,
		IsDefault: true,
	}
	repo := &MockRepository{
		GetDefaultFn: func(_ context.Context, uid uuid.UUID) (*models.EmailSignature, error) {
			if uid == userID {
				return expected, nil
			}
			return nil, ErrSignatureNotFound
		},
	}
	svc := NewService(repo)

	sig, err := svc.GetDefault(context.Background(), userID)
	require.NoError(t, err)
	assert.True(t, sig.IsDefault)
}

func TestListByUser_Success(t *testing.T) {
	userID := uuid.New()
	sigs := []*models.EmailSignature{
		{ID: uuid.New(), UserID: userID, Name: "Sig A"},
		{ID: uuid.New(), UserID: userID, Name: "Sig B"},
	}
	repo := &MockRepository{
		ListByUserFn: func(_ context.Context, _ uuid.UUID) ([]*models.EmailSignature, error) {
			return sigs, nil
		},
	}
	svc := NewService(repo)

	result, err := svc.ListByUser(context.Background(), userID)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestUpdate_Success(t *testing.T) {
	sigID := uuid.New()
	existing := &models.EmailSignature{
		ID:          sigID,
		UserID:      uuid.New(),
		Name:        "Old Name",
		HTMLContent: "<p>old</p>",
		UpdatedAt:   time.Now().Add(-1 * time.Hour),
	}
	repo := &MockRepository{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*models.EmailSignature, error) {
			if id == sigID {
				return existing, nil
			}
			return nil, ErrSignatureNotFound
		},
	}
	svc := NewService(repo)

	updated, err := svc.Update(context.Background(), sigID, "New Name", "<p>new</p>")
	require.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
	assert.Equal(t, "<p>new</p>", updated.HTMLContent)
}

func TestDelete_Success(t *testing.T) {
	called := false
	repo := &MockRepository{
		DeleteFn: func(_ context.Context, _ uuid.UUID) error {
			called = true
			return nil
		},
	}
	svc := NewService(repo)

	err := svc.Delete(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.True(t, called)
}

func TestSetDefault_Success(t *testing.T) {
	userID := uuid.New()
	sigID := uuid.New()
	existing := &models.EmailSignature{
		ID:        sigID,
		UserID:    userID,
		IsDefault: false,
	}
	clearCalled := false
	repo := &MockRepository{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*models.EmailSignature, error) {
			if id == sigID {
				return existing, nil
			}
			return nil, ErrSignatureNotFound
		},
		ClearDefaultForUserFn: func(_ context.Context, uid uuid.UUID) error {
			assert.Equal(t, userID, uid)
			clearCalled = true
			return nil
		},
		UpdateFn: func(_ context.Context, sig *models.EmailSignature) error {
			assert.True(t, sig.IsDefault)
			return nil
		},
	}
	svc := NewService(repo)

	err := svc.SetDefault(context.Background(), sigID)
	require.NoError(t, err)
	assert.True(t, clearCalled)
}

func TestSetDefault_NotFound(t *testing.T) {
	repo := &MockRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*models.EmailSignature, error) {
			return nil, ErrSignatureNotFound
		},
	}
	svc := NewService(repo)

	err := svc.SetDefault(context.Background(), uuid.New())
	assert.ErrorIs(t, err, ErrSignatureNotFound)
}
