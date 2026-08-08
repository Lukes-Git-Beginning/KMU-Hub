package produktion

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeInventarLookup is a configurable InventarLookup test double. It also
// records how many times ResolveByNames was called, so tests can assert the
// batch-call requirement (one call regardless of BOM position count).
type fakeInventarLookup struct {
	stock     map[string]InventarStock // keyed by normalized material name
	err       error
	callCount int
	lastNames []string
}

func (f *fakeInventarLookup) ResolveByNames(_ context.Context, _ uuid.UUID, materialNames []string) (map[string]InventarStock, error) {
	f.callCount++
	f.lastNames = materialNames
	if f.err != nil {
		return nil, f.err
	}
	return f.stock, nil
}

func addBOM(repo *mockRepository, tenantID uuid.UUID, items []*BomItem) *BOM {
	bom := &BOM{
		ID:          uuid.New(),
		TenantID:    tenantID,
		ProductName: "Widget",
		SKU:         "SKU-" + uuid.NewString(),
		Version:     "1.0",
		Active:      true,
		Items:       items,
	}
	repo.boms[bom.ID] = bom
	return bom
}

func TestService_GetMaterialAvailability_Success(t *testing.T) {
	repo := newMockRepository()
	tenantID := uuid.New()
	bom := addBOM(repo, tenantID, []*BomItem{
		{ID: uuid.New(), MaterialName: "Steel Bolt M6", Quantity: 4, Unit: "pcs"},
		{ID: uuid.New(), MaterialName: "Untracked Widget", Quantity: 2, Unit: "pcs"},
	})
	order := addOrder(repo, tenantID, "PO-1", OrderStatusPlanned)
	order.Quantity = 5
	order.BomID = &bom.ID

	lookup := &fakeInventarLookup{
		stock: map[string]InventarStock{
			normalizeMaterialName("Steel Bolt M6"): {Quantity: 15, Unit: "pcs"},
		},
	}
	svc := NewService(repo).WithInventarLookup(lookup)

	result, err := svc.GetMaterialAvailability(context.Background(), tenantID, order.ID)
	require.NoError(t, err)
	require.Len(t, result.Lines, 2)

	// 4 required per unit * 5 units = 20 required, 15 available -> shortfall 5.
	bolt := result.Lines[0]
	assert.Equal(t, "Steel Bolt M6", bolt.MaterialName)
	assert.Equal(t, 20.0, bolt.RequiredQuantity)
	require.NotNil(t, bolt.AvailableQuantity)
	assert.Equal(t, 15.0, *bolt.AvailableQuantity)
	require.NotNil(t, bolt.ShortfallQuantity)
	assert.Equal(t, 5.0, *bolt.ShortfallQuantity)

	// Unmatched material -> unknown availability, not an error and not zero.
	unknown := result.Lines[1]
	assert.Equal(t, "Untracked Widget", unknown.MaterialName)
	assert.Nil(t, unknown.AvailableQuantity)
	assert.Nil(t, unknown.ShortfallQuantity)

	assert.Equal(t, 1, lookup.callCount, "must resolve all BOM positions in a single batch call")
}

func TestService_GetMaterialAvailability_SufficientStockHasZeroShortfall(t *testing.T) {
	repo := newMockRepository()
	tenantID := uuid.New()
	bom := addBOM(repo, tenantID, []*BomItem{
		{ID: uuid.New(), MaterialName: "Steel Bolt M6", Quantity: 2, Unit: "pcs"},
	})
	order := addOrder(repo, tenantID, "PO-1", OrderStatusPlanned)
	order.Quantity = 3
	order.BomID = &bom.ID

	lookup := &fakeInventarLookup{
		stock: map[string]InventarStock{
			normalizeMaterialName("Steel Bolt M6"): {Quantity: 100, Unit: "pcs"},
		},
	}
	svc := NewService(repo).WithInventarLookup(lookup)

	result, err := svc.GetMaterialAvailability(context.Background(), tenantID, order.ID)
	require.NoError(t, err)
	require.Len(t, result.Lines, 1)
	require.NotNil(t, result.Lines[0].ShortfallQuantity)
	assert.Equal(t, 0.0, *result.Lines[0].ShortfallQuantity)
}

func TestService_GetMaterialAvailability_OrderWithoutBOM(t *testing.T) {
	repo := newMockRepository()
	tenantID := uuid.New()
	order := addOrder(repo, tenantID, "PO-1", OrderStatusPlanned)

	svc := NewService(repo)
	_, err := svc.GetMaterialAvailability(context.Background(), tenantID, order.ID)
	assert.ErrorIs(t, err, ErrOrderHasNoBOM)
}

func TestService_GetMaterialAvailability_OrderNotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	_, err := svc.GetMaterialAvailability(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrOrderNotFound)
}

func TestService_GetMaterialAvailability_CrossTenantOrderNotFound(t *testing.T) {
	repo := newMockRepository()
	tenantID := uuid.New()
	order := addOrder(repo, tenantID, "PO-1", OrderStatusPlanned)

	svc := NewService(repo)
	_, err := svc.GetMaterialAvailability(context.Background(), uuid.New(), order.ID)
	assert.ErrorIs(t, err, ErrOrderNotFound)
}

func TestService_GetMaterialAvailability_NoLookupConfigured(t *testing.T) {
	repo := newMockRepository()
	tenantID := uuid.New()
	bom := addBOM(repo, tenantID, []*BomItem{
		{ID: uuid.New(), MaterialName: "Steel Bolt M6", Quantity: 4, Unit: "pcs"},
	})
	order := addOrder(repo, tenantID, "PO-1", OrderStatusPlanned)
	order.BomID = &bom.ID

	// No WithInventarLookup call -- every position must degrade to unknown,
	// not fail the whole request.
	svc := NewService(repo)
	result, err := svc.GetMaterialAvailability(context.Background(), tenantID, order.ID)
	require.NoError(t, err)
	require.Len(t, result.Lines, 1)
	assert.Nil(t, result.Lines[0].AvailableQuantity)
	assert.Nil(t, result.Lines[0].ShortfallQuantity)
}

func TestService_GetMaterialAvailability_LookupFailureDegradesToUnknown(t *testing.T) {
	repo := newMockRepository()
	tenantID := uuid.New()
	bom := addBOM(repo, tenantID, []*BomItem{
		{ID: uuid.New(), MaterialName: "Steel Bolt M6", Quantity: 4, Unit: "pcs"},
	})
	order := addOrder(repo, tenantID, "PO-1", OrderStatusPlanned)
	order.BomID = &bom.ID

	lookup := &fakeInventarLookup{err: assert.AnError}
	svc := NewService(repo).WithInventarLookup(lookup)

	result, err := svc.GetMaterialAvailability(context.Background(), tenantID, order.ID)
	require.NoError(t, err, "an unreachable inventar lookup must not fail the whole check")
	require.Len(t, result.Lines, 1)
	assert.Nil(t, result.Lines[0].AvailableQuantity)
}

func TestService_CreateOrder_WithUnknownBomID(t *testing.T) {
	repo := newMockRepository()
	tenantID := uuid.New()

	unknownBomID := uuid.New()
	svc := NewService(repo)
	_, err := svc.CreateOrder(context.Background(), CreateOrderInput{
		TenantID:     tenantID,
		OrderNumber:  "PO-1",
		ProductName:  "Widget",
		Quantity:     1,
		PlannedStart: t0,
		PlannedEnd:   t1,
		BomID:        &unknownBomID,
	})
	assert.ErrorIs(t, err, ErrBOMNotFound)
}

func TestService_UpdateOrder_LinksBOM(t *testing.T) {
	repo := newMockRepository()
	tenantID := uuid.New()
	bom := addBOM(repo, tenantID, nil)
	order := addOrder(repo, tenantID, "PO-1", OrderStatusPlanned)
	require.Nil(t, order.BomID)

	svc := NewService(repo)
	updated, err := svc.UpdateOrder(context.Background(), UpdateOrderInput{
		TenantID: tenantID,
		OrderID:  order.ID,
		BomID:    &bom.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, updated.BomID)
	assert.Equal(t, bom.ID, *updated.BomID)
}
