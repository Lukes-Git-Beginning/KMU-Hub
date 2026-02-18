package pdf

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/pagesize"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Generator creates PDF documents for financial records.
type Generator struct {
	settings models.CompanySettings
}

// NewGenerator creates a new PDF generator with company settings for defaults.
func NewGenerator(settings models.CompanySettings) *Generator {
	return &Generator{settings: settings}
}

// newMaroto creates a configured maroto instance with A4 page, 15mm margins, and Pflichtangaben footer.
func (g *Generator) newMaroto() core.Maroto {
	accentColor := parseColor(g.settings.AccentColor)

	cfg := config.NewBuilder().
		WithPageSize(pagesize.A4).
		WithLeftMargin(15).
		WithTopMargin(15).
		WithRightMargin(15).
		WithPageNumber().
		Build()

	mrt := maroto.New(cfg)

	// Register footer with Pflichtangaben on every page
	footerRow := buildFooter(g.settings)
	if err := mrt.RegisterFooter(footerRow); err != nil {
		slog.Warn("failed to register PDF footer", "error", err)
	}

	// Register header with accent line
	headerRow := buildAccentLine(accentColor)
	if err := mrt.RegisterHeader(headerRow); err != nil {
		slog.Warn("failed to register PDF header", "error", err)
	}

	return mrt
}

// GenerateQuotePDF generates a PDF for a quote (Angebot).
func (g *Generator) GenerateQuotePDF(quote models.Quote) ([]byte, error) {
	m := g.newMaroto()
	accentColor := parseColor(g.settings.AccentColor)

	// Company header
	m.AddRows(buildHeader(g.settings, accentColor))
	m.AddRows(row.New(5)) // spacer

	// Recipient
	m.AddRows(buildRecipient(quote.CustomerName, quote.CustomerAddress))
	m.AddRows(row.New(5)) // spacer

	// Document title
	dateStr := quote.CreatedAt.Format("02.01.2006")
	m.AddRows(buildDocumentTitle("Angebot", quote.QuoteNumber, dateStr, accentColor))
	m.AddRows(row.New(3))

	// Line items
	lineItems, err := parseLineItems(quote.LineItems)
	if err != nil {
		return nil, fmt.Errorf("parse quote line items: %w", err)
	}

	m.AddRows(buildLineItemHeader())
	for i, item := range lineItems {
		pos := i + 1
		if item.Position > 0 {
			pos = item.Position
		}
		m.AddRows(buildLineItemRow(pos, item, i%2 == 1))
	}

	// Totals
	breakdown, err := parseTaxBreakdown(quote.TaxBreakdownRaw)
	if err != nil {
		return nil, fmt.Errorf("parse quote tax breakdown: %w", err)
	}
	if breakdown == nil {
		breakdown = &models.TaxBreakdown{
			Subtotal:   quote.Subtotal,
			TotalTax:   quote.TotalTax,
			GrossTotal: quote.GrossTotal,
			TaxByRate:  make(map[string]decimal.Decimal),
		}
	}
	m.AddRows(buildTotalsSection(*breakdown, quote.TaxMode)...)

	// Validity note
	if quote.ValidUntil != nil {
		m.AddRows(row.New(3))
		validStr := quote.ValidUntil.Format("02.01.2006")
		m.AddRows(row.New(6).Add(
			buildTextCol(12, fmt.Sprintf("Dieses Angebot ist gueltig bis %s.", validStr), 8),
		))
	}

	// Notes
	m.AddRows(buildNotesSection("", quote.Notes)...)

	return g.generate(m)
}

// GenerateInvoicePDF generates a PDF for an invoice (Rechnung).
// Uses snapshot_data if available (sent invoice), otherwise live data (draft preview).
func (g *Generator) GenerateInvoicePDF(invoice models.Invoice) ([]byte, error) {
	m := g.newMaroto()
	accentColor := parseColor(g.settings.AccentColor)

	// Company header
	m.AddRows(buildHeader(g.settings, accentColor))
	m.AddRows(row.New(5))

	// Recipient
	m.AddRows(buildRecipient(invoice.CustomerName, invoice.CustomerAddress))
	m.AddRows(row.New(5))

	// Document title
	dateStr := invoice.InvoiceDate.Format("02.01.2006")
	m.AddRows(buildDocumentTitle("Rechnung", invoice.InvoiceNumber, dateStr, accentColor))

	// Due date and delivery date info
	m.AddRows(row.New(5).Add(
		buildTextCol(6, fmt.Sprintf("Faellig: %s", invoice.DueDate.Format("02.01.2006")), 8),
	))
	if invoice.DeliveryDate != nil {
		m.AddRows(row.New(5).Add(
			buildTextCol(6, fmt.Sprintf("Lieferdatum: %s", invoice.DeliveryDate.Format("02.01.2006")), 8),
		))
	}
	m.AddRows(row.New(3))

	// Line items
	lineItems, err := parseLineItems(invoice.LineItems)
	if err != nil {
		return nil, fmt.Errorf("parse invoice line items: %w", err)
	}

	m.AddRows(buildLineItemHeader())
	for i, item := range lineItems {
		pos := i + 1
		if item.Position > 0 {
			pos = item.Position
		}
		m.AddRows(buildLineItemRow(pos, item, i%2 == 1))
	}

	// Totals
	breakdown, err := parseTaxBreakdown(invoice.TaxBreakdownRaw)
	if err != nil {
		return nil, fmt.Errorf("parse invoice tax breakdown: %w", err)
	}
	if breakdown == nil {
		breakdown = &models.TaxBreakdown{
			Subtotal:   invoice.Subtotal,
			TotalTax:   invoice.TotalTax,
			GrossTotal: invoice.GrossTotal,
			TaxByRate:  make(map[string]decimal.Decimal),
		}
	}
	m.AddRows(buildTotalsSection(*breakdown, invoice.TaxMode)...)

	// Payment terms and notes
	m.AddRows(buildNotesSection(invoice.PaymentTerms, invoice.Notes)...)

	return g.generate(m)
}

// GenerateCreditNotePDF generates a PDF for a credit note (Gutschrift).
func (g *Generator) GenerateCreditNotePDF(creditNote models.CreditNote) ([]byte, error) {
	m := g.newMaroto()
	accentColor := parseColor(g.settings.AccentColor)

	// Company header
	m.AddRows(buildHeader(g.settings, accentColor))
	m.AddRows(row.New(5))

	// Recipient
	m.AddRows(buildRecipient(creditNote.CustomerName, creditNote.CustomerAddress))
	m.AddRows(row.New(5))

	// Document title
	dateStr := creditNote.CreatedAt.Format("02.01.2006")
	m.AddRows(buildDocumentTitle("Gutschrift", creditNote.CreditNoteNumber, dateStr, accentColor))
	m.AddRows(row.New(3))

	// Reference to original invoice
	m.AddRows(row.New(5).Add(
		buildTextCol(12, fmt.Sprintf("Zu Rechnung: %s", creditNote.OriginalInvoiceID.String()), 8),
	))
	if creditNote.Reason != "" {
		m.AddRows(row.New(5).Add(
			buildTextCol(12, fmt.Sprintf("Grund: %s", creditNote.Reason), 8),
		))
	}
	m.AddRows(row.New(3))

	// Line items
	lineItems, err := parseLineItems(creditNote.LineItems)
	if err != nil {
		return nil, fmt.Errorf("parse credit note line items: %w", err)
	}

	m.AddRows(buildLineItemHeader())
	for i, item := range lineItems {
		pos := i + 1
		if item.Position > 0 {
			pos = item.Position
		}
		m.AddRows(buildLineItemRow(pos, item, i%2 == 1))
	}

	// Totals
	breakdown, err := parseTaxBreakdown(creditNote.TaxBreakdownRaw)
	if err != nil {
		return nil, fmt.Errorf("parse credit note tax breakdown: %w", err)
	}
	if breakdown == nil {
		breakdown = &models.TaxBreakdown{
			Subtotal:   creditNote.Subtotal,
			TotalTax:   creditNote.TotalTax,
			GrossTotal: creditNote.GrossTotal,
			TaxByRate:  make(map[string]decimal.Decimal),
		}
	}
	m.AddRows(buildTotalsSection(*breakdown, creditNote.TaxMode)...)

	return g.generate(m)
}

// GenerateDunningPDF generates a PDF for a dunning letter (Mahnung).
// Level determines the tone: 1=friendly, 2=formal, 3=urgent (threatens Inkasso).
func (g *Generator) GenerateDunningPDF(dunning models.DunningRecord, invoice models.Invoice, level int) ([]byte, error) {
	m := g.newMaroto()
	accentColor := parseColor(g.settings.AccentColor)

	// Company header
	m.AddRows(buildHeader(g.settings, accentColor))
	m.AddRows(row.New(5))

	// Recipient
	m.AddRows(buildRecipient(invoice.CustomerName, invoice.CustomerAddress))
	m.AddRows(row.New(5))

	// Date
	dateStr := dunning.CreatedAt.Format("02.01.2006")
	m.AddRows(row.New(5).Add(
		buildTextCol(12, fmt.Sprintf("Datum: %s", dateStr), 8),
	))
	m.AddRows(row.New(3))

	// Dunning body (level-specific tone, fee, interest breakdown)
	bodyRows := buildDunningBody(level, invoice.InvoiceNumber, invoice.GrossTotal, dunning.Fee, dunning.Interest)
	m.AddRows(bodyRows...)

	return g.generate(m)
}

// generate finalizes the maroto document and returns PDF bytes.
func (g *Generator) generate(m core.Maroto) ([]byte, error) {
	doc, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate PDF: %w", err)
	}
	return doc.GetBytes(), nil
}

// parseLineItems parses JSONB line items from raw JSON.
func parseLineItems(raw json.RawMessage) ([]models.LineItem, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var items []models.LineItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// parseTaxBreakdown parses JSONB tax breakdown from raw JSON.
func parseTaxBreakdown(raw json.RawMessage) (*models.TaxBreakdown, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var tb models.TaxBreakdown
	if err := json.Unmarshal(raw, &tb); err != nil {
		return nil, err
	}
	return &tb, nil
}
