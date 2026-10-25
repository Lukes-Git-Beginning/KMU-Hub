package message

// Pins the team inbox reach of List against a real Postgres (as kmuhub_app under RLS): filtered to a
// team inbox, a member of that team sees all of its messages; everyone else, and every list without
// the team filter, keeps seeing only their own messages.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestList_TeamInboxFilterFollowsMembership(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Inbox Team List Tenant")

	seedUser := func(label string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "users", map[string]any{
			"tenant_id":     tenantID,
			"email":         fmt.Sprintf("inbox-list-%s-%s@tenantown.local", label, uuid.New().String()[:8]),
			"password_hash": "$argon2id$v=19$test",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
		return id
	}
	owner, member, stranger := seedUser("owner"), seedUser("member"), seedUser("stranger")

	sysCtx := testutil.WithSystemCtx(context.Background())
	seedTeam := func(members ...uuid.UUID) uuid.UUID {
		id := testutil.SeedRow(t, pool, "team_inboxes", map[string]any{
			"tenant_id":  tenantID,
			"name":       "Team " + uuid.New().String()[:8],
			"created_by": owner,
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "team_inboxes", id) })
		for _, u := range members {
			if _, err := pool.Exec(sysCtx,
				"INSERT INTO team_inbox_members (team_inbox_id, user_id, role, tenant_id) VALUES ($1, $2, 'member', $3)",
				id, u, tenantID); err != nil {
				t.Fatalf("seed team member: %v", err)
			}
		}
		return id
	}
	team, otherTeam := seedTeam(member), seedTeam(owner)

	repo := NewPostgresRepository(pool)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	seed := func(userID uuid.UUID, teamID *uuid.UUID) uuid.UUID {
		msg := &models.InboxMessage{
			ID: uuid.New(), TenantID: tenantID, UserID: userID, Channel: "email", TeamInboxID: teamID,
			SourceID: "list-" + uuid.New().String()[:8], Tags: []string{}, ReceivedAt: time.Now().UTC(),
		}
		if err := repo.Create(ctx, msg); err != nil {
			t.Fatalf("seed message: %v", err)
		}
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "inbox_messages", msg.ID) })
		return msg.ID
	}
	teamMsg := seed(owner, &team)
	strangerTeamMsg := seed(stranger, &team)
	seed(owner, &otherTeam)
	seed(owner, nil)

	list := func(caller uuid.UUID, teamID *uuid.UUID) ([]uuid.UUID, int) {
		t.Helper()
		msgs, total, err := repo.List(ctx, ListFilter{TenantID: tenantID, UserID: caller, TeamInboxID: teamID})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		ids := make([]uuid.UUID, 0, len(msgs))
		for _, m := range msgs {
			ids = append(ids, m.ID)
		}
		return ids, total
	}
	same := func(got []uuid.UUID, want ...uuid.UUID) bool {
		if len(got) != len(want) {
			return false
		}
		seen := make(map[uuid.UUID]bool, len(got))
		for _, id := range got {
			seen[id] = true
		}
		for _, id := range want {
			if !seen[id] {
				return false
			}
		}
		return true
	}

	for _, tc := range []struct {
		name   string
		caller uuid.UUID
		teamID *uuid.UUID
		want   []uuid.UUID
	}{
		{"team member sees every message of the team", member, &team, []uuid.UUID{teamMsg, strangerTeamMsg}},
		{"team member sees nothing of a team it is not in", member, &otherTeam, nil},
		{"non-member keeps only its own messages of the team", stranger, &team, []uuid.UUID{strangerTeamMsg}},
		{"without the team filter a member lists only its own messages", member, nil, nil},
	} {
		ids, total := list(tc.caller, tc.teamID)
		if !same(ids, tc.want...) {
			t.Errorf("%s: got %v, want %v", tc.name, ids, tc.want)
		}
		if total != len(tc.want) {
			t.Errorf("%s: total %d, want %d", tc.name, total, len(tc.want))
		}
	}
}
