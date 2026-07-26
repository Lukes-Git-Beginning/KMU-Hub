package models

import (
	"time"

	"github.com/google/uuid"
)

// DefaultTenantID is the bootstrap tenant created by migration 000114.
// Users registered via public sign-up land here; an accepted invitation puts
// the account into the tenant that issued it (migration 000249).
var DefaultTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// Tenant is one customer installation and the root of the tenant model: every
// other table's tenant_id points here, and the RLS policy on tenants itself
// compares id against current_tenant_id().
//
// The contract fields live on the row rather than in a table of their own
// because a booked plan is a property of the tenant, not an event log — see
// migration 000250.
type Tenant struct {
	ID                 uuid.UUID  `json:"id"`
	Name               string     `json:"name"`
	PlanType           string     `json:"plan_type"`
	SupportTier        string     `json:"support_tier"`
	SubscriptionStatus string     `json:"subscription_status"`
	BillingPeriodEnd   *time.Time `json:"billing_period_end,omitempty"`
	// SeatLimit is nil for a tenant without a booked cap.
	SeatLimit *int      `json:"seat_limit,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
