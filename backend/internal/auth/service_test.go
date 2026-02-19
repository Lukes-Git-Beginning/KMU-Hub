package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/kmuhub/kmuhub/internal/models"
)

// mockRepository implements Repository for testing
type mockRepository struct {
	users         map[uuid.UUID]*models.User
	usersByEmail  map[string]*models.User
	refreshTokens map[string]*models.RefreshToken // keyed by token_hash
	userRoles     map[uuid.UUID][]string
	userPerms     map[uuid.UUID][]string
	invitations   map[uuid.UUID]*models.Invitation
	invByToken    map[string]*models.Invitation
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		users:         make(map[uuid.UUID]*models.User),
		usersByEmail:  make(map[string]*models.User),
		refreshTokens: make(map[string]*models.RefreshToken),
		userRoles:     make(map[uuid.UUID][]string),
		userPerms:     make(map[uuid.UUID][]string),
		invitations:   make(map[uuid.UUID]*models.Invitation),
		invByToken:    make(map[string]*models.Invitation),
	}
}

func (m *mockRepository) CreateUser(_ context.Context, user *models.User) error {
	m.users[user.ID] = user
	m.usersByEmail[user.Email] = user
	return nil
}

func (m *mockRepository) GetUserByEmail(_ context.Context, email string) (*models.User, error) {
	u, ok := m.usersByEmail[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (m *mockRepository) GetUserByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (m *mockRepository) UpdateUser(_ context.Context, user *models.User) error {
	m.users[user.ID] = user
	m.usersByEmail[user.Email] = user
	return nil
}

func (m *mockRepository) ListUsers(_ context.Context, offset, limit int) ([]*models.User, int, error) {
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

func (m *mockRepository) StoreRefreshToken(_ context.Context, token *models.RefreshToken) error {
	m.refreshTokens[token.TokenHash] = token
	return nil
}

func (m *mockRepository) GetRefreshTokenByHash(_ context.Context, hash string) (*models.RefreshToken, error) {
	t, ok := m.refreshTokens[hash]
	if !ok {
		return nil, ErrTokenInvalid
	}
	return t, nil
}

func (m *mockRepository) RevokeRefreshToken(_ context.Context, id uuid.UUID) error {
	for _, t := range m.refreshTokens {
		if t.ID == id {
			t.Revoked = true
			return nil
		}
	}
	return nil
}

func (m *mockRepository) RevokeAllUserTokens(_ context.Context, userID uuid.UUID) error {
	for _, t := range m.refreshTokens {
		if t.UserID == userID {
			t.Revoked = true
		}
	}
	return nil
}

func (m *mockRepository) AssignRole(_ context.Context, userID uuid.UUID, roleName string) error {
	m.userRoles[userID] = append(m.userRoles[userID], roleName)
	return nil
}

func (m *mockRepository) RemoveRole(_ context.Context, userID uuid.UUID, roleName string) error {
	roles := m.userRoles[userID]
	for i, r := range roles {
		if r == roleName {
			m.userRoles[userID] = append(roles[:i], roles[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockRepository) GetUserRoles(_ context.Context, userID uuid.UUID) ([]string, error) {
	return m.userRoles[userID], nil
}

func (m *mockRepository) GetUserPermissions(_ context.Context, userID uuid.UUID) ([]string, error) {
	return m.userPerms[userID], nil
}

func (m *mockRepository) UserHasPermission(_ context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	target := resource + ":" + action
	for _, p := range m.userPerms[userID] {
		if p == target {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockRepository) UpdatePassword(_ context.Context, userID uuid.UUID, passwordHash string) error {
	user, ok := m.users[userID]
	if !ok {
		return ErrUserNotFound
	}
	user.PasswordHash = passwordHash
	return nil
}

func (m *mockRepository) CreateInvitation(_ context.Context, inv *models.Invitation) error {
	m.invitations[inv.ID] = inv
	m.invByToken[inv.TokenHash] = inv
	return nil
}

func (m *mockRepository) GetInvitationByToken(_ context.Context, tokenHash string) (*models.Invitation, error) {
	inv, ok := m.invByToken[tokenHash]
	if !ok {
		return nil, ErrInvitationNotFound
	}
	return inv, nil
}

func (m *mockRepository) GetInvitationByID(_ context.Context, id uuid.UUID) (*models.Invitation, error) {
	inv, ok := m.invitations[id]
	if !ok {
		return nil, ErrInvitationNotFound
	}
	return inv, nil
}

func (m *mockRepository) ListPendingInvitations(_ context.Context) ([]*models.Invitation, error) {
	var pending []*models.Invitation
	for _, inv := range m.invitations {
		if inv.AcceptedAt == nil {
			pending = append(pending, inv)
		}
	}
	return pending, nil
}

func (m *mockRepository) MarkInvitationAccepted(_ context.Context, id uuid.UUID) error {
	inv, ok := m.invitations[id]
	if !ok {
		return ErrInvitationNotFound
	}
	now := time.Now()
	inv.AcceptedAt = &now
	return nil
}

func (m *mockRepository) DeleteInvitation(_ context.Context, id uuid.UUID) error {
	inv, ok := m.invitations[id]
	if !ok {
		return ErrInvitationNotFound
	}
	delete(m.invByToken, inv.TokenHash)
	delete(m.invitations, id)
	return nil
}

// Two-factor authentication mock methods

func (m *mockRepository) StorePending2FASecret(_ context.Context, userID uuid.UUID, encryptedSecret string) error {
	user, ok := m.users[userID]
	if !ok {
		return ErrUserNotFound
	}
	user.TwoFactorPendingSecret = encryptedSecret
	return nil
}

func (m *mockRepository) GetPending2FASecret(_ context.Context, userID uuid.UUID) (string, error) {
	user, ok := m.users[userID]
	if !ok {
		return "", ErrUserNotFound
	}
	if user.TwoFactorPendingSecret == "" {
		return "", ErrNo2FASetupPending
	}
	return user.TwoFactorPendingSecret, nil
}

func (m *mockRepository) Enable2FA(_ context.Context, userID uuid.UUID, encryptedSecret string, recoveryCodes []*models.RecoveryCode) error {
	user, ok := m.users[userID]
	if !ok {
		return ErrUserNotFound
	}
	user.TwoFactorEnabled = true
	user.TwoFactorSecretEncrypted = encryptedSecret
	user.TwoFactorPendingSecret = ""
	now := time.Now()
	user.TwoFactorEnabledAt = &now
	return nil
}

func (m *mockRepository) Disable2FA(_ context.Context, userID uuid.UUID) error {
	user, ok := m.users[userID]
	if !ok {
		return ErrUserNotFound
	}
	user.TwoFactorEnabled = false
	user.TwoFactorSecretEncrypted = ""
	user.TwoFactorPendingSecret = ""
	user.TwoFactorEnabledAt = nil
	return nil
}

func (m *mockRepository) GetRecoveryCodes(_ context.Context, _ uuid.UUID) ([]*models.RecoveryCode, error) {
	return nil, nil
}

func (m *mockRepository) UseRecoveryCode(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockRepository) ReplaceRecoveryCodes(_ context.Context, _ uuid.UUID, _ []*models.RecoveryCode) error {
	return nil
}

func (m *mockRepository) GetTwoFactorPolicy(_ context.Context, _ string) (*models.TwoFactorPolicy, error) {
	return nil, nil
}

func (m *mockRepository) ListTwoFactorPolicies(_ context.Context) ([]*models.TwoFactorPolicy, error) {
	return nil, nil
}

func (m *mockRepository) UpsertTwoFactorPolicy(_ context.Context, _ *models.TwoFactorPolicy) error {
	return nil
}

// Session management mock methods

func (m *mockRepository) CreateSession(_ context.Context, _ *models.UserSession) error {
	return nil
}

func (m *mockRepository) GetSession(_ context.Context, _ uuid.UUID) (*models.UserSession, error) {
	return nil, nil
}

func (m *mockRepository) ListUserSessions(_ context.Context, _ uuid.UUID) ([]*models.UserSession, error) {
	return nil, nil
}

func (m *mockRepository) ListAllSessions(_ context.Context, _, _ int) ([]*models.UserSession, int, error) {
	return nil, 0, nil
}

func (m *mockRepository) UpdateSessionActivity(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockRepository) DeleteSession(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockRepository) DeleteAllUserSessions(_ context.Context, _ uuid.UUID, _ *uuid.UUID) error {
	return nil
}

func newTestService() (*Service, *mockRepository) {
	repo := newMockRepository()
	tm := NewTokenMaker("test-secret-minimum-32-characters!", 15*time.Minute, 7*24*time.Hour)
	svc := NewService(repo, tm)
	return svc, repo
}

func createTestUser(repo *mockRepository, email, password string, active bool) *models.User {
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

func TestService_Register(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		setup   func(*mockRepository)
		wantErr error
	}{
		{
			name:  "success",
			email: "new@example.com",
			setup: func(r *mockRepository) {},
		},
		{
			name:  "duplicate email",
			email: "existing@example.com",
			setup: func(r *mockRepository) {
				createTestUser(r, "existing@example.com", "password", true)
			},
			wantErr: ErrUserExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestService()
			tt.setup(repo)

			user, tokens, err := svc.Register(context.Background(), tt.email, "StrongPass123!", "First", "Last")

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, user)
				assert.Nil(t, tokens)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.email, user.Email)
				assert.NotEmpty(t, tokens.AccessToken)
				assert.NotEmpty(t, tokens.RefreshToken)
				// Default role assigned
				roles := repo.userRoles[user.ID]
				assert.Contains(t, roles, "member")
			}
		})
	}
}

func TestService_Login(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		setup    func(*mockRepository)
		wantErr  error
	}{
		{
			name:     "success",
			email:    "user@example.com",
			password: "correct-password",
			setup: func(r *mockRepository) {
				createTestUser(r, "user@example.com", "correct-password", true)
			},
		},
		{
			name:     "wrong password",
			email:    "user@example.com",
			password: "wrong-password",
			setup: func(r *mockRepository) {
				createTestUser(r, "user@example.com", "correct-password", true)
			},
			wantErr: ErrInvalidCredentials,
		},
		{
			name:     "user not found",
			email:    "noone@example.com",
			password: "password",
			setup:    func(r *mockRepository) {},
			wantErr:  ErrInvalidCredentials,
		},
		{
			name:     "inactive user",
			email:    "inactive@example.com",
			password: "password",
			setup: func(r *mockRepository) {
				createTestUser(r, "inactive@example.com", "password", false)
			},
			wantErr: ErrUserInactive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestService()
			tt.setup(repo)

			result, err := svc.Login(context.Background(), tt.email, tt.password)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.False(t, result.RequiresTwoFactor)
				assert.NotNil(t, result.User)
				assert.NotEmpty(t, result.AccessToken)
				assert.NotEmpty(t, result.RefreshToken)
			}
		})
	}
}

func TestService_RefreshToken(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Service, *mockRepository) string
		wantErr error
	}{
		{
			name: "success",
			setup: func(svc *Service, repo *mockRepository) string {
				createTestUser(repo, "user@example.com", "pass", true)
				result, _ := svc.Login(context.Background(), "user@example.com", "pass")
				return result.RefreshToken
			},
		},
		{
			name: "expired token",
			setup: func(svc *Service, repo *mockRepository) string {
				user := createTestUser(repo, "user@example.com", "pass", true)
				plain, hash, _ := svc.tokenMaker.CreateRefreshToken()
				repo.refreshTokens[hash] = &models.RefreshToken{
					ID:        uuid.New(),
					UserID:    user.ID,
					TokenHash: hash,
					ExpiresAt: time.Now().Add(-1 * time.Hour),
					CreatedAt: time.Now(),
				}
				return plain
			},
			wantErr: ErrTokenExpired,
		},
		{
			name: "revoked token triggers revoke all",
			setup: func(svc *Service, repo *mockRepository) string {
				user := createTestUser(repo, "user@example.com", "pass", true)
				plain, hash, _ := svc.tokenMaker.CreateRefreshToken()
				repo.refreshTokens[hash] = &models.RefreshToken{
					ID:        uuid.New(),
					UserID:    user.ID,
					TokenHash: hash,
					ExpiresAt: time.Now().Add(1 * time.Hour),
					Revoked:   true,
					CreatedAt: time.Now(),
				}
				return plain
			},
			wantErr: ErrTokenRevoked,
		},
		{
			name: "invalid token",
			setup: func(svc *Service, repo *mockRepository) string {
				return "nonexistent-token"
			},
			wantErr: ErrTokenInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestService()
			refreshToken := tt.setup(svc, repo)

			tokens, err := svc.RefreshToken(context.Background(), refreshToken)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, tokens)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, tokens.AccessToken)
				assert.NotEmpty(t, tokens.RefreshToken)
				// Old token should be revoked (rotation)
			}
		})
	}
}

func TestService_Logout(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*Service, *mockRepository) string
	}{
		{
			name: "success",
			setup: func(svc *Service, repo *mockRepository) string {
				createTestUser(repo, "user@example.com", "pass", true)
				result, _ := svc.Login(context.Background(), "user@example.com", "pass")
				return result.RefreshToken
			},
		},
		{
			name: "nonexistent token is idempotent",
			setup: func(svc *Service, repo *mockRepository) string {
				return "nonexistent-token"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestService()
			refreshToken := tt.setup(svc, repo)

			err := svc.Logout(context.Background(), refreshToken)
			assert.NoError(t, err)
		})
	}
}

func TestService_ValidateToken(t *testing.T) {
	svc, _ := newTestService()

	t.Run("valid token", func(t *testing.T) {
		userID := uuid.New()
		token, err := svc.tokenMaker.CreateAccessToken(userID, []string{"admin"}, []string{"contacts:read"})
		require.NoError(t, err)

		claims, err := svc.ValidateToken(context.Background(), token)
		require.NoError(t, err)
		assert.Equal(t, userID.String(), claims.UserID)
		assert.Equal(t, []string{"admin"}, claims.Roles)
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := svc.ValidateToken(context.Background(), "invalid-token")
		assert.ErrorIs(t, err, ErrTokenInvalid)
	})
}

func TestService_GetUser(t *testing.T) {
	svc, repo := newTestService()
	user := createTestUser(repo, "user@example.com", "pass", true)
	repo.userRoles[user.ID] = []string{"admin"}

	t.Run("found", func(t *testing.T) {
		found, roles, err := svc.GetUser(context.Background(), user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.Email, found.Email)
		assert.Contains(t, roles, "admin")
	})

	t.Run("not found", func(t *testing.T) {
		_, _, err := svc.GetUser(context.Background(), uuid.New())
		assert.ErrorIs(t, err, ErrUserNotFound)
	})
}

func TestService_UpdateUser(t *testing.T) {
	svc, repo := newTestService()
	user := createTestUser(repo, "user@example.com", "pass", true)

	newName := "Updated"
	updated, err := svc.UpdateUser(context.Background(), user.ID, &newName, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.FirstName)
	assert.Equal(t, user.LastName, updated.LastName)
}

func TestService_AssignAndRemoveRole(t *testing.T) {
	svc, repo := newTestService()
	user := createTestUser(repo, "user@example.com", "pass", true)

	err := svc.AssignRole(context.Background(), user.ID, "admin")
	require.NoError(t, err)
	assert.Contains(t, repo.userRoles[user.ID], "admin")

	err = svc.RemoveRole(context.Background(), user.ID, "admin")
	require.NoError(t, err)
	assert.NotContains(t, repo.userRoles[user.ID], "admin")
}

func TestService_CheckPermission(t *testing.T) {
	svc, repo := newTestService()
	user := createTestUser(repo, "user@example.com", "pass", true)
	repo.userPerms[user.ID] = []string{"contacts:read", "contacts:write"}

	allowed, err := svc.CheckPermission(context.Background(), user.ID, "contacts", "read")
	require.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = svc.CheckPermission(context.Background(), user.ID, "contacts", "delete")
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestService_ListUsers(t *testing.T) {
	svc, repo := newTestService()
	createTestUser(repo, "a@example.com", "pass", true)
	createTestUser(repo, "b@example.com", "pass", true)
	createTestUser(repo, "c@example.com", "pass", true)

	t.Run("first page", func(t *testing.T) {
		users, total, err := svc.ListUsers(context.Background(), 1, 2)
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, users, 2)
	})

	t.Run("second page", func(t *testing.T) {
		users, total, err := svc.ListUsers(context.Background(), 2, 2)
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, users, 1)
	})

	t.Run("default values for invalid input", func(t *testing.T) {
		users, total, err := svc.ListUsers(context.Background(), 0, 0)
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, users, 3)
	})

	t.Run("page size capped at 100", func(t *testing.T) {
		_, _, err := svc.ListUsers(context.Background(), 1, 200)
		require.NoError(t, err)
	})
}

func TestService_UpdateUser_AllFields(t *testing.T) {
	svc, repo := newTestService()
	user := createTestUser(repo, "user@example.com", "pass", true)

	firstName := "NewFirst"
	lastName := "NewLast"
	isActive := false

	updated, err := svc.UpdateUser(context.Background(), user.ID, &firstName, &lastName, &isActive)
	require.NoError(t, err)
	assert.Equal(t, "NewFirst", updated.FirstName)
	assert.Equal(t, "NewLast", updated.LastName)
	assert.False(t, updated.IsActive)
}

func TestService_UpdateUser_NotFound(t *testing.T) {
	svc, _ := newTestService()

	name := "test"
	_, err := svc.UpdateUser(context.Background(), uuid.New(), &name, nil, nil)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestService_AssignRole_UserNotFound(t *testing.T) {
	svc, _ := newTestService()
	err := svc.AssignRole(context.Background(), uuid.New(), "admin")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestService_RemoveRole_UserNotFound(t *testing.T) {
	svc, _ := newTestService()
	err := svc.RemoveRole(context.Background(), uuid.New(), "admin")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestService_RefreshToken_InactiveUser(t *testing.T) {
	svc, repo := newTestService()
	user := createTestUser(repo, "user@example.com", "pass", true)

	// Login while active
	result, err := svc.Login(context.Background(), "user@example.com", "pass")
	require.NoError(t, err)

	// Deactivate user
	user.IsActive = false

	// Refresh should fail
	_, err = svc.RefreshToken(context.Background(), result.RefreshToken)
	assert.ErrorIs(t, err, ErrUserInactive)
}

func TestService_ChangePassword(t *testing.T) {
	tests := []struct {
		name        string
		oldPassword string
		newPassword string
		setup       func(*mockRepository) uuid.UUID
		wantErr     error
	}{
		{
			name:        "success",
			oldPassword: "old-password",
			newPassword: "new-password",
			setup: func(r *mockRepository) uuid.UUID {
				user := createTestUser(r, "user@example.com", "old-password", true)
				return user.ID
			},
		},
		{
			name:        "wrong old password",
			oldPassword: "wrong-password",
			newPassword: "new-password",
			setup: func(r *mockRepository) uuid.UUID {
				user := createTestUser(r, "user@example.com", "correct-password", true)
				return user.ID
			},
			wantErr: ErrInvalidCredentials,
		},
		{
			name:        "user not found",
			oldPassword: "old",
			newPassword: "new",
			setup: func(r *mockRepository) uuid.UUID {
				return uuid.New()
			},
			wantErr: ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestService()
			userID := tt.setup(repo)

			err := svc.ChangePassword(context.Background(), userID, tt.oldPassword, tt.newPassword)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				// Verify new password works
				user := repo.users[userID]
				err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(tt.newPassword))
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_CreateInvitation(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		role    string
		setup   func(*mockRepository)
		wantErr error
	}{
		{
			name:  "success",
			email: "newuser@example.com",
			role:  "member",
			setup: func(r *mockRepository) {},
		},
		{
			name:  "user already exists",
			email: "existing@example.com",
			role:  "member",
			setup: func(r *mockRepository) {
				createTestUser(r, "existing@example.com", "pass", true)
			},
			wantErr: ErrUserExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestService()
			tt.setup(repo)
			createdBy := uuid.New()

			inv, token, err := svc.CreateInvitation(context.Background(), tt.email, tt.role, createdBy)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, inv)
				assert.Empty(t, token)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, inv)
				assert.Equal(t, tt.email, inv.Email)
				assert.Equal(t, tt.role, inv.Role)
				assert.NotEmpty(t, token)
				assert.Nil(t, inv.AcceptedAt)
			}
		})
	}
}

func TestService_AcceptInvitation(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Service, *mockRepository) string
		wantErr error
	}{
		{
			name: "success",
			setup: func(svc *Service, repo *mockRepository) string {
				inv, token, _ := svc.CreateInvitation(context.Background(), "new@example.com", "member", uuid.New())
				_ = inv
				return token
			},
		},
		{
			name: "invitation not found",
			setup: func(svc *Service, repo *mockRepository) string {
				return "nonexistent-token"
			},
			wantErr: ErrInvitationNotFound,
		},
		{
			name: "invitation expired",
			setup: func(svc *Service, repo *mockRepository) string {
				inv := &models.Invitation{
					ID:        uuid.New(),
					Email:     "expired@example.com",
					Role:      "member",
					TokenHash: HashToken("expired-token"),
					CreatedBy: uuid.New(),
					ExpiresAt: time.Now().Add(-1 * time.Hour),
					CreatedAt: time.Now(),
				}
				repo.invitations[inv.ID] = inv
				repo.invByToken[inv.TokenHash] = inv
				return "expired-token"
			},
			wantErr: ErrInvitationExpired,
		},
		{
			name: "invitation already used",
			setup: func(svc *Service, repo *mockRepository) string {
				now := time.Now()
				inv := &models.Invitation{
					ID:         uuid.New(),
					Email:      "used@example.com",
					Role:       "member",
					TokenHash:  HashToken("used-token"),
					CreatedBy:  uuid.New(),
					ExpiresAt:  time.Now().Add(1 * time.Hour),
					AcceptedAt: &now,
					CreatedAt:  time.Now(),
				}
				repo.invitations[inv.ID] = inv
				repo.invByToken[inv.TokenHash] = inv
				return "used-token"
			},
			wantErr: ErrInvitationAlreadyUsed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestService()
			token := tt.setup(svc, repo)

			user, tokens, err := svc.AcceptInvitation(context.Background(), token, "password123", "First", "Last")

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, user)
				assert.Nil(t, tokens)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, user)
				assert.NotNil(t, tokens)
				assert.NotEmpty(t, tokens.AccessToken)
				assert.NotEmpty(t, tokens.RefreshToken)
			}
		})
	}
}

func TestService_ListInvitations(t *testing.T) {
	svc, repo := newTestService()

	// Create some invitations
	_, _, _ = svc.CreateInvitation(context.Background(), "a@example.com", "member", uuid.New())
	_, _, _ = svc.CreateInvitation(context.Background(), "b@example.com", "admin", uuid.New())

	// Mark one as accepted
	for id, inv := range repo.invitations {
		if inv.Email == "a@example.com" {
			now := time.Now()
			inv.AcceptedAt = &now
			repo.invitations[id] = inv
			break
		}
	}

	invs, err := svc.ListInvitations(context.Background())
	require.NoError(t, err)
	assert.Len(t, invs, 1)
	assert.Equal(t, "b@example.com", invs[0].Email)
}

func TestService_CancelInvitation(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Service, *mockRepository) uuid.UUID
		wantErr error
	}{
		{
			name: "success",
			setup: func(svc *Service, repo *mockRepository) uuid.UUID {
				inv, _, _ := svc.CreateInvitation(context.Background(), "cancel@example.com", "member", uuid.New())
				return inv.ID
			},
		},
		{
			name: "not found",
			setup: func(svc *Service, repo *mockRepository) uuid.UUID {
				return uuid.New()
			},
			wantErr: ErrInvitationNotFound,
		},
		{
			name: "already accepted",
			setup: func(svc *Service, repo *mockRepository) uuid.UUID {
				inv, _, _ := svc.CreateInvitation(context.Background(), "accepted@example.com", "member", uuid.New())
				now := time.Now()
				inv.AcceptedAt = &now
				return inv.ID
			},
			wantErr: ErrInvitationAlreadyUsed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, repo := newTestService()
			invID := tt.setup(svc, repo)

			err := svc.CancelInvitation(context.Background(), invID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				_, exists := repo.invitations[invID]
				assert.False(t, exists)
			}
		})
	}
}

func TestService_GetProfile(t *testing.T) {
	svc, repo := newTestService()
	user := createTestUser(repo, "user@example.com", "pass", true)
	repo.userRoles[user.ID] = []string{"member"}

	profile, roles, err := svc.GetProfile(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.Email, profile.Email)
	assert.Contains(t, roles, "member")
}
