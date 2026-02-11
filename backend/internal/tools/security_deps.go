// Package tools holds build-time dependency imports for security packages.
// These imports ensure the dependencies remain in go.mod until they are
// used by service code in subsequent plans (09-02, 09-03, etc.).
package tools

import (
	_ "github.com/pquerna/otp"
	_ "github.com/wagslane/go-password-validator"
)
