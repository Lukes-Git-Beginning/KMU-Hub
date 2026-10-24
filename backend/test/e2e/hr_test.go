//go:build e2e

package e2e

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// hrCreateEmployeeProfile files the HR profile through the admin route (createEmployeeHTTPReq). The
// reporting line is set afterwards with setReportingLine.
func hrCreateEmployeeProfile(t *testing.T, base, adminToken, userID string) {
	t.Helper()
	resp, body := postJSON(t, base+"/api/v1/hr/employees", map[string]interface{}{
		"user_id":            userID,
		"department":         "E2E",
		"position_title":     "Tester",
		"contract_type":      "full_time",
		"work_days_per_week": 5,
		"annual_leave_days":  30,
		"start_date":         "2024-01-01",
	}, adminToken)
	requireStatus(t, resp, body, http.StatusCreated)
}

// hrLeaveRequests decodes a leave-request list (`{leave_requests, total}`, the key the desktop
// reads first, hr-client.ts listRequests) and returns its items.
func hrLeaveRequests(t *testing.T, base, query, token string) []map[string]interface{} {
	t.Helper()
	resp, body := getJSON(t, base+"/api/v1/hr/leave/requests"+query, token)
	requireStatus(t, resp, body, http.StatusOK)
	var result struct {
		LeaveRequests []map[string]interface{} `json:"leave_requests"`
		Total         int                      `json:"total"`
	}
	decodeBody(t, body, &result)
	if result.LeaveRequests == nil {
		t.Fatalf("expected a leave_requests array, got %s", string(body))
	}
	return result.LeaveRequests
}

func hrFindLeaveRequest(list []map[string]interface{}, id string) map[string]interface{} {
	for _, r := range list {
		if r["id"] == id {
			return r
		}
	}
	return nil
}

// TestMemberHRSelfServiceFlow is the working day of an employee in HR, as member, and the manager
// decisions on top of it, against the real seed keys (hr:self, team:absence:approve scope team,
// zeiterfassung:team:view scope team). The shape of every body and response is taken from
// desktop/src/renderer/src/api/hr-client.ts.
//
// Roles: A is a member, M is A's manager (reporting line set in the DB), M2 is a manager without
// any reporting line to A.
func TestMemberHRSelfServiceFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	adminToken, _ := registerAndLoginWithRole(t, base, "admin")
	memberToken, memberID := registerAndLoginWithRole(t, base, "member")
	managerToken, managerID := registerAndLoginWithRole(t, base, "manager")
	otherManagerToken, _ := registerAndLoginWithRole(t, base, "manager")

	hrCreateEmployeeProfile(t, base, adminToken, memberID)
	hrCreateEmployeeProfile(t, base, adminToken, managerID)
	setReportingLine(t, memberID, managerID)

	var leaveRequestID, leaveTypeID string

	t.Run("member reads the leave types", func(t *testing.T) {
		// hr-client.ts listTypes: a bare array (ProtoList), the FE also tolerates {types}.
		resp, body := getJSON(t, base+"/api/v1/hr/leave/types", memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var types []map[string]interface{}
		decodeBody(t, body, &types)
		if len(types) == 0 {
			t.Fatalf("expected the seeded leave types, got %s", string(body))
		}
		for _, lt := range types {
			if lt["requires_approval"] == true && lt["deducts_from_balance"] == true {
				leaveTypeID, _ = lt["id"].(string)
				break
			}
		}
		if leaveTypeID == "" {
			t.Fatalf("no leave type with requires_approval and deducts_from_balance in %s", string(body))
		}
	})
	t.Run("member files a leave request", func(t *testing.T) {
		// Two working days (Monday and Tuesday) four weeks out. Body: toSnakeCaseLeaveRequestBody.
		start := time.Now().UTC()
		for start.Weekday() != time.Monday {
			start = start.AddDate(0, 0, 1)
		}
		start = start.AddDate(0, 0, 28)
		resp, body := postJSON(t, base+"/api/v1/hr/leave/requests", map[string]interface{}{
			"leave_type_id":     leaveTypeID,
			"start_date":        start.Format("2006-01-02"),
			"end_date":          start.AddDate(0, 0, 1).Format("2006-01-02"),
			"is_half_day_start": false,
			"is_half_day_end":   false,
			"reason":            "E2E Urlaub",
		}, memberToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var lr map[string]interface{}
		decodeBody(t, body, &lr)
		if lr["employee_id"] != memberID {
			t.Fatalf("employee_id = %v, want the member %s", lr["employee_id"], memberID)
		}
		if lr["status"] != "LEAVE_PENDING" {
			t.Fatalf("status = %v, want LEAVE_PENDING", lr["status"])
		}
		leaveRequestID, _ = lr["id"].(string)
		if leaveRequestID == "" {
			t.Fatalf("expected an id, got %s", string(body))
		}
	})

	t.Run("member sees the request in the own list", func(t *testing.T) {
		list := hrLeaveRequests(t, base, "", memberToken)
		if hrFindLeaveRequest(list, leaveRequestID) == nil {
			t.Fatalf("own request %s missing from the member's list", leaveRequestID)
		}
		for _, r := range list {
			if r["employee_id"] != memberID {
				t.Fatalf("member list holds a foreign request: %v", r)
			}
		}
	})

	t.Run("member reads the own balance but not a colleague's", func(t *testing.T) {
		// hr-client.ts getBalance: the bare LeaveBalance.
		resp, body := getJSON(t, base+"/api/v1/hr/leave/balance", memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var bal map[string]interface{}
		decodeBody(t, body, &bal)
		if bal["employee_id"] != memberID {
			t.Fatalf("balance employee_id = %v, want %s", bal["employee_id"], memberID)
		}
		if s, _ := bal["entitlement"].(string); s == "" {
			t.Fatalf("expected an entitlement, got %s", string(body))
		}

		resp, body = getJSON(t, base+"/api/v1/hr/leave/balance/"+memberID, memberToken)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = getJSON(t, base+"/api/v1/hr/leave/balance/"+managerID, memberToken)
		requireStatus(t, resp, body, http.StatusForbidden)
	})

	t.Run("member cannot decide the own request", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/hr/leave/requests/"+leaveRequestID+"/approve",
			map[string]string{}, memberToken)
		requireStatus(t, resp, body, http.StatusForbidden)
	})

	t.Run("manager without a reporting line cannot approve", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/hr/leave/requests/"+leaveRequestID+"/approve",
			map[string]string{}, otherManagerToken)
		requireStatus(t, resp, body, http.StatusForbidden)
	})

	t.Run("member clocks in, takes a break and clocks out", func(t *testing.T) {
		// hr-client.ts clockIn: POST without a body; the gateway answers 201 with the bare entry.
		resp, body := postJSON(t, base+"/api/v1/hr/time/clock-in", nil, memberToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var entry map[string]interface{}
		decodeBody(t, body, &entry)
		if entry["employee_id"] != memberID {
			t.Fatalf("clock-in employee_id = %v, want %s", entry["employee_id"], memberID)
		}
		if _, ok := entry["clock_in"]; !ok {
			t.Fatalf("expected clock_in on the entry, got %s", string(body))
		}

		// startBreak/endBreak: the gateway sends the bare BreakEntry (no `break_entry` wrapper).
		resp, body = postJSON(t, base+"/api/v1/hr/time/break/start", nil, memberToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var br map[string]interface{}
		decodeBody(t, body, &br)
		if _, ok := br["start_time"]; !ok {
			t.Fatalf("expected start_time on the break, got %s", string(body))
		}

		resp, body = postJSON(t, base+"/api/v1/hr/time/break/end", nil, memberToken)
		requireStatus(t, resp, body, http.StatusOK)

		// clockOut: {entry, severity?, arbzg_message?}.
		resp, body = postJSON(t, base+"/api/v1/hr/time/clock-out", nil, memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var out struct {
			Entry map[string]interface{} `json:"entry"`
		}
		decodeBody(t, body, &out)
		if out.Entry["employee_id"] != memberID {
			t.Fatalf("clock-out entry = %s", string(body))
		}
		if _, ok := out.Entry["clock_out"]; !ok {
			t.Fatalf("expected clock_out after clocking out, got %s", string(body))
		}
	})

	t.Run("member reads the own time entries but not a colleague's", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/hr/time/entries", memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var result struct {
			Entries []map[string]interface{} `json:"entries"`
			Total   int                      `json:"total"`
		}
		decodeBody(t, body, &result)
		if len(result.Entries) == 0 {
			t.Fatalf("expected the shift just worked, got %s", string(body))
		}
		for _, e := range result.Entries {
			if e["employee_id"] != memberID {
				t.Fatalf("member entries hold a foreign entry: %v", e)
			}
		}

		resp, body = getJSON(t, base+"/api/v1/hr/time/entries?employee_id="+managerID, memberToken)
		requireStatus(t, resp, body, http.StatusForbidden)
	})

	// The week is the current one; the service normalises any day to the week start.
	weekStart := time.Now().UTC().Format("2006-01-02")

	t.Run("member submits the week", func(t *testing.T) {
		// hr-client.ts submitWeek: {week_start}, answer is the bare WeekApproval.
		resp, body := postJSON(t, base+"/api/v1/hr/time/weeks/submit",
			map[string]string{"week_start": weekStart}, memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var wa map[string]interface{}
		decodeBody(t, body, &wa)
		if wa["status"] != "submitted" {
			t.Fatalf("week status = %v, want submitted", wa["status"])
		}

		// getMyWeekStatus: {week_approval, total_minutes, target_minutes}.
		resp, body = getJSON(t, base+"/api/v1/hr/time/weeks/status?week_start="+weekStart, memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var st struct {
			WeekApproval map[string]interface{} `json:"week_approval"`
		}
		decodeBody(t, body, &st)
		if st.WeekApproval["status"] != "submitted" {
			t.Fatalf("week_approval = %s", string(body))
		}
	})

	t.Run("manager approves the request of the reporting line", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/hr/leave/requests/"+leaveRequestID+"/approve",
			map[string]string{}, managerToken)
		requireStatus(t, resp, body, http.StatusOK)
		var lr map[string]interface{}
		decodeBody(t, body, &lr)
		if lr["status"] != "LEAVE_APPROVED" {
			t.Fatalf("status = %v, want LEAVE_APPROVED", lr["status"])
		}
		if lr["approved_by"] != managerID {
			t.Fatalf("approved_by = %v, want %s", lr["approved_by"], managerID)
		}

		// The member sees the decision on the own request.
		resp, body = getJSON(t, base+"/api/v1/hr/leave/requests/"+leaveRequestID, memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		decodeBody(t, body, &lr)
		if lr["status"] != "LEAVE_APPROVED" {
			t.Fatalf("member sees status %v, want LEAVE_APPROVED", lr["status"])
		}
	})

	teamRows := func(t *testing.T, token string) []map[string]interface{} {
		t.Helper()
		// hr-client.ts getTeamTime: {team: [...]}, rows keyed by employee_id.
		resp, body := getJSON(t, base+"/api/v1/hr/time/team?week_start="+weekStart, token)
		requireStatus(t, resp, body, http.StatusOK)
		var result struct {
			Team []map[string]interface{} `json:"team"`
		}
		decodeBody(t, body, &result)
		if result.Team == nil {
			t.Fatalf("expected a team array, got %s", string(body))
		}
		return result.Team
	}
	rowFor := func(rows []map[string]interface{}, id string) map[string]interface{} {
		for _, r := range rows {
			if rid, _ := r["employee_id"].(string); strings.EqualFold(rid, id) {
				return r
			}
		}
		return nil
	}

	t.Run("team time shows the member to the manager only", func(t *testing.T) {
		row := rowFor(teamRows(t, managerToken), memberID)
		if row == nil {
			t.Fatal("the manager's team time does not list the member of the reporting line")
		}
		if row["week_status"] != "submitted" {
			t.Fatalf("week_status = %v, want submitted", row["week_status"])
		}

		if rowFor(teamRows(t, otherManagerToken), memberID) != nil {
			t.Fatal("a manager without a reporting line sees the member's team time")
		}
	})

	t.Run("member has no team time", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/hr/time/team?week_start="+weekStart, memberToken)
		requireStatus(t, resp, body, http.StatusForbidden)
	})
}
