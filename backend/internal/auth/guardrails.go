package auth

import (
	"context"
	"slices"

	"github.com/google/uuid"
)

// roleAdminKeys are the permissions role administration itself runs on. They
// are deliberately the same pair the assignment routes guard with
// (RequirePermissionAny{roles:manage, admin:role:assign}): the coarse legacy
// key every admin token carries since migration 000002, and the fine successor
// seeded for admin/it_admin/hr_admin in 000256.
//
// "Administrator" is defined here as "may hand out roles", not as "wears the
// admin preset". A tenant that clones the preset into "Geschäftsführung" and
// retires the original still has administrators — a name check would not see
// them, and the last-admin guardrail would fire on a tenant that is perfectly
// healthy. Conversely, whoever ends up holding this pair can repair every other
// right in the tenant, which is exactly what makes losing the last holder
// unrecoverable.
var roleAdminKeys = []string{"roles:manage", "admin:role:assign"}

// grantReach is a caller's resolved grant set: capability key -> widest scope.
// Absence means the caller does not hold the key at all.
type grantReach map[string]string

// covers reports whether the reach permits granting (key, scope) — the key has
// to be present and its own scope at least as wide. This is the escalation
// rule in one line: nobody hands out a right they do not hold, and nobody
// widens a scope past their own (own -> all).
func (r grantReach) covers(key, scope string) bool {
	held, ok := r[key]
	if !ok {
		return false
	}
	return scopeRank[normalizeScope(held)] >= scopeRank[normalizeScope(scope)]
}

// callerReach resolves everything the caller may currently hand out, unioned
// across their roles the same way GetEffectivePermissions unions the screen
// view: the widest scope any role grants wins, because a second role can only
// ever add reach.
//
// It reads GetUserGrants rather than GetEffectivePermissions because a grant
// set may name any key of the catalogue, coarse ones included — see the
// repository comment on why those two queries stay separate.
func (s *Service) callerReach(ctx context.Context, actorID uuid.UUID) (grantReach, error) {
	rows, err := s.repo.GetUserGrants(ctx, actorID)
	if err != nil {
		return nil, err
	}

	reach := make(grantReach, len(rows))
	for _, row := range rows {
		if row.Key == "" {
			continue
		}
		scope := normalizeScope(row.Scope)
		if held, ok := reach[row.Key]; !ok || scopeRank[scope] > scopeRank[held] {
			reach[row.Key] = scope
		}
	}
	return reach, nil
}

// assertWithinReach rejects a grant set that goes beyond what the caller holds.
// It is the (c) guardrail of PHASE-1-RBAC-PLAN §4 and applies to the two paths
// that DEFINE what a role can do — cloning one (CreateRole) and replacing its
// grants (SetRolePermissions).
//
// Handing an existing role to someone else is deliberately not covered: a role
// is a bundle the tenant's administrators defined, and delegating "may put
// people into these bundles" without also granting their contents is the whole
// point of the hr_admin preset (role:assign without role:create/edit). What is
// covered is taking a bundle for oneself — see AssignUserRole.
func (s *Service) assertWithinReach(ctx context.Context, actorID uuid.UUID, grants []RoleGrant) error {
	if len(grants) == 0 {
		return nil
	}

	reach, err := s.callerReach(ctx, actorID)
	if err != nil {
		return err
	}
	for _, g := range grants {
		if !reach.covers(g.Key, g.Scope) {
			return ErrPrivilegeEscalation
		}
	}
	return nil
}

// roleGrantsRoleAdmin reports whether a role carries the role-administration
// capability, i.e. whether losing it can cost someone that right.
func (s *Service) roleGrantsRoleAdmin(ctx context.Context, roleID uuid.UUID) (bool, error) {
	grants, err := s.repo.GetRolePermissions(ctx, roleID)
	if err != nil {
		return false, err
	}
	for _, g := range grants {
		if slices.Contains(roleAdminKeys, g.Key) {
			return true, nil
		}
	}
	return false, nil
}

// holdsRoleAdminWithout reports whether the user would still be able to
// administer roles if excludeRoleID were taken off them — "do they have a
// second way in".
func (s *Service) holdsRoleAdminWithout(ctx context.Context, userID, excludeRoleID uuid.UUID) (bool, error) {
	rows, err := s.repo.GetUserGrants(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, row := range rows {
		if row.RoleID != excludeRoleID && slices.Contains(roleAdminKeys, row.Key) {
			return true, nil
		}
	}
	return false, nil
}

// assertRevokeKeepsAdmins runs the (a) and (b) guardrails in front of a revoke:
// the tenant keeps at least one administrator, and the caller does not remove
// the very capability they are working with.
//
// Both only apply when the role being taken away actually carries the
// capability. Running the count unconditionally would be worse than useless: a
// tenant whose administrators are all deactivated would count zero and every
// unrelated revoke would fail with last_admin.
//
// Order matters. last_admin is checked first because it is the stronger
// statement (nobody is left at all) and the only one of the two the frontend
// already renders as a precise message.
func (s *Service) assertRevokeKeepsAdmins(ctx context.Context, actorID, userID, roleID uuid.UUID) error {
	isAdminRole, err := s.roleGrantsRoleAdmin(ctx, roleID)
	if err != nil {
		return err
	}
	if !isAdminRole {
		return nil
	}

	remaining, err := s.repo.CountRoleAdminsExcluding(ctx, roleAdminKeys, userID, roleID)
	if err != nil {
		return err
	}
	if remaining == 0 {
		return ErrLastAdmin
	}

	if actorID != userID {
		return nil
	}
	stillAdmin, err := s.holdsRoleAdminWithout(ctx, userID, roleID)
	if err != nil {
		return err
	}
	if !stillAdmin {
		return ErrSelfLockout
	}
	return nil
}

// assertKeepsOwnRoleAdmin is the (b) guardrail on the grant-set path: rewriting
// a role the caller wears must not strip the role-administration capability off
// themselves. Revoking is not the only way to lock yourself out — emptying the
// grant set of the role you administer through does the same thing, and it is
// the easier mistake to make in a builder that shows checkboxes.
func (s *Service) assertKeepsOwnRoleAdmin(ctx context.Context, actorID, roleID uuid.UUID, grants []RoleGrant) error {
	for _, g := range grants {
		if slices.Contains(roleAdminKeys, g.Key) {
			return nil // the capability survives the rewrite
		}
	}

	rows, err := s.repo.GetUserGrants(ctx, actorID)
	if err != nil {
		return err
	}
	wearsRole := false
	for _, row := range rows {
		if row.RoleID == roleID {
			wearsRole = true
			break
		}
	}
	if !wearsRole {
		return nil
	}

	for _, row := range rows {
		if row.RoleID != roleID && slices.Contains(roleAdminKeys, row.Key) {
			return nil // another role of theirs still carries it
		}
	}
	return ErrSelfLockout
}
