package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/sysctx"
)

// ctxCapturingRepo wraps mockRepository and records the IsSystemContext
// state of the ctx passed to each pre-JWT method. Welle 2a contract is
// that Login/Register/RefreshToken/Logout/CompleteTwoFactorLogin/
// AcceptInvitation set sysctx.With before any repo call so RLS policies
// admit the user/session lookup.
type ctxCapturingRepo struct {
	*mockRepository

	getUserByEmailSysCtx        bool
	getUserByEmailCalled        bool
	getUserByIDSysCtx           bool
	getRefreshTokenByHashSysCtx bool
	getInvitationByTokenSysCtx  bool
	updateUserSysCtx            bool
}

func (r *ctxCapturingRepo) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	r.getUserByEmailCalled = true
	r.getUserByEmailSysCtx = sysctx.Is(ctx)
	return r.mockRepository.GetUserByEmail(ctx, email)
}

func (r *ctxCapturingRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	r.getUserByIDSysCtx = sysctx.Is(ctx)
	return r.mockRepository.GetUserByID(ctx, id)
}

func (r *ctxCapturingRepo) GetRefreshTokenByHash(ctx context.Context, hash string) (*models.RefreshToken, error) {
	r.getRefreshTokenByHashSysCtx = sysctx.Is(ctx)
	return r.mockRepository.GetRefreshTokenByHash(ctx, hash)
}

func (r *ctxCapturingRepo) GetInvitationByToken(ctx context.Context, hash string) (*models.Invitation, error) {
	r.getInvitationByTokenSysCtx = sysctx.Is(ctx)
	return r.mockRepository.GetInvitationByToken(ctx, hash)
}

func (r *ctxCapturingRepo) UpdateUser(ctx context.Context, user *models.User) error {
	r.updateUserSysCtx = sysctx.Is(ctx)
	return r.mockRepository.UpdateUser(ctx, user)
}

func newCtxCapturingService(t *testing.T) (*Service, *ctxCapturingRepo) {
	t.Helper()
	repo := &ctxCapturingRepo{mockRepository: newMockRepository()}
	tm := NewTokenMaker("test-secret-min-32-chars-test-secret-x", 15*time.Minute, 7*24*time.Hour)
	return NewService(repo, tm), repo
}

func TestService_Login_WrapsWithSystemContext(t *testing.T) {
	svc, repo := newCtxCapturingService(t)

	hash, _ := bcrypt.GenerateFromPassword([]byte("pw12345678"), bcryptCost)
	repo.users[uuid.New()] = nil
	user := &models.User{
		ID:           uuid.New(),
		TenantID:     models.DefaultTenantID,
		Email:        "rls-login@test.local",
		PasswordHash: string(hash),
		IsActive:     true,
	}
	repo.usersByEmail[user.Email] = user
	repo.users[user.ID] = user

	if _, err := svc.Login(context.Background(), user.Email, "pw12345678"); err != nil {
		t.Fatalf("login: %v", err)
	}

	if !repo.getUserByEmailCalled {
		t.Fatalf("expected GetUserByEmail to be called")
	}
	if !repo.getUserByEmailSysCtx {
		t.Errorf("Login must wrap ctx with sysctx.With before repo lookup")
	}
}

func TestService_Register_WrapsWithSystemContext(t *testing.T) {
	svc, repo := newCtxCapturingService(t)

	if _, _, err := svc.Register(context.Background(), "rls-register@test.local", "pw12345678", "A", "B"); err != nil {
		t.Fatalf("register: %v", err)
	}

	if !repo.getUserByEmailSysCtx {
		t.Errorf("Register must wrap ctx with sysctx.With before existing-user check")
	}
}

func TestService_RefreshToken_WrapsWithSystemContext(t *testing.T) {
	svc, repo := newCtxCapturingService(t)

	user := &models.User{ID: uuid.New(), TenantID: models.DefaultTenantID, Email: "rls-refresh@test.local", IsActive: true}
	repo.users[user.ID] = user

	rt := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: HashToken("opaque-refresh-token"),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	repo.refreshTokens[rt.TokenHash] = rt

	if _, err := svc.RefreshToken(context.Background(), "opaque-refresh-token"); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	if !repo.getRefreshTokenByHashSysCtx {
		t.Errorf("RefreshToken must wrap ctx with sysctx.With before repo lookup")
	}
	if !repo.getUserByIDSysCtx {
		t.Errorf("RefreshToken must propagate sysctx through to GetUserByID")
	}
}

func TestService_Logout_WrapsWithSystemContext(t *testing.T) {
	svc, repo := newCtxCapturingService(t)

	rt := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: HashToken("opaque-refresh-token"),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	repo.refreshTokens[rt.TokenHash] = rt

	if err := svc.Logout(context.Background(), "opaque-refresh-token"); err != nil {
		t.Fatalf("logout: %v", err)
	}

	if !repo.getRefreshTokenByHashSysCtx {
		t.Errorf("Logout must wrap ctx with sysctx.With before refresh-token lookup")
	}
}

func TestService_AcceptInvitation_WrapsWithSystemContext(t *testing.T) {
	svc, repo := newCtxCapturingService(t)

	tokenHash := HashToken("invite-token")
	repo.invByToken[tokenHash] = &models.Invitation{
		ID:        uuid.New(),
		Email:     "rls-invite@test.local",
		Role:      "member",
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	if _, _, err := svc.AcceptInvitation(context.Background(), "invite-token", "pw12345678", "A", "B"); err != nil {
		t.Fatalf("accept: %v", err)
	}

	if !repo.getInvitationByTokenSysCtx {
		t.Errorf("AcceptInvitation must wrap ctx with sysctx.With before invitation lookup")
	}
}

func TestService_UpdateUser_DoesNotWrapWithSystemContext(t *testing.T) {
	svc, repo := newCtxCapturingService(t)

	user := &models.User{ID: uuid.New(), TenantID: models.DefaultTenantID, Email: "rls-update@test.local", IsActive: true}
	repo.users[user.ID] = user

	first := "Updated"
	if _, err := svc.UpdateUser(context.Background(), user.ID, &first, nil, nil); err != nil {
		t.Fatalf("update: %v", err)
	}

	if repo.updateUserSysCtx {
		t.Errorf("UpdateUser must NOT wrap with sysctx (post-JWT path; tenant flows through middleware)")
	}
}
