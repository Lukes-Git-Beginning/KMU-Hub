// Package security provides gateway-layer security primitives that do not
// belong to a specific sub-domain (audit, vault, …).
package security

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// CaptchaVerifier validates client tokens against a siteverify-compatible API
// (Cloudflare Turnstile, hCaptcha, Google reCAPTCHA v2/v3 all share this shape).
//
// When Secret is empty the verifier is disabled and Verify always returns true,
// allowing the hook to be toggled via environment variable without code changes.
type CaptchaVerifier struct {
	secret    string
	verifyURL string
	client    *http.Client
}

// NewCaptchaVerifier constructs a CaptchaVerifier. An empty secret disables
// verification (all requests pass through). Use a 5-second HTTP timeout so a
// slow upstream cannot stall the booking request indefinitely.
func NewCaptchaVerifier(secret, verifyURL string) *CaptchaVerifier {
	return &CaptchaVerifier{
		secret:    secret,
		verifyURL: verifyURL,
		client:    &http.Client{Timeout: 5 * time.Second},
	}
}

// Enabled reports whether captcha verification is active.
// Returns false when Secret is empty — the hook is disabled.
func (c *CaptchaVerifier) Enabled() bool {
	return c.secret != ""
}

// siteverifyResponse is the common JSON shape returned by Cloudflare Turnstile,
// hCaptcha, and Google reCAPTCHA siteverify endpoints.
type siteverifyResponse struct {
	Success bool `json:"success"`
}

// Verify submits token to the upstream siteverify endpoint and returns whether
// the challenge was solved. remoteIP is forwarded as a hint; it is optional
// and some providers ignore it.
//
// Behavior:
//   - Disabled (Enabled() == false): returns true, nil — pass-through.
//   - Empty token with verification enabled: returns false, nil — fail fast,
//     no network round-trip needed.
//   - Network or parse error: returns false + error (fail-closed) and emits
//     slog.Warn so on-call can distinguish provider outages from abuse.
func (c *CaptchaVerifier) Verify(ctx context.Context, token, remoteIP string) (bool, error) {
	if !c.Enabled() {
		return true, nil
	}
	if token == "" {
		return false, nil
	}

	form := url.Values{}
	form.Set("secret", c.secret)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.verifyURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		slog.Warn("captcha: failed to build siteverify request", "error", err)
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.client.Do(req)
	if err != nil {
		slog.Warn("captcha: siteverify request failed", "error", err, "url", c.verifyURL)
		return false, err
	}
	defer resp.Body.Close()

	var result siteverifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		slog.Warn("captcha: failed to decode siteverify response", "error", err)
		return false, err
	}

	return result.Success, nil
}
