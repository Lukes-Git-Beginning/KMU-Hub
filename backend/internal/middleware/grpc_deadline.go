package middleware

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// DeadlineOutboundUnaryInterceptor bounds every outbound unary gRPC call with a
// timeout when the caller's context has no deadline of its own. A hung or very
// slow downstream service then fails fast with DeadlineExceeded (surfaced as
// 504 by the gateway) instead of pinning the calling goroutine — and the HTTP
// request behind it — indefinitely, which would eventually exhaust goroutines.
//
// An existing, shorter caller deadline is always respected: the timeout is only
// applied when the context carries no deadline. Streaming RPCs are unaffected
// (this is a unary client interceptor) so long-lived streams are not severed.
func DeadlineOutboundUnaryInterceptor(timeout time.Duration) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if _, hasDeadline := ctx.Deadline(); !hasDeadline && timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
