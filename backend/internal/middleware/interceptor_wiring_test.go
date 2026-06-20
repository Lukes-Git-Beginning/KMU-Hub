package middleware_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAllGRPCServicesWireTenantInterceptor is an architecture guard against the
// R3-P0-3 class of bug: a service binary that builds an inbound gRPC server but
// forgets middleware.TenantInboundUnaryInterceptor. Without it, the per-request
// tenant id never reaches the handler, current_tenant_id() stays NULL and every
// RLS-protected table returns zero rows (phantom-404 when the module is enabled).
//
// Rule: any cmd/<svc>/main.go that installs a server-side
// grpc.ChainUnaryInterceptor MUST also reference TenantInboundUnaryInterceptor.
// The gateway sets the tenant from the JWT and forwards it as metadata, so it
// uses grpc.WithChainUnaryInterceptor (client/dial) and is correctly skipped.
func TestAllGRPCServicesWireTenantInterceptor(t *testing.T) {
	root := moduleRoot(t)
	mains, err := filepath.Glob(filepath.Join(root, "cmd", "*", "main.go"))
	if err != nil {
		t.Fatalf("glob cmd/*/main.go: %v", err)
	}
	if len(mains) == 0 {
		t.Fatalf("no cmd/*/main.go found under %s", root)
	}

	checked := 0
	for _, path := range mains {
		src, readErr := os.ReadFile(path) //nolint:gosec // test reads repo source
		if readErr != nil {
			t.Fatalf("read %s: %v", path, readErr)
		}
		content := string(src)

		// Only services that build a server-side unary interceptor chain are in scope.
		if !strings.Contains(content, "grpc.ChainUnaryInterceptor(") {
			continue
		}
		checked++

		svc := filepath.Base(filepath.Dir(path))
		if !strings.Contains(content, "TenantInboundUnaryInterceptor") {
			t.Errorf("cmd/%s/main.go installs a server gRPC interceptor chain but is missing middleware.TenantInboundUnaryInterceptor() — RLS-scoped tables would return zero rows (R3-P0-3)", svc)
		}
	}

	// Sanity: we expect the full backend service fleet (23 services) to be covered.
	if checked < 20 {
		t.Fatalf("expected to check ~23 gRPC service binaries, only matched %d — glob or detection likely broke", checked)
	}
}

// moduleRoot walks up from the test working directory until it finds go.mod.
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found walking up from test dir")
		}
		dir = parent
	}
}
