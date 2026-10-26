package middleware

import (
	"context"
	"slices"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/kmuhub/kmuhub/internal/clientctx"
)

const (
	// gRPCTenantHeader carries the request tenant_id between gateway and backend
	// services. The value mirrors the JWT `tid` claim that the HTTP Auth
	// middleware already extracted for the same request.
	gRPCTenantHeader = "x-tenant-id"
	// gRPCUserHeader carries the request user_id (sub claim) so service handlers
	// can identify the caller without re-validating the JWT.
	gRPCUserHeader = "x-user-id"
	// gRPCPermsHeader carries the subset of the caller's permissions listed in
	// forwardedPermissions. The full list lives in the JWT and is not sent: a
	// service that needs a capability decision (not just a gateway guard)
	// opts in by adding its key below.
	gRPCPermsHeader = "x-user-perms"
	// gRPCRolesHeader carries the caller's roles listed in forwardedRoles.
	// RLS policy hr_document_access (000128) reads them via app.user_roles,
	// which internal/database sets from GetUserRoles — without the header that
	// GUC is empty in every backend service.
	gRPCRolesHeader = "x-user-roles"
	// gRPCClientIPHeader and gRPCClientUAHeader carry the originating client's
	// IP and user agent (see internal/clientctx). Unlike the two above they are
	// also present on pre-JWT calls — login has no tenant yet but still needs
	// to record which device it came from.
	gRPCClientIPHeader = "x-client-ip"
	gRPCClientUAHeader = "x-client-user-agent"
)

// PermTeamInboxManage lets its holder manage the members of every team inbox
// of the tenant without being an admin of that inbox.
const PermTeamInboxManage = "kommunikation:team_inbox:manage"

// PermRecordingsAdmin lets its holder delete a recording of the tenant that
// somebody else started.
const PermRecordingsAdmin = "recordings:admin"

// PermEmailWrite and PermEmailDelete are the coarse email keys of admin
// provisioning: their holder sets up, changes and removes the mail accounts of
// other users of the tenant and maintains the shared email templates. Without
// them the email server confines account and template changes to the caller's
// own accounts and personal templates.
//
// lean: the hop carries key names, not scopes; a custom role holding the coarse
// key at scope 'own' would reach other users' accounts. Forward the scope when a
// role ever needs the coarse email keys narrowed (today admin only, 'all').
const (
	PermEmailWrite  = "email:write"
	PermEmailDelete = "email:delete"
)

// forwardedPermissions are the capability keys a backend service decides on
// itself, so they must cross the gateway->service hop.
var forwardedPermissions = []string{PermTeamInboxManage, PermRecordingsAdmin, PermEmailWrite, PermEmailDelete}

// forwardedRoles are the role names a backend decides on through RLS
// (hr_document_access: admin and hr_admin read every tier and may write).
//
// lean: manager is deliberately missing. The policy grants the manager role
// the manager AND employee tier of the whole tenant, so forwarding it would
// open every colleague's file (gehaltsabrechnung included) through
// /hr/personnel-documents. Add it together with the reporting-line limit of
// feat-hr-team-documents-for-managers.
var forwardedRoles = []string{"admin", "hr_admin"}

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
		if fwd := forwardedOf(GetUserPermissions(ctx)); len(fwd) > 0 {
			ctx = metadata.AppendToOutgoingContext(ctx, gRPCPermsHeader, strings.Join(fwd, ","))
		}
		if roles := allowedOf(GetUserRoles(ctx), forwardedRoles); len(roles) > 0 {
			ctx = metadata.AppendToOutgoingContext(ctx, gRPCRolesHeader, strings.Join(roles, ","))
		}
		if client := clientctx.From(ctx); client.IP != "" || client.UserAgent != "" {
			// Metadata values must be valid HTTP/2 header values; a user agent
			// with a stray newline would make the whole call fail with a
			// transport error, so anything unprintable is dropped rather than
			// costing the caller their login.
			if isPrintableASCII(client.IP) {
				ctx = metadata.AppendToOutgoingContext(ctx, gRPCClientIPHeader, client.IP)
			}
			if isPrintableASCII(client.UserAgent) {
				ctx = metadata.AppendToOutgoingContext(ctx, gRPCClientUAHeader, client.UserAgent)
			}
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
// It never rejects a call itself: an absent or empty tenant_id simply leaves
// TenantIDKey unset, and each handler that needs a tenant fails on its own
// GetTenantID check (typically with codes.Unauthenticated).
func TenantInboundUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, _ := metadata.FromIncomingContext(ctx)

		if vals := md.Get(gRPCTenantHeader); len(vals) > 0 && vals[0] != "" {
			ctx = context.WithValue(ctx, TenantIDKey, vals[0])
		}
		if vals := md.Get(gRPCUserHeader); len(vals) > 0 && vals[0] != "" {
			ctx = context.WithValue(ctx, UserIDKey, vals[0])
		}

		// Only forwardedPermissions arrive, never the caller's full list, so
		// GetUserPermissions on a service context is a partial view by design.
		if vals := md.Get(gRPCPermsHeader); len(vals) > 0 && vals[0] != "" {
			ctx = context.WithValue(ctx, UserPermsKey, forwardedOf(strings.Split(vals[0], ",")))
		}
		if vals := md.Get(gRPCRolesHeader); len(vals) > 0 && vals[0] != "" {
			ctx = context.WithValue(ctx, UserRolesKey, allowedOf(strings.Split(vals[0], ","), forwardedRoles))
		}

		var client clientctx.Info
		if vals := md.Get(gRPCClientIPHeader); len(vals) > 0 {
			client.IP = vals[0]
		}
		if vals := md.Get(gRPCClientUAHeader); len(vals) > 0 {
			client.UserAgent = vals[0]
		}
		if client.IP != "" || client.UserAgent != "" {
			ctx = clientctx.With(ctx, client)
		}

		// Downstream handlers report their own missing-context errors via
		// GetTenantID() — no hard fail here, since some service-internal RPCs
		// may legitimately run without a tenant (system context, future).
		return handler(ctx, req)
	}
}

// forwardedOf keeps the keys of perms that are in forwardedPermissions.
func forwardedOf(perms []string) []string {
	return allowedOf(perms, forwardedPermissions)
}

// allowedOf keeps the values of vals that are in allowed, without duplicates.
func allowedOf(vals, allowed []string) []string {
	var out []string
	for _, v := range vals {
		if slices.Contains(allowed, v) && !slices.Contains(out, v) {
			out = append(out, v)
		}
	}
	return out
}

// isPrintableASCII reports whether s is safe to put into gRPC metadata: only
// printable ASCII, no control characters. Empty is not printable — an empty
// value is nothing worth sending.
func isPrintableASCII(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7e {
			return false
		}
	}
	return true
}
