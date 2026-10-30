//go:build e2e

package e2e

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestFormulareMemberFlow runs the forms module against the real seed with the bodies the desktop sends
// (formulare-client.ts, FormularePage.tsx): the manager builds a public form, publishes it, mints and revokes a
// share link and a webhook; members fill the form out, read the submissions and export them. Needs the
// formulare service in the stack.
//
// IST of the seed that the test pins and does not judge:
//   - member has `formulare:schemas|submissions:read`, `formulare:submissions:write` and the legacy
//     `formulare:webhooks:read` (000234, 000256) and nothing else: schema writes, share links and webhook
//     writes are 403; manager has the whole module;
//   - reads are tenant-wide for member (no own-scope): a second member lists, reads, exports and re-labels the
//     submissions of the first, draft forms included;
//   - `submittedBy` in the member's body is taken as sent: a member can file a submission under another name;
//   - the public redemption reads `answers` as bytes, so a JSON object is a 400 and only the base64 string
//     passes (fix-formulare-public-submit-accepts-json-answers);
//   - the share link is a plain `link` (the proto has no channel), a revoked link reads `disabled`, a private
//     form refuses a link with 409, and the desktop's toggle `PATCH /share-links/{id}` has no route.
//
// There is no second tenant in the stack, so a foreign-tenant read is not part of this flow.
func TestFormulareMemberFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	managerToken, managerID := registerAndLoginWithRole(t, base, "manager")
	memberAToken, memberAID := registerAndLoginWithRole(t, base, "member")
	memberBToken, _ := registerAndLoginWithRole(t, base, "member")

	fo := base + "/api/v1/formulare"
	suffix := time.Now().UnixNano()
	title := fmt.Sprintf("E2E Beschwerde %d", suffix)

	patch := func(t *testing.T, url string, body interface{}, token string) (*http.Response, []byte) {
		t.Helper()
		return doRequest(t, http.MethodPatch, url, body, token)
	}
	obj := func(t *testing.T, body []byte) map[string]interface{} {
		t.Helper()
		var out map[string]interface{}
		decodeBody(t, body, &out)
		return out
	}
	str := func(m map[string]interface{}, key string) string {
		s, _ := m[key].(string)
		return s
	}
	// items decodes a wrapped list; `null` is the break the desktop would not survive.
	items := func(t *testing.T, url, token string) ([]map[string]interface{}, int) {
		t.Helper()
		resp, body := getJSON(t, url, token)
		requireStatus(t, resp, body, http.StatusOK)
		var raw map[string]json.RawMessage
		decodeBody(t, body, &raw)
		if strings.TrimSpace(string(raw["items"])) == "null" {
			t.Fatalf("items must never be null, got %s", string(body))
		}
		var out []map[string]interface{}
		decodeBody(t, raw["items"], &out)
		total := len(out)
		if tr, ok := raw["total"]; ok {
			decodeBody(t, tr, &total)
		}
		return out, total
	}
	// bare decodes a list the desktop reads as a bare array (share links, webhooks).
	bare := func(t *testing.T, url, token string) []map[string]interface{} {
		t.Helper()
		resp, body := getJSON(t, url, token)
		requireStatus(t, resp, body, http.StatusOK)
		if strings.TrimSpace(string(body)) == "null" {
			t.Fatalf("a bare list must never be null")
		}
		var out []map[string]interface{}
		decodeBody(t, body, &out)
		return out
	}
	// publicSubmit redeems a token without any credential. The gateway reads `answers` as bytes, so the answers
	// travel base64-encoded.
	publicSubmit := func(t *testing.T, token string, answers map[string]interface{}) (*http.Response, []byte) {
		t.Helper()
		raw, err := json.Marshal(answers)
		if err != nil {
			t.Fatalf("marshal answers: %v", err)
		}
		return postJSON(t, base+"/api/v1/public/formulare/submit/"+token,
			map[string]string{"answers": base64.StdEncoding.EncodeToString(raw)}, "")
	}

	var schemaID, privateID, shareID, shareToken, webhookID, submissionID string
	schemaURL := func() string { return fo + "/schemas/" + schemaID }

	t.Run("manager creates a public form with the desktop's body, member cannot", func(t *testing.T) {
		body := map[string]interface{}{
			"title": title, "description": "Beschwerden aus dem E2E-Lauf",
			"fields": []map[string]interface{}{
				{"id": "f1", "type": "text", "label": "Name", "required": true},
				{"id": "f2", "type": "textarea", "label": "Anliegen", "required": false},
			},
			"status": "draft", "isTemplate": false, "isPublic": true, "pageCount": 1,
		}
		resp, b := postJSON(t, fo+"/schemas", body, memberAToken)
		requireStatus(t, resp, b, http.StatusForbidden)

		resp, b = postJSON(t, fo+"/schemas", body, managerToken)
		requireStatus(t, resp, b, http.StatusCreated)
		s := obj(t, b)
		schemaID = str(s, "id")
		if schemaID == "" || str(s, "title") != title || str(s, "status") != "draft" || str(s, "tenantId") == "" {
			t.Fatalf("unexpected created schema: %s", string(b))
		}
		if s["isPublic"] != true {
			t.Fatalf("isPublic lost: %s", string(b))
		}
		if fields, ok := s["fields"].([]interface{}); !ok || len(fields) != 2 {
			t.Fatalf("fields must come back as the two-element array, got %s", string(b))
		}

		// A private form, kept for the share-link refusal below.
		resp, b = postJSON(t, fo+"/schemas", map[string]interface{}{
			"title": title + " intern", "fields": []map[string]interface{}{}, "status": "active", "isPublic": false,
		}, managerToken)
		requireStatus(t, resp, b, http.StatusCreated)
		privateID = str(obj(t, b), "id")

		for name, bad := range map[string]map[string]interface{}{
			"no title":      {"title": "", "fields": []interface{}{}},
			"closed status": {"title": "x", "status": "closed"},
		} {
			resp, b = postJSON(t, fo+"/schemas", bad, managerToken)
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("%s: expected 400, got %d: %s", name, resp.StatusCode, string(b))
			}
		}
	})

	t.Run("members read the form list and the form", func(t *testing.T) {
		for _, token := range []string{memberAToken, memberBToken} {
			rows, total := items(t, fo+"/schemas?search="+fmt.Sprint(suffix), token)
			if len(rows) != 2 || total != 2 {
				t.Fatalf("expected the two forms of this run, got %d rows / total %d", len(rows), total)
			}
			resp, b := getJSON(t, schemaURL(), token)
			requireStatus(t, resp, b, http.StatusOK)
			if str(obj(t, b), "id") != schemaID {
				t.Fatalf("unexpected form for member: %s", string(b))
			}
		}
		resp, b := getJSON(t, fo+"/schemas/00000000-0000-4000-8000-000000000001", memberAToken)
		requireStatus(t, resp, b, http.StatusNotFound)
		resp, b = getJSON(t, fo+"/schemas/not-a-uuid", memberAToken)
		requireStatus(t, resp, b, http.StatusBadRequest)
	})

	t.Run("member cannot edit, publish, duplicate, delete, share or wire up webhooks", func(t *testing.T) {
		resp, b := patch(t, schemaURL(), map[string]interface{}{"status": "active"}, memberAToken)
		requireStatus(t, resp, b, http.StatusForbidden)
		resp, b = patch(t, schemaURL(), map[string]interface{}{"title": "gekapert"}, memberAToken)
		requireStatus(t, resp, b, http.StatusForbidden)
		resp, b = postJSON(t, schemaURL()+"/duplicate", map[string]string{"title": "Kopie"}, memberAToken)
		requireStatus(t, resp, b, http.StatusForbidden)
		resp, b = deleteJSON(t, schemaURL(), memberAToken)
		requireStatus(t, resp, b, http.StatusForbidden)

		resp, b = getJSON(t, schemaURL()+"/share-links", memberAToken)
		requireStatus(t, resp, b, http.StatusForbidden)
		resp, b = postJSON(t, schemaURL()+"/share-links", map[string]interface{}{"channel": "link", "expiresAt": nil, "maxSubmissions": nil}, memberAToken)
		requireStatus(t, resp, b, http.StatusForbidden)

		resp, b = postJSON(t, schemaURL()+"/webhooks", map[string]interface{}{"url": "https://example.com/hook", "events": []string{"submission.created"}}, memberAToken)
		requireStatus(t, resp, b, http.StatusForbidden)
		resp, b = getJSON(t, fo+"/deliveries", memberAToken)
		requireStatus(t, resp, b, http.StatusOK)

		// Nothing changed.
		resp, b = getJSON(t, schemaURL(), managerToken)
		requireStatus(t, resp, b, http.StatusOK)
		if s := obj(t, b); str(s, "title") != title || str(s, "status") != "draft" {
			t.Fatalf("a refused member write changed the form: %s", string(b))
		}
	})

	t.Run("manager publishes the form, duplicates one and deletes the copy", func(t *testing.T) {
		resp, b := patch(t, schemaURL(), map[string]interface{}{"status": "active"}, managerToken)
		requireStatus(t, resp, b, http.StatusOK)
		if str(obj(t, b), "status") != "active" {
			t.Fatalf("publish did not stick: %s", string(b))
		}
		resp, b = patch(t, schemaURL(), map[string]interface{}{"status": "closed"}, managerToken)
		requireStatus(t, resp, b, http.StatusBadRequest)

		resp, b = postJSON(t, schemaURL()+"/duplicate", map[string]string{"title": title + " Kopie"}, managerToken)
		requireStatus(t, resp, b, http.StatusCreated)
		copyID := str(obj(t, b), "id")
		if copyID == "" || copyID == schemaID {
			t.Fatalf("expected a new form, got %s", string(b))
		}
		resp, b = deleteJSON(t, fo+"/schemas/"+copyID, managerToken)
		requireStatus(t, resp, b, http.StatusNoContent)
		resp, b = getJSON(t, fo+"/schemas/"+copyID, memberAToken)
		requireStatus(t, resp, b, http.StatusNotFound)
	})

	t.Run("manager adds a webhook, member sees it with a masked secret and cannot touch it", func(t *testing.T) {
		resp, b := postJSON(t, schemaURL()+"/webhooks", map[string]interface{}{
			"url": "https://example.com/hook", "secret": "supersecret-1234", "events": []string{"submission.created"}, "active": true,
		}, managerToken)
		requireStatus(t, resp, b, http.StatusCreated)
		webhookID = str(obj(t, b), "id")
		if webhookID == "" {
			t.Fatalf("expected a webhook id, got %s", string(b))
		}

		hooks := bare(t, schemaURL()+"/webhooks", memberAToken)
		if len(hooks) != 1 || str(hooks[0], "id") != webhookID {
			t.Fatalf("member should see the one webhook, got %v", hooks)
		}
		if strings.Contains(str(hooks[0], "secret"), "supersecret") {
			t.Fatalf("the secret must be masked for readers, got %q", str(hooks[0], "secret"))
		}

		resp, b = patch(t, fo+"/webhooks/"+webhookID, map[string]interface{}{"active": false}, memberAToken)
		requireStatus(t, resp, b, http.StatusForbidden)
		resp, b = deleteJSON(t, fo+"/webhooks/"+webhookID, memberAToken)
		requireStatus(t, resp, b, http.StatusForbidden)
	})

	t.Run("member A fills the form out, member B sees it", func(t *testing.T) {
		resp, b := postJSON(t, schemaURL()+"/submissions", map[string]interface{}{
			"answers": map[string]interface{}{"f1": "Anna", "f2": "Der Kaffee ist alle"},
		}, memberAToken)
		requireStatus(t, resp, b, http.StatusCreated)
		sub := obj(t, b)
		submissionID = str(sub, "id")
		if submissionID == "" || str(sub, "status") != "new" || str(sub, "submittedBy") != memberAID || str(sub, "formSchemaId") != schemaID {
			t.Fatalf("unexpected submission: %s", string(b))
		}
		if answers, _ := sub["answers"].(map[string]interface{}); answers["f1"] != "Anna" {
			t.Fatalf("answers lost: %s", string(b))
		}

		// `submittedBy` is trusted as sent.
		resp, b = postJSON(t, schemaURL()+"/submissions", map[string]interface{}{
			"answers": map[string]interface{}{"f1": "Unterschoben"}, "submittedBy": managerID,
		}, memberAToken)
		requireStatus(t, resp, b, http.StatusCreated)
		if got := str(obj(t, b), "submittedBy"); got != managerID {
			t.Fatalf("IST: submittedBy is taken as sent (%s), got %q", managerID, got)
		}

		for _, token := range []string{memberBToken, managerToken} {
			rows, total := items(t, schemaURL()+"/submissions", token)
			if len(rows) != 2 || total != 2 {
				t.Fatalf("expected both submissions, got %d / %d", len(rows), total)
			}
			resp, b = getJSON(t, fo+"/submissions/"+submissionID, token)
			requireStatus(t, resp, b, http.StatusOK)
			if str(obj(t, b), "submittedBy") != memberAID {
				t.Fatalf("unexpected submission for reader: %s", string(b))
			}
		}

		resp, b = postJSON(t, fo+"/schemas/00000000-0000-4000-8000-000000000001/submissions", map[string]interface{}{"answers": map[string]interface{}{"f1": "x"}}, memberAToken)
		requireStatus(t, resp, b, http.StatusNotFound)
		resp, b = getJSON(t, fo+"/submissions/00000000-0000-4000-8000-000000000001", memberAToken)
		requireStatus(t, resp, b, http.StatusNotFound)
	})

	t.Run("submission status, stats and export as member", func(t *testing.T) {
		resp, b := patch(t, fo+"/submissions/"+submissionID, map[string]string{"status": "read"}, memberBToken)
		requireStatus(t, resp, b, http.StatusOK)
		if str(obj(t, b), "status") != "read" {
			t.Fatalf("status did not change: %s", string(b))
		}
		resp, b = patch(t, fo+"/submissions/"+submissionID, map[string]string{"status": "bogus"}, memberBToken)
		requireStatus(t, resp, b, http.StatusBadRequest)
		resp, b = patch(t, fo+"/submissions/00000000-0000-4000-8000-000000000001", map[string]string{"status": "read"}, memberBToken)
		requireStatus(t, resp, b, http.StatusNotFound)

		resp, b = getJSON(t, schemaURL()+"/stats", memberAToken)
		requireStatus(t, resp, b, http.StatusOK)

		resp, b = getJSON(t, schemaURL()+"/submissions/export?format=csv", memberAToken)
		requireStatus(t, resp, b, http.StatusOK)
		if !strings.Contains(resp.Header.Get("Content-Type"), "csv") || !strings.Contains(string(b), "Anna") {
			t.Fatalf("expected a CSV with the answers, got %q: %s", resp.Header.Get("Content-Type"), string(b))
		}
	})

	t.Run("share link: manager mints, member cannot, a private form has none", func(t *testing.T) {
		resp, b := postJSON(t, schemaURL()+"/share-links", map[string]interface{}{"channel": "link", "expiresAt": nil, "maxSubmissions": nil}, managerToken)
		requireStatus(t, resp, b, http.StatusCreated)
		link := obj(t, b)
		shareID, shareToken = str(link, "id"), str(link, "token")
		if shareID == "" || shareToken == "" || str(link, "status") != "active" || str(link, "formSchemaId") != schemaID {
			t.Fatalf("unexpected share link: %s", string(b))
		}
		if link["expiresAt"] == nil {
			t.Fatalf("a link must carry an expiry, got %s", string(b))
		}

		links := bare(t, schemaURL()+"/share-links", managerToken)
		if len(links) != 1 || str(links[0], "id") != shareID {
			t.Fatalf("expected the one link, got %v", links)
		}

		past := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
		resp, b = postJSON(t, schemaURL()+"/share-links", map[string]interface{}{"channel": "link", "expiresAt": past}, managerToken)
		requireStatus(t, resp, b, http.StatusBadRequest)
		resp, b = postJSON(t, schemaURL()+"/share-links", map[string]interface{}{"channel": "link", "maxSubmissions": -1}, managerToken)
		requireStatus(t, resp, b, http.StatusBadRequest)

		resp, b = postJSON(t, fo+"/schemas/"+privateID+"/share-links", map[string]interface{}{"channel": "link"}, managerToken)
		requireStatus(t, resp, b, http.StatusConflict)
	})

	t.Run("public fill-out through the link: object body, base64 body, unknown token", func(t *testing.T) {
		// The natural body, a JSON object, is stored (the base64 string still works, see publicSubmit below).
		resp, b := postJSON(t, base+"/api/v1/public/formulare/submit/"+shareToken,
			map[string]interface{}{"answers": map[string]interface{}{"f1": "Extern-Objekt"}}, "")
		requireStatus(t, resp, b, http.StatusCreated)
		resp, b = postJSON(t, base+"/api/v1/public/formulare/submit/"+shareToken,
			map[string]interface{}{"answers": []interface{}{"not", "an", "object"}}, "")
		requireStatus(t, resp, b, http.StatusBadRequest)

		resp, b = publicSubmit(t, shareToken, map[string]interface{}{"f1": "Extern", "f2": "über den Link"})
		requireStatus(t, resp, b, http.StatusCreated)
		wire := obj(t, b)
		if str(wire, "submission_id") == "" || len(wire) != 1 {
			t.Fatalf("the public answer must carry the submission id and nothing else, got %s", string(b))
		}

		resp, b = publicSubmit(t, "this-token-does-not-exist", map[string]interface{}{"f1": "x"})
		requireStatus(t, resp, b, http.StatusNotFound)

		rows, total := items(t, schemaURL()+"/submissions", managerToken)
		want := 4
		if len(rows) != want || total != want {
			t.Fatalf("expected %d stored submissions after the public fill-out, got %d / %d", want, len(rows), total)
		}
		var external map[string]interface{}
		for _, r := range rows {
			if answers, _ := r["answers"].(map[string]interface{}); answers["f1"] == "Extern" {
				external = r
			}
		}
		if external == nil || external["submittedBy"] != nil {
			t.Fatalf("the public submission should be stored without a submitter, got %v", external)
		}

		links := bare(t, schemaURL()+"/share-links", managerToken)
		if n, _ := links[0]["submissions"].(float64); n < 1 {
			t.Fatalf("the link did not count the redemption: %v", links[0])
		}
	})

	t.Run("a link with a quota stops after its last submission", func(t *testing.T) {
		resp, b := postJSON(t, schemaURL()+"/share-links", map[string]interface{}{"channel": "link", "maxSubmissions": 1}, managerToken)
		requireStatus(t, resp, b, http.StatusCreated)
		token := str(obj(t, b), "token")

		resp, b = publicSubmit(t, token, map[string]interface{}{"f1": "erste"})
		requireStatus(t, resp, b, http.StatusCreated)
		resp, b = publicSubmit(t, token, map[string]interface{}{"f1": "zweite"})
		requireStatus(t, resp, b, http.StatusNotFound)
	})

	t.Run("manager revokes the link, member cannot, the token is dead", func(t *testing.T) {
		resp, b := deleteJSON(t, fo+"/share-links/"+shareID, memberAToken)
		requireStatus(t, resp, b, http.StatusForbidden)

		resp, b = deleteJSON(t, fo+"/share-links/"+shareID, managerToken)
		requireStatus(t, resp, b, http.StatusOK)
		resp, b = deleteJSON(t, fo+"/share-links/"+shareID, managerToken)
		requireStatus(t, resp, b, http.StatusOK)
		resp, b = deleteJSON(t, fo+"/share-links/00000000-0000-4000-8000-000000000001", managerToken)
		requireStatus(t, resp, b, http.StatusNotFound)

		resp, b = publicSubmit(t, shareToken, map[string]interface{}{"f1": "zu spät"})
		requireStatus(t, resp, b, http.StatusNotFound)

		var revoked map[string]interface{}
		for _, l := range bare(t, schemaURL()+"/share-links", managerToken) {
			if str(l, "id") == shareID {
				revoked = l
			}
		}
		if revoked == nil {
			t.Fatalf("a revoked link must stay in the list")
		}
		if st := str(revoked, "status"); st != "disabled" {
			t.Fatalf("a revoked link must read as the desktop's `disabled`, got %q", st)
		}
	})

	t.Run("taking the form private kills a link that was already handed out", func(t *testing.T) {
		resp, b := postJSON(t, schemaURL()+"/share-links", map[string]interface{}{"channel": "link"}, managerToken)
		requireStatus(t, resp, b, http.StatusCreated)
		token := str(obj(t, b), "token")

		resp, b = publicSubmit(t, token, map[string]interface{}{"f1": "noch offen"})
		requireStatus(t, resp, b, http.StatusCreated)

		resp, b = patch(t, schemaURL(), map[string]interface{}{"isPublic": false}, managerToken)
		requireStatus(t, resp, b, http.StatusOK)
		resp, b = publicSubmit(t, token, map[string]interface{}{"f1": "jetzt zu"})
		requireStatus(t, resp, b, http.StatusNotFound)
	})

	t.Run("manager deletes the form: reads and submissions answer 404", func(t *testing.T) {
		resp, b := patch(t, schemaURL(), map[string]interface{}{"status": "archived"}, managerToken)
		requireStatus(t, resp, b, http.StatusOK)
		resp, b = postJSON(t, schemaURL()+"/submissions", map[string]interface{}{"answers": map[string]interface{}{"f1": "x"}}, memberAToken)
		requireStatus(t, resp, b, http.StatusConflict)

		resp, b = deleteJSON(t, fo+"/webhooks/"+webhookID, managerToken)
		requireStatus(t, resp, b, http.StatusNoContent)

		resp, b = deleteJSON(t, schemaURL(), managerToken)
		requireStatus(t, resp, b, http.StatusNoContent)
		resp, b = getJSON(t, schemaURL(), memberAToken)
		requireStatus(t, resp, b, http.StatusNotFound)
		resp, b = postJSON(t, schemaURL()+"/submissions", map[string]interface{}{"answers": map[string]interface{}{"f1": "x"}}, memberAToken)
		requireStatus(t, resp, b, http.StatusNotFound)
		rows, _ := items(t, fo+"/schemas?search="+fmt.Sprint(suffix), memberAToken)
		if len(rows) != 1 {
			t.Fatalf("only the private form should be left, got %d", len(rows))
		}
	})
}
