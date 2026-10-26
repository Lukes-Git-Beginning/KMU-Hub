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

// TestVermietungMemberInspectionFlow runs the rental module the way the roles meet it against the
// real seed: the manager puts a rental object into the catalog and books the first rental, a member
// takes a second rental on the same object, runs the handover and the return with the state
// reports (Zustandsprotokoll) and finishes it. The bodies are the ones the desktop sends
// (vermietung-client.ts / ZustandsprotokollDialog.tsx: kind, notes, photo_urls, no checklist, no
// performed_by). Needs the vermietung service in the stack.
//
// IST of the seed that the test pins and does not judge:
//   - member holds object:read, rental:create|edit|handover|read, inspection:create|read (migrations
//     256 and 339); manager adds object:create|edit|delete and rental:cancel;
//   - vermietung has no own-scope: rentals and inspections are tenant-wide, a second member reads
//     and drives the first one's rental (including start/end) and files the other inspection kind;
//   - PATCH /inspections/{id} needs the legacy inspection:write, which only admin holds: a state
//     report is write-once for member and manager (403), the desktop has no editor for it;
//   - DELETE /rentals/{id} (cancel) is manager and admin, a member 403; PUT /signature needs the
//     legacy rental:write, admin only, and has no desktop caller;
//   - the body's performed_by is stored as sent, the gateway does not rewrite it to the caller.
func TestVermietungMemberInspectionFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	adminToken, _ := registerAndLoginWithRole(t, base, "admin")
	managerToken, _ := registerAndLoginWithRole(t, base, "manager")
	memberAToken, memberAID := registerAndLoginWithRole(t, base, "member")
	memberBToken, _ := registerAndLoginWithRole(t, base, "member")

	vm := base + "/api/v1/vermietung"
	suffix := time.Now().UnixNano()

	// Windows far in the future on full days; the object is new, so no other flow shares it.
	day0 := time.Now().UTC().AddDate(0, 0, 30).Truncate(24 * time.Hour)
	stamp := func(days int) string { return day0.AddDate(0, 0, days).Format(time.RFC3339) }

	var objectID, managerRental, memberRental, handoverID, returnID string

	// ids decodes a list route and returns the row ids. An empty list may come as `{}`, `null` is the
	// break the desktop's `?? []` would not survive and is refused.
	ids := func(t *testing.T, url, token, key string) map[string]bool {
		t.Helper()
		resp, body := getJSON(t, url, token)
		requireStatus(t, resp, body, http.StatusOK)
		if strings.TrimSpace(string(body)) == "null" {
			t.Fatalf("a list must never be null, got %s", string(body))
		}
		var raw map[string]json.RawMessage
		decodeBody(t, body, &raw)
		var rows []map[string]interface{}
		if rawRows, ok := raw[key]; ok {
			decodeBody(t, rawRows, &rows)
		}
		out := map[string]bool{}
		for _, r := range rows {
			if id, _ := r["id"].(string); id != "" {
				out[id] = true
			}
		}
		return out
	}
	rentalBody := func(from, to int, renter string) map[string]interface{} {
		return map[string]interface{}{
			"object_id": objectID, "renter_name": renter,
			"start_date": stamp(from), "end_date": stamp(to),
			"total_price": 240.0, "deposit_paid": false,
		}
	}
	createRental := func(t *testing.T, token string, body map[string]interface{}, want int) string {
		t.Helper()
		resp, raw := postJSON(t, vm+"/rentals", body, token)
		requireStatus(t, resp, raw, want)
		if want != http.StatusCreated {
			return ""
		}
		var out struct {
			Rental map[string]interface{} `json:"rental"`
		}
		decodeBody(t, raw, &out)
		id, _ := out.Rental["id"].(string)
		if id == "" || out.Rental["status"] != "reserved" || out.Rental["object_id"] != objectID {
			t.Fatalf("expected a reserved rental on the object in {rental}, got %s", string(raw))
		}
		return id
	}
	rentalStatus := func(t *testing.T, token, id string) string {
		t.Helper()
		resp, raw := getJSON(t, vm+"/rentals/"+id, token)
		requireStatus(t, resp, raw, http.StatusOK)
		var out struct {
			Rental map[string]interface{} `json:"rental"`
		}
		decodeBody(t, raw, &out)
		s, _ := out.Rental["status"].(string)
		return s
	}
	createInspection := func(t *testing.T, token, rentalID string, body map[string]interface{}, want int) (string, map[string]interface{}) {
		t.Helper()
		resp, raw := postJSON(t, vm+"/rentals/"+rentalID+"/inspections", body, token)
		requireStatus(t, resp, raw, want)
		if want != http.StatusCreated {
			return "", nil
		}
		var out struct {
			Inspection map[string]interface{} `json:"inspection"`
		}
		decodeBody(t, raw, &out)
		id, _ := out.Inspection["id"].(string)
		if id == "" || out.Inspection["rental_id"] != rentalID || out.Inspection["kind"] != body["kind"] {
			t.Fatalf("expected the inspection of the rental in {inspection}, got %s", string(raw))
		}
		return id, out.Inspection
	}

	t.Run("manager puts the object into the catalog, member only reads", func(t *testing.T) {
		resp, body := postJSON(t, vm+"/objects", map[string]interface{}{
			"name": fmt.Sprintf("E2E Anhänger %d", suffix), "category": "trailer",
			"daily_rate": 80.0, "deposit": 150.0, "location": "Halle 2",
		}, managerToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var out struct {
			Object map[string]interface{} `json:"object"`
		}
		decodeBody(t, body, &out)
		objectID, _ = out.Object["id"].(string)
		if objectID == "" {
			t.Fatalf("expected the object in {object}, got %s", string(body))
		}

		resp, body = postJSON(t, vm+"/objects", map[string]interface{}{
			"name": "E2E Member Objekt", "category": "trailer", "daily_rate": 10.0, "deposit": 0.0,
		}, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)
		resp, body = doRequest(t, http.MethodPatch, vm+"/objects/"+objectID, map[string]interface{}{"active": false}, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)
		resp, body = deleteJSON(t, vm+"/objects/"+objectID, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)

		resp, body = getJSON(t, vm+"/objects/"+objectID, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		if !ids(t, vm+"/objects?page_size=200&active_only=true", memberAToken, "objects")[objectID] {
			t.Fatalf("member A must see the new object in the list")
		}
	})

	t.Run("manager books the first rental, member books a free window, overlaps are 409", func(t *testing.T) {
		managerRental = createRental(t, managerToken, rentalBody(0, 3, "E2E Kunde Manager"), http.StatusCreated)

		// Member A takes the window after it. The desktop sends the same body shape as the manager.
		memberRental = createRental(t, memberAToken, rentalBody(5, 8, "E2E Kunde Member"), http.StatusCreated)

		// Any overlap with a live rental is refused, whoever asks.
		createRental(t, memberBToken, rentalBody(1, 2, "E2E Doppelt"), http.StatusConflict)
		createRental(t, memberBToken, rentalBody(7, 10, "E2E Doppelt"), http.StatusConflict)
		createRental(t, managerToken, rentalBody(-1, 1, "E2E Doppelt"), http.StatusConflict)

		// End before start is refused at the boundary, an unknown object is a 404.
		createRental(t, memberBToken, rentalBody(20, 19, "E2E Rückwärts"), http.StatusBadRequest)
		unknown := rentalBody(20, 22, "E2E Unbekannt")
		unknown["object_id"] = "00000000-0000-4000-8000-000000000000"
		createRental(t, memberBToken, unknown, http.StatusNotFound)

		// The availability probe the booking form runs agrees with the 409s.
		resp, body := getJSON(t, vm+"/objects/"+objectID+"/availability?start_date="+
			stamp(5)+"&end_date="+stamp(8), memberBToken)
		requireStatus(t, resp, body, http.StatusOK)
		var av struct {
			Available bool `json:"available"`
		}
		decodeBody(t, body, &av)
		if av.Available {
			t.Fatalf("a window with a live rental must not be available, got %s", string(body))
		}
	})

	t.Run("both members read both rentals, tenant-wide", func(t *testing.T) {
		for _, token := range []string{memberAToken, memberBToken} {
			got := ids(t, vm+"/rentals?object_id="+objectID+"&page_size=200", token, "rentals")
			if !got[managerRental] || !got[memberRental] || len(got) != 2 {
				t.Fatalf("a member must see exactly the two rentals of the object, got %v", got)
			}
		}
	})

	t.Run("member files the handover report with the desktop body", func(t *testing.T) {
		var ins map[string]interface{}
		handoverID, ins = createInspection(t, memberAToken, memberRental, map[string]interface{}{
			"kind":       "handover",
			"notes":      "[by:Max Muster] Kratzer links\n[{\"label\":\"Reifen\",\"ok\":true}]",
			"photo_urls": []string{"vermietung/e2e/handover-1.jpg", "vermietung/e2e/handover-2.jpg"},
		}, http.StatusCreated)
		if photos, _ := ins["photo_urls"].([]interface{}); len(photos) != 2 {
			t.Fatalf("both photo keys must be stored, got %v", ins["photo_urls"])
		}

		// The same kind twice for one rental is a conflict, an unknown kind a 400, an unknown rental a 404.
		createInspection(t, memberAToken, memberRental, map[string]interface{}{"kind": "handover", "notes": "doppelt"}, http.StatusConflict)
		createInspection(t, memberAToken, memberRental, map[string]interface{}{"kind": "inspection", "notes": "x"}, http.StatusBadRequest)
		createInspection(t, memberAToken, "00000000-0000-4000-8000-000000000000",
			map[string]interface{}{"kind": "handover", "notes": "x"}, http.StatusNotFound)

		resp, body := getJSON(t, vm+"/inspections/"+handoverID, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		if !ids(t, vm+"/rentals/"+memberRental+"/inspections", memberAToken, "inspections")[handoverID] {
			t.Fatalf("the rental's inspection list must hold the report")
		}
		if got := ids(t, vm+"/rentals/"+managerRental+"/inspections", memberAToken, "inspections"); len(got) != 0 {
			t.Fatalf("the other rental has no report yet, got %v", got)
		}
	})

	t.Run("a state report is write-once for member and manager, admin corrects it", func(t *testing.T) {
		patch := map[string]interface{}{"notes": "[by:Max Muster] korrigiert"}
		for name, token := range map[string]string{"member": memberAToken, "manager": managerToken} {
			resp, body := doRequest(t, http.MethodPatch, vm+"/inspections/"+handoverID, patch, token)
			if resp.StatusCode != http.StatusForbidden {
				t.Fatalf("%s must not edit a state report, got %d %s", name, resp.StatusCode, string(body))
			}
		}
		resp, body := getJSON(t, vm+"/inspections/"+handoverID, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		if strings.Contains(string(body), "korrigiert") {
			t.Fatalf("a refused edit must leave the report unchanged, got %s", string(body))
		}

		resp, body = doRequest(t, http.MethodPatch, vm+"/inspections/"+handoverID, patch, adminToken)
		requireStatus(t, resp, body, http.StatusOK)
		resp, body = getJSON(t, vm+"/inspections/"+handoverID, memberBToken)
		requireStatus(t, resp, body, http.StatusOK)
		if !strings.Contains(string(body), "korrigiert") {
			t.Fatalf("the admin correction must be visible to the other member, got %s", string(body))
		}
	})

	t.Run("the other member reads the report and files the return kind", func(t *testing.T) {
		// IST: no own-scope, B reads A's report and may file the other kind of A's rental.
		resp, body := getJSON(t, vm+"/inspections/"+handoverID, memberBToken)
		requireStatus(t, resp, body, http.StatusOK)

		// performed_by is stored as sent: B names member A, the gateway does not rewrite it.
		var ins map[string]interface{}
		returnID, ins = createInspection(t, memberBToken, memberRental, map[string]interface{}{
			"kind": "return", "notes": "[by:Erika Muster] ohne Mängel", "performed_by": memberAID,
		}, http.StatusCreated)
		if ins["performed_by"] != memberAID {
			t.Fatalf("IST: performed_by is stored as sent (%s), got %v", memberAID, ins["performed_by"])
		}
		// A malformed performed_by never reaches the service.
		createInspection(t, memberBToken, managerRental, map[string]interface{}{
			"kind": "return", "notes": "x", "performed_by": "kein-uuid",
		}, http.StatusBadRequest)
	})

	t.Run("member runs the rental through start and end, the order is enforced", func(t *testing.T) {
		resp, body := postJSON(t, vm+"/rentals/"+memberRental+"/end", nil, memberAToken)
		requireStatus(t, resp, body, http.StatusConflict) // reserved cannot be ended
		if got := rentalStatus(t, memberAToken, memberRental); got != "reserved" {
			t.Fatalf("a refused end must leave the rental reserved, got %q", got)
		}

		resp, body = postJSON(t, vm+"/rentals/"+memberRental+"/start", nil, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		if got := rentalStatus(t, memberBToken, memberRental); got != "active" {
			t.Fatalf("start must make the rental active for every reader, got %q", got)
		}
		resp, body = postJSON(t, vm+"/rentals/"+memberRental+"/start", nil, memberAToken)
		requireStatus(t, resp, body, http.StatusConflict) // already active

		resp, body = postJSON(t, vm+"/rentals/"+memberRental+"/end", nil, memberBToken)
		requireStatus(t, resp, body, http.StatusOK)
		if got := rentalStatus(t, memberAToken, memberRental); got != "completed" {
			t.Fatalf("end must complete the rental, got %q", got)
		}
		resp, body = postJSON(t, vm+"/rentals/"+memberRental+"/end", nil, memberAToken)
		requireStatus(t, resp, body, http.StatusConflict) // already completed
	})

	t.Run("member edits the booking, cancel and signature stay with the higher roles", func(t *testing.T) {
		resp, body := doRequest(t, http.MethodPatch, vm+"/rentals/"+managerRental,
			map[string]interface{}{"notes": "E2E Notiz vom Member", "deposit_paid": true}, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		resp, body = getJSON(t, vm+"/rentals/"+managerRental, memberBToken)
		requireStatus(t, resp, body, http.StatusOK)
		if !strings.Contains(string(body), "E2E Notiz vom Member") {
			t.Fatalf("the member edit must be stored, got %s", string(body))
		}

		// Moving the window onto the completed rental's days is an overlap, an inverted window a 400.
		resp, body = doRequest(t, http.MethodPatch, vm+"/rentals/"+managerRental,
			map[string]interface{}{"start_date": stamp(6), "end_date": stamp(7)}, memberAToken)
		requireStatus(t, resp, body, http.StatusConflict)
		resp, body = doRequest(t, http.MethodPatch, vm+"/rentals/"+managerRental,
			map[string]interface{}{"start_date": stamp(3), "end_date": stamp(0)}, memberAToken)
		requireStatus(t, resp, body, http.StatusBadRequest)

		sig := map[string]interface{}{
			"signature_data": "data:image/png;base64,iVBORw0KGgo=", "signed_by": "E2E Kunde Manager",
		}
		for name, token := range map[string]string{"member": memberAToken, "manager": managerToken} {
			resp, body = putJSON(t, vm+"/rentals/"+managerRental+"/signature", sig, token)
			if resp.StatusCode != http.StatusForbidden {
				t.Fatalf("IST: %s has no rental:write for the signature, got %d %s", name, resp.StatusCode, string(body))
			}
		}
		resp, body = deleteJSON(t, vm+"/rentals/"+managerRental, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)
		if got := rentalStatus(t, memberAToken, managerRental); got != "reserved" {
			t.Fatalf("a refused cancel must leave the rental reserved, got %q", got)
		}

		resp, body = deleteJSON(t, vm+"/rentals/"+managerRental, managerToken)
		requireStatus(t, resp, body, http.StatusNoContent)
		// The cancelled window is free again: member B takes it.
		createRental(t, memberBToken, rentalBody(0, 3, "E2E Kunde B"), http.StatusCreated)
	})

	t.Run("the reports outlive the finished rental and stay readable", func(t *testing.T) {
		got := ids(t, vm+"/rentals/"+memberRental+"/inspections", memberBToken, "inspections")
		if !got[handoverID] || !got[returnID] || len(got) != 2 {
			t.Fatalf("both reports must be listed for the completed rental, got %v", got)
		}
	})
}
