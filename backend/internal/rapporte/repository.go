package rapporte

import (
	"context"

	"github.com/google/uuid"
)

// ListReportsFilter holds optional filtering for report list queries.
type ListReportsFilter struct {
	Status   *ReportStatus
	AuthorID *uuid.UUID
	Search   string
}

// Repository defines the persistence interface for the rapporte module.
type Repository interface {
	// Reports
	CreateReport(ctx context.Context, report *WorkReport) error
	UpdateReport(ctx context.Context, report *WorkReport) error
	SoftDeleteReport(ctx context.Context, tenantID, reportID uuid.UUID) error
	GetReport(ctx context.Context, tenantID, reportID uuid.UUID) (*WorkReport, error)
	ListReports(ctx context.Context, tenantID uuid.UUID, filter ListReportsFilter, offset, limit int) ([]*WorkReport, int, error)

	// Lines
	CreateLine(ctx context.Context, line *ReportLine) error
	UpdateLine(ctx context.Context, line *ReportLine) error
	DeleteLine(ctx context.Context, tenantID, lineID uuid.UUID) error
	GetLine(ctx context.Context, tenantID, lineID uuid.UUID) (*ReportLine, error)
	ListLines(ctx context.Context, tenantID, reportID uuid.UUID) ([]*ReportLine, error)

	// Attachments
	CreateAttachment(ctx context.Context, att *ReportAttachment) error
	DeleteAttachment(ctx context.Context, tenantID, attachmentID uuid.UUID) error
	GetAttachment(ctx context.Context, tenantID, attachmentID uuid.UUID) (*ReportAttachment, error)
	ListAttachments(ctx context.Context, tenantID, reportID uuid.UUID, lineID *uuid.UUID) ([]*ReportAttachment, error)
}
