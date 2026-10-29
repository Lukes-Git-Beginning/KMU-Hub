package message

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestCreate_RefusesARecipientOfAnotherTenant pins the recipient check in
// Create. The user_id foreign key alone accepted a user of any tenant (FK
// lookups bypass RLS), so an event naming a foreign user -- notification.send
// takes its target from a free template -- filed a message under the sender's
// tenant and pushed its subject to that user's WebSocket. The system-context
// case is the one that pins the tenant predicate itself: there RLS on users
// admits every row, so only "AND tenant_id = $2" can refuse the insert.
func TestCreate_RefusesARecipientOfAnotherTenant(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Inbox Recipient Own Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Inbox Recipient Other Tenant")
	userOwn := seedRecipient(t, pool, tenantOwn)
	userOther := seedRecipient(t, pool, tenantOther)

	repo := NewPostgresRepository(pool)
	newMsg := func(userID uuid.UUID) *models.InboxMessage {
		return &models.InboxMessage{
			ID: uuid.New(), TenantID: tenantOwn, UserID: userID, Channel: "notification",
			SourceID: uuid.NewString(), Subject: "Vertraulich", Tags: []string{}, ReceivedAt: time.Now().UTC(),
		}
	}
	sysCtx := testutil.WithSystemCtx(context.Background())

	for name, ctx := range map[string]context.Context{
		"tenant context": testutil.WithTenantCtx(context.Background(), tenantOwn),
		"system context": sysCtx,
	} {
		t.Run(name, func(t *testing.T) {
			foreign := newMsg(userOther)
			if err := repo.Create(ctx, foreign); !errors.Is(err, ErrRecipientNotInTenant) {
				t.Fatalf("Create for a foreign-tenant user: got %v, want ErrRecipientNotInTenant", err)
			}
			testutil.AssertRowCount(t, pool, sysCtx, "inbox_messages", foreign.ID, 0)

			own := newMsg(userOwn)
			if err := repo.Create(ctx, own); err != nil {
				t.Fatalf("Create for an own-tenant user: %v", err)
			}
			testutil.AssertRowCount(t, pool, sysCtx, "inbox_messages", own.ID, 1)
		})
	}
}

func seedRecipient(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("inbox-recipient-%s@tenant.local", uuid.NewString()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
	return id
}
