package bexio

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

// CreateQuote creates a new quote (KB offer) in Bexio.
func (c *Client) CreateQuote(ctx context.Context, tenantID uuid.UUID, quote *BexioQuote) (*BexioQuote, error) {
	var result BexioQuote
	if err := c.do(ctx, tenantID, http.MethodPost, "kb_offer", quote, &result); err != nil {
		return nil, fmt.Errorf("bexio: create quote: %w", err)
	}
	return &result, nil
}

// UpdateQuote updates an existing quote in Bexio.
// Note: Bexio uses POST for updates, not PATCH.
func (c *Client) UpdateQuote(ctx context.Context, tenantID uuid.UUID, bexioID int, quote *BexioQuote) (*BexioQuote, error) {
	var result BexioQuote
	if err := c.do(ctx, tenantID, http.MethodPost, fmt.Sprintf("kb_offer/%d", bexioID), quote, &result); err != nil {
		return nil, fmt.Errorf("bexio: update quote %d: %w", bexioID, err)
	}
	return &result, nil
}

// GetQuote retrieves a single quote by its Bexio ID.
func (c *Client) GetQuote(ctx context.Context, tenantID uuid.UUID, bexioID int) (*BexioQuote, error) {
	var result BexioQuote
	if err := c.do(ctx, tenantID, http.MethodGet, fmt.Sprintf("kb_offer/%d", bexioID), nil, &result); err != nil {
		return nil, fmt.Errorf("bexio: get quote %d: %w", bexioID, err)
	}
	return &result, nil
}
