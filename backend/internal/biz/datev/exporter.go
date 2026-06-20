package datev

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
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
//
// Export is a thin in-memory wrapper around StreamWriter: it buffers the whole
// CSV. For large exports prefer NewStreamWriter with paged DB reads (see
// BizGRPCServer.ExportDATEV), which bounds read-memory while still materializing
// the CSV once. The write order here is identical to the streaming path, so the
// produced bytes are unchanged.
func (e *Exporter) Export(
	invoices []*models.Invoice,
	creditNotes []*models.CreditNote,
	beraterNr, mandantNr string,
	fiscalYearStart time.Time,
	generatedAt time.Time,
) ([]byte, error) {
	var buf bytes.Buffer
	sw, err := e.NewStreamWriter(&buf, beraterNr, mandantNr, fiscalYearStart, generatedAt)
	if err != nil {
		return nil, err
	}
	if err := sw.WriteInvoices(invoices); err != nil {
		return nil, err
	}
	if err := sw.WriteCreditNotes(creditNotes); err != nil {
		return nil, err
	}
	if err := sw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// StreamWriter incrementally writes a DATEV Buchungsstapel CSV to an io.Writer.
// It carries the debitor-account assignment state (customerIndex) across multiple
// WriteInvoices/WriteCreditNotes calls so booking lines can be written page-by-page
// without re-numbering customers per page. Create one via NewStreamWriter, call
// WriteInvoices then WriteCreditNotes (any number of times), then Close.
type StreamWriter struct {
	w             *csv.Writer
	customerIndex map[string]int
	nextIndex     int
	lineCount     int
}

// NewStreamWriter writes the UTF-8 BOM, EXTF header and column headers to w and
// returns a StreamWriter ready to accept booking lines.
func (e *Exporter) NewStreamWriter(w io.Writer, beraterNr, mandantNr string, fiscalYearStart, generatedAt time.Time) (*StreamWriter, error) {
	// Write UTF-8 BOM for DATEV compatibility
	if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return nil, fmt.Errorf("write BOM: %w", err)
	}

	cw := csv.NewWriter(w)
	cw.Comma = ';'

	// Line 1: EXTF header
	if err := writeEXTFHeader(cw, beraterNr, mandantNr, fiscalYearStart, generatedAt); err != nil {
		return nil, fmt.Errorf("write EXTF header: %w", err)
	}

	// Line 2: Column headers
	if err := writeColumnHeaders(cw); err != nil {
		return nil, fmt.Errorf("write column headers: %w", err)
	}

	return &StreamWriter{
		w:             cw,
		customerIndex: make(map[string]int),
		nextIndex:     1,
	}, nil
}

// WriteInvoices appends booking lines for the given invoices. Only sent, paid or
// overdue invoices are exported; others are skipped. Flushes at the end of the call.
func (sw *StreamWriter) WriteInvoices(invoices []*models.Invoice) error {
	for _, inv := range invoices {
		// Only export sent, paid, or overdue invoices
		if inv.Status != models.InvoiceStatusSent &&
			inv.Status != models.InvoiceStatusPaid &&
			inv.Status != models.InvoiceStatusOverdue {
			continue
		}

		debitorAccount := sw.debitorFor(inv.CustomerName)

		lineItems, err := parseLineItems(inv.LineItems)
		if err != nil {
			return fmt.Errorf("parse invoice %s line items: %w", inv.ID, err)
		}

		for _, item := range lineItems {
			if err := writeBookingLine(sw.w, item, "S", debitorAccount, inv.TaxMode, inv.Currency, inv.InvoiceDate, inv.InvoiceNumber, false); err != nil {
				return fmt.Errorf("write invoice booking line: %w", err)
			}
			sw.lineCount++
		}
	}

	sw.w.Flush()
	return sw.w.Error()
}

// WriteCreditNotes appends booking lines for the given credit notes. Only sent
// credit notes are exported. Flushes at the end of the call.
func (sw *StreamWriter) WriteCreditNotes(creditNotes []*models.CreditNote) error {
	for _, cn := range creditNotes {
		if cn.Status != models.CreditNoteStatusSent {
			continue
		}

		debitorAccount := sw.debitorFor(cn.CustomerName)

		lineItems, err := parseLineItems(cn.LineItems)
		if err != nil {
			return fmt.Errorf("parse credit note %s line items: %w", cn.ID, err)
		}

		for _, item := range lineItems {
			if err := writeBookingLine(sw.w, item, "H", debitorAccount, cn.TaxMode, cn.Currency, cn.CreatedAt, cn.CreditNoteNumber, true); err != nil {
				return fmt.Errorf("write credit note booking line: %w", err)
			}
			sw.lineCount++
		}
	}

	sw.w.Flush()
	return sw.w.Error()
}

// debitorFor returns the SKR03 debitor account for a customer name, assigning a
// new sequential index on first sight (stable across the StreamWriter's lifetime).
func (sw *StreamWriter) debitorFor(customerName string) int {
	if _, ok := sw.customerIndex[customerName]; !ok {
		sw.customerIndex[customerName] = sw.nextIndex
		sw.nextIndex++
	}
	return DebitorAccountBase + sw.customerIndex[customerName]
}

// LineCount returns the number of booking lines written so far (one per exported
// line item). Used to populate ExportDATEVResponse.RecordCount.
func (sw *StreamWriter) LineCount() int {
	return sw.lineCount
}

// Close flushes any buffered data and returns the first write error encountered.
func (sw *StreamWriter) Close() error {
	sw.w.Flush()
	if err := sw.w.Error(); err != nil {
		return fmt.Errorf("csv flush: %w", err)
	}
	return nil
}

// writeEXTFHeader writes the DATEV EXTF header line (line 1).
func writeEXTFHeader(w *csv.Writer, beraterNr, mandantNr string, fiscalYearStart, generatedAt time.Time) error {
	// EXTF format: "EXTF";700;21;"Buchungsstapel";13;timestamp;;"KMU Hub";;;Berater;Mandant;fiscal_year_start_YYYYMMDD;4;...
	record := []string{
		"EXTF",                                      // Format
		"700",                                       // Format version
		"21",                                        // Data category (Buchungsstapel)
		"Buchungsstapel",                            // Format name
		"13",                                        // Format version (inner)
		generatedAt.Format("20060102150405000"),      // Generated timestamp
		"",                                          // Imported
		"KMU Hub",                                   // Source application
		"",                                          // Exported by
		"",                                          // Imported by
		beraterNr,                                   // Beraternummer
		mandantNr,                                   // Mandantennummer
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
	currency string, // document currency (WKZ Umsatz); defaults to EUR
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

	// WKZ Umsatz: the document's own currency. Hardcoding EUR mis-booked
	// foreign-currency (CHF/USD) documents. Kurs / Basis-Umsatz stay empty —
	// no per-document exchange rate is stored, so DATEV applies its own daily
	// rate on import; at least the currency code is now truthful.
	wkz := currency
	if wkz == "" {
		wkz = models.DefaultCurrency
	}

	record := []string{
		formatDecimalForDATEV(grossAmount), // Umsatz
		sollHaben,                          // Soll/Haben-Kennzeichen
		wkz,                                // WKZ Umsatz (document currency)
		"",                                 // Kurs (empty: no stored exchange rate)
		"",                                 // Basis-Umsatz (empty: EUR base not derivable without a rate)
		"",                                 // WKZ Basis-Umsatz
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
