//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestCalendarResourcesFlow walks the resource paths that run with the service-side
// admin check switched off (resourcesOpenToTenant) and the booking paths that read the
// actor from the token: create/read/update/list/delete a resource, book it for an event,
// see the booking in the availability view, hit a double booking, and cancel as the owner.
// Cancelling compares booked_by to the caller, so a nil actor shows up as a 403 for the owner.
func TestCalendarResourcesFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	token, userID := registerAndLoginAdmin(t, base)
	otherToken, _ := registerAndLoginAdmin(t, base)

	suffix := time.Now().UnixNano()
	resName := fmt.Sprintf("E2E Room %d", suffix)
	start := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Hour)
	end := start.Add(time.Hour)

	var resourceID, calendarID, eventID, bookingID string

	t.Run("create_resource", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/calendar/resources", map[string]interface{}{
			"name":          resName,
			"resource_type": "room",
			"capacity":      8,
			"floor":         "2",
			"tags":          []string{"projector", "whiteboard"},
		}, token)
		requireStatus(t, resp, body, http.StatusCreated)

		var result struct {
			Resource map[string]interface{} `json:"resource"`
		}
		decodeBody(t, body, &result)
		id, _ := result.Resource["id"].(string)
		if id == "" {
			t.Fatalf("expected resource id, got %s", string(body))
		}
		if result.Resource["name"] != resName || result.Resource["resource_type"] != "room" {
			t.Fatalf("unexpected resource: %s", string(body))
		}
		if result.Resource["is_active"] != true {
			t.Fatalf("expected an active resource, got %s", string(body))
		}
		if result.Resource["created_by"] != userID {
			t.Fatalf("expected created_by %s, got %v", userID, result.Resource["created_by"])
		}
		if capacity, _ := result.Resource["capacity"].(float64); capacity != 8 {
			t.Fatalf("expected capacity 8, got %v", result.Resource["capacity"])
		}
		resourceID = id
	})

	t.Run("create_resource_validation", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/calendar/resources", map[string]interface{}{
			"name":          "E2E invalid type",
			"resource_type": "spaceship",
		}, token)
		requireStatus(t, resp, body, http.StatusBadRequest)

		resp, body = postJSON(t, base+"/api/v1/calendar/resources", map[string]interface{}{
			"resource_type": "room",
		}, token)
		requireStatus(t, resp, body, http.StatusBadRequest)
	})

	t.Run("get_resource", func(t *testing.T) {
		if resourceID == "" {
			t.Skip("resource was not created")
		}
		resp, body := getJSON(t, base+"/api/v1/calendar/resources/"+resourceID, token)
		requireStatus(t, resp, body, http.StatusOK)

		var result struct {
			Resource struct {
				ID   string   `json:"id"`
				Name string   `json:"name"`
				Tags []string `json:"tags"`
			} `json:"resource"`
		}
		decodeBody(t, body, &result)
		if result.Resource.ID != resourceID || result.Resource.Name != resName {
			t.Fatalf("unexpected resource: %s", string(body))
		}
		if len(result.Resource.Tags) != 2 {
			t.Fatalf("expected 2 tags, got %s", string(body))
		}
	})

	t.Run("list_resources_filters", func(t *testing.T) {
		if resourceID == "" {
			t.Skip("resource was not created")
		}
		if !resourceListed(t, base, token, "", resourceID) {
			t.Fatalf("resource %s missing from the unfiltered list", resourceID)
		}
		if !resourceListed(t, base, token, "?type=room&min_capacity=8", resourceID) {
			t.Fatalf("resource %s missing from the room/capacity filter", resourceID)
		}
		if resourceListed(t, base, token, "?type=vehicle", resourceID) {
			t.Fatalf("resource %s listed under type=vehicle", resourceID)
		}
		if resourceListed(t, base, token, "?min_capacity=100", resourceID) {
			t.Fatalf("resource %s listed under min_capacity=100", resourceID)
		}
	})

	t.Run("update_resource", func(t *testing.T) {
		if resourceID == "" {
			t.Skip("resource was not created")
		}
		resp, body := putJSON(t, base+"/api/v1/calendar/resources/"+resourceID, map[string]interface{}{
			"name":     resName + " renamed",
			"capacity": 12,
			"location": "Building B",
		}, token)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = getJSON(t, base+"/api/v1/calendar/resources/"+resourceID, token)
		requireStatus(t, resp, body, http.StatusOK)
		var result struct {
			Resource map[string]interface{} `json:"resource"`
		}
		decodeBody(t, body, &result)
		if result.Resource["name"] != resName+" renamed" || result.Resource["location"] != "Building B" {
			t.Fatalf("update not persisted: %s", string(body))
		}
		if capacity, _ := result.Resource["capacity"].(float64); capacity != 12 {
			t.Fatalf("expected capacity 12, got %v", result.Resource["capacity"])
		}

		resp, body = putJSON(t, base+"/api/v1/calendar/resources/"+resourceID, map[string]interface{}{
			"resource_type": "spaceship",
		}, token)
		requireStatus(t, resp, body, http.StatusBadRequest)
	})

	t.Run("create_event_to_book_for", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/calendar/calendars", map[string]interface{}{
			"name":          fmt.Sprintf("E2E Resource Calendar %d", suffix),
			"calendar_type": "shared",
		}, token)
		requireStatus(t, resp, body, http.StatusCreated)
		var cal struct {
			Calendar map[string]interface{} `json:"calendar"`
		}
		decodeBody(t, body, &cal)
		calendarID, _ = cal.Calendar["id"].(string)
		if calendarID == "" {
			t.Fatalf("expected calendar id, got %s", string(body))
		}

		resp, body = postJSON(t, base+"/api/v1/calendar/events", map[string]interface{}{
			"calendar_id": calendarID,
			"title":       "E2E Meeting",
			"start_time":  start.Format(time.RFC3339),
			"end_time":    end.Format(time.RFC3339),
			"timezone":    "Europe/Berlin",
		}, token)
		requireStatus(t, resp, body, http.StatusCreated)
		var ev struct {
			Event map[string]interface{} `json:"event"`
		}
		decodeBody(t, body, &ev)
		eventID, _ = ev.Event["id"].(string)
		if eventID == "" {
			t.Fatalf("expected event id, got %s", string(body))
		}
	})

	bookingPayload := func(from, to time.Time) map[string]interface{} {
		return map[string]interface{}{
			"resource_id": resourceID,
			"event_id":    eventID,
			"start_time":  from.Format(time.RFC3339),
			"end_time":    to.Format(time.RFC3339),
		}
	}

	t.Run("book_resource", func(t *testing.T) {
		if resourceID == "" || eventID == "" {
			t.Skip("resource or event was not created")
		}
		// The desktop books under /resources/bookings (9e6e19af).
		resp, body := postJSON(t, base+"/api/v1/calendar/resources/bookings", bookingPayload(start, end), token)
		requireStatus(t, resp, body, http.StatusCreated)

		var result struct {
			Booking map[string]interface{} `json:"booking"`
		}
		decodeBody(t, body, &result)
		id, _ := result.Booking["id"].(string)
		if id == "" {
			t.Fatalf("expected booking id, got %s", string(body))
		}
		if result.Booking["resource_id"] != resourceID || result.Booking["event_id"] != eventID {
			t.Fatalf("unexpected booking: %s", string(body))
		}
		if result.Booking["booked_by"] != userID {
			t.Fatalf("expected booked_by %s, got %v", userID, result.Booking["booked_by"])
		}
		bookingID = id
	})

	t.Run("availability_shows_booking", func(t *testing.T) {
		if bookingID == "" {
			t.Skip("booking was not created")
		}
		got := availabilityBookingIDs(t, base, token, resourceID, start, end)
		if len(got) != 1 || got[0] != bookingID {
			t.Fatalf("expected availability to list booking %s, got %v", bookingID, got)
		}
	})

	t.Run("double_booking_conflicts", func(t *testing.T) {
		if bookingID == "" {
			t.Skip("booking was not created")
		}
		resp, body := postJSON(t, base+"/api/v1/calendar/resources/bookings", bookingPayload(start, end), otherToken)
		requireStatus(t, resp, body, http.StatusConflict)
	})

	t.Run("booking_validation", func(t *testing.T) {
		if resourceID == "" || eventID == "" {
			t.Skip("resource or event was not created")
		}
		resp, body := postJSON(t, base+"/api/v1/calendar/resources/bookings", bookingPayload(start, start.Add(-time.Hour)), token)
		requireStatus(t, resp, body, http.StatusBadRequest)

		resp, body = getJSON(t, base+"/api/v1/calendar/resources/"+resourceID+"/availability", token)
		requireStatus(t, resp, body, http.StatusBadRequest)
	})

	t.Run("cancel_by_non_owner_is_forbidden", func(t *testing.T) {
		if bookingID == "" {
			t.Skip("booking was not created")
		}
		resp, body := deleteJSON(t, base+"/api/v1/calendar/resources/bookings/"+bookingID, otherToken)
		requireStatus(t, resp, body, http.StatusForbidden)
	})

	t.Run("cancel_booking", func(t *testing.T) {
		if bookingID == "" {
			t.Skip("booking was not created")
		}
		// 403 here for the owner is the nil actor: CancelBooking compares booked_by to the caller.
		resp, body := deleteJSON(t, base+"/api/v1/calendar/resources/bookings/"+bookingID, token)
		requireStatus(t, resp, body, http.StatusOK)

		if got := availabilityBookingIDs(t, base, token, resourceID, start, end); len(got) != 0 {
			t.Fatalf("expected no bookings after cancel, got %v", got)
		}

		resp, body = deleteJSON(t, base+"/api/v1/calendar/resources/bookings/"+bookingID, token)
		requireStatus(t, resp, body, http.StatusConflict)

		// A cancelled booking frees the slot again.
		resp, body = postJSON(t, base+"/api/v1/calendar/resources/bookings", bookingPayload(start, end), otherToken)
		requireStatus(t, resp, body, http.StatusCreated)
	})

	t.Run("cancel_unknown_booking", func(t *testing.T) {
		resp, body := deleteJSON(t, base+"/api/v1/calendar/resources/bookings/00000000-0000-0000-0000-00000000dead", token)
		requireStatus(t, resp, body, http.StatusNotFound)
	})

	t.Run("delete_resource", func(t *testing.T) {
		if resourceID == "" || eventID == "" {
			t.Skip("resource or event was not created")
		}
		resp, body := deleteJSON(t, base+"/api/v1/calendar/resources/"+resourceID, token)
		requireStatus(t, resp, body, http.StatusOK)

		// Delete is a soft delete: the resource leaves the default list but is still listed
		// with include_inactive.
		if resourceListed(t, base, token, "", resourceID) {
			t.Fatalf("deleted resource %s still in the default list", resourceID)
		}
		if !resourceListed(t, base, token, "?include_inactive=true", resourceID) {
			t.Fatalf("deleted resource %s missing from include_inactive=true", resourceID)
		}

		// An inactive resource can no longer be booked.
		resp, body = postJSON(t, base+"/api/v1/calendar/resources/bookings",
			bookingPayload(start.Add(3*time.Hour), start.Add(4*time.Hour)), token)
		requireStatus(t, resp, body, http.StatusConflict)

		resp, body = deleteJSON(t, base+"/api/v1/calendar/resources/00000000-0000-0000-0000-00000000dead", token)
		requireStatus(t, resp, body, http.StatusNotFound)
	})

	t.Run("cleanup_calendar", func(t *testing.T) {
		if calendarID == "" {
			t.Skip("calendar was not created")
		}
		resp, body := deleteJSON(t, base+"/api/v1/calendar/calendars/"+calendarID, token)
		requireStatus(t, resp, body, http.StatusOK)
	})

	t.Run("no_auth", func(t *testing.T) {
		resp, _ := getJSON(t, base+"/api/v1/calendar/resources", "")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})
}

// resourceListed reports whether resourceID is in GET /calendar/resources<query>. An empty
// list arrives without the `resources` key (protojson omits empty repeated fields); the
// desktop reads it with `?? []`, so absence counts as empty here too.
func resourceListed(t *testing.T, base, token, query, resourceID string) bool {
	t.Helper()
	resp, body := getJSON(t, base+"/api/v1/calendar/resources"+query, token)
	requireStatus(t, resp, body, http.StatusOK)

	var result struct {
		Resources []map[string]interface{} `json:"resources"`
	}
	decodeBody(t, body, &result)
	for _, r := range result.Resources {
		if r["id"] == resourceID {
			return true
		}
	}
	return false
}

// availabilityBookingIDs returns the booking IDs the availability view shows for the day
// of start; an empty day arrives without the `bookings` key.
func availabilityBookingIDs(t *testing.T, base, token, resourceID string, start, end time.Time) []string {
	t.Helper()
	url := fmt.Sprintf("%s/api/v1/calendar/resources/%s/availability?start=%s&end=%s",
		base, resourceID, start.Format(time.RFC3339), end.Format(time.RFC3339))
	resp, body := getJSON(t, url, token)
	requireStatus(t, resp, body, http.StatusOK)

	var result struct {
		Bookings []map[string]interface{} `json:"bookings"`
	}
	decodeBody(t, body, &result)
	ids := make([]string, 0, len(result.Bookings))
	for _, b := range result.Bookings {
		id, _ := b["id"].(string)
		ids = append(ids, id)
	}
	return ids
}
