//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// projectMembers decodes a members list and fails if the envelope does not
// carry a members array (the desktop reads `members ?? []`, a null would hide
// a broken wire shape behind the fallback).
func projectMembers(t *testing.T, base, projectID, token string) []map[string]interface{} {
	t.Helper()
	resp, body := getJSON(t, base+"/api/v1/projects/"+projectID+"/members", token)
	requireStatus(t, resp, body, http.StatusOK)

	var raw map[string]json.RawMessage
	decodeBody(t, body, &raw)
	if string(raw["members"]) == "null" || len(raw["members"]) == 0 {
		t.Fatalf("expected members array, got %s", string(body))
	}
	var result struct {
		Members []map[string]interface{} `json:"members"`
	}
	decodeBody(t, body, &result)
	return result.Members
}

func memberByUser(members []map[string]interface{}, userID string) map[string]interface{} {
	for _, m := range members {
		if m["user_id"] == userID {
			return m
		}
	}
	return nil
}

// projectListed reports whether the project shows up in GET /projects<query>.
func projectListed(t *testing.T, base, query, projectID, token string) bool {
	t.Helper()
	resp, body := getJSON(t, base+"/api/v1/projects"+query, token)
	requireStatus(t, resp, body, http.StatusOK)

	var result struct {
		Projects []map[string]interface{} `json:"projects"`
	}
	decodeBody(t, body, &result)
	for _, p := range result.Projects {
		if p["id"] == projectID {
			return true
		}
	}
	return false
}

// TestWorkProjectMembersFlow covers the member lifecycle, archive and delete of
// a project. Projects are open to everyone with project rights (decision
// 2026-10-23, projectsOpenToTenant in work_grpc.go), so a second admin who is
// not a member must still see and read the project.
func TestWorkProjectMembersFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	ownerToken, ownerID := registerAndLoginAdmin(t, base)
	outsiderToken, outsiderID := registerAndLoginAdmin(t, base)
	_, memberID := registerAndLogin(t, base)

	projectKey := fmt.Sprintf("M%d", time.Now().Unix()%1_000_000_000)
	var projectID string

	t.Run("create_project", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/projects", map[string]interface{}{
			"name":        "E2E Members Project",
			"project_key": projectKey,
		}, ownerToken)
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

	t.Run("creator_is_owner", func(t *testing.T) {
		members := projectMembers(t, base, projectID, ownerToken)
		if len(members) != 1 {
			t.Fatalf("expected the creator as only member, got %d", len(members))
		}
		m := memberByUser(members, ownerID)
		if m == nil || m["role"] != "owner" {
			t.Fatalf("expected creator as owner, got %v", members)
		}
		if m["first_name"] != "E2E" || m["last_name"] != "Admin" {
			t.Fatalf("expected names on the member row, got %v", m)
		}
	})

	t.Run("add_member", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/projects/"+projectID+"/members", map[string]interface{}{
			"user_id": memberID,
			"role":    "member",
		}, ownerToken)
		requireStatus(t, resp, body, http.StatusCreated)

		var result struct {
			Member map[string]interface{} `json:"member"`
		}
		decodeBody(t, body, &result)
		if result.Member["user_id"] != memberID || result.Member["role"] != "member" {
			t.Fatalf("expected the added member in the response, got %v", result.Member)
		}
		if result.Member["first_name"] != "E2E" || result.Member["last_name"] != "Test" {
			t.Fatalf("expected names on the added member, got %v", result.Member)
		}
	})

	t.Run("list_shows_member", func(t *testing.T) {
		members := projectMembers(t, base, projectID, ownerToken)
		if len(members) != 2 {
			t.Fatalf("expected 2 members, got %d", len(members))
		}
		if m := memberByUser(members, memberID); m == nil || m["role"] != "member" {
			t.Fatalf("expected the new member with role member, got %v", members)
		}
	})

	t.Run("project_list_counts_members", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/projects", ownerToken)
		requireStatus(t, resp, body, http.StatusOK)

		var result struct {
			Projects []map[string]interface{} `json:"projects"`
		}
		decodeBody(t, body, &result)
		for _, p := range result.Projects {
			if p["id"] == projectID {
				if p["member_count"] != float64(2) {
					t.Fatalf("expected member_count 2, got %v", p["member_count"])
				}
				return
			}
		}
		t.Fatal("project not found in list")
	})

	t.Run("change_role", func(t *testing.T) {
		resp, body := putJSON(t, base+"/api/v1/projects/"+projectID+"/members/"+memberID, map[string]interface{}{
			"role": "viewer",
		}, ownerToken)
		requireStatus(t, resp, body, http.StatusOK)

		var result struct {
			Member map[string]interface{} `json:"member"`
		}
		decodeBody(t, body, &result)
		if result.Member["user_id"] != memberID || result.Member["role"] != "viewer" {
			t.Fatalf("expected viewer role in the response, got %v", result.Member)
		}

		if m := memberByUser(projectMembers(t, base, projectID, ownerToken), memberID); m == nil || m["role"] != "viewer" {
			t.Fatalf("expected the list to show the new role, got %v", m)
		}
	})

	t.Run("invalid_role_rejected", func(t *testing.T) {
		resp, body := putJSON(t, base+"/api/v1/projects/"+projectID+"/members/"+memberID, map[string]interface{}{
			"role": "emperor",
		}, ownerToken)
		requireStatus(t, resp, body, http.StatusBadRequest)
	})

	t.Run("only_owner_cannot_be_removed", func(t *testing.T) {
		resp, body := deleteJSON(t, base+"/api/v1/projects/"+projectID+"/members/"+ownerID, ownerToken)
		requireStatus(t, resp, body, http.StatusConflict)

		if memberByUser(projectMembers(t, base, projectID, ownerToken), ownerID) == nil {
			t.Fatal("the only owner must still be a member")
		}
	})

	// Open-projects decision: a user with project rights who is not a member
	// still sees the project, its member list and the list entry.
	t.Run("non_member_with_project_rights_sees_project", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/projects/"+projectID, outsiderToken)
		requireStatus(t, resp, body, http.StatusOK)

		if !projectListed(t, base, "", projectID, outsiderToken) {
			t.Fatal("non-member with project rights does not see the project in the list")
		}
		if memberByUser(projectMembers(t, base, projectID, outsiderToken), outsiderID) != nil {
			t.Fatal("the outsider must not appear as a member")
		}
	})

	t.Run("remove_member", func(t *testing.T) {
		resp, body := deleteJSON(t, base+"/api/v1/projects/"+projectID+"/members/"+memberID, ownerToken)
		requireStatus(t, resp, body, http.StatusOK)

		members := projectMembers(t, base, projectID, ownerToken)
		if len(members) != 1 || memberByUser(members, memberID) != nil {
			t.Fatalf("expected only the owner left, got %v", members)
		}
	})

	t.Run("archive_project", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/projects/"+projectID+"/archive", nil, ownerToken)
		requireStatus(t, resp, body, http.StatusOK)

		var result struct {
			Project map[string]interface{} `json:"project"`
		}
		decodeBody(t, body, &result)
		if result.Project["id"] != projectID {
			t.Fatalf("expected the archived project in the response, got %v", result.Project)
		}
		if at, _ := result.Project["archived_at"].(string); at == "" {
			t.Fatalf("expected archived_at to be set, got %v", result.Project["archived_at"])
		}
	})

	t.Run("archived_project_leaves_default_list", func(t *testing.T) {
		if projectListed(t, base, "", projectID, ownerToken) {
			t.Fatal("archived project must not be in the default list")
		}
		if !projectListed(t, base, "?include_archived=true", projectID, ownerToken) {
			t.Fatal("archived project must be in the list with include_archived=true")
		}
	})

	t.Run("archived_project_rejects_new_members", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/projects/"+projectID+"/members", map[string]interface{}{
			"user_id": memberID,
			"role":    "member",
		}, ownerToken)
		requireStatus(t, resp, body, http.StatusConflict)
	})

	t.Run("archive_twice_conflicts", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/projects/"+projectID+"/archive", nil, ownerToken)
		requireStatus(t, resp, body, http.StatusConflict)
	})

	t.Run("delete_project", func(t *testing.T) {
		resp, body := deleteJSON(t, base+"/api/v1/projects/"+projectID, ownerToken)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = getJSON(t, base+"/api/v1/projects/"+projectID, ownerToken)
		requireStatus(t, resp, body, http.StatusNotFound)
	})

	t.Run("no_auth", func(t *testing.T) {
		resp, _ := getJSON(t, base+"/api/v1/projects/"+projectID+"/members", "")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})
}
