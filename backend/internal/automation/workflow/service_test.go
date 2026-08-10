package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/automation/condition"
	"github.com/kmuhub/kmuhub/internal/idempotency"
	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// Mock Repositories
// ============================================================================

type mockRepo struct {
	createFn            func(ctx context.Context, a *models.Automation) error
	updateFn            func(ctx context.Context, a *models.Automation) error
	deleteFn            func(ctx context.Context, id uuid.UUID) error
	getByIDFn           func(ctx context.Context, id uuid.UUID) (*models.Automation, error)
	getByIDUnscopedFn   func(ctx context.Context, id uuid.UUID) (*models.Automation, error)
	listFn              func(ctx context.Context, f ListFilter) ([]*models.Automation, int, error)
	listActiveByTrigFn  func(ctx context.Context, triggerType string) ([]*models.Automation, error)
	listActiveTimeBasFn func(ctx context.Context, triggerTypes []string) ([]*models.Automation, error)
	setActiveFn         func(ctx context.Context, id uuid.UUID, active bool) error
	updateLastTriggFn   func(ctx context.Context, id uuid.UUID, at time.Time) error
	claimTimeTriggerFn  func(ctx context.Context, id uuid.UUID, previous *time.Time, now time.Time) (bool, error)
}

func (m *mockRepo) Create(ctx context.Context, a *models.Automation) error {
	if m.createFn != nil {
		return m.createFn(ctx, a)
	}
	return nil
}

func (m *mockRepo) Update(ctx context.Context, a *models.Automation) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, a)
	}
	return nil
}

func (m *mockRepo) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockRepo) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.Automation, error) {
	if m.getByIDFn != nil {
		auto, err := m.getByIDFn(ctx, id)
		if err != nil || auto == nil {
			return auto, err
		}
		// Enforce tenant isolation when a real tenantID is supplied.
		if tenantID != uuid.Nil && auto.TenantID != uuid.Nil && auto.TenantID != tenantID {
			return nil, ErrAutomationNotFound
		}
		return auto, nil
	}
	return nil, nil
}

func (m *mockRepo) GetByIDUnscoped(ctx context.Context, id uuid.UUID) (*models.Automation, error) {
	if m.getByIDUnscopedFn != nil {
		return m.getByIDUnscopedFn(ctx, id)
	}
	return nil, ErrAutomationNotFound
}

func (m *mockRepo) List(ctx context.Context, f ListFilter) ([]*models.Automation, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, f)
	}
	return nil, 0, nil
}

func (m *mockRepo) ListActiveByTriggerType(ctx context.Context, triggerType string) ([]*models.Automation, error) {
	if m.listActiveByTrigFn != nil {
		return m.listActiveByTrigFn(ctx, triggerType)
	}
	return nil, nil
}

func (m *mockRepo) ListActiveTimeBased(ctx context.Context, triggerTypes []string) ([]*models.Automation, error) {
	if m.listActiveTimeBasFn != nil {
		return m.listActiveTimeBasFn(ctx, triggerTypes)
	}
	return nil, nil
}

func (m *mockRepo) SetActive(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, active bool) error {
	if m.setActiveFn != nil {
		return m.setActiveFn(ctx, id, active)
	}
	return nil
}

func (m *mockRepo) UpdateLastTriggered(ctx context.Context, id uuid.UUID, at time.Time) error {
	if m.updateLastTriggFn != nil {
		return m.updateLastTriggFn(ctx, id, at)
	}
	return nil
}

func (m *mockRepo) ClaimTimeTrigger(ctx context.Context, id uuid.UUID, previous *time.Time, now time.Time) (bool, error) {
	if m.claimTimeTriggerFn != nil {
		return m.claimTimeTriggerFn(ctx, id, previous, now)
	}
	return true, nil
}

func (m *mockRepo) ClaimTimeTriggerFire(_ context.Context, _, _ uuid.UUID, _ string, _ time.Time) (bool, error) {
	return true, nil
}

type mockExecRepo struct {
	createExecFn func(ctx context.Context, e *models.AutomationExecution) error
	updateExecFn func(ctx context.Context, e *models.AutomationExecution) error
	getExecFn    func(ctx context.Context, id, tenantID uuid.UUID) (*models.AutomationExecution, error)
	listExecFn   func(ctx context.Context, f ExecutionFilter) ([]*models.AutomationExecution, int, error)
	cleanupOldFn func(ctx context.Context, olderThan time.Duration) (int64, error)
}

func (m *mockExecRepo) CreateExecution(ctx context.Context, e *models.AutomationExecution) error {
	if m.createExecFn != nil {
		return m.createExecFn(ctx, e)
	}
	return nil
}

func (m *mockExecRepo) UpdateExecution(ctx context.Context, e *models.AutomationExecution) error {
	if m.updateExecFn != nil {
		return m.updateExecFn(ctx, e)
	}
	return nil
}

func (m *mockExecRepo) GetExecution(ctx context.Context, id, tenantID uuid.UUID) (*models.AutomationExecution, error) {
	if m.getExecFn != nil {
		return m.getExecFn(ctx, id, tenantID)
	}
	return nil, nil
}

func (m *mockExecRepo) ListExecutions(ctx context.Context, f ExecutionFilter) ([]*models.AutomationExecution, int, error) {
	if m.listExecFn != nil {
		return m.listExecFn(ctx, f)
	}
	return nil, 0, nil
}

func (m *mockExecRepo) CleanupOldExecutions(ctx context.Context, olderThan time.Duration) (int64, error) {
	if m.cleanupOldFn != nil {
		return m.cleanupOldFn(ctx, olderThan)
	}
	return 0, nil
}

type mockTemplateRepo struct {
	listTemplatesFn func(ctx context.Context, category *string) ([]*models.AutomationTemplate, error)
	getTemplateFn   func(ctx context.Context, id string) (*models.AutomationTemplate, error)
	upsertFn        func(ctx context.Context, t *models.AutomationTemplate) error
}

func (m *mockTemplateRepo) ListTemplates(ctx context.Context, category *string) ([]*models.AutomationTemplate, error) {
	if m.listTemplatesFn != nil {
		return m.listTemplatesFn(ctx, category)
	}
	return nil, nil
}

func (m *mockTemplateRepo) GetTemplate(ctx context.Context, id string) (*models.AutomationTemplate, error) {
	if m.getTemplateFn != nil {
		return m.getTemplateFn(ctx, id)
	}
	return nil, nil
}

func (m *mockTemplateRepo) UpsertTemplate(ctx context.Context, t *models.AutomationTemplate) error {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, t)
	}
	return nil
}

// fakeIdempotencyRepo is a minimal in-memory stand-in for idempotency.Repository,
// mirroring the (tenant_id, key) semantics of the real Postgres-backed store
// closely enough for TriggerWebhook's tests: fresh key -> (nil, nil); same
// key+hash after Complete -> cached record; same key, different hash -> ErrConflict.
type fakeIdempotencyRepo struct {
	records map[string]*idempotency.Record
}

func newFakeIdempotencyRepo() *fakeIdempotencyRepo {
	return &fakeIdempotencyRepo{records: map[string]*idempotency.Record{}}
}

func (f *fakeIdempotencyRepo) recKey(tenantID uuid.UUID, key string) string {
	return tenantID.String() + "|" + key
}

func (f *fakeIdempotencyRepo) Reserve(_ context.Context, key string, tenantID, userID uuid.UUID, method, path, hash string) (*idempotency.Record, error) {
	rk := f.recKey(tenantID, key)
	if rec, ok := f.records[rk]; ok {
		if rec.RequestHash != hash {
			return nil, idempotency.ErrConflict
		}
		if rec.CompletedAt == nil {
			return nil, nil
		}
		return rec, nil
	}
	f.records[rk] = &idempotency.Record{
		Key: key, TenantID: tenantID, UserID: userID,
		Method: method, Path: path, RequestHash: hash,
		CreatedAt: time.Now(), ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	return nil, nil
}

func (f *fakeIdempotencyRepo) Get(_ context.Context, tenantID uuid.UUID, key string) (*idempotency.Record, error) {
	rec, ok := f.records[f.recKey(tenantID, key)]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return rec, nil
}

func (f *fakeIdempotencyRepo) Complete(_ context.Context, tenantID uuid.UUID, key string, status int, body []byte) error {
	rec, ok := f.records[f.recKey(tenantID, key)]
	if !ok {
		return idempotency.ErrKeyMissing
	}
	rec.ResponseStatus = &status
	rec.ResponseBody = body
	now := time.Now()
	rec.CompletedAt = &now
	return nil
}

func (f *fakeIdempotencyRepo) Cleanup(_ context.Context) (int, error) { return 0, nil }

func (f *fakeIdempotencyRepo) CleanupWithLock(_ context.Context, _ int64) (int, error) {
	return 0, nil
}

// fakeExecutor is a minimal Executor stand-in that records the last automation
// and event it was called with, optionally blocking until executed is signalled.
type fakeExecutor struct {
	mu        sync.Mutex
	calls     int
	lastAuto  models.Automation
	lastEvt   models.EventPayload
	executed  chan struct{}
	returnErr error
}

func newFakeExecutor() *fakeExecutor {
	return &fakeExecutor{executed: make(chan struct{}, 16)}
}

func (f *fakeExecutor) Execute(_ context.Context, auto models.Automation, evt models.EventPayload) error {
	f.mu.Lock()
	f.calls++
	f.lastAuto = auto
	f.lastEvt = evt
	f.mu.Unlock()
	f.executed <- struct{}{}
	return f.returnErr
}

func (f *fakeExecutor) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// ============================================================================
// Helpers
// ============================================================================

func triggerGet(t string) (any, bool) {
	if t == "event" || t == "schedule" || t == webhookReceivedTriggerType {
		return "trigger-def", true
	}
	return nil, false
}

func triggerAll() []any {
	return []any{"trigger1", "trigger2"}
}

func actionGet(t string) (any, bool) {
	if t == "send_email" || t == "add_tag" {
		return "action-def", true
	}
	return nil, false
}

func actionAllDefs() []any {
	return []any{"action1", "action2"}
}

func newTestService(repo *mockRepo, execRepo *mockExecRepo, tmplRepo *mockTemplateRepo) *Service {
	return newTestServiceWithWebhookDeps(repo, execRepo, tmplRepo, nil, nil)
}

// newTestServiceWithWebhookDeps additionally lets TriggerWebhook tests inject
// a specific idempotency repo / executor fake; every other test uses
// newTestService, which defaults both to a fresh fake.
func newTestServiceWithWebhookDeps(repo *mockRepo, execRepo *mockExecRepo, tmplRepo *mockTemplateRepo, idemRepo idempotency.Repository, executor Executor) *Service {
	if repo == nil {
		repo = &mockRepo{}
	}
	if execRepo == nil {
		execRepo = &mockExecRepo{}
	}
	if tmplRepo == nil {
		tmplRepo = &mockTemplateRepo{}
	}
	if idemRepo == nil {
		idemRepo = newFakeIdempotencyRepo()
	}
	if executor == nil {
		executor = newFakeExecutor()
	}
	return NewService(repo, execRepo, tmplRepo, triggerGet, triggerAll, actionGet, actionAllDefs, condition.NewEvaluator(), idemRepo, executor)
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// ============================================================================
// Create
// ============================================================================

func TestCreate_Success(t *testing.T) {
	var captured *models.Automation
	repo := &mockRepo{
		createFn: func(_ context.Context, a *models.Automation) error {
			captured = a
			return nil
		},
	}
	svc := newTestService(repo, nil, nil)

	auto := &models.Automation{
		Name:        "Test Automation",
		TriggerType: "event",
		Actions:     mustJSON([]models.ActionConfig{{Type: "send_email"}}),
	}

	err := svc.Create(context.Background(), auto)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, captured.ID)
	assert.False(t, captured.CreatedAt.IsZero())
	assert.False(t, captured.UpdatedAt.IsZero())
	assert.Equal(t, 10, captured.MaxSteps, "default MaxSteps should be 10")
}

// TestCreate_DefaultsNilJSONBFields verifies that omitted trigger_config/
// conditions carry the schema's NOT NULL default instead of a nil
// json.RawMessage, which the repo would otherwise INSERT as an explicit NULL
// and violate the constraint (found via a real-DB write test, see
// tenant_write_test.go).
func TestCreate_DefaultsNilJSONBFields(t *testing.T) {
	var captured *models.Automation
	repo := &mockRepo{
		createFn: func(_ context.Context, a *models.Automation) error {
			captured = a
			return nil
		},
	}
	svc := newTestService(repo, nil, nil)

	auto := &models.Automation{
		Name:        "No Trigger Config Or Conditions",
		TriggerType: "event",
		Actions:     mustJSON([]models.ActionConfig{{Type: "send_email"}}),
	}

	require.Nil(t, auto.TriggerConfig, "test setup: TriggerConfig must start nil")
	require.Nil(t, auto.Conditions, "test setup: Conditions must start nil")

	err := svc.Create(context.Background(), auto)
	require.NoError(t, err)
	assert.Equal(t, json.RawMessage(`{}`), captured.TriggerConfig)
	assert.Equal(t, json.RawMessage(`{}`), captured.Conditions)
}

func TestCreate_UnknownTriggerType(t *testing.T) {
	svc := newTestService(nil, nil, nil)

	auto := &models.Automation{
		Name:        "Bad Trigger",
		TriggerType: "nonexistent",
		Actions:     mustJSON([]models.ActionConfig{{Type: "send_email"}}),
	}

	err := svc.Create(context.Background(), auto)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidCondition))
}

func TestCreate_UnknownActionType(t *testing.T) {
	svc := newTestService(nil, nil, nil)

	auto := &models.Automation{
		Name:        "Bad Action",
		TriggerType: "event",
		Actions:     mustJSON([]models.ActionConfig{{Type: "launch_rocket"}}),
	}

	err := svc.Create(context.Background(), auto)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidAction))
}

func TestCreate_CustomMaxSteps(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ context.Context, a *models.Automation) error {
			return nil
		},
	}
	svc := newTestService(repo, nil, nil)

	auto := &models.Automation{
		Name:        "Custom Steps",
		TriggerType: "event",
		Actions:     mustJSON([]models.ActionConfig{{Type: "send_email"}}),
		MaxSteps:    25,
	}

	err := svc.Create(context.Background(), auto)
	require.NoError(t, err)
	assert.Equal(t, 25, auto.MaxSteps, "custom MaxSteps should be preserved")
}

// ============================================================================
// Update
// ============================================================================

func TestUpdate_Success(t *testing.T) {
	updateCalled := false
	repo := &mockRepo{
		updateFn: func(_ context.Context, a *models.Automation) error {
			updateCalled = true
			return nil
		},
	}
	svc := newTestService(repo, nil, nil)

	auto := &models.Automation{
		ID:          uuid.New(),
		Name:        "Updated Automation",
		TriggerType: "schedule",
		Actions:     mustJSON([]models.ActionConfig{{Type: "add_tag"}}),
	}

	err := svc.Update(context.Background(), auto)
	require.NoError(t, err)
	assert.True(t, updateCalled)
	assert.False(t, auto.UpdatedAt.IsZero())
}

func TestUpdate_ValidationError(t *testing.T) {
	svc := newTestService(nil, nil, nil)

	auto := &models.Automation{
		ID:          uuid.New(),
		Name:        "Bad Update",
		TriggerType: "invalid_trigger",
	}

	err := svc.Update(context.Background(), auto)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidCondition))
}

// ============================================================================
// Delete
// ============================================================================

func TestDelete_Success(t *testing.T) {
	deletedID := uuid.Nil
	repo := &mockRepo{
		deleteFn: func(_ context.Context, id uuid.UUID) error {
			deletedID = id
			return nil
		},
	}
	svc := newTestService(repo, nil, nil)

	targetID := uuid.New()
	err := svc.Delete(context.Background(), targetID, uuid.Nil)
	require.NoError(t, err)
	assert.Equal(t, targetID, deletedID)
}

// ============================================================================
// GetByID
// ============================================================================

func TestGetByID_Success(t *testing.T) {
	expected := &models.Automation{ID: uuid.New(), Name: "Found"}
	repo := &mockRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*models.Automation, error) {
			return expected, nil
		},
	}
	svc := newTestService(repo, nil, nil)

	result, err := svc.GetByID(context.Background(), expected.ID, uuid.Nil)
	require.NoError(t, err)
	assert.Equal(t, expected.Name, result.Name)
}

func TestGetByID_NotFound(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*models.Automation, error) {
			return nil, ErrAutomationNotFound
		},
	}
	svc := newTestService(repo, nil, nil)

	_, err := svc.GetByID(context.Background(), uuid.New(), uuid.Nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAutomationNotFound))
}

// ============================================================================
// List
// ============================================================================

func TestList_Success(t *testing.T) {
	expected := []*models.Automation{{ID: uuid.New(), Name: "Auto1"}}
	repo := &mockRepo{
		listFn: func(_ context.Context, f ListFilter) ([]*models.Automation, int, error) {
			return expected, 1, nil
		},
	}
	svc := newTestService(repo, nil, nil)

	results, count, err := svc.List(context.Background(), ListFilter{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Len(t, results, 1)
}

func TestList_LimitClampZeroToDefault(t *testing.T) {
	var capturedFilter ListFilter
	repo := &mockRepo{
		listFn: func(_ context.Context, f ListFilter) ([]*models.Automation, int, error) {
			capturedFilter = f
			return nil, 0, nil
		},
	}
	svc := newTestService(repo, nil, nil)

	_, _, err := svc.List(context.Background(), ListFilter{Limit: 0})
	require.NoError(t, err)
	assert.Equal(t, 50, capturedFilter.Limit, "zero limit should be clamped to 50")
}

func TestList_LimitClampOverMax(t *testing.T) {
	var capturedFilter ListFilter
	repo := &mockRepo{
		listFn: func(_ context.Context, f ListFilter) ([]*models.Automation, int, error) {
			capturedFilter = f
			return nil, 0, nil
		},
	}
	svc := newTestService(repo, nil, nil)

	_, _, err := svc.List(context.Background(), ListFilter{Limit: 500})
	require.NoError(t, err)
	assert.Equal(t, 100, capturedFilter.Limit, "limit over 100 should be clamped to 100")
}

// ============================================================================
// SetActive
// ============================================================================

func TestSetActive_Success(t *testing.T) {
	var capturedActive bool
	repo := &mockRepo{
		setActiveFn: func(_ context.Context, _ uuid.UUID, active bool) error {
			capturedActive = active
			return nil
		},
	}
	svc := newTestService(repo, nil, nil)

	err := svc.SetActive(context.Background(), uuid.New(), uuid.Nil, true)
	require.NoError(t, err)
	assert.True(t, capturedActive)
}

// ============================================================================
// ListExecutions
// ============================================================================

func TestListExecutions_Success(t *testing.T) {
	expected := []*models.AutomationExecution{{ID: uuid.New(), Status: "completed"}}
	execRepo := &mockExecRepo{
		listExecFn: func(_ context.Context, f ExecutionFilter) ([]*models.AutomationExecution, int, error) {
			return expected, 1, nil
		},
	}
	svc := newTestService(nil, execRepo, nil)

	results, count, err := svc.ListExecutions(context.Background(), ExecutionFilter{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Len(t, results, 1)
}

func TestListExecutions_LimitClampZero(t *testing.T) {
	var capturedFilter ExecutionFilter
	execRepo := &mockExecRepo{
		listExecFn: func(_ context.Context, f ExecutionFilter) ([]*models.AutomationExecution, int, error) {
			capturedFilter = f
			return nil, 0, nil
		},
	}
	svc := newTestService(nil, execRepo, nil)

	_, _, err := svc.ListExecutions(context.Background(), ExecutionFilter{Limit: 0})
	require.NoError(t, err)
	assert.Equal(t, 50, capturedFilter.Limit)
}

func TestListExecutions_LimitClampOver100(t *testing.T) {
	var capturedFilter ExecutionFilter
	execRepo := &mockExecRepo{
		listExecFn: func(_ context.Context, f ExecutionFilter) ([]*models.AutomationExecution, int, error) {
			capturedFilter = f
			return nil, 0, nil
		},
	}
	svc := newTestService(nil, execRepo, nil)

	_, _, err := svc.ListExecutions(context.Background(), ExecutionFilter{Limit: 200})
	require.NoError(t, err)
	assert.Equal(t, 100, capturedFilter.Limit)
}

// ============================================================================
// GetExecution
// ============================================================================

func TestGetExecution_Success(t *testing.T) {
	expected := &models.AutomationExecution{ID: uuid.New(), TenantID: uuid.New(), Status: "completed"}
	execRepo := &mockExecRepo{
		getExecFn: func(_ context.Context, id, tenantID uuid.UUID) (*models.AutomationExecution, error) {
			return expected, nil
		},
	}
	svc := newTestService(nil, execRepo, nil)

	result, err := svc.GetExecution(context.Background(), expected.ID, expected.TenantID)
	require.NoError(t, err)
	assert.Equal(t, expected.ID, result.ID)
}

// ============================================================================
// ListTriggers / ListActions
// ============================================================================

func TestListTriggers_ReturnsCatalog(t *testing.T) {
	svc := newTestService(nil, nil, nil)
	triggers := svc.ListTriggers()
	assert.Len(t, triggers, 2)
	assert.Equal(t, "trigger1", triggers[0])
	assert.Equal(t, "trigger2", triggers[1])
}

func TestListActions_ReturnsCatalog(t *testing.T) {
	svc := newTestService(nil, nil, nil)
	actions := svc.ListActions()
	assert.Len(t, actions, 2)
	assert.Equal(t, "action1", actions[0])
	assert.Equal(t, "action2", actions[1])
}

// ============================================================================
// Templates
// ============================================================================

func TestListTemplates_Success(t *testing.T) {
	expected := []*models.AutomationTemplate{{ID: "tmpl-1", Name: "Welcome Email"}}
	tmplRepo := &mockTemplateRepo{
		listTemplatesFn: func(_ context.Context, _ *string) ([]*models.AutomationTemplate, error) {
			return expected, nil
		},
	}
	svc := newTestService(nil, nil, tmplRepo)

	results, err := svc.ListTemplates(context.Background(), nil)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Welcome Email", results[0].Name)
}

func TestGetTemplate_Success(t *testing.T) {
	expected := &models.AutomationTemplate{ID: "tmpl-1", Name: "Welcome Email"}
	tmplRepo := &mockTemplateRepo{
		getTemplateFn: func(_ context.Context, id string) (*models.AutomationTemplate, error) {
			return expected, nil
		},
	}
	svc := newTestService(nil, nil, tmplRepo)

	result, err := svc.GetTemplate(context.Background(), "tmpl-1")
	require.NoError(t, err)
	assert.Equal(t, "tmpl-1", result.ID)
}

// ============================================================================
// CreateFromTemplate
// ============================================================================

func TestCreateFromTemplate_Success(t *testing.T) {
	tmpl := &models.AutomationTemplate{
		ID:            "tmpl-welcome",
		Name:          "Welcome",
		Description:   "Welcome email automation",
		TriggerType:   "event",
		TriggerConfig: mustJSON(map[string]string{"event": "contact.created"}),
		Conditions:    nil,
		Actions:       mustJSON([]models.ActionConfig{{Type: "send_email"}}),
	}

	var createdAuto *models.Automation
	repo := &mockRepo{
		createFn: func(_ context.Context, a *models.Automation) error {
			createdAuto = a
			return nil
		},
	}
	tmplRepo := &mockTemplateRepo{
		getTemplateFn: func(_ context.Context, id string) (*models.AutomationTemplate, error) {
			return tmpl, nil
		},
	}
	svc := newTestService(repo, nil, tmplRepo)

	ownerID := uuid.New()
	result, err := svc.CreateFromTemplate(context.Background(), "tmpl-welcome", uuid.Nil, ownerID, "My Welcome")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "My Welcome", result.Name)
	assert.Equal(t, tmpl.Description, result.Description)
	assert.Equal(t, models.AutomationScopePersonal, result.Scope)
	assert.Equal(t, ownerID, result.OwnerID)
	assert.False(t, result.IsActive, "should be inactive by default")
	assert.Equal(t, 10, result.MaxSteps)
	require.NotNil(t, result.TemplateID)
	assert.Equal(t, "tmpl-welcome", *result.TemplateID)
	assert.NotNil(t, createdAuto)
}

func TestCreateFromTemplate_TemplateNotFound(t *testing.T) {
	tmplRepo := &mockTemplateRepo{
		getTemplateFn: func(_ context.Context, id string) (*models.AutomationTemplate, error) {
			return nil, nil
		},
	}
	svc := newTestService(nil, nil, tmplRepo)

	_, err := svc.CreateFromTemplate(context.Background(), "nonexistent", uuid.Nil, uuid.New(), "Test")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrTemplateNotFound))
}

// ============================================================================
// GetStats
// ============================================================================

func TestGetStats_Success(t *testing.T) {
	repo := &mockRepo{
		listFn: func(_ context.Context, f ListFilter) ([]*models.Automation, int, error) {
			if f.IsActive != nil && *f.IsActive {
				return nil, 3, nil
			}
			return nil, 10, nil
		},
	}
	execRepo := &mockExecRepo{
		listExecFn: func(_ context.Context, f ExecutionFilter) ([]*models.AutomationExecution, int, error) {
			if f.Status == nil {
				return nil, 0, nil
			}
			switch *f.Status {
			case models.ExecutionStatusCompleted:
				return nil, 15, nil
			case models.ExecutionStatusFailed:
				return nil, 5, nil
			case models.ExecutionStatusSkipped:
				return nil, 2, nil
			}
			return nil, 0, nil
		},
	}
	svc := newTestService(repo, execRepo, nil)

	stats, err := svc.GetStats(context.Background(), uuid.Nil)
	require.NoError(t, err)

	assert.Equal(t, 10, stats.TotalAutomations)
	assert.Equal(t, 3, stats.ActiveAutomations)
	assert.Equal(t, 15, stats.SuccessfulExecs)
	assert.Equal(t, 5, stats.FailedExecs)
	assert.Equal(t, 2, stats.SkippedExecs)
	assert.Equal(t, 22, stats.TotalExecutions)
	// 15/22 * 100 = 68.18...
	assert.InDelta(t, 68.18, stats.SuccessRate, 0.1)
}

// ============================================================================
// TestCondition
// ============================================================================

func TestTestCondition_EmptyModeAlwaysTrue(t *testing.T) {
	svc := newTestService(nil, nil, nil)

	condJSON := mustJSON(models.ConditionConfig{Mode: ""})
	result, err := svc.TestCondition(context.Background(), condJSON, nil)
	require.NoError(t, err)
	assert.True(t, result, "empty mode should always return true")
}

func TestTestCondition_NilConditionJSONAlwaysTrue(t *testing.T) {
	svc := newTestService(nil, nil, nil)

	result, err := svc.TestCondition(context.Background(), nil, nil)
	require.NoError(t, err, "an omitted conditions payload must not fail unmarshal, symmetric to DryRun")
	assert.True(t, result, "no conditions configured should always match, like DryRun treats empty auto.Conditions")
}

func TestTestCondition_SimpleCondition(t *testing.T) {
	svc := newTestService(nil, nil, nil)

	condJSON := mustJSON(models.ConditionConfig{
		Mode: models.ConditionModeSimple,
		Simple: &models.Condition{
			Field:    "status",
			Operator: "equals",
			Value:    "active",
		},
	})
	env := map[string]any{"status": "active"}

	result, err := svc.TestCondition(context.Background(), condJSON, env)
	require.NoError(t, err)
	assert.True(t, result)
}

// ============================================================================
// DryRun
// ============================================================================

func TestDryRun_Success(t *testing.T) {
	autoID := uuid.New()
	auto := &models.Automation{
		ID:          autoID,
		Name:        "Test Auto",
		TriggerType: "event",
		Conditions:  nil,
		Actions:     mustJSON([]models.ActionConfig{{Type: "send_email"}, {Type: "add_tag"}}),
		IsActive:    true,
	}
	repo := &mockRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*models.Automation, error) {
			if id == autoID {
				return auto, nil
			}
			return nil, nil
		},
	}
	svc := newTestService(repo, nil, nil)

	result, err := svc.DryRun(context.Background(), autoID, uuid.Nil, map[string]any{"x": 1})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.ConditionResult, "empty conditions should match")
	assert.Len(t, result.Steps, 2)
	assert.Equal(t, "send_email", result.Steps[0].ActionType)
	assert.True(t, result.Steps[0].WouldRun)
	assert.Equal(t, "add_tag", result.Steps[1].ActionType)
	assert.True(t, result.Steps[1].WouldRun)
}

func TestDryRun_AutomationNotFound(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*models.Automation, error) {
			return nil, nil
		},
	}
	svc := newTestService(repo, nil, nil)

	_, err := svc.DryRun(context.Background(), uuid.New(), uuid.Nil, nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAutomationNotFound))
}

func TestDryRun_UnknownActionMarkedAsError(t *testing.T) {
	autoID := uuid.New()
	auto := &models.Automation{
		ID:          autoID,
		TriggerType: "event",
		Actions:     mustJSON([]models.ActionConfig{{Type: "send_email"}, {Type: "unknown_action"}}),
	}
	repo := &mockRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*models.Automation, error) {
			return auto, nil
		},
	}
	svc := newTestService(repo, nil, nil)

	result, err := svc.DryRun(context.Background(), autoID, uuid.Nil, nil)
	require.NoError(t, err)

	assert.True(t, result.Steps[0].WouldRun)
	assert.Empty(t, result.Steps[0].Error)
	assert.False(t, result.Steps[1].WouldRun)
	assert.Equal(t, "action type not registered", result.Steps[1].Error)
}

// ============================================================================
// Cross-tenant isolation tests (automations)
// ============================================================================

// TestGetByID_CrossTenant verifies that an automation owned by tenant A cannot
// be retrieved by a request carrying tenant B's ID.
func TestGetByID_CrossTenant(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	autoID := uuid.New()

	auto := &models.Automation{
		ID:          autoID,
		TenantID:    tenantA,
		Name:        "Tenant A Automation",
		TriggerType: "event",
	}
	repo := &mockRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*models.Automation, error) {
			if id == autoID {
				return auto, nil
			}
			return nil, ErrAutomationNotFound
		},
	}
	svc := newTestService(repo, nil, nil)

	result, err := svc.GetByID(context.Background(), autoID, tenantB)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrAutomationNotFound)
}

// TestDryRun_CrossTenant verifies that DryRun on an automation owned by
// tenant A is rejected when the request carries tenant B's ID.
func TestDryRun_CrossTenant(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()
	autoID := uuid.New()

	auto := &models.Automation{
		ID:          autoID,
		TenantID:    tenantA,
		TriggerType: "event",
	}
	repo := &mockRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*models.Automation, error) {
			if id == autoID {
				return auto, nil
			}
			return nil, ErrAutomationNotFound
		},
	}
	svc := newTestService(repo, nil, nil)

	result, err := svc.DryRun(context.Background(), autoID, tenantB, nil)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrAutomationNotFound)
}
