package inventar

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// MockRepository
// ============================================================================

type mockRepository struct {
	items       map[uuid.UUID]*Item
	movements   map[uuid.UUID]*Movement
	warnings    map[uuid.UUID]*Warning
	attachments []*ItemAttachment

	// sku -> itemID for uniqueness checks (tenant-scoped)
	skus map[string]uuid.UUID

	createItemErr    error
	updateItemErr    error
	getItemErr       error
	createWarningErr error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		items:     make(map[uuid.UUID]*Item),
		movements: make(map[uuid.UUID]*Movement),
		warnings:  make(map[uuid.UUID]*Warning),
		skus:      make(map[string]uuid.UUID),
	}
}

func (m *mockRepository) CreateItem(ctx context.Context, item *Item) error {
	if m.createItemErr != nil {
		return m.createItemErr
	}
	m.items[item.ID] = item
	key := item.TenantID.String() + ":" + item.SKU
	m.skus[key] = item.ID
	return nil
}

func (m *mockRepository) UpdateItem(ctx context.Context, item *Item) error {
	if m.updateItemErr != nil {
		return m.updateItemErr
	}
	m.items[item.ID] = item
	return nil
}

func (m *mockRepository) SoftDeleteItem(ctx context.Context, tenantID, itemID uuid.UUID) error {
	item, ok := m.items[itemID]
	if !ok {
		return ErrItemNotFound
	}
	now := time.Now()
	item.DeletedAt = &now
	m.items[itemID] = item
	return nil
}

func (m *mockRepository) GetItem(ctx context.Context, tenantID, itemID uuid.UUID) (*Item, error) {
	if m.getItemErr != nil {
		return nil, m.getItemErr
	}
	item, ok := m.items[itemID]
	if !ok || item.TenantID != tenantID || item.DeletedAt != nil {
		return nil, ErrItemNotFound
	}
	return item, nil
}

func (m *mockRepository) ListItems(ctx context.Context, tenantID uuid.UUID, filter ListItemsFilter, offset, limit int) ([]*Item, int, error) {
	var result []*Item
	for _, item := range m.items {
		if item.TenantID != tenantID || item.DeletedAt != nil {
			continue
		}
		if filter.LowStock && item.Quantity > item.MinQuantity {
			continue
		}
		result = append(result, item)
	}
	total := len(result)
	if offset >= total {
		return []*Item{}, total, nil
	}
	end := min(offset+limit, total)
	return result[offset:end], total, nil
}

func (m *mockRepository) SKUExists(ctx context.Context, tenantID uuid.UUID, sku string, excludeID *uuid.UUID) (bool, error) {
	key := tenantID.String() + ":" + sku
	existingID, ok := m.skus[key]
	if !ok {
		return false, nil
	}
	if excludeID != nil && existingID == *excludeID {
		return false, nil
	}
	return true, nil
}

func (m *mockRepository) CreateMovement(ctx context.Context, movement *Movement) error {
	m.movements[movement.ID] = movement
	return nil
}

func (m *mockRepository) GetMovement(ctx context.Context, tenantID, movementID uuid.UUID) (*Movement, error) {
	mv, ok := m.movements[movementID]
	if !ok {
		return nil, ErrMovementNotFound
	}
	if mv.TenantID != tenantID {
		return nil, ErrMovementNotFound
	}
	return mv, nil
}

func (m *mockRepository) ListMovements(ctx context.Context, tenantID, itemID uuid.UUID, offset, limit int) ([]*Movement, int, error) {
	var result []*Movement
	for _, mv := range m.movements {
		if mv.TenantID == tenantID && mv.ItemID == itemID {
			result = append(result, mv)
		}
	}
	total := len(result)
	if offset >= total {
		return []*Movement{}, total, nil
	}
	end := min(offset+limit, total)
	return result[offset:end], total, nil
}

func (m *mockRepository) CreateWarning(ctx context.Context, warning *Warning) error {
	if m.createWarningErr != nil {
		return m.createWarningErr
	}
	m.warnings[warning.ID] = warning
	return nil
}

func (m *mockRepository) UpdateWarning(ctx context.Context, warning *Warning) error {
	m.warnings[warning.ID] = warning
	return nil
}

func (m *mockRepository) GetWarning(ctx context.Context, tenantID, warningID uuid.UUID) (*Warning, error) {
	w, ok := m.warnings[warningID]
	if !ok || w.TenantID != tenantID {
		return nil, ErrWarningNotFound
	}
	return w, nil
}

func (m *mockRepository) GetActiveWarningForItem(ctx context.Context, tenantID, itemID uuid.UUID) (*Warning, error) {
	for _, w := range m.warnings {
		if w.TenantID == tenantID && w.ItemID == itemID && w.Status == WarningStatusActive {
			return w, nil
		}
	}
	return nil, ErrWarningNotFound
}

func (m *mockRepository) ListWarnings(ctx context.Context, tenantID uuid.UUID, status *WarningStatus, offset, limit int) ([]*Warning, int, error) {
	var result []*Warning
	for _, w := range m.warnings {
		if w.TenantID != tenantID {
			continue
		}
		if status != nil && w.Status != *status {
			continue
		}
		result = append(result, w)
	}
	total := len(result)
	if offset >= total {
		return []*Warning{}, total, nil
	}
	end := min(offset+limit, total)
	return result[offset:end], total, nil
}

// ============================================================================
// Location mock methods
// ============================================================================

func (m *mockRepository) CreateLocation(_ context.Context, loc *InventoryLocation) error {
	return nil
}

func (m *mockRepository) UpdateLocation(_ context.Context, loc *InventoryLocation) error {
	return nil
}

func (m *mockRepository) SoftDeleteLocation(_ context.Context, tenantID, locID uuid.UUID) error {
	return nil
}

func (m *mockRepository) GetLocation(_ context.Context, tenantID, locID uuid.UUID) (*InventoryLocation, error) {
	return nil, ErrLocationNotFound
}

func (m *mockRepository) ListLocations(_ context.Context, tenantID uuid.UUID, offset, limit int) ([]*InventoryLocation, int, error) {
	return nil, 0, nil
}

// ============================================================================
// Inventur Session mock methods
// ============================================================================

func (m *mockRepository) CreateInventurSession(_ context.Context, session *InventurSession) error {
	return nil
}

func (m *mockRepository) UpdateInventurSession(_ context.Context, session *InventurSession) error {
	return nil
}

func (m *mockRepository) DeleteInventurSession(_ context.Context, tenantID, sessionID uuid.UUID) error {
	return nil
}

func (m *mockRepository) GetInventurSession(_ context.Context, tenantID, sessionID uuid.UUID) (*InventurSession, error) {
	return nil, ErrInventurSessionNotFound
}

func (m *mockRepository) ListInventurSessions(_ context.Context, tenantID uuid.UUID, offset, limit int) ([]*InventurSession, int, error) {
	return nil, 0, nil
}

func (m *mockRepository) UpsertInventurCount(_ context.Context, count *InventurCount) error {
	return nil
}

func (m *mockRepository) ListInventurCounts(_ context.Context, tenantID, sessionID uuid.UUID) ([]*InventurCount, error) {
	return nil, nil
}

// compile-time interface check
func (m *mockRepository) CreateItemAttachment(ctx context.Context, att *ItemAttachment) error {
	m.attachments = append(m.attachments, att)
	return nil
}

func (m *mockRepository) ListItemAttachments(ctx context.Context, tenantID, itemID uuid.UUID) ([]*ItemAttachment, error) {
	var out []*ItemAttachment
	for _, a := range m.attachments {
		if a.TenantID == tenantID && a.ItemID == itemID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (m *mockRepository) DeleteItemAttachment(ctx context.Context, tenantID, attachmentID uuid.UUID) error {
	for i, a := range m.attachments {
		if a.ID == attachmentID && a.TenantID == tenantID {
			m.attachments = append(m.attachments[:i], m.attachments[i+1:]...)
			return nil
		}
	}
	return ErrAttachmentNotFound
}

var _ Repository = (*mockRepository)(nil)

// ============================================================================
// Helpers
// ============================================================================

func addItem(repo *mockRepository, tenantID uuid.UUID, name, sku string, qty, minQty int64) *Item {
	item := &Item{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        name,
		SKU:         sku,
		Quantity:    qty,
		MinQuantity: minQty,
		Unit:        "Stk",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.items[item.ID] = item
	key := tenantID.String() + ":" + sku
	repo.skus[key] = item.ID
	return item
}

// ============================================================================
// CreateItem Tests
// ============================================================================

func TestService_CreateItem_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	item, err := svc.CreateItem(context.Background(), CreateItemInput{
		TenantID:    tenantID,
		Name:        "Schrauben M8",
		SKU:         "SCR-M8-001",
		Quantity:    100,
		MinQuantity: 10,
		Unit:        "Stk",
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, item.ID)
	assert.Equal(t, "Schrauben M8", item.Name)
	assert.Equal(t, "SCR-M8-001", item.SKU)
	assert.Equal(t, int64(100), item.Quantity)
	assert.Equal(t, tenantID, item.TenantID)
}

func TestService_CreateItem_EmptyName(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateItem(context.Background(), CreateItemInput{
		TenantID:    uuid.New(),
		Name:        "  ",
		SKU:         "SCR-001",
		MinQuantity: 5,
		Unit:        "Stk",
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_CreateItem_EmptySKU(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateItem(context.Background(), CreateItemInput{
		TenantID:    uuid.New(),
		Name:        "Produkt",
		SKU:         "",
		MinQuantity: 5,
		Unit:        "Stk",
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_CreateItem_NegativeMinQuantity(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateItem(context.Background(), CreateItemInput{
		TenantID:    uuid.New(),
		Name:        "Produkt",
		SKU:         "P-001",
		MinQuantity: -1,
		Unit:        "Stk",
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_CreateItem_SKUTaken(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	addItem(repo, tenantID, "Existing", "SCR-001", 50, 5)

	_, err := svc.CreateItem(context.Background(), CreateItemInput{
		TenantID: tenantID,
		Name:     "Duplicate",
		SKU:      "SCR-001",
		Unit:     "Stk",
	})

	assert.ErrorIs(t, err, ErrSKUTaken)
}

func TestService_CreateItem_RepoError(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)
	repo.createItemErr = ErrInvalidInput

	_, err := svc.CreateItem(context.Background(), CreateItemInput{
		TenantID: uuid.New(),
		Name:     "Produkt",
		SKU:      "P-001",
		Unit:     "Stk",
	})

	assert.Error(t, err)
}

// ============================================================================
// UpdateItem Tests
// ============================================================================

func TestService_UpdateItem_UpdatesName(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	item := addItem(repo, tenantID, "Alt", "P-001", 50, 5)

	newName := "Neu"
	updated, err := svc.UpdateItem(context.Background(), UpdateItemInput{
		TenantID: tenantID,
		ItemID:   item.ID,
		Name:     &newName,
	})

	require.NoError(t, err)
	assert.Equal(t, "Neu", updated.Name)
}

func TestService_UpdateItem_SKUTaken(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	addItem(repo, tenantID, "Existing", "P-002", 20, 2)
	item := addItem(repo, tenantID, "Mine", "P-001", 10, 1)

	takenSKU := "P-002"
	_, err := svc.UpdateItem(context.Background(), UpdateItemInput{
		TenantID: tenantID,
		ItemID:   item.ID,
		SKU:      &takenSKU,
	})

	assert.ErrorIs(t, err, ErrSKUTaken)
}

func TestService_UpdateItem_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.UpdateItem(context.Background(), UpdateItemInput{
		TenantID: uuid.New(),
		ItemID:   uuid.New(),
	})

	assert.ErrorIs(t, err, ErrItemNotFound)
}

// ============================================================================
// DeleteItem Tests
// ============================================================================

func TestService_DeleteItem_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	item := addItem(repo, tenantID, "Produkt", "P-001", 10, 1)

	err := svc.DeleteItem(context.Background(), tenantID, item.ID)

	require.NoError(t, err)
	// Soft-delete: deleted_at should be set
	assert.NotNil(t, repo.items[item.ID].DeletedAt)
}

func TestService_DeleteItem_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	err := svc.DeleteItem(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, ErrItemNotFound)
}

// ============================================================================
// GetItem / ListItems Tests
// ============================================================================

func TestService_GetItem_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	item := addItem(repo, tenantID, "Produkt", "P-001", 5, 1)

	result, err := svc.GetItem(context.Background(), tenantID, item.ID)

	require.NoError(t, err)
	assert.Equal(t, item.ID, result.ID)
}

func TestService_GetItem_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.GetItem(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, ErrItemNotFound)
}

func TestService_ListItems_DefaultPagination(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	for i := range 60 {
		sku := uuid.New().String()[:8] + string(rune('A'+i%26))
		addItem(repo, tenantID, "Item", sku, 10, 1)
	}

	items, total, err := svc.ListItems(context.Background(), ListItemsInput{
		TenantID: tenantID,
	})

	require.NoError(t, err)
	assert.Equal(t, 60, total)
	assert.Len(t, items, 50)
}

func TestService_ListItems_LowStockFilter(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	addItem(repo, tenantID, "OK", "P-001", 50, 10)
	addItem(repo, tenantID, "Low", "P-002", 3, 10)
	addItem(repo, tenantID, "Zero", "P-003", 0, 5)

	items, total, err := svc.ListItems(context.Background(), ListItemsInput{
		TenantID: tenantID,
		LowStock: true,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, items, 2)
}

// ============================================================================
// AdjustStock Tests
// ============================================================================

func TestService_AdjustStock_Positive(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	item := addItem(repo, tenantID, "Produkt", "P-001", 10, 5)
	performer := uuid.New()

	updated, err := svc.AdjustStock(context.Background(), AdjustStockInput{
		TenantID:    tenantID,
		ItemID:      item.ID,
		Delta:       20,
		PerformedBy: &performer,
		Reason:      "Lieferung eingegangen",
	})

	require.NoError(t, err)
	assert.Equal(t, int64(30), updated.Quantity)
	// One movement should be recorded
	assert.Len(t, repo.movements, 1)
}

func TestService_AdjustStock_Negative(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	item := addItem(repo, tenantID, "Produkt", "P-001", 20, 5)

	updated, err := svc.AdjustStock(context.Background(), AdjustStockInput{
		TenantID: tenantID,
		ItemID:   item.ID,
		Delta:    -5,
		Reason:   "Verbrauch",
	})

	require.NoError(t, err)
	assert.Equal(t, int64(15), updated.Quantity)
}

func TestService_AdjustStock_ZeroDelta(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	item := addItem(repo, tenantID, "Produkt", "P-001", 10, 5)

	_, err := svc.AdjustStock(context.Background(), AdjustStockInput{
		TenantID: tenantID,
		ItemID:   item.ID,
		Delta:    0,
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_AdjustStock_ItemNotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.AdjustStock(context.Background(), AdjustStockInput{
		TenantID: uuid.New(),
		ItemID:   uuid.New(),
		Delta:    10,
	})

	assert.ErrorIs(t, err, ErrItemNotFound)
}

func TestService_AdjustStock_TriggersWarning(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	// quantity will drop below min_quantity
	item := addItem(repo, tenantID, "Produkt", "P-001", 10, 8)

	_, err := svc.AdjustStock(context.Background(), AdjustStockInput{
		TenantID: tenantID,
		ItemID:   item.ID,
		Delta:    -5, // new qty = 5, min_qty = 8 → warning
		Reason:   "Entnahme",
	})

	require.NoError(t, err)
	// An automatic warning should have been created
	assert.Len(t, repo.warnings, 1)
	for _, w := range repo.warnings {
		assert.Equal(t, WarningStatusActive, w.Status)
		assert.Equal(t, item.ID, w.ItemID)
	}
}

func TestService_AdjustStock_WarningIdempotent(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	item := addItem(repo, tenantID, "Produkt", "P-001", 6, 10)

	// First adjustment — creates warning
	_, err := svc.AdjustStock(context.Background(), AdjustStockInput{
		TenantID: tenantID,
		ItemID:   item.ID,
		Delta:    -1,
		Reason:   "Erste Entnahme",
	})
	require.NoError(t, err)

	// Second adjustment — must NOT create a duplicate warning
	_, err = svc.AdjustStock(context.Background(), AdjustStockInput{
		TenantID: tenantID,
		ItemID:   item.ID,
		Delta:    -1,
		Reason:   "Zweite Entnahme",
	})
	require.NoError(t, err)

	// Still exactly one warning
	assert.Len(t, repo.warnings, 1)
}

// ============================================================================
// TransferStock Tests
// ============================================================================

func TestService_TransferStock_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	fromItem := addItem(repo, tenantID, "Quelle", "SRC-001", 30, 2)
	toItem := addItem(repo, tenantID, "Ziel", "DST-001", 10, 0)

	err := svc.TransferStock(context.Background(), TransferStockInput{
		TenantID:   tenantID,
		FromItemID: fromItem.ID,
		ToItemID:   toItem.ID,
		Quantity:   10,
		Reason:     "Umlagerung",
	})

	require.NoError(t, err)
	assert.Equal(t, int64(20), repo.items[fromItem.ID].Quantity)
	assert.Equal(t, int64(20), repo.items[toItem.ID].Quantity)
	// Two movements recorded (transfer-out + transfer-in)
	assert.Len(t, repo.movements, 2)
}

func TestService_TransferStock_ZeroQuantity(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()

	err := svc.TransferStock(context.Background(), TransferStockInput{
		TenantID:   tenantID,
		FromItemID: uuid.New(),
		ToItemID:   uuid.New(),
		Quantity:   0,
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

// ============================================================================
// RecordMovement Tests
// ============================================================================

func TestService_RecordMovement_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	item := addItem(repo, tenantID, "Produkt", "P-001", 100, 10)

	mv, err := svc.RecordMovement(context.Background(), RecordMovementInput{
		TenantID:     tenantID,
		ItemID:       item.ID,
		MovementType: MovementTypeIn,
		Quantity:     50,
		Reason:       "Batch Import",
	})

	require.NoError(t, err)
	assert.Equal(t, MovementTypeIn, mv.MovementType)
	assert.Equal(t, int64(50), mv.Quantity)
}

func TestService_RecordMovement_ZeroQuantity(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	item := addItem(repo, tenantID, "Produkt", "P-001", 10, 1)

	_, err := svc.RecordMovement(context.Background(), RecordMovementInput{
		TenantID:     tenantID,
		ItemID:       item.ID,
		MovementType: MovementTypeIn,
		Quantity:     0,
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

// ============================================================================
// Warning Tests
// ============================================================================

func TestService_CreateWarning_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	item := addItem(repo, tenantID, "Produkt", "P-001", 3, 10)

	w, err := svc.CreateWarning(context.Background(), CreateWarningInput{
		TenantID:        tenantID,
		ItemID:          item.ID,
		Threshold:       10,
		CurrentQuantity: 3,
	})

	require.NoError(t, err)
	assert.Equal(t, WarningStatusActive, w.Status)
	assert.Equal(t, item.ID, w.ItemID)
}

func TestService_CreateWarning_ItemNotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateWarning(context.Background(), CreateWarningInput{
		TenantID:        uuid.New(),
		ItemID:          uuid.New(),
		Threshold:       5,
		CurrentQuantity: 2,
	})

	assert.ErrorIs(t, err, ErrItemNotFound)
}

func TestService_AcknowledgeWarning_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	item := addItem(repo, tenantID, "Produkt", "P-001", 2, 10)

	// Pre-create a warning directly in the repo
	warnID := uuid.New()
	repo.warnings[warnID] = &Warning{
		ID:              warnID,
		TenantID:        tenantID,
		ItemID:          item.ID,
		Threshold:       10,
		CurrentQuantity: 2,
		Status:          WarningStatusActive,
		CreatedAt:       time.Now(),
	}

	userID := uuid.New()
	w, err := svc.AcknowledgeWarning(context.Background(), AcknowledgeWarningInput{
		TenantID:       tenantID,
		WarningID:      warnID,
		AcknowledgedBy: userID,
	})

	require.NoError(t, err)
	assert.Equal(t, WarningStatusAcknowledged, w.Status)
	assert.NotNil(t, w.AcknowledgedAt)
	assert.Equal(t, userID, *w.AcknowledgedBy)
}

func TestService_AcknowledgeWarning_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.AcknowledgeWarning(context.Background(), AcknowledgeWarningInput{
		TenantID:       uuid.New(),
		WarningID:      uuid.New(),
		AcknowledgedBy: uuid.New(),
	})

	assert.ErrorIs(t, err, ErrWarningNotFound)
}

func TestService_ListWarnings_FilterByStatus(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()

	for range 3 {
		repo.warnings[uuid.New()] = &Warning{
			ID:       uuid.New(),
			TenantID: tenantID,
			Status:   WarningStatusActive,
		}
	}
	repo.warnings[uuid.New()] = &Warning{
		ID:       uuid.New(),
		TenantID: tenantID,
		Status:   WarningStatusResolved,
	}

	activeStatus := WarningStatusActive
	warnings, total, err := svc.ListWarnings(context.Background(), ListWarningsInput{
		TenantID: tenantID,
		Status:   &activeStatus,
	})

	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, warnings, 3)
}

// ============================================================================
// GetStockReport Tests
// ============================================================================

func TestService_GetStockReport_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	addItem(repo, tenantID, "OK-Item", "P-001", 50, 10)
	addItem(repo, tenantID, "Low-Item", "P-002", 3, 10)

	// Active warning
	repo.warnings[uuid.New()] = &Warning{
		ID:       uuid.New(),
		TenantID: tenantID,
		Status:   WarningStatusActive,
	}

	report, err := svc.GetStockReport(context.Background(), tenantID)

	require.NoError(t, err)
	assert.Equal(t, 2, report.TotalItems)
	assert.Equal(t, int64(53), report.TotalQuantity)
	assert.Equal(t, 1, report.LowStockCount)
	assert.Equal(t, 1, report.ActiveWarnings)
}

// ============================================================================
// ListMovements Tests
// ============================================================================

func TestService_ListMovements_Pagination(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	item := addItem(repo, tenantID, "Produkt", "P-001", 100, 5)

	for range 15 {
		repo.movements[uuid.New()] = &Movement{
			ID:           uuid.New(),
			TenantID:     tenantID,
			ItemID:       item.ID,
			MovementType: MovementTypeIn,
			Quantity:     1,
			CreatedAt:    time.Now(),
		}
	}

	mvs, total, err := svc.ListMovements(context.Background(), ListMovementsInput{
		TenantID: tenantID,
		ItemID:   item.ID,
		Page:     1,
		PageSize: 10,
	})

	require.NoError(t, err)
	assert.Equal(t, 15, total)
	assert.Len(t, mvs, 10)
}

// ============================================================================
// Oversell / Insufficient-Stock Tests
// ============================================================================

func TestService_AdjustStock_OversellRejected(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	item := addItem(repo, tenantID, "Produkt", "P-001", 10, 0)

	_, err := svc.AdjustStock(context.Background(), AdjustStockInput{
		TenantID: tenantID,
		ItemID:   item.ID,
		Delta:    -100,
		Reason:   "Oversell attempt",
	})

	assert.ErrorIs(t, err, ErrInsufficientStock)
	// Item quantity must be unchanged
	assert.Equal(t, int64(10), repo.items[item.ID].Quantity)
	// No movement must have been recorded
	assert.Len(t, repo.movements, 0)
}

func TestService_TransferStock_InsufficientSource(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	fromItem := addItem(repo, tenantID, "Quelle", "SRC-001", 5, 0)
	toItem := addItem(repo, tenantID, "Ziel", "DST-001", 0, 0)

	err := svc.TransferStock(context.Background(), TransferStockInput{
		TenantID:   tenantID,
		FromItemID: fromItem.ID,
		ToItemID:   toItem.ID,
		Quantity:   10,
		Reason:     "Oversell transfer attempt",
	})

	assert.ErrorIs(t, err, ErrInsufficientStock)
	// From-item quantity must be unchanged
	assert.Equal(t, int64(5), repo.items[fromItem.ID].Quantity)
	// To-item quantity must be unchanged
	assert.Equal(t, int64(0), repo.items[toItem.ID].Quantity)
	// No movements must have been recorded
	assert.Len(t, repo.movements, 0)
}
