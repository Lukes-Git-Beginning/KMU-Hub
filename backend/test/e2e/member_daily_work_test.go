//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestMemberDailyWorkFlow is the working day of a plain member outside HR: calendar, chat, tasks,
// the document view, the team picker and the header search, against the real seed keys. Every body
// is copied from the desktop (file named at the step).
//
// Roles: A is the member under test, B a second member (chat partner), the admin owns the project
// and the contact.
func TestMemberDailyWorkFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	adminToken, _ := registerAndLoginWithRole(t, base, "admin")
	memberToken, memberID := registerAndLoginWithRole(t, base, "member")
	colleagueToken, colleagueID := registerAndLoginWithRole(t, base, "member")

	suffix := time.Now().UnixNano()

	t.Run("calendar: member creates and deletes an own event", func(t *testing.T) {
		// calendar-types.ts CreateCalendarRequest / CreateEventRequest (useCalendars.ts, useEvents.ts).
		resp, body := postJSON(t, base+"/api/v1/calendar/calendars", map[string]interface{}{
			"name":          fmt.Sprintf("E2E member calendar %d", suffix),
			"calendar_type": "personal",
			"color":         "#336699",
		}, memberToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var cal struct {
			Calendar map[string]interface{} `json:"calendar"`
		}
		decodeBody(t, body, &cal)
		calendarID, _ := cal.Calendar["id"].(string)
		if calendarID == "" || cal.Calendar["owner_id"] != memberID {
			t.Fatalf("expected an own calendar, got %s", string(body))
		}

		start := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Hour)
		resp, body = postJSON(t, base+"/api/v1/calendar/events", map[string]interface{}{
			"calendar_id": calendarID,
			"title":       "E2E member event",
			"start_time":  start.Format(time.RFC3339),
			"end_time":    start.Add(time.Hour).Format(time.RFC3339),
			"is_all_day":  false,
			"timezone":    "Europe/Berlin",
		}, memberToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var ev struct {
			Event map[string]interface{} `json:"event"`
		}
		decodeBody(t, body, &ev)
		eventID, _ := ev.Event["id"].(string)
		if eventID == "" || ev.Event["created_by"] != memberID {
			t.Fatalf("expected an event created by the member, got %s", string(body))
		}

		// Deleting needs calendars:delete (scope own, migration 000336); the owner check is in the service.
		resp, body = deleteJSON(t, base+"/api/v1/calendar/events/"+eventID, memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		resp, body = getJSON(t, base+"/api/v1/calendar/events/"+eventID, memberToken)
		requireStatus(t, resp, body, http.StatusNotFound)
	})

	t.Run("chat: member sends and deletes the own message, a colleague's stays", func(t *testing.T) {
		// useChannels.ts: POST /channels/dm {other_user_id}; handlers.ts: POST /channels/{id}/messages {content}.
		resp, body := postJSON(t, base+"/api/v1/channels/dm", map[string]interface{}{
			"other_user_id": colleagueID,
		}, memberToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var dm struct {
			Channel map[string]interface{} `json:"channel"`
		}
		decodeBody(t, body, &dm)
		channelID, _ := dm.Channel["id"].(string)
		if channelID == "" {
			t.Fatalf("expected a DM channel id, got %s", string(body))
		}

		send := func(token, content string) string {
			resp, body := postJSON(t, base+"/api/v1/channels/"+channelID+"/messages", map[string]interface{}{
				"content": content,
			}, token)
			requireStatus(t, resp, body, http.StatusCreated)
			var m struct {
				Message map[string]interface{} `json:"message"`
			}
			decodeBody(t, body, &m)
			id, _ := m.Message["id"].(string)
			if id == "" {
				t.Fatalf("expected a message id, got %s", string(body))
			}
			return id
		}
		ownMessage := send(memberToken, "E2E own message")
		foreignMessage := send(colleagueToken, "E2E colleague message")

		// Fremde Nachricht: messages:delete is held (scope own), the service refuses a non-author
		// who is no channel moderator.
		resp, body = deleteJSON(t, base+"/api/v1/messages/"+foreignMessage, memberToken)
		requireStatus(t, resp, body, http.StatusForbidden)

		resp, body = deleteJSON(t, base+"/api/v1/messages/"+ownMessage, memberToken)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = getJSON(t, base+"/api/v1/channels/"+channelID+"/messages", memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var list struct {
			Messages []map[string]interface{} `json:"messages"`
		}
		decodeBody(t, body, &list)
		var sawForeign bool
		for _, m := range list.Messages {
			if m["id"] == ownMessage && m["is_deleted"] != true {
				t.Fatalf("deleted own message still listed as live: %v", m)
			}
			if m["id"] == foreignMessage {
				sawForeign = true
			}
		}
		if !sawForeign {
			t.Fatalf("the colleague's message must survive the refused delete, got %s", string(body))
		}
	})

	t.Run("tasks: member works in a project they are no member of", func(t *testing.T) {
		// useTasks.ts: create {title, priority, project_id, status_id}, PUT {title, priority},
		// POST /tasks/{id}/move {status_id, sort_order}. Open-project rule: fix-work-task-ops-on-open-projects.
		projectKey := fmt.Sprintf("M%d", suffix%1_000_000_000)
		resp, body := postJSON(t, base+"/api/v1/projects", map[string]interface{}{
			"name":        "E2E open project",
			"project_key": projectKey,
		}, adminToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var proj struct {
			Project map[string]interface{} `json:"project"`
		}
		decodeBody(t, body, &proj)
		projectID, _ := proj.Project["id"].(string)
		if projectID == "" {
			t.Fatalf("expected a project id, got %s", string(body))
		}

		newStatus := func(name string) string {
			resp, body := postJSON(t, base+"/api/v1/projects/"+projectID+"/statuses", map[string]interface{}{
				"name":  name,
				"color": "#6B7280",
			}, adminToken)
			requireStatus(t, resp, body, http.StatusCreated)
			var s struct {
				Status map[string]interface{} `json:"status"`
			}
			decodeBody(t, body, &s)
			id, _ := s.Status["id"].(string)
			if id == "" {
				t.Fatalf("expected a status id, got %s", string(body))
			}
			return id
		}
		todoID, doneID := newStatus("To Do"), newStatus("Done")

		resp, body = getJSON(t, base+"/api/v1/projects", memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var plist struct {
			Projects []map[string]interface{} `json:"projects"`
		}
		decodeBody(t, body, &plist)
		var seen bool
		for _, p := range plist.Projects {
			if p["id"] == projectID {
				seen = true
			}
		}
		if !seen {
			t.Fatalf("the admin's project is not listed for the member: %s", string(body))
		}

		resp, body = postJSON(t, base+"/api/v1/tasks", map[string]interface{}{
			"title":      "E2E member task",
			"priority":   "normal",
			"project_id": projectID,
			"status_id":  todoID,
		}, memberToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var task struct {
			Task map[string]interface{} `json:"task"`
		}
		decodeBody(t, body, &task)
		taskID, _ := task.Task["id"].(string)
		if taskID == "" {
			t.Fatalf("expected a task id, got %s", string(body))
		}

		resp, body = putJSON(t, base+"/api/v1/tasks/"+taskID, map[string]interface{}{
			"title":    "E2E member task renamed",
			"priority": "high",
		}, memberToken)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = postJSON(t, base+"/api/v1/tasks/"+taskID+"/move", map[string]interface{}{
			"status_id":  doneID,
			"sort_order": 1,
		}, memberToken)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = getJSON(t, base+"/api/v1/tasks/"+taskID, memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		decodeBody(t, body, &task)
		if task.Task["title"] != "E2E member task renamed" || task.Task["priority"] != "high" {
			t.Fatalf("update did not arrive: %s", string(body))
		}
		if task.Task["status_id"] != doneID {
			t.Fatalf("move did not arrive: status_id = %v, want %s", task.Task["status_id"], doneID)
		}

		// Delete: tasks:delete / work:task:delete are not in the member preset (manager and admin
		// have them). Open product question: BACKLOG-NEXT decide-member-manager-module-defaults.
		resp, body = deleteJSON(t, base+"/api/v1/tasks/"+taskID, memberToken)
		requireStatus(t, resp, body, http.StatusForbidden)
		resp, body = deleteJSON(t, base+"/api/v1/tasks/"+taskID, adminToken)
		requireStatus(t, resp, body, http.StatusOK)
	})

	t.Run("documents: member loads the virtual file view", func(t *testing.T) {
		// document-client.ts getVirtualFiles: {files: [], total}; the helper pins both.
		virtualFilesList(t, base, memberToken, "")
	})

	t.Run("team: member loads the people picker", func(t *testing.T) {
		// hr-client.ts listEmployees (MentionAutocomplete, HelpdeskPage): the narrow directory
		// projection {employees, total}, no hr:read fields.
		hrCreateEmployeeProfile(t, base, adminToken, memberID)
		hrCreateEmployeeProfile(t, base, adminToken, colleagueID)
		resp, body := getJSON(t, base+"/api/v1/hr/employees", memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var dir struct {
			Employees []map[string]interface{} `json:"employees"`
			Total     float64                  `json:"total"`
		}
		decodeBody(t, body, &dir)
		found := map[string]bool{}
		for _, e := range dir.Employees {
			if uid, _ := e["user_id"].(string); uid != "" {
				found[uid] = true
			}
			if _, leaked := e["contract_type"]; leaked {
				t.Fatalf("directory entry carries a field beyond the projection: %v", e)
			}
		}
		if !found[memberID] || !found[colleagueID] {
			t.Fatalf("member and colleague must be listed in the directory, got %s", string(body))
		}
	})

	t.Run("search: member finds a contact, tasks are no search source", func(t *testing.T) {
		// useGlobalSearch.ts: GET /search/global?q=, {query, modules: [{module, results, total}]}.
		term := fmt.Sprintf("Zq%d", suffix)
		resp, body := postJSON(t, base+"/api/v1/contacts", map[string]interface{}{
			"first_name": "Suche",
			"last_name":  term,
		}, adminToken)
		requireStatus(t, resp, body, http.StatusCreated)
		var c struct {
			Contact map[string]interface{} `json:"contact"`
		}
		decodeBody(t, body, &c)
		contactID, _ := c.Contact["id"].(string)

		resp, body = getJSON(t, base+"/api/v1/search/global?q="+term, memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var res struct {
			Query   string `json:"query"`
			Modules []struct {
				Module  string                   `json:"module"`
				Results []map[string]interface{} `json:"results"`
				Error   string                   `json:"error"`
			} `json:"modules"`
		}
		decodeBody(t, body, &res)
		var hit bool
		for _, m := range res.Modules {
			if m.Module == "tasks" {
				t.Fatalf("tasks are not a global-search source (route_search_global.go): %s", string(body))
			}
			if m.Error != "" {
				t.Fatalf("module %s failed: %s", m.Module, m.Error)
			}
			if m.Module != "crm" {
				continue
			}
			for _, r := range m.Results {
				if r["id"] == contactID && r["entity_type"] == "contact" {
					hit = true
				}
			}
		}
		if !hit {
			t.Fatalf("contact %s not found for %q: %s", contactID, term, string(body))
		}
	})
}
