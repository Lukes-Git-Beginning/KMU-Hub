package gateway

// WOPI PutFile (POST /wopi/files/{id}/contents) is how OnlyOffice saves an edited document.
// The adapter used to read the body and throw it away (`_ = body`), and the WOPI context carried
// neither tenant (open) nor author (save) across the gRPC hop. These tests drive the real WOPI
// handler through the real adapter and read what the document service receives.

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/document/wopi"
	"github.com/kmuhub/kmuhub/internal/testutil"
	documentv1 "github.com/kmuhub/kmuhub/proto/document/v1"
)

const (
	testWOPIFileID = "770e8400-e29b-41d4-a716-446655440000"
	testWOPIUser   = "66666666-6666-6666-6666-666666666666"
)

type wopiDocStub struct {
	documentv1.UnimplementedDocumentServiceServer
	mu    sync.Mutex
	saves []*documentv1.CreateFileVersionRequest
	md    []metadata.MD
}

// GetFile only knows the file inside testTenantID, like the tenant-scoped service.
func (s *wopiDocStub) GetFile(ctx context.Context, r *documentv1.GetFileRequest) (*documentv1.GetFileResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if t := md.Get("x-tenant-id"); len(t) == 0 || t[0] != testTenantID.String() {
		return nil, status.Error(codes.NotFound, "file not found")
	}
	return &documentv1.GetFileResponse{File: &documentv1.DocumentFile{
		Id: r.Id, Filename: "angebot.docx", StorageKey: "documents/live/angebot.docx", CurrentVersion: 1,
		MimeType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	}}, nil
}

func (s *wopiDocStub) CreateFileVersion(ctx context.Context, r *documentv1.CreateFileVersionRequest) (*documentv1.CreateFileVersionResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saves = append(s.saves, r)
	s.md = append(s.md, md)
	return &documentv1.CreateFileVersionResponse{Version: &documentv1.DocumentFileVersion{
		Id: uuid.NewString(), FileId: r.FileId, VersionNumber: 2,
	}}, nil
}

type wopiStoreStub struct{ downloads []string }

func (s *wopiStoreStub) Upload(context.Context, string, io.Reader, int64, string) error { return nil }
func (s *wopiStoreStub) Download(_ context.Context, key string) (io.ReadCloser, error) {
	s.downloads = append(s.downloads, key)
	return io.NopCloser(strings.NewReader("gespeicherter Inhalt")), nil
}
func (s *wopiStoreStub) Delete(context.Context, string) error { return nil }
func (s *wopiStoreStub) GetPresignedURL(context.Context, string, time.Duration) (string, error) {
	return "", nil
}
func (s *wopiStoreStub) GetPresignedUploadURL(context.Context, string, time.Duration) (string, error) {
	return "", nil
}

func newWOPITestHandler(t *testing.T) (*wopi.Handler, *wopi.TokenService, *wopiDocStub, *wopiStoreStub) {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := grpc.NewServer(grpc.MaxRecvMsgSize(60 << 20))
	stub := &wopiDocStub{}
	documentv1.RegisterDocumentServiceServer(srv, stub)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("document", lis.Addr().String())
	t.Cleanup(reg.Close)

	tokens := wopi.NewTokenService("wopi-adapter-test-secret-at-least-32-chars")
	store := &wopiStoreStub{}
	return wopi.NewHandler(tokens, wopi.NewLockService(pool), NewWOPIFileAdapter(reg), store), tokens, stub, store
}

func wopiToken(t *testing.T, tokens *wopi.TokenService, tenant string) string {
	t.Helper()
	tok, _, err := tokens.Generate(testWOPIFileID, testWOPIUser, "Tester", tenant, true)
	require.NoError(t, err)
	return tok
}

func TestWOPIPutFile_SendsTheSavedBodyAndTheTokensTenantAndUser(t *testing.T) {
	h, tokens, stub, _ := newWOPITestHandler(t)
	body := []byte("PK\x03\x04 bearbeitetes Angebot")

	req := httptest.NewRequest(http.MethodPost,
		"/wopi/files/"+testWOPIFileID+"/contents?access_token="+wopiToken(t, tokens, testTenantID.String()),
		bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.PutFile(rec, req, testWOPIFileID)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Len(t, stub.saves, 1)
	assert.Equal(t, body, stub.saves[0].Content, "the editor's bytes reach the document service")
	assert.Equal(t, testWOPIFileID, stub.saves[0].FileId)
	assert.Equal(t, "Auto-saved via WOPI", stub.saves[0].VersionLabel)
	assert.Equal(t, []string{testTenantID.String()}, stub.md[0].Get("x-tenant-id"))
	assert.Equal(t, []string{testWOPIUser}, stub.md[0].Get("x-user-id"), "the author comes from the WOPI token")
}

func TestWOPIPutFile_AnEmptySaveIsStillContentNotASnapshot(t *testing.T) {
	h, tokens, stub, _ := newWOPITestHandler(t)

	req := httptest.NewRequest(http.MethodPost,
		"/wopi/files/"+testWOPIFileID+"/contents?access_token="+wopiToken(t, tokens, testTenantID.String()), nil)
	rec := httptest.NewRecorder()
	h.PutFile(rec, req, testWOPIFileID)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Len(t, stub.saves, 1)
	assert.NotNil(t, stub.saves[0].Content, "an empty body must not fall back to freezing the old content")
}

func TestWOPIPutFile_ForeignTenantTokenIsNotFoundAndSavesNothing(t *testing.T) {
	h, tokens, stub, _ := newWOPITestHandler(t)

	req := httptest.NewRequest(http.MethodPost,
		"/wopi/files/"+testWOPIFileID+"/contents?access_token="+wopiToken(t, tokens, uuid.NewString()),
		strings.NewReader("fremd"))
	rec := httptest.NewRecorder()
	h.PutFile(rec, req, testWOPIFileID)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Empty(t, stub.saves)
}

func TestWOPIPutFile_OverTheUploadLimitIs413(t *testing.T) {
	h, tokens, stub, _ := newWOPITestHandler(t)

	req := httptest.NewRequest(http.MethodPost,
		"/wopi/files/"+testWOPIFileID+"/contents?access_token="+wopiToken(t, tokens, testTenantID.String()),
		bytes.NewReader(make([]byte, maxDocumentUploadBytes+1)))
	rec := httptest.NewRecorder()
	h.PutFile(rec, req, testWOPIFileID)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	assert.Empty(t, stub.saves)
}

func TestWOPIGetFile_CarriesTheTokensTenantToTheDocumentService(t *testing.T) {
	h, tokens, _, store := newWOPITestHandler(t)

	req := httptest.NewRequest(http.MethodGet,
		"/wopi/files/"+testWOPIFileID+"/contents?access_token="+wopiToken(t, tokens, testTenantID.String()), nil)
	rec := httptest.NewRecorder()
	h.GetFile(rec, req, testWOPIFileID)

	require.Equal(t, http.StatusOK, rec.Code, "without the tenant the document service refused the lookup and the editor could not open the file")
	assert.Equal(t, "gespeicherter Inhalt", rec.Body.String())
	assert.Equal(t, []string{"documents/live/angebot.docx"}, store.downloads)

	info := httptest.NewRecorder()
	h.CheckFileInfo(info, httptest.NewRequest(http.MethodGet,
		"/wopi/files/"+testWOPIFileID+"?access_token="+wopiToken(t, tokens, testTenantID.String()), nil), testWOPIFileID)
	assert.Equal(t, http.StatusOK, info.Code)
}
