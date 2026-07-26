// Package gateway_test contains the OpenAPI route↔spec drift guard.
//
// Motivation: the "openapi-validate" CI job only lints api/openapi.yaml for
// syntactic validity — it never compares the spec against the routes the
// gateway actually registers. That gap let entire module groups (~185
// endpoints) go undocumented for years without CI ever noticing. This test
// builds a real chi.Router the same way cmd/gateway/main.go does (same
// RouteRegistrar list, same construction order) and asserts that every
// /api/v1/* route the gateway registers has a matching top-level path key
// in api/openapi.yaml.
//
// Deliberately out of scope (see allowlist / comments below for why):
//   - non-/api/v1 protocol routes (health, metrics, WOPI, CalDAV/CardDAV
//     WebDAV verbs, /.well-known discovery) — not REST, not OpenAPI material.
//   - routes wired outside the RouteRegistrar assembly that need a live
//     WebSocket hub (/api/v1/ws, /api/v1/files/upload) — constructing that
//     needs a real gRPC-reachable chat/auth service; out of reach for an
//     in-memory, DB-less unit test.
//   - the /api/v1/caldav/* and /api/v1/admin/caldav/* REST routes — reachable
//     only via cmd/gateway's unexported adapter types (package main), which
//     cannot be imported from a test in another package. These paths are
//     already documented today; if this guard should cover them too, extract
//     the adapters into an exported gateway-internal helper first.
//   - Plugin API routes (plugins.api flag) — off by default, registered
//     conditionally in main.go outside the RouteRegistrar loop; the task
//     this guard protects against explicitly allows this ("registered ⊆
//     documented", not the reverse).
package gateway_test

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/chat/guest"
	"github.com/kmuhub/kmuhub/internal/featureflag"
	"github.com/kmuhub/kmuhub/internal/gateway"
	"github.com/kmuhub/kmuhub/internal/security"
)

// passthroughAuth stands in for the real JWT auth middleware. Route
// registration never invokes middleware or handlers — chi.Walk only inspects
// registered patterns — so a no-op is sufficient here.
func passthroughAuth(next http.Handler) http.Handler { return next }

// allModulesEnabledEnv fakes os.Getenv so every optional "modules.*" feature
// flag resolves to enabled. Without this, BerichteRoutes, InventarRoutes,
// etc. early-return out of RegisterRoutes before mounting anything (see e.g.
// route_berichte.go: "if !br.flags.IsEnabled(...) { return }"), and their
// endpoints would never reach chi.Walk — silently hiding drift in exactly
// the module groups this guard exists to catch.
func allModulesEnabledEnv(key string) string {
	if strings.HasPrefix(key, "COSMI_MODULE_") && strings.HasSuffix(key, "_ENABLED") {
		return "true"
	}
	return ""
}

// buildGatewayRouter mirrors the RouteRegistrar assembly in
// cmd/gateway/main.go (lines ~239-305): same registrar list, same
// construction order. Every constructor here is invoked with a nil/zero
// backend (nil *ServiceRegistry connections, nil DB pool, nil Redis client)
// because RegisterRoutes only binds handler *method values* as closures —
// it never dials a service or touches the DB during registration. Backends
// are only reached once a handler actually executes, which this test never
// does (chi.Walk inspects the routing tree, it does not serve requests).
func buildGatewayRouter(t *testing.T) chi.Router {
	t.Helper()

	registry := gateway.NewServiceRegistry(nil)
	flagRegistry := featureflag.NewRegistry().Load(allModulesEnabledEnv)

	crmExt := gateway.NewCRMExtRoutes(registry)
	bizExt := gateway.NewBizExtRoutes(registry)
	captchaVerifier := security.NewCaptchaVerifier("", "")
	bookingRoutes := gateway.NewBookingRoutes(registry, captchaVerifier)
	crmRoutes := gateway.NewCRMRoutes(registry, crmExt)
	videoRoutes := gateway.NewVideoRoutes(registry, "", "")
	dashboardService := gateway.NewDashboardStack(nil, nil)

	registrars := []gateway.RouteRegistrar{
		gateway.NewAuthRoutes(registry),
		crmRoutes,
		gateway.NewChatRoutes(registry),
		gateway.NewNotificationRoutes(registry),
		gateway.NewWorkRoutes(registry),
		gateway.NewCalendarRoutes(registry),
		videoRoutes,
		gateway.NewSecurityRoutes(registry),
		gateway.NewEmailRoutes(registry),
		gateway.NewDocumentRoutes(registry),
		gateway.NewBizRoutes(registry),
		gateway.NewBexioRoutes(registry, "test-state-secret"),
		gateway.NewDatevUploadRoutes(registry, "test-state-secret"),
		gateway.NewLexwareRoutes(registry, "test-webhook-secret", false),
		gateway.NewIntegrationRoutes(registry),
		gateway.NewHRRoutes(registry, bizExt),
		gateway.NewInboxRoutes(registry),
		gateway.NewAutomationRoutes(registry),
		gateway.NewDialerRoutes(registry),
		gateway.NewWikiRoutes(registry, flagRegistry),
		gateway.NewHelpdeskRoutes(registry, flagRegistry),
		gateway.NewBerichteRoutes(registry, flagRegistry),
		gateway.NewFormulareRoutes(registry, flagRegistry),
		gateway.NewInventarRoutes(registry, flagRegistry),
		gateway.NewEinkaufRoutes(registry, flagRegistry),
		gateway.NewProduktionRoutes(registry, flagRegistry),
		gateway.NewVertraegeRoutes(registry, flagRegistry),
		gateway.NewRapporteRoutes(registry, flagRegistry),
		gateway.NewSchichtenRoutes(registry, flagRegistry),
		gateway.NewVermietungRoutes(registry, flagRegistry),
		gateway.NewFuhrparkRoutes(registry, flagRegistry),
		gateway.NewFileRoutes(registry),
		gateway.NewGlobalSearchRoutes(registry),
		gateway.NewDashboardRoutes(dashboardService),
		gateway.NewFeatureFlagRoutes(flagRegistry),
		gateway.NewHealthRoutes(nil, registry),
		bookingRoutes,
		gateway.NewSettingsRoutes(registry, flagRegistry),
	}

	r := chi.NewRouter()
	for _, reg := range registrars {
		reg.RegisterRoutes(r, passthroughAuth)
	}

	// Advisory-protocol routes (CRM, separate registration — ZFA Beratungsprotokoll).
	crmRoutes.RegisterAdvisoryRoutes(r, passthroughAuth)

	// Public (unauthenticated) routes registered outside the RouteRegistrar
	// loop in main.go — cheap to construct (no adapters, no live backend
	// needed), and they carry real /api/v1/public/* and /api/v1/guest/*
	// paths, so it is worth including them in the drift check.
	guestRoutes := gateway.NewGuestRoutes(guest.NewService(guest.NewPostgresRepository(nil)), registry)
	guestRoutes.RegisterPublicRoutes(r)
	bookingRoutes.RegisterPublicRoutes(r, passthroughAuth)

	return r
}

// registeredAPIv1Paths walks the router and returns the set of distinct,
// normalised /api/v1/* route patterns it exposes (method-agnostic — this
// guard checks path-level drift, not per-method drift).
func registeredAPIv1Paths(t *testing.T, r chi.Router) map[string]bool {
	t.Helper()

	paths := make(map[string]bool)
	err := chi.Walk(r, func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		route = normalizeChiRoute(route)
		if strings.HasPrefix(route, "/api/v1/") {
			paths[route] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk failed: %v", err)
	}
	return paths
}

// normalizeChiRoute strips the trailing slash chi.Walk adds for handlers
// mounted at the root of an r.Route(...) sub-router (e.g. r.Route("/foo",
// func(r chi.Router) { r.Get("/", h) }) walks as "/foo/", but the OpenAPI
// spec documents it as "/foo").
func normalizeChiRoute(route string) string {
	if len(route) > 1 && strings.HasSuffix(route, "/") {
		return strings.TrimSuffix(route, "/")
	}
	return route
}

// openapiPathKeyRE matches a top-level path key under the "paths:" map in
// api/openapi.yaml, e.g. "  /api/v1/contacts/{id}:". OpenAPI/YAML always
// indents these two spaces under "paths:", one level deeper than the
// "components:" sibling that follows.
var openapiPathKeyRE = regexp.MustCompile(`^  (/\S+):\s*$`)

// documentedPaths parses api/openapi.yaml with a plain line scan (no YAML
// parser dependency — the file's "paths:" map keys follow a rigid,
// consistently-indented format so a regex scan is both simpler and faster
// than pulling in a YAML library just for this). Returns the set of
// documented top-level path keys.
func documentedPaths(t *testing.T, specPath string) map[string]bool {
	t.Helper()

	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", specPath, err)
	}

	documented := make(map[string]bool)
	inPaths := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r") // tolerate CRLF checkouts (Windows autocrlf)
		if !inPaths {
			if strings.HasPrefix(line, "paths:") {
				inPaths = true
			}
			continue
		}
		// A non-indented, non-empty line ends the "paths:" map (e.g. "components:").
		if line != "" && line[0] != ' ' && line[0] != '\t' {
			break
		}
		if m := openapiPathKeyRE.FindStringSubmatch(line); m != nil {
			documented[m[1]] = true
		}
	}

	if len(documented) == 0 {
		t.Fatalf("parsed 0 documented paths from %s — parser is broken or file moved", specPath)
	}
	return documented
}

// apiV1UndocumentedAllowlist lists /api/v1/* routes the gateway registers
// that are deliberately NOT expected to appear in api/openapi.yaml, with a
// reason per entry. Keep this list empty by default — every new entry is a
// conscious decision to exempt a real REST endpoint from API documentation,
// which should be rare and justified.
var apiV1UndocumentedAllowlist = map[string]string{
	// (currently empty — see package doc comment for the endpoints excluded
	// at construction time instead, e.g. /api/v1/ws, /api/v1/caldav/*)
}

// TestOpenAPIRouteDrift fails if the gateway registers an /api/v1/* route
// that has no matching top-level path key in api/openapi.yaml. This is the
// regression guard for the "~185 undocumented endpoints" sweep: it makes it
// impossible to add a new module's routes without also documenting them,
// because CI's "test" job runs `go test ./...` on every push.
func TestOpenAPIRouteDrift(t *testing.T) {
	r := buildGatewayRouter(t)
	registered := registeredAPIv1Paths(t, r)

	specPath := filepath.Join("..", "..", "api", "openapi.yaml")
	documented := documentedPaths(t, specPath)

	var undocumented []string
	for path := range registered {
		if documented[path] {
			continue
		}
		if reason, ok := apiV1UndocumentedAllowlist[path]; ok {
			t.Logf("allowlisted undocumented route %s: %s", path, reason)
			continue
		}
		undocumented = append(undocumented, path)
	}

	if len(undocumented) > 0 {
		sort.Strings(undocumented)
		t.Errorf(
			"%d registered /api/v1/* route(s) missing from api/openapi.yaml (add a path entry, or add a justified apiV1UndocumentedAllowlist entry):\n  %s",
			len(undocumented), strings.Join(undocumented, "\n  "),
		)
	}

	t.Logf("checked %d registered /api/v1/* routes against %d documented paths", len(registered), len(documented))
}

// TestOpenAPIRouteDriftParserSanity is a narrow sanity check on
// documentedPaths itself: it must find well-known, stable paths that have
// existed since before this guard was written. If this fails, the regex
// scan (not the drift check) is broken — fix the parser before trusting
// TestOpenAPIRouteDrift's result.
func TestOpenAPIRouteDriftParserSanity(t *testing.T) {
	specPath := filepath.Join("..", "..", "api", "openapi.yaml")
	documented := documentedPaths(t, specPath)

	for _, want := range []string{
		"/api/v1/auth/login",
		"/api/v1/contacts",
		"/health",
	} {
		if !documented[want] {
			t.Errorf("expected documentedPaths to find %q in %s, it did not", want, specPath)
		}
	}
}
