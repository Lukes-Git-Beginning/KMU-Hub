package auth

import (
	"context"
	"slices"

	"github.com/google/uuid"
)

// TokenPermissions is everything an access token says about what an account may
// do. The three parts are minted together on purpose — see ResolveTokenPermissions.
type TokenPermissions struct {
	// Permissions is the allow set the guards read: the coarse legacy keys
	// ("files:write") and the fine catalogue keys ("documents:file:upload")
	// side by side, with allow overrides added and deny overrides removed.
	Permissions []string

	// Denied lists the keys a deny override took away. They are already absent
	// from Permissions; this list exists because absence alone does not close a
	// door — see RequirePermissionAny in the middleware, which has to know that
	// a coarse key must stop standing in for a fine one that was denied.
	Denied []string

	// Scopes carries only the keys that reach less far than the whole tenant.
	// Absence reads as "all" (middleware.PermissionScope), which is why a denied
	// key is written in at the narrowest scope rather than dropped.
	Scopes map[string]string
}

// ResolveTokenPermissions resolves the permission side of an access token for
// one account: the role union with the account's per-user overrides folded in
// (RBAC R-6).
//
// The three claims are built here in one place, from one read, because they can
// only be correct together. The scopes claim cannot express "denied" — a key
// missing from it reads as "all". Making it override-aware on its own would let
// a deny hand out the widest reach instead of taking it away. Making the
// permissions claim override-aware on its own would leave the scope of a key an
// allow override just granted unbounded. Both, or neither.
//
// tenantID is a parameter rather than something read off the context because
// every caller of this is inside createTokenPair, which runs under sysctx.With():
// RLS does not filter there, so the override read has to name its tenant itself.
// The same trap this repository hit on two_factor_policy and presence_config.
//
// An account without overrides gets the pure role union back, byte for byte
// identical to what was minted before overrides existed: the override read is
// one indexed lookup that returns nothing, and nothing after it runs.
func (s *Service) ResolveTokenPermissions(ctx context.Context, tenantID, userID uuid.UUID) (*TokenPermissions, error) {
	permissions, err := s.repo.GetUserPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}

	// The union is what carries the scopes; GetUserPermissions above is a flat
	// name list with no scope column.
	union, err := s.GetRoleUnion(ctx, userID)
	if err != nil {
		return nil, err
	}

	overrides, err := s.repo.GetUserOverridesForTenant(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	if len(overrides) == 0 {
		return &TokenPermissions{
			Permissions: permissions,
			Scopes:      narrowScopes(union.Capabilities),
		}, nil
	}
	return applyOverridesToToken(permissions, union.Capabilities, overrides), nil
}

// applyOverridesToToken folds the override map into the three claim parts.
//
// The allow set keeps BOTH permission worlds. Rebuilding it from the fine
// catalogue keys alone — which is what the union carries — would drop every
// coarse legacy key, and those are the currency of most RequirePermission
// gates: every one of them would answer 403 the moment an account got its first
// override. So the coarse keys are carried through untouched and the overrides
// only ever act on the keys they actually name.
func applyOverridesToToken(permissions []string, caps []EffectiveCapability, overrides []CapabilityOverride) *TokenPermissions {
	scopes := narrowScopes(caps)
	denied := make(map[string]bool, len(overrides))
	granted := make([]string, 0, len(overrides))

	for _, o := range overrides {
		if o.Mode != OverrideModeAllow {
			// Same downward fallback as applyOverrides: anything that is not a
			// literal allow takes the key away.
			denied[o.Key] = true
			// A denied key is written into the scopes map at its narrowest
			// rather than removed from it. Removing it would read as "all"
			// (middleware.PermissionScope), so a handler that asks for the
			// scope without a matching fine-grained guard in front of it would
			// see the widest reach of the very key that was taken away.
			if scopes == nil {
				scopes = make(map[string]string)
			}
			scopes[o.Key] = narrowestScope
			continue
		}

		granted = append(granted, o.Key)
		// Set, not raise — the override editor can also narrow one person
		// without cloning a role (applyOverrides documents the same rule).
		scope := normalizeScope(o.Scope)
		if scope == ScopeAll {
			delete(scopes, o.Key)
			continue
		}
		if scopes == nil {
			scopes = make(map[string]string)
		}
		scopes[o.Key] = scope
	}

	out := make([]string, 0, len(permissions)+len(granted))
	held := make(map[string]bool, len(permissions))
	for _, p := range permissions {
		if denied[p] {
			continue
		}
		held[p] = true
		out = append(out, p)
	}
	// Allow overrides go last, in the order the repository read them (by key),
	// so the claim stays stable across two logins of the same account.
	for _, key := range granted {
		if !held[key] {
			held[key] = true
			out = append(out, key)
		}
	}

	deniedKeys := make([]string, 0, len(denied))
	for key := range denied {
		deniedKeys = append(deniedKeys, key)
	}
	slices.Sort(deniedKeys)

	if len(scopes) == 0 {
		scopes = nil
	}
	return &TokenPermissions{Permissions: out, Denied: deniedKeys, Scopes: scopes}
}

// narrowScopes reduces resolved capabilities to the keys that reach less far
// than the whole tenant, ready to travel in the access token.
//
// Everything absent from the map reaches "all" — that is what lets the map stay
// small. A member holds ~159 keys of which 15 are narrowed today; carrying all
// of them would inflate every request header for no gain, and an admin (454
// keys, every one of them "all") ends up with no scope claim at all.
//
// The same asymmetry makes the map safe to read with an "all" default: a key
// missing from a token minted here genuinely is unrestricted, and a legacy
// token minted before this claim existed keeps behaving exactly as it did.
// Narrowing on absence would instead shrink every list for every user still
// holding a valid old token.
//
// nil rather than an empty map when nothing is narrowed: the claim is
// `omitempty`, so an admin's token drops it entirely instead of shipping `{}`.
func narrowScopes(caps []EffectiveCapability) map[string]string {
	var scopes map[string]string
	for _, c := range caps {
		if c.Scope == ScopeAll {
			continue
		}
		if scopes == nil {
			scopes = make(map[string]string)
		}
		scopes[c.Key] = c.Scope
	}
	return scopes
}
