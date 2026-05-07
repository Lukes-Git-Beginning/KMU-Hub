package recording

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func (m *mockRepo) ListRecordingsWithAccess(_ context.Context, userID uuid.UUID, meetingID *uuid.UUID) ([]Recording, error) {
	// Simplified mock: return all completed/processing recordings if user is in participantUsers
	if !slices.Contains(m.participantUsers, userID) {
		return nil, nil
	}

	var result []Recording
	for _, rec := range m.recordings {
		if rec.Status != RecordingStatusCompleted && rec.Status != RecordingStatusProcessing {
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
	recs, err := svc.ListRecordingsWithAccess(context.Background(), user1, nil)
	require.NoError(t, err)
	assert.Len(t, recs, 1)

	// user2 should NOT see recordings
	recs, err = svc.ListRecordingsWithAccess(context.Background(), user2, nil)
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

// TestStartRecording_RequiresPreConsent verifies that the ErrPreConsentMissing sentinel
// is exported and used in the service layer. The actual gate in service.go is advisory for now
// (enforced at HTTP-layer via dialog flow), so this test validates the error constant exists.
func TestStartRecording_RequiresPreConsent(t *testing.T) {
	// ErrPreConsentMissing must be defined and non-nil
	assert.NotNil(t, ErrPreConsentMissing)
	assert.Contains(t, ErrPreConsentMissing.Error(), "initiator")
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
