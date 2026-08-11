package server

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/einkauf"
	einkaufv1 "github.com/kmuhub/kmuhub/proto/einkauf/v1"
)

// ============================================================================
// Stub repository — implements einkauf.RepositoryExtended plus the unexported
// QueryRowContractItem the service type-asserts for (contractItemQuerier).
// ============================================================================

type stubEinkaufRepo struct {
	mu sync.Mutex

	suppliers map[uuid.UUID]*einkauf.Supplier
	pos       map[uuid.UUID]*einkauf.PurchaseOrder
	lines     map[uuid.UUID]*einkauf.POLine
	poNumbers map[string]uuid.UUID // tenant:poNumber -> poID

	catalogItems map[uuid.UUID]*einkauf.CatalogItem
	ratings      map[uuid.UUID]*einkauf.SupplierRating
	contracts    map[uuid.UUID]*einkauf.FrameworkContract
	contractNrs  map[string]uuid.UUID // tenant:contractNr -> contractID
	contractItems map[uuid.UUID]*einkauf.FrameworkContractItem
	contractCalls map[uuid.UUID]*einkauf.FrameworkContractCall

	getSupplierErr error
	getPOErr       error
}

func newStubEinkaufRepo() *stubEinkaufRepo {
	return &stubEinkaufRepo{
		suppliers:     make(map[uuid.UUID]*einkauf.Supplier),
		pos:           make(map[uuid.UUID]*einkauf.PurchaseOrder),
		lines:         make(map[uuid.UUID]*einkauf.POLine),
		poNumbers:     make(map[string]uuid.UUID),
		catalogItems:  make(map[uuid.UUID]*einkauf.CatalogItem),
		ratings:       make(map[uuid.UUID]*einkauf.SupplierRating),
		contracts:     make(map[uuid.UUID]*einkauf.FrameworkContract),
		contractNrs:   make(map[string]uuid.UUID),
		contractItems: make(map[uuid.UUID]*einkauf.FrameworkContractItem),
		contractCalls: make(map[uuid.UUID]*einkauf.FrameworkContractCall),
	}
}

// --- Suppliers ---

func (r *stubEinkaufRepo) CreateSupplier(_ context.Context, s *einkauf.Supplier) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.suppliers[s.ID] = s
	return nil
}

func (r *stubEinkaufRepo) UpdateSupplier(_ context.Context, s *einkauf.Supplier) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.suppliers[s.ID]; !ok {
		return einkauf.ErrSupplierNotFound
	}
	r.suppliers[s.ID] = s
	return nil
}

func (r *stubEinkaufRepo) DeleteSupplier(_ context.Context, _, supplierID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.suppliers[supplierID]
	if !ok {
		return einkauf.ErrSupplierNotFound
	}
	now := time.Now()
	s.DeletedAt = &now
	return nil
}

func (r *stubEinkaufRepo) GetSupplier(_ context.Context, tenantID, supplierID uuid.UUID) (*einkauf.Supplier, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.getSupplierErr != nil {
		return nil, r.getSupplierErr
	}
	s, ok := r.suppliers[supplierID]
	if !ok || s.TenantID != tenantID || s.DeletedAt != nil {
		return nil, einkauf.ErrSupplierNotFound
	}
	return s, nil
}

func (r *stubEinkaufRepo) ListSuppliers(_ context.Context, tenantID uuid.UUID, filter einkauf.ListSuppliersFilter, offset, limit int) ([]*einkauf.Supplier, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*einkauf.Supplier
	for _, s := range r.suppliers {
		if s.TenantID != tenantID || s.DeletedAt != nil {
			continue
		}
		result = append(result, s)
	}
	total := len(result)
	if offset >= total {
		return []*einkauf.Supplier{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return result[offset:end], total, nil
}

// --- Purchase Orders ---

func (r *stubEinkaufRepo) CreatePO(_ context.Context, po *einkauf.PurchaseOrder) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pos[po.ID] = po
	r.poNumbers[po.TenantID.String()+":"+po.PONumber] = po.ID
	return nil
}

func (r *stubEinkaufRepo) UpdatePO(_ context.Context, po *einkauf.PurchaseOrder) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pos[po.ID] = po
	return nil
}

func (r *stubEinkaufRepo) DeletePO(_ context.Context, _, poID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.pos, poID)
	return nil
}

func (r *stubEinkaufRepo) GetPO(_ context.Context, tenantID, poID uuid.UUID) (*einkauf.PurchaseOrder, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.getPOErr != nil {
		return nil, r.getPOErr
	}
	po, ok := r.pos[poID]
	if !ok || po.TenantID != tenantID {
		return nil, einkauf.ErrPONotFound
	}
	return po, nil
}

func (r *stubEinkaufRepo) GetPOWithLines(ctx context.Context, tenantID, poID uuid.UUID) (*einkauf.PurchaseOrder, error) {
	po, err := r.GetPO(ctx, tenantID, poID)
	if err != nil {
		return nil, err
	}
	lines, _ := r.ListPOLines(ctx, tenantID, poID)
	po.Lines = lines
	return po, nil
}

func (r *stubEinkaufRepo) ListPOs(_ context.Context, tenantID uuid.UUID, filter einkauf.ListPOsFilter, offset, limit int) ([]*einkauf.PurchaseOrder, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*einkauf.PurchaseOrder
	for _, po := range r.pos {
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
		return []*einkauf.PurchaseOrder{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return result[offset:end], total, nil
}

func (r *stubEinkaufRepo) UpdatePOStatus(_ context.Context, _, poID uuid.UUID, status einkauf.POStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	po, ok := r.pos[poID]
	if !ok {
		return einkauf.ErrPONotFound
	}
	po.Status = status
	return nil
}

func (r *stubEinkaufRepo) PONumberExists(_ context.Context, tenantID uuid.UUID, poNumber string, excludeID *uuid.UUID) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.poNumbers[tenantID.String()+":"+poNumber]
	if !ok {
		return false, nil
	}
	if excludeID != nil && id == *excludeID {
		return false, nil
	}
	return true, nil
}

// --- PO Lines ---

func (r *stubEinkaufRepo) CreatePOLine(_ context.Context, line *einkauf.POLine) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lines[line.ID] = line
	return nil
}

func (r *stubEinkaufRepo) UpdatePOLine(_ context.Context, line *einkauf.POLine) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lines[line.ID] = line
	return nil
}

func (r *stubEinkaufRepo) DeletePOLine(_ context.Context, _, lineID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.lines, lineID)
	return nil
}

func (r *stubEinkaufRepo) GetPOLine(_ context.Context, tenantID, lineID uuid.UUID) (*einkauf.POLine, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.lines[lineID]
	if !ok || l.TenantID != tenantID {
		return nil, einkauf.ErrPOLineNotFound
	}
	return l, nil
}

func (r *stubEinkaufRepo) ListPOLines(_ context.Context, tenantID, poID uuid.UUID) ([]*einkauf.POLine, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*einkauf.POLine
	for _, l := range r.lines {
		if l.POID == poID && l.TenantID == tenantID {
			result = append(result, l)
		}
	}
	return result, nil
}

func (r *stubEinkaufRepo) UpdatePOLineReceivedQuantity(_ context.Context, tenantID, lineID uuid.UUID, receivedQty string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.lines[lineID]
	if !ok || l.TenantID != tenantID {
		return einkauf.ErrPOLineNotFound
	}
	l.ReceivedQuantity = receivedQty
	return nil
}

func (r *stubEinkaufRepo) CountPOLines(_ context.Context, tenantID, poID uuid.UUID) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, l := range r.lines {
		if l.POID == poID && l.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

// RecomputePOTotal mirrors numeric(15,4): sum(quantity * unit_price) formatted
// with four decimal places, not the two-decimal float most callers assume.
func (r *stubEinkaufRepo) RecomputePOTotal(_ context.Context, tenantID, poID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	po, ok := r.pos[poID]
	if !ok || po.TenantID != tenantID {
		return einkauf.ErrPONotFound
	}
	var total float64
	for _, l := range r.lines {
		if l.POID != poID || l.TenantID != tenantID {
			continue
		}
		qty, _ := strconv.ParseFloat(l.Quantity, 64)
		price, _ := strconv.ParseFloat(l.UnitPrice, 64)
		total += qty * price
	}
	po.TotalAmount = strconv.FormatFloat(total, 'f', 4, 64)
	return nil
}

// --- Catalog Items ---

func (r *stubEinkaufRepo) CreateCatalogItem(_ context.Context, item *einkauf.CatalogItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.catalogItems[item.ID] = item
	return nil
}

func (r *stubEinkaufRepo) UpdateCatalogItem(_ context.Context, item *einkauf.CatalogItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.catalogItems[item.ID]; !ok {
		return einkauf.ErrCatalogItemNotFound
	}
	r.catalogItems[item.ID] = item
	return nil
}

func (r *stubEinkaufRepo) DeleteCatalogItem(_ context.Context, _, itemID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.catalogItems[itemID]; !ok {
		return einkauf.ErrCatalogItemNotFound
	}
	delete(r.catalogItems, itemID)
	return nil
}

func (r *stubEinkaufRepo) GetCatalogItem(_ context.Context, tenantID, itemID uuid.UUID) (*einkauf.CatalogItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.catalogItems[itemID]
	if !ok || item.TenantID != tenantID {
		return nil, einkauf.ErrCatalogItemNotFound
	}
	return item, nil
}

func (r *stubEinkaufRepo) ListCatalogItems(_ context.Context, tenantID uuid.UUID, filter einkauf.ListCatalogItemsFilter, offset, limit int) ([]*einkauf.CatalogItem, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*einkauf.CatalogItem
	for _, item := range r.catalogItems {
		if item.TenantID != tenantID {
			continue
		}
		if filter.SupplierID != nil && item.SupplierID != *filter.SupplierID {
			continue
		}
		if filter.Available != nil && item.Available != *filter.Available {
			continue
		}
		result = append(result, item)
	}
	total := len(result)
	if offset >= total {
		return []*einkauf.CatalogItem{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return result[offset:end], total, nil
}

// --- Supplier Ratings ---

func (r *stubEinkaufRepo) CreateSupplierRating(_ context.Context, rating *einkauf.SupplierRating) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ratings[rating.ID] = rating
	return nil
}

func (r *stubEinkaufRepo) DeleteSupplierRating(_ context.Context, _, ratingID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.ratings[ratingID]; !ok {
		return einkauf.ErrSupplierRatingNotFound
	}
	delete(r.ratings, ratingID)
	return nil
}

func (r *stubEinkaufRepo) GetSupplierRating(_ context.Context, tenantID, ratingID uuid.UUID) (*einkauf.SupplierRating, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rt, ok := r.ratings[ratingID]
	if !ok || rt.TenantID != tenantID {
		return nil, einkauf.ErrSupplierRatingNotFound
	}
	return rt, nil
}

func (r *stubEinkaufRepo) ListSupplierRatings(_ context.Context, tenantID, supplierID uuid.UUID) ([]*einkauf.SupplierRating, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*einkauf.SupplierRating
	for _, rt := range r.ratings {
		if rt.TenantID == tenantID && rt.SupplierID == supplierID {
			result = append(result, rt)
		}
	}
	return result, nil
}

// --- Framework Contracts ---

func (r *stubEinkaufRepo) CreateFrameworkContract(_ context.Context, fc *einkauf.FrameworkContract) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.contracts[fc.ID] = fc
	if fc.ContractNr != "" {
		r.contractNrs[fc.TenantID.String()+":"+fc.ContractNr] = fc.ID
	}
	return nil
}

func (r *stubEinkaufRepo) UpdateFrameworkContract(_ context.Context, fc *einkauf.FrameworkContract) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.contracts[fc.ID]; !ok {
		return einkauf.ErrContractNotFound
	}
	r.contracts[fc.ID] = fc
	if fc.ContractNr != "" {
		r.contractNrs[fc.TenantID.String()+":"+fc.ContractNr] = fc.ID
	}
	return nil
}

func (r *stubEinkaufRepo) DeleteFrameworkContract(_ context.Context, _, contractID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.contracts[contractID]; !ok {
		return einkauf.ErrContractNotFound
	}
	delete(r.contracts, contractID)
	return nil
}

func (r *stubEinkaufRepo) GetFrameworkContract(_ context.Context, tenantID, contractID uuid.UUID) (*einkauf.FrameworkContract, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	fc, ok := r.contracts[contractID]
	if !ok || fc.TenantID != tenantID {
		return nil, einkauf.ErrContractNotFound
	}
	return fc, nil
}

func (r *stubEinkaufRepo) GetFrameworkContractWithItems(ctx context.Context, tenantID, contractID uuid.UUID) (*einkauf.FrameworkContract, error) {
	fc, err := r.GetFrameworkContract(ctx, tenantID, contractID)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var items []*einkauf.FrameworkContractItem
	for _, item := range r.contractItems {
		if item.ContractID == contractID && item.TenantID == tenantID {
			items = append(items, item)
		}
	}
	fc.Items = items
	return fc, nil
}

func (r *stubEinkaufRepo) ListFrameworkContracts(_ context.Context, tenantID uuid.UUID, filter einkauf.ListContractsFilter, offset, limit int) ([]*einkauf.FrameworkContract, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*einkauf.FrameworkContract
	for _, fc := range r.contracts {
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
		return []*einkauf.FrameworkContract{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return result[offset:end], total, nil
}

func (r *stubEinkaufRepo) ContractNrExists(_ context.Context, tenantID uuid.UUID, contractNr string, excludeID *uuid.UUID) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.contractNrs[tenantID.String()+":"+contractNr]
	if !ok {
		return false, nil
	}
	if excludeID != nil && id == *excludeID {
		return false, nil
	}
	return true, nil
}

// --- Framework Contract Items ---

func (r *stubEinkaufRepo) CreateContractItem(_ context.Context, item *einkauf.FrameworkContractItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.contractItems[item.ID] = item
	return nil
}

func (r *stubEinkaufRepo) UpdateContractItem(_ context.Context, item *einkauf.FrameworkContractItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.contractItems[item.ID]; !ok {
		return einkauf.ErrContractItemNotFound
	}
	r.contractItems[item.ID] = item
	return nil
}

func (r *stubEinkaufRepo) DeleteContractItem(_ context.Context, _, itemID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.contractItems[itemID]; !ok {
		return einkauf.ErrContractItemNotFound
	}
	delete(r.contractItems, itemID)
	return nil
}

func (r *stubEinkaufRepo) ListContractItems(_ context.Context, tenantID, contractID uuid.UUID) ([]*einkauf.FrameworkContractItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*einkauf.FrameworkContractItem
	for _, item := range r.contractItems {
		if item.ContractID == contractID && item.TenantID == tenantID {
			result = append(result, item)
		}
	}
	return result, nil
}

// QueryRowContractItem satisfies the unexported contractItemQuerier interface
// that Service.UpdateContractItem type-asserts for. Without it, every
// UpdateContractItem call falls through to ErrContractItemNotFound regardless
// of whether the item exists — see einkauf.getContractItemByID.
func (r *stubEinkaufRepo) QueryRowContractItem(_ context.Context, tenantID, itemID uuid.UUID) (*einkauf.FrameworkContractItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.contractItems[itemID]
	if !ok || item.TenantID != tenantID {
		return nil, einkauf.ErrContractItemNotFound
	}
	return item, nil
}

// --- Framework Contract Calls ---

// CreateContractCall stands in for the repository's transaction: it persists
// the call and recomputes used_value in one step. The status and remaining
// value checks the real implementation performs are exercised in
// internal/einkauf — this stub only has to keep the gRPC layer honest.
func (r *stubEinkaufRepo) CreateContractCall(_ context.Context, call *einkauf.FrameworkContractCall) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	fc, ok := r.contracts[call.ContractID]
	if !ok || fc.TenantID != call.TenantID {
		return einkauf.ErrContractNotFound
	}
	r.contractCalls[call.ID] = call

	var used float64
	for _, c := range r.contractCalls {
		if c.ContractID != call.ContractID || c.TenantID != call.TenantID {
			continue
		}
		amt, _ := strconv.ParseFloat(c.Amount, 64)
		used += amt
	}
	fc.UsedValue = strconv.FormatFloat(used, 'f', 4, 64)
	return nil
}

func (r *stubEinkaufRepo) ListContractCalls(_ context.Context, tenantID, contractID uuid.UUID) ([]*einkauf.FrameworkContractCall, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*einkauf.FrameworkContractCall
	for _, call := range r.contractCalls {
		if call.ContractID == contractID && call.TenantID == tenantID {
			result = append(result, call)
		}
	}
	return result, nil
}

var _ einkauf.RepositoryExtended = (*stubEinkaufRepo)(nil)

// ============================================================================
// Test server helpers
// ============================================================================

func newTestEinkaufServer() *EinkaufGRPCServer {
	return NewEinkaufGRPCServer(nil)
}

func newEinkaufServerWithRepo(repo *stubEinkaufRepo) *EinkaufGRPCServer {
	svc := einkauf.NewServiceExtended(repo)
	return NewEinkaufGRPCServer(svc)
}

func addStubSupplier(repo *stubEinkaufRepo, tenantID uuid.UUID) *einkauf.Supplier {
	s := &einkauf.Supplier{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      "Test Supplier",
		Email:     "supplier@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.suppliers[s.ID] = s
	return s
}

func addStubPO(repo *stubEinkaufRepo, tenantID, supplierID uuid.UUID, poNumber string, status einkauf.POStatus) *einkauf.PurchaseOrder {
	po := &einkauf.PurchaseOrder{
		ID:          uuid.New(),
		TenantID:    tenantID,
		SupplierID:  supplierID,
		PONumber:    poNumber,
		Status:      status,
		OrderDate:   time.Now(),
		TotalAmount: "0.0000",
		Currency:    "EUR",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.pos[po.ID] = po
	repo.poNumbers[tenantID.String()+":"+poNumber] = po.ID
	return po
}

func addStubPOLine(repo *stubEinkaufRepo, tenantID, poID uuid.UUID, qty, unitPrice string) *einkauf.POLine {
	l := &einkauf.POLine{
		ID:               uuid.New(),
		TenantID:         tenantID,
		POID:             poID,
		ProductName:      "Widget",
		Quantity:         qty,
		UnitPrice:        unitPrice,
		TaxRate:          "0",
		ReceivedQuantity: "0",
		LinePosition:     1,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	repo.lines[l.ID] = l
	return l
}

func addStubContract(repo *stubEinkaufRepo, tenantID, supplierID uuid.UUID, contractNr string, status einkauf.ContractStatus) *einkauf.FrameworkContract {
	fc := &einkauf.FrameworkContract{
		ID:         uuid.New(),
		TenantID:   tenantID,
		SupplierID: supplierID,
		Title:      "Rahmenvertrag",
		ContractNr: contractNr,
		TotalValue: "5000.0000",
		UsedValue:  "0.0000",
		Currency:   "EUR",
		Status:     status,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	repo.contracts[fc.ID] = fc
	if contractNr != "" {
		repo.contractNrs[tenantID.String()+":"+contractNr] = fc.ID
	}
	return fc
}

// ============================================================================
// UUID validation — table test over the nil-service server. These branches
// return before any service call, so a nil svc never panics.
// ============================================================================

func TestEinkauf_UUIDValidation(t *testing.T) {
	s := newTestEinkaufServer()
	ctx := context.Background()
	badID := "not-a-uuid"
	valid := uuid.New().String()

	cases := []struct {
		name string
		call func() error
	}{
		{"CreateSupplier/tenant", func() error {
			_, err := s.CreateSupplier(ctx, &einkaufv1.CreateSupplierRequest{TenantId: badID})
			return err
		}},
		{"CreateSupplier/contact", func() error {
			cid := badID
			_, err := s.CreateSupplier(ctx, &einkaufv1.CreateSupplierRequest{TenantId: valid, Name: "x", ContactId: &cid})
			return err
		}},
		{"UpdateSupplier/tenant", func() error {
			_, err := s.UpdateSupplier(ctx, &einkaufv1.UpdateSupplierRequest{TenantId: badID})
			return err
		}},
		{"UpdateSupplier/supplier", func() error {
			_, err := s.UpdateSupplier(ctx, &einkaufv1.UpdateSupplierRequest{TenantId: valid, SupplierId: badID})
			return err
		}},
		{"UpdateSupplier/contact", func() error {
			cid := badID
			_, err := s.UpdateSupplier(ctx, &einkaufv1.UpdateSupplierRequest{TenantId: valid, SupplierId: valid, ContactId: &cid})
			return err
		}},
		{"DeleteSupplier/tenant", func() error {
			_, err := s.DeleteSupplier(ctx, &einkaufv1.DeleteSupplierRequest{TenantId: badID})
			return err
		}},
		{"DeleteSupplier/supplier", func() error {
			_, err := s.DeleteSupplier(ctx, &einkaufv1.DeleteSupplierRequest{TenantId: valid, SupplierId: badID})
			return err
		}},
		{"GetSupplier/tenant", func() error {
			_, err := s.GetSupplier(ctx, &einkaufv1.GetSupplierRequest{TenantId: badID})
			return err
		}},
		{"GetSupplier/supplier", func() error {
			_, err := s.GetSupplier(ctx, &einkaufv1.GetSupplierRequest{TenantId: valid, SupplierId: badID})
			return err
		}},
		{"ListSuppliers/tenant", func() error {
			_, err := s.ListSuppliers(ctx, &einkaufv1.ListSuppliersRequest{TenantId: badID})
			return err
		}},
		{"CreatePO/tenant", func() error {
			_, err := s.CreatePO(ctx, &einkaufv1.CreatePORequest{TenantId: badID})
			return err
		}},
		{"CreatePO/supplier", func() error {
			_, err := s.CreatePO(ctx, &einkaufv1.CreatePORequest{TenantId: valid, SupplierId: badID})
			return err
		}},
		{"CreatePO/createdBy", func() error {
			cb := badID
			_, err := s.CreatePO(ctx, &einkaufv1.CreatePORequest{TenantId: valid, SupplierId: valid, PoNumber: "x", CreatedBy: &cb})
			return err
		}},
		{"UpdatePO/tenant", func() error {
			_, err := s.UpdatePO(ctx, &einkaufv1.UpdatePORequest{TenantId: badID})
			return err
		}},
		{"UpdatePO/po", func() error {
			_, err := s.UpdatePO(ctx, &einkaufv1.UpdatePORequest{TenantId: valid, PoId: badID})
			return err
		}},
		{"UpdatePO/supplier", func() error {
			sid := badID
			_, err := s.UpdatePO(ctx, &einkaufv1.UpdatePORequest{TenantId: valid, PoId: valid, SupplierId: &sid})
			return err
		}},
		{"DeletePO/tenant", func() error {
			_, err := s.DeletePO(ctx, &einkaufv1.DeletePORequest{TenantId: badID})
			return err
		}},
		{"DeletePO/po", func() error {
			_, err := s.DeletePO(ctx, &einkaufv1.DeletePORequest{TenantId: valid, PoId: badID})
			return err
		}},
		{"GetPO/tenant", func() error {
			_, err := s.GetPO(ctx, &einkaufv1.GetPORequest{TenantId: badID})
			return err
		}},
		{"GetPO/po", func() error {
			_, err := s.GetPO(ctx, &einkaufv1.GetPORequest{TenantId: valid, PoId: badID})
			return err
		}},
		{"ListPOs/tenant", func() error {
			_, err := s.ListPOs(ctx, &einkaufv1.ListPOsRequest{TenantId: badID})
			return err
		}},
		{"ListPOs/supplier", func() error {
			sid := badID
			_, err := s.ListPOs(ctx, &einkaufv1.ListPOsRequest{TenantId: valid, SupplierId: &sid})
			return err
		}},
		{"AddPOLine/tenant", func() error {
			_, err := s.AddPOLine(ctx, &einkaufv1.AddPOLineRequest{TenantId: badID})
			return err
		}},
		{"AddPOLine/po", func() error {
			_, err := s.AddPOLine(ctx, &einkaufv1.AddPOLineRequest{TenantId: valid, PoId: badID})
			return err
		}},
		{"UpdatePOLine/tenant", func() error {
			_, err := s.UpdatePOLine(ctx, &einkaufv1.UpdatePOLineRequest{TenantId: badID})
			return err
		}},
		{"UpdatePOLine/line", func() error {
			_, err := s.UpdatePOLine(ctx, &einkaufv1.UpdatePOLineRequest{TenantId: valid, LineId: badID})
			return err
		}},
		{"DeletePOLine/tenant", func() error {
			_, err := s.DeletePOLine(ctx, &einkaufv1.DeletePOLineRequest{TenantId: badID})
			return err
		}},
		{"DeletePOLine/line", func() error {
			_, err := s.DeletePOLine(ctx, &einkaufv1.DeletePOLineRequest{TenantId: valid, LineId: badID})
			return err
		}},
		{"ListPOLines/tenant", func() error {
			_, err := s.ListPOLines(ctx, &einkaufv1.ListPOLinesRequest{TenantId: badID})
			return err
		}},
		{"ListPOLines/po", func() error {
			_, err := s.ListPOLines(ctx, &einkaufv1.ListPOLinesRequest{TenantId: valid, PoId: badID})
			return err
		}},
		{"SubmitPO/tenant", func() error {
			_, err := s.SubmitPO(ctx, &einkaufv1.SubmitPORequest{TenantId: badID})
			return err
		}},
		{"SubmitPO/po", func() error {
			_, err := s.SubmitPO(ctx, &einkaufv1.SubmitPORequest{TenantId: valid, PoId: badID})
			return err
		}},
		{"CancelPO/tenant", func() error {
			_, err := s.CancelPO(ctx, &einkaufv1.CancelPORequest{TenantId: badID})
			return err
		}},
		{"CancelPO/po", func() error {
			_, err := s.CancelPO(ctx, &einkaufv1.CancelPORequest{TenantId: valid, PoId: badID})
			return err
		}},
		{"ReceiveGoods/tenant", func() error {
			_, err := s.ReceiveGoods(ctx, &einkaufv1.ReceiveGoodsRequest{TenantId: badID})
			return err
		}},
		{"ReceiveGoods/po", func() error {
			_, err := s.ReceiveGoods(ctx, &einkaufv1.ReceiveGoodsRequest{TenantId: valid, PoId: badID})
			return err
		}},
		{"PartialReceive/tenant", func() error {
			_, err := s.PartialReceive(ctx, &einkaufv1.PartialReceiveRequest{TenantId: badID})
			return err
		}},
		{"PartialReceive/po", func() error {
			_, err := s.PartialReceive(ctx, &einkaufv1.PartialReceiveRequest{TenantId: valid, PoId: badID})
			return err
		}},
		{"PartialReceive/item", func() error {
			_, err := s.PartialReceive(ctx, &einkaufv1.PartialReceiveRequest{
				TenantId: valid, PoId: valid,
				Items: []*einkaufv1.PartialReceiveItem{{LineId: badID}},
			})
			return err
		}},
		{"ListCatalogItems/tenant", func() error {
			_, err := s.ListCatalogItems(ctx, &einkaufv1.ListCatalogItemsRequest{TenantId: badID})
			return err
		}},
		{"ListCatalogItems/supplier", func() error {
			sid := badID
			_, err := s.ListCatalogItems(ctx, &einkaufv1.ListCatalogItemsRequest{TenantId: valid, SupplierId: &sid})
			return err
		}},
		{"GetCatalogItem/tenant", func() error {
			_, err := s.GetCatalogItem(ctx, &einkaufv1.GetCatalogItemRequest{TenantId: badID})
			return err
		}},
		{"GetCatalogItem/item", func() error {
			_, err := s.GetCatalogItem(ctx, &einkaufv1.GetCatalogItemRequest{TenantId: valid, ItemId: badID})
			return err
		}},
		{"CreateCatalogItem/tenant", func() error {
			_, err := s.CreateCatalogItem(ctx, &einkaufv1.CreateCatalogItemRequest{TenantId: badID})
			return err
		}},
		{"CreateCatalogItem/supplier", func() error {
			_, err := s.CreateCatalogItem(ctx, &einkaufv1.CreateCatalogItemRequest{TenantId: valid, SupplierId: badID})
			return err
		}},
		{"UpdateCatalogItem/tenant", func() error {
			_, err := s.UpdateCatalogItem(ctx, &einkaufv1.UpdateCatalogItemRequest{TenantId: badID})
			return err
		}},
		{"UpdateCatalogItem/item", func() error {
			_, err := s.UpdateCatalogItem(ctx, &einkaufv1.UpdateCatalogItemRequest{TenantId: valid, ItemId: badID})
			return err
		}},
		{"DeleteCatalogItem/tenant", func() error {
			_, err := s.DeleteCatalogItem(ctx, &einkaufv1.DeleteCatalogItemRequest{TenantId: badID})
			return err
		}},
		{"DeleteCatalogItem/item", func() error {
			_, err := s.DeleteCatalogItem(ctx, &einkaufv1.DeleteCatalogItemRequest{TenantId: valid, ItemId: badID})
			return err
		}},
		{"ListSupplierRatings/tenant", func() error {
			_, err := s.ListSupplierRatings(ctx, &einkaufv1.ListSupplierRatingsRequest{TenantId: badID})
			return err
		}},
		{"ListSupplierRatings/supplier", func() error {
			_, err := s.ListSupplierRatings(ctx, &einkaufv1.ListSupplierRatingsRequest{TenantId: valid, SupplierId: badID})
			return err
		}},
		{"CreateSupplierRating/tenant", func() error {
			_, err := s.CreateSupplierRating(ctx, &einkaufv1.CreateSupplierRatingRequest{TenantId: badID})
			return err
		}},
		{"CreateSupplierRating/supplier", func() error {
			_, err := s.CreateSupplierRating(ctx, &einkaufv1.CreateSupplierRatingRequest{TenantId: valid, SupplierId: badID})
			return err
		}},
		{"CreateSupplierRating/ratedBy", func() error {
			rb := badID
			_, err := s.CreateSupplierRating(ctx, &einkaufv1.CreateSupplierRatingRequest{TenantId: valid, SupplierId: valid, RatedBy: &rb})
			return err
		}},
		{"DeleteSupplierRating/tenant", func() error {
			_, err := s.DeleteSupplierRating(ctx, &einkaufv1.DeleteSupplierRatingRequest{TenantId: badID})
			return err
		}},
		{"DeleteSupplierRating/rating", func() error {
			_, err := s.DeleteSupplierRating(ctx, &einkaufv1.DeleteSupplierRatingRequest{TenantId: valid, RatingId: badID})
			return err
		}},
		{"ListFrameworkContracts/tenant", func() error {
			_, err := s.ListFrameworkContracts(ctx, &einkaufv1.ListFrameworkContractsRequest{TenantId: badID})
			return err
		}},
		{"ListFrameworkContracts/supplier", func() error {
			sid := badID
			_, err := s.ListFrameworkContracts(ctx, &einkaufv1.ListFrameworkContractsRequest{TenantId: valid, SupplierId: &sid})
			return err
		}},
		{"GetFrameworkContract/tenant", func() error {
			_, err := s.GetFrameworkContract(ctx, &einkaufv1.GetFrameworkContractRequest{TenantId: badID})
			return err
		}},
		{"GetFrameworkContract/contract", func() error {
			_, err := s.GetFrameworkContract(ctx, &einkaufv1.GetFrameworkContractRequest{TenantId: valid, ContractId: badID})
			return err
		}},
		{"CreateFrameworkContract/tenant", func() error {
			_, err := s.CreateFrameworkContract(ctx, &einkaufv1.CreateFrameworkContractRequest{TenantId: badID})
			return err
		}},
		{"CreateFrameworkContract/supplier", func() error {
			_, err := s.CreateFrameworkContract(ctx, &einkaufv1.CreateFrameworkContractRequest{TenantId: valid, SupplierId: badID})
			return err
		}},
		{"UpdateFrameworkContract/tenant", func() error {
			_, err := s.UpdateFrameworkContract(ctx, &einkaufv1.UpdateFrameworkContractRequest{TenantId: badID})
			return err
		}},
		{"UpdateFrameworkContract/contract", func() error {
			_, err := s.UpdateFrameworkContract(ctx, &einkaufv1.UpdateFrameworkContractRequest{TenantId: valid, ContractId: badID})
			return err
		}},
		{"UpdateFrameworkContract/supplier", func() error {
			sid := badID
			_, err := s.UpdateFrameworkContract(ctx, &einkaufv1.UpdateFrameworkContractRequest{TenantId: valid, ContractId: valid, SupplierId: &sid})
			return err
		}},
		{"DeleteFrameworkContract/tenant", func() error {
			_, err := s.DeleteFrameworkContract(ctx, &einkaufv1.DeleteFrameworkContractRequest{TenantId: badID})
			return err
		}},
		{"DeleteFrameworkContract/contract", func() error {
			_, err := s.DeleteFrameworkContract(ctx, &einkaufv1.DeleteFrameworkContractRequest{TenantId: valid, ContractId: badID})
			return err
		}},
		{"CreateContractItem/tenant", func() error {
			_, err := s.CreateContractItem(ctx, &einkaufv1.CreateContractItemRequest{TenantId: badID})
			return err
		}},
		{"CreateContractItem/contract", func() error {
			_, err := s.CreateContractItem(ctx, &einkaufv1.CreateContractItemRequest{TenantId: valid, ContractId: badID})
			return err
		}},
		{"UpdateContractItem/tenant", func() error {
			_, err := s.UpdateContractItem(ctx, &einkaufv1.UpdateContractItemRequest{TenantId: badID})
			return err
		}},
		{"UpdateContractItem/item", func() error {
			_, err := s.UpdateContractItem(ctx, &einkaufv1.UpdateContractItemRequest{TenantId: valid, ItemId: badID})
			return err
		}},
		{"DeleteContractItem/tenant", func() error {
			_, err := s.DeleteContractItem(ctx, &einkaufv1.DeleteContractItemRequest{TenantId: badID})
			return err
		}},
		{"DeleteContractItem/item", func() error {
			_, err := s.DeleteContractItem(ctx, &einkaufv1.DeleteContractItemRequest{TenantId: valid, ItemId: badID})
			return err
		}},
		{"CreateContractCall/tenant", func() error {
			_, err := s.CreateContractCall(ctx, &einkaufv1.CreateContractCallRequest{TenantId: badID})
			return err
		}},
		{"CreateContractCall/contract", func() error {
			_, err := s.CreateContractCall(ctx, &einkaufv1.CreateContractCallRequest{TenantId: valid, ContractId: badID})
			return err
		}},
		{"CreateContractCall/po", func() error {
			pid := badID
			_, err := s.CreateContractCall(ctx, &einkaufv1.CreateContractCallRequest{TenantId: valid, ContractId: valid, PoId: &pid})
			return err
		}},
		{"ListContractCalls/tenant", func() error {
			_, err := s.ListContractCalls(ctx, &einkaufv1.ListContractCallsRequest{TenantId: badID})
			return err
		}},
		{"ListContractCalls/contract", func() error {
			_, err := s.ListContractCalls(ctx, &einkaufv1.ListContractCallsRequest{TenantId: valid, ContractId: badID})
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireGRPCCode(t, tc.call(), codes.InvalidArgument)
		})
	}
}

// ============================================================================
// Supplier CRUD
// ============================================================================

func TestEinkauf_SupplierCRUD(t *testing.T) {
	repo := newStubEinkaufRepo()
	srv := newEinkaufServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New()

	created, err := srv.CreateSupplier(ctx, &einkaufv1.CreateSupplierRequest{
		TenantId: tenantID.String(),
		Name:     "Acme GmbH",
		Email:    "info@acme.de",
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.Supplier.Id)
	assert.Equal(t, "Acme GmbH", created.Supplier.Name)

	newName := "Acme Neu GmbH"
	updated, err := srv.UpdateSupplier(ctx, &einkaufv1.UpdateSupplierRequest{
		TenantId:   tenantID.String(),
		SupplierId: created.Supplier.Id,
		Name:       &newName,
	})
	require.NoError(t, err)
	assert.Equal(t, newName, updated.Supplier.Name)

	got, err := srv.GetSupplier(ctx, &einkaufv1.GetSupplierRequest{TenantId: tenantID.String(), SupplierId: created.Supplier.Id})
	require.NoError(t, err)
	assert.Equal(t, newName, got.Supplier.Name)

	list, err := srv.ListSuppliers(ctx, &einkaufv1.ListSuppliersRequest{TenantId: tenantID.String()})
	require.NoError(t, err)
	assert.Equal(t, int32(1), list.Total)
	assert.Len(t, list.Suppliers, 1)

	_, err = srv.DeleteSupplier(ctx, &einkaufv1.DeleteSupplierRequest{TenantId: tenantID.String(), SupplierId: created.Supplier.Id})
	require.NoError(t, err)

	_, err = srv.GetSupplier(ctx, &einkaufv1.GetSupplierRequest{TenantId: tenantID.String(), SupplierId: created.Supplier.Id})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestEinkauf_GetSupplier_NotFound(t *testing.T) {
	repo := newStubEinkaufRepo()
	srv := newEinkaufServerWithRepo(repo)
	_, err := srv.GetSupplier(context.Background(), &einkaufv1.GetSupplierRequest{
		TenantId: uuid.New().String(), SupplierId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestEinkauf_CreateSupplier_EmptyName(t *testing.T) {
	repo := newStubEinkaufRepo()
	srv := newEinkaufServerWithRepo(repo)
	_, err := srv.CreateSupplier(context.Background(), &einkaufv1.CreateSupplierRequest{
		TenantId: uuid.New().String(), Name: "   ",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// Purchase order lifecycle — the invalid-transition cases the scope calls out.
// ============================================================================

func TestEinkauf_PO_Lifecycle_InvalidTransitions(t *testing.T) {
	repo := newStubEinkaufRepo()
	srv := newEinkaufServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New()
	supplier := addStubSupplier(repo, tenantID)

	t.Run("SubmitPO without lines fails", func(t *testing.T) {
		po := addStubPO(repo, tenantID, supplier.ID, "PO-1", einkauf.POStatusDraft)
		_, err := srv.SubmitPO(ctx, &einkaufv1.SubmitPORequest{TenantId: tenantID.String(), PoId: po.ID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("SubmitPO on non-draft fails", func(t *testing.T) {
		po := addStubPO(repo, tenantID, supplier.ID, "PO-2", einkauf.POStatusSubmitted)
		_, err := srv.SubmitPO(ctx, &einkaufv1.SubmitPORequest{TenantId: tenantID.String(), PoId: po.ID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("CancelPO on received fails", func(t *testing.T) {
		po := addStubPO(repo, tenantID, supplier.ID, "PO-3", einkauf.POStatusReceived)
		_, err := srv.CancelPO(ctx, &einkaufv1.CancelPORequest{TenantId: tenantID.String(), PoId: po.ID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("ReceiveGoods on draft fails", func(t *testing.T) {
		po := addStubPO(repo, tenantID, supplier.ID, "PO-4", einkauf.POStatusDraft)
		_, err := srv.ReceiveGoods(ctx, &einkaufv1.ReceiveGoodsRequest{TenantId: tenantID.String(), PoId: po.ID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("PartialReceive on draft fails", func(t *testing.T) {
		po := addStubPO(repo, tenantID, supplier.ID, "PO-5", einkauf.POStatusDraft)
		_, err := srv.PartialReceive(ctx, &einkaufv1.PartialReceiveRequest{TenantId: tenantID.String(), PoId: po.ID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("DeletePO on non-draft fails", func(t *testing.T) {
		po := addStubPO(repo, tenantID, supplier.ID, "PO-6", einkauf.POStatusSubmitted)
		_, err := srv.DeletePO(ctx, &einkaufv1.DeletePORequest{TenantId: tenantID.String(), PoId: po.ID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("UpdatePO on closed fails", func(t *testing.T) {
		po := addStubPO(repo, tenantID, supplier.ID, "PO-7", einkauf.POStatusClosed)
		newNotes := "x"
		_, err := srv.UpdatePO(ctx, &einkaufv1.UpdatePORequest{TenantId: tenantID.String(), PoId: po.ID.String(), Notes: &newNotes})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("PartialReceive exceeding ordered quantity fails", func(t *testing.T) {
		po := addStubPO(repo, tenantID, supplier.ID, "PO-8", einkauf.POStatusSubmitted)
		line := addStubPOLine(repo, tenantID, po.ID, "5", "10.00")
		_, err := srv.PartialReceive(ctx, &einkaufv1.PartialReceiveRequest{
			TenantId: tenantID.String(), PoId: po.ID.String(),
			Items: []*einkaufv1.PartialReceiveItem{{LineId: line.ID.String(), ReceivedQuantity: "6"}},
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
}

func TestEinkauf_PO_FullLifecycle_HappyPath(t *testing.T) {
	repo := newStubEinkaufRepo()
	srv := newEinkaufServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New()
	supplier := addStubSupplier(repo, tenantID)

	created, err := srv.CreatePO(ctx, &einkaufv1.CreatePORequest{
		TenantId:   tenantID.String(),
		SupplierId: supplier.ID.String(),
		PoNumber:   "PO-100",
	})
	require.NoError(t, err)
	assert.Equal(t, "draft", created.Po.Status)

	// Duplicate PO number is rejected.
	_, err = srv.CreatePO(ctx, &einkaufv1.CreatePORequest{
		TenantId: tenantID.String(), SupplierId: supplier.ID.String(), PoNumber: "PO-100",
	})
	requireGRPCCode(t, err, codes.AlreadyExists)

	lineResp, err := srv.AddPOLine(ctx, &einkaufv1.AddPOLineRequest{
		TenantId: tenantID.String(), PoId: created.Po.Id,
		ProductName: "Widget", Quantity: "3", UnitPrice: "3.3333",
	})
	require.NoError(t, err)

	// The recomputed header total must reach the wire as the exact repository
	// string (numeric(15,4) precision), not a value rounded to two decimals
	// by an intermediate float64 conversion.
	afterAdd, err := srv.GetPO(ctx, &einkaufv1.GetPORequest{TenantId: tenantID.String(), PoId: created.Po.Id})
	require.NoError(t, err)
	assert.Equal(t, "9.9999", afterAdd.Po.TotalAmount)

	_, err = srv.UpdatePOLine(ctx, &einkaufv1.UpdatePOLineRequest{
		TenantId: tenantID.String(), LineId: lineResp.Line.Id,
		Quantity: strPtr("2"),
	})
	require.NoError(t, err)

	afterUpdate, err := srv.GetPO(ctx, &einkaufv1.GetPORequest{TenantId: tenantID.String(), PoId: created.Po.Id})
	require.NoError(t, err)
	assert.Equal(t, "6.6666", afterUpdate.Po.TotalAmount)

	lines, err := srv.ListPOLines(ctx, &einkaufv1.ListPOLinesRequest{TenantId: tenantID.String(), PoId: created.Po.Id})
	require.NoError(t, err)
	assert.Len(t, lines.Lines, 1)

	submitted, err := srv.SubmitPO(ctx, &einkaufv1.SubmitPORequest{TenantId: tenantID.String(), PoId: created.Po.Id})
	require.NoError(t, err)
	assert.Equal(t, "submitted", submitted.Po.Status)

	received, err := srv.ReceiveGoods(ctx, &einkaufv1.ReceiveGoodsRequest{TenantId: tenantID.String(), PoId: created.Po.Id})
	require.NoError(t, err)
	assert.Equal(t, "received", received.Po.Status)

	_, err = srv.DeletePOLine(ctx, &einkaufv1.DeletePOLineRequest{TenantId: tenantID.String(), LineId: lineResp.Line.Id})
	require.NoError(t, err)

	afterDelete, err := srv.GetPO(ctx, &einkaufv1.GetPORequest{TenantId: tenantID.String(), PoId: created.Po.Id})
	require.NoError(t, err)
	assert.Equal(t, "0.0000", afterDelete.Po.TotalAmount)
}

func TestEinkauf_PartialReceive_TransitionsToPartiallyReceived(t *testing.T) {
	repo := newStubEinkaufRepo()
	srv := newEinkaufServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New()
	supplier := addStubSupplier(repo, tenantID)
	po := addStubPO(repo, tenantID, supplier.ID, "PO-200", einkauf.POStatusSubmitted)
	line := addStubPOLine(repo, tenantID, po.ID, "10", "1.00")

	resp, err := srv.PartialReceive(ctx, &einkaufv1.PartialReceiveRequest{
		TenantId: tenantID.String(), PoId: po.ID.String(),
		Items: []*einkaufv1.PartialReceiveItem{{LineId: line.ID.String(), ReceivedQuantity: "4"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "partially_received", resp.Po.Status)

	full, err := srv.PartialReceive(ctx, &einkaufv1.PartialReceiveRequest{
		TenantId: tenantID.String(), PoId: po.ID.String(),
		Items: []*einkaufv1.PartialReceiveItem{{LineId: line.ID.String(), ReceivedQuantity: "10"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "received", full.Po.Status)
}

func TestEinkauf_ListPOs_Filtering(t *testing.T) {
	repo := newStubEinkaufRepo()
	srv := newEinkaufServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New()
	supplier := addStubSupplier(repo, tenantID)
	addStubPO(repo, tenantID, supplier.ID, "PO-A", einkauf.POStatusDraft)
	addStubPO(repo, tenantID, supplier.ID, "PO-B", einkauf.POStatusSubmitted)

	draft := "draft"
	list, err := srv.ListPOs(ctx, &einkaufv1.ListPOsRequest{TenantId: tenantID.String(), Status: &draft})
	require.NoError(t, err)
	assert.Equal(t, int32(1), list.Total)
	require.Len(t, list.Pos, 1)
	assert.Equal(t, "PO-A", list.Pos[0].PoNumber)

	bySupplier, err := srv.ListPOs(ctx, &einkaufv1.ListPOsRequest{TenantId: tenantID.String(), SupplierId: strPtr(supplier.ID.String())})
	require.NoError(t, err)
	assert.Equal(t, int32(2), bySupplier.Total)

	from := timestamppb.New(time.Now().Add(-time.Hour))
	to := timestamppb.New(time.Now().Add(time.Hour))
	byDate, err := srv.ListPOs(ctx, &einkaufv1.ListPOsRequest{TenantId: tenantID.String(), DateFrom: from, DateTo: to})
	require.NoError(t, err)
	assert.Equal(t, int32(2), byDate.Total)
}

func TestEinkauf_UpdatePO_AllFields(t *testing.T) {
	repo := newStubEinkaufRepo()
	srv := newEinkaufServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New()
	supplierA := addStubSupplier(repo, tenantID)
	supplierB := addStubSupplier(repo, tenantID)
	po := addStubPO(repo, tenantID, supplierA.ID, "PO-UPD-1", einkauf.POStatusDraft)
	addStubPO(repo, tenantID, supplierA.ID, "PO-UPD-TAKEN", einkauf.POStatusDraft)

	newNumber := "PO-UPD-2"
	newCurrency := "USD"
	newNotes := "updated notes"
	orderDate := timestamppb.New(time.Now())
	expectedDelivery := timestamppb.New(time.Now().Add(72 * time.Hour))

	updated, err := srv.UpdatePO(ctx, &einkaufv1.UpdatePORequest{
		TenantId:             tenantID.String(),
		PoId:                 po.ID.String(),
		SupplierId:           strPtr(supplierB.ID.String()),
		PoNumber:             &newNumber,
		Currency:             &newCurrency,
		Notes:                &newNotes,
		OrderDate:            orderDate,
		ExpectedDeliveryDate: expectedDelivery,
	})
	require.NoError(t, err)
	assert.Equal(t, supplierB.ID.String(), updated.Po.SupplierId)
	assert.Equal(t, newNumber, updated.Po.PoNumber)
	assert.Equal(t, newCurrency, updated.Po.Currency)
	assert.Equal(t, newNotes, updated.Po.Notes)
	require.NotNil(t, updated.Po.ExpectedDeliveryDate)

	// Clearing the expected delivery date via the epoch-pointer convention.
	epoch := timestamppb.New(time.Unix(0, 0))
	cleared, err := srv.UpdatePO(ctx, &einkaufv1.UpdatePORequest{
		TenantId: tenantID.String(), PoId: po.ID.String(), ExpectedDeliveryDate: epoch,
	})
	require.NoError(t, err)
	assert.Nil(t, cleared.Po.ExpectedDeliveryDate)

	// PO number already used by another PO for this tenant.
	takenNumber := "PO-UPD-TAKEN"
	_, err = srv.UpdatePO(ctx, &einkaufv1.UpdatePORequest{TenantId: tenantID.String(), PoId: po.ID.String(), PoNumber: &takenNumber})
	requireGRPCCode(t, err, codes.AlreadyExists)

	// Unknown supplier on update.
	unknownSupplier := uuid.New().String()
	_, err = srv.UpdatePO(ctx, &einkaufv1.UpdatePORequest{TenantId: tenantID.String(), PoId: po.ID.String(), SupplierId: &unknownSupplier})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestEinkauf_UpdatePOLine_AllFields(t *testing.T) {
	repo := newStubEinkaufRepo()
	srv := newEinkaufServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New()
	supplier := addStubSupplier(repo, tenantID)
	po := addStubPO(repo, tenantID, supplier.ID, "PO-LINE-1", einkauf.POStatusDraft)
	line := addStubPOLine(repo, tenantID, po.ID, "1", "1.00")

	newSKU := "SKU-2"
	newTax := "19"
	newPos := int32(2)
	updated, err := srv.UpdatePOLine(ctx, &einkaufv1.UpdatePOLineRequest{
		TenantId: tenantID.String(), LineId: line.ID.String(),
		Sku: &newSKU, TaxRate: &newTax, LinePosition: &newPos,
	})
	require.NoError(t, err)
	assert.Equal(t, newSKU, updated.Line.Sku)
	assert.Equal(t, newTax, updated.Line.TaxRate)
	assert.Equal(t, int32(2), updated.Line.LinePosition)

	emptyName := "   "
	_, err = srv.UpdatePOLine(ctx, &einkaufv1.UpdatePOLineRequest{TenantId: tenantID.String(), LineId: line.ID.String(), ProductName: &emptyName})
	requireGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.UpdatePOLine(ctx, &einkaufv1.UpdatePOLineRequest{TenantId: tenantID.String(), LineId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

// ============================================================================
// Catalog items
// ============================================================================

func TestEinkauf_CatalogItemCRUD(t *testing.T) {
	repo := newStubEinkaufRepo()
	srv := newEinkaufServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New()
	supplier := addStubSupplier(repo, tenantID)

	created, err := srv.CreateCatalogItem(ctx, &einkaufv1.CreateCatalogItemRequest{
		TenantId: tenantID.String(), SupplierId: supplier.ID.String(),
		Name: "Bolt", Price: "1.50",
	})
	require.NoError(t, err)
	assert.True(t, created.Item.Available == false)

	newPrice := "2.00"
	updated, err := srv.UpdateCatalogItem(ctx, &einkaufv1.UpdateCatalogItemRequest{
		TenantId: tenantID.String(), ItemId: created.Item.Id, Price: &newPrice,
	})
	require.NoError(t, err)
	assert.Equal(t, "2.00", updated.Item.Price)

	got, err := srv.GetCatalogItem(ctx, &einkaufv1.GetCatalogItemRequest{TenantId: tenantID.String(), ItemId: created.Item.Id})
	require.NoError(t, err)
	assert.Equal(t, "Bolt", got.Item.Name)

	list, err := srv.ListCatalogItems(ctx, &einkaufv1.ListCatalogItemsRequest{TenantId: tenantID.String()})
	require.NoError(t, err)
	assert.Equal(t, int32(1), list.Total)

	bySupplier, err := srv.ListCatalogItems(ctx, &einkaufv1.ListCatalogItemsRequest{TenantId: tenantID.String(), SupplierId: strPtr(supplier.ID.String())})
	require.NoError(t, err)
	assert.Equal(t, int32(1), bySupplier.Total)

	unavailable := false
	byAvailability, err := srv.ListCatalogItems(ctx, &einkaufv1.ListCatalogItemsRequest{TenantId: tenantID.String(), Available: &unavailable})
	require.NoError(t, err)
	assert.Equal(t, int32(1), byAvailability.Total)

	_, err = srv.DeleteCatalogItem(ctx, &einkaufv1.DeleteCatalogItemRequest{TenantId: tenantID.String(), ItemId: created.Item.Id})
	require.NoError(t, err)

	_, err = srv.GetCatalogItem(ctx, &einkaufv1.GetCatalogItemRequest{TenantId: tenantID.String(), ItemId: created.Item.Id})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestEinkauf_CreateCatalogItem_InvalidPrice(t *testing.T) {
	repo := newStubEinkaufRepo()
	srv := newEinkaufServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New()
	supplier := addStubSupplier(repo, tenantID)

	_, err := srv.CreateCatalogItem(ctx, &einkaufv1.CreateCatalogItemRequest{
		TenantId: tenantID.String(), SupplierId: supplier.ID.String(), Name: "Bolt", Price: "-1",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// Supplier ratings
// ============================================================================

func TestEinkauf_SupplierRating_CreateListDelete(t *testing.T) {
	repo := newStubEinkaufRepo()
	srv := newEinkaufServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New()
	supplier := addStubSupplier(repo, tenantID)

	created, err := srv.CreateSupplierRating(ctx, &einkaufv1.CreateSupplierRatingRequest{
		TenantId: tenantID.String(), SupplierId: supplier.ID.String(),
		Category: "quality", Rating: 4,
	})
	require.NoError(t, err)
	assert.Equal(t, int32(4), created.Rating.Rating)

	list, err := srv.ListSupplierRatings(ctx, &einkaufv1.ListSupplierRatingsRequest{TenantId: tenantID.String(), SupplierId: supplier.ID.String()})
	require.NoError(t, err)
	assert.Len(t, list.Ratings, 1)

	_, err = srv.DeleteSupplierRating(ctx, &einkaufv1.DeleteSupplierRatingRequest{TenantId: tenantID.String(), RatingId: created.Rating.Id})
	require.NoError(t, err)

	list2, err := srv.ListSupplierRatings(ctx, &einkaufv1.ListSupplierRatingsRequest{TenantId: tenantID.String(), SupplierId: supplier.ID.String()})
	require.NoError(t, err)
	assert.Empty(t, list2.Ratings)
}

func TestEinkauf_CreateSupplierRating_OutOfRange(t *testing.T) {
	repo := newStubEinkaufRepo()
	srv := newEinkaufServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New()
	supplier := addStubSupplier(repo, tenantID)

	_, err := srv.CreateSupplierRating(ctx, &einkaufv1.CreateSupplierRatingRequest{
		TenantId: tenantID.String(), SupplierId: supplier.ID.String(), Category: "quality", Rating: 9,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestEinkauf_CreateSupplierRating_InvalidCategory(t *testing.T) {
	repo := newStubEinkaufRepo()
	srv := newEinkaufServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New()
	supplier := addStubSupplier(repo, tenantID)

	_, err := srv.CreateSupplierRating(ctx, &einkaufv1.CreateSupplierRatingRequest{
		TenantId: tenantID.String(), SupplierId: supplier.ID.String(), Category: "not-a-category", Rating: 3,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// Framework contracts, items, calls
// ============================================================================

func TestEinkauf_FrameworkContract_CRUD(t *testing.T) {
	repo := newStubEinkaufRepo()
	srv := newEinkaufServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New()
	supplier := addStubSupplier(repo, tenantID)

	created, err := srv.CreateFrameworkContract(ctx, &einkaufv1.CreateFrameworkContractRequest{
		TenantId: tenantID.String(), SupplierId: supplier.ID.String(),
		Title: "Rahmenvertrag 2026", ContractNr: "RV-1", TotalValue: "5000",
	})
	require.NoError(t, err)
	assert.Equal(t, "draft", created.Contract.Status)

	_, err = srv.CreateFrameworkContract(ctx, &einkaufv1.CreateFrameworkContractRequest{
		TenantId: tenantID.String(), SupplierId: supplier.ID.String(), Title: "Duplicate", ContractNr: "RV-1",
	})
	requireGRPCCode(t, err, codes.AlreadyExists)

	newTitle := "Rahmenvertrag 2026 v2"
	updated, err := srv.UpdateFrameworkContract(ctx, &einkaufv1.UpdateFrameworkContractRequest{
		TenantId: tenantID.String(), ContractId: created.Contract.Id, Title: &newTitle,
	})
	require.NoError(t, err)
	assert.Equal(t, newTitle, updated.Contract.Title)

	got, err := srv.GetFrameworkContract(ctx, &einkaufv1.GetFrameworkContractRequest{TenantId: tenantID.String(), ContractId: created.Contract.Id})
	require.NoError(t, err)
	assert.Equal(t, newTitle, got.Contract.Title)

	list, err := srv.ListFrameworkContracts(ctx, &einkaufv1.ListFrameworkContractsRequest{TenantId: tenantID.String()})
	require.NoError(t, err)
	assert.Equal(t, int32(1), list.Total)

	bySupplier, err := srv.ListFrameworkContracts(ctx, &einkaufv1.ListFrameworkContractsRequest{TenantId: tenantID.String(), SupplierId: strPtr(supplier.ID.String())})
	require.NoError(t, err)
	assert.Equal(t, int32(1), bySupplier.Total)

	activeStatus := "active"
	byStatus, err := srv.ListFrameworkContracts(ctx, &einkaufv1.ListFrameworkContractsRequest{TenantId: tenantID.String(), Status: &activeStatus})
	require.NoError(t, err)
	assert.Equal(t, int32(0), byStatus.Total)

	_, err = srv.DeleteFrameworkContract(ctx, &einkaufv1.DeleteFrameworkContractRequest{TenantId: tenantID.String(), ContractId: created.Contract.Id})
	require.NoError(t, err)

	_, err = srv.GetFrameworkContract(ctx, &einkaufv1.GetFrameworkContractRequest{TenantId: tenantID.String(), ContractId: created.Contract.Id})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestEinkauf_UpdateFrameworkContract_AllFields(t *testing.T) {
	repo := newStubEinkaufRepo()
	srv := newEinkaufServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New()
	supplierA := addStubSupplier(repo, tenantID)
	supplierB := addStubSupplier(repo, tenantID)
	contract := addStubContract(repo, tenantID, supplierA.ID, "RV-UPD-1", einkauf.ContractStatusDraft)
	addStubContract(repo, tenantID, supplierA.ID, "RV-UPD-TAKEN", einkauf.ContractStatusDraft)

	newNr := "RV-UPD-2"
	newTotal := "9999.99"
	newCurrency := "USD"
	newStatus := "active"
	startDate := timestamppb.New(time.Now())
	endDate := timestamppb.New(time.Now().Add(365 * 24 * time.Hour))

	updated, err := srv.UpdateFrameworkContract(ctx, &einkaufv1.UpdateFrameworkContractRequest{
		TenantId:   tenantID.String(),
		ContractId: contract.ID.String(),
		SupplierId: strPtr(supplierB.ID.String()),
		ContractNr: &newNr,
		TotalValue: &newTotal,
		Currency:   &newCurrency,
		Status:     &newStatus,
		StartDate:  startDate,
		EndDate:    endDate,
	})
	require.NoError(t, err)
	assert.Equal(t, supplierB.ID.String(), updated.Contract.SupplierId)
	assert.Equal(t, newNr, updated.Contract.ContractNr)
	assert.Equal(t, newTotal, updated.Contract.TotalValue)
	assert.Equal(t, newCurrency, updated.Contract.Currency)
	assert.Equal(t, "active", updated.Contract.Status)
	require.NotNil(t, updated.Contract.StartDate)
	require.NotNil(t, updated.Contract.EndDate)

	// Clearing start/end dates via the epoch-pointer convention.
	epoch := timestamppb.New(time.Unix(0, 0))
	cleared, err := srv.UpdateFrameworkContract(ctx, &einkaufv1.UpdateFrameworkContractRequest{
		TenantId: tenantID.String(), ContractId: contract.ID.String(), StartDate: epoch, EndDate: epoch,
	})
	require.NoError(t, err)
	assert.Nil(t, cleared.Contract.StartDate)
	assert.Nil(t, cleared.Contract.EndDate)

	// Contract number already used by another contract for this tenant.
	takenNr := "RV-UPD-TAKEN"
	_, err = srv.UpdateFrameworkContract(ctx, &einkaufv1.UpdateFrameworkContractRequest{TenantId: tenantID.String(), ContractId: contract.ID.String(), ContractNr: &takenNr})
	requireGRPCCode(t, err, codes.AlreadyExists)

	emptyTitle := "   "
	_, err = srv.UpdateFrameworkContract(ctx, &einkaufv1.UpdateFrameworkContractRequest{TenantId: tenantID.String(), ContractId: contract.ID.String(), Title: &emptyTitle})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestEinkauf_ContractItem_CreateUpdateDelete(t *testing.T) {
	repo := newStubEinkaufRepo()
	srv := newEinkaufServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New()
	supplier := addStubSupplier(repo, tenantID)
	contract := addStubContract(repo, tenantID, supplier.ID, "RV-2", einkauf.ContractStatusActive)

	created, err := srv.CreateContractItem(ctx, &einkaufv1.CreateContractItemRequest{
		TenantId: tenantID.String(), ContractId: contract.ID.String(), Name: "Schraube", UnitPrice: "0.10", AgreedQty: "1000",
	})
	require.NoError(t, err)
	assert.Equal(t, "Schraube", created.Item.Name)

	// UpdateContractItem relies on the unexported QueryRowContractItem lookup —
	// without wiring it into the stub, this call falls through to
	// ErrContractItemNotFound (mutation-probe target below covers the
	// alternative failure mode).
	newName := "Schraube M4"
	updated, err := srv.UpdateContractItem(ctx, &einkaufv1.UpdateContractItemRequest{
		TenantId: tenantID.String(), ItemId: created.Item.Id, Name: &newName,
	})
	require.NoError(t, err)
	assert.Equal(t, newName, updated.Item.Name)

	_, err = srv.DeleteContractItem(ctx, &einkaufv1.DeleteContractItemRequest{TenantId: tenantID.String(), ItemId: created.Item.Id})
	require.NoError(t, err)
}

func TestEinkauf_UpdateContractItem_NotFound(t *testing.T) {
	repo := newStubEinkaufRepo()
	srv := newEinkaufServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New()

	newName := "x"
	_, err := srv.UpdateContractItem(ctx, &einkaufv1.UpdateContractItemRequest{
		TenantId: tenantID.String(), ItemId: uuid.New().String(), Name: &newName,
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestEinkauf_ContractCall_CreateAndList(t *testing.T) {
	repo := newStubEinkaufRepo()
	srv := newEinkaufServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New()
	supplier := addStubSupplier(repo, tenantID)
	contract := addStubContract(repo, tenantID, supplier.ID, "RV-3", einkauf.ContractStatusActive)

	created, err := srv.CreateContractCall(ctx, &einkaufv1.CreateContractCallRequest{
		TenantId: tenantID.String(), ContractId: contract.ID.String(), Amount: "500.5",
	})
	require.NoError(t, err)
	assert.Equal(t, "500.5", created.Call.Amount)

	list, err := srv.ListContractCalls(ctx, &einkaufv1.ListContractCallsRequest{TenantId: tenantID.String(), ContractId: contract.ID.String()})
	require.NoError(t, err)
	require.Len(t, list.Calls, 1)

	// used_value is recomputed inside the same repository call.
	refreshed, err := srv.GetFrameworkContract(ctx, &einkaufv1.GetFrameworkContractRequest{TenantId: tenantID.String(), ContractId: contract.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, "500.5000", refreshed.Contract.UsedValue)
}

func TestEinkauf_CreateContractCall_NegativeAmount(t *testing.T) {
	repo := newStubEinkaufRepo()
	srv := newEinkaufServerWithRepo(repo)
	ctx := context.Background()
	tenantID := uuid.New()
	supplier := addStubSupplier(repo, tenantID)
	contract := addStubContract(repo, tenantID, supplier.ID, "RV-4", einkauf.ContractStatusActive)

	_, err := srv.CreateContractCall(ctx, &einkaufv1.CreateContractCallRequest{
		TenantId: tenantID.String(), ContractId: contract.ID.String(), Amount: "-5",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// Conversion helpers — nil-safety and wire-shape.
// ============================================================================

func TestEinkauf_ToProto_NilSafety(t *testing.T) {
	assert.Nil(t, supplierToProto(nil))
	assert.Nil(t, poToProto(nil))
	assert.Nil(t, lineToProto(nil))
	assert.Nil(t, catalogItemToProto(nil))
	assert.Nil(t, supplierRatingToProto(nil))
	assert.Nil(t, frameworkContractToProto(nil))
	assert.Nil(t, contractItemToProto(nil))
	assert.Nil(t, contractCallToProto(nil))
}

func TestEinkauf_PoToProto_EmbedsLinesAndOptionalFields(t *testing.T) {
	poID := uuid.New()
	tenantID := uuid.New()
	supplierID := uuid.New()
	createdBy := uuid.New()
	expected := time.Now().Add(48 * time.Hour)

	po := &einkauf.PurchaseOrder{
		ID:                   poID,
		TenantID:             tenantID,
		SupplierID:           supplierID,
		PONumber:             "PO-XYZ",
		Status:               einkauf.POStatusDraft,
		OrderDate:            time.Now(),
		ExpectedDeliveryDate: &expected,
		TotalAmount:          "1234.5678",
		Currency:             "EUR",
		CreatedBy:            &createdBy,
		Lines: []*einkauf.POLine{
			{ID: uuid.New(), TenantID: tenantID, POID: poID, ProductName: "A", Quantity: "1", UnitPrice: "1"},
		},
	}

	p := poToProto(po)
	require.NotNil(t, p)
	assert.Equal(t, "1234.5678", p.TotalAmount)
	require.Len(t, p.Lines, 1)
	require.NotNil(t, p.CreatedBy)
	assert.Equal(t, createdBy.String(), *p.CreatedBy)
	require.NotNil(t, p.ExpectedDeliveryDate)
	assert.Equal(t, timestamppb.New(expected).AsTime(), p.ExpectedDeliveryDate.AsTime())
}

// ============================================================================
// mapEinkaufError — table test over every sentinel plus the default branch.
// ============================================================================

func TestMapEinkaufError_Table(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"nil", nil, codes.OK},
		{"supplier not found", einkauf.ErrSupplierNotFound, codes.NotFound},
		{"po not found", einkauf.ErrPONotFound, codes.NotFound},
		{"po line not found", einkauf.ErrPOLineNotFound, codes.NotFound},
		{"po number taken", einkauf.ErrPONumberTaken, codes.AlreadyExists},
		{"invalid input", einkauf.ErrInvalidInput, codes.InvalidArgument},
		{"po not draft", einkauf.ErrPONotDraft, codes.FailedPrecondition},
		{"po not submittable", einkauf.ErrPONotSubmittable, codes.FailedPrecondition},
		{"po not receivable", einkauf.ErrPONotReceivable, codes.FailedPrecondition},
		{"po not cancellable", einkauf.ErrPONotCancellable, codes.FailedPrecondition},
		{"invalid quantity", einkauf.ErrInvalidQuantity, codes.InvalidArgument},
		{"exceeds ordered qty", einkauf.ErrExceedsOrderedQty, codes.InvalidArgument},
		{"catalog item not found", einkauf.ErrCatalogItemNotFound, codes.NotFound},
		{"supplier rating not found", einkauf.ErrSupplierRatingNotFound, codes.NotFound},
		{"duplicate rating", einkauf.ErrDuplicateRating, codes.AlreadyExists},
		{"contract not found", einkauf.ErrContractNotFound, codes.NotFound},
		{"contract item not found", einkauf.ErrContractItemNotFound, codes.NotFound},
		{"contract nr taken", einkauf.ErrContractNrTaken, codes.AlreadyExists},
		{"contract call not found", einkauf.ErrContractCallNotFound, codes.NotFound},
		{"unmapped error falls through to Internal", assertUnmappedErr, codes.Internal},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := mapEinkaufError(tc.err)
			if tc.err == nil {
				assert.NoError(t, err)
				return
			}
			requireGRPCCode(t, err, tc.code)
		})
	}
}

var assertUnmappedErr = &customEinkaufErr{}

type customEinkaufErr struct{}

func (e *customEinkaufErr) Error() string { return "some unrelated wrapped failure" }
