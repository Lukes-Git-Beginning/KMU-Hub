//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestRapporteMemberFlow runs the field-report module the way the roles meet it against the real
// seed: a member writes a report (lines, signature), hands it in, the manager rejects it with a
// note, the member corrects and resubmits, the manager approves. Bodies come from the desktop
// (rapporte-client.ts, rapporte-types.ts); the desktop sends reviewer_id and author_id, the gateway
// must take the caller instead. Needs the rapporte service in the stack.
//
// IST of the seed that the test pins and does not judge:
//   - the member holds report:create at scope all but edit/read at scope own, so a foreign
//     author_id on create is rewritten to the caller, and a second member sees nothing;
//   - approve/reject need report:approve (manager, admin) and are NOT narrowed to a reporting line:
//     any manager of the tenant can review any submitted report;
//   - template:write is admin only, a member only reads templates. The desktop has no template
//     caller, so "fill from a template" is the member reading the template and posting its lines;
//   - stats are tenant-wide counts for every reader (no own-scope narrowing).
func TestRapporteMemberFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	adminToken, _ := registerAndLoginWithRole(t, base, "admin")
	managerToken, managerID := registerAndLoginWithRole(t, base, "manager")
	memberAToken, memberAID := registerAndLoginWithRole(t, base, "member")
	memberBToken, memberBID := registerAndLoginWithRole(t, base, "member")

	rap := base + "/api/v1/rapporte"
	suffix := time.Now().UnixNano()

	type reportEnvelope struct {
		Report map[string]interface{} `json:"report"`
	}
	type reportList struct {
		Reports []map[string]interface{} `json:"reports"`
		Total   float64                  `json:"total"`
	}
	getReport := func(t *testing.T, token, id string, want int) map[string]interface{} {
		t.Helper()
		resp, body := getJSON(t, rap+"/reports/"+id, token)
		requireStatus(t, resp, body, want)
		if want != http.StatusOK {
			return nil
		}
		var out reportEnvelope
		decodeBody(t, body, &out)
		return out.Report
	}
	listHas := func(t *testing.T, url, token, id string) bool {
		t.Helper()
		resp, body := getJSON(t, url, token)
		requireStatus(t, resp, body, http.StatusOK)
		var out reportList
		decodeBody(t, body, &out)
		// An empty list is `{"reports":[],"total":0}`, never `{}` and never `null`.
		var raw map[string]json.RawMessage
		decodeBody(t, body, &raw)
		if got := strings.TrimSpace(string(raw["reports"])); !strings.HasPrefix(got, "[") {
			t.Fatalf("reports must be an array, got %q in %s", got, string(body))
		}
		if _, ok := raw["total"]; !ok {
			t.Fatalf("total missing in %s", string(body))
		}
		for _, r := range out.Reports {
			if r["id"] == id {
				return true
			}
		}
		return false
	}
	newReport := func(t *testing.T, token, authorID, title string) string {
		t.Helper()
		// CreateReportInput: title, description, author_id, lat, lon, report_date.
		resp, body := postJSON(t, rap+"/reports", map[string]interface{}{
			"title":       title,
			"description": "E2E Baustellenrapport",
			"author_id":   authorID,
			"lat":         48.137,
			"lon":         11.575,
			"report_date": time.Now().UTC().Format("2006-01-02"),
		}, token)
		requireStatus(t, resp, body, http.StatusCreated)
		var out reportEnvelope
		decodeBody(t, body, &out)
		id, _ := out.Report["id"].(string)
		if id == "" || out.Report["status"] != "draft" || out.Report["title"] != title {
			t.Fatalf("expected a draft report in {report}, got %s", string(body))
		}
		return id
	}

	var templateID, reportA, reportB, lineID string

	t.Run("template: admin writes, member only reads and fills", func(t *testing.T) {
		resp, body := postJSON(t, rap+"/templates", map[string]interface{}{
			"name":               "E2E Vorlage",
			"description":        "Tagesrapport",
			"category":           "bau",
			"default_lines_json": `[{"description":"Erdarbeiten","unit":"h"},{"description":"Material","unit":"Stk"}]`,
		}, adminToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var tpl struct {
			Template map[string]interface{} `json:"template"`
		}
		decodeBody(t, body, &tpl)
		templateID, _ = tpl.Template["id"].(string)
		if templateID == "" {
			t.Fatalf("expected the template in {template}, got %s", string(body))
		}

		resp, body = postJSON(t, rap+"/templates", map[string]interface{}{"name": "E2E member template"}, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)

		resp, body = getJSON(t, rap+"/templates?active_only=true", memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		var list struct {
			Templates []map[string]interface{} `json:"templates"`
		}
		decodeBody(t, body, &list)
		found := false
		for _, tp := range list.Templates {
			if tp["id"] == templateID {
				found = true
			}
		}
		if !found {
			t.Fatalf("the member must see the admin's template, got %s", string(body))
		}

		resp, body = getJSON(t, rap+"/templates/"+templateID, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		resp, body = deleteJSON(t, rap+"/templates/"+templateID, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)
	})

	t.Run("member A: report under own name, lines from the template, signature", func(t *testing.T) {
		// The body names member B as author — an own-scope author cannot file under another name.
		reportA = newReport(t, memberAToken, memberBID, fmt.Sprintf("E2E Rapport A %d", suffix))
		if author := getReport(t, memberAToken, reportA, http.StatusOK)["author_id"]; author != memberAID {
			t.Fatalf("a foreign author_id must be rewritten to the caller %s, got %v", memberAID, author)
		}

		resp, body := putPatch(t, rap+"/reports/"+reportA, map[string]interface{}{"description": "E2E geändert"}, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		var upd reportEnvelope
		decodeBody(t, body, &upd)
		if upd.Report["description"] != "E2E geändert" || upd.Report["title"] == "" {
			t.Fatalf("a partial update must change only the sent fields, got %s", string(body))
		}

		// AddLineInput: description, quantity, unit, notes, position.
		for i, ln := range []struct {
			desc, unit string
			qty        float64
		}{{"Erdarbeiten", "h", 6.5}, {"Material", "Stk", 12}} {
			resp, body = postJSON(t, rap+"/reports/"+reportA+"/lines", map[string]interface{}{
				"description": ln.desc, "quantity": ln.qty, "unit": ln.unit, "position": i + 1,
			}, memberAToken)
			requireStatus(t, resp, body, http.StatusCreated)
			var out struct {
				Line map[string]interface{} `json:"line"`
			}
			decodeBody(t, body, &out)
			if i == 0 {
				lineID, _ = out.Line["id"].(string)
			}
			if out.Line["id"] == "" || out.Line["report_id"] != reportA {
				t.Fatalf("expected the line in {line}, got %s", string(body))
			}
		}

		resp, body = putPatch(t, rap+"/lines/"+lineID, map[string]interface{}{"quantity": 7.5, "notes": "inkl. Pause"}, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = getJSON(t, rap+"/reports/"+reportA+"/lines", memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		var lines struct {
			Lines []map[string]interface{} `json:"lines"`
		}
		decodeBody(t, body, &lines)
		if len(lines.Lines) != 2 || lines.Lines[0]["quantity"] != 7.5 {
			t.Fatalf("expected two lines with the edited quantity in {lines}, got %s", string(body))
		}

		resp, body = putJSON(t, rap+"/reports/"+reportA+"/signature", map[string]interface{}{
			"signature_data": "data:image/png;base64,iVBORw0KGgo=",
			"signed_by":      "E2E Test",
		}, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = getJSON(t, rap+"/reports/"+reportA+"/export/pdf", memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		if !strings.HasPrefix(string(body), "%PDF") {
			t.Fatalf("expected a PDF body for the own report, got %d bytes", len(body))
		}

		resp, body = getJSON(t, rap+"/stats", memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
	})

	t.Run("member B: sees and touches nothing of member A", func(t *testing.T) {
		if listHas(t, rap+"/reports?page_size=100", memberBToken, reportA) {
			t.Fatal("member B must not see member A's report in the list")
		}
		// A client-chosen author_id cannot widen an own-scope list.
		if listHas(t, rap+"/reports?page_size=100&author_id="+memberAID, memberBToken, reportA) {
			t.Fatal("author_id in the query must not widen member B's list")
		}
		getReport(t, memberBToken, reportA, http.StatusNotFound)

		resp, body := putPatch(t, rap+"/reports/"+reportA, map[string]interface{}{"title": "gekapert"}, memberBToken)
		requireStatus(t, resp, body, http.StatusNotFound)
		resp, body = postJSON(t, rap+"/reports/"+reportA+"/submit", nil, memberBToken)
		requireStatus(t, resp, body, http.StatusNotFound)
		resp, body = getJSON(t, rap+"/reports/"+reportA+"/lines", memberBToken)
		requireStatus(t, resp, body, http.StatusNotFound)
		resp, body = postJSON(t, rap+"/reports/"+reportA+"/lines", map[string]interface{}{"description": "x", "quantity": 1}, memberBToken)
		requireStatus(t, resp, body, http.StatusNotFound)
		resp, body = putPatch(t, rap+"/lines/"+lineID, map[string]interface{}{"quantity": 99}, memberBToken)
		requireStatus(t, resp, body, http.StatusNotFound)
		resp, body = deleteJSON(t, rap+"/lines/"+lineID, memberBToken)
		requireStatus(t, resp, body, http.StatusNotFound)
		resp, body = getJSON(t, rap+"/reports/"+reportA+"/export/pdf", memberBToken)
		requireStatus(t, resp, body, http.StatusNotFound)
		// No report:delete in the member preset: refused before the author check is reached.
		resp, body = deleteJSON(t, rap+"/reports/"+reportA, memberBToken)
		requireStatus(t, resp, body, http.StatusForbidden)

		if r := getReport(t, memberAToken, reportA, http.StatusOK); r["title"] == "gekapert" || r["status"] != "draft" {
			t.Fatalf("member A's report must be unchanged after B's attempts, got %v", r)
		}
	})

	t.Run("member A submits; the report is then locked for the author", func(t *testing.T) {
		resp, body := postJSON(t, rap+"/reports/"+reportA+"/submit", nil, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		var out reportEnvelope
		decodeBody(t, body, &out)
		if out.Report["status"] != "submitted" {
			t.Fatalf("expected status submitted, got %s", string(body))
		}

		resp, body = postJSON(t, rap+"/reports/"+reportA+"/submit", nil, memberAToken)
		requireStatus(t, resp, body, http.StatusConflict)
		resp, body = putPatch(t, rap+"/reports/"+reportA, map[string]interface{}{"title": "nachträglich"}, memberAToken)
		requireStatus(t, resp, body, http.StatusConflict)
		resp, body = postJSON(t, rap+"/reports/"+reportA+"/lines", map[string]interface{}{"description": "spät", "quantity": 1}, memberAToken)
		requireStatus(t, resp, body, http.StatusConflict)

		// The review surface is closed to a member, also for the own report.
		resp, body = postJSON(t, rap+"/reports/"+reportA+"/approve", map[string]interface{}{"reviewer_id": memberAID}, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)
		resp, body = postJSON(t, rap+"/reports/"+reportA+"/reject", map[string]interface{}{"reviewer_id": memberAID, "review_note": "x"}, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)
	})

	t.Run("manager: queue, reject needs a note, reject, approve a draft is a conflict", func(t *testing.T) {
		resp, body := getJSON(t, rap+"/pending?page_size=100", managerToken)
		requireStatus(t, resp, body, http.StatusOK)
		var pending reportList
		decodeBody(t, body, &pending)
		found := false
		for _, r := range pending.Reports {
			if r["id"] == reportA {
				found = true
			}
		}
		if !found {
			t.Fatalf("the manager's approval queue must hold the submitted report, got %s", string(body))
		}

		// The queue of member B holds nothing of A.
		resp, body = getJSON(t, rap+"/pending?page_size=100", memberBToken)
		requireStatus(t, resp, body, http.StatusOK)
		var bPending reportList
		decodeBody(t, body, &bPending)
		for _, r := range bPending.Reports {
			if r["id"] == reportA {
				t.Fatal("member B's pending list leaks member A's submitted report")
			}
		}

		resp, body = postJSON(t, rap+"/reports/"+reportA+"/reject", map[string]interface{}{"reviewer_id": managerID, "review_note": "  "}, managerToken)
		requireStatus(t, resp, body, http.StatusBadRequest)

		// The desktop names a reviewer_id; the stored reviewer is the caller, never the body.
		resp, body = postJSON(t, rap+"/reports/"+reportA+"/reject", map[string]interface{}{"reviewer_id": memberBID, "review_note": "Stunden prüfen"}, managerToken)
		requireStatus(t, resp, body, http.StatusOK)
		var out reportEnvelope
		decodeBody(t, body, &out)
		if out.Report["status"] != "rejected" || out.Report["review_note"] != "Stunden prüfen" || out.Report["reviewer_id"] != managerID {
			t.Fatalf("expected a rejected report reviewed by the manager, got %s", string(body))
		}

		resp, body = postJSON(t, rap+"/reports/"+reportA+"/approve", map[string]interface{}{"reviewer_id": managerID}, managerToken)
		requireStatus(t, resp, body, http.StatusConflict)

		// A draft cannot be approved either.
		reportB = newReport(t, memberAToken, memberAID, fmt.Sprintf("E2E Rapport B %d", suffix))
		resp, body = postJSON(t, rap+"/reports/"+reportB+"/approve", map[string]interface{}{"reviewer_id": managerID}, managerToken)
		requireStatus(t, resp, body, http.StatusConflict)
	})

	t.Run("member A corrects the rejected report and resubmits; manager approves", func(t *testing.T) {
		r := getReport(t, memberAToken, reportA, http.StatusOK)
		if r["status"] != "rejected" || r["review_note"] != "Stunden prüfen" {
			t.Fatalf("the author must see the rejection and its note, got %v", r)
		}

		resp, body := putPatch(t, rap+"/lines/"+lineID, map[string]interface{}{"quantity": 6}, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		resp, body = putPatch(t, rap+"/reports/"+reportA, map[string]interface{}{"description": "E2E korrigiert"}, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = postJSON(t, rap+"/reports/"+reportA+"/submit", nil, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		var sub reportEnvelope
		decodeBody(t, body, &sub)
		if sub.Report["status"] != "submitted" {
			t.Fatalf("expected the resubmitted report to be submitted, got %s", string(body))
		}

		resp, body = postJSON(t, rap+"/reports/"+reportA+"/approve", map[string]interface{}{"reviewer_id": managerID, "review_note": "passt"}, managerToken)
		requireStatus(t, resp, body, http.StatusOK)
		var out reportEnvelope
		decodeBody(t, body, &out)
		if out.Report["status"] != "approved" || out.Report["reviewer_id"] != managerID {
			t.Fatalf("expected an approved report, got %s", string(body))
		}

		resp, body = postJSON(t, rap+"/reports/"+reportA+"/approve", map[string]interface{}{"reviewer_id": managerID}, managerToken)
		requireStatus(t, resp, body, http.StatusConflict)

		// Approved is immutable, also for the author.
		resp, body = putPatch(t, rap+"/reports/"+reportA, map[string]interface{}{"title": "nach Freigabe"}, memberAToken)
		requireStatus(t, resp, body, http.StatusConflict)
		resp, body = deleteJSON(t, rap+"/lines/"+lineID, memberAToken)
		requireStatus(t, resp, body, http.StatusConflict)

		// The approved report stays readable for its author.
		if r := getReport(t, memberAToken, reportA, http.StatusOK); r["status"] != "approved" {
			t.Fatalf("expected the author to read the approved report, got %v", r)
		}
	})

	t.Run("member A: delete is refused for report:delete; own draft can be cleaned up by the manager", func(t *testing.T) {
		// IST: the member has no report:delete (only the manager and admin do), so even a
		// draft cannot be dropped by its author.
		resp, body := deleteJSON(t, rap+"/reports/"+reportB, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)

		resp, body = deleteJSON(t, rap+"/reports/"+reportB, managerToken)
		requireStatus(t, resp, body, http.StatusNoContent)
		getReport(t, managerToken, reportB, http.StatusNotFound)
	})
}
