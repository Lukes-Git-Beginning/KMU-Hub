//go:build e2e

package e2e

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// inboxArray decodes a body that must be a bare JSON array. The settings panel
// and the routing rule editor map over the response directly, so a wrapping
// object or a null would throw there.
func inboxArray(t *testing.T, body []byte) []map[string]interface{} {
	t.Helper()
	if !bytes.HasPrefix(bytes.TrimSpace(body), []byte("[")) {
		t.Fatalf("expected a bare JSON array, got %s", string(body))
	}
	var items []map[string]interface{}
	decodeBody(t, body, &items)
	return items
}

func inboxItemByID(items []map[string]interface{}, id string) map[string]interface{} {
	for _, it := range items {
		if it["id"] == id {
			return it
		}
	}
	return nil
}

// inboxMemberByUser finds a team member by user_id (members carry no id).
func inboxMemberByUser(items []map[string]interface{}, userID string) map[string]interface{} {
	for _, it := range items {
		if it["user_id"] == userID {
			return it
		}
	}
	return nil
}

func inboxMembers(t *testing.T, base, teamID, token string) []map[string]interface{} {
	t.Helper()
	resp, body := getJSON(t, base+"/api/v1/inbox/teams/"+teamID+"/members", token)
	requireStatus(t, resp, body, http.StatusOK)
	return inboxArray(t, body)
}

func inboxRules(t *testing.T, base, token string) []map[string]interface{} {
	t.Helper()
	resp, body := getJSON(t, base+"/api/v1/inbox/rules", token)
	requireStatus(t, resp, body, http.StatusOK)
	return inboxArray(t, body)
}

// TestInboxTeamAndRoutingFlow covers the team inbox member lifecycle and the
// routing rules the way the settings panel and the rule editor use them.
//
// The second admin is not a member of the inbox the first one created, so every
// step it does is carried by kommunikation:team_inbox:manage alone (decision
// 2026-10-23: the permission manages the members of every team inbox of the
// tenant). The admin role holds that key (migration 256); a plain member does
// not and must be turned away at the gateway.
func TestInboxTeamAndRoutingFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	creatorToken, creatorID := registerAndLoginAdmin(t, base)
	managerToken, _ := registerAndLoginAdmin(t, base)
	memberToken, memberID := registerAndLogin(t, base)

	var teamID, ruleID string
	teamName := fmt.Sprintf("E2E Team Inbox %d", time.Now().UnixNano())

	t.Run("create_team_inbox", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/inbox/teams", map[string]interface{}{
			"name":        teamName,
			"description": "e2e",
		}, creatorToken)
		requireStatus(t, resp, body, http.StatusCreated)

		// The desktop types the body as the bare team inbox, enums as strings.
		var team map[string]interface{}
		decodeBody(t, body, &team)
		teamID, _ = team["id"].(string)
		if teamID == "" {
			t.Fatalf("expected team inbox id, got %s", string(body))
		}
		if team["name"] != teamName {
			t.Errorf("expected name %q, got %v", teamName, team["name"])
		}
		if team["assignment_mode"] != "manual" {
			t.Errorf("expected default assignment_mode manual, got %v", team["assignment_mode"])
		}
		if team["visibility"] != "open" {
			t.Errorf("expected default visibility open, got %v", team["visibility"])
		}
		if team["created_by"] != creatorID {
			t.Errorf("expected created_by %s, got %v", creatorID, team["created_by"])
		}
	})

	t.Run("create_team_inbox_validation", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/inbox/teams", map[string]interface{}{
			"description": "no name",
		}, creatorToken)
		requireStatus(t, resp, body, http.StatusBadRequest)

		resp, body = postJSON(t, base+"/api/v1/inbox/teams", map[string]interface{}{
			"name":            "bad mode",
			"assignment_mode": "random",
		}, creatorToken)
		requireStatus(t, resp, body, http.StatusBadRequest)
	})

	t.Run("creator_is_team_admin", func(t *testing.T) {
		members := inboxMembers(t, base, teamID, creatorToken)
		m := inboxMemberByUser(members, creatorID)
		if m == nil {
			t.Fatalf("creator missing from the member list: %v", members)
		}
		if m["role"] != "admin" {
			t.Errorf("expected creator role admin, got %v", m["role"])
		}
		if m["team_inbox_id"] != teamID {
			t.Errorf("expected team_inbox_id %s, got %v", teamID, m["team_inbox_id"])
		}
	})

	t.Run("list_team_inboxes_is_per_member", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/inbox/teams", creatorToken)
		requireStatus(t, resp, body, http.StatusOK)
		if inboxItemByID(inboxArray(t, body), teamID) == nil {
			t.Errorf("creator does not see its own team inbox: %s", string(body))
		}

		// Listing is by membership: the manager is no member yet. An empty
		// list must still be an array, never null.
		resp, body = getJSON(t, base+"/api/v1/inbox/teams", managerToken)
		requireStatus(t, resp, body, http.StatusOK)
		if inboxItemByID(inboxArray(t, body), teamID) != nil {
			t.Errorf("non-member sees the team inbox in its list: %s", string(body))
		}
	})

	t.Run("member_without_permission_cannot_add_members", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/inbox/teams/"+teamID+"/members", map[string]interface{}{
			"user_id": memberID,
			"role":    "member",
		}, memberToken)
		requireStatus(t, resp, body, http.StatusForbidden)

		if inboxMemberByUser(inboxMembers(t, base, teamID, creatorToken), memberID) != nil {
			t.Error("a rejected request still added the member")
		}
	})

	t.Run("manage_permission_adds_member", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/inbox/teams/"+teamID+"/members", map[string]interface{}{
			"user_id": memberID,
			"role":    "member",
		}, managerToken)
		requireStatus(t, resp, body, http.StatusCreated)

		var m map[string]interface{}
		decodeBody(t, body, &m)
		if m["user_id"] != memberID || m["team_inbox_id"] != teamID {
			t.Errorf("unexpected member in response: %s", string(body))
		}
		if m["role"] != "member" {
			t.Errorf("expected role member, got %v", m["role"])
		}

		got := inboxMemberByUser(inboxMembers(t, base, teamID, creatorToken), memberID)
		if got == nil || got["role"] != "member" {
			t.Errorf("added member missing or wrong role in the list: %v", got)
		}
	})

	t.Run("member_reads_its_team_inbox", func(t *testing.T) {
		// kommunikation:team_inbox:read (decision 2026-10-25): a member reads
		// the team inboxes it belongs to, bounded by membership.
		resp, body := getJSON(t, base+"/api/v1/inbox/teams", memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		if inboxItemByID(inboxArray(t, body), teamID) == nil {
			t.Errorf("member does not see its team inbox: %s", string(body))
		}

		if inboxMemberByUser(inboxMembers(t, base, teamID, memberToken), memberID) == nil {
			t.Error("member does not find itself in its team's member list")
		}

		resp, body = getJSON(t, base+"/api/v1/inbox/messages?team_inbox_id="+teamID, memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var list map[string]interface{}
		decodeBody(t, body, &list)
		if _, ok := list["messages"].([]interface{}); !ok {
			t.Errorf("messages is not an array: %s", string(body))
		}
	})

	t.Run("adding_a_member_twice_conflicts", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/inbox/teams/"+teamID+"/members", map[string]interface{}{
			"user_id": memberID,
			"role":    "member",
		}, managerToken)
		requireStatus(t, resp, body, http.StatusConflict)
	})

	t.Run("add_member_validation", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/inbox/teams/"+teamID+"/members", map[string]interface{}{
			"role": "member",
		}, managerToken)
		requireStatus(t, resp, body, http.StatusBadRequest)

		resp, body = postJSON(t, base+"/api/v1/inbox/teams/"+teamID+"/members", map[string]interface{}{
			"user_id": "not-a-uuid",
			"role":    "member",
		}, managerToken)
		requireStatus(t, resp, body, http.StatusBadRequest)
	})

	t.Run("manage_permission_updates_team_inbox", func(t *testing.T) {
		resp, body := putJSON(t, base+"/api/v1/inbox/teams/"+teamID, map[string]interface{}{
			"assignment_mode": "round_robin",
			"description":     "updated by manager",
		}, managerToken)
		requireStatus(t, resp, body, http.StatusOK)

		var team map[string]interface{}
		decodeBody(t, body, &team)
		if team["assignment_mode"] != "round_robin" {
			t.Errorf("expected assignment_mode round_robin, got %v", team["assignment_mode"])
		}
		if team["name"] != teamName {
			t.Errorf("a partial update changed the name to %v", team["name"])
		}

		resp, body = getJSON(t, base+"/api/v1/inbox/teams", creatorToken)
		requireStatus(t, resp, body, http.StatusOK)
		listed := inboxItemByID(inboxArray(t, body), teamID)
		if listed == nil || listed["assignment_mode"] != "round_robin" {
			t.Errorf("update not visible in the list: %v", listed)
		}
	})

	t.Run("manage_permission_removes_member", func(t *testing.T) {
		resp, body := deleteJSON(t, base+"/api/v1/inbox/teams/"+teamID+"/members/"+memberID, managerToken)
		requireStatus(t, resp, body, http.StatusOK)

		if inboxMemberByUser(inboxMembers(t, base, teamID, creatorToken), memberID) != nil {
			t.Error("removed member is still listed")
		}
	})

	t.Run("removed_member_loses_the_team_inbox", func(t *testing.T) {
		resp, body := getJSON(t, base+"/api/v1/inbox/teams", memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		if inboxItemByID(inboxArray(t, body), teamID) != nil {
			t.Errorf("removed member still lists the team inbox: %s", string(body))
		}

		resp, body = getJSON(t, base+"/api/v1/inbox/messages?team_inbox_id="+teamID, memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var list map[string]interface{}
		decodeBody(t, body, &list)
		if msgs, _ := list["messages"].([]interface{}); len(msgs) != 0 {
			t.Errorf("removed member still reads team messages: %s", string(body))
		}
	})

	t.Run("last_admin_cannot_be_removed", func(t *testing.T) {
		resp, body := deleteJSON(t, base+"/api/v1/inbox/teams/"+teamID+"/members/"+creatorID, managerToken)
		requireStatus(t, resp, body, http.StatusConflict)

		if inboxMemberByUser(inboxMembers(t, base, teamID, creatorToken), creatorID) == nil {
			t.Error("the last admin disappeared from the member list")
		}
	})

	t.Run("create_routing_rule", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/inbox/rules", map[string]interface{}{
			"name":     "E2E Rechnungen",
			"channel":  "email",
			"priority": 10,
			"conditions": map[string]interface{}{
				"field":    "subject",
				"operator": "contains",
				"value":    "Rechnung",
			},
			"actions": []map[string]interface{}{
				{"type": "route_to_team", "config": map[string]interface{}{"team_inbox_id": teamID}},
			},
		}, creatorToken)
		requireStatus(t, resp, body, http.StatusCreated)

		// The desktop types the body as the bare RoutingRule: channel as a
		// name, is_active and priority always present.
		var rule map[string]interface{}
		decodeBody(t, body, &rule)
		ruleID, _ = rule["id"].(string)
		if ruleID == "" {
			t.Fatalf("expected rule id, got %s", string(body))
		}
		if rule["name"] != "E2E Rechnungen" {
			t.Errorf("unexpected name %v", rule["name"])
		}
		if rule["channel"] != "email" {
			t.Errorf("expected channel email, got %v", rule["channel"])
		}
		if rule["is_active"] != true {
			t.Errorf("expected a new rule to be active, got %v", rule["is_active"])
		}
		if rule["priority"] != float64(10) {
			t.Errorf("expected priority 10, got %v", rule["priority"])
		}
		if rule["created_by"] != creatorID {
			t.Errorf("expected created_by %s, got %v", creatorID, rule["created_by"])
		}
		cond, _ := rule["conditions"].(map[string]interface{})
		if cond["field"] != "subject" || cond["operator"] != "contains" || cond["value"] != "Rechnung" {
			t.Errorf("conditions did not round-trip: %v", rule["conditions"])
		}
		actions, _ := rule["actions"].([]interface{})
		if len(actions) != 1 {
			t.Fatalf("expected one action, got %v", rule["actions"])
		}
		action, _ := actions[0].(map[string]interface{})
		cfg, _ := action["config"].(map[string]interface{})
		if action["type"] != "route_to_team" || cfg["team_inbox_id"] != teamID {
			t.Errorf("action did not round-trip: %v", action)
		}
	})

	t.Run("create_routing_rule_validation", func(t *testing.T) {
		cond := map[string]interface{}{"field": "subject", "operator": "contains", "value": "x"}
		valid := []map[string]interface{}{{"type": "add_tags", "config": map[string]interface{}{"tags": []string{"x"}}}}

		resp, body := postJSON(t, base+"/api/v1/inbox/rules", map[string]interface{}{
			"conditions": cond, "actions": valid,
		}, creatorToken)
		requireStatus(t, resp, body, http.StatusBadRequest)

		resp, body = postJSON(t, base+"/api/v1/inbox/rules", map[string]interface{}{
			"name": "unknown action", "conditions": cond,
			"actions": []map[string]interface{}{{"type": "delete_everything", "config": map[string]interface{}{}}},
		}, creatorToken)
		requireStatus(t, resp, body, http.StatusBadRequest)

		resp, body = postJSON(t, base+"/api/v1/inbox/rules", map[string]interface{}{
			"name": "no actions", "conditions": cond,
		}, creatorToken)
		requireStatus(t, resp, body, http.StatusBadRequest)
	})

	t.Run("member_cannot_create_routing_rule", func(t *testing.T) {
		resp, body := postJSON(t, base+"/api/v1/inbox/rules", map[string]interface{}{
			"name":       "not allowed",
			"conditions": map[string]interface{}{"field": "subject", "operator": "contains", "value": "x"},
			"actions":    []map[string]interface{}{{"type": "add_tags", "config": map[string]interface{}{"tags": []string{"x"}}}},
		}, memberToken)
		requireStatus(t, resp, body, http.StatusForbidden)
	})

	t.Run("list_routing_rules_is_a_bare_array", func(t *testing.T) {
		// RoutingRulesEditor maps over the body directly (fix 3c13fd66): a
		// wrapping {rules: [...]} object crashed it with a TypeError.
		rule := inboxItemByID(inboxRules(t, base, creatorToken), ruleID)
		if rule == nil {
			t.Fatal("created rule missing from GET /inbox/rules")
		}
		if rule["name"] != "E2E Rechnungen" || rule["channel"] != "email" || rule["is_active"] != true {
			t.Errorf("listed rule differs from the created one: %v", rule)
		}
		if _, ok := rule["conditions"].(map[string]interface{}); !ok {
			t.Errorf("conditions is not an object: %v", rule["conditions"])
		}
		if _, ok := rule["actions"].([]interface{}); !ok {
			t.Errorf("actions is not an array: %v", rule["actions"])
		}
	})

	t.Run("update_routing_rule", func(t *testing.T) {
		resp, body := putJSON(t, base+"/api/v1/inbox/rules/"+ruleID, map[string]interface{}{
			"is_active": false,
			"priority":  5,
		}, creatorToken)
		requireStatus(t, resp, body, http.StatusOK)

		var rule map[string]interface{}
		decodeBody(t, body, &rule)
		if rule["is_active"] != false {
			t.Errorf("expected is_active false in the response, got %v", rule["is_active"])
		}
		if rule["priority"] != float64(5) {
			t.Errorf("expected priority 5, got %v", rule["priority"])
		}
		if rule["name"] != "E2E Rechnungen" {
			t.Errorf("a partial update changed the name to %v", rule["name"])
		}

		// is_active=false must survive the read: protojson drops false, the
		// wire shape adds it back, the editor reads it as a switch.
		listed := inboxItemByID(inboxRules(t, base, creatorToken), ruleID)
		if listed == nil {
			t.Fatal("rule missing after update")
		}
		if listed["is_active"] != false {
			t.Errorf("expected is_active=false in the list, got %v", listed["is_active"])
		}
	})

	t.Run("update_unknown_routing_rule_is_404", func(t *testing.T) {
		resp, body := putJSON(t, base+"/api/v1/inbox/rules/00000000-0000-4000-8000-000000000001",
			map[string]interface{}{"priority": 1}, creatorToken)
		requireStatus(t, resp, body, http.StatusNotFound)
	})

	t.Run("test_routing_rule", func(t *testing.T) {
		cond := map[string]interface{}{"field": "subject", "operator": "contains", "value": "Rechnung"}

		resp, body := postJSON(t, base+"/api/v1/inbox/rules/test", map[string]interface{}{
			"conditions": cond,
			"message":    map[string]interface{}{"channel": "email", "subject": "Rechnung 4711", "sender_name": "Kunde"},
		}, creatorToken)
		requireStatus(t, resp, body, http.StatusOK)
		var hit map[string]interface{}
		decodeBody(t, body, &hit)
		if hit["matches"] != true {
			t.Errorf("expected matches=true, got %s", string(body))
		}

		// matches:false must stay in the body, the editor reads the key.
		resp, body = postJSON(t, base+"/api/v1/inbox/rules/test", map[string]interface{}{
			"conditions": cond,
			"message":    map[string]interface{}{"channel": "email", "subject": "Hallo", "sender_name": "Kunde"},
		}, creatorToken)
		requireStatus(t, resp, body, http.StatusOK)
		var miss map[string]interface{}
		decodeBody(t, body, &miss)
		if v, present := miss["matches"]; !present || v != false {
			t.Errorf("expected matches=false to be present, got %s", string(body))
		}
	})

	t.Run("delete_routing_rule", func(t *testing.T) {
		resp, body := deleteJSON(t, base+"/api/v1/inbox/rules/"+ruleID, creatorToken)
		requireStatus(t, resp, body, http.StatusOK)

		if inboxItemByID(inboxRules(t, base, creatorToken), ruleID) != nil {
			t.Error("deleted rule is still listed")
		}

		resp, body = deleteJSON(t, base+"/api/v1/inbox/rules/"+ruleID, creatorToken)
		requireStatus(t, resp, body, http.StatusNotFound)
	})

	t.Run("delete_team_inbox", func(t *testing.T) {
		resp, body := deleteJSON(t, base+"/api/v1/inbox/teams/"+teamID, creatorToken)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = getJSON(t, base+"/api/v1/inbox/teams/"+teamID+"/members", creatorToken)
		requireStatus(t, resp, body, http.StatusNotFound)
	})

	t.Run("no_auth", func(t *testing.T) {
		for _, path := range []string{"/api/v1/inbox/teams", "/api/v1/inbox/rules"} {
			resp, body := getJSON(t, base+path, "")
			requireStatus(t, resp, body, http.StatusUnauthorized)
		}
	})
}
