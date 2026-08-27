package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// Helpers
//
// mockRepository (service_test.go) answers the 2FA reads with nil: no recovery
// codes, no policies. Both are what the login path branches on, so this file
// embeds it and fills exactly those two gaps. Everything else stays shared —
// a second full mock would drift away from the first one.
// ============================================================================

type authPathsRepo struct {
	*mockRepository

	// policies is keyed by role name; a role that is absent has no policy,
	// which is what the real repository returns as (nil, nil).
	policies map[string]*models.TwoFactorPolicy

	// lookupErr, when set, makes GetUserByEmail fail with something other than
	// ErrUserNotFound — the "broken lookup" branch of Login.
	lookupErr error
}

func (r *authPathsRepo) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	if r.lookupErr != nil {
		return nil, r.lookupErr
	}
	return r.mockRepository.GetUserByEmail(ctx, email)
}

// GetRecoveryCodes reads the codes Enable2FA stored on the embedded mock, so a
// test can walk the real enrolment flow instead of hand-building hashes.
func (r *authPathsRepo) GetRecoveryCodes(_ context.Context, userID uuid.UUID) ([]*models.RecoveryCode, error) {
	var out []*models.RecoveryCode
	for _, c := range r.recoveryCodes {
		if c.UserID == userID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *authPathsRepo) UseRecoveryCode(_ context.Context, id uuid.UUID) error {
	for _, c := range r.recoveryCodes {
		if c.ID == id {
			now := time.Now()
			c.UsedAt = &now
			return nil
		}
	}
	return ErrInvalidRecoveryCode
}

func (r *authPathsRepo) ReplaceRecoveryCodes(_ context.Context, userID uuid.UUID, codes []*models.RecoveryCode) error {
	kept := make([]*models.RecoveryCode, 0, len(r.recoveryCodes))
	for _, c := range r.recoveryCodes {
		if c.UserID != userID {
			kept = append(kept, c)
		}
	}
	r.recoveryCodes = append(kept, codes...)
	return nil
}

func (r *authPathsRepo) GetTwoFactorPolicy(_ context.Context, _ uuid.UUID, roleName string) (*models.TwoFactorPolicy, error) {
	return r.policies[roleName], nil
}

func (r *authPathsRepo) ListTwoFactorPolicies(_ context.Context, tenantID uuid.UUID) ([]*models.TwoFactorPolicy, error) {
	var out []*models.TwoFactorPolicy
	for _, p := range r.policies {
		if p.TenantID == tenantID {
			out = append(out, p)
		}
	}
	return out, nil
}

func newAuthPathsService() (*Service, *authPathsRepo) {
	repo := &authPathsRepo{
		mockRepository: newMockRepository(),
		policies:       make(map[string]*models.TwoFactorPolicy),
	}
	tm := NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	return NewService(repo, tm), repo
}

// enrol2FA walks the real enrolment flow (Setup2FA -> TOTP code -> Verify2FA)
// and returns the plaintext secret and recovery codes. Building the account
// state by hand would test the test, not the service.
func enrol2FA(t *testing.T, svc *Service, user *models.User) (secret string, recoveryCodes []string) {
	t.Helper()
	ctx := context.Background()

	setup, err := svc.Setup2FA(ctx, user.ID)
	require.NoError(t, err)
	require.NotEmpty(t, setup.Secret)
	require.NotEmpty(t, setup.QRCodePNG)

	code, err := totp.GenerateCode(setup.Secret, time.Now())
	require.NoError(t, err)

	codes, err := svc.Verify2FA(ctx, user.ID, code)
	require.NoError(t, err)
	require.Len(t, codes, recoveryCodeCount)

	return setup.Secret, codes
}

// signPendingToken mints a pending token with an arbitrary expiry and type so
// the rejection paths of ValidatePendingToken can be reached — CreatePendingToken
// only ever produces a valid one.
func signPendingToken(t *testing.T, tm *TokenMaker, userID uuid.UUID, tokenType string, expiresAt time.Time) string {
	t.Helper()
	claims := PendingClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			Issuer:    "kmuhub",
		},
		UserID:    userID.String(),
		TokenType: tokenType,
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(tm.secret)
	require.NoError(t, err)
	return signed
}

// ============================================================================
// Login failure paths
// ============================================================================

// TestLogin_FailurePaths_ShareOneAnswerExceptInactive pins which login failures
// are indistinguishable to the caller and which one is not.
//
// Unknown address, wrong password and a broken user lookup all answer with the
// SAME sentinel and the SAME message — that is the anti-enumeration promise the
// comment in Login makes, and this test is what keeps it.
//
// A deactivated account is the exception, and it is a real hole: it answers
// ErrUserInactive, which internal/server/grpc.go:1321 maps to
// codes.PermissionDenied (HTTP 403) while ErrInvalidCredentials maps to
// codes.Unauthenticated (HTTP 401). An attacker who tries an address therefore
// learns "this account exists but is switched off". The test asserts the
// divergence as it stands rather than the promise it breaks — see the backlog
// unit fix-login-inactive-account-is-an-enumeration-oracle.
func TestLogin_FailurePaths_ShareOneAnswerExceptInactive(t *testing.T) {
	const password = "correct-horse-battery"

	t.Run("unknown email, wrong password and broken lookup are one answer", func(t *testing.T) {
		answers := make([]error, 0, 3)

		for _, tc := range []struct {
			name  string
			setup func(*authPathsRepo)
			email string
			pass  string
		}{
			{
				name:  "unknown email",
				setup: func(*authPathsRepo) {},
				email: "nobody@example.com",
				pass:  password,
			},
			{
				name: "wrong password",
				setup: func(r *authPathsRepo) {
					createTestUser(r.mockRepository, "known@example.com", password, true)
				},
				email: "known@example.com",
				pass:  "not-the-password",
			},
			{
				name: "lookup error",
				setup: func(r *authPathsRepo) {
					r.lookupErr = errors.New("connection refused")
				},
				email: "known@example.com",
				pass:  password,
			},
		} {
			svc, repo := newAuthPathsService()
			tc.setup(repo)

			result, err := svc.Login(context.Background(), tc.email, tc.pass)
			require.Error(t, err, tc.name)
			assert.Nil(t, result, tc.name)
			assert.ErrorIs(t, err, ErrInvalidCredentials, tc.name)
			answers = append(answers, err)
		}

		for _, err := range answers {
			assert.Equal(t, ErrInvalidCredentials.Error(), err.Error(),
				"every non-inactive failure must read identically to the caller")
		}
	})

	t.Run("deactivated account answers differently (enumeration oracle)", func(t *testing.T) {
		svc, repo := newAuthPathsService()
		createTestUser(repo.mockRepository, "off@example.com", password, false)

		_, err := svc.Login(context.Background(), "off@example.com", password)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUserInactive)
		assert.NotEqual(t, ErrInvalidCredentials.Error(), err.Error(),
			"documented divergence: a deactivated account is distinguishable from an unknown one")
	})

	t.Run("wrong password on a deactivated account still reveals the account", func(t *testing.T) {
		// The inactive check runs BEFORE the password comparison, so the
		// oracle does not even need a correct password.
		svc, repo := newAuthPathsService()
		createTestUser(repo.mockRepository, "off@example.com", password, false)

		_, err := svc.Login(context.Background(), "off@example.com", "wrong")
		assert.ErrorIs(t, err, ErrUserInactive)
	})

	t.Run("email is matched case-insensitively on every failure path", func(t *testing.T) {
		svc, repo := newAuthPathsService()
		createTestUser(repo.mockRepository, "known@example.com", password, true)

		result, err := svc.Login(context.Background(), "  KNOWN@Example.com ", password)
		require.NoError(t, err)
		assert.NotEmpty(t, result.AccessToken)
	})
}

// TestLogin_NoBruteForceBrakeInTheService records what the service does NOT do:
// there is no failed-attempt counter and no account lockout. Ten wrong
// passwords in a row are ten identical answers and leave no state behind.
// The only brake in front of /api/v1/auth/login is the global per-IP limiter
// (RATE_LIMIT_RPS, default 100 — cmd/gateway/main.go:162); the strict public
// limiter (default 10) is wired to the booking/wiki/reset-page routes, not to
// login. If a lockout is ever added, this test is the one that has to change.
func TestLogin_NoBruteForceBrakeInTheService(t *testing.T) {
	svc, repo := newAuthPathsService()
	user := createTestUser(repo.mockRepository, "target@example.com", "right-password", true)

	for i := 0; i < 10; i++ {
		_, err := svc.Login(context.Background(), "target@example.com", "wrong")
		assert.ErrorIs(t, err, ErrInvalidCredentials, "attempt %d", i+1)
	}

	assert.True(t, repo.users[user.ID].IsActive,
		"no lockout exists: ten failures leave the account untouched")

	result, err := svc.Login(context.Background(), "target@example.com", "right-password")
	require.NoError(t, err)
	assert.NotEmpty(t, result.AccessToken, "the 11th attempt with the right password still succeeds")
}

// ============================================================================
// 2FA login: pending token, TOTP, recovery codes
// ============================================================================

func TestLogin_TwoFactorEnabled_IssuesOnlyAPendingToken(t *testing.T) {
	svc, repo := newAuthPathsService()
	user := createTestUser(repo.mockRepository, "2fa@example.com", "pw", true)
	enrol2FA(t, svc, user)

	result, err := svc.Login(context.Background(), "2fa@example.com", "pw")
	require.NoError(t, err)

	assert.True(t, result.RequiresTwoFactor)
	assert.NotEmpty(t, result.PendingToken)
	assert.Empty(t, result.AccessToken, "no access token before the second factor")
	assert.Empty(t, result.RefreshToken, "no refresh token before the second factor")
	assert.Nil(t, result.User, "the user object must not leak before the second factor")

	gotID, err := svc.tokenMaker.ValidatePendingToken(result.PendingToken)
	require.NoError(t, err)
	assert.Equal(t, user.ID, gotID)
}

func TestCompleteTwoFactorLogin_TOTPPath(t *testing.T) {
	svc, repo := newAuthPathsService()
	user := createTestUser(repo.mockRepository, "2fa@example.com", "pw", true)
	secret, _ := enrol2FA(t, svc, user)

	pending, err := svc.tokenMaker.CreatePendingToken(user.ID)
	require.NoError(t, err)

	code, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err)

	got, tokens, err := svc.CompleteTwoFactorLogin(context.Background(), pending, code, false)
	require.NoError(t, err)
	assert.Equal(t, user.ID, got.ID)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
}

func TestCompleteTwoFactorLogin_RejectedTokensAndCodes(t *testing.T) {
	svc, repo := newAuthPathsService()
	user := createTestUser(repo.mockRepository, "2fa@example.com", "pw", true)
	enrol2FA(t, svc, user)

	validPending, err := svc.tokenMaker.CreatePendingToken(user.ID)
	require.NoError(t, err)

	// A full access token must not double as a pending token: it carries no
	// "2fa_pending" type, so the second factor cannot be skipped with it.
	loginTokens, err := svc.createTokenPair(context.Background(), user, nil)
	require.NoError(t, err)

	tests := []struct {
		name    string
		token   string
		code    string
		wantErr error
	}{
		{"garbage token", "not-a-jwt", "000000", ErrInvalidPendingToken},
		{"expired pending token", signPendingToken(t, svc.tokenMaker, user.ID, "2fa_pending", time.Now().Add(-time.Second)), "000000", ErrInvalidPendingToken},
		{"wrong token type", signPendingToken(t, svc.tokenMaker, user.ID, "access", time.Now().Add(time.Hour)), "000000", ErrInvalidPendingToken},
		{"access token as pending token", loginTokens.AccessToken, "000000", ErrInvalidPendingToken},
		{"wrong TOTP code", validPending, "000000", ErrInvalidTOTPCode},
		{"unknown user", signPendingToken(t, svc.tokenMaker, uuid.New(), "2fa_pending", time.Now().Add(time.Hour)), "000000", ErrUserNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUser, tokens, err := svc.CompleteTwoFactorLogin(context.Background(), tt.token, tt.code, false)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, gotUser)
			assert.Nil(t, tokens)
		})
	}
}

func TestCompleteTwoFactorLogin_DeactivatedBetweenTheTwoSteps(t *testing.T) {
	svc, repo := newAuthPathsService()
	user := createTestUser(repo.mockRepository, "2fa@example.com", "pw", true)
	secret, _ := enrol2FA(t, svc, user)

	pending, err := svc.tokenMaker.CreatePendingToken(user.ID)
	require.NoError(t, err)

	// The account is switched off while the pending token is still valid.
	user.IsActive = false

	code, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err)

	_, tokens, err := svc.CompleteTwoFactorLogin(context.Background(), pending, code, false)
	assert.ErrorIs(t, err, ErrUserInactive)
	assert.Nil(t, tokens)
}

func TestCompleteTwoFactorLogin_RecoveryCodeIsSingleUse(t *testing.T) {
	svc, repo := newAuthPathsService()
	user := createTestUser(repo.mockRepository, "2fa@example.com", "pw", true)
	_, codes := enrol2FA(t, svc, user)

	newPending := func() string {
		token, err := svc.tokenMaker.CreatePendingToken(user.ID)
		require.NoError(t, err)
		return token
	}

	// First use succeeds.
	got, tokens, err := svc.CompleteTwoFactorLogin(context.Background(), newPending(), codes[0], true)
	require.NoError(t, err)
	assert.Equal(t, user.ID, got.ID)
	assert.NotEmpty(t, tokens.AccessToken)

	// Replaying the very same code fails: it is marked used.
	_, _, err = svc.CompleteTwoFactorLogin(context.Background(), newPending(), codes[0], true)
	assert.ErrorIs(t, err, ErrInvalidRecoveryCode)

	// A code that was never issued fails the same way.
	_, _, err = svc.CompleteTwoFactorLogin(context.Background(), newPending(), "deadbeef00", true)
	assert.ErrorIs(t, err, ErrInvalidRecoveryCode)

	// Burn the rest, then the exhausted set answers with its own sentinel.
	for _, c := range codes[1:] {
		_, _, useErr := svc.CompleteTwoFactorLogin(context.Background(), newPending(), c, true)
		require.NoError(t, useErr)
	}
	_, _, err = svc.CompleteTwoFactorLogin(context.Background(), newPending(), codes[0], true)
	assert.ErrorIs(t, err, ErrAllRecoveryCodesUsed)
}

func TestValidate2FALogin_WithoutEnrolment(t *testing.T) {
	svc, repo := newAuthPathsService()
	user := createTestUser(repo.mockRepository, "plain@example.com", "pw", true)

	used, err := svc.Validate2FALogin(context.Background(), user.ID, "000000", false)
	assert.ErrorIs(t, err, ErrTwoFactorNotEnabled)
	assert.False(t, used)

	_, err = svc.Validate2FALogin(context.Background(), uuid.New(), "000000", false)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

// ============================================================================
// 2FA enforcement by policy
// ============================================================================

func TestCheck2FAEnforcement_GraceWindow(t *testing.T) {
	setup := func(gracePeriodDays int, accountAge time.Duration) (*Service, *models.User) {
		svc, repo := newAuthPathsService()
		user := createTestUser(repo.mockRepository, "member@example.com", "pw", true)
		user.TenantID = models.DefaultTenantID
		user.CreatedAt = time.Now().Add(-accountAge)
		repo.userRoles[user.ID] = []string{"admin"}
		repo.policies["admin"] = &models.TwoFactorPolicy{
			ID:              uuid.New(),
			TenantID:        models.DefaultTenantID,
			RoleName:        "admin",
			Enforced:        true,
			GracePeriodDays: gracePeriodDays,
		}
		return svc, user
	}

	t.Run("inside the grace window login still works", func(t *testing.T) {
		svc, user := setup(7, 24*time.Hour)

		res, err := svc.Check2FAEnforcement(context.Background(), user.ID)
		require.NoError(t, err)
		assert.True(t, res.Required)
		assert.False(t, res.GraceExpired)
		assert.Equal(t, []string{"admin"}, res.EnforcedByRoles)
		require.NotNil(t, res.GraceDeadline)

		_, err = svc.Login(context.Background(), "member@example.com", "pw")
		assert.NoError(t, err)
	})

	t.Run("expired grace window blocks the login", func(t *testing.T) {
		svc, user := setup(1, 72*time.Hour)

		res, err := svc.Check2FAEnforcement(context.Background(), user.ID)
		require.NoError(t, err)
		assert.True(t, res.Required)
		assert.True(t, res.GraceExpired)

		_, err = svc.Login(context.Background(), "member@example.com", "pw")
		assert.ErrorIs(t, err, Err2FAEnforcementRequired)
	})

	t.Run("an enrolled account satisfies enforcement", func(t *testing.T) {
		svc, user := setup(1, 72*time.Hour)
		enrol2FA(t, svc, user)

		res, err := svc.Check2FAEnforcement(context.Background(), user.ID)
		require.NoError(t, err)
		assert.False(t, res.Required)

		// And the login now stops at the pending token instead of the block.
		result, err := svc.Login(context.Background(), "member@example.com", "pw")
		require.NoError(t, err)
		assert.True(t, result.RequiresTwoFactor)
	})

	t.Run("unknown user", func(t *testing.T) {
		svc, _ := newAuthPathsService()
		_, err := svc.Check2FAEnforcement(context.Background(), uuid.New())
		assert.ErrorIs(t, err, ErrUserNotFound)
	})
}

func TestTwoFactorPolicy_UpdateAndRead(t *testing.T) {
	svc, repo := newAuthPathsService()
	admin := uuid.New()

	policy, err := svc.UpdateTwoFactorPolicy(context.Background(), models.DefaultTenantID, "admin", true, 14, admin)
	require.NoError(t, err)
	assert.True(t, policy.Enforced)
	assert.Equal(t, 14, policy.GracePeriodDays)
	require.NotNil(t, policy.UpdatedBy)
	assert.Equal(t, admin, *policy.UpdatedBy)
	assert.Equal(t, models.DefaultTenantID, policy.TenantID,
		"the policy carries the tenant it was written for, not the caller's default")

	// The read-through methods are thin, but they are the ones the gRPC layer
	// calls — a wrong tenant argument here would hand out another tenant's policy.
	repo.policies["admin"] = policy
	got, err := svc.GetTwoFactorPolicy(context.Background(), models.DefaultTenantID, "admin")
	require.NoError(t, err)
	assert.Equal(t, policy.ID, got.ID)

	list, err := svc.ListTwoFactorPolicies(context.Background(), models.DefaultTenantID)
	require.NoError(t, err)
	require.Len(t, list, 1)

	foreign, err := svc.ListTwoFactorPolicies(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Empty(t, foreign, "a foreign tenant sees no policy")
}

// ============================================================================
// 2FA management: disable, regenerate, admin reset
// ============================================================================

func TestDisable2FA_NeedsAValidCode(t *testing.T) {
	svc, repo := newAuthPathsService()
	user := createTestUser(repo.mockRepository, "2fa@example.com", "pw", true)
	secret, _ := enrol2FA(t, svc, user)

	assert.ErrorIs(t, svc.Disable2FA(context.Background(), user.ID, "000000"), ErrInvalidTOTPCode)
	assert.True(t, user.TwoFactorEnabled, "a wrong code leaves 2FA on")

	assert.ErrorIs(t, svc.Disable2FA(context.Background(), uuid.New(), "000000"), ErrUserNotFound)

	code, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err)
	require.NoError(t, svc.Disable2FA(context.Background(), user.ID, code))
	assert.False(t, user.TwoFactorEnabled)
	assert.Empty(t, user.TwoFactorSecretEncrypted)

	assert.ErrorIs(t, svc.Disable2FA(context.Background(), user.ID, code), ErrTwoFactorNotEnabled)
}

func TestRegenerateRecoveryCodes_ReplacesTheOldSet(t *testing.T) {
	svc, repo := newAuthPathsService()
	user := createTestUser(repo.mockRepository, "2fa@example.com", "pw", true)
	secret, oldCodes := enrol2FA(t, svc, user)

	_, err := svc.RegenerateRecoveryCodes(context.Background(), user.ID, "000000")
	assert.ErrorIs(t, err, ErrInvalidTOTPCode)

	code, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err)

	newCodes, err := svc.RegenerateRecoveryCodes(context.Background(), user.ID, code)
	require.NoError(t, err)
	require.Len(t, newCodes, recoveryCodeCount)
	assert.NotEqual(t, oldCodes, newCodes)

	// An old code must be dead the moment the set is replaced.
	pending, err := svc.tokenMaker.CreatePendingToken(user.ID)
	require.NoError(t, err)
	_, _, err = svc.CompleteTwoFactorLogin(context.Background(), pending, oldCodes[0], true)
	assert.ErrorIs(t, err, ErrInvalidRecoveryCode)

	// A new one works.
	pending, err = svc.tokenMaker.CreatePendingToken(user.ID)
	require.NoError(t, err)
	_, tokens, err := svc.CompleteTwoFactorLogin(context.Background(), pending, newCodes[0], true)
	require.NoError(t, err)
	assert.NotEmpty(t, tokens.AccessToken)
}

func TestAdminReset2FA_RequiresAReasonAndAnEnrolledAccount(t *testing.T) {
	svc, repo := newAuthPathsService()
	user := createTestUser(repo.mockRepository, "2fa@example.com", "pw", true)
	enrol2FA(t, svc, user)
	admin := uuid.New()

	err := svc.AdminReset2FA(context.Background(), user.ID, admin, "")
	require.Error(t, err, "an audit trail without a reason is worthless")
	assert.True(t, user.TwoFactorEnabled)

	assert.ErrorIs(t, svc.AdminReset2FA(context.Background(), uuid.New(), admin, "lost phone"), ErrUserNotFound)

	require.NoError(t, svc.AdminReset2FA(context.Background(), user.ID, admin, "lost phone"))
	assert.False(t, user.TwoFactorEnabled)

	assert.ErrorIs(t, svc.AdminReset2FA(context.Background(), user.ID, admin, "lost phone"), ErrTwoFactorNotEnabled)
}

func TestVerify2FA_RejectionPaths(t *testing.T) {
	svc, repo := newAuthPathsService()
	user := createTestUser(repo.mockRepository, "2fa@example.com", "pw", true)

	_, err := svc.Verify2FA(context.Background(), user.ID, "000000")
	assert.ErrorIs(t, err, ErrNo2FASetupPending, "verify without setup has nothing to check against")

	_, err = svc.Setup2FA(context.Background(), user.ID)
	require.NoError(t, err)

	_, err = svc.Verify2FA(context.Background(), user.ID, "000000")
	assert.ErrorIs(t, err, ErrInvalidTOTPCode)
	assert.False(t, user.TwoFactorEnabled, "a wrong first code must not enable 2FA")

	enrol2FA(t, svc, user)
	_, err = svc.Setup2FA(context.Background(), user.ID)
	assert.ErrorIs(t, err, ErrTwoFactorAlreadyEnabled)

	_, err = svc.Setup2FA(context.Background(), uuid.New())
	assert.ErrorIs(t, err, ErrUserNotFound)
}

// ============================================================================
// Access-token lifetime after password and role changes
// ============================================================================

// TestAccessTokenOutlivesPasswordChange pins the deliberate asymmetry: the
// access token is a signed JWT and nothing revokes it, so it keeps working
// until it expires (15 min in production, config.JWTAccessExpiry). What the
// password change does kill is the refresh token, so the session dies at the
// next rotation instead of immediately.
//
// This is intended, not a bug — but it IS a security property, so it is written
// down here rather than left to be rediscovered.
func TestAccessTokenOutlivesPasswordChange(t *testing.T) {
	svc, repo := newAuthPathsService()
	createTestUser(repo.mockRepository, "user@example.com", "old-password", true)

	login, err := svc.Login(context.Background(), "user@example.com", "old-password")
	require.NoError(t, err)

	require.NoError(t, svc.ChangePassword(context.Background(), login.User.ID, "old-password", "new-password"))

	claims, err := svc.ValidateToken(context.Background(), login.AccessToken)
	require.NoError(t, err, "the access token survives the password change by design")
	assert.Equal(t, login.User.ID.String(), claims.UserID)

	_, err = svc.RefreshToken(context.Background(), login.RefreshToken)
	assert.ErrorIs(t, err, ErrTokenRevoked, "the refresh token does not survive it")

	_, err = svc.Login(context.Background(), "user@example.com", "old-password")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

// TestAccessTokenOutlivesPasswordReset is the same promise for the reset path,
// which revokes tokens through a different call site (ResetPassword, not
// ChangePassword) and is the one an attacker-triggered lockout would use.
func TestAccessTokenOutlivesPasswordReset(t *testing.T) {
	svc, repo := newAuthPathsService()
	user := createTestUser(repo.mockRepository, "user@example.com", "old-password", true)
	svc.SetPasswordValidator(&mockPasswordValidator{valid: true})

	login, err := svc.Login(context.Background(), "user@example.com", "old-password")
	require.NoError(t, err)

	plain := generateSecureToken()
	repo.passwordResetTokens[HashToken(plain)] = &models.PasswordResetToken{
		ID:        uuid.New(),
		TenantID:  user.TenantID,
		UserID:    user.ID,
		TokenHash: HashToken(plain),
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}

	require.NoError(t, svc.ResetPassword(context.Background(), plain, "brand-new-password"))

	_, err = svc.ValidateToken(context.Background(), login.AccessToken)
	assert.NoError(t, err, "same asymmetry as ChangePassword")

	_, err = svc.RefreshToken(context.Background(), login.RefreshToken)
	assert.ErrorIs(t, err, ErrTokenRevoked)
}

// TestAccessTokenOutlivesRoleChange pins the second half of the same property:
// roles and permissions are baked into the JWT at login and are NOT re-read per
// request, so a role taken away only bites at the next refresh. Every
// RequirePermission guard depends on this — it is why a guard may only ever be
// widened with RequirePermissionAny, never replaced.
func TestAccessTokenOutlivesRoleChange(t *testing.T) {
	svc, repo := newAuthPathsService()
	user := createTestUser(repo.mockRepository, "user@example.com", "pw", true)
	repo.userRoles[user.ID] = []string{"member"}

	login, err := svc.Login(context.Background(), "user@example.com", "pw")
	require.NoError(t, err)

	before, err := svc.ValidateToken(context.Background(), login.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, []string{"member"}, before.Roles)

	require.NoError(t, svc.RemoveRole(context.Background(), user.ID, "member"))
	require.NoError(t, svc.AssignRole(context.Background(), user.ID, "admin"))

	after, err := svc.ValidateToken(context.Background(), login.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, []string{"member"}, after.Roles,
		"the old token still carries the old roles: claims are minted, not looked up")

	// The refresh is where the new roles arrive.
	tokens, err := svc.RefreshToken(context.Background(), login.RefreshToken)
	require.NoError(t, err)
	refreshed, err := svc.ValidateToken(context.Background(), tokens.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, []string{"admin"}, refreshed.Roles)
}

func TestRefreshToken_ReuseOfARevokedTokenKillsEverySession(t *testing.T) {
	svc, repo := newAuthPathsService()
	createTestUser(repo.mockRepository, "user@example.com", "pw", true)

	first, err := svc.Login(context.Background(), "user@example.com", "pw")
	require.NoError(t, err)
	second, err := svc.Login(context.Background(), "user@example.com", "pw")
	require.NoError(t, err)

	// Rotate the first one normally.
	rotated, err := svc.RefreshToken(context.Background(), first.RefreshToken)
	require.NoError(t, err)

	// Replaying the consumed token is the theft signal: everything dies.
	_, err = svc.RefreshToken(context.Background(), first.RefreshToken)
	assert.ErrorIs(t, err, ErrTokenRevoked)

	_, err = svc.RefreshToken(context.Background(), rotated.RefreshToken)
	assert.ErrorIs(t, err, ErrTokenRevoked, "the rotated token is revoked too")
	_, err = svc.RefreshToken(context.Background(), second.RefreshToken)
	assert.ErrorIs(t, err, ErrTokenRevoked, "the second device is signed out as well")
}

func TestRefreshToken_ExpiredIsRejectedWithoutRevokingAnything(t *testing.T) {
	svc, repo := newAuthPathsService()
	createTestUser(repo.mockRepository, "user@example.com", "pw", true)

	login, err := svc.Login(context.Background(), "user@example.com", "pw")
	require.NoError(t, err)

	stored := repo.refreshTokens[HashToken(login.RefreshToken)]
	require.NotNil(t, stored)
	stored.ExpiresAt = time.Now().Add(-time.Minute)

	_, err = svc.RefreshToken(context.Background(), login.RefreshToken)
	assert.ErrorIs(t, err, ErrTokenExpired)
	assert.False(t, stored.Revoked, "an expired token is not a theft signal")
}

// ============================================================================
// Password-reset token: user binding and re-issue
// ============================================================================

// TestResetPassword_TokenIsBoundToItsUser is the one property the other reset
// tests do not cover: the new password lands on the account the token was
// issued for, and on no other.
func TestResetPassword_TokenIsBoundToItsUser(t *testing.T) {
	svc, repo := newAuthPathsService()
	svc.SetMailer(&mockMailer{})
	svc.SetResetBaseURL("https://app.test.local/reset-password")
	svc.SetPasswordValidator(&mockPasswordValidator{valid: true})

	victim := createTestUser(repo.mockRepository, "victim@example.com", "victim-password", true)
	other := createTestUser(repo.mockRepository, "other@example.com", "other-password", true)
	otherHashBefore := other.PasswordHash

	require.NoError(t, svc.RequestPasswordReset(context.Background(), "victim@example.com"))
	require.Len(t, repo.passwordResetTokens, 1)

	var stored *models.PasswordResetToken
	for _, prt := range repo.passwordResetTokens {
		stored = prt
	}
	require.NotNil(t, stored)
	assert.Equal(t, victim.ID, stored.UserID, "the token names the account it was issued for")

	link := svc.mailer.(*mockMailer).lastLink
	require.Contains(t, link, "?token=")
	plain := link[len(link)-64:]

	require.NoError(t, svc.ResetPassword(context.Background(), plain, "victim-new-password"))

	_, err := svc.Login(context.Background(), "victim@example.com", "victim-new-password")
	assert.NoError(t, err)
	assert.Equal(t, otherHashBefore, other.PasswordHash, "the second account is untouched")

	_, err = svc.Login(context.Background(), "other@example.com", "other-password")
	assert.NoError(t, err)
}

// TestRequestPasswordReset_ASecondRequestDoesNotKillTheFirstToken records a
// property worth knowing before it is discovered in an incident: asking for a
// reset twice leaves BOTH links usable until one is spent or an hour passes.
// Whoever reads the older mail (a shared mailbox, a forwarded message) can
// still take the account over. Pinned as-is; the fix is a backlog unit
// (harden-password-reset-invalidate-previous-tokens).
func TestRequestPasswordReset_ASecondRequestDoesNotKillTheFirstToken(t *testing.T) {
	svc, repo := newAuthPathsService()
	mailer := &mockMailer{}
	svc.SetMailer(mailer)
	svc.SetResetBaseURL("https://app.test.local/reset-password")
	svc.SetPasswordValidator(&mockPasswordValidator{valid: true})
	createTestUser(repo.mockRepository, "user@example.com", "pw", true)

	require.NoError(t, svc.RequestPasswordReset(context.Background(), "user@example.com"))
	firstLink := mailer.lastLink
	require.NoError(t, svc.RequestPasswordReset(context.Background(), "user@example.com"))
	secondLink := mailer.lastLink

	require.NotEqual(t, firstLink, secondLink)
	assert.Len(t, repo.passwordResetTokens, 2,
		"the first token is not invalidated by the second request")

	first := firstLink[len(firstLink)-64:]
	assert.NoError(t, svc.ResetPassword(context.Background(), first, "taken-over-password"),
		"the OLDER link still works")
}
