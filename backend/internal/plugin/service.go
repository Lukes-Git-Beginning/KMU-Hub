package plugin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/plugin/config"
)

// Service handles plugin business logic
type Service struct {
	manifests        ManifestRepo
	installations    InstallationRepo
	permissions      PermissionRepo
	kvStore          KVStoreRepo
	executionLog     ExecutionLogRepo
	validationRules  ValidationRuleRepo
	workflowRules    WorkflowRuleRepo
	templates        IndustryTemplateRepo
	schemaValidator  *config.SchemaValidator
	validationEngine *config.ValidationEngine
	workflowEngine   *config.WorkflowEngine
}

// NewService creates a new plugin service
func NewService(
	manifests ManifestRepo,
	installations InstallationRepo,
	permissions PermissionRepo,
	kvStore KVStoreRepo,
	executionLog ExecutionLogRepo,
	validationRules ValidationRuleRepo,
	workflowRules WorkflowRuleRepo,
	templates IndustryTemplateRepo,
) *Service {
	return &Service{
		manifests:        manifests,
		installations:    installations,
		permissions:      permissions,
		kvStore:          kvStore,
		executionLog:     executionLog,
		validationRules:  validationRules,
		workflowRules:    workflowRules,
		templates:        templates,
		schemaValidator:  config.NewSchemaValidator(),
		validationEngine: config.NewValidationEngine(),
		workflowEngine:   config.NewWorkflowEngine(),
	}
}

// ============================================================================
// Manifest CRUD
// ============================================================================

// CreateManifest creates a new plugin manifest
func (s *Service) CreateManifest(ctx context.Context, m *models.PluginManifest) (*models.PluginManifest, error) {
	existing, _ := s.manifests.GetBySlug(ctx, m.Slug)
	if existing != nil {
		return nil, ErrManifestSlugExists
	}

	if m.PluginType == models.PluginTypeWASM && len(m.WASMBinary) == 0 {
		return nil, ErrWASMBinaryRequired
	}
	if m.PluginType == models.PluginTypeConfig && len(m.WASMBinary) > 0 {
		return nil, ErrWASMBinaryNotAllowed
	}

	m.ID = uuid.New()
	now := time.Now()
	m.CreatedAt = now
	m.UpdatedAt = now

	if len(m.WASMBinary) > 0 {
		hash := sha256.Sum256(m.WASMBinary)
		m.WASMBinaryHash = hex.EncodeToString(hash[:])
	}
	if m.SettingsSchema == nil {
		m.SettingsSchema = json.RawMessage("{}")
	}
	if m.HookRegistrations == nil {
		m.HookRegistrations = []models.HookRegistration{}
	}
	if m.Permissions == nil {
		m.Permissions = []string{}
	}

	if err := s.manifests.Create(ctx, m); err != nil {
		return nil, fmt.Errorf("create manifest: %w", err)
	}

	slog.Info("plugin manifest created",
		"manifest_id", m.ID,
		"slug", m.Slug,
		"type", m.PluginType,
	)

	return m, nil
}

// GetManifest retrieves a plugin manifest by ID
func (s *Service) GetManifest(ctx context.Context, id uuid.UUID) (*models.PluginManifest, error) {
	m, err := s.manifests.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrManifestNotFound
	}
	return m, nil
}

// ListManifests retrieves all plugin manifests
func (s *Service) ListManifests(ctx context.Context, pluginType string) ([]*models.PluginManifest, error) {
	return s.manifests.List(ctx, pluginType)
}

// DeleteManifest removes a plugin manifest
func (s *Service) DeleteManifest(ctx context.Context, id uuid.UUID) error {
	m, err := s.manifests.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrManifestNotFound
	}

	hasInstalls, err := s.manifests.HasActiveInstallations(ctx, id)
	if err != nil {
		return err
	}
	if hasInstalls {
		return ErrPluginHasInstallations
	}

	slog.Info("plugin manifest deleted", "manifest_id", id, "slug", m.Slug)
	return s.manifests.Delete(ctx, id)
}

// ============================================================================
// Installation Lifecycle
// ============================================================================

// InstallPlugin installs a plugin for a tenant
func (s *Service) InstallPlugin(ctx context.Context, tenantID, manifestID, installedBy uuid.UUID) (*models.PluginInstallation, error) {
	m, err := s.manifests.GetByID(ctx, manifestID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrManifestNotFound
	}

	existing, _ := s.installations.GetByTenantAndManifest(ctx, tenantID, manifestID)
	if existing != nil && existing.Status != models.InstallationStatusUninstalled {
		return nil, ErrAlreadyInstalled
	}

	inst := &models.PluginInstallation{
		ID:          uuid.New(),
		TenantID:    tenantID,
		ManifestID:  manifestID,
		Status:      models.InstallationStatusPendingApproval,
		Settings:    json.RawMessage("{}"),
		InstalledBy: installedBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.installations.Create(ctx, inst); err != nil {
		return nil, fmt.Errorf("create installation: %w", err)
	}

	slog.Info("plugin installed",
		"installation_id", inst.ID,
		"manifest_slug", m.Slug,
		"tenant_id", tenantID,
	)

	return inst, nil
}

// GetInstallation retrieves a plugin installation by ID
func (s *Service) GetInstallation(ctx context.Context, id uuid.UUID) (*models.PluginInstallation, error) {
	inst, err := s.installations.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if inst == nil {
		return nil, ErrInstallationNotFound
	}
	return inst, nil
}

// ListInstallations retrieves all installations for a tenant
func (s *Service) ListInstallations(ctx context.Context, tenantID uuid.UUID, status string) ([]*models.PluginInstallationWithManifest, error) {
	installations, err := s.installations.List(ctx, tenantID, status)
	if err != nil {
		return nil, err
	}

	// Enrich with granted permissions
	for _, inst := range installations {
		granted, _ := s.permissions.ListByInstallation(ctx, inst.ID)
		inst.Granted = granted
	}

	return installations, nil
}

// EnablePlugin activates a plugin installation
func (s *Service) EnablePlugin(ctx context.Context, id uuid.UUID) (*models.PluginInstallation, error) {
	inst, err := s.installations.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if inst == nil {
		return nil, ErrInstallationNotFound
	}

	if err := s.installations.UpdateStatus(ctx, id, models.InstallationStatusActive, nil); err != nil {
		return nil, err
	}
	inst.Status = models.InstallationStatusActive

	slog.Info("plugin enabled", "installation_id", id)
	return inst, nil
}

// DisablePlugin deactivates a plugin installation
func (s *Service) DisablePlugin(ctx context.Context, id uuid.UUID) (*models.PluginInstallation, error) {
	inst, err := s.installations.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if inst == nil {
		return nil, ErrInstallationNotFound
	}

	if err := s.installations.UpdateStatus(ctx, id, models.InstallationStatusDisabled, nil); err != nil {
		return nil, err
	}
	inst.Status = models.InstallationStatusDisabled

	slog.Info("plugin disabled", "installation_id", id)
	return inst, nil
}

// UninstallPlugin marks a plugin installation as uninstalled
func (s *Service) UninstallPlugin(ctx context.Context, id uuid.UUID) error {
	inst, err := s.installations.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if inst == nil {
		return ErrInstallationNotFound
	}

	if err := s.permissions.RevokeAll(ctx, id); err != nil {
		return err
	}

	slog.Info("plugin uninstalled", "installation_id", id)
	return s.installations.UpdateStatus(ctx, id, models.InstallationStatusUninstalled, nil)
}

// ============================================================================
// Permissions
// ============================================================================

// ApprovePermissions grants permissions to a plugin installation
func (s *Service) ApprovePermissions(ctx context.Context, installationID uuid.UUID, permissions []string, grantedBy uuid.UUID) error {
	inst, err := s.installations.GetByID(ctx, installationID)
	if err != nil {
		return err
	}
	if inst == nil {
		return ErrInstallationNotFound
	}

	// Verify all permissions are declared in manifest
	manifest, err := s.manifests.GetByID(ctx, inst.ManifestID)
	if err != nil {
		return err
	}
	declared := make(map[string]bool)
	for _, p := range manifest.Permissions {
		declared[p] = true
	}
	for _, p := range permissions {
		if !declared[p] {
			return fmt.Errorf("%w: %s", ErrPermissionNotDeclared, p)
		}
	}

	now := time.Now()
	for _, perm := range permissions {
		if grantErr := s.permissions.Grant(ctx, &models.PluginPermission{
			ID:             uuid.New(),
			InstallationID: installationID,
			Permission:     perm,
			GrantedBy:      grantedBy,
			GrantedAt:      now,
		}); grantErr != nil {
			return grantErr
		}
	}

	slog.Info("plugin permissions approved",
		"installation_id", installationID,
		"permissions", permissions,
	)

	return nil
}

// ListGrantedPermissions retrieves granted permissions for an installation
func (s *Service) ListGrantedPermissions(ctx context.Context, installationID uuid.UUID) ([]string, error) {
	return s.permissions.ListByInstallation(ctx, installationID)
}

// HasPermission checks if a plugin has a specific permission
func (s *Service) HasPermission(ctx context.Context, installationID uuid.UUID, permission string) (bool, error) {
	return s.permissions.HasPermission(ctx, installationID, permission)
}

// ============================================================================
// Settings
// ============================================================================

// GetPluginSettings retrieves settings for a plugin installation
func (s *Service) GetPluginSettings(ctx context.Context, installationID uuid.UUID) (json.RawMessage, error) {
	inst, err := s.installations.GetByID(ctx, installationID)
	if err != nil {
		return nil, err
	}
	if inst == nil {
		return nil, ErrInstallationNotFound
	}
	return inst.Settings, nil
}

// UpdatePluginSettings updates settings for a plugin installation
func (s *Service) UpdatePluginSettings(ctx context.Context, installationID uuid.UUID, settings json.RawMessage) error {
	inst, err := s.installations.GetByID(ctx, installationID)
	if err != nil {
		return err
	}
	if inst == nil {
		return ErrInstallationNotFound
	}

	// Validate against schema
	manifest, err := s.manifests.GetByID(ctx, inst.ManifestID)
	if err != nil {
		return err
	}
	result := s.schemaValidator.Validate(manifest.SettingsSchema, settings)
	if !result.Valid {
		return fmt.Errorf("%w: %v", ErrInvalidSettings, result.Errors)
	}

	return s.installations.UpdateSettings(ctx, installationID, settings)
}

// GetPluginSettingsSchema retrieves the settings schema for a plugin installation
func (s *Service) GetPluginSettingsSchema(ctx context.Context, installationID uuid.UUID) (json.RawMessage, error) {
	inst, err := s.installations.GetByID(ctx, installationID)
	if err != nil {
		return nil, err
	}
	if inst == nil {
		return nil, ErrInstallationNotFound
	}

	manifest, err := s.manifests.GetByID(ctx, inst.ManifestID)
	if err != nil {
		return nil, err
	}
	return manifest.SettingsSchema, nil
}

// ============================================================================
// Validation Rules
// ============================================================================

// CreateValidationRule creates a new validation rule
func (s *Service) CreateValidationRule(ctx context.Context, rule *models.ValidationRule) (*models.ValidationRule, error) {
	rule.ID = uuid.New()
	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now
	rule.Enabled = true

	if err := s.validationRules.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("create validation rule: %w", err)
	}

	slog.Info("validation rule created",
		"rule_id", rule.ID,
		"name", rule.Name,
		"entity_type", rule.EntityType,
	)

	return rule, nil
}

// ListValidationRules retrieves validation rules for a tenant
func (s *Service) ListValidationRules(ctx context.Context, tenantID uuid.UUID, entityType string) ([]*models.ValidationRule, error) {
	return s.validationRules.List(ctx, tenantID, entityType)
}

// UpdateValidationRule updates a validation rule
func (s *Service) UpdateValidationRule(ctx context.Context, rule *models.ValidationRule) (*models.ValidationRule, error) {
	existing, err := s.validationRules.GetByID(ctx, rule.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrValidationRuleNotFound
	}

	existing.Name = rule.Name
	existing.Description = rule.Description
	existing.RuleConfig = rule.RuleConfig
	existing.ErrorMessage = rule.ErrorMessage
	existing.Priority = rule.Priority
	existing.Enabled = rule.Enabled
	existing.UpdatedAt = time.Now()

	if err := s.validationRules.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

// DeleteValidationRule removes a validation rule
func (s *Service) DeleteValidationRule(ctx context.Context, id uuid.UUID) error {
	existing, err := s.validationRules.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrValidationRuleNotFound
	}
	return s.validationRules.Delete(ctx, id)
}

// ValidateEntity validates entity data against configured rules
func (s *Service) ValidateEntity(ctx context.Context, tenantID uuid.UUID, entityType string, data map[string]any) []config.FieldError {
	rules, err := s.validationRules.ListEnabled(ctx, tenantID, entityType)
	if err != nil {
		slog.Warn("failed to load validation rules", "error", err)
		return nil
	}
	return s.validationEngine.ValidateEntity(rules, data)
}

// ============================================================================
// Workflow Rules
// ============================================================================

// CreateWorkflowRule creates a new workflow rule
func (s *Service) CreateWorkflowRule(ctx context.Context, rule *models.WorkflowRule) (*models.WorkflowRule, error) {
	rule.ID = uuid.New()
	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now
	rule.Enabled = true

	if err := s.workflowRules.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("create workflow rule: %w", err)
	}

	slog.Info("workflow rule created",
		"rule_id", rule.ID,
		"name", rule.Name,
		"trigger_event", rule.TriggerEvent,
	)

	return rule, nil
}

// ListWorkflowRules retrieves workflow rules for a tenant
func (s *Service) ListWorkflowRules(ctx context.Context, tenantID uuid.UUID, triggerEvent string) ([]*models.WorkflowRule, error) {
	return s.workflowRules.List(ctx, tenantID, triggerEvent)
}

// UpdateWorkflowRule updates a workflow rule
func (s *Service) UpdateWorkflowRule(ctx context.Context, rule *models.WorkflowRule) (*models.WorkflowRule, error) {
	existing, err := s.workflowRules.GetByID(ctx, rule.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrWorkflowRuleNotFound
	}

	existing.Name = rule.Name
	existing.Description = rule.Description
	existing.Conditions = rule.Conditions
	existing.Actions = rule.Actions
	existing.Priority = rule.Priority
	existing.Enabled = rule.Enabled
	existing.UpdatedAt = time.Now()

	if err := s.workflowRules.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

// DeleteWorkflowRule removes a workflow rule
func (s *Service) DeleteWorkflowRule(ctx context.Context, id uuid.UUID) error {
	existing, err := s.workflowRules.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrWorkflowRuleNotFound
	}
	return s.workflowRules.Delete(ctx, id)
}

// EvaluateWorkflows evaluates workflow rules for a given event
func (s *Service) EvaluateWorkflows(ctx context.Context, tenantID uuid.UUID, eventType string, data map[string]any) []config.TriggeredAction {
	rules, err := s.workflowRules.ListEnabled(ctx, tenantID, eventType)
	if err != nil {
		slog.Warn("failed to load workflow rules", "error", err)
		return nil
	}
	return s.workflowEngine.EvaluateWorkflows(rules, eventType, data)
}

// ============================================================================
// Industry Templates
// ============================================================================

// ListIndustryTemplates retrieves available industry templates
func (s *Service) ListIndustryTemplates(ctx context.Context, industry string) ([]*models.IndustryTemplate, error) {
	return s.templates.List(ctx, industry)
}

// ApplyIndustryTemplate applies an industry template to a tenant
func (s *Service) ApplyIndustryTemplate(ctx context.Context, tenantID uuid.UUID, templateID, appliedBy uuid.UUID) error {
	tmpl, err := s.templates.GetByID(ctx, templateID)
	if err != nil {
		return err
	}
	if tmpl == nil {
		return ErrTemplateNotFound
	}

	// Apply validation rules from template
	var validationRules []models.ValidationRule
	if err := json.Unmarshal(tmpl.ValidationRules, &validationRules); err == nil {
		for i := range validationRules {
			validationRules[i].TenantID = tenantID
			if _, createErr := s.CreateValidationRule(ctx, &validationRules[i]); createErr != nil {
				slog.Warn("failed to create validation rule from template",
					"rule_name", validationRules[i].Name,
					"error", createErr,
				)
			}
		}
	}

	// Apply workflow rules from template
	var workflowRules []models.WorkflowRule
	if err := json.Unmarshal(tmpl.WorkflowRules, &workflowRules); err == nil {
		for i := range workflowRules {
			workflowRules[i].TenantID = tenantID
			if _, createErr := s.CreateWorkflowRule(ctx, &workflowRules[i]); createErr != nil {
				slog.Warn("failed to create workflow rule from template",
					"rule_name", workflowRules[i].Name,
					"error", createErr,
				)
			}
		}
	}

	slog.Info("industry template applied",
		"template_slug", tmpl.Slug,
		"tenant_id", tenantID,
	)

	return nil
}

// ============================================================================
// KV Store
// ============================================================================

// KVGet retrieves a value from the plugin KV store
func (s *Service) KVGet(ctx context.Context, installationID uuid.UUID, key string) (json.RawMessage, bool, error) {
	return s.kvStore.Get(ctx, installationID, key)
}

// KVSet stores a value in the plugin KV store
func (s *Service) KVSet(ctx context.Context, installationID uuid.UUID, key string, value json.RawMessage) error {
	return s.kvStore.Set(ctx, installationID, key, value)
}

// KVDelete removes a value from the plugin KV store
func (s *Service) KVDelete(ctx context.Context, installationID uuid.UUID, key string) error {
	return s.kvStore.Delete(ctx, installationID, key)
}

// KVList retrieves all key-value pairs for a plugin with optional prefix filter
func (s *Service) KVList(ctx context.Context, installationID uuid.UUID, keyPrefix string) (map[string]json.RawMessage, error) {
	return s.kvStore.List(ctx, installationID, keyPrefix)
}

// ============================================================================
// Execution Logs
// ============================================================================

// LogExecution logs a hook execution
func (s *Service) LogExecution(ctx context.Context, log *models.PluginExecutionLog) error {
	log.ID = uuid.New()
	log.CreatedAt = time.Now()
	return s.executionLog.Create(ctx, log)
}

// ListExecutionLogs retrieves execution logs
func (s *Service) ListExecutionLogs(ctx context.Context, tenantID uuid.UUID, installationID *uuid.UUID, limit int) ([]*models.PluginExecutionLog, error) {
	return s.executionLog.List(ctx, tenantID, installationID, limit)
}

// ============================================================================
// Hook Execution
// ============================================================================

// ListActiveInstallationsForHook finds active plugins that registered for a specific hook
func (s *Service) ListActiveInstallationsForHook(ctx context.Context, tenantID uuid.UUID, hookType, module, entityType string) ([]*models.PluginInstallation, error) {
	return s.installations.ListActiveByHook(ctx, tenantID, hookType, module, entityType)
}

// GetManifestForInstallation retrieves the manifest for an installation
func (s *Service) GetManifestForInstallation(ctx context.Context, installationID uuid.UUID) (*models.PluginManifest, error) {
	inst, err := s.installations.GetByID(ctx, installationID)
	if err != nil {
		return nil, err
	}
	if inst == nil {
		return nil, ErrInstallationNotFound
	}
	return s.manifests.GetByID(ctx, inst.ManifestID)
}

// GetWASMBinary retrieves the WASM binary for a manifest
func (s *Service) GetWASMBinary(ctx context.Context, manifestID uuid.UUID) ([]byte, error) {
	return s.manifests.GetWASMBinary(ctx, manifestID)
}
