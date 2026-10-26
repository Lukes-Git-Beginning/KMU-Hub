//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"
)

// qtyOf reads an int64 the way the desktop does (inventar-types.ts: "accept both and coerce"):
// protojson writes int64 as a JSON string, an int32 as a number.
func qtyOf(t *testing.T, v interface{}) int64 {
	t.Helper()
	switch x := v.(type) {
	case float64:
		return int64(x)
	case string:
		n, err := strconv.ParseInt(x, 10, 64)
		if err != nil {
			t.Fatalf("quantity %q is not an integer: %v", x, err)
		}
		return n
	}
	t.Fatalf("quantity has an unexpected JSON type %T (%v)", v, v)
	return 0
}

// concurrentPost is postJSON for goroutines: t.Fatalf must not be called off the test goroutine,
// so the outcome comes back as values.
func concurrentPost(url string, body interface{}, token string) (int, []byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return 0, nil, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, buf.Bytes(), nil
}

// TestInventarEinkaufFlow runs stock and purchasing the way a plain member meets them: the admin
// sets up the warehouse, the supplier and the orders, the member books stock movements and receives
// goods. Bodies are copied from the desktop (inventar-client.ts / inventar-types.ts,
// einkauf-client.ts / einkauf-types.ts). Needs the inventar and einkauf services in the stack.
func TestInventarEinkaufFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	adminToken, _ := registerAndLoginWithRole(t, base, "admin")
	memberToken, _ := registerAndLoginWithRole(t, base, "member")

	suffix := time.Now().UnixNano()
	inv := base + "/api/v1/inventar"
	buy := base + "/api/v1/einkauf"

	createItem := func(t *testing.T, sku string, quantity, minQuantity int) string {
		t.Helper()
		// CreateItemInput: name, sku, quantity, min_quantity, unit, location.
		resp, body := postJSON(t, inv+"/items", map[string]interface{}{
			"name":         "E2E item " + sku,
			"sku":          sku,
			"quantity":     quantity,
			"min_quantity": minQuantity,
			"unit":         "Stk",
		}, adminToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var out struct {
			Item map[string]interface{} `json:"item"`
		}
		decodeBody(t, body, &out)
		id, _ := out.Item["id"].(string)
		if id == "" || qtyOf(t, out.Item["quantity"]) != int64(quantity) {
			t.Fatalf("expected the item with quantity %d, got %s", quantity, string(body))
		}
		return id
	}
	itemQuantity := func(t *testing.T, token, id string) int64 {
		t.Helper()
		resp, body := getJSON(t, inv+"/items/"+id, token)
		requireStatus(t, resp, body, http.StatusOK)
		var out struct {
			Item map[string]interface{} `json:"item"`
		}
		decodeBody(t, body, &out)
		return qtyOf(t, out.Item["quantity"])
	}
	recordMovement := func(t *testing.T, id, movementType string, quantity int, wantStatus int) []byte {
		t.Helper()
		// RecordMovementInput: movement_type, quantity, reason (performed_by is optional).
		resp, body := postJSON(t, fmt.Sprintf("%s/items/%s/movements", inv, id), map[string]interface{}{
			"movement_type": movementType,
			"quantity":      quantity,
			"reason":        "E2E movement",
		}, memberToken)
		requireStatus(t, resp, body, wantStatus)
		return body
	}

	stockSKU := fmt.Sprintf("E2E-STK-%d", suffix)
	var stockItemID string

	t.Run("inventar: admin sets up a warehouse and an item", func(t *testing.T) {
		// CreateLocationInput: name, address, type.
		resp, body := postJSON(t, inv+"/locations", map[string]interface{}{
			"name":    fmt.Sprintf("E2E Lager %d", suffix),
			"address": "Teststrasse 1, 10115 Berlin",
			"type":    "warehouse",
		}, adminToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var loc struct {
			Location map[string]interface{} `json:"location"`
		}
		decodeBody(t, body, &loc)
		if id, _ := loc.Location["id"].(string); id == "" || loc.Location["type"] != "warehouse" {
			t.Fatalf("expected a warehouse location, got %s", string(body))
		}

		// Stock 10, minimum 5.
		stockItemID = createItem(t, stockSKU, 10, 5)
	})

	t.Run("inventar: member reads items, locations and the report", func(t *testing.T) {
		resp, body := getJSON(t, inv+"/items?search="+stockSKU, memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var list struct {
			Items []map[string]interface{} `json:"items"`
			Total *float64                 `json:"total"`
		}
		decodeBody(t, body, &list)
		if list.Total == nil || len(list.Items) != 1 || list.Items[0]["id"] != stockItemID {
			t.Fatalf("expected {items,total} with the new item, got %s", string(body))
		}

		resp, body = getJSON(t, inv+"/locations", memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var locs struct {
			Locations []map[string]interface{} `json:"locations"`
		}
		decodeBody(t, body, &locs)
		if len(locs.Locations) == 0 {
			t.Fatalf("expected the warehouse in the location list, got %s", string(body))
		}

		resp, body = getJSON(t, inv+"/report", memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var report map[string]interface{}
		decodeBody(t, body, &report)
		if _, ok := report["total_items"]; !ok {
			t.Fatalf("expected the stock report with total_items, got %s", string(body))
		}
	})

	t.Run("inventar: member books goods in and out, a low stock raises a warning", func(t *testing.T) {
		body := recordMovement(t, stockItemID, "in", 20, http.StatusCreated)
		var mv struct {
			Movement map[string]interface{} `json:"movement"`
		}
		decodeBody(t, body, &mv)
		if mv.Movement["movement_type"] != "in" || qtyOf(t, mv.Movement["quantity"]) != 20 {
			t.Fatalf("expected the in movement of 20, got %s", string(body))
		}
		if got := itemQuantity(t, memberToken, stockItemID); got != 30 {
			t.Fatalf("10 + 20 should leave 30, got %d", got)
		}

		// 30 - 28 = 2, below the minimum of 5: the service opens an active warning on its own.
		recordMovement(t, stockItemID, "out", 28, http.StatusCreated)
		if got := itemQuantity(t, memberToken, stockItemID); got != 2 {
			t.Fatalf("30 - 28 should leave 2, got %d", got)
		}

		resp, wbody := getJSON(t, inv+"/warnings?status=active", memberToken)
		requireStatus(t, resp, wbody, http.StatusOK)
		var warnings struct {
			Warnings []map[string]interface{} `json:"warnings"`
			Total    *float64                 `json:"total"`
		}
		decodeBody(t, wbody, &warnings)
		var found bool
		for _, w := range warnings.Warnings {
			if w["item_id"] == stockItemID {
				found = true
				if w["status"] != "active" {
					t.Fatalf("expected an active warning, got %v", w)
				}
			}
		}
		if warnings.Total == nil || !found {
			t.Fatalf("expected {warnings,total} with a warning for the item, got %s", string(wbody))
		}

		// More out than in stock is a precondition conflict, not a server fault, and changes nothing.
		recordMovement(t, stockItemID, "out", 5, http.StatusConflict)
		if got := itemQuantity(t, memberToken, stockItemID); got != 2 {
			t.Fatalf("a refused out movement must leave 2, got %d", got)
		}
		// Only in/out are bookable here; the other movement types and a zero quantity are 400.
		recordMovement(t, stockItemID, "adjustment", 1, http.StatusBadRequest)
		recordMovement(t, stockItemID, "in", 0, http.StatusBadRequest)

		for _, path := range []string{"movements", "history"} {
			resp, mbody := getJSON(t, fmt.Sprintf("%s/items/%s/%s", inv, stockItemID, path), memberToken)
			requireStatus(t, resp, mbody, http.StatusOK)
			var list struct {
				Movements []map[string]interface{} `json:"movements"`
				Total     *float64                 `json:"total"`
			}
			decodeBody(t, mbody, &list)
			if list.Total == nil || len(list.Movements) != 2 {
				t.Fatalf("%s: expected {movements,total} with the two booked movements, got %s", path, string(mbody))
			}
		}
	})

	t.Run("inventar: member is refused what the preset does not hold but transfers stock", func(t *testing.T) {
		other := createItem(t, fmt.Sprintf("E2E-OTH-%d", suffix), 1, 0)

		resp, body := postJSON(t, fmt.Sprintf("%s/items/%s/adjust", inv, stockItemID), map[string]interface{}{
			"delta": 5, "reason": "E2E adjust",
		}, memberToken)
		requireStatus(t, resp, body, http.StatusForbidden)

		resp, body = postJSON(t, inv+"/items", map[string]interface{}{
			"name": "E2E member item", "sku": fmt.Sprintf("E2E-MEM-%d", suffix), "unit": "Stk",
		}, memberToken)
		requireStatus(t, resp, body, http.StatusForbidden)

		resp, body = postJSON(t, inv+"/warnings", map[string]interface{}{
			"item_id": stockItemID, "threshold": 3, "current_quantity": 2,
		}, memberToken)
		requireStatus(t, resp, body, http.StatusForbidden)

		if got := itemQuantity(t, adminToken, stockItemID); got != 2 {
			t.Fatalf("refused writes must leave the stock at 2, got %d", got)
		}

		// Umbuchung: a transfer is one "out" plus one "in", so the member's movement:create covers
		// it (decision 2026-10-25). The transfer moves one unit.
		resp, body = postJSON(t, inv+"/transfer", map[string]interface{}{
			"from_item_id": stockItemID,
			"to_item_id":   other,
			"quantity":     1,
			"reason":       "E2E transfer",
		}, memberToken)
		requireStatus(t, resp, body, http.StatusNoContent)
		if from, to := itemQuantity(t, adminToken, stockItemID), itemQuantity(t, adminToken, other); from != 1 || to != 2 {
			t.Fatalf("transfer of 1 should give 1 and 2, got %d and %d", from, to)
		}
	})

	t.Run("inventar: parallel goods receipts on one item add up", func(t *testing.T) {
		// Lost-update fix 8cc079ce: ApplyStockMovementsTx locks the row, so concurrent movements
		// must not overwrite each other.
		id := createItem(t, fmt.Sprintf("E2E-PAR-%d", suffix), 0, 0)
		amounts := []int{3, 5, 7, 11, 13, 17}
		var want int64
		var wg sync.WaitGroup
		errs := make(chan string, len(amounts))
		for _, a := range amounts {
			want += int64(a)
			wg.Add(1)
			go func(a int) {
				defer wg.Done()
				status, body, err := concurrentPost(fmt.Sprintf("%s/items/%s/movements", inv, id), map[string]interface{}{
					"movement_type": "in", "quantity": a, "reason": "E2E parallel",
				}, memberToken)
				if err != nil || status != http.StatusCreated {
					errs <- fmt.Sprintf("movement %d: status %d err %v body %s", a, status, err, string(body))
				}
			}(a)
		}
		wg.Wait()
		close(errs)
		for e := range errs {
			t.Error(e)
		}
		if got := itemQuantity(t, memberToken, id); got != want {
			t.Fatalf("parallel receipts of %v should add up to %d, got %d", amounts, want, got)
		}
	})

	t.Run("einkauf: member receives goods and the stock rises", func(t *testing.T) {
		// CreateSupplierInput: name, email, ... Admin only (einkauf:supplier:create is not in the preset).
		resp, body := postJSON(t, buy+"/suppliers", map[string]interface{}{
			"name": fmt.Sprintf("E2E Lieferant %d", suffix), "email": "einkauf@e2e-lieferant.example",
		}, adminToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var sup struct {
			Supplier map[string]interface{} `json:"supplier"`
		}
		decodeBody(t, body, &sup)
		supplierID, _ := sup.Supplier["id"].(string)
		if supplierID == "" {
			t.Fatalf("expected a supplier id, got %s", string(body))
		}

		resp, body = postJSON(t, buy+"/suppliers", map[string]interface{}{"name": "E2E member supplier"}, memberToken)
		requireStatus(t, resp, body, http.StatusForbidden)

		resp, body = getJSON(t, buy+"/suppliers", memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var suppliers struct {
			Suppliers []map[string]interface{} `json:"suppliers"`
			Total     *float64                 `json:"total"`
		}
		decodeBody(t, body, &suppliers)
		if suppliers.Total == nil || len(suppliers.Suppliers) == 0 {
			t.Fatalf("expected {suppliers,total} for a reader, got %s", string(body))
		}

		sku := fmt.Sprintf("E2E-BUY-%d", suffix)
		itemID := createItem(t, sku, 0, 0)

		// newPO: CreatePOInput (supplier_id, po_number, order_date as YYYY-MM-DD) and one
		// AddPOLineInput (decimal strings). The admin builds and submits.
		var poSeq int
		newPO := func(t *testing.T, lineQuantity string) (poID, lineID string) {
			t.Helper()
			poSeq++
			resp, body := postJSON(t, buy+"/pos", map[string]interface{}{
				"supplier_id": supplierID,
				"po_number":   fmt.Sprintf("E2E-%d-%d", suffix, poSeq),
				"order_date":  time.Now().UTC().Format("2006-01-02"),
				"currency":    "EUR",
			}, adminToken)
			requireStatus(t, resp, body, http.StatusCreated)
			var po struct {
				PO map[string]interface{} `json:"po"`
			}
			decodeBody(t, body, &po)
			poID, _ = po.PO["id"].(string)
			if poID == "" || po.PO["status"] != "draft" {
				t.Fatalf("expected a draft order, got %s", string(body))
			}

			resp, body = postJSON(t, fmt.Sprintf("%s/pos/%s/lines", buy, poID), map[string]interface{}{
				"product_name": "E2E ware",
				"sku":          sku,
				"quantity":     lineQuantity,
				"unit_price":   "4.50",
				"tax_rate":     "0.19", // a fraction, NUMERIC(5,4); the desktop line dialog sends none
			}, adminToken)
			requireStatus(t, resp, body, http.StatusCreated)
			var line struct {
				Line map[string]interface{} `json:"line"`
			}
			decodeBody(t, body, &line)
			lineID, _ = line.Line["id"].(string)
			if lineID == "" {
				t.Fatalf("expected a line id, got %s", string(body))
			}
			return poID, lineID
		}
		submit := func(t *testing.T, poID string) {
			t.Helper()
			resp, body := postJSON(t, fmt.Sprintf("%s/pos/%s/submit", buy, poID), nil, adminToken)
			requireStatus(t, resp, body, http.StatusOK)
			var po struct {
				PO map[string]interface{} `json:"po"`
			}
			decodeBody(t, body, &po)
			if po.PO["status"] != "submitted" {
				t.Fatalf("expected a submitted order, got %s", string(body))
			}
		}
		poStatus := func(t *testing.T, poID string) string {
			t.Helper()
			resp, body := getJSON(t, buy+"/pos/"+poID, memberToken)
			requireStatus(t, resp, body, http.StatusOK)
			var po struct {
				PO map[string]interface{} `json:"po"`
			}
			decodeBody(t, body, &po)
			s, _ := po.PO["status"].(string)
			return s
		}

		// Full receipt by the member. Submitting stays with the buyer: einkauf:po:send/approve and
		// the line write are not in the member preset (BACKLOG-NEXT decide-member-manager-module-defaults).
		poID, _ := newPO(t, "12")
		resp, body = postJSON(t, fmt.Sprintf("%s/pos/%s/submit", buy, poID), nil, memberToken)
		requireStatus(t, resp, body, http.StatusForbidden)
		resp, body = postJSON(t, fmt.Sprintf("%s/pos/%s/receive", buy, poID), nil, memberToken)
		requireStatus(t, resp, body, http.StatusConflict) // still a draft
		submit(t, poID)

		resp, body = postJSON(t, fmt.Sprintf("%s/pos/%s/receive", buy, poID), nil, memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var received struct {
			PO map[string]interface{} `json:"po"`
		}
		decodeBody(t, body, &received)
		if received.PO["status"] != "received" {
			t.Fatalf("expected a received order, got %s", string(body))
		}
		if got := itemQuantity(t, memberToken, itemID); got != 12 {
			t.Fatalf("the goods receipt should lift the inventar stock to 12, got %d", got)
		}

		// A second receipt of the same order is refused and books nothing.
		resp, body = postJSON(t, fmt.Sprintf("%s/pos/%s/receive", buy, poID), nil, memberToken)
		requireStatus(t, resp, body, http.StatusConflict)
		if got := itemQuantity(t, memberToken, itemID); got != 12 {
			t.Fatalf("a refused receipt must leave 12, got %d", got)
		}

		// Partial receipt: received_quantity is the cumulative amount per line (4, then 10 of 10).
		partialPO, partialLine := newPO(t, "10")
		submit(t, partialPO)
		for i, step := range []struct {
			cumulative string
			status     string
			stock      int64
		}{{"4", "partially_received", 16}, {"10", "received", 22}} {
			resp, body = postJSON(t, fmt.Sprintf("%s/pos/%s/partial-receive", buy, partialPO), map[string]interface{}{
				"items": []map[string]interface{}{{"line_id": partialLine, "received_quantity": step.cumulative}},
			}, memberToken)
			requireStatus(t, resp, body, http.StatusOK)
			if got := poStatus(t, partialPO); got != step.status {
				t.Fatalf("step %d: expected order status %s, got %s", i, step.status, got)
			}
			if got := itemQuantity(t, memberToken, itemID); got != step.stock {
				t.Fatalf("step %d: expected stock %d, got %d", i, step.stock, got)
			}
		}
		// More than ordered is a 400 and books nothing.
		overPO, overLine := newPO(t, "2")
		submit(t, overPO)
		resp, body = postJSON(t, fmt.Sprintf("%s/pos/%s/partial-receive", buy, overPO), map[string]interface{}{
			"items": []map[string]interface{}{{"line_id": overLine, "received_quantity": "3"}},
		}, memberToken)
		requireStatus(t, resp, body, http.StatusBadRequest)
		if got := itemQuantity(t, memberToken, itemID); got != 22 {
			t.Fatalf("an over-receipt must leave 22, got %d", got)
		}

		// Two orders received in parallel on the same article: the stock is the sum of both.
		first, _ := newPO(t, "5")
		second, _ := newPO(t, "9")
		submit(t, first)
		submit(t, second)
		var wg sync.WaitGroup
		errs := make(chan string, 2)
		for _, id := range []string{first, second} {
			wg.Add(1)
			go func(id string) {
				defer wg.Done()
				status, rbody, err := concurrentPost(fmt.Sprintf("%s/pos/%s/receive", buy, id), nil, memberToken)
				if err != nil || status != http.StatusOK {
					errs <- fmt.Sprintf("receive %s: status %d err %v body %s", id, status, err, string(rbody))
				}
			}(id)
		}
		wg.Wait()
		close(errs)
		for e := range errs {
			t.Error(e)
		}
		if got := itemQuantity(t, memberToken, itemID); got != 22+5+9 {
			t.Fatalf("parallel receipts of 5 and 9 on 22 should give %d, got %d", 22+5+9, got)
		}

		// Members may open an order (einkauf:po:create) but not fill it (einkauf:line:write is held back).
		resp, body = postJSON(t, buy+"/pos", map[string]interface{}{
			"supplier_id": supplierID,
			"po_number":   fmt.Sprintf("E2E-%d-member", suffix),
		}, memberToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var own struct {
			PO map[string]interface{} `json:"po"`
		}
		decodeBody(t, body, &own)
		ownID, _ := own.PO["id"].(string)
		resp, body = postJSON(t, fmt.Sprintf("%s/pos/%s/lines", buy, ownID), map[string]interface{}{
			"product_name": "E2E ware", "sku": sku, "quantity": "1", "unit_price": "1.00",
		}, memberToken)
		requireStatus(t, resp, body, http.StatusForbidden)
	})
}
