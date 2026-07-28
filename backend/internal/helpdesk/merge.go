package helpdesk

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DetectDuplicates returns open/pending tickets from the same requester that
// share a common subject prefix with the given ticket. Duplicate detection
// uses ILIKE prefix matching (see FindOpenTicketsByRequester). If pg_trgm is
// enabled in the DB, replace with a similarity query for better fuzzy results.
func DetectDuplicates(ctx context.Context, repo Repository, tenantID uuid.UUID, ticket *Ticket) ([]*Ticket, error) {
	prefix := subjectPrefix(ticket.Subject, 5)
	if prefix == "" {
		return nil, nil
	}

	candidates, err := repo.FindOpenTicketsByRequester(ctx, tenantID, ticket.RequesterID, prefix)
	if err != nil {
		return nil, fmt.Errorf("detect duplicates – query: %w", err)
	}

	// Exclude the ticket itself from the results.
	result := candidates[:0]
	for _, c := range candidates {
		if c.ID != ticket.ID {
			result = append(result, c)
		}
	}
	return result, nil
}

// MergeTickets merges sourceID into targetID:
//  1. Validates that source != target and source is not already merged.
//  2. Reassigns all messages from source to target.
//  3. Sets source.merged_into_id = targetID and status = 'merged'.
func MergeTickets(ctx context.Context, repo Repository, tenantID, sourceID, targetID uuid.UUID) error {
	if sourceID == targetID {
		return ErrCannotMergeSelf
	}

	source, err := repo.GetTicketByID(ctx, sourceID, tenantID)
	if err != nil {
		return fmt.Errorf("merge tickets – load source: %w", err)
	}
	if source.Status == TicketStatusMerged {
		return ErrAlreadyMerged
	}

	// Verify target exists.
	if _, err := repo.GetTicketByID(ctx, targetID, tenantID); err != nil {
		return fmt.Errorf("merge tickets – load target: %w", err)
	}

	// Set source fields before the atomic TX so the UPDATE SQL inside sees
	// the correct values.
	now := time.Now().UTC()
	source.Status = TicketStatusMerged
	source.MergedIntoID = &targetID
	source.UpdatedAt = now

	// Atomically reassign messages and close source in one transaction.
	if err := repo.MergeTicketTx(ctx, source, targetID); err != nil {
		return fmt.Errorf("merge tickets – atomic tx: %w", err)
	}

	return nil
}

// subjectPrefix returns the first n words of a subject string (lowercased),
// suitable for ILIKE prefix matching.
func subjectPrefix(subject string, words int) string {
	s := strings.TrimSpace(subject)
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return ""
	}
	if len(parts) < words {
		words = len(parts)
	}
	return strings.Join(parts[:words], " ")
}
