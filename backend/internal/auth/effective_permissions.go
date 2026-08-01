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

// EffectivePermissions is the resolved answer for one user.
type EffectivePermissions struct {
	Roles        []EffectiveRole
	Capabilities []EffectiveCapability
}

// scopeRank orders the three data scopes from narrowest to widest.
var scopeRank = map[string]int{"own": 0, "team": 1, "all": 2}

// narrowestScope is what an unrecognised scope collapses to. The CHECK
// constraint on role_permissions.scope makes that unreachable today; should it
// ever become reachable, granting more than intended is the failure we cannot
// afford, so the fallback errs downwards.
const narrowestScope = "own"

func normalizeScope(scope string) string {
	if _, ok := scopeRank[scope]; !ok {
		return narrowestScope
	}
	return scope
}

// GetEffectivePermissions resolves everything a user may do into the shape the
// RBAC frontend consumes: the roles they hold and, per capability key, the
// widest scope any of those roles grants.
//
// Union rule: the same key held through several roles resolves to the widest
// scope (own < team < all) — a role can only ever add reach, never take it
// away. A key that appears nowhere is absent from the result; absence means
// denied, so an empty scope must never be emitted.
func (s *Service) GetEffectivePermissions(ctx context.Context, userID uuid.UUID) (*EffectivePermissions, error) {
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
