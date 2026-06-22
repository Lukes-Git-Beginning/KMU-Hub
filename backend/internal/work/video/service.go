package video

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/middleware"
)

// MaxGroupParticipants is the maximum number of participants in a group call
const MaxGroupParticipants = 25

// MaxOneToOneParticipants is the maximum number of participants in a 1:1 call
const MaxOneToOneParticipants = 2

// IceServerConfig carries per-session WebRTC ICE/TURN credentials (RTCIceServer
// shape) so the client can apply them to RTCConfiguration before opening the
// peer connection. It is the domain-side mirror of the LiveKit adapter's
// IceServer, keeping this package decoupled from the livekit package.
type IceServerConfig struct {
	URLs       []string
	Username   string
	Credential string
}

// RoomManager abstracts LiveKit room operations for testability
type RoomManager interface {
	CreateRoom(ctx context.Context, name string, maxParticipants uint32) error
	DeleteRoom(ctx context.Context, name string) error
	GenerateToken(roomName, userID, displayName string) (string, error)
	// TURNIceServers returns per-session TURN credentials for the given user, or
	// nil when TURN is not configured. Callers must treat nil as "no relay".
	TURNIceServers(userID string) []IceServerConfig
	// ListParticipants returns the identity strings of all currently connected
	// participants in the given room. Returns an empty slice when the room is empty
	// or does not exist.
	ListParticipants(ctx context.Context, roomName string) ([]string, error)
	// MuteParticipant server-mutes all published tracks of the given participant.
	// Idempotent — no-op when the participant is not in the room.
	MuteParticipant(ctx context.Context, roomName, identity string) error
	// RemoveParticipant forcibly disconnects a participant from the room.
	// Idempotent — no-op when the participant is not in the room.
	RemoveParticipant(ctx context.Context, roomName, identity string) error
}

// Service handles video call business logic
type Service struct {
	repo    Repository
	roomMgr RoomManager // nil if LiveKit disabled
	enabled bool
}

// NewService creates a new video call service.
// If roomMgr is nil, the service operates in disabled mode and returns
// ErrLiveKitNotConfigured for operations that require LiveKit.
func NewService(repo Repository, roomMgr RoomManager) *Service {
	return &Service{
		repo:    repo,
		roomMgr: roomMgr,
		enabled: roomMgr != nil,
	}
}

// CreateCall initiates a new video/voice call.
// Returns the call session, a join token for the initiator, and any error.
func (s *Service) CreateCall(ctx context.Context, initiatorID uuid.UUID, callType string, targetUserIDs []uuid.UUID, channelID *uuid.UUID) (*CallSession, string, error) {
	if !s.enabled {
		return nil, "", ErrLiveKitNotConfigured
	}

	if !ValidCallTypes[callType] {
		return nil, "", ErrInvalidCallType
	}

	if len(targetUserIDs) == 0 {
		return nil, "", ErrNoTargetUsers
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("create call: %w", err)
	}

	// Determine max participants based on call type
	var maxParticipants uint32
	if callType == CallTypeOneToOne {
		maxParticipants = MaxOneToOneParticipants
	} else {
		maxParticipants = MaxGroupParticipants
	}

	// Generate unique room name
	roomName := fmt.Sprintf("call-%s", uuid.New().String()[:8])

	// Create LiveKit room
	if err := s.roomMgr.CreateRoom(ctx, roomName, maxParticipants); err != nil {
		return nil, "", fmt.Errorf("create livekit room: %w", err)
	}

	now := time.Now()
	session := &CallSession{
		ID:          uuid.New(),
		TenantID:    tenantID,
		CallType:    callType,
		Status:      CallStatusRinging,
		RoomName:    roomName,
		InitiatorID: initiatorID,
		ChannelID:   channelID,
		CreatedAt:   now,
	}

	if err := s.repo.CreateCallSession(ctx, session); err != nil {
		// Best effort: delete the LiveKit room if DB insert fails
		if delErr := s.roomMgr.DeleteRoom(ctx, roomName); delErr != nil {
			slog.Error("failed to cleanup livekit room after db error",
				"room_name", roomName,
				"error", delErr,
			)
		}
		return nil, "", fmt.Errorf("create call session: %w", err)
	}

	// Add initiator as first participant
	participant := &CallParticipant{
		CallID:   session.ID,
		TenantID: tenantID,
		UserID:   initiatorID,
		JoinedAt: now,
		HasVideo: true,
		HasAudio: true,
	}
	if err := s.repo.AddParticipant(ctx, participant); err != nil {
		return nil, "", fmt.Errorf("add initiator participant: %w", err)
	}

	// Generate join token for initiator
	token, err := s.roomMgr.GenerateToken(roomName, initiatorID.String(), "")
	if err != nil {
		return nil, "", fmt.Errorf("generate initiator token: %w", err)
	}

	slog.Info("call created",
		"call_id", session.ID,
		"call_type", callType,
		"room_name", roomName,
		"initiator_id", initiatorID,
	)

	return session, token, nil
}

// JoinCall allows a user to join an existing call.
// Returns a join token for the user.
func (s *Service) JoinCall(ctx context.Context, callID, userID uuid.UUID, displayName string) (string, error) {
	if !s.enabled {
		return "", ErrLiveKitNotConfigured
	}

	session, err := s.repo.GetCallSession(ctx, callID)
	if err != nil {
		return "", err
	}

	if session.Status == CallStatusEnded {
		return "", ErrCallEnded
	}

	// Check max participants
	count, err := s.repo.CountActiveParticipants(ctx, callID)
	if err != nil {
		return "", fmt.Errorf("count participants: %w", err)
	}

	maxParticipants := MaxGroupParticipants
	if session.CallType == CallTypeOneToOne {
		maxParticipants = MaxOneToOneParticipants
	}

	if count >= maxParticipants {
		return "", ErrMaxParticipants
	}

	// Add participant (inherit tenant from parent session)
	now := time.Now()
	participant := &CallParticipant{
		CallID:   callID,
		TenantID: session.TenantID,
		UserID:   userID,
		JoinedAt: now,
		HasVideo: true,
		HasAudio: true,
	}
	if err := s.repo.AddParticipant(ctx, participant); err != nil {
		return "", fmt.Errorf("add participant: %w", err)
	}

	// Transition ringing -> active when first non-initiator joins
	if session.Status == CallStatusRinging {
		session.Status = CallStatusActive
		if err := s.repo.UpdateCallSession(ctx, session); err != nil {
			return "", fmt.Errorf("update call status: %w", err)
		}
	}

	// Generate join token
	token, err := s.roomMgr.GenerateToken(session.RoomName, userID.String(), displayName)
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	slog.Info("participant joined call",
		"call_id", callID,
		"user_id", userID,
	)

	return token, nil
}

// EndCall terminates a call session.
func (s *Service) EndCall(ctx context.Context, callID, userID uuid.UUID) error {
	session, err := s.repo.GetCallSession(ctx, callID)
	if err != nil {
		return err
	}

	if session.Status == CallStatusEnded {
		return nil // Already ended, idempotent
	}

	now := time.Now()
	durationSec := int(now.Sub(session.CreatedAt).Seconds())
	session.Status = CallStatusEnded
	session.EndedAt = &now
	session.DurationSeconds = &durationSec

	if err := s.repo.UpdateCallSession(ctx, session); err != nil {
		return fmt.Errorf("update call session: %w", err)
	}

	// Delete LiveKit room if enabled
	if s.enabled {
		if err := s.roomMgr.DeleteRoom(ctx, session.RoomName); err != nil {
			slog.Error("failed to delete livekit room",
				"room_name", session.RoomName,
				"error", err,
			)
			// Non-fatal: room will expire on its own
		}
	}

	slog.Info("call ended",
		"call_id", callID,
		"ended_by", userID,
		"duration_seconds", durationSec,
	)

	return nil
}

// GetCall retrieves a call session by ID.
func (s *Service) GetCall(ctx context.Context, callID uuid.UUID) (*CallSessionWithParticipants, error) {
	session, err := s.repo.GetCallSession(ctx, callID)
	if err != nil {
		return nil, err
	}

	participants, err := s.repo.GetParticipants(ctx, callID)
	if err != nil {
		return nil, fmt.Errorf("get participants: %w", err)
	}

	// Convert to CallParticipantWithUser (names are populated at gateway/gRPC level)
	var participantsWithUser []CallParticipantWithUser
	for _, p := range participants {
		participantsWithUser = append(participantsWithUser, CallParticipantWithUser{
			CallParticipant: p,
		})
	}

	return &CallSessionWithParticipants{
		CallSession:  *session,
		Participants: participantsWithUser,
	}, nil
}

// ListActiveCalls returns all active calls for a user.
func (s *Service) ListActiveCalls(ctx context.Context, userID uuid.UUID) ([]CallSession, error) {
	return s.repo.ListActiveCallsForUser(ctx, userID)
}

// HandleParticipantJoined processes a LiveKit webhook for participant joined event.
func (s *Service) HandleParticipantJoined(ctx context.Context, roomName, userIdentity string) error {
	session, err := s.repo.GetCallSessionByRoomName(ctx, roomName)
	if err != nil {
		return err
	}

	userID, err := uuid.Parse(userIdentity)
	if err != nil {
		return fmt.Errorf("parse user identity: %w", err)
	}

	now := time.Now()
	participant := &CallParticipant{
		CallID:   session.ID,
		TenantID: session.TenantID,
		UserID:   userID,
		JoinedAt: now,
		HasVideo: true,
		HasAudio: true,
	}

	if err := s.repo.AddParticipant(ctx, participant); err != nil {
		return fmt.Errorf("add participant via webhook: %w", err)
	}

	// Transition ringing -> active
	if session.Status == CallStatusRinging {
		session.Status = CallStatusActive
		if err := s.repo.UpdateCallSession(ctx, session); err != nil {
			return fmt.Errorf("update call status: %w", err)
		}
	}

	slog.Info("webhook: participant joined",
		"room_name", roomName,
		"user_id", userID,
	)

	return nil
}

// HandleParticipantLeft processes a LiveKit webhook for participant left event.
// If no active participants remain, the call is automatically ended.
func (s *Service) HandleParticipantLeft(ctx context.Context, roomName, userIdentity string) error {
	session, err := s.repo.GetCallSessionByRoomName(ctx, roomName)
	if err != nil {
		return err
	}

	userID, err := uuid.Parse(userIdentity)
	if err != nil {
		return fmt.Errorf("parse user identity: %w", err)
	}

	if err := s.repo.RemoveParticipant(ctx, session.ID, userID); err != nil {
		return fmt.Errorf("remove participant: %w", err)
	}

	// Check if call should auto-end
	count, err := s.repo.CountActiveParticipants(ctx, session.ID)
	if err != nil {
		return fmt.Errorf("count active participants: %w", err)
	}

	if count == 0 && session.Status != CallStatusEnded {
		now := time.Now()
		durationSec := int(now.Sub(session.CreatedAt).Seconds())
		session.Status = CallStatusEnded
		session.EndedAt = &now
		session.DurationSeconds = &durationSec

		if err := s.repo.UpdateCallSession(ctx, session); err != nil {
			return fmt.Errorf("auto-end call: %w", err)
		}

		slog.Info("call auto-ended (no participants)",
			"call_id", session.ID,
			"room_name", roomName,
		)
	}

	return nil
}

// HandleRoomFinished processes a LiveKit webhook for room finished event.
// Ends the call session if it is still active.
func (s *Service) HandleRoomFinished(ctx context.Context, roomName string) error {
	session, err := s.repo.GetCallSessionByRoomName(ctx, roomName)
	if err != nil {
		return err
	}

	if session.Status == CallStatusEnded {
		return nil // Already ended, idempotent
	}

	now := time.Now()
	durationSec := int(now.Sub(session.CreatedAt).Seconds())
	session.Status = CallStatusEnded
	session.EndedAt = &now
	session.DurationSeconds = &durationSec

	if err := s.repo.UpdateCallSession(ctx, session); err != nil {
		return fmt.Errorf("end call on room finish: %w", err)
	}

	slog.Info("call ended via room finished webhook",
		"call_id", session.ID,
		"room_name", roomName,
	)

	return nil
}
