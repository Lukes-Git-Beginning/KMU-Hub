package rapporte

import (
	"context"
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

// CreateReport creates a new work report in Draft status.
func (s *Service) CreateReport(ctx context.Context, input CreateReportInput) (*WorkReport, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, ErrInvalidInput
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

// ApproveReport transitions a report from Submitted → Approved.
// Guard 2: only Submitted can be approved.
func (s *Service) ApproveReport(ctx context.Context, input ApproveReportInput) (*WorkReport, error) {
	report, err := s.repo.GetReport(ctx, input.TenantID, input.ReportID)
	if err != nil {
		return nil, err
	}

	// Guard 2: pre-check before DB write — only submitted can be approved
	if report.Status != StatusSubmitted {
		if report.Status == StatusApproved {
			return nil, ErrAlreadyApproved
		}
		return nil, ErrInvalidStateTransition
	}

	now := time.Now()
	report.Status = StatusApproved
	report.ReviewerID = &input.ReviewerID
	report.ReviewedAt = &now
	report.ReviewNote = input.ReviewNote
	report.UpdatedAt = now

	if err := s.repo.UpdateReport(ctx, report); err != nil {
		return nil, fmt.Errorf("approve work report: %w", err)
	}

	slog.Info("work report approved",
		"report_id", report.ID,
		"reviewer_id", input.ReviewerID,
	)
	return report, nil
}

// RejectReport transitions a report from Submitted → Rejected.
// Guard 2: only Submitted can be rejected.
func (s *Service) RejectReport(ctx context.Context, input RejectReportInput) (*WorkReport, error) {
	report, err := s.repo.GetReport(ctx, input.TenantID, input.ReportID)
	if err != nil {
		return nil, err
	}

	// Guard 2: pre-check before DB write
	if report.Status != StatusSubmitted {
		if report.Status == StatusApproved {
			return nil, ErrAlreadyApproved
		}
		return nil, ErrInvalidStateTransition
	}

	now := time.Now()
	report.Status = StatusRejected
	report.ReviewerID = &input.ReviewerID
	report.ReviewedAt = &now
	report.ReviewNote = input.ReviewNote
	report.UpdatedAt = now

	if err := s.repo.UpdateReport(ctx, report); err != nil {
		return nil, fmt.Errorf("reject work report: %w", err)
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
func (s *Service) GetReportStats(ctx context.Context, tenantID uuid.UUID) (*ReportStats, error) {
	all, _, err := s.repo.ListReports(ctx, tenantID, ListReportsFilter{}, 0, 100000)
	if err != nil {
		return nil, fmt.Errorf("get report stats: %w", err)
	}

	stats := &ReportStats{}
	for _, r := range all {
		stats.TotalReports++
		switch r.Status {
		case StatusDraft:
			stats.DraftCount++
		case StatusSubmitted:
			stats.SubmittedCount++
		case StatusApproved:
			stats.ApprovedCount++
		case StatusRejected:
			stats.RejectedCount++
		}
	}
	return stats, nil
}

// ListPendingApprovals returns paginated reports in Submitted status.
func (s *Service) ListPendingApprovals(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]*WorkReport, int, error) {
	submitted := StatusSubmitted
	return s.ListReports(ctx, ListReportsInput{
		TenantID: tenantID,
		Status:   &submitted,
		Page:     page,
		PageSize: pageSize,
	})
}

// ExportPDF generates a stub PDF for a report with a customer-signature placeholder.
// TODO Sprint 3: integrate a real PDF library (e.g. gofpdf or chromedp headless).
func (s *Service) ExportPDF(ctx context.Context, tenantID, reportID uuid.UUID) ([]byte, error) {
	report, err := s.repo.GetReport(ctx, tenantID, reportID)
	if err != nil {
		return nil, err
	}

	// Stub: minimal text payload representing a PDF skeleton.
	// Real implementation to follow in Sprint 3.
	stub := fmt.Sprintf(
		"Arbeitsbericht: %s\nStatus: %s\nDatum: %s\n\n[Kundenunterschrift]\n_______________________\n",
		report.Title, report.Status, report.ReportDate.Format("2006-01-02"),
	)
	return []byte(stub), nil
}
