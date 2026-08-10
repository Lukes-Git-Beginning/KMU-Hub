package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/kmuhub/kmuhub/internal/automation/action"
	"github.com/kmuhub/kmuhub/internal/automation/condition"
	"github.com/kmuhub/kmuhub/internal/automation/trigger"
	"github.com/kmuhub/kmuhub/internal/automation/workflow"
	"github.com/kmuhub/kmuhub/internal/idempotency"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	automationv1 "github.com/kmuhub/kmuhub/proto/automation/v1"
)

// ============================================================================
// Stub repositories (implement workflow.Repository / ExecutionRepository /
// TemplateRepository). AutomationGRPCServer.svc is a concrete *workflow.Service,
// not an interface, so a real Service must be built from these stubs -- there
// is no way to fake the RPC layer directly (same pattern as
// newPluginTestServer / newRapporteTestServer).
// ============================================================================

type stubAutomationRepo struct {
	createFn          func(ctx context.Context, a *models.Automation) error
	updateFn          func(ctx context.Context, a *models.Automation) error
	deleteFn          func(ctx context.Context, id, tenantID uuid.UUID) error
	getByIDFn         func(ctx context.Context, id, tenantID uuid.UUID) (*models.Automation, error)
	getByIDUnscopedFn func(ctx context.Context, id uuid.UUID) (*models.Automation, error)
	listFn            func(ctx context.Context, f workflow.ListFilter) ([]*models.Automation, int, error)
	setActiveFn       func(ctx context.Context, id, tenantID uuid.UUID, active bool) error
}

func (r *stubAutomationRepo) Create(ctx context.Context, a *models.Automation) error {
	if r.createFn != nil {
		return r.createFn(ctx, a)
	}
	return nil
}

func (r *stubAutomationRepo) Update(ctx context.Context, a *models.Automation) error {
	if r.updateFn != nil {
		return r.updateFn(ctx, a)
	}
	return nil
}

func (r *stubAutomationRepo) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	if r.deleteFn != nil {
		return r.deleteFn(ctx, id, tenantID)
	}
	return nil
}

func (r *stubAutomationRepo) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.Automation, error) {
	if r.getByIDFn != nil {
		return r.getByIDFn(ctx, id, tenantID)
	}
	return nil, workflow.ErrAutomationNotFound
}

func (r *stubAutomationRepo) GetByIDUnscoped(ctx context.Context, id uuid.UUID) (*models.Automation, error) {
	if r.getByIDUnscopedFn != nil {
		return r.getByIDUnscopedFn(ctx, id)
	}
	return nil, workflow.ErrAutomationNotFound
}

func (r *stubAutomationRepo) List(ctx context.Context, f workflow.ListFilter) ([]*models.Automation, int, error) {
	if r.listFn != nil {
		return r.listFn(ctx, f)
	}
	return nil, 0, nil
}

func (r *stubAutomationRepo) ListActiveByTriggerType(_ context.Context, _ string) ([]*models.Automation, error) {
	return nil, nil
}

func (r *stubAutomationRepo) ListActiveTimeBased(_ context.Context, _ []string) ([]*models.Automation, error) {
	return nil, nil
}

func (r *stubAutomationRepo) SetActive(ctx context.Context, id, tenantID uuid.UUID, active bool) error {
	if r.setActiveFn != nil {
		return r.setActiveFn(ctx, id, tenantID, active)
	}
	return nil
}

func (r *stubAutomationRepo) UpdateLastTriggered(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}

func (r *stubAutomationRepo) ClaimTimeTrigger(_ context.Context, _ uuid.UUID, _ *time.Time, _ time.Time) (bool, error) {
	return true, nil
}

func (r *stubAutomationRepo) ClaimTimeTriggerFire(_ context.Context, _, _ uuid.UUID, _ string, _ time.Time) (bool, error) {
	return true, nil
}

type stubAutomationExecRepo struct {
	getExecFn  func(ctx context.Context, id, tenantID uuid.UUID) (*models.AutomationExecution, error)
	listExecFn func(ctx context.Context, f workflow.ExecutionFilter) ([]*models.AutomationExecution, int, error)
}

func (r *stubAutomationExecRepo) CreateExecution(_ context.Context, _ *models.AutomationExecution) error {
	return nil
}

func (r *stubAutomationExecRepo) UpdateExecution(_ context.Context, _ *models.AutomationExecution) error {
	return nil
}

func (r *stubAutomationExecRepo) GetExecution(ctx context.Context, id, tenantID uuid.UUID) (*models.AutomationExecution, error) {
	if r.getExecFn != nil {
		return r.getExecFn(ctx, id, tenantID)
	}
	return nil, workflow.ErrExecutionNotFound
}

func (r *stubAutomationExecRepo) ListExecutions(ctx context.Context, f workflow.ExecutionFilter) ([]*models.AutomationExecution, int, error) {
	if r.listExecFn != nil {
		return r.listExecFn(ctx, f)
	}
	return nil, 0, nil
}

func (r *stubAutomationExecRepo) CleanupOldExecutions(_ context.Context, _ time.Duration) (int64, error) {
	return 0, nil
}

type stubAutomationTemplateRepo struct {
	listTemplatesFn func(ctx context.Context, category *string) ([]*models.AutomationTemplate, error)
	getTemplateFn   func(ctx context.Context, id string) (*models.AutomationTemplate, error)
}

func (r *stubAutomationTemplateRepo) ListTemplates(ctx context.Context, category *string) ([]*models.AutomationTemplate, error) {
	if r.listTemplatesFn != nil {
		return r.listTemplatesFn(ctx, category)
	}
	return nil, nil
}

func (r *stubAutomationTemplateRepo) GetTemplate(ctx context.Context, id string) (*models.AutomationTemplate, error) {
	if r.getTemplateFn != nil {
		return r.getTemplateFn(ctx, id)
	}
	return nil, nil
}

func (r *stubAutomationTemplateRepo) UpsertTemplate(_ context.Context, _ *models.AutomationTemplate) error {
	return nil
}

// stubAutomationIdempotencyRepo / stubAutomationExecutor exist only to satisfy
// workflow.NewService's constructor signature -- none of the tests in this
// file exercise TriggerWebhook past the automation lookup, so both are no-ops.
type stubAutomationIdempotencyRepo struct{}

func (stubAutomationIdempotencyRepo) Reserve(_ context.Context, _ string, _, _ uuid.UUID, _, _, _ string) (*idempotency.Record, error) {
	return nil, nil
}
func (stubAutomationIdempotencyRepo) Get(_ context.Context, _ uuid.UUID, _ string) (*idempotency.Record, error) {
	return nil, nil
}
func (stubAutomationIdempotencyRepo) Complete(_ context.Context, _ uuid.UUID, _ string, _ int, _ []byte) error {
	return nil
}
func (stubAutomationIdempotencyRepo) Cleanup(_ context.Context) (int, error) { return 0, nil }
func (stubAutomationIdempotencyRepo) CleanupWithLock(_ context.Context, _ int64) (int, error) {
	return 0, nil
}

type stubAutomationExecutor struct{}

func (stubAutomationExecutor) Execute(_ context.Context, _ models.Automation, _ models.EventPayload) error {
	return nil
}

// ============================================================================
// Trigger / action catalog fakes
// ============================================================================

func automationTriggerGet(t string) (any, bool) {
	if t == "event" || t == "schedule" || t == "webhook.received" {
		return trigger.TriggerDefinition{Type: t, Module: "crm", Name: "Trigger " + t, Description: "desc"}, true
	}
	return nil, false
}

func automationTriggerAll() []any {
	return []any{trigger.TriggerDefinition{Type: "event", Module: "crm", Name: "Ereignis-Trigger", Description: "Feuert bei einem Ereignis"}}
}

func automationActionGet(t string) (any, bool) {
	if t == "send_email" || t == "add_tag" {
		return action.ActionDefinition{Type: t, Module: "notification", Name: "Aktion " + t, Description: "desc"}, true
	}
	return nil, false
}

func automationActionAllDefs() []any {
	return []any{action.ActionDefinition{Type: "send_email", Module: "notification", Name: "E-Mail senden", Description: "Sendet eine E-Mail"}}
}

// ============================================================================
// Test helpers
// ============================================================================

func newAutomationTestServer(repo *stubAutomationRepo, execRepo *stubAutomationExecRepo, tmplRepo *stubAutomationTemplateRepo) *AutomationGRPCServer {
	if repo == nil {
		repo = &stubAutomationRepo{}
	}
	if execRepo == nil {
		execRepo = &stubAutomationExecRepo{}
	}
	if tmplRepo == nil {
		tmplRepo = &stubAutomationTemplateRepo{}
	}
	svc := workflow.NewService(
		repo, execRepo, tmplRepo,
		automationTriggerGet, automationTriggerAll,
		automationActionGet, automationActionAllDefs,
		condition.NewEvaluator(),
		stubAutomationIdempotencyRepo{},
		stubAutomationExecutor{},
	)
	return NewAutomationGRPCServer(svc)
}

func automationCtxWithTenant(tenantID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), middleware.TenantIDKey, tenantID.String())
}

func automationStruct(t *testing.T, v map[string]any) *structpb.Struct {
	t.Helper()
	s, err := structpb.NewStruct(v)
	require.NoError(t, err)
	return s
}

func automationList(t *testing.T, v []any) *structpb.ListValue {
	t.Helper()
	l, err := structpb.NewList(v)
	require.NoError(t, err)
	return l
}

// ============================================================================
// mapDomainError
// ============================================================================

func TestMapDomainError_Table(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"automation not found", workflow.ErrAutomationNotFound, codes.NotFound},
		{"execution not found", workflow.ErrExecutionNotFound, codes.NotFound},
		{"template not found", workflow.ErrTemplateNotFound, codes.NotFound},
		{"invalid condition", workflow.ErrInvalidCondition, codes.InvalidArgument},
		{"invalid action", workflow.ErrInvalidAction, codes.InvalidArgument},
		{"inactive", workflow.ErrAutomationInactive, codes.FailedPrecondition},
		{"loop detected", workflow.ErrLoopDetected, codes.Aborted},
		{"circuit breaker open", workflow.ErrCircuitBreakerOpen, codes.ResourceExhausted},
		{"webhook not found", workflow.ErrWebhookNotFound, codes.NotFound},
		{"webhook signature invalid", workflow.ErrWebhookSignatureInvalid, codes.Unauthenticated},
		{"webhook payload too large", workflow.ErrWebhookPayloadTooLarge, codes.InvalidArgument},
		{"webhook idempotency conflict", workflow.ErrWebhookIdempotencyConflict, codes.InvalidArgument},
		{"unmapped sentinel", errors.New("boom"), codes.Internal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireGRPCCode(t, mapDomainError(tc.err), tc.want)
		})
	}

	assert.NoError(t, mapDomainError(nil))
}

// ============================================================================
// CreateAutomation
// ============================================================================

func TestCreateAutomation_MissingTenant(t *testing.T) {
	s := newAutomationTestServer(nil, nil, nil)
	_, err := s.CreateAutomation(context.Background(), &automationv1.CreateAutomationRequest{OwnerId: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateAutomation_InvalidOwnerID(t *testing.T) {
	s := newAutomationTestServer(nil, nil, nil)
	req := &automationv1.CreateAutomationRequest{OwnerId: "not-a-uuid", TriggerType: "event"}
	_, err := s.CreateAutomation(automationCtxWithTenant(uuid.New()), req)
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateAutomation_UnknownTriggerType(t *testing.T) {
	s := newAutomationTestServer(nil, nil, nil)
	req := &automationv1.CreateAutomationRequest{OwnerId: uuid.NewString(), TriggerType: "nonexistent"}
	_, err := s.CreateAutomation(automationCtxWithTenant(uuid.New()), req)
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// TestCreateAutomation_ActionsStructCannotCarryAnArray documents the fix for
// a wire-shape defect (see JOURNAL.md, fix-automation-actions-struct-cannot-
// represent-array): automation.proto used to declare `actions` as
// google.protobuf.Struct (JSON-object-only), but workflow.Service.validateAutomation
// unmarshals auto.Actions into []models.ActionConfig -- a JSON array. A
// structpb.Struct can only ever marshal to `{...}`, never `[...]`, so every
// non-nil Actions payload -- even a single, otherwise-valid action -- was
// rejected as ErrInvalidAction. `actions` is now google.protobuf.ListValue,
// so a real action list round-trips instead of being rejected.
func TestCreateAutomation_ActionsStructCannotCarryAnArray(t *testing.T) {
	var captured *models.Automation
	repo := &stubAutomationRepo{
		createFn: func(_ context.Context, a *models.Automation) error {
			captured = a
			return nil
		},
	}
	s := newAutomationTestServer(repo, nil, nil)
	req := &automationv1.CreateAutomationRequest{
		OwnerId:     uuid.NewString(),
		Name:        "Would-be valid automation",
		TriggerType: "event",
		Actions:     automationList(t, []any{map[string]any{"type": "send_email"}}),
	}
	resp, err := s.CreateAutomation(automationCtxWithTenant(uuid.New()), req)
	require.NoError(t, err)
	require.NotNil(t, captured)
	require.JSONEq(t, `[{"type":"send_email"}]`, string(captured.Actions))
	require.NotNil(t, resp.GetAutomation().GetActions())
	require.Len(t, resp.GetAutomation().GetActions().GetValues(), 1)
}

func TestCreateAutomation_Success(t *testing.T) {
	var captured *models.Automation
	repo := &stubAutomationRepo{
		createFn: func(_ context.Context, a *models.Automation) error {
			captured = a
			return nil
		},
	}
	s := newAutomationTestServer(repo, nil, nil)

	tenantID := uuid.New()
	ownerID := uuid.New()
	req := &automationv1.CreateAutomationRequest{
		OwnerId:       ownerID.String(),
		Name:          "Notify on deal win",
		Description:   "desc",
		Scope:         automationv1.AutomationScope_SCOPE_TEAM,
		TriggerType:   "event",
		TriggerConfig: automationStruct(t, map[string]any{"event": "deal.won"}),
		MaxSteps:      5,
	}

	resp, err := s.CreateAutomation(automationCtxWithTenant(tenantID), req)
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, tenantID, captured.TenantID)
	assert.Equal(t, ownerID, captured.OwnerID)
	assert.Equal(t, models.AutomationScopeTeam, captured.Scope)
	assert.Equal(t, 5, captured.MaxSteps)

	require.NotNil(t, resp.Automation)
	assert.Equal(t, "Notify on deal win", resp.Automation.Name)
	assert.Equal(t, automationv1.AutomationScope_SCOPE_TEAM, resp.Automation.Scope)
	require.NotNil(t, resp.Automation.TriggerConfig)
	assert.Equal(t, "deal.won", resp.Automation.TriggerConfig.Fields["event"].GetStringValue())
}

func TestCreateAutomation_DefaultScopeWhenUnspecified(t *testing.T) {
	var captured *models.Automation
	repo := &stubAutomationRepo{
		createFn: func(_ context.Context, a *models.Automation) error {
			captured = a
			return nil
		},
	}
	s := newAutomationTestServer(repo, nil, nil)

	req := &automationv1.CreateAutomationRequest{
		OwnerId:     uuid.NewString(),
		Name:        "Default scope",
		TriggerType: "event",
	}
	_, err := s.CreateAutomation(automationCtxWithTenant(uuid.New()), req)
	require.NoError(t, err)
	assert.Equal(t, models.AutomationScopePersonal, captured.Scope)
}

// ============================================================================
// UpdateAutomation
// ============================================================================

func TestUpdateAutomation_MissingTenant(t *testing.T) {
	s := newAutomationTestServer(nil, nil, nil)
	_, err := s.UpdateAutomation(context.Background(), &automationv1.UpdateAutomationRequest{AutomationId: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateAutomation_InvalidID(t *testing.T) {
	s := newAutomationTestServer(nil, nil, nil)
	req := &automationv1.UpdateAutomationRequest{AutomationId: "not-a-uuid"}
	_, err := s.UpdateAutomation(automationCtxWithTenant(uuid.New()), req)
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateAutomation_NotFound(t *testing.T) {
	repo := &stubAutomationRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*models.Automation, error) {
			return nil, workflow.ErrAutomationNotFound
		},
	}
	s := newAutomationTestServer(repo, nil, nil)
	req := &automationv1.UpdateAutomationRequest{AutomationId: uuid.NewString()}
	_, err := s.UpdateAutomation(automationCtxWithTenant(uuid.New()), req)
	requireGRPCCode(t, err, codes.NotFound)
}

func TestUpdateAutomation_PartialUpdatePreservesUntouchedFields(t *testing.T) {
	tenantID := uuid.New()
	autoID := uuid.New()
	existing := &models.Automation{
		ID:          autoID,
		TenantID:    tenantID,
		Name:        "Original Name",
		Description: "Original Description",
		TriggerType: "event",
		MaxSteps:    10,
	}
	var updated *models.Automation
	repo := &stubAutomationRepo{
		getByIDFn: func(_ context.Context, id, tid uuid.UUID) (*models.Automation, error) {
			if id == autoID && tid == tenantID {
				return existing, nil
			}
			return nil, workflow.ErrAutomationNotFound
		},
		updateFn: func(_ context.Context, a *models.Automation) error {
			updated = a
			return nil
		},
	}
	s := newAutomationTestServer(repo, nil, nil)

	newName := "Renamed"
	req := &automationv1.UpdateAutomationRequest{
		AutomationId: autoID.String(),
		Name:         &newName,
	}
	resp, err := s.UpdateAutomation(automationCtxWithTenant(tenantID), req)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "Renamed", updated.Name)
	assert.Equal(t, "Original Description", updated.Description, "untouched field must survive a partial update")
	assert.Equal(t, "Renamed", resp.Automation.Name)
}

// ============================================================================
// DeleteAutomation
// ============================================================================

func TestDeleteAutomation_MissingTenant(t *testing.T) {
	s := newAutomationTestServer(nil, nil, nil)
	_, err := s.DeleteAutomation(context.Background(), &automationv1.DeleteAutomationRequest{AutomationId: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeleteAutomation_InvalidID(t *testing.T) {
	s := newAutomationTestServer(nil, nil, nil)
	req := &automationv1.DeleteAutomationRequest{AutomationId: "bad"}
	_, err := s.DeleteAutomation(automationCtxWithTenant(uuid.New()), req)
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeleteAutomation_Success(t *testing.T) {
	tenantID := uuid.New()
	autoID := uuid.New()
	var deletedID, deletedTenant uuid.UUID
	repo := &stubAutomationRepo{
		deleteFn: func(_ context.Context, id, tid uuid.UUID) error {
			deletedID, deletedTenant = id, tid
			return nil
		},
	}
	s := newAutomationTestServer(repo, nil, nil)
	req := &automationv1.DeleteAutomationRequest{AutomationId: autoID.String()}
	_, err := s.DeleteAutomation(automationCtxWithTenant(tenantID), req)
	require.NoError(t, err)
	assert.Equal(t, autoID, deletedID)
	assert.Equal(t, tenantID, deletedTenant)
}

// ============================================================================
// GetAutomation
// ============================================================================

func TestGetAutomation_MissingTenant(t *testing.T) {
	s := newAutomationTestServer(nil, nil, nil)
	_, err := s.GetAutomation(context.Background(), &automationv1.GetAutomationRequest{AutomationId: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetAutomation_InvalidID(t *testing.T) {
	s := newAutomationTestServer(nil, nil, nil)
	req := &automationv1.GetAutomationRequest{AutomationId: "bad"}
	_, err := s.GetAutomation(automationCtxWithTenant(uuid.New()), req)
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetAutomation_NotFound(t *testing.T) {
	repo := &stubAutomationRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*models.Automation, error) {
			return nil, workflow.ErrAutomationNotFound
		},
	}
	s := newAutomationTestServer(repo, nil, nil)
	req := &automationv1.GetAutomationRequest{AutomationId: uuid.NewString()}
	_, err := s.GetAutomation(automationCtxWithTenant(uuid.New()), req)
	requireGRPCCode(t, err, codes.NotFound)
}

func TestGetAutomation_Success(t *testing.T) {
	tenantID := uuid.New()
	autoID := uuid.New()
	expected := &models.Automation{ID: autoID, TenantID: tenantID, Name: "Found", TriggerType: "event"}
	repo := &stubAutomationRepo{
		getByIDFn: func(_ context.Context, id, tid uuid.UUID) (*models.Automation, error) {
			if id == autoID && tid == tenantID {
				return expected, nil
			}
			return nil, workflow.ErrAutomationNotFound
		},
	}
	s := newAutomationTestServer(repo, nil, nil)
	req := &automationv1.GetAutomationRequest{AutomationId: autoID.String()}
	resp, err := s.GetAutomation(automationCtxWithTenant(tenantID), req)
	require.NoError(t, err)
	assert.Equal(t, "Found", resp.Automation.Name)
}

// ============================================================================
// ListAutomations
// ============================================================================

func TestListAutomations_MalformedOwnerIDIsIgnoredNotRejected(t *testing.T) {
	var captured workflow.ListFilter
	repo := &stubAutomationRepo{
		listFn: func(_ context.Context, f workflow.ListFilter) ([]*models.Automation, int, error) {
			captured = f
			return nil, 0, nil
		},
	}
	s := newAutomationTestServer(repo, nil, nil)
	req := &automationv1.ListAutomationsRequest{OwnerId: "not-a-uuid"}
	_, err := s.ListAutomations(automationCtxWithTenant(uuid.New()), req)
	require.NoError(t, err)
	assert.Nil(t, captured.OwnerID, "a malformed owner_id must be dropped from the filter, not rejected")
}

func TestListAutomations_ScopeAndTriggerTypeFilters(t *testing.T) {
	var captured workflow.ListFilter
	repo := &stubAutomationRepo{
		listFn: func(_ context.Context, f workflow.ListFilter) ([]*models.Automation, int, error) {
			captured = f
			return []*models.Automation{{ID: uuid.New(), Name: "A"}}, 1, nil
		},
	}
	s := newAutomationTestServer(repo, nil, nil)
	isActive := true
	req := &automationv1.ListAutomationsRequest{
		Scope:       automationv1.AutomationScope_SCOPE_ORGANIZATION.Enum(),
		TriggerType: strPtr("event"),
		IsActive:    &isActive,
	}
	resp, err := s.ListAutomations(automationCtxWithTenant(uuid.New()), req)
	require.NoError(t, err)
	require.NotNil(t, captured.Scope)
	assert.Equal(t, models.AutomationScopeOrganization, *captured.Scope)
	require.NotNil(t, captured.TriggerType)
	assert.Equal(t, "event", *captured.TriggerType)
	require.NotNil(t, captured.IsActive)
	assert.True(t, *captured.IsActive)
	assert.Equal(t, int32(1), resp.TotalCount)
	assert.Len(t, resp.Automations, 1)
}

// ============================================================================
// EnableAutomation / DisableAutomation
// ============================================================================

func TestEnableAutomation_NotFound(t *testing.T) {
	repo := &stubAutomationRepo{
		setActiveFn: func(_ context.Context, _, _ uuid.UUID, _ bool) error {
			return workflow.ErrAutomationNotFound
		},
	}
	s := newAutomationTestServer(repo, nil, nil)
	req := &automationv1.EnableAutomationRequest{AutomationId: uuid.NewString()}
	_, err := s.EnableAutomation(automationCtxWithTenant(uuid.New()), req)
	requireGRPCCode(t, err, codes.NotFound)
}

func TestEnableAutomation_Success(t *testing.T) {
	tenantID := uuid.New()
	autoID := uuid.New()
	auto := &models.Automation{ID: autoID, TenantID: tenantID, TriggerType: "event"}
	var capturedActive bool
	repo := &stubAutomationRepo{
		setActiveFn: func(_ context.Context, _, _ uuid.UUID, active bool) error {
			capturedActive = active
			auto.IsActive = active
			return nil
		},
		getByIDFn: func(_ context.Context, id, tid uuid.UUID) (*models.Automation, error) {
			if id == autoID && tid == tenantID {
				return auto, nil
			}
			return nil, workflow.ErrAutomationNotFound
		},
	}
	s := newAutomationTestServer(repo, nil, nil)
	req := &automationv1.EnableAutomationRequest{AutomationId: autoID.String()}
	resp, err := s.EnableAutomation(automationCtxWithTenant(tenantID), req)
	require.NoError(t, err)
	assert.True(t, capturedActive)
	assert.True(t, resp.Automation.IsActive)
}

func TestDisableAutomation_Success(t *testing.T) {
	tenantID := uuid.New()
	autoID := uuid.New()
	auto := &models.Automation{ID: autoID, TenantID: tenantID, TriggerType: "event", IsActive: true}
	repo := &stubAutomationRepo{
		setActiveFn: func(_ context.Context, _, _ uuid.UUID, active bool) error {
			auto.IsActive = active
			return nil
		},
		getByIDFn: func(_ context.Context, id, tid uuid.UUID) (*models.Automation, error) {
			if id == autoID && tid == tenantID {
				return auto, nil
			}
			return nil, workflow.ErrAutomationNotFound
		},
	}
	s := newAutomationTestServer(repo, nil, nil)
	req := &automationv1.DisableAutomationRequest{AutomationId: autoID.String()}
	resp, err := s.DisableAutomation(automationCtxWithTenant(tenantID), req)
	require.NoError(t, err)
	assert.False(t, resp.Automation.IsActive)
}

// ============================================================================
// ListExecutions / GetExecution
// ============================================================================

func TestListExecutions_MalformedAutomationIDIsIgnored(t *testing.T) {
	var captured workflow.ExecutionFilter
	execRepo := &stubAutomationExecRepo{
		listExecFn: func(_ context.Context, f workflow.ExecutionFilter) ([]*models.AutomationExecution, int, error) {
			captured = f
			return nil, 0, nil
		},
	}
	s := newAutomationTestServer(nil, execRepo, nil)
	badID := "not-a-uuid"
	req := &automationv1.ListExecutionsRequest{AutomationId: &badID}
	_, err := s.ListExecutions(automationCtxWithTenant(uuid.New()), req)
	require.NoError(t, err)
	assert.Nil(t, captured.AutomationID)
}

func TestListExecutions_StatusFilter(t *testing.T) {
	var captured workflow.ExecutionFilter
	execRepo := &stubAutomationExecRepo{
		listExecFn: func(_ context.Context, f workflow.ExecutionFilter) ([]*models.AutomationExecution, int, error) {
			captured = f
			return []*models.AutomationExecution{{ID: uuid.New(), Status: models.ExecutionStatusFailed}}, 1, nil
		},
	}
	s := newAutomationTestServer(nil, execRepo, nil)
	status := automationv1.ExecutionStatus_EXECUTION_STATUS_FAILED
	req := &automationv1.ListExecutionsRequest{Status: &status}
	resp, err := s.ListExecutions(automationCtxWithTenant(uuid.New()), req)
	require.NoError(t, err)
	require.NotNil(t, captured.Status)
	assert.Equal(t, models.ExecutionStatusFailed, *captured.Status)
	require.Len(t, resp.Executions, 1)
	assert.Equal(t, automationv1.ExecutionStatus_EXECUTION_STATUS_FAILED, resp.Executions[0].Status)
}

func TestGetExecution_InvalidID(t *testing.T) {
	s := newAutomationTestServer(nil, nil, nil)
	req := &automationv1.GetExecutionRequest{ExecutionId: "bad"}
	_, err := s.GetExecution(automationCtxWithTenant(uuid.New()), req)
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetExecution_Success(t *testing.T) {
	tenantID := uuid.New()
	execID := uuid.New()
	expected := &models.AutomationExecution{ID: execID, TenantID: tenantID, Status: models.ExecutionStatusCompleted}
	execRepo := &stubAutomationExecRepo{
		getExecFn: func(_ context.Context, id, tid uuid.UUID) (*models.AutomationExecution, error) {
			if id == execID && tid == tenantID {
				return expected, nil
			}
			return nil, workflow.ErrExecutionNotFound
		},
	}
	s := newAutomationTestServer(nil, execRepo, nil)
	req := &automationv1.GetExecutionRequest{ExecutionId: execID.String()}
	resp, err := s.GetExecution(automationCtxWithTenant(tenantID), req)
	require.NoError(t, err)
	assert.Equal(t, execID.String(), resp.Execution.Id)
}

// ============================================================================
// ListTriggerDefinitions / ListActionDefinitions
// ============================================================================

func TestListTriggerDefinitions_ConvertsCatalog(t *testing.T) {
	s := newAutomationTestServer(nil, nil, nil)
	resp, err := s.ListTriggerDefinitions(context.Background(), &automationv1.ListTriggerDefinitionsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Triggers, 1)
	assert.Equal(t, "event", resp.Triggers[0].Type)
	assert.Equal(t, "crm", resp.Triggers[0].Module)
	assert.Equal(t, "Ereignis-Trigger", resp.Triggers[0].Name)
	assert.Equal(t, "Feuert bei einem Ereignis", resp.Triggers[0].Description)
}

func TestListActionDefinitions_ConvertsCatalog(t *testing.T) {
	s := newAutomationTestServer(nil, nil, nil)
	resp, err := s.ListActionDefinitions(context.Background(), &automationv1.ListActionDefinitionsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.Actions, 1)
	assert.Equal(t, "send_email", resp.Actions[0].Type)
	assert.Equal(t, "notification", resp.Actions[0].Module)
	assert.Equal(t, "E-Mail senden", resp.Actions[0].Name)
}

// ============================================================================
// ListTemplates / CreateFromTemplate
// ============================================================================

func TestListTemplates_CategoryFilterPassedThrough(t *testing.T) {
	var captured *string
	tmplRepo := &stubAutomationTemplateRepo{
		listTemplatesFn: func(_ context.Context, category *string) ([]*models.AutomationTemplate, error) {
			captured = category
			return []*models.AutomationTemplate{{ID: "tmpl-1", Name: "Welcome"}}, nil
		},
	}
	s := newAutomationTestServer(nil, nil, tmplRepo)
	category := "onboarding"
	req := &automationv1.ListTemplatesRequest{Category: &category}
	resp, err := s.ListTemplates(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, "onboarding", *captured)
	require.Len(t, resp.Templates, 1)
	assert.Equal(t, "Welcome", resp.Templates[0].Name)
}

func TestCreateFromTemplate_MissingTenant(t *testing.T) {
	s := newAutomationTestServer(nil, nil, nil)
	_, err := s.CreateFromTemplate(context.Background(), &automationv1.CreateFromTemplateRequest{OwnerId: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateFromTemplate_InvalidOwnerID(t *testing.T) {
	s := newAutomationTestServer(nil, nil, nil)
	req := &automationv1.CreateFromTemplateRequest{OwnerId: "bad", TemplateId: "tmpl-1"}
	_, err := s.CreateFromTemplate(automationCtxWithTenant(uuid.New()), req)
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateFromTemplate_TemplateNotFound(t *testing.T) {
	tmplRepo := &stubAutomationTemplateRepo{
		getTemplateFn: func(_ context.Context, _ string) (*models.AutomationTemplate, error) {
			return nil, nil
		},
	}
	s := newAutomationTestServer(nil, nil, tmplRepo)
	req := &automationv1.CreateFromTemplateRequest{OwnerId: uuid.NewString(), TemplateId: "nonexistent"}
	_, err := s.CreateFromTemplate(automationCtxWithTenant(uuid.New()), req)
	requireGRPCCode(t, err, codes.NotFound)
}

func TestCreateFromTemplate_Success(t *testing.T) {
	tmpl := &models.AutomationTemplate{ID: "tmpl-1", Name: "Welcome", TriggerType: "event"}
	tmplRepo := &stubAutomationTemplateRepo{
		getTemplateFn: func(_ context.Context, id string) (*models.AutomationTemplate, error) {
			if id == "tmpl-1" {
				return tmpl, nil
			}
			return nil, nil
		},
	}
	var created *models.Automation
	repo := &stubAutomationRepo{
		createFn: func(_ context.Context, a *models.Automation) error {
			created = a
			return nil
		},
	}
	s := newAutomationTestServer(repo, nil, tmplRepo)
	req := &automationv1.CreateFromTemplateRequest{OwnerId: uuid.NewString(), TemplateId: "tmpl-1", Name: "My Welcome"}
	resp, err := s.CreateFromTemplate(automationCtxWithTenant(uuid.New()), req)
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.False(t, created.IsActive, "created-from-template automations must start inactive")
	assert.Equal(t, "My Welcome", resp.Automation.Name)
}

// ============================================================================
// TestCondition
// ============================================================================

// TestTestCondition_OmittedConditionsSurfacesAsError documents an inconsistency
// with DryRunAutomation: DryRun treats a nil/empty Conditions payload as "no
// conditions -- always matches" (auto.Conditions len check before unmarshal,
// service.go DryRun). TestCondition has no such guard -- it unmarshals
// automationStructToJSON(req.GetConditions()) unconditionally, and that helper
// returns nil for an omitted (nil) Struct, so json.Unmarshal(nil, &config)
// fails. A caller testing "no conditions configured" (a normal case: the
// automation always fires) gets a confusing error instead of Matches=true.
func TestTestCondition_OmittedConditionsSurfacesAsError(t *testing.T) {
	s := newAutomationTestServer(nil, nil, nil)
	resp, err := s.TestCondition(context.Background(), &automationv1.TestConditionRequest{})
	require.NoError(t, err, "unmarshal failure is a soft error, not a gRPC error")
	assert.False(t, resp.Matches, "documents the current gap: an omitted conditions payload does not match-all like DryRun does")
	assert.Contains(t, resp.ErrorMessage, "unexpected end of JSON input")
}

func TestTestCondition_InvalidConfigReturnsSoftError(t *testing.T) {
	s := newAutomationTestServer(nil, nil, nil)
	req := &automationv1.TestConditionRequest{
		Conditions: automationStruct(t, map[string]any{"mode": "expression", "expression": "this is not valid expr syntax {{{"}),
	}
	resp, err := s.TestCondition(context.Background(), req)
	require.NoError(t, err, "an invalid condition must surface as a response field, not a gRPC error")
	assert.False(t, resp.Matches)
	assert.NotEmpty(t, resp.ErrorMessage)
}

func TestTestCondition_EmptyModeAlwaysMatches(t *testing.T) {
	s := newAutomationTestServer(nil, nil, nil)
	req := &automationv1.TestConditionRequest{Conditions: automationStruct(t, map[string]any{})}
	resp, err := s.TestCondition(context.Background(), req)
	require.NoError(t, err)
	assert.True(t, resp.Matches)
	assert.Empty(t, resp.ErrorMessage)
}

// ============================================================================
// DryRunAutomation
// ============================================================================

func TestDryRunAutomation_MissingTenant(t *testing.T) {
	s := newAutomationTestServer(nil, nil, nil)
	_, err := s.DryRunAutomation(context.Background(), &automationv1.DryRunAutomationRequest{AutomationId: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDryRunAutomation_InvalidID(t *testing.T) {
	s := newAutomationTestServer(nil, nil, nil)
	req := &automationv1.DryRunAutomationRequest{AutomationId: "bad"}
	_, err := s.DryRunAutomation(automationCtxWithTenant(uuid.New()), req)
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// TestDryRunAutomation_NotFoundIsASoftError documents that, unlike most other
// RPCs, a not-found automation here is NOT a gRPC error -- DryRun's service
// method returns (nil, err) and the handler folds that into
// DryRunAutomationResponse.ErrorMessage with a 200-equivalent OK status.
func TestDryRunAutomation_NotFoundIsASoftError(t *testing.T) {
	s := newAutomationTestServer(nil, nil, nil)
	req := &automationv1.DryRunAutomationRequest{AutomationId: uuid.NewString()}
	resp, err := s.DryRunAutomation(automationCtxWithTenant(uuid.New()), req)
	require.NoError(t, err)
	assert.False(t, resp.ConditionMatched)
	assert.NotEmpty(t, resp.ErrorMessage)
}

func TestDryRunAutomation_Success(t *testing.T) {
	tenantID := uuid.New()
	autoID := uuid.New()
	auto := &models.Automation{
		ID:          autoID,
		TenantID:    tenantID,
		TriggerType: "event",
		Actions:     []byte(`[{"type":"send_email"},{"type":"unregistered_action"}]`),
	}
	repo := &stubAutomationRepo{
		getByIDFn: func(_ context.Context, id, tid uuid.UUID) (*models.Automation, error) {
			if id == autoID && tid == tenantID {
				return auto, nil
			}
			return nil, workflow.ErrAutomationNotFound
		},
	}
	s := newAutomationTestServer(repo, nil, nil)
	req := &automationv1.DryRunAutomationRequest{AutomationId: autoID.String()}
	resp, err := s.DryRunAutomation(automationCtxWithTenant(tenantID), req)
	require.NoError(t, err)
	assert.True(t, resp.ConditionMatched, "no conditions configured must match")
	require.Len(t, resp.SimulatedSteps, 2)
	assert.Equal(t, "send_email", resp.SimulatedSteps[0].ActionType)
	assert.Empty(t, resp.SimulatedSteps[0].Error)
	assert.Equal(t, "unregistered_action", resp.SimulatedSteps[1].ActionType)
	assert.NotEmpty(t, resp.SimulatedSteps[1].Error, "an action type missing from the catalog must be reported, not silently dropped")
}

// ============================================================================
// GetAutomationStats
// ============================================================================

func TestGetAutomationStats_MissingTenant(t *testing.T) {
	s := newAutomationTestServer(nil, nil, nil)
	_, err := s.GetAutomationStats(context.Background(), &automationv1.GetAutomationStatsRequest{})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetAutomationStats_Success(t *testing.T) {
	repo := &stubAutomationRepo{
		listFn: func(_ context.Context, f workflow.ListFilter) ([]*models.Automation, int, error) {
			if f.IsActive != nil && *f.IsActive {
				return nil, 2, nil
			}
			return nil, 5, nil
		},
	}
	execRepo := &stubAutomationExecRepo{
		listExecFn: func(_ context.Context, f workflow.ExecutionFilter) ([]*models.AutomationExecution, int, error) {
			if f.Status == nil {
				return nil, 0, nil
			}
			switch *f.Status {
			case models.ExecutionStatusCompleted:
				return nil, 8, nil
			case models.ExecutionStatusFailed:
				return nil, 2, nil
			}
			return nil, 0, nil
		},
	}
	s := newAutomationTestServer(repo, execRepo, nil)
	resp, err := s.GetAutomationStats(automationCtxWithTenant(uuid.New()), &automationv1.GetAutomationStatsRequest{})
	require.NoError(t, err)
	assert.Equal(t, int32(5), resp.TotalAutomations)
	assert.Equal(t, int32(2), resp.ActiveAutomations)
	assert.Equal(t, int32(8), resp.SuccessfulExecutions)
	assert.Equal(t, int32(2), resp.FailedExecutions)
	assert.Equal(t, int32(10), resp.TotalExecutions)
}

// ============================================================================
// TriggerWebhook
// ============================================================================

func TestTriggerWebhook_InvalidAutomationID(t *testing.T) {
	s := newAutomationTestServer(nil, nil, nil)
	_, err := s.TriggerWebhook(context.Background(), &automationv1.TriggerWebhookRequest{AutomationId: "bad"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// TestTriggerWebhook_UnknownAutomationMapsToNotFound exercises the full
// handler -> service -> mapDomainError chain for the one RPC that
// deliberately skips middleware.GetTenantID (see the doc comment on
// TriggerWebhook in automation_grpc.go): the automation is looked up
// unscoped, and an unknown ID surfaces as NotFound like any other RPC.
func TestTriggerWebhook_UnknownAutomationMapsToNotFound(t *testing.T) {
	repo := &stubAutomationRepo{
		getByIDUnscopedFn: func(_ context.Context, _ uuid.UUID) (*models.Automation, error) {
			return nil, workflow.ErrAutomationNotFound
		},
	}
	s := newAutomationTestServer(repo, nil, nil)
	req := &automationv1.TriggerWebhookRequest{AutomationId: uuid.NewString()}
	_, err := s.TriggerWebhook(context.Background(), req)
	requireGRPCCode(t, err, codes.NotFound)
}
