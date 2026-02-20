package integration

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/preference"
)

// PlatformPoster is the interface that Teams and Slack clients must implement.
type PlatformPoster interface {
	PostNotification(ctx context.Context, mapping *ChannelMapping, notif *models.Notification, actions []ActionType) (*DeliveryResult, error)
}

// Forwarder dispatches notifications to external platforms based on channel mappings.
// It is registered as a DeliveryCallback on the notification dispatcher.
type Forwarder struct {
	repo    Repository
	teams   PlatformPoster // nil if Teams not configured
	slack   PlatformPoster // nil if Slack not configured
	limiter *RateLimiter
	cache   *MappingCache

	// Failure tracking: mapping_id -> consecutive failure count
	failureMu sync.Mutex
	failures  map[uuid.UUID]int
}

// NewForwarder creates a new notification forwarder.
func NewForwarder(repo Repository, teams PlatformPoster, slack PlatformPoster, limiter *RateLimiter) *Forwarder {
	return &Forwarder{
		repo:     repo,
		teams:    teams,
		slack:    slack,
		limiter:  limiter,
		cache:    NewMappingCache(repo, 30*time.Second),
		failures: make(map[uuid.UUID]int),
	}
}

// HandleNotification is the DeliveryCallback signature. It forwards notifications
// to configured external platform channels based on module-level routing.
func (f *Forwarder) HandleNotification(ctx context.Context, notif *models.Notification, _ *preference.DeliveryDecision) {
	mappings, err := f.cache.GetMappingsForModule(ctx, notif.ModuleID)
	if err != nil {
		slog.Error("forwarder: failed to get channel mappings",
			"module_id", notif.ModuleID,
			"error", err,
		)
		return
	}

	if len(mappings) == 0 {
		return
	}

	// Most-specific-wins: select best matching mappings
	selected := selectMostSpecific(mappings, notif.ModuleID)
	if len(selected) == 0 {
		return
	}

	// Determine context-dependent action buttons based on event type
	actions := buildActionSet(notif.EventTypeKey)

	// Dispatch to each selected mapping
	for _, mapping := range selected {
		m := mapping // capture for goroutine
		go f.dispatchToMapping(ctx, m, notif, actions)
	}
}

// dispatchToMapping sends a notification to a single channel mapping.
func (f *Forwarder) dispatchToMapping(ctx context.Context, mapping *ChannelMapping, notif *models.Notification, actions []ActionType) {
	// Determine platform from config
	platform := f.getPlatformForMapping(ctx, mapping)
	if platform == "" {
		return
	}

	// Check rate limiter
	if !f.limiter.Allow(platform, mapping.ChannelID) {
		slog.Warn("forwarder: rate limited",
			"platform", platform,
			"channel_id", mapping.ChannelID,
			"notification_id", notif.ID,
		)
		f.logDelivery(ctx, notif.ID, mapping.ID, platform, DeliveryStatusRateLimited, "", "rate limited")
		return
	}

	// Dispatch to appropriate platform client
	var result *DeliveryResult
	var err error

	switch platform {
	case PlatformTeams:
		if f.teams == nil {
			return
		}
		result, err = f.teams.PostNotification(ctx, mapping, notif, actions)
	case PlatformSlack:
		if f.slack == nil {
			return
		}
		result, err = f.slack.PostNotification(ctx, mapping, notif, actions)
	default:
		slog.Warn("forwarder: unknown platform", "platform", platform)
		return
	}

	if err != nil {
		slog.Error("forwarder: delivery failed",
			"platform", platform,
			"channel_id", mapping.ChannelID,
			"notification_id", notif.ID,
			"error", err,
		)
		f.logDelivery(ctx, notif.ID, mapping.ID, platform, DeliveryStatusFailed, "", err.Error())
		f.trackFailure(ctx, mapping)
		return
	}

	// Reset failure counter on success
	f.resetFailures(mapping.ID)

	platformMsgID := ""
	if result != nil {
		platformMsgID = result.PlatformMessageID
	}
	f.logDelivery(ctx, notif.ID, mapping.ID, platform, DeliveryStatusSent, platformMsgID, "")
}

// getPlatformForMapping resolves the platform string for a channel mapping
// by looking up the parent integration config.
func (f *Forwarder) getPlatformForMapping(ctx context.Context, mapping *ChannelMapping) string {
	cfg, err := f.cache.GetConfigForMapping(ctx, mapping.ConfigID)
	if err != nil {
		slog.Error("forwarder: failed to get config for mapping",
			"config_id", mapping.ConfigID,
			"error", err,
		)
		return ""
	}
	return cfg.Platform
}

// logDelivery writes an entry to the integration delivery log.
func (f *Forwarder) logDelivery(ctx context.Context, notifID, mappingID uuid.UUID, platform, status, platformMsgID, errorMsg string) {
	entry := &DeliveryLogEntry{
		ID:                uuid.New(),
		NotificationID:    notifID,
		MappingID:         mappingID,
		Platform:          platform,
		Status:            status,
		PlatformMessageID: platformMsgID,
		ErrorMessage:      errorMsg,
		CreatedAt:         time.Now(),
	}
	if err := f.repo.LogDelivery(ctx, entry); err != nil {
		slog.Error("forwarder: failed to log delivery",
			"notification_id", notifID,
			"error", err,
		)
	}
}

// trackFailure increments the consecutive failure count for a mapping.
// If >10 consecutive failures, auto-disable the mapping.
func (f *Forwarder) trackFailure(ctx context.Context, mapping *ChannelMapping) {
	f.failureMu.Lock()
	f.failures[mapping.ID]++
	count := f.failures[mapping.ID]
	f.failureMu.Unlock()

	if count > 10 {
		slog.Error("forwarder: auto-disabling channel mapping after consecutive failures",
			"mapping_id", mapping.ID,
			"channel_name", mapping.ChannelName,
			"failure_count", count,
		)
		mapping.IsActive = false
		if err := f.repo.UpdateMapping(ctx, mapping); err != nil {
			slog.Error("forwarder: failed to auto-disable mapping",
				"mapping_id", mapping.ID,
				"error", err,
			)
		}
		// Reset counter after disabling
		f.failureMu.Lock()
		delete(f.failures, mapping.ID)
		f.failureMu.Unlock()
	}
}

// resetFailures clears the failure counter for a mapping.
func (f *Forwarder) resetFailures(mappingID uuid.UUID) {
	f.failureMu.Lock()
	delete(f.failures, mappingID)
	f.failureMu.Unlock()
}

// selectMostSpecific selects mappings using most-specific-wins routing.
// Exact module match is more specific than wildcard (empty modules list = all modules).
// If two exact matches exist for same module, the first is selected.
func selectMostSpecific(mappings []*ChannelMapping, moduleID string) []*ChannelMapping {
	var exact []*ChannelMapping
	var wildcard []*ChannelMapping

	for _, m := range mappings {
		if len(m.Modules) == 0 {
			wildcard = append(wildcard, m)
			continue
		}
		for _, mod := range m.Modules {
			if mod == moduleID {
				exact = append(exact, m)
				break
			}
		}
	}

	// Most-specific-wins: if we have exact matches, use them; otherwise fall back to wildcard
	if len(exact) > 0 {
		// Deduplicate by channel: first exact match per channel wins
		seen := make(map[string]bool)
		var deduped []*ChannelMapping
		for _, m := range exact {
			if !seen[m.ChannelID] {
				seen[m.ChannelID] = true
				deduped = append(deduped, m)
			}
		}
		return deduped
	}

	return wildcard
}

// buildActionSet returns the context-dependent action buttons for a notification's event type.
func buildActionSet(eventTypeKey string) []ActionType {
	switch {
	case strings.HasPrefix(eventTypeKey, "hr.leave."):
		// HR leave events: acknowledge + approve + reject
		return []ActionType{ActionAcknowledge, ActionApprove, ActionReject}
	case strings.HasPrefix(eventTypeKey, "biz."):
		// Finance events: acknowledge only
		return []ActionType{ActionAcknowledge}
	case strings.HasPrefix(eventTypeKey, "work.task.") || strings.HasPrefix(eventTypeKey, "crm."):
		// Task/CRM events: acknowledge + reply
		return []ActionType{ActionAcknowledge, ActionReply}
	default:
		// Default: acknowledge only
		return []ActionType{ActionAcknowledge}
	}
}

// ============================================================================
// Mapping Cache
// ============================================================================

// MappingCache provides a TTL-based cache for active channel mappings.
type MappingCache struct {
	repo Repository
	ttl  time.Duration

	mu          sync.RWMutex
	modules     map[string][]*ChannelMapping // moduleID -> mappings
	configs     map[uuid.UUID]*IntegrationConfig
	lastRefresh time.Time
}

// NewMappingCache creates a new mapping cache with the given TTL.
func NewMappingCache(repo Repository, ttl time.Duration) *MappingCache {
	return &MappingCache{
		repo:    repo,
		ttl:     ttl,
		modules: make(map[string][]*ChannelMapping),
		configs: make(map[uuid.UUID]*IntegrationConfig),
	}
}

// GetMappingsForModule returns active channel mappings for a module.
func (c *MappingCache) GetMappingsForModule(ctx context.Context, moduleID string) ([]*ChannelMapping, error) {
	c.mu.RLock()
	if time.Since(c.lastRefresh) < c.ttl {
		mappings := c.modules[moduleID]
		c.mu.RUnlock()
		return mappings, nil
	}
	c.mu.RUnlock()

	// Refresh cache
	if err := c.refresh(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.modules[moduleID], nil
}

// GetConfigForMapping returns the integration config for a channel mapping.
func (c *MappingCache) GetConfigForMapping(ctx context.Context, configID uuid.UUID) (*IntegrationConfig, error) {
	c.mu.RLock()
	if cfg, ok := c.configs[configID]; ok && time.Since(c.lastRefresh) < c.ttl {
		c.mu.RUnlock()
		return cfg, nil
	}
	c.mu.RUnlock()

	if err := c.refresh(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	cfg, ok := c.configs[configID]
	if !ok {
		return nil, ErrConfigNotFound
	}
	return cfg, nil
}

// refresh reloads all active configs and mappings from the database.
func (c *MappingCache) refresh(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if time.Since(c.lastRefresh) < c.ttl {
		return nil
	}

	// Load active configs
	configs, err := c.repo.ListConfigs(ctx)
	if err != nil {
		return err
	}

	newConfigs := make(map[uuid.UUID]*IntegrationConfig)
	newModules := make(map[string][]*ChannelMapping)

	for _, cfg := range configs {
		if !cfg.IsActive {
			continue
		}
		newConfigs[cfg.ID] = cfg

		// Load mappings for this config
		mappings, err := c.repo.ListMappingsByConfig(ctx, cfg.ID)
		if err != nil {
			continue
		}

		for _, m := range mappings {
			if !m.IsActive {
				continue
			}

			if len(m.Modules) == 0 {
				// Wildcard mapping: add to all known module keys
				// We'll handle this at query time in GetMappingsForModule
				newModules["*"] = append(newModules["*"], m)
			} else {
				for _, mod := range m.Modules {
					newModules[mod] = append(newModules[mod], m)
				}
			}
		}
	}

	c.configs = newConfigs
	c.modules = newModules
	c.lastRefresh = time.Now()

	return nil
}
