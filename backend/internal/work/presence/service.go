package presence

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/middleware"
)

// Service handles presence business logic
type Service struct {
	store      Store
	configRepo ConfigRepository
	users      UserDirectory

	// Cached away timeout per tenant, to avoid DB hits on every heartbeat.
	// Keyed by tenant since 000274: one shared entry would have handed the
	// first tenant to warm the cache its timeout to everyone else for the
	// next minute, re-creating across the cache the leak the migration
	// closed in the table.
	mu               sync.RWMutex
	awayTimeoutCache map[uuid.UUID]awayTimeoutEntry
}

// awayTimeoutEntry is one tenant's cached away timeout with its fetch time.
type awayTimeoutEntry struct {
	timeout   time.Duration
	fetchedAt time.Time
}

const (
	// configCacheDuration is how long we cache the away_timeout config
	configCacheDuration = 60 * time.Second
)

// NewService creates a new presence service
func NewService(store Store, configRepo ConfigRepository, users UserDirectory) *Service {
	return &Service{
		store:            store,
		configRepo:       configRepo,
		users:            users,
		awayTimeoutCache: make(map[uuid.UUID]awayTimeoutEntry),
	}
}

// tenantFromContext resolves the caller's tenant. Presence config is per
// tenant since 000274, so every config path needs it; the gRPC inbound
// interceptor puts it on the context for authenticated calls.
func tenantFromContext(ctx context.Context) (uuid.UUID, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return uuid.Nil, ErrMissingTenant
	}
	return tenantID, nil
}

// Heartbeat creates or renews a user's online presence.
// It does NOT override a manual DND status.
func (s *Service) Heartbeat(ctx context.Context, userID string) error {
	existing, err := s.store.GetPresence(ctx, userID)
	if err != nil {
		return err
	}

	// If user has manually set DND, don't override -- just refresh activity
	if existing != nil && existing.ManualOverride && existing.Status == PresenceDND {
		return s.store.UpdateLastActivity(ctx, userID)
	}

	// If user has manually set away, don't override
	if existing != nil && existing.ManualOverride && existing.Status == PresenceAway {
		return s.store.UpdateLastActivity(ctx, userID)
	}

	// If user is in_call, just update activity
	if existing != nil && existing.Status == PresenceInCall {
		return s.store.UpdateLastActivity(ctx, userID)
	}

	// Set/refresh online status
	return s.store.SetPresence(ctx, userID, PresenceOnline, false)
}

// visibleTargets returns the subset of userIDs whose presence the caller may
// read: the caller's own ID always, anyone else only when the directory finds
// them in the caller's tenant. The Redis key carries no tenant, so this is the
// isolation boundary for the read paths — the same one the WebSocket
// subscribe path draws in tenantAllowsPresenceTarget. Fails closed: a missing
// tenant rejects the call, a missing directory admits nobody but the caller.
func (s *Service) visibleTargets(ctx context.Context, userIDs []string) ([]string, error) {
	tenantID, err := tenantFromContext(ctx)
	if err != nil {
		return nil, err
	}
	callerID := middleware.GetUserID(ctx)
	isCaller := func(id string) bool { return callerID != "" && id == callerID }

	lookup := make([]uuid.UUID, 0, len(userIDs))
	for _, id := range userIDs {
		if isCaller(id) {
			continue
		}
		if uid, parseErr := uuid.Parse(id); parseErr == nil {
			lookup = append(lookup, uid)
		}
	}

	visible := map[uuid.UUID]struct{}{}
	if len(lookup) > 0 && s.users != nil {
		visible, err = s.users.VisibleUserIDs(ctx, tenantID, lookup)
		if err != nil {
			return nil, err
		}
	}

	allowed := make([]string, 0, len(userIDs))
	for _, id := range userIDs {
		if isCaller(id) {
			allowed = append(allowed, id)
			continue
		}
		uid, parseErr := uuid.Parse(id)
		if parseErr != nil {
			continue
		}
		if _, ok := visible[uid]; ok {
			allowed = append(allowed, id)
		}
	}

	if dropped := len(userIDs) - len(allowed); dropped > 0 {
		slog.Warn("presence: dropped targets not visible to caller's tenant",
			"caller", callerID,
			"tenant_id", tenantID,
			"dropped", dropped,
		)
	}
	return allowed, nil
}

// GetPresence retrieves a user's presence with lazy away detection. A target
// outside the caller's tenant yields ErrUserNotVisible.
func (s *Service) GetPresence(ctx context.Context, userID string) (*UserPresence, error) {
	allowed, err := s.visibleTargets(ctx, []string{userID})
	if err != nil {
		return nil, err
	}
	if len(allowed) == 0 {
		return nil, ErrUserNotVisible
	}

	p, err := s.store.GetPresence(ctx, userID)
	if err != nil {
		return nil, err
	}

	if p == nil {
		// User has no presence key = offline
		uid, parseErr := uuid.Parse(userID)
		if parseErr != nil {
			return nil, parseErr
		}
		return &UserPresence{
			UserID: uid,
			Status: PresenceOffline,
		}, nil
	}

	// Lazy away detection: if status is online and last activity exceeds timeout
	if p.Status == PresenceOnline && !p.ManualOverride {
		timeout, err := s.getAwayTimeout(ctx)
		if err != nil {
			slog.Warn("failed to get away timeout, using default",
				"error", err,
			)
			timeout = time.Duration(DefaultAwayTimeoutSeconds) * time.Second
		}

		if p.LastActivity != nil && time.Since(*p.LastActivity) > timeout {
			p.Status = PresenceAway
		}
	}

	return p, nil
}

// GetBulkPresence retrieves presence for multiple users with away detection.
// Targets outside the caller's tenant are removed from the answer, not
// reported as offline — an offline entry would still confirm the ID.
func (s *Service) GetBulkPresence(ctx context.Context, userIDs []string) (map[string]*UserPresence, error) {
	userIDs, err := s.visibleTargets(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	result, err := s.store.GetBulkPresence(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	timeout, err := s.getAwayTimeout(ctx)
	if err != nil {
		slog.Warn("failed to get away timeout for bulk, using default",
			"error", err,
		)
		timeout = time.Duration(DefaultAwayTimeoutSeconds) * time.Second
	}

	// Apply lazy away detection to all online users
	for _, p := range result {
		if p.Status == PresenceOnline && !p.ManualOverride {
			if p.LastActivity != nil && time.Since(*p.LastActivity) > timeout {
				p.Status = PresenceAway
			}
		}
	}

	// Fill in offline status for missing users
	for _, uid := range userIDs {
		if _, ok := result[uid]; !ok {
			parsedUID, parseErr := uuid.Parse(uid)
			if parseErr != nil {
				continue
			}
			result[uid] = &UserPresence{
				UserID: parsedUID,
				Status: PresenceOffline,
			}
		}
	}

	return result, nil
}

// SetManualStatus allows a user to manually set their status.
// Only Online, Away, and DND can be set manually.
func (s *Service) SetManualStatus(ctx context.Context, userID string, status PresenceStatus) error {
	if status != PresenceOnline && status != PresenceAway && status != PresenceDND {
		return ErrInvalidManualStatus
	}

	manual := status != PresenceOnline
	return s.store.SetPresence(ctx, userID, status, manual)
}

// SetInCall sets the user's status to in_call automatically when joining a call
func (s *Service) SetInCall(ctx context.Context, userID string) error {
	return s.store.SetPresence(ctx, userID, PresenceInCall, false)
}

// ClearInCall reverts the user from in_call to online, unless they have manual DND
func (s *Service) ClearInCall(ctx context.Context, userID string) error {
	existing, err := s.store.GetPresence(ctx, userID)
	if err != nil {
		return err
	}

	// If no existing presence or not in_call, nothing to do
	if existing == nil || existing.Status != PresenceInCall {
		return nil
	}

	// Check if user had manual DND before the call (we lose this info with current design,
	// so default to online after call ends)
	return s.store.SetPresence(ctx, userID, PresenceOnline, false)
}

// Disconnect removes a user's presence (e.g., on WebSocket disconnect)
func (s *Service) Disconnect(ctx context.Context, userID string) error {
	return s.store.RemovePresence(ctx, userID)
}

// GetConfig retrieves the calling tenant's presence configuration. A tenant
// that has never written one gets the code default rather than an error: the
// row is created on first write, so its absence means "unchanged", not "broken".
func (s *Service) GetConfig(ctx context.Context) (*PresenceConfig, error) {
	tenantID, err := tenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	cfg, err := s.configRepo.GetConfig(ctx, tenantID)
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) {
			return defaultConfigFor(tenantID), nil
		}
		return nil, err
	}
	return cfg, nil
}

// defaultConfigFor is the configuration a tenant has before its first write.
func defaultConfigFor(tenantID uuid.UUID) *PresenceConfig {
	return &PresenceConfig{
		TenantID:           tenantID,
		AwayTimeoutSeconds: DefaultAwayTimeoutSeconds,
	}
}

// UpdateConfig updates the calling tenant's presence configuration
func (s *Service) UpdateConfig(ctx context.Context, awayTimeoutSeconds int, updatedBy uuid.UUID) error {
	if awayTimeoutSeconds < 60 || awayTimeoutSeconds > 3600 {
		return ErrInvalidAwayTimeout
	}

	tenantID, err := tenantFromContext(ctx)
	if err != nil {
		return err
	}

	if err := s.configRepo.UpdateConfig(ctx, tenantID, awayTimeoutSeconds, updatedBy); err != nil {
		return err
	}

	// Invalidate this tenant's cached timeout; other tenants keep theirs.
	s.mu.Lock()
	delete(s.awayTimeoutCache, tenantID)
	s.mu.Unlock()

	slog.Info("presence config updated",
		"tenant_id", tenantID,
		"away_timeout_seconds", awayTimeoutSeconds,
		"updated_by", updatedBy,
	)

	return nil
}

// getAwayTimeout returns the tenant's cached away timeout, refreshing from DB
// if stale.
func (s *Service) getAwayTimeout(ctx context.Context) (time.Duration, error) {
	tenantID, err := tenantFromContext(ctx)
	if err != nil {
		return 0, err
	}

	s.mu.RLock()
	entry, ok := s.awayTimeoutCache[tenantID]
	s.mu.RUnlock()
	if ok && time.Since(entry.fetchedAt) < configCacheDuration {
		return entry.timeout, nil
	}

	// Cache miss or stale - fetch from DB
	cfg, err := s.configRepo.GetConfig(ctx, tenantID)
	if err != nil {
		if !errors.Is(err, ErrConfigNotFound) {
			return 0, err
		}
		cfg = defaultConfigFor(tenantID)
	}

	timeout := time.Duration(cfg.AwayTimeoutSeconds) * time.Second

	s.mu.Lock()
	s.awayTimeoutCache[tenantID] = awayTimeoutEntry{timeout: timeout, fetchedAt: time.Now()}
	s.mu.Unlock()

	return timeout, nil
}
