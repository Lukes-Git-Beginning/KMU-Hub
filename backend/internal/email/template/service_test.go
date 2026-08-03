package template

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

var testTenantID = uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

// MockRepository implements Repository for testing.
type MockRepository struct {
	CreateFn      func(ctx context.Context, tpl *models.EmailTemplate) error
	GetByIDFn     func(ctx context.Context, id, tenantID, userID uuid.UUID, isAdmin bool) (*models.EmailTemplate, error)
	ListVisibleFn func(ctx context.Context, tenantID, userID uuid.UUID, isAdmin bool) ([]*models.EmailTemplate, error)
	UpdateFn      func(ctx context.Context, tpl *models.EmailTemplate) error
	DeleteFn      func(ctx context.Context, id, tenantID uuid.UUID) error
}

func (m *MockRepository) Create(ctx context.Context, tpl *models.EmailTemplate) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, tpl)
	}
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, id, tenantID, userID uuid.UUID, isAdmin bool) (*models.EmailTemplate, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id, tenantID, userID, isAdmin)
	}
	return nil, ErrTemplateNotFound
}

func (m *MockRepository) ListVisible(ctx context.Context, tenantID, userID uuid.UUID, isAdmin bool) ([]*models.EmailTemplate, error) {
	if m.ListVisibleFn != nil {
		return m.ListVisibleFn(ctx, tenantID, userID, isAdmin)
	}
	return nil, nil
}

func (m *MockRepository) Update(ctx context.Context, tpl *models.EmailTemplate) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, tpl)
	}
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id, tenantID)
	}
	return nil
}

func TestCreate_DefaultsToPersonalAndOwnsCaller(t *testing.T) {
	repo := &MockRepository{}
	svc := NewService(repo)
	ownerID := uuid.New()

	tpl, err := svc.Create(context.Background(), testTenantID, ownerID, "Angebot", "Betreff", "<p>Hi</p>", "Hi", "")
	require.NoError(t, err)
	assert.Equal(t, VisibilityPersonal, tpl.Visibility)
	require.NotNil(t, tpl.OwnerID)
	assert.Equal(t, ownerID, *tpl.OwnerID)
}

func TestCreate_SharedHasNoOwner(t *testing.T) {
	repo := &MockRepository{}
	svc := NewService(repo)

	tpl, err := svc.Create(context.Background(), testTenantID, uuid.New(), "Angebot", "Betreff", "<p>Hi</p>", "Hi", VisibilityShared)
	require.NoError(t, err)
	assert.Equal(t, VisibilityShared, tpl.Visibility)
	assert.Nil(t, tpl.OwnerID)
}

func TestCreate_NameRequired(t *testing.T) {
	repo := &MockRepository{}
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), testTenantID, uuid.New(), "   ", "", "", "", "")
	assert.ErrorIs(t, err, ErrNameRequired)
}

func TestCreate_InvalidVisibility(t *testing.T) {
	repo := &MockRepository{}
	svc := NewService(repo)

	_, err := svc.Create(context.Background(), testTenantID, uuid.New(), "Angebot", "", "", "", "public")
	assert.ErrorIs(t, err, ErrInvalidVisibility)
}

func TestUpdate_SwitchingToSharedClearsOwner(t *testing.T) {
	id := uuid.New()
	ownerID := uuid.New()
	existing := &models.EmailTemplate{ID: id, TenantID: testTenantID, OwnerID: &ownerID, Visibility: VisibilityPersonal, Name: "Alt"}
	repo := &MockRepository{
		GetByIDFn: func(_ context.Context, gotID, gotTenant, _ uuid.UUID, _ bool) (*models.EmailTemplate, error) {
			if gotID == id && gotTenant == testTenantID {
				return existing, nil
			}
			return nil, ErrTemplateNotFound
		},
	}
	svc := NewService(repo)

	shared := VisibilityShared
	updated, err := svc.Update(context.Background(), id, testTenantID, ownerID, false, nil, nil, nil, nil, &shared)
	require.NoError(t, err)
	assert.Equal(t, VisibilityShared, updated.Visibility)
	assert.Nil(t, updated.OwnerID)
}

func TestUpdate_SwitchingToPersonalOwnsCaller(t *testing.T) {
	id := uuid.New()
	existing := &models.EmailTemplate{ID: id, TenantID: testTenantID, OwnerID: nil, Visibility: VisibilityShared, Name: "Alt"}
	repo := &MockRepository{
		GetByIDFn: func(_ context.Context, gotID, gotTenant, _ uuid.UUID, _ bool) (*models.EmailTemplate, error) {
			return existing, nil
		},
	}
	svc := NewService(repo)
	caller := uuid.New()

	personal := VisibilityPersonal
	updated, err := svc.Update(context.Background(), id, testTenantID, caller, false, nil, nil, nil, nil, &personal)
	require.NoError(t, err)
	require.NotNil(t, updated.OwnerID)
	assert.Equal(t, caller, *updated.OwnerID)
}

func TestUpdate_ForeignPersonalTemplateNotFound(t *testing.T) {
	id := uuid.New()
	repo := &MockRepository{
		GetByIDFn: func(_ context.Context, _, _, _ uuid.UUID, _ bool) (*models.EmailTemplate, error) {
			// mirrors the real repository: a foreign personal template is
			// simply not found by the visibility-filtered query.
			return nil, ErrTemplateNotFound
		},
	}
	svc := NewService(repo)

	name := "Gekapert"
	_, err := svc.Update(context.Background(), id, testTenantID, uuid.New(), false, &name, nil, nil, nil, nil)
	assert.ErrorIs(t, err, ErrTemplateNotFound)
}

func TestDelete_ChecksVisibilityBeforeDeleting(t *testing.T) {
	id := uuid.New()
	deleteCalled := false
	repo := &MockRepository{
		GetByIDFn: func(_ context.Context, _, _, _ uuid.UUID, _ bool) (*models.EmailTemplate, error) {
			return nil, ErrTemplateNotFound
		},
		DeleteFn: func(_ context.Context, _, _ uuid.UUID) error {
			deleteCalled = true
			return nil
		},
	}
	svc := NewService(repo)

	err := svc.Delete(context.Background(), id, testTenantID, uuid.New(), false)
	assert.ErrorIs(t, err, ErrTemplateNotFound)
	assert.False(t, deleteCalled, "Delete must not reach the repository once GetByID refuses visibility")
}

func TestRender_SubstitutesOnlyAllowedPlaceholders(t *testing.T) {
	id := uuid.New()
	tpl := &models.EmailTemplate{
		ID: id, TenantID: testTenantID,
		Subject:  "Hallo {{contact_first_name}}, von {{sender_name}}",
		BodyHTML: "<p>{{contact_first_name}} {{contact_last_name}}, {{unknown_key}}</p>",
		BodyText: "{{contact_first_name}} {{unknown_key}}",
	}
	repo := &MockRepository{
		GetByIDFn: func(_ context.Context, _, _, _ uuid.UUID, _ bool) (*models.EmailTemplate, error) {
			return tpl, nil
		},
	}
	svc := NewService(repo)

	subject, bodyHTML, bodyText, err := svc.Render(context.Background(), id, testTenantID, uuid.New(), false, map[string]string{
		"contact_first_name": "Anna",
		"sender_name":        "Team",
		"unknown_key":        "<script>alert(1)</script>",
	})
	require.NoError(t, err)
	assert.Equal(t, "Hallo Anna, von Team", subject)
	// contact_last_name was never supplied, so its token stays literal.
	// unknown_key is not in AllowedPlaceholders, so it is never substituted --
	// the literal token survives untouched rather than reflecting caller input.
	assert.Equal(t, "<p>Anna {{contact_last_name}}, {{unknown_key}}</p>", bodyHTML)
	assert.Equal(t, "Anna {{unknown_key}}", bodyText)
}

func TestRender_NotFoundPropagates(t *testing.T) {
	repo := &MockRepository{
		GetByIDFn: func(_ context.Context, _, _, _ uuid.UUID, _ bool) (*models.EmailTemplate, error) {
			return nil, ErrTemplateNotFound
		},
	}
	svc := NewService(repo)

	_, _, _, err := svc.Render(context.Background(), uuid.New(), testTenantID, uuid.New(), false, nil)
	assert.ErrorIs(t, err, ErrTemplateNotFound)
}
