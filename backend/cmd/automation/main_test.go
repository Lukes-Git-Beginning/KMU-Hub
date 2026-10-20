package main

import (
	"context"
	"net"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/kmuhub/kmuhub/internal/middleware"
)

// tenantProbe is a health service that reports the tenant its inbound
// interceptor extracted from the x-tenant-id metadata.
type tenantProbe struct {
	grpc_health_v1.UnimplementedHealthServer
	got chan string
}

func (p *tenantProbe) Check(ctx context.Context, _ *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	id, err := middleware.GetTenantID(ctx)
	if err != nil {
		p.got <- ""
	} else {
		p.got <- id.String()
	}
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

// TestDialService_ForwardsTenantAsMetadata pins that every client the automation
// service dials carries the tenant on the wire. Without the outbound interceptor
// the target RPC sees no tenant and fails with "tenant_id missing from context".
func TestDialService_ForwardsTenantAsMetadata(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	probe := &tenantProbe{got: make(chan string, 2)}
	srv := grpc.NewServer(grpc.UnaryInterceptor(middleware.TenantInboundUnaryInterceptor()))
	grpc_health_v1.RegisterHealthServer(srv, probe)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn := dialService("probe", lis.Addr().String())
	require.NotNil(t, conn)
	t.Cleanup(func() { _ = conn.Close() })
	client := grpc_health_v1.NewHealthClient(conn)

	tenantID := uuid.New()
	ctx := context.WithValue(context.Background(), middleware.TenantIDKey, tenantID.String())
	_, err = client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, tenantID.String(), <-probe.got)

	// No tenant on the context: nothing is forwarded, the server must not invent one.
	_, err = client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, "", <-probe.got)
}
