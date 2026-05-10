package share

// Cross-tenant isolation tests for document_shares (Migration 000114).
//
// document_shares.tenant_id is NOT NULL after migration. ShareInput now requires
// a TenantID, which is stamped onto every new DocumentShare. These tests verify
// that shares created for tenant A carry tenant A's ID, that tenant B shares
// carry tenant B's ID, and that the service-layer validation fires before any
// persistence attempt.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

var (
	isolTenantA = uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001")
	isolTenantB = uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000002")
)

// TestShareEntity_TenantID_Stamped verifies that a share created with tenant A's
// ID carries that tenant ID in the persisted DocumentShare.
func TestShareEntity_TenantID_Stamped(t *testing.T) {
	var captured *models.DocumentShare
	repo := &MockRepository{
		GetUserPermissionFn: func(_ context.Context, _ string, _ uuid.UUID, _ uuid.UUID) (*models.DocumentShare, error) {
			return nil, ErrShareNotFound
		},
		CreateFn: func(_ context.Context, share *models.DocumentShare) error {
			captured = share
			return nil
		},
	}
	svc := NewService(repo)

	sharedBy := uuid.New()
	sharedWith := uuid.New()

	result, err := svc.ShareEntity(context.Background(), ShareInput{
		TenantID:         isolTenantA,
		EntityType:       models.ShareEntityFile,
		EntityID:         uuid.New(),
		SharedWithUserID: sharedWith,
		Permission:       models.SharePermissionRead,
		SharedBy:         sharedBy,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, isolTenantA, result.TenantID, "share must carry tenant A's ID")
	require.NotNil(t, captured)
	assert.Equal(t, isolTenantA, captured.TenantID, "repo must receive share with tenant A's ID")
}

// TestShareEntity_TenantIsolation_DifferentShares verifies that shares for two
// different tenants carry their respective tenant IDs and are not confused.
func TestShareEntity_TenantIsolation_DifferentShares(t *testing.T) {
	var created []*models.DocumentShare
	repo := &MockRepository{
		GetUserPermissionFn: func(_ context.Context, _ string, _ uuid.UUID, _ uuid.UUID) (*models.DocumentShare, error) {
			return nil, ErrShareNotFound
		},
		CreateFn: func(_ context.Context, share *models.DocumentShare) error {
			clone := *share
			created = append(created, &clone)
			return nil
		},
	}
	svc := NewService(repo)
	ctx := context.Background()

	fileID := uuid.New()
	userA := uuid.New()
	userB := uuid.New()

	shareA, err := svc.ShareEntity(ctx, ShareInput{
		TenantID:         isolTenantA,
		EntityType:       models.ShareEntityFile,
		EntityID:         fileID,
		SharedWithUserID: userA,
		Permission:       models.SharePermissionRead,
		SharedBy:         uuid.New(),
	})
	require.NoError(t, err)

	shareB, err := svc.ShareEntity(ctx, ShareInput{
		TenantID:         isolTenantB,
		EntityType:       models.ShareEntityFile,
		EntityID:         fileID,
		SharedWithUserID: userB,
		Permission:       models.SharePermissionWrite,
		SharedBy:         uuid.New(),
	})
	require.NoError(t, err)

	assert.Equal(t, isolTenantA, shareA.TenantID)
	assert.Equal(t, isolTenantB, shareB.TenantID)
	assert.NotEqual(t, shareA.TenantID, shareB.TenantID, "shares for different tenants must not share tenant_id")
	assert.NotEqual(t, shareA.ID, shareB.ID, "shares must have distinct IDs")

	require.Len(t, created, 2)
	assert.Equal(t, isolTenantA, created[0].TenantID)
	assert.Equal(t, isolTenantB, created[1].TenantID)
}

// TestShareEntity_SelfShare_RejectsBeforePersist verifies that the self-share check
// fires before repo.Create so no invalid share is ever persisted.
func TestShareEntity_SelfShare_RejectsBeforePersist(t *testing.T) {
	called := false
	repo := &MockRepository{
		GetUserPermissionFn: func(_ context.Context, _ string, _ uuid.UUID, _ uuid.UUID) (*models.DocumentShare, error) {
			return nil, ErrShareNotFound
		},
		CreateFn: func(_ context.Context, _ *models.DocumentShare) error {
			called = true
			return nil
		},
	}
	svc := NewService(repo)

	actorID := uuid.New()
	_, err := svc.ShareEntity(context.Background(), ShareInput{
		TenantID:         isolTenantA,
		EntityType:       models.ShareEntityFile,
		EntityID:         uuid.New(),
		SharedWithUserID: actorID,
		Permission:       models.SharePermissionRead,
		SharedBy:         actorID, // same — must be rejected
	})

	assert.ErrorIs(t, err, ErrCannotShareWithSelf)
	assert.False(t, called, "repo.Create must not be called on self-share attempt")
}
