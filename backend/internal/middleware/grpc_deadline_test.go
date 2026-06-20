package middleware

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
)

func TestDeadlineOutboundUnaryInterceptor(t *testing.T) {
	interceptor := DeadlineOutboundUnaryInterceptor(50 * time.Millisecond)

	t.Run("sets deadline when caller has none", func(t *testing.T) {
		var sawDeadline bool
		var remaining time.Duration
		inv := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
			if dl, ok := ctx.Deadline(); ok {
				sawDeadline = true
				remaining = time.Until(dl)
			}
			return nil
		}
		if err := interceptor(context.Background(), "/svc/M", nil, nil, nil, inv); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !sawDeadline {
			t.Fatal("expected the interceptor to set a deadline when none was present")
		}
		if remaining <= 0 || remaining > 50*time.Millisecond {
			t.Fatalf("deadline out of range: %v", remaining)
		}
	})

	t.Run("respects a shorter caller deadline", func(t *testing.T) {
		parent, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		var remaining time.Duration
		inv := func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
			dl, ok := ctx.Deadline()
			if !ok {
				t.Fatal("expected the caller deadline to be preserved")
			}
			remaining = time.Until(dl)
			return nil
		}
		if err := interceptor(parent, "/svc/M", nil, nil, nil, inv); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if remaining > 10*time.Millisecond {
			t.Fatalf("interceptor must not extend a shorter caller deadline, got %v", remaining)
		}
	})
}
