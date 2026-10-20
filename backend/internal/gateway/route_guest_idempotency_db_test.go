package gateway

// route_guest_idempotency_db_test.go belongs to
// harden-guest-chat-message-retry-duplicates (BACKLOG.yml). A guest has no JWT,
// so the Idempotency middleware cannot guard POST /guest/channels/{id}/messages;
// a retry after a lost response posted the same support message twice. The tests
// run against the real Postgres-backed idempotency.Repository (RLS active), so
// they also prove the session tenant reaches the tenant-scoped store.

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/chat/guest"
	"github.com/kmuhub/kmuhub/internal/idempotency"
	"github.com/kmuhub/kmuhub/internal/testutil"
	chatv1 "github.com/kmuhub/kmuhub/proto/chat/v1"
)

type countingChatServer struct {
	chatv1.UnimplementedChatServiceServer
	sent atomic.Int32
}

func (s *countingChatServer) SendMessage(_ context.Context, in *chatv1.SendMessageRequest) (*chatv1.SendMessageResponse, error) {
	s.sent.Add(1)
	return &chatv1.SendMessageResponse{Message: &chatv1.MessageInfo{Id: uuid.New().String(), Content: in.Content}}, nil
}

type guestIdempotencyFixture struct {
	srv    *countingChatServer
	repo   idempotency.Repository
	routes *GuestRoutes
	router chi.Router
	tenant uuid.UUID
}

func newGuestIdempotencyFixture(t *testing.T) *guestIdempotencyFixture {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Guest Idempotency")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })

	srv := &countingChatServer{}
	gs := grpc.NewServer()
	chatv1.RegisterChatServiceServer(gs, srv)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { _ = gs.Serve(l) }()
	t.Cleanup(gs.Stop)

	reg := NewServiceRegistry(nil)
	reg.Register("chat", l.Addr().String())
	t.Cleanup(reg.Close)

	repo := idempotency.NewPostgresRepository(pool)
	routes := NewGuestRoutes(guest.NewService(nil), reg)
	routes.SetIdempotency(repo)

	r := chi.NewRouter()
	r.Post("/api/v1/guest/channels/{channelID}/messages", routes.HandleSendMessage)
	return &guestIdempotencyFixture{srv: srv, repo: repo, routes: routes, router: r, tenant: tenantID}
}

func (f *guestIdempotencyFixture) newSession() *guest.GuestSession {
	return &guest.GuestSession{ID: uuid.New(), TenantID: f.tenant, ChannelID: uuid.New()}
}

// send posts one message as the guest session. Complete runs in a goroutine
// after the response, so an immediate retry can still see the key in flight
// (409): it is re-sent until the key has settled.
func (f *guestIdempotencyFixture) send(t *testing.T, s *guest.GuestSession, key, content string) *httptest.ResponseRecorder {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		req := httptest.NewRequest("POST", "/api/v1/guest/channels/"+s.ChannelID.String()+"/messages",
			jsonBody(t, map[string]interface{}{"content": content}))
		req = req.WithContext(context.WithValue(req.Context(), guestSessionContextKey{}, s))
		if key != "" {
			req.Header.Set("Idempotency-Key", key)
		}
		rec := httptest.NewRecorder()
		f.router.ServeHTTP(rec, req)
		if rec.Code != http.StatusConflict || time.Now().After(deadline) {
			return rec
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func messageID(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Message struct {
			ID string `json:"id"`
		} `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return body.Message.ID
}

func TestGuestSendMessage_RetryWithSameKeyPostsOnce(t *testing.T) {
	f := newGuestIdempotencyFixture(t)
	s := f.newSession()
	key := "guest-" + uuid.NewString()

	first := f.send(t, s, key, "hallo")
	assertStatus(t, first, http.StatusCreated)
	second := f.send(t, s, key, "hallo")
	assertStatus(t, second, http.StatusCreated)

	if got := f.srv.sent.Load(); got != 1 {
		t.Fatalf("chat SendMessage calls = %d, want 1", got)
	}
	if second.Header().Get("Idempotency-Replayed") != "true" {
		t.Error("retry was not served as a replay")
	}
	// The store keeps the body as JSONB, so only the content (not the bytes) matches.
	if a, b := messageID(t, first), messageID(t, second); a == "" || a != b {
		t.Errorf("replayed message id = %q, want the first response's %q", b, a)
	}
}

func TestGuestSendMessage_WithoutKeyKeepsPostingEveryTime(t *testing.T) {
	f := newGuestIdempotencyFixture(t)
	s := f.newSession()

	assertStatus(t, f.send(t, s, "", "hallo"), http.StatusCreated)
	assertStatus(t, f.send(t, s, "", "hallo"), http.StatusCreated)

	if got := f.srv.sent.Load(); got != 2 {
		t.Fatalf("chat SendMessage calls = %d, want 2 (no header, no dedup)", got)
	}
}

func TestGuestSendMessage_SameKeyDifferentContentIsRejected(t *testing.T) {
	f := newGuestIdempotencyFixture(t)
	s := f.newSession()
	key := "guest-" + uuid.NewString()

	assertStatus(t, f.send(t, s, key, "hallo"), http.StatusCreated)
	assertStatus(t, f.send(t, s, key, "etwas anderes"), http.StatusUnprocessableEntity)

	if got := f.srv.sent.Load(); got != 1 {
		t.Fatalf("chat SendMessage calls = %d, want 1", got)
	}
}

// Two guests of one tenant may pick the same header value (a counter, a short
// random string): their messages must both go through, not collide.
func TestGuestSendMessage_OtherSessionSameKeyDoesNotCollide(t *testing.T) {
	f := newGuestIdempotencyFixture(t)
	a, b := f.newSession(), f.newSession()
	key := "guest-" + uuid.NewString()

	assertStatus(t, f.send(t, a, key, "hallo"), http.StatusCreated)
	assertStatus(t, f.send(t, b, key, "hallo"), http.StatusCreated)

	if got := f.srv.sent.Load(); got != 2 {
		t.Fatalf("chat SendMessage calls = %d, want 2", got)
	}
}
