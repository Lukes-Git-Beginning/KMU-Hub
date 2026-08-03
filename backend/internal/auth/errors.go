package auth

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExists         = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrTokenExpired       = errors.New("token expired")
	ErrTokenRevoked       = errors.New("token revoked")
	ErrTokenInvalid       = errors.New("token invalid")
	ErrUserInactive       = errors.New("user account is inactive")
	ErrRoleNotFound       = errors.New("role not found")

	ErrTenantNotFound = errors.New("tenant not found")

	// Tenant provisioning errors. All of them are input problems and map to
	// 400 — a caller who cannot spell a plan type should not see a 500.
	ErrTenantNameRequired  = errors.New("tenant name is required")
	ErrTenantNameTooLong   = errors.New("tenant name is too long")
	ErrInvalidPlanType     = errors.New("plan type must be cosmi or orbit")
	ErrInvalidSupportTier  = errors.New("support tier must be standard, priority or enterprise")
	ErrInvalidSeatLimit    = errors.New("seat limit must be at least 1: the administrator invitation takes the first seat")
	ErrAdminEmailRequired  = errors.New("a valid administrator email is required")
	ErrProvisionerRequired = errors.New("provisioning user is required")
	ErrUnknownModule       = errors.New("unknown module id")

	// Invitation errors
	ErrInvitationNotFound    = errors.New("invitation not found")
	ErrInvitationExpired     = errors.New("invitation expired")
	ErrInvitationAlreadyUsed = errors.New("invitation already used")
	ErrInvitationExists      = errors.New("pending invitation already exists for this email")
	ErrSeatLimitReached      = errors.New("no seat available: the booked seats are all taken by active users and pending invitations")

	// Two-factor authentication errors
	ErrTwoFactorAlreadyEnabled = errors.New("two-factor authentication is already enabled")
	ErrTwoFactorNotEnabled     = errors.New("two-factor authentication is not enabled")
	ErrNo2FASetupPending       = errors.New("no 2FA setup pending")
	ErrInvalidTOTPCode         = errors.New("invalid TOTP code")
	ErrInvalidRecoveryCode     = errors.New("invalid recovery code")
	ErrAllRecoveryCodesUsed    = errors.New("all recovery codes have been used")
	ErrInvalidPendingToken     = errors.New("invalid pending token")
	ErrPendingTokenExpired     = errors.New("pending token expired")
	Err2FAEnforcementRequired  = errors.New("two-factor authentication setup required by policy")

	// Role administration errors (RBAC phase 1 wave 1b). Their messages are
	// the error CODES the RBAC frontend maps to i18n (rbac-format.ts,
	// RBAC_ERROR_CODES) — mapError passes err.Error() through as the gRPC
	// status message and the gateway renders it as {"error": "<code>"}, so a
	// prose message here would reach the user as "Unbekannter Fehler".
	ErrRoleLimitReached = errors.New("role_limit_reached")
	ErrRoleNameExists   = errors.New("role_name_exists")
	// ErrBaseRoleNotFound is a distinct sentinel from ErrRoleNotFound: the
	// latter is the legacy "unknown role name" of AssignRole and maps to 409,
	// while a missing clone source has to reach the builder as a 404. It also
	// covers a role from another tenant: the RLS read policy makes it
	// invisible, so "foreign role" and "unknown id" are indistinguishable and
	// must answer the same way.
	ErrBaseRoleNotFound = errors.New("not_found")
	// ErrRolePresetImmutable fires on PATCH/DELETE against a system preset.
	// The write policy already confines those statements to zero rows, but a
	// silent no-op would look like a successful save to the builder.
	ErrRolePresetImmutable = errors.New("preset_immutable")
	// ErrRoleHasMembers blocks deleting a role that is still assigned to at
	// least one account in the caller's tenant.
	ErrRoleHasMembers = errors.New("role_has_members")
	// ErrCapabilityKeyUnknown fires when a PUT grant set names a key the
	// permissions catalogue does not have, or (as a defense-in-depth backstop
	// the frontend's CapabilityScope union should make unreachable) a scope
	// outside own|team|all. Accepting either silently would let the builder
	// believe it saved a right nothing in the system checks for.
	ErrCapabilityKeyUnknown = errors.New("unknown_capability_key")
	// ErrLastAdmin blocks taking the role-administration capability off the
	// last account in the tenant that still carries it. A tenant that loses it
	// cannot hand it back to itself — only Zentria could, by hand, in the
	// database.
	ErrLastAdmin = errors.New("last_admin")
	// ErrSelfLockout blocks the caller from removing the very capability they
	// are using right now, while other administrators still exist. It is not a
	// weaker ErrLastAdmin: someone can lock themselves out of a tenant that
	// still has three other admins, and would then need one of them to undo it.
	//
	// The message is not (yet) in the frontend's RBAC_ERROR_CODES, so the
	// builder renders the generic message until rbac-format.ts and the four
	// message catalogues learn the code.
	ErrSelfLockout = errors.New("self_lockout")
	// ErrPrivilegeEscalation blocks defining or taking on a right the caller
	// does not hold themselves — including widening a scope (own -> all) past
	// their own. Same frontend caveat as ErrSelfLockout.
	ErrPrivilegeEscalation = errors.New("privilege_escalation")
	// ErrSelfDeactivation blocks an administrator from switching their own
	// account off. It is not covered by ErrSelfLockout, which is about losing
	// a capability: deactivation takes away every capability at once, and does
	// it silently — the current session keeps working until it expires, so the
	// mistake only surfaces at the next login.
	ErrSelfDeactivation = errors.New("self_deactivation")
	// ErrStatusNotAssignable rejects a status an account cannot be put into.
	// "invited" is the only one: it is derived from a pending invitation, not
	// a state a real account can hold, so accepting it would silently do
	// nothing.
	ErrStatusNotAssignable = errors.New("status_not_assignable")

	// Session errors
	ErrSessionNotFound = errors.New("session not found")

	// Password reset errors
	ErrResetTokenInvalid = errors.New("password reset token is invalid")
	ErrResetTokenExpired = errors.New("password reset token has expired")
	ErrResetTokenUsed    = errors.New("password reset token has already been used")
	ErrPasswordTooWeak   = errors.New("password does not meet strength requirements")
)
