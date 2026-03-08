package plugin

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ManifestRepo defines the interface for plugin manifest persistence
type ManifestRepo interface {
	Create(ctx context.Context, m *models.PluginManifest) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.PluginManifest, error)
	GetBySlug(ctx context.Context, slug string) (*models.PluginManifest, error)
	List(ctx context.Context, pluginType string) ([]*models.PluginManifest, error)
	Delete(ctx context.Context, id uuid.UUID) error
	HasActiveInstallations(ctx context.Context, id uuid.UUID) (bool, error)
	GetWASMBinary(ctx context.Context, id uuid.UUID) ([]byte, error)
}

// InstallationRepo defines the interface for plugin installation persistence
type InstallationRepo interface {
	Create(ctx context.Context, inst *models.PluginInstallation) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.PluginInstallation, error)
	GetByTenantAndManifest(ctx context.Context, tenantID, manifestID uuid.UUID) (*models.PluginInstallation, error)
	List(ctx context.Context, tenantID uuid.UUID, status string) ([]*models.PluginInstallationWithManifest, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.InstallationStatus, errorMsg *string) error
	UpdateSettings(ctx context.Context, id uuid.UUID, settings []byte) error
	ListActiveByHook(ctx context.Context, tenantID uuid.UUID, hookType, module, entityType string) ([]*models.PluginInstallation, error)
}

// PermissionRepo defines the interface for plugin permission persistence
type PermissionRepo interface {
	Grant(ctx context.Context, perm *models.PluginPermission) error
	RevokeAll(ctx context.Context, installationID uuid.UUID) error
	ListByInstallation(ctx context.Context, installationID uuid.UUID) ([]string, error)
	HasPermission(ctx context.Context, installationID uuid.UUID, permission string) (bool, error)
}

// KVStoreRepo defines the interface for plugin key-value store persistence
type KVStoreRepo interface {
	Get(ctx context.Context, installationID uuid.UUID, key string) (json.RawMessage, bool, error)
	Set(ctx context.Context, installationID uuid.UUID, key string, value json.RawMessage) error
	Delete(ctx context.Context, installationID uuid.UUID, key string) error
	List(ctx context.Context, installationID uuid.UUID, keyPrefix string) (map[string]json.RawMessage, error)
}

// ExecutionLogRepo defines the interface for plugin execution log persistence
type ExecutionLogRepo interface {
	Create(ctx context.Context, log *models.PluginExecutionLog) error
	List(ctx context.Context, tenantID uuid.UUID, installationID *uuid.UUID, limit int) ([]*models.PluginExecutionLog, error)
}

// ValidationRuleRepo defines the interface for validation rule persistence
type ValidationRuleRepo interface {
	Create(ctx context.Context, rule *models.ValidationRule) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.ValidationRule, error)
	Update(ctx context.Context, rule *models.ValidationRule) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, entityType string) ([]*models.ValidationRule, error)
	ListEnabled(ctx context.Context, tenantID uuid.UUID, entityType string) ([]*models.ValidationRule, error)
}

// WorkflowRuleRepo defines the interface for workflow rule persistence
type WorkflowRuleRepo interface {
	Create(ctx context.Context, rule *models.WorkflowRule) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.WorkflowRule, error)
	Update(ctx context.Context, rule *models.WorkflowRule) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, triggerEvent string) ([]*models.WorkflowRule, error)
	ListEnabled(ctx context.Context, tenantID uuid.UUID, triggerEvent string) ([]*models.WorkflowRule, error)
}

// IndustryTemplateRepo defines the interface for industry template persistence
type IndustryTemplateRepo interface {
	List(ctx context.Context, industry string) ([]*models.IndustryTemplate, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.IndustryTemplate, error)
}
