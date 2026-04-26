package inventar

import (
	"context"

	"github.com/google/uuid"
)

// ListItemsFilter holds optional filtering for item list queries.
type ListItemsFilter struct {
	Search   string
	Location *string
	LowStock bool // only items where quantity <= min_quantity
}

// Repository defines the persistence interface for the inventar module.
type Repository interface {
	// Items
	CreateItem(ctx context.Context, item *Item) error
	UpdateItem(ctx context.Context, item *Item) error
	SoftDeleteItem(ctx context.Context, tenantID, itemID uuid.UUID) error
	GetItem(ctx context.Context, tenantID, itemID uuid.UUID) (*Item, error)
	ListItems(ctx context.Context, tenantID uuid.UUID, filter ListItemsFilter, offset, limit int) ([]*Item, int, error)
	SKUExists(ctx context.Context, tenantID uuid.UUID, sku string, excludeID *uuid.UUID) (bool, error)

	// Movements
	CreateMovement(ctx context.Context, movement *Movement) error
	GetMovement(ctx context.Context, tenantID, movementID uuid.UUID) (*Movement, error)
	ListMovements(ctx context.Context, tenantID, itemID uuid.UUID, offset, limit int) ([]*Movement, int, error)

	// Warnings
	CreateWarning(ctx context.Context, warning *Warning) error
	UpdateWarning(ctx context.Context, warning *Warning) error
	GetWarning(ctx context.Context, tenantID, warningID uuid.UUID) (*Warning, error)
	GetActiveWarningForItem(ctx context.Context, tenantID, itemID uuid.UUID) (*Warning, error)
	ListWarnings(ctx context.Context, tenantID uuid.UUID, status *WarningStatus, offset, limit int) ([]*Warning, int, error)
}
