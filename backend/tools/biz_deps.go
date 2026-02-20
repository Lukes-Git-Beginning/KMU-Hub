//go:build tools

// Package tools holds build-time dependency imports for biz (finance) packages.
// These imports ensure the dependencies remain in go.mod until they are
// used by service code in subsequent plans (12-02, 12-03, etc.).
package tools

import (
	_ "github.com/johnfercher/maroto/v2"
)
