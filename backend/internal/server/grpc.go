package server

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/models"
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
	user, tokens, err := s.authService.Login(ctx, req.Email, req.Password)
	if err != nil {
		return nil, mapError(err)
	}

	_, roles, _ := s.authService.GetUser(ctx, user.ID)

	return &authv1.LoginResponse{
		User:         toUserInfo(user, roles),
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
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

func toUserInfo(user *models.User, roles []string) *authv1.UserInfo {
	return &authv1.UserInfo{
		Id:        user.ID.String(),
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		IsActive:  user.IsActive,
		Roles:     roles,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
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
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
