package lexware

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/sysctx"
)

// Service is the central orchestrator for all Lexware integration operations.
// Unlike Bexio (OAuth), Lexware uses API key authentication.
// Unlike Bexio (payment polling), Lexware relies on webhooks for real-time updates.
type Service struct {
	client         *Client
	repo           Repository
	configRepo     IntegrationConfigRepo
	vault          VaultService
	contacts       ContactService
	invoices       InvoiceReader
	quotes         QuoteReader
	fieldMapper    *FieldMapper
	contactSyncer  *ContactSyncer
	invoicePusher  *InvoicePusher
	quotePusher    *QuotePusher
	webhookHandler *WebhookHandler
	scheduler      *Scheduler
	emitter        EventEmitter
}

// NewService creates a new Lexware integration service.
func NewService(
	client *Client,
	repo Repository,
	configRepo IntegrationConfigRepo,
	vault VaultService,
	contacts ContactService,
	invoices InvoiceReader,
	quotes QuoteReader,
) *Service {
	fieldMapper := NewFieldMapper()
	contactSyncer := NewContactSyncer(client, repo, fieldMapper, contacts)

	s := &Service{
		client:         client,
		repo:           repo,
		configRepo:     configRepo,
		vault:          vault,
		contacts:       contacts,
		invoices:       invoices,
		quotes:         quotes,
		fieldMapper:    fieldMapper,
		contactSyncer:  contactSyncer,
		invoicePusher:  NewInvoicePusher(client, repo, fieldMapper, invoices, contacts),
		quotePusher:    NewQuotePusher(client, repo, fieldMapper, quotes, contacts),
		webhookHandler: NewWebhookHandler(client, repo, noopEmitter{}, contactSyncer),
		emitter:        noopEmitter{},
	}

	s.scheduler = NewScheduler(s, repo, configRepo)
	return s
}

// SetEventEmitter sets the event emitter for the Lexware service.
func (s *Service) SetEventEmitter(emitter EventEmitter) {
	s.emitter = emitter
	if s.webhookHandler != nil {
		s.webhookHandler.SetEventEmitter(emitter)
	}
}

// --- API Key Connection (no OAuth) ---

// Connect stores the API key in the vault and activates the integration.
func (s *Service) Connect(ctx context.Context, tenantID uuid.UUID, apiKey string) error {
	vaultKey := fmt.Sprintf("lexware_api_key_%s", tenantID.String())

	if err := s.vault.SetSecret(ctx, vaultKey, apiKey, "Lexware API key", tenantID); err != nil {
		return fmt.Errorf("lexware: store api key: %w", err)
	}

	config := &IntegrationConfig{
		TenantID:            tenantID,
		Platform:            "lexware",
		IsActive:            true,
		CredentialsVaultKey: vaultKey,
		CreatedBy:           tenantID,
		Metadata:            map[string]any{"auth_type": "api_key"},
	}
	if err := s.configRepo.Upsert(ctx, config); err != nil {
		return fmt.Errorf("lexware: save integration config: %w", err)
	}

	// Create default sync config
	syncConfig := &models.LexwareSyncConfig{
		ConfigID:               config.ID,
		ContactSyncEnabled:     true,
		ContactSyncIntervalMin: 15,
		InvoicePushEnabled:     true,
		QuotePushEnabled:       true,
		WebhookEnabled:         true,
	}
	if err := s.repo.UpsertSyncConfig(ctx, syncConfig); err != nil {
		return fmt.Errorf("lexware: create default sync config: %w", err)
	}

	_ = s.emitter.Emit(ctx, EventPayload{
		Type:     EventConnected,
		TenantID: tenantID.String(),
	})

	// Start scheduler for newly connected tenant
	if err := s.scheduler.AddTenant(ctx, config.ID, tenantID); err != nil {
		slog.Error("lexware: failed to start scheduler after connect",
			"tenant_id", tenantID,
			"error", err,
		)
	}

	return nil
}

// Disconnect deactivates the integration and stops the scheduler.
func (s *Service) Disconnect(ctx context.Context, tenantID uuid.UUID) error {
	s.scheduler.RemoveTenant(tenantID)

	config, err := s.configRepo.GetByPlatform(ctx, "lexware")
	if err != nil {
		return fmt.Errorf("lexware: get config for disconnect: %w", err)
	}

	if err := s.configRepo.Deactivate(ctx, config.ID); err != nil {
		return fmt.Errorf("lexware: deactivate config: %w", err)
	}

	_ = s.emitter.Emit(ctx, EventPayload{
		Type:     EventDisconnected,
		TenantID: tenantID.String(),
	})

	return nil
}

// TestConnection verifies the stored API key is valid by making a lightweight API call.
// The check is deliberately end to end: a stored key that Lexware rejects (revoked,
// typo'd, wrong organisation) must fail here rather than show a green connection
// state and then break silently on the first sync.
func (s *Service) TestConnection(ctx context.Context) error {
	config, err := s.configRepo.GetByPlatform(ctx, "lexware")
	if err != nil {
		return fmt.Errorf("lexware: not connected: %w", err)
	}
	if !config.IsActive {
		return fmt.Errorf("lexware: integration is not active")
	}

	apiKey, err := s.vault.GetSecret(ctx, config.CredentialsVaultKey)
	if err != nil {
		return fmt.Errorf("lexware: retrieve api key: %w", err)
	}

	if apiKey == "" {
		return fmt.Errorf("lexware: api key is empty")
	}

	if s.client == nil {
		return fmt.Errorf("lexware: api client is not configured")
	}

	// The client resolves the key itself from the vault (lexware_api_key_<tenant>),
	// the same key Connect stores — so this proves the credential the syncs will
	// actually use, not just the one this method happened to read.
	if _, err := s.client.GetProfile(ctx, config.TenantID); err != nil {
		return fmt.Errorf("lexware: connection test failed: %w", err)
	}

	return nil
}

// GetConnectionStatus returns the current connection state.
func (s *Service) GetConnectionStatus(ctx context.Context) (*ConnectionStatus, error) {
	config, err := s.configRepo.GetByPlatform(ctx, "lexware")
	if err != nil {
		return &ConnectionStatus{Connected: false}, nil
	}

	status := &ConnectionStatus{
		Connected:   config.IsActive,
		ConnectedAt: &config.CreatedAt,
		ConfigID:    &config.ID,
	}
	return status, nil
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

	// The syncer owns the sync-log lifecycle (running → completed/partial/failed)
	// and the last-sync timestamp; duplicating either here would produce two log
	// rows per run and a timestamp that moves even when the sync failed.
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

// PushInvoice pushes an invoice to Lexware.
func (s *Service) PushInvoice(ctx context.Context, tenantID uuid.UUID, invoiceID uuid.UUID) error {
	configID, err := s.getConfigID(ctx)
	if err != nil {
		return err
	}
	return s.invoicePusher.PushInvoice(ctx, configID, tenantID, invoiceID)
}

// PushQuote pushes a quote to Lexware.
func (s *Service) PushQuote(ctx context.Context, tenantID uuid.UUID, quoteID uuid.UUID) error {
	configID, err := s.getConfigID(ctx)
	if err != nil {
		return err
	}
	return s.quotePusher.PushQuote(ctx, configID, tenantID, quoteID)
}

// HandleWebhookEvent processes an incoming Lexware webhook event.
// The payload keys are the ones the gRPC layer puts in (resource_id,
// organization_id); anything else is ignored.
//
// Pre-JWT path: the request comes from Lexware and is authenticated by HMAC, so
// there is no tenant in the context. Without the system marker the very first
// read (integration_configs, RLS-enabled since mig 000125) returns zero rows and
// every webhook would look like "integration not configured".
func (s *Service) HandleWebhookEvent(ctx context.Context, eventType string, payload map[string]any) error {
	ctx = sysctx.With(ctx)

	config, err := s.configRepo.GetByPlatform(ctx, "lexware")
	if err != nil {
		return fmt.Errorf("lexware: webhook for unconfigured integration: %w", err)
	}
	if !config.IsActive {
		return fmt.Errorf("lexware: webhook for inactive integration")
	}

	event := LexwareWebhookEvent{
		EventType:  eventType,
		ResourceID: payloadString(payload, "resource_id"),
		OrgID:      payloadString(payload, "organization_id"),
	}

	return s.webhookHandler.HandleEvent(ctx, config.ID, config.TenantID, event)
}

// payloadString reads a string value from a webhook payload map, returning ""
// for a missing key or a non-string value.
func payloadString(payload map[string]any, key string) string {
	if v, ok := payload[key].(string); ok {
		return v
	}
	return ""
}

// TriggerSync triggers all enabled sync types immediately.
func (s *Service) TriggerSync(ctx context.Context, tenantID uuid.UUID) error {
	if _, err := s.SyncContacts(ctx, tenantID); err != nil {
		slog.Error("lexware trigger sync: contacts failed", "error", err)
	}
	return nil
}

// --- Status & Config ---

// GetSyncStatus returns a computed sync status overview.
func (s *Service) GetSyncStatus(ctx context.Context) (*models.LexwareSyncStatus, error) {
	configID, err := s.getConfigID(ctx)
	if err != nil {
		return nil, err
	}

	syncConfig, err := s.repo.GetSyncConfig(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("lexware: get sync status: %w", err)
	}

	contactMappings, _ := s.repo.ListEntityMappings(ctx, configID, "contact")
	invoiceMappings, _ := s.repo.ListEntityMappings(ctx, configID, "invoice")
	quoteMappings, _ := s.repo.ListEntityMappings(ctx, configID, "quote")

	status := &models.LexwareSyncStatus{
		ContactSyncEnabled:  syncConfig.ContactSyncEnabled,
		InvoicePushEnabled:  syncConfig.InvoicePushEnabled,
		QuotePushEnabled:    syncConfig.QuotePushEnabled,
		WebhookEnabled:      syncConfig.WebhookEnabled,
		LastContactSyncAt:   syncConfig.LastContactSyncAt,
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
func (s *Service) GetSyncLogs(ctx context.Context, limit int) ([]models.LexwareSyncLog, error) {
	configID, err := s.getConfigID(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListSyncLogs(ctx, configID, limit)
}

// GetFieldMappings returns field mappings for an entity type.
func (s *Service) GetFieldMappings(ctx context.Context, entityType string) (*models.LexwareFieldMapping, error) {
	configID, err := s.getConfigID(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.GetFieldMappings(ctx, configID, entityType)
}

// UpdateFieldMappings validates and saves field mappings.
func (s *Service) UpdateFieldMappings(ctx context.Context, entityType string, mappings []models.LexwareFieldMappingEntry) error {
	if err := ValidateFieldMappings(entityType, mappings); err != nil {
		return err
	}

	configID, err := s.getConfigID(ctx)
	if err != nil {
		return err
	}

	fm := &models.LexwareFieldMapping{
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

// getConfigID looks up the integration_configs entry for platform='lexware'.
func (s *Service) getConfigID(ctx context.Context) (uuid.UUID, error) {
	config, err := s.configRepo.GetByPlatform(ctx, "lexware")
	if err != nil {
		return uuid.Nil, fmt.Errorf("lexware: integration not configured: %w", err)
	}
	if !config.IsActive {
		return uuid.Nil, fmt.Errorf("lexware: integration is not active")
	}
	return config.ID, nil
}
