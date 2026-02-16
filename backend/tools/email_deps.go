//go:build tools

// Package tools holds build-time dependency imports for email packages.
// These imports ensure the dependencies remain in go.mod until they are
// used by service code in subsequent plans (10-05, 10-06, etc.).
package tools

import (
	_ "github.com/emersion/go-imap/v2"
	_ "github.com/emersion/go-message"
	_ "github.com/emersion/go-sasl"
	_ "github.com/emersion/go-smtp"
	_ "github.com/emersion/go-vcard"
	_ "github.com/gatherstars-com/jwz"
	_ "github.com/gocarina/gocsv"
)
