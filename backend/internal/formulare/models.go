package formulare

import (
	"time"

	"github.com/google/uuid"
)

// FormSchemaStatus represents the lifecycle state of a form schema.
type FormSchemaStatus string

const (
	FormSchemaStatusDraft    FormSchemaStatus = "draft"
	FormSchemaStatusActive   FormSchemaStatus = "active"
	FormSchemaStatusArchived FormSchemaStatus = "archived"
)

// FormSubmissionStatus represents the review state of a form submission.
type FormSubmissionStatus string

const (
	FormSubmissionStatusNew      FormSubmissionStatus = "new"
	FormSubmissionStatusRead     FormSubmissionStatus = "read"
	FormSubmissionStatusArchived FormSubmissionStatus = "archived"
)

// WebhookDeliveryStatus represents the delivery lifecycle of a webhook attempt.
type WebhookDeliveryStatus string

const (
	WebhookDeliveryStatusPending   WebhookDeliveryStatus = "pending"
	WebhookDeliveryStatusDelivered WebhookDeliveryStatus = "delivered"
	WebhookDeliveryStatusFailed    WebhookDeliveryStatus = "failed"
	WebhookDeliveryStatusDead      WebhookDeliveryStatus = "dead"
)

// ExportFormat represents the file format for submission exports.
type ExportFormat string

const (
	ExportFormatCSV  ExportFormat = "csv"
	ExportFormatXLSX ExportFormat = "xlsx"
)

// FormSchema represents a form definition stored in form_schemas.
type FormSchema struct {
	ID              uuid.UUID        `json:"id"`
	TenantID        uuid.UUID        `json:"tenant_id"`
	Title           string           `json:"title"`
	Description     string           `json:"description"`
	Fields          []byte           `json:"fields"`           // raw JSONB JSON-array of FormField
	Status          FormSchemaStatus `json:"status"`
	IsTemplate      bool             `json:"is_template"`
	IsPublic        bool             `json:"is_public"`
	PageCount       int              `json:"page_count"`
	SubmissionCount int              `json:"submission_count"`
	CreatedBy       *uuid.UUID       `json:"created_by,omitempty"`
	// IntakeTargetID names the module record this form feeds (e.g.
	// "helpdesk_ticket"), a key into the shared intake registry, not a FK.
	// Nil = a plain form, no intake binding.
	IntakeTargetID *string    `json:"intake_target_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

// FormField describes a single field within a form schema (stored as JSONB array).
type FormField struct {
	ID               string          `json:"id"`
	Type             string          `json:"type"`               // text|email|number|select|checkbox|radio|textarea|date|file
	Label            string          `json:"label"`
	Required         bool            `json:"required"`
	Placeholder      string          `json:"placeholder,omitempty"`
	Options          []FieldOption   `json:"options,omitempty"`  // for select/radio/checkbox
	ConditionalLogic *FieldCondition `json:"conditional_logic,omitempty"`
	// Role marks this field as a first-class property of the bound intake
	// target (subject/description/priority/…). Empty/absent means the
	// answer flows into the target record's custom fields instead.
	Role string `json:"role,omitempty"`
}

// FieldOption is a selectable option within a select/radio/checkbox field.
type FieldOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// FieldCondition defines a conditional visibility rule for a form field.
type FieldCondition struct {
	FieldID  string `json:"field_id"`
	Operator string `json:"operator"` // equals|not_equals|contains
	Value    string `json:"value"`
}

// FormSubmission represents a single form response stored in form_submissions.
type FormSubmission struct {
	ID           uuid.UUID            `json:"id"`
	FormSchemaID *uuid.UUID           `json:"form_schema_id,omitempty"`
	TenantID     uuid.UUID            `json:"tenant_id"`
	Answers      []byte               `json:"answers"`      // raw JSONB of field answers
	Status       FormSubmissionStatus `json:"status"`
	SubmittedBy  *string              `json:"submitted_by,omitempty"`
	IPAddress    *string              `json:"ip_address,omitempty"` // DSGVO: nur mit Consent
	SubmittedAt  time.Time            `json:"submitted_at"`
}

// FormWebhook represents a webhook endpoint configured for a form schema.
type FormWebhook struct {
	ID              uuid.UUID  `json:"id"`
	FormSchemaID    uuid.UUID  `json:"form_schema_id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	URL             string     `json:"url"`
	Secret          *string    `json:"secret,omitempty"`
	Events          []string   `json:"events"`
	Active          bool       `json:"active"`
	LastTriggeredAt *time.Time `json:"last_triggered_at,omitempty"`
	LastStatus      *string    `json:"last_status,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// WebhookDelivery represents a single delivery attempt for a webhook.
type WebhookDelivery struct {
	ID               uuid.UUID             `json:"id"`
	WebhookID        uuid.UUID             `json:"webhook_id"`
	SubmissionID     uuid.UUID             `json:"submission_id"`
	TenantID         uuid.UUID             `json:"tenant_id"`
	Payload          []byte                `json:"payload"` // raw JSONB delivery body
	Status           WebhookDeliveryStatus `json:"status"`
	AttemptCount     int                   `json:"attempt_count"`
	MaxAttempts      int                   `json:"max_attempts"`
	NextAttemptAt    time.Time             `json:"next_attempt_at"`
	LastError        *string               `json:"last_error,omitempty"`
	LastResponseCode *int                  `json:"last_response_code,omitempty"`
	CreatedAt        time.Time             `json:"created_at"`
	DeliveredAt      *time.Time            `json:"delivered_at,omitempty"`
}
