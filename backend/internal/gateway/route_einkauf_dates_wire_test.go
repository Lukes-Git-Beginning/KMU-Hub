package gateway

// Dates and empty lists of the Einkauf purchase orders and framework contracts as the desktop
// sends and reads them: the order dates and contract term reach the einkauf service, `null`
// empties a date, and a contract date reads `YYYY-MM-DD`. A real gRPC server on loopback plays
// the einkauf service.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	einkaufv1 "github.com/kmuhub/kmuhub/proto/einkauf/v1"
)

type einkaufDatesStub struct {
	einkaufv1.UnimplementedEinkaufServiceServer
	createPO       *einkaufv1.CreatePORequest
	updatePO       *einkaufv1.UpdatePORequest
	createContract *einkaufv1.CreateFrameworkContractRequest
	updateContract *einkaufv1.UpdateFrameworkContractRequest
}

func (s *einkaufDatesStub) CreatePO(_ context.Context, r *einkaufv1.CreatePORequest) (*einkaufv1.POResponse, error) {
	s.createPO = r
	return &einkaufv1.POResponse{Po: &einkaufv1.PurchaseOrder{Id: "po"}}, nil
}

func (s *einkaufDatesStub) UpdatePO(_ context.Context, r *einkaufv1.UpdatePORequest) (*einkaufv1.POResponse, error) {
	s.updatePO = r
	return &einkaufv1.POResponse{Po: &einkaufv1.PurchaseOrder{Id: "po"}}, nil
}

func (s *einkaufDatesStub) ListPOLines(context.Context, *einkaufv1.ListPOLinesRequest) (*einkaufv1.ListPOLinesResponse, error) {
	return &einkaufv1.ListPOLinesResponse{}, nil
}

func contractWithTerm() *einkaufv1.FrameworkContract {
	return &einkaufv1.FrameworkContract{
		Id:        "fc",
		StartDate: timestamppb.New(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)),
		EndDate:   timestamppb.New(time.Date(2027, 2, 28, 0, 0, 0, 0, time.UTC)),
	}
}

func (s *einkaufDatesStub) CreateFrameworkContract(_ context.Context, r *einkaufv1.CreateFrameworkContractRequest) (*einkaufv1.FrameworkContractResponse, error) {
	s.createContract = r
	return &einkaufv1.FrameworkContractResponse{Contract: contractWithTerm()}, nil
}

func (s *einkaufDatesStub) UpdateFrameworkContract(_ context.Context, r *einkaufv1.UpdateFrameworkContractRequest) (*einkaufv1.FrameworkContractResponse, error) {
	s.updateContract = r
	return &einkaufv1.FrameworkContractResponse{Contract: &einkaufv1.FrameworkContract{Id: "fc"}}, nil
}

// ListFrameworkContracts returns one contract once a create call went through, none before.
func (s *einkaufDatesStub) ListFrameworkContracts(context.Context, *einkaufv1.ListFrameworkContractsRequest) (*einkaufv1.ListFrameworkContractsResponse, error) {
	if s.createContract == nil {
		return &einkaufv1.ListFrameworkContractsResponse{}, nil
	}
	return &einkaufv1.ListFrameworkContractsResponse{Contracts: []*einkaufv1.FrameworkContract{contractWithTerm()}, Total: 1}, nil
}

func newEinkaufDatesRoutes(t *testing.T) (*EinkaufRoutes, *einkaufDatesStub) {
	t.Helper()
	stub := &einkaufDatesStub{}
	return newEinkaufRoutes(registryFor(t, "einkauf", func(s *grpc.Server) { einkaufv1.RegisterEinkaufServiceServer(s, stub) })), stub
}

const supplierJSON = `"supplier_id":"11111111-1111-1111-1111-111111111111"`

func TestEinkaufDates_CreatePOForwardsBothDates(t *testing.T) {
	routes, stub := newEinkaufDatesRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleCreatePO(rec, taxModeRequest(http.MethodPost,
		`{`+supplierJSON+`,"po_number":"PO-1","order_date":"2026-05-04T10:30:00.000Z","expected_delivery_date":"2026-05-20"}`))

	assertStatus(t, rec, http.StatusCreated)
	require.NotNil(t, stub.createPO)
	assert.Equal(t, time.Date(2026, 5, 4, 10, 30, 0, 0, time.UTC), stub.createPO.GetOrderDate().AsTime())
	assert.Equal(t, time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC), stub.createPO.GetExpectedDeliveryDate().AsTime())
}

func TestEinkaufDates_CreatePORejectsMalformedDate(t *testing.T) {
	routes, stub := newEinkaufDatesRoutes(t)

	for field, value := range map[string]string{"order_date": "gestern", "expected_delivery_date": "20.05.2026"} {
		rec := httptest.NewRecorder()
		routes.HandleCreatePO(rec, taxModeRequest(http.MethodPost,
			`{`+supplierJSON+`,"po_number":"PO-1","`+field+`":"`+value+`"}`))
		assertStatus(t, rec, http.StatusBadRequest)
		assertErrorContains(t, rec, field)
	}
	assert.Nil(t, stub.createPO, "a malformed date must not reach the service")
}

// epochOf reads the clear marker: a set Timestamp at the epoch (a nil one would also read as 0).
func epochOf(t *testing.T, ts *timestamppb.Timestamp) int64 {
	t.Helper()
	require.NotNil(t, ts, "the clear marker must be sent, not omitted")
	return ts.AsTime().Unix()
}

func TestEinkaufDates_UpdatePOKeepsClearsAndSetsDeliveryDate(t *testing.T) {
	routes, stub := newEinkaufDatesRoutes(t)
	update := func(body string) *einkaufv1.UpdatePORequest {
		stub.updatePO = nil
		rec := httptest.NewRecorder()
		routes.HandleUpdatePO(rec, taxModeRequest(http.MethodPatch, body))
		assertStatus(t, rec, http.StatusOK)
		require.NotNil(t, stub.updatePO)
		return stub.updatePO
	}

	assert.Nil(t, update(`{"notes":"x"}`).ExpectedDeliveryDate, "an omitted key leaves the date alone")
	assert.EqualValues(t, 0, epochOf(t, update(`{"expected_delivery_date":null}`).ExpectedDeliveryDate),
		"null sends the epoch the service reads as clear")
	assert.EqualValues(t, 0, epochOf(t, update(`{"expected_delivery_date":""}`).ExpectedDeliveryDate))
	assert.Equal(t, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		update(`{"expected_delivery_date":"2026-06-01","order_date":"2026-05-01"}`).GetExpectedDeliveryDate().AsTime())

	rec := httptest.NewRecorder()
	routes.HandleUpdatePO(rec, taxModeRequest(http.MethodPatch, `{"order_date":"bald"}`))
	assertStatus(t, rec, http.StatusBadRequest)
}

func TestEinkaufDates_POWithoutLinesListsEmptyArray(t *testing.T) {
	routes, _ := newEinkaufDatesRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleListPOLines(rec, taxModeRequest(http.MethodGet, ""))

	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, []any{}, decodeBody(t, rec)["lines"], rec.Body.String())
}

func TestEinkaufDates_ContractTermReachesServiceAndReadsAsDate(t *testing.T) {
	routes, stub := newEinkaufDatesRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleCreateFrameworkContract(rec, taxModeRequest(http.MethodPost,
		`{`+supplierJSON+`,"title":"Rahmen","contract_nr":"R-1","start_date":"2026-03-01","end_date":"2027-02-28T00:00:00Z"}`))

	assertStatus(t, rec, http.StatusCreated)
	require.NotNil(t, stub.createContract)
	assert.Equal(t, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), stub.createContract.GetStartDate().AsTime())
	assert.Equal(t, time.Date(2027, 2, 28, 0, 0, 0, 0, time.UTC), stub.createContract.GetEndDate().AsTime())
	contract := decodeBody(t, rec)["contract"].(map[string]any)
	assert.Equal(t, "2026-03-01", contract["start_date"], rec.Body.String())
	assert.Equal(t, "2027-02-28", contract["end_date"], rec.Body.String())

	rec = httptest.NewRecorder()
	routes.HandleListFrameworkContracts(rec, taxModeRequest(http.MethodGet, ""))
	assertStatus(t, rec, http.StatusOK)
	row := decodeBody(t, rec)["contracts"].([]any)[0].(map[string]any)
	assert.Equal(t, "2026-03-01", row["start_date"], rec.Body.String())
}

func TestEinkaufDates_ContractListWithoutRowsIsEmptyArray(t *testing.T) {
	routes, _ := newEinkaufDatesRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleListFrameworkContracts(rec, taxModeRequest(http.MethodGet, ""))

	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, []any{}, decodeBody(t, rec)["contracts"], rec.Body.String())
}

func TestEinkaufDates_UpdateContractKeepsClearsAndRejects(t *testing.T) {
	routes, stub := newEinkaufDatesRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleUpdateFrameworkContract(rec, taxModeRequest(http.MethodPatch, `{"title":"x"}`))
	assertStatus(t, rec, http.StatusOK)
	assert.Nil(t, stub.updateContract.StartDate)
	assert.Nil(t, stub.updateContract.EndDate)

	rec = httptest.NewRecorder()
	routes.HandleUpdateFrameworkContract(rec, taxModeRequest(http.MethodPatch, `{"start_date":null,"end_date":"2027-01-31"}`))
	assertStatus(t, rec, http.StatusOK)
	assert.EqualValues(t, 0, epochOf(t, stub.updateContract.StartDate))
	assert.Equal(t, time.Date(2027, 1, 31, 0, 0, 0, 0, time.UTC), stub.updateContract.GetEndDate().AsTime())

	rec = httptest.NewRecorder()
	routes.HandleUpdateFrameworkContract(rec, taxModeRequest(http.MethodPatch, `{"end_date":"nie"}`))
	assertStatus(t, rec, http.StatusBadRequest)
}
