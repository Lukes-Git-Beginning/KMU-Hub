package action

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/kmuhub/kmuhub/internal/security/safehttp"
)

// ============================================================================
// HTTPRequestAction
// ============================================================================

const (
	// maxRequestBodyBytes caps the body an automation may send. Templates
	// interpolate record data, so the size is not bounded by the config alone.
	maxRequestBodyBytes = 256 * 1024

	// maxOutputBodyChars caps how much of the response is chained into the
	// following actions. The environment is written to automation_executions on
	// every run, so a large response would land in the database once per
	// execution rather than once.
	maxOutputBodyChars = 4096
)

// allowedMethods excludes CONNECT (tunnels) and TRACE (reflects request
// headers, which is a credential-disclosure primitive when combined with a
// tenant-configured Authorization header).
var allowedMethods = map[string]bool{
	http.MethodGet:    true,
	http.MethodPost:   true,
	http.MethodPut:    true,
	http.MethodPatch:  true,
	http.MethodDelete: true,
	http.MethodHead:   true,
}

// forbiddenHeaders would let a config rewrite the framing of the request rather
// than add to it.
var forbiddenHeaders = map[string]bool{
	"Host":              true,
	"Content-Length":    true,
	"Transfer-Encoding": true,
	"Connection":        true,
	"Upgrade":           true,
}

// HTTPRequestAction calls an HTTP endpoint a tenant configured.
//
// The target is tenant data, so it is treated as hostile: the client comes from
// safehttp and can only reach public addresses (see that package for why a URL
// check alone does not hold). Nothing about the request or its headers is
// logged beyond host and path — a configured Authorization or X-Api-Key header
// is exactly the kind of value that must not end up in a log aggregator.
type HTTPRequestAction struct {
	client *http.Client
}

// NewHTTPRequestAction creates an HTTPRequestAction. A nil client is replaced
// by a default safehttp client, so a caller cannot accidentally construct an
// unguarded one.
func NewHTTPRequestAction(client *http.Client) *HTTPRequestAction {
	if client == nil {
		client = safehttp.New()
	}
	return &HTTPRequestAction{client: client}
}

func (a *HTTPRequestAction) Type() string { return "http.request" }

type httpRequestConfig struct {
	Method  string            `json:"method"`  // GET, POST, PUT, PATCH, DELETE, HEAD
	URL     string            `json:"url"`     // template
	Headers map[string]string `json:"headers"` // values are templates
	Body    string            `json:"body"`    // template
}

func (a *HTTPRequestAction) Execute(ctx context.Context, config json.RawMessage, env map[string]any) (ActionResult, error) {
	var cfg httpRequestConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return ActionResult{Success: false, Error: "invalid config: " + err.Error()}, err
	}

	req, err := a.buildRequest(ctx, cfg, env)
	if err != nil {
		return ActionResult{Success: false, Error: err.Error()}, err
	}

	resp, err := a.client.Do(req)
	if err != nil {
		// The URL is in the error, so it is kept out of the message the tenant
		// sees only insofar as it is theirs anyway; headers never appear.
		msg := requestErrorMessage(err)
		slog.Warn("automation http request failed",
			"url", safeURLForLog(req.URL),
			"method", req.Method,
			"error", msg,
		)
		return ActionResult{Success: false, Error: msg}, err
	}
	defer resp.Body.Close() //nolint:errcheck

	body, readErr := io.ReadAll(resp.Body)
	truncated := false
	if readErr != nil {
		if !errors.Is(readErr, safehttp.ErrResponseTooLarge) {
			return ActionResult{Success: false, Error: "read response: " + readErr.Error()}, readErr
		}
		// The status line is still a usable result; say the body is short
		// rather than pretend it is complete.
		truncated = true
	}

	text := string(body)
	if len(text) > maxOutputBodyChars {
		text = text[:maxOutputBodyChars]
		truncated = true
	}

	slog.Info("automation http request",
		"url", safeURLForLog(req.URL),
		"method", req.Method,
		"status_code", resp.StatusCode,
		"response_bytes", len(body),
	)

	output := map[string]any{
		"status_code":    resp.StatusCode,
		"body":           text,
		"body_truncated": truncated,
	}

	// A 4xx/5xx is a failed action, not a failed call: the automation's
	// on_error branch is what decides whether the run continues, and it can
	// only do that if the status is reported honestly.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ActionResult{
			Success: false,
			Output:  output,
			Error:   fmt.Sprintf("endpoint returned status %d", resp.StatusCode),
		}, nil
	}

	return ActionResult{Success: true, Output: output}, nil
}

func (a *HTTPRequestAction) buildRequest(ctx context.Context, cfg httpRequestConfig, env map[string]any) (*http.Request, error) {
	method := strings.ToUpper(strings.TrimSpace(cfg.Method))
	if method == "" {
		method = http.MethodGet
	}
	if !allowedMethods[method] {
		return nil, fmt.Errorf("method %q is not allowed", method)
	}

	rawURL := strings.TrimSpace(resolveTemplate(cfg.URL, env))
	if rawURL == "" {
		return nil, errors.New("url is required")
	}
	target, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	if err := safehttp.CheckURL(target); err != nil {
		return nil, err
	}

	body := resolveTemplate(cfg.Body, env)
	if len(body) > maxRequestBodyBytes {
		return nil, fmt.Errorf("request body of %d bytes exceeds the limit of %d", len(body), maxRequestBodyBytes)
	}

	var reader io.Reader
	if body != "" && method != http.MethodGet && method != http.MethodHead {
		reader = bytes.NewReader([]byte(body))
	}

	req, err := http.NewRequestWithContext(ctx, method, target.String(), reader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "Cosmi-Automation/1.0")
	if reader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for name, rawValue := range cfg.Headers {
		name = strings.TrimSpace(name)
		value := resolveTemplate(rawValue, env)
		if err := validateHeader(name, value); err != nil {
			return nil, err
		}
		req.Header.Set(name, value)
	}

	return req, nil
}

// validateHeader rejects the two ways a templated header can do more than it
// says: a control character splits the request (header injection, and the
// template carries record data), and a framing header rewrites it.
func validateHeader(name, value string) error {
	if name == "" {
		return errors.New("header name must not be empty")
	}
	if forbiddenHeaders[http.CanonicalHeaderKey(name)] {
		return fmt.Errorf("header %q cannot be set", name)
	}
	if strings.ContainsAny(name, "\r\n\x00 ") {
		return fmt.Errorf("header name %q contains an invalid character", name)
	}
	if strings.ContainsAny(value, "\r\n\x00") {
		// The value is not echoed: it may be a secret, and this error text is
		// stored with the execution log.
		return fmt.Errorf("value of header %q contains a line break", name)
	}
	return nil
}

// safeURLForLog keeps scheme, host and path and drops the query, which is where
// endpoints commonly carry tokens.
func safeURLForLog(u *url.URL) string {
	if u == nil {
		return ""
	}
	redacted := &url.URL{Scheme: u.Scheme, Host: u.Host, Path: u.Path}
	return redacted.String()
}

// requestErrorMessage keeps the blocked-address case recognisable for the
// person configuring the automation, who would otherwise see a bare dial error
// and assume the endpoint is down.
func requestErrorMessage(err error) string {
	switch {
	case errors.Is(err, safehttp.ErrBlockedAddress):
		return "target resolves to a non-public address and was blocked"
	case errors.Is(err, safehttp.ErrUnsupportedScheme):
		return "redirect target uses an unsupported scheme"
	case errors.Is(err, context.DeadlineExceeded):
		return "request timed out"
	default:
		return err.Error()
	}
}

// HTTPRequestDefinition returns the action definition for the catalog.
func HTTPRequestDefinition() *ActionDefinition {
	return &ActionDefinition{
		Type:        "http.request",
		Module:      "automation",
		Name:        "HTTP-Anfrage senden",
		Description: "Ruft eine externe HTTP-Schnittstelle auf. Nur oeffentlich erreichbare Adressen.",
		Params: []ConfigParam{
			{Key: "method", Label: "Methode", Type: "select", Required: true, Default: "POST"},
			{Key: "url", Label: "URL", Type: "template", Required: true},
			{Key: "headers", Label: "Header", Type: "string", Required: false},
			{Key: "body", Label: "Inhalt", Type: "template", Required: false},
		},
		OutputFields: []OutputField{
			{Key: "status_code", Label: "Statuscode", Type: "number"},
			{Key: "body", Label: "Antwort", Type: "string"},
			{Key: "body_truncated", Label: "Antwort gekuerzt", Type: "boolean"},
		},
	}
}
