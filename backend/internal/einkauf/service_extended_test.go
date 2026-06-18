package einkauf

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// mockRepositoryExtended
// ============================================================================
// Embeds mockRepository (base Repository) and adds all RepositoryExtended
// methods. Also implements contractItemQuerier so that getContractItemByID
// (used by Service.UpdateContractItem) resolves via type assertion.

type mockRepositoryExtended struct {
	*mockRepository

	catalogItems    map[uuid.UUID]*CatalogItem
	supplierRatings map[uuid.UUID]*SupplierRating
	contracts       map[uuid.UUID]*FrameworkContract
	contractItems   map[uuid.UUID]*FrameworkContractItem
	contractCalls   map[uuid.UUID]*FrameworkContractCall
	contractNrs     map[string]uuid.UUID // tenantID+":"+contractNr -> contractID

	createContractErr error
}

func newMockRepositoryExtended() *mockRepositoryExtended {
	return &mockRepositoryExtended{
		mockRepository:  newMockRepository(),
		catalogItems:    make(map[uuid.UUID]*CatalogItem),
		supplierRatings: make(map[uuid.UUID]*SupplierRating),
		contracts:       make(map[uuid.UUID]*FrameworkContract),
		contractItems:   make(map[uuid.UUID]*FrameworkContractItem),
		contractCalls:   make(map[uuid.UUID]*FrameworkContractCall),
		contractNrs:     make(map[string]uuid.UUID),
	}
}

// compile-time check
var _ RepositoryExtended = (*mockRepositoryExtended)(nil)

// -------------------------------------------------------------------------
// Catalog Items
// -------------------------------------------------------------------------

func (m *mockRepositoryExtended) CreateCatalogItem(_ context.Context, item *CatalogItem) error {
	m.catalogItems[item.ID] = item
	return nil
}

func (m *mockRepositoryExtended) UpdateCatalogItem(_ context.Context, item *CatalogItem) error {
	m.catalogItems[item.ID] = item
	return nil
}

func (m *mockRepositoryExtended) DeleteCatalogItem(_ context.Context, _, itemID uuid.UUID) error {
	delete(m.catalogItems, itemID)
	return nil
}

func (m *mockRepositoryExtended) GetCatalogItem(_ context.Context, tenantID, itemID uuid.UUID) (*CatalogItem, error) {
	item, ok := m.catalogItems[itemID]
	if !ok || item.TenantID != tenantID {
		return nil, ErrCatalogItemNotFound
	}
	return item, nil
}

func (m *mockRepositoryExtended) ListCatalogItems(_ context.Context, tenantID uuid.UUID, filter ListCatalogItemsFilter, offset, limit int) ([]*CatalogItem, int, error) {
	var result []*CatalogItem
	for _, item := range m.catalogItems {
		if item.TenantID != tenantID {
			continue
		}
		if filter.SupplierID != nil && item.SupplierID != *filter.SupplierID {
			continue
		}
		if filter.Category != "" && item.Category != filter.Category {
			continue
		}
		result = append(result, item)
	}
	total := len(result)
	if offset >= total {
		return []*CatalogItem{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return result[offset:end], total, nil
}

// -------------------------------------------------------------------------
// Supplier Ratings
// -------------------------------------------------------------------------

func (m *mockRepositoryExtended) CreateSupplierRating(_ context.Context, rating *SupplierRating) error {
	m.supplierRatings[rating.ID] = rating
	return nil
}

func (m *mockRepositoryExtended) DeleteSupplierRating(_ context.Context, _, ratingID uuid.UUID) error {
	delete(m.supplierRatings, ratingID)
	return nil
}

func (m *mockRepositoryExtended) GetSupplierRating(_ context.Context, tenantID, ratingID uuid.UUID) (*SupplierRating, error) {
	r, ok := m.supplierRatings[ratingID]
	if !ok || r.TenantID != tenantID {
		return nil, ErrSupplierRatingNotFound
	}
	return r, nil
}

func (m *mockRepositoryExtended) ListSupplierRatings(_ context.Context, tenantID, supplierID uuid.UUID) ([]*SupplierRating, error) {
	var result []*SupplierRating
	for _, r := range m.supplierRatings {
		if r.TenantID == tenantID && r.SupplierID == supplierID {
			result = append(result, r)
		}
	}
	return result, nil
}

// -------------------------------------------------------------------------
// Framework Contracts
// -------------------------------------------------------------------------

func (m *mockRepositoryExtended) CreateFrameworkContract(_ context.Context, fc *FrameworkContract) error {
	if m.createContractErr != nil {
		return m.createContractErr
	}
	m.contracts[fc.ID] = fc
	if fc.ContractNr != "" {
		key := fc.TenantID.String() + ":" + fc.ContractNr
		m.contractNrs[key] = fc.ID
	}
	return nil
}

func (m *mockRepositoryExtended) UpdateFrameworkContract(_ context.Context, fc *FrameworkContract) error {
	m.contracts[fc.ID] = fc
	return nil
}

func (m *mockRepositoryExtended) DeleteFrameworkContract(_ context.Context, _, contractID uuid.UUID) error {
	delete(m.contracts, contractID)
	return nil
}

func (m *mockRepositoryExtended) GetFrameworkContract(_ context.Context, tenantID, contractID uuid.UUID) (*FrameworkContract, error) {
	fc, ok := m.contracts[contractID]
	if !ok || fc.TenantID != tenantID {
		return nil, ErrContractNotFound
	}
	return fc, nil
}

func (m *mockRepositoryExtended) GetFrameworkContractWithItems(ctx context.Context, tenantID, contractID uuid.UUID) (*FrameworkContract, error) {
	fc, err := m.GetFrameworkContract(ctx, tenantID, contractID)
	if err != nil {
		return nil, err
	}
	items, _ := m.ListContractItems(ctx, tenantID, contractID)
	fc.Items = items
	return fc, nil
}

func (m *mockRepositoryExtended) ListFrameworkContracts(_ context.Context, tenantID uuid.UUID, filter ListContractsFilter, offset, limit int) ([]*FrameworkContract, int, error) {
	var result []*FrameworkContract
	for _, fc := range m.contracts {
		if fc.TenantID != tenantID {
			continue
		}
		if filter.SupplierID != nil && fc.SupplierID != *filter.SupplierID {
			continue
		}
		if filter.Status != nil && fc.Status != *filter.Status {
			continue
		}
		result = append(result, fc)
	}
	total := len(result)
	if offset >= total {
		return []*FrameworkContract{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return result[offset:end], total, nil
}

func (m *mockRepositoryExtended) ContractNrExists(_ context.Context, tenantID uuid.UUID, contractNr string, excludeID *uuid.UUID) (bool, error) {
	key := tenantID.String() + ":" + contractNr
	existingID, ok := m.contractNrs[key]
	if !ok {
		return false, nil
	}
	if excludeID != nil && existingID == *excludeID {
		return false, nil
	}
	return true, nil
}

func (m *mockRepositoryExtended) UpdateContractUsedValue(_ context.Context, tenantID, contractID uuid.UUID) error {
	fc, ok := m.contracts[contractID]
	if !ok {
		return ErrContractNotFound
	}
	// Sum all call amounts for this contract.
	var total float64
	for _, call := range m.contractCalls {
		if call.TenantID != tenantID || call.ContractID != contractID {
			continue
		}
		v, err := strconv.ParseFloat(call.Amount, 64)
		if err == nil {
			total += v
		}
	}
	fc.UsedValue = fmt.Sprintf("%g", total)
	return nil
}

// -------------------------------------------------------------------------
// Framework Contract Items
// -------------------------------------------------------------------------

func (m *mockRepositoryExtended) CreateContractItem(_ context.Context, item *FrameworkContractItem) error {
	m.contractItems[item.ID] = item
	return nil
}

func (m *mockRepositoryExtended) UpdateContractItem(_ context.Context, item *FrameworkContractItem) error {
	m.contractItems[item.ID] = item
	return nil
}

func (m *mockRepositoryExtended) DeleteContractItem(_ context.Context, _, itemID uuid.UUID) error {
	delete(m.contractItems, itemID)
	return nil
}

func (m *mockRepositoryExtended) ListContractItems(_ context.Context, tenantID, contractID uuid.UUID) ([]*FrameworkContractItem, error) {
	var result []*FrameworkContractItem
	for _, item := range m.contractItems {
		if item.TenantID == tenantID && item.ContractID == contractID {
			result = append(result, item)
		}
	}
	return result, nil
}

// QueryRowContractItem implements contractItemQuerier so that
// getContractItemByID (type-asserted in Service.UpdateContractItem) works in tests.
func (m *mockRepositoryExtended) QueryRowContractItem(_ context.Context, tenantID, itemID uuid.UUID) (*FrameworkContractItem, error) {
	item, ok := m.contractItems[itemID]
	if !ok || item.TenantID != tenantID {
		return nil, ErrContractItemNotFound
	}
	return item, nil
}

// -------------------------------------------------------------------------
// Framework Contract Calls
// -------------------------------------------------------------------------

func (m *mockRepositoryExtended) CreateContractCall(_ context.Context, call *FrameworkContractCall) error {
	m.contractCalls[call.ID] = call
	return nil
}

func (m *mockRepositoryExtended) ListContractCalls(_ context.Context, tenantID, contractID uuid.UUID) ([]*FrameworkContractCall, error) {
	var result []*FrameworkContractCall
	for _, call := range m.contractCalls {
		if call.TenantID == tenantID && call.ContractID == contractID {
			result = append(result, call)
		}
	}
	return result, nil
}

// ============================================================================
// Test helpers
// ============================================================================

func addCatalogItem(repo *mockRepositoryExtended, tenantID, supplierID uuid.UUID, name, category string) *CatalogItem {
	item := &CatalogItem{
		ID:          uuid.New(),
		TenantID:    tenantID,
		SupplierID:  supplierID,
		Name:        name,
		Category:    category,
		Price:       "10.00",
		Currency:    "EUR",
		Unit:        "Stk",
		Available:   true,
		MinOrderQty: "1",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.catalogItems[item.ID] = item
	return item
}

func addFrameworkContract(repo *mockRepositoryExtended, tenantID, supplierID uuid.UUID, title, contractNr string, status ContractStatus) *FrameworkContract {
	fc := &FrameworkContract{
		ID:         uuid.New(),
		TenantID:   tenantID,
		SupplierID: supplierID,
		Title:      title,
		ContractNr: contractNr,
		TotalValue: "10000",
		UsedValue:  "0",
		Currency:   "EUR",
		Status:     status,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	repo.contracts[fc.ID] = fc
	if contractNr != "" {
		key := tenantID.String() + ":" + contractNr
		repo.contractNrs[key] = fc.ID
	}
	return fc
}

// ============================================================================
// Catalog Tests
// ============================================================================

func TestExtended_CreateCatalogItem_Success(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID := uuid.New()
	supplierID := uuid.New()

	item, err := svc.CreateCatalogItem(context.Background(), CreateCatalogItemInput{
		TenantID:   tenantID,
		SupplierID: supplierID,
		Name:       "Screw M6",
		SKU:        "SCR-M6",
		Category:   "fasteners",
		Price:      "0.05",
		Currency:   "EUR",
		Unit:       "Stk",
		Available:  true,
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, item.ID)
	assert.Equal(t, "Screw M6", item.Name)
	assert.Equal(t, tenantID, item.TenantID)
	assert.Equal(t, supplierID, item.SupplierID)
	assert.Equal(t, "EUR", item.Currency)
	assert.Equal(t, "1", item.MinOrderQty) // default when not specified
}

func TestExtended_CreateCatalogItem_EmptyName(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	_, err := svc.CreateCatalogItem(context.Background(), CreateCatalogItemInput{
		TenantID:   uuid.New(),
		SupplierID: uuid.New(),
		Name:       "   ",
		Price:      "1.00",
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestExtended_ListCatalogItems_BySupplier(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID := uuid.New()
	supplierA := uuid.New()
	supplierB := uuid.New()

	addCatalogItem(repo, tenantID, supplierA, "Bolt A1", "fasteners")
	addCatalogItem(repo, tenantID, supplierA, "Bolt A2", "fasteners")
	addCatalogItem(repo, tenantID, supplierB, "Gear B1", "mechanical")

	items, total, err := svc.ListCatalogItems(context.Background(), ListCatalogItemsInput{
		TenantID:   tenantID,
		SupplierID: &supplierA,
		Page:       1,
		PageSize:   20,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, items, 2)
	for _, item := range items {
		assert.Equal(t, supplierA, item.SupplierID)
	}
}

// ============================================================================
// Supplier Rating Tests
// ============================================================================

func TestExtended_CreateSupplierRating_Success(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID := uuid.New()
	supplierID := uuid.New()
	ratedBy := uuid.New()

	rating, err := svc.CreateSupplierRating(context.Background(), CreateSupplierRatingInput{
		TenantID:   tenantID,
		SupplierID: supplierID,
		Category:   RatingCategoryQuality,
		Rating:     4,
		Comment:    "Good quality",
		RatedBy:    &ratedBy,
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, rating.ID)
	assert.Equal(t, tenantID, rating.TenantID)
	assert.Equal(t, supplierID, rating.SupplierID)
	assert.Equal(t, RatingCategoryQuality, rating.Category)
	assert.Equal(t, int32(4), rating.Rating)
	assert.Equal(t, "Good quality", rating.Comment)
	assert.NotNil(t, rating.RatedBy)
}

func TestExtended_CreateSupplierRating_InvalidRating_Zero(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	_, err := svc.CreateSupplierRating(context.Background(), CreateSupplierRatingInput{
		TenantID:   uuid.New(),
		SupplierID: uuid.New(),
		Category:   RatingCategoryDelivery,
		Rating:     0,
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestExtended_CreateSupplierRating_InvalidRating_Six(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	_, err := svc.CreateSupplierRating(context.Background(), CreateSupplierRatingInput{
		TenantID:   uuid.New(),
		SupplierID: uuid.New(),
		Category:   RatingCategoryPrice,
		Rating:     6,
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestExtended_CreateSupplierRating_InvalidCategory(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	_, err := svc.CreateSupplierRating(context.Background(), CreateSupplierRatingInput{
		TenantID:   uuid.New(),
		SupplierID: uuid.New(),
		Category:   RatingCategory("unknown"),
		Rating:     3,
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

// ============================================================================
// Framework Contract Tests
// ============================================================================

func TestExtended_CreateFrameworkContract_Success(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID := uuid.New()
	supplierID := uuid.New()

	fc, err := svc.CreateFrameworkContract(context.Background(), CreateFrameworkContractInput{
		TenantID:   tenantID,
		SupplierID: supplierID,
		Title:      "Annual Supply Agreement",
		ContractNr: "FC-2026-001",
		TotalValue: "50000",
		Currency:   "EUR",
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, fc.ID)
	assert.Equal(t, tenantID, fc.TenantID)
	assert.Equal(t, supplierID, fc.SupplierID)
	assert.Equal(t, "Annual Supply Agreement", fc.Title)
	assert.Equal(t, "FC-2026-001", fc.ContractNr)
	assert.Equal(t, "50000", fc.TotalValue)
	assert.Equal(t, "0", fc.UsedValue)
	assert.Equal(t, ContractStatusDraft, fc.Status) // defaults to draft
}

func TestExtended_CreateFrameworkContract_DuplicateContractNr(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID := uuid.New()
	supplierID := uuid.New()

	// Pre-seed a contract with the same number under the same tenant.
	addFrameworkContract(repo, tenantID, supplierID, "Existing Contract", "FC-2026-DUP", ContractStatusActive)

	_, err := svc.CreateFrameworkContract(context.Background(), CreateFrameworkContractInput{
		TenantID:   tenantID,
		SupplierID: supplierID,
		Title:      "Duplicate",
		ContractNr: "FC-2026-DUP",
	})

	assert.ErrorIs(t, err, ErrContractNrTaken)
}

func TestExtended_CreateContractCall_UpdatesUsedValue(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID := uuid.New()
	supplierID := uuid.New()

	fc := addFrameworkContract(repo, tenantID, supplierID, "Supply Frame", "FC-2026-CALL", ContractStatusActive)
	require.Equal(t, "0", fc.UsedValue)

	call, err := svc.CreateContractCall(context.Background(), CreateContractCallInput{
		TenantID:   tenantID,
		ContractID: fc.ID,
		Amount:     "1500",
		Currency:   "EUR",
		Notes:      "First call-off",
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, call.ID)
	assert.Equal(t, "1500", call.Amount)

	// After CreateContractCall the service calls UpdateContractUsedValue.
	// Verify that used_value was updated on the in-memory contract.
	updatedFC := repo.contracts[fc.ID]
	assert.NotEqual(t, "0", updatedFC.UsedValue,
		"used_value must be non-zero after a call-off of 1500")
}
