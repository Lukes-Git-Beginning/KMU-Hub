package gdpr

import (
	"context"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ErasureAction defines the type of erasure operation for a module.
type ErasureAction string

const (
	// ErasureAnonymize replaces personal data with anonymized labels.
	ErasureAnonymize ErasureAction = "anonymize"
	// ErasureDelete removes records entirely.
	ErasureDelete ErasureAction = "delete"
	// ErasureRetain keeps records unchanged (e.g., audit logs for compliance).
	ErasureRetain ErasureAction = "retain"
)

// ErasureHandler defines the interface for per-module GDPR right-to-erasure execution.
// Each module registers a handler to preview and execute user data erasure.
type ErasureHandler interface {
	// ModuleName returns the unique module identifier (e.g., "auth", "crm", "chat").
	ModuleName() string

	// PreviewErasure returns a preview of what erasure will affect in this module.
	// Does not modify any data.
	PreviewErasure(ctx context.Context, userID uuid.UUID) (*models.ModuleErasurePreview, error)

	// ExecuteErasure performs the actual data erasure/anonymization for the given user.
	// The anonymizedLabel is used to replace personal data (e.g., "Geloeschter Benutzer #42").
	// The action specifies the erasure type (anonymize, delete, retain).
	// Returns the number of records affected.
	ExecuteErasure(ctx context.Context, userID uuid.UUID, anonymizedLabel string, action ErasureAction) (int, error)
}

// ---------------------------------------------------------------------------
// Per-module erasure handlers
// ---------------------------------------------------------------------------

// AuthErasureHandler handles erasure for authentication data.
// Anonymizes user profile, removes sessions, clears 2FA secrets.
type AuthErasureHandler struct{}

func (h *AuthErasureHandler) ModuleName() string { return "auth" }

func (h *AuthErasureHandler) PreviewErasure(ctx context.Context, userID uuid.UUID) (*models.ModuleErasurePreview, error) {
	// Real implementation queries user record, sessions, recovery codes.
	// For now, returns a representative preview.
	return &models.ModuleErasurePreview{
		ModuleName:  "auth",
		RecordCount: 1, // User profile record
		Action:      string(ErasureAnonymize),
	}, nil
}

func (h *AuthErasureHandler) ExecuteErasure(ctx context.Context, userID uuid.UUID, anonymizedLabel string, action ErasureAction) (int, error) {
	// Real implementation will:
	// 1. Replace user name/email with anonymizedLabel
	// 2. Clear 2FA secrets (two_factor_secret_encrypted, two_factor_pending_secret)
	// 3. Delete all sessions
	// 4. Delete all recovery codes
	// 5. Delete password history
	// Returns count of affected records.
	return 1, nil
}

// CRMErasureHandler handles erasure for CRM data.
// Anonymizes contacts, deals, and activities owned by the user.
type CRMErasureHandler struct{}

func (h *CRMErasureHandler) ModuleName() string { return "crm" }

func (h *CRMErasureHandler) PreviewErasure(ctx context.Context, userID uuid.UUID) (*models.ModuleErasurePreview, error) {
	// Stub: will query contacts, deals, activities created by user
	return &models.ModuleErasurePreview{
		ModuleName:  "crm",
		RecordCount: 0,
		Action:      string(ErasureAnonymize),
	}, nil
}

func (h *CRMErasureHandler) ExecuteErasure(ctx context.Context, userID uuid.UUID, anonymizedLabel string, action ErasureAction) (int, error) {
	// Stub: will anonymize CRM records (replace user references with anonymizedLabel)
	return 0, nil
}

// ChatErasureHandler handles erasure for chat data.
// Anonymizes messages and removes channel memberships.
type ChatErasureHandler struct{}

func (h *ChatErasureHandler) ModuleName() string { return "chat" }

func (h *ChatErasureHandler) PreviewErasure(ctx context.Context, userID uuid.UUID) (*models.ModuleErasurePreview, error) {
	// Stub: will query messages, channel memberships
	return &models.ModuleErasurePreview{
		ModuleName:  "chat",
		RecordCount: 0,
		Action:      string(ErasureAnonymize),
	}, nil
}

func (h *ChatErasureHandler) ExecuteErasure(ctx context.Context, userID uuid.UUID, anonymizedLabel string, action ErasureAction) (int, error) {
	// Stub: will anonymize messages (replace sender with anonymizedLabel), remove memberships
	return 0, nil
}

// WorkErasureHandler handles erasure for work/project data.
// Anonymizes tasks, time entries, and comments owned by the user.
type WorkErasureHandler struct{}

func (h *WorkErasureHandler) ModuleName() string { return "work" }

func (h *WorkErasureHandler) PreviewErasure(ctx context.Context, userID uuid.UUID) (*models.ModuleErasurePreview, error) {
	// Stub: will query tasks, time entries, comments
	return &models.ModuleErasurePreview{
		ModuleName:  "work",
		RecordCount: 0,
		Action:      string(ErasureAnonymize),
	}, nil
}

func (h *WorkErasureHandler) ExecuteErasure(ctx context.Context, userID uuid.UUID, anonymizedLabel string, action ErasureAction) (int, error) {
	// Stub: will anonymize work records (reassign to anonymizedLabel, clear personal data)
	return 0, nil
}

// CalendarErasureHandler handles erasure for calendar data.
// Deletes personal calendars and anonymizes shared event participation.
type CalendarErasureHandler struct{}

func (h *CalendarErasureHandler) ModuleName() string { return "calendar" }

func (h *CalendarErasureHandler) PreviewErasure(ctx context.Context, userID uuid.UUID) (*models.ModuleErasurePreview, error) {
	// Stub: will query calendars, events, bookings
	return &models.ModuleErasurePreview{
		ModuleName:  "calendar",
		RecordCount: 0,
		Action:      string(ErasureDelete),
	}, nil
}

func (h *CalendarErasureHandler) ExecuteErasure(ctx context.Context, userID uuid.UUID, anonymizedLabel string, action ErasureAction) (int, error) {
	// Stub: will delete personal calendars, anonymize shared event participations
	return 0, nil
}

// NotificationErasureHandler handles erasure for notification data.
// Deletes notification preferences and delivery history.
type NotificationErasureHandler struct{}

func (h *NotificationErasureHandler) ModuleName() string { return "notifications" }

func (h *NotificationErasureHandler) PreviewErasure(ctx context.Context, userID uuid.UUID) (*models.ModuleErasurePreview, error) {
	// Stub: will query notification preferences and history
	return &models.ModuleErasurePreview{
		ModuleName:  "notifications",
		RecordCount: 0,
		Action:      string(ErasureDelete),
	}, nil
}

func (h *NotificationErasureHandler) ExecuteErasure(ctx context.Context, userID uuid.UUID, anonymizedLabel string, action ErasureAction) (int, error) {
	// Stub: will delete notification preferences and history
	return 0, nil
}

// AuditErasureHandler is a no-op handler for audit logs.
// Audit logs are retained for compliance requirements (cannot be erased per DSGVO Art. 17(3)(e)).
type AuditErasureHandler struct{}

func (h *AuditErasureHandler) ModuleName() string { return "audit" }

func (h *AuditErasureHandler) PreviewErasure(ctx context.Context, userID uuid.UUID) (*models.ModuleErasurePreview, error) {
	// Audit logs are retained for legal compliance -- no erasure possible.
	return &models.ModuleErasurePreview{
		ModuleName:  "audit",
		RecordCount: 0,
		Action:      string(ErasureRetain),
	}, nil
}

func (h *AuditErasureHandler) ExecuteErasure(ctx context.Context, userID uuid.UUID, anonymizedLabel string, action ErasureAction) (int, error) {
	// No-op: audit logs must be retained for legal compliance (DSGVO Art. 17(3)(e)).
	return 0, nil
}
