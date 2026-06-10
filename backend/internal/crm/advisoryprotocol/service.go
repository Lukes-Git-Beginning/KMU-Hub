package advisoryprotocol

import (
	"bytes"
	"strings"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/pagesize"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// Service implements advisory protocol business logic.
// Immutability rule: Update and Delete are rejected with ErrProtocolFinalized
// once status == "finalized". HandOver atomically sets status + handed_over_at.
type Service struct {
	repo Repository
}

// NewService creates a new advisory protocol service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput holds the data for creating a new draft protocol.
type CreateInput struct {
	TenantID  uuid.UUID
	ContactID uuid.UUID
	CreatedBy uuid.UUID
}

// Create creates a blank draft protocol for a contact.
func (s *Service) Create(ctx context.Context, in CreateInput) (*Protocol, error) {
	if in.TenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}

	exists, err := s.repo.ContactExists(ctx, in.ContactID, in.TenantID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrContactNotFound
	}

	now := time.Now()
	p := &Protocol{
		ID:                    uuid.New(),
		TenantID:              in.TenantID,
		ContactID:             in.ContactID,
		CreatedBy:             in.CreatedBy,
		Status:                "draft",
		KnownAssetClasses:     []string{},
		InvestmentPurpose:     []string{},
		WarningsGiven:         []string{},
		Products:              []Product{},
		RiskClass:             3,
		DeliveryForm:          "email",
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}

	slog.Info("advisory_protocol.created",
		"protocol_id", p.ID,
		"contact_id", p.ContactID,
		"tenant_id", p.TenantID,
	)
	return p, nil
}

// GetByID retrieves a protocol by ID scoped to a tenant.
func (s *Service) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*Protocol, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	p, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, ErrProtocolNotFound
	}
	return p, nil
}

// ListByContact returns all protocols for a contact ordered by date desc.
func (s *Service) ListByContact(ctx context.Context, contactID, tenantID uuid.UUID) ([]*Protocol, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	return s.repo.ListByContact(ctx, contactID, tenantID)
}

// UpdateInput carries the mutable fields. nil-pointer fields are left unchanged.
// All fields are applied wholesale because the FE sends a full protocol snapshot.
type UpdateInput struct {
	Date             *string
	TimeFrom         string
	TimeTo           string
	Location         string
	Advisor          string
	Occasion         string
	OccasionNote     string
	CustomerCategory string
	BirthDate        *string
	MaritalStatus    string
	TaxStatus        string

	KnownAssetClasses      []string
	PastTransactions       string
	FinancialEducation     string
	ProfessionalExperience string
	SelfAssessment         *int

	MonthlyNetIncome      *float64
	RecurringLiabilities  *float64
	LiquidAssets          *float64
	CurrentInvestments    *float64
	RealEstate            string
	ExistingInsurance     string
	MaxLossCapacityAbs    *float64
	MaxLossCapacityPct    *float64

	InvestmentPurpose []string
	Horizon           string
	RiskTolerance     string
	RiskCapacity      string
	RiskClass         int
	EsgPreference     bool
	EsgDetails        string
	OneTimeAmount     *float64
	MonthlySavings    *float64

	Products []Product

	RecommendationSummary string
	SuitabilityReasoning  string
	GoalReference         string
	Alternatives          string
	NotRecommended        string

	MainConcerns          string
	WarningsGiven         []string
	DocumentDelivered     bool
	DocumentDeliveredDate *string
	DeliveryForm          string
	AdvisorSignature      string
	CustomerConfirmation  bool
	DocumentWaiver        bool
	FollowupDate          *string
	InternalNotes         string
}

// Update patches a draft protocol. Returns ErrProtocolFinalized if already handed over.
func (s *Service) Update(ctx context.Context, id, tenantID uuid.UUID, in UpdateInput) (*Protocol, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}

	p, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, ErrProtocolNotFound
	}
	if p.Status == "finalized" {
		return nil, ErrProtocolFinalized
	}

	if in.RiskClass < 1 || in.RiskClass > 7 {
		return nil, ErrInvalidRiskClass
	}

	// Apply all fields
	p.Date = in.Date
	p.TimeFrom = in.TimeFrom
	p.TimeTo = in.TimeTo
	p.Location = in.Location
	p.Advisor = in.Advisor
	p.Occasion = in.Occasion
	p.OccasionNote = in.OccasionNote
	p.CustomerCategory = in.CustomerCategory
	p.BirthDate = in.BirthDate
	p.MaritalStatus = in.MaritalStatus
	p.TaxStatus = in.TaxStatus
	p.KnownAssetClasses = in.KnownAssetClasses
	p.PastTransactions = in.PastTransactions
	p.FinancialEducation = in.FinancialEducation
	p.ProfessionalExperience = in.ProfessionalExperience
	p.SelfAssessment = in.SelfAssessment
	p.MonthlyNetIncome = in.MonthlyNetIncome
	p.RecurringLiabilities = in.RecurringLiabilities
	p.LiquidAssets = in.LiquidAssets
	p.CurrentInvestments = in.CurrentInvestments
	p.RealEstate = in.RealEstate
	p.ExistingInsurance = in.ExistingInsurance
	p.MaxLossCapacityAbs = in.MaxLossCapacityAbs
	p.MaxLossCapacityPct = in.MaxLossCapacityPct
	p.InvestmentPurpose = in.InvestmentPurpose
	p.Horizon = in.Horizon
	p.RiskTolerance = in.RiskTolerance
	p.RiskCapacity = in.RiskCapacity
	p.RiskClass = in.RiskClass
	p.EsgPreference = in.EsgPreference
	p.EsgDetails = in.EsgDetails
	p.OneTimeAmount = in.OneTimeAmount
	p.MonthlySavings = in.MonthlySavings
	p.Products = in.Products
	p.RecommendationSummary = in.RecommendationSummary
	p.SuitabilityReasoning = in.SuitabilityReasoning
	p.GoalReference = in.GoalReference
	p.Alternatives = in.Alternatives
	p.NotRecommended = in.NotRecommended
	p.MainConcerns = in.MainConcerns
	p.WarningsGiven = in.WarningsGiven
	p.DocumentDelivered = in.DocumentDelivered
	p.DocumentDeliveredDate = in.DocumentDeliveredDate
	p.DeliveryForm = in.DeliveryForm
	p.AdvisorSignature = in.AdvisorSignature
	p.CustomerConfirmation = in.CustomerConfirmation
	p.DocumentWaiver = in.DocumentWaiver
	p.FollowupDate = in.FollowupDate
	p.InternalNotes = in.InternalNotes
	p.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}

	slog.Info("advisory_protocol.updated",
		"protocol_id", p.ID,
		"tenant_id", p.TenantID,
	)
	return p, nil
}

// Delete removes a draft protocol. Returns ErrProtocolFinalized if already handed over.
func (s *Service) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	if tenantID == uuid.Nil {
		return ErrInvalidTenant
	}

	p, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return ErrProtocolNotFound
	}
	if p.Status == "finalized" {
		return ErrProtocolFinalized
	}

	if err := s.repo.Delete(ctx, id, tenantID); err != nil {
		return err
	}

	slog.Info("advisory_protocol.deleted",
		"protocol_id", id,
		"tenant_id", tenantID,
	)
	return nil
}

// HandOver finalizes a protocol and records the handover timestamp.
// After this call the protocol is immutable (ErrProtocolFinalized on Update/Delete).
func (s *Service) HandOver(ctx context.Context, id, tenantID uuid.UUID) (*Protocol, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}

	p, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, ErrProtocolNotFound
	}
	if p.Status == "finalized" {
		// Idempotent: already handed over, return as-is.
		return p, nil
	}

	now := time.Now()
	if err := s.repo.HandOver(ctx, id, tenantID, now); err != nil {
		return nil, err
	}

	p.Status = "finalized"
	p.HandedOverAt = &now

	slog.Info("advisory_protocol.handed_over",
		"protocol_id", id,
		"tenant_id", tenantID,
	)
	return p, nil
}

// GetReferralReport returns a per-referrer aggregation for the tenant.
func (s *Service) GetReferralReport(ctx context.Context, tenantID uuid.UUID) ([]*ReferralEntry, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenant
	}
	return s.repo.GetReferralReport(ctx, tenantID)
}

// GeneratePDF renders a finalized protocol as a PDF byte slice.
// Uses the same maroto v2 library as internal/biz/pdf — no new dependency.
// The PDF is a server-side "Geeignetheitserklärung" suitable for archiving.
func (s *Service) GeneratePDF(ctx context.Context, p *Protocol, contactName, companyName string) ([]byte, error) {
	cfg := config.NewBuilder().
		WithPageSize(pagesize.A4).
		WithLeftMargin(15).
		WithTopMargin(15).
		WithRightMargin(15).
		WithPageNumber().
		Build()

	m := maroto.New(cfg)

	// Title
	m.AddRows(
		row.New(12).Add(
			text.NewCol(12, "Geeignetheitserklärung / Beratungsprotokoll",
				props.Text{Size: 16, Style: fontstyle.Bold, Align: align.Left}),
		),
	)
	if companyName != "" {
		m.AddRows(
			row.New(7).Add(
				text.NewCol(12, companyName,
					props.Text{Size: 10, Align: align.Left}),
			),
		)
	}
	m.AddRows(row.New(4))

	// Section helper
	addSection := func(title string) {
		m.AddRows(row.New(2))
		m.AddRows(
			row.New(7).Add(
				text.NewCol(12, title,
					props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Left}),
			),
		)
	}
	addField := func(label, value string) {
		if value == "" {
			return
		}
		m.AddRows(
			row.New(6).Add(
				text.NewCol(4, label+":", props.Text{Size: 8, Style: fontstyle.Bold}),
				text.NewCol(8, value, props.Text{Size: 8}),
			),
		)
	}

	// Parties
	addField("Kunde", contactName)
	addField("Berater", p.Advisor)
	if p.Date != nil {
		addField("Datum", *p.Date)
	}
	if p.TimeFrom != "" && p.TimeTo != "" {
		addField("Uhrzeit", fmt.Sprintf("%s – %s", p.TimeFrom, p.TimeTo))
	}
	addField("Ort", p.Location)
	addField("Anlass", p.Occasion)
	addField("Kundenkategorie", p.CustomerCategory)

	// § 3
	addSection("Kenntnisse & Erfahrungen")
	addField("Bekannte Anlagearten", joinStrings(p.KnownAssetClasses))
	addField("Frühere Transaktionen", p.PastTransactions)
	addField("Finanzbildung", p.FinancialEducation)
	addField("Berufserfahrung", p.ProfessionalExperience)

	// § 4
	addSection("Finanzielle Situation")
	if p.MonthlyNetIncome != nil {
		addField("Nettoeinkommen/Monat", fmt.Sprintf("%.2f EUR", *p.MonthlyNetIncome))
	}
	if p.LiquidAssets != nil {
		addField("Liquide Mittel", fmt.Sprintf("%.2f EUR", *p.LiquidAssets))
	}
	if p.MaxLossCapacityPct != nil {
		addField("Max. Verlusttragfähigkeit", fmt.Sprintf("%.1f%%", *p.MaxLossCapacityPct))
	}

	// § 5
	addSection("Anlageziele & Risikoprofil")
	addField("Anlagezweck", joinStrings(p.InvestmentPurpose))
	addField("Anlagehorizont", p.Horizon)
	addField("Risikoklasse (SRI)", fmt.Sprintf("%d / 7", p.RiskClass))
	if p.EsgPreference {
		addField("ESG-Präferenz", "Ja — "+p.EsgDetails)
	} else {
		addField("ESG-Präferenz", "Nein")
	}

	// § 7
	addSection("Empfehlung & Geeignetheitsbegründung")
	addField("Zusammenfassung", p.RecommendationSummary)
	addField("Geeignetheitsbegründung", p.SuitabilityReasoning)
	addField("Bezug zu Zielen", p.GoalReference)

	// § 8
	addSection("Abschluss & Compliance")
	addField("Hauptanliegen", p.MainConcerns)
	addField("Warnhinweise erteilt", joinStrings(p.WarningsGiven))
	addField("Aushändigungsform", p.DeliveryForm)
	if p.DocumentDeliveredDate != nil {
		addField("Aushändigungsdatum", *p.DocumentDeliveredDate)
	}
	addField("Unterschrift Berater", p.AdvisorSignature)

	// Signature lines
	m.AddRows(row.New(15))
	m.AddRows(
		row.New(6).Add(
			text.NewCol(5, "____________________________", props.Text{Size: 8}),
			text.NewCol(2, ""),
			text.NewCol(5, "____________________________", props.Text{Size: 8}),
		),
	)
	m.AddRows(
		row.New(6).Add(
			text.NewCol(5, "Unterschrift Berater", props.Text{Size: 7}),
			text.NewCol(2, ""),
			text.NewCol(5, "Unterschrift Kunde", props.Text{Size: 7}),
		),
	)

	// Retention notice
	m.AddRows(row.New(8))
	m.AddRows(
		row.New(6).Add(
			text.NewCol(12,
				"Dieses Dokument wird gemäß §18a FinVermV 10 Jahre aufbewahrt (Rechtsgrundlage: Art. 6 Abs. 1 lit. c DSGVO).",
				props.Text{Size: 7}),
		),
	)

	var buf bytes.Buffer
	doc, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("advisory protocol PDF generation failed: %w", err)
	}
	if _, err := buf.Write(doc.GetBytes()); err != nil {
		return nil, fmt.Errorf("writing advisory PDF to buffer: %w", err)
	}
	return buf.Bytes(), nil
}

func joinStrings(ss []string) string {
	return strings.Join(ss, ", ")
}
