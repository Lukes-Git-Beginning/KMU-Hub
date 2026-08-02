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
// grant of the source role including its scope. Actor and tenant arrive as
// parameters rather than being read from the context because this package must
// not import middleware (import cycle) — the gRPC layer resolves both, exactly
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
//
// The clone is an escalation path and is guarded like one: without the check,
// an it_admin could clone the admin preset into a role carrying rights they do
// not hold themselves and then hand it to someone else — the grant-set check
// on SetRolePermissions would never see it, because the rights arrive through
// the copy rather than through an edit.
func (s *Service) CreateRole(ctx context.Context, actorID, tenantID uuid.UUID, in CreateRoleInput) (*Role, error) {
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

	source, err := s.repo.GetRolePermissions(ctx, in.BasedOn)
	if err != nil {
		return nil, err
	}
	if err := s.assertWithinReach(ctx, actorID, source); err != nil {
		return nil, err
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

// RoleGrant is one capability→scope pair of a role's grant set. Key is the
// permissions table's `name` column (the catalogue's "resource:action"
// composition), Scope one of own|team|all.
type RoleGrant struct {
	Key   string
	Scope string
}

// GetRolePermissions returns the grant set of a role visible to the caller —
// presets included, since a builder needs to read what it is cloning. Only
// the write side (SetRolePermissions) restricts presets.
func (s *Service) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]RoleGrant, error) {
	if _, err := s.repo.GetRoleByID(ctx, roleID); err != nil {
		return nil, err
	}
	return s.repo.GetRolePermissions(ctx, roleID)
}

// SetRolePermissions replaces the entire grant set of a tenant-owned role.
// The preset check happens here for the same reason as UpdateRole/DeleteRole:
// the write policy would just touch zero rows on a preset, which is not the
// explicit 403 the builder needs to tell "nothing changed" from "not allowed".
//
// The scope check is a second boundary validation, not a duplicate of the
// database's role_permissions_scope_check: a value longer than the scope
// column's VARCHAR(8) fails with a length error the constraint never sees,
// which would otherwise reach the caller as a raw 500 instead of the clean
// 422 an out-of-range grant deserves. The frontend's CapabilityScope union
// should make this unreachable through the UI; this is the boundary for
// anything else that speaks the API directly.
//
// Two guardrails sit in front of the write. The grant set may not reach past
// what the caller holds themselves (escalation, PHASE-1-RBAC-PLAN §4c), and
// that covers the WHOLE resulting set rather than only what changed: a PUT is
// a full replace, so afterwards the role carries every key of it because this
// caller put it there. And a caller who wears this role may not strip their
// own role administration out of it (§4b).
func (s *Service) SetRolePermissions(ctx context.Context, actorID, roleID uuid.UUID, grants []RoleGrant) ([]RoleGrant, error) {
	role, err := s.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role.TenantID == nil {
		return nil, ErrRolePresetImmutable
	}

	keys := make([]string, len(grants))
	for i, g := range grants {
		if g.Scope != "own" && g.Scope != "team" && g.Scope != "all" {
			return nil, ErrCapabilityKeyUnknown
		}
		keys[i] = g.Key
	}

	// Catalogue before reach, or a typo would come back as a 403: an
	// unknown key is inside nobody's grant set, so the escalation check
	// below cannot tell it apart from an attempt to grab a real right.
	unknown, err := s.repo.CountUnknownPermissionKeys(ctx, keys)
	if err != nil {
		return nil, err
	}
	if unknown > 0 {
		return nil, ErrCapabilityKeyUnknown
	}

	if err := s.assertWithinReach(ctx, actorID, grants); err != nil {
		return nil, err
	}
	if err := s.assertKeepsOwnRoleAdmin(ctx, actorID, roleID, grants); err != nil {
		return nil, err
	}

	return s.repo.SetRolePermissions(ctx, roleID, grants)
}

// AssignUserRole grants roleID to userID and returns the role ids the account
// holds afterwards. Presets are assignable — only editing them is forbidden;
// "admin" and "member" are presets, so a preset check here would lock out the
// most common assignment there is.
//
// Both ids are resolved before the write, and both resolutions are the tenant
// boundary rather than a friendliness: users and roles each carry an RLS read
// policy, user_roles carries none. A foreign account and a foreign role are
// therefore invisible, not forbidden, and both answer 404 — the same
// indistinguishability GetRoleByID documents.
//
// Assigning a role the account already holds is a no-op that still returns the
// current list, so the builder can treat the call as "make it so".
//
// Handing a role to someone ELSE is not capped by what the caller holds: that
// delegation is the whole point of hr_admin (admin:role:assign without
// admin:role:create/edit), which is supposed to staff roles richer than its
// own. Handing one to ONESELF is capped, because that is the direct route from
// "may assign roles" to "has every right in the tenant" — the escalation
// guardrail of PHASE-1-RBAC-PLAN §4c applied to the one case where the
// assignment enlarges the caller.
func (s *Service) AssignUserRole(ctx context.Context, actorID, userID, roleID uuid.UUID) ([]string, error) {
	if _, err := s.repo.GetUserByID(ctx, userID); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetRoleByID(ctx, roleID); err != nil {
		return nil, err
	}

	if actorID == userID {
		grants, err := s.repo.GetRolePermissions(ctx, roleID)
		if err != nil {
			return nil, err
		}
		if err := s.assertWithinReach(ctx, actorID, grants); err != nil {
			return nil, err
		}
	}

	if err := s.repo.AssignUserRole(ctx, userID, roleID); err != nil {
		return nil, err
	}
	return s.repo.GetUserRoleIDs(ctx, userID)
}

// RevokeUserRole takes roleID off userID and returns the remaining role ids.
//
// The two guardrails in front of it (PHASE-1-RBAC-PLAN §4a/b) only look at
// roles that carry the role-administration capability — see
// assertRevokeKeepsAdmins. Taking any other role away is unremarkable and
// stays a plain revoke.
func (s *Service) RevokeUserRole(ctx context.Context, actorID, userID, roleID uuid.UUID) ([]string, error) {
	if _, err := s.repo.GetUserByID(ctx, userID); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetRoleByID(ctx, roleID); err != nil {
		return nil, err
	}

	if err := s.assertRevokeKeepsAdmins(ctx, actorID, userID, roleID); err != nil {
		return nil, err
	}

	if err := s.repo.RevokeUserRole(ctx, userID, roleID); err != nil {
		return nil, err
	}
	return s.repo.GetUserRoleIDs(ctx, userID)
}
