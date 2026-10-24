package gateway

// Wire shape of the Inventar responses as the desktop reads them: a stock of 0 stays a
// member, an uncounted position is `counted: null` (not a missing key), and empty lists are
// `[]`. A real gRPC server on loopback plays the inventar service.

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

type inventarWireStub struct {
	inventarv1.UnimplementedInventarServiceServer
}

func (inventarWireStub) ListItems(context.Context, *inventarv1.ListItemsRequest) (*inventarv1.ListItemsResponse, error) {
	return &inventarv1.ListItemsResponse{Items: []*inventarv1.Item{{Id: "i1", Name: "Schraube"}}}, nil
}

func (inventarWireStub) GetInventurSession(context.Context, *inventarv1.GetInventurSessionRequest) (*inventarv1.InventurSessionResponse, error) {
	return &inventarv1.InventurSessionResponse{Session: &inventarv1.InventurSession{
		Id: "s1", Status: "counting",
		Counts: []*inventarv1.InventurCount{
			{Id: "c1", ItemId: "i1"}, // never counted, nothing expected
			{Id: "c2", ItemId: "i2", Expected: 4, Counted: proto.Int64(0)}, // counted as zero
		},
	}}, nil
}

func (inventarWireStub) ListInventurSessions(context.Context, *inventarv1.ListInventurSessionsRequest) (*inventarv1.ListInventurSessionsResponse, error) {
	return &inventarv1.ListInventurSessionsResponse{Sessions: []*inventarv1.InventurSession{{Id: "s1", Status: "open"}}}, nil
}

func (inventarWireStub) ListMovements(context.Context, *inventarv1.ListMovementsRequest) (*inventarv1.ListMovementsResponse, error) {
	return &inventarv1.ListMovementsResponse{}, nil
}

func newInventarWireRoutes(t *testing.T) *InventarRoutes {
	t.Helper()
	return NewInventarRoutes(registryFor(t, "inventar", func(s *grpc.Server) { inventarv1.RegisterInventarServiceServer(s, inventarWireStub{}) }), nil)
}

func TestInventarWire_ItemWithZeroStockKeepsCounters(t *testing.T) {
	rec := httptest.NewRecorder()
	newInventarWireRoutes(t).HandleListItems(rec, taxModeRequest(http.MethodGet, ""))

	assertStatus(t, rec, http.StatusOK)
	body := decodeBody(t, rec)
	item := body["items"].([]any)[0].(map[string]any)
	assert.Equal(t, "0", item["quantity"], rec.Body.String())
	assert.Equal(t, "0", item["min_quantity"], rec.Body.String())
	assert.Contains(t, item, "barcode")
	assert.Nil(t, item["barcode"])
	assert.Equal(t, []any{}, item["serial_numbers"])
}

func TestInventarWire_UncountedPositionIsNullNotMissing(t *testing.T) {
	rec := httptest.NewRecorder()
	newInventarWireRoutes(t).HandleGetInventurSession(rec, taxModeRequest(http.MethodGet, ""))

	assertStatus(t, rec, http.StatusOK)
	counts := decodeBody(t, rec)["session"].(map[string]any)["counts"].([]any)
	require.Len(t, counts, 2)

	uncounted := counts[0].(map[string]any)
	assert.Contains(t, uncounted, "counted", rec.Body.String())
	assert.Nil(t, uncounted["counted"], "an uncounted position is null")
	assert.Equal(t, "0", uncounted["expected"], rec.Body.String())

	zero := counts[1].(map[string]any)
	assert.Equal(t, "0", zero["counted"], "counted as 0 is a count, not null")
	assert.Equal(t, "4", zero["expected"])
}

func TestInventarWire_EmptyListsAndTotalsAreKept(t *testing.T) {
	routes := newInventarWireRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleListMovements(rec, taxModeRequest(http.MethodGet, ""))
	assertStatus(t, rec, http.StatusOK)
	body := decodeBody(t, rec)
	assert.Equal(t, []any{}, body["movements"], rec.Body.String())
	assert.EqualValues(t, 0, body["total"])

	rec = httptest.NewRecorder()
	routes.HandleListInventurSessions(rec, taxModeRequest(http.MethodGet, ""))
	assertStatus(t, rec, http.StatusOK)
	session := decodeBody(t, rec)["sessions"].([]any)[0].(map[string]any)
	assert.Equal(t, []any{}, session["counts"], "a fresh session lists no counts as []")
}
