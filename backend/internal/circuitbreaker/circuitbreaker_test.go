package circuitbreaker_test

import (
	"errors"
	"testing"
	"time"

	"github.com/kmuhub/kmuhub/internal/circuitbreaker"
)

// errSentinel is a generic downstream error for tests.
var errSentinel = errors.New("downstream error")

// failFn returns a function that always returns errSentinel.
func failFn() func() error { return func() error { return errSentinel } }

// okFn returns a function that always succeeds.
func okFn() func() error { return func() error { return nil } }

// newTestBreaker builds a breaker with deterministic time and a threshold of 3.
func newTestBreaker(t *testing.T, startTime time.Time, cooldown time.Duration) (*circuitbreaker.CircuitBreaker, *time.Time) {
	t.Helper()
	now := startTime
	cb := circuitbreaker.New("test",
		circuitbreaker.WithFailureThreshold(3),
		circuitbreaker.WithCooldown(cooldown),
		circuitbreaker.WithNow(func() time.Time { return now }),
	)
	return cb, &now
}

// TestInitialState verifies that a new breaker starts closed.
func TestInitialState(t *testing.T) {
	cb := circuitbreaker.New("x")
	if got := cb.State(); got != circuitbreaker.StateClosed {
		t.Fatalf("expected StateClosed, got %s", got)
	}
}

// TestSuccessDoesNotOpenCircuit verifies many successes leave the circuit closed.
func TestSuccessDoesNotOpenCircuit(t *testing.T) {
	cb := circuitbreaker.New("x", circuitbreaker.WithFailureThreshold(3))
	for i := 0; i < 10; i++ {
		if err := cb.Execute(okFn()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if got := cb.State(); got != circuitbreaker.StateClosed {
		t.Fatalf("expected StateClosed after successes, got %s", got)
	}
}

// TestOpenAfterThresholdFailures verifies the circuit opens after N consecutive
// failures.
func TestOpenAfterThresholdFailures(t *testing.T) {
	cb, _ := newTestBreaker(t, time.Now(), 30*time.Second)

	// Two failures: still closed.
	for i := 0; i < 2; i++ {
		_ = cb.Execute(failFn())
	}
	if got := cb.State(); got != circuitbreaker.StateClosed {
		t.Fatalf("expected closed after 2 failures (threshold=3), got %s", got)
	}

	// Third failure: should open.
	_ = cb.Execute(failFn())
	if got := cb.State(); got != circuitbreaker.StateOpen {
		t.Fatalf("expected open after 3 failures, got %s", got)
	}
}

// TestOpenCircuitSheddingRequests verifies that an open circuit returns
// ErrCircuitOpen immediately.
func TestOpenCircuitSheddingRequests(t *testing.T) {
	cb, _ := newTestBreaker(t, time.Now(), 30*time.Second)

	for i := 0; i < 3; i++ {
		_ = cb.Execute(failFn())
	}

	// Subsequent calls must be shed.
	for i := 0; i < 5; i++ {
		err := cb.Execute(okFn())
		if !errors.Is(err, circuitbreaker.ErrCircuitOpen) {
			t.Fatalf("expected ErrCircuitOpen, got %v", err)
		}
	}
}

// TestHalfOpenAfterCooldown verifies that after the cooldown one probe is
// allowed through.
func TestHalfOpenAfterCooldown(t *testing.T) {
	now := time.Now()
	cb := circuitbreaker.New("test",
		circuitbreaker.WithFailureThreshold(3),
		circuitbreaker.WithCooldown(10*time.Second),
		circuitbreaker.WithNow(func() time.Time { return now }),
	)

	// Trip the breaker.
	for i := 0; i < 3; i++ {
		_ = cb.Execute(failFn())
	}
	if cb.State() != circuitbreaker.StateOpen {
		t.Fatal("expected open")
	}

	// Advance time beyond cooldown.
	now = now.Add(11 * time.Second)

	// Execute should transition to half-open, call fn, and since fn succeeds →
	// close.
	err := cb.Execute(okFn())
	if err != nil {
		t.Fatalf("unexpected error after cooldown: %v", err)
	}
	if got := cb.State(); got != circuitbreaker.StateClosed {
		t.Fatalf("expected closed after successful probe, got %s", got)
	}
}

// TestHalfOpenProbeFailureReopens verifies that a failing probe re-opens the
// circuit.
func TestHalfOpenProbeFailureReopens(t *testing.T) {
	now := time.Now()
	cb := circuitbreaker.New("test",
		circuitbreaker.WithFailureThreshold(3),
		circuitbreaker.WithCooldown(10*time.Second),
		circuitbreaker.WithNow(func() time.Time { return now }),
	)

	// Trip the breaker.
	for i := 0; i < 3; i++ {
		_ = cb.Execute(failFn())
	}

	// Advance past cooldown.
	now = now.Add(11 * time.Second)

	// Failing probe → should re-open.
	_ = cb.Execute(failFn())
	if got := cb.State(); got != circuitbreaker.StateOpen {
		t.Fatalf("expected open after failed probe, got %s", got)
	}
}

// TestCooldownNotElapsedKeepsCircuitOpen verifies that before cooldown expires
// all requests are still shed.
func TestCooldownNotElapsedKeepsCircuitOpen(t *testing.T) {
	now := time.Now()
	cb := circuitbreaker.New("test",
		circuitbreaker.WithFailureThreshold(3),
		circuitbreaker.WithCooldown(30*time.Second),
		circuitbreaker.WithNow(func() time.Time { return now }),
	)

	for i := 0; i < 3; i++ {
		_ = cb.Execute(failFn())
	}

	// Advance time but NOT past cooldown.
	now = now.Add(15 * time.Second)

	err := cb.Execute(okFn())
	if !errors.Is(err, circuitbreaker.ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen before cooldown, got %v", err)
	}
}

// TestSuccessResetsConsecutiveFailCount verifies that a success between
// failures resets the consecutive counter.
func TestSuccessResetsConsecutiveFailCount(t *testing.T) {
	cb := circuitbreaker.New("test", circuitbreaker.WithFailureThreshold(3))

	// Two failures, then a success.
	_ = cb.Execute(failFn())
	_ = cb.Execute(failFn())
	_ = cb.Execute(okFn())

	// Two more failures: should still be closed (counter was reset).
	_ = cb.Execute(failFn())
	_ = cb.Execute(failFn())
	if got := cb.State(); got != circuitbreaker.StateClosed {
		t.Fatalf("expected closed (counter reset by success), got %s", got)
	}
}

// TestStateString covers the String() method for readable output.
func TestStateString(t *testing.T) {
	cases := []struct {
		s    circuitbreaker.State
		want string
	}{
		{circuitbreaker.StateClosed, "closed"},
		{circuitbreaker.StateOpen, "open"},
		{circuitbreaker.StateHalfOpen, "half-open"},
	}
	for _, tc := range cases {
		if got := tc.s.String(); got != tc.want {
			t.Errorf("State(%d).String() = %q, want %q", tc.s, got, tc.want)
		}
	}
}

// TestMultipleOpenCooldownCycles verifies that the breaker can trip, recover,
// and trip again correctly.
func TestMultipleOpenCooldownCycles(t *testing.T) {
	now := time.Now()
	cb := circuitbreaker.New("test",
		circuitbreaker.WithFailureThreshold(2),
		circuitbreaker.WithCooldown(5*time.Second),
		circuitbreaker.WithNow(func() time.Time { return now }),
	)

	// Cycle 1: trip.
	_ = cb.Execute(failFn())
	_ = cb.Execute(failFn())
	if cb.State() != circuitbreaker.StateOpen {
		t.Fatal("cycle1: expected open")
	}

	// Recover.
	now = now.Add(6 * time.Second)
	if err := cb.Execute(okFn()); err != nil {
		t.Fatalf("cycle1 probe: unexpected error: %v", err)
	}
	if cb.State() != circuitbreaker.StateClosed {
		t.Fatal("cycle1: expected closed after recovery")
	}

	// Cycle 2: trip again.
	_ = cb.Execute(failFn())
	_ = cb.Execute(failFn())
	if cb.State() != circuitbreaker.StateOpen {
		t.Fatal("cycle2: expected open")
	}
}
