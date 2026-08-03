package auth

import (
	"context"
	"slices"

	"github.com/google/uuid"
)

// CapabilityOverride is one per-user deviation from the role union: `allow`
// sets or raises a capability key, `deny` takes it away no matter how many
// roles grant it. Scope is carried for both modes but only means anything for
// allow — the frontend type does the same (rbac-types.ts).
type CapabilityOverride struct {
	Key   string
	Mode  string
	Scope string
}

const (
	// OverrideModeAllow grants or widens a key the roles do not (fully) give.
	OverrideModeAllow = "allow"
	// OverrideModeDeny removes a key the roles grant.
	OverrideModeDeny = "deny"
)

// OverrideChange is one entry of the diff a write produced — the audit trail
// of "who changed which key from what to what". Before nil means the key
// followed the roles until now, After nil means it follows them again.
type OverrideChange struct {
	Key    string
	Before *CapabilityOverride
	After  *CapabilityOverride
}

// GetUserOverrides returns the override map of an account. The account is
// resolved first: users carries an RLS read policy, so an account of another
// tenant is invisible rather than forbidden and answers 404 — the same
// indistinguishability GetRoleByID documents.
func (s *Service) GetUserOverrides(ctx context.Context, userID uuid.UUID) ([]CapabilityOverride, error) {
	if _, err := s.repo.GetUserByID(ctx, userID); err != nil {
		return nil, err
	}
	return s.repo.GetUserOverrides(ctx, userID)
}

// SetUserOverrides replaces the entire override map of an account and returns
// the stored map together with the diff against the previous one.
//
// It is a full replace, like SetRolePermissions: a key missing from the input
// goes back to following the roles live. An empty input clears everything,
// which is what the editor's "auf Rollen-Stand zurücksetzen" does.
//
// Four guardrails run before the write, in this order:
//
//	(1) the caller may not edit their own account at all,
//	(2) mode/scope have to be spellable and every key has to be in the
//	    catalogue,
//	(3) no allow may reach past what the caller holds themselves,
//	(4) the tenant has to keep at least one role administrator.
//
// (2) runs before (3) for the same reason SetRolePermissions orders them that
// way: an unknown key is inside nobody's grant set, so the escalation check
// cannot tell a typo apart from an attempt to grab a real right and would
// answer 403 to a 422 problem.
func (s *Service) SetUserOverrides(
	ctx context.Context,
	actorID, tenantID, userID uuid.UUID,
	overrides []CapabilityOverride,
) ([]CapabilityOverride, []OverrideChange, error) {
	if actorID == userID {
		return nil, nil, ErrSelfLockout
	}
	if _, err := s.repo.GetUserByID(ctx, userID); err != nil {
		return nil, nil, err
	}

	normalized, err := s.validateOverrides(ctx, overrides)
	if err != nil {
		return nil, nil, err
	}
	if err := s.assertOverridesWithinReach(ctx, actorID, normalized); err != nil {
		return nil, nil, err
	}

	if err := s.assertOverridesKeepAdmins(ctx, userID, normalized); err != nil {
		return nil, nil, err
	}

	before, err := s.repo.GetUserOverrides(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	stored, err := s.repo.SetUserOverrides(ctx, tenantID, actorID, userID, normalized)
	if err != nil {
		return nil, nil, err
	}
	return stored, diffOverrides(before, stored), nil
}

// ClearUserOverrides drops every override of an account, putting it back on
// the pure role stand. It returns the diff so the caller can audit each key
// that disappeared.
//
// The self-edit block applies here too: "the caller may not edit their own
// account" is not about the direction of the change. A reset of one's own
// account can drop an allow the account depends on just as well.
//
// No last-admin check: a clear only ever REMOVES overrides. A deny that made
// somebody a non-administrator disappears, an allow that made them one
// disappears too — but that allow could only have been set by a caller who
// held the right themselves, so the tenant had a second administrator at that
// moment. What the clear cannot do is take the last one away without also
// having a role administrator on the other end of the call.
func (s *Service) ClearUserOverrides(ctx context.Context, actorID, userID uuid.UUID) ([]OverrideChange, error) {
	if actorID == userID {
		return nil, ErrSelfLockout
	}
	if _, err := s.repo.GetUserByID(ctx, userID); err != nil {
		return nil, err
	}

	before, err := s.repo.GetUserOverrides(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.ClearUserOverrides(ctx, userID); err != nil {
		return nil, err
	}
	return diffOverrides(before, nil), nil
}

// validateOverrides checks mode, scope and the catalogue, and returns the
// input with an empty deny scope filled in.
//
// A bad mode or scope answers ErrCapabilityKeyUnknown (422) rather than a
// sentinel of its own. That is the same call SetRolePermissions made for an
// out-of-range scope: the error message doubles as the frontend's error code,
// and a code rbac-format.ts does not know renders as "Unbekannter Fehler" —
// worse than a slightly broad but mapped one.
func (s *Service) validateOverrides(ctx context.Context, overrides []CapabilityOverride) ([]CapabilityOverride, error) {
	if len(overrides) == 0 {
		return nil, nil
	}

	normalized := make([]CapabilityOverride, len(overrides))
	keys := make([]string, len(overrides))
	for i, o := range overrides {
		if o.Mode != OverrideModeAllow && o.Mode != OverrideModeDeny {
			return nil, ErrCapabilityKeyUnknown
		}
		// A deny does not use the scope, so a caller that leaves it out is
		// not making a mistake — fill in the column default instead of
		// rejecting a request that says exactly what it means.
		scope := o.Scope
		if scope == "" && o.Mode == OverrideModeDeny {
			scope = ScopeAll
		}
		if _, ok := scopeRank[scope]; !ok {
			return nil, ErrCapabilityKeyUnknown
		}
		normalized[i] = CapabilityOverride{Key: o.Key, Mode: o.Mode, Scope: scope}
		keys[i] = o.Key
	}

	unknown, err := s.repo.CountUnknownPermissionKeys(ctx, keys)
	if err != nil {
		return nil, err
	}
	if unknown > 0 {
		return nil, ErrCapabilityKeyUnknown
	}
	return normalized, nil
}

// assertOverridesWithinReach applies the escalation guardrail
// (PHASE-1-RBAC-PLAN §4c) to the allow half of the map: nobody hands out a
// right they do not hold, and nobody widens a scope past their own.
//
// deny is deliberately not covered. Taking a right away from somebody else
// enlarges nobody, and an administrator who may not revoke rights they do not
// personally hold could not clean up after a colleague with a different role.
func (s *Service) assertOverridesWithinReach(ctx context.Context, actorID uuid.UUID, overrides []CapabilityOverride) error {
	allows := make([]RoleGrant, 0, len(overrides))
	for _, o := range overrides {
		if o.Mode == OverrideModeAllow {
			allows = append(allows, RoleGrant{Key: o.Key, Scope: o.Scope})
		}
	}
	return s.assertWithinReach(ctx, actorID, allows)
}

// assertOverridesKeepAdmins is the last-admin guardrail (§4a) on the override
// path: a deny may not take the role administration off the last account that
// still has it.
//
// It only runs when the new map actually denies every role-admin key the
// target currently holds — denying one of the pair while the other survives
// leaves them an administrator. And the count of everyone else is effective,
// not role-only: once overrides exist, an account can be an administrator
// purely through an allow, and a count that ignored that would refuse a
// perfectly safe deny.
func (s *Service) assertOverridesKeepAdmins(ctx context.Context, userID uuid.UUID, overrides []CapabilityOverride) error {
	denied := make(map[string]bool, len(overrides))
	allowed := false
	for _, o := range overrides {
		if !slices.Contains(roleAdminKeys, o.Key) {
			continue
		}
		if o.Mode == OverrideModeDeny {
			denied[o.Key] = true
		} else {
			allowed = true
		}
	}
	if len(denied) == 0 || allowed {
		return nil // nothing revoked, or another allow keeps them in
	}

	grants, err := s.repo.GetUserGrants(ctx, userID)
	if err != nil {
		return err
	}
	stillAdmin := false
	for _, row := range grants {
		if slices.Contains(roleAdminKeys, row.Key) && !denied[row.Key] {
			stillAdmin = true
			break
		}
	}
	if stillAdmin {
		return nil
	}

	remaining, err := s.repo.CountEffectiveRoleAdminsExcluding(ctx, roleAdminKeys, userID)
	if err != nil {
		return err
	}
	if remaining == 0 {
		return ErrLastAdmin
	}
	return nil
}

// diffOverrides pairs the map before a write against the one after it, one
// entry per key that changed. Keys whose mode and scope are identical produce
// no entry — a PUT that re-sends the unchanged map must not fill the audit log
// with events that describe nothing.
func diffOverrides(before, after []CapabilityOverride) []OverrideChange {
	byKey := func(list []CapabilityOverride) map[string]CapabilityOverride {
		m := make(map[string]CapabilityOverride, len(list))
		for _, o := range list {
			m[o.Key] = o
		}
		return m
	}
	oldMap, newMap := byKey(before), byKey(after)

	keys := make([]string, 0, len(oldMap)+len(newMap))
	for k := range oldMap {
		keys = append(keys, k)
	}
	for k := range newMap {
		if _, ok := oldMap[k]; !ok {
			keys = append(keys, k)
		}
	}
	slices.Sort(keys)

	changes := make([]OverrideChange, 0, len(keys))
	for _, key := range keys {
		oldVal, hadOld := oldMap[key]
		newVal, hasNew := newMap[key]
		if hadOld && hasNew && oldVal.Mode == newVal.Mode && oldVal.Scope == newVal.Scope {
			continue
		}
		change := OverrideChange{Key: key}
		if hadOld {
			change.Before = &CapabilityOverride{Key: key, Mode: oldVal.Mode, Scope: oldVal.Scope}
		}
		if hasNew {
			change.After = &CapabilityOverride{Key: key, Mode: newVal.Mode, Scope: newVal.Scope}
		}
		changes = append(changes, change)
	}
	return changes
}
