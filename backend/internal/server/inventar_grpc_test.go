package server

import (
	"context"
	"encoding/csv"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/inventar"
	inventarv1 "github.com/kmuhub/kmuhub/proto/inventar/v1"
)

// ============================================================================
// Stub repository (server-package copy — inventar's own test doubles live in
// that package's _test.go files and aren't importable here). Every method
// returns r.err when set; otherwise it echoes back plausible defaults so the
// proto mappers in inventar_grpc.go never dereference a nil pointer.
// r.itemQuantity controls the stock level GetItem reports, so AdjustStock and
// TransferStock can be pushed into their insufficient-stock branch.
// ============================================================================

type stubInventarRepo struct {
	err error

	itemQuantity   int64
	sessionStatus  inventar.InventurStatus
	pickingStatus  inventar.PickingStatus
	bookingClaimed bool

	// listItems, when set, is returned verbatim by ListItems (default nil keeps
	// the existing always-empty behaviour other tests rely on).
	listItems []*inventar.Item
}

func newStubInventarRepo() *stubInventarRepo {
	return &stubInventarRepo{
		itemQuantity:   100,
		sessionStatus:  inventar.InventurStatusOpen,
		pickingStatus:  inventar.PickingStatusOpen,
		bookingClaimed: true,
	}
}

// --- Items ---

func (r *stubInventarRepo) CreateItem(_ context.Context, item *inventar.Item) error {
	if r.err != nil {
		return r.err
	}
	item.ID = uuid.New()
	return nil
}

func (r *stubInventarRepo) UpdateItem(_ context.Context, _ *inventar.Item) error { return r.err }

func (r *stubInventarRepo) SoftDeleteItem(_ context.Context, _, _ uuid.UUID) error { return r.err }

func (r *stubInventarRepo) GetItem(_ context.Context, tenantID, itemID uuid.UUID) (*inventar.Item, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &inventar.Item{
		ID: itemID, TenantID: tenantID, Name: "Test Item", SKU: "SKU-1",
		Quantity: r.itemQuantity, MinQuantity: 5, Unit: "pcs",
	}, nil
}

func (r *stubInventarRepo) ListItems(_ context.Context, _ uuid.UUID, _ inventar.ListItemsFilter, _, _ int) ([]*inventar.Item, int, error) {
	if r.err != nil {
		return nil, 0, r.err
	}
	return r.listItems, len(r.listItems), nil
}

func (r *stubInventarRepo) SKUExists(_ context.Context, _ uuid.UUID, _ string, _ *uuid.UUID) (bool, error) {
	if r.err != nil {
		return false, r.err
	}
	return false, nil
}

// --- Movements ---

func (r *stubInventarRepo) CreateMovement(_ context.Context, movement *inventar.Movement) error {
	if r.err != nil {
		return r.err
	}
	movement.ID = uuid.New()
	return nil
}

func (r *stubInventarRepo) GetMovement(_ context.Context, tenantID, movementID uuid.UUID) (*inventar.Movement, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &inventar.Movement{ID: movementID, TenantID: tenantID, ItemID: uuid.New(), MovementType: inventar.MovementTypeIn, Quantity: 1}, nil
}

func (r *stubInventarRepo) ListMovements(_ context.Context, _, _ uuid.UUID, _, _ int) ([]*inventar.Movement, int, error) {
	if r.err != nil {
		return nil, 0, r.err
	}
	return nil, 0, nil
}

// --- Warnings ---

func (r *stubInventarRepo) CreateWarning(_ context.Context, warning *inventar.Warning) error {
	if r.err != nil {
		return r.err
	}
	warning.ID = uuid.New()
	return nil
}

func (r *stubInventarRepo) UpdateWarning(_ context.Context, _ *inventar.Warning) error { return r.err }

func (r *stubInventarRepo) GetWarning(_ context.Context, tenantID, warningID uuid.UUID) (*inventar.Warning, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &inventar.Warning{ID: warningID, TenantID: tenantID, ItemID: uuid.New(), Status: inventar.WarningStatusActive}, nil
}

func (r *stubInventarRepo) GetActiveWarningForItem(_ context.Context, _, _ uuid.UUID) (*inventar.Warning, error) {
	return nil, inventar.ErrWarningNotFound
}

func (r *stubInventarRepo) ListWarnings(_ context.Context, _ uuid.UUID, _ *inventar.WarningStatus, _, _ int) ([]*inventar.Warning, int, error) {
	if r.err != nil {
		return nil, 0, r.err
	}
	return nil, 0, nil
}

// --- Locations ---

func (r *stubInventarRepo) CreateLocation(_ context.Context, loc *inventar.InventoryLocation) error {
	if r.err != nil {
		return r.err
	}
	loc.ID = uuid.New()
	return nil
}

func (r *stubInventarRepo) UpdateLocation(_ context.Context, _ *inventar.InventoryLocation) error {
	return r.err
}

func (r *stubInventarRepo) SoftDeleteLocation(_ context.Context, _, _ uuid.UUID) error { return r.err }

func (r *stubInventarRepo) GetLocation(_ context.Context, tenantID, locID uuid.UUID) (*inventar.InventoryLocation, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &inventar.InventoryLocation{ID: locID, TenantID: tenantID, Name: "WH", Type: inventar.LocationTypeWarehouse}, nil
}

func (r *stubInventarRepo) ListLocations(_ context.Context, _ uuid.UUID, _, _ int) ([]*inventar.InventoryLocation, int, error) {
	if r.err != nil {
		return nil, 0, r.err
	}
	return nil, 0, nil
}

// --- Inventur sessions ---

func (r *stubInventarRepo) CreateInventurSession(_ context.Context, session *inventar.InventurSession) error {
	if r.err != nil {
		return r.err
	}
	session.ID = uuid.New()
	return nil
}

func (r *stubInventarRepo) UpdateInventurSession(_ context.Context, _ *inventar.InventurSession) error {
	return r.err
}

func (r *stubInventarRepo) DeleteInventurSession(_ context.Context, _, _ uuid.UUID) error {
	return r.err
}

func (r *stubInventarRepo) GetInventurSession(_ context.Context, tenantID, sessionID uuid.UUID) (*inventar.InventurSession, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &inventar.InventurSession{ID: sessionID, TenantID: tenantID, Name: "Session", Status: r.sessionStatus}, nil
}

func (r *stubInventarRepo) ListInventurSessions(_ context.Context, _ uuid.UUID, _, _ int) ([]*inventar.InventurSession, int, error) {
	if r.err != nil {
		return nil, 0, r.err
	}
	return nil, 0, nil
}

func (r *stubInventarRepo) CompleteInventurSessionTx(_ context.Context, _, _ uuid.UUID, _ []inventar.StockMovementInput) ([]*inventar.Item, error) {
	if r.err != nil {
		return nil, r.err
	}
	return nil, nil
}

// --- Inventur counts ---

func (r *stubInventarRepo) UpsertInventurCount(_ context.Context, count *inventar.InventurCount) error {
	if r.err != nil {
		return r.err
	}
	count.ID = uuid.New()
	return nil
}

func (r *stubInventarRepo) ListInventurCounts(_ context.Context, _, _ uuid.UUID) ([]*inventar.InventurCount, error) {
	return nil, r.err
}

// --- Picking lists ---

func (r *stubInventarRepo) CreatePickingList(_ context.Context, list *inventar.PickingList) error {
	if r.err != nil {
		return r.err
	}
	list.ID = uuid.New()
	return nil
}

func (r *stubInventarRepo) UpdatePickingList(_ context.Context, _ *inventar.PickingList) error {
	return r.err
}

func (r *stubInventarRepo) BookPickingListTx(_ context.Context, _, _ uuid.UUID, _ []inventar.StockMovementInput) (bool, []*inventar.Item, error) {
	if r.err != nil {
		return false, nil, r.err
	}
	return r.bookingClaimed, nil, nil
}

func (r *stubInventarRepo) DeletePickingList(_ context.Context, _, _ uuid.UUID) error { return r.err }

func (r *stubInventarRepo) GetPickingList(_ context.Context, tenantID, listID uuid.UUID) (*inventar.PickingList, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &inventar.PickingList{ID: listID, TenantID: tenantID, Reference: "REF-1", Status: r.pickingStatus}, nil
}

func (r *stubInventarRepo) ListPickingLists(_ context.Context, _ uuid.UUID, _ *inventar.PickingStatus, _, _ int) ([]*inventar.PickingList, int, error) {
	if r.err != nil {
		return nil, 0, r.err
	}
	return nil, 0, nil
}

// --- Picking list items ---

func (r *stubInventarRepo) UpsertPickingListItem(_ context.Context, item *inventar.PickingListItem) error {
	if r.err != nil {
		return r.err
	}
	item.ID = uuid.New()
	return nil
}

func (r *stubInventarRepo) DeletePickingListItem(_ context.Context, _, _ uuid.UUID) error {
	return r.err
}

func (r *stubInventarRepo) ListPickingListItems(_ context.Context, _, _ uuid.UUID) ([]*inventar.PickingListItem, error) {
	return nil, r.err
}

// --- Item attachments ---

func (r *stubInventarRepo) CreateItemAttachment(_ context.Context, att *inventar.ItemAttachment) error {
	if r.err != nil {
		return r.err
	}
	att.ID = uuid.New()
	return nil
}

func (r *stubInventarRepo) ListItemAttachments(_ context.Context, _, _ uuid.UUID) ([]*inventar.ItemAttachment, error) {
	return nil, r.err
}

func (r *stubInventarRepo) DeleteItemAttachment(_ context.Context, _, _ uuid.UUID) error {
	return r.err
}

// ============================================================================
// Test helpers
// ============================================================================

func newInventarTestServer(repo inventar.Repository) *InventarGRPCServer {
	return NewInventarGRPCServer(inventar.NewService(repo))
}

// ============================================================================
// Error mapping — table test
// ============================================================================

func TestMapInventarError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"item not found", inventar.ErrItemNotFound, codes.NotFound},
		{"movement not found", inventar.ErrMovementNotFound, codes.NotFound},
		{"warning not found", inventar.ErrWarningNotFound, codes.NotFound},
		{"location not found", inventar.ErrLocationNotFound, codes.NotFound},
		{"inventur session not found", inventar.ErrInventurSessionNotFound, codes.NotFound},
		{"inventur count not found", inventar.ErrInventurCountNotFound, codes.NotFound},
		{"attachment not found", inventar.ErrAttachmentNotFound, codes.NotFound},
		{"picking list not found", inventar.ErrPickingListNotFound, codes.NotFound},
		{"picking list item not found", inventar.ErrPickingListItemNotFound, codes.NotFound},
		{"inventur already completed", inventar.ErrInventurAlreadyCompleted, codes.FailedPrecondition},
		{"picking list already booked", inventar.ErrPickingListAlreadyBooked, codes.FailedPrecondition},
		{"insufficient stock", inventar.ErrInsufficientStock, codes.FailedPrecondition},
		{"sku taken", inventar.ErrSKUTaken, codes.AlreadyExists},
		{"invalid input", inventar.ErrInvalidInput, codes.InvalidArgument},
		{"unmapped error falls back to internal", errors.New("boom"), codes.Internal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireGRPCCode(t, mapInventarError(tc.err), tc.want)
		})
	}
}

// ============================================================================
// Item RPCs
// ============================================================================

func TestInventarItemHandlers(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	itemID := uuid.New()

	t.Run("CreateItem invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.CreateItem(ctx, &inventarv1.CreateItemRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateItem invalid location_id", func(t *testing.T) {
		bad := "not-a-uuid"
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.CreateItem(ctx, &inventarv1.CreateItemRequest{TenantId: tenantID.String(), Name: "N", Sku: "S", LocationId: &bad})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateItem repo error maps through", func(t *testing.T) {
		repo := newStubInventarRepo()
		repo.err = inventar.ErrSKUTaken
		s := newInventarTestServer(repo)
		_, err := s.CreateItem(ctx, &inventarv1.CreateItemRequest{TenantId: tenantID.String(), Name: "N", Sku: "S"})
		requireGRPCCode(t, err, codes.AlreadyExists)
	})

	t.Run("CreateItem happy path echoes tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		resp, err := s.CreateItem(ctx, &inventarv1.CreateItemRequest{TenantId: tenantID.String(), Name: "Widget", Sku: "SKU-9", Unit: "pcs"})
		requireGRPCOK(t, err)
		require.Equal(t, tenantID.String(), resp.Item.TenantId)
		require.Equal(t, "Widget", resp.Item.Name)
	})

	t.Run("GetItem invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.GetItem(ctx, &inventarv1.GetItemRequest{TenantId: "bad", ItemId: itemID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetItem invalid item_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.GetItem(ctx, &inventarv1.GetItemRequest{TenantId: tenantID.String(), ItemId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetItem not found maps through", func(t *testing.T) {
		repo := newStubInventarRepo()
		repo.err = inventar.ErrItemNotFound
		s := newInventarTestServer(repo)
		_, err := s.GetItem(ctx, &inventarv1.GetItemRequest{TenantId: tenantID.String(), ItemId: itemID.String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("UpdateItem invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.UpdateItem(ctx, &inventarv1.UpdateItemRequest{TenantId: "bad", ItemId: itemID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateItem invalid item_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.UpdateItem(ctx, &inventarv1.UpdateItemRequest{TenantId: tenantID.String(), ItemId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateItem invalid location_id", func(t *testing.T) {
		bad := "nope"
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.UpdateItem(ctx, &inventarv1.UpdateItemRequest{TenantId: tenantID.String(), ItemId: itemID.String(), LocationId: &bad})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteItem invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.DeleteItem(ctx, &inventarv1.DeleteItemRequest{TenantId: "bad", ItemId: itemID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteItem invalid item_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.DeleteItem(ctx, &inventarv1.DeleteItemRequest{TenantId: tenantID.String(), ItemId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteItem not found maps through", func(t *testing.T) {
		repo := newStubInventarRepo()
		repo.err = inventar.ErrItemNotFound
		s := newInventarTestServer(repo)
		_, err := s.DeleteItem(ctx, &inventarv1.DeleteItemRequest{TenantId: tenantID.String(), ItemId: itemID.String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("ListItems invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.ListItems(ctx, &inventarv1.ListItemsRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListItems empty result wraps as empty slice not null", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		resp, err := s.ListItems(ctx, &inventarv1.ListItemsRequest{TenantId: tenantID.String()})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.Items)
		require.Empty(t, resp.Items)
	})
}

// ============================================================================
// Stock operation RPCs — in and out are exercised separately, including the
// insufficient-stock branch, since AdjustStock/TransferStock are the only
// signed-quantity paths in this file.
// ============================================================================

func TestInventarStockHandlers(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	itemID := uuid.New()
	fromID := uuid.New()
	toID := uuid.New()

	t.Run("AdjustStock invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.AdjustStock(ctx, &inventarv1.AdjustStockRequest{TenantId: "bad", ItemId: itemID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("AdjustStock invalid item_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.AdjustStock(ctx, &inventarv1.AdjustStockRequest{TenantId: tenantID.String(), ItemId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("AdjustStock invalid performed_by", func(t *testing.T) {
		bad := "nope"
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.AdjustStock(ctx, &inventarv1.AdjustStockRequest{
			TenantId: tenantID.String(), ItemId: itemID.String(), Delta: 5, PerformedBy: &bad,
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("AdjustStock positive delta books an inbound movement", func(t *testing.T) {
		repo := newStubInventarRepo()
		repo.itemQuantity = 10
		s := newInventarTestServer(repo)
		resp, err := s.AdjustStock(ctx, &inventarv1.AdjustStockRequest{TenantId: tenantID.String(), ItemId: itemID.String(), Delta: 5, Reason: "restock"})
		requireGRPCOK(t, err)
		require.Equal(t, int64(15), resp.Item.Quantity)
	})

	t.Run("AdjustStock negative delta within stock books an outbound movement", func(t *testing.T) {
		repo := newStubInventarRepo()
		repo.itemQuantity = 10
		s := newInventarTestServer(repo)
		resp, err := s.AdjustStock(ctx, &inventarv1.AdjustStockRequest{TenantId: tenantID.String(), ItemId: itemID.String(), Delta: -4, Reason: "pick"})
		requireGRPCOK(t, err)
		require.Equal(t, int64(6), resp.Item.Quantity)
	})

	t.Run("AdjustStock negative delta below zero is rejected as failed precondition", func(t *testing.T) {
		repo := newStubInventarRepo()
		repo.itemQuantity = 3
		s := newInventarTestServer(repo)
		_, err := s.AdjustStock(ctx, &inventarv1.AdjustStockRequest{TenantId: tenantID.String(), ItemId: itemID.String(), Delta: -10, Reason: "pick"})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("AdjustStock zero delta is rejected as invalid argument", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.AdjustStock(ctx, &inventarv1.AdjustStockRequest{TenantId: tenantID.String(), ItemId: itemID.String(), Delta: 0})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("TransferStock invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.TransferStock(ctx, &inventarv1.TransferStockRequest{TenantId: "bad", FromItemId: fromID.String(), ToItemId: toID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("TransferStock invalid from_item_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.TransferStock(ctx, &inventarv1.TransferStockRequest{TenantId: tenantID.String(), FromItemId: "bad", ToItemId: toID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("TransferStock invalid to_item_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.TransferStock(ctx, &inventarv1.TransferStockRequest{TenantId: tenantID.String(), FromItemId: fromID.String(), ToItemId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("TransferStock invalid performed_by", func(t *testing.T) {
		bad := "nope"
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.TransferStock(ctx, &inventarv1.TransferStockRequest{
			TenantId: tenantID.String(), FromItemId: fromID.String(), ToItemId: toID.String(), Quantity: 1, PerformedBy: &bad,
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("TransferStock zero or negative quantity is rejected", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.TransferStock(ctx, &inventarv1.TransferStockRequest{TenantId: tenantID.String(), FromItemId: fromID.String(), ToItemId: toID.String(), Quantity: 0})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("TransferStock more than available stock is rejected as failed precondition", func(t *testing.T) {
		repo := newStubInventarRepo()
		repo.itemQuantity = 3
		s := newInventarTestServer(repo)
		_, err := s.TransferStock(ctx, &inventarv1.TransferStockRequest{TenantId: tenantID.String(), FromItemId: fromID.String(), ToItemId: toID.String(), Quantity: 100})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("TransferStock within stock succeeds", func(t *testing.T) {
		repo := newStubInventarRepo()
		repo.itemQuantity = 10
		s := newInventarTestServer(repo)
		_, err := s.TransferStock(ctx, &inventarv1.TransferStockRequest{TenantId: tenantID.String(), FromItemId: fromID.String(), ToItemId: toID.String(), Quantity: 5})
		requireGRPCOK(t, err)
	})

	t.Run("RecordMovement invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.RecordMovement(ctx, &inventarv1.RecordMovementRequest{TenantId: "bad", ItemId: itemID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("RecordMovement invalid item_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.RecordMovement(ctx, &inventarv1.RecordMovementRequest{TenantId: tenantID.String(), ItemId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("RecordMovement invalid performed_by", func(t *testing.T) {
		bad := "nope"
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.RecordMovement(ctx, &inventarv1.RecordMovementRequest{
			TenantId: tenantID.String(), ItemId: itemID.String(), Quantity: 1, MovementType: "in", PerformedBy: &bad,
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("RecordMovement zero quantity is rejected", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.RecordMovement(ctx, &inventarv1.RecordMovementRequest{TenantId: tenantID.String(), ItemId: itemID.String(), Quantity: 0})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("RecordMovement happy path maps type through", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		resp, err := s.RecordMovement(ctx, &inventarv1.RecordMovementRequest{
			TenantId: tenantID.String(), ItemId: itemID.String(), Quantity: -3, MovementType: "out", Reason: "pick",
		})
		requireGRPCOK(t, err)
		require.Equal(t, "out", resp.Movement.MovementType)
		require.Equal(t, int64(-3), resp.Movement.Quantity)
	})

	t.Run("ListMovements invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.ListMovements(ctx, &inventarv1.ListMovementsRequest{TenantId: "bad", ItemId: itemID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListMovements invalid item_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.ListMovements(ctx, &inventarv1.ListMovementsRequest{TenantId: tenantID.String(), ItemId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListMovements empty result wraps as empty slice not null", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		resp, err := s.ListMovements(ctx, &inventarv1.ListMovementsRequest{TenantId: tenantID.String(), ItemId: itemID.String()})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.Movements)
		require.Empty(t, resp.Movements)
	})

	t.Run("GetStockHistory is an alias for ListMovements", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.GetStockHistory(ctx, &inventarv1.GetStockHistoryRequest{TenantId: "bad", ItemId: itemID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)

		resp, err := s.GetStockHistory(ctx, &inventarv1.GetStockHistoryRequest{TenantId: tenantID.String(), ItemId: itemID.String()})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.Movements)
	})
}

// ============================================================================
// Warning RPCs
// ============================================================================

func TestInventarWarningHandlers(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	itemID := uuid.New()
	warningID := uuid.New()
	userID := uuid.New()

	t.Run("CreateWarning invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.CreateWarning(ctx, &inventarv1.CreateWarningRequest{TenantId: "bad", ItemId: itemID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateWarning invalid item_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.CreateWarning(ctx, &inventarv1.CreateWarningRequest{TenantId: tenantID.String(), ItemId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateWarning item not found maps through", func(t *testing.T) {
		repo := newStubInventarRepo()
		repo.err = inventar.ErrItemNotFound
		s := newInventarTestServer(repo)
		_, err := s.CreateWarning(ctx, &inventarv1.CreateWarningRequest{TenantId: tenantID.String(), ItemId: itemID.String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("UpdateWarning invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.UpdateWarning(ctx, &inventarv1.UpdateWarningRequest{TenantId: "bad", WarningId: warningID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateWarning invalid warning_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.UpdateWarning(ctx, &inventarv1.UpdateWarningRequest{TenantId: tenantID.String(), WarningId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateWarning not found maps through", func(t *testing.T) {
		repo := newStubInventarRepo()
		repo.err = inventar.ErrWarningNotFound
		s := newInventarTestServer(repo)
		_, err := s.UpdateWarning(ctx, &inventarv1.UpdateWarningRequest{TenantId: tenantID.String(), WarningId: warningID.String(), Status: "resolved"})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("AcknowledgeWarning invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.AcknowledgeWarning(ctx, &inventarv1.AcknowledgeWarningRequest{TenantId: "bad", WarningId: warningID.String(), AcknowledgedBy: userID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("AcknowledgeWarning invalid warning_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.AcknowledgeWarning(ctx, &inventarv1.AcknowledgeWarningRequest{TenantId: tenantID.String(), WarningId: "bad", AcknowledgedBy: userID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("AcknowledgeWarning invalid acknowledged_by", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.AcknowledgeWarning(ctx, &inventarv1.AcknowledgeWarningRequest{TenantId: tenantID.String(), WarningId: warningID.String(), AcknowledgedBy: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("AcknowledgeWarning happy path sets status", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		resp, err := s.AcknowledgeWarning(ctx, &inventarv1.AcknowledgeWarningRequest{TenantId: tenantID.String(), WarningId: warningID.String(), AcknowledgedBy: userID.String()})
		requireGRPCOK(t, err)
		require.Equal(t, string(inventar.WarningStatusAcknowledged), resp.Warning.Status)
		require.Equal(t, userID.String(), *resp.Warning.AcknowledgedBy)
	})

	t.Run("ListWarnings invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.ListWarnings(ctx, &inventarv1.ListWarningsRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListWarnings empty result wraps as empty slice not null", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		resp, err := s.ListWarnings(ctx, &inventarv1.ListWarningsRequest{TenantId: tenantID.String()})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.Warnings)
		require.Empty(t, resp.Warnings)
	})
}

// ============================================================================
// Report RPCs
// ============================================================================

func TestInventarReportHandlers(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("GetStockReport invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.GetStockReport(ctx, &inventarv1.GetStockReportRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetStockReport happy path", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		resp, err := s.GetStockReport(ctx, &inventarv1.GetStockReportRequest{TenantId: tenantID.String()})
		requireGRPCOK(t, err)
		require.Equal(t, int32(0), resp.TotalItems)
	})

	t.Run("ExportInventory invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.ExportInventory(ctx, &inventarv1.ExportInventoryRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ExportInventory happy path returns csv with header", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		resp, err := s.ExportInventory(ctx, &inventarv1.ExportInventoryRequest{TenantId: tenantID.String()})
		requireGRPCOK(t, err)
		require.Equal(t, "text/csv", resp.ContentType)
		require.Contains(t, string(resp.Payload), "sku")
	})
}

// ============================================================================
// Location RPCs
// ============================================================================

func TestInventarLocationHandlers(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	locID := uuid.New()

	t.Run("CreateLocation invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.CreateLocation(ctx, &inventarv1.CreateLocationRequest{TenantId: "bad", Name: "WH"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateLocation empty name is rejected", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.CreateLocation(ctx, &inventarv1.CreateLocationRequest{TenantId: tenantID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetLocation invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.GetLocation(ctx, &inventarv1.GetLocationRequest{TenantId: "bad", LocationId: locID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetLocation invalid location_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.GetLocation(ctx, &inventarv1.GetLocationRequest{TenantId: tenantID.String(), LocationId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateLocation invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.UpdateLocation(ctx, &inventarv1.UpdateLocationRequest{TenantId: "bad", LocationId: locID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateLocation invalid location_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.UpdateLocation(ctx, &inventarv1.UpdateLocationRequest{TenantId: tenantID.String(), LocationId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateLocation not found maps through", func(t *testing.T) {
		repo := newStubInventarRepo()
		repo.err = inventar.ErrLocationNotFound
		s := newInventarTestServer(repo)
		_, err := s.UpdateLocation(ctx, &inventarv1.UpdateLocationRequest{TenantId: tenantID.String(), LocationId: locID.String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("DeleteLocation invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.DeleteLocation(ctx, &inventarv1.DeleteLocationRequest{TenantId: "bad", LocationId: locID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteLocation invalid location_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.DeleteLocation(ctx, &inventarv1.DeleteLocationRequest{TenantId: tenantID.String(), LocationId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListLocations invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.ListLocations(ctx, &inventarv1.ListLocationsRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListLocations empty result wraps as empty slice not null", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		resp, err := s.ListLocations(ctx, &inventarv1.ListLocationsRequest{TenantId: tenantID.String()})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.Locations)
		require.Empty(t, resp.Locations)
	})
}

// ============================================================================
// Inventur session RPCs
// ============================================================================

func TestInventarInventurSessionHandlers(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	sessionID := uuid.New()

	t.Run("CreateInventurSession invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.CreateInventurSession(ctx, &inventarv1.CreateInventurSessionRequest{TenantId: "bad", Name: "Q1"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateInventurSession invalid location_id", func(t *testing.T) {
		bad := "nope"
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.CreateInventurSession(ctx, &inventarv1.CreateInventurSessionRequest{TenantId: tenantID.String(), Name: "Q1", LocationId: &bad})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateInventurSession invalid created_by", func(t *testing.T) {
		bad := "nope"
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.CreateInventurSession(ctx, &inventarv1.CreateInventurSessionRequest{TenantId: tenantID.String(), Name: "Q1", CreatedBy: &bad})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateInventurSession invalid item_id in item_ids", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.CreateInventurSession(ctx, &inventarv1.CreateInventurSessionRequest{TenantId: tenantID.String(), Name: "Q1", ItemIds: []string{"bad"}})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateInventurSession empty name is rejected", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.CreateInventurSession(ctx, &inventarv1.CreateInventurSessionRequest{TenantId: tenantID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetInventurSession invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.GetInventurSession(ctx, &inventarv1.GetInventurSessionRequest{TenantId: "bad", SessionId: sessionID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetInventurSession invalid session_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.GetInventurSession(ctx, &inventarv1.GetInventurSessionRequest{TenantId: tenantID.String(), SessionId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateInventurSessionStatus invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.UpdateInventurSessionStatus(ctx, &inventarv1.UpdateInventurSessionStatusRequest{TenantId: "bad", SessionId: sessionID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateInventurSessionStatus invalid session_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.UpdateInventurSessionStatus(ctx, &inventarv1.UpdateInventurSessionStatusRequest{TenantId: tenantID.String(), SessionId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdateInventurSessionStatus already completed is rejected", func(t *testing.T) {
		repo := newStubInventarRepo()
		repo.sessionStatus = inventar.InventurStatusCompleted
		s := newInventarTestServer(repo)
		_, err := s.UpdateInventurSessionStatus(ctx, &inventarv1.UpdateInventurSessionStatusRequest{TenantId: tenantID.String(), SessionId: sessionID.String(), Status: "counting"})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("DeleteInventurSession invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.DeleteInventurSession(ctx, &inventarv1.DeleteInventurSessionRequest{TenantId: "bad", SessionId: sessionID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteInventurSession invalid session_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.DeleteInventurSession(ctx, &inventarv1.DeleteInventurSessionRequest{TenantId: tenantID.String(), SessionId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListInventurSessions invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.ListInventurSessions(ctx, &inventarv1.ListInventurSessionsRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpsertInventurCount invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.UpsertInventurCount(ctx, &inventarv1.UpsertInventurCountRequest{TenantId: "bad", SessionId: sessionID.String(), ItemId: uuid.New().String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpsertInventurCount invalid session_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.UpsertInventurCount(ctx, &inventarv1.UpsertInventurCountRequest{TenantId: tenantID.String(), SessionId: "bad", ItemId: uuid.New().String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpsertInventurCount invalid item_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.UpsertInventurCount(ctx, &inventarv1.UpsertInventurCountRequest{TenantId: tenantID.String(), SessionId: sessionID.String(), ItemId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpsertInventurCount on a completed session is rejected", func(t *testing.T) {
		repo := newStubInventarRepo()
		repo.sessionStatus = inventar.InventurStatusCompleted
		s := newInventarTestServer(repo)
		_, err := s.UpsertInventurCount(ctx, &inventarv1.UpsertInventurCountRequest{TenantId: tenantID.String(), SessionId: sessionID.String(), ItemId: uuid.New().String(), Counted: 5})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("BookInventurDifferences invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.BookInventurDifferences(ctx, &inventarv1.BookInventurDifferencesRequest{TenantId: "bad", SessionId: sessionID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("BookInventurDifferences invalid session_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.BookInventurDifferences(ctx, &inventarv1.BookInventurDifferencesRequest{TenantId: tenantID.String(), SessionId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("BookInventurDifferences invalid booked_by", func(t *testing.T) {
		bad := "nope"
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.BookInventurDifferences(ctx, &inventarv1.BookInventurDifferencesRequest{TenantId: tenantID.String(), SessionId: sessionID.String(), BookedBy: &bad})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("BookInventurDifferences on an already completed session is rejected", func(t *testing.T) {
		repo := newStubInventarRepo()
		repo.sessionStatus = inventar.InventurStatusCompleted
		s := newInventarTestServer(repo)
		_, err := s.BookInventurDifferences(ctx, &inventarv1.BookInventurDifferencesRequest{TenantId: tenantID.String(), SessionId: sessionID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})
}

// ============================================================================
// Picking list RPCs
// ============================================================================

func TestInventarPickingListHandlers(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	listID := uuid.New()
	itemID := uuid.New()
	rowID := uuid.New()

	t.Run("CreatePickingList invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.CreatePickingList(ctx, &inventarv1.CreatePickingListRequest{TenantId: "bad", Reference: "REF"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreatePickingList invalid assigned_to", func(t *testing.T) {
		bad := "nope"
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.CreatePickingList(ctx, &inventarv1.CreatePickingListRequest{TenantId: tenantID.String(), Reference: "REF", AssignedTo: &bad})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreatePickingList invalid created_by", func(t *testing.T) {
		bad := "nope"
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.CreatePickingList(ctx, &inventarv1.CreatePickingListRequest{TenantId: tenantID.String(), Reference: "REF", CreatedBy: &bad})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreatePickingList invalid item_id in items", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.CreatePickingList(ctx, &inventarv1.CreatePickingListRequest{
			TenantId: tenantID.String(), Reference: "REF",
			Items: []*inventarv1.CreatePickingListItemInput{{ItemId: "bad", QuantityRequested: 1}},
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreatePickingList empty reference is rejected", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.CreatePickingList(ctx, &inventarv1.CreatePickingListRequest{TenantId: tenantID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreatePickingList picked exceeding requested is rejected", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.CreatePickingList(ctx, &inventarv1.CreatePickingListRequest{
			TenantId: tenantID.String(), Reference: "REF",
			Items: []*inventarv1.CreatePickingListItemInput{{ItemId: itemID.String(), QuantityRequested: 1, QuantityPicked: 5}},
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetPickingList invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.GetPickingList(ctx, &inventarv1.GetPickingListRequest{TenantId: "bad", PickingListId: listID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("GetPickingList invalid picking_list_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.GetPickingList(ctx, &inventarv1.GetPickingListRequest{TenantId: tenantID.String(), PickingListId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdatePickingList invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.UpdatePickingList(ctx, &inventarv1.UpdatePickingListRequest{TenantId: "bad", PickingListId: listID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdatePickingList invalid picking_list_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.UpdatePickingList(ctx, &inventarv1.UpdatePickingListRequest{TenantId: tenantID.String(), PickingListId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdatePickingList invalid assigned_to", func(t *testing.T) {
		bad := "nope"
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.UpdatePickingList(ctx, &inventarv1.UpdatePickingListRequest{TenantId: tenantID.String(), PickingListId: listID.String(), AssignedTo: &bad})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdatePickingList already completed is rejected", func(t *testing.T) {
		repo := newStubInventarRepo()
		repo.pickingStatus = inventar.PickingStatusCompleted
		s := newInventarTestServer(repo)
		ref := "REF-2"
		_, err := s.UpdatePickingList(ctx, &inventarv1.UpdatePickingListRequest{TenantId: tenantID.String(), PickingListId: listID.String(), Reference: &ref})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("UpdatePickingList setting status to completed directly is rejected", func(t *testing.T) {
		completed := "completed"
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.UpdatePickingList(ctx, &inventarv1.UpdatePickingListRequest{TenantId: tenantID.String(), PickingListId: listID.String(), Status: &completed})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpdatePickingList unknown status is rejected", func(t *testing.T) {
		bogus := "bogus"
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.UpdatePickingList(ctx, &inventarv1.UpdatePickingListRequest{TenantId: tenantID.String(), PickingListId: listID.String(), Status: &bogus})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeletePickingList invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.DeletePickingList(ctx, &inventarv1.DeletePickingListRequest{TenantId: "bad", PickingListId: listID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeletePickingList invalid picking_list_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.DeletePickingList(ctx, &inventarv1.DeletePickingListRequest{TenantId: tenantID.String(), PickingListId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListPickingLists invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.ListPickingLists(ctx, &inventarv1.ListPickingListsRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListPickingLists empty result wraps as empty slice not null", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		resp, err := s.ListPickingLists(ctx, &inventarv1.ListPickingListsRequest{TenantId: tenantID.String()})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.PickingLists)
		require.Empty(t, resp.PickingLists)
	})

	t.Run("UpsertPickingListItem invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.UpsertPickingListItem(ctx, &inventarv1.UpsertPickingListItemRequest{TenantId: "bad", PickingListId: listID.String(), ItemId: itemID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpsertPickingListItem invalid picking_list_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.UpsertPickingListItem(ctx, &inventarv1.UpsertPickingListItemRequest{TenantId: tenantID.String(), PickingListId: "bad", ItemId: itemID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpsertPickingListItem invalid item_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.UpsertPickingListItem(ctx, &inventarv1.UpsertPickingListItemRequest{TenantId: tenantID.String(), PickingListId: listID.String(), ItemId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpsertPickingListItem picked exceeding requested is rejected", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.UpsertPickingListItem(ctx, &inventarv1.UpsertPickingListItemRequest{
			TenantId: tenantID.String(), PickingListId: listID.String(), ItemId: itemID.String(),
			QuantityRequested: 2, QuantityPicked: 3,
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("UpsertPickingListItem on an already completed list is rejected", func(t *testing.T) {
		repo := newStubInventarRepo()
		repo.pickingStatus = inventar.PickingStatusCompleted
		s := newInventarTestServer(repo)
		_, err := s.UpsertPickingListItem(ctx, &inventarv1.UpsertPickingListItemRequest{
			TenantId: tenantID.String(), PickingListId: listID.String(), ItemId: itemID.String(), QuantityRequested: 2, QuantityPicked: 1,
		})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("DeletePickingListItem invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.DeletePickingListItem(ctx, &inventarv1.DeletePickingListItemRequest{TenantId: "bad", PickingListItemId: rowID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeletePickingListItem invalid picking_list_item_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.DeletePickingListItem(ctx, &inventarv1.DeletePickingListItemRequest{TenantId: tenantID.String(), PickingListItemId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("BookPickingList invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.BookPickingList(ctx, &inventarv1.BookPickingListRequest{TenantId: "bad", PickingListId: listID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("BookPickingList invalid picking_list_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.BookPickingList(ctx, &inventarv1.BookPickingListRequest{TenantId: tenantID.String(), PickingListId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("BookPickingList invalid booked_by", func(t *testing.T) {
		bad := "nope"
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.BookPickingList(ctx, &inventarv1.BookPickingListRequest{TenantId: tenantID.String(), PickingListId: listID.String(), BookedBy: &bad})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("BookPickingList already completed is rejected without touching the repo", func(t *testing.T) {
		repo := newStubInventarRepo()
		repo.pickingStatus = inventar.PickingStatusCompleted
		s := newInventarTestServer(repo)
		_, err := s.BookPickingList(ctx, &inventarv1.BookPickingListRequest{TenantId: tenantID.String(), PickingListId: listID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("BookPickingList second concurrent booking finds it already claimed", func(t *testing.T) {
		repo := newStubInventarRepo()
		repo.bookingClaimed = false // Tx reports another caller already claimed it
		s := newInventarTestServer(repo)
		_, err := s.BookPickingList(ctx, &inventarv1.BookPickingListRequest{TenantId: tenantID.String(), PickingListId: listID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("BookPickingList happy path books an open list", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		resp, err := s.BookPickingList(ctx, &inventarv1.BookPickingListRequest{TenantId: tenantID.String(), PickingListId: listID.String()})
		requireGRPCOK(t, err)
		require.Equal(t, listID.String(), resp.PickingList.Id)
	})
}

// ============================================================================
// Item attachment RPCs
// ============================================================================

func TestInventarItemAttachmentHandlers(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	itemID := uuid.New()
	attachmentID := uuid.New()

	t.Run("CreateItemAttachment invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.CreateItemAttachment(ctx, &inventarv1.CreateItemAttachmentRequest{TenantId: "bad", ItemId: itemID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateItemAttachment invalid item_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.CreateItemAttachment(ctx, &inventarv1.CreateItemAttachmentRequest{TenantId: tenantID.String(), ItemId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateItemAttachment missing name or object_key is rejected", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.CreateItemAttachment(ctx, &inventarv1.CreateItemAttachmentRequest{TenantId: tenantID.String(), ItemId: itemID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("CreateItemAttachment item not found maps through", func(t *testing.T) {
		repo := newStubInventarRepo()
		repo.err = inventar.ErrItemNotFound
		s := newInventarTestServer(repo)
		_, err := s.CreateItemAttachment(ctx, &inventarv1.CreateItemAttachmentRequest{
			TenantId: tenantID.String(), ItemId: itemID.String(), Name: "invoice.pdf", ObjectKey: "k",
		})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("ListItemAttachments invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.ListItemAttachments(ctx, &inventarv1.ListItemAttachmentsRequest{TenantId: "bad", ItemId: itemID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListItemAttachments invalid item_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.ListItemAttachments(ctx, &inventarv1.ListItemAttachmentsRequest{TenantId: tenantID.String(), ItemId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("ListItemAttachments empty result is [] not nil", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		resp, err := s.ListItemAttachments(ctx, &inventarv1.ListItemAttachmentsRequest{TenantId: tenantID.String(), ItemId: itemID.String()})
		require.NoError(t, err)
		require.NotNil(t, resp.Attachments)
		require.Empty(t, resp.Attachments)
	})

	t.Run("DeleteItemAttachment invalid tenant", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.DeleteItemAttachment(ctx, &inventarv1.DeleteItemAttachmentRequest{TenantId: "bad", AttachmentId: attachmentID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("DeleteItemAttachment invalid attachment_id", func(t *testing.T) {
		s := newInventarTestServer(newStubInventarRepo())
		_, err := s.DeleteItemAttachment(ctx, &inventarv1.DeleteItemAttachmentRequest{TenantId: tenantID.String(), AttachmentId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestInventar_ExportInventory_NeutralizesFormulaInjection(t *testing.T) {
	tenantID := uuid.New()
	barcode := "=1+1"
	location := "+cmd|'/c calc'!A1"
	repo := newStubInventarRepo()
	repo.listItems = []*inventar.Item{
		{
			ID: uuid.New(), TenantID: tenantID,
			Name: "=SUM(A1:A9)", SKU: "@import(evil)", Barcode: &barcode,
			Quantity: 3, MinQuantity: 1, Unit: "-1", Location: &location,
		},
		{
			ID: uuid.New(), TenantID: tenantID,
			Name: "Normale Schraube", SKU: "SKU-42", Quantity: 10, MinQuantity: 2, Unit: "pcs",
		},
	}
	s := newInventarTestServer(repo)

	resp, err := s.ExportInventory(context.Background(), &inventarv1.ExportInventoryRequest{TenantId: tenantID.String()})
	require.NoError(t, err)

	rows, err := csv.NewReader(strings.NewReader(string(resp.Payload))).ReadAll()
	require.NoError(t, err)
	require.Len(t, rows, 3) // header + 2 items

	dangerous := rows[1]
	require.Equal(t, "'=SUM(A1:A9)", dangerous[1], "name")
	require.Equal(t, "'@import(evil)", dangerous[2], "sku")
	require.Equal(t, "'=1+1", dangerous[3], "barcode")
	require.Equal(t, "'-1", dangerous[6], "unit")
	require.Equal(t, "'+cmd|'/c calc'!A1", dangerous[7], "location")

	normal := rows[2]
	require.Equal(t, "Normale Schraube", normal[1])
	require.Equal(t, "SKU-42", normal[2])
	require.Equal(t, "pcs", normal[6])
}
