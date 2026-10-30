//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestFuhrparkTripLogPrivateMaskingFlow runs the Fahrtenbuch against the real seed: a member (the
// driver) files a business trip and private trips, and everybody but the driver must see a private
// trip only as date, odometer readings, distance and the label "Privatfahrt" -- in the tenant list,
// the vehicle list and the CSV export (decision 2026-10-25). Bodies follow CreateTripLogRequest in
// desktop/src/renderer/src/api/fuhrpark-client.ts, which has no driver_user_id field.
//
// IST of the seed that the test pins and does not judge:
//   - member and manager hold fuhrpark:trip:create|read, never :write: creating and reading is
//     open, PATCH/DELETE of a trip log is admin only (member and manager 403);
//   - the unmasking key is trip_logs.driver_user_id, which the desktop dialog never sends, so a
//     private trip the desktop files is masked for its own driver too (see the subtest that pins it
//     and fix-fuhrpark-trip-log-driver-defaults-to-caller);
//   - the list is tenant-wide at every role, there is no own scope on trip logs.
func TestFuhrparkTripLogPrivateMaskingFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	managerToken, _ := registerAndLoginWithRole(t, base, "manager")
	adminToken, _ := registerAndLoginWithRole(t, base, "admin")
	driverToken, driverID := registerAndLoginWithRole(t, base, "member")
	otherToken, _ := registerAndLoginWithRole(t, base, "member")

	fp := base + "/api/v1/fuhrpark"
	suffix := time.Now().UnixNano()
	today := time.Now().UTC().Format("2006-01-02")

	const (
		secretStart = "Geheimstrasse 1 Klinik"
		secretEnd   = "Geheimweg 9 Arzt"
		secretNote  = "privater Notizvermerk"
		bizStart    = "Werk Nord"
		bizEnd      = "Kunde Sued"
		bizPurpose  = "Kundenbesuch Angebot"
	)

	var vehicleID, bizID, privLinkedID, privDesktopID string

	type trip struct {
		ID            string `json:"id"`
		VehicleID     string `json:"vehicle_id"`
		Date          string `json:"date"`
		StartLocation string `json:"start_location"`
		EndLocation   string `json:"end_location"`
		Purpose       string `json:"purpose"`
		IsPrivate     bool   `json:"is_private"`
		DriverName    string `json:"driver_name"`
		Notes         string `json:"notes"`
		Partner       string `json:"business_partner"`
	}
	// tripList returns the vehicle's trips by id; the vehicle is new, so only this flow's rows are in it.
	tripList := func(t *testing.T, url, token string) map[string]trip {
		t.Helper()
		resp, body := getJSON(t, url, token)
		requireStatus(t, resp, body, http.StatusOK)
		if strings.TrimSpace(string(body)) == "null" {
			t.Fatalf("a list must never be null, got %s", string(body))
		}
		var out struct {
			TripLogs []trip `json:"trip_logs"`
		}
		decodeBody(t, body, &out)
		rows := map[string]trip{}
		for _, r := range out.TripLogs {
			rows[r.ID] = r
		}
		return rows
	}
	create := func(t *testing.T, token string, body map[string]interface{}, want int) string {
		t.Helper()
		resp, raw := postJSON(t, fp+"/vehicles/"+vehicleID+"/trip-logs", body, token)
		requireStatus(t, resp, raw, want)
		if want != http.StatusCreated {
			return ""
		}
		var out struct {
			TripLog trip `json:"trip_log"`
		}
		decodeBody(t, raw, &out)
		if out.TripLog.ID == "" {
			t.Fatalf("expected the trip in {trip_log}, got %s", string(raw))
		}
		return out.TripLog.ID
	}
	requireFull := func(t *testing.T, who string, rows map[string]trip, id, start, end, purpose string) {
		t.Helper()
		got, ok := rows[id]
		if !ok {
			t.Fatalf("%s must see trip %s, got %v", who, id, rows)
		}
		if got.StartLocation != start || got.EndLocation != end || got.Purpose != purpose {
			t.Fatalf("%s must see trip %s unmasked (%q -> %q, %q), got %+v", who, id, start, end, purpose, got)
		}
	}
	requireMasked := func(t *testing.T, who string, rows map[string]trip, id string) {
		t.Helper()
		got, ok := rows[id]
		if !ok {
			t.Fatalf("%s must still see the private trip %s (the logbook stays gap-free), got %v", who, id, rows)
		}
		if got.StartLocation != "" || got.EndLocation != "" || got.Notes != "" || got.Partner != "" || got.Purpose != "Privatfahrt" {
			t.Fatalf("%s must see trip %s only as the label Privatfahrt, got %+v", who, id, got)
		}
		if !got.IsPrivate || got.DriverName == "" {
			t.Fatalf("%s: a masked trip keeps is_private and the driver name, got %+v", who, got)
		}
	}

	t.Run("manager puts the vehicle into the fleet", func(t *testing.T) {
		resp, body := postJSON(t, fp+"/vehicles", map[string]interface{}{
			"license_plate": fmt.Sprintf("M-FB %d", suffix%100000),
			"make":          "Skoda",
			"model":         "Octavia",
			"year":          2024,
			"fuel_type":     "diesel",
			"mileage_km":    5000,
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
	})

	t.Run("driver files a business and two private trips", func(t *testing.T) {
		bizID = create(t, driverToken, map[string]interface{}{
			"vehicle_id": vehicleID, "date": today,
			"start_location": bizStart, "end_location": bizEnd, "purpose": bizPurpose,
			"start_km": 5000, "end_km": 5040, "is_private": false, "driver_name": "E2E Fahrer",
		}, http.StatusCreated)

		// The desktop body: no driver_user_id, no purpose on a private trip.
		privDesktopID = create(t, driverToken, map[string]interface{}{
			"vehicle_id": vehicleID, "date": today,
			"start_location": secretStart, "end_location": secretEnd, "purpose": "",
			"start_km": 5040, "end_km": 5070, "is_private": true, "driver_name": "E2E Fahrer",
			"notes": secretNote,
		}, http.StatusCreated)

		// A client that links the driver account (the key the masking unmasks by).
		privLinkedID = create(t, driverToken, map[string]interface{}{
			"vehicle_id": vehicleID, "date": today,
			"start_location": secretStart, "end_location": secretEnd, "purpose": "Arzt",
			"start_km": 5070, "end_km": 5090, "is_private": true, "driver_name": "E2E Fahrer",
			"notes": secretNote, "driver_user_id": driverID,
		}, http.StatusCreated)
	})

	t.Run("trip body validation at the boundary", func(t *testing.T) {
		// A business trip needs its purpose (§ 6 EStG), a private one does not.
		create(t, driverToken, map[string]interface{}{
			"vehicle_id": vehicleID, "date": today,
			"start_location": bizStart, "end_location": bizEnd, "purpose": "  ",
			"start_km": 5090, "end_km": 5100, "is_private": false, "driver_name": "E2E Fahrer",
		}, http.StatusBadRequest)
		// The odometer never runs backwards.
		create(t, driverToken, map[string]interface{}{
			"vehicle_id": vehicleID, "date": today,
			"start_location": bizStart, "end_location": bizEnd, "purpose": bizPurpose,
			"start_km": 5100, "end_km": 5090, "is_private": false, "driver_name": "E2E Fahrer",
		}, http.StatusBadRequest)
		// A driver account of nobody in this tenant is refused.
		create(t, driverToken, map[string]interface{}{
			"vehicle_id": vehicleID, "date": today,
			"start_location": bizStart, "end_location": bizEnd, "purpose": bizPurpose,
			"start_km": 5090, "end_km": 5100, "is_private": false, "driver_name": "E2E Fahrer",
			"driver_user_id": "00000000-0000-4000-8000-000000000000",
		}, http.StatusBadRequest)
		if got := tripList(t, fp+"/vehicles/"+vehicleID+"/trip-logs", driverToken); len(got) != 3 {
			t.Fatalf("a refused trip must leave no row, got %d rows", len(got))
		}
	})

	t.Run("the driver sees the business trip and the linked private trip in full", func(t *testing.T) {
		for name, url := range map[string]string{
			"vehicle list": fp + "/vehicles/" + vehicleID + "/trip-logs",
			"tenant list":  fp + "/trip-logs?vehicle_id=" + vehicleID,
		} {
			rows := tripList(t, url, driverToken)
			requireFull(t, "driver ("+name+")", rows, bizID, bizStart, bizEnd, bizPurpose)
			requireFull(t, "driver ("+name+")", rows, privLinkedID, secretStart, secretEnd, "Arzt")
			if rows[privLinkedID].Notes != secretNote {
				t.Fatalf("driver (%s) must see the own private note, got %+v", name, rows[privLinkedID])
			}
		}
	})

	t.Run("IST: a private trip filed through the desktop dialog is masked for its own driver", func(t *testing.T) {
		// The desktop sends no driver_user_id, so the trip has no linked account and the
		// masking treats the driver like a stranger. Pinned as IST; the fix is
		// fix-fuhrpark-trip-log-driver-defaults-to-caller. Flip this to requireFull then.
		rows := tripList(t, fp+"/vehicles/"+vehicleID+"/trip-logs", driverToken)
		requireMasked(t, "driver (desktop-filed private trip, IST)", rows, privDesktopID)
	})

	t.Run("everybody else sees private trips only as the label, business in full", func(t *testing.T) {
		for who, token := range map[string]string{"other member": otherToken, "manager": managerToken, "admin": adminToken} {
			for name, url := range map[string]string{
				"vehicle list": fp + "/vehicles/" + vehicleID + "/trip-logs",
				"tenant list":  fp + "/trip-logs?vehicle_id=" + vehicleID,
			} {
				label := who + " (" + name + ")"
				rows := tripList(t, url, token)
				if len(rows) != 3 {
					t.Fatalf("%s must see all three trips, got %d", label, len(rows))
				}
				requireFull(t, label, rows, bizID, bizStart, bizEnd, bizPurpose)
				requireMasked(t, label, rows, privLinkedID)
				requireMasked(t, label, rows, privDesktopID)
			}
		}
	})

	t.Run("export keeps every trip with its km and hides the private destinations", func(t *testing.T) {
		export := func(token string) string {
			resp, body := getJSON(t, fp+"/trip-logs/export?format=csv&vehicle_id="+vehicleID, token)
			requireStatus(t, resp, body, http.StatusOK)
			if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "csv") {
				t.Fatalf("expected a csv content type, got %q", ct)
			}
			return string(body)
		}
		for who, token := range map[string]string{"other member": otherToken, "manager": managerToken, "admin": adminToken} {
			csv := export(token)
			if strings.Contains(csv, secretStart) || strings.Contains(csv, secretEnd) || strings.Contains(csv, secretNote) {
				t.Fatalf("%s: the export leaks a private destination or note:\n%s", who, csv)
			}
			if got := strings.Count(csv, "Privatfahrt"); got != 2 {
				t.Fatalf("%s: expected both private trips as the label, got %d in\n%s", who, got, csv)
			}
			if !strings.Contains(csv, bizStart) || !strings.Contains(csv, bizPurpose) {
				t.Fatalf("%s: the business trip must stay in the export:\n%s", who, csv)
			}
			// Gap-free: three data rows plus the header, and the km of the private legs stay.
			if lines := strings.Count(strings.TrimSpace(csv), "\n"); lines != 3 {
				t.Fatalf("%s: expected header plus three rows, got %d line breaks in\n%s", who, lines, csv)
			}
			for _, km := range []string{"5040", "5070", "5090"} {
				if !strings.Contains(csv, km) {
					t.Fatalf("%s: odometer reading %s missing from the export:\n%s", who, km, csv)
				}
			}
		}
		// The driver's own export shows the linked private trip in full, the desktop-filed one masked.
		csv := export(driverToken)
		if !strings.Contains(csv, secretEnd) || strings.Count(csv, "Privatfahrt") != 1 {
			t.Fatalf("driver export: linked private trip in full, desktop-filed one masked, got\n%s", csv)
		}
		if resp, body := getJSON(t, fp+"/trip-logs/export?format=xml", driverToken); resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("an unknown export format must be 400, got %d: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("IST: changing or deleting a trip log is admin only", func(t *testing.T) {
		for who, token := range map[string]string{"driver": driverToken, "other member": otherToken, "manager": managerToken} {
			resp, body := doRequest(t, http.MethodPatch, fp+"/trip-logs/"+bizID, map[string]interface{}{"purpose": "umgeschrieben"}, token)
			requireStatus(t, resp, body, http.StatusForbidden)
			resp, body = deleteJSON(t, fp+"/trip-logs/"+privLinkedID, token)
			if resp.StatusCode != http.StatusForbidden {
				t.Fatalf("%s: delete of a trip log must be 403, got %d: %s", who, resp.StatusCode, string(body))
			}
		}
		rows := tripList(t, fp+"/vehicles/"+vehicleID+"/trip-logs", driverToken)
		requireFull(t, "driver after refused writes", rows, bizID, bizStart, bizEnd, bizPurpose)
		if _, ok := rows[privLinkedID]; !ok {
			t.Fatalf("the refused delete must leave the trip, got %v", rows)
		}
	})

	t.Run("admin corrects a trip with the full body, the answer is the stored row", func(t *testing.T) {
		resp, body := doRequest(t, http.MethodPatch, fp+"/trip-logs/"+bizID, map[string]interface{}{
			"date": today, "start_location": bizStart, "end_location": bizEnd, "purpose": "Kundenbesuch Nachtrag",
			"start_km": 5000, "end_km": 5040, "is_private": false, "driver_name": "E2E Fahrer",
		}, adminToken)
		requireStatus(t, resp, body, http.StatusOK)
		var out struct {
			TripLog trip `json:"trip_log"`
		}
		decodeBody(t, body, &out)
		if out.TripLog.ID != bizID || out.TripLog.Purpose != "Kundenbesuch Nachtrag" || out.TripLog.StartLocation != bizStart {
			t.Fatalf("expected the corrected trip in {trip_log}, got %s", string(body))
		}
		rows := tripList(t, fp+"/vehicles/"+vehicleID+"/trip-logs", otherToken)
		requireFull(t, "other member after the correction", rows, bizID, bizStart, bizEnd, "Kundenbesuch Nachtrag")
		resp, body = doRequest(t, http.MethodPatch, fp+"/trip-logs/"+bizID, map[string]interface{}{
			"start_km": 5100, "end_km": 5040,
		}, adminToken)
		requireStatus(t, resp, body, http.StatusBadRequest)
	})

	t.Run("a partial PATCH of a trip log keeps every field it does not send", func(t *testing.T) {
		id := create(t, driverToken, map[string]interface{}{
			"vehicle_id": vehicleID, "date": today,
			"start_location": secretStart, "end_location": secretEnd, "purpose": "",
			"start_km": 5100, "end_km": 5110, "is_private": true, "driver_name": "E2E Fahrer",
		}, http.StatusCreated)
		resp, body := doRequest(t, http.MethodPatch, fp+"/trip-logs/"+id, map[string]interface{}{"purpose": "nur der Zweck"}, adminToken)
		requireStatus(t, resp, body, http.StatusOK)
		var out struct {
			TripLog trip `json:"trip_log"`
		}
		decodeBody(t, body, &out)
		if out.TripLog.Purpose != "nur der Zweck" {
			t.Fatalf("the sent field must be stored, got %s", string(body))
		}
		if out.TripLog.StartLocation != secretStart || out.TripLog.EndLocation != secretEnd || !out.TripLog.IsPrivate ||
			out.TripLog.Date != today || out.TripLog.DriverName != "E2E Fahrer" {
			t.Fatalf("the partial PATCH must keep every unsent field, got %s", string(body))
		}
	})
}
