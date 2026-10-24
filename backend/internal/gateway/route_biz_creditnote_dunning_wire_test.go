package gateway

// Wire shape of credit notes, dunnings and the dunning config as the desktop hooks read them:
// single entities under their key, statuses as strings (the panels index a config table with
// them), and a credit note create without a customer. A real gRPC server on loopback plays the
// biz service.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

type creditDunningStub struct {
	bizv1.UnimplementedFinanceServiceServer
	created *bizv1.CreateCreditNoteRequest
}

func (s *creditDunningStub) GetCreditNote(context.Context, *bizv1.GetCreditNoteRequest) (*bizv1.GetCreditNoteResponse, error) {
	return &bizv1.GetCreditNoteResponse{CreditNote: &bizv1.CreditNote{Id: "cn", Status: bizv1.CreditNoteStatus_CREDIT_DRAFT}}, nil
}

func (s *creditDunningStub) ListCreditNotes(context.Context, *bizv1.ListCreditNotesRequest) (*bizv1.ListCreditNotesResponse, error) {
	return &bizv1.ListCreditNotesResponse{CreditNotes: []*bizv1.CreditNote{{Id: "cn", Status: bizv1.CreditNoteStatus_CREDIT_SENT}}, Total: 1}, nil
}

func (s *creditDunningStub) CreateCreditNote(_ context.Context, r *bizv1.CreateCreditNoteRequest) (*bizv1.CreateCreditNoteResponse, error) {
	s.created = r
	return &bizv1.CreateCreditNoteResponse{CreditNote: &bizv1.CreditNote{Id: "cn", Status: bizv1.CreditNoteStatus_CREDIT_DRAFT}}, nil
}

func (s *creditDunningStub) ListDunnings(context.Context, *bizv1.ListDunningsRequest) (*bizv1.ListDunningsResponse, error) {
	return &bizv1.ListDunningsResponse{Dunnings: []*bizv1.DunningRecord{{Id: "d", Level: 1, Status: bizv1.DunningStatus_DUNNING_SENT}}, Total: 1}, nil
}

func (s *creditDunningStub) GetDunningConfig(context.Context, *bizv1.GetDunningConfigRequest) (*bizv1.GetDunningConfigResponse, error) {
	return &bizv1.GetDunningConfigResponse{Config: &bizv1.DunningConfig{Level1DaysAfterDue: 14, Level1Fee: "5.00"}}, nil
}

func (s *creditDunningStub) UpdateDunningConfig(_ context.Context, r *bizv1.UpdateDunningConfigRequest) (*bizv1.UpdateDunningConfigResponse, error) {
	return &bizv1.UpdateDunningConfigResponse{Config: r.GetConfig()}, nil
}

func newCreditDunningRoutes(t *testing.T) (*BizRoutes, *creditDunningStub) {
	t.Helper()
	stub := &creditDunningStub{}
	return NewBizRoutes(registryFor(t, "biz", func(s *grpc.Server) { bizv1.RegisterFinanceServiceServer(s, stub) })), stub
}

func TestCreditDunningWire_CreditNoteGetIsWrapped(t *testing.T) {
	routes, _ := newCreditDunningRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleGetCreditNote(rec, taxModeRequest(http.MethodGet, ""))

	assertStatus(t, rec, http.StatusOK)
	cn, ok := decodeBody(t, rec)["credit_note"].(map[string]any)
	require.True(t, ok, rec.Body.String())
	assert.Equal(t, "cn", cn["id"])
	assert.Equal(t, "draft", cn["status"])
}

func TestCreditDunningWire_DunningConfigIsWrappedOnGetAndPut(t *testing.T) {
	routes, _ := newCreditDunningRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleGetDunningConfig(rec, taxModeRequest(http.MethodGet, ""))
	assertStatus(t, rec, http.StatusOK)
	cfg, ok := decodeBody(t, rec)["config"].(map[string]any)
	require.True(t, ok, rec.Body.String())
	assert.EqualValues(t, 14, cfg["level1_days_after_due"])
	assert.Equal(t, "5.00", cfg["level1_fee"])

	rec = httptest.NewRecorder()
	routes.HandleUpdateDunningConfig(rec, taxModeRequest(http.MethodPut,
		`{"level1_days_after_due":7,"level2_days_after_level1":7,"level3_days_after_level2":7,"level1_fee":"1.00"}`))
	assertStatus(t, rec, http.StatusOK)
	cfg, ok = decodeBody(t, rec)["config"].(map[string]any)
	require.True(t, ok, rec.Body.String())
	assert.EqualValues(t, 7, cfg["level1_days_after_due"])
}

func TestCreditDunningWire_StatusesAreStrings(t *testing.T) {
	routes, _ := newCreditDunningRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleListDunnings(rec, taxModeRequest(http.MethodGet, ""))
	assertStatus(t, rec, http.StatusOK)
	d := decodeBody(t, rec)["dunnings"].([]any)[0].(map[string]any)
	assert.Equal(t, "sent", d["status"])

	rec = httptest.NewRecorder()
	routes.HandleListCreditNotes(rec, taxModeRequest(http.MethodGet, ""))
	assertStatus(t, rec, http.StatusOK)
	cn := decodeBody(t, rec)["credit_notes"].([]any)[0].(map[string]any)
	assert.Equal(t, "sent", cn["status"])
}

// The desktop dialog sends no customer: the service reads it from the original invoice.
func TestCreditDunningWire_CreditNoteCreateWithoutCustomer(t *testing.T) {
	routes, stub := newCreditDunningRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleCreateCreditNote(rec, taxModeRequest(http.MethodPost, `{
		"original_invoice_id":"11111111-1111-1111-1111-111111111111",
		"line_items":[{"position":1,"description":"x","quantity":"1","unit_price":"10.00","tax_rate":"19"}],
		"tax_mode":"standard","reason":"r","is_storno":true}`))

	assertStatus(t, rec, http.StatusCreated)
	require.NotNil(t, stub.created)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", stub.created.GetOriginalInvoiceId())
	assert.Nil(t, stub.created.GetCustomer())
}

// Without line items or a reason the request is still refused.
func TestCreditDunningWire_CreditNoteCreateStillValidates(t *testing.T) {
	routes, stub := newCreditDunningRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleCreateCreditNote(rec, taxModeRequest(http.MethodPost, strings.TrimSpace(`{"tax_mode":"standard"}`)))

	assertStatus(t, rec, http.StatusBadRequest)
	assert.Nil(t, stub.created)
}
