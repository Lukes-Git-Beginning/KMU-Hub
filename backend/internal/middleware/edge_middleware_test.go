package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/metrics"
)

// --- CORS ---

func corsServe(h http.Handler, method, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/v1/contacts", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCORS_AllowlistedOriginGetsCredentialedHeaders(t *testing.T) {
	called := false
	h := CORS([]string{" https://app.example.com ", "https://other.example.com"})(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { called = true }))

	rec := corsServe(h, http.MethodGet, "https://app.example.com")

	assert.True(t, called, "a non-preflight request reaches the handler")
	assert.Equal(t, "https://app.example.com", rec.Header().Get("Access-Control-Allow-Origin"),
		"the allowlist entry is trimmed and the origin echoed, never '*'")
	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
	assert.Contains(t, rec.Header().Get("Access-Control-Allow-Headers"), "Idempotency-Key")
	assert.Contains(t, rec.Header().Get("Access-Control-Allow-Methods"), "PATCH")
	assert.Equal(t, "86400", rec.Header().Get("Access-Control-Max-Age"))
}

func TestCORS_ForeignOrEmptyOriginGetsNoCORSHeaders(t *testing.T) {
	h := CORS([]string{"https://app.example.com"})(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))

	for _, origin := range []string{"https://evil.example.com", "https://app.example.com.evil.io", "HTTPS://APP.EXAMPLE.COM", ""} {
		rec := corsServe(h, http.MethodGet, origin)
		assert.Equal(t, http.StatusOK, rec.Code, "origin %q", origin)
		assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"), "origin %q must not be echoed", origin)
		assert.Empty(t, rec.Header().Get("Access-Control-Allow-Credentials"), "origin %q", origin)
	}
}

func TestCORS_PreflightAnswers204WithoutReachingHandler(t *testing.T) {
	called := false
	h := CORS([]string{"https://app.example.com"})(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { called = true }))

	rec := corsServe(h, http.MethodOptions, "https://app.example.com")
	assert.False(t, called)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "https://app.example.com", rec.Header().Get("Access-Control-Allow-Origin"))

	// A preflight from a foreign origin is also answered 204 but carries no grant, so the
	// browser blocks the real request.
	rec = corsServe(h, http.MethodOptions, "https://evil.example.com")
	assert.False(t, called)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

// --- gRPC panic recovery ---

func TestRecoveryUnaryInterceptor_TurnsPanicIntoInternal(t *testing.T) {
	ic := RecoveryUnaryInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: "/crm.v1.CRM/GetContact"}

	resp, err := ic(context.Background(), "req", info, func(context.Context, any) (any, error) {
		var req *grpc.UnaryServerInfo
		_ = req.FullMethod // nil-pointer dereference, the realistic handler panic
		return "unreachable", nil
	})
	assert.Nil(t, resp)
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.NotContains(t, err.Error(), "nil pointer", "the panic value must not leak to the caller")

	resp, err = ic(context.Background(), "req", info, func(_ context.Context, req any) (any, error) {
		return req, nil
	})
	require.NoError(t, err)
	assert.Equal(t, "req", resp)

	wantErr := status.Error(codes.NotFound, "nope")
	_, err = ic(context.Background(), "req", info, func(context.Context, any) (any, error) { return nil, wantErr })
	assert.Equal(t, wantErr, err, "handler errors pass through untouched")
}

func TestRecoveryStreamInterceptor_TurnsPanicIntoInternal(t *testing.T) {
	ic := RecoveryStreamInterceptor()
	info := &grpc.StreamServerInfo{FullMethod: "/chat.v1.Chat/Subscribe"}

	err := ic(nil, nil, info, func(any, grpc.ServerStream) error { panic("stream boom") })
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.NotContains(t, err.Error(), "stream boom")

	err = ic(nil, nil, info, func(any, grpc.ServerStream) error { return nil })
	assert.NoError(t, err)
}

// --- Request ID ---

func TestRequestID_GeneratesOrEchoesHeader(t *testing.T) {
	var seen string
	h := RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = GetRequestID(r.Context())
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	_, err := uuid.Parse(seen)
	require.NoError(t, err, "without a header a UUID is generated")
	assert.Equal(t, seen, rec.Header().Get("X-Request-ID"))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "trace-42")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, "trace-42", seen)
	assert.Equal(t, "trace-42", rec.Header().Get("X-Request-ID"))
}

func TestGetRequestID_EmptyWithoutMiddleware(t *testing.T) {
	assert.Equal(t, "", GetRequestID(context.Background()))
	assert.Equal(t, "", GetRequestID(context.WithValue(context.Background(), RequestIDKey, 42)),
		"a non-string value is not mistaken for an id")
}

// --- Logging response writer ---

// plainWriter implements only http.ResponseWriter, so neither Hijacker nor Flusher.
type plainWriter struct {
	header http.Header
	code   int
}

func (p *plainWriter) Header() http.Header         { return p.header }
func (p *plainWriter) Write(b []byte) (int, error) { return len(b), nil }
func (p *plainWriter) WriteHeader(code int)        { p.code = code }

func TestLoggingResponseWriter_RecordsStatusAndForwards(t *testing.T) {
	var inner *responseWriter
	h := Logging(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		inner, _ = w.(*responseWriter)
		w.WriteHeader(http.StatusTeapot)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	require.NotNil(t, inner, "Logging wraps the writer")
	assert.Equal(t, http.StatusTeapot, inner.statusCode)
	assert.Equal(t, http.StatusTeapot, rec.Code)

	// No explicit WriteHeader: the implicit 200 is logged.
	h = Logging(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		inner, _ = w.(*responseWriter)
		_, _ = w.Write([]byte("ok"))
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
	assert.Equal(t, http.StatusOK, inner.statusCode)
}

func TestLoggingResponseWriter_FlushAndHijackNeedUnderlyingSupport(t *testing.T) {
	// Flush reaches a flushing writer ...
	rec := httptest.NewRecorder()
	(&responseWriter{ResponseWriter: rec}).Flush()
	assert.True(t, rec.Flushed)

	// ... and is a harmless no-op on a writer that cannot flush.
	assert.NotPanics(t, func() { (&responseWriter{ResponseWriter: &plainWriter{header: http.Header{}}}).Flush() })

	// Hijack on a writer without the capability is an error, not a nil-pointer panic: the
	// WebSocket upgrade must fail cleanly.
	conn, brw, err := (&responseWriter{ResponseWriter: &plainWriter{header: http.Header{}}}).Hijack()
	assert.Error(t, err)
	assert.Nil(t, conn)
	assert.Nil(t, brw)

	// httptest.ResponseRecorder is no Hijacker either.
	_, _, err = (&responseWriter{ResponseWriter: httptest.NewRecorder()}).Hijack()
	assert.Error(t, err)
}

// metricValue scrapes the registry's /metrics text and returns the value of one series;
// the prometheus testutil package would pull in a module this repo does not vendor.
func metricValue(t *testing.T, reg *metrics.Registry, series string) float64 {
	t.Helper()
	rec := httptest.NewRecorder()
	reg.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	for _, line := range strings.Split(rec.Body.String(), "\n") {
		if strings.HasPrefix(line, series+" ") {
			v, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimPrefix(line, series)), 64)
			require.NoError(t, err)
			return v
		}
	}
	return 0
}

// --- Metrics ---

func TestMetrics_CountsByRoutePatternNotRawPath(t *testing.T) {
	reg := metrics.NewRegistry()
	r := chi.NewRouter()
	r.Use(Metrics(reg))
	r.Get("/api/v1/contacts/{id}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	r.Post("/api/v1/contacts", func(w http.ResponseWriter, _ *http.Request) {
		// body without an explicit WriteHeader: implicit 200
		_, _ = w.Write([]byte("{}"))
	})

	for _, id := range []string{"a", "b", "c"} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/contacts/"+id, nil))
		require.Equal(t, http.StatusNoContent, rec.Code)
	}
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/v1/contacts", nil))
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/nowhere/123", nil))

	assert.Equal(t, 3.0, metricValue(t, reg, `http_requests_total{method="GET",path="/api/v1/contacts/{id}",status="204"}`),
		"three ids share one label value; a raw path label would explode cardinality")
	assert.Equal(t, 1.0, metricValue(t, reg, `http_requests_total{method="POST",path="/api/v1/contacts",status="200"}`))
	assert.Equal(t, 1.0, metricValue(t, reg, `http_requests_total{method="GET",path="unknown",status="404"}`),
		"unmatched routes collapse into 'unknown'")
	assert.Equal(t, 0.0, metricValue(t, reg, `http_requests_in_flight{method="GET"}`),
		"the in-flight gauge returns to zero")
}

func TestMetrics_InFlightGaugeRisesWhileHandlerRuns(t *testing.T) {
	reg := metrics.NewRegistry()
	started, release := make(chan struct{}), make(chan struct{})
	h := Metrics(reg)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(started)
		<-release
	}))

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPut, "/slow", nil))
	}()
	<-started
	assert.Equal(t, 1.0, metricValue(t, reg, `http_requests_in_flight{method="PUT"}`))
	close(release)
	<-done
	assert.Equal(t, 0.0, metricValue(t, reg, `http_requests_in_flight{method="PUT"}`))
}

func TestStatusWriter_FirstStatusWins(t *testing.T) {
	sw := &statusWriter{ResponseWriter: httptest.NewRecorder(), statusCode: http.StatusOK}
	sw.WriteHeader(http.StatusCreated)
	sw.WriteHeader(http.StatusInternalServerError) // net/http ignores this; the metric must too
	assert.Equal(t, http.StatusCreated, sw.statusCode)

	// A Write after the status was set does not overwrite it.
	_, err := sw.Write([]byte("x"))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, sw.statusCode)

	// A Write before any WriteHeader counts as the implicit 200.
	sw = &statusWriter{ResponseWriter: httptest.NewRecorder(), statusCode: http.StatusOK}
	_, _ = sw.Write([]byte("x"))
	sw.WriteHeader(http.StatusTeapot) // too late, headers already went out as 200
	assert.Equal(t, http.StatusOK, sw.statusCode)
}

// --- Rate limiter ---

func TestClientIP_PrefersFirstForwardedHopForNonSecurityCallers(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.9:5555"
	assert.Equal(t, "10.0.0.9", ClientIP(req))

	req.RemoteAddr = "no-port"
	assert.Equal(t, "no-port", ClientIP(req))

	req.Header.Set("X-Forwarded-For", "203.0.113.5 , 10.0.0.1")
	assert.Equal(t, "203.0.113.5", ClientIP(req))
}

// The limiter keys on the authenticated user, so two users behind one NAT address do not
// share a bucket and one user switching IPs does not get a fresh one.
func TestRateLimiter_AuthenticatedUserKeyedByUserNotIP(t *testing.T) {
	rl := NewRateLimiter(nil, 1, false)
	h := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))

	serve := func(user, addr string) int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = addr
		req = req.WithContext(context.WithValue(req.Context(), UserIDKey, user))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	assert.Equal(t, http.StatusOK, serve("user-a", "10.0.0.1:1"))
	assert.Equal(t, http.StatusTooManyRequests, serve("user-a", "10.0.0.2:1"), "same user, new IP: same bucket")
	assert.Equal(t, http.StatusOK, serve("user-b", "10.0.0.1:1"), "other user, same IP: own bucket")
}

// A Redis outage must degrade to the in-memory limiter, not to "allow everything" and not
// to a 500.
func TestRateLimiter_RedisDown_FallsBackToInMemoryLimit(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1, DialTimeout: 200 * time.Millisecond})
	t.Cleanup(func() { _ = client.Close() })
	mr.Close()

	rl := NewRateLimiter(client, 2, false)
	h := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))

	assert.Equal(t, http.StatusOK, serveFrom(h, "192.0.2.1:1"))
	assert.Equal(t, http.StatusOK, serveFrom(h, "192.0.2.1:1"))
	assert.Equal(t, http.StatusTooManyRequests, serveFrom(h, "192.0.2.1:1"), "the fallback still enforces the limit")
}

// --- RBAC helpers ---

func TestIsAdmin(t *testing.T) {
	ctxWith := func(roles []string) context.Context {
		return context.WithValue(context.Background(), UserRolesKey, roles)
	}
	assert.True(t, IsAdmin(ctxWith([]string{"member", "admin"})))
	assert.False(t, IsAdmin(ctxWith([]string{"manager", "member"})))
	assert.False(t, IsAdmin(ctxWith([]string{"Admin", "administrator"})), "exact, case-sensitive role name")
	assert.False(t, IsAdmin(ctxWith(nil)))
	assert.False(t, IsAdmin(context.Background()))
}

func TestHoldsAnyPermission_MirrorsRequirePermissionAny(t *testing.T) {
	ctx := func(perms, denied []string) context.Context {
		c := context.WithValue(context.Background(), UserPermsKey, perms)
		return context.WithValue(c, UserDeniedKey, denied)
	}
	coarse := [2]string{"documents", "read"}
	fine := [2]string{"documents:file", "read"}

	assert.True(t, HoldsAnyPermission(ctx([]string{"documents:read"}, nil), coarse))
	assert.False(t, HoldsAnyPermission(ctx([]string{"documents:write"}, nil), coarse))
	assert.False(t, HoldsAnyPermission(context.Background(), coarse), "an anonymous context holds nothing")

	// Either of several keys is enough.
	assert.True(t, HoldsAnyPermission(ctx([]string{"documents:file:read"}, nil), coarse, fine))

	// A per-user deny on the fine key switches the coarse fallback off: the legacy
	// coarse permission must not smuggle back what the admin explicitly took away.
	assert.True(t, HoldsAnyPermission(ctx([]string{"documents:read"}, nil), coarse, fine))
	assert.False(t, HoldsAnyPermission(ctx([]string{"documents:read"}, []string{"documents:file:read"}), coarse, fine))
	// ... while an explicit grant of the fine key survives its own deny list entry for another key.
	assert.True(t, HoldsAnyPermission(ctx([]string{"documents:file:read"}, []string{"documents:file:write"}), coarse, fine))
}

// --- Audit classification ---

func TestShouldAudit_ClassifiesRequests(t *testing.T) {
	cases := []struct {
		method, path string
		want         bool
	}{
		{http.MethodPost, "/api/v1/auth/login", true},
		{http.MethodGet, "/api/v1/auth/me", false},
		{http.MethodPost, "/api/v1/contacts", true},
		{http.MethodPatch, "/api/v1/contacts/123", true},
		{http.MethodDelete, "/api/v1/deals/9", true},
		{http.MethodPut, "/api/v1/security/policies", true},
		{http.MethodPost, "/api/v1/files/upload", true},
		{http.MethodGet, "/api/v1/contacts", false},
		{http.MethodHead, "/api/v1/contacts", false},
		{http.MethodOptions, "/api/v1/contacts", false},
		{http.MethodPost, "/api/v1/weather", false},
		{http.MethodPost, "/health", false},
	}
	for _, c := range cases {
		r := httptest.NewRequest(c.method, c.path, nil)
		assert.Equal(t, c.want, shouldAudit(r), "%s %s", c.method, c.path)
	}
}

func TestDeriveActionFromMethod(t *testing.T) {
	cases := []struct{ method, path, want string }{
		{http.MethodPost, "/api/v1/auth/login", "user.login"},
		{http.MethodPost, "/api/v1/auth/register", "user.register"},
		{http.MethodPost, "/api/v1/auth/logout", "user.logout"},
		{http.MethodPost, "/api/v1/contacts", "resource.create"},
		{http.MethodPut, "/api/v1/contacts/1", "resource.update"},
		{http.MethodPatch, "/api/v1/contacts/1", "resource.update"},
		{http.MethodDelete, "/api/v1/contacts/1", "resource.delete"},
		{http.MethodGet, "/api/v1/contacts/1", "resource.access"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, deriveActionFromMethod(c.method, c.path), "%s %s", c.method, c.path)
	}
}

func TestAuditClientIP_HeaderPrecedence(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:99"
	r.Header.Set("X-Real-IP", "198.51.100.9")
	assert.Equal(t, "198.51.100.9", auditClientIP(r), "X-Real-IP beats RemoteAddr")

	r.Header.Set("X-Forwarded-For", "203.0.113.2")
	assert.Equal(t, "203.0.113.2", auditClientIP(r), "single-hop X-Forwarded-For is used as-is")
}

// --- Idempotency without a JWT / cleanup worker ---

func TestBeginIdempotentFor_NamespacesKeyPerPrincipal(t *testing.T) {
	repo := newMemIdempotencyRepo()
	tenant := uuid.New()
	guestA, guestB := uuid.New(), uuid.New()

	begin := func(principal uuid.UUID, key, fp string) (*httptest.ResponseRecorder, bool) {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/public/chat/messages", nil)
		if key != "" {
			r.Header.Set("Idempotency-Key", key)
		}
		rec := httptest.NewRecorder()
		w, done, ok := BeginIdempotentFor(repo, rec, r, tenant, principal, fp)
		if ok {
			okHandler(w, r)
			done()
		}
		return rec, ok
	}
	waitCompleted := func(key string) {
		require.True(t, assert.Eventually(t, func() bool {
			repo.mu.Lock()
			defer repo.mu.Unlock()
			rec := repo.records[key]
			return rec != nil && rec.CompletedAt != nil
		}, 2*time.Second, 5*time.Millisecond))
	}

	// No header: a no-op that stores nothing.
	_, ok := begin(guestA, "", "fp")
	require.True(t, ok)
	assert.Empty(t, repo.records)

	// nil repo: also a no-op.
	r := httptest.NewRequest(http.MethodPost, "/x", nil)
	r.Header.Set("Idempotency-Key", "k")
	_, _, ok = BeginIdempotentFor(nil, httptest.NewRecorder(), r, tenant, guestA, "fp")
	require.True(t, ok)

	rec, ok := begin(guestA, "same-key", "fp")
	require.True(t, ok)
	require.Equal(t, http.StatusCreated, rec.Code)
	waitCompleted(guestA.String() + ":same-key")

	// Guest B reuses the header value: its own namespace, so the handler runs instead of
	// replaying guest A's response.
	rec, ok = begin(guestB, "same-key", "fp")
	require.True(t, ok, "another principal must not collide on the header value")
	assert.Empty(t, rec.Header().Get("Idempotency-Replayed"))
	waitCompleted(guestB.String() + ":same-key")
	assert.Len(t, repo.records, 2)

	// Guest A repeating its own key replays.
	rec, ok = begin(guestA, "same-key", "fp")
	assert.False(t, ok)
	assert.Equal(t, "true", rec.Header().Get("Idempotency-Replayed"))
	assert.Equal(t, tenant, repo.records[guestA.String()+":same-key"].TenantID)
}

func TestIdempotencyCleanupWorker_StopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		IdempotencyCleanupWorker(ctx, newMemIdempotencyRepo())
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("cleanup worker did not stop after its context was cancelled")
	}
}
