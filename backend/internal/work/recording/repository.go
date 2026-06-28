package recording

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository defines the interface for recording persistence
type Repository interface {
	CreateRecording(ctx context.Context, rec *Recording) error
	GetRecording(ctx context.Context, id uuid.UUID) (*Recording, error)
	GetRecordingByEgressID(ctx context.Context, egressID string) (*Recording, error)
	UpdateRecording(ctx context.Context, rec *Recording) error
	TagRecordingWithConsents(ctx context.Context, recordingID uuid.UUID, snapshot []ParticipantConsentInfo) error
	ListRecordingsByCall(ctx context.Context, callID uuid.UUID) ([]Recording, error)
	ListRecordingsByMeeting(ctx context.Context, meetingID uuid.UUID) ([]Recording, error)
	DeleteRecording(ctx context.Context, id uuid.UUID) error
	SetConsent(ctx context.Context, consent *RecordingConsent) error
	GetConsents(ctx context.Context, recordingID uuid.UUID) ([]RecordingConsent, error)
	GetConsentsWithUser(ctx context.Context, recordingID uuid.UUID) ([]RecordingConsentWithUser, error)
	CountPendingConsents(ctx context.Context, recordingID uuid.UUID, participantIDs []uuid.UUID) (int, error)
	ListExpiredRecordings(ctx context.Context, before time.Time) ([]Recording, error)
	ListRecordingsWithAccess(ctx context.Context, userID uuid.UUID, callID, meetingID *uuid.UUID) ([]Recording, error)
	GetRecordingParticipants(ctx context.Context, recordingID uuid.UUID) ([]uuid.UUID, error)
	// Pre-recording consent (Migration 000107)
	MarkInitiatorConsent(ctx context.Context, recordingID, userID, tenantID uuid.UUID) error
	GetPreConsentStatus(ctx context.Context, recordingID, tenantID uuid.UUID) (bool, error)
}
