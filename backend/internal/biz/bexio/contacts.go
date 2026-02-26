package bexio

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
)

// ListContacts retrieves contacts from Bexio. If updatedSince is provided,
// only contacts modified after that time are returned (delta sync).
func (c *Client) ListContacts(ctx context.Context, tenantID uuid.UUID, updatedSince *time.Time) ([]BexioContact, error) {
	params := url.Values{}
	if updatedSince != nil {
		params.Set("updated_from", updatedSince.Format("2006-01-02T15:04:05"))
	}

	var contacts []BexioContact
	if err := c.doList(ctx, tenantID, "contact", params, &contacts); err != nil {
		return nil, fmt.Errorf("bexio: list contacts: %w", err)
	}
	return contacts, nil
}

// GetContact retrieves a single contact by its Bexio ID.
func (c *Client) GetContact(ctx context.Context, tenantID uuid.UUID, bexioID int) (*BexioContact, error) {
	var contact BexioContact
	if err := c.do(ctx, tenantID, http.MethodGet, fmt.Sprintf("contact/%d", bexioID), nil, &contact); err != nil {
		return nil, fmt.Errorf("bexio: get contact %d: %w", bexioID, err)
	}
	return &contact, nil
}

// CreateContact creates a new contact in Bexio.
func (c *Client) CreateContact(ctx context.Context, tenantID uuid.UUID, contact *BexioContact) (*BexioContact, error) {
	var result BexioContact
	if err := c.do(ctx, tenantID, http.MethodPost, "contact", contact, &result); err != nil {
		return nil, fmt.Errorf("bexio: create contact: %w", err)
	}
	return &result, nil
}

// UpdateContact updates an existing contact in Bexio.
// Note: Bexio uses POST for updates, not PATCH.
func (c *Client) UpdateContact(ctx context.Context, tenantID uuid.UUID, bexioID int, contact *BexioContact) (*BexioContact, error) {
	var result BexioContact
	if err := c.do(ctx, tenantID, http.MethodPost, fmt.Sprintf("contact/%d", bexioID), contact, &result); err != nil {
		return nil, fmt.Errorf("bexio: update contact %d: %w", bexioID, err)
	}
	return &result, nil
}
