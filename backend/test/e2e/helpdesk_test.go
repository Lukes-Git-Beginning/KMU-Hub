//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestHelpdeskMemberTicketFlow runs the ticket desk against the real seed with the bodies the desktop sends
// (helpdesk-client.ts, helpdesk-types.ts): the manager raises and assigns a ticket, members raise their own,
// read, reply and are turned away from everything an agent does. Needs the helpdesk service in the stack.
//
// IST of the seed that the test pins and does not judge:
//   - member holds `helpdesk:ticket:create|reply` at scope all, `helpdesk:ticket:read` at scope own and the
//     legacy coarse `helpdesk:read` (000234): lists, queues, canned responses, KB, SLA, routing rules, business
//     hours and stats are readable, every write on a ticket (edit, close, reopen, assign, merge, CSAT) and every
//     configuration write is 403; manager has the whole module;
//   - editing a ticket assigned to the member is 403 today (feat-helpdesk-assigned-tickets-edit-for-members);
//   - own scope narrows only the LIST (raised by or assigned to the caller): `GET /tickets/{id}`,
//     `GET|POST /tickets/{id}/messages` take the id as sent, so a second member opens, reads (internal notes
//     included) and answers a ticket that is neither theirs nor assigned to them
//     (fix-helpdesk-own-scope-ticket-detail-and-messages);
//   - the requester is always the caller: a `requester_id` in the desktop's body is dropped.
//
// There is no second tenant in the stack, so a foreign-tenant read is not part of this flow.
func TestHelpdeskMemberTicketFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	managerToken, managerID := registerAndLoginWithRole(t, base, "manager")
	memberAToken, memberAID := registerAndLoginWithRole(t, base, "member")
	memberBToken, memberBID := registerAndLoginWithRole(t, base, "member")

	hd := base + "/api/v1/helpdesk"
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
	// tickets decodes the list the desktop reads as `{tickets,total}`. An empty page may omit the array
	// (the desktop reads it with a fallback), but a `null` is the break it would not survive.
	tickets := func(t *testing.T, url, token string) ([]map[string]interface{}, int) {
		t.Helper()
		resp, body := getJSON(t, url, token)
		requireStatus(t, resp, body, http.StatusOK)
		var raw map[string]json.RawMessage
		decodeBody(t, body, &raw)
		if strings.TrimSpace(string(raw["tickets"])) == "null" {
			t.Fatalf("tickets must never be null, got %s", string(body))
		}
		var out []map[string]interface{}
		if rows, ok := raw["tickets"]; ok {
			decodeBody(t, rows, &out)
		}
		total := len(out)
		if tr, ok := raw["total"]; ok {
			decodeBody(t, tr, &total)
		}
		return out, total
	}
	hasID := func(rows []map[string]interface{}, id string) bool {
		for _, r := range rows {
			if str(r, "id") == id {
				return true
			}
		}
		return false
	}
	messages := func(t *testing.T, ticketID, token string) []map[string]interface{} {
		t.Helper()
		resp, body := getJSON(t, hd+"/tickets/"+ticketID+"/messages", token)
		requireStatus(t, resp, body, http.StatusOK)
		var raw map[string]json.RawMessage
		decodeBody(t, body, &raw)
		var out []map[string]interface{}
		if rows, ok := raw["messages"]; ok {
			decodeBody(t, rows, &out)
		}
		return out
	}
	readTicket := func(t *testing.T, id, token string) map[string]interface{} {
		t.Helper()
		resp, body := getJSON(t, hd+"/tickets/"+id, token)
		requireStatus(t, resp, body, http.StatusOK)
		return obj(t, body)
	}

	var assignedID, ownAID, ownBID string
	listURL := hd + "/tickets?page=1&page_size=100"

	t.Run("manager raises a ticket with the desktop's body and assigns it to member A", func(t *testing.T) {
		resp, b := postJSON(t, hd+"/tickets", map[string]interface{}{
			"subject":      fmt.Sprintf("E2E Drucker %d", suffix),
			"description":  "Druckt nur noch schwarz-weiß",
			"priority":     "high",
			"channel":      "agent",
			"requester_id": memberBID, // the desktop may send it; the gateway takes the caller
		}, managerToken)
		requireStatus(t, resp, b, http.StatusCreated)
		tk := obj(t, b)
		assignedID = str(tk, "id")
		if assignedID == "" || str(tk, "status") != "open" || str(tk, "priority") != "high" || str(tk, "tenant_id") == "" {
			t.Fatalf("unexpected created ticket: %s", string(b))
		}
		if str(tk, "requester_id") != managerID {
			t.Fatalf("requester must be the caller %s, got %s", managerID, str(tk, "requester_id"))
		}

		// Assigning is an agent action: member 403, ticket unchanged.
		resp, b = postJSON(t, hd+"/tickets/"+assignedID+"/assign", map[string]string{"assignee_id": memberAID}, memberAToken)
		requireStatus(t, resp, b, http.StatusForbidden)
		if a := str(readTicket(t, assignedID, managerToken), "assignee_id"); a != "" {
			t.Fatalf("a refused assignment must leave the ticket unassigned, got %q", a)
		}

		resp, b = postJSON(t, hd+"/tickets/"+assignedID+"/assign", map[string]string{"assignee_id": "nope"}, managerToken)
		requireStatus(t, resp, b, http.StatusBadRequest)
		resp, b = postJSON(t, hd+"/tickets/"+assignedID+"/assign", map[string]string{"assignee_id": memberAID}, managerToken)
		requireStatus(t, resp, b, http.StatusOK)
		if a := str(obj(t, b), "assignee_id"); a != memberAID {
			t.Fatalf("assignee after assign: want %s, got %q", memberAID, a)
		}
	})

	t.Run("members raise their own tickets and the requester is the caller", func(t *testing.T) {
		raise := func(token, subject string) map[string]interface{} {
			resp, b := postJSON(t, hd+"/tickets", map[string]interface{}{
				"subject": subject, "description": "vom E2E-Lauf", "priority": "normal", "channel": "agent",
				"requester_id": managerID, // not the caller: must be ignored
			}, token)
			requireStatus(t, resp, b, http.StatusCreated)
			return obj(t, b)
		}
		a := raise(memberAToken, fmt.Sprintf("E2E Tastatur A %d", suffix))
		ownAID = str(a, "id")
		if str(a, "requester_id") != memberAID || str(a, "status") != "open" {
			t.Fatalf("member A ticket: requester must be A, got %s", a)
		}
		b := raise(memberBToken, fmt.Sprintf("E2E Monitor B %d", suffix))
		ownBID = str(b, "id")
		if str(b, "requester_id") != memberBID {
			t.Fatalf("member B ticket: requester must be B, got %s", b)
		}

		for name, bad := range map[string]map[string]interface{}{
			"no subject":        {"subject": ""},
			"unknown priority":  {"subject": "x", "priority": "sofort"},
			"subject too long":  {"subject": strings.Repeat("x", 201)},
			"queue id not uuid": {"subject": "x", "queue_id": "q"},
			"assignee not uuid": {"subject": "x", "assignee_id": "a"},
		} {
			resp, body := postJSON(t, hd+"/tickets", bad, memberAToken)
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("%s: want 400, got %d: %s", name, resp.StatusCode, string(body))
			}
		}
	})

	t.Run("own scope narrows the list to raised or assigned tickets", func(t *testing.T) {
		rowsA, totalA := tickets(t, listURL, memberAToken)
		if !hasID(rowsA, ownAID) || !hasID(rowsA, assignedID) || hasID(rowsA, ownBID) {
			t.Fatalf("member A must see own + assigned and not B's ticket, got %d rows", len(rowsA))
		}
		if totalA != 2 || len(rowsA) != 2 {
			t.Fatalf("member A: want 2 rows / total 2, got %d / %d", len(rowsA), totalA)
		}
		rowsB, totalB := tickets(t, listURL, memberBToken)
		if len(rowsB) != 1 || totalB != 1 || !hasID(rowsB, ownBID) {
			t.Fatalf("member B must see exactly the own ticket, got %d / %d", len(rowsB), totalB)
		}

		// The status filter keeps the narrowing and the count.
		rowsF, totalF := tickets(t, hd+"/tickets?status=open&page_size=100", memberAToken)
		if totalF != 2 || len(rowsF) != 2 {
			t.Fatalf("filtered member A list: want 2 / 2, got %d / %d", len(rowsF), totalF)
		}
		rowsF, totalF = tickets(t, hd+"/tickets?status=solved&page_size=100", memberAToken)
		if totalF != 0 || len(rowsF) != 0 {
			t.Fatalf("no solved tickets yet, got %d / %d", len(rowsF), totalF)
		}

		// Manager sees the whole tenant.
		rowsM, _ := tickets(t, listURL, managerToken)
		for _, id := range []string{assignedID, ownAID, ownBID} {
			if !hasID(rowsM, id) {
				t.Fatalf("manager list must contain %s", id)
			}
		}
	})

	t.Run("member A reads the assigned ticket and replies", func(t *testing.T) {
		tk := readTicket(t, assignedID, memberAToken)
		if str(tk, "id") != assignedID || str(tk, "assignee_id") != memberAID {
			t.Fatalf("unexpected assigned ticket: %v", tk)
		}

		resp, b := postJSON(t, hd+"/tickets/"+assignedID+"/messages",
			map[string]interface{}{"body": "Ich schaue mir das an.", "internal": false, "attachments": []string{}}, memberAToken)
		requireStatus(t, resp, b, http.StatusCreated)
		m := obj(t, b)
		if str(m, "author_id") != memberAID || str(m, "ticket_id") != assignedID || m["internal"] == true {
			t.Fatalf("unexpected message: %s", string(b))
		}
		resp, b = postJSON(t, hd+"/tickets/"+assignedID+"/messages", map[string]interface{}{"body": ""}, memberAToken)
		requireStatus(t, resp, b, http.StatusBadRequest)

		resp, b = postJSON(t, hd+"/tickets/"+assignedID+"/messages",
			map[string]interface{}{"body": "Danke, Toner ist bestellt."}, managerToken)
		requireStatus(t, resp, b, http.StatusCreated)

		for who, token := range map[string]string{"member A": memberAToken, "manager": managerToken} {
			msgs := messages(t, assignedID, token)
			if len(msgs) != 2 {
				t.Fatalf("%s: want the two messages, got %d", who, len(msgs))
			}
			// An empty `attachments` is omitted on the wire; the desktop never reads it from the list.
		}
		// An untouched ticket reads as an empty list, never an error.
		if n := len(messages(t, ownBID, memberBToken)); n != 0 {
			t.Fatalf("a fresh ticket has no messages, got %d", n)
		}
	})

	t.Run("member is turned away from every ticket write and from configuration", func(t *testing.T) {
		// Editing the assigned ticket is 403 today: feat-helpdesk-assigned-tickets-edit-for-members flips this.
		resp, b := putJSON(t, hd+"/tickets/"+assignedID, map[string]interface{}{"priority": "low"}, memberAToken)
		requireStatus(t, resp, b, http.StatusForbidden)
		resp, b = putJSON(t, hd+"/tickets/"+ownAID, map[string]interface{}{"subject": "umbenannt"}, memberAToken)
		requireStatus(t, resp, b, http.StatusForbidden)
		for _, path := range []string{"/close", "/reopen"} {
			resp, b = postJSON(t, hd+"/tickets/"+assignedID+path, nil, memberAToken)
			requireStatus(t, resp, b, http.StatusForbidden)
		}
		resp, b = postJSON(t, hd+"/tickets/"+assignedID+"/csat", map[string]interface{}{"rating": 5, "comment": "top"}, memberAToken)
		requireStatus(t, resp, b, http.StatusForbidden)
		resp, b = postJSON(t, hd+"/tickets/"+ownAID+"/merge", map[string]string{"target_ticket_id": assignedID}, memberAToken)
		requireStatus(t, resp, b, http.StatusForbidden)

		tk := readTicket(t, assignedID, managerToken)
		if str(tk, "priority") != "high" || str(tk, "status") != "open" {
			t.Fatalf("refused writes must leave the ticket alone, got %v", tk)
		}
		if str(readTicket(t, ownAID, managerToken), "subject") != fmt.Sprintf("E2E Tastatur A %d", suffix) {
			t.Fatalf("the subject must be unchanged")
		}

		for name, c := range map[string]struct {
			method, path string
			body         interface{}
		}{
			"queue create":        {http.MethodPost, "/queues", map[string]string{"name": "Support"}},
			"canned create":       {http.MethodPost, "/canned-responses", map[string]string{"name": "Gruß", "body": "Hallo"}},
			"kb create":           {http.MethodPost, "/kb-articles", map[string]string{"title": "A", "content": "B"}},
			"sla create":          {http.MethodPost, "/sla-policies", map[string]interface{}{"name": "Std"}},
			"routing rule create": {http.MethodPost, "/routing-rules", map[string]interface{}{"name": "R"}},
			"business hours put":  {http.MethodPut, "/business-hours", map[string]interface{}{}},
		} {
			resp, body := doRequest(t, c.method, hd+c.path, c.body, memberAToken)
			if resp.StatusCode != http.StatusForbidden {
				t.Fatalf("%s: want 403, got %d: %s", name, resp.StatusCode, string(body))
			}
		}

		for _, path := range []string{"/queues", "/canned-responses", "/kb-articles", "/sla-policies", "/routing-rules", "/business-hours", "/stats"} {
			resp, body := getJSON(t, hd+path, memberAToken)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("GET %s as member: want 200, got %d: %s", path, resp.StatusCode, string(body))
			}
		}
	})

	t.Run("ids that do not exist are 404 or 400, not 500", func(t *testing.T) {
		resp, b := getJSON(t, hd+"/tickets/"+uuid.NewString(), memberAToken)
		requireStatus(t, resp, b, http.StatusNotFound)
		resp, b = getJSON(t, hd+"/tickets/not-a-uuid", memberAToken)
		requireStatus(t, resp, b, http.StatusBadRequest)
		// IST: the message list does not look the ticket up, so an unknown id reads as an empty list.
		if n := len(messages(t, uuid.NewString(), memberAToken)); n != 0 {
			t.Fatalf("an unknown ticket has no messages, got %d", n)
		}
		resp, b = postJSON(t, hd+"/tickets/"+uuid.NewString()+"/messages", map[string]string{"body": "hi"}, memberAToken)
		requireStatus(t, resp, b, http.StatusNotFound)
	})

	t.Run("a second member reaches a foreign ticket by id (own scope covers only the list)", func(t *testing.T) {
		// IST, fix-helpdesk-own-scope-ticket-detail-and-messages: member B neither raised the ticket nor is
		// assigned to it, yet detail, messages and replies work. When the unit lands these become 403/404.
		resp, b := getJSON(t, hd+"/tickets/"+ownAID, memberBToken)
		requireStatus(t, resp, b, http.StatusOK)
		resp, b = getJSON(t, hd+"/tickets/"+assignedID, memberBToken)
		requireStatus(t, resp, b, http.StatusOK)
		if n := len(messages(t, assignedID, memberBToken)); n != 2 {
			t.Fatalf("member B reads the messages of a foreign ticket today, want 2, got %d", n)
		}
		// Internal notes: the agent's note reaches the member's list as well, flagged `internal`.
		resp, b = postJSON(t, hd+"/tickets/"+assignedID+"/messages",
			map[string]interface{}{"body": "Nur intern: Garantie prüfen", "internal": true}, managerToken)
		requireStatus(t, resp, b, http.StatusCreated)
		seen := false
		for _, m := range messages(t, assignedID, memberBToken) {
			if m["internal"] == true && str(m, "body") == "Nur intern: Garantie prüfen" {
				seen = true
			}
		}
		if !seen {
			t.Fatalf("IST: the internal note is in the list of member B today")
		}
		resp, b = postJSON(t, hd+"/tickets/"+ownAID+"/messages", map[string]string{"body": "Mitgelesen von B"}, memberBToken)
		requireStatus(t, resp, b, http.StatusCreated)
		if got := str(obj(t, b), "author_id"); got != memberBID {
			t.Fatalf("the author of the foreign reply must be the caller, got %s", got)
		}
	})

	t.Run("manager runs the ticket through its states", func(t *testing.T) {
		resp, b := putJSON(t, hd+"/tickets/"+assignedID, map[string]interface{}{"status": "pending", "priority": "urgent"}, managerToken)
		requireStatus(t, resp, b, http.StatusOK)
		tk := obj(t, b)
		if str(tk, "status") != "pending" || str(tk, "priority") != "urgent" {
			t.Fatalf("update not applied: %s", string(b))
		}
		resp, b = putJSON(t, hd+"/tickets/"+assignedID, map[string]interface{}{"priority": "sofort"}, managerToken)
		requireStatus(t, resp, b, http.StatusBadRequest)

		resp, b = postJSON(t, hd+"/tickets/"+assignedID+"/close", nil, managerToken)
		requireStatus(t, resp, b, http.StatusOK)
		closed := str(obj(t, b), "status")
		if closed == "" || closed == "open" || closed == "pending" {
			t.Fatalf("close must leave a closed state, got %q", closed)
		}
		// A closed ticket leaves the open filter of its assignee and stays readable.
		_, totalOpen := tickets(t, hd+"/tickets?status=open&page_size=100", memberAToken)
		if totalOpen != 1 {
			t.Fatalf("member A has one open ticket left, got %d", totalOpen)
		}
		resp, b = postJSON(t, hd+"/tickets/"+assignedID+"/reopen", nil, managerToken)
		requireStatus(t, resp, b, http.StatusOK)
		if s := str(obj(t, b), "status"); s != "open" {
			t.Fatalf("reopen must give back an open ticket, got %q", s)
		}

		resp, b = getJSON(t, hd+"/stats", managerToken)
		requireStatus(t, resp, b, http.StatusOK)
	})
}
