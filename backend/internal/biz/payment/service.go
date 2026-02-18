// Package payment provides the business logic for recording payments against invoices.
// When payments fully cover the invoice gross total, the invoice status is automatically
// transitioned to "paid". Deleting a payment re-evaluates the paid status.
package payment

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles payment business logic.
type Service struct {
	repo           Repository
	invoiceReader  InvoiceReader
	invoiceUpdater InvoiceStatusUpdater
}

// NewService creates a new payment service.
func NewService(
	repo Repository,
	invoiceReader InvoiceReader,
	invoiceUpdater InvoiceStatusUpdater,
) *Service {
	return &Service{
		repo:           repo,
		invoiceReader:  invoiceReader,
		invoiceUpdater: invoiceUpdater,
	}
}

// RecordInput contains the data needed to record a payment.
type RecordInput struct {
	TenantID  uuid.UUID
	InvoiceID uuid.UUID
	Amount    decimal.Decimal
	Date      time.Time
	Method    string
	Reference string
	Notes     string
	UserID    uuid.UUID
}

// Record records a payment against an invoice. After recording, checks if the sum
// of all payments >= invoice gross_total. If so, transitions the invoice to "paid".
func (s *Service) Record(ctx context.Context, input RecordInput) (*models.Payment, error) {
	// Validate invoice exists and is in a payable state
	inv, err := s.invoiceReader.GetByID(ctx, input.TenantID, input.InvoiceID)
	if err != nil {
		return nil, fmt.Errorf("get invoice: %w", err)
	}

	switch inv.Status {
	case models.InvoiceStatusSent, models.InvoiceStatusOverdue:
		// Valid states for payment
	case models.InvoiceStatusPaid:
		// Allow overpayment recording (partial refund scenario)
		slog.Warn("recording payment on already-paid invoice",
			"invoice_id", input.InvoiceID,
			"tenant_id", input.TenantID,
		)
	default:
		return nil, fmt.Errorf("cannot record payment on invoice with status %s", inv.Status)
	}

	if input.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("payment amount must be positive")
	}

	now := time.Now()
	paymentDate := input.Date
	if paymentDate.IsZero() {
		paymentDate = now
	}

	payment := &models.Payment{
		ID:          uuid.New(),
		TenantID:    input.TenantID,
		InvoiceID:   input.InvoiceID,
		Amount:      input.Amount,
		PaymentDate: paymentDate,
		Method:      input.Method,
		Reference:   input.Reference,
		Notes:       input.Notes,
		CreatedBy:   input.UserID,
		CreatedAt:   now,
	}

	if createErr := s.repo.Create(ctx, payment); createErr != nil {
		return nil, createErr
	}

	slog.Info("payment recorded",
		"payment_id", payment.ID,
		"invoice_id", payment.InvoiceID,
		"tenant_id", payment.TenantID,
		"amount", payment.Amount,
		"method", payment.Method,
	)

	// Check if invoice is now fully paid
	if err := s.checkAndTransitionToPaid(ctx, input.TenantID, input.InvoiceID, inv); err != nil {
		slog.Warn("failed to check/transition invoice to paid",
			"invoice_id", input.InvoiceID,
			"error", err,
		)
	}

	return payment, nil
}

// List retrieves all payments for a given invoice.
func (s *Service) List(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]*models.Payment, error) {
	return s.repo.List(ctx, tenantID, invoiceID)
}

// Delete removes a payment. After deletion, re-checks if the invoice should revert
// from paid to its previous state (sent or overdue based on due date).
func (s *Service) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	// Get the payment to find the associated invoice
	payment, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	// Get the invoice to check state
	inv, invErr := s.invoiceReader.GetByID(ctx, tenantID, payment.InvoiceID)
	if invErr != nil {
		return fmt.Errorf("get invoice for payment deletion: %w", invErr)
	}

	// Only allow deletion if invoice is not cancelled
	if inv.Status == models.InvoiceStatusCancelled {
		return fmt.Errorf("cannot delete payment on cancelled invoice")
	}

	if delErr := s.repo.Delete(ctx, tenantID, id); delErr != nil {
		return delErr
	}

	slog.Info("payment deleted",
		"payment_id", id,
		"invoice_id", payment.InvoiceID,
		"tenant_id", tenantID,
	)

	// Re-evaluate invoice status after payment deletion
	if inv.Status == models.InvoiceStatusPaid {
		if revertErr := s.revertPaidStatus(ctx, tenantID, payment.InvoiceID, inv); revertErr != nil {
			slog.Warn("failed to revert invoice paid status",
				"invoice_id", payment.InvoiceID,
				"error", revertErr,
			)
		}
	}

	return nil
}

// checkAndTransitionToPaid checks if the sum of payments covers the invoice total
// and transitions to paid if so.
func (s *Service) checkAndTransitionToPaid(ctx context.Context, tenantID, invoiceID uuid.UUID, inv *models.Invoice) error {
	// Skip if already paid
	if inv.Status == models.InvoiceStatusPaid {
		return nil
	}

	totalPaid, err := s.repo.SumByInvoiceID(ctx, tenantID, invoiceID)
	if err != nil {
		return fmt.Errorf("sum payments: %w", err)
	}

	if totalPaid.GreaterThanOrEqual(inv.GrossTotal) {
		if updateErr := s.invoiceUpdater.UpdateStatus(ctx, tenantID, invoiceID, models.InvoiceStatusPaid); updateErr != nil {
			return fmt.Errorf("transition to paid: %w", updateErr)
		}
		slog.Info("invoice auto-transitioned to paid",
			"invoice_id", invoiceID,
			"tenant_id", tenantID,
			"total_paid", totalPaid,
			"gross_total", inv.GrossTotal,
		)
	}

	return nil
}

// revertPaidStatus reverts an invoice from paid to sent or overdue
// when a payment is deleted and the total no longer covers the invoice amount.
func (s *Service) revertPaidStatus(ctx context.Context, tenantID, invoiceID uuid.UUID, inv *models.Invoice) error {
	totalPaid, err := s.repo.SumByInvoiceID(ctx, tenantID, invoiceID)
	if err != nil {
		return fmt.Errorf("sum payments: %w", err)
	}

	if totalPaid.LessThan(inv.GrossTotal) {
		// Determine correct status: overdue if past due date, otherwise sent
		newStatus := models.InvoiceStatusSent
		if time.Now().After(inv.DueDate) {
			newStatus = models.InvoiceStatusOverdue
		}

		if updateErr := s.invoiceUpdater.UpdateStatus(ctx, tenantID, invoiceID, newStatus); updateErr != nil {
			return fmt.Errorf("revert from paid: %w", updateErr)
		}
		slog.Info("invoice reverted from paid",
			"invoice_id", invoiceID,
			"tenant_id", tenantID,
			"new_status", newStatus,
			"total_paid", totalPaid,
			"gross_total", inv.GrossTotal,
		)
	}

	return nil
}
