package gateway

// Optional item fields of the Inventar module as the desktop sends them: category, price,
// currency, batch and serial numbers reach the inventar service on create, and all but the
// serial numbers on update. They come back in the response. A real gRPC server on loopback
// plays the inventar service.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	inventarv1 "github.com/kmuhub/kmuhub/proto/inventar/v1"
)

type inventarFieldsStub struct {
	inventarv1.UnimplementedInventarServiceServer
	create *inventarv1.CreateItemRequest
	update *inventarv1.UpdateItemRequest
}

func (s *inventarFieldsStub) CreateItem(_ context.Context, r *inventarv1.CreateItemRequest) (*inventarv1.ItemResponse, error) {
	s.create = r
	return &inventarv1.ItemResponse{Item: &inventarv1.Item{
		Id: "i1", Name: r.GetName(), Category: r.Category, Price: r.Price, Currency: r.Currency,
		BatchNumber: r.BatchNumber, SerialNumbers: r.GetSerialNumbers(),
	}}, nil
}

func (s *inventarFieldsStub) UpdateItem(_ context.Context, r *inventarv1.UpdateItemRequest) (*inventarv1.ItemResponse, error) {
	s.update = r
	return &inventarv1.ItemResponse{Item: &inventarv1.Item{Id: "i1", Category: r.Category, Currency: r.Currency}}, nil
}

func newInventarFieldsRoutes(t *testing.T) (*InventarRoutes, *inventarFieldsStub) {
	t.Helper()
	stub := &inventarFieldsStub{}
	return NewInventarRoutes(registryFor(t, "inventar", func(s *grpc.Server) { inventarv1.RegisterInventarServiceServer(s, stub) }), nil), stub
}

func TestInventarFields_CreateForwardsCategoryCurrencyBatchAndSerials(t *testing.T) {
	routes, stub := newInventarFieldsRoutes(t)
	rec := httptest.NewRecorder()
	routes.HandleCreateItem(rec, taxModeRequest(http.MethodPost, `{"name":"Bohrer","sku":"B-1","min_quantity":2,"unit":"Stk",`+
		`"category":"Werkzeug","price":19.9,"currency":"CHF","batch_number":"L-7","serial_numbers":["a1","b2"]}`))

	assertStatus(t, rec, http.StatusCreated)
	require.NotNil(t, stub.create)
	assert.Equal(t, "Werkzeug", stub.create.GetCategory())
	assert.Equal(t, "CHF", stub.create.GetCurrency())
	assert.Equal(t, "L-7", stub.create.GetBatchNumber())
	assert.Equal(t, 19.9, stub.create.GetPrice())
	assert.Equal(t, []string{"a1", "b2"}, stub.create.GetSerialNumbers())

	item := decodeBody(t, rec)["item"].(map[string]any)
	assert.Equal(t, "Werkzeug", item["category"], rec.Body.String())
	assert.Equal(t, "CHF", item["currency"], rec.Body.String())
	assert.Equal(t, []any{"a1", "b2"}, item["serial_numbers"], rec.Body.String())
}

func TestInventarFields_UpdateForwardsFieldsAndLeavesOmittedOnesUnset(t *testing.T) {
	routes, stub := newInventarFieldsRoutes(t)
	rec := httptest.NewRecorder()
	routes.HandleUpdateItem(rec, taxModeRequest(http.MethodPatch, `{"category":"Zubehör","currency":"EUR","batch_number":"L-8"}`))

	assertStatus(t, rec, http.StatusOK)
	require.NotNil(t, stub.update)
	assert.Equal(t, proto.String("Zubehör"), stub.update.Category)
	assert.Equal(t, proto.String("EUR"), stub.update.Currency)
	assert.Equal(t, proto.String("L-8"), stub.update.BatchNumber)
	assert.Nil(t, stub.update.Price, "an omitted key stays unset so the service keeps the stored value")
	assert.Nil(t, stub.update.Name)
}

func TestInventarFields_RejectsUnknownCurrencyAndNegativePrice(t *testing.T) {
	routes, stub := newInventarFieldsRoutes(t)
	for _, body := range []string{`{"name":"x","sku":"s","currency":"EURO"}`, `{"name":"x","sku":"s","price":-1}`} {
		rec := httptest.NewRecorder()
		routes.HandleCreateItem(rec, taxModeRequest(http.MethodPost, body))
		assertStatus(t, rec, http.StatusBadRequest)
	}
	assert.Nil(t, stub.create, "invalid input never reaches the service")
}
