// This file is an external test package on purpose: it drives the outbound
// e-invoice RPC in internal/server and feeds its output back through the
// inbound Service.Import. internal/server already imports einvoice, so an
// in-package test importing server would close an import cycle — the same
// reason pdf_extract_test.go lives here rather than in package einvoice.
package einvoice_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/biz/einvoice"
	"github.com/kmuhub/kmuhub/internal/biz/invoice"
	"github.com/kmuhub/kmuhub/internal/biz/quote"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/testutil"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// roundtripCase is one invoice fixture plus the amounts the generated document
// has to carry. The expectations are written out as literals, never recomputed
// from the items with the same arithmetic the code under test uses — a test that
// reruns the implementation proves nothing about the implementation.
type roundtripCase struct {
	items    []models.LineItem
	lineNets []string // BT-131 per line, as the document must write them
	subtotal string   // BT-106 = sum of lineNets, exactly (BR-CO-10)
	totalTax string
	gross    string
	taxByRate [][2]string // {rate, tax amount} per BG-23 group, in first-seen order
}

// smoothRoundtripCase is a two-rate invoice on whole amounts (2 x 100.00 at 19%,
// 3 x 50.00 at 7%). Two rates prove the tax breakdown, not just a single total,
// survives the round trip. No rounding is involved anywhere — that is what makes
// it the control against fractionalRoundtripCase.
func smoothRoundtripCase() roundtripCase {
	return roundtripCase{
		items: []models.LineItem{
			{
				Position: 1, Description: "Beratung & Analyse",
				Quantity: decimal.NewFromInt(2), UnitPrice: decimal.RequireFromString("100.00"),
				TaxRate: decimal.NewFromInt(19), LineTotal: decimal.RequireFromString("200.00"),
			},
			{
				Position: 2, Description: "Handbuch",
				Quantity: decimal.NewFromInt(3), UnitPrice: decimal.RequireFromString("50.00"),
				TaxRate: decimal.NewFromInt(7), LineTotal: decimal.RequireFromString("150.00"),
			},
		},
		lineNets:  []string{"200.00", "150.00"},
		subtotal:  "350.00",
		totalTax:  "48.50",
		gross:     "398.50",
		taxByRate: [][2]string{{"19", "38.00"}, {"7", "10.50"}},
	}
}

// fractionalRoundtripCase is the case that actually exercises the rounding order.
// It is built so that BOTH EN 16931 sum rules break if either rounding step is
// dropped — a fixture that only breaks one of them lets the other regress unseen.
//
//	1.5 x 33.33 = 49.995 -> 50.00 | 19 %
//	0.5 x 16.58 =  8.29  ->  8.29 | 19 %
//	0.5 x 16.58 =  8.29  ->  8.29 | 19 %
//	1.5 x 11.11 = 16.665 -> 16.67 |  7 %
//	0.5 x  9.99 =  4.995 ->  5.00 |  7 %
//
// BR-CO-10 (no tolerance): rounded-then-summed is 88.25, summed-then-rounded is
// 88.24. The document must carry 88.25, because BT-106 is compared against the five
// line amounts as written.
//
// BR-CO-17: the 19 % group net is 66.58, and 66.58 x 19 % = 12.6502 -> 12.65.
// Accumulating per-line taxes instead gives 9.50 + 1.58 + 1.58 = 12.66, because
// 8.29 x 19 % = 1.5751 rounds up twice. One cent, on every invoice with an odd
// unit price — and it is the cent an XRechnung validator recomputes.
//
// LineTotal is deliberately left unset: the generator then computes the net from
// quantity x unit price, which is the path a stored sub-cent line total takes too.
func fractionalRoundtripCase() roundtripCase {
	dec := decimal.RequireFromString
	return roundtripCase{
		items: []models.LineItem{
			{
				Position: 1, Description: "Projektstunden Senior",
				Quantity: dec("1.5"), UnitPrice: dec("33.33"), TaxRate: decimal.NewFromInt(19),
			},
			{
				Position: 2, Description: "Materialpauschale Nord",
				Quantity: dec("0.5"), UnitPrice: dec("16.58"), TaxRate: decimal.NewFromInt(19),
			},
			{
				Position: 3, Description: "Materialpauschale Sued",
				Quantity: dec("0.5"), UnitPrice: dec("16.58"), TaxRate: decimal.NewFromInt(19),
			},
			{
				Position: 4, Description: "Schulungsunterlagen",
				Quantity: dec("1.5"), UnitPrice: dec("11.11"), TaxRate: decimal.NewFromInt(7),
			},
			{
				Position: 5, Description: "Handbuch Nachdruck",
				Quantity: dec("0.5"), UnitPrice: dec("9.99"), TaxRate: decimal.NewFromInt(7),
			},
		},
		lineNets:  []string{"50.00", "8.29", "8.29", "16.67", "5.00"},
		subtotal:  "88.25",
		totalTax:  "14.17",
		gross:     "102.42",
		taxByRate: [][2]string{{"19", "12.65"}, {"7", "1.52"}},
	}
}

// setupRoundtripFixture persists an EN-16931-conformant invoice and its
// company settings for a fresh tenant, and wires the two pieces
// GenerateEInvoice reads (invoice lookup, company settings). The remaining
// BizGRPCServer dependencies go unused by that RPC and stay nil.
func setupRoundtripFixture(t *testing.T, pool *pgxpool.Pool, invoiceNumber string, c roundtripCase) (context.Context, uuid.UUID, *models.Invoice, *server.BizGRPCServer, *einvoice.Service) {
	t.Helper()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "EInvoice Roundtrip "+invoiceNumber)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	raw, err := json.Marshal(c.items)
	require.NoError(t, err)
	now := time.Now().UTC().Truncate(time.Second)

	inv := &models.Invoice{
		ID:              uuid.New(),
		TenantID:        tenantID,
		InvoiceNumber:   invoiceNumber,
		Status:          models.InvoiceStatusSent,
		CustomerName:    "Stadtwerke Musterstadt AöR",
		CustomerAddress: "Rathausplatz 3\n12345 Musterstadt",
		CustomerUStIDNr: "DE987654321",
		TaxMode:         models.TaxModeStandard,
		LineItems:       raw,
		Subtotal:        decimal.RequireFromString(c.subtotal),
		TotalTax:        decimal.RequireFromString(c.totalTax),
		GrossTotal:      decimal.RequireFromString(c.gross),
		Currency:        models.DefaultCurrency,
		InvoiceDate:     time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		DueDate:         time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC),
		PaymentTerms:    "Zahlbar innerhalb von 14 Tagen ohne Abzug",
		CreatedBy:       uuid.New(),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	settings := &models.CompanySettings{
		TenantID: tenantID,
		Name:     "Zentria UG",
		Street:   "Musterstrasse 1",
		PLZ:      "80331",
		City:     "München",
		Country:  "DE",
		UStIDNr:  "DE123456789",
		BankName: "Beispielbank",
		IBAN:     "DE02120300000000202051",
		BIC:      "BYLADEM1001",
	}

	invoiceRepo := invoice.NewPostgresRepository(pool)
	require.NoError(t, invoiceRepo.Create(ctx, inv))

	settingsRepo := quote.NewPostgresCompanySettingsRepo(pool)
	require.NoError(t, settingsRepo.Upsert(ctx, settings))

	invoiceService := invoice.NewService(invoiceRepo, nil, nil, nil, pool)
	bizServer := server.NewBizGRPCServer(
		nil, invoiceService, nil, nil, nil, nil, nil, nil,
		settingsRepo, nil, nil, nil, nil,
	)

	einvoiceSvc := einvoice.NewService(einvoice.NewPostgresRepository(pool))

	return ctx, tenantID, inv, bizServer, einvoiceSvc
}

// TestRoundtrip_XRechnung_ThroughInboundImport proves the outbound
// /erechnung?format=xrechnung RPC and the inbound Import agree: the UBL XML
// GenerateEInvoice renders from a persisted invoice parses back into the
// same amounts, positions and tax breakdown through the real DB-backed
// import path (not just Generate -> Parse in memory).
func TestRoundtrip_XRechnung_ThroughInboundImport(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	c := smoothRoundtripCase()
	ctx, tenantID, inv, bizServer, einvoiceSvc := setupRoundtripFixture(t, pool, "RE-RT-XR-0001", c)

	resp, err := bizServer.GenerateEInvoice(ctx, &bizv1.GenerateEInvoiceRequest{
		Id:       inv.ID.String(),
		TenantId: tenantID.String(),
		Format:   "xrechnung",
	})
	require.NoError(t, err)
	assert.Equal(t, "xrechnung_RE-RT-XR-0001.xml", resp.Filename)

	imported, err := einvoiceSvc.Import(ctx, einvoice.ImportInput{
		TenantID:  tenantID,
		CreatedBy: inv.CreatedBy,
		Filename:  resp.Filename,
		Content:   resp.Data,
		MimeType:  "application/xml",
	})
	require.NoError(t, err)

	assertRoundtripMatches(t, inv, imported, models.IncomingInvoiceFormatXRechnungUBL, c)
}

// TestRoundtrip_ZUGFeRD_ThroughInboundImport is the same proof for
// format=zugferd. The RPC's output there is a full PDF, so this also
// exercises the embedded-XML extraction (ExtractXMLFromPDF) Service.Import
// runs for a "pdf" MIME type — not just the XML generators.
func TestRoundtrip_ZUGFeRD_ThroughInboundImport(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	c := smoothRoundtripCase()
	ctx, tenantID, inv, bizServer, einvoiceSvc := setupRoundtripFixture(t, pool, "RE-RT-ZF-0001", c)

	resp, err := bizServer.GenerateEInvoice(ctx, &bizv1.GenerateEInvoiceRequest{
		Id:       inv.ID.String(),
		TenantId: tenantID.String(),
		Format:   "zugferd",
	})
	require.NoError(t, err)
	assert.Equal(t, "factur-x_RE-RT-ZF-0001.pdf", resp.Filename)

	imported, err := einvoiceSvc.Import(ctx, einvoice.ImportInput{
		TenantID:  tenantID,
		CreatedBy: inv.CreatedBy,
		Filename:  resp.Filename,
		Content:   resp.Data,
		MimeType:  "application/pdf",
	})
	require.NoError(t, err)

	assertRoundtripMatches(t, inv, imported, models.IncomingInvoiceFormatZUGFeRDCII, c)
}

// TestRoundtrip_FractionalQuantities_HoldsEN16931SumRules is the reason this file
// exists in its current shape. The smooth fixture above cannot fail on rounding —
// every one of its amounts is already a whole cent — so it proved the plumbing but
// not the arithmetic. This case puts line nets on half cents and groups several
// lines per rate, so that both "round then sum" and "sum then round" produce a
// visibly wrong document (see fractionalRoundtripCase for the two numbers).
//
// It runs through the same generate -> import path, so it covers the write side
// (buildLinesAndTaxGroups), the UBL rendering and the parser at once.
func TestRoundtrip_FractionalQuantities_HoldsEN16931SumRules(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	c := fractionalRoundtripCase()
	ctx, tenantID, inv, bizServer, einvoiceSvc := setupRoundtripFixture(t, pool, "RE-RT-XR-0002", c)

	resp, err := bizServer.GenerateEInvoice(ctx, &bizv1.GenerateEInvoiceRequest{
		Id:       inv.ID.String(),
		TenantId: tenantID.String(),
		Format:   "xrechnung",
	})
	require.NoError(t, err)

	imported, err := einvoiceSvc.Import(ctx, einvoice.ImportInput{
		TenantID:  tenantID,
		CreatedBy: inv.CreatedBy,
		Filename:  resp.Filename,
		Content:   resp.Data,
		MimeType:  "application/xml",
	})
	require.NoError(t, err)

	assertRoundtripMatches(t, inv, imported, models.IncomingInvoiceFormatXRechnungUBL, c)
}

// assertRoundtripMatches compares the amounts, line items and tax breakdown
// the inbound Import produced against the outbound invoice they were
// rendered from.
func assertRoundtripMatches(t *testing.T, outbound *models.Invoice, inbound *models.IncomingInvoice, wantFormat string, c roundtripCase) {
	t.Helper()

	assert.Equal(t, wantFormat, inbound.SourceFormat)
	assert.Equal(t, outbound.InvoiceNumber, inbound.InvoiceNumber)
	assert.True(t, outbound.Subtotal.Equal(inbound.Subtotal), "subtotal: want %s, got %s", outbound.Subtotal, inbound.Subtotal)
	assert.True(t, outbound.TotalTax.Equal(inbound.TotalTax), "total tax: want %s, got %s", outbound.TotalTax, inbound.TotalTax)
	assert.True(t, outbound.GrossTotal.Equal(inbound.GrossTotal), "gross total: want %s, got %s", outbound.GrossTotal, inbound.GrossTotal)

	var gotItems []struct {
		Description string          `json:"description"`
		LineTotal   decimal.Decimal `json:"line_total"`
	}
	require.NoError(t, json.Unmarshal(inbound.LineItems, &gotItems))
	require.Len(t, gotItems, len(c.items))
	sumOfLines := decimal.Zero
	for i, wantItem := range c.items {
		wantNet := decimal.RequireFromString(c.lineNets[i])
		assert.Equal(t, wantItem.Description, gotItems[i].Description)
		assert.True(t, wantNet.Equal(gotItems[i].LineTotal),
			"line %d net (BT-131): want %s, got %s", i+1, wantNet, gotItems[i].LineTotal)
		sumOfLines = sumOfLines.Add(gotItems[i].LineTotal)
	}

	// BR-CO-10 has no tolerance: the document total must be the sum of the line
	// amounts the document itself carries. This is the assertion that goes red when
	// the generator sums unrounded nets and rounds only on output.
	assert.True(t, sumOfLines.Equal(inbound.Subtotal),
		"BR-CO-10: sum of line nets %s != BT-106 %s", sumOfLines, inbound.Subtotal)

	var gotTax []struct {
		TaxRate    decimal.Decimal `json:"tax_rate"`
		TaxableNet decimal.Decimal `json:"taxable_net"`
		TaxAmount  decimal.Decimal `json:"tax_amount"`
	}
	require.NoError(t, json.Unmarshal(inbound.TaxBreakdownRaw, &gotTax))
	require.Len(t, gotTax, len(c.taxByRate))
	sumOfGroupTax := decimal.Zero
	hundred := decimal.NewFromInt(100)
	for i, want := range c.taxByRate {
		wantRate := decimal.RequireFromString(want[0])
		wantTax := decimal.RequireFromString(want[1])
		assert.True(t, wantRate.Equal(gotTax[i].TaxRate),
			"group %d rate (BT-119): want %s, got %s", i, wantRate, gotTax[i].TaxRate)
		assert.True(t, wantTax.Equal(gotTax[i].TaxAmount),
			"group %d tax (BT-117): want %s, got %s", i, wantTax, gotTax[i].TaxAmount)
		// BR-CO-17: BT-117 = BT-116 x rate / 100, rounded once.
		fromNet := gotTax[i].TaxableNet.Mul(gotTax[i].TaxRate).Div(hundred).Round(2)
		assert.True(t, fromNet.Equal(gotTax[i].TaxAmount),
			"BR-CO-17: group %d net %s at %s%% is %s, document says %s",
			i, gotTax[i].TaxableNet, gotTax[i].TaxRate, fromNet, gotTax[i].TaxAmount)
		sumOfGroupTax = sumOfGroupTax.Add(gotTax[i].TaxAmount)
	}

	// BR-CO-14: BT-110 is the sum of the BG-23 group tax amounts.
	assert.True(t, sumOfGroupTax.Equal(inbound.TotalTax),
		"BR-CO-14: sum of group tax %s != BT-110 %s", sumOfGroupTax, inbound.TotalTax)
	// BR-CO-15: BT-112 = BT-109 + BT-110.
	assert.True(t, inbound.Subtotal.Add(inbound.TotalTax).Equal(inbound.GrossTotal),
		"BR-CO-15: %s + %s != %s", inbound.Subtotal, inbound.TotalTax, inbound.GrossTotal)
}
