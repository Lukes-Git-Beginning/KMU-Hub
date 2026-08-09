package bexio

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// QuoteReader provides read access to KMU Hub quotes for the push engine.
type QuoteReader interface {
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Quote, error)
}

// QuotePusher pushes KMU Hub quotes to Bexio.
type QuotePusher struct {
	client      *Client
	repo        Repository
	fieldMapper *FieldMapper
	quotes      QuoteReader
	contacts    ContactService
}

// NewQuotePusher creates a new quote pusher.
func NewQuotePusher(client *Client, repo Repository, fieldMapper *FieldMapper, quotes QuoteReader, contacts ContactService) *QuotePusher {
	return &QuotePusher{
		client:      client,
		repo:        repo,
		fieldMapper: fieldMapper,
		quotes:      quotes,
		contacts:    contacts,
	}
}

// PushQuote pushes a single quote to Bexio.
func (qp *QuotePusher) PushQuote(ctx context.Context, configID, tenantID, quoteID uuid.UUID) error {
	// Load quote
	quote, err := qp.quotes.GetByID(ctx, tenantID, quoteID)
	if err != nil {
		return fmt.Errorf("bexio push quote: load quote: %w", err)
	}

	// Load field mappings
	fieldMappings, err := qp.repo.GetFieldMappings(ctx, configID, "quote")
	if err != nil {
		return fmt.Errorf("bexio push quote: get field mappings: %w", err)
	}
	var mappingEntries []models.BexioFieldMappingEntry
	if fieldMappings != nil {
		mappingEntries = fieldMappings.Mappings
	} else {
		mappingEntries = DefaultQuoteMappings()
	}

	// Resolve contact mapping
	contactBexioID, err := qp.resolveContactBexioID(ctx, configID, quote)
	if err != nil {
		return fmt.Errorf("bexio push quote: resolve contact: %w", err)
	}

	// Resolve the tenant's Bexio currency IDs (tenant-specific) so the document
	// currency maps correctly instead of always defaulting to CHF (B6). On refresh
	// failure the mapper falls back to CHF.
	lookupCache := NewLookupCache()
	if refreshErr := lookupCache.Refresh(ctx, qp.client, tenantID); refreshErr != nil {
		slog.Warn("bexio: lookup cache refresh failed; quote currency falls back to CHF",
			"tenant_id", tenantID, "error", refreshErr)
	}

	// Map to Bexio format
	bexioQuote, err := qp.fieldMapper.MapQuoteToBexio(quote, contactBexioID, mappingEntries, lookupCache)
	if err != nil {
		return fmt.Errorf("bexio push quote: map: %w", err)
	}

	// Check existing entity mapping
	now := time.Now().UTC()
	mapping, err := qp.repo.GetEntityMapping(ctx, configID, "quote", quoteID)
	if err != nil && !errors.Is(err, ErrMappingNotFound) {
		return fmt.Errorf("bexio push quote: check mapping: %w", err)
	}

	if mapping != nil {
		// Update existing
		if _, err := qp.client.UpdateQuote(ctx, tenantID, mapping.BexioID, bexioQuote); err != nil {
			return fmt.Errorf("bexio push quote: update in bexio: %w", err)
		}
		mapping.LastSyncedAt = now
		mapping.KmuhubUpdatedAt = &quote.UpdatedAt
		if err := qp.repo.UpsertEntityMapping(ctx, mapping); err != nil {
			slog.Error("bexio: failed to update quote mapping", "quote_id", quoteID, "error", err)
		}
	} else {
		// Create new
		created, err := qp.client.CreateQuote(ctx, tenantID, bexioQuote)
		if err != nil {
			return fmt.Errorf("bexio push quote: create in bexio: %w", err)
		}

		newMapping := &models.BexioEntityMapping{
			TenantID:        tenantID,
			ConfigID:        configID,
			EntityType:      "quote",
			KmuhubID:        quoteID,
			BexioID:         created.ID,
			LastSyncedAt:    now,
			KmuhubUpdatedAt: &quote.UpdatedAt,
			SyncDirection:   "outbound",
		}
		if err := qp.repo.UpsertEntityMapping(ctx, newMapping); err != nil {
			slog.Error("bexio: failed to create quote mapping", "quote_id", quoteID, "error", err)
		}
	}

	// Log sync
	syncLog := &models.BexioSyncLog{
		TenantID:       tenantID,
		ConfigID:       configID,
		SyncType:       "quote_push",
		Status:         "completed",
		ItemsProcessed: 1,
		ItemsCreated:   boolToInt(mapping == nil),
		ItemsUpdated:   boolToInt(mapping != nil),
		StartedAt:      now,
		CompletedAt:    &now,
		Metadata:       map[string]any{"quote_id": quoteID.String()},
	}
	if err := qp.repo.CreateSyncLog(ctx, syncLog); err != nil {
		slog.Error("bexio: failed to create quote push log", "error", err)
	}

	slog.Info("bexio quote pushed",
		"quote_id", quoteID,
		"config_id", configID,
	)

	return nil
}

func (qp *QuotePusher) resolveContactBexioID(ctx context.Context, configID uuid.UUID, quote *models.Quote) (int, error) {
	return resolveContactBexioIDByEmail(ctx, qp.repo, qp.contacts, configID, quote.CustomerEmail)
}
