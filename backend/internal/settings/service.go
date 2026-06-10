package settings

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// RoleChecker is a narrow interface to determine whether a user has admin role.
// The concrete implementation queries the user_roles table directly — roles are
// not propagated through gRPC metadata, so we cannot rely on JWT claims here.
type RoleChecker interface {
	IsAdmin(ctx context.Context, tenantID, userID uuid.UUID) (bool, error)
}

// Service contains all settings business logic.
// It is the only layer that enforces RBAC rules beyond what the gateway's
// RequirePermission guard provides (the guard allows all authenticated users
// to call settings:write; the service then enforces the module-lead restriction
// for tenant-scope writes specifically).
type Service struct {
	repo        Repository
	roleChecker RoleChecker
}

// NewService creates a new Settings service.
func NewService(repo Repository, roleChecker RoleChecker) *Service {
	return &Service{repo: repo, roleChecker: roleChecker}
}

// ============================================================================
// Module-lead management
// ============================================================================

// ListModuleLeads returns module-lead assignments, optionally filtered by user or module.
func (s *Service) ListModuleLeads(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, moduleID *string) ([]*ModuleLead, error) {
	return s.repo.ListModuleLeads(ctx, tenantID, userID, moduleID)
}

// GrantModuleLead assigns module-lead rights. Only admins may do this.
func (s *Service) GrantModuleLead(ctx context.Context, tenantID, callerID, targetUserID uuid.UUID, moduleID string) (*ModuleLead, error) {
	if moduleID == "" {
		return nil, ErrInvalidModuleID
	}

	isAdmin, err := s.roleChecker.IsAdmin(ctx, tenantID, callerID)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, ErrNotModuleLead
	}

	ml, err := s.repo.GrantModuleLead(ctx, tenantID, targetUserID, moduleID, &callerID)
	if err != nil {
		return nil, err
	}
	slog.Info("module lead granted",
		"tenant_id", tenantID,
		"user_id", targetUserID,
		"module_id", moduleID,
		"granted_by", callerID,
	)
	return ml, nil
}

// RevokeModuleLead removes module-lead rights. Only admins may do this.
func (s *Service) RevokeModuleLead(ctx context.Context, tenantID, callerID, targetUserID uuid.UUID, moduleID string) error {
	if moduleID == "" {
		return ErrInvalidModuleID
	}

	isAdmin, err := s.roleChecker.IsAdmin(ctx, tenantID, callerID)
	if err != nil {
		return err
	}
	if !isAdmin {
		return ErrNotModuleLead
	}

	if err := s.repo.RevokeModuleLead(ctx, tenantID, targetUserID, moduleID); err != nil {
		return err
	}
	slog.Info("module lead revoked",
		"tenant_id", tenantID,
		"user_id", targetUserID,
		"module_id", moduleID,
		"revoked_by", callerID,
	)
	return nil
}

// GetMyModuleLeads returns the list of modules the caller leads.
// Admins implicitly lead all modules — the response includes is_admin=true
// so the FE does not need to hard-code the admin fast-path.
func (s *Service) GetMyModuleLeads(ctx context.Context, tenantID, userID uuid.UUID) ([]string, bool, error) {
	isAdmin, err := s.roleChecker.IsAdmin(ctx, tenantID, userID)
	if err != nil {
		return nil, false, err
	}
	if isAdmin {
		// Admin leads all — return empty list + is_admin flag so FE can short-circuit.
		return nil, true, nil
	}

	moduleIDs, err := s.repo.ListLeadModulesForUser(ctx, tenantID, userID)
	if err != nil {
		return nil, false, err
	}
	return moduleIDs, false, nil
}

// ============================================================================
// Settings management
// ============================================================================

// GetResolvedSettings returns the merged view for a user in a module:
// user overrides win over tenant defaults. Keys absent at both levels are omitted.
func (s *Service) GetResolvedSettings(ctx context.Context, tenantID, userID uuid.UUID, moduleID string) ([]*SettingEntry, error) {
	if moduleID == "" {
		return nil, ErrInvalidModuleID
	}

	tenantEntries, err := s.repo.GetTenantSettings(ctx, tenantID, moduleID)
	if err != nil {
		return nil, err
	}
	userEntries, err := s.repo.GetUserSettings(ctx, tenantID, userID, moduleID)
	if err != nil {
		return nil, err
	}

	// Build a map from tenant settings, then overlay user settings.
	resolved := make(map[string]*SettingEntry, len(tenantEntries)+len(userEntries))
	for _, e := range tenantEntries {
		resolved[e.Key] = e
	}
	for _, e := range userEntries {
		resolved[e.Key] = e // user wins
	}

	result := make([]*SettingEntry, 0, len(resolved))
	for _, e := range resolved {
		result = append(result, e)
	}
	return result, nil
}

// GetTenantSettings returns the raw tenant-scope settings for a module.
func (s *Service) GetTenantSettings(ctx context.Context, tenantID uuid.UUID, moduleID string) ([]*SettingEntry, error) {
	if moduleID == "" {
		return nil, ErrInvalidModuleID
	}
	return s.repo.GetTenantSettings(ctx, tenantID, moduleID)
}

// PutTenantSettings writes tenant-scope settings. Caller must be admin or module-lead.
func (s *Service) PutTenantSettings(ctx context.Context, tenantID, callerID uuid.UUID, moduleID string, entries []*SettingEntry) ([]*SettingEntry, error) {
	if moduleID == "" {
		return nil, ErrInvalidModuleID
	}
	for _, e := range entries {
		if e.Key == "" {
			return nil, ErrInvalidKey
		}
	}

	// RBAC: admin or module-lead
	allowed, err := s.callerMayWriteTenantSettings(ctx, tenantID, callerID, moduleID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrNotModuleLead
	}

	return s.repo.PutTenantSettings(ctx, tenantID, moduleID, &callerID, entries)
}

// GetUserSettings returns user-scope settings for a module.
func (s *Service) GetUserSettings(ctx context.Context, tenantID, userID uuid.UUID, moduleID string) ([]*SettingEntry, error) {
	if moduleID == "" {
		return nil, ErrInvalidModuleID
	}
	return s.repo.GetUserSettings(ctx, tenantID, userID, moduleID)
}

// PutUserSettings writes user-scope settings. A user may only write their own settings.
// The gateway enforces that callerID == targetUserID via middleware.GetUserID before
// invoking this method.
func (s *Service) PutUserSettings(ctx context.Context, tenantID, userID uuid.UUID, moduleID string, entries []*SettingEntry) ([]*SettingEntry, error) {
	if moduleID == "" {
		return nil, ErrInvalidModuleID
	}
	for _, e := range entries {
		if e.Key == "" {
			return nil, ErrInvalidKey
		}
	}
	return s.repo.PutUserSettings(ctx, tenantID, userID, moduleID, entries)
}

// ============================================================================
// Internal helpers
// ============================================================================

// callerMayWriteTenantSettings returns true if the caller is an admin or a
// Modul-Leiter for the given module in the tenant.
func (s *Service) callerMayWriteTenantSettings(ctx context.Context, tenantID, callerID uuid.UUID, moduleID string) (bool, error) {
	isAdmin, err := s.roleChecker.IsAdmin(ctx, tenantID, callerID)
	if err != nil {
		return false, err
	}
	if isAdmin {
		return true, nil
	}
	return s.repo.IsModuleLead(ctx, tenantID, callerID, moduleID)
}
