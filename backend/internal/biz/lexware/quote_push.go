package lexware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

type QuoteReader interface {
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Quote, error)
}

type QuotePusher struct {
	client      *Client
	repo        Repository
	fieldMapper *FieldMapper
	quotes      QuoteReader
}

func NewQuotePusher(client *Client, repo Repository, fieldMapper *FieldMapper, quotes QuoteReader) *QuotePusher {
	return &QuotePusher{
		client:      client,
		repo:        repo,
		fieldMapper: fieldMapper,
		quotes:      quotes,
	}
}

func (qp *QuotePusher) PushQuote(ctx context.Context, configID, tenantID, quoteID uuid.UUID) error {
	quote, err := qp.quotes.GetByID(ctx, tenantID, quoteID)
	if err != nil {
		return fmt.Errorf("lexware push quote: load quote: %w", err)
	}

	fieldMappings, err := qp.repo.GetFieldMappings(ctx, configID, "quote")
	if err != nil {
		return fmt.Errorf("lexware push quote: get field mappings: %w", err)
	}
	var mappingEntries []models.LexwareFieldMappingEntry
	if fieldMappings != nil {
		mappingEntries = fieldMappings.Mappings
	} else {
		mappingEntries = DefaultQuoteMappings()
	}

	contactLexwareID, err := qp.resolveContactLexwareID(ctx, configID, quote)
	if err != nil {
		return fmt.Errorf("lexware push quote: resolve contact: %w", err)
	}

	lexwareQuote, err := qp.fieldMapper.MapQuoteToLexware(quote, contactLexwareID, mappingEntries)
	if err != nil {
		return fmt.Errorf("lexware push quote: map: %w", err)
	}

	now := time.Now().UTC()
	mapping, err := qp.repo.GetEntityMapping(ctx, configID, "quote", quoteID)
	if err != nil && !errors.Is(err, ErrMappingNotFound) {
		return fmt.Errorf("lexware push quote: check mapping: %w", err)
	}

	if mapping != nil {
		lexwareQuote.Version = mapping.LexwareVersion
		updated, err := qp.client.UpdateQuotation(ctx, tenantID, mapping.LexwareID, lexwareQuote)
		if err != nil {
			return fmt.Errorf("lexware push quote: update: %w", err)
		}
		mapping.LastSyncedAt = now
		mapping.KmuhubUpdatedAt = &quote.UpdatedAt
		mapping.LexwareVersion = updated.Version
		if err := qp.repo.UpsertEntityMapping(ctx, mapping); err != nil {
			slog.Error("lexware: failed to update quote mapping", "quote_id", quoteID, "error", err)
		}
	} else {
		created, err := qp.client.CreateQuotation(ctx, tenantID, lexwareQuote)
		if err != nil {
			return fmt.Errorf("lexware push quote: create: %w", err)
		}

		newMapping := &models.LexwareEntityMapping{
			ConfigID:        configID,
			EntityType:      "quote",
			KmuhubID:        quoteID,
			LexwareID:       created.ID,
			LexwareVersion:  created.Version,
			LastSyncedAt:    now,
			KmuhubUpdatedAt: &quote.UpdatedAt,
			SyncDirection:   "outbound",
		}
		if err := qp.repo.UpsertEntityMapping(ctx, newMapping); err != nil {
			slog.Error("lexware: failed to create quote mapping", "quote_id", quoteID, "error", err)
		}
	}

	syncLog := &models.LexwareSyncLog{
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
		slog.Error("lexware: failed to create quote push log", "error", err)
	}

	slog.Info("lexware quote pushed", "quote_id", quoteID, "config_id", configID)
	return nil
}

func (qp *QuotePusher) resolveContactLexwareID(ctx context.Context, configID uuid.UUID, quote *models.Quote) (string, error) {
	if quote.CustomerEmail != "" {
		mappings, err := qp.repo.ListEntityMappings(ctx, configID, "contact")
		if err != nil {
			return "", fmt.Errorf("list contact mappings: %w", err)
		}
		if len(mappings) > 0 {
			return mappings[0].LexwareID, nil
		}
	}
	return "", fmt.Errorf("no synced Lexware contact found for quote customer %q", quote.CustomerEmail)
}
