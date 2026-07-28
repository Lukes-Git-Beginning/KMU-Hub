package datev

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// buchungsstapelPageSize bounds read-memory: the rows are pulled keyset-paged,
// only the finished CSV is materialized once.
const buchungsstapelPageSize = 1000

var (
	// ErrBuilderNotConfigured is returned when the builder was constructed
	// without the readers it needs. It is a wiring fault, never a user error.
	ErrBuilderNotConfigured = errors.New("datev: buchungsstapel builder not configured")

	// ErrInvalidPeriod is returned when the requested period is inverted.
	ErrInvalidPeriod = errors.New("datev: end date is before start date")
)

// InvoicePager reads a tenant's exportable invoices for a period, keyset-paged
// by (invoice_date, id). Implemented by invoice.Service.
type InvoicePager interface {
	ListForDATEVExport(ctx context.Context, tenantID uuid.UUID, fromDate, toDate time.Time,
		afterDate *time.Time, afterID *uuid.UUID, limit int) ([]*models.Invoice, error)
}

// CreditNotePager reads a tenant's exportable credit notes for a period,
// keyset-paged by (created_at, id). Implemented by creditnote.Service.
type CreditNotePager interface {
	ListForDATEVExport(ctx context.Context, tenantID uuid.UUID, fromDate, toDate time.Time,
		afterDate *time.Time, afterID *uuid.UUID, limit int) ([]*models.CreditNote, error)
}

// CompanySettingsReader supplies the Berater- and Mandantennummer for the EXTF
// header. Implemented by the company settings repository.
type CompanySettingsReader interface {
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*models.CompanySettings, error)
}

// Buchungsstapel is a rendered DATEV CSV together with what it actually carries.
type Buchungsstapel struct {
	CSV []byte
	// DocumentCount counts the invoices and credit notes that produced booking
	// lines. Drafts are skipped by the exporter and are not counted — a count
	// taken from the loaded rows would overstate what the file contains.
	DocumentCount int
	// LineCount is the number of booking lines (one per exported line item).
	LineCount int
	// BeraterNr and MandantNr are the header numbers the export was built with.
	// They are empty when company settings do not carry them; the download path
	// tolerates that, the upload path refuses (DATEV cannot file the batch).
	BeraterNr string
	MandantNr string
}

// BuchungsstapelBuilder renders a tenant's DATEV Buchungsstapel for a period.
//
// It lives in the service layer rather than in a gRPC handler because two
// callers need exactly the same rows, the same header numbers and the same skip
// rules: the GoBD download (BizGRPCServer.ExportDATEV) and the DATEV API upload
// (UploadService.UploadBuchungsstapel). Two copies of the paging loop would
// drift, and a divergence between what the client downloads and what the tax
// advisor receives is invisible until an audit.
type BuchungsstapelBuilder struct {
	exporter    *Exporter
	invoices    InvoicePager
	creditNotes CreditNotePager
	settings    CompanySettingsReader
}

// NewBuchungsstapelBuilder wires the readers the export needs.
func NewBuchungsstapelBuilder(
	exporter *Exporter,
	invoices InvoicePager,
	creditNotes CreditNotePager,
	settings CompanySettingsReader,
) *BuchungsstapelBuilder {
	return &BuchungsstapelBuilder{
		exporter:    exporter,
		invoices:    invoices,
		creditNotes: creditNotes,
		settings:    settings,
	}
}

// Build renders the Buchungsstapel for one tenant and period. tenantID is passed
// explicitly into every read; nothing is inferred from the context.
func (b *BuchungsstapelBuilder) Build(
	ctx context.Context,
	tenantID uuid.UUID,
	startDate, endDate, fiscalYearStart time.Time,
) (*Buchungsstapel, error) {
	if b == nil || b.exporter == nil || b.invoices == nil || b.creditNotes == nil {
		return nil, ErrBuilderNotConfigured
	}
	if endDate.Before(startDate) {
		return nil, ErrInvalidPeriod
	}

	// Berater-/Mandantennummer are best effort here: an export without them is
	// still a valid file, it just has to be filed manually. Callers that cannot
	// live with that check the returned numbers.
	var beraterNr, mandantNr string
	if b.settings != nil {
		if cs, err := b.settings.GetByTenantID(ctx, tenantID); err == nil && cs != nil {
			beraterNr, mandantNr = cs.DatevBeraterNr, cs.DatevMandantNr
		}
	}

	var buf bytes.Buffer
	sw, err := b.exporter.NewStreamWriter(&buf, beraterNr, mandantNr, fiscalYearStart, time.Now())
	if err != nil {
		return nil, fmt.Errorf("datev export init: %w", err)
	}

	if err := b.writeInvoices(ctx, sw, tenantID, startDate, endDate); err != nil {
		return nil, err
	}
	if err := b.writeCreditNotes(ctx, sw, tenantID, startDate, endDate); err != nil {
		return nil, err
	}

	if err := sw.Close(); err != nil {
		return nil, fmt.Errorf("datev export: %w", err)
	}

	return &Buchungsstapel{
		CSV:           buf.Bytes(),
		DocumentCount: sw.DocumentCount(),
		LineCount:     sw.LineCount(),
		BeraterNr:     beraterNr,
		MandantNr:     mandantNr,
	}, nil
}

// writeInvoices streams the period's invoices into sw, keyset-paged by
// (invoice_date, id).
func (b *BuchungsstapelBuilder) writeInvoices(
	ctx context.Context,
	sw *StreamWriter,
	tenantID uuid.UUID,
	startDate, endDate time.Time,
) error {
	var cursorDate *time.Time
	var cursorID *uuid.UUID
	for {
		page, err := b.invoices.ListForDATEVExport(ctx, tenantID, startDate, endDate, cursorDate, cursorID, buchungsstapelPageSize)
		if err != nil {
			return err
		}
		if len(page) == 0 {
			return nil
		}
		if err := sw.WriteInvoices(page); err != nil {
			return fmt.Errorf("datev export: %w", err)
		}
		last := page[len(page)-1]
		d, id := last.InvoiceDate, last.ID
		cursorDate, cursorID = &d, &id
		if len(page) < buchungsstapelPageSize {
			return nil
		}
	}
}

// writeCreditNotes streams the period's credit notes into sw, keyset-paged by
// (created_at, id).
func (b *BuchungsstapelBuilder) writeCreditNotes(
	ctx context.Context,
	sw *StreamWriter,
	tenantID uuid.UUID,
	startDate, endDate time.Time,
) error {
	var cursorDate *time.Time
	var cursorID *uuid.UUID
	for {
		page, err := b.creditNotes.ListForDATEVExport(ctx, tenantID, startDate, endDate, cursorDate, cursorID, buchungsstapelPageSize)
		if err != nil {
			return err
		}
		if len(page) == 0 {
			return nil
		}
		if err := sw.WriteCreditNotes(page); err != nil {
			return fmt.Errorf("datev export: %w", err)
		}
		last := page[len(page)-1]
		d, id := last.CreatedAt, last.ID
		cursorDate, cursorID = &d, &id
		if len(page) < buchungsstapelPageSize {
			return nil
		}
	}
}
