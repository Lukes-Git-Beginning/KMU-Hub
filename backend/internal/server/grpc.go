package server

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/sysctx"
	authv1 "github.com/kmuhub/kmuhub/proto/auth/v1"
)

type AuthGRPCServer struct {
	authv1.UnimplementedAuthServiceServer
	authService *auth.Service
}

func NewAuthGRPCServer(authService *auth.Service) *AuthGRPCServer {
	return &AuthGRPCServer{authService: authService}
}

func (s *AuthGRPCServer) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	user, tokens, err := s.authService.Register(ctx, req.Email, req.Password, req.FirstName, req.LastName)
	if err != nil {
		return nil, mapError(err)
	}

	roles, _ := s.authService.CheckPermission(ctx, user.ID, "", "")
	_ = roles

	userRoles := []string{"member"}

	return &authv1.RegisterResponse{
		User:         toUserInfo(user, userRoles),
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (s *AuthGRPCServer) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	result, err := s.authService.Login(ctx, req.Email, req.Password)
	if err != nil {
		return nil, mapError(err)
	}

	if result.RequiresTwoFactor {
		return &authv1.LoginResponse{
			RequiresTwoFactor: true,
			PendingToken:      result.PendingToken,
		}, nil
	}

	var roles []string
	if result.User != nil {
		// Login is a pre-JWT path: the request ctx has no tenant set, so
		// GetUser would hit RLS with zero GUCs and return 0 rows. Wrap with
		// system context for the post-login user fetch (matches the wrap
		// inside Service.Login itself).
		lookupCtx := sysctx.With(ctx)
		_, roles, err = s.authService.GetUser(lookupCtx, result.User.ID)
		if err != nil {
			return nil, mapError(err)
		}
	}

	return &authv1.LoginResponse{
		User:         toUserInfo(result.User, roles),
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	}, nil
}

func (s *AuthGRPCServer) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	tokens, err := s.authService.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, mapError(err)
	}

	return &authv1.RefreshTokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (s *AuthGRPCServer) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	if err := s.authService.Logout(ctx, req.RefreshToken); err != nil {
		return nil, mapError(err)
	}
	return &authv1.LogoutResponse{}, nil
}

func (s *AuthGRPCServer) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	claims, err := s.authService.ValidateToken(ctx, req.AccessToken)
	if err != nil {
		return &authv1.ValidateTokenResponse{Valid: false}, nil
	}

	return &authv1.ValidateTokenResponse{
		Valid:       true,
		UserId:      claims.UserID,
		Roles:       claims.Roles,
		Permissions: claims.Permissions,
	}, nil
}

func (s *AuthGRPCServer) GetUser(ctx context.Context, req *authv1.GetUserRequest) (*authv1.GetUserResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	user, roles, err := s.authService.GetUser(ctx, userID)
	if err != nil {
		return nil, mapError(err)
	}

	return &authv1.GetUserResponse{
		User: toUserInfo(user, roles),
	}, nil
}

func (s *AuthGRPCServer) ListUsers(ctx context.Context, req *authv1.ListUsersRequest) (*authv1.ListUsersResponse, error) {
	users, total, err := s.authService.ListUsers(ctx, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list users")
	}

	var infos []*authv1.UserInfo
	for _, u := range users {
		roles, _ := s.authService.CheckPermission(ctx, u.ID, "", "")
		_ = roles
		infos = append(infos, toUserInfo(u, nil))
	}

	return &authv1.ListUsersResponse{
		Users: infos,
		Total: int32(total),
	}, nil
}

func (s *AuthGRPCServer) UpdateUser(ctx context.Context, req *authv1.UpdateUserRequest) (*authv1.UpdateUserResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	var firstName, lastName *string
	var isActive *bool
	if req.FirstName != nil {
		v := *req.FirstName
		firstName = &v
	}
	if req.LastName != nil {
		v := *req.LastName
		lastName = &v
	}
	if req.IsActive != nil {
		v := *req.IsActive
		isActive = &v
	}

	user, err := s.authService.UpdateUser(ctx, userID, firstName, lastName, isActive)
	if err != nil {
		return nil, mapError(err)
	}

	return &authv1.UpdateUserResponse{
		User: toUserInfo(user, nil),
	}, nil
}

func (s *AuthGRPCServer) AssignRole(ctx context.Context, req *authv1.AssignRoleRequest) (*authv1.AssignRoleResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	if err := s.authService.AssignRole(ctx, userID, req.RoleName); err != nil {
		return nil, mapError(err)
	}

	return &authv1.AssignRoleResponse{}, nil
}

func (s *AuthGRPCServer) RemoveRole(ctx context.Context, req *authv1.RemoveRoleRequest) (*authv1.RemoveRoleResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	if err := s.authService.RemoveRole(ctx, userID, req.RoleName); err != nil {
		return nil, mapError(err)
	}

	return &authv1.RemoveRoleResponse{}, nil
}

func (s *AuthGRPCServer) CheckPermission(ctx context.Context, req *authv1.CheckPermissionRequest) (*authv1.CheckPermissionResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	allowed, err := s.authService.CheckPermission(ctx, userID, req.Resource, req.Action)
	if err != nil {
		return nil, status.Error(codes.Internal, "permission check failed")
	}

	return &authv1.CheckPermissionResponse{Allowed: allowed}, nil
}

func (s *AuthGRPCServer) GetProfile(ctx context.Context, req *authv1.GetProfileRequest) (*authv1.GetProfileResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	user, roles, err := s.authService.GetProfile(ctx, userID)
	if err != nil {
		return nil, mapError(err)
	}

	return &authv1.GetProfileResponse{
		User: toUserInfo(user, roles),
	}, nil
}

func (s *AuthGRPCServer) UpdateProfile(ctx context.Context, req *authv1.UpdateProfileRequest) (*authv1.UpdateProfileResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	var firstName, lastName, avatarURL *string
	if req.FirstName != nil {
		v := *req.FirstName
		firstName = &v
	}
	if req.LastName != nil {
		v := *req.LastName
		lastName = &v
	}
	if req.AvatarUrl != nil {
		v := *req.AvatarUrl
		avatarURL = &v
	}

	user, err := s.authService.UpdateProfile(ctx, userID, firstName, lastName, avatarURL)
	if err != nil {
		return nil, mapError(err)
	}

	return &authv1.UpdateProfileResponse{
		User: toUserInfo(user, nil),
	}, nil
}

func (s *AuthGRPCServer) ChangePassword(ctx context.Context, req *authv1.ChangePasswordRequest) (*authv1.ChangePasswordResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	if err := s.authService.ChangePassword(ctx, userID, req.OldPassword, req.NewPassword); err != nil {
		return nil, mapError(err)
	}

	return &authv1.ChangePasswordResponse{}, nil
}

func (s *AuthGRPCServer) CreateInvitation(ctx context.Context, req *authv1.CreateInvitationRequest) (*authv1.CreateInvitationResponse, error) {
	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by user id")
	}

	inv, token, err := s.authService.CreateInvitation(ctx, req.Email, req.Role, createdBy)
	if err != nil {
		return nil, mapError(err)
	}

	return &authv1.CreateInvitationResponse{
		Invitation: toInvitationInfo(inv),
		Token:      token,
	}, nil
}

func (s *AuthGRPCServer) ListInvitations(ctx context.Context, req *authv1.ListInvitationsRequest) (*authv1.ListInvitationsResponse, error) {
	invs, err := s.authService.ListInvitations(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list invitations")
	}

	var infos []*authv1.InvitationInfo
	for _, inv := range invs {
		infos = append(infos, toInvitationInfo(inv))
	}

	return &authv1.ListInvitationsResponse{Invitations: infos}, nil
}

func (s *AuthGRPCServer) AcceptInvitation(ctx context.Context, req *authv1.AcceptInvitationRequest) (*authv1.AcceptInvitationResponse, error) {
	user, tokens, err := s.authService.AcceptInvitation(ctx, req.Token, req.Password, req.FirstName, req.LastName)
	if err != nil {
		return nil, mapError(err)
	}

	lookupCtx := sysctx.With(ctx)
	_, roles, _ := s.authService.GetUser(lookupCtx, user.ID)

	return &authv1.AcceptInvitationResponse{
		User:         toUserInfo(user, roles),
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (s *AuthGRPCServer) CancelInvitation(ctx context.Context, req *authv1.CancelInvitationRequest) (*authv1.CancelInvitationResponse, error) {
	invID, err := uuid.Parse(req.InvitationId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid invitation id")
	}

	if err := s.authService.CancelInvitation(ctx, invID); err != nil {
		return nil, mapError(err)
	}

	return &authv1.CancelInvitationResponse{}, nil
}

// ============================================================================
// Two-Factor Authentication Handlers
// ============================================================================

func (s *AuthGRPCServer) Setup2FA(ctx context.Context, req *authv1.Setup2FARequest) (*authv1.Setup2FAResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	result, err := s.authService.Setup2FA(ctx, userID)
	if err != nil {
		return nil, mapError(err)
	}

	return &authv1.Setup2FAResponse{
		QrCodePng:    result.QRCodePNG,
		ManualSecret: result.Secret,
	}, nil
}

func (s *AuthGRPCServer) Verify2FA(ctx context.Context, req *authv1.Verify2FARequest) (*authv1.Verify2FAResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	codes, err := s.authService.Verify2FA(ctx, userID, req.Code)
	if err != nil {
		return nil, mapError(err)
	}

	return &authv1.Verify2FAResponse{
		RecoveryCodes: codes,
	}, nil
}

func (s *AuthGRPCServer) Validate2FALogin(ctx context.Context, req *authv1.Validate2FALoginRequest) (*authv1.Validate2FALoginResponse, error) {
	user, tokens, err := s.authService.CompleteTwoFactorLogin(ctx, req.PendingToken, req.Code, req.IsRecoveryCode)
	if err != nil {
		return nil, mapError(err)
	}

	var roles []string
	if user != nil {
		// Pre-JWT path (same as Login): wrap with system context for the
		// post-2FA user fetch so the RLS-enabled users table is readable.
		lookupCtx := sysctx.With(ctx)
		_, roles, err = s.authService.GetUser(lookupCtx, user.ID)
		if err != nil {
			return nil, mapError(err)
		}
	}

	return &authv1.Validate2FALoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         toUserInfo(user, roles),
	}, nil
}

func (s *AuthGRPCServer) Disable2FA(ctx context.Context, req *authv1.Disable2FARequest) (*authv1.Disable2FAResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	if err := s.authService.Disable2FA(ctx, userID, req.Code); err != nil {
		return nil, mapError(err)
	}

	return &authv1.Disable2FAResponse{}, nil
}

func (s *AuthGRPCServer) RegenerateRecoveryCodes(ctx context.Context, req *authv1.RegenerateRecoveryCodesRequest) (*authv1.RegenerateRecoveryCodesResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	codes, err := s.authService.RegenerateRecoveryCodes(ctx, userID, req.Code)
	if err != nil {
		return nil, mapError(err)
	}

	return &authv1.RegenerateRecoveryCodesResponse{
		RecoveryCodes: codes,
	}, nil
}

func (s *AuthGRPCServer) AdminReset2FA(ctx context.Context, req *authv1.AdminReset2FARequest) (*authv1.AdminReset2FAResponse, error) {
	targetUserID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	adminID, err := uuid.Parse(req.AdminId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid admin id")
	}

	if err := s.authService.AdminReset2FA(ctx, targetUserID, adminID, req.Reason); err != nil {
		return nil, mapError(err)
	}

	return &authv1.AdminReset2FAResponse{}, nil
}

// ============================================================================
// Session Handlers
// ============================================================================

func (s *AuthGRPCServer) ListSessions(ctx context.Context, req *authv1.ListSessionsRequest) (*authv1.ListSessionsResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	sessions, err := s.authService.ListSessions(ctx, userID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list sessions")
	}

	var pbSessions []*authv1.SessionInfo
	for _, sess := range sessions {
		pbSessions = append(pbSessions, toSessionInfo(sess))
	}

	return &authv1.ListSessionsResponse{
		Sessions: pbSessions,
	}, nil
}

func (s *AuthGRPCServer) TerminateSession(ctx context.Context, req *authv1.TerminateSessionRequest) (*authv1.TerminateSessionResponse, error) {
	sessionID, err := uuid.Parse(req.SessionId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid session id")
	}

	// UserId is available for authorization checks but TerminateSession only needs sessionID
	if _, err := uuid.Parse(req.UserId); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	if err := s.authService.TerminateSession(ctx, sessionID); err != nil {
		return nil, mapError(err)
	}

	return &authv1.TerminateSessionResponse{}, nil
}

func (s *AuthGRPCServer) TerminateAllSessions(ctx context.Context, req *authv1.TerminateAllSessionsRequest) (*authv1.TerminateAllSessionsResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	var currentSessionID *uuid.UUID
	if req.CurrentSessionId != "" {
		parsed, parseErr := uuid.Parse(req.CurrentSessionId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid current session id")
		}
		currentSessionID = &parsed
	}

	if err := s.authService.TerminateAllSessions(ctx, userID, currentSessionID); err != nil {
		return nil, mapError(err)
	}

	return &authv1.TerminateAllSessionsResponse{}, nil
}

// ============================================================================
// Two-Factor Policy Handlers
// ============================================================================

func (s *AuthGRPCServer) GetTwoFactorPolicy(ctx context.Context, req *authv1.GetTwoFactorPolicyRequest) (*authv1.GetTwoFactorPolicyResponse, error) {
	if req.RoleName != "" {
		policy, err := s.authService.GetTwoFactorPolicy(ctx, req.RoleName)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to get 2FA policy")
		}
		var policies []*authv1.TwoFactorPolicy
		if policy != nil {
			policies = append(policies, toTwoFactorPolicyProto(policy))
		}
		return &authv1.GetTwoFactorPolicyResponse{Policies: policies}, nil
	}

	// Return all policies
	dbPolicies, err := s.authService.ListTwoFactorPolicies(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list 2FA policies")
	}

	var policies []*authv1.TwoFactorPolicy
	for _, p := range dbPolicies {
		policies = append(policies, toTwoFactorPolicyProto(p))
	}

	return &authv1.GetTwoFactorPolicyResponse{Policies: policies}, nil
}

func (s *AuthGRPCServer) UpdateTwoFactorPolicy(ctx context.Context, req *authv1.UpdateTwoFactorPolicyRequest) (*authv1.UpdateTwoFactorPolicyResponse, error) {
	if req.RoleName == "" {
		return nil, status.Error(codes.InvalidArgument, "role_name is required")
	}

	updatedBy, err := uuid.Parse(req.UpdatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid updated_by")
	}

	policy, err := s.authService.UpdateTwoFactorPolicy(ctx, req.RoleName, req.Enforced, int(req.GracePeriodDays), updatedBy)
	if err != nil {
		return nil, mapError(err)
	}

	return &authv1.UpdateTwoFactorPolicyResponse{
		Policy: toTwoFactorPolicyProto(policy),
	}, nil
}

func toSessionInfo(sess *models.UserSession) *authv1.SessionInfo {
	return &authv1.SessionInfo{
		Id:           sess.ID.String(),
		UserId:       sess.UserID.String(),
		DeviceName:   sess.DeviceName,
		DeviceType:   sess.DeviceType,
		IpAddress:    sess.IPAddress,
		Location:     sess.Location,
		UserAgent:    sess.UserAgent,
		IsCurrent:    sess.IsCurrent,
		LastActiveAt: timestamppb.New(sess.LastActiveAt),
		CreatedAt:    timestamppb.New(sess.CreatedAt),
	}
}

func toTwoFactorPolicyProto(p *models.TwoFactorPolicy) *authv1.TwoFactorPolicy {
	pb := &authv1.TwoFactorPolicy{
		Id:              p.ID.String(),
		RoleName:        p.RoleName,
		Enforced:        p.Enforced,
		GracePeriodDays: int32(p.GracePeriodDays),
		UpdatedAt:       timestamppb.New(p.UpdatedAt),
	}
	if p.UpdatedBy != nil {
		pb.UpdatedBy = p.UpdatedBy.String()
	}
	return pb
}

func toInvitationInfo(inv *models.Invitation) *authv1.InvitationInfo {
	return &authv1.InvitationInfo{
		Id:        inv.ID.String(),
		Email:     inv.Email,
		Role:      inv.Role,
		ExpiresAt: inv.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		CreatedBy: inv.CreatedBy.String(),
		CreatedAt: inv.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toUserInfo(user *models.User, roles []string) *authv1.UserInfo {
	return &authv1.UserInfo{
		Id:               user.ID.String(),
		Email:            user.Email,
		FirstName:        user.FirstName,
		LastName:         user.LastName,
		IsActive:         user.IsActive,
		Roles:            roles,
		CreatedAt:        user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		TwoFactorEnabled: user.TwoFactorEnabled,
		Locale:           user.Locale,
		AvatarUrl:        user.AvatarURL,
	}
}

// ============================================================================
// Password Reset Handlers
// ============================================================================

func (s *AuthGRPCServer) RequestPasswordReset(ctx context.Context, req *authv1.RequestPasswordResetRequest) (*authv1.RequestPasswordResetResponse, error) {
	// Intentionally ignores the error — always returns the same response
	// to prevent user enumeration.
	_ = s.authService.RequestPasswordReset(ctx, req.Email)
	return &authv1.RequestPasswordResetResponse{
		Message: "If an account with that email exists, a reset link has been sent.",
	}, nil
}

func (s *AuthGRPCServer) ResetPassword(ctx context.Context, req *authv1.ResetPasswordRequest) (*authv1.ResetPasswordResponse, error) {
	if err := s.authService.ResetPassword(ctx, req.Token, req.NewPassword); err != nil {
		return nil, mapError(err)
	}
	return &authv1.ResetPasswordResponse{}, nil
}

func mapError(err error) error {
	switch {
	case errors.Is(err, auth.ErrUserExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, auth.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, auth.ErrUserNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, auth.ErrTokenExpired), errors.Is(err, auth.ErrTokenRevoked), errors.Is(err, auth.ErrTokenInvalid):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, auth.ErrUserInactive):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, auth.ErrInvitationNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, auth.ErrInvitationExpired):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, auth.ErrInvitationAlreadyUsed):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, auth.ErrInvitationExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, auth.ErrTwoFactorAlreadyEnabled):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, auth.ErrTwoFactorNotEnabled):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, auth.ErrNo2FASetupPending):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, auth.ErrInvalidTOTPCode), errors.Is(err, auth.ErrInvalidRecoveryCode):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, auth.ErrAllRecoveryCodesUsed):
		return status.Error(codes.ResourceExhausted, err.Error())
	case errors.Is(err, auth.ErrInvalidPendingToken), errors.Is(err, auth.ErrPendingTokenExpired):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, auth.Err2FAEnforcementRequired):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, auth.ErrSessionNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, auth.ErrResetTokenInvalid):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, auth.ErrResetTokenExpired):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, auth.ErrResetTokenUsed):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, auth.ErrPasswordTooWeak):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
