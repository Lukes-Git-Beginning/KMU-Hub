package einvoice

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// Neutral outbound document model
//
// buildInvoiceDoc normalises a stored invoice into the shape both outbound
// formats render. All amount and tax-grouping logic lives here so UBL
// (generator_ubl.go) and CII can never disagree about the numbers.
// ============================================================================

// EN 16931 VAT category codes (UNTDID 5305 subset).
const (
	taxCategoryStandard      = "S"  // standard rate
	taxCategoryZeroRated     = "Z"  // zero rated
	taxCategoryExempt        = "E"  // exempt from VAT (Kleinunternehmer, §19 UStG)
	taxCategoryReverseCharge = "AE" // VAT reverse charge

	exemptionReasonReverseCharge    = "Reverse charge"
	exemptionReasonKleinunternehmer = "Kleinunternehmer gemaess Paragraph 19 UStG"
)

// halfCent is the largest amount a single line can move when its net is rounded
// to cents. It is the unit the totals tolerance is built from.
var halfCent = decimal.New(5, -3) // 0.005

// totalsTolerance returns the largest difference between recomputed and stored
// totals that is still a rounding artefact rather than a data defect.
//
// The write path (invoice.Service, tax.Calculate) sums UNROUNDED line nets, this
// document rounds every line net to cents before summing (BR-CO-10, see
// buildLinesAndTaxGroups). The two orders can disagree by up to half a cent per
// line, so a fixed 0.01 would reject perfectly good three-line invoices with
// fractional quantities. Anything beyond that bound is not a rounding order — it
// is a stale stored figure, and that is what this guard is for.
func totalsTolerance(lineCount int) decimal.Decimal {
	if lineCount < 2 {
		return decimal.New(1, -2) // 0.01 — floor, one line can never drift further
	}
	return halfCent.Mul(decimal.NewFromInt(int64(lineCount)))
}

// invoiceDoc is the format-neutral view of an outbound invoice.
type invoiceDoc struct {
	Number         string
	IssueDate      time.Time
	DueDate        time.Time // zero when the invoice carries no due date
	DeliveryDate   time.Time // BT-72, falls back to the issue date
	Currency       string
	BuyerReference string // BT-10 Leitweg-ID
	Note           string
	PaymentTerms   string

	Seller docParty
	Buyer  docParty
	Bank   docBank

	Lines     []docLine
	TaxGroups []docTaxGroup

	// Totals are recomputed from Lines, never copied from the invoice record, so
	// the emitted document always satisfies EN 16931 BR-CO-10/13/14/15.
	LineTotal  decimal.Decimal
	TaxTotal   decimal.Decimal
	GrossTotal decimal.Decimal
}

type docParty struct {
	Name string
	// Street carries the free-text address when the record stores it unstructured.
	Street     string
	PostalZone string
	City       string
	Country    string // ISO 3166-1 alpha-2
	VATID      string // BT-31 / BT-48
	// TaxRegID is BT-32 (Steuernummer), the fallback identification for a seller
	// without a VAT identifier. Only ever set when VATID is empty — see
	// buildSellerParty.
	TaxRegID string
}

type docBank struct {
	Name string
	IBAN string
	BIC  string
}

type docLine struct {
	Position     int
	Description  string
	Quantity     decimal.Decimal
	UnitPrice    decimal.Decimal
	TaxRate      decimal.Decimal
	Net          decimal.Decimal
	CategoryCode string
}

type docTaxGroup struct {
	CategoryCode    string
	Rate            decimal.Decimal
	TaxableNet      decimal.Decimal
	TaxAmount       decimal.Decimal
	ExemptionReason string
}

// buildInvoiceDoc validates the structurally required values and normalises the
// invoice into the neutral model. buyerReference is BT-10 (Leitweg-ID), required
// for XRechnung towards public-sector buyers and empty otherwise.
//
// Full EN 16931 conformance checking (missing VAT ID, missing Leitweg-ID, line
// without a tax rate, ...) is not done here — that is w5-en16931-validation, which
// reports every violation at once. This function only rejects invoices that cannot
// be rendered into a well-formed document at all.
func buildInvoiceDoc(invoice models.Invoice, settings models.CompanySettings, buyerReference string) (*invoiceDoc, error) {
	if strings.TrimSpace(invoice.InvoiceNumber) == "" {
		return nil, fmt.Errorf("%w: invoice number (BT-1) is empty", ErrGenerateFailed)
	}
	if invoice.InvoiceDate.IsZero() {
		return nil, fmt.Errorf("%w: issue date (BT-2) is not set on invoice %s", ErrGenerateFailed, invoice.InvoiceNumber)
	}

	var items []models.LineItem
	if len(invoice.LineItems) > 0 {
		if err := json.Unmarshal(invoice.LineItems, &items); err != nil {
			return nil, fmt.Errorf("%w: line items of invoice %s: %s", ErrGenerateFailed, invoice.InvoiceNumber, err.Error())
		}
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("%w: invoice %s has no line items (BG-25)", ErrGenerateFailed, invoice.InvoiceNumber)
	}

	currency := strings.TrimSpace(invoice.Currency)
	if currency == "" {
		currency = models.DefaultCurrency
	}

	deliveryDate := invoice.InvoiceDate
	if invoice.DeliveryDate != nil && !invoice.DeliveryDate.IsZero() {
		deliveryDate = *invoice.DeliveryDate
	}

	lines, groups := buildLinesAndTaxGroups(items, invoice.TaxMode)

	lineTotal := decimal.Zero
	for _, l := range lines {
		lineTotal = lineTotal.Add(l.Net)
	}
	taxTotal := decimal.Zero
	for _, g := range groups {
		taxTotal = taxTotal.Add(g.TaxAmount)
	}
	grossTotal := lineTotal.Add(taxTotal)

	if err := assertTotalsMatch(invoice, len(lines), lineTotal, taxTotal, grossTotal); err != nil {
		return nil, err
	}

	return &invoiceDoc{
		Number:         invoice.InvoiceNumber,
		IssueDate:      invoice.InvoiceDate,
		DueDate:        invoice.DueDate,
		DeliveryDate:   deliveryDate,
		Currency:       currency,
		BuyerReference: strings.TrimSpace(buyerReference),
		Note:           invoice.Notes,
		PaymentTerms:   invoice.PaymentTerms,
		Seller:         buildSellerParty(settings),
		Buyer:          buildBuyerParty(invoice, settings),
		Bank: docBank{
			Name: settings.BankName,
			IBAN: settings.IBAN,
			BIC:  settings.BIC,
		},
		Lines:      lines,
		TaxGroups:  groups,
		LineTotal:  lineTotal,
		TaxTotal:   taxTotal,
		GrossTotal: grossTotal,
	}, nil
}

// buildSellerParty derives the seller (BG-4/BG-5) from the company settings.
//
// EN 16931 accepts either the VAT identifier (BT-31) or the tax number (BT-32) as
// the seller's tax identification, and BR-S-02/BR-E-02 demand one of them for every
// VAT category. A Kleinunternehmer has no VAT identifier at all, so without the
// BT-32 fallback their invoices would be rejected by every receiver with no way to
// fix it. Only one of the two is written: the inbound parser reads a single tax
// registration element and would otherwise pick up the wrong one.
func buildSellerParty(settings models.CompanySettings) docParty {
	p := docParty{
		Name:       settings.Name,
		Street:     settings.Street,
		PostalZone: settings.PLZ,
		City:       settings.City,
		Country:    isoCountryCode(settings.Country),
		VATID:      strings.TrimSpace(settings.UStIDNr),
	}
	if p.VATID == "" {
		p.TaxRegID = strings.TrimSpace(settings.Steuernummer)
	}
	return p
}

// buildBuyerParty derives the buyer (BG-7/BG-8) from the invoice record.
//
// lean: the invoice stores the buyer address as one free-text field, so street,
// postal code and city are recovered by shape rather than read from columns. Drop
// this once finance_invoices carries the address parts apart.
func buildBuyerParty(invoice models.Invoice, settings models.CompanySettings) docParty {
	street, postalZone, city := splitPostalAddress(invoice.CustomerAddress)
	return docParty{
		Name:       invoice.CustomerName,
		Street:     street,
		PostalZone: postalZone,
		City:       city,
		// lean: the buyer country (BT-55) is not stored, so the seller's is assumed —
		// right for the domestic invoices this covers, and validation cannot flag it
		// because the assumption leaves no gap to detect. Store the country on the
		// customer record once invoices cross a border.
		Country: isoCountryCode(settings.Country),
		VATID:   invoice.CustomerUStIDNr,
	}
}

// splitPostalAddress separates a free-text address into street, postal code and
// city. German addresses are written "street\npostal code city", and XRechnung
// makes BT-50/52/53 mandatory — a document carrying the whole string in one field
// is rejected by public-sector receivers. Anything not matching that shape stays in
// the street line as-is: an intact free-text line beats a wrongly split one.
func splitPostalAddress(raw string) (street, postalZone, city string) {
	lines := make([]string, 0, 3)
	for _, l := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	if len(lines) == 0 {
		return "", "", ""
	}
	if len(lines) >= 2 {
		last := strings.Fields(lines[len(lines)-1])
		if len(last) >= 2 && isPostalCode(last[0]) {
			return strings.Join(lines[:len(lines)-1], ", "), last[0], strings.Join(last[1:], " ")
		}
	}
	return strings.Join(lines, ", "), "", ""
}

// isPostalCode reports whether s looks like a DACH postal code (4 or 5 digits).
func isPostalCode(s string) bool {
	if len(s) < 4 || len(s) > 5 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// buildLinesAndTaxGroups converts the stored line items into document lines and the
// BG-23 VAT breakdown. Groups keep first-seen order so the output is deterministic.
//
// The taxable basis of every group is the sum of the line net amounts actually
// written into the document — not a separately recomputed quantity × unit price.
// Deriving both from the same figure is what keeps BR-S-08 (category taxable amount
// equals the sum of its line net amounts) true even when a stored line total was
// rounded or adjusted.
//
// Rounding order (EN 16931, and the order the official Schematron enforces):
//
//   - Line net (BT-131) is rounded to cents HERE, before it is summed. BT-106 is
//     compared against the sum of the line amounts as written, and BR-CO-10 has no
//     tolerance: "Sum of Invoice line net amount (BT-106) = Σ (BT-131)". Summing
//     unrounded nets and rounding only on output made BT-106 disagree with its own
//     lines by a cent per fractional quantity — a document every XRechnung portal
//     rejects.
//   - Group tax (BT-117) is computed from the finished group net (BT-116), NOT as
//     the sum of per-line taxes. BR-CO-17: "VAT category tax amount = VAT category
//     taxable amount × (rate / 100), rounded to two decimals".
//
// This is deliberately NOT the order tax.Calculate uses — see the note there. Both
// are correct for their purpose; what was wrong was that nobody had written down
// which is which.
func buildLinesAndTaxGroups(items []models.LineItem, taxMode string) ([]docLine, []docTaxGroup) {
	hundred := decimal.NewFromInt(100)
	taxFree := taxMode == models.TaxModeReverseCharge || taxMode == models.TaxModeKleinunternehmer

	lines := make([]docLine, 0, len(items))
	order := make([]string, 0, len(items))
	groups := make(map[string]*docTaxGroup)

	for i, item := range items {
		net := item.LineTotal
		if net.IsZero() {
			net = item.Quantity.Mul(item.UnitPrice)
		}
		// BR-CO-10 compares BT-106 against the line amounts as written, exactly.
		net = net.Round(2)

		rate := item.TaxRate
		if taxFree {
			// BR-AE-05 / BR-E-05: a reverse-charge or exempt line must carry rate 0.
			rate = decimal.Zero
		}
		category, exemptionReason := taxCategoryFor(taxMode, rate)

		position := item.Position
		if position <= 0 {
			position = i + 1
		}

		lines = append(lines, docLine{
			Position:     position,
			Description:  item.Description,
			Quantity:     item.Quantity,
			UnitPrice:    item.UnitPrice,
			TaxRate:      rate,
			Net:          net,
			CategoryCode: category,
		})

		key := category + "|" + rate.StringFixed(2)
		g, ok := groups[key]
		if !ok {
			g = &docTaxGroup{CategoryCode: category, Rate: rate, ExemptionReason: exemptionReason}
			groups[key] = g
			order = append(order, key)
		}
		g.TaxableNet = g.TaxableNet.Add(net)
	}

	out := make([]docTaxGroup, 0, len(order))
	for _, key := range order {
		g := groups[key]
		// BR-CO-17: BT-117 is derived from the finished BT-116, not accumulated
		// from per-line taxes. The two differ once a group holds enough lines.
		g.TaxAmount = g.TaxableNet.Mul(g.Rate).Div(hundred).Round(2)
		out = append(out, *g)
	}
	return lines, out
}

// taxCategoryFor maps the invoice tax mode and the effective rate to the EN 16931
// VAT category code and, where the category demands one, its exemption reason.
func taxCategoryFor(taxMode string, rate decimal.Decimal) (category, exemptionReason string) {
	switch taxMode {
	case models.TaxModeReverseCharge:
		return taxCategoryReverseCharge, exemptionReasonReverseCharge
	case models.TaxModeKleinunternehmer:
		return taxCategoryExempt, exemptionReasonKleinunternehmer
	default:
		if rate.IsZero() {
			return taxCategoryZeroRated, ""
		}
		return taxCategoryStandard, ""
	}
}

// assertTotalsMatch guards against emitting an e-invoice whose amounts differ from
// the ones the customer sees on the PDF. The document itself always uses the
// recomputed totals; a disagreement beyond rounding means the stored figures are
// stale, and shipping either version silently would be wrong.
func assertTotalsMatch(invoice models.Invoice, lineCount int, lineTotal, taxTotal, grossTotal decimal.Decimal) error {
	if invoice.GrossTotal.IsZero() && invoice.Subtotal.IsZero() {
		// Totals were never computed for this record — nothing to compare against.
		return nil
	}
	for _, c := range []struct {
		name             string
		stored, computed decimal.Decimal
	}{
		{"subtotal", invoice.Subtotal, lineTotal},
		{"total tax", invoice.TotalTax, taxTotal},
		{"gross total", invoice.GrossTotal, grossTotal},
	} {
		if c.stored.Sub(c.computed).Abs().GreaterThan(totalsTolerance(lineCount)) {
			return fmt.Errorf("%w: invoice %s %s stored %s, recomputed %s",
				ErrTotalsMismatch, invoice.InvoiceNumber, c.name,
				c.stored.StringFixed(2), c.computed.StringFixed(2))
		}
	}
	return nil
}

// isoCountryCode maps a stored country name to its ISO 3166-1 alpha-2 code,
// defaulting to DE for the German company profile.
func isoCountryCode(country string) string {
	switch strings.TrimSpace(country) {
	case "Deutschland", "Germany", "DE", "":
		return "DE"
	case "Österreich", "Oesterreich", "Austria", "AT":
		return "AT"
	case "Schweiz", "Switzerland", "CH":
		return "CH"
	default:
		if len(country) == 2 {
			return strings.ToUpper(country)
		}
		return "DE"
	}
}
