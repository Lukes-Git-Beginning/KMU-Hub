package server

// The intake validation (BACKLOG B2) is only worth having if a rejected value
// reaches the caller as a 400. A sentinel that mapHelpdeskError does not know
// falls through to Internal, which the gateway turns into a 500 -- the module
// would then show "Serverfehler" for a typo in a channel name, and nobody
// would think to look at the request.

import (
	"errors"
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/helpdesk"
)

func TestMapHelpdeskError_IntakeValidationIsInvalidArgument(t *testing.T) {
	cases := map[string]error{
		"unknown channel":     helpdesk.ErrInvalidChannel,
		"bad requester email": helpdesk.ErrInvalidRequesterEmail,
		"requester name":      helpdesk.ErrRequesterNameTooLong,
		"custom fields":       helpdesk.ErrInvalidCustomFields,
	}

	for name, sentinel := range cases {
		t.Run(name, func(t *testing.T) {
			if got := status.Code(mapHelpdeskError(sentinel)); got != codes.InvalidArgument {
				t.Errorf("bare sentinel: code = %v, want InvalidArgument", got)
			}
			// Wrapped too: the service layer may add context on its way out,
			// and errors.Is has to keep carrying the verdict.
			wrapped := fmt.Errorf("create ticket: %w", sentinel)
			if got := status.Code(mapHelpdeskError(wrapped)); got != codes.InvalidArgument {
				t.Errorf("wrapped: code = %v, want InvalidArgument", got)
			}
			if !errors.Is(wrapped, sentinel) {
				t.Error("wrapping broke errors.Is")
			}
		})
	}
}
