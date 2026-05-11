package middleware

import (
	"testing"

	"github.com/kmuhub/kmuhub/internal/sysctx"
)

// TestEventCtx_WrapsContextWithSystemContext locks in the contract that the
// audit worker tags every CreateAuditEntry call as a system operation, so the
// audit_log SELECT used for hash-chain previous-hash lookup is admitted by
// the post-RLS policy.
func TestEventCtx_WrapsContextWithSystemContext(t *testing.T) {
	if !sysctx.Is(eventCtx()) {
		t.Fatal("eventCtx() must return a context flagged as system-context")
	}
}
