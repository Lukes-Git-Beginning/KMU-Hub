package server

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/security/audit"
	"github.com/kmuhub/kmuhub/internal/sysctx"
	authv1 "github.com/kmuhub/kmuhub/proto/auth/v1"
)

type AuthGRPCServer struct {
	authv1.UnimplementedAuthServiceServer
	authService  *auth.Service
	auditService *audit.Service
}

func NewAuthGRPCServer(authService *auth.Service, auditService *audit.Service) *AuthGRPCServer {
	return &AuthGRPCServer{authService: authService, auditService: auditService}
}

// tenantAndCaller resolves the two identities every permission-change audit
// event needs: the tenant the change happened in, and the account that made
// it. Both come from the same JWT claims callerID already reads — this just
// adds the tenant half so callers that did not otherwise need it (UpdateRole,
// DeleteRole) can still be audited.
func tenantAndCaller(ctx context.Context) (uuid.UUID, uuid.UUID, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return uuid.Nil, uuid.Nil, status.Error(codes.Unauthenticated, "missing tenant context")
	}
	actorID, err := callerID(ctx)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return tenantID, actorID, nil
}

// logPermissionEvent appends one audit_log entry for a role/permission change.
// It runs after the write already succeeded — the four guardrails in
// internal/auth reject an unlawful attempt before it reaches here, so a
// rejected attempt never reaches this call and never writes an event.
func (s *AuthGRPCServer) logPermissionEvent(ctx context.Context, tenantID, actorID uuid.UUID, action, target, targetType string, details map[string]any) {
	s.auditService.LogEvent(ctx, tenantID, &actorID, action, target, targetType, details, "", "", "success")
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

// GetEffectivePermissions resolves the caller's (or, for the admin route, a
// target user's) fine-grained capabilities. An empty user_id means "me" and is
// resolved from the caller's own JWT via the tenant/user metadata forwarded by
// TenantInboundUnaryInterceptor — never a silent uuid.Nil, which would read as
// "user with no roles" instead of "no caller identified".
func (s *AuthGRPCServer) GetEffectivePermissions(ctx context.Context, req *authv1.GetEffectivePermissionsRequest) (*authv1.GetEffectivePermissionsResponse, error) {
	rawUserID := req.UserId
	if rawUserID == "" {
		rawUserID = middleware.GetUserID(ctx)
	}
	userID, err := uuid.Parse(rawUserID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	perms, err := s.authService.GetEffectivePermissions(ctx, userID)
	if err != nil {
		return nil, mapError(err)
	}

	roles := make([]*authv1.EffectiveRole, len(perms.Roles))
	for i, r := range perms.Roles {
		roles[i] = &authv1.EffectiveRole{
			Id:       r.ID.String(),
			Name:     r.Name,
			IsSystem: r.IsSystem,
			Color:    r.Color,
		}
	}

	capabilities := make([]*authv1.EffectiveCapability, len(perms.Capabilities))
	for i, c := range perms.Capabilities {
		capabilities[i] = &authv1.EffectiveCapability{
			Key:     c.Key,
			Scope:   c.Scope,
			Sources: c.Sources,
		}
	}

	return &authv1.GetEffectivePermissionsResponse{
		Roles:        roles,
		Capabilities: capabilities,
	}, nil
}

// callerID resolves the account behind the request from the x-user-id the
// TenantInboundUnaryInterceptor forwards over the gRPC hop. The role guardrails
// are stated in terms of the caller ("may not hand out what they lack", "may
// not lock themselves out"), so an unidentified caller cannot be evaluated —
// and uuid.Nil would read as "an account holding nothing", which turns every
// guardrail into a rejection instead of an error.
func callerID(ctx context.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(middleware.GetUserID(ctx))
	if err != nil {
		return uuid.Nil, status.Error(codes.Unauthenticated, "missing caller context")
	}
	return id, nil
}

// ListRoles returns the system presets plus the calling tenant's custom
// roles. TenantID/BasedOn nil renders as the proto zero value (empty string);
// the gateway is responsible for turning that back into JSON null.
func (s *AuthGRPCServer) ListRoles(ctx context.Context, req *authv1.ListRolesRequest) (*authv1.ListRolesResponse, error) {
	roles, err := s.authService.ListRoles(ctx)
	if err != nil {
		return nil, mapError(err)
	}

	out := make([]*authv1.Role, len(roles))
	for i := range roles {
		out[i] = toProtoRole(&roles[i])
	}

	return &authv1.ListRolesResponse{Roles: out}, nil
}

// CreateRole clones an existing role — preset or custom — into a new role
// owned by the calling tenant.
func (s *AuthGRPCServer) CreateRole(ctx context.Context, req *authv1.CreateRoleRequest) (*authv1.CreateRoleResponse, error) {
	basedOn, err := uuid.Parse(req.BasedOn)
	if err != nil {
		// A based_on that is not even a uuid names no role, so this is the
		// same answer as one that names a role nobody can see: 404.
		return nil, status.Error(codes.NotFound, auth.ErrBaseRoleNotFound.Error())
	}

	tenantID, actorID, err := tenantAndCaller(ctx)
	if err != nil {
		return nil, err
	}

	role, err := s.authService.CreateRole(ctx, actorID, tenantID, auth.CreateRoleInput{
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
		BasedOn:     basedOn,
	})
	if err != nil {
		return nil, mapError(err)
	}

	s.logPermissionEvent(ctx, tenantID, actorID, "permission.role_created", role.ID.String(), "role",
		map[string]any{"name": role.Name, "based_on": basedOn.String()})

	return &authv1.CreateRoleResponse{Role: toProtoRole(role)}, nil
}

// UpdateRole renames/re-describes/recolors a tenant-owned role. Tenant
// scoping needs no explicit parameter here — GetRoleByID and the UPDATE both
// run through the roles table's RLS policies on the request-scoped connection.
// Tenant and actor are still resolved explicitly, purely for the audit event.
func (s *AuthGRPCServer) UpdateRole(ctx context.Context, req *authv1.UpdateRoleRequest) (*authv1.UpdateRoleResponse, error) {
	roleID, err := uuid.Parse(req.RoleId)
	if err != nil {
		// Same reasoning as CreateRole's based_on: an id that isn't even a
		// uuid names no role, so it is the same 404 as one nobody can see.
		return nil, status.Error(codes.NotFound, auth.ErrBaseRoleNotFound.Error())
	}

	tenantID, actorID, err := tenantAndCaller(ctx)
	if err != nil {
		return nil, err
	}

	role, err := s.authService.UpdateRole(ctx, roleID, auth.UpdateRoleInput{
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
	})
	if err != nil {
		return nil, mapError(err)
	}

	s.logPermissionEvent(ctx, tenantID, actorID, "permission.role_updated", role.ID.String(), "role",
		map[string]any{"name": role.Name})

	return &authv1.UpdateRoleResponse{Role: toProtoRole(role)}, nil
}

// DeleteRole removes a tenant-owned role. Tenant and actor are resolved
// purely for the audit event — GetRoleByID and the DELETE are RLS-scoped on
// their own.
func (s *AuthGRPCServer) DeleteRole(ctx context.Context, req *authv1.DeleteRoleRequest) (*authv1.DeleteRoleResponse, error) {
	roleID, err := uuid.Parse(req.RoleId)
	if err != nil {
		return nil, status.Error(codes.NotFound, auth.ErrBaseRoleNotFound.Error())
	}

	tenantID, actorID, err := tenantAndCaller(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.authService.DeleteRole(ctx, roleID); err != nil {
		return nil, mapError(err)
	}

	s.logPermissionEvent(ctx, tenantID, actorID, "permission.role_deleted", roleID.String(), "role", nil)

	return &authv1.DeleteRoleResponse{}, nil
}

// GetRolePermissions returns the grant set of a role visible to the caller —
// presets included, since the builder reads a preset's grants when cloning it.
func (s *AuthGRPCServer) GetRolePermissions(ctx context.Context, req *authv1.GetRolePermissionsRequest) (*authv1.GetRolePermissionsResponse, error) {
	roleID, err := uuid.Parse(req.RoleId)
	if err != nil {
		return nil, status.Error(codes.NotFound, auth.ErrBaseRoleNotFound.Error())
	}

	grants, err := s.authService.GetRolePermissions(ctx, roleID)
	if err != nil {
		return nil, mapError(err)
	}

	return &authv1.GetRolePermissionsResponse{RoleId: req.RoleId, Grants: toProtoRoleGrants(grants)}, nil
}

// SetRolePermissions replaces the entire grant set of a tenant-owned role.
func (s *AuthGRPCServer) SetRolePermissions(ctx context.Context, req *authv1.SetRolePermissionsRequest) (*authv1.SetRolePermissionsResponse, error) {
	roleID, err := uuid.Parse(req.RoleId)
	if err != nil {
		return nil, status.Error(codes.NotFound, auth.ErrBaseRoleNotFound.Error())
	}

	actorID, err := callerID(ctx)
	if err != nil {
		return nil, err
	}

	grants := make([]auth.RoleGrant, len(req.Grants))
	for i, g := range req.Grants {
		grants[i] = auth.RoleGrant{Key: g.Key, Scope: g.Scope}
	}

	updated, err := s.authService.SetRolePermissions(ctx, actorID, roleID, grants)
	if err != nil {
		return nil, mapError(err)
	}

	return &authv1.SetRolePermissionsResponse{RoleId: req.RoleId, Grants: toProtoRoleGrants(updated)}, nil
}

// AssignUserRole grants a role — preset or custom — to an account and answers
// with the account's role ids afterwards. An unparsable id gets the same
// not_found a foreign or unknown one gets: the caller may not learn from the
// error which of the two it hit.
func (s *AuthGRPCServer) AssignUserRole(ctx context.Context, req *authv1.AssignUserRoleRequest) (*authv1.AssignUserRoleResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.NotFound, auth.ErrUserNotFound.Error())
	}
	roleID, err := uuid.Parse(req.RoleId)
	if err != nil {
		return nil, status.Error(codes.NotFound, auth.ErrBaseRoleNotFound.Error())
	}

	tenantID, actorID, err := tenantAndCaller(ctx)
	if err != nil {
		return nil, err
	}

	roleIDs, err := s.authService.AssignUserRole(ctx, actorID, userID, roleID)
	if err != nil {
		return nil, mapError(err)
	}

	s.logPermissionEvent(ctx, tenantID, actorID, "permission.assigned", userID.String(), "user",
		map[string]any{"role_id": roleID.String()})

	return &authv1.AssignUserRoleResponse{RoleIds: roleIDs}, nil
}

// RevokeUserRole takes a role off an account and answers with what is left.
func (s *AuthGRPCServer) RevokeUserRole(ctx context.Context, req *authv1.RevokeUserRoleRequest) (*authv1.RevokeUserRoleResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.NotFound, auth.ErrUserNotFound.Error())
	}
	roleID, err := uuid.Parse(req.RoleId)
	if err != nil {
		return nil, status.Error(codes.NotFound, auth.ErrBaseRoleNotFound.Error())
	}

	tenantID, actorID, err := tenantAndCaller(ctx)
	if err != nil {
		return nil, err
	}

	roleIDs, err := s.authService.RevokeUserRole(ctx, actorID, userID, roleID)
	if err != nil {
		return nil, mapError(err)
	}

	s.logPermissionEvent(ctx, tenantID, actorID, "permission.revoked", userID.String(), "user",
		map[string]any{"role_id": roleID.String()})

	return &authv1.RevokeUserRoleResponse{RoleIds: roleIDs}, nil
}

// toProtoRoleGrants renders a grant set for the wire, never nil — an empty
// slice still marshals through the gateway as [], not the JSON null a
// nil-slice repeated field would leave in the proto response.
func toProtoRoleGrants(grants []auth.RoleGrant) []*authv1.RoleGrant {
	out := make([]*authv1.RoleGrant, len(grants))
	for i, g := range grants {
		out[i] = &authv1.RoleGrant{Key: g.Key, Scope: g.Scope}
	}
	return out
}

// toProtoRole renders a role for the wire. TenantID/BasedOn nil becomes the
// proto zero value (empty string); the gateway turns that back into JSON null.
func toProtoRole(r *auth.Role) *authv1.Role {
	return &authv1.Role{
		Id:              r.ID.String(),
		Name:            r.Name,
		Description:     r.Description,
		TenantId:        uuidStringOrEmpty(r.TenantID),
		PresetId:        uuidStringOrEmpty(r.BasedOn),
		IsSystem:        r.IsSystem,
		Color:           r.Color,
		MemberCount:     int32(r.MemberCount),
		CapabilityCount: int32(r.CapabilityCount),
	}
}

// uuidStringOrEmpty renders a nullable uuid column as the proto3 zero value
// for a missing string — callers that need to distinguish "empty" from
// "absent" (the gateway, rendering tenant_id/preset_id as JSON null) do so on
// the way out, not here.
func uuidStringOrEmpty(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
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

// ProvisionTenant creates a tenant. Unlike every other method here it takes no
// tenant from ctx: the tenant it writes does not exist yet, and the caller's
// own tenant is irrelevant to it. Authorisation is the tenants:write permission
// the gateway checks, not RLS.
func (s *AuthGRPCServer) ProvisionTenant(ctx context.Context, req *authv1.ProvisionTenantRequest) (*authv1.ProvisionTenantResponse, error) {
	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by user id")
	}

	in := auth.ProvisionTenantInput{
		Name:        req.Name,
		PlanType:    req.PlanType,
		SupportTier: req.SupportTier,
		Modules:     req.Modules,
		AdminEmail:  req.AdminEmail,
		CreatedBy:   createdBy,
	}

	if req.HasSeatLimit {
		limit := int(req.SeatLimit)
		in.SeatLimit = &limit
	}

	if req.BillingPeriodEnd != "" {
		// Date only: a billing period ends on a day, and the column is DATE.
		end, err := time.Parse("2006-01-02", req.BillingPeriodEnd)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "billing_period_end must be YYYY-MM-DD")
		}
		in.BillingPeriodEnd = &end
	}

	res, err := s.authService.ProvisionTenant(ctx, in)
	if err != nil {
		return nil, mapError(err)
	}

	return &authv1.ProvisionTenantResponse{
		Tenant:          toTenantInfo(res.Tenant, res.Modules),
		Invitation:      toInvitationInfo(res.Invitation),
		InvitationToken: res.Token,
	}, nil
}

func (s *AuthGRPCServer) CreateInvitation(ctx context.Context, req *authv1.CreateInvitationRequest) (*authv1.CreateInvitationResponse, error) {
	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by user id")
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant context")
	}

	inv, token, err := s.authService.CreateInvitation(ctx, tenantID, req.Email, req.Role, createdBy)
	if err != nil {
		return nil, mapError(err)
	}

	return &authv1.CreateInvitationResponse{
		Invitation: toInvitationInfo(inv),
		Token:      token,
	}, nil
}

func (s *AuthGRPCServer) ListInvitations(ctx context.Context, req *authv1.ListInvitationsRequest) (*authv1.ListInvitationsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant context")
	}

	invs, err := s.authService.ListInvitations(ctx, tenantID)
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

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant context")
	}

	if err := s.authService.CancelInvitation(ctx, tenantID, invID); err != nil {
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

	// The session must belong to the caller — the gateway fills UserId from
	// the JWT, the session id comes from the URL.
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	if err := s.authService.TerminateSession(ctx, sessionID, userID); err != nil {
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

// GetTwoFactorPolicy returns the calling tenant's 2FA policies. Since migration
// 000273 the table is tenant-scoped, so the tenant has to be resolved here —
// the auth package cannot read it from the context itself.
func (s *AuthGRPCServer) GetTwoFactorPolicy(ctx context.Context, req *authv1.GetTwoFactorPolicyRequest) (*authv1.GetTwoFactorPolicyResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant context")
	}

	if req.RoleName != "" {
		policy, err := s.authService.GetTwoFactorPolicy(ctx, tenantID, req.RoleName)
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
	dbPolicies, err := s.authService.ListTwoFactorPolicies(ctx, tenantID)
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

	// The route guard is RequireRole("admin"), which answers "is an admin" but
	// not "an admin of which tenant" — before migration 000273 that let any
	// tenant's admin rewrite everyone's policy. The tenant from the propagated
	// claims is what scopes the write now.
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant context")
	}

	policy, err := s.authService.UpdateTwoFactorPolicy(ctx, tenantID, req.RoleName, req.Enforced, int(req.GracePeriodDays), updatedBy)
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

// toTenantInfo renders a provisioned tenant. moduleIDs is the resolved
// activation list, which the tenant row itself does not carry.
func toTenantInfo(t *models.Tenant, moduleIDs []string) *authv1.TenantInfo {
	info := &authv1.TenantInfo{
		Id:                 t.ID.String(),
		Name:               t.Name,
		PlanType:           t.PlanType,
		SupportTier:        t.SupportTier,
		SubscriptionStatus: t.SubscriptionStatus,
		CreatedAt:          t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Modules:            moduleIDs,
	}
	if t.BillingPeriodEnd != nil {
		info.BillingPeriodEnd = t.BillingPeriodEnd.Format("2006-01-02")
	}
	if t.SeatLimit != nil {
		info.SeatLimit = int32(*t.SeatLimit)
		info.HasSeatLimit = true
	}
	return info
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

// timeStringOrEmpty renders a nullable timestamp as the proto3 zero value for
// a missing string — the gateway turns "" back into JSON null, same
// convention as uuidStringOrEmpty.
func timeStringOrEmpty(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

func toAdminUserInfo(u *auth.AdminUser) *authv1.AdminUserInfo {
	return &authv1.AdminUserInfo{
		Id:          u.ID.String(),
		FirstName:   u.FirstName,
		LastName:    u.LastName,
		Email:       u.Email,
		RoleIds:     u.RoleIDs,
		Status:      string(u.Status),
		LastLoginAt: timeStringOrEmpty(u.LastLoginAt),
		InvitedAt:   timeStringOrEmpty(u.InvitedAt),
	}
}

// ListAdminUsers returns the calling tenant's account roster — real accounts
// and still-open invitations merged into one list (admin-types.ts, AdminUser).
func (s *AuthGRPCServer) ListAdminUsers(ctx context.Context, _ *authv1.ListAdminUsersRequest) (*authv1.ListAdminUsersResponse, error) {
	users, err := s.authService.ListAdminUsers(ctx)
	if err != nil {
		return nil, mapError(err)
	}

	out := make([]*authv1.AdminUserInfo, len(users))
	for i := range users {
		out[i] = toAdminUserInfo(&users[i])
	}

	return &authv1.ListAdminUsersResponse{Users: out}, nil
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
	// Provisioning input errors. All InvalidArgument — the gateway turns that
	// into 400, which is what a malformed provisioning request is.
	case errors.Is(err, auth.ErrTenantNameRequired),
		errors.Is(err, auth.ErrTenantNameTooLong),
		errors.Is(err, auth.ErrInvalidPlanType),
		errors.Is(err, auth.ErrInvalidSupportTier),
		errors.Is(err, auth.ErrInvalidSeatLimit),
		errors.Is(err, auth.ErrAdminEmailRequired),
		errors.Is(err, auth.ErrProvisionerRequired),
		errors.Is(err, auth.ErrUnknownModule):
		return status.Error(codes.InvalidArgument, err.Error())
	// FailedPrecondition, not ResourceExhausted: the gateway maps the latter to
	// 429, which reads as "retry in a moment" — a full seat plan is not.
	case errors.Is(err, auth.ErrSeatLimitReached):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, auth.ErrRoleNotFound):
		return status.Error(codes.FailedPrecondition, err.Error())
	// Role administration (RBAC wave 1b). The messages are the frontend's
	// error codes, so the code choice only has to land on the right HTTP
	// status: AlreadyExists and FailedPrecondition both become 409, NotFound
	// becomes 404.
	case errors.Is(err, auth.ErrRoleNameExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, auth.ErrRoleLimitReached):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, auth.ErrBaseRoleNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, auth.ErrRolePresetImmutable):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, auth.ErrRoleHasMembers):
		return status.Error(codes.FailedPrecondition, err.Error())
	// Guardrails (wave 1b). last_admin and self_lockout are conflicts with the
	// tenant's current state, so 409 like the other two above; an escalation
	// attempt is a permission problem and gets the same 403 preset_immutable
	// gets.
	case errors.Is(err, auth.ErrLastAdmin):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, auth.ErrSelfLockout):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, auth.ErrPrivilegeEscalation):
		return status.Error(codes.PermissionDenied, err.Error())
	// OutOfRange, not InvalidArgument: the gateway maps it to 422, matching
	// the frontend contract's "unbekannter Key -> 422" — InvalidArgument
	// would land on 400, which the builder does not distinguish from a
	// malformed request body.
	case errors.Is(err, auth.ErrCapabilityKeyUnknown):
		return status.Error(codes.OutOfRange, err.Error())
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
