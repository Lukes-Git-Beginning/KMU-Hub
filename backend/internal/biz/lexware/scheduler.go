package lexware

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// tenantScheduler holds the cancel function and metadata for a single tenant's scheduler.
type tenantScheduler struct {
	cancel   context.CancelFunc
	configID uuid.UUID
	tenantID uuid.UUID
}

// Scheduler manages background polling goroutines per active Lexware tenant.
// Only contacts are polled on a timer (15min default); invoices and quotes are
// push-only, and real-time updates come through webhooks.
type Scheduler struct {
	service    *Service
	repo       Repository
	configRepo IntegrationConfigRepo

	mu        sync.Mutex
	tenants   map[uuid.UUID]*tenantScheduler
	parentCtx context.Context
}

// NewScheduler creates a new background scheduler.
func NewScheduler(service *Service, repo Repository, configRepo IntegrationConfigRepo) *Scheduler {
	return &Scheduler{
		service:    service,
		repo:       repo,
		configRepo: configRepo,
		tenants:    make(map[uuid.UUID]*tenantScheduler),
	}
}

// StartAll loads all active Lexware integrations and starts their schedulers.
func (s *Scheduler) StartAll(ctx context.Context) error {
	s.parentCtx = ctx

	// Single-tenant mode: look up the one lexware config.
	// Multi-tenant: iterate all active integration_configs where platform='lexware'.
	config, err := s.configRepo.GetByPlatform(ctx, "lexware")
	if err != nil {
		slog.Info("lexware scheduler: no active integration found, skipping")
		return nil
	}

	if !config.IsActive {
		slog.Info("lexware scheduler: integration not active, skipping")
		return nil
	}

	tenantID := config.CreatedBy
	if err := s.AddTenant(ctx, config.ID, tenantID); err != nil {
		return err
	}

	slog.Info("lexware scheduler started", "tenants", 1)
	return nil
}

// AddTenant starts a scheduler for a specific tenant.
func (s *Scheduler) AddTenant(ctx context.Context, configID, tenantID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop existing scheduler if any
	if existing, ok := s.tenants[tenantID]; ok {
		existing.cancel()
	}

	parentCtx := ctx
	if s.parentCtx != nil {
		parentCtx = s.parentCtx
	}

	tenantCtx, cancel := context.WithCancel(parentCtx)
	ts := &tenantScheduler{
		cancel:   cancel,
		configID: configID,
		tenantID: tenantID,
	}
	s.tenants[tenantID] = ts

	// Load sync config for intervals
	syncConfig, err := s.repo.GetSyncConfig(ctx, configID)
	if err != nil {
		cancel()
		delete(s.tenants, tenantID)
		return err
	}

	go s.runTenantScheduler(tenantCtx, ts, syncConfig)

	slog.Info("lexware tenant scheduler started",
		"tenant_id", tenantID,
		"config_id", configID,
		"contact_interval_min", syncConfig.ContactSyncIntervalMin,
	)

	return nil
}

// RemoveTenant stops the scheduler for a specific tenant.
func (s *Scheduler) RemoveTenant(tenantID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ts, ok := s.tenants[tenantID]; ok {
		ts.cancel()
		delete(s.tenants, tenantID)
		slog.Info("lexware tenant scheduler stopped", "tenant_id", tenantID)
	}
}

// StopAll gracefully stops all tenant schedulers.
func (s *Scheduler) StopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for tenantID, ts := range s.tenants {
		ts.cancel()
		slog.Info("lexware tenant scheduler stopped", "tenant_id", tenantID)
	}
	s.tenants = make(map[uuid.UUID]*tenantScheduler)

	slog.Info("lexware scheduler: all tenants stopped")
}

// runTenantScheduler runs the contact sync ticker for a single tenant.
// Only contacts are polled; Lexware webhooks are the primary mechanism for
// real-time updates on invoices and other entities.
func (s *Scheduler) runTenantScheduler(ctx context.Context, ts *tenantScheduler, syncConfig *models.LexwareSyncConfig) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("lexware scheduler panic recovered",
				"tenant_id", ts.tenantID,
				"panic", r,
			)
		}
	}()

	contactInterval := time.Duration(syncConfig.ContactSyncIntervalMin) * time.Minute
	if contactInterval < time.Minute {
		contactInterval = 15 * time.Minute // Default 15 min
	}

	contactTicker := time.NewTicker(contactInterval)
	defer contactTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-contactTicker.C:
			if syncConfig.ContactSyncEnabled {
				if _, err := s.service.SyncContacts(ctx, ts.tenantID); err != nil {
					slog.Error("lexware scheduled contact sync failed",
						"tenant_id", ts.tenantID,
						"error", err,
					)
				}
			}
		}
	}
}
