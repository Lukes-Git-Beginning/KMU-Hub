package rapporte

import (
	"database/sql"
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
	// Extended fields (Migration 000162)
	Weather      *string  `json:"weather,omitempty"`
	Temperature  *float64 `json:"temperature,omitempty"`
	WorkStart    *string  `json:"work_start,omitempty"` // HH:MM
	WorkEnd      *string  `json:"work_end,omitempty"`   // HH:MM
	BreakMinutes *int     `json:"break_minutes,omitempty"`
	ProjectName  *string  `json:"project_name,omitempty"`
	Workers      []Worker `json:"workers,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
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
	// Extended fields (Migration 000162)
	Category  *string `json:"category,omitempty"`
	Article   *string `json:"article,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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

// Worker represents an individual assigned to a work report.
type Worker struct {
	ID        uuid.UUID       `json:"id"`
	TenantID  uuid.UUID       `json:"tenant_id"`
	ReportID  uuid.UUID       `json:"report_id"`
	Name      string          `json:"name"`
	Role      sql.NullString  `json:"role"`
	Hours     sql.NullFloat64 `json:"hours"`
	CreatedAt time.Time       `json:"created_at"`
}

// Measurement is a set of measured positions for a project.
type Measurement struct {
	ID         uuid.UUID             `json:"id"`
	TenantID   uuid.UUID             `json:"tenant_id"`
	ReportID   *uuid.UUID            `json:"report_id,omitempty"`
	Title      string                `json:"title"`
	Location   sql.NullString        `json:"location"`
	MeasuredBy sql.NullString        `json:"measured_by"`
	MeasuredAt sql.NullString        `json:"measured_at"` // DATE as string YYYY-MM-DD
	Notes      sql.NullString        `json:"notes"`
	Positions  []MeasurementPosition `json:"positions,omitempty"`
	CreatedAt  time.Time             `json:"created_at"`
	UpdatedAt  time.Time             `json:"updated_at"`
}

// MeasurementPosition is a single line within a Measurement.
type MeasurementPosition struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	MeasurementID  uuid.UUID       `json:"measurement_id"`
	PositionNumber int             `json:"position_number"`
	Description    string          `json:"description"`
	Unit           sql.NullString  `json:"unit"`
	Quantity       sql.NullFloat64 `json:"quantity"`
	UnitPrice      sql.NullFloat64 `json:"unit_price"`
	CreatedAt      time.Time       `json:"created_at"`
}

// ReportTemplate is a reusable template for pre-filling report lines.
type ReportTemplate struct {
	ID               uuid.UUID      `json:"id"`
	TenantID         uuid.UUID      `json:"tenant_id"`
	Name             string         `json:"name"`
	Description      sql.NullString `json:"description"`
	Category         sql.NullString `json:"category"`
	DefaultLinesJSON sql.NullString `json:"default_lines_json"`
	IsActive         bool           `json:"is_active"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}
