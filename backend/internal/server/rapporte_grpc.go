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

	"github.com/kmuhub/kmuhub/internal/rapporte"
	rapportev1 "github.com/kmuhub/kmuhub/proto/rapporte/v1"
)

// RapporteGRPCServer implements the RapporteService gRPC server.
type RapporteGRPCServer struct {
	rapportev1.UnimplementedRapporteServiceServer
	svc *rapporte.Service
}

// NewRapporteGRPCServer creates a new Rapporte gRPC server.
func NewRapporteGRPCServer(svc *rapporte.Service) *RapporteGRPCServer {
	return &RapporteGRPCServer{svc: svc}
}

// ============================================================================
// Report RPCs
// ============================================================================

func (s *RapporteGRPCServer) CreateReport(ctx context.Context, req *rapportev1.CreateReportRequest) (*rapportev1.ReportResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
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
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
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
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
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
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
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
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
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
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
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
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
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
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
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
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
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
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
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
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
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
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
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
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
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
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
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
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
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
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
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
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
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
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	reports, total, err := s.svc.ListPendingApprovals(ctx, tenantID, int(req.GetPage()), int(req.GetPageSize()))
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
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	reportID, err := uuid.Parse(req.GetReportId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid report_id: %v", err)
	}

	payload, err := s.svc.ExportPDF(ctx, tenantID, reportID)
	if err != nil {
		return nil, mapRapporteError(err)
	}

	filename := fmt.Sprintf("arbeitsbericht_%s.txt", reportID.String()[:8])
	return &rapportev1.ExportPDFResponse{
		Payload:     payload,
		ContentType: "text/plain", // TODO Sprint 3: application/pdf
		Filename:    filename,
	}, nil
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
	return proto
}

func rapporteLineToProto(l *rapporte.ReportLine) *rapportev1.ReportLine {
	if l == nil {
		return nil
	}
	return &rapportev1.ReportLine{
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
