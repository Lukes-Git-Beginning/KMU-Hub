package middleware

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	// gRPCTenantHeader carries the request tenant_id between gateway and backend
	// services. The value mirrors the JWT `tid` claim that the HTTP Auth
	// middleware already extracted for the same request.
	gRPCTenantHeader = "x-tenant-id"
	// gRPCUserHeader carries the request user_id (sub claim) so service handlers
	// can identify the caller without re-validating the JWT.
	gRPCUserHeader = "x-user-id"
)

// TenantOutboundUnaryInterceptor is a gRPC client interceptor that copies the
// tenant_id and user_id values from the HTTP request context (already populated
// by the Auth middleware) into the outgoing gRPC metadata. The corresponding
// inbound interceptor on the receiving service unpacks them back into context.
//
// This is the client-side half of the Sprint-4-Welle-5 "gRPC tenant trust"
// pattern. It is currently wired as a precursor for chat-service in
// Welle 0.6 (channel CRUD was 401-ing because chat_grpc.go reads the tenant
// from context but the gateway handlers never propagated it).
func TenantOutboundUnaryInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// AppendToOutgoingContext is nil-safe: it constructs an MD if none
		// exists, otherwise merges into the existing one. Avoids the
		// nil-map panic that `md.Set` would trigger when no metadata has
		// been attached yet.
		if tenantID, err := GetTenantID(ctx); err == nil {
			ctx = metadata.AppendToOutgoingContext(ctx, gRPCTenantHeader, tenantID.String())
		}
		if userID := GetUserID(ctx); userID != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, gRPCUserHeader, userID)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// TenantInboundUnaryInterceptor is a gRPC server interceptor that reads
// x-tenant-id and x-user-id from the inbound metadata and stores them in the
// request context under the same keys as the HTTP Auth middleware.
//
// Service handlers can then call middleware.GetTenantID(ctx) and
// middleware.GetUserID(ctx) without caring whether the caller arrived via HTTP
// (gateway routes that already populated context) or via gRPC (this interceptor).
//
// Returns codes.Unauthenticated when tenant_id is absent or empty — every
// authenticated route must carry a tenant.
func TenantInboundUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, _ := metadata.FromIncomingContext(ctx)

		if vals := md.Get(gRPCTenantHeader); len(vals) > 0 && vals[0] != "" {
			ctx = context.WithValue(ctx, TenantIDKey, vals[0])
		}
		if vals := md.Get(gRPCUserHeader); len(vals) > 0 && vals[0] != "" {
			ctx = context.WithValue(ctx, UserIDKey, vals[0])
		}

		// Downstream handlers report their own missing-context errors via
		// GetTenantID() — no hard fail here, since some service-internal RPCs
		// may legitimately run without a tenant (system context, future).
		return handler(ctx, req)
	}
}
