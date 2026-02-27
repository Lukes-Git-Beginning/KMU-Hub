package lexware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

func (c *Client) CreateQuotation(ctx context.Context, tenantID uuid.UUID, quotation *LexwareQuotation) (*LexwareQuotation, error) {
	var result LexwareQuotation
	if err := c.do(ctx, tenantID, http.MethodPost, "v1/quotations", quotation, &result); err != nil {
		return nil, fmt.Errorf("lexware: create quotation: %w", err)
	}
	return &result, nil
}

func (c *Client) UpdateQuotation(ctx context.Context, tenantID uuid.UUID, lexwareID string, quotation *LexwareQuotation) (*LexwareQuotation, error) {
	var result LexwareQuotation
	if err := c.do(ctx, tenantID, http.MethodPut, fmt.Sprintf("v1/quotations/%s", lexwareID), quotation, &result); err != nil {
		return nil, fmt.Errorf("lexware: update quotation %s: %w", lexwareID, err)
	}
	return &result, nil
}

func (c *Client) GetQuotation(ctx context.Context, tenantID uuid.UUID, lexwareID string) (*LexwareQuotation, error) {
	var result LexwareQuotation
	if err := c.do(ctx, tenantID, http.MethodGet, fmt.Sprintf("v1/quotations/%s", lexwareID), nil, &result); err != nil {
		return nil, fmt.Errorf("lexware: get quotation %s: %w", lexwareID, err)
	}
	return &result, nil
}
