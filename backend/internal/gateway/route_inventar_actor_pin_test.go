package gateway

// The audit actor of a stock write (movement, adjustment, transfer, warning acknowledgement,
// stocktake and picking-list booking) is the caller. The desktop sends no performed_by; a body
// that names someone else must not win. A real gRPC server on loopback plays the inventar service.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	inventarv1 "github.com/kmuhub/kmuhub/proto/inventar/v1"
)

type inventarActorStub struct {
	inventarv1.UnimplementedInventarServiceServer
	adjust   *inventarv1.AdjustStockRequest
	transfer *inventarv1.TransferStockRequest
	movement *inventarv1.RecordMovementRequest
	ack      *inventarv1.AcknowledgeWarningRequest
	inventur *inventarv1.BookInventurDifferencesRequest
	picking  *inventarv1.BookPickingListRequest
}

func (s *inventarActorStub) AdjustStock(_ context.Context, r *inventarv1.AdjustStockRequest) (*inventarv1.ItemResponse, error) {
	s.adjust = r
	return &inventarv1.ItemResponse{Item: &inventarv1.Item{Id: "i1"}}, nil
}

func (s *inventarActorStub) TransferStock(_ context.Context, r *inventarv1.TransferStockRequest) (*inventarv1.TransferStockResponse, error) {
	s.transfer = r
	return &inventarv1.TransferStockResponse{}, nil
}

func (s *inventarActorStub) RecordMovement(_ context.Context, r *inventarv1.RecordMovementRequest) (*inventarv1.MovementResponse, error) {
	s.movement = r
	return &inventarv1.MovementResponse{}, nil
}

func (s *inventarActorStub) AcknowledgeWarning(_ context.Context, r *inventarv1.AcknowledgeWarningRequest) (*inventarv1.WarningResponse, error) {
	s.ack = r
	return &inventarv1.WarningResponse{}, nil
}

func (s *inventarActorStub) BookInventurDifferences(_ context.Context, r *inventarv1.BookInventurDifferencesRequest) (*inventarv1.InventurSessionResponse, error) {
	s.inventur = r
	return &inventarv1.InventurSessionResponse{}, nil
}

func (s *inventarActorStub) BookPickingList(_ context.Context, r *inventarv1.BookPickingListRequest) (*inventarv1.PickingListResponse, error) {
	s.picking = r
	return &inventarv1.PickingListResponse{}, nil
}

func TestInventarActor_StockWritesRecordTheCallerWhateverTheBodySays(t *testing.T) {
	stub := &inventarActorStub{}
	ir := NewInventarRoutes(registryFor(t, "inventar", func(s *grpc.Server) { inventarv1.RegisterInventarServiceServer(s, stub) }), nil)

	const id = "11111111-1111-1111-1111-111111111111"
	call := func(h http.HandlerFunc, body string) {
		req := withChiURLParam(withAuth(httptest.NewRequest(http.MethodPost, "/x/"+id, strings.NewReader(body)), leaveCallerID, testTenantID), "id", id)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h(rec, req)
		require.Less(t, rec.Code, 300, rec.Body.String())
	}
	forged := `"` + leaveOtherID + `"`

	for _, body := range []string{
		`{"delta":3,"reason":"x"}`,
		`{"delta":3,"reason":"x","performed_by":` + forged + `}`,
	} {
		stub.adjust = nil
		call(ir.HandleAdjustStock, body)
		require.NotNil(t, stub.adjust)
		assert.Equal(t, leaveCallerID, stub.adjust.GetPerformedBy(), "adjustment: %s", body)

		stub.movement = nil
		call(ir.HandleRecordMovement, strings.Replace(body, `"delta":3`, `"movement_type":"in","quantity":3`, 1))
		require.NotNil(t, stub.movement)
		assert.Equal(t, leaveCallerID, stub.movement.GetPerformedBy(), "movement: %s", body)

		stub.transfer = nil
		call(ir.HandleTransferStock, strings.Replace(body, `"delta":3`, `"from_item_id":"`+id+`","to_item_id":"`+id+`","quantity":3`, 1))
		require.NotNil(t, stub.transfer)
		assert.Equal(t, leaveCallerID, stub.transfer.GetPerformedBy(), "transfer: %s", body)
	}

	for _, body := range []string{`{}`, `{"booked_by":` + forged + `,"acknowledged_by":` + forged + `}`} {
		stub.ack, stub.inventur, stub.picking = nil, nil, nil
		call(ir.HandleAcknowledgeWarning, body)
		call(ir.HandleBookInventurDifferences, body)
		call(ir.HandleBookPickingList, body)
		require.NotNil(t, stub.ack)
		require.NotNil(t, stub.inventur)
		require.NotNil(t, stub.picking)
		assert.Equal(t, leaveCallerID, stub.ack.GetAcknowledgedBy(), "acknowledge: %s", body)
		assert.Equal(t, leaveCallerID, stub.inventur.GetBookedBy(), "stocktake booking: %s", body)
		assert.Equal(t, leaveCallerID, stub.picking.GetBookedBy(), "picking booking: %s", body)
	}
}
