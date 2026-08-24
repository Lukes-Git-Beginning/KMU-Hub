package einvoice

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// XRechnung UBL 2.1 writer (EN 16931)
// ============================================================================

const (
	ublNamespaceInvoice = "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
	ublNamespaceCAC     = "urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2"
	ublNamespaceCBC     = "urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2"

	// xrechnungCustomizationID marks the document as XRechnung 3.0, the CIUS German
	// public-sector buyers require. Receivers reject documents without it.
	//
	// BR-01 (specification identifier, BT-24) needs no runtime check: renderUBL
	// writes this constant into CustomizationID unconditionally (as does renderCII
	// with its own guideline ID, generator_cii.go), so no code path can emit a
	// document lacking it.
	xrechnungCustomizationID = "urn:cen.eu:en16931:2017#compliant#urn:xoev-de:kosit:standard:xrechnung_3.0"
	peppolBillingProfileID   = "urn:fdc:peppol.eu:2017:poacc:billing:01:1.0"

	// BR-04 (invoice type code, BT-3): also unconditional, see BR-01 above — this
	// product only ever emits commercial invoices ("380"), never a credit note
	// ("381").
	invoiceTypeCodeCommercial  = "380" // UNTDID 1001: commercial invoice
	unitCodePiece              = "C62" // UN/ECE Rec 20: one (piece)
	paymentMeansCodeSEPACredit = "58"  // UNTDID 4461: SEPA credit transfer
	taxSchemeVAT               = "VAT"
	// taxSchemeFiscal identifies BT-32 (Steuernummer) as opposed to BT-31 (VAT id).
	taxSchemeFiscal = "FC"
)

// GenerateUBL renders an invoice as an XRechnung UBL 2.1 document (EN 16931).
// buyerReference is BT-10 (Leitweg-ID); pass an empty string for private-sector
// buyers, where the element is omitted.
func GenerateUBL(invoice models.Invoice, settings models.CompanySettings, buyerReference string) ([]byte, error) {
	doc, err := buildInvoiceDoc(invoice, settings, buyerReference)
	if err != nil {
		return nil, err
	}
	// EN 16931 only: whether the German CIUS also applies depends on the receiver,
	// which the generator does not know. Call Validate with ProfileXRechnung before
	// sending to a public-sector buyer.
	if err := validateInvoiceDoc(doc, ProfileEN16931); err != nil {
		return nil, err
	}
	return renderUBL(doc)
}

func renderUBL(doc *invoiceDoc) ([]byte, error) {
	out := ublInvoiceOut{
		Xmlns:                ublNamespaceInvoice,
		XmlnsCAC:             ublNamespaceCAC,
		XmlnsCBC:             ublNamespaceCBC,
		CustomizationID:      xrechnungCustomizationID,
		ProfileID:            peppolBillingProfileID,
		ID:                   doc.Number,
		IssueDate:            doc.IssueDate.Format("2006-01-02"),
		InvoiceTypeCode:      invoiceTypeCodeCommercial,
		Note:                 doc.Note,
		DocumentCurrencyCode: doc.Currency,
		BuyerReference:       doc.BuyerReference,
		SupplierParty:        ublPartyWrapOut{Party: buildUBLParty(doc.Seller)},
		CustomerParty:        ublPartyWrapOut{Party: buildUBLParty(doc.Buyer)},
		TaxTotal: ublTaxTotalOut{
			TaxAmount: ublAmount(doc.Currency, doc.TaxTotal),
			Subtotals: make([]ublTaxSubtotalOut, 0, len(doc.TaxGroups)),
		},
		LegalMonetaryTotal: ublMonetaryTotalOut{
			LineExtensionAmount: ublAmount(doc.Currency, doc.LineTotal),
			TaxExclusiveAmount:  ublAmount(doc.Currency, doc.LineTotal),
			TaxInclusiveAmount:  ublAmount(doc.Currency, doc.GrossTotal),
			PayableAmount:       ublAmount(doc.Currency, doc.GrossTotal),
		},
		Lines: make([]ublInvoiceLineOut, 0, len(doc.Lines)),
	}

	if !doc.DueDate.IsZero() {
		out.DueDate = doc.DueDate.Format("2006-01-02")
	}
	if !doc.DeliveryDate.IsZero() {
		out.Delivery = &ublDeliveryOut{ActualDeliveryDate: doc.DeliveryDate.Format("2006-01-02")}
	}
	if iban := strings.TrimSpace(doc.Bank.IBAN); iban != "" {
		means := &ublPaymentMeansOut{
			PaymentMeansCode: paymentMeansCodeSEPACredit,
			PayeeAccount:     ublFinancialAccountOut{ID: iban, Name: doc.Bank.Name},
		}
		if bic := strings.TrimSpace(doc.Bank.BIC); bic != "" {
			means.PayeeAccount.Branch = &ublBranchOut{ID: bic}
		}
		out.PaymentMeans = means
	}
	if terms := strings.TrimSpace(doc.PaymentTerms); terms != "" {
		out.PaymentTerms = &ublPaymentTermsOut{Note: terms}
	}

	for _, g := range doc.TaxGroups {
		out.TaxTotal.Subtotals = append(out.TaxTotal.Subtotals, ublTaxSubtotalOut{
			TaxableAmount: ublAmount(doc.Currency, g.TaxableNet),
			TaxAmount:     ublAmount(doc.Currency, g.TaxAmount),
			TaxCategory: ublTaxCategoryOut{
				ID:              g.CategoryCode,
				Percent:         g.Rate.StringFixed(2),
				ExemptionReason: g.ExemptionReason,
				TaxScheme:       ublTaxSchemeOut{ID: taxSchemeVAT},
			},
		})
	}

	for _, l := range doc.Lines {
		out.Lines = append(out.Lines, ublInvoiceLineOut{
			ID:                  strconv.Itoa(l.Position),
			InvoicedQuantity:    ublQuantityOut{UnitCode: unitCodePiece, Value: l.Quantity.StringFixed(2)},
			LineExtensionAmount: ublAmount(doc.Currency, l.Net),
			Item: ublItemOut{
				Name: l.Description,
				ClassifiedTaxCategory: ublTaxCategoryOut{
					ID:        l.CategoryCode,
					Percent:   l.TaxRate.StringFixed(2),
					TaxScheme: ublTaxSchemeOut{ID: taxSchemeVAT},
				},
			},
			Price: ublPriceOut{PriceAmount: ublAmount(doc.Currency, l.UnitPrice)},
		})
	}

	body, err := xml.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("%w: marshal UBL for invoice %s: %s", ErrGenerateFailed, doc.Number, err.Error())
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.Write(body)
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

func buildUBLParty(p docParty) ublPartyOut {
	out := ublPartyOut{
		Name: p.Name,
		PostalAddress: ublAddressOut{
			StreetName: p.Street,
			CityName:   p.City,
			PostalZone: p.PostalZone,
			Country:    ublCountryOut{IdentificationCode: p.Country},
		},
		LegalEntity: ublLegalEntityOut{RegistrationName: p.Name},
	}
	switch {
	case strings.TrimSpace(p.VATID) != "":
		out.PartyTaxScheme = &ublPartyTaxSchemeOut{
			CompanyID: strings.TrimSpace(p.VATID),
			TaxScheme: ublTaxSchemeOut{ID: taxSchemeVAT},
		}
	case strings.TrimSpace(p.TaxRegID) != "":
		out.PartyTaxScheme = &ublPartyTaxSchemeOut{
			CompanyID: strings.TrimSpace(p.TaxRegID),
			TaxScheme: ublTaxSchemeOut{ID: taxSchemeFiscal},
		}
	}
	return out
}

func ublAmount(currency string, d decimal.Decimal) ublAmountOut {
	return ublAmountOut{CurrencyID: currency, Value: d.StringFixed(2)}
}

// ============================================================================
// Write-side UBL structs
//
// These are deliberately separate from the read-side structs in parser.go. The
// parser matches element local names in ANY namespace so inbound documents parse
// no matter which prefixes the sender chose; writing XRechnung requires the
// literal cbc:/cac: prefixes. encoding/xml cannot express both on one field — a
// namespace-qualified tag would make the parser reject documents from senders
// using a different namespace, and a prefixed tag never matches while decoding.
// TestGenerateUBL_RoundTripThroughParser covers that both directions agree.
//
// Field order follows the UBL 2.1 XSD sequences; reordering breaks schema
// validation at the receiver even though the document still parses here.
// ============================================================================

type ublInvoiceOut struct {
	XMLName  xml.Name `xml:"Invoice"`
	Xmlns    string   `xml:"xmlns,attr"`
	XmlnsCAC string   `xml:"xmlns:cac,attr"`
	XmlnsCBC string   `xml:"xmlns:cbc,attr"`

	CustomizationID      string `xml:"cbc:CustomizationID"`
	ProfileID            string `xml:"cbc:ProfileID"`
	ID                   string `xml:"cbc:ID"`
	IssueDate            string `xml:"cbc:IssueDate"`
	DueDate              string `xml:"cbc:DueDate,omitempty"`
	InvoiceTypeCode      string `xml:"cbc:InvoiceTypeCode"`
	Note                 string `xml:"cbc:Note,omitempty"`
	DocumentCurrencyCode string `xml:"cbc:DocumentCurrencyCode"`
	BuyerReference       string `xml:"cbc:BuyerReference,omitempty"`

	SupplierParty      ublPartyWrapOut     `xml:"cac:AccountingSupplierParty"`
	CustomerParty      ublPartyWrapOut     `xml:"cac:AccountingCustomerParty"`
	Delivery           *ublDeliveryOut     `xml:"cac:Delivery,omitempty"`
	PaymentMeans       *ublPaymentMeansOut `xml:"cac:PaymentMeans,omitempty"`
	PaymentTerms       *ublPaymentTermsOut `xml:"cac:PaymentTerms,omitempty"`
	TaxTotal           ublTaxTotalOut      `xml:"cac:TaxTotal"`
	LegalMonetaryTotal ublMonetaryTotalOut `xml:"cac:LegalMonetaryTotal"`
	Lines              []ublInvoiceLineOut `xml:"cac:InvoiceLine"`
}

type ublPartyWrapOut struct {
	Party ublPartyOut `xml:"cac:Party"`
}

type ublPartyOut struct {
	Name           string                `xml:"cac:PartyName>cbc:Name"`
	PostalAddress  ublAddressOut         `xml:"cac:PostalAddress"`
	PartyTaxScheme *ublPartyTaxSchemeOut `xml:"cac:PartyTaxScheme,omitempty"`
	LegalEntity    ublLegalEntityOut     `xml:"cac:PartyLegalEntity"`
}

type ublAddressOut struct {
	StreetName string        `xml:"cbc:StreetName,omitempty"`
	CityName   string        `xml:"cbc:CityName,omitempty"`
	PostalZone string        `xml:"cbc:PostalZone,omitempty"`
	Country    ublCountryOut `xml:"cac:Country"`
}

type ublCountryOut struct {
	IdentificationCode string `xml:"cbc:IdentificationCode"`
}

type ublPartyTaxSchemeOut struct {
	CompanyID string          `xml:"cbc:CompanyID"`
	TaxScheme ublTaxSchemeOut `xml:"cac:TaxScheme"`
}

type ublLegalEntityOut struct {
	RegistrationName string `xml:"cbc:RegistrationName"`
}

type ublAmountOut struct {
	CurrencyID string `xml:"currencyID,attr"`
	Value      string `xml:",chardata"`
}

type ublQuantityOut struct {
	UnitCode string `xml:"unitCode,attr"`
	Value    string `xml:",chardata"`
}

type ublDeliveryOut struct {
	ActualDeliveryDate string `xml:"cbc:ActualDeliveryDate"`
}

type ublPaymentMeansOut struct {
	PaymentMeansCode string                 `xml:"cbc:PaymentMeansCode"`
	PayeeAccount     ublFinancialAccountOut `xml:"cac:PayeeFinancialAccount"`
}

type ublFinancialAccountOut struct {
	ID     string        `xml:"cbc:ID"`
	Name   string        `xml:"cbc:Name,omitempty"`
	Branch *ublBranchOut `xml:"cac:FinancialInstitutionBranch,omitempty"`
}

type ublBranchOut struct {
	ID string `xml:"cbc:ID"`
}

type ublPaymentTermsOut struct {
	Note string `xml:"cbc:Note"`
}

type ublTaxTotalOut struct {
	TaxAmount ublAmountOut        `xml:"cbc:TaxAmount"`
	Subtotals []ublTaxSubtotalOut `xml:"cac:TaxSubtotal"`
}

type ublTaxSubtotalOut struct {
	TaxableAmount ublAmountOut      `xml:"cbc:TaxableAmount"`
	TaxAmount     ublAmountOut      `xml:"cbc:TaxAmount"`
	TaxCategory   ublTaxCategoryOut `xml:"cac:TaxCategory"`
}

type ublTaxCategoryOut struct {
	ID              string          `xml:"cbc:ID"`
	Percent         string          `xml:"cbc:Percent"`
	ExemptionReason string          `xml:"cbc:TaxExemptionReason,omitempty"`
	TaxScheme       ublTaxSchemeOut `xml:"cac:TaxScheme"`
}

type ublTaxSchemeOut struct {
	ID string `xml:"cbc:ID"`
}

type ublMonetaryTotalOut struct {
	LineExtensionAmount ublAmountOut `xml:"cbc:LineExtensionAmount"`
	TaxExclusiveAmount  ublAmountOut `xml:"cbc:TaxExclusiveAmount"`
	TaxInclusiveAmount  ublAmountOut `xml:"cbc:TaxInclusiveAmount"`
	PayableAmount       ublAmountOut `xml:"cbc:PayableAmount"`
}

type ublInvoiceLineOut struct {
	ID                  string         `xml:"cbc:ID"`
	InvoicedQuantity    ublQuantityOut `xml:"cbc:InvoicedQuantity"`
	LineExtensionAmount ublAmountOut   `xml:"cbc:LineExtensionAmount"`
	Item                ublItemOut     `xml:"cac:Item"`
	Price               ublPriceOut    `xml:"cac:Price"`
}

type ublItemOut struct {
	Name                  string            `xml:"cbc:Name"`
	ClassifiedTaxCategory ublTaxCategoryOut `xml:"cac:ClassifiedTaxCategory"`
}

type ublPriceOut struct {
	PriceAmount ublAmountOut `xml:"cbc:PriceAmount"`
}
