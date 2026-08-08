package inventar

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Picking part of mockRepository (struct fields live in service_test.go)
// ============================================================================

func (m *mockRepository) ensurePicking() {
	if m.pickingLists == nil {
		m.pickingLists = make(map[uuid.UUID]*PickingList)
	}
	if m.pickingItems == nil {
		m.pickingItems = make(map[uuid.UUID][]*PickingListItem)
	}
}

func (m *mockRepository) CreatePickingList(_ context.Context, list *PickingList) error {
	m.ensurePicking()
	cp := *list
	m.pickingLists[list.ID] = &cp
	return nil
}

func (m *mockRepository) UpdatePickingList(_ context.Context, list *PickingList) error {
	m.ensurePicking()
	stored, ok := m.pickingLists[list.ID]
	if !ok || stored.TenantID != list.TenantID {
		return ErrPickingListNotFound
	}
	cp := *list
	cp.Items = nil
	m.pickingLists[list.ID] = &cp
	return nil
}

func (m *mockRepository) CompletePickingList(_ context.Context, tenantID, listID uuid.UUID) (bool, error) {
	m.ensurePicking()
	stored, ok := m.pickingLists[listID]
	if !ok || stored.TenantID != tenantID || stored.Status == PickingStatusCompleted {
		return false, nil
	}
	stored.Status = PickingStatusCompleted
	return true, nil
}

func (m *mockRepository) DeletePickingList(_ context.Context, tenantID, listID uuid.UUID) error {
	m.ensurePicking()
	stored, ok := m.pickingLists[listID]
	if !ok || stored.TenantID != tenantID {
		return ErrPickingListNotFound
	}
	delete(m.pickingLists, listID)
	delete(m.pickingItems, listID)
	return nil
}

func (m *mockRepository) GetPickingList(_ context.Context, tenantID, listID uuid.UUID) (*PickingList, error) {
	m.ensurePicking()
	stored, ok := m.pickingLists[listID]
	if !ok || stored.TenantID != tenantID {
		return nil, ErrPickingListNotFound
	}
	cp := *stored
	cp.Items = nil
	for _, it := range m.pickingItems[listID] {
		cp.Items = append(cp.Items, *it)
	}
	return &cp, nil
}

func (m *mockRepository) ListPickingLists(_ context.Context, tenantID uuid.UUID, status *PickingStatus, offset, limit int) ([]*PickingList, int, error) {
	m.ensurePicking()
	all := []*PickingList{}
	for _, l := range m.pickingLists {
		if l.TenantID != tenantID {
			continue
		}
		if status != nil && l.Status != *status {
			continue
		}
		cp := *l
		all = append(all, &cp)
	}
	total := len(all)
	if offset >= total {
		return []*PickingList{}, total, nil
	}
	return all[offset:min(offset+limit, total)], total, nil
}

func (m *mockRepository) UpsertPickingListItem(_ context.Context, item *PickingListItem) error {
	m.ensurePicking()
	for _, existing := range m.pickingItems[item.PickingListID] {
		if existing.ItemID == item.ItemID {
			existing.QuantityRequested = item.QuantityRequested
			existing.QuantityPicked = item.QuantityPicked
			existing.Location = item.Location
			item.ID = existing.ID
			return nil
		}
	}
	cp := *item
	m.pickingItems[item.PickingListID] = append(m.pickingItems[item.PickingListID], &cp)
	return nil
}

func (m *mockRepository) DeletePickingListItem(_ context.Context, tenantID, itemRowID uuid.UUID) error {
	m.ensurePicking()
	for listID, rows := range m.pickingItems {
		for i, row := range rows {
			if row.ID == itemRowID && row.TenantID == tenantID {
				m.pickingItems[listID] = append(rows[:i], rows[i+1:]...)
				return nil
			}
		}
	}
	return ErrPickingListItemNotFound
}

func (m *mockRepository) ListPickingListItems(_ context.Context, tenantID, listID uuid.UUID) ([]*PickingListItem, error) {
	m.ensurePicking()
	out := []*PickingListItem{}
	for _, it := range m.pickingItems[listID] {
		if it.TenantID == tenantID {
			out = append(out, it)
		}
	}
	return out, nil
}

// ============================================================================
// Tests
// ============================================================================

func newPickingFixture(t *testing.T) (*Service, *mockRepository, uuid.UUID, *Item) {
	t.Helper()
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()
	item := addItem(repo, tenantID, "Schraube M6", "SKU-M6", 100, 5)
	return svc, repo, tenantID, item
}

func TestCreatePickingList_RequiresReference(t *testing.T) {
	svc, _, tenantID, _ := newPickingFixture(t)

	_, err := svc.CreatePickingList(context.Background(), CreatePickingListInput{
		TenantID: tenantID,
	})
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestCreatePickingList_RejectsPickedAboveRequested(t *testing.T) {
	svc, _, tenantID, item := newPickingFixture(t)

	_, err := svc.CreatePickingList(context.Background(), CreatePickingListInput{
		TenantID:  tenantID,
		Reference: "AUF-1001",
		Items: []PickingListItemInput{
			{ItemID: item.ID, QuantityRequested: 5, QuantityPicked: 6},
		},
	})
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestCreatePickingList_UnknownItemFails(t *testing.T) {
	svc, _, tenantID, _ := newPickingFixture(t)

	_, err := svc.CreatePickingList(context.Background(), CreatePickingListInput{
		TenantID:  tenantID,
		Reference: "AUF-1002",
		Items: []PickingListItemInput{
			{ItemID: uuid.New(), QuantityRequested: 1},
		},
	})
	assert.ErrorIs(t, err, ErrItemNotFound)
}

func TestUpsertPickingListItem_FirstPickAdvancesStatus(t *testing.T) {
	svc, _, tenantID, item := newPickingFixture(t)
	ctx := context.Background()

	list, err := svc.CreatePickingList(ctx, CreatePickingListInput{
		TenantID:  tenantID,
		Reference: "AUF-1003",
		Items:     []PickingListItemInput{{ItemID: item.ID, QuantityRequested: 10}},
	})
	require.NoError(t, err)
	require.Equal(t, PickingStatusOpen, list.Status)

	_, err = svc.UpsertPickingListItem(ctx, UpsertPickingListItemInput{
		TenantID:          tenantID,
		ListID:            list.ID,
		ItemID:            item.ID,
		QuantityRequested: 10,
		QuantityPicked:    4,
	})
	require.NoError(t, err)

	reloaded, err := svc.GetPickingList(ctx, tenantID, list.ID)
	require.NoError(t, err)
	assert.Equal(t, PickingStatusPicking, reloaded.Status)
}

func TestBookPickingList_MovesStockThroughMovementPath(t *testing.T) {
	svc, repo, tenantID, item := newPickingFixture(t)
	ctx := context.Background()

	list, err := svc.CreatePickingList(ctx, CreatePickingListInput{
		TenantID:  tenantID,
		Reference: "AUF-1004",
		Items: []PickingListItemInput{
			{ItemID: item.ID, QuantityRequested: 30, QuantityPicked: 30},
		},
	})
	require.NoError(t, err)

	booked, err := svc.BookPickingList(ctx, BookPickingListInput{TenantID: tenantID, ListID: list.ID})
	require.NoError(t, err)
	assert.Equal(t, PickingStatusCompleted, booked.Status)

	assert.Equal(t, int64(70), repo.items[item.ID].Quantity)
	require.Len(t, repo.movements, 1, "booking must go through the movement path")
	for _, mv := range repo.movements {
		assert.Equal(t, MovementTypeOut, mv.MovementType)
		assert.Equal(t, int64(-30), mv.Quantity)
		assert.Equal(t, "Kommissionierung: AUF-1004", mv.Reason)
	}
}

func TestBookPickingList_SecondBookingDoesNotMoveStockAgain(t *testing.T) {
	svc, repo, tenantID, item := newPickingFixture(t)
	ctx := context.Background()

	list, err := svc.CreatePickingList(ctx, CreatePickingListInput{
		TenantID:  tenantID,
		Reference: "AUF-1005",
		Items: []PickingListItemInput{
			{ItemID: item.ID, QuantityRequested: 20, QuantityPicked: 20},
		},
	})
	require.NoError(t, err)

	_, err = svc.BookPickingList(ctx, BookPickingListInput{TenantID: tenantID, ListID: list.ID})
	require.NoError(t, err)
	require.Equal(t, int64(80), repo.items[item.ID].Quantity)

	_, err = svc.BookPickingList(ctx, BookPickingListInput{TenantID: tenantID, ListID: list.ID})
	assert.ErrorIs(t, err, ErrPickingListAlreadyBooked)
	assert.Equal(t, int64(80), repo.items[item.ID].Quantity, "stock must not move twice")
	assert.Len(t, repo.movements, 1)
}

func TestBookPickingList_InsufficientStockLeavesEverythingUntouched(t *testing.T) {
	svc, repo, tenantID, item := newPickingFixture(t)
	ctx := context.Background()

	second := addItem(repo, tenantID, "Mutter M6", "SKU-MU6", 2, 0)
	list, err := svc.CreatePickingList(ctx, CreatePickingListInput{
		TenantID:  tenantID,
		Reference: "AUF-1006",
		Items: []PickingListItemInput{
			{ItemID: item.ID, QuantityRequested: 10, QuantityPicked: 10},
			{ItemID: second.ID, QuantityRequested: 5, QuantityPicked: 5},
		},
	})
	require.NoError(t, err)

	_, err = svc.BookPickingList(ctx, BookPickingListInput{TenantID: tenantID, ListID: list.ID})
	assert.ErrorIs(t, err, ErrInsufficientStock)

	assert.Equal(t, int64(100), repo.items[item.ID].Quantity, "the bookable position must not move either")
	assert.Equal(t, int64(2), repo.items[second.ID].Quantity)
	assert.Empty(t, repo.movements)

	still, err := svc.GetPickingList(ctx, tenantID, list.ID)
	require.NoError(t, err)
	assert.NotEqual(t, PickingStatusCompleted, still.Status)
}

func TestBookPickingList_SkipsUnpickedPositions(t *testing.T) {
	svc, repo, tenantID, item := newPickingFixture(t)
	ctx := context.Background()

	list, err := svc.CreatePickingList(ctx, CreatePickingListInput{
		TenantID:  tenantID,
		Reference: "AUF-1007",
		Items: []PickingListItemInput{
			{ItemID: item.ID, QuantityRequested: 10, QuantityPicked: 0},
		},
	})
	require.NoError(t, err)

	booked, err := svc.BookPickingList(ctx, BookPickingListInput{TenantID: tenantID, ListID: list.ID})
	require.NoError(t, err)
	assert.Equal(t, PickingStatusCompleted, booked.Status)
	assert.Equal(t, int64(100), repo.items[item.ID].Quantity)
	assert.Empty(t, repo.movements)
}

func TestBookPickingList_UnknownListIsNotFound(t *testing.T) {
	svc, _, tenantID, _ := newPickingFixture(t)

	_, err := svc.BookPickingList(context.Background(), BookPickingListInput{
		TenantID: tenantID,
		ListID:   uuid.New(),
	})
	assert.ErrorIs(t, err, ErrPickingListNotFound)
}

func TestUpdatePickingList_CompletedIsBookingOnly(t *testing.T) {
	svc, _, tenantID, item := newPickingFixture(t)
	ctx := context.Background()

	list, err := svc.CreatePickingList(ctx, CreatePickingListInput{
		TenantID:  tenantID,
		Reference: "AUF-1008",
		Items:     []PickingListItemInput{{ItemID: item.ID, QuantityRequested: 3}},
	})
	require.NoError(t, err)

	completed := PickingStatusCompleted
	_, err = svc.UpdatePickingList(ctx, UpdatePickingListInput{
		TenantID: tenantID,
		ListID:   list.ID,
		Status:   &completed,
	})
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestListPickingLists_FiltersByStatusAndIsTenantScoped(t *testing.T) {
	svc, repo, tenantID, item := newPickingFixture(t)
	ctx := context.Background()

	_, err := svc.CreatePickingList(ctx, CreatePickingListInput{
		TenantID:  tenantID,
		Reference: "AUF-1009",
		Items:     []PickingListItemInput{{ItemID: item.ID, QuantityRequested: 1}},
	})
	require.NoError(t, err)

	otherTenant := uuid.New()
	otherItem := addItem(repo, otherTenant, "Fremd", "SKU-FR", 10, 0)
	_, err = svc.CreatePickingList(ctx, CreatePickingListInput{
		TenantID:  otherTenant,
		Reference: "AUF-2001",
		Items:     []PickingListItemInput{{ItemID: otherItem.ID, QuantityRequested: 1}},
	})
	require.NoError(t, err)

	lists, total, err := svc.ListPickingLists(ctx, ListPickingListsInput{TenantID: tenantID})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, lists, 1)
	assert.Equal(t, "AUF-1009", lists[0].Reference)

	completed := PickingStatusCompleted
	_, total, err = svc.ListPickingLists(ctx, ListPickingListsInput{TenantID: tenantID, Status: &completed})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
}
