package gateway

// The team directory: member and manager reach GET /hr/employees through team:employee:read and
// get only the directory fields; hr:read keeps the full profile. A real gRPC server on loopback
// plays the HR service and returns a fully populated profile, so a missing projection shows up as
// leaked fields.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/kmuhub/kmuhub/internal/auth"
	hrv1 "github.com/kmuhub/kmuhub/proto/hr/v1"
)

type employeeDirectoryStub struct {
	hrv1.UnimplementedHRServiceServer
	gotUserID string
}

func fullEmployeeProfile(userID string) *hrv1.EmployeeProfile {
	return &hrv1.EmployeeProfile{
		Id:                    "44444444-4444-4444-4444-444444444444",
		UserId:                userID,
		UserName:              "Erika Muster",
		UserEmail:             "erika@example.test",
		Department:            "Vertrieb",
		PositionTitle:         "Account Manager",
		ContractType:          hrv1.ContractType_CONTRACT_PART_TIME,
		WorkDaysPerWeek:       3,
		AnnualLeaveDays:       30,
		ManagerUserId:         leaveOtherID,
		ManagerName:           "Max Chef",
		StartDate:             "2020-01-01",
		EmergencyContactName:  "Hans Muster",
		EmergencyContactPhone: "+49 170 0000000",
		AddressStreet:         "Hauptstr. 1",
		AddressCity:           "Berlin",
		AddressPostalCode:     "10115",
		AddressCountry:        "DE",
		Status:                "active",
		ExitType:              "resignation",
		ExitReason:            "privat",
		IsMinor:               true,
	}
}

func (s *employeeDirectoryStub) ListEmployees(_ context.Context, _ *hrv1.ListEmployeesReq) (*hrv1.ListEmployeesResp, error) {
	return &hrv1.ListEmployeesResp{Employees: []*hrv1.EmployeeProfile{fullEmployeeProfile(leaveCallerID)}, Total: 1}, nil
}

func (s *employeeDirectoryStub) GetEmployee(_ context.Context, r *hrv1.GetEmployeeReq) (*hrv1.GetEmployeeResp, error) {
	s.gotUserID = r.GetUserId()
	return &hrv1.GetEmployeeResp{Employee: fullEmployeeProfile(r.GetUserId())}, nil
}

func newEmployeeDirectoryRouter(t *testing.T) (*employeeDirectoryStub, http.Handler) {
	t.Helper()
	stub := &employeeDirectoryStub{}
	router := chi.NewRouter()
	NewHRRoutes(registryFor(t, "biz", func(s *grpc.Server) { hrv1.RegisterHRServiceServer(s, stub) }), nil).
		RegisterRoutes(router, guardTestAuth)
	return stub, router
}

// Token shapes from the preset roles (migration 000256 plus hr:self from 000333).
var (
	directoryMemberToken  = tokenShape{perms: []string{"hr:self", "team:employee:read"}, scopes: map[string]string{"hr:self": auth.ScopeOwn, "team:employee:read": auth.ScopeAll}}
	directoryManagerToken = tokenShape{perms: []string{"hr:self", "team:employee:read"}, scopes: map[string]string{"hr:self": auth.ScopeOwn, "team:employee:read": auth.ScopeTeam}}
)

// Profile fields that must never reach a directory reader.
var directoryHiddenFields = []string{
	"contract_type", "work_days_per_week", "annual_leave_days", "start_date",
	"emergency_contact_name", "emergency_contact_phone",
	"address_street", "address_city", "address_postal_code", "address_country",
	"exit_type", "exit_reason", "is_minor",
}

func listEmployees(t *testing.T, router http.Handler, tk tokenShape) []map[string]any {
	t.Helper()
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, tk.request(http.MethodGet, "/api/v1/hr/employees"))
	assertStatus(t, rec, http.StatusOK)
	var body struct {
		Employees []map[string]any `json:"employees"`
		Total     int              `json:"total"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Employees, 1)
	assert.Equal(t, 1, body.Total)
	return body.Employees
}

func TestEmployeeDirectory_TeamEmployeeReadGetsOnlyTheDirectoryFields(t *testing.T) {
	_, router := newEmployeeDirectoryRouter(t)

	for name, tk := range map[string]tokenShape{"member": directoryMemberToken, "manager": directoryManagerToken} {
		t.Run(name, func(t *testing.T) {
			emp := listEmployees(t, router, tk)[0]
			for _, f := range directoryHiddenFields {
				assert.NotContains(t, emp, f, "a directory reader must not see %s", f)
			}
			// Exactly what the pickers, the team page and the org chart read.
			assert.Equal(t, map[string]any{
				"id":              "44444444-4444-4444-4444-444444444444",
				"user_id":         leaveCallerID,
				"user_name":       "Erika Muster",
				"user_email":      "erika@example.test",
				"department":      "Vertrieb",
				"position_title":  "Account Manager",
				"manager_user_id": leaveOtherID,
				"manager_name":    "Max Chef",
				"status":          "active",
			}, emp)
		})
	}
}

func TestEmployeeDirectory_HRReadKeepsTheFullProfile(t *testing.T) {
	_, router := newEmployeeDirectoryRouter(t)

	for name, tk := range map[string]tokenShape{
		"admin":        {perms: []string{"hr:read", "team:employee:read"}, scopes: map[string]string{"hr:read": auth.ScopeAll, "team:employee:read": auth.ScopeAll}},
		"legacy token": {perms: []string{"hr:read"}},
	} {
		t.Run(name, func(t *testing.T) {
			emp := listEmployees(t, router, tk)[0]
			for _, f := range []string{"annual_leave_days", "address_city", "emergency_contact_name", "start_date", "is_minor", "contract_type"} {
				assert.Contains(t, emp, f, "hr:read keeps %s", f)
			}
		})
	}
}

func TestEmployeeDirectory_SelfProfileForDirectoryReaders(t *testing.T) {
	stub, router := newEmployeeDirectoryRouter(t)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, directoryMemberToken.request(http.MethodGet, "/api/v1/hr/employees/me"))
	assertStatus(t, rec, http.StatusOK)
	assert.Equal(t, leaveCallerID, stub.gotUserID, "the own profile is looked up by the token's user id")

	var body struct {
		Employee map[string]any `json:"employee"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, leaveCallerID, body.Employee["user_id"])
	assert.Contains(t, body.Employee, "annual_leave_days", "the own profile stays complete")
}

func TestEmployeeDirectory_WithoutEitherKeyIsRefused(t *testing.T) {
	_, router := newEmployeeDirectoryRouter(t)
	noKey := tokenShape{perms: []string{"hr:self"}, scopes: map[string]string{"hr:self": auth.ScopeOwn}}

	for _, path := range []string{"/api/v1/hr/employees", "/api/v1/hr/employees/me", "/api/v1/hr/employees/" + leaveOtherID} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, noKey.request(http.MethodGet, path))
		assertStatus(t, rec, http.StatusForbidden)
	}

	// A single foreign profile stays behind hr:read.
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, directoryMemberToken.request(http.MethodGet, "/api/v1/hr/employees/"+leaveOtherID))
	assertStatus(t, rec, http.StatusForbidden)
}
