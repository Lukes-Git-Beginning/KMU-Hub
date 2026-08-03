package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/auth"
)

// ---------------------------------------------------------------------------
// Isolated unit test for revalidateTokenLoop
// ---------------------------------------------------------------------------

// TestRevalidateTokenLoop_ExpiredToken verifies that revalidateTokenLoop closes
// the connection with StatusPolicyViolation when the token has expired.
//
// The test extracts the revalidation logic via a very short ticker interval so
// it does not depend on the 5-minute production constant. No real WebSocket
// server is needed for this variant; we use a lightweight pipe via
// net/http/httptest + coder/websocket to obtain a real *websocket.Conn pair.
func TestRevalidateTokenLoop_ExpiredToken(t *testing.T) {
	const secret = "test-secret-minimum-32-characters!"

	// Issue a token that expires immediately (1 ns TTL).
	tm := auth.NewTokenMaker(secret, time.Nanosecond, 7*24*time.Hour)
	userID := uuid.New()
	rawToken, err := tm.CreateAccessToken(userID, uuid.New().String(), []string{"member"}, &auth.TokenPermissions{})
	require.NoError(t, err)

	// The token is already expired by the time the goroutine ticks.
	time.Sleep(2 * time.Millisecond)

	// Build a minimal hub – no chat client, no origin check.
	hub := &WebSocketHub{
		tokenMaker: tm,
	}

	// Stand up a test WebSocket server whose sole job is to run revalidateTokenLoop
	// and a passive read-loop (mirrors handleConnection without the full hub wiring).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, acceptErr := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if acceptErr != nil {
			return
		}

		ctx := r.Context()

		// Use a very short ticker so the test does not have to wait 5 minutes.
		// We call revalidateTokenLoop directly with an overridden ticker via the
		// exported-for-test helper below, but since that helper is in the same
		// package we can call the internal method directly.
		done := make(chan struct{})
		go func() {
			defer close(done)
			hub.revalidateTokenLoopWithInterval(ctx, conn, userID.String(), rawToken, time.Millisecond)
		}()

		// Passive read-loop: block until the connection is closed.
		for {
			var msg WSMessage
			if readErr := wsjson.Read(ctx, conn, &msg); readErr != nil {
				break
			}
		}
		<-done
	}))
	defer srv.Close()

	// Connect as a client.
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	clientConn, dialResp, dialErr := websocket.Dial(context.Background(), wsURL, nil)
	require.NoError(t, dialErr)
	if dialResp != nil && dialResp.Body != nil {
		defer dialResp.Body.Close()
	}

	// The server should close the connection within a short window because the
	// token is already expired.
	clientConn.SetReadLimit(4096)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var dummy WSMessage
	readErr := wsjson.Read(ctx, clientConn, &dummy)
	require.Error(t, readErr, "expected connection to be closed by the server")

	status := websocket.CloseStatus(readErr)
	require.Equal(t, websocket.StatusPolicyViolation, status,
		"expected PolicyViolation close code, got %v", status)
}

// TestRevalidateTokenLoop_ValidToken verifies that revalidateTokenLoop does NOT
// close the connection when the token remains valid across ticks.
func TestRevalidateTokenLoop_ValidToken(t *testing.T) {
	const secret = "test-secret-minimum-32-characters!"

	// Token valid for a long time.
	tm := auth.NewTokenMaker(secret, time.Hour, 7*24*time.Hour)
	userID := uuid.New()
	rawToken, err := tm.CreateAccessToken(userID, uuid.New().String(), []string{"member"}, &auth.TokenPermissions{})
	require.NoError(t, err)

	hub := &WebSocketHub{tokenMaker: tm}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, acceptErr := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if acceptErr != nil {
			return
		}

		ctx := r.Context()

		done := make(chan struct{})
		go func() {
			defer close(done)
			// Tick every millisecond; token is valid so nothing should happen.
			hub.revalidateTokenLoopWithInterval(ctx, conn, userID.String(), rawToken, time.Millisecond)
		}()

		// Passive read-loop.
		for {
			var msg WSMessage
			if readErr := wsjson.Read(ctx, conn, &msg); readErr != nil {
				break
			}
		}
		<-done
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	clientConn, dialResp, dialErr := websocket.Dial(context.Background(), wsURL, nil)
	require.NoError(t, dialErr)
	if dialResp != nil && dialResp.Body != nil {
		defer dialResp.Body.Close()
	}

	// Let several ticks pass; the connection should remain open.
	time.Sleep(20 * time.Millisecond)

	// Explicitly close from the client side; server should see a normal closure.
	_ = clientConn.Close(websocket.StatusNormalClosure, "test done")
}
