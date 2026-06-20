package einkauf

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	inventarv1 "github.com/kmuhub/kmuhub/proto/inventar/v1"
)

// GRPCInventarAdjuster implements InventarAdjuster by calling the inventar
// gRPC service. It resolves the inventar item by SKU using ListItems(search=sku)
// and then calls AdjustStock. If no item matches the SKU the call is a no-op.
type GRPCInventarAdjuster struct {
	client inventarv1.InventarServiceClient
}

// NewGRPCInventarAdjuster creates a new GRPCInventarAdjuster.
func NewGRPCInventarAdjuster(client inventarv1.InventarServiceClient) *GRPCInventarAdjuster {
	return &GRPCInventarAdjuster{client: client}
}

// AdjustStockBySKU finds the inventar item with the given SKU and increments
// its quantity by delta. Returns nil (soft-fail) if no item has that SKU.
func (a *GRPCInventarAdjuster) AdjustStockBySKU(
	ctx context.Context,
	tenantID uuid.UUID,
	sku string,
	delta int64,
	reason string,
) error {
	if sku == "" || delta == 0 {
		return nil
	}

	// Resolve item ID by SKU. We use the search field and page_size=1 to
	// keep overhead minimal; exact SKU match is filtered client-side.
	listResp, err := a.client.ListItems(ctx, &inventarv1.ListItemsRequest{
		TenantId: tenantID.String(),
		Search:   sku,
		PageSize: 10,
	})
	if err != nil {
		return fmt.Errorf("list inventar items by sku %q: %w", sku, err)
	}

	var itemID string
	for _, item := range listResp.GetItems() {
		if item.GetSku() == sku {
			itemID = item.GetId()
			break
		}
	}
	if itemID == "" {
		// SKU not yet tracked in inventar — log and skip (graceful degradation).
		slog.Debug("inventar item not found for SKU, skipping stock adjustment",
			"sku", sku, "tenant_id", tenantID)
		return nil
	}

	_, err = a.client.AdjustStock(ctx, &inventarv1.AdjustStockRequest{
		TenantId: tenantID.String(),
		ItemId:   itemID,
		Delta:    delta,
		Reason:   reason,
	})
	if err != nil {
		return fmt.Errorf("adjust stock for sku %q item %s: %w", sku, itemID, err)
	}

	slog.Info("inventar stock adjusted via einkauf goods receipt",
		"sku", sku, "item_id", itemID, "delta", delta, "tenant_id", tenantID)
	return nil
}
