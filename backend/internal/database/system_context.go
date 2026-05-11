package database

import (
	"context"

	"github.com/kmuhub/kmuhub/internal/sysctx"
)

// WithSystemContext marks ctx as a system operation. Re-export of
// sysctx.With kept for backward compatibility with the 10 worker sites
// wrapped in Welle 1. New callers (notably internal/auth, which would
// import-cycle through this package) should use sysctx.With directly.
func WithSystemContext(ctx context.Context) context.Context {
	return sysctx.With(ctx)
}

// IsSystemContext reports whether ctx was wrapped with WithSystemContext
// (or sysctx.With). Re-export of sysctx.Is.
func IsSystemContext(ctx context.Context) bool {
	return sysctx.Is(ctx)
}
