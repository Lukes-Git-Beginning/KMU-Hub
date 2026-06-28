package recording

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RetentionDays is the default retention period for completed recordings (DSGVO compliance)
const RetentionDays = 30

// EgressManager abstracts LiveKit Egress operations for testability
type EgressManager interface {
	StartRoomCompositeEgress(ctx context.Context, roomName, templateURL string, s3Config S3Config) (string, error)
	StopEgress(ctx context.Context, egressID string) error
}

// PreConsentChecker verifies that an initiator has explicitly confirmed pre-recording consent
// before a recording is allowed to start. Checked in the service layer so that direct
// gRPC calls (bypassing the HTTP gateway) cannot skip the consent gate.
//
// Implementations look up whether the given initiator has an active pre-consent token
// for the specified call or meeting session scoped to the tenant.
type PreConsentChecker interface {
	HasInitiatorConsented(ctx context.Context, callID *uuid.UUID, meetingID *uuid.UUID, initiatorID uuid.UUID, tenantID uuid.UUID) (bool, error)
}

// S3Config holds the S3-compatible storage configuration for recordings
type S3Config struct {
	Endpoint        string
	AccessKey       string
	Secret          string
	Bucket          string
	UseSSL          bool
	PublicEndpoint  string
	PublicUseSSL    bool
}

// Service handles recording business logic including DSGVO consent management
type Service struct {
	repo          Repository
	egressManager EgressManager // nil if not configured
	preConsent    PreConsentChecker // nil disables the service-layer gate (legacy/test mode)
	s3Config      S3Config
	templateURL   string
	enabled       bool
	// lazy MinIO client for presigned downloads and object deletion (A4/A5)
	minioOnce   sync.Once
	minioClient *minio.Client
	minioErr    error
}

// NewService creates a new recording service.
// If egressManager is nil, the service operates in disabled mode.
// The preConsent gate is disabled when checker is nil (backward-compatible).
func NewService(repo Repository, egressManager EgressManager, templateURL string, s3Config S3Config, checker ...PreConsentChecker) *Service {
	svc := &Service{
		repo:          repo,
		egressManager: egressManager,
		s3Config:      s3Config,
		templateURL:   templateURL,
		enabled:       egressManager != nil,
	}
	if len(checker) > 0 {
		svc.preConsent = checker[0]
	}
	return svc
}

// StartRecording initiates a recording for a call or meeting.
// The initiator must have confirmed their own pre-recording consent via ConfirmInitiatorConsent
// before calling this method (enforced via ErrPreConsentMissing).
// All participants must have responded to the consent prompt before recording can begin.
// startedBy is the user who triggered the recording (stored for audit).
// participants is the full list of call/meeting participants at start time; their IDs are
// used for the consent gate and the list is persisted as an immutable DSGVO snapshot.
// Sets a 30-day retention period on the recording.
// tenantID is required for tenant-scoped consent lookup and recording creation.
//
// P0 fix (Welle-3.5): Pre-consent check is performed BEFORE CreateRecording to prevent
// orphaned rows when consent is still pending.
func (s *Service) StartRecording(ctx context.Context, callID *uuid.UUID, meetingID *uuid.UUID, roomName string, startedBy uuid.UUID, participants []ParticipantConsentInfo, tenantID ...uuid.UUID) (*Recording, error) {
	if !s.enabled {
		return nil, ErrEgressNotConfigured
	}

	if callID == nil && meetingID == nil {
		return nil, ErrNoCallOrMeeting
	}

	if len(participants) == 0 {
		return nil, ErrNoParticipants
	}

	// --- Service-layer gate: initiator pre-recording consent (R2-P0.4 / W3-2 Defense-in-Depth) ---
	// This check mirrors the HTTP-gateway dialog gate so that direct gRPC calls (service-to-service
	// or external clients bypassing the gateway) cannot start a recording without the initiator's
	// explicit consent acknowledgement.
	//
	// The gate is only active when a PreConsentChecker is wired (production path via NewService).
	// Legacy callers and tests that do not supply a checker skip the gate gracefully.
	if s.preConsent != nil {
		var tid uuid.UUID
		if len(tenantID) > 0 {
			tid = tenantID[0]
		}
		consented, err := s.preConsent.HasInitiatorConsented(ctx, callID, meetingID, startedBy, tid)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "pre-consent check failed: %v", err)
		}
		if !consented {
			return nil, status.Error(codes.FailedPrecondition, "initiator pre-consent missing")
		}
	}

	// Extract participant IDs for the consent gate
	participantIDs := make([]uuid.UUID, len(participants))
	for i, p := range participants {
		participantIDs[i] = p.UserID
	}

	// Use a temporary recording ID to check pending consents before creating the row.
	// CountPendingConsents only needs participant IDs — it does not require a stored recording.
	// We use a sentinel recording ID here; the real ID is generated after consent passes.
	//
	// NOTE: CountPendingConsents is called with the final recording ID after creation
	// for the *participant* consent gate (those are per-recording). The *initiator* pre-consent
	// check (GetPreConsentStatus) is a separate gate enforced by the gateway before StartRecording
	// is even called. So we proceed directly to creating the recording row and then checking
	// participant consent — but we add a defer-rollback if consent is still pending to avoid orphans.
	now := time.Now()
	retentionExpires := now.Add(RetentionDays * 24 * time.Hour)

	// Extract optional tenantID (variadic for backwards-compatibility with call-sites that
	// do not yet pass a tenant). When provided it is stored on the recording row so the
	// repository layer can enforce tenant_id-scoped queries.
	var tid uuid.UUID
	if len(tenantID) > 0 {
		tid = tenantID[0]
	}

	rec := &Recording{
		ID:                 uuid.New(),
		TenantID:           tid,
		CallID:             callID,
		MeetingID:          meetingID,
		StartedBy:          &startedBy,
		ConsentSnapshot:    participants,
		Status:             RecordingStatusActive,
		RetentionExpiresAt: &retentionExpires,
		CreatedAt:          now,
	}

	if err := s.repo.CreateRecording(ctx, rec); err != nil {
		return nil, fmt.Errorf("create recording: %w", err)
	}

	// Check consent: all current participants must have responded.
	// If pending > 0 we delete the just-created recording row to avoid orphans.
	pending, err := s.repo.CountPendingConsents(ctx, rec.ID, participantIDs)
	if err != nil {
		// Best-effort cleanup
		if delErr := s.repo.DeleteRecording(ctx, rec.ID); delErr != nil {
			slog.Error("failed to delete orphaned recording after consent-check error",
				"recording_id", rec.ID, "error", delErr)
		}
		return nil, fmt.Errorf("check consents: %w", err)
	}

	if pending > 0 {
		// Consent still pending — delete the recording row so no orphan is left.
		if delErr := s.repo.DeleteRecording(ctx, rec.ID); delErr != nil {
			slog.Error("failed to delete orphaned recording on ErrConsentPending",
				"recording_id", rec.ID, "error", delErr)
		}
		slog.Info("recording consent pending — row cleaned up",
			"pending_count", pending,
		)
		return nil, ErrConsentPending
	}

	// All consented - start egress
	egressID, err := s.egressManager.StartRoomCompositeEgress(ctx, roomName, s.templateURL, s.s3Config)
	if err != nil {
		// Mark recording as failed if egress cannot start
		rec.Status = RecordingStatusFailed
		if updateErr := s.repo.UpdateRecording(ctx, rec); updateErr != nil {
			slog.Error("failed to mark recording as failed after egress error",
				"recording_id", rec.ID,
				"error", updateErr,
			)
		}
		return nil, fmt.Errorf("start egress: %w", err)
	}

	rec.EgressID = &egressID
	if err := s.repo.UpdateRecording(ctx, rec); err != nil {
		return nil, fmt.Errorf("update recording with egress id: %w", err)
	}

	slog.Info("recording started",
		"recording_id", rec.ID,
		"room_name", roomName,
		"egress_id", egressID,
	)

	return rec, nil
}

// StopRecording stops an active recording.
func (s *Service) StopRecording(ctx context.Context, recordingID uuid.UUID) (*Recording, error) {
	rec, err := s.repo.GetRecording(ctx, recordingID)
	if err != nil {
		return nil, err
	}

	if rec.Status != RecordingStatusActive {
		return nil, ErrRecordingNotActive
	}

	// Stop egress if configured
	if s.enabled && rec.EgressID != nil {
		if err := s.egressManager.StopEgress(ctx, *rec.EgressID); err != nil {
			slog.Error("failed to stop egress",
				"recording_id", recordingID,
				"egress_id", *rec.EgressID,
				"error", err,
			)
			// Continue to update status despite egress error
		}
	}

	rec.Status = RecordingStatusProcessing
	if err := s.repo.UpdateRecording(ctx, rec); err != nil {
		return nil, fmt.Errorf("update recording status: %w", err)
	}

	slog.Info("recording stopped",
		"recording_id", recordingID,
		"status", rec.Status,
	)

	return rec, nil
}

// ConfirmInitiatorConsent records that the recording initiator has acknowledged the
// pre-recording consent dialog. This must be called before StartRecording when the
// flow requires the initiator to explicitly confirm they understand the recording will begin.
// The stamp (pre_recording_consent_at + sentinel initiator_consent_id) is written via the
// repo. Returns ErrNotFound if the recording does not exist or does not belong to userID.
func (s *Service) ConfirmInitiatorConsent(ctx context.Context, recordingID, userID, tenantID uuid.UUID) error {
	if err := s.repo.MarkInitiatorConsent(ctx, recordingID, userID, tenantID); err != nil {
		return fmt.Errorf("mark initiator consent: %w", err)
	}

	slog.Info("initiator pre-recording consent confirmed",
		"recording_id", recordingID,
		"user_id", userID,
	)

	return nil
}

// SetConsent stores a participant's consent response for a recording.
func (s *Service) SetConsent(ctx context.Context, recordingID, userID uuid.UUID, consented bool) error {
	consent := &RecordingConsent{
		RecordingID: recordingID,
		UserID:      userID,
		Consented:   consented,
		RespondedAt: time.Now(),
	}

	if err := s.repo.SetConsent(ctx, consent); err != nil {
		return fmt.Errorf("set consent: %w", err)
	}

	slog.Info("recording consent set",
		"recording_id", recordingID,
		"user_id", userID,
		"consented", consented,
	)

	return nil
}

// GetConsentStatus checks whether all specified participants have responded to consent.
//
// participantIDs == nil/empty is intentionally not treated as "all participants":
// the underlying CountPendingConsents query unnests the array, so an empty input
// returns 0 unconditionally and would falsely report allResponded == true. When
// the caller cannot supply the expected participant set (e.g. read-only GET
// endpoints that just list current consents), we return allResponded = false so
// downstream code never starts a recording based on a meaningless zero count.
// The list of stored consents is still returned so the UI can render them.
func (s *Service) GetConsentStatus(ctx context.Context, recordingID uuid.UUID, participantIDs []uuid.UUID) (bool, []RecordingConsent, error) {
	consents, err := s.repo.GetConsents(ctx, recordingID)
	if err != nil {
		return false, nil, fmt.Errorf("get consents: %w", err)
	}

	if len(participantIDs) == 0 {
		// Cannot decide allResponded without an expected participant list.
		return false, consents, nil
	}

	pending, err := s.repo.CountPendingConsents(ctx, recordingID, participantIDs)
	if err != nil {
		return false, nil, fmt.Errorf("count pending consents: %w", err)
	}

	allResponded := pending == 0
	return allResponded, consents, nil
}

// CompleteRecording updates a recording after Egress processing completes.
// Called by the LiveKit Egress webhook when the file is ready.
func (s *Service) CompleteRecording(ctx context.Context, recordingID uuid.UUID, fileURL string, fileSizeBytes int64, durationSeconds int) error {
	rec, err := s.repo.GetRecording(ctx, recordingID)
	if err != nil {
		return err
	}

	rec.Status = RecordingStatusCompleted
	rec.FileURL = &fileURL
	rec.FileSizeBytes = &fileSizeBytes
	rec.DurationSeconds = &durationSeconds

	if err := s.repo.UpdateRecording(ctx, rec); err != nil {
		return fmt.Errorf("complete recording: %w", err)
	}

	slog.Info("recording completed",
		"recording_id", recordingID,
		"file_url", fileURL,
		"file_size_bytes", fileSizeBytes,
		"duration_seconds", durationSeconds,
	)

	return nil
}

// FailRecording marks a recording as failed.
// Called by the LiveKit Egress webhook on processing failure.
func (s *Service) FailRecording(ctx context.Context, recordingID uuid.UUID, reason string) error {
	rec, err := s.repo.GetRecording(ctx, recordingID)
	if err != nil {
		return err
	}

	rec.Status = RecordingStatusFailed
	if err := s.repo.UpdateRecording(ctx, rec); err != nil {
		return fmt.Errorf("fail recording: %w", err)
	}

	slog.Error("recording failed",
		"recording_id", recordingID,
		"reason", reason,
	)

	return nil
}

// CompleteRecordingByEgressID updates a recording by egress ID after Egress
// processing completes. Called by the LiveKit Egress webhook path.
func (s *Service) CompleteRecordingByEgressID(ctx context.Context, egressID, fileURL string, fileSizeBytes int64, durationSeconds int) error {
	rec, err := s.repo.GetRecordingByEgressID(ctx, egressID)
	if err != nil {
		return fmt.Errorf("get recording by egress id: %w", err)
	}
	return s.CompleteRecording(ctx, rec.ID, fileURL, fileSizeBytes, durationSeconds)
}

// FailRecordingByEgressID marks a recording as failed, looked up by egress ID.
// Called by the LiveKit Egress webhook path on processing failure.
func (s *Service) FailRecordingByEgressID(ctx context.Context, egressID, reason string) error {
	rec, err := s.repo.GetRecordingByEgressID(ctx, egressID)
	if err != nil {
		return fmt.Errorf("get recording by egress id: %w", err)
	}
	return s.FailRecording(ctx, rec.ID, reason)
}

// CleanupExpiredRecordings deletes recordings past their retention period.
// Before deleting the DB row it attempts to remove the underlying MinIO object.
// Object deletion is best-effort: on failure it logs and proceeds with the DB delete.
// Returns the number of deleted recordings.
func (s *Service) CleanupExpiredRecordings(ctx context.Context) (int, error) {
	expired, err := s.repo.ListExpiredRecordings(ctx, time.Now())
	if err != nil {
		return 0, fmt.Errorf("list expired recordings: %w", err)
	}

	deleted := 0
	for _, rec := range expired {
		// A5: best-effort MinIO object deletion before removing the DB row.
		if rec.FileURL != nil && *rec.FileURL != "" {
			if mc, mcErr := s.objectStore(); mcErr != nil {
				slog.Error("failed to obtain minio client for retention cleanup",
					"recording_id", rec.ID,
					"error", mcErr,
				)
			} else {
				objKey := fileURLToObjectKey(*rec.FileURL, s.s3Config.Endpoint, s.s3Config.PublicEndpoint, s.s3Config.Bucket)
				if removeErr := mc.RemoveObject(ctx, s.s3Config.Bucket, objKey, minio.RemoveObjectOptions{}); removeErr != nil {
					slog.Error("failed to delete recording object from storage",
						"recording_id", rec.ID,
						"object_key", objKey,
						"error", removeErr,
					)
					// Non-fatal: continue to DB delete.
				} else {
					slog.Info("recording object deleted from storage",
						"recording_id", rec.ID,
						"object_key", objKey,
					)
				}
			}
		}

		if err := s.repo.DeleteRecording(ctx, rec.ID); err != nil {
			slog.Error("failed to delete expired recording",
				"recording_id", rec.ID,
				"error", err,
			)
			continue
		}
		deleted++

		slog.Info("expired recording deleted",
			"recording_id", rec.ID,
			"retention_expired_at", rec.RetentionExpiresAt,
		)
	}

	return deleted, nil
}

// ListRecordings returns recordings for a call or meeting.
func (s *Service) ListRecordings(ctx context.Context, callID *uuid.UUID, meetingID *uuid.UUID) ([]Recording, error) {
	if callID != nil {
		return s.repo.ListRecordingsByCall(ctx, *callID)
	}
	if meetingID != nil {
		return s.repo.ListRecordingsByMeeting(ctx, *meetingID)
	}
	return nil, ErrNoCallOrMeeting
}

// GetRecordingConsents returns the consent snapshot stored on the recording, optionally enriched
// with live consent records from the recording_consents table.
func (s *Service) GetRecordingConsents(ctx context.Context, recordingID uuid.UUID) (*RecordingConsentStatus, error) {
	rec, err := s.repo.GetRecording(ctx, recordingID)
	if err != nil {
		return nil, err
	}

	// Fetch live consent rows for up-to-date responded_at / consented values
	liveConsents, err := s.repo.GetConsentsWithUser(ctx, recordingID)
	if err != nil {
		return nil, fmt.Errorf("get consents with user: %w", err)
	}

	allConsented := true
	for _, c := range liveConsents {
		if !c.Consented {
			allConsented = false
			break
		}
	}
	// If no live consents yet, fall back to snapshot to determine pending state
	if len(liveConsents) == 0 && len(rec.ConsentSnapshot) > 0 {
		allConsented = false
	}

	return &RecordingConsentStatus{
		RecordingID:  recordingID,
		Consents:     liveConsents,
		AllConsented: allConsented,
	}, nil
}

// TagRecordingWithConsents overwrites the consent_snapshot JSONB on an existing recording.
// Called after StartRecording when the snapshot needs to be refreshed (e.g. late-joiners).
// Errors are non-fatal in the StartRecording flow — callers should log and continue.
func (s *Service) TagRecordingWithConsents(ctx context.Context, recordingID uuid.UUID, snapshot []ParticipantConsentInfo) error {
	if err := s.repo.TagRecordingWithConsents(ctx, recordingID, snapshot); err != nil {
		return fmt.Errorf("tag recording consents: %w", err)
	}

	slog.Info("recording consent snapshot tagged",
		"recording_id", recordingID,
		"participant_count", len(snapshot),
	)

	return nil
}

// UpdateRecordingMetadata updates mutable metadata fields on an existing recording.
// Used by the egress webhook and administrative operations.
func (s *Service) UpdateRecordingMetadata(ctx context.Context, recordingID uuid.UUID, meta RecordingMetadata) error {
	rec, err := s.repo.GetRecording(ctx, recordingID)
	if err != nil {
		return err
	}

	if meta.FileURL != nil {
		rec.FileURL = meta.FileURL
	}
	if meta.FileSizeBytes != nil {
		rec.FileSizeBytes = meta.FileSizeBytes
	}
	if meta.DurationSeconds != nil {
		rec.DurationSeconds = meta.DurationSeconds
	}
	if meta.Status != nil {
		if !ValidRecordingStatuses[*meta.Status] {
			return ErrInvalidStatus
		}
		rec.Status = *meta.Status
	}

	if err := s.repo.UpdateRecording(ctx, rec); err != nil {
		return fmt.Errorf("update recording metadata: %w", err)
	}

	slog.Info("recording metadata updated",
		"recording_id", recordingID,
	)

	return nil
}

// GetRecordingStatus returns the current status of a single recording.
// This is the primary polling target for the useRecordingStatus frontend hook.
func (s *Service) GetRecordingStatus(ctx context.Context, recordingID uuid.UUID) (*Recording, error) {
	rec, err := s.repo.GetRecording(ctx, recordingID)
	if err != nil {
		return nil, err
	}
	return rec, nil
}

// CleanupExpiredRecording deletes a single recording by ID if its retention period has elapsed.
// Intended to be called by a cron job (Sprint 4 / R2-P1.11). Returns ErrNotExpired if the
// recording has not yet reached its retention expiry.
func (s *Service) CleanupExpiredRecording(ctx context.Context, recordingID uuid.UUID) error {
	rec, err := s.repo.GetRecording(ctx, recordingID)
	if err != nil {
		return err
	}

	if rec.RetentionExpiresAt == nil || rec.RetentionExpiresAt.After(time.Now()) {
		return ErrNotExpired
	}

	if err := s.repo.DeleteRecording(ctx, recordingID); err != nil {
		return fmt.Errorf("delete expired recording: %w", err)
	}

	slog.Info("expired recording deleted (single)",
		"recording_id", recordingID,
		"retention_expired_at", rec.RetentionExpiresAt,
	)

	return nil
}

// ListRecordingsByMeeting returns all recordings for a given meeting, scoped to tenantID.
// TenantID is currently unused (single-tenant), but included for Option-B-Retrofit in Sprint 2/3.
func (s *Service) ListRecordingsByMeeting(ctx context.Context, meetingID uuid.UUID, _ uuid.UUID, page, pageSize int) ([]Recording, int, error) {
	all, err := s.repo.ListRecordingsByMeeting(ctx, meetingID)
	if err != nil {
		return nil, 0, err
	}

	total := len(all)
	if page > 0 && pageSize > 0 {
		start := (page - 1) * pageSize
		if start >= total {
			return []Recording{}, total, nil
		}
		end := min(start+pageSize, total)
		all = all[start:end]
	}

	return all, total, nil
}

// ListRecordingsWithAccess returns recordings a user has access to, optionally filtered by meeting.
// This is the Phase 11 integration point: the central file manager calls this to discover
// recordings per meeting and enforce participant-only access.
func (s *Service) ListRecordingsWithAccess(ctx context.Context, userID uuid.UUID, callID, meetingID *uuid.UUID) ([]Recording, error) {
	return s.repo.ListRecordingsWithAccess(ctx, userID, callID, meetingID)
}

// GetRecordingParticipants returns the user IDs who have access to a recording.
// Phase 11 uses this to enforce file-level ACL on recording downloads.
func (s *Service) GetRecordingParticipants(ctx context.Context, recordingID uuid.UUID) ([]uuid.UUID, error) {
	return s.repo.GetRecordingParticipants(ctx, recordingID)
}

// GetRecordingDownloadURL generates a presigned MinIO GET URL for a completed recording.
// ACL: caller must be a participant (via call_participants or meeting_attendees).
// Returns codes.FailedPrecondition if the recording is not yet completed,
// codes.PermissionDenied if the caller was not a participant.
func (s *Service) GetRecordingDownloadURL(ctx context.Context, recordingID, callerID uuid.UUID) (url string, expiresAt time.Time, err error) {
	rec, err := s.repo.GetRecording(ctx, recordingID)
	if err != nil {
		return "", time.Time{}, err
	}

	if rec.Status != RecordingStatusCompleted {
		return "", time.Time{}, status.Errorf(codes.FailedPrecondition,
			"recording is not completed (status: %s)", rec.Status)
	}

	if rec.FileURL == nil || *rec.FileURL == "" {
		return "", time.Time{}, status.Error(codes.FailedPrecondition, "recording file not available")
	}

	// ACL: caller must be a participant
	participants, err := s.repo.GetRecordingParticipants(ctx, recordingID)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("get recording participants: %w", err)
	}
	allowed := false
	for _, uid := range participants {
		if uid == callerID {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", time.Time{}, status.Error(codes.PermissionDenied, "not a participant of this recording")
	}

	mc, err := s.objectStore()
	if err != nil {
		return "", time.Time{}, status.Errorf(codes.Internal, "storage client unavailable: %v", err)
	}

	objKey := fileURLToObjectKey(*rec.FileURL, s.s3Config.Endpoint, s.s3Config.PublicEndpoint, s.s3Config.Bucket)
	const downloadExpiry = time.Hour
	expiresAt = time.Now().Add(downloadExpiry)

	// Use public endpoint client when available so the URL carries the browser-reachable host.
	presignClient := mc
	if s.s3Config.PublicEndpoint != "" {
		if pubClient, pubErr := minio.New(s.s3Config.PublicEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(s.s3Config.AccessKey, s.s3Config.Secret, ""),
			Secure: s.s3Config.PublicUseSSL,
		}); pubErr == nil {
			presignClient = pubClient
		}
	}

	presignURL, err := presignClient.PresignedGetObject(ctx, s.s3Config.Bucket, objKey, downloadExpiry, nil)
	if err != nil {
		return "", time.Time{}, status.Errorf(codes.Internal, "failed to generate presigned URL: %v", err)
	}

	slog.Info("recording download URL generated",
		"recording_id", recordingID,
		"caller_id", callerID,
		"object_key", objKey,
		"expires_at", expiresAt,
	)

	return presignURL.String(), expiresAt, nil
}

// objectStore lazily constructs the MinIO client from s3Config.
// Subsequent calls return the cached client without locking overhead.
func (s *Service) objectStore() (*minio.Client, error) {
	s.minioOnce.Do(func() {
		s.minioClient, s.minioErr = minio.New(s.s3Config.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(s.s3Config.AccessKey, s.s3Config.Secret, ""),
			Secure: s.s3Config.UseSSL,
		})
	})
	return s.minioClient, s.minioErr
}

// fileURLToObjectKey converts a file_url stored by the LiveKit Egress webhook into
// a bucket-relative MinIO object key for presign / deletion.
//
// LiveKit Egress writes one of:
//   - A full URL:  http(s)://<endpoint>/<bucket>/<key>  or  s3://<bucket>/<key>
//   - A bare path: <bucket>/<key>  or  <key>
//
// The function strips any scheme+host, then strips a leading /<bucket>/ or <bucket>/
// prefix so the remainder is the object key only.
func fileURLToObjectKey(fileURL, endpoint, publicEndpoint, bucket string) string {
	key := fileURL

	// Strip http/https scheme and host (internal or public endpoint).
	for _, host := range []string{endpoint, publicEndpoint} {
		if host == "" {
			continue
		}
		for _, scheme := range []string{"https://", "http://"} {
			prefix := scheme + host + "/"
			if strings.HasPrefix(key, prefix) {
				key = key[len(prefix):]
				break
			}
		}
	}

	// Strip s3:// scheme
	key = strings.TrimPrefix(key, "s3://")

	// Strip leading slash
	key = strings.TrimPrefix(key, "/")

	// Strip bucket prefix (with or without trailing slash)
	key = strings.TrimPrefix(key, bucket+"/")

	return key
}
