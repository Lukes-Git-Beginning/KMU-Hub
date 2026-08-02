package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	authv1 "github.com/kmuhub/kmuhub/proto/auth/v1"
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
		// --- Role administration (RBAC wave 1b) ---
		// The gateway turns AlreadyExists and FailedPrecondition into 409 and
		// NotFound into 404, which is the contract the RBAC builder expects.
		{"role name exists", auth.ErrRoleNameExists, codes.AlreadyExists},
		{"role limit reached", auth.ErrRoleLimitReached, codes.FailedPrecondition},
		{"clone source not found", auth.ErrBaseRoleNotFound, codes.NotFound},
		{"preset immutable", auth.ErrRolePresetImmutable, codes.PermissionDenied},
		{"role has members", auth.ErrRoleHasMembers, codes.FailedPrecondition},
		// --- Guardrails (RBAC wave 1b) ---
		// last_admin and self_lockout are conflicts with the tenant's current
		// state (409), privilege_escalation is a rights problem (403) — the
		// same code preset_immutable gets. guardrails_db_test.go proves the
		// auth.Service returns these sentinels; this proves mapError still
		// carries them to the right gRPC code, which is the only part of the
		// chain a copy-paste error in the switch would not otherwise catch.
		{"last admin", auth.ErrLastAdmin, codes.FailedPrecondition},
		{"self lockout", auth.ErrSelfLockout, codes.FailedPrecondition},
		{"privilege escalation", auth.ErrPrivilegeEscalation, codes.PermissionDenied},
		// --- Fallback ---
		{"unknown error", errors.New("boom"), codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireGRPCCode(t, mapError(tt.err), tt.wantCode)
		})
	}
}

// ============================================================================
// Core Auth Endpoints
// ============================================================================

func TestAuthGRPC_Register(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		resp, err := srv.Register(ctx, &authv1.RegisterRequest{
			Email:     "new@example.com",
			Password:  "StrongPass123!",
			FirstName: "First",
			LastName:  "Last",
		})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.User)
		assert.Equal(t, "new@example.com", resp.User.Email)
		assert.Equal(t, "First", resp.User.FirstName)
		assert.Equal(t, "Last", resp.User.LastName)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.Contains(t, resp.User.Roles, "member")
	})

	t.Run("duplicate email", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		createAuthTestUser(repo, "dup@example.com", "pass", true)

		_, err := srv.Register(ctx, &authv1.RegisterRequest{
			Email:     "dup@example.com",
			Password:  "StrongPass123!",
			FirstName: "A",
			LastName:  "B",
		})
		requireGRPCCode(t, err, codes.AlreadyExists)
	})
}

func TestAuthGRPC_Login(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		createAuthTestUser(repo, "user@example.com", "correct-password", true)

		resp, err := srv.Login(ctx, &authv1.LoginRequest{
			Email:    "user@example.com",
			Password: "correct-password",
		})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.User)
		assert.Equal(t, "user@example.com", resp.User.Email)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.False(t, resp.RequiresTwoFactor)
	})

	t.Run("wrong password", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		createAuthTestUser(repo, "user@example.com", "correct-password", true)

		_, err := srv.Login(ctx, &authv1.LoginRequest{
			Email:    "user@example.com",
			Password: "wrong-password",
		})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("user not found", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()

		_, err := srv.Login(ctx, &authv1.LoginRequest{
			Email:    "nobody@example.com",
			Password: "pass",
		})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("inactive user", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		createAuthTestUser(repo, "inactive@example.com", "pass", false)

		_, err := srv.Login(ctx, &authv1.LoginRequest{
			Email:    "inactive@example.com",
			Password: "pass",
		})
		requireGRPCCode(t, err, codes.PermissionDenied)
	})
}

func TestAuthGRPC_RefreshToken(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		createAuthTestUser(repo, "user@example.com", "pass", true)
		loginResp, err := srv.Login(ctx, &authv1.LoginRequest{Email: "user@example.com", Password: "pass"})
		require.NoError(t, err)

		resp, err := srv.RefreshToken(ctx, &authv1.RefreshTokenRequest{
			RefreshToken: loginResp.RefreshToken,
		})
		requireGRPCOK(t, err)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
	})

	t.Run("invalid token", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()

		_, err := srv.RefreshToken(ctx, &authv1.RefreshTokenRequest{
			RefreshToken: "nonexistent-token",
		})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
}

func TestAuthGRPC_Logout(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		createAuthTestUser(repo, "user@example.com", "pass", true)
		loginResp, _ := srv.Login(ctx, &authv1.LoginRequest{Email: "user@example.com", Password: "pass"})

		resp, err := srv.Logout(ctx, &authv1.LogoutRequest{
			RefreshToken: loginResp.RefreshToken,
		})
		requireGRPCOK(t, err)
		require.NotNil(t, resp)
	})

	t.Run("nonexistent token is idempotent", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()

		resp, err := srv.Logout(ctx, &authv1.LogoutRequest{
			RefreshToken: "nonexistent",
		})
		requireGRPCOK(t, err)
		require.NotNil(t, resp)
	})
}

func TestAuthGRPC_ValidateToken(t *testing.T) {
	ctx := context.Background()

	t.Run("valid token", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		createAuthTestUser(repo, "user@example.com", "pass", true)
		loginResp, _ := srv.Login(ctx, &authv1.LoginRequest{Email: "user@example.com", Password: "pass"})

		resp, err := srv.ValidateToken(ctx, &authv1.ValidateTokenRequest{
			AccessToken: loginResp.AccessToken,
		})
		requireGRPCOK(t, err)
		assert.True(t, resp.Valid)
		assert.NotEmpty(t, resp.UserId)
	})

	t.Run("invalid token returns valid=false without error", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()

		resp, err := srv.ValidateToken(ctx, &authv1.ValidateTokenRequest{
			AccessToken: "garbage",
		})
		requireGRPCOK(t, err)
		assert.False(t, resp.Valid)
	})
}

// ============================================================================
// User Management
// ============================================================================

func TestAuthGRPC_GetUser(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		user := createAuthTestUser(repo, "user@example.com", "pass", true)
		repo.userRoles[user.ID] = []string{"admin"}

		resp, err := srv.GetUser(ctx, &authv1.GetUserRequest{UserId: user.ID.String()})
		requireGRPCOK(t, err)
		assert.Equal(t, user.Email, resp.User.Email)
		assert.Contains(t, resp.User.Roles, "admin")
	})

	t.Run("invalid uuid", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()

		_, err := srv.GetUser(ctx, &authv1.GetUserRequest{UserId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not found", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()

		_, err := srv.GetUser(ctx, &authv1.GetUserRequest{UserId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

func TestAuthGRPC_ListUsers(t *testing.T) {
	ctx := context.Background()

	t.Run("returns users with pagination", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		createAuthTestUser(repo, "a@example.com", "pass", true)
		createAuthTestUser(repo, "b@example.com", "pass", true)
		createAuthTestUser(repo, "c@example.com", "pass", true)

		resp, err := srv.ListUsers(ctx, &authv1.ListUsersRequest{Page: 1, PageSize: 2})
		requireGRPCOK(t, err)
		assert.Equal(t, int32(3), resp.Total)
		assert.Len(t, resp.Users, 2)
	})

	t.Run("empty list", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()

		resp, err := srv.ListUsers(ctx, &authv1.ListUsersRequest{Page: 1, PageSize: 10})
		requireGRPCOK(t, err)
		assert.Equal(t, int32(0), resp.Total)
		assert.Empty(t, resp.Users)
	})
}

func TestAuthGRPC_UpdateUser(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		user := createAuthTestUser(repo, "user@example.com", "pass", true)

		newFirst := "Updated"
		resp, err := srv.UpdateUser(ctx, &authv1.UpdateUserRequest{
			UserId:    user.ID.String(),
			FirstName: &newFirst,
		})
		requireGRPCOK(t, err)
		assert.Equal(t, "Updated", resp.User.FirstName)
	})

	t.Run("invalid uuid", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()

		_, err := srv.UpdateUser(ctx, &authv1.UpdateUserRequest{UserId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not found", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		name := "X"
		_, err := srv.UpdateUser(ctx, &authv1.UpdateUserRequest{
			UserId:    uuid.New().String(),
			FirstName: &name,
		})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

// ============================================================================
// RBAC
// ============================================================================

func TestAuthGRPC_AssignRole(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		user := createAuthTestUser(repo, "user@example.com", "pass", true)

		_, err := srv.AssignRole(ctx, &authv1.AssignRoleRequest{
			UserId:   user.ID.String(),
			RoleName: "admin",
		})
		requireGRPCOK(t, err)
		assert.Contains(t, repo.userRoles[user.ID], "admin")
	})

	t.Run("invalid uuid", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()

		_, err := srv.AssignRole(ctx, &authv1.AssignRoleRequest{UserId: "bad", RoleName: "admin"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("user not found", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()

		_, err := srv.AssignRole(ctx, &authv1.AssignRoleRequest{
			UserId:   uuid.New().String(),
			RoleName: "admin",
		})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

func TestAuthGRPC_RemoveRole(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		user := createAuthTestUser(repo, "user@example.com", "pass", true)
		repo.userRoles[user.ID] = []string{"admin"}

		_, err := srv.RemoveRole(ctx, &authv1.RemoveRoleRequest{
			UserId:   user.ID.String(),
			RoleName: "admin",
		})
		requireGRPCOK(t, err)
		assert.NotContains(t, repo.userRoles[user.ID], "admin")
	})

	t.Run("invalid uuid", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()

		_, err := srv.RemoveRole(ctx, &authv1.RemoveRoleRequest{UserId: "bad", RoleName: "admin"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestAuthGRPC_CheckPermission(t *testing.T) {
	ctx := context.Background()

	t.Run("allowed", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		user := createAuthTestUser(repo, "user@example.com", "pass", true)
		repo.userPerms[user.ID] = []string{"contacts:read"}

		resp, err := srv.CheckPermission(ctx, &authv1.CheckPermissionRequest{
			UserId:   user.ID.String(),
			Resource: "contacts",
			Action:   "read",
		})
		requireGRPCOK(t, err)
		assert.True(t, resp.Allowed)
	})

	t.Run("denied", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		user := createAuthTestUser(repo, "user@example.com", "pass", true)

		resp, err := srv.CheckPermission(ctx, &authv1.CheckPermissionRequest{
			UserId:   user.ID.String(),
			Resource: "contacts",
			Action:   "delete",
		})
		requireGRPCOK(t, err)
		assert.False(t, resp.Allowed)
	})

	t.Run("invalid uuid", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()

		_, err := srv.CheckPermission(ctx, &authv1.CheckPermissionRequest{
			UserId: "bad", Resource: "x", Action: "y",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestAuthGRPC_GetEffectivePermissions(t *testing.T) {
	ctx := context.Background()

	t.Run("explicit user id", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		user := createAuthTestUser(repo, "user@example.com", "pass", true)
		roleID := uuid.New()
		repo.effectiveGrants[user.ID] = []auth.EffectiveGrantRow{
			{RoleID: roleID, RoleName: "member", RoleColor: "hsl(0 0% 50%)", RoleIsSystem: true, Key: "work:task:edit", Scope: "team"},
		}

		resp, err := srv.GetEffectivePermissions(ctx, &authv1.GetEffectivePermissionsRequest{UserId: user.ID.String()})
		requireGRPCOK(t, err)
		require.Len(t, resp.Roles, 1)
		assert.Equal(t, "member", resp.Roles[0].Name)
		assert.True(t, resp.Roles[0].IsSystem)
		require.Len(t, resp.Capabilities, 1)
		assert.Equal(t, "work:task:edit", resp.Capabilities[0].Key)
		assert.Equal(t, "team", resp.Capabilities[0].Scope)
		// Sources carries role IDs — the frontend resolves them against Roles.
		assert.Equal(t, []string{roleID.String()}, resp.Capabilities[0].Sources)
	})

	t.Run("empty user id falls back to caller from context", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		user := createAuthTestUser(repo, "caller@example.com", "pass", true)
		repo.effectiveGrants[user.ID] = []auth.EffectiveGrantRow{
			{RoleID: uuid.New(), RoleName: "extern", Key: "documents:file:read", Scope: "own"},
		}
		callerCtx := context.WithValue(ctx, middleware.UserIDKey, user.ID.String())

		resp, err := srv.GetEffectivePermissions(callerCtx, &authv1.GetEffectivePermissionsRequest{})
		requireGRPCOK(t, err)
		require.Len(t, resp.Capabilities, 1)
		assert.Equal(t, "documents:file:read", resp.Capabilities[0].Key)
	})

	t.Run("no roles resolves to empty, never nil, slices", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		user := createAuthTestUser(repo, "norole@example.com", "pass", true)

		resp, err := srv.GetEffectivePermissions(ctx, &authv1.GetEffectivePermissionsRequest{UserId: user.ID.String()})
		requireGRPCOK(t, err)
		assert.NotNil(t, resp.Roles)
		assert.NotNil(t, resp.Capabilities)
		assert.Empty(t, resp.Roles)
		assert.Empty(t, resp.Capabilities)
	})

	t.Run("invalid uuid", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()

		_, err := srv.GetEffectivePermissions(ctx, &authv1.GetEffectivePermissionsRequest{UserId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("empty user id and no caller in context", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()

		_, err := srv.GetEffectivePermissions(ctx, &authv1.GetEffectivePermissionsRequest{})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

// ============================================================================
// Profile & Password
// ============================================================================

func TestAuthGRPC_GetProfile(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		user := createAuthTestUser(repo, "user@example.com", "pass", true)
		repo.userRoles[user.ID] = []string{"member"}

		resp, err := srv.GetProfile(ctx, &authv1.GetProfileRequest{UserId: user.ID.String()})
		requireGRPCOK(t, err)
		assert.Equal(t, user.Email, resp.User.Email)
		assert.Contains(t, resp.User.Roles, "member")
	})

	t.Run("invalid uuid", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		_, err := srv.GetProfile(ctx, &authv1.GetProfileRequest{UserId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not found", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		_, err := srv.GetProfile(ctx, &authv1.GetProfileRequest{UserId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

func TestAuthGRPC_ChangePassword(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		user := createAuthTestUser(repo, "user@example.com", "old-password", true)

		_, err := srv.ChangePassword(ctx, &authv1.ChangePasswordRequest{
			UserId:      user.ID.String(),
			OldPassword: "old-password",
			NewPassword: "new-password",
		})
		requireGRPCOK(t, err)
	})

	t.Run("wrong old password", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		user := createAuthTestUser(repo, "user@example.com", "correct", true)

		_, err := srv.ChangePassword(ctx, &authv1.ChangePasswordRequest{
			UserId:      user.ID.String(),
			OldPassword: "wrong",
			NewPassword: "new-password",
		})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("invalid uuid", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		_, err := srv.ChangePassword(ctx, &authv1.ChangePasswordRequest{
			UserId: "bad", OldPassword: "a", NewPassword: "b",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("user not found", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		_, err := srv.ChangePassword(ctx, &authv1.ChangePasswordRequest{
			UserId:      uuid.New().String(),
			OldPassword: "a",
			NewPassword: "b",
		})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

// ============================================================================
// Invitations
// ============================================================================

// invitationTestTenant is the tenant every invitation test acts as. The
// invitation RPCs read it from the context — an invitation without a tenant is
// exactly what migration 000249 removed.
var invitationTestTenant = uuid.MustParse("b2b2b2b2-0000-4000-8000-000000000002")

func invitationCtx() context.Context {
	return context.WithValue(context.Background(), middleware.TenantIDKey, invitationTestTenant.String())
}

func TestAuthGRPC_CreateInvitation(t *testing.T) {
	ctx := invitationCtx()

	t.Run("success", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		createdBy := uuid.New()

		resp, err := srv.CreateInvitation(ctx, &authv1.CreateInvitationRequest{
			Email:     "invite@example.com",
			Role:      "member",
			CreatedBy: createdBy.String(),
		})
		requireGRPCOK(t, err)
		assert.Equal(t, "invite@example.com", resp.Invitation.Email)
		assert.Equal(t, "member", resp.Invitation.Role)
		assert.NotEmpty(t, resp.Token)
	})

	t.Run("user already exists", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		createAuthTestUser(repo, "existing@example.com", "pass", true)

		_, err := srv.CreateInvitation(ctx, &authv1.CreateInvitationRequest{
			Email:     "existing@example.com",
			Role:      "member",
			CreatedBy: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.AlreadyExists)
	})

	t.Run("invalid created_by uuid", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()

		_, err := srv.CreateInvitation(ctx, &authv1.CreateInvitationRequest{
			Email:     "new@example.com",
			Role:      "member",
			CreatedBy: "bad",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestAuthGRPC_ListInvitations(t *testing.T) {
	ctx := invitationCtx()

	t.Run("returns pending invitations", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		createdBy := uuid.New()
		_, _ = srv.CreateInvitation(ctx, &authv1.CreateInvitationRequest{
			Email: "a@example.com", Role: "member", CreatedBy: createdBy.String(),
		})
		_, _ = srv.CreateInvitation(ctx, &authv1.CreateInvitationRequest{
			Email: "b@example.com", Role: "admin", CreatedBy: createdBy.String(),
		})

		resp, err := srv.ListInvitations(ctx, &authv1.ListInvitationsRequest{})
		requireGRPCOK(t, err)
		assert.Len(t, resp.Invitations, 2)
	})
}

func TestAuthGRPC_AcceptInvitation(t *testing.T) {
	ctx := invitationCtx()

	t.Run("success", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		createResp, _ := srv.CreateInvitation(ctx, &authv1.CreateInvitationRequest{
			Email: "new@example.com", Role: "member", CreatedBy: uuid.New().String(),
		})

		resp, err := srv.AcceptInvitation(ctx, &authv1.AcceptInvitationRequest{
			Token:     createResp.Token,
			Password:  "StrongPass123!",
			FirstName: "First",
			LastName:  "Last",
		})
		requireGRPCOK(t, err)
		assert.Equal(t, "new@example.com", resp.User.Email)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
	})

	t.Run("invalid token", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()

		_, err := srv.AcceptInvitation(ctx, &authv1.AcceptInvitationRequest{
			Token:     "nonexistent",
			Password:  "pass",
			FirstName: "A",
			LastName:  "B",
		})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("expired invitation", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		inv := &models.Invitation{
			ID:        uuid.New(),
			Email:     "expired@example.com",
			Role:      "member",
			TokenHash: auth.HashToken("expired-token"),
			CreatedBy: uuid.New(),
			ExpiresAt: time.Now().Add(-1 * time.Hour),
			CreatedAt: time.Now(),
		}
		repo.invitations[inv.ID] = inv
		repo.invByToken[inv.TokenHash] = inv

		_, err := srv.AcceptInvitation(ctx, &authv1.AcceptInvitationRequest{
			Token: "expired-token", Password: "pass", FirstName: "A", LastName: "B",
		})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})
}

func TestAuthGRPC_CancelInvitation(t *testing.T) {
	ctx := invitationCtx()

	t.Run("success", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		createResp, _ := srv.CreateInvitation(ctx, &authv1.CreateInvitationRequest{
			Email: "cancel@example.com", Role: "member", CreatedBy: uuid.New().String(),
		})

		invID := createResp.Invitation.Id
		_, err := srv.CancelInvitation(ctx, &authv1.CancelInvitationRequest{InvitationId: invID})
		requireGRPCOK(t, err)
	})

	t.Run("invalid uuid", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		_, err := srv.CancelInvitation(ctx, &authv1.CancelInvitationRequest{InvitationId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not found", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		_, err := srv.CancelInvitation(ctx, &authv1.CancelInvitationRequest{InvitationId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})
}

// ============================================================================
// 2FA Endpoints
// ============================================================================

func TestAuthGRPC_Setup2FA(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		user := createAuthTestUser(repo, "user@example.com", "pass", true)

		resp, err := srv.Setup2FA(ctx, &authv1.Setup2FARequest{UserId: user.ID.String()})
		requireGRPCOK(t, err)
		assert.NotEmpty(t, resp.QrCodePng)
		assert.NotEmpty(t, resp.ManualSecret)
	})

	t.Run("invalid uuid", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		_, err := srv.Setup2FA(ctx, &authv1.Setup2FARequest{UserId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("user not found", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		_, err := srv.Setup2FA(ctx, &authv1.Setup2FARequest{UserId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("already enabled", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		user := createAuthTestUser(repo, "user@example.com", "pass", true)
		user.TwoFactorEnabled = true

		_, err := srv.Setup2FA(ctx, &authv1.Setup2FARequest{UserId: user.ID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})
}

func TestAuthGRPC_Verify2FA(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid uuid", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		_, err := srv.Verify2FA(ctx, &authv1.Verify2FARequest{UserId: "bad", Code: "123456"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("no setup pending", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		user := createAuthTestUser(repo, "user@example.com", "pass", true)

		_, err := srv.Verify2FA(ctx, &authv1.Verify2FARequest{UserId: user.ID.String(), Code: "123456"})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})
}

func TestAuthGRPC_Disable2FA(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid uuid", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		_, err := srv.Disable2FA(ctx, &authv1.Disable2FARequest{UserId: "bad", Code: "123456"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("2fa not enabled", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		user := createAuthTestUser(repo, "user@example.com", "pass", true)

		_, err := srv.Disable2FA(ctx, &authv1.Disable2FARequest{UserId: user.ID.String(), Code: "123456"})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})
}

func TestAuthGRPC_RegenerateRecoveryCodes(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid uuid", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		_, err := srv.RegenerateRecoveryCodes(ctx, &authv1.RegenerateRecoveryCodesRequest{UserId: "bad", Code: "123456"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("2fa not enabled", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		user := createAuthTestUser(repo, "user@example.com", "pass", true)

		_, err := srv.RegenerateRecoveryCodes(ctx, &authv1.RegenerateRecoveryCodesRequest{UserId: user.ID.String(), Code: "123456"})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})
}

func TestAuthGRPC_AdminReset2FA(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid user uuid", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		_, err := srv.AdminReset2FA(ctx, &authv1.AdminReset2FARequest{
			UserId: "bad", AdminId: uuid.New().String(), Reason: "test",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid admin uuid", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		_, err := srv.AdminReset2FA(ctx, &authv1.AdminReset2FARequest{
			UserId: uuid.New().String(), AdminId: "bad", Reason: "test",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("user not found", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		_, err := srv.AdminReset2FA(ctx, &authv1.AdminReset2FARequest{
			UserId: uuid.New().String(), AdminId: uuid.New().String(), Reason: "test",
		})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("2fa not enabled", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		user := createAuthTestUser(repo, "user@example.com", "pass", true)

		_, err := srv.AdminReset2FA(ctx, &authv1.AdminReset2FARequest{
			UserId: user.ID.String(), AdminId: uuid.New().String(), Reason: "test",
		})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})
}

func TestAuthGRPC_Validate2FALogin(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid pending token", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		_, err := srv.Validate2FALogin(ctx, &authv1.Validate2FALoginRequest{
			PendingToken: "nonexistent", Code: "123456",
		})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})
}

// ============================================================================
// Sessions
// ============================================================================

func TestAuthGRPC_ListSessions(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		user := createAuthTestUser(repo, "user@example.com", "pass", true)
		sessID := uuid.New()
		repo.sessions[sessID] = &models.UserSession{
			ID:           sessID,
			UserID:       user.ID,
			DeviceName:   "Chrome",
			DeviceType:   "browser",
			IPAddress:    "127.0.0.1",
			LastActiveAt: time.Now(),
			CreatedAt:    time.Now(),
		}

		resp, err := srv.ListSessions(ctx, &authv1.ListSessionsRequest{UserId: user.ID.String()})
		requireGRPCOK(t, err)
		assert.Len(t, resp.Sessions, 1)
		assert.Equal(t, "Chrome", resp.Sessions[0].DeviceName)
	})

	t.Run("invalid uuid", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		_, err := srv.ListSessions(ctx, &authv1.ListSessionsRequest{UserId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestAuthGRPC_TerminateSession(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		user := createAuthTestUser(repo, "user@example.com", "pass", true)
		sessID := uuid.New()
		repo.sessions[sessID] = &models.UserSession{
			ID: sessID, UserID: user.ID, CreatedAt: time.Now(), LastActiveAt: time.Now(),
		}

		_, err := srv.TerminateSession(ctx, &authv1.TerminateSessionRequest{
			SessionId: sessID.String(),
			UserId:    user.ID.String(),
		})
		requireGRPCOK(t, err)
	})

	t.Run("invalid session uuid", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		_, err := srv.TerminateSession(ctx, &authv1.TerminateSessionRequest{
			SessionId: "bad", UserId: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid user uuid", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		_, err := srv.TerminateSession(ctx, &authv1.TerminateSessionRequest{
			SessionId: uuid.New().String(), UserId: "bad",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	// The gateway fills UserId from the JWT, SessionId comes from the URL —
	// so this is the only thing standing between a signed-in user and signing
	// out anyone else whose session id they can guess.
	t.Run("session of another user is not found", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		owner := createAuthTestUser(repo, "owner@example.com", "pass", true)
		stranger := createAuthTestUser(repo, "stranger@example.com", "pass", true)
		sessID := uuid.New()
		repo.sessions[sessID] = &models.UserSession{
			ID: sessID, UserID: owner.ID, CreatedAt: time.Now(), LastActiveAt: time.Now(),
		}

		_, err := srv.TerminateSession(ctx, &authv1.TerminateSessionRequest{
			SessionId: sessID.String(),
			UserId:    stranger.ID.String(),
		})
		requireGRPCCode(t, err, codes.NotFound)

		if _, ok := repo.sessions[sessID]; !ok {
			t.Error("the owner's session was deleted by a stranger's request")
		}
	})
}

func TestAuthGRPC_TerminateAllSessions(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		user := createAuthTestUser(repo, "user@example.com", "pass", true)
		for i := 0; i < 3; i++ {
			id := uuid.New()
			repo.sessions[id] = &models.UserSession{
				ID: id, UserID: user.ID, CreatedAt: time.Now(), LastActiveAt: time.Now(),
			}
		}

		_, err := srv.TerminateAllSessions(ctx, &authv1.TerminateAllSessionsRequest{
			UserId: user.ID.String(),
		})
		requireGRPCOK(t, err)
	})

	t.Run("with current session excluded", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		user := createAuthTestUser(repo, "user@example.com", "pass", true)
		currentID := uuid.New()
		otherID := uuid.New()
		repo.sessions[currentID] = &models.UserSession{
			ID: currentID, UserID: user.ID, CreatedAt: time.Now(), LastActiveAt: time.Now(),
		}
		repo.sessions[otherID] = &models.UserSession{
			ID: otherID, UserID: user.ID, CreatedAt: time.Now(), LastActiveAt: time.Now(),
		}

		_, err := srv.TerminateAllSessions(ctx, &authv1.TerminateAllSessionsRequest{
			UserId:           user.ID.String(),
			CurrentSessionId: currentID.String(),
		})
		requireGRPCOK(t, err)
		// Current session should be preserved
		_, exists := repo.sessions[currentID]
		assert.True(t, exists)
		// Other session should be deleted
		_, exists = repo.sessions[otherID]
		assert.False(t, exists)
	})

	t.Run("invalid uuid", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		_, err := srv.TerminateAllSessions(ctx, &authv1.TerminateAllSessionsRequest{UserId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid current session uuid", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		_, err := srv.TerminateAllSessions(ctx, &authv1.TerminateAllSessionsRequest{
			UserId:           uuid.New().String(),
			CurrentSessionId: "bad",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

// ============================================================================
// 2FA Policy
// ============================================================================

func TestAuthGRPC_GetTwoFactorPolicy(t *testing.T) {
	ctx := context.Background()

	t.Run("by role name", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		repo.policies["admin"] = &models.TwoFactorPolicy{
			ID:              uuid.New(),
			RoleName:        "admin",
			Enforced:        true,
			GracePeriodDays: 7,
			UpdatedAt:       time.Now(),
		}

		resp, err := srv.GetTwoFactorPolicy(ctx, &authv1.GetTwoFactorPolicyRequest{RoleName: "admin"})
		requireGRPCOK(t, err)
		require.Len(t, resp.Policies, 1)
		assert.Equal(t, "admin", resp.Policies[0].RoleName)
		assert.True(t, resp.Policies[0].Enforced)
	})

	t.Run("list all policies", func(t *testing.T) {
		srv, repo := newTestAuthGRPCServer()
		repo.policies["admin"] = &models.TwoFactorPolicy{
			ID: uuid.New(), RoleName: "admin", Enforced: true, UpdatedAt: time.Now(),
		}
		repo.policies["member"] = &models.TwoFactorPolicy{
			ID: uuid.New(), RoleName: "member", Enforced: false, UpdatedAt: time.Now(),
		}

		resp, err := srv.GetTwoFactorPolicy(ctx, &authv1.GetTwoFactorPolicyRequest{})
		requireGRPCOK(t, err)
		assert.Len(t, resp.Policies, 2)
	})

	t.Run("role not found returns empty", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()

		resp, err := srv.GetTwoFactorPolicy(ctx, &authv1.GetTwoFactorPolicyRequest{RoleName: "nonexistent"})
		requireGRPCOK(t, err)
		assert.Empty(t, resp.Policies)
	})
}

func TestAuthGRPC_UpdateTwoFactorPolicy(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		updatedBy := uuid.New()

		resp, err := srv.UpdateTwoFactorPolicy(ctx, &authv1.UpdateTwoFactorPolicyRequest{
			RoleName:        "admin",
			Enforced:        true,
			GracePeriodDays: 14,
			UpdatedBy:       updatedBy.String(),
		})
		requireGRPCOK(t, err)
		assert.Equal(t, "admin", resp.Policy.RoleName)
		assert.True(t, resp.Policy.Enforced)
		assert.Equal(t, int32(14), resp.Policy.GracePeriodDays)
	})

	t.Run("missing role name", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		_, err := srv.UpdateTwoFactorPolicy(ctx, &authv1.UpdateTwoFactorPolicyRequest{
			RoleName:  "",
			UpdatedBy: uuid.New().String(),
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("invalid updated_by uuid", func(t *testing.T) {
		srv, _ := newTestAuthGRPCServer()
		_, err := srv.UpdateTwoFactorPolicy(ctx, &authv1.UpdateTwoFactorPolicyRequest{
			RoleName:  "admin",
			UpdatedBy: "bad",
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

// ============================================================================
// Converter Functions
// ============================================================================

func TestToUserInfo(t *testing.T) {
	user := &models.User{
		ID:               uuid.New(),
		Email:            "test@example.com",
		FirstName:        "Test",
		LastName:         "User",
		IsActive:         true,
		TwoFactorEnabled: true,
		Locale:           "de",
		CreatedAt:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	roles := []string{"admin", "member"}

	info := toUserInfo(user, roles)

	assert.Equal(t, user.ID.String(), info.Id)
	assert.Equal(t, "test@example.com", info.Email)
	assert.Equal(t, "Test", info.FirstName)
	assert.Equal(t, "User", info.LastName)
	assert.True(t, info.IsActive)
	assert.True(t, info.TwoFactorEnabled)
	assert.Equal(t, "de", info.Locale)
	assert.Equal(t, []string{"admin", "member"}, info.Roles)
	assert.Equal(t, "2025-01-01T00:00:00Z", info.CreatedAt)
}

func TestToSessionInfo(t *testing.T) {
	sessID := uuid.New()
	userID := uuid.New()
	now := time.Now()
	sess := &models.UserSession{
		ID:           sessID,
		UserID:       userID,
		DeviceName:   "Chrome",
		DeviceType:   "browser",
		IPAddress:    "192.168.1.1",
		Location:     "Berlin, DE",
		UserAgent:    "Mozilla/5.0",
		IsCurrent:    true,
		LastActiveAt: now,
		CreatedAt:    now,
	}

	info := toSessionInfo(sess)

	assert.Equal(t, sessID.String(), info.Id)
	assert.Equal(t, userID.String(), info.UserId)
	assert.Equal(t, "Chrome", info.DeviceName)
	assert.Equal(t, "browser", info.DeviceType)
	assert.Equal(t, "192.168.1.1", info.IpAddress)
	assert.Equal(t, "Berlin, DE", info.Location)
	assert.Equal(t, "Mozilla/5.0", info.UserAgent)
	assert.True(t, info.IsCurrent)
}

func TestToInvitationInfo(t *testing.T) {
	inv := &models.Invitation{
		ID:        uuid.New(),
		Email:     "invite@example.com",
		Role:      "member",
		ExpiresAt: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		CreatedBy: uuid.New(),
		CreatedAt: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
	}

	info := toInvitationInfo(inv)

	assert.Equal(t, inv.ID.String(), info.Id)
	assert.Equal(t, "invite@example.com", info.Email)
	assert.Equal(t, "member", info.Role)
	assert.Equal(t, inv.CreatedBy.String(), info.CreatedBy)
}

func TestToTwoFactorPolicyProto(t *testing.T) {
	t.Run("with updated_by", func(t *testing.T) {
		updatedBy := uuid.New()
		policy := &models.TwoFactorPolicy{
			ID:              uuid.New(),
			RoleName:        "admin",
			Enforced:        true,
			GracePeriodDays: 7,
			UpdatedBy:       &updatedBy,
			UpdatedAt:       time.Now(),
		}

		pb := toTwoFactorPolicyProto(policy)
		assert.Equal(t, "admin", pb.RoleName)
		assert.True(t, pb.Enforced)
		assert.Equal(t, int32(7), pb.GracePeriodDays)
		assert.Equal(t, updatedBy.String(), pb.UpdatedBy)
	})

	t.Run("without updated_by", func(t *testing.T) {
		policy := &models.TwoFactorPolicy{
			ID:        uuid.New(),
			RoleName:  "member",
			Enforced:  false,
			UpdatedAt: time.Now(),
		}

		pb := toTwoFactorPolicyProto(policy)
		assert.Equal(t, "member", pb.RoleName)
		assert.False(t, pb.Enforced)
		assert.Empty(t, pb.UpdatedBy)
	})
}
