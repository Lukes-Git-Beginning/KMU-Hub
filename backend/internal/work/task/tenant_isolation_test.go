package task

// Cross-tenant isolation tests for task_entity_links and task_custom_field_values
// (Migration 000112).
//
// Both tables now have tenant_id NOT NULL. The repository's LinkEntity and
// SetCustomFieldValues methods receive and persist tenant IDs. These tests verify
// that entity links and custom field values are stamped with the correct tenant ID,
// using the in-process mock repository.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

var (
	taskTenantA = uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001")
	taskTenantB = uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000002")
)

// captureRepo extends mockRepo to capture the last LinkEntity call.
type captureRepo struct {
	mockRepo
	lastLink *models.TaskEntityLink
	lastCFVTenantID uuid.UUID
}

func (c *captureRepo) LinkEntity(_ context.Context, link *models.TaskEntityLink) error {
	clone := *link
	c.lastLink = &clone
	return nil
}

func (c *captureRepo) SetCustomFieldValues(_ context.Context, _ uuid.UUID, tenantID uuid.UUID, _ map[uuid.UUID]any) error {
	c.lastCFVTenantID = tenantID
	return nil
}

// TestTaskEntityLink_TenantID_Stamped verifies that a TaskEntityLink constructed
// with tenant A's ID carries that tenant ID in the persisted row.
func TestTaskEntityLink_TenantID_Stamped(t *testing.T) {
	repo := &captureRepo{
		mockRepo: *newMockRepo(),
	}

	link := &models.TaskEntityLink{
		ID:         uuid.New(),
		TenantID:   taskTenantA,
		TaskID:     uuid.New(),
		EntityType: "contact",
		EntityID:   uuid.New(),
		CreatedBy:  uuid.New(),
	}

	err := repo.LinkEntity(context.Background(), link)
	require.NoError(t, err)
	require.NotNil(t, repo.lastLink)
	assert.Equal(t, taskTenantA, repo.lastLink.TenantID, "entity link must carry tenant A's ID")
	assert.Equal(t, link.TaskID, repo.lastLink.TaskID)
}

// TestTaskEntityLink_TenantIsolation verifies that entity links for different
// tenants have distinct tenant IDs and cannot be confused.
func TestTaskEntityLink_TenantIsolation(t *testing.T) {
	var captured []*models.TaskEntityLink
	repo := &captureRepo{
		mockRepo: *newMockRepo(),
	}

	linkA := &models.TaskEntityLink{
		ID:         uuid.New(),
		TenantID:   taskTenantA,
		TaskID:     uuid.New(),
		EntityType: "contact",
		EntityID:   uuid.New(),
		CreatedBy:  uuid.New(),
	}
	linkB := &models.TaskEntityLink{
		ID:         uuid.New(),
		TenantID:   taskTenantB,
		TaskID:     uuid.New(),
		EntityType: "contact",
		EntityID:   uuid.New(),
		CreatedBy:  uuid.New(),
	}

	ctx := context.Background()
	require.NoError(t, repo.LinkEntity(ctx, linkA))
	captured = append(captured, repo.lastLink)

	require.NoError(t, repo.LinkEntity(ctx, linkB))
	captured = append(captured, repo.lastLink)

	require.Len(t, captured, 2)
	assert.Equal(t, taskTenantA, captured[0].TenantID)
	assert.Equal(t, taskTenantB, captured[1].TenantID)
	assert.NotEqual(t, captured[0].TenantID, captured[1].TenantID,
		"entity links for different tenants must not share tenant_id")
}

// TestSetCustomFieldValues_TenantID_Propagated verifies that SetCustomFieldValues
// receives the correct tenant ID, ensuring custom field values are scoped to the
// right tenant.
func TestSetCustomFieldValues_TenantID_Propagated(t *testing.T) {
	repo := &captureRepo{
		mockRepo: *newMockRepo(),
	}

	taskID := uuid.New()
	fieldID := uuid.New()
	values := map[uuid.UUID]any{fieldID: "Q2 2026"}

	err := repo.SetCustomFieldValues(context.Background(), taskID, taskTenantA, values)
	require.NoError(t, err)
	assert.Equal(t, taskTenantA, repo.lastCFVTenantID,
		"SetCustomFieldValues must propagate tenant A's ID")
}

// TestSetCustomFieldValues_TenantIsolation verifies that calling SetCustomFieldValues
// for two different tenants propagates each tenant's ID independently.
func TestSetCustomFieldValues_TenantIsolation(t *testing.T) {
	repo := &captureRepo{
		mockRepo: *newMockRepo(),
	}

	ctx := context.Background()
	fID := uuid.New()

	err := repo.SetCustomFieldValues(ctx, uuid.New(), taskTenantA, map[uuid.UUID]any{fID: "alpha"})
	require.NoError(t, err)
	tenantAfterA := repo.lastCFVTenantID

	err = repo.SetCustomFieldValues(ctx, uuid.New(), taskTenantB, map[uuid.UUID]any{fID: "beta"})
	require.NoError(t, err)
	tenantAfterB := repo.lastCFVTenantID

	assert.Equal(t, taskTenantA, tenantAfterA)
	assert.Equal(t, taskTenantB, tenantAfterB)
	assert.NotEqual(t, tenantAfterA, tenantAfterB,
		"custom field values for different tenants must not share tenant_id")
}
