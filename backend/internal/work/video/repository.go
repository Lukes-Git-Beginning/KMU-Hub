package video

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the interface for call session persistence
type Repository interface {
	CreateCallSession(ctx context.Context, session *CallSession) error
	GetCallSession(ctx context.Context, id uuid.UUID) (*CallSession, error)
	GetCallSessionByRoomName(ctx context.Context, roomName string) (*CallSession, error)
	UpdateCallSession(ctx context.Context, session *CallSession) error
	ListActiveCallsForUser(ctx context.Context, userID uuid.UUID) ([]CallSession, error)
	AddParticipant(ctx context.Context, p *CallParticipant) error
	RemoveParticipant(ctx context.Context, callID, userID uuid.UUID) error
	GetParticipants(ctx context.Context, callID uuid.UUID) ([]CallParticipant, error)
	CountActiveParticipants(ctx context.Context, callID uuid.UUID) (int, error)
}
