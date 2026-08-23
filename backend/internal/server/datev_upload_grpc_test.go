package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/biz/datev"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
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
		{"reauth required", datev.ErrReauthRequired, codes.FailedPrecondition},
		{"wrapped reauth required", fmt.Errorf("datev upload failed: %w", datev.ErrReauthRequired), codes.FailedPrecondition},
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

// TestMapDatevUploadErrorReauthMessageIsActionable is the mutations-probe
// target for the ErrReauthRequired case: before it existed, an expired or
// revoked DATEV refresh token fell into the default branch and came back as
// the same "DATEV upload failed" an actual internal bug would produce,
// leaving the admin no next step. It must now carry a fixed, actionable
// message telling them to reconnect — distinct from the generic one, and
// without leaking the DATEV token endpoint's status code or body (those stay
// server-side in the slog.Error call in oauth.go).
func TestMapDatevUploadErrorReauthMessageIsActionable(t *testing.T) {
	err := mapDatevUploadError(fmt.Errorf("datev upload failed: %w", datev.ErrReauthRequired))
	msg := status.Convert(err).Message()
	if msg == "DATEV upload failed" {
		t.Fatalf("message = %q, want a distinct actionable message, not the generic internal-error one", msg)
	}
	if !strings.Contains(msg, "reconnect") {
		t.Errorf("message = %q, want it to tell the admin to reconnect", msg)
	}
}

// leakyVaultStub implements datev.VaultService; SetSecret always fails with a
// message shaped like a real Vault/network error, including control
// characters, to prove the fix does not just happen to avoid a plain-ASCII
// leak.
type leakyVaultStub struct{}

func (leakyVaultStub) GetSecret(context.Context, string) (string, error) {
	return "", errors.New("not implemented")
}

func (leakyVaultStub) SetSecret(context.Context, string, string, string, uuid.UUID) error {
	return errors.New("vault write failed: dial tcp 10.0.0.5:8200: connect: refused\r\nSet-Cookie: evil=1&x=y")
}

// TestHandleDatevOAuthCallback_MasksInternalError is the mutations-probe
// target for fix-gateway-datev-upload-error-message-leakage:
// HandleDatevOAuthCallback used to return Success:false +
// ErrorMessage: err.Error() directly to the client, letting ExchangeCode's
// wrapped Vault/HTTP errors (which can carry connection strings, hostnames,
// or control characters) reach the gateway's redirect Location raw. It must
// now come back as a masked, fixed-message gRPC error regardless of what the
// underlying service failure said.
func TestHandleDatevOAuthCallback_MasksInternalError(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"tok","expires_in":60,"refresh_token":"refresh"}`))
	}))
	defer tokenServer.Close()

	om := datev.NewOAuthManager(leakyVaultStub{}, "cid", "csecret", tokenServer.URL)
	s := &DatevUploadGRPCServer{uploadService: datev.NewUploadService(nil, nil, nil, nil, nil, nil, om)}

	_, err := s.HandleDatevOAuthCallback(context.Background(), &bizv1.HandleDatevOAuthCallbackRequest{
		TenantId:    uuid.New().String(),
		Code:        "auth-code",
		RedirectUrl: "https://app.example/callback",
	})

	requireGRPCCode(t, err, codes.Internal)
	msg := status.Convert(err).Message()
	if msg != "DATEV OAuth callback failed" {
		t.Errorf("message = %q, want the generic masked message", msg)
	}
	if strings.ContainsAny(msg, "\r\n") || strings.Contains(msg, "10.0.0.5") || strings.Contains(msg, "vault write failed") {
		t.Errorf("message leaked internal details: %q", msg)
	}
}
