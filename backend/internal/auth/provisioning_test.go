package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/modules"
	"github.com/kmuhub/kmuhub/internal/sysctx"
)

func provisionInput() ProvisionTenantInput {
	return ProvisionTenantInput{
		Name:       "Muster GmbH",
		AdminEmail: "chef@muster.example",
		CreatedBy:  uuid.New(),
	}
}

func TestProvisionTenant_CreatesTenantModulesAndInvitation(t *testing.T) {
	svc, repo := newTestService()

	in := provisionInput()
	res, err := svc.ProvisionTenant(context.Background(), in)
	require.NoError(t, err)

	// Column defaults from migration 000250 apply when the caller books nothing.
	require.Equal(t, "Muster GmbH", res.Tenant.Name)
	require.Equal(t, "cosmi", res.Tenant.PlanType)
	require.Equal(t, "standard", res.Tenant.SupportTier)
	require.Equal(t, "active", res.Tenant.SubscriptionStatus)
	require.Nil(t, res.Tenant.SeatLimit, "no cap booked means unlimited")

	// The invitation is what makes the tenant reachable at all.
	require.Equal(t, res.Tenant.ID, res.Invitation.TenantID)
	require.Equal(t, "admin", res.Invitation.Role)
	require.Equal(t, "chef@muster.example", res.Invitation.Email)
	require.Equal(t, in.CreatedBy, res.Invitation.CreatedBy)
	require.True(t, res.Invitation.ExpiresAt.After(time.Now()))

	// The plaintext token is returned once; only its hash reaches the store.
	require.NotEmpty(t, res.Token)
	require.Equal(t, HashToken(res.Token), res.Invitation.TokenHash)
	require.NotEqual(t, res.Token, res.Invitation.TokenHash)

	// An empty module request activates the whole catalogue.
	require.Len(t, res.Modules, len(modules.Catalog))
	require.Equal(t, res.Modules, repo.provisionedModules[res.Tenant.ID])
	require.NotNil(t, repo.provisionedTenants[res.Tenant.ID])
}

func TestProvisionTenant_RunsInSystemContext(t *testing.T) {
	// Without the system context every write in the transaction hits an RLS
	// policy for a tenant the caller does not belong to. Asserting it here is
	// cheaper than discovering it as a 500 in production.
	repo := &provisionCtxRepo{mockRepository: newMockRepository()}
	svc := NewService(repo, NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour))

	_, err := svc.ProvisionTenant(context.Background(), provisionInput())
	require.NoError(t, err)
	require.True(t, repo.sawSystemContext, "ProvisionTenant must run under sysctx.With")
}

func TestProvisionTenant_ActivatesOnlyRequestedModules(t *testing.T) {
	svc, _ := newTestService()

	in := provisionInput()
	in.Modules = []string{"crm", "tasks", "crm"} // duplicate is collapsed

	res, err := svc.ProvisionTenant(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, []string{"crm", "tasks"}, res.Modules)
}

func TestProvisionTenant_RejectsUnknownModule(t *testing.T) {
	svc, repo := newTestService()

	in := provisionInput()
	in.Modules = []string{"crm", "not-a-module"}

	_, err := svc.ProvisionTenant(context.Background(), in)
	require.ErrorIs(t, err, ErrUnknownModule)
	require.Empty(t, repo.provisionedTenants, "a rejected request must not create a tenant")
}

func TestProvisionTenant_RejectsInvalidInput(t *testing.T) {
	seatLimit := func(n int) *int { return &n }

	tests := []struct {
		name    string
		mutate  func(*ProvisionTenantInput)
		wantErr error
	}{
		{"empty name", func(in *ProvisionTenantInput) { in.Name = "   " }, ErrTenantNameRequired},
		{"name too long", func(in *ProvisionTenantInput) {
			in.Name = string(make([]rune, maxTenantNameLen+1))
		}, ErrTenantNameTooLong},
		{"unknown plan", func(in *ProvisionTenantInput) { in.PlanType = "enterprise" }, ErrInvalidPlanType},
		{"unknown support tier", func(in *ProvisionTenantInput) { in.SupportTier = "gold" }, ErrInvalidSupportTier},
		{"zero seats", func(in *ProvisionTenantInput) { in.SeatLimit = seatLimit(0) }, ErrInvalidSeatLimit},
		{"no admin email", func(in *ProvisionTenantInput) { in.AdminEmail = "" }, ErrAdminEmailRequired},
		{"malformed admin email", func(in *ProvisionTenantInput) { in.AdminEmail = "nobody" }, ErrAdminEmailRequired},
		{"no provisioner", func(in *ProvisionTenantInput) { in.CreatedBy = uuid.Nil }, ErrProvisionerRequired},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newTestService()

			in := provisionInput()
			tc.mutate(&in)

			_, err := svc.ProvisionTenant(context.Background(), in)
			require.ErrorIs(t, err, tc.wantErr)
			require.Empty(t, repo.provisionedTenants)
		})
	}
}

func TestProvisionTenant_RejectsEmailThatAlreadyHasAnAccount(t *testing.T) {
	// users.email is globally unique. Provisioning around an existing account
	// would create a tenant whose invitation can never be accepted.
	repo := newMockRepository()
	repo.usersByEmail["chef@muster.example"] = &models.User{ID: uuid.New(), Email: "chef@muster.example"}
	svc := NewService(repo, NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour))

	_, err := svc.ProvisionTenant(context.Background(), provisionInput())
	require.ErrorIs(t, err, ErrUserExists)
	require.Empty(t, repo.provisionedTenants)
}

func TestProvisionTenant_RejectsEmailInvitedElsewhere(t *testing.T) {
	repo := newMockRepository()
	otherTenant := uuid.New()
	repo.invitations[uuid.New()] = &models.Invitation{
		TenantID:  otherTenant,
		Email:     "chef@muster.example",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	svc := NewService(repo, NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour))

	_, err := svc.ProvisionTenant(context.Background(), provisionInput())
	require.ErrorIs(t, err, ErrInvitationExists)
	require.Empty(t, repo.provisionedTenants)
}

func TestProvisionTenant_NormalizesAdminEmail(t *testing.T) {
	svc, _ := newTestService()

	in := provisionInput()
	in.AdminEmail = "  Chef@Muster.Example  "

	res, err := svc.ProvisionTenant(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, "chef@muster.example", res.Invitation.Email)
}

func TestProvisionTenant_KeepsBookedPlanAndSeatLimit(t *testing.T) {
	svc, _ := newTestService()

	limit := 25
	end := time.Date(2027, 1, 31, 0, 0, 0, 0, time.UTC)
	in := provisionInput()
	in.PlanType = "orbit"
	in.SupportTier = "enterprise"
	in.SeatLimit = &limit
	in.BillingPeriodEnd = &end

	res, err := svc.ProvisionTenant(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, "orbit", res.Tenant.PlanType)
	require.Equal(t, "enterprise", res.Tenant.SupportTier)
	require.Equal(t, 25, *res.Tenant.SeatLimit)
	require.Equal(t, end, *res.Tenant.BillingPeriodEnd)
}

func TestProvisionTenant_FailedWriteLeavesNothing(t *testing.T) {
	repo := newMockRepository()
	repo.provisionFails = errors.New("constraint violation")
	svc := NewService(repo, NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour))

	_, err := svc.ProvisionTenant(context.Background(), provisionInput())
	require.Error(t, err)
	require.Empty(t, repo.provisionedTenants)
	require.Empty(t, repo.invitations, "a failed provisioning must not leave an invitation behind")
}

// provisionCtxRepo records whether ProvisionTenant saw the system context.
type provisionCtxRepo struct {
	*mockRepository
	sawSystemContext bool
}

func (r *provisionCtxRepo) ProvisionTenant(ctx context.Context, tenant *models.Tenant, moduleIDs []string, inv *models.Invitation) error {
	r.sawSystemContext = sysctx.Is(ctx)
	return r.mockRepository.ProvisionTenant(ctx, tenant, moduleIDs, inv)
}
