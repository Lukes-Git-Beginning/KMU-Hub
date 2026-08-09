package lexware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

type SyncResult struct {
	ItemsProcessed int      `json:"items_processed"`
	ItemsCreated   int      `json:"items_created"`
	ItemsUpdated   int      `json:"items_updated"`
	ItemsFailed    int      `json:"items_failed"`
	Errors         []string `json:"errors,omitempty"`
}

type ContactService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*ContactResult, error)
	GetByEmail(ctx context.Context, email string) (*ContactResult, error)
	CreateForSync(ctx context.Context, data *ContactSyncData, createdBy uuid.UUID) (uuid.UUID, error)
	UpdateForSync(ctx context.Context, id uuid.UUID, data *ContactSyncData) error
	ListModifiedSince(ctx context.Context, since time.Time) ([]ContactResult, error)
}

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

type ContactSyncer struct {
	client      *Client
	repo        Repository
	fieldMapper *FieldMapper
	contacts    ContactService
}

func NewContactSyncer(client *Client, repo Repository, fieldMapper *FieldMapper, contacts ContactService) *ContactSyncer {
	return &ContactSyncer{
		client:      client,
		repo:        repo,
		fieldMapper: fieldMapper,
		contacts:    contacts,
	}
}

func (cs *ContactSyncer) SyncContacts(ctx context.Context, configID, tenantID uuid.UUID) (*SyncResult, error) {
	result := &SyncResult{}

	syncConfig, err := cs.repo.GetSyncConfig(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("contact sync: get config: %w", err)
	}

	syncLog := &models.LexwareSyncLog{
		TenantID:  tenantID,
		ConfigID:  configID,
		SyncType:  "contact_delta",
		Status:    "running",
		StartedAt: time.Now().UTC(),
		Metadata:  map[string]any{},
	}

	var updatedSince *time.Time
	if syncConfig.LastContactSyncAt != nil {
		updatedSince = syncConfig.LastContactSyncAt
	} else {
		syncLog.SyncType = "contact_full"
	}

	if err := cs.repo.CreateSyncLog(ctx, syncLog); err != nil {
		slog.Error("lexware: failed to create sync log", "error", err)
	}

	mappingEntries, err := cs.contactFieldMappings(ctx, configID)
	if err != nil {
		return nil, err
	}

	cs.syncInbound(ctx, configID, tenantID, mappingEntries, result)

	if updatedSince != nil {
		cs.syncOutbound(ctx, configID, tenantID, *updatedSince, mappingEntries, result)
	}

	now := time.Now().UTC()
	if err := cs.repo.UpdateLastSyncTime(ctx, configID, syncLog.SyncType, now); err != nil {
		slog.Error("lexware: failed to update last sync time", "error", err)
	}

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
		slog.Error("lexware: failed to update sync log", "error", err)
	}

	slog.Info("lexware contact sync completed",
		"config_id", configID,
		"processed", result.ItemsProcessed,
		"created", result.ItemsCreated,
		"updated", result.ItemsUpdated,
		"failed", result.ItemsFailed,
	)

	return result, nil
}

// contactFieldMappings loads the tenant's configured contact field mappings,
// falling back to the built-in defaults when none are stored.
func (cs *ContactSyncer) contactFieldMappings(ctx context.Context, configID uuid.UUID) ([]models.LexwareFieldMappingEntry, error) {
	fieldMappings, err := cs.repo.GetFieldMappings(ctx, configID, "contact")
	if err != nil {
		return nil, fmt.Errorf("contact sync: get field mappings: %w", err)
	}
	if fieldMappings != nil {
		return fieldMappings.Mappings, nil
	}
	return DefaultContactMappings(), nil
}

// SyncContactByLexwareID pulls a single contact from Lexware and applies it to
// the CRM, using the same create/update/skip rules as a full inbound sync.
// This is the path a contact.created / contact.changed webhook takes: the event
// carries only the resource ID, so the record itself must be fetched.
func (cs *ContactSyncer) SyncContactByLexwareID(ctx context.Context, configID, tenantID uuid.UUID, lexwareID string) (*SyncResult, error) {
	if lexwareID == "" {
		return nil, fmt.Errorf("contact sync: empty lexware contact id")
	}
	if cs.contacts == nil {
		return nil, fmt.Errorf("contact sync: contact service not wired")
	}

	mappings, err := cs.contactFieldMappings(ctx, configID)
	if err != nil {
		return nil, err
	}

	lc, err := cs.client.GetContact(ctx, tenantID, lexwareID)
	if err != nil {
		return nil, fmt.Errorf("contact sync: fetch lexware contact %s: %w", lexwareID, err)
	}

	result := &SyncResult{}
	cs.upsertInboundContact(ctx, configID, tenantID, lc, mappings, result)

	if result.ItemsFailed > 0 {
		return result, fmt.Errorf("contact sync: lexware contact %s failed: %s", lexwareID, strings.Join(result.Errors, "; "))
	}
	return result, nil
}

func (cs *ContactSyncer) syncInbound(ctx context.Context, configID, tenantID uuid.UUID, mappings []models.LexwareFieldMappingEntry, result *SyncResult) {
	// ContactService may not be wired (CRM gRPC address unset); the inbound path
	// writes into the CRM, so skip it rather than panic on a nil interface.
	if cs.contacts == nil {
		slog.Warn("lexware: contact service not wired, skipping inbound sync")
		return
	}

	page := 0
	for {
		listResp, err := cs.client.ListContacts(ctx, tenantID, page, 250)
		if err != nil {
			slog.Error("lexware: failed to list contacts", "error", err, "page", page)
			result.ItemsFailed++
			result.Errors = append(result.Errors, fmt.Sprintf("list lexware contacts page %d: %v", page, err))
			return
		}

		for i := range listResp.Content {
			cs.upsertInboundContact(ctx, configID, tenantID, &listResp.Content[i], mappings, result)
		}

		if listResp.Last {
			break
		}
		page++
	}
}

// upsertInboundContact applies one Lexware contact to the CRM: update the
// mapped contact when Lexware's copy is newer, otherwise create it and record
// the mapping. Failures are accumulated in result rather than returned, so a
// bulk sync keeps going past a single bad record.
func (cs *ContactSyncer) upsertInboundContact(
	ctx context.Context,
	configID, tenantID uuid.UUID,
	lc *LexwareContact,
	mappings []models.LexwareFieldMappingEntry,
	result *SyncResult,
) {
	result.ItemsProcessed++

	mapping, err := cs.repo.GetEntityMappingByLexwareID(ctx, configID, "contact", lc.ID)
	if err != nil && !errors.Is(err, ErrMappingNotFound) {
		result.ItemsFailed++
		result.Errors = append(result.Errors, fmt.Sprintf("lookup mapping for lexware %s: %v", lc.ID, err))
		return
	}

	syncData, err := cs.fieldMapper.MapContactToKMUHub(lc, mappings)
	if err != nil {
		result.ItemsFailed++
		result.Errors = append(result.Errors, fmt.Sprintf("map lexware contact %s: %v", lc.ID, err))
		return
	}

	now := time.Now().UTC()
	lexwareTime := parseLexwareTime(lc.UpdatedDate)

	if mapping != nil {
		if mapping.KmuhubUpdatedAt != nil && lexwareTime != nil && !lexwareTime.After(*mapping.KmuhubUpdatedAt) {
			return
		}

		if err := cs.contacts.UpdateForSync(ctx, mapping.KmuhubID, syncData); err != nil {
			result.ItemsFailed++
			result.Errors = append(result.Errors, fmt.Sprintf("update KMU Hub contact %s: %v", mapping.KmuhubID, err))
			return
		}

		mapping.LastSyncedAt = now
		mapping.LexwareUpdatedAt = lexwareTime
		mapping.LexwareVersion = lc.Version
		if err := cs.repo.UpsertEntityMapping(ctx, mapping); err != nil {
			slog.Error("lexware: failed to update mapping", "lexware_id", lc.ID, "error", err)
		}
		result.ItemsUpdated++
		return
	}

	kmuhubID, err := cs.contacts.CreateForSync(ctx, syncData, uuid.Nil)
	if err != nil {
		result.ItemsFailed++
		result.Errors = append(result.Errors, fmt.Sprintf("create KMU Hub contact from lexware %s: %v", lc.ID, err))
		return
	}

	newMapping := &models.LexwareEntityMapping{
		TenantID:         tenantID,
		ConfigID:         configID,
		EntityType:       "contact",
		KmuhubID:         kmuhubID,
		LexwareID:        lc.ID,
		LexwareVersion:   lc.Version,
		LastSyncedAt:     now,
		LexwareUpdatedAt: lexwareTime,
		SyncDirection:    "both",
	}
	if err := cs.repo.UpsertEntityMapping(ctx, newMapping); err != nil {
		slog.Error("lexware: failed to create mapping", "lexware_id", lc.ID, "error", err)
	}
	result.ItemsCreated++
}

func (cs *ContactSyncer) syncOutbound(ctx context.Context, configID, tenantID uuid.UUID, since time.Time, mappings []models.LexwareFieldMappingEntry, result *SyncResult) {
	if cs.contacts == nil {
		slog.Warn("lexware: contact service not wired, skipping outbound sync")
		return
	}

	modified, err := cs.contacts.ListModifiedSince(ctx, since)
	if err != nil {
		slog.Error("lexware: failed to list modified contacts", "error", err)
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
			if mapping.LexwareUpdatedAt != nil && !contact.UpdatedAt.After(*mapping.LexwareUpdatedAt) {
				continue
			}

			lexwareContact, err := cs.fieldMapper.MapContactToLexware(mc, contact.CompanyName, mappings)
			if err != nil {
				result.ItemsFailed++
				continue
			}

			lexwareContact.Version = mapping.LexwareVersion

			updated, err := cs.client.UpdateContact(ctx, tenantID, mapping.LexwareID, lexwareContact)
			if err != nil {
				result.ItemsFailed++
				result.Errors = append(result.Errors, fmt.Sprintf("update lexware contact %s: %v", mapping.LexwareID, err))
				continue
			}

			mapping.LastSyncedAt = now
			mapping.KmuhubUpdatedAt = &contact.UpdatedAt
			mapping.LexwareVersion = updated.Version
			if err := cs.repo.UpsertEntityMapping(ctx, mapping); err != nil {
				slog.Error("lexware: failed to update mapping", "kmuhub_id", contact.ID, "error", err)
			}
			result.ItemsUpdated++
		} else {
			lexwareContact, err := cs.fieldMapper.MapContactToLexware(mc, contact.CompanyName, mappings)
			if err != nil {
				result.ItemsFailed++
				continue
			}

			created, err := cs.client.CreateContact(ctx, tenantID, lexwareContact)
			if err != nil {
				result.ItemsFailed++
				result.Errors = append(result.Errors, fmt.Sprintf("create lexware contact for %s: %v", contact.ID, err))
				continue
			}

			newMapping := &models.LexwareEntityMapping{
				TenantID:        tenantID,
				ConfigID:        configID,
				EntityType:      "contact",
				KmuhubID:        contact.ID,
				LexwareID:       created.ID,
				LexwareVersion:  created.Version,
				LastSyncedAt:    now,
				KmuhubUpdatedAt: &contact.UpdatedAt,
				SyncDirection:   "both",
			}
			if err := cs.repo.UpsertEntityMapping(ctx, newMapping); err != nil {
				slog.Error("lexware: failed to create mapping", "kmuhub_id", contact.ID, "error", err)
			}
			result.ItemsCreated++
		}
	}
}

func parseLexwareTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05.000Z", "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}
