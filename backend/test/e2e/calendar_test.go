//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestCalendarLifecycleFlow walks the calendar paths that returned 403 for every
// real user until ea5a39a9, because calendar_grpc.go passed uuid.Nil as the actor
// ("Gateway handles auth"): update/delete calendar, add/remove member, update/delete
// event, set reminders and delete category. Each of those checks the caller against
// the owner or the event creator, so an owner-only flow through the gateway is the
// one that fails with a nil actor.
func TestCalendarLifecycleFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	token, userID := registerAndLoginAdmin(t, base)
	_, memberID := registerAndLogin(t, base)

	suffix := time.Now().UnixNano()
	calName := fmt.Sprintf("E2E Calendar %d", suffix)

	var calendarID, categoryID, eventID string

	t.Run("create_calendar", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/calendar/calendars", map[string]interface{}{
			"name":          calName,
			"calendar_type": "shared",
			"color":         "#112233",
		}, token)
		requireStatus(t, resp, body, http.StatusCreated)

		var result struct {
			Calendar map[string]interface{} `json:"calendar"`
		}
		decodeBody(t, body, &result)

		id, _ := result.Calendar["id"].(string)
		if id == "" {
			t.Fatalf("expected calendar id, got %s", string(body))
		}
		if result.Calendar["name"] != calName {
			t.Fatalf("expected calendar name %q, got %v", calName, result.Calendar["name"])
		}
		if result.Calendar["owner_id"] != userID {
			t.Fatalf("expected owner_id %s, got %v", userID, result.Calendar["owner_id"])
		}
		calendarID = id
	})

	t.Run("update_calendar", func(t *testing.T) {
		if calendarID == "" {
			t.Skip("calendar was not created")
		}
		// 403 here for every user is the uuid.Nil actor: Update compares the actor to the owner.
		resp, body := putJSON(t, base+"/api/v1/calendar/calendars/"+calendarID, map[string]interface{}{
			"name":  calName + " renamed",
			"color": "#445566",
		}, token)
		requireStatus(t, resp, body, http.StatusOK)

		var result struct {
			Calendar map[string]interface{} `json:"calendar"`
		}
		decodeBody(t, body, &result)
		if result.Calendar["name"] != calName+" renamed" {
			t.Fatalf("expected renamed calendar, got %v", result.Calendar["name"])
		}
		if result.Calendar["color"] != "#445566" {
			t.Fatalf("expected color #445566, got %v", result.Calendar["color"])
		}
	})

	t.Run("list_calendars_contains_new_calendar", func(t *testing.T) {
		if calendarID == "" {
			t.Skip("calendar was not created")
		}
		resp, body := getJSON(t, base+"/api/v1/calendar/calendars", token)
		requireStatus(t, resp, body, http.StatusOK)

		var result struct {
			Calendars []struct {
				Calendar map[string]interface{} `json:"calendar"`
			} `json:"calendars"`
		}
		decodeBody(t, body, &result)
		for _, c := range result.Calendars {
			if c.Calendar["id"] == calendarID {
				return
			}
		}
		t.Fatalf("calendar %s not in list: %s", calendarID, string(body))
	})

	t.Run("add_member", func(t *testing.T) {
		if calendarID == "" {
			t.Skip("calendar was not created")
		}
		resp, body := postJSON(t, base+"/api/v1/calendar/calendars/"+calendarID+"/members", map[string]interface{}{
			"user_id":    memberID,
			"permission": "view",
		}, token)
		requireStatus(t, resp, body, http.StatusCreated)

		requireMemberPermission(t, base, token, calendarID, memberID, "view")
	})

	t.Run("update_member_permission", func(t *testing.T) {
		if calendarID == "" {
			t.Skip("calendar was not created")
		}
		resp, body := putJSON(t, base+"/api/v1/calendar/calendars/"+calendarID+"/members/"+memberID+"/permission", map[string]interface{}{
			"permission": "edit",
		}, token)
		requireStatus(t, resp, body, http.StatusOK)

		requireMemberPermission(t, base, token, calendarID, memberID, "edit")
	})

	t.Run("remove_member", func(t *testing.T) {
		if calendarID == "" {
			t.Skip("calendar was not created")
		}
		resp, body := deleteJSON(t, base+"/api/v1/calendar/calendars/"+calendarID+"/members/"+memberID, token)
		requireStatus(t, resp, body, http.StatusOK)

		requireMemberPermission(t, base, token, calendarID, memberID, "")
	})

	t.Run("create_category", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/calendar/categories", map[string]interface{}{
			"name":  fmt.Sprintf("E2E Category %d", suffix),
			"color": "#AA5500",
		}, token)
		requireStatus(t, resp, body, http.StatusCreated)

		var result struct {
			Category map[string]interface{} `json:"category"`
		}
		decodeBody(t, body, &result)
		id, _ := result.Category["id"].(string)
		if id == "" {
			t.Fatalf("expected category id, got %s", string(body))
		}
		categoryID = id
	})

	t.Run("create_event", func(t *testing.T) {
		if calendarID == "" {
			t.Skip("calendar was not created")
		}
		start := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Hour)
		payload := map[string]interface{}{
			"calendar_id": calendarID,
			"title":       "E2E Event",
			"start_time":  start.Format(time.RFC3339),
			"end_time":    start.Add(time.Hour).Format(time.RFC3339),
			"timezone":    "Europe/Berlin",
		}
		if categoryID != "" {
			payload["category_id"] = categoryID
		}
		resp, body := postJSON(t, base+"/api/v1/calendar/events", payload, token)
		requireStatus(t, resp, body, http.StatusCreated)

		var result struct {
			Event map[string]interface{} `json:"event"`
		}
		decodeBody(t, body, &result)
		id, _ := result.Event["id"].(string)
		if id == "" {
			t.Fatalf("expected event id, got %s", string(body))
		}
		if result.Event["calendar_id"] != calendarID {
			t.Fatalf("expected calendar_id %s, got %v", calendarID, result.Event["calendar_id"])
		}
		if result.Event["created_by"] != userID {
			t.Fatalf("expected created_by %s, got %v", userID, result.Event["created_by"])
		}
		eventID = id
	})

	t.Run("update_event", func(t *testing.T) {
		if eventID == "" {
			t.Skip("event was not created")
		}
		resp, body := putJSON(t, base+"/api/v1/calendar/events/"+eventID, map[string]interface{}{
			"title":    "E2E Event updated",
			"location": "Room 1",
		}, token)
		requireStatus(t, resp, body, http.StatusOK)

		var result struct {
			Event map[string]interface{} `json:"event"`
		}
		decodeBody(t, body, &result)
		if result.Event["title"] != "E2E Event updated" {
			t.Fatalf("expected updated title, got %v", result.Event["title"])
		}

		// The change must be readable through the single-event route as well.
		resp, body = getJSON(t, base+"/api/v1/calendar/events/"+eventID, token)
		requireStatus(t, resp, body, http.StatusOK)
		decodeBody(t, body, &result)
		if result.Event["title"] != "E2E Event updated" || result.Event["location"] != "Room 1" {
			t.Fatalf("event not persisted: %s", string(body))
		}
	})

	t.Run("set_reminders", func(t *testing.T) {
		if eventID == "" {
			t.Skip("event was not created")
		}
		resp, body := putJSON(t, base+"/api/v1/calendar/events/"+eventID+"/reminders", map[string]interface{}{
			"minutes_before": []int{15, 60},
		}, token)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = getJSON(t, base+"/api/v1/calendar/events/"+eventID+"/reminders", token)
		requireStatus(t, resp, body, http.StatusOK)

		var result struct {
			Reminders []map[string]interface{} `json:"reminders"`
		}
		decodeBody(t, body, &result)
		if len(result.Reminders) != 2 {
			t.Fatalf("expected 2 reminders, got %s", string(body))
		}
		got := map[float64]bool{}
		for _, r := range result.Reminders {
			m, _ := r["minutes_before"].(float64)
			got[m] = true
		}
		if !got[15] || !got[60] {
			t.Fatalf("expected reminders 15 and 60, got %s", string(body))
		}
	})

	t.Run("delete_event", func(t *testing.T) {
		if eventID == "" {
			t.Skip("event was not created")
		}
		resp, body := deleteJSON(t, base+"/api/v1/calendar/events/"+eventID, token)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = getJSON(t, base+"/api/v1/calendar/events/"+eventID, token)
		requireStatus(t, resp, body, http.StatusNotFound)
	})

	t.Run("delete_category", func(t *testing.T) {
		if categoryID == "" {
			t.Skip("category was not created")
		}
		resp, body := deleteJSON(t, base+"/api/v1/calendar/categories/"+categoryID, token)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = getJSON(t, base+"/api/v1/calendar/categories", token)
		requireStatus(t, resp, body, http.StatusOK)

		var result struct {
			Categories []map[string]interface{} `json:"categories"`
		}
		decodeBody(t, body, &result)
		for _, c := range result.Categories {
			if c["id"] == categoryID {
				t.Fatalf("category %s still listed after delete: %s", categoryID, string(body))
			}
		}
	})

	t.Run("delete_calendar", func(t *testing.T) {
		if calendarID == "" {
			t.Skip("calendar was not created")
		}
		resp, body := deleteJSON(t, base+"/api/v1/calendar/calendars/"+calendarID, token)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = getJSON(t, base+"/api/v1/calendar/calendars/"+calendarID, token)
		requireStatus(t, resp, body, http.StatusNotFound)
	})

	t.Run("no_auth", func(t *testing.T) {
		resp, _ := getJSON(t, base+"/api/v1/calendar/calendars", "")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})
}

// requireMemberPermission lists the calendar members and asserts the permission of
// memberID; an empty want asserts that the user is not a member.
func requireMemberPermission(t *testing.T, base, token, calendarID, memberID, want string) {
	t.Helper()
	resp, body := getJSON(t, base+"/api/v1/calendar/calendars/"+calendarID+"/members", token)
	requireStatus(t, resp, body, http.StatusOK)

	var result struct {
		Members []map[string]interface{} `json:"members"`
	}
	decodeBody(t, body, &result)

	got := ""
	for _, m := range result.Members {
		if m["user_id"] == memberID {
			got, _ = m["permission"].(string)
		}
	}
	if got != want {
		t.Fatalf("expected member permission %q, got %q: %s", want, got, string(body))
	}
}
