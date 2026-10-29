package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/inbox/adapter"
	"github.com/kmuhub/kmuhub/internal/inbox/message"
	"github.com/kmuhub/kmuhub/internal/inbox/routing"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/event"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// TestInboxConsumer_DeliversAutomationNotification runs the event the
// notification.send action publishes through the consumer against the real
// database as kmuhub_app, on a context stamped the way EventBus.dispatch
// stamps it. The target of the event's tenant gets an inbox message carrying
// title, body and deep link; a user of another tenant named in the same event
// gets nothing.
func TestInboxConsumer_DeliversAutomationNotification(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantOwn, tenantOther := uuid.New(), uuid.New()
	testutil.EnsureTenant(t, pool, tenantOwn, "Inbox Consumer Own Tenant")
	testutil.EnsureTenant(t, pool, tenantOther, "Inbox Consumer Other Tenant")
	seedUser := func(tenantID uuid.UUID) uuid.UUID {
		id := testutil.SeedRow(t, pool, "users", map[string]any{
			"tenant_id":     tenantID,
			"email":         fmt.Sprintf("inbox-consumer-%s@tenant.local", uuid.NewString()[:8]),
			"password_hash": "$argon2id$v=19$test",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
		return id
	}
	target, foreign := seedUser(tenantOwn), seedUser(tenantOther)

	msgRepo := message.NewPostgresRepository(pool)
	consumer := &InboxConsumer{
		messageService: message.NewService(msgRepo, adapter.NewAdapterRegistry()),
		messageRepo:    msgRepo,
		routingService: routing.NewService(routing.NewPostgresRepository(pool), msgRepo, nil),
		adapters:       adapter.NewAdapterRegistry(),
	}

	evt := models.EventPayload{
		Type:          "automation.notification",
		TenantID:      tenantOwn,
		Priority:      models.PriorityNormal,
		ModuleID:      event.ModuleAutomation,
		ResourceID:    uuid.NewString(),
		TargetUserIDs: []string{target.String(), foreign.String()},
		Title:         "Deal Acme gewonnen",
		Body:          "Bitte Rechnung anlegen",
		DeepLink:      "/crm/deals/42",
		Timestamp:     time.Now().UTC(),
	}
	if err := consumer.HandleEvent(testutil.WithTenantCtx(context.Background(), tenantOwn), evt); err != nil {
		t.Fatalf("HandleEvent: %v", err)
	}

	sysCtx := testutil.WithSystemCtx(context.Background())
	var tenantID uuid.UUID
	var channel, subject, preview, deepLink string
	if err := pool.QueryRow(sysCtx,
		`SELECT tenant_id, channel, subject, preview, deep_link FROM inbox_messages WHERE user_id = $1 AND source_id = $2`,
		target, evt.ResourceID).Scan(&tenantID, &channel, &subject, &preview, &deepLink); err != nil {
		t.Fatalf("inbox message for the target: %v", err)
	}
	if tenantID != tenantOwn || channel != "notification" || subject != evt.Title || preview != evt.Body || deepLink != evt.DeepLink {
		t.Fatalf("inbox message = (%s, %q, %q, %q, %q), want the event's tenant, notification channel, title, body and deep link",
			tenantID, channel, subject, preview, deepLink)
	}

	var foreignRows int
	if err := pool.QueryRow(sysCtx, `SELECT count(*) FROM inbox_messages WHERE user_id = $1`, foreign).Scan(&foreignRows); err != nil {
		t.Fatalf("count foreign: %v", err)
	}
	if foreignRows != 0 {
		t.Fatalf("a user of another tenant got %d inbox message(s) from this tenant's event", foreignRows)
	}
}
