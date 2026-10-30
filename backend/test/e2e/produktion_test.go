//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestProduktionMemberFlow runs the production module the way the roles meet it against the real
// seed (migrations 339 and earlier): the manager plans orders and machines, a plain member reads
// orders, reports work steps, starts and completes an order and files a quality check. Bodies are
// copied from the desktop (produktion-client.ts, produktion-types.ts). Needs the produktion service
// in the stack.
//
// Two things the seed does not give anyone but admin are checked as the IST state and point at
// `decide-member-manager-module-defaults` (BACKLOG-NEXT): creating plans and work steps
// (plan:write, workstep:write) and every machine-booking route (booking:read/write). The desktop
// has no caller for any of them yet, so no screen runs into the 403.
func TestProduktionMemberFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	adminToken, _ := registerAndLoginWithRole(t, base, "admin")
	managerToken, _ := registerAndLoginWithRole(t, base, "manager")
	memberAToken, _ := registerAndLoginWithRole(t, base, "member")
	memberBToken, _ := registerAndLoginWithRole(t, base, "member")

	prod := base + "/api/v1/produktion"
	suffix := time.Now().UnixNano()
	start := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Hour)

	type orderEnvelope struct {
		Order map[string]interface{} `json:"order"`
	}
	newOrder := func(t *testing.T, token, tag string) string {
		t.Helper()
		// CreateOrderInput: order_number, product_name, quantity, planned_start, planned_end (RFC3339),
		// priority, notes.
		resp, body := postJSON(t, prod+"/orders", map[string]interface{}{
			"order_number":  fmt.Sprintf("E2E-%s-%d", tag, suffix),
			"product_name":  "E2E Welle " + tag,
			"quantity":      25,
			"planned_start": start.Format(time.RFC3339),
			"planned_end":   start.Add(8 * time.Hour).Format(time.RFC3339),
			"priority":      2,
			"notes":         "E2E order",
		}, token)
		requireStatus(t, resp, body, http.StatusCreated)
		var out orderEnvelope
		decodeBody(t, body, &out)
		id, _ := out.Order["id"].(string)
		if id == "" || out.Order["status"] != "planned" || out.Order["quantity"] != float64(25) {
			t.Fatalf("expected a planned order with its quantity in {order}, got %s", string(body))
		}
		return id
	}

	var orderID, machineID, stepA, stepB string

	t.Run("manager: order, machine and the partial update", func(t *testing.T) {
		orderID = newOrder(t, managerToken, "A")

		resp, body := putPatch(t, prod+"/orders/"+orderID, map[string]interface{}{"quantity": 40, "notes": "E2E updated"}, managerToken)
		requireStatus(t, resp, body, http.StatusOK)
		var out orderEnvelope
		decodeBody(t, body, &out)
		if out.Order["quantity"] != float64(40) || out.Order["product_name"] != "E2E Welle A" {
			t.Fatalf("a partial update must change only the sent fields, got %s", string(body))
		}

		// A second order number in the same tenant is a conflict, not a 500.
		resp, body = postJSON(t, prod+"/orders", map[string]interface{}{
			"order_number":  fmt.Sprintf("E2E-A-%d", suffix),
			"product_name":  "dup",
			"quantity":      1,
			"planned_start": start.Format(time.RFC3339),
			"planned_end":   start.Add(time.Hour).Format(time.RFC3339),
		}, managerToken)
		requireStatus(t, resp, body, http.StatusConflict)

		// CreateMachineInput: name, type, notes.
		resp, body = postJSON(t, prod+"/machines", map[string]interface{}{
			"name":  fmt.Sprintf("E2E Fräse %d", suffix),
			"type":  "CNC",
			"notes": "E2E machine",
		}, managerToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var m struct {
			Machine map[string]interface{} `json:"machine"`
		}
		decodeBody(t, body, &m)
		machineID, _ = m.Machine["id"].(string)
		if machineID == "" || m.Machine["status"] != "available" {
			t.Fatalf("expected an available machine in {machine}, got %s", string(body))
		}

		resp, body = getJSON(t, prod+"/orders?status=planned&page_size=100", managerToken)
		requireStatus(t, resp, body, http.StatusOK)
		var list struct {
			Orders []map[string]interface{} `json:"orders"`
			Total  float64                  `json:"total"`
		}
		decodeBody(t, body, &list)
		found := false
		for _, o := range list.Orders {
			if o["id"] == orderID {
				found = true
			}
		}
		if !found || list.Total < 1 {
			t.Fatalf("expected the order in {orders,total}, got %s", string(body))
		}
	})

	t.Run("plans and work steps: admin creates, manager is refused", func(t *testing.T) {
		// CreatePlanInput: name, week_number, year, total_capacity_hours, planned_capacity_hours, notes.
		plan := map[string]interface{}{
			"name":                   "E2E Plan",
			"week_number":            12,
			"year":                   2027,
			"total_capacity_hours":   40,
			"planned_capacity_hours": 10,
		}
		resp, body := postJSON(t, prod+"/plans", plan, managerToken)
		requireStatus(t, resp, body, http.StatusForbidden)
		resp, body = postJSON(t, prod+"/plans", plan, adminToken)
		requireStatus(t, resp, body, http.StatusCreated)
		// The gateway answers the proto envelope {plan}; produktion-client.ts types createPlan/getPlan
		// as the flat ProductionPlan (no screen calls them yet — fix-or-adapt is a desktop decision).
		var pl struct {
			Plan map[string]interface{} `json:"plan"`
		}
		decodeBody(t, body, &pl)
		if pl.Plan["id"] == nil || pl.Plan["status"] != "draft" {
			t.Fatalf("expected a draft plan in {plan}, got %s", string(body))
		}

		// CreateWorkStepInput without order_id (the client puts it in the path).
		for i, name := range []string{"Zuschnitt", "Montage"} {
			step := map[string]interface{}{"step_nr": i + 1, "name": name, "description": "E2E", "duration_minutes": 30}
			resp, body = postJSON(t, prod+"/orders/"+orderID+"/steps", step, managerToken)
			requireStatus(t, resp, body, http.StatusForbidden)
			resp, body = postJSON(t, prod+"/orders/"+orderID+"/steps", step, adminToken)
			requireStatus(t, resp, body, http.StatusCreated)
			var s struct {
				Step map[string]interface{} `json:"step"`
			}
			decodeBody(t, body, &s)
			id, _ := s.Step["id"].(string)
			if id == "" || s.Step["status"] != "pending" || s.Step["order_id"] != orderID {
				t.Fatalf("expected a pending step of the order in {step}, got %s", string(body))
			}
			if i == 0 {
				stepA = id
			} else {
				stepB = id
			}
		}
	})

	t.Run("member: reads orders, machines and steps, reports a step", func(t *testing.T) {
		resp, body := getJSON(t, prod+"/orders/"+orderID, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		var o orderEnvelope
		decodeBody(t, body, &o)
		if o.Order["id"] != orderID {
			t.Fatalf("expected the order in {order}, got %s", string(body))
		}

		resp, body = getJSON(t, prod+"/machines", memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		var ml struct {
			Machines []map[string]interface{} `json:"machines"`
			Total    float64                  `json:"total"`
		}
		decodeBody(t, body, &ml)
		if ml.Machines == nil || ml.Total < 1 {
			t.Fatalf("expected {machines,total} with the machine, got %s", string(body))
		}

		resp, body = getJSON(t, prod+"/orders/"+orderID+"/steps", memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		var sl struct {
			Steps []map[string]interface{} `json:"steps"`
			Total float64                  `json:"total"`
		}
		decodeBody(t, body, &sl)
		if len(sl.Steps) != 2 || sl.Total != 2 {
			t.Fatalf("expected the two steps in {steps,total}, got %s", string(body))
		}

		// UpdateWorkStepInput: the order modal advances pending -> in_progress -> completed.
		for _, status := range []string{"in_progress", "completed"} {
			resp, body = putPatch(t, prod+"/orders/"+orderID+"/steps/"+stepA, map[string]interface{}{"status": status}, memberAToken)
			requireStatus(t, resp, body, http.StatusOK)
			var s struct {
				Step map[string]interface{} `json:"step"`
			}
			decodeBody(t, body, &s)
			if s.Step["status"] != status || s.Step["name"] != "Zuschnitt" {
				t.Fatalf("expected step status %s with the name kept, got %s", status, string(body))
			}
		}

		// A step id from another order does not resolve under this order. PATCH needs workstep:edit (member), DELETE workstep:write (admin).
		other := newOrder(t, managerToken, "S")
		resp, body = putPatch(t, prod+"/orders/"+other+"/steps/"+stepB, map[string]interface{}{"status": "in_progress"}, memberAToken)
		requireStatus(t, resp, body, http.StatusNotFound)
		resp, body = deleteJSON(t, prod+"/orders/"+other+"/steps/"+stepB, adminToken)
		requireStatus(t, resp, body, http.StatusNotFound)
	})

	t.Run("member: starts and completes the order, files a quality check", func(t *testing.T) {
		resp, body := postJSON(t, prod+"/orders/"+orderID+"/complete", nil, memberAToken)
		requireStatus(t, resp, body, http.StatusConflict) // planned -> completed is not a transition

		resp, body = postJSON(t, prod+"/orders/"+orderID+"/start", nil, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		var o orderEnvelope
		decodeBody(t, body, &o)
		if o.Order["status"] != "in_progress" || o.Order["actual_start"] == nil {
			t.Fatalf("expected an in_progress order with actual_start, got %s", string(body))
		}
		resp, body = postJSON(t, prod+"/orders/"+orderID+"/start", nil, memberAToken)
		requireStatus(t, resp, body, http.StatusConflict)

		// CreateQualityCheckInput: order_id, inspector, checked_at (optional), passed, defects_found, notes.
		resp, body = postJSON(t, prod+"/quality", map[string]interface{}{
			"order_id":      orderID,
			"inspector":     "E2E Prüfer",
			"passed":        true,
			"defects_found": 0,
			"notes":         "E2E check",
		}, memberAToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var qc struct {
			Check map[string]interface{} `json:"check"`
		}
		decodeBody(t, body, &qc)
		if qc.Check["order_id"] != orderID || qc.Check["passed"] != true {
			t.Fatalf("expected the check in {check}, got %s", string(body))
		}

		resp, body = getJSON(t, prod+"/quality?order_id="+orderID, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		var ql struct {
			Checks []map[string]interface{} `json:"checks"`
			Total  float64                  `json:"total"`
		}
		decodeBody(t, body, &ql)
		if len(ql.Checks) != 1 || ql.Total != 1 {
			t.Fatalf("expected exactly the one check of the order in {checks,total}, got %s", string(body))
		}

		resp, body = postJSON(t, prod+"/orders/"+orderID+"/complete", nil, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		o.Order = nil
		decodeBody(t, body, &o)
		if o.Order["status"] != "completed" || o.Order["actual_end"] == nil {
			t.Fatalf("expected a completed order with actual_end, got %s", string(body))
		}
	})

	t.Run("member: what the seed keeps from a member", func(t *testing.T) {
		// Order master data, cancelling, machines, plans, steps and the bookings belong to the
		// manager/admin (decide-member-manager-module-defaults). Member B stands in for "a second
		// member": orders are tenant-wide by design (order:read at scope all), so B sees A's order;
		// what B must not do is change it.
		resp, body := getJSON(t, prod+"/orders/"+orderID, memberBToken)
		requireStatus(t, resp, body, http.StatusOK)

		refused := []struct {
			name, method, path string
			body               interface{}
		}{
			{"create order", http.MethodPost, "/orders", map[string]interface{}{
				"order_number": fmt.Sprintf("E2E-M-%d", suffix), "product_name": "x", "quantity": 1,
				"planned_start": start.Format(time.RFC3339), "planned_end": start.Add(time.Hour).Format(time.RFC3339),
			}},
			{"edit order", http.MethodPatch, "/orders/" + orderID, map[string]interface{}{"quantity": 1}},
			{"delete order", http.MethodDelete, "/orders/" + orderID, nil},
			{"cancel order", http.MethodPost, "/orders/" + orderID + "/cancel", nil},
			{"create machine", http.MethodPost, "/machines", map[string]interface{}{"name": "x"}},
			{"delete machine", http.MethodDelete, "/machines/" + machineID, nil},
			{"create step", http.MethodPost, "/orders/" + orderID + "/steps", map[string]interface{}{"name": "x"}},
			{"delete step", http.MethodDelete, "/orders/" + orderID + "/steps/" + stepB, nil},
			{"create plan", http.MethodPost, "/plans", map[string]interface{}{"name": "x", "total_capacity_hours": 1}},
			{"list bookings", http.MethodGet, "/bookings", nil},
			{"create booking", http.MethodPost, "/bookings", map[string]interface{}{
				"machine_id": machineID, "production_order_id": orderID,
				"starts_at": start.Format(time.RFC3339), "ends_at": start.Add(time.Hour).Format(time.RFC3339),
			}},
		}
		for _, tc := range refused {
			for _, tok := range []string{memberAToken, memberBToken} {
				resp, body = doRequest(t, tc.method, prod+tc.path, tc.body, tok)
				if resp.StatusCode != http.StatusForbidden {
					t.Errorf("member %s: expected 403, got %d: %s", tc.name, resp.StatusCode, string(body))
				}
			}
		}

		// Nothing above went through.
		resp, body = getJSON(t, prod+"/orders/"+orderID, managerToken)
		requireStatus(t, resp, body, http.StatusOK)
		var o orderEnvelope
		decodeBody(t, body, &o)
		if o.Order["quantity"] != float64(40) || o.Order["status"] != "completed" {
			t.Fatalf("a refused member write must leave the order untouched, got %s", string(body))
		}
	})

	t.Run("manager: cancels and deletes planned orders, not finished ones", func(t *testing.T) {
		cancelID := newOrder(t, managerToken, "C")
		resp, body := postJSON(t, prod+"/orders/"+cancelID+"/cancel", nil, managerToken)
		requireStatus(t, resp, body, http.StatusOK)
		var o orderEnvelope
		decodeBody(t, body, &o)
		if o.Order["status"] != "cancelled" {
			t.Fatalf("expected a cancelled order, got %s", string(body))
		}

		deleteID := newOrder(t, managerToken, "D")
		resp, body = deleteJSON(t, prod+"/orders/"+deleteID, managerToken)
		requireStatus(t, resp, body, http.StatusNoContent)
		resp, body = getJSON(t, prod+"/orders/"+deleteID, managerToken)
		requireStatus(t, resp, body, http.StatusNotFound)

		resp, body = deleteJSON(t, prod+"/orders/"+orderID, managerToken)
		requireStatus(t, resp, body, http.StatusConflict) // completed: only planned orders are deletable
	})

	t.Run("admin: two moves onto one window, one wins and one is a 409", func(t *testing.T) {
		bookingOrder := newOrder(t, managerToken, "B")
		day := start.Add(24 * time.Hour)
		window := func(h int) (string, string) {
			return day.Add(time.Duration(h) * time.Hour).Format(time.RFC3339), day.Add(time.Duration(h+1) * time.Hour).Format(time.RFC3339)
		}

		// CreateBookingInput: machine_id, production_order_id, starts_at, ends_at (RFC3339), notes.
		book := func(h int) string {
			t.Helper()
			from, to := window(h)
			resp, body := postJSON(t, prod+"/bookings", map[string]interface{}{
				"machine_id": machineID, "production_order_id": bookingOrder,
				"starts_at": from, "ends_at": to, "notes": "E2E booking",
			}, adminToken)
			requireStatus(t, resp, body, http.StatusCreated)
			// {booking}: produktion-client.ts types createMachineBooking as the flat MachineBooking.
			var b struct {
				Booking map[string]interface{} `json:"booking"`
			}
			decodeBody(t, body, &b)
			id, _ := b.Booking["id"].(string)
			if id == "" || b.Booking["status"] != "booked" || b.Booking["machine_id"] != machineID {
				t.Fatalf("expected a booked booking in {booking}, got %s", string(body))
			}
			return id
		}

		first, second := book(1), book(3)

		// Overlapping create on a taken window is refused, an adjacent one is fine.
		from, to := window(1)
		resp, body := postJSON(t, prod+"/bookings", map[string]interface{}{
			"machine_id": machineID, "production_order_id": bookingOrder, "starts_at": from, "ends_at": to,
		}, adminToken)
		requireStatus(t, resp, body, http.StatusConflict)
		book(2)

		resp, body = getJSON(t, fmt.Sprintf("%s/bookings?machine_id=%s", prod, machineID), adminToken)
		requireStatus(t, resp, body, http.StatusOK)
		var list struct {
			Bookings []map[string]interface{} `json:"bookings"`
			Total    float64                  `json:"total"`
		}
		decodeBody(t, body, &list)
		if len(list.Bookings) != 3 || list.Total != 3 {
			t.Fatalf("expected the three bookings of the machine in {bookings,total}, got %s", string(body))
		}

		// Both bookings move onto the same free window at once (UpdateBookingInput: starts_at/ends_at).
		target := window
		from, to = target(6)
		var (
			wg    sync.WaitGroup
			gate  = make(chan struct{})
			codes = make([]int, 2)
			resps = make([]string, 2)
		)
		for i, id := range []string{first, second} {
			wg.Add(1)
			go func(i int, id string) {
				defer wg.Done()
				<-gate
				r, b := putPatch(t, prod+"/bookings/"+id, map[string]interface{}{"starts_at": from, "ends_at": to}, adminToken)
				codes[i], resps[i] = r.StatusCode, string(b)
			}(i, id)
		}
		close(gate)
		wg.Wait()

		ok, conflict := 0, 0
		for i, c := range codes {
			switch c {
			case http.StatusOK:
				ok++
			case http.StatusConflict:
				conflict++
			default:
				t.Errorf("racer %d: expected 200 or 409, got %d: %s", i, c, resps[i])
			}
		}
		if ok != 1 || conflict != 1 {
			t.Fatalf("expected exactly one move to win and one 409, got %v (%s | %s)", codes, resps[0], resps[1])
		}

		resp, body = getJSON(t, fmt.Sprintf("%s/bookings?machine_id=%s&production_order_id=%s", prod, machineID, bookingOrder), adminToken)
		requireStatus(t, resp, body, http.StatusOK)
		decodeBody(t, body, &list)
		onTarget := 0
		for _, b := range list.Bookings {
			if s, _ := b["starts_at"].(string); strings.HasPrefix(s, strings.TrimSuffix(from, "Z")) || s == from {
				onTarget++
			}
		}
		if onTarget != 1 {
			t.Fatalf("expected exactly one booking on the contested window, got %d in %s", onTarget, string(body))
		}
	})
}
