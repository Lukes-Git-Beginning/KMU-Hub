package gateway

// Contract, party and reminder dates must reach the vertraege service: the handlers used to
// decode starts_on/ends_on/signed_on/remind_at and drop them, so every contract was stored with
// starts_on 0001-01-01. A real gRPC server on loopback records what arrives.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	vertraegev1 "github.com/kmuhub/kmuhub/proto/vertraege/v1"
)

type vertraegeDatesStub struct {
	vertraegev1.UnimplementedVertraegeServiceServer
	create   *vertraegev1.CreateContractRequest
	update   *vertraegev1.UpdateContractRequest
	party    *vertraegev1.AddPartyRequest
	remindIn *vertraegev1.CreateReminderRequest
	remindUp *vertraegev1.UpdateReminderRequest
}

func (s *vertraegeDatesStub) CreateContract(_ context.Context, r *vertraegev1.CreateContractRequest) (*vertraegev1.ContractResponse, error) {
	s.create = r
	return &vertraegev1.ContractResponse{Contract: &vertraegev1.Contract{Id: "c", StartsOn: r.GetStartsOn(), EndsOn: r.GetEndsOn()}}, nil
}

func (s *vertraegeDatesStub) UpdateContract(_ context.Context, r *vertraegev1.UpdateContractRequest) (*vertraegev1.ContractResponse, error) {
	s.update = r
	return &vertraegev1.ContractResponse{Contract: &vertraegev1.Contract{Id: "c", StartsOn: r.GetStartsOn(), EndsOn: r.GetEndsOn()}}, nil
}

func (s *vertraegeDatesStub) AddParty(_ context.Context, r *vertraegev1.AddPartyRequest) (*vertraegev1.PartyResponse, error) {
	s.party = r
	return &vertraegev1.PartyResponse{Party: &vertraegev1.ContractParty{Id: "p", SignedOn: r.GetSignedOn()}}, nil
}

func (s *vertraegeDatesStub) CreateReminder(_ context.Context, r *vertraegev1.CreateReminderRequest) (*vertraegev1.ReminderResponse, error) {
	s.remindIn = r
	return &vertraegev1.ReminderResponse{Reminder: &vertraegev1.ContractReminder{Id: "r", RemindAt: r.GetRemindAt()}}, nil
}

func (s *vertraegeDatesStub) UpdateReminder(_ context.Context, r *vertraegev1.UpdateReminderRequest) (*vertraegev1.ReminderResponse, error) {
	s.remindUp = r
	return &vertraegev1.ReminderResponse{Reminder: &vertraegev1.ContractReminder{Id: "r", RemindAt: r.GetRemindAt()}}, nil
}

func newVertraegeDatesRoutes(t *testing.T) (*VertraegeRoutes, *vertraegeDatesStub) {
	t.Helper()
	stub := &vertraegeDatesStub{}
	return newVertraegeRoutes(registryFor(t, "vertraege", func(s *grpc.Server) { vertraegev1.RegisterVertraegeServiceServer(s, stub) })), stub
}

func vertraegeDatesRequest(method, body string) *http.Request {
	req := taxModeRequest(method, body)
	return withChiURLParam(req, "reminderId", "22222222-2222-2222-2222-222222222222")
}

func utcDay(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestVertraegeDates_CreateContractCarriesDates(t *testing.T) {
	routes, stub := newVertraegeDatesRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleCreateContract(rec, vertraegeDatesRequest(http.MethodPost,
		`{"contract_number":"C-1","title":"Wartung","contract_type":"service","starts_on":"2026-03-01","ends_on":"2027-02-28T00:00:00Z"}`))

	assertStatus(t, rec, http.StatusCreated)
	require.NotNil(t, stub.create)
	assert.Equal(t, utcDay(2026, 3, 1), stub.create.GetStartsOn().AsTime())
	assert.Equal(t, utcDay(2027, 2, 28), stub.create.GetEndsOn().AsTime())
	contract := decodeBody(t, rec)["contract"].(map[string]any)
	assert.Contains(t, contract["starts_on"], "2026-03-01", rec.Body.String())
}

func TestVertraegeDates_CreateContractWithoutDatesStaysOptional(t *testing.T) {
	routes, stub := newVertraegeDatesRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleCreateContract(rec, vertraegeDatesRequest(http.MethodPost,
		`{"contract_number":"C-1","title":"Wartung","contract_type":"service"}`))

	assertStatus(t, rec, http.StatusCreated)
	assert.Nil(t, stub.create.GetStartsOn())
	assert.Nil(t, stub.create.GetEndsOn())
}

func TestVertraegeDates_UpdateContractCarriesDatesAndClearFlag(t *testing.T) {
	routes, stub := newVertraegeDatesRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleUpdateContract(rec, vertraegeDatesRequest(http.MethodPatch,
		`{"starts_on":"2026-04-15","ends_on":"2026-12-31","clear_ends_on":false}`))
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, utcDay(2026, 4, 15), stub.update.GetStartsOn().AsTime())
	assert.Equal(t, utcDay(2026, 12, 31), stub.update.GetEndsOn().AsTime())

	rec = httptest.NewRecorder()
	routes.HandleUpdateContract(rec, vertraegeDatesRequest(http.MethodPatch, `{"clear_ends_on":true}`))
	assertStatus(t, rec, http.StatusOK)
	assert.Nil(t, stub.update.GetStartsOn(), "an unsent date stays unchanged")
	assert.True(t, stub.update.GetClearEndsOn())
}

func TestVertraegeDates_AddPartyAndReminderCarryDates(t *testing.T) {
	routes, stub := newVertraegeDatesRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleAddParty(rec, vertraegeDatesRequest(http.MethodPost,
		`{"party_type":"internal","role_in_contract":"signatory","signed_on":"2026-05-02"}`))
	assertStatus(t, rec, http.StatusCreated)
	assert.Equal(t, utcDay(2026, 5, 2), stub.party.GetSignedOn().AsTime())

	rec = httptest.NewRecorder()
	routes.HandleCreateReminder(rec, vertraegeDatesRequest(http.MethodPost,
		`{"remind_at":"2026-11-01T09:00:00Z","reminder_type":"expiry","subject":"Kündigungsfrist"}`))
	assertStatus(t, rec, http.StatusCreated)
	assert.Equal(t, time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC), stub.remindIn.GetRemindAt().AsTime())

	rec = httptest.NewRecorder()
	routes.HandleUpdateReminder(rec, vertraegeDatesRequest(http.MethodPatch, `{"remind_at":"2026-12-01"}`))
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, utcDay(2026, 12, 1), stub.remindUp.GetRemindAt().AsTime())
}

func TestVertraegeDates_MalformedDateIsBadRequest(t *testing.T) {
	routes, stub := newVertraegeDatesRoutes(t)

	cases := map[string]struct {
		call   func(http.ResponseWriter, *http.Request)
		method string
		body   string
		field  string
	}{
		"create starts_on": {routes.HandleCreateContract, http.MethodPost, `{"contract_number":"C","title":"T","contract_type":"service","starts_on":"01.03.2026"}`, "starts_on"},
		"create ends_on":   {routes.HandleCreateContract, http.MethodPost, `{"contract_number":"C","title":"T","contract_type":"service","ends_on":"soon"}`, "ends_on"},
		"update starts_on": {routes.HandleUpdateContract, http.MethodPatch, `{"starts_on":"2026-13-40"}`, "starts_on"},
		"update empty":     {routes.HandleUpdateContract, http.MethodPatch, `{"starts_on":""}`, "starts_on"},
		"party signed_on":  {routes.HandleAddParty, http.MethodPost, `{"party_type":"internal","role_in_contract":"x","signed_on":"x"}`, "signed_on"},
		"reminder create":  {routes.HandleCreateReminder, http.MethodPost, `{"remind_at":"tomorrow","subject":"s"}`, "remind_at"},
		"reminder update":  {routes.HandleUpdateReminder, http.MethodPatch, `{"remind_at":"tomorrow"}`, "remind_at"},
	}
	for label, c := range cases {
		t.Run(label, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c.call(rec, vertraegeDatesRequest(c.method, c.body))
			assertStatus(t, rec, http.StatusBadRequest)
			assertErrorContains(t, rec, c.field)
		})
	}
	assert.Nil(t, stub.create, "a rejected date never reaches the service")
	assert.Nil(t, stub.update)
}
