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

	// Invitation errors
	ErrInvitationNotFound    = errors.New("invitation not found")
	ErrInvitationExpired     = errors.New("invitation expired")
	ErrInvitationAlreadyUsed = errors.New("invitation already used")
	ErrInvitationExists      = errors.New("pending invitation already exists for this email")

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
