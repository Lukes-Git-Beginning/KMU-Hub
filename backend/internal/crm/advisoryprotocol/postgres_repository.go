package advisoryprotocol

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository against PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new repository backed by the given pool.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, p *Protocol) error {
	productsJSON, err := json.Marshal(p.Products)
	if err != nil {
		return err
	}
	knownClasses := nullableTextArray(p.KnownAssetClasses)
	purposes := nullableTextArray(p.InvestmentPurpose)
	warnings := nullableTextArray(p.WarningsGiven)

	_, err = r.pool.Exec(ctx,
		`INSERT INTO advisory_protocols (
			id, tenant_id, contact_id, created_by, status,
			time_from, time_to, location, advisor, occasion, occasion_note, customer_category,
			marital_status, tax_status,
			known_asset_classes, past_transactions, financial_education, professional_experience, self_assessment,
			real_estate, existing_insurance,
			investment_purpose, horizon, risk_tolerance, risk_capacity, risk_class,
			esg_preference, esg_details,
			products,
			recommendation_summary, suitability_reasoning, goal_reference, alternatives, not_recommended,
			main_concerns, warnings_given, document_delivered, delivery_form, advisor_signature,
			customer_confirmation, document_waiver, internal_notes,
			created_at, updated_at,
			date, birth_date,
			monthly_net_income, recurring_liabilities, liquid_assets, current_investments,
			max_loss_capacity_abs, max_loss_capacity_pct,
			one_time_amount, monthly_savings,
			document_delivered_date, followup_date
		) VALUES (
			$1,$2,$3,$4,$5,
			$6,$7,$8,$9,$10,$11,$12,
			$13,$14,
			$15,$16,$17,$18,$19,
			$20,$21,
			$22,$23,$24,$25,$26,
			$27,$28,
			$29,
			$30,$31,$32,$33,$34,
			$35,$36,$37,$38,$39,
			$40,$41,$42,
			$43,$44,
			$45,$46,
			$47,$48,$49,$50,
			$51,$52,
			$53,$54,
			$55,$56
		)`,
		p.ID, p.TenantID, p.ContactID, p.CreatedBy, p.Status,
		emptyToNil(p.TimeFrom), emptyToNil(p.TimeTo), emptyToNil(p.Location), p.Advisor,
		emptyToNil(p.Occasion), p.OccasionNote, emptyToNil(p.CustomerCategory),
		p.MaritalStatus, p.TaxStatus,
		knownClasses, p.PastTransactions, p.FinancialEducation, p.ProfessionalExperience, p.SelfAssessment,
		p.RealEstate, p.ExistingInsurance,
		purposes, emptyToNil(p.Horizon), p.RiskTolerance, p.RiskCapacity, p.RiskClass,
		p.EsgPreference, p.EsgDetails,
		productsJSON,
		p.RecommendationSummary, p.SuitabilityReasoning, p.GoalReference, p.Alternatives, p.NotRecommended,
		p.MainConcerns, warnings, p.DocumentDelivered, p.DeliveryForm, p.AdvisorSignature,
		p.CustomerConfirmation, p.DocumentWaiver, p.InternalNotes,
		p.CreatedAt, p.UpdatedAt,
		p.Date, p.BirthDate,
		p.MonthlyNetIncome, p.RecurringLiabilities, p.LiquidAssets, p.CurrentInvestments,
		p.MaxLossCapacityAbs, p.MaxLossCapacityPct,
		p.OneTimeAmount, p.MonthlySavings,
		p.DocumentDeliveredDate, p.FollowupDate,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*Protocol, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT
			id, tenant_id, contact_id, created_by, status, handed_over_at,
			date, time_from, time_to, location, advisor, occasion, occasion_note, customer_category,
			birth_date, marital_status, tax_status,
			known_asset_classes, past_transactions, financial_education, professional_experience, self_assessment,
			monthly_net_income, recurring_liabilities, liquid_assets, current_investments,
			real_estate, existing_insurance, max_loss_capacity_abs, max_loss_capacity_pct,
			investment_purpose, horizon, risk_tolerance, risk_capacity, risk_class,
			esg_preference, esg_details, one_time_amount, monthly_savings,
			products,
			recommendation_summary, suitability_reasoning, goal_reference, alternatives, not_recommended,
			main_concerns, warnings_given, document_delivered, document_delivered_date, delivery_form,
			advisor_signature, customer_confirmation, document_waiver, followup_date, internal_notes,
			created_at, updated_at
		FROM advisory_protocols WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	)
	return scanProtocol(row)
}

func (r *PostgresRepository) ListByContact(ctx context.Context, contactID, tenantID uuid.UUID) ([]*Protocol, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT
			id, tenant_id, contact_id, created_by, status, handed_over_at,
			date, time_from, time_to, location, advisor, occasion, occasion_note, customer_category,
			birth_date, marital_status, tax_status,
			known_asset_classes, past_transactions, financial_education, professional_experience, self_assessment,
			monthly_net_income, recurring_liabilities, liquid_assets, current_investments,
			real_estate, existing_insurance, max_loss_capacity_abs, max_loss_capacity_pct,
			investment_purpose, horizon, risk_tolerance, risk_capacity, risk_class,
			esg_preference, esg_details, one_time_amount, monthly_savings,
			products,
			recommendation_summary, suitability_reasoning, goal_reference, alternatives, not_recommended,
			main_concerns, warnings_given, document_delivered, document_delivered_date, delivery_form,
			advisor_signature, customer_confirmation, document_waiver, followup_date, internal_notes,
			created_at, updated_at
		FROM advisory_protocols
		WHERE contact_id = $1 AND tenant_id = $2
		ORDER BY COALESCE(date, created_at::date) DESC, created_at DESC`,
		contactID, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*Protocol
	for rows.Next() {
		p, err := scanProtocol(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, p *Protocol) error {
	productsJSON, err := json.Marshal(p.Products)
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(ctx,
		`UPDATE advisory_protocols SET
			date=$3, time_from=$4, time_to=$5, location=$6, advisor=$7, occasion=$8,
			occasion_note=$9, customer_category=$10,
			birth_date=$11, marital_status=$12, tax_status=$13,
			known_asset_classes=$14, past_transactions=$15, financial_education=$16,
			professional_experience=$17, self_assessment=$18,
			monthly_net_income=$19, recurring_liabilities=$20, liquid_assets=$21,
			current_investments=$22, real_estate=$23, existing_insurance=$24,
			max_loss_capacity_abs=$25, max_loss_capacity_pct=$26,
			investment_purpose=$27, horizon=$28, risk_tolerance=$29, risk_capacity=$30,
			risk_class=$31, esg_preference=$32, esg_details=$33,
			one_time_amount=$34, monthly_savings=$35,
			products=$36,
			recommendation_summary=$37, suitability_reasoning=$38, goal_reference=$39,
			alternatives=$40, not_recommended=$41,
			main_concerns=$42, warnings_given=$43, document_delivered=$44,
			document_delivered_date=$45, delivery_form=$46, advisor_signature=$47,
			customer_confirmation=$48, document_waiver=$49, followup_date=$50,
			internal_notes=$51, updated_at=$52
		WHERE id=$1 AND tenant_id=$2 AND status='draft'`,
		p.ID, p.TenantID,
		p.Date, emptyToNil(p.TimeFrom), emptyToNil(p.TimeTo), emptyToNil(p.Location), p.Advisor,
		emptyToNil(p.Occasion), p.OccasionNote, emptyToNil(p.CustomerCategory),
		p.BirthDate, p.MaritalStatus, p.TaxStatus,
		nullableTextArray(p.KnownAssetClasses), p.PastTransactions, p.FinancialEducation,
		p.ProfessionalExperience, p.SelfAssessment,
		p.MonthlyNetIncome, p.RecurringLiabilities, p.LiquidAssets,
		p.CurrentInvestments, p.RealEstate, p.ExistingInsurance,
		p.MaxLossCapacityAbs, p.MaxLossCapacityPct,
		nullableTextArray(p.InvestmentPurpose), emptyToNil(p.Horizon), p.RiskTolerance, p.RiskCapacity,
		p.RiskClass, p.EsgPreference, p.EsgDetails,
		p.OneTimeAmount, p.MonthlySavings,
		productsJSON,
		p.RecommendationSummary, p.SuitabilityReasoning, p.GoalReference,
		p.Alternatives, p.NotRecommended,
		p.MainConcerns, nullableTextArray(p.WarningsGiven), p.DocumentDelivered,
		p.DocumentDeliveredDate, emptyToNil(p.DeliveryForm), p.AdvisorSignature,
		p.CustomerConfirmation, p.DocumentWaiver, p.FollowupDate,
		p.InternalNotes, p.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM advisory_protocols WHERE id=$1 AND tenant_id=$2 AND status='draft'`,
		id, tenantID,
	)
	return err
}

func (r *PostgresRepository) HandOver(ctx context.Context, id, tenantID uuid.UUID, at time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE advisory_protocols
		 SET status='finalized', handed_over_at=$3, updated_at=$3
		 WHERE id=$1 AND tenant_id=$2`,
		id, tenantID, at,
	)
	return err
}

func (r *PostgresRepository) ContactExists(ctx context.Context, contactID, tenantID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM contacts WHERE id=$1 AND tenant_id=$2)`,
		contactID, tenantID,
	).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) GetReferralReport(ctx context.Context, tenantID uuid.UUID) ([]*ReferralEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT
			c.referred_by_contact_id,
			r.first_name,
			r.last_name,
			COUNT(*) AS referred_count
		FROM contacts c
		JOIN contacts r ON r.id = c.referred_by_contact_id AND r.tenant_id = $1
		WHERE c.tenant_id = $1
		  AND c.referred_by_contact_id IS NOT NULL
		GROUP BY c.referred_by_contact_id, r.first_name, r.last_name
		ORDER BY referred_count DESC`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*ReferralEntry
	for rows.Next() {
		var e ReferralEntry
		if err := rows.Scan(&e.ReferrerID, &e.ReferrerFirstName, &e.ReferrerLastName, &e.ReferredCount); err != nil {
			return nil, err
		}
		result = append(result, &e)
	}
	return result, rows.Err()
}

// ---------------------------------------------------------------------------
// Scan helpers
// ---------------------------------------------------------------------------

type scannable interface {
	Scan(dest ...any) error
}

func scanProtocol(row scannable) (*Protocol, error) {
	var p Protocol
	var productsJSON []byte
	var knownClasses, investmentPurpose, warningsGiven []string

	err := row.Scan(
		&p.ID, &p.TenantID, &p.ContactID, &p.CreatedBy, &p.Status, &p.HandedOverAt,
		&p.Date, &p.TimeFrom, &p.TimeTo, &p.Location, &p.Advisor, &p.Occasion, &p.OccasionNote, &p.CustomerCategory,
		&p.BirthDate, &p.MaritalStatus, &p.TaxStatus,
		&knownClasses, &p.PastTransactions, &p.FinancialEducation, &p.ProfessionalExperience, &p.SelfAssessment,
		&p.MonthlyNetIncome, &p.RecurringLiabilities, &p.LiquidAssets, &p.CurrentInvestments,
		&p.RealEstate, &p.ExistingInsurance, &p.MaxLossCapacityAbs, &p.MaxLossCapacityPct,
		&investmentPurpose, &p.Horizon, &p.RiskTolerance, &p.RiskCapacity, &p.RiskClass,
		&p.EsgPreference, &p.EsgDetails, &p.OneTimeAmount, &p.MonthlySavings,
		&productsJSON,
		&p.RecommendationSummary, &p.SuitabilityReasoning, &p.GoalReference, &p.Alternatives, &p.NotRecommended,
		&p.MainConcerns, &warningsGiven, &p.DocumentDelivered, &p.DocumentDeliveredDate, &p.DeliveryForm,
		&p.AdvisorSignature, &p.CustomerConfirmation, &p.DocumentWaiver, &p.FollowupDate, &p.InternalNotes,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProtocolNotFound
		}
		return nil, err
	}

	p.KnownAssetClasses = orEmpty(knownClasses)
	p.InvestmentPurpose = orEmpty(investmentPurpose)
	p.WarningsGiven = orEmpty(warningsGiven)

	if len(productsJSON) > 0 {
		if err := json.Unmarshal(productsJSON, &p.Products); err != nil {
			return nil, err
		}
	}
	if p.Products == nil {
		p.Products = []Product{}
	}

	return &p, nil
}

// nullableTextArray converts []string to nil when empty so pgx stores NULL.
func nullableTextArray(ss []string) interface{} {
	if len(ss) == 0 {
		return nil
	}
	return ss
}

// emptyToNil converts an empty string to nil for nullable VARCHAR columns.
func emptyToNil(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func orEmpty(ss []string) []string {
	if ss == nil {
		return []string{}
	}
	return ss
}
