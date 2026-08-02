package gateway

// Time entries (Stunden -> Rechnung): billable time entries for invoicing.
// The data is owned by Work (time_entries is a Work table joined to
// tasks/projects), not Biz -- this handler calls the Work gRPC client
// directly even though the route lives under /finance for the desktop client.

import (
	"net/http"

	"github.com/kmuhub/kmuhub/internal/server/response"
	workv1 "github.com/kmuhub/kmuhub/proto/work/v1"
)

// getWorkClient lazily obtains a gRPC client for the Work service.
func (b *BizRoutes) getWorkClient() (workv1.WorkServiceClient, error) {
	conn, err := b.registry.GetConnection("work")
	if err != nil {
		return nil, err
	}
	return workv1.NewWorkServiceClient(conn), nil
}

// timeEntryWire is the exact shape the desktop client reads (FinanceTimeEntry
// in types/finance-types.ts, consumed through HoursToInvoiceDialog).
type timeEntryWire struct {
	ID          string  `json:"id"`
	Date        string  `json:"date"`
	Project     string  `json:"project"`
	Task        string  `json:"task"`
	Employee    string  `json:"employee"`
	Hours       float64 `json:"hours"`
	Description string  `json:"description"`
	Billed      bool    `json:"billed"`
}

// HandleListTimeEntries serves billable time entries for invoicing.
// GET /api/v1/finance/time-entries?billed=false
func (b *BizRoutes) HandleListTimeEntries(w http.ResponseWriter, r *http.Request) {
	client, err := b.getWorkClient()
	if err != nil {
		respondServiceUnavailable(w, "work")
		return
	}

	resp, err := client.ListBillableTimeEntries(r.Context(), &workv1.ListBillableTimeEntriesRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	// lean: nothing marks a time entry as billed yet -- the "Stunden ->
	// Rechnung" dialog creates the resulting invoice through the generic
	// create-invoice endpoint with hand-built line items, and never records
	// which entries it covered. Every entry is reported unbilled until that
	// linkage exists (upgrade trigger: persist entry IDs on the created
	// invoice, the way CreateInvoiceFromTimeEntries already does for HR work
	// time via finance_invoices.time_tracking_source).
	billedFilter := r.URL.Query().Get("billed")
	if billedFilter == "true" {
		response.JSON(w, http.StatusOK, map[string]any{"entries": []timeEntryWire{}})
		return
	}

	entries := make([]timeEntryWire, 0, len(resp.GetEntries()))
	for _, e := range resp.GetEntries() {
		entries = append(entries, timeEntryWire{
			ID:          e.GetId(),
			Date:        e.GetDate(),
			Project:     e.GetProject(),
			Task:        e.GetTask(),
			Employee:    e.GetEmployee(),
			Hours:       e.GetHours(),
			Description: e.GetDescription(),
			Billed:      false,
		})
	}

	response.JSON(w, http.StatusOK, map[string]any{"entries": entries})
}
