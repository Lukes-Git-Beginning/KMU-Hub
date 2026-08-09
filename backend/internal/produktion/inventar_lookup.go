package produktion

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	inventarv1 "github.com/kmuhub/kmuhub/proto/inventar/v1"
)

// InventarStock is the read-only stock snapshot for a single inventar item,
// as needed by the material availability check.
type InventarStock struct {
	Quantity float64
	Unit     string
}

// InventarLookup is a narrow interface that the produktion service uses to
// check real stock levels for BOM material positions. production_bom_items
// carries no SKU (only a free-text material_name), so resolution matches on
// the item name -- the only identity BOM items and inventar items currently
// share.
type InventarLookup interface {
	// ResolveByNames resolves inventar items whose name case-insensitively
	// matches one of the given material names, in a single batch call.
	// Names with no match are simply absent from the returned map. The map
	// key is the normalized (trimmed, lower-cased) material name.
	ResolveByNames(ctx context.Context, tenantID uuid.UUID, materialNames []string) (map[string]InventarStock, error)
}

// normalizeMaterialName is the shared key used to match a BOM material_name
// against an inventar item name.
func normalizeMaterialName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// GRPCInventarLookup implements InventarLookup by calling the inventar gRPC
// service.
type GRPCInventarLookup struct {
	client inventarv1.InventarServiceClient
}

// NewGRPCInventarLookup creates a new GRPCInventarLookup.
func NewGRPCInventarLookup(client inventarv1.InventarServiceClient) *GRPCInventarLookup {
	return &GRPCInventarLookup{client: client}
}

// ResolveByNames fetches the tenant's inventar catalog in one ListItems call
// (capped at the service's own page-size ceiling of 200, same constraint
// every other inventar consumer in this repo lives with) and matches it
// against the requested material names client-side. A material with no
// match is simply absent from the result -- unknown availability, not an
// error.
func (l *GRPCInventarLookup) ResolveByNames(ctx context.Context, tenantID uuid.UUID, materialNames []string) (map[string]InventarStock, error) {
	if len(materialNames) == 0 {
		return map[string]InventarStock{}, nil
	}

	wanted := make(map[string]bool, len(materialNames))
	for _, name := range materialNames {
		if norm := normalizeMaterialName(name); norm != "" {
			wanted[norm] = true
		}
	}

	resp, err := l.client.ListItems(ctx, &inventarv1.ListItemsRequest{
		TenantId: tenantID.String(),
		PageSize: 200,
	})
	if err != nil {
		return nil, fmt.Errorf("list inventar items: %w", err)
	}

	result := make(map[string]InventarStock, len(wanted))
	for _, item := range resp.GetItems() {
		key := normalizeMaterialName(item.GetName())
		if wanted[key] {
			result[key] = InventarStock{
				Quantity: float64(item.GetQuantity()),
				Unit:     item.GetUnit(),
			}
		}
	}
	return result, nil
}
