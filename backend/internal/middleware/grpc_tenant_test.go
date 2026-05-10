package middleware

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// fakeUnaryHandler is a grpc.UnaryHandler that returns its input context (as the response value)
// and no error, so tests can inspect what context the interceptor produced.
func fakeUnaryHandler(ctx context.Context, req any) (any, error) {
	return ctx, nil
}

// fakeUnaryInvoker is a grpc.UnaryInvoker that records the outgoing context for
// inspection by the test and forwards the call to the real invoker slot (ignored here).
func fakeUnaryInvoker(capturedCtx *context.Context) grpc.UnaryInvoker {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		*capturedCtx = ctx
		return nil
	}
}

// ─── Outbound interceptor ──────────────────────────────────────────────────────

// TestTenantOutboundUnaryInterceptor_PropagatesTenant verifies that the outbound
// interceptor copies a valid tenant_id from the context into the outgoing metadata.
func TestTenantOutboundUnaryInterceptor_PropagatesTenant(t *testing.T) {
	const tenantID = "11111111-1111-1111-1111-111111111111"
	const userID = "u-abc"

	// Build an incoming context that already has tenant + user (as Auth middleware would set).
	ctx := context.WithValue(context.Background(), TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, UserIDKey, userID)

	var outgoingCtx context.Context
	interceptor := TenantOutboundUnaryInterceptor()
	err := interceptor(ctx, "/svc/Method", nil, nil, nil, fakeUnaryInvoker(&outgoingCtx))
	if err != nil {
		t.Fatalf("interceptor returned unexpected error: %v", err)
	}

	md, ok := metadata.FromOutgoingContext(outgoingCtx)
	if !ok {
		t.Fatal("no outgoing metadata attached to context")
	}

	if got := md.Get(gRPCTenantHeader); len(got) == 0 || got[0] != tenantID {
		t.Errorf("x-tenant-id in metadata = %v; want %q", got, tenantID)
	}
	if got := md.Get(gRPCUserHeader); len(got) == 0 || got[0] != userID {
		t.Errorf("x-user-id in metadata = %v; want %q", got, userID)
	}
}

// TestTenantOutboundUnaryInterceptor_NoTenant verifies that the outbound interceptor
// does NOT attach metadata when the context carries no tenant (unauthenticated path).
func TestTenantOutboundUnaryInterceptor_NoTenant(t *testing.T) {
	ctx := context.Background()

	var outgoingCtx context.Context
	interceptor := TenantOutboundUnaryInterceptor()
	_ = interceptor(ctx, "/svc/Method", nil, nil, nil, fakeUnaryInvoker(&outgoingCtx))

	md, _ := metadata.FromOutgoingContext(outgoingCtx)
	if vals := md.Get(gRPCTenantHeader); len(vals) > 0 && vals[0] != "" {
		t.Errorf("expected no %s header, got %v", gRPCTenantHeader, vals)
	}
}

// ─── Inbound interceptor ──────────────────────────────────────────────────────

// TestTenantInboundUnaryInterceptor_ExtractsTenant verifies that the inbound
// interceptor reads x-tenant-id from incoming metadata and makes it retrievable
// via GetTenantID(ctx).
func TestTenantInboundUnaryInterceptor_ExtractsTenant(t *testing.T) {
	const tenantID = "22222222-2222-2222-2222-222222222222"
	const userID = "u-xyz"

	md := metadata.Pairs(gRPCTenantHeader, tenantID, gRPCUserHeader, userID)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	interceptor := TenantInboundUnaryInterceptor()
	result, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, fakeUnaryHandler)
	if err != nil {
		t.Fatalf("interceptor returned unexpected error: %v", err)
	}

	// The fake handler returns its ctx as the result value.
	handlerCtx, ok := result.(context.Context)
	if !ok {
		t.Fatal("handler did not return a context")
	}

	gotTenant, tenantErr := GetTenantID(handlerCtx)
	if tenantErr != nil {
		t.Fatalf("GetTenantID returned error: %v", tenantErr)
	}
	if gotTenant.String() != tenantID {
		t.Errorf("GetTenantID = %q; want %q", gotTenant.String(), tenantID)
	}

	if got := GetUserID(handlerCtx); got != userID {
		t.Errorf("GetUserID = %q; want %q", got, userID)
	}
}

// TestTenantInboundUnaryInterceptor_NoMetadata verifies that the inbound interceptor
// does NOT fail when no metadata is present, and that GetTenantID returns
// ErrMissingTenantID on the resulting context — the handler is always invoked.
func TestTenantInboundUnaryInterceptor_NoMetadata(t *testing.T) {
	ctx := context.Background() // no incoming metadata at all

	interceptor := TenantInboundUnaryInterceptor()
	result, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, fakeUnaryHandler)
	if err != nil {
		t.Fatalf("interceptor must not fail on missing metadata, got: %v", err)
	}

	handlerCtx, ok := result.(context.Context)
	if !ok {
		t.Fatal("handler did not return a context")
	}

	_, tenantErr := GetTenantID(handlerCtx)
	if !errors.Is(tenantErr, ErrMissingTenantID) {
		t.Errorf("GetTenantID on context-without-tenant = %v; want ErrMissingTenantID", tenantErr)
	}
}

// TestTenantInboundUnaryInterceptor_EmptyTenantHeader verifies that an explicitly
// empty x-tenant-id header is treated as absent (no value written to context).
func TestTenantInboundUnaryInterceptor_EmptyTenantHeader(t *testing.T) {
	md := metadata.Pairs(gRPCTenantHeader, "") // header present but empty
	ctx := metadata.NewIncomingContext(context.Background(), md)

	interceptor := TenantInboundUnaryInterceptor()
	result, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, fakeUnaryHandler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	handlerCtx := result.(context.Context)
	_, tenantErr := GetTenantID(handlerCtx)
	if !errors.Is(tenantErr, ErrMissingTenantID) {
		t.Errorf("expected ErrMissingTenantID for empty header, got %v", tenantErr)
	}
}
