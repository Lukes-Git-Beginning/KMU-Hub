package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	authv1 "github.com/kmuhub/kmuhub/proto/auth/v1"
)

// TestMapError_RemainingSentinels covers the branches TestMapError does not:
// tenant provisioning, roster guardrails, password reset. Together the two
// tests hit every case of mapError, including the Internal default.
func TestMapError_RemainingSentinels(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"tenant name required", auth.ErrTenantNameRequired, codes.InvalidArgument},
		{"tenant name too long", auth.ErrTenantNameTooLong, codes.InvalidArgument},
		{"invalid plan type", auth.ErrInvalidPlanType, codes.InvalidArgument},
		{"invalid support tier", auth.ErrInvalidSupportTier, codes.InvalidArgument},
		{"invalid seat limit", auth.ErrInvalidSeatLimit, codes.InvalidArgument},
		{"admin email required", auth.ErrAdminEmailRequired, codes.InvalidArgument},
		{"provisioner required", auth.ErrProvisionerRequired, codes.InvalidArgument},
		{"unknown module", auth.ErrUnknownModule, codes.InvalidArgument},
		// FailedPrecondition (409), never ResourceExhausted (429, "retry").
		{"seat limit reached", auth.ErrSeatLimitReached, codes.FailedPrecondition},
		{"role not found", auth.ErrRoleNotFound, codes.FailedPrecondition},
		{"self deactivation", auth.ErrSelfDeactivation, codes.FailedPrecondition},
		{"status not assignable", auth.ErrStatusNotAssignable, codes.InvalidArgument},
		// OutOfRange is what the gateway turns into 422.
		{"capability key unknown", auth.ErrCapabilityKeyUnknown, codes.OutOfRange},
		{"reset token invalid", auth.ErrResetTokenInvalid, codes.NotFound},
		{"reset token expired", auth.ErrResetTokenExpired, codes.FailedPrecondition},
		{"reset token used", auth.ErrResetTokenUsed, codes.FailedPrecondition},
		{"password too weak", auth.ErrPasswordTooWeak, codes.InvalidArgument},
		// auth.CountSeatsInUse returns ErrTenantNotFound for a vanished tenant
		// row; mapError has no case for it, so it surfaces as Internal (500).
		// Pinned so a future mapping is a conscious change.
		{"tenant not found falls through", auth.ErrTenantNotFound, codes.Internal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireGRPCCode(t, mapError(tc.err), tc.code)
		})
	}
}

func TestMapError_WrappedSentinelKeepsCode(t *testing.T) {
	wrapped := fmt.Errorf("create role: %w", auth.ErrRoleNameExists)
	requireGRPCCode(t, mapError(wrapped), codes.AlreadyExists)
}

func TestMapError_DefaultHidesInternals(t *testing.T) {
	err := mapError(fmt.Errorf("pq: connection refused to 10.0.0.5: %w", context.DeadlineExceeded))
	requireGRPCCode(t, err, codes.Internal)
	require.NotContains(t, err.Error(), "10.0.0.5")
}

func TestToTenantInfo(t *testing.T) {
	created := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	end := time.Date(2027, 1, 31, 0, 0, 0, 0, time.UTC)
	seats := 12

	full := toTenantInfo(&models.Tenant{
		ID: uuid.New(), Name: "Acme", PlanType: "cosmi", SupportTier: "standard",
		SubscriptionStatus: "active", CreatedAt: created, BillingPeriodEnd: &end, SeatLimit: &seats,
	}, []string{"crm", "inbox"})
	require.Equal(t, "2026-03-04T05:06:07Z", full.CreatedAt)
	require.Equal(t, "2027-01-31", full.BillingPeriodEnd)
	require.True(t, full.HasSeatLimit)
	require.Equal(t, int32(12), full.SeatLimit)
	require.Equal(t, []string{"crm", "inbox"}, full.Modules)

	bare := toTenantInfo(&models.Tenant{ID: uuid.New(), CreatedAt: created}, nil)
	require.Empty(t, bare.BillingPeriodEnd)
	require.False(t, bare.HasSeatLimit)
}

func TestSmallRenderers(t *testing.T) {
	require.Empty(t, timeStringOrEmpty(nil))
	ts := time.Date(2026, 5, 6, 7, 8, 9, 0, time.UTC)
	require.Equal(t, "2026-05-06T07:08:09Z", timeStringOrEmpty(&ts))

	require.Empty(t, uuidStringOrEmpty(nil))
	id := uuid.New()
	require.Equal(t, id.String(), uuidStringOrEmpty(&id))

	// Never nil: an empty grant set must marshal as [] through the gateway.
	require.NotNil(t, toProtoRoleGrants(nil))
	require.Empty(t, toProtoRoleGrants(nil))
	g := toProtoRoleGrants([]auth.RoleGrant{{Key: "crm:read", Scope: "own"}})
	require.Equal(t, "crm:read", g[0].Key)
	require.Equal(t, "own", g[0].Scope)

	uid := uuid.New()
	info := toAdminUserInfo(&auth.AdminUser{
		ID: uid, FirstName: "A", LastName: "B", Email: "a@b.de",
		RoleIDs: []string{"r1"}, Status: "invited", InvitedAt: &ts,
	})
	require.Equal(t, uid.String(), info.Id)
	require.Equal(t, "invited", info.Status)
	require.Empty(t, info.LastLoginAt)
	require.Equal(t, "2026-05-06T07:08:09Z", info.InvitedAt)
}

func TestParseRoleIDs(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	ids, err := parseRoleIDs([]string{a.String(), b.String()})
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{a, b}, ids)

	// One malformed id refuses the whole set; nothing is silently dropped.
	_, err = parseRoleIDs([]string{a.String(), "nope"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestAuthGRPC_RoleAndRosterReads(t *testing.T) {
	srv, _ := newTestAuthGRPCServer()
	ctx := tenantCtx(uuid.New())

	roles, err := srv.ListRoles(ctx, &authv1.ListRolesRequest{})
	require.NoError(t, err)
	require.NotNil(t, roles.Roles)
	require.Empty(t, roles.Roles)

	users, err := srv.ListAdminUsers(ctx, &authv1.ListAdminUsersRequest{})
	require.NoError(t, err)
	require.NotNil(t, users.Users)
	require.Empty(t, users.Users)

	grants, err := srv.GetRolePermissions(ctx, &authv1.GetRolePermissionsRequest{RoleId: uuid.NewString()})
	require.NoError(t, err)
	require.NotNil(t, grants.Grants)
}

func TestAuthGRPC_MalformedIDsAreRefusedBeforeTheService(t *testing.T) {
	srv, _ := newTestAuthGRPCServer()
	ctx := tenantCtx(uuid.New())
	actor := context.WithValue(ctx, middleware.UserIDKey, uuid.NewString())

	// An id that is not a uuid names nothing visible: 404, same as an unknown id.
	_, err := srv.GetRolePermissions(ctx, &authv1.GetRolePermissionsRequest{RoleId: "nope"})
	requireGRPCCode(t, err, codes.NotFound)
	_, err = srv.SetRolePermissions(ctx, &authv1.SetRolePermissionsRequest{RoleId: "nope"})
	requireGRPCCode(t, err, codes.NotFound)
	_, err = srv.GetUserOverrides(ctx, &authv1.GetUserOverridesRequest{UserId: "nope"})
	requireGRPCCode(t, err, codes.NotFound)

	// SetRolePermissions needs an authenticated caller for the audit trail.
	_, err = srv.SetRolePermissions(ctx, &authv1.SetRolePermissionsRequest{RoleId: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)

	_, err = srv.UpdateProfile(ctx, &authv1.UpdateProfileRequest{UserId: "nope"})
	requireGRPCCode(t, err, codes.InvalidArgument)

	// Roster: tenant and actor are both required, role ids must all parse.
	_, err = srv.InviteAdminUser(context.Background(), &authv1.InviteAdminUserRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
	_, err = srv.InviteAdminUser(ctx, &authv1.InviteAdminUserRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
	_, err = srv.InviteAdminUser(actor, &authv1.InviteAdminUserRequest{RoleIds: []string{"nope"}})
	requireGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.UpdateAdminUser(ctx, &authv1.UpdateAdminUserRequest{UserId: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
	_, err = srv.UpdateAdminUser(actor, &authv1.UpdateAdminUserRequest{UserId: "nope"})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.UpdateAdminUser(actor, &authv1.UpdateAdminUserRequest{
		UserId: uuid.NewString(), HasRoleIds: true, RoleIds: []string{"nope"},
	})
	requireGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.ResendAdminUserInvite(context.Background(), &authv1.ResendAdminUserInviteRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
	_, err = srv.ResendAdminUserInvite(ctx, &authv1.ResendAdminUserInviteRequest{InvitationId: "nope"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestAuthGRPC_ProvisionTenant(t *testing.T) {
	srv, repo := newTestAuthGRPCServer()
	ctx := context.Background()

	_, err := srv.ProvisionTenant(ctx, &authv1.ProvisionTenantRequest{CreatedBy: "nope"})
	requireGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.ProvisionTenant(ctx, &authv1.ProvisionTenantRequest{
		CreatedBy: uuid.NewString(), Name: "Acme", BillingPeriodEnd: "31.01.2027",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)

	// Service-side input validation reaches the caller as InvalidArgument.
	_, err = srv.ProvisionTenant(ctx, &authv1.ProvisionTenantRequest{CreatedBy: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)

	resp, err := srv.ProvisionTenant(ctx, &authv1.ProvisionTenantRequest{
		CreatedBy:        uuid.NewString(),
		Name:             "Acme GmbH",
		PlanType:         "cosmi",
		SupportTier:      "standard",
		AdminEmail:       "admin@acme.example",
		HasSeatLimit:     true,
		SeatLimit:        5,
		BillingPeriodEnd: "2027-01-31",
	})
	require.NoError(t, err)
	require.Equal(t, "Acme GmbH", resp.Tenant.Name)
	require.True(t, resp.Tenant.HasSeatLimit)
	require.Equal(t, int32(5), resp.Tenant.SeatLimit)
	require.Equal(t, "2027-01-31", resp.Tenant.BillingPeriodEnd)
	require.NotEmpty(t, resp.InvitationToken)
	require.Len(t, repo.provisionedTenants, 1)
}

func TestAuthGRPC_RequestPasswordReset_AnswersTheSameForUnknownAccounts(t *testing.T) {
	srv, _ := newTestAuthGRPCServer()
	resp, err := srv.RequestPasswordReset(context.Background(), &authv1.RequestPasswordResetRequest{Email: "nobody@example.com"})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Message)
}

func TestAuthGRPC_ResetPassword_UnknownTokenIsRefused(t *testing.T) {
	srv, _ := newTestAuthGRPCServer()
	_, err := srv.ResetPassword(context.Background(), &authv1.ResetPasswordRequest{Token: "bogus", NewPassword: "Sup3r-Secret-Pass!"})
	require.Error(t, err)
}
