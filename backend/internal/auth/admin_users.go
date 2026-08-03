package auth

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// AdminUserStatus mirrors AdminUserStatus in admin-types.ts.
type AdminUserStatus string

const (
	AdminUserStatusActive      AdminUserStatus = "active"
	AdminUserStatusInvited     AdminUserStatus = "invited"
	AdminUserStatusDeactivated AdminUserStatus = "deactivated"
)

// AdminUser is one row of the tenant's account roster (admin-types.ts,
// AdminUser), merged from two sources that can never overlap for the same
// email: users (real accounts, active/deactivated) and invitations (still
// pending — no user row exists yet, so FirstName/LastName are empty and
// RoleIDs holds at most the one preset the invite named).
type AdminUser struct {
	ID          uuid.UUID
	FirstName   string
	LastName    string
	Email       string
	RoleIDs     []string
	Status      AdminUserStatus
	LastLoginAt *time.Time
	InvitedAt   *time.Time
}

// ListAdminUsers returns the calling tenant's account roster. Both sources
// are RLS-scoped (users, invitations), so no tenant id travels as a
// parameter — the same shape as ListUsers and ListRoles.
func (s *Service) ListAdminUsers(ctx context.Context) ([]AdminUser, error) {
	return s.repo.ListAdminUsers(ctx)
}

// InviteAdminUser opens an invitation for an address and returns the roster row
// it produces, plus the one-time token. The token travels back to the caller
// rather than into a mail, because that is how the existing invitation route
// works — internal/auth's Mailer only knows password resets.
//
// Every role id is resolved through GetRoleByID first, and that resolution IS
// the tenant boundary: roles carries an RLS read policy admitting presets and
// the caller's own tenant, so a foreign custom role is invisible and answers
// not_found. Without it the invitation could name a role from another tenant
// and the accept path would hand it out.
//
// Not capped by what the CALLER holds, deliberately, and for the same reason
// AssignUserRole documents: staffing a role richer than one's own is what
// admin:role:assign is for. What is capped there — taking a role for oneself —
// cannot happen here, since the account being invited does not exist yet.
func (s *Service) InviteAdminUser(ctx context.Context, tenantID, actorID uuid.UUID, email, firstName, lastName string, roleIDs []uuid.UUID) (*AdminUser, string, error) {
	if len(roleIDs) == 0 {
		// An account with no role has no rights at all and cannot be repaired
		// by the person accepting it. The accept path would refuse it anyway
		// (zero rows -> ErrRoleNotFound); refusing here says so at the door.
		return nil, "", ErrRoleNotFound
	}

	names := make([]string, len(roleIDs))
	for i, id := range roleIDs {
		role, err := s.repo.GetRoleByID(ctx, id)
		if err != nil {
			return nil, "", err
		}
		names[i] = role.Name
	}

	// names[0] fills the legacy display column only. The frontend sends roles
	// in the order the builder shows them, so the first one is the closest
	// thing to a primary role that exists here.
	inv, token, err := s.createInvitation(ctx, tenantID, email, firstName, lastName, names[0], roleIDs, actorID)
	if err != nil {
		return nil, "", err
	}

	row, err := s.repo.GetInvitationAsAdminUser(ctx, inv.ID)
	if err != nil {
		return nil, "", err
	}
	return row, token, nil
}

// ResendAdminUserInvite re-issues a pending invitation: a fresh token with a
// fresh expiry, replacing the old hash in place so the previous link stops
// working. Two live tokens to one account would be two ways in, and cancelling
// one would not cancel the other.
//
// The id is the INVITATION id — the id an invited roster row carries, because
// no account exists for it yet.
func (s *Service) ResendAdminUserInvite(ctx context.Context, tenantID, invitationID uuid.UUID) (*AdminUser, string, error) {
	token := generateSecureToken()

	inv, err := s.repo.RefreshInvitationToken(ctx, tenantID, invitationID, HashToken(token), time.Now().Add(invitationExpiry))
	if err != nil {
		return nil, "", err
	}

	row, err := s.repo.GetInvitationAsAdminUser(ctx, inv.ID)
	if err != nil {
		return nil, "", err
	}

	slog.Info("invitation resent", "invitation_id", inv.ID, "email", inv.Email)
	return row, token, nil
}

// UpdateAdminUser applies the two edits the account admin surface offers —
// a full replacement of the account's roles, and its active/deactivated state
// — and returns the resulting roster row. Both are optional; nil means
// untouched, which is why roleIDs is a pointer to a slice (an empty slice is a
// legitimate value meaning "strip every role").
//
// The role edit runs through AssignUserRole/RevokeUserRole rather than writing
// user_roles directly, so every wave-1b guardrail applies unchanged: last
// admin, self-lockout, and the escalation cap on assigning to oneself. Building
// a second path around them is how they stop being guardrails.
//
// The two edits are NOT atomic: the status guardrails are checked up front,
// but a failure while writing the status leaves an already-applied role change
// standing. That is the honest trade for reusing the guarded methods, and the
// frontend sends one or the other, never both (useAdminUsers.ts).
func (s *Service) UpdateAdminUser(ctx context.Context, actorID, userID uuid.UUID, roleIDs *[]uuid.UUID, status *AdminUserStatus) (*AdminUser, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	var activate *bool
	if status != nil {
		activate, err = s.plannedActivation(ctx, actorID, userID, user.TenantID, user.IsActive, *status)
		if err != nil {
			return nil, err
		}
	}

	if roleIDs != nil {
		if err := s.replaceUserRoles(ctx, actorID, userID, *roleIDs); err != nil {
			return nil, err
		}
	}

	if activate != nil {
		if _, err := s.UpdateUser(ctx, userID, nil, nil, activate); err != nil {
			return nil, err
		}
	}

	return s.repo.GetAdminUser(ctx, userID)
}

// plannedActivation validates a requested status change and returns the
// is_active value to write, or nil when the account is already in that state.
//
// Three refusals, all of them state conflicts rather than input errors:
//   - "invited" is not a state an existing account can be put into; it is
//     derived from a pending invitation, not stored.
//   - nobody deactivates themselves — the session stays valid until it
//     expires, so the mistake is silent until the next login fails.
//   - the last active account carrying role administration cannot go dark,
//     for the same reason ErrLastAdmin exists on the revoke path: a tenant
//     that loses it cannot give it back to itself.
//
// Reactivation takes a seat, so it runs the same check the invite path runs.
func (s *Service) plannedActivation(ctx context.Context, actorID, userID, tenantID uuid.UUID, isActive bool, status AdminUserStatus) (*bool, error) {
	var want bool
	switch status {
	case AdminUserStatusActive:
		want = true
	case AdminUserStatusDeactivated:
		want = false
	case AdminUserStatusInvited:
		return nil, ErrStatusNotAssignable
	default:
		return nil, ErrStatusNotAssignable
	}

	if want == isActive {
		return nil, nil
	}

	if !want {
		if actorID == userID {
			return nil, ErrSelfDeactivation
		}
		remaining, err := s.repo.CountActiveRoleAdminsExcludingUser(ctx, roleAdminKeys, userID)
		if err != nil {
			return nil, err
		}
		if remaining == 0 {
			return nil, ErrLastAdmin
		}
	} else if err := s.assertSeatAvailable(ctx, tenantID, nil); err != nil {
		return nil, err
	}

	return &want, nil
}

// replaceUserRoles brings the account's roles to exactly roleIDs, one guarded
// assign or revoke per difference. Revokes run first: taking a role away and
// putting a wider one on in the same request would otherwise trip the
// last-admin guard on the intermediate state.
func (s *Service) replaceUserRoles(ctx context.Context, actorID, userID uuid.UUID, roleIDs []uuid.UUID) error {
	current, err := s.repo.GetUserRoleIDs(ctx, userID)
	if err != nil {
		return err
	}

	want := make(map[string]uuid.UUID, len(roleIDs))
	for _, id := range roleIDs {
		want[id.String()] = id
	}

	for _, held := range current {
		if _, keep := want[held]; keep {
			delete(want, held)
			continue
		}
		roleID, parseErr := uuid.Parse(held)
		if parseErr != nil {
			return parseErr
		}
		if _, err := s.RevokeUserRole(ctx, actorID, userID, roleID); err != nil {
			return err
		}
	}

	for _, id := range want {
		if _, err := s.AssignUserRole(ctx, actorID, userID, id); err != nil {
			return err
		}
	}
	return nil
}
