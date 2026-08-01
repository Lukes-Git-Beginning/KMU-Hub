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
//
// The reverse direction (documented ⊆ registered) lives in TestOpenAPISpecDrift
// below and is what actually catches a route block that was built, documented
// and generated into the frontend types but never mounted in main.go.
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

	// Berichte: authenticated routes via the registrar loop, the public read of
	// a shared report outside it — same split as main.go.
	berichteRoutes := gateway.NewBerichteRoutes(registry, flagRegistry)
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
		berichteRoutes,
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
		gateway.NewCustomizationRoutes(registry),
	}

	r := chi.NewRouter()
	for _, reg := range registrars {
		reg.RegisterRoutes(r, passthroughAuth)
	}

	// Advisory-protocol routes (CRM, separate registration — ZFA Beratungsprotokoll).
	crmRoutes.RegisterAdvisoryRoutes(r, passthroughAuth)

	// Plugin API routes. main.go registers these only when the plugins.api flag
	// is on (Phase D, off by default), but the routes are documented
	// unconditionally, so the reverse guard has to see them — otherwise the
	// whole /api/v1/plugins/* block would sit on the allowlist and a lost
	// constructor call there would go unnoticed, which is the exact defect this
	// file guards against.
	gateway.NewPluginRoutes(registry).RegisterRoutes(r, passthroughAuth)

	// CalDAV/CardDAV REST routes (/api/v1/caldav/*, /api/v1/admin/caldav/*).
	// main.go builds these through cmd/gateway's unexported adapters, but
	// NewCalDAVRoutes only takes interfaces and http.Handlers, so a nil backend
	// registers the same tree: RegisterRoutes wraps both protocol handlers in
	// closures and binds the REST handlers as method values, touching neither
	// during registration.
	gateway.NewCalDAVRoutes(nil, nil, nil, nil, nil, passthroughAuth).RegisterRoutes(r)

	// Public (unauthenticated) routes registered outside the RouteRegistrar
	// loop in main.go — cheap to construct (no adapters, no live backend
	// needed), and they carry real /api/v1/public/* and /api/v1/guest/*
	// paths, so it is worth including them in the drift check.
	guestRoutes := gateway.NewGuestRoutes(guest.NewService(guest.NewPostgresRepository(nil)), registry)
	guestRoutes.RegisterPublicRoutes(r)
	bookingRoutes.RegisterPublicRoutes(r, passthroughAuth)
	berichteRoutes.RegisterPublicRoutes(r, passthroughAuth)

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

// apiV1UnregisteredAllowlist lists /api/v1/* paths api/openapi.yaml documents
// that buildGatewayRouter deliberately does not register, with a reason per
// entry. Every entry is a promise that the endpoint exists in production even
// though this test cannot see it — anything else belongs in the failure list,
// because a documented path with no endpoint behind it is exactly the defect
// this guard exists to catch.
var apiV1UnregisteredAllowlist = map[string]string{
	"/api/v1/files/upload": "registered in cmd/gateway/main.go on the raw router, not through a RouteRegistrar: " +
		"server.NewFileUploadHandler needs the live WebSocket hub and the chat file service, neither of which " +
		"this DB-less test can construct. Verified present at main.go:381.",
}

// TestOpenAPISpecDrift is the reverse of TestOpenAPIRouteDrift: it fails if
// api/openapi.yaml documents an /api/v1/* path the gateway never registers.
//
// That direction was unguarded for years, and it is the one that hurt. Three
// whole route blocks (produktion ext, DATEV upload, integrations) were fully
// built, fully documented and generated into desktop/src/renderer/src/api/
// types.ts — but their constructors had no caller in cmd/gateway/main.go, so
// every one of those paths was a 404 against a real backend. "registered ⊆
// documented" cannot see that; only "documented ⊆ registered" can.
func TestOpenAPISpecDrift(t *testing.T) {
	r := buildGatewayRouter(t)
	registered := registeredAPIv1Paths(t, r)

	specPath := filepath.Join("..", "..", "api", "openapi.yaml")
	documented := documentedPaths(t, specPath)

	var unregistered []string
	checked := 0
	for path := range documented {
		if !strings.HasPrefix(path, "/api/v1/") {
			continue // non-/api/v1 protocol routes are out of scope, same as forward direction
		}
		checked++
		if registered[path] {
			continue
		}
		if reason, ok := apiV1UnregisteredAllowlist[path]; ok {
			t.Logf("allowlisted unregistered path %s: %s", path, reason)
			continue
		}
		unregistered = append(unregistered, path)
	}

	if len(unregistered) > 0 {
		sort.Strings(unregistered)
		t.Errorf(
			"%d documented /api/v1/* path(s) that the gateway never registers — the endpoint is a 404 (wire the route, remove the spec entry, or add a justified apiV1UnregisteredAllowlist entry):\n  %s",
			len(unregistered), strings.Join(unregistered, "\n  "),
		)
	}

	t.Logf("checked %d documented /api/v1/* paths against %d registered routes", checked, len(registered))
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
