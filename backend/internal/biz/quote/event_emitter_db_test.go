package quote

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

// Creating a quote with the production emitter must put biz.quote.created on
// the bus carrying the caller's tenant — the automation trigger of the same
// name routes on exactly that event type and tenant.
func TestService_Create_PGEmitterPublishesTenantScopedEvent(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "quote-created-event")

	conn, err := pool.Acquire(context.Background())
	require.NoError(t, err)
	defer conn.Release()
	_, err = conn.Exec(context.Background(), "LISTEN events")
	require.NoError(t, err)

	repo := NewMockRepository()
	svc, _, _ := newTestService(repo)
	svc.SetEventEmitter(NewPGEventEmitter(pool))

	q, err := svc.Create(testutil.WithTenantCtx(context.Background(), tenantID), sampleCreateInput(tenantID))
	require.NoError(t, err)

	deadline, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		n, err := conn.Conn().WaitForNotification(deadline)
		require.NoError(t, err, "no biz.quote.created event arrived")
		var got models.EventPayload
		require.NoError(t, json.Unmarshal([]byte(n.Payload), &got))
		if got.ResourceID != q.ID.String() {
			continue // event of a parallel test on the shared channel
		}
		assert.Equal(t, "biz.quote.created", got.Type)
		assert.Equal(t, tenantID, got.TenantID)
		return
	}
}
