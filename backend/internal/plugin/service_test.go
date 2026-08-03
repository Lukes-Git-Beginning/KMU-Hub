package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/plugin/config"
)

// testTenantID is the tenant every service test operates as. CreateManifest
// stamps it on the manifest since 000276, so a bare context.Background() no
// longer gets past the manifest paths.
var testTenantID = uuid.MustParse("11111111-1111-1111-1111-111111111111")

func tenantCtx() context.Context {
	return context.WithValue(context.Background(), middleware.TenantIDKey, testTenantID.String())
}

// ============================================================================
// Mock Repositories
// ============================================================================

type mockManifestRepo struct {
	manifests map[uuid.UUID]*models.PluginManifest
	slugIndex map[string]uuid.UUID
	hasActive map[uuid.UUID]bool
}

func newMockManifestRepo() *mockManifestRepo {
	return &mockManifestRepo{
		manifests: make(map[uuid.UUID]*models.PluginManifest),
		slugIndex: make(map[string]uuid.UUID),
		hasActive: make(map[uuid.UUID]bool),
	}
}

func (m *mockManifestRepo) Create(_ context.Context, manifest *models.PluginManifest) error {
	m.manifests[manifest.ID] = manifest
	m.slugIndex[manifest.Slug] = manifest.ID
	return nil
}

func (m *mockManifestRepo) GetByID(_ context.Context, id uuid.UUID) (*models.PluginManifest, error) {
	manifest, ok := m.manifests[id]
	if !ok {
		return nil, nil
	}
	return manifest, nil
}

func (m *mockManifestRepo) GetBySlug(_ context.Context, slug string) (*models.PluginManifest, error) {
	id, ok := m.slugIndex[slug]
	if !ok {
		return nil, nil
	}
	return m.manifests[id], nil
}

func (m *mockManifestRepo) List(_ context.Context, pluginType string) ([]*models.PluginManifest, error) {
	var result []*models.PluginManifest
	for _, manifest := range m.manifests {
		if pluginType == "" || manifest.PluginType == models.PluginType(pluginType) {
			result = append(result, manifest)
		}
	}
	return result, nil
}

func (m *mockManifestRepo) Delete(_ context.Context, id uuid.UUID) error {
	manifest, ok := m.manifests[id]
	if ok {
		delete(m.slugIndex, manifest.Slug)
		delete(m.manifests, id)
	}
	return nil
}

func (m *mockManifestRepo) HasActiveInstallations(_ context.Context, id uuid.UUID) (bool, error) {
	return m.hasActive[id], nil
}

func (m *mockManifestRepo) GetWASMBinary(_ context.Context, id uuid.UUID) ([]byte, error) {
	manifest, ok := m.manifests[id]
	if !ok {
		return nil, nil
	}
	return manifest.WASMBinary, nil
}

// ---

type mockInstallationRepo struct {
	installations map[uuid.UUID]*models.PluginInstallation
	tenantIndex   map[string]*models.PluginInstallation // key: tenantID-manifestID
}

func newMockInstallationRepo() *mockInstallationRepo {
	return &mockInstallationRepo{
		installations: make(map[uuid.UUID]*models.PluginInstallation),
		tenantIndex:   make(map[string]*models.PluginInstallation),
	}
}

func tenantManifestKey(tenantID, manifestID uuid.UUID) string {
	return tenantID.String() + "-" + manifestID.String()
}

func (m *mockInstallationRepo) Create(_ context.Context, inst *models.PluginInstallation) error {
	m.installations[inst.ID] = inst
	m.tenantIndex[tenantManifestKey(inst.TenantID, inst.ManifestID)] = inst
	return nil
}

func (m *mockInstallationRepo) GetByID(_ context.Context, id uuid.UUID) (*models.PluginInstallation, error) {
	inst, ok := m.installations[id]
	if !ok {
		return nil, nil
	}
	return inst, nil
}

func (m *mockInstallationRepo) GetByTenantAndManifest(_ context.Context, tenantID, manifestID uuid.UUID) (*models.PluginInstallation, error) {
	inst, ok := m.tenantIndex[tenantManifestKey(tenantID, manifestID)]
	if !ok {
		return nil, nil
	}
	return inst, nil
}

func (m *mockInstallationRepo) List(_ context.Context, tenantID uuid.UUID, status string) ([]*models.PluginInstallationWithManifest, error) {
	var result []*models.PluginInstallationWithManifest
	for _, inst := range m.installations {
		if inst.TenantID == tenantID {
			if status == "" || string(inst.Status) == status {
				result = append(result, &models.PluginInstallationWithManifest{
					PluginInstallation: *inst,
				})
			}
		}
	}
	return result, nil
}

func (m *mockInstallationRepo) UpdateStatus(_ context.Context, id uuid.UUID, status models.InstallationStatus, errMsg *string) error {
	inst, ok := m.installations[id]
	if !ok {
		return errors.New("not found")
	}
	inst.Status = status
	inst.ErrorMessage = errMsg
	return nil
}

func (m *mockInstallationRepo) UpdateSettings(_ context.Context, id uuid.UUID, settings []byte) error {
	inst, ok := m.installations[id]
	if !ok {
		return errors.New("not found")
	}
	inst.Settings = settings
	return nil
}

func (m *mockInstallationRepo) ListActiveByHook(_ context.Context, tenantID uuid.UUID, _, _, _ string) ([]*models.PluginInstallation, error) {
	var result []*models.PluginInstallation
	for _, inst := range m.installations {
		if inst.TenantID == tenantID && inst.Status == models.InstallationStatusActive {
			result = append(result, inst)
		}
	}
	return result, nil
}

// ---

type mockPermissionRepo struct {
	permissions map[uuid.UUID][]string // installationID -> permissions
}

func newMockPermissionRepo() *mockPermissionRepo {
	return &mockPermissionRepo{
		permissions: make(map[uuid.UUID][]string),
	}
}

func (m *mockPermissionRepo) Grant(_ context.Context, perm *models.PluginPermission) error {
	m.permissions[perm.InstallationID] = append(m.permissions[perm.InstallationID], perm.Permission)
	return nil
}

func (m *mockPermissionRepo) RevokeAll(_ context.Context, installationID uuid.UUID) error {
	delete(m.permissions, installationID)
	return nil
}

func (m *mockPermissionRepo) ListByInstallation(_ context.Context, installationID uuid.UUID) ([]string, error) {
	return m.permissions[installationID], nil
}

func (m *mockPermissionRepo) HasPermission(_ context.Context, installationID uuid.UUID, permission string) (bool, error) {
	for _, p := range m.permissions[installationID] {
		if p == permission {
			return true, nil
		}
	}
	return false, nil
}

// ---

type mockKVStoreRepo struct {
	store map[string]json.RawMessage // key: installationID-key
}

func newMockKVStoreRepo() *mockKVStoreRepo {
	return &mockKVStoreRepo{
		store: make(map[string]json.RawMessage),
	}
}

func kvKey(installationID uuid.UUID, key string) string {
	return installationID.String() + ":" + key
}

func (m *mockKVStoreRepo) Get(_ context.Context, installationID uuid.UUID, key string) (json.RawMessage, bool, error) {
	val, ok := m.store[kvKey(installationID, key)]
	return val, ok, nil
}

func (m *mockKVStoreRepo) Set(_ context.Context, installationID uuid.UUID, key string, value json.RawMessage) error {
	m.store[kvKey(installationID, key)] = value
	return nil
}

func (m *mockKVStoreRepo) Delete(_ context.Context, installationID uuid.UUID, key string) error {
	delete(m.store, kvKey(installationID, key))
	return nil
}

func (m *mockKVStoreRepo) List(_ context.Context, _ uuid.UUID, _ string) (map[string]json.RawMessage, error) {
	result := make(map[string]json.RawMessage)
	for k, v := range m.store {
		result[k] = v
	}
	return result, nil
}

// ---

type mockExecutionLogRepo struct {
	logs []*models.PluginExecutionLog
}

func newMockExecutionLogRepo() *mockExecutionLogRepo {
	return &mockExecutionLogRepo{}
}

func (m *mockExecutionLogRepo) Create(_ context.Context, log *models.PluginExecutionLog) error {
	m.logs = append(m.logs, log)
	return nil
}

func (m *mockExecutionLogRepo) List(_ context.Context, _ uuid.UUID, _ *uuid.UUID, limit int) ([]*models.PluginExecutionLog, error) {
	if limit > len(m.logs) {
		limit = len(m.logs)
	}
	return m.logs[:limit], nil
}

// ---

type mockValidationRuleRepo struct {
	rules map[uuid.UUID]*models.ValidationRule
}

func newMockValidationRuleRepo() *mockValidationRuleRepo {
	return &mockValidationRuleRepo{
		rules: make(map[uuid.UUID]*models.ValidationRule),
	}
}

func (m *mockValidationRuleRepo) Create(_ context.Context, rule *models.ValidationRule) error {
	m.rules[rule.ID] = rule
	return nil
}

func (m *mockValidationRuleRepo) GetByID(_ context.Context, id uuid.UUID) (*models.ValidationRule, error) {
	rule, ok := m.rules[id]
	if !ok {
		return nil, nil
	}
	return rule, nil
}

func (m *mockValidationRuleRepo) Update(_ context.Context, rule *models.ValidationRule) error {
	m.rules[rule.ID] = rule
	return nil
}

func (m *mockValidationRuleRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.rules, id)
	return nil
}

func (m *mockValidationRuleRepo) List(_ context.Context, tenantID uuid.UUID, entityType string) ([]*models.ValidationRule, error) {
	var result []*models.ValidationRule
	for _, rule := range m.rules {
		if rule.TenantID == tenantID {
			if entityType == "" || rule.EntityType == entityType {
				result = append(result, rule)
			}
		}
	}
	return result, nil
}

func (m *mockValidationRuleRepo) ListEnabled(_ context.Context, tenantID uuid.UUID, entityType string) ([]*models.ValidationRule, error) {
	var result []*models.ValidationRule
	for _, rule := range m.rules {
		if rule.TenantID == tenantID && rule.Enabled {
			if entityType == "" || rule.EntityType == entityType {
				result = append(result, rule)
			}
		}
	}
	return result, nil
}

// ---

type mockWorkflowRuleRepo struct {
	rules map[uuid.UUID]*models.WorkflowRule
}

func newMockWorkflowRuleRepo() *mockWorkflowRuleRepo {
	return &mockWorkflowRuleRepo{
		rules: make(map[uuid.UUID]*models.WorkflowRule),
	}
}

func (m *mockWorkflowRuleRepo) Create(_ context.Context, rule *models.WorkflowRule) error {
	m.rules[rule.ID] = rule
	return nil
}

func (m *mockWorkflowRuleRepo) GetByID(_ context.Context, id uuid.UUID) (*models.WorkflowRule, error) {
	rule, ok := m.rules[id]
	if !ok {
		return nil, nil
	}
	return rule, nil
}

func (m *mockWorkflowRuleRepo) Update(_ context.Context, rule *models.WorkflowRule) error {
	m.rules[rule.ID] = rule
	return nil
}

func (m *mockWorkflowRuleRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.rules, id)
	return nil
}

func (m *mockWorkflowRuleRepo) List(_ context.Context, tenantID uuid.UUID, triggerEvent string) ([]*models.WorkflowRule, error) {
	var result []*models.WorkflowRule
	for _, rule := range m.rules {
		if rule.TenantID == tenantID {
			if triggerEvent == "" || rule.TriggerEvent == triggerEvent {
				result = append(result, rule)
			}
		}
	}
	return result, nil
}

func (m *mockWorkflowRuleRepo) ListEnabled(_ context.Context, tenantID uuid.UUID, triggerEvent string) ([]*models.WorkflowRule, error) {
	var result []*models.WorkflowRule
	for _, rule := range m.rules {
		if rule.TenantID == tenantID && rule.Enabled {
			if triggerEvent == "" || rule.TriggerEvent == triggerEvent {
				result = append(result, rule)
			}
		}
	}
	return result, nil
}

// ---

type mockIndustryTemplateRepo struct {
	templates map[uuid.UUID]*models.IndustryTemplate
}

func newMockIndustryTemplateRepo() *mockIndustryTemplateRepo {
	return &mockIndustryTemplateRepo{
		templates: make(map[uuid.UUID]*models.IndustryTemplate),
	}
}

func (m *mockIndustryTemplateRepo) List(_ context.Context, industry string) ([]*models.IndustryTemplate, error) {
	var result []*models.IndustryTemplate
	for _, tmpl := range m.templates {
		if industry == "" || tmpl.Industry == industry {
			result = append(result, tmpl)
		}
	}
	return result, nil
}

func (m *mockIndustryTemplateRepo) GetByID(_ context.Context, id uuid.UUID) (*models.IndustryTemplate, error) {
	tmpl, ok := m.templates[id]
	if !ok {
		return nil, nil
	}
	return tmpl, nil
}

// ============================================================================
// Compile-time interface assertions
// ============================================================================

var (
	_ ManifestRepo         = (*mockManifestRepo)(nil)
	_ InstallationRepo     = (*mockInstallationRepo)(nil)
	_ PermissionRepo       = (*mockPermissionRepo)(nil)
	_ KVStoreRepo          = (*mockKVStoreRepo)(nil)
	_ ExecutionLogRepo     = (*mockExecutionLogRepo)(nil)
	_ ValidationRuleRepo   = (*mockValidationRuleRepo)(nil)
	_ WorkflowRuleRepo     = (*mockWorkflowRuleRepo)(nil)
	_ IndustryTemplateRepo = (*mockIndustryTemplateRepo)(nil)
)

// ============================================================================
// Test Helpers
// ============================================================================

func newTestService() (*Service, *testRepos) {
	repos := &testRepos{
		manifests:     newMockManifestRepo(),
		installations: newMockInstallationRepo(),
		permissions:   newMockPermissionRepo(),
		kvStore:       newMockKVStoreRepo(),
		executionLog:  newMockExecutionLogRepo(),
		validations:   newMockValidationRuleRepo(),
		workflows:     newMockWorkflowRuleRepo(),
		templates:     newMockIndustryTemplateRepo(),
	}

	svc := NewService(
		repos.manifests,
		repos.installations,
		repos.permissions,
		repos.kvStore,
		repos.executionLog,
		repos.validations,
		repos.workflows,
		repos.templates,
	)

	return svc, repos
}

type testRepos struct {
	manifests     *mockManifestRepo
	installations *mockInstallationRepo
	permissions   *mockPermissionRepo
	kvStore       *mockKVStoreRepo
	executionLog  *mockExecutionLogRepo
	validations   *mockValidationRuleRepo
	workflows     *mockWorkflowRuleRepo
	templates     *mockIndustryTemplateRepo
}

// ============================================================================
// Manifest Tests
// ============================================================================

func TestCreateManifestConfigSuccess(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	m := &models.PluginManifest{
		Slug:       "test-plugin",
		Name:       "Test Plugin",
		Version:    "1.0.0",
		Author:     "test",
		PluginType: models.PluginTypeConfig,
	}

	result, err := svc.CreateManifest(ctx, m)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, result.ID)
	assert.Equal(t, "test-plugin", result.Slug)
	assert.False(t, result.CreatedAt.IsZero())
	assert.False(t, result.UpdatedAt.IsZero())
	assert.Equal(t, json.RawMessage("{}"), result.SettingsSchema)
	assert.Empty(t, result.Permissions)
	assert.Empty(t, result.HookRegistrations)
}

func TestCreateManifestSlugExists(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	m := &models.PluginManifest{
		Slug:       "duplicate",
		Name:       "First",
		PluginType: models.PluginTypeConfig,
	}
	_, err := svc.CreateManifest(ctx, m)
	require.NoError(t, err)

	m2 := &models.PluginManifest{
		Slug:       "duplicate",
		Name:       "Second",
		PluginType: models.PluginTypeConfig,
	}
	_, err = svc.CreateManifest(ctx, m2)
	assert.ErrorIs(t, err, ErrManifestSlugExists)
}

func TestCreateManifestWASMBinaryRequired(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	m := &models.PluginManifest{
		Slug:       "wasm-no-binary",
		Name:       "WASM Plugin",
		PluginType: models.PluginTypeWASM,
	}
	_, err := svc.CreateManifest(ctx, m)
	assert.ErrorIs(t, err, ErrWASMBinaryRequired)
}

func TestCreateManifestWASMBinaryNotAllowedForConfig(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	m := &models.PluginManifest{
		Slug:       "config-with-binary",
		Name:       "Config Plugin",
		PluginType: models.PluginTypeConfig,
		WASMBinary: []byte("fake-wasm-binary"),
	}
	_, err := svc.CreateManifest(ctx, m)
	assert.ErrorIs(t, err, ErrWASMBinaryNotAllowed)
}

func TestCreateManifestWASMSuccess(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	m := &models.PluginManifest{
		Slug:       "wasm-plugin",
		Name:       "WASM Plugin",
		PluginType: models.PluginTypeWASM,
		WASMBinary: []byte("wasm-binary-data"),
	}
	result, err := svc.CreateManifest(ctx, m)
	require.NoError(t, err)
	assert.NotEmpty(t, result.WASMBinaryHash)
}

func TestGetManifestSuccess(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	m := &models.PluginManifest{
		Slug:       "get-me",
		Name:       "Get Me",
		PluginType: models.PluginTypeConfig,
	}
	created, err := svc.CreateManifest(ctx, m)
	require.NoError(t, err)

	result, err := svc.GetManifest(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, result.ID)
	assert.Equal(t, "get-me", result.Slug)
}

func TestGetManifestNotFound(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	_, err := svc.GetManifest(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrManifestNotFound)
}

func TestListManifests(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	_, err := svc.CreateManifest(ctx, &models.PluginManifest{
		Slug: "p1", Name: "P1", PluginType: models.PluginTypeConfig,
	})
	require.NoError(t, err)
	_, err = svc.CreateManifest(ctx, &models.PluginManifest{
		Slug: "p2", Name: "P2", PluginType: models.PluginTypeConfig,
	})
	require.NoError(t, err)

	result, err := svc.ListManifests(ctx, "")
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestDeleteManifestSuccess(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	m, err := svc.CreateManifest(ctx, &models.PluginManifest{
		Slug: "delete-me", Name: "Delete Me", PluginType: models.PluginTypeConfig,
	})
	require.NoError(t, err)

	err = svc.DeleteManifest(ctx, m.ID)
	require.NoError(t, err)

	_, err = svc.GetManifest(ctx, m.ID)
	assert.ErrorIs(t, err, ErrManifestNotFound)
}

func TestDeleteManifestNotFound(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	err := svc.DeleteManifest(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrManifestNotFound)
}

func TestDeleteManifestHasInstallations(t *testing.T) {
	svc, repos := newTestService()
	ctx := tenantCtx()

	m, err := svc.CreateManifest(ctx, &models.PluginManifest{
		Slug: "has-installs", Name: "Has Installs", PluginType: models.PluginTypeConfig,
	})
	require.NoError(t, err)

	repos.manifests.hasActive[m.ID] = true

	err = svc.DeleteManifest(ctx, m.ID)
	assert.ErrorIs(t, err, ErrPluginHasInstallations)
}

func TestCreateManifestStampsCallingTenant(t *testing.T) {
	svc, _ := newTestService()

	m, err := svc.CreateManifest(tenantCtx(), &models.PluginManifest{
		Slug: "owned", Name: "Owned", PluginType: models.PluginTypeConfig,
	})
	require.NoError(t, err)
	require.NotNil(t, m.TenantID)
	assert.Equal(t, testTenantID, *m.TenantID)
}

func TestCreateManifestWithoutTenantContext(t *testing.T) {
	svc, _ := newTestService()

	_, err := svc.CreateManifest(context.Background(), &models.PluginManifest{
		Slug: "no-tenant", Name: "No Tenant", PluginType: models.PluginTypeConfig,
	})
	assert.Error(t, err)
}

// A catalogue manifest (tenant_id NULL) is readable by every tenant and owned
// by none. Without this guard the DELETE would match no rows under the write
// policy and still report success.
func TestDeleteManifestCatalogueImmutable(t *testing.T) {
	svc, repos := newTestService()

	catalogue := &models.PluginManifest{
		ID: uuid.New(), Slug: "catalogue", Name: "Catalogue", PluginType: models.PluginTypeConfig,
	}
	repos.manifests.manifests[catalogue.ID] = catalogue
	repos.manifests.slugIndex[catalogue.Slug] = catalogue.ID

	err := svc.DeleteManifest(tenantCtx(), catalogue.ID)
	assert.ErrorIs(t, err, ErrManifestImmutable)
}

// ============================================================================
// Installation Tests
// ============================================================================

func TestInstallPluginSuccess(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	m, err := svc.CreateManifest(ctx, &models.PluginManifest{
		Slug: "installable", Name: "Installable", PluginType: models.PluginTypeConfig,
	})
	require.NoError(t, err)

	tenantID := uuid.New()
	userID := uuid.New()
	inst, err := svc.InstallPlugin(ctx, tenantID, m.ID, userID)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, inst.ID)
	assert.Equal(t, tenantID, inst.TenantID)
	assert.Equal(t, m.ID, inst.ManifestID)
	assert.Equal(t, models.InstallationStatusPendingApproval, inst.Status)
	assert.Equal(t, userID, inst.InstalledBy)
}

func TestInstallPluginManifestNotFound(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	_, err := svc.InstallPlugin(ctx, uuid.New(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrManifestNotFound)
}

func TestInstallPluginAlreadyInstalled(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	m, err := svc.CreateManifest(ctx, &models.PluginManifest{
		Slug: "double-install", Name: "Double", PluginType: models.PluginTypeConfig,
	})
	require.NoError(t, err)

	tenantID := uuid.New()
	_, err = svc.InstallPlugin(ctx, tenantID, m.ID, uuid.New())
	require.NoError(t, err)

	_, err = svc.InstallPlugin(ctx, tenantID, m.ID, uuid.New())
	assert.ErrorIs(t, err, ErrAlreadyInstalled)
}

func TestInstallPluginReinstallAfterUninstall(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	m, err := svc.CreateManifest(ctx, &models.PluginManifest{
		Slug: "reinstall", Name: "Reinstall", PluginType: models.PluginTypeConfig,
	})
	require.NoError(t, err)

	tenantID := uuid.New()
	inst, err := svc.InstallPlugin(ctx, tenantID, m.ID, uuid.New())
	require.NoError(t, err)

	err = svc.UninstallPlugin(ctx, inst.ID)
	require.NoError(t, err)

	inst2, err := svc.InstallPlugin(ctx, tenantID, m.ID, uuid.New())
	require.NoError(t, err)
	assert.Equal(t, models.InstallationStatusPendingApproval, inst2.Status)
}

func TestGetInstallationSuccess(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	m, err := svc.CreateManifest(ctx, &models.PluginManifest{
		Slug: "get-inst", Name: "Get Inst", PluginType: models.PluginTypeConfig,
	})
	require.NoError(t, err)

	created, err := svc.InstallPlugin(ctx, uuid.New(), m.ID, uuid.New())
	require.NoError(t, err)

	result, err := svc.GetInstallation(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, result.ID)
}

func TestGetInstallationNotFound(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	_, err := svc.GetInstallation(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrInstallationNotFound)
}

func TestEnablePluginSuccess(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	m, err := svc.CreateManifest(ctx, &models.PluginManifest{
		Slug: "enable-me", Name: "Enable Me", PluginType: models.PluginTypeConfig,
	})
	require.NoError(t, err)

	inst, err := svc.InstallPlugin(ctx, uuid.New(), m.ID, uuid.New())
	require.NoError(t, err)

	result, err := svc.EnablePlugin(ctx, inst.ID)
	require.NoError(t, err)
	assert.Equal(t, models.InstallationStatusActive, result.Status)
}

func TestEnablePluginNotFound(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	_, err := svc.EnablePlugin(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrInstallationNotFound)
}

func TestDisablePluginSuccess(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	m, err := svc.CreateManifest(ctx, &models.PluginManifest{
		Slug: "disable-me", Name: "Disable Me", PluginType: models.PluginTypeConfig,
	})
	require.NoError(t, err)

	inst, err := svc.InstallPlugin(ctx, uuid.New(), m.ID, uuid.New())
	require.NoError(t, err)

	result, err := svc.DisablePlugin(ctx, inst.ID)
	require.NoError(t, err)
	assert.Equal(t, models.InstallationStatusDisabled, result.Status)
}

func TestUninstallPluginSuccess(t *testing.T) {
	svc, repos := newTestService()
	ctx := tenantCtx()

	m, err := svc.CreateManifest(ctx, &models.PluginManifest{
		Slug: "uninstall-me", Name: "Uninstall Me", PluginType: models.PluginTypeConfig,
	})
	require.NoError(t, err)

	inst, err := svc.InstallPlugin(ctx, uuid.New(), m.ID, uuid.New())
	require.NoError(t, err)

	// Grant some permissions first
	repos.permissions.permissions[inst.ID] = []string{"read:contacts", "write:contacts"}

	err = svc.UninstallPlugin(ctx, inst.ID)
	require.NoError(t, err)

	// Verify permissions revoked
	perms, err := svc.ListGrantedPermissions(ctx, inst.ID)
	require.NoError(t, err)
	assert.Empty(t, perms)

	// Verify status
	result, err := svc.GetInstallation(ctx, inst.ID)
	require.NoError(t, err)
	assert.Equal(t, models.InstallationStatusUninstalled, result.Status)
}

func TestUninstallPluginNotFound(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	err := svc.UninstallPlugin(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrInstallationNotFound)
}

// ============================================================================
// Permission Tests
// ============================================================================

func TestApprovePermissionsSuccess(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	m, err := svc.CreateManifest(ctx, &models.PluginManifest{
		Slug:        "perm-plugin",
		Name:        "Perm Plugin",
		PluginType:  models.PluginTypeConfig,
		Permissions: []string{"read:contacts", "write:contacts"},
	})
	require.NoError(t, err)

	inst, err := svc.InstallPlugin(ctx, uuid.New(), m.ID, uuid.New())
	require.NoError(t, err)

	err = svc.ApprovePermissions(ctx, inst.ID, []string{"read:contacts", "write:contacts"}, uuid.New())
	require.NoError(t, err)

	granted, err := svc.ListGrantedPermissions(ctx, inst.ID)
	require.NoError(t, err)
	assert.Len(t, granted, 2)
	assert.Contains(t, granted, "read:contacts")
	assert.Contains(t, granted, "write:contacts")
}

func TestApprovePermissionsNotDeclared(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	m, err := svc.CreateManifest(ctx, &models.PluginManifest{
		Slug:        "limited-perm",
		Name:        "Limited Perm",
		PluginType:  models.PluginTypeConfig,
		Permissions: []string{"read:contacts"},
	})
	require.NoError(t, err)

	inst, err := svc.InstallPlugin(ctx, uuid.New(), m.ID, uuid.New())
	require.NoError(t, err)

	err = svc.ApprovePermissions(ctx, inst.ID, []string{"write:deals"}, uuid.New())
	assert.ErrorIs(t, err, ErrPermissionNotDeclared)
}

func TestApprovePermissionsInstallationNotFound(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	err := svc.ApprovePermissions(ctx, uuid.New(), []string{"read:contacts"}, uuid.New())
	assert.ErrorIs(t, err, ErrInstallationNotFound)
}

func TestHasPermission(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	m, err := svc.CreateManifest(ctx, &models.PluginManifest{
		Slug:        "has-perm",
		Name:        "Has Perm",
		PluginType:  models.PluginTypeConfig,
		Permissions: []string{"read:contacts"},
	})
	require.NoError(t, err)

	inst, err := svc.InstallPlugin(ctx, uuid.New(), m.ID, uuid.New())
	require.NoError(t, err)

	err = svc.ApprovePermissions(ctx, inst.ID, []string{"read:contacts"}, uuid.New())
	require.NoError(t, err)

	has, err := svc.HasPermission(ctx, inst.ID, "read:contacts")
	require.NoError(t, err)
	assert.True(t, has)

	has, err = svc.HasPermission(ctx, inst.ID, "write:contacts")
	require.NoError(t, err)
	assert.False(t, has)
}

// ============================================================================
// Settings Tests
// ============================================================================

func TestUpdatePluginSettingsSuccess(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"api_key": {"type": "string"},
			"enabled": {"type": "boolean"}
		},
		"required": ["api_key"]
	}`)

	m, err := svc.CreateManifest(ctx, &models.PluginManifest{
		Slug:           "settings-plugin",
		Name:           "Settings Plugin",
		PluginType:     models.PluginTypeConfig,
		SettingsSchema: schema,
	})
	require.NoError(t, err)

	inst, err := svc.InstallPlugin(ctx, uuid.New(), m.ID, uuid.New())
	require.NoError(t, err)

	settings := json.RawMessage(`{"api_key": "test-key-123", "enabled": true}`)
	err = svc.UpdatePluginSettings(ctx, inst.ID, settings)
	require.NoError(t, err)

	result, err := svc.GetPluginSettings(ctx, inst.ID)
	require.NoError(t, err)
	assert.JSONEq(t, `{"api_key": "test-key-123", "enabled": true}`, string(result))
}

func TestUpdatePluginSettingsInvalid(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"api_key": {"type": "string"}
		},
		"required": ["api_key"]
	}`)

	m, err := svc.CreateManifest(ctx, &models.PluginManifest{
		Slug:           "invalid-settings",
		Name:           "Invalid Settings",
		PluginType:     models.PluginTypeConfig,
		SettingsSchema: schema,
	})
	require.NoError(t, err)

	inst, err := svc.InstallPlugin(ctx, uuid.New(), m.ID, uuid.New())
	require.NoError(t, err)

	// Missing required field
	settings := json.RawMessage(`{"other_field": "value"}`)
	err = svc.UpdatePluginSettings(ctx, inst.ID, settings)
	assert.ErrorIs(t, err, ErrInvalidSettings)
}

func TestGetPluginSettingsNotFound(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	_, err := svc.GetPluginSettings(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrInstallationNotFound)
}

func TestGetPluginSettingsSchema(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	schema := json.RawMessage(`{"type": "object", "properties": {"key": {"type": "string"}}}`)
	m, err := svc.CreateManifest(ctx, &models.PluginManifest{
		Slug:           "schema-test",
		Name:           "Schema Test",
		PluginType:     models.PluginTypeConfig,
		SettingsSchema: schema,
	})
	require.NoError(t, err)

	inst, err := svc.InstallPlugin(ctx, uuid.New(), m.ID, uuid.New())
	require.NoError(t, err)

	result, err := svc.GetPluginSettingsSchema(ctx, inst.ID)
	require.NoError(t, err)
	assert.JSONEq(t, string(schema), string(result))
}

// ============================================================================
// KV Store Tests
// ============================================================================

func TestKVSetAndGet(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()
	instID := uuid.New()

	value := json.RawMessage(`{"foo": "bar"}`)
	err := svc.KVSet(ctx, instID, "my-key", value)
	require.NoError(t, err)

	result, found, err := svc.KVGet(ctx, instID, "my-key")
	require.NoError(t, err)
	assert.True(t, found)
	assert.JSONEq(t, `{"foo": "bar"}`, string(result))
}

func TestKVGetNotFound(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	_, found, err := svc.KVGet(ctx, uuid.New(), "nonexistent")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestKVDelete(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()
	instID := uuid.New()

	err := svc.KVSet(ctx, instID, "delete-key", json.RawMessage(`"value"`))
	require.NoError(t, err)

	err = svc.KVDelete(ctx, instID, "delete-key")
	require.NoError(t, err)

	_, found, err := svc.KVGet(ctx, instID, "delete-key")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestKVList(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()
	instID := uuid.New()

	err := svc.KVSet(ctx, instID, "key1", json.RawMessage(`"v1"`))
	require.NoError(t, err)
	err = svc.KVSet(ctx, instID, "key2", json.RawMessage(`"v2"`))
	require.NoError(t, err)

	result, err := svc.KVList(ctx, instID, "")
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

// ============================================================================
// Validation Rule Tests
// ============================================================================

func TestCreateValidationRuleSuccess(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	rule := &models.ValidationRule{
		TenantID:     uuid.New(),
		Name:         "email-format",
		Description:  "Validate email format",
		EntityType:   "contact",
		FieldName:    "email",
		RuleType:     models.ValidationRuleTypeFormat,
		RuleConfig:   json.RawMessage(`{"format": "email", "required": true}`),
		ErrorMessage: "Invalid email format",
		Priority:     10,
	}

	result, err := svc.CreateValidationRule(ctx, rule)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, result.ID)
	assert.True(t, result.Enabled)
	assert.False(t, result.CreatedAt.IsZero())
}

func TestListValidationRules(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()
	tenantID := uuid.New()

	_, err := svc.CreateValidationRule(ctx, &models.ValidationRule{
		TenantID:   tenantID,
		Name:       "rule1",
		EntityType: "contact",
		RuleConfig: json.RawMessage(`{}`),
	})
	require.NoError(t, err)

	_, err = svc.CreateValidationRule(ctx, &models.ValidationRule{
		TenantID:   tenantID,
		Name:       "rule2",
		EntityType: "contact",
		RuleConfig: json.RawMessage(`{}`),
	})
	require.NoError(t, err)

	result, err := svc.ListValidationRules(ctx, tenantID, "contact")
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestDeleteValidationRuleSuccess(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	rule, err := svc.CreateValidationRule(ctx, &models.ValidationRule{
		TenantID:   uuid.New(),
		Name:       "delete-rule",
		EntityType: "contact",
		RuleConfig: json.RawMessage(`{}`),
	})
	require.NoError(t, err)

	err = svc.DeleteValidationRule(ctx, rule.ID)
	require.NoError(t, err)
}

func TestDeleteValidationRuleNotFound(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	err := svc.DeleteValidationRule(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrValidationRuleNotFound)
}

func TestUpdateValidationRule(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	rule, err := svc.CreateValidationRule(ctx, &models.ValidationRule{
		TenantID:     uuid.New(),
		Name:         "original",
		Description:  "Original description",
		EntityType:   "contact",
		RuleConfig:   json.RawMessage(`{}`),
		ErrorMessage: "Original error",
		Priority:     1,
	})
	require.NoError(t, err)

	updated, err := svc.UpdateValidationRule(ctx, &models.ValidationRule{
		ID:           rule.ID,
		Name:         "updated",
		Description:  "Updated description",
		RuleConfig:   json.RawMessage(`{"new": true}`),
		ErrorMessage: "Updated error",
		Priority:     5,
		Enabled:      false,
	})
	require.NoError(t, err)
	assert.Equal(t, "updated", updated.Name)
	assert.Equal(t, "Updated description", updated.Description)
	assert.Equal(t, 5, updated.Priority)
	assert.False(t, updated.Enabled)
}

func TestUpdateValidationRuleNotFound(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	_, err := svc.UpdateValidationRule(ctx, &models.ValidationRule{
		ID:   uuid.New(),
		Name: "nonexistent",
	})
	assert.ErrorIs(t, err, ErrValidationRuleNotFound)
}

func TestValidateEntity(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()
	tenantID := uuid.New()

	_, err := svc.CreateValidationRule(ctx, &models.ValidationRule{
		TenantID:     tenantID,
		Name:         "email-format",
		EntityType:   "contact",
		FieldName:    "email",
		RuleType:     models.ValidationRuleTypeFormat,
		RuleConfig:   json.RawMessage(`{"format": "email", "required": true}`),
		ErrorMessage: "Invalid email",
	})
	require.NoError(t, err)

	// Valid data
	errors := svc.ValidateEntity(ctx, tenantID, "contact", map[string]any{
		"email": "user@example.com",
	})
	assert.Empty(t, errors)

	// Invalid data
	errors = svc.ValidateEntity(ctx, tenantID, "contact", map[string]any{
		"email": "not-an-email",
	})
	assert.NotEmpty(t, errors)
	assert.Equal(t, "email", errors[0].Field)
}

// ============================================================================
// Workflow Rule Tests
// ============================================================================

func TestCreateWorkflowRuleSuccess(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	rule := &models.WorkflowRule{
		TenantID:     uuid.New(),
		Name:         "deal-created-notify",
		Description:  "Notify on deal creation",
		TriggerEvent: "deal.created",
		Conditions:   json.RawMessage(`[{"field": "amount", "operator": "gt", "value": 1000}]`),
		Actions:      json.RawMessage(`[{"type": "send_notification", "config": {"message": "Big deal!"}}]`),
		Priority:     1,
	}

	result, err := svc.CreateWorkflowRule(ctx, rule)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, result.ID)
	assert.True(t, result.Enabled)
}

func TestListWorkflowRules(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()
	tenantID := uuid.New()

	_, err := svc.CreateWorkflowRule(ctx, &models.WorkflowRule{
		TenantID:     tenantID,
		Name:         "wf1",
		TriggerEvent: "deal.created",
		Conditions:   json.RawMessage(`[]`),
		Actions:      json.RawMessage(`[]`),
	})
	require.NoError(t, err)

	result, err := svc.ListWorkflowRules(ctx, tenantID, "deal.created")
	require.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestDeleteWorkflowRuleSuccess(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	rule, err := svc.CreateWorkflowRule(ctx, &models.WorkflowRule{
		TenantID:     uuid.New(),
		Name:         "delete-wf",
		TriggerEvent: "deal.created",
		Conditions:   json.RawMessage(`[]`),
		Actions:      json.RawMessage(`[]`),
	})
	require.NoError(t, err)

	err = svc.DeleteWorkflowRule(ctx, rule.ID)
	require.NoError(t, err)
}

func TestDeleteWorkflowRuleNotFound(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	err := svc.DeleteWorkflowRule(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrWorkflowRuleNotFound)
}

func TestUpdateWorkflowRule(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	rule, err := svc.CreateWorkflowRule(ctx, &models.WorkflowRule{
		TenantID:     uuid.New(),
		Name:         "original-wf",
		TriggerEvent: "deal.created",
		Conditions:   json.RawMessage(`[]`),
		Actions:      json.RawMessage(`[]`),
		Priority:     1,
	})
	require.NoError(t, err)

	updated, err := svc.UpdateWorkflowRule(ctx, &models.WorkflowRule{
		ID:          rule.ID,
		Name:        "updated-wf",
		Description: "Updated",
		Conditions:  json.RawMessage(`[{"field": "status", "operator": "eq", "value": "won"}]`),
		Actions:     json.RawMessage(`[{"type": "log", "config": {}}]`),
		Priority:    5,
		Enabled:     false,
	})
	require.NoError(t, err)
	assert.Equal(t, "updated-wf", updated.Name)
	assert.Equal(t, 5, updated.Priority)
	assert.False(t, updated.Enabled)
}

func TestUpdateWorkflowRuleNotFound(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	_, err := svc.UpdateWorkflowRule(ctx, &models.WorkflowRule{
		ID:   uuid.New(),
		Name: "nonexistent",
	})
	assert.ErrorIs(t, err, ErrWorkflowRuleNotFound)
}

func TestEvaluateWorkflows(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()
	tenantID := uuid.New()

	_, err := svc.CreateWorkflowRule(ctx, &models.WorkflowRule{
		TenantID:     tenantID,
		Name:         "big-deal-alert",
		TriggerEvent: "deal.created",
		Conditions:   json.RawMessage(`[{"field": "amount", "operator": "gt", "value": 1000}]`),
		Actions:      json.RawMessage(`[{"type": "send_notification", "config": {"message": "Big deal!"}}]`),
		Priority:     1,
	})
	require.NoError(t, err)

	// Trigger with matching data
	triggered := svc.EvaluateWorkflows(ctx, tenantID, "deal.created", map[string]any{
		"amount": float64(5000),
	})
	assert.Len(t, triggered, 1)
	assert.Equal(t, "big-deal-alert", triggered[0].RuleName)
	assert.Equal(t, "send_notification", triggered[0].Action.Type)

	// Trigger with non-matching data
	triggered = svc.EvaluateWorkflows(ctx, tenantID, "deal.created", map[string]any{
		"amount": float64(500),
	})
	assert.Empty(t, triggered)
}

// ============================================================================
// Execution Log Tests
// ============================================================================

func TestLogExecution(t *testing.T) {
	svc, repos := newTestService()
	ctx := tenantCtx()

	log := &models.PluginExecutionLog{
		InstallationID: uuid.New(),
		HookType:       "before_create",
		Module:         "crm",
		EntityType:     "contact",
		DurationMs:     42,
		Status:         models.ExecutionStatusSuccess,
	}

	err := svc.LogExecution(ctx, log)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, log.ID)
	assert.False(t, log.CreatedAt.IsZero())
	assert.Len(t, repos.executionLog.logs, 1)
}

func TestListExecutionLogs(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()
	tenantID := uuid.New()

	for i := 0; i < 3; i++ {
		err := svc.LogExecution(ctx, &models.PluginExecutionLog{
			InstallationID: uuid.New(),
			HookType:       "before_create",
			Module:         "crm",
			EntityType:     "contact",
			DurationMs:     10,
			Status:         models.ExecutionStatusSuccess,
		})
		require.NoError(t, err)
	}

	result, err := svc.ListExecutionLogs(ctx, tenantID, nil, 2)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

// ============================================================================
// Industry Template Tests
// ============================================================================

func TestListIndustryTemplates(t *testing.T) {
	svc, repos := newTestService()
	ctx := tenantCtx()

	id := uuid.New()
	repos.templates.templates[id] = &models.IndustryTemplate{
		ID:       id,
		Slug:     "handwerk",
		Name:     "Handwerk",
		Industry: "handwerk",
	}

	result, err := svc.ListIndustryTemplates(ctx, "handwerk")
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "handwerk", result[0].Slug)
}

func TestApplyIndustryTemplateSuccess(t *testing.T) {
	svc, repos := newTestService()
	ctx := tenantCtx()
	tenantID := uuid.New()

	tmplID := uuid.New()
	repos.templates.templates[tmplID] = &models.IndustryTemplate{
		ID:       tmplID,
		Slug:     "handwerk",
		Name:     "Handwerk Template",
		Industry: "handwerk",
		ValidationRules: json.RawMessage(`[{
			"name": "phone-format",
			"entity_type": "contact",
			"field_name": "phone",
			"rule_type": "format",
			"rule_config": "{\"format\": \"phone\", \"required\": false}",
			"error_message": "Invalid phone"
		}]`),
		WorkflowRules: json.RawMessage(`[{
			"name": "auto-assign",
			"trigger_event": "contact.created",
			"conditions": "[]",
			"actions": "[{\"type\": \"log\", \"config\": {}}]"
		}]`),
	}

	err := svc.ApplyIndustryTemplate(ctx, tenantID, tmplID, uuid.New())
	require.NoError(t, err)

	// Verify validation rules were created
	valRules, err := svc.ListValidationRules(ctx, tenantID, "")
	require.NoError(t, err)
	assert.NotEmpty(t, valRules)
}

func TestApplyIndustryTemplateNotFound(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	err := svc.ApplyIndustryTemplate(ctx, uuid.New(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrTemplateNotFound)
}

// ============================================================================
// Hook / Manifest for Installation Tests
// ============================================================================

func TestGetManifestForInstallation(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	m, err := svc.CreateManifest(ctx, &models.PluginManifest{
		Slug: "manifest-lookup", Name: "Manifest Lookup", PluginType: models.PluginTypeConfig,
	})
	require.NoError(t, err)

	inst, err := svc.InstallPlugin(ctx, uuid.New(), m.ID, uuid.New())
	require.NoError(t, err)

	result, err := svc.GetManifestForInstallation(ctx, inst.ID)
	require.NoError(t, err)
	assert.Equal(t, m.ID, result.ID)
}

func TestGetManifestForInstallationNotFound(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	_, err := svc.GetManifestForInstallation(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrInstallationNotFound)
}

func TestGetWASMBinary(t *testing.T) {
	svc, _ := newTestService()
	ctx := tenantCtx()

	binary := []byte("wasm-content-here")
	m, err := svc.CreateManifest(ctx, &models.PluginManifest{
		Slug:       "wasm-binary-test",
		Name:       "WASM Binary Test",
		PluginType: models.PluginTypeWASM,
		WASMBinary: binary,
	})
	require.NoError(t, err)

	result, err := svc.GetWASMBinary(ctx, m.ID)
	require.NoError(t, err)
	assert.Equal(t, binary, result)
}

// ============================================================================
// ListInstallations Tests
// ============================================================================

func TestListInstallationsEnrichesPermissions(t *testing.T) {
	svc, repos := newTestService()
	ctx := tenantCtx()

	m, err := svc.CreateManifest(ctx, &models.PluginManifest{
		Slug:        "list-inst",
		Name:        "List Inst",
		PluginType:  models.PluginTypeConfig,
		Permissions: []string{"read:contacts"},
	})
	require.NoError(t, err)

	tenantID := uuid.New()
	inst, err := svc.InstallPlugin(ctx, tenantID, m.ID, uuid.New())
	require.NoError(t, err)

	repos.permissions.permissions[inst.ID] = []string{"read:contacts"}

	result, err := svc.ListInstallations(ctx, tenantID, "")
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Contains(t, result[0].Granted, "read:contacts")
}

// ============================================================================
// Compile-time: verify real config engines are usable
// ============================================================================

func TestConfigEnginesIntegration(t *testing.T) {
	// Verify config engines work with real implementations
	sv := config.NewSchemaValidator()
	result := sv.Validate(json.RawMessage(`{}`), json.RawMessage(`{}`))
	assert.True(t, result.Valid)

	ve := config.NewValidationEngine()
	fieldErrors := ve.ValidateEntity(nil, nil)
	assert.Empty(t, fieldErrors)

	we := config.NewWorkflowEngine()
	triggered := we.EvaluateWorkflows(nil, "test", nil)
	assert.Empty(t, triggered)
}
