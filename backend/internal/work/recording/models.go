package recording

import (
	"time"

	"github.com/google/uuid"
)

// Recording status constants
const (
	RecordingStatusActive     = "active"
	RecordingStatusProcessing = "processing"
	RecordingStatusCompleted  = "completed"
	RecordingStatusFailed     = "failed"
	RecordingStatusDeleted    = "deleted"
)

// ValidRecordingStatuses contains all valid recording status values
var ValidRecordingStatuses = map[string]bool{
	RecordingStatusActive:     true,
	RecordingStatusProcessing: true,
	RecordingStatusCompleted:  true,
	RecordingStatusFailed:     true,
	RecordingStatusDeleted:    true,
}

// Recording represents a recorded call or meeting session
type Recording struct {
	ID                 uuid.UUID  `json:"id"`
	CallID             *uuid.UUID `json:"call_id,omitempty"`
	MeetingID          *uuid.UUID `json:"meeting_id,omitempty"`
	Status             string     `json:"status"`
	EgressID           *string    `json:"egress_id,omitempty"`
	FileURL            *string    `json:"file_url,omitempty"`
	FileSizeBytes      *int64     `json:"file_size_bytes,omitempty"`
	DurationSeconds    *int       `json:"duration_seconds,omitempty"`
	RetentionExpiresAt *time.Time `json:"retention_expires_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

// RecordingConsent represents a participant's consent for being recorded
type RecordingConsent struct {
	RecordingID uuid.UUID  `json:"recording_id"`
	UserID      uuid.UUID  `json:"user_id"`
	Consented   bool       `json:"consented"`
	RespondedAt time.Time  `json:"responded_at"`
}

// RecordingConsentWithUser includes denormalized user info
type RecordingConsentWithUser struct {
	RecordingConsent
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// RecordingConsentStatus aggregates consent info for a recording
type RecordingConsentStatus struct {
	RecordingID  uuid.UUID                  `json:"recording_id"`
	Consents     []RecordingConsentWithUser  `json:"consents"`
	AllConsented bool                        `json:"all_consented"`
}
