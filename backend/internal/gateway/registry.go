package gateway

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"github.com/kmuhub/kmuhub/internal/middleware"
)

// ServiceConnection holds the gRPC connection state for a single backend service.
type ServiceConnection struct {
	conn    *grpc.ClientConn
	address string
	mu      sync.RWMutex
}

// ServiceRegistry manages lazy gRPC connections to backend services.
// Services are registered with their addresses at startup, and connections
// are created lazily on first use via GetConnection.
type ServiceRegistry struct {
	services  map[string]*ServiceConnection
	mu        sync.RWMutex
	tlsConfig *tls.Config // nil = insecure (local dev)
}

// NewServiceRegistry creates a new empty ServiceRegistry.
// Pass a non-nil tlsConfig to enable mTLS for all gRPC connections.
func NewServiceRegistry(tlsConfig *tls.Config) *ServiceRegistry {
	return &ServiceRegistry{
		services:  make(map[string]*ServiceConnection),
		tlsConfig: tlsConfig,
	}
}

// Register adds a service address to the registry without connecting.
// This should be called during gateway startup for each known backend service.
func (r *ServiceRegistry) Register(name, address string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.services[name] = &ServiceConnection{address: address}
	slog.Info("service registered", "service", name, "address", address)
}

// GetConnection returns a cached gRPC connection for the named service,
// or creates one lazily if none exists yet. Uses double-checked locking
// (read lock first, then write lock if nil) for thread safety.
//
// Note: grpc.NewClient is non-blocking and does not perform I/O. The actual
// TCP connection is established on the first RPC call. If the backend service
// is down, the RPC call itself will fail with a gRPC Unavailable error.
func (r *ServiceRegistry) GetConnection(name string) (*grpc.ClientConn, error) {
	r.mu.RLock()
	svc, ok := r.services[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("service %q not registered", name)
	}

	// Fast path: connection already exists
	svc.mu.RLock()
	if svc.conn != nil {
		conn := svc.conn
		svc.mu.RUnlock()
		return conn, nil
	}
	svc.mu.RUnlock()

	// Slow path: create connection under write lock
	svc.mu.Lock()
	defer svc.mu.Unlock()

	// Double-check after acquiring write lock
	if svc.conn != nil {
		return svc.conn, nil
	}

	conn, err := grpc.NewClient(
		svc.address,
		grpc.WithTransportCredentials(r.transportCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                60 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithUnaryInterceptor(middleware.TenantOutboundUnaryInterceptor()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client for %q at %s: %w", name, svc.address, err)
	}

	svc.conn = conn
	slog.Info("gRPC client created", "service", name, "address", svc.address)
	return conn, nil
}

// RegisteredServices returns the names of all registered services.
func (r *ServiceRegistry) RegisteredServices() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.services))
	for name := range r.services {
		names = append(names, name)
	}
	return names
}

// transportCredentials returns mTLS credentials if configured, or insecure credentials for local dev.
func (r *ServiceRegistry) transportCredentials() credentials.TransportCredentials {
	if r.tlsConfig != nil {
		return credentials.NewTLS(r.tlsConfig)
	}
	return insecure.NewCredentials()
}

// Close gracefully closes all active gRPC connections.
// Should be called during gateway shutdown.
func (r *ServiceRegistry) Close() {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, svc := range r.services {
		svc.mu.Lock()
		if svc.conn != nil {
			if err := svc.conn.Close(); err != nil {
				slog.Error("failed to close gRPC connection", "service", name, "error", err)
			} else {
				slog.Info("gRPC connection closed", "service", name)
			}
			svc.conn = nil
		}
		svc.mu.Unlock()
	}
}
