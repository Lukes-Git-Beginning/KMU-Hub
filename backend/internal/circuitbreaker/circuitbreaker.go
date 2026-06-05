// Package circuitbreaker provides a simple, thread-safe 3-state circuit breaker
// (Closed → Open → Half-Open) with no external dependencies.
package circuitbreaker

import (
	"errors"
	"log/slog"
	"sync"
	"time"
)

// ErrCircuitOpen is returned by Execute when the circuit is open and requests
// are being shed immediately without reaching the underlying operation.
var ErrCircuitOpen = errors.New("circuit breaker: circuit is open")

// State represents the current state of the circuit breaker.
type State int

const (
	// StateClosed is the normal operating state; requests pass through.
	StateClosed State = iota
	// StateOpen means the breaker has tripped; requests are rejected immediately.
	StateOpen
	// StateHalfOpen means the cooldown has elapsed and one probe request is
	// allowed through to test whether the downstream is healthy again.
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Option is a functional option for configuring a CircuitBreaker.
type Option func(*CircuitBreaker)

// WithFailureThreshold sets the number of consecutive failures required to open
// the circuit. Default: 5.
func WithFailureThreshold(n int) Option {
	return func(cb *CircuitBreaker) { cb.failureThreshold = n }
}

// WithCooldown sets the duration to wait in the open state before probing again.
// Default: 30s.
func WithCooldown(d time.Duration) Option {
	return func(cb *CircuitBreaker) { cb.cooldown = d }
}

// WithNow overrides the time source — useful in tests to control time precisely
// without sleeps.
func WithNow(fn func() time.Time) Option {
	return func(cb *CircuitBreaker) { cb.now = fn }
}

// CircuitBreaker is a 3-state (Closed / Open / Half-Open) circuit breaker.
// It is safe for concurrent use by multiple goroutines.
type CircuitBreaker struct {
	mu               sync.Mutex
	name             string
	state            State
	consecutiveFails int
	failureThreshold int
	cooldown         time.Duration
	openedAt         time.Time
	now              func() time.Time
}

// New creates a new CircuitBreaker with the given name and options.
func New(name string, opts ...Option) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:             name,
		state:            StateClosed,
		failureThreshold: 5,
		cooldown:         30 * time.Second,
		now:              time.Now,
	}
	for _, opt := range opts {
		opt(cb)
	}
	return cb
}

// Execute runs fn if the circuit allows it.
//
//   - Closed: fn is called. Success → reset consecutive-fail counter.
//     Failure → increment counter; if threshold reached → open the circuit.
//   - Open: if the cooldown has not elapsed, return ErrCircuitOpen immediately.
//     If cooldown has elapsed → transition to Half-Open and allow the probe.
//   - Half-Open: fn is called as a probe. Success → close the circuit.
//     Failure → re-open the circuit and reset the cooldown timer.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()

	switch cb.state {
	case StateOpen:
		if cb.now().Sub(cb.openedAt) < cb.cooldown {
			cb.mu.Unlock()
			return ErrCircuitOpen
		}
		// Cooldown elapsed → allow one probe through.
		cb.transitionTo(StateHalfOpen)
	}

	// StateClosed or StateHalfOpen: release lock before calling fn so we don't
	// hold the mutex while the downstream I/O is in flight.
	currentState := cb.state
	cb.mu.Unlock()

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.recordFailureLocked(currentState)
	} else {
		cb.recordSuccessLocked()
	}
	return err
}

// recordSuccessLocked must be called with cb.mu held.
func (cb *CircuitBreaker) recordSuccessLocked() {
	if cb.state == StateHalfOpen {
		cb.transitionTo(StateClosed)
	}
	cb.consecutiveFails = 0
}

// recordFailureLocked must be called with cb.mu held.
func (cb *CircuitBreaker) recordFailureLocked(priorState State) {
	cb.consecutiveFails++

	if priorState == StateHalfOpen {
		// Probe failed → immediately re-open.
		cb.transitionTo(StateOpen)
		return
	}

	if cb.consecutiveFails >= cb.failureThreshold {
		cb.transitionTo(StateOpen)
	}
}

// transitionTo changes the state and logs the transition. Must be called with
// cb.mu held.
func (cb *CircuitBreaker) transitionTo(next State) {
	prev := cb.state
	cb.state = next

	switch next {
	case StateOpen:
		cb.openedAt = cb.now()
		cb.consecutiveFails = 0
		slog.Warn("circuit breaker opened",
			"name", cb.name,
			"from", prev.String(),
		)
	case StateHalfOpen:
		slog.Info("circuit breaker probing (half-open)",
			"name", cb.name,
		)
	case StateClosed:
		slog.Info("circuit breaker closed (recovered)",
			"name", cb.name,
		)
	}
}

// State returns the current state of the circuit breaker (for observability).
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}
