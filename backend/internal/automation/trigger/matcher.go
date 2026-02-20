package trigger

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/kmuhub/kmuhub/internal/automation/workflow"
	"github.com/kmuhub/kmuhub/internal/models"
)

// TriggerMatcher finds active automations that match a given event.
// It maintains a cache of active automations that is refreshed periodically.
type TriggerMatcher struct {
	workflowRepo workflow.Repository
	registry     *TriggerRegistry

	mu        sync.RWMutex
	cache     []*models.Automation
	cacheTime time.Time
	cacheTTL  time.Duration
}

// NewTriggerMatcher creates a new TriggerMatcher.
func NewTriggerMatcher(repo workflow.Repository, registry *TriggerRegistry) *TriggerMatcher {
	return &TriggerMatcher{
		workflowRepo: repo,
		registry:     registry,
		cacheTTL:     30 * time.Second,
	}
}

// FindMatching returns all active automations whose trigger_type matches the event type.
func (m *TriggerMatcher) FindMatching(ctx context.Context, eventType string) ([]*TriggerMatch, error) {
	automations, err := m.getActiveAutomations(ctx)
	if err != nil {
		return nil, err
	}

	var matches []*TriggerMatch
	for _, auto := range automations {
		// Check if the automation's trigger type maps to this event type
		triggerDef, ok := m.registry.Get(auto.TriggerType)
		if !ok {
			continue
		}

		if triggerDef.EventType == eventType {
			matches = append(matches, &TriggerMatch{
				Automation: auto,
				TriggerDef: triggerDef,
			})
		}
	}

	return matches, nil
}

// InvalidateCache forces a refresh of the automation cache on next access.
func (m *TriggerMatcher) InvalidateCache() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cacheTime = time.Time{}
}

// getActiveAutomations returns cached active automations, refreshing if stale.
func (m *TriggerMatcher) getActiveAutomations(ctx context.Context) ([]*models.Automation, error) {
	m.mu.RLock()
	if time.Since(m.cacheTime) < m.cacheTTL && m.cache != nil {
		automations := m.cache
		m.mu.RUnlock()
		return automations, nil
	}
	m.mu.RUnlock()

	// Refresh cache
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if time.Since(m.cacheTime) < m.cacheTTL && m.cache != nil {
		return m.cache, nil
	}

	isActive := true
	automations, _, err := m.workflowRepo.List(ctx, workflow.ListFilter{
		IsActive: &isActive,
		Limit:    10000, // Load all active automations
	})
	if err != nil {
		slog.Error("failed to refresh automation cache", "error", err)
		// Return stale cache if available
		if m.cache != nil {
			return m.cache, nil
		}
		return nil, err
	}

	m.cache = automations
	m.cacheTime = time.Now()
	slog.Debug("automation cache refreshed", "count", len(automations))

	return automations, nil
}
