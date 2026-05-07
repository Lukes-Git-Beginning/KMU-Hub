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

var testTenantID = uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

// MockRepository implements Repository for testing.
type MockRepository struct {
	CreateFn              func(ctx context.Context, sig *models.EmailSignature) error
	GetByIDFn             func(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.EmailSignature, error)
	GetDefaultFn          func(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) (*models.EmailSignature, error)
	ListByUserFn          func(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) ([]*models.EmailSignature, error)
	UpdateFn              func(ctx context.Context, sig *models.EmailSignature) error
	DeleteFn              func(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error
	ClearDefaultForUserFn func(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) error
	CountByUserFn         func(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) (int, error)
}

func (m *MockRepository) Create(ctx context.Context, sig *models.EmailSignature) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, sig)
	}
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.EmailSignature, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id, tenantID)
	}
	return nil, ErrSignatureNotFound
}

func (m *MockRepository) GetDefault(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) (*models.EmailSignature, error) {
	if m.GetDefaultFn != nil {
		return m.GetDefaultFn(ctx, userID, tenantID)
	}
	return nil, ErrSignatureNotFound
}

func (m *MockRepository) ListByUser(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) ([]*models.EmailSignature, error) {
	if m.ListByUserFn != nil {
		return m.ListByUserFn(ctx, userID, tenantID)
	}
	return nil, nil
}

func (m *MockRepository) Update(ctx context.Context, sig *models.EmailSignature) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, sig)
	}
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id, tenantID)
	}
	return nil
}

func (m *MockRepository) ClearDefaultForUser(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) error {
	if m.ClearDefaultForUserFn != nil {
		return m.ClearDefaultForUserFn(ctx, userID, tenantID)
	}
	return nil
}

func (m *MockRepository) CountByUser(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) (int, error) {
	if m.CountByUserFn != nil {
		return m.CountByUserFn(ctx, userID, tenantID)
	}
	return 0, nil
}

func TestCreate_Success(t *testing.T) {
	repo := &MockRepository{
		CountByUserFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (int, error) {
			return 1, nil
		},
	}
	svc := NewService(repo)

	sig, err := svc.Create(context.Background(), uuid.New(), testTenantID, "Work Sig", "<p>Regards</p>")
	require.NoError(t, err)
	require.NotNil(t, sig)
	assert.Equal(t, "Work Sig", sig.Name)
	assert.Equal(t, "<p>Regards</p>", sig.HTMLContent)
	assert.Equal(t, testTenantID, sig.TenantID)
	assert.False(t, sig.IsDefault)
}

func TestCreate_FirstSignatureAutoDefault(t *testing.T) {
	repo := &MockRepository{
		CountByUserFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (int, error) {
			return 0, nil
		},
	}
	svc := NewService(repo)

	sig, err := svc.Create(context.Background(), uuid.New(), testTenantID, "First Sig", "<p>Hi</p>")
	require.NoError(t, err)
	assert.True(t, sig.IsDefault)
}

func TestCreate_SubsequentNotDefault(t *testing.T) {
	repo := &MockRepository{
		CountByUserFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (int, error) {
			return 1, nil
		},
	}
	svc := NewService(repo)

	sig, err := svc.Create(context.Background(), uuid.New(), testTenantID, "Second Sig", "<p>Bye</p>")
	require.NoError(t, err)
	assert.False(t, sig.IsDefault)
}

func TestCreate_RepoError(t *testing.T) {
	repoErr := errors.New("db failure")
	repo := &MockRepository{
		CountByUserFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (int, error) {
			return 0, nil
		},
		CreateFn: func(_ context.Context, _ *models.EmailSignature) error {
			return repoErr
		},
	}
	svc := NewService(repo)

	sig, err := svc.Create(context.Background(), uuid.New(), testTenantID, "Fail", "<p>x</p>")
	assert.Nil(t, sig)
	assert.Error(t, err)
	assert.ErrorIs(t, err, repoErr)
}

func TestGetByID_Success(t *testing.T) {
	sigID := uuid.New()
	expected := &models.EmailSignature{
		ID:       sigID,
		TenantID: testTenantID,
		Name:     "My Sig",
	}
	repo := &MockRepository{
		GetByIDFn: func(_ context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.EmailSignature, error) {
			if id == sigID && tenantID == testTenantID {
				return expected, nil
			}
			return nil, ErrSignatureNotFound
		},
	}
	svc := NewService(repo)

	sig, err := svc.GetByID(context.Background(), sigID, testTenantID)
	require.NoError(t, err)
	assert.Equal(t, "My Sig", sig.Name)
}

func TestGetByID_NotFound(t *testing.T) {
	repo := &MockRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*models.EmailSignature, error) {
			return nil, ErrSignatureNotFound
		},
	}
	svc := NewService(repo)

	sig, err := svc.GetByID(context.Background(), uuid.New(), testTenantID)
	assert.Nil(t, sig)
	assert.ErrorIs(t, err, ErrSignatureNotFound)
}

func TestGetDefault_Success(t *testing.T) {
	userID := uuid.New()
	expected := &models.EmailSignature{
		ID:        uuid.New(),
		TenantID:  testTenantID,
		UserID:    userID,
		IsDefault: true,
	}
	repo := &MockRepository{
		GetDefaultFn: func(_ context.Context, uid uuid.UUID, tenantID uuid.UUID) (*models.EmailSignature, error) {
			if uid == userID && tenantID == testTenantID {
				return expected, nil
			}
			return nil, ErrSignatureNotFound
		},
	}
	svc := NewService(repo)

	sig, err := svc.GetDefault(context.Background(), userID, testTenantID)
	require.NoError(t, err)
	assert.True(t, sig.IsDefault)
}

func TestListByUser_Success(t *testing.T) {
	userID := uuid.New()
	sigs := []*models.EmailSignature{
		{ID: uuid.New(), TenantID: testTenantID, UserID: userID, Name: "Sig A"},
		{ID: uuid.New(), TenantID: testTenantID, UserID: userID, Name: "Sig B"},
	}
	repo := &MockRepository{
		ListByUserFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) ([]*models.EmailSignature, error) {
			return sigs, nil
		},
	}
	svc := NewService(repo)

	result, err := svc.ListByUser(context.Background(), userID, testTenantID)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestUpdate_Success(t *testing.T) {
	sigID := uuid.New()
	existing := &models.EmailSignature{
		ID:          sigID,
		TenantID:    testTenantID,
		UserID:      uuid.New(),
		Name:        "Old Name",
		HTMLContent: "<p>old</p>",
		UpdatedAt:   time.Now().Add(-1 * time.Hour),
	}
	repo := &MockRepository{
		GetByIDFn: func(_ context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.EmailSignature, error) {
			if id == sigID && tenantID == testTenantID {
				return existing, nil
			}
			return nil, ErrSignatureNotFound
		},
	}
	svc := NewService(repo)

	updated, err := svc.Update(context.Background(), sigID, testTenantID, "New Name", "<p>new</p>")
	require.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
	assert.Equal(t, "<p>new</p>", updated.HTMLContent)
}

func TestDelete_Success(t *testing.T) {
	called := false
	repo := &MockRepository{
		DeleteFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
			called = true
			return nil
		},
	}
	svc := NewService(repo)

	err := svc.Delete(context.Background(), uuid.New(), testTenantID)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestSetDefault_Success(t *testing.T) {
	userID := uuid.New()
	sigID := uuid.New()
	existing := &models.EmailSignature{
		ID:        sigID,
		TenantID:  testTenantID,
		UserID:    userID,
		IsDefault: false,
	}
	clearCalled := false
	repo := &MockRepository{
		GetByIDFn: func(_ context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.EmailSignature, error) {
			if id == sigID && tenantID == testTenantID {
				return existing, nil
			}
			return nil, ErrSignatureNotFound
		},
		ClearDefaultForUserFn: func(_ context.Context, uid uuid.UUID, tenantID uuid.UUID) error {
			assert.Equal(t, userID, uid)
			assert.Equal(t, testTenantID, tenantID)
			clearCalled = true
			return nil
		},
		UpdateFn: func(_ context.Context, sig *models.EmailSignature) error {
			assert.True(t, sig.IsDefault)
			return nil
		},
	}
	svc := NewService(repo)

	err := svc.SetDefault(context.Background(), sigID, testTenantID)
	require.NoError(t, err)
	assert.True(t, clearCalled)
}

func TestSetDefault_NotFound(t *testing.T) {
	repo := &MockRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*models.EmailSignature, error) {
			return nil, ErrSignatureNotFound
		},
	}
	svc := NewService(repo)

	err := svc.SetDefault(context.Background(), uuid.New(), testTenantID)
	assert.ErrorIs(t, err, ErrSignatureNotFound)
}

func TestSignatureTenantIsolation(t *testing.T) {
	tenantA := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	tenantB := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	sigID := uuid.New()
	sigA := &models.EmailSignature{
		ID:       sigID,
		TenantID: tenantA,
		Name:     "Tenant A Sig",
	}
	repo := &MockRepository{
		GetByIDFn: func(_ context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.EmailSignature, error) {
			if id == sigID && tenantID == tenantA {
				return sigA, nil
			}
			return nil, ErrSignatureNotFound
		},
	}
	svc := NewService(repo)

	// Tenant A can read their signature
	sig, err := svc.GetByID(context.Background(), sigID, tenantA)
	require.NoError(t, err)
	assert.Equal(t, "Tenant A Sig", sig.Name)

	// Tenant B cannot access tenant A's signature
	_, err = svc.GetByID(context.Background(), sigID, tenantB)
	assert.ErrorIs(t, err, ErrSignatureNotFound)
}
