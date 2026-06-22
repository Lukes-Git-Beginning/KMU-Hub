package video

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/middleware"
)

// tenantCtx returns a context with the given tenant_id set, as the middleware
// Auth handler would do in production.
func tenantCtx(tenantID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), middleware.TenantIDKey, tenantID.String())
}

// --- Mock RoomManager ---

type mockRoomManager struct {
	rooms        map[string]uint32
	createErr    error
	deleteErr    error
	tokenErr     error
	lastToken    string
	deleteCalled bool
	iceServers   []IceServerConfig
	participants []string
	listPartErr  error
}

func newMockRoomManager() *mockRoomManager {
	return &mockRoomManager{
		rooms:     make(map[string]uint32),
		lastToken: "mock-token-123",
	}
}

func (m *mockRoomManager) CreateRoom(_ context.Context, name string, maxParticipants uint32) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.rooms[name] = maxParticipants
	return nil
}

func (m *mockRoomManager) DeleteRoom(_ context.Context, name string) error {
	m.deleteCalled = true
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.rooms, name)
	return nil
}

func (m *mockRoomManager) GenerateToken(_, _, _ string) (string, error) {
	if m.tokenErr != nil {
		return "", m.tokenErr
	}
	return m.lastToken, nil
}

func (m *mockRoomManager) TURNIceServers(_ string) []IceServerConfig {
	return m.iceServers
}

func (m *mockRoomManager) ListParticipants(_ context.Context, _ string) ([]string, error) {
	if m.listPartErr != nil {
		return nil, m.listPartErr
	}
	return m.participants, nil
}

// --- Mock Repository ---

type mockRepo struct {
	sessions     map[uuid.UUID]*CallSession
	byRoom       map[string]*CallSession
	participants map[uuid.UUID][]CallParticipant
	createErr    error
	updateErr    error
	addPartErr   error
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		sessions:     make(map[uuid.UUID]*CallSession),
		byRoom:       make(map[string]*CallSession),
		participants: make(map[uuid.UUID][]CallParticipant),
	}
}

func (m *mockRepo) CreateCallSession(_ context.Context, session *CallSession) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.sessions[session.ID] = session
	m.byRoom[session.RoomName] = session
	return nil
}

func (m *mockRepo) GetCallSession(_ context.Context, id uuid.UUID) (*CallSession, error) {
	s, ok := m.sessions[id]
	if !ok {
		return nil, ErrNotFound
	}
	return s, nil
}

func (m *mockRepo) GetCallSessionByRoomName(_ context.Context, roomName string) (*CallSession, error) {
	s, ok := m.byRoom[roomName]
	if !ok {
		return nil, ErrNotFound
	}
	return s, nil
}

func (m *mockRepo) UpdateCallSession(_ context.Context, session *CallSession) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.sessions[session.ID] = session
	m.byRoom[session.RoomName] = session
	return nil
}

func (m *mockRepo) ListActiveCallsForUser(_ context.Context, userID uuid.UUID) ([]CallSession, error) {
	var result []CallSession
	for _, s := range m.sessions {
		if s.Status == CallStatusEnded {
			continue
		}
		for _, p := range m.participants[s.ID] {
			if p.UserID == userID && p.LeftAt == nil {
				result = append(result, *s)
				break
			}
		}
	}
	return result, nil
}

func (m *mockRepo) AddParticipant(_ context.Context, p *CallParticipant) error {
	if m.addPartErr != nil {
		return m.addPartErr
	}
	// Check if participant already exists and rejoin
	parts := m.participants[p.CallID]
	for i, existing := range parts {
		if existing.UserID == p.UserID {
			parts[i].LeftAt = nil
			parts[i].JoinedAt = p.JoinedAt
			m.participants[p.CallID] = parts
			return nil
		}
	}
	m.participants[p.CallID] = append(m.participants[p.CallID], *p)
	return nil
}

func (m *mockRepo) RemoveParticipant(_ context.Context, callID, userID uuid.UUID) error {
	parts := m.participants[callID]
	now := time.Now()
	for i, p := range parts {
		if p.UserID == userID && p.LeftAt == nil {
			parts[i].LeftAt = &now
			break
		}
	}
	m.participants[callID] = parts
	return nil
}

func (m *mockRepo) GetParticipants(_ context.Context, callID uuid.UUID) ([]CallParticipant, error) {
	return m.participants[callID], nil
}

func (m *mockRepo) CountActiveParticipants(_ context.Context, callID uuid.UUID) (int, error) {
	count := 0
	for _, p := range m.participants[callID] {
		if p.LeftAt == nil {
			count++
		}
	}
	return count, nil
}

// --- Tests ---

func TestCreateCall_Success(t *testing.T) {
	repo := newMockRepo()
	rm := newMockRoomManager()
	svc := NewService(repo, rm)

	initiator := uuid.New()
	target := uuid.New()

	session, token, err := svc.CreateCall(tenantCtx(uuid.Nil), initiator, CallTypeOneToOne, []uuid.UUID{target}, nil)

	require.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, "mock-token-123", token)
	assert.Equal(t, CallStatusRinging, session.Status)
	assert.Equal(t, CallTypeOneToOne, session.CallType)
	assert.Equal(t, initiator, session.InitiatorID)
	assert.Nil(t, session.ChannelID)
	assert.Contains(t, session.RoomName, "call-")

	// Verify room was created in LiveKit
	assert.Len(t, rm.rooms, 1)

	// Verify initiator was added as participant
	parts := repo.participants[session.ID]
	require.Len(t, parts, 1)
	assert.Equal(t, initiator, parts[0].UserID)
}

func TestCreateCall_WithChannelID(t *testing.T) {
	repo := newMockRepo()
	rm := newMockRoomManager()
	svc := NewService(repo, rm)

	initiator := uuid.New()
	target := uuid.New()
	channelID := uuid.New()

	session, _, err := svc.CreateCall(tenantCtx(uuid.Nil), initiator, CallTypeGroup, []uuid.UUID{target}, &channelID)

	require.NoError(t, err)
	assert.NotNil(t, session.ChannelID)
	assert.Equal(t, channelID, *session.ChannelID)
}

func TestCreateCall_FailsWhenLiveKitDisabled(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil) // nil roomMgr = disabled

	initiator := uuid.New()
	target := uuid.New()

	session, token, err := svc.CreateCall(tenantCtx(uuid.Nil), initiator, CallTypeOneToOne, []uuid.UUID{target}, nil)

	assert.ErrorIs(t, err, ErrLiveKitNotConfigured)
	assert.Nil(t, session)
	assert.Empty(t, token)
}

func TestCreateCall_FailsWithInvalidCallType(t *testing.T) {
	repo := newMockRepo()
	rm := newMockRoomManager()
	svc := NewService(repo, rm)

	_, _, err := svc.CreateCall(tenantCtx(uuid.Nil), uuid.New(), "invalid", []uuid.UUID{uuid.New()}, nil)
	assert.ErrorIs(t, err, ErrInvalidCallType)
}

func TestCreateCall_FailsWithNoTargets(t *testing.T) {
	repo := newMockRepo()
	rm := newMockRoomManager()
	svc := NewService(repo, rm)

	_, _, err := svc.CreateCall(tenantCtx(uuid.Nil), uuid.New(), CallTypeOneToOne, nil, nil)
	assert.ErrorIs(t, err, ErrNoTargetUsers)
}

func TestCreateCall_GroupCallMaxParticipants(t *testing.T) {
	repo := newMockRepo()
	rm := newMockRoomManager()
	svc := NewService(repo, rm)

	initiator := uuid.New()
	targets := []uuid.UUID{uuid.New(), uuid.New()}

	session, _, err := svc.CreateCall(tenantCtx(uuid.Nil), initiator, CallTypeGroup, targets, nil)
	require.NoError(t, err)

	// Verify the room was created with group max participants
	assert.Equal(t, uint32(MaxGroupParticipants), rm.rooms[session.RoomName])
}

func TestJoinCall_Success(t *testing.T) {
	repo := newMockRepo()
	rm := newMockRoomManager()
	svc := NewService(repo, rm)

	// Create a call first
	initiator := uuid.New()
	session, _, err := svc.CreateCall(tenantCtx(uuid.Nil), initiator, CallTypeGroup, []uuid.UUID{uuid.New()}, nil)
	require.NoError(t, err)

	// Join the call
	joiner := uuid.New()
	token, err := svc.JoinCall(tenantCtx(uuid.Nil), session.ID, joiner, "Max Mustermann")

	require.NoError(t, err)
	assert.Equal(t, "mock-token-123", token)

	// Verify status transitioned to active
	updated := repo.sessions[session.ID]
	assert.Equal(t, CallStatusActive, updated.Status)

	// Verify participant was added
	parts := repo.participants[session.ID]
	assert.Len(t, parts, 2) // initiator + joiner
}

func TestJoinCall_RejectsEndedCall(t *testing.T) {
	repo := newMockRepo()
	rm := newMockRoomManager()
	svc := NewService(repo, rm)

	// Create and end a call
	initiator := uuid.New()
	session, _, err := svc.CreateCall(tenantCtx(uuid.Nil), initiator, CallTypeOneToOne, []uuid.UUID{uuid.New()}, nil)
	require.NoError(t, err)

	err = svc.EndCall(tenantCtx(uuid.Nil), session.ID, initiator)
	require.NoError(t, err)

	// Try to join ended call
	_, err = svc.JoinCall(tenantCtx(uuid.Nil), session.ID, uuid.New(), "Late Joiner")
	assert.ErrorIs(t, err, ErrCallEnded)
}

func TestJoinCall_RejectsAtMaxParticipants(t *testing.T) {
	repo := newMockRepo()
	rm := newMockRoomManager()
	svc := NewService(repo, rm)

	// Create a 1:1 call
	initiator := uuid.New()
	session, _, err := svc.CreateCall(tenantCtx(uuid.Nil), initiator, CallTypeOneToOne, []uuid.UUID{uuid.New()}, nil)
	require.NoError(t, err)

	// Join with second participant (fills 1:1 capacity)
	joiner := uuid.New()
	_, err = svc.JoinCall(tenantCtx(uuid.Nil), session.ID, joiner, "Joiner")
	require.NoError(t, err)

	// Third participant should be rejected
	_, err = svc.JoinCall(tenantCtx(uuid.Nil), session.ID, uuid.New(), "Third")
	assert.ErrorIs(t, err, ErrMaxParticipants)
}

func TestJoinCall_FailsWhenLiveKitDisabled(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil)

	_, err := svc.JoinCall(tenantCtx(uuid.Nil), uuid.New(), uuid.New(), "User")
	assert.ErrorIs(t, err, ErrLiveKitNotConfigured)
}

func TestEndCall_Success(t *testing.T) {
	repo := newMockRepo()
	rm := newMockRoomManager()
	svc := NewService(repo, rm)

	initiator := uuid.New()
	session, _, err := svc.CreateCall(tenantCtx(uuid.Nil), initiator, CallTypeOneToOne, []uuid.UUID{uuid.New()}, nil)
	require.NoError(t, err)

	err = svc.EndCall(tenantCtx(uuid.Nil), session.ID, initiator)
	require.NoError(t, err)

	updated := repo.sessions[session.ID]
	assert.Equal(t, CallStatusEnded, updated.Status)
	assert.NotNil(t, updated.EndedAt)
	assert.NotNil(t, updated.DurationSeconds)
	assert.True(t, rm.deleteCalled)
}

func TestEndCall_Idempotent(t *testing.T) {
	repo := newMockRepo()
	rm := newMockRoomManager()
	svc := NewService(repo, rm)

	initiator := uuid.New()
	session, _, err := svc.CreateCall(tenantCtx(uuid.Nil), initiator, CallTypeOneToOne, []uuid.UUID{uuid.New()}, nil)
	require.NoError(t, err)

	// End twice - should not error
	err = svc.EndCall(tenantCtx(uuid.Nil), session.ID, initiator)
	require.NoError(t, err)

	err = svc.EndCall(tenantCtx(uuid.Nil), session.ID, initiator)
	require.NoError(t, err)
}

func TestEndCall_NotFound(t *testing.T) {
	repo := newMockRepo()
	rm := newMockRoomManager()
	svc := NewService(repo, rm)

	err := svc.EndCall(tenantCtx(uuid.Nil), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetCall_Success(t *testing.T) {
	repo := newMockRepo()
	rm := newMockRoomManager()
	svc := NewService(repo, rm)

	initiator := uuid.New()
	session, _, err := svc.CreateCall(tenantCtx(uuid.Nil), initiator, CallTypeGroup, []uuid.UUID{uuid.New()}, nil)
	require.NoError(t, err)

	result, err := svc.GetCall(tenantCtx(uuid.Nil), session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.ID, result.ID)
	assert.Len(t, result.Participants, 1) // initiator
}

func TestListActiveCalls(t *testing.T) {
	repo := newMockRepo()
	rm := newMockRoomManager()
	svc := NewService(repo, rm)

	initiator := uuid.New()

	// Create two calls
	session1, _, err := svc.CreateCall(tenantCtx(uuid.Nil), initiator, CallTypeGroup, []uuid.UUID{uuid.New()}, nil)
	require.NoError(t, err)

	_, _, err = svc.CreateCall(tenantCtx(uuid.Nil), initiator, CallTypeOneToOne, []uuid.UUID{uuid.New()}, nil)
	require.NoError(t, err)

	// End first call
	err = svc.EndCall(tenantCtx(uuid.Nil), session1.ID, initiator)
	require.NoError(t, err)

	calls, err := svc.ListActiveCalls(tenantCtx(uuid.Nil), initiator)
	require.NoError(t, err)
	assert.Len(t, calls, 1) // Only the second call is still active
}

func TestHandleParticipantLeft_AutoEndsCall(t *testing.T) {
	repo := newMockRepo()
	rm := newMockRoomManager()
	svc := NewService(repo, rm)

	initiator := uuid.New()
	session, _, err := svc.CreateCall(tenantCtx(uuid.Nil), initiator, CallTypeOneToOne, []uuid.UUID{uuid.New()}, nil)
	require.NoError(t, err)

	// Participant leaves
	err = svc.HandleParticipantLeft(tenantCtx(uuid.Nil), session.RoomName, initiator.String())
	require.NoError(t, err)

	// Call should be auto-ended
	updated := repo.sessions[session.ID]
	assert.Equal(t, CallStatusEnded, updated.Status)
	assert.NotNil(t, updated.EndedAt)
}

func TestHandleParticipantLeft_DoesNotEndWithRemainingParticipants(t *testing.T) {
	repo := newMockRepo()
	rm := newMockRoomManager()
	svc := NewService(repo, rm)

	initiator := uuid.New()
	session, _, err := svc.CreateCall(tenantCtx(uuid.Nil), initiator, CallTypeGroup, []uuid.UUID{uuid.New()}, nil)
	require.NoError(t, err)

	// Add a second participant
	joiner := uuid.New()
	_, err = svc.JoinCall(tenantCtx(uuid.Nil), session.ID, joiner, "Joiner")
	require.NoError(t, err)

	// First participant leaves
	err = svc.HandleParticipantLeft(tenantCtx(uuid.Nil), session.RoomName, initiator.String())
	require.NoError(t, err)

	// Call should still be active (joiner remains)
	updated := repo.sessions[session.ID]
	assert.Equal(t, CallStatusActive, updated.Status)
}

func TestHandleRoomFinished_EndsCall(t *testing.T) {
	repo := newMockRepo()
	rm := newMockRoomManager()
	svc := NewService(repo, rm)

	initiator := uuid.New()
	session, _, err := svc.CreateCall(tenantCtx(uuid.Nil), initiator, CallTypeOneToOne, []uuid.UUID{uuid.New()}, nil)
	require.NoError(t, err)

	err = svc.HandleRoomFinished(tenantCtx(uuid.Nil), session.RoomName)
	require.NoError(t, err)

	updated := repo.sessions[session.ID]
	assert.Equal(t, CallStatusEnded, updated.Status)
	assert.NotNil(t, updated.EndedAt)
	assert.NotNil(t, updated.DurationSeconds)
}

func TestHandleRoomFinished_IdempotentForEndedCall(t *testing.T) {
	repo := newMockRepo()
	rm := newMockRoomManager()
	svc := NewService(repo, rm)

	initiator := uuid.New()
	session, _, err := svc.CreateCall(tenantCtx(uuid.Nil), initiator, CallTypeOneToOne, []uuid.UUID{uuid.New()}, nil)
	require.NoError(t, err)

	// End call first
	err = svc.EndCall(tenantCtx(uuid.Nil), session.ID, initiator)
	require.NoError(t, err)

	// Room finished webhook should be idempotent
	err = svc.HandleRoomFinished(tenantCtx(uuid.Nil), session.RoomName)
	require.NoError(t, err)
}

func TestHandleParticipantJoined(t *testing.T) {
	repo := newMockRepo()
	rm := newMockRoomManager()
	svc := NewService(repo, rm)

	initiator := uuid.New()
	session, _, err := svc.CreateCall(tenantCtx(uuid.Nil), initiator, CallTypeGroup, []uuid.UUID{uuid.New()}, nil)
	require.NoError(t, err)

	// Webhook: new participant joins
	joiner := uuid.New()
	err = svc.HandleParticipantJoined(tenantCtx(uuid.Nil), session.RoomName, joiner.String())
	require.NoError(t, err)

	// Verify participant was added
	parts := repo.participants[session.ID]
	assert.Len(t, parts, 2)

	// Verify status transitioned
	updated := repo.sessions[session.ID]
	assert.Equal(t, CallStatusActive, updated.Status)
}

func TestNewService_DisabledMode(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil)

	assert.False(t, svc.enabled)

	// EndCall should still work (it does not require LiveKit for DB updates)
	// But CreateCall should fail
	_, _, err := svc.CreateCall(tenantCtx(uuid.Nil), uuid.New(), CallTypeOneToOne, []uuid.UUID{uuid.New()}, nil)
	assert.ErrorIs(t, err, ErrLiveKitNotConfigured)
}
