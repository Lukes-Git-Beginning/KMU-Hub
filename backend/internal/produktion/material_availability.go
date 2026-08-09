package produktion

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// MaterialAvailabilityLine is the per-BOM-position availability result: how
// much is required for the order, how much is on hand, and the shortfall.
// AvailableQuantity and ShortfallQuantity are nil when the position could
// not be matched against inventar -- unknown availability, not a zero stock
// claim.
type MaterialAvailabilityLine struct {
	MaterialName      string
	Unit              string
	RequiredQuantity  float64
	AvailableQuantity *float64
	ShortfallQuantity *float64
}

// MaterialAvailability is the material availability check result for one
// production order against its linked BOM.
type MaterialAvailability struct {
	OrderID uuid.UUID
	BomID   uuid.UUID
	Lines   []MaterialAvailabilityLine
}

// GetMaterialAvailability checks a production order's BOM positions against
// real inventar stock. The order must have a linked BOM (ErrOrderHasNoBOM
// otherwise). A BOM position whose material_name has no matching inventar
// item -- or when no InventarLookup is configured, or the lookup call itself
// fails -- comes back with AvailableQuantity/ShortfallQuantity unset rather
// than as an error: an unresolvable position is a data gap, not a reason to
// fail the whole check (architecture rule 8, graceful degradation).
func (s *Service) GetMaterialAvailability(ctx context.Context, tenantID, orderID uuid.UUID) (*MaterialAvailability, error) {
	order, err := s.repo.GetOrder(ctx, tenantID, orderID)
	if err != nil {
		return nil, err
	}
	if order.BomID == nil {
		return nil, ErrOrderHasNoBOM
	}

	bom, err := s.repo.GetBOM(ctx, tenantID, *order.BomID)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(bom.Items))
	for _, item := range bom.Items {
		names = append(names, item.MaterialName)
	}

	var stock map[string]InventarStock
	if s.inventarLookup != nil && len(names) > 0 {
		resolved, lookupErr := s.inventarLookup.ResolveByNames(ctx, tenantID, names)
		if lookupErr != nil {
			slog.WarnContext(ctx, "produktion: inventar stock lookup failed, availability unknown for all positions",
				"order_id", orderID, "tenant_id", tenantID, "bom_id", *order.BomID, "error", lookupErr,
			)
		} else {
			stock = resolved
		}
	}

	lines := make([]MaterialAvailabilityLine, 0, len(bom.Items))
	for _, item := range bom.Items {
		line := MaterialAvailabilityLine{
			MaterialName:     item.MaterialName,
			Unit:             item.Unit,
			RequiredQuantity: item.Quantity * float64(order.Quantity),
		}
		if info, ok := stock[normalizeMaterialName(item.MaterialName)]; ok {
			available := info.Quantity
			line.AvailableQuantity = &available

			shortfall := line.RequiredQuantity - available
			if shortfall < 0 {
				shortfall = 0
			}
			line.ShortfallQuantity = &shortfall
		}
		lines = append(lines, line)
	}

	return &MaterialAvailability{
		OrderID: orderID,
		BomID:   *order.BomID,
		Lines:   lines,
	}, nil
}
