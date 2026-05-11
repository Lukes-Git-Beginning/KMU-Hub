package models

import (
	"time"

	"github.com/google/uuid"
)

// GDPR export status constants
const (
	ExportStatusPending    = "pending"
	ExportStatusApproved   = "approved"
	ExportStatusDenied     = "denied"
	ExportStatusProcessing = "processing"
	ExportStatusReady      = "ready"
	ExportStatusDownloaded = "downloaded"
	ExportStatusExpired    = "expired"
)

// IP access rule type constants
const (
	IPRuleAllow = "allow"
	IPRuleBlock = "block"
)

// Audit result constants
const (
	AuditResultSuccess = "success"
	AuditResultFailure = "failure"
)

// AuditEntry represents a single entry in the tamper-evident audit log.
type AuditEntry struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	SequenceNum  int64      `json:"sequence_num"`
	Timestamp    time.Time  `json:"timestamp"`
	UserID       *uuid.UUID `json:"user_id,omitempty"`
	Action       string     `json:"action"`
	Target       string     `json:"target,omitempty"`
	TargetType   string     `json:"target_type,omitempty"`
	Details      string     `json:"details,omitempty"` // JSON string
	IPAddress    string     `json:"ip_address,omitempty"`
	UserAgent    string     `json:"user_agent,omitempty"`
	Result       string     `json:"result"`
	PreviousHash string     `json:"previous_hash"`
	EntryHash    string     `json:"entry_hash"`
}

// AuditFilter defines query parameters for listing audit entries.
type AuditFilter struct {
	TenantID uuid.UUID  `json:"tenant_id"` // Required: always filters by tenant
	DateFrom *time.Time `json:"date_from,omitempty"`
	DateTo   *time.Time `json:"date_to,omitempty"`
	UserID   *uuid.UUID `json:"user_id,omitempty"`
	Action   string     `json:"action,omitempty"`
	Result   string     `json:"result,omitempty"`
	Offset   int        `json:"offset"`
	Limit    int        `json:"limit"`
}

// RecoveryCode represents a hashed single-use 2FA recovery code.
type RecoveryCode struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	UserID    uuid.UUID  `json:"user_id"`
	CodeHash  string     `json:"-"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// TwoFactorPolicy defines per-role 2FA enforcement settings.
type TwoFactorPolicy struct {
	ID              uuid.UUID  `json:"id"`
	RoleName        string     `json:"role_name"`
	Enforced        bool       `json:"enforced"`
	GracePeriodDays int        `json:"grace_period_days"`
	UpdatedAt       time.Time  `json:"updated_at"`
	UpdatedBy       *uuid.UUID `json:"updated_by,omitempty"`
}

// UserSession represents an active user session with device metadata.
type UserSession struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	UserID         uuid.UUID  `json:"user_id"`
	RefreshTokenID *uuid.UUID `json:"refresh_token_id,omitempty"`
	DeviceName     string     `json:"device_name,omitempty"`
	DeviceType     string     `json:"device_type,omitempty"`
	IPAddress      string     `json:"ip_address,omitempty"`
	Location       string     `json:"location,omitempty"`
	UserAgent      string     `json:"user_agent,omitempty"`
	IsCurrent      bool       `json:"is_current"`
	LastActiveAt   time.Time  `json:"last_active_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

// VaultSecret represents an encrypted secret stored in the vault.
type VaultSecret struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	KeyName        string     `json:"key_name"`
	EncryptedValue string     `json:"-"`
	KeyVersion     int        `json:"key_version"`
	Description    string     `json:"description,omitempty"`
	CreatedBy      *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// GDPRExportRequest tracks a user's data export request through the approval workflow.
type GDPRExportRequest struct {
	ID                uuid.UUID  `json:"id"`
	TenantID          uuid.UUID  `json:"tenant_id"`
	UserID            uuid.UUID  `json:"user_id"`
	Status            string     `json:"status"`
	RequestedAt       time.Time  `json:"requested_at"`
	ReviewedBy        *uuid.UUID `json:"reviewed_by,omitempty"`
	ReviewedAt        *time.Time `json:"reviewed_at,omitempty"`
	ReviewNote        string     `json:"review_note,omitempty"`
	ExportData        []byte     `json:"-"`
	DownloadToken     string     `json:"-"`
	DownloadExpiresAt *time.Time `json:"download_expires_at,omitempty"`
	DownloadedAt      *time.Time `json:"downloaded_at,omitempty"`
}

// GDPRErasureLog records details of a GDPR right-to-erasure execution.
type GDPRErasureLog struct {
	ID               uuid.UUID         `json:"id"`
	TenantID         uuid.UUID         `json:"tenant_id"`
	OriginalUserID   uuid.UUID         `json:"original_user_id"`
	AnonymizedLabel  string            `json:"anonymized_label"`
	ExecutedBy       uuid.UUID         `json:"executed_by"`
	ExecutedAt       time.Time         `json:"executed_at"`
	ModulesAffected  map[string]string `json:"modules_affected"`
	ConfirmationHash string            `json:"confirmation_hash"`
}

// PasswordPolicy defines organization-wide password requirements.
type PasswordPolicy struct {
	ID                uuid.UUID  `json:"id"`
	TenantID          uuid.UUID  `json:"tenant_id"`
	MinLength         int        `json:"min_length"`
	RequireUppercase  bool       `json:"require_uppercase"`
	RequireLowercase  bool       `json:"require_lowercase"`
	RequireDigit      bool       `json:"require_digit"`
	RequireSpecial    bool       `json:"require_special"`
	MinEntropy        float64    `json:"min_entropy"`
	MaxAgeDays        *int       `json:"max_age_days,omitempty"`
	PreventReuseCount int        `json:"prevent_reuse_count"`
	UpdatedAt         time.Time  `json:"updated_at"`
	UpdatedBy         *uuid.UUID `json:"updated_by,omitempty"`
}

// PasswordHistoryEntry records a previous password hash for reuse prevention.
type PasswordHistoryEntry struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	UserID       uuid.UUID `json:"user_id"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// IPAccessRule defines an IP allow/block list entry.
type IPAccessRule struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	IPCIDR      string     `json:"ip_cidr"`
	RuleType    string     `json:"rule_type"`
	Description string     `json:"description,omitempty"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// LoginResult extends auth login to support 2FA pending state.
type LoginResult struct {
	User              *User  `json:"user,omitempty"`
	AccessToken       string `json:"access_token,omitempty"`
	RefreshToken      string `json:"refresh_token,omitempty"`
	RequiresTwoFactor bool   `json:"requires_two_factor"`
	PendingToken      string `json:"pending_token,omitempty"`
}

// ModuleErasurePreview describes what erasure will affect in one module.
type ModuleErasurePreview struct {
	ModuleName  string `json:"module_name"`
	RecordCount int    `json:"record_count"`
	Action      string `json:"action"` // "anonymize" or "delete"
}
