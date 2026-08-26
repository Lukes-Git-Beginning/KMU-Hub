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

// CreateContractCall mirrors the guarantees the Postgres implementation gives
// inside its transaction: existence, status, remaining value, and the
// used_value recompute. The SQL itself is proven against a real database in
// postgres_repository_extended_test.go — this double exists so the service
// tests can observe which of those rejections the service surfaces.
func (m *mockRepositoryExtended) CreateContractCall(_ context.Context, call *FrameworkContractCall) error {
	fc, ok := m.contracts[call.ContractID]
	if !ok || fc.TenantID != call.TenantID {
		return ErrContractNotFound
	}
	if fc.Status != ContractStatusActive {
		return fmt.Errorf("contract is in status %q: %w", fc.Status, ErrContractNotActive)
	}

	used := m.sumContractCalls(call.TenantID, call.ContractID)
	total, _ := strconv.ParseFloat(fc.TotalValue, 64)
	amount, _ := strconv.ParseFloat(call.Amount, 64)
	remaining := total - used
	if amount > remaining {
		return fmt.Errorf("call-off of %s, remaining %s: %w",
			call.Amount, strconv.FormatFloat(remaining, 'f', -1, 64), ErrContractBudgetExceeded)
	}

	m.contractCalls[call.ID] = call
	fc.UsedValue = strconv.FormatFloat(used+amount, 'f', -1, 64)
	return nil
}

func (m *mockRepositoryExtended) sumContractCalls(tenantID, contractID uuid.UUID) float64 {
	var total float64
	for _, c := range m.contractCalls {
		if c.TenantID != tenantID || c.ContractID != contractID {
			continue
		}
		if v, err := strconv.ParseFloat(c.Amount, 64); err == nil {
			total += v
		}
	}
	return total
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

	// used_value is written by the same repository call that inserts the
	// call-off, not by a separate follow-up the service could lose.
	updatedFC := repo.contracts[fc.ID]
	assert.NotEqual(t, "0", updatedFC.UsedValue,
		"used_value must be non-zero after a call-off of 1500")
}

// The framework contract's total_value was a decoration until 2026-08-10: the
// service loaded the contract only to prove it existed and never compared the
// amount against it, so a 10.000-EUR frame could be called off without limit.
func TestExtended_CreateContractCall_ExceedingRemainingIsRejected(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID, supplierID := uuid.New(), uuid.New()
	fc := addFrameworkContract(repo, tenantID, supplierID, "Capped Frame", "FC-CAP", ContractStatusActive)

	// 8000 of 10000 are already called off.
	_, err := svc.CreateContractCall(context.Background(), CreateContractCallInput{
		TenantID: tenantID, ContractID: fc.ID, Amount: "8000",
	})
	require.NoError(t, err)

	_, err = svc.CreateContractCall(context.Background(), CreateContractCallInput{
		TenantID: tenantID, ContractID: fc.ID, Amount: "2000.01",
	})
	require.ErrorIs(t, err, ErrContractBudgetExceeded)
	assert.Contains(t, err.Error(), "remaining 2000",
		"the message must name the remaining value, or the buyer cannot tell what would still fit")

	// The rejected call must not have been persisted.
	calls, listErr := svc.ListContractCalls(context.Background(), tenantID, fc.ID)
	require.NoError(t, listErr)
	assert.Len(t, calls, 1)
}

func TestExtended_CreateContractCall_ExactRemainingIsAccepted(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID, supplierID := uuid.New(), uuid.New()
	fc := addFrameworkContract(repo, tenantID, supplierID, "Exhausted Frame", "FC-EXACT", ContractStatusActive)

	_, err := svc.CreateContractCall(context.Background(), CreateContractCallInput{
		TenantID: tenantID, ContractID: fc.ID, Amount: "7500",
	})
	require.NoError(t, err)

	// Exactly the remaining 2500 still fits — the cap is inclusive.
	_, err = svc.CreateContractCall(context.Background(), CreateContractCallInput{
		TenantID: tenantID, ContractID: fc.ID, Amount: "2500",
	})
	require.NoError(t, err)

	// One cent beyond it does not.
	_, err = svc.CreateContractCall(context.Background(), CreateContractCallInput{
		TenantID: tenantID, ContractID: fc.ID, Amount: "0.01",
	})
	require.ErrorIs(t, err, ErrContractBudgetExceeded)
}

func TestExtended_CreateContractCall_InactiveContractIsRejected(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status ContractStatus
	}{
		{"draft", ContractStatusDraft},
		{"expired", ContractStatusExpired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := newMockRepositoryExtended()
			svc := NewServiceExtended(repo)

			tenantID, supplierID := uuid.New(), uuid.New()
			fc := addFrameworkContract(repo, tenantID, supplierID, "Inactive Frame", "FC-"+tc.name, tc.status)

			_, err := svc.CreateContractCall(context.Background(), CreateContractCallInput{
				TenantID: tenantID, ContractID: fc.ID, Amount: "10",
			})
			require.ErrorIs(t, err, ErrContractNotActive)
			assert.Contains(t, err.Error(), string(tc.status))
		})
	}
}

func TestExtended_CreateContractCall_UnknownContractIsNotFound(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	_, err := svc.CreateContractCall(context.Background(), CreateContractCallInput{
		TenantID: uuid.New(), ContractID: uuid.New(), Amount: "10",
	})
	require.ErrorIs(t, err, ErrContractNotFound)
}

// ============================================================================
// UpdateCatalogItem / DeleteCatalogItem / GetCatalogItem Tests
// ============================================================================

func TestExtended_UpdateCatalogItem_Success(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID, supplierID := uuid.New(), uuid.New()
	item := addCatalogItem(repo, tenantID, supplierID, "Widget", "tools")

	newName := "Widget Pro"
	newPrice := "12.50"
	result, err := svc.UpdateCatalogItem(context.Background(), UpdateCatalogItemInput{
		TenantID: tenantID, ItemID: item.ID, Name: &newName, Price: &newPrice,
	})

	require.NoError(t, err)
	assert.Equal(t, "Widget Pro", result.Name)
	assert.Equal(t, "12.50", result.Price)
}

func TestExtended_UpdateCatalogItem_NotFound(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	newName := "Ghost"
	_, err := svc.UpdateCatalogItem(context.Background(), UpdateCatalogItemInput{
		TenantID: uuid.New(), ItemID: uuid.New(), Name: &newName,
	})

	assert.ErrorIs(t, err, ErrCatalogItemNotFound)
}

func TestExtended_UpdateCatalogItem_InvalidPrice(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID, supplierID := uuid.New(), uuid.New()
	item := addCatalogItem(repo, tenantID, supplierID, "Widget", "tools")

	badPrice := "not-a-number"
	_, err := svc.UpdateCatalogItem(context.Background(), UpdateCatalogItemInput{
		TenantID: tenantID, ItemID: item.ID, Price: &badPrice,
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestExtended_UpdateCatalogItem_EmptyNameRejected(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID, supplierID := uuid.New(), uuid.New()
	item := addCatalogItem(repo, tenantID, supplierID, "Widget", "tools")

	blank := "   "
	_, err := svc.UpdateCatalogItem(context.Background(), UpdateCatalogItemInput{
		TenantID: tenantID, ItemID: item.ID, Name: &blank,
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestExtended_DeleteCatalogItem_Success(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID, supplierID := uuid.New(), uuid.New()
	item := addCatalogItem(repo, tenantID, supplierID, "Widget", "tools")

	err := svc.DeleteCatalogItem(context.Background(), tenantID, item.ID)

	require.NoError(t, err)
	assert.NotContains(t, repo.catalogItems, item.ID)
}

func TestExtended_DeleteCatalogItem_NotFound(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	err := svc.DeleteCatalogItem(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, ErrCatalogItemNotFound)
}

func TestExtended_GetCatalogItem_Success(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID, supplierID := uuid.New(), uuid.New()
	item := addCatalogItem(repo, tenantID, supplierID, "Widget", "tools")

	got, err := svc.GetCatalogItem(context.Background(), tenantID, item.ID)

	require.NoError(t, err)
	assert.Equal(t, item.ID, got.ID)
}

func TestExtended_GetCatalogItem_NotFound(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	_, err := svc.GetCatalogItem(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, ErrCatalogItemNotFound)
}

// ============================================================================
// DeleteSupplierRating / ListSupplierRatings Tests
// ============================================================================

func TestExtended_DeleteSupplierRating_Success(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID, supplierID := uuid.New(), uuid.New()
	rating, err := svc.CreateSupplierRating(context.Background(), CreateSupplierRatingInput{
		TenantID: tenantID, SupplierID: supplierID, Category: RatingCategoryQuality, Rating: 4,
	})
	require.NoError(t, err)

	err = svc.DeleteSupplierRating(context.Background(), tenantID, rating.ID)

	require.NoError(t, err)
	assert.NotContains(t, repo.supplierRatings, rating.ID)
}

func TestExtended_DeleteSupplierRating_NotFound(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	err := svc.DeleteSupplierRating(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, ErrSupplierRatingNotFound)
}

func TestExtended_ListSupplierRatings_ScopedToSupplierAndTenant(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID, supplierA, supplierB := uuid.New(), uuid.New(), uuid.New()
	_, err := svc.CreateSupplierRating(context.Background(), CreateSupplierRatingInput{
		TenantID: tenantID, SupplierID: supplierA, Category: RatingCategoryQuality, Rating: 5,
	})
	require.NoError(t, err)
	_, err = svc.CreateSupplierRating(context.Background(), CreateSupplierRatingInput{
		TenantID: tenantID, SupplierID: supplierA, Category: RatingCategoryDelivery, Rating: 3,
	})
	require.NoError(t, err)
	_, err = svc.CreateSupplierRating(context.Background(), CreateSupplierRatingInput{
		TenantID: tenantID, SupplierID: supplierB, Category: RatingCategoryQuality, Rating: 2,
	})
	require.NoError(t, err)

	result, err := svc.ListSupplierRatings(context.Background(), tenantID, supplierA)

	require.NoError(t, err)
	assert.Len(t, result, 2)
}

// ============================================================================
// UpdateFrameworkContract / DeleteFrameworkContract / Get / List Tests
// ============================================================================

func TestExtended_UpdateFrameworkContract_Success(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID, supplierID := uuid.New(), uuid.New()
	fc := addFrameworkContract(repo, tenantID, supplierID, "Frame A", "FC-UPD-001", ContractStatusDraft)

	newTitle := "Frame A Revised"
	activeStatus := ContractStatusActive
	result, err := svc.UpdateFrameworkContract(context.Background(), UpdateFrameworkContractInput{
		TenantID: tenantID, ContractID: fc.ID, Title: &newTitle, Status: &activeStatus,
	})

	require.NoError(t, err)
	assert.Equal(t, "Frame A Revised", result.Title)
	assert.Equal(t, ContractStatusActive, result.Status)
}

func TestExtended_UpdateFrameworkContract_NotFound(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	newTitle := "Ghost"
	_, err := svc.UpdateFrameworkContract(context.Background(), UpdateFrameworkContractInput{
		TenantID: uuid.New(), ContractID: uuid.New(), Title: &newTitle,
	})

	assert.ErrorIs(t, err, ErrContractNotFound)
}

func TestExtended_UpdateFrameworkContract_DuplicateContractNr(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID, supplierID := uuid.New(), uuid.New()
	_ = addFrameworkContract(repo, tenantID, supplierID, "Taken", "FC-TAKEN", ContractStatusDraft)
	fc := addFrameworkContract(repo, tenantID, supplierID, "Frame B", "FC-UPD-002", ContractStatusDraft)

	taken := "FC-TAKEN"
	_, err := svc.UpdateFrameworkContract(context.Background(), UpdateFrameworkContractInput{
		TenantID: tenantID, ContractID: fc.ID, ContractNr: &taken,
	})

	assert.ErrorIs(t, err, ErrContractNrTaken)
}

func TestExtended_UpdateFrameworkContract_EmptyTitleRejected(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID, supplierID := uuid.New(), uuid.New()
	fc := addFrameworkContract(repo, tenantID, supplierID, "Frame C", "FC-UPD-003", ContractStatusDraft)

	blank := "   "
	_, err := svc.UpdateFrameworkContract(context.Background(), UpdateFrameworkContractInput{
		TenantID: tenantID, ContractID: fc.ID, Title: &blank,
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestExtended_UpdateFrameworkContract_ClearEndDate(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID, supplierID := uuid.New(), uuid.New()
	fc := addFrameworkContract(repo, tenantID, supplierID, "Frame D", "FC-UPD-004", ContractStatusDraft)
	future := time.Now().Add(24 * time.Hour)
	fc.EndDate = &future

	result, err := svc.UpdateFrameworkContract(context.Background(), UpdateFrameworkContractInput{
		TenantID: tenantID, ContractID: fc.ID, EndDate: &time.Time{},
	})

	require.NoError(t, err)
	assert.Nil(t, result.EndDate)
}

func TestExtended_DeleteFrameworkContract_Success(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID, supplierID := uuid.New(), uuid.New()
	fc := addFrameworkContract(repo, tenantID, supplierID, "Frame E", "FC-DEL-001", ContractStatusDraft)

	err := svc.DeleteFrameworkContract(context.Background(), tenantID, fc.ID)

	require.NoError(t, err)
	assert.NotContains(t, repo.contracts, fc.ID)
}

func TestExtended_DeleteFrameworkContract_NotFound(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	err := svc.DeleteFrameworkContract(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, ErrContractNotFound)
}

func TestExtended_GetFrameworkContract_IncludesItems(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID, supplierID := uuid.New(), uuid.New()
	fc := addFrameworkContract(repo, tenantID, supplierID, "Frame F", "FC-GET-001", ContractStatusActive)
	item, err := svc.CreateContractItem(context.Background(), CreateContractItemInput{
		TenantID: tenantID, ContractID: fc.ID, Name: "Bolt M6", UnitPrice: "0.10", AgreedQty: "1000",
	})
	require.NoError(t, err)

	got, err := svc.GetFrameworkContract(context.Background(), tenantID, fc.ID)

	require.NoError(t, err)
	require.Len(t, got.Items, 1)
	assert.Equal(t, item.ID, got.Items[0].ID)
}

func TestExtended_ListFrameworkContracts_FilterByStatus(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID, supplierID := uuid.New(), uuid.New()
	addFrameworkContract(repo, tenantID, supplierID, "Active One", "FC-LIST-001", ContractStatusActive)
	addFrameworkContract(repo, tenantID, supplierID, "Draft One", "FC-LIST-002", ContractStatusDraft)

	active := ContractStatusActive
	result, total, err := svc.ListFrameworkContracts(context.Background(), ListFrameworkContractsInput{
		TenantID: tenantID, Status: &active,
	})

	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, result, 1)
	assert.Equal(t, ContractStatusActive, result[0].Status)
}

// ============================================================================
// CreateContractItem / UpdateContractItem / DeleteContractItem Tests
// ============================================================================

func TestExtended_CreateContractItem_Success(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID, supplierID := uuid.New(), uuid.New()
	fc := addFrameworkContract(repo, tenantID, supplierID, "Frame G", "FC-ITEM-001", ContractStatusActive)

	item, err := svc.CreateContractItem(context.Background(), CreateContractItemInput{
		TenantID: tenantID, ContractID: fc.ID, Name: "Bolt M8", UnitPrice: "0.15", AgreedQty: "500",
	})

	require.NoError(t, err)
	assert.Equal(t, "Bolt M8", item.Name)
	assert.Equal(t, "0", item.CalledQty)
}

func TestExtended_CreateContractItem_EmptyNameRejected(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID, supplierID := uuid.New(), uuid.New()
	fc := addFrameworkContract(repo, tenantID, supplierID, "Frame H", "FC-ITEM-002", ContractStatusActive)

	_, err := svc.CreateContractItem(context.Background(), CreateContractItemInput{
		TenantID: tenantID, ContractID: fc.ID, Name: "   ",
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestExtended_CreateContractItem_UnknownContractIsNotFound(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	_, err := svc.CreateContractItem(context.Background(), CreateContractItemInput{
		TenantID: uuid.New(), ContractID: uuid.New(), Name: "Bolt M10",
	})

	assert.ErrorIs(t, err, ErrContractNotFound)
}

func TestExtended_UpdateContractItem_Success(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID, supplierID := uuid.New(), uuid.New()
	fc := addFrameworkContract(repo, tenantID, supplierID, "Frame I", "FC-ITEM-003", ContractStatusActive)
	item, err := svc.CreateContractItem(context.Background(), CreateContractItemInput{
		TenantID: tenantID, ContractID: fc.ID, Name: "Bolt M12", UnitPrice: "0.20", AgreedQty: "200",
	})
	require.NoError(t, err)

	newPrice := "0.25"
	result, err := svc.UpdateContractItem(context.Background(), UpdateContractItemInput{
		TenantID: tenantID, ItemID: item.ID, UnitPrice: &newPrice,
	})

	require.NoError(t, err)
	assert.Equal(t, "0.25", result.UnitPrice)
}

func TestExtended_UpdateContractItem_NotFound(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	newPrice := "0.25"
	_, err := svc.UpdateContractItem(context.Background(), UpdateContractItemInput{
		TenantID: uuid.New(), ItemID: uuid.New(), UnitPrice: &newPrice,
	})

	assert.ErrorIs(t, err, ErrContractItemNotFound)
}

func TestExtended_UpdateContractItem_EmptyNameRejected(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID, supplierID := uuid.New(), uuid.New()
	fc := addFrameworkContract(repo, tenantID, supplierID, "Frame J", "FC-ITEM-004", ContractStatusActive)
	item, err := svc.CreateContractItem(context.Background(), CreateContractItemInput{
		TenantID: tenantID, ContractID: fc.ID, Name: "Bolt M14",
	})
	require.NoError(t, err)

	blank := "   "
	_, err = svc.UpdateContractItem(context.Background(), UpdateContractItemInput{
		TenantID: tenantID, ItemID: item.ID, Name: &blank,
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestExtended_DeleteContractItem_Success(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID, supplierID := uuid.New(), uuid.New()
	fc := addFrameworkContract(repo, tenantID, supplierID, "Frame K", "FC-ITEM-005", ContractStatusActive)
	item, err := svc.CreateContractItem(context.Background(), CreateContractItemInput{
		TenantID: tenantID, ContractID: fc.ID, Name: "Bolt M16",
	})
	require.NoError(t, err)

	err = svc.DeleteContractItem(context.Background(), tenantID, item.ID)

	require.NoError(t, err)
	assert.NotContains(t, repo.contractItems, item.ID)
}

// ============================================================================
// Bug-Fund: ReceiveGoods/PartialReceive never record a framework-contract
// call-off, even though PurchaseOrder.FrameworkContractID (models.go) and
// migration 000208_einkauf_po_framework_ref carry the column specifically for
// that purpose ("When set, ReceiveGoods will automatically record a contract
// call-off."). Neither CreatePOInput nor UpdatePOInput exposes the field, so
// the column can only be populated by hand-written SQL today — and even then,
// service.go's ReceiveGoods/PartialReceive never read po.FrameworkContractID
// or call CreateContractCall. Filed as
// feat-einkauf-po-framework-contract-call-off-wiring in BACKLOG.yml.
// ============================================================================

func TestExtended_ReceiveGoods_LinkedToFrameworkContract_DoesNotRecordCallOff(t *testing.T) {
	repo := newMockRepositoryExtended()
	svc := NewServiceExtended(repo)

	tenantID, supplierID := uuid.New(), uuid.New()
	fc := addFrameworkContract(repo, tenantID, supplierID, "Frame Linked", "FC-LINK-001", ContractStatusActive)
	s := addSupplier(repo.mockRepository, tenantID, "Linked Supplier")
	po := addPO(repo.mockRepository, tenantID, s.ID, "PO-LINKED-001", POStatusSubmitted)
	po.FrameworkContractID = &fc.ID
	addPOLine(repo.mockRepository, tenantID, po.ID, "Bolt M6", "100")

	result, err := svc.ReceiveGoods(context.Background(), tenantID, po.ID)

	require.NoError(t, err)
	assert.Equal(t, POStatusReceived, result.Status)

	calls, err := svc.ListContractCalls(context.Background(), tenantID, fc.ID)
	require.NoError(t, err)
	assert.Empty(t, calls, "documents the gap: ReceiveGoods does not create a contract call-off despite the docstring promising it")

	unchanged, err := svc.GetFrameworkContract(context.Background(), tenantID, fc.ID)
	require.NoError(t, err)
	assert.Equal(t, "0", unchanged.UsedValue, "documents the gap: used_value is never updated by a linked PO's goods receipt")
}
