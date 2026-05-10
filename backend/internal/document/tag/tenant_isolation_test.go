package tag

// Cross-tenant isolation tests for document_tags (Migration 000114).
//
// document_tags.tenant_id is NOT NULL after migration. The service's CreateTag
// method now accepts a tenantID and stamps it on every new tag. These tests
// verify that tags created for tenant A are distinct from tags created for
// tenant B, using the in-process mock repository.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

var (
	tenantA = uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001")
	tenantB = uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000002")
)

// TestCreateTag_TenantID_Stamped verifies that a tag created for tenant A carries
// tenant A's ID and not tenant B's.
func TestCreateTag_TenantID_Stamped(t *testing.T) {
	var captured *models.DocumentTag
	repo := &MockRepository{
		CreateFn: func(_ context.Context, tag *models.DocumentTag) error {
			captured = tag
			return nil
		},
	}
	svc := NewService(repo)

	tag, err := svc.CreateTag(context.Background(), "Finance", "", uuid.New(), tenantA)

	require.NoError(t, err)
	require.NotNil(t, tag)
	assert.Equal(t, tenantA, tag.TenantID, "tag must carry tenant A's ID")
	assert.Equal(t, tenantA, captured.TenantID, "repo must receive tag with tenant A's ID")
}

// TestCreateTag_TenantIsolation_DifferentTags verifies that two tags created for
// different tenants carry their respective tenant IDs and cannot be confused.
func TestCreateTag_TenantIsolation_DifferentTags(t *testing.T) {
	var createdTags []*models.DocumentTag
	repo := &MockRepository{
		CreateFn: func(_ context.Context, tag *models.DocumentTag) error {
			clone := *tag
			createdTags = append(createdTags, &clone)
			return nil
		},
	}
	svc := NewService(repo)

	ctx := context.Background()
	tagA, err := svc.CreateTag(ctx, "Invoices", "", uuid.New(), tenantA)
	require.NoError(t, err)

	tagB, err := svc.CreateTag(ctx, "Invoices", "", uuid.New(), tenantB)
	require.NoError(t, err)

	assert.Equal(t, tenantA, tagA.TenantID)
	assert.Equal(t, tenantB, tagB.TenantID)
	assert.NotEqual(t, tagA.TenantID, tagB.TenantID, "tags for different tenants must not share tenant_id")
	assert.NotEqual(t, tagA.ID, tagB.ID, "tags must have distinct IDs")

	require.Len(t, createdTags, 2)
	assert.Equal(t, tenantA, createdTags[0].TenantID)
	assert.Equal(t, tenantB, createdTags[1].TenantID)
}

// TestCreateTag_TenantID_NotZero verifies that a tag is rejected early (name empty)
// before the tenant ID matters, ensuring validation runs before persistence.
func TestCreateTag_TenantID_EmptyNameRejectedBeforePersist(t *testing.T) {
	called := false
	repo := &MockRepository{
		CreateFn: func(_ context.Context, _ *models.DocumentTag) error {
			called = true
			return nil
		},
	}
	svc := NewService(repo)

	_, err := svc.CreateTag(context.Background(), "", "", uuid.New(), tenantA)

	assert.ErrorIs(t, err, ErrTagNameRequired)
	assert.False(t, called, "repo.Create must not be called when validation fails")
}
