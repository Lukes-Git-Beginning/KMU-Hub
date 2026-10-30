package inventar

// Service-level CRUD paths that the mock repository stubs out (its location
// methods are no-ops), so they were never run: locations, Inventur sessions,
// picking lists and item attachments. Everything here goes through the real
// PostgresRepository as kmuhub_app so tenant predicates and RLS take part.

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func ptr[T any](v T) *T { return &v }

func TestLocationLifecycle_ServiceAgainstRealSchema(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRepo(t)
	svc := NewService(repo)

	_, err := svc.CreateLocation(ctx, CreateLocationInput{TenantID: tenantID, Name: ""})
	require.ErrorIs(t, err, ErrInvalidInput, "a nameless location is rejected")

	loc, err := svc.CreateLocation(ctx, CreateLocationInput{TenantID: tenantID, Name: "Hauptlager", Address: "Weg 1", Type: "spaceship"})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inventory_locations", loc.ID) })
	assert.Equal(t, LocationTypeWarehouse, loc.Type, "an unknown type falls back to warehouse")

	van, err := svc.CreateLocation(ctx, CreateLocationInput{TenantID: tenantID, Name: "Transporter", Type: LocationTypeVehicle})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inventory_locations", van.ID) })
	assert.Equal(t, LocationTypeVehicle, van.Type, "a valid type is kept")

	got, err := svc.GetLocation(ctx, tenantID, loc.ID)
	require.NoError(t, err)
	assert.Equal(t, "Hauptlager", got.Name)

	updated, err := svc.UpdateLocation(ctx, UpdateLocationInput{
		TenantID: tenantID, LocationID: loc.ID,
		Name: ptr("Zentrallager"), Address: ptr("Weg 2"), Type: ptr(LocationTypeStore),
	})
	require.NoError(t, err)
	assert.Equal(t, "Zentrallager", updated.Name)
	assert.Equal(t, "Weg 2", updated.Address)
	assert.Equal(t, LocationTypeStore, updated.Type)

	// A nil field leaves the stored value alone.
	same, err := svc.UpdateLocation(ctx, UpdateLocationInput{TenantID: tenantID, LocationID: loc.ID})
	require.NoError(t, err)
	assert.Equal(t, "Zentrallager", same.Name)
	assert.Equal(t, "Weg 2", same.Address)

	_, err = svc.UpdateLocation(ctx, UpdateLocationInput{TenantID: tenantID, LocationID: uuid.New(), Name: ptr("x")})
	require.ErrorIs(t, err, ErrLocationNotFound)

	// Paging: a nonsense page/size falls back to page 1 / 50, a real size pages.
	all, total, err := svc.ListLocations(ctx, ListLocationsInput{TenantID: tenantID, Page: -3, PageSize: 9999})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, all, 2)
	one, total, err := svc.ListLocations(ctx, ListLocationsInput{TenantID: tenantID, Page: 2, PageSize: 1})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, one, 1, "page 2 of size 1 holds the second row")

	require.NoError(t, svc.DeleteLocation(ctx, tenantID, van.ID))
	_, err = svc.GetLocation(ctx, tenantID, van.ID)
	require.ErrorIs(t, err, ErrLocationNotFound)
	require.ErrorIs(t, svc.DeleteLocation(ctx, tenantID, van.ID), ErrLocationNotFound, "second delete finds nothing")
	_, total, err = svc.ListLocations(ctx, ListLocationsInput{TenantID: tenantID})
	require.NoError(t, err)
	assert.Equal(t, 1, total, "a deleted location leaves the list")

	// Another tenant neither sees, changes nor deletes it.
	other := uuid.New()
	testutil.EnsureTenant(t, pool, other, "Inventar Location Other")
	otherCtx := testutil.WithTenantCtx(ctx, other)
	_, err = svc.GetLocation(otherCtx, other, loc.ID)
	require.ErrorIs(t, err, ErrLocationNotFound)
	require.ErrorIs(t, svc.DeleteLocation(otherCtx, other, loc.ID), ErrLocationNotFound)
	_, total, err = svc.ListLocations(otherCtx, ListLocationsInput{TenantID: other})
	require.NoError(t, err)
	assert.Zero(t, total)
}

func TestInventurSession_StatusListAndDelete(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRepo(t)
	svc := NewService(repo)

	_, err := svc.CreateInventurSession(ctx, CreateInventurSessionInput{TenantID: tenantID})
	require.ErrorIs(t, err, ErrInvalidInput, "a nameless session is rejected")

	a := newTestItem(t, repo, ctx, pool, tenantID, "Schraube", "SKU-IS-A", 12, 0)
	b := newTestItem(t, repo, ctx, pool, tenantID, "Mutter", "SKU-IS-B", 3, 0)

	// Unknown item ids are skipped when pre-populating, known ones carry their stock.
	session, err := svc.CreateInventurSession(ctx, CreateInventurSessionInput{
		TenantID: tenantID, Name: "Stichtag", ItemIDs: []uuid.UUID{a.ID, uuid.New(), b.ID},
	})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inventur_sessions", session.ID) })
	assert.Equal(t, InventurStatusOpen, session.Status)
	assert.False(t, session.Date.IsZero(), "a missing date defaults to today")

	full, err := svc.GetInventurSession(ctx, tenantID, session.ID)
	require.NoError(t, err)
	require.Len(t, full.Counts, 2)
	expected := map[uuid.UUID]int64{}
	for _, c := range full.Counts {
		expected[c.ItemID] = c.Expected
	}
	assert.Equal(t, map[uuid.UUID]int64{a.ID: 12, b.ID: 3}, expected)

	counting, err := svc.UpdateInventurSessionStatus(ctx, UpdateInventurSessionStatusInput{
		TenantID: tenantID, SessionID: session.ID, Status: InventurStatusCounting,
	})
	require.NoError(t, err)
	assert.Equal(t, InventurStatusCounting, counting.Status)
	reloaded, err := repo.GetInventurSession(ctx, tenantID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, InventurStatusCounting, reloaded.Status, "the new status is persisted")

	_, err = svc.UpdateInventurSessionStatus(ctx, UpdateInventurSessionStatusInput{
		TenantID: tenantID, SessionID: uuid.New(), Status: InventurStatusReview,
	})
	require.ErrorIs(t, err, ErrInventurSessionNotFound)

	listed, total, err := svc.ListInventurSessions(ctx, ListInventurSessionsInput{TenantID: tenantID, Page: 0, PageSize: 0})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, listed, 1)
	assert.Equal(t, session.ID, listed[0].ID)
	empty, total, err := svc.ListInventurSessions(ctx, ListInventurSessionsInput{TenantID: tenantID, Page: 2, PageSize: 1})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Empty(t, empty, "page 2 of one row is empty")

	// A completed session is a booked record: no status change, no delete.
	_, err = svc.UpsertInventurCount(ctx, UpsertInventurCountInput{TenantID: tenantID, SessionID: session.ID, ItemID: a.ID, Counted: 10})
	require.NoError(t, err)
	_, err = svc.BookInventurDifferences(ctx, BookInventurDifferencesInput{TenantID: tenantID, SessionID: session.ID})
	require.NoError(t, err)
	_, err = svc.UpdateInventurSessionStatus(ctx, UpdateInventurSessionStatusInput{
		TenantID: tenantID, SessionID: session.ID, Status: InventurStatusOpen,
	})
	require.ErrorIs(t, err, ErrInventurAlreadyCompleted)
	require.ErrorIs(t, svc.DeleteInventurSession(ctx, tenantID, session.ID), ErrInventurAlreadyCompleted)

	// An open session can be deleted, then it is gone.
	open, err := svc.CreateInventurSession(ctx, CreateInventurSessionInput{TenantID: tenantID, Name: "Entwurf"})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inventur_sessions", open.ID) })
	require.NoError(t, svc.DeleteInventurSession(ctx, tenantID, open.ID))
	_, err = svc.GetInventurSession(ctx, tenantID, open.ID)
	require.ErrorIs(t, err, ErrInventurSessionNotFound)
	require.ErrorIs(t, svc.DeleteInventurSession(ctx, tenantID, open.ID), ErrInventurSessionNotFound)
}

func TestPickingList_UpdateAndPositionLifecycle(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRepo(t)
	svc := NewService(repo)

	item := newTestItem(t, repo, ctx, pool, tenantID, "Dübel", "SKU-PK-A", 20, 0)
	list, err := svc.CreatePickingList(ctx, CreatePickingListInput{TenantID: tenantID, Reference: "PL-1"})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "picking_lists", list.ID) })
	assert.Equal(t, PickingStatusOpen, list.Status)

	// Head edits: guards first, then a real change.
	_, err = svc.UpdatePickingList(ctx, UpdatePickingListInput{TenantID: tenantID, ListID: uuid.New()})
	require.ErrorIs(t, err, ErrPickingListNotFound)
	_, err = svc.UpdatePickingList(ctx, UpdatePickingListInput{TenantID: tenantID, ListID: list.ID, Reference: ptr("")})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = svc.UpdatePickingList(ctx, UpdatePickingListInput{TenantID: tenantID, ListID: list.ID, Status: ptr(PickingStatusCompleted)})
	require.ErrorIs(t, err, ErrInvalidInput, "completed is only reachable by booking")
	_, err = svc.UpdatePickingList(ctx, UpdatePickingListInput{TenantID: tenantID, ListID: list.ID, Status: ptr(PickingStatus("bogus"))})
	require.ErrorIs(t, err, ErrInvalidInput)
	untouched, err := repo.GetPickingList(ctx, tenantID, list.ID)
	require.NoError(t, err)
	assert.Equal(t, "PL-1", untouched.Reference)
	assert.Equal(t, PickingStatusOpen, untouched.Status, "rejected edits change nothing")

	assignee := uuid.New()
	edited, err := svc.UpdatePickingList(ctx, UpdatePickingListInput{
		TenantID: tenantID, ListID: list.ID,
		Reference: ptr("PL-1b"), Status: ptr(PickingStatusPicking), AssignedTo: &assignee,
	})
	require.NoError(t, err)
	assert.Equal(t, "PL-1b", edited.Reference)
	assert.Equal(t, PickingStatusPicking, edited.Status)
	require.NotNil(t, edited.AssignedTo)
	assert.Equal(t, assignee, *edited.AssignedTo)
	_, err = svc.UpdatePickingList(ctx, UpdatePickingListInput{TenantID: tenantID, ListID: list.ID, Status: ptr(PickingStatusOpen)})
	require.NoError(t, err)

	// Positions: validation, unknown item, then the first pick advances the list.
	_, err = svc.UpsertPickingListItem(ctx, UpsertPickingListItemInput{TenantID: tenantID, ListID: uuid.New(), ItemID: item.ID, QuantityRequested: 5})
	require.ErrorIs(t, err, ErrPickingListNotFound)
	_, err = svc.UpsertPickingListItem(ctx, UpsertPickingListItemInput{TenantID: tenantID, ListID: list.ID, ItemID: item.ID, QuantityRequested: 5, QuantityPicked: 6})
	require.ErrorIs(t, err, ErrInvalidInput, "picked beyond requested")
	_, err = svc.UpsertPickingListItem(ctx, UpsertPickingListItemInput{TenantID: tenantID, ListID: list.ID, ItemID: uuid.New(), QuantityRequested: 5})
	require.ErrorIs(t, err, ErrItemNotFound)

	_, err = svc.UpsertPickingListItem(ctx, UpsertPickingListItemInput{TenantID: tenantID, ListID: list.ID, ItemID: item.ID, QuantityRequested: 5})
	require.NoError(t, err)
	afterZeroPick, err := repo.GetPickingList(ctx, tenantID, list.ID)
	require.NoError(t, err)
	assert.Equal(t, PickingStatusOpen, afterZeroPick.Status, "a position with nothing picked keeps the list open")

	_, err = svc.UpsertPickingListItem(ctx, UpsertPickingListItemInput{
		TenantID: tenantID, ListID: list.ID, ItemID: item.ID, QuantityRequested: 5, QuantityPicked: 2, Location: ptr("Regal 3"),
	})
	require.NoError(t, err)
	afterPick, err := repo.GetPickingList(ctx, tenantID, list.ID)
	require.NoError(t, err)
	assert.Equal(t, PickingStatusPicking, afterPick.Status, "the first recorded pick moves the list on")
	require.Len(t, afterPick.Items, 1, "the same item is one position, not two")
	assert.EqualValues(t, 2, afterPick.Items[0].QuantityPicked)

	// Deleting a position / the list removes them; a stranger's id finds nothing.
	require.ErrorIs(t, svc.DeletePickingListItem(ctx, tenantID, uuid.New()), ErrPickingListItemNotFound)
	require.NoError(t, svc.DeletePickingListItem(ctx, tenantID, afterPick.Items[0].ID))
	emptied, err := repo.GetPickingList(ctx, tenantID, list.ID)
	require.NoError(t, err)
	assert.Empty(t, emptied.Items)

	require.ErrorIs(t, svc.DeletePickingList(ctx, tenantID, uuid.New()), ErrPickingListNotFound)
	require.NoError(t, svc.DeletePickingList(ctx, tenantID, list.ID))
	_, err = repo.GetPickingList(ctx, tenantID, list.ID)
	require.ErrorIs(t, err, ErrPickingListNotFound)
}

func TestPickingList_BookedListIsFrozen(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRepo(t)
	svc := NewService(repo)

	item := newTestItem(t, repo, ctx, pool, tenantID, "Dübel", "SKU-PK-B", 20, 0)
	list, err := svc.CreatePickingList(ctx, CreatePickingListInput{
		TenantID: tenantID, Reference: "PL-2",
		Items: []PickingListItemInput{{ItemID: item.ID, QuantityRequested: 4, QuantityPicked: 4}},
	})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "picking_lists", list.ID) })

	_, err = svc.BookPickingList(ctx, BookPickingListInput{TenantID: tenantID, ListID: list.ID})
	require.NoError(t, err)

	_, err = svc.UpdatePickingList(ctx, UpdatePickingListInput{TenantID: tenantID, ListID: list.ID, Reference: ptr("neu")})
	require.ErrorIs(t, err, ErrPickingListAlreadyBooked)
	_, err = svc.UpsertPickingListItem(ctx, UpsertPickingListItemInput{TenantID: tenantID, ListID: list.ID, ItemID: item.ID, QuantityRequested: 9})
	require.ErrorIs(t, err, ErrPickingListAlreadyBooked)

	stock, err := repo.GetItem(ctx, tenantID, item.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 16, stock.Quantity, "booked once, rejected edits moved nothing")
}

func TestItemAttachments_ServiceAgainstRealSchema(t *testing.T) {
	t.Parallel()
	repo, pool, ctx, tenantID := setupRepo(t)
	svc := NewService(repo)

	item := newTestItem(t, repo, ctx, pool, tenantID, "Palette", "SKU-AT-A", 1, 0)

	for name, in := range map[string]CreateItemAttachmentInput{
		"blank name":       {TenantID: tenantID, ItemID: item.ID, Name: "  ", ObjectKey: "k"},
		"blank object key": {TenantID: tenantID, ItemID: item.ID, Name: "n.pdf", ObjectKey: " "},
	} {
		_, err := svc.CreateItemAttachment(ctx, in)
		assert.ErrorIs(t, err, ErrInvalidInput, name)
	}
	_, err := svc.CreateItemAttachment(ctx, CreateItemAttachmentInput{TenantID: tenantID, ItemID: uuid.New(), Name: "n.pdf", ObjectKey: "k"})
	require.ErrorIs(t, err, ErrItemNotFound, "attaching to an unknown item is a clean not-found, not an FK error")

	att, err := svc.CreateItemAttachment(ctx, CreateItemAttachmentInput{
		TenantID: tenantID, ItemID: item.ID, Name: "Datenblatt.pdf", ObjectKey: "inventar/" + item.ID.String() + "/d.pdf", FileType: "application/pdf",
	})
	require.NoError(t, err)
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inventory_item_attachments", att.ID) })

	list, err := svc.ListItemAttachments(ctx, tenantID, item.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Datenblatt.pdf", list[0].Name)

	// Another tenant sees nothing and cannot remove it.
	other := uuid.New()
	testutil.EnsureTenant(t, pool, other, "Inventar Attachment Other")
	otherCtx := testutil.WithTenantCtx(ctx, other)
	foreign, err := svc.ListItemAttachments(otherCtx, other, item.ID)
	require.NoError(t, err)
	assert.Empty(t, foreign)
	require.True(t, errors.Is(svc.DeleteItemAttachment(otherCtx, other, att.ID), ErrAttachmentNotFound))

	require.NoError(t, svc.DeleteItemAttachment(ctx, tenantID, att.ID))
	require.ErrorIs(t, svc.DeleteItemAttachment(ctx, tenantID, att.ID), ErrAttachmentNotFound)
	left, err := svc.ListItemAttachments(ctx, tenantID, item.ID)
	require.NoError(t, err)
	assert.Empty(t, left)
}

func TestMovementTypeForDelta_AndPickQuantities(t *testing.T) {
	t.Parallel()
	assert.Equal(t, MovementTypeOut, movementTypeForDelta(-1))
	assert.Equal(t, MovementTypeIn, movementTypeForDelta(1))
	assert.Equal(t, MovementTypeAdjustment, movementTypeForDelta(0))

	assert.ErrorIs(t, validatePickQuantities(0, 0), ErrInvalidInput)
	assert.ErrorIs(t, validatePickQuantities(3, -1), ErrInvalidInput)
	assert.ErrorIs(t, validatePickQuantities(3, 4), ErrInvalidInput)
	assert.NoError(t, validatePickQuantities(3, 3))
	assert.NoError(t, validatePickQuantities(3, 0))
}
