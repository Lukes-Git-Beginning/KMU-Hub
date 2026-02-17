//go:build tools

// Package tools holds build-time dependency imports for document packages.
// These imports ensure the dependencies remain in go.mod until they are
// used by service code in subsequent plans (11-02, 11-03, etc.).
package tools

import (
	_ "code.sajari.com/docconv/v2"
)
