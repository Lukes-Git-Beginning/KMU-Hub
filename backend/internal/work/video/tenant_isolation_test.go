package video

// Cross-tenant isolation tests for call_sessions and call_participants (Migration 000114).
//
// call_sessions.tenant_id and call_participants.tenant_id are NOT NULL after Migration 000114.
// These tests verify that:
//   - CreateCall stores the tenant from context on the CallSession and participant.
//   - Sessions created under tenant A are not visible when querying for tenant B's data
//     (verified at the mock level, mirroring what the DB enforces with RLS).

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCallSession_TenantIDStoredOnCreate verifies that CreateCall sets TenantID on
// the resulting CallSession and on the initiator's CallParticipant record.
func TestCallSession_TenantIDStoredOnCreate(t *testing.T) {
	repo := newMockRepo()
	rm := newMockRoomManager()
	svc := NewService(repo, rm)

	tenantID := uuid.New()
	initiator := uuid.New()
	target := uuid.New()

	session, _, err := svc.CreateCall(tenantCtx(tenantID), initiator, CallTypeOneToOne, []uuid.UUID{target}, nil)
	require.NoError(t, err)

	// Session must carry the tenant from context.
	assert.Equal(t, tenantID, session.TenantID, "call session must carry TenantID from context")

	// Initiator participant must carry the same tenant.
	parts := repo.participants[session.ID]
	require.Len(t, parts, 1)
	assert.Equal(t, tenantID, parts[0].TenantID, "initiator participant must carry same TenantID as session")
}

// TestCallParticipant_TenantIDInheritedOnJoin verifies that a joining participant
// inherits the tenant from the parent call session, not from the join context.
func TestCallParticipant_TenantIDInheritedOnJoin(t *testing.T) {
	repo := newMockRepo()
	rm := newMockRoomManager()
	svc := NewService(repo, rm)

	tenantA := uuid.New()
	initiator := uuid.New()
	joiner := uuid.New()

	// Initiator creates call under tenant A.
	session, _, err := svc.CreateCall(tenantCtx(tenantA), initiator, CallTypeGroup, []uuid.UUID{joiner}, nil)
	require.NoError(t, err)

	// Joiner joins — even if ctx carries a different tenant, the participant's
	// TenantID must be inherited from the session (not injected from the joiner's ctx).
	tenantB := uuid.New()
	_, err = svc.JoinCall(tenantCtx(tenantB), session.ID, joiner, "Joiner")
	require.NoError(t, err)

	// Find joiner's participant record.
	parts := repo.participants[session.ID]
	require.Len(t, parts, 2)
	var joinerPart *CallParticipant
	for i := range parts {
		if parts[i].UserID == joiner {
			joinerPart = &parts[i]
			break
		}
	}
	require.NotNil(t, joinerPart, "joiner participant must exist")
	// Joiner must inherit tenant from session, not from its own context.
	assert.Equal(t, tenantA, joinerPart.TenantID, "joiner participant must inherit session TenantID")
}

// TestCallSession_CrossTenantVisibility verifies that sessions stored under tenant A
// are not returned when listing active calls for a user under tenant B.
// (This is a mock-level check; Postgres enforces it via RLS on the tenant_id column.)
func TestCallSession_CrossTenantVisibility(t *testing.T) {
	repo := newMockRepo()
	rm := newMockRoomManager()
	svc := NewService(repo, rm)

	tenantA := uuid.New()
	tenantB := uuid.New()
	userA := uuid.New()
	userB := uuid.New()

	// Create a call for tenant A.
	_, _, err := svc.CreateCall(tenantCtx(tenantA), userA, CallTypeGroup, []uuid.UUID{uuid.New()}, nil)
	require.NoError(t, err)

	// List calls for userB — must not see tenant A's call.
	calls, err := svc.ListActiveCalls(tenantCtx(tenantB), userB)
	require.NoError(t, err)

	for _, c := range calls {
		assert.NotEqual(t, tenantA, c.TenantID, "tenant A call must not be visible to tenant B user")
	}
}
