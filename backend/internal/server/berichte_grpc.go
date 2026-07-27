package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/berichte"
	berichtev1 "github.com/kmuhub/kmuhub/proto/berichte/v1"
)

// ============================================================================
// Exporter stub interface
// ============================================================================

// BerichteExporter renders a ReportResult to the given writer.
// The concrete implementation lives in backend/internal/berichte/export/
// and is wired in after WP-3 merges (wave 3).
type BerichteExporter interface {
	Export(result *berichte.ReportResult, w io.Writer) error
	ContentType() string
	FileExtension() string
}

// BerichteExporterFactory returns an exporter for the given format (pdf|csv|xlsx).
// Use export.NewExporter from the berichte/export package as the concrete factory.
type BerichteExporterFactory func(format string) (BerichteExporter, error)

// ============================================================================
// Server
// ============================================================================

// BerichteGRPCServer implements the BerichteService gRPC server.
type BerichteGRPCServer struct {
	berichtev1.UnimplementedBerichteServiceServer
	svc             *berichte.Service
	exporterFactory BerichteExporterFactory
}

// NewBerichteGRPCServer creates a new Berichte gRPC server.
// exporterFactory must be non-nil; pass export.NewExporter (berichte/export package)
// as the factory. A nil factory causes ExportReport to return codes.Internal.
func NewBerichteGRPCServer(svc *berichte.Service, exporterFactory BerichteExporterFactory) *BerichteGRPCServer {
	return &BerichteGRPCServer{
		svc:             svc,
		exporterFactory: exporterFactory,
	}
}

// ============================================================================
// Definition RPCs
// ============================================================================

func (s *BerichteGRPCServer) CreateDefinition(ctx context.Context, req *berichtev1.CreateDefinitionRequest) (*berichtev1.DefinitionResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := berichte.CreateDefinitionInput{
		TenantID:      tenantID,
		Name:          req.GetName(),
		Description:   req.GetDescription(),
		Module:        req.GetModule(),
		Kind:          req.GetKind(),
		QueryConfig:   json.RawMessage(req.GetQueryConfig()),
		DefaultFormat: req.GetDefaultFormat(),
		IsPublished:   req.GetIsPublished(),
	}
	if req.CreatedBy != nil {
		id, parseErr := uuid.Parse(*req.CreatedBy)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid created_by: %v", parseErr)
		}
		input.CreatedBy = &id
	}

	def, err := s.svc.CreateDefinition(ctx, input)
	if err != nil {
		return nil, mapBerichteError(err)
	}
	return &berichtev1.DefinitionResponse{Definition: definitionToProto(def)}, nil
}

func (s *BerichteGRPCServer) GetDefinition(ctx context.Context, req *berichtev1.GetDefinitionRequest) (*berichtev1.DefinitionResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	definitionID, err := uuid.Parse(req.GetDefinitionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid definition_id: %v", err)
	}

	def, err := s.svc.GetDefinition(ctx, tenantID, definitionID)
	if err != nil {
		return nil, mapBerichteError(err)
	}
	return &berichtev1.DefinitionResponse{Definition: definitionToProto(def)}, nil
}

func (s *BerichteGRPCServer) UpdateDefinition(ctx context.Context, req *berichtev1.UpdateDefinitionRequest) (*berichtev1.DefinitionResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	definitionID, err := uuid.Parse(req.GetDefinitionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid definition_id: %v", err)
	}

	input := berichte.UpdateDefinitionInput{
		TenantID:     tenantID,
		DefinitionID: definitionID,
		QueryConfig:  json.RawMessage(req.GetQueryConfig()),
	}
	if req.Name != nil {
		input.Name = req.Name
	}
	if req.Description != nil {
		input.Description = req.Description
	}
	if req.Module != nil {
		input.Module = req.Module
	}
	if req.DefaultFormat != nil {
		input.DefaultFormat = req.DefaultFormat
	}
	if req.IsPublished != nil {
		input.IsPublished = req.IsPublished
	}

	def, err := s.svc.UpdateDefinition(ctx, input)
	if err != nil {
		return nil, mapBerichteError(err)
	}
	return &berichtev1.DefinitionResponse{Definition: definitionToProto(def)}, nil
}

func (s *BerichteGRPCServer) DeleteDefinition(ctx context.Context, req *berichtev1.DeleteDefinitionRequest) (*berichtev1.DeleteDefinitionResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	definitionID, err := uuid.Parse(req.GetDefinitionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid definition_id: %v", err)
	}

	if err := s.svc.DeleteDefinition(ctx, tenantID, definitionID); err != nil {
		return nil, mapBerichteError(err)
	}
	return &berichtev1.DeleteDefinitionResponse{}, nil
}

func (s *BerichteGRPCServer) ListDefinitions(ctx context.Context, req *berichtev1.ListDefinitionsRequest) (*berichtev1.ListDefinitionsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := berichte.ListDefinitionsInput{
		TenantID: tenantID,
		Search:   req.GetSearch(),
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
		SortBy:   req.GetSortBy(),
		SortDesc: req.GetSortDesc(),
	}
	if req.Module != nil {
		input.Module = req.Module
	}
	if req.Kind != nil {
		input.Kind = req.Kind
	}
	if req.IsPublished != nil {
		input.IsPublished = req.IsPublished
	}

	defs, total, err := s.svc.ListDefinitions(ctx, input)
	if err != nil {
		return nil, mapBerichteError(err)
	}

	protoDefs := make([]*berichtev1.Definition, len(defs))
	for i, d := range defs {
		protoDefs[i] = definitionToProto(d)
	}
	return &berichtev1.ListDefinitionsResponse{
		Definitions: protoDefs,
		Total:       int32(total),
	}, nil
}

// ============================================================================
// Run & Cache RPCs
// ============================================================================

func (s *BerichteGRPCServer) RunReport(ctx context.Context, req *berichtev1.RunReportRequest) (*berichtev1.RunReportResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	definitionID, err := uuid.Parse(req.GetDefinitionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid definition_id: %v", err)
	}

	input := berichte.RunReportInput{
		TenantID:     tenantID,
		DefinitionID: definitionID,
		Params:       json.RawMessage(req.GetParams()),
		ForceRefresh: req.GetForceRefresh(),
		Trigger:      req.GetTrigger(),
	}

	result, run, err := s.svc.RunReport(ctx, input)
	if err != nil {
		return nil, mapBerichteError(err)
	}
	return &berichtev1.RunReportResponse{
		Result: resultToProto(result),
		Run:    runToProto(run),
	}, nil
}

func (s *BerichteGRPCServer) GetCachedResult(ctx context.Context, req *berichtev1.GetCachedResultRequest) (*berichtev1.RunReportResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	definitionID, err := uuid.Parse(req.GetDefinitionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid definition_id: %v", err)
	}

	result, err := s.svc.GetCachedResult(ctx, tenantID, definitionID, json.RawMessage(req.GetParams()))
	if err != nil {
		return nil, mapBerichteError(err)
	}
	return &berichtev1.RunReportResponse{
		Result: resultToProto(result),
	}, nil
}

func (s *BerichteGRPCServer) InvalidateCache(ctx context.Context, req *berichtev1.InvalidateCacheRequest) (*berichtev1.InvalidateCacheResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	definitionID, err := uuid.Parse(req.GetDefinitionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid definition_id: %v", err)
	}

	evicted, err := s.svc.InvalidateCache(ctx, tenantID, definitionID)
	if err != nil {
		return nil, mapBerichteError(err)
	}
	return &berichtev1.InvalidateCacheResponse{Evicted: int32(evicted)}, nil
}

// ============================================================================
// Export RPC
// ============================================================================

func (s *BerichteGRPCServer) ExportReport(ctx context.Context, req *berichtev1.ExportReportRequest) (*berichtev1.ExportReportResponse, error) {
	if s.exporterFactory == nil {
		// exporterFactory must always be set by the caller (see cmd/berichte/main.go).
		// Return Internal rather than Unimplemented so misconfiguration is surfaced
		// immediately as a server-side error rather than a "feature not available" signal.
		slog.Error("ExportReport called but exporterFactory is nil — check service wiring")
		return nil, status.Error(codes.Internal, "export not configured")
	}

	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	definitionID, err := uuid.Parse(req.GetDefinitionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid definition_id: %v", err)
	}

	format := req.GetFormat()
	if format == "" {
		format = "pdf"
	}

	exporter, err := s.exporterFactory(format)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported export format %q: %v", format, err)
	}

	result, _, runErr := s.svc.RunReport(ctx, berichte.RunReportInput{
		TenantID:     tenantID,
		DefinitionID: definitionID,
		Params:       json.RawMessage(req.GetParams()),
		Trigger:      "api",
	})
	if runErr != nil {
		return nil, mapBerichteError(runErr)
	}

	var buf bytes.Buffer
	if exportErr := exporter.Export(result, &buf); exportErr != nil {
		return nil, status.Errorf(codes.Internal, "export failed: %v", exportErr)
	}

	filename := fmt.Sprintf("bericht-%s.%s", definitionID.String(), exporter.FileExtension())
	return &berichtev1.ExportReportResponse{
		Payload:     buf.Bytes(),
		Filename:    filename,
		ContentType: exporter.ContentType(),
	}, nil
}

// ============================================================================
// Schedule RPCs
// ============================================================================

func (s *BerichteGRPCServer) CreateSchedule(ctx context.Context, req *berichtev1.CreateScheduleRequest) (*berichtev1.ScheduleResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	definitionID, err := uuid.Parse(req.GetDefinitionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid definition_id: %v", err)
	}

	input := berichte.CreateScheduleInput{
		TenantID:       tenantID,
		DefinitionID:   definitionID,
		Name:           req.GetName(),
		CronExpression: req.GetCronExpression(),
		Recipients:     req.GetRecipients(),
		Format:         req.GetFormat(),
		Params:         json.RawMessage(req.GetParams()),
		Active:         req.GetActive(),
	}
	if req.CreatedBy != nil {
		id, parseErr := uuid.Parse(*req.CreatedBy)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid created_by: %v", parseErr)
		}
		input.CreatedBy = &id
	}

	sch, err := s.svc.CreateSchedule(ctx, input)
	if err != nil {
		return nil, mapBerichteError(err)
	}
	return &berichtev1.ScheduleResponse{Schedule: scheduleToProto(sch)}, nil
}

func (s *BerichteGRPCServer) UpdateSchedule(ctx context.Context, req *berichtev1.UpdateScheduleRequest) (*berichtev1.ScheduleResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	scheduleID, err := uuid.Parse(req.GetScheduleId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid schedule_id: %v", err)
	}

	input := berichte.UpdateScheduleInput{
		TenantID:      tenantID,
		ScheduleID:    scheduleID,
		Recipients:    req.GetRecipients(),
		RecipientsSet: req.GetRecipientsSet(),
		Params:        json.RawMessage(req.GetParams()),
	}
	if req.Name != nil {
		input.Name = req.Name
	}
	if req.CronExpression != nil {
		input.CronExpression = req.CronExpression
	}
	if req.Format != nil {
		input.Format = req.Format
	}
	if req.Active != nil {
		input.Active = req.Active
	}

	sch, err := s.svc.UpdateSchedule(ctx, input)
	if err != nil {
		return nil, mapBerichteError(err)
	}
	return &berichtev1.ScheduleResponse{Schedule: scheduleToProto(sch)}, nil
}

func (s *BerichteGRPCServer) DeleteSchedule(ctx context.Context, req *berichtev1.DeleteScheduleRequest) (*berichtev1.DeleteScheduleResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	scheduleID, err := uuid.Parse(req.GetScheduleId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid schedule_id: %v", err)
	}

	if err := s.svc.DeleteSchedule(ctx, tenantID, scheduleID); err != nil {
		return nil, mapBerichteError(err)
	}
	return &berichtev1.DeleteScheduleResponse{}, nil
}

func (s *BerichteGRPCServer) ListSchedules(ctx context.Context, req *berichtev1.ListSchedulesRequest) (*berichtev1.ListSchedulesResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := berichte.ListSchedulesInput{
		TenantID: tenantID,
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	}
	if req.DefinitionId != nil {
		id, parseErr := uuid.Parse(*req.DefinitionId)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid definition_id: %v", parseErr)
		}
		input.DefinitionID = &id
	}
	if req.Active != nil {
		input.Active = req.Active
	}

	scheds, total, err := s.svc.ListSchedules(ctx, input)
	if err != nil {
		return nil, mapBerichteError(err)
	}

	protoScheds := make([]*berichtev1.Schedule, len(scheds))
	for i, sc := range scheds {
		protoScheds[i] = scheduleToProto(sc)
	}
	return &berichtev1.ListSchedulesResponse{
		Schedules: protoScheds,
		Total:     int32(total),
	}, nil
}

func (s *BerichteGRPCServer) ToggleSchedule(ctx context.Context, req *berichtev1.ToggleScheduleRequest) (*berichtev1.ScheduleResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	scheduleID, err := uuid.Parse(req.GetScheduleId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid schedule_id: %v", err)
	}

	sch, err := s.svc.ToggleSchedule(ctx, tenantID, scheduleID, req.GetActive())
	if err != nil {
		return nil, mapBerichteError(err)
	}
	return &berichtev1.ScheduleResponse{Schedule: scheduleToProto(sch)}, nil
}

// ============================================================================
// Dashboard KPIs RPC
// ============================================================================

func (s *BerichteGRPCServer) GetDashboardKPIs(ctx context.Context, req *berichtev1.DashboardKPIsRequest) (*berichtev1.DashboardKPIsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	kpis, err := s.svc.GetDashboardKPIs(ctx, tenantID, req.GetModules())
	if err != nil {
		return nil, mapBerichteError(err)
	}

	protoKPIs := make([]*berichtev1.KPI, len(kpis))
	for i, k := range kpis {
		protoKPIs[i] = kpiToProto(&k)
	}
	return &berichtev1.DashboardKPIsResponse{
		Kpis:        protoKPIs,
		GeneratedAt: timestamppb.New(time.Now()),
	}, nil
}

// ============================================================================
// Conversion helpers
// ============================================================================

// ============================================================================
// Document RPCs (multi-page authoring)
// ============================================================================

func (s *BerichteGRPCServer) CreateDocument(ctx context.Context, req *berichtev1.CreateDocumentRequest) (*berichtev1.DocumentResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := berichte.CreateDocumentInput{
		TenantID:    tenantID,
		Title:       req.GetTitle(),
		Description: req.GetDescription(),
		Module:      req.GetModule(),
		TemplateID:  req.TemplateId,
		Rows:        json.RawMessage(req.GetRows()),
		Settings:    json.RawMessage(req.GetSettings()),
	}
	if req.CreatedBy != nil {
		id, parseErr := uuid.Parse(*req.CreatedBy)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid created_by: %v", parseErr)
		}
		input.CreatedBy = &id
	}

	doc, err := s.svc.CreateDocument(ctx, input)
	if err != nil {
		return nil, mapBerichteError(err)
	}
	return &berichtev1.DocumentResponse{Document: documentToProto(doc)}, nil
}

func (s *BerichteGRPCServer) GetDocument(ctx context.Context, req *berichtev1.GetDocumentRequest) (*berichtev1.DocumentResponse, error) {
	tenantID, documentID, err := parseTenantAndDocumentID(req.GetTenantId(), req.GetDocumentId())
	if err != nil {
		return nil, err
	}

	doc, err := s.svc.GetDocument(ctx, tenantID, documentID)
	if err != nil {
		return nil, mapBerichteError(err)
	}
	return &berichtev1.DocumentResponse{Document: documentToProto(doc)}, nil
}

func (s *BerichteGRPCServer) UpdateDocument(ctx context.Context, req *berichtev1.UpdateDocumentRequest) (*berichtev1.DocumentResponse, error) {
	tenantID, documentID, err := parseTenantAndDocumentID(req.GetTenantId(), req.GetDocumentId())
	if err != nil {
		return nil, err
	}

	doc, err := s.svc.UpdateDocument(ctx, berichte.UpdateDocumentInput{
		TenantID:    tenantID,
		DocumentID:  documentID,
		Title:       req.Title,
		Description: req.Description,
		Module:      req.Module,
		Status:      req.Status,
		Rows:        json.RawMessage(req.GetRows()),
		Settings:    json.RawMessage(req.GetSettings()),
	})
	if err != nil {
		return nil, mapBerichteError(err)
	}
	return &berichtev1.DocumentResponse{Document: documentToProto(doc)}, nil
}

func (s *BerichteGRPCServer) DeleteDocument(ctx context.Context, req *berichtev1.DeleteDocumentRequest) (*berichtev1.DeleteDocumentResponse, error) {
	tenantID, documentID, err := parseTenantAndDocumentID(req.GetTenantId(), req.GetDocumentId())
	if err != nil {
		return nil, err
	}

	if err := s.svc.DeleteDocument(ctx, tenantID, documentID); err != nil {
		return nil, mapBerichteError(err)
	}
	return &berichtev1.DeleteDocumentResponse{}, nil
}

func (s *BerichteGRPCServer) ListDocuments(ctx context.Context, req *berichtev1.ListDocumentsRequest) (*berichtev1.ListDocumentsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	docs, total, err := s.svc.ListDocuments(ctx, berichte.ListDocumentsInput{
		TenantID: tenantID,
		Module:   req.Module,
		Status:   req.Status,
		Search:   req.GetSearch(),
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	})
	if err != nil {
		return nil, mapBerichteError(err)
	}

	out := make([]*berichtev1.Document, 0, len(docs))
	for _, d := range docs {
		out = append(out, documentToProto(d))
	}
	return &berichtev1.ListDocumentsResponse{Documents: out, Total: int32(total)}, nil
}

// ============================================================================
// Template RPC
// ============================================================================

func (s *BerichteGRPCServer) ListTemplates(ctx context.Context, req *berichtev1.ListTemplatesRequest) (*berichtev1.ListTemplatesResponse, error) {
	templates := s.svc.ListTemplates()
	out := make([]*berichtev1.Template, 0, len(templates))
	for _, t := range templates {
		out = append(out, berichteTemplateToProto(&t))
	}
	return &berichtev1.ListTemplatesResponse{Templates: out}, nil
}

func berichteTemplateToProto(t *berichte.Template) *berichtev1.Template {
	if t == nil {
		return nil
	}
	proto := &berichtev1.Template{
		Id:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		Module:      t.Module,
		Icon:        t.Icon,
		Rows:        t.Rows,
		Settings:    t.Settings,
	}
	return proto
}

func parseTenantAndDocumentID(rawTenantID, rawDocumentID string) (uuid.UUID, uuid.UUID, error) {
	tenantID, err := uuid.Parse(rawTenantID)
	if err != nil {
		return uuid.Nil, uuid.Nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	documentID, err := uuid.Parse(rawDocumentID)
	if err != nil {
		return uuid.Nil, uuid.Nil, status.Errorf(codes.InvalidArgument, "invalid document_id: %v", err)
	}
	return tenantID, documentID, nil
}

func documentToProto(d *berichte.Document) *berichtev1.Document {
	if d == nil {
		return nil
	}
	proto := &berichtev1.Document{
		Id:          d.ID.String(),
		TenantId:    d.TenantID.String(),
		Title:       d.Title,
		Description: d.Description,
		Module:      d.Module,
		Status:      d.Status,
		Rows:        d.Rows,
		Settings:    d.Settings,
		TemplateId:  d.TemplateID,
		CreatedAt:   timestamppb.New(d.CreatedAt),
		UpdatedAt:   timestamppb.New(d.UpdatedAt),
	}
	if d.CreatedBy != nil {
		s := d.CreatedBy.String()
		proto.CreatedBy = &s
	}
	if d.ReleasedAt != nil {
		proto.ReleasedAt = timestamppb.New(*d.ReleasedAt)
	}
	return proto
}

func definitionToProto(d *berichte.Definition) *berichtev1.Definition {
	if d == nil {
		return nil
	}
	proto := &berichtev1.Definition{
		Id:            d.ID.String(),
		TenantId:      d.TenantID.String(),
		Name:          d.Name,
		Description:   d.Description,
		Module:        d.Module,
		Kind:          d.Kind,
		QueryConfig:   d.QueryConfig,
		DefaultFormat: d.DefaultFormat,
		IsPublished:   d.IsPublished,
		CreatedAt:     timestamppb.New(d.CreatedAt),
		UpdatedAt:     timestamppb.New(d.UpdatedAt),
	}
	if d.CreatedBy != nil {
		s := d.CreatedBy.String()
		proto.CreatedBy = &s
	}
	return proto
}

func scheduleToProto(sc *berichte.Schedule) *berichtev1.Schedule {
	if sc == nil {
		return nil
	}
	proto := &berichtev1.Schedule{
		Id:             sc.ID.String(),
		TenantId:       sc.TenantID.String(),
		DefinitionId:   sc.DefinitionID.String(),
		Name:           sc.Name,
		CronExpression: sc.CronExpression,
		Recipients:     sc.Recipients,
		Format:         sc.Format,
		Params:         sc.Params,
		Active:         sc.Active,
		LastRunStatus:  sc.LastRunStatus,
		LastRunError:   sc.LastRunError,
		CreatedAt:      timestamppb.New(sc.CreatedAt),
		UpdatedAt:      timestamppb.New(sc.UpdatedAt),
	}
	if sc.LastRunAt != nil && !sc.LastRunAt.IsZero() {
		proto.LastRunAt = timestamppb.New(*sc.LastRunAt)
	}
	if sc.CreatedBy != nil {
		s := sc.CreatedBy.String()
		proto.CreatedBy = &s
	}
	return proto
}

func runToProto(r *berichte.Run) *berichtev1.Run {
	if r == nil {
		return nil
	}
	proto := &berichtev1.Run{
		Id:           r.ID.String(),
		TenantId:     r.TenantID.String(),
		DefinitionId: r.DefinitionID.String(),
		Trigger:      r.Trigger,
		Params:       r.Params,
		Status:       r.Status,
		Error:        r.Error,
		StartedAt:    timestamppb.New(r.StartedAt),
	}
	if r.ScheduleID != nil {
		s := r.ScheduleID.String()
		proto.ScheduleId = &s
	}
	if r.DurationMs != nil {
		v := int32(*r.DurationMs)
		proto.DurationMs = &v
	}
	if r.RowCount != nil {
		v := int32(*r.RowCount)
		proto.RowCount = &v
	}
	if r.CompletedAt != nil && !r.CompletedAt.IsZero() {
		proto.CompletedAt = timestamppb.New(*r.CompletedAt)
	}
	return proto
}

func resultToProto(r *berichte.ReportResult) *berichtev1.ReportResult {
	if r == nil {
		return nil
	}
	payload, _ := json.Marshal(r)
	proto := &berichtev1.ReportResult{
		Payload:   payload,
		RowCount:  int32(r.Meta.RowCount),
		FromCache: r.Meta.FromCache,
	}
	if !r.Meta.GeneratedAt.IsZero() {
		proto.GeneratedAt = timestamppb.New(r.Meta.GeneratedAt)
	}
	return proto
}

func kpiToProto(k *berichte.KPI) *berichtev1.KPI {
	if k == nil {
		return nil
	}
	proto := &berichtev1.KPI{
		Id:       k.ID,
		Label:    k.Label,
		Value:    k.Value,
		Unit:     k.Unit,
		ModuleId: k.ModuleID,
	}
	if k.ChangePercent != nil {
		proto.ChangePercent = k.ChangePercent
	}
	return proto
}

// ============================================================================
// Error mapping
// ============================================================================

func mapBerichteError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, berichte.ErrDefinitionNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, berichte.ErrScheduleNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, berichte.ErrCacheMiss):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, berichte.ErrInvalidQueryConfig):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, berichte.ErrInvalidCron):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, berichte.ErrInvalidFormat):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, berichte.ErrInvalidModule):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, berichte.ErrInvalidKind):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, berichte.ErrExecutorUnavailable):
		return status.Error(codes.Unavailable, err.Error())
	case errors.Is(err, berichte.ErrDocumentNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, berichte.ErrInvalidTitle),
		errors.Is(err, berichte.ErrInvalidStatus),
		errors.Is(err, berichte.ErrInvalidRows),
		errors.Is(err, berichte.ErrInvalidSettings),
		errors.Is(err, berichte.ErrDocumentTooLarge):
		return status.Error(codes.InvalidArgument, err.Error())
	// Unknown, expired and revoked share links all land here as the same
	// NotFound the gateway turns into a 404. Nothing in this branch tells an
	// unauthenticated caller which of the three it hit.
	case errors.Is(err, berichte.ErrShareNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, berichte.ErrSharePasswordRequired),
		errors.Is(err, berichte.ErrSharePasswordInvalid):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, berichte.ErrInvalidShareExpiry),
		errors.Is(err, berichte.ErrSharePasswordTooLong):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		slog.Error("unhandled berichte service error", "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}

// ============================================================================
// Share tokens
// ============================================================================

func (s *BerichteGRPCServer) CreateShareToken(ctx context.Context, req *berichtev1.CreateShareTokenRequest) (*berichtev1.ShareTokenResponse, error) {
	tenantID, documentID, err := parseTenantAndDocumentID(req.GetTenantId(), req.GetDocumentId())
	if err != nil {
		return nil, err
	}

	in := berichte.CreateShareTokenInput{
		TenantID:   tenantID,
		DocumentID: documentID,
		Password:   req.GetPassword(),
	}
	if req.ExpiresInDays != nil {
		in.ExpiresInDays = req.ExpiresInDays
	}
	if raw := req.GetCreatedBy(); raw != "" {
		createdBy, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid created_by: %v", parseErr)
		}
		in.CreatedBy = &createdBy
	}

	token, err := s.svc.CreateShareToken(ctx, in)
	if err != nil {
		return nil, mapBerichteError(err)
	}
	return &berichtev1.ShareTokenResponse{Share: reportShareTokenToProto(token)}, nil
}

func (s *BerichteGRPCServer) ListShareTokens(ctx context.Context, req *berichtev1.ListShareTokensRequest) (*berichtev1.ListShareTokensResponse, error) {
	tenantID, documentID, err := parseTenantAndDocumentID(req.GetTenantId(), req.GetDocumentId())
	if err != nil {
		return nil, err
	}

	tokens, err := s.svc.ListShareTokens(ctx, tenantID, documentID)
	if err != nil {
		return nil, mapBerichteError(err)
	}

	protoTokens := make([]*berichtev1.ShareToken, len(tokens))
	for i, t := range tokens {
		protoTokens[i] = reportShareTokenToProto(t)
	}
	return &berichtev1.ListShareTokensResponse{Tokens: protoTokens}, nil
}

func (s *BerichteGRPCServer) RevokeShareToken(ctx context.Context, req *berichtev1.RevokeShareTokenRequest) (*berichtev1.RevokeShareTokenResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	shareID, err := uuid.Parse(req.GetShareId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid share_id: %v", err)
	}

	if err := s.svc.RevokeShareToken(ctx, tenantID, shareID); err != nil {
		return nil, mapBerichteError(err)
	}
	return &berichtev1.RevokeShareTokenResponse{}, nil
}

// GetSharedDocument is the only RPC in this service that takes no tenant_id.
// The caller is an anonymous visitor holding a link; the service resolves the
// tenant from the link itself and scopes everything after that to it.
func (s *BerichteGRPCServer) GetSharedDocument(ctx context.Context, req *berichtev1.GetSharedDocumentRequest) (*berichtev1.GetSharedDocumentResponse, error) {
	doc, err := s.svc.GetSharedDocument(ctx, req.GetToken(), req.GetPassword())
	if err != nil {
		return nil, mapBerichteError(err)
	}
	return &berichtev1.GetSharedDocumentResponse{Document: documentToProto(doc)}, nil
}

// reportShareTokenToProto maps a share link to the wire. PasswordHash has no field on
// the proto message at all — the bcrypt hash must not leave the service, and
// leaving it unmappable is a stronger guarantee than remembering not to set it.
func reportShareTokenToProto(t *berichte.ShareToken) *berichtev1.ShareToken {
	if t == nil {
		return nil
	}
	p := &berichtev1.ShareToken{
		Id:          t.ID.String(),
		DocumentId:  t.DocumentID.String(),
		Token:       t.Token,
		HasPassword: t.PasswordHash != nil,
		ViewCount:   int32(t.ViewCount),
		CreatedAt:   timestamppb.New(t.CreatedAt),
	}
	if t.ExpiresAt != nil {
		p.ExpiresAt = timestamppb.New(*t.ExpiresAt)
	}
	return p
}
