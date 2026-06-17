package bexio

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/sysctx"
)

// Service is the central orchestrator for all Bexio integration operations.
type Service struct {
	client        *Client
	repo          Repository
	oauthHandler  *OAuthHandler
	contactSyncer *ContactSyncer
	invoicePusher *InvoicePusher
	quotePusher   *QuotePusher
	paymentPoller *PaymentPoller
	fieldMapper   *FieldMapper
	scheduler     *Scheduler
	configRepo    IntegrationConfigRepo
	emitter       EventEmitter
}

// NewService creates a new Bexio integration service.
func NewService(
	client *Client,
	repo Repository,
	configRepo IntegrationConfigRepo,
	vault VaultService,
	config ClientConfig,
	contacts ContactService,
	invoices InvoiceReader,
	invoiceUpdater InvoiceStatusUpdater,
	quotes QuoteReader,
) *Service {
	fieldMapper := NewFieldMapper()

	s := &Service{
		client:        client,
		repo:          repo,
		configRepo:    configRepo,
		fieldMapper:   fieldMapper,
		contactSyncer: NewContactSyncer(client, repo, fieldMapper, contacts),
		invoicePusher: NewInvoicePusher(client, repo, fieldMapper, invoices, contacts),
		quotePusher:   NewQuotePusher(client, repo, fieldMapper, quotes, contacts),
		paymentPoller: NewPaymentPoller(client, repo, invoiceUpdater),
		emitter:       noopEmitter{},
	}

	s.oauthHandler = NewOAuthHandler(client.TokenManager(), repo, configRepo, vault, config)
	s.scheduler = NewScheduler(s, repo, configRepo)

	return s
}

// SetEventEmitter sets the event emitter for the Bexio service.
func (s *Service) SetEventEmitter(emitter EventEmitter) {
	s.emitter = emitter
}

// --- OAuth ---

// GetAuthorizationURL returns the Bexio OAuth authorization URL.
func (s *Service) GetAuthorizationURL(state string) string {
	return s.oauthHandler.GetAuthorizationURL(state)
}

// HandleOAuthCallback exchanges the authorization code and sets up the integration.
// Pre-JWT path (OAuth redirect). Wrap at the service-layer entry point so the
// post-callback emitter.Emit + GetConnectionStatus + scheduler.AddTenant calls
// also run under sysctx (oauthHandler.HandleCallback wraps only its own ctx).
func (s *Service) HandleOAuthCallback(ctx context.Context, tenantID uuid.UUID, code string) error {
	ctx = sysctx.With(ctx)

	if err := s.oauthHandler.HandleCallback(ctx, tenantID, code); err != nil {
		return err
	}

	_ = s.emitter.Emit(ctx, EventPayload{
		Type:     EventConnected,
		TenantID: tenantID.String(),
	})

	// Start scheduler for newly connected tenant
	status, err := s.oauthHandler.GetConnectionStatus(ctx)
	if err == nil && status.ConfigID != nil {
		if err := s.scheduler.AddTenant(ctx, *status.ConfigID, tenantID); err != nil {
			slog.Error("bexio: failed to start scheduler after connect", "tenant_id", tenantID, "error", err)
		}
	}

	return nil
}

// Disconnect revokes tokens and stops the integration.
func (s *Service) Disconnect(ctx context.Context, tenantID uuid.UUID) error {
	s.scheduler.RemoveTenant(tenantID)

	if err := s.oauthHandler.Disconnect(ctx, tenantID); err != nil {
		return err
	}

	_ = s.emitter.Emit(ctx, EventPayload{
		Type:     EventDisconnected,
		TenantID: tenantID.String(),
	})

	return nil
}

// GetConnectionStatus returns the current connection state.
func (s *Service) GetConnectionStatus(ctx context.Context) (*ConnectionStatus, error) {
	return s.oauthHandler.GetConnectionStatus(ctx)
}

// --- Sync Operations ---

// SyncContacts triggers a contact sync (manual or from scheduler).
func (s *Service) SyncContacts(ctx context.Context, tenantID uuid.UUID) (*SyncResult, error) {
	configID, err := s.getConfigID(ctx)
	if err != nil {
		return nil, err
	}

	_ = s.emitter.Emit(ctx, EventPayload{
		Type:     EventSyncStarted,
		TenantID: tenantID.String(),
		Data:     map[string]any{"sync_type": "contact"},
	})

	result, err := s.contactSyncer.SyncContacts(ctx, configID, tenantID)
	if err != nil {
		_ = s.emitter.Emit(ctx, EventPayload{
			Type:     EventSyncFailed,
			TenantID: tenantID.String(),
			Data:     map[string]any{"sync_type": "contact", "error": err.Error()},
		})
		return nil, err
	}

	_ = s.emitter.Emit(ctx, EventPayload{
		Type:     EventSyncCompleted,
		TenantID: tenantID.String(),
		Data:     map[string]any{"sync_type": "contact", "result": result},
	})

	return result, nil
}

// PushInvoice pushes an invoice to Bexio.
func (s *Service) PushInvoice(ctx context.Context, tenantID uuid.UUID, invoiceID uuid.UUID) error {
	configID, err := s.getConfigID(ctx)
	if err != nil {
		return err
	}
	return s.invoicePusher.PushInvoice(ctx, configID, tenantID, invoiceID)
}

// PushQuote pushes a quote to Bexio.
func (s *Service) PushQuote(ctx context.Context, tenantID uuid.UUID, quoteID uuid.UUID) error {
	configID, err := s.getConfigID(ctx)
	if err != nil {
		return err
	}
	return s.quotePusher.PushQuote(ctx, configID, tenantID, quoteID)
}

// PollPayments triggers a payment status poll (manual or from scheduler).
func (s *Service) PollPayments(ctx context.Context, tenantID uuid.UUID) (*SyncResult, error) {
	configID, err := s.getConfigID(ctx)
	if err != nil {
		return nil, err
	}

	_ = s.emitter.Emit(ctx, EventPayload{
		Type:     EventSyncStarted,
		TenantID: tenantID.String(),
		Data:     map[string]any{"sync_type": "payment_poll"},
	})

	result, err := s.paymentPoller.PollPayments(ctx, configID, tenantID)
	if err != nil {
		_ = s.emitter.Emit(ctx, EventPayload{
			Type:     EventSyncFailed,
			TenantID: tenantID.String(),
			Data:     map[string]any{"sync_type": "payment_poll", "error": err.Error()},
		})
		return nil, err
	}

	_ = s.emitter.Emit(ctx, EventPayload{
		Type:     EventSyncCompleted,
		TenantID: tenantID.String(),
		Data:     map[string]any{"sync_type": "payment_poll", "result": result},
	})

	return result, nil
}

// TriggerSync triggers all sync types immediately.
func (s *Service) TriggerSync(ctx context.Context, tenantID uuid.UUID) error {
	if _, err := s.SyncContacts(ctx, tenantID); err != nil {
		slog.Error("bexio trigger sync: contacts failed", "error", err)
	}
	if _, err := s.PollPayments(ctx, tenantID); err != nil {
		slog.Error("bexio trigger sync: payments failed", "error", err)
	}
	return nil
}

// --- Status & Config ---

// GetSyncStatus returns a computed sync status overview.
func (s *Service) GetSyncStatus(ctx context.Context) (*models.BexioSyncStatus, error) {
	configID, err := s.getConfigID(ctx)
	if err != nil {
		return nil, err
	}

	syncConfig, err := s.repo.GetSyncConfig(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("bexio: get sync status: %w", err)
	}

	contactMappings, _ := s.repo.ListEntityMappings(ctx, configID, "contact")
	invoiceMappings, _ := s.repo.ListEntityMappings(ctx, configID, "invoice")
	quoteMappings, _ := s.repo.ListEntityMappings(ctx, configID, "quote")

	status := &models.BexioSyncStatus{
		ContactSyncEnabled:  syncConfig.ContactSyncEnabled,
		InvoicePushEnabled:  syncConfig.InvoicePushEnabled,
		QuotePushEnabled:    syncConfig.QuotePushEnabled,
		PaymentPollEnabled:  syncConfig.PaymentPollEnabled,
		LastContactSyncAt:   syncConfig.LastContactSyncAt,
		LastPaymentPollAt:   syncConfig.LastPaymentPollAt,
		TotalContactsMapped: len(contactMappings),
		TotalInvoicesMapped: len(invoiceMappings),
		TotalQuotesMapped:   len(quoteMappings),
	}

	// Get last error from sync log
	latestLog, _ := s.repo.GetLatestSyncLog(ctx, configID, "contact_delta")
	if latestLog != nil && latestLog.Status == "failed" {
		status.LastSyncError = latestLog.ErrorMessage
		status.LastSyncErrorAt = &latestLog.StartedAt
	}

	return status, nil
}

// GetSyncLogs returns recent sync log entries.
func (s *Service) GetSyncLogs(ctx context.Context, limit int) ([]models.BexioSyncLog, error) {
	configID, err := s.getConfigID(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListSyncLogs(ctx, configID, limit)
}

// GetFieldMappings returns field mappings for an entity type.
func (s *Service) GetFieldMappings(ctx context.Context, entityType string) (*models.BexioFieldMapping, error) {
	configID, err := s.getConfigID(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.GetFieldMappings(ctx, configID, entityType)
}

// UpdateFieldMappings validates and saves field mappings.
func (s *Service) UpdateFieldMappings(ctx context.Context, entityType string, mappings []models.BexioFieldMappingEntry) error {
	if err := ValidateFieldMappings(entityType, mappings); err != nil {
		return err
	}

	configID, err := s.getConfigID(ctx)
	if err != nil {
		return err
	}

	fm := &models.BexioFieldMapping{
		ConfigID:   configID,
		EntityType: entityType,
		Mappings:   mappings,
	}
	return s.repo.UpsertFieldMappings(ctx, fm)
}

// StartScheduler starts the background polling scheduler for all active tenants.
func (s *Service) StartScheduler(ctx context.Context) error {
	return s.scheduler.StartAll(ctx)
}

// StopScheduler stops all background polling.
func (s *Service) StopScheduler() {
	s.scheduler.StopAll()
}

// getConfigID looks up the integration_configs entry for platform='bexio'.
func (s *Service) getConfigID(ctx context.Context) (uuid.UUID, error) {
	config, err := s.configRepo.GetByPlatform(ctx, "bexio")
	if err != nil {
		return uuid.Nil, fmt.Errorf("bexio: integration not configured: %w", err)
	}
	if !config.IsActive {
		return uuid.Nil, fmt.Errorf("bexio: integration is not active")
	}
	return config.ID, nil
}
