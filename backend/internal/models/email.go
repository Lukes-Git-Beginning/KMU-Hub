package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Folder type constants
const (
	FolderTypeInbox   = "inbox"
	FolderTypeSent    = "sent"
	FolderTypeDrafts  = "drafts"
	FolderTypeTrash   = "trash"
	FolderTypeSpam    = "spam"
	FolderTypeArchive = "archive"
	FolderTypeCustom  = "custom"
)

// Link type constants
const (
	LinkTypeAuto   = "auto"
	LinkTypeManual = "manual"
)

// Visibility constants
const (
	VisibilityShared   = "shared"
	VisibilityPersonal = "personal"
)

// EmailAccount represents an IMAP/SMTP email account configured by a user.
type EmailAccount struct {
	ID                uuid.UUID  `json:"id"`
	TenantID          uuid.UUID  `json:"tenant_id"`
	UserID            uuid.UUID  `json:"user_id"`
	EmailAddress      string     `json:"email_address"`
	DisplayName       string     `json:"display_name"`
	IMAPHost          string     `json:"imap_host"`
	IMAPPort          int        `json:"imap_port"`
	SMTPHost          string     `json:"smtp_host"`
	SMTPPort          int        `json:"smtp_port"`
	Username          string     `json:"username"`
	PasswordEncrypted string     `json:"-"` // Never exposed in JSON
	UseSSL            bool       `json:"use_ssl"`
	LastSyncAt        *time.Time `json:"last_sync_at,omitempty"`
	SyncEnabled       bool       `json:"sync_enabled"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// EmailFolder represents a cached IMAP folder.
type EmailFolder struct {
	ID           uuid.UUID `json:"id"`
	AccountID    uuid.UUID `json:"account_id"`
	Name         string    `json:"name"`
	IMAPName     string    `json:"imap_name"`
	FolderType   string    `json:"folder_type"`
	UIDValidity  int64     `json:"uid_validity"`
	HighestUID   int64     `json:"highest_uid"`
	MessageCount int       `json:"message_count"`
	UnreadCount  int       `json:"unread_count"`
	SortOrder    int       `json:"sort_order"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// EmailAddress represents a name + email pair used in message to/cc/bcc fields.
type EmailAddress struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// EmailMessage represents a cached email message.
type EmailMessage struct {
	ID              uuid.UUID      `json:"id"`
	TenantID        uuid.UUID      `json:"tenant_id"`
	AccountID       uuid.UUID      `json:"account_id"`
	FolderID        uuid.UUID      `json:"folder_id"`
	UID             int64          `json:"uid"`
	MessageID       string         `json:"message_id"`
	InReplyTo       string         `json:"in_reply_to"`
	References      []string       `json:"references"`
	ThreadID        *uuid.UUID     `json:"thread_id,omitempty"`
	FromName        string         `json:"from_name"`
	FromEmail       string         `json:"from_email"`
	ToAddresses     []EmailAddress `json:"to_addresses"`
	CcAddresses     []EmailAddress `json:"cc_addresses"`
	BccAddresses    []EmailAddress `json:"bcc_addresses"`
	Subject         string         `json:"subject"`
	Preview         string         `json:"preview"`
	BodyText        string         `json:"body_text"`
	BodyHTML        string         `json:"body_html"`
	IsRead          bool           `json:"is_read"`
	IsStarred       bool           `json:"is_starred"`
	IsDraft         bool           `json:"is_draft"`
	HasAttachments  bool           `json:"has_attachments"`
	Date            time.Time      `json:"date"`
	SizeBytes       int64          `json:"size_bytes"`
	RawHeaders      string         `json:"raw_headers,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// ToAddressesJSON returns the ToAddresses as a JSON byte slice for DB storage.
func (m *EmailMessage) ToAddressesJSON() ([]byte, error) {
	return json.Marshal(m.ToAddresses)
}

// CcAddressesJSON returns the CcAddresses as a JSON byte slice for DB storage.
func (m *EmailMessage) CcAddressesJSON() ([]byte, error) {
	return json.Marshal(m.CcAddresses)
}

// BccAddressesJSON returns the BccAddresses as a JSON byte slice for DB storage.
func (m *EmailMessage) BccAddressesJSON() ([]byte, error) {
	return json.Marshal(m.BccAddresses)
}

// EmailContactLink represents the junction between an email message and a CRM contact.
type EmailContactLink struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	MessageID uuid.UUID `json:"message_id"`
	ContactID uuid.UUID `json:"contact_id"`
	LinkType  string    `json:"link_type"` // auto, manual
	CreatedAt time.Time `json:"created_at"`
}

// EmailAttachment represents metadata for an email attachment stored in MinIO.
type EmailAttachment struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	MessageID   uuid.UUID `json:"message_id"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	MinIOKey    string    `json:"minio_key"`
	ContentID   string    `json:"content_id,omitempty"` // For inline images (CID)
	IsInline    bool      `json:"is_inline"`
	CreatedAt   time.Time `json:"created_at"`
}

// EmailSignature represents a per-user HTML email signature.
type EmailSignature struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	UserID      uuid.UUID `json:"user_id"`
	Name        string    `json:"name"`
	HTMLContent string    `json:"html_content"`
	IsDefault   bool      `json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Email rule field, operator and action values. They match the frontend's
// EmailRuleInfo unions and the CHECK constraints of migration 000260.
const (
	EmailRuleFieldFrom    = "from"
	EmailRuleFieldSubject = "subject"

	EmailRuleOpContains = "contains"

	EmailRuleActionLabel = "label"
	EmailRuleActionMove  = "move"
)

// EmailRule is one stored "if <field> <op> <value> then <action>" rule of the
// mails module, evaluated against a message by the shared automation condition
// evaluator.
type EmailRule struct {
	ID       uuid.UUID `json:"id"`
	TenantID uuid.UUID `json:"tenant_id"`
	Name     string    `json:"name"`
	Field    string    `json:"field"`
	Op       string    `json:"op"`
	Value    string    `json:"value"`
	// ActionType is EmailRuleActionLabel or EmailRuleActionMove.
	ActionType string `json:"action_type"`
	// ActionTarget is a label id (label action) or folder id (move action).
	ActionTarget uuid.UUID `json:"action_target"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// EmailRuleCandidate is the slim projection of a message that rule evaluation
// needs: the two matchable fields plus the two mutable targets. Loading whole
// messages (bodies included) for a bulk apply would be wasteful.
type EmailRuleCandidate struct {
	ID        uuid.UUID
	FromName  string
	FromEmail string
	Subject   string
	FolderID  uuid.UUID
	LabelIDs  []uuid.UUID
}
