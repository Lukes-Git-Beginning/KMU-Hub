package gateway

import (
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestRegisterAndGetConnection(t *testing.T) {
	reg := NewServiceRegistry(nil)
	reg.Register("test-svc", "localhost:0")

	conn, err := reg.GetConnection("test-svc")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
}

func TestGetConnectionUnregisteredService(t *testing.T) {
	reg := NewServiceRegistry(nil)

	_, err := reg.GetConnection("unknown")
	if err == nil {
		t.Fatal("expected error for unregistered service")
	}
}

func TestGetConnectionCaching(t *testing.T) {
	reg := NewServiceRegistry(nil)
	reg.Register("cache-test", "localhost:0")

	conn1, err := reg.GetConnection("cache-test")
	if err != nil {
		t.Fatalf("first GetConnection failed: %v", err)
	}

	conn2, err := reg.GetConnection("cache-test")
	if err != nil {
		t.Fatalf("second GetConnection failed: %v", err)
	}

	if conn1 != conn2 {
		t.Fatal("expected same connection instance on repeated calls")
	}
}

func TestCloseNoPanic(t *testing.T) {
	reg := NewServiceRegistry(nil)
	reg.Register("svc-a", "localhost:0")
	reg.Register("svc-b", "localhost:0")

	// Create a connection for svc-a so Close has something to clean up
	_, _ = reg.GetConnection("svc-a")

	// Close should not panic, even with a mix of connected and unconnected services
	reg.Close()
}

func TestCloseResetsConnections(t *testing.T) {
	reg := NewServiceRegistry(nil)
	reg.Register("svc", "localhost:0")

	_, err := reg.GetConnection("svc")
	if err != nil {
		t.Fatalf("GetConnection failed: %v", err)
	}

	reg.Close()

	// After Close, GetConnection should create a new connection
	conn, err := reg.GetConnection("svc")
	if err != nil {
		t.Fatalf("GetConnection after Close failed: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection after Close and re-get")
	}
}

func TestRegisteredServices(t *testing.T) {
	reg := NewServiceRegistry(nil)
	reg.Register("auth", "localhost:50051")
	reg.Register("crm", "localhost:50052")

	names := reg.RegisteredServices()
	if len(names) != 2 {
		t.Fatalf("expected 2 registered services, got %d", len(names))
	}

	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["auth"] || !nameSet["crm"] {
		t.Fatalf("expected auth and crm in registered services, got %v", names)
	}
}

func TestGetConnectionWithRealDialOptions(t *testing.T) {
	// Verify that grpc.NewClient with insecure credentials works
	// (same options the registry uses internally)
	conn, err := grpc.NewClient("localhost:0", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc.NewClient failed: %v", err)
	}
	defer func() { _ = conn.Close() }()

	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
}
