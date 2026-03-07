package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/models"
)

// ---------------------------------------------------------------------------
// gRPC assertion helpers
// ---------------------------------------------------------------------------

func requireGRPCCode(t *testing.T, err error, code codes.Code) {
	t.Helper()
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "expected gRPC status error, got %T: %v", err, err)
	require.Equal(t, code, st.Code(), "expected gRPC code %s, got %s: %s", code, st.Code(), st.Message())
}

func requireGRPCOK(t *testing.T, err error) {
	t.Helper()
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Auth mock repository  (implements auth.Repository)
// ---------------------------------------------------------------------------

type authMockRepo struct {
	users         map[uuid.UUID]*models.User
	usersByEmail  map[string]*models.User
	refreshTokens map[string]*models.RefreshToken // keyed by token_hash
	userRoles     map[uuid.UUID][]string
	userPerms     map[uuid.UUID][]string
	invitations   map[uuid.UUID]*models.Invitation
	invByToken    map[string]*models.Invitation
	sessions      map[uuid.UUID]*models.UserSession
	policies      map[string]*models.TwoFactorPolicy
}

func newAuthMockRepo() *authMockRepo {
	return &authMockRepo{
		users:         make(map[uuid.UUID]*models.User),
		usersByEmail:  make(map[string]*models.User),
		refreshTokens: make(map[string]*models.RefreshToken),
		userRoles:     make(map[uuid.UUID][]string),
		userPerms:     make(map[uuid.UUID][]string),
		invitations:   make(map[uuid.UUID]*models.Invitation),
		invByToken:    make(map[string]*models.Invitation),
		sessions:      make(map[uuid.UUID]*models.UserSession),
		policies:      make(map[string]*models.TwoFactorPolicy),
	}
}

// --- User CRUD ---

func (m *authMockRepo) CreateUser(_ context.Context, user *models.User) error {
	m.users[user.ID] = user
	m.usersByEmail[user.Email] = user
	return nil
}

func (m *authMockRepo) GetUserByEmail(_ context.Context, email string) (*models.User, error) {
	u, ok := m.usersByEmail[email]
	if !ok {
		return nil, auth.ErrUserNotFound
	}
	return u, nil
}

func (m *authMockRepo) GetUserByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, auth.ErrUserNotFound
	}
	return u, nil
}

func (m *authMockRepo) UpdateUser(_ context.Context, user *models.User) error {
	m.users[user.ID] = user
	m.usersByEmail[user.Email] = user
	return nil
}

func (m *authMockRepo) ListUsers(_ context.Context, offset, limit int) ([]*models.User, int, error) {
	var all []*models.User
	for _, u := range m.users {
		all = append(all, u)
	}
	total := len(all)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func (m *authMockRepo) UpdatePassword(_ context.Context, userID uuid.UUID, passwordHash string) error {
	user, ok := m.users[userID]
	if !ok {
		return auth.ErrUserNotFound
	}
	user.PasswordHash = passwordHash
	return nil
}

// --- Tokens ---

func (m *authMockRepo) StoreRefreshToken(_ context.Context, token *models.RefreshToken) error {
	m.refreshTokens[token.TokenHash] = token
	return nil
}

func (m *authMockRepo) GetRefreshTokenByHash(_ context.Context, hash string) (*models.RefreshToken, error) {
	t, ok := m.refreshTokens[hash]
	if !ok {
		return nil, auth.ErrTokenInvalid
	}
	return t, nil
}

func (m *authMockRepo) RevokeRefreshToken(_ context.Context, id uuid.UUID) error {
	for _, t := range m.refreshTokens {
		if t.ID == id {
			t.Revoked = true
			return nil
		}
	}
	return nil
}

func (m *authMockRepo) RevokeAllUserTokens(_ context.Context, userID uuid.UUID) error {
	for _, t := range m.refreshTokens {
		if t.UserID == userID {
			t.Revoked = true
		}
	}
	return nil
}

// --- Roles & Permissions ---

func (m *authMockRepo) AssignRole(_ context.Context, userID uuid.UUID, roleName string) error {
	m.userRoles[userID] = append(m.userRoles[userID], roleName)
	return nil
}

func (m *authMockRepo) RemoveRole(_ context.Context, userID uuid.UUID, roleName string) error {
	roles := m.userRoles[userID]
	for i, r := range roles {
		if r == roleName {
			m.userRoles[userID] = append(roles[:i], roles[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *authMockRepo) GetUserRoles(_ context.Context, userID uuid.UUID) ([]string, error) {
	return m.userRoles[userID], nil
}

func (m *authMockRepo) GetUserPermissions(_ context.Context, userID uuid.UUID) ([]string, error) {
	return m.userPerms[userID], nil
}

func (m *authMockRepo) UserHasPermission(_ context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	target := resource + ":" + action
	for _, p := range m.userPerms[userID] {
		if p == target {
			return true, nil
		}
	}
	return false, nil
}

// --- Invitations ---

func (m *authMockRepo) CreateInvitation(_ context.Context, inv *models.Invitation) error {
	m.invitations[inv.ID] = inv
	m.invByToken[inv.TokenHash] = inv
	return nil
}

func (m *authMockRepo) GetInvitationByToken(_ context.Context, tokenHash string) (*models.Invitation, error) {
	inv, ok := m.invByToken[tokenHash]
	if !ok {
		return nil, auth.ErrInvitationNotFound
	}
	return inv, nil
}

func (m *authMockRepo) GetInvitationByID(_ context.Context, id uuid.UUID) (*models.Invitation, error) {
	inv, ok := m.invitations[id]
	if !ok {
		return nil, auth.ErrInvitationNotFound
	}
	return inv, nil
}

func (m *authMockRepo) ListPendingInvitations(_ context.Context) ([]*models.Invitation, error) {
	var pending []*models.Invitation
	for _, inv := range m.invitations {
		if inv.AcceptedAt == nil {
			pending = append(pending, inv)
		}
	}
	return pending, nil
}

func (m *authMockRepo) MarkInvitationAccepted(_ context.Context, id uuid.UUID) error {
	inv, ok := m.invitations[id]
	if !ok {
		return auth.ErrInvitationNotFound
	}
	now := time.Now()
	inv.AcceptedAt = &now
	return nil
}

func (m *authMockRepo) DeleteInvitation(_ context.Context, id uuid.UUID) error {
	inv, ok := m.invitations[id]
	if !ok {
		return auth.ErrInvitationNotFound
	}
	delete(m.invByToken, inv.TokenHash)
	delete(m.invitations, id)
	return nil
}

// --- 2FA ---

func (m *authMockRepo) StorePending2FASecret(_ context.Context, userID uuid.UUID, encryptedSecret string) error {
	user, ok := m.users[userID]
	if !ok {
		return auth.ErrUserNotFound
	}
	user.TwoFactorPendingSecret = encryptedSecret
	return nil
}

func (m *authMockRepo) GetPending2FASecret(_ context.Context, userID uuid.UUID) (string, error) {
	user, ok := m.users[userID]
	if !ok {
		return "", auth.ErrUserNotFound
	}
	if user.TwoFactorPendingSecret == "" {
		return "", auth.ErrNo2FASetupPending
	}
	return user.TwoFactorPendingSecret, nil
}

func (m *authMockRepo) Enable2FA(_ context.Context, userID uuid.UUID, encryptedSecret string, _ []*models.RecoveryCode) error {
	user, ok := m.users[userID]
	if !ok {
		return auth.ErrUserNotFound
	}
	user.TwoFactorEnabled = true
	user.TwoFactorSecretEncrypted = encryptedSecret
	user.TwoFactorPendingSecret = ""
	now := time.Now()
	user.TwoFactorEnabledAt = &now
	return nil
}

func (m *authMockRepo) Disable2FA(_ context.Context, userID uuid.UUID) error {
	user, ok := m.users[userID]
	if !ok {
		return auth.ErrUserNotFound
	}
	user.TwoFactorEnabled = false
	user.TwoFactorSecretEncrypted = ""
	user.TwoFactorPendingSecret = ""
	user.TwoFactorEnabledAt = nil
	return nil
}

func (m *authMockRepo) GetRecoveryCodes(_ context.Context, _ uuid.UUID) ([]*models.RecoveryCode, error) {
	return nil, nil
}

func (m *authMockRepo) UseRecoveryCode(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *authMockRepo) ReplaceRecoveryCodes(_ context.Context, _ uuid.UUID, _ []*models.RecoveryCode) error {
	return nil
}

// --- 2FA Policies ---

func (m *authMockRepo) GetTwoFactorPolicy(_ context.Context, roleName string) (*models.TwoFactorPolicy, error) {
	p, ok := m.policies[roleName]
	if !ok {
		return nil, nil
	}
	return p, nil
}

func (m *authMockRepo) ListTwoFactorPolicies(_ context.Context) ([]*models.TwoFactorPolicy, error) {
	var all []*models.TwoFactorPolicy
	for _, p := range m.policies {
		all = append(all, p)
	}
	return all, nil
}

func (m *authMockRepo) UpsertTwoFactorPolicy(_ context.Context, policy *models.TwoFactorPolicy) error {
	m.policies[policy.RoleName] = policy
	return nil
}

// --- Sessions ---

func (m *authMockRepo) CreateSession(_ context.Context, session *models.UserSession) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *authMockRepo) GetSession(_ context.Context, id uuid.UUID) (*models.UserSession, error) {
	s, ok := m.sessions[id]
	if !ok {
		return nil, auth.ErrSessionNotFound
	}
	return s, nil
}

func (m *authMockRepo) ListUserSessions(_ context.Context, userID uuid.UUID) ([]*models.UserSession, error) {
	var out []*models.UserSession
	for _, s := range m.sessions {
		if s.UserID == userID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (m *authMockRepo) ListAllSessions(_ context.Context, offset, limit int) ([]*models.UserSession, int, error) {
	var all []*models.UserSession
	for _, s := range m.sessions {
		all = append(all, s)
	}
	total := len(all)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func (m *authMockRepo) UpdateSessionActivity(_ context.Context, sessionID uuid.UUID) error {
	s, ok := m.sessions[sessionID]
	if !ok {
		return auth.ErrSessionNotFound
	}
	s.LastActiveAt = time.Now()
	return nil
}

func (m *authMockRepo) DeleteSession(_ context.Context, id uuid.UUID) error {
	delete(m.sessions, id)
	return nil
}

func (m *authMockRepo) DeleteAllUserSessions(_ context.Context, userID uuid.UUID, exceptSessionID *uuid.UUID) error {
	for id, s := range m.sessions {
		if s.UserID == userID {
			if exceptSessionID != nil && id == *exceptSessionID {
				continue
			}
			delete(m.sessions, id)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Auth test helpers
// ---------------------------------------------------------------------------

const testJWTSecret = "test-secret-minimum-32-characters!"

func newTestAuthGRPCServer() (*AuthGRPCServer, *authMockRepo) {
	repo := newAuthMockRepo()
	tm := auth.NewTokenMaker(testJWTSecret, 15*time.Minute, 7*24*time.Hour)
	svc := auth.NewService(repo, tm)
	return NewAuthGRPCServer(svc), repo
}

func createAuthTestUser(repo *authMockRepo, email, password string, active bool) *models.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	user := &models.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
		FirstName:    "Test",
		LastName:     "User",
		IsActive:     active,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	repo.users[user.ID] = user
	repo.usersByEmail[user.Email] = user
	return user
}
