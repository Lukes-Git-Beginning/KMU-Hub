package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/plugin"
	pluginv1 "github.com/kmuhub/kmuhub/proto/plugin/v1"
)

// ============================================================================
// Stub repositories
//
// internal/plugin/service_test.go already has mocks for each of these eight
// interfaces, but they are unexported types in package plugin and cannot be
// reused from package server. Each interface below declares methods with the
// same names (Create, GetByID, List, Delete, ...) but different signatures,
// so a single struct cannot implement more than one of them - hence eight
// small stub types instead of one combined stub, unlike fuhrpark/inventar
// which each have exactly one Repository interface.
// ============================================================================

type stubPluginManifestRepo struct {
	byID              map[uuid.UUID]*models.PluginManifest
	hasActiveInstalls map[uuid.UUID]bool
	listErr           error
}

func newStubPluginManifestRepo() *stubPluginManifestRepo {
	return &stubPluginManifestRepo{
		byID:              make(map[uuid.UUID]*models.PluginManifest),
		hasActiveInstalls: make(map[uuid.UUID]bool),
	}
}

func (r *stubPluginManifestRepo) Create(_ context.Context, m *models.PluginManifest) error {
	r.byID[m.ID] = m
	return nil
}

func (r *stubPluginManifestRepo) GetByID(_ context.Context, id uuid.UUID) (*models.PluginManifest, error) {
	return r.byID[id], nil
}

func (r *stubPluginManifestRepo) GetBySlug(_ context.Context, slug string) (*models.PluginManifest, error) {
	for _, m := range r.byID {
		if m.Slug == slug {
			return m, nil
		}
	}
	return nil, nil
}

func (r *stubPluginManifestRepo) List(_ context.Context, pluginType string) ([]*models.PluginManifest, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	slugs := make([]string, 0, len(r.byID))
	bySlug := make(map[string]*models.PluginManifest, len(r.byID))
	for _, m := range r.byID {
		if pluginType != "" && string(m.PluginType) != pluginType {
			continue
		}
		slugs = append(slugs, m.Slug)
		bySlug[m.Slug] = m
	}
	sort.Strings(slugs)
	result := make([]*models.PluginManifest, 0, len(slugs))
	for _, s := range slugs {
		result = append(result, bySlug[s])
	}
	return result, nil
}

func (r *stubPluginManifestRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.byID, id)
	return nil
}

func (r *stubPluginManifestRepo) HasActiveInstallations(_ context.Context, id uuid.UUID) (bool, error) {
	return r.hasActiveInstalls[id], nil
}

func (r *stubPluginManifestRepo) GetWASMBinary(_ context.Context, id uuid.UUID) ([]byte, error) {
	if m, ok := r.byID[id]; ok {
		return m.WASMBinary, nil
	}
	return nil, nil
}

type stubPluginInstallationRepo struct {
	byID             map[uuid.UUID]*models.PluginInstallation
	listWithManifest []*models.PluginInstallationWithManifest
	listErr          error
	activeByHook     []*models.PluginInstallation
	hookErr          error
}

func newStubPluginInstallationRepo() *stubPluginInstallationRepo {
	return &stubPluginInstallationRepo{byID: make(map[uuid.UUID]*models.PluginInstallation)}
}

func (r *stubPluginInstallationRepo) Create(_ context.Context, inst *models.PluginInstallation) error {
	r.byID[inst.ID] = inst
	return nil
}

func (r *stubPluginInstallationRepo) GetByID(_ context.Context, id uuid.UUID) (*models.PluginInstallation, error) {
	return r.byID[id], nil
}

func (r *stubPluginInstallationRepo) GetByTenantAndManifest(_ context.Context, tenantID, manifestID uuid.UUID) (*models.PluginInstallation, error) {
	for _, inst := range r.byID {
		if inst.TenantID == tenantID && inst.ManifestID == manifestID {
			return inst, nil
		}
	}
	return nil, nil
}

func (r *stubPluginInstallationRepo) List(_ context.Context, _ uuid.UUID, _ string) ([]*models.PluginInstallationWithManifest, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	if r.listWithManifest != nil {
		return r.listWithManifest, nil
	}
	return []*models.PluginInstallationWithManifest{}, nil
}

func (r *stubPluginInstallationRepo) UpdateStatus(_ context.Context, id uuid.UUID, status models.InstallationStatus, errorMsg *string) error {
	inst, ok := r.byID[id]
	if !ok {
		return nil
	}
	inst.Status = status
	inst.ErrorMessage = errorMsg
	return nil
}

func (r *stubPluginInstallationRepo) UpdateSettings(_ context.Context, id uuid.UUID, settings []byte) error {
	inst, ok := r.byID[id]
	if !ok {
		return nil
	}
	inst.Settings = settings
	return nil
}

func (r *stubPluginInstallationRepo) ListActiveByHook(_ context.Context, _ uuid.UUID, _, _, _ string) ([]*models.PluginInstallation, error) {
	if r.hookErr != nil {
		return nil, r.hookErr
	}
	return r.activeByHook, nil
}

type stubPluginPermissionRepo struct {
	granted map[uuid.UUID][]string
	listErr error
	hasErr  error
}

func newStubPluginPermissionRepo() *stubPluginPermissionRepo {
	return &stubPluginPermissionRepo{granted: make(map[uuid.UUID][]string)}
}

func (r *stubPluginPermissionRepo) Grant(_ context.Context, perm *models.PluginPermission) error {
	r.granted[perm.InstallationID] = append(r.granted[perm.InstallationID], perm.Permission)
	return nil
}

func (r *stubPluginPermissionRepo) RevokeAll(_ context.Context, installationID uuid.UUID) error {
	delete(r.granted, installationID)
	return nil
}

func (r *stubPluginPermissionRepo) ListByInstallation(_ context.Context, installationID uuid.UUID) ([]string, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.granted[installationID], nil
}

func (r *stubPluginPermissionRepo) HasPermission(_ context.Context, installationID uuid.UUID, permission string) (bool, error) {
	if r.hasErr != nil {
		return false, r.hasErr
	}
	return slices.Contains(r.granted[installationID], permission), nil
}

type stubPluginKVRepo struct {
	store   map[uuid.UUID]map[string]json.RawMessage
	getErr  error
	setErr  error
	delErr  error
	listErr error
}

func newStubPluginKVRepo() *stubPluginKVRepo {
	return &stubPluginKVRepo{store: make(map[uuid.UUID]map[string]json.RawMessage)}
}

func (r *stubPluginKVRepo) Get(_ context.Context, installationID uuid.UUID, key string) (json.RawMessage, bool, error) {
	if r.getErr != nil {
		return nil, false, r.getErr
	}
	m, ok := r.store[installationID]
	if !ok {
		return nil, false, nil
	}
	v, ok := m[key]
	return v, ok, nil
}

func (r *stubPluginKVRepo) Set(_ context.Context, installationID uuid.UUID, key string, value json.RawMessage) error {
	if r.setErr != nil {
		return r.setErr
	}
	if r.store[installationID] == nil {
		r.store[installationID] = make(map[string]json.RawMessage)
	}
	r.store[installationID][key] = value
	return nil
}

func (r *stubPluginKVRepo) Delete(_ context.Context, installationID uuid.UUID, key string) error {
	if r.delErr != nil {
		return r.delErr
	}
	delete(r.store[installationID], key)
	return nil
}

func (r *stubPluginKVRepo) List(_ context.Context, installationID uuid.UUID, keyPrefix string) (map[string]json.RawMessage, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	result := make(map[string]json.RawMessage)
	for k, v := range r.store[installationID] {
		if keyPrefix == "" || strings.HasPrefix(k, keyPrefix) {
			result[k] = v
		}
	}
	return result, nil
}

type stubPluginExecutionLogRepo struct {
	logs    []*models.PluginExecutionLog
	listErr error
}

func (r *stubPluginExecutionLogRepo) Create(_ context.Context, log *models.PluginExecutionLog) error {
	r.logs = append(r.logs, log)
	return nil
}

func (r *stubPluginExecutionLogRepo) List(_ context.Context, _ uuid.UUID, _ *uuid.UUID, _ int) ([]*models.PluginExecutionLog, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.logs, nil
}

type stubPluginValidationRuleRepo struct {
	byID      map[uuid.UUID]*models.ValidationRule
	createErr error
	listErr   error
}

func newStubPluginValidationRuleRepo() *stubPluginValidationRuleRepo {
	return &stubPluginValidationRuleRepo{byID: make(map[uuid.UUID]*models.ValidationRule)}
}

func (r *stubPluginValidationRuleRepo) Create(_ context.Context, rule *models.ValidationRule) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.byID[rule.ID] = rule
	return nil
}

func (r *stubPluginValidationRuleRepo) GetByID(_ context.Context, id uuid.UUID) (*models.ValidationRule, error) {
	return r.byID[id], nil
}

func (r *stubPluginValidationRuleRepo) Update(_ context.Context, rule *models.ValidationRule) error {
	r.byID[rule.ID] = rule
	return nil
}

func (r *stubPluginValidationRuleRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.byID, id)
	return nil
}

func (r *stubPluginValidationRuleRepo) List(_ context.Context, tenantID uuid.UUID, entityType string) ([]*models.ValidationRule, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	var result []*models.ValidationRule
	for _, rule := range r.byID {
		if rule.TenantID == tenantID && (entityType == "" || rule.EntityType == entityType) {
			result = append(result, rule)
		}
	}
	return result, nil
}

func (r *stubPluginValidationRuleRepo) ListEnabled(ctx context.Context, tenantID uuid.UUID, entityType string) ([]*models.ValidationRule, error) {
	rules, err := r.List(ctx, tenantID, entityType)
	if err != nil {
		return nil, err
	}
	var enabled []*models.ValidationRule
	for _, rule := range rules {
		if rule.Enabled {
			enabled = append(enabled, rule)
		}
	}
	return enabled, nil
}

type stubPluginWorkflowRuleRepo struct {
	byID      map[uuid.UUID]*models.WorkflowRule
	createErr error
	listErr   error
}

func newStubPluginWorkflowRuleRepo() *stubPluginWorkflowRuleRepo {
	return &stubPluginWorkflowRuleRepo{byID: make(map[uuid.UUID]*models.WorkflowRule)}
}

func (r *stubPluginWorkflowRuleRepo) Create(_ context.Context, rule *models.WorkflowRule) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.byID[rule.ID] = rule
	return nil
}

func (r *stubPluginWorkflowRuleRepo) GetByID(_ context.Context, id uuid.UUID) (*models.WorkflowRule, error) {
	return r.byID[id], nil
}

func (r *stubPluginWorkflowRuleRepo) Update(_ context.Context, rule *models.WorkflowRule) error {
	r.byID[rule.ID] = rule
	return nil
}

func (r *stubPluginWorkflowRuleRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.byID, id)
	return nil
}

func (r *stubPluginWorkflowRuleRepo) List(_ context.Context, tenantID uuid.UUID, triggerEvent string) ([]*models.WorkflowRule, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	var result []*models.WorkflowRule
	for _, rule := range r.byID {
		if rule.TenantID == tenantID && (triggerEvent == "" || rule.TriggerEvent == triggerEvent) {
			result = append(result, rule)
		}
	}
	return result, nil
}

func (r *stubPluginWorkflowRuleRepo) ListEnabled(ctx context.Context, tenantID uuid.UUID, triggerEvent string) ([]*models.WorkflowRule, error) {
	rules, err := r.List(ctx, tenantID, triggerEvent)
	if err != nil {
		return nil, err
	}
	var enabled []*models.WorkflowRule
	for _, rule := range rules {
		if rule.Enabled {
			enabled = append(enabled, rule)
		}
	}
	return enabled, nil
}

type stubPluginTemplateRepo struct {
	byID    map[uuid.UUID]*models.IndustryTemplate
	listErr error
}

func newStubPluginTemplateRepo() *stubPluginTemplateRepo {
	return &stubPluginTemplateRepo{byID: make(map[uuid.UUID]*models.IndustryTemplate)}
}

func (r *stubPluginTemplateRepo) List(_ context.Context, industry string) ([]*models.IndustryTemplate, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	var result []*models.IndustryTemplate
	for _, t := range r.byID {
		if industry == "" || t.Industry == industry {
			result = append(result, t)
		}
	}
	return result, nil
}

func (r *stubPluginTemplateRepo) GetByID(_ context.Context, id uuid.UUID) (*models.IndustryTemplate, error) {
	return r.byID[id], nil
}

// ============================================================================
// Test helpers
// ============================================================================

type pluginTestRepos struct {
	manifests       *stubPluginManifestRepo
	installations   *stubPluginInstallationRepo
	permissions     *stubPluginPermissionRepo
	kv              *stubPluginKVRepo
	executionLogs   *stubPluginExecutionLogRepo
	validationRules *stubPluginValidationRuleRepo
	workflowRules   *stubPluginWorkflowRuleRepo
	templates       *stubPluginTemplateRepo
}

func newPluginTestRepos() *pluginTestRepos {
	return &pluginTestRepos{
		manifests:       newStubPluginManifestRepo(),
		installations:   newStubPluginInstallationRepo(),
		permissions:     newStubPluginPermissionRepo(),
		kv:              newStubPluginKVRepo(),
		executionLogs:   &stubPluginExecutionLogRepo{},
		validationRules: newStubPluginValidationRuleRepo(),
		workflowRules:   newStubPluginWorkflowRuleRepo(),
		templates:       newStubPluginTemplateRepo(),
	}
}

func newPluginTestServer(repos *pluginTestRepos) *PluginGRPCServer {
	svc := plugin.NewService(
		repos.manifests,
		repos.installations,
		repos.permissions,
		repos.kv,
		repos.executionLogs,
		repos.validationRules,
		repos.workflowRules,
		repos.templates,
	)
	return NewPluginGRPCServer(svc)
}

func pluginCtxWithTenant(tenantID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), middleware.TenantIDKey, tenantID.String())
}

func seedManifest(repos *pluginTestRepos, tenantID *uuid.UUID, slug string, pluginType models.PluginType) *models.PluginManifest {
	m := &models.PluginManifest{
		ID:                uuid.New(),
		TenantID:          tenantID,
		Slug:              slug,
		Name:              slug,
		SettingsSchema:    json.RawMessage("{}"),
		Permissions:       []string{"crm:read"},
		HookRegistrations: []models.HookRegistration{},
		PluginType:        pluginType,
	}
	repos.manifests.byID[m.ID] = m
	return m
}

func seedInstallation(repos *pluginTestRepos, tenantID, manifestID uuid.UUID, status models.InstallationStatus) *models.PluginInstallation {
	inst := &models.PluginInstallation{
		ID:          uuid.New(),
		TenantID:    tenantID,
		ManifestID:  manifestID,
		Status:      status,
		Settings:    json.RawMessage("{}"),
		InstalledBy: uuid.New(),
	}
	repos.installations.byID[inst.ID] = inst
	return inst
}

// ============================================================================
// Error mapping — table test
// ============================================================================

func TestMapPluginError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"manifest not found", plugin.ErrManifestNotFound, codes.NotFound},
		{"installation not found", plugin.ErrInstallationNotFound, codes.NotFound},
		{"validation rule not found", plugin.ErrValidationRuleNotFound, codes.NotFound},
		{"workflow rule not found", plugin.ErrWorkflowRuleNotFound, codes.NotFound},
		{"template not found", plugin.ErrTemplateNotFound, codes.NotFound},
		{"kv key not found", plugin.ErrKVKeyNotFound, codes.NotFound},
		{"manifest slug exists", plugin.ErrManifestSlugExists, codes.AlreadyExists},
		{"already installed", plugin.ErrAlreadyInstalled, codes.AlreadyExists},
		{"invalid settings", plugin.ErrInvalidSettings, codes.InvalidArgument},
		{"invalid settings schema", plugin.ErrInvalidSettingsSchema, codes.InvalidArgument},
		{"invalid rule config", plugin.ErrInvalidRuleConfig, codes.InvalidArgument},
		{"permission not declared", plugin.ErrPermissionNotDeclared, codes.InvalidArgument},
		{"wasm binary required", plugin.ErrWASMBinaryRequired, codes.InvalidArgument},
		{"wasm binary not allowed", plugin.ErrWASMBinaryNotAllowed, codes.InvalidArgument},
		{"manifest immutable", plugin.ErrManifestImmutable, codes.PermissionDenied},
		{"generic unmapped error", errors.New("boom"), codes.Internal},
		// Known gap #1: ErrPluginHasInstallations is a real sentinel returned by
		// Service.DeleteManifest (service.go:165) but mapPluginError has no case
		// for it at all - it falls through to the generic Internal branch instead
		// of a client-actionable code such as FailedPrecondition. See
		// TestPluginDeleteManifest/has_active_installations_maps_to_Internal
		// (documents the same gap at the handler level) and the journal entry for
		// this iteration.
		{"plugin has installations (undocumented sentinel, falls through to Internal)", plugin.ErrPluginHasInstallations, codes.Internal},
		// Known gap #2: isNotFound/isAlreadyExists/isInvalidArgument compare with
		// a plain `==` against the sentinel, not errors.Is. Every sentinel that
		// Service wraps with fmt.Errorf("%w: ...", ...) - ErrPermissionNotDeclared
		// in ApprovePermissions, ErrInvalidSettings in UpdatePluginSettings - is
		// therefore never recognized here and always falls through to Internal,
		// even though the unwrapped sentinel maps correctly one line above.
		{"wrapped invalid-argument sentinel (== instead of errors.Is, falls through to Internal)", fmt.Errorf("wrap: %w", plugin.ErrPermissionNotDeclared), codes.Internal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := mapPluginError(tc.err)
			requireGRPCCode(t, err, tc.want)
		})
	}
	t.Run("nil", func(t *testing.T) {
		require.NoError(t, mapPluginError(nil))
	})
}

// ============================================================================
// Manifest CRUD
// ============================================================================

func TestPluginCreateManifest(t *testing.T) {
	t.Run("success config plugin", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		tenantID := uuid.New()

		resp, err := srv.CreateManifest(pluginCtxWithTenant(tenantID), &pluginv1.CreateManifestRequest{
			Slug:        "my-plugin",
			Name:        "My Plugin",
			Description: "does things",
			Version:     "1.0.0",
			Author:      "Acme",
			Permissions: []string{"crm:read", "crm:write"},
			HookRegistrations: []*pluginv1.HookRegistrationMsg{
				{HookType: "before_save", Module: "crm", EntityType: "contact", Priority: 10},
			},
			PluginType: "config",
		})

		requireGRPCOK(t, err)
		m := resp.GetManifest()
		require.Equal(t, "my-plugin", m.GetSlug())
		require.Equal(t, "config", m.GetPluginType())
		require.Equal(t, []string{"crm:read", "crm:write"}, m.GetPermissions())
		require.Len(t, m.GetHookRegistrations(), 1)
		require.Equal(t, "before_save", m.GetHookRegistrations()[0].GetHookType())
		require.Empty(t, m.GetWasmBinaryHash())

		id, parseErr := uuid.Parse(m.GetId())
		require.NoError(t, parseErr)
		stored := repos.manifests.byID[id]
		require.NotNil(t, stored.TenantID)
		require.Equal(t, tenantID, *stored.TenantID)
	})

	t.Run("wasm plugin stamps binary hash", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)

		resp, err := srv.CreateManifest(pluginCtxWithTenant(uuid.New()), &pluginv1.CreateManifestRequest{
			Slug:       "wasm-plugin",
			Name:       "WASM Plugin",
			PluginType: "wasm",
			WasmBinary: []byte("fake-wasm-bytes"),
		})

		requireGRPCOK(t, err)
		require.NotEmpty(t, resp.GetManifest().GetWasmBinaryHash())
	})

	t.Run("missing slug is rejected", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)

		_, err := srv.CreateManifest(pluginCtxWithTenant(uuid.New()), &pluginv1.CreateManifestRequest{Name: "No Slug"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("missing name is rejected", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)

		_, err := srv.CreateManifest(pluginCtxWithTenant(uuid.New()), &pluginv1.CreateManifestRequest{Slug: "no-name"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("duplicate slug is rejected", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		tenantID := uuid.New()
		seedManifest(repos, &tenantID, "taken", models.PluginTypeConfig)

		_, err := srv.CreateManifest(pluginCtxWithTenant(tenantID), &pluginv1.CreateManifestRequest{
			Slug: "taken", Name: "Duplicate", PluginType: "config",
		})
		requireGRPCCode(t, err, codes.AlreadyExists)
	})

	t.Run("wasm type without binary is rejected", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)

		_, err := srv.CreateManifest(pluginCtxWithTenant(uuid.New()), &pluginv1.CreateManifestRequest{
			Slug: "wasm-no-binary", Name: "WASM No Binary", PluginType: "wasm",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("config type with binary is rejected", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)

		_, err := srv.CreateManifest(pluginCtxWithTenant(uuid.New()), &pluginv1.CreateManifestRequest{
			Slug: "config-with-binary", Name: "Config With Binary", PluginType: "config",
			WasmBinary: []byte("nope"),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	// Finding: unlike CreateValidationRule/CreateWorkflowRule (which resolve
	// the tenant via ruleTenant() and return an explicit InvalidArgument
	// status.Error before ever reaching the service), CreateManifest resolves
	// the tenant inside Service.CreateManifest and wraps middleware's sentinel
	// with fmt.Errorf("create manifest: %w", err). Because mapPluginError
	// compares with == (see TestMapPluginError gap #2), that wrapped error is
	// never recognized and this request gets Internal instead of the
	// InvalidArgument every other tenant-less request in this file gets.
	t.Run("missing tenant context maps to Internal, not InvalidArgument (documents current gap)", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)

		_, err := srv.CreateManifest(context.Background(), &pluginv1.CreateManifestRequest{
			Slug: "no-tenant", Name: "No Tenant", PluginType: "config",
		})
		requireGRPCCode(t, err, codes.Internal)
	})
}

func TestPluginGetManifest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		tenantID := uuid.New()
		m := seedManifest(repos, &tenantID, "existing", models.PluginTypeConfig)

		resp, err := srv.GetManifest(context.Background(), &pluginv1.GetManifestRequest{ManifestId: m.ID.String()})
		requireGRPCOK(t, err)
		require.Equal(t, "existing", resp.GetManifest().GetSlug())
	})

	t.Run("invalid manifest_id", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)

		_, err := srv.GetManifest(context.Background(), &pluginv1.GetManifestRequest{ManifestId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not found", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)

		_, err := srv.GetManifest(context.Background(), &pluginv1.GetManifestRequest{ManifestId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

func TestPluginListManifests(t *testing.T) {
	t.Run("empty result is an empty slice, not nil", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)

		resp, err := srv.ListManifests(context.Background(), &pluginv1.ListManifestsRequest{})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.GetManifests())
		require.Empty(t, resp.GetManifests())
	})

	t.Run("filters by plugin_type", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		tenantID := uuid.New()
		seedManifest(repos, &tenantID, "config-one", models.PluginTypeConfig)
		seedManifest(repos, &tenantID, "wasm-one", models.PluginTypeWASM)

		resp, err := srv.ListManifests(context.Background(), &pluginv1.ListManifestsRequest{PluginType: "wasm"})
		requireGRPCOK(t, err)
		require.Len(t, resp.GetManifests(), 1)
		require.Equal(t, "wasm-one", resp.GetManifests()[0].GetSlug())
	})

	t.Run("repository error surfaces as Internal", func(t *testing.T) {
		repos := newPluginTestRepos()
		repos.manifests.listErr = errors.New("db down")
		srv := newPluginTestServer(repos)

		_, err := srv.ListManifests(context.Background(), &pluginv1.ListManifestsRequest{})
		requireGRPCCode(t, err, codes.Internal)
	})
}

func TestPluginDeleteManifest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		tenantID := uuid.New()
		m := seedManifest(repos, &tenantID, "deletable", models.PluginTypeConfig)

		resp, err := srv.DeleteManifest(context.Background(), &pluginv1.DeleteManifestRequest{ManifestId: m.ID.String()})
		requireGRPCOK(t, err)
		require.True(t, resp.GetSuccess())
		_, stillThere := repos.manifests.byID[m.ID]
		require.False(t, stillThere)
	})

	t.Run("invalid manifest_id", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)

		_, err := srv.DeleteManifest(context.Background(), &pluginv1.DeleteManifestRequest{ManifestId: "garbage"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not found", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)

		_, err := srv.DeleteManifest(context.Background(), &pluginv1.DeleteManifestRequest{ManifestId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("catalogue manifest is immutable", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		m := seedManifest(repos, nil, "catalogue", models.PluginTypeConfig)

		_, err := srv.DeleteManifest(context.Background(), &pluginv1.DeleteManifestRequest{ManifestId: m.ID.String()})
		requireGRPCCode(t, err, codes.PermissionDenied)
	})

	// Finding: ErrPluginHasInstallations is a distinct, well-named sentinel
	// but mapPluginError has no branch for it (see TestMapPluginError gap #1),
	// so a client trying to delete an in-use manifest gets an opaque 500
	// instead of a 400/409 they could act on.
	t.Run("has active installations maps to Internal (documents current gap)", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		tenantID := uuid.New()
		m := seedManifest(repos, &tenantID, "in-use", models.PluginTypeConfig)
		repos.manifests.hasActiveInstalls[m.ID] = true

		_, err := srv.DeleteManifest(context.Background(), &pluginv1.DeleteManifestRequest{ManifestId: m.ID.String()})
		requireGRPCCode(t, err, codes.Internal)
	})
}

// ============================================================================
// Installation lifecycle
// ============================================================================

func TestPluginInstallPlugin(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		tenantID := uuid.New()
		m := seedManifest(repos, &tenantID, "installable", models.PluginTypeConfig)
		installedBy := uuid.New()

		resp, err := srv.InstallPlugin(context.Background(), &pluginv1.InstallPluginRequest{
			TenantId: tenantID.String(), ManifestId: m.ID.String(), InstalledBy: installedBy.String(),
		})
		requireGRPCOK(t, err)
		require.Equal(t, "pending_approval", resp.GetInstallation().GetStatus())
	})

	for _, tc := range []struct {
		name string
		req  *pluginv1.InstallPluginRequest
	}{
		{"invalid tenant_id", &pluginv1.InstallPluginRequest{TenantId: "bad", ManifestId: uuid.New().String(), InstalledBy: uuid.New().String()}},
		{"invalid manifest_id", &pluginv1.InstallPluginRequest{TenantId: uuid.New().String(), ManifestId: "bad", InstalledBy: uuid.New().String()}},
		{"invalid installed_by", &pluginv1.InstallPluginRequest{TenantId: uuid.New().String(), ManifestId: uuid.New().String(), InstalledBy: "bad"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repos := newPluginTestRepos()
			srv := newPluginTestServer(repos)
			_, err := srv.InstallPlugin(context.Background(), tc.req)
			requireGRPCCode(t, err, codes.InvalidArgument)
		})
	}

	t.Run("manifest not found", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)

		_, err := srv.InstallPlugin(context.Background(), &pluginv1.InstallPluginRequest{
			TenantId: uuid.New().String(), ManifestId: uuid.New().String(), InstalledBy: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("already installed", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		tenantID := uuid.New()
		m := seedManifest(repos, &tenantID, "already", models.PluginTypeConfig)
		seedInstallation(repos, tenantID, m.ID, models.InstallationStatusActive)

		_, err := srv.InstallPlugin(context.Background(), &pluginv1.InstallPluginRequest{
			TenantId: tenantID.String(), ManifestId: m.ID.String(), InstalledBy: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.AlreadyExists)
	})

	t.Run("reinstall after uninstall succeeds", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		tenantID := uuid.New()
		m := seedManifest(repos, &tenantID, "reinstall", models.PluginTypeConfig)
		seedInstallation(repos, tenantID, m.ID, models.InstallationStatusUninstalled)

		_, err := srv.InstallPlugin(context.Background(), &pluginv1.InstallPluginRequest{
			TenantId: tenantID.String(), ManifestId: m.ID.String(), InstalledBy: uuid.New().String(),
		})
		requireGRPCOK(t, err)
	})
}

func TestPluginUninstallEnableDisable(t *testing.T) {
	tenantID := uuid.New()

	t.Run("uninstall success", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		m := seedManifest(repos, &tenantID, "u1", models.PluginTypeConfig)
		inst := seedInstallation(repos, tenantID, m.ID, models.InstallationStatusActive)

		resp, err := srv.UninstallPlugin(context.Background(), &pluginv1.UninstallPluginRequest{InstallationId: inst.ID.String()})
		requireGRPCOK(t, err)
		require.True(t, resp.GetSuccess())
		require.Equal(t, models.InstallationStatusUninstalled, repos.installations.byID[inst.ID].Status)
	})

	t.Run("uninstall invalid id", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.UninstallPlugin(context.Background(), &pluginv1.UninstallPluginRequest{InstallationId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("uninstall not found", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.UninstallPlugin(context.Background(), &pluginv1.UninstallPluginRequest{InstallationId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("enable success", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		m := seedManifest(repos, &tenantID, "e1", models.PluginTypeConfig)
		inst := seedInstallation(repos, tenantID, m.ID, models.InstallationStatusPendingApproval)

		resp, err := srv.EnablePlugin(context.Background(), &pluginv1.EnablePluginRequest{InstallationId: inst.ID.String()})
		requireGRPCOK(t, err)
		require.Equal(t, "active", resp.GetInstallation().GetStatus())
	})

	t.Run("enable invalid id", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.EnablePlugin(context.Background(), &pluginv1.EnablePluginRequest{InstallationId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("enable not found", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.EnablePlugin(context.Background(), &pluginv1.EnablePluginRequest{InstallationId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("disable success", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		m := seedManifest(repos, &tenantID, "d1", models.PluginTypeConfig)
		inst := seedInstallation(repos, tenantID, m.ID, models.InstallationStatusActive)

		resp, err := srv.DisablePlugin(context.Background(), &pluginv1.DisablePluginRequest{InstallationId: inst.ID.String()})
		requireGRPCOK(t, err)
		require.Equal(t, "disabled", resp.GetInstallation().GetStatus())
	})

	t.Run("disable invalid id", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.DisablePlugin(context.Background(), &pluginv1.DisablePluginRequest{InstallationId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("disable not found", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.DisablePlugin(context.Background(), &pluginv1.DisablePluginRequest{InstallationId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

func TestPluginGetInstallation(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		tenantID := uuid.New()
		m := seedManifest(repos, &tenantID, "g1", models.PluginTypeConfig)
		inst := seedInstallation(repos, tenantID, m.ID, models.InstallationStatusActive)

		resp, err := srv.GetInstallation(context.Background(), &pluginv1.GetInstallationRequest{InstallationId: inst.ID.String()})
		requireGRPCOK(t, err)
		require.Equal(t, inst.ID.String(), resp.GetInstallation().GetId())
	})

	t.Run("invalid id", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.GetInstallation(context.Background(), &pluginv1.GetInstallationRequest{InstallationId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not found", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.GetInstallation(context.Background(), &pluginv1.GetInstallationRequest{InstallationId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

func TestPluginListInstallations(t *testing.T) {
	t.Run("success enriches granted permissions", func(t *testing.T) {
		repos := newPluginTestRepos()
		instID := uuid.New()
		repos.permissions.granted[instID] = []string{"crm:read"}
		repos.installations.listWithManifest = []*models.PluginInstallationWithManifest{
			{
				PluginInstallation: models.PluginInstallation{ID: instID, Status: models.InstallationStatusActive},
				ManifestSlug:       "installed-one",
				Permissions:        []string{"crm:read", "crm:write"},
			},
		}
		srv := newPluginTestServer(repos)

		resp, err := srv.ListInstallations(context.Background(), &pluginv1.ListInstallationsRequest{TenantId: uuid.New().String()})
		requireGRPCOK(t, err)
		require.Len(t, resp.GetInstallations(), 1)
		require.Equal(t, []string{"crm:read"}, resp.GetInstallations()[0].GetGrantedPermissions())
		require.Equal(t, []string{"crm:read", "crm:write"}, resp.GetInstallations()[0].GetRequiredPermissions())
	})

	t.Run("empty result is an empty slice, not nil", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)

		resp, err := srv.ListInstallations(context.Background(), &pluginv1.ListInstallationsRequest{TenantId: uuid.New().String()})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.GetInstallations())
		require.Empty(t, resp.GetInstallations())
	})

	t.Run("invalid tenant_id", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.ListInstallations(context.Background(), &pluginv1.ListInstallationsRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

// ============================================================================
// Permissions
// ============================================================================

func TestPluginApprovePermissions(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		tenantID := uuid.New()
		m := seedManifest(repos, &tenantID, "perm-ok", models.PluginTypeConfig)
		m.Permissions = []string{"crm:read", "crm:write"}
		inst := seedInstallation(repos, tenantID, m.ID, models.InstallationStatusPendingApproval)

		resp, err := srv.ApprovePermissions(context.Background(), &pluginv1.ApprovePermissionsRequest{
			InstallationId: inst.ID.String(), Permissions: []string{"crm:read"}, GrantedBy: uuid.New().String(),
		})
		requireGRPCOK(t, err)
		require.True(t, resp.GetSuccess())
		require.Equal(t, []string{"crm:read"}, repos.permissions.granted[inst.ID])
	})

	t.Run("invalid installation_id", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.ApprovePermissions(context.Background(), &pluginv1.ApprovePermissionsRequest{
			InstallationId: "bad", GrantedBy: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid granted_by", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.ApprovePermissions(context.Background(), &pluginv1.ApprovePermissionsRequest{
			InstallationId: uuid.New().String(), GrantedBy: "bad",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("installation not found", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.ApprovePermissions(context.Background(), &pluginv1.ApprovePermissionsRequest{
			InstallationId: uuid.New().String(), GrantedBy: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.NotFound)
	})

	// Finding: Service.ApprovePermissions wraps ErrPermissionNotDeclared with
	// fmt.Errorf("%w: %s", ...), so mapPluginError's == comparison never
	// matches it (TestMapPluginError gap #2) and an undeclared-permission
	// request gets Internal instead of InvalidArgument.
	t.Run("undeclared permission maps to Internal, not InvalidArgument (documents current gap)", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		tenantID := uuid.New()
		m := seedManifest(repos, &tenantID, "perm-gap", models.PluginTypeConfig)
		m.Permissions = []string{"crm:read"}
		inst := seedInstallation(repos, tenantID, m.ID, models.InstallationStatusPendingApproval)

		_, err := srv.ApprovePermissions(context.Background(), &pluginv1.ApprovePermissionsRequest{
			InstallationId: inst.ID.String(), Permissions: []string{"crm:delete-everything"}, GrantedBy: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.Internal)
	})

	// Finding: ApprovePermissions dereferences the manifest it looks up
	// (service.go: `for _, p := range manifest.Permissions`) without checking
	// it for nil first. An installation whose manifest has since been deleted
	// - possible because DeleteManifest's HasActiveInstallations check only
	// counts non-uninstalled installations, so an uninstalled installation
	// row can outlive its manifest - makes this panic. cmd/plugin/main.go
	// wires middleware.RecoveryUnaryInterceptor(), so in production this
	// becomes an opaque Internal instead of crashing the process, but it is
	// still a panic on every such call, not a handled error path.
	t.Run("orphaned installation (deleted manifest) panics (documents current gap)", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		tenantID := uuid.New()
		orphanManifestID := uuid.New() // never stored in repos.manifests
		inst := seedInstallation(repos, tenantID, orphanManifestID, models.InstallationStatusPendingApproval)

		require.Panics(t, func() {
			_, _ = srv.ApprovePermissions(context.Background(), &pluginv1.ApprovePermissionsRequest{
				InstallationId: inst.ID.String(), Permissions: []string{"crm:read"}, GrantedBy: uuid.New().String(),
			})
		})
	})
}

func TestPluginListGrantedPermissions(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repos := newPluginTestRepos()
		instID := uuid.New()
		repos.permissions.granted[instID] = []string{"crm:read"}
		srv := newPluginTestServer(repos)

		resp, err := srv.ListGrantedPermissions(context.Background(), &pluginv1.ListGrantedPermissionsRequest{InstallationId: instID.String()})
		requireGRPCOK(t, err)
		require.Equal(t, []string{"crm:read"}, resp.GetPermissions())
	})

	t.Run("empty for unknown installation", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)

		resp, err := srv.ListGrantedPermissions(context.Background(), &pluginv1.ListGrantedPermissionsRequest{InstallationId: uuid.New().String()})
		requireGRPCOK(t, err)
		require.Empty(t, resp.GetPermissions())
	})

	t.Run("invalid installation_id", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.ListGrantedPermissions(context.Background(), &pluginv1.ListGrantedPermissionsRequest{InstallationId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("repository error surfaces as Internal", func(t *testing.T) {
		repos := newPluginTestRepos()
		repos.permissions.listErr = errors.New("db down")
		srv := newPluginTestServer(repos)
		_, err := srv.ListGrantedPermissions(context.Background(), &pluginv1.ListGrantedPermissionsRequest{InstallationId: uuid.New().String()})
		requireGRPCCode(t, err, codes.Internal)
	})
}

// ============================================================================
// Settings
// ============================================================================

func TestPluginSettings(t *testing.T) {
	t.Run("get success", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		tenantID := uuid.New()
		m := seedManifest(repos, &tenantID, "settings-1", models.PluginTypeConfig)
		inst := seedInstallation(repos, tenantID, m.ID, models.InstallationStatusActive)
		inst.Settings = json.RawMessage(`{"foo":"bar"}`)

		resp, err := srv.GetPluginSettings(context.Background(), &pluginv1.GetPluginSettingsRequest{InstallationId: inst.ID.String()})
		requireGRPCOK(t, err)
		require.JSONEq(t, `{"foo":"bar"}`, resp.GetSettings())
	})

	t.Run("get invalid id", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.GetPluginSettings(context.Background(), &pluginv1.GetPluginSettingsRequest{InstallationId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("get not found", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.GetPluginSettings(context.Background(), &pluginv1.GetPluginSettingsRequest{InstallationId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("update success against empty schema", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		tenantID := uuid.New()
		m := seedManifest(repos, &tenantID, "settings-2", models.PluginTypeConfig)
		inst := seedInstallation(repos, tenantID, m.ID, models.InstallationStatusActive)

		resp, err := srv.UpdatePluginSettings(context.Background(), &pluginv1.UpdatePluginSettingsRequest{
			InstallationId: inst.ID.String(), Settings: `{"anything":"goes"}`,
		})
		requireGRPCOK(t, err)
		require.True(t, resp.GetSuccess())
		require.JSONEq(t, `{"anything":"goes"}`, string(repos.installations.byID[inst.ID].Settings))
	})

	t.Run("update invalid id", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.UpdatePluginSettings(context.Background(), &pluginv1.UpdatePluginSettingsRequest{InstallationId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("update not found", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.UpdatePluginSettings(context.Background(), &pluginv1.UpdatePluginSettingsRequest{InstallationId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	// Finding: same == vs errors.Is gap as ApprovePermissions - a genuine
	// schema mismatch (missing required field) should be InvalidArgument but
	// UpdatePluginSettings wraps ErrInvalidSettings with fmt.Errorf("%w: %v",
	// ...) and mapPluginError never recognizes it.
	t.Run("schema mismatch maps to Internal, not InvalidArgument (documents current gap)", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		tenantID := uuid.New()
		m := seedManifest(repos, &tenantID, "settings-3", models.PluginTypeConfig)
		m.SettingsSchema = json.RawMessage(`{"type":"object","required":["api_key"],"properties":{"api_key":{"type":"string"}}}`)
		inst := seedInstallation(repos, tenantID, m.ID, models.InstallationStatusActive)

		_, err := srv.UpdatePluginSettings(context.Background(), &pluginv1.UpdatePluginSettingsRequest{
			InstallationId: inst.ID.String(), Settings: `{}`,
		})
		requireGRPCCode(t, err, codes.Internal)
	})

	// Finding: same nil-manifest-dereference class as ApprovePermissions -
	// UpdatePluginSettings calls schemaValidator.Validate(manifest.SettingsSchema, ...)
	// without a nil check on manifest.
	t.Run("orphaned installation (deleted manifest) panics on update (documents current gap)", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		inst := seedInstallation(repos, uuid.New(), uuid.New(), models.InstallationStatusActive)

		require.Panics(t, func() {
			_, _ = srv.UpdatePluginSettings(context.Background(), &pluginv1.UpdatePluginSettingsRequest{
				InstallationId: inst.ID.String(), Settings: `{}`,
			})
		})
	})

	t.Run("schema success", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		tenantID := uuid.New()
		m := seedManifest(repos, &tenantID, "schema-1", models.PluginTypeConfig)
		m.SettingsSchema = json.RawMessage(`{"type":"object"}`)
		inst := seedInstallation(repos, tenantID, m.ID, models.InstallationStatusActive)

		resp, err := srv.GetPluginSettingsSchema(context.Background(), &pluginv1.GetPluginSettingsSchemaRequest{InstallationId: inst.ID.String()})
		requireGRPCOK(t, err)
		require.JSONEq(t, `{"type":"object"}`, resp.GetSchema())
	})

	t.Run("schema invalid id", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.GetPluginSettingsSchema(context.Background(), &pluginv1.GetPluginSettingsSchemaRequest{InstallationId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("schema installation not found", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.GetPluginSettingsSchema(context.Background(), &pluginv1.GetPluginSettingsSchemaRequest{InstallationId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	// Finding: same nil-manifest-dereference class again, this time on
	// `return manifest.SettingsSchema, nil`.
	t.Run("orphaned installation (deleted manifest) panics on schema lookup (documents current gap)", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		inst := seedInstallation(repos, uuid.New(), uuid.New(), models.InstallationStatusActive)

		require.Panics(t, func() {
			_, _ = srv.GetPluginSettingsSchema(context.Background(), &pluginv1.GetPluginSettingsSchemaRequest{InstallationId: inst.ID.String()})
		})
	})
}

// ============================================================================
// Validation rules
// ============================================================================

func TestPluginValidationRules(t *testing.T) {
	t.Run("create success without installation scope", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		tenantID := uuid.New()

		resp, err := srv.CreateValidationRule(pluginCtxWithTenant(tenantID), &pluginv1.CreateValidationRuleRequest{
			Name: "iban-format", EntityType: "contact", FieldName: "iban", RuleType: "format",
			RuleConfig: `{"format":"iban","required":true}`, ErrorMessage: "invalid IBAN",
		})
		requireGRPCOK(t, err)
		require.Empty(t, resp.GetRule().GetInstallationId())
		require.True(t, resp.GetRule().GetEnabled())
	})

	t.Run("create success scoped to an installation", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		tenantID := uuid.New()
		instID := uuid.New()

		resp, err := srv.CreateValidationRule(pluginCtxWithTenant(tenantID), &pluginv1.CreateValidationRuleRequest{
			Name: "scoped", EntityType: "deal", FieldName: "amount", RuleType: "range",
			RuleConfig: `{"min":0}`, InstallationId: instID.String(),
		})
		requireGRPCOK(t, err)
		require.Equal(t, instID.String(), resp.GetRule().GetInstallationId())
	})

	t.Run("create missing tenant context", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.CreateValidationRule(context.Background(), &pluginv1.CreateValidationRuleRequest{Name: "x"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("create invalid installation_id", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.CreateValidationRule(pluginCtxWithTenant(uuid.New()), &pluginv1.CreateValidationRuleRequest{
			Name: "x", InstallationId: "bad",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("create repository error surfaces as Internal", func(t *testing.T) {
		repos := newPluginTestRepos()
		repos.validationRules.createErr = errors.New("db down")
		srv := newPluginTestServer(repos)
		_, err := srv.CreateValidationRule(pluginCtxWithTenant(uuid.New()), &pluginv1.CreateValidationRuleRequest{Name: "x"})
		requireGRPCCode(t, err, codes.Internal)
	})

	t.Run("list success and empty is not nil", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		tenantID := uuid.New()

		resp, err := srv.ListValidationRules(pluginCtxWithTenant(tenantID), &pluginv1.ListValidationRulesRequest{EntityType: "contact"})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.GetRules())
		require.Empty(t, resp.GetRules())
	})

	t.Run("list missing tenant context", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.ListValidationRules(context.Background(), &pluginv1.ListValidationRulesRequest{})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("update success", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		tenantID := uuid.New()
		rule := &models.ValidationRule{ID: uuid.New(), TenantID: tenantID, Name: "old", Enabled: true}
		repos.validationRules.byID[rule.ID] = rule

		resp, err := srv.UpdateValidationRule(context.Background(), &pluginv1.UpdateValidationRuleRequest{
			RuleId: rule.ID.String(), Name: "new", Enabled: false,
		})
		requireGRPCOK(t, err)
		require.Equal(t, "new", resp.GetRule().GetName())
		require.False(t, resp.GetRule().GetEnabled())
	})

	t.Run("update invalid rule_id", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.UpdateValidationRule(context.Background(), &pluginv1.UpdateValidationRuleRequest{RuleId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("update not found", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.UpdateValidationRule(context.Background(), &pluginv1.UpdateValidationRuleRequest{RuleId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("delete success", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		rule := &models.ValidationRule{ID: uuid.New()}
		repos.validationRules.byID[rule.ID] = rule

		resp, err := srv.DeleteValidationRule(context.Background(), &pluginv1.DeleteValidationRuleRequest{RuleId: rule.ID.String()})
		requireGRPCOK(t, err)
		require.True(t, resp.GetSuccess())
	})

	t.Run("delete invalid rule_id", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.DeleteValidationRule(context.Background(), &pluginv1.DeleteValidationRuleRequest{RuleId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("delete not found", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.DeleteValidationRule(context.Background(), &pluginv1.DeleteValidationRuleRequest{RuleId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

// ============================================================================
// Workflow rules
// ============================================================================

func TestPluginWorkflowRules(t *testing.T) {
	t.Run("create success", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		tenantID := uuid.New()

		resp, err := srv.CreateWorkflowRule(pluginCtxWithTenant(tenantID), &pluginv1.CreateWorkflowRuleRequest{
			Name: "notify-on-win", TriggerEvent: "deal.won", Conditions: `{}`, Actions: `[{"type":"notify"}]`,
		})
		requireGRPCOK(t, err)
		require.True(t, resp.GetRule().GetEnabled())
	})

	t.Run("create missing tenant context", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.CreateWorkflowRule(context.Background(), &pluginv1.CreateWorkflowRuleRequest{Name: "x"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("create invalid installation_id", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.CreateWorkflowRule(pluginCtxWithTenant(uuid.New()), &pluginv1.CreateWorkflowRuleRequest{
			Name: "x", InstallationId: "bad",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("create repository error surfaces as Internal", func(t *testing.T) {
		repos := newPluginTestRepos()
		repos.workflowRules.createErr = errors.New("db down")
		srv := newPluginTestServer(repos)
		_, err := srv.CreateWorkflowRule(pluginCtxWithTenant(uuid.New()), &pluginv1.CreateWorkflowRuleRequest{Name: "x"})
		requireGRPCCode(t, err, codes.Internal)
	})

	t.Run("list success and empty is not nil", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)

		resp, err := srv.ListWorkflowRules(pluginCtxWithTenant(uuid.New()), &pluginv1.ListWorkflowRulesRequest{})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.GetRules())
		require.Empty(t, resp.GetRules())
	})

	t.Run("list missing tenant context", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.ListWorkflowRules(context.Background(), &pluginv1.ListWorkflowRulesRequest{})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("update success", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		rule := &models.WorkflowRule{ID: uuid.New(), Name: "old", Enabled: true}
		repos.workflowRules.byID[rule.ID] = rule

		resp, err := srv.UpdateWorkflowRule(context.Background(), &pluginv1.UpdateWorkflowRuleRequest{
			RuleId: rule.ID.String(), Name: "new", Enabled: false,
		})
		requireGRPCOK(t, err)
		require.Equal(t, "new", resp.GetRule().GetName())
	})

	t.Run("update invalid rule_id", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.UpdateWorkflowRule(context.Background(), &pluginv1.UpdateWorkflowRuleRequest{RuleId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("update not found", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.UpdateWorkflowRule(context.Background(), &pluginv1.UpdateWorkflowRuleRequest{RuleId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("delete success", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		rule := &models.WorkflowRule{ID: uuid.New()}
		repos.workflowRules.byID[rule.ID] = rule

		resp, err := srv.DeleteWorkflowRule(context.Background(), &pluginv1.DeleteWorkflowRuleRequest{RuleId: rule.ID.String()})
		requireGRPCOK(t, err)
		require.True(t, resp.GetSuccess())
	})

	t.Run("delete invalid rule_id", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.DeleteWorkflowRule(context.Background(), &pluginv1.DeleteWorkflowRuleRequest{RuleId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("delete not found", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.DeleteWorkflowRule(context.Background(), &pluginv1.DeleteWorkflowRuleRequest{RuleId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

// ============================================================================
// Industry templates
// ============================================================================

func TestPluginIndustryTemplates(t *testing.T) {
	t.Run("list success and empty is not nil", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)

		resp, err := srv.ListIndustryTemplates(context.Background(), &pluginv1.ListIndustryTemplatesRequest{})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.GetTemplates())
		require.Empty(t, resp.GetTemplates())
	})

	t.Run("list filters by industry", func(t *testing.T) {
		repos := newPluginTestRepos()
		tpl := &models.IndustryTemplate{ID: uuid.New(), Slug: "handwerk", Industry: "handwerk",
			ValidationRules: json.RawMessage("[]"), WorkflowRules: json.RawMessage("[]")}
		repos.templates.byID[tpl.ID] = tpl
		srv := newPluginTestServer(repos)

		resp, err := srv.ListIndustryTemplates(context.Background(), &pluginv1.ListIndustryTemplatesRequest{Industry: "handwerk"})
		requireGRPCOK(t, err)
		require.Len(t, resp.GetTemplates(), 1)
	})

	t.Run("apply success creates rules from template", func(t *testing.T) {
		repos := newPluginTestRepos()
		tenantID := uuid.New()
		tpl := &models.IndustryTemplate{
			ID:   uuid.New(),
			Slug: "handwerk",
			ValidationRules: mustJSON(t, []models.ValidationRule{
				{Name: "vrule", EntityType: "contact", FieldName: "iban", RuleType: "format", RuleConfig: json.RawMessage(`{}`)},
			}),
			WorkflowRules: mustJSON(t, []models.WorkflowRule{
				{Name: "wrule", TriggerEvent: "deal.won", Conditions: json.RawMessage(`{}`), Actions: json.RawMessage(`[]`)},
			}),
		}
		repos.templates.byID[tpl.ID] = tpl
		srv := newPluginTestServer(repos)

		resp, err := srv.ApplyIndustryTemplate(pluginCtxWithTenant(tenantID), &pluginv1.ApplyIndustryTemplateRequest{
			TemplateId: tpl.ID.String(), AppliedBy: uuid.New().String(),
		})
		requireGRPCOK(t, err)
		require.True(t, resp.GetSuccess())
		require.Len(t, repos.validationRules.byID, 1)
		require.Len(t, repos.workflowRules.byID, 1)
	})

	t.Run("apply missing tenant context", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.ApplyIndustryTemplate(context.Background(), &pluginv1.ApplyIndustryTemplateRequest{
			TemplateId: uuid.New().String(), AppliedBy: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("apply invalid template_id", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.ApplyIndustryTemplate(pluginCtxWithTenant(uuid.New()), &pluginv1.ApplyIndustryTemplateRequest{
			TemplateId: "bad", AppliedBy: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("apply invalid applied_by", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.ApplyIndustryTemplate(pluginCtxWithTenant(uuid.New()), &pluginv1.ApplyIndustryTemplateRequest{
			TemplateId: uuid.New().String(), AppliedBy: "bad",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("apply template not found", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.ApplyIndustryTemplate(pluginCtxWithTenant(uuid.New()), &pluginv1.ApplyIndustryTemplateRequest{
			TemplateId: uuid.New().String(), AppliedBy: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

// ============================================================================
// Execution logs
// ============================================================================

func TestPluginListExecutionLogs(t *testing.T) {
	t.Run("success without installation filter", func(t *testing.T) {
		repos := newPluginTestRepos()
		repos.executionLogs.logs = []*models.PluginExecutionLog{
			{ID: uuid.New(), InstallationID: uuid.New(), HookType: "before_save", Status: models.ExecutionStatusSuccess},
		}
		srv := newPluginTestServer(repos)

		resp, err := srv.ListExecutionLogs(context.Background(), &pluginv1.ListExecutionLogsRequest{TenantId: uuid.New().String(), Limit: 10})
		requireGRPCOK(t, err)
		require.Len(t, resp.GetLogs(), 1)
	})

	t.Run("success with installation filter", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)

		resp, err := srv.ListExecutionLogs(context.Background(), &pluginv1.ListExecutionLogsRequest{
			TenantId: uuid.New().String(), InstallationId: uuid.New().String(),
		})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.GetLogs())
		require.Empty(t, resp.GetLogs())
	})

	t.Run("invalid tenant_id", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.ListExecutionLogs(context.Background(), &pluginv1.ListExecutionLogsRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid installation_id", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.ListExecutionLogs(context.Background(), &pluginv1.ListExecutionLogsRequest{
			TenantId: uuid.New().String(), InstallationId: "bad",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("repository error surfaces as Internal", func(t *testing.T) {
		repos := newPluginTestRepos()
		repos.executionLogs.listErr = errors.New("db down")
		srv := newPluginTestServer(repos)
		_, err := srv.ListExecutionLogs(context.Background(), &pluginv1.ListExecutionLogsRequest{TenantId: uuid.New().String()})
		requireGRPCCode(t, err, codes.Internal)
	})
}

// ============================================================================
// KV store
// ============================================================================

func TestPluginKVStore(t *testing.T) {
	t.Run("set then get round-trips", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		instID := uuid.New()

		_, err := srv.KVSet(context.Background(), &pluginv1.KVSetRequest{InstallationId: instID.String(), Key: "k", Value: `{"v":1}`})
		requireGRPCOK(t, err)

		resp, err := srv.KVGet(context.Background(), &pluginv1.KVGetRequest{InstallationId: instID.String(), Key: "k"})
		requireGRPCOK(t, err)
		require.True(t, resp.GetFound())
		require.JSONEq(t, `{"v":1}`, resp.GetValue())
	})

	t.Run("get missing key is not found but not an error", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)

		resp, err := srv.KVGet(context.Background(), &pluginv1.KVGetRequest{InstallationId: uuid.New().String(), Key: "absent"})
		requireGRPCOK(t, err)
		require.False(t, resp.GetFound())
	})

	t.Run("delete removes the key", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		instID := uuid.New()
		repos.kv.store[instID] = map[string]json.RawMessage{"k": json.RawMessage(`1`)}

		_, err := srv.KVDelete(context.Background(), &pluginv1.KVDeleteRequest{InstallationId: instID.String(), Key: "k"})
		requireGRPCOK(t, err)
		_, ok := repos.kv.store[instID]["k"]
		require.False(t, ok)
	})

	t.Run("list filters by prefix and empty is not nil", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		instID := uuid.New()
		repos.kv.store[instID] = map[string]json.RawMessage{
			"cache:a": json.RawMessage(`1`),
			"cache:b": json.RawMessage(`2`),
			"other":   json.RawMessage(`3`),
		}

		resp, err := srv.KVList(context.Background(), &pluginv1.KVListRequest{InstallationId: instID.String(), KeyPrefix: "cache:"})
		requireGRPCOK(t, err)
		require.Len(t, resp.GetEntries(), 2)

		empty, err := srv.KVList(context.Background(), &pluginv1.KVListRequest{InstallationId: uuid.New().String()})
		requireGRPCOK(t, err)
		require.NotNil(t, empty.GetEntries())
		require.Empty(t, empty.GetEntries())
	})

	for _, tc := range []struct {
		name string
		call func(*PluginGRPCServer) error
	}{
		{"get invalid id", func(s *PluginGRPCServer) error {
			_, err := s.KVGet(context.Background(), &pluginv1.KVGetRequest{InstallationId: "bad"})
			return err
		}},
		{"set invalid id", func(s *PluginGRPCServer) error {
			_, err := s.KVSet(context.Background(), &pluginv1.KVSetRequest{InstallationId: "bad"})
			return err
		}},
		{"delete invalid id", func(s *PluginGRPCServer) error {
			_, err := s.KVDelete(context.Background(), &pluginv1.KVDeleteRequest{InstallationId: "bad"})
			return err
		}},
		{"list invalid id", func(s *PluginGRPCServer) error {
			_, err := s.KVList(context.Background(), &pluginv1.KVListRequest{InstallationId: "bad"})
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repos := newPluginTestRepos()
			srv := newPluginTestServer(repos)
			requireGRPCCode(t, tc.call(srv), codes.InvalidArgument)
		})
	}

	t.Run("repository errors surface as Internal", func(t *testing.T) {
		repos := newPluginTestRepos()
		repos.kv.getErr = errors.New("down")
		repos.kv.setErr = errors.New("down")
		repos.kv.delErr = errors.New("down")
		repos.kv.listErr = errors.New("down")
		srv := newPluginTestServer(repos)
		instID := uuid.New().String()

		_, err := srv.KVGet(context.Background(), &pluginv1.KVGetRequest{InstallationId: instID})
		requireGRPCCode(t, err, codes.Internal)
		_, err = srv.KVSet(context.Background(), &pluginv1.KVSetRequest{InstallationId: instID})
		requireGRPCCode(t, err, codes.Internal)
		_, err = srv.KVDelete(context.Background(), &pluginv1.KVDeleteRequest{InstallationId: instID})
		requireGRPCCode(t, err, codes.Internal)
		_, err = srv.KVList(context.Background(), &pluginv1.KVListRequest{InstallationId: instID})
		requireGRPCCode(t, err, codes.Internal)
	})
}

// ============================================================================
// Hook execution
// ============================================================================

func TestPluginExecuteHooks(t *testing.T) {
	t.Run("no active installations reports not modified", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)

		resp, err := srv.ExecuteHooks(context.Background(), &pluginv1.ExecuteHooksRequest{TenantId: uuid.New().String(), HookType: "before_save"})
		requireGRPCOK(t, err)
		require.False(t, resp.GetModified())
		require.Empty(t, resp.GetResults())
	})

	t.Run("active installations without validation errors return entity data and results", func(t *testing.T) {
		repos := newPluginTestRepos()
		manifestID := uuid.New()
		repos.installations.activeByHook = []*models.PluginInstallation{{ID: uuid.New(), ManifestID: manifestID}}
		srv := newPluginTestServer(repos)

		data, err := structpb.NewStruct(map[string]any{"name": "Acme"})
		require.NoError(t, err)

		resp, err := srv.ExecuteHooks(context.Background(), &pluginv1.ExecuteHooksRequest{
			TenantId: uuid.New().String(), HookType: "before_save", EntityData: data,
		})
		requireGRPCOK(t, err)
		require.False(t, resp.GetModified())
		require.Len(t, resp.GetResults(), 1)
		require.Equal(t, manifestID.String(), resp.GetResults()[0].GetPluginSlug())
		require.NotNil(t, resp.GetModifiedData())
	})

	t.Run("validation errors suppress modified data but keep results", func(t *testing.T) {
		repos := newPluginTestRepos()
		tenantID := uuid.New()
		repos.installations.activeByHook = []*models.PluginInstallation{{ID: uuid.New(), ManifestID: uuid.New()}}
		repos.validationRules.byID[uuid.New()] = &models.ValidationRule{
			ID: uuid.New(), TenantID: tenantID, EntityType: "contact", FieldName: "email",
			RuleType: models.ValidationRuleTypeFormat, RuleConfig: json.RawMessage(`{"format":"email","required":true}`),
			Enabled: true, ErrorMessage: "invalid email",
		}
		srv := newPluginTestServer(repos)

		resp, err := srv.ExecuteHooks(context.Background(), &pluginv1.ExecuteHooksRequest{
			TenantId: tenantID.String(), HookType: "before_save", EntityType: "contact",
		})
		requireGRPCOK(t, err)
		require.False(t, resp.GetModified())
		require.Nil(t, resp.GetModifiedData())
		require.Len(t, resp.GetResults(), 1)
	})

	t.Run("repository error is swallowed and reported as not modified, not a gRPC error", func(t *testing.T) {
		repos := newPluginTestRepos()
		repos.installations.hookErr = errors.New("db down")
		srv := newPluginTestServer(repos)

		resp, err := srv.ExecuteHooks(context.Background(), &pluginv1.ExecuteHooksRequest{TenantId: uuid.New().String()})
		requireGRPCOK(t, err)
		require.False(t, resp.GetModified())
	})

	t.Run("invalid tenant_id", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.ExecuteHooks(context.Background(), &pluginv1.ExecuteHooksRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestPluginValidateEntity(t *testing.T) {
	t.Run("no rules is valid", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)

		resp, err := srv.ValidateEntity(context.Background(), &pluginv1.ValidateEntityRequest{TenantId: uuid.New().String(), EntityType: "contact"})
		requireGRPCOK(t, err)
		require.True(t, resp.GetValid())
		require.Empty(t, resp.GetErrors())
	})

	t.Run("nil entity_data does not panic", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)

		resp, err := srv.ValidateEntity(context.Background(), &pluginv1.ValidateEntityRequest{TenantId: uuid.New().String(), EntityType: "contact"})
		requireGRPCOK(t, err)
		require.True(t, resp.GetValid())
	})

	t.Run("failing rule reports field errors", func(t *testing.T) {
		repos := newPluginTestRepos()
		tenantID := uuid.New()
		repos.validationRules.byID[uuid.New()] = &models.ValidationRule{
			ID: uuid.New(), TenantID: tenantID, Name: "iban-check", EntityType: "contact", FieldName: "iban",
			RuleType: models.ValidationRuleTypeFormat, RuleConfig: json.RawMessage(`{"format":"iban","required":true}`),
			Enabled: true, ErrorMessage: "invalid IBAN",
		}
		srv := newPluginTestServer(repos)

		data, err := structpb.NewStruct(map[string]any{"iban": "not-an-iban"})
		require.NoError(t, err)

		resp, err := srv.ValidateEntity(context.Background(), &pluginv1.ValidateEntityRequest{
			TenantId: tenantID.String(), EntityType: "contact", EntityData: data,
		})
		requireGRPCOK(t, err)
		require.False(t, resp.GetValid())
		require.Len(t, resp.GetErrors(), 1)
		require.Equal(t, "iban", resp.GetErrors()[0].GetField())
		require.Equal(t, "invalid IBAN", resp.GetErrors()[0].GetMessage())
	})

	t.Run("invalid tenant_id", func(t *testing.T) {
		repos := newPluginTestRepos()
		srv := newPluginTestServer(repos)
		_, err := srv.ValidateEntity(context.Background(), &pluginv1.ValidateEntityRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}
