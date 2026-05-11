package middleware

import (
	"context"
	"testing"

	"github.com/kmuhub/kmuhub/internal/sysctx"
)

// TestCleanupCtx_WrapsContextWithSystemContext locks in the contract that
// IdempotencyCleanupWorker tags every repo call as a system operation, so
// the post-RLS DELETE on idempotency_keys is admitted — the worker has no
// per-request tenant and would otherwise match zero rows.
func TestCleanupCtx_WrapsContextWithSystemContext(t *testing.T) {
	ctx := cleanupCtx(context.Background())
	if !sysctx.Is(ctx) {
		t.Fatal("cleanupCtx must return a context flagged as system-context")
	}
}

// TestCleanupCtx_PreservesParentCancellation ensures the wrap does not break
// shutdown propagation — graceful Close on the parent must still cancel the
// derived ctx.
func TestCleanupCtx_PreservesParentCancellation(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	derived := cleanupCtx(parent)

	cancel()

	select {
	case <-derived.Done():
	default:
		t.Fatal("derived ctx should be cancelled when parent is cancelled")
	}
}
