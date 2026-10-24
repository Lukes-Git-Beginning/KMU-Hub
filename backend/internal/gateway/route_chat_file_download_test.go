package gateway

// The desktop downloads task attachments through GET /files/{id}/download, the chat file
// route. An id that is no chat file falls back to the work service's task files; chat file
// downloads stay as they were.

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	chatv1 "github.com/kmuhub/kmuhub/proto/chat/v1"
	workv1 "github.com/kmuhub/kmuhub/proto/work/v1"
)

const (
	downloadChatFileID = "11111111-1111-4111-8111-111111111111"
	downloadTaskFileID = "22222222-2222-4222-8222-222222222222"
)

type chatFileDownloadStub struct {
	chatv1.UnimplementedChatServiceServer
}

func (chatFileDownloadStub) GetFileDownloadURL(_ context.Context, r *chatv1.GetFileDownloadURLRequest) (*chatv1.GetFileDownloadURLResponse, error) {
	if r.GetFileId() == downloadChatFileID {
		return &chatv1.GetFileDownloadURLResponse{Url: "https://files.test/chat"}, nil
	}
	return nil, status.Error(codes.NotFound, "file not found")
}

type taskFileDownloadStub struct {
	workv1.UnimplementedWorkServiceServer
	asked []string
}

func (s *taskFileDownloadStub) GetTaskFileDownloadURL(_ context.Context, r *workv1.GetTaskFileDownloadURLRequest) (*workv1.GetTaskFileDownloadURLResponse, error) {
	s.asked = append(s.asked, r.GetId())
	if r.GetId() == downloadTaskFileID {
		return &workv1.GetTaskFileDownloadURLResponse{Url: "https://files.test/task"}, nil
	}
	return nil, status.Error(codes.NotFound, "task file not found")
}

func newFileDownloadRoutes(t *testing.T) (*ChatRoutes, *taskFileDownloadStub) {
	t.Helper()
	serve := func(register func(*grpc.Server)) string {
		lis, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		srv := grpc.NewServer()
		register(srv)
		go func() { _ = srv.Serve(lis) }()
		t.Cleanup(srv.Stop)
		return lis.Addr().String()
	}
	work := &taskFileDownloadStub{}
	reg := NewServiceRegistry(nil)
	reg.Register("chat", serve(func(s *grpc.Server) { chatv1.RegisterChatServiceServer(s, chatFileDownloadStub{}) }))
	reg.Register("work", serve(func(s *grpc.Server) { workv1.RegisterWorkServiceServer(s, work) }))
	t.Cleanup(reg.Close)
	return NewChatRoutes(reg), work
}

func downloadFile(t *testing.T, routes *ChatRoutes, id string) *httptest.ResponseRecorder {
	t.Helper()
	req := withChiURLParam(withAuth(httptest.NewRequest(http.MethodGet, "/x", nil), testWorkTaskUserID, testTenantID), "id", id)
	rec := httptest.NewRecorder()
	routes.HandleGetFileDownloadURL(rec, req)
	return rec
}

func TestFileDownload_TaskFileIDFallsBackToWork(t *testing.T) {
	routes, work := newFileDownloadRoutes(t)

	rec := downloadFile(t, routes, downloadTaskFileID)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, map[string]any{"url": "https://files.test/task"}, decodeObject(t, rec))
	assert.Equal(t, []string{downloadTaskFileID}, work.asked)
}

func TestFileDownload_ChatFileUnchanged(t *testing.T) {
	routes, work := newFileDownloadRoutes(t)

	rec := downloadFile(t, routes, downloadChatFileID)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "https://files.test/chat", decodeObject(t, rec)["url"])
	assert.Empty(t, work.asked, "a chat file never reaches the work service")
}

func TestFileDownload_UnknownIDIsNotFound(t *testing.T) {
	routes, _ := newFileDownloadRoutes(t)

	rec := downloadFile(t, routes, "33333333-3333-4333-8333-333333333333")

	assert.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
}
