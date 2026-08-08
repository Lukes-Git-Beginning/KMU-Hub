package server

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/crm/contact"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// ============================================================================
// Lead gRPC handlers
// ============================================================================
//
// Leads are contacts at an earlier lifecycle stage (migration 000259), so
// these handlers route through contactService -- there is no lead service and
// no lead table.

func toLeadInfo(c *models.ContactWithRelations) *crmv1.LeadInfo {
	info := &crmv1.LeadInfo{
		Id:             c.ID.String(),
		FirstName:      c.FirstName,
		LastName:       c.LastName,
		LifecycleStage: c.LifecycleStage,
		CreatedAt:      c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// The list query resolves this via COALESCE(company.name, lead_company);
	// the create path returns the fresh row before that join runs, so fall
	// back to the free-text employer here rather than answering with "".
	switch {
	case c.CompanyName != nil:
		info.Company = *c.CompanyName
	case c.LeadCompany != nil:
		info.Company = *c.LeadCompany
	}
	if c.Email != nil {
		info.Email = *c.Email
	}
	if c.Phone != nil {
		info.Phone = *c.Phone
	}
	if c.Notes != nil {
		info.Notes = *c.Notes
	}
	if c.LeadSource != nil {
		info.Source = *c.LeadSource
	}
	if c.LeadScore != nil {
		info.Score = int32(*c.LeadScore)
	}
	if c.LeadStatus != nil {
		info.Status = *c.LeadStatus
	}
	if c.LeadTemperature != nil {
		override := *c.LeadTemperature
		info.TemperatureOverride = &override
	}
	// Effective temperature: the client should not have to re-derive it, and
	// deriving it in two places is how the two drift apart.
	info.Temperature = contact.EffectiveTemperature(c.LeadScore, c.LeadTemperature)

	return info
}

func (s *CRMGRPCServer) ListLeads(ctx context.Context, req *crmv1.ListLeadsRequest) (*crmv1.ListLeadsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	leads, total, err := s.contactService.ListLeads(ctx, contact.LeadListInput{
		TenantID: tenantID,
		Stage:    req.GetStage(),
		Status:   req.GetStatus(),
		Search:   req.GetSearch(),
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	})
	if err != nil {
		return nil, mapCRMError(err)
	}

	infos := make([]*crmv1.LeadInfo, 0, len(leads))
	for _, l := range leads {
		infos = append(infos, toLeadInfo(l))
	}

	return &crmv1.ListLeadsResponse{Leads: infos, Total: int32(total)}, nil
}

func (s *CRMGRPCServer) CreateLead(ctx context.Context, req *crmv1.CreateLeadRequest) (*crmv1.CreateLeadResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	createdBy, err := uuid.Parse(req.GetCreatedBy())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by")
	}

	created, err := s.contactService.CreateLead(ctx, contact.CreateLeadInput{
		FirstName: req.GetFirstName(),
		LastName:  req.GetLastName(),
		Company:   req.GetCompany(),
		Email:     req.GetEmail(),
		Phone:     req.GetPhone(),
		Notes:     req.GetNotes(),
		Source:    req.GetSource(),
		CreatedBy: createdBy,
		TenantID:  tenantID,
	})
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.CreateLeadResponse{Lead: toLeadInfo(created)}, nil
}

func (s *CRMGRPCServer) UpdateLead(ctx context.Context, req *crmv1.UpdateLeadRequest) (*crmv1.UpdateLeadResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	leadID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid lead id")
	}

	updated, err := s.contactService.UpdateLead(ctx, leadID, contact.UpdateLeadInput{
		Status:      req.Status,
		Temperature: req.Temperature,
	}, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.UpdateLeadResponse{Lead: toLeadInfo(updated)}, nil
}

func (s *CRMGRPCServer) ConvertLead(ctx context.Context, req *crmv1.ConvertLeadRequest) (*crmv1.ConvertLeadResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	leadID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid lead id")
	}

	converted, err := s.contactService.ConvertLead(ctx, leadID, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.ConvertLeadResponse{Lead: toLeadInfo(converted)}, nil
}

// PromoteContactToLead is called service-to-service by the dialer when a
// call outcome carries a callback request. It is not exposed over the HTTP
// gateway -- there is no product surface for a human to trigger this
// directly, only the dialer's outcome-logging path.
func (s *CRMGRPCServer) PromoteContactToLead(ctx context.Context, req *crmv1.PromoteContactToLeadRequest) (*crmv1.PromoteContactToLeadResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	contactID, err := uuid.Parse(req.GetContactId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
	}

	promoted, err := s.contactService.PromoteFromDialerCallback(ctx, contactID, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.PromoteContactToLeadResponse{Lead: toLeadInfo(promoted)}, nil
}
