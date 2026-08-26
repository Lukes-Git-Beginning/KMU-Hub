package recording

// Covers the Service methods that sat at 0% coverage before this unit
// (see coverage_start): the gateway-facing lifecycle wrappers used by
// route_video.go's status/download/metadata/cleanup/list-by-meeting
// handlers, the webhook-by-egress-ID pair, and the pure fileURLToObjectKey
// helper. StartRecording/StopRecording/SetConsent etc. already had extensive
// coverage in service_test.go via the same mockRepo -- this file fills the
// gap rather than duplicating it.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetRecordingStatus_ReturnsRecording(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	id := uuid.New()
	repo.recordings[id] = &Recording{ID: id, Status: RecordingStatusProcessing}

	got, err := svc.GetRecordingStatus(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, RecordingStatusProcessing, got.Status)
}

func TestGetRecordingStatus_NotFound(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	_, err := svc.GetRecordingStatus(context.Background(), uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestListRecordingsByMeeting_Paginates(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	meetingID := uuid.New()
	for i := 0; i < 5; i++ {
		id := uuid.New()
		repo.recordings[id] = &Recording{ID: id, MeetingID: &meetingID, Status: RecordingStatusCompleted}
	}

	page1, total, err := svc.ListRecordingsByMeeting(context.Background(), meetingID, uuid.Nil, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, page1, 2)

	page3, total, err := svc.ListRecordingsByMeeting(context.Background(), meetingID, uuid.Nil, 3, 2)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, page3, 1, "last page must carry the remainder, not overflow past total")
}

func TestListRecordingsByMeeting_PageBeyondTotalReturnsEmpty(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	meetingID := uuid.New()
	id := uuid.New()
	repo.recordings[id] = &Recording{ID: id, MeetingID: &meetingID, Status: RecordingStatusCompleted}

	page, total, err := svc.ListRecordingsByMeeting(context.Background(), meetingID, uuid.Nil, 5, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Empty(t, page)
}

func TestListRecordingsByMeeting_NoPaginationReturnsAll(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	meetingID := uuid.New()
	for i := 0; i < 3; i++ {
		id := uuid.New()
		repo.recordings[id] = &Recording{ID: id, MeetingID: &meetingID, Status: RecordingStatusCompleted}
	}

	all, total, err := svc.ListRecordingsByMeeting(context.Background(), meetingID, uuid.Nil, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, all, 3)
}

func TestUpdateRecordingMetadata_UpdatesGivenFields(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	id := uuid.New()
	repo.recordings[id] = &Recording{ID: id, Status: RecordingStatusProcessing}

	url := "https://minio.example.com/recordings/rec.mp4"
	size := int64(1024)
	duration := 90
	completed := RecordingStatusCompleted

	err := svc.UpdateRecordingMetadata(context.Background(), id, RecordingMetadata{
		FileURL: &url, FileSizeBytes: &size, DurationSeconds: &duration, Status: &completed,
	})
	require.NoError(t, err)

	got := repo.recordings[id]
	assert.Equal(t, url, *got.FileURL)
	assert.Equal(t, size, *got.FileSizeBytes)
	assert.Equal(t, duration, *got.DurationSeconds)
	assert.Equal(t, RecordingStatusCompleted, got.Status)
}

func TestUpdateRecordingMetadata_LeavesUnsetFieldsUntouched(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	id := uuid.New()
	existingURL := "https://minio.example.com/recordings/existing.mp4"
	repo.recordings[id] = &Recording{ID: id, Status: RecordingStatusProcessing, FileURL: &existingURL}

	duration := 42
	err := svc.UpdateRecordingMetadata(context.Background(), id, RecordingMetadata{DurationSeconds: &duration})
	require.NoError(t, err)

	got := repo.recordings[id]
	assert.Equal(t, existingURL, *got.FileURL, "fields omitted from the patch must survive untouched")
	assert.Equal(t, duration, *got.DurationSeconds)
}

func TestUpdateRecordingMetadata_RejectsInvalidStatus(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	id := uuid.New()
	repo.recordings[id] = &Recording{ID: id, Status: RecordingStatusProcessing}

	bogus := "not-a-real-status"
	err := svc.UpdateRecordingMetadata(context.Background(), id, RecordingMetadata{Status: &bogus})
	assert.ErrorIs(t, err, ErrInvalidStatus)
	assert.Equal(t, RecordingStatusProcessing, repo.recordings[id].Status, "rejected status must not be written")
}

func TestUpdateRecordingMetadata_NotFound(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	err := svc.UpdateRecordingMetadata(context.Background(), uuid.New(), RecordingMetadata{})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestTagRecordingWithConsents_OverwritesSnapshot(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	id := uuid.New()
	repo.recordings[id] = &Recording{ID: id, ConsentSnapshot: makeParticipants(uuid.New())}

	newParticipant := uuid.New()
	err := svc.TagRecordingWithConsents(context.Background(), id, makeParticipants(newParticipant))
	require.NoError(t, err)
	assert.Len(t, repo.recordings[id].ConsentSnapshot, 1)
	assert.Equal(t, newParticipant, repo.recordings[id].ConsentSnapshot[0].UserID)
}

func TestGetRecordingConsents_AllConsentedTrueWhenEveryLiveConsentTrue(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	id := uuid.New()
	userA, userB := uuid.New(), uuid.New()
	repo.recordings[id] = &Recording{ID: id, ConsentSnapshot: makeParticipants(userA, userB)}
	repo.consents[id] = []RecordingConsent{
		{RecordingID: id, UserID: userA, Consented: true, RespondedAt: time.Now()},
		{RecordingID: id, UserID: userB, Consented: true, RespondedAt: time.Now()},
	}

	status, err := svc.GetRecordingConsents(context.Background(), id)
	require.NoError(t, err)
	assert.True(t, status.AllConsented)
	assert.Len(t, status.Consents, 2)
}

func TestGetRecordingConsents_FalseWhenAnyLiveConsentDeclined(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	id := uuid.New()
	userA, userB := uuid.New(), uuid.New()
	repo.recordings[id] = &Recording{ID: id, ConsentSnapshot: makeParticipants(userA, userB)}
	repo.consents[id] = []RecordingConsent{
		{RecordingID: id, UserID: userA, Consented: true, RespondedAt: time.Now()},
		{RecordingID: id, UserID: userB, Consented: false, RespondedAt: time.Now()},
	}

	status, err := svc.GetRecordingConsents(context.Background(), id)
	require.NoError(t, err)
	assert.False(t, status.AllConsented)
}

func TestGetRecordingConsents_FalseWhenSnapshotPendingAndNoLiveRows(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	id := uuid.New()
	repo.recordings[id] = &Recording{ID: id, ConsentSnapshot: makeParticipants(uuid.New())}
	// No live consent rows recorded yet.

	status, err := svc.GetRecordingConsents(context.Background(), id)
	require.NoError(t, err)
	assert.False(t, status.AllConsented, "an unanswered snapshot must not read as all-consented")
}

func TestCompleteRecordingByEgressID_UpdatesMatchingRecording(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	id := uuid.New()
	egressID := "egress-xyz"
	repo.recordings[id] = &Recording{ID: id, Status: RecordingStatusActive, EgressID: &egressID}

	err := svc.CompleteRecordingByEgressID(context.Background(), egressID, "https://files/rec.mp4", 2048, 60)
	require.NoError(t, err)

	got := repo.recordings[id]
	assert.Equal(t, RecordingStatusCompleted, got.Status)
	assert.Equal(t, "https://files/rec.mp4", *got.FileURL)
}

func TestCompleteRecordingByEgressID_UnknownEgressID(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	err := svc.CompleteRecordingByEgressID(context.Background(), "does-not-exist", "url", 1, 1)
	assert.Error(t, err)
}

func TestFailRecordingByEgressID_MarksMatchingRecordingFailed(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	id := uuid.New()
	egressID := "egress-fail"
	repo.recordings[id] = &Recording{ID: id, Status: RecordingStatusActive, EgressID: &egressID}

	err := svc.FailRecordingByEgressID(context.Background(), egressID, "egress crashed")
	require.NoError(t, err)
	assert.Equal(t, RecordingStatusFailed, repo.recordings[id].Status)
}

func TestFailRecordingByEgressID_UnknownEgressID(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	err := svc.FailRecordingByEgressID(context.Background(), "does-not-exist", "reason")
	assert.Error(t, err)
}

// ============================================================================
// CleanupExpiredRecording (single) — admin/cron endpoint.
//
// This does NOT assert any MinIO object deletion: unlike the bulk
// CleanupExpiredRecordings (service.go:396), the single-recording path never
// calls s.objectStore()/RemoveObject at all before deleting the DB row -- a
// real gap flagged as fix-cleanup-single-recording-orphans-minio-object in
// BACKLOG.yml. Asserting the current (missing) deletion here would either
// pin the bug as "correct" or silently start failing the moment it is fixed;
// the mockRepo also has no object-store double to assert against. This test
// only locks the retention-gate and not-found behavior that IS the contract.
// ============================================================================

func TestCleanupExpiredRecording_DeletesWhenExpired(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	id := uuid.New()
	past := time.Now().Add(-time.Hour)
	repo.recordings[id] = &Recording{ID: id, Status: RecordingStatusCompleted, RetentionExpiresAt: &past}

	err := svc.CleanupExpiredRecording(context.Background(), id)
	require.NoError(t, err)
	_, stillThere := repo.recordings[id]
	assert.False(t, stillThere)
}

func TestCleanupExpiredRecording_NotYetExpired(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	id := uuid.New()
	future := time.Now().Add(time.Hour)
	repo.recordings[id] = &Recording{ID: id, Status: RecordingStatusCompleted, RetentionExpiresAt: &future}

	err := svc.CleanupExpiredRecording(context.Background(), id)
	assert.ErrorIs(t, err, ErrNotExpired)
	_, stillThere := repo.recordings[id]
	assert.True(t, stillThere, "a not-yet-expired recording must not be deleted")
}

func TestCleanupExpiredRecording_NilRetentionTreatedAsNotExpired(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	id := uuid.New()
	repo.recordings[id] = &Recording{ID: id, Status: RecordingStatusCompleted, RetentionExpiresAt: nil}

	err := svc.CleanupExpiredRecording(context.Background(), id)
	assert.ErrorIs(t, err, ErrNotExpired)
}

func TestCleanupExpiredRecording_NotFound(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	err := svc.CleanupExpiredRecording(context.Background(), uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

// ============================================================================
// GetRecordingDownloadURL — guard clauses only.
//
// The presign happy path needs a reachable MinIO/S3 endpoint (or an
// interface seam that does not exist on *minio.Client today), so it is out
// of reach for a mockRepo-only test. These cover everything that runs
// BEFORE the storage client is touched: status precondition, missing
// file_url, and the participant ACL check that stands in for tenant
// isolation on this read path (service.go:615-679).
// ============================================================================

func TestGetRecordingDownloadURL_FailsWhenNotCompleted(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	id := uuid.New()
	repo.recordings[id] = &Recording{ID: id, Status: RecordingStatusProcessing}

	_, _, err := svc.GetRecordingDownloadURL(context.Background(), id, uuid.New())
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestGetRecordingDownloadURL_FailsWhenFileURLMissing(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	id := uuid.New()
	repo.recordings[id] = &Recording{ID: id, Status: RecordingStatusCompleted, FileURL: nil}

	_, _, err := svc.GetRecordingDownloadURL(context.Background(), id, uuid.New())
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestGetRecordingDownloadURL_DeniesNonParticipant(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	id := uuid.New()
	fileURL := "https://minio.example.com/recordings/rec.mp4"
	repo.recordings[id] = &Recording{ID: id, Status: RecordingStatusCompleted, FileURL: &fileURL}
	repo.participantUsers = []uuid.UUID{uuid.New()} // caller is not in this list

	_, _, err := svc.GetRecordingDownloadURL(context.Background(), id, uuid.New())
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestGetRecordingDownloadURL_NotFound(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo, nil, "", S3Config{})

	_, _, err := svc.GetRecordingDownloadURL(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

// ============================================================================
// fileURLToObjectKey — pure function, no repo/DB needed.
// ============================================================================

func TestFileURLToObjectKey(t *testing.T) {
	cases := []struct {
		name           string
		fileURL        string
		endpoint       string
		publicEndpoint string
		bucket         string
		want           string
	}{
		{
			name:     "full https URL with internal endpoint and bucket prefix",
			fileURL:  "https://minio.internal:9000/recordings/2026/rec.mp4",
			endpoint: "minio.internal:9000",
			bucket:   "recordings",
			want:     "2026/rec.mp4",
		},
		{
			name:           "full https URL matching the public endpoint instead",
			fileURL:        "https://cdn.example.com/recordings/2026/rec.mp4",
			endpoint:       "minio.internal:9000",
			publicEndpoint: "cdn.example.com",
			bucket:         "recordings",
			want:           "2026/rec.mp4",
		},
		{
			name:     "s3 scheme with bucket prefix",
			fileURL:  "s3://recordings/2026/rec.mp4",
			endpoint: "minio.internal:9000",
			bucket:   "recordings",
			want:     "2026/rec.mp4",
		},
		{
			name:     "bare bucket-relative path",
			fileURL:  "recordings/2026/rec.mp4",
			endpoint: "minio.internal:9000",
			bucket:   "recordings",
			want:     "2026/rec.mp4",
		},
		{
			name:     "bare key without bucket prefix",
			fileURL:  "2026/rec.mp4",
			endpoint: "minio.internal:9000",
			bucket:   "recordings",
			want:     "2026/rec.mp4",
		},
		{
			name:     "leading slash without scheme",
			fileURL:  "/recordings/2026/rec.mp4",
			endpoint: "minio.internal:9000",
			bucket:   "recordings",
			want:     "2026/rec.mp4",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := fileURLToObjectKey(tc.fileURL, tc.endpoint, tc.publicEndpoint, tc.bucket)
			assert.Equal(t, tc.want, got)
		})
	}
}
