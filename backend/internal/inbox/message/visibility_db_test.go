package message

// Pins visibleToCaller against a real Postgres (as kmuhub_app under RLS): a message is visible to
// its owner, its assignee and the members of its team inbox; a colleague in the same tenant gets
// ErrMessageNotFound and their bulk writes skip the message.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestGetByID_VisibleToOwnerAssigneeAndTeamMemberOnly(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Inbox Visibility Tenant")

	seedUser := func(label string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "users", map[string]any{
			"tenant_id":     tenantID,
			"email":         fmt.Sprintf("inbox-vis-%s-%s@tenantown.local", label, uuid.New().String()[:8]),
			"password_hash": "$argon2id$v=19$test",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
		return id
	}
	owner, assignee, member, stranger := seedUser("owner"), seedUser("assignee"), seedUser("member"), seedUser("stranger")

	teamID := testutil.SeedRow(t, pool, "team_inboxes", map[string]any{
		"tenant_id":  tenantID,
		"name":       "Support " + uuid.New().String()[:8],
		"created_by": owner,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "team_inboxes", teamID) })
	sysCtx := testutil.WithSystemCtx(context.Background())
	if _, err := pool.Exec(sysCtx,
		"INSERT INTO team_inbox_members (team_inbox_id, user_id, role, tenant_id) VALUES ($1, $2, 'member', $3)",
		teamID, member, tenantID); err != nil {
		t.Fatalf("seed team member: %v", err)
	}

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	seed := func(mutate func(*models.InboxMessage)) *models.InboxMessage {
		msg := &models.InboxMessage{
			ID: uuid.New(), TenantID: tenantID, UserID: owner, Channel: "email",
			SourceID: "vis-" + uuid.New().String()[:8], Tags: []string{}, ReceivedAt: time.Now().UTC(),
		}
		if mutate != nil {
			mutate(msg)
		}
		if err := repo.Create(ctx, msg); err != nil {
			t.Fatalf("seed message: %v", err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "inbox_messages", msg.ID) })
		return msg
	}
	teamMsg := seed(func(m *models.InboxMessage) { m.TeamInboxID = &teamID; m.AssignedTo = &assignee })
	personal := seed(nil)

	for _, tc := range []struct {
		name    string
		msg     *models.InboxMessage
		caller  uuid.UUID
		visible bool
	}{
		{"owner", teamMsg, owner, true},
		{"assignee", teamMsg, assignee, true},
		{"team member", teamMsg, member, true},
		{"colleague outside the team", teamMsg, stranger, false},
		{"team member on a personal message", personal, member, false},
		{"colleague on a personal message", personal, stranger, false},
	} {
		_, err := repo.GetByID(ctx, tc.msg.ID, tenantID, tc.caller)
		switch {
		case tc.visible && err != nil:
			t.Errorf("%s: expected the message, got %v", tc.name, err)
		case !tc.visible && !errors.Is(err, ErrMessageNotFound):
			t.Errorf("%s: expected ErrMessageNotFound, got %v", tc.name, err)
		}
	}

	ids := []uuid.UUID{teamMsg.ID, personal.ID}
	if err := repo.BulkMarkRead(ctx, tenantID, stranger, ids); err != nil {
		t.Fatalf("BulkMarkRead (stranger): %v", err)
	}
	if err := repo.BulkArchive(ctx, tenantID, member, ids); err != nil {
		t.Fatalf("BulkArchive (member): %v", err)
	}
	gotTeam, err := repo.GetByID(ctx, teamMsg.ID, tenantID, owner)
	if err != nil {
		t.Fatalf("GetByID (team message): %v", err)
	}
	gotPersonal, err := repo.GetByID(ctx, personal.ID, tenantID, owner)
	if err != nil {
		t.Fatalf("GetByID (personal message): %v", err)
	}
	if gotTeam.IsRead || gotPersonal.IsRead {
		t.Errorf("a colleague's BulkMarkRead reached a message they cannot see")
	}
	if !gotTeam.IsArchived {
		t.Errorf("the team member's BulkArchive did not land on the team message")
	}
	if gotPersonal.IsArchived {
		t.Errorf("the team member's BulkArchive reached the owner's personal message")
	}
}
