package datev

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Exporter generates DATEV Buchungsstapel CSV files in EXTF format.
type Exporter struct{}

// NewExporter creates a new DATEV exporter.
func NewExporter() *Exporter {
	return &Exporter{}
}

// Export generates a DATEV Buchungsstapel CSV from invoices and credit notes.
// Returns CSV bytes with UTF-8 BOM, semicolon delimiter, EXTF header.
func (e *Exporter) Export(
	invoices []*models.Invoice,
	creditNotes []*models.CreditNote,
	fiscalYearStart time.Time,
	generatedAt time.Time,
) ([]byte, error) {
	var buf bytes.Buffer

	// Write UTF-8 BOM for DATEV compatibility
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	w := csv.NewWriter(&buf)
	w.Comma = ';'

	// Line 1: EXTF header
	if err := writeEXTFHeader(w, fiscalYearStart, generatedAt); err != nil {
		return nil, fmt.Errorf("write EXTF header: %w", err)
	}

	// Line 2: Column headers
	if err := writeColumnHeaders(w); err != nil {
		return nil, fmt.Errorf("write column headers: %w", err)
	}

	// Track unique customers for debitor account assignment
	customerIndex := make(map[string]int)
	nextIndex := 1

	// Write invoice booking lines
	for _, inv := range invoices {
		// Only export sent, paid, or overdue invoices
		if inv.Status != models.InvoiceStatusSent &&
			inv.Status != models.InvoiceStatusPaid &&
			inv.Status != models.InvoiceStatusOverdue {
			continue
		}

		custKey := inv.CustomerName
		if _, ok := customerIndex[custKey]; !ok {
			customerIndex[custKey] = nextIndex
			nextIndex++
		}
		debitorAccount := DebitorAccountBase + customerIndex[custKey]

		lineItems, err := parseLineItems(inv.LineItems)
		if err != nil {
			return nil, fmt.Errorf("parse invoice %s line items: %w", inv.ID, err)
		}

		for _, item := range lineItems {
			if err := writeBookingLine(w, item, "S", debitorAccount, inv.TaxMode, inv.InvoiceDate, inv.InvoiceNumber, false); err != nil {
				return nil, fmt.Errorf("write invoice booking line: %w", err)
			}
		}
	}

	// Write credit note booking lines
	for _, cn := range creditNotes {
		if cn.Status != models.CreditNoteStatusSent {
			continue
		}

		custKey := cn.CustomerName
		if _, ok := customerIndex[custKey]; !ok {
			customerIndex[custKey] = nextIndex
			nextIndex++
		}
		debitorAccount := DebitorAccountBase + customerIndex[custKey]

		lineItems, err := parseLineItems(cn.LineItems)
		if err != nil {
			return nil, fmt.Errorf("parse credit note %s line items: %w", cn.ID, err)
		}

		for _, item := range lineItems {
			if err := writeBookingLine(w, item, "H", debitorAccount, cn.TaxMode, cn.CreatedAt, cn.CreditNoteNumber, true); err != nil {
				return nil, fmt.Errorf("write credit note booking line: %w", err)
			}
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("csv flush: %w", err)
	}

	return buf.Bytes(), nil
}

// writeEXTFHeader writes the DATEV EXTF header line (line 1).
func writeEXTFHeader(w *csv.Writer, fiscalYearStart, generatedAt time.Time) error {
	// EXTF format: "EXTF";700;21;"Buchungsstapel";13;timestamp;;"KMU Hub";;;;fiscal_year_start_YYYYMMDD;4;...
	record := []string{
		"EXTF",                                      // Format
		"700",                                       // Format version
		"21",                                        // Data category (Buchungsstapel)
		"Buchungsstapel",                            // Format name
		"13",                                        // Format version (inner)
		generatedAt.Format("20060102150405000"),      // Generated timestamp
		"",                                          // Reserved
		"KMU Hub",                                   // Source application
		"",                                          // Reserved
		"",                                          // Reserved
		"",                                          // Reserved
		"",                                          // Reserved
		fiscalYearStart.Format("20060102"),           // Fiscal year start
		"4",                                         // Account length (SKR03 = 4 digits)
		"",                                          // Reserved
		"",                                          // Reserved
		"",                                          // Reserved
		"",                                          // Reserved
		"",                                          // Reserved
		"",                                          // Reserved
	}
	return w.Write(record)
}

// writeColumnHeaders writes the DATEV column header line (line 2).
func writeColumnHeaders(w *csv.Writer) error {
	headers := []string{
		"Umsatz (ohne Soll/Haben-Kz)",
		"Soll/Haben-Kennzeichen",
		"WKZ Umsatz",
		"Kurs",
		"Basis-Umsatz",
		"WKZ Basis-Umsatz",
		"Konto",
		"Gegenkonto (ohne BU-Schluessel)",
		"BU-Schluessel",
		"Belegdatum",
		"Belegfeld 1",
		"Belegfeld 2",
		"Skonto",
		"Buchungstext",
	}
	return w.Write(headers)
}

// writeBookingLine writes a single booking line to the CSV.
func writeBookingLine(
	w *csv.Writer,
	item models.LineItem,
	sollHaben string, // "S" for debit (invoice), "H" for credit (credit note)
	debitorAccount int,
	taxMode string,
	docDate time.Time,
	docNumber string,
	isCreditNote bool,
) error {
	// Calculate gross amount for this line item (net + tax)
	grossAmount := item.LineTotal.Add(item.LineTotal.Mul(item.TaxRate.Div(decimal.NewFromInt(100))))
	grossAmount = grossAmount.Round(2)

	// For credit notes, use absolute value (DATEV uses H for the direction)
	if isCreditNote {
		grossAmount = grossAmount.Abs()
	}

	// Determine rate key for account mapping
	rateKey := truncateRate(item.TaxRate)
	revenueAccount := RevenueAccountForRateAndMode(rateKey, taxMode)
	buSchluessel := BUSchluesselForRate(rateKey)

	// Belegdatum: DDMM format (4 digits, no separator)
	belegdatum := docDate.Format("0201") // Go: day=02, month=01

	// Buchungstext: truncated to 60 chars
	buchungstext := item.Description
	if len(buchungstext) > 60 {
		buchungstext = buchungstext[:60]
	}

	record := []string{
		formatDecimalForDATEV(grossAmount), // Umsatz
		sollHaben,                          // Soll/Haben-Kennzeichen
		"EUR",                              // WKZ Umsatz
		"",                                 // Kurs (empty for EUR)
		"",                                 // Basis-Umsatz (empty for EUR)
		"",                                 // WKZ Basis-Umsatz (empty for EUR)
		fmt.Sprintf("%d", debitorAccount),  // Konto (debitor)
		revenueAccount,                     // Gegenkonto (revenue account)
		buSchluessel,                       // BU-Schluessel
		belegdatum,                         // Belegdatum (DDMM)
		docNumber,                          // Belegfeld 1 (invoice/credit note number)
		"",                                 // Belegfeld 2
		"",                                 // Skonto
		buchungstext,                       // Buchungstext
	}

	return w.Write(record)
}

// formatDecimalForDATEV formats a decimal for DATEV (comma as decimal separator).
func formatDecimalForDATEV(d decimal.Decimal) string {
	s := d.StringFixed(2)
	return strings.Replace(s, ".", ",", 1)
}

// truncateRate converts a decimal tax rate to its whole number string representation.
// e.g., 19.00 -> "19", 7.00 -> "7", 0.00 -> "0"
func truncateRate(rate decimal.Decimal) string {
	intPart := rate.IntPart()
	return fmt.Sprintf("%d", intPart)
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
