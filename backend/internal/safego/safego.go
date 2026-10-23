// Package safego keeps a panic in a background goroutine from taking the whole
// process down. Go terminates the program on an unrecovered panic in ANY
// goroutine; the gRPC recovery interceptor (middleware/grpc_recovery.go) only
// covers request handlers, not workers, tickers or fire-and-forget goroutines.
package safego

import (
	"log/slog"
	"runtime/debug"
)

// Do runs fn and recovers a panic: it is logged with its stack and Do returns
// false. Wrap the BODY of one tick, not the whole loop -- a recover on the loop
// function ends the loop for good after the first panic.
func Do(name string, fn func()) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("background task panic recovered",
				"task", name,
				"panic", r,
				"stack", string(debug.Stack()),
			)
			ok = false
		}
	}()
	fn()
	return true
}

// Go starts fn in a new goroutine with panic recovery.
func Go(name string, fn func()) {
	go Do(name, fn)
}
