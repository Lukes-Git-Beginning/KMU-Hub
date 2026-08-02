package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/clientctx"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// The session lifecycle is only provable against the real table. A mock cannot
// show that ip_address survives the INET column (an empty string bound there
// fails the whole insert with SQLSTATE 22P02, which would turn every login
// without a resolvable peer address into a 500), that a refresh re-points the
// existing row instead of adding a second one, or that the row is actually
// gone after logout.
//
// Own tenant, never the shared testutil.TenantA/B — this file seeds users with
// fixed emails and runs beside the rest of the package.
var sessionLifecycleTenant = uuid.MustParse("5e551011-0000-4000-8000-000000000001")

func sessionSetup(t *testing.T) (*pgxpool.Pool, *auth.Service) {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	testutil.EnsureTenant(t, pool, sessionLifecycleTenant, "SessionLifecycleTenant")

	svc := auth.NewService(
		auth.NewPostgresRepository(pool),
		auth.NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour),
	)
	return pool, svc
}

// sessionUser creates a user with a known password so Login can be driven end
// to end rather than the session being written by the test itself.
func sessionUser(t *testing.T, pool *pgxpool.Pool, svc *auth.Service, email string) uuid.UUID {
	t.Helper()

	user, _, err := svc.Register(
		testutil.WithSystemCtx(context.Background()),
		email, "Sess-Lifecycle-Pw1!", "Sess", "User",
	)
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", user.ID) })

	sysCtx := testutil.WithSystemCtx(context.Background())

	// Register pins every user to models.DefaultTenantID; move it to this
	// test's own tenant so the session rows land there too (CreateSession
	// copies user.TenantID).
	_, err = pool.Exec(sysCtx,
		`UPDATE users SET tenant_id = $2 WHERE id = $1`, user.ID, sessionLifecycleTenant)
	require.NoError(t, err)

	// Registering signs the user in, so it leaves a session of its own behind
	// — under the old tenant, and counting towards the assertions below.
	// Clear it so each test starts from "no devices".
	_, err = pool.Exec(sysCtx, `DELETE FROM user_sessions WHERE user_id = $1`, user.ID)
	require.NoError(t, err)

	return user.ID
}

func countSessions(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) int {
	t.Helper()
	var n int
	err := pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT COUNT(*) FROM user_sessions WHERE user_id = $1`, userID).Scan(&n)
	require.NoError(t, err)
	return n
}

// TestSessionLifecycle_LoginRefreshLogout walks the whole path: a login writes
// one session carrying the caller's device, a refresh moves that same row onto
// the new token, and a logout removes it.
func TestSessionLifecycle_LoginRefreshLogout(t *testing.T) {
	pool, svc := sessionSetup(t)
	userID := sessionUser(t, pool, svc, "sess-lifecycle@test.local")

	ctx := clientctx.With(context.Background(), clientctx.Info{
		IP:        "203.0.113.7",
		UserAgent: "Mozilla/5.0 (Windows NT 10.0) KMUHub-Electron/1.0",
	})

	// --- Login writes exactly one session with the device metadata ---
	result, err := svc.Login(ctx, "sess-lifecycle@test.local", "Sess-Lifecycle-Pw1!")
	require.NoError(t, err)
	require.NotEmpty(t, result.RefreshToken)

	sessions, err := svc.ListSessions(testutil.WithSystemCtx(context.Background()), userID)
	require.NoError(t, err)
	require.Len(t, sessions, 1, "login must create exactly one session")

	first := sessions[0]
	assert.Equal(t, "203.0.113.7", first.IPAddress, "client IP must survive the INET column")
	assert.Equal(t, "desktop", first.DeviceType, "electron user agent is a desktop device")
	assert.NotNil(t, first.RefreshTokenID, "session must point at the token it was issued with")

	// --- Refresh rotates the same row, it does not add a second one ---
	newTokens, err := svc.RefreshToken(ctx, result.RefreshToken)
	require.NoError(t, err)

	sessions, err = svc.ListSessions(testutil.WithSystemCtx(context.Background()), userID)
	require.NoError(t, err)
	require.Len(t, sessions, 1, "refresh must rotate the existing session, not create a new device entry")
	assert.Equal(t, first.ID, sessions[0].ID, "the session id must stay stable across a refresh")
	require.NotNil(t, sessions[0].RefreshTokenID)
	assert.NotEqual(t, *first.RefreshTokenID, *sessions[0].RefreshTokenID,
		"the session must point at the rotated token, otherwise the next refresh loses it")

	// --- Logout removes the device entry ---
	require.NoError(t, svc.Logout(ctx, newTokens.RefreshToken))
	assert.Equal(t, 0, countSessions(t, pool, userID),
		"a signed-out device must not keep showing as active")
}

// TestSessionLifecycle_LoginWithoutClientInfo covers the case the INET column
// punishes: no proxy header, no peer address, empty user agent. The login must
// still succeed and still produce a session row.
func TestSessionLifecycle_LoginWithoutClientInfo(t *testing.T) {
	pool, svc := sessionSetup(t)
	userID := sessionUser(t, pool, svc, "sess-no-client-info@test.local")

	result, err := svc.Login(context.Background(), "sess-no-client-info@test.local", "Sess-Lifecycle-Pw1!")
	require.NoError(t, err, "an unresolvable client address must not cost the user their login")
	require.NotEmpty(t, result.RefreshToken)

	sessions, err := svc.ListSessions(testutil.WithSystemCtx(context.Background()), userID)
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Empty(t, sessions[0].IPAddress, "an unknown address reads back as empty, not as a bogus value")

	require.NoError(t, svc.Logout(context.Background(), result.RefreshToken))
}

// TestSessionLifecycle_TerminateForeignSessionIsRejected is the security case:
// the session id travels in the URL of an authenticated route, so a signed-in
// user must not be able to sign out anybody else with a guessed id.
func TestSessionLifecycle_TerminateForeignSessionIsRejected(t *testing.T) {
	pool, svc := sessionSetup(t)
	victimID := sessionUser(t, pool, svc, "sess-victim@test.local")
	attackerID := sessionUser(t, pool, svc, "sess-attacker@test.local")

	_, err := svc.Login(context.Background(), "sess-victim@test.local", "Sess-Lifecycle-Pw1!")
	require.NoError(t, err)

	sysCtx := testutil.WithSystemCtx(context.Background())
	victimSessions, err := svc.ListSessions(sysCtx, victimID)
	require.NoError(t, err)
	require.Len(t, victimSessions, 1)

	err = svc.TerminateSession(sysCtx, victimSessions[0].ID, attackerID)
	require.ErrorIs(t, err, auth.ErrSessionNotFound,
		"terminating a session of another user must fail, and must not disclose that it exists")
	assert.Equal(t, 1, countSessions(t, pool, victimID), "the victim's session must survive")

	// The owner can still terminate it.
	require.NoError(t, svc.TerminateSession(sysCtx, victimSessions[0].ID, victimID))
	assert.Equal(t, 0, countSessions(t, pool, victimID))
}

// TestSessionLifecycle_TerminateRevokesRefreshToken proves the part that makes
// remote sign-out more than a display: the terminated device's refresh token
// must stop working.
func TestSessionLifecycle_TerminateRevokesRefreshToken(t *testing.T) {
	pool, svc := sessionSetup(t)
	userID := sessionUser(t, pool, svc, "sess-revoke@test.local")

	result, err := svc.Login(context.Background(), "sess-revoke@test.local", "Sess-Lifecycle-Pw1!")
	require.NoError(t, err)

	sysCtx := testutil.WithSystemCtx(context.Background())
	sessions, err := svc.ListSessions(sysCtx, userID)
	require.NoError(t, err)
	require.Len(t, sessions, 1)

	require.NoError(t, svc.TerminateSession(sysCtx, sessions[0].ID, userID))

	_, err = svc.RefreshToken(context.Background(), result.RefreshToken)
	require.Error(t, err, "a terminated device must not be able to refresh its way back in")

	assert.Equal(t, 0, countSessions(t, pool, userID))
}
