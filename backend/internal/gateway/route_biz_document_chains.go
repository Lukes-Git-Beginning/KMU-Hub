package gateway

// Document chains (Belegkette): the lifecycle of a customer document from
// quote through invoice to payment, dunning, or credit note — a tenant-wide
// overview page, not filtered by any single document.

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/server/response"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// chainNodeWire is the exact shape the desktop client reads (ChainNode in
// types/finance-types.ts, consumed through BelegketteTab). Hand-mapped for
// the same reason as bank accounts/transactions next door: camelCase where
// the repo-wide marshaller emits snake_case, and a pre-formatted display
// amount ("CHF 12.450,00") where the client renders the string verbatim
// without parsing it.
type chainNodeWire struct {
	Type   string `json:"type"`
	Number string `json:"number"`
	Date   string `json:"date"`
	Amount string `json:"amount"`
	Status string `json:"status"`
}

type documentChainWire struct {
	ID         string          `json:"id"`
	Customer   string          `json:"customer"`
	TotalValue string          `json:"totalValue"`
	IsComplete bool            `json:"isComplete"`
	Nodes      []chainNodeWire `json:"nodes"`
}

func toDocumentChainWire(c *bizv1.DocumentChain) documentChainWire {
	nodes := make([]chainNodeWire, 0, len(c.GetNodes()))
	for _, n := range c.GetNodes() {
		number, date := n.GetNumber(), n.GetDate()
		if number == "" {
			number = "—"
		}
		if date == "" {
			date = "—"
		}
		nodes = append(nodes, chainNodeWire{
			Type:   n.GetType(),
			Number: number,
			Date:   date,
			Amount: formatChainAmount(n.GetAmount(), c.GetCurrency()),
			Status: n.GetStatus(),
		})
	}
	return documentChainWire{
		ID:         c.GetId(),
		Customer:   c.GetCustomer(),
		TotalValue: formatChainAmount(c.GetTotalValue(), c.GetCurrency()),
		IsComplete: c.GetIsComplete(),
		Nodes:      nodes,
	}
}

// formatChainAmount renders a decimal amount string in the grouping the
// desktop client displays verbatim: "." between thousands, "," before the two
// decimals, prefixed with the ISO currency code — "CHF 12.450,00". An
// unparsable amount would be a bug upstream; it renders as "0,00" rather than
// propagating a broken string into the UI.
func formatChainAmount(raw, currency string) string {
	amount, err := decimal.NewFromString(raw)
	if err != nil {
		amount = decimal.Zero
	}
	rounded := amount.Round(2)
	negative := rounded.IsNegative()
	if negative {
		rounded = rounded.Neg()
	}
	whole := rounded.IntPart()
	frac := rounded.Sub(decimal.NewFromInt(whole)).Mul(decimal.NewFromInt(100)).Round(0).IntPart()

	sign := ""
	if negative {
		sign = "-"
	}
	return fmt.Sprintf("%s %s%s,%02d", currency, sign, groupThousands(strconv.FormatInt(whole, 10)), frac)
}

// groupThousands inserts "." every three digits from the right, the DACH
// grouping convention ("12.450" not "12,450").
func groupThousands(digits string) string {
	if len(digits) <= 3 {
		return digits
	}
	var b strings.Builder
	first := len(digits) % 3
	if first == 0 {
		first = 3
	}
	b.WriteString(digits[:first])
	for i := first; i < len(digits); i += 3 {
		b.WriteByte('.')
		b.WriteString(digits[i : i+3])
	}
	return b.String()
}

// HandleListDocumentChains serves the tenant's document chains.
// GET /api/v1/finance/document-chains
func (b *BizRoutes) HandleListDocumentChains(w http.ResponseWriter, r *http.Request) {
	client, err := b.getBizClient()
	if err != nil {
		respondServiceUnavailable(w, b.ServiceName())
		return
	}
	tenantID, err := getTenantID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	resp, err := client.ListDocumentChains(r.Context(), &bizv1.ListDocumentChainsRequest{TenantId: tenantID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	chains := make([]documentChainWire, 0, len(resp.GetChains()))
	for _, c := range resp.GetChains() {
		chains = append(chains, toDocumentChainWire(c))
	}
	response.JSON(w, http.StatusOK, map[string]any{"chains": chains})
}
