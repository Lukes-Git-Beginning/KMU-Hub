package gateway

// Finance action buttons of the desktop client: they send no body (convert, escalate), expect a
// wrapped list from "detect" and download DATEV with a GET. A real gRPC server on loopback plays
// the biz service and records what arrives.

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

const testDunningID = "22222222-2222-2222-2222-222222222222"

type financeActionsStub struct {
	bizv1.UnimplementedFinanceServiceServer
	converted    *bizv1.ConvertQuoteToInvoiceRequest
	detected     []*bizv1.DunningRecord // what CreateDunning answers; empty -> NotFound like the service
	escalated    *bizv1.EscalateDunningRequest
	exportedFrom string
	exportedTo   string
}

func (s *financeActionsStub) ConvertQuoteToInvoice(_ context.Context, r *bizv1.ConvertQuoteToInvoiceRequest) (*bizv1.ConvertQuoteToInvoiceResponse, error) {
	s.converted = r
	return &bizv1.ConvertQuoteToInvoiceResponse{Invoice: &bizv1.Invoice{Id: "inv"}}, nil
}

func (s *financeActionsStub) CreateDunning(context.Context, *bizv1.CreateDunningRequest) (*bizv1.CreateDunningResponse, error) {
	if len(s.detected) == 0 {
		return nil, status.Error(codes.NotFound, "no overdue invoices found for dunning creation")
	}
	return &bizv1.CreateDunningResponse{Dunning: s.detected[0], Dunnings: s.detected}, nil
}

func (s *financeActionsStub) EscalateDunning(_ context.Context, r *bizv1.EscalateDunningRequest) (*bizv1.EscalateDunningResponse, error) {
	s.escalated = r
	return &bizv1.EscalateDunningResponse{Dunning: &bizv1.DunningRecord{Id: "d2", Level: 2}}, nil
}

func (s *financeActionsStub) ExportDATEV(_ context.Context, r *bizv1.ExportDATEVRequest) (*bizv1.ExportDATEVResponse, error) {
	s.exportedFrom, s.exportedTo = r.GetStartDate(), r.GetEndDate()
	return &bizv1.ExportDATEVResponse{CsvData: []byte("EXTF;700\n"), Filename: "EXTF_Buchungsstapel.csv"}, nil
}

func newFinanceActionsRoutes(t *testing.T) (*BizRoutes, *financeActionsStub) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	stub := &financeActionsStub{}
	srv := grpc.NewServer()
	bizv1.RegisterFinanceServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("biz", lis.Addr().String())
	t.Cleanup(reg.Close)
	return NewBizRoutes(reg), stub
}

// noBodyRequest is a request that carries no body at all, like the client's action buttons.
func noBodyRequest(method, target, idParam string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	req = withTenantID(req, testTenantID)
	if idParam != "" {
		req = withChiURLParam(req, "id", idParam)
	}
	return req
}

func TestFinanceActions_ConvertQuoteWithoutBody(t *testing.T) {
	routes, stub := newFinanceActionsRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleConvertQuoteToInvoice(rec, noBodyRequest(http.MethodPost, "/x", testQuoteID))

	assertStatus(t, rec, http.StatusCreated)
	require.NotNil(t, stub.converted)
	assert.Equal(t, testQuoteID, stub.converted.GetId())
}

func TestFinanceActions_ConvertQuoteStillValidatesGivenFields(t *testing.T) {
	routes, _ := newFinanceActionsRoutes(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/x", jsonBody(t, map[string]any{"due_date": "not-a-date"}))
	req = withChiURLParam(withTenantID(req, testTenantID), "id", testQuoteID)
	routes.HandleConvertQuoteToInvoice(rec, req)
	assertValidationError(t, rec, "due_date")
}

func TestFinanceActions_DetectReturnsWrappedList(t *testing.T) {
	routes, stub := newFinanceActionsRoutes(t)
	stub.detected = []*bizv1.DunningRecord{{Id: "d1", Level: 1}, {Id: "d2", Level: 1}}

	rec := httptest.NewRecorder()
	routes.HandleCreateDunning(rec, noBodyRequest(http.MethodPost, "/x", ""))

	assertStatus(t, rec, http.StatusOK)
	got := decodeBody(t, rec)
	items, ok := got["dunnings"].([]any)
	require.True(t, ok, rec.Body.String())
	assert.Len(t, items, 2)
	assert.EqualValues(t, 2, got["total"])
}

func TestFinanceActions_DetectNothingDueIsEmptyList(t *testing.T) {
	routes, _ := newFinanceActionsRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleCreateDunning(rec, noBodyRequest(http.MethodPost, "/x", ""))

	assertStatus(t, rec, http.StatusOK)
	got := decodeBody(t, rec)
	assert.Equal(t, []any{}, got["dunnings"])
	assert.EqualValues(t, 0, got["total"])
}

func TestFinanceActions_EscalateWithoutBodyUsesPathID(t *testing.T) {
	routes, stub := newFinanceActionsRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleEscalateDunning(rec, noBodyRequest(http.MethodPost, "/x", testDunningID))

	assertStatus(t, rec, http.StatusOK)
	require.NotNil(t, stub.escalated)
	assert.Equal(t, testDunningID, stub.escalated.GetDunningId())
	assert.Empty(t, stub.escalated.GetInvoiceId())
}

func TestFinanceActions_DatevGetUsesDesktopQueryNames(t *testing.T) {
	routes, stub := newFinanceActionsRoutes(t)

	rec := httptest.NewRecorder()
	req := noBodyRequest(http.MethodGet, "/x?date_from=2026-01-01&date_to=2026-03-31", "")
	routes.HandleExportDATEVGet(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, "2026-01-01", stub.exportedFrom)
	assert.Equal(t, "2026-03-31", stub.exportedTo)
	assert.Equal(t, "text/csv; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Equal(t, "EXTF;700\n", rec.Body.String())
}

func TestFinanceActions_DatevGetRejectsMissingOrBadDates(t *testing.T) {
	routes, _ := newFinanceActionsRoutes(t)

	for name, target := range map[string]string{
		"missing":   "/x",
		"bad range": "/x?date_from=yesterday&date_to=2026-03-31",
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			routes.HandleExportDATEVGet(rec, noBodyRequest(http.MethodGet, target, ""))
			assertStatus(t, rec, http.StatusBadRequest)
		})
	}
}
