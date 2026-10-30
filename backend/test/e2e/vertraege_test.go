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

// TestVertraegeMemberFlow runs the contracts module against the real seed with the bodies the desktop
// sends (vertraege-client.ts, VertraegePage.tsx): the manager creates a contract with the create dialog's
// body, edits it like the edit dialog (title, type, dates, clear_ends_on, notes), attaches a party and a
// reminder and terminates it with the termination dialog's `{status:"terminated"}`; members read. Needs the
// vertraege service in the stack.
//
// IST of the seed that the test pins and does not judge:
//   - member has `vertraege:contract|party|reminder:read` (000234) and nothing else: every write, the
//     signature PUT and the delete are 403; manager has the whole resource (000234) plus the fine keys
//     of 000256, admin everything;
//   - reads are tenant-wide for member (no own-scope): the second member sees and exports the contract
//     of the manager, the audit trail included;
//   - the create dialog sends no status, so a new contract stays `draft` in the backend (only a draft
//     can be deleted: 409 otherwise), the dialog's local store shows it as `active`;
//   - the desktop list call sends no page_size (`useContracts()`), so it sees the default page of 20 and
//     `total` carries the rest;
//   - the desktop's contract signing runs in a local store, the backend `PUT /signature` has no caller in
//     the client; the route is pinned for its permission and its validation only.
//
// There is no second tenant in the stack, so a foreign-tenant read is not part of this flow.
func TestVertraegeMemberFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	managerToken, _ := registerAndLoginWithRole(t, base, "manager")
	memberAToken, _ := registerAndLoginWithRole(t, base, "member")
	memberBToken, _ := registerAndLoginWithRole(t, base, "member")

	vt := base + "/api/v1/vertraege"
	suffix := time.Now().UnixNano()
	day := func(offset int) string { return time.Now().UTC().AddDate(0, 0, offset).Format("2006-01-02") }

	patch := func(t *testing.T, url string, body interface{}, token string) (*http.Response, []byte) {
		t.Helper()
		return doRequest(t, http.MethodPatch, url, body, token)
	}
	object := func(t *testing.T, body []byte, key string) map[string]interface{} {
		t.Helper()
		var raw map[string]json.RawMessage
		decodeBody(t, body, &raw)
		rr, ok := raw[key]
		if !ok {
			t.Fatalf("expected %q in the response, got %s", key, string(body))
		}
		var out map[string]interface{}
		decodeBody(t, rr, &out)
		return out
	}
	// list decodes a wrapped list; a missing key is an empty list, `null` is the break the desktop's
	// `?? []` would not survive.
	list := func(t *testing.T, url, token, key string) ([]map[string]interface{}, int) {
		t.Helper()
		resp, body := getJSON(t, url, token)
		requireStatus(t, resp, body, http.StatusOK)
		var raw map[string]json.RawMessage
		decodeBody(t, body, &raw)
		var out []map[string]interface{}
		if rr, ok := raw[key]; ok {
			if strings.TrimSpace(string(rr)) == "null" {
				t.Fatalf("%s must never be null, got %s", key, string(body))
			}
			decodeBody(t, rr, &out)
		}
		total := len(out)
		if tr, ok := raw["total"]; ok {
			decodeBody(t, tr, &total)
		}
		return out, total
	}
	str := func(m map[string]interface{}, key string) string {
		s, _ := m[key].(string)
		return s
	}

	number := fmt.Sprintf("E2E-V-%d", suffix)
	var contractID, partyID, reminderID string
	contractURL := func() string { return vt + "/contracts/" + contractID }

	t.Run("manager creates a contract with the create dialog's body, member cannot", func(t *testing.T) {
		body := map[string]interface{}{
			"contract_number": number, "title": "E2E Wartungsvertrag", "contract_type": "service",
			"starts_on": day(-30), "ends_on": day(335), "notes": "Kündigungsfrist 90 Tage",
		}
		resp, b := postJSON(t, vt+"/contracts", body, memberAToken)
		requireStatus(t, resp, b, http.StatusForbidden)

		resp, b = postJSON(t, vt+"/contracts", body, managerToken)
		requireStatus(t, resp, b, http.StatusCreated)
		c := object(t, b, "contract")
		contractID = str(c, "id")
		if contractID == "" {
			t.Fatalf("expected the contract in {contract}, got %s", string(b))
		}
		if str(c, "status") != "draft" || str(c, "contract_number") != number || str(c, "contract_type") != "service" {
			t.Fatalf("unexpected created contract: %s", string(b))
		}

		// The number is unique per tenant.
		resp, b = postJSON(t, vt+"/contracts", body, managerToken)
		requireStatus(t, resp, b, http.StatusConflict)

		for name, bad := range map[string]map[string]interface{}{
			"no title":      {"contract_number": number + "-b", "title": "", "contract_type": "nda", "starts_on": day(0)},
			"no number":     {"title": "x", "contract_type": "nda", "starts_on": day(0)},
			"no type":       {"contract_number": number + "-c", "title": "x", "starts_on": day(0)},
			"malformed day": {"contract_number": number + "-d", "title": "x", "contract_type": "nda", "starts_on": "01.02.2026"},
		} {
			resp, b = postJSON(t, vt+"/contracts", bad, managerToken)
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("%s: expected 400, got %d: %s", name, resp.StatusCode, string(b))
			}
		}
	})

	t.Run("members read the contract, the list and the empty child lists", func(t *testing.T) {
		for _, token := range []string{memberAToken, memberBToken} {
			resp, b := getJSON(t, contractURL(), token)
			requireStatus(t, resp, b, http.StatusOK)
			c := object(t, b, "contract")
			if str(c, "id") != contractID || str(c, "title") != "E2E Wartungsvertrag" {
				t.Fatalf("unexpected contract for member: %s", string(b))
			}
			if !strings.HasPrefix(str(c, "starts_on"), day(-30)) {
				t.Fatalf("starts_on lost its day: %s", string(b))
			}

			rows, total := list(t, vt+"/contracts?status=draft&page_size=100", token, "contracts")
			found := false
			for _, r := range rows {
				if str(r, "id") == contractID {
					found = true
				}
			}
			if !found || total < 1 {
				t.Fatalf("expected the contract in the draft list (total %d)", total)
			}

			if parties, _ := list(t, contractURL()+"/parties", token, "parties"); len(parties) != 0 {
				t.Fatalf("expected no parties yet, got %d", len(parties))
			}
			if rems, _ := list(t, contractURL()+"/reminders", token, "reminders"); len(rems) != 0 {
				t.Fatalf("expected no reminders yet, got %d", len(rems))
			}
		}
		resp, b := getJSON(t, vt+"/contracts?status=draft&contract_type=bogus", memberAToken)
		if resp.StatusCode >= 500 {
			t.Fatalf("a filter value outside the enum must not be a server error, got %d: %s", resp.StatusCode, string(b))
		}
		resp, b = getJSON(t, vt+"/contracts/not-a-uuid", memberAToken)
		requireStatus(t, resp, b, http.StatusBadRequest)
		resp, b = getJSON(t, vt+"/contracts/00000000-0000-4000-8000-000000000000", memberAToken)
		requireStatus(t, resp, b, http.StatusNotFound)
	})

	t.Run("manager edits like the edit dialog and clears the end date", func(t *testing.T) {
		resp, b := patch(t, contractURL(), map[string]interface{}{
			"title": "E2E Wartungsvertrag 2", "contract_type": "rental", "starts_on": day(-30),
			"ends_on": day(400), "clear_ends_on": false, "notes": "Verlängert",
		}, managerToken)
		requireStatus(t, resp, b, http.StatusOK)
		c := object(t, b, "contract")
		if str(c, "title") != "E2E Wartungsvertrag 2" || str(c, "contract_type") != "rental" || !strings.HasPrefix(str(c, "ends_on"), day(400)) {
			t.Fatalf("edit not applied: %s", string(b))
		}

		// The dialog sends `clear_ends_on: true` and no end date when the field was emptied.
		resp, b = patch(t, contractURL(), map[string]interface{}{
			"title": "E2E Wartungsvertrag 2", "contract_type": "rental", "starts_on": day(-30), "clear_ends_on": true, "notes": "Unbefristet",
		}, managerToken)
		requireStatus(t, resp, b, http.StatusOK)
		c = object(t, b, "contract")
		if v, present := c["ends_on"]; present && v != nil {
			t.Fatalf("expected ends_on cleared, got %v", v)
		}

		// uploadDocument() patches only the object key.
		resp, b = patch(t, contractURL(), map[string]interface{}{"document_url": "vertraege/e2e/" + number + ".pdf"}, managerToken)
		requireStatus(t, resp, b, http.StatusOK)
		c = object(t, b, "contract")
		if str(c, "document_url") != "vertraege/e2e/"+number+".pdf" || str(c, "title") != "E2E Wartungsvertrag 2" {
			t.Fatalf("a document-only patch must keep the other fields: %s", string(b))
		}

		resp, b = patch(t, contractURL(), map[string]interface{}{"title": "   "}, managerToken)
		requireStatus(t, resp, b, http.StatusBadRequest)
		resp, b = patch(t, contractURL(), map[string]interface{}{"ends_on": "tomorrow"}, managerToken)
		requireStatus(t, resp, b, http.StatusBadRequest)
	})

	// A value outside an enum is an input error (400), an unknown contract is 404, an unknown contact
	// is 400. The CHECK constraints and foreign keys never see them.
	t.Run("values outside the enums and unknown references are refused as input errors", func(t *testing.T) {
		unknown := vt + "/contracts/00000000-0000-4000-8000-000000000000"
		for name, c := range map[string]struct {
			method, url string
			body        map[string]interface{}
			want        int
		}{
			"patch status":      {http.MethodPatch, contractURL(), map[string]interface{}{"status": "bogus"}, http.StatusBadRequest},
			"patch type":        {http.MethodPatch, contractURL(), map[string]interface{}{"contract_type": "bogus"}, http.StatusBadRequest},
			"create type":       {http.MethodPost, vt + "/contracts", map[string]interface{}{"contract_number": number + "-p", "title": "x", "contract_type": "bogus", "starts_on": day(0)}, http.StatusBadRequest},
			"party type":        {http.MethodPost, contractURL() + "/parties", map[string]interface{}{"party_type": "bogus", "external_name": "x", "role_in_contract": "client"}, http.StatusBadRequest},
			"party contact":     {http.MethodPost, contractURL() + "/parties", map[string]interface{}{"party_type": "contact", "contact_id": "00000000-0000-4000-8000-000000000000", "role_in_contract": "client"}, http.StatusBadRequest},
			"party contract":    {http.MethodPost, unknown + "/parties", map[string]interface{}{"party_type": "external", "external_name": "x", "role_in_contract": "client"}, http.StatusNotFound},
			"reminder type":     {http.MethodPost, contractURL() + "/reminders", map[string]interface{}{"remind_at": day(30), "reminder_type": "bogus", "subject": "x"}, http.StatusBadRequest},
			"reminder no type":  {http.MethodPost, contractURL() + "/reminders", map[string]interface{}{"remind_at": day(30), "subject": "x"}, http.StatusBadRequest},
			"reminder contract": {http.MethodPost, unknown + "/reminders", map[string]interface{}{"remind_at": day(30), "reminder_type": "custom", "subject": "x"}, http.StatusNotFound},
		} {
			resp, b := doRequest(t, c.method, c.url, c.body, managerToken)
			if resp.StatusCode != c.want {
				t.Errorf("%s: expected %d, got %d: %s", name, c.want, resp.StatusCode, string(b))
			}
		}
		// The refused patches changed nothing.
		resp, b := getJSON(t, contractURL(), memberAToken)
		requireStatus(t, resp, b, http.StatusOK)
		if c := object(t, b, "contract"); str(c, "status") != "draft" || str(c, "contract_type") != "rental" {
			t.Fatalf("a refused patch changed the row: %s", string(b))
		}
	})

	t.Run("member writes are all refused and leave the row untouched", func(t *testing.T) {
		for _, token := range []string{memberAToken, memberBToken} {
			checks := []struct {
				name string
				do   func() (*http.Response, []byte)
			}{
				{"patch", func() (*http.Response, []byte) {
					return patch(t, contractURL(), map[string]interface{}{"title": "gehackt"}, token)
				}},
				{"terminate", func() (*http.Response, []byte) {
					return patch(t, contractURL(), map[string]interface{}{"status": "terminated"}, token)
				}},
				{"delete", func() (*http.Response, []byte) { return deleteJSON(t, contractURL(), token) }},
				{"signature", func() (*http.Response, []byte) {
					return putJSON(t, contractURL()+"/signature", map[string]interface{}{"signature_data": "data:image/png;base64,AAAA", "signed_by": "x"}, token)
				}},
				{"add party", func() (*http.Response, []byte) {
					return postJSON(t, contractURL()+"/parties", map[string]interface{}{"party_type": "external", "external_name": "x", "role_in_contract": "client"}, token)
				}},
				{"add reminder", func() (*http.Response, []byte) {
					return postJSON(t, contractURL()+"/reminders", map[string]interface{}{"remind_at": day(30), "reminder_type": "renewal", "subject": "x"}, token)
				}},
			}
			for _, c := range checks {
				resp, b := c.do()
				if resp.StatusCode != http.StatusForbidden {
					t.Fatalf("member %s: expected 403, got %d: %s", c.name, resp.StatusCode, string(b))
				}
			}
		}
		resp, b := getJSON(t, contractURL(), memberAToken)
		requireStatus(t, resp, b, http.StatusOK)
		c := object(t, b, "contract")
		if str(c, "title") != "E2E Wartungsvertrag 2" || str(c, "status") != "draft" {
			t.Fatalf("a refused write changed the row: %s", string(b))
		}
	})

	t.Run("manager attaches a party and a reminder, members read them", func(t *testing.T) {
		resp, b := postJSON(t, contractURL()+"/parties", map[string]interface{}{
			"party_type": "external", "external_name": "Muster GmbH", "role_in_contract": "client",
		}, managerToken)
		requireStatus(t, resp, b, http.StatusCreated)
		p := object(t, b, "party")
		partyID = str(p, "id")
		if partyID == "" || str(p, "external_name") != "Muster GmbH" {
			t.Fatalf("expected the party in {party}, got %s", string(b))
		}
		resp, b = postJSON(t, contractURL()+"/parties", map[string]interface{}{"party_type": "external", "role_in_contract": "client"}, managerToken)
		requireStatus(t, resp, b, http.StatusBadRequest)
		resp, b = postJSON(t, contractURL()+"/parties", map[string]interface{}{"party_type": "contact", "contact_id": "nope", "role_in_contract": "client"}, managerToken)
		requireStatus(t, resp, b, http.StatusBadRequest)

		resp, b = postJSON(t, contractURL()+"/reminders", map[string]interface{}{
			"remind_at": day(300), "reminder_type": "renewal", "subject": "Verlängerung prüfen", "message": "90 Tage vorher",
		}, managerToken)
		requireStatus(t, resp, b, http.StatusCreated)
		r := object(t, b, "reminder")
		reminderID = str(r, "id")
		if reminderID == "" || str(r, "status") != "pending" || str(r, "subject") != "Verlängerung prüfen" {
			t.Fatalf("expected a pending reminder in {reminder}, got %s", string(b))
		}
		resp, b = postJSON(t, contractURL()+"/reminders", map[string]interface{}{"reminder_type": "renewal", "subject": "ohne Datum"}, managerToken)
		requireStatus(t, resp, b, http.StatusBadRequest)
		resp, b = postJSON(t, contractURL()+"/reminders", map[string]interface{}{"remind_at": day(30), "reminder_type": "renewal"}, managerToken)
		requireStatus(t, resp, b, http.StatusBadRequest)

		for _, token := range []string{memberAToken, memberBToken} {
			if parties, _ := list(t, contractURL()+"/parties", token, "parties"); len(parties) != 1 || str(parties[0], "id") != partyID {
				t.Fatalf("expected the one party for member, got %v", parties)
			}
			if rems, _ := list(t, contractURL()+"/reminders?only_pending=true", token, "reminders"); len(rems) != 1 || str(rems[0], "id") != reminderID {
				t.Fatalf("expected the one pending reminder for member, got %v", rems)
			}
		}

		// Cancelling the reminder drops it from the pending list but not from the full one.
		resp, b = patch(t, contractURL()+"/reminders/"+reminderID, map[string]interface{}{"status": "cancelled"}, managerToken)
		requireStatus(t, resp, b, http.StatusOK)
		if rems, _ := list(t, contractURL()+"/reminders?only_pending=true", memberAToken, "reminders"); len(rems) != 0 {
			t.Fatalf("a cancelled reminder must leave the pending list, got %v", rems)
		}
		if rems, _ := list(t, contractURL()+"/reminders", memberAToken, "reminders"); len(rems) != 1 {
			t.Fatalf("expected the cancelled reminder in the full list, got %v", rems)
		}

		// The two child writes of a member are refused at the item level too.
		resp, b = patch(t, contractURL()+"/reminders/"+reminderID, map[string]interface{}{"status": "pending"}, memberAToken)
		requireStatus(t, resp, b, http.StatusForbidden)
		resp, b = deleteJSON(t, contractURL()+"/reminders/"+reminderID, memberAToken)
		requireStatus(t, resp, b, http.StatusForbidden)
		resp, b = deleteJSON(t, contractURL()+"/parties/"+partyID, memberAToken)
		requireStatus(t, resp, b, http.StatusForbidden)
		if parties, _ := list(t, contractURL()+"/parties", memberAToken, "parties"); len(parties) != 1 {
			t.Fatalf("a refused party delete removed the row")
		}

		resp, b = deleteJSON(t, contractURL()+"/reminders/"+reminderID, managerToken)
		requireStatus(t, resp, b, http.StatusNoContent)
		resp, b = deleteJSON(t, contractURL()+"/parties/"+partyID, managerToken)
		requireStatus(t, resp, b, http.StatusNoContent)
		if parties, _ := list(t, contractURL()+"/parties", memberAToken, "parties"); len(parties) != 0 {
			t.Fatalf("expected the party gone, got %v", parties)
		}
	})

	t.Run("member exports the contract and reads its audit trail", func(t *testing.T) {
		resp, b := getJSON(t, contractURL()+"/export", memberBToken)
		requireStatus(t, resp, b, http.StatusOK)
		if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
			t.Fatalf("expected an attachment, got Content-Disposition %q", cd)
		}
		if len(b) == 0 {
			t.Fatal("expected a non-empty export")
		}

		events, total := list(t, contractURL()+"/events", memberBToken, "items")
		seen := map[string]int{}
		for _, e := range events {
			seen[str(e, "action")]++
		}
		for _, want := range []string{"created", "updated", "party_added", "party_removed"} {
			if seen[want] == 0 {
				t.Fatalf("expected a %q entry in the trail, got %v (total %d)", want, seen, total)
			}
		}
		resp, b = getJSON(t, vt+"/contracts/00000000-0000-4000-8000-000000000000/events", memberBToken)
		requireStatus(t, resp, b, http.StatusNotFound)
	})

	t.Run("signature route needs the write permission and a valid image", func(t *testing.T) {
		// A one-pixel PNG.
		const png = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="
		resp, b := putJSON(t, contractURL()+"/signature", map[string]interface{}{"signature_data": png, "signed_by": "Erika Muster"}, memberAToken)
		requireStatus(t, resp, b, http.StatusForbidden)
		resp, b = putJSON(t, contractURL()+"/signature", map[string]interface{}{"signature_data": "data:text/html;base64,AAAA", "signed_by": "Erika Muster"}, managerToken)
		requireStatus(t, resp, b, http.StatusBadRequest)
		resp, b = putJSON(t, contractURL()+"/signature", map[string]interface{}{"signature_data": png}, managerToken)
		requireStatus(t, resp, b, http.StatusBadRequest)

		resp, b = putJSON(t, contractURL()+"/signature", map[string]interface{}{"signature_data": png, "signed_by": "Erika Muster"}, managerToken)
		requireStatus(t, resp, b, http.StatusOK)
		c := object(t, b, "contract")
		if str(c, "signed_by") != "Erika Muster" || str(c, "signed_at") == "" {
			t.Fatalf("expected signed_by and signed_at on the contract, got %s", string(b))
		}
		resp, b = putJSON(t, contractURL()+"/signature", map[string]interface{}{"signature_data": png, "signed_by": "Zweiter"}, managerToken)
		requireStatus(t, resp, b, http.StatusConflict)
	})

	t.Run("terminating files one entry, a draft can be deleted and a terminated contract cannot", func(t *testing.T) {
		// The termination dialog sends only the status.
		resp, b := patch(t, contractURL(), map[string]interface{}{"status": "terminated"}, managerToken)
		requireStatus(t, resp, b, http.StatusOK)
		if c := object(t, b, "contract"); str(c, "status") != "terminated" {
			t.Fatalf("expected terminated, got %s", string(b))
		}
		// Saving it again does not file a second termination.
		resp, b = patch(t, contractURL(), map[string]interface{}{"notes": "Gekündigt zum Jahresende"}, managerToken)
		requireStatus(t, resp, b, http.StatusOK)

		events, _ := list(t, contractURL()+"/events?page_size=100", memberAToken, "items")
		terminated, signed := 0, 0
		for _, e := range events {
			switch str(e, "action") {
			case "terminated":
				terminated++
			case "signed":
				signed++
			}
		}
		if terminated != 1 || signed != 1 {
			t.Fatalf("expected one terminated and one signed entry, got %d/%d", terminated, signed)
		}

		resp, b = deleteJSON(t, contractURL(), managerToken)
		requireStatus(t, resp, b, http.StatusConflict)
		resp, b = getJSON(t, contractURL(), memberAToken)
		requireStatus(t, resp, b, http.StatusOK)

		// A fresh draft goes.
		resp, b = postJSON(t, vt+"/contracts", map[string]interface{}{
			"contract_number": number + "-draft", "title": "E2E Entwurf", "contract_type": "nda", "starts_on": day(0),
		}, managerToken)
		requireStatus(t, resp, b, http.StatusCreated)
		draftID := str(object(t, b, "contract"), "id")
		resp, b = deleteJSON(t, vt+"/contracts/"+draftID, memberAToken)
		requireStatus(t, resp, b, http.StatusForbidden)
		resp, b = deleteJSON(t, vt+"/contracts/"+draftID, managerToken)
		requireStatus(t, resp, b, http.StatusNoContent)
		resp, b = getJSON(t, vt+"/contracts/"+draftID, memberAToken)
		requireStatus(t, resp, b, http.StatusNotFound)
		resp, b = deleteJSON(t, vt+"/contracts/"+draftID, managerToken)
		requireStatus(t, resp, b, http.StatusNotFound)
	})

	t.Run("the desktop list call without page_size sees the default page and the total", func(t *testing.T) {
		const extra = 22 // above the default page size of 20
		for i := 0; i < extra; i++ {
			resp, b := postJSON(t, vt+"/contracts", map[string]interface{}{
				"contract_number": fmt.Sprintf("%s-L%02d", number, i), "title": fmt.Sprintf("E2E Liste %02d", i),
				"contract_type": "other", "starts_on": day(0),
			}, managerToken)
			requireStatus(t, resp, b, http.StatusCreated)
		}
		first, total := list(t, vt+"/contracts", memberAToken, "contracts")
		if len(first) != 20 || total < extra+1 {
			t.Fatalf("expected the default page of 20 with a total of at least %d, got %d rows and total %d", extra+1, len(first), total)
		}
		second, _ := list(t, vt+"/contracts?page=2", memberAToken, "contracts")
		if len(second) == 0 {
			t.Fatal("expected a second page")
		}
		seen := map[string]bool{}
		for _, r := range first {
			seen[str(r, "id")] = true
		}
		for _, r := range second {
			if seen[str(r, "id")] {
				t.Fatalf("contract %s is on both pages", str(r, "id"))
			}
		}
		// A page size above the gateway ceiling is clamped, not refused.
		big, _ := list(t, vt+"/contracts?page_size=500", memberAToken, "contracts")
		if len(big) > 100 || len(big) <= 20 {
			t.Fatalf("expected a page between 21 and 100 rows for page_size=500, got %d", len(big))
		}
	})
}
