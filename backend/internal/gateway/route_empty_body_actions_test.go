package gateway

// Action buttons of the desktop client that send no request body at all: bank "confirm suggestion"
// and inventory "book differences". decodeAndValidate reads a missing body as an empty object.
// A real gRPC server on loopback plays the service and records what arrives.

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
	inventarv1 "github.com/kmuhub/kmuhub/proto/inventar/v1"
)

type emptyBodyBizStub struct {
	bizv1.UnimplementedFinanceServiceServer
	reconciled *bizv1.ReconcileBankTransactionRequest
}

func (s *emptyBodyBizStub) ReconcileBankTransaction(_ context.Context, r *bizv1.ReconcileBankTransactionRequest) (*bizv1.ReconcileBankTransactionResponse, error) {
	s.reconciled = r
	return &bizv1.ReconcileBankTransactionResponse{Transaction: &bizv1.BankTransaction{Id: r.GetId()}}, nil
}

type emptyBodyInventarStub struct {
	inventarv1.UnimplementedInventarServiceServer
	booked *inventarv1.BookInventurDifferencesRequest
}

func (s *emptyBodyInventarStub) BookInventurDifferences(_ context.Context, r *inventarv1.BookInventurDifferencesRequest) (*inventarv1.InventurSessionResponse, error) {
	s.booked = r
	return &inventarv1.InventurSessionResponse{}, nil
}

func registryFor(t *testing.T, name string, register func(*grpc.Server)) *ServiceRegistry {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer()
	register(srv)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register(name, lis.Addr().String())
	t.Cleanup(reg.Close)
	return reg
}

func TestEmptyBodyActions_BankMatchWithoutBodyConfirmsSuggestion(t *testing.T) {
	stub := &emptyBodyBizStub{}
	routes := NewBizRoutes(registryFor(t, "biz", func(s *grpc.Server) { bizv1.RegisterFinanceServiceServer(s, stub) }))

	rec := httptest.NewRecorder()
	routes.HandleMatchBankTransaction(rec, noBodyRequest(http.MethodPost, "/x", testDunningID))

	assertStatus(t, rec, http.StatusOK)
	require.NotNil(t, stub.reconciled)
	assert.Equal(t, testDunningID, stub.reconciled.GetId())
	assert.Empty(t, stub.reconciled.GetInvoiceId())
	assert.Empty(t, stub.reconciled.GetInvoiceNumber())
}

func TestEmptyBodyActions_InventurBookWithoutBody(t *testing.T) {
	stub := &emptyBodyInventarStub{}
	routes := NewInventarRoutes(registryFor(t, "inventar", func(s *grpc.Server) { inventarv1.RegisterInventarServiceServer(s, stub) }), nil)
	const id = "550e8400-e29b-41d4-a716-446655440000"

	rec := httptest.NewRecorder()
	routes.HandleBookInventurDifferences(rec, noBodyRequest(http.MethodPost, "/x", id))

	assertStatus(t, rec, http.StatusOK)
	require.NotNil(t, stub.booked)
	assert.Equal(t, id, stub.booked.GetSessionId())
	assert.Nil(t, stub.booked.BookedBy)
}

// A missing body is an empty object, not a free pass: handlers with required fields still say 400.
func TestEmptyBodyActions_RequiredFieldsStillFailWithoutBody(t *testing.T) {
	const id = "550e8400-e29b-41d4-a716-446655440000"
	inv := NewInventarRoutes(registryWithService("inventar"), nil)
	biz := NewBizRoutes(registryWithService("biz"))

	cases := map[string]func(http.ResponseWriter, *http.Request){
		"inventur count (item_id)": inv.HandleUpsertInventurCount,
		"datev export (dates)":     biz.HandleExportDATEV,
		"create expense":           biz.HandleCreateExpense,
	}
	for name, handler := range cases {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler(rec, noBodyRequest(http.MethodPost, "/x", id))
			assertStatus(t, rec, http.StatusBadRequest)
		})
	}
}
