package inventar

// BookInventurDifferences shares its transactional booking primitive with
// BookPickingList (see the fix note on both functions in service.go). These
// tests prove the shared guarantee from the Inventur side specifically: a
// difference that cannot be booked must leave the session exactly as it was,
// not completed with some deltas silently dropped and logged away.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newInventurFixture(t *testing.T) (*Service, *mockRepository, uuid.UUID, *Item) {
	t.Helper()
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()
	item := addItem(repo, tenantID, "Schraube M8", "SKU-M8", 40, 5)
	return svc, repo, tenantID, item
}

func createOpenSession(t *testing.T, svc *Service, ctx context.Context, tenantID uuid.UUID) *InventurSession {
	t.Helper()
	session, err := svc.CreateInventurSession(ctx, CreateInventurSessionInput{
		TenantID: tenantID,
		Name:     "Session KW10",
	})
	require.NoError(t, err)
	return session
}

func TestBookInventurDifferences_AppliesDiffsAndCompletesSession(t *testing.T) {
	svc, repo, tenantID, item := newInventurFixture(t)
	ctx := context.Background()

	session := createOpenSession(t, svc, ctx, tenantID)
	// Expected comes from the item at count time (40); counted 25 -> diff -15.
	_, err := svc.UpsertInventurCount(ctx, UpsertInventurCountInput{
		TenantID:  tenantID,
		SessionID: session.ID,
		ItemID:    item.ID,
		Counted:   25,
	})
	require.NoError(t, err)

	booked, err := svc.BookInventurDifferences(ctx, BookInventurDifferencesInput{TenantID: tenantID, SessionID: session.ID})
	require.NoError(t, err)
	assert.Equal(t, InventurStatusCompleted, booked.Status)

	assert.Equal(t, int64(25), repo.items[item.ID].Quantity)
	require.Len(t, repo.movements, 1, "booking must go through the movement path")
	for _, mv := range repo.movements {
		assert.Equal(t, MovementTypeOut, mv.MovementType)
		assert.Equal(t, int64(-15), mv.Quantity)
	}
}

func TestBookInventurDifferences_SkipsUncountedAndZeroDiff(t *testing.T) {
	svc, repo, tenantID, item := newInventurFixture(t)
	ctx := context.Background()

	second := addItem(repo, tenantID, "Mutter M8", "SKU-MU8", 10, 0)
	session := createOpenSession(t, svc, ctx, tenantID)

	// item: counted equals expected (40) -> zero diff, no movement.
	_, err := svc.UpsertInventurCount(ctx, UpsertInventurCountInput{
		TenantID: tenantID, SessionID: session.ID, ItemID: item.ID, Counted: 40,
	})
	require.NoError(t, err)
	// second: left uncounted entirely by never calling UpsertInventurCount for it.
	_ = second

	booked, err := svc.BookInventurDifferences(ctx, BookInventurDifferencesInput{TenantID: tenantID, SessionID: session.ID})
	require.NoError(t, err)
	assert.Equal(t, InventurStatusCompleted, booked.Status)
	assert.Empty(t, repo.movements)
	assert.Equal(t, int64(40), repo.items[item.ID].Quantity)
	assert.Equal(t, int64(10), repo.items[second.ID].Quantity)
}

func TestBookInventurDifferences_PartialFailureLeavesSessionOpenAndStockUntouched(t *testing.T) {
	svc, repo, tenantID, item := newInventurFixture(t)
	ctx := context.Background()

	// short has only 2 in stock; counting it at -5 relative to expected would
	// drive it negative, which must fail the whole batch, not just this item.
	short := addItem(repo, tenantID, "Mutter M8", "SKU-MU8", 2, 0)
	session := createOpenSession(t, svc, ctx, tenantID)

	_, err := svc.UpsertInventurCount(ctx, UpsertInventurCountInput{
		TenantID: tenantID, SessionID: session.ID, ItemID: item.ID, Counted: 25,
	})
	require.NoError(t, err)
	_, err = svc.UpsertInventurCount(ctx, UpsertInventurCountInput{
		TenantID: tenantID, SessionID: session.ID, ItemID: short.ID, Counted: -3,
	})
	require.NoError(t, err)

	_, err = svc.BookInventurDifferences(ctx, BookInventurDifferencesInput{TenantID: tenantID, SessionID: session.ID})
	assert.ErrorIs(t, err, ErrInsufficientStock)

	assert.Equal(t, int64(40), repo.items[item.ID].Quantity, "the bookable position must not move either")
	assert.Equal(t, int64(2), repo.items[short.ID].Quantity)
	assert.Empty(t, repo.movements)

	still, err := svc.GetInventurSession(ctx, tenantID, session.ID)
	require.NoError(t, err)
	assert.NotEqual(t, InventurStatusCompleted, still.Status)
}

func TestBookInventurDifferences_AlreadyCompletedFails(t *testing.T) {
	svc, _, tenantID, item := newInventurFixture(t)
	ctx := context.Background()

	session := createOpenSession(t, svc, ctx, tenantID)
	_, err := svc.UpsertInventurCount(ctx, UpsertInventurCountInput{
		TenantID: tenantID, SessionID: session.ID, ItemID: item.ID, Counted: 30,
	})
	require.NoError(t, err)

	_, err = svc.BookInventurDifferences(ctx, BookInventurDifferencesInput{TenantID: tenantID, SessionID: session.ID})
	require.NoError(t, err)

	_, err = svc.BookInventurDifferences(ctx, BookInventurDifferencesInput{TenantID: tenantID, SessionID: session.ID})
	assert.ErrorIs(t, err, ErrInventurAlreadyCompleted)
}

// TestDeleteInventurSession_CompletedSessionIsNotProtected documents a real
// gap found while building cov-gateway-inventar-inventur-and-reports: unlike
// UpdateInventurSessionStatus and UpsertInventurCount, which both reject a
// completed session (see ErrInventurAlreadyCompleted above),
// Service.DeleteInventurSession (service.go) never checks session.Status
// before calling the repository DELETE. A completed Inventur is a
// handelsrechtlicher Vorgang (Paragraph 240 HGB) and must not be erasable —
// this test proves it currently is. Fix tracked as its own backlog unit
// (fix-inventur-delete-completed-session-unprotected), not fixed here: a
// coverage unit documents behaviour, it does not change it.
func TestDeleteInventurSession_CompletedSessionIsNotProtected(t *testing.T) {
	svc, _, tenantID, item := newInventurFixture(t)
	ctx := context.Background()

	session := createOpenSession(t, svc, ctx, tenantID)
	_, err := svc.UpsertInventurCount(ctx, UpsertInventurCountInput{
		TenantID: tenantID, SessionID: session.ID, ItemID: item.ID, Counted: 30,
	})
	require.NoError(t, err)
	booked, err := svc.BookInventurDifferences(ctx, BookInventurDifferencesInput{TenantID: tenantID, SessionID: session.ID})
	require.NoError(t, err)
	require.Equal(t, InventurStatusCompleted, booked.Status)

	err = svc.DeleteInventurSession(ctx, tenantID, session.ID)
	require.NoError(t, err, "BUG: a completed inventur session can be deleted without restriction")

	_, err = svc.GetInventurSession(ctx, tenantID, session.ID)
	assert.ErrorIs(t, err, ErrInventurSessionNotFound, "the completed session's record is gone after delete")
}
