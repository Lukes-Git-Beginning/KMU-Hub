// Package einvoice provides parsing and persistence of inbound electronic invoices
// in ZUGFeRD/Factur-X (CII, EN 16931) and XRechnung (UBL) formats.
package einvoice

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// ============================================================================
// Common output types
// ============================================================================

// ParsedLineItem represents a single line item extracted from a parsed e-invoice.
type ParsedLineItem struct {
	Position    int
	Description string
	Quantity    decimal.Decimal
	UnitPrice   decimal.Decimal
	TaxRate     decimal.Decimal
	LineTotal   decimal.Decimal
}

// ParsedTaxEntry holds tax aggregation for one rate bucket.
type ParsedTaxEntry struct {
	TaxRate    decimal.Decimal
	TaxableNet decimal.Decimal
	TaxAmount  decimal.Decimal
}

// ParsedInvoice is the unified output from both CII and UBL parsers.
type ParsedInvoice struct {
	SupplierName    string
	SupplierAddress string
	SupplierVATID   string
	InvoiceNumber   string
	InvoiceDate     time.Time
	DueDate         *time.Time
	Currency        string
	LineItems       []ParsedLineItem
	TaxBreakdown    []ParsedTaxEntry
	Subtotal        decimal.Decimal
	TotalTax        decimal.Decimal
	GrossTotal      decimal.Decimal
	SourceFormat    string // IncomingInvoiceFormatZUGFeRDCII | XRechnungUBL | XMLOnly
}

// ============================================================================
// Format detection
// ============================================================================

// DetectFormat inspects the XML root element and namespace to determine the
// e-invoice format:
//
//   - CrossIndustryInvoice (CII) → zugferd_cii  (ZUGFeRD / Factur-X / EN 16931)
//   - Invoice with UBL namespace  → xrechnung_ubl (XRechnung UBL 2.1)
//   - anything else                → xml_only
func DetectFormat(xmlData []byte) string {
	dec := xml.NewDecoder(bytes.NewReader(xmlData))
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "CrossIndustryInvoice":
			return "zugferd_cii"
		case "Invoice":
			// Check namespace for UBL
			for _, attr := range se.Attr {
				if strings.Contains(attr.Value, "urn:oasis:names:specification:ubl") {
					return "xrechnung_ubl"
				}
			}
			// Check element namespace directly
			if strings.Contains(se.Name.Space, "urn:oasis:names:specification:ubl") {
				return "xrechnung_ubl"
			}
			return "xml_only"
		}
		// Only inspect the root element
		break
	}
	return "xml_only"
}

// ============================================================================
// CII (Cross-Industry Invoice) structs — mirrors the Factur-X EN16931 profile
// as produced by internal/biz/pdf/zugferd.go GenerateZUGFeRDXML.
// ============================================================================

type ciiEnvelope struct {
	XMLName               xml.Name           `xml:"CrossIndustryInvoice"`
	ExchangedDocument     ciiExchangedDoc    `xml:"ExchangedDocument"`
	SupplyChainTrade      ciiSupplyChainTrade `xml:"SupplyChainTradeTransaction"`
}

type ciiExchangedDoc struct {
	ID           string       `xml:"ID"`
	TypeCode     string       `xml:"TypeCode"`
	IssueDateTime ciiDateTime `xml:"IssueDateTime"`
}

type ciiDateTime struct {
	DateTimeString string `xml:"DateTimeString"`
}

type ciiSupplyChainTrade struct {
	HeaderTradeAgreement   ciiHeaderTradeAgreement  `xml:"ApplicableHeaderTradeAgreement"`
	HeaderTradeSettlement  ciiHeaderTradeSettlement `xml:"ApplicableHeaderTradeSettlement"`
	LineItems              []ciiLineItem            `xml:"IncludedSupplyChainTradeLineItem"`
}

type ciiHeaderTradeAgreement struct {
	SellerParty ciiSellerParty `xml:"SellerTradeParty"`
	BuyerParty  ciiBuyerParty  `xml:"BuyerTradeParty"`
}

type ciiSellerParty struct {
	Name    string         `xml:"Name"`
	Address ciiPostalAddr  `xml:"PostalTradeAddress"`
	TaxReg  ciiTaxReg      `xml:"SpecifiedTaxRegistration"`
}

type ciiPostalAddr struct {
	PostcodeCode string `xml:"PostcodeCode"`
	LineOne      string `xml:"LineOne"`
	CityName     string `xml:"CityName"`
	CountryID    string `xml:"CountryID"`
}

type ciiTaxReg struct {
	ID string `xml:"ID"`
}

type ciiBuyerParty struct {
	Name string `xml:"Name"`
}

type ciiHeaderTradeSettlement struct {
	PaymentReference string          `xml:"PaymentReference"`
	CurrencyCode     string          `xml:"InvoiceCurrencyCode"`
	PaymentTerms     ciiPaymentTerms `xml:"SpecifiedTradePaymentTerms"`
	MonetarySummation ciiMonetarySummation `xml:"SpecifiedTradeSettlementHeaderMonetarySummation"`
	TradeTax         []ciiApplicableTax   `xml:"ApplicableTradeTax"`
}

type ciiPaymentTerms struct {
	DueDateDateTime ciiDateTime `xml:"DueDateDateTime"`
}

type ciiMonetarySummation struct {
	LineTotalAmount string `xml:"LineTotalAmount"`
	TaxTotalAmount  string `xml:"TaxTotalAmount"`
	GrandTotalAmount string `xml:"GrandTotalAmount"`
	DuePayableAmount string `xml:"DuePayableAmount"`
}

type ciiApplicableTax struct {
	TypeCode              string `xml:"TypeCode"`
	RateApplicablePercent string `xml:"RateApplicablePercent"`
	BasisAmount           string `xml:"BasisAmount"`
	CalculatedAmount      string `xml:"CalculatedAmount"`
}

type ciiLineItem struct {
	LineDoc   ciiLineDoc             `xml:"AssociatedDocumentLineDocument"`
	Product   ciiProduct             `xml:"SpecifiedTradeProduct"`
	Agreement ciiLineTradeAgreement  `xml:"SpecifiedLineTradeAgreement"`
	Delivery  ciiLineTradeDelivery   `xml:"SpecifiedLineTradeDelivery"`
	Settlement ciiLineTradeSettlement `xml:"SpecifiedLineTradeSettlement"`
}

type ciiLineDoc struct {
	LineID string `xml:"LineID"`
}

type ciiProduct struct {
	Name string `xml:"Name"`
}

type ciiLineTradeAgreement struct {
	NetPrice ciiNetPrice `xml:"NetPriceProductTradePrice"`
}

type ciiNetPrice struct {
	ChargeAmount string `xml:"ChargeAmount"`
}

type ciiLineTradeDelivery struct {
	BilledQuantity string `xml:"BilledQuantity"`
}

type ciiLineTradeSettlement struct {
	TradeTax    ciiLineTax               `xml:"ApplicableTradeTax"`
	MonetarySummation ciiLineMonetary   `xml:"SpecifiedTradeSettlementLineMonetarySummation"`
}

type ciiLineTax struct {
	TypeCode              string `xml:"TypeCode"`
	RateApplicablePercent string `xml:"RateApplicablePercent"`
}

type ciiLineMonetary struct {
	LineTotalAmount string `xml:"LineTotalAmount"`
}

// ParseCII parses a ZUGFeRD / Factur-X EN16931 CII XML document.
func ParseCII(xmlData []byte) (*ParsedInvoice, error) {
	var env ciiEnvelope
	if err := xml.Unmarshal(xmlData, &env); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrParseFailed, err.Error())
	}

	doc := env.ExchangedDocument
	if doc.ID == "" {
		return nil, fmt.Errorf("%w: missing ExchangedDocument/ID", ErrParseFailed)
	}

	invoiceDate, err := parseCIIDate(doc.IssueDateTime.DateTimeString)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid invoice date: %s", ErrParseFailed, err.Error())
	}

	seller := env.SupplyChainTrade.HeaderTradeAgreement.SellerParty
	settlement := env.SupplyChainTrade.HeaderTradeSettlement

	supplierAddress := strings.TrimSpace(strings.Join(removeEmpty([]string{
		seller.Address.LineOne,
		seller.Address.PostcodeCode,
		seller.Address.CityName,
		seller.Address.CountryID,
	}), ", "))

	currency := settlement.CurrencyCode
	if currency == "" {
		currency = "EUR"
	}

	// Due date (optional)
	var dueDate *time.Time
	if dueDateStr := settlement.PaymentTerms.DueDateDateTime.DateTimeString; dueDateStr != "" {
		if dd, parseErr := parseCIIDate(dueDateStr); parseErr == nil {
			dueDate = &dd
		}
	}

	// Monetary totals
	subtotal := mustDecimal(settlement.MonetarySummation.LineTotalAmount)
	totalTax := mustDecimal(settlement.MonetarySummation.TaxTotalAmount)
	grossTotal := mustDecimal(settlement.MonetarySummation.GrandTotalAmount)
	if grossTotal.IsZero() {
		return nil, fmt.Errorf("%w: grand total amount missing or unparseable (got %q)", ErrParseFailed, settlement.MonetarySummation.GrandTotalAmount)
	}

	// Line items
	lineItems := make([]ParsedLineItem, 0, len(env.SupplyChainTrade.LineItems))
	for i, li := range env.SupplyChainTrade.LineItems {
		qty := mustDecimal(li.Delivery.BilledQuantity)
		price := mustDecimal(li.Agreement.NetPrice.ChargeAmount)
		taxRate := mustDecimal(li.Settlement.TradeTax.RateApplicablePercent)
		lineTotal := mustDecimal(li.Settlement.MonetarySummation.LineTotalAmount)

		lineItems = append(lineItems, ParsedLineItem{
			Position:    i + 1,
			Description: li.Product.Name,
			Quantity:    qty,
			UnitPrice:   price,
			TaxRate:     taxRate,
			LineTotal:   lineTotal,
		})
	}

	// Tax breakdown from header-level ApplicableTradeTax entries
	taxBreakdown := make([]ParsedTaxEntry, 0, len(settlement.TradeTax))
	for _, tt := range settlement.TradeTax {
		taxBreakdown = append(taxBreakdown, ParsedTaxEntry{
			TaxRate:    mustDecimal(tt.RateApplicablePercent),
			TaxableNet: mustDecimal(tt.BasisAmount),
			TaxAmount:  mustDecimal(tt.CalculatedAmount),
		})
	}
	// If no header-level tax entries were provided, derive from line items
	if len(taxBreakdown) == 0 {
		taxBreakdown = deriveTaxBreakdown(lineItems)
	}

	return &ParsedInvoice{
		SupplierName:    seller.Name,
		SupplierAddress: supplierAddress,
		SupplierVATID:   seller.TaxReg.ID,
		InvoiceNumber:   doc.ID,
		InvoiceDate:     invoiceDate,
		DueDate:         dueDate,
		Currency:        currency,
		LineItems:       lineItems,
		TaxBreakdown:    taxBreakdown,
		Subtotal:        subtotal,
		TotalTax:        totalTax,
		GrossTotal:      grossTotal,
		SourceFormat:    "zugferd_cii",
	}, nil
}

// ============================================================================
// UBL (Universal Business Language) structs — XRechnung UBL 2.1
// ============================================================================

type ublInvoice struct {
	XMLName          xml.Name         `xml:"Invoice"`
	ID               string           `xml:"ID"`
	IssueDate        string           `xml:"IssueDate"`
	DueDate          string           `xml:"DueDate"`
	DocumentCurrencyCode string       `xml:"DocumentCurrencyCode"`
	AccountingSupplierParty ublSupplierParty `xml:"AccountingSupplierParty"`
	LegalMonetaryTotal      ublMonetaryTotal `xml:"LegalMonetaryTotal"`
	TaxTotal                ublTaxTotal      `xml:"TaxTotal"`
	InvoiceLines            []ublInvoiceLine `xml:"InvoiceLine"`
}

type ublSupplierParty struct {
	Party ublParty `xml:"Party"`
}

type ublParty struct {
	PartyName        ublPartyName       `xml:"PartyName"`
	PostalAddress    ublAddress         `xml:"PostalAddress"`
	PartyTaxScheme   ublPartyTaxScheme  `xml:"PartyTaxScheme"`
	PartyLegalEntity ublPartyLegalEntity `xml:"PartyLegalEntity"`
}

type ublPartyName struct {
	Name string `xml:"Name"`
}

type ublAddress struct {
	StreetName      string `xml:"StreetName"`
	CityName        string `xml:"CityName"`
	PostalZone      string `xml:"PostalZone"`
	Country         ublCountry `xml:"Country"`
}

type ublCountry struct {
	IdentificationCode string `xml:"IdentificationCode"`
}

type ublPartyTaxScheme struct {
	CompanyID string `xml:"CompanyID"`
}

type ublPartyLegalEntity struct {
	RegistrationName string `xml:"RegistrationName"`
}

type ublMonetaryTotal struct {
	LineExtensionAmount string `xml:"LineExtensionAmount"`
	TaxExclusiveAmount  string `xml:"TaxExclusiveAmount"`
	TaxInclusiveAmount  string `xml:"TaxInclusiveAmount"`
	PayableAmount       string `xml:"PayableAmount"`
}

type ublTaxTotal struct {
	TaxAmount    string        `xml:"TaxAmount"`
	TaxSubtotals []ublTaxSubtotal `xml:"TaxSubtotal"`
}

type ublTaxSubtotal struct {
	TaxableAmount string   `xml:"TaxableAmount"`
	TaxAmount     string   `xml:"TaxAmount"`
	TaxCategory   ublTaxCategory `xml:"TaxCategory"`
}

type ublTaxCategory struct {
	Percent string `xml:"Percent"`
}

type ublInvoiceLine struct {
	ID                  string           `xml:"ID"`
	InvoicedQuantity    string           `xml:"InvoicedQuantity"`
	LineExtensionAmount string           `xml:"LineExtensionAmount"`
	Item                ublItem          `xml:"Item"`
	Price               ublPrice         `xml:"Price"`
}

type ublItem struct {
	Name        string         `xml:"Name"`
	ClassifiedTaxCategory ublTaxCategory `xml:"ClassifiedTaxCategory"`
}

type ublPrice struct {
	PriceAmount string `xml:"PriceAmount"`
}

// ParseUBL parses an XRechnung UBL 2.1 XML document.
func ParseUBL(xmlData []byte) (*ParsedInvoice, error) {
	var inv ublInvoice
	if err := xml.Unmarshal(xmlData, &inv); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrParseFailed, err.Error())
	}

	if inv.ID == "" {
		return nil, fmt.Errorf("%w: missing Invoice/ID", ErrParseFailed)
	}

	invoiceDate, err := parseISODate(inv.IssueDate)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid IssueDate: %s", ErrParseFailed, err.Error())
	}

	var dueDate *time.Time
	if inv.DueDate != "" {
		if dd, parseErr := parseISODate(inv.DueDate); parseErr == nil {
			dueDate = &dd
		}
	}

	currency := inv.DocumentCurrencyCode
	if currency == "" {
		currency = "EUR"
	}

	party := inv.AccountingSupplierParty.Party
	supplierName := party.PartyName.Name
	if supplierName == "" {
		supplierName = party.PartyLegalEntity.RegistrationName
	}

	supplierAddress := strings.TrimSpace(strings.Join(removeEmpty([]string{
		party.PostalAddress.StreetName,
		party.PostalAddress.PostalZone,
		party.PostalAddress.CityName,
		party.PostalAddress.Country.IdentificationCode,
	}), ", "))

	supplierVATID := party.PartyTaxScheme.CompanyID

	// Totals
	subtotal := mustDecimal(inv.LegalMonetaryTotal.TaxExclusiveAmount)
	if subtotal.IsZero() {
		subtotal = mustDecimal(inv.LegalMonetaryTotal.LineExtensionAmount)
	}
	totalTax := mustDecimal(inv.TaxTotal.TaxAmount)
	grossTotal := mustDecimal(inv.LegalMonetaryTotal.TaxInclusiveAmount)
	if grossTotal.IsZero() {
		grossTotal = mustDecimal(inv.LegalMonetaryTotal.PayableAmount)
	}
	if grossTotal.IsZero() {
		return nil, fmt.Errorf("%w: tax-inclusive/payable amount missing or unparseable", ErrParseFailed)
	}

	// Line items
	lineItems := make([]ParsedLineItem, 0, len(inv.InvoiceLines))
	for i, li := range inv.InvoiceLines {
		qty := mustDecimal(li.InvoicedQuantity)
		price := mustDecimal(li.Price.PriceAmount)
		taxRate := mustDecimal(li.Item.ClassifiedTaxCategory.Percent)
		lineTotal := mustDecimal(li.LineExtensionAmount)

		lineItems = append(lineItems, ParsedLineItem{
			Position:    i + 1,
			Description: li.Item.Name,
			Quantity:    qty,
			UnitPrice:   price,
			TaxRate:     taxRate,
			LineTotal:   lineTotal,
		})
	}

	// Tax breakdown
	taxBreakdown := make([]ParsedTaxEntry, 0, len(inv.TaxTotal.TaxSubtotals))
	for _, ts := range inv.TaxTotal.TaxSubtotals {
		taxBreakdown = append(taxBreakdown, ParsedTaxEntry{
			TaxRate:    mustDecimal(ts.TaxCategory.Percent),
			TaxableNet: mustDecimal(ts.TaxableAmount),
			TaxAmount:  mustDecimal(ts.TaxAmount),
		})
	}
	if len(taxBreakdown) == 0 {
		taxBreakdown = deriveTaxBreakdown(lineItems)
	}

	return &ParsedInvoice{
		SupplierName:    supplierName,
		SupplierAddress: supplierAddress,
		SupplierVATID:   supplierVATID,
		InvoiceNumber:   inv.ID,
		InvoiceDate:     invoiceDate,
		DueDate:         dueDate,
		Currency:        currency,
		LineItems:       lineItems,
		TaxBreakdown:    taxBreakdown,
		Subtotal:        subtotal,
		TotalTax:        totalTax,
		GrossTotal:      grossTotal,
		SourceFormat:    "xrechnung_ubl",
	}, nil
}

// ============================================================================
// Helpers
// ============================================================================

// parseCIIDate parses a CII DateTimeString in format "102" (yyyyMMdd).
func parseCIIDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if len(s) != 8 {
		return time.Time{}, fmt.Errorf("expected yyyyMMdd, got %q", s)
	}
	return time.Parse("20060102", s)
}

// parseISODate parses an ISO 8601 date string (YYYY-MM-DD).
func parseISODate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(s))
}

// mustDecimal converts a string to decimal.Decimal, returning zero on failure.
func mustDecimal(s string) decimal.Decimal {
	s = strings.TrimSpace(s)
	if s == "" {
		return decimal.Zero
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

// removeEmpty filters out empty strings from a slice.
func removeEmpty(parts []string) []string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return out
}

// deriveTaxBreakdown computes a tax breakdown from line items when the document
// does not contain explicit header-level tax entries.
func deriveTaxBreakdown(items []ParsedLineItem) []ParsedTaxEntry {
	type bucket struct {
		net decimal.Decimal
		tax decimal.Decimal
	}
	m := make(map[string]*bucket)
	keys := make([]string, 0)

	for _, li := range items {
		key := li.TaxRate.StringFixed(2)
		if _, ok := m[key]; !ok {
			m[key] = &bucket{}
			keys = append(keys, key)
		}
		m[key].net = m[key].net.Add(li.LineTotal)
		m[key].tax = m[key].tax.Add(li.LineTotal.Mul(li.TaxRate).Div(decimal.NewFromInt(100)))
	}

	result := make([]ParsedTaxEntry, 0, len(keys))
	for _, key := range keys {
		rate := mustDecimal(key)
		result = append(result, ParsedTaxEntry{
			TaxRate:    rate,
			TaxableNet: m[key].net,
			TaxAmount:  m[key].tax.Round(2),
		})
	}
	return result
}
