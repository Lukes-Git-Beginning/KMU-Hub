package einkauf

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// InventarAdjuster is a narrow interface that the einkauf service uses to
// record incoming stock movements in the inventar module on goods receipt.
// The implementation resolves the inventar item by SKU and calls AdjustStock.
// If the SKU is not found in inventar (item not yet created there), the
// call is a no-op so that goods receipt never fails due to inventar state
// (graceful degradation per architecture rule 8).
type InventarAdjuster interface {
	// AdjustStockBySKU increments the inventar item identified by sku by
	// delta units. Returns nil if the SKU is not found (soft-fail).
	AdjustStockBySKU(ctx context.Context, tenantID uuid.UUID, sku string, delta int64, reason string) error
}

// Service handles einkauf (purchasing) business logic.
type Service struct {
	repo    Repository
	// repoExt is non-nil when the service was created via NewServiceExtended.
	// It grants access to the catalog, ratings, and framework contract persistence.
	repoExt RepositoryExtended
	// inventarAdj is optional; if non-nil, ReceiveGoods and PartialReceive
	// will trigger inventory movements in the inventar module.
	inventarAdj InventarAdjuster
}

// NewService creates a new einkauf service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// WithInventarAdjuster attaches an InventarAdjuster to the service so that
// goods-receipt operations also update the inventar module's stock levels.
func (s *Service) WithInventarAdjuster(adj InventarAdjuster) *Service {
	s.inventarAdj = adj
	return s
}

// ============================================================================
// Input types — Suppliers
// ============================================================================

// CreateSupplierInput contains data for creating a supplier.
type CreateSupplierInput struct {
	TenantID     uuid.UUID
	Name         string
	ContactID    *uuid.UUID
	Email        string
	Phone        string
	Address      string
	PaymentTerms string
	Notes        string
}

// UpdateSupplierInput contains data for updating a supplier.
type UpdateSupplierInput struct {
	TenantID     uuid.UUID
	SupplierID   uuid.UUID
	Name         *string
	ContactID    *uuid.UUID // uuid.Nil to clear
	Email        *string
	Phone        *string
	Address      *string
	PaymentTerms *string
	Notes        *string
}

// ListSuppliersInput contains options for listing suppliers.
type ListSuppliersInput struct {
	TenantID   uuid.UUID
	Search     string
	ActiveOnly bool
	Page       int
	PageSize   int
}

// ============================================================================
// Input types — Purchase Orders
// ============================================================================

// CreatePOInput contains data for creating a purchase order.
type CreatePOInput struct {
	TenantID             uuid.UUID
	SupplierID           uuid.UUID
	PONumber             string
	OrderDate            time.Time
	ExpectedDeliveryDate *time.Time
	Currency             string
	Notes                string
	CreatedBy            *uuid.UUID
}

// UpdatePOInput contains data for updating a purchase order.
type UpdatePOInput struct {
	TenantID             uuid.UUID
	POID                 uuid.UUID
	SupplierID           *uuid.UUID
	PONumber             *string
	OrderDate            *time.Time
	ExpectedDeliveryDate *time.Time // send zero-value pointer to clear
	Currency             *string
	Notes                *string
}

// ListPOsInput contains options for listing purchase orders.
type ListPOsInput struct {
	TenantID   uuid.UUID
	SupplierID *uuid.UUID
	Status     *POStatus
	DateFrom   *time.Time
	DateTo     *time.Time
	Page       int
	PageSize   int
}

// ============================================================================
// Input types — PO Lines
// ============================================================================

// AddPOLineInput contains data for adding a line to a PO.
type AddPOLineInput struct {
	TenantID     uuid.UUID
	POID         uuid.UUID
	ProductName  string
	SKU          string
	Quantity     string
	UnitPrice    string
	TaxRate      string
	LinePosition int
}

// UpdatePOLineInput contains data for updating a PO line.
type UpdatePOLineInput struct {
	TenantID     uuid.UUID
	LineID       uuid.UUID
	ProductName  *string
	SKU          *string
	Quantity     *string
	UnitPrice    *string
	TaxRate      *string
	LinePosition *int
}

// PartialReceiveItem maps a line ID to its received quantity.
type PartialReceiveItem struct {
	LineID           uuid.UUID
	ReceivedQuantity string
}

// ============================================================================
// Supplier methods
// ============================================================================

// CreateSupplier creates a new supplier.
func (s *Service) CreateSupplier(ctx context.Context, input CreateSupplierInput) (*Supplier, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrInvalidInput
	}

	now := time.Now()
	supplier := &Supplier{
		ID:           uuid.New(),
		TenantID:     input.TenantID,
		Name:         name,
		ContactID:    input.ContactID,
		Email:        strings.TrimSpace(input.Email),
		Phone:        strings.TrimSpace(input.Phone),
		Address:      strings.TrimSpace(input.Address),
		PaymentTerms: strings.TrimSpace(input.PaymentTerms),
		Notes:        strings.TrimSpace(input.Notes),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.CreateSupplier(ctx, supplier); err != nil {
		return nil, fmt.Errorf("create supplier: %w", err)
	}

	slog.Info("einkauf supplier created", "supplier_id", supplier.ID, "tenant_id", supplier.TenantID, "name", supplier.Name)
	return supplier, nil
}

// UpdateSupplier updates an existing supplier.
func (s *Service) UpdateSupplier(ctx context.Context, input UpdateSupplierInput) (*Supplier, error) {
	supplier, err := s.repo.GetSupplier(ctx, input.TenantID, input.SupplierID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, ErrInvalidInput
		}
		supplier.Name = name
	}
	if input.ContactID != nil {
		if *input.ContactID == uuid.Nil {
			supplier.ContactID = nil
		} else {
			supplier.ContactID = input.ContactID
		}
	}
	if input.Email != nil {
		supplier.Email = strings.TrimSpace(*input.Email)
	}
	if input.Phone != nil {
		supplier.Phone = strings.TrimSpace(*input.Phone)
	}
	if input.Address != nil {
		supplier.Address = strings.TrimSpace(*input.Address)
	}
	if input.PaymentTerms != nil {
		supplier.PaymentTerms = strings.TrimSpace(*input.PaymentTerms)
	}
	if input.Notes != nil {
		supplier.Notes = strings.TrimSpace(*input.Notes)
	}

	supplier.UpdatedAt = time.Now()

	if updateErr := s.repo.UpdateSupplier(ctx, supplier); updateErr != nil {
		return nil, fmt.Errorf("update supplier: %w", updateErr)
	}

	slog.Info("einkauf supplier updated", "supplier_id", supplier.ID, "tenant_id", supplier.TenantID)
	return supplier, nil
}

// DeleteSupplier soft-deletes a supplier.
func (s *Service) DeleteSupplier(ctx context.Context, tenantID, supplierID uuid.UUID) error {
	if _, err := s.repo.GetSupplier(ctx, tenantID, supplierID); err != nil {
		return err
	}
	if err := s.repo.DeleteSupplier(ctx, tenantID, supplierID); err != nil {
		return fmt.Errorf("delete supplier: %w", err)
	}
	slog.Info("einkauf supplier deleted", "supplier_id", supplierID, "tenant_id", tenantID)
	return nil
}

// GetSupplier retrieves a supplier by ID.
func (s *Service) GetSupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (*Supplier, error) {
	return s.repo.GetSupplier(ctx, tenantID, supplierID)
}

// ListSuppliers retrieves suppliers with optional filtering and pagination.
func (s *Service) ListSuppliers(ctx context.Context, input ListSuppliersInput) ([]*Supplier, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}
	offset := (input.Page - 1) * input.PageSize

	return s.repo.ListSuppliers(ctx, input.TenantID, ListSuppliersFilter{
		Search:     input.Search,
		ActiveOnly: input.ActiveOnly,
	}, offset, input.PageSize)
}

// ============================================================================
// Purchase Order methods
// ============================================================================

// CreatePO creates a new purchase order in draft status.
func (s *Service) CreatePO(ctx context.Context, input CreatePOInput) (*PurchaseOrder, error) {
	if input.PONumber == "" {
		return nil, ErrInvalidInput
	}

	// Validate supplier exists
	if _, err := s.repo.GetSupplier(ctx, input.TenantID, input.SupplierID); err != nil {
		return nil, err
	}

	// Validate PO number uniqueness
	exists, err := s.repo.PONumberExists(ctx, input.TenantID, input.PONumber, nil)
	if err != nil {
		return nil, fmt.Errorf("check po number: %w", err)
	}
	if exists {
		return nil, ErrPONumberTaken
	}

	currency := strings.TrimSpace(input.Currency)
	if currency == "" {
		currency = "EUR"
	}

	orderDate := input.OrderDate
	if orderDate.IsZero() {
		orderDate = time.Now()
	}

	now := time.Now()
	po := &PurchaseOrder{
		ID:                   uuid.New(),
		TenantID:             input.TenantID,
		SupplierID:           input.SupplierID,
		PONumber:             input.PONumber,
		Status:               POStatusDraft,
		OrderDate:            orderDate,
		ExpectedDeliveryDate: input.ExpectedDeliveryDate,
		TotalAmount:          "0",
		Currency:             currency,
		Notes:                strings.TrimSpace(input.Notes),
		CreatedBy:            input.CreatedBy,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := s.repo.CreatePO(ctx, po); err != nil {
		return nil, fmt.Errorf("create purchase order: %w", err)
	}

	slog.Info("einkauf po created", "po_id", po.ID, "tenant_id", po.TenantID, "po_number", po.PONumber)
	return po, nil
}

// UpdatePO updates an existing purchase order (only allowed in non-closed states).
func (s *Service) UpdatePO(ctx context.Context, input UpdatePOInput) (*PurchaseOrder, error) {
	po, err := s.repo.GetPO(ctx, input.TenantID, input.POID)
	if err != nil {
		return nil, err
	}

	if po.Status == POStatusClosed || po.Status == POStatusCancelled {
		return nil, ErrPONotDraft
	}

	if input.SupplierID != nil {
		if _, sErr := s.repo.GetSupplier(ctx, input.TenantID, *input.SupplierID); sErr != nil {
			return nil, sErr
		}
		po.SupplierID = *input.SupplierID
	}

	if input.PONumber != nil {
		newNum := strings.TrimSpace(*input.PONumber)
		if newNum != "" && newNum != po.PONumber {
			numExists, numErr := s.repo.PONumberExists(ctx, input.TenantID, newNum, &po.ID)
			if numErr != nil {
				return nil, fmt.Errorf("check po number: %w", numErr)
			}
			if numExists {
				return nil, ErrPONumberTaken
			}
			po.PONumber = newNum
		}
	}

	if input.OrderDate != nil {
		po.OrderDate = *input.OrderDate
	}
	if input.ExpectedDeliveryDate != nil {
		if input.ExpectedDeliveryDate.IsZero() {
			po.ExpectedDeliveryDate = nil
		} else {
			po.ExpectedDeliveryDate = input.ExpectedDeliveryDate
		}
	}
	if input.Currency != nil {
		po.Currency = strings.TrimSpace(*input.Currency)
	}
	if input.Notes != nil {
		po.Notes = strings.TrimSpace(*input.Notes)
	}

	po.UpdatedAt = time.Now()

	if updateErr := s.repo.UpdatePO(ctx, po); updateErr != nil {
		return nil, fmt.Errorf("update purchase order: %w", updateErr)
	}

	slog.Info("einkauf po updated", "po_id", po.ID, "tenant_id", po.TenantID)
	return po, nil
}

// DeletePO deletes a purchase order — only allowed when status is draft.
func (s *Service) DeletePO(ctx context.Context, tenantID, poID uuid.UUID) error {
	po, err := s.repo.GetPO(ctx, tenantID, poID)
	if err != nil {
		return err
	}
	if po.Status != POStatusDraft {
		return ErrPONotDraft
	}
	if delErr := s.repo.DeletePO(ctx, tenantID, poID); delErr != nil {
		return fmt.Errorf("delete purchase order: %w", delErr)
	}
	slog.Info("einkauf po deleted", "po_id", poID, "tenant_id", tenantID)
	return nil
}

// GetPO retrieves a purchase order by ID with its lines populated.
func (s *Service) GetPO(ctx context.Context, tenantID, poID uuid.UUID) (*PurchaseOrder, error) {
	return s.repo.GetPOWithLines(ctx, tenantID, poID)
}

// ListPOs retrieves purchase orders with optional filtering and pagination.
func (s *Service) ListPOs(ctx context.Context, input ListPOsInput) ([]*PurchaseOrder, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 100 {
		input.PageSize = 20
	}
	offset := (input.Page - 1) * input.PageSize

	return s.repo.ListPOs(ctx, input.TenantID, ListPOsFilter{
		SupplierID: input.SupplierID,
		Status:     input.Status,
		DateFrom:   input.DateFrom,
		DateTo:     input.DateTo,
	}, offset, input.PageSize)
}

// ============================================================================
// PO Line methods
// ============================================================================

// AddPOLine adds a line item to a purchase order.
func (s *Service) AddPOLine(ctx context.Context, input AddPOLineInput) (*POLine, error) {
	if strings.TrimSpace(input.ProductName) == "" {
		return nil, ErrInvalidInput
	}
	if strings.TrimSpace(input.Quantity) == "" {
		return nil, ErrInvalidInput
	}
	if strings.TrimSpace(input.UnitPrice) == "" {
		return nil, ErrInvalidInput
	}

	// Ensure PO exists
	if _, err := s.repo.GetPO(ctx, input.TenantID, input.POID); err != nil {
		return nil, err
	}

	taxRate := strings.TrimSpace(input.TaxRate)
	if taxRate == "" {
		taxRate = "0"
	}

	now := time.Now()
	line := &POLine{
		ID:               uuid.New(),
		TenantID:         input.TenantID,
		POID:             input.POID,
		ProductName:      strings.TrimSpace(input.ProductName),
		SKU:              strings.TrimSpace(input.SKU),
		Quantity:         strings.TrimSpace(input.Quantity),
		UnitPrice:        strings.TrimSpace(input.UnitPrice),
		TaxRate:          taxRate,
		ReceivedQuantity: "0",
		LinePosition:     input.LinePosition,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.repo.CreatePOLine(ctx, line); err != nil {
		return nil, fmt.Errorf("add po line: %w", err)
	}

	if err := s.repo.RecomputePOTotal(ctx, input.TenantID, input.POID); err != nil {
		return nil, fmt.Errorf("recompute po total: %w", err)
	}

	slog.Info("einkauf po line added", "line_id", line.ID, "po_id", line.POID, "tenant_id", line.TenantID)
	return line, nil
}

// UpdatePOLine updates an existing PO line.
func (s *Service) UpdatePOLine(ctx context.Context, input UpdatePOLineInput) (*POLine, error) {
	line, err := s.repo.GetPOLine(ctx, input.TenantID, input.LineID)
	if err != nil {
		return nil, err
	}

	if input.ProductName != nil {
		name := strings.TrimSpace(*input.ProductName)
		if name == "" {
			return nil, ErrInvalidInput
		}
		line.ProductName = name
	}
	if input.SKU != nil {
		line.SKU = strings.TrimSpace(*input.SKU)
	}
	if input.Quantity != nil {
		line.Quantity = strings.TrimSpace(*input.Quantity)
	}
	if input.UnitPrice != nil {
		line.UnitPrice = strings.TrimSpace(*input.UnitPrice)
	}
	if input.TaxRate != nil {
		line.TaxRate = strings.TrimSpace(*input.TaxRate)
	}
	if input.LinePosition != nil {
		line.LinePosition = *input.LinePosition
	}

	line.UpdatedAt = time.Now()

	if updateErr := s.repo.UpdatePOLine(ctx, line); updateErr != nil {
		return nil, fmt.Errorf("update po line: %w", updateErr)
	}

	if err := s.repo.RecomputePOTotal(ctx, input.TenantID, line.POID); err != nil {
		return nil, fmt.Errorf("recompute po total: %w", err)
	}

	slog.Info("einkauf po line updated", "line_id", line.ID, "tenant_id", input.TenantID)
	return line, nil
}

// DeletePOLine removes a line item from a PO.
func (s *Service) DeletePOLine(ctx context.Context, tenantID, lineID uuid.UUID) error {
	line, err := s.repo.GetPOLine(ctx, tenantID, lineID)
	if err != nil {
		return err
	}
	if delErr := s.repo.DeletePOLine(ctx, tenantID, lineID); delErr != nil {
		return fmt.Errorf("delete po line: %w", delErr)
	}
	if err := s.repo.RecomputePOTotal(ctx, tenantID, line.POID); err != nil {
		return fmt.Errorf("recompute po total: %w", err)
	}
	slog.Info("einkauf po line deleted", "line_id", lineID, "tenant_id", tenantID)
	return nil
}

// ListPOLines returns all lines for a given PO.
func (s *Service) ListPOLines(ctx context.Context, tenantID, poID uuid.UUID) ([]*POLine, error) {
	return s.repo.ListPOLines(ctx, tenantID, poID)
}

// ============================================================================
// Workflow methods
// ============================================================================

// SubmitPO transitions a PO from draft → submitted.
// Validates that the PO has at least one line.
func (s *Service) SubmitPO(ctx context.Context, tenantID, poID uuid.UUID) (*PurchaseOrder, error) {
	po, err := s.repo.GetPO(ctx, tenantID, poID)
	if err != nil {
		return nil, err
	}
	if po.Status != POStatusDraft {
		return nil, ErrPONotDraft
	}

	count, err := s.repo.CountPOLines(ctx, tenantID, poID)
	if err != nil {
		return nil, fmt.Errorf("count po lines: %w", err)
	}
	if count == 0 {
		return nil, ErrPONotSubmittable
	}

	if err := s.repo.UpdatePOStatus(ctx, tenantID, poID, POStatusSubmitted); err != nil {
		return nil, fmt.Errorf("submit po: %w", err)
	}

	po.Status = POStatusSubmitted
	slog.Info("einkauf po submitted", "po_id", poID, "tenant_id", tenantID)
	return po, nil
}

// ReceiveGoods marks all lines as fully received and transitions the PO to
// received status. If an InventarAdjuster is configured, it also increments
// the inventar stock level for each line's SKU by the ordered quantity.
// Missing SKUs in inventar are silently skipped (graceful degradation).
func (s *Service) ReceiveGoods(ctx context.Context, tenantID, poID uuid.UUID) (*PurchaseOrder, error) {
	po, err := s.repo.GetPO(ctx, tenantID, poID)
	if err != nil {
		return nil, err
	}
	if po.Status != POStatusSubmitted && po.Status != POStatusSent && po.Status != POStatusPartiallyReceived {
		return nil, ErrPONotReceivable
	}

	lines, err := s.repo.ListPOLines(ctx, tenantID, poID)
	if err != nil {
		return nil, fmt.Errorf("list po lines for receive: %w", err)
	}

	// Update received quantities for every line (full delivery).
	for _, line := range lines {
		if err := s.repo.UpdatePOLineReceivedQuantity(ctx, tenantID, line.ID, line.Quantity); err != nil {
			return nil, fmt.Errorf("update received quantity for line %s: %w", line.ID, err)
		}
	}

	// Transition PO to received.
	if err := s.repo.UpdatePOStatus(ctx, tenantID, poID, POStatusReceived); err != nil {
		return nil, fmt.Errorf("update po status received: %w", err)
	}
	po.Status = POStatusReceived

	// Record inventory movements for each line (best-effort: skip on error so
	// that a missing inventar item never blocks a goods receipt).
	if s.inventarAdj != nil {
		reason := fmt.Sprintf("goods receipt PO %s", po.PONumber)
		for _, line := range lines {
			qty, parseErr := strconv.ParseFloat(line.Quantity, 64)
			if parseErr != nil || qty <= 0 {
				continue
			}
			if adjErr := s.inventarAdj.AdjustStockBySKU(ctx, tenantID, line.SKU, int64(qty), reason); adjErr != nil {
				slog.Warn("inventar stock adjustment failed during ReceiveGoods",
					"po_id", poID, "line_id", line.ID, "sku", line.SKU, "error", adjErr)
			}
		}
	}

	slog.Info("einkauf goods received (full)", "po_id", poID, "tenant_id", tenantID, "lines_count", len(lines))
	return po, nil
}

// PartialReceive updates received quantities per line and transitions the PO
// to partially_received or received depending on whether all lines are complete.
// If an InventarAdjuster is configured, it also increments inventar stock for
// each supplied line's SKU by the newly received delta quantity.
// Missing SKUs in inventar are silently skipped (graceful degradation).
func (s *Service) PartialReceive(ctx context.Context, tenantID, poID uuid.UUID, items []PartialReceiveItem) (*PurchaseOrder, error) {
	po, err := s.repo.GetPO(ctx, tenantID, poID)
	if err != nil {
		return nil, err
	}
	if po.Status != POStatusSubmitted && po.Status != POStatusSent && po.Status != POStatusPartiallyReceived {
		return nil, ErrPONotReceivable
	}

	// -------------------------------------------------------------------------
	// Phase 1: Validate all items + compute incoming deltas (fail-fast).
	// lineDeltas maps lineID → (sku, incomingQty) for stock adjustment.
	// -------------------------------------------------------------------------
	type lineAdjust struct {
		sku      string
		incomingQty float64
	}
	lineAdjusts := make(map[uuid.UUID]lineAdjust, len(items))

	for _, item := range items {
		recv, err := strconv.ParseFloat(item.ReceivedQuantity, 64)
		if err != nil || recv < 0 {
			return nil, fmt.Errorf("partial receive line %s: %w", item.LineID, ErrInvalidQuantity)
		}
		line, err := s.repo.GetPOLine(ctx, tenantID, item.LineID)
		if err != nil {
			return nil, fmt.Errorf("partial receive line %s lookup: %w", item.LineID, err)
		}
		ordered, err := strconv.ParseFloat(line.Quantity, 64)
		if err != nil {
			return nil, fmt.Errorf("partial receive line %s ordered qty parse: %w", item.LineID, err)
		}
		if recv > ordered {
			return nil, fmt.Errorf("partial receive line %s: %w", item.LineID, ErrExceedsOrderedQty)
		}

		// Delta = newly arriving quantity (the full new received qty; callers
		// pass the cumulative received amount, so we track per-line delta by
		// comparing against the previous received_quantity).
		prevRecv, _ := strconv.ParseFloat(line.ReceivedQuantity, 64)
		delta := recv - prevRecv
		if delta > 0 && line.SKU != "" {
			lineAdjusts[item.LineID] = lineAdjust{sku: line.SKU, incomingQty: delta}
		}
	}

	// -------------------------------------------------------------------------
	// Phase 2: Persist received-quantity updates.
	// -------------------------------------------------------------------------
	for _, item := range items {
		if err := s.repo.UpdatePOLineReceivedQuantity(ctx, tenantID, item.LineID, item.ReceivedQuantity); err != nil {
			return nil, fmt.Errorf("partial receive line %s: %w", item.LineID, err)
		}
	}

	// -------------------------------------------------------------------------
	// Phase 3: Determine new PO status.
	// -------------------------------------------------------------------------
	lines, err := s.repo.ListPOLines(ctx, tenantID, poID)
	if err != nil {
		return nil, fmt.Errorf("reload lines after partial receive: %w", err)
	}

	newStatus := POStatusPartiallyReceived
	if allFullyReceived(lines) {
		newStatus = POStatusReceived
	}

	if err := s.repo.UpdatePOStatus(ctx, tenantID, poID, newStatus); err != nil {
		return nil, fmt.Errorf("update po status after partial receive: %w", err)
	}
	po.Status = newStatus

	// -------------------------------------------------------------------------
	// Phase 4: Record inventory movements (best-effort, non-blocking).
	// -------------------------------------------------------------------------
	if s.inventarAdj != nil {
		reason := fmt.Sprintf("partial goods receipt PO %s", po.PONumber)
		for lineID, adj := range lineAdjusts {
			if adjErr := s.inventarAdj.AdjustStockBySKU(ctx, tenantID, adj.sku, int64(adj.incomingQty), reason); adjErr != nil {
				slog.Warn("inventar stock adjustment failed during PartialReceive",
					"po_id", poID, "line_id", lineID, "sku", adj.sku, "error", adjErr)
			}
		}
	}

	slog.Info("einkauf partial receive applied", "po_id", poID, "tenant_id", tenantID, "new_status", newStatus)
	return po, nil
}

// ExportPO is a stub for exporting a PO as PDF/CSV.
// Full streaming implementation is a follow-up task in Sprint 3.
func (s *Service) ExportPO(ctx context.Context, tenantID, poID uuid.UUID, format string) (*PurchaseOrder, error) {
	return s.repo.GetPOWithLines(ctx, tenantID, poID)
}

// ============================================================================
// Helpers
// ============================================================================

// numericEqual compares two decimal strings numerically (float64 precision is
// sufficient for numeric(15,4) values which fit comfortably in 53 mantissa bits).
// Falls back to string equality when either value cannot be parsed.
func numericEqual(a, b string) bool {
	af, errA := strconv.ParseFloat(a, 64)
	bf, errB := strconv.ParseFloat(b, 64)
	if errA != nil || errB != nil {
		return a == b
	}
	return af == bf
}

// allFullyReceived returns true if every line's received_quantity equals its
// quantity. Uses numeric comparison so that "10.0000" == "10" evaluates as true.
func allFullyReceived(lines []*POLine) bool {
	if len(lines) == 0 {
		return false
	}
	for _, l := range lines {
		if !numericEqual(l.ReceivedQuantity, l.Quantity) {
			return false
		}
	}
	return true
}
