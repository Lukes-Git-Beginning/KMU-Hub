package bexio

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// SyncResult holds the outcome of a sync operation.
type SyncResult struct {
	ItemsProcessed int      `json:"items_processed"`
	ItemsCreated   int      `json:"items_created"`
	ItemsUpdated   int      `json:"items_updated"`
	ItemsFailed    int      `json:"items_failed"`
	Errors         []string `json:"errors,omitempty"`
}

// ContactService defines the CRM contact operations needed by the sync engine.
type ContactService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*ContactResult, error)
	GetByEmail(ctx context.Context, email string) (*ContactResult, error)
	CreateForSync(ctx context.Context, data *ContactSyncData, createdBy uuid.UUID) (uuid.UUID, error)
	UpdateForSync(ctx context.Context, id uuid.UUID, data *ContactSyncData) error
	ListModifiedSince(ctx context.Context, since time.Time) ([]ContactResult, error)
}

// ContactResult is a simplified contact view for sync operations.
type ContactResult struct {
	ID          uuid.UUID  `json:"id"`
	FirstName   string     `json:"first_name"`
	LastName    string     `json:"last_name"`
	Email       *string    `json:"email,omitempty"`
	Phone       *string    `json:"phone,omitempty"`
	CompanyName *string    `json:"company_name,omitempty"`
	Notes       *string    `json:"notes,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ContactSyncer handles bidirectional contact sync between KMU Hub and Bexio.
type ContactSyncer struct {
	client      *Client
	repo        Repository
	fieldMapper *FieldMapper
	contacts    ContactService
	lookupCache *LookupCache
}

// NewContactSyncer creates a new contact syncer.
func NewContactSyncer(client *Client, repo Repository, fieldMapper *FieldMapper, contacts ContactService) *ContactSyncer {
	return &ContactSyncer{
		client:      client,
		repo:        repo,
		fieldMapper: fieldMapper,
		contacts:    contacts,
		lookupCache: NewLookupCache(),
	}
}

// SyncContacts performs a full bidirectional contact sync.
func (cs *ContactSyncer) SyncContacts(ctx context.Context, configID, tenantID uuid.UUID) (*SyncResult, error) {
	result := &SyncResult{}

	// Create sync log entry
	syncLog := &models.BexioSyncLog{
		ConfigID:  configID,
		SyncType:  "contact_delta",
		Status:    "running",
		StartedAt: time.Now().UTC(),
		Metadata:  map[string]any{},
	}

	// Determine if this is a full or delta sync
	syncConfig, err := cs.repo.GetSyncConfig(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("contact sync: get config: %w", err)
	}

	var updatedSince *time.Time
	if syncConfig.LastContactSyncAt != nil {
		updatedSince = syncConfig.LastContactSyncAt
	} else {
		syncLog.SyncType = "contact_full"
	}

	if err := cs.repo.CreateSyncLog(ctx, syncLog); err != nil {
		slog.Error("bexio: failed to create sync log", "error", err)
	}

	// Load field mappings
	fieldMappings, err := cs.repo.GetFieldMappings(ctx, configID, "contact")
	if err != nil {
		return nil, fmt.Errorf("contact sync: get field mappings: %w", err)
	}
	var mappingEntries []models.BexioFieldMappingEntry
	if fieldMappings != nil {
		mappingEntries = fieldMappings.Mappings
	} else {
		mappingEntries = DefaultContactMappings()
	}

	// Refresh lookup cache
	if err := cs.lookupCache.Refresh(ctx, cs.client, tenantID); err != nil {
		slog.Error("bexio: failed to refresh lookup cache", "error", err)
	}

	// Inbound sync (Bexio -> KMU Hub)
	cs.syncInbound(ctx, configID, tenantID, updatedSince, mappingEntries, result)

	// Outbound sync (KMU Hub -> Bexio)
	if updatedSince != nil {
		cs.syncOutbound(ctx, configID, tenantID, *updatedSince, mappingEntries, result)
	}

	// Update last sync time
	now := time.Now().UTC()
	if err := cs.repo.UpdateLastSyncTime(ctx, configID, syncLog.SyncType, now); err != nil {
		slog.Error("bexio: failed to update last sync time", "error", err)
	}

	// Finalize sync log
	syncLog.ItemsProcessed = result.ItemsProcessed
	syncLog.ItemsCreated = result.ItemsCreated
	syncLog.ItemsUpdated = result.ItemsUpdated
	syncLog.ItemsFailed = result.ItemsFailed
	syncLog.CompletedAt = &now

	if result.ItemsFailed > 0 && result.ItemsCreated+result.ItemsUpdated > 0 {
		syncLog.Status = "partial"
	} else if result.ItemsFailed > 0 {
		syncLog.Status = "failed"
		errMsg := fmt.Sprintf("%d items failed", result.ItemsFailed)
		syncLog.ErrorMessage = &errMsg
	} else {
		syncLog.Status = "completed"
	}

	if err := cs.repo.UpdateSyncLog(ctx, syncLog); err != nil {
		slog.Error("bexio: failed to update sync log", "error", err)
	}

	slog.Info("bexio contact sync completed",
		"config_id", configID,
		"processed", result.ItemsProcessed,
		"created", result.ItemsCreated,
		"updated", result.ItemsUpdated,
		"failed", result.ItemsFailed,
	)

	return result, nil
}

func (cs *ContactSyncer) syncInbound(ctx context.Context, configID, tenantID uuid.UUID, updatedSince *time.Time, mappings []models.BexioFieldMappingEntry, result *SyncResult) {
	// G2: ContactService may not be wired; skip inbound create/update paths
	// that call into the CRM to avoid nil-pointer panics.
	if cs.contacts == nil {
		slog.Warn("bexio: contact service not wired, skipping inbound sync")
		return
	}

	bexioContacts, err := cs.client.ListContacts(ctx, tenantID, updatedSince)
	if err != nil {
		slog.Error("bexio: failed to list contacts from Bexio", "error", err)
		result.ItemsFailed++
		result.Errors = append(result.Errors, fmt.Sprintf("list bexio contacts: %v", err))
		return
	}

	for _, bc := range bexioContacts {
		result.ItemsProcessed++

		// Check existing mapping
		mapping, err := cs.repo.GetEntityMappingByBexioID(ctx, configID, "contact", bc.ID)
		if err != nil && !errors.Is(err, ErrMappingNotFound) {
			result.ItemsFailed++
			result.Errors = append(result.Errors, fmt.Sprintf("lookup mapping for bexio %d: %v", bc.ID, err))
			continue
		}

		syncData, err := cs.fieldMapper.MapContactToKMUHub(&bc, mappings, cs.lookupCache)
		if err != nil {
			result.ItemsFailed++
			result.Errors = append(result.Errors, fmt.Sprintf("map bexio contact %d: %v", bc.ID, err))
			continue
		}

		now := time.Now().UTC()

		if mapping != nil {
			// Existing mapping — check last-write-wins
			bexioTime := parseBexioTime(bc.UpdatedAt)
			if mapping.KmuhubUpdatedAt != nil && bexioTime != nil && !bexioTime.After(*mapping.KmuhubUpdatedAt) {
				// KMU Hub version is newer or equal, skip inbound update
				continue
			}

			// Bexio is newer — update KMU Hub
			if err := cs.contacts.UpdateForSync(ctx, mapping.KmuhubID, syncData); err != nil {
				result.ItemsFailed++
				result.Errors = append(result.Errors, fmt.Sprintf("update KMU Hub contact %s: %v", mapping.KmuhubID, err))
				continue
			}

			mapping.LastSyncedAt = now
			mapping.BexioUpdatedAt = bexioTime
			if err := cs.repo.UpsertEntityMapping(ctx, mapping); err != nil {
				slog.Error("bexio: failed to update mapping", "bexio_id", bc.ID, "error", err)
			}
			result.ItemsUpdated++
		} else {
			// New contact from Bexio — create in KMU Hub
			kmuhubID, err := cs.contacts.CreateForSync(ctx, syncData, uuid.Nil)
			if err != nil {
				result.ItemsFailed++
				result.Errors = append(result.Errors, fmt.Sprintf("create KMU Hub contact from bexio %d: %v", bc.ID, err))
				continue
			}

			bexioTime := parseBexioTime(bc.UpdatedAt)
			newMapping := &models.BexioEntityMapping{
				ConfigID:       configID,
				EntityType:     "contact",
				KmuhubID:       kmuhubID,
				BexioID:        bc.ID,
				LastSyncedAt:   now,
				BexioUpdatedAt: bexioTime,
				SyncDirection:  "both",
			}
			if err := cs.repo.UpsertEntityMapping(ctx, newMapping); err != nil {
				slog.Error("bexio: failed to create mapping", "bexio_id", bc.ID, "error", err)
			}
			result.ItemsCreated++
		}
	}
}

func (cs *ContactSyncer) syncOutbound(ctx context.Context, configID, tenantID uuid.UUID, since time.Time, mappings []models.BexioFieldMappingEntry, result *SyncResult) {
	// G2: ContactService may not be wired (e.g. CRM gRPC unavailable at startup).
	// Guard here so a nil cs.contacts never panics; the inbound path is unaffected.
	if cs.contacts == nil {
		slog.Warn("bexio: contact service not wired, skipping outbound sync")
		return
	}

	modified, err := cs.contacts.ListModifiedSince(ctx, since)
	if err != nil {
		slog.Error("bexio: failed to list modified contacts", "error", err)
		result.ItemsFailed++
		result.Errors = append(result.Errors, fmt.Sprintf("list modified contacts: %v", err))
		return
	}

	for _, contact := range modified {
		result.ItemsProcessed++

		mapping, err := cs.repo.GetEntityMapping(ctx, configID, "contact", contact.ID)
		if err != nil && !errors.Is(err, ErrMappingNotFound) {
			result.ItemsFailed++
			continue
		}

		// Build a minimal models.Contact for the mapper
		mc := &models.Contact{
			ID:        contact.ID,
			FirstName: contact.FirstName,
			LastName:  contact.LastName,
			Email:     contact.Email,
			Phone:     contact.Phone,
			Notes:     contact.Notes,
			UpdatedAt: contact.UpdatedAt,
		}

		now := time.Now().UTC()

		if mapping != nil {
			// Existing mapping — check last-write-wins
			if mapping.BexioUpdatedAt != nil && !contact.UpdatedAt.After(*mapping.BexioUpdatedAt) {
				// Bexio version is newer or equal, skip outbound update
				continue
			}

			bexioContact, err := cs.fieldMapper.MapContactToBexio(mc, contact.CompanyName, mappings, cs.lookupCache)
			if err != nil {
				result.ItemsFailed++
				continue
			}

			if _, err := cs.client.UpdateContact(ctx, tenantID, mapping.BexioID, bexioContact); err != nil {
				result.ItemsFailed++
				result.Errors = append(result.Errors, fmt.Sprintf("update bexio contact %d: %v", mapping.BexioID, err))
				continue
			}

			mapping.LastSyncedAt = now
			mapping.KmuhubUpdatedAt = &contact.UpdatedAt
			if err := cs.repo.UpsertEntityMapping(ctx, mapping); err != nil {
				slog.Error("bexio: failed to update mapping", "kmuhub_id", contact.ID, "error", err)
			}
			result.ItemsUpdated++
		} else {
			// New contact — create in Bexio
			bexioContact, err := cs.fieldMapper.MapContactToBexio(mc, contact.CompanyName, mappings, cs.lookupCache)
			if err != nil {
				result.ItemsFailed++
				continue
			}

			created, err := cs.client.CreateContact(ctx, tenantID, bexioContact)
			if err != nil {
				result.ItemsFailed++
				result.Errors = append(result.Errors, fmt.Sprintf("create bexio contact for %s: %v", contact.ID, err))
				continue
			}

			newMapping := &models.BexioEntityMapping{
				ConfigID:        configID,
				EntityType:      "contact",
				KmuhubID:        contact.ID,
				BexioID:         created.ID,
				LastSyncedAt:    now,
				KmuhubUpdatedAt: &contact.UpdatedAt,
				SyncDirection:   "both",
			}
			if err := cs.repo.UpsertEntityMapping(ctx, newMapping); err != nil {
				slog.Error("bexio: failed to create mapping", "kmuhub_id", contact.ID, "error", err)
			}
			result.ItemsCreated++
		}
	}
}

// parseBexioTime attempts to parse a Bexio timestamp string.
func parseBexioTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}
