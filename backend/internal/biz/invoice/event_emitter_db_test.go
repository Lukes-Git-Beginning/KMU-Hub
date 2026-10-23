package invoice

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// Sending an invoice with the production emitter must put biz.invoice.sent on
// the bus carrying the caller's tenant — the automation trigger of the same
// name routes on exactly that event type and tenant.
func TestService_Send_PGEmitterPublishesTenantScopedEvent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "invoice-sent-event")

	conn, err := pool.Acquire(context.Background())
	require.NoError(t, err)
	defer conn.Release()
	_, err = conn.Exec(context.Background(), "LISTEN events")
	require.NoError(t, err)

	svc, repo, _, _, _ := newTestService()
	svc.SetEventEmitter(NewPGEventEmitter(pool))
	inv := createDraftInvoice(t, repo, tenantID)

	require.NoError(t, svc.Send(testutil.WithTenantCtx(context.Background(), tenantID), tenantID, inv.ID, uuid.New()))

	deadline, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		n, err := conn.Conn().WaitForNotification(deadline)
		require.NoError(t, err, "no biz.invoice.sent event arrived")
		var got models.EventPayload
		require.NoError(t, json.Unmarshal([]byte(n.Payload), &got))
		if got.ResourceID != inv.ID.String() {
			continue // event of a parallel test on the shared channel
		}
		assert.Equal(t, "biz.invoice.sent", got.Type)
		assert.Equal(t, tenantID, got.TenantID)
		return
	}
}
