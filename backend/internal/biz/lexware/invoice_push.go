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

type InvoiceReader interface {
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error)
}

type InvoicePusher struct {
	client      *Client
	repo        Repository
	fieldMapper *FieldMapper
	invoices    InvoiceReader
	contacts    ContactService
}

func NewInvoicePusher(client *Client, repo Repository, fieldMapper *FieldMapper, invoices InvoiceReader, contacts ContactService) *InvoicePusher {
	return &InvoicePusher{
		client:      client,
		repo:        repo,
		fieldMapper: fieldMapper,
		invoices:    invoices,
		contacts:    contacts,
	}
}

func (ip *InvoicePusher) PushInvoice(ctx context.Context, configID, tenantID, invoiceID uuid.UUID) error {
	invoice, err := ip.invoices.GetByID(ctx, tenantID, invoiceID)
	if err != nil {
		return fmt.Errorf("lexware push invoice: load invoice: %w", err)
	}

	fieldMappings, err := ip.repo.GetFieldMappings(ctx, configID, "invoice")
	if err != nil {
		return fmt.Errorf("lexware push invoice: get field mappings: %w", err)
	}
	var mappingEntries []models.LexwareFieldMappingEntry
	if fieldMappings != nil {
		mappingEntries = fieldMappings.Mappings
	} else {
		mappingEntries = DefaultInvoiceMappings()
	}

	contactLexwareID, err := resolveContactLexwareIDByEmail(ctx, ip.repo, ip.contacts, configID, invoice.CustomerEmail)
	if err != nil {
		return fmt.Errorf("lexware push invoice: resolve contact: %w", err)
	}

	lexwareInvoice, err := ip.fieldMapper.MapInvoiceToLexware(invoice, contactLexwareID, mappingEntries)
	if err != nil {
		return fmt.Errorf("lexware push invoice: map: %w", err)
	}

	now := time.Now().UTC()
	mapping, err := ip.repo.GetEntityMapping(ctx, configID, "invoice", invoiceID)
	if err != nil && !errors.Is(err, ErrMappingNotFound) {
		return fmt.Errorf("lexware push invoice: check mapping: %w", err)
	}

	if mapping != nil {
		lexwareInvoice.Version = mapping.LexwareVersion
		updated, err := ip.client.UpdateInvoice(ctx, tenantID, mapping.LexwareID, lexwareInvoice)
		if err != nil {
			return fmt.Errorf("lexware push invoice: update: %w", err)
		}
		mapping.LastSyncedAt = now
		mapping.KmuhubUpdatedAt = &invoice.UpdatedAt
		mapping.LexwareVersion = updated.Version
		if err := ip.repo.UpsertEntityMapping(ctx, mapping); err != nil {
			slog.Error("lexware: failed to update invoice mapping", "invoice_id", invoiceID, "error", err)
		}
	} else {
		created, err := ip.client.CreateInvoice(ctx, tenantID, lexwareInvoice)
		if err != nil {
			return fmt.Errorf("lexware push invoice: create: %w", err)
		}

		newMapping := &models.LexwareEntityMapping{
			TenantID:        tenantID,
			ConfigID:        configID,
			EntityType:      "invoice",
			KmuhubID:        invoiceID,
			LexwareID:       created.ID,
			LexwareVersion:  created.Version,
			LastSyncedAt:    now,
			KmuhubUpdatedAt: &invoice.UpdatedAt,
			SyncDirection:   "outbound",
		}
		if err := ip.repo.UpsertEntityMapping(ctx, newMapping); err != nil {
			slog.Error("lexware: failed to create invoice mapping", "invoice_id", invoiceID, "error", err)
		}
	}

	syncLog := &models.LexwareSyncLog{
		TenantID:       tenantID,
		ConfigID:       configID,
		SyncType:       "invoice_push",
		Status:         "completed",
		ItemsProcessed: 1,
		ItemsCreated:   boolToInt(mapping == nil),
		ItemsUpdated:   boolToInt(mapping != nil),
		StartedAt:      now,
		CompletedAt:    &now,
		Metadata:       map[string]any{"invoice_id": invoiceID.String()},
	}
	if err := ip.repo.CreateSyncLog(ctx, syncLog); err != nil {
		slog.Error("lexware: failed to create invoice push log", "error", err)
	}

	slog.Info("lexware invoice pushed", "invoice_id", invoiceID, "config_id", configID)
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
