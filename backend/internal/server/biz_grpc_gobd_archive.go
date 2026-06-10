package server

// GoBD Belegarchiv gRPC handlers for the FinanceService (§147 AO).
// All 6 RPCs delegate to gobdarchive.Service.

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/biz/gobdarchive"
	"github.com/kmuhub/kmuhub/internal/models"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// ============================================================================
// ArchiveDocument
// ============================================================================

func (s *BizGRPCServer) ArchiveDocument(ctx context.Context, req *bizv1.ArchiveDocumentRequest) (*bizv1.ArchiveDocumentResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	archivedBy, err := uuid.Parse(req.GetArchivedBy())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid archived_by")
	}

	content := req.GetContent()
	if len(content) == 0 {
		return nil, status.Error(codes.InvalidArgument, "content must not be empty")
	}

	var sourceInvoiceID *uuid.UUID
	if srcStr := req.GetSourceInvoiceId(); srcStr != "" {
		parsed, parseErr := uuid.Parse(srcStr)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid source_invoice_id")
		}
		sourceInvoiceID = &parsed
	}

	doc, err := s.gobdArchiveSvc.ArchiveDocument(ctx, gobdarchive.ArchiveInput{
		TenantID:         tenantID,
		DocType:          req.GetDocType(),
		SourceInvoiceID:  sourceInvoiceID,
		OriginalFilename: req.GetOriginalFilename(),
		MimeType:         req.GetMimeType(),
		Content:          bytes.NewReader(content),
		Size:             int64(len(content)),
		ArchivedBy:       archivedBy,
	})
	if err != nil {
		return nil, mapGobdArchiveError(err)
	}

	return &bizv1.ArchiveDocumentResponse{
		Document: toProtoGobdDocument(doc),
	}, nil
}

// ============================================================================
// ArchiveInvoiceDocument
// ============================================================================

func (s *BizGRPCServer) ArchiveInvoiceDocument(ctx context.Context, req *bizv1.ArchiveInvoiceDocumentRequest) (*bizv1.ArchiveInvoiceDocumentResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	invoiceID, err := uuid.Parse(req.GetInvoiceId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid invoice_id")
	}
	archivedBy, err := uuid.Parse(req.GetArchivedBy())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid archived_by")
	}

	// Render the invoice PDF using the existing pdfGenerator on BizGRPCServer.
	inv, err := s.invoiceService.GetByID(ctx, tenantID, invoiceID)
	if err != nil {
		return nil, mapBizError(err)
	}

	pdfBytes, pdfErr := s.pdfGenerator.GenerateInvoicePDF(*inv)
	if pdfErr != nil {
		slog.Error("failed to generate invoice pdf for archive",
			"invoice_id", invoiceID,
			"error", pdfErr,
		)
		return nil, status.Error(codes.Internal, "failed to generate invoice PDF")
	}

	doc, archiveErr := s.gobdArchiveSvc.ArchiveInvoiceDocument(ctx, tenantID, invoiceID, archivedBy, pdfBytes)
	if archiveErr != nil {
		return nil, mapGobdArchiveError(archiveErr)
	}

	return &bizv1.ArchiveInvoiceDocumentResponse{
		Document: toProtoGobdDocument(doc),
	}, nil
}

// ============================================================================
// GetGobdDocument
// ============================================================================

func (s *BizGRPCServer) GetGobdDocument(ctx context.Context, req *bizv1.GetGobdDocumentRequest) (*bizv1.GetGobdDocumentResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	docID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	doc, err := s.gobdArchiveSvc.GetByID(ctx, tenantID, docID)
	if err != nil {
		return nil, mapGobdArchiveError(err)
	}

	events, err := s.gobdArchiveSvc.ListEvents(ctx, tenantID, docID)
	if err != nil {
		return nil, mapGobdArchiveError(err)
	}

	return &bizv1.GetGobdDocumentResponse{
		Document: toProtoGobdDocument(doc),
		Events:   toProtoGobdEventList(events),
	}, nil
}

// ============================================================================
// ListGobdDocuments
// ============================================================================

func (s *BizGRPCServer) ListGobdDocuments(ctx context.Context, req *bizv1.ListGobdDocumentsRequest) (*bizv1.ListGobdDocumentsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	filter := gobdarchive.ListFilter{
		TenantID: tenantID,
		DocType:  req.GetDocType(),
		Page:     int(req.GetPage()),
		PerPage:  int(req.GetPerPage()),
	}

	if srcStr := req.GetSourceInvoiceId(); srcStr != "" {
		parsed, parseErr := uuid.Parse(srcStr)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid source_invoice_id")
		}
		filter.SourceInvoiceID = &parsed
	}

	if dateFromStr := req.GetDateFrom(); dateFromStr != "" {
		t, parseErr := time.Parse("2006-01-02", dateFromStr)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid date_from; expected YYYY-MM-DD")
		}
		filter.DateFrom = &t
	}

	if dateToStr := req.GetDateTo(); dateToStr != "" {
		t, parseErr := time.Parse("2006-01-02", dateToStr)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid date_to; expected YYYY-MM-DD")
		}
		filter.DateTo = &t
	}

	docs, total, err := s.gobdArchiveSvc.List(ctx, filter)
	if err != nil {
		return nil, mapGobdArchiveError(err)
	}

	protoDocuments := make([]*bizv1.GobdDocumentProto, 0, len(docs))
	for _, doc := range docs {
		protoDocuments = append(protoDocuments, toProtoGobdDocument(doc))
	}

	perPage := req.GetPerPage()
	if perPage <= 0 {
		perPage = 50
	}
	page := req.GetPage()
	if page <= 0 {
		page = 1
	}

	return &bizv1.ListGobdDocumentsResponse{
		Documents: protoDocuments,
		Total:     int32(total),
		Page:      page,
		PerPage:   perPage,
	}, nil
}

// ============================================================================
// DownloadGobdDocument
// ============================================================================

func (s *BizGRPCServer) DownloadGobdDocument(ctx context.Context, req *bizv1.DownloadGobdDocumentRequest) (*bizv1.DownloadGobdDocumentResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	docID, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}
	requestedBy, err := uuid.Parse(req.GetRequestedBy())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid requested_by")
	}

	// Load document first to get filename/mime_type for the response.
	doc, err := s.gobdArchiveSvc.GetByID(ctx, tenantID, docID)
	if err != nil {
		return nil, mapGobdArchiveError(err)
	}

	presignedURL, err := s.gobdArchiveSvc.GetDownloadURL(ctx, tenantID, docID, requestedBy)
	if err != nil {
		return nil, mapGobdArchiveError(err)
	}

	return &bizv1.DownloadGobdDocumentResponse{
		PresignedUrl:     presignedURL,
		OriginalFilename: doc.OriginalFilename,
		MimeType:         doc.MimeType,
	}, nil
}

// ============================================================================
// AddDocumentAnnotation
// ============================================================================

func (s *BizGRPCServer) AddDocumentAnnotation(ctx context.Context, req *bizv1.AddDocumentAnnotationRequest) (*bizv1.AddDocumentAnnotationResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	docID, err := uuid.Parse(req.GetDocumentId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid document_id")
	}
	addedBy, err := uuid.Parse(req.GetAddedBy())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid added_by")
	}

	if err := s.gobdArchiveSvc.AddAnnotation(ctx, tenantID, docID, addedBy, req.GetNote()); err != nil {
		return nil, mapGobdArchiveError(err)
	}

	return &bizv1.AddDocumentAnnotationResponse{Ok: true}, nil
}

// ============================================================================
// Mapping helpers
// ============================================================================

// toProtoGobdDocument converts a models.GobdDocument to its proto representation.
func toProtoGobdDocument(doc *models.GobdDocument) *bizv1.GobdDocumentProto {
	if doc == nil {
		return nil
	}
	sourceInvID := ""
	if doc.SourceInvoiceID != nil {
		sourceInvID = doc.SourceInvoiceID.String()
	}
	return &bizv1.GobdDocumentProto{
		Id:               doc.ID.String(),
		TenantId:         doc.TenantID.String(),
		DocType:          doc.DocType,
		SourceInvoiceId:  sourceInvID,
		StorageKey:       doc.StorageKey,
		Sha256:           doc.SHA256,
		OriginalFilename: doc.OriginalFilename,
		MimeType:         doc.MimeType,
		FileSizeBytes:    doc.FileSizeBytes,
		ArchivedAt:       timestamppb.New(doc.ArchivedAt),
		ArchivedBy:       doc.ArchivedBy.String(),
		RetentionUntil:   doc.RetentionUntil.Format("2006-01-02"),
	}
}

// toProtoGobdEventList converts a slice of GobdDocumentEvent to proto.
func toProtoGobdEventList(events []*models.GobdDocumentEvent) []*bizv1.GobdDocumentEventProto {
	result := make([]*bizv1.GobdDocumentEventProto, 0, len(events))
	for _, ev := range events {
		result = append(result, &bizv1.GobdDocumentEventProto{
			Id:         ev.ID.String(),
			TenantId:   ev.TenantID.String(),
			DocumentId: ev.DocumentID.String(),
			EventType:  ev.EventType,
			CreatedBy:  ev.CreatedBy.String(),
			CreatedAt:  timestamppb.New(ev.CreatedAt),
			Note:       ev.Note,
			MetadataJson: func() string {
				if len(ev.Metadata) > 0 {
					return string(ev.Metadata)
				}
				return "{}"
			}(),
		})
	}
	return result
}

// mapGobdArchiveError maps gobdarchive errors to gRPC status errors.
func mapGobdArchiveError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, gobdarchive.ErrDocumentNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, gobdarchive.ErrDocumentImmutable):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, gobdarchive.ErrInvalidDocumentType):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, gobdarchive.ErrFileTooLarge):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, gobdarchive.ErrEmptyFile):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, gobdarchive.ErrInvoiceNotLocked):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		slog.Error("unhandled gobd archive error", "error", err)
		return status.Error(codes.Internal, "internal server error")
	}
}
