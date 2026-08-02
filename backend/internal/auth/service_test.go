package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/kmuhub/kmuhub/internal/models"
)

// testInviteTenant is deliberately not models.DefaultTenantID: an account
// created from an invitation must land in the inviting tenant, and only a
// non-default tenant can tell the two apart.
var testInviteTenant = uuid.MustParse("a1a1a1a1-0000-4000-8000-000000000001")

// mockRepository implements Repository for testing
type mockRepository struct {
	users               map[uuid.UUID]*models.User
	usersByEmail        map[string]*models.User
	refreshTokens       map[string]*models.RefreshToken // keyed by token_hash
	userRoles           map[uuid.UUID][]string
	userPerms           map[uuid.UUID][]string
	effectiveGrants     map[uuid.UUID][]EffectiveGrantRow
	invitations         map[uuid.UUID]*models.Invitation
	invByToken          map[string]*models.Invitation
	sessions            []*models.UserSession
	recoveryCodes       []*models.RecoveryCode
	passwordResetTokens map[string]*models.PasswordResetToken // keyed by token_hash
	seatLimits          map[uuid.UUID]*int                    // tenant → booked seats, absent = unlimited

	provisionedTenants map[uuid.UUID]*models.Tenant
	provisionedModules map[uuid.UUID][]string
	provisionFails     error // set to make ProvisionTenant fail

	rotatedFrom       []uuid.UUID // refresh token ids RotateSessionRefreshToken was asked about
	prunedFor         []uuid.UUID // user ids DeleteStaleUserSessions was called for
	sessionWriteFails bool        // make CreateSession fail, to prove a login survives it
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		users:               make(map[uuid.UUID]*models.User),
		usersByEmail:        make(map[string]*models.User),
		refreshTokens:       make(map[string]*models.RefreshToken),
		userRoles:           make(map[uuid.UUID][]string),
		userPerms:           make(map[uuid.UUID][]string),
		effectiveGrants:     make(map[uuid.UUID][]EffectiveGrantRow),
		invitations:         make(map[uuid.UUID]*models.Invitation),
		invByToken:          make(map[string]*models.Invitation),
		sessions:            nil,
		recoveryCodes:       nil,
		passwordResetTokens: make(map[string]*models.PasswordResetToken),
		seatLimits:          make(map[uuid.UUID]*int),
		provisionedTenants:  make(map[uuid.UUID]*models.Tenant),
		provisionedModules:  make(map[uuid.UUID][]string),
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

func (m *mockRepository) UpdateProfile(_ context.Context, user *models.User) error {
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

func (m *mockRepository) GetEffectivePermissions(_ context.Context, userID uuid.UUID) ([]EffectiveGrantRow, error) {
	return m.effectiveGrants[userID], nil
}

func (m *mockRepository) ListRoles(_ context.Context) ([]Role, error) {
	return nil, nil
}

// Role administration is exercised against the real database
// (roles_admin_db_test.go); these only satisfy the interface.
func (m *mockRepository) CountCustomRoles(_ context.Context) (int, error) {
	return 0, nil
}

func (m *mockRepository) RoleNameExists(_ context.Context, _ string, _ uuid.UUID) (bool, error) {
	return false, nil
}

func (m *mockRepository) CreateRole(_ context.Context, _ uuid.UUID, _ CreateRoleInput) (*Role, error) {
	return nil, nil
}

func (m *mockRepository) GetRoleByID(_ context.Context, _ uuid.UUID) (*Role, error) {
	return nil, nil
}

func (m *mockRepository) UpdateRole(_ context.Context, _ uuid.UUID, _ UpdateRoleInput) (*Role, error) {
	return nil, nil
}

func (m *mockRepository) RoleHasMembers(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil
}

func (m *mockRepository) DeleteRole(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockRepository) GetRolePermissions(_ context.Context, _ uuid.UUID) ([]RoleGrant, error) {
	return nil, nil
}

func (m *mockRepository) SetRolePermissions(_ context.Context, _ uuid.UUID, _ []RoleGrant) ([]RoleGrant, error) {
	return nil, nil
}

func (m *mockRepository) AssignUserRole(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

func (m *mockRepository) RevokeUserRole(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

func (m *mockRepository) GetUserRoleIDs(_ context.Context, _ uuid.UUID) ([]string, error) {
	return nil, nil
}

func (m *mockRepository) GetUserGrants(_ context.Context, _ uuid.UUID) ([]EffectiveGrantRow, error) {
	return nil, nil
}

func (m *mockRepository) CountRoleAdminsExcluding(_ context.Context, _ []string, _, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockRepository) CountUnknownPermissionKeys(_ context.Context, _ []string) (int, error) {
	return 0, nil
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

func (m *mockRepository) GetPendingInvitationByEmail(_ context.Context, email string) (*models.Invitation, error) {
	for _, inv := range m.invitations {
		if inv.Email == email && inv.AcceptedAt == nil && inv.ExpiresAt.After(time.Now()) {
			return inv, nil
		}
	}
	return nil, ErrInvitationNotFound
}

// ProvisionTenant records the three writes the real repository does in one
// transaction. provisionedModules doubles as the assertion target for the
// catalogue resolution.
func (m *mockRepository) ProvisionTenant(_ context.Context, tenant *models.Tenant, moduleIDs []string, inv *models.Invitation) error {
	if m.provisionFails != nil {
		return m.provisionFails
	}
	m.provisionedTenants[tenant.ID] = tenant
	m.provisionedModules[tenant.ID] = moduleIDs
	m.invitations[inv.ID] = inv
	m.invByToken[inv.TokenHash] = inv
	return nil
}

func (m *mockRepository) GetInvitationByID(_ context.Context, tenantID, id uuid.UUID) (*models.Invitation, error) {
	inv, ok := m.invitations[id]
	if !ok || inv.TenantID != tenantID {
		return nil, ErrInvitationNotFound
	}
	return inv, nil
}

func (m *mockRepository) ListPendingInvitations(_ context.Context, tenantID uuid.UUID) ([]*models.Invitation, error) {
	pending := []*models.Invitation{}
	for _, inv := range m.invitations {
		if inv.AcceptedAt == nil && inv.TenantID == tenantID {
			pending = append(pending, inv)
		}
	}
	return pending, nil
}

// AcceptInvitation mirrors the repository's transaction: the claim is
// conditional on the invitation still being pending, and nothing is written
// when it is not.
func (m *mockRepository) AcceptInvitation(ctx context.Context, inv *models.Invitation, user *models.User) error {
	stored, ok := m.invitations[inv.ID]
	if !ok {
		return ErrInvitationNotFound
	}
	if stored.AcceptedAt != nil {
		return ErrInvitationAlreadyUsed
	}
	now := time.Now()
	stored.AcceptedAt = &now
	if err := m.CreateUser(ctx, user); err != nil {
		return err
	}
	return m.AssignRole(ctx, user.ID, inv.Role)
}

// CountSeatsInUse counts the same two populations as the SQL: active users of
// the tenant plus its pending, unexpired invitations.
func (m *mockRepository) CountSeatsInUse(_ context.Context, tenantID uuid.UUID, excludeInvitation *uuid.UUID) (int, *int, error) {
	used := 0
	for _, u := range m.users {
		if u.TenantID == tenantID && u.IsActive {
			used++
		}
	}
	for _, inv := range m.invitations {
		if inv.TenantID != tenantID || inv.AcceptedAt != nil || !inv.ExpiresAt.After(time.Now()) {
			continue
		}
		if excludeInvitation != nil && inv.ID == *excludeInvitation {
			continue
		}
		used++
	}
	return used, m.seatLimits[tenantID], nil
}

func (m *mockRepository) DeleteInvitation(_ context.Context, tenantID, id uuid.UUID) error {
	inv, ok := m.invitations[id]
	if !ok || inv.TenantID != tenantID {
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
	m.recoveryCodes = append(m.recoveryCodes, recoveryCodes...)
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

func (m *mockRepository) CreateSession(_ context.Context, session *models.UserSession) error {
	if m.sessionWriteFails {
		return errors.New("session write failed")
	}
	m.sessions = append(m.sessions, session)
	return nil
}

func (m *mockRepository) GetSession(_ context.Context, id uuid.UUID) (*models.UserSession, error) {
	for _, s := range m.sessions {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, ErrSessionNotFound
}

func (m *mockRepository) ListUserSessions(_ context.Context, userID uuid.UUID) ([]*models.UserSession, error) {
	var out []*models.UserSession
	for _, s := range m.sessions {
		if s.UserID == userID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (m *mockRepository) ListAllSessions(_ context.Context, _, _ int) ([]*models.UserSession, int, error) {
	return nil, 0, nil
}

func (m *mockRepository) UpdateSessionActivity(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockRepository) DeleteSession(_ context.Context, id uuid.UUID) error {
	kept := m.sessions[:0]
	for _, s := range m.sessions {
		if s.ID != id {
			kept = append(kept, s)
		}
	}
	m.sessions = kept
	return nil
}

func (m *mockRepository) DeleteAllUserSessions(_ context.Context, userID uuid.UUID, exceptSessionID *uuid.UUID) error {
	kept := m.sessions[:0]
	for _, s := range m.sessions {
		if s.UserID != userID || (exceptSessionID != nil && s.ID == *exceptSessionID) {
			kept = append(kept, s)
		}
	}
	m.sessions = kept
	return nil
}

func (m *mockRepository) RotateSessionRefreshToken(_ context.Context, oldTokenID, newTokenID uuid.UUID, ipAddress, userAgent string) (bool, error) {
	m.rotatedFrom = append(m.rotatedFrom, oldTokenID)
	for _, s := range m.sessions {
		if s.RefreshTokenID != nil && *s.RefreshTokenID == oldTokenID {
			s.RefreshTokenID = &newTokenID
			if ipAddress != "" {
				s.IPAddress = ipAddress
			}
			if userAgent != "" {
				s.UserAgent = userAgent
			}
			return true, nil
		}
	}
	return false, nil
}

func (m *mockRepository) DeleteSessionByRefreshTokenID(_ context.Context, refreshTokenID uuid.UUID) error {
	kept := m.sessions[:0]
	for _, s := range m.sessions {
		if s.RefreshTokenID == nil || *s.RefreshTokenID != refreshTokenID {
			kept = append(kept, s)
		}
	}
	m.sessions = kept
	return nil
}

func (m *mockRepository) DeleteStaleUserSessions(_ context.Context, userID uuid.UUID) error {
	m.prunedFor = append(m.prunedFor, userID)
	return nil
}

// Password reset token methods

func (m *mockRepository) CreatePasswordResetToken(_ context.Context, token *models.PasswordResetToken) error {
	m.passwordResetTokens[token.TokenHash] = token
	return nil
}

func (m *mockRepository) GetPasswordResetToken(_ context.Context, tokenHash string) (*models.PasswordResetToken, error) {
	t, ok := m.passwordResetTokens[tokenHash]
	if !ok {
		return nil, ErrResetTokenInvalid
	}
	return t, nil
}

func (m *mockRepository) MarkPasswordResetTokenUsed(_ context.Context, id uuid.UUID) error {
	for _, t := range m.passwordResetTokens {
		if t.ID == id {
			now := time.Now()
			t.UsedAt = &now
			return nil
		}
	}
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
				// Default tenant assigned (sprint-4 welle-0.5: prevent uuid.Nil tenant)
				assert.Equal(t, models.DefaultTenantID, user.TenantID,
					"Register must set TenantID to models.DefaultTenantID, got %s", user.TenantID)
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
		token, err := svc.tokenMaker.CreateAccessToken(userID, uuid.New().String(), []string{"admin"}, []string{"contacts:read"}, nil)
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

			inv, token, err := svc.CreateInvitation(context.Background(), testInviteTenant, tt.email, tt.role, createdBy)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, inv)
				assert.Empty(t, token)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, inv)
				assert.Equal(t, testInviteTenant, inv.TenantID)
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
				inv, token, _ := svc.CreateInvitation(context.Background(), testInviteTenant, "new@example.com", "member", uuid.New())
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
					TenantID:  testInviteTenant,
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
					TenantID:   testInviteTenant,
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
				// The account belongs to the tenant that invited it, not to the
				// default tenant (migration 000249).
				assert.Equal(t, testInviteTenant, user.TenantID,
					"AcceptInvitation must take the tenant from the invitation, got %s", user.TenantID)
			}
		})
	}
}

func TestService_ListInvitations(t *testing.T) {
	svc, repo := newTestService()

	// Create some invitations
	_, _, _ = svc.CreateInvitation(context.Background(), testInviteTenant, "a@example.com", "member", uuid.New())
	_, _, _ = svc.CreateInvitation(context.Background(), testInviteTenant, "b@example.com", "admin", uuid.New())

	// Mark one as accepted
	for id, inv := range repo.invitations {
		if inv.Email == "a@example.com" {
			now := time.Now()
			inv.AcceptedAt = &now
			repo.invitations[id] = inv
			break
		}
	}

	invs, err := svc.ListInvitations(context.Background(), testInviteTenant)
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
				inv, _, _ := svc.CreateInvitation(context.Background(), testInviteTenant, "cancel@example.com", "member", uuid.New())
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
				inv, _, _ := svc.CreateInvitation(context.Background(), testInviteTenant, "accepted@example.com", "member", uuid.New())
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

			err := svc.CancelInvitation(context.Background(), testInviteTenant, invID)

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

// TestService_CreateSession_TenantID verifies that CreateSession propagates TenantID
// from the caller into the persisted UserSession entity (wiring-gap closure for
// user_sessions.tenant_id NOT NULL — Sprint 4 Welle 1b Stream A).
func TestService_CreateSession_TenantID(t *testing.T) {
	svc, repo := newTestService()
	tenantID := uuid.New()
	userID := uuid.New()
	refreshTokenID := uuid.New()

	session, err := svc.CreateSession(
		context.Background(),
		userID,
		tenantID,
		"127.0.0.1",
		"Mozilla/5.0 (Windows NT 10.0) AppleWebKit/537.36 Chrome/120 Safari/537.36",
		refreshTokenID,
	)
	require.NoError(t, err)
	assert.Equal(t, tenantID, session.TenantID,
		"CreateSession must propagate tenantID into UserSession.TenantID")
	require.Len(t, repo.sessions, 1)
	assert.Equal(t, tenantID, repo.sessions[0].TenantID,
		"persisted UserSession must carry tenant_id (not uuid.Nil)")
}

// TestService_RecoveryCode_TenantID verifies that recovery codes generated during
// 2FA setup carry the user's TenantID (wiring-gap closure for
// recovery_codes.tenant_id NOT NULL — Sprint 4 Welle 1b Stream A).
func TestService_RecoveryCode_TenantID(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	codes, hashed, err := generateRecoveryCodes(userID, tenantID)
	require.NoError(t, err)
	assert.Len(t, codes, recoveryCodeCount)
	require.Len(t, hashed, recoveryCodeCount)
	for i, rc := range hashed {
		assert.Equal(t, tenantID, rc.TenantID,
			"RecoveryCode[%d] must have TenantID set (not uuid.Nil)", i)
		assert.Equal(t, userID, rc.UserID,
			"RecoveryCode[%d] must have correct UserID", i)
		assert.NotEmpty(t, rc.CodeHash,
			"RecoveryCode[%d] must have non-empty CodeHash", i)
	}
}

// TestService_AcceptInvitation_IsSingleUse redeems the same token twice. The
// second attempt must be rejected and must not leave a second account behind:
// before the claim moved into the repository transaction, two accepts could
// both pass the AcceptedAt check and create two users from one invitation.
func TestService_AcceptInvitation_IsSingleUse(t *testing.T) {
	svc, repo := newTestService()

	_, token, err := svc.CreateInvitation(context.Background(), testInviteTenant, "single@example.com", "member", uuid.New())
	require.NoError(t, err)

	user, _, err := svc.AcceptInvitation(context.Background(), token, "password123", "First", "Last")
	require.NoError(t, err)
	require.NotNil(t, user)

	_, _, err = svc.AcceptInvitation(context.Background(), token, "password123", "Second", "Try")
	assert.ErrorIs(t, err, ErrInvitationAlreadyUsed)
	assert.Len(t, repo.users, 1, "a redeemed invitation must not create a second account")
}

// TestService_CancelInvitation_ForeignTenant guards the read side: cancelling
// takes a tenant, and an invitation of another tenant must be invisible rather
// than deletable.
func TestService_CancelInvitation_ForeignTenant(t *testing.T) {
	svc, _ := newTestService()

	inv, _, err := svc.CreateInvitation(context.Background(), testInviteTenant, "foreign@example.com", "member", uuid.New())
	require.NoError(t, err)

	err = svc.CancelInvitation(context.Background(), uuid.New(), inv.ID)
	assert.ErrorIs(t, err, ErrInvitationNotFound)
}

// TestService_ListInvitations_ScopedToTenant verifies that the pending list is
// per tenant — the pre-000249 list was global.
func TestService_ListInvitations_ScopedToTenant(t *testing.T) {
	svc, _ := newTestService()
	otherTenant := uuid.New()

	_, _, err := svc.CreateInvitation(context.Background(), testInviteTenant, "mine@example.com", "member", uuid.New())
	require.NoError(t, err)
	_, _, err = svc.CreateInvitation(context.Background(), otherTenant, "theirs@example.com", "member", uuid.New())
	require.NoError(t, err)

	invs, err := svc.ListInvitations(context.Background(), testInviteTenant)
	require.NoError(t, err)
	require.Len(t, invs, 1)
	assert.Equal(t, "mine@example.com", invs[0].Email)
}

func TestService_CreateInvitation_SeatLimit(t *testing.T) {
	limit := func(n int) *int { return &n }

	t.Run("pending invitation occupies a seat", func(t *testing.T) {
		svc, repo := newTestService()
		repo.seatLimits[testInviteTenant] = limit(1)

		_, _, err := svc.CreateInvitation(context.Background(), testInviteTenant, "first@example.com", "member", uuid.New())
		require.NoError(t, err)

		_, _, err = svc.CreateInvitation(context.Background(), testInviteTenant, "second@example.com", "member", uuid.New())
		assert.ErrorIs(t, err, ErrSeatLimitReached,
			"the pending invitation holds the only seat")
	})

	t.Run("active user occupies a seat", func(t *testing.T) {
		svc, repo := newTestService()
		repo.seatLimits[testInviteTenant] = limit(1)
		occupant := createTestUser(repo, "active@example.com", "pass", true)
		occupant.TenantID = testInviteTenant

		_, _, err := svc.CreateInvitation(context.Background(), testInviteTenant, "next@example.com", "member", uuid.New())
		assert.ErrorIs(t, err, ErrSeatLimitReached)
	})

	t.Run("another tenant's seats do not count", func(t *testing.T) {
		svc, repo := newTestService()
		repo.seatLimits[testInviteTenant] = limit(1)
		occupant := createTestUser(repo, "elsewhere@example.com", "pass", true)
		occupant.TenantID = uuid.New()

		_, _, err := svc.CreateInvitation(context.Background(), testInviteTenant, "ours@example.com", "member", uuid.New())
		require.NoError(t, err)
	})

	t.Run("no limit booked means unlimited", func(t *testing.T) {
		svc, repo := newTestService()
		for i := 0; i < 3; i++ {
			occupant := createTestUser(repo, "u"+string(rune('a'+i))+"@example.com", "pass", true)
			occupant.TenantID = testInviteTenant
		}

		_, _, err := svc.CreateInvitation(context.Background(), testInviteTenant, "more@example.com", "member", uuid.New())
		require.NoError(t, err)
	})
}

// TestService_AcceptInvitation_SeatWithdrawn covers the downgrade case: the
// invitation was issued while a seat was free, the plan shrank in the meantime,
// and the account must not be created past the limit.
func TestService_AcceptInvitation_SeatWithdrawn(t *testing.T) {
	svc, repo := newTestService()

	_, token, err := svc.CreateInvitation(context.Background(), testInviteTenant, "late@example.com", "member", uuid.New())
	require.NoError(t, err)

	occupant := createTestUser(repo, "incumbent@example.com", "pass", true)
	occupant.TenantID = testInviteTenant
	seats := 1
	repo.seatLimits[testInviteTenant] = &seats

	_, _, err = svc.AcceptInvitation(context.Background(), token, "password123", "Late", "Comer")
	assert.ErrorIs(t, err, ErrSeatLimitReached)
}
