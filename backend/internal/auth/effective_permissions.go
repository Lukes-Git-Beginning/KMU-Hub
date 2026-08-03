package auth

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// EffectiveGrantRow is one row of the resolver query: a role the user holds,
// together with one fine-grained capability that role grants. A role without a
// single fine-grained grant still appears once, with Key and Scope empty.
type EffectiveGrantRow struct {
	RoleID       uuid.UUID
	RoleName     string
	RoleColor    string
	RoleIsSystem bool

	Key   string
	Scope string
}

// EffectiveRole is a role the user holds. IsSystem marks a preset — a role
// with tenant_id IS NULL, shared by every tenant and not editable by them.
type EffectiveRole struct {
	ID       uuid.UUID
	Name     string
	IsSystem bool
	Color    string
}

// EffectiveCapability is one capability key after the union across all roles.
// Sources lists every contributing role, not only the one that won the scope —
// the admin UI shows where a right comes from, and "you have it through
// manager" is the wrong answer when hr_admin grants it too.
//
// Sources carries role IDs, not names: the frontend resolves each entry against
// the Roles slice of this same response (`roles.find(r => r.id === src)` in
// EffectivePermissionsView) to render the provenance chip with the role's
// colour and its translated label. Names would silently fall back to the raw
// string — no colour, no i18n.
type EffectiveCapability struct {
	Key     string
	Scope   string
	Sources []string
}

// DeniedCapability is a key the user's roles grant but a deny override takes
// away. It is absent from Capabilities; this list carries what the roles would
// have given, so the effective view can render it struck through instead of
// simply missing — "taken from you" and "never had it" are different answers.
type DeniedCapability struct {
	Key       string
	RoleScope string
	Sources   []string
}

// EffectivePermissions is the resolved answer for one user.
//
// HasOverrides and DeniedByOverride stay zero for the pure role union (the
// ?base=1 baseline) and for every account without a single override, which is
// what keeps their answer identical to the pre-override one.
type EffectivePermissions struct {
	Roles            []EffectiveRole
	Capabilities     []EffectiveCapability
	HasOverrides     bool
	DeniedByOverride []DeniedCapability
}

// OverrideSource marks a capability an allow override put there or raised. It
// travels inside Sources next to the contributing role ids, because the
// provenance chip in EffectivePermissionsView asks one list "where does this
// come from" — a separate flag would have to be threaded through every layer
// for the same answer. The frontend knows the sentinel as OVERRIDE_SOURCE
// (rbac-types.ts) and filters it out of the role lookup.
const OverrideSource = "override"

// The three data scopes a grant can carry. They answer "how many rows", never
// "which action" — that is the permission key's job. A guard decides whether a
// request may run at all; the scope decides how much of the tenant's data the
// answer may contain.
const (
	ScopeOwn  = "own"
	ScopeTeam = "team"
	ScopeAll  = "all"
)

// scopeRank orders the three data scopes from narrowest to widest.
var scopeRank = map[string]int{ScopeOwn: 0, ScopeTeam: 1, ScopeAll: 2}

// narrowestScope is what an unrecognised scope collapses to. The CHECK
// constraint on role_permissions.scope makes that unreachable today; should it
// ever become reachable, granting more than intended is the failure we cannot
// afford, so the fallback errs downwards.
const narrowestScope = ScopeOwn

func normalizeScope(scope string) string {
	if _, ok := scopeRank[scope]; !ok {
		return narrowestScope
	}
	return scope
}

// GetEffectivePermissions resolves everything a user may do: the role union
// with the account's per-user overrides folded in (RBAC R-6).
//
// This is the single seam the frontend's applyUserOverrides() mirrors — role
// union first, overrides on top, never two places that each apply a bit.
//
// An account without overrides gets the union back unchanged, down to the
// allocation: the override lookup is one indexed read that returns nothing, and
// nothing after it runs.
func (s *Service) GetEffectivePermissions(ctx context.Context, userID uuid.UUID) (*EffectivePermissions, error) {
	union, err := s.GetRoleUnion(ctx, userID)
	if err != nil {
		return nil, err
	}

	overrides, err := s.repo.GetUserOverrides(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(overrides) == 0 {
		return union, nil
	}
	return applyOverrides(union, overrides), nil
}

// applyOverrides folds one account's overrides into its resolved role union.
// The override wins per key over every role: deny drops the key regardless of
// how many roles grant it, allow sets the key to the override's scope — set,
// not raise, so an admin can also narrow one person without cloning a role.
//
// Mirrors applyUserOverrides in mocks/data/rbac.ts, which is the contract the
// override editor was built against.
func applyOverrides(union *EffectivePermissions, overrides []CapabilityOverride) *EffectivePermissions {
	byKey := make(map[string]EffectiveCapability, len(union.Capabilities))
	for _, c := range union.Capabilities {
		byKey[c.Key] = c
	}

	denied := make([]DeniedCapability, 0, len(overrides))
	for _, o := range overrides {
		inherited, hasRoleGrant := byKey[o.Key]

		// Anything but a literal allow takes the key away. The CHECK
		// constraint on mode makes a third value unreachable today; if it ever
		// becomes reachable, handing out a right nobody meant to grant is the
		// failure we cannot afford, so the fallback errs downwards — the same
		// direction narrowestScope errs in.
		if o.Mode != OverrideModeAllow {
			if hasRoleGrant {
				denied = append(denied, DeniedCapability{
					Key:       o.Key,
					RoleScope: inherited.Scope,
					Sources:   inherited.Sources,
				})
			}
			delete(byKey, o.Key)
			continue
		}

		// The sentinel goes last, after the roles that grant the key too: the
		// union already sorted those, and "inherited from A and B, then raised
		// by hand" is the order the provenance chips read in.
		sources := make([]string, 0, len(inherited.Sources)+1)
		if hasRoleGrant {
			sources = append(sources, inherited.Sources...)
		}
		sources = append(sources, OverrideSource)

		byKey[o.Key] = EffectiveCapability{
			Key:     o.Key,
			Scope:   normalizeScope(o.Scope),
			Sources: sources,
		}
	}

	// Same byte-stability rule as the union: sorted keys, and the denied list
	// sorted too — it is rendered as a list, not looked up by key.
	result := &EffectivePermissions{
		Roles:            union.Roles,
		Capabilities:     make([]EffectiveCapability, 0, len(byKey)),
		HasOverrides:     true,
		DeniedByOverride: denied,
	}
	for _, key := range slices.Sorted(maps.Keys(byKey)) {
		result.Capabilities = append(result.Capabilities, byKey[key])
	}
	slices.SortFunc(result.DeniedByOverride, func(a, b DeniedCapability) int {
		return strings.Compare(a.Key, b.Key)
	})
	return result
}

// GetRoleUnion resolves what a user's roles alone grant: the roles they hold
// and, per capability key, the widest scope any of those roles gives.
//
// Union rule: the same key held through several roles resolves to the widest
// scope (own < team < all) — a role can only ever add reach, never take it
// away. A key that appears nowhere is absent from the result; absence means
// denied, so an empty scope must never be emitted.
//
// This is the ?base=1 answer of the override editor (the greyed "inherited"
// column) and the input GetEffectivePermissions applies overrides to.
func (s *Service) GetRoleUnion(ctx context.Context, userID uuid.UUID) (*EffectivePermissions, error) {
	grants, err := s.repo.GetEffectivePermissions(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := &EffectivePermissions{
		Roles:        []EffectiveRole{},
		Capabilities: []EffectiveCapability{},
	}
	seenRole := make(map[uuid.UUID]struct{})
	byKey := make(map[string]*EffectiveCapability)

	for _, g := range grants {
		if _, ok := seenRole[g.RoleID]; !ok {
			seenRole[g.RoleID] = struct{}{}
			result.Roles = append(result.Roles, EffectiveRole{
				ID:       g.RoleID,
				Name:     g.RoleName,
				IsSystem: g.RoleIsSystem,
				Color:    g.RoleColor,
			})
		}
		if g.Key == "" {
			continue
		}

		scope := normalizeScope(g.Scope)
		roleID := g.RoleID.String()
		existing, ok := byKey[g.Key]
		if !ok {
			byKey[g.Key] = &EffectiveCapability{
				Key:     g.Key,
				Scope:   scope,
				Sources: []string{roleID},
			}
			continue
		}
		if scopeRank[scope] > scopeRank[existing.Scope] {
			existing.Scope = scope
		}
		if !slices.Contains(existing.Sources, roleID) {
			existing.Sources = append(existing.Sources, roleID)
		}
	}

	// Sorted output keeps the response byte-stable across calls, which matters
	// for HTTP caching and for diffing two users' rights against each other.
	for _, key := range slices.Sorted(maps.Keys(byKey)) {
		c := byKey[key]
		slices.Sort(c.Sources)
		result.Capabilities = append(result.Capabilities, *c)
	}
	slices.SortFunc(result.Roles, func(a, b EffectiveRole) int {
		return strings.Compare(a.Name, b.Name)
	})
	return result, nil
}

// The scopes claim of the access token is built from a role union in
// narrowScopes (token_permissions.go), together with the permissions claim —
// the two can only be override-aware at the same time.
