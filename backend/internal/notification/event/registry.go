package event

import (
	"context"
	"sync"

	"github.com/kmuhub/kmuhub/internal/models"
)

// EventTypeRepository provides access to event types in the database.
type EventTypeRepository interface {
	ListAll(ctx context.Context) ([]models.EventType, error)
	GetByKey(ctx context.Context, key string) (*models.EventType, error)
}

// EventTypeRegistry is a thread-safe in-memory registry of event types.
// Modules register event types at startup; the registry can also be
// loaded from the database.
type EventTypeRegistry struct {
	mu       sync.RWMutex
	byKey    map[string]models.EventType
	byModule map[string][]models.EventType
}

// NewEventTypeRegistry creates a new empty event type registry.
func NewEventTypeRegistry() *EventTypeRegistry {
	return &EventTypeRegistry{
		byKey:    make(map[string]models.EventType),
		byModule: make(map[string][]models.EventType),
	}
}

// Register adds an event type to the registry. If an event type with the
// same key already exists, it is overwritten.
func (r *EventTypeRegistry) Register(et models.EventType) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Remove old entry from module list if overwriting
	if old, exists := r.byKey[et.EventKey]; exists {
		r.removeFromModule(old.ModuleID, old.EventKey)
	}

	r.byKey[et.EventKey] = et
	r.byModule[et.ModuleID] = append(r.byModule[et.ModuleID], et)
}

// Get retrieves an event type by its key. Returns nil if not found.
func (r *EventTypeRegistry) Get(eventKey string) *models.EventType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	et, ok := r.byKey[eventKey]
	if !ok {
		return nil
	}
	return &et
}

// ListByModule returns all event types for a given module.
func (r *EventTypeRegistry) ListByModule(moduleID string) []models.EventType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := r.byModule[moduleID]
	result := make([]models.EventType, len(types))
	copy(result, types)
	return result
}

// ListAll returns all registered event types.
func (r *EventTypeRegistry) ListAll() []models.EventType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]models.EventType, 0, len(r.byKey))
	for _, et := range r.byKey {
		result = append(result, et)
	}
	return result
}

// LoadFromDB loads all event types from the database into the registry.
func (r *EventTypeRegistry) LoadFromDB(ctx context.Context, repo EventTypeRepository) error {
	types, err := repo.ListAll(ctx)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.byKey = make(map[string]models.EventType, len(types))
	r.byModule = make(map[string][]models.EventType)

	for _, et := range types {
		r.byKey[et.EventKey] = et
		r.byModule[et.ModuleID] = append(r.byModule[et.ModuleID], et)
	}

	return nil
}

// removeFromModule removes an event type from the module list (must be called with lock held).
func (r *EventTypeRegistry) removeFromModule(moduleID, eventKey string) {
	types := r.byModule[moduleID]
	for i, t := range types {
		if t.EventKey == eventKey {
			r.byModule[moduleID] = append(types[:i], types[i+1:]...)
			return
		}
	}
}
