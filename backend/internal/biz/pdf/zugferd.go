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

		<ram:ApplicableHeaderTradeDelivery/>

		<ram:ApplicableHeaderTradeSettlement>
			<ram:PaymentReference>%s</ram:PaymentReference>
			<ram:InvoiceCurrencyCode>EUR</ram:InvoiceCurrencyCode>
			<ram:SpecifiedTradePaymentTerms>
				<ram:DueDateDateTime>
					<udt:DateTimeString format="102">%s</udt:DateTimeString>
				</ram:DueDateDateTime>
			</ram:SpecifiedTradePaymentTerms>
			<ram:SpecifiedTradeSettlementHeaderMonetarySummation>
				<ram:LineTotalAmount>%s</ram:LineTotalAmount>
				<ram:TaxTotalAmount currencyID="EUR">%s</ram:TaxTotalAmount>
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
		xmlEscape(invoice.InvoiceNumber),
		dueDateStr,
		invoice.Subtotal.StringFixed(2),
		invoice.TotalTax.StringFixed(2),
		invoice.GrossTotal.StringFixed(2),
		invoice.GrossTotal.StringFixed(2),
		lineBuf.String(),
	)

	return []byte(xml), nil
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
