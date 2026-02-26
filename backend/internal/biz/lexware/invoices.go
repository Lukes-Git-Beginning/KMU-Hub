package lexware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

func (c *Client) CreateInvoice(ctx context.Context, tenantID uuid.UUID, invoice *LexwareInvoice) (*LexwareInvoice, error) {
	var result LexwareInvoice
	if err := c.do(ctx, tenantID, http.MethodPost, "v1/invoices", invoice, &result); err != nil {
		return nil, fmt.Errorf("lexware: create invoice: %w", err)
	}
	return &result, nil
}

func (c *Client) UpdateInvoice(ctx context.Context, tenantID uuid.UUID, lexwareID string, invoice *LexwareInvoice) (*LexwareInvoice, error) {
	var result LexwareInvoice
	if err := c.do(ctx, tenantID, http.MethodPut, fmt.Sprintf("v1/invoices/%s", lexwareID), invoice, &result); err != nil {
		return nil, fmt.Errorf("lexware: update invoice %s: %w", lexwareID, err)
	}
	return &result, nil
}

func (c *Client) GetInvoice(ctx context.Context, tenantID uuid.UUID, lexwareID string) (*LexwareInvoice, error) {
	var result LexwareInvoice
	if err := c.do(ctx, tenantID, http.MethodGet, fmt.Sprintf("v1/invoices/%s", lexwareID), nil, &result); err != nil {
		return nil, fmt.Errorf("lexware: get invoice %s: %w", lexwareID, err)
	}
	return &result, nil
}

func (c *Client) FinalizeInvoice(ctx context.Context, tenantID uuid.UUID, lexwareID string) error {
	if err := c.do(ctx, tenantID, http.MethodPost, fmt.Sprintf("v1/invoices/%s/actions/finalize", lexwareID), nil, nil); err != nil {
		return fmt.Errorf("lexware: finalize invoice %s: %w", lexwareID, err)
	}
	return nil
}
