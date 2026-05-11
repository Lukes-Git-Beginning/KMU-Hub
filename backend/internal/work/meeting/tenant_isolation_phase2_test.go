package meeting

// Cross-tenant isolation tests for meeting tables (Migration 000124).
// meetings, meeting_notes, meeting_action_items all have tenant_id NOT NULL after backfill.
// meeting_attendees is a composite-PK table (meeting_id, user_id) — covered by
// meetings isolation.
//
// meetings.organizer_id FK to users(id). meeting_notes.author_id FK to users(id).

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestTenantIsolation_Meeting(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	testutil.EnsureTenant(t, pool, testutil.TenantA, "Tenant A")
	testutil.EnsureTenant(t, pool, testutil.TenantB, "Tenant B")

	// Seed a user for TenantA (organizer + note author).
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     testutil.TenantA,
		"email":         fmt.Sprintf("meeting-test-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	now := time.Now().UTC()

	// meetings.
	meetingID := testutil.SeedRow(t, pool, "meetings", map[string]any{
		"tenant_id":       testutil.TenantA,
		"title":           "Sprint Planning",
		"organizer_id":    userID,
		"scheduled_start": now,
		"scheduled_end":   now.Add(time.Hour),
	})
	defer testutil.CleanupRow(t, pool, "meetings", meetingID)

	// meeting_notes.
	noteID := testutil.SeedRow(t, pool, "meeting_notes", map[string]any{
		"tenant_id":  testutil.TenantA,
		"meeting_id": meetingID,
		"author_id":  userID,
	})
	defer testutil.CleanupRow(t, pool, "meeting_notes", noteID)

	// meeting_action_items.
	actionID := testutil.SeedRow(t, pool, "meeting_action_items", map[string]any{
		"tenant_id":   testutil.TenantA,
		"meeting_id":  meetingID,
		"description": "Follow up with client",
	})
	defer testutil.CleanupRow(t, pool, "meeting_action_items", actionID)

	ctxA := testutil.WithTenantCtx(context.Background(), testutil.TenantA)
	ctxB := testutil.WithTenantCtx(context.Background(), testutil.TenantB)

	tests := []struct {
		name  string
		table string
		id    uuid.UUID
	}{
		{"meetings", "meetings", meetingID},
		{"meeting_notes", "meeting_notes", noteID},
		{"meeting_action_items", "meeting_action_items", actionID},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testutil.AssertRowCount(t, pool, ctxA, tc.table, tc.id, 1)
			testutil.AssertRowCount(t, pool, ctxB, tc.table, tc.id, 0)
		})
	}
}
