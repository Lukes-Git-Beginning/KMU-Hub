package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/crm/advisoryprotocol"
	"github.com/kmuhub/kmuhub/internal/middleware"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// ============================================================================
// Advisory Protocol gRPC handlers (P8 — Beratungsprotokolle)
// ============================================================================

func (s *CRMGRPCServer) CreateAdvisoryProtocol(ctx context.Context, req *crmv1.CreateAdvisoryProtocolRequest) (*crmv1.CreateAdvisoryProtocolResponse, error) {
	if s.advisoryProtocolService == nil {
		return nil, status.Error(codes.Unavailable, "advisory protocol service not configured")
	}
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}
	userID, err := uuid.Parse(middleware.GetUserID(ctx))
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "user_id missing from context")
	}
	contactID, err := uuid.Parse(req.ContactId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
	}

	p, err := s.advisoryProtocolService.Create(ctx, advisoryprotocol.CreateInput{
		TenantID:  tenantID,
		ContactID: contactID,
		CreatedBy: userID,
	})
	if err != nil {
		return nil, mapCRMError(err)
	}
	return &crmv1.CreateAdvisoryProtocolResponse{Protocol: toAdvisoryProto(p)}, nil
}

func (s *CRMGRPCServer) GetAdvisoryProtocol(ctx context.Context, req *crmv1.GetAdvisoryProtocolRequest) (*crmv1.GetAdvisoryProtocolResponse, error) {
	if s.advisoryProtocolService == nil {
		return nil, status.Error(codes.Unavailable, "advisory protocol service not configured")
	}
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	p, err := s.advisoryProtocolService.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}
	return &crmv1.GetAdvisoryProtocolResponse{Protocol: toAdvisoryProto(p)}, nil
}

func (s *CRMGRPCServer) ListAdvisoryProtocols(ctx context.Context, req *crmv1.ListAdvisoryProtocolsRequest) (*crmv1.ListAdvisoryProtocolsResponse, error) {
	if s.advisoryProtocolService == nil {
		return nil, status.Error(codes.Unavailable, "advisory protocol service not configured")
	}
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}
	contactID, err := uuid.Parse(req.ContactId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
	}

	protocols, err := s.advisoryProtocolService.ListByContact(ctx, contactID, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	var protoList []*crmv1.AdvisoryProtocolProto
	for _, p := range protocols {
		protoList = append(protoList, toAdvisoryProto(p))
	}
	return &crmv1.ListAdvisoryProtocolsResponse{Protocols: protoList}, nil
}

func (s *CRMGRPCServer) UpdateAdvisoryProtocol(ctx context.Context, req *crmv1.UpdateAdvisoryProtocolRequest) (*crmv1.UpdateAdvisoryProtocolResponse, error) {
	if s.advisoryProtocolService == nil {
		return nil, status.Error(codes.Unavailable, "advisory protocol service not configured")
	}
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}
	if req.Protocol == nil {
		return nil, status.Error(codes.InvalidArgument, "protocol payload required")
	}

	in := updateInputFromProto(req.Protocol)
	p, err := s.advisoryProtocolService.Update(ctx, id, tenantID, in)
	if err != nil {
		return nil, mapCRMError(err)
	}
	return &crmv1.UpdateAdvisoryProtocolResponse{Protocol: toAdvisoryProto(p)}, nil
}

func (s *CRMGRPCServer) DeleteAdvisoryProtocol(ctx context.Context, req *crmv1.DeleteAdvisoryProtocolRequest) (*crmv1.DeleteAdvisoryProtocolResponse, error) {
	if s.advisoryProtocolService == nil {
		return nil, status.Error(codes.Unavailable, "advisory protocol service not configured")
	}
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	if err := s.advisoryProtocolService.Delete(ctx, id, tenantID); err != nil {
		return nil, mapCRMError(err)
	}
	return &crmv1.DeleteAdvisoryProtocolResponse{Ok: true}, nil
}

func (s *CRMGRPCServer) HandOverAdvisoryProtocol(ctx context.Context, req *crmv1.HandOverAdvisoryProtocolRequest) (*crmv1.HandOverAdvisoryProtocolResponse, error) {
	if s.advisoryProtocolService == nil {
		return nil, status.Error(codes.Unavailable, "advisory protocol service not configured")
	}
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	p, err := s.advisoryProtocolService.HandOver(ctx, id, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}
	return &crmv1.HandOverAdvisoryProtocolResponse{Protocol: toAdvisoryProto(p)}, nil
}

// GenerateAdvisoryProtocolPDF renders a protocol as a server-side PDF
// (Geeignetheitserklärung) for archiving and printing. It reuses the existing
// maroto renderer in advisoryprotocol.Service.GeneratePDF — no new dependency —
// and resolves the party names from the contact record for the document header.
func (s *CRMGRPCServer) GenerateAdvisoryProtocolPDF(ctx context.Context, req *crmv1.GenerateAdvisoryProtocolPDFRequest) (*crmv1.GenerateAdvisoryProtocolPDFResponse, error) {
	if s.advisoryProtocolService == nil {
		return nil, status.Error(codes.Unavailable, "advisory protocol service not configured")
	}
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	p, err := s.advisoryProtocolService.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	// Resolve denormalized party names for the header. Best-effort: a missing or
	// unreadable contact must not block archiving of an already-finalized protocol.
	var contactName, companyName string
	if s.contactService != nil {
		if c, cErr := s.contactService.GetByID(ctx, p.ContactID, tenantID); cErr == nil {
			contactName = strings.TrimSpace(c.FirstName + " " + c.LastName)
			if c.CompanyName != nil {
				companyName = *c.CompanyName
			}
		}
	}

	pdfBytes, err := s.advisoryProtocolService.GeneratePDF(ctx, p, contactName, companyName)
	if err != nil {
		return nil, status.Error(codes.Internal, "generate advisory protocol pdf: "+err.Error())
	}

	return &crmv1.GenerateAdvisoryProtocolPDFResponse{
		PdfData:  pdfBytes,
		Filename: fmt.Sprintf("Beratungsprotokoll_%s.pdf", id.String()[:8]),
	}, nil
}

func (s *CRMGRPCServer) GetReferralReport(ctx context.Context, _ *crmv1.GetReferralReportRequest) (*crmv1.GetReferralReportResponse, error) {
	if s.advisoryProtocolService == nil {
		return nil, status.Error(codes.Unavailable, "advisory protocol service not configured")
	}
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	entries, err := s.advisoryProtocolService.GetReferralReport(ctx, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	var protoEntries []*crmv1.ReferralReportEntry
	for _, e := range entries {
		protoEntries = append(protoEntries, &crmv1.ReferralReportEntry{
			ReferrerId:         e.ReferrerID.String(),
			ReferrerFirstName:  e.ReferrerFirstName,
			ReferrerLastName:   e.ReferrerLastName,
			ReferredCount:      int32(e.ReferredCount),
		})
	}
	return &crmv1.GetReferralReportResponse{Entries: protoEntries}, nil
}

// ---------------------------------------------------------------------------
// Mapping helpers
// ---------------------------------------------------------------------------

func toAdvisoryProto(p *advisoryprotocol.Protocol) *crmv1.AdvisoryProtocolProto {
	proto := &crmv1.AdvisoryProtocolProto{
		Id:                    p.ID.String(),
		TenantId:              p.TenantID.String(),
		ContactId:             p.ContactID.String(),
		CreatedBy:             p.CreatedBy.String(),
		Status:                p.Status,
		TimeFrom:              p.TimeFrom,
		TimeTo:                p.TimeTo,
		Location:              p.Location,
		Advisor:               p.Advisor,
		Occasion:              p.Occasion,
		OccasionNote:          p.OccasionNote,
		CustomerCategory:      p.CustomerCategory,
		MaritalStatus:         p.MaritalStatus,
		TaxStatus:             p.TaxStatus,
		KnownAssetClasses:     p.KnownAssetClasses,
		PastTransactions:      p.PastTransactions,
		FinancialEducation:    p.FinancialEducation,
		ProfessionalExperience: p.ProfessionalExperience,
		RealEstate:            p.RealEstate,
		ExistingInsurance:     p.ExistingInsurance,
		InvestmentPurpose:     p.InvestmentPurpose,
		Horizon:               p.Horizon,
		RiskTolerance:         p.RiskTolerance,
		RiskCapacity:          p.RiskCapacity,
		RiskClass:             int32(p.RiskClass),
		EsgPreference:         p.EsgPreference,
		EsgDetails:            p.EsgDetails,
		RecommendationSummary: p.RecommendationSummary,
		SuitabilityReasoning:  p.SuitabilityReasoning,
		GoalReference:         p.GoalReference,
		Alternatives:          p.Alternatives,
		NotRecommended:        p.NotRecommended,
		MainConcerns:          p.MainConcerns,
		WarningsGiven:         p.WarningsGiven,
		DocumentDelivered:     p.DocumentDelivered,
		DeliveryForm:          p.DeliveryForm,
		AdvisorSignature:      p.AdvisorSignature,
		CustomerConfirmation:  p.CustomerConfirmation,
		DocumentWaiver:        p.DocumentWaiver,
		InternalNotes:         p.InternalNotes,
		CreatedAt:             p.CreatedAt.UTC().Format(http.TimeFormat),
		UpdatedAt:             p.UpdatedAt.UTC().Format(http.TimeFormat),
	}

	if p.HandedOverAt != nil {
		s := p.HandedOverAt.UTC().Format(http.TimeFormat)
		proto.HandedOverAt = &s
	}
	if p.Date != nil {
		proto.Date = p.Date
	}
	if p.BirthDate != nil {
		proto.BirthDate = p.BirthDate
	}
	if p.SelfAssessment != nil {
		v := int32(*p.SelfAssessment)
		proto.SelfAssessment = &v
	}
	if p.MonthlyNetIncome != nil {
		proto.MonthlyNetIncome = p.MonthlyNetIncome
	}
	if p.RecurringLiabilities != nil {
		proto.RecurringLiabilities = p.RecurringLiabilities
	}
	if p.LiquidAssets != nil {
		proto.LiquidAssets = p.LiquidAssets
	}
	if p.CurrentInvestments != nil {
		proto.CurrentInvestments = p.CurrentInvestments
	}
	if p.MaxLossCapacityAbs != nil {
		proto.MaxLossCapacityAbs = p.MaxLossCapacityAbs
	}
	if p.MaxLossCapacityPct != nil {
		proto.MaxLossCapacityPct = p.MaxLossCapacityPct
	}
	if p.OneTimeAmount != nil {
		proto.OneTimeAmount = p.OneTimeAmount
	}
	if p.MonthlySavings != nil {
		proto.MonthlySavings = p.MonthlySavings
	}
	if p.DocumentDeliveredDate != nil {
		proto.DocumentDeliveredDate = p.DocumentDeliveredDate
	}
	if p.FollowupDate != nil {
		proto.FollowupDate = p.FollowupDate
	}

	for _, prod := range p.Products {
		protoProd := &crmv1.AdvisoryProductProto{
			Id:          prod.ID,
			Name:        prod.Name,
			RiskClass:   int32(prod.RiskClass),
			Risks:       prod.Risks,
			Recommended: prod.Recommended,
		}
		if prod.ISIN != nil {
			protoProd.Isin = prod.ISIN
		}
		if prod.Category != nil {
			protoProd.Category = prod.Category
		}
		if prod.Opportunities != nil {
			protoProd.Opportunities = prod.Opportunities
		}
		if prod.CostsOneTime != nil {
			protoProd.CostsOneTime = prod.CostsOneTime
		}
		if prod.CostsRunning != nil {
			protoProd.CostsRunning = prod.CostsRunning
		}
		proto.Products = append(proto.Products, protoProd)
	}

	return proto
}

func updateInputFromProto(p *crmv1.AdvisoryProtocolProto) advisoryprotocol.UpdateInput {
	in := advisoryprotocol.UpdateInput{
		Date:                  p.Date,
		TimeFrom:              p.TimeFrom,
		TimeTo:                p.TimeTo,
		Location:              p.Location,
		Advisor:               p.Advisor,
		Occasion:              p.Occasion,
		OccasionNote:          p.OccasionNote,
		CustomerCategory:      p.CustomerCategory,
		BirthDate:             p.BirthDate,
		MaritalStatus:         p.MaritalStatus,
		TaxStatus:             p.TaxStatus,
		KnownAssetClasses:     p.KnownAssetClasses,
		PastTransactions:      p.PastTransactions,
		FinancialEducation:    p.FinancialEducation,
		ProfessionalExperience: p.ProfessionalExperience,
		MonthlyNetIncome:      p.MonthlyNetIncome,
		RecurringLiabilities:  p.RecurringLiabilities,
		LiquidAssets:          p.LiquidAssets,
		CurrentInvestments:    p.CurrentInvestments,
		RealEstate:            p.RealEstate,
		ExistingInsurance:     p.ExistingInsurance,
		MaxLossCapacityAbs:    p.MaxLossCapacityAbs,
		MaxLossCapacityPct:    p.MaxLossCapacityPct,
		InvestmentPurpose:     p.InvestmentPurpose,
		Horizon:               p.Horizon,
		RiskTolerance:         p.RiskTolerance,
		RiskCapacity:          p.RiskCapacity,
		RiskClass:             int(p.RiskClass),
		EsgPreference:         p.EsgPreference,
		EsgDetails:            p.EsgDetails,
		OneTimeAmount:         p.OneTimeAmount,
		MonthlySavings:        p.MonthlySavings,
		RecommendationSummary: p.RecommendationSummary,
		SuitabilityReasoning:  p.SuitabilityReasoning,
		GoalReference:         p.GoalReference,
		Alternatives:          p.Alternatives,
		NotRecommended:        p.NotRecommended,
		MainConcerns:          p.MainConcerns,
		WarningsGiven:         p.WarningsGiven,
		DocumentDelivered:     p.DocumentDelivered,
		DocumentDeliveredDate: p.DocumentDeliveredDate,
		DeliveryForm:          p.DeliveryForm,
		AdvisorSignature:      p.AdvisorSignature,
		CustomerConfirmation:  p.CustomerConfirmation,
		DocumentWaiver:        p.DocumentWaiver,
		FollowupDate:          p.FollowupDate,
		InternalNotes:         p.InternalNotes,
	}

	if p.SelfAssessment != nil {
		v := int(*p.SelfAssessment)
		in.SelfAssessment = &v
	}

	for _, prod := range p.Products {
		ap := advisoryprotocol.Product{
			ID:          prod.Id,
			Name:        prod.Name,
			RiskClass:   int(prod.RiskClass),
			Risks:       prod.Risks,
			Recommended: prod.Recommended,
			ISIN:        prod.Isin,
			Category:    prod.Category,
			Opportunities: prod.Opportunities,
			CostsOneTime: prod.CostsOneTime,
			CostsRunning: prod.CostsRunning,
		}
		in.Products = append(in.Products, ap)
	}
	if in.Products == nil {
		in.Products = []advisoryprotocol.Product{}
	}

	return in
}
