package rapporte

import (
	"time"

	"github.com/google/uuid"
)

// ReportStatus represents the state-machine status of a work report.
type ReportStatus string

const (
	StatusDraft     ReportStatus = "draft"
	StatusSubmitted ReportStatus = "submitted"
	StatusApproved  ReportStatus = "approved"
	StatusRejected  ReportStatus = "rejected"
)

// WorkReport is a field work report created by a technician or employee.
type WorkReport struct {
	ID            uuid.UUID    `json:"id"`
	TenantID      uuid.UUID    `json:"tenant_id"`
	Title         string       `json:"title"`
	Description   string       `json:"description"`
	Status        ReportStatus `json:"status"`
	AuthorID      uuid.UUID    `json:"author_id"`
	ReviewerID    *uuid.UUID   `json:"reviewer_id,omitempty"`
	ReviewedAt    *time.Time   `json:"reviewed_at,omitempty"`
	ReviewNote    string       `json:"review_note"`
	Lat           *float64     `json:"lat,omitempty"`
	Lon           *float64     `json:"lon,omitempty"`
	ReportDate    time.Time    `json:"report_date"`
	SignatureData *string      `json:"signature_data,omitempty"`
	SignedAt      *time.Time   `json:"signed_at,omitempty"`
	SignedBy      *string      `json:"signed_by,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
	DeletedAt     *time.Time   `json:"deleted_at,omitempty"`
}

// ReportLine is a single line item within a work report.
type ReportLine struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	ReportID    uuid.UUID `json:"report_id"`
	Position    int       `json:"position"`
	Description string    `json:"description"`
	Quantity    float64   `json:"quantity"`
	Unit        string    `json:"unit"`
	Notes       string    `json:"notes"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ReportAttachment stores metadata for a file uploaded to MinIO.
type ReportAttachment struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	ReportID    uuid.UUID  `json:"report_id"`
	LineID      *uuid.UUID `json:"line_id,omitempty"`
	Filename    string     `json:"filename"`
	ContentType string     `json:"content_type"`
	SizeBytes   int64      `json:"size_bytes"`
	ObjectKey   string     `json:"object_key"`
	UploadedBy  uuid.UUID  `json:"uploaded_by"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ReportStats holds aggregate counts for a tenant's reports.
type ReportStats struct {
	TotalReports   int
	DraftCount     int
	SubmittedCount int
	ApprovedCount  int
	RejectedCount  int
}
