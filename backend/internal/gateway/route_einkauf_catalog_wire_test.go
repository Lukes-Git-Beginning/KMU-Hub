package gateway

// Wire shape of the Einkauf catalog as the desktop reads it: the list under `catalog_items`
// and single items under `catalog_item`, next to the proto names for older callers. A real
// gRPC server on loopback plays the einkauf service.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	einkaufv1 "github.com/kmuhub/kmuhub/proto/einkauf/v1"
)

type catalogStub struct {
	einkaufv1.UnimplementedEinkaufServiceServer
	items []*einkaufv1.CatalogItem
}

func (s *catalogStub) ListCatalogItems(context.Context, *einkaufv1.ListCatalogItemsRequest) (*einkaufv1.ListCatalogItemsResponse, error) {
	return &einkaufv1.ListCatalogItemsResponse{Items: s.items, Total: int32(len(s.items))}, nil
}

func (s *catalogStub) GetCatalogItem(context.Context, *einkaufv1.GetCatalogItemRequest) (*einkaufv1.CatalogItemResponse, error) {
	return &einkaufv1.CatalogItemResponse{Item: &einkaufv1.CatalogItem{Id: "ci", Name: "Schraube"}}, nil
}

func (s *catalogStub) CreateCatalogItem(context.Context, *einkaufv1.CreateCatalogItemRequest) (*einkaufv1.CatalogItemResponse, error) {
	return &einkaufv1.CatalogItemResponse{Item: &einkaufv1.CatalogItem{Id: "ci", Name: "Schraube"}}, nil
}

func (s *catalogStub) UpdateCatalogItem(context.Context, *einkaufv1.UpdateCatalogItemRequest) (*einkaufv1.CatalogItemResponse, error) {
	return &einkaufv1.CatalogItemResponse{Item: &einkaufv1.CatalogItem{Id: "ci", Name: "Mutter"}}, nil
}

func newCatalogRoutes(t *testing.T, items ...*einkaufv1.CatalogItem) *EinkaufRoutes {
	t.Helper()
	stub := &catalogStub{items: items}
	return newEinkaufRoutes(registryFor(t, "einkauf", func(s *grpc.Server) { einkaufv1.RegisterEinkaufServiceServer(s, stub) }))
}

func TestCatalogWire_ListIsUnderCatalogItems(t *testing.T) {
	routes := newCatalogRoutes(t, &einkaufv1.CatalogItem{Id: "ci", Name: "Schraube", Price: "1.20"})

	rec := httptest.NewRecorder()
	routes.HandleListCatalogItems(rec, taxModeRequest(http.MethodGet, ""))

	assertStatus(t, rec, http.StatusOK)
	body := decodeBody(t, rec)
	for _, key := range []string{"catalog_items", "items"} {
		rows, ok := body[key].([]any)
		require.True(t, ok, "%s in %s", key, rec.Body.String())
		require.Len(t, rows, 1)
		assert.Equal(t, "Schraube", rows[0].(map[string]any)["name"])
	}
	assert.EqualValues(t, 1, body["total"])
}

func TestCatalogWire_EmptyListIsEmptyArrays(t *testing.T) {
	routes := newCatalogRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleListCatalogItems(rec, taxModeRequest(http.MethodGet, ""))

	assertStatus(t, rec, http.StatusOK)
	body := decodeBody(t, rec)
	assert.Equal(t, []any{}, body["catalog_items"], rec.Body.String())
	assert.Equal(t, []any{}, body["items"], rec.Body.String())
	assert.EqualValues(t, 0, body["total"])
}

func TestCatalogWire_SingleItemIsUnderCatalogItem(t *testing.T) {
	routes := newCatalogRoutes(t)
	body := `{"supplier_id":"11111111-1111-1111-1111-111111111111","name":"Schraube","price":"1.20"}`

	cases := map[string]struct {
		call   func(http.ResponseWriter, *http.Request)
		method string
		status int
		name   string
	}{
		"get":    {routes.HandleGetCatalogItem, http.MethodGet, http.StatusOK, "Schraube"},
		"create": {routes.HandleCreateCatalogItem, http.MethodPost, http.StatusCreated, "Schraube"},
		"update": {routes.HandleUpdateCatalogItem, http.MethodPatch, http.StatusOK, "Mutter"},
	}
	for label, c := range cases {
		t.Run(label, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c.call(rec, taxModeRequest(c.method, strings.TrimSpace(body)))

			assertStatus(t, rec, c.status)
			got := decodeBody(t, rec)
			for _, key := range []string{"catalog_item", "item"} {
				item, ok := got[key].(map[string]any)
				require.True(t, ok, "%s in %s", key, rec.Body.String())
				assert.Equal(t, c.name, item["name"])
			}
		})
	}
}
