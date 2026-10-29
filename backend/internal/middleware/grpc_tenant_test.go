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

// ─── Forwarded permissions ────────────────────────────────────────────────────

// Only the allow-listed keys cross the gateway->service hop; the rest of the
// JWT permission list stays at the gateway.
func TestPermissionForwarding_OnlyAllowListedKeysArrive(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserPermsKey,
		[]string{"inbox:write", PermTeamInboxManage, "crm:delete"})

	var outgoingCtx context.Context
	if err := TenantOutboundUnaryInterceptor()(ctx, "/svc/Method", nil, nil, nil, fakeUnaryInvoker(&outgoingCtx)); err != nil {
		t.Fatalf("outbound: %v", err)
	}
	md, _ := metadata.FromOutgoingContext(outgoingCtx)
	if got := md.Get(gRPCPermsHeader); len(got) != 1 || got[0] != PermTeamInboxManage {
		t.Fatalf("x-user-perms = %v; want only %q", got, PermTeamInboxManage)
	}

	// The receiving side filters again: a hand-built header cannot smuggle keys in.
	in := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs(gRPCPermsHeader, "crm:delete,"+PermTeamInboxManage))
	res, err := TenantInboundUnaryInterceptor()(in, nil, &grpc.UnaryServerInfo{}, fakeUnaryHandler)
	if err != nil {
		t.Fatalf("inbound: %v", err)
	}
	got := GetUserPermissions(res.(context.Context))
	if len(got) != 1 || got[0] != PermTeamInboxManage {
		t.Fatalf("GetUserPermissions = %v; want only %q", got, PermTeamInboxManage)
	}
}

func TestPermissionForwarding_NoKeyNoHeader(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserPermsKey, []string{"inbox:write"})

	var outgoingCtx context.Context
	_ = TenantOutboundUnaryInterceptor()(ctx, "/svc/Method", nil, nil, nil, fakeUnaryInvoker(&outgoingCtx))
	md, _ := metadata.FromOutgoingContext(outgoingCtx)
	if got := md.Get(gRPCPermsHeader); len(got) != 0 {
		t.Fatalf("unexpected x-user-perms header: %v", got)
	}
}

// recordings:admin decides the delete-any-recording waiver inside the video
// service, so it must cross the hop; unrelated keys must not.
func TestPermissionForwarding_RecordingsAdminCrossesTheHop(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserPermsKey,
		[]string{"recordings:write", PermRecordingsAdmin})

	var outgoingCtx context.Context
	if err := TenantOutboundUnaryInterceptor()(ctx, "/svc/Method", nil, nil, nil, fakeUnaryInvoker(&outgoingCtx)); err != nil {
		t.Fatalf("outbound: %v", err)
	}
	md, _ := metadata.FromOutgoingContext(outgoingCtx)
	if got := md.Get(gRPCPermsHeader); len(got) != 1 || got[0] != PermRecordingsAdmin {
		t.Fatalf("x-user-perms = %v; want only %q", got, PermRecordingsAdmin)
	}
}

// The coarse email keys decide inside the email server whether an account
// change may touch another user's account; the self-setup keys must not cross.
func TestPermissionForwarding_CoarseEmailKeysCrossTheHop(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserPermsKey,
		[]string{"email:read", PermEmailWrite, PermEmailDelete, "email:account:write", "email:mailbox:write"})

	var outgoingCtx context.Context
	if err := TenantOutboundUnaryInterceptor()(ctx, "/svc/Method", nil, nil, nil, fakeUnaryInvoker(&outgoingCtx)); err != nil {
		t.Fatalf("outbound: %v", err)
	}
	md, _ := metadata.FromOutgoingContext(outgoingCtx)
	want := PermEmailWrite + "," + PermEmailDelete
	if got := md.Get(gRPCPermsHeader); len(got) != 1 || got[0] != want {
		t.Fatalf("x-user-perms = %v; want only %q", got, want)
	}
}

// ─── Forwarded roles ──────────────────────────────────────────────────────────

// hr_document_access reads the caller's roles from app.user_roles, which the
// service-side pool sets from GetUserRoles. admin, hr_admin and manager must
// arrive (the HR service holds manager to the reporting line); nothing else.
func TestRoleForwarding_HRRolesCrossTheHop(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserRolesKey, []string{"manager", "hr_admin", "admin", "Büro"})

	var outgoingCtx context.Context
	if err := TenantOutboundUnaryInterceptor()(ctx, "/svc/Method", nil, nil, nil, fakeUnaryInvoker(&outgoingCtx)); err != nil {
		t.Fatalf("outbound: %v", err)
	}
	md, _ := metadata.FromOutgoingContext(outgoingCtx)
	got := md.Get(gRPCRolesHeader)
	if len(got) != 1 || got[0] != "manager,hr_admin,admin" {
		t.Fatalf("x-user-roles = %v; want only %q", got, "manager,hr_admin,admin")
	}

	res, err := TenantInboundUnaryInterceptor()(metadata.NewIncomingContext(context.Background(), md), nil,
		&grpc.UnaryServerInfo{}, fakeUnaryHandler)
	if err != nil {
		t.Fatalf("inbound: %v", err)
	}
	roles := GetUserRoles(res.(context.Context))
	if len(roles) != 3 || roles[0] != "manager" || roles[1] != "hr_admin" || roles[2] != "admin" {
		t.Fatalf("GetUserRoles = %v; want [manager hr_admin admin]", roles)
	}

	// The receiving side filters again: a hand-built header cannot smuggle a role in.
	in := metadata.NewIncomingContext(context.Background(), metadata.Pairs(gRPCRolesHeader, "member,platform_admin"))
	res, err = TenantInboundUnaryInterceptor()(in, nil, &grpc.UnaryServerInfo{}, fakeUnaryHandler)
	if err != nil {
		t.Fatalf("inbound: %v", err)
	}
	if roles := GetUserRoles(res.(context.Context)); len(roles) != 0 {
		t.Fatalf("GetUserRoles = %v; want none", roles)
	}
}

func TestRoleForwarding_MemberSendsNoHeader(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserRolesKey, []string{"member", "Büro"})

	var outgoingCtx context.Context
	_ = TenantOutboundUnaryInterceptor()(ctx, "/svc/Method", nil, nil, nil, fakeUnaryInvoker(&outgoingCtx))
	md, _ := metadata.FromOutgoingContext(outgoingCtx)
	if got := md.Get(gRPCRolesHeader); len(got) != 0 {
		t.Fatalf("unexpected x-user-roles header: %v", got)
	}
}
