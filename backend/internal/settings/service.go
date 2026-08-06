package settings

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/modules"
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
// Module-access grant management
// ============================================================================

// ListModuleGrants returns access grants, optionally filtered by user or module.
func (s *Service) ListModuleGrants(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, moduleID *string) ([]*UserModuleGrant, error) {
	return s.repo.ListModuleGrants(ctx, tenantID, userID, moduleID)
}

// GrantModuleAccess gives a user access to a module. Only admins may do this.
func (s *Service) GrantModuleAccess(ctx context.Context, tenantID, callerID, targetUserID uuid.UUID, moduleID string) (*UserModuleGrant, error) {
	if moduleID == "" {
		return nil, ErrInvalidModuleID
	}
	if err := s.requireAdmin(ctx, tenantID, callerID); err != nil {
		return nil, err
	}
	g, err := s.repo.GrantModuleAccess(ctx, tenantID, targetUserID, moduleID, &callerID)
	if err != nil {
		return nil, err
	}
	slog.Info("module access granted",
		"tenant_id", tenantID, "user_id", targetUserID, "module_id", moduleID, "granted_by", callerID)
	return g, nil
}

// RevokeModuleAccess removes a user's access to a module. Only admins may do this.
func (s *Service) RevokeModuleAccess(ctx context.Context, tenantID, callerID, targetUserID uuid.UUID, moduleID string) error {
	if moduleID == "" {
		return ErrInvalidModuleID
	}
	if err := s.requireAdmin(ctx, tenantID, callerID); err != nil {
		return err
	}
	if err := s.repo.RevokeModuleAccess(ctx, tenantID, targetUserID, moduleID); err != nil {
		return err
	}
	slog.Info("module access revoked",
		"tenant_id", tenantID, "user_id", targetUserID, "module_id", moduleID, "revoked_by", callerID)
	return nil
}

// BulkRevokeModuleAccess removes multiple grants at once. Only admins may do this.
func (s *Service) BulkRevokeModuleAccess(ctx context.Context, tenantID, callerID uuid.UUID, refs []ModuleGrantRef) (int, error) {
	if err := s.requireAdmin(ctx, tenantID, callerID); err != nil {
		return 0, err
	}
	for _, ref := range refs {
		if ref.ModuleID == "" {
			return 0, ErrInvalidModuleID
		}
	}
	n, err := s.repo.BulkRevokeModuleAccess(ctx, tenantID, refs)
	if err != nil {
		return 0, err
	}
	slog.Info("module access bulk-revoked", "tenant_id", tenantID, "count", n, "revoked_by", callerID)
	return n, nil
}

// requireAdmin returns ErrNotAdmin when the caller is not an admin in the tenant.
func (s *Service) requireAdmin(ctx context.Context, tenantID, callerID uuid.UUID) error {
	isAdmin, err := s.roleChecker.IsAdmin(ctx, tenantID, callerID)
	if err != nil {
		return err
	}
	if !isAdmin {
		return ErrNotAdmin
	}
	return nil
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

// ============================================================================
// Licensing
// ============================================================================

// GetTenantLicense returns every catalogue module with this tenant's activation
// state and the number of users holding a grant for it.
//
// The catalogue drives the list, not the table: a module the tenant has never
// touched has no row and comes back inactive, which is what an unbooked module
// is. Seats are reported as 0 for an inactive module — the grants stay in the
// database so reactivating restores them, but a module nobody can reach does
// not occupy seats in the cost view.
func (s *Service) GetTenantLicense(ctx context.Context, tenantID uuid.UUID) ([]*TenantModule, error) {
	activations, err := s.repo.ListModuleActivations(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	seats, err := s.repo.CountGrantsByModule(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	out := make([]*TenantModule, 0, len(modules.Catalog))
	for _, m := range modules.Catalog {
		out = append(out, buildTenantModule(m, activations[m.ID], seats[m.ID]))
	}
	return out, nil
}

// SetTenantModuleActive activates or deactivates a module tenant-wide.
//
// It does not consult the modules.* feature flags: those describe what the
// deployment runs, which the gateway knows and this service does not. The
// gateway rejects an activation the flag does not cover before calling here.
func (s *Service) SetTenantModuleActive(ctx context.Context, tenantID uuid.UUID, moduleID string, active bool, updatedBy *uuid.UUID) (*TenantModule, error) {
	if moduleID == "" {
		return nil, ErrInvalidModuleID
	}
	m, ok := modules.Get(moduleID)
	if !ok {
		return nil, ErrUnknownModule
	}

	if err := s.repo.SetModuleActivation(ctx, tenantID, moduleID, active, updatedBy); err != nil {
		return nil, err
	}

	seats, err := s.repo.CountGrantsByModule(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	slog.Info("tenant module activation changed",
		"tenant_id", tenantID,
		"module_id", moduleID,
		"active", active,
	)
	return buildTenantModule(m, active, seats[moduleID]), nil
}

// GetTenantSubscription returns the booked plan of the tenant.
func (s *Service) GetTenantSubscription(ctx context.Context, tenantID uuid.UUID) (*TenantSubscription, error) {
	return s.repo.GetTenantSubscription(ctx, tenantID)
}

func buildTenantModule(m modules.Module, active bool, grantedSeats int32) *TenantModule {
	seats := grantedSeats
	if !active {
		seats = 0
	}
	return &TenantModule{
		ModuleID:      m.ID,
		Group:         string(m.Group),
		Active:        active,
		AssignedSeats: seats,
		FlagKey:       m.FlagKey,
	}
}

// ============================================================================
// Value-sets
//
// Reads merge the shipped registry (valueset.go) with the tenant's override
// rows. Authorisation is the gateway's existing customization permission guard;
// this layer adds no policy of its own beyond validation, so that value-set
// editing stays reachable by exactly the people who can already rename labels.
// ============================================================================

// ListValueSets returns every known value-set, resolved for this tenant.
//
// The key set is the union of the shipped registry and the tenant's overrides.
// The FE mock lists only its code defaults (listResolvedValueSets iterates
// DEFAULT_VALUE_SETS) — a deliberate difference: a tenant-created list has no
// baseline and would otherwise be invisible in the very editor that made it.
//
// base=true returns the shipped baselines only (the editor's "reset to" view);
// tenant-only lists have no baseline and are omitted from that view.
func (s *Service) ListValueSets(ctx context.Context, tenantID uuid.UUID, base bool) ([]*ResolvedValueSet, error) {
	if base {
		keys := SystemValueSetKeys()
		out := make([]*ResolvedValueSet, 0, len(keys))
		for _, key := range keys {
			if resolved := BaseValueSet(key); resolved != nil {
				out = append(out, resolved)
			}
		}
		return out, nil
	}

	overrides, err := s.repo.ListValueSetOverrides(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	byKey := make(map[string]*ValueSet, len(overrides))
	for _, set := range overrides {
		byKey[set.Key] = set
	}

	keys := SystemValueSetKeys()
	for _, set := range overrides {
		if !IsSystemValueSet(set.Key) {
			keys = append(keys, set.Key)
		}
	}
	sort.Strings(keys)

	out := make([]*ResolvedValueSet, 0, len(keys))
	for _, key := range keys {
		if resolved := ResolveValueSet(key, byKey[key]); resolved != nil {
			out = append(out, resolved)
		}
	}
	return out, nil
}

// GetValueSet resolves one value-set for this tenant. ErrNotFound when neither
// the registry nor the tenant knows the key.
func (s *Service) GetValueSet(ctx context.Context, tenantID uuid.UUID, key string, base bool) (*ResolvedValueSet, error) {
	if !valueSetKeyPattern.MatchString(key) {
		return nil, ErrInvalidValueSetKey
	}
	if base {
		resolved := BaseValueSet(key)
		if resolved == nil {
			return nil, ErrNotFound
		}
		return resolved, nil
	}

	override, err := s.repo.GetValueSetOverride(ctx, tenantID, key)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	resolved := ResolveValueSet(key, override)
	if resolved == nil {
		return nil, ErrNotFound
	}
	return resolved, nil
}

// UpsertValueSet stores a tenant's override and returns the resolved result.
//
// For a system key this overrides the shipped list; for an unknown key it
// creates a tenant-owned list. Both are the same write — what differs is what
// deleting it later means.
func (s *Service) UpsertValueSet(ctx context.Context, tenantID uuid.UUID, callerID *uuid.UUID, set *ValueSet) (*ResolvedValueSet, error) {
	if err := validateValueSet(set); err != nil {
		return nil, err
	}
	set.Name = strings.TrimSpace(set.Name)
	for i := range set.Options {
		set.Options[i].Label = strings.TrimSpace(set.Options[i].Label)
	}

	stored, err := s.repo.UpsertValueSetOverride(ctx, tenantID, callerID, set)
	if err != nil {
		return nil, err
	}
	return ResolveValueSet(set.Key, stored), nil
}

// DeleteValueSet drops the tenant's override.
//
// The two meanings of that one DELETE:
//   - system key  -> reset. The list survives on its shipped baseline, which is
//     why a system list cannot be deleted at all. Returns the baseline.
//   - tenant key  -> the list is gone. Returns nil.
//
// Resetting a system list that was never overridden is a no-op, not an error:
// the caller asked for a state that already holds.
func (s *Service) DeleteValueSet(ctx context.Context, tenantID uuid.UUID, key string) (*ResolvedValueSet, error) {
	if !valueSetKeyPattern.MatchString(key) {
		return nil, ErrInvalidValueSetKey
	}

	err := s.repo.DeleteValueSetOverride(ctx, tenantID, key)
	switch {
	case err == nil:
	case errors.Is(err, ErrNotFound) && IsSystemValueSet(key):
		// Nothing stored, baseline already in effect.
	default:
		return nil, err
	}

	if IsSystemValueSet(key) {
		slog.InfoContext(ctx, "value-set reset to shipped baseline",
			"tenant_id", tenantID, "value_set", key)
		return BaseValueSet(key), nil
	}
	slog.InfoContext(ctx, "tenant value-set deleted",
		"tenant_id", tenantID, "value_set", key)
	return nil, nil
}
