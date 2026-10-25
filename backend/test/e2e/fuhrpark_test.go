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

// TestFuhrparkMemberFlow runs the fleet module the way the roles meet it against the real seed:
// the manager puts a pool vehicle into the fleet, members book it, report a damage and file their
// own driver license check, and a second member must see and change none of it. The desktop has no
// caller for bookings and driver licenses (only the generated types.ts lists the routes), so those
// bodies come from the gateway request structs and the migration 339 comment; the damage body
// follows fuhrpark-types.ts (ReportDamageInput). Needs the fuhrpark service in the stack.
//
// IST of the seed that the test pins and does not judge:
//   - booking:read|write, license:read|write and damage:read are held by member AND manager at
//     scope own (migration 339): the gateway rewrites the body's user_id/driver_id/reported_by to
//     the caller and narrows every list to the caller, whatever the query names;
//   - a foreign booking/license is NotFound (404) rather than forbidden, so its existence does not
//     leak; damage update/resolve needs damage:write, which only admin holds (member 403);
//   - vehicle create is vehicle:manage (manager, admin), a member only reads vehicles;
//   - the manager is narrowed too: it does not see a member's damage in /damages, only admin does.
func TestFuhrparkMemberFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	adminToken, _ := registerAndLoginWithRole(t, base, "admin")
	managerToken, _ := registerAndLoginWithRole(t, base, "manager")
	memberAToken, memberAID := registerAndLoginWithRole(t, base, "member")
	memberBToken, memberBID := registerAndLoginWithRole(t, base, "member")

	fp := base + "/api/v1/fuhrpark"
	suffix := time.Now().UnixNano()

	// A window two days out on a full hour; the vehicle is new, so no other flow shares it.
	start := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Hour)
	window := func(fromHours, toHours int) map[string]interface{} {
		return map[string]interface{}{
			"starts_at": start.Add(time.Duration(fromHours) * time.Hour).Format(time.RFC3339),
			"ends_at":   start.Add(time.Duration(toHours) * time.Hour).Format(time.RFC3339),
		}
	}

	var vehicleID, bookingA, bookingB, damageID, licenseA string

	// list decodes a list route and returns the row ids. The service answers an empty list as `{}`
	// (protojson drops the empty repeated field) or `{"x":[]}`; both are fine for the desktop's
	// `?? []`, `null` is the break and is refused.
	list := func(t *testing.T, url, token, key string) []map[string]interface{} {
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
		return rows
	}
	has := func(rows []map[string]interface{}, id string) bool {
		for _, r := range rows {
			if r["id"] == id {
				return true
			}
		}
		return false
	}
	book := func(t *testing.T, token string, userID string, from, to int, want int) string {
		t.Helper()
		body := window(from, to)
		body["vehicle_id"] = vehicleID
		body["user_id"] = userID
		body["purpose"] = "E2E Kundentermin"
		resp, raw := postJSON(t, fp+"/bookings", body, token)
		requireStatus(t, resp, raw, want)
		if want != http.StatusCreated {
			return ""
		}
		var out struct {
			Booking map[string]interface{} `json:"booking"`
		}
		decodeBody(t, raw, &out)
		id, _ := out.Booking["id"].(string)
		if id == "" || out.Booking["status"] != "booked" {
			t.Fatalf("expected a booked booking in {booking}, got %s", string(raw))
		}
		return id
	}

	t.Run("manager puts a pool vehicle into the fleet, member only reads", func(t *testing.T) {
		resp, body := postJSON(t, fp+"/vehicles", map[string]interface{}{
			"license_plate": fmt.Sprintf("M-E2E %d", suffix%100000),
			"make":          "VW",
			"model":         "Caddy",
			"year":          2023,
			"fuel_type":     "diesel",
			"mileage_km":    12000,
		}, managerToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var out struct {
			Vehicle map[string]interface{} `json:"vehicle"`
		}
		decodeBody(t, body, &out)
		vehicleID, _ = out.Vehicle["id"].(string)
		if vehicleID == "" {
			t.Fatalf("expected the vehicle in {vehicle}, got %s", string(body))
		}

		resp, body = postJSON(t, fp+"/vehicles", map[string]interface{}{
			"license_plate": "M-E2E 0", "make": "VW", "model": "Up",
		}, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)

		resp, body = getJSON(t, fp+"/vehicles/"+vehicleID, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		if !has(list(t, fp+"/vehicles?page_size=200", memberAToken, "vehicles"), vehicleID) {
			t.Fatalf("member A must see the pool vehicle in the list")
		}
	})

	t.Run("member books the vehicle, the booking is theirs", func(t *testing.T) {
		// The body names member B as the user; the gateway must file it under the caller.
		bookingA = book(t, memberAToken, memberBID, 0, 4, http.StatusCreated)

		rows := list(t, fp+"/bookings?vehicle_id="+vehicleID, memberAToken, "bookings")
		if !has(rows, bookingA) {
			t.Fatalf("member A must see the own booking, got %v", rows)
		}
		for _, r := range rows {
			if r["id"] == bookingA && (r["user_id"] != memberAID || r["vehicle_id"] != vehicleID) {
				t.Fatalf("booking must carry the caller as user_id (%s), got %v", memberAID, r)
			}
		}
	})

	t.Run("double booking of the same window is 409, an adjacent one is not", func(t *testing.T) {
		book(t, memberAToken, memberAID, 0, 4, http.StatusConflict)  // identical window
		book(t, memberBToken, memberBID, 2, 6, http.StatusConflict)  // overlaps the tail
		book(t, memberBToken, memberBID, -1, 1, http.StatusConflict) // overlaps the head

		resp, body := postJSON(t, fp+"/bookings", map[string]interface{}{
			"vehicle_id": vehicleID, "user_id": memberBID,
			"starts_at": start.Add(4 * time.Hour).Format(time.RFC3339),
			"ends_at":   start.Add(2 * time.Hour).Format(time.RFC3339),
		}, memberBToken)
		requireStatus(t, resp, body, http.StatusBadRequest) // ends before it starts

		bookingB = book(t, memberBToken, memberBID, 4, 8, http.StatusCreated) // back to back
	})

	t.Run("second member sees and changes nothing of the first", func(t *testing.T) {
		// Whatever user_id the query names, member B is narrowed to B.
		rows := list(t, fp+"/bookings?vehicle_id="+vehicleID+"&user_id="+memberAID, memberBToken, "bookings")
		if has(rows, bookingA) || !has(rows, bookingB) {
			t.Fatalf("member B must see only the own booking, got %v", rows)
		}

		resp, body := putPatch(t, fp+"/bookings/"+bookingA, map[string]interface{}{"purpose": "hijacked"}, memberBToken)
		requireStatus(t, resp, body, http.StatusNotFound)
		resp, body = putPatch(t, fp+"/bookings/"+bookingA, map[string]interface{}{"status": "cancelled"}, memberBToken)
		requireStatus(t, resp, body, http.StatusNotFound)
		resp, body = deleteJSON(t, fp+"/bookings/"+bookingA, memberBToken)
		requireStatus(t, resp, body, http.StatusNotFound)

		for _, r := range list(t, fp+"/bookings?vehicle_id="+vehicleID, memberAToken, "bookings") {
			if r["id"] == bookingA && (r["status"] != "booked" || r["purpose"] != "E2E Kundentermin") {
				t.Fatalf("the foreign attempts must leave the booking untouched, got %v", r)
			}
		}
	})

	t.Run("member moves, cancels and frees the own booking", func(t *testing.T) {
		resp, body := putPatch(t, fp+"/bookings/"+bookingA, map[string]interface{}{"purpose": "E2E verschoben"}, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)

		// Moving into B's window collides; an unknown status is refused at the edge.
		resp, body = putPatch(t, fp+"/bookings/"+bookingA, window(3, 5), memberAToken)
		requireStatus(t, resp, body, http.StatusConflict)
		resp, body = putPatch(t, fp+"/bookings/"+bookingA, map[string]interface{}{"status": "teleported"}, memberAToken)
		requireStatus(t, resp, body, http.StatusBadRequest)

		resp, body = putPatch(t, fp+"/bookings/"+bookingA, map[string]interface{}{"status": "cancelled"}, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		// A cancelled booking frees its slot, so B can take the window A held.
		book(t, memberBToken, memberBID, 0, 4, http.StatusCreated)

		resp, body = deleteJSON(t, fp+"/bookings/"+bookingA, memberAToken)
		requireStatus(t, resp, body, http.StatusNoContent)
		resp, body = deleteJSON(t, fp+"/bookings/"+bookingA, memberAToken)
		requireStatus(t, resp, body, http.StatusNotFound)
	})

	t.Run("member reports a damage, only the reporter and admin see it", func(t *testing.T) {
		// ReportDamageInput: description, severity, reported_by, photo_keys, cost_cents, notes.
		// reported_by names member B; the gateway files it under the caller.
		resp, body := postJSON(t, fp+"/vehicles/"+vehicleID+"/damages", map[string]interface{}{
			"description": "Kratzer an der Fahrertür",
			"severity":    "minor",
			"reported_by": memberBID,
		}, memberAToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var out struct {
			Damage map[string]interface{} `json:"damage"`
		}
		decodeBody(t, body, &out)
		damageID, _ = out.Damage["id"].(string)
		if damageID == "" || out.Damage["status"] != "reported" || out.Damage["reported_by"] != memberAID {
			t.Fatalf("expected a reported damage filed under member A (%s), got %s", memberAID, string(body))
		}

		resp, body = postJSON(t, fp+"/vehicles/"+vehicleID+"/damages", map[string]interface{}{
			"description": "kaputt", "severity": "explosive",
		}, memberAToken)
		requireStatus(t, resp, body, http.StatusBadRequest)

		for name, url := range map[string]string{
			"top-level": fp + "/damages?vehicle_id=" + vehicleID,
			"vehicle":   fp + "/vehicles/" + vehicleID + "/damages",
		} {
			if !has(list(t, url, memberAToken, "damages"), damageID) {
				t.Fatalf("member A must see the own damage in the %s list", name)
			}
			if has(list(t, url, memberBToken, "damages"), damageID) {
				t.Fatalf("member B must not see a colleague's damage in the %s list", name)
			}
			if has(list(t, url, managerToken, "damages"), damageID) {
				t.Fatalf("IST: the manager is narrowed to own damages as well (%s list)", name)
			}
			if !has(list(t, url, adminToken, "damages"), damageID) {
				t.Fatalf("admin must see every damage in the %s list", name)
			}
		}
	})

	t.Run("damage update and resolve are admin only", func(t *testing.T) {
		for _, token := range []string{memberAToken, memberBToken, managerToken} {
			resp, body := putPatch(t, fp+"/damages/"+damageID, map[string]interface{}{"status": "in_repair"}, token)
			requireStatus(t, resp, body, http.StatusForbidden)
			resp, body = postJSON(t, fp+"/damages/"+damageID+"/resolve", map[string]interface{}{"notes": "x"}, token)
			requireStatus(t, resp, body, http.StatusForbidden)
		}
		resp, body := putPatch(t, fp+"/damages/"+damageID, map[string]interface{}{"status": "in_repair"}, adminToken)
		requireStatus(t, resp, body, http.StatusOK)
		resp, body = postJSON(t, fp+"/damages/"+damageID+"/resolve", map[string]interface{}{"notes": "repariert"}, adminToken)
		requireStatus(t, resp, body, http.StatusOK)

		for _, r := range list(t, fp+"/damages?vehicle_id="+vehicleID, memberAToken, "damages") {
			if r["id"] == damageID && r["status"] != "resolved" {
				t.Fatalf("the reporter must see the resolved status, got %v", r)
			}
		}
	})

	t.Run("member files the own driver license check", func(t *testing.T) {
		// driver_id names member B; the gateway files it under the caller.
		resp, body := postJSON(t, fp+"/driver-licenses", map[string]interface{}{
			"driver_id":           memberBID,
			"license_classes":     []string{"B", "BE"},
			"expiry_date":         "2040-06-30",
			"checked_at":          time.Now().UTC().Format("2006-01-02"),
			"next_check_due_date": time.Now().UTC().AddDate(0, 6, 0).Format("2006-01-02"),
			"notes":               "E2E Sichtkontrolle",
		}, memberAToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var out struct {
			License map[string]interface{} `json:"license"`
		}
		decodeBody(t, body, &out)
		licenseA, _ = out.License["id"].(string)
		if licenseA == "" || out.License["driver_id"] != memberAID {
			t.Fatalf("expected the license filed under member A (%s), got %s", memberAID, string(body))
		}

		// Both required fields are refused at the edge.
		resp, body = postJSON(t, fp+"/driver-licenses", map[string]interface{}{
			"driver_id": memberAID, "license_classes": []string{}, "next_check_due_date": "2027-01-01",
		}, memberAToken)
		requireStatus(t, resp, body, http.StatusBadRequest)
		resp, body = postJSON(t, fp+"/driver-licenses", map[string]interface{}{
			"driver_id": memberAID, "license_classes": []string{"B"},
		}, memberAToken)
		requireStatus(t, resp, body, http.StatusBadRequest)

		if !has(list(t, fp+"/driver-licenses", memberAToken, "licenses"), licenseA) {
			t.Fatalf("member A must see the own license check")
		}
		// B names A as the driver in the query and is narrowed to B anyway.
		if has(list(t, fp+"/driver-licenses?driver_id="+memberAID, memberBToken, "licenses"), licenseA) {
			t.Fatalf("member B must not see a colleague's license check")
		}
		if !has(list(t, fp+"/driver-licenses?driver_id="+memberAID, adminToken, "licenses"), licenseA) {
			t.Fatalf("admin must see the license check of a driver")
		}
	})

	t.Run("a foreign license check is neither changed nor deleted", func(t *testing.T) {
		update := map[string]interface{}{
			"license_classes":     []string{"B"},
			"next_check_due_date": time.Now().UTC().AddDate(1, 0, 0).Format("2006-01-02"),
			"notes":               "hijacked",
		}
		resp, body := putPatch(t, fp+"/driver-licenses/"+licenseA, update, memberBToken)
		requireStatus(t, resp, body, http.StatusNotFound)
		resp, body = deleteJSON(t, fp+"/driver-licenses/"+licenseA, memberBToken)
		requireStatus(t, resp, body, http.StatusNotFound)

		for _, r := range list(t, fp+"/driver-licenses", memberAToken, "licenses") {
			if r["id"] == licenseA && r["notes"] != "E2E Sichtkontrolle" {
				t.Fatalf("the foreign attempts must leave the license untouched, got %v", r)
			}
		}

		update["notes"] = "E2E nachgeprüft"
		resp, body = putPatch(t, fp+"/driver-licenses/"+licenseA, update, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = deleteJSON(t, fp+"/driver-licenses/"+licenseA, memberAToken)
		requireStatus(t, resp, body, http.StatusNoContent)
		if has(list(t, fp+"/driver-licenses", memberAToken, "licenses"), licenseA) {
			t.Fatalf("a deleted license check must be gone")
		}
	})
}
