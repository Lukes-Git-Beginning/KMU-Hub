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
}

// NewQuotePusher creates a new quote pusher.
func NewQuotePusher(client *Client, repo Repository, fieldMapper *FieldMapper, quotes QuoteReader) *QuotePusher {
	return &QuotePusher{
		client:      client,
		repo:        repo,
		fieldMapper: fieldMapper,
		quotes:      quotes,
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

	// Map to Bexio format
	bexioQuote, err := qp.fieldMapper.MapQuoteToBexio(quote, contactBexioID, mappingEntries)
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
	if quote.CustomerEmail != "" {
		mappings, err := qp.repo.ListEntityMappings(ctx, configID, "contact")
		if err != nil {
			return 0, fmt.Errorf("list contact mappings: %w", err)
		}
		if len(mappings) > 0 {
			return mappings[0].BexioID, nil
		}
	}

	return 0, fmt.Errorf("no synced Bexio contact found for quote customer %q", quote.CustomerEmail)
}
