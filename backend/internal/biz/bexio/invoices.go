package bexio

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

// CreateInvoice creates a new invoice in Bexio.
func (c *Client) CreateInvoice(ctx context.Context, tenantID uuid.UUID, invoice *BexioInvoice) (*BexioInvoice, error) {
	var result BexioInvoice
	if err := c.do(ctx, tenantID, http.MethodPost, "kb_invoice", invoice, &result); err != nil {
		return nil, fmt.Errorf("bexio: create invoice: %w", err)
	}
	return &result, nil
}

// UpdateInvoice updates an existing invoice in Bexio.
// Note: Bexio uses POST for updates, not PATCH.
func (c *Client) UpdateInvoice(ctx context.Context, tenantID uuid.UUID, bexioID int, invoice *BexioInvoice) (*BexioInvoice, error) {
	var result BexioInvoice
	if err := c.do(ctx, tenantID, http.MethodPost, fmt.Sprintf("kb_invoice/%d", bexioID), invoice, &result); err != nil {
		return nil, fmt.Errorf("bexio: update invoice %d: %w", bexioID, err)
	}
	return &result, nil
}

// GetInvoice retrieves a single invoice by its Bexio ID.
func (c *Client) GetInvoice(ctx context.Context, tenantID uuid.UUID, bexioID int) (*BexioInvoice, error) {
	var result BexioInvoice
	if err := c.do(ctx, tenantID, http.MethodGet, fmt.Sprintf("kb_invoice/%d", bexioID), nil, &result); err != nil {
		return nil, fmt.Errorf("bexio: get invoice %d: %w", bexioID, err)
	}
	return &result, nil
}

// ListInvoicePayments retrieves payment information for a specific invoice.
func (c *Client) ListInvoicePayments(ctx context.Context, tenantID uuid.UUID, bexioID int) ([]BexioPaymentStatus, error) {
	var payments []BexioPaymentStatus
	if err := c.do(ctx, tenantID, http.MethodGet, fmt.Sprintf("kb_invoice/%d/payment", bexioID), nil, &payments); err != nil {
		return nil, fmt.Errorf("bexio: list invoice %d payments: %w", bexioID, err)
	}
	return payments, nil
}
