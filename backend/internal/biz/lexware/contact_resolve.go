package lexware

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// resolveContactLexwareIDByEmail resolves the Lexware contact ID for a given
// customer email address.
//
// Resolution order (mirrors bexio.resolveContactBexioIDByEmail):
//  1. If a ContactService is wired: look up the local contact by email, then
//     fetch the Lexware entity mapping for that exact contact UUID → unambiguous.
//  2. Fallback (ContactService nil): the mapping table carries no email column,
//     so a lookup is only safe when exactly one contact mapping exists. With
//     more than one we must refuse rather than guess — pushing an invoice to
//     the wrong Lexware customer is silent, billable data corruption at the
//     customer's end.
//
// An empty email always returns an error — the document has no addressee.
func resolveContactLexwareIDByEmail(
	ctx context.Context,
	repo Repository,
	contacts ContactService,
	configID uuid.UUID,
	email string,
) (string, error) {
	if email == "" {
		return "", fmt.Errorf("lexware contact resolve: customer email is empty")
	}

	// --- Path 1: ContactService is wired (preferred) ---
	if contacts != nil {
		local, err := contacts.GetByEmail(ctx, email)
		if err != nil {
			return "", fmt.Errorf("lexware contact resolve: lookup local contact by email %q: %w", email, err)
		}
		if local == nil {
			return "", fmt.Errorf("lexware contact resolve: no local contact found for email %q", email)
		}

		mapping, err := repo.GetEntityMapping(ctx, configID, "contact", local.ID)
		if err != nil {
			return "", fmt.Errorf("lexware contact resolve: no lexware mapping for contact %s (email %q): %w", local.ID, email, err)
		}
		if mapping == nil {
			return "", fmt.Errorf("lexware contact resolve: no lexware mapping for contact %s (email %q)", local.ID, email)
		}
		return mapping.LexwareID, nil
	}

	// --- Path 2: Fallback — ContactService not wired ---
	mappings, err := repo.ListEntityMappings(ctx, configID, "contact")
	if err != nil {
		return "", fmt.Errorf("lexware contact resolve: list contact mappings: %w", err)
	}
	switch len(mappings) {
	case 0:
		return "", fmt.Errorf("lexware contact resolve: no lexware contact mapping found for email %q (contact service not wired)", email)
	case 1:
		return mappings[0].LexwareID, nil
	default:
		return "", fmt.Errorf("lexware contact resolve: %d contact mappings exist but contact service is not wired — cannot determine which belongs to email %q; wire the CRM contact service to enable exact lookup", len(mappings), email)
	}
}
