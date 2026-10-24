package gateway

// Partial saves of PUT /finance/settings (desktop StammdatenTab saves name/address, tax,
// bank and payment terms as separate blocks). A real gRPC server on loopback stands in for
// the biz service and, like BizGRPCServer.UpdateCompanySettings, replaces the whole row
// with whatever the gateway sends — so the test sees exactly what would reach the database.

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

type settingsStub struct {
	bizv1.UnimplementedFinanceServiceServer
	stored  *bizv1.CompanySettings
	getErr  error
	updates int
}

func (s *settingsStub) GetCompanySettings(context.Context, *bizv1.GetCompanySettingsRequest) (*bizv1.GetCompanySettingsResponse, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return &bizv1.GetCompanySettingsResponse{Settings: s.stored}, nil
}

func (s *settingsStub) UpdateCompanySettings(_ context.Context, r *bizv1.UpdateCompanySettingsRequest) (*bizv1.UpdateCompanySettingsResponse, error) {
	s.updates++
	s.stored = r.GetSettings()
	return &bizv1.UpdateCompanySettingsResponse{Settings: s.stored}, nil
}

func newSettingsRoutes(t *testing.T) (*BizRoutes, *settingsStub) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	stub := &settingsStub{}
	srv := grpc.NewServer()
	bizv1.RegisterFinanceServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("biz", lis.Addr().String())
	t.Cleanup(reg.Close)
	return NewBizRoutes(reg), stub
}

func putSettings(t *testing.T, routes *BizRoutes, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/finance/settings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withTenantID(req, testTenantID)
	rec := httptest.NewRecorder()
	routes.HandleUpdateCompanySettings(rec, req)
	return rec
}

func fullSettings() *bizv1.CompanySettings {
	return &bizv1.CompanySettings{
		Id:                       "11111111-1111-1111-1111-111111111111",
		TenantId:                 testTenantID.String(),
		Name:                     "Muster GmbH",
		Street:                   "Hauptstr. 1",
		Plz:                      "10115",
		City:                     "Berlin",
		Country:                  "DE",
		Steuernummer:             "12/345/67890",
		BankName:                 "Sparkasse",
		Iban:                     "DE89370400440532013000",
		IsKleinunternehmer:       true,
		DefaultPaymentTermsDays:  14,
		DefaultQuoteValidityDays: 30,
		Basiszinssatz:            "2.27",
	}
}

func TestHandleUpdateCompanySettings_PartialSaveKeepsOtherBlocks(t *testing.T) {
	routes, stub := newSettingsRoutes(t)
	stub.stored = fullSettings()

	// StammdatenTab bank block: only the bank fields.
	rec := putSettings(t, routes, `{"bank_name":"Volksbank","iban":"DE02120300000000202051","bic":"BYLADEM1001"}`)
	assertStatus(t, rec, http.StatusOK)

	want := fullSettings()
	want.BankName = "Volksbank"
	want.Iban = "DE02120300000000202051"
	want.Bic = "BYLADEM1001"
	assert.Equal(t, want.Name, stub.stored.Name)
	assert.Equal(t, want.Street, stub.stored.Street)
	assert.Equal(t, want.Plz, stub.stored.Plz)
	assert.Equal(t, want.City, stub.stored.City)
	assert.Equal(t, want.Steuernummer, stub.stored.Steuernummer)
	assert.Equal(t, int32(14), stub.stored.DefaultPaymentTermsDays)
	assert.Equal(t, int32(30), stub.stored.DefaultQuoteValidityDays)
	assert.Equal(t, "2.27", stub.stored.Basiszinssatz)
	assert.True(t, stub.stored.IsKleinunternehmer)
	assert.Equal(t, "Volksbank", stub.stored.BankName)
	assert.Equal(t, "DE02120300000000202051", stub.stored.Iban)
	assert.Equal(t, "BYLADEM1001", stub.stored.Bic)

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "Muster GmbH", got["name"])
	assert.Equal(t, "Volksbank", got["bank_name"])
}

func TestHandleUpdateCompanySettings_PaymentTermsOnlyKeepsEverythingElse(t *testing.T) {
	routes, stub := newSettingsRoutes(t)
	stub.stored = fullSettings()

	assertStatus(t, putSettings(t, routes, `{"default_payment_terms_days":30}`), http.StatusOK)

	assert.Equal(t, int32(30), stub.stored.DefaultPaymentTermsDays)
	assert.Equal(t, "Muster GmbH", stub.stored.Name)
	assert.Equal(t, "DE89370400440532013000", stub.stored.Iban)
	assert.Equal(t, "2.27", stub.stored.Basiszinssatz)
}

func TestHandleUpdateCompanySettings_ExplicitEmptyAndFalseStillClear(t *testing.T) {
	routes, stub := newSettingsRoutes(t)
	stub.stored = fullSettings()

	rec := putSettings(t, routes, `{"bank_name":"","iban":"","steuernummer":"","is_kleinunternehmer":false,"default_payment_terms_days":0}`)
	assertStatus(t, rec, http.StatusOK)

	assert.Empty(t, stub.stored.BankName)
	assert.Empty(t, stub.stored.Iban)
	assert.Empty(t, stub.stored.Steuernummer)
	assert.False(t, stub.stored.IsKleinunternehmer)
	assert.Equal(t, int32(0), stub.stored.DefaultPaymentTermsDays)
	assert.Equal(t, "Muster GmbH", stub.stored.Name, "absent fields stay untouched")
	assert.Equal(t, "Berlin", stub.stored.City)
}

func TestHandleUpdateCompanySettings_FirstSaveWithoutStoredRow(t *testing.T) {
	routes, stub := newSettingsRoutes(t)

	rec := putSettings(t, routes, `{"name":"Neu GmbH","city":"Köln"}`)
	assertStatus(t, rec, http.StatusOK)

	require.NotNil(t, stub.stored)
	assert.Equal(t, "Neu GmbH", stub.stored.Name)
	assert.Equal(t, "Köln", stub.stored.City)
	assert.Equal(t, testTenantID.String(), stub.stored.TenantId)
}

func TestHandleUpdateCompanySettings_InvalidFieldStillRejectedBeforeAnyWrite(t *testing.T) {
	routes, stub := newSettingsRoutes(t)
	stub.stored = fullSettings()

	assertStatus(t, putSettings(t, routes, `{"iban":"not-an-iban"}`), http.StatusBadRequest)
	assertStatus(t, putSettings(t, routes, `{"default_payment_terms_days":-1}`), http.StatusBadRequest)
	assert.Zero(t, stub.updates)
	assert.Equal(t, "DE89370400440532013000", stub.stored.Iban)
}

func TestHandleUpdateCompanySettings_LoadFailureWritesNothing(t *testing.T) {
	routes, stub := newSettingsRoutes(t)
	stub.stored = fullSettings()
	stub.getErr = status.Error(codes.Internal, "boom")

	rec := putSettings(t, routes, `{"bank_name":"X"}`)
	assertStatus(t, rec, http.StatusInternalServerError)
	assert.Zero(t, stub.updates, "a failed read must not fall through to an overwrite")
}
