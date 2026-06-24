//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestWorkProjectFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	token, _ := registerAndLoginAdmin(t, base)

	var projectID string

	// Create project. project_key is required (uppercase A-Z0-9, 2-10 chars,
	// unique per tenant among active projects) — derive it from the clock so
	// repeated runs against a persistent DB do not collide.
	projectKey := fmt.Sprintf("E%d", time.Now().Unix()%1_000_000_000)

	t.Run("create_project", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/projects", map[string]interface{}{
			"name":        "E2E Test Project",
			"project_key": projectKey,
			"description": "Project created by E2E test",
		}, token)
		requireStatus(t, resp, body, http.StatusCreated)

		var result struct {
			Project map[string]interface{} `json:"project"`
		}
		decodeBody(t, body, &result)

		id, ok := result.Project["id"].(string)
		if !ok || id == "" {
			t.Fatal("expected project id")
		}
		projectID = id
	})

	// Get project
	t.Run("get_project", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/projects/"+projectID, token)
		requireStatus(t, resp, body, http.StatusOK)

		var result struct {
			Project map[string]interface{} `json:"project"`
		}
		decodeBody(t, body, &result)

		if result.Project["name"] != "E2E Test Project" {
			t.Fatalf("expected project name, got %v", result.Project["name"])
		}
	})

	// List projects
	t.Run("list_projects", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/projects", token)
		requireStatus(t, resp, body, http.StatusOK)

		var result struct {
			Projects []map[string]interface{} `json:"projects"`
		}
		decodeBody(t, body, &result)

		found := false
		for _, p := range result.Projects {
			if p["id"] == projectID {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("created project not found in list")
		}
	})

	// Update project
	t.Run("update_project", func(t *testing.T) {
		resp, body := putJSON(t, base+"/api/v1/projects/"+projectID, map[string]interface{}{
			"name":        "E2E Test Project Updated",
			"description": "Updated by E2E test",
		}, token)
		requireStatus(t, resp, body, http.StatusOK)
	})

	// Create task status for the project
	var statusID string
	t.Run("create_project_status", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/projects/"+projectID+"/statuses", map[string]interface{}{
			"name":  "To Do",
			"color": "#6B7280",
		}, token)
		requireStatus(t, resp, body, http.StatusCreated)

		var result struct {
			Status map[string]interface{} `json:"status"`
		}
		decodeBody(t, body, &result)

		id, ok := result.Status["id"].(string)
		if !ok || id == "" {
			t.Fatal("expected status id")
		}
		statusID = id
	})

	// Create task
	var taskID string
	t.Run("create_task", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/tasks", map[string]interface{}{
			"title":      "E2E Test Task",
			"project_id": projectID,
			"status_id":  statusID,
			"priority":   "normal",
		}, token)
		requireStatus(t, resp, body, http.StatusCreated)

		var result struct {
			Task map[string]interface{} `json:"task"`
		}
		decodeBody(t, body, &result)

		id, ok := result.Task["id"].(string)
		if !ok || id == "" {
			t.Fatal("expected task id")
		}
		taskID = id
	})

	// Get task
	t.Run("get_task", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/tasks/"+taskID, token)
		requireStatus(t, resp, body, http.StatusOK)

		var result struct {
			Task map[string]interface{} `json:"task"`
		}
		decodeBody(t, body, &result)

		if result.Task["title"] != "E2E Test Task" {
			t.Fatalf("expected task title, got %v", result.Task["title"])
		}
	})

	// List tasks
	t.Run("list_tasks", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/tasks?project_id="+projectID, token)
		requireStatus(t, resp, body, http.StatusOK)

		var raw map[string]json.RawMessage
		decodeBody(t, body, &raw)
		if _, ok := raw["tasks"]; !ok {
			t.Fatal("expected tasks key in response")
		}
	})

	// Update task
	t.Run("update_task", func(t *testing.T) {
		resp, body := putJSON(t, base+"/api/v1/tasks/"+taskID, map[string]interface{}{
			"title":    "E2E Test Task Updated",
			"priority": "high",
		}, token)
		requireStatus(t, resp, body, http.StatusOK)
	})

	// Add task comment
	t.Run("add_task_comment", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/tasks/"+taskID+"/comments", map[string]interface{}{
			"content": "E2E test comment",
		}, token)
		requireStatus(t, resp, body, http.StatusCreated)
	})

	// List task comments
	t.Run("list_task_comments", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/tasks/"+taskID+"/comments", token)
		requireStatus(t, resp, body, http.StatusOK)
	})

	// Delete task
	t.Run("delete_task", func(t *testing.T) {
		resp, body := deleteJSON(t, base+"/api/v1/tasks/"+taskID, token)
		requireStatus(t, resp, body, http.StatusOK)
	})

	// No auth
	t.Run("no_auth", func(t *testing.T) {
		resp, _ := getJSON(t, base+"/api/v1/projects", "")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})
}
