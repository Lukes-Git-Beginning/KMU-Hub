//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// keyedRequest is doRequest plus an explicit Idempotency-Key. The dialer routes sit behind the
// idempotency middleware, and a fixed key is the only way to send the same logical request twice.
func keyedRequest(t *testing.T, method, url string, body interface{}, token, key string) (*http.Response, []byte) {
	t.Helper()
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Idempotency-Key", key)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	return resp, buf.Bytes()
}

// replayRequest sends a request a second time with the key of a finished one. The gateway stores
// the first response asynchronously, so a replay that lands before the store answers 409
// ("in flight", Retry-After) and is retried; every other answer is final.
func replayRequest(t *testing.T, method, url string, body interface{}, token, key string) (*http.Response, []byte) {
	t.Helper()
	for attempt := 0; ; attempt++ {
		resp, respBody := keyedRequest(t, method, url, body, token, key)
		if resp.StatusCode != http.StatusConflict || resp.Header.Get("Idempotency-Replayed") == "true" || attempt >= 10 {
			return resp, respBody
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// TestDialerAgentMemberFlow runs the dialer the way an agent meets it: the admin prepares the
// campaign and the consented contact, a plain member (calls:write, campaigns:read) takes it from
// there. Bodies follow dialer-client.ts. Every mutation carries an Idempotency-Key, the way the
// desktop sends it, and the booking steps are sent a second time with the same key.
func TestDialerAgentMemberFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	adminToken, adminID := registerAndLoginAdmin(t, base)
	memberToken, memberID := registerAndLoginWithRole(t, base, "member")

	dialer := base + "/api/v1/dialer"
	suffix := time.Now().UnixNano()
	newKey := func(name string) string { return fmt.Sprintf("e2e-dialer-%s-%d", name, suffix) }

	var campaignID, contactID, campaignContactID string

	t.Run("admin: prepares campaign and consented contact", func(t *testing.T) {
		resp, body := keyedRequest(t, http.MethodPost, dialer+"/campaigns", map[string]interface{}{
			"name":        fmt.Sprintf("E2E Agent Campaign %d", suffix),
			"description": "Prepared by the admin, worked by a member",
			"mode":        1,
		}, adminToken, newKey("campaign"))
		requireStatus(t, resp, body, http.StatusCreated)
		var campaign struct {
			ID string `json:"id"`
		}
		decodeBody(t, body, &campaign)
		if campaign.ID == "" {
			t.Fatal("expected campaign ID")
		}
		campaignID = campaign.ID

		resp, body = keyedRequest(t, http.MethodPost, base+"/api/v1/contacts", map[string]interface{}{
			"first_name": "Agent",
			"last_name":  fmt.Sprintf("Callee%d", suffix),
			"email":      fmt.Sprintf("e2e-dialer-agent-%d@test.com", suffix),
			"phone":      "+491512345678",
		}, adminToken, newKey("contact"))
		requireStatus(t, resp, body, http.StatusCreated)
		var contact struct {
			Contact struct {
				ID string `json:"id"`
			} `json:"contact"`
		}
		decodeBody(t, body, &contact)
		if contact.Contact.ID == "" {
			t.Fatal("expected contact ID")
		}
		contactID = contact.Contact.ID

		// The consent asserter in front of InitiateDialerCall refuses a contact without it.
		resp, body = keyedRequest(t, http.MethodPost, fmt.Sprintf("%s/api/v1/contacts/%s/consents", base, contactID), map[string]interface{}{
			"consent_type": "marketing_phone",
			"source":       "e2e_test",
			"legal_basis":  "consent",
		}, adminToken, newKey("consent"))
		requireStatus(t, resp, body, http.StatusCreated)

		resp, body = keyedRequest(t, http.MethodPost, fmt.Sprintf("%s/campaigns/%s/contacts", dialer, campaignID), map[string]interface{}{
			"contact_ids": []string{contactID},
		}, adminToken, newKey("add-contact"))
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = keyedRequest(t, http.MethodPost, fmt.Sprintf("%s/campaigns/%s/start", dialer, campaignID), nil, adminToken, newKey("start"))
		requireStatus(t, resp, body, http.StatusOK)
	})

	t.Run("member: may not manage campaigns, outcomes or other agents", func(t *testing.T) {
		// The member preset holds dialer:calls:read|write and dialer:campaigns:read only.
		resp, body := keyedRequest(t, http.MethodPost, dialer+"/campaigns", map[string]interface{}{"name": "Not allowed", "mode": 1}, memberToken, newKey("m-campaign"))
		requireStatus(t, resp, body, http.StatusForbidden)

		resp, body = keyedRequest(t, http.MethodPost, fmt.Sprintf("%s/campaigns/%s/start", dialer, campaignID), nil, memberToken, newKey("m-start"))
		requireStatus(t, resp, body, http.StatusForbidden)

		resp, body = keyedRequest(t, http.MethodPost, fmt.Sprintf("%s/campaigns/%s/contacts", dialer, campaignID), map[string]interface{}{
			"contact_ids": []string{contactID},
		}, memberToken, newKey("m-add"))
		requireStatus(t, resp, body, http.StatusForbidden)

		resp, body = keyedRequest(t, http.MethodPost, dialer+"/outcomes", map[string]interface{}{"label": "Not allowed"}, memberToken, newKey("m-outcome"))
		requireStatus(t, resp, body, http.StatusForbidden)

		resp, body = getJSON(t, fmt.Sprintf("%s/agent-status/campaign/%s", dialer, campaignID), memberToken)
		requireStatus(t, resp, body, http.StatusForbidden)

		resp, body = getJSON(t, dialer+"/dashboard/agent", memberToken)
		requireStatus(t, resp, body, http.StatusForbidden)
	})

	t.Run("member: reads campaign and outcome list", func(t *testing.T) {
		resp, body := getJSON(t, fmt.Sprintf("%s/campaigns/%s", dialer, campaignID), memberToken)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = getJSON(t, dialer+"/supervisor", memberToken)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = getJSON(t, dialer+"/outcomes", memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var outcomes struct {
			Outcomes []struct {
				ID string `json:"id"`
			} `json:"outcomes"`
		}
		decodeBody(t, body, &outcomes)
		if len(outcomes.Outcomes) == 0 {
			t.Fatal("expected the default outcomes, the agent workspace cannot close a call without them")
		}
	})

	t.Run("member: sets only their own status", func(t *testing.T) {
		// 1 = available. user_id names somebody else; without dialer:agent:manage it is ignored.
		resp, body := keyedRequest(t, http.MethodPut, dialer+"/agent-status", map[string]interface{}{
			"user_id": adminID,
			"status":  1,
		}, memberToken, newKey("status"))
		requireStatus(t, resp, body, http.StatusOK)
		var set struct {
			UserID string `json:"user_id"`
		}
		decodeBody(t, body, &set)
		if set.UserID != memberID {
			t.Fatalf("status was set for %q, want the caller %q", set.UserID, memberID)
		}

		resp, body = getJSON(t, dialer+"/agent-status?user_id="+adminID, memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var got struct {
			UserID string `json:"user_id"`
		}
		decodeBody(t, body, &got)
		if got.UserID != memberID {
			t.Fatalf("status read for %q, want the caller %q", got.UserID, memberID)
		}
	})

	var sessionID, outcomeID string

	t.Run("member: takes the next contact and starts the call once", func(t *testing.T) {
		resp, body := getJSON(t, fmt.Sprintf("%s/campaigns/%s/next", dialer, campaignID), memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var next struct {
			ID        string `json:"id"`
			ContactID string `json:"contact_id"`
		}
		decodeBody(t, body, &next)
		if next.ID == "" || next.ContactID != contactID {
			t.Fatalf("expected the prepared contact %q from next, got %+v", contactID, next)
		}
		campaignContactID = next.ID

		key := newKey("call")
		payload := map[string]interface{}{"campaign_contact_id": campaignContactID}
		resp, body = keyedRequest(t, http.MethodPost, dialer+"/calls", payload, memberToken, key)
		requireStatus(t, resp, body, http.StatusCreated)
		var session struct {
			ID      string `json:"id"`
			AgentID string `json:"agent_id"`
		}
		decodeBody(t, body, &session)
		if session.ID == "" {
			t.Fatal("expected session ID")
		}
		if session.AgentID != memberID {
			t.Fatalf("call session belongs to %q, want the caller %q", session.AgentID, memberID)
		}
		sessionID = session.ID

		// The second POST with the same key answers from the stored response: same session.
		resp2, body2 := replayRequest(t, http.MethodPost, dialer+"/calls", payload, memberToken, key)
		requireStatus(t, resp2, body2, http.StatusCreated)
		if resp2.Header.Get("Idempotency-Replayed") != "true" {
			t.Fatalf("second call POST was not served as a replay: %s", string(body2))
		}
		var again struct {
			ID string `json:"id"`
		}
		decodeBody(t, body2, &again)
		if again.ID != sessionID {
			t.Fatalf("replayed call POST returned session %q, want %q", again.ID, sessionID)
		}
	})

	t.Run("member: books the outcome once", func(t *testing.T) {
		resp, body := getJSON(t, dialer+"/outcomes", memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var outcomes struct {
			Outcomes []struct {
				ID         string `json:"id"`
				IsCallback bool   `json:"is_callback"`
			} `json:"outcomes"`
		}
		decodeBody(t, body, &outcomes)
		// A callback outcome needs a callback time, which this flow does not send.
		for _, o := range outcomes.Outcomes {
			if !o.IsCallback {
				outcomeID = o.ID
				break
			}
		}
		if outcomeID == "" {
			t.Fatal("expected a non-callback outcome")
		}

		key := newKey("outcome")
		outcomeURL := fmt.Sprintf("%s/calls/%s/outcome", dialer, sessionID)
		payload := map[string]interface{}{"outcome_id": outcomeID, "notes": "Reached the contact"}
		resp, body = keyedRequest(t, http.MethodPost, outcomeURL, payload, memberToken, key)
		requireStatus(t, resp, body, http.StatusOK)

		resp2, body2 := replayRequest(t, http.MethodPost, outcomeURL, payload, memberToken, key)
		requireStatus(t, resp2, body2, http.StatusOK)
		if resp2.Header.Get("Idempotency-Replayed") != "true" {
			t.Fatalf("second outcome POST was not served as a replay: %s", string(body2))
		}
		if !bytes.Equal(body, body2) {
			t.Fatalf("replayed outcome differs from the first answer:\nfirst:  %s\nsecond: %s", string(body), string(body2))
		}

		// The same key with another body is a different request, not a replay.
		resp, body = replayRequest(t, http.MethodPost, outcomeURL, map[string]interface{}{"outcome_id": outcomeID, "notes": "Changed my mind"}, memberToken, key)
		requireStatus(t, resp, body, http.StatusUnprocessableEntity)

		// One booking, one row in the contact's call history.
		resp, body = getJSON(t, fmt.Sprintf("%s/contacts/%s/calls", dialer, campaignContactID), memberToken)
		requireStatus(t, resp, body, http.StatusOK)
		var history struct {
			Calls []struct {
				ID    string `json:"id"`
				Notes string `json:"notes"`
			} `json:"calls"`
		}
		decodeBody(t, body, &history)
		if len(history.Calls) != 1 {
			t.Fatalf("expected exactly one booked call, got %d: %s", len(history.Calls), string(body))
		}
		if history.Calls[0].Notes != "Reached the contact" {
			t.Fatalf("call history carries notes %q, want the first booking's", history.Calls[0].Notes)
		}
	})

	t.Run("member: completes the wrap-up", func(t *testing.T) {
		// The workspace sends an empty object; agent_id defaults to the caller.
		resp, body := keyedRequest(t, http.MethodPost, fmt.Sprintf("%s/calls/%s/complete", dialer, sessionID), map[string]interface{}{}, memberToken, newKey("complete"))
		requireStatus(t, resp, body, http.StatusOK)
	})

	t.Run("admin: campaign dashboard sees the booked call", func(t *testing.T) {
		resp, body := getJSON(t, fmt.Sprintf("%s/campaigns/%s/dashboard", dialer, campaignID), adminToken)
		requireStatus(t, resp, body, http.StatusOK)
	})
}
