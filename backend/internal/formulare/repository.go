package formulare

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ListSchemasFilter narrows a form schema listing.
type ListSchemasFilter struct {
	Status     *FormSchemaStatus
	IsTemplate *bool
	Search     string
	Limit      int
	Offset     int
}

// ListSubmissionsFilter narrows a form submission listing.
type ListSubmissionsFilter struct {
	FormSchemaID *uuid.UUID
	Status       *FormSubmissionStatus
	Limit        int
	Offset       int
}

// ExportFilter narrows a submission export query.
type ExportFilter struct {
	Status          *FormSubmissionStatus
	SubmittedAfter  *time.Time
	SubmittedBefore *time.Time
}

// ListDeliveriesFilter narrows a webhook delivery listing.
type ListDeliveriesFilter struct {
	WebhookID    *uuid.UUID
	SubmissionID *uuid.UUID
	Status       *WebhookDeliveryStatus
	Limit        int
	Offset       int
}

// Repository defines the persistence interface for the formulare module.
type Repository interface {
	// Schema
	CreateSchema(ctx context.Context, schema *FormSchema) error
	GetSchema(ctx context.Context, id, tenantID uuid.UUID) (*FormSchema, error)
	UpdateSchema(ctx context.Context, schema *FormSchema) error
	SoftDeleteSchema(ctx context.Context, id, tenantID uuid.UUID) error
	ListSchemas(ctx context.Context, tenantID uuid.UUID, filter ListSchemasFilter) ([]*FormSchema, int, error)
	DuplicateSchema(ctx context.Context, id, tenantID uuid.UUID, newTitle string) (*FormSchema, error)

	// Submission
	CreateSubmission(ctx context.Context, submission *FormSubmission, webhookIDs []uuid.UUID) error
	GetSubmission(ctx context.Context, id, tenantID uuid.UUID) (*FormSubmission, error)
	ListSubmissions(ctx context.Context, tenantID uuid.UUID, filter ListSubmissionsFilter) ([]*FormSubmission, int, error)
	UpdateSubmissionStatus(ctx context.Context, id, tenantID uuid.UUID, status FormSubmissionStatus) error
	ListSubmissionsForExport(ctx context.Context, formSchemaID, tenantID uuid.UUID, filter ExportFilter) ([]*FormSubmission, error)

	// Webhook
	CreateWebhook(ctx context.Context, webhook *FormWebhook) error
	GetWebhook(ctx context.Context, id, tenantID uuid.UUID) (*FormWebhook, error)
	UpdateWebhook(ctx context.Context, webhook *FormWebhook) error
	DeleteWebhook(ctx context.Context, id, tenantID uuid.UUID) error
	ListWebhooks(ctx context.Context, formSchemaID, tenantID uuid.UUID) ([]*FormWebhook, error)
	ListActiveWebhooksForSchema(ctx context.Context, formSchemaID uuid.UUID) ([]*FormWebhook, error)

	// Delivery (used by Worker in Welle 2)
	ClaimPendingDeliveries(ctx context.Context, batchSize int) ([]*WebhookDelivery, error)
	MarkDeliveryResult(ctx context.Context, id uuid.UUID, status WebhookDeliveryStatus, attemptCount int, nextAttemptAt *time.Time, lastError string, lastCode int) error
	ListWebhookDeliveries(ctx context.Context, tenantID uuid.UUID, filter ListDeliveriesFilter) ([]*WebhookDelivery, int, error)
	GetWebhookDelivery(ctx context.Context, id, tenantID uuid.UUID) (*WebhookDelivery, error)
}
