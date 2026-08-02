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

	ErrTenantNotFound     = errors.New("tenant not found")

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

	// Session errors
	ErrSessionNotFound = errors.New("session not found")

	// Password reset errors
	ErrResetTokenInvalid = errors.New("password reset token is invalid")
	ErrResetTokenExpired = errors.New("password reset token has expired")
	ErrResetTokenUsed    = errors.New("password reset token has already been used")
	ErrPasswordTooWeak   = errors.New("password does not meet strength requirements")
)
