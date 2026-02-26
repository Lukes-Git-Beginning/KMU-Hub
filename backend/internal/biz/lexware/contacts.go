package lexware

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/uuid"
)

func (c *Client) ListContacts(ctx context.Context, tenantID uuid.UUID, page, size int) (*LexwareListResponse[LexwareContact], error) {
	params := url.Values{}
	params.Set("page", fmt.Sprintf("%d", page))
	params.Set("size", fmt.Sprintf("%d", size))

	var result LexwareListResponse[LexwareContact]
	if err := c.doList(ctx, tenantID, "v1/contacts", params, &result); err != nil {
		return nil, fmt.Errorf("lexware: list contacts: %w", err)
	}
	return &result, nil
}

func (c *Client) GetContact(ctx context.Context, tenantID uuid.UUID, lexwareID string) (*LexwareContact, error) {
	var contact LexwareContact
	if err := c.do(ctx, tenantID, http.MethodGet, fmt.Sprintf("v1/contacts/%s", lexwareID), nil, &contact); err != nil {
		return nil, fmt.Errorf("lexware: get contact %s: %w", lexwareID, err)
	}
	return &contact, nil
}

func (c *Client) CreateContact(ctx context.Context, tenantID uuid.UUID, contact *LexwareContact) (*LexwareContact, error) {
	var result LexwareContact
	if err := c.do(ctx, tenantID, http.MethodPost, "v1/contacts", contact, &result); err != nil {
		return nil, fmt.Errorf("lexware: create contact: %w", err)
	}
	return &result, nil
}

func (c *Client) UpdateContact(ctx context.Context, tenantID uuid.UUID, lexwareID string, contact *LexwareContact) (*LexwareContact, error) {
	var result LexwareContact
	if err := c.do(ctx, tenantID, http.MethodPut, fmt.Sprintf("v1/contacts/%s", lexwareID), contact, &result); err != nil {
		return nil, fmt.Errorf("lexware: update contact %s: %w", lexwareID, err)
	}
	return &result, nil
}
