package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/biz/bexio"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// crmContactAdapter implements bexio.ContactService (and lexware.ContactService)
// by calling the CRM gRPC service.  It is wired into the Bexio integration when
// CRM_GRPC_ADDRESS is set so that resolveContactBexioIDByEmail can look up a
// local contact UUID by email and then fetch the exact Bexio entity mapping,
// rather than guessing from an unordered mapping list.
type crmContactAdapter struct {
	client crmv1.CRMServiceClient
}

// Verify at compile time that crmContactAdapter satisfies the bexio.ContactService interface.
var _ bexio.ContactService = (*crmContactAdapter)(nil)

// GetByID fetches a contact by its local UUID.
func (a *crmContactAdapter) GetByID(ctx context.Context, id uuid.UUID) (*bexio.ContactResult, error) {
	resp, err := a.client.GetContact(ctx, &crmv1.GetContactRequest{Id: id.String()})
	if err != nil {
		return nil, fmt.Errorf("crm contact adapter GetByID: %w", err)
	}
	if resp.GetContact() == nil {
		return nil, fmt.Errorf("crm contact adapter GetByID: contact %s not found", id)
	}
	return toContactResult(resp.GetContact()), nil
}

// GetByEmail looks up a contact by email address using the CRM search path.
// It issues a ListContacts request with Search=email and PageSize=1, then
// confirms the returned email matches exactly (case-insensitive).
func (a *crmContactAdapter) GetByEmail(ctx context.Context, email string) (*bexio.ContactResult, error) {
	resp, err := a.client.ListContacts(ctx, &crmv1.ListContactsRequest{
		Search:   email,
		PageSize: 5,
		// IsAdmin=true so we see all contacts regardless of visibility.
		IsAdmin: true,
	})
	if err != nil {
		return nil, fmt.Errorf("crm contact adapter GetByEmail: %w", err)
	}

	// The CRM search is a free-text search; filter for an exact email match.
	for _, c := range resp.GetContacts() {
		if equalEmailCI(c.GetEmail(), email) {
			return toContactResult(c), nil
		}
	}

	return nil, fmt.Errorf("crm contact adapter GetByEmail: no contact found with email %q", email)
}

// CreateForSync creates a new contact in the CRM during an inbound Bexio sync.
func (a *crmContactAdapter) CreateForSync(ctx context.Context, data *bexio.ContactSyncData, createdBy uuid.UUID) (uuid.UUID, error) {
	email := ""
	if data.Email != nil {
		email = *data.Email
	}
	phone := ""
	if data.Phone != nil {
		phone = *data.Phone
	}
	notes := ""
	if data.Notes != nil {
		notes = *data.Notes
	}

	req := &crmv1.CreateContactRequest{
		FirstName: data.FirstName,
		LastName:  data.LastName,
		Email:     email,
		Phone:     phone,
		Notes:     notes,
		CreatedBy: createdBy.String(),
	}
	resp, err := a.client.CreateContact(ctx, req)
	if err != nil {
		return uuid.Nil, fmt.Errorf("crm contact adapter CreateForSync: %w", err)
	}
	id, err := uuid.Parse(resp.GetContact().GetId())
	if err != nil {
		return uuid.Nil, fmt.Errorf("crm contact adapter CreateForSync: parse returned id: %w", err)
	}
	return id, nil
}

// UpdateForSync updates an existing CRM contact with data from Bexio.
func (a *crmContactAdapter) UpdateForSync(ctx context.Context, id uuid.UUID, data *bexio.ContactSyncData) error {
	req := &crmv1.UpdateContactRequest{Id: id.String()}
	if data.Email != nil {
		req.Email = data.Email
	}
	if data.Phone != nil {
		req.Phone = data.Phone
	}
	if data.Notes != nil {
		req.Notes = data.Notes
	}
	// First/last name are always present in sync data.
	req.FirstName = &data.FirstName
	req.LastName = &data.LastName

	_, err := a.client.UpdateContact(ctx, req)
	if err != nil {
		return fmt.Errorf("crm contact adapter UpdateForSync: %w", err)
	}
	return nil
}

// ListModifiedSince returns all contacts whose updated_at timestamp is after
// the given time.  The CRM proto has no updated_after filter, so we page
// through all contacts and filter client-side.  This is acceptable for the
// current single-tenant scale; add a proto field when contact counts grow large.
func (a *crmContactAdapter) ListModifiedSince(ctx context.Context, since time.Time) ([]bexio.ContactResult, error) {
	const pageSize = 500

	var results []bexio.ContactResult
	page := int32(1)

	for {
		resp, err := a.client.ListContacts(ctx, &crmv1.ListContactsRequest{
			Page:     page,
			PageSize: pageSize,
			IsAdmin:  true,
		})
		if err != nil {
			return nil, fmt.Errorf("crm contact adapter ListModifiedSince: page %d: %w", page, err)
		}

		contacts := resp.GetContacts()
		for _, c := range contacts {
			updatedAt, parseErr := time.Parse(time.RFC3339, c.GetUpdatedAt())
			if parseErr != nil {
				slog.Warn("crm contact adapter: could not parse updated_at, skipping",
					"contact_id", c.GetId(),
					"updated_at", c.GetUpdatedAt(),
					"error", parseErr,
				)
				continue
			}
			if updatedAt.After(since) {
				results = append(results, *toContactResult(c))
			}
		}

		// Stop when we receive fewer items than a full page.
		if int32(len(contacts)) < pageSize {
			break
		}
		page++
	}

	return results, nil
}

// toContactResult converts a CRM proto ContactInfo to bexio.ContactResult.
func toContactResult(c *crmv1.ContactInfo) *bexio.ContactResult {
	r := &bexio.ContactResult{
		FirstName: c.GetFirstName(),
		LastName:  c.GetLastName(),
	}

	if id, err := uuid.Parse(c.GetId()); err == nil {
		r.ID = id
	}

	if email := c.GetEmail(); email != "" {
		r.Email = &email
	}
	if phone := c.GetPhone(); phone != "" {
		r.Phone = &phone
	}
	if company := c.GetCompanyName(); company != "" {
		r.CompanyName = &company
	}
	if notes := c.GetNotes(); notes != "" {
		r.Notes = &notes
	}

	if t, err := time.Parse(time.RFC3339, c.GetUpdatedAt()); err == nil {
		r.UpdatedAt = t
	}

	return r
}

// equalEmailCI compares two email addresses case-insensitively.
func equalEmailCI(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ca, cb := a[i], b[i]
		// ASCII lower-case normalisation (email local-part is case-insensitive
		// by convention even though RFC 5321 says the domain part is).
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
