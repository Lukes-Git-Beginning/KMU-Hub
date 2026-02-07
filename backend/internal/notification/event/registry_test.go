package event

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

func makeEventType(moduleID, eventKey, displayName string) models.EventType {
	return models.EventType{
		ID:                 uuid.New(),
		ModuleID:           moduleID,
		EventKey:           eventKey,
		DisplayName:        displayName,
		DefaultPriority:    models.PriorityNormal,
		DefaultInApp:       true,
		DefaultDesktopPush: true,
		DefaultSound:       "default",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
}

func TestRegistryRegisterAndGet(t *testing.T) {
	r := NewEventTypeRegistry()

	et := makeEventType("chat", "chat.mention", "Mentioned in message")
	r.Register(et)

	got := r.Get("chat.mention")
	require.NotNil(t, got)
	assert.Equal(t, "chat.mention", got.EventKey)
	assert.Equal(t, "chat", got.ModuleID)
	assert.Equal(t, "Mentioned in message", got.DisplayName)
}

func TestRegistryGetNotFound(t *testing.T) {
	r := NewEventTypeRegistry()

	got := r.Get("nonexistent")
	assert.Nil(t, got)
}

func TestRegistryListByModule(t *testing.T) {
	r := NewEventTypeRegistry()

	r.Register(makeEventType("chat", "chat.mention", "Mention"))
	r.Register(makeEventType("chat", "chat.dm.new", "DM"))
	r.Register(makeEventType("crm", "crm.deal.assigned", "Deal Assigned"))

	chatTypes := r.ListByModule("chat")
	assert.Len(t, chatTypes, 2)

	crmTypes := r.ListByModule("crm")
	assert.Len(t, crmTypes, 1)

	emptyTypes := r.ListByModule("system")
	assert.Empty(t, emptyTypes)
}

func TestRegistryListAll(t *testing.T) {
	r := NewEventTypeRegistry()

	r.Register(makeEventType("chat", "chat.mention", "Mention"))
	r.Register(makeEventType("crm", "crm.deal.assigned", "Deal Assigned"))

	all := r.ListAll()
	assert.Len(t, all, 2)
}

func TestRegistryRegisterOverwrite(t *testing.T) {
	r := NewEventTypeRegistry()

	et1 := makeEventType("chat", "chat.mention", "Old Name")
	r.Register(et1)

	et2 := makeEventType("chat", "chat.mention", "New Name")
	r.Register(et2)

	got := r.Get("chat.mention")
	require.NotNil(t, got)
	assert.Equal(t, "New Name", got.DisplayName)

	// Should not duplicate in module list
	chatTypes := r.ListByModule("chat")
	assert.Len(t, chatTypes, 1)
}

func TestRegistryLoadFromDB(t *testing.T) {
	r := NewEventTypeRegistry()
	repo := &mockEventTypeRepo{
		types: []models.EventType{
			makeEventType("chat", "chat.mention", "Mention"),
			makeEventType("crm", "crm.deal.assigned", "Deal Assigned"),
		},
	}

	err := r.LoadFromDB(context.Background(), repo)
	require.NoError(t, err)

	assert.Len(t, r.ListAll(), 2)
	assert.NotNil(t, r.Get("chat.mention"))
	assert.NotNil(t, r.Get("crm.deal.assigned"))
}

func TestRegistryLoadFromDBError(t *testing.T) {
	r := NewEventTypeRegistry()
	repo := &mockEventTypeRepo{err: assert.AnError}

	err := r.LoadFromDB(context.Background(), repo)
	assert.Error(t, err)
}

func TestRegistryConcurrentAccess(t *testing.T) {
	r := NewEventTypeRegistry()
	var wg sync.WaitGroup

	// Concurrent writers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := "test.event." + string(rune('a'+idx%26))
			r.Register(makeEventType("test", key, "Test Event"))
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.Get("test.event.a")
			_ = r.ListByModule("test")
			_ = r.ListAll()
		}()
	}

	wg.Wait()
	// No panic = success for concurrent access test
}

// mockEventTypeRepo implements EventTypeRepository for testing
type mockEventTypeRepo struct {
	types []models.EventType
	err   error
}

func (m *mockEventTypeRepo) ListAll(_ context.Context) ([]models.EventType, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.types, nil
}

func (m *mockEventTypeRepo) GetByKey(_ context.Context, key string) (*models.EventType, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, et := range m.types {
		if et.EventKey == key {
			return &et, nil
		}
	}
	return nil, nil
}
