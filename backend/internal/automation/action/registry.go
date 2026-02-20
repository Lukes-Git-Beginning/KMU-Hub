package action

import "sync"

// ActionRegistry holds all registered action executors and their definitions.
type ActionRegistry struct {
	mu          sync.RWMutex
	executors   map[string]ActionExecutor
	definitions map[string]*ActionDefinition
}

// NewActionRegistry creates a new empty ActionRegistry.
func NewActionRegistry() *ActionRegistry {
	return &ActionRegistry{
		executors:   make(map[string]ActionExecutor),
		definitions: make(map[string]*ActionDefinition),
	}
}

// Register adds an action executor and its definition to the registry.
func (r *ActionRegistry) Register(executor ActionExecutor, def *ActionDefinition) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.executors[executor.Type()] = executor
	r.definitions[executor.Type()] = def
}

// Get returns an action executor by type.
func (r *ActionRegistry) Get(actionType string) (ActionExecutor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	exec, ok := r.executors[actionType]
	return exec, ok
}

// AllDefinitions returns all registered action definitions.
func (r *ActionRegistry) AllDefinitions() []*ActionDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]*ActionDefinition, 0, len(r.definitions))
	for _, def := range r.definitions {
		defs = append(defs, def)
	}
	return defs
}

// Count returns the number of registered actions.
func (r *ActionRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.executors)
}
