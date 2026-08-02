package einvoice

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// ZUGFeRD 2.1 / Factur-X CII writer (EN 16931 profile)
// ============================================================================

const (
	ciiNamespaceRSM = "urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
	ciiNamespaceRAM = "urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100"
	ciiNamespaceUDT = "urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100"

	// facturXEN16931GuidelineID marks the document as EN 16931 (Comfort) conformant.
	// Receivers use it to pick the validation profile; without it the document is
	// treated as an unknown Factur-X profile.
	facturXEN16931GuidelineID = "urn:cen.eu:en16931:2017#compliant#urn:factur-x.eu:1p0:en16931"

	ciiDateFormatCode = "102" // UNTDID 2379: CCYYMMDD
	ciiDateLayout     = "20060102"

	// UNTDID 1153 scheme identifiers for the seller tax registration.
	ciiTaxSchemeVAT    = "VA" // BT-31 VAT identifier
	ciiTaxSchemeFiscal = "FC" // BT-32 tax number (Steuernummer)
)

// GenerateCII renders an invoice as a ZUGFeRD 2.1 / Factur-X Cross Industry
// Invoice (EN 16931 profile). buyerReference is BT-10 (Leitweg-ID); pass an empty
// string for private-sector buyers, where the element is omitted.
//
// Amounts, tax categories and the BG-23 breakdown come from buildInvoiceDoc, the
// same source the UBL writer uses, so the two formats cannot report different
// numbers for one invoice.
func GenerateCII(invoice models.Invoice, settings models.CompanySettings, buyerReference string) ([]byte, error) {
	doc, err := buildInvoiceDoc(invoice, settings, buyerReference)
	if err != nil {
		return nil, err
	}
	if err := validateInvoiceDoc(doc, ProfileEN16931); err != nil {
		return nil, err
	}
	return renderCII(doc)
}

func renderCII(doc *invoiceDoc) ([]byte, error) {
	out := ciiInvoiceOut{
		XmlnsRSM: ciiNamespaceRSM,
		XmlnsRAM: ciiNamespaceRAM,
		XmlnsUDT: ciiNamespaceUDT,
		Context:  ciiContextOut{Guideline: ciiGuidelineOut{ID: facturXEN16931GuidelineID}},
		Document: ciiDocumentOut{
			ID:            doc.Number,
			TypeCode:      invoiceTypeCodeCommercial,
			IssueDateTime: ciiDateTimeOut{DateTimeString: ciiDateString(doc.IssueDate)},
		},
		Transaction: ciiTransactionOut{
			Lines: make([]ciiLineOut, 0, len(doc.Lines)),
			Agreement: ciiAgreementOut{
				BuyerReference: doc.BuyerReference,
				Seller:         buildCIIParty(doc.Seller),
				Buyer:          buildCIIParty(doc.Buyer),
			},
			Settlement: ciiSettlementOut{
				PaymentReference: doc.Number,
				CurrencyCode:     doc.Currency,
				TradeTax:         make([]ciiTradeTaxOut, 0, len(doc.TaxGroups)),
				Summation: ciiSummationOut{
					LineTotalAmount:     doc.LineTotal.StringFixed(2),
					TaxBasisTotalAmount: doc.LineTotal.StringFixed(2),
					TaxTotalAmount:      ciiCurrencyAmountOut{CurrencyID: doc.Currency, Value: doc.TaxTotal.StringFixed(2)},
					GrandTotalAmount:    doc.GrossTotal.StringFixed(2),
					DuePayableAmount:    doc.GrossTotal.StringFixed(2),
				},
			},
		},
	}

	if note := strings.TrimSpace(doc.Note); note != "" {
		out.Document.Note = &ciiNoteOut{Content: note}
	}
	if !doc.DeliveryDate.IsZero() {
		out.Transaction.Delivery.Event = &ciiDeliveryEventOut{
			OccurrenceDateTime: ciiDateTimeOut{DateTimeString: ciiDateString(doc.DeliveryDate)},
		}
	}
	if iban := strings.TrimSpace(doc.Bank.IBAN); iban != "" {
		means := &ciiPaymentMeansOut{
			TypeCode: paymentMeansCodeSEPACredit,
			Account:  ciiCreditorAccountOut{IBANID: iban, AccountName: doc.Bank.Name},
		}
		if bic := strings.TrimSpace(doc.Bank.BIC); bic != "" {
			means.Institution = &ciiCreditorInstitutionOut{BICID: bic}
		}
		out.Transaction.Settlement.PaymentMeans = means
	}
	if terms := strings.TrimSpace(doc.PaymentTerms); terms != "" || !doc.DueDate.IsZero() {
		pt := &ciiPaymentTermsOut{Description: terms}
		if !doc.DueDate.IsZero() {
			pt.DueDateDateTime = &ciiDateTimeOut{DateTimeString: ciiDateString(doc.DueDate)}
		}
		out.Transaction.Settlement.PaymentTerms = pt
	}

	for _, g := range doc.TaxGroups {
		out.Transaction.Settlement.TradeTax = append(out.Transaction.Settlement.TradeTax, ciiTradeTaxOut{
			CalculatedAmount:      g.TaxAmount.StringFixed(2),
			TypeCode:              taxSchemeVAT,
			ExemptionReason:       g.ExemptionReason,
			BasisAmount:           g.TaxableNet.StringFixed(2),
			CategoryCode:          g.CategoryCode,
			RateApplicablePercent: g.Rate.StringFixed(2),
		})
	}

	for _, l := range doc.Lines {
		out.Transaction.Lines = append(out.Transaction.Lines, ciiLineOut{
			LineDoc:   ciiLineDocOut{LineID: strconv.Itoa(l.Position)},
			Product:   ciiProductOut{Name: l.Description},
			Agreement: ciiLineAgreementOut{NetPrice: ciiNetPriceOut{ChargeAmount: l.UnitPrice.StringFixed(2)}},
			Delivery: ciiLineDeliveryOut{
				BilledQuantity: ciiQuantityOut{UnitCode: unitCodePiece, Value: l.Quantity.StringFixed(2)},
			},
			Settlement: ciiLineSettlementOut{
				TradeTax: ciiLineTaxOut{
					TypeCode:              taxSchemeVAT,
					CategoryCode:          l.CategoryCode,
					RateApplicablePercent: l.TaxRate.StringFixed(2),
				},
				Summation: ciiLineSummationOut{LineTotalAmount: l.Net.StringFixed(2)},
			},
		})
	}

	body, err := xml.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("%w: marshal CII for invoice %s: %s", ErrGenerateFailed, doc.Number, err.Error())
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.Write(body)
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

func buildCIIParty(p docParty) ciiPartyOut {
	out := ciiPartyOut{
		Name: p.Name,
		Address: ciiAddressOut{
			PostcodeCode: p.PostalZone,
			LineOne:      p.Street,
			CityName:     p.City,
			CountryID:    p.Country,
		},
	}
	switch {
	case strings.TrimSpace(p.VATID) != "":
		out.TaxReg = &ciiTaxRegOut{ID: ciiSchemeIDOut{SchemeID: ciiTaxSchemeVAT, Value: strings.TrimSpace(p.VATID)}}
	case strings.TrimSpace(p.TaxRegID) != "":
		out.TaxReg = &ciiTaxRegOut{ID: ciiSchemeIDOut{SchemeID: ciiTaxSchemeFiscal, Value: strings.TrimSpace(p.TaxRegID)}}
	}
	return out
}

func ciiDateString(t time.Time) ciiDateStringOut {
	return ciiDateStringOut{Format: ciiDateFormatCode, Value: t.Format(ciiDateLayout)}
}

// ============================================================================
// Write-side CII structs
//
// Separate from the read-side structs in parser.go for the same reason the UBL
// writer keeps its own: the parser matches element local names in ANY namespace
// so inbound documents parse whatever prefixes the sender chose, while writing
// requires the literal rsm:/ram:/udt: prefixes. encoding/xml cannot express both
// on one field. TestGenerateCII_RoundTripThroughParser keeps the two in agreement.
//
// Field order follows the CII XSD sequences — in particular the line items come
// BEFORE the header groups, which the previous string-template writer got wrong.
// Reordering still parses here but fails schema validation at the receiver.
// ============================================================================

type ciiInvoiceOut struct {
	XMLName  xml.Name `xml:"rsm:CrossIndustryInvoice"`
	XmlnsRSM string   `xml:"xmlns:rsm,attr"`
	XmlnsRAM string   `xml:"xmlns:ram,attr"`
	XmlnsUDT string   `xml:"xmlns:udt,attr"`

	Context     ciiContextOut     `xml:"rsm:ExchangedDocumentContext"`
	Document    ciiDocumentOut    `xml:"rsm:ExchangedDocument"`
	Transaction ciiTransactionOut `xml:"rsm:SupplyChainTradeTransaction"`
}

type ciiContextOut struct {
	Guideline ciiGuidelineOut `xml:"ram:GuidelineSpecifiedDocumentContextParameter"`
}

type ciiGuidelineOut struct {
	ID string `xml:"ram:ID"`
}

type ciiDocumentOut struct {
	ID            string         `xml:"ram:ID"`
	TypeCode      string         `xml:"ram:TypeCode"`
	IssueDateTime ciiDateTimeOut `xml:"ram:IssueDateTime"`
	Note          *ciiNoteOut    `xml:"ram:IncludedNote,omitempty"`
}

type ciiNoteOut struct {
	Content string `xml:"ram:Content"`
}

type ciiDateTimeOut struct {
	DateTimeString ciiDateStringOut `xml:"udt:DateTimeString"`
}

type ciiDateStringOut struct {
	Format string `xml:"format,attr"`
	Value  string `xml:",chardata"`
}

type ciiTransactionOut struct {
	Lines      []ciiLineOut     `xml:"ram:IncludedSupplyChainTradeLineItem"`
	Agreement  ciiAgreementOut  `xml:"ram:ApplicableHeaderTradeAgreement"`
	Delivery   ciiDeliveryOut   `xml:"ram:ApplicableHeaderTradeDelivery"`
	Settlement ciiSettlementOut `xml:"ram:ApplicableHeaderTradeSettlement"`
}

type ciiAgreementOut struct {
	BuyerReference string      `xml:"ram:BuyerReference,omitempty"`
	Seller         ciiPartyOut `xml:"ram:SellerTradeParty"`
	Buyer          ciiPartyOut `xml:"ram:BuyerTradeParty"`
}

type ciiPartyOut struct {
	Name    string        `xml:"ram:Name"`
	Address ciiAddressOut `xml:"ram:PostalTradeAddress"`
	TaxReg  *ciiTaxRegOut `xml:"ram:SpecifiedTaxRegistration,omitempty"`
}

type ciiAddressOut struct {
	PostcodeCode string `xml:"ram:PostcodeCode,omitempty"`
	LineOne      string `xml:"ram:LineOne,omitempty"`
	CityName     string `xml:"ram:CityName,omitempty"`
	CountryID    string `xml:"ram:CountryID"`
}

type ciiTaxRegOut struct {
	ID ciiSchemeIDOut `xml:"ram:ID"`
}

type ciiSchemeIDOut struct {
	SchemeID string `xml:"schemeID,attr"`
	Value    string `xml:",chardata"`
}

type ciiDeliveryOut struct {
	Event *ciiDeliveryEventOut `xml:"ram:ActualDeliverySupplyChainEvent,omitempty"`
}

type ciiDeliveryEventOut struct {
	OccurrenceDateTime ciiDateTimeOut `xml:"ram:OccurrenceDateTime"`
}

type ciiSettlementOut struct {
	PaymentReference string              `xml:"ram:PaymentReference,omitempty"`
	CurrencyCode     string              `xml:"ram:InvoiceCurrencyCode"`
	PaymentMeans     *ciiPaymentMeansOut `xml:"ram:SpecifiedTradeSettlementPaymentMeans,omitempty"`
	TradeTax         []ciiTradeTaxOut    `xml:"ram:ApplicableTradeTax"`
	PaymentTerms     *ciiPaymentTermsOut `xml:"ram:SpecifiedTradePaymentTerms,omitempty"`
	Summation        ciiSummationOut     `xml:"ram:SpecifiedTradeSettlementHeaderMonetarySummation"`
}

type ciiPaymentMeansOut struct {
	TypeCode    string                     `xml:"ram:TypeCode"`
	Account     ciiCreditorAccountOut      `xml:"ram:PayeePartyCreditorFinancialAccount"`
	Institution *ciiCreditorInstitutionOut `xml:"ram:PayeeSpecifiedCreditorFinancialInstitution,omitempty"`
}

type ciiCreditorAccountOut struct {
	IBANID      string `xml:"ram:IBANID"`
	AccountName string `xml:"ram:AccountName,omitempty"`
}

type ciiCreditorInstitutionOut struct {
	BICID string `xml:"ram:BICID"`
}

type ciiTradeTaxOut struct {
	CalculatedAmount      string `xml:"ram:CalculatedAmount"`
	TypeCode              string `xml:"ram:TypeCode"`
	ExemptionReason       string `xml:"ram:ExemptionReason,omitempty"`
	BasisAmount           string `xml:"ram:BasisAmount"`
	CategoryCode          string `xml:"ram:CategoryCode"`
	RateApplicablePercent string `xml:"ram:RateApplicablePercent"`
}

type ciiPaymentTermsOut struct {
	Description     string          `xml:"ram:Description,omitempty"`
	DueDateDateTime *ciiDateTimeOut `xml:"ram:DueDateDateTime,omitempty"`
}

type ciiSummationOut struct {
	LineTotalAmount     string               `xml:"ram:LineTotalAmount"`
	TaxBasisTotalAmount string               `xml:"ram:TaxBasisTotalAmount"`
	TaxTotalAmount      ciiCurrencyAmountOut `xml:"ram:TaxTotalAmount"`
	GrandTotalAmount    string               `xml:"ram:GrandTotalAmount"`
	DuePayableAmount    string               `xml:"ram:DuePayableAmount"`
}

type ciiCurrencyAmountOut struct {
	CurrencyID string `xml:"currencyID,attr"`
	Value      string `xml:",chardata"`
}

type ciiLineOut struct {
	LineDoc    ciiLineDocOut        `xml:"ram:AssociatedDocumentLineDocument"`
	Product    ciiProductOut        `xml:"ram:SpecifiedTradeProduct"`
	Agreement  ciiLineAgreementOut  `xml:"ram:SpecifiedLineTradeAgreement"`
	Delivery   ciiLineDeliveryOut   `xml:"ram:SpecifiedLineTradeDelivery"`
	Settlement ciiLineSettlementOut `xml:"ram:SpecifiedLineTradeSettlement"`
}

type ciiLineDocOut struct {
	LineID string `xml:"ram:LineID"`
}

type ciiProductOut struct {
	Name string `xml:"ram:Name"`
}

type ciiLineAgreementOut struct {
	NetPrice ciiNetPriceOut `xml:"ram:NetPriceProductTradePrice"`
}

type ciiNetPriceOut struct {
	ChargeAmount string `xml:"ram:ChargeAmount"`
}

type ciiLineDeliveryOut struct {
	BilledQuantity ciiQuantityOut `xml:"ram:BilledQuantity"`
}

type ciiQuantityOut struct {
	UnitCode string `xml:"unitCode,attr"`
	Value    string `xml:",chardata"`
}

type ciiLineSettlementOut struct {
	TradeTax  ciiLineTaxOut       `xml:"ram:ApplicableTradeTax"`
	Summation ciiLineSummationOut `xml:"ram:SpecifiedTradeSettlementLineMonetarySummation"`
}

type ciiLineTaxOut struct {
	TypeCode              string `xml:"ram:TypeCode"`
	CategoryCode          string `xml:"ram:CategoryCode"`
	RateApplicablePercent string `xml:"ram:RateApplicablePercent"`
}

type ciiLineSummationOut struct {
	LineTotalAmount string `xml:"ram:LineTotalAmount"`
}
