// Package dunning — GoBD-completion extension methods.
// Implements UpdateDunningStatus, SendDunningNotice, GenerateGoBDExport.
// Added in Sprint 2 / Wave 1.B (buchhaltung-Completion).
package dunning

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// UpdateDunningStatus
// ============================================================================

// UpdateDunningStatus administratively overrides the status of a dunning record.
// Intended for corrections (e.g., marking a sent dunning as withdrawn).
// Allowed transitions are enforced by the caller; this method is intentionally
// permissive to support admin overrides.
func (s *Service) UpdateDunningStatus(ctx context.Context, tenantID, id uuid.UUID, newStatus string) (*models.DunningRecord, error) {
	record, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	// Validate status value
	validStatuses := map[string]bool{
		models.DunningStatusDraft: true,
		models.DunningStatusSent:  true,
		models.DunningStatusPaid:  true,
	}
	if !validStatuses[newStatus] {
		return nil, fmt.Errorf("invalid dunning status %q: must be one of draft, sent, paid", newStatus)
	}

	var sentAt *time.Time
	if newStatus == models.DunningStatusSent && record.Status != models.DunningStatusSent {
		now := time.Now()
		sentAt = &now
	}

	if updateErr := s.repo.UpdateStatus(ctx, tenantID, id, newStatus, sentAt); updateErr != nil {
		return nil, fmt.Errorf("update dunning status: %w", updateErr)
	}

	oldStatus := record.Status
	record.Status = newStatus
	if sentAt != nil {
		record.SentAt = sentAt
	}

	slog.Info("dunning status updated",
		"dunning_id", id,
		"tenant_id", tenantID,
		"old_status", oldStatus,
		"new_status", newStatus,
	)

	return record, nil
}

// ============================================================================
// SendDunningNotice
// ============================================================================

// SendDunningNotice transitions a draft dunning record to sent and sets sent_at.
// Email delivery is a Sprint 3 concern; this Sprint 2 implementation marks the
// record as sent and logs the intent.
//
// TODO Sprint 3: integrate email.Sender to deliver the dunning PDF via SMTP/SES.
func (s *Service) SendDunningNotice(ctx context.Context, tenantID, id, userID uuid.UUID) (*models.DunningRecord, error) {
	record, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if record.Status != models.DunningStatusDraft {
		return nil, ErrDunningNotDraft
	}

	now := time.Now()
	if updateErr := s.repo.UpdateStatus(ctx, tenantID, id, models.DunningStatusSent, &now); updateErr != nil {
		return nil, fmt.Errorf("mark dunning sent: %w", updateErr)
	}

	record.Status = models.DunningStatusSent
	record.SentAt = &now

	slog.Info("dunning notice sent",
		"dunning_id", id,
		"invoice_id", record.InvoiceID,
		"tenant_id", tenantID,
		"level", record.Level,
		"sent_by", userID,
		"sprint3_todo", "email delivery pending",
	)

	return record, nil
}

// ============================================================================
// GenerateGoBDExport
// ============================================================================

// GoBDExportRow represents one posting line in the GoBD CSV export. One row is
// emitted per (invoice, VAT-rate) so each posting carries its own net amount,
// VAT amount, tax key and revenue account — the columns GoBD/IDEA require for a
// machine-auditable journal. The caller builds the rows (see invoicesToGoBDRows
// in the biz gRPC server).
type GoBDExportRow struct {
	InvoiceNumber string
	InvoiceDate   string
	CustomerName  string
	Account       string // SKR03 revenue account (Konto)
	TaxKey        string // DATEV BU-Schluessel (Steuerschluessel)
	TaxRate       string // whole-number percent, e.g. "19"
	NetAmount     string // Nettobetrag
	TaxAmount     string // MwSt-Betrag
	GrossTotal    string // Bruttobetrag (net + VAT)
	Status        string
	TaxMode       string
	BookingText   string // Buchungstext
}

// GoBDExportResult is returned by GenerateGoBDExport.
type GoBDExportResult struct {
	CSVData  []byte
	RowCount int
	FromDate time.Time
	ToDate   time.Time
}

// GenerateGoBDExport produces a GoBD-compliant CSV export from the provided rows.
// The caller (gRPC handler) is responsible for fetching and converting invoice data
// to []GoBDExportRow before calling this method. This design avoids a circular
// dependency between the dunning and invoice packages.
//
// The CSV follows Grundsätze zur ordnungsmäßigen Führung und Aufbewahrung von
// Büchern, Aufzeichnungen und Unterlagen in elektronischer Form (GoBD, BMF 2019).
//
// Column order: BelegNr, BelegDatum, Empfaenger, Konto, Steuerschluessel,
// Steuersatz, Nettobetrag, MwSt, Bruttobetrag, Status, Steuerart, Buchungstext
func (s *Service) GenerateGoBDExport(_ context.Context, tenantID uuid.UUID, fromDate, toDate time.Time, rows []GoBDExportRow) (GoBDExportResult, error) {
	csvData := buildGoBDCSV(rows)

	slog.Info("GoBD export generated",
		"tenant_id", tenantID,
		"from_date", fromDate.Format("2006-01-02"),
		"to_date", toDate.Format("2006-01-02"),
		"row_count", len(rows),
		"bytes", len(csvData),
	)

	return GoBDExportResult{
		CSVData:  csvData,
		RowCount: len(rows),
		FromDate: fromDate,
		ToDate:   toDate,
	}, nil
}

// buildGoBDCSV serialises GoBDExportRows into a UTF-8 CSV with semicolon delimiter
// (standard German accounting CSV format). The output is prefixed with a UTF-8 BOM
// so that DATEV / Lexware / Excel render Umlaute in customer names correctly when
// the CSV is opened locally.
func buildGoBDCSV(rows []GoBDExportRow) []byte {
	var buf bytes.Buffer
	// UTF-8 BOM (EF BB BF) — required by most German accounting tools for Umlaut-safe CSV import.
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	w := csv.NewWriter(&buf)
	w.Comma = ';'

	// Header
	_ = w.Write([]string{
		"BelegNr", "BelegDatum", "Empfaenger", "Konto", "Steuerschluessel",
		"Steuersatz", "Nettobetrag", "MwSt", "Bruttobetrag", "Status",
		"Steuerart", "Buchungstext",
	})

	for _, r := range rows {
		_ = w.Write([]string{
			r.InvoiceNumber,
			r.InvoiceDate,
			r.CustomerName,
			r.Account,
			r.TaxKey,
			r.TaxRate,
			r.NetAmount,
			r.TaxAmount,
			r.GrossTotal,
			r.Status,
			r.TaxMode,
			r.BookingText,
		})
	}
	w.Flush()
	return buf.Bytes()
}
