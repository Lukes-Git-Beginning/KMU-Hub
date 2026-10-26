package database_test

import (
	"context"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"

	"github.com/kmuhub/kmuhub/internal/database"
)

func TestNewRedisClient_PingsReachableServer(t *testing.T) {
	mr := miniredis.RunT(t)

	client, err := database.NewRedisClient(context.Background(), "redis://"+mr.Addr())
	if err != nil {
		t.Fatalf("NewRedisClient: %v", err)
	}
	defer client.Close()

	if err := client.Set(context.Background(), "k", "v", 0).Err(); err != nil {
		t.Fatalf("set: %v", err)
	}
	if got, _ := mr.Get("k"); got != "v" {
		t.Fatalf("server value = %q; want v", got)
	}
}

func TestNewRedisClient_RejectsMalformedURL(t *testing.T) {
	client, err := database.NewRedisClient(context.Background(), "http://localhost:6379")
	if err == nil || client != nil {
		t.Fatalf("client=%v err=%v; want a parse error and no client", client, err)
	}
	if !strings.Contains(err.Error(), "parse redis url") {
		t.Fatalf("err = %v; want parse redis url", err)
	}
}

// Auth must start even when Redis is down: an unreachable server yields a
// client, not an error.
func TestNewRedisClient_UnreachableServerStillReturnsClient(t *testing.T) {
	mr := miniredis.RunT(t)
	addr := mr.Addr()
	mr.Close()

	// Cancelled context: the ping fails at once instead of burning the retry budget.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client, err := database.NewRedisClient(ctx, "redis://"+addr)
	if err != nil || client == nil {
		t.Fatalf("failed ping must not fail startup: client=%v err=%v", client, err)
	}
	defer client.Close()
}
