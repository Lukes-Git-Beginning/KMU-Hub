package recording

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestConsentGate_DB_ParticipantsConsentToThePendingRecording runs the whole recording
// consent flow against the real repository as kmuhub_app: the participants consent on the
// ID StartRecording hands out, and the egress starts only when the initiator confirms after
// every other participant said yes. Before, StartRecording counted consents for an ID it had
// just created, so every start with a second participant failed.
func TestConsentGate_DB_ParticipantsConsentToThePendingRecording(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Recording Consent Gate Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenant)

	initiator := seedRecordingUser(t, pool, tenant)
	defer testutil.CleanupRow(t, pool, "users", initiator)
	second := seedRecordingUser(t, pool, tenant)
	defer testutil.CleanupRow(t, pool, "users", second)
	third := seedRecordingUser(t, pool, tenant)
	defer testutil.CleanupRow(t, pool, "users", third)
	meeting := seedRecordingMeeting(t, pool, tenant, initiator)
	defer testutil.CleanupRow(t, pool, "meetings", meeting)

	repo := NewPostgresRepository(pool)
	egress := newMockEgressManager()
	svc := NewService(repo, egress, "https://template.example.com", S3Config{Bucket: "recordings"})
	ctx := testutil.WithTenantCtx(context.Background(), tenant)

	now := time.Now().UTC()
	participants := []ParticipantConsentInfo{
		{UserID: initiator, DisplayName: "Initiator", JoinedAt: now},
		{UserID: second, DisplayName: "Second", JoinedAt: now},
		{UserID: third, DisplayName: "Third", JoinedAt: now},
	}
	rec, err := svc.StartRecording(ctx, nil, &meeting, "room-consent-gate", initiator, participants, tenant)
	if err != nil {
		t.Fatalf("StartRecording: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "recordings", rec.ID)

	requirePending := func(step string) {
		t.Helper()
		got, getErr := repo.GetRecording(ctx, rec.ID)
		if getErr != nil {
			t.Fatalf("%s: GetRecording: %v", step, getErr)
		}
		if got.Status != RecordingStatusPending || got.EgressID != nil {
			t.Fatalf("%s: status %q egress %v, want pending without egress", step, got.Status, got.EgressID)
		}
		if egress.startCalls != 0 {
			t.Fatalf("%s: %d egress starts, want 0", step, egress.startCalls)
		}
	}
	requirePending("after start")

	// Nobody answered yet.
	if err := svc.ConfirmInitiatorConsent(ctx, rec.ID, initiator, tenant); !errors.Is(err, ErrConsentPending) {
		t.Fatalf("confirm without answers: %v, want ErrConsentPending", err)
	}
	requirePending("confirm without answers")

	// One yes, one no: a no is an answer but not a consent.
	if err := svc.SetConsent(ctx, rec.ID, second, true); err != nil {
		t.Fatalf("SetConsent second: %v", err)
	}
	if err := svc.SetConsent(ctx, rec.ID, third, false); err != nil {
		t.Fatalf("SetConsent third (no): %v", err)
	}
	if err := svc.ConfirmInitiatorConsent(ctx, rec.ID, initiator, tenant); !errors.Is(err, ErrConsentPending) {
		t.Fatalf("confirm with a no: %v, want ErrConsentPending", err)
	}
	requirePending("confirm with a no")

	// Everyone else said yes; the initiator's own consent is the confirmation.
	if err := svc.SetConsent(ctx, rec.ID, third, true); err != nil {
		t.Fatalf("SetConsent third (yes): %v", err)
	}
	if err := svc.ConfirmInitiatorConsent(ctx, rec.ID, initiator, tenant); err != nil {
		t.Fatalf("confirm after all yes: %v", err)
	}
	if err := svc.ConfirmInitiatorConsent(ctx, rec.ID, initiator, tenant); err != nil {
		t.Fatalf("repeated confirm: %v", err)
	}
	if egress.startCalls != 1 || egress.lastRoom != "room-consent-gate" {
		t.Fatalf("egress starts %d on %q, want exactly one on room-consent-gate", egress.startCalls, egress.lastRoom)
	}
	got, err := repo.GetRecording(ctx, rec.ID)
	if err != nil {
		t.Fatalf("GetRecording after start: %v", err)
	}
	if got.Status != RecordingStatusActive || got.EgressID == nil {
		t.Fatalf("after confirm: status %q egress %v, want active with egress", got.Status, got.EgressID)
	}
}
