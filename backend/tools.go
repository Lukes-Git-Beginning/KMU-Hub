//go:build tools

package tools

// Tool and SDK dependencies that are referenced in go.mod but not yet
// imported by application code. This file ensures `go mod tidy` does
// not remove them.
import (
	_ "github.com/livekit/server-sdk-go/v2"
)
