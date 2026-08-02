package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/rapporte"
	rapportev1 "github.com/kmuhub/kmuhub/proto/rapporte/v1"
)

// RapporteGRPCServer implements the RapporteService gRPC server.
type RapporteGRPCServer struct {
	rapportev1.UnimplementedRapporteServiceServer
	svc  *rapporte.Service
	repo rapporte.Repository
}

// NewRapporteGRPCServer creates a new Rapporte gRPC server.
func NewRapporteGRPCServer(svc *rapporte.Service, repo rapporte.Repository) *RapporteGRPCServer {
	return &RapporteGRPCServer{svc: svc, repo: repo}
}

// ============================================================================
// Report RPCs
// ============================================================================

func (s *RapporteGRPCServer) CreateReport(ctx context.Context, req *rapportev1.CreateReportRequest) (*rapportev1.ReportResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	authorID, err := uuid.Parse(req.GetAuthorId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid author_id: %v", err)
	}

	input := rapporte.CreateReportInput{
		TenantID:    tenantID,
		Title:       req.GetTitle(),
		Description: req.GetDescription(),
		AuthorID:    authorID,
		Lat:         req.Lat,
		Lon:         req.Lon,
	}
	if req.GetReportDate() != "" {
		rd, parseErr := time.Parse("2006-01-02", req.GetReportDate())
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid report_date: %v", parseErr)
		}
		input.ReportDate = &rd
	}

	rep, err := s.svc.CreateReport(ctx, input)
	if err != nil {
		return nil, mapRapporteError(err)
	}
	return &rapportev1.ReportResponse{Report: rapporteReportToProto(rep)}, nil
}

func (s *RapporteGRPCServer) GetReport(ctx context.Context, req *rapportev1.GetReportRequest) (*rapportev1.ReportResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	reportID, err := uuid.Parse(req.GetReportId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid report_id: %v", err)
	}

	rep, err := s.svc.GetReport(ctx, tenantID, reportID)
	if err != nil {
		return nil, mapRapporteError(err)
	}
	return &rapportev1.ReportResponse{Report: rapporteReportToProto(rep)}, nil
}

func (s *RapporteGRPCServer) UpdateReport(ctx context.Context, req *rapportev1.UpdateReportRequest) (*rapportev1.ReportResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	reportID, err := uuid.Parse(req.GetReportId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid report_id: %v", err)
	}

	input := rapporte.UpdateReportInput{
		TenantID:    tenantID,
		ReportID:    reportID,
		Title:       req.Title,
		Description: req.Description,
		Lat:         req.Lat,
		Lon:         req.Lon,
	}
	if req.ReportDate != nil {
		rd, parseErr := time.Parse("2006-01-02", *req.ReportDate)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid report_date: %v", parseErr)
		}
		input.ReportDate = &rd
	}

	rep, err := s.svc.UpdateReport(ctx, input)
	if err != nil {
		return nil, mapRapporteError(err)
	}
	return &rapportev1.ReportResponse{Report: rapporteReportToProto(rep)}, nil
}

func (s *RapporteGRPCServer) DeleteReport(ctx context.Context, req *rapportev1.DeleteReportRequest) (*rapportev1.DeleteReportResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	reportID, err := uuid.Parse(req.GetReportId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid report_id: %v", err)
	}

	if err := s.svc.DeleteReport(ctx, tenantID, reportID); err != nil {
		return nil, mapRapporteError(err)
	}
	return &rapportev1.DeleteReportResponse{}, nil
}

func (s *RapporteGRPCServer) ListReports(ctx context.Context, req *rapportev1.ListReportsRequest) (*rapportev1.ListReportsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}

	input := rapporte.ListReportsInput{
		TenantID: tenantID,
		Search:   req.GetSearch(),
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	}
	if req.Status != nil {
		st := rapporte.ReportStatus(*req.Status)
		input.Status = &st
	}
	if req.AuthorId != nil {
		aid, parseErr := uuid.Parse(*req.AuthorId)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid author_id: %v", parseErr)
		}
		input.AuthorID = &aid
	}

	reports, total, err := s.svc.ListReports(ctx, input)
	if err != nil {
		return nil, mapRapporteError(err)
	}

	protoReports := make([]*rapportev1.WorkReport, len(reports))
	for i, r := range reports {
		protoReports[i] = rapporteReportToProto(r)
	}
	return &rapportev1.ListReportsResponse{
		Reports: protoReports,
		Total:   int32(total),
	}, nil
}

// ============================================================================
// State Machine RPCs
// ============================================================================

func (s *RapporteGRPCServer) SubmitReport(ctx context.Context, req *rapportev1.SubmitReportRequest) (*rapportev1.ReportResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	reportID, err := uuid.Parse(req.GetReportId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid report_id: %v", err)
	}

	rep, err := s.svc.SubmitReport(ctx, rapporte.SubmitReportInput{
		TenantID: tenantID,
		ReportID: reportID,
	})
	if err != nil {
		return nil, mapRapporteError(err)
	}
	return &rapportev1.ReportResponse{Report: rapporteReportToProto(rep)}, nil
}

func (s *RapporteGRPCServer) ApproveReport(ctx context.Context, req *rapportev1.ApproveReportRequest) (*rapportev1.ReportResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	reportID, err := uuid.Parse(req.GetReportId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid report_id: %v", err)
	}
	reviewerID, err := uuid.Parse(req.GetReviewerId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid reviewer_id: %v", err)
	}

	rep, err := s.svc.ApproveReport(ctx, rapporte.ApproveReportInput{
		TenantID:   tenantID,
		ReportID:   reportID,
		ReviewerID: reviewerID,
		ReviewNote: req.GetReviewNote(),
	})
	if err != nil {
		return nil, mapRapporteError(err)
	}
	return &rapportev1.ReportResponse{Report: rapporteReportToProto(rep)}, nil
}

func (s *RapporteGRPCServer) RejectReport(ctx context.Context, req *rapportev1.RejectReportRequest) (*rapportev1.ReportResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	reportID, err := uuid.Parse(req.GetReportId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid report_id: %v", err)
	}
	reviewerID, err := uuid.Parse(req.GetReviewerId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid reviewer_id: %v", err)
	}

	rep, err := s.svc.RejectReport(ctx, rapporte.RejectReportInput{
		TenantID:   tenantID,
		ReportID:   reportID,
		ReviewerID: reviewerID,
		ReviewNote: req.GetReviewNote(),
	})
	if err != nil {
		return nil, mapRapporteError(err)
	}
	return &rapportev1.ReportResponse{Report: rapporteReportToProto(rep)}, nil
}

// ============================================================================
// Line RPCs
// ============================================================================

func (s *RapporteGRPCServer) AddLine(ctx context.Context, req *rapportev1.AddLineRequest) (*rapportev1.LineResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	reportID, err := uuid.Parse(req.GetReportId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid report_id: %v", err)
	}

	line, err := s.svc.AddLine(ctx, rapporte.AddLineInput{
		TenantID:    tenantID,
		ReportID:    reportID,
		Description: req.GetDescription(),
		Quantity:    req.GetQuantity(),
		Unit:        req.GetUnit(),
		Notes:       req.GetNotes(),
		Position:    int(req.GetPosition()),
	})
	if err != nil {
		return nil, mapRapporteError(err)
	}
	return &rapportev1.LineResponse{Line: rapporteLineToProto(line)}, nil
}

func (s *RapporteGRPCServer) UpdateLine(ctx context.Context, req *rapportev1.UpdateLineRequest) (*rapportev1.LineResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	lineID, err := uuid.Parse(req.GetLineId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid line_id: %v", err)
	}

	input := rapporte.UpdateLineInput{
		TenantID:    tenantID,
		LineID:      lineID,
		Description: req.Description,
		Unit:        req.Unit,
		Notes:       req.Notes,
	}
	if req.Quantity != nil {
		input.Quantity = req.Quantity
	}
	if req.Position != nil {
		pos := int(*req.Position)
		input.Position = &pos
	}

	line, err := s.svc.UpdateLine(ctx, input)
	if err != nil {
		return nil, mapRapporteError(err)
	}
	return &rapportev1.LineResponse{Line: rapporteLineToProto(line)}, nil
}

func (s *RapporteGRPCServer) DeleteLine(ctx context.Context, req *rapportev1.DeleteLineRequest) (*rapportev1.DeleteLineResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	lineID, err := uuid.Parse(req.GetLineId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid line_id: %v", err)
	}

	if err := s.svc.DeleteLine(ctx, tenantID, lineID); err != nil {
		return nil, mapRapporteError(err)
	}
	return &rapportev1.DeleteLineResponse{}, nil
}

func (s *RapporteGRPCServer) ListLines(ctx context.Context, req *rapportev1.ListLinesRequest) (*rapportev1.ListLinesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	reportID, err := uuid.Parse(req.GetReportId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid report_id: %v", err)
	}

	lines, err := s.svc.ListLines(ctx, tenantID, reportID)
	if err != nil {
		return nil, mapRapporteError(err)
	}

	protoLines := make([]*rapportev1.ReportLine, len(lines))
	for i, l := range lines {
		protoLines[i] = rapporteLineToProto(l)
	}
	return &rapportev1.ListLinesResponse{Lines: protoLines}, nil
}

// ============================================================================
// Attachment RPCs
// ============================================================================

func (s *RapporteGRPCServer) UploadAttachment(ctx context.Context, req *rapportev1.UploadAttachmentRequest) (*rapportev1.AttachmentResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	reportID, err := uuid.Parse(req.GetReportId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid report_id: %v", err)
	}
	uploadedBy, err := uuid.Parse(req.GetUploadedBy())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid uploaded_by: %v", err)
	}

	input := rapporte.UploadAttachmentInput{
		TenantID:    tenantID,
		ReportID:    reportID,
		Filename:    req.GetFilename(),
		ContentType: req.GetContentType(),
		SizeBytes:   req.GetSizeBytes(),
		ObjectKey:   req.GetObjectKey(),
		UploadedBy:  uploadedBy,
	}
	if req.LineId != nil {
		lid, parseErr := uuid.Parse(*req.LineId)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid line_id: %v", parseErr)
		}
		input.LineID = &lid
	}

	att, err := s.svc.UploadAttachment(ctx, input)
	if err != nil {
		return nil, mapRapporteError(err)
	}
	return &rapportev1.AttachmentResponse{Attachment: rapporteAttachmentToProto(att)}, nil
}

func (s *RapporteGRPCServer) ListAttachments(ctx context.Context, req *rapportev1.ListAttachmentsRequest) (*rapportev1.ListAttachmentsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	reportID, err := uuid.Parse(req.GetReportId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid report_id: %v", err)
	}

	var lineID *uuid.UUID
	if req.LineId != nil {
		lid, parseErr := uuid.Parse(*req.LineId)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid line_id: %v", parseErr)
		}
		lineID = &lid
	}

	atts, err := s.svc.ListAttachments(ctx, tenantID, reportID, lineID)
	if err != nil {
		return nil, mapRapporteError(err)
	}

	protoAtts := make([]*rapportev1.ReportAttachment, len(atts))
	for i, a := range atts {
		protoAtts[i] = rapporteAttachmentToProto(a)
	}
	return &rapportev1.ListAttachmentsResponse{Attachments: protoAtts}, nil
}

func (s *RapporteGRPCServer) DeleteAttachment(ctx context.Context, req *rapportev1.DeleteAttachmentRequest) (*rapportev1.DeleteAttachmentResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	attachmentID, err := uuid.Parse(req.GetAttachmentId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid attachment_id: %v", err)
	}

	if err := s.svc.DeleteAttachment(ctx, tenantID, attachmentID); err != nil {
		return nil, mapRapporteError(err)
	}
	return &rapportev1.DeleteAttachmentResponse{}, nil
}

// ============================================================================
// Signature RPC
// ============================================================================

func (s *RapporteGRPCServer) SaveSignature(ctx context.Context, req *rapportev1.SaveReportSignatureRequest) (*rapportev1.ReportResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	reportID, err := uuid.Parse(req.GetReportId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid report_id: %v", err)
	}

	report, err := s.svc.SaveSignature(ctx, tenantID, reportID, req.GetSignatureData(), req.GetSignedBy())
	if err != nil {
		return nil, mapRapporteError(err)
	}
	return &rapportev1.ReportResponse{Report: rapporteReportToProto(report)}, nil
}

// ============================================================================
// Stats & Export RPCs
// ============================================================================

func (s *RapporteGRPCServer) GetReportStats(ctx context.Context, req *rapportev1.GetReportStatsRequest) (*rapportev1.ReportStatsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}

	stats, err := s.svc.GetReportStats(ctx, tenantID)
	if err != nil {
		return nil, mapRapporteError(err)
	}
	return &rapportev1.ReportStatsResponse{
		TotalReports:   int32(stats.TotalReports),
		DraftCount:     int32(stats.DraftCount),
		SubmittedCount: int32(stats.SubmittedCount),
		ApprovedCount:  int32(stats.ApprovedCount),
		RejectedCount:  int32(stats.RejectedCount),
	}, nil
}

func (s *RapporteGRPCServer) ListPendingApprovals(ctx context.Context, req *rapportev1.ListPendingApprovalsRequest) (*rapportev1.ListReportsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}

	var authorID *uuid.UUID
	if req.AuthorId != nil {
		aid, parseErr := uuid.Parse(*req.AuthorId)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid author_id: %v", parseErr)
		}
		authorID = &aid
	}

	reports, total, err := s.svc.ListPendingApprovals(ctx, tenantID, authorID, int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		return nil, mapRapporteError(err)
	}

	protoReports := make([]*rapportev1.WorkReport, len(reports))
	for i, r := range reports {
		protoReports[i] = rapporteReportToProto(r)
	}
	return &rapportev1.ListReportsResponse{
		Reports: protoReports,
		Total:   int32(total),
	}, nil
}

func (s *RapporteGRPCServer) ExportPDF(ctx context.Context, req *rapportev1.ExportPDFRequest) (*rapportev1.ExportPDFResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	reportID, err := uuid.Parse(req.GetReportId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid report_id: %v", err)
	}

	payload, err := s.svc.ExportPDF(ctx, tenantID, reportID)
	if err != nil {
		return nil, mapRapporteError(err)
	}

	filename := fmt.Sprintf("arbeitsbericht_%s.pdf", reportID.String()[:8])
	return &rapportev1.ExportPDFResponse{
		Payload:     payload,
		ContentType: "application/pdf",
		Filename:    filename,
	}, nil
}

// ============================================================================
// Worker RPCs
// ============================================================================

func (s *RapporteGRPCServer) AddWorker(ctx context.Context, req *rapportev1.AddWorkerRequest) (*rapportev1.AddWorkerResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	reportID, err := uuid.Parse(req.GetReportId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid report_id: %v", err)
	}

	w, err := s.repo.AddWorker(ctx, tenantID, reportID, req.GetName(), req.GetRole(), req.GetHours())
	if err != nil {
		return nil, mapRapporteError(err)
	}
	slog.InfoContext(ctx, "worker added", "worker_id", w.ID, "report_id", reportID)
	return &rapportev1.AddWorkerResponse{Worker: rapporteWorkerToProto(w)}, nil
}

func (s *RapporteGRPCServer) RemoveWorker(ctx context.Context, req *rapportev1.RemoveWorkerRequest) (*rapportev1.RemoveWorkerResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	workerID, err := uuid.Parse(req.GetWorkerId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid worker_id: %v", err)
	}

	if err := s.repo.RemoveWorker(ctx, tenantID, workerID); err != nil {
		return nil, mapRapporteError(err)
	}
	return &rapportev1.RemoveWorkerResponse{Success: true}, nil
}

func (s *RapporteGRPCServer) ListWorkers(ctx context.Context, req *rapportev1.ListWorkersRequest) (*rapportev1.ListWorkersResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	reportID, err := uuid.Parse(req.GetReportId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid report_id: %v", err)
	}

	workers, err := s.repo.ListWorkers(ctx, tenantID, reportID)
	if err != nil {
		return nil, mapRapporteError(err)
	}

	protoWorkers := make([]*rapportev1.Worker, len(workers))
	for i, w := range workers {
		wCopy := w
		protoWorkers[i] = rapporteWorkerToProto(&wCopy)
	}
	return &rapportev1.ListWorkersResponse{Workers: protoWorkers}, nil
}

// ============================================================================
// Measurement RPCs
// ============================================================================

func (s *RapporteGRPCServer) CreateMeasurement(ctx context.Context, req *rapportev1.CreateMeasurementRequest) (*rapportev1.CreateMeasurementResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}

	var reportID *uuid.UUID
	if rid := req.GetReportId(); rid != "" {
		parsed, parseErr := uuid.Parse(rid)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid report_id: %v", parseErr)
		}
		reportID = &parsed
	}

	m, err := s.repo.CreateMeasurement(ctx, tenantID, reportID,
		req.GetTitle(), req.GetLocation(), req.GetMeasuredBy(), req.GetMeasuredAt(), req.GetNotes())
	if err != nil {
		return nil, mapRapporteError(err)
	}
	slog.InfoContext(ctx, "measurement created", "measurement_id", m.ID, "tenant_id", tenantID)
	return &rapportev1.CreateMeasurementResponse{Measurement: rapporteMeasurementToProto(m)}, nil
}

func (s *RapporteGRPCServer) GetMeasurement(ctx context.Context, req *rapportev1.GetMeasurementRequest) (*rapportev1.GetMeasurementResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	measurementID, err := uuid.Parse(req.GetMeasurementId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid measurement_id: %v", err)
	}

	m, err := s.repo.GetMeasurement(ctx, tenantID, measurementID)
	if err != nil {
		return nil, mapRapporteError(err)
	}
	return &rapportev1.GetMeasurementResponse{Measurement: rapporteMeasurementToProto(m)}, nil
}

func (s *RapporteGRPCServer) ListMeasurements(ctx context.Context, req *rapportev1.ListMeasurementsRequest) (*rapportev1.ListMeasurementsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}

	var reportID *uuid.UUID
	if rid := req.GetReportId(); rid != "" {
		parsed, parseErr := uuid.Parse(rid)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid report_id: %v", parseErr)
		}
		reportID = &parsed
	}

	measurements, total, err := s.repo.ListMeasurements(ctx, tenantID, reportID, int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		return nil, mapRapporteError(err)
	}

	protoMeasurements := make([]*rapportev1.Measurement, len(measurements))
	for i, m := range measurements {
		mCopy := m
		protoMeasurements[i] = rapporteMeasurementToProto(&mCopy)
	}
	return &rapportev1.ListMeasurementsResponse{
		Measurements: protoMeasurements,
		Total:        int32(total),
	}, nil
}

func (s *RapporteGRPCServer) UpdateMeasurement(ctx context.Context, req *rapportev1.UpdateMeasurementRequest) (*rapportev1.UpdateMeasurementResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	measurementID, err := uuid.Parse(req.GetMeasurementId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid measurement_id: %v", err)
	}

	m, err := s.repo.UpdateMeasurement(ctx, tenantID, measurementID,
		req.GetTitle(), req.GetLocation(), req.GetMeasuredBy(), req.GetMeasuredAt(), req.GetNotes())
	if err != nil {
		return nil, mapRapporteError(err)
	}
	return &rapportev1.UpdateMeasurementResponse{Measurement: rapporteMeasurementToProto(m)}, nil
}

func (s *RapporteGRPCServer) DeleteMeasurement(ctx context.Context, req *rapportev1.DeleteMeasurementRequest) (*rapportev1.DeleteMeasurementResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	measurementID, err := uuid.Parse(req.GetMeasurementId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid measurement_id: %v", err)
	}

	if err := s.repo.DeleteMeasurement(ctx, tenantID, measurementID); err != nil {
		return nil, mapRapporteError(err)
	}
	return &rapportev1.DeleteMeasurementResponse{Success: true}, nil
}

func (s *RapporteGRPCServer) AddMeasurementPosition(ctx context.Context, req *rapportev1.AddMeasurementPositionRequest) (*rapportev1.AddMeasurementPositionResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	measurementID, err := uuid.Parse(req.GetMeasurementId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid measurement_id: %v", err)
	}

	p, err := s.repo.AddMeasurementPosition(ctx, tenantID, measurementID,
		int(req.GetPositionNumber()), req.GetDescription(), req.GetUnit(),
		req.GetQuantity(), req.GetUnitPrice())
	if err != nil {
		return nil, mapRapporteError(err)
	}
	return &rapportev1.AddMeasurementPositionResponse{Position: rapportePositionToProto(p)}, nil
}

func (s *RapporteGRPCServer) DeleteMeasurementPosition(ctx context.Context, req *rapportev1.DeleteMeasurementPositionRequest) (*rapportev1.DeleteMeasurementPositionResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	positionID, err := uuid.Parse(req.GetPositionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid position_id: %v", err)
	}

	if err := s.repo.DeleteMeasurementPosition(ctx, tenantID, positionID); err != nil {
		return nil, mapRapporteError(err)
	}
	return &rapportev1.DeleteMeasurementPositionResponse{Success: true}, nil
}

// ============================================================================
// Template RPCs
// ============================================================================

func (s *RapporteGRPCServer) CreateTemplate(ctx context.Context, req *rapportev1.CreateTemplateRequest) (*rapportev1.CreateTemplateResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}

	t, err := s.repo.CreateTemplate(ctx, tenantID,
		req.GetName(), req.GetDescription(), req.GetCategory(), req.GetDefaultLinesJson())
	if err != nil {
		return nil, mapRapporteError(err)
	}
	slog.InfoContext(ctx, "template created", "template_id", t.ID, "tenant_id", tenantID)
	return &rapportev1.CreateTemplateResponse{Template: rapporteTemplateToProto(t)}, nil
}

func (s *RapporteGRPCServer) GetTemplate(ctx context.Context, req *rapportev1.GetTemplateRequest) (*rapportev1.GetTemplateResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	templateID, err := uuid.Parse(req.GetTemplateId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid template_id: %v", err)
	}

	t, err := s.repo.GetTemplate(ctx, tenantID, templateID)
	if err != nil {
		return nil, mapRapporteError(err)
	}
	return &rapportev1.GetTemplateResponse{Template: rapporteTemplateToProto(t)}, nil
}

func (s *RapporteGRPCServer) ListTemplates(ctx context.Context, req *rapportev1.ListTemplatesRequest) (*rapportev1.ListTemplatesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}

	templates, total, err := s.repo.ListTemplates(ctx, tenantID, req.GetActiveOnly(), int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		return nil, mapRapporteError(err)
	}

	protoTemplates := make([]*rapportev1.ReportTemplate, len(templates))
	for i, t := range templates {
		tCopy := t
		protoTemplates[i] = rapporteTemplateToProto(&tCopy)
	}
	return &rapportev1.ListTemplatesResponse{
		Templates: protoTemplates,
		Total:     int32(total),
	}, nil
}

func (s *RapporteGRPCServer) UpdateTemplate(ctx context.Context, req *rapportev1.UpdateTemplateRequest) (*rapportev1.UpdateTemplateResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	templateID, err := uuid.Parse(req.GetTemplateId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid template_id: %v", err)
	}

	t, err := s.repo.UpdateTemplate(ctx, tenantID, templateID,
		req.GetName(), req.GetDescription(), req.GetCategory(), req.GetDefaultLinesJson(), req.GetIsActive())
	if err != nil {
		return nil, mapRapporteError(err)
	}
	return &rapportev1.UpdateTemplateResponse{Template: rapporteTemplateToProto(t)}, nil
}

func (s *RapporteGRPCServer) DeleteTemplate(ctx context.Context, req *rapportev1.DeleteTemplateRequest) (*rapportev1.DeleteTemplateResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing tenant: %v", err)
	}
	templateID, err := uuid.Parse(req.GetTemplateId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid template_id: %v", err)
	}

	if err := s.repo.DeleteTemplate(ctx, tenantID, templateID); err != nil {
		return nil, mapRapporteError(err)
	}
	return &rapportev1.DeleteTemplateResponse{Success: true}, nil
}

// ============================================================================
// Conversion helpers
// ============================================================================

func rapporteReportToProto(r *rapporte.WorkReport) *rapportev1.WorkReport {
	if r == nil {
		return nil
	}
	proto := &rapportev1.WorkReport{
		Id:          r.ID.String(),
		TenantId:    r.TenantID.String(),
		Title:       r.Title,
		Description: r.Description,
		Status:      string(r.Status),
		AuthorId:    r.AuthorID.String(),
		ReviewNote:  r.ReviewNote,
		ReportDate:  r.ReportDate.Format("2006-01-02"),
		CreatedAt:   timestamppb.New(r.CreatedAt),
		UpdatedAt:   timestamppb.New(r.UpdatedAt),
	}
	if r.ReviewerID != nil {
		s := r.ReviewerID.String()
		proto.ReviewerId = &s
	}
	if r.ReviewedAt != nil {
		proto.ReviewedAt = timestamppb.New(*r.ReviewedAt)
	}
	if r.Lat != nil {
		proto.Lat = r.Lat
	}
	if r.Lon != nil {
		proto.Lon = r.Lon
	}
	// Signature fields (fix: was missing in original)
	if r.SignatureData != nil {
		proto.SignatureData = *r.SignatureData
	}
	if r.SignedAt != nil {
		proto.SignedAt = timestamppb.New(*r.SignedAt)
	}
	if r.SignedBy != nil {
		proto.SignedBy = *r.SignedBy
	}
	// Extended fields
	if r.Weather != nil {
		proto.Weather = *r.Weather
	}
	if r.Temperature != nil {
		proto.Temperature = *r.Temperature
	}
	if r.WorkStart != nil {
		proto.WorkStart = *r.WorkStart
	}
	if r.WorkEnd != nil {
		proto.WorkEnd = *r.WorkEnd
	}
	if r.BreakMinutes != nil {
		proto.BreakMinutes = int32(*r.BreakMinutes)
	}
	if r.ProjectName != nil {
		proto.ProjectName = *r.ProjectName
	}
	// Workers
	if len(r.Workers) > 0 {
		protoWorkers := make([]*rapportev1.Worker, len(r.Workers))
		for i, w := range r.Workers {
			wCopy := w
			protoWorkers[i] = rapporteWorkerToProto(&wCopy)
		}
		proto.Workers = protoWorkers
	}
	return proto
}

func rapporteLineToProto(l *rapporte.ReportLine) *rapportev1.ReportLine {
	if l == nil {
		return nil
	}
	proto := &rapportev1.ReportLine{
		Id:          l.ID.String(),
		TenantId:    l.TenantID.String(),
		ReportId:    l.ReportID.String(),
		Position:    int32(l.Position),
		Description: l.Description,
		Quantity:    l.Quantity,
		Unit:        l.Unit,
		Notes:       l.Notes,
		CreatedAt:   timestamppb.New(l.CreatedAt),
		UpdatedAt:   timestamppb.New(l.UpdatedAt),
	}
	if l.Category != nil {
		proto.Category = *l.Category
	}
	if l.Article != nil {
		proto.Article = *l.Article
	}
	return proto
}

func rapporteAttachmentToProto(a *rapporte.ReportAttachment) *rapportev1.ReportAttachment {
	if a == nil {
		return nil
	}
	proto := &rapportev1.ReportAttachment{
		Id:          a.ID.String(),
		TenantId:    a.TenantID.String(),
		ReportId:    a.ReportID.String(),
		Filename:    a.Filename,
		ContentType: a.ContentType,
		SizeBytes:   a.SizeBytes,
		ObjectKey:   a.ObjectKey,
		// PresignedUrl: filled by gateway layer if needed
		UploadedBy: a.UploadedBy.String(),
		CreatedAt:  timestamppb.New(a.CreatedAt),
	}
	if a.LineID != nil {
		s := a.LineID.String()
		proto.LineId = &s
	}
	return proto
}

func rapporteWorkerToProto(w *rapporte.Worker) *rapportev1.Worker {
	if w == nil {
		return nil
	}
	return &rapportev1.Worker{
		Id:        w.ID.String(),
		TenantId:  w.TenantID.String(),
		ReportId:  w.ReportID.String(),
		Name:      w.Name,
		Role:      w.Role.String,
		Hours:     w.Hours.Float64,
		CreatedAt: w.CreatedAt.Format(time.RFC3339),
	}
}

func rapporteMeasurementToProto(m *rapporte.Measurement) *rapportev1.Measurement {
	if m == nil {
		return nil
	}
	proto := &rapportev1.Measurement{
		Id:        m.ID.String(),
		TenantId:  m.TenantID.String(),
		Title:     m.Title,
		Location:  m.Location.String,
		MeasuredBy: m.MeasuredBy.String,
		MeasuredAt: m.MeasuredAt.String,
		Notes:     m.Notes.String,
		CreatedAt: m.CreatedAt.Format(time.RFC3339),
		UpdatedAt: m.UpdatedAt.Format(time.RFC3339),
	}
	if m.ReportID != nil {
		proto.ReportId = m.ReportID.String()
	}
	if len(m.Positions) > 0 {
		protoPositions := make([]*rapportev1.MeasurementPosition, len(m.Positions))
		for i, p := range m.Positions {
			pCopy := p
			protoPositions[i] = rapportePositionToProto(&pCopy)
		}
		proto.Positions = protoPositions
	}
	return proto
}

func rapportePositionToProto(p *rapporte.MeasurementPosition) *rapportev1.MeasurementPosition {
	if p == nil {
		return nil
	}
	return &rapportev1.MeasurementPosition{
		Id:             p.ID.String(),
		TenantId:       p.TenantID.String(),
		MeasurementId:  p.MeasurementID.String(),
		PositionNumber: int32(p.PositionNumber),
		Description:    p.Description,
		Unit:           p.Unit.String,
		Quantity:       p.Quantity.Float64,
		UnitPrice:      p.UnitPrice.Float64,
		CreatedAt:      p.CreatedAt.Format(time.RFC3339),
	}
}

func rapporteTemplateToProto(t *rapporte.ReportTemplate) *rapportev1.ReportTemplate {
	if t == nil {
		return nil
	}
	return &rapportev1.ReportTemplate{
		Id:              t.ID.String(),
		TenantId:        t.TenantID.String(),
		Name:            t.Name,
		Description:     t.Description.String,
		Category:        t.Category.String,
		DefaultLinesJson: t.DefaultLinesJSON.String,
		IsActive:        t.IsActive,
		CreatedAt:       t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       t.UpdatedAt.Format(time.RFC3339),
	}
}

// ============================================================================
// Error mapping
// ============================================================================

func mapRapporteError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, rapporte.ErrReportNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, rapporte.ErrLineNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, rapporte.ErrAttachmentNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, rapporte.ErrWorkerNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, rapporte.ErrMeasurementNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, rapporte.ErrPositionNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, rapporte.ErrTemplateNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, rapporte.ErrAlreadyApproved):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, rapporte.ErrInvalidStateTransition):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, rapporte.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		slog.Error("unhandled rapporte service error", "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}
