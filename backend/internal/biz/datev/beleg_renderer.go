package datev

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/biz/pdf"
	"github.com/kmuhub/kmuhub/internal/models"
)

var (
	// ErrInvoiceNotFound is returned when the requested invoice does not exist
	// in this tenant. The gRPC layer maps it to NotFound.
	ErrInvoiceNotFound = errors.New("datev: invoice not found")

	// ErrCompanySettingsIncomplete is returned when the tenant's company settings
	// are absent or miss a Pflichtangabe of section 14 UStG. A Belegbild without
	// them is not a document the tax advisor may book, so it is refused here
	// rather than rendered and sent.
	ErrCompanySettingsIncomplete = errors.New("datev: company settings incomplete")
)

// InvoiceReader loads a single invoice of one tenant. Implemented by
// invoice.Service.
type InvoiceReader interface {
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error)
}

// BelegSource renders the Belegbild that accompanies a booking line.
type BelegSource interface {
	RenderInvoice(ctx context.Context, tenantID, invoiceID uuid.UUID) (data []byte, filename string, err error)
}

// BelegRenderer renders an invoice PDF through the existing maroto generator —
// the same one the client downloads from the finance module, so the tax advisor
// sees the document the customer received, not a second rendering of it.
type BelegRenderer struct {
	invoices InvoiceReader
	settings CompanySettingsReader
}

// NewBelegRenderer wires the readers the renderer needs.
func NewBelegRenderer(invoices InvoiceReader, settings CompanySettingsReader) *BelegRenderer {
	return &BelegRenderer{invoices: invoices, settings: settings}
}

// RenderInvoice returns the invoice PDF and the filename DATEV should store it
// under. Both reads are tenant-scoped through the explicit tenantID.
func (r *BelegRenderer) RenderInvoice(ctx context.Context, tenantID, invoiceID uuid.UUID) ([]byte, string, error) {
	if r == nil || r.invoices == nil || r.settings == nil {
		return nil, "", ErrBuilderNotConfigured
	}

	inv, err := r.invoices.GetByID(ctx, tenantID, invoiceID)
	if err != nil {
		return nil, "", fmt.Errorf("datev: load invoice: %w", err)
	}
	if inv == nil {
		return nil, "", ErrInvoiceNotFound
	}

	settings, err := r.settings.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, "", fmt.Errorf("datev: load company settings: %w", err)
	}
	if settings == nil {
		return nil, "", ErrCompanySettingsIncomplete
	}
	// Checked before rendering so the missing Pflichtangaben reach the client as
	// a fixable precondition instead of a generic render failure.
	if err := pdf.ValidateCompanySettingsForPDF(*settings); err != nil {
		return nil, "", fmt.Errorf("%w: %w", ErrCompanySettingsIncomplete, err)
	}

	data, err := pdf.NewGenerator(*settings).GenerateInvoicePDF(*inv)
	if err != nil {
		return nil, "", fmt.Errorf("datev: render invoice pdf: %w", err)
	}

	return data, fmt.Sprintf("Rechnung_%s.pdf", inv.InvoiceNumber), nil
}
