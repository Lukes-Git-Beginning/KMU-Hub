// Package invoice — GoBD-compliance extension methods.
// Implements GetJournalSummary, ValidateInvoiceNumber, LockInvoice, GetPaymentStats.
// Added in Sprint 2 / Wave 1.B (buchhaltung-Completion).
package invoice

import (
	"context"
	"encoding/json"
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
// The lock is recorded in snapshot_data JSONB until Sprint 4 adds a dedicated
// locked_at TIMESTAMPTZ column (migration sprint4_add_invoice_locked_at not yet created).
//
// Only sent, paid, or overdue invoices can be locked (draft invoices should be
// cancelled or sent first).
func (s *Service) LockInvoice(ctx context.Context, tenantID, id, lockedBy uuid.UUID) (LockInvoiceResult, error) {
	inv, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return LockInvoiceResult{}, err
	}

	// Only immutable (post-send) invoices can be locked
	if inv.Status == "draft" {
		return LockInvoiceResult{}, fmt.Errorf("only sent/paid/overdue invoices can be locked (GoBD: lock applies to completed documents)")
	}

	// Parse existing snapshot_data once. A corrupt JSONB blob must NOT be silently
	// overwritten — that would destroy whatever the snapshot was previously holding.
	var snap map[string]any
	if len(inv.SnapshotData) > 0 {
		if parseErr := json.Unmarshal(inv.SnapshotData, &snap); parseErr != nil {
			return LockInvoiceResult{}, fmt.Errorf("invoice snapshot_data is corrupt for id %s: %w", id, parseErr)
		}
		if _, ok := snap["locked_at"]; ok {
			return LockInvoiceResult{}, ErrInvoiceLocked
		}
	}

	// Record lock in snapshot_data JSONB
	// TODO Sprint 4: replace with dedicated locked_at/locked_by columns once migration is applied.
	now := time.Now()
	if snap == nil {
		snap = make(map[string]any)
	}
	snap["locked_at"] = now.Format(time.RFC3339)
	snap["locked_by"] = lockedBy.String()

	snapJSON, marshalErr := json.Marshal(snap)
	if marshalErr != nil {
		return LockInvoiceResult{}, fmt.Errorf("marshal lock snapshot: %w", marshalErr)
	}
	inv.SnapshotData = snapJSON
	inv.UpdatedAt = now

	if updateErr := s.repo.Update(ctx, inv); updateErr != nil {
		return LockInvoiceResult{}, fmt.Errorf("persist lock: %w", updateErr)
	}

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

// isInvoiceLocked reports whether inv carries a locked_at marker in its
// snapshot_data JSONB blob.
//
// Design contract (different from LockInvoice's own parse path):
//   - Corrupt snapshot_data is treated as "not locked" here so that write
//     guards never hard-block on a corrupt blob. The corruption is logged for
//     ops awareness. LockInvoice itself errors hard on corruption to prevent
//     silent overwrites.
//   - An empty snapshot_data blob means no lock has been recorded → false.
func isInvoiceLocked(inv *models.Invoice) bool {
	if len(inv.SnapshotData) == 0 {
		return false
	}
	var snap map[string]any
	if err := json.Unmarshal(inv.SnapshotData, &snap); err != nil {
		slog.Warn("invoice snapshot_data is corrupt — treating as unlocked",
			"invoice_id", inv.ID,
			"error", err,
		)
		return false
	}
	_, locked := snap["locked_at"]
	return locked
}
