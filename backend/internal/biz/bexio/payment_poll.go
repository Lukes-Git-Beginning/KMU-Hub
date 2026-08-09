package bexio

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// InvoiceStatusUpdater provides the ability to mark invoices as paid.
type InvoiceStatusUpdater interface {
	MarkPaid(ctx context.Context, tenantID, id uuid.UUID) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error)
}

// PaymentPoller polls Bexio for invoice payment status updates.
type PaymentPoller struct {
	client   *Client
	repo     Repository
	invoices InvoiceStatusUpdater
}

// NewPaymentPoller creates a new payment poller.
func NewPaymentPoller(client *Client, repo Repository, invoices InvoiceStatusUpdater) *PaymentPoller {
	return &PaymentPoller{
		client:   client,
		repo:     repo,
		invoices: invoices,
	}
}

// PollPayments checks payment status for all mapped invoices.
func (pp *PaymentPoller) PollPayments(ctx context.Context, configID, tenantID uuid.UUID) (*SyncResult, error) {
	result := &SyncResult{}

	now := time.Now().UTC()
	syncLog := &models.BexioSyncLog{
		TenantID:  tenantID,
		ConfigID:  configID,
		SyncType:  "payment_poll",
		Status:    "running",
		StartedAt: now,
		Metadata:  map[string]any{},
	}
	if err := pp.repo.CreateSyncLog(ctx, syncLog); err != nil {
		slog.Error("bexio: failed to create payment poll log", "error", err)
	}

	// List all invoice entity mappings
	mappings, err := pp.repo.ListEntityMappings(ctx, configID, "invoice")
	if err != nil {
		return nil, fmt.Errorf("bexio payment poll: list mappings: %w", err)
	}

	for _, mapping := range mappings {
		// Skip already-paid invoices in KMU Hub
		invoice, err := pp.invoices.GetByID(ctx, tenantID, mapping.KmuhubID)
		if err != nil {
			result.ItemsFailed++
			result.Errors = append(result.Errors, fmt.Sprintf("load invoice %s: %v", mapping.KmuhubID, err))
			continue
		}
		if invoice.Status == "paid" {
			continue // Already settled
		}

		result.ItemsProcessed++

		// Fetch payments from Bexio
		payments, err := pp.client.ListInvoicePayments(ctx, tenantID, mapping.BexioID)
		if err != nil {
			result.ItemsFailed++
			result.Errors = append(result.Errors, fmt.Sprintf("poll bexio invoice %d payments: %v", mapping.BexioID, err))
			continue
		}

		// Calculate total paid
		totalPaid := decimal.Zero
		for _, p := range payments {
			amount, err := decimal.NewFromString(p.Value)
			if err != nil {
				continue
			}
			totalPaid = totalPaid.Add(amount)
		}

		// Compare with invoice gross total
		if totalPaid.GreaterThanOrEqual(invoice.GrossTotal) {
			// Fully paid — mark as paid in KMU Hub
			if err := pp.invoices.MarkPaid(ctx, tenantID, mapping.KmuhubID); err != nil {
				result.ItemsFailed++
				result.Errors = append(result.Errors, fmt.Sprintf("mark invoice %s paid: %v", mapping.KmuhubID, err))
				continue
			}
			result.ItemsUpdated++

			slog.Info("bexio payment: invoice marked paid",
				"invoice_id", mapping.KmuhubID,
				"bexio_invoice_id", mapping.BexioID,
				"total_paid", totalPaid.String(),
			)
		}
	}

	// Update last poll time
	completeTime := time.Now().UTC()
	if err := pp.repo.UpdateLastSyncTime(ctx, configID, "payment_poll", completeTime); err != nil {
		slog.Error("bexio: failed to update last payment poll time", "error", err)
	}

	// Finalize sync log
	syncLog.ItemsProcessed = result.ItemsProcessed
	syncLog.ItemsUpdated = result.ItemsUpdated
	syncLog.ItemsFailed = result.ItemsFailed
	syncLog.CompletedAt = &completeTime
	if result.ItemsFailed > 0 {
		syncLog.Status = "partial"
	} else {
		syncLog.Status = "completed"
	}
	if err := pp.repo.UpdateSyncLog(ctx, syncLog); err != nil {
		slog.Error("bexio: failed to update payment poll log", "error", err)
	}

	slog.Info("bexio payment poll completed",
		"config_id", configID,
		"processed", result.ItemsProcessed,
		"updated", result.ItemsUpdated,
		"failed", result.ItemsFailed,
	)

	return result, nil
}
