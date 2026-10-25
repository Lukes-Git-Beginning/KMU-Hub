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

// TestWikiMemberFlow runs the wiki against the real seed with the bodies the desktop sends (wiki-client.ts,
// wiki-types.ts): a member writes and edits articles, two members and a manager read each other's work, and
// everything an editor or admin does is turned away. Needs the wiki service in the stack.
//
// IST of the seed that the test pins and does not judge:
//   - member holds `wiki:article:create|read` at scope all, `wiki:article:edit` at scope own, the legacy
//     `wiki:articles:read` and `wiki:categories:read` (000234); delete, share links, attachment delete and every
//     category write are 403; manager holds the legacy `wiki:articles:write` and `wiki:categories:write`
//     (000234), which open delete, share links and category management on top of the catalogue keys;
//   - the own scope of `wiki:article:edit` is NOT applied: a second member edits, restores and attaches to a
//     foreign article (fix-wiki-own-scope-article-edit);
//   - article reads are tenant-wide, drafts included: a second member lists and opens another member's draft;
//   - Create/Get/Update/Restore answer `{article:{…}}`, the desktop client reads the article flat;
//   - `tags`, `icon`, `cover_url`, `change_note` of the desktop body are not part of the gateway request and
//     are dropped without an error.
//
// There is no second tenant in the stack, so a foreign-tenant read is not part of this flow.
func TestWikiMemberFlow(t *testing.T) {
	base := gatewayURL()
	waitForHealth(t, base)

	managerToken, _ := registerAndLoginWithRole(t, base, "manager")
	memberAToken, memberAID := registerAndLoginWithRole(t, base, "member")
	memberBToken, _ := registerAndLoginWithRole(t, base, "member")

	wiki := base + "/api/v1/wiki"
	suffix := time.Now().UnixNano()

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
	// nested reads `{article:{…}}`-style wrappers: the proto response of the single-entity routes.
	nested := func(t *testing.T, body []byte, key string) map[string]interface{} {
		t.Helper()
		inner, ok := obj(t, body)[key].(map[string]interface{})
		if !ok {
			t.Fatalf("expected %q object in %s", key, string(body))
		}
		return inner
	}
	// content is TipTap JSON; the desktop encodes it as base64 of the UTF-8 JSON (wiki-content.ts).
	content := func(text string) string {
		doc := fmt.Sprintf(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":%q}]}]}`, text)
		return base64.StdEncoding.EncodeToString([]byte(doc))
	}
	// articles decodes a list or search answer; an empty page may omit the array, a `null` is the break.
	articles := func(t *testing.T, url, token string) []map[string]interface{} {
		t.Helper()
		resp, body := getJSON(t, url, token)
		requireStatus(t, resp, body, http.StatusOK)
		var raw map[string]json.RawMessage
		decodeBody(t, body, &raw)
		if strings.TrimSpace(string(raw["articles"])) == "null" {
			t.Fatalf("articles must never be null, got %s", string(body))
		}
		var out []map[string]interface{}
		if rows, ok := raw["articles"]; ok {
			decodeBody(t, rows, &out)
		}
		return out
	}
	hasID := func(rows []map[string]interface{}, id string) bool {
		for _, r := range rows {
			if str(r, "id") == id {
				return true
			}
		}
		return false
	}
	// bodyOf is what the desktop's createArticle sends (wiki-types.ts CreateArticleInput, content encoded).
	bodyOf := func(title, slug, text string, published bool) map[string]interface{} {
		return map[string]interface{}{
			"title":     title,
			"slug":      slug,
			"content":   content(text),
			"published": published,
			"tags":      []string{"e2e"},
			"icon":      "BookOpen",
		}
	}

	var (
		categoryID string
		articleID  string
		draftID    string
		tokenID    string
		token      string
		firstVer   string
		attachID   string
		slugA      = fmt.Sprintf("e2e-wiki-%d", suffix)
		titleA     = fmt.Sprintf("Onboarding Guide %d", suffix)
	)

	t.Run("manager creates a category, member cannot", func(t *testing.T) {
		resp, body := postJSON(t, wiki+"/categories", map[string]interface{}{"name": "Guides " + fmt.Sprint(suffix)}, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)

		resp, body = postJSON(t, wiki+"/categories", map[string]interface{}{"name": "Guides " + fmt.Sprint(suffix), "position": 1}, managerToken)
		requireStatus(t, resp, body, http.StatusCreated)
		categoryID = str(nested(t, body, "category"), "id")
		if categoryID == "" {
			t.Fatalf("category created without id: %s", string(body))
		}

		resp, body = getJSON(t, wiki+"/categories", memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		var list struct {
			Categories []map[string]interface{} `json:"categories"`
		}
		decodeBody(t, body, &list)
		if !hasID(list.Categories, categoryID) {
			t.Fatalf("member does not see the category %s in %s", categoryID, string(body))
		}

		resp, body = doRequest(t, http.MethodPatch, wiki+"/categories/"+categoryID, map[string]interface{}{"name": "x"}, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)
		resp, body = deleteJSON(t, wiki+"/categories/"+categoryID, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)
	})

	t.Run("member creates an article with the desktop body", func(t *testing.T) {
		// Validation first: title is required, a bad category id is refused, content must be JSON.
		resp, body := postJSON(t, wiki+"/articles", map[string]interface{}{"title": "", "content": content("x")}, memberAToken)
		requireStatus(t, resp, body, http.StatusBadRequest)
		resp, body = postJSON(t, wiki+"/articles", map[string]interface{}{"title": "No category", "category_id": "nope"}, memberAToken)
		requireStatus(t, resp, body, http.StatusBadRequest)
		resp, body = postJSON(t, wiki+"/articles", map[string]interface{}{
			"title":   "Bad content",
			"content": base64.StdEncoding.EncodeToString([]byte("{not json")),
		}, memberAToken)
		requireStatus(t, resp, body, http.StatusBadRequest)

		in := bodyOf(titleA, slugA, "Welcome aboard", true)
		in["category_id"] = categoryID
		resp, body = postJSON(t, wiki+"/articles", in, memberAToken)
		requireStatus(t, resp, body, http.StatusCreated)
		art := nested(t, body, "article")
		articleID = str(art, "id")
		if articleID == "" || str(art, "author_id") != memberAID || str(art, "slug") != slugA || str(art, "category_id") != categoryID {
			t.Fatalf("unexpected article %s", string(body))
		}
		if art["published"] != true {
			t.Fatalf("published article came back unpublished: %s", string(body))
		}

		// The same slug is a conflict, an unknown category is refused.
		resp, body = postJSON(t, wiki+"/articles", bodyOf("Twin", slugA, "x", true), memberAToken)
		requireStatus(t, resp, body, http.StatusConflict)
		missing := bodyOf("Lost category", fmt.Sprintf("e2e-wiki-lost-%d", suffix), "x", true)
		missing["category_id"] = "00000000-0000-4000-8000-000000000000"
		resp, body = postJSON(t, wiki+"/articles", missing, memberAToken)
		if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("unknown category: expected 404 or 400, got %d: %s", resp.StatusCode, string(body))
		}

		// A draft, kept for the visibility checks below.
		resp, body = postJSON(t, wiki+"/articles", bodyOf("Draft "+fmt.Sprint(suffix), fmt.Sprintf("e2e-wiki-draft-%d", suffix), "unfinished", false), memberAToken)
		requireStatus(t, resp, body, http.StatusCreated)
		draftID = str(nested(t, body, "article"), "id")
	})

	t.Run("a second member reads the article, list and search", func(t *testing.T) {
		resp, body := getJSON(t, wiki+"/articles/"+articleID, memberBToken)
		requireStatus(t, resp, body, http.StatusOK)
		art := nested(t, body, "article")
		if str(art, "title") != titleA {
			t.Fatalf("unexpected title in %s", string(body))
		}
		// content rides as base64 of the TipTap JSON; the desktop decodes it back (wiki-content.ts).
		raw, err := base64.StdEncoding.DecodeString(str(art, "content"))
		if err != nil || !strings.Contains(string(raw), "Welcome aboard") {
			t.Fatalf("content did not round-trip: %q (%v)", str(art, "content"), err)
		}

		rows := articles(t, wiki+"/articles?page=1&page_size=50&sort_by=updated_at&sort_desc=true", memberBToken)
		if !hasID(rows, articleID) {
			t.Fatalf("member B does not see the published article in the list")
		}
		rows = articles(t, wiki+"/articles?search="+fmt.Sprint(suffix)+"&category_id="+categoryID, memberBToken)
		if !hasID(rows, articleID) || len(rows) != 1 {
			t.Fatalf("category + search filter returned %d rows, want exactly the article", len(rows))
		}
		rows = articles(t, wiki+"/articles?author_id="+memberAID+"&published=true", memberBToken)
		if !hasID(rows, articleID) || hasID(rows, draftID) {
			t.Fatalf("author + published filter must hold the article and not the draft")
		}

		rows = articles(t, wiki+"/search?q=Onboarding&limit=20", memberBToken)
		if !hasID(rows, articleID) {
			t.Fatalf("search does not find the published article")
		}
		resp, body = getJSON(t, wiki+"/search", memberBToken)
		requireStatus(t, resp, body, http.StatusBadRequest)

		resp, body = getJSON(t, wiki+"/articles/not-a-uuid", memberBToken)
		requireStatus(t, resp, body, http.StatusBadRequest)
		resp, body = getJSON(t, wiki+"/articles/00000000-0000-4000-8000-000000000000", memberBToken)
		requireStatus(t, resp, body, http.StatusNotFound)

		// IST: drafts are tenant-wide readable, member B opens and lists member A's draft.
		resp, body = getJSON(t, wiki+"/articles/"+draftID, memberBToken)
		requireStatus(t, resp, body, http.StatusOK)
		rows = articles(t, wiki+"/articles?published=false&search="+fmt.Sprint(suffix), memberBToken)
		if !hasID(rows, draftID) {
			t.Fatalf("IST changed: member B no longer lists a foreign draft")
		}
	})

	t.Run("author edits the article and walks the versions", func(t *testing.T) {
		resp, body := doRequest(t, http.MethodPatch, wiki+"/articles/"+articleID, map[string]interface{}{
			"title":       titleA + " v2",
			"content":     content("Second draft"),
			"change_note": "reworded",
			"tags":        []string{"e2e", "hr"},
		}, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		if got := str(nested(t, body, "article"), "title"); got != titleA+" v2" {
			t.Fatalf("title not updated: %s", string(body))
		}

		// An empty title and a foreign slug are refused, an unknown id is 404.
		resp, body = doRequest(t, http.MethodPatch, wiki+"/articles/"+articleID, map[string]interface{}{"title": ""}, memberAToken)
		requireStatus(t, resp, body, http.StatusBadRequest)
		resp, body = doRequest(t, http.MethodPatch, wiki+"/articles/"+articleID, map[string]interface{}{"slug": fmt.Sprintf("e2e-wiki-draft-%d", suffix)}, memberAToken)
		requireStatus(t, resp, body, http.StatusConflict)
		resp, body = doRequest(t, http.MethodPatch, wiki+"/articles/00000000-0000-4000-8000-000000000000", map[string]interface{}{"title": "x"}, memberAToken)
		requireStatus(t, resp, body, http.StatusNotFound)

		// Publishing state and category clear (empty string = clear, as the move dialog sends).
		resp, body = doRequest(t, http.MethodPatch, wiki+"/articles/"+draftID, map[string]interface{}{"published": true, "category_id": ""}, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = getJSON(t, wiki+"/articles/"+articleID+"/versions", memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		var vers struct {
			Versions []map[string]interface{} `json:"versions"`
		}
		decodeBody(t, body, &vers)
		if len(vers.Versions) != 1 {
			t.Fatalf("one content edit must leave exactly one version, got %d: %s", len(vers.Versions), string(body))
		}
		firstVer = str(vers.Versions[0], "id")

		resp, body = getJSON(t, wiki+"/versions/"+firstVer, memberBToken)
		requireStatus(t, resp, body, http.StatusOK)
		ver := nested(t, body, "version")
		raw, _ := base64.StdEncoding.DecodeString(str(ver, "content"))
		if !strings.Contains(string(raw), "Welcome aboard") {
			t.Fatalf("version 1 must hold the content before the edit, got %q", string(raw))
		}

		resp, body = postJSON(t, wiki+"/articles/"+articleID+"/versions/"+firstVer+"/restore", nil, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		raw, _ = base64.StdEncoding.DecodeString(str(nested(t, body, "article"), "content"))
		if !strings.Contains(string(raw), "Welcome aboard") {
			t.Fatalf("restore did not bring the first content back: %q", string(raw))
		}
		resp, body = postJSON(t, wiki+"/articles/"+articleID+"/versions/00000000-0000-4000-8000-000000000000/restore", nil, memberAToken)
		requireStatus(t, resp, body, http.StatusNotFound)
	})

	t.Run("attachments: member adds and lists, deleting is turned away", func(t *testing.T) {
		resp, body := postJSON(t, wiki+"/articles/"+articleID+"/attachments", map[string]interface{}{"file_ref": ""}, memberAToken)
		requireStatus(t, resp, body, http.StatusBadRequest)

		// The desktop sends a demo `data_url` too; the gateway ignores it.
		resp, body = postJSON(t, wiki+"/articles/"+articleID+"/attachments", map[string]interface{}{
			"file_ref": fmt.Sprintf("wiki/%d/handbook.pdf", suffix),
			"mime":     "application/pdf",
			"size":     2048,
			"data_url": "data:application/pdf;base64,AAAA",
		}, memberAToken)
		requireStatus(t, resp, body, http.StatusCreated)
		attachID = str(nested(t, body, "attachment"), "id")
		if attachID == "" {
			t.Fatalf("attachment without id: %s", string(body))
		}

		resp, body = getJSON(t, wiki+"/articles/"+articleID+"/attachments", memberBToken)
		requireStatus(t, resp, body, http.StatusOK)
		var list struct {
			Attachments []map[string]interface{} `json:"attachments"`
		}
		decodeBody(t, body, &list)
		if !hasID(list.Attachments, attachID) {
			t.Fatalf("attachment %s missing in %s", attachID, string(body))
		}

		resp, body = deleteJSON(t, wiki+"/attachments/"+attachID, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)
		resp, body = deleteJSON(t, wiki+"/articles/"+articleID+"/attachments/"+attachID, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)
	})

	t.Run("delete and share links are turned away for member", func(t *testing.T) {
		resp, body := deleteJSON(t, wiki+"/articles/"+articleID, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)
		resp, body = postJSON(t, wiki+"/articles/"+articleID+"/share", map[string]interface{}{"permissions": []string{"read"}}, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)
		resp, body = deleteJSON(t, wiki+"/share/00000000-0000-4000-8000-000000000000", memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)

		// Share-token listing is a read: member sees an empty set, not an error.
		resp, body = getJSON(t, wiki+"/articles/"+articleID+"/share", memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
	})

	t.Run("manager shares the article, the token redeems without login and revokes", func(t *testing.T) {
		resp, body := postJSON(t, wiki+"/articles/"+articleID+"/share", map[string]interface{}{
			"permissions": []string{"read"},
			"expires_at":  time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
		}, managerToken)
		requireStatus(t, resp, body, http.StatusCreated)
		tok := nested(t, body, "token")
		tokenID, token = str(tok, "id"), str(tok, "token")
		if tokenID == "" || token == "" {
			t.Fatalf("share token without id/token: %s", string(body))
		}

		resp, body = getJSON(t, wiki+"/articles/"+articleID+"/share", memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		var list struct {
			Tokens []map[string]interface{} `json:"tokens"`
		}
		decodeBody(t, body, &list)
		if !hasID(list.Tokens, tokenID) {
			t.Fatalf("token %s not listed for the member: %s", tokenID, string(body))
		}

		// The public route needs no login and answers only title, content and update time.
		resp, body = postJSON(t, base+"/api/v1/public/wiki/articles/"+token, nil, "")
		requireStatus(t, resp, body, http.StatusOK)
		shared := nested(t, body, "article")
		if str(shared, "title") != titleA+" v2" {
			t.Fatalf("unexpected shared article %s", string(body))
		}
		for _, leak := range []string{"tenant_id", "author_id", "id", "category_id"} {
			if _, ok := shared[leak]; ok {
				t.Fatalf("public wire type leaks %q: %s", leak, string(body))
			}
		}
		resp, body = postJSON(t, base+"/api/v1/public/wiki/articles/not-a-token", nil, "")
		requireStatus(t, resp, body, http.StatusNotFound)

		resp, body = deleteJSON(t, wiki+"/share/"+tokenID, memberAToken)
		requireStatus(t, resp, body, http.StatusForbidden)
		resp, body = deleteJSON(t, wiki+"/share/"+tokenID, managerToken)
		requireStatus(t, resp, body, http.StatusNoContent)
		resp, body = postJSON(t, base+"/api/v1/public/wiki/articles/"+token, nil, "")
		requireStatus(t, resp, body, http.StatusNotFound)
	})

	t.Run("a second member edits a foreign article (IST, fix-wiki-own-scope-article-edit)", func(t *testing.T) {
		// `wiki:article:edit` is granted to member at scope own, yet the PATCH, restore and attachment routes
		// check only the key. Flip these three to 403/404 together with the fix.
		resp, body := doRequest(t, http.MethodPatch, wiki+"/articles/"+articleID, map[string]interface{}{"title": titleA + " hijacked"}, memberBToken)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = postJSON(t, wiki+"/articles/"+articleID+"/versions/"+firstVer+"/restore", nil, memberBToken)
		requireStatus(t, resp, body, http.StatusOK)

		resp, body = postJSON(t, wiki+"/articles/"+articleID+"/attachments", map[string]interface{}{
			"file_ref": fmt.Sprintf("wiki/%d/foreign.png", suffix), "mime": "image/png", "size": 10,
		}, memberBToken)
		requireStatus(t, resp, body, http.StatusCreated)

		resp, body = getJSON(t, wiki+"/articles/"+articleID, memberAToken)
		requireStatus(t, resp, body, http.StatusOK)
		if got := str(nested(t, body, "article"), "title"); got != titleA+" hijacked" {
			t.Fatalf("expected the foreign edit to be visible to the author, got %q", got)
		}
	})

	t.Run("manager removes the drafts and attachments, the article is gone for everyone", func(t *testing.T) {
		resp, body := deleteJSON(t, wiki+"/attachments/"+attachID, managerToken)
		requireStatus(t, resp, body, http.StatusNoContent)

		resp, body = deleteJSON(t, wiki+"/articles/"+draftID, managerToken)
		requireStatus(t, resp, body, http.StatusNoContent)
		resp, body = getJSON(t, wiki+"/articles/"+draftID, memberAToken)
		requireStatus(t, resp, body, http.StatusNotFound)
		resp, body = deleteJSON(t, wiki+"/articles/"+draftID, managerToken)
		requireStatus(t, resp, body, http.StatusNotFound)

		resp, body = deleteJSON(t, wiki+"/categories/"+categoryID, managerToken)
		requireStatus(t, resp, body, http.StatusNoContent)
	})
}
