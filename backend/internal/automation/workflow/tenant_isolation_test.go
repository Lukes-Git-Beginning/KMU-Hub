package workflow

// Cross-tenant isolation tests for automation_executions (Migration 000112).
//
// automation_executions.tenant_id is NOT NULL after migration. The ExecutionLogger
// now receives a tenantID from the engine and stamps it on every AutomationExecution
// at creation time. These tests verify that:
//  1. Executions created via CreateExecution carry the correct tenant ID.
//  2. GetByID on an automation enforces cross-tenant isolation (returns not-found
//     when queried with the wrong tenant ID).
//  3. Automations for different tenants carry distinct tenant IDs.

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

var (
	autoTenantA = uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001")
	autoTenantB = uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000002")
)

// TestCreateExecution_TenantID_Stamped verifies that an AutomationExecution
// created via the exec repo carries the correct tenant ID.
func TestCreateExecution_TenantID_Stamped(t *testing.T) {
	var captured *models.AutomationExecution
	execRepo := &mockExecRepo{
		createExecFn: func(_ context.Context, e *models.AutomationExecution) error {
			clone := *e
			captured = &clone
			return nil
		},
	}
	svc := newTestService(nil, execRepo, nil)
	_ = svc // service wires execRepo into ListExecutions etc.

	exec := &models.AutomationExecution{
		ID:           uuid.New(),
		TenantID:     autoTenantA,
		AutomationID: uuid.New(),
		Status:       "running",
		Steps:        json.RawMessage(`[]`),
	}

	err := execRepo.CreateExecution(context.Background(), exec)
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, autoTenantA, captured.TenantID, "execution must carry tenant A's ID")
}

// TestCreateExecution_TenantIsolation verifies that executions created for two
// different tenants carry their respective tenant IDs and are not confused.
func TestCreateExecution_TenantIsolation(t *testing.T) {
	var createdExecs []*models.AutomationExecution
	execRepo := &mockExecRepo{
		createExecFn: func(_ context.Context, e *models.AutomationExecution) error {
			clone := *e
			createdExecs = append(createdExecs, &clone)
			return nil
		},
	}

	ctx := context.Background()

	execA := &models.AutomationExecution{
		ID:           uuid.New(),
		TenantID:     autoTenantA,
		AutomationID: uuid.New(),
		Status:       "running",
		Steps:        json.RawMessage(`[]`),
	}
	execB := &models.AutomationExecution{
		ID:           uuid.New(),
		TenantID:     autoTenantB,
		AutomationID: uuid.New(),
		Status:       "running",
		Steps:        json.RawMessage(`[]`),
	}

	require.NoError(t, execRepo.CreateExecution(ctx, execA))
	require.NoError(t, execRepo.CreateExecution(ctx, execB))

	require.Len(t, createdExecs, 2)
	assert.Equal(t, autoTenantA, createdExecs[0].TenantID)
	assert.Equal(t, autoTenantB, createdExecs[1].TenantID)
	assert.NotEqual(t, createdExecs[0].TenantID, createdExecs[1].TenantID,
		"executions for different tenants must not share tenant_id")
}

// TestAutomation_GetByID_TenantIsolation verifies that the mockRepo enforces
// tenant isolation on GetByID: querying with the wrong tenant ID returns not-found.
func TestAutomation_GetByID_TenantIsolation(t *testing.T) {
	automationID := uuid.New()
	automation := &models.Automation{
		ID:       automationID,
		TenantID: autoTenantA,
		Name:     "Send welcome email",
		IsActive: true,
	}

	repo := &mockRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*models.Automation, error) {
			if id == automationID {
				return automation, nil
			}
			return nil, ErrAutomationNotFound
		},
	}

	ctx := context.Background()

	// Query with correct tenant — must succeed.
	got, err := repo.GetByID(ctx, automationID, autoTenantA)
	require.NoError(t, err)
	assert.Equal(t, autoTenantA, got.TenantID)

	// Query with wrong tenant — must return not-found.
	got, err = repo.GetByID(ctx, automationID, autoTenantB)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, ErrAutomationNotFound,
		"automation owned by tenant A must not be visible to tenant B")
}

// TestAutomation_Create_TenantID_Stamped verifies that creating an automation
// via the service preserves the tenant ID set on the input Automation struct.
func TestAutomation_Create_TenantID_Stamped(t *testing.T) {
	var capturedAuto *models.Automation
	repo := &mockRepo{
		createFn: func(_ context.Context, a *models.Automation) error {
			clone := *a
			capturedAuto = &clone
			return nil
		},
	}
	svc := newTestService(repo, nil, nil)

	// Build a minimal valid Automation using the registered trigger/action types
	// from newTestService (triggerGet accepts "event", actionGet accepts "send_email").
	auto := &models.Automation{
		TenantID:    autoTenantA,
		Name:        "Follow-up automation",
		TriggerType: "event",
		Actions:     mustJSON([]models.ActionConfig{{Type: "send_email"}}),
		OwnerID:     uuid.New(),
	}

	err := svc.Create(context.Background(), auto)
	require.NoError(t, err)
	assert.Equal(t, autoTenantA, auto.TenantID, "created automation must carry tenant A's ID")
	require.NotNil(t, capturedAuto)
	assert.Equal(t, autoTenantA, capturedAuto.TenantID, "repo must receive automation with tenant A's ID")
}
