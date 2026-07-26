package einkauf

import (
	"context"
	"strconv"
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
	suppliers map[uuid.UUID]*Supplier
	pos       map[uuid.UUID]*PurchaseOrder
	lines     map[uuid.UUID]*POLine

	// tenant+poNumber -> poID for uniqueness checks
	poNumbers map[string]uuid.UUID

	createSupplierErr error
	getSupplierErr    error
	createPOErr       error
	getPOErr          error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		suppliers: make(map[uuid.UUID]*Supplier),
		pos:       make(map[uuid.UUID]*PurchaseOrder),
		lines:     make(map[uuid.UUID]*POLine),
		poNumbers: make(map[string]uuid.UUID),
	}
}

// Suppliers

func (m *mockRepository) CreateSupplier(ctx context.Context, s *Supplier) error {
	if m.createSupplierErr != nil {
		return m.createSupplierErr
	}
	m.suppliers[s.ID] = s
	return nil
}

func (m *mockRepository) UpdateSupplier(ctx context.Context, s *Supplier) error {
	m.suppliers[s.ID] = s
	return nil
}

func (m *mockRepository) DeleteSupplier(ctx context.Context, tenantID, supplierID uuid.UUID) error {
	s, ok := m.suppliers[supplierID]
	if !ok {
		return ErrSupplierNotFound
	}
	now := time.Now()
	s.DeletedAt = &now
	return nil
}

func (m *mockRepository) GetSupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (*Supplier, error) {
	if m.getSupplierErr != nil {
		return nil, m.getSupplierErr
	}
	s, ok := m.suppliers[supplierID]
	if !ok || s.TenantID != tenantID || s.DeletedAt != nil {
		return nil, ErrSupplierNotFound
	}
	return s, nil
}

func (m *mockRepository) ListSuppliers(ctx context.Context, tenantID uuid.UUID, filter ListSuppliersFilter, offset, limit int) ([]*Supplier, int, error) {
	var result []*Supplier
	for _, s := range m.suppliers {
		if s.TenantID != tenantID || s.DeletedAt != nil {
			continue
		}
		result = append(result, s)
	}
	total := len(result)
	if offset >= total {
		return []*Supplier{}, total, nil
	}
	end := min(offset+limit, total)
	return result[offset:end], total, nil
}

// Purchase Orders

func (m *mockRepository) CreatePO(ctx context.Context, po *PurchaseOrder) error {
	if m.createPOErr != nil {
		return m.createPOErr
	}
	m.pos[po.ID] = po
	key := po.TenantID.String() + ":" + po.PONumber
	m.poNumbers[key] = po.ID
	return nil
}

func (m *mockRepository) UpdatePO(ctx context.Context, po *PurchaseOrder) error {
	m.pos[po.ID] = po
	return nil
}

func (m *mockRepository) DeletePO(ctx context.Context, tenantID, poID uuid.UUID) error {
	delete(m.pos, poID)
	return nil
}

func (m *mockRepository) GetPO(ctx context.Context, tenantID, poID uuid.UUID) (*PurchaseOrder, error) {
	if m.getPOErr != nil {
		return nil, m.getPOErr
	}
	po, ok := m.pos[poID]
	if !ok || po.TenantID != tenantID {
		return nil, ErrPONotFound
	}
	return po, nil
}

func (m *mockRepository) GetPOWithLines(ctx context.Context, tenantID, poID uuid.UUID) (*PurchaseOrder, error) {
	po, err := m.GetPO(ctx, tenantID, poID)
	if err != nil {
		return nil, err
	}
	lines, _ := m.ListPOLines(ctx, tenantID, poID)
	po.Lines = lines
	return po, nil
}

func (m *mockRepository) ListPOs(ctx context.Context, tenantID uuid.UUID, filter ListPOsFilter, offset, limit int) ([]*PurchaseOrder, int, error) {
	var result []*PurchaseOrder
	for _, po := range m.pos {
		if po.TenantID != tenantID {
			continue
		}
		if filter.Status != nil && po.Status != *filter.Status {
			continue
		}
		if filter.SupplierID != nil && po.SupplierID != *filter.SupplierID {
			continue
		}
		result = append(result, po)
	}
	total := len(result)
	if offset >= total {
		return []*PurchaseOrder{}, total, nil
	}
	end := min(offset+limit, total)
	return result[offset:end], total, nil
}

func (m *mockRepository) UpdatePOStatus(ctx context.Context, tenantID, poID uuid.UUID, status POStatus) error {
	po, ok := m.pos[poID]
	if !ok {
		return ErrPONotFound
	}
	po.Status = status
	po.UpdatedAt = time.Now()
	return nil
}

func (m *mockRepository) PONumberExists(ctx context.Context, tenantID uuid.UUID, poNumber string, excludeID *uuid.UUID) (bool, error) {
	key := tenantID.String() + ":" + poNumber
	existingID, ok := m.poNumbers[key]
	if !ok {
		return false, nil
	}
	if excludeID != nil && existingID == *excludeID {
		return false, nil
	}
	return true, nil
}

// PO Lines

func (m *mockRepository) CreatePOLine(ctx context.Context, line *POLine) error {
	m.lines[line.ID] = line
	return nil
}

func (m *mockRepository) UpdatePOLine(ctx context.Context, line *POLine) error {
	m.lines[line.ID] = line
	return nil
}

func (m *mockRepository) DeletePOLine(ctx context.Context, tenantID, lineID uuid.UUID) error {
	delete(m.lines, lineID)
	return nil
}

func (m *mockRepository) GetPOLine(ctx context.Context, tenantID, lineID uuid.UUID) (*POLine, error) {
	l, ok := m.lines[lineID]
	if !ok || l.TenantID != tenantID {
		return nil, ErrPOLineNotFound
	}
	return l, nil
}

func (m *mockRepository) ListPOLines(ctx context.Context, tenantID, poID uuid.UUID) ([]*POLine, error) {
	var result []*POLine
	for _, l := range m.lines {
		if l.POID == poID && l.TenantID == tenantID {
			result = append(result, l)
		}
	}
	return result, nil
}

func (m *mockRepository) UpdatePOLineReceivedQuantity(ctx context.Context, tenantID, lineID uuid.UUID, receivedQty string) error {
	l, ok := m.lines[lineID]
	if !ok {
		return ErrPOLineNotFound
	}
	l.ReceivedQuantity = receivedQty
	return nil
}

func (m *mockRepository) CountPOLines(ctx context.Context, tenantID, poID uuid.UUID) (int, error) {
	count := 0
	for _, l := range m.lines {
		if l.POID == poID && l.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

func (m *mockRepository) RecomputePOTotal(ctx context.Context, tenantID, poID uuid.UUID) error {
	po, ok := m.pos[poID]
	if !ok || po.TenantID != tenantID {
		return ErrPONotFound
	}
	var total float64
	for _, l := range m.lines {
		if l.POID != poID || l.TenantID != tenantID {
			continue
		}
		qty, _ := strconv.ParseFloat(l.Quantity, 64)
		price, _ := strconv.ParseFloat(l.UnitPrice, 64)
		total += qty * price
	}
	po.TotalAmount = strconv.FormatFloat(total, 'f', 2, 64)
	po.UpdatedAt = time.Now()
	return nil
}

// compile-time check
var _ Repository = (*mockRepository)(nil)

// ============================================================================
// Helpers
// ============================================================================

func addSupplier(repo *mockRepository, tenantID uuid.UUID, name string) *Supplier {
	s := &Supplier{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      name,
		Email:     "supplier@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.suppliers[s.ID] = s
	return s
}

func addPO(repo *mockRepository, tenantID, supplierID uuid.UUID, poNumber string, status POStatus) *PurchaseOrder {
	po := &PurchaseOrder{
		ID:          uuid.New(),
		TenantID:    tenantID,
		SupplierID:  supplierID,
		PONumber:    poNumber,
		Status:      status,
		OrderDate:   time.Now(),
		TotalAmount: "0",
		Currency:    "EUR",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.pos[po.ID] = po
	key := tenantID.String() + ":" + poNumber
	repo.poNumbers[key] = po.ID
	return po
}

func addPOLine(repo *mockRepository, tenantID, poID uuid.UUID, productName, qty string) *POLine {
	l := &POLine{
		ID:               uuid.New(),
		TenantID:         tenantID,
		POID:             poID,
		ProductName:      productName,
		Quantity:         qty,
		UnitPrice:        "10.00",
		TaxRate:          "0",
		ReceivedQuantity: "0",
		LinePosition:     1,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	repo.lines[l.ID] = l
	return l
}

// ============================================================================
// CreateSupplier Tests
// ============================================================================

func TestService_CreateSupplier_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s, err := svc.CreateSupplier(context.Background(), CreateSupplierInput{
		TenantID: tenantID,
		Name:     "Acme GmbH",
		Email:    "info@acme.de",
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, s.ID)
	assert.Equal(t, "Acme GmbH", s.Name)
	assert.Equal(t, tenantID, s.TenantID)
}

func TestService_CreateSupplier_EmptyName(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateSupplier(context.Background(), CreateSupplierInput{
		TenantID: uuid.New(),
		Name:     "   ",
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_CreateSupplier_RepoError(t *testing.T) {
	repo := newMockRepository()
	repo.createSupplierErr = ErrInvalidInput
	svc := NewService(repo)

	_, err := svc.CreateSupplier(context.Background(), CreateSupplierInput{
		TenantID: uuid.New(),
		Name:     "Supplier",
	})

	assert.Error(t, err)
}

// ============================================================================
// UpdateSupplier Tests
// ============================================================================

func TestService_UpdateSupplier_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s := addSupplier(repo, tenantID, "Old Name")

	newName := "New Name"
	updated, err := svc.UpdateSupplier(context.Background(), UpdateSupplierInput{
		TenantID:   tenantID,
		SupplierID: s.ID,
		Name:       &newName,
	})

	require.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
}

func TestService_UpdateSupplier_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.UpdateSupplier(context.Background(), UpdateSupplierInput{
		TenantID:   uuid.New(),
		SupplierID: uuid.New(),
	})

	assert.ErrorIs(t, err, ErrSupplierNotFound)
}

func TestService_UpdateSupplier_EmptyName(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s := addSupplier(repo, tenantID, "Supplier")

	emptyName := "   "
	_, err := svc.UpdateSupplier(context.Background(), UpdateSupplierInput{
		TenantID:   tenantID,
		SupplierID: s.ID,
		Name:       &emptyName,
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

// ============================================================================
// DeleteSupplier Tests
// ============================================================================

func TestService_DeleteSupplier_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s := addSupplier(repo, tenantID, "Supplier")

	err := svc.DeleteSupplier(context.Background(), tenantID, s.ID)

	require.NoError(t, err)
	assert.NotNil(t, repo.suppliers[s.ID].DeletedAt)
}

func TestService_DeleteSupplier_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	err := svc.DeleteSupplier(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, ErrSupplierNotFound)
}

// ============================================================================
// CreatePO Tests
// ============================================================================

func TestService_CreatePO_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s := addSupplier(repo, tenantID, "Supplier")

	po, err := svc.CreatePO(context.Background(), CreatePOInput{
		TenantID:   tenantID,
		SupplierID: s.ID,
		PONumber:   "PO-2026-001",
		OrderDate:  time.Now(),
	})

	require.NoError(t, err)
	assert.Equal(t, POStatusDraft, po.Status)
	assert.Equal(t, "PO-2026-001", po.PONumber)
	assert.Equal(t, "EUR", po.Currency) // default
}

func TestService_CreatePO_EmptyPONumber(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s := addSupplier(repo, tenantID, "Supplier")

	_, err := svc.CreatePO(context.Background(), CreatePOInput{
		TenantID:   tenantID,
		SupplierID: s.ID,
		PONumber:   "",
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_CreatePO_SupplierNotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreatePO(context.Background(), CreatePOInput{
		TenantID:   uuid.New(),
		SupplierID: uuid.New(),
		PONumber:   "PO-001",
		OrderDate:  time.Now(),
	})

	assert.ErrorIs(t, err, ErrSupplierNotFound)
}

func TestService_CreatePO_DuplicatePONumber(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s := addSupplier(repo, tenantID, "Supplier")
	addPO(repo, tenantID, s.ID, "PO-001", POStatusDraft)

	_, err := svc.CreatePO(context.Background(), CreatePOInput{
		TenantID:   tenantID,
		SupplierID: s.ID,
		PONumber:   "PO-001",
		OrderDate:  time.Now(),
	})

	assert.ErrorIs(t, err, ErrPONumberTaken)
}

// ============================================================================
// DeletePO Tests
// ============================================================================

func TestService_DeletePO_DraftAllowed(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s := addSupplier(repo, tenantID, "Supplier")
	po := addPO(repo, tenantID, s.ID, "PO-001", POStatusDraft)

	err := svc.DeletePO(context.Background(), tenantID, po.ID)

	require.NoError(t, err)
	assert.NotContains(t, repo.pos, po.ID)
}

func TestService_DeletePO_NonDraftRejected(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s := addSupplier(repo, tenantID, "Supplier")
	po := addPO(repo, tenantID, s.ID, "PO-002", POStatusSubmitted)

	err := svc.DeletePO(context.Background(), tenantID, po.ID)

	assert.ErrorIs(t, err, ErrPONotDraft)
}

// ============================================================================
// SubmitPO Tests
// ============================================================================

func TestService_SubmitPO_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s := addSupplier(repo, tenantID, "Supplier")
	po := addPO(repo, tenantID, s.ID, "PO-003", POStatusDraft)
	addPOLine(repo, tenantID, po.ID, "Widget", "10")

	result, err := svc.SubmitPO(context.Background(), tenantID, po.ID)

	require.NoError(t, err)
	assert.Equal(t, POStatusSubmitted, result.Status)
}

func TestService_SubmitPO_NoLines(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s := addSupplier(repo, tenantID, "Supplier")
	po := addPO(repo, tenantID, s.ID, "PO-004", POStatusDraft)

	_, err := svc.SubmitPO(context.Background(), tenantID, po.ID)

	assert.ErrorIs(t, err, ErrPONotSubmittable)
}

func TestService_SubmitPO_NonDraftRejected(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s := addSupplier(repo, tenantID, "Supplier")
	po := addPO(repo, tenantID, s.ID, "PO-005", POStatusSubmitted)

	_, err := svc.SubmitPO(context.Background(), tenantID, po.ID)

	assert.ErrorIs(t, err, ErrPONotDraft)
}

// ============================================================================
// ReceiveGoods Tests
// ============================================================================

func TestService_ReceiveGoods_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s := addSupplier(repo, tenantID, "Supplier")
	po := addPO(repo, tenantID, s.ID, "PO-006", POStatusSubmitted)
	addPOLine(repo, tenantID, po.ID, "Widget", "5")

	result, err := svc.ReceiveGoods(context.Background(), tenantID, po.ID)

	require.NoError(t, err)
	assert.Equal(t, POStatusReceived, result.Status)

	// check that line received_quantity was updated
	for _, l := range repo.lines {
		if l.POID == po.ID {
			assert.Equal(t, "5", l.ReceivedQuantity)
		}
	}
}

func TestService_ReceiveGoods_WrongStatus(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s := addSupplier(repo, tenantID, "Supplier")
	po := addPO(repo, tenantID, s.ID, "PO-007", POStatusDraft)

	_, err := svc.ReceiveGoods(context.Background(), tenantID, po.ID)

	assert.ErrorIs(t, err, ErrPONotReceivable)
}

// ============================================================================
// PartialReceive Tests
// ============================================================================

func TestService_PartialReceive_PartialSetsStatus(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s := addSupplier(repo, tenantID, "Supplier")
	po := addPO(repo, tenantID, s.ID, "PO-008", POStatusSubmitted)
	l1 := addPOLine(repo, tenantID, po.ID, "Widget A", "10")
	_ = addPOLine(repo, tenantID, po.ID, "Widget B", "5")

	// Only partially receive line 1
	items := []PartialReceiveItem{
		{LineID: l1.ID, ReceivedQuantity: "7"},
	}

	result, err := svc.PartialReceive(context.Background(), tenantID, po.ID, items)

	require.NoError(t, err)
	assert.Equal(t, POStatusPartiallyReceived, result.Status)
}

func TestService_PartialReceive_AllFullySetsReceived(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s := addSupplier(repo, tenantID, "Supplier")
	po := addPO(repo, tenantID, s.ID, "PO-009", POStatusSubmitted)
	l1 := addPOLine(repo, tenantID, po.ID, "Widget A", "10")

	items := []PartialReceiveItem{
		{LineID: l1.ID, ReceivedQuantity: "10"},
	}

	result, err := svc.PartialReceive(context.Background(), tenantID, po.ID, items)

	require.NoError(t, err)
	assert.Equal(t, POStatusReceived, result.Status)
}

// ============================================================================
// AddPOLine Tests
// ============================================================================

func TestService_AddPOLine_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s := addSupplier(repo, tenantID, "Supplier")
	po := addPO(repo, tenantID, s.ID, "PO-010", POStatusDraft)

	line, err := svc.AddPOLine(context.Background(), AddPOLineInput{
		TenantID:    tenantID,
		POID:        po.ID,
		ProductName: "Screw M4",
		Quantity:    "100",
		UnitPrice:   "0.05",
	})

	require.NoError(t, err)
	assert.Equal(t, "Screw M4", line.ProductName)
	assert.Equal(t, "0", line.ReceivedQuantity) // starts at 0
	assert.Equal(t, "0", line.TaxRate)          // default

	got, err := svc.GetPO(context.Background(), tenantID, po.ID)
	require.NoError(t, err)
	assert.Equal(t, "5.00", got.TotalAmount) // 100 * 0.05
}

func TestService_AddPOLine_RecomputesTotalAcrossLines(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s := addSupplier(repo, tenantID, "Supplier")
	po := addPO(repo, tenantID, s.ID, "PO-012", POStatusDraft)

	_, err := svc.AddPOLine(context.Background(), AddPOLineInput{
		TenantID: tenantID, POID: po.ID, ProductName: "Line A", Quantity: "2", UnitPrice: "10.00",
	})
	require.NoError(t, err)

	_, err = svc.AddPOLine(context.Background(), AddPOLineInput{
		TenantID: tenantID, POID: po.ID, ProductName: "Line B", Quantity: "3", UnitPrice: "5.00",
	})
	require.NoError(t, err)

	got, err := svc.GetPO(context.Background(), tenantID, po.ID)
	require.NoError(t, err)
	assert.Equal(t, "35.00", got.TotalAmount) // 2*10 + 3*5
}

func TestService_UpdatePOLine_RecomputesTotal(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s := addSupplier(repo, tenantID, "Supplier")
	po := addPO(repo, tenantID, s.ID, "PO-013", POStatusDraft)

	line, err := svc.AddPOLine(context.Background(), AddPOLineInput{
		TenantID: tenantID, POID: po.ID, ProductName: "Line A", Quantity: "2", UnitPrice: "10.00",
	})
	require.NoError(t, err)

	newQty := "4"
	_, err = svc.UpdatePOLine(context.Background(), UpdatePOLineInput{
		TenantID: tenantID, LineID: line.ID, Quantity: &newQty,
	})
	require.NoError(t, err)

	got, err := svc.GetPO(context.Background(), tenantID, po.ID)
	require.NoError(t, err)
	assert.Equal(t, "40.00", got.TotalAmount) // 4*10
}

func TestService_DeletePOLine_RecomputesTotal(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s := addSupplier(repo, tenantID, "Supplier")
	po := addPO(repo, tenantID, s.ID, "PO-014", POStatusDraft)

	lineA, err := svc.AddPOLine(context.Background(), AddPOLineInput{
		TenantID: tenantID, POID: po.ID, ProductName: "Line A", Quantity: "2", UnitPrice: "10.00",
	})
	require.NoError(t, err)
	_, err = svc.AddPOLine(context.Background(), AddPOLineInput{
		TenantID: tenantID, POID: po.ID, ProductName: "Line B", Quantity: "3", UnitPrice: "5.00",
	})
	require.NoError(t, err)

	require.NoError(t, svc.DeletePOLine(context.Background(), tenantID, lineA.ID))

	got, err := svc.GetPO(context.Background(), tenantID, po.ID)
	require.NoError(t, err)
	assert.Equal(t, "15.00", got.TotalAmount) // remaining 3*5
}

func TestService_AddPOLine_EmptyProductName(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s := addSupplier(repo, tenantID, "Supplier")
	po := addPO(repo, tenantID, s.ID, "PO-011", POStatusDraft)

	_, err := svc.AddPOLine(context.Background(), AddPOLineInput{
		TenantID:    tenantID,
		POID:        po.ID,
		ProductName: "  ",
		Quantity:    "1",
		UnitPrice:   "10",
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_AddPOLine_PONotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.AddPOLine(context.Background(), AddPOLineInput{
		TenantID:    uuid.New(),
		POID:        uuid.New(),
		ProductName: "Widget",
		Quantity:    "1",
		UnitPrice:   "10",
	})

	assert.ErrorIs(t, err, ErrPONotFound)
}

// ============================================================================
// ListSuppliers / ListPOs Tests
// ============================================================================

func TestService_ListSuppliers_Pagination(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	for range 5 {
		addSupplier(repo, tenantID, "Supplier")
	}

	suppliers, total, err := svc.ListSuppliers(context.Background(), ListSuppliersInput{
		TenantID: tenantID,
		Page:     1,
		PageSize: 3,
	})

	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, suppliers, 3)
}

func TestService_ListPOs_FilterByStatus(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s := addSupplier(repo, tenantID, "Supplier")
	addPO(repo, tenantID, s.ID, "PO-A", POStatusDraft)
	addPO(repo, tenantID, s.ID, "PO-B", POStatusSubmitted)
	addPO(repo, tenantID, s.ID, "PO-C", POStatusDraft)

	draftStatus := POStatusDraft
	pos, total, err := svc.ListPOs(context.Background(), ListPOsInput{
		TenantID: tenantID,
		Status:   &draftStatus,
		Page:     1,
		PageSize: 20,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, pos, 2)
	for _, po := range pos {
		assert.Equal(t, POStatusDraft, po.Status)
	}
}

// ============================================================================
// allFullyReceived helper Tests
// ============================================================================

func TestAllFullyReceived_True(t *testing.T) {
	lines := []*POLine{
		{Quantity: "10", ReceivedQuantity: "10"},
		{Quantity: "5", ReceivedQuantity: "5"},
	}
	assert.True(t, allFullyReceived(lines))
}

func TestAllFullyReceived_False(t *testing.T) {
	lines := []*POLine{
		{Quantity: "10", ReceivedQuantity: "7"},
		{Quantity: "5", ReceivedQuantity: "5"},
	}
	assert.False(t, allFullyReceived(lines))
}

func TestAllFullyReceived_Empty(t *testing.T) {
	assert.False(t, allFullyReceived(nil))
}

// ============================================================================
// Bug-Fix Tests — numeric precision + PartialReceive validation
// ============================================================================

// TestService_AllFullyReceived_DecimalPrecisionMismatch verifies that
// "10.0000" and "10" are treated as equal (Bug Fix #1).
func TestService_AllFullyReceived_DecimalPrecisionMismatch(t *testing.T) {
	lines := []*POLine{
		{Quantity: "10.0000", ReceivedQuantity: "10"},
		{Quantity: "5.00", ReceivedQuantity: "5.0"},
	}
	assert.True(t, allFullyReceived(lines))
}

// TestService_PartialReceive_NegativeQuantity verifies that a negative
// ReceivedQuantity is rejected with ErrInvalidQuantity (Bug Fix #2).
func TestService_PartialReceive_NegativeQuantity(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s := addSupplier(repo, tenantID, "Supplier")
	po := addPO(repo, tenantID, s.ID, "PO-NEG-001", POStatusSubmitted)
	l1 := addPOLine(repo, tenantID, po.ID, "Widget", "10")

	_, err := svc.PartialReceive(context.Background(), tenantID, po.ID, []PartialReceiveItem{
		{LineID: l1.ID, ReceivedQuantity: "-5"},
	})

	assert.ErrorIs(t, err, ErrInvalidQuantity)
}

// TestService_PartialReceive_ExceedsOrdered verifies that receiving more than
// the ordered quantity is rejected with ErrExceedsOrderedQty (Bug Fix #2).
func TestService_PartialReceive_ExceedsOrdered(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s := addSupplier(repo, tenantID, "Supplier")
	po := addPO(repo, tenantID, s.ID, "PO-EXCEED-001", POStatusSubmitted)
	l1 := addPOLine(repo, tenantID, po.ID, "Widget", "10")

	_, err := svc.PartialReceive(context.Background(), tenantID, po.ID, []PartialReceiveItem{
		{LineID: l1.ID, ReceivedQuantity: "15"},
	})

	assert.ErrorIs(t, err, ErrExceedsOrderedQty)
}

// TestService_PartialReceive_InvalidString verifies that a non-numeric
// ReceivedQuantity string is rejected with ErrInvalidQuantity (Bug Fix #2).
func TestService_PartialReceive_InvalidString(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	s := addSupplier(repo, tenantID, "Supplier")
	po := addPO(repo, tenantID, s.ID, "PO-NAN-001", POStatusSubmitted)
	l1 := addPOLine(repo, tenantID, po.ID, "Widget", "10")

	_, err := svc.PartialReceive(context.Background(), tenantID, po.ID, []PartialReceiveItem{
		{LineID: l1.ID, ReceivedQuantity: "abc"},
	})

	assert.ErrorIs(t, err, ErrInvalidQuantity)
}
