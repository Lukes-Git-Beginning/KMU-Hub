package bexio

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// resolveContactBexioIDByEmail resolves the Bexio contact ID for a given customer
// email address.
//
// Resolution order:
//  1. If a ContactService is wired: look up the local contact by email, then
//     fetch the Bexio entity mapping for that exact contact UUID → unambiguous.
//  2. Fallback (ContactService nil): scan all contact entity mappings and find the
//     one whose KmuhubID matches a prior sync for this email.  Because the
//     mapping table carries no email column this fallback is inherently
//     ambiguous; it returns an error when more than one mapping exists.
//
// An empty email always returns an error — the document has no addressee.
func resolveContactBexioIDByEmail(
	ctx context.Context,
	repo Repository,
	contacts ContactService,
	configID uuid.UUID,
	email string,
) (int, error) {
	if email == "" {
		return 0, fmt.Errorf("bexio contact resolve: customer email is empty")
	}

	// --- Path 1: ContactService is wired (preferred) ---
	if contacts != nil {
		local, err := contacts.GetByEmail(ctx, email)
		if err != nil {
			return 0, fmt.Errorf("bexio contact resolve: lookup local contact by email %q: %w", email, err)
		}
		if local == nil {
			return 0, fmt.Errorf("bexio contact resolve: no local contact found for email %q", email)
		}

		mapping, err := repo.GetEntityMapping(ctx, configID, "contact", local.ID)
		if err != nil {
			return 0, fmt.Errorf("bexio contact resolve: no bexio mapping for contact %s (email %q): %w", local.ID, email, err)
		}
		return mapping.BexioID, nil
	}

	// --- Path 2: Fallback — ContactService not wired ---
	// We have no way to filter mappings by email; if there is exactly one
	// contact mapping we can use it, otherwise we must refuse to guess.
	mappings, err := repo.ListEntityMappings(ctx, configID, "contact")
	if err != nil {
		return 0, fmt.Errorf("bexio contact resolve: list contact mappings: %w", err)
	}
	switch len(mappings) {
	case 0:
		return 0, fmt.Errorf("bexio contact resolve: no bexio contact mapping found for email %q (contact service not wired)", email)
	case 1:
		return mappings[0].BexioID, nil
	default:
		return 0, fmt.Errorf("bexio contact resolve: %d contact mappings exist but contact service is not wired — cannot determine which belongs to email %q; wire the CRM contact service to enable exact lookup", len(mappings), email)
	}
}
