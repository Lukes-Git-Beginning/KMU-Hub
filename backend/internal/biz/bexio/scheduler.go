package bexio

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/safego"
)

// tenantScheduler holds the cancel function and metadata for a single tenant's scheduler.
type tenantScheduler struct {
	cancel   context.CancelFunc
	configID uuid.UUID
	tenantID uuid.UUID
}

// defaultTickInterval is used when a stored interval is zero or negative.
const defaultTickInterval = 15 * time.Minute

// Scheduler manages background polling goroutines per active Bexio tenant.
type Scheduler struct {
	service    *Service
	repo       Repository
	configRepo IntegrationConfigRepo

	// intervalUnit scales the *_interval_min columns; time.Minute in production,
	// shortened by tests so a tick can actually fire.
	intervalUnit time.Duration

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

		intervalUnit: time.Minute,
	}
}

// StartAll loads all active Bexio integrations and starts a scheduler per tenant.
func (s *Scheduler) StartAll(ctx context.Context) error {
	ctx = database.WithSystemContext(ctx)
	s.parentCtx = ctx

	configs, err := s.configRepo.ListActiveByPlatform(ctx, "bexio")
	if err != nil {
		slog.Info("bexio scheduler: failed to list active integrations, skipping", "error", err)
		return nil
	}

	if len(configs) == 0 {
		slog.Info("bexio scheduler: no active integrations found, skipping")
		return nil
	}

	started := 0
	for _, config := range configs {
		if err := s.AddTenant(ctx, config.ID, config.TenantID); err != nil {
			slog.Error("bexio scheduler: failed to start tenant scheduler",
				"tenant_id", config.TenantID,
				"config_id", config.ID,
				"error", err,
			)
			continue
		}
		started++
	}

	slog.Info("bexio scheduler started", "tenants", started, "total_configs", len(configs))
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

	slog.Info("bexio tenant scheduler started",
		"tenant_id", tenantID,
		"config_id", configID,
		"contact_interval_min", syncConfig.ContactSyncIntervalMin,
		"payment_interval_min", syncConfig.PaymentPollIntervalMin,
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
		slog.Info("bexio tenant scheduler stopped", "tenant_id", tenantID)
	}
}

// StopAll gracefully stops all tenant schedulers.
func (s *Scheduler) StopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for tenantID, ts := range s.tenants {
		ts.cancel()
		slog.Info("bexio tenant scheduler stopped", "tenant_id", tenantID)
	}
	s.tenants = make(map[uuid.UUID]*tenantScheduler)

	slog.Info("bexio scheduler: all tenants stopped")
}

func (s *Scheduler) runTenantScheduler(ctx context.Context, ts *tenantScheduler, syncConfig *models.BexioSyncConfig) {
	// lean: syncConfig is a snapshot taken once in AddTenant. Enabling/disabling a sync
	// or changing an interval at runtime takes effect for these tickers only after the
	// tenant scheduler is restarted (Disconnect/reconnect or biz restart). The manual
	// gRPC trigger path is unaffected and applies immediately. Reload the config per tick
	// (or restart the tenant on UpdateSyncConfig) if live toggles become a requirement.
	contactInterval := time.Duration(syncConfig.ContactSyncIntervalMin) * s.intervalUnit
	paymentInterval := time.Duration(syncConfig.PaymentPollIntervalMin) * s.intervalUnit
	invoicePullInterval := time.Duration(syncConfig.InvoicePullIntervalMin) * s.intervalUnit
	// time.NewTicker panics on a non-positive interval; the columns default to 15
	// but guard defensively against a zero from an older row.
	if contactInterval <= 0 {
		contactInterval = defaultTickInterval
	}
	if paymentInterval <= 0 {
		paymentInterval = defaultTickInterval
	}
	if invoicePullInterval <= 0 {
		invoicePullInterval = defaultTickInterval
	}

	contactTicker := time.NewTicker(contactInterval)
	paymentTicker := time.NewTicker(paymentInterval)
	invoicePullTicker := time.NewTicker(invoicePullInterval)
	defer contactTicker.Stop()
	defer paymentTicker.Stop()
	defer invoicePullTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-contactTicker.C:
			safego.Do("bexio scheduled contact sync", func() {
				if syncConfig.ContactSyncEnabled {
					// lean: SyncContacts re-resolves the config via getConfigID under system
					// context (RLS off), which can pick the wrong tenant's config when >1
					// Bexio integration is active (same latent G8 bug the invoice pull avoids).
					// Harmless while data is single-tenant; pass ts.configID here too before
					// multi-tenant activation.
					if _, err := s.service.SyncContacts(ctx, ts.tenantID); err != nil {
						slog.Error("bexio scheduled contact sync failed",
							"tenant_id", ts.tenantID,
							"error", err,
						)
					}
				}
			})

		case <-paymentTicker.C:
			safego.Do("bexio scheduled payment poll", func() {
				if syncConfig.PaymentPollEnabled {
					// lean: same latent G8 config-resolution bug as the contact tick above;
					// pass ts.configID before multi-tenant activation.
					if _, err := s.service.PollPayments(ctx, ts.tenantID); err != nil {
						slog.Error("bexio scheduled payment poll failed",
							"tenant_id", ts.tenantID,
							"error", err,
						)
					}
				}
			})

		case <-invoicePullTicker.C:
			safego.Do("bexio scheduled invoice pull", func() {
				if syncConfig.InvoicePullEnabled {
					// G8-correct: pass the tenant's own configID explicitly instead of
					// re-resolving it under system context.
					if _, err := s.service.PullInvoicesWithConfig(ctx, ts.configID, ts.tenantID); err != nil {
						slog.Error("bexio scheduled invoice pull failed",
							"tenant_id", ts.tenantID,
							"error", err,
						)
					}
				}
			})
		}
	}
}
