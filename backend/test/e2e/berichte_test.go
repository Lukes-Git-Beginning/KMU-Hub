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

// TestBerichteMemberFlow runs berichte against the real seed with the bodies the desktop sends (berichte-client.ts,
// berichte-types.ts): a member builds a report definition and a report document, runs and exports them, a manager
// schedules and runs it now, and everything a schedule or share link needs is turned away for the member. Needs the
// berichte service in the stack.
//
// IST of the seed that the test pins and does not judge:
//   - member holds `berichte:reports:create|read` at scope all and `berichte:reports:edit` at scope own (000256), plus the
//     legacy `berichte:reports:read` (000234); delete, publish, schedules and share links are 403; manager holds the
//     legacy `berichte:reports:write` and so also schedule:manage, share:manage and cache invalidation;
//   - the own scope of `berichte:reports:edit` holds: PATCH on a foreign definition or document answers 404 to a second
//     member; manager (legacy `berichte:reports:write`) and admin stay tenant-wide. A document's status change (the
//     lifecycle) needs `berichte:reports:publish` or `write`, so a member gets 403 even on their own document;
//   - the Report Builder saves the module of its source (`work`, `hr`, `rapporte`, ...); the service only knows
//     finanzen|crm|helpdesk|inventar|produktion|cross and answers 400 for the other eight sources
//     (fix-berichte-definition-module-allowlist-vs-builder-sources);
//   - `POST /berichte/preview` (Builder live preview) has no gateway route and answers 404 (fe-berichte-hide-live-preview);
//   - in the e2e stack the downstream services of the report executor are not wired, a run answers 200 with
//     `meta.warning = downstream_not_available` and no rows.
//
// There is no second tenant in the stack, so a foreign-tenant read is not part of this flow.
func TestBerichteMemberFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	managerToken, _ := registerAndLoginWithRole(t, base, "manager")
	memberAToken, memberAID := registerAndLoginWithRole(t, base, "member")
	memberBToken, _ := registerAndLoginWithRole(t, base, "member")
	adminToken, _ := registerAndLoginWithRole(t, base, "admin")

	b := base + "/api/v1/berichte"
	suffix := time.Now().UnixNano()

	obj := func(t *testing.T, body []byte) map[string]interface{} {
		t.Helper()
		var out map[string]interface{}
		decodeBody(t, body, &out)
		return out
	}
	str := func(m map[string]interface{}, key string) string {
		s, _ := m[key].(string)
		return s
	}
	nested := func(t *testing.T, body []byte, key string) map[string]interface{} {
		t.Helper()
		inner, ok := obj(t, body)[key].(map[string]interface{})
		if !ok {
			t.Fatalf("expected %q object in %s", key, string(body))
		}
		return inner
	}
	// rows decodes a `{<key>:[…], total}` list; the array must be present and never `null`.
	rows := func(t *testing.T, url, token, key string) []map[string]interface{} {
		t.Helper()
		resp, body := getJSON(t, url, token)
		requireStatus(t, resp, body, http.StatusOK)
		var raw map[string]json.RawMessage
		decodeBody(t, body, &raw)
		list, ok := raw[key]
		if !ok || strings.TrimSpace(string(list)) == "null" {
			t.Fatalf("%s must be an array, got %s", key, string(body))
		}
		var out []map[string]interface{}
		decodeBody(t, list, &out)
		return out
	}
	hasID := func(list []map[string]interface{}, id string) bool {
		for _, r := range list {
			if str(r, "id") == id {
				return true
			}
		}
		return false
	}

	var (
		defID         string
		docID         string
		builderDefIDs []string
		builderDocID  string
		schedID       string
		defName       = fmt.Sprintf("Pipeline Q4 %d", suffix)
		cronWeek      = "0 8 * * 1"
	)

	t.Run("member reads the module lists and the dashboard", func(t *testing.T) {
		defs := rows(t, b+"/definitions?page=1&page_size=50", memberAToken, "definitions")
		if len(defs) == 0 {
			t.Fatalf("the seed ships system definitions, member sees none")
		}
		_ = rows(t, b+"/schedules", memberAToken, "schedules")
		_ = rows(t, b+"/documents", memberAToken, "documents")
		if tpl := rows(t, b+"/templates", memberAToken, "templates"); len(tpl) == 0 {
			t.Fatalf("report templates are static starter structures, member sees none")
		}
		resp, body := getJSON(t, b+"/kpis", memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		if _, ok := obj(t, body)["kpis"].([]interface{}); !ok {
			t.Fatalf("kpis must be an array in %s", string(body))
		}
	})

	t.Run("member creates a definition with the desktop body", func(t *testing.T) {
		query := map[string]interface{}{"kind": "pipeline"}
		resp, body := postJSON(t, b+"/definitions", map[string]interface{}{"name": "", "module": "crm", "query_config": query}, memberAToken)
		requireStatus(t, resp, body, http.StatusBadRequest)
		resp, body = postJSON(t, b+"/definitions", map[string]interface{}{"name": "Bad module", "module": "bogus", "query_config": query}, memberAToken)
		requireStatus(t, resp, body, http.StatusBadRequest)
		resp, body = postJSON(t, b+"/definitions", map[string]interface{}{"name": "Bad format", "module": "crm", "query_config": query, "default_format": "doc"}, memberAToken)
		requireStatus(t, resp, body, http.StatusBadRequest)

		resp, body = postJSON(t, b+"/definitions", map[string]interface{}{
			"name": defName, "module": "crm", "kind": "custom", "query_config": query, "default_format": "pdf", "is_published": false,
		}, memberAToken)
		requireStatus(t, resp, body, http.StatusCreated)
		def := nested(t, body, "definition")
		defID = str(def, "id")
		if defID == "" || str(def, "created_by") != memberAID || str(def, "kind") != "custom" || def["is_published"] != false {
			t.Fatalf("unexpected definition %s", string(body))
		}

		// The Builder sends the module of its source; every source module is stored.
		builder := map[string]interface{}{"kind": "builder", "sourceId": "work", "viz": "bar", "dimensions": []string{}, "measures": []interface{}{}, "filters": []interface{}{}}
		for _, module := range []string{"work", "hr", "rapporte", "fuhrpark", "einkauf", "vertraege", "kommunikation", "zeiterfassung"} {
			resp, body = postJSON(t, b+"/definitions", map[string]interface{}{"name": "Builder " + module, "module": module, "kind": "custom", "query_config": builder}, memberAToken)
			requireStatus(t, resp, body, http.StatusCreated)
			saved := nested(t, body, "definition")
			if str(saved, "module") != module {
				t.Fatalf("builder definition stored under the wrong module: %s", string(body))
			}
			builderDefIDs = append(builderDefIDs, str(saved, "id"))
		}
		if !hasID(rows(t, b+"/definitions?module=fuhrpark&page_size=50", memberBToken, "definitions"), builderDefIDs[3]) {
			t.Fatalf("the module filter does not find the fuhrpark builder definition")
		}

		// A document can carry a builder module too.
		resp, body = postJSON(t, b+"/documents", map[string]interface{}{"title": "Builder doc", "module": "hr"}, memberAToken)
		requireStatus(t, resp, body, http.StatusCreated)
		builderDoc := nested(t, body, "document")
		if str(builderDoc, "module") != "hr" {
			t.Fatalf("document stored under the wrong module: %s", string(body))
		}
		builderDocID = str(builderDoc, "id")

		// The server executor does not know the builder kind: a run is a 400 with a message, never a 500.
		resp, body = postJSON(t, b+"/definitions/"+builderDefIDs[0]+"/run", map[string]interface{}{}, memberAToken)
		requireStatus(t, resp, body, http.StatusBadRequest)
		if !strings.Contains(string(body), "query config is invalid") {
			t.Fatalf("a builder run must name the invalid query config: %s", string(body))
		}
	})

	t.Run("a second member reads, runs and exports it but cannot delete it", func(t *testing.T) {
		resp, body := getJSON(t, b+"/definitions/"+defID, memberBToken)
		requireStatus(t, resp, body, http.StatusOK)
		if str(nested(t, body, "definition"), "name") != defName {
			t.Fatalf("unexpected definition %s", string(body))
		}
		if !hasID(rows(t, b+"/definitions?module=crm&search=Pipeline&page_size=50", memberBToken, "definitions"), defID) {
			t.Fatalf("member B does not find the definition by module and search")
		}
		resp, body = getJSON(t, b+"/definitions/00000000-0000-4000-8000-000000000000", memberBToken)
		requireStatus(t, resp, body, http.StatusNotFound)
		resp, body = getJSON(t, b+"/definitions/not-a-uuid", memberBToken)
		requireStatus(t, resp, body, http.StatusBadRequest)

		resp, body = postJSON(t, b+"/definitions/"+defID+"/run", map[string]interface{}{}, memberBToken)
		requireStatus(t, resp, body, http.StatusOK)
		run := obj(t, body)
		result, _ := run["result"].(map[string]interface{})
		if result == nil || run["run"] == nil {
			t.Fatalf("a run answers {result, run}: %s", string(body))
		}
		if _, ok := result["rows"].([]interface{}); !ok {
			t.Fatalf("result.rows must be an array in %s", string(body))
		}

		resp, body = postJSON(t, b+"/definitions/"+defID+"/export?format=csv", map[string]interface{}{"params": map[string]interface{}{}}, memberBToken)
		requireStatus(t, resp, body, http.StatusOK)
		if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, ".csv") {
			t.Fatalf("export lacks a csv Content-Disposition: %q", cd)
		}

		resp, body = deleteJSON(t, b+"/definitions/"+defID, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)
		resp, body = deleteJSON(t, b+"/definitions/"+defID+"/cache", memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)
	})

	t.Run("the author edits the definition, a second member cannot, manager and admin can", func(t *testing.T) {
		resp, body := doRequest(t, http.MethodPatch, b+"/definitions/"+defID, map[string]interface{}{"name": defName + " v2", "is_published": true}, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		def := nested(t, body, "definition")
		if str(def, "name") != defName+" v2" || def["is_published"] != true {
			t.Fatalf("unexpected definition %s", string(body))
		}
		// `berichte:reports:edit` is own for member: a foreign definition is not found, and stays as it was.
		resp, body = doRequest(t, http.MethodPatch, b+"/definitions/"+defID, map[string]interface{}{"name": "hijacked"}, memberBToken)
		requireStatus(t, resp, body, http.StatusNotFound)
		resp, body = getJSON(t, b+"/definitions/"+defID, memberBToken)
		requireStatus(t, resp, body, http.StatusOK)
		if str(nested(t, body, "definition"), "name") != defName+" v2" {
			t.Fatalf("a refused edit changed the definition: %s", string(body))
		}

		resp, body = doRequest(t, http.MethodPatch, b+"/definitions/"+defID, map[string]interface{}{"description": "manager"}, managerToken)
		requireStatus(t, resp, body, http.StatusOK)
		resp, body = doRequest(t, http.MethodPatch, b+"/definitions/"+defID, map[string]interface{}{"name": defName}, adminToken)
		requireStatus(t, resp, body, http.StatusOK)
		if str(nested(t, body, "definition"), "name") != defName {
			t.Fatalf("unexpected definition %s", string(body))
		}
	})

	t.Run("member creates a report document, edits but does not release it, a second member cannot touch it", func(t *testing.T) {
		resp, body := postJSON(t, b+"/documents", map[string]interface{}{"module": "bogus"}, memberAToken)
		requireStatus(t, resp, body, http.StatusBadRequest)

		resp, body = postJSON(t, b+"/documents", map[string]interface{}{
			"title": "Monatsbericht " + fmt.Sprint(suffix), "description": "e2e", "module": "cross",
			"rows":     []interface{}{},
			"settings": map[string]interface{}{"showHeader": true, "showPageNumbers": true},
		}, memberAToken)
		requireStatus(t, resp, body, http.StatusCreated)
		doc := nested(t, body, "document")
		docID = str(doc, "id")
		if docID == "" || str(doc, "created_by") != memberAID || str(doc, "status") != "draft" {
			t.Fatalf("unexpected document %s", string(body))
		}
		if _, ok := doc["rows"].([]interface{}); !ok {
			t.Fatalf("rows must be an array in %s", string(body))
		}

		resp, body = getJSON(t, b+"/documents/"+docID, memberBToken)
		requireStatus(t, resp, body, http.StatusOK)
		if !hasID(rows(t, b+"/documents?module=cross&status=draft", memberBToken, "documents"), docID) {
			t.Fatalf("member B does not see the document in the list")
		}

		resp, body = doRequest(t, http.MethodPatch, b+"/documents/"+docID, map[string]interface{}{"status": "archived-soon"}, memberAToken)
		requireStatus(t, resp, body, http.StatusBadRequest)
		// The lifecycle needs `publish`: the author's own status change is refused, a content save goes through.
		resp, body = doRequest(t, http.MethodPatch, b+"/documents/"+docID, map[string]interface{}{"title": "Monatsbericht v2", "status": "final"}, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)
		resp, body = doRequest(t, http.MethodPatch, b+"/documents/"+docID, map[string]interface{}{"title": "Monatsbericht v2", "rows": []interface{}{}}, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		if d := nested(t, body, "document"); str(d, "title") != "Monatsbericht v2" || str(d, "status") != "draft" {
			t.Fatalf("unexpected document %s", string(body))
		}
		// A second member neither edits nor releases the foreign document.
		resp, body = doRequest(t, http.MethodPatch, b+"/documents/"+docID, map[string]interface{}{"title": "hijacked"}, memberBToken)
		requireStatus(t, resp, body, http.StatusNotFound)
		resp, body = doRequest(t, http.MethodPatch, b+"/documents/"+docID, map[string]interface{}{"status": "released"}, memberBToken)
		requireStatus(t, resp, body, http.StatusForbidden)
		// manager (publish, legacy write) runs the lifecycle on the member's document, admin edits it.
		resp, body = doRequest(t, http.MethodPatch, b+"/documents/"+docID, map[string]interface{}{"status": "final"}, managerToken)
		requireStatus(t, resp, body, http.StatusOK)
		resp, body = doRequest(t, http.MethodPatch, b+"/documents/"+docID, map[string]interface{}{"title": "Monatsbericht v3", "status": "released"}, adminToken)
		requireStatus(t, resp, body, http.StatusOK)
		if d := nested(t, body, "document"); str(d, "title") != "Monatsbericht v3" || str(d, "status") != "released" {
			t.Fatalf("unexpected document %s", string(body))
		}

		resp, body = deleteJSON(t, b+"/documents/"+docID, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)
	})

	t.Run("share links are refused for a member, listing works", func(t *testing.T) {
		resp, body := postJSON(t, b+"/documents/"+docID+"/shares", map[string]interface{}{}, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)
		resp, body = getJSON(t, b+"/documents/"+docID+"/shares", memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		resp, body = deleteJSON(t, b+"/shares/00000000-0000-4000-8000-000000000000", memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)
	})

	t.Run("manager schedules the report and runs it now", func(t *testing.T) {
		// Validation: cron, address, format, name, unknown definition.
		in := func(m map[string]interface{}) map[string]interface{} {
			out := map[string]interface{}{
				"definition_id": defID, "name": "Weekly pipeline", "cron_expression": cronWeek,
				"recipients": []string{"team@example.com"}, "format": "pdf", "active": true,
			}
			for k, v := range m {
				out[k] = v
			}
			return out
		}
		for label, bad := range map[string]map[string]interface{}{
			"cron":       {"cron_expression": "not a cron"},
			"recipients": {"recipients": []string{"no-address"}},
			"format":     {"format": "doc"},
			"name":       {"name": ""},
		} {
			resp, body := postJSON(t, b+"/schedules", in(bad), managerToken)
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("bad %s: expected 400, got %d: %s", label, resp.StatusCode, string(body))
			}
		}
		resp, body := postJSON(t, b+"/schedules", in(map[string]interface{}{"definition_id": "00000000-0000-4000-8000-000000000000"}), managerToken)
		requireStatus(t, resp, body, http.StatusNotFound)

		resp, body = postJSON(t, b+"/schedules", in(nil), memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)

		resp, body = postJSON(t, b+"/schedules", in(nil), managerToken)
		requireStatus(t, resp, body, http.StatusCreated)
		sched := nested(t, body, "schedule")
		schedID = str(sched, "id")
		if schedID == "" || str(sched, "definition_id") != defID || str(sched, "cron_expression") != cronWeek || sched["active"] != true {
			t.Fatalf("unexpected schedule %s", string(body))
		}
		if rec, _ := sched["recipients"].([]interface{}); len(rec) != 1 {
			t.Fatalf("recipients must round-trip in %s", string(body))
		}

		// Run now answers the schedule back; an unknown id is 404, a bad id 400.
		resp, body = postJSON(t, b+"/schedules/"+schedID+"/run", nil, managerToken)
		requireStatus(t, resp, body, http.StatusOK)
		if str(nested(t, body, "schedule"), "id") != schedID {
			t.Fatalf("run-now must answer the schedule: %s", string(body))
		}
		resp, body = postJSON(t, b+"/schedules/00000000-0000-4000-8000-000000000000/run", nil, managerToken)
		requireStatus(t, resp, body, http.StatusNotFound)
		resp, body = postJSON(t, b+"/schedules/nope/run", nil, managerToken)
		requireStatus(t, resp, body, http.StatusBadRequest)
	})

	t.Run("member sees the schedule but changes nothing on it", func(t *testing.T) {
		// IST: schedules are readable with `berichte:reports:read`, recipients included.
		list := rows(t, b+"/schedules?definition_id="+defID+"&active=true", memberAToken, "schedules")
		if !hasID(list, schedID) {
			t.Fatalf("member does not see the schedule of the definition")
		}
		resp, body := postJSON(t, b+"/schedules/"+schedID+"/run", nil, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)
		resp, body = postJSON(t, b+"/schedules/"+schedID+"/toggle", map[string]interface{}{"active": false}, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)
		resp, body = doRequest(t, http.MethodPatch, b+"/schedules/"+schedID, map[string]interface{}{"name": "x"}, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)
		resp, body = deleteJSON(t, b+"/schedules/"+schedID, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)
	})

	t.Run("manager toggles, edits and removes the schedule", func(t *testing.T) {
		resp, body := postJSON(t, b+"/schedules/"+schedID+"/toggle", map[string]interface{}{"active": false}, managerToken)
		requireStatus(t, resp, body, http.StatusOK)
		if nested(t, body, "schedule")["active"] != false {
			t.Fatalf("toggle did not switch the schedule off: %s", string(body))
		}
		resp, body = doRequest(t, http.MethodPatch, b+"/schedules/"+schedID, map[string]interface{}{"name": "Weekly pipeline v2", "cron_expression": "30 7 * * 5", "format": "xlsx"}, managerToken)
		requireStatus(t, resp, body, http.StatusOK)
		sched := nested(t, body, "schedule")
		if str(sched, "name") != "Weekly pipeline v2" || str(sched, "cron_expression") != "30 7 * * 5" || str(sched, "format") != "xlsx" {
			t.Fatalf("unexpected schedule %s", string(body))
		}
		resp, body = doRequest(t, http.MethodPatch, b+"/schedules/"+schedID, map[string]interface{}{"cron_expression": "nope"}, managerToken)
		requireStatus(t, resp, body, http.StatusBadRequest)

		resp, body = deleteJSON(t, b+"/schedules/"+schedID, managerToken)
		requireStatus(t, resp, body, http.StatusNoContent)
		if hasID(rows(t, b+"/schedules", managerToken, "schedules"), schedID) {
			t.Fatalf("the schedule is still listed after the delete")
		}
		resp, body = postJSON(t, b+"/schedules/"+schedID+"/run", nil, managerToken)
		requireStatus(t, resp, body, http.StatusNotFound)
	})

	t.Run("the builder preview has no route (IST)", func(t *testing.T) {
		resp, body := postJSON(t, b+"/preview", map[string]interface{}{"query": map[string]interface{}{"kind": "builder", "sourceId": "work"}}, memberAToken)
		requireStatus(t, resp, body, http.StatusNotFound)
	})

	t.Run("manager cleans up the definition and the document", func(t *testing.T) {
		resp, body := deleteJSON(t, b+"/documents/"+docID, managerToken)
		requireStatus(t, resp, body, http.StatusNoContent)
		resp, body = getJSON(t, b+"/documents/"+docID, memberAToken)
		requireStatus(t, resp, body, http.StatusNotFound)
		resp, body = deleteJSON(t, b+"/definitions/"+defID, managerToken)
		requireStatus(t, resp, body, http.StatusNoContent)
		resp, body = getJSON(t, b+"/definitions/"+defID, memberAToken)
		requireStatus(t, resp, body, http.StatusNotFound)
		for _, id := range builderDefIDs {
			resp, body = deleteJSON(t, b+"/definitions/"+id, managerToken)
			requireStatus(t, resp, body, http.StatusNoContent)
		}
		resp, body = deleteJSON(t, b+"/documents/"+builderDocID, managerToken)
		requireStatus(t, resp, body, http.StatusNoContent)
	})
}
