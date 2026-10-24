package gateway

// Smaller wire gaps of the branch modules (Produktion, Rapporte, Vermietung, Fuhrpark) as the
// desktop hits them: zero counters and empty lists that vanished from the JSON, form fields the
// gateway dropped, a free-text damage reporter that was rejected as "not a UUID", and query
// parameters that never reached the service. A real gRPC server on loopback plays each service.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	fuhrparkv1 "github.com/kmuhub/kmuhub/proto/fuhrpark/v1"
	produktionv1 "github.com/kmuhub/kmuhub/proto/produktion/v1"
	rapportev1 "github.com/kmuhub/kmuhub/proto/rapporte/v1"
	vermietungv1 "github.com/kmuhub/kmuhub/proto/vermietung/v1"
)

const branchenUserID = "22222222-2222-2222-2222-222222222222"

func branchenRequest(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withAuth(req, branchenUserID, testTenantID)
	req = withChiURLParam(req, "orderId", "11111111-1111-1111-1111-111111111111")
	return withChiURLParam(req, "id", "11111111-1111-1111-1111-111111111111")
}

// --- Produktion -------------------------------------------------------------

type produktionShapeStub struct {
	produktionv1.UnimplementedProduktionServiceServer
	listBOMs *produktionv1.ListBOMsRequest
}

func (s *produktionShapeStub) ListBOMs(_ context.Context, r *produktionv1.ListBOMsRequest) (*produktionv1.ListBOMsResponse, error) {
	s.listBOMs = r
	// A BOM without positions and a position with sort_order 0: both are zero values.
	return &produktionv1.ListBOMsResponse{Boms: []*produktionv1.BOM{{Id: "b1", ProductName: "Tisch"}}, Total: 1}, nil
}

func (s *produktionShapeStub) ListWorkSteps(context.Context, *produktionv1.ListWorkStepsRequest) (*produktionv1.ListWorkStepsResponse, error) {
	return &produktionv1.ListWorkStepsResponse{Steps: []*produktionv1.WorkStep{{Id: "s1", Name: "Sägen"}}, Total: 1}, nil
}

func (s *produktionShapeStub) ListQualityChecks(context.Context, *produktionv1.ListQualityChecksRequest) (*produktionv1.ListQualityChecksResponse, error) {
	return &produktionv1.ListQualityChecksResponse{Checks: []*produktionv1.QualityCheck{{Id: "q1", Passed: true}}, Total: 1}, nil
}

func newProduktionShapeRoutes(t *testing.T) (*ProduktionRoutes, *produktionShapeStub) {
	t.Helper()
	stub := &produktionShapeStub{}
	return NewProduktionRoutes(registryFor(t, "produktion", func(s *grpc.Server) { produktionv1.RegisterProduktionServiceServer(s, stub) }), nil), stub
}

func TestBranchenShape_ProduktionZeroValuesStayInTheJSON(t *testing.T) {
	routes, _ := newProduktionShapeRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleListBOMs(rec, branchenRequest(http.MethodGet, "/boms", ""))
	assertStatus(t, rec, http.StatusOK)
	bom := decodeBody(t, rec)["boms"].([]any)[0].(map[string]any)
	assert.Equal(t, []any{}, bom["items"], "a BOM without positions lists an empty array, bom.items.map must not throw: "+rec.Body.String())

	rec = httptest.NewRecorder()
	routes.HandleListWorkSteps(rec, branchenRequest(http.MethodGet, "/orders/x/steps", ""))
	assertStatus(t, rec, http.StatusOK)
	step := decodeBody(t, rec)["steps"].([]any)[0].(map[string]any)
	assert.Equal(t, float64(0), step["duration_minutes"], "a step without duration shows 0, not NaN: "+rec.Body.String())

	rec = httptest.NewRecorder()
	routes.HandleListQualityChecks(rec, branchenRequest(http.MethodGet, "/quality", ""))
	assertStatus(t, rec, http.StatusOK)
	check := decodeBody(t, rec)["checks"].([]any)[0].(map[string]any)
	assert.Equal(t, float64(0), check["defects_found"], rec.Body.String())
}

func TestBranchenShape_ProduktionBOMListForwardsActiveFilter(t *testing.T) {
	routes, stub := newProduktionShapeRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleListBOMs(rec, branchenRequest(http.MethodGet, "/boms?active=false", ""))
	assertStatus(t, rec, http.StatusOK)
	require.NotNil(t, stub.listBOMs.Active)
	assert.False(t, *stub.listBOMs.Active)

	rec = httptest.NewRecorder()
	routes.HandleListBOMs(rec, branchenRequest(http.MethodGet, "/boms", ""))
	assertStatus(t, rec, http.StatusOK)
	assert.Nil(t, stub.listBOMs.Active, "no filter given means no filter sent")
}

// --- Rapporte ---------------------------------------------------------------

type rapporteShapeStub struct {
	rapportev1.UnimplementedRapporteServiceServer
	create *rapportev1.CreateReportRequest
	update *rapportev1.UpdateReportRequest
}

func (s *rapporteShapeStub) CreateReport(_ context.Context, r *rapportev1.CreateReportRequest) (*rapportev1.ReportResponse, error) {
	s.create = r
	return &rapportev1.ReportResponse{Report: &rapportev1.WorkReport{Id: "r1", Weather: r.GetWeather(), WorkStart: r.GetWorkStart(), BreakMinutes: r.GetBreakMinutes()}}, nil
}

func (s *rapporteShapeStub) UpdateReport(_ context.Context, r *rapportev1.UpdateReportRequest) (*rapportev1.ReportResponse, error) {
	s.update = r
	return &rapportev1.ReportResponse{Report: &rapportev1.WorkReport{Id: "r1"}}, nil
}

func (s *rapporteShapeStub) GetReportStats(context.Context, *rapportev1.GetReportStatsRequest) (*rapportev1.ReportStatsResponse, error) {
	return &rapportev1.ReportStatsResponse{}, nil // a new tenant: every counter is 0
}

func newRapporteShapeRoutes(t *testing.T) (*RapporteRoutes, *rapporteShapeStub) {
	t.Helper()
	stub := &rapporteShapeStub{}
	return NewRapporteRoutes(registryFor(t, "rapporte", func(s *grpc.Server) { rapportev1.RegisterRapporteServiceServer(s, stub) }), nil), stub
}

func TestBranchenShape_RapporteStatsKeepZeroCounters(t *testing.T) {
	routes, _ := newRapporteShapeRoutes(t)
	rec := httptest.NewRecorder()
	routes.HandleGetReportStats(rec, branchenRequest(http.MethodGet, "/stats", ""))
	assertStatus(t, rec, http.StatusOK)
	body := decodeBody(t, rec)
	for _, k := range []string{"total_reports", "draft_count", "submitted_count", "approved_count", "rejected_count"} {
		assert.Equal(t, float64(0), body[k], "%s must be present as 0 (approved+submitted would be NaN): %s", k, rec.Body.String())
	}
}

func TestBranchenShape_RapporteCreateForwardsSiteDiaryFields(t *testing.T) {
	routes, stub := newRapporteShapeRoutes(t)
	rec := httptest.NewRecorder()
	routes.HandleCreateReport(rec, branchenRequest(http.MethodPost, "/reports",
		`{"title":"Baustelle","author_id":"`+branchenUserID+`","weather":"rainy","temperature":0,"work_start":"07:30",`+
			`"work_end":"16:00","break_minutes":45,"project_name":"Neubau"}`))

	assertStatus(t, rec, http.StatusCreated)
	require.NotNil(t, stub.create)
	assert.Equal(t, "rainy", stub.create.GetWeather())
	require.NotNil(t, stub.create.Temperature, "0 degrees is a reading, not an omitted field")
	assert.Equal(t, 0.0, *stub.create.Temperature)
	assert.Equal(t, "07:30", stub.create.GetWorkStart())
	assert.Equal(t, "16:00", stub.create.GetWorkEnd())
	require.NotNil(t, stub.create.BreakMinutes)
	assert.Equal(t, int32(45), *stub.create.BreakMinutes)
	assert.Equal(t, "Neubau", stub.create.GetProjectName())
}

func TestBranchenShape_RapporteUpdateForwardsSiteDiaryFieldsAndLeavesOmittedUnset(t *testing.T) {
	routes, stub := newRapporteShapeRoutes(t)
	rec := httptest.NewRecorder()
	routes.HandleUpdateReport(rec, branchenRequest(http.MethodPut, "/reports/x", `{"weather":"sunny","break_minutes":30}`))

	assertStatus(t, rec, http.StatusOK)
	require.NotNil(t, stub.update)
	require.NotNil(t, stub.update.Weather)
	assert.Equal(t, "sunny", *stub.update.Weather)
	require.NotNil(t, stub.update.BreakMinutes)
	assert.Equal(t, int32(30), *stub.update.BreakMinutes)
	assert.Nil(t, stub.update.WorkStart, "an omitted key stays unset so the stored value is kept")
	assert.Nil(t, stub.update.Temperature)
}

// --- Vermietung -------------------------------------------------------------

type vermietungShapeStub struct {
	vermietungv1.UnimplementedVermietungServiceServer
}

func (s *vermietungShapeStub) ListRentals(context.Context, *vermietungv1.ListRentalsRequest) (*vermietungv1.ListRentalsResponse, error) {
	return &vermietungv1.ListRentalsResponse{Rentals: []*vermietungv1.Rental{{Id: "m1", RenterName: "Max"}}, Total: 1}, nil
}

func TestBranchenShape_VermietungRentalKeepsZeroTotalPrice(t *testing.T) {
	stub := &vermietungShapeStub{}
	routes := NewVermietungRoutes(registryFor(t, "vermietung", func(s *grpc.Server) { vermietungv1.RegisterVermietungServiceServer(s, stub) }), nil)
	rec := httptest.NewRecorder()
	routes.HandleListRentals(rec, branchenRequest(http.MethodGet, "/rentals", ""))
	assertStatus(t, rec, http.StatusOK)
	rental := decodeBody(t, rec)["rentals"].([]any)[0].(map[string]any)
	assert.Equal(t, float64(0), rental["total_price"], "a free rental shows 0 EUR, not undefined: "+rec.Body.String())
}

// --- Fuhrpark ---------------------------------------------------------------

type fuhrparkShapeStub struct {
	fuhrparkv1.UnimplementedFuhrparkServiceServer
	damage   *fuhrparkv1.ReportDamageRequest
	trip     *fuhrparkv1.CreateTripLogRequest
	tuev     *fuhrparkv1.CheckTuevDueRequest
	upcoming *fuhrparkv1.ListUpcomingServicesRequest
	gps      *fuhrparkv1.GetGpsPositionsRequest
}

func (s *fuhrparkShapeStub) ReportDamage(_ context.Context, r *fuhrparkv1.ReportDamageRequest) (*fuhrparkv1.DamageResponse, error) {
	s.damage = r
	return &fuhrparkv1.DamageResponse{Damage: &fuhrparkv1.Damage{Id: "d1"}}, nil
}

func (s *fuhrparkShapeStub) CreateTripLog(_ context.Context, r *fuhrparkv1.CreateTripLogRequest) (*fuhrparkv1.CreateTripLogResponse, error) {
	s.trip = r
	return &fuhrparkv1.CreateTripLogResponse{TripLog: &fuhrparkv1.TripLog{Id: "t1"}}, nil
}

func (s *fuhrparkShapeStub) CheckTuevDue(_ context.Context, r *fuhrparkv1.CheckTuevDueRequest) (*fuhrparkv1.CheckTuevDueResponse, error) {
	s.tuev = r
	return &fuhrparkv1.CheckTuevDueResponse{}, nil
}

func (s *fuhrparkShapeStub) ListUpcomingServices(_ context.Context, r *fuhrparkv1.ListUpcomingServicesRequest) (*fuhrparkv1.ListServicesResponse, error) {
	s.upcoming = r
	return &fuhrparkv1.ListServicesResponse{}, nil
}

func (s *fuhrparkShapeStub) GetGpsPositions(_ context.Context, r *fuhrparkv1.GetGpsPositionsRequest) (*fuhrparkv1.GetGpsPositionsResponse, error) {
	s.gps = r
	return &fuhrparkv1.GetGpsPositionsResponse{}, nil
}

func newFuhrparkShapeRoutes(t *testing.T) (*FuhrparkRoutes, *fuhrparkShapeStub) {
	t.Helper()
	stub := &fuhrparkShapeStub{}
	return NewFuhrparkRoutes(registryFor(t, "fuhrpark", func(s *grpc.Server) { fuhrparkv1.RegisterFuhrparkServiceServer(s, stub) }), nil), stub
}

func TestBranchenShape_FuhrparkDamageReporterNameMovesIntoNotes(t *testing.T) {
	routes, stub := newFuhrparkShapeRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleReportDamage(rec, branchenRequest(http.MethodPost, "/vehicles/x/damages",
		`{"description":"Kratzer","reported_by":"Max Mustermann","notes":"Tür links"}`))
	assertStatus(t, rec, http.StatusCreated)
	require.NotNil(t, stub.damage.ReportedBy)
	assert.Equal(t, branchenUserID, *stub.damage.ReportedBy, "a name is not an ID: the caller is the reporter")
	require.NotNil(t, stub.damage.Notes)
	assert.Equal(t, "Tür links\nGemeldet von: Max Mustermann", *stub.damage.Notes)

	rec = httptest.NewRecorder()
	routes.HandleReportDamage(rec, branchenRequest(http.MethodPost, "/vehicles/x/damages",
		`{"description":"Delle","reported_by":"`+branchenUserID+`"}`))
	assertStatus(t, rec, http.StatusCreated)
	assert.Equal(t, branchenUserID, stub.damage.GetReportedBy(), "a UUID passes through untouched")
	assert.Nil(t, stub.damage.Notes)
}

func TestBranchenShape_FuhrparkTripLogPrivateTripNeedsNeitherPurposeNorDriver(t *testing.T) {
	routes, stub := newFuhrparkShapeRoutes(t)
	base := `"date":"2026-10-20","start_location":"A","end_location":"B","start_km":10,"end_km":20`

	rec := httptest.NewRecorder()
	routes.HandleCreateTripLog(rec, branchenRequest(http.MethodPost, "/vehicles/x/trip-logs", `{`+base+`,"is_private":true}`))
	assertStatus(t, rec, http.StatusCreated)
	require.NotNil(t, stub.trip)
	assert.True(t, stub.trip.IsPrivate)

	stub.trip = nil
	rec = httptest.NewRecorder()
	routes.HandleCreateTripLog(rec, branchenRequest(http.MethodPost, "/vehicles/x/trip-logs", `{`+base+`,"driver_name":""}`))
	assertStatus(t, rec, http.StatusBadRequest)
	assert.Nil(t, stub.trip, "a business trip without a purpose never reaches the service")

	rec = httptest.NewRecorder()
	routes.HandleCreateTripLog(rec, branchenRequest(http.MethodPost, "/vehicles/x/trip-logs", `{`+base+`,"purpose":"Kundentermin"}`))
	assertStatus(t, rec, http.StatusCreated)
	assert.Equal(t, "Kundentermin", stub.trip.GetPurpose(), "driver_name stays optional")
}

func TestBranchenShape_FuhrparkQueryParametersReachTheService(t *testing.T) {
	routes, stub := newFuhrparkShapeRoutes(t)

	rec := httptest.NewRecorder()
	routes.HandleListUpcomingServices(rec, branchenRequest(http.MethodGet, "/services/upcoming?days_ahead=90", ""))
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, int32(90), stub.upcoming.GetDaysAhead())

	rec = httptest.NewRecorder()
	routes.HandleCheckTuevDue(rec, branchenRequest(http.MethodGet, "/tuev-due?days_ahead=60", ""))
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, int32(60), stub.tuev.GetDaysAhead())

	rec = httptest.NewRecorder()
	routes.HandleCheckTuevDue(rec, branchenRequest(http.MethodGet, "/tuev-due?days_ahead=-5", ""))
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, int32(0), stub.tuev.GetDaysAhead(), "an invalid value falls back to the service default")

	rec = httptest.NewRecorder()
	routes.HandleGetGpsPositions(rec, branchenRequest(http.MethodGet, "/gps/positions?vehicle_id=v1&limit=250", ""))
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, int32(250), stub.gps.GetLimit())
}
