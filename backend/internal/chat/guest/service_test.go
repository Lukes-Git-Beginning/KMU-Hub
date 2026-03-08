package guest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRepository implements Repository for testing.
type MockRepository struct {
	CreateSessionFn          func(ctx context.Context, session *GuestSession) error
	GetSessionByTokenHashFn  func(ctx context.Context, tokenHash string) (*GuestSession, error)
	GetSessionByIDFn         func(ctx context.Context, id uuid.UUID) (*GuestSession, error)
	ListSessionsByChannelFn  func(ctx context.Context, channelID uuid.UUID) ([]*GuestSession, error)
	UpdateLastActivityFn     func(ctx context.Context, id uuid.UUID) error
	DeactivateSessionFn      func(ctx context.Context, id uuid.UUID) error
	CleanupExpiredSessionsFn func(ctx context.Context) (int, error)
	CreateConfigFn           func(ctx context.Context, config *GuestChannelConfig) error
	GetConfigByChannelFn     func(ctx context.Context, channelID uuid.UUID) (*GuestChannelConfig, error)
	UpdateConfigFn           func(ctx context.Context, config *GuestChannelConfig) error
	DeleteConfigFn           func(ctx context.Context, channelID uuid.UUID) error
}

func (m *MockRepository) CreateSession(ctx context.Context, session *GuestSession) error {
	if m.CreateSessionFn != nil {
		return m.CreateSessionFn(ctx, session)
	}
	return nil
}

func (m *MockRepository) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*GuestSession, error) {
	if m.GetSessionByTokenHashFn != nil {
		return m.GetSessionByTokenHashFn(ctx, tokenHash)
	}
	return nil, ErrSessionNotFound
}

func (m *MockRepository) GetSessionByID(ctx context.Context, id uuid.UUID) (*GuestSession, error) {
	if m.GetSessionByIDFn != nil {
		return m.GetSessionByIDFn(ctx, id)
	}
	return nil, ErrSessionNotFound
}

func (m *MockRepository) ListSessionsByChannel(ctx context.Context, channelID uuid.UUID) ([]*GuestSession, error) {
	if m.ListSessionsByChannelFn != nil {
		return m.ListSessionsByChannelFn(ctx, channelID)
	}
	return nil, nil
}

func (m *MockRepository) UpdateLastActivity(ctx context.Context, id uuid.UUID) error {
	if m.UpdateLastActivityFn != nil {
		return m.UpdateLastActivityFn(ctx, id)
	}
	return nil
}

func (m *MockRepository) DeactivateSession(ctx context.Context, id uuid.UUID) error {
	if m.DeactivateSessionFn != nil {
		return m.DeactivateSessionFn(ctx, id)
	}
	return nil
}

func (m *MockRepository) CleanupExpiredSessions(ctx context.Context) (int, error) {
	if m.CleanupExpiredSessionsFn != nil {
		return m.CleanupExpiredSessionsFn(ctx)
	}
	return 0, nil
}

func (m *MockRepository) CreateConfig(ctx context.Context, config *GuestChannelConfig) error {
	if m.CreateConfigFn != nil {
		return m.CreateConfigFn(ctx, config)
	}
	return nil
}

func (m *MockRepository) GetConfigByChannel(ctx context.Context, channelID uuid.UUID) (*GuestChannelConfig, error) {
	if m.GetConfigByChannelFn != nil {
		return m.GetConfigByChannelFn(ctx, channelID)
	}
	return nil, ErrConfigNotFound
}

func (m *MockRepository) UpdateConfig(ctx context.Context, config *GuestChannelConfig) error {
	if m.UpdateConfigFn != nil {
		return m.UpdateConfigFn(ctx, config)
	}
	return nil
}

func (m *MockRepository) DeleteConfig(ctx context.Context, channelID uuid.UUID) error {
	if m.DeleteConfigFn != nil {
		return m.DeleteConfigFn(ctx, channelID)
	}
	return nil
}

func TestCreateSession_Success(t *testing.T) {
	repo := &MockRepository{
		CreateSessionFn: func(_ context.Context, _ *GuestSession) error {
			return nil
		},
	}
	svc := NewService(repo)

	result, err := svc.CreateSession(context.Background(), CreateSessionInput{
		ChannelID:   uuid.New(),
		DisplayName: "Test Guest",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Token)
	assert.NotNil(t, result.Session)
	assert.Equal(t, "Test Guest", result.Session.DisplayName)
	assert.True(t, result.Session.IsActive)
	assert.False(t, result.Session.ExpiresAt.IsZero())
}

func TestCreateSession_EmptyDisplayName(t *testing.T) {
	svc := NewService(&MockRepository{})

	result, err := svc.CreateSession(context.Background(), CreateSessionInput{
		ChannelID:   uuid.New(),
		DisplayName: "",
	})

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrDisplayNameRequired)
}

func TestCreateSession_DisplayNameTooLong(t *testing.T) {
	svc := NewService(&MockRepository{})

	longName := make([]byte, 201)
	for i := range longName {
		longName[i] = 'a'
	}

	result, err := svc.CreateSession(context.Background(), CreateSessionInput{
		ChannelID:   uuid.New(),
		DisplayName: string(longName),
	})

	assert.Nil(t, result)
	assert.ErrorIs(t, err, ErrDisplayNameTooLong)
}

func TestValidateToken_Success(t *testing.T) {
	// We need to create a session first, then validate the returned token.
	var storedSession *GuestSession
	repo := &MockRepository{
		CreateSessionFn: func(_ context.Context, session *GuestSession) error {
			storedSession = session
			return nil
		},
		GetSessionByTokenHashFn: func(_ context.Context, tokenHash string) (*GuestSession, error) {
			if storedSession != nil && storedSession.TokenHash == tokenHash {
				return storedSession, nil
			}
			return nil, ErrSessionNotFound
		},
	}
	svc := NewService(repo)

	result, err := svc.CreateSession(context.Background(), CreateSessionInput{
		ChannelID:   uuid.New(),
		DisplayName: "Validator",
	})
	require.NoError(t, err)

	session, err := svc.ValidateToken(context.Background(), result.Token)
	require.NoError(t, err)
	assert.Equal(t, storedSession.ID, session.ID)
	assert.Equal(t, "Validator", session.DisplayName)
}

func TestValidateToken_InvalidToken(t *testing.T) {
	svc := NewService(&MockRepository{})

	session, err := svc.ValidateToken(context.Background(), "")
	assert.Nil(t, session)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateToken_Expired(t *testing.T) {
	expiredSession := &GuestSession{
		ID:        uuid.New(),
		IsActive:  true,
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	repo := &MockRepository{
		GetSessionByTokenHashFn: func(_ context.Context, _ string) (*GuestSession, error) {
			return expiredSession, nil
		},
	}
	svc := NewService(repo)

	session, err := svc.ValidateToken(context.Background(), "some-token")
	assert.Nil(t, session)
	assert.ErrorIs(t, err, ErrSessionExpired)
}

func TestValidateToken_Inactive(t *testing.T) {
	inactiveSession := &GuestSession{
		ID:        uuid.New(),
		IsActive:  false,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	repo := &MockRepository{
		GetSessionByTokenHashFn: func(_ context.Context, _ string) (*GuestSession, error) {
			return inactiveSession, nil
		},
	}
	svc := NewService(repo)

	session, err := svc.ValidateToken(context.Background(), "some-token")
	assert.Nil(t, session)
	assert.ErrorIs(t, err, ErrSessionInactive)
}

func TestDeactivateSession_Success(t *testing.T) {
	called := false
	repo := &MockRepository{
		DeactivateSessionFn: func(_ context.Context, _ uuid.UUID) error {
			called = true
			return nil
		},
	}
	svc := NewService(repo)

	err := svc.DeactivateSession(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.True(t, called)
}

func TestCleanupExpired_Success(t *testing.T) {
	repo := &MockRepository{
		CleanupExpiredSessionsFn: func(_ context.Context) (int, error) {
			return 5, nil
		},
	}
	svc := NewService(repo)

	count, err := svc.CleanupExpired(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestCreateConfig_Success(t *testing.T) {
	repo := &MockRepository{
		CreateConfigFn: func(_ context.Context, config *GuestChannelConfig) error {
			return nil
		},
	}
	svc := NewService(repo)

	channelID := uuid.New()
	createdBy := uuid.New()
	config, err := svc.CreateConfig(context.Background(), CreateConfigInput{
		ChannelID: channelID,
		CreatedBy: createdBy,
	})

	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, channelID, config.ChannelID)
	assert.Equal(t, createdBy, config.CreatedBy)
	assert.Equal(t, "Willkommen! Wie können wir Ihnen helfen?", config.WelcomeMessage)
	assert.Equal(t, "#3b82f6", config.PrimaryColor)
	assert.Equal(t, 168, config.TokenExpiryHours)
	assert.Equal(t, 10, config.MaxFileSizeMB)
	assert.True(t, config.IsActive)
}

func TestGetConfig_Success(t *testing.T) {
	channelID := uuid.New()
	expected := &GuestChannelConfig{
		ID:        uuid.New(),
		ChannelID: channelID,
	}
	repo := &MockRepository{
		GetConfigByChannelFn: func(_ context.Context, id uuid.UUID) (*GuestChannelConfig, error) {
			if id == channelID {
				return expected, nil
			}
			return nil, ErrConfigNotFound
		},
	}
	svc := NewService(repo)

	config, err := svc.GetConfig(context.Background(), channelID)
	require.NoError(t, err)
	assert.Equal(t, expected.ID, config.ID)
}

func TestGetConfig_NotFound(t *testing.T) {
	repo := &MockRepository{
		GetConfigByChannelFn: func(_ context.Context, _ uuid.UUID) (*GuestChannelConfig, error) {
			return nil, ErrConfigNotFound
		},
	}
	svc := NewService(repo)

	config, err := svc.GetConfig(context.Background(), uuid.New())
	assert.Nil(t, config)
	assert.ErrorIs(t, err, ErrConfigNotFound)
}

func TestCheckRateLimit_Allowed(t *testing.T) {
	svc := NewService(&MockRepository{})

	err := svc.CheckRateLimit(uuid.New())
	assert.NoError(t, err)
}

func TestCheckRateLimit_Exceeded(t *testing.T) {
	svc := NewService(&MockRepository{})
	sessionID := uuid.New()

	// Exhaust the rate limiter (30 msgs/min)
	for i := 0; i < 30; i++ {
		_ = svc.CheckRateLimit(sessionID)
	}

	err := svc.CheckRateLimit(sessionID)
	assert.ErrorIs(t, err, ErrRateLimited)
}

func TestCreateSession_RepoError(t *testing.T) {
	repoErr := errors.New("db connection failed")
	repo := &MockRepository{
		CreateSessionFn: func(_ context.Context, _ *GuestSession) error {
			return repoErr
		},
	}
	svc := NewService(repo)

	result, err := svc.CreateSession(context.Background(), CreateSessionInput{
		ChannelID:   uuid.New(),
		DisplayName: "Test",
	})

	assert.Nil(t, result)
	assert.ErrorIs(t, err, repoErr)
}
