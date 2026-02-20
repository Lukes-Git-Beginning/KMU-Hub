//go:build tools

// Package tools holds build-time dependency imports for integration packages.
// These imports ensure the dependencies remain in go.mod until they are
// used by service code in subsequent plans (17-02, 17-03, etc.).
package tools

import (
	_ "github.com/atc0005/go-teams-notify/v2"
	_ "github.com/infracloudio/msbotbuilder-go/core"
	_ "github.com/slack-go/slack"
)
