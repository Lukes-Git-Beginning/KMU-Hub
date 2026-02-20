//go:build tools

// Package tools holds build-time dependency imports for automation packages.
// These imports ensure the dependencies remain in go.mod until they are
// used by service code in subsequent plans (16-02, 16-03, etc.).
package tools

import (
	_ "github.com/expr-lang/expr"
)
