package vertraege

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/middleware"
)

// ctxWithUser mirrors what the gateway (HTTP auth middleware) and the gRPC
// inbound interceptor both put into the request context: the caller's `sub` as
// a plain string under middleware.UserIDKey.
func ctxWithUser(userID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), middleware.UserIDKey, userID.String())
}

// ============================================================================
// Write path
// ============================================================================

func TestService_CreateContract_RecordsCreatedEvent(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID, userID := uuid.New(), uuid.New()
	c, err := svc.CreateContract(ctxWithUser(userID), CreateContractInput{
		TenantID:       tenantID,
		ContractNumber: "CT-100",
		Title:          "Wartungsvertrag",
		ContractType:   ContractTypeService,
	})
	require.NoError(t, err)

	events := repo.eventsFor(c.ID)
	require.Len(t, events, 1)
	assert.Equal(t, ContractEventCreated, events[0].Action)
	assert.Equal(t, tenantID, events[0].TenantID)
	require.NotNil(t, events[0].UserID)
	assert.Equal(t, userID, *events[0].UserID)
	assert.Equal(t, "CT-100", events[0].Payload["contract_number"])
}

// The reminder worker and the auto-expiry job mutate contracts with no caller
// at all — that must produce an entry without an actor, not no entry.
func TestService_CreateContract_RecordsEventWithoutCaller(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	c, err := svc.CreateContract(context.Background(), CreateContractInput{
		TenantID:       uuid.New(),
		ContractNumber: "CT-101",
		Title:          "Ohne Aufrufer",
		ContractType:   ContractTypeService,
	})
	require.NoError(t, err)

	events := repo.eventsFor(c.ID)
	require.Len(t, events, 1)
	assert.Nil(t, events[0].UserID)
}

func TestService_UpdateContract_RecordsUpdatedEventWithChangedFields(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	c := addContract(repo, tenantID, "CT-200", "Alt", ContractStatusActive)

	newTitle, newNotes := "Neu", "Nachtrag"
	_, err := svc.UpdateContract(ctxWithUser(uuid.New()), UpdateContractInput{
		TenantID:   tenantID,
		ContractID: c.ID,
		Title:      &newTitle,
		Notes:      &newNotes,
	})
	require.NoError(t, err)

	events := repo.eventsFor(c.ID)
	require.Len(t, events, 1)
	assert.Equal(t, ContractEventUpdated, events[0].Action)
	assert.Equal(t, []string{"title", "notes"}, events[0].Payload["fields"])
	assert.NotContains(t, events[0].Payload, "from_status")
}

func TestService_UpdateContract_RecordsTerminationWithPreviousStatus(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	c := addContract(repo, tenantID, "CT-201", "Zu kuendigen", ContractStatusActive)

	terminated := ContractStatusTerminated
	_, err := svc.UpdateContract(ctxWithUser(uuid.New()), UpdateContractInput{
		TenantID:   tenantID,
		ContractID: c.ID,
		Status:     &terminated,
	})
	require.NoError(t, err)

	events := repo.eventsFor(c.ID)
	require.Len(t, events, 1)
	assert.Equal(t, ContractEventTerminated, events[0].Action)
	assert.Equal(t, string(ContractStatusActive), events[0].Payload["from_status"])
}

// Re-saving an already terminated contract is an edit, not a second
// termination — otherwise the trail claims the contract was cut twice.
func TestService_UpdateContract_TerminationIsRecordedOnTransitionOnly(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	c := addContract(repo, tenantID, "CT-202", "Schon gekuendigt", ContractStatusTerminated)

	terminated := ContractStatusTerminated
	notes := "Nachtrag nach Kuendigung"
	_, err := svc.UpdateContract(ctxWithUser(uuid.New()), UpdateContractInput{
		TenantID:   tenantID,
		ContractID: c.ID,
		Status:     &terminated,
		Notes:      &notes,
	})
	require.NoError(t, err)

	events := repo.eventsFor(c.ID)
	require.Len(t, events, 1)
	assert.Equal(t, ContractEventUpdated, events[0].Action)
}

func TestService_UpdateContract_NotFound_RecordsNothing(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	title := "Egal"
	_, err := svc.UpdateContract(ctxWithUser(uuid.New()), UpdateContractInput{
		TenantID:   uuid.New(),
		ContractID: uuid.New(),
		Title:      &title,
	})

	assert.ErrorIs(t, err, ErrContractNotFound)
	assert.Empty(t, repo.events)
}

func TestService_SaveSignature_RecordsSignedEvent(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	c := addContract(repo, tenantID, "CT-300", "Zu unterschreiben", ContractStatusActive)

	_, err := svc.SaveSignature(ctxWithUser(uuid.New()),
		tenantID.String(), c.ID.String(), "data:image/png;base64,AAAA", "Anna Muster")
	require.NoError(t, err)

	events := repo.eventsFor(c.ID)
	require.Len(t, events, 1)
	assert.Equal(t, ContractEventSigned, events[0].Action)
	assert.Equal(t, "Anna Muster", events[0].Payload["signed_by"])
}

func TestService_SaveSignature_InvalidPayload_RecordsNothing(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	c := addContract(repo, tenantID, "CT-301", "Zu unterschreiben", ContractStatusActive)

	_, err := svc.SaveSignature(ctxWithUser(uuid.New()),
		tenantID.String(), c.ID.String(), "data:application/pdf;base64,AAAA", "Anna Muster")

	assert.ErrorIs(t, err, ErrInvalidInput)
	assert.Empty(t, repo.events)
}

func TestService_AddAndRemoveParty_RecordEvents(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	c := addContract(repo, tenantID, "CT-400", "Mit Partei", ContractStatusActive)

	ctx := ctxWithUser(uuid.New())
	p, err := svc.AddParty(ctx, AddPartyInput{
		TenantID:       tenantID,
		ContractID:     c.ID,
		PartyType:      PartyTypeCompany,
		RoleInContract: "Auftraggeber",
	})
	require.NoError(t, err)
	require.NoError(t, svc.RemoveParty(ctx, tenantID, p.ID))

	events := repo.eventsFor(c.ID)
	require.Len(t, events, 2)
	assert.Equal(t, ContractEventPartyAdded, events[0].Action)
	assert.Equal(t, "Auftraggeber", events[0].Payload["role_in_contract"])
	assert.Equal(t, ContractEventPartyRemoved, events[1].Action)
	assert.Equal(t, p.ID.String(), events[1].Payload["party_id"])
}

// Removing a party that is not there stays a no-op; inventing a trail entry
// for it would put a change into the history that never happened.
func TestService_RemoveParty_Missing_RecordsNothing(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	require.NoError(t, svc.RemoveParty(ctxWithUser(uuid.New()), uuid.New(), uuid.New()))
	assert.Empty(t, repo.events)
}

// A trail write that fails must not fail the mutation it describes — the
// contract change is already committed at that point.
func TestService_CreateContract_EventWriteFailureIsNonFatal(t *testing.T) {
	repo := newMockRepository()
	repo.createEventErr = errors.New("audit table unreachable")
	svc := NewService(repo)

	c, err := svc.CreateContract(ctxWithUser(uuid.New()), CreateContractInput{
		TenantID:       uuid.New(),
		ContractNumber: "CT-500",
		Title:          "Trotzdem angelegt",
		ContractType:   ContractTypeService,
	})

	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Empty(t, repo.events)
}

// ============================================================================
// Read path
// ============================================================================

func TestService_ListContractEvents_NewestFirstAndPaginated(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	c := addContract(repo, tenantID, "CT-600", "Mit Verlauf", ContractStatusActive)

	ctx := ctxWithUser(uuid.New())
	notes := "eins"
	for i := 0; i < 3; i++ {
		_, err := svc.UpdateContract(ctx, UpdateContractInput{
			TenantID: tenantID, ContractID: c.ID, Notes: &notes,
		})
		require.NoError(t, err)
	}

	events, total, err := svc.ListContractEvents(ctx, ListContractEventsInput{
		TenantID: tenantID, ContractID: c.ID, Page: 1, PageSize: 2,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, events, 2)
	// Newest first: the last write must lead the page.
	assert.True(t, !events[0].CreatedAt.Before(events[1].CreatedAt))

	page2, total, err := svc.ListContractEvents(ctx, ListContractEventsInput{
		TenantID: tenantID, ContractID: c.ID, Page: 2, PageSize: 2,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, page2, 1)
}

func TestService_ListContractEvents_DefaultsPagination(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	c := addContract(repo, tenantID, "CT-601", "Ohne Verlauf", ContractStatusActive)

	events, total, err := svc.ListContractEvents(context.Background(), ListContractEventsInput{
		TenantID: tenantID, ContractID: c.ID, Page: 0, PageSize: 5000,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, events)
}

// An unknown or foreign contract must answer 404, not an empty trail —
// otherwise "no history" and "not yours" are indistinguishable.
func TestService_ListContractEvents_UnknownContract(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, _, err := svc.ListContractEvents(context.Background(), ListContractEventsInput{
		TenantID: uuid.New(), ContractID: uuid.New(), Page: 1, PageSize: 20,
	})
	assert.ErrorIs(t, err, ErrContractNotFound)
}

func TestService_ListContractEvents_ForeignTenant(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	owner := uuid.New()
	c := addContract(repo, owner, "CT-602", "Fremd", ContractStatusActive)
	_, err := svc.CreateContract(ctxWithUser(uuid.New()), CreateContractInput{
		TenantID: owner, ContractNumber: "CT-603", Title: "X", ContractType: ContractTypeService,
	})
	require.NoError(t, err)

	_, _, listErr := svc.ListContractEvents(context.Background(), ListContractEventsInput{
		TenantID: uuid.New(), ContractID: c.ID, Page: 1, PageSize: 20,
	})
	assert.ErrorIs(t, listErr, ErrContractNotFound)
}

func TestService_ListContractEvents_RepositoryError(t *testing.T) {
	repo := newMockRepository()
	repo.listContractEventErr = errors.New("db down")
	svc := NewService(repo)

	tenantID := uuid.New()
	c := addContract(repo, tenantID, "CT-604", "Kaputt", ContractStatusActive)

	_, _, err := svc.ListContractEvents(context.Background(), ListContractEventsInput{
		TenantID: tenantID, ContractID: c.ID, Page: 1, PageSize: 20,
	})
	assert.Error(t, err)
}
