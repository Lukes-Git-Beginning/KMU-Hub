package auth

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/modules"
	"github.com/kmuhub/kmuhub/internal/sysctx"
)

// Tenant provisioning lives in internal/auth because everything it writes is
// already owned here: the tenant row, the module activations that hang off it
// and the invitation that lets the first administrator in. Splitting the write
// across two services would mean two transactions, and a half-provisioned
// tenant — a tenant nobody can log into — is worse than a failed request.

// maxTenantNameLen bounds the company name. The column is TEXT; the limit is
// about keeping a display value displayable, not about storage.
const maxTenantNameLen = 200

// validPlanTypes, validSupportTiers mirror the CHECK constraints migration
// 000250 put on tenants. Rejecting here turns a constraint violation (500)
// into a validation error (400) and keeps the two definitions visibly paired.
var (
	validPlanTypes    = map[string]bool{"cosmi": true, "orbit": true}
	validSupportTiers = map[string]bool{"standard": true, "priority": true, "enterprise": true}
)

// ProvisionTenantInput describes the tenant to create. Everything but Name and
// AdminEmail is optional: a tenant provisioned without a plan gets the column
// defaults from migration 000250, and an empty Modules activates the whole
// catalogue.
type ProvisionTenantInput struct {
	Name             string
	PlanType         string
	SupportTier      string
	SeatLimit        *int
	BillingPeriodEnd *time.Time
	// Modules are the catalogue ids to activate. Empty means "all of them".
	Modules []string
	// AdminEmail receives the invitation that creates the first account. It is
	// mandatory: a tenant without a way in is not provisioned, it is stranded.
	AdminEmail string
	// CreatedBy is the platform operator issuing the invitation. It is their
	// own account, from their own tenant — invitations.created_by is NOT NULL
	// and records who provisioned, not who the invitation belongs to.
	CreatedBy uuid.UUID
}

// ProvisionTenantResult is what provisioning produced. Modules is the
// resolved list, not the requested one — an empty request activates the whole
// catalogue, and the caller should be told which modules that turned out to be.
type ProvisionTenantResult struct {
	Tenant     *models.Tenant
	Invitation *models.Invitation
	Modules    []string
	// Token is the plaintext invitation token. It exists exactly once, here;
	// only its hash is stored.
	Token string
}

// ProvisionTenant creates a tenant, activates its modules and issues the
// invitation for its first administrator — all in one transaction, so a
// failure anywhere leaves no tenant behind.
//
// The whole call runs in system context: the tenant being created is by
// definition not the caller's tenant, and the RLS policies on tenants,
// tenant_module_activations and invitations would otherwise reject every write.
// Authorisation therefore cannot come from RLS here — it is the
// tenants:write permission on the route, held only by the platform_admin role.
func (s *Service) ProvisionTenant(ctx context.Context, in ProvisionTenantInput) (*ProvisionTenantResult, error) {
	tenant, inv, err := s.buildProvisioning(in)
	if err != nil {
		return nil, err
	}

	moduleIDs, err := resolveModules(in.Modules)
	if err != nil {
		return nil, err
	}

	ctx = sysctx.With(ctx)

	// Both checks look across every tenant, which is exactly why they need the
	// system context: users.email is globally unique, so an address that
	// already has an account — or an open invitation anywhere — cannot become
	// this tenant's administrator. Without the check the tenant would be
	// created and the invitation would fail to be accepted later, which is the
	// stranded case this method exists to avoid.
	if existing, _ := s.repo.GetUserByEmail(ctx, inv.Email); existing != nil {
		return nil, ErrUserExists
	}
	if pending, _ := s.repo.GetPendingInvitationByEmail(ctx, inv.Email); pending != nil {
		return nil, ErrInvitationExists
	}

	token := generateSecureToken()
	inv.TokenHash = HashToken(token)

	if err := s.repo.ProvisionTenant(ctx, tenant, moduleIDs, inv); err != nil {
		return nil, err
	}

	slog.Info("tenant provisioned",
		"tenant_id", tenant.ID,
		"plan_type", tenant.PlanType,
		"modules", len(moduleIDs),
		"admin_email", inv.Email)

	return &ProvisionTenantResult{
		Tenant:     tenant,
		Invitation: inv,
		Modules:    moduleIDs,
		Token:      token,
	}, nil
}

// buildProvisioning validates the input and assembles the two rows. It touches
// no repository, so the validation rules are testable without a database.
func (s *Service) buildProvisioning(in ProvisionTenantInput) (*models.Tenant, *models.Invitation, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, nil, ErrTenantNameRequired
	}
	if len([]rune(name)) > maxTenantNameLen {
		return nil, nil, ErrTenantNameTooLong
	}

	planType := defaultIfEmpty(in.PlanType, "cosmi")
	if !validPlanTypes[planType] {
		return nil, nil, ErrInvalidPlanType
	}

	supportTier := defaultIfEmpty(in.SupportTier, "standard")
	if !validSupportTiers[supportTier] {
		return nil, nil, ErrInvalidSupportTier
	}

	// The administrator's invitation takes the first seat, so a cap below one
	// would provision a tenant that cannot accept its own admin.
	if in.SeatLimit != nil && *in.SeatLimit < 1 {
		return nil, nil, ErrInvalidSeatLimit
	}

	email := normalizeEmail(in.AdminEmail)
	if email == "" || !strings.Contains(email, "@") {
		return nil, nil, ErrAdminEmailRequired
	}

	if in.CreatedBy == uuid.Nil {
		return nil, nil, ErrProvisionerRequired
	}

	now := time.Now()
	tenant := &models.Tenant{
		ID:                 uuid.New(),
		Name:               name,
		PlanType:           planType,
		SupportTier:        supportTier,
		SubscriptionStatus: "active",
		BillingPeriodEnd:   in.BillingPeriodEnd,
		SeatLimit:          in.SeatLimit,
		CreatedAt:          now,
	}

	inv := &models.Invitation{
		ID:        uuid.New(),
		TenantID:  tenant.ID,
		Email:     email,
		Role:      "admin",
		CreatedBy: in.CreatedBy,
		ExpiresAt: now.Add(invitationExpiry),
		CreatedAt: now,
	}

	return tenant, inv, nil
}

// resolveModules maps the requested ids onto the catalogue. An unknown id is
// rejected rather than skipped: silently dropping it would provision a tenant
// missing a module somebody thought they had booked.
//
// Deployment feature flags are deliberately not consulted. They describe what
// this installation runs today and the gateway masks the activation on read —
// letting a flag decide what gets written would mean a tenant loses its
// booking the moment a flag is switched off.
func resolveModules(requested []string) ([]string, error) {
	if len(requested) == 0 {
		all := make([]string, 0, len(modules.Catalog))
		for _, m := range modules.Catalog {
			all = append(all, m.ID)
		}
		return all, nil
	}

	seen := make(map[string]bool, len(requested))
	out := make([]string, 0, len(requested))
	for _, id := range requested {
		if _, ok := modules.Get(id); !ok {
			return nil, ErrUnknownModule
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out, nil
}

func defaultIfEmpty(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
