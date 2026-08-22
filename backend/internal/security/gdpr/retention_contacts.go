package gdpr

// ContactRetentionHandler implements RetentionHandler for resource_type
// "contacts" — the first real handler wired into the retention engine
// (retention.go).
//
// Two RESTRICT foreign keys (dialer_campaign_contacts, advisory_protocols)
// keep some contacts from being hard-deleted, the same gate the manual
// contact.Service.Delete path already respects via repo.IsInUse. This
// handler reuses that check rather than rebuilding it, and never escalates
// on its own: a policy configured with action "delete" skips a blocked
// contact and records why, instead of failing the whole run or silently
// anonymizing behind the administrator's back. A policy configured with
// "anonymize" reaches blocked contacts too, because consent.AnonymizeContact
// never touches the two RESTRICT-guarded tables.
//
// Idempotency comes for free: AnonymizeContact bumps updated_at to NOW(), so
// a repeated run's ListRetentionCandidates query — which keys off the later
// of updated_at and the contact's most recent activity — no longer matches an
// already-anonymized row until a full retention period elapses again, at
// which point re-applying the same placeholder values is a harmless no-op.

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/crm/consent"
	"github.com/kmuhub/kmuhub/internal/crm/contact"
	"github.com/kmuhub/kmuhub/internal/models"
)

// ContactRetentionHandler is the contacts entry in the retention registry.
type ContactRetentionHandler struct {
	repo        contact.Repository
	consentRepo consent.Repository
}

// NewContactRetentionHandler wires the handler. consentRepo supplies
// AnonymizeContact — the same mechanism the GDPR erasure-request path
// (consent.Service.ProcessDeletion) already uses, not a second one.
func NewContactRetentionHandler(repo contact.Repository, consentRepo consent.Repository) *ContactRetentionHandler {
	return &ContactRetentionHandler{repo: repo, consentRepo: consentRepo}
}

// ResourceType is the value a retention_policies row carries for contacts.
func (h *ContactRetentionHandler) ResourceType() string { return "contacts" }

// Table names the governed table for the admin view.
func (h *ContactRetentionHandler) Table() string { return "contacts" }

// DateColumn discloses what starts the retention clock. It is not a single
// SQL column: ListRetentionCandidates takes the later of updated_at and the
// contact's most recent activity, so a contact still being worked on is
// never retired just because it is old.
func (h *ContactRetentionHandler) DateColumn() string {
	return "updated_at (oder juengste activities.created_at, je spaeter)"
}

// SupportsAction reports that contacts can be both hard-deleted and
// anonymized — which one applies to a given record is decided in Plan.
func (h *ContactRetentionHandler) SupportsAction(action string) bool {
	return action == models.RetentionActionDelete || action == models.RetentionActionAnonymize
}

// Plan lists contacts past their retention cutoff. For action == delete, a
// contact blocked by IsInUse is moved to Skipped with the blocking reason
// instead of Due — the same information Service.Delete already surfaces as
// an error, here surfaced as a plannable outcome instead of a failure. For
// action == anonymize, IsInUse plays no role: AnonymizeContact does not
// touch the RESTRICT-guarded rows, so every candidate is Due.
func (h *ContactRetentionHandler) Plan(ctx context.Context, tenantID uuid.UUID, cutoff time.Time, action string) (*RetentionPlan, error) {
	candidates, err := h.repo.ListRetentionCandidates(ctx, tenantID, cutoff)
	if err != nil {
		return nil, fmt.Errorf("contact retention: list candidates: %w", err)
	}

	plan := &RetentionPlan{Due: make([]uuid.UUID, 0, len(candidates))}
	if action != models.RetentionActionDelete {
		plan.Due = candidates
		return plan, nil
	}

	for _, id := range candidates {
		inUse, reason, inUseErr := h.repo.IsInUse(ctx, id, tenantID)
		if inUseErr != nil {
			return nil, fmt.Errorf("contact retention: check in-use %s: %w", id, inUseErr)
		}
		if inUse {
			plan.Skipped = append(plan.Skipped, RetentionSkip{
				RecordID: id,
				Reason:   fmt.Sprintf("kann nicht geloescht werden: %s; Policy-Aktion auf anonymize stellen", reason),
			})
			continue
		}
		plan.Due = append(plan.Due, id)
	}
	return plan, nil
}

// Apply deletes or anonymizes the given contacts. It stops at the first
// error rather than continuing past it: an error here (as opposed to a
// planned RESTRICT skip) is unexpected, and the partial affected count is
// reported back to the engine, which records the item as failed.
func (h *ContactRetentionHandler) Apply(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID, action, _ string) (int, error) {
	affected := 0
	switch action {
	case models.RetentionActionDelete:
		for _, id := range ids {
			if err := h.repo.Delete(ctx, id, tenantID); err != nil {
				return affected, fmt.Errorf("contact retention: delete %s: %w", id, err)
			}
			affected++
		}
	case models.RetentionActionAnonymize:
		for _, id := range ids {
			if err := h.consentRepo.AnonymizeContact(ctx, id, tenantID); err != nil {
				return affected, fmt.Errorf("contact retention: anonymize %s: %w", id, err)
			}
			affected++
		}
	default:
		return 0, fmt.Errorf("contact retention: unsupported action %q", action)
	}
	return affected, nil
}
