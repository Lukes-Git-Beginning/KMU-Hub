package server

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/biz/lexware"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// TestMapLexwareError pins the sentinel -> gRPC code mapping mapLexwareError
// relies on: a known Lexware failure must arrive as a specific, client-actable
// code, and anything else must collapse to a masked Internal.
func TestMapLexwareError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"unauthorized", lexware.ErrLexwareUnauthorized, codes.Unauthenticated},
		{"rate limited", lexware.ErrLexwareRateLimited, codes.ResourceExhausted},
		{"not found", lexware.ErrLexwareNotFound, codes.NotFound},
		{"config not found", lexware.ErrConfigNotFound, codes.NotFound},
		{"mapping not found", lexware.ErrMappingNotFound, codes.NotFound},
		{"sync already running", lexware.ErrSyncAlreadyRunning, codes.AlreadyExists},
		{"sync conflict", lexware.ErrSyncConflict, codes.Aborted},
		{"version conflict", lexware.ErrLexwareVersionConflict, codes.Aborted},
		{"invalid field mapping", lexware.ErrInvalidFieldMapping, codes.InvalidArgument},
		{"server error", lexware.ErrLexwareServerError, codes.Unavailable},
		{"unknown cause stays internal", errors.New("dial tcp 10.0.0.5:5432: refused"), codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := status.Code(mapLexwareError(tt.err)); got != tt.want {
				t.Errorf("code = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestMapLexwareErrorHidesUnknownCause is the mutations-probe target: an
// unclassified error must never reach the client with its original message,
// since that message can carry Vault/Postgres internals (connection strings,
// hostnames, constraint names).
func TestMapLexwareErrorHidesUnknownCause(t *testing.T) {
	err := mapLexwareError(errors.New("dial tcp 10.0.0.5:5432: refused"))
	if msg := status.Convert(err).Message(); msg != "internal error" {
		t.Errorf("message = %q, want the generic one", msg)
	}
}

// leakyLexwareVaultStub implements lexware.VaultService; SetSecret always
// fails with a message shaped like a real Vault/network error, including
// control characters, to prove the fix does not just happen to avoid a
// plain-ASCII leak.
type leakyLexwareVaultStub struct{}

func (leakyLexwareVaultStub) GetSecret(context.Context, string) (string, error) {
	return "", errors.New("not implemented")
}

func (leakyLexwareVaultStub) SetSecret(context.Context, string, string, string, uuid.UUID) error {
	return errors.New("vault write failed: dial tcp 10.0.0.5:8200: connect: refused\r\nSet-Cookie: evil=1&x=y")
}

// TestConnectLexware_MasksInternalError used to return
// Success:false + ErrorMessage: err.Error() directly, letting Connect's
// wrapped Vault error (which can carry connection strings, hostnames, or
// control characters) reach the client verbatim. It must now come back as a
// masked gRPC error via mapLexwareError.
func TestConnectLexware_MasksInternalError(t *testing.T) {
	svc := lexware.NewService(nil, nil, nil, leakyLexwareVaultStub{}, nil, nil, nil)
	s := NewLexwareGRPCServer(svc)

	_, err := s.ConnectLexware(context.Background(), &bizv1.ConnectLexwareRequest{
		TenantId: uuid.New().String(),
		ApiKey:   "some-api-key",
	})

	requireGRPCCode(t, err, codes.Internal)
	msg := status.Convert(err).Message()
	if msg != "internal error" {
		t.Errorf("message = %q, want the generic masked message", msg)
	}
	if strings.ContainsAny(msg, "\r\n") || strings.Contains(msg, "10.0.0.5") || strings.Contains(msg, "vault write failed") {
		t.Errorf("message leaked internal details: %q", msg)
	}
}
