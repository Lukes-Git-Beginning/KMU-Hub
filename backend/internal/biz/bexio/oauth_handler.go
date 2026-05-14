package bexio

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/sysctx"
)

// ConnectionStatus represents the current Bexio connection state.
type ConnectionStatus struct {
	Connected   bool       `json:"connected"`
	ConfigID    *uuid.UUID `json:"config_id,omitempty"`
	ConnectedAt *time.Time `json:"connected_at,omitempty"`
}

// IntegrationConfigRepo defines access to the shared integration_configs table.
type IntegrationConfigRepo interface {
	GetByPlatform(ctx context.Context, platform string) (*IntegrationConfig, error)
	Upsert(ctx context.Context, config *IntegrationConfig) error
	Deactivate(ctx context.Context, id uuid.UUID) error
}

// IntegrationConfig mirrors the integration_configs table row.
type IntegrationConfig struct {
	ID                 uuid.UUID      `json:"id"`
	TenantID           uuid.UUID      `json:"tenant_id"`
	Platform           string         `json:"platform"`
	IsActive           bool           `json:"is_active"`
	CredentialsVaultKey string        `json:"credentials_vault_key"`
	Metadata           map[string]any `json:"metadata"`
	CreatedBy          uuid.UUID      `json:"created_by"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

// OAuthHandler manages the Bexio OAuth2 connect/disconnect flow.
type OAuthHandler struct {
	tokenManager *TokenManager
	repo         Repository
	configRepo   IntegrationConfigRepo
	vault        VaultService
	config       ClientConfig
}

// NewOAuthHandler creates a new OAuth handler.
func NewOAuthHandler(tokenManager *TokenManager, repo Repository, configRepo IntegrationConfigRepo, vault VaultService, config ClientConfig) *OAuthHandler {
	return &OAuthHandler{
		tokenManager: tokenManager,
		repo:         repo,
		configRepo:   configRepo,
		vault:        vault,
		config:       config,
	}
}

// GetAuthorizationURL builds the Bexio OAuth authorization URL.
func (h *OAuthHandler) GetAuthorizationURL(state string) string {
	params := url.Values{
		"response_type": {"code"},
		"client_id":     {h.config.ClientID},
		"redirect_uri":  {h.config.RedirectURL},
		"scope":         {strings.Join(h.config.Scopes, " ")},
		"state":         {state},
	}
	return h.config.AuthURL + "?" + params.Encode()
}

// HandleCallback exchanges the authorization code for tokens and sets up the integration.
// Pre-JWT path (OAuth redirect from Bexio with state token, no user JWT): wrap in
// sysctx so the INSERTs into integration_configs / bexio_sync_configs /
// bexio_field_mappings (all RLS-enabled in mig 122) pass WITH CHECK. tenant_id
// is set explicitly from the state-token-resolved tenantID parameter.
func (h *OAuthHandler) HandleCallback(ctx context.Context, tenantID uuid.UUID, code string) error {
	ctx = sysctx.With(ctx)

	// Exchange code for tokens
	if err := h.tokenManager.ExchangeCode(ctx, tenantID, code, h.config.RedirectURL); err != nil {
		return fmt.Errorf("bexio oauth callback: %w", err)
	}

	// Create or update integration config
	now := time.Now().UTC()
	ic := &IntegrationConfig{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		Platform:           "bexio",
		IsActive:           true,
		CredentialsVaultKey: vaultKey(tenantID),
		Metadata:           map[string]any{"connected_at": now.Format(time.RFC3339)},
		CreatedBy:          tenantID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := h.configRepo.Upsert(ctx, ic); err != nil {
		return fmt.Errorf("bexio oauth: upsert config: %w", err)
	}

	// Create default sync config
	syncConfig := &models.BexioSyncConfig{
		ConfigID:               ic.ID,
		ContactSyncEnabled:     true,
		ContactSyncIntervalMin: 15,
		InvoicePushEnabled:     true,
		QuotePushEnabled:       true,
		PaymentPollEnabled:     true,
		PaymentPollIntervalMin: 5,
	}
	if err := h.repo.UpsertSyncConfig(ctx, syncConfig); err != nil {
		return fmt.Errorf("bexio oauth: create sync config: %w", err)
	}

	// Create default field mappings
	for _, entityType := range []string{"contact", "invoice", "quote"} {
		var mappings []models.BexioFieldMappingEntry
		switch entityType {
		case "contact":
			mappings = DefaultContactMappings()
		case "invoice":
			mappings = DefaultInvoiceMappings()
		case "quote":
			mappings = DefaultQuoteMappings()
		}

		fm := &models.BexioFieldMapping{
			ConfigID:   ic.ID,
			EntityType: entityType,
			Mappings:   mappings,
		}
		if err := h.repo.UpsertFieldMappings(ctx, fm); err != nil {
			return fmt.Errorf("bexio oauth: create %s field mappings: %w", entityType, err)
		}
	}

	slog.Info("bexio integration connected",
		"tenant_id", tenantID,
		"config_id", ic.ID,
	)

	return nil
}

// Disconnect revokes tokens and deactivates the integration.
func (h *OAuthHandler) Disconnect(ctx context.Context, tenantID uuid.UUID) error {
	config, err := h.configRepo.GetByPlatform(ctx, "bexio")
	if err != nil {
		return fmt.Errorf("bexio disconnect: %w", err)
	}

	if err := h.tokenManager.RevokeTokens(ctx, tenantID); err != nil {
		slog.Error("bexio: failed to revoke tokens", "tenant_id", tenantID, "error", err)
	}

	if err := h.configRepo.Deactivate(ctx, config.ID); err != nil {
		return fmt.Errorf("bexio disconnect: deactivate config: %w", err)
	}

	slog.Info("bexio integration disconnected", "tenant_id", tenantID)
	return nil
}

// GetConnectionStatus returns the current Bexio connection state.
func (h *OAuthHandler) GetConnectionStatus(ctx context.Context) (*ConnectionStatus, error) {
	config, err := h.configRepo.GetByPlatform(ctx, "bexio")
	if err != nil {
		return &ConnectionStatus{Connected: false}, nil
	}

	status := &ConnectionStatus{
		Connected: config.IsActive,
		ConfigID:  &config.ID,
	}

	if connAt, ok := config.Metadata["connected_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, connAt); err == nil {
			status.ConnectedAt = &t
		}
	}

	return status, nil
}
