package datev

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

type invoiceReaderStub struct {
	invoice *models.Invoice
	err     error
	tenant  uuid.UUID
	id      uuid.UUID
}

func (s *invoiceReaderStub) GetByID(_ context.Context, tenantID, id uuid.UUID) (*models.Invoice, error) {
	s.tenant, s.id = tenantID, id
	return s.invoice, s.err
}

func TestBelegRenderer_ProducesRealPDF(t *testing.T) {
	// The point of this test is that a PDF is actually rendered — the endpoint it
	// serves used to report a transferred Belegbild without ever producing one.
	reader := &invoiceReaderStub{invoice: sentInvoice("RE-2026-0007")}
	renderer := NewBelegRenderer(reader, &settingsStub{settings: &models.CompanySettings{
		Name:         "Test GmbH",
		Street:       "Teststrasse 1",
		PLZ:          "10115",
		City:         "Berlin",
		Steuernummer: "12/345/67890",
		AccentColor:  "#1a73e8",
	}})

	tenantID, invoiceID := uuid.New(), uuid.New()
	data, filename, err := renderer.RenderInvoice(context.Background(), tenantID, invoiceID)
	if err != nil {
		t.Fatalf("RenderInvoice: %v", err)
	}

	if !bytes.HasPrefix(data, []byte("%PDF")) {
		t.Errorf("rendered %d bytes that are not a PDF", len(data))
	}
	if filename != "Rechnung_RE-2026-0007.pdf" {
		t.Errorf("filename = %q, want Rechnung_RE-2026-0007.pdf", filename)
	}
	if reader.tenant != tenantID || reader.id != invoiceID {
		t.Error("the invoice read was not scoped to the requested tenant and id")
	}
}

func TestBelegRenderer_MissingPreconditions(t *testing.T) {
	tests := []struct {
		name     string
		reader   *invoiceReaderStub
		settings *settingsStub
		want     error
	}{
		{
			name:     "invoice does not exist in this tenant",
			reader:   &invoiceReaderStub{invoice: nil},
			settings: &settingsStub{settings: &models.CompanySettings{}},
			want:     ErrInvoiceNotFound,
		},
		{
			name:     "company settings not configured",
			reader:   &invoiceReaderStub{invoice: sentInvoice("RE-1")},
			settings: &settingsStub{settings: nil},
			want:     ErrCompanySettingsIncomplete,
		},
		{
			// Rendering would otherwise emit an invoice without the section 14
			// UStG Pflichtangaben and send it to the tax advisor.
			name:     "company settings miss a Pflichtangabe",
			reader:   &invoiceReaderStub{invoice: sentInvoice("RE-1")},
			settings: &settingsStub{settings: &models.CompanySettings{Name: "Test GmbH"}},
			want:     ErrCompanySettingsIncomplete,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := NewBelegRenderer(tt.reader, tt.settings).RenderInvoice(context.Background(), uuid.New(), uuid.New())
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}
