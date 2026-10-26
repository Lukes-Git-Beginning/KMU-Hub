package channel

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// Group DMs against the real schema (000364) and RLS, as kmuhub_app: the
// member set is the identity, non-members do not see the channel, and another
// tenant neither sees it nor can pull its users into one.
func TestGroupDM_IdentityAndVisibility(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })

	tenant, otherTenant := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Group DM Tenant")
	testutil.EnsureTenant(t, pool, otherTenant, "Group DM Other Tenant")
	seedUser := func(tenantID uuid.UUID, label string) uuid.UUID {
		id := testutil.SeedRow(t, pool, "users", map[string]any{
			"tenant_id":     tenantID,
			"email":         fmt.Sprintf("group-dm-%s-%s@tenant.local", label, uuid.New().String()[:8]),
			"password_hash": "$argon2id$v=19$test",
			"first_name":    label,
			"last_name":     "Group",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
		return id
	}
	a, b, c, outsider := seedUser(tenant, "a"), seedUser(tenant, "b"), seedUser(tenant, "c"), seedUser(tenant, "outsider")
	foreign := seedUser(otherTenant, "foreign")

	repo := NewPostgresRepository(pool)
	svc := NewService(repo)
	ctxOwn := testutil.WithTenantCtx(context.Background(), tenant)
	ctxOther := testutil.WithTenantCtx(context.Background(), otherTenant)
	cleanupChannel := func(id uuid.UUID) {
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "channels", id) })
	}

	first, err := svc.GetOrCreateGroupDM(ctxOwn, GetOrCreateGroupDMInput{TenantID: tenant, UserID: a, ParticipantIDs: []uuid.UUID{b, c}})
	if err != nil {
		t.Fatalf("create group DM: %v", err)
	}
	cleanupChannel(first.Channel.ID)
	if !first.Created || !first.Channel.IsDM || first.Channel.MemberCount != 3 {
		t.Fatalf("expected a new DM with 3 members, got created=%v is_dm=%v members=%d",
			first.Created, first.Channel.IsDM, first.Channel.MemberCount)
	}

	again, err := svc.GetOrCreateGroupDM(ctxOwn, GetOrCreateGroupDMInput{TenantID: tenant, UserID: c, ParticipantIDs: []uuid.UUID{a, b}})
	if err != nil {
		t.Fatalf("same set again: %v", err)
	}
	if again.Created || again.Channel.ID != first.Channel.ID {
		t.Fatalf("same member set must return channel %s, got %s (created=%v)", first.Channel.ID, again.Channel.ID, again.Created)
	}

	other, err := svc.GetOrCreateGroupDM(ctxOwn, GetOrCreateGroupDMInput{TenantID: tenant, UserID: a, ParticipantIDs: []uuid.UUID{b, c, outsider}})
	if err != nil {
		t.Fatalf("different set: %v", err)
	}
	cleanupChannel(other.Channel.ID)
	if !other.Created || other.Channel.ID == first.Channel.ID {
		t.Fatalf("a different member set must create a new channel, got %s (created=%v)", other.Channel.ID, other.Created)
	}

	// The unique index catches a second insert of the same set.
	now := time.Now().UTC()
	dup := &models.Channel{ID: uuid.New(), TenantID: tenant, Name: "DM", IsPrivate: true, IsDM: true, CreatedBy: a, CreatedAt: now, UpdatedAt: now}
	dupMember := &models.ChannelMembership{ChannelID: dup.ID, UserID: a, TenantID: tenant, Role: models.ChannelRoleMember, JoinedAt: now}
	if err := repo.CreateGroupDMChannel(ctxOwn, dup, groupDMKey([]uuid.UUID{a, b, c}), []*models.ChannelMembership{dupMember}); !errors.Is(err, ErrGroupDMExists) {
		t.Fatalf("duplicate member set: expected ErrGroupDMExists, got %v", err)
	}
	testutil.AssertRowCount(t, pool, ctxOwn, "channels", dup.ID, 0)

	// A tenant member outside the set does not see it.
	if _, err := svc.GetByID(ctxOwn, first.Channel.ID, outsider, tenant); !errors.Is(err, ErrNotChannelMember) {
		t.Fatalf("non-member GetByID: expected ErrNotChannelMember, got %v", err)
	}
	dms, _, err := svc.ListDMs(ctxOwn, outsider, tenant, 1, 50)
	if err != nil {
		t.Fatalf("ListDMs outsider: %v", err)
	}
	for _, dm := range dms {
		if dm.ID == first.Channel.ID {
			t.Fatalf("non-member sees group DM %s in ListDMs", first.Channel.ID)
		}
	}
	memberDMs, _, err := svc.ListDMs(ctxOwn, b, tenant, 1, 50)
	if err != nil {
		t.Fatalf("ListDMs member: %v", err)
	}
	found := false
	for _, dm := range memberDMs {
		found = found || dm.ID == first.Channel.ID
	}
	if !found {
		t.Fatalf("member b does not see group DM %s in ListDMs", first.Channel.ID)
	}

	// Another tenant sees neither the channel nor its key.
	testutil.AssertRowCount(t, pool, ctxOther, "channels", first.Channel.ID, 0)
	if _, err := repo.FindGroupDMChannel(ctxOther, groupDMKey([]uuid.UUID{a, b, c}), otherTenant); !errors.Is(err, ErrChannelNotFound) {
		t.Fatalf("foreign tenant FindGroupDMChannel: expected ErrChannelNotFound, got %v", err)
	}

	// A user of another tenant cannot be pulled into a group DM.
	if _, err := svc.GetOrCreateGroupDM(ctxOwn, GetOrCreateGroupDMInput{TenantID: tenant, UserID: a, ParticipantIDs: []uuid.UUID{b, foreign}}); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("foreign participant: expected ErrUserNotFound, got %v", err)
	}
}

// The CHECK constraint keeps the two DM shapes apart: a group key never sits
// next to a user pair, and a non-DM channel never carries a key.
func TestGroupDM_CheckConstraint(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(func() { pool.Close() })
	fx := newReadFixture(t, pool)
	ctxOwn := testutil.WithTenantCtx(context.Background(), fx.tenant)

	user1, user2 := fx.userA, fx.userB
	if user1.String() > user2.String() {
		user1, user2 = user2, user1
	}
	for name, insert := range map[string]func(*pgxpool.Pool) error{
		"pair plus key": func(p *pgxpool.Pool) error {
			_, err := p.Exec(ctxOwn, `INSERT INTO channels (tenant_id, name, is_private, is_dm, created_by, dm_user1, dm_user2, dm_group_key)
				VALUES ($1, 'DM', TRUE, TRUE, $2, $3, $4, 'k')`, fx.tenant, fx.userA, user1, user2)
			return err
		},
		"key on a normal channel": func(p *pgxpool.Pool) error {
			_, err := p.Exec(ctxOwn, `INSERT INTO channels (tenant_id, name, is_private, is_dm, created_by, dm_group_key)
				VALUES ($1, 'x', FALSE, FALSE, $2, 'k')`, fx.tenant, fx.userA)
			return err
		},
		"dm without pair or key": func(p *pgxpool.Pool) error {
			_, err := p.Exec(ctxOwn, `INSERT INTO channels (tenant_id, name, is_private, is_dm, created_by)
				VALUES ($1, 'DM', TRUE, TRUE, $2)`, fx.tenant, fx.userA)
			return err
		},
	} {
		var pgErr *pgconn.PgError
		if err := insert(pool); !errors.As(err, &pgErr) || pgErr.ConstraintName != "chk_dm_users_set" {
			t.Fatalf("%s: expected chk_dm_users_set to reject the row, got %v", name, err)
		}
	}
}
