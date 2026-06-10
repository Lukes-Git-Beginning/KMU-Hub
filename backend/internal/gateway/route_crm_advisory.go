package gateway

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// ============================================================================
// Advisory Protocol HTTP handlers (Beratungsprotokolle — P8)
// ============================================================================
//
// Wiring snippet for setup.go (add AFTER existing crmRoutes.RegisterRoutes call):
//
//   crmRoutes.RegisterAdvisoryRoutes(r, authMiddleware)
//
// That is the only change needed in setup.go/main.go — no other files to touch.
// ============================================================================

// RegisterAdvisoryRoutes mounts all advisory protocol endpoints on the given router.
func (c *CRMRoutes) RegisterAdvisoryRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/contacts/{contactId}/advisory-protocols", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("advisory-protocols", "read")).Get("/", c.HandleListAdvisoryProtocols)
		r.With(middleware.RequirePermission("advisory-protocols", "write")).Post("/", c.HandleCreateAdvisoryProtocol)
	})

	r.Route("/api/v1/advisory-protocols/{id}", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("advisory-protocols", "read")).Get("/", c.HandleGetAdvisoryProtocol)
		r.With(middleware.RequirePermission("advisory-protocols", "write")).Put("/", c.HandleUpdateAdvisoryProtocol)
		r.With(middleware.RequirePermission("advisory-protocols", "delete")).Delete("/", c.HandleDeleteAdvisoryProtocol)
		r.With(middleware.RequirePermission("advisory-protocols", "write")).Post("/hand-over", c.HandleHandOverAdvisoryProtocol)
		r.With(middleware.RequirePermission("advisory-protocols", "read")).Get("/pdf", c.HandleAdvisoryProtocolPDF)
	})

	// Referral report (tenant-wide, read permission)
	r.Route("/api/v1/contacts/referral-report", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("advisory-protocols", "read")).Get("/", c.HandleReferralReport)
	})
}

// HandleCreateAdvisoryProtocol POST /api/v1/contacts/{contactId}/advisory-protocols
func (c *CRMRoutes) HandleCreateAdvisoryProtocol(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	contactID, ok := validateUUIDParam(w, r, "contactId")
	if !ok {
		return
	}

	resp, err := client.CreateAdvisoryProtocol(r.Context(), &crmv1.CreateAdvisoryProtocolRequest{
		ContactId: contactID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

// HandleGetAdvisoryProtocol GET /api/v1/advisory-protocols/{id}
func (c *CRMRoutes) HandleGetAdvisoryProtocol(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetAdvisoryProtocol(r.Context(), &crmv1.GetAdvisoryProtocolRequest{Id: id})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// HandleListAdvisoryProtocols GET /api/v1/contacts/{contactId}/advisory-protocols
func (c *CRMRoutes) HandleListAdvisoryProtocols(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	contactID, ok := validateUUIDParam(w, r, "contactId")
	if !ok {
		return
	}

	resp, err := client.ListAdvisoryProtocols(r.Context(), &crmv1.ListAdvisoryProtocolsRequest{ContactId: contactID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// updateAdvisoryProtocolRequest is the JSON body for updating a protocol.
// All fields are sent wholesale (full snapshot from FE store).
type updateAdvisoryProtocolRequest struct {
	Date                  *string           `json:"date,omitempty"`
	TimeFrom              string            `json:"time_from"`
	TimeTo                string            `json:"time_to"`
	Location              string            `json:"location"`
	Advisor               string            `json:"advisor" validate:"required"`
	Occasion              string            `json:"occasion"`
	OccasionNote          string            `json:"occasion_note"`
	CustomerCategory      string            `json:"customer_category"`
	BirthDate             *string           `json:"birth_date,omitempty"`
	MaritalStatus         string            `json:"marital_status"`
	TaxStatus             string            `json:"tax_status"`
	KnownAssetClasses     []string          `json:"known_asset_classes"`
	PastTransactions      string            `json:"past_transactions"`
	FinancialEducation    string            `json:"financial_education"`
	ProfessionalExperience string           `json:"professional_experience"`
	SelfAssessment        *int32            `json:"self_assessment,omitempty" validate:"omitempty,min=1,max=5"`
	MonthlyNetIncome      *float64          `json:"monthly_net_income,omitempty"`
	RecurringLiabilities  *float64          `json:"recurring_liabilities,omitempty"`
	LiquidAssets          *float64          `json:"liquid_assets,omitempty"`
	CurrentInvestments    *float64          `json:"current_investments,omitempty"`
	RealEstate            string            `json:"real_estate"`
	ExistingInsurance     string            `json:"existing_insurance"`
	MaxLossCapacityAbs    *float64          `json:"max_loss_capacity_abs,omitempty"`
	MaxLossCapacityPct    *float64          `json:"max_loss_capacity_pct,omitempty"`
	InvestmentPurpose     []string          `json:"investment_purpose"`
	Horizon               string            `json:"horizon"`
	RiskTolerance         string            `json:"risk_tolerance"`
	RiskCapacity          string            `json:"risk_capacity"`
	RiskClass             int32             `json:"risk_class" validate:"min=1,max=7"`
	EsgPreference         bool              `json:"esg_preference"`
	EsgDetails            string            `json:"esg_details"`
	OneTimeAmount         *float64          `json:"one_time_amount,omitempty"`
	MonthlySavings        *float64          `json:"monthly_savings,omitempty"`
	Products              []advisoryProduct `json:"products"`
	RecommendationSummary string            `json:"recommendation_summary"`
	SuitabilityReasoning  string            `json:"suitability_reasoning"`
	GoalReference         string            `json:"goal_reference"`
	Alternatives          string            `json:"alternatives"`
	NotRecommended        string            `json:"not_recommended"`
	MainConcerns          string            `json:"main_concerns"`
	WarningsGiven         []string          `json:"warnings_given"`
	DocumentDelivered     bool              `json:"document_delivered"`
	DocumentDeliveredDate *string           `json:"document_delivered_date,omitempty"`
	DeliveryForm          string            `json:"delivery_form"`
	AdvisorSignature      string            `json:"advisor_signature"`
	CustomerConfirmation  bool              `json:"customer_confirmation"`
	DocumentWaiver        bool              `json:"document_waiver"`
	FollowupDate          *string           `json:"followup_date,omitempty"`
	InternalNotes         string            `json:"internal_notes"`
}

type advisoryProduct struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	ISIN          *string  `json:"isin,omitempty"`
	Category      *string  `json:"category,omitempty"`
	RiskClass     int32    `json:"risk_class" validate:"min=1,max=7"`
	Opportunities *string  `json:"opportunities,omitempty"`
	Risks         string   `json:"risks"`
	CostsOneTime  *float64 `json:"costs_one_time,omitempty"`
	CostsRunning  *float64 `json:"costs_running,omitempty"`
	Recommended   bool     `json:"recommended"`
}

// HandleUpdateAdvisoryProtocol PUT /api/v1/advisory-protocols/{id}
func (c *CRMRoutes) HandleUpdateAdvisoryProtocol(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateAdvisoryProtocolRequest](w, r)
	if !ok {
		return
	}

	protoPayload := &crmv1.AdvisoryProtocolProto{
		Date:                  req.Date,
		TimeFrom:              req.TimeFrom,
		TimeTo:                req.TimeTo,
		Location:              req.Location,
		Advisor:               req.Advisor,
		Occasion:              req.Occasion,
		OccasionNote:          req.OccasionNote,
		CustomerCategory:      req.CustomerCategory,
		BirthDate:             req.BirthDate,
		MaritalStatus:         req.MaritalStatus,
		TaxStatus:             req.TaxStatus,
		KnownAssetClasses:     req.KnownAssetClasses,
		PastTransactions:      req.PastTransactions,
		FinancialEducation:    req.FinancialEducation,
		ProfessionalExperience: req.ProfessionalExperience,
		SelfAssessment:        req.SelfAssessment,
		MonthlyNetIncome:      req.MonthlyNetIncome,
		RecurringLiabilities:  req.RecurringLiabilities,
		LiquidAssets:          req.LiquidAssets,
		CurrentInvestments:    req.CurrentInvestments,
		RealEstate:            req.RealEstate,
		ExistingInsurance:     req.ExistingInsurance,
		MaxLossCapacityAbs:    req.MaxLossCapacityAbs,
		MaxLossCapacityPct:    req.MaxLossCapacityPct,
		InvestmentPurpose:     req.InvestmentPurpose,
		Horizon:               req.Horizon,
		RiskTolerance:         req.RiskTolerance,
		RiskCapacity:          req.RiskCapacity,
		RiskClass:             req.RiskClass,
		EsgPreference:         req.EsgPreference,
		EsgDetails:            req.EsgDetails,
		OneTimeAmount:         req.OneTimeAmount,
		MonthlySavings:        req.MonthlySavings,
		RecommendationSummary: req.RecommendationSummary,
		SuitabilityReasoning:  req.SuitabilityReasoning,
		GoalReference:         req.GoalReference,
		Alternatives:          req.Alternatives,
		NotRecommended:        req.NotRecommended,
		MainConcerns:          req.MainConcerns,
		WarningsGiven:         req.WarningsGiven,
		DocumentDelivered:     req.DocumentDelivered,
		DocumentDeliveredDate: req.DocumentDeliveredDate,
		DeliveryForm:          req.DeliveryForm,
		AdvisorSignature:      req.AdvisorSignature,
		CustomerConfirmation:  req.CustomerConfirmation,
		DocumentWaiver:        req.DocumentWaiver,
		FollowupDate:          req.FollowupDate,
		InternalNotes:         req.InternalNotes,
	}

	for _, p := range req.Products {
		protoPayload.Products = append(protoPayload.Products, &crmv1.AdvisoryProductProto{
			Id:            p.ID,
			Name:          p.Name,
			Isin:          p.ISIN,
			Category:      p.Category,
			RiskClass:     p.RiskClass,
			Opportunities: p.Opportunities,
			Risks:         p.Risks,
			CostsOneTime:  p.CostsOneTime,
			CostsRunning:  p.CostsRunning,
			Recommended:   p.Recommended,
		})
	}

	resp, err := client.UpdateAdvisoryProtocol(r.Context(), &crmv1.UpdateAdvisoryProtocolRequest{
		Id:       id,
		Protocol: protoPayload,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// HandleDeleteAdvisoryProtocol DELETE /api/v1/advisory-protocols/{id}
func (c *CRMRoutes) HandleDeleteAdvisoryProtocol(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteAdvisoryProtocol(r.Context(), &crmv1.DeleteAdvisoryProtocolRequest{Id: id})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleHandOverAdvisoryProtocol POST /api/v1/advisory-protocols/{id}/hand-over
func (c *CRMRoutes) HandleHandOverAdvisoryProtocol(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.HandOverAdvisoryProtocol(r.Context(), &crmv1.HandOverAdvisoryProtocolRequest{Id: id})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// HandleAdvisoryProtocolPDF GET /api/v1/advisory-protocols/{id}/pdf
// Returns a server-generated PDF binary for archival and printing.
// The PDF is generated on-demand from the stored protocol data.
func (c *CRMRoutes) HandleAdvisoryProtocolPDF(w http.ResponseWriter, r *http.Request) {
	// Note: PDF generation requires the CRM service to have the advisoryProtocolService
	// configured AND access to contact/company names. The gateway fetches the protocol
	// via gRPC, then requests the PDF binary from a dedicated gRPC streaming call.
	//
	// For now we return a stub that signals "not yet available in gateway" — the FE
	// uses window.print() (browser PDF) as its primary PDF path, which does NOT
	// require this endpoint. This endpoint is for server-side archiving only.
	//
	// TODO: wire GeneratePDF once a dedicated gRPC streaming RPC is added, or expose
	// via a dedicated PDF service. The CRM service already has the logic in
	// internal/crm/advisoryprotocol/service.go#GeneratePDF.
	response.Error(w, http.StatusNotImplemented, "server-side PDF generation: use browser print for now")
}

// HandleReferralReport GET /api/v1/contacts/referral-report
func (c *CRMRoutes) HandleReferralReport(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	resp, err := client.GetReferralReport(r.Context(), &crmv1.GetReferralReportRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}
