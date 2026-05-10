package project

// Cross-tenant isolation tests for user_project_preferences (Migration 000112).
//
// user_project_preferences.tenant_id is NOT NULL after migration. SetUserPreference
// now stamps tenant_id on every INSERT/UPDATE. These tests verify that preferences
// for different tenants carry the correct tenant ID and are stored independently.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// TestSetUserPreference_TenantID_Stamped verifies that a preference set for
// tenant A carries tenant A's ID after persistence.
func TestSetUserPreference_TenantID_Stamped(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	userID := uuid.New()
	projectID := uuid.New()

	pref := &models.UserProjectPreference{
		TenantID:  testTenantID,
		UserID:    userID,
		ProjectID: projectID,
		ViewType:  "list",
	}

	err := svc.SetUserPreference(context.Background(), pref)
	require.NoError(t, err)

	// Retrieve and verify tenant ID is persisted correctly.
	key := userID.String() + ":" + projectID.String()
	stored, ok := repo.preferences[key]
	require.True(t, ok, "preference must be stored in the mock repository")
	assert.Equal(t, testTenantID, stored.TenantID, "stored preference must carry tenant A's ID")
}

// TestSetUserPreference_TenantIsolation_DifferentPreferences verifies that
// preferences for two different tenants carry their respective tenant IDs.
func TestSetUserPreference_TenantIsolation_DifferentPreferences(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	userA := uuid.New()
	projectA := uuid.New()
	userB := uuid.New()
	projectB := uuid.New()

	ctx := context.Background()

	err := svc.SetUserPreference(ctx, &models.UserProjectPreference{
		TenantID:  testTenantID,
		UserID:    userA,
		ProjectID: projectA,
		ViewType:  "kanban",
	})
	require.NoError(t, err)

	err = svc.SetUserPreference(ctx, &models.UserProjectPreference{
		TenantID:  otherTenantID,
		UserID:    userB,
		ProjectID: projectB,
		ViewType:  "list",
	})
	require.NoError(t, err)

	keyA := userA.String() + ":" + projectA.String()
	keyB := userB.String() + ":" + projectB.String()

	prefA, ok := repo.preferences[keyA]
	require.True(t, ok)
	prefB, ok := repo.preferences[keyB]
	require.True(t, ok)

	assert.Equal(t, testTenantID, prefA.TenantID)
	assert.Equal(t, otherTenantID, prefB.TenantID)
	assert.NotEqual(t, prefA.TenantID, prefB.TenantID,
		"preferences for different tenants must not share tenant_id")
}

// TestSetUserPreference_InvalidViewType_NoTenantLeak verifies that validation
// failure on view type does not cause any preference to be persisted (for either
// tenant), preventing partial writes.
func TestSetUserPreference_InvalidViewType_NoTenantLeak(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	err := svc.SetUserPreference(context.Background(), &models.UserProjectPreference{
		TenantID:  testTenantID,
		UserID:    uuid.New(),
		ProjectID: uuid.New(),
		ViewType:  "invalid_view",
	})

	assert.ErrorIs(t, err, ErrInvalidViewType)
	assert.Empty(t, repo.preferences, "no preference must be persisted when validation fails")
}
