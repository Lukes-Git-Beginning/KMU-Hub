package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ============================================================================
// HR Status Constants
// ============================================================================

// LeaveRequestStatus represents the state of a leave request.
type LeaveRequestStatus string

const (
	LeaveStatusPending   LeaveRequestStatus = "pending"
	LeaveStatusApproved  LeaveRequestStatus = "approved"
	LeaveStatusRejected  LeaveRequestStatus = "rejected"
	LeaveStatusCancelled LeaveRequestStatus = "cancelled"
)

// WorkTimeEntryStatus represents the state of a work time entry.
type WorkTimeEntryStatus string

const (
	WorkTimeStatusActive             WorkTimeEntryStatus = "active"
	WorkTimeStatusCompleted          WorkTimeEntryStatus = "completed"
	WorkTimeStatusCorrectionPending  WorkTimeEntryStatus = "correction_pending"
	WorkTimeStatusCorrectionApproved WorkTimeEntryStatus = "correction_approved"
)

// HRContractType represents the type of employment contract.
type HRContractType string

const (
	HRContractFullTime  HRContractType = "full_time"
	HRContractPartTime  HRContractType = "part_time"
	HRContractMiniJob   HRContractType = "mini_job"
	HRContractIntern    HRContractType = "intern"
	HRContractTemporary HRContractType = "temporary"
)

// HRDocVisibility represents the visibility level of HR documents.
type HRDocVisibility string

const (
	HRDocVisibilityHROnly   HRDocVisibility = "hr_only"
	HRDocVisibilityManager  HRDocVisibility = "manager"
	HRDocVisibilityEmployee HRDocVisibility = "employee"
)

// HalfDayPeriod represents morning or afternoon for half-day leave.
type HalfDayPeriod string

const (
	HalfDayMorning   HalfDayPeriod = "morning"
	HalfDayAfternoon HalfDayPeriod = "afternoon"
)

// ============================================================================
// HR Domain Models
// ============================================================================

// HRCompanySettings holds per-tenant HR configuration.
type HRCompanySettings struct {
	ID                    uuid.UUID `json:"id"`
	TenantID              uuid.UUID `json:"tenant_id"`
	AUThresholdDays       int       `json:"au_threshold_days"`
	ShowAbsenceReason     bool      `json:"show_absence_reason"`
	DefaultAnnualLeaveDays int      `json:"default_annual_leave_days"`
	Timezone              string    `json:"timezone"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// EmployeeProfile extends users with HR-specific data.
type EmployeeProfile struct {
	ID                    uuid.UUID      `json:"id"`
	UserID                uuid.UUID      `json:"user_id"`
	Department            string         `json:"department"`
	PositionTitle         string         `json:"position_title"`
	ContractType          HRContractType `json:"contract_type"`
	WorkDaysPerWeek       int            `json:"work_days_per_week"`
	AnnualLeaveDays       int            `json:"annual_leave_days"`
	ManagerUserID         *uuid.UUID     `json:"manager_user_id,omitempty"`
	StartDate             time.Time      `json:"start_date"`
	EmergencyContactName  string           `json:"emergency_contact_name"`
	EmergencyContactPhone string           `json:"emergency_contact_phone"`
	AddressStreet         string           `json:"address_street"`
	AddressCity           string           `json:"address_city"`
	AddressPostalCode     string           `json:"address_postal_code"`
	AddressCountry        string           `json:"address_country"`
	HourlyRate            *decimal.Decimal `json:"hourly_rate,omitempty"`
	CreatedAt             time.Time        `json:"created_at"`
	UpdatedAt             time.Time        `json:"updated_at"`

	// Denormalized fields (populated by queries, not stored)
	UserName     string `json:"user_name,omitempty"`
	UserEmail    string `json:"user_email,omitempty"`
	ManagerName  string `json:"manager_name,omitempty"`
}

// LeaveType defines a type of leave (system or admin-created).
type LeaveType struct {
	ID                  uuid.UUID `json:"id"`
	TenantID            uuid.UUID `json:"tenant_id"`
	Name                string    `json:"name"`
	Key                 string    `json:"key"`
	Color               string    `json:"color"`
	DeductsFromBalance  bool      `json:"deducts_from_balance"`
	RequiresApproval    bool      `json:"requires_approval"`
	RequiresAUDocument  bool      `json:"requires_au_document"`
	IsSystem            bool      `json:"is_system"`
	SortOrder           int       `json:"sort_order"`
	CreatedAt           time.Time `json:"created_at"`
}

// LeaveRequest represents a leave/vacation request.
type LeaveRequest struct {
	ID                  uuid.UUID          `json:"id"`
	TenantID            uuid.UUID          `json:"tenant_id"`
	EmployeeID          uuid.UUID          `json:"employee_id"`
	LeaveTypeID         uuid.UUID          `json:"leave_type_id"`
	StartDate           time.Time          `json:"start_date"`
	EndDate             time.Time          `json:"end_date"`
	IsHalfDayStart      bool               `json:"is_half_day_start"`
	HalfDayPeriodStart  *HalfDayPeriod     `json:"half_day_period_start,omitempty"`
	IsHalfDayEnd        bool               `json:"is_half_day_end"`
	HalfDayPeriodEnd    *HalfDayPeriod     `json:"half_day_period_end,omitempty"`
	TotalDays           decimal.Decimal    `json:"total_days"`
	Reason              string             `json:"reason"`
	Status              LeaveRequestStatus `json:"status"`
	ApprovedBy          *uuid.UUID         `json:"approved_by,omitempty"`
	ApprovalComment     string             `json:"approval_comment"`
	ApprovedAt          *time.Time         `json:"approved_at,omitempty"`
	AUDocumentRequired  bool               `json:"au_document_required"`
	AUDocumentFileID    *uuid.UUID         `json:"au_document_file_id,omitempty"`
	CreatedAt           time.Time          `json:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at"`

	// Denormalized fields
	EmployeeName   string `json:"employee_name,omitempty"`
	LeaveTypeName  string `json:"leave_type_name,omitempty"`
	LeaveTypeColor string `json:"leave_type_color,omitempty"`
}

// HRLeaveBalance tracks per-employee per-year leave entitlement and usage.
type HRLeaveBalance struct {
	ID                uuid.UUID       `json:"id"`
	TenantID          uuid.UUID       `json:"tenant_id"`
	EmployeeID        uuid.UUID       `json:"employee_id"`
	Year              int             `json:"year"`
	Entitlement       decimal.Decimal `json:"entitlement"`
	CarriedOver       decimal.Decimal `json:"carried_over"`
	Used              decimal.Decimal `json:"used"`
	Remaining         decimal.Decimal `json:"remaining"`
	CarryoverExpiresAt *time.Time     `json:"carryover_expires_at,omitempty"`
	CarryoverNotified bool            `json:"carryover_notified"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// HRWorkTimeEntry represents a clock in/out record.
type HRWorkTimeEntry struct {
	ID                    uuid.UUID           `json:"id"`
	TenantID              uuid.UUID           `json:"tenant_id"`
	EmployeeID            uuid.UUID           `json:"employee_id"`
	ClockIn               time.Time           `json:"clock_in"`
	ClockOut              *time.Time          `json:"clock_out,omitempty"`
	BreakMinutes          int                 `json:"break_minutes"`
	AutoBreakDeducted     int                 `json:"auto_break_deducted"`
	NetWorkMinutes        *int                `json:"net_work_minutes,omitempty"`
	Status                WorkTimeEntryStatus `json:"status"`
	IsCorrection          bool                `json:"is_correction"`
	OriginalEntryID       *uuid.UUID          `json:"original_entry_id,omitempty"`
	CorrectionReason      string              `json:"correction_reason"`
	CorrectionApprovedBy  *uuid.UUID          `json:"correction_approved_by,omitempty"`
	CorrectionApprovedAt  *time.Time          `json:"correction_approved_at,omitempty"`
	CreatedAt             time.Time           `json:"created_at"`
	UpdatedAt             time.Time           `json:"updated_at"`

	// Nested break entries (populated on detail fetch)
	Breaks []HRBreakEntry `json:"breaks,omitempty"`

	// Denormalized fields
	EmployeeName string `json:"employee_name,omitempty"`
}

// HRBreakEntry represents a break period within a work time entry.
type HRBreakEntry struct {
	ID              uuid.UUID  `json:"id"`
	WorkTimeEntryID uuid.UUID  `json:"work_time_entry_id"`
	StartTime       time.Time  `json:"start_time"`
	EndTime         *time.Time `json:"end_time,omitempty"`
	DurationMinutes *int       `json:"duration_minutes,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// HRDocumentCategory defines categories for HR documents.
type HRDocumentCategory struct {
	ID         uuid.UUID       `json:"id"`
	TenantID   uuid.UUID       `json:"tenant_id"`
	Name       string          `json:"name"`
	Key        string          `json:"key"`
	Visibility HRDocVisibility `json:"visibility"`
	IsSystem   bool            `json:"is_system"`
	SortOrder  int             `json:"sort_order"`
	CreatedAt  time.Time       `json:"created_at"`
}

// EmployeeDocument links a file from the document service to an HR context.
type EmployeeDocument struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	EmployeeID  uuid.UUID `json:"employee_id"`
	CategoryID  uuid.UUID `json:"category_id"`
	FileID      uuid.UUID `json:"file_id"`
	UploadedBy  uuid.UUID `json:"uploaded_by"`
	Notes       string    `json:"notes"`
	CreatedAt   time.Time `json:"created_at"`

	// Denormalized fields
	CategoryName   string `json:"category_name,omitempty"`
	FileName       string `json:"file_name,omitempty"`
	FileSize       string `json:"file_size,omitempty"`
	UploadedByName string `json:"uploaded_by_name,omitempty"`
}
