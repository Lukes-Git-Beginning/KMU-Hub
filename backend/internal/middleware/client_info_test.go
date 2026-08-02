package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/kmuhub/kmuhub/internal/clientctx"
	"github.com/kmuhub/kmuhub/internal/middleware"
)

// The client info has to survive three hops before it can be written to a
// session row: HTTP request → context → gRPC metadata → service context. Each
// hop is tested here; a break in any of them shows up in production as a
// device list full of blank entries, which is indistinguishable from "the
// feature was never wired".

func TestClientInfo_StoresTrustedIPAndUserAgent(t *testing.T) {
	tests := []struct {
		name        string
		behindProxy bool
		xff         string
		remoteAddr  string
		wantIP      string
	}{
		{
			name:        "direct connection ignores forwarding headers",
			behindProxy: false,
			xff:         "1.2.3.4",
			remoteAddr:  "203.0.113.9:51234",
			wantIP:      "203.0.113.9",
		},
		{
			name:        "behind proxy trusts the last hop, not the client-supplied first",
			behindProxy: true,
			xff:         "1.2.3.4, 203.0.113.9",
			remoteAddr:  "10.0.0.1:51234",
			wantIP:      "203.0.113.9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got clientctx.Info
			h := middleware.ClientInfo(tt.behindProxy)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got = clientctx.From(r.Context())
			}))

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
			req.RemoteAddr = tt.remoteAddr
			req.Header.Set("X-Forwarded-For", tt.xff)
			req.Header.Set("User-Agent", "KMUHub-Electron/1.0")
			h.ServeHTTP(httptest.NewRecorder(), req)

			if got.IP != tt.wantIP {
				t.Errorf("ip = %q, want %q", got.IP, tt.wantIP)
			}
			if got.UserAgent != "KMUHub-Electron/1.0" {
				t.Errorf("user agent = %q", got.UserAgent)
			}
		})
	}
}

// An unauthenticated endpoint takes the user agent straight from the caller,
// so its length is the caller's choice. Bound it before it reaches a column.
func TestClientInfo_TruncatesOversizedUserAgent(t *testing.T) {
	var got clientctx.Info
	h := middleware.ClientInfo(false)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = clientctx.From(r.Context())
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.RemoteAddr = "203.0.113.9:1234"
	req.Header.Set("User-Agent", strings.Repeat("A", 4096))
	h.ServeHTTP(httptest.NewRecorder(), req)

	if len(got.UserAgent) != 512 {
		t.Errorf("user agent length = %d, want it bounded at 512", len(got.UserAgent))
	}
}

// The two interceptors are the halves of one channel: whatever the outbound
// one puts into metadata, the inbound one must put back into the context.
func TestClientInfo_SurvivesTheInterceptorPair(t *testing.T) {
	outbound := middleware.TenantOutboundUnaryInterceptor()
	inbound := middleware.TenantInboundUnaryInterceptor()

	sent := clientctx.Info{IP: "203.0.113.9", UserAgent: "KMUHub-Electron/1.0"}
	ctx := clientctx.With(context.Background(), sent)

	var received clientctx.Info
	err := outbound(ctx, "/auth.v1.AuthService/Login", nil, nil, nil,
		func(outCtx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
			// What the wire carries: outgoing metadata on the client side
			// arrives as incoming metadata on the server side.
			md, ok := metadata.FromOutgoingContext(outCtx)
			if !ok {
				t.Fatal("outbound interceptor attached no metadata")
			}
			serverCtx := metadata.NewIncomingContext(context.Background(), md)

			_, hErr := inbound(serverCtx, nil, &grpc.UnaryServerInfo{}, func(handlerCtx context.Context, _ any) (any, error) {
				received = clientctx.From(handlerCtx)
				return nil, nil
			})
			return hErr
		})
	if err != nil {
		t.Fatalf("interceptor chain: %v", err)
	}

	if received != sent {
		t.Errorf("client info after the round trip = %+v, want %+v", received, sent)
	}
}

// A user agent with a control character would make the whole gRPC call fail at
// the transport layer. Dropping the value is the right trade: a session row
// without a device beats a rejected login.
func TestClientInfo_DropsUnsendableUserAgent(t *testing.T) {
	outbound := middleware.TenantOutboundUnaryInterceptor()
	ctx := clientctx.With(context.Background(), clientctx.Info{
		IP:        "203.0.113.9",
		UserAgent: "bad\nagent",
	})

	err := outbound(ctx, "/auth.v1.AuthService/Login", nil, nil, nil,
		func(outCtx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
			md, _ := metadata.FromOutgoingContext(outCtx)
			if vals := md.Get("x-client-user-agent"); len(vals) != 0 {
				t.Errorf("unsendable user agent was forwarded: %q", vals)
			}
			if vals := md.Get("x-client-ip"); len(vals) != 1 || vals[0] != "203.0.113.9" {
				t.Errorf("ip should still travel: %q", vals)
			}
			return nil
		})
	if err != nil {
		t.Fatalf("interceptor: %v", err)
	}
}
