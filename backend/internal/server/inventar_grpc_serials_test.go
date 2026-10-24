package server

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/inventar"
	inventarv1 "github.com/kmuhub/kmuhub/proto/inventar/v1"
)

// serialsRepo reports an item that already carries serial numbers.
type serialsRepo struct{ *stubInventarRepo }

func (r serialsRepo) GetItem(ctx context.Context, tenantID, itemID uuid.UUID) (*inventar.Item, error) {
	item, err := r.stubInventarRepo.GetItem(ctx, tenantID, itemID)
	if err != nil {
		return nil, err
	}
	item.SerialNumbers = []string{"old-1", "old-2"}
	return item, nil
}

func TestUpdateItem_SerialNumbersReplaceClearOrStay(t *testing.T) {
	s := newInventarTestServer(serialsRepo{newStubInventarRepo()})
	base := func() *inventarv1.UpdateItemRequest {
		return &inventarv1.UpdateItemRequest{TenantId: uuid.NewString(), ItemId: uuid.NewString()}
	}

	t.Run("a list replaces the stored one", func(t *testing.T) {
		req := base()
		req.SerialNumbers = &inventarv1.SerialNumberList{Values: []string{"n-1"}}
		resp, err := s.UpdateItem(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, []string{"n-1"}, resp.GetItem().GetSerialNumbers())
	})

	t.Run("an empty list clears them", func(t *testing.T) {
		req := base()
		req.SerialNumbers = &inventarv1.SerialNumberList{}
		resp, err := s.UpdateItem(context.Background(), req)
		require.NoError(t, err)
		assert.Empty(t, resp.GetItem().GetSerialNumbers())
	})

	t.Run("an absent field leaves them alone", func(t *testing.T) {
		resp, err := s.UpdateItem(context.Background(), base())
		require.NoError(t, err)
		assert.Equal(t, []string{"old-1", "old-2"}, resp.GetItem().GetSerialNumbers())
	})
}
