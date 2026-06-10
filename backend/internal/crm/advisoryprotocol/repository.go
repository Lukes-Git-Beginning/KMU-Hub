package advisoryprotocol

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Protocol holds all fields of an advisory session document.
// Field naming mirrors the FE AdvisoryProtocol type in lib/advisory.ts.
type Protocol struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	ContactID uuid.UUID `json:"contact_id"`
	CreatedBy uuid.UUID `json:"created_by"`

	// Lifecycle
	Status      string     `json:"status"` // draft | finalized
	HandedOverAt *time.Time `json:"handed_over_at,omitempty"`

	// § 1 — Gesprächskopf
	Date             *string `json:"date,omitempty"`              // ISO date string yyyy-mm-dd
	TimeFrom         string  `json:"time_from"`
	TimeTo           string  `json:"time_to"`
	Location         string  `json:"location"`                    // office|phone|video|onsite
	Advisor          string  `json:"advisor"`
	Occasion         string  `json:"occasion"`                    // initial|followup|event
	OccasionNote     string  `json:"occasion_note"`
	CustomerCategory string  `json:"customer_category"`           // private|professional

	// § 2 — Kunde & Profil
	BirthDate     *string `json:"birth_date,omitempty"`
	MaritalStatus string  `json:"marital_status"`
	TaxStatus     string  `json:"tax_status"`

	// § 3 — Kenntnisse & Erfahrungen
	KnownAssetClasses    []string `json:"known_asset_classes"`
	PastTransactions     string   `json:"past_transactions"`
	FinancialEducation   string   `json:"financial_education"`
	ProfessionalExperience string `json:"professional_experience"`
	SelfAssessment       *int     `json:"self_assessment,omitempty"` // 1-5

	// § 4 — Finanzielle Situation
	MonthlyNetIncome    *float64 `json:"monthly_net_income,omitempty"`
	RecurringLiabilities *float64 `json:"recurring_liabilities,omitempty"`
	LiquidAssets         *float64 `json:"liquid_assets,omitempty"`
	CurrentInvestments   *float64 `json:"current_investments,omitempty"`
	RealEstate           string   `json:"real_estate"`
	ExistingInsurance    string   `json:"existing_insurance"`
	MaxLossCapacityAbs   *float64 `json:"max_loss_capacity_abs,omitempty"`
	MaxLossCapacityPct   *float64 `json:"max_loss_capacity_pct,omitempty"`

	// § 5 — Anlageziele & Risikoprofil (PRIIP SRI 1-7)
	InvestmentPurpose []string `json:"investment_purpose"` // retirement|liquidity|growth|saving|speculation
	Horizon           string   `json:"horizon"`            // lt1|1to3|3to5|5to10|gt10
	RiskTolerance     string   `json:"risk_tolerance"`
	RiskCapacity      string   `json:"risk_capacity"`
	RiskClass         int      `json:"risk_class"` // 1-7 SRI
	EsgPreference     bool     `json:"esg_preference"`
	EsgDetails        string   `json:"esg_details"`
	OneTimeAmount     *float64 `json:"one_time_amount,omitempty"`
	MonthlySavings    *float64 `json:"monthly_savings,omitempty"`

	// § 6 — Besprochene Produkte (stored as JSON in DB)
	Products []Product `json:"products"`

	// § 7 — Empfehlung & Geeignetheit
	RecommendationSummary string `json:"recommendation_summary"`
	SuitabilityReasoning  string `json:"suitability_reasoning"`
	GoalReference         string `json:"goal_reference"`
	Alternatives          string `json:"alternatives"`
	NotRecommended        string `json:"not_recommended"`

	// § 8 — Abschluss & Compliance
	MainConcerns          string   `json:"main_concerns"`
	WarningsGiven         []string `json:"warnings_given"` // risk|costs|conflicts|pastPerformance|liquidity
	DocumentDelivered     bool     `json:"document_delivered"`
	DocumentDeliveredDate *string  `json:"document_delivered_date,omitempty"`
	DeliveryForm          string   `json:"delivery_form"` // paper|email|portal
	AdvisorSignature      string   `json:"advisor_signature"`
	CustomerConfirmation  bool     `json:"customer_confirmation"`
	DocumentWaiver        bool     `json:"document_waiver"`
	FollowupDate          *string  `json:"followup_date,omitempty"`
	InternalNotes         string   `json:"internal_notes"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Product represents a single investment product discussed during the advisory session.
// Mirrors the FE AdvisoryProduct type.
type Product struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	ISIN         *string  `json:"isin,omitempty"`
	Category     *string  `json:"category,omitempty"`
	RiskClass    int      `json:"riskClass"` // SRI 1-7; camelCase matches FE JSON
	Opportunities *string `json:"opportunities,omitempty"`
	Risks        string   `json:"risks"`
	CostsOneTime *float64 `json:"costsOneTime,omitempty"`
	CostsRunning *float64 `json:"costsRunning,omitempty"`
	Recommended  bool     `json:"recommended"`
}

// ReferralEntry is returned by the referral report.
type ReferralEntry struct {
	ReferrerID        uuid.UUID `json:"referrer_id"`
	ReferrerFirstName string    `json:"referrer_first_name"`
	ReferrerLastName  string    `json:"referrer_last_name"`
	ReferredCount     int       `json:"referred_count"`
}

// Repository defines the persistence interface for advisory protocols.
type Repository interface {
	Create(ctx context.Context, p *Protocol) error
	GetByID(ctx context.Context, id, tenantID uuid.UUID) (*Protocol, error)
	ListByContact(ctx context.Context, contactID, tenantID uuid.UUID) ([]*Protocol, error)
	Update(ctx context.Context, p *Protocol) error
	Delete(ctx context.Context, id, tenantID uuid.UUID) error
	HandOver(ctx context.Context, id, tenantID uuid.UUID, at time.Time) error

	// ContactExists checks whether the contact belongs to the given tenant.
	ContactExists(ctx context.Context, contactID, tenantID uuid.UUID) (bool, error)

	// GetReferralReport returns aggregated referral counts per referrer for a tenant.
	GetReferralReport(ctx context.Context, tenantID uuid.UUID) ([]*ReferralEntry, error)
}
