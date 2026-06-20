// Package payment provides the business logic for recording payments against invoices.
// When payments fully cover the invoice gross total, the invoice status is automatically
// transitioned to "paid". Deleting a payment re-evaluates the paid status.
package payment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// txBeginner begins a transaction. Satisfied by *pgxpool.Pool in production and a
// no-op fake in unit tests so the orchestration runs without a database (real
// atomicity is covered by the integration test).
type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Service handles payment business logic.
type Service struct {
	repo           Repository
	invoiceReader  InvoiceReader
	invoiceUpdater InvoiceStatusUpdater
	// pool backs the atomic Record/Delete transactions (payment write + invoice
	// status transition in one tx). Required for Record and Delete.
	pool txBeginner
}

// NewService creates a new payment service. pool is required for Record/Delete so
// the payment write and the coupled invoice status transition commit atomically.
func NewService(
	repo Repository,
	invoiceReader InvoiceReader,
	invoiceUpdater InvoiceStatusUpdater,
	pool txBeginner,
) *Service {
	return &Service{
		repo:           repo,
		invoiceReader:  invoiceReader,
		invoiceUpdater: invoiceUpdater,
		pool:           pool,
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
	// IdempotencyKey is the client Idempotency-Key (when present) used for
	// DB-level deduplication so a retried submission records exactly one payment.
	IdempotencyKey string
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
		ID:             uuid.New(),
		TenantID:       input.TenantID,
		InvoiceID:      input.InvoiceID,
		Amount:         input.Amount,
		PaymentDate:    paymentDate,
		Method:         input.Method,
		Reference:      input.Reference,
		Notes:          input.Notes,
		CreatedBy:      input.UserID,
		CreatedAt:      now,
		IdempotencyKey: input.IdempotencyKey,
	}

	if s.pool == nil {
		return nil, fmt.Errorf("payment service: pool not configured (required for Record)")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if createErr := s.repo.CreateInTx(ctx, tx, payment); createErr != nil {
		// DB-level idempotency (F5): a duplicate key means this payment was already
		// recorded; its original transaction also ran the status transition, so we
		// drop this tx and return the existing payment without re-applying side effects.
		if errors.Is(createErr, ErrDuplicatePayment) {
			existing, getErr := s.repo.GetByIdempotencyKey(ctx, input.TenantID, input.IdempotencyKey)
			if getErr != nil {
				return nil, getErr
			}
			if existing != nil {
				slog.Info("payment idempotent replay (duplicate idempotency key)",
					"payment_id", existing.ID,
					"invoice_id", existing.InvoiceID,
					"tenant_id", existing.TenantID,
				)
				return existing, nil
			}
		}
		return nil, createErr
	}

	// Transition the invoice to paid within the same transaction so the payment and
	// the coupled status change commit atomically — no status drift on partial failure.
	if err := s.transitionToPaidInTx(ctx, tx, input.TenantID, input.InvoiceID, inv); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	slog.Info("payment recorded",
		"payment_id", payment.ID,
		"invoice_id", payment.InvoiceID,
		"tenant_id", payment.TenantID,
		"amount", payment.Amount,
		"method", payment.Method,
	)

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

	if s.pool == nil {
		return fmt.Errorf("payment service: pool not configured (required for Delete)")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if delErr := s.repo.DeleteInTx(ctx, tx, tenantID, id); delErr != nil {
		return delErr
	}

	// Re-evaluate the invoice status within the same transaction so the deletion and
	// the coupled revert commit atomically — no drift on partial failure.
	if inv.Status == models.InvoiceStatusPaid {
		if revertErr := s.revertPaidStatusInTx(ctx, tx, tenantID, payment.InvoiceID, inv); revertErr != nil {
			return revertErr
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	slog.Info("payment deleted",
		"payment_id", id,
		"invoice_id", payment.InvoiceID,
		"tenant_id", tenantID,
	)

	return nil
}

// transitionToPaidInTx checks whether payments now cover the invoice total and, if
// so, transitions the invoice to paid within the caller's transaction (atomic with
// the payment write).
func (s *Service) transitionToPaidInTx(ctx context.Context, tx pgx.Tx, tenantID, invoiceID uuid.UUID, inv *models.Invoice) error {
	// Skip if already paid
	if inv.Status == models.InvoiceStatusPaid {
		return nil
	}

	totalPaid, err := s.repo.SumByInvoiceIDInTx(ctx, tx, tenantID, invoiceID)
	if err != nil {
		return fmt.Errorf("sum payments: %w", err)
	}

	if totalPaid.GreaterThanOrEqual(inv.GrossTotal) {
		if updateErr := s.invoiceUpdater.UpdateStatusInTx(ctx, tx, tenantID, invoiceID, models.InvoiceStatusPaid); updateErr != nil {
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

// revertPaidStatusInTx reverts an invoice from paid to sent or overdue when a
// payment is deleted and the remaining total no longer covers the invoice amount,
// within the caller's transaction (atomic with the deletion).
func (s *Service) revertPaidStatusInTx(ctx context.Context, tx pgx.Tx, tenantID, invoiceID uuid.UUID, inv *models.Invoice) error {
	totalPaid, err := s.repo.SumByInvoiceIDInTx(ctx, tx, tenantID, invoiceID)
	if err != nil {
		return fmt.Errorf("sum payments: %w", err)
	}

	if totalPaid.LessThan(inv.GrossTotal) {
		// Determine correct status: overdue if past due date, otherwise sent
		newStatus := models.InvoiceStatusSent
		if time.Now().After(inv.DueDate) {
			newStatus = models.InvoiceStatusOverdue
		}

		if updateErr := s.invoiceUpdater.UpdateStatusInTx(ctx, tx, tenantID, invoiceID, newStatus); updateErr != nil {
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
