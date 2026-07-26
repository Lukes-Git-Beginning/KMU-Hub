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

	// Session errors
	ErrSessionNotFound = errors.New("session not found")

	// Password reset errors
	ErrResetTokenInvalid = errors.New("password reset token is invalid")
	ErrResetTokenExpired = errors.New("password reset token has expired")
	ErrResetTokenUsed    = errors.New("password reset token has already been used")
	ErrPasswordTooWeak   = errors.New("password does not meet strength requirements")
)
