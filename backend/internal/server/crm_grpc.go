package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/crm/activity"
	"github.com/kmuhub/kmuhub/internal/crm/advisoryprotocol"
	"github.com/kmuhub/kmuhub/internal/crm/company"
	"github.com/kmuhub/kmuhub/internal/crm/consent"
	"github.com/kmuhub/kmuhub/internal/crm/contact"
	"github.com/kmuhub/kmuhub/internal/crm/customfield"
	"github.com/kmuhub/kmuhub/internal/crm/deal"
	emailcontact "github.com/kmuhub/kmuhub/internal/email/contact"
	"github.com/kmuhub/kmuhub/internal/crm/pipelinestage"
	"github.com/kmuhub/kmuhub/internal/crm/report"
	"github.com/kmuhub/kmuhub/internal/crm/savedfilter"
	"github.com/kmuhub/kmuhub/internal/crm/search"
	"github.com/kmuhub/kmuhub/internal/crm/tag"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// CRMGRPCServer implements the CRM gRPC service
type CRMGRPCServer struct {
	crmv1.UnimplementedCRMServiceServer
	customFieldService      *customfield.Service
	tagService              *tag.Service
	contactService          *contact.Service
	companyService          *company.Service
	pipelineStageService    *pipelinestage.Service
	dealService             *deal.Service
	activityService         *activity.Service
	consentService          *consent.Service
	searchService           *search.Service
	savedFilterService      *savedfilter.Service
	reportService           *report.Service
	advisoryProtocolService *advisoryprotocol.Service
	importService           *emailcontact.ImportService
	exportService           *emailcontact.ExportService
	visibilityService       *emailcontact.VisibilityService
}

// NewCRMGRPCServer creates a new CRM gRPC server
func NewCRMGRPCServer(
	customFieldService *customfield.Service,
	tagService *tag.Service,
	contactService *contact.Service,
	companyService *company.Service,
	pipelineStageService *pipelinestage.Service,
	dealService *deal.Service,
	activityService *activity.Service,
	consentService *consent.Service,
	searchService *search.Service,
	savedFilterService *savedfilter.Service,
	reportService *report.Service,
) *CRMGRPCServer {
	return &CRMGRPCServer{
		customFieldService:   customFieldService,
		tagService:           tagService,
		contactService:       contactService,
		companyService:       companyService,
		pipelineStageService: pipelineStageService,
		dealService:          dealService,
		activityService:      activityService,
		consentService:       consentService,
		searchService:        searchService,
		savedFilterService:   savedFilterService,
		reportService:        reportService,
	}
}

// SetAdvisoryProtocolService wires in the advisory protocol service (called from cmd/crm/main.go).
func (s *CRMGRPCServer) SetAdvisoryProtocolService(svc *advisoryprotocol.Service) {
	s.advisoryProtocolService = svc
}

// SetImportExportServices sets the import/export/visibility services.
// Called after construction to avoid circular dependencies.
func (s *CRMGRPCServer) SetImportExportServices(
	importSvc *emailcontact.ImportService,
	exportSvc *emailcontact.ExportService,
	visibilitySvc *emailcontact.VisibilityService,
) {
	s.importService = importSvc
	s.exportService = exportSvc
	s.visibilityService = visibilitySvc
}

// ============================================================================
// Custom Fields
// ============================================================================

func (s *CRMGRPCServer) CreateCustomField(ctx context.Context, req *crmv1.CreateCustomFieldRequest) (*crmv1.CreateCustomFieldResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by user id")
	}

	input := customfield.CreateInput{
		TenantID:   tenantID,
		EntityType: models.EntityType(req.EntityType),
		FieldName:  req.FieldName,
		FieldLabel: req.FieldLabel,
		FieldType:  models.FieldType(req.FieldType),
		Options:    req.Options,
		IsRequired: req.IsRequired,
		SortOrder:  int(req.SortOrder),
		CreatedBy:  createdBy,
	}

	if req.DefaultValue != nil {
		input.DefaultValue = req.DefaultValue
	}

	field, err := s.customFieldService.Create(ctx, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.CreateCustomFieldResponse{
		CustomField: toCustomFieldInfo(field),
	}, nil
}

func (s *CRMGRPCServer) GetCustomField(ctx context.Context, req *crmv1.GetCustomFieldRequest) (*crmv1.GetCustomFieldResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid custom field id")
	}

	field, err := s.customFieldService.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.GetCustomFieldResponse{
		CustomField: toCustomFieldInfo(field),
	}, nil
}

func (s *CRMGRPCServer) ListCustomFields(ctx context.Context, req *crmv1.ListCustomFieldsRequest) (*crmv1.ListCustomFieldsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	input := customfield.ListInput{
		TenantID: tenantID,
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
	}

	if req.EntityType != "" {
		entityType := models.EntityType(req.EntityType)
		input.EntityType = &entityType
	}

	fields, total, err := s.customFieldService.List(ctx, input)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list custom fields")
	}

	var infos []*crmv1.CustomFieldInfo
	for _, f := range fields {
		infos = append(infos, toCustomFieldInfo(f))
	}

	return &crmv1.ListCustomFieldsResponse{
		CustomFields: infos,
		Total:        int32(total),
	}, nil
}

func (s *CRMGRPCServer) UpdateCustomField(ctx context.Context, req *crmv1.UpdateCustomFieldRequest) (*crmv1.UpdateCustomFieldResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid custom field id")
	}

	input := customfield.UpdateInput{
		TenantID: tenantID,
	}

	if req.FieldLabel != nil {
		input.FieldLabel = req.FieldLabel
	}
	if len(req.Options) > 0 {
		input.Options = req.Options
	}
	if req.DefaultValue != nil {
		input.DefaultValue = req.DefaultValue
	}
	if req.IsRequired != nil {
		input.IsRequired = req.IsRequired
	}
	if req.SortOrder != nil {
		sortOrder := int(*req.SortOrder)
		input.SortOrder = &sortOrder
	}

	field, err := s.customFieldService.Update(ctx, id, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.UpdateCustomFieldResponse{
		CustomField: toCustomFieldInfo(field),
	}, nil
}

func (s *CRMGRPCServer) DeleteCustomField(ctx context.Context, req *crmv1.DeleteCustomFieldRequest) (*crmv1.DeleteCustomFieldResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid custom field id")
	}

	if err := s.customFieldService.Delete(ctx, id, tenantID); err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.DeleteCustomFieldResponse{}, nil
}

// ============================================================================
// Tags
// ============================================================================

func (s *CRMGRPCServer) CreateTag(ctx context.Context, req *crmv1.CreateTagRequest) (*crmv1.CreateTagResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	input := tag.CreateInput{
		Name:       req.Name,
		Color:      req.Color,
		EntityType: models.EntityType(req.EntityType),
	}

	t, err := s.tagService.Create(ctx, tenantID, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.CreateTagResponse{
		Tag: toTagInfo(t),
	}, nil
}

func (s *CRMGRPCServer) GetTag(ctx context.Context, req *crmv1.GetTagRequest) (*crmv1.GetTagResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tag id")
	}

	t, err := s.tagService.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.GetTagResponse{
		Tag: toTagInfo(t),
	}, nil
}

func (s *CRMGRPCServer) ListTags(ctx context.Context, req *crmv1.ListTagsRequest) (*crmv1.ListTagsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	input := tag.ListInput{
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
	}

	if req.EntityType != "" {
		entityType := models.EntityType(req.EntityType)
		input.EntityType = &entityType
	}

	tags, total, err := s.tagService.List(ctx, tenantID, input)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list tags")
	}

	var infos []*crmv1.TagInfo
	for _, t := range tags {
		infos = append(infos, toTagInfo(t))
	}

	return &crmv1.ListTagsResponse{
		Tags:  infos,
		Total: int32(total),
	}, nil
}

func (s *CRMGRPCServer) UpdateTag(ctx context.Context, req *crmv1.UpdateTagRequest) (*crmv1.UpdateTagResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tag id")
	}

	input := tag.UpdateInput{}

	if req.Name != nil {
		input.Name = req.Name
	}
	if req.Color != nil {
		input.Color = req.Color
	}

	t, err := s.tagService.Update(ctx, id, tenantID, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.UpdateTagResponse{
		Tag: toTagInfo(t),
	}, nil
}

func (s *CRMGRPCServer) DeleteTag(ctx context.Context, req *crmv1.DeleteTagRequest) (*crmv1.DeleteTagResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tag id")
	}

	if err := s.tagService.Delete(ctx, id, tenantID); err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.DeleteTagResponse{}, nil
}

// ============================================================================
// Contacts
// ============================================================================

func (s *CRMGRPCServer) CreateContact(ctx context.Context, req *crmv1.CreateContactRequest) (*crmv1.CreateContactResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by user id")
	}

	input := contact.CreateInput{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		CreatedBy: createdBy,
		TenantID:  tenantID,
	}

	if req.Email != "" {
		input.Email = &req.Email
	}
	if req.Phone != "" {
		input.Phone = &req.Phone
	}
	if req.CompanyId != nil {
		companyID, parseErr := uuid.Parse(*req.CompanyId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid company_id")
		}
		input.CompanyID = &companyID
	}
	if req.Position != "" {
		input.Position = &req.Position
	}
	if req.Notes != "" {
		input.Notes = &req.Notes
	}

	// Parse tag IDs
	for _, tagIDStr := range req.TagIds {
		tagID, parseErr := uuid.Parse(tagIDStr)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid tag_id")
		}
		input.TagIDs = append(input.TagIDs, tagID)
	}

	// Parse custom field values
	if len(req.CustomFields) > 0 {
		input.CustomFields = make(map[uuid.UUID]any)
		for _, cf := range req.CustomFields {
			fieldID, parseErr := uuid.Parse(cf.FieldId)
			if parseErr != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid custom field_id")
			}
			var value any
			if unmarshalErr := json.Unmarshal([]byte(cf.Value), &value); unmarshalErr != nil {
				value = cf.Value // Use as string if not valid JSON
			}
			input.CustomFields[fieldID] = value
		}
	}

	c, err := s.contactService.Create(ctx, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.CreateContactResponse{
		Contact: toContactInfo(c),
	}, nil
}

func (s *CRMGRPCServer) GetContact(ctx context.Context, req *crmv1.GetContactRequest) (*crmv1.GetContactResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid contact id")
	}

	c, err := s.contactService.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.GetContactResponse{
		Contact: toContactInfo(c),
	}, nil
}

func (s *CRMGRPCServer) ListContacts(ctx context.Context, req *crmv1.ListContactsRequest) (*crmv1.ListContactsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	input := contact.ListInput{
		Search:           req.Search,
		Page:             int(req.Page),
		PageSize:         int(req.PageSize),
		SortBy:           req.SortBy,
		SortDesc:         req.SortDesc,
		VisibilityFilter: req.VisibilityFilter,
		TenantID:         tenantID,
	}

	if req.CompanyId != nil {
		companyID, err := uuid.Parse(*req.CompanyId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid company_id")
		}
		input.CompanyID = &companyID
	}

	for _, tagIDStr := range req.TagIds {
		tagID, err := uuid.Parse(tagIDStr)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid tag_id")
		}
		input.TagIDs = append(input.TagIDs, tagID)
	}

	// Use visibility-aware listing when user_id is provided
	if req.UserId != "" {
		userID, err := uuid.Parse(req.UserId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid user_id")
		}

		contacts, total, err := s.contactService.ListWithVisibility(ctx, userID, req.IsAdmin, input)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to list contacts")
		}

		var infos []*crmv1.ContactInfo
		for _, c := range contacts {
			infos = append(infos, toContactInfo(c))
		}

		return &crmv1.ListContactsResponse{
			Contacts: infos,
			Total:    int32(total),
		}, nil
	}

	// Fallback: no visibility filtering (backward compatible)
	contacts, total, err := s.contactService.List(ctx, input)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list contacts")
	}

	var infos []*crmv1.ContactInfo
	for _, c := range contacts {
		infos = append(infos, toContactInfo(c))
	}

	return &crmv1.ListContactsResponse{
		Contacts: infos,
		Total:    int32(total),
	}, nil
}

func (s *CRMGRPCServer) UpdateContact(ctx context.Context, req *crmv1.UpdateContactRequest) (*crmv1.UpdateContactResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid contact id")
	}

	input := contact.UpdateInput{}

	if req.FirstName != nil {
		input.FirstName = req.FirstName
	}
	if req.LastName != nil {
		input.LastName = req.LastName
	}
	if req.Email != nil {
		input.Email = req.Email
	}
	if req.Phone != nil {
		input.Phone = req.Phone
	}
	if req.CompanyId != nil {
		if *req.CompanyId == "" {
			nilUUID := uuid.Nil
			input.CompanyID = &nilUUID
		} else {
			companyID, parseErr := uuid.Parse(*req.CompanyId)
			if parseErr != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid company_id")
			}
			input.CompanyID = &companyID
		}
	}
	if req.Position != nil {
		input.Position = req.Position
	}
	if req.Notes != nil {
		input.Notes = req.Notes
	}

	// Parse custom field values
	if len(req.CustomFields) > 0 {
		input.CustomFields = make(map[uuid.UUID]any)
		for _, cf := range req.CustomFields {
			fieldID, parseErr := uuid.Parse(cf.FieldId)
			if parseErr != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid custom field_id")
			}
			var value any
			if unmarshalErr := json.Unmarshal([]byte(cf.Value), &value); unmarshalErr != nil {
				value = cf.Value
			}
			input.CustomFields[fieldID] = value
		}
	}

	c, err := s.contactService.Update(ctx, id, input, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.UpdateContactResponse{
		Contact: toContactInfo(c),
	}, nil
}

func (s *CRMGRPCServer) DeleteContact(ctx context.Context, req *crmv1.DeleteContactRequest) (*crmv1.DeleteContactResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid contact id")
	}

	if err := s.contactService.Delete(ctx, id, tenantID); err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.DeleteContactResponse{}, nil
}

func (s *CRMGRPCServer) AddContactTags(ctx context.Context, req *crmv1.AddContactTagsRequest) (*crmv1.AddContactTagsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	contactID, err := uuid.Parse(req.ContactId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
	}

	var tagIDs []uuid.UUID
	for _, tagIDStr := range req.TagIds {
		tagID, parseErr := uuid.Parse(tagIDStr)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid tag_id")
		}
		tagIDs = append(tagIDs, tagID)
	}

	c, err := s.contactService.AddTags(ctx, contactID, tagIDs, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.AddContactTagsResponse{
		Contact: toContactInfo(c),
	}, nil
}

func (s *CRMGRPCServer) RemoveContactTags(ctx context.Context, req *crmv1.RemoveContactTagsRequest) (*crmv1.RemoveContactTagsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	contactID, err := uuid.Parse(req.ContactId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
	}

	var tagIDs []uuid.UUID
	for _, tagIDStr := range req.TagIds {
		tagID, parseErr := uuid.Parse(tagIDStr)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid tag_id")
		}
		tagIDs = append(tagIDs, tagID)
	}

	c, err := s.contactService.RemoveTags(ctx, contactID, tagIDs, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.RemoveContactTagsResponse{
		Contact: toContactInfo(c),
	}, nil
}

// ============================================================================
// Companies
// ============================================================================

func (s *CRMGRPCServer) CreateCompany(ctx context.Context, req *crmv1.CreateCompanyRequest) (*crmv1.CreateCompanyResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by user id")
	}

	input := company.CreateInput{
		Name:      req.Name,
		CreatedBy: createdBy,
		TenantID:  tenantID,
	}

	if req.Domain != "" {
		input.Domain = &req.Domain
	}
	if req.Industry != "" {
		input.Industry = &req.Industry
	}
	if req.EmployeeCount != 0 {
		ec := int(req.EmployeeCount)
		input.EmployeeCount = &ec
	}
	if req.Address != "" {
		input.Address = &req.Address
	}
	if req.City != "" {
		input.City = &req.City
	}
	if req.Country != "" {
		input.Country = &req.Country
	}
	if req.Notes != "" {
		input.Notes = &req.Notes
	}

	// Parse tag IDs
	for _, tagIDStr := range req.TagIds {
		tagID, parseErr := uuid.Parse(tagIDStr)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid tag_id")
		}
		input.TagIDs = append(input.TagIDs, tagID)
	}

	// Parse custom field values
	if len(req.CustomFields) > 0 {
		input.CustomFields = make(map[uuid.UUID]any)
		for _, cf := range req.CustomFields {
			fieldID, parseErr := uuid.Parse(cf.FieldId)
			if parseErr != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid custom field_id")
			}
			var value any
			if unmarshalErr := json.Unmarshal([]byte(cf.Value), &value); unmarshalErr != nil {
				value = cf.Value
			}
			input.CustomFields[fieldID] = value
		}
	}

	c, err := s.companyService.Create(ctx, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.CreateCompanyResponse{
		Company: toCompanyInfo(c),
	}, nil
}

func (s *CRMGRPCServer) GetCompany(ctx context.Context, req *crmv1.GetCompanyRequest) (*crmv1.GetCompanyResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid company id")
	}

	c, err := s.companyService.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.GetCompanyResponse{
		Company: toCompanyInfo(c),
	}, nil
}

func (s *CRMGRPCServer) ListCompanies(ctx context.Context, req *crmv1.ListCompaniesRequest) (*crmv1.ListCompaniesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	input := company.ListInput{
		Search:   req.Search,
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
		SortBy:   req.SortBy,
		SortDesc: req.SortDesc,
		TenantID: tenantID,
	}

	if req.Industry != "" {
		input.Industry = &req.Industry
	}

	for _, tagIDStr := range req.TagIds {
		tagID, err := uuid.Parse(tagIDStr)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid tag_id")
		}
		input.TagIDs = append(input.TagIDs, tagID)
	}

	companies, total, err := s.companyService.List(ctx, input)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list companies")
	}

	var infos []*crmv1.CompanyInfo
	for _, c := range companies {
		infos = append(infos, toCompanyInfo(c))
	}

	return &crmv1.ListCompaniesResponse{
		Companies: infos,
		Total:     int32(total),
	}, nil
}

func (s *CRMGRPCServer) UpdateCompany(ctx context.Context, req *crmv1.UpdateCompanyRequest) (*crmv1.UpdateCompanyResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid company id")
	}

	input := company.UpdateInput{}

	if req.Name != nil {
		input.Name = req.Name
	}
	if req.Domain != nil {
		input.Domain = req.Domain
	}
	if req.Industry != nil {
		input.Industry = req.Industry
	}
	if req.EmployeeCount != nil {
		ec := int(*req.EmployeeCount)
		input.EmployeeCount = &ec
	}
	if req.Address != nil {
		input.Address = req.Address
	}
	if req.City != nil {
		input.City = req.City
	}
	if req.Country != nil {
		input.Country = req.Country
	}
	if req.Notes != nil {
		input.Notes = req.Notes
	}

	// Parse custom field values
	if len(req.CustomFields) > 0 {
		input.CustomFields = make(map[uuid.UUID]any)
		for _, cf := range req.CustomFields {
			fieldID, parseErr := uuid.Parse(cf.FieldId)
			if parseErr != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid custom field_id")
			}
			var value any
			if unmarshalErr := json.Unmarshal([]byte(cf.Value), &value); unmarshalErr != nil {
				value = cf.Value
			}
			input.CustomFields[fieldID] = value
		}
	}

	c, err := s.companyService.Update(ctx, id, input, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.UpdateCompanyResponse{
		Company: toCompanyInfo(c),
	}, nil
}

func (s *CRMGRPCServer) DeleteCompany(ctx context.Context, req *crmv1.DeleteCompanyRequest) (*crmv1.DeleteCompanyResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid company id")
	}

	if err := s.companyService.Delete(ctx, id, tenantID); err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.DeleteCompanyResponse{}, nil
}

func (s *CRMGRPCServer) GetCompanyContacts(ctx context.Context, req *crmv1.GetCompanyContactsRequest) (*crmv1.GetCompanyContactsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	companyID, err := uuid.Parse(req.CompanyId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid company_id")
	}

	input := contact.ListInput{
		CompanyID: &companyID,
		Page:      int(req.Page),
		PageSize:  int(req.PageSize),
		TenantID:  tenantID,
	}

	contacts, total, err := s.contactService.List(ctx, input)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list company contacts")
	}

	var infos []*crmv1.ContactInfo
	for _, c := range contacts {
		infos = append(infos, toContactInfo(c))
	}

	return &crmv1.GetCompanyContactsResponse{
		Contacts: infos,
		Total:    int32(total),
	}, nil
}

// ============================================================================
// Pipeline Stages
// ============================================================================

func (s *CRMGRPCServer) CreatePipelineStage(ctx context.Context, req *crmv1.CreatePipelineStageRequest) (*crmv1.CreatePipelineStageResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	input := pipelinestage.CreateInput{
		TenantID:    tenantID,
		Name:        req.Name,
		Color:       req.Color,
		IsWon:       req.IsWon,
		IsLost:      req.IsLost,
		Probability: req.Probability,
	}

	stage, err := s.pipelineStageService.Create(ctx, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.CreatePipelineStageResponse{
		Stage: toPipelineStageInfo(stage, 0, 0),
	}, nil
}

func (s *CRMGRPCServer) GetPipelineStage(ctx context.Context, req *crmv1.GetPipelineStageRequest) (*crmv1.GetPipelineStageResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid pipeline stage id")
	}

	stage, err := s.pipelineStageService.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.GetPipelineStageResponse{
		Stage: toPipelineStageInfo(stage, 0, 0),
	}, nil
}

func (s *CRMGRPCServer) ListPipelineStages(ctx context.Context, req *crmv1.ListPipelineStagesRequest) (*crmv1.ListPipelineStagesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	stages, err := s.pipelineStageService.ListWithStats(ctx, tenantID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list pipeline stages")
	}

	var infos []*crmv1.PipelineStageInfo
	for _, stage := range stages {
		infos = append(infos, toPipelineStageInfoWithStats(stage))
	}

	return &crmv1.ListPipelineStagesResponse{
		Stages: infos,
	}, nil
}

func (s *CRMGRPCServer) UpdatePipelineStage(ctx context.Context, req *crmv1.UpdatePipelineStageRequest) (*crmv1.UpdatePipelineStageResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid pipeline stage id")
	}

	input := pipelinestage.UpdateInput{}

	if req.Name != nil {
		input.Name = req.Name
	}
	if req.Color != nil {
		input.Color = req.Color
	}
	if req.IsWon != nil {
		input.IsWon = req.IsWon
	}
	if req.IsLost != nil {
		input.IsLost = req.IsLost
	}
	if req.Probability != nil {
		input.Probability = req.Probability
	}

	stage, err := s.pipelineStageService.Update(ctx, id, tenantID, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.UpdatePipelineStageResponse{
		Stage: toPipelineStageInfo(stage, 0, 0),
	}, nil
}

func (s *CRMGRPCServer) DeletePipelineStage(ctx context.Context, req *crmv1.DeletePipelineStageRequest) (*crmv1.DeletePipelineStageResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid pipeline stage id")
	}

	if err := s.pipelineStageService.Delete(ctx, id, tenantID); err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.DeletePipelineStageResponse{}, nil
}

func (s *CRMGRPCServer) ReorderPipelineStages(ctx context.Context, req *crmv1.ReorderPipelineStagesRequest) (*crmv1.ReorderPipelineStagesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	var stageIDs []uuid.UUID
	for _, idStr := range req.StageIds {
		id, parseErr := uuid.Parse(idStr)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid stage id in list")
		}
		stageIDs = append(stageIDs, id)
	}

	if err := s.pipelineStageService.Reorder(ctx, tenantID, stageIDs); err != nil {
		return nil, mapCRMError(err)
	}

	// Return updated list
	stages, err := s.pipelineStageService.ListWithStats(ctx, tenantID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list pipeline stages")
	}

	var infos []*crmv1.PipelineStageInfo
	for _, stage := range stages {
		infos = append(infos, toPipelineStageInfoWithStats(stage))
	}

	return &crmv1.ReorderPipelineStagesResponse{
		Stages: infos,
	}, nil
}

// ============================================================================
// Deals
// ============================================================================

func (s *CRMGRPCServer) CreateDeal(ctx context.Context, req *crmv1.CreateDealRequest) (*crmv1.CreateDealResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by user id")
	}

	stageID, err := uuid.Parse(req.StageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid stage_id")
	}

	input := deal.CreateInput{
		TenantID:  tenantID,
		Name:      req.Name,
		Value:     req.Value,
		Currency:  req.Currency,
		StageID:   stageID,
		CreatedBy: createdBy,
	}

	if req.ContactId != nil {
		contactID, parseErr := uuid.Parse(*req.ContactId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
		}
		input.ContactID = &contactID
	}

	if req.CompanyId != nil {
		companyID, parseErr := uuid.Parse(*req.CompanyId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid company_id")
		}
		input.CompanyID = &companyID
	}

	if req.OwnerId != nil {
		ownerID, parseErr := uuid.Parse(*req.OwnerId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid owner_id")
		}
		input.OwnerID = &ownerID
	}

	if req.ExpectedCloseDate != "" {
		t, parseErr := parseDate(req.ExpectedCloseDate)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid expected_close_date format")
		}
		input.ExpectedCloseDate = &t
	}

	if req.Notes != "" {
		input.Notes = &req.Notes
	}

	// Parse tag IDs
	for _, tagIDStr := range req.TagIds {
		tagID, parseErr := uuid.Parse(tagIDStr)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid tag_id")
		}
		input.TagIDs = append(input.TagIDs, tagID)
	}

	// Parse custom field values
	if len(req.CustomFields) > 0 {
		input.CustomFields = make(map[uuid.UUID]any)
		for _, cf := range req.CustomFields {
			fieldID, parseErr := uuid.Parse(cf.FieldId)
			if parseErr != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid custom field_id")
			}
			var value any
			if unmarshalErr := json.Unmarshal([]byte(cf.Value), &value); unmarshalErr != nil {
				value = cf.Value
			}
			input.CustomFields[fieldID] = value
		}
	}

	d, err := s.dealService.Create(ctx, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.CreateDealResponse{
		Deal: toDealInfo(d),
	}, nil
}

func (s *CRMGRPCServer) GetDeal(ctx context.Context, req *crmv1.GetDealRequest) (*crmv1.GetDealResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid deal id")
	}

	d, err := s.dealService.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.GetDealResponse{
		Deal: toDealInfo(d),
	}, nil
}

func (s *CRMGRPCServer) ListDeals(ctx context.Context, req *crmv1.ListDealsRequest) (*crmv1.ListDealsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	input := deal.ListInput{
		TenantID: tenantID,
		Search:   req.Search,
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
		SortBy:   req.SortBy,
		SortDesc: req.SortDesc,
	}

	if req.StageId != nil {
		stageID, err := uuid.Parse(*req.StageId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid stage_id")
		}
		input.StageID = &stageID
	}

	if req.ContactId != nil {
		contactID, err := uuid.Parse(*req.ContactId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
		}
		input.ContactID = &contactID
	}

	if req.CompanyId != nil {
		companyID, err := uuid.Parse(*req.CompanyId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid company_id")
		}
		input.CompanyID = &companyID
	}

	if req.OwnerId != nil {
		ownerID, err := uuid.Parse(*req.OwnerId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid owner_id")
		}
		input.OwnerID = &ownerID
	}

	for _, tagIDStr := range req.TagIds {
		tagID, err := uuid.Parse(tagIDStr)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid tag_id")
		}
		input.TagIDs = append(input.TagIDs, tagID)
	}

	deals, total, err := s.dealService.List(ctx, input)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list deals")
	}

	var infos []*crmv1.DealInfo
	for _, d := range deals {
		infos = append(infos, toDealInfo(d))
	}

	return &crmv1.ListDealsResponse{
		Deals: infos,
		Total: int32(total),
	}, nil
}

func (s *CRMGRPCServer) UpdateDeal(ctx context.Context, req *crmv1.UpdateDealRequest) (*crmv1.UpdateDealResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid deal id")
	}

	input := deal.UpdateInput{}

	if req.Name != nil {
		input.Name = req.Name
	}
	if req.Value != nil {
		input.Value = req.Value
	}
	if req.Currency != nil {
		input.Currency = req.Currency
	}
	if req.ContactId != nil {
		if *req.ContactId == "" {
			nilUUID := uuid.Nil
			input.ContactID = &nilUUID
		} else {
			contactID, parseErr := uuid.Parse(*req.ContactId)
			if parseErr != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
			}
			input.ContactID = &contactID
		}
	}
	if req.CompanyId != nil {
		if *req.CompanyId == "" {
			nilUUID := uuid.Nil
			input.CompanyID = &nilUUID
		} else {
			companyID, parseErr := uuid.Parse(*req.CompanyId)
			if parseErr != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid company_id")
			}
			input.CompanyID = &companyID
		}
	}
	if req.OwnerId != nil {
		if *req.OwnerId == "" {
			nilUUID := uuid.Nil
			input.OwnerID = &nilUUID
		} else {
			ownerID, parseErr := uuid.Parse(*req.OwnerId)
			if parseErr != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid owner_id")
			}
			input.OwnerID = &ownerID
		}
	}
	if req.ExpectedCloseDate != nil {
		if *req.ExpectedCloseDate == "" {
			// Clear the date
			input.ExpectedCloseDate = nil
		} else {
			t, parseErr := parseDate(*req.ExpectedCloseDate)
			if parseErr != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid expected_close_date format")
			}
			input.ExpectedCloseDate = &t
		}
	}
	if req.Notes != nil {
		input.Notes = req.Notes
	}

	// Parse custom field values
	if len(req.CustomFields) > 0 {
		input.CustomFields = make(map[uuid.UUID]any)
		for _, cf := range req.CustomFields {
			fieldID, parseErr := uuid.Parse(cf.FieldId)
			if parseErr != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid custom field_id")
			}
			var value any
			if unmarshalErr := json.Unmarshal([]byte(cf.Value), &value); unmarshalErr != nil {
				value = cf.Value
			}
			input.CustomFields[fieldID] = value
		}
	}

	d, err := s.dealService.Update(ctx, tenantID, id, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.UpdateDealResponse{
		Deal: toDealInfo(d),
	}, nil
}

func (s *CRMGRPCServer) DeleteDeal(ctx context.Context, req *crmv1.DeleteDealRequest) (*crmv1.DeleteDealResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid deal id")
	}

	if err := s.dealService.Delete(ctx, tenantID, id); err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.DeleteDealResponse{}, nil
}

func (s *CRMGRPCServer) MoveDealToStage(ctx context.Context, req *crmv1.MoveDealToStageRequest) (*crmv1.MoveDealToStageResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	dealID, err := uuid.Parse(req.DealId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid deal_id")
	}

	stageID, err := uuid.Parse(req.StageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid stage_id")
	}

	d, err := s.dealService.MoveToStage(ctx, tenantID, dealID, stageID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.MoveDealToStageResponse{
		Deal: toDealInfo(d),
	}, nil
}

// ============================================================================
// Activities
// ============================================================================

func (s *CRMGRPCServer) CreateActivity(ctx context.Context, req *crmv1.CreateActivityRequest) (*crmv1.CreateActivityResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by user id")
	}

	input := activity.CreateInput{
		TenantID:     tenantID,
		ActivityType: models.ActivityType(req.ActivityType),
		Subject:      req.Subject,
		CreatedBy:    createdBy,
	}

	if req.Description != "" {
		input.Description = &req.Description
	}
	if req.ContactId != nil {
		contactID, parseErr := uuid.Parse(*req.ContactId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
		}
		input.ContactID = &contactID
	}
	if req.CompanyId != nil {
		companyID, parseErr := uuid.Parse(*req.CompanyId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid company_id")
		}
		input.CompanyID = &companyID
	}
	if req.DealId != nil {
		dealID, parseErr := uuid.Parse(*req.DealId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid deal_id")
		}
		input.DealID = &dealID
	}
	if req.AssignedTo != nil {
		assignedTo, parseErr := uuid.Parse(*req.AssignedTo)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid assigned_to")
		}
		input.AssignedTo = &assignedTo
	}
	if req.DueDate != "" {
		t, parseErr := time.Parse(time.RFC3339, req.DueDate)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid due_date format")
		}
		input.DueDate = &t
	}

	a, err := s.activityService.Create(ctx, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.CreateActivityResponse{
		Activity: toActivityInfo(a),
	}, nil
}

func (s *CRMGRPCServer) GetActivity(ctx context.Context, req *crmv1.GetActivityRequest) (*crmv1.GetActivityResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid activity id")
	}

	a, err := s.activityService.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.GetActivityResponse{
		Activity: toActivityInfo(a),
	}, nil
}

func (s *CRMGRPCServer) ListActivities(ctx context.Context, req *crmv1.ListActivitiesRequest) (*crmv1.ListActivitiesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	input := activity.ListInput{
		TenantID: tenantID,
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
		SortBy:   req.SortBy,
		SortDesc: req.SortDesc,
	}

	if req.ActivityType != nil && *req.ActivityType != "" {
		actType := models.ActivityType(*req.ActivityType)
		input.ActivityType = &actType
	}
	if req.ContactId != nil {
		contactID, err := uuid.Parse(*req.ContactId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
		}
		input.ContactID = &contactID
	}
	if req.CompanyId != nil {
		companyID, err := uuid.Parse(*req.CompanyId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid company_id")
		}
		input.CompanyID = &companyID
	}
	if req.DealId != nil {
		dealID, err := uuid.Parse(*req.DealId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid deal_id")
		}
		input.DealID = &dealID
	}
	if req.AssignedTo != nil {
		assignedTo, err := uuid.Parse(*req.AssignedTo)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid assigned_to")
		}
		input.AssignedTo = &assignedTo
	}
	if req.IsCompleted != nil {
		input.IsCompleted = req.IsCompleted
	}

	activities, total, err := s.activityService.List(ctx, input)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list activities")
	}

	var infos []*crmv1.ActivityInfo
	for _, a := range activities {
		infos = append(infos, toActivityInfo(a))
	}

	return &crmv1.ListActivitiesResponse{
		Activities: infos,
		Total:      int32(total),
	}, nil
}

func (s *CRMGRPCServer) UpdateActivity(ctx context.Context, req *crmv1.UpdateActivityRequest) (*crmv1.UpdateActivityResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid activity id")
	}

	input := activity.UpdateInput{}

	if req.Subject != nil {
		input.Subject = req.Subject
	}
	if req.Description != nil {
		input.Description = req.Description
	}
	if req.ContactId != nil {
		if *req.ContactId == "" {
			nilUUID := uuid.Nil
			input.ContactID = &nilUUID
		} else {
			contactID, parseErr := uuid.Parse(*req.ContactId)
			if parseErr != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
			}
			input.ContactID = &contactID
		}
	}
	if req.CompanyId != nil {
		if *req.CompanyId == "" {
			nilUUID := uuid.Nil
			input.CompanyID = &nilUUID
		} else {
			companyID, parseErr := uuid.Parse(*req.CompanyId)
			if parseErr != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid company_id")
			}
			input.CompanyID = &companyID
		}
	}
	if req.DealId != nil {
		if *req.DealId == "" {
			nilUUID := uuid.Nil
			input.DealID = &nilUUID
		} else {
			dealID, parseErr := uuid.Parse(*req.DealId)
			if parseErr != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid deal_id")
			}
			input.DealID = &dealID
		}
	}
	if req.AssignedTo != nil {
		if *req.AssignedTo == "" {
			nilUUID := uuid.Nil
			input.AssignedTo = &nilUUID
		} else {
			assignedTo, parseErr := uuid.Parse(*req.AssignedTo)
			if parseErr != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid assigned_to")
			}
			input.AssignedTo = &assignedTo
		}
	}
	if req.DueDate != nil {
		if *req.DueDate == "" {
			input.DueDate = nil
		} else {
			t, parseErr := time.Parse(time.RFC3339, *req.DueDate)
			if parseErr != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid due_date format")
			}
			input.DueDate = &t
		}
	}

	a, err := s.activityService.Update(ctx, tenantID, id, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.UpdateActivityResponse{
		Activity: toActivityInfo(a),
	}, nil
}

func (s *CRMGRPCServer) DeleteActivity(ctx context.Context, req *crmv1.DeleteActivityRequest) (*crmv1.DeleteActivityResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid activity id")
	}

	if err := s.activityService.Delete(ctx, tenantID, id); err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.DeleteActivityResponse{}, nil
}

func (s *CRMGRPCServer) CompleteActivity(ctx context.Context, req *crmv1.CompleteActivityRequest) (*crmv1.CompleteActivityResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid activity id")
	}

	a, err := s.activityService.Complete(ctx, tenantID, id)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.CompleteActivityResponse{
		Activity: toActivityInfo(a),
	}, nil
}

// ============================================================================
// Search
// ============================================================================

func (s *CRMGRPCServer) Search(ctx context.Context, req *crmv1.SearchRequest) (*crmv1.SearchResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	input := search.SearchInput{
		Query:    req.Query,
		Limit:    int(req.Limit),
		TenantID: tenantID,
	}

	for _, et := range req.EntityTypes {
		input.EntityTypes = append(input.EntityTypes, models.SearchEntityType(et))
	}

	results, err := s.searchService.Search(ctx, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	var infos []*crmv1.SearchResult
	for _, r := range results {
		infos = append(infos, toSearchResultInfo(r))
	}

	return &crmv1.SearchResponse{
		Results: infos,
	}, nil
}

// ============================================================================
// Saved Filters
// ============================================================================

func (s *CRMGRPCServer) CreateSavedFilter(ctx context.Context, req *crmv1.CreateSavedFilterRequest) (*crmv1.CreateSavedFilterResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by user id")
	}

	input := savedfilter.CreateInput{
		TenantID:   tenantID,
		Name:       req.Name,
		EntityType: req.EntityType,
		FilterJSON: req.FilterJson,
		IsDefault:  req.IsDefault,
		CreatedBy:  createdBy,
	}

	filter, err := s.savedFilterService.Create(ctx, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.CreateSavedFilterResponse{
		Filter: toSavedFilterInfo(filter),
	}, nil
}

func (s *CRMGRPCServer) GetSavedFilter(ctx context.Context, req *crmv1.GetSavedFilterRequest) (*crmv1.GetSavedFilterResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid saved filter id")
	}

	filter, err := s.savedFilterService.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.GetSavedFilterResponse{
		Filter: toSavedFilterInfo(filter),
	}, nil
}

func (s *CRMGRPCServer) ListSavedFilters(ctx context.Context, req *crmv1.ListSavedFiltersRequest) (*crmv1.ListSavedFiltersResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	input := savedfilter.ListInput{
		TenantID: tenantID,
	}

	if req.EntityType != "" {
		input.EntityType = &req.EntityType
	}
	if req.UserId != "" {
		userID, err := uuid.Parse(req.UserId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid user_id")
		}
		input.UserID = &userID
	}

	filters, err := s.savedFilterService.List(ctx, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	var infos []*crmv1.SavedFilterInfo
	for _, f := range filters {
		infos = append(infos, toSavedFilterInfo(f))
	}

	return &crmv1.ListSavedFiltersResponse{
		Filters: infos,
	}, nil
}

func (s *CRMGRPCServer) UpdateSavedFilter(ctx context.Context, req *crmv1.UpdateSavedFilterRequest) (*crmv1.UpdateSavedFilterResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid saved filter id")
	}

	// Get the filter first to extract the user ID (already tenant-scoped)
	existingFilter, err := s.savedFilterService.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	input := savedfilter.UpdateInput{}

	if req.Name != nil {
		input.Name = req.Name
	}
	if req.FilterJson != nil {
		input.FilterJSON = req.FilterJson
	}
	if req.IsDefault != nil {
		input.IsDefault = req.IsDefault
	}

	filter, err := s.savedFilterService.Update(ctx, id, tenantID, existingFilter.CreatedBy, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.UpdateSavedFilterResponse{
		Filter: toSavedFilterInfo(filter),
	}, nil
}

func (s *CRMGRPCServer) DeleteSavedFilter(ctx context.Context, req *crmv1.DeleteSavedFilterRequest) (*crmv1.DeleteSavedFilterResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid saved filter id")
	}

	// Get the filter first to extract the user ID (already tenant-scoped)
	existingFilter, err := s.savedFilterService.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	if err := s.savedFilterService.Delete(ctx, id, tenantID, existingFilter.CreatedBy); err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.DeleteSavedFilterResponse{}, nil
}

// ============================================================================
// Reports
// ============================================================================

func (s *CRMGRPCServer) GetPipelineReport(ctx context.Context, req *crmv1.GetPipelineReportRequest) (*crmv1.GetPipelineReportResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant context")
	}

	startDate, err := parseDate(req.StartDate)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid start_date format")
	}

	endDate, err := parseDate(req.EndDate)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid end_date format")
	}

	input := report.PipelineReportInput{
		TenantID:  tenantID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	if req.OwnerId != nil && *req.OwnerId != "" {
		ownerID, parseErr := uuid.Parse(*req.OwnerId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid owner_id")
		}
		input.OwnerID = &ownerID
	}

	r, err := s.reportService.GetPipelineReport(ctx, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	var stages []*crmv1.PipelineStageValue
	for _, stage := range r.Stages {
		stages = append(stages, &crmv1.PipelineStageValue{
			StageId:       stage.StageID.String(),
			StageName:     stage.StageName,
			DealCount:     stage.DealCount,
			TotalValue:    stage.TotalValue.InexactFloat64(),
			WeightedValue: stage.WeightedValue.InexactFloat64(),
		})
	}

	return &crmv1.GetPipelineReportResponse{
		Stages:             stages,
		TotalDeals:         r.TotalDeals,
		TotalValue:         r.TotalValue.InexactFloat64(),
		TotalWeightedValue: r.TotalWeightedValue.InexactFloat64(),
	}, nil
}

func (s *CRMGRPCServer) GetConversionReport(ctx context.Context, req *crmv1.GetConversionReportRequest) (*crmv1.GetConversionReportResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant context")
	}

	startDate, err := parseDate(req.StartDate)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid start_date format")
	}

	endDate, err := parseDate(req.EndDate)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid end_date format")
	}

	input := report.ConversionReportInput{
		TenantID:  tenantID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	r, err := s.reportService.GetConversionReport(ctx, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	var metrics []*crmv1.ConversionMetric
	for _, m := range r.Metrics {
		metrics = append(metrics, &crmv1.ConversionMetric{
			FromStage:      m.FromStage,
			ToStage:        m.ToStage,
			ConvertedCount: m.ConvertedCount,
			ConversionRate: m.ConversionRate,
			AverageDays:    m.AverageDays,
		})
	}

	return &crmv1.GetConversionReportResponse{
		Metrics:          metrics,
		OverallWinRate:   r.OverallWinRate,
		AverageDealCycle: r.AverageDealCycle,
	}, nil
}

func (s *CRMGRPCServer) GetActivityReport(ctx context.Context, req *crmv1.GetActivityReportRequest) (*crmv1.GetActivityReportResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant context")
	}

	startDate, err := parseDate(req.StartDate)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid start_date format")
	}

	endDate, err := parseDate(req.EndDate)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid end_date format")
	}

	input := report.ActivityReportInput{
		TenantID:  tenantID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	if req.UserId != nil && *req.UserId != "" {
		userID, parseErr := uuid.Parse(*req.UserId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid user_id")
		}
		input.UserID = &userID
	}

	r, err := s.reportService.GetActivityReport(ctx, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	var metrics []*crmv1.ActivityMetric
	for _, m := range r.Metrics {
		metrics = append(metrics, &crmv1.ActivityMetric{
			ActivityType:   m.ActivityType,
			TotalCount:     m.TotalCount,
			CompletedCount: m.CompletedCount,
			OverdueCount:   m.OverdueCount,
		})
	}

	return &crmv1.GetActivityReportResponse{
		Metrics:         metrics,
		TotalActivities: r.TotalActivities,
		CompletionRate:  r.CompletionRate,
	}, nil
}

// ============================================================================
// Contact Import/Export
// ============================================================================

func (s *CRMGRPCServer) ImportContactsCSV(ctx context.Context, req *crmv1.ImportContactsCSVRequest) (*crmv1.ImportContactsResponse, error) {
	if s.importService == nil {
		return nil, status.Error(codes.Unimplemented, "import service not configured")
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	// Wrap the contact service with a tenant-scoped adapter so the import
	// provider is bound to the caller's tenant for this request.
	provider := emailcontact.NewTenantScopedAdapter(s.contactService, s.companyService, tenantID)
	importSvc := emailcontact.NewImportService(provider, nil)

	result, err := importSvc.ImportCSV(ctx, bytes.NewReader(req.FileContent), req.FieldMapping, req.Visibility, userID, req.MergeByEmail)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "import failed: %v", err)
	}

	return toImportResponse(result), nil
}

func (s *CRMGRPCServer) ImportContactsVCard(ctx context.Context, req *crmv1.ImportContactsVCardRequest) (*crmv1.ImportContactsResponse, error) {
	if s.importService == nil {
		return nil, status.Error(codes.Unimplemented, "import service not configured")
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	provider := emailcontact.NewTenantScopedAdapter(s.contactService, s.companyService, tenantID)
	importSvc := emailcontact.NewImportService(provider, nil)

	result, err := importSvc.ImportVCard(ctx, bytes.NewReader(req.FileContent), req.Visibility, userID, req.MergeByEmail)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "import failed: %v", err)
	}

	return toImportResponse(result), nil
}

func (s *CRMGRPCServer) ImportContactsXLSX(ctx context.Context, req *crmv1.ImportContactsXLSXRequest) (*crmv1.ImportContactsResponse, error) {
	if s.importService == nil {
		return nil, status.Error(codes.Unimplemented, "import service not configured")
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	provider := emailcontact.NewTenantScopedAdapter(s.contactService, s.companyService, tenantID)
	importSvc := emailcontact.NewImportService(provider, nil)

	result, err := importSvc.ImportXLSX(ctx, bytes.NewReader(req.FileContent), req.FieldMapping, req.Visibility, userID, req.MergeByEmail)
	if err != nil {
		if errors.Is(err, emailcontact.ErrInvalidXLSX) {
			return nil, status.Errorf(codes.InvalidArgument, "invalid xlsx file: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "import failed: %v", err)
	}

	return toImportResponse(result), nil
}

func (s *CRMGRPCServer) ExportContactsCSV(ctx context.Context, req *crmv1.ExportContactsCSVRequest) (*crmv1.ExportContactsResponse, error) {
	if s.exportService == nil {
		return nil, status.Error(codes.Unimplemented, "export service not configured")
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	var ids []uuid.UUID
	for _, idStr := range req.ContactIds {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
		}
		ids = append(ids, id)
	}

	provider := emailcontact.NewTenantScopedAdapter(s.contactService, s.companyService, tenantID)
	exportSvc := emailcontact.NewExportService(provider, nil)

	var data []byte
	var exportErr error

	if len(ids) > 0 {
		data, exportErr = exportSvc.ExportCSV(ctx, ids, req.Fields)
	} else {
		userID, err := uuid.Parse(req.UserId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid user_id")
		}
		data, exportErr = exportSvc.ExportAllCSV(ctx, userID, req.Fields, req.IsAdmin)
	}

	if exportErr != nil {
		return nil, status.Errorf(codes.Internal, "export failed: %v", exportErr)
	}

	return &crmv1.ExportContactsResponse{
		FileContent: data,
		ContentType: "text/csv",
		Filename:    "kontakte.csv",
	}, nil
}

func (s *CRMGRPCServer) ExportContactsVCard(ctx context.Context, req *crmv1.ExportContactsVCardRequest) (*crmv1.ExportContactsResponse, error) {
	if s.exportService == nil {
		return nil, status.Error(codes.Unimplemented, "export service not configured")
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	var ids []uuid.UUID
	for _, idStr := range req.ContactIds {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
		}
		ids = append(ids, id)
	}

	provider := emailcontact.NewTenantScopedAdapter(s.contactService, s.companyService, tenantID)
	exportSvc := emailcontact.NewExportService(provider, nil)

	data, err := exportSvc.ExportVCard(ctx, ids)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "export failed: %v", err)
	}

	return &crmv1.ExportContactsResponse{
		FileContent: data,
		ContentType: "text/vcard",
		Filename:    "kontakte.vcf",
	}, nil
}

func (s *CRMGRPCServer) PreviewImportCSV(ctx context.Context, req *crmv1.PreviewImportCSVRequest) (*crmv1.PreviewImportCSVResponse, error) {
	if s.importService == nil {
		return nil, status.Error(codes.Unimplemented, "import service not configured")
	}

	preview, err := s.importService.PreviewCSV(ctx, bytes.NewReader(req.FileContent))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "preview failed: %v", err)
	}

	var sampleRows []*crmv1.CSVSampleRow
	for _, row := range preview.SampleRows {
		sampleRows = append(sampleRows, &crmv1.CSVSampleRow{Values: row})
	}

	return &crmv1.PreviewImportCSVResponse{
		Columns:         preview.Columns,
		SampleRows:      sampleRows,
		DetectedMapping: preview.DetectedMapping,
	}, nil
}

func (s *CRMGRPCServer) UpdateContactVisibility(ctx context.Context, req *crmv1.UpdateContactVisibilityRequest) (*crmv1.UpdateContactVisibilityResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	contactID, err := uuid.Parse(req.ContactId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if s.visibilityService != nil {
		if req.IsAdmin {
			if visErr := s.visibilityService.AdminOverride(ctx, contactID, req.Visibility, tenantID); visErr != nil {
				return nil, mapCRMError(visErr)
			}
		} else {
			if visErr := s.visibilityService.SetVisibility(ctx, contactID, req.Visibility, userID, tenantID); visErr != nil {
				return nil, mapCRMError(visErr)
			}
		}
	} else {
		// Fallback: direct service call
		var ownerID *uuid.UUID
		if req.Visibility == "personal" {
			ownerID = &userID
		}
		if visErr := s.contactService.UpdateVisibility(ctx, contactID, req.Visibility, ownerID, tenantID); visErr != nil {
			return nil, mapCRMError(visErr)
		}
	}

	// Return updated contact
	c, err := s.contactService.GetByID(ctx, contactID, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.UpdateContactVisibilityResponse{
		Contact: toContactInfo(c),
	}, nil
}

func toImportResponse(result *emailcontact.ImportResult) *crmv1.ImportContactsResponse {
	resp := &crmv1.ImportContactsResponse{
		ImportedCount: int32(result.ImportedCount),
		MergedCount:   int32(result.MergedCount),
		SkippedCount:  int32(result.SkippedCount),
	}
	for _, e := range result.Errors {
		resp.Errors = append(resp.Errors, &crmv1.ImportContactError{
			Row:    int32(e.Row),
			Reason: e.Reason,
		})
	}
	return resp
}

// ============================================================================
// Helpers
// ============================================================================

func toSavedFilterInfo(f *models.SavedFilter) *crmv1.SavedFilterInfo {
	return &crmv1.SavedFilterInfo{
		Id:         f.ID.String(),
		Name:       f.Name,
		EntityType: string(f.EntityType),
		FilterJson: f.FilterJSON,
		IsDefault:  f.IsDefault,
		CreatedBy:  f.CreatedBy.String(),
		CreatedAt:  f.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  f.UpdatedAt.Format(time.RFC3339),
	}
}

func toActivityInfo(a *models.ActivityWithRelations) *crmv1.ActivityInfo {
	info := &crmv1.ActivityInfo{
		Id:           a.ID.String(),
		ActivityType: string(a.ActivityType),
		Subject:      a.Subject,
		IsCompleted:  a.IsCompleted,
		CreatedBy:    a.CreatedBy.String(),
		CreatedAt:    a.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    a.UpdatedAt.Format(time.RFC3339),
	}

	if a.Description != nil {
		info.Description = *a.Description
	}
	if a.ContactID != nil {
		contactIDStr := a.ContactID.String()
		info.ContactId = &contactIDStr
	}
	if a.ContactName != nil {
		info.ContactName = a.ContactName
	}
	if a.CompanyID != nil {
		companyIDStr := a.CompanyID.String()
		info.CompanyId = &companyIDStr
	}
	if a.CompanyName != nil {
		info.CompanyName = a.CompanyName
	}
	if a.DealID != nil {
		dealIDStr := a.DealID.String()
		info.DealId = &dealIDStr
	}
	if a.DealName != nil {
		info.DealName = a.DealName
	}
	if a.AssignedTo != nil {
		assignedToStr := a.AssignedTo.String()
		info.AssignedTo = &assignedToStr
	}
	if a.AssignedToName != nil {
		info.AssignedToName = a.AssignedToName
	}
	if a.DueDate != nil {
		info.DueDate = a.DueDate.Format(time.RFC3339)
	}
	if a.CompletedAt != nil {
		completedAtStr := a.CompletedAt.Format(time.RFC3339)
		info.CompletedAt = &completedAtStr
	}

	// Note: Tags and CustomFields are not in the proto ActivityInfo
	// They would need to be fetched separately if needed

	return info
}

func toSearchResultInfo(r *models.SearchResult) *crmv1.SearchResult {
	return &crmv1.SearchResult{
		Id:         r.ID,
		EntityType: r.EntityType,
		Title:      r.Title,
		Subtitle:   r.Subtitle,
		Score:      r.Score,
	}
}

func toDealInfo(d *models.DealWithRelations) *crmv1.DealInfo {
	value, _ := d.Value.Float64()
	info := &crmv1.DealInfo{
		Id:        d.ID.String(),
		Name:      d.Name,
		Value:     value,
		Currency:  d.Currency,
		StageId:   d.StageID.String(),
		StageName: d.StageName,
		CreatedBy: d.CreatedBy.String(),
		CreatedAt: d.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: d.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if d.ContactID != nil {
		contactIDStr := d.ContactID.String()
		info.ContactId = &contactIDStr
	}
	if d.ContactName != nil {
		info.ContactName = d.ContactName
	}
	if d.CompanyID != nil {
		companyIDStr := d.CompanyID.String()
		info.CompanyId = &companyIDStr
	}
	if d.CompanyName != nil {
		info.CompanyName = d.CompanyName
	}
	if d.OwnerID != nil {
		ownerIDStr := d.OwnerID.String()
		info.OwnerId = &ownerIDStr
	}
	if d.OwnerName != nil {
		info.OwnerName = d.OwnerName
	}
	if d.ExpectedCloseDate != nil {
		info.ExpectedCloseDate = d.ExpectedCloseDate.Format("2006-01-02")
	}
	if d.Notes != nil {
		info.Notes = *d.Notes
	}
	if d.ClosedAt != nil {
		closedAtStr := d.ClosedAt.Format("2006-01-02T15:04:05Z07:00")
		info.ClosedAt = &closedAtStr
	}

	for _, t := range d.Tags {
		info.Tags = append(info.Tags, toTagInfo(t))
	}

	if len(d.CustomFields) > 0 {
		info.CustomFields = make(map[string]string)
		for k, v := range d.CustomFields {
			jsonBytes, _ := json.Marshal(v)
			info.CustomFields[k] = string(jsonBytes)
		}
	}

	return info
}

func parseDate(s string) (time.Time, error) {
	// Try multiple formats
	formats := []string{
		"2006-01-02",
		"2006-01-02T15:04:05Z07:00",
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("invalid date format")
}

func toPipelineStageInfo(stage *models.PipelineStage, dealCount int, totalValue float64) *crmv1.PipelineStageInfo {
	prob, _ := stage.Probability.Float64()
	return &crmv1.PipelineStageInfo{
		Id:          stage.ID.String(),
		Name:        stage.Name,
		Color:       stage.Color,
		SortOrder:   int32(stage.SortOrder),
		IsWon:       stage.IsWon,
		IsLost:      stage.IsLost,
		Probability: prob,
		DealCount:   int32(dealCount),
		TotalValue:  totalValue,
		CreatedAt:   stage.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toPipelineStageInfoWithStats(stage *models.PipelineStageWithStats) *crmv1.PipelineStageInfo {
	prob, _ := stage.Probability.Float64()
	totalValue, _ := stage.TotalValue.Float64()
	return &crmv1.PipelineStageInfo{
		Id:          stage.ID.String(),
		Name:        stage.Name,
		Color:       stage.Color,
		SortOrder:   int32(stage.SortOrder),
		IsWon:       stage.IsWon,
		IsLost:      stage.IsLost,
		Probability: prob,
		DealCount:   int32(stage.DealCount),
		TotalValue:  totalValue,
		CreatedAt:   stage.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toTagInfo(t *models.Tag) *crmv1.TagInfo {
	return &crmv1.TagInfo{
		Id:         t.ID.String(),
		Name:       t.Name,
		Color:      t.Color,
		EntityType: string(t.EntityType),
		CreatedAt:  t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toContactInfo(c *models.ContactWithRelations) *crmv1.ContactInfo {
	info := &crmv1.ContactInfo{
		Id:         c.ID.String(),
		FirstName:  c.FirstName,
		LastName:   c.LastName,
		Visibility: c.Visibility,
		CreatedBy:  c.CreatedBy.String(),
		CreatedAt:  c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:  c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if c.OwnerID != nil {
		ownerIDStr := c.OwnerID.String()
		info.OwnerId = &ownerIDStr
	}
	if c.Email != nil {
		info.Email = *c.Email
	}
	if c.Phone != nil {
		info.Phone = *c.Phone
	}
	if c.CompanyID != nil {
		companyIDStr := c.CompanyID.String()
		info.CompanyId = &companyIDStr
	}
	if c.CompanyName != nil {
		info.CompanyName = c.CompanyName
	}
	if c.Position != nil {
		info.Position = *c.Position
	}
	if c.Notes != nil {
		info.Notes = *c.Notes
	}

	for _, t := range c.Tags {
		info.Tags = append(info.Tags, toTagInfo(t))
	}

	if len(c.CustomFields) > 0 {
		info.CustomFields = make(map[string]string)
		for k, v := range c.CustomFields {
			jsonBytes, _ := json.Marshal(v)
			info.CustomFields[k] = string(jsonBytes)
		}
	}

	return info
}

func toCompanyInfo(c *models.CompanyWithRelations) *crmv1.CompanyInfo {
	info := &crmv1.CompanyInfo{
		Id:           c.ID.String(),
		Name:         c.Name,
		ContactCount: int32(c.ContactCount),
		CreatedBy:    c.CreatedBy.String(),
		CreatedAt:    c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if c.Domain != nil {
		info.Domain = *c.Domain
	}
	if c.Industry != nil {
		info.Industry = *c.Industry
	}
	if c.EmployeeCount != nil {
		info.EmployeeCount = int32(*c.EmployeeCount)
	}
	if c.Address != nil {
		info.Address = *c.Address
	}
	if c.City != nil {
		info.City = *c.City
	}
	if c.Country != nil {
		info.Country = *c.Country
	}
	if c.Notes != nil {
		info.Notes = *c.Notes
	}

	for _, t := range c.Tags {
		info.Tags = append(info.Tags, toTagInfo(t))
	}

	if len(c.CustomFields) > 0 {
		info.CustomFields = make(map[string]string)
		for k, v := range c.CustomFields {
			jsonBytes, _ := json.Marshal(v)
			info.CustomFields[k] = string(jsonBytes)
		}
	}

	return info
}

func toCustomFieldInfo(field *models.CustomFieldDefinition) *crmv1.CustomFieldInfo {
	info := &crmv1.CustomFieldInfo{
		Id:         field.ID.String(),
		EntityType: string(field.EntityType),
		FieldName:  field.FieldName,
		FieldLabel: field.FieldLabel,
		FieldType:  string(field.FieldType),
		Options:    field.Options,
		IsRequired: field.IsRequired,
		SortOrder:  int32(field.SortOrder),
		CreatedBy:  field.CreatedBy.String(),
		CreatedAt:  field.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:  field.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if field.DefaultValue != nil {
		info.DefaultValue = field.DefaultValue
	}

	return info
}

func mapCRMError(err error) error {
	switch {
	// Custom field errors
	case errors.Is(err, customfield.ErrFieldNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, customfield.ErrFieldExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, customfield.ErrInvalidFieldType):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, customfield.ErrInvalidEntityType):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, customfield.ErrOptionsRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, customfield.ErrFieldNameRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, customfield.ErrFieldLabelRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, customfield.ErrFieldInUse):
		return status.Error(codes.FailedPrecondition, err.Error())
	// Tag errors
	case errors.Is(err, tag.ErrTagNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, tag.ErrTagExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, tag.ErrTagNameRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, tag.ErrInvalidColor):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, tag.ErrInvalidEntityType):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, tag.ErrTagInUse):
		return status.Error(codes.FailedPrecondition, err.Error())
	// Tenant errors (contact and company share the same message)
	case errors.Is(err, contact.ErrInvalidTenant):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, company.ErrInvalidTenant):
		return status.Error(codes.Unauthenticated, err.Error())
	// Contact errors
	case errors.Is(err, contact.ErrContactNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, contact.ErrEmailExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, contact.ErrFirstNameRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, contact.ErrLastNameRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, contact.ErrInvalidEmail):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, contact.ErrCompanyNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, contact.ErrContactInUse):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, contact.ErrTagNotFound):
		return status.Error(codes.NotFound, err.Error())
	// Lead lifecycle errors
	case errors.Is(err, contact.ErrLeadNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, contact.ErrInvalidLeadSource),
		errors.Is(err, contact.ErrInvalidLeadStatus),
		errors.Is(err, contact.ErrInvalidLeadTemperature),
		errors.Is(err, contact.ErrInvalidLifecycleStage),
		errors.Is(err, contact.ErrNoLeadChanges):
		return status.Error(codes.InvalidArgument, err.Error())
	// Company errors
	case errors.Is(err, company.ErrCompanyNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, company.ErrNameRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, company.ErrCompanyInUse):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, company.ErrTagNotFound):
		return status.Error(codes.NotFound, err.Error())
	// Pipeline stage errors
	case errors.Is(err, pipelinestage.ErrStageNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, pipelinestage.ErrNameRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, pipelinestage.ErrInvalidColor):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, pipelinestage.ErrInvalidProbability):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, pipelinestage.ErrStageHasDeals):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, pipelinestage.ErrWonStageExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, pipelinestage.ErrLostStageExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, pipelinestage.ErrInvalidReorder):
		return status.Error(codes.InvalidArgument, err.Error())
	// Deal errors
	case errors.Is(err, deal.ErrDealNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, deal.ErrNameRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, deal.ErrInvalidCurrency):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, deal.ErrStageNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, deal.ErrContactNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, deal.ErrCompanyNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, deal.ErrOwnerNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, deal.ErrTagNotFound):
		return status.Error(codes.NotFound, err.Error())
	// Activity errors
	case errors.Is(err, activity.ErrActivityNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, activity.ErrSubjectRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, activity.ErrInvalidActivityType):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, activity.ErrContactNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, activity.ErrCompanyNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, activity.ErrDealNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, activity.ErrUserNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, activity.ErrTagNotFound):
		return status.Error(codes.NotFound, err.Error())
	// Search errors
	case errors.Is(err, search.ErrQueryTooShort):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, search.ErrQueryTooLong):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, search.ErrInvalidEntity):
		return status.Error(codes.InvalidArgument, err.Error())
	// Saved filter errors
	case errors.Is(err, savedfilter.ErrFilterNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, savedfilter.ErrNameRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, savedfilter.ErrInvalidEntityType):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, savedfilter.ErrInvalidFilterJSON):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, savedfilter.ErrFilterNotOwned):
		return status.Error(codes.PermissionDenied, err.Error())
	// Report errors
	case errors.Is(err, report.ErrInvalidDateRange):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, report.ErrStartDateRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, report.ErrEndDateRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	// Visibility errors
	case errors.Is(err, emailcontact.ErrInvalidVisibility):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, emailcontact.ErrNotAuthorized):
		return status.Error(codes.PermissionDenied, err.Error())
	// Consent errors
	case errors.Is(err, consent.ErrContactNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, consent.ErrInvalidConsentType):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, consent.ErrInvalidLegalBasis):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, consent.ErrDeletionRequestNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, consent.ErrDeletionAlreadyComplete):
		return status.Error(codes.AlreadyExists, err.Error())
	// Merge errors
	case errors.Is(err, contact.ErrCannotMergeSelf):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, contact.ErrAlreadyMerged):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, company.ErrCannotMergeSelf):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, company.ErrAlreadyMerged):
		return status.Error(codes.FailedPrecondition, err.Error())
	// Advisory protocol errors
	case errors.Is(err, advisoryprotocol.ErrProtocolNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, advisoryprotocol.ErrProtocolFinalized):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, advisoryprotocol.ErrContactNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, advisoryprotocol.ErrInvalidRiskClass):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		slog.Error("unhandled crm service error", "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}

// ============================================================================
// Duplicate Detection
// ============================================================================

func (s *CRMGRPCServer) FindContactDuplicates(ctx context.Context, req *crmv1.FindContactDuplicatesRequest) (*crmv1.FindContactDuplicatesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	contactID, err := uuid.Parse(req.ContactId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
	}

	candidates, err := s.contactService.FindDuplicates(ctx, contactID, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	var results []*crmv1.DuplicateContactCandidate
	for _, c := range candidates {
		results = append(results, &crmv1.DuplicateContactCandidate{
			Contact:    toContactInfo(c.Contact),
			Similarity: c.Similarity,
			MatchType:  c.MatchType,
		})
	}

	return &crmv1.FindContactDuplicatesResponse{
		Duplicates: results,
		Total:      int32(len(results)),
	}, nil
}

func (s *CRMGRPCServer) MergeContacts(ctx context.Context, req *crmv1.MergeContactsRequest) (*crmv1.MergeContactsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	primaryID, err := uuid.Parse(req.PrimaryId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid primary_id")
	}
	duplicateID, err := uuid.Parse(req.DuplicateId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid duplicate_id")
	}

	merged, err := s.contactService.MergeContacts(ctx, primaryID, duplicateID, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.MergeContactsResponse{
		Contact: toContactInfo(merged),
	}, nil
}

func (s *CRMGRPCServer) FindCompanyDuplicates(ctx context.Context, req *crmv1.FindCompanyDuplicatesRequest) (*crmv1.FindCompanyDuplicatesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	companyID, err := uuid.Parse(req.CompanyId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid company_id")
	}

	candidates, err := s.companyService.FindDuplicates(ctx, companyID, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	var results []*crmv1.DuplicateCompanyCandidate
	for _, c := range candidates {
		results = append(results, &crmv1.DuplicateCompanyCandidate{
			Company:    toCompanyInfo(c.Company),
			Similarity: c.Similarity,
			MatchType:  c.MatchType,
		})
	}

	return &crmv1.FindCompanyDuplicatesResponse{
		Duplicates: results,
		Total:      int32(len(results)),
	}, nil
}

func (s *CRMGRPCServer) MergeCompanies(ctx context.Context, req *crmv1.MergeCompaniesRequest) (*crmv1.MergeCompaniesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	primaryID, err := uuid.Parse(req.PrimaryId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid primary_id")
	}
	duplicateID, err := uuid.Parse(req.DuplicateId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid duplicate_id")
	}

	merged, err := s.companyService.MergeCompanies(ctx, primaryID, duplicateID, tenantID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.MergeCompaniesResponse{
		Company: toCompanyInfo(merged),
	}, nil
}

// ============================================================================
// Contact Timeline
// ============================================================================

func (s *CRMGRPCServer) GetContactTimeline(ctx context.Context, req *crmv1.GetContactTimelineRequest) (*crmv1.GetContactTimelineResponse, error) {
	contactID, err := uuid.Parse(req.ContactId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	result, err := s.activityService.GetContactTimeline(ctx, contactID, tenantID, page, pageSize)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var events []*crmv1.TimelineEvent
	for _, e := range result.Events {
		event := &crmv1.TimelineEvent{
			Id:            e.ID.String(),
			EventType:     e.EventType,
			OccurredAt:    e.OccurredAt.Format("2006-01-02T15:04:05Z07:00"),
			Title:         e.Title,
			CreatedByName: e.CreatedByName,
		}
		if e.Description != nil {
			event.Description = e.Description
		}
		if len(e.Metadata) > 0 {
			meta := &crmv1.TimelineEventMetadata{Data: make(map[string]string)}
			for k, v := range e.Metadata {
				jsonBytes, _ := json.Marshal(v)
				meta.Data[k] = string(jsonBytes)
			}
			event.Metadata = meta
		}
		events = append(events, event)
	}

	return &crmv1.GetContactTimelineResponse{
		Events: events,
		Total:  int32(result.Total),
	}, nil
}

// ============================================================================
// GDPR Consent Management
// ============================================================================

func toConsentRecord(r *consent.ConsentRecord) *crmv1.ConsentRecord {
	rec := &crmv1.ConsentRecord{
		Id:          r.ID.String(),
		ContactId:   r.ContactID.String(),
		ConsentType: r.ConsentType,
		Granted:     r.Granted,
		LegalBasis:  r.LegalBasis,
		Source:      r.Source,
		Notes:       r.Notes,
		CreatedAt:   r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if r.IPAddress != nil {
		rec.IpAddress = r.IPAddress
	}
	if r.GrantedAt != nil {
		t := r.GrantedAt.Format("2006-01-02T15:04:05Z07:00")
		rec.GrantedAt = &t
	}
	if r.RevokedAt != nil {
		t := r.RevokedAt.Format("2006-01-02T15:04:05Z07:00")
		rec.RevokedAt = &t
	}
	if r.CreatedBy != nil {
		s := r.CreatedBy.String()
		rec.CreatedBy = &s
	}
	return rec
}

func (s *CRMGRPCServer) GetContactConsents(ctx context.Context, req *crmv1.GetContactConsentsRequest) (*crmv1.GetContactConsentsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	contactID, err := uuid.Parse(req.ContactId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
	}

	summary, err := s.consentService.GetContactConsents(ctx, tenantID, contactID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	protoSummary := &crmv1.ConsentSummary{
		ContactId: summary.ContactID.String(),
		Consents:  make(map[string]*crmv1.ConsentRecord),
	}
	for k, v := range summary.Consents {
		protoSummary.Consents[k] = toConsentRecord(v)
	}

	return &crmv1.GetContactConsentsResponse{
		Summary: protoSummary,
	}, nil
}

func (s *CRMGRPCServer) GrantConsent(ctx context.Context, req *crmv1.GrantConsentRequest) (*crmv1.GrantConsentResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	contactID, err := uuid.Parse(req.ContactId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
	}

	var createdBy *uuid.UUID
	if req.CreatedBy != nil {
		uid, parseErr := uuid.Parse(*req.CreatedBy)
		if parseErr == nil {
			createdBy = &uid
		}
	}

	record, err := s.consentService.GrantConsent(ctx, tenantID, contactID, req.ConsentType, req.Source, req.LegalBasis, req.IpAddress, createdBy)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.GrantConsentResponse{
		Record: toConsentRecord(record),
	}, nil
}

func (s *CRMGRPCServer) RevokeConsent(ctx context.Context, req *crmv1.RevokeConsentRequest) (*crmv1.RevokeConsentResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	contactID, err := uuid.Parse(req.ContactId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
	}

	var revokedBy *uuid.UUID
	if req.RevokedBy != nil {
		uid, parseErr := uuid.Parse(*req.RevokedBy)
		if parseErr == nil {
			revokedBy = &uid
		}
	}

	record, err := s.consentService.RevokeConsent(ctx, tenantID, contactID, req.ConsentType, req.Notes, revokedBy)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.RevokeConsentResponse{
		Record: toConsentRecord(record),
	}, nil
}

func (s *CRMGRPCServer) GetConsentHistory(ctx context.Context, req *crmv1.GetConsentHistoryRequest) (*crmv1.GetConsentHistoryResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	contactID, err := uuid.Parse(req.ContactId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
	}

	records, err := s.consentService.GetConsentHistory(ctx, tenantID, contactID, req.ConsentType)
	if err != nil {
		return nil, mapCRMError(err)
	}

	var history []*crmv1.ConsentRecord
	for _, r := range records {
		history = append(history, toConsentRecord(r))
	}

	return &crmv1.GetConsentHistoryResponse{
		History: history,
		Total:   int32(len(history)),
	}, nil
}

func (s *CRMGRPCServer) RequestDeletion(ctx context.Context, req *crmv1.RequestDeletionRequest) (*crmv1.RequestDeletionResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	contactID, err := uuid.Parse(req.ContactId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
	}

	var requestedBy *uuid.UUID
	if req.RequestedBy != nil {
		uid, parseErr := uuid.Parse(*req.RequestedBy)
		if parseErr == nil {
			requestedBy = &uid
		}
	}

	deletionReq, err := s.consentService.RequestDeletion(ctx, tenantID, contactID, req.Reason, requestedBy)
	if err != nil {
		return nil, mapCRMError(err)
	}

	proto := &crmv1.GDPRDeletionRequest{
		Id:        deletionReq.ID.String(),
		ContactId: deletionReq.ContactID.String(),
		Reason:    deletionReq.Reason,
		Status:    deletionReq.Status,
		CreatedAt: deletionReq.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if deletionReq.RequestedBy != nil {
		s := deletionReq.RequestedBy.String()
		proto.RequestedBy = &s
	}
	if deletionReq.CompletedAt != nil {
		t := deletionReq.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
		proto.CompletedAt = &t
	}

	return &crmv1.RequestDeletionResponse{
		DeletionRequest: proto,
	}, nil
}

func (s *CRMGRPCServer) ProcessDeletion(ctx context.Context, req *crmv1.ProcessDeletionRequest) (*crmv1.ProcessDeletionResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	requestID, err := uuid.Parse(req.RequestId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request_id")
	}

	if err := s.consentService.ProcessDeletion(ctx, tenantID, requestID); err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.ProcessDeletionResponse{
		Status: "completed",
	}, nil
}
