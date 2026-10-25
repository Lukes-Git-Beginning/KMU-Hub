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

// TestVermietungListsAndCalendarFlow runs the two reads the rental page opens with against the real
// seed, with more rows than the default page size: the object list with page_size 200 and the rental
// list with page_size 500 (VermietungPage.tsx:643/644), plus the month calendar of vermietung-client.ts
// (getRentalCalendar: month, year; the page has no component for it yet, the hook exists). The manager
// stocks the catalog, two members book; one of them reads like the desktop does. Needs the vermietung
// service in the stack.
//
// IST of the seed that the test pins and does not judge:
//   - a page_size above the gateway ceiling (200) is clamped, not refused: the desktop's 500 returns
//     the first 200 rows and `total` carries the full count;
//   - the lists and the calendar are tenant-wide for member (no own-scope), so the second member sees
//     the first one's rentals;
//   - the gateway reads only month and year of the calendar, the desktop's optional object_id is
//     ignored and every object with a live rental of the month is returned;
//   - "Stornieren" in the desktop is DELETE /rentals/{id}, a hard delete (route_vermietung.go: "soft
//     label over DeleteRental"): the rental leaves the calendar, the list and its total, GET is 404.
//
// The data sits in one March of a far future year (random per run) so reruns against the same stack
// neither overlap nor change the counts. There is no second tenant in the stack, so a foreign-tenant
// read is not part of this flow.
func TestVermietungListsAndCalendarFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	managerToken, _ := registerAndLoginWithRole(t, base, "manager")
	memberAToken, _ := registerAndLoginWithRole(t, base, "member")
	memberBToken, _ := registerAndLoginWithRole(t, base, "member")

	vm := base + "/api/v1/vermietung"
	suffix := time.Now().UnixNano()
	year := 2200 + int(suffix%700)
	march := func(day int) string {
		return time.Date(year, time.March, day, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	}
	const count = 55 // above the default page size of 50

	// rows decodes a list route into its rows and total. A missing key (empty list as `{}`) is an empty
	// list, `null` is the break the desktop's `?? []` would not survive.
	rows := func(t *testing.T, url, token, key string) ([]map[string]interface{}, int) {
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
		total := 0
		if tr, ok := raw["total"]; ok {
			decodeBody(t, tr, &total)
		}
		return out, total
	}
	idSet := func(list []map[string]interface{}) map[string]bool {
		out := map[string]bool{}
		for _, r := range list {
			if id, _ := r["id"].(string); id != "" {
				out[id] = true
			}
		}
		return out
	}

	objectIDs := make([]string, 0, count)
	rentalIDs := make([]string, 0, count)
	objectOf := map[string]string{} // rental id -> object id

	t.Run("manager stocks the catalog with more objects than one default page", func(t *testing.T) {
		for i := 0; i < count; i++ {
			resp, body := postJSON(t, vm+"/objects", map[string]interface{}{
				"name": fmt.Sprintf("E2E Liste %d-%02d", suffix, i), "category": "trailer",
				"daily_rate": 40.0, "deposit": 0.0,
			}, managerToken)
			requireStatus(t, resp, body, http.StatusCreated)
			var out struct {
				Object map[string]interface{} `json:"object"`
			}
			decodeBody(t, body, &out)
			id, _ := out.Object["id"].(string)
			if id == "" {
				t.Fatalf("expected the object in {object}, got %s", string(body))
			}
			objectIDs = append(objectIDs, id)
		}
	})

	t.Run("the members book one rental per object, alternating", func(t *testing.T) {
		for i, objectID := range objectIDs {
			token := memberAToken
			if i%2 == 1 {
				token = memberBToken
			}
			resp, body := postJSON(t, vm+"/rentals", map[string]interface{}{
				"object_id": objectID, "renter_name": fmt.Sprintf("E2E Mieter %02d", i),
				"start_date": march(5), "end_date": march(7), "total_price": 80.0, "deposit_paid": false,
			}, token)
			requireStatus(t, resp, body, http.StatusCreated)
			var out struct {
				Rental map[string]interface{} `json:"rental"`
			}
			decodeBody(t, body, &out)
			id, _ := out.Rental["id"].(string)
			if id == "" {
				t.Fatalf("expected the rental in {rental}, got %s", string(body))
			}
			rentalIDs = append(rentalIDs, id)
			objectOf[id] = objectID
		}
	})

	t.Run("the object list with the desktop's page_size 200 returns more than 50 rows", func(t *testing.T) {
		list, total := rows(t, vm+"/objects?page=1&page_size=200", memberAToken, "objects")
		if len(list) <= 50 || len(list) > 200 || total < count {
			t.Fatalf("expected 51..200 objects and a total of at least %d, got %d rows, total %d", count, len(list), total)
		}

		// A search on the run's name finds exactly the 55 new rows, whatever else sits in the tenant.
		found, foundTotal := rows(t, fmt.Sprintf("%s/objects?page=1&page_size=200&search=E2E+Liste+%d", vm, suffix), memberBToken, "objects")
		set := idSet(found)
		if len(found) != count || foundTotal != count {
			t.Fatalf("expected exactly %d objects for the run, got %d rows, total %d", count, len(found), foundTotal)
		}
		for _, id := range objectIDs {
			if !set[id] {
				t.Fatalf("object %s is missing from the list", id)
			}
		}
	})

	t.Run("the rental list with the desktop's page_size 500 is clamped to 200, total stays whole", func(t *testing.T) {
		list, total := rows(t, vm+"/rentals?page=1&page_size=500", memberAToken, "rentals")
		if len(list) <= 50 || len(list) > 200 {
			t.Fatalf("expected 51..200 rentals for page_size 500, got %d", len(list))
		}
		if total < count {
			t.Fatalf("total must carry the full count (at least %d), got %d", count, total)
		}
		for _, r := range list {
			for _, k := range []string{"id", "object_id", "renter_name", "start_date", "end_date", "status"} {
				if v, _ := r[k].(string); v == "" {
					t.Fatalf("a rental row misses %q (the desktop type has it as required): %v", k, r)
				}
			}
		}

		// The run's own rentals, narrowed by the window the way the desktop narrows by date.
		window := fmt.Sprintf("%s/rentals?page=1&page_size=500&from=%s&to=%s", vm, march(1), march(31))
		inWindow, windowTotal := rows(t, window, memberBToken, "rentals")
		set := idSet(inWindow)
		if len(inWindow) != count || windowTotal != count {
			t.Fatalf("expected exactly %d rentals in the window, got %d rows, total %d", count, len(inWindow), windowTotal)
		}
		for _, id := range rentalIDs {
			if !set[id] {
				t.Fatalf("rental %s is missing from the window list (member B reads member A's too)", id)
			}
		}
	})

	t.Run("paging walks the rows without gaps or overlap", func(t *testing.T) {
		seen := map[string]bool{}
		for page := 1; page <= 3; page++ {
			url := fmt.Sprintf("%s/rentals?page=%d&page_size=20&from=%s&to=%s", vm, page, march(1), march(31))
			list, total := rows(t, url, memberAToken, "rentals")
			want := 20
			if page == 3 {
				want = count - 40
			}
			if len(list) != want || total != count {
				t.Fatalf("page %d: expected %d rows of total %d, got %d rows, total %d", page, want, count, len(list), total)
			}
			for _, r := range list {
				id, _ := r["id"].(string)
				if seen[id] {
					t.Fatalf("rental %s showed up on two pages", id)
				}
				seen[id] = true
			}
		}
		if len(seen) != count {
			t.Fatalf("three pages must cover all %d rentals, got %d", count, len(seen))
		}
		// A page past the end is an empty list, not an error.
		list, total := rows(t, fmt.Sprintf("%s/rentals?page=4&page_size=20&from=%s&to=%s", vm, march(1), march(31)), memberAToken, "rentals")
		if len(list) != 0 || total != count {
			t.Fatalf("page 4 must be empty with the total kept, got %d rows, total %d", len(list), total)
		}
	})

	calendar := func(t *testing.T, token string, month int, extra string) map[string]map[string]interface{} {
		t.Helper()
		resp, body := getJSON(t, fmt.Sprintf("%s/calendar?month=%d&year=%d%s", vm, month, year, extra), token)
		requireStatus(t, resp, body, http.StatusOK)
		var out struct {
			Entries []map[string]interface{} `json:"entries"`
		}
		decodeBody(t, body, &out)
		if out.Entries == nil {
			t.Fatalf("the calendar must carry entries as an array (the desktop's CalendarResult), got %s", string(body))
		}
		byObject := map[string]map[string]interface{}{}
		for _, e := range out.Entries {
			if id, _ := e["object_id"].(string); id != "" {
				byObject[id] = e
			}
		}
		return byObject
	}

	t.Run("the month calendar groups the rentals by object for a member", func(t *testing.T) {
		entries := calendar(t, memberBToken, int(time.March), "")
		for i, objectID := range objectIDs {
			e, ok := entries[objectID]
			if !ok {
				t.Fatalf("object %s has no calendar entry for the month", objectID)
			}
			if name, _ := e["object_name"].(string); name != fmt.Sprintf("E2E Liste %d-%02d", suffix, i) {
				t.Fatalf("entry carries the wrong object_name %q", name)
			}
			rentals, _ := e["rentals"].([]interface{})
			if len(rentals) != 1 {
				t.Fatalf("object %s must hold exactly its one rental, got %v", objectID, e["rentals"])
			}
			if got, _ := rentals[0].(map[string]interface{})["id"].(string); got != rentalIDs[i] {
				t.Fatalf("entry of object %s holds rental %s, want %s", objectID, got, rentalIDs[i])
			}
		}

		// IST: the optional object_id of the desktop's params is not read, every object still comes back.
		if got := calendar(t, memberBToken, int(time.March), "&object_id="+objectIDs[0]); len(got) < count {
			t.Fatalf("IST: the gateway ignores object_id, expected at least %d entries, got %d", count, len(got))
		}

		// A month without rentals is an empty array, and the neighbouring month does not hold the March rentals.
		for _, month := range []int{int(time.February), int(time.April)} {
			if got := calendar(t, memberAToken, month, ""); len(got) != 0 {
				t.Fatalf("month %d of the run's year must be empty, got %d entries", month, len(got))
			}
		}
	})

	t.Run("a cancelled rental is gone from the calendar and the list", func(t *testing.T) {
		cancelled := rentalIDs[0]
		resp, body := deleteJSON(t, vm+"/rentals/"+cancelled, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden) // cancel is manager and admin
		resp, body = deleteJSON(t, vm+"/rentals/"+cancelled, managerToken)
		requireStatus(t, resp, body, http.StatusNoContent)

		entries := calendar(t, memberAToken, int(time.March), "")
		if _, still := entries[objectOf[cancelled]]; still {
			t.Fatalf("a cancelled rental must not stay in the calendar")
		}
		if len(entries) != count-1 {
			t.Fatalf("the other %d objects stay in the calendar, got %d", count-1, len(entries))
		}

		url := fmt.Sprintf("%s/rentals?page=1&page_size=500&from=%s&to=%s", vm, march(1), march(31))
		list, total := rows(t, url, memberBToken, "rentals")
		if len(list) != count-1 || total != count-1 || idSet(list)[cancelled] {
			t.Fatalf("IST: the rental is deleted, %d rows must stay (total %d), got %d rows, total %d", count-1, count-1, len(list), total)
		}
		resp, body = getJSON(t, vm+"/rentals/"+cancelled, memberBToken)
		requireStatus(t, resp, body, http.StatusNotFound)
	})
}
