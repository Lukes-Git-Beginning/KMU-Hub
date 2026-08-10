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

// InvoiceImporter is the read-only import sink for invoices pulled from Bexio.
// It is satisfied structurally by *invoice.Service; kept as a narrow interface so
// the bexio package never imports the finance package (mirror of InvoiceReader in
// invoice_push.go and InvoiceStatusUpdater in payment_poll.go).
//
// GetByID is needed for loop-prevention: a Bexio invoice whose Cosmi mapping points
// at a source='cosmi' record was pushed FROM Cosmi and must never be re-imported as
// a read-only mirror.
type InvoiceImporter interface {
	UpsertImported(ctx context.Context, tenantID uuid.UUID, in models.ImportedInvoiceInput) (*models.Invoice, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*models.Invoice, error)
}

// InvoicePuller mirrors Bexio-native invoices into Cosmi as read-only records
// (source='bexio'). It never writes back to Bexio and never assigns a Cosmi invoice
// number — the finance import path (UpsertImported) enforces that. Delta-forward: the
// cursor (last_invoice_pull_at) is seeded to NOW() on activation, so no history is
// back-filled.
type InvoicePuller struct {
	client      invoiceLister
	repo        Repository
	fieldMapper *FieldMapper
	importer    InvoiceImporter
	lookupCache *LookupCache
}

// invoiceLister is the slice of the Bexio client the puller needs; a narrow interface
// keeps the puller unit-testable without an HTTP round-trip. *Client satisfies it.
type invoiceLister interface {
	ListInvoices(ctx context.Context, tenantID uuid.UUID, updatedSince *time.Time) ([]BexioInvoice, error)
}

// NewInvoicePuller creates a new invoice puller.
func NewInvoicePuller(client *Client, repo Repository, fieldMapper *FieldMapper, importer InvoiceImporter) *InvoicePuller {
	return &InvoicePuller{
		client:      client,
		repo:        repo,
		fieldMapper: fieldMapper,
		importer:    importer,
		// lean: the reverse mapper only reads the currency cache in a no-op path
		// (resolveKMUHubCurrency), so the cache is created but not refreshed here to
		// avoid extra Bexio API calls. Refresh (like ContactSyncer) when imported
		// foreign-currency invoices must show their true currency.
		lookupCache: NewLookupCache(),
	}
}

// PullInvoices imports invoices changed in Bexio since the last pull cursor. configID
// is passed explicitly; see Service.PullInvoicesWithConfig for the G8 rationale.
func (ip *InvoicePuller) PullInvoices(ctx context.Context, configID, tenantID uuid.UUID) (*SyncResult, error) {
	result := &SyncResult{}

	// The importer (finance service) may not be wired in every deployment; guard so a
	// nil sink never panics the scheduler goroutine (G2 pattern, contact_sync.go:161).
	if ip.importer == nil {
		slog.Warn("bexio: invoice importer not wired, skipping invoice pull")
		return result, nil
	}

	syncConfig, err := ip.repo.GetSyncConfig(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("invoice pull: get config: %w", err)
	}

	// Delta-forward: the first pull imports no history. When the cursor is unset (never
	// pulled — e.g. just activated), seed it to now and return; only changes from here
	// on are mirrored. This is the single enforcement point, robust across the scheduled
	// tick and the manual trigger.
	if syncConfig.LastInvoicePullAt == nil {
		now := time.Now().UTC()
		if err := ip.repo.UpdateLastSyncTime(ctx, configID, "invoice_pull", now); err != nil {
			return nil, fmt.Errorf("invoice pull: seed cursor: %w", err)
		}
		slog.Info("bexio invoice pull: cursor seeded delta-forward, no history imported", "config_id", configID)
		return result, nil
	}

	syncLog := &models.BexioSyncLog{
		TenantID:  tenantID,
		ConfigID:  configID,
		SyncType:  "invoice_pull",
		Status:    "running",
		StartedAt: time.Now().UTC(),
		Metadata:  map[string]any{},
	}
	if err := ip.repo.CreateSyncLog(ctx, syncLog); err != nil {
		slog.Error("bexio: failed to create invoice pull log", "error", err)
	}

	bexioInvoices, err := ip.client.ListInvoices(ctx, tenantID, syncConfig.LastInvoicePullAt)
	if err != nil {
		// Do NOT advance the cursor on a list failure — the whole delta window must be
		// retried on the next tick.
		result.ItemsFailed++
		result.Errors = append(result.Errors, fmt.Sprintf("list bexio invoices: %v", err))
		ip.finalizeLog(ctx, syncLog, result, 0, false)
		return result, fmt.Errorf("invoice pull: list invoices: %w", err)
	}

	skipped := 0
	for i := range bexioInvoices {
		bi := bexioInvoices[i]
		result.ItemsProcessed++

		mapping, err := ip.repo.GetEntityMappingByBexioID(ctx, configID, "invoice", bi.ID)
		if err != nil && !errors.Is(err, ErrMappingNotFound) {
			result.ItemsFailed++
			result.Errors = append(result.Errors, fmt.Sprintf("lookup invoice mapping for bexio %d: %v", bi.ID, err))
			continue
		}

		bexioTime := parseBexioTime(bi.UpdatedAt)

		if mapping != nil {
			// Loop-prevention: if the mapped Cosmi invoice is source='cosmi', it was
			// pushed FROM Cosmi — never re-import our own invoice as a bexio mirror.
			existing, gerr := ip.importer.GetByID(ctx, tenantID, mapping.KmuhubID)
			if gerr != nil {
				result.ItemsFailed++
				result.Errors = append(result.Errors, fmt.Sprintf("load mapped invoice %s: %v", mapping.KmuhubID, gerr))
				continue
			}
			if existing.Source == models.InvoiceSourceCosmi {
				skipped++
				continue
			}

			// LWW: skip if the imported mirror is already at least as fresh as this
			// Bexio version.
			if mapping.BexioUpdatedAt != nil && bexioTime != nil && !bexioTime.After(*mapping.BexioUpdatedAt) {
				continue
			}
		}

		// Resolve the Cosmi contact via the existing contact entity-mapping; unresolved
		// contacts import with uuid.Nil (the mapper tolerates it).
		contactID := uuid.Nil
		if bi.ContactID != 0 {
			if cm, cerr := ip.repo.GetEntityMappingByBexioID(ctx, configID, "contact", bi.ContactID); cerr == nil && cm != nil {
				contactID = cm.KmuhubID
			}
		}

		in, err := ip.fieldMapper.MapInvoiceToKMUHub(&bi, contactID, nil, ip.lookupCache)
		if err != nil {
			result.ItemsFailed++
			result.Errors = append(result.Errors, fmt.Sprintf("map bexio invoice %d: %v", bi.ID, err))
			continue
		}

		imported, err := ip.importer.UpsertImported(ctx, tenantID, *in)
		if err != nil {
			result.ItemsFailed++
			result.Errors = append(result.Errors, fmt.Sprintf("import bexio invoice %d: %v", bi.ID, err))
			continue
		}

		wasCreate := mapping == nil
		if mapping == nil {
			mapping = &models.BexioEntityMapping{
				TenantID:      tenantID,
				ConfigID:      configID,
				EntityType:    "invoice",
				KmuhubID:      imported.ID,
				BexioID:       bi.ID,
				SyncDirection: "inbound",
			}
		}
		mapping.LastSyncedAt = time.Now().UTC()
		mapping.BexioUpdatedAt = bexioTime
		if err := ip.repo.UpsertEntityMapping(ctx, mapping); err != nil {
			slog.Error("bexio: failed to upsert invoice mapping", "bexio_id", bi.ID, "error", err)
		}

		if wasCreate {
			result.ItemsCreated++
		} else {
			result.ItemsUpdated++
		}
	}

	ip.finalizeLog(ctx, syncLog, result, skipped, true)

	// Advance the delta cursor to now only after a successful list. Matches the
	// contact-sync semantics (cursor = now, not last-item timestamp).
	if err := ip.repo.UpdateLastSyncTime(ctx, configID, "invoice_pull", time.Now().UTC()); err != nil {
		slog.Error("bexio: failed to update last invoice pull time", "error", err)
	}

	slog.Info("bexio invoice pull completed",
		"config_id", configID,
		"processed", result.ItemsProcessed,
		"created", result.ItemsCreated,
		"updated", result.ItemsUpdated,
		"skipped_cosmi_origin", skipped,
		"failed", result.ItemsFailed,
	)

	return result, nil
}

// finalizeLog writes the terminal state of the sync-log row.
func (ip *InvoicePuller) finalizeLog(ctx context.Context, syncLog *models.BexioSyncLog, result *SyncResult, skipped int, listed bool) {
	completeTime := time.Now().UTC()
	syncLog.ItemsProcessed = result.ItemsProcessed
	syncLog.ItemsCreated = result.ItemsCreated
	syncLog.ItemsUpdated = result.ItemsUpdated
	syncLog.ItemsFailed = result.ItemsFailed
	syncLog.CompletedAt = &completeTime
	syncLog.Metadata["skipped_cosmi_origin"] = skipped

	switch {
	case !listed:
		syncLog.Status = "failed"
		if len(result.Errors) > 0 {
			syncLog.ErrorMessage = &result.Errors[0]
		}
	case result.ItemsFailed > 0 && result.ItemsCreated+result.ItemsUpdated > 0:
		syncLog.Status = "partial"
	case result.ItemsFailed > 0:
		syncLog.Status = "failed"
		errMsg := fmt.Sprintf("%d items failed", result.ItemsFailed)
		syncLog.ErrorMessage = &errMsg
	default:
		syncLog.Status = "completed"
	}

	if err := ip.repo.UpdateSyncLog(ctx, syncLog); err != nil {
		slog.Error("bexio: failed to update invoice pull log", "error", err)
	}
}
