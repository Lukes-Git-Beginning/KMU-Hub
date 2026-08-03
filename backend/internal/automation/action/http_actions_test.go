package action

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/kmuhub/kmuhub/internal/security/safehttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testAction talks to an httptest server, which lives on loopback. Only the
// address filter is relaxed for that; redirect handling, the body cap and the
// timeout are the production ones.
func testAction() *HTTPRequestAction {
	return NewHTTPRequestAction(safehttp.New(safehttp.AllowLoopback()))
}

func run(t *testing.T, a *HTTPRequestAction, cfg map[string]any, env map[string]any) (ActionResult, error) {
	t.Helper()
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)
	return a.Execute(context.Background(), raw, env)
}

func TestHTTPRequestAction_Type(t *testing.T) {
	assert.Equal(t, "http.request", testAction().Type())
}

func TestHTTPRequestAction_PostsTemplatedBodyAndHeaders(t *testing.T) {
	var (
		gotMethod string
		gotBody   string
		gotHeader string
		gotPath   string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotHeader = r.Header.Get("X-Api-Key")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"received":true}`))
	}))
	defer srv.Close()

	res, err := run(t, testAction(), map[string]any{
		"method":  "post",
		"url":     srv.URL + "/hook",
		"headers": map[string]string{"X-Api-Key": "k-{{deal.id}}"},
		"body":    `{"deal":"{{deal.name}}"}`,
	}, map[string]any{
		"deal": map[string]any{"id": "42", "name": "Acme"},
	})

	require.NoError(t, err)
	require.True(t, res.Success, "error: %s", res.Error)
	assert.Equal(t, http.MethodPost, gotMethod, "the method is upper-cased, not passed through")
	assert.Equal(t, "/hook", gotPath)
	assert.Equal(t, "k-42", gotHeader, "header values are templated")
	assert.JSONEq(t, `{"deal":"Acme"}`, gotBody)
	assert.Equal(t, http.StatusOK, res.Output["status_code"])
	assert.Equal(t, `{"received":true}`, res.Output["body"])
	assert.Equal(t, false, res.Output["body_truncated"])
}

func TestHTTPRequestAction_GetSendsNoBody(t *testing.T) {
	var contentLength int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentLength = r.ContentLength
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res, err := run(t, testAction(), map[string]any{
		"url":  srv.URL,
		"body": "ignored",
	}, nil)

	require.NoError(t, err)
	assert.True(t, res.Success)
	assert.Zero(t, contentLength, "a GET must not carry the configured body")
}

// A 4xx is the endpoint answering, not the call failing. The action reports it
// as unsuccessful but returns no Go error, so the automation's on_error branch
// decides what happens and the status stays visible in the execution log.
func TestHTTPRequestAction_NonSuccessStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("nope"))
	}))
	defer srv.Close()

	res, err := run(t, testAction(), map[string]any{"url": srv.URL}, nil)

	require.NoError(t, err)
	assert.False(t, res.Success)
	assert.Contains(t, res.Error, "403")
	assert.Equal(t, http.StatusForbidden, res.Output["status_code"])
	assert.Equal(t, "nope", res.Output["body"], "the body of an error response is still chainable")
}

func TestHTTPRequestAction_TruncatesLongResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", maxOutputBodyChars+500)))
	}))
	defer srv.Close()

	res, err := run(t, testAction(), map[string]any{"url": srv.URL}, nil)

	require.NoError(t, err)
	require.True(t, res.Success, "error: %s", res.Error)
	assert.Len(t, res.Output["body"], maxOutputBodyChars)
	assert.Equal(t, true, res.Output["body_truncated"], "truncation must be visible, not silent")
}

// The core of the unit: a configured URL cannot reach the network the server
// sits in. These go through the production client — no AllowLoopback.
func TestHTTPRequestAction_BlocksPrivateTargets(t *testing.T) {
	action := NewHTTPRequestAction(nil)

	for _, target := range []string{
		"http://169.254.169.254/latest/meta-data/",
		"http://10.0.0.1/internal",
		"http://127.0.0.1:5432/",
		"http://[::1]:6379/",
		"http://192.168.1.1/",
	} {
		t.Run(target, func(t *testing.T) {
			res, err := run(t, action, map[string]any{"url": target}, nil)
			require.Error(t, err)
			assert.False(t, res.Success)
			assert.Contains(t, res.Error, "non-public address")
		})
	}
}

// The target can be assembled from record data, so the block has to survive
// template resolution rather than run before it.
func TestHTTPRequestAction_BlocksTemplatedPrivateTarget(t *testing.T) {
	res, err := run(t, NewHTTPRequestAction(nil), map[string]any{
		"url": "http://{{contact.host}}/x",
	}, map[string]any{
		"contact": map[string]any{"host": "169.254.169.254"},
	})

	require.Error(t, err)
	assert.False(t, res.Success)
	assert.Contains(t, res.Error, "non-public address")
}

func TestHTTPRequestAction_BlocksRedirectIntoPrivateRange(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://169.254.169.254/latest/meta-data/", http.StatusFound)
	}))
	defer srv.Close()

	res, err := run(t, testAction(), map[string]any{"url": srv.URL}, nil)

	require.Error(t, err)
	assert.False(t, res.Success)
	assert.Contains(t, res.Error, "non-public address")
}

func TestHTTPRequestAction_TimeoutSurfacesAsFailure(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-release
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	defer close(release)

	action := NewHTTPRequestAction(safehttp.New(safehttp.AllowLoopback(), safehttp.WithTimeout(150*time.Millisecond)))
	res, err := run(t, action, map[string]any{"url": srv.URL}, nil)

	require.Error(t, err)
	assert.False(t, res.Success)
	assert.NotEmpty(t, res.Error)
}

func TestHTTPRequestAction_RejectsBadConfig(t *testing.T) {
	action := testAction()

	cases := []struct {
		name    string
		cfg     map[string]any
		wantErr string
	}{
		{"empty url", map[string]any{"url": "  "}, "url is required"},
		{"file scheme", map[string]any{"url": "file:///etc/passwd"}, "only http and https"},
		{"ftp scheme", map[string]any{"url": "ftp://example.com/x"}, "only http and https"},
		{"credentials in url", map[string]any{"url": "https://u:p@example.com/x"}, "credentials"},
		{"connect method", map[string]any{"url": "https://example.com", "method": "CONNECT"}, "not allowed"},
		{"trace method", map[string]any{"url": "https://example.com", "method": "TRACE"}, "not allowed"},
		{
			"header injection",
			map[string]any{"url": "https://example.com", "headers": map[string]string{"X-A": "a\r\nX-Evil: 1"}},
			"line break",
		},
		{
			"framing header",
			map[string]any{"url": "https://example.com", "headers": map[string]string{"Host": "evil.example"}},
			"cannot be set",
		},
		{
			"oversized body",
			map[string]any{"url": "https://example.com", "method": "POST", "body": strings.Repeat("x", maxRequestBodyBytes+1)},
			"exceeds the limit",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := run(t, action, tc.cfg, nil)
			require.Error(t, err)
			assert.False(t, res.Success)
			assert.Contains(t, res.Error, tc.wantErr)
		})
	}
}

// A header value assembled from record data must not be able to smuggle a
// second header in — the check runs after the template, not on the config.
func TestHTTPRequestAction_BlocksInjectionViaTemplate(t *testing.T) {
	res, err := run(t, testAction(), map[string]any{
		"url":     "https://example.com",
		"headers": map[string]string{"X-Note": "{{contact.note}}"},
	}, map[string]any{
		"contact": map[string]any{"note": "ok\r\nX-Admin: true"},
	})

	require.Error(t, err)
	assert.False(t, res.Success)
	assert.Contains(t, res.Error, "line break")
	assert.NotContains(t, res.Error, "X-Admin", "the rejected value must not be echoed back into the log")
}

func TestHTTPRequestAction_InvalidJSONConfig(t *testing.T) {
	res, err := testAction().Execute(context.Background(), json.RawMessage(`{"url":`), nil)
	require.Error(t, err)
	assert.False(t, res.Success)
	assert.Contains(t, res.Error, "invalid config")
}

func TestHTTPRequestDefinition(t *testing.T) {
	def := HTTPRequestDefinition()
	assert.Equal(t, "http.request", def.Type)
	assert.Equal(t, (&HTTPRequestAction{}).Type(), def.Type, "catalog entry and executor must agree")
	require.NotEmpty(t, def.Params)
	require.NotEmpty(t, def.OutputFields)
}

func TestSafeURLForLog(t *testing.T) {
	u, err := url.Parse("https://api.example.com/hook/abc?token=s3cret&x=1")
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com/hook/abc", safeURLForLog(u),
		"the query is where endpoints carry tokens and must not be logged")
	assert.Empty(t, safeURLForLog(nil))
}
