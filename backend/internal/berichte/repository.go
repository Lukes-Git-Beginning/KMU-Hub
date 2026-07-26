package berichte

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ListDefinitionsFilter narrows a definition listing.
type ListDefinitionsFilter struct {
	Module      *string
	Kind        *string
	IsPublished *bool
	Search      string
	SortBy      string // name, created_at, updated_at
	SortDesc    bool
}

// ListSchedulesFilter narrows a schedule listing.
type ListSchedulesFilter struct {
	DefinitionID *uuid.UUID
	Active       *bool
}

// ListDocumentsFilter narrows a report document listing.
type ListDocumentsFilter struct {
	Module *string
	Status *string
	Search string
}

// Repository defines the persistence interface for the berichte module.
type Repository interface {
	// Definitions
	CreateDefinition(ctx context.Context, def *Definition) error
	UpdateDefinition(ctx context.Context, def *Definition) error
	DeleteDefinition(ctx context.Context, tenantID, definitionID uuid.UUID) error
	GetDefinition(ctx context.Context, tenantID, definitionID uuid.UUID) (*Definition, error)
	ListDefinitions(ctx context.Context, tenantID uuid.UUID, filter ListDefinitionsFilter, offset, limit int) ([]*Definition, int, error)

	// Cache
	GetCacheEntry(ctx context.Context, tenantID, definitionID uuid.UUID, paramsHash string) (*CacheEntry, error)
	UpsertCacheEntry(ctx context.Context, entry *CacheEntry) error
	InvalidateCache(ctx context.Context, tenantID, definitionID uuid.UUID) (int, error)
	DeleteExpiredCacheEntries(ctx context.Context, before time.Time) (int, error)

	// Schedules
	CreateSchedule(ctx context.Context, sch *Schedule) error
	UpdateSchedule(ctx context.Context, sch *Schedule) error
	DeleteSchedule(ctx context.Context, tenantID, scheduleID uuid.UUID) error
	GetSchedule(ctx context.Context, tenantID, scheduleID uuid.UUID) (*Schedule, error)
	ListSchedules(ctx context.Context, tenantID uuid.UUID, filter ListSchedulesFilter, offset, limit int) ([]*Schedule, int, error)
	ListDueSchedules(ctx context.Context, now time.Time) ([]*Schedule, error)
	ClaimSchedule(ctx context.Context, scheduleID uuid.UUID, previousLastRunAt *time.Time, now time.Time) (bool, error)
	UpdateScheduleLastRun(ctx context.Context, scheduleID uuid.UUID, status string, runErr *string, at time.Time) error

	// Runs
	InsertRun(ctx context.Context, run *Run) error

	// Documents (multi-page authoring)
	CreateDocument(ctx context.Context, doc *Document) error
	UpdateDocument(ctx context.Context, doc *Document) error
	DeleteDocument(ctx context.Context, tenantID, documentID uuid.UUID) error
	GetDocument(ctx context.Context, tenantID, documentID uuid.UUID) (*Document, error)
	ListDocuments(ctx context.Context, tenantID uuid.UUID, filter ListDocumentsFilter, offset, limit int) ([]*Document, int, error)
}
