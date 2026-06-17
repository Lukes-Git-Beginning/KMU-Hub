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

// InvoiceReader provides read access to KMU Hub invoices for the push engine.
type InvoiceReader interface {
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error)
}

// InvoicePusher pushes KMU Hub invoices to Bexio.
type InvoicePusher struct {
	client      *Client
	repo        Repository
	fieldMapper *FieldMapper
	invoices    InvoiceReader
	contacts    ContactService
}

// NewInvoicePusher creates a new invoice pusher.
func NewInvoicePusher(client *Client, repo Repository, fieldMapper *FieldMapper, invoices InvoiceReader, contacts ContactService) *InvoicePusher {
	return &InvoicePusher{
		client:      client,
		repo:        repo,
		fieldMapper: fieldMapper,
		invoices:    invoices,
		contacts:    contacts,
	}
}

// PushInvoice pushes a single invoice to Bexio.
func (ip *InvoicePusher) PushInvoice(ctx context.Context, configID, tenantID, invoiceID uuid.UUID) error {
	// Load invoice
	invoice, err := ip.invoices.GetByID(ctx, tenantID, invoiceID)
	if err != nil {
		return fmt.Errorf("bexio push invoice: load invoice: %w", err)
	}

	// Load field mappings
	fieldMappings, err := ip.repo.GetFieldMappings(ctx, configID, "invoice")
	if err != nil {
		return fmt.Errorf("bexio push invoice: get field mappings: %w", err)
	}
	var mappingEntries []models.BexioFieldMappingEntry
	if fieldMappings != nil {
		mappingEntries = fieldMappings.Mappings
	} else {
		mappingEntries = DefaultInvoiceMappings()
	}

	// Resolve contact mapping — invoice must have a synced contact
	contactBexioID, err := ip.resolveContactBexioID(ctx, configID, invoice)
	if err != nil {
		return fmt.Errorf("bexio push invoice: resolve contact: %w", err)
	}

	// Map to Bexio format
	bexioInvoice, err := ip.fieldMapper.MapInvoiceToBexio(invoice, contactBexioID, mappingEntries)
	if err != nil {
		return fmt.Errorf("bexio push invoice: map: %w", err)
	}

	// Check existing entity mapping
	now := time.Now().UTC()
	mapping, err := ip.repo.GetEntityMapping(ctx, configID, "invoice", invoiceID)
	if err != nil && !errors.Is(err, ErrMappingNotFound) {
		return fmt.Errorf("bexio push invoice: check mapping: %w", err)
	}

	if mapping != nil {
		// Update existing
		if _, err := ip.client.UpdateInvoice(ctx, tenantID, mapping.BexioID, bexioInvoice); err != nil {
			return fmt.Errorf("bexio push invoice: update in bexio: %w", err)
		}
		mapping.LastSyncedAt = now
		mapping.KmuhubUpdatedAt = &invoice.UpdatedAt
		if err := ip.repo.UpsertEntityMapping(ctx, mapping); err != nil {
			slog.Error("bexio: failed to update invoice mapping", "invoice_id", invoiceID, "error", err)
		}
	} else {
		// Create new
		created, err := ip.client.CreateInvoice(ctx, tenantID, bexioInvoice)
		if err != nil {
			return fmt.Errorf("bexio push invoice: create in bexio: %w", err)
		}

		newMapping := &models.BexioEntityMapping{
			ConfigID:        configID,
			EntityType:      "invoice",
			KmuhubID:        invoiceID,
			BexioID:         created.ID,
			LastSyncedAt:    now,
			KmuhubUpdatedAt: &invoice.UpdatedAt,
			SyncDirection:   "outbound",
		}
		if err := ip.repo.UpsertEntityMapping(ctx, newMapping); err != nil {
			slog.Error("bexio: failed to create invoice mapping", "invoice_id", invoiceID, "error", err)
		}
	}

	// Log sync
	syncLog := &models.BexioSyncLog{
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
		slog.Error("bexio: failed to create invoice push log", "error", err)
	}

	slog.Info("bexio invoice pushed",
		"invoice_id", invoiceID,
		"config_id", configID,
	)

	return nil
}

func (ip *InvoicePusher) resolveContactBexioID(ctx context.Context, configID uuid.UUID, invoice *models.Invoice) (int, error) {
	return resolveContactBexioIDByEmail(ctx, ip.repo, ip.contacts, configID, invoice.CustomerEmail)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
