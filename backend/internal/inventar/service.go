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
	TenantID    uuid.UUID
	Name        string
	SKU         string
	Barcode     *string
	Quantity    int64
	MinQuantity int64
	Unit        string
	Location    *string
}

// UpdateItemInput contains fields that can be updated on an item.
type UpdateItemInput struct {
	TenantID    uuid.UUID
	ItemID      uuid.UUID
	Name        *string
	SKU         *string
	Barcode     *string
	MinQuantity *int64
	Unit        *string
	Location    *string
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
	TotalItems    int
	TotalQuantity int64
	LowStockCount int
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
		ID:          uuid.New(),
		TenantID:    input.TenantID,
		Name:        name,
		SKU:         sku,
		Barcode:     input.Barcode,
		Quantity:    input.Quantity,
		MinQuantity: input.MinQuantity,
		Unit:        input.Unit,
		Location:    input.Location,
		CreatedAt:   now,
		UpdatedAt:   now,
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

	item.Quantity = max(item.Quantity+input.Delta, 0)
	item.UpdatedAt = time.Now()

	if updateErr := s.repo.UpdateItem(ctx, item); updateErr != nil {
		return nil, fmt.Errorf("update item quantity: %w", updateErr)
	}

	movementType := MovementTypeIn
	if input.Delta < 0 {
		movementType = MovementTypeOut
	} else {
		movementType = MovementTypeAdjustment
	}

	movement := &Movement{
		ID:           uuid.New(),
		TenantID:     input.TenantID,
		ItemID:       item.ID,
		MovementType: movementType,
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

	now := time.Now()

	// Deduct from source
	fromItem.Quantity -= input.Quantity
	if fromItem.Quantity < 0 {
		fromItem.Quantity = 0
	}
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
