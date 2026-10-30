package fuhrpark

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// Two moves onto one window can end in a deadlock instead of an exclusion violation; the
// repository has to report the loser as a conflict either way. A real deadlock cannot be forced
// reliably, so the mapping is pinned here.
func TestAsBookingConflict(t *testing.T) {
	t.Parallel()

	for _, code := range []string{pgErrCodeExclusionViolation, pgErrCodeDeadlockDetected} {
		err := asBookingConflict(fmt.Errorf("exec: %w", &pgconn.PgError{Code: code}))
		if !errors.Is(err, ErrBookingConflict) {
			t.Errorf("code %s: expected ErrBookingConflict, got %v", code, err)
		}
	}

	other := &pgconn.PgError{Code: "23505"}
	if err := asBookingConflict(other); !errors.Is(err, other) || errors.Is(err, ErrBookingConflict) {
		t.Errorf("unrelated code must pass through unchanged, got %v", err)
	}
	if err := asBookingConflict(nil); err != nil {
		t.Errorf("nil must stay nil, got %v", err)
	}
}
