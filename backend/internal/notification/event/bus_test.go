package event

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
)

func TestEventBusRegisterHandler(t *testing.T) {
	bus := NewEventBus("postgres://test")

	handler := func(_ context.Context, _ models.EventPayload) error { return nil }

	bus.RegisterHandler("chat.mention", handler)
	bus.RegisterHandler("crm.deal.assigned", handler)
	bus.RegisterHandler("*", handler)

	assert.Equal(t, 3, bus.HandlerCount())
}

func TestEventBusDispatchSpecificHandler(t *testing.T) {
	bus := NewEventBus("postgres://test")

	var called atomic.Bool
	bus.RegisterHandler("chat.mention", func(_ context.Context, event models.EventPayload) error {
		called.Store(true)
		assert.Equal(t, "chat.mention", event.Type)
		assert.Equal(t, "chat", event.ModuleID)
		return nil
	})

	event := models.EventPayload{
		Type:      "chat.mention",
		Priority:  "normal",
		ModuleID:  "chat",
		Timestamp: time.Now(),
	}

	bus.dispatch(context.Background(), event)
	assert.True(t, called.Load())
}

func TestEventBusDispatchWildcardHandler(t *testing.T) {
	bus := NewEventBus("postgres://test")

	var events []string
	var mu sync.Mutex

	bus.RegisterHandler("*", func(_ context.Context, event models.EventPayload) error {
		mu.Lock()
		events = append(events, event.Type)
		mu.Unlock()
		return nil
	})

	bus.dispatch(context.Background(), models.EventPayload{
		Type: "chat.mention", ModuleID: "chat", Timestamp: time.Now(),
	})
	bus.dispatch(context.Background(), models.EventPayload{
		Type: "crm.deal.assigned", ModuleID: "crm", Timestamp: time.Now(),
	})

	assert.Len(t, events, 2)
	assert.Contains(t, events, "chat.mention")
	assert.Contains(t, events, "crm.deal.assigned")
}

func TestEventBusDispatchBothSpecificAndWildcard(t *testing.T) {
	bus := NewEventBus("postgres://test")

	var callCount atomic.Int32

	bus.RegisterHandler("chat.mention", func(_ context.Context, _ models.EventPayload) error {
		callCount.Add(1)
		return nil
	})
	bus.RegisterHandler("*", func(_ context.Context, _ models.EventPayload) error {
		callCount.Add(1)
		return nil
	})

	bus.dispatch(context.Background(), models.EventPayload{
		Type: "chat.mention", ModuleID: "chat", Timestamp: time.Now(),
	})

	assert.Equal(t, int32(2), callCount.Load())
}

func TestEventBusDispatchNoMatchingHandler(t *testing.T) {
	bus := NewEventBus("postgres://test")

	bus.RegisterHandler("chat.mention", func(_ context.Context, _ models.EventPayload) error {
		t.Fatal("should not be called")
		return nil
	})

	// Should not panic or call handler
	bus.dispatch(context.Background(), models.EventPayload{
		Type: "crm.deal.assigned", ModuleID: "crm", Timestamp: time.Now(),
	})
}

func TestEventBusDispatchHandlerError(t *testing.T) {
	bus := NewEventBus("postgres://test")

	var secondCalled atomic.Bool

	bus.RegisterHandler("chat.mention", func(_ context.Context, _ models.EventPayload) error {
		return assert.AnError
	})
	bus.RegisterHandler("chat.mention", func(_ context.Context, _ models.EventPayload) error {
		secondCalled.Store(true)
		return nil
	})

	// Error in first handler should not prevent second from running
	bus.dispatch(context.Background(), models.EventPayload{
		Type: "chat.mention", ModuleID: "chat", Timestamp: time.Now(),
	})

	assert.True(t, secondCalled.Load())
}

func TestEventBusConcurrentDispatch(t *testing.T) {
	bus := NewEventBus("postgres://test")

	var callCount atomic.Int32

	bus.RegisterHandler("*", func(_ context.Context, _ models.EventPayload) error {
		callCount.Add(1)
		return nil
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.dispatch(context.Background(), models.EventPayload{
				Type: "test.event", ModuleID: "test", Timestamp: time.Now(),
			})
		}()
	}

	wg.Wait()
	assert.Equal(t, int32(100), callCount.Load())
}

func TestEventBusConcurrentRegisterAndDispatch(t *testing.T) {
	bus := NewEventBus("postgres://test")

	var wg sync.WaitGroup

	// Concurrent registration
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.RegisterHandler("test.event", func(_ context.Context, _ models.EventPayload) error {
				return nil
			})
		}()
	}

	// Concurrent dispatch
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.dispatch(context.Background(), models.EventPayload{
				Type: "test.event", ModuleID: "test", Timestamp: time.Now(),
			})
		}()
	}

	wg.Wait()
	// No panic = success
}

func TestEventBusWithReconnectWait(t *testing.T) {
	bus := NewEventBus("postgres://test", WithReconnectWait(10*time.Second))
	assert.Equal(t, 10*time.Second, bus.reconnectWait)
}

func TestEventBusProcessBacklog(t *testing.T) {
	bus := NewEventBus("postgres://test")

	var processed []string
	var mu sync.Mutex

	bus.RegisterHandler("*", func(_ context.Context, event models.EventPayload) error {
		mu.Lock()
		processed = append(processed, event.Type)
		mu.Unlock()
		return nil
	})

	repo := &mockEventRepo{
		events: []models.Event{
			{
				EventTypeKey: "chat.mention",
				ModuleID:     "chat",
				Priority:     "normal",
				CreatedAt:    time.Now(),
			},
			{
				EventTypeKey: "crm.deal.assigned",
				ModuleID:     "crm",
				Priority:     "normal",
				CreatedAt:    time.Now(),
			},
		},
	}

	err := bus.ProcessBacklog(context.Background(), repo)
	require.NoError(t, err)

	assert.Len(t, processed, 2)
	assert.Contains(t, processed, "chat.mention")
	assert.Contains(t, processed, "crm.deal.assigned")
	assert.Equal(t, 2, repo.markProcessedCount)
}

func TestEventBusProcessBacklogEmpty(t *testing.T) {
	bus := NewEventBus("postgres://test")

	repo := &mockEventRepo{events: []models.Event{}}
	err := bus.ProcessBacklog(context.Background(), repo)
	require.NoError(t, err)
}

func TestEventBusProcessBacklogRepoError(t *testing.T) {
	bus := NewEventBus("postgres://test")

	repo := &mockEventRepo{err: assert.AnError}
	err := bus.ProcessBacklog(context.Background(), repo)
	assert.Error(t, err)
}

// mockEventRepo implements EventRepository for testing
type mockEventRepo struct {
	events             []models.Event
	err                error
	markProcessedCount int
}

func (m *mockEventRepo) CreateEvent(_ context.Context, _ *models.Event) error {
	return m.err
}

func (m *mockEventRepo) ListUnprocessed(_ context.Context, _ int) ([]models.Event, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.events, nil
}

func (m *mockEventRepo) MarkProcessed(_ context.Context, _ string) error {
	m.markProcessedCount++
	return m.err
}

// dispatch must put the event's tenant on the context before any handler runs.
// Handlers execute on the listener's background context, which carries no
// tenant, and every table they write is under RLS — an unstamped context makes
// the policy admit nothing and the notification insert fails with "new row
// violates row-level security policy", which ProcessEvent logs and swallows.
func TestEventBusDispatchStampsTenantContext(t *testing.T) {
	bus := NewEventBus("postgres://test")
	tenantID := uuid.New()

	var got string
	var ok bool
	bus.RegisterHandler("*", func(ctx context.Context, _ models.EventPayload) error {
		got, ok = ctx.Value(middleware.TenantIDKey).(string)
		return nil
	})

	bus.dispatch(context.Background(), models.EventPayload{
		Type:     "crm.deal.assigned",
		TenantID: tenantID,
		ModuleID: "crm",
	})

	require.True(t, ok, "handler context carries no tenant — RLS will reject every write it makes")
	assert.Equal(t, tenantID.String(), got)
}

// An event without a tenant must not overwrite one already on the context.
// ProcessBacklog replays rows whose tenant_id is NOT NULL since 000271, but a
// caller may still dispatch a zero value; stamping it would be worse than
// leaving the context alone.
func TestEventBusDispatchLeavesContextAloneWithoutTenant(t *testing.T) {
	bus := NewEventBus("postgres://test")
	existing := uuid.New()

	var got string
	bus.RegisterHandler("*", func(ctx context.Context, _ models.EventPayload) error {
		got, _ = ctx.Value(middleware.TenantIDKey).(string)
		return nil
	})

	ctx := context.WithValue(context.Background(), middleware.TenantIDKey, existing.String())
	bus.dispatch(ctx, models.EventPayload{Type: "crm.deal.assigned", ModuleID: "crm"})

	assert.Equal(t, existing.String(), got)
}

// ProcessBacklog rebuilds the payload from the stored row. Before 000271 the
// tenant could not survive that round trip, so every event replayed after a
// restart reached preference.Evaluate as uuid.Nil.
func TestEventBusProcessBacklogRestoresTenant(t *testing.T) {
	bus := NewEventBus("postgres://test")
	tenantID := uuid.New()

	var seen uuid.UUID
	bus.RegisterHandler("*", func(_ context.Context, event models.EventPayload) error {
		seen = event.TenantID
		return nil
	})

	repo := &mockEventRepo{events: []models.Event{{
		ID:           uuid.New(),
		TenantID:     tenantID,
		EventTypeKey: "crm.deal.assigned",
		ModuleID:     "crm",
		Priority:     "normal",
		CreatedAt:    time.Now(),
	}}}

	require.NoError(t, bus.ProcessBacklog(context.Background(), repo))
	assert.Equal(t, tenantID, seen)
}
