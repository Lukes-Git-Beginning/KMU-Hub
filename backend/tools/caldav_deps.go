//go:build tools

// Package tools holds build-time dependency imports for CalDAV/CardDAV packages.
// These imports ensure the dependencies remain in go.mod until they are
// used by service code in subsequent plans (15-02, 15-03, etc.).
package tools

import (
	_ "github.com/emersion/go-ical"
	_ "github.com/emersion/go-webdav"
)
