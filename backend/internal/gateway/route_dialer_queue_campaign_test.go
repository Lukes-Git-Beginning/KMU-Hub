package gateway

// Skip and requeue carry the campaign from the path to the dialer service,
// which refuses a queue entry of another campaign. A real gRPC server on
// loopback plays the dialer service.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	dialerv1 "github.com/kmuhub/kmuhub/proto/dialer/v1"
)

type queueCampaignStub struct {
	dialerv1.UnimplementedDialerServiceServer
	skip    *dialerv1.SkipContactRequest
	requeue *dialerv1.RequeueContactRequest
}

func (s *queueCampaignStub) SkipContact(_ context.Context, r *dialerv1.SkipContactRequest) (*emptypb.Empty, error) {
	s.skip = r
	return &emptypb.Empty{}, nil
}

func (s *queueCampaignStub) RequeueContact(_ context.Context, r *dialerv1.RequeueContactRequest) (*emptypb.Empty, error) {
	s.requeue = r
	return &emptypb.Empty{}, nil
}

func TestDialerQueueActions_CarryTheCampaignFromThePath(t *testing.T) {
	stub := &queueCampaignStub{}
	dr := NewDialerRoutes(registryFor(t, "dialer", func(s *grpc.Server) { dialerv1.RegisterDialerServiceServer(s, stub) }))
	router := chi.NewRouter()
	dr.RegisterRoutes(router, guardTestAuth)

	agent := tokenShape{perms: []string{"dialer:calls:write"}}
	const campaign = "66666666-6666-6666-6666-666666666666"
	const entry = "77777777-7777-7777-7777-777777777777"
	base := "/api/v1/dialer/campaigns/" + campaign + "/contacts/" + entry

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, agent.request(http.MethodPost, base+"/skip"))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotNil(t, stub.skip)
	require.Equal(t, campaign, stub.skip.GetCampaignId())
	require.Equal(t, entry, stub.skip.GetCampaignContactId())

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, agent.request(http.MethodPost, base+"/requeue"))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotNil(t, stub.requeue)
	require.Equal(t, campaign, stub.requeue.GetCampaignId())
	require.Equal(t, entry, stub.requeue.GetCampaignContactId())
}
