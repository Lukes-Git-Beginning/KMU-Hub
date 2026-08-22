package gateway

// Open items (Offene Posten) — the receivables list with its aging summary.

import (
	"net/http"

	"github.com/kmuhub/kmuhub/internal/server/response"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// HandleListOpenItems serves the tenant's unpaid receivables, most overdue
// first, with the tenant-wide aging summary.
// GET /api/v1/finance/open-items?bucket=&overdue_only=&page=&page_size=
func (b *BizRoutes) HandleListOpenItems(w http.ResponseWriter, r *http.Request) {
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
	page, pageSize := parsePagination(r, 1, 50)

	resp, err := client.ListOpenItems(r.Context(), &bizv1.ListOpenItemsRequest{
		TenantId:    tenantID,
		Bucket:      r.URL.Query().Get("bucket"),
		OverdueOnly: r.URL.Query().Get("overdue_only") == "true",
		Page:        int32(page),
		PerPage:     int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	items, err := hrMarshalSlice(resp.GetItems())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	summary := resp.GetSummary()
	totals, err := hrMarshalSlice(summary.GetTotals())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	buckets, err := hrMarshalSlice(summary.GetBuckets())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"items": items,
		"total": resp.GetTotal(),
		"summary": map[string]any{
			"totals":  totals,
			"buckets": buckets,
		},
	})
}
