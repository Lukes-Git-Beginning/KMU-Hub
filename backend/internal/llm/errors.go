package llm

import "errors"

var (
	// ErrEmptyInput is returned when Summarize is called with blank input.
	ErrEmptyInput = errors.New("llm: empty input")
	// ErrNoCompletion is returned when the model responds without usable content.
	ErrNoCompletion = errors.New("llm: model returned no completion")
	// ErrUpstream wraps a retryable upstream failure (5xx / rate limit) after retries.
	ErrUpstream = errors.New("llm: upstream error")
)
