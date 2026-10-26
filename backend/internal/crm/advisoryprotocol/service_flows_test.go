package advisoryprotocol

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pdfText returns the page content of a maroto/gofpdf document (streams are
// written uncompressed), so tests can assert on the rendered strings.
func pdfText(t *testing.T, doc []byte) string {
	t.Helper()
	require.True(t, bytes.HasPrefix(doc, []byte("%PDF-")), "not a PDF")
	return string(doc)
}

func TestGeneratePDF_FullProtocolRendersEveryField(t *testing.T) {
	svc := NewService(newMockRepository())
	tid, cid := uuid.New(), uuid.New()
	p := fullProtocol(uuid.New(), tid, cid, uuid.New())
	liquid, pct := 12000.0, 35.5
	p.LiquidAssets = &liquid
	p.MaxLossCapacityPct = &pct
	delivered := "2026-03-20"
	p.DocumentDeliveredDate = &delivered

	doc, err := svc.GeneratePDF(context.Background(), p, "Max Mustermann", "Zentria Beratung")
	require.NoError(t, err)
	txt := pdfText(t, doc)

	for _, want := range []string{
		"Max Mustermann", "Zentria Beratung", "Erika Musterberaterin", "2026-03-15",
		"09:00", "10:00", "Aktienfonds seit 2015", "Bankkaufmann", "10 Jahre Finanzbranche",
		"4200.50 EUR", "12000.00 EUR", "35.5%", "stocks, etf", "retirement, growth",
		"4 / 7", "Nachhaltige Fonds bevorzugt", "Breit gestreutes ETF-Portfolio",
		"Passt zu Zielen und Risikoprofil", "Altersvorsorge", "risk, costs", "2026-03-20",
		"E. Musterberaterin", "FinVermV",
	} {
		assert.Contains(t, txt, want)
	}
}

func TestGeneratePDF_OptionalPartsAreOmittedWhenEmpty(t *testing.T) {
	svc := NewService(newMockRepository())
	p := &Protocol{ID: uuid.New(), RiskClass: 2, TimeFrom: "09:00"} // TimeTo empty

	doc, err := svc.GeneratePDF(context.Background(), p, "", "")
	require.NoError(t, err)
	txt := pdfText(t, doc)

	assert.Contains(t, txt, "2 / 7")
	assert.Contains(t, txt, "Nein") // ESG off
	assert.NotContains(t, txt, "Kunde:")
	assert.NotContains(t, txt, "Datum:")
	assert.NotContains(t, txt, "Uhrzeit:")
	assert.NotContains(t, txt, "Nettoeinkommen")
	assert.NotContains(t, txt, "Liquide Mittel")
	assert.NotContains(t, txt, "Verlusttrag")
	assert.NotContains(t, txt, "Aush") // delivery date + form both empty
}

func TestGeneratePDF_EsgPreferenceYesCarriesDetails(t *testing.T) {
	svc := NewService(newMockRepository())
	p := &Protocol{ID: uuid.New(), RiskClass: 5, EsgPreference: true, EsgDetails: "Artikel-8-Fonds"}

	doc, err := svc.GeneratePDF(context.Background(), p, "Kunde X", "")
	require.NoError(t, err)
	txt := pdfText(t, doc)

	assert.Contains(t, txt, "Artikel-8-Fonds")
	assert.NotContains(t, txt, "(Nein)")
}

func TestJoinStrings(t *testing.T) {
	assert.Equal(t, "", joinStrings(nil))
	assert.Equal(t, "a", joinStrings([]string{"a"}))
	assert.Equal(t, "a, b, c", joinStrings([]string{"a", "b", "c"}))
}

func TestService_ListByContact(t *testing.T) {
	svc, _, tenantID, contactID := setupService(t)
	ctx := context.Background()

	_, err := svc.ListByContact(ctx, contactID, uuid.Nil)
	assert.ErrorIs(t, err, ErrInvalidTenant)

	for range 2 {
		_, err := svc.Create(ctx, CreateInput{TenantID: tenantID, ContactID: contactID, CreatedBy: uuid.New()})
		require.NoError(t, err)
	}
	got, err := svc.ListByContact(ctx, contactID, tenantID)
	require.NoError(t, err)
	assert.Len(t, got, 2)

	other, err := svc.ListByContact(ctx, contactID, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, other)
}

func TestService_InvalidTenantIsRejectedOnEveryEntryPoint(t *testing.T) {
	svc, _, _, _ := setupService(t)
	ctx := context.Background()
	id := uuid.New()

	_, err := svc.GetByID(ctx, id, uuid.Nil)
	assert.ErrorIs(t, err, ErrInvalidTenant)
	_, err = svc.Update(ctx, id, uuid.Nil, UpdateInput{RiskClass: 3})
	assert.ErrorIs(t, err, ErrInvalidTenant)
	assert.ErrorIs(t, svc.Delete(ctx, id, uuid.Nil), ErrInvalidTenant)
	_, err = svc.HandOver(ctx, id, uuid.Nil)
	assert.ErrorIs(t, err, ErrInvalidTenant)
	_, err = svc.GetReferralReport(ctx, uuid.Nil)
	assert.ErrorIs(t, err, ErrInvalidTenant)
}

func TestService_RepositoryFailuresSurface(t *testing.T) {
	boom := errors.New("db down")
	create := func(t *testing.T) (*Service, *mockRepository, uuid.UUID, *Protocol) {
		svc, repo, tenantID, contactID := setupService(t)
		p, err := svc.Create(context.Background(), CreateInput{TenantID: tenantID, ContactID: contactID, CreatedBy: uuid.New()})
		require.NoError(t, err)
		return svc, repo, tenantID, p
	}

	t.Run("create: contact lookup error", func(t *testing.T) {
		svc, repo, tenantID, contactID := setupService(t)
		repo.contactErr = boom
		_, err := svc.Create(context.Background(), CreateInput{TenantID: tenantID, ContactID: contactID})
		assert.ErrorIs(t, err, boom)
	})
	t.Run("create: insert error", func(t *testing.T) {
		svc, repo, tenantID, contactID := setupService(t)
		repo.createErr = boom
		_, err := svc.Create(context.Background(), CreateInput{TenantID: tenantID, ContactID: contactID})
		assert.ErrorIs(t, err, boom)
	})
	t.Run("lookup errors map to not found", func(t *testing.T) {
		svc, repo, tenantID, p := create(t)
		repo.getErr = boom
		ctx := context.Background()
		_, err := svc.GetByID(ctx, p.ID, tenantID)
		assert.ErrorIs(t, err, ErrProtocolNotFound)
		assert.ErrorIs(t, svc.Delete(ctx, p.ID, tenantID), ErrProtocolNotFound)
		_, err = svc.HandOver(ctx, p.ID, tenantID)
		assert.ErrorIs(t, err, ErrProtocolNotFound)
		_, err = svc.Update(ctx, p.ID, tenantID, UpdateInput{RiskClass: 3})
		assert.ErrorIs(t, err, ErrProtocolNotFound)
	})
	t.Run("write errors are returned and leave the row alone", func(t *testing.T) {
		svc, repo, tenantID, p := create(t)
		ctx := context.Background()

		repo.updateErr = boom
		_, err := svc.Update(ctx, p.ID, tenantID, UpdateInput{Advisor: "x", RiskClass: 3})
		assert.ErrorIs(t, err, boom)

		repo.deleteErr = boom
		assert.ErrorIs(t, svc.Delete(ctx, p.ID, tenantID), boom)

		repo.handOverErr = boom
		_, err = svc.HandOver(ctx, p.ID, tenantID)
		assert.ErrorIs(t, err, boom)

		got, err := repo.GetByID(ctx, p.ID, tenantID)
		require.NoError(t, err)
		assert.Equal(t, "draft", got.Status)
		assert.Nil(t, got.HandedOverAt)
	})
	t.Run("referral report error", func(t *testing.T) {
		svc, repo, tenantID, _ := create(t)
		repo.referralErr = boom
		_, err := svc.GetReferralReport(context.Background(), tenantID)
		assert.ErrorIs(t, err, boom)
	})
}
