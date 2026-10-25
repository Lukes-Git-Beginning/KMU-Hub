package event

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestRequireParticipant_OnlyOrganiserAndAttendees runs the guard behind the
// video join token against the real attendee table: organiser and invited
// user pass, a colleague of the same tenant gets the same ErrEventNotFound as
// for a missing event.
func TestRequireParticipant_OnlyOrganiserAndAttendees(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID, organiser, calID := seedEventFixtureTenant(t, pool, "Join Token Tenant")
	defer testutil.CleanupRow(t, pool, "tenants", tenantID)
	defer testutil.CleanupRow(t, pool, "users", organiser)
	defer testutil.CleanupRow(t, pool, "calendars", calID)
	attendee := seedEventUser(t, pool, tenantID)
	defer testutil.CleanupRow(t, pool, "users", attendee)
	colleague := seedEventUser(t, pool, tenantID)
	defer testutil.CleanupRow(t, pool, "users", colleague)

	repo := NewPostgresRepository(pool)
	svc := NewService(repo, nil)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	ev := newTestEvent(tenantID, calID, organiser, "Join-Token-Test", time.Date(2031, 5, 1, 9, 0, 0, 0, time.UTC))
	if err := repo.Create(ctx, ev); err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "calendar_events", ev.ID)
	if err := repo.AddAttendee(ctx, &models.EventAttendee{
		EventID: ev.ID, UserID: attendee, RSVPStatus: "pending", CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("AddAttendee: %v", err)
	}

	for _, tc := range []struct {
		name string
		user uuid.UUID
	}{{"organiser", organiser}, {"attendee", attendee}} {
		got, err := svc.RequireParticipant(ctx, ev.ID, tc.user, tenantID)
		if err != nil {
			t.Fatalf("%s: expected access, got %v", tc.name, err)
		}
		if got.ID != ev.ID {
			t.Fatalf("%s: got event %s, want %s", tc.name, got.ID, ev.ID)
		}
	}

	if _, err := svc.RequireParticipant(ctx, ev.ID, colleague, tenantID); !errors.Is(err, ErrEventNotFound) {
		t.Fatalf("colleague: expected ErrEventNotFound, got %v", err)
	}
	if _, err := svc.RequireParticipant(ctx, uuid.New(), organiser, tenantID); !errors.Is(err, ErrEventNotFound) {
		t.Fatalf("missing event: expected ErrEventNotFound, got %v", err)
	}
}
