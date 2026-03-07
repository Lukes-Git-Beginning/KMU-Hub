package server

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/auth"
)

func TestMapError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		// --- User errors ---
		{"user exists", auth.ErrUserExists, codes.AlreadyExists},
		{"invalid credentials", auth.ErrInvalidCredentials, codes.Unauthenticated},
		{"user not found", auth.ErrUserNotFound, codes.NotFound},
		{"user inactive", auth.ErrUserInactive, codes.PermissionDenied},
		// --- Token errors ---
		{"token expired", auth.ErrTokenExpired, codes.Unauthenticated},
		{"token revoked", auth.ErrTokenRevoked, codes.Unauthenticated},
		{"token invalid", auth.ErrTokenInvalid, codes.Unauthenticated},
		// --- Invitation errors ---
		{"invitation not found", auth.ErrInvitationNotFound, codes.NotFound},
		{"invitation expired", auth.ErrInvitationExpired, codes.FailedPrecondition},
		{"invitation already used", auth.ErrInvitationAlreadyUsed, codes.FailedPrecondition},
		{"invitation exists", auth.ErrInvitationExists, codes.AlreadyExists},
		// --- 2FA errors ---
		{"2fa already enabled", auth.ErrTwoFactorAlreadyEnabled, codes.FailedPrecondition},
		{"2fa not enabled", auth.ErrTwoFactorNotEnabled, codes.FailedPrecondition},
		{"no 2fa setup pending", auth.ErrNo2FASetupPending, codes.FailedPrecondition},
		{"invalid totp code", auth.ErrInvalidTOTPCode, codes.Unauthenticated},
		{"invalid recovery code", auth.ErrInvalidRecoveryCode, codes.Unauthenticated},
		{"all recovery codes used", auth.ErrAllRecoveryCodesUsed, codes.ResourceExhausted},
		{"invalid pending token", auth.ErrInvalidPendingToken, codes.Unauthenticated},
		{"pending token expired", auth.ErrPendingTokenExpired, codes.Unauthenticated},
		{"2fa enforcement required", auth.Err2FAEnforcementRequired, codes.FailedPrecondition},
		// --- Session errors ---
		{"session not found", auth.ErrSessionNotFound, codes.NotFound},
		// --- Fallback ---
		{"unknown error", errors.New("boom"), codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireGRPCCode(t, mapError(tt.err), tt.wantCode)
		})
	}
}
