//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"
)

// TestSchichtenMemberFlow runs shift planning the way the roles meet it: the manager plans
// (templates, shifts, assignments, publishing) and decides on swap requests, plain members read
// the plan and ask for swaps. Bodies are copied from the desktop (schichten-client.ts,
// schichten-types.ts, useSchichten.ts). Needs the schichten service in the stack.
//
// What a member may not do is checked as 403 and points at `decide-member-manager-module-defaults`
// (BACKLOG-NEXT): the member preset holds only shift:read, swap:read and swap:create.
func TestSchichtenMemberFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	managerToken, _ := registerAndLoginWithRole(t, base, "manager")
	memberAToken, memberAID := registerAndLoginWithRole(t, base, "member")
	_, memberBID := registerAndLoginWithRole(t, base, "member")
	memberCToken, memberCID := registerAndLoginWithRole(t, base, "member")

	sch := base + "/api/v1/schichten"

	// A weekday eight hours in a quiet part of the calendar, so no other flow's shifts touch the
	// ArbZG rest-period check; the nanosecond offset keeps reruns on a shared database apart.
	start := time.Now().UTC().Add(30 * 24 * time.Hour).Truncate(24 * time.Hour).Add(8 * time.Hour)
	end := start.Add(8 * time.Hour)
	suffix := time.Now().UnixNano()

	var templateID, shiftID, assignmentA, assignmentB string

	assignmentIDs := func(t *testing.T) map[string]string {
		t.Helper()
		resp, body := getJSON(t, fmt.Sprintf("%s/shifts/%s/assignments", sch, shiftID), managerToken)
		requireStatus(t, resp, body, http.StatusOK)
		var out struct {
			Assignments []map[string]interface{} `json:"assignments"`
		}
		decodeBody(t, body, &out)
		byEmployee := map[string]string{}
		for _, a := range out.Assignments {
			emp, _ := a["employee_id"].(string)
			id, _ := a["id"].(string)
			byEmployee[emp] = id
		}
		return byEmployee
	}

	t.Run("manager: template with the desktop body", func(t *testing.T) {
		// CreateTemplateInput: name, description, day_of_week (0=Sunday), start_hour, start_minute,
		// duration_minutes, location.
		resp, body := postJSON(t, sch+"/templates", map[string]interface{}{
			"name":             fmt.Sprintf("E2E Frühschicht %d", suffix),
			"description":      "E2E template",
			"day_of_week":      1,
			"start_hour":       8,
			"start_minute":     0,
			"duration_minutes": 480,
			"location":         "Lager",
		}, managerToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var out struct {
			Template map[string]interface{} `json:"template"`
		}
		decodeBody(t, body, &out)
		templateID, _ = out.Template["id"].(string)
		if templateID == "" || out.Template["duration_minutes"] != float64(480) {
			t.Fatalf("expected the created template with its duration, got %s", string(body))
		}

		resp, body = putPatch(t, sch+"/templates/"+templateID, map[string]interface{}{"duration_minutes": 360}, managerToken)
		requireStatus(t, resp, body, http.StatusOK)
		out.Template = nil
		decodeBody(t, body, &out)
		if out.Template["duration_minutes"] != float64(360) || out.Template["start_hour"] != float64(8) {
			t.Fatalf("a partial update must change only the sent field, got %s", string(body))
		}

		resp, body = getJSON(t, sch+"/templates", managerToken)
		requireStatus(t, resp, body, http.StatusOK)
		var list struct {
			Templates []map[string]interface{} `json:"templates"`
			Total     float64                  `json:"total"`
		}
		decodeBody(t, body, &list)
		found := false
		for _, tpl := range list.Templates {
			if tpl["id"] == templateID {
				found = true
			}
		}
		if !found || list.Total < 1 {
			t.Fatalf("expected the template in {templates,total}, got %s", string(body))
		}
	})

	t.Run("manager: shift, assignments and the duplicate", func(t *testing.T) {
		// CreateShiftInput: title, description, start_time, end_time (RFC3339), location.
		resp, body := postJSON(t, sch+"/shifts", map[string]interface{}{
			"title":       fmt.Sprintf("E2E Schicht %d", suffix),
			"description": "E2E shift",
			"start_time":  start.Format(time.RFC3339),
			"end_time":    end.Format(time.RFC3339),
			"location":    "Lager",
		}, managerToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var out struct {
			Shift map[string]interface{} `json:"shift"`
		}
		decodeBody(t, body, &out)
		shiftID, _ = out.Shift["id"].(string)
		if shiftID == "" || out.Shift["status"] != "draft" {
			t.Fatalf("expected a draft shift, got %s", string(body))
		}

		// AssignEmployeeInput: employee_id (the user id).
		for _, emp := range []string{memberAID, memberBID} {
			resp, body = postJSON(t, fmt.Sprintf("%s/shifts/%s/assignments", sch, shiftID),
				map[string]interface{}{"employee_id": emp}, managerToken)
			requireStatus(t, resp, body, http.StatusCreated)
			var a struct {
				Assignment map[string]interface{} `json:"assignment"`
			}
			decodeBody(t, body, &a)
			if a.Assignment["employee_id"] != emp || a.Assignment["shift_id"] != shiftID {
				t.Fatalf("expected the assignment of %s, got %s", emp, string(body))
			}
		}

		resp, body = postJSON(t, fmt.Sprintf("%s/shifts/%s/assignments", sch, shiftID),
			map[string]interface{}{"employee_id": memberAID}, managerToken)
		requireStatus(t, resp, body, http.StatusConflict)

		ids := assignmentIDs(t)
		assignmentA, assignmentB = ids[memberAID], ids[memberBID]
		if len(ids) != 2 || assignmentA == "" || assignmentB == "" {
			t.Fatalf("expected exactly the two assignments, got %v", ids)
		}
	})

	t.Run("member: reads the plan and their own roster, nothing else", func(t *testing.T) {
		q := url.Values{"from": {start.Add(-time.Hour).Format(time.RFC3339)}, "to": {end.Add(time.Hour).Format(time.RFC3339)}}
		resp, body := getJSON(t, sch+"/shifts?"+q.Encode(), memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		var list struct {
			Shifts []map[string]interface{} `json:"shifts"`
			Total  float64                  `json:"total"`
		}
		decodeBody(t, body, &list)
		found := false
		for _, s := range list.Shifts {
			if s["id"] == shiftID {
				found = true
			}
		}
		if !found || list.Total < 1 {
			t.Fatalf("expected the shift in {shifts,total}, got %s", string(body))
		}

		resp, body = getJSON(t, sch+"/shifts/"+shiftID, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = getJSON(t, sch+"/stats", memberAToken)
		requireStatus(t, resp, body, http.StatusOK)

		cq := url.Values{
			"employee_id":     {memberAID},
			"new_shift_start": {start.Add(48 * time.Hour).Format(time.RFC3339)},
			"new_shift_end":   {end.Add(48 * time.Hour).Format(time.RFC3339)},
		}
		resp, body = getJSON(t, sch+"/compliance?"+cq.Encode(), memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		var comp map[string]interface{}
		decodeBody(t, body, &comp)
		if _, ok := comp["compliant"].(bool); !ok {
			t.Fatalf("expected a boolean compliant, got %s", string(body))
		}

		// The member reads their own roster (schichten:assignment:read at scope own, 000355): on
		// the shared shift only their own assignment, a member without one gets an empty list.
		ownRoster := func(token string) []map[string]interface{} {
			t.Helper()
			resp, body := getJSON(t, fmt.Sprintf("%s/shifts/%s/assignments", sch, shiftID), token)
			requireStatus(t, resp, body, http.StatusOK)
			var out struct {
				Assignments []map[string]interface{} `json:"assignments"`
			}
			decodeBody(t, body, &out)
			if out.Assignments == nil {
				t.Fatalf("expected assignments as a list, got %s", string(body))
			}
			return out.Assignments
		}
		if mine := ownRoster(memberAToken); len(mine) != 1 || mine[0]["employee_id"] != memberAID || mine[0]["id"] != assignmentA {
			t.Fatalf("member A should see only their own assignment %s, got %v", assignmentA, mine)
		}
		if none := ownRoster(memberCToken); len(none) != 0 {
			t.Fatalf("member C has no assignment on the shift and should see none, got %v", none)
		}

		// Shift templates are readable for the member (useTemplatesList, the "Vorlagen" tab).
		resp, body = getJSON(t, sch+"/templates", memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		var tpl struct {
			Templates []map[string]interface{} `json:"templates"`
		}
		decodeBody(t, body, &tpl)
		foundTpl := false
		for _, x := range tpl.Templates {
			if x["id"] == templateID {
				foundTpl = true
			}
		}
		if !foundTpl {
			t.Fatalf("expected template %s in the member's list, got %s", templateID, string(body))
		}

		// Everything that plans the roster is the manager's (decide-member-manager-module-defaults).
		refused := []struct {
			name, method, path string
			body               interface{}
		}{
			{"create template", http.MethodPost, "/templates", map[string]interface{}{
				"name": "x", "day_of_week": 1, "start_hour": 8, "start_minute": 0, "duration_minutes": 60,
			}},
			{"create shift", http.MethodPost, "/shifts", map[string]interface{}{
				"title": "x", "start_time": start.Format(time.RFC3339), "end_time": end.Format(time.RFC3339),
			}},
			{"update shift", http.MethodPatch, "/shifts/" + shiftID, map[string]interface{}{"title": "hijacked"}},
			{"delete shift", http.MethodDelete, "/shifts/" + shiftID, nil},
			{"publish", http.MethodPost, "/shifts/publish", map[string]interface{}{
				"from": start.Add(-time.Hour).Format(time.RFC3339), "to": end.Add(time.Hour).Format(time.RFC3339),
			}},
			{"assign", http.MethodPost, "/shifts/" + shiftID + "/assignments", map[string]interface{}{"employee_id": memberCID}},
			{"unassign", http.MethodDelete, "/shifts/" + shiftID + "/assignments/" + memberBID, nil},
		}
		for _, r := range refused {
			resp, body := doRequest(t, r.method, sch+r.path, r.body, memberAToken)
			if resp.StatusCode != http.StatusForbidden {
				t.Errorf("member %s: expected 403, got %d: %s", r.name, resp.StatusCode, string(body))
			}
		}

		// The refused calls changed nothing.
		resp, body = getJSON(t, sch+"/shifts/"+shiftID, managerToken)
		requireStatus(t, resp, body, http.StatusOK)
		var after struct {
			Shift map[string]interface{} `json:"shift"`
		}
		decodeBody(t, body, &after)
		if after.Shift["status"] != "draft" || after.Shift["title"] == "hijacked" {
			t.Fatalf("a refused member call changed the shift: %s", string(body))
		}
		if ids := assignmentIDs(t); len(ids) != 2 {
			t.Fatalf("a refused member call changed the roster: %v", ids)
		}
	})

	swapBody := func(assignmentID, requestedBy, swapWith string) map[string]interface{} {
		// useCreateSwapRequest: assignment_id, shift_id, requested_by_employee_id,
		// swap_with_employee_id, reason, idempotency_key (the header carries the key too).
		return map[string]interface{}{
			"assignment_id":            assignmentID,
			"shift_id":                 shiftID,
			"requested_by_employee_id": requestedBy,
			"swap_with_employee_id":    swapWith,
			"reason":                   "E2E swap",
			"idempotency_key":          fmt.Sprintf("e2e-swap-%d-%s", suffix, swapWith[:8]),
		}
	}
	type swapOut struct {
		SwapRequest map[string]interface{} `json:"swap_request"`
	}
	createSwap := func(t *testing.T, token string, body map[string]interface{}) string {
		t.Helper()
		resp, raw := financeRequest(t, http.MethodPost, sch+"/swap-requests", body, token)
		requireStatus(t, resp, raw, http.StatusCreated)
		var out swapOut
		decodeBody(t, raw, &out)
		id, _ := out.SwapRequest["id"].(string)
		if id == "" || out.SwapRequest["status"] != "pending" ||
			out.SwapRequest["requested_by_employee_id"] != body["requested_by_employee_id"] {
			t.Fatalf("expected a pending swap request, got %s", string(raw))
		}
		return id
	}
	listSwaps := func(t *testing.T, token string) []map[string]interface{} {
		t.Helper()
		resp, body := getJSON(t, sch+"/swap-requests", token)
		requireStatus(t, resp, body, http.StatusOK)
		var out struct {
			SwapRequests []map[string]interface{} `json:"swap_requests"`
			Total        float64                  `json:"total"`
		}
		decodeBody(t, body, &out)
		if out.SwapRequests == nil || int(out.Total) != len(out.SwapRequests) {
			t.Fatalf("expected {swap_requests,total} with a list that is never null, got %s", string(body))
		}
		return out.SwapRequests
	}
	swapStatus := func(t *testing.T, id string) string {
		t.Helper()
		for _, s := range listSwaps(t, managerToken) {
			if s["id"] == id {
				st, _ := s["status"].(string)
				return st
			}
		}
		t.Fatalf("swap request %s not in the manager's list", id)
		return ""
	}

	var swapToB, swapToC string

	t.Run("member: asks for a swap", func(t *testing.T) {
		swapToB = createSwap(t, memberAToken, swapBody(assignmentA, memberAID, memberBID))

		// Without the Idempotency-Key header the request is refused (the desktop adds it to every mutation).
		resp, body := postJSON(t, sch+"/swap-requests", swapBody(assignmentA, memberAID, memberBID), memberAToken)
		requireStatus(t, resp, body, http.StatusBadRequest)

		// A swap with oneself is not a swap.
		resp, body = financeRequest(t, http.MethodPost, sch+"/swap-requests", swapBody(assignmentA, memberAID, memberAID), memberAToken)
		requireStatus(t, resp, body, http.StatusBadRequest)

		// Not in a colleague's name: at scope own the service checks the requester against the
		// caller (user id or HR profile id).
		resp, body = financeRequest(t, http.MethodPost, sch+"/swap-requests", swapBody(assignmentB, memberBID, memberCID), memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)

		// A colleague who is not on the shift can be asked; the manager's approval is where it ends.
		swapToC = createSwap(t, memberAToken, swapBody(assignmentA, memberAID, memberCID))
	})

	t.Run("member: sees only the swaps they are part of", func(t *testing.T) {
		own := map[string]bool{}
		for _, s := range listSwaps(t, memberAToken) {
			own[s["id"].(string)] = true
		}
		if !own[swapToB] || !own[swapToC] {
			t.Fatalf("requester A must see both of their requests, got %v", own)
		}

		// C is the partner of the second request only; the first one is none of C's business.
		cSees := map[string]bool{}
		for _, s := range listSwaps(t, memberCToken) {
			cSees[s["id"].(string)] = true
		}
		if cSees[swapToB] || !cSees[swapToC] {
			t.Fatalf("partner C must see exactly the request that names them, got %v", cSees)
		}
	})

	t.Run("member: cannot decide on a swap", func(t *testing.T) {
		for _, action := range []string{"approve", "reject"} {
			resp, body := postJSON(t, fmt.Sprintf("%s/swap-requests/%s/%s", sch, swapToB, action), nil, memberAToken)
			if resp.StatusCode != http.StatusForbidden {
				t.Errorf("member %s: expected 403, got %d: %s", action, resp.StatusCode, string(body))
			}
		}
		if st := swapStatus(t, swapToB); st != "pending" {
			t.Fatalf("a refused decision changed the request to %q", st)
		}
	})

	t.Run("manager: decides", func(t *testing.T) {
		// An approved swap hands the requester's seat over (decision 2026-10-25). Partner B
		// already holds a seat on this shift and cannot take a second one: refused, nothing changes.
		resp, body := postJSON(t, fmt.Sprintf("%s/swap-requests/%s/approve", sch, swapToB), nil, managerToken)
		requireStatus(t, resp, body, http.StatusConflict)
		if st := swapStatus(t, swapToB); st != "pending" {
			t.Fatalf("a refused approval changed the request to %q", st)
		}
		if ids := assignmentIDs(t); len(ids) != 2 || ids[memberAID] != assignmentA || ids[memberBID] != assignmentB {
			t.Fatalf("a refused approval changed the roster: %v", ids)
		}

		resp, body = postJSON(t, fmt.Sprintf("%s/swap-requests/%s/reject", sch, swapToB), nil, managerToken)
		requireStatus(t, resp, body, http.StatusOK)
		var out swapOut
		decodeBody(t, body, &out)
		if out.SwapRequest["status"] != "rejected" {
			t.Fatalf("expected a rejected swap request, got %s", string(body))
		}

		// Partner C is not on the shift: approved, C takes over A's assignment row, A is off.
		resp, body = postJSON(t, fmt.Sprintf("%s/swap-requests/%s/approve", sch, swapToC), nil, managerToken)
		requireStatus(t, resp, body, http.StatusOK)
		out.SwapRequest = nil
		decodeBody(t, body, &out)
		if out.SwapRequest["status"] != "approved" {
			t.Fatalf("expected an approved swap request, got %s", string(body))
		}
		if st := swapStatus(t, swapToC); st != "approved" {
			t.Fatalf("expected the list to show approved, got %q", st)
		}
		ids := assignmentIDs(t)
		if _, ok := ids[memberAID]; ok {
			t.Fatalf("A must be off the shift after handing it over to C: %v", ids)
		}
		if ids[memberCID] != assignmentA || ids[memberBID] != assignmentB || len(ids) != 2 {
			t.Fatalf("C must hold A's former assignment %s, B keeps theirs: %v", assignmentA, ids)
		}

		// A decided request cannot be decided again.
		resp, body = postJSON(t, fmt.Sprintf("%s/swap-requests/%s/approve", sch, swapToC), nil, managerToken)
		requireStatus(t, resp, body, http.StatusConflict)
		resp, body = postJSON(t, fmt.Sprintf("%s/swap-requests/%s/reject", sch, swapToC), nil, managerToken)
		requireStatus(t, resp, body, http.StatusConflict)
	})

	t.Run("manager: publishes, unassigns and removes the shift", func(t *testing.T) {
		// PublishShiftsInput: from, to (RFC3339).
		resp, body := postJSON(t, sch+"/shifts/publish", map[string]interface{}{
			"from": start.Add(-time.Hour).Format(time.RFC3339),
			"to":   end.Add(time.Hour).Format(time.RFC3339),
		}, managerToken)
		requireStatus(t, resp, body, http.StatusOK)
		var pub struct {
			PublishedCount float64 `json:"published_count"`
		}
		decodeBody(t, body, &pub)
		if pub.PublishedCount < 1 {
			t.Fatalf("expected at least the one shift to be published, got %s", string(body))
		}

		// The member sees the published state.
		resp, body = getJSON(t, sch+"/shifts/"+shiftID, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		var shift struct {
			Shift map[string]interface{} `json:"shift"`
		}
		decodeBody(t, body, &shift)
		if shift.Shift["status"] != "published" {
			t.Fatalf("expected the member to see a published shift, got %s", string(body))
		}

		resp, body = deleteJSON(t, fmt.Sprintf("%s/shifts/%s/assignments/%s", sch, shiftID, memberBID), managerToken)
		requireStatus(t, resp, body, http.StatusNoContent)
		if ids := assignmentIDs(t); len(ids) != 1 {
			t.Fatalf("expected one assignment after unassigning B, got %v", ids)
		}

		resp, body = deleteJSON(t, sch+"/shifts/"+shiftID, managerToken)
		requireStatus(t, resp, body, http.StatusNoContent)
		resp, body = getJSON(t, sch+"/shifts/"+shiftID, managerToken)
		requireStatus(t, resp, body, http.StatusNotFound)

		resp, body = deleteJSON(t, sch+"/templates/"+templateID, managerToken)
		requireStatus(t, resp, body, http.StatusNoContent)
	})
}

// putPatch is doRequest with PATCH (the shift and template updates use it).
func putPatch(t *testing.T, url string, body interface{}, token string) (*http.Response, []byte) {
	t.Helper()
	return doRequest(t, http.MethodPatch, url, body, token)
}
