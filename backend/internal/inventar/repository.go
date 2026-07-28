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

	// Locations
	CreateLocation(ctx context.Context, loc *InventoryLocation) error
	UpdateLocation(ctx context.Context, loc *InventoryLocation) error
	SoftDeleteLocation(ctx context.Context, tenantID, locID uuid.UUID) error
	GetLocation(ctx context.Context, tenantID, locID uuid.UUID) (*InventoryLocation, error)
	ListLocations(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*InventoryLocation, int, error)

	// Inventur Sessions
	CreateInventurSession(ctx context.Context, session *InventurSession) error
	UpdateInventurSession(ctx context.Context, session *InventurSession) error
	DeleteInventurSession(ctx context.Context, tenantID, sessionID uuid.UUID) error
	GetInventurSession(ctx context.Context, tenantID, sessionID uuid.UUID) (*InventurSession, error)
	ListInventurSessions(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*InventurSession, int, error)

	// Inventur Counts
	UpsertInventurCount(ctx context.Context, count *InventurCount) error
	ListInventurCounts(ctx context.Context, tenantID, sessionID uuid.UUID) ([]*InventurCount, error)

	// Item Attachments
	CreateItemAttachment(ctx context.Context, att *ItemAttachment) error
	ListItemAttachments(ctx context.Context, tenantID, itemID uuid.UUID) ([]*ItemAttachment, error)
	DeleteItemAttachment(ctx context.Context, tenantID, attachmentID uuid.UUID) error
}
