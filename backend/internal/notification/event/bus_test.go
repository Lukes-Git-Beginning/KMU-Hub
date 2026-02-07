package event

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
