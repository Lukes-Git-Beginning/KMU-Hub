package pdf

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ZUGFeRD profiles supported (Factur-X / EN 16931)
const (
	ZUGFeRDProfileMinimum  = "MINIMUM"
	ZUGFeRDProfileBasicWL  = "BASIC_WL"
	ZUGFeRDProfileEN16931  = "EN16931"
)

// GenerateZUGFeRDXML generates a Factur-X / ZUGFeRD 2.1 XML (Cross-Industry Invoice, EN16931 profile).
func GenerateZUGFeRDXML(invoice models.Invoice, settings models.CompanySettings) ([]byte, error) {
	if invoice.InvoiceDate.IsZero() {
		return nil, fmt.Errorf("zugferd: invoice %s is missing the issue date (BT-2)", invoice.InvoiceNumber)
	}
	if invoice.DueDate.IsZero() {
		return nil, fmt.Errorf("zugferd: invoice %s is missing the due date (BT-9)", invoice.InvoiceNumber)
	}
	if err := ValidateCompanySettingsForPDF(settings); err != nil {
		return nil, fmt.Errorf("zugferd: %w", err)
	}
	lineItems, err := parseLineItems(invoice.LineItems)
	if err != nil {
		return nil, fmt.Errorf("parse line items for zugferd: %w", err)
	}

	var lineBuf bytes.Buffer
	for i, item := range lineItems {
		lineBuf.WriteString(fmt.Sprintf(`
		<ram:IncludedSupplyChainTradeLineItem>
			<ram:AssociatedDocumentLineDocument>
				<ram:LineID>%d</ram:LineID>
			</ram:AssociatedDocumentLineDocument>
			<ram:SpecifiedTradeProduct>
				<ram:Name>%s</ram:Name>
			</ram:SpecifiedTradeProduct>
			<ram:SpecifiedLineTradeAgreement>
				<ram:NetPriceProductTradePrice>
					<ram:ChargeAmount>%s</ram:ChargeAmount>
				</ram:NetPriceProductTradePrice>
			</ram:SpecifiedLineTradeAgreement>
			<ram:SpecifiedLineTradeDelivery>
				<ram:BilledQuantity unitCode="C62">%s</ram:BilledQuantity>
			</ram:SpecifiedLineTradeDelivery>
			<ram:SpecifiedLineTradeSettlement>
				<ram:ApplicableTradeTax>
					<ram:TypeCode>VAT</ram:TypeCode>
					<ram:RateApplicablePercent>%s</ram:RateApplicablePercent>
				</ram:ApplicableTradeTax>
				<ram:SpecifiedTradeSettlementLineMonetarySummation>
					<ram:LineTotalAmount>%s</ram:LineTotalAmount>
				</ram:SpecifiedTradeSettlementLineMonetarySummation>
			</ram:SpecifiedLineTradeSettlement>
		</ram:IncludedSupplyChainTradeLineItem>`,
			i+1,
			xmlEscape(item.Description),
			item.UnitPrice.StringFixed(2),
			item.Quantity.StringFixed(2),
			item.TaxRate.StringFixed(2),
			item.LineTotal.StringFixed(2),
		))
	}

	dueDateStr := invoice.DueDate.Format("20060102")
	invoiceDateStr := invoice.InvoiceDate.Format("20060102")
	currency := invoice.Currency
	if currency == "" {
		currency = models.DefaultCurrency
	}

	// BT-72 Actual delivery date (Leistungsdatum): use the explicit delivery date
	// when set, otherwise fall back to the issue date. EN16931 requires a non-empty
	// ApplicableHeaderTradeDelivery for a §14 UStG invoice.
	deliveryDate := invoice.InvoiceDate
	if invoice.DeliveryDate != nil && !invoice.DeliveryDate.IsZero() {
		deliveryDate = *invoice.DeliveryDate
	}
	deliveryDateStr := deliveryDate.Format("20060102")

	// BG-23 VAT breakdown (one ram:ApplicableTradeTax per category/rate). Without it
	// no EN16931 validator accepts the invoice.
	headerTradeTax := buildHeaderTradeTax(lineItems, invoice.TaxMode, invoice.Subtotal)

	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rsm:CrossIndustryInvoice
	xmlns:rsm="urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
	xmlns:ram="urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100"
	xmlns:udt="urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100">

	<rsm:ExchangedDocumentContext>
		<ram:GuidelineSpecifiedDocumentContextParameter>
			<ram:ID>urn:cen.eu:en16931:2017#compliant#urn:factur-x.eu:1p0:en16931</ram:ID>
		</ram:GuidelineSpecifiedDocumentContextParameter>
	</rsm:ExchangedDocumentContext>

	<rsm:ExchangedDocument>
		<ram:ID>%s</ram:ID>
		<ram:TypeCode>380</ram:TypeCode>
		<ram:IssueDateTime>
			<udt:DateTimeString format="102">%s</udt:DateTimeString>
		</ram:IssueDateTime>
	</rsm:ExchangedDocument>

	<rsm:SupplyChainTradeTransaction>
		<ram:ApplicableHeaderTradeAgreement>
			<ram:SellerTradeParty>
				<ram:Name>%s</ram:Name>
				<ram:PostalTradeAddress>
					<ram:PostcodeCode>%s</ram:PostcodeCode>
					<ram:LineOne>%s</ram:LineOne>
					<ram:CityName>%s</ram:CityName>
					<ram:CountryID>%s</ram:CountryID>
				</ram:PostalTradeAddress>
				<ram:SpecifiedTaxRegistration>
					<ram:ID schemeID="VA">%s</ram:ID>
				</ram:SpecifiedTaxRegistration>
			</ram:SellerTradeParty>
			<ram:BuyerTradeParty>
				<ram:Name>%s</ram:Name>
			</ram:BuyerTradeParty>
		</ram:ApplicableHeaderTradeAgreement>

		<ram:ApplicableHeaderTradeDelivery>
			<ram:ActualDeliverySupplyChainEvent>
				<ram:OccurrenceDateTime>
					<udt:DateTimeString format="102">%s</udt:DateTimeString>
				</ram:OccurrenceDateTime>
			</ram:ActualDeliverySupplyChainEvent>
		</ram:ApplicableHeaderTradeDelivery>

		<ram:ApplicableHeaderTradeSettlement>
			<ram:PaymentReference>%s</ram:PaymentReference>
			<ram:InvoiceCurrencyCode>%s</ram:InvoiceCurrencyCode>%s
			<ram:SpecifiedTradePaymentTerms>
				<ram:DueDateDateTime>
					<udt:DateTimeString format="102">%s</udt:DateTimeString>
				</ram:DueDateDateTime>
			</ram:SpecifiedTradePaymentTerms>
			<ram:SpecifiedTradeSettlementHeaderMonetarySummation>
				<ram:LineTotalAmount>%s</ram:LineTotalAmount>
				<ram:TaxBasisTotalAmount>%s</ram:TaxBasisTotalAmount>
				<ram:TaxTotalAmount currencyID="%s">%s</ram:TaxTotalAmount>
				<ram:GrandTotalAmount>%s</ram:GrandTotalAmount>
				<ram:DuePayableAmount>%s</ram:DuePayableAmount>
			</ram:SpecifiedTradeSettlementHeaderMonetarySummation>
		</ram:ApplicableHeaderTradeSettlement>

		%s
	</rsm:SupplyChainTradeTransaction>
</rsm:CrossIndustryInvoice>`,
		xmlEscape(invoice.InvoiceNumber),
		invoiceDateStr,
		xmlEscape(settings.Name),
		xmlEscape(settings.PLZ),
		xmlEscape(settings.Street),
		xmlEscape(settings.City),
		countryCode(settings.Country),
		xmlEscape(settings.UStIDNr),
		xmlEscape(invoice.CustomerName),
		deliveryDateStr,
		xmlEscape(invoice.InvoiceNumber),
		currency,
		headerTradeTax,
		dueDateStr,
		invoice.Subtotal.StringFixed(2),
		invoice.Subtotal.StringFixed(2),
		currency,
		invoice.TotalTax.StringFixed(2),
		invoice.GrossTotal.StringFixed(2),
		invoice.GrossTotal.StringFixed(2),
		lineBuf.String(),
	)

	return []byte(xml), nil
}

// buildHeaderTradeTax renders the EN16931 BG-23 VAT breakdown — one
// ram:ApplicableTradeTax per VAT category/rate — for the header settlement block.
// Standard mode groups line items by their rate (basis = sum of line nets,
// calculated = sum of per-line rounded tax, matching tax.Calculate). The tax-free
// modes emit a single exempt category with the statutory exemption reason.
func buildHeaderTradeTax(items []models.LineItem, taxMode string, subtotal decimal.Decimal) string {
	var buf bytes.Buffer

	switch taxMode {
	case models.TaxModeReverseCharge:
		buf.WriteString(fmt.Sprintf(`
			<ram:ApplicableTradeTax>
				<ram:CalculatedAmount>0.00</ram:CalculatedAmount>
				<ram:TypeCode>VAT</ram:TypeCode>
				<ram:ExemptionReason>Reverse charge</ram:ExemptionReason>
				<ram:BasisAmount>%s</ram:BasisAmount>
				<ram:CategoryCode>AE</ram:CategoryCode>
				<ram:RateApplicablePercent>0.00</ram:RateApplicablePercent>
			</ram:ApplicableTradeTax>`, subtotal.StringFixed(2)))
	case models.TaxModeKleinunternehmer:
		buf.WriteString(fmt.Sprintf(`
			<ram:ApplicableTradeTax>
				<ram:CalculatedAmount>0.00</ram:CalculatedAmount>
				<ram:TypeCode>VAT</ram:TypeCode>
				<ram:ExemptionReason>Kleinunternehmer gemaess Paragraph 19 UStG</ram:ExemptionReason>
				<ram:BasisAmount>%s</ram:BasisAmount>
				<ram:CategoryCode>E</ram:CategoryCode>
				<ram:RateApplicablePercent>0.00</ram:RateApplicablePercent>
			</ram:ApplicableTradeTax>`, subtotal.StringFixed(2)))
	default: // standard — one block per distinct VAT rate, in first-seen order
		hundred := decimal.NewFromInt(100)
		type rateGroup struct {
			rate  decimal.Decimal
			basis decimal.Decimal
			tax   decimal.Decimal
		}
		order := make([]string, 0)
		groups := make(map[string]*rateGroup)
		for _, item := range items {
			lineTotal := item.Quantity.Mul(item.UnitPrice)
			lineTax := lineTotal.Mul(item.TaxRate).Div(hundred).Round(2)
			key := item.TaxRate.StringFixed(2)
			g, ok := groups[key]
			if !ok {
				g = &rateGroup{rate: item.TaxRate}
				groups[key] = g
				order = append(order, key)
			}
			g.basis = g.basis.Add(lineTotal)
			g.tax = g.tax.Add(lineTax)
		}
		for _, key := range order {
			g := groups[key]
			buf.WriteString(fmt.Sprintf(`
			<ram:ApplicableTradeTax>
				<ram:CalculatedAmount>%s</ram:CalculatedAmount>
				<ram:TypeCode>VAT</ram:TypeCode>
				<ram:BasisAmount>%s</ram:BasisAmount>
				<ram:CategoryCode>S</ram:CategoryCode>
				<ram:RateApplicablePercent>%s</ram:RateApplicablePercent>
			</ram:ApplicableTradeTax>`, g.tax.StringFixed(2), g.basis.StringFixed(2), g.rate.StringFixed(2)))
		}
	}

	return buf.String()
}

// EmbedZUGFeRDXML embeds the Factur-X XML into an existing PDF as /EmbeddedFiles.
// The resulting PDF/A-3b compliant file has the XML attachment visible to e-invoice validators.
func EmbedZUGFeRDXML(pdfBytes []byte, xmlBytes []byte, invoiceNumber string) ([]byte, error) {
	fileName := fmt.Sprintf("factur-x_%s_%s.xml", invoiceNumber, time.Now().Format("20060102"))

	// Write XML to temp file (pdfcpu AddAttachments requires file paths)
	tmpDir, err := os.MkdirTemp("", "zugferd-*")
	if err != nil {
		slog.Warn("zugferd embed skipped: mkdir temp failed, returning PDF without XML attachment",
			"invoice_number", invoiceNumber, "error", err)
		return pdfBytes, nil
	}
	defer os.RemoveAll(tmpDir)

	xmlPath := filepath.Join(tmpDir, fileName)
	if err := os.WriteFile(xmlPath, xmlBytes, 0o600); err != nil {
		slog.Warn("zugferd embed skipped: write temp XML failed, returning PDF without XML attachment",
			"invoice_number", invoiceNumber, "error", err)
		return pdfBytes, nil
	}

	pdfReader := bytes.NewReader(pdfBytes)
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	var outBuf bytes.Buffer
	if err := api.AddAttachments(io.ReadSeeker(pdfReader), &outBuf, []string{xmlPath}, false, conf); err != nil {
		slog.Warn("zugferd embed skipped: AddAttachments failed, returning PDF without XML attachment",
			"invoice_number", invoiceNumber, "error", err)
		return pdfBytes, nil
	}

	if outBuf.Len() == 0 {
		slog.Warn("zugferd embed skipped: AddAttachments produced empty output, returning original PDF",
			"invoice_number", invoiceNumber)
		return pdfBytes, nil
	}

	return outBuf.Bytes(), nil
}

// xmlEscape escapes special XML characters.
func xmlEscape(s string) string {
	var buf bytes.Buffer
	for _, c := range s {
		switch c {
		case '<':
			buf.WriteString("&lt;")
		case '>':
			buf.WriteString("&gt;")
		case '&':
			buf.WriteString("&amp;")
		case '"':
			buf.WriteString("&quot;")
		case '\'':
			buf.WriteString("&apos;")
		default:
			buf.WriteRune(c)
		}
	}
	return buf.String()
}

// countryCode returns a 2-letter ISO country code from a country name.
// Falls back to "DE" for German company defaults.
func countryCode(country string) string {
	switch country {
	case "Deutschland", "Germany", "DE":
		return "DE"
	case "Österreich", "Austria", "AT":
		return "AT"
	case "Schweiz", "Switzerland", "CH":
		return "CH"
	default:
		if len(country) == 2 {
			return country
		}
		return "DE"
	}
}
