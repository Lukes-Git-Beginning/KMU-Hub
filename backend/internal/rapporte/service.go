package rapporte

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service handles rapporte business logic.
type Service struct {
	repo Repository
}

// NewService creates a new rapporte service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ============================================================================
// Input types — Reports
// ============================================================================

// CreateReportInput contains data to create a work report.
type CreateReportInput struct {
	TenantID    uuid.UUID
	Title       string
	Description string
	AuthorID    uuid.UUID
	Lat         *float64
	Lon         *float64
	ReportDate  *time.Time
}

// UpdateReportInput contains fields that can be updated on a report.
type UpdateReportInput struct {
	TenantID    uuid.UUID
	ReportID    uuid.UUID
	Title       *string
	Description *string
	Lat         *float64
	Lon         *float64
	ReportDate  *time.Time
}

// ListReportsInput contains pagination and filtering for report listing.
type ListReportsInput struct {
	TenantID uuid.UUID
	Status   *ReportStatus
	AuthorID *uuid.UUID
	Search   string
	Page     int
	PageSize int
}

// SubmitReportInput moves a report from Draft → Submitted.
type SubmitReportInput struct {
	TenantID uuid.UUID
	ReportID uuid.UUID
}

// ApproveReportInput moves a report from Submitted → Approved.
type ApproveReportInput struct {
	TenantID   uuid.UUID
	ReportID   uuid.UUID
	ReviewerID uuid.UUID
	ReviewNote string
}

// RejectReportInput moves a report from Submitted → Rejected.
type RejectReportInput struct {
	TenantID   uuid.UUID
	ReportID   uuid.UUID
	ReviewerID uuid.UUID
	ReviewNote string
}

// ============================================================================
// Input types — Lines
// ============================================================================

// AddLineInput contains data for a new report line.
type AddLineInput struct {
	TenantID    uuid.UUID
	ReportID    uuid.UUID
	Description string
	Quantity    float64
	Unit        string
	Notes       string
	Position    int
}

// UpdateLineInput contains updatable fields on a line.
type UpdateLineInput struct {
	TenantID    uuid.UUID
	LineID      uuid.UUID
	Description *string
	Quantity    *float64
	Unit        *string
	Notes       *string
	Position    *int
}

// ============================================================================
// Input types — Attachments
// ============================================================================

// UploadAttachmentInput registers an attachment already placed in MinIO.
type UploadAttachmentInput struct {
	TenantID    uuid.UUID
	ReportID    uuid.UUID
	LineID      *uuid.UUID
	Filename    string
	ContentType string
	SizeBytes   int64
	ObjectKey   string
	UploadedBy  uuid.UUID
}

// ============================================================================
// Report methods
// ============================================================================

// validateGPS returns ErrInvalidInput if lat/lon are out of range.
func validateGPS(lat, lon *float64) error {
	if lat != nil && (*lat < -90 || *lat > 90) {
		return ErrInvalidInput
	}
	if lon != nil && (*lon < -180 || *lon > 180) {
		return ErrInvalidInput
	}
	return nil
}

// CreateReport creates a new work report in Draft status.
func (s *Service) CreateReport(ctx context.Context, input CreateReportInput) (*WorkReport, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, ErrInvalidInput
	}
	if err := validateGPS(input.Lat, input.Lon); err != nil {
		return nil, err
	}

	now := time.Now()
	reportDate := now
	if input.ReportDate != nil {
		reportDate = *input.ReportDate
	}

	report := &WorkReport{
		ID:          uuid.New(),
		TenantID:    input.TenantID,
		Title:       title,
		Description: input.Description,
		Status:      StatusDraft,
		AuthorID:    input.AuthorID,
		Lat:         input.Lat,
		Lon:         input.Lon,
		ReportDate:  reportDate,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateReport(ctx, report); err != nil {
		return nil, fmt.Errorf("create work report: %w", err)
	}

	slog.Info("work report created",
		"report_id", report.ID,
		"tenant_id", report.TenantID,
		"author_id", report.AuthorID,
	)

	return report, nil
}

// UpdateReport updates mutable fields on a draft or rejected report.
// Guard 1 (ErrAlreadyApproved): approved reports cannot be edited.
func (s *Service) UpdateReport(ctx context.Context, input UpdateReportInput) (*WorkReport, error) {
	report, err := s.repo.GetReport(ctx, input.TenantID, input.ReportID)
	if err != nil {
		return nil, err
	}

	// Guard 1: block edits on approved reports
	if report.Status == StatusApproved {
		return nil, ErrAlreadyApproved
	}

	if input.Title != nil {
		t := strings.TrimSpace(*input.Title)
		if t == "" {
			return nil, ErrInvalidInput
		}
		report.Title = t
	}
	if input.Description != nil {
		report.Description = *input.Description
	}
	if err := validateGPS(input.Lat, input.Lon); err != nil {
		return nil, err
	}
	if input.Lat != nil {
		report.Lat = input.Lat
	}
	if input.Lon != nil {
		report.Lon = input.Lon
	}
	if input.ReportDate != nil {
		report.ReportDate = *input.ReportDate
	}

	report.UpdatedAt = time.Now()

	if err := s.repo.UpdateReport(ctx, report); err != nil {
		return nil, fmt.Errorf("update work report: %w", err)
	}

	slog.Info("work report updated", "report_id", report.ID, "tenant_id", report.TenantID)
	return report, nil
}

// DeleteReport soft-deletes a report. Approved reports cannot be deleted.
func (s *Service) DeleteReport(ctx context.Context, tenantID, reportID uuid.UUID) error {
	report, err := s.repo.GetReport(ctx, tenantID, reportID)
	if err != nil {
		return err
	}

	// Guard 1: block deletion of approved reports
	if report.Status == StatusApproved {
		return ErrAlreadyApproved
	}

	if err := s.repo.SoftDeleteReport(ctx, tenantID, reportID); err != nil {
		return fmt.Errorf("delete work report: %w", err)
	}

	slog.Info("work report deleted", "report_id", reportID, "tenant_id", tenantID)
	return nil
}

// GetReport retrieves a report by ID.
func (s *Service) GetReport(ctx context.Context, tenantID, reportID uuid.UUID) (*WorkReport, error) {
	return s.repo.GetReport(ctx, tenantID, reportID)
}

// ListReports retrieves reports with optional filtering and pagination.
func (s *Service) ListReports(ctx context.Context, input ListReportsInput) ([]*WorkReport, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 200 {
		input.PageSize = 50
	}

	offset := (input.Page - 1) * input.PageSize
	filter := ListReportsFilter{
		Status:   input.Status,
		AuthorID: input.AuthorID,
		Search:   input.Search,
	}

	return s.repo.ListReports(ctx, input.TenantID, filter, offset, input.PageSize)
}

// ============================================================================
// State machine methods
// ============================================================================

// SubmitReport transitions a report from Draft → Submitted.
// Guard 2 (ErrInvalidStateTransition): only Draft can move to Submitted.
func (s *Service) SubmitReport(ctx context.Context, input SubmitReportInput) (*WorkReport, error) {
	report, err := s.repo.GetReport(ctx, input.TenantID, input.ReportID)
	if err != nil {
		return nil, err
	}

	// Guard 2: pre-check before DB write
	if report.Status != StatusDraft {
		return nil, ErrInvalidStateTransition
	}

	report.Status = StatusSubmitted
	report.UpdatedAt = time.Now()

	if err := s.repo.UpdateReport(ctx, report); err != nil {
		return nil, fmt.Errorf("submit work report: %w", err)
	}

	slog.Info("work report submitted", "report_id", report.ID, "tenant_id", report.TenantID)
	return report, nil
}

// ApproveReport transitions a report from Submitted → Approved atomically.
// Uses atomic UPDATE WHERE status='submitted' to close the TOCTOU race window.
func (s *Service) ApproveReport(ctx context.Context, input ApproveReportInput) (*WorkReport, error) {
	affected, err := s.repo.AtomicApproveReport(ctx, input.TenantID, input.ReportID, input.ReviewerID, input.ReviewNote)
	if err != nil {
		return nil, fmt.Errorf("approve work report: %w", err)
	}
	if !affected {
		// Row was not in 'submitted' state — distinguish approved vs. other
		existing, getErr := s.repo.GetReport(ctx, input.TenantID, input.ReportID)
		if getErr != nil {
			return nil, getErr
		}
		if existing.Status == StatusApproved {
			return nil, ErrAlreadyApproved
		}
		return nil, ErrInvalidStateTransition
	}

	report, err := s.repo.GetReport(ctx, input.TenantID, input.ReportID)
	if err != nil {
		return nil, err
	}

	slog.Info("work report approved",
		"report_id", report.ID,
		"reviewer_id", input.ReviewerID,
	)
	return report, nil
}

// RejectReport transitions a report from Submitted → Rejected atomically.
// Uses atomic UPDATE WHERE status='submitted' to close the TOCTOU race window.
func (s *Service) RejectReport(ctx context.Context, input RejectReportInput) (*WorkReport, error) {
	affected, err := s.repo.AtomicRejectReport(ctx, input.TenantID, input.ReportID, input.ReviewerID, input.ReviewNote)
	if err != nil {
		return nil, fmt.Errorf("reject work report: %w", err)
	}
	if !affected {
		existing, getErr := s.repo.GetReport(ctx, input.TenantID, input.ReportID)
		if getErr != nil {
			return nil, getErr
		}
		if existing.Status == StatusApproved {
			return nil, ErrAlreadyApproved
		}
		return nil, ErrInvalidStateTransition
	}

	report, err := s.repo.GetReport(ctx, input.TenantID, input.ReportID)
	if err != nil {
		return nil, err
	}

	slog.Info("work report rejected",
		"report_id", report.ID,
		"reviewer_id", input.ReviewerID,
	)
	return report, nil
}

// ============================================================================
// Line methods
// ============================================================================

// AddLine adds a line item to a report. Approved reports are immutable.
func (s *Service) AddLine(ctx context.Context, input AddLineInput) (*ReportLine, error) {
	desc := strings.TrimSpace(input.Description)
	if desc == "" {
		return nil, ErrInvalidInput
	}

	// Guard 1: block adds on approved reports
	report, err := s.repo.GetReport(ctx, input.TenantID, input.ReportID)
	if err != nil {
		return nil, err
	}
	if report.Status == StatusApproved {
		return nil, ErrAlreadyApproved
	}

	now := time.Now()
	line := &ReportLine{
		ID:          uuid.New(),
		TenantID:    input.TenantID,
		ReportID:    input.ReportID,
		Position:    input.Position,
		Description: desc,
		Quantity:    input.Quantity,
		Unit:        input.Unit,
		Notes:       input.Notes,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateLine(ctx, line); err != nil {
		return nil, fmt.Errorf("add report line: %w", err)
	}

	slog.Info("report line added", "line_id", line.ID, "report_id", line.ReportID)
	return line, nil
}

// UpdateLine updates a line item. Approved reports are immutable.
func (s *Service) UpdateLine(ctx context.Context, input UpdateLineInput) (*ReportLine, error) {
	line, err := s.repo.GetLine(ctx, input.TenantID, input.LineID)
	if err != nil {
		return nil, err
	}

	// Guard 1: block updates on approved reports
	report, err := s.repo.GetReport(ctx, input.TenantID, line.ReportID)
	if err != nil {
		return nil, err
	}
	if report.Status == StatusApproved {
		return nil, ErrAlreadyApproved
	}

	if input.Description != nil {
		d := strings.TrimSpace(*input.Description)
		if d == "" {
			return nil, ErrInvalidInput
		}
		line.Description = d
	}
	if input.Quantity != nil {
		line.Quantity = *input.Quantity
	}
	if input.Unit != nil {
		line.Unit = *input.Unit
	}
	if input.Notes != nil {
		line.Notes = *input.Notes
	}
	if input.Position != nil {
		line.Position = *input.Position
	}

	line.UpdatedAt = time.Now()

	if err := s.repo.UpdateLine(ctx, line); err != nil {
		return nil, fmt.Errorf("update report line: %w", err)
	}

	return line, nil
}

// DeleteLine removes a line from a report. Approved reports are immutable.
func (s *Service) DeleteLine(ctx context.Context, tenantID, lineID uuid.UUID) error {
	line, err := s.repo.GetLine(ctx, tenantID, lineID)
	if err != nil {
		return err
	}

	// Guard 1: block deletes on approved reports
	report, err := s.repo.GetReport(ctx, tenantID, line.ReportID)
	if err != nil {
		return err
	}
	if report.Status == StatusApproved {
		return ErrAlreadyApproved
	}

	if err := s.repo.DeleteLine(ctx, tenantID, lineID); err != nil {
		return fmt.Errorf("delete report line: %w", err)
	}

	slog.Info("report line deleted", "line_id", lineID)
	return nil
}

// ListLines returns all lines for a report, ordered by position.
func (s *Service) ListLines(ctx context.Context, tenantID, reportID uuid.UUID) ([]*ReportLine, error) {
	if _, err := s.repo.GetReport(ctx, tenantID, reportID); err != nil {
		return nil, err
	}
	return s.repo.ListLines(ctx, tenantID, reportID)
}

// ============================================================================
// Attachment methods
// ============================================================================

// UploadAttachment registers a file that the client already uploaded to MinIO.
// ObjectKey follows the convention: tenants/<tenant_id>/rapporte/<report_id>/<line_id>/<filename>
func (s *Service) UploadAttachment(ctx context.Context, input UploadAttachmentInput) (*ReportAttachment, error) {
	if input.Filename == "" || input.ObjectKey == "" {
		return nil, ErrInvalidInput
	}

	// Security: validate that the object key is scoped to this tenant's path.
	// Prevents cross-tenant MinIO object access via a crafted object_key.
	expectedPrefix := fmt.Sprintf("tenants/%s/", input.TenantID)
	if !strings.HasPrefix(input.ObjectKey, expectedPrefix) {
		return nil, ErrInvalidInput
	}

	if _, err := s.repo.GetReport(ctx, input.TenantID, input.ReportID); err != nil {
		return nil, err
	}

	att := &ReportAttachment{
		ID:          uuid.New(),
		TenantID:    input.TenantID,
		ReportID:    input.ReportID,
		LineID:      input.LineID,
		Filename:    input.Filename,
		ContentType: input.ContentType,
		SizeBytes:   input.SizeBytes,
		ObjectKey:   input.ObjectKey,
		UploadedBy:  input.UploadedBy,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.CreateAttachment(ctx, att); err != nil {
		return nil, fmt.Errorf("upload attachment: %w", err)
	}

	slog.Info("report attachment uploaded",
		"attachment_id", att.ID,
		"report_id", att.ReportID,
		"filename", att.Filename,
	)
	return att, nil
}

// ListAttachments returns attachments for a report, optionally filtered by line.
func (s *Service) ListAttachments(ctx context.Context, tenantID, reportID uuid.UUID, lineID *uuid.UUID) ([]*ReportAttachment, error) {
	if _, err := s.repo.GetReport(ctx, tenantID, reportID); err != nil {
		return nil, err
	}
	return s.repo.ListAttachments(ctx, tenantID, reportID, lineID)
}

// DeleteAttachment removes an attachment record. Does NOT delete from MinIO (caller's responsibility).
func (s *Service) DeleteAttachment(ctx context.Context, tenantID, attachmentID uuid.UUID) error {
	if err := s.repo.DeleteAttachment(ctx, tenantID, attachmentID); err != nil {
		return fmt.Errorf("delete attachment: %w", err)
	}
	slog.Info("report attachment deleted", "attachment_id", attachmentID)
	return nil
}

// ============================================================================
// Stats & pending approvals
// ============================================================================

// GetReportStats returns status-level counts for a tenant.
// Uses an efficient GROUP BY query instead of fetching all rows.
func (s *Service) GetReportStats(ctx context.Context, tenantID uuid.UUID) (*ReportStats, error) {
	counts, err := s.repo.GetReportStatsCounts(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get report stats: %w", err)
	}

	stats := &ReportStats{}
	for status, count := range counts {
		stats.TotalReports += count
		switch ReportStatus(status) {
		case StatusDraft:
			stats.DraftCount = count
		case StatusSubmitted:
			stats.SubmittedCount = count
		case StatusApproved:
			stats.ApprovedCount = count
		case StatusRejected:
			stats.RejectedCount = count
		}
	}
	return stats, nil
}

// ListPendingApprovals returns paginated reports in Submitted status.
// A non-nil authorID restricts the queue to that author's own reports, which is
// what a caller whose rapporte:report:read grant is scoped to "own" may see.
func (s *Service) ListPendingApprovals(ctx context.Context, tenantID uuid.UUID, authorID *uuid.UUID, page, pageSize int) ([]*WorkReport, int, error) {
	submitted := StatusSubmitted
	return s.ListReports(ctx, ListReportsInput{
		TenantID: tenantID,
		Status:   &submitted,
		AuthorID: authorID,
		Page:     page,
		PageSize: pageSize,
	})
}

// ============================================================================
// Signature methods
// ============================================================================

// SaveSignature stores a customer signature on a report.
// signatureData must be a PNG or SVG data URL, capped at 1 MB.
// signedBy is a free-text identifier (e.g. customer name or user ID string).
func (s *Service) SaveSignature(ctx context.Context, tenantID, reportID uuid.UUID, signatureData, signedBy string) (*WorkReport, error) {
	if tenantID == uuid.Nil || reportID == uuid.Nil {
		return nil, errors.New("tenantID and reportID are required")
	}
	if signatureData == "" {
		return nil, errors.New("signature_data is required")
	}
	if !strings.HasPrefix(signatureData, "data:image/png;base64,") && !strings.HasPrefix(signatureData, "data:image/svg+xml;base64,") {
		return nil, errors.New("signature_data must be a valid data URL (data:image/png;base64,... or data:image/svg+xml;base64,...)")
	}
	if len(signatureData) > 1_048_576 { // 1 MB
		return nil, errors.New("signature_data exceeds maximum size of 1 MB")
	}
	if signedBy == "" {
		return nil, errors.New("signed_by is required")
	}

	report, err := s.repo.SaveSignature(ctx, tenantID, reportID, signatureData, signedBy)
	if err != nil {
		return nil, fmt.Errorf("save signature: %w", err)
	}

	slog.Info("report signature saved",
		"report_id", reportID,
		"tenant_id", tenantID,
		"signed_by", signedBy,
	)
	return report, nil
}

// ExportPDF renders a report and its line items as a PDF via maroto.
func (s *Service) ExportPDF(ctx context.Context, tenantID, reportID uuid.UUID) ([]byte, error) {
	report, err := s.repo.GetReport(ctx, tenantID, reportID)
	if err != nil {
		return nil, err
	}

	lines, err := s.repo.ListLines(ctx, tenantID, reportID)
	if err != nil {
		return nil, fmt.Errorf("list report lines: %w", err)
	}

	pdf, err := renderReportPDF(report, lines)
	if err != nil {
		return nil, fmt.Errorf("render report pdf: %w", err)
	}
	return pdf, nil
}
