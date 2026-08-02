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

// roundtripLineItems is a two-rate invoice (2 x 100.00 at 19%, 3 x 50.00 at
// 7%): net 350.00, tax 38.00 + 10.50 = 48.50, gross 398.50. Two rates prove
// the tax breakdown, not just a single total, survives the round trip.
func roundtripLineItems() []models.LineItem {
	return []models.LineItem{
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
	}
}

// setupRoundtripFixture persists an EN-16931-conformant invoice and its
// company settings for a fresh tenant, and wires the two pieces
// GenerateEInvoice reads (invoice lookup, company settings). The remaining
// BizGRPCServer dependencies go unused by that RPC and stay nil.
func setupRoundtripFixture(t *testing.T, pool *pgxpool.Pool, invoiceNumber string) (context.Context, uuid.UUID, *models.Invoice, *server.BizGRPCServer, *einvoice.Service) {
	t.Helper()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "EInvoice Roundtrip "+invoiceNumber)
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	raw, err := json.Marshal(roundtripLineItems())
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
		Subtotal:        decimal.RequireFromString("350.00"),
		TotalTax:        decimal.RequireFromString("48.50"),
		GrossTotal:      decimal.RequireFromString("398.50"),
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

	ctx, tenantID, inv, bizServer, einvoiceSvc := setupRoundtripFixture(t, pool, "RE-RT-XR-0001")

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

	assertRoundtripMatches(t, inv, imported, models.IncomingInvoiceFormatXRechnungUBL)
}

// TestRoundtrip_ZUGFeRD_ThroughInboundImport is the same proof for
// format=zugferd. The RPC's output there is a full PDF, so this also
// exercises the embedded-XML extraction (ExtractXMLFromPDF) Service.Import
// runs for a "pdf" MIME type — not just the XML generators.
func TestRoundtrip_ZUGFeRD_ThroughInboundImport(t *testing.T) {
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	ctx, tenantID, inv, bizServer, einvoiceSvc := setupRoundtripFixture(t, pool, "RE-RT-ZF-0001")

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

	assertRoundtripMatches(t, inv, imported, models.IncomingInvoiceFormatZUGFeRDCII)
}

// assertRoundtripMatches compares the amounts, line items and tax breakdown
// the inbound Import produced against the outbound invoice they were
// rendered from.
func assertRoundtripMatches(t *testing.T, outbound *models.Invoice, inbound *models.IncomingInvoice, wantFormat string) {
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
	wantItems := roundtripLineItems()
	require.Len(t, gotItems, len(wantItems))
	for i, wantItem := range wantItems {
		assert.Equal(t, wantItem.Description, gotItems[i].Description)
		assert.True(t, wantItem.LineTotal.Equal(gotItems[i].LineTotal),
			"line %d total: want %s, got %s", i, wantItem.LineTotal, gotItems[i].LineTotal)
	}

	var gotTax []struct {
		TaxRate   decimal.Decimal `json:"tax_rate"`
		TaxAmount decimal.Decimal `json:"tax_amount"`
	}
	require.NoError(t, json.Unmarshal(inbound.TaxBreakdownRaw, &gotTax))
	require.Len(t, gotTax, 2)
	assert.True(t, gotTax[0].TaxRate.Equal(decimal.NewFromInt(19)))
	assert.True(t, gotTax[0].TaxAmount.Equal(decimal.RequireFromString("38.00")))
	assert.True(t, gotTax[1].TaxRate.Equal(decimal.NewFromInt(7)))
	assert.True(t, gotTax[1].TaxAmount.Equal(decimal.RequireFromString("10.50")))
}
