package settings_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/settings"
)

func TestGetBranding_DefaultsBeforeFirstWrite(t *testing.T) {
	repo := newFakeRepo()
	svc := newService(repo)

	b, err := svc.GetBranding(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, "#10B981", b.AccentColor, "default accent color must match the frontend mock's prior default")
	assert.Empty(t, b.Name)
	assert.Empty(t, b.LogoObjectKey)
	assert.Empty(t, b.IconObjectKey)
}

func TestPutBranding_AdminMayWrite_RoundTrips(t *testing.T) {
	adminID := uuid.New()
	tenantID := uuid.New()
	repo := newFakeRepo()
	svc := newService(repo, adminID)

	logoKey := tenantID.String() + "/branding/logo.png"
	updated, err := svc.PutBranding(context.Background(), tenantID, adminID, &settings.Branding{
		Name:          "Zentria",
		LogoObjectKey: logoKey,
		AccentColor:   "#3B82F6",
	})
	require.NoError(t, err)
	assert.Equal(t, "Zentria", updated.Name)
	assert.Equal(t, logoKey, updated.LogoObjectKey)
	assert.Equal(t, "#3B82F6", updated.AccentColor)
	assert.Empty(t, updated.IconObjectKey)

	// Round-trip through Get.
	got, err := svc.GetBranding(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Equal(t, updated, got)
}

func TestPutBranding_FullReplaceClearsPreviousLogo(t *testing.T) {
	adminID := uuid.New()
	tenantID := uuid.New()
	repo := newFakeRepo()
	svc := newService(repo, adminID)

	logoKey := tenantID.String() + "/branding/logo.png"
	_, err := svc.PutBranding(context.Background(), tenantID, adminID, &settings.Branding{
		Name: "Zentria", LogoObjectKey: logoKey, AccentColor: "#3B82F6",
	})
	require.NoError(t, err)

	// Second write omits the logo — must clear it, not keep the old value
	// (this is a full replace, not a merge patch).
	updated, err := svc.PutBranding(context.Background(), tenantID, adminID, &settings.Branding{
		Name: "Zentria", AccentColor: "#3B82F6",
	})
	require.NoError(t, err)
	assert.Empty(t, updated.LogoObjectKey, "omitting logo_object_key on a PUT must clear it")
}

func TestPutBranding_NonAdminNonLeadRejected(t *testing.T) {
	regularUser := uuid.New()
	tenantID := uuid.New()
	repo := newFakeRepo()
	svc := newService(repo) // no admin

	_, err := svc.PutBranding(context.Background(), tenantID, regularUser, &settings.Branding{
		Name: "Evil Corp", AccentColor: "#3B82F6",
	})
	assert.ErrorIs(t, err, settings.ErrNotModuleLead)
}

func TestPutBranding_RejectsInvalidAccentColor(t *testing.T) {
	adminID := uuid.New()
	tenantID := uuid.New()
	repo := newFakeRepo()
	svc := newService(repo, adminID)

	_, err := svc.PutBranding(context.Background(), tenantID, adminID, &settings.Branding{
		Name: "Zentria", AccentColor: "#FF00FF", // not in the swatch palette
	})
	assert.ErrorIs(t, err, settings.ErrInvalidAccentColor)
}

func TestPutBranding_RejectsMissingAccentColor(t *testing.T) {
	adminID := uuid.New()
	tenantID := uuid.New()
	repo := newFakeRepo()
	svc := newService(repo, adminID)

	_, err := svc.PutBranding(context.Background(), tenantID, adminID, &settings.Branding{Name: "Zentria"})
	assert.ErrorIs(t, err, settings.ErrInvalidAccentColor)
}

func TestPutBranding_RejectsNameTooLong(t *testing.T) {
	adminID := uuid.New()
	tenantID := uuid.New()
	repo := newFakeRepo()
	svc := newService(repo, adminID)

	longName := make([]byte, 201)
	for i := range longName {
		longName[i] = 'a'
	}

	_, err := svc.PutBranding(context.Background(), tenantID, adminID, &settings.Branding{
		Name: string(longName), AccentColor: "#3B82F6",
	})
	assert.ErrorIs(t, err, settings.ErrBrandingNameTooLong)
}

func TestPutBranding_RejectsForeignObjectKey(t *testing.T) {
	adminID := uuid.New()
	tenantID := uuid.New()
	otherTenant := uuid.New()
	repo := newFakeRepo()
	svc := newService(repo, adminID)

	_, err := svc.PutBranding(context.Background(), tenantID, adminID, &settings.Branding{
		Name:          "Zentria",
		LogoObjectKey: otherTenant.String() + "/branding/logo.png",
		AccentColor:   "#3B82F6",
	})
	assert.ErrorIs(t, err, settings.ErrInvalidBrandingObjectKey)
}

func TestPutBranding_RejectsObjectKeyFromDifferentScope(t *testing.T) {
	adminID := uuid.New()
	tenantID := uuid.New()
	repo := newFakeRepo()
	svc := newService(repo, adminID)

	// Own tenant, but the "avatar" presign scope — not "branding".
	_, err := svc.PutBranding(context.Background(), tenantID, adminID, &settings.Branding{
		Name:          "Zentria",
		IconObjectKey: tenantID.String() + "/avatar/pic.png",
		AccentColor:   "#3B82F6",
	})
	assert.ErrorIs(t, err, settings.ErrInvalidBrandingObjectKey)
}

func TestPutBranding_EmptyObjectKeysAllowed(t *testing.T) {
	adminID := uuid.New()
	tenantID := uuid.New()
	repo := newFakeRepo()
	svc := newService(repo, adminID)

	updated, err := svc.PutBranding(context.Background(), tenantID, adminID, &settings.Branding{
		Name: "Zentria", AccentColor: "#3B82F6",
	})
	require.NoError(t, err)
	assert.Empty(t, updated.LogoObjectKey)
	assert.Empty(t, updated.IconObjectKey)
}

func TestPutBranding_ModuleLeadMayWrite(t *testing.T) {
	leadID := uuid.New()
	tenantID := uuid.New()
	repo := newFakeRepo()
	svc := newService(repo) // no admin

	_, err := repo.GrantModuleLead(context.Background(), tenantID, leadID, "branding", nil)
	require.NoError(t, err)

	_, err = svc.PutBranding(context.Background(), tenantID, leadID, &settings.Branding{
		Name: "Zentria", AccentColor: "#3B82F6",
	})
	require.NoError(t, err)
}

func TestBranding_CrossTenantIsolation(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	adminA := uuid.New()
	repo := newFakeRepo()
	svc := newService(repo, adminA)

	_, err := svc.PutBranding(context.Background(), tenantA, adminA, &settings.Branding{
		Name: "Tenant A Inc", AccentColor: "#3B82F6",
	})
	require.NoError(t, err)

	b, err := svc.GetBranding(context.Background(), tenantB)
	require.NoError(t, err)
	assert.Empty(t, b.Name, "tenant B must not see tenant A's branding")
	assert.Equal(t, "#10B981", b.AccentColor, "tenant B falls back to the default")
}
