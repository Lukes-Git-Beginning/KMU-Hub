package einvoice

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Untrusted-input hardening for the inbound e-invoice parser (ParseCII/ParseUBL).
//
// This file is the largest unkontrolled input surface in the finance module:
// XML from arbitrary senders, turned into booking data. Every case here is
// either a security property that must hold, or a documented decision about
// what the parser deliberately does NOT check.
// ============================================================================

// parseCIIWithTimeout guards against an untrusted document making ParseCII hang
// (e.g. a maliciously deep or wide structure). A parser that never returns is as
// much a DoS as one that panics.
func parseCIIWithTimeout(t *testing.T, xmlData []byte, timeout time.Duration) (*ParsedInvoice, error) {
	t.Helper()
	type result struct {
		inv *ParsedInvoice
		err error
	}
	ch := make(chan result, 1)
	go func() {
		inv, err := ParseCII(xmlData)
		ch <- result{inv, err}
	}()
	select {
	case r := <-ch:
		return r.inv, r.err
	case <-time.After(timeout):
		t.Fatalf("ParseCII did not return within %s — possible DoS via untrusted XML", timeout)
		return nil, nil
	}
}

// ============================================================================
// XXE (XML External Entity) — must not resolve external/SYSTEM entities.
// ============================================================================

func TestParseCII_ExternalEntityIsNotExpanded(t *testing.T) {
	// &xxe; references a SYSTEM entity that would read a local file if the
	// decoder expanded it. Go's encoding/xml has no DTD entity support and does
	// not populate a custom Entity map here, so this must fail to parse rather
	// than silently substitute file content into SupplierName.
	xxe := []byte(`<?xml version="1.0"?>
<!DOCTYPE rsm:CrossIndustryInvoice [
  <!ENTITY xxe SYSTEM "file:///etc/passwd">
]>
<rsm:CrossIndustryInvoice
	xmlns:rsm="urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
	xmlns:ram="urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100"
	xmlns:udt="urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100">
	<rsm:ExchangedDocument>
		<ram:ID>RE-XXE-001</ram:ID>
		<ram:IssueDateTime><udt:DateTimeString format="102">20240115</udt:DateTimeString></ram:IssueDateTime>
	</rsm:ExchangedDocument>
	<rsm:SupplyChainTradeTransaction>
		<ram:ApplicableHeaderTradeAgreement>
			<ram:SellerTradeParty><ram:Name>&xxe;</ram:Name></ram:SellerTradeParty>
			<ram:BuyerTradeParty><ram:Name>Y</ram:Name></ram:BuyerTradeParty>
		</ram:ApplicableHeaderTradeAgreement>
		<ram:ApplicableHeaderTradeSettlement>
			<ram:InvoiceCurrencyCode>EUR</ram:InvoiceCurrencyCode>
			<ram:SpecifiedTradeSettlementHeaderMonetarySummation>
				<ram:LineTotalAmount>0</ram:LineTotalAmount>
				<ram:TaxTotalAmount>0</ram:TaxTotalAmount>
				<ram:GrandTotalAmount>100.00</ram:GrandTotalAmount>
			</ram:SpecifiedTradeSettlementHeaderMonetarySummation>
		</ram:ApplicableHeaderTradeSettlement>
	</rsm:SupplyChainTradeTransaction>
</rsm:CrossIndustryInvoice>`)

	parsed, err := parseCIIWithTimeout(t, xxe, 5*time.Second)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrParseFailed)
	assert.Nil(t, parsed)
	// The failure mode itself must not be the leak: no file content anywhere.
	assert.NotContains(t, err.Error(), "root:")
}

// ============================================================================
// Billion laughs / entity expansion — must not expand nested custom entities.
// ============================================================================

func TestParseCII_EntityExpansionBombIsNotExpanded(t *testing.T) {
	// Classic exponential entity blow-up (10 entities nested 3 levels deep would
	// expand to 1000x). Same underlying protection as the XXE case — Go's
	// decoder does not resolve DTD-declared entities at all — but this is worth
	// its own test because a hypothetical fix for the XXE case (e.g. wiring a
	// permissive Entity map for legitimate accented-character entities) could
	// reintroduce this without reintroducing that.
	lol := []byte(`<?xml version="1.0"?>
<!DOCTYPE rsm:CrossIndustryInvoice [
  <!ENTITY lol "lol">
  <!ENTITY lol2 "&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;">
  <!ENTITY lol3 "&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;">
]>
<rsm:CrossIndustryInvoice
	xmlns:rsm="urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
	xmlns:ram="urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100">
	<rsm:ExchangedDocument>
		<ram:ID>&lol3;</ram:ID>
	</rsm:ExchangedDocument>
</rsm:CrossIndustryInvoice>`)

	parsed, err := parseCIIWithTimeout(t, lol, 5*time.Second)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrParseFailed)
	assert.Nil(t, parsed)
}

// ============================================================================
// Deep nesting — must not stack-overflow or hang, regardless of parse outcome.
// ============================================================================

func TestParseCII_DeeplyNestedUnknownElement_DoesNotHangOrPanic(t *testing.T) {
	// ApplicableHeaderTradeDelivery is present in the wire format but has no
	// corresponding struct field in ciiSupplyChainTrade — encoding/xml skips it
	// token by token. A document that nests 50k levels inside that skipped
	// region must still be handled: the skip walks every token, so this proves
	// the walk itself is bounded and doesn't blow the goroutine stack.
	const depth = 50000
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?>
<rsm:CrossIndustryInvoice
	xmlns:rsm="urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
	xmlns:ram="urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100"
	xmlns:udt="urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100">
	<rsm:ExchangedDocument>
		<ram:ID>RE-DEEP-001</ram:ID>
		<ram:IssueDateTime><udt:DateTimeString format="102">20240115</udt:DateTimeString></ram:IssueDateTime>
	</rsm:ExchangedDocument>
	<rsm:SupplyChainTradeTransaction>
		<ram:ApplicableHeaderTradeAgreement>
			<ram:SellerTradeParty><ram:Name>X</ram:Name></ram:SellerTradeParty>
			<ram:BuyerTradeParty><ram:Name>Y</ram:Name></ram:BuyerTradeParty>
		</ram:ApplicableHeaderTradeAgreement>
		<ram:ApplicableHeaderTradeDelivery>`)
	for range depth {
		b.WriteString("<a>")
	}
	for range depth {
		b.WriteString("</a>")
	}
	b.WriteString(`</ram:ApplicableHeaderTradeDelivery>
		<ram:ApplicableHeaderTradeSettlement>
			<ram:InvoiceCurrencyCode>EUR</ram:InvoiceCurrencyCode>
			<ram:SpecifiedTradeSettlementHeaderMonetarySummation>
				<ram:LineTotalAmount>0</ram:LineTotalAmount>
				<ram:TaxTotalAmount>0</ram:TaxTotalAmount>
				<ram:GrandTotalAmount>100.00</ram:GrandTotalAmount>
			</ram:SpecifiedTradeSettlementHeaderMonetarySummation>
		</ram:ApplicableHeaderTradeSettlement>
	</rsm:SupplyChainTradeTransaction>
</rsm:CrossIndustryInvoice>`)

	// No lines and a non-zero gross total: expected to be rejected by the new
	// totals-consistency check, not by the deep nesting. The point of this test
	// is "does not hang or panic", which require.Error below already proves by
	// virtue of returning at all.
	parsed, err := parseCIIWithTimeout(t, []byte(b.String()), 10*time.Second)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrParseFailed)
	assert.Nil(t, parsed)
}

// ============================================================================
// Large document — many line items must parse in bounded time, not O(n^2).
// ============================================================================

func TestParseCII_LargeLineItemCount_ParsesInBoundedTime(t *testing.T) {
	const lineCount = 3000
	const unitPrice = "10.00"

	var lines strings.Builder
	for i := 1; i <= lineCount; i++ {
		fmt.Fprintf(&lines, `
			<ram:IncludedSupplyChainTradeLineItem>
				<ram:AssociatedDocumentLineDocument><ram:LineID>%d</ram:LineID></ram:AssociatedDocumentLineDocument>
				<ram:SpecifiedTradeProduct><ram:Name>Position %d</ram:Name></ram:SpecifiedTradeProduct>
				<ram:SpecifiedLineTradeAgreement><ram:NetPriceProductTradePrice><ram:ChargeAmount>%s</ram:ChargeAmount></ram:NetPriceProductTradePrice></ram:SpecifiedLineTradeAgreement>
				<ram:SpecifiedLineTradeDelivery><ram:BilledQuantity unitCode="C62">1.00</ram:BilledQuantity></ram:SpecifiedLineTradeDelivery>
				<ram:SpecifiedLineTradeSettlement>
					<ram:ApplicableTradeTax><ram:TypeCode>VAT</ram:TypeCode><ram:RateApplicablePercent>19.00</ram:RateApplicablePercent></ram:ApplicableTradeTax>
					<ram:SpecifiedTradeSettlementLineMonetarySummation><ram:LineTotalAmount>%s</ram:LineTotalAmount></ram:SpecifiedTradeSettlementLineMonetarySummation>
				</ram:SpecifiedLineTradeSettlement>
			</ram:IncludedSupplyChainTradeLineItem>`, i, i, unitPrice, unitPrice)
	}

	// lineCount * 10.00 net, 19% tax, exact decimal arithmetic (no rounding).
	xmlData := fmt.Sprintf(`<?xml version="1.0"?>
<rsm:CrossIndustryInvoice
	xmlns:rsm="urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
	xmlns:ram="urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100"
	xmlns:udt="urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100">
	<rsm:ExchangedDocument>
		<ram:ID>RE-LARGE-001</ram:ID>
		<ram:IssueDateTime><udt:DateTimeString format="102">20240115</udt:DateTimeString></ram:IssueDateTime>
	</rsm:ExchangedDocument>
	<rsm:SupplyChainTradeTransaction>
		<ram:ApplicableHeaderTradeAgreement>
			<ram:SellerTradeParty><ram:Name>Grossauftrag GmbH</ram:Name></ram:SellerTradeParty>
			<ram:BuyerTradeParty><ram:Name>Kaeufer AG</ram:Name></ram:BuyerTradeParty>
		</ram:ApplicableHeaderTradeAgreement>
		<ram:ApplicableHeaderTradeSettlement>
			<ram:InvoiceCurrencyCode>EUR</ram:InvoiceCurrencyCode>
			<ram:SpecifiedTradeSettlementHeaderMonetarySummation>
				<ram:LineTotalAmount>30000.00</ram:LineTotalAmount>
				<ram:TaxTotalAmount>5700.00</ram:TaxTotalAmount>
				<ram:GrandTotalAmount>35700.00</ram:GrandTotalAmount>
			</ram:SpecifiedTradeSettlementHeaderMonetarySummation>
		</ram:ApplicableHeaderTradeSettlement>
		%s
	</rsm:SupplyChainTradeTransaction>
</rsm:CrossIndustryInvoice>`, lines.String())

	start := time.Now()
	parsed, err := parseCIIWithTimeout(t, []byte(xmlData), 10*time.Second)
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, parsed)
	assert.Len(t, parsed.LineItems, lineCount)
	assert.Equal(t, "30000.00", parsed.Subtotal.StringFixed(2))
	assert.Equal(t, "35700.00", parsed.GrossTotal.StringFixed(2))
	assert.Less(t, elapsed, 5*time.Second, "parsing %d line items took %s — check for O(n^2) behaviour", lineCount, elapsed)
}

// ============================================================================
// Totals consistency — a document whose own declared figures don't add up
// must be rejected, not silently imported as booking data.
// ============================================================================

func TestParseCII_TotalsMismatch_LineItemsDoNotMatchDeclaredSubtotal(t *testing.T) {
	// Header declares a 300.00 subtotal but the single line only totals 150.00.
	xmlData := []byte(`<?xml version="1.0"?>
<rsm:CrossIndustryInvoice
	xmlns:rsm="urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
	xmlns:ram="urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100"
	xmlns:udt="urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100">
	<rsm:ExchangedDocument>
		<ram:ID>RE-MISMATCH-LINES-001</ram:ID>
		<ram:IssueDateTime><udt:DateTimeString format="102">20240115</udt:DateTimeString></ram:IssueDateTime>
	</rsm:ExchangedDocument>
	<rsm:SupplyChainTradeTransaction>
		<ram:ApplicableHeaderTradeAgreement>
			<ram:SellerTradeParty><ram:Name>Lieferant GmbH</ram:Name></ram:SellerTradeParty>
			<ram:BuyerTradeParty><ram:Name>Kaeufer AG</ram:Name></ram:BuyerTradeParty>
		</ram:ApplicableHeaderTradeAgreement>
		<ram:ApplicableHeaderTradeSettlement>
			<ram:InvoiceCurrencyCode>EUR</ram:InvoiceCurrencyCode>
			<ram:SpecifiedTradeSettlementHeaderMonetarySummation>
				<ram:LineTotalAmount>300.00</ram:LineTotalAmount>
				<ram:TaxTotalAmount>57.00</ram:TaxTotalAmount>
				<ram:GrandTotalAmount>357.00</ram:GrandTotalAmount>
			</ram:SpecifiedTradeSettlementHeaderMonetarySummation>
		</ram:ApplicableHeaderTradeSettlement>
		<ram:IncludedSupplyChainTradeLineItem>
			<ram:AssociatedDocumentLineDocument><ram:LineID>1</ram:LineID></ram:AssociatedDocumentLineDocument>
			<ram:SpecifiedTradeProduct><ram:Name>Beratungsleistung</ram:Name></ram:SpecifiedTradeProduct>
			<ram:SpecifiedLineTradeAgreement><ram:NetPriceProductTradePrice><ram:ChargeAmount>150.00</ram:ChargeAmount></ram:NetPriceProductTradePrice></ram:SpecifiedLineTradeAgreement>
			<ram:SpecifiedLineTradeDelivery><ram:BilledQuantity unitCode="C62">1.00</ram:BilledQuantity></ram:SpecifiedLineTradeDelivery>
			<ram:SpecifiedLineTradeSettlement>
				<ram:ApplicableTradeTax><ram:TypeCode>VAT</ram:TypeCode><ram:RateApplicablePercent>19.00</ram:RateApplicablePercent></ram:ApplicableTradeTax>
				<ram:SpecifiedTradeSettlementLineMonetarySummation><ram:LineTotalAmount>150.00</ram:LineTotalAmount></ram:SpecifiedTradeSettlementLineMonetarySummation>
			</ram:SpecifiedLineTradeSettlement>
		</ram:IncludedSupplyChainTradeLineItem>
	</rsm:SupplyChainTradeTransaction>
</rsm:CrossIndustryInvoice>`)

	parsed, err := ParseCII(xmlData)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrParseFailed)
	assert.Contains(t, err.Error(), "line items sum")
	assert.Nil(t, parsed)
}

func TestParseCII_TotalsMismatch_SubtotalPlusTaxDoesNotMatchGrossTotal(t *testing.T) {
	// Subtotal (300) + tax (57) = 357, but the document declares a gross total
	// of 999.00 — the kind of tampering or transmission error that must not
	// become a silent booking entry.
	xmlData := []byte(`<?xml version="1.0"?>
<rsm:CrossIndustryInvoice
	xmlns:rsm="urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
	xmlns:ram="urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100"
	xmlns:udt="urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100">
	<rsm:ExchangedDocument>
		<ram:ID>RE-MISMATCH-GROSS-001</ram:ID>
		<ram:IssueDateTime><udt:DateTimeString format="102">20240115</udt:DateTimeString></ram:IssueDateTime>
	</rsm:ExchangedDocument>
	<rsm:SupplyChainTradeTransaction>
		<ram:ApplicableHeaderTradeAgreement>
			<ram:SellerTradeParty><ram:Name>Lieferant GmbH</ram:Name></ram:SellerTradeParty>
			<ram:BuyerTradeParty><ram:Name>Kaeufer AG</ram:Name></ram:BuyerTradeParty>
		</ram:ApplicableHeaderTradeAgreement>
		<ram:ApplicableHeaderTradeSettlement>
			<ram:InvoiceCurrencyCode>EUR</ram:InvoiceCurrencyCode>
			<ram:SpecifiedTradeSettlementHeaderMonetarySummation>
				<ram:LineTotalAmount>300.00</ram:LineTotalAmount>
				<ram:TaxTotalAmount>57.00</ram:TaxTotalAmount>
				<ram:GrandTotalAmount>999.00</ram:GrandTotalAmount>
			</ram:SpecifiedTradeSettlementHeaderMonetarySummation>
		</ram:ApplicableHeaderTradeSettlement>
		<ram:IncludedSupplyChainTradeLineItem>
			<ram:AssociatedDocumentLineDocument><ram:LineID>1</ram:LineID></ram:AssociatedDocumentLineDocument>
			<ram:SpecifiedTradeProduct><ram:Name>Beratungsleistung</ram:Name></ram:SpecifiedTradeProduct>
			<ram:SpecifiedLineTradeAgreement><ram:NetPriceProductTradePrice><ram:ChargeAmount>150.00</ram:ChargeAmount></ram:NetPriceProductTradePrice></ram:SpecifiedLineTradeAgreement>
			<ram:SpecifiedLineTradeDelivery><ram:BilledQuantity unitCode="C62">2.00</ram:BilledQuantity></ram:SpecifiedLineTradeDelivery>
			<ram:SpecifiedLineTradeSettlement>
				<ram:ApplicableTradeTax><ram:TypeCode>VAT</ram:TypeCode><ram:RateApplicablePercent>19.00</ram:RateApplicablePercent></ram:ApplicableTradeTax>
				<ram:SpecifiedTradeSettlementLineMonetarySummation><ram:LineTotalAmount>300.00</ram:LineTotalAmount></ram:SpecifiedTradeSettlementLineMonetarySummation>
			</ram:SpecifiedLineTradeSettlement>
		</ram:IncludedSupplyChainTradeLineItem>
	</rsm:SupplyChainTradeTransaction>
</rsm:CrossIndustryInvoice>`)

	parsed, err := ParseCII(xmlData)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrParseFailed)
	assert.Contains(t, err.Error(), "does not match declared gross total")
	assert.Nil(t, parsed)
}

func TestParseUBL_TotalsMismatch_LineItemsDoNotMatchDeclaredSubtotal(t *testing.T) {
	xmlData := []byte(`<?xml version="1.0"?>
<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
         xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2"
         xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2">
  <cbc:ID>XR-MISMATCH-LINES-001</cbc:ID>
  <cbc:IssueDate>2024-01-20</cbc:IssueDate>
  <cbc:DocumentCurrencyCode>EUR</cbc:DocumentCurrencyCode>
  <cac:AccountingSupplierParty><cac:Party><cac:PartyName><cbc:Name>Lieferant GmbH</cbc:Name></cac:PartyName></cac:Party></cac:AccountingSupplierParty>
  <cac:TaxTotal><cbc:TaxAmount currencyID="EUR">95.00</cbc:TaxAmount></cac:TaxTotal>
  <cac:LegalMonetaryTotal>
    <cbc:LineExtensionAmount currencyID="EUR">500.00</cbc:LineExtensionAmount>
    <cbc:TaxExclusiveAmount currencyID="EUR">500.00</cbc:TaxExclusiveAmount>
    <cbc:TaxInclusiveAmount currencyID="EUR">595.00</cbc:TaxInclusiveAmount>
  </cac:LegalMonetaryTotal>
  <cac:InvoiceLine>
    <cbc:ID>1</cbc:ID>
    <cbc:InvoicedQuantity unitCode="HUR">1</cbc:InvoicedQuantity>
    <cbc:LineExtensionAmount currencyID="EUR">100.00</cbc:LineExtensionAmount>
    <cac:Item><cbc:Name>Softwareentwicklung</cbc:Name><cac:ClassifiedTaxCategory><cbc:Percent>19</cbc:Percent></cac:ClassifiedTaxCategory></cac:Item>
    <cac:Price><cbc:PriceAmount currencyID="EUR">100.00</cbc:PriceAmount></cac:Price>
  </cac:InvoiceLine>
</Invoice>`)

	parsed, err := ParseUBL(xmlData)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrParseFailed)
	assert.Contains(t, err.Error(), "line items sum")
	assert.Nil(t, parsed)
}

func TestParseUBL_TotalsMismatch_SubtotalPlusTaxDoesNotMatchGrossTotal(t *testing.T) {
	xmlData := []byte(`<?xml version="1.0"?>
<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
         xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2"
         xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2">
  <cbc:ID>XR-MISMATCH-GROSS-001</cbc:ID>
  <cbc:IssueDate>2024-01-20</cbc:IssueDate>
  <cbc:DocumentCurrencyCode>EUR</cbc:DocumentCurrencyCode>
  <cac:AccountingSupplierParty><cac:Party><cac:PartyName><cbc:Name>Lieferant GmbH</cbc:Name></cac:PartyName></cac:Party></cac:AccountingSupplierParty>
  <cac:TaxTotal><cbc:TaxAmount currencyID="EUR">95.00</cbc:TaxAmount></cac:TaxTotal>
  <cac:LegalMonetaryTotal>
    <cbc:LineExtensionAmount currencyID="EUR">500.00</cbc:LineExtensionAmount>
    <cbc:TaxExclusiveAmount currencyID="EUR">500.00</cbc:TaxExclusiveAmount>
    <cbc:TaxInclusiveAmount currencyID="EUR">1234.00</cbc:TaxInclusiveAmount>
  </cac:LegalMonetaryTotal>
  <cac:InvoiceLine>
    <cbc:ID>1</cbc:ID>
    <cbc:InvoicedQuantity unitCode="HUR">5</cbc:InvoicedQuantity>
    <cbc:LineExtensionAmount currencyID="EUR">500.00</cbc:LineExtensionAmount>
    <cac:Item><cbc:Name>Softwareentwicklung</cbc:Name><cac:ClassifiedTaxCategory><cbc:Percent>19</cbc:Percent></cac:ClassifiedTaxCategory></cac:Item>
    <cac:Price><cbc:PriceAmount currencyID="EUR">100.00</cbc:PriceAmount></cac:Price>
  </cac:InvoiceLine>
</Invoice>`)

	parsed, err := ParseUBL(xmlData)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrParseFailed)
	assert.Contains(t, err.Error(), "does not match declared gross total")
	assert.Nil(t, parsed)
}

// ============================================================================
// Unknown currency / tax category — documented behaviour, not a rejection.
//
// lean: the inbound parser accepts any DocumentCurrencyCode/Percent verbatim
// without checking it against the ISO 4217 / UNCL5305 whitelists that A5
// (feat-einvoice-codelist-validation) built for the OUTBOUND generator. A
// foreign sender's currency code is not this product's to reject at parse
// time — Service.Import persists it as received, and a human reviews incoming
// invoices before they post. Add an inbound whitelist check the day incoming
// invoices post automatically without review.
// ============================================================================

func TestParseUBL_UnknownCurrencyCode_AcceptedVerbatim(t *testing.T) {
	xmlData := []byte(`<?xml version="1.0"?>
<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
         xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2"
         xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2">
  <cbc:ID>XR-CURRENCY-001</cbc:ID>
  <cbc:IssueDate>2024-01-20</cbc:IssueDate>
  <cbc:DocumentCurrencyCode>ZORKMID</cbc:DocumentCurrencyCode>
  <cac:AccountingSupplierParty><cac:Party><cac:PartyName><cbc:Name>Lieferant GmbH</cbc:Name></cac:PartyName></cac:Party></cac:AccountingSupplierParty>
  <cac:TaxTotal><cbc:TaxAmount currencyID="ZORKMID">19.00</cbc:TaxAmount></cac:TaxTotal>
  <cac:LegalMonetaryTotal>
    <cbc:LineExtensionAmount currencyID="ZORKMID">100.00</cbc:LineExtensionAmount>
    <cbc:TaxExclusiveAmount currencyID="ZORKMID">100.00</cbc:TaxExclusiveAmount>
    <cbc:TaxInclusiveAmount currencyID="ZORKMID">119.00</cbc:TaxInclusiveAmount>
  </cac:LegalMonetaryTotal>
  <cac:InvoiceLine>
    <cbc:ID>1</cbc:ID>
    <cbc:InvoicedQuantity unitCode="HUR">1</cbc:InvoicedQuantity>
    <cbc:LineExtensionAmount currencyID="ZORKMID">100.00</cbc:LineExtensionAmount>
    <cac:Item><cbc:Name>Softwareentwicklung</cbc:Name><cac:ClassifiedTaxCategory><cbc:Percent>19</cbc:Percent></cac:ClassifiedTaxCategory></cac:Item>
    <cac:Price><cbc:PriceAmount currencyID="ZORKMID">100.00</cbc:PriceAmount></cac:Price>
  </cac:InvoiceLine>
</Invoice>`)

	parsed, err := ParseUBL(xmlData)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	assert.Equal(t, "ZORKMID", parsed.Currency, "unknown currency codes pass through unchecked (documented decision, see lean comment above)")
}

func TestParseUBL_UnknownTaxCategoryID_NotExtractedOrChecked(t *testing.T) {
	// ublTaxCategory only maps Percent (see parser.go) — the category ID
	// (S/Z/E/AE/...) that EN 16931 uses to select the exemption-reason rule
	// family is not read by the inbound parser at all. An invalid or unknown
	// category ID like "ZZ-NOT-A-CATEGORY" therefore has no effect on parsing:
	// only the numeric rate matters here.
	xmlData := []byte(`<?xml version="1.0"?>
<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
         xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2"
         xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2">
  <cbc:ID>XR-CATEGORY-001</cbc:ID>
  <cbc:IssueDate>2024-01-20</cbc:IssueDate>
  <cbc:DocumentCurrencyCode>EUR</cbc:DocumentCurrencyCode>
  <cac:AccountingSupplierParty><cac:Party><cac:PartyName><cbc:Name>Lieferant GmbH</cbc:Name></cac:PartyName></cac:Party></cac:AccountingSupplierParty>
  <cac:TaxTotal><cbc:TaxAmount currencyID="EUR">0.00</cbc:TaxAmount></cac:TaxTotal>
  <cac:LegalMonetaryTotal>
    <cbc:LineExtensionAmount currencyID="EUR">100.00</cbc:LineExtensionAmount>
    <cbc:TaxExclusiveAmount currencyID="EUR">100.00</cbc:TaxExclusiveAmount>
    <cbc:TaxInclusiveAmount currencyID="EUR">100.00</cbc:TaxInclusiveAmount>
  </cac:LegalMonetaryTotal>
  <cac:InvoiceLine>
    <cbc:ID>1</cbc:ID>
    <cbc:InvoicedQuantity unitCode="HUR">1</cbc:InvoicedQuantity>
    <cbc:LineExtensionAmount currencyID="EUR">100.00</cbc:LineExtensionAmount>
    <cac:Item><cbc:Name>Steuerfreie Leistung</cbc:Name><cac:ClassifiedTaxCategory><cbc:ID>ZZ-NOT-A-CATEGORY</cbc:ID><cbc:Percent>0</cbc:Percent></cac:ClassifiedTaxCategory></cac:Item>
    <cac:Price><cbc:PriceAmount currencyID="EUR">100.00</cbc:PriceAmount></cac:Price>
  </cac:InvoiceLine>
</Invoice>`)

	parsed, err := ParseUBL(xmlData)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	require.Len(t, parsed.LineItems, 1)
	assert.Equal(t, "0.00", parsed.LineItems[0].TaxRate.StringFixed(2))
}

