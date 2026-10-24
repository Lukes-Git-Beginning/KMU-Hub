package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	videov1 "github.com/kmuhub/kmuhub/proto/video/v1"
)

type initiatorConsentStub struct {
	videov1.UnimplementedVideoServiceServer
	got     *videov1.ConfirmInitiatorConsentRequest
	stamped bool
	err     error
}

func (s *initiatorConsentStub) ConfirmInitiatorConsent(_ context.Context, r *videov1.ConfirmInitiatorConsentRequest) (*videov1.ConfirmInitiatorConsentResponse, error) {
	s.got = r
	if s.err != nil {
		return nil, s.err
	}
	return &videov1.ConfirmInitiatorConsentResponse{Stamped: s.stamped}, nil
}

// TestConfirmInitiatorConsent_Wire runs the handler against a real gRPC server.
// Until the RPC lived in video.proto the client could not even marshal the
// request, so the route answered 500 for every user.
func TestConfirmInitiatorConsent_Wire(t *testing.T) {
	caller := uuid.New().String()
	recID := uuid.New().String()

	call := func(t *testing.T, stub *initiatorConsentStub) *httptest.ResponseRecorder {
		t.Helper()
		routes := NewVideoRoutes(registryFor(t, "work", func(s *grpc.Server) { videov1.RegisterVideoServiceServer(s, stub) }), "", "")
		// A foreign user_id in query and body must not reach the service.
		req := httptest.NewRequest(http.MethodPost, "/x?user_id="+uuid.New().String(), strings.NewReader(`{"user_id":"`+uuid.New().String()+`"}`))
		req = withChiURLParam(withAuth(req, caller, testTenantID), "id", recID)
		rec := httptest.NewRecorder()
		routes.HandleConfirmInitiatorConsent(rec, req)
		return rec
	}

	t.Run("stamped answers 200 with the JWT user", func(t *testing.T) {
		stub := &initiatorConsentStub{stamped: true}
		rec := call(t, stub)
		assertStatus(t, rec, http.StatusOK)
		var body map[string]bool
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, map[string]bool{"stamped": true}, body)
		require.NotNil(t, stub.got)
		assert.Equal(t, recID, stub.got.GetRecordingId())
		assert.Equal(t, caller, stub.got.GetUserId())
		assert.Equal(t, testTenantID.String(), stub.got.GetTenantId())
	})

	t.Run("not stamped answers 412", func(t *testing.T) {
		assertStatus(t, call(t, &initiatorConsentStub{stamped: false}), http.StatusPreconditionFailed)
	})

	t.Run("a foreign recording answers 404", func(t *testing.T) {
		assertStatus(t, call(t, &initiatorConsentStub{err: status.Error(codes.NotFound, "recording not found")}), http.StatusNotFound)
	})
}
