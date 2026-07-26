package settings

import (
	"context"

	"github.com/google/uuid"
)

// Repository is the storage abstraction for settings and module-leads.
type Repository interface {
	// Module-leads
	ListModuleLeads(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, moduleID *string) ([]*ModuleLead, error)
	GetModuleLead(ctx context.Context, tenantID, userID uuid.UUID, moduleID string) (*ModuleLead, error)
	GrantModuleLead(ctx context.Context, tenantID, userID uuid.UUID, moduleID string, grantedBy *uuid.UUID) (*ModuleLead, error)
	RevokeModuleLead(ctx context.Context, tenantID, userID uuid.UUID, moduleID string) error
	IsModuleLead(ctx context.Context, tenantID, userID uuid.UUID, moduleID string) (bool, error)
	ListLeadModulesForUser(ctx context.Context, tenantID, userID uuid.UUID) ([]string, error)

	// Module-access grants
	ListModuleGrants(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, moduleID *string) ([]*UserModuleGrant, error)
	GrantModuleAccess(ctx context.Context, tenantID, userID uuid.UUID, moduleID string, grantedBy *uuid.UUID) (*UserModuleGrant, error)
	RevokeModuleAccess(ctx context.Context, tenantID, userID uuid.UUID, moduleID string) error
	BulkRevokeModuleAccess(ctx context.Context, tenantID uuid.UUID, refs []ModuleGrantRef) (int, error)

	// Tenant settings
	GetTenantSettings(ctx context.Context, tenantID uuid.UUID, moduleID string) ([]*SettingEntry, error)
	PutTenantSettings(ctx context.Context, tenantID uuid.UUID, moduleID string, updatedBy *uuid.UUID, entries []*SettingEntry) ([]*SettingEntry, error)

	// User settings
	GetUserSettings(ctx context.Context, tenantID, userID uuid.UUID, moduleID string) ([]*SettingEntry, error)
	PutUserSettings(ctx context.Context, tenantID, userID uuid.UUID, moduleID string, entries []*SettingEntry) ([]*SettingEntry, error)

	// Licensing
	ListModuleActivations(ctx context.Context, tenantID uuid.UUID) (map[string]bool, error)
	SetModuleActivation(ctx context.Context, tenantID uuid.UUID, moduleID string, active bool, updatedBy *uuid.UUID) error
	CountGrantsByModule(ctx context.Context, tenantID uuid.UUID) (map[string]int32, error)
	GetTenantSubscription(ctx context.Context, tenantID uuid.UUID) (*TenantSubscription, error)
}
