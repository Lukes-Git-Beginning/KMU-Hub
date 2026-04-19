package server

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/formulare"
	formularev1 "github.com/kmuhub/kmuhub/proto/formulare/v1"
)

// ============================================================================
// Server
// ============================================================================

// FormulareGRPCServer implements the FormulareService gRPC server.
type FormulareGRPCServer struct {
	formularev1.UnimplementedFormulareServiceServer
	svc *formulare.Service
}

// NewFormulareGRPCServer creates a new Formulare gRPC server.
func NewFormulareGRPCServer(svc *formulare.Service) *FormulareGRPCServer {
	return &FormulareGRPCServer{svc: svc}
}

// ============================================================================
// Schema Cluster RPCs
// ============================================================================

func (s *FormulareGRPCServer) CreateFormSchema(ctx context.Context, req *formularev1.CreateFormSchemaRequest) (*formularev1.CreateFormSchemaResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	var createdBy *uuid.UUID
	if cb := req.GetCreatedBy(); cb != "" {
		id, parseErr := uuid.Parse(cb)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid created_by: %v", parseErr)
		}
		createdBy = &id
	}

	input := formulare.CreateSchemaInput{
		TenantID:    tenantID,
		Title:       req.GetTitle(),
		Description: req.GetDescription(),
		Fields:      req.GetFields(),
		Status:      fromProtoFormSchemaStatus(req.GetStatus()),
		IsTemplate:  req.GetIsTemplate(),
		IsPublic:    req.GetIsPublic(),
		PageCount:   int(req.GetPageCount()),
		CreatedBy:   createdBy,
	}

	schema, err := s.svc.CreateFormSchema(ctx, input)
	if err != nil {
		return nil, mapFormulareError(err)
	}
	return &formularev1.CreateFormSchemaResponse{Schema: toProtoFormSchema(schema)}, nil
}

func (s *FormulareGRPCServer) GetFormSchema(ctx context.Context, req *formularev1.GetFormSchemaRequest) (*formularev1.GetFormSchemaResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	schemaID, err := uuid.Parse(req.GetSchemaId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid schema_id: %v", err)
	}

	schema, err := s.svc.GetFormSchema(ctx, schemaID, tenantID)
	if err != nil {
		return nil, mapFormulareError(err)
	}
	return &formularev1.GetFormSchemaResponse{Schema: toProtoFormSchema(schema)}, nil
}

func (s *FormulareGRPCServer) UpdateFormSchema(ctx context.Context, req *formularev1.UpdateFormSchemaRequest) (*formularev1.UpdateFormSchemaResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	schemaID, err := uuid.Parse(req.GetSchemaId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid schema_id: %v", err)
	}

	input := formulare.UpdateSchemaInput{
		ID:       schemaID,
		TenantID: tenantID,
		Fields:   req.GetFields(),
	}
	if req.Title != nil {
		input.Title = req.Title
	}
	if req.Description != nil {
		input.Description = req.Description
	}
	if req.Status != nil {
		st := fromProtoFormSchemaStatus(*req.Status)
		input.Status = &st
	}
	if req.IsTemplate != nil {
		input.IsTemplate = req.IsTemplate
	}
	if req.IsPublic != nil {
		input.IsPublic = req.IsPublic
	}
	if req.PageCount != nil {
		pc := int(*req.PageCount)
		input.PageCount = &pc
	}

	schema, err := s.svc.UpdateFormSchema(ctx, input)
	if err != nil {
		return nil, mapFormulareError(err)
	}
	return &formularev1.UpdateFormSchemaResponse{Schema: toProtoFormSchema(schema)}, nil
}

func (s *FormulareGRPCServer) DeleteFormSchema(ctx context.Context, req *formularev1.DeleteFormSchemaRequest) (*formularev1.DeleteFormSchemaResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	schemaID, err := uuid.Parse(req.GetSchemaId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid schema_id: %v", err)
	}

	if err := s.svc.DeleteFormSchema(ctx, schemaID, tenantID); err != nil {
		return nil, mapFormulareError(err)
	}
	return &formularev1.DeleteFormSchemaResponse{}, nil
}

func (s *FormulareGRPCServer) ListFormSchemas(ctx context.Context, req *formularev1.ListFormSchemasRequest) (*formularev1.ListFormSchemasResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := formulare.ListSchemasInput{
		TenantID: tenantID,
		Search:   req.GetSearch(),
		Limit:    req.GetLimit(),
		Offset:   req.GetOffset(),
	}
	if req.Status != nil {
		st := fromProtoFormSchemaStatus(*req.Status)
		input.Status = &st
	}
	if req.IsTemplate != nil {
		input.IsTemplate = req.IsTemplate
	}

	schemas, total, err := s.svc.ListFormSchemas(ctx, input)
	if err != nil {
		return nil, mapFormulareError(err)
	}

	protoSchemas := make([]*formularev1.FormSchema, len(schemas))
	for i, sc := range schemas {
		protoSchemas[i] = toProtoFormSchema(sc)
	}
	return &formularev1.ListFormSchemasResponse{
		Schemas: protoSchemas,
		Total:   int32(total),
	}, nil
}

func (s *FormulareGRPCServer) DuplicateFormSchema(ctx context.Context, req *formularev1.DuplicateFormSchemaRequest) (*formularev1.DuplicateFormSchemaResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	schemaID, err := uuid.Parse(req.GetSchemaId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid schema_id: %v", err)
	}

	newTitle := ""
	if req.NewTitle != nil {
		newTitle = *req.NewTitle
	}

	schema, err := s.svc.DuplicateFormSchema(ctx, schemaID, tenantID, newTitle)
	if err != nil {
		return nil, mapFormulareError(err)
	}
	return &formularev1.DuplicateFormSchemaResponse{Schema: toProtoFormSchema(schema)}, nil
}

// ============================================================================
// Submission Cluster RPCs
// ============================================================================

func (s *FormulareGRPCServer) CreateSubmission(ctx context.Context, req *formularev1.CreateSubmissionRequest) (*formularev1.CreateSubmissionResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := formulare.CreateSubmissionInput{
		TenantID:    tenantID,
		Answers:     req.GetAnswers(),
		IPAddress:   req.GetIpAddress(),
		SubmittedBy: req.SubmittedBy,
	}

	if fsID := req.GetFormSchemaId(); fsID != "" {
		id, parseErr := uuid.Parse(fsID)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid form_schema_id: %v", parseErr)
		}
		input.FormSchemaID = &id
	}

	submission, err := s.svc.CreateSubmission(ctx, input)
	if err != nil {
		return nil, mapFormulareError(err)
	}
	return &formularev1.CreateSubmissionResponse{Submission: toProtoFormSubmission(submission)}, nil
}

func (s *FormulareGRPCServer) GetSubmission(ctx context.Context, req *formularev1.GetSubmissionRequest) (*formularev1.GetSubmissionResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	submissionID, err := uuid.Parse(req.GetSubmissionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid submission_id: %v", err)
	}

	submission, err := s.svc.GetSubmission(ctx, submissionID, tenantID)
	if err != nil {
		return nil, mapFormulareError(err)
	}
	return &formularev1.GetSubmissionResponse{Submission: toProtoFormSubmission(submission)}, nil
}

func (s *FormulareGRPCServer) ListSubmissions(ctx context.Context, req *formularev1.ListSubmissionsRequest) (*formularev1.ListSubmissionsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := formulare.ListSubmissionsInput{
		TenantID: tenantID,
		Limit:    req.GetLimit(),
		Offset:   req.GetOffset(),
	}
	if req.FormSchemaId != nil {
		id, parseErr := uuid.Parse(*req.FormSchemaId)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid form_schema_id: %v", parseErr)
		}
		input.FormSchemaID = &id
	}
	if req.Status != nil {
		st := fromProtoSubmissionStatus(*req.Status)
		input.Status = &st
	}

	submissions, total, err := s.svc.ListSubmissions(ctx, input)
	if err != nil {
		return nil, mapFormulareError(err)
	}

	protoSubs := make([]*formularev1.FormSubmission, len(submissions))
	for i, sub := range submissions {
		protoSubs[i] = toProtoFormSubmission(sub)
	}
	return &formularev1.ListSubmissionsResponse{
		Submissions: protoSubs,
		Total:       int32(total),
	}, nil
}

func (s *FormulareGRPCServer) UpdateSubmissionStatus(ctx context.Context, req *formularev1.UpdateSubmissionStatusRequest) (*formularev1.UpdateSubmissionStatusResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	submissionID, err := uuid.Parse(req.GetSubmissionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid submission_id: %v", err)
	}

	domainStatus := fromProtoSubmissionStatus(req.GetStatus())

	submission, err := s.svc.UpdateSubmissionStatus(ctx, submissionID, tenantID, domainStatus)
	if err != nil {
		return nil, mapFormulareError(err)
	}
	return &formularev1.UpdateSubmissionStatusResponse{Submission: toProtoFormSubmission(submission)}, nil
}

func (s *FormulareGRPCServer) ExportSubmissions(ctx context.Context, req *formularev1.ExportSubmissionsRequest) (*formularev1.ExportSubmissionsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	formSchemaID, err := uuid.Parse(req.GetFormSchemaId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid form_schema_id: %v", err)
	}

	input := formulare.ExportInput{
		TenantID:     tenantID,
		FormSchemaID: formSchemaID,
		Format:       fromProtoExportFormat(req.GetFormat()),
	}
	if req.Status != nil {
		st := fromProtoSubmissionStatus(*req.Status)
		input.Status = &st
	}
	if req.SubmittedAfter != nil {
		t := req.SubmittedAfter.AsTime()
		input.SubmittedAfter = &t
	}
	if req.SubmittedBefore != nil {
		t := req.SubmittedBefore.AsTime()
		input.SubmittedBefore = &t
	}

	result, err := s.svc.ExportSubmissions(ctx, input)
	if err != nil {
		return nil, mapFormulareError(err)
	}
	return &formularev1.ExportSubmissionsResponse{
		Content:  result.Content,
		Filename: result.Filename,
		MimeType: result.MimeType,
	}, nil
}

// ============================================================================
// Webhook Cluster RPCs
// ============================================================================

func (s *FormulareGRPCServer) CreateWebhook(ctx context.Context, req *formularev1.CreateWebhookRequest) (*formularev1.CreateWebhookResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	formSchemaID, err := uuid.Parse(req.GetFormSchemaId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid form_schema_id: %v", err)
	}

	input := formulare.CreateWebhookInput{
		TenantID:     tenantID,
		FormSchemaID: formSchemaID,
		URL:          req.GetUrl(),
		Secret:       req.Secret,
		Events:       req.GetEvents(),
		Active:       req.GetActive(),
	}

	webhook, err := s.svc.CreateWebhook(ctx, input)
	if err != nil {
		return nil, mapFormulareError(err)
	}
	return &formularev1.CreateWebhookResponse{Webhook: toProtoFormWebhook(webhook)}, nil
}

func (s *FormulareGRPCServer) GetWebhook(ctx context.Context, req *formularev1.GetWebhookRequest) (*formularev1.GetWebhookResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	webhookID, err := uuid.Parse(req.GetWebhookId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid webhook_id: %v", err)
	}

	webhook, err := s.svc.GetWebhook(ctx, webhookID, tenantID)
	if err != nil {
		return nil, mapFormulareError(err)
	}
	return &formularev1.GetWebhookResponse{Webhook: toProtoFormWebhook(webhook)}, nil
}

func (s *FormulareGRPCServer) UpdateWebhook(ctx context.Context, req *formularev1.UpdateWebhookRequest) (*formularev1.UpdateWebhookResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	webhookID, err := uuid.Parse(req.GetWebhookId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid webhook_id: %v", err)
	}

	input := formulare.UpdateWebhookInput{
		ID:        webhookID,
		TenantID:  tenantID,
		Events:    req.GetEvents(),
		EventsSet: req.GetEventsSet(),
	}
	if req.Url != nil {
		input.URL = req.Url
	}
	if req.Secret != nil {
		input.Secret = req.Secret
	}
	if req.Active != nil {
		input.Active = req.Active
	}

	webhook, err := s.svc.UpdateWebhook(ctx, input)
	if err != nil {
		return nil, mapFormulareError(err)
	}
	return &formularev1.UpdateWebhookResponse{Webhook: toProtoFormWebhook(webhook)}, nil
}

func (s *FormulareGRPCServer) DeleteWebhook(ctx context.Context, req *formularev1.DeleteWebhookRequest) (*formularev1.DeleteWebhookResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	webhookID, err := uuid.Parse(req.GetWebhookId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid webhook_id: %v", err)
	}

	if err := s.svc.DeleteWebhook(ctx, webhookID, tenantID); err != nil {
		return nil, mapFormulareError(err)
	}
	return &formularev1.DeleteWebhookResponse{}, nil
}

func (s *FormulareGRPCServer) ListWebhooks(ctx context.Context, req *formularev1.ListWebhooksRequest) (*formularev1.ListWebhooksResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	formSchemaID, err := uuid.Parse(req.GetFormSchemaId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid form_schema_id: %v", err)
	}

	webhooks, err := s.svc.ListWebhooks(ctx, formSchemaID, tenantID)
	if err != nil {
		return nil, mapFormulareError(err)
	}

	protoWebhooks := make([]*formularev1.FormWebhook, len(webhooks))
	for i, w := range webhooks {
		protoWebhooks[i] = toProtoFormWebhook(w)
	}
	return &formularev1.ListWebhooksResponse{Webhooks: protoWebhooks}, nil
}

// ============================================================================
// Delivery Observability RPCs
// ============================================================================

func (s *FormulareGRPCServer) ListWebhookDeliveries(ctx context.Context, req *formularev1.ListWebhookDeliveriesRequest) (*formularev1.ListWebhookDeliveriesResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := formulare.ListDeliveriesInput{
		TenantID: tenantID,
		Limit:    req.GetLimit(),
		Offset:   req.GetOffset(),
	}
	if req.WebhookId != nil {
		id, parseErr := uuid.Parse(*req.WebhookId)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid webhook_id: %v", parseErr)
		}
		input.WebhookID = &id
	}
	if req.SubmissionId != nil {
		id, parseErr := uuid.Parse(*req.SubmissionId)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid submission_id: %v", parseErr)
		}
		input.SubmissionID = &id
	}
	if req.Status != nil {
		st := fromProtoDeliveryStatus(*req.Status)
		input.Status = &st
	}

	deliveries, total, err := s.svc.ListWebhookDeliveries(ctx, input)
	if err != nil {
		return nil, mapFormulareError(err)
	}

	protoDels := make([]*formularev1.WebhookDelivery, len(deliveries))
	for i, d := range deliveries {
		protoDels[i] = toProtoWebhookDelivery(d)
	}
	return &formularev1.ListWebhookDeliveriesResponse{
		Deliveries: protoDels,
		Total:      int32(total),
	}, nil
}

// ============================================================================
// Stats RPC
// ============================================================================

func (s *FormulareGRPCServer) GetFormStats(ctx context.Context, req *formularev1.GetFormStatsRequest) (*formularev1.GetFormStatsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	formSchemaID, err := uuid.Parse(req.GetFormSchemaId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid form_schema_id: %v", err)
	}

	stats, err := s.svc.GetFormStats(ctx, formulare.GetFormStatsInput{
		TenantID:     tenantID,
		FormSchemaID: formSchemaID,
	})
	if err != nil {
		return nil, mapFormulareError(err)
	}

	return &formularev1.GetFormStatsResponse{
		Stats: &formularev1.FormStats{
			FormSchemaId:  stats.FormSchemaID.String(),
			TotalCount:    int32(stats.TotalCount),
			NewCount:      int32(stats.NewCount),
			Last_7DCount:  int32(stats.Last7dCount),
			Last_30DCount: int32(stats.Last30dCount),
		},
	}, nil
}

// ============================================================================
// Conversion helpers — domain → proto
// ============================================================================

func toProtoFormSchema(s *formulare.FormSchema) *formularev1.FormSchema {
	if s == nil {
		return nil
	}
	p := &formularev1.FormSchema{
		Id:              s.ID.String(),
		TenantId:        s.TenantID.String(),
		Title:           s.Title,
		Description:     s.Description,
		Fields:          s.Fields,
		Status:          toProtoFormSchemaStatus(s.Status),
		IsTemplate:      s.IsTemplate,
		IsPublic:        s.IsPublic,
		PageCount:       int32(s.PageCount),
		SubmissionCount: int32(s.SubmissionCount),
		CreatedAt:       timestamppb.New(s.CreatedAt),
		UpdatedAt:       timestamppb.New(s.UpdatedAt),
	}
	if s.CreatedBy != nil {
		p.CreatedBy = s.CreatedBy.String()
	}
	if s.DeletedAt != nil && !s.DeletedAt.IsZero() {
		p.DeletedAt = timestamppb.New(*s.DeletedAt)
	}
	return p
}

func toProtoFormSubmission(sub *formulare.FormSubmission) *formularev1.FormSubmission {
	if sub == nil {
		return nil
	}
	p := &formularev1.FormSubmission{
		Id:          sub.ID.String(),
		TenantId:    sub.TenantID.String(),
		Answers:     sub.Answers,
		Status:      toProtoSubmissionStatus(sub.Status),
		SubmittedAt: timestamppb.New(sub.SubmittedAt),
	}
	if sub.FormSchemaID != nil {
		p.FormSchemaId = sub.FormSchemaID.String()
	}
	if sub.SubmittedBy != nil {
		p.SubmittedBy = *sub.SubmittedBy
	}
	if sub.IPAddress != nil {
		p.IpAddress = *sub.IPAddress
	}
	return p
}

func toProtoFormWebhook(w *formulare.FormWebhook) *formularev1.FormWebhook {
	if w == nil {
		return nil
	}
	p := &formularev1.FormWebhook{
		Id:           w.ID.String(),
		FormSchemaId: w.FormSchemaID.String(),
		TenantId:     w.TenantID.String(),
		Url:          w.URL,
		Events:       w.Events,
		Active:       w.Active,
		CreatedAt:    timestamppb.New(w.CreatedAt),
		UpdatedAt:    timestamppb.New(w.UpdatedAt),
	}
	if w.Secret != nil {
		p.Secret = *w.Secret
	}
	if w.LastTriggeredAt != nil && !w.LastTriggeredAt.IsZero() {
		p.LastTriggeredAt = timestamppb.New(*w.LastTriggeredAt)
	}
	if w.LastStatus != nil {
		p.LastStatus = *w.LastStatus
	}
	return p
}

func toProtoWebhookDelivery(d *formulare.WebhookDelivery) *formularev1.WebhookDelivery {
	if d == nil {
		return nil
	}
	p := &formularev1.WebhookDelivery{
		Id:           d.ID.String(),
		WebhookId:    d.WebhookID.String(),
		SubmissionId: d.SubmissionID.String(),
		TenantId:     d.TenantID.String(),
		Payload:      d.Payload,
		Status:       toProtoDeliveryStatus(d.Status),
		AttemptCount: int32(d.AttemptCount),
		MaxAttempts:  int32(d.MaxAttempts),
		CreatedAt:    timestamppb.New(d.CreatedAt),
	}
	if !d.NextAttemptAt.IsZero() {
		p.NextAttemptAt = timestamppb.New(d.NextAttemptAt)
	}
	if d.LastError != nil {
		p.LastError = *d.LastError
	}
	if d.LastResponseCode != nil {
		p.LastResponseCode = int32(*d.LastResponseCode)
	}
	if d.DeliveredAt != nil && !d.DeliveredAt.IsZero() {
		p.DeliveredAt = timestamppb.New(*d.DeliveredAt)
	}
	return p
}

// ============================================================================
// Conversion helpers — proto → domain
// ============================================================================

func fromProtoFormSchemaStatus(s formularev1.FormSchemaStatus) formulare.FormSchemaStatus {
	switch s {
	case formularev1.FormSchemaStatus_FORM_SCHEMA_STATUS_ACTIVE:
		return formulare.FormSchemaStatusActive
	case formularev1.FormSchemaStatus_FORM_SCHEMA_STATUS_ARCHIVED:
		return formulare.FormSchemaStatusArchived
	default:
		return formulare.FormSchemaStatusDraft
	}
}

func toProtoFormSchemaStatus(s formulare.FormSchemaStatus) formularev1.FormSchemaStatus {
	switch s {
	case formulare.FormSchemaStatusActive:
		return formularev1.FormSchemaStatus_FORM_SCHEMA_STATUS_ACTIVE
	case formulare.FormSchemaStatusArchived:
		return formularev1.FormSchemaStatus_FORM_SCHEMA_STATUS_ARCHIVED
	default:
		return formularev1.FormSchemaStatus_FORM_SCHEMA_STATUS_DRAFT
	}
}

func fromProtoSubmissionStatus(s formularev1.FormSubmissionStatus) formulare.FormSubmissionStatus {
	switch s {
	case formularev1.FormSubmissionStatus_FORM_SUBMISSION_STATUS_READ:
		return formulare.FormSubmissionStatusRead
	case formularev1.FormSubmissionStatus_FORM_SUBMISSION_STATUS_ARCHIVED:
		return formulare.FormSubmissionStatusArchived
	default:
		return formulare.FormSubmissionStatusNew
	}
}

func toProtoSubmissionStatus(s formulare.FormSubmissionStatus) formularev1.FormSubmissionStatus {
	switch s {
	case formulare.FormSubmissionStatusRead:
		return formularev1.FormSubmissionStatus_FORM_SUBMISSION_STATUS_READ
	case formulare.FormSubmissionStatusArchived:
		return formularev1.FormSubmissionStatus_FORM_SUBMISSION_STATUS_ARCHIVED
	default:
		return formularev1.FormSubmissionStatus_FORM_SUBMISSION_STATUS_NEW
	}
}

func fromProtoDeliveryStatus(s formularev1.WebhookDeliveryStatus) formulare.WebhookDeliveryStatus {
	switch s {
	case formularev1.WebhookDeliveryStatus_WEBHOOK_DELIVERY_STATUS_DELIVERED:
		return formulare.WebhookDeliveryStatusDelivered
	case formularev1.WebhookDeliveryStatus_WEBHOOK_DELIVERY_STATUS_FAILED:
		return formulare.WebhookDeliveryStatusFailed
	case formularev1.WebhookDeliveryStatus_WEBHOOK_DELIVERY_STATUS_DEAD:
		return formulare.WebhookDeliveryStatusDead
	default:
		return formulare.WebhookDeliveryStatusPending
	}
}

func toProtoDeliveryStatus(s formulare.WebhookDeliveryStatus) formularev1.WebhookDeliveryStatus {
	switch s {
	case formulare.WebhookDeliveryStatusDelivered:
		return formularev1.WebhookDeliveryStatus_WEBHOOK_DELIVERY_STATUS_DELIVERED
	case formulare.WebhookDeliveryStatusFailed:
		return formularev1.WebhookDeliveryStatus_WEBHOOK_DELIVERY_STATUS_FAILED
	case formulare.WebhookDeliveryStatusDead:
		return formularev1.WebhookDeliveryStatus_WEBHOOK_DELIVERY_STATUS_DEAD
	default:
		return formularev1.WebhookDeliveryStatus_WEBHOOK_DELIVERY_STATUS_PENDING
	}
}

func fromProtoExportFormat(f formularev1.ExportFormat) formulare.ExportFormat {
	switch f {
	case formularev1.ExportFormat_EXPORT_FORMAT_XLSX:
		return formulare.ExportFormatXLSX
	default:
		return formulare.ExportFormatCSV
	}
}

// ============================================================================
// Error mapping
// ============================================================================

func mapFormulareError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, formulare.ErrSchemaNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, formulare.ErrSubmissionNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, formulare.ErrWebhookNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, formulare.ErrDeliveryNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, formulare.ErrSchemaDeleted):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, formulare.ErrInvalidFields):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, formulare.ErrInvalidStatus):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, formulare.ErrInvalidURL):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, formulare.ErrInvalidExportFormat):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, formulare.ErrWebhookInactive):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
