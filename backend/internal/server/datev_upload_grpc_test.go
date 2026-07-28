package server

import (
	"errors"
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/biz/datev"
)

// TestMapDatevUploadError pins that a precondition an admin can fix arrives as
// such. Collapsing these into Internal would show "upload failed" for a missing
// client number and leave the admin without a next step, and the gateway maps
// FailedPrecondition to 409 — a code the client renders, unlike a 500.
func TestMapDatevUploadError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"not connected", datev.ErrNotConnected, codes.FailedPrecondition},
		{"no api config", datev.ErrNoAPIConfig, codes.FailedPrecondition},
		{"no upload config", datev.ErrNoUploadConfig, codes.FailedPrecondition},
		{"advisor numbers missing", datev.ErrAdvisorNumbersMissing, codes.FailedPrecondition},
		{"company settings incomplete", datev.ErrCompanySettingsIncomplete, codes.FailedPrecondition},
		{"nothing to upload", datev.ErrNothingToUpload, codes.FailedPrecondition},
		{"wrapped sentinel", fmt.Errorf("datev upload failed: %w", datev.ErrNotConnected), codes.FailedPrecondition},
		{"invoice not found", datev.ErrInvoiceNotFound, codes.NotFound},
		{"inverted period", datev.ErrInvalidPeriod, codes.InvalidArgument},
		{"builder not wired", datev.ErrBuilderNotConfigured, codes.Unavailable},
		{"unknown cause stays internal", errors.New("connection reset"), codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := status.Code(mapDatevUploadError(tt.err)); got != tt.want {
				t.Errorf("code = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestMapDatevUploadErrorHidesUnknownCause keeps internal failures opaque: the
// message of an arbitrary error can carry connection strings or host names.
func TestMapDatevUploadErrorHidesUnknownCause(t *testing.T) {
	err := mapDatevUploadError(errors.New("dial tcp 10.0.0.5:5432: refused"))
	if msg := status.Convert(err).Message(); msg != "DATEV upload failed" {
		t.Errorf("message = %q, want the generic one", msg)
	}
}
