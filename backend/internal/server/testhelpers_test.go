package server

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/chat/file"
	"github.com/kmuhub/kmuhub/internal/middleware"
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

	provisionedTenants map[uuid.UUID]*models.Tenant
	provisionedModules map[uuid.UUID][]string
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

		provisionedTenants: make(map[uuid.UUID]*models.Tenant),
		provisionedModules: make(map[uuid.UUID][]string),
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

func (m *authMockRepo) UpdateProfile(_ context.Context, user *models.User) error {
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

func (m *authMockRepo) GetEffectivePermissions(_ context.Context, _ uuid.UUID) ([]auth.EffectiveGrantRow, error) {
	return nil, nil
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

func (m *authMockRepo) GetPendingInvitationByEmail(_ context.Context, email string) (*models.Invitation, error) {
	for _, inv := range m.invitations {
		if inv.Email == email && inv.AcceptedAt == nil && inv.ExpiresAt.After(time.Now()) {
			return inv, nil
		}
	}
	return nil, auth.ErrInvitationNotFound
}

func (m *authMockRepo) ProvisionTenant(_ context.Context, tenant *models.Tenant, moduleIDs []string, inv *models.Invitation) error {
	m.provisionedTenants[tenant.ID] = tenant
	m.provisionedModules[tenant.ID] = moduleIDs
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

func (m *authMockRepo) GetInvitationByID(_ context.Context, tenantID, id uuid.UUID) (*models.Invitation, error) {
	inv, ok := m.invitations[id]
	if !ok || inv.TenantID != tenantID {
		return nil, auth.ErrInvitationNotFound
	}
	return inv, nil
}

func (m *authMockRepo) ListPendingInvitations(_ context.Context, tenantID uuid.UUID) ([]*models.Invitation, error) {
	pending := []*models.Invitation{}
	for _, inv := range m.invitations {
		if inv.AcceptedAt == nil && inv.TenantID == tenantID {
			pending = append(pending, inv)
		}
	}
	return pending, nil
}

func (m *authMockRepo) AcceptInvitation(ctx context.Context, inv *models.Invitation, user *models.User) error {
	stored, ok := m.invitations[inv.ID]
	if !ok {
		return auth.ErrInvitationNotFound
	}
	if stored.AcceptedAt != nil {
		return auth.ErrInvitationAlreadyUsed
	}
	now := time.Now()
	stored.AcceptedAt = &now
	if err := m.CreateUser(ctx, user); err != nil {
		return err
	}
	return m.AssignRole(ctx, user.ID, inv.Role)
}

// CountSeatsInUse: the mock tenant has no seat cap booked (nil limit), so the
// invite path is never blocked here.
func (m *authMockRepo) CountSeatsInUse(_ context.Context, _ uuid.UUID, _ *uuid.UUID) (int, *int, error) {
	return 0, nil, nil
}

func (m *authMockRepo) DeleteInvitation(_ context.Context, tenantID, id uuid.UUID) error {
	inv, ok := m.invitations[id]
	if !ok || inv.TenantID != tenantID {
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

func (m *authMockRepo) CreatePasswordResetToken(_ context.Context, _ *models.PasswordResetToken) error {
	return nil
}

func (m *authMockRepo) GetPasswordResetToken(_ context.Context, _ string) (*models.PasswordResetToken, error) {
	return nil, auth.ErrResetTokenInvalid
}

func (m *authMockRepo) MarkPasswordResetTokenUsed(_ context.Context, _ uuid.UUID) error {
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

// ---------------------------------------------------------------------------
// File upload mock repository (implements file.Repository)
// ---------------------------------------------------------------------------

type fileMockRepo struct {
	members  map[string]bool    // "channelID:userID" → true
	archived map[uuid.UUID]bool // channelID → archived
	files    map[uuid.UUID]*models.ChatFile
	quota    *models.StorageQuota
	userInfo map[uuid.UUID][2]string       // userID → [firstName, lastName]
	roles    map[string]models.ChannelRole // "channelID:userID" → role
}

func newFileMockRepo() *fileMockRepo {
	return &fileMockRepo{
		members:  make(map[string]bool),
		archived: make(map[uuid.UUID]bool),
		files:    make(map[uuid.UUID]*models.ChatFile),
		quota: &models.StorageQuota{
			ID:        uuid.New(),
			MaxBytes:  1 << 30, // 1 GB
			UsedBytes: 0,
		},
		userInfo: make(map[uuid.UUID][2]string),
		roles:    make(map[string]models.ChannelRole),
	}
}

func (m *fileMockRepo) memberKey(channelID, userID uuid.UUID) string {
	return channelID.String() + ":" + userID.String()
}

func (m *fileMockRepo) IsChannelMember(_ context.Context, channelID, userID uuid.UUID) (bool, error) {
	return m.members[m.memberKey(channelID, userID)], nil
}

func (m *fileMockRepo) IsChannelArchived(_ context.Context, channelID uuid.UUID) (bool, error) {
	return m.archived[channelID], nil
}

func (m *fileMockRepo) GetStorageQuota(_ context.Context) (*models.StorageQuota, error) {
	return m.quota, nil
}

func (m *fileMockRepo) CreateFile(_ context.Context, f *models.ChatFile) error {
	m.files[f.ID] = f
	return nil
}

func (m *fileMockRepo) GetFileByID(_ context.Context, id uuid.UUID) (*models.ChatFile, error) {
	f, ok := m.files[id]
	if !ok {
		return nil, file.ErrFileNotFound
	}
	return f, nil
}

func (m *fileMockRepo) ListChannelFiles(_ context.Context, _ uuid.UUID, _, _ int) ([]*models.ChatFileWithUploader, int, error) {
	return nil, 0, nil
}

func (m *fileMockRepo) GetFilesByMessageIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID][]models.ChatFileWithUploader, error) {
	return nil, nil
}

func (m *fileMockRepo) DeleteFile(_ context.Context, id uuid.UUID) error {
	f, ok := m.files[id]
	if !ok {
		return file.ErrFileNotFound
	}
	f.IsDeleted = true
	now := time.Now()
	f.DeletedAt = &now
	return nil
}

func (m *fileMockRepo) IncrementUsedBytes(_ context.Context, bytes int64) error {
	m.quota.UsedBytes += bytes
	return nil
}

func (m *fileMockRepo) DecrementUsedBytes(_ context.Context, bytes int64) error {
	m.quota.UsedBytes -= bytes
	return nil
}

func (m *fileMockRepo) GetChannelRole(_ context.Context, channelID, userID uuid.UUID) (models.ChannelRole, error) {
	role, ok := m.roles[channelID.String()+":"+userID.String()]
	if !ok {
		return "", file.ErrNotChannelMember
	}
	return role, nil
}

func (m *fileMockRepo) GetUserInfo(_ context.Context, userID uuid.UUID) (string, string, error) {
	info, ok := m.userInfo[userID]
	if !ok {
		return "Test", "User", nil
	}
	return info[0], info[1], nil
}

// ---------------------------------------------------------------------------
// File upload mock store (implements file.FileStore)
// ---------------------------------------------------------------------------

type fileMockStore struct {
	errUpload error
}

func (m *fileMockStore) Upload(_ context.Context, _ string, _ io.Reader, _ int64, _ string) error {
	return m.errUpload
}

func (m *fileMockStore) Download(_ context.Context, _ string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), nil
}

func (m *fileMockStore) Delete(_ context.Context, _ string) error {
	return nil
}

func (m *fileMockStore) GetPresignedURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "https://example.com/presigned", nil
}

func (m *fileMockStore) GetPresignedUploadURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "https://example.com/presigned-upload", nil
}

// ---------------------------------------------------------------------------
// File upload mock scanner (implements file.FileScanner)
// ---------------------------------------------------------------------------

type fileMockScanner struct {
	scanErr error
}

func (m *fileMockScanner) Scan(_ context.Context, _ io.Reader, _ string) error {
	return m.scanErr
}

// ---------------------------------------------------------------------------
// File upload mock thumbnail generator (implements file.ThumbnailGenerator)
// ---------------------------------------------------------------------------

type fileMockThumbGen struct{}

func (m *fileMockThumbGen) CanGenerate(_ string) bool { return false }
func (m *fileMockThumbGen) Generate(_ context.Context, _ io.Reader, _ string) (io.Reader, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// File upload test helpers
// ---------------------------------------------------------------------------

func newTestFileUploadHandler() (*FileUploadHandler, *fileMockRepo, *fileMockStore, *fileMockScanner) {
	repo := newFileMockRepo()
	store := &fileMockStore{}
	scanner := &fileMockScanner{}
	thumbGen := &fileMockThumbGen{}
	svc := file.NewService(repo, store, scanner, thumbGen, 50)
	handler := NewFileUploadHandler(svc, nil)
	return handler, repo, store, scanner
}

func newMultipartUploadRequest(t *testing.T, userID, channelID, filename, mimeType string, fileContent []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if channelID != "" {
		require.NoError(t, writer.WriteField("channel_id", channelID))
	}

	if filename != "" {
		part, err := writer.CreateFormFile("file", filename)
		require.NoError(t, err)
		_, err = part.Write(fileContent)
		require.NoError(t, err)
	}

	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	if userID != "" {
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		// Stamp a tenant id alongside the user so service code that calls
		// middleware.GetTenantID(ctx) (Welle 1b file/upload wiring) succeeds.
		ctx = context.WithValue(ctx, middleware.TenantIDKey, uuid.New().String())
		req = req.WithContext(ctx)
	}

	return req
}

func newMultipartUploadRequestWithMessageID(t *testing.T, userID, channelID, messageID, filename, mimeType string, fileContent []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if channelID != "" {
		require.NoError(t, writer.WriteField("channel_id", channelID))
	}
	if messageID != "" {
		require.NoError(t, writer.WriteField("message_id", messageID))
	}

	if filename != "" {
		h := make(map[string][]string)
		h["Content-Disposition"] = []string{`form-data; name="file"; filename="` + filename + `"`}
		h["Content-Type"] = []string{mimeType}
		part, err := writer.CreatePart(h)
		require.NoError(t, err)
		_, err = part.Write(fileContent)
		require.NoError(t, err)
	}

	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	if userID != "" {
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		// Stamp a tenant id alongside the user so service code that calls
		// middleware.GetTenantID(ctx) (Welle 1b file/upload wiring) succeeds.
		ctx = context.WithValue(ctx, middleware.TenantIDKey, uuid.New().String())
		req = req.WithContext(ctx)
	}

	return req
}
