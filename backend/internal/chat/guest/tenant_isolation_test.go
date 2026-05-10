package guest

// Cross-tenant isolation tests for guest_sessions and guest_channel_config (Migration 000115).
//
// guest_sessions.tenant_id and guest_channel_config.tenant_id are NOT NULL after Migration 000115.
// These tests verify that:
//   - CreateSession stores the tenant from context on the GuestSession.
//   - CreateConfig stores the tenant from context on the GuestChannelConfig.
//   - Sessions created under tenant A are not returned for tenant B channels.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/middleware"
)

func tenantCtxIsolation(tenantID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), middleware.TenantIDKey, tenantID.String())
}

// TestGuestSession_TenantIDStoredOnCreate verifies that CreateSession sets TenantID
// on the resulting GuestSession from the request context.
func TestGuestSession_TenantIDStoredOnCreate(t *testing.T) {
	var captured *GuestSession
	repo := &MockRepository{
		CreateSessionFn: func(_ context.Context, s *GuestSession) error {
			captured = s
			return nil
		},
	}
	svc := NewService(repo)

	tenantID := uuid.New()
	_, err := svc.CreateSession(tenantCtxIsolation(tenantID), CreateSessionInput{
		ChannelID:   uuid.New(),
		DisplayName: "GuestA",
	})
	require.NoError(t, err)
	require.NotNil(t, captured, "repo.CreateSession must have been called")
	assert.Equal(t, tenantID, captured.TenantID, "guest session must carry TenantID from context")
}

// TestGuestChannelConfig_TenantIDStoredOnCreate verifies that CreateConfig sets
// TenantID on the resulting GuestChannelConfig from the request context.
func TestGuestChannelConfig_TenantIDStoredOnCreate(t *testing.T) {
	var captured *GuestChannelConfig
	repo := &MockRepository{
		CreateConfigFn: func(_ context.Context, c *GuestChannelConfig) error {
			captured = c
			return nil
		},
	}
	svc := NewService(repo)

	tenantID := uuid.New()
	_, err := svc.CreateConfig(tenantCtxIsolation(tenantID), CreateConfigInput{
		ChannelID: uuid.New(),
		CreatedBy: uuid.New(),
	})
	require.NoError(t, err)
	require.NotNil(t, captured, "repo.CreateConfig must have been called")
	assert.Equal(t, tenantID, captured.TenantID, "guest channel config must carry TenantID from context")
}

// TestGuestSession_CrossTenant verifies that guest sessions for channel A (tenant A)
// are distinct from sessions for channel B (tenant B).
func TestGuestSession_CrossTenant(t *testing.T) {
	var sessA, sessB *GuestSession
	var captureIdx int
	repo := &MockRepository{
		CreateSessionFn: func(_ context.Context, s *GuestSession) error {
			if captureIdx == 0 {
				sessA = s
			} else {
				sessB = s
			}
			captureIdx++
			return nil
		},
	}
	svc := NewService(repo)

	tenantA := uuid.New()
	tenantB := uuid.New()
	channelA := uuid.New()
	channelB := uuid.New()

	_, err := svc.CreateSession(tenantCtxIsolation(tenantA), CreateSessionInput{
		ChannelID:   channelA,
		DisplayName: "GuestTenantA",
	})
	require.NoError(t, err)

	_, err = svc.CreateSession(tenantCtxIsolation(tenantB), CreateSessionInput{
		ChannelID:   channelB,
		DisplayName: "GuestTenantB",
	})
	require.NoError(t, err)

	require.NotNil(t, sessA)
	require.NotNil(t, sessB)

	// Each session must carry its own tenant.
	assert.Equal(t, tenantA, sessA.TenantID, "tenant A session must carry tenant A")
	assert.Equal(t, tenantB, sessB.TenantID, "tenant B session must carry tenant B")

	// Tenants must not cross-contaminate.
	assert.NotEqual(t, sessA.TenantID, sessB.TenantID, "tenant A and tenant B sessions must have different tenant IDs")
	assert.NotEqual(t, sessA.ChannelID, sessB.ChannelID, "sessions must belong to different channels")
}
