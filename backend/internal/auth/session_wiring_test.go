package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/clientctx"
	"github.com/kmuhub/kmuhub/internal/models"
)

// These cover the decisions the DB test cannot isolate: which repository call
// each token path makes, and that a rotation whose session predates session
// tracking still ends up with one.

func TestCreateTokenPair_LoginRecordsSessionWithClientInfo(t *testing.T) {
	svc, repo := newTestService()
	user := createTestUser(repo, "wiring-login@test.local", "Test-Pw-1234!", true)

	ctx := clientctx.With(context.Background(), clientctx.Info{
		IP:        "198.51.100.4",
		UserAgent: "Mozilla/5.0 (Macintosh) Chrome/120",
	})

	if _, err := svc.Login(ctx, user.Email, "Test-Pw-1234!"); err != nil {
		t.Fatalf("login: %v", err)
	}

	if len(repo.sessions) != 1 {
		t.Fatalf("expected 1 session after login, got %d", len(repo.sessions))
	}
	sess := repo.sessions[0]
	if sess.IPAddress != "198.51.100.4" {
		t.Errorf("ip address = %q, want the caller's", sess.IPAddress)
	}
	if sess.DeviceType != "browser" || sess.DeviceName != "Google Chrome (macOS)" {
		t.Errorf("device = %q/%q, want the parsed user agent", sess.DeviceName, sess.DeviceType)
	}
	if sess.TenantID != user.TenantID {
		t.Errorf("tenant = %s, want the user's %s", sess.TenantID, user.TenantID)
	}
	if len(repo.prunedFor) != 1 || repo.prunedFor[0] != user.ID {
		t.Errorf("stale sessions were not pruned for the signing-in user: %v", repo.prunedFor)
	}
}

func TestCreateTokenPair_RefreshRotatesInsteadOfAdding(t *testing.T) {
	svc, repo := newTestService()
	user := createTestUser(repo, "wiring-refresh@test.local", "Test-Pw-1234!", true)

	result, err := svc.Login(context.Background(), user.Email, "Test-Pw-1234!")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	oldTokenID := *repo.sessions[0].RefreshTokenID
	sessionID := repo.sessions[0].ID

	if _, err := svc.RefreshToken(context.Background(), result.RefreshToken); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	if len(repo.sessions) != 1 {
		t.Fatalf("refresh must not add a device entry, got %d sessions", len(repo.sessions))
	}
	if repo.sessions[0].ID != sessionID {
		t.Errorf("session id changed on refresh: %s → %s", sessionID, repo.sessions[0].ID)
	}
	if *repo.sessions[0].RefreshTokenID == oldTokenID {
		t.Error("session still points at the revoked token — the next refresh would lose it")
	}
	if len(repo.rotatedFrom) != 1 || repo.rotatedFrom[0] != oldTokenID {
		t.Errorf("rotation asked about %v, want the token being replaced %s", repo.rotatedFrom, oldTokenID)
	}
}

// A refresh of a token issued before sessions were recorded finds nothing to
// rotate. It must create a session rather than leave the device invisible
// forever — this is the state every existing production token is in on the
// day this ships.
func TestCreateTokenPair_RefreshWithoutSessionCreatesOne(t *testing.T) {
	svc, repo := newTestService()
	user := createTestUser(repo, "wiring-legacy@test.local", "Test-Pw-1234!", true)

	result, err := svc.Login(context.Background(), user.Email, "Test-Pw-1234!")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	repo.sessions = nil // pretend the login predates session tracking

	if _, err := svc.RefreshToken(context.Background(), result.RefreshToken); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	if len(repo.sessions) != 1 {
		t.Fatalf("expected a session to be created on rotation fallback, got %d", len(repo.sessions))
	}
}

// A failing session write must not cost the user their login — the device list
// is a convenience view, not an authentication requirement.
func TestCreateTokenPair_SessionFailureDoesNotBlockLogin(t *testing.T) {
	svc, repo := newTestService()
	user := createTestUser(repo, "wiring-sessfail@test.local", "Test-Pw-1234!", true)
	repo.sessionWriteFails = true

	result, err := svc.Login(context.Background(), user.Email, "Test-Pw-1234!")
	if err != nil {
		t.Fatalf("login must survive a failing session write: %v", err)
	}
	if result.AccessToken == "" {
		t.Error("no access token issued")
	}
	if len(repo.sessions) != 0 {
		t.Errorf("expected no session to have been stored, got %d", len(repo.sessions))
	}
}

func TestLogout_RemovesSession(t *testing.T) {
	svc, repo := newTestService()
	user := createTestUser(repo, "wiring-logout@test.local", "Test-Pw-1234!", true)

	result, err := svc.Login(context.Background(), user.Email, "Test-Pw-1234!")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if len(repo.sessions) != 1 {
		t.Fatalf("setup: expected 1 session, got %d", len(repo.sessions))
	}

	if err := svc.Logout(context.Background(), result.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if len(repo.sessions) != 0 {
		t.Error("a signed-out device must not keep showing as active")
	}
}

func TestTerminateSession_RejectsForeignOwner(t *testing.T) {
	svc, repo := newTestService()
	owner := createTestUser(repo, "wiring-owner@test.local", "Test-Pw-1234!", true)
	stranger := createTestUser(repo, "wiring-stranger@test.local", "Test-Pw-1234!", true)

	tokenID := uuid.New()
	session := &models.UserSession{
		ID:             uuid.New(),
		TenantID:       owner.TenantID,
		UserID:         owner.ID,
		RefreshTokenID: &tokenID,
	}
	repo.sessions = append(repo.sessions, session)

	err := svc.TerminateSession(context.Background(), session.ID, stranger.ID)
	if err != ErrSessionNotFound {
		t.Fatalf("terminating a foreign session: got %v, want ErrSessionNotFound", err)
	}
	if len(repo.sessions) != 1 {
		t.Error("the owner's session must survive a foreign termination attempt")
	}

	if err := svc.TerminateSession(context.Background(), session.ID, owner.ID); err != nil {
		t.Fatalf("owner terminating own session: %v", err)
	}
	if len(repo.sessions) != 0 {
		t.Error("the owner's own termination must delete the session")
	}
}
