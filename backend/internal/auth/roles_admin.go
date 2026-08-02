package auth

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

// CustomRoleLimit caps how many roles a tenant may own. It mirrors
// CUSTOM_ROLE_LIMIT in the frontend (mocks/data/rbac.ts) and counts only
// tenant-owned roles — the shared system presets never eat into it.
const CustomRoleLimit = 20

// Role is a role as the admin surface sees it: either a Zentria-maintained
// system preset (TenantID nil, shared by every tenant, immutable) or a
// tenant-owned custom role. BasedOn names the preset a custom role was cloned
// from and is nil on presets themselves.
type Role struct {
	ID              uuid.UUID
	Name            string
	Description     string
	TenantID        *uuid.UUID
	BasedOn         *uuid.UUID
	Color           string
	IsSystem        bool
	MemberCount     int
	CapabilityCount int
}

// ListRoles returns the system presets together with the custom roles of the
// calling tenant — exactly the rows the roles table's RLS read policy admits,
// so no tenant filter is needed here.
func (s *Service) ListRoles(ctx context.Context) ([]Role, error) {
	return s.repo.ListRoles(ctx)
}

// CreateRoleInput describes a new tenant-owned role. BasedOn is mandatory:
// creating a role is always a clone, never a blank slate.
type CreateRoleInput struct {
	Name        string
	Description string
	Color       string
	BasedOn     uuid.UUID
}

// CreateRole clones BasedOn into a new role owned by tenantID, copying every
// grant of the source role including its scope. The tenant arrives as a
// parameter rather than being read from the context because this package must
// not import middleware (import cycle) — the gRPC layer resolves it, exactly
// like CreateInvitation.
//
// The three business rules live here rather than in the repository: the custom
// role budget, the name collision, and the requirement that the clone source
// is visible to the caller. The name check is deliberately case-insensitive
// and also covers the presets, matching the frontend contract — the unique
// index cannot do either (it is case-sensitive, and presets sit in their own
// COALESCE bucket, so "Admin" beside the "admin" preset would pass it).
// The repository still maps a unique violation back onto ErrRoleNameExists as
// a backstop for the race between this check and the insert.
func (s *Service) CreateRole(ctx context.Context, tenantID uuid.UUID, in CreateRoleInput) (*Role, error) {
	name := strings.TrimSpace(in.Name)

	count, err := s.repo.CountCustomRoles(ctx)
	if err != nil {
		return nil, err
	}
	if count >= CustomRoleLimit {
		return nil, ErrRoleLimitReached
	}

	exists, err := s.repo.RoleNameExists(ctx, name, uuid.Nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrRoleNameExists
	}

	return s.repo.CreateRole(ctx, tenantID, CreateRoleInput{
		Name:        name,
		Description: strings.TrimSpace(in.Description),
		Color:       in.Color,
		BasedOn:     in.BasedOn,
	})
}

// UpdateRoleInput carries the optional fields of a role rename — nil means
// "leave unchanged", matching the frontend's partial PATCH body.
type UpdateRoleInput struct {
	Name        *string
	Description *string
	Color       *string
}

// UpdateRole renames/re-describes/recolors a tenant-owned role. The preset
// check happens here rather than relying on the write policy's zero-row
// UPDATE, because a silent no-op would look like a successful save in the
// builder — the caller needs the explicit 403.
func (s *Service) UpdateRole(ctx context.Context, roleID uuid.UUID, in UpdateRoleInput) (*Role, error) {
	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role.TenantID == nil {
		return nil, ErrRolePresetImmutable
	}

	if in.Name != nil {
		trimmed := strings.TrimSpace(*in.Name)
		in.Name = &trimmed
		exists, err := s.repo.RoleNameExists(ctx, trimmed, roleID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrRoleNameExists
		}
	}
	if in.Description != nil {
		trimmed := strings.TrimSpace(*in.Description)
		in.Description = &trimmed
	}

	return s.repo.UpdateRole(ctx, roleID, in)
}

// DeleteRole removes a tenant-owned custom role. Presets are immutable, and a
// role still worn by a member cannot be deleted out from under its holders —
// both checks run before the DELETE, not after a constraint violation.
func (s *Service) DeleteRole(ctx context.Context, roleID uuid.UUID) error {
	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if role.TenantID == nil {
		return ErrRolePresetImmutable
	}

	hasMembers, err := s.repo.RoleHasMembers(ctx, roleID)
	if err != nil {
		return err
	}
	if hasMembers {
		return ErrRoleHasMembers
	}

	return s.repo.DeleteRole(ctx, roleID)
}
