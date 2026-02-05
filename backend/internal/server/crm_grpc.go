package server

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/crm/company"
	"github.com/kmuhub/kmuhub/internal/crm/contact"
	"github.com/kmuhub/kmuhub/internal/crm/customfield"
	"github.com/kmuhub/kmuhub/internal/crm/deal"
	"github.com/kmuhub/kmuhub/internal/crm/pipelinestage"
	"github.com/kmuhub/kmuhub/internal/crm/tag"
	"github.com/kmuhub/kmuhub/internal/models"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// CRMGRPCServer implements the CRM gRPC service
type CRMGRPCServer struct {
	crmv1.UnimplementedCRMServiceServer
	customFieldService   *customfield.Service
	tagService           *tag.Service
	contactService       *contact.Service
	companyService       *company.Service
	pipelineStageService *pipelinestage.Service
	dealService          *deal.Service
}

// NewCRMGRPCServer creates a new CRM gRPC server
func NewCRMGRPCServer(
	customFieldService *customfield.Service,
	tagService *tag.Service,
	contactService *contact.Service,
	companyService *company.Service,
	pipelineStageService *pipelinestage.Service,
	dealService *deal.Service,
) *CRMGRPCServer {
	return &CRMGRPCServer{
		customFieldService:   customFieldService,
		tagService:           tagService,
		contactService:       contactService,
		companyService:       companyService,
		pipelineStageService: pipelineStageService,
		dealService:          dealService,
	}
}

// ============================================================================
// Custom Fields
// ============================================================================

func (s *CRMGRPCServer) CreateCustomField(ctx context.Context, req *crmv1.CreateCustomFieldRequest) (*crmv1.CreateCustomFieldResponse, error) {
	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by user id")
	}

	input := customfield.CreateInput{
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
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid custom field id")
	}

	field, err := s.customFieldService.GetByID(ctx, id)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.GetCustomFieldResponse{
		CustomField: toCustomFieldInfo(field),
	}, nil
}

func (s *CRMGRPCServer) ListCustomFields(ctx context.Context, req *crmv1.ListCustomFieldsRequest) (*crmv1.ListCustomFieldsResponse, error) {
	input := customfield.ListInput{
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
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid custom field id")
	}

	input := customfield.UpdateInput{}

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
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid custom field id")
	}

	if err := s.customFieldService.Delete(ctx, id); err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.DeleteCustomFieldResponse{}, nil
}

// ============================================================================
// Tags
// ============================================================================

func (s *CRMGRPCServer) CreateTag(ctx context.Context, req *crmv1.CreateTagRequest) (*crmv1.CreateTagResponse, error) {
	input := tag.CreateInput{
		Name:       req.Name,
		Color:      req.Color,
		EntityType: models.EntityType(req.EntityType),
	}

	t, err := s.tagService.Create(ctx, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.CreateTagResponse{
		Tag: toTagInfo(t),
	}, nil
}

func (s *CRMGRPCServer) GetTag(ctx context.Context, req *crmv1.GetTagRequest) (*crmv1.GetTagResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tag id")
	}

	t, err := s.tagService.GetByID(ctx, id)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.GetTagResponse{
		Tag: toTagInfo(t),
	}, nil
}

func (s *CRMGRPCServer) ListTags(ctx context.Context, req *crmv1.ListTagsRequest) (*crmv1.ListTagsResponse, error) {
	input := tag.ListInput{
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
	}

	if req.EntityType != "" {
		entityType := models.EntityType(req.EntityType)
		input.EntityType = &entityType
	}

	tags, total, err := s.tagService.List(ctx, input)
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

	t, err := s.tagService.Update(ctx, id, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.UpdateTagResponse{
		Tag: toTagInfo(t),
	}, nil
}

func (s *CRMGRPCServer) DeleteTag(ctx context.Context, req *crmv1.DeleteTagRequest) (*crmv1.DeleteTagResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tag id")
	}

	if err := s.tagService.Delete(ctx, id); err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.DeleteTagResponse{}, nil
}

// ============================================================================
// Contacts
// ============================================================================

func (s *CRMGRPCServer) CreateContact(ctx context.Context, req *crmv1.CreateContactRequest) (*crmv1.CreateContactResponse, error) {
	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by user id")
	}

	input := contact.CreateInput{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		CreatedBy: createdBy,
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
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid contact id")
	}

	c, err := s.contactService.GetByID(ctx, id)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.GetContactResponse{
		Contact: toContactInfo(c),
	}, nil
}

func (s *CRMGRPCServer) ListContacts(ctx context.Context, req *crmv1.ListContactsRequest) (*crmv1.ListContactsResponse, error) {
	input := contact.ListInput{
		Search:   req.Search,
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
		SortBy:   req.SortBy,
		SortDesc: req.SortDesc,
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

	c, err := s.contactService.Update(ctx, id, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.UpdateContactResponse{
		Contact: toContactInfo(c),
	}, nil
}

func (s *CRMGRPCServer) DeleteContact(ctx context.Context, req *crmv1.DeleteContactRequest) (*crmv1.DeleteContactResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid contact id")
	}

	if err := s.contactService.Delete(ctx, id); err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.DeleteContactResponse{}, nil
}

func (s *CRMGRPCServer) AddContactTags(ctx context.Context, req *crmv1.AddContactTagsRequest) (*crmv1.AddContactTagsResponse, error) {
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

	c, err := s.contactService.AddTags(ctx, contactID, tagIDs)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.AddContactTagsResponse{
		Contact: toContactInfo(c),
	}, nil
}

func (s *CRMGRPCServer) RemoveContactTags(ctx context.Context, req *crmv1.RemoveContactTagsRequest) (*crmv1.RemoveContactTagsResponse, error) {
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

	c, err := s.contactService.RemoveTags(ctx, contactID, tagIDs)
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
	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by user id")
	}

	input := company.CreateInput{
		Name:      req.Name,
		CreatedBy: createdBy,
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
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid company id")
	}

	c, err := s.companyService.GetByID(ctx, id)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.GetCompanyResponse{
		Company: toCompanyInfo(c),
	}, nil
}

func (s *CRMGRPCServer) ListCompanies(ctx context.Context, req *crmv1.ListCompaniesRequest) (*crmv1.ListCompaniesResponse, error) {
	input := company.ListInput{
		Search:   req.Search,
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
		SortBy:   req.SortBy,
		SortDesc: req.SortDesc,
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

	c, err := s.companyService.Update(ctx, id, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.UpdateCompanyResponse{
		Company: toCompanyInfo(c),
	}, nil
}

func (s *CRMGRPCServer) DeleteCompany(ctx context.Context, req *crmv1.DeleteCompanyRequest) (*crmv1.DeleteCompanyResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid company id")
	}

	if err := s.companyService.Delete(ctx, id); err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.DeleteCompanyResponse{}, nil
}

func (s *CRMGRPCServer) GetCompanyContacts(ctx context.Context, req *crmv1.GetCompanyContactsRequest) (*crmv1.GetCompanyContactsResponse, error) {
	companyID, err := uuid.Parse(req.CompanyId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid company_id")
	}

	input := contact.ListInput{
		CompanyID: &companyID,
		Page:      int(req.Page),
		PageSize:  int(req.PageSize),
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
	input := pipelinestage.CreateInput{
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
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid pipeline stage id")
	}

	stage, err := s.pipelineStageService.GetByID(ctx, id)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.GetPipelineStageResponse{
		Stage: toPipelineStageInfo(stage, 0, 0),
	}, nil
}

func (s *CRMGRPCServer) ListPipelineStages(ctx context.Context, req *crmv1.ListPipelineStagesRequest) (*crmv1.ListPipelineStagesResponse, error) {
	stages, err := s.pipelineStageService.ListWithStats(ctx)
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

	stage, err := s.pipelineStageService.Update(ctx, id, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.UpdatePipelineStageResponse{
		Stage: toPipelineStageInfo(stage, 0, 0),
	}, nil
}

func (s *CRMGRPCServer) DeletePipelineStage(ctx context.Context, req *crmv1.DeletePipelineStageRequest) (*crmv1.DeletePipelineStageResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid pipeline stage id")
	}

	if err := s.pipelineStageService.Delete(ctx, id); err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.DeletePipelineStageResponse{}, nil
}

func (s *CRMGRPCServer) ReorderPipelineStages(ctx context.Context, req *crmv1.ReorderPipelineStagesRequest) (*crmv1.ReorderPipelineStagesResponse, error) {
	var stageIDs []uuid.UUID
	for _, idStr := range req.StageIds {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid stage id in list")
		}
		stageIDs = append(stageIDs, id)
	}

	if err := s.pipelineStageService.Reorder(ctx, stageIDs); err != nil {
		return nil, mapCRMError(err)
	}

	// Return updated list
	stages, err := s.pipelineStageService.ListWithStats(ctx)
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
	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by user id")
	}

	stageID, err := uuid.Parse(req.StageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid stage_id")
	}

	input := deal.CreateInput{
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
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid deal id")
	}

	d, err := s.dealService.GetByID(ctx, id)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.GetDealResponse{
		Deal: toDealInfo(d),
	}, nil
}

func (s *CRMGRPCServer) ListDeals(ctx context.Context, req *crmv1.ListDealsRequest) (*crmv1.ListDealsResponse, error) {
	input := deal.ListInput{
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

	d, err := s.dealService.Update(ctx, id, input)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.UpdateDealResponse{
		Deal: toDealInfo(d),
	}, nil
}

func (s *CRMGRPCServer) DeleteDeal(ctx context.Context, req *crmv1.DeleteDealRequest) (*crmv1.DeleteDealResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid deal id")
	}

	if err := s.dealService.Delete(ctx, id); err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.DeleteDealResponse{}, nil
}

func (s *CRMGRPCServer) MoveDealToStage(ctx context.Context, req *crmv1.MoveDealToStageRequest) (*crmv1.MoveDealToStageResponse, error) {
	dealID, err := uuid.Parse(req.DealId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid deal_id")
	}

	stageID, err := uuid.Parse(req.StageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid stage_id")
	}

	d, err := s.dealService.MoveToStage(ctx, dealID, stageID)
	if err != nil {
		return nil, mapCRMError(err)
	}

	return &crmv1.MoveDealToStageResponse{
		Deal: toDealInfo(d),
	}, nil
}

// ============================================================================
// Helpers
// ============================================================================

func toDealInfo(d *models.DealWithRelations) *crmv1.DealInfo {
	value, _ := d.Value.Float64()
	info := &crmv1.DealInfo{
		Id:         d.ID.String(),
		Name:       d.Name,
		Value:      value,
		Currency:   d.Currency,
		StageId:    d.StageID.String(),
		StageName:  d.StageName,
		CreatedBy:  d.CreatedBy.String(),
		CreatedAt:  d.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:  d.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
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
		Id:        c.ID.String(),
		FirstName: c.FirstName,
		LastName:  c.LastName,
		CreatedBy: c.CreatedBy.String(),
		CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
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
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
