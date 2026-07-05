// Package invoice — GoBD-compliance extension methods.
// Implements GetJournalSummary, ValidateInvoiceNumber, LockInvoice, GetPaymentStats.
// Added in Sprint 2 / Wave 1.B (buchhaltung-Completion).
package invoice

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// invoiceNumberPattern matches RE-YYYY-NNNN (prefix RE, 4-digit year, 4+ digit seq).
var invoiceNumberPattern = regexp.MustCompile(`^([A-Z]{2})-(\d{4})-(\d{4,})$`)

// ============================================================================
// ValidateInvoiceNumber
// ============================================================================

// ValidateInvoiceNumberResult carries the outcome of invoice number validation.
type ValidateInvoiceNumberResult struct {
	ValidFormat bool
	AlreadyUsed bool
	Canonical   string // Empty when format is invalid.
}

// ValidateInvoiceNumber checks whether a candidate invoice number is:
//  1. Format-valid (matches RE-YYYY-NNNN)
//  2. Not yet assigned to any invoice in the tenant
func (s *Service) ValidateInvoiceNumber(ctx context.Context, tenantID uuid.UUID, invoiceNumber string) (ValidateInvoiceNumberResult, error) {
	m := invoiceNumberPattern.FindStringSubmatch(invoiceNumber)
	if m == nil {
		return ValidateInvoiceNumberResult{ValidFormat: false}, nil
	}

	// Canonical form: uppercase, zero-padded to 4 digits minimum
	prefix := m[1]
	year := m[2]
	seq, _ := strconv.Atoi(m[3])
	canonical := fmt.Sprintf("%s-%s-%04d", prefix, year, seq)

	// Check whether the number is already in use
	used, err := s.repo.InvoiceNumberExists(ctx, tenantID, canonical)
	if err != nil {
		return ValidateInvoiceNumberResult{}, fmt.Errorf("check invoice number: %w", err)
	}

	return ValidateInvoiceNumberResult{
		ValidFormat: true,
		AlreadyUsed: used,
		Canonical:   canonical,
	}, nil
}

// ============================================================================
// LockInvoice
// ============================================================================

// ErrInvoiceLocked is returned when trying to lock an already-locked invoice.
var ErrInvoiceLocked = fmt.Errorf("invoice is already locked")

// LockInvoiceResult carries the locked invoice and lock timestamp.
type LockInvoiceResult struct {
	Invoice  any // *models.Invoice
	LockedAt time.Time
}

// LockInvoice administratively locks an invoice, preventing any further modifications.
// The lock is persisted in the dedicated locked_at / locked_by columns introduced
// by Migration 000132 (ADR-0007).
//
// Only sent, paid, or overdue invoices can be locked (draft invoices should be
// cancelled or sent first).
func (s *Service) LockInvoice(ctx context.Context, tenantID, id, lockedBy uuid.UUID) (LockInvoiceResult, error) {
	inv, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return LockInvoiceResult{}, err
	}

	// Imported (source='bexio') invoices are read-only mirrors of the external book.
	if inv.Source == models.InvoiceSourceBexio {
		return LockInvoiceResult{}, ErrExternalReadOnly
	}

	// Only immutable (post-send) invoices can be locked
	if inv.Status == "draft" {
		return LockInvoiceResult{}, fmt.Errorf("only sent/paid/overdue invoices can be locked (GoBD: lock applies to completed documents)")
	}

	if inv.LockedAt != nil {
		return LockInvoiceResult{}, ErrInvoiceLocked
	}

	now := time.Now()

	if setErr := s.repo.SetLock(ctx, tenantID, id, now, lockedBy); setErr != nil {
		return LockInvoiceResult{}, fmt.Errorf("persist lock: %w", setErr)
	}

	// Update in-memory representation so callers see the locked state.
	inv.LockedAt = &now
	inv.LockedBy = &lockedBy

	slog.Info("invoice locked",
		"invoice_id", id,
		"tenant_id", tenantID,
		"locked_by", lockedBy,
		"locked_at", now,
	)

	return LockInvoiceResult{Invoice: inv, LockedAt: now}, nil
}

// ============================================================================
// GetJournalSummary
// ============================================================================

// JournalSummary contains GoBD journal compliance information for a fiscal year.
type JournalSummary struct {
	Year                int
	TotalInvoicesIssued int
	GapsDetected        int    // Should always be 0 for GoBD compliance
	FirstNumber         string // e.g. "RE-2026-0001"
	LastNumber          string // e.g. "RE-2026-0042"
	HighestSequence     int
}

// GetJournalSummary reports the sequential numbering state for a fiscal year.
// GoBD Grundsätze ordnungsgemäßer Buchführung require a gap-free journal.
// GapsDetected should always be 0; any positive value is a compliance finding.
func (s *Service) GetJournalSummary(ctx context.Context, tenantID uuid.UUID, year int) (JournalSummary, error) {
	seq, err := s.numberSeqRepo.GetSequenceInfo(ctx, tenantID, "invoice", year)
	if err != nil {
		return JournalSummary{}, fmt.Errorf("get sequence info: %w", err)
	}

	if seq == nil {
		// No invoices issued in this year
		return JournalSummary{Year: year}, nil
	}

	prefix := "RE"
	first := fmt.Sprintf("%s-%d-%04d", prefix, year, 1)
	last := fmt.Sprintf("%s-%d-%04d", prefix, year, seq.CurrentNumber)

	// Gap detection: compare assigned count vs. sequence counter.
	// A gap exists when current_number > actual issued invoices.
	// current_number is the SEQUENCE counter; we need the actual invoice count.
	invoiceCount, countErr := s.repo.CountByFiscalYear(ctx, tenantID, year)
	if countErr != nil {
		return JournalSummary{}, fmt.Errorf("count invoices: %w", countErr)
	}

	gaps := max(seq.CurrentNumber-invoiceCount, 0)

	return JournalSummary{
		Year:                year,
		TotalInvoicesIssued: invoiceCount,
		GapsDetected:        gaps,
		FirstNumber:         first,
		LastNumber:          last,
		HighestSequence:     seq.CurrentNumber,
	}, nil
}

// ============================================================================
// GetPaymentStats
// ============================================================================

// GetPaymentStats returns aggregated payment statistics for invoices in the date range.
// TotalPaid/Outstanding are invoice counts, not amounts.
// AverageDaysToPay is the mean of (payment_date - invoice_date) for paid invoices.
func (s *Service) GetPaymentStats(ctx context.Context, tenantID uuid.UUID, fromDate, toDate time.Time) (PaymentStats, error) {
	stats, err := s.repo.AggregatePaymentStats(ctx, tenantID, fromDate, toDate)
	if err != nil {
		return PaymentStats{}, fmt.Errorf("aggregate payment stats: %w", err)
	}
	return stats, nil
}

// ListForGoBDExport returns non-draft invoices with invoice_number assigned within
// the date range. Used by the gRPC handler to populate the GoBD CSV export.
func (s *Service) ListForGoBDExport(ctx context.Context, tenantID uuid.UUID, fromDate, toDate time.Time) ([]*models.Invoice, error) {
	return s.repo.ListForGoBDExport(ctx, tenantID, fromDate, toDate)
}

// ============================================================================
// Lock-state helper
// ============================================================================

// isInvoiceLocked reports whether inv is administratively locked.
// Uses the dedicated locked_at column (ADR-0007 / Migration 000132).
func isInvoiceLocked(inv *models.Invoice) bool {
	return inv.LockedAt != nil
}
