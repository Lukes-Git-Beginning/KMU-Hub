package workflow

import "errors"

// Domain errors for the automation workflow engine.
var (
	// ErrAutomationNotFound is returned when an automation does not exist.
	ErrAutomationNotFound = errors.New("automation not found")

	// ErrAutomationInactive is returned when trying to execute an inactive automation.
	ErrAutomationInactive = errors.New("automation is inactive")

	// ErrMaxStepsExceeded is returned when an execution exceeds the max step limit.
	ErrMaxStepsExceeded = errors.New("maximum automation steps exceeded")

	// ErrExecutionNotFound is returned when an execution log does not exist.
	ErrExecutionNotFound = errors.New("execution not found")

	// ErrTemplateNotFound is returned when a template does not exist.
	ErrTemplateNotFound = errors.New("automation template not found")

	// ErrInvalidCondition is returned when a condition configuration is malformed.
	ErrInvalidCondition = errors.New("invalid condition configuration")

	// ErrInvalidAction is returned when an action configuration is malformed.
	ErrInvalidAction = errors.New("invalid action configuration")

	// ErrLoopDetected is returned when a chain of automations forms a loop.
	ErrLoopDetected = errors.New("automation loop detected")

	// ErrCircuitBreakerOpen is returned when the circuit breaker is tripped
	// due to too many consecutive failures.
	ErrCircuitBreakerOpen = errors.New("automation circuit breaker is open")
)
