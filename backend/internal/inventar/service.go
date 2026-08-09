package inventar

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Service handles inventar business logic.
type Service struct {
	repo Repository
}

// NewService creates a new inventar service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ============================================================================
// Input types
// ============================================================================

// CreateItemInput contains data to create an inventory item.
type CreateItemInput struct {
	TenantID            uuid.UUID
	Name                string
	SKU                 string
	Barcode             *string
	Quantity            int64
	MinQuantity         int64
	Unit                string
	Location            *string
	LocationID          *uuid.UUID
	Category            *string
	Price               *float64
	Currency            *string
	BatchNumber         *string
	SerialNumbers       []string
	LinkedPurchaseOrder *string
}

// UpdateItemInput contains fields that can be updated on an item.
type UpdateItemInput struct {
	TenantID            uuid.UUID
	ItemID              uuid.UUID
	Name                *string
	SKU                 *string
	Barcode             *string
	MinQuantity         *int64
	Unit                *string
	Location            *string
	LocationID          *uuid.UUID
	Category            *string
	Price               *float64
	Currency            *string
	BatchNumber         *string
	LinkedPurchaseOrder *string
}

// ListItemsInput contains pagination and filtering for item listing.
type ListItemsInput struct {
	TenantID uuid.UUID
	Search   string
	Location *string
	LowStock bool
	Page     int
	PageSize int
}

// AdjustStockInput contains data for a stock adjustment.
type AdjustStockInput struct {
	TenantID    uuid.UUID
	ItemID      uuid.UUID
	Delta       int64 // positive = in, negative = out
	PerformedBy *uuid.UUID
	Reason      string
}

// TransferStockInput moves quantity from one item to another.
type TransferStockInput struct {
	TenantID    uuid.UUID
	FromItemID  uuid.UUID
	ToItemID    uuid.UUID
	Quantity    int64
	PerformedBy *uuid.UUID
	Reason      string
}

// RecordMovementInput is the low-level movement recorder (used for batch imports).
type RecordMovementInput struct {
	TenantID     uuid.UUID
	ItemID       uuid.UUID
	MovementType MovementType
	Quantity     int64
	PerformedBy  *uuid.UUID
	Reason       string
}

// ListMovementsInput contains pagination for movement listing.
type ListMovementsInput struct {
	TenantID uuid.UUID
	ItemID   uuid.UUID
	Page     int
	PageSize int
}

// CreateWarningInput contains data for a manual stock warning.
type CreateWarningInput struct {
	TenantID        uuid.UUID
	ItemID          uuid.UUID
	Threshold       int64
	CurrentQuantity int64
}

// UpdateWarningInput contains fields to update on a warning.
type UpdateWarningInput struct {
	TenantID  uuid.UUID
	WarningID uuid.UUID
	Status    WarningStatus
}

// AcknowledgeWarningInput acknowledges a warning.
type AcknowledgeWarningInput struct {
	TenantID       uuid.UUID
	WarningID      uuid.UUID
	AcknowledgedBy uuid.UUID
}

// ListWarningsInput contains filtering and pagination for warnings.
type ListWarningsInput struct {
	TenantID uuid.UUID
	Status   *WarningStatus
	Page     int
	PageSize int
}

// StockReport is an aggregated view of inventory health.
type StockReport struct {
	TotalItems     int
	TotalQuantity  int64
	LowStockCount  int
	ActiveWarnings int
}

// ============================================================================
// Item methods
// ============================================================================

// CreateItem creates a new inventory item.
func (s *Service) CreateItem(ctx context.Context, input CreateItemInput) (*Item, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrInvalidInput
	}
	sku := strings.TrimSpace(input.SKU)
	if sku == "" {
		return nil, ErrInvalidInput
	}
	if input.MinQuantity < 0 {
		return nil, ErrInvalidInput
	}

	exists, err := s.repo.SKUExists(ctx, input.TenantID, sku, nil)
	if err != nil {
		return nil, fmt.Errorf("check sku: %w", err)
	}
	if exists {
		return nil, ErrSKUTaken
	}

	now := time.Now()
	item := &Item{
		ID:                  uuid.New(),
		TenantID:            input.TenantID,
		Name:                name,
		SKU:                 sku,
		Barcode:             input.Barcode,
		Quantity:            input.Quantity,
		MinQuantity:         input.MinQuantity,
		Unit:                input.Unit,
		Location:            input.Location,
		LocationID:          input.LocationID,
		Category:            input.Category,
		Price:               input.Price,
		Currency:            input.Currency,
		BatchNumber:         input.BatchNumber,
		SerialNumbers:       input.SerialNumbers,
		LinkedPurchaseOrder: input.LinkedPurchaseOrder,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if createErr := s.repo.CreateItem(ctx, item); createErr != nil {
		return nil, fmt.Errorf("create inventory item: %w", createErr)
	}

	slog.Info("inventory item created",
		"item_id", item.ID,
		"tenant_id", item.TenantID,
		"sku", item.SKU,
	)

	return item, nil
}

// UpdateItem updates mutable fields on an item. Quantity is not updated here;
// use AdjustStock or RecordMovement for quantity changes.
func (s *Service) UpdateItem(ctx context.Context, input UpdateItemInput) (*Item, error) {
	item, err := s.repo.GetItem(ctx, input.TenantID, input.ItemID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		n := strings.TrimSpace(*input.Name)
		if n == "" {
			return nil, ErrInvalidInput
		}
		item.Name = n
	}

	if input.SKU != nil {
		sku := strings.TrimSpace(*input.SKU)
		if sku != "" && sku != item.SKU {
			exists, skuErr := s.repo.SKUExists(ctx, input.TenantID, sku, &item.ID)
			if skuErr != nil {
				return nil, fmt.Errorf("check sku: %w", skuErr)
			}
			if exists {
				return nil, ErrSKUTaken
			}
			item.SKU = sku
		}
	}

	if input.Barcode != nil {
		item.Barcode = input.Barcode
	}
	if input.MinQuantity != nil {
		if *input.MinQuantity < 0 {
			return nil, ErrInvalidInput
		}
		item.MinQuantity = *input.MinQuantity
	}
	if input.Unit != nil {
		item.Unit = *input.Unit
	}
	if input.Location != nil {
		item.Location = input.Location
	}
	if input.LocationID != nil {
		item.LocationID = input.LocationID
	}
	if input.Category != nil {
		item.Category = input.Category
	}
	if input.Price != nil {
		item.Price = input.Price
	}
	if input.Currency != nil {
		item.Currency = input.Currency
	}
	if input.BatchNumber != nil {
		item.BatchNumber = input.BatchNumber
	}
	if input.LinkedPurchaseOrder != nil {
		item.LinkedPurchaseOrder = input.LinkedPurchaseOrder
	}

	item.UpdatedAt = time.Now()

	if updateErr := s.repo.UpdateItem(ctx, item); updateErr != nil {
		return nil, fmt.Errorf("update inventory item: %w", updateErr)
	}

	slog.Info("inventory item updated", "item_id", item.ID, "tenant_id", item.TenantID)
	return item, nil
}

// DeleteItem soft-deletes an item by setting deleted_at.
func (s *Service) DeleteItem(ctx context.Context, tenantID, itemID uuid.UUID) error {
	if _, err := s.repo.GetItem(ctx, tenantID, itemID); err != nil {
		return err
	}

	if delErr := s.repo.SoftDeleteItem(ctx, tenantID, itemID); delErr != nil {
		return fmt.Errorf("soft delete inventory item: %w", delErr)
	}

	slog.Info("inventory item deleted", "item_id", itemID, "tenant_id", tenantID)
	return nil
}

// GetItem retrieves an item by ID.
func (s *Service) GetItem(ctx context.Context, tenantID, itemID uuid.UUID) (*Item, error) {
	return s.repo.GetItem(ctx, tenantID, itemID)
}

// ListItems retrieves items with optional filtering and pagination.
func (s *Service) ListItems(ctx context.Context, input ListItemsInput) ([]*Item, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 200 {
		input.PageSize = 50
	}

	offset := (input.Page - 1) * input.PageSize
	filter := ListItemsFilter{
		Search:   input.Search,
		Location: input.Location,
		LowStock: input.LowStock,
	}

	return s.repo.ListItems(ctx, input.TenantID, filter, offset, input.PageSize)
}

// ============================================================================
// Stock adjustment methods
// ============================================================================

// AdjustStock applies a delta to an item's quantity and records a movement.
// Automatically creates a stock_warning (idempotent) if quantity <= min_quantity.
func (s *Service) AdjustStock(ctx context.Context, input AdjustStockInput) (*Item, error) {
	if input.Delta == 0 {
		return nil, ErrInvalidInput
	}

	item, err := s.repo.GetItem(ctx, input.TenantID, input.ItemID)
	if err != nil {
		return nil, err
	}

	if item.Quantity+input.Delta < 0 {
		return nil, ErrInsufficientStock
	}

	item.Quantity = item.Quantity + input.Delta
	item.UpdatedAt = time.Now()

	if updateErr := s.repo.UpdateItem(ctx, item); updateErr != nil {
		return nil, fmt.Errorf("update item quantity: %w", updateErr)
	}

	movement := &Movement{
		ID:           uuid.New(),
		TenantID:     input.TenantID,
		ItemID:       item.ID,
		MovementType: movementTypeForDelta(input.Delta),
		Quantity:     input.Delta,
		PerformedBy:  input.PerformedBy,
		Reason:       input.Reason,
		CreatedAt:    time.Now(),
	}
	if createErr := s.repo.CreateMovement(ctx, movement); createErr != nil {
		return nil, fmt.Errorf("record adjustment movement: %w", createErr)
	}

	s.maybeCreateWarning(ctx, item)

	slog.Info("inventory stock adjusted",
		"item_id", item.ID,
		"delta", input.Delta,
		"new_quantity", item.Quantity,
	)

	return item, nil
}

// TransferStock moves quantity from one item to another.
func (s *Service) TransferStock(ctx context.Context, input TransferStockInput) error {
	if input.Quantity <= 0 {
		return ErrInvalidInput
	}

	fromItem, err := s.repo.GetItem(ctx, input.TenantID, input.FromItemID)
	if err != nil {
		return err
	}
	toItem, err := s.repo.GetItem(ctx, input.TenantID, input.ToItemID)
	if err != nil {
		return err
	}

	if fromItem.Quantity < input.Quantity {
		return ErrInsufficientStock
	}

	now := time.Now()

	// Deduct from source
	fromItem.Quantity -= input.Quantity
	fromItem.UpdatedAt = now
	if updateErr := s.repo.UpdateItem(ctx, fromItem); updateErr != nil {
		return fmt.Errorf("update from-item quantity: %w", updateErr)
	}

	// Add to destination
	toItem.Quantity += input.Quantity
	toItem.UpdatedAt = now
	if updateErr := s.repo.UpdateItem(ctx, toItem); updateErr != nil {
		return fmt.Errorf("update to-item quantity: %w", updateErr)
	}

	// Record outbound movement
	outMovement := &Movement{
		ID:           uuid.New(),
		TenantID:     input.TenantID,
		ItemID:       fromItem.ID,
		MovementType: MovementTypeTransfer,
		Quantity:     -input.Quantity,
		PerformedBy:  input.PerformedBy,
		Reason:       input.Reason,
		CreatedAt:    now,
	}
	if createErr := s.repo.CreateMovement(ctx, outMovement); createErr != nil {
		return fmt.Errorf("record transfer-out movement: %w", createErr)
	}

	// Record inbound movement
	inMovement := &Movement{
		ID:           uuid.New(),
		TenantID:     input.TenantID,
		ItemID:       toItem.ID,
		MovementType: MovementTypeTransfer,
		Quantity:     input.Quantity,
		PerformedBy:  input.PerformedBy,
		Reason:       input.Reason,
		CreatedAt:    now,
	}
	if createErr := s.repo.CreateMovement(ctx, inMovement); createErr != nil {
		return fmt.Errorf("record transfer-in movement: %w", createErr)
	}

	s.maybeCreateWarning(ctx, fromItem)

	slog.Info("inventory stock transferred",
		"from_item_id", fromItem.ID,
		"to_item_id", toItem.ID,
		"quantity", input.Quantity,
	)

	return nil
}

// RecordMovement records a movement without changing item quantity directly.
// Intended for batch imports where the caller manages quantity changes.
func (s *Service) RecordMovement(ctx context.Context, input RecordMovementInput) (*Movement, error) {
	if input.Quantity == 0 {
		return nil, ErrInvalidInput
	}

	if _, err := s.repo.GetItem(ctx, input.TenantID, input.ItemID); err != nil {
		return nil, err
	}

	movement := &Movement{
		ID:           uuid.New(),
		TenantID:     input.TenantID,
		ItemID:       input.ItemID,
		MovementType: input.MovementType,
		Quantity:     input.Quantity,
		PerformedBy:  input.PerformedBy,
		Reason:       input.Reason,
		CreatedAt:    time.Now(),
	}

	if createErr := s.repo.CreateMovement(ctx, movement); createErr != nil {
		return nil, fmt.Errorf("record movement: %w", createErr)
	}

	slog.Info("inventory movement recorded",
		"movement_id", movement.ID,
		"item_id", movement.ItemID,
		"type", movement.MovementType,
	)

	return movement, nil
}

// ListMovements returns paginated movements for an item.
func (s *Service) ListMovements(ctx context.Context, input ListMovementsInput) ([]*Movement, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 200 {
		input.PageSize = 50
	}
	offset := (input.Page - 1) * input.PageSize
	return s.repo.ListMovements(ctx, input.TenantID, input.ItemID, offset, input.PageSize)
}

// ============================================================================
// Warning methods
// ============================================================================

// CreateWarning creates a manual stock warning.
func (s *Service) CreateWarning(ctx context.Context, input CreateWarningInput) (*Warning, error) {
	if _, err := s.repo.GetItem(ctx, input.TenantID, input.ItemID); err != nil {
		return nil, err
	}

	warning := &Warning{
		ID:              uuid.New(),
		TenantID:        input.TenantID,
		ItemID:          input.ItemID,
		Threshold:       input.Threshold,
		CurrentQuantity: input.CurrentQuantity,
		Status:          WarningStatusActive,
		CreatedAt:       time.Now(),
	}

	if createErr := s.repo.CreateWarning(ctx, warning); createErr != nil {
		return nil, fmt.Errorf("create stock warning: %w", createErr)
	}

	slog.Info("stock warning created", "warning_id", warning.ID, "item_id", warning.ItemID)
	return warning, nil
}

// UpdateWarning updates the status of a warning.
func (s *Service) UpdateWarning(ctx context.Context, input UpdateWarningInput) (*Warning, error) {
	warning, err := s.repo.GetWarning(ctx, input.TenantID, input.WarningID)
	if err != nil {
		return nil, err
	}

	warning.Status = input.Status

	if updateErr := s.repo.UpdateWarning(ctx, warning); updateErr != nil {
		return nil, fmt.Errorf("update stock warning: %w", updateErr)
	}

	return warning, nil
}

// AcknowledgeWarning marks a warning as acknowledged.
func (s *Service) AcknowledgeWarning(ctx context.Context, input AcknowledgeWarningInput) (*Warning, error) {
	warning, err := s.repo.GetWarning(ctx, input.TenantID, input.WarningID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	warning.Status = WarningStatusAcknowledged
	warning.AcknowledgedAt = &now
	warning.AcknowledgedBy = &input.AcknowledgedBy

	if updateErr := s.repo.UpdateWarning(ctx, warning); updateErr != nil {
		return nil, fmt.Errorf("acknowledge stock warning: %w", updateErr)
	}

	slog.Info("stock warning acknowledged",
		"warning_id", warning.ID,
		"acknowledged_by", input.AcknowledgedBy,
	)

	return warning, nil
}

// ListWarnings returns paginated warnings filtered by optional status.
func (s *Service) ListWarnings(ctx context.Context, input ListWarningsInput) ([]*Warning, int, error) {
	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 || input.PageSize > 200 {
		input.PageSize = 50
	}
	offset := (input.Page - 1) * input.PageSize
	return s.repo.ListWarnings(ctx, input.TenantID, input.Status, offset, input.PageSize)
}

// GetStockReport returns aggregated inventory statistics for a tenant.
func (s *Service) GetStockReport(ctx context.Context, tenantID uuid.UUID) (*StockReport, error) {
	allItems, totalItems, err := s.repo.ListItems(ctx, tenantID, ListItemsFilter{}, 0, 10000)
	if err != nil {
		return nil, fmt.Errorf("get stock report items: %w", err)
	}

	var totalQty int64
	var lowStockCount int
	for _, item := range allItems {
		totalQty += item.Quantity
		if item.Quantity <= item.MinQuantity {
			lowStockCount++
		}
	}

	activeStatus := WarningStatusActive
	_, activeWarnings, err := s.repo.ListWarnings(ctx, tenantID, &activeStatus, 0, 1)
	if err != nil {
		return nil, fmt.Errorf("get stock report warnings: %w", err)
	}

	return &StockReport{
		TotalItems:     totalItems,
		TotalQuantity:  totalQty,
		LowStockCount:  lowStockCount,
		ActiveWarnings: activeWarnings,
	}, nil
}

// ============================================================================
// Location inputs
// ============================================================================

// CreateLocationInput holds fields required to create an inventory location.
type CreateLocationInput struct {
	TenantID uuid.UUID
	Name     string
	Address  string
	Type     LocationType
}

// UpdateLocationInput holds fields for updating an inventory location.
type UpdateLocationInput struct {
	TenantID   uuid.UUID
	LocationID uuid.UUID
	Name       *string
	Address    *string
	Type       *LocationType
}

// ListLocationsInput holds pagination for listing locations.
type ListLocationsInput struct {
	TenantID uuid.UUID
	Page     int
	PageSize int
}

// ============================================================================
// Location service methods
// ============================================================================

// CreateLocation creates a new inventory location.
func (s *Service) CreateLocation(ctx context.Context, input CreateLocationInput) (*InventoryLocation, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if input.Type != LocationTypeWarehouse && input.Type != LocationTypeStore && input.Type != LocationTypeVehicle {
		input.Type = LocationTypeWarehouse
	}
	now := time.Now()
	loc := &InventoryLocation{
		ID:        uuid.New(),
		TenantID:  input.TenantID,
		Name:      input.Name,
		Address:   input.Address,
		Type:      input.Type,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.CreateLocation(ctx, loc); err != nil {
		return nil, fmt.Errorf("create location: %w", err)
	}
	slog.InfoContext(ctx, "inventory location created", "location_id", loc.ID, "tenant_id", loc.TenantID)
	return loc, nil
}

// GetLocation retrieves a single inventory location.
func (s *Service) GetLocation(ctx context.Context, tenantID, locID uuid.UUID) (*InventoryLocation, error) {
	return s.repo.GetLocation(ctx, tenantID, locID)
}

// UpdateLocation updates mutable fields of an inventory location.
func (s *Service) UpdateLocation(ctx context.Context, input UpdateLocationInput) (*InventoryLocation, error) {
	loc, err := s.repo.GetLocation(ctx, input.TenantID, input.LocationID)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		loc.Name = *input.Name
	}
	if input.Address != nil {
		loc.Address = *input.Address
	}
	if input.Type != nil {
		loc.Type = *input.Type
	}
	loc.UpdatedAt = time.Now()
	if err := s.repo.UpdateLocation(ctx, loc); err != nil {
		return nil, fmt.Errorf("update location: %w", err)
	}
	return loc, nil
}

// DeleteLocation soft-deletes an inventory location.
func (s *Service) DeleteLocation(ctx context.Context, tenantID, locID uuid.UUID) error {
	return s.repo.SoftDeleteLocation(ctx, tenantID, locID)
}

// ListLocations returns a paginated list of inventory locations.
func (s *Service) ListLocations(ctx context.Context, input ListLocationsInput) ([]*InventoryLocation, int, error) {
	page := input.Page
	if page < 1 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize
	return s.repo.ListLocations(ctx, input.TenantID, offset, pageSize)
}

// ============================================================================
// Inventur Session inputs
// ============================================================================

// CreateInventurSessionInput holds fields for creating an inventur session.
type CreateInventurSessionInput struct {
	TenantID   uuid.UUID
	Name       string
	Date       time.Time
	LocationID *uuid.UUID
	CreatedBy  *uuid.UUID
	ItemIDs    []uuid.UUID // items to include in counting — pre-populated with current quantities
}

// UpdateInventurSessionStatusInput holds fields for updating session status.
type UpdateInventurSessionStatusInput struct {
	TenantID  uuid.UUID
	SessionID uuid.UUID
	Status    InventurStatus
}

// UpsertInventurCountInput holds fields for recording a count for one item.
type UpsertInventurCountInput struct {
	TenantID  uuid.UUID
	SessionID uuid.UUID
	ItemID    uuid.UUID
	Counted   int64
}

// ListInventurSessionsInput holds pagination for listing sessions.
type ListInventurSessionsInput struct {
	TenantID uuid.UUID
	Page     int
	PageSize int
}

// BookInventurDifferencesInput triggers stock adjustments for all counted differences.
type BookInventurDifferencesInput struct {
	TenantID  uuid.UUID
	SessionID uuid.UUID
	BookedBy  *uuid.UUID
}

// PickingListItemInput is one requested position when creating or updating a
// picking list.
type PickingListItemInput struct {
	ItemID            uuid.UUID
	QuantityRequested int64
	QuantityPicked    int64
	Location          *string
}

// CreatePickingListInput creates a picking list with its positions.
type CreatePickingListInput struct {
	TenantID   uuid.UUID
	Reference  string
	AssignedTo *uuid.UUID
	CreatedBy  *uuid.UUID
	Items      []PickingListItemInput
}

// UpdatePickingListInput changes head data of a picking list.
type UpdatePickingListInput struct {
	TenantID   uuid.UUID
	ListID     uuid.UUID
	Reference  *string
	Status     *PickingStatus
	AssignedTo *uuid.UUID
}

// ListPickingListsInput holds pagination and the optional status filter.
type ListPickingListsInput struct {
	TenantID uuid.UUID
	Status   *PickingStatus
	Page     int
	PageSize int
}

// UpsertPickingListItemInput records the picked quantity for one position.
type UpsertPickingListItemInput struct {
	TenantID          uuid.UUID
	ListID            uuid.UUID
	ItemID            uuid.UUID
	QuantityRequested int64
	QuantityPicked    int64
	Location          *string
}

// BookPickingListInput books the picked quantities out of stock.
type BookPickingListInput struct {
	TenantID uuid.UUID
	ListID   uuid.UUID
	BookedBy *uuid.UUID
}

// ============================================================================
// Inventur Session service methods
// ============================================================================

// CreateInventurSession creates a new inventur session with optional pre-populated counts.
func (s *Service) CreateInventurSession(ctx context.Context, input CreateInventurSessionInput) (*InventurSession, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("%w: session name is required", ErrInvalidInput)
	}
	date := input.Date
	if date.IsZero() {
		date = time.Now()
	}
	now := time.Now()
	session := &InventurSession{
		ID:         uuid.New(),
		TenantID:   input.TenantID,
		Name:       input.Name,
		Date:       date,
		Status:     InventurStatusOpen,
		LocationID: input.LocationID,
		CreatedBy:  input.CreatedBy,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.repo.CreateInventurSession(ctx, session); err != nil {
		return nil, fmt.Errorf("create inventur session: %w", err)
	}

	// Pre-populate counts for given items
	for _, itemID := range input.ItemIDs {
		item, err := s.repo.GetItem(ctx, input.TenantID, itemID)
		if err != nil {
			continue // skip items not found
		}
		count := &InventurCount{
			ID:        uuid.New(),
			TenantID:  input.TenantID,
			SessionID: session.ID,
			ItemID:    itemID,
			Expected:  item.Quantity,
		}
		_ = s.repo.UpsertInventurCount(ctx, count)
	}

	slog.InfoContext(ctx, "inventur session created", "session_id", session.ID, "tenant_id", session.TenantID)
	return session, nil
}

// GetInventurSession retrieves a single inventur session including counts.
func (s *Service) GetInventurSession(ctx context.Context, tenantID, sessionID uuid.UUID) (*InventurSession, error) {
	return s.repo.GetInventurSession(ctx, tenantID, sessionID)
}

// UpdateInventurSessionStatus updates the status of an inventur session.
func (s *Service) UpdateInventurSessionStatus(ctx context.Context, input UpdateInventurSessionStatusInput) (*InventurSession, error) {
	session, err := s.repo.GetInventurSession(ctx, input.TenantID, input.SessionID)
	if err != nil {
		return nil, err
	}
	if session.Status == InventurStatusCompleted {
		return nil, ErrInventurAlreadyCompleted
	}
	session.Status = input.Status
	session.UpdatedAt = time.Now()
	if err := s.repo.UpdateInventurSession(ctx, session); err != nil {
		return nil, fmt.Errorf("update inventur session status: %w", err)
	}
	return session, nil
}

// DeleteInventurSession deletes an inventur session (cascades to counts).
func (s *Service) DeleteInventurSession(ctx context.Context, tenantID, sessionID uuid.UUID) error {
	return s.repo.DeleteInventurSession(ctx, tenantID, sessionID)
}

// ListInventurSessions returns a paginated list of inventur sessions.
func (s *Service) ListInventurSessions(ctx context.Context, input ListInventurSessionsInput) ([]*InventurSession, int, error) {
	page := input.Page
	if page < 1 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize
	return s.repo.ListInventurSessions(ctx, input.TenantID, offset, pageSize)
}

// UpsertInventurCount records or updates the counted quantity for one item in a session.
func (s *Service) UpsertInventurCount(ctx context.Context, input UpsertInventurCountInput) (*InventurCount, error) {
	session, err := s.repo.GetInventurSession(ctx, input.TenantID, input.SessionID)
	if err != nil {
		return nil, err
	}
	if session.Status == InventurStatusCompleted {
		return nil, ErrInventurAlreadyCompleted
	}
	now := time.Now()
	count := &InventurCount{
		ID:        uuid.New(),
		TenantID:  input.TenantID,
		SessionID: input.SessionID,
		ItemID:    input.ItemID,
		Counted:   &input.Counted,
		CountedAt: &now,
	}
	// Fetch expected from item
	item, err := s.repo.GetItem(ctx, input.TenantID, input.ItemID)
	if err == nil {
		count.Expected = item.Quantity
	}
	if err := s.repo.UpsertInventurCount(ctx, count); err != nil {
		return nil, fmt.Errorf("upsert inventur count: %w", err)
	}
	return count, nil
}

// BookInventurDifferences applies stock adjustments for all counted
// differences and marks the session completed, both in one transaction: a
// difference that cannot be booked (unknown item, insufficient stock) must
// leave the session exactly as it was, not completed with some deltas
// silently dropped.
func (s *Service) BookInventurDifferences(ctx context.Context, input BookInventurDifferencesInput) (*InventurSession, error) {
	session, err := s.repo.GetInventurSession(ctx, input.TenantID, input.SessionID)
	if err != nil {
		return nil, err
	}
	if session.Status == InventurStatusCompleted {
		return nil, ErrInventurAlreadyCompleted
	}

	counts, err := s.repo.ListInventurCounts(ctx, input.TenantID, input.SessionID)
	if err != nil {
		return nil, fmt.Errorf("list counts for booking: %w", err)
	}

	var movements []StockMovementInput
	for _, count := range counts {
		if count.Counted == nil {
			continue // uncounted items are skipped
		}
		diff := *count.Counted - count.Expected
		if diff == 0 {
			continue
		}
		movements = append(movements, StockMovementInput{
			ItemID:       count.ItemID,
			Delta:        diff,
			MovementType: movementTypeForDelta(diff),
			PerformedBy:  input.BookedBy,
			Reason:       "Inventur-Buchung: " + session.Name,
		})
	}

	items, err := s.repo.CompleteInventurSessionTx(ctx, input.TenantID, input.SessionID, movements)
	if err != nil {
		return nil, fmt.Errorf("book inventur differences: %w", err)
	}
	for _, item := range items {
		s.maybeCreateWarning(ctx, item)
	}

	slog.InfoContext(ctx, "inventur session booked", "session_id", session.ID, "tenant_id", session.TenantID)
	return s.repo.GetInventurSession(ctx, input.TenantID, input.SessionID)
}

// ============================================================================
// Picking list service methods
// ============================================================================

// CreatePickingList creates a picking list head plus its positions.
func (s *Service) CreatePickingList(ctx context.Context, input CreatePickingListInput) (*PickingList, error) {
	if input.Reference == "" {
		return nil, fmt.Errorf("%w: picking list reference is required", ErrInvalidInput)
	}
	for _, it := range input.Items {
		if err := validatePickQuantities(it.QuantityRequested, it.QuantityPicked); err != nil {
			return nil, err
		}
	}

	now := time.Now()
	list := &PickingList{
		ID:         uuid.New(),
		TenantID:   input.TenantID,
		Reference:  input.Reference,
		Status:     PickingStatusOpen,
		AssignedTo: input.AssignedTo,
		CreatedBy:  input.CreatedBy,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.repo.CreatePickingList(ctx, list); err != nil {
		return nil, fmt.Errorf("create picking list: %w", err)
	}

	for _, in := range input.Items {
		if _, err := s.repo.GetItem(ctx, input.TenantID, in.ItemID); err != nil {
			return nil, err // unknown item is a caller error, not something to swallow
		}
		row := &PickingListItem{
			ID:                uuid.New(),
			TenantID:          input.TenantID,
			PickingListID:     list.ID,
			ItemID:            in.ItemID,
			QuantityRequested: in.QuantityRequested,
			QuantityPicked:    in.QuantityPicked,
			Location:          in.Location,
		}
		if err := s.repo.UpsertPickingListItem(ctx, row); err != nil {
			return nil, fmt.Errorf("create picking list item: %w", err)
		}
		list.Items = append(list.Items, *row)
	}

	slog.InfoContext(ctx, "picking list created", "list_id", list.ID, "tenant_id", list.TenantID, "positions", len(list.Items))
	return list, nil
}

// GetPickingList returns a single picking list including its positions.
func (s *Service) GetPickingList(ctx context.Context, tenantID, listID uuid.UUID) (*PickingList, error) {
	return s.repo.GetPickingList(ctx, tenantID, listID)
}

// ListPickingLists returns a paginated list of picking lists.
func (s *Service) ListPickingLists(ctx context.Context, input ListPickingListsInput) ([]*PickingList, int, error) {
	page := input.Page
	if page < 1 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	return s.repo.ListPickingLists(ctx, input.TenantID, input.Status, (page-1)*pageSize, pageSize)
}

// UpdatePickingList changes reference, status or assignee of a picking list.
func (s *Service) UpdatePickingList(ctx context.Context, input UpdatePickingListInput) (*PickingList, error) {
	list, err := s.repo.GetPickingList(ctx, input.TenantID, input.ListID)
	if err != nil {
		return nil, err
	}
	if list.Status == PickingStatusCompleted {
		return nil, ErrPickingListAlreadyBooked
	}
	if input.Reference != nil {
		if *input.Reference == "" {
			return nil, fmt.Errorf("%w: picking list reference is required", ErrInvalidInput)
		}
		list.Reference = *input.Reference
	}
	if input.Status != nil {
		switch *input.Status {
		case PickingStatusOpen, PickingStatusPicking:
			list.Status = *input.Status
		case PickingStatusCompleted:
			// Completion is what booking does; letting it be set by hand would
			// mark the stock as moved without moving it.
			return nil, fmt.Errorf("%w: completed is reached by booking the list", ErrInvalidInput)
		default:
			return nil, fmt.Errorf("%w: unknown picking status %q", ErrInvalidInput, *input.Status)
		}
	}
	if input.AssignedTo != nil {
		list.AssignedTo = input.AssignedTo
	}
	list.UpdatedAt = time.Now()
	if err := s.repo.UpdatePickingList(ctx, list); err != nil {
		return nil, fmt.Errorf("update picking list: %w", err)
	}
	return list, nil
}

// DeletePickingList removes a picking list (cascades to its positions).
func (s *Service) DeletePickingList(ctx context.Context, tenantID, listID uuid.UUID) error {
	return s.repo.DeletePickingList(ctx, tenantID, listID)
}

// UpsertPickingListItem records or updates one position of a picking list.
func (s *Service) UpsertPickingListItem(ctx context.Context, input UpsertPickingListItemInput) (*PickingListItem, error) {
	list, err := s.repo.GetPickingList(ctx, input.TenantID, input.ListID)
	if err != nil {
		return nil, err
	}
	if list.Status == PickingStatusCompleted {
		return nil, ErrPickingListAlreadyBooked
	}
	if err := validatePickQuantities(input.QuantityRequested, input.QuantityPicked); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetItem(ctx, input.TenantID, input.ItemID); err != nil {
		return nil, err
	}

	row := &PickingListItem{
		ID:                uuid.New(),
		TenantID:          input.TenantID,
		PickingListID:     input.ListID,
		ItemID:            input.ItemID,
		QuantityRequested: input.QuantityRequested,
		QuantityPicked:    input.QuantityPicked,
		Location:          input.Location,
	}
	if err := s.repo.UpsertPickingListItem(ctx, row); err != nil {
		return nil, fmt.Errorf("upsert picking list item: %w", err)
	}

	// First recorded pick moves the list out of "open", mirroring the
	// open->counting transition of an Inventur session.
	if list.Status == PickingStatusOpen && input.QuantityPicked > 0 {
		list.Status = PickingStatusPicking
		list.UpdatedAt = time.Now()
		if err := s.repo.UpdatePickingList(ctx, list); err != nil {
			return nil, fmt.Errorf("advance picking list status: %w", err)
		}
	}
	return row, nil
}

// DeletePickingListItem removes one position from a picking list.
func (s *Service) DeletePickingListItem(ctx context.Context, tenantID, itemRowID uuid.UUID) error {
	return s.repo.DeletePickingListItem(ctx, tenantID, itemRowID)
}

// BookPickingList books the picked quantities out of stock and completes the
// list. The claim and every stock movement run in one transaction
// (Repository.BookPickingListTx): a position that cannot be booked (unknown
// item, insufficient stock) rolls back the claim too, so a list that cannot
// be booked in full is never left half-booked. A second booking finds the
// list already completed, gets ErrPickingListAlreadyBooked, and moves nothing.
func (s *Service) BookPickingList(ctx context.Context, input BookPickingListInput) (*PickingList, error) {
	list, err := s.repo.GetPickingList(ctx, input.TenantID, input.ListID)
	if err != nil {
		return nil, err
	}
	if list.Status == PickingStatusCompleted {
		return nil, ErrPickingListAlreadyBooked
	}

	positions, err := s.repo.ListPickingListItems(ctx, input.TenantID, input.ListID)
	if err != nil {
		return nil, fmt.Errorf("list picking positions for booking: %w", err)
	}

	movements := make([]StockMovementInput, 0, len(positions))
	for _, pos := range positions {
		if pos.QuantityPicked <= 0 {
			continue
		}
		movements = append(movements, StockMovementInput{
			ItemID:       pos.ItemID,
			Delta:        -pos.QuantityPicked,
			MovementType: MovementTypeOut,
			PerformedBy:  input.BookedBy,
			Reason:       "Kommissionierung: " + list.Reference,
		})
	}

	claimed, items, err := s.repo.BookPickingListTx(ctx, input.TenantID, input.ListID, movements)
	if err != nil {
		return nil, fmt.Errorf("book picking list: %w", err)
	}
	if !claimed {
		return nil, ErrPickingListAlreadyBooked
	}
	for _, item := range items {
		s.maybeCreateWarning(ctx, item)
	}

	slog.InfoContext(ctx, "picking list booked",
		"list_id", input.ListID, "tenant_id", input.TenantID, "positions", len(movements))
	return s.repo.GetPickingList(ctx, input.TenantID, input.ListID)
}

// movementTypeForDelta classifies a signed quantity delta into the movement
// type it represents, shared by every stock-changing path (manual
// adjustment, picking, Inventur booking) so the classification cannot drift.
func movementTypeForDelta(delta int64) MovementType {
	switch {
	case delta < 0:
		return MovementTypeOut
	case delta > 0:
		return MovementTypeIn
	default:
		return MovementTypeAdjustment
	}
}

// validatePickQuantities rejects positions that cannot describe a real pick.
func validatePickQuantities(requested, picked int64) error {
	if requested <= 0 {
		return fmt.Errorf("%w: quantity_requested must be greater than zero", ErrInvalidInput)
	}
	if picked < 0 {
		return fmt.Errorf("%w: quantity_picked must not be negative", ErrInvalidInput)
	}
	if picked > requested {
		return fmt.Errorf("%w: quantity_picked (%d) exceeds quantity_requested (%d)", ErrInvalidInput, picked, requested)
	}
	return nil
}

// ============================================================================
// Internal helpers
// ============================================================================

// maybeCreateWarning creates an active stock_warning when quantity <= min_quantity.
// Idempotent: skips creation if an active warning already exists.
func (s *Service) maybeCreateWarning(ctx context.Context, item *Item) {
	if item.Quantity > item.MinQuantity {
		return
	}

	_, err := s.repo.GetActiveWarningForItem(ctx, item.TenantID, item.ID)
	if err == nil {
		// Active warning already exists — nothing to do.
		return
	}
	if !errors.Is(err, ErrWarningNotFound) {
		slog.Warn("failed to check active warning", "item_id", item.ID, "error", err)
		return
	}

	warning := &Warning{
		ID:              uuid.New(),
		TenantID:        item.TenantID,
		ItemID:          item.ID,
		Threshold:       item.MinQuantity,
		CurrentQuantity: item.Quantity,
		Status:          WarningStatusActive,
		CreatedAt:       time.Now(),
	}

	if createErr := s.repo.CreateWarning(ctx, warning); createErr != nil {
		slog.Warn("failed to create automatic stock warning",
			"item_id", item.ID,
			"error", createErr,
		)
	} else {
		slog.Info("automatic stock warning created",
			"warning_id", warning.ID,
			"item_id", item.ID,
			"quantity", item.Quantity,
			"threshold", item.MinQuantity,
		)
	}
}

// ============================================================================
// Item Attachment methods
// ============================================================================

// CreateItemAttachmentInput holds the data to attach an uploaded file to an item.
type CreateItemAttachmentInput struct {
	TenantID  uuid.UUID
	ItemID    uuid.UUID
	Name      string
	ObjectKey string
	FileType  string
}

// CreateItemAttachment registers a browser-direct uploaded file against an item.
func (s *Service) CreateItemAttachment(ctx context.Context, input CreateItemAttachmentInput) (*ItemAttachment, error) {
	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.ObjectKey) == "" {
		return nil, ErrInvalidInput
	}
	// Ensures the item exists in this tenant before attaching (also surfaces a
	// clean NotFound instead of an FK violation).
	if _, err := s.repo.GetItem(ctx, input.TenantID, input.ItemID); err != nil {
		return nil, err
	}

	att := &ItemAttachment{
		ID:        uuid.New(),
		TenantID:  input.TenantID,
		ItemID:    input.ItemID,
		Name:      input.Name,
		ObjectKey: input.ObjectKey,
		FileType:  input.FileType,
	}
	if err := s.repo.CreateItemAttachment(ctx, att); err != nil {
		return nil, fmt.Errorf("create item attachment: %w", err)
	}

	slog.Info("item attachment created", "attachment_id", att.ID, "item_id", att.ItemID)
	return att, nil
}

// ListItemAttachments returns all attachments for an item in the tenant.
func (s *Service) ListItemAttachments(ctx context.Context, tenantID, itemID uuid.UUID) ([]*ItemAttachment, error) {
	return s.repo.ListItemAttachments(ctx, tenantID, itemID)
}

// DeleteItemAttachment removes an attachment registration (the MinIO object is
// left in place; lifecycle/retention handles orphan cleanup).
func (s *Service) DeleteItemAttachment(ctx context.Context, tenantID, attachmentID uuid.UUID) error {
	return s.repo.DeleteItemAttachment(ctx, tenantID, attachmentID)
}
