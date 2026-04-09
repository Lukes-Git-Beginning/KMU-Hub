package gateway

import (
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kmuhub/kmuhub/internal/server/response"
	securityv1 "github.com/kmuhub/kmuhub/proto/security/v1"
)

const (
	ipRuleCacheTTL     = 60 * time.Second
	ipRuleMaxStaleness = 5 * time.Minute
)

var errIPRulesUnavailable = errors.New("IP rules unavailable: auth service unreachable")

// IPFilterMiddleware checks client IP addresses against configured allow/block rules.
// Rules are cached and refreshed every 60 seconds from the security gRPC service.
type IPFilterMiddleware struct {
	registry       *ServiceRegistry
	mu             sync.RWMutex
	rules          []*securityv1.IPAccessRule
	lastLoad       time.Time
	rulesEverLoaded bool
}

// NewIPFilterMiddleware creates a new IP filter middleware.
func NewIPFilterMiddleware(registry *ServiceRegistry) *IPFilterMiddleware {
	return &IPFilterMiddleware{
		registry: registry,
	}
}

// Middleware returns the HTTP middleware handler.
func (f *IPFilterMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rules, err := f.getRules(r)
		if err != nil {
			slog.Error("IP filter: blocking request, rules unavailable", "error", err)
			response.Error(w, http.StatusServiceUnavailable, "service temporarily unavailable")
			return
		}
		if len(rules) == 0 {
			// No rules configured: allow all
			next.ServeHTTP(w, r)
			return
		}

		clientIP := extractClientIP(r)
		if clientIP == "" {
			// Cannot determine IP: allow (fail open)
			next.ServeHTTP(w, r)
			return
		}

		ip := net.ParseIP(clientIP)
		if ip == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Check blocklist first
		for _, rule := range rules {
			if rule.RuleType != "block" {
				continue
			}
			_, cidr, err := net.ParseCIDR(rule.IpCidr)
			if err != nil {
				continue
			}
			if cidr.Contains(ip) {
				slog.Warn("IP blocked by rule",
					"ip", clientIP,
					"rule_cidr", rule.IpCidr,
					"rule_id", rule.Id,
				)
				response.Error(w, http.StatusForbidden, "access denied")
				return
			}
		}

		// Check if any allowlist rules exist
		hasAllowRules := false
		for _, rule := range rules {
			if rule.RuleType == "allow" {
				hasAllowRules = true
				break
			}
		}

		// If allowlist exists, client must match at least one
		if hasAllowRules {
			allowed := false
			for _, rule := range rules {
				if rule.RuleType != "allow" {
					continue
				}
				_, cidr, err := net.ParseCIDR(rule.IpCidr)
				if err != nil {
					continue
				}
				if cidr.Contains(ip) {
					allowed = true
					break
				}
			}
			if !allowed {
				slog.Warn("IP not in allowlist",
					"ip", clientIP,
				)
				response.Error(w, http.StatusForbidden, "access denied")
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// handleFetchFailure decides whether to serve stale rules or fail closed.
func (f *IPFilterMiddleware) handleFetchFailure(err error) ([]*securityv1.IPAccessRule, error) {
	if !f.rulesEverLoaded {
		slog.Error("IP filter: no rules ever loaded, failing closed", "error", err)
		return nil, errIPRulesUnavailable
	}
	if time.Since(f.lastLoad) > ipRuleMaxStaleness {
		slog.Error("IP filter: cache exceeds max staleness, failing closed",
			"stale_for", time.Since(f.lastLoad).String(), "error", err)
		return nil, errIPRulesUnavailable
	}
	slog.Warn("IP filter: serving stale rules", "stale_for", time.Since(f.lastLoad).String(), "error", err)
	return f.rules, nil
}

// getRules returns cached rules, refreshing if stale.
func (f *IPFilterMiddleware) getRules(r *http.Request) ([]*securityv1.IPAccessRule, error) {
	f.mu.RLock()
	if time.Since(f.lastLoad) < ipRuleCacheTTL && f.rules != nil {
		rules := f.rules
		f.mu.RUnlock()
		return rules, nil
	}
	f.mu.RUnlock()

	f.mu.Lock()
	defer f.mu.Unlock()

	// Double-check after acquiring write lock
	if time.Since(f.lastLoad) < ipRuleCacheTTL && f.rules != nil {
		return f.rules, nil
	}

	conn, err := f.registry.GetConnection("auth")
	if err != nil {
		return f.handleFetchFailure(err)
	}

	client := securityv1.NewSecurityServiceClient(conn)
	resp, err := client.ListIPRules(r.Context(), &securityv1.ListIPRulesRequest{})
	if err != nil {
		return f.handleFetchFailure(err)
	}

	f.rules = resp.Rules
	f.lastLoad = time.Now()
	f.rulesEverLoaded = true
	slog.Debug("IP filter rules refreshed", "count", len(f.rules))
	return f.rules, nil
}

// extractClientIP extracts the client IP from X-Forwarded-For or RemoteAddr.
func extractClientIP(r *http.Request) string {
	// Check X-Forwarded-For first (may contain comma-separated IPs)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return ip
		}
	}

	// Check X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
