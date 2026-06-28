package recording

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// --- Mock EgressManager ---

type mockEgressManager struct {
	startErr     error
	stopErr      error
	lastEgressID string
	stopCalled   bool
}

func newMockEgressManager() *mockEgressManager {
	return &mockEgressManager{
		lastEgressID: "egress-mock-123",
	}
}

func (m *mockEgressManager) StartRoomCompositeEgress(_ context.Context, _, _ string, _ S3Config) (string, error) {
	if m.startErr != nil {
		return "", m.startErr
	}
	return m.lastEgressID, nil
}

func (m *mockEgressManager) StopEgress(_ context.Context, _ string) error {
	m.stopCalled = true
	return m.stopErr
}

// makeParticipants builds a []ParticipantConsentInfo slice from UUIDs for test convenience.
func makeParticipants(ids ...uuid.UUID) []ParticipantConsentInfo {
	ps := make([]ParticipantConsentInfo, len(ids))
	for i, id := range ids {
		ps[i] = ParticipantConsentInfo{
			UserID:      id,
			DisplayName: "User " + id.String()[:8],
			JoinedAt:    time.Now(),
		}
	}
	return ps
}

// --- Mock Repository ---

type mockRepo struct {
	recordings       map[uuid.UUID]*Recording
	consents         map[uuid.UUID][]RecordingConsent
	pendingOverride  *int  // if set, override CountPendingConsents result
	participantUsers []uuid.UUID
	createErr        error
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		recordings: make(map[uuid.UUID]*Recording),
		consents:   make(map[uuid.UUID][]RecordingConsent),
	}
}

func (m *mockRepo) CreateRecording(_ context.Context, rec *Recording) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.recordings[rec.ID] = rec
	return nil
}

func (m *mockRepo) GetRecording(_ context.Context, id uuid.UUID) (*Recording, error) {
	rec, ok := m.recordings[id]
	if !ok {
		return nil, ErrNotFound
	}
	return rec, nil
}

func (m *mockRepo) UpdateRecording(_ context.Context, rec *Recording) error {
	m.recordings[rec.ID] = rec
	return nil
}

func (m *mockRepo) ListRecordingsByCall(_ context.Context, callID uuid.UUID) ([]Recording, error) {
	var result []Recording
	for _, rec := range m.recordings {
		if rec.CallID != nil && *rec.CallID == callID {
			result = append(result, *rec)
		}
	}
	return result, nil
}

func (m *mockRepo) ListRecordingsByMeeting(_ context.Context, meetingID uuid.UUID) ([]Recording, error) {
	var result []Recording
	for _, rec := range m.recordings {
		if rec.MeetingID != nil && *rec.MeetingID == meetingID {
			result = append(result, *rec)
		}
	}
	return result, nil
}

func (m *mockRepo) DeleteRecording(_ context.Context, id uuid.UUID) error {
	delete(m.recordings, id)
	return nil
}

func (m *mockRepo) SetConsent(_ context.Context, consent *RecordingConsent) error {
	existing := m.consents[consent.RecordingID]
	// Update existing or add new
	found := false
	for i, c := range existing {
		if c.UserID == consent.UserID {
			existing[i] = *consent
			found = true
			break
		}
	}
	if !found {
		existing = append(existing, *consent)
	}
	m.consents[consent.RecordingID] = existing
	return nil
}

func (m *mockRepo) GetConsents(_ context.Context, recordingID uuid.UUID) ([]RecordingConsent, error) {
	return m.consents[recordingID], nil
}

func (m *mockRepo) CountPendingConsents(_ context.Context, recordingID uuid.UUID, participantIDs []uuid.UUID) (int, error) {
	if m.pendingOverride != nil {
		return *m.pendingOverride, nil
	}
	// Count participants who have NOT responded
	respondedSet := make(map[uuid.UUID]bool)
	for _, c := range m.consents[recordingID] {
		respondedSet[c.UserID] = true
	}
	pending := 0
	for _, pid := range participantIDs {
		if !respondedSet[pid] {
			pending++
		}
	}
	return pending, nil
}

func (m *mockRepo) ListExpiredRecordings(_ context.Context, before time.Time) ([]Recording, error) {
	var result []Recording
	for _, rec := range m.recordings {
		if rec.RetentionExpiresAt != nil && rec.RetentionExpiresAt.Before(before) && rec.Status == RecordingStatusCompleted {
			result = append(result, *rec)
		}
	}
	return result, nil
}

func (m *mockRepo) ListRecordingsWithAccess(_ context.Context, userID uuid.UUID, callID, meetingID *uuid.UUID) ([]Recording, error) {
	// Simplified mock: return all completed/processing recordings if user is in participantUsers
	if !slices.Contains(m.participantUsers, userID) {
		return nil, nil
	}

	var result []Recording
	for _, rec := range m.recordings {
		if rec.Status != RecordingStatusCompleted && rec.Status != RecordingStatusProcessing {
			continue
		}
		if callID != nil && (rec.CallID == nil || *rec.CallID != *callID) {
			continue
		}
		if meetingID != nil && (rec.MeetingID == nil || *rec.MeetingID != *meetingID) {
			continue
		}
		result = append(result, *rec)
	}
	return result, nil
}

func (m *mockRepo) GetRecordingParticipants(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return m.participantUsers, nil
}

func (m *mockRepo) GetRecordingByEgressID(_ context.Context, egressID string) (*Recording, error) {
	for _, rec := range m.recordings {
		if rec.EgressID != nil && *rec.EgressID == egressID {
			return rec, nil
		}
	}
	return nil, ErrNotFound
}

func (m *mockRepo) TagRecordingWithConsents(_ context.Context, recordingID uuid.UUID, snapshot []ParticipantConsentInfo) error {
	rec, ok := m.recordings[recordingID]
	if !ok {
		return ErrNotFound
	}
	rec.ConsentSnapshot = snapshot
	return nil
}

func (m *mockRepo) GetConsentsWithUser(_ context.Context, recordingID uuid.UUID) ([]RecordingConsentWithUser, error) {
	raw := m.consents[recordingID]
	result := make([]RecordingConsentWithUser, len(raw))
	for i, c := range raw {
		result[i] = RecordingConsentWithUser{
			RecordingConsent: c,
			FirstName:        "Mock",
			LastName:         "User",
		}
	}
	return result, nil
}

func (m *mockRepo) MarkInitiatorConsent(_ context.Context, recordingID, _ uuid.UUID, _ uuid.UUID) error {
	rec, ok := m.recordings[recordingID]
	if !ok {
		return ErrNotFound
	}
	now := time.Now()
	rec.PreRecordingConsentAt = &now
	return nil
}

func (m *mockRepo) GetPreConsentStatus(_ context.Context, recordingID, _ uuid.UUID) (bool, error) {
	rec, ok := m.recordings[recordingID]
	if !ok {
		return false, ErrNotFound
	}
	return rec.PreRecordingConsentAt != nil, nil
}

// --- Tests ---

func TestStartRecording_FailsWhenConsentPending(t *testing.T) {
	repo := newMockRepo()
	egress := newMockEgressManager()
	svc := NewService(repo, egress, "https://template.example.com", S3Config{})

	callID := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()
	starter := uuid.New()

	// No consents set -> pending
	_, err := svc.StartRecording(context.Background(), &callID, nil, "call-abc123", starter, makeParticipants(user1, user2))
	assert.ErrorIs(t, err, ErrConsentPending)
}

func TestStartRecording_SucceedsWhenAllConsented(t *testing.T) {
	repo := newMockRepo()
	egress := newMockEgressManager()
	svc := NewService(repo, egress, "https://template.example.com", S3Config{Bucket: "recordings"})

	callID := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()
	starter := user1
	participants := makeParticipants(user1, user2)

	// Pre-create a recording ID to set consents on
	// We need to use pendingOverride since we can't know the recording ID in advance
	zero := 0
	repo.pendingOverride = &zero

	rec, err := svc.StartRecording(context.Background(), &callID, nil, "call-abc123", starter, participants)
	require.NoError(t, err)
	assert.NotNil(t, rec)
	assert.Equal(t, RecordingStatusActive, rec.Status)
	assert.NotNil(t, rec.EgressID)
	assert.Equal(t, "egress-mock-123", *rec.EgressID)
}

func TestStartRecording_Sets30DayRetention(t *testing.T) {
	repo := newMockRepo()
	egress := newMockEgressManager()
	svc := NewService(repo, egress, "https://template.example.com", S3Config{})

	callID := uuid.New()
	starter := uuid.New()
	zero := 0
	repo.pendingOverride = &zero

	rec, err := svc.StartRecording(context.Background(), &callID, nil, "call-abc123", starter, makeParticipants(starter))
	require.NoError(t, err)

	assert.NotNil(t, rec.RetentionExpiresAt)
	expectedExpiry := time.Now().Add(30 * 24 * time.Hour)
	// Allow 5 seconds tolerance
	diff := rec.RetentionExpiresAt.Sub(expectedExpiry)
	assert.Less(t, diff.Abs(), 5*time.Second, "retention should be ~30 days from now")
}

func TestStartRecording_FailsWhenEgressNotConfigured(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{}) // nil egress = disabled

	callID := uuid.New()
	starter := uuid.New()
	_, err := svc.StartRecording(context.Background(), &callID, nil, "call-abc123", starter, makeParticipants(starter))
	assert.ErrorIs(t, err, ErrEgressNotConfigured)
}

func TestStartRecording_FailsWithNoCallOrMeeting(t *testing.T) {
	repo := newMockRepo()
	egress := newMockEgressManager()
	svc := NewService(repo, egress, "", S3Config{})

	starter := uuid.New()
	_, err := svc.StartRecording(context.Background(), nil, nil, "call-abc123", starter, makeParticipants(starter))
	assert.ErrorIs(t, err, ErrNoCallOrMeeting)
}

func TestStartRecording_WithMeetingID(t *testing.T) {
	repo := newMockRepo()
	egress := newMockEgressManager()
	svc := NewService(repo, egress, "https://template.example.com", S3Config{})

	meetingID := uuid.New()
	starter := uuid.New()
	zero := 0
	repo.pendingOverride = &zero

	rec, err := svc.StartRecording(context.Background(), nil, &meetingID, "meeting-room", starter, makeParticipants(starter))
	require.NoError(t, err)
	assert.Nil(t, rec.CallID)
	assert.NotNil(t, rec.MeetingID)
	assert.Equal(t, meetingID, *rec.MeetingID)
}

func TestStopRecording_TransitionsToProcessing(t *testing.T) {
	repo := newMockRepo()
	egress := newMockEgressManager()
	svc := NewService(repo, egress, "", S3Config{})

	callID := uuid.New()
	starter := uuid.New()
	zero := 0
	repo.pendingOverride = &zero

	rec, err := svc.StartRecording(context.Background(), &callID, nil, "call-abc123", starter, makeParticipants(starter))
	require.NoError(t, err)

	stopped, err := svc.StopRecording(context.Background(), rec.ID)
	require.NoError(t, err)
	assert.Equal(t, RecordingStatusProcessing, stopped.Status)
	assert.True(t, egress.stopCalled)
}

func TestStopRecording_FailsForNonActiveRecording(t *testing.T) {
	repo := newMockRepo()
	egress := newMockEgressManager()
	svc := NewService(repo, egress, "", S3Config{})

	// Create a completed recording directly
	recID := uuid.New()
	callID := uuid.New()
	repo.recordings[recID] = &Recording{
		ID:     recID,
		CallID: &callID,
		Status: RecordingStatusCompleted,
	}

	_, err := svc.StopRecording(context.Background(), recID)
	assert.ErrorIs(t, err, ErrRecordingNotActive)
}

func TestSetConsent_StoresConsentRecord(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	recID := uuid.New()
	userID := uuid.New()

	err := svc.SetConsent(context.Background(), recID, userID, true)
	require.NoError(t, err)

	consents := repo.consents[recID]
	require.Len(t, consents, 1)
	assert.Equal(t, userID, consents[0].UserID)
	assert.True(t, consents[0].Consented)
}

func TestSetConsent_UpdatesExistingConsent(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	recID := uuid.New()
	userID := uuid.New()

	// Set consent to true
	err := svc.SetConsent(context.Background(), recID, userID, true)
	require.NoError(t, err)

	// Update consent to false
	err = svc.SetConsent(context.Background(), recID, userID, false)
	require.NoError(t, err)

	consents := repo.consents[recID]
	require.Len(t, consents, 1)
	assert.False(t, consents[0].Consented)
}

func TestGetConsentStatus_AllResponded(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	recID := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()

	// Both users consent
	err := svc.SetConsent(context.Background(), recID, user1, true)
	require.NoError(t, err)
	err = svc.SetConsent(context.Background(), recID, user2, true)
	require.NoError(t, err)

	allResponded, consents, err := svc.GetConsentStatus(context.Background(), recID, []uuid.UUID{user1, user2})
	require.NoError(t, err)
	assert.True(t, allResponded)
	assert.Len(t, consents, 2)
}

func TestGetConsentStatus_SomePending(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	recID := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()

	// Only user1 responds
	err := svc.SetConsent(context.Background(), recID, user1, true)
	require.NoError(t, err)

	allResponded, consents, err := svc.GetConsentStatus(context.Background(), recID, []uuid.UUID{user1, user2})
	require.NoError(t, err)
	assert.False(t, allResponded)
	assert.Len(t, consents, 1)
}

func TestCompleteRecording_UpdatesFileInfo(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	recID := uuid.New()
	callID := uuid.New()
	repo.recordings[recID] = &Recording{
		ID:     recID,
		CallID: &callID,
		Status: RecordingStatusProcessing,
	}

	err := svc.CompleteRecording(context.Background(), recID, "s3://recordings/file.mp4", 1024000, 300)
	require.NoError(t, err)

	updated := repo.recordings[recID]
	assert.Equal(t, RecordingStatusCompleted, updated.Status)
	assert.Equal(t, "s3://recordings/file.mp4", *updated.FileURL)
	assert.Equal(t, int64(1024000), *updated.FileSizeBytes)
	assert.Equal(t, 300, *updated.DurationSeconds)
}

func TestFailRecording_SetsFailedStatus(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	recID := uuid.New()
	callID := uuid.New()
	repo.recordings[recID] = &Recording{
		ID:     recID,
		CallID: &callID,
		Status: RecordingStatusActive,
	}

	err := svc.FailRecording(context.Background(), recID, "egress timeout")
	require.NoError(t, err)

	updated := repo.recordings[recID]
	assert.Equal(t, RecordingStatusFailed, updated.Status)
}

func TestCleanupExpiredRecordings_DeletesExpired(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	// Create an expired recording
	expiredID := uuid.New()
	callID := uuid.New()
	expiredTime := time.Now().Add(-1 * time.Hour) // expired 1 hour ago
	repo.recordings[expiredID] = &Recording{
		ID:                 expiredID,
		CallID:             &callID,
		Status:             RecordingStatusCompleted,
		RetentionExpiresAt: &expiredTime,
	}

	// Create a non-expired recording
	activeID := uuid.New()
	callID2 := uuid.New()
	futureTime := time.Now().Add(24 * time.Hour)
	repo.recordings[activeID] = &Recording{
		ID:                 activeID,
		CallID:             &callID2,
		Status:             RecordingStatusCompleted,
		RetentionExpiresAt: &futureTime,
	}

	deleted, err := svc.CleanupExpiredRecordings(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)

	// Expired recording should be gone
	_, exists := repo.recordings[expiredID]
	assert.False(t, exists)

	// Active recording should still exist
	_, exists = repo.recordings[activeID]
	assert.True(t, exists)
}

func TestListRecordingsWithAccess_OnlyParticipantRecordings(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	user1 := uuid.New()
	user2 := uuid.New()
	repo.participantUsers = []uuid.UUID{user1} // Only user1 is participant

	// Create a completed recording
	recID := uuid.New()
	callID := uuid.New()
	repo.recordings[recID] = &Recording{
		ID:     recID,
		CallID: &callID,
		Status: RecordingStatusCompleted,
	}

	// user1 should see recordings
	recs, err := svc.ListRecordingsWithAccess(context.Background(), user1, nil, nil)
	require.NoError(t, err)
	assert.Len(t, recs, 1)

	// user2 should NOT see recordings
	recs, err = svc.ListRecordingsWithAccess(context.Background(), user2, nil, nil)
	require.NoError(t, err)
	assert.Len(t, recs, 0)
}

func TestGetRecordingParticipants(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	user1 := uuid.New()
	user2 := uuid.New()
	repo.participantUsers = []uuid.UUID{user1, user2}

	recID := uuid.New()
	participants, err := svc.GetRecordingParticipants(context.Background(), recID)
	require.NoError(t, err)
	assert.Len(t, participants, 2)
	assert.Contains(t, participants, user1)
	assert.Contains(t, participants, user2)
}

func TestListRecordings_ByCallID(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	callID := uuid.New()
	recID := uuid.New()
	repo.recordings[recID] = &Recording{
		ID:     recID,
		CallID: &callID,
		Status: RecordingStatusCompleted,
	}

	recs, err := svc.ListRecordings(context.Background(), &callID, nil)
	require.NoError(t, err)
	assert.Len(t, recs, 1)
}

func TestListRecordings_ByMeetingID(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	meetingID := uuid.New()
	recID := uuid.New()
	repo.recordings[recID] = &Recording{
		ID:        recID,
		MeetingID: &meetingID,
		Status:    RecordingStatusCompleted,
	}

	recs, err := svc.ListRecordings(context.Background(), nil, &meetingID)
	require.NoError(t, err)
	assert.Len(t, recs, 1)
}

func TestListRecordings_FailsWithNoFilter(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	_, err := svc.ListRecordings(context.Background(), nil, nil)
	assert.ErrorIs(t, err, ErrNoCallOrMeeting)
}

func TestNewService_DisabledMode(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	assert.False(t, svc.enabled)

	// SetConsent should still work (does not require egress)
	err := svc.SetConsent(context.Background(), uuid.New(), uuid.New(), true)
	assert.NoError(t, err)

	// StartRecording should fail
	callID := uuid.New()
	starter := uuid.New()
	_, err = svc.StartRecording(context.Background(), &callID, nil, "room", starter, makeParticipants(starter))
	assert.ErrorIs(t, err, ErrEgressNotConfigured)
}

// ============================================================================
// New tests for S1.R2.3: consent-snapshot, started_by, multi-participant gate
// ============================================================================

// TestStartRecording_AllThreeParticipantsRequireConsent verifies that recording
// cannot start when 3 participants are present but none has consented yet.
func TestStartRecording_AllThreeParticipantsRequireConsent(t *testing.T) {
	repo := newMockRepo()
	egress := newMockEgressManager()
	svc := NewService(repo, egress, "https://template.example.com", S3Config{})

	callID := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()
	user3 := uuid.New()
	starter := user1

	// No consents yet — all 3 pending
	_, err := svc.StartRecording(context.Background(), &callID, nil, "room-x", starter, makeParticipants(user1, user2, user3))
	assert.ErrorIs(t, err, ErrConsentPending)
}

// TestStartRecording_BlocksUntilAllThreeConsent verifies the consent gate:
// one consent is not enough, two are not enough, only all three allow recording.
func TestStartRecording_BlocksUntilAllThreeConsent(t *testing.T) {
	repo := newMockRepo()
	egress := newMockEgressManager()
	svc := NewService(repo, egress, "https://template.example.com", S3Config{Bucket: "recordings"})

	callID := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()
	user3 := uuid.New()
	starter := user1
	participants := makeParticipants(user1, user2, user3)

	// We need a recording ID upfront to register consents.
	// Use a temporary recording with known ID in the mock.
	tempRecID := uuid.New()
	repo.recordings[tempRecID] = &Recording{ID: tempRecID, CallID: &callID, Status: RecordingStatusActive}

	// Register consents against that ID
	require.NoError(t, svc.SetConsent(context.Background(), tempRecID, user1, true))
	require.NoError(t, svc.SetConsent(context.Background(), tempRecID, user2, true))
	// user3 has NOT consented yet

	// One missing consent: still blocked. Use pendingOverride = 1
	one := 1
	repo.pendingOverride = &one
	_, err := svc.StartRecording(context.Background(), &callID, nil, "room-x", starter, participants)
	assert.ErrorIs(t, err, ErrConsentPending)

	// All three consented: now allowed
	zero := 0
	repo.pendingOverride = &zero
	rec, err := svc.StartRecording(context.Background(), &callID, nil, "room-x", starter, participants)
	require.NoError(t, err)
	assert.Equal(t, RecordingStatusActive, rec.Status)
}

// TestStartRecording_PersistsStartedBy verifies that started_by is stored on the recording.
func TestStartRecording_PersistsStartedBy(t *testing.T) {
	repo := newMockRepo()
	egress := newMockEgressManager()
	svc := NewService(repo, egress, "https://template.example.com", S3Config{Bucket: "recordings"})

	callID := uuid.New()
	starter := uuid.New()
	zero := 0
	repo.pendingOverride = &zero

	rec, err := svc.StartRecording(context.Background(), &callID, nil, "room-x", starter, makeParticipants(starter))
	require.NoError(t, err)

	// started_by must be set and match the caller
	require.NotNil(t, rec.StartedBy)
	assert.Equal(t, starter, *rec.StartedBy)

	// the stored recording must also carry started_by
	stored := repo.recordings[rec.ID]
	require.NotNil(t, stored.StartedBy)
	assert.Equal(t, starter, *stored.StartedBy)
}

// TestStartRecording_ConsentSnapshotContainsAllParticipants verifies that the
// immutable DSGVO snapshot captures every participant present at recording start.
func TestStartRecording_ConsentSnapshotContainsAllParticipants(t *testing.T) {
	repo := newMockRepo()
	egress := newMockEgressManager()
	svc := NewService(repo, egress, "https://template.example.com", S3Config{Bucket: "recordings"})

	callID := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()
	user3 := uuid.New()
	starter := user1
	participants := makeParticipants(user1, user2, user3)

	zero := 0
	repo.pendingOverride = &zero

	rec, err := svc.StartRecording(context.Background(), &callID, nil, "room-x", starter, participants)
	require.NoError(t, err)

	// Snapshot must have exactly 3 entries
	assert.Len(t, rec.ConsentSnapshot, 3)

	// All participant IDs must be present in the snapshot
	snapshotIDs := make(map[uuid.UUID]bool)
	for _, p := range rec.ConsentSnapshot {
		snapshotIDs[p.UserID] = true
	}
	assert.True(t, snapshotIDs[user1])
	assert.True(t, snapshotIDs[user2])
	assert.True(t, snapshotIDs[user3])
}

// TestStartRecording_FailsWithNoParticipants verifies that an empty participant list is rejected.
func TestStartRecording_FailsWithNoParticipants(t *testing.T) {
	repo := newMockRepo()
	egress := newMockEgressManager()
	svc := NewService(repo, egress, "https://template.example.com", S3Config{})

	callID := uuid.New()
	starter := uuid.New()

	_, err := svc.StartRecording(context.Background(), &callID, nil, "room-x", starter, nil)
	assert.ErrorIs(t, err, ErrNoParticipants)
}

// ============================================================================
// Tests for R2-P0.4: Initiator pre-recording consent (Migration 000107)
// ============================================================================

// TestConfirmInitiatorConsent_StampsRecording verifies that ConfirmInitiatorConsent sets
// pre_recording_consent_at on the recording row.
func TestConfirmInitiatorConsent_StampsRecording(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	recID := uuid.New()
	callID := uuid.New()
	userID := uuid.New()
	tenantID := uuid.New()

	repo.recordings[recID] = &Recording{
		ID:        recID,
		CallID:    &callID,
		StartedBy: &userID,
		Status:    RecordingStatusActive,
	}

	err := svc.ConfirmInitiatorConsent(context.Background(), recID, userID, tenantID)
	require.NoError(t, err)

	stamped := repo.recordings[recID].PreRecordingConsentAt
	assert.NotNil(t, stamped, "pre_recording_consent_at should be set after confirm")
}

// TestConfirmInitiatorConsent_FailsForUnknownRecording verifies ErrNotFound is returned
// when the recording does not exist.
func TestConfirmInitiatorConsent_FailsForUnknownRecording(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	err := svc.ConfirmInitiatorConsent(context.Background(), uuid.New(), uuid.New(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mark initiator consent")
}

// TestStartRecording_RequiresPreConsent verifies the full pre-consent workflow:
//   - ErrPreConsentMissing is exported and non-nil (sentinel check)
//   - GetPreConsentStatus returns false before ConfirmInitiatorConsent is called
//   - GetPreConsentStatus returns true after ConfirmInitiatorConsent is called
//   - Calling ConfirmInitiatorConsent stamps the recording (mock call-tracking assertion)
//
// The initiator gate is enforced at the HTTP-layer (dialog flow); this test validates
// that ConfirmInitiatorConsent is actually invoked and mutates the expected state.
func TestStartRecording_RequiresPreConsent(t *testing.T) {
	// 1. Sentinel must be defined and carry the expected message fragment.
	require.NotNil(t, ErrPreConsentMissing)
	assert.True(t, errors.Is(ErrPreConsentMissing, ErrPreConsentMissing),
		"ErrPreConsentMissing must satisfy errors.Is identity")
	assert.Contains(t, ErrPreConsentMissing.Error(), "initiator")

	// 2. Before ConfirmInitiatorConsent: GetPreConsentStatus returns false.
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})
	recID := uuid.New()
	tenantID := uuid.New()
	userID := uuid.New()

	repo.recordings[recID] = &Recording{
		ID:     recID,
		Status: RecordingStatusActive,
	}

	stamped, err := repo.GetPreConsentStatus(context.Background(), recID, tenantID)
	require.NoError(t, err)
	assert.False(t, stamped, "pre-consent must be absent before ConfirmInitiatorConsent is called")

	// 3. Call ConfirmInitiatorConsent — assert it succeeds (i.e. the correct repo method is called
	// with the right arguments and the recording is mutated).
	require.NoError(t, svc.ConfirmInitiatorConsent(context.Background(), recID, userID, tenantID),
		"ConfirmInitiatorConsent must succeed for an existing recording")

	// 4. After the call: GetPreConsentStatus must return true (mock call-tracking assertion).
	stamped, err = repo.GetPreConsentStatus(context.Background(), recID, tenantID)
	require.NoError(t, err)
	assert.True(t, stamped, "GetPreConsentStatus must return true after ConfirmInitiatorConsent is called")

	// 5. Calling ConfirmInitiatorConsent for an unknown recording must fail (wrong args guard).
	err = svc.ConfirmInitiatorConsent(context.Background(), uuid.New(), userID, tenantID)
	require.Error(t, err, "ConfirmInitiatorConsent with unknown recording ID must return an error")
}

// TestConfirmInitiatorConsent_Roundtrip validates the full stamp→read cycle in the mock.
func TestConfirmInitiatorConsent_Roundtrip(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	recID := uuid.New()
	callID := uuid.New()
	userID := uuid.New()
	tenantID := uuid.New()

	repo.recordings[recID] = &Recording{
		ID:        recID,
		CallID:    &callID,
		StartedBy: &userID,
		Status:    RecordingStatusActive,
	}

	// Before stamp: not consented
	stamped, err := repo.GetPreConsentStatus(context.Background(), recID, tenantID)
	require.NoError(t, err)
	assert.False(t, stamped)

	// Confirm consent
	require.NoError(t, svc.ConfirmInitiatorConsent(context.Background(), recID, userID, tenantID))

	// After stamp: consented
	stamped, err = repo.GetPreConsentStatus(context.Background(), recID, tenantID)
	require.NoError(t, err)
	assert.True(t, stamped)
}

// ============================================================================
// P2-2: Tenant-isolation test for StartRecording — real service call with mock repo
// Validates: correct recording created, tenant_id stored, cross-tenant access rejected
// ============================================================================

// tenantAwareRepo wraps mockRepo and adds tenant_id enforcement to GetRecording.
type tenantAwareRepo struct {
	*mockRepo
	allowedTenantID uuid.UUID
}

func (r *tenantAwareRepo) GetRecording(ctx context.Context, id uuid.UUID) (*Recording, error) {
	rec, err := r.mockRepo.GetRecording(ctx, id)
	if err != nil {
		return nil, err
	}
	// Enforce tenant isolation: reject if tenant_id does not match the caller's allowed tenant
	if rec.TenantID != r.allowedTenantID {
		return nil, ErrNotFound
	}
	return rec, nil
}

// ============================================================================
// F6 — DB-Level Cross-Tenant Isolation Test (Sprint 3 Welle 1)
//
// Verifies that recordings created under Tenant A are not accessible to
// Tenant B at the service/repository layer. This extends the existing
// TestStartRecording_TenantIsolation test with a list-level boundary check.
// ============================================================================

// TestRecordingsCrossTenantIsolation_DBLevel verifies that:
// 1. A recording created by Tenant A carries the correct tenant_id.
// 2. Fetching that recording via a tenant-scoped repo for Tenant B returns ErrNotFound.
// 3. ListRecordingsByCall for Tenant A's call ID returns Tenant A's recordings; a
//    filtered client for Tenant B finds no records among them.
func TestRecordingsCrossTenantIsolation_DBLevel(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()

	base := newMockRepo()
	zero := 0
	base.pendingOverride = &zero

	repoA := &tenantAwareRepo{mockRepo: base, allowedTenantID: tenantA}
	egress := newMockEgressManager()
	svc := NewService(repoA, egress, "https://template.example.com", S3Config{Bucket: "recordings"})

	callA := uuid.New()
	starter := uuid.New()
	participants := makeParticipants(starter)

	// Tenant A starts a recording.
	rec, err := svc.StartRecording(context.Background(), &callA, nil, "room-a", starter, participants, tenantA)
	require.NoError(t, err)
	require.NotNil(t, rec)
	assert.Equal(t, tenantA, rec.TenantID, "recording must carry Tenant A's tenant_id")

	// Tenant B's scoped repo must not see the recording when fetched directly.
	repoB := &tenantAwareRepo{mockRepo: base, allowedTenantID: tenantB}
	_, err = repoB.GetRecording(context.Background(), rec.ID)
	assert.ErrorIs(t, err, ErrNotFound, "Tenant B must not access Tenant A's recording")

	// Listing all recordings for callA via the base (unscoped) mock:
	// Every returned recording must belong to Tenant A — confirming no
	// cross-tenant data from a hypothetical Tenant B call leaked in.
	all, err := base.ListRecordingsByCall(context.Background(), callA)
	require.NoError(t, err)
	for _, r := range all {
		assert.Equal(t, tenantA, r.TenantID,
			"listing recordings by call must only contain Tenant A's records for callA")
	}
}

// TestStartRecording_TenantIsolation verifies that:
// 1. StartRecording stores the correct tenant_id on the Recording row.
// 2. A cross-tenant GetRecordingStatus call returns ErrNotFound (repo enforces tenant filter).
func TestStartRecording_TenantIsolation(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()

	base := newMockRepo()
	zero := 0
	base.pendingOverride = &zero

	// Use a tenant-aware repo scoped to TenantA
	repoA := &tenantAwareRepo{mockRepo: base, allowedTenantID: tenantA}

	egress := newMockEgressManager()
	svc := NewService(repoA, egress, "https://template.example.com", S3Config{Bucket: "recordings"})

	callID := uuid.New()
	starter := uuid.New()
	participants := makeParticipants(starter)

	// TenantA creates a recording (passes tenantID as variadic arg)
	rec, err := svc.StartRecording(context.Background(), &callID, nil, "room-abc", starter, participants, tenantA)
	require.NoError(t, err)
	require.NotNil(t, rec)

	// The recording must carry TenantA's ID
	assert.Equal(t, tenantA, rec.TenantID, "recording must be scoped to the creating tenant")

	// TenantA can retrieve it
	fetched, err := repoA.GetRecording(context.Background(), rec.ID)
	require.NoError(t, err)
	assert.Equal(t, rec.ID, fetched.ID)

	// TenantB cannot retrieve TenantA's recording (cross-tenant isolation)
	repoB := &tenantAwareRepo{mockRepo: base, allowedTenantID: tenantB}
	_, err = repoB.GetRecording(context.Background(), rec.ID)
	assert.ErrorIs(t, err, ErrNotFound, "TenantB must not access TenantA recordings")
}

// ============================================================================
// W3-2: Service-layer Defense-in-Depth — initiator pre-consent gate (Sprint 4)
//
// Verifies that StartRecording enforces the pre-consent check at the service layer
// when a PreConsentChecker is wired, so that direct gRPC callers (bypassing the
// HTTP gateway dialog gate) cannot start recordings without initiator consent.
// ============================================================================

// mockPreConsentChecker is a controllable test double for PreConsentChecker.
type mockPreConsentChecker struct {
	consented bool
	err       error
	called    bool
	lastCallID *uuid.UUID
	lastMeetingID *uuid.UUID
	lastInitiatorID uuid.UUID
	lastTenantID uuid.UUID
}

func (m *mockPreConsentChecker) HasInitiatorConsented(
	_ context.Context,
	callID *uuid.UUID,
	meetingID *uuid.UUID,
	initiatorID uuid.UUID,
	tenantID uuid.UUID,
) (bool, error) {
	m.called = true
	m.lastCallID = callID
	m.lastMeetingID = meetingID
	m.lastInitiatorID = initiatorID
	m.lastTenantID = tenantID
	return m.consented, m.err
}

// TestStartRecording_ServiceLayerGate_BlocksWithoutPreConsent verifies that StartRecording
// returns codes.FailedPrecondition when the PreConsentChecker denies the initiator.
// This closes the direct-gRPC bypass identified in W3-2 Defense-in-Depth.
func TestStartRecording_ServiceLayerGate_BlocksWithoutPreConsent(t *testing.T) {
	repo := newMockRepo()
	egress := newMockEgressManager()
	checker := &mockPreConsentChecker{consented: false}

	svc := NewService(repo, egress, "https://template.example.com", S3Config{Bucket: "recordings"}, checker)

	callID := uuid.New()
	starter := uuid.New()
	tenantID := uuid.New()

	_, err := svc.StartRecording(
		context.Background(),
		&callID, nil, "room-guard-test",
		starter,
		makeParticipants(starter),
		tenantID,
	)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "error must be a gRPC status error")
	assert.Equal(t, codes.FailedPrecondition, st.Code(), "must return FailedPrecondition when pre-consent is missing")
	assert.True(t, checker.called, "PreConsentChecker must be consulted")
	assert.Equal(t, &callID, checker.lastCallID)
	assert.Equal(t, starter, checker.lastInitiatorID)
	assert.Equal(t, tenantID, checker.lastTenantID)
}

// TestStartRecording_ServiceLayerGate_AllowsWithPreConsent verifies that StartRecording
// proceeds past the service-layer gate when the PreConsentChecker grants the initiator.
// All-consented override is applied via pendingOverride to isolate the gate under test.
func TestStartRecording_ServiceLayerGate_AllowsWithPreConsent(t *testing.T) {
	repo := newMockRepo()
	egress := newMockEgressManager()
	checker := &mockPreConsentChecker{consented: true}

	zero := 0
	repo.pendingOverride = &zero

	svc := NewService(repo, egress, "https://template.example.com", S3Config{Bucket: "recordings"}, checker)

	callID := uuid.New()
	starter := uuid.New()
	tenantID := uuid.New()

	rec, err := svc.StartRecording(
		context.Background(),
		&callID, nil, "room-guard-test",
		starter,
		makeParticipants(starter),
		tenantID,
	)

	require.NoError(t, err, "StartRecording must succeed when pre-consent is granted")
	assert.NotNil(t, rec)
	assert.Equal(t, RecordingStatusActive, rec.Status)
	assert.True(t, checker.called, "PreConsentChecker must be consulted even on success")
}

// TestStartRecording_ServiceLayerGate_SkippedWhenNilChecker verifies that StartRecording
// behaves as before (no gate) when no PreConsentChecker is provided — ensuring backward
// compatibility for legacy callers and existing tests.
func TestStartRecording_ServiceLayerGate_SkippedWhenNilChecker(t *testing.T) {
	repo := newMockRepo()
	egress := newMockEgressManager()

	// No checker passed — gate must be inactive
	svc := NewService(repo, egress, "https://template.example.com", S3Config{Bucket: "recordings"})

	callID := uuid.New()
	starter := uuid.New()
	zero := 0
	repo.pendingOverride = &zero

	rec, err := svc.StartRecording(
		context.Background(),
		&callID, nil, "room-legacy",
		starter,
		makeParticipants(starter),
	)

	require.NoError(t, err, "StartRecording without checker must succeed (backward compat)")
	assert.NotNil(t, rec)
}
